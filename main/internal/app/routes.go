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

	apphandler "github.com/felixbrock/lemonai/internal/appHandler"
)

type Response struct {
	Body       []byte
	StatusCode int
}

type RunProto struct {
	AssistantId string
}

type Run struct {
	Id     string `json:"id"`
	Status string `json:"status"`
}

type MessageProto struct {
	Role    string
	Content []byte
}

type MessageContentText struct {
	Value string `json:"value"`
}

type MessageContent struct {
	Text MessageContentText `json:"text"`
}

type Message struct {
	Id      string           `json:"id"`
	Content []MessageContent `json:"content"`
	Role    string           `json:"role"`
}

type MessageListing struct {
	Data []Message `json:"data"`
}

type Thread struct {
	Id string `json:"id"`
}

type ImprovementSuggestion struct {
	Original  string `json:"original"`
	Improved  string `json:"new"`
	Reasoning string `json:"reasoning"`
}

type Improvement struct {
	OriginalPrompt         string
	ImprovementSuggestions []ImprovementSuggestion
	ImprovedPrompt         *string
}

type assistant struct {
	Id   string
	Name string
}

type chatRequest struct {
	Prompt string `json:"prompt"`
}

func request[T any](method string, url string, headerProtos []string, reqBody []byte) (*T, error) {
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
	} else if resp.StatusCode != 200 {
		return nil, errors.New("unexpected response status code error")
	}

	var t *T
	t, err = readJSON[T](resp.Body)

	if err != nil {
		return nil, err
	}

	return t, nil
}

func postRun(threadId string, runId string, headerProtos []string) (*Run, error) {
	url := fmt.Sprintf(`https://api.openai.com/v1/threads/%s/runs/%s`, threadId, runId)

	run, err := request[Run]("GET", url, headerProtos, nil)

	if err != nil {
		return nil, err
	}

	return run, nil
}

func createRun(runProto RunProto, threadId string, headerProtos []string) (*Run, error) {
	body := []byte(fmt.Sprintf(`{"assistant_id": "%s"}`, runProto.AssistantId))
	url := fmt.Sprintf(`https://api.openai.com/v1/threads/%s/runs`, threadId)

	run, err := request[Run]("POST", url, headerProtos, body)

	if err != nil {
		return nil, err
	}

	return run, nil
}

func listMsgs(threadId string, headerProtos []string) (*[]Message, error) {
	url := fmt.Sprintf(`https://api.openai.com/v1/threads/%s/messages`, threadId)

	msgs, err := request[MessageListing]("GET", url, headerProtos, nil)

	if err != nil {
		return nil, err
	}

	return &msgs.Data, nil
}

func createMsg(msgProto MessageProto, threadId string, headerProtos []string) (*string, error) {
	body := []byte(fmt.Sprintf(`{"role": "%s", "content": %s}`, msgProto.Role, msgProto.Content))
	url := fmt.Sprintf(`https://api.openai.com/v1/threads/%s/messages`, threadId)

	msg, err := request[Message]("POST", url, headerProtos, body)

	if err != nil {
		return nil, err
	}

	return &msg.Id, nil
}

func createThread(headerProtos []string) (*string, error) {
	resp, err := request[Thread]("POST", "https://api.openai.com/v1/threads", headerProtos, nil)

	if err != nil {
		return nil, err
	}

	return &resp.Id, nil
}

func deleteThread(threadId string, headerProtos []string) error {
	url := fmt.Sprintf(`https://api.openai.com/v1/threads/%s`, threadId)
	_, err := request[Thread]("DELETE", url, headerProtos, nil)

	if err != nil {
		return err
	}

	return nil
}

func runAssistant(threadId *string, userPrompt string, assistant assistant, headerProtos []string) (*[]byte, error) {
	err := writeUserPrompt(threadId, userPrompt, headerProtos)

	if err != nil {
		return nil, err
	}

	run, err := createRun(RunProto{AssistantId: assistant.Id}, *threadId, headerProtos)

	if err != nil {
		return nil, err
	}

	fmt.Printf("waiting for %s assistant run to complete...", assistant.Name)
	for run.Status != "completed" {
		run, err = postRun(*threadId, run.Id, headerProtos)
		if err != nil {
			return nil, err
		}
		time.Sleep(1000)
	}
	fmt.Printf("completed %s assistant run.", assistant.Name)

	var msgs *[]Message
	msgs, err = listMsgs(*threadId, headerProtos)

	if err != nil {
		return nil, err
	}

	msg := (*msgs)[len(*msgs)-1]
	if msg.Role != "assistant" || len(msg.Content) != 1 {
		return nil, errors.New("unexpected assistant response error")
	}

	bMsg := []byte(msg.Content[0].Text.Value)
	return &bMsg, nil
}

func writeUserPrompt(threadId *string, prompt string, headerProtos []string) error {
	msgContent, err := json.Marshal(prompt)

	if err != nil {
		return err
	}

	_, err = createMsg(MessageProto{Role: "user", Content: msgContent}, *threadId, headerProtos)

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

func improve(threadId *string, prompt []byte, targetAssistant assistant, headerProtos []string) (*[]byte, error) {
	userPrompt := genAssistantUserPrompt(targetAssistant.Name, prompt)

	msg, err := runAssistant(threadId, userPrompt, targetAssistant, headerProtos)

	if err != nil {
		return nil, err
	}

	fmt.Printf(`%s >> %s`, targetAssistant.Name, *msg)

	operator := assistant{Id: "asst_qUn97Ck3zzdvNToMVAMhNzTk", Name: "Operator"}

	userPrompt = genOperatorUserPrompt(operator.Name, *msg)

	msg, err = runAssistant(threadId, userPrompt, operator, headerProtos)

	if err != nil {
		return nil, err
	}

	fmt.Printf(`%s >> %s`, operator.Name, *msg)

	return msg, nil
}

func readJSON[T any](reader io.ReadCloser) (*T, error) {
	var err error

	defer func() {
		err = reader.Close()
		if err != nil {
			slog.Error(fmt.Sprintf(`Error occured: %s`, err.Error()))
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

func chat(w http.ResponseWriter, r *http.Request) *apphandler.AppError {
	chatReq, err := readJSON[chatRequest](r.Body)

	headerProtos := []string{
		"Content-Type:application/json",
		"Authorization:Bearer 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",
		"OpenAI-Beta:assistants=v1",
	}

	threadId, err := createThread(headerProtos)

	defer func() {
		err = deleteThread(*threadId, headerProtos)
		if err != nil {
			fmt.Printf(`Error occured: %s`, err.Error())
		}
	}()

	if err != nil {
		return &apphandler.AppError{Error: err, Message: "Service temporarily unavailable.", Code: 500}
	}

	assistants := []assistant{{Id: "asst_BxUQqxSD8tcvQoyR6T5iom3L", Name: "Contextual Richness"},
		{Id: "asst_3q6LvmiPZyoPChdrcuqMxOvh", Name: "Conciseness"},
		{Id: "asst_8IjCbTm7tsgCtSbhEL7E7rjB", Name: "Clarity"},
		{Id: "asst_221Q0E9EeazCHcGV4Qd050Gy", Name: "Consistency"}}

	prompt := []byte(chatReq.Prompt)
	for i := 0; i < len(assistants); i++ {
		var tempPrompt *[]byte
		tempPrompt, err = improve(threadId, prompt, assistants[i], headerProtos)

		if err != nil {
			return &apphandler.AppError{Error: err, Message: "Service temporarily unavailable.", Code: 500}
		}

		prompt = *tempPrompt
	}

	fmt.Fprint(w, prompt)

	return nil
	// tmpl := template.Must(template.ParseFiles("./templates/index.html"))
	// tmpl.Execute(w, nil)
}

// func eval(w http.ResponseWriter, r *http.Request) *apphandler.AppError {
// 	v, err := http.Get("http://0.0.0.0:80")

// 	fmt.Println(v)
// 	fmt.Println(err)

// 	if err != nil {
// 		return &apphandler.AppError{Error: err, Message: "test went wrong", Code: 500}
// 	}
// 	return nil
// }

// func clicked(w http.ResponseWriter, r *http.Request) {
// 	tmpl := template.Must(template.ParseFiles("./templates/fragments/button.html"))
// 	tmpl.Execute(w, nil)
// }

// func team(w http.ResponseWriter, r *http.Request) {
// 	tmpl := template.Must(template.ParseFiles("./templates/fragments/button.html"))
// 	tmpl.Execute(w, nil)

// }
