package memory

import (
	"fmt"
	"strings"
)

// KnowledgeIntegrator gère l'intégration de la base de connaissances avec l'agent
type KnowledgeIntegrator struct {
	knowledgeBase *KnowledgeBase
}

// NewKnowledgeIntegrator crée un nouvel intégrateur de connaissances
func NewKnowledgeIntegrator(kb *KnowledgeBase) *KnowledgeIntegrator {
	return &KnowledgeIntegrator{
		knowledgeBase: kb,
	}
}

// Remember permet à l'agent de se souvenir d'informations importantes
func (ki *KnowledgeIntegrator) Remember(category, key, value string, metadata map[string]string) {
	ki.knowledgeBase.Add(category, key, value, metadata)
}

// Recall permet à l'agent de se souvenir d'informations passées
func (ki *KnowledgeIntegrator) Recall(key string) []KnowledgeEntry {
	return ki.knowledgeBase.GetByKey(key)
}

// RecallByCategory permet de récupérer toutes les connaissances d'une catégorie
func (ki *KnowledgeIntegrator) RecallByCategory(category string) []KnowledgeEntry {
	return ki.knowledgeBase.GetByCategory(category)
}

// LearnFromInteraction enregistre les interactions utilisateur pour améliorer la mémoire
func (ki *KnowledgeIntegrator) LearnFromInteraction(userInput, aiResponse string) {
	metadata := map[string]string{
		"input": userInput,
		"source": "interaction",
	}
	
	// Extraire les informations importantes de l'interaction
	keywords := extractKeywords(userInput + " " + aiResponse)
	for _, keyword := range keywords {
		ki.knowledgeBase.Add("interaction", keyword, aiResponse, metadata)
	}
}

// ExtractKeywords extrait les mots-clés d'un texte
func extractKeywords(text string) []string {
	// Liste de mots à ignorer
	stopWords := map[string]bool{
		"le": true, "la": true, "les": true, "un": true, "une": true, "des": true,
		"et": true, "ou": true, "mais": true, "donc": true, "or": true, "ni": true, "car": true,
		"à": true, "de": true, "en": true, "dans": true, "par": true, "pour": true, "avec": true,
		"sur": true, "sous": true, "entre": true, "avant": true, "après": true, "pendant": true,
		"comme": true, "que": true, "qui": true, "quoi": true, "quand": true, "où": true, "comment": true,
		"quel": true, "quelle": true, "quels": true, "quelles": true, "ce": true, "cette": true, "ces": true,
		"il": true, "elle": true, "ils": true, "elles": true, "nous": true, "vous": true, "je": true, "tu": true,
		"me": true, "te": true, "se": true, "lui": true, "leur": true,
		"y": true, "ci": true, "là": true, "ici": true, "là-bas": true,
	}
	
	words := strings.Fields(strings.ToLower(text))
	keywords := make([]string, 0)
	
	for _, word := range words {
		// Enlever la ponctuation
		word = strings.Trim(word, ".,;:!?\"'()[]{}")
		
		// Vérifier si c'est un mot important
		if len(word) > 3 && !stopWords[word] {
			// Vérifier si ce n'est pas déjà dans les mots-clés
			found := false
			for _, k := range keywords {
				if k == word {
					found = true
					break
				}
			}
			if !found {
				keywords = append(keywords, word)
			}
		}
	}
	
	return keywords
}

// SearchKnowledge permet de chercher dans la base de connaissances
func (ki *KnowledgeIntegrator) SearchKnowledge(query string) []KnowledgeEntry {
	results := make([]KnowledgeEntry, 0)
	
	// Rechercher dans toutes les entrées
	for _, entry := range ki.knowledgeBase.GetAll() {
		if strings.Contains(strings.ToLower(entry.Value), strings.ToLower(query)) ||
		   strings.Contains(strings.ToLower(entry.Key), strings.ToLower(query)) {
			results = append(results, entry)
		}
	}
	
	return results
}

// FormatKnowledgeResponse formate une réponse basée sur les connaissances
func (ki *KnowledgeIntegrator) FormatKnowledgeResponse(entries []KnowledgeEntry) string {
	if len(entries) == 0 {
		return "Je ne trouve pas d'informations pertinentes dans ma mémoire à long terme."
	}
	
	var sb strings.Builder
	sb.WriteString("Informations trouvées dans ma mémoire à long terme :\n\n")
	
	for i, entry := range entries {
		if i >= 5 { // Limiter à 5 résultats
			break
		}
		sb.WriteString(fmt.Sprintf("%d. %s : %s\n", i+1, entry.Key, entry.Value))
		if entry.Category != "" {
			sb.WriteString(fmt.Sprintf("   Catégorie: %s\n", entry.Category))
		}
		if entry.Metadata != nil {
			for k, v := range entry.Metadata {
				sb.WriteString(fmt.Sprintf("   %s: %s\n", k, v))
			}
		}
		sb.WriteString("\n")
	}
	
	return sb.String()
}
