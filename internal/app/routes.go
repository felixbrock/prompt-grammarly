package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	OriginalPrompt string `json:"originalPrompt"`
	Instructions   string `json:"instructions"`
	ParentId       string `json:"parentId"`
}

type oaiSuggestion struct {
	Suggestion string `json:"suggestion"`
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

	fmt.Printf("waiting for %s assistant entity to complete...\n", assistant.Name)
	for entity.Status != "completed" {
		entity, err = c.Repo.OAIRepo.GetRun(threadId, entity.Id)
		if err != nil {
			return nil, err
		}
		time.Sleep(1000)
	}
	fmt.Printf("completed %s assistant entity\n", assistant.Name)

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

func (c OptimizationController) apply(threadId string, suggestions []oaiSuggestion) ([]byte, error) {
	name := "operator"
	operator := assistant{Id: "asst_qUn97Ck3zzdvNToMVAMhNzTk", Name: name}

	bSugg, err := json.Marshal(suggestions)

	if err != nil {
		return nil, err
	}

	userPrompt := c.genOperatorUserPrompt(operator.Name, bSugg)

	msg, err := c.runAssistant(threadId, userPrompt, operator)

	if err != nil {
		return nil, err
	}

	slog.Info(fmt.Sprintf("Successfully generated %s suggestions", name))

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
			return nil, errors.New("missing custom instructions error")
		}

		userPrompt = c.genCustomAssistantUserPrompt(base.Instructions, base.Prompt)
	} else {

		userPrompt = c.genAssistantUserPrompt(targetAssistant.Name, base.Prompt)
	}

	msg, err := c.runAssistant(threadId, userPrompt, targetAssistant)

	if err != nil {
		return nil, err
	}

	var suggestions *[]oaiSuggestion
	suggestions, err = ReadJSON[[]oaiSuggestion](msg)

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

	err = c.Repo.SuggestionRepo.Insert(suggestionRecords)

	if err != nil {
		return nil, err
	}

	slog.Info(fmt.Sprintf("Successfully generated %s suggestions", targetAssistant.Name))

	return *suggestions, nil
}

func (c OptimizationController) optimize(optimizationId string, threadId string, base optimizationBase) {
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
			suggestions, err := c.suggest(optimizationId, threadId, base, assistants[id])

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
	msg, err := c.apply(threadId, suggestions)

	if err != nil {
		slog.Error(fmt.Sprintf("Error occured: %s", err.Error()))
		return
	}

	c.Repo.OptimizationRepo.Update(optimizationId, "completed", msg)
}

func (c OptimizationController) run(optimizationId string, body io.ReadCloser) {
	bbody, err := Read(body)

	if err != nil {
		slog.Error(fmt.Sprintf("Error occured: %s", err.Error()))
		return
	}

	optimizationReqBody, err := ReadJSON[optimizationReq](bbody)

	if err != nil {
		slog.Error(fmt.Sprintf("Error occured: %s", err.Error()))
		return
	}

	optimization := domain.Optimization{
		Id:              optimizationId,
		OriginalPrompt:  optimizationReqBody.OriginalPrompt,
		Instructions:    optimizationReqBody.Instructions,
		ParentId:        optimizationReqBody.ParentId,
		OptimizedPrompt: "",
		State:           "pending"}

	err = c.Repo.OptimizationRepo.Insert(optimization)

	if err != nil {
		slog.Error(fmt.Sprintf("Error occured: %s", err.Error()))
		return
	}

	threadId, err := c.Repo.OAIRepo.PostThread()

	defer func() {
		err = c.Repo.OAIRepo.DeleteThread(threadId)
		if err != nil {
			slog.Error(fmt.Sprintf("Error occured: %s", err.Error()))
		}
	}()

	if err != nil {
		slog.Error(fmt.Sprintf("Error occured: %s", err.Error()))
		return
	}

	c.optimize(optimizationId, threadId, optimizationBase{Prompt: optimizationReqBody.OriginalPrompt,
		Instructions: optimizationReqBody.Instructions})
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
			state.ConsistencyCompleted = runCompleted
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

type optimizationRunRes struct {
	optimization *domain.Optimization
	suggestions  *[]domain.Suggestion
}

func (c EditModeEditorController) readOptimizationRun(optimizationId string) (*optimizationRunRes, error) {
	{
		var wg sync.WaitGroup

		opOutputCh := make(chan *domain.Optimization)
		wg.Add(1)
		go func() {
			defer wg.Done()
			optimization, err := c.Repo.OptimizationRepo.Read(optimizationId)

			if err != nil {
				slog.Error(fmt.Sprintf("Error occured: %s", err.Error()))
				return
			}

			opOutputCh <- optimization

		}()

		suggOutputCh := make(chan *[]domain.Suggestion)
		wg.Add(1)
		go func() {
			defer wg.Done()
			suggestions, err := c.Repo.SuggestionRepo.Read(SuggestionReadFilter{OptimizationId: optimizationId})

			if err != nil {
				slog.Error(fmt.Sprintf("Error occured: %s", err.Error()))
				return
			}

			suggOutputCh <- suggestions

		}()

		go func() {
			wg.Wait()
			close(opOutputCh)
			close(suggOutputCh)
		}()

		suggestions, ok := <-suggOutputCh
		if !ok {
			return nil, errors.New("error occured while retrieving suggestions")
		}

		optimization, ok := <-opOutputCh
		if !ok {
			return nil, errors.New("error occured while retrieving optimization")
		}

		return &optimizationRunRes{optimization: optimization, suggestions: suggestions}, nil
	}
}

type IndexController struct {
	ComponentBuilder *ComponentBuilder
}

func (c IndexController) Handle(w http.ResponseWriter, r *http.Request) *AppResp {
	switch r.Method {
	case "GET":
		return &AppResp{Component: c.ComponentBuilder.Index(), Code: 200, Message: "OK", ContentType: "text/html", Error: nil}
	default:
		code := 405
		title := "Method not allowed"
		msg := "Sorry, we couldn't find the page you were looking for."
		err := errors.New("method not allowed")
		return &AppResp{Component: c.ComponentBuilder.Error(strconv.Itoa(code), title, msg),
			Code: code, Message: msg, ContentType: "text/html", Error: err}
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
		code := 405
		title := "Method not allowed"
		msg := "Sorry, we couldn't find the page you were looking for."
		err := errors.New("method not allowed")
		return &AppResp{Component: c.ComponentBuilder.Error(strconv.Itoa(code), title, msg),
			Code: code, Message: msg, ContentType: "text/html", Error: err}
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
		code := 405
		title := "Method not allowed"
		msg := "Sorry, we couldn't find the page you were looking for."
		err := errors.New("method not allowed")
		return &AppResp{Component: c.ComponentBuilder.Error(strconv.Itoa(code), title, msg),
			Code: code, Message: msg, ContentType: "text/html", Error: err}

	}
}

type EditModeEditorController struct {
	ComponentBuilder *ComponentBuilder
	Repo             *Repo
}

func (c EditModeEditorController) Handle(w http.ResponseWriter, r *http.Request) *AppResp {
	switch r.Method {
	case "GET":
		id := r.URL.Query().Get("optimization_id")

		if id == "" {
			code := 400
			title := "Bad request"
			msg := "Sorry, we couldn't find the page you were looking for."
			err := errors.New("missing optimization_id query parameter")
			return &AppResp{Component: c.ComponentBuilder.Error(strconv.Itoa(code), title, msg),
				Code: code, Message: msg, ContentType: "text/html", Error: err}
		}

		runData, err := c.readOptimizationRun(id)

		serverErrCode := 500
		serverErrTitle := "Internal server error"
		serverErrMsg := "Sorry, there was an internal server error."

		if err != nil {
			return &AppResp{Component: c.ComponentBuilder.Error(strconv.Itoa(serverErrCode), serverErrTitle, serverErrMsg),
				Code:        serverErrCode,
				Message:     serverErrMsg,
				ContentType: "text/html",
				Error:       err}
		}

		return &AppResp{Component: c.ComponentBuilder.Edit(
			runData.optimization.OriginalPrompt,
			runData.optimization.OptimizedPrompt,
			runData.optimization.Instructions,
			runData.suggestions),
			Code: 200, Message: "OK", ContentType: "text/html", Error: nil}
	default:
		code := 405
		title := "Method not allowed"
		msg := "Sorry, we couldn't find the page you were looking for."
		err := errors.New("method not allowed")
		return &AppResp{Component: c.ComponentBuilder.Error(strconv.Itoa(code), title, msg),
			Code: code, Message: msg, ContentType: "text/html", Error: err}

	}
}

type ReviewModeEditorController struct {
	ComponentBuilder *ComponentBuilder
	Repo             *Repo
}

func (c ReviewModeEditorController) Handle(w http.ResponseWriter, r *http.Request) *AppResp {
	switch r.Method {
	case "GET":
		id := r.URL.Query().Get("id")

		optimization, err := c.Repo.OptimizationRepo.Read(id)

		serverErrCode := 500
		serverErrTitle := "Internal server error"
		serverErrMsg := "Sorry, there was an internal server error."
		if err != nil {
			return &AppResp{Component: c.ComponentBuilder.Error(strconv.Itoa(serverErrCode), serverErrTitle, serverErrMsg),
				Code:        serverErrCode,
				Message:     serverErrMsg,
				ContentType: "text/html",
				Error:       err}
		}

		return &AppResp{Component: c.ComponentBuilder.Review(optimization.OriginalPrompt, optimization.OptimizedPrompt),
			Code: 200, Message: "OK", ContentType: "text/html", Error: nil}
	default:
		code := 405
		title := "Method not allowed"
		msg := "Sorry, we couldn't find the page you were looking for."
		err := errors.New("method not allowed")
		return &AppResp{Component: c.ComponentBuilder.Error(strconv.Itoa(code), title, msg),
			Code: code, Message: msg, ContentType: "text/html", Error: err}

	}
}

type OptimizationController struct {
	ComponentBuilder *ComponentBuilder
	Repo             *Repo
}

func (c OptimizationController) Handle(w http.ResponseWriter, r *http.Request) *AppResp {
	switch r.Method {
	case "GET":
		id := r.URL.Query().Get("id")

		if id == "" {
			code := 400
			title := "Bad request"
			msg := "Sorry, we couldn't find the page you were looking for."
			err := errors.New("missing id query parameter")
			return &AppResp{Component: c.ComponentBuilder.Error(strconv.Itoa(code), title, msg),
				Code: code, Message: msg, ContentType: "text/html", Error: err}
		}

		state, err := c.readAnalysisState(id)

		serverErrCode := 500
		serverErrTitle := "Internal server error"
		serverErrMsg := "Sorry, there was an internal server error."

		if err != nil {
			return &AppResp{Component: c.ComponentBuilder.Error(strconv.Itoa(serverErrCode), serverErrTitle, serverErrMsg),
				Code:        serverErrCode,
				Message:     serverErrMsg,
				ContentType: "text/html",
				Error:       err}
		}

		if state.Completed() {
			optimization, err := c.Repo.OptimizationRepo.Read(id)

			if err != nil {
				return &AppResp{Component: c.ComponentBuilder.Error(strconv.Itoa(serverErrCode), serverErrTitle, serverErrMsg),
					Code:        serverErrCode,
					Message:     serverErrMsg,
					ContentType: "text/html",
					Error:       err}
			}

			return &AppResp{Component: c.ComponentBuilder.Review(optimization.OriginalPrompt, optimization.OptimizedPrompt),
				Code: 200, Message: "OK", ContentType: "text/html", Error: nil}
		}

		return &AppResp{Component: c.ComponentBuilder.Loading(id, *state),
			Code: 200, Message: "OK", ContentType: "text/html", Error: nil}
	case "POST":
		optimizationId := uuid.New().String()

		go c.run(optimizationId, r.Body)

		return &AppResp{Component: c.ComponentBuilder.Loading(optimizationId, AnalysisState{
			CustomCompleted:             false,
			ContextualRichnessCompleted: false,
			ConcisenessCompleted:        false,
			ClarityCompleted:            false,
			ConsistencyCompleted:        false}),
			Code: 200, Message: "OK", ContentType: "text/html", Error: nil}
	default:
		code := 405
		title := "Method not allowed"
		msg := "Sorry, we couldn't find the page you were looking for."
		err := errors.New("method not allowed")
		return &AppResp{Component: c.ComponentBuilder.Error(strconv.Itoa(code), title, msg),
			Code: code, Message: msg, ContentType: "text/html", Error: err}

	}

}
