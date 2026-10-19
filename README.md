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

```bash
git clone git@github.com:miage-if-gin/bons-comptes-gin.git
cd bons-comptes-gin
go mod download
```

*...*

## Structure du projet

*...*

## Contribuer

Voir [CONTRIBUTING.md](CONTRIBUTING.md).