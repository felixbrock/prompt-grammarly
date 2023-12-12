package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/felixbrock/lemonai/internal/components"
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

type editorReq struct {
	EditorName string `json:"editorName"`
}

type suggestion struct {
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

func (c OptimizationController) runAssistant(threadId string, userPrompt string, assistant assistant) (*[]byte, error) {
	err := c.writeUserPrompt(threadId, userPrompt)

	if err != nil {
		return nil, err
	}

	entity, err := c.App.OAIRepo.postRun(assistant.Id, threadId)

	if err != nil {
		return nil, err
	}

	fmt.Printf("waiting for %s assistant entity to complete...\n", assistant.Name)
	for entity.Status != "completed" {
		entity, err = c.App.OAIRepo.getRun(threadId, entity.Id)
		if err != nil {
			return nil, err
		}
		time.Sleep(1000)
	}
	fmt.Printf("completed %s assistant entity\n", assistant.Name)

	var msgs *[]OAIMessage
	msgs, err = c.App.OAIRepo.getMsgs(threadId)

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
	return &bMsg, nil
}

func (c OptimizationController) writeUserPrompt(threadId string, prompt string) error {
	msgContent, err := json.Marshal(prompt)

	if err != nil {
		return err
	}

	err = c.App.OAIRepo.postMsg(MessageProto{Role: "user", Content: msgContent}, threadId)

	if err != nil {
		return err
	}

	return nil
}

func (c OptimizationController) genAssistantUserPrompt(assistantName string, prompt []byte) string {
	return fmt.Sprintf(
		`Evaluate the %s of the following model instruction:

		%s

		`, strings.ToLower(assistantName), prompt)
}

func (c OptimizationController) genOperatorUserPrompt(assistantName string, msg []byte) string {
	return fmt.Sprintf(
		`Apply the following list of suggestions to improve contextual richness to the following model instructions.

		Model Instructions:

		%s


		Suggestions:

		%s

		`, strings.ToLower(assistantName), msg)
}

func (c OptimizationController) applySuggestions(optimizationId string, threadId string, suggestions []byte) ([]byte, error) {
	operator := assistant{Id: "asst_qUn97Ck3zzdvNToMVAMhNzTk", Name: "Operator"}

	userPrompt := c.genOperatorUserPrompt(operator.Name, suggestions)

	msg, err := c.runAssistant(threadId, userPrompt, operator)

	if err != nil {
		return nil, err
	}

	fmt.Printf("%s >> %s\n", operator.Name, *msg)

	return *msg, nil
}

func (c OptimizationController) suggest(optimizationId string, threadId string, prompt []byte, targetAssistant assistant) ([]suggestion, error) {
	runId := uuid.New().String()
	run := domain.Run{
		Id:             runId,
		Type:           targetAssistant.Name,
		State:          "running",
		OptimizationId: optimizationId}

	err := c.App.RunRepo.Insert(run)

	if err != nil {
		return nil, err
	}

	defer func() {
		if err == nil {
			err = c.App.RunRepo.Update(runId, "completed")
			if err != nil {
				slog.Error(fmt.Sprintf("Error occured: %s", err.Error()))
			}
		} else {
			err = c.App.RunRepo.Update(runId, "failed")
			if err != nil {
				slog.Error(fmt.Sprintf("Error occured: %s", err.Error()))
			}
		}

	}()

	userPrompt := c.genAssistantUserPrompt(targetAssistant.Name, prompt)

	msg, err := c.runAssistant(threadId, userPrompt, targetAssistant)

	if err != nil {
		return nil, err
	}

	var suggestions *[]suggestion
	suggestions, err = ReadJSON[[]suggestion](*msg)

	suggestionRecords := make([]domain.Suggestion, len(*suggestions))
	for i := 0; i < len(*suggestions); i++ {
		suggestionRecords[i] = domain.Suggestion{
			Id:           uuid.New().String(),
			Suggestion:   (*suggestions)[i].Suggestion,
			Reasoning:    (*suggestions)[i].Reasoning,
			UserFeedback: 0,
			Target:       (*suggestions)[i].Target,
			RunId:        runId}
	}

	err = c.App.SuggestionRepo.Insert(suggestionRecords)

	if err != nil {
		return nil, err
	}

	slog.Info(fmt.Sprintf("Successfully generated %s suggestions", targetAssistant.Name))

	return *suggestions, nil
}

func (c OptimizationController) optimize(optimizationId string, threadId string, originalPrompt []byte) {
	assistants := []assistant{{Id: "asst_BxUQqxSD8tcvQoyR6T5iom3L", Name: "Contextual Richness"},
		{Id: "asst_3q6LvmiPZyoPChdrcuqMxOvh", Name: "Conciseness"},
		{Id: "asst_8IjCbTm7tsgCtSbhEL7E7rjB", Name: "Clarity"},
		{Id: "asst_221Q0E9EeazCHcGV4Qd050Gy", Name: "Consistency"},
		{Id: "asst_221Q0E9EeazCHcGV4Qd050Gy", Name: "Custom"}}
	}

	var wg sync.WaitGroup
	outputCh := make(chan int)

	goroutines := xxxx

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			c.suggest(optimizationId, threadId, []byte(optimizationReqBody.OriginalPrompt), assistants[i])
			outputCh <- output
		}(i)
	}

	go func() {
		wg.Wait()
		close(outputCh)
	}()

	var results []int
	for {
		output, ok := <-outputCh
		if !ok {
			break // Break the loop if the channel is closed
		}
		results = append(results, output)
	}

	fmt.Println("Results:", results)

	go c.apply(optimizationId, threadId, []byte(optimizationReqBody.OriginalPrompt), assistants[i])

	c.App.OptimimizationRepo.Update(optimizationId, "completed", *msg)
}

func (c OptimizationController) run(body io.ReadCloser) error {
	bbody, err := Read(body)

	if err != nil {
		return err
	}

	optimizationReqBody, err := ReadJSON[optimizationReq](bbody)

	if err != nil {
		return err
	}

	optimizationId := uuid.New().String()

	optimization := domain.Optimization{
		Id:              optimizationId,
		OriginalPrompt:  optimizationReqBody.OriginalPrompt,
		Instructions:    optimizationReqBody.Instructions,
		ParentId:        optimizationReqBody.ParentId,
		OptimizedPrompt: "",
		State:           "pending"}

	err = c.App.OptimimizationRepo.Insert(optimization)

	if err != nil {
		return err
	}

	threadId, err := c.App.OAIRepo.postThread()

	defer func() {
		err = c.App.OAIRepo.deleteThread(threadId)
		if err != nil {
			slog.Error(fmt.Sprintf("Error occured: %s", err.Error()))
		}
	}()

	if err != nil {
		return err
	}

	go c.optimize(optimizationId, threadId, []byte(optimizationReqBody.OriginalPrompt), assistants[i], headerProtos)

	fmt.Fprint(w, prompt)
}

type IndexController struct {
}

func (c IndexController) Handle(w http.ResponseWriter, r *http.Request) *AppResp {
	return &AppResp{Component: components.Index(), Code: 200, Message: "OK", ContentType: "text/html", Error: nil}
}

type AppController struct {
}

func (c AppController) Handle(w http.ResponseWriter, r *http.Request) *AppResp {
	return &AppResp{Component: components.App(), Code: 200, Message: "OK", ContentType: "text/html", Error: nil}
}

type DraftModeEditorController struct {
}

func (c DraftModeEditorController) Handle(w http.ResponseWriter, r *http.Request) *AppResp {
	return &AppResp{Component: components.DraftModeEditor(), Code: 200, Message: "OK", ContentType: "text/html", Error: nil}
}

type EditModeEditorController struct {
}

func (c EditModeEditorController) Handle(w http.ResponseWriter, r *http.Request) *AppResp {
	return &AppResp{Component: components.EditModeEditor(), Code: 200, Message: "OK", ContentType: "text/html", Error: nil}
}

type ReviewModeEditorController struct {
}

func (c ReviewModeEditorController) Handle(w http.ResponseWriter, r *http.Request) *AppResp {
	return &AppResp{Component: components.ReviewModeEditor(), Code: 200, Message: "OK", ContentType: "text/html", Error: nil}
}

type OptimizationController struct {
	App *App
}

func (c OptimizationController) Handle(w http.ResponseWriter, r *http.Request) *AppResp {
	err := c.optimize(r.Body)

	return &AppResp{Component: components.Loading("foo"), Code: 200, Message: "OK", ContentType: "text/html", Error: nil}
}
