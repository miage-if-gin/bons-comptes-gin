# Les bons comptes

Application de gestion de dépenses partagées : on crée un porte-monnaie, on y
invite des comptes, chacun déclare ses dépenses, et à la clôture l'application
calcule qui doit combien à qui.

Projet du module **Infrastructures & Frameworks** (M2 MIAGE IFI, Université de
Lille). Technologie attribuée : **Gin**, framework web minimaliste en **Go**.

**Équipe** — Sofia Demchuk, Yuliia Los, Baptiste Delisle, Thomas Devinck
**Encadrants** — Romain Rouvoy, Gaëtan Hasbroucq

---

## Démarrer

| Prérequis                                               | Version | Vérifier                 |
|---------------------------------------------------------|---------|--------------------------|
| [Go](https://go.dev/dl)                                 | ≥ 1.27  | `go version`             |
| [Docker + Compose](https://docs.docker.com/get-started) | —       | `docker compose version` |

```bash
git clone git@github.com:miage-if-gin/bons-comptes-gin.git
cd bons-comptes-gin
go mod download

cp .env.example .env
openssl rand -hex 32          # à reporter dans SESSION_SECRET

make up                       # PostgreSQL + Adminer
make run                      # http://localhost:8080
```

Les migrations s'appliquent au démarrage du serveur : la base est prête sans
commande supplémentaire.

Vérifier : `curl localhost:8080/readyz` → `{"db":"ok","status":"ok"}`

| Service    | URL                   | Identifiants                           |
|------------|-----------------------|----------------------------------------|
| PostgreSQL | `localhost:5432`      | `bonscomptes` / `bonscomptes`          |
| Adminer    | http://localhost:8081 | serveur `postgres`, base `bonscomptes` |

### Sur Windows

`make` et `openssl` ne sont pas disponibles nativement. Le plus simple est de
travailler dans **WSL 2**, où toutes les commandes de ce README fonctionnent
telles quelles.

Sans WSL, remplacer les cibles `make` par leurs commandes, et exporter les
variables du `.env` manuellement — c'est le `Makefile` qui les charge :

| Cible       | Commande                     |
|-------------|------------------------------|
| `make up`   | `docker compose up -d`       |
| `make run`  | `go run ./cmd/server`        |
| `make seed` | `go run ./cmd/seed`          |
| `make test` | `go test ./... -race -cover` |
| `make down` | `docker compose down`        |

Générer le secret de session avec PowerShell :

```powershell
-join ((1..32) | ForEach-Object { '{0:x2}' -f (Get-Random -Max 256) })
```

### Jeu de données

Partir d'une base réaliste plutôt que vide :

```bash
make seed                     # 40 comptes, 25 porte-monnaies, 5 000 dépenses
```

Tous les comptes générés ont le mot de passe `demo1234`. Options disponibles
via `go run ./cmd/seed -h`.

### Configuration

Toute la configuration passe par l'environnement (Twelve-Factor, principe III).
Aucune valeur n'est codée en dur ailleurs dans le code.

| Variable         | Rôle                                   | Défaut        |
|------------------|----------------------------------------|---------------|
| `APP_ENV`        | `development` ou `production`          | `development` |
| `PORT`           | port d'écoute HTTP                     | `8080`        |
| `DATABASE_URL`   | chaîne de connexion PostgreSQL         | base locale   |
| `SESSION_SECRET` | clé de signature des sessions (64 hex) | —             |

`.env` est ignoré par Git : aucun secret n'est versionné, chacun génère le sien.

### Commandes

```
make help     liste toutes les commandes
make up       démarre PostgreSQL et Adminer
make run      lance le serveur
make seed     peuple la base de démonstration
make test     tests avec détection de data races
make fmt      formate le code
make vet      analyse statique
make psql     shell SQL sur la base locale
make down     arrête les conteneurs
make reset    détruit la base et ses données
```

### Base partagée

L'équipe travaille sur une instance PostgreSQL commune, hébergée sur une
machine virtuelle de l'OpenStack du FIL ([ADR-009](docs/adr.md)). Remplacer
`DATABASE_URL` par l'URL du groupe, transmise en privé.

Accessible depuis le campus, ou via le VPN du FIL. La liaison est chiffrée
(TLS 1.3, `sslmode=require`). Un verrou consultatif PostgreSQL empêche deux
instances de rejouer la même migration simultanément.

---

## Architecture

```
cmd/server/     point d'entrée : configuration, base, serveur, arrêt propre
cmd/seed/       génération du jeu de données de démonstration
internal/
  config/       configuration lue depuis l'environnement
  db/           connexion PostgreSQL et migrations
  domain/       structures métier, sans dépendance technique
  repository/   accès aux données
  service/      règles métier, testables sans serveur ni base
  http/         routeur, handlers, middlewares
  web/          chargement et rendu des templates
migrations/     schéma SQL versionné, embarqué dans le binaire
web/            templates HTML et fichiers statiques
docs/adr.md     décisions d'architecture
```

Gin n'impose ni ORM, ni migrations, ni authentification, ni structure de
projet. Chacun de ces choix est donc le nôtre :

| Besoin        | Choix                  | Pourquoi                                                   |
|---------------|------------------------|------------------------------------------------------------|
| Accès données | GORM                   | répandu, aucune installation pour les participants du TP   |
| Migrations    | golang-migrate         | SQL versionné, embarqué via `go:embed`                     |
| Vues          | `html/template` + htmx | pas de build frontend, échappement XSS natif               |
| CSS           | Tailwind (CDN)         | aucune installation ; un build serait requis en production |
| Sessions      | table `sessions`       | révocation réelle à la déconnexion, contrairement au JWT   |
| Montants      | `BIGINT` en centimes   | jamais de flottant pour représenter de l'argent            |

Le raisonnement complet, les options écartées et leurs conséquences sont dans
**[docs/adr.md](docs/adr.md)** — douze décisions documentées.

---

## Déploiement

L'application tourne sur la VM du FIL, dans un conteneur, à côté de sa base.
L'image fait **13 Mo** : binaire statique, templates et schéma SQL compris.

```bash
docker build -t bons-comptes .
docker save bons-comptes:latest | gzip > /tmp/bons-comptes.tar.gz
scp -i ~/.ssh/bons-comptes.pem /tmp/bons-comptes.tar.gz ubuntu@<vm>:~/

ssh -i ~/.ssh/bons-comptes.pem ubuntu@<vm>
docker load < ~/bons-comptes.tar.gz
cd ~/bons-comptes && docker compose up -d
```

L'image est construite pour `linux/amd64` : un binaire Go est lié à une
architecture, et la VM est en x86. Docker Desktop ne traversant pas le VPN du
FIL, construire VPN déconnecté puis le reconnecter pour le transfert.

En production, `APP_ENV=production` fait passer Gin en mode release et réduit
la verbosité des logs SQL. Le `SESSION_SECRET` doit y rester stable : le
changer invalide toutes les sessions.

### Supervision

| Endpoint   | Question à laquelle il répond    | Réaction attendue en cas d'échec                 |
|------------|----------------------------------|--------------------------------------------------|
| `/healthz` | le processus est-il vivant ?     | redémarrer l'instance                            |
| `/readyz`  | la base répond-elle ?            | retirer l'instance du trafic, sans la redémarrer |
| `/metrics` | comment se porte l'application ? | —                                                |

`/metrics` expose au format Prometheus le nombre de requêtes par route,
méthode et code de retour, ainsi qu'un histogramme des temps de réponse. Les
routes sont étiquetées par leur motif (`/wallets/:id`) et non par l'URL
réelle : sans cela, chaque porte-monnaie créerait sa propre série.

---

## Contribuer

Conventions de branches, de commits et d'architecture :
**[CONTRIBUTING.md](CONTRIBUTING.md)**.

Chaque modification passe par une pull request relue par un autre membre. La
CI vérifie le formatage, l'analyse statique, la compilation et les tests.