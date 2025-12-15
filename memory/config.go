package memory

import (
	"os"
	"path/filepath"
)

// Config contient la configuration de la mémoire
type Config struct {
	// Chemin vers le fichier de stockage de la base de connaissances
	StoragePath string
	
	// Temps de rétention des entrées (en jours)
	RetentionDays int
	
	// Taille maximale de la base de connaissances (en nombre d'entrées)
	MaxEntries int
	
	// Active ou désactive la mémoire à long terme
	Enabled bool
}

// DefaultConfig retourne la configuration par défaut
func DefaultConfig() *Config {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "/home/user"
	}
	
	return &Config{
		StoragePath:   filepath.Join(homeDir, ".cline", "knowledge_base.json"),
		RetentionDays: 365,
		MaxEntries:    10000,
		Enabled:       true,
	}
}

// Validate valide la configuration
func (c *Config) Validate() error {
	if c.StoragePath == "" {
		return &ConfigError{Field: "StoragePath", Message: "le chemin de stockage ne peut pas être vide"}
	}
	
	if c.RetentionDays <= 0 {
		return &ConfigError{Field: "RetentionDays", Message: "le temps de rétention doit être supérieur à zéro"}
	}
	
	if c.MaxEntries <= 0 {
		return &ConfigError{Field: "MaxEntries", Message: "le nombre maximal d'entrées doit être supérieur à zéro"}
	}
	
	return nil
}

// ConfigError représente une erreur de configuration
type ConfigError struct {
	Field   string
	Message string
}

// Error implémente l'interface error
func (e *ConfigError) Error() string {
	return "erreur de configuration: champ '" + e.Field + "': " + e.Message
}
