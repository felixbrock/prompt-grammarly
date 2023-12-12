package persistence

import (
	"fmt"

	"github.com/felixbrock/lemonai/internal/app"
)

type OpenAIRepo struct {
	BaseHeaders []string
}

func (r OpenAIRepo) getRun(threadId string, runId string) (*app.OAIRun, error) {
	url := fmt.Sprintf("https://api.openai.com/v1/threads/%s/runs/%s", threadId, runId)

	record, err := request[app.OAIRun](reqConfig{"GET", url, r.BaseHeaders, nil}, 200)

	if err != nil {
		return nil, err
	}

	return record, nil
}

func (r OpenAIRepo) postRun(assistantId string, threadId string) (*app.OAIRun, error) {
	body := []byte(fmt.Sprintf(`{"assistant_id": "%s"}`, assistantId))
	url := fmt.Sprintf("https://api.openai.com/v1/threads/%s/runs", threadId)

	record, err := request[app.OAIRun](reqConfig{"POST", url, r.BaseHeaders, body}, 200)

	if err != nil {
		return nil, err
	}

	return record, nil
}

func (r OpenAIRepo) getMsgs(threadId string) (*[]app.OAIMessage, error) {
	url := fmt.Sprintf("https://api.openai.com/v1/threads/%s/messages", threadId)

	msgs, err := request[app.OAIMessageListing](reqConfig{"GET", url, r.BaseHeaders, nil}, 200)

	if err != nil {
		return nil, err
	}

	return &msgs.Data, nil
}

func (r OpenAIRepo) postMsg(proto app.MessageProto, threadId string) error {
	body := []byte(fmt.Sprintf(`{"role": "%s", "content": %s}`, proto.Role, proto.Content))
	url := fmt.Sprintf("https://api.openai.com/v1/threads/%s/messages", threadId)

	_, err := request[app.OAIMessage](reqConfig{"POST", url, r.BaseHeaders, body}, 200)

	if err != nil {
		return err
	}

	return nil
}

func (r OpenAIRepo) postThread() (string, error) {
	thread, err := request[app.OAIThread](reqConfig{"POST", "https://api.openai.creqConfigom/v1/threads", r.BaseHeaders, nil}, 200)

	if err != nil {
		return "", err
	}

	return thread.Id, nil
}

func (r OpenAIRepo) deleteThread(threadId string) error {
	url := fmt.Sprintf(`https://api.openai.com/v1/threads/%s`, threadId)
	_, err := request[app.OAIThread](reqConfig{"DELETE", url, r.BaseHeaders, nil}, 200)

	if err != nil {
		return err
	}

	return nil
}
