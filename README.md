# Les bons comptes — Gin (Go)

Application web de gestion de dépenses partagées, développée dans le cadre du module
**Infrastructures & Frameworks** (M2 MIAGE IFI, Université de Lille).

Technologie attribuée : **Gin**, framework web minimaliste en **Go**.

## Équipe

- Sofia Demchuk
- Yuliia Los
- Baptiste Delisle
- Thomas DEVINCK

Encadrants : Romain Rouvoy, Gaëtan Hasbroucq

## Prérequis

| Outil                   | Version | Vérifier                 |
|-------------------------|---------|--------------------------|
| Go                      | ≥ 1.22  | `go version`             |
| Git                     | —       | `git --version`          |
| Docker + Docker Compose | —       | `docker compose version` |

### Installer Go

**macOS**

    brew install go

ou l'installeur `.pkg` depuis https://go.dev/dl (choisir arm64 pour les Mac Apple Silicon).

**Linux** — télécharger l'archive sur https://go.dev/dl puis la décompresser dans
`/usr/local` et ajouter `/usr/local/go/bin` au `PATH`.

**Windows** — installeur `.msi` depuis https://go.dev/dl, puis rouvrir le terminal.

### Installer Docker

**macOS**

    brew install --cask docker

ou Docker Desktop depuis https://www.docker.com/products/docker-desktop.
Lancer l'application : le démon ne tourne que si Docker Desktop est ouvert.

**Linux** — Docker Engine et le plugin Compose en suivant
https://docs.docker.com/engine/install. Ajouter son utilisateur au groupe
`docker` pour éviter `sudo`.

**Windows** — Docker Desktop avec le backend WSL 2.

### Configurer l'éditeur

Activer le formatage automatique à la sauvegarde : Go impose `gofmt`, et la CI
rejette tout fichier non formaté.

- **IntelliJ IDEA / GoLand** — installer le plugin **Go**, puis
  Settings → Tools → Actions on Save → cocher *Reformat code*
- **VS Code** — extension **Go** officielle, le formatage est actif par défaut

Vérifier à tout moment avec :

    gofmt -l .

La commande ne doit rien afficher.

## Démarrage

### 1. Cloner et installer les dépendances

    git clone git@github.com:miage-if-gin/bons-comptes-gin.git
    cd bons-comptes-gin
    go mod download

### 2. Configurer l'environnement

    cp .env.example .env

Puis générer un secret de session avec `openssl rand -hex 32` et le reporter
dans `.env`.

> `.env` est ignoré par Git. Chacun génère son propre `SESSION_SECRET` en
> local ; aucun secret ne doit être commité ni partagé dans le dépôt.

| Variable         | Rôle                                   | Défaut        |
|------------------|----------------------------------------|---------------|
| `APP_ENV`        | `development` ou `production`          | `development` |
| `PORT`           | port d'écoute HTTP                     | `8080`        |
| `DATABASE_URL`   | chaîne de connexion PostgreSQL         | base locale   |
| `SESSION_SECRET` | clé de signature des sessions (64 hex) | —             |

### 3. Lancer la base de données

Docker Desktop doit être démarré.

    make up              # démarre PostgreSQL et Adminer
    docker compose ps    # postgres doit être "healthy"

| Service    | URL                   | Identifiants                           |
|------------|-----------------------|----------------------------------------|
| PostgreSQL | `localhost:5432`      | `bonscomptes` / `bonscomptes`          |
| Adminer    | http://localhost:8081 | serveur `postgres`, base `bonscomptes` |

### 4. Lancer le serveur

Les migrations s'appliquent automatiquement au démarrage.

    make run
    curl localhost:8080/readyz    # {"db":"ok","status":"ok"}

L'application est disponible sur http://localhost:8080.

### Commandes utiles

    make help        # liste toutes les commandes
    make psql        # shell SQL sur la base locale
    make fmt         # formate le code
    make vet         # analyse statique
    make test        # tests avec détection de data races
    make down        # arrête les conteneurs
    make reset       # détruit la base et ses données

### Notes Windows

`make` et `openssl` ne sont pas disponibles nativement. Le plus simple est de
travailler dans **WSL 2** (Ubuntu), où toutes les commandes ci-dessus
fonctionnent telles quelles.

Sans WSL, remplacer les cibles `make` par leurs commandes :

| Cible       | Commande                     |
|-------------|------------------------------|
| `make up`   | `docker compose up -d`       |
| `make down` | `docker compose down`        |
| `make run`  | `go run ./cmd/server`        |
| `make test` | `go test ./... -race -cover` |

Attention : le `Makefile` charge le `.env` automatiquement. Sans `make`, il faut
exporter les variables manuellement avant de lancer le serveur.

Générer le secret avec PowerShell plutôt qu'`openssl` :

    -join ((1..32) | ForEach-Object { '{0:x2}' -f (Get-Random -Max 256) })

## Structure du projet

    cmd/server/        point d'entrée : configuration, base, serveur, arrêt propre
    internal/
      config/          chargement de la configuration depuis l'environnement
      db/              connexion PostgreSQL et application des migrations
      http/            routeur et handlers
      web/             chargement et rendu des templates
    migrations/        schéma SQL versionné (embarqué dans le binaire)
    web/
      templates/       layouts et pages HTML
      static/          fichiers statiques

## Choix techniques

| Besoin         | Choix                     | Pourquoi                                                       |
|----------------|---------------------------|----------------------------------------------------------------|
| Framework HTTP | Gin                       | imposé par le sujet                                            |
| Accès données  | GORM                      | ORM le plus répandu en Go, rapide à prendre en main            |
| Migrations     | golang-migrate            | fichiers SQL versionnés, embarqués via `go:embed`              |
| Vues           | `html/template` + htmx    | pas de build frontend, rendu côté serveur                      |
| CSS            | Tailwind (CDN)            | aucune installation ; un build serait nécessaire en production |
| Configuration  | variables d'environnement | Twelve-Factor, principe III                                    |

Les montants sont stockés en **centimes** (`BIGINT`) : jamais de flottant pour
représenter de l'argent.

## Supervision

| Endpoint   | Rôle                                           |
|------------|------------------------------------------------|
| `/healthz` | sonde de vivacité : le processus répond-il ?   |
| `/readyz`  | sonde de disponibilité : la base répond-elle ? |

Un orchestrateur redémarre l'instance sur l'échec de la première, la retire du
load balancer sur l'échec de la seconde.

## Contribuer

Voir [CONTRIBUTING.md](CONTRIBUTING.md).