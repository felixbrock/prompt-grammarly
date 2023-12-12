package app

import (
	"fmt"
	"log"
	"net/http"

	"github.com/felixbrock/lemonai/internal/domain"
)

type Config struct {
	Port      string
	OAIApiKey string
	DBApiKey  string
}

type OptimimizationRepo interface {
	Insert(optimization domain.Optimization) error
	Update(id string, state string, optimizedPrompt []byte) error
}

type RunRepo interface {
	Insert(run domain.Run) error
	Update(id string, state string) error
}

type SuggestionRepo interface {
	Insert(suggestions []domain.Suggestion) error
	Updata(id string, userFeedback string) error
}

type OAIRepo interface {
	getRun(threadId string, runId string) (*OAIRun, error)
	postRun(assistantId string, threadId string) (*OAIRun, error)
	getMsgs(threadId string) (*[]OAIMessage, error)
	postMsg(proto MessageProto, threadId string) error
	postThread() (string, error)
	deleteThread(threadId string) error
}

type App struct {
	OptimimizationRepo OptimimizationRepo
	RunRepo            RunRepo
	SuggestionRepo     SuggestionRepo
	OAIRepo            OAIRepo
	Config             Config
}

func (a App) Start() {
	http.Handle("/static/",
		http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	http.Handle("/", AppHandler{IndexController{}})
	http.Handle("/app", AppHandler{AppController{}})
	http.Handle("/editor/draft", AppHandler{DraftModeEditorController{}})
	http.Handle("/editor/edit", AppHandler{EditModeEditorController{}})
	http.Handle("/editor/review", AppHandler{ReviewModeEditorController{}})
	http.Handle("/optimizations", AppHandler{OptimizationController{
		App: &a,
	}})

	log.Printf("App running on %s...", a.Config.Port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", a.Config.Port), nil))
}
