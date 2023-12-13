package persistence

import (
	"fmt"

	"github.com/felixbrock/lemonai/internal/app"
)

type OpenAIRepo struct {
	BaseHeaders []string
}

func (r OpenAIRepo) GetRun(threadId string, runId string) (*app.OAIRun, error) {
	url := fmt.Sprintf("https://api.openai.com/v1/threads/%s/runs/%s", threadId, runId)

	record, err := request[app.OAIRun](reqConfig{Method: "GET", Url: url, Headers: r.BaseHeaders}, 200)

	if err != nil {
		return nil, err
	}

	return record, nil
}

func (r OpenAIRepo) PostRun(assistantId string, threadId string) (*app.OAIRun, error) {
	body := []byte(fmt.Sprintf(`{"assistant_id": "%s"}`, assistantId))
	url := fmt.Sprintf("https://api.openai.com/v1/threads/%s/runs", threadId)

	record, err := request[app.OAIRun](reqConfig{Method: "POST", Url: url, Headers: r.BaseHeaders, Body: body}, 200)

	if err != nil {
		return nil, err
	}

	return record, nil
}

func (r OpenAIRepo) GetMsgs(threadId string) (*[]app.OAIMessage, error) {
	url := fmt.Sprintf("https://api.openai.com/v1/threads/%s/messages", threadId)

	msgs, err := request[app.OAIMessageListing](reqConfig{Method: "GET", Url: url, Headers: r.BaseHeaders}, 200)

	if err != nil {
		return nil, err
	}

	return &msgs.Data, nil
}

func (r OpenAIRepo) PostMsg(proto app.MessageProto, threadId string) error {
	body := []byte(fmt.Sprintf(`{"role": "%s", "content": %s}`, proto.Role, proto.Content))
	url := fmt.Sprintf("https://api.openai.com/v1/threads/%s/messages", threadId)

	_, err := request[app.OAIMessage](reqConfig{Method: "POST", Url: url, Headers: r.BaseHeaders, Body: body}, 200)

	if err != nil {
		return err
	}

	return nil
}

func (r OpenAIRepo) PostThread() (string, error) {
	thread, err := request[app.OAIThread](reqConfig{Method: "POST", Url: "https://api.openai.creqConfigom/v1/threads", Headers: r.BaseHeaders}, 200)

	if err != nil {
		return "", err
	}

	return thread.Id, nil
}

func (r OpenAIRepo) DeleteThread(threadId string) error {
	url := fmt.Sprintf(`https://api.openai.com/v1/threads/%s`, threadId)
	_, err := request[app.OAIThread](reqConfig{Method: "DELETE", Url: url, Headers: r.BaseHeaders}, 200)

	if err != nil {
		return err
	}

	return nil
}
