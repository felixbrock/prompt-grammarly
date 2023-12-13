package app

import (
	"fmt"
	"log"
	"net/http"

	"github.com/a-h/templ"
	"github.com/felixbrock/lemonai/internal/domain"
)

type ComponentBuilder struct {
	Index   func() templ.Component
	App     func() templ.Component
	Draft   func() templ.Component
	Edit    func(original string, optimized string, instructions string, suggestions *[]domain.Suggestion) templ.Component
	Review  func(original string, optimized string) templ.Component
	Loading func(optimizationId string, state AnalysisState) templ.Component
	Error   func(code string, title string, msg string) templ.Component
}

type Config struct {
	Port      string
	OAIApiKey string
	DBApiKey  string
}

type optimimizationRepo interface {
	Insert(optimization domain.Optimization) error
	Update(id string, state string, optimizedPrompt []byte) error
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

type SuggestionReadFilter struct {
	OptimizationId string
}

type suggestionRepo interface {
	Insert(suggestions []domain.Suggestion) error
	Update(id string, userFeedback string) error
	Read(filter SuggestionReadFilter) (*[]domain.Suggestion, error)
}

type oaiRepo interface {
	GetRun(threadId string, runId string) (*OAIRun, error)
	PostRun(assistantId string, threadId string) (*OAIRun, error)
	GetMsgs(threadId string) (*[]OAIMessage, error)
	PostMsg(proto MessageProto, threadId string) error
	PostThread() (string, error)
	DeleteThread(threadId string) error
}

type App struct {
	OptimimizationRepo optimimizationRepo
	RunRepo            runRepo
	SuggestionRepo     suggestionRepo
	OAIRepo            oaiRepo
	ComponentBuilder   ComponentBuilder
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
