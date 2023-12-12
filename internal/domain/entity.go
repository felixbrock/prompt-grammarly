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
	Id              string `json:"id"`
	OriginalPrompt  string `json:"original_prompt"`
	OptimizedPrompt string `json:"optimized_prompt"`
	Instructions    string `json:"instructions"`
	State           string `json:"state"`
	ParentId        string `json:"parent_id"`
}
