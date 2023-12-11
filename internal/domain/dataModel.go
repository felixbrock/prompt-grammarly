package domain

type Suggestion struct {
	Id           string
	Suggestion   string
	Reasoning    string
	Target       string
	UserFeedback int16
	RunId        string
}

type Run struct {
	Id             string
	Type           string
	State          string
	OptimizationId string
}

type Optimization struct {
	Id              string
	OriginalPrompt  string
	OptimizedPrompt string
	Instructions    string
	State           string
	ParentId        string
}
