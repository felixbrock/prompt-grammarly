package app

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/a-h/templ"
	"github.com/felixbrock/lemonai/internal/domain"
)

type ComponentBuilder struct {
	Index       func() templ.Component
	App         func() templ.Component
	Draft       func() templ.Component
	Edit        func(id string, original string, optimized string, instructions string, suggestions *[]domain.Suggestion) templ.Component
	Review      func(opId string, original string, optimized string) templ.Component
	Loading     func(optimizationId string, state AnalysisState) templ.Component
	Error       func(code string, title string, msg string) templ.Component
	FeedbackBtn func(btnId string, fType string, fVal int, suggId string) templ.Component
}

type Config struct {
	Port      string `json:"GO_PORT"`
	OAIApiKey string `json:"OAI_API_KEY"`
	DBApiKey  string `json:"DB_API_KEY"`
}

type OpUpdateOpts struct {
	State           string `json:"state"`
	OptimizedPrompt string `json:"optimized_prompt"`
	ParentId        string `json:"parent_id"`
}

type opRepo interface {
	Insert(optimization domain.Optimization) error
	Update(id string, opts OpUpdateOpts) error
	Read(id string) (*domain.Optimization, error)
}

type RunReadFilter struct {
	OptimizationId string
}

type runRepo interface {
	Insert(run domain.Run) error
	Update(id string, state string) error
	Read(filter RunReadFilter) (*[]domain.Run, error)
}

type SuggReadFilter struct {
	OptimizationId string
}

type suggRepo interface {
	Insert(suggestions []domain.Suggestion) error
	Update(id string, userFeedback int16) error
	Read(filter SuggReadFilter) (*[]domain.Suggestion, error)
}

type oaiRepo interface {
	GetRun(threadId string, runId string) (*OAIRun, error)
	PostRun(assistantId string, threadId string) (*OAIRun, error)
	GetMsgs(threadId string) (*[]OAIMessage, error)
	PostMsg(proto MessageProto, threadId string) error
	PostThread() (string, error)
	DeleteThread(threadId string) error
}

type Repo struct {
	OpRepo   opRepo
	RunRepo  runRepo
	SuggRepo suggRepo
	OAIRepo  oaiRepo
}

type App struct {
	Repo             Repo
	ComponentBuilder ComponentBuilder
	Config           Config
}

func (a App) Start() {

	mux := http.NewServeMux()

	mux.Handle("/static/",
		http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	mux.Handle("/", AppHandler{IndexController{ComponentBuilder: &a.ComponentBuilder}})
	mux.Handle("/app", AppHandler{AppController{ComponentBuilder: &a.ComponentBuilder}})
	mux.Handle("/editor/draft", AppHandler{DraftModeEditorController{ComponentBuilder: &a.ComponentBuilder}})
	mux.Handle("/editor/edit", AppHandler{EditModeEditorController{ComponentBuilder: &a.ComponentBuilder, Repo: &a.Repo}})
	mux.Handle("/editor/review", AppHandler{ReviewModeEditorController{ComponentBuilder: &a.ComponentBuilder, Repo: &a.Repo}})
	mux.Handle("/optimizations", AppHandler{OptimizationController{
		ComponentBuilder: &a.ComponentBuilder, Repo: &a.Repo,
	}})

	s := &http.Server{
		Addr:              fmt.Sprintf(":%s", a.Config.Port),
		ReadHeaderTimeout: 500 * time.Millisecond,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
		Handler:           http.TimeoutHandler(mux, time.Second, "foo"),
	}

	log.Fatal(s.ListenAndServe())
	log.Printf("App running on %s...", a.Config.Port)
}
