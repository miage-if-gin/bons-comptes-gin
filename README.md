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
```bash
brew install go
```
ou l'installeur `.pkg` depuis https://go.dev/dl (choisir arm64 pour les Mac Apple Silicon).

**Linux** — télécharger l'archive sur https://go.dev/dl puis :
```bash
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.XX.X.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc && source ~/.bashrc
```

**Windows** — installeur `.msi` depuis https://go.dev/dl, puis rouvrir le terminal.

## Démarrage

### 1. Cloner et installer les dépendances

```bash
git clone git@github.com:miage-if-gin/bons-comptes-gin.git
cd bons-comptes-gin
go mod download
```

### 2. Configurer l'environnement

```bash
cp .env.example .env
```

Puis générer un secret de session et le reporter dans `.env` :

```bash
openssl rand -hex 32
```

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

```bash
docker compose up -d
docker compose ps        # postgres doit être "healthy"
```

| Service | URL | Identifiants |
|---|---|---|
| PostgreSQL | `localhost:5432` | `bonscomptes` / `bonscomptes` |
| Adminer | http://localhost:8081 | serveur `postgres`, base `bonscomptes` |

### 4. Lancer le serveur

```bash
go run ./cmd/server
curl localhost:8080/healthz    # {"status":"ok"}
```

### Arrêter

```bash
docker compose down      # arrête, conserve les données
docker compose down -v   # arrête et détruit les données
```

## Structure du projet

*...*

## Contribuer

Voir [CONTRIBUTING.md](CONTRIBUTING.md).