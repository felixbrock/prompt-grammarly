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

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
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

type shotInstruct struct {
	Instruct string
	Ctx      string
}

func (c OptimizationController) getShotPrompt(wrongShots *[]domain.Suggestion) (*shotInstruct, error) {
	shots, err := json.Marshal(*wrongShots)

	if err != nil {
		return nil, err
	}

	instruct := `.

			Study the included 'Wrong Shots', which represent suggestions that you have made in your last optimization attempt. Those were not creating enough value for the user.
			Only create suggestions that are covering issues different from the ones included in the wrong shots
			`
	ctx := fmt.Sprintf(
		`Wrong Shots:

			%s
		`, shots)
	return &shotInstruct{Instruct: instruct, Ctx: ctx}, nil
}

func (c OptimizationController) genCustomAssistantUserPrompt(customInstructions string, prompt string, wrongShots *[]domain.Suggestion) (string, error) {
	var shotInstruct string
	var shotCtx string
	if *wrongShots != nil && len(*wrongShots) != 0 {
		shotInstructs, err := c.getShotPrompt(wrongShots)

		if err != nil {
			return "", err
		}

		shotInstruct = (*shotInstructs).Instruct
		shotCtx = (*shotInstructs).Ctx
	}

	return fmt.Sprintf(
		`Evaluate the the following "Model Instruction" against the "Custom Goal"%s:
		
		Custom Goal:
		%s
		

		Model Instructions:
		%s


		%s
		`, shotInstruct, customInstructions, prompt, shotCtx), nil
}

func (c OptimizationController) genAssistantUserPrompt(assistantName string, prompt string, wrongShots *[]domain.Suggestion) (string, error) {

	var shotInstruct string
	var shotCtx string
	if *wrongShots != nil && len(*wrongShots) != 0 {
		shotInstructs, err := c.getShotPrompt(wrongShots)

		if err != nil {
			return "", err
		}

		shotInstruct = (*shotInstructs).Instruct
		shotCtx = (*shotInstructs).Ctx
	}

	return fmt.Sprintf(
		`Evaluate the %s of the following "Model Instructions"%s:

		Model Instructions:

		%s

		%s
		`, strings.Join(strings.Split(assistantName, "_"), " "), shotInstruct, prompt, shotCtx), nil
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

type suggestArgs struct {
	WrongShots *[]domain.Suggestion
	Base       optimizationBase
	Assistant  assistant
	OpId       string
	ThId       string
}

func (c OptimizationController) suggest(args suggestArgs) ([]oaiSuggestion, error) {
	runId := uuid.New().String()
	run := domain.Run{
		Id:             runId,
		Type:           args.Assistant.Name,
		State:          "running",
		OptimizationId: args.OpId}

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
	if args.Assistant.Name == "custom" {
		if args.Base.Instructions == "" {
			return []oaiSuggestion{}, nil
		}

		userPrompt, err = c.genCustomAssistantUserPrompt(args.Base.Instructions, args.Base.Prompt, args.WrongShots)

		if err != nil {
			return nil, err
		}
	} else {
		userPrompt, err = c.genAssistantUserPrompt(args.Assistant.Name, args.Base.Prompt, args.WrongShots)

		if err != nil {
			return nil, err
		}
	}

	msg, err := c.runAssistant(args.ThId, userPrompt, args.Assistant)

	if err != nil {
		return nil, err
	} else if len(msg) == 0 {
		// handling timed out assistant runs
		return make([]oaiSuggestion, 0), nil
	}

	var suggestions *[]oaiSuggestion
	suggestions, err = ReadJSON[[]oaiSuggestion](msg)

	if err != nil {
		slog.Warn(fmt.Sprintf("Assistant %s produced unparseable JSON suggestions. Ignoring suggestions...", args.Assistant.Name))
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
			Type:           args.Assistant.Name,
			RunId:          runId,
			OptimizationId: args.OpId}
	}

	err = c.Repo.SuggRepo.Insert(suggestionRecords)

	if err != nil {
		return nil, err
	}

	slog.Info(fmt.Sprintf("Successfully generated %s suggestions", args.Assistant.Name))

	return *suggestions, nil
}

func (c OptimizationController) groupByType(shots *[]domain.Suggestion, shotsByType *map[string][]domain.Suggestion) {
	for i := 0; i < len(*shots); i++ {
		shot := (*shots)[i]
		if _, ok := (*shotsByType)[shot.Type]; ok {
			(*shotsByType)[shot.Type] = append((*shotsByType)[shot.Type], shot)
		} else {
			(*shotsByType)[shot.Type] = []domain.Suggestion{shot}
		}
	}
}

func (c OptimizationController) optimize(opId string, parentId string, base optimizationBase) {
	assistants := []assistant{{Id: "asst_BxUQqxSD8tcvQoyR6T5iom3L", Name: "contextual_richness"},
		{Id: "asst_3q6LvmiPZyoPChdrcuqMxOvh", Name: "conciseness"},
		{Id: "asst_8IjCbTm7tsgCtSbhEL7E7rjB", Name: "clarity"},
		{Id: "asst_221Q0E9EeazCHcGV4Qd050Gy", Name: "consistency"},
		{Id: "asst_9zcQxyRh4E10Agg08p8mYDO8", Name: "custom"}}

	shotsByType := make(map[string][]domain.Suggestion)
	if parentId != "" {
		wrongShots, err := c.Repo.SuggRepo.Read(SuggReadFilter{OpIdCond: fmt.Sprintf("eq.%s", parentId), UFeedbCond: "eq.-1"})

		if err != nil {
			slog.Error(fmt.Sprintf("Error occured: %s", err.Error()))
			return
		}

		c.groupByType(wrongShots, &shotsByType)
	}

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

			shots := shotsByType[assistants[id].Name]
			suggestions, err := c.suggest(suggestArgs{OpId: opId, ThId: thId, Base: base, Assistant: assistants[id], WrongShots: &shots})

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

type SuggestionController struct {
	ComponentBuilder *ComponentBuilder
	Repo             *Repo
	Config           *Config
}

func (c SuggestionController) Handle(w http.ResponseWriter, r *http.Request) *AppResp {
	switch r.Method {
	case "PATCH":
		id := r.URL.Query().Get("sugg_id")
		opId := r.URL.Query().Get("op_id")
		fVal := r.URL.Query().Get("feedb_val")

		if id == "" || opId == "" || fVal == "" {
			errConfig400 := get400()
			err := errors.New("missing query parameter")
			return &AppResp{Component: c.ComponentBuilder.Error(strconv.Itoa(errConfig400.Code), errConfig400.Title, errConfig400.Msg),
				Code: errConfig400.Code, Message: errConfig400.Msg, ContentType: "text/html", Error: err}
		}

		fValI, err := strconv.Atoi(fVal)

		errConfig500 := get500()
		if err != nil {
			return &AppResp{Component: c.ComponentBuilder.Error(strconv.Itoa(errConfig500.Code), errConfig500.Title, errConfig500.Msg),
				Code:        errConfig500.Code,
				Message:     errConfig500.Msg,
				ContentType: "text/html",
				Error:       err}
		}

		err = c.Repo.SuggRepo.Update(id, int16(fValI))

		if err != nil {
			return &AppResp{Component: c.ComponentBuilder.Error(strconv.Itoa(errConfig500.Code), errConfig500.Title, errConfig500.Msg),
				Code:        errConfig500.Code,
				Message:     errConfig500.Msg,
				ContentType: "text/html",
				Error:       err}
		}

		suggs, err := c.Repo.SuggRepo.Read(SuggReadFilter{OpIdCond: fmt.Sprintf("eq.%s", opId), UFeedbCond: fmt.Sprintf("gt.%d", fValI)})

		if err != nil {
			return &AppResp{Component: c.ComponentBuilder.Error(strconv.Itoa(errConfig500.Code), errConfig500.Title, errConfig500.Msg),
				Code:        errConfig500.Code,
				Message:     errConfig500.Msg,
				ContentType: "text/html",
				Error:       err}
		}

		return &AppResp{Component: c.ComponentBuilder.SuggestionWindow(suggs),
			Code: 200, Message: "OK", ContentType: "text/html", Error: nil}
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

			suggs, err := c.Repo.SuggRepo.Read(SuggReadFilter{OpIdCond: fmt.Sprintf("eq.%s", id)})

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
