package app

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/a-h/templ"
	"github.com/felixbrock/lemonai/internal/domain"
	"golang.org/x/time/rate"
)

type ComponentBuilder struct {
	Index       func() templ.Component
	App         func() templ.Component
	Draft       func() templ.Component
	Edit        func(id string, original string, optimized string, instructions string, suggestions *[]domain.Suggestion) templ.Component
	Loading     func(optimizationId string, state AnalysisState) templ.Component
	Error       func(code string, title string, msg string) templ.Component
	FeedbackBtn func(btnId string, fType string, fVal int, suggId string) templ.Component
}

type Config struct {
	Env       string `json:"Env"`
	Port      string `json:"GO_PORT"`
	DBApiKey  string `json:"DB_API_KEY"`
	DBUrl     string `json:"DB_URL"`
	OAIApiKey string `json:"OAI_API_KEY"`
	PHApiKey  string `json:"PH_API_KEY"`
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

type phRepo interface {
	Capture(eventType string, opid string) error
}

type Repo struct {
	OpRepo   opRepo
	RunRepo  runRepo
	SuggRepo suggRepo
	OAIRepo  oaiRepo
	PHRepo   phRepo
}

type App struct {
	Repo             Repo
	ComponentBuilder ComponentBuilder
	Config           Config
}

func (a App) rateLimit(limiter *rate.Limiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !limiter.Allow() {
				http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (a App) registerEndpoints(h *http.ServeMux) {
	limiter := rate.NewLimiter(2, 2)

	h.Handle("/static/",
		http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	h.Handle("/", a.rateLimit(limiter)(AppHandler{IndexController{ComponentBuilder: &a.ComponentBuilder}}))
	h.Handle("/app", a.rateLimit(limiter)(AppHandler{AppController{ComponentBuilder: &a.ComponentBuilder}}))
	h.Handle("/editor/draft", a.rateLimit(limiter)(AppHandler{DraftModeEditorController{ComponentBuilder: &a.ComponentBuilder}}))
	h.Handle("/optimizations", a.rateLimit(limiter)(AppHandler{OptimizationController{
		ComponentBuilder: &a.ComponentBuilder,
		Repo:             &a.Repo,
		Config:           &a.Config,
	}}))
	h.Handle("/captures", a.rateLimit(limiter)(AppHandler{CaptureController{
		ComponentBuilder: &a.ComponentBuilder,
		Repo:             &a.Repo,
		Config:           &a.Config,
	}}))
}

func (a App) Start() {
	mux := http.NewServeMux()

	a.registerEndpoints(mux)

	s := &http.Server{
		Addr:              fmt.Sprintf(":%s", a.Config.Port),
		ReadHeaderTimeout: 500 * time.Millisecond,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
		Handler:           http.TimeoutHandler(mux, time.Second, "Timeout of server handler"),
	}

	log.Fatal(s.ListenAndServe())
	log.Printf("App running on %s...", a.Config.Port)
}
