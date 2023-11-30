package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

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
	Id string `json:"id"`
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
}

type MessageListing struct {
	Data []Message `json:"data"`
}

type Thread struct {
	Id string `json:"id"`
}

func request(method string, url string, headerProtos []string, reqBody []byte) (*Response, error) {
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
	}

	defer func() {
		err = resp.Body.Close()
		if err != nil {
			slog.Error(fmt.Sprintf(`Error occured: %s`, err.Error()))
		}
	}()

	var respBody []byte
	respBody, err = io.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	return &Response{Body: respBody, StatusCode: resp.StatusCode}, nil
}

func getAssistantId(assistantName string) (string, error) {
	// TODO: retrieve assistant id from OAI and cache

	switch assistantName {
	case "ContextualRichness":
		return "asst_BxUQqxSD8tcvQoyR6T5iom3L", nil
	case "Conciseness":
		return "asst_3q6LvmiPZyoPChdrcuqMxOvh", nil
	case "Clarity":
		return "asst_8IjCbTm7tsgCtSbhEL7E7rjB", nil
	case "Consistency":
		return "asst_221Q0E9EeazCHcGV4Qd050Gy", nil
	case "Operator":
		return "asst_qUn97Ck3zzdvNToMVAMhNzTk", nil
	default:
		return "", fmt.Errorf("assistant %s not found", assistantName)
	}
}

func createRun(runProto RunProto, threadId string, headerProtos []string) (*string, error) {
	body := []byte(fmt.Sprintf(`{"assistant_id": "%s"}`, runProto.AssistantId))
	url := fmt.Sprintf(`https://api.openai.com/v1/threads/%s/runs`, threadId)

	resp, err := request("POST", url, headerProtos, body)

	if err != nil || resp.StatusCode != 200 {
		return nil, err
	}

	var run Run
	err = json.Unmarshal(resp.Body, &run)

	if err != nil {
		return nil, err
	}

	return &run.Id, nil
}

func listMsgs(threadId string, headerProtos []string) (*[]Message, error) {
	url := fmt.Sprintf(`https://api.openai.com/v1/threads/%s/messages`, threadId)

	resp, err := request("GET", url, headerProtos, nil)

	if err != nil || resp.StatusCode != 200 {
		return nil, err
	}

	var msgs MessageListing
	err = json.Unmarshal(resp.Body, &msgs)

	if err != nil {
		return nil, err
	}

	return &msgs.Data, nil
}

func createMsg(msgProto MessageProto, threadId string, headerProtos []string) (*string, error) {
	body := []byte(fmt.Sprintf(`{"role": "%s", "content": %s}`, msgProto.Role, msgProto.Content))
	url := fmt.Sprintf(`https://api.openai.com/v1/threads/%s/messages`, threadId)

	resp, err := request("POST", url, headerProtos, body)

	if err != nil || resp.StatusCode != 200 {
		return nil, err
	}

	var msg Message
	err = json.Unmarshal(resp.Body, &msg)

	if err != nil {
		return nil, err
	}

	return &msg.Id, nil
}

func createThread(headerProtos []string) (*string, error) {
	resp, err := request("POST", "https://api.openai.com/v1/threads", headerProtos, nil)

	if err != nil || resp.StatusCode != 200 {
		return nil, err
	}

	var thread Thread
	err = json.Unmarshal(resp.Body, &thread)

	if err != nil {
		return nil, err
	}

	return &thread.Id, nil
}

func deleteThread(threadId string, headerProtos []string) error {
	url := fmt.Sprintf(`https://api.openai.com/v1/threads/%s`, threadId)
	resp, err := request("DELETE", url, headerProtos, nil)

	if err != nil || resp.StatusCode != 200 {
		return err
	}

	return nil
}

func chat(w http.ResponseWriter, r *http.Request) *apphandler.AppError {
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

	var msgContent []byte
	msgContent, err = json.Marshal(`Evaluate the contextual richness of the following model instruction:
	You are an AI research assistant designed to aid qualitative research consultants working on a project titled "Chocolate".

	You must ensure that all responses are directly drawn from the user's research data. Your responses should never be derived from general knowledge or made up, even when the question seems mundane or the topic appears trivial.
	
	If the user's research data doesn't contain the information to answer the question, you should inform the user that the research data does not have the specific answer, and refrain from generating a response based on general knowledge or assumptions. Always prioritize the user's specific research context in your responses. This is of utmost importance.
	`)

	if err != nil {
		return &apphandler.AppError{Error: err, Message: "Service temporarily unavailable.", Code: 500}
	}

	_, err = createMsg(MessageProto{Role: "user", Content: msgContent}, *threadId, headerProtos)

	if err != nil {
		return &apphandler.AppError{Error: err, Message: "Service temporarily unavailable.", Code: 500}
	}

	var assistantId string
	assistantId, err = getAssistantId("ContextualRichness")

	if err != nil {
		return &apphandler.AppError{Error: err, Message: "Service temporarily unavailable.", Code: 500}
	}

	createRun(RunProto{AssistantId: assistantId}, *threadId, headerProtos)

	var msgs *[]Message
	msgs, err = listMsgs(*threadId, headerProtos)

	if err != nil {
		return &apphandler.AppError{Error: err, Message: "Service temporarily unavailable.", Code: 500}
	}

	content := (*msgs)[len(*msgs)-1].Content
	if len(content) != 0 {
		fmt.Fprint(w, content[0].Text.Value)
		return nil
	}

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
