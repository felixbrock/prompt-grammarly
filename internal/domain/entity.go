package domain

type Suggestion struct {
	Id             string `json:"id"`
	Suggestion     string `json:"suggestion"`
	Reasoning      string `json:"reasoning"`
	Target         string `json:"target"`
	Type           string `json:"type"`
	UserFeedback   int16  `json:"user_feedback"`
	RunId          string `json:"run_id"`
	OptimizationId string `json:"optimization_id"`
}

type Run struct {
	Id             string `json:"id"`
	Type           string `json:"type"`
	State          string `json:"state"`
	OptimizationId string `json:"optimization_id"`
}

type Optimization struct {
	Id              string `json:"id"`
	OriginalPrompt  string `json:"original_prompt"`
	OptimizedPrompt string `json:"optimized_prompt"`
	Instructions    string `json:"instructions"`
	State           string `json:"state"`
	ParentId        string `json:"parent_id"`
}
