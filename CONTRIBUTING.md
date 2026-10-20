# Conventions de travail

## Branches

`main` est protégée : aucun push direct, tout passe par une PR approuvée
par un autre membre.

| Préfixe  | Usage                          |
|----------|--------------------------------|
| `feat/`  | nouvelle fonctionnalité        |
| `fix/`   | correction de bug              |
| `chore/` | outillage, config, dépendances |
| `docs/`  | documentation                  |

## Commits

Format [Conventional Commits](https://www.conventionalcommits.org) :

```
feat(wallet): ajoute la création d'un porte-monnaie
fix(auth): corrige la validation du mot de passe
chore(ci): ajoute le workflow de lint
```

## Code Go

Formatage automatique obligatoire : `gofmt` doit être appliqué à la sauvegarde.
La CI rejette tout fichier non formaté. Vérifier avec `gofmt -l .` (sortie vide).

Nommage des fichiers : minuscules, underscores si nécessaire, jamais de
camelCase. Le suffixe `_test.go` est réservé aux tests.

Le nom du package porte déjà le contexte : dans `repository`, le fichier est
`account.go` et non `account_repository.go`, puisque le type s'utilise sous la
forme `repository.AccountRepository`. Même logique que `net/http/server.go`
dans la bibliothèque standard.

Exception quand un package regroupe plusieurs rôles : dans `internal/http`,
les fichiers sont préfixés par leur rôle (`handlers_auth.go`, `middleware.go`,
`router.go`).

## Architecture

Quatre couches, chacune ne connaissant que la suivante :

| Couche       | Rôle                  | Dépend de              |
|--------------|-----------------------|------------------------|
| `domain`     | structures métier     | rien                   |
| `repository` | accès base            | `domain`               |
| `service`    | règles métier         | `repository`, `domain` |
| `http`       | handlers, middlewares | `service`              |

Un handler n'écrit jamais de SQL. Une règle métier ne connaît ni Gin ni HTTP,
ce qui la rend testable sans serveur.

Chaque lot ajoute son fichier `handlers_xxx.go` dans `internal/http` avec une
fonction `registerXxx(r, d)` appelée depuis `NewRouter`. Les lots ne se
touchent donc pas.

Toute décision structurante est consignée dans [docs/adr.md](docs/adr.md).
