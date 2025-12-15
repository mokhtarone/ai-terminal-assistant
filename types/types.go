package types

// Message représente un message dans une conversation
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest représente la requête pour une complétion de chat
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
}

// Choice représente une réponse d'API
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// Usage représente l'utilisation des tokens
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatResponse représente la réponse de l'API pour une complétion de chat
type ChatResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Model représente un modèle AI disponible
type Model struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

// ModelsResponse représente la réponse de l'API pour la liste des modèles
type ModelsResponse struct {
	Object string  `json:"object"`
	Data   []Model `json:"data"`
}
