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

| Outil | Version | Vérifier |
|---|---|---|
| Go | ≥ 1.22 | `go version` |
| Git | — | `git --version` |
| Docker + Docker Compose | — | `docker compose version` |

### Installer Go

**macOS**

    brew install go

ou l'installeur `.pkg` depuis https://go.dev/dl (choisir arm64 pour les Mac Apple Silicon).

**Linux** — télécharger l'archive sur https://go.dev/dl puis la décompresser dans
`/usr/local` et ajouter `/usr/local/go/bin` au `PATH`.

**Windows** — installeur `.msi` depuis https://go.dev/dl, puis rouvrir le terminal.

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

| Variable | Rôle | Défaut |
|---|---|---|
| `APP_ENV` | `development` ou `production` | `development` |
| `PORT` | port d'écoute HTTP | `8080` |
| `DATABASE_URL` | chaîne de connexion PostgreSQL | base locale |
| `SESSION_SECRET` | clé de signature des sessions (64 hex) | — |

### 3. Lancer la base de données

Docker Desktop doit être démarré.

    make up              # démarre PostgreSQL et Adminer
    docker compose ps    # postgres doit être "healthy"

| Service | URL | Identifiants |
|---|---|---|
| PostgreSQL | `localhost:5432` | `bonscomptes` / `bonscomptes` |
| Adminer | http://localhost:8081 | serveur `postgres`, base `bonscomptes` |

### 4. Lancer le serveur

Les migrations s'appliquent automatiquement au démarrage.

    make run
    curl localhost:8080/readyz    # {"db":"ok","status":"ok"}

### Commandes utiles

    make help        # liste toutes les commandes
    make psql        # shell SQL sur la base locale
    make test        # tests avec détection de data races
    make down        # arrête les conteneurs
    make reset       # détruit la base et ses données

## Structure du projet

    cmd/server/        point d'entrée de l'application
    internal/
      config/          chargement de la configuration
      db/              connexion PostgreSQL et migrations
    migrations/        schéma SQL versionné (embarqué dans le binaire)

## Contribuer

Voir [CONTRIBUTING.md](CONTRIBUTING.md).