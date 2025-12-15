package api

import (
	"asione-agent/types"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
)

// Client gère les appels à l'API OpenAI compatible
type Client struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
	model      string
}

// NewClient crée un nouveau client API
func NewClient(baseURL, apiKey, model string) *Client {
	return &Client{
		httpClient: &http.Client{},
		baseURL:    baseURL,
		apiKey:     apiKey,
		model:      model,
	}
}

// SetCredentials met à jour les informations d'authentification
func (c *Client) SetCredentials(baseURL, apiKey, model string) {
	c.baseURL = baseURL
	c.apiKey = apiKey
	c.model = model
}

// ChatCompletion effectue un appel de complétion de chat
func (c *Client) ChatCompletion(ctx context.Context, messages []types.Message) (*types.ChatResponse, error) {
	// Utiliser la valeur MAX_TOKENS depuis .env comme limite supérieure
	// mais permettre au modèle de retourner des réponses complètes sans troncature
	maxTokens := 8192
	if val, exists := os.LookupEnv("MAX_TOKENS"); exists {
		if parsedVal, err := strconv.Atoi(val); err == nil {
			maxTokens = parsedVal
		}
	}

	// Construction de la requête
	requestBody := types.ChatRequest{
		Model:       c.model,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: 0.7,
	}

	// Sérialisation du corps de la requête
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la sérialisation de la requête: %w", err)
	}

	// Création de la requête HTTP
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la création de la requête: %w", err)
	}

	// Définition des headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	// Envoi de la requête
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de l'envoi de la requête: %w", err)
	}
	defer resp.Body.Close()

	// Lecture du corps de la réponse
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la lecture de la réponse: %w", err)
	}

	// Vérification du statut HTTP
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("erreur API: %s (code: %d) - %s", resp.Status, resp.StatusCode, string(respBody))
	}

	// Désérialisation de la réponse
	var chatResp types.ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("erreur lors de la désérialisation de la réponse: %w - %s", err, string(respBody))
	}

	return &chatResp, nil
}

// ListModels récupère la liste des modèles disponibles
func (c *Client) ListModels(ctx context.Context) (*types.ModelsResponse, error) {
	// Création de la requête HTTP
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/models", nil)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la création de la requête: %w", err)
	}

	// Définition des headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	// Envoi de la requête
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de l'envoi de la requête: %w", err)
	}
	defer resp.Body.Close()

	// Lecture du corps de la réponse
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la lecture de la réponse: %w", err)
	}

	// Vérification du statut HTTP
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("erreur API: %s (code: %d) - %s", resp.Status, resp.StatusCode, string(respBody))
	}

	// Désérialisation de la réponse
	var modelsResp types.ModelsResponse
	if err := json.Unmarshal(respBody, &modelsResp); err != nil {
		return nil, fmt.Errorf("erreur lors de la désérialisation de la réponse: %w - %s", err, string(respBody))
	}

	return &modelsResp, nil
}
