# ASIONE Agent

Un agent AI puissant qui aide à accomplir des tâches grâce à l'intégration d'intelligence artificielle, de recherche Internet et de mémoire à long terme.

##Ce code est un FORK de Cline

## Fonctionnalités principales

- **Assistant intelligent** : Posez des questions, demandez des informations ou exécutez des tâches
- **Recherche Internet intégrée** : Accès aux informations récentes via des moteurs de recherche
- **Mémoire à long terme** : L'agent apprend de vos interactions et se souvient des informations importantes
- **Terminal intégré** : Exécution de commandes système avec confirmation de sécurité
- **Multimodal** : Support des entrées textuelles complexes et des tâches avancées
- **Mail fonction** : unstable
<img width="1606" height="1199" alt="Copie d&#39;écran_20251215_212739" src="https://github.com/user-attachments/assets/b9d32476-cf84-4862-aa49-6f77f3f787e2" />
<img width="1606" height="1199" alt="Copie d&#39;écran_20251215_212627" src="https://github.com/user-attachments/assets/ab2fa97c-4fb4-4929-86a7-2f11e4f45def" />
<img width="1606" height="1199" alt="Copie d&#39;écran_20251215_212641" src="https://github.com/user-attachments/assets/87db505d-e29a-4c2e-8561-5c6500cd5282" />
<img width="1606" height="1199" alt="Copie d&#39;écran_20251215_212649" src="https://github.com/user-attachments/assets/098772d7-fc87-44af-bc52-a7da65d110b7" />
<img width="1606" height="1199" alt="Copie d&#39;écran_20251215_212703" src="https://github.com/user-attachments/assets/1dcd860d-5b57-4b50-abb5-a7156a77d7ec" />

## Configuration des API

L'agent nécessite la configuration d'une API pour fonctionner correctement. Voici comment configurer les clés d'API :

### 1. Création du fichier de configuration

Créez un fichier `.env` dans le répertoire racine du projet :

```bash
touch .env
```

### 2. Configuration des variables d'environnement

Ajoutez les variables suivantes au fichier `.env` :

```env
# Configuration du fournisseur API OpenAI compatible
API_BASE_URL=https://inference.asicloud.cudos.org/v1
API_KEY=api
MODEL_NAME=asi1-mini
MAX_TOKENS=16384

# Configuration de l'API de recherche SerpAPI
SEARCH_API_KEY=api

# Configuration SMTP pour l'envoi d'emails
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=
SMTP_PASSWORD=
SMTP_FROM="ASIONE Agent" <.com>

# Options d'export des reponses
EXPORT_FORMAT=markdown
EXPORT_DIRECTORY=./exports
PAGINATION_ENABLED=true
PAGINATION_SIZE=20

```
Modifiez le chemin du dossier dans main.go qui correspond au chemin de la directory.
absPath := "/home/arch/Desktop/asione-agent/.env"
### 3. Clés API requises

- **API_BASE_URL** : L'URL du fournisseur de services d'IA (par défaut: ASI Cloud)
- **API_KEY** : Votre clé d'authentification pour le service d'IA
- **MODEL_NAME** : Le modèle d'IA à utiliser (par défaut: asi1-mini)
- **SEARCH_API_KEY** : La clé API pour les recherches Internet (optionnel)

> **Remarque** : Remplacez `votre_clé_api_ici` et `votre_clé_api_recherche_ici` par vos clés API réelles. Ne partagez jamais vos clés API publiques.

## Guide d'installation

1. Clonez le dépôt :
   ```bash
   git clone https://github.com/ai-terminal-assistant/ai-terminal-assistant.git
   ```

2. Installez Go 1.16 ou supérieur

3. Installez les dépendances :
   ```bash
   go mod tidy
   ```

4. Configurez les variables d'environnement (voir section précédente)

5. Exécutez l'agent :
   ```bash
   go run main.go
   ```

## Commandes disponibles

- `help` - Affiche cette aide
- `exit` / `quit` - Quitte l'agent
- `config` - Affiche la configuration actuelle
- `set-api-key <key>` - Définit la clé API
- `set-base-url <url>` - Définit l'URL de base
- `set-model <model>` - Définit le modèle à utiliser
- `yes-to-all` - Active la confirmation automatique
- `no-to-all` - Désactive la confirmation automatique

## Journal des modifications (Changelog)


### 2025-10-15 - Ajout de la mémoire à long terme
- Implémentation d'une base de connaissances persistante
- Ajout de la capacité d'apprentissage des interactions
- Recherche dans la mémoire personnelle
- Commandes `memory` et `remember`

### 2025-08-20 - Recherche Internet améliorée
- Intégration de SerpAPI pour les résultats de recherche
- Support de DuckDuckGo comme alternative
- Prétraitement des résultats pour une meilleure compréhension
- Confirmation utilisateur avant utilisation des résultats

### 2025-06-10 - Version initiale
- Déploiement de la version initiale avec interface en ligne de commande
- Intégration de base avec l'API d'inférence ASI
- Prise en charge des commandes système avec confirmation
- Interface utilisateur interactive

## Conditions de sécurité

- **Confidentialité** : Aucune donnée personnelle n'est stockée sans permission explicite
- **Confirmation des commandes critiques** : Toutes les commandes système nécessitant des privilèges sont confirmées
- **Gestion des erreurs** : Comportement robuste en cas d'erreurs de réseau ou d'API
- **Journalisation** : Toutes les interactions sont enregistrées localement pour audit

## Dépendances

- Go 1.16+
- github.com/joho/godotenv v1.5.1
- Accès Internet pour les appels API
- (Optionnel) Clé API pour l'accès avancé

## Support

Pour toute question ou problème, veuillez ouvrir un ticket sur le dépôt GitHub.
