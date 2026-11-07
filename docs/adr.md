# Décisions d'architecture

Chaque entrée documente un choix structurant : le contexte, les options
écartées, la décision retenue et ses conséquences.

Gin n'impose presque rien — pas d'ORM, pas de migrations, pas
d'authentification, pas de structure de projet. Ces décisions sont donc toutes
les nôtres, contrairement à ce qu'aurait produit un framework « batteries
included » comme Django ou Rails.

---

## ADR-001 — GORM pour l'accès aux données

**Contexte.** Gin ne fournit aucune couche de persistance. Il faut choisir
entre un ORM, un générateur de code SQL et l'accès direct au driver.

**Options.**

| Option   | Avantages                                                                  | Inconvénients                                                         |
|----------|----------------------------------------------------------------------------|-----------------------------------------------------------------------|
| GORM     | répandu, rapide à prendre en main, peu de code                             | magie implicite, requêtes N+1 faciles, SQL généré parfois inefficace  |
| sqlc     | SQL écrit à la main, typage vérifié à la compilation, performances natives | outil externe à installer, régénération à chaque changement de schéma |
| sqlx     | fine couche, transparent                                                   | verbeux                                                               |
| pgx seul | le plus performant                                                         | le plus verbeux                                                       |

**Décision.** GORM, avec recours au SQL brut (`db.Raw`) sur les requêtes
d'agrégation de la clôture.

**Pourquoi.** Le critère décisif est pédagogique : nos camarades
redévelopperont une partie de l'application en deux heures sans connaître Go.
Avec sqlc, ils devraient installer un outil et comprendre la génération de
code ; une erreur de configuration leur coûterait la séance. GORM ne demande
qu'un `go mod download`.

**Conséquences.** Vigilance requise sur les requêtes N+1, particulièrement sur
le listing des porte-monnaies. Le mélange GORM / SQL brut nous donne les deux
approches dans un même projet, utile pour la comparaison.

---

## ADR-002 — Migrations SQL versionnées plutôt qu'AutoMigrate

**Contexte.** GORM propose `AutoMigrate`, qui dérive le schéma des structures
Go. Alternative : des fichiers SQL numérotés, appliqués par golang-migrate.

**Décision.** Fichiers SQL versionnés dans `migrations/`, embarqués dans le
binaire par `go:embed`, appliqués au démarrage.

**Pourquoi.** `AutoMigrate` ne sait ni supprimer une colonne, ni renommer, ni
exprimer une contrainte `CHECK`, et son comportement en production est
imprévisible. Les fichiers SQL donnent un schéma explicite, relisible en revue
de code, et réversible.

L'embarquement via `go:embed` a une conséquence forte : le binaire contient son
propre schéma. Aucun outil à installer sur les machines de l'équipe ni en
production, et il est impossible de déployer un binaire avec les mauvaises
migrations.

**Conséquences.** golang-migrate pose un verrou consultatif PostgreSQL pendant
l'opération : plusieurs instances peuvent démarrer simultanément sur la base
partagée sans rejouer la même migration. C'est ce qui rend tenable la
contrainte de backend partagé imposée par le sujet.

---

## ADR-003 — Sessions en base plutôt que JWT

**Contexte.** Il faut identifier l'utilisateur d'une requête à l'autre.

**Décision.** Table `sessions`, token opaque dans un cookie `HttpOnly`.

**Pourquoi.** La révocation. Se déconnecter supprime réellement la session. Un
JWT reste valide jusqu'à son expiration même après déconnexion, à moins de
maintenir une liste de révocation — qui annule son principal avantage, celui de
ne pas interroger la base.

**Conséquences.** Une requête en base par page chargée, acceptable à notre
échelle et servie par un index sur la clé primaire. Il faut purger
périodiquement les sessions expirées, sinon la table grossit indéfiniment.

---

## ADR-004 — Montants en centimes, jamais en flottant

**Contexte.** Représenter des sommes d'argent.

**Décision.** `BIGINT` en centimes. 12,50 € est stocké `1250`.

**Pourquoi.** Les flottants ne représentent pas exactement les décimales :
`0.1 + 0.2` ne vaut pas `0.3` en binaire. Sur une clôture qui additionne des
centaines de dépenses puis divise, l'erreur s'accumule et les comptes ne tombent
pas juste. Les entiers rendent additions et comparaisons exactes.

**Conséquences.** Conversion à l'affichage et à la saisie. La division lors du
partage laisse un reste à attribuer explicitement, ce qui est préférable à une
erreur silencieuse.

---

## ADR-005 — Partage automatique entre tous les membres

**Contexte.** L'énoncé mentionne un « compte dépenseur » optionnel, mais ne dit
pas qui participe à chaque dépense.

**Options.** Partage entre tous les membres acceptés, ou sélection des
participants dépense par dépense (nécessitant une table
`expense_participants`).

**Décision.** Partage automatique entre tous les membres acceptés, à parts
égales.

**Pourquoi.** L'énoncé ne demande jamais de choisir des participants. Le bonus
porte sur l'optimisation des virements, pas sur la granularité du partage.
Ajouter une table, des écrans et des cas limites pour une fonctionnalité non
demandée aurait coûté du temps au détriment de la préparation de l'atelier.

**Conséquences.** Modèle plus simple. Un partage inégal serait une évolution
possible, sans remise en cause du schéma existant.

---

## ADR-006 — Rendu côté serveur avec html/template et htmx

**Contexte.** Choisir entre un rendu serveur et une application monopage
appelant une API JSON.

**Décision.** `html/template` de la bibliothèque standard, avec htmx pour les
interactions partielles. Pas de build frontend.

**Pourquoi.** L'évaluation porte sur Gin, pas sur un framework JavaScript. Une
SPA aurait déporté l'essentiel du travail côté client et rendu l'atelier moins
lisible : nos camarades auraient passé leur temps sur React plutôt que sur Gin.

`html/template` échappe automatiquement les variables selon leur contexte, ce
qui neutralise les injections XSS sans effort. Vérifié : un nom contenant
`<script>alert(1)</script>` s'affiche comme texte.

**Conséquences.** Un piège notable : `html/template` partage un espace de noms
global. Charger toutes les pages dans une même instance ferait que le dernier
`{{ define "content" }}` écraserait les précédents. Nous compilons donc une
instance par page, ce que `LoadHTMLGlob` de Gin ne permet pas — d'où notre
`internal/web/render.go`.

---

## ADR-007 — Tailwind par CDN

**Décision.** Tailwind chargé depuis un CDN, compilé dans le navigateur.

**Pourquoi.** Aucune installation, aucun build. Les participants de l'atelier
écrivent du HTML immédiatement présentable sans configurer Node.

**Conséquences.** Tailwind déconseille explicitement cet usage en production :
la feuille complète est téléchargée et les classes sont calculées côté client.
Un déploiement réel exigerait un build qui ne conserve que les classes
utilisées. Compromis assumé pour un projet pédagogique, à mentionner en
atelier.

---

## ADR-008 — Architecture en quatre couches

**Contexte.** Gin n'impose aucune structure de projet, contrairement au MVC de
Rails ou Laravel.

**Décision.** `domain` (structures métier), `repository` (accès base),
`service` (règles métier), `http` (handlers et middlewares).

**Pourquoi.** La logique de clôture — calcul des soldes, minimisation des
virements — doit être testable sans serveur HTTP ni base de données. La
séparation le permet. Elle évite aussi que quatre personnes travaillant en
parallèle inventent chacune sa propre organisation.

**Conséquences.** Plus de fichiers pour une fonctionnalité simple. Le câblage
des dépendances est manuel dans `main.go` : contrairement à Spring ou NestJS,
rien ne vérifie qu'une dépendance est fournie, et un champ oublié compile puis
plante à l'exécution.

---

## ADR-009 — Base partagée sur une VM OpenStack du FIL

**Contexte.** Le sujet impose un backend partagé entre les étudiants du groupe,
et suggère « une connexion à une instance de base de données partagée entre
plusieurs machines ». Cette base doit aussi être joignable depuis la salle de
TP le jour de notre atelier.

**Blocage rencontré.** Le réseau eduroam filtre les ports sortants. Tests
`nc -zv` depuis le campus :

| Hôte              | Port | Résultat |
|-------------------|------|----------|
| google.com        | 443  | ouvert   |
| github.com        | 22   | ouvert   |
| Neon (pooler)     | 5432 | bloqué   |
| Supabase (pooler) | 6543 | bloqué   |

Les mêmes hôtes répondent en partage de connexion mobile : le blocage vient du
réseau, pas des hébergeurs. Or aucun hébergeur PostgreSQL managé (Neon,
Supabase, Aiven) ne permet de choisir son port d'écoute, celui-ci faisant
partie de leur architecture mutualisée. Une base managée est donc inaccessible
depuis la salle.

**Options.**

| Option                                  | Verdict                                                                    |
|-----------------------------------------|----------------------------------------------------------------------------|
| PostgreSQL auto-hébergé sur le port 443 | fonctionnel, mais nécessite un serveur payant ou une carte bancaire        |
| Tunnel SSH monté par le code Go         | fonctionnel, mais met de la plomberie réseau dans l'application            |
| Proxy WebSocket vers Neon               | non documenté pour Go, non supporté                                        |
| GitHub Codespaces                       | supprime le problème, mais déplace tout l'environnement dans le navigateur |
| VM sur l'OpenStack du FIL               | retenue                                                                    |

**Décision.** Une machine virtuelle Ubuntu 24.04 sur `cloud.univ-lille.fr`,
hébergeant PostgreSQL 17 dans un conteneur Docker.

**Pourquoi.** La VM est sur le réseau `172.28.0.0/16` de l'université : le
trafic entre les postes de TP et la base ne sort jamais du réseau
universitaire, donc le filtrage d'eduroam ne s'applique pas. Vérifié :
`nc -zv <vm> 5432` répond depuis eduroam.

L'architecture reste classique — une `DATABASE_URL` standard, rien à installer
ni à configurer côté participant, ce qui répond à la consigne « les autres
participants sont ignorants ». Gratuit, hébergé par l'université, donc sans
question de transfert de données hors UE.

**Conséquences.** La base n'est pas joignable hors campus sans le VPN du FIL.
Nous administrons nous-mêmes le serveur : configuration, pare-feu, sauvegardes,
supervision — ce qui alimente directement la partie « configuration et
déploiement en production » attendue dans l'atelier. La même VM servira à
déployer l'application.

Le changement d'hébergeur n'aurait coûté qu'une variable d'environnement : le
code ne connaît pas son fournisseur de base. C'est le bénéfice concret du
principe III des Twelve-Factor Apps.

**TLS.** PostgreSQL écoute en TLS 1.3 avec un certificat auto-signé monté en
lecture seule dans le conteneur. Vérifié : `pg_stat_ssl` confirme `ssl = t`
sur la connexion de l'application.

Le certificat chiffre le transport mais n'atteste pas l'identité du serveur,
faute d'autorité de certification. Suffisant ici : la VM est sur le réseau
interne et nous contrôlons les deux extrémités. Un déploiement public
exigerait un certificat signé.

Piège rencontré : PostgreSQL refuse de démarrer si sa clé privée n'appartient
pas à son utilisateur. L'UID de `postgres` est 70 sur les images Alpine,
contre 999 sur les images Debian.

**Détail de mise en œuvre.** Le mot de passe de la base est généré avec
`openssl rand -hex 24` et non `-base64` : les caractères `/`, `@`, `:` et `?`
ont une signification dans une URL de connexion et la coupent silencieusement.
Erreur rencontrée et à signaler aux participants.


---

## ADR-010 — Règles de la clôture

**Contexte.** L'énoncé demande, à la clôture, la liste des comptes qui
doivent donner et recevoir, et en bonus « qui donne combien à qui en
minimisant les virements ». Plusieurs points ne sont pas précisés.

**Décisions.**

| Question                        | Choix                                                             |
|---------------------------------|-------------------------------------------------------------------|
| Qui peut clôturer ?             | le créateur du porte-monnaie uniquement                           |
| Dépense sans payeur             | révisé : voir ADR-011                                             |            |
| Centimes qui ne se divisent pas | un centime de plus aux premiers membres, par ordre d'identifiant  |
| Calcul des soldes               | `SUM ... GROUP BY` en SQL brut, partage et virements en Go        |
| Minimisation des virements      | paires exactes d'abord, puis glouton (au plus n − 1 virements)    |
| Remboursement                   | simulé : le débiteur clique, on enregistre la date (`settled_at`) |

**Pourquoi.**

- *Créateur seul.* Clôturer est irréversible : les dépenses sont figées. Le
  laisser à n'importe quel membre permettrait de couper l'herbe sous le pied
  des autres.
- *Dépense sans payeur.* Si personne n'est crédité, l'argent dépensé
  disparaît du calcul et la somme des soldes ne vaut plus zéro. La personne
  qui saisit la dépense est la plus probablement celle qui l'a réglée.
- *Reste des centimes.* 1 € entre trois ne tombe pas juste. Plutôt que de
  perdre un centime ou d'arrondir en flottant (ADR-004), on l'attribue
  explicitement et de façon reproductible. La somme des soldes vaut ainsi
  toujours exactement zéro.
- *SQL / Go.* PostgreSQL additionne sans charger les dépenses en mémoire
  (« Think HUGE »). Le partage et l'algorithme restent en Go, dans des
  fonctions pures testées sans base (ADR-008).
- *Glouton.* Le minimum absolu de virements est un problème NP-difficile :
  il faut chercher le plus grand nombre de sous-groupes dont les soldes
  s'annulent. L'heuristique retenue reste en O(n log n), garantit au plus
  n − 1 virements, et l'étape des paires exactes règle le cas fréquent où
  deux montants se compensent tout juste.

**Conséquences.** Les virements sont calculés une seule fois, à la clôture,
et enregistrés dans `settlements` : ils ne changent plus ensuite. La clôture
et l'écriture des virements se font dans une transaction, et la condition
`status = 'open'` empêche une double clôture simultanée.


---

## ADR-011 — Dépenses sans payeur exclues des soldes

**Contexte.** L'ADR-010 imputait une dépense sans payeur à la personne qui
l'avait saisie. Déclarer une dépense commune suffisait donc à devenir
créancier d'un argent jamais avancé (relevé en relecture : 100 € non
attribués comptés comme payés par l'auteur de la saisie).

**Options.**

| Option                               | Verdict                                                           |
|--------------------------------------|-------------------------------------------------------------------|
| Imputer à l'auteur de la saisie      | crédite quelqu'un qui n'a rien avancé                             |
| Garder dans le partage sans créditer | somme des soldes négative : de l'argent dû que personne ne reçoit |
| Exclure du payé et du partage        | retenue                                                           |

**Décision.** Une dépense sans payeur ne crédite personne et n'entre pas
dans le partage. `PaidByAccount` ne compte que les dépenses avec
`payer_id IS NOT NULL`. Comme `CalculerSoldes` déduit le total des sommes
payées, le partage s'ajuste seul et la somme des soldes reste à zéro.

**Conséquences.** Le total affiché à la clôture vient de
`ExpenseRepository.Totals`, comme sur la page des dépenses, et la part non
répartie est indiquée à part. Pour qu'une dépense soit partagée, il faut
lui associer un payeur.

---

## ADR-012 — Déploiement par image Docker transférée

**Contexte.** L'application doit tourner sur la VM du FIL, à côté de sa base.

**Options.** Construire sur la VM depuis Git, passer par un registre, ou
transférer l'image par SSH.

**Décision.** Construction en local, `docker save` puis `scp`, `docker load`
sur la VM.

**Pourquoi.** La VM n'a que 2 Go de RAM et pas de toolchain Go. Un registre
demanderait un compte et des identifiants à gérer. Le transfert d'une image
de 13 Mo prend deux secondes.

**Conséquences.** Le déploiement est manuel : pas de livraison continue.
Acceptable pour un projet de quelques semaines, à automatiser dans un vrai
projet — c'est précisément ce qu'apporte une chaîne de CI/CD.

L'image est construite pour `linux/amd64` : un binaire Go est lié à une
architecture, et développer sur Mac ARM pour déployer sur x86 impose de le
préciser. Contrainte absente des langages interprétés.