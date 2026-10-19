.PHONY: help setup up down reset run psql fmt vet test tidy

# Charge le .env et exporte ses variables vers les commandes ci-dessous.
ifneq (,$(wildcard .env))
include .env
export
endif

help: ## Affiche cette aide
	@grep -hE '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-8s\033[0m %s\n", $$1, $$2}'

setup: ## Première installation
	@test -f .env || cp .env.example .env
	go mod download
	@echo "Renseigne SESSION_SECRET dans .env : openssl rand -hex 32"

up: ## Démarre PostgreSQL et Adminer
	docker compose up -d

down: ## Arrête les conteneurs
	docker compose down

reset: ## Détruit la base locale et ses données
	docker compose down -v

run: ## Lance le serveur
	go run ./cmd/server

psql: ## Ouvre un shell SQL sur la base locale
	docker compose exec postgres psql -U bonscomptes -d bonscomptes

fmt: ## Formate le code
	go fmt ./...

vet: ## Analyse statique
	go vet ./...

test: ## Lance les tests
	go test ./... -race -cover

tidy: ## Nettoie go.mod et go.sum
	go mod tidy