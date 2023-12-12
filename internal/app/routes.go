package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/felixbrock/lemonai/internal/components"
	"github.com/felixbrock/lemonai/internal/domain"
	"github.com/google/uuid"
)

type Response struct {
	Body       []byte
	StatusCode int
}

type runProto struct {
	AssistantId string
}

type run struct {
	Id     string `json:"id"`
	Status string `json:"status"`
}

type messageProto struct {
	Role    string
	Content []byte
}

type messageContentText struct {
	Value string `json:"value"`
}

type messageContent struct {
	Text messageContentText `json:"text"`
}

type message struct {
	Id      string           `json:"id"`
	Content []messageContent `json:"content"`
	Role    string           `json:"role"`
}

type messageListing struct {
	Data []message `json:"data"`
}

type thread struct {
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

type reqConfig struct {
	Method       string
	Url          string
	HeaderProtos []string
	Body         []byte
}

type suggestion struct {
	Suggestion string `json:"suggestion"`
	Reasoning  string `json:"reasoning"`
	Target     string `json:"original"`
}

type OptimimizationRepo interface {
	Insert(optimization domain.Optimization) error
}

type RunRepo interface {
	Insert(run domain.Run) error
	Update(id string, state string) error
}

type SuggestionRepo interface {
	Insert(suggestion []domain.Suggestion) error
}

func request[T any](config reqConfig, expectedResCode int) (T, error) {
	req, err := http.NewRequest(method, url, bytes.NewBuffer(reqBody))

	if err != nil {
		return nil, err
	}

	for i := 0; i < len(headerProtos); i++ {
		headerKV := strings.Split(headerProtos[i], ":")
		req.Header.Add(headerKV[0], headerKV[1])
	}

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return nil, err
	} else if resp.StatusCode != expectedResCode {
		return nil, errors.New("unexpected response status code error")
	}

	var t *T
	t, err = readJSON[T](resp.Body)

	if err != nil {
		return nil, err
	}

	return t, nil
}

func getRun(threadId string, runId string, headerProtos []string) (*run, error) {
	url := fmt.Sprintf("https://api.openai.com/v1/threads/%s/runs/%s", threadId, runId)

	entity, err := request[run](reqConfig{"GET", url, headerProtos, nil}, 200)

	if err != nil {
		return nil, err
	}

	return entity, nil
}

func postRun(proto runProto, threadId string, headerProtos []string) (*run, error) {
	body := []byte(fmt.Sprintf(`{"assistant_id": "%s"}`, proto.AssistantId))
	url := fmt.Sprintf("https://api.openai.com/v1/threads/%s/runs", threadId)

	entity, err := request[run](reqConfig{"POST", url, headerProtos, body}, 200)

	if err != nil {
		return nil, err
	}

	return entity, nil
}

func getMsgs(threadId string, headerProtos []string) (*[]message, error) {
	url := fmt.Sprintf("https://api.openai.com/v1/threads/%s/messages", threadId)

	msgs, err := request[messageListing](reqConfig{"GET", url, headerProtos, nil}, 200)

	if err != nil {
		return nil, err
	}

	return &msgs.Data, nil
}

func postMsg(proto messageProto, threadId string, headerProtos []string) (*string, error) {
	body := []byte(fmt.Sprintf(`{"role": "%s", "content": %s}`, proto.Role, proto.Content))
	url := fmt.Sprintf("https://api.openai.com/v1/threads/%s/messages", threadId)

	msg, err := request[message](reqConfig{"POST", url, headerProtos, body}, 200)

	if err != nil {
		return nil, err
	}

	return &msg.Id, nil
}

func postThread(headerProtos []string) (*string, error) {
	resp, err := request[thread](reqConfig{"POST", "https://api.openai.creqConfigom/v1/threads", headerProtos, nil}, 200)

	if err != nil {
		return nil, err
	}

	return &resp.Id, nil
}

func deleteThread(threadId string, headerProtos []string) error {
	url := fmt.Sprintf(`https://api.openai.com/v1/threads/%s`, threadId)
	_, err := request[thread](reqConfig{"DELETE", url, headerProtos, nil}, 200)

	if err != nil {
		return err
	}

	return nil
}

func runAssistant(threadId string, userPrompt string, assistant assistant, headerProtos []string) (*[]byte, error) {
	err := writeUserPrompt(threadId, userPrompt, headerProtos)

	if err != nil {
		return nil, err
	}

	entity, err := postRun(runProto{AssistantId: assistant.Id}, threadId, headerProtos)

	if err != nil {
		return nil, err
	}

	fmt.Printf("waiting for %s assistant entity to complete...\n", assistant.Name)
	for entity.Status != "completed" {
		entity, err = getRun(threadId, entity.Id, headerProtos)
		if err != nil {
			return nil, err
		}
		time.Sleep(1000)
	}
	fmt.Printf("completed %s assistant entity\n", assistant.Name)

	var msgs *[]message
	msgs, err = getMsgs(threadId, headerProtos)

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

func writeUserPrompt(threadId string, prompt string, headerProtos []string) error {
	msgContent, err := json.Marshal(prompt)

	if err != nil {
		return err
	}

	_, err = postMsg(messageProto{Role: "user", Content: msgContent}, threadId, headerProtos)

	if err != nil {
		return err
	}

	return nil
}

func genAssistantUserPrompt(assistantName string, prompt []byte) string {
	return fmt.Sprintf(
		`Evaluate the %s of the following model instruction:

		%s

		`, strings.ToLower(assistantName), prompt)
}

func genOperatorUserPrompt(assistantName string, prompt []byte) string {
	return fmt.Sprintf(
		`Apply the following list of suggestions to improve contextual richness to the following model instructions.

		Model Instructions:

		%s


		Suggestions:

		%s

		`, strings.ToLower(assistantName), prompt)
}

func applySuggestions() {
	operator := assistant{Id: "asst_qUn97Ck3zzdvNToMVAMhNzTk", Name: "Operator"}

	userPrompt = genOperatorUserPrompt(operator.Name, *msg)

	msg, err = runAssistant(threadId, userPrompt, operator, headerProtos)

	if err != nil {
		return nil, err
	}

	fmt.Printf("%s >> %s\n", operator.Name, *msg)

	persistence.Update(domain.Optimization{Id: id.String(),
		Prompt:       optimizationReqBody.Prompt,
		Instructions: optimizationReqBody.Instructions,
		State:        "pending",
		ParentId:     optimizationReqBody.OptimizationParentId})

}

func insertOptimization(optimization domain.Optimization) (string, error) {
	apiKey := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6ImNsbGV2bHJva2lnd3ZiYm5iZml1Iiwicm9sZSI6ImFub24iLCJpYXQiOjE3MDIyOTY5OTYsImV4cCI6MjAxNzg3Mjk5Nn0.xlOS-PwHf2vX3pULG6UtA0OZQqiyt8A3PyGTxz2LZoA"
	headerProtos := []string{
		fmt.Sprintf("apikey: %s", apiKey),
		fmt.Sprintf("Authorization: Bearer %s", apiKey)}

	body, err := json.Marshal(optimization)

	if err != nil {
		return nil, err
	}

	resp, err := request[domain.Optimization](reqConfig{"POST", "https://cllevlrokigwvbbnbfiu.supabase.co/rest/v1/optimization", headerProtos, body}, 201)

	if err != nil {
		return nil, err
	}

	return resp.Id, nil
}

func improve(threadId string, prompt []byte, targetAssistant assistant, headerProtos []string) {
	runId := uuid.New().String()
	run := domain.Run{
		Id:             runId,
		Type:           targetAssistant.Name,
		State:          "running",
		OptimizationId: optimizationId}

	err := RunRepo.Insert(run)

	if err != nil {
		slog.Error(fmt.Sprintf("Error occured: %s", err.Error()))
	}

	defer func() {
		if err == nil {
			err = RunRepo.Update(runId, "completed")
			if err != nil {
				slog.Error(fmt.Sprintf("Error occured: %s", err.Error()))
			}
		} else {
			err = RunRepo.Update(runId, "failed")
			if err != nil {
				slog.Error(fmt.Sprintf("Error occured: %s", err.Error()))
			}
		}

	}()

	userPrompt := genAssistantUserPrompt(targetAssistant.Name, prompt)

	msg, err := runAssistant(threadId, userPrompt, targetAssistant, headerProtos)

	if err != nil {
		slog.Error(fmt.Sprintf("Error occured: %s", err.Error()))
	}

	var suggestions []suggestion
	suggestions, err = readJSON[[]suggestion](bytes.NewBuffer(*msg))

	suggestionRecords := make([]domain.Suggestion, len(suggestions))
	for i := 0; i < len(suggestions); i++ {
		suggestionRecords[i] = domain.Suggestion{
			Id:           uuid.New().String(),
			Suggestion:   suggestions[i].Suggestion,
			Reasoning:    suggestions[i].Reasoning,
			UserFeedback: 0,
			Target:       suggestions[i].Target,
			RunId:        runId}
	}

	err = SuggestionRepo.Insert(suggestion)

	if err != nil {
		slog.Error(fmt.Sprintf("Error occured: %s", err.Error()))
	}

	slog.Info(fmt.Sprintf("%s >> %s\n", targetAssistant.Name, *msg))
}

func readJSON[T any](content []byte) (*T, error) {
	var t *T
	err := json.Unmarshal(content, &t)

	if err != nil {
		return nil, err
	}

	return t, nil
}

func read(reader io.ReadCloser) ([]byte, error) {
	var err error

	defer func() {
		err = reader.Close()
		if err != nil {
			slog.Error(fmt.Sprintf("Error occured: %s", err.Error()))
		}
	}()

	var content []byte
	content, err = io.ReadAll(reader)

	if err != nil {
		return nil, err
	} else if len(content) == 0 {
		return nil, errors.New("no reader content error")
	}

	return content, nil
}

func optimize(body io.ReadCloser) error {
	bbody, err := read(body)

	if err != nil {
		return err
	}

	optimizationReqBody, err := readJSON[optimizationReq](bbody)

	if err != nil {
		return err
	}

	apiKey := os.getEnv("SUPABASE_API_KEY")
	headerProtos := []string{
		fmt.Sprintf("apikey: %s", apiKey),
		fmt.Sprintf("Authorization: Bearer %s", apiKey)}

	optimizationId := uuid.New().String()

	optimization := domain.Optimization{
		Id:              optimizationId,
		OriginalPrompt:  optimizationReqBody.OriginalPrompt,
		Instructions:    optimizationReqBody.Instructions,
		ParentId:        optimizationReqBody.ParentId,
		OptimizedPrompt: "",
		State:           "pending"}

	_, err := insertOptimization(optimization)

	headerProtos := []string{
		"Content-Type:application/json",
		"Authorization:Bearer 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",
		"OpenAI-Beta:assistants=v1",
	}

	threadId, err := postThread(headerProtos)

	defer func() {
		err = deleteThread(*threadId, headerProtos)
		if err != nil {
			slog.Error(fmt.Sprintf("Error occured: %s", err.Error()))
		}
	}()

	if err != nil {
		return err
	}

	assistants := []assistant{{Id: "asst_BxUQqxSD8tcvQoyR6T5iom3L", Name: "Contextual Richness"},
		{Id: "asst_3q6LvmiPZyoPChdrcuqMxOvh", Name: "Conciseness"},
		{Id: "asst_8IjCbTm7tsgCtSbhEL7E7rjB", Name: "Clarity"},
		{Id: "asst_221Q0E9EeazCHcGV4Qd050Gy", Name: "Consistency"}}

	for i := 0; i < len(assistants); i++ {
		go improve(*threadId, []byte(optimizationReqBody.OriginalPrompt), assistants[i], headerProtos)
	}

	fmt.Fprint(w, prompt)
}

func index(w http.ResponseWriter, r *http.Request) *ComponentResponse {
	return &ComponentResponse{Component: components.Index(), Code: 200, Message: "OK", ContentType: "text/html", Error: nil}
}

func app(w http.ResponseWriter, r *http.Request) *ComponentResponse {
	return &ComponentResponse{Component: components.App(), Code: 200, Message: "OK", ContentType: "text/html", Error: nil}
}

func draftModeEditor(w http.ResponseWriter, r *http.Request) *ComponentResponse {
	return &ComponentResponse{Component: components.DraftModeEditor(), Code: 200, Message: "OK", ContentType: "text/html", Error: nil}
}
func editModeEditor(w http.ResponseWriter, r *http.Request) *ComponentResponse {
	return &ComponentResponse{Component: components.EditModeEditor(), Code: 200, Message: "OK", ContentType: "text/html", Error: nil}
}
func reviewModeEditor(w http.ResponseWriter, r *http.Request) *ComponentResponse {
	return &ComponentResponse{Component: components.ReviewModeEditor(), Code: 200, Message: "OK", ContentType: "text/html", Error: nil}
}

func handleOptimizationReq(w http.ResponseWriter, r *http.Request) *ComponentResponse {
	err := optimize(r.Body)

	return &ComponentResponse{Component: components.Loading("foo"), Code: 200, Message: "OK", ContentType: "text/html", Error: nil}
}
