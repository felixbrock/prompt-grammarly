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
	"github.com/felixbrock/lemonai/internal/persistence"
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
	OriginalPrompt       string `json:"originalPrompt"`
	Instructions         string `json:"instructions"`
	OptimizationParentId string `json:"optimizationParentId"`
}

type editorReq struct {
	EditorName string `json:"editorName"`
}

type reqConfig struct {
	Method string
	Url string
	HeaderProtos []string
	Body []byte
}

func request[T any](config reqConfig, expectedResCode int) (*T, error) {
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

	entity, err := request[run]("GET", url, headerProtos, nil)

	if err != nil {
		return nil, err
	}

	return entity, nil
}

func postRun(proto runProto, threadId string, headerProtos []string) (*run, error) {
	body := []byte(fmt.Sprintf(`{"assistant_id": "%s"}`, proto.AssistantId))
	url := fmt.Sprintf("https://api.openai.com/v1/threads/%s/runs", threadId)

	entity, err := request[run]("POST", url, headerProtos, body)

	if err != nil {
		return nil, err
	}

	return entity, nil
}

func getMsgs(threadId string, headerProtos []string) (*[]message, error) {
	url := fmt.Sprintf("https://api.openai.com/v1/threads/%s/messages", threadId)

	msgs, err := request[messageListing]("GET", url, headerProtos, nil)

	if err != nil {
		return nil, err
	}

	return &msgs.Data, nil
}

func postMsg(proto messageProto, threadId string, headerProtos []string) (*string, error) {
	body := []byte(fmt.Sprintf(`{"role": "%s", "content": %s}`, proto.Role, proto.Content))
	url := fmt.Sprintf("https://api.openai.com/v1/threads/%s/messages", threadId)

	msg, err := request[message]("POST", url, headerProtos, body)

	if err != nil {
		return nil, err
	}

	return &msg.Id, nil
}

func postThread(headerProtos []string) (*string, error) {
	resp, err := request[thread]("POST", "https://api.openai.com/v1/threads", headerProtos, nil)

	if err != nil {
		return nil, err
	}

	return &resp.Id, nil
}

func deleteThread(threadId string, headerProtos []string) error {
	url := fmt.Sprintf(`https://api.openai.com/v1/threads/%s`, threadId)
	_, err := request[thread]("DELETE", url, headerProtos, nil)

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

	fmt.Printf("Waiting for %s assistant entity to complete...\n", assistant.Name)
	for entity.Status != "completed" {
		entity, err = getRun(threadId, entity.Id, headerProtos)
		if err != nil {
			return nil, err
		}
		time.Sleep(1000)
	}
	fmt.Printf("Completed %s assistant entity\n", assistant.Name)

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

func postOptimization(headerProtos []string, optimization domain.Optimization) (*string, error) {
	resp, err := request[domain.Optimization]("POST", "https://api.openai.com/v1/threads", headerProtos, nil)

	if err != nil {
		return nil, err
	}

	return &resp.Id, nil
}

func improve(threadId string, prompt []byte, targetAssistant assistant, headerProtos []string) {
	userPrompt := genAssistantUserPrompt(targetAssistant.Name, prompt)

	msg, err := runAssistant(threadId, userPrompt, targetAssistant, headerProtos)

	if err != nil {
		slog.Error(fmt.Sprintf("Error occured: %s", err.Error()))
	}

Write Operation

	slog.Info(fmt.Sprintf("%s >> %s\n", targetAssistant.Name, *msg))
}

func readJSON[T any](reader io.ReadCloser) (*T, error) {
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

	var t *T
	err = json.Unmarshal(content, &t)

	if err != nil {
		return nil, err
	}

	return t, nil
}

func optimize(body io.ReadCloser) {
	optimizationReqBody, err := readJSON[optimizationReq](body)

	if err != nil {
		slog.Error(fmt.Sprintf("Error occured: %s", err.Error()))
	}

	id := uuid.New()



	// POST /table_name HTTP/1.1
// { "col1": "value1", "col2": "value2" }

	persistence.Write(domain.Optimization{Id: id.String(),
		OriginalPrompt:  optimizationReqBody.OriginalPrompt,
		OptimizedPrompt: "",
		Instructions:    optimizationReqBody.Instructions,
		State:           "running",
		ParentId:        optimizationReqBody.OptimizationParentId})

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
		slog.Error(fmt.Sprintf("Error occured: %s", err.Error()))
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
	optimize(r.Body)

	return &ComponentResponse{Component: components.Loading("foo"), Code: 200, Message: "OK", ContentType: "text/html", Error: nil}
}
