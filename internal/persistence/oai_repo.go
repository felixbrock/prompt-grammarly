package persistence

import (
	"context"
	"fmt"

	"github.com/felixbrock/prompt-grammarly/internal/app"
)

type OAIRepo struct {
	BaseHeaders []string
}

func (r OAIRepo) GetRun(threadId string, runId string) (*app.OAIRun, error) {

	url := fmt.Sprintf("https://api.openai.com/v1/threads/%s/runs/%s", threadId, runId)

	record, err := request[app.OAIRun](context.TODO(), reqConfig{Method: "GET", Url: url, Headers: r.BaseHeaders}, 200)

	if err != nil {
		return nil, err
	}

	return record, nil
}

func (r OAIRepo) PostRun(assistantId string, threadId string) (*app.OAIRun, error) {
	body := []byte(fmt.Sprintf(`{"assistant_id": "%s"}`, assistantId))
	url := fmt.Sprintf("https://api.openai.com/v1/threads/%s/runs", threadId)

	record, err := request[app.OAIRun](context.TODO(), reqConfig{Method: "POST", Url: url, Headers: r.BaseHeaders, Body: body}, 200)

	if err != nil {
		return nil, err
	}

	return record, nil
}

func (r OAIRepo) GetMsgs(threadId string) (*[]app.OAIMessage, error) {
	url := fmt.Sprintf("https://api.openai.com/v1/threads/%s/messages", threadId)

	msgs, err := request[app.OAIMessageListing](context.TODO(), reqConfig{Method: "GET", Url: url, Headers: r.BaseHeaders}, 200)

	if err != nil {
		return nil, err
	}

	return &msgs.Data, nil
}

func (r OAIRepo) PostMsg(proto app.MessageProto, threadId string) error {
	body := []byte(fmt.Sprintf(`{"role": "%s", "content": %s}`, proto.Role, proto.Content))
	url := fmt.Sprintf("https://api.openai.com/v1/threads/%s/messages", threadId)

	_, err := request[app.OAIMessage](context.TODO(), reqConfig{Method: "POST", Url: url, Headers: r.BaseHeaders, Body: body}, 200)

	if err != nil {
		return err
	}

	return nil
}

func (r OAIRepo) PostThread() (string, error) {
	thread, err := request[app.OAIThread](context.TODO(), reqConfig{Method: "POST", Url: "https://api.openai.com/v1/threads", Headers: r.BaseHeaders}, 200)

	if err != nil {
		return "", err
	}

	return thread.Id, nil
}

func (r OAIRepo) DeleteThread(threadId string) error {
	url := fmt.Sprintf(`https://api.openai.com/v1/threads/%s`, threadId)
	_, err := request[app.OAIThread](context.TODO(), reqConfig{Method: "DELETE", Url: url, Headers: r.BaseHeaders}, 200)

	if err != nil {
		return err
	}

	return nil
}
