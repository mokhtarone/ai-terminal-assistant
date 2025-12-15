package memory

import (
	"bufio"
	"encoding/json"
	"os"
	"sync"
	"time"
)

// KnowledgeEntry représente une entrée dans la base de connaissances
type KnowledgeEntry struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Category  string    `json:"category"`
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// KnowledgeBase gère la base de connaissances à long terme
type KnowledgeBase struct {
	entries  map[string]KnowledgeEntry
	filePath string
	mu       sync.RWMutex
}

// NewKnowledgeBase crée une nouvelle base de connaissances
func NewKnowledgeBase(filePath string) (*KnowledgeBase, error) {
	kb := &KnowledgeBase{
		entries:  make(map[string]KnowledgeEntry),
		filePath: filePath,
	}
	
	// Charger les données existantes si le fichier existe
	if _, err := os.Stat(filePath); err == nil {
		if err := kb.load(); err != nil {
			return nil, err
		}
	}
	
	return kb, nil
}

// Add ajoute une nouvelle entrée à la base de connaissances
func (kb *KnowledgeBase) Add(category, key, value string, metadata map[string]string) {
	kb.mu.Lock()
	defer kb.mu.Unlock()
	
	entry := KnowledgeEntry{
		ID:        generateID(),
		Timestamp: time.Now(),
		Category:  category,
		Key:       key,
		Value:     value,
		Metadata:  metadata,
	}
	
	kb.entries[entry.ID] = entry
	
	// Sauvegarder immédiatement
	kb.save()
}

// GetByKey récupère une entrée par sa clé
func (kb *KnowledgeBase) GetByKey(key string) []KnowledgeEntry {
	kb.mu.RLock()
	defer kb.mu.RUnlock()
	
	var results []KnowledgeEntry
	for _, entry := range kb.entries {
		if entry.Key == key {
			results = append(results, entry)
		}
	}
	
	return results
}

// GetByCategory récupère toutes les entrées d'une catégorie
func (kb *KnowledgeBase) GetByCategory(category string) []KnowledgeEntry {
	kb.mu.RLock()
	defer kb.mu.RUnlock()
	
	var results []KnowledgeEntry
	for _, entry := range kb.entries {
		if entry.Category == category {
			results = append(results, entry)
		}
	}
	
	return results
}

// GetAll retourne toutes les entrées
func (kb *KnowledgeBase) GetAll() []KnowledgeEntry {
	kb.mu.RLock()
	defer kb.mu.RUnlock()
	
	var results []KnowledgeEntry
	for _, entry := range kb.entries {
		results = append(results, entry)
	}
	
	return results
}

// Save force la sauvegarde de la base de connaissances
func (kb *KnowledgeBase) Save() error {
	kb.mu.RLock()
	defer kb.mu.RUnlock()
	
	return kb.save()
}

// load charge la base de connaissances depuis le fichier
func (kb *KnowledgeBase) load() error {
	file, err := os.Open(kb.filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	decoder := json.NewDecoder(bufio.NewReader(file))
	return decoder.Decode(&kb.entries)
}

// save sauvegarde la base de connaissances dans le fichier
func (kb *KnowledgeBase) save() error {
	file, err := os.Create(kb.filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	encoder := json.NewEncoder(bufio.NewWriter(file))
	encoder.SetIndent("", "  ")
	return encoder.Encode(kb.entries)
}

// generateID génère un identifiant unique
func generateID() string {
	return time.Now().Format("20060102150405") + "-" + string(rune(time.Now().UnixNano()%26+'a'))
}
