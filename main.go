package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"net/smtp"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"asione-agent/api"
	"asione-agent/memory"
	"asione-agent/search"
	"asione-agent/types"
)

// SystemInfo stocke les informations du système détectées au démarrage
type SystemInfo struct {
	OSName        string
	OSVersion     string
	OSID          string
	OSBuild       string
	KernelVersion string
	Architecture  string
}

// Agent représente notre agent intelligent avec terminal intégré
type Agent struct {
	// Configuration pour les appels API
	APIConfig struct {
		BaseURL string
		APIKey  string
		Model   string
	}

	// Configuration SMTP pour l'envoi d'emails
	SMTPConfig struct {
		Host     string
		Port     int
		Username string
		Password string
		From     string
	}

	// Client API
	apiClient *api.Client

	// Recherche Internet
	webSearcher *search.WebSearcher

	// Scanner pour lire les entrées utilisateur
	scanner *bufio.Scanner

	// Historique des messages pour maintenir le contexte
	messages []types.Message

	// Base de connaissances pour la mémoire à long terme
	knowledgeBase *memory.KnowledgeBase

	// Intégrateur de connaissances
	knowledgeIntegrator *memory.KnowledgeIntegrator

	// Informations système détectées au démarrage
	systemInfo *SystemInfo
}

// detectSystemInfo détecte les informations du système (OS, kernel, architecture)
// à partir de /etc/os-release et `uname`
func detectSystemInfo() *SystemInfo {
	info := &SystemInfo{}

	// Lire /etc/os-release
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		fmt.Printf("Avertissement: Impossible de lire /etc/os-release: %v\n", err)
		return info
	}

	// Parser les valeurs
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "NAME=") {
			info.OSName = strings.Trim(strings.TrimPrefix(line, "NAME="), "\"")
		} else if strings.HasPrefix(line, "VERSION=") {
			info.OSVersion = strings.Trim(strings.TrimPrefix(line, "VERSION="), "\"")
		} else if strings.HasPrefix(line, "ID=") {
			info.OSID = strings.Trim(strings.TrimPrefix(line, "ID="), "\"")
		} else if strings.HasPrefix(line, "BUILD_ID=") {
			info.OSBuild = strings.Trim(strings.TrimPrefix(line, "BUILD_ID="), "\"")
		}
	}

	// Récupérer les infos noyau et architecture via uname
	cmd := exec.Command("uname", "-srm")
	out, err := cmd.Output()
	if err != nil {
		fmt.Printf("Avertissement: Impossible d'exécuter uname: %v\n", err)
		return info
	}

	parts := strings.Fields(strings.TrimSpace(string(out)))
	if len(parts) >= 3 {
		info.KernelVersion = parts[2]
		info.Architecture = parts[1]
	}

	return info
}

// NewAgent crée une nouvelle instance d'agent
func NewAgent() *Agent {
	// Charger manuellement les variables d'environnement depuis le fichier .env
	absPath := "/home/arch/Desktop/asione-agent/.env"

	// Lire le fichier .env directement
	content, err := os.ReadFile(absPath)
	if err != nil {
		fmt.Printf("Erreur: Impossible de lire le fichier .env: %v\n", err)
		fmt.Println("Les variables d'environnement seront chargées depuis les valeurs par défaut ou définies manuellement.")
	} else {
		// Supprimer les messages individuels de définition des variables
		// et ne pas afficher le contenu du fichier .env

		// Parser les lignes du fichier
		lines := strings.Split(string(content), "\n")

		// Définir les variables d'environnement
		for _, line := range lines {
			// Ignorer les lignes vides et les commentaires
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}

			// Trouver le premier signe égal
			if i := strings.Index(line, "="); i != -1 {
				key := line[:i]
				value := line[i+1:]

				// Enlever les guillemets autour de la valeur si présents
				value = strings.Trim(value, "\"'")

				// Définir la variable d'environnement
				if err := os.Setenv(key, value); err != nil {
					// Ne pas afficher les erreurs individuelles non plus
				}
			}
		}
		// Indiquer que le fichier .env a été chargé sans détailler les variables
		fmt.Println("✅ Fichier .env chargé avec succès")
	}

	// Récupérer les valeurs de configuration
	baseURL := os.Getenv("API_BASE_URL")
	apiKey := os.Getenv("API_KEY")
	model := os.Getenv("MODEL_NAME")

	// Utiliser les valeurs par défaut si les variables d'environnement ne sont pas définies
	if baseURL == "" {
		baseURL = "https://inference.asicloud.cudos.org/v1"
	}
	if model == "" {
		model = "asi1-mini"
	}
	if apiKey == "" {
		apiKey = os.Getenv("API_KEY")
	}

	// Détecter les informations système
	systemInfo := detectSystemInfo()

	// Afficher les informations du système
	fmt.Printf("Système détecté: %s %s (%s)\n", systemInfo.OSName, systemInfo.OSVersion, systemInfo.OSID)
	fmt.Printf("Kernel: %s, Architecture: %s\n\n", systemInfo.KernelVersion, systemInfo.Architecture)

	// Créer l'agent
	agent := &Agent{
		APIConfig: struct {
			BaseURL string
			APIKey  string
			Model   string
		}{
			BaseURL: baseURL,
			APIKey:  apiKey,
			Model:   model,
		},
		SMTPConfig: struct {
			Host     string
			Port     int
			Username string
			Password string
			From     string
		}{
			Host:     os.Getenv("SMTP_HOST"),
			Port:     587, // Utiliser une valeur par défaut si non spécifié
			Username: os.Getenv("SMTP_USERNAME"),
			Password: os.Getenv("SMTP_PASSWORD"),
			From:     os.Getenv("SMTP_FROM"),
		},
		scanner: bufio.NewScanner(os.Stdin),
		messages: []types.Message{
			{
				Role: "system",
				Content: "Vous êtes un agent AI puissant qui aide l'utilisateur à accomplir ses tâches. " +
					"Répondez de manière concise et directe. Utilisez des listes à puces pour les étapes. " +
					"Si la tâche nécessite des commandes shell, ajoutez un bloc de code avec la commande à exécuter. " +
					"Pour les opérations sur le système de fichiers, fournissez les commandes appropriées. " +
					"Vous allez générer des commandes qui seront exécutées par l'agent.\n\n" +
					"Contexte système:\n" +
					fmt.Sprintf("  - Distribution: %s (ID: %s)\n", systemInfo.OSName, systemInfo.OSID) +
					fmt.Sprintf("  - Version: %s\n", systemInfo.OSVersion) +
					fmt.Sprintf("  - Kernel: %s\n", systemInfo.KernelVersion) +
					fmt.Sprintf("  - Architecture: %s\n", systemInfo.Architecture) +
					"Utilisez cette information pour adapter les commandes système en conséquence.",
			},
		},
		systemInfo: systemInfo,
	}

	// Récupérer la clé API de recherche
	searchAPIKey := os.Getenv("SEARCH_API_KEY")

	if searchAPIKey != "" {
		agent.webSearcher = search.NewWebSearcher(searchAPIKey, "google")
	} else {
		// Si aucune clé API n'est disponible, on ne peut pas faire de recherche
		agent.webSearcher = nil
		fmt.Println("Avertissement: Aucune clé API de recherche configurée ou échec du chargement du fichier .env. Les fonctionnalités de recherche seront limitées.")
	}

	// Configuration alternative avec DuckDuckGo si besoin
	// agent.webSearcher = search.NewWebSearcher("", "duckduckgo")

	// Créer le répertoire de configuration si nécessaire
	configDir := filepath.Join(os.Getenv("HOME"), ".cline")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		fmt.Printf("Avertissement: Impossible de créer le répertoire de configuration: %v\n", err)
	}

	// Initialiser la base de connaissances
	kbConfig := memory.DefaultConfig()
	if err := kbConfig.Validate(); err != nil {
		fmt.Printf("Avertissement: Configuration de la mémoire invalide: %v\n", err)
	} else {
		kb, err := memory.NewKnowledgeBase(kbConfig.StoragePath)
		if err != nil {
			fmt.Printf("Avertissement: Impossible d'initialiser la base de connaissances: %v\n", err)
		} else {
			agent.knowledgeBase = kb
			agent.knowledgeIntegrator = memory.NewKnowledgeIntegrator(kb)
		}
	}

	return agent
}

func (a *Agent) rememberInteraction(userInput, aiResponse string) {
	if a.knowledgeIntegrator != nil {
		a.knowledgeIntegrator.LearnFromInteraction(userInput, aiResponse)
	}
}

// Start démarre l'agent avec son terminal intégré
func (a *Agent) Start() {
	// Allouer les ressources nécessaires
	if a.apiClient == nil {
		a.apiClient = api.NewClient(a.APIConfig.BaseURL, a.APIConfig.APIKey, a.APIConfig.Model)
	}

	fmt.Println("┌─────────────────────────────────────────┐")
	fmt.Println("│         ASIONE Agent démarré            │")
	fmt.Println("│  (Tapez 'help' pour voir les commandes) │")
	fmt.Println("│              by MokhtarOne              │")
	fmt.Println("└─────────────────────────────────────────┘")
	fmt.Println()

	for {
		fmt.Print("ASI-agent> ")
		if !a.scanner.Scan() {
			break
		}

		input := strings.TrimSpace(a.scanner.Text())
		if input == "" {
			continue
		}

		// Gestion des commandes
		a.handleCommand(input)
	}
}

// sendMessage envoie une notification par email
func (a *Agent) sendMessage(to, subject, body string) error {
	// Vérifier si la configuration SMTP est complète
	if a.SMTPConfig.Host == "" || a.SMTPConfig.Username == "" || a.SMTPConfig.Password == "" {
		return fmt.Errorf("configuration SMTP incomplète")
	}

	// Créer le message
	message := fmt.Sprintf("From: %s\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"MIME-Version: 1.0\r\n"+
		"Content-Type: text/plain; charset=utf-8\r\n\r\n"+
		"%s",
		a.SMTPConfig.From,
		to,
		subject,
		body)

	// Créer une adresse d'envoi au format host:port
	addr := fmt.Sprintf("%s:%d", a.SMTPConfig.Host, a.SMTPConfig.Port)

	// Créer une authentification
	auth := smtp.PlainAuth("", a.SMTPConfig.Username, a.SMTPConfig.Password, a.SMTPConfig.Host)

	// Activer TLS ou SSL selon le port
	if a.SMTPConfig.Port == 465 {
		// Pour le port 465 (SSL), on utilise une connexion TLS directe
		return a.sendMailSSL(addr, auth, a.SMTPConfig.From, []string{to}, []byte(message))
	} else {
		// Pour les autres ports (587 avec STARTTLS)
		return smtp.SendMail(addr, auth, a.SMTPConfig.From, []string{to}, []byte(message))
	}
}

// sendMailSSL envoie un email via SMTP SSL (port 465)
func (a *Agent) sendMailSSL(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	conn, err := tls.Dial("tcp", addr, nil)
	if err != nil {
		return fmt.Errorf("erreur de connexion TLS: %v", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, "")
	if err != nil {
		return fmt.Errorf("erreur création client SMTP: %v", err)
	}
	defer client.Quit()

	if auth != nil {
		if ok, _ := client.Extension("AUTH"); ok {
			if err = client.Auth(auth); err != nil {
				return fmt.Errorf("erreur d'authentification: %v", err)
			}
		}
	}

	if err = client.Mail(from); err != nil {
		return fmt.Errorf("erreur MAIL FROM: %v", err)
	}
	for _, addr := range to {
		if err = client.Rcpt(addr); err != nil {
			return fmt.Errorf("erreur RCPT TO: %v", err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("erreur DATA: %v", err)
	}
	_, err = w.Write(msg)
	if err != nil {
		return fmt.Errorf("erreur écriture message: %v", err)
	}
	err = w.Close()
	if err != nil {
		return fmt.Errorf("erreur fermeture DATA: %v", err)
	}

	return client.Quit()
}

// handleEmailCommand gère la commande email utilisateur
func (a *Agent) handleEmailCommand(emailArgs string) {
	// L'argument est au format: to subject body
	// Utiliser les guillemets pour permettre les espaces dans le sujet et le corps
	parts := strings.Fields(emailArgs)
	if len(parts) < 3 {
		fmt.Println("\nUsage: email <destinataire> <sujet> <corps>\n")
		return
	}

	// Construire le sujet (peut contenir des espaces)
	to := parts[0]
	// Trouver la fin du sujet (peut être entre guillemets)
	subjectEnd := 2
	var subject, body string

	// Vérifier si le sujet est entre guillemets
	if len(parts) > 2 && strings.HasPrefix(parts[1], "\"") {
		// Chercher la fin du sujet entre guillemets
		for i := 2; i < len(parts); i++ {
			if strings.HasSuffix(parts[i], "\"") {
				subjectEnd = i
				break
			}
		}
		// Extraire le sujet, en enlevant les guillemets
		subjectParts := parts[1 : subjectEnd+1]
		subject = strings.Join(subjectParts, " ")
		subject = strings.Trim(subject, "\"")
		bodyStart := subjectEnd + 1
		if bodyStart < len(parts) {
			body = strings.Join(parts[bodyStart:], " ")
		}
	} else {
		// Pas de guillemets, sujet est un mot simple
		subject = parts[1]
		if len(parts) > 2 {
			body = strings.Join(parts[2:], " ")
		}
	}

	a.sendEmail(to, subject, body)
}

// sendEmail envoit un email
func (a *Agent) sendEmail(to, subject, body string) {
	// Vérifier si la configuration SMTP est complète
	if a.SMTPConfig.Host == "" || a.SMTPConfig.Username == "" || a.SMTPConfig.Password == "" {
		fmt.Printf("\n❌ Configuration SMTP incomplète. Veuillez configurer les variables d'environnement:\n")
		fmt.Printf("SMTP_HOST, SMTP_USERNAME, SMTP_PASSWORD\n\n")
		return
	}

	fmt.Printf("\n📧 Envoi d'email à %s...\n", to)

	err := a.sendMessage(to, subject, body)
	if err != nil {
		fmt.Printf("\n❌ Échec de l'envoi de l'email: %v\n\n", err)
	} else {
		fmt.Printf("\n✅ Email envoyé avec succès à %s\n\n", to)
	}
}

// handleCommand traite les commandes utilisateur
func (a *Agent) handleCommand(input string) {
	lowerInput := strings.ToLower(input)

	switch {
	case lowerInput == "help":
		a.showHelp()
	case lowerInput == "exit" || lowerInput == "quit":
		fmt.Println("Arrêt de ASIONE Agent...")
		os.Exit(0)
	case lowerInput == "config":
		a.showConfig()
	case lowerInput == "yes-to-all" || lowerInput == "oui à tout":
		a.enableAutoConfirm(true)
		fmt.Println("\n✅ Confirmation automatique activée. Toutes les commandes seront exécutées sans confirmation.\n")
	case lowerInput == "no-to-all" || lowerInput == "non à tout":
		a.enableAutoConfirm(false)
		fmt.Println("\n✅ Confirmation automatique désactivée. Toutes les commandes doivent être confirmées manuellement.\n")
	case strings.HasPrefix(lowerInput, "email "):
		a.handleEmailCommand(input[6:]) // "email" suivi par le reste de la commande
	case strings.HasPrefix(lowerInput, "set-api-key "):
		a.setAPIKey(input[12:])
	case strings.HasPrefix(lowerInput, "set-base-url "):
		a.setBaseURL(input[13:])
	case strings.HasPrefix(lowerInput, "set-model "):
		a.setModel(input[10:])
	default:
		a.processTask(input)
	}
}

// enableAutoConfirm active ou désactive la confirmation automatique
func (a *Agent) enableAutoConfirm(enabled bool) {
	if enabled {
		// Ajouter un message système pour que l'IA soit au courant
		a.messages = append(a.messages, types.Message{
			Role:    "system",
			Content: "L'utilisateur a activé le mode 'oui à tout'. Les commandes critiques doivent être exécutées sans confirmation.",
		})
	} else {
		// Informer que le mode a été désactivé
		a.messages = append(a.messages, types.Message{
			Role:    "system",
			Content: "Le mode 'oui à tout' a été désactivé. Toutes les commandes doivent être confirmées manuellement.",
		})
	}
}

// showMemoryStatus affiche l'état de la mémoire à long terme
func (a *Agent) showMemoryStatus() {
	if a.knowledgeBase == nil {
		fmt.Println("\nLa base de connaissances n'est pas disponible.\n")
		return
	}

	entries := a.knowledgeBase.GetAll()
	fmt.Printf("\nÉtat de la mémoire à long terme :\n")
	fmt.Printf("  Nombre d'entrées : %d\n", len(entries))
	fmt.Printf("  Taille estimée : %.1f KB\n", float64(len(entries)*512)/1024)
	fmt.Printf("  Première interaction : %s\n", entries[0].Timestamp.Format("2006-01-02"))
	fmt.Printf("  Dernière interaction : %s\n", entries[len(entries)-1].Timestamp.Format("2006-01-02"))
	fmt.Println()
}

// searchMemory recherche dans la mémoire à long terme
func (a *Agent) searchMemory(query string) {
	if a.knowledgeIntegrator == nil {
		fmt.Println("\nLa fonctionnalité de mémoire à long terme n'est pas disponible.\n")
		return
	}

	results := a.knowledgeIntegrator.SearchKnowledge(query)
	fmt.Println(a.knowledgeIntegrator.FormatKnowledgeResponse(results))
}

// rememberManual permet de mémoriser manuellement une information
func (a *Agent) rememberManual(content string) {
	if a.knowledgeIntegrator == nil {
		fmt.Println("\nLa fonctionnalité de mémoire à long terme n'est pas disponible.\n")
		return
	}

	metadata := map[string]string{
		"source":    "manual",
		"timestamp": time.Now().Format(time.RFC3339),
	}

	// Extraire des mots-clés de la description
	description := fmt.Sprintf("Information mémorisée manuellement : %s", content)
	keywords := a.extractKeywords(description)

	for _, keyword := range keywords {
		a.knowledgeIntegrator.Remember("manual", keyword, content, metadata)
	}

	fmt.Printf("\nInformation mémorisée avec les mots-clés : %v\n\n", keywords)
}

// extractKeywords extrait les mots-clés d'un texte
func (a *Agent) extractKeywords(text string) []string {
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

// showHelp affiche l'aide
func (a *Agent) showHelp() {
	fmt.Println("\nCommandes disponibles :")
	fmt.Println("  help                     - Affiche cette aide")
	fmt.Println("  exit/quit                - Quitte l'agent")
	fmt.Println("  config                   - Affiche la configuration")
	fmt.Println("  set-api-key <key>        - Définit la clé API")
	fmt.Println("  set-base-url <url>       - Définit l'URL de base du fournisseur")
	fmt.Println("  set-model <model>        - Définit le modèle à utiliser")
	fmt.Println("  <tâche>                  - Exécute une tâche (ex: coder, chercher, etc.)")
	fmt.Println()
}

// showConfig affiche la configuration actuelle
func (a *Agent) showConfig() {
	fmt.Printf("\nConfiguration actuelle :\n")
	fmt.Printf("  Base URL: %s\n", a.APIConfig.BaseURL)
	fmt.Printf("  API Key: %s\n", maskString(a.APIConfig.APIKey))
	fmt.Printf("  Model: %s\n", a.APIConfig.Model)
	fmt.Println()
}

// setAPIKey définit la clé API
func (a *Agent) setAPIKey(key string) {
	a.APIConfig.APIKey = strings.TrimSpace(key)
	a.apiClient.SetCredentials(a.APIConfig.BaseURL, a.APIConfig.APIKey, a.APIConfig.Model)
	fmt.Println("\nClé API définie avec succès\n")

	// Mettre à jour le moteur de recherche avec la clé si disponible
	if strings.Contains(a.APIConfig.BaseURL, "serpapi") {
		a.webSearcher = search.NewWebSearcher(a.APIConfig.APIKey, "google")
	}
}

// setBaseURL définit l'URL de base
func (a *Agent) setBaseURL(url string) {
	url = strings.TrimSpace(url)
	if url != "" {
		a.APIConfig.BaseURL = url
		a.apiClient.SetCredentials(a.APIConfig.BaseURL, a.APIConfig.APIKey, a.APIConfig.Model)
		fmt.Println("\nURL de base définie avec succès\n")

		// Mettre à jour le moteur de recherche si c'est un service de recherche
		if strings.Contains(url, "serpapi") || strings.Contains(url, "googleapis") {
			a.webSearcher = search.NewWebSearcher(a.APIConfig.APIKey, "google")
		}
	} else {
		fmt.Println("\nURL invalide\n")
	}
}

// setModel définit le modèle
func (a *Agent) setModel(model string) {
	model = strings.TrimSpace(model)
	if model != "" {
		a.APIConfig.Model = model
		a.apiClient.SetCredentials(a.APIConfig.BaseURL, a.APIConfig.APIKey, a.APIConfig.Model)
		fmt.Println("\nModèle défini avec succès\n")
	} else {
		fmt.Println("\nModèle invalide\n")
	}
}

// processTask traite une tâche utilisateur
func (a *Agent) processTask(task string) {
	// Vérifier si l'utilisateur demande une recherche
	if strings.Contains(strings.ToLower(task), "cherche") ||
		strings.Contains(strings.ToLower(task), "recherche") ||
		strings.Contains(strings.ToLower(task), "trouve") {
		a.performWebSearch(task)
		return
	}

	// Vérifier si l'utilisateur demande des informations récentes
	needsRecentInfo := strings.Contains(strings.ToLower(task), "récemment") ||
		strings.Contains(strings.ToLower(task), "dernières") ||
		strings.Contains(strings.ToLower(task), "2024") ||
		strings.Contains(strings.ToLower(task), "2025")

	// Si besoin d'informations récentes ou pas de clé API configurée, faire une recherche
	if needsRecentInfo || a.APIConfig.APIKey == "" {
		a.performWebSearch(task)
		return
	}

	// Sinon, traiter la tâche avec le modèle AI
	a.processWithAI(task)
}

// performWebSearch effectue une recherche sur Internet
func (a *Agent) performWebSearch(task string) {
	fmt.Printf("\nRecherche sur Internet pour: %s\n", task)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Extraire le terme de recherche de la tâche
	query := a.extractSearchQuery(task)

	results, err := a.webSearcher.Search(ctx, query)
	if err != nil {
		fmt.Printf("Erreur lors de la recherche: %v\n\n", err)
		return
	}

	// Afficher les résultats
	fmt.Println(search.FormatSearchResults(results))

	// Demander confirmation pour utiliser ces résultats
	fmt.Println("Voulez-vous utiliser ces informations pour compléter votre tâche ? (oui/non) [ENTRÉE pour 'oui']")
	fmt.Print(">")
	if !a.scanner.Scan() {
		return
	}

	input := a.scanner.Text()
	response := strings.ToLower(strings.TrimSpace(input))

	// Par défaut, saisie vide = "oui"
	if response == "" {
		response = "oui"
		fmt.Println("✅ (confirmation par défaut)")
	}

	if response == "oui" || response == "yes" || response == "y" {
		// Utiliser les résultats pour compléter la tâche
		a.processWithAIBasedOnSearch(task, search.FormatSearchResults(results))
	}
}

// extractSearchQuery extrait le terme de recherche de la tâche
func (a *Agent) extractSearchQuery(task string) string {
	lowerTask := strings.ToLower(task)

	// Supprimer les mots-clés de recherche communs
	terms := []string{"cherche", "recherche", "trouve", "informations sur", "dernières nouvelles sur", "qu'est-ce que"}

	result := task
	for _, term := range terms {
		if strings.Contains(lowerTask, term) {
			result = strings.ReplaceAll(result, term, "")
		}
	}

	return strings.TrimSpace(result)
}

// executeCommand exécute une commande shell, une par une, après confirmation et explication
// Affiche et conserve les logs d'exécution pour le débogage
// Relaie l'entrée standard pour permettre la saisie du mot de passe sudo
func (a *Agent) executeCommand(cmd string) error {
	// Résumer l'intention de la commande
	fmt.Printf("\nJe m'apprête à exécuter la commande suivante :\n")
	fmt.Printf("$ %s\n", cmd)
	fmt.Printf("Voulez-vous que je l'exécute ? (oui/non) [ENTRÉE pour 'oui'] ")

	// Attendre la confirmation de l'utilisateur
	if !a.scanner.Scan() {
		return fmt.Errorf("lecture de l'entrée utilisateur interrompue")
	}
	input := a.scanner.Text()
	response := strings.ToLower(strings.TrimSpace(input))

	// Par défaut, saisie vide = "oui"
	if response == "" {
		response = "oui"
		fmt.Println("✅ (confirmation par défaut)")
	}

	if response != "oui" && response != "yes" && response != "y" {
		return fmt.Errorf("exécution de la commande annulée par l'utilisateur")
	}

	// Vérifier si la commande nécessite des privilèges élevés
	needsSudo := strings.Contains(strings.ToLower(cmd), "sudo") || strings.Contains(strings.ToLower(cmd), "/etc/") || strings.Contains(strings.ToLower(cmd), "apt") || strings.Contains(strings.ToLower(cmd), "yum") || strings.Contains(strings.ToLower(cmd), "systemctl")

	var command *exec.Cmd
	if needsSudo {
		fmt.Println("\n⚠️  Cette commande nécessite des privilèges administrateur (sudo).")
		fmt.Print("Confirmer l'exécution avec sudo ? (oui/non) [ENTRÉE pour 'oui'] ")
		if !a.scanner.Scan() {
			return fmt.Errorf("lecture de l'entrée utilisateur interrompue")
		}
		input := a.scanner.Text()
		response := strings.ToLower(strings.TrimSpace(input))

		// Par défaut, saisie vide = "oui"
		if response == "" {
			response = "oui"
			fmt.Println("✅ (confirmation par défaut)")
		}

		if response != "oui" && response != "yes" && response != "y" {
			return fmt.Errorf("exécution de la commande annulée par l'utilisateur")
		}

		// Exécuter avec sudo -S pour permettre la saisie du mot de passe
		cmd = strings.ReplaceAll(cmd, "sudo ", "")
		fmt.Printf("🔐 Exécution avec sudo -S : %s\n", cmd)
		command = exec.Command("sudo", "-S", "sh", "-c", cmd)

		// Relier l'entrée/sortie du terminal au processus
		command.Stdin = os.Stdin
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
	} else {
		fmt.Printf("➡️  Exécution : %s\n", cmd)
		command = exec.Command("sh", "-c", cmd)
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
		command.Stdin = os.Stdin
	}

	// Exécuter la commande (sans capturer la sortie, car affichée en direct)
	err := command.Run()
	if err != nil {
		fmt.Printf("❌ La commande a échoué : %v\n", err)
		return err
	}

	fmt.Printf("✅ Commande exécutée avec succès.\n")
	return nil
}

// extractCommandFromResponse extrait la commande d'un bloc de code dans la réponse
func (a *Agent) extractCommandFromResponse(response string) (string, bool) {
	// Rechercher un bloc de code dans les ``` ```
	start := strings.Index(response, "```bash")
	if start == -1 {
		start = strings.Index(response, "```sh")
	}
	if start == -1 {
		start = strings.Index(response, "```")
	}

	if start != -1 {
		start += len("```")
		// Extraire le contenu du bloc de code
		end := strings.Index(response[start:], "```")
		if end != -1 {
			codeBlock := strings.TrimSpace(response[start : start+end])
			// Enlever "bash" ou "sh" du début si présent
			codeBlock = strings.TrimPrefix(codeBlock, "bash")
			codeBlock = strings.TrimPrefix(codeBlock, "sh")
			codeBlock = strings.TrimSpace(codeBlock)
			return codeBlock, true
		}
	}

	// Vérifier si la réponse contient une commande directement (sans bloc de code)
	// Chercher les séquences de commande après "```" ou "commande :"
	lowerResp := strings.ToLower(response)
	if strings.Contains(lowerResp, "commande :") {
		start = strings.Index(lowerResp, "commande :") + len("commande :")
		// Chercher la ligne suivante ou la fin de la ligne
		end := strings.Index(response[start:], "\n")
		if end == -1 {
			end = len(response) - start
		} else {
			end = end + strings.Index(response, response[start:]) + start - start
		}
		cmd := strings.TrimSpace(response[start:end])
		if cmd != "" {
			return cmd, true
		}
	}

	// Chercher les blocs de commandes après "voici la commande" ou "vous pouvez utiliser"
	indicators := []string{"voici la commande", "vous pouvez utiliser", "utilise cette commande", "commande pour", "exécute cette commande"}
	for _, indicator := range indicators {
		if idx := strings.Index(lowerResp, indicator); idx != -1 {
			// Chercher le début du bloc de commande
			cmdStart := idx + len(indicator)
			// Chercher un bloc de code ou une commande en ligne
			if nextCode := strings.Index(response[cmdStart:], "```"); nextCode != -1 {
				cmdStart += nextCode + 3
				if endCode := strings.Index(response[cmdStart:], "```"); endCode != -1 {
					cmd := strings.TrimSpace(response[cmdStart : cmdStart+endCode])
					if cmd != "" {
						return cmd, true
					}
				}
			}
		}
	}

	return "", false
}

// processWithAI traite une tâche avec le modèle d'intelligence artificielle
func (a *Agent) processWithAI(task string) {
	if a.APIConfig.APIKey == "" {
		fmt.Println("\nErreur: Clé API non configurée. Veuillez configurer votre clé API avec 'set-api-key'.\n")
		return
	}

	fmt.Printf("\n[AI] Analyse et exécution de la tâche...\n")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Ajouter le message utilisateur à l'historique
	a.messages = append(a.messages, types.Message{
		Role:    "user",
		Content: task,
	})

	// Appeler l'API avec tout l'historique des messages
	resp, err := a.apiClient.ChatCompletion(ctx, a.messages)
	if err != nil {
		// Vérifier si c'est une erreur 503 (Service Unavailable)
		if strings.Contains(err.Error(), "503") || strings.Contains(err.Error(), "engine_overloaded") {
			fmt.Printf("\nLe service d'IA est temporairement surchargé. Tentative de récupération avec recherche...\n")
			// Forcer l'utilisation de la recherche Internet en cas de panne du service IA
			a.performWebSearch(task)
			return
		}
		fmt.Printf("\nErreur lors de l'appel API: %v\n", err)
		return
	}

	// Ajouter la réponse de l'IA à l'historique
	a.messages = append(a.messages, types.Message{
		Role:    "assistant",
		Content: resp.Choices[0].Message.Content,
	})

	// Enregistrer l'interaction dans la mémoire à long terme
	a.rememberInteraction(task, resp.Choices[0].Message.Content)

	// Afficher la réponse
	fmt.Printf("\n%s\n\n", resp.Choices[0].Message.Content)

	// Afficher la réponse
	fmt.Printf("\n%s\n\n", resp.Choices[0].Message.Content)

	// Extraire et exécuter la commande si présente
	if cmd, found := a.extractCommandFromResponse(resp.Choices[0].Message.Content); found {
		err := a.executeCommand(cmd)
		if err != nil {
			fmt.Printf("\nErreur lors de l'exécution de la commande: %v\n", err)
		} else {
			fmt.Printf("\nCommande exécutée avec succès.\n\n")
		}
	}
}

// processWithAIBasedOnSearch traite une tâche avec le modèle d'intelligence artificielle en utilisant les résultats de recherche
func (a *Agent) processWithAIBasedOnSearch(task, searchResults string) {
	if a.APIConfig.APIKey == "" {
		fmt.Println("\nErreur: Clé API non configurée. Veuillez configurer votre clé API avec 'set-api-key'.\n")
		return
	}

	fmt.Printf("\n[AI] Analyse des résultats de recherche et exécution de la tâche...\n")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Ajouter le message utilisateur à l'historique
	a.messages = append(a.messages, types.Message{
		Role:    "user",
		Content: fmt.Sprintf("Tâche: %s\n\nRésultats de recherche:\n%s", task, searchResults),
	})

	// Appeler l'API avec tout l'historique des messages
	resp, err := a.apiClient.ChatCompletion(ctx, a.messages)
	if err != nil {
		fmt.Printf("\nErreur lors de l'appel API: %v\n", err)
		return
	}

	// Afficher la réponse
	fmt.Printf("\n%s\n\n", resp.Choices[0].Message.Content)

	// Ajouter la réponse de l'IA à l'historique
	a.messages = append(a.messages, types.Message{
		Role:    "assistant",
		Content: resp.Choices[0].Message.Content,
	})

	// Extraire et exécuter la commande si présente
	if cmd, found := a.extractCommandFromResponse(resp.Choices[0].Message.Content); found {
		err := a.executeCommand(cmd)
		if err != nil {
			fmt.Printf("\nErreur lors de l'exécution de la commande: %v\n\n", err)
		} else {
			fmt.Printf("\nCommande exécutée avec succès.\n\n")
		}
	}
}

// maskString masque une partie d'une chaîne (pour les clés API)
func maskString(s string) string {
	if len(s) <= 8 {
		return "********"
	}
	return s[:4] + "..." + s[len(s)-4:]
}

func main() {
	agent := NewAgent()
	agent.Start()
}
