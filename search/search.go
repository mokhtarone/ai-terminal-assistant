package search

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// WebSearcher gère les recherches sur Internet
type WebSearcher struct {
	apiKey string
	engine string // "google", "bing", etc.
}

// NewWebSearcher crée un nouveau WebSearcher
func NewWebSearcher(apiKey, engine string) *WebSearcher {
	if engine == "" {
		engine = "google"
	}
	return &WebSearcher{
		apiKey: apiKey,
		engine: engine,
	}
}

// Search effectue une recherche sur Internet
func (w *WebSearcher) Search(ctx context.Context, query string) (*SearchResults, error) {
	// Vérification des dépendances
	if w.apiKey == "" {
		return nil, fmt.Errorf("clé API de recherche non configurée")
	}

	// Utilisation de SerpAPI qui supporte plusieurs moteurs de recherche
	params := url.Values{}
	params.Set("q", query)
	params.Set("api_key", w.apiKey)
	params.Set("engine", w.engine) // Utiliser le moteur configuré
	params.Set("num", "10")

	// Construction de l'URL (SerpAPI)
	endpoint := "https://serpapi.com/search?" + params.Encode()

	// Création et envoi de la requête
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la création de la requête: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de l'envoi de la requête: %w", err)
	}
	defer resp.Body.Close()

	// Lecture de la réponse
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la lecture de la réponse: %w", err)
	}

	// Vérification du statut
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("erreur de recherche: %s (code: %d)", resp.Status, resp.StatusCode)
	}

	// Désérialisation de la réponse
	var searchResp SearchResults
	if err := json.Unmarshal(body, &searchResp); err != nil {
		return nil, fmt.Errorf("erreur lors de la désérialisation de la réponse: %w", err)
	}

	return &searchResp, nil
}

// SearchResults représente la réponse de la recherche
type SearchResults struct {
	Kind string `json:"kind"`
	URL  struct {
		Type     string `json:"type"`
		Template string `json:"template"`
	} `json:"url"`
	Queries struct {
		NextPage []struct {
			Title           string `json:"title"`
			TotalResults    string `json:"totalResults"`
			SearchTerms     string `json:"searchTerms"`
			Count           int    `json:"count"`
			SearchTime      string `json:"searchTime"`
			InputEncoding   string `json:"inputEncoding"`
			OutputEncoding  string `json:"outputEncoding"`
			Safe            string `json:"safe"`
			Cx              string `json:"cx"`
			SearchType      string `json:"searchType"`
			StartIndex      int    `json:"startIndex"`
			SearchIntervals []int  `json:"searchIntervals"`
		} `json:"nextPage"`
		Request []struct {
			Title        string `json:"title"`
			TotalResults string `json:"totalResults"`
			SearchTerms  string `json:"searchTerms"`
			Count        int    `json:"count"`
			SearchType   string `json:"searchType"`
			Safe         string `json:"safe"`
			Cx           string `json:"cx"`
		} `json:"request"`
	} `json:"queries"`
	Context struct {
		Title string `json:"title"`
	} `json:"context"`
	SearchInformation struct {
		SearchTime            float64 `json:"searchTime"`
		FormattedSearchTime   string  `json:"formattedSearchTime"`
		TotalResults          string  `json:"totalResults"`
		FormattedTotalResults string  `json:"formattedTotalResults"`
	} `json:"searchInformation"`
	Items []SearchItem `json:"items"`
}

// SearchItem représente un résultat de recherche
type SearchItem struct {
	Kind          string `json:"kind"`
	Title         string `json:"title"`
	HTMLTitle     string `json:"htmlTitle"`
	Link          string `json:"link"`
	DisplayLink   string `json:"displayLink"`
	Snippet       string `json:"snippet"`
	HTMLSnippet   string `json:"htmlSnippet"`
	CacheID       string `json:"cacheId"`
	CachedPageURL string `json:"cachedPageUrl"`
	FormattedURL  string `json:"formattedUrl"`
}

// FormatSearchResults formate les résultats de recherche en texte
func FormatSearchResults(results *SearchResults) string {
	if results == nil || len(results.Items) == 0 {
		return "Aucun résultat trouvé pour cette recherche."
	}

	var sb strings.Builder
	sb.WriteString("Résultats de recherche:\n")
	sb.WriteString(fmt.Sprintf("Temps de recherche: %.2f secondes\n", results.SearchInformation.SearchTime))
	sb.WriteString(fmt.Sprintf("Résultats totaux: %s\n\n", results.SearchInformation.FormattedTotalResults))

	for i, item := range results.Items {
		if i >= 5 { // Limiter à 5 résultats
			break
		}
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, item.Title))
		sb.WriteString(fmt.Sprintf("   %s\n", item.Snippet))
		sb.WriteString(fmt.Sprintf("   [%s]\n\n", item.Link))
	}

	return sb.String()
}
