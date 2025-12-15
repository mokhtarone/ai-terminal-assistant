#!/bin/bash
# Script de lancement pour Cline Agent

# Ajouter le module Go au GOPATH
export GO111MODULE=on

# Naviguer vers le répertoire de l'agent
cd "$(dirname "$0")" || exit

# Exécuter l'agent
go run main.go
