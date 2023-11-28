package app

import (
	"encoding/json"
	"fmt"
	"io"
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
	Content string
}

type Message struct {
	Id string `json:"id"`
}

type Thread struct {
	Id string `json:"id"`
}

func request(method string, url string, headerProtos []string, reqBody io.Reader) (*Response, error) {
	req, err := http.NewRequest(method, url, reqBody)

	if err != nil {
		return nil, err
	}

	for i := 0; i < len(headerProtos); i++ {
		headerKV := strings.Split(headerProtos[i], ",")
		req.Header.Add(headerKV[0], headerKV[1])
	}

	resp, err := (&http.Client{}).Do(req)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	resBody := make([]byte, resp.ContentLength)
	_, err = resp.Body.Read(resBody)

	if err != nil {
		return nil, err
	}

	return &Response{Body: resBody, StatusCode: resp.StatusCode}, nil
}

func getAssistantId(assistantName string) (string, error) {
// TODO: retrieve assistant id from OAI and cache

switch assistantName {
	case "ContextualRichness":
		return , nil
	case "ContextualRichness":
		return []byte("assistant-id"), nil
	
	default:
		return nil, fmt.Errorf("Assistant %s not found", assistantName)



}

func createRun(runProto RunProto, threadId string, headerProtos []string) (*string, error) {
	body := strings.NewReader(fmt.Sprintf(`{"assistant_id": "%s"}`, runProto.AssistantId))
	url := fmt.Sprintf(`https://api.openai.com/v1/threads/%s/messages`, threadId)

	res, err := request("POST", url, headerProtos, body)

	if err != nil || res.StatusCode != 201 {
		return nil, err
	}

	var run Run
	err = json.Unmarshal(res.Body, &run)

	if err != nil {
		return nil, err
	}

	return &run.Id, nil
}

func createMsg(msgProto MessageProto, threadId string, headerProtos []string) (*string, error) {
	body := strings.NewReader(fmt.Sprintf(`{"role": "%s", "content": "%s"}`, msgProto.Role, msgProto.Content))
	url := fmt.Sprintf(`https://api.openai.com/v1/threads/%s/messages`, threadId)

	res, err := request("POST", url, headerProtos, body)

	if err != nil || res.StatusCode != 201 {
		return nil, err
	}

	var msg Message
	err = json.Unmarshal(res.Body, &msg)

	if err != nil {
		return nil, err
	}

	return &msg.Id, nil
}

func createThread(headerProtos []string) (*string, error) {
	res, err := request("POST", "https://api.openai.com/v1/threads", headerProtos, nil)

	if err != nil || res.StatusCode != 201 {
		return nil, err
	}

	var thread Thread
	err = json.Unmarshal(res.Body, &thread)

	if err != nil {
		return nil, err
	}

	return &thread.Id, nil
}

func deleteThread(threadId string, headerProtos []string) error {
	url := fmt.Sprintf(`https://api.openai.com/v1/threads/%s/messages`, threadId)
	res, err := request("DELETE", url, headerProtos, nil)

	if err != nil || res.StatusCode != 200 {
		return err
	}

	return nil
}

func chat(w http.ResponseWriter, r *http.Request) (*string, *apphandler.AppError) {
	headerProtos := []string{"Content-Type: application/json", "Authorization: Bearer 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", "OpenAI-Beta: assistants=v1"}

	threadId, err := createThread(headerProtos)
	defer deleteThread(*threadId, headerProtos)

	if err != nil {
		return nil, &apphandler.AppError{Error: err, Message: "Service temporarily unavailable.", Code: 500}
	}

	_, createMsgErr := createMsg(MessageProto{Role: "customer", Content: "Hello, I'm having trouble with my invoice."}, *threadId, headerProtos)

	if createMsgErr != nil {
		return nil, &apphandler.AppError{Error: createMsgErr, Message: "Service temporarily unavailable.", Code: 500}
	}

	createRun(RunProto{AssistantId:  } ,*threadId, headerProtos)

	return thread.Id, nil
	// tmpl := template.Must(template.ParseFiles("./templates/index.html"))
	// tmpl.Execute(w, nil)
}

func eval(w http.ResponseWriter, r *http.Request) *apphandler.AppError {
	v, err := http.Get("http://0.0.0.0:80")

	fmt.Println(v)
	fmt.Println(err)

	if err != nil {
		return &apphandler.AppError{Error: err, Message: "test went wrong", Code: 500}
	}
	return nil
}

// func clicked(w http.ResponseWriter, r *http.Request) {
// 	tmpl := template.Must(template.ParseFiles("./templates/fragments/button.html"))
// 	tmpl.Execute(w, nil)
// }

// func team(w http.ResponseWriter, r *http.Request) {
// 	tmpl := template.Must(template.ParseFiles("./templates/fragments/button.html"))
// 	tmpl.Execute(w, nil)

// }
