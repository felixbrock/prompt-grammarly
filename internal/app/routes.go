package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/felixbrock/lemonai/internal/domain"
	"github.com/google/uuid"
)

type MessageProto struct {
	Role    string
	Content []byte
}

type OAIThread struct {
	Id string `json:"id"`
}

type assistant struct {
	Id   string
	Name string
}

type optimizationReq struct {
	OriginalPrompt string `json:"prompt"`
	Instructions   string `json:"instructions"`
}

type oaiSuggestion struct {
	Suggestion string `json:"new"`
	Reasoning  string `json:"reasoning"`
	Target     string `json:"original"`
}

type OAIRun struct {
	Id     string `json:"id"`
	Status string `json:"status"`
}

type messageContentText struct {
	Value string `json:"value"`
}

type messageContent struct {
	Text messageContentText `json:"text"`
}

type OAIMessage struct {
	Content []messageContent `json:"content"`
	Role    string           `json:"role"`
}

type OAIMessageListing struct {
	Data []OAIMessage `json:"data"`
}

type AnalysisState struct {
	CustomCompleted             bool
	ContextualRichnessCompleted bool
	ConcisenessCompleted        bool
	ClarityCompleted            bool
	ConsistencyCompleted        bool
}

func (s AnalysisState) Completed() bool {
	return s.CustomCompleted && s.ContextualRichnessCompleted && s.ConcisenessCompleted && s.ClarityCompleted && s.ConsistencyCompleted
}

func (c OptimizationController) runAssistant(threadId string, userPrompt string, assistant assistant) ([]byte, error) {
	err := c.writeUserPrompt(threadId, userPrompt)

	if err != nil {
		return nil, err
	}

	entity, err := c.Repo.OAIRepo.PostRun(assistant.Id, threadId)

	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	slog.Info(fmt.Sprintf("Running %s analysis...\n", assistant.Name))
	completed := false
	for !completed {
		select {
		case <-ctx.Done():
			slog.Warn(fmt.Sprintf("Assistant %s run timed out. Ignoring run...", assistant.Name))
			return make([]byte, 0), nil
		default:
			entity, err = c.Repo.OAIRepo.GetRun(threadId, entity.Id)

			if err != nil {
				return nil, err
			} else if entity.Status == "completed" {
				completed = true
			}

			time.Sleep(time.Second)
		}
	}

	var msgs *[]OAIMessage
	msgs, err = c.Repo.OAIRepo.GetMsgs(threadId)

	if err != nil {
		return nil, err
	}

	if len(*msgs) == 0 {
		return nil, errors.New("unexpected thread messages state error")
	}
	msg := (*msgs)[0]
	if msg.Role != "assistant" || len(msg.Content) != 1 {
		return nil, errors.New("unexpected assistant response error")
	}

	bMsg := []byte(msg.Content[0].Text.Value)
	return bMsg, nil
}

func (c OptimizationController) writeUserPrompt(threadId string, prompt string) error {
	msgContent, err := json.Marshal(prompt)

	if err != nil {
		return err
	}

	err = c.Repo.OAIRepo.PostMsg(MessageProto{Role: "user", Content: msgContent}, threadId)

	if err != nil {
		return err
	}

	return nil
}

func (c OptimizationController) genCustomAssistantUserPrompt(customInstructions string, prompt string) string {
	return fmt.Sprintf(
		`Consider the following Custom Goal:

		%s

		Evaluate the the following model instruction against the custom goal:

		%s

		`, customInstructions, prompt)
}

func (c OptimizationController) genAssistantUserPrompt(assistantName string, prompt string) string {
	return fmt.Sprintf(
		`Evaluate the %s of the following model instruction:

		%s

		`, strings.Join(strings.Split(assistantName, "_"), " "), prompt)
}

func (c OptimizationController) genOperatorUserPrompt(originalPrompt string, msg []byte) string {

	return fmt.Sprintf(
		`Apply the following list of suggestions to improve the following model instructions.
		Make sure to apply all suggestions and to weight suggestions of type 'custom' higher than the other suggestions .

		Model Instructions:

		%s


		Suggestions:

		%s

		`, originalPrompt, msg)
}

func (c OptimizationController) apply(suggestions []oaiSuggestion) ([]byte, error) {
	operator := assistant{Id: "asst_qUn97Ck3zzdvNToMVAMhNzTk", Name: "operator"}

	bSuggs, err := json.Marshal(suggestions)

	if err != nil {
		return nil, err
	}

	userPrompt := c.genOperatorUserPrompt(operator.Name, bSuggs)

	thId, err := c.Repo.OAIRepo.PostThread()

	if err != nil {
		return nil, err
	}

	defer func() {
		err = c.Repo.OAIRepo.DeleteThread(thId)
		if err != nil {
			slog.Error(fmt.Sprintf("Error occured: %s", err.Error()))
		}
	}()

	msg, err := c.runAssistant(thId, userPrompt, operator)

	if err != nil {
		return nil, err
	}

	return msg, nil
}

type optimizationBase struct {
	Prompt       string
	Instructions string
}

func (c OptimizationController) suggest(optimizationId string, threadId string, base optimizationBase, targetAssistant assistant) ([]oaiSuggestion, error) {
	runId := uuid.New().String()
	run := domain.Run{
		Id:             runId,
		Type:           targetAssistant.Name,
		State:          "running",
		OptimizationId: optimizationId}

	err := c.Repo.RunRepo.Insert(run)

	if err != nil {
		return nil, err
	}

	defer func() {
		if err == nil {
			err = c.Repo.RunRepo.Update(runId, "completed")
			if err != nil {
				slog.Error(fmt.Sprintf("Error occured: %s", err.Error()))
			}
		} else {
			err = c.Repo.RunRepo.Update(runId, "failed")
			if err != nil {
				slog.Error(fmt.Sprintf("Error occured: %s", err.Error()))
			}
		}

	}()

	var userPrompt string
	if targetAssistant.Name == "custom" {
		if base.Instructions == "" {
			return []oaiSuggestion{}, nil
		}

		userPrompt = c.genCustomAssistantUserPrompt(base.Instructions, base.Prompt)
	} else {

		userPrompt = c.genAssistantUserPrompt(targetAssistant.Name, base.Prompt)
	}

	msg, err := c.runAssistant(threadId, userPrompt, targetAssistant)

	if err != nil {
		return nil, err
	} else if len(msg) == 0 {
		// handling timed out assistant runs
		return make([]oaiSuggestion, 0), nil
	}

	var suggestions *[]oaiSuggestion
	suggestions, err = ReadJSON[[]oaiSuggestion](msg)

	if err != nil {
		slog.Warn(fmt.Sprintf("Assistant %s produced unparseable JSON suggestions. Ignoring suggestions...", targetAssistant.Name))
		// to accommodate defer statement
		err = nil
		return make([]oaiSuggestion, 0), nil
	}

	suggestionRecords := make([]domain.Suggestion, len(*suggestions))
	for i := 0; i < len(*suggestions); i++ {
		suggestionRecords[i] = domain.Suggestion{
			Id:             uuid.New().String(),
			Suggestion:     (*suggestions)[i].Suggestion,
			Reasoning:      (*suggestions)[i].Reasoning,
			UserFeedback:   0,
			Target:         (*suggestions)[i].Target,
			Type:           targetAssistant.Name,
			RunId:          runId,
			OptimizationId: optimizationId}
	}

	err = c.Repo.SuggRepo.Insert(suggestionRecords)

	if err != nil {
		return nil, err
	}

	slog.Info(fmt.Sprintf("Successfully generated %s suggestions", targetAssistant.Name))

	return *suggestions, nil
}

func (c OptimizationController) optimize(opId string, parentId string, base optimizationBase) {
	assistants := []assistant{{Id: "asst_BxUQqxSD8tcvQoyR6T5iom3L", Name: "contextual_richness"},
		{Id: "asst_3q6LvmiPZyoPChdrcuqMxOvh", Name: "conciseness"},
		{Id: "asst_8IjCbTm7tsgCtSbhEL7E7rjB", Name: "clarity"},
		{Id: "asst_221Q0E9EeazCHcGV4Qd050Gy", Name: "consistency"},
		{Id: "asst_9zcQxyRh4E10Agg08p8mYDO8", Name: "custom"}}

	var wg sync.WaitGroup
	outputCh := make(chan []oaiSuggestion)

	for i := 0; i < len(assistants); i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			thId, err := c.Repo.OAIRepo.PostThread()

			if err != nil {
				slog.Error(fmt.Sprintf("Error occured: %s", err.Error()))
				return
			}

			defer func() {
				err = c.Repo.OAIRepo.DeleteThread(thId)
				if err != nil {
					slog.Error(fmt.Sprintf("Error occured: %s", err.Error()))
				}
			}()

			suggestions, err := c.suggest(opId, thId, base, assistants[id])

			if err != nil {
				slog.Error(fmt.Sprintf("Error occured: %s", err.Error()))
				return
			}

			outputCh <- suggestions
		}(i)
	}

	go func() {
		wg.Wait()
		close(outputCh)
	}()

	var suggestions []oaiSuggestion
	for {
		output, ok := <-outputCh
		if !ok {
			break
		}
		suggestions = append(suggestions, output...)
	}

	var msg []byte
	msg, err := c.apply(suggestions)

	if err != nil {
		slog.Error(fmt.Sprintf("Error occured: %s", err.Error()))
		return
	}

	var opts OpUpdateOpts
	opts.State = "completed"
	opts.OptimizedPrompt = string(msg)
	if parentId != "" {
		opts.ParentId = parentId
	}

	c.Repo.OpRepo.Update(opId, opts)
}

func (c OptimizationController) run(opId string, parentId string, body []byte) {
	opReqBody, err := ReadJSON[optimizationReq](body)

	if err != nil {
		slog.Error(fmt.Sprintf("Error occured: %s", err.Error()))
		return
	}

	if parentId != "" {
		c.Repo.PHRepo.Capture(fmt.Sprintf("%s_user_regenerated", c.Config.Env), opId)
	} else {
		c.Repo.PHRepo.Capture(fmt.Sprintf("%s_user_generated", c.Config.Env), opId)
	}

	optimization := domain.Optimization{
		Id:              opId,
		OriginalPrompt:  opReqBody.OriginalPrompt,
		Instructions:    opReqBody.Instructions,
		ParentId:        parentId,
		OptimizedPrompt: "",
		State:           "pending"}

	err = c.Repo.OpRepo.Insert(optimization)

	if err != nil {
		slog.Error(fmt.Sprintf("Error occured: %s", err.Error()))
		return
	}

	c.optimize(opId, parentId, optimizationBase{Prompt: opReqBody.OriginalPrompt,
		Instructions: opReqBody.Instructions})
}

func (c OptimizationController) readAnalysisState(optimizationId string) (*AnalysisState, error) {
	records, err := c.Repo.RunRepo.Read(RunReadFilter{OptimizationId: optimizationId})

	if err != nil {
		return nil, err
	}

	var state AnalysisState
	for i := 0; i < len(*records); i++ {
		record := (*records)[i]
		runCompleted := record.State == "completed"
		switch record.Type {
		case "contextual_richness":
			state.ContextualRichnessCompleted = runCompleted
		case "conciseness":
			state.ConcisenessCompleted = runCompleted
		case "clarity":
			state.ClarityCompleted = runCompleted
		case "consistency":
			state.ConsistencyCompleted = runCompleted
		case "custom":
			state.CustomCompleted = runCompleted
		default:
			return nil, errors.New("unexpected run type error")
		}
	}

	return &state, nil
}

type IndexController struct {
	ComponentBuilder *ComponentBuilder
}

func (c IndexController) Handle(w http.ResponseWriter, r *http.Request) *AppResp {
	switch r.Method {
	case "GET":
		return &AppResp{Component: c.ComponentBuilder.Index(), Code: 200, Message: "OK", ContentType: "text/html", Error: nil}
	default:
		errConfig := get405()
		err := errors.New("method not allowed")
		return &AppResp{Component: c.ComponentBuilder.Error(strconv.Itoa(errConfig.Code), errConfig.Title, errConfig.Msg),
			Code: errConfig.Code, Message: errConfig.Msg, ContentType: "text/html", Error: err}
	}
}

type AppController struct {
	ComponentBuilder *ComponentBuilder
}

func (c AppController) Handle(w http.ResponseWriter, r *http.Request) *AppResp {
	switch r.Method {
	case "GET":
		return &AppResp{Component: c.ComponentBuilder.App(), Code: 200, Message: "OK", ContentType: "text/html", Error: nil}
	default:
		errConfig := get405()
		err := errors.New("method not allowed")
		return &AppResp{Component: c.ComponentBuilder.Error(strconv.Itoa(errConfig.Code), errConfig.Title, errConfig.Msg),
			Code: errConfig.Code, Message: errConfig.Msg, ContentType: "text/html", Error: err}
	}
}

type DraftModeEditorController struct {
	ComponentBuilder *ComponentBuilder
}

func (c DraftModeEditorController) Handle(w http.ResponseWriter, r *http.Request) *AppResp {
	switch r.Method {
	case "GET":
		return &AppResp{Component: c.ComponentBuilder.Draft(), Code: 200, Message: "OK", ContentType: "text/html", Error: nil}
	default:
		errConfig := get405()
		err := errors.New("method not allowed")
		return &AppResp{Component: c.ComponentBuilder.Error(strconv.Itoa(errConfig.Code), errConfig.Title, errConfig.Msg),
			Code: errConfig.Code, Message: errConfig.Msg, ContentType: "text/html", Error: err}

	}
}

type OptimizationController struct {
	ComponentBuilder *ComponentBuilder
	Repo             *Repo
	Config           *Config
}

func (c OptimizationController) Handle(w http.ResponseWriter, r *http.Request) *AppResp {
	errConfig400 := get400()

	switch r.Method {
	case "GET":
		id := r.URL.Query().Get("id")

		if id == "" {
			err := errors.New("missing id query parameter")
			return &AppResp{Component: c.ComponentBuilder.Error(strconv.Itoa(errConfig400.Code), errConfig400.Title, errConfig400.Msg),
				Code: errConfig400.Code, Message: errConfig400.Msg, ContentType: "text/html", Error: err}
		}

		state, err := c.readAnalysisState(id)

		errConfig500 := get500()
		if err != nil {
			return &AppResp{Component: c.ComponentBuilder.Error(strconv.Itoa(errConfig500.Code), errConfig500.Title, errConfig500.Msg),
				Code:        errConfig500.Code,
				Message:     errConfig500.Msg,
				ContentType: "text/html",
				Error:       err}
		}

		if state.Completed() {
			op, err := c.Repo.OpRepo.Read(id)

			if err != nil {
				return &AppResp{Component: c.ComponentBuilder.Error(strconv.Itoa(errConfig500.Code), errConfig500.Title, errConfig500.Msg),
					Code:        errConfig500.Code,
					Message:     errConfig500.Msg,
					ContentType: "text/html",
					Error:       err}
			}

			suggs, err := c.Repo.SuggRepo.Read(SuggReadFilter{OptimizationId: id})

			if err != nil {
				return &AppResp{Component: c.ComponentBuilder.Error(strconv.Itoa(errConfig500.Code), errConfig500.Title, errConfig500.Msg),
					Code:        errConfig500.Code,
					Message:     errConfig500.Msg,
					ContentType: "text/html",
					Error:       err}
			}

			if op.State == "completed" {
				return &AppResp{Component: c.ComponentBuilder.Edit(op.Id, op.OriginalPrompt, op.OptimizedPrompt, op.Instructions, suggs),
					Code: 200, Message: "OK", ContentType: "text/html", Error: nil}
			}
		}

		return &AppResp{Component: c.ComponentBuilder.Loading(id, *state),
			Code: 200, Message: "OK", ContentType: "text/html", Error: nil}
	case "POST":
		parentId := r.URL.Query().Get("parent_id")
		optimizationId := uuid.New().String()

		body, err := Read(r.Body)

		if err != nil {
			return &AppResp{Component: c.ComponentBuilder.Error(strconv.Itoa(errConfig400.Code), errConfig400.Title, errConfig400.Msg),
				Code:        errConfig400.Code,
				Message:     errConfig400.Msg,
				ContentType: "text/html",
				Error:       err}
		}

		go c.run(optimizationId, parentId, body)

		return &AppResp{Component: c.ComponentBuilder.Loading(optimizationId, AnalysisState{
			CustomCompleted:             false,
			ContextualRichnessCompleted: false,
			ConcisenessCompleted:        false,
			ClarityCompleted:            false,
			ConsistencyCompleted:        false}),
			Code: 200, Message: "OK", ContentType: "text/html", Error: nil}
	default:
		errConfig := get405()
		err := errors.New("method not allowed")
		return &AppResp{Component: c.ComponentBuilder.Error(strconv.Itoa(errConfig.Code), errConfig.Title, errConfig.Msg),
			Code: errConfig.Code, Message: errConfig.Msg, ContentType: "text/html", Error: err}

	}
}

func (c CaptureController) capture(eventType string, opId string) {
	err := c.Repo.PHRepo.Capture(fmt.Sprintf("%s_%s", c.Config.Env, eventType), opId)

	if err != nil {
		slog.Error(fmt.Sprintf("Error occured: %s", err.Error()))
	}

}

type CaptureController struct {
	ComponentBuilder *ComponentBuilder
	Repo             *Repo
	Config           *Config
}

func (c CaptureController) Handle(w http.ResponseWriter, r *http.Request) *AppResp {
	errConfig400 := get400()

	switch r.Method {
	case "POST":
		eventType := r.URL.Query().Get("event_type")
		opId := r.URL.Query().Get("optimization_id")

		if eventType == "" || opId == "" {
			err := errors.New("missing query parameter")
			return &AppResp{Component: c.ComponentBuilder.Error(strconv.Itoa(errConfig400.Code), errConfig400.Title, errConfig400.Msg),
				Code: errConfig400.Code, Message: errConfig400.Msg, ContentType: "text/html", Error: err}
		}

		go c.capture(eventType, opId)

		return &AppResp{Component: nil,
			Code: 200, Message: "OK", ContentType: "text/html", Error: nil}
	default:
		errConfig := get405()
		err := errors.New("method not allowed")
		return &AppResp{Component: c.ComponentBuilder.Error(strconv.Itoa(errConfig.Code), errConfig.Title, errConfig.Msg),
			Code: errConfig.Code, Message: errConfig.Msg, ContentType: "text/html", Error: err}

	}
}
