# Conventions de travail

## Branches

`main` est protégée : aucun push direct, tout passe par une PR approuvée
par un autre membre.

| Préfixe | Usage |
|---|---|
| `feat/` | nouvelle fonctionnalité |
| `fix/` | correction de bug |
| `chore/` | outillage, config, dépendances |
| `docs/` | documentation |

## Commits

Format [Conventional Commits](https://www.conventionalcommits.org) :

```
feat(wallet): ajoute la création d'un porte-monnaie
fix(auth): corrige la validation du mot de passe
chore(ci): ajoute le workflow de lint
```
