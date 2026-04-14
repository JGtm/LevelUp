> [!WARNING]
> DOCUMENT DE CADRAGE — a valider avant execution.
> Les decisions D1-D7 doivent etre tranchees, le POC initial (2 jours) doit passer, et le corpus de reference Go doit etre gele avant execution.

> [!NOTE]
> **Revision du 2026-04-13** — Plan complete avec : inventaire fonctionnel complet, catalogue
> des algorithmes metier a porter, inventaire des requetes SQL, strategie de tests, analyse
> des erreurs et risques du plan initial, matrice de complexite par feature, strategie i18n,
> couverture PvE/Firefight, et corrections factuelles.
> **Revision du 2026-04-13 (2)** — Ajout : Sprint 0 POC (2 jours), criteres d'abandon (kill switch),
> modele de deploiement cible (binaire unique, go:embed React, sous-commandes),
> runbook migration donnees utilisateurs, strategie d'evolution produit pendant le portage,
> gestion multi-joueurs (pools par gamertag), opportunites Go (zero-dep, concurrence backfill,
> SSE, -race), adaptation solo dev, graceful shutdown, notifications Discord, correction
> estimation LOC (25-35K Go), src/app/ dans la matrice, MSAL Go maturite confirmee.
> **Revision du 2026-04-13 (3)** — Consolidation structurelle : fusion des conditions de succes
> (11→18) et d'echec (11→16) en listes uniques, section SPNKr condensee (120→30 lignes),
> POC A/B/C/D fusionnes dans Sprint 0, DoD par sprint ajoute aux livrables Phase 0,
> Risques 5-6 alignes avec la strategie d'evolution (pas de freeze total), D2 mis a jour
> (binaire unique), suppression des sections dupliquees du complement.

# Plan de migration Python -> Go

> [!IMPORTANT]
> **Statut documentaire** — ce plan cadre le chantier de remplacement du runtime backend Python par Go.
> - Il ne supersede pas automatiquement des documents d'un autre chantier ou d'un autre projet.
> - Prerequis dur : le corpus de reference Go doit etre gele avant l'ouverture effective du Sprint 0 Go : facade web cible, contrats API, fixtures, golden values et matrice de couverture.
> - La reference contractuelle de depart du portage Go est le produit actuel gele : facade web, contrats API, fixtures et suites de validation associees.
> - Aucun agent ne doit utiliser ce document seul.

## Lecture obligatoire avant toute action

1. [PLAN_MIGRATION_PYTHON_TO_GO.md](PLAN_MIGRATION_PYTHON_TO_GO.md) — trajectoire, phases, gates, risques, decisions.
2. [SPRINT_ROADMAP.md](SPRINT_ROADMAP.md) — decoupage lineaire de A a Z en 28 sprints, pour repartir les taches et suivre l'avancement.
3. [MATRIX.md](MATRIX.md) — couverture package/script/commande/bitmask, surfaces hors scope et statut de chaque zone.
4. [OPS_COMPAT_CHECKLIST.md](OPS_COMPAT_CHECKLIST.md) — compat auth/jobs/mode de test, exploitation, packaging et migration utilisateur.
5. [ZERO_PYTHON_STRATEGY.md](ZERO_PYTHON_STRATEGY.md) — objectif zero Python, inventaire module par module, strategie d'extinction SPNKr.
6. [GO_MIGRATION_CHECKLIST.md](GO_MIGRATION_CHECKLIST.md) — suivi vivant du chantier, statuts d'avancement, preuves attendues et blocages.
7. Regle simple : si une surface n'apparait ni dans la matrice ni dans la checklist ops, elle est consideree comme non couverte.

## Objectif

Remplacer progressivement le runtime Python de LevelUp par un backend et des outils Go, sans casser les invariants metier existants : DuckDB v6, auth Halo, sync/backfill, logique analytique et parcours produits V7 deja geles.

Ce plan part d'un constat simple : une migration Python -> Go ne doit pas devenir une reecriture big bang. Elle doit s'appuyer sur la facade web, les contrats API et les golden values deja stabilises, puis remplacer Python derriere cette reference sans reouvrir un chantier produit en parallele.

## Resume executif

- La migration complete vers Go est faisable, mais elle doit etre menee comme un programme de remplacement progressif, pas comme une reecriture simultanee.
- Le but est de remplacer Python, pas de relancer une migration frontend ou une refonte produit.
- DuckDB doit rester la source de verite. Revoir en meme temps le schema, le moteur et le runtime multiplierait inutilement le risque.
- Les ecrans read-only doivent etre portes avant l'auth, les jobs et le moteur de sync.
- Les calculs aujourd'hui portes par Polars ne doivent pas etre recodes "a l'intuition" en Go. Il faut basculer soit vers SQL explicite, soit vers des pipelines Go verifies contre des golden values.
- Le programme doit utiliser une approche strangler au moment de l'integration et de la bascule : Python et Go cohabitent a ce niveau, meme si l'implementation locale se fait tranquillement dans un worktree dedie.

## Ce que "remplacer Python" veut dire

1. Remplacer l'API backend Python, les jobs longs, la sync/backfill et les CLI d'exploitation par des equivalents Go.
2. Garder la facade web actuelle comme consommateur de reference, pas comme chantier a reouvrir.
3. Garder par defaut le schema DuckDB, le contrat HTTP et les fixtures de validation tant qu'une divergence n'est pas explicitement decidee et tracee.
4. Sortent du scope : refonte frontend, redesign produit, changement gratuit des payloads, et double maintenance durable Python/Go apres bascule finale.

## Methodologie explicite du plan

### Nom de la methode

**Strangler oriente parite, contract-first et stateful-last pour migration Python -> Go**.

Cette methode est adaptee a un remplacement de runtime entre deux langages tres differents : Python permet beaucoup d'implicite, de coercions et de transformations compactes ; Go force l'explicite, la separation des chemins d'erreur et la modelisation stricte. Le plan ne cherche donc pas a "traduire" Python en Go ligne a ligne. Il cherche a remplacer Python par slices observables et verifiables.

### Principes de la methode

1. **Contract-first** : figer d'abord la reference externe observable (OpenAPI, payloads, statuts HTTP, semantique des nulls, tri, pagination, filtres).
2. **Python as oracle** : Python sert d'oracle de comportement tant que Go n'a pas prouve sa parite ; Python n'est pas un blueprint a recopier module par module.
3. **Read-only first** : commencer par les parcours read-only avant auth, jobs, sync et ecritures DuckDB.
4. **Stateful-last** : porter en dernier ce qui concentre le plus de risques de bord : auth, sessions, jobs persistants, sync, backfill, refresh des vues, media indexing.
5. **SQL-first** : quand Python s'appuie sur Polars ou des post-traitements implicites, tenter d'abord un calcul SQL explicite ; n'ecrire une logique Go que si SQL ne suffit pas proprement.
6. **Parity-before-switch** : aucune surface n'est basculee tant qu'un corpus golden values ou un test de parite n'a pas demontre que Go reproduit la meme semantique utile.
7. **One-way replacement** : on garde Python comme baseline tant que Go n'est pas merge ; une fois la bascule finale validee, on n'organise pas une cohabitation durable des deux runtimes.

### Cycle standard d'un lot Python -> Go

1. Nommer la surface a porter et geler son contrat observable.
2. Capturer ou completer les golden values de cette surface.
3. Identifier ce qui, dans Python, releve de : contrat, logique metier, SQL, coercion implicite, side effects.
4. Reconstituer la frontiere Go cible : handler, service, repository, adaptateur externe.
5. Porter d'abord le chemin read-only ou deterministe si la surface le permet.
6. Comparer Go a Python sur le corpus de reference.
7. Corriger les ecarts jusqu'a parite ou documenter un ecart volontaire.
8. Integrer la surface derriere feature flag, shadow mode ou bascule controlee.
9. Supprimer le morceau Python seulement quand le lot Go a passe son gate de sortie.

### Anti-patterns specifiques Python -> Go

1. Traduire ligne a ligne un module Python complexe vers Go.
2. Reproduire en Go des coercions implicites Python non documentees.
3. Porter du Polars en Go "a l'intuition" sans oracle chiffre.
4. Changer en meme temps la forme JSON, la logique metier et l'implementation.
5. Porter d'abord la sync ou l'auth avant d'avoir prouve la parite read-only.
6. Confondre "compile" avec "meme comportement".

### Regles de suivi de projet obligatoires

Le plan ne doit pas seulement definir quoi porter. Il doit aussi imposer comment suivre le chantier pour savoir ce qui est ouvert, ce qui est bloque, ce qui est termine et dans quel ordre les lots ont le droit d'avancer.

#### Sources de verite du suivi

1. [PLAN_MIGRATION_PYTHON_TO_GO.md](PLAN_MIGRATION_PYTHON_TO_GO.md) fixe l'ordre macro, les phases, les gates et les conditions d'ouverture/fermeture.
2. [MATRIX.md](MATRIX.md) porte la couverture exhaustive : quelles surfaces existent, si elles sont a porter, a supprimer, hors scope ou a garder temporairement.
3. [OPS_COMPAT_CHECKLIST.md](OPS_COMPAT_CHECKLIST.md) porte les contraintes runtime/exploitation : auth, jobs, mode de test, packaging, migration utilisateur.
4. [GO_MIGRATION_CHECKLIST.md](GO_MIGRATION_CHECKLIST.md) porte l'avancement vivant du chantier lot par lot : non demarre, en cours, bloque, termine, prochaine preuve attendue.
5. [../thought_log.md](../thought_log.md) porte les decisions techniques et arbitrages qui changent le plan, le perimetre ou les hypotheses.

#### Statuts d'avancement obligatoires

1. `non_demarre` : lot identifie, pas ouvert.
2. `cadre` : contrat, corpus, golden values et gate de sortie definis.
3. `en_cours` : implementation active en worktree.
4. `en_verif_parite` : code runnable/testable, comparaison Python vs Go en cours.
5. `pret_integration` : parite utile demontree, ecarts traces, lot pret pour revue/bascule controlee.
6. `termine` : gate de sortie passe, docs de suivi mises a jour, destin du morceau Python explicitement tranche.
7. `bloque` : dependance externe ou prerequis manquant, raison visible noir sur blanc.

#### Regles d'ouverture et de fermeture d'un lot

1. Aucune surface ne passe en `en_cours` sans ligne explicite dans la matrice et dans la checklist de suivi.
2. Aucun lot d'une phase N+1 ne passe en `en_cours` tant que le gate de la phase N n'est pas passe, sauf exploration ou POC explicitement notes comme tels.
3. Un lot ne passe de `non_demarre` a `cadre` que si son contrat cible, son oracle Python et son corpus de verification sont identifies.
4. Un lot ne passe de `cadre` a `en_cours` que si son critere de sortie est ecrit et si son impact sur DuckDB/auth/jobs est compris.
5. Un lot ne passe de `en_cours` a `en_verif_parite` que s'il est revenu a un etat runnable ou testable.
6. Un lot ne passe de `en_verif_parite` a `pret_integration` que si la parite utile est demontree ou qu'un ecart volontaire est documente.
7. Un lot ne passe de `pret_integration` a `termine` que si la documentation de suivi est a jour et si la strategie sur le morceau Python remplace est explicite : garder temporairement, supprimer ou conserver via bridge etroit.

#### Cadence de mise a jour obligatoire

1. Mettre a jour la matrice et la checklist avant d'ouvrir un lot.
2. Mettre a jour la checklist a chaque changement de statut reel.
3. Mettre a jour le thought log a chaque decision technique qui modifie le perimetre, l'ordre, les gates ou une hypothese de portage.
4. A la fin de chaque session de travail, noter ce qui a ete fait, ce qui reste a prouver et le prochain pas concret.
5. Avant toute suppression de Python, toute bascule ou tout merge important, verifier que le lot est marque `termine` et que sa preuve de parite est referencable.

## Hypothese de travail : worktree dedie

Ce plan suppose explicitement que l'implementation se fait dans un worktree dedie. Cela change la discipline de travail locale, mais pas les criteres de validation avant integration.

## Ce que cela change vraiment

1. Il n'est pas necessaire de garder le projet localement fonctionnel a chaque commit intermediaire dans le worktree.
2. Il est acceptable de casser provisoirement les imports, le build, les scripts ou une partie des parcours pendant un gros refactor.
3. Il est acceptable de faire des deplacements massifs de packages, de renommer large, ou de remplacer des frontieres entieres sans maintenir une compatibilite locale temporaire.
4. Les bridges temporaires, feature flags, shadow mode et rollback ne sont pas des obligations de confort pendant le dev local ; ce sont des obligations d'integration et de bascule.

## Ce que cela ne change pas

1. Avant merge, revue formelle ou bascule, le lot doit revenir a un etat testable et explicable.
2. Les contrats cibles, les golden values et la parite metier restent la reference obligatoire.
3. Les migrations DuckDB, les jobs longs, l'auth et les scripts d'exploitation doivent toujours avoir une strategie claire.
4. Le worktree dedie ne justifie pas de perdre la traçabilite, d'ignorer les registres anti-oubli ou de supprimer du Python sans plan de remplacement.

## Regle de pilotage associee

Pendant l'implementation dans le worktree, on optimise pour la vitesse de transformation. Avant integration, on redevient strict : tests, parite, runbook, rollback, observabilite et criteres Go/No-Go.

## Workflow concret de chantier dans le worktree

Ce workflow est volontairement pragmatique : il assume que le worktree peut etre temporairement casse, mais il impose des checkpoints de remise en etat avant toute integration.

### Etape 1 - Ouvrir un lot explicite

1. Nommer le lot par intention metier ou technique : repositories DuckDB, auth/session, sync scope, SPNKr bridge, scripts d'exploitation.
2. Declarer ce que le lot a le droit de casser localement : build complet, imports temporaires, scripts legacy, routes non cibles.
3. Declarer ce que le lot n'a pas le droit de casser sans preuve immediate : fixtures, golden values, schemas DuckDB, secrets, corpus de reference.
4. Lier le lot a un ou plusieurs criteres de sortie observables.

### Etape 2 - Refactor libre dans le worktree

1. Autoriser les deplacements massifs, renames, extractions et suppressions provisoires.
2. Autoriser un remplacement direct de Python par Go a l'interieur du worktree si cela accelere la transformation.
3. Accepter que le projet ne soit plus runnable localement pendant cette phase.
4. Continuer a tenir a jour la matrice Python -> Go, le registre des dependances et les decisions sur les bridges.

### Etape 3 - Checkpoint structurel interne

1. Reconstituer au moins un build minimal ou une compilation partielle sur la zone touchee.
2. Verifier que les contrats d'entree/sortie du lot sont toujours lisibles.
3. Verifier qu'aucun fichier critique n'est sorti du radar de la matrice non exhaustive.
4. Verifier que les migrations DuckDB et les effets de bord restent explicables.

### Etape 4 - Remise en etat avant integration

1. Remettre le lot dans un etat runnable ou testable.
2. Rebrancher les tests de parite, les golden values et les smoke tests concernes.
3. Retablir les chemins de build, les scripts minimaux et les points d'entree necessaires a la revue.
4. Documenter les ecarts volontaires, les dettes gardees et les bridges temporaires.

### Etape 5 - Gate pre-merge

1. Verifier que le lot est compréhensible sans contexte oral implicite.
2. Verifier que la parite utile au lot est demontree ou que l'ecart est volontaire et trace.
3. Verifier que le rollback, le shadow mode ou le bridge requis pour l'integration existent si le lot touche une surface active.
4. Refuser la fusion d'un lot uniquement "presque fini" qui resterait structurellement opaque ou non testable.

## Lots autorises a casser localement

1. Reorganisation des packages Go.
2. Remplacement d'une couche entiere de repositories ou de services.
3. Suppression temporaire d'anciens adapteurs internes pendant une reecriture.
4. Restructuration des scripts et des points d'entree CLI.

## Lots qui doivent revenir propres avant merge

1. Tout ce qui touche les schemas DuckDB ou leurs migrations.
2. Tout ce qui touche l'auth, les secrets, les cookies ou les caches de tokens.
3. Tout ce qui touche les endpoints deja consommes par la facade V7.
4. Tout ce qui touche sync, backfill, smoke test, backup, restore ou media indexing.

## Decision de cadrage

### Ce que le plan assume

- La facade web actuelle reste le consommateur cible et la reference contractuelle de depart.
- L'architecture de donnees v6 reste intacte : metadata, shared matches, player DBs, vues SQL garanties.
- Les contrats fonctionnels existants restent la reference : filtres, sessions, carriere, match history, explorer, match view, settings, setup.
- Les scripts d'exploitation et la sync finissent eux aussi en Go, mais seulement apres stabilisation de la couche API read-only.

### Ce que le plan refuse

- Pas de migration big bang.
- Pas de changement simultane de schema DuckDB, d'auth, de design produit et de runtime.
- Pas de portage direct de chaque module Python sans redecoupage. Le but est de reconstituer des frontieres propres, pas de reproduire les memes couplages dans une autre langue.
- Pas de bascule production sans mode shadow et sans parite automatisee.

## Perimetre exact a porter vers Go

### Couches a migrer

1. Couche HTTP/API aujourd'hui exposee par FastAPI.
2. Couche services et orchestration de pages.
3. Couche repositories / acces DuckDB.
4. Couche auth / session / jobs longs.
5. Couche sync, backfill, media indexing et outillage CLI.
6. Scripts de maintenance et de diagnostic relies au runtime Python.

### Couches a conserver telles quelles au debut

1. Frontend React/TypeScript.
2. Structure des donnees DuckDB et les fichiers de configuration existants.
3. Fixtures, corpus de tests et golden values deja utilises pour la parite.

## Risques techniques structurants

### Risque 1 - DuckDB en Go

Le programme ne doit pas demarrer sans valider un driver DuckDB stable sur Windows et Linux, avec une strategie claire sur les verrous de fichiers, les connexions read-only/read-write et le comportement en process multiples.

### Risque 2 - Polars n'a pas d'equivalent direct rentable

Une part importante du code analytique Python s'appuie sur DuckDB, Polars et Pydantic. Go n'offre pas un remplacement 1:1. Il faut donc preferer :

1. SQL-first quand un calcul peut vivre proprement dans DuckDB.
2. Pipelines Go explicites quand le calcul doit sortir du SQL.
3. Comparaison systematique avec golden values avant toute bascule.

### Risque 3 - Auth Halo / MSAL

Le flux device code, le cache MSAL et l'echange de tokens Halo sont des zones a fort risque. Il faut les traiter comme une tranche separee, avec POC obligatoire avant toute promesse de migration complete.

### Risque 4 - Sync et backfill

Le moteur de sync concentre beaucoup d'orchestration, de migrations idempotentes, de backfill, de jobs longs et de controle de schema. C'est la derniere couche a porter, pas la premiere.

## Architecture cible recommande en Go

## Principe general

Introduire Go a cote de Python, puis basculer progressivement les responsabilites. Dans le worktree dedie, cette cohabitation n'a pas besoin d'etre maintenue a chaque instant. En revanche, tant que la parite n'est pas validee, le service Go ne doit pas se presenter comme unique proprietaire du produit au moment de l'integration ou de la bascule.

## Structure repo proposee

```text
apps/
  api/              # API Python existante pendant la transition
  web/              # Frontend React/TypeScript conserve
  go-api/
    cmd/
      levelup-api/
      levelup-jobs/
      levelup-sync/
    internal/
      api/
      bootstrap/
      players/
      filters/
      career/
      history/
      explorer/
      matches/
      settings/
      auth/
      jobs/
      media/
      sync/
      platform/
        config/
        duckdb/
        http/
        logging/
        telemetry/
        testdata/
```

## Regles de conception

1. Le service Go expose des contrats de page stables, pas des details internes de tables.
2. Les calculs critiques restent cote backend, jamais reimplementes dans le frontend.
3. Les graphes doivent sortir sous forme de donnees ou de series pretes a tracer, pas comme une imitation de la stack Plotly Python.
4. Les jobs longs doivent avoir un modele explicite : start, status, cancel si necessaire, result, warnings, erreurs.
5. Les migrations DuckDB doivent rester idempotentes et automatisees.
6. Graceful shutdown obligatoire : intercepter `SIGTERM`/`SIGINT` (ou `os.Interrupt` sur Windows), drainer les requetes HTTP en cours, fermer proprement les connexions DuckDB, et ne jamais interrompre un sync en pleine ecriture (attendre le commit ou rollback).
7. Notifications Discord : le backend Go doit etre capable d'envoyer des embeds Discord (webhook) apres sync/backfill, exactement comme le Python actuel (`src/utils/discord_notifier.py`). Inclure : embeds bilingues, thumbnail upload, anti-spam via `discord_notified_at`, notifications new version.

## Decisions techniques a prendre avant la premiere ligne de Go

1. Valider un driver DuckDB compatible Windows/Linux et le comportement de lock associe.
2. Choisir le socle HTTP Go et la strategie de validation des payloads.
3. Choisir la forme des contrats de charting : series JSON, points, buckets, annotations, jamais des widgets Python encodes.
4. Choisir la strategie de session et de cookies pour remplacer le modele actuel.
5. Choisir la strategie de logging, trace, request_id et observabilite.
6. Decider si le cache de tokens Halo est porte nativement en Go des la phase auth, ou transite provisoirement par un bridge Python a eteindre ensuite.
7. Decider la strategie de generation OpenAPI et de types frontend.

## Comment etre sur de ne rien oublier

Ce document n'est volontairement plus autosuffisant.

- La couverture package/script/commande/bitmask est maintenue dans [MATRIX.md](MATRIX.md).
- Les exigences de compatibilite runtime et d'exploitation sont maintenues dans [OPS_COMPAT_CHECKLIST.md](OPS_COMPAT_CHECKLIST.md).
- Les contrats applicatifs a preserver doivent etre documentes dans ce chantier ou ses sous-docs dedies.

## Regle simple

Si une surface Python, un script, une route ou un comportement runtime n'apparait dans aucun de ces sous-docs, il est considere comme non couvert.

## Routine de couverture obligatoire avant chaque phase

1. Mettre a jour la matrice avant de toucher une nouvelle surface.
2. Mettre a jour la checklist ops avant de modifier auth, jobs, mode de test, packaging ou runbook.
3. Verifier que les contrats applicatifs cibles restent couverts par les tests et golden values existants.

## Definition operationnelle de rien oublier

La couverture est suffisante seulement si :

1. Chaque surface active a un statut explicite dans la matrice.
2. Chaque comportement runtime critique a une exigence explicite dans la checklist ops.
3. Chaque contrat applicatif touche par le portage renvoie a un corpus de parite ou a une decision documentee.

## Strategie SPNKr / Client API Halo

SPNKr est une librairie Python tierce qui fait des appels HTTP vers les endpoints 343 Industries. `src/ports/api.py` definit deja un `HaloAPIPort` abstrait. En Go, le vrai travail n'est pas "migrer SPNKr" mais implementer `HaloAPIPort` en Go (client HTTP direct).

### Position

1. **Ne pas migrer SPNKr en premier** — la priorite est DuckDB + read-only.
2. **Ne pas la laisser comme dette permanente** — tant qu'elle reste Python, le backend Go n'est pas autonome.
3. **L'isoler derriere une frontiere claire** pendant la transition.

### Ce qu'il faut reproduire en Go

- 6 endpoints 343i reels (profile, match list, match details, skill, discovery, economy)
- Rate limiting : 60 req/min avec retry exponentiel
- Cycle de tokens : spartan_token + clearance_token avec refresh MSAL
- Circuit breaker sur 3 echecs consecutifs
- Parsing des reponses JSON (modeles Pydantic → structs Go)

### Temporaire : bridge Python etroit (Phase 3-4)

Si le portage complet du client HTTP est trop risque a un moment donne, garder un adapter Python minimal (1-2 operations) derriere un contrat explicite : entrees, sorties, erreurs, timeouts. Le bridge doit etre etroit et remplacable — jamais une dependance diffuse.

### Criteres d'extinction du bridge

1. Tous les endpoints Halo utiles ont un equivalent Go teste sur fixtures.
2. 3 cycles de sync passent sans recours au bridge Python.
3. Les secrets et caches ne dependent plus d'un processus Python.

## POC et validation initiale

Le Sprint 0 (2 jours — voir section "Ordre de migration recommande") couvre les validations critiques :

- **DuckDB Go read-only** : ouvrir les 3 types de DB, executer des requetes representatives, verifier types et locks.
- **Client HTTP Go** : MSAL Go device code flow (`AcquireTokenByDeviceCode`), endpoint `/health` + `/bootstrap`.
- **Compatibilite** : fichiers DuckDB crees par Python ouverts par go-duckdb, cache MSAL Python deserialise en Go.

**Gate** : si un de ces points echoue de facon non contournable, la migration est re-evaluee.

> Note : le plan initial contenait des POC A/B/C/D separes. Ils ont ete fusionnes dans le Sprint 0 pour eviter la duplication. Le POC B (bridge SPNKr) n'est plus necessaire a ce stade — SPNKr est un client HTTP simple a reimplementer directement en Go (voir "Strategie SPNKr").

## Ordre de migration recommande

### Sprint 0 — POC rapide (2 jours max)

Objectif : valider en 2 jours que les briques fondamentales Go fonctionnent, avant tout travail de cadrage detaille. C'est un test de faisabilite, pas un prototype. Si ca ne passe pas, le plan s'arrete la.

**Jour 1 — DuckDB Go + types** :
1. `go mod init`, ajouter `github.com/marcboeker/go-duckdb`
2. **Verifier la version DuckDB embarquee par go-duckdb** : elle doit etre compatible avec les fichiers crees par DuckDB Python 1.4.4 (meme majeure+mineure). Si go-duckdb embarque une version differente, tester l'ouverture et verifier qu'aucune migration implicite du format de stockage ne se produit (`PRAGMA database_size` avant/apres).
3. Ouvrir `metadata.duckdb` en read-only, executer `SELECT * FROM career_ranks LIMIT 5`
4. Ouvrir `shared_matches_v2.duckdb` en read-only, executer une requete bootstrap joueur (Q1 du catalogue)
5. **Tester ATTACH via `database/sql`** : `database/sql` gere son pool de connections de facon transparente — il n'y a aucune garantie qu'une requete suivante s'execute sur la meme connection que l'ATTACH. Valider une des strategies suivantes : `sql.Conn` (connexion pinee), `ConnInitFunc` du driver, ou pool custom hors `database/sql`.
6. Verifier les types critiques : UBIGINT → uint64, TIMESTAMP WITH TIME ZONE → time.Time, VARCHAR, BOOLEAN
7. Tester le lock : ouvrir en read-write, tenter une seconde connexion read-write → observer le comportement
8. Compiler et executer sur Windows avec un toolchain CGo explicite et documente (ex : `w64devkit`, `tdm-gcc`, ou MSYS2 ucrt64). **Documenter exactement le toolchain utilise** pour reproduction CI.

**Jour 2 — HTTP + MSAL** :
1. Monter un `net/http` minimal avec un handler `/health` qui retourne le nombre de matchs en DB
2. Ajouter un handler GET `/api/bootstrap` qui lit les memes donnees que le Python
3. Comparer le JSON de sortie avec la golden value Python
4. Tester `github.com/AzureAD/microsoft-authentication-library-for-go` : instancier un `PublicClientApplication`, appeler `AcquireTokenByDeviceCode()`, verifier que le user_code + verification_url arrivent
5. **Tester la deserialisation du cache MSAL existant** : MSAL Python et MSAL Go peuvent avoir des formats de serialisation de cache differents. Lire le cache JSON cree par `SerializableTokenCache` Python et verifier que MSAL Go le comprend, ou documenter la strategie de migration (ex : invalidation explicite au premier demarrage Go).

**Gate Sprint 0** :
- ✅ DuckDB Go lit les 3 types de DB sans erreur sur Windows
- ✅ La version DuckDB embarquee par go-duckdb est compatible avec les fichiers Python 1.4.4 (pas de migration implicite)
- ✅ ATTACH fonctionne correctement avec la strategie de pool choisie (`sql.Conn`, `ConnInitFunc` ou pool custom)
- ✅ Les types UBIGINT/TIMESTAMP sont correctement mappes
- ✅ CGo compile sur Windows avec un toolchain documente et reproductible
- ✅ Un endpoint HTTP retourne un JSON coherent avec le Python
- ✅ MSAL Go device code flow fonctionne (au moins jusqu'au user_code)
- ✅ La strategie sur le cache MSAL existant est documentee (lecture directe ou invalidation)
- ❌ Si un de ces points echoue de facon non contournable → re-evaluer le plan

### Phase 0 - Cadrage, inventaire et corpus

Objectif : figer la reference avant d'ecrire du Go.

Travaux :

1. Inventorier les surfaces Python a porter : API, services, repositories, auth, sync, scripts, **y compris `src/app/`** (voir ci-dessous).
2. Figer les contrats API existants cote React et les payloads metiers critiques.
3. Constituer un corpus figé pour Career, Match History, Explorer, Match View, Setup et Settings.
4. Ecrire une matrice Python -> Go par package avec proprietaire, criticite et dependances.
5. Definir des golden values chiffrées pour les parcours prioritaires.

Livrables :

1. Inventaire complet des modules a migrer.
2. Liste des invariants non negociables.
3. Corpus de tests rejouable sous Windows et Linux.
4. Premier Go/No-Go technique sur DuckDB et auth.
5. **Definition of Done par sprint** : pour chaque sprint des phases 1-5, definir les criteres de sortie explicites (ex : "Sprint 1.2 termine quand Q1, Q2, Q3, Q5 passent en Go avec golden values identiques a < 0.01").

### Phase 1 - Socle Go read-only

Objectif : prouver que Go peut lire les memes donnees et exposer les memes contrats sans ecrire dans les DBs.

Travaux :

1. Monter le service Go, le healthcheck, le request_id, le logging et la config.
2. Implementer bootstrap, players et resolveur de filtres.
3. Valider la compatibilite avec la facade V7 via shadow mode ou feature flag au moment de l'integration ; le frontend n'est pas un chantier a refaire.
4. Comparer les payloads Go contre les payloads Python sur le corpus de reference.

Livrables :

1. Service Go runnable localement.
2. Endpoints read-only equivalents sur bootstrap, players, filters.
3. Suite de tests de parite backend Go vs golden values.

### Phase 2 - Portage des parcours read-only prioritaires

Objectif : deplacer d'abord la valeur visible et relativement stable.

Ordre recommande :

1. Career.
2. Match History.
3. Explorer.
4. Match View.
5. Last Match si derive du meme contrat.

Travaux :

1. Porter les services de page en respectant les contrats deja figes de la facade V7.
2. Transformer les calculs Polars en SQL ou en pipelines Go verifies.
3. Exposer des payloads de page ou de sous-ressources stables.
4. Mettre ces routes en shadow mode sur des fixtures puis sur des donnees reelles avant integration ; dans le worktree, il est acceptable de court-circuiter temporairement la version Python pour accelerer le refactor.

Livrables :

1. Parite fonctionnelle sur les parcours P1.
2. Rapport d'ecarts documente pour les differences assumees.
3. Feature flags de bascule par surface.

### Phase 3 - Settings, session, auth et jobs longs

Objectif : porter les mutations avant les traitements systeme lourds.

Travaux :

1. Porter GET/PATCH settings.
2. Porter le modele de session web, les cookies et le bootstrap associe.
3. Porter le device flow, le cache de tokens et l'echange Halo.
4. Porter les jobs smoke test, media reset/reindex et les statuts de job.

Livrables :

1. Parcours setup/settings fonctionnels sans Python sur le chemin nominal.
2. Tests de reprise de session, expiration et reauth.
3. Observabilite complete sur les routes sensibles.

### Phase 4 - Sync, backfill, media indexing et CLI

Objectif : basculer les traitements les plus risqués une fois les contrats web stabilises.

Travaux :

1. Porter le moteur de sync par sous-domaines, pas en bloc monolithique.
2. Porter les migrations idempotentes de schema DuckDB.
3. Porter les strategies de backfill et la modelisation equivalente a SyncScope.
4. Porter l'indexation media et les scripts critiques d'exploitation.
5. Reconstituer les commandes CLI equivalentes a sync.py, backfill_data.py, check_env.py et scripts de diagnostic.

Livrables :

1. Commandes Go equivalentes pour sync, backfill, smoke test et maintenance critique.
2. Tests de non-regression sur les schemas DuckDB et les effets de bord.
3. Benchmarks minimum sur debit, erreurs et comportement de lock.

### Phase 5 - Bascule et extinction de Python

Objectif : retirer Python seulement quand Go a deja prouve sa tenue sur la duree.

Travaux :

1. Passer les surfaces React une par une sur le backend Go.
2. Tant que le worktree Go n'est pas valide pour integration, la branche principale Python reste simplement la baseline non fusionnee ; apres bascule finale, pas de retour strategique durable vers Python.
3. Mesurer erreurs, latence, ecarts de calcul et issues de jobs.
4. Retirer progressivement les endpoints Python devenus redondants.
5. Supprimer les scripts Python restants seulement apres soak test de plusieurs cycles reels.

Note : dans ce plan, le "rollback" ne veut pas dire maintenir Python comme solution cible de secours apres bascule finale. Cela veut seulement dire : ne rien merger ni supprimer irreversiblement tant que le gate de validation n'est pas passe.

Livrables :

1. Backend Go canonique.
2. Python retire du chemin critique de prod.
3. Documentation d'exploitation, de build et de reprise entierement mise a jour.

## Chantiers transverses obligatoires

### 1. Parite et fixtures

Le programme doit vivre avec un corpus gelé et des golden values pour les ecrans critiques. Sans cela, chaque regression sera discutable et la migration deviendra un debat d'impression.

### 2. Contrats API

Les contrats React doivent etre figes avant le portage de masse. Il est beaucoup moins couteux de porter une implementation derriere un contrat stable que de redefinir front et back a chaque phase.

### 3. Observabilite

Il faut suivre : latence p50/p95, erreurs, ecarts de parite, statut des jobs, incidents de lock DuckDB, temps de sync, retries auth.

### 4. Packaging et exploitation

Le programme doit prevoir tres tot : Docker, Windows, lancement local, CI, generation OpenAPI, binaire API, binaire sync, variables d'environnement et secrets.

## Estimation d'effort

Ordre de grandeur realiste, sous reserve qu'il n'y ait pas de refonte produit parallele :

- 1 ingenieur backend principal : 4 a 7 mois.
- 2 ingenieurs backend + 1 support frontend : 3 a 5 mois.
- 1 seule personne a temps partiel : risque eleve de glissement au-dela de 8 mois.

Le vrai determinant n'est pas la taille du code Python seule. C'est la vitesse a laquelle les golden values, le driver DuckDB, l'auth Halo et le moteur de sync seront stabilises en Go.

## Conditions de succes

Traiter les conditions de succes comme une check-list d'actions a executer, pas comme des voeux generiques.

1. Geler les contrats API, les invariants de filtres et les golden values avant tout portage significatif.
2. Valider par POC le driver DuckDB Go sur Windows et Linux, avec lecture, ecriture, verrous et performance acceptables.
3. Ouvrir la migration par des endpoints read-only et prouver la parite sur bootstrap, filters, career, history, explorer et match view.
4. Porter les calculs analytiques en privilegiant SQL et des transformations deterministes, puis verifier chaque resultat contre les sorties Python de reference.
5. Valider la parite par surface via golden values (diff JSON Go vs Python) avant chaque bascule.
6. Isoler l'auth, la session et les jobs dans une tranche dediee, avec tests de reprise, expiration et reauth.
7. Porter le moteur de sync seulement apres stabilisation des parcours web read-only et des mutations settings/setup.
8. Prevoir un rollback simple surface par surface, sans migration de donnees ad hoc.
9. Mesurer la migration avec des indicateurs lisibles : latence, erreurs, ecarts de payloads, jobs failed, duree de sync.
10. Retirer Python uniquement quand Go couvre aussi les chemins d'exploitation, de maintenance et de reprise incident.
11. Autoriser un worktree local temporairement casse pendant les gros refactors, mais imposer un retour a un etat testable avant revue, merge ou bascule.
12. Chaque algorithme metier (performance score, LUSR, sessions, citations, killer/victim) a des golden values **chiffrees** et les tolerances sont documentees (ε < 0.01 pour les scores, ε < 0.1 pour mu/sigma).
13. Le systeme de backfill bitmask (22 bits) est numeriquement identique entre Python et Go — pas "equivalent", identique.
14. Le modele de concurrence Go (pool read-only + write lease) est teste sous charge avec au moins 10 requetes read paralleles + 1 sync write.
15. La facade V7 continue a fonctionner sans changement de semantique pendant tout le portage.
16. Les 14 langues i18n fonctionnent identiquement (traductions dynamiques via DuckDB).
17. Le mode PvE/Firefight est couvert (pas seulement le PvP).
18. Le media indexing fonctionne bout en bout (ffprobe, hash, association match).

## Conditions d'echec

Traiter ces conditions comme des signaux d'arret ou de replanification.

1. Demarrer un portage big bang sans corpus fige, sans golden values et sans contrats API stables.
2. Melanger dans la meme iteration le portage Go, une refonte produit, un redesign de schema DuckDB et une refonte auth.
3. Reproduire les calculs Polars en Go sans preuve de parite chiffree sur les parcours critiques.
4. Basculer la production sans validation de parite, sans rollback et sans metriques de comparaison Python/Go.
5. Porter la sync avant d'avoir demontre que Go tient correctement les parcours read-only et les mutations simples.
6. Ignorer les contraintes de lock DuckDB ou les differences Windows/Linux jusqu'a la phase de bascule.
7. Sous-estimer la migration du device flow, du cache de tokens et des jobs longs.
8. Laisser Python et Go devenir deux backends canoniques concurrents pendant plusieurs mois.
9. Continuer a ajouter des features metier importantes dans Python sans planifier leur portage Go (voir "Strategie d'evolution du produit pendant le portage").
10. Declarer la migration terminee alors que les scripts critiques, la sync, les migrations de schema ou les procedures d'exploitation sont encore Python-only.
11. Confondre la liberte offerte par le worktree dedie avec une dispense de parite, de tests ou de gates d'integration.
12. Porter les algorithmes de scoring sans golden values et decouvrir des ecarts 6 mois apres en production.
13. Oublier le backfill bitmask et casser la detection de donnees manquantes.
14. Ne pas tester le driver DuckDB Go sur Windows et decouvrir un probleme de build CGo au moment du deploiement.
15. Ignorer le mode PvE et n'en prendre conscience qu'au moment du retrait de Python.
16. Lancer le portage Go avant d'avoir gele la reference contractuelle, les golden values et la matrice de couverture.

## Gates Go/No-Go recommandes

### Gate 1 - Faisabilite technique minimale

Passer ce gate seulement si :

1. DuckDB Go est valide sur les OS cibles.
2. Les lectures read-only ont une parite demonstrable.
3. L'equipe accepte une migration par strangler et non par big bang.

### Gate 2 - Viabilite produit

Passer ce gate seulement si :

1. Les parcours Career, History, Explorer et Match View tournent en Go avec parite acceptable.
2. La facade V7 consomme les endpoints Go sans changement de semantique.
3. Les ecarts restants sont documentes et volontaires.

### Gate 3 - Viabilite d'exploitation

Passer ce gate seulement si :

1. Settings, session, auth et jobs tournent sans bridge Python sur le chemin nominal.
2. La supervision et le rollback sont testes.
3. Les procedures de build, de deploy et de diagnostic sont reproductibles.

### Gate 4 - Extinction Python

Passer ce gate seulement si :

1. Sync, backfill, scripts critiques et migrations DuckDB ont leur equivalent Go.
2. Un cycle reel complet de sync et d'usage React a ete observe sans regression majeure.
3. Python n'est plus requis pour le runbook de prod.

## Recommandation finale

Si la question est "peut-on tout migrer vers Go ?", la reponse est oui.

Si la question est "faut-il lancer maintenant une reecriture complete de tout le Python ?", la reponse est non.

La bonne trajectoire est :

1. Stabiliser les contrats et la parite.
2. Introduire Go sur le read-only.
3. Porter ensuite auth/settings/jobs.
4. Porter enfin sync/backfill/outillage.
5. Eteindre Python a la fin, jamais au debut.

---

## Criteres d'abandon (kill switch)

Le plan a des gates Go/No-Go par phase. Il faut aussi des criteres d'abandon definitif. La migration Go doit etre **arretee** (pas ralentie — arretee) si :

1. **POC Sprint 0 echoue** : go-duckdb ne fonctionne pas sur Windows, ou CGo est trop fragile pour un build reproductible.
2. **Phase 1 depasse 3× l'estimation** : si 4-6 semaines prevues deviennent 15+ semaines sans parcours read-only fonctionnel.
3. **343 Industries change fondamentalement l'API Halo** : nouveau systeme d'auth, changement radical des endpoints, deprecation de l'API stats. Le portage Go cible un contrat API qui n'existe plus.
4. **DuckDB Go driver est abandonne** : le driver `go-duckdb` n'est plus maintenu et aucune alternative credible n'emerge.
5. **Le produit evolue plus vite que le portage** : si apres 3 mois les golden values les sont obsoletes parce que le produit Python a trop bouge.
6. **Fatigue / motivation** : pour un developpeur solo, 6-10 mois de portage sans feature visible est un risque reel. Si le portage devient une corvee, il vaut mieux rester sur Python+FastAPI qui fonctionne.

**Consequence d'un arret** : le worktree Go est abandonne ou archive sans merge. Il n'y a pas de "retour" a organiser apres coup puisque Python reste la baseline tant que Go n'a pas franchi les gates d'integration.

---

## Modele de deploiement cible

Le detail runtime, packaging, Docker, jobs persistants et contraintes ffprobe/ffmpeg est maintenu dans [OPS_COMPAT_CHECKLIST.md](OPS_COMPAT_CHECKLIST.md).

Principes retenus :

1. La cible preferee reste un binaire principal avec sous-commandes, sous reserve de validation packaging et CI.
2. "Zero Python" ne veut pas dire "zero binaire auxiliaire" si media indexing est conserve.
3. L'integration de la facade web peut etre embarquee ou servie a part ; ce point ne doit pas etre fige avant validation packaging.
4. Le packaging final doit rester compatible avec les contraintes d'exploitation reelles du repo, pas avec une promesse abstraite de binaire seul.

---

## Migration des donnees utilisateurs existants

Le runbook de compatibilite DuckDB, cache MSAL, sessions HTTP, configuration et transition utilisateur est maintenu dans [OPS_COMPAT_CHECKLIST.md](OPS_COMPAT_CHECKLIST.md).

Regle de base :

1. Les fichiers DuckDB existants doivent s'ouvrir sans migration implicite non voulue.
2. MSAL reste la cible canonique, avec support refresh tokens de compatibilite.
3. Les sessions FastAPI existantes peuvent etre invalidees au premier demarrage Go, mais cette decision doit rester explicite dans le runbook.

---

## Strategie d'evolution du produit pendant le portage

La version maintenue de cette discipline est dans [OPS_COMPAT_CHECKLIST.md](OPS_COMPAT_CHECKLIST.md).

Regles retenues :

1. Pas de freeze total du produit.
2. Pas de dette silencieuse de plusieurs semaines entre Python et Go sur OpenAPI, golden values ou schema DuckDB.
3. Chaque ecart volontaire doit etre visible dans la checklist ops avant de devenir acceptable.

---

## Gestion multi-joueurs en Go

Le detail du modele multi-joueurs, des pools par gamertag et des contraintes de shutdown est maintenu dans [OPS_COMPAT_CHECKLIST.md](OPS_COMPAT_CHECKLIST.md).

Regles retenues :

1. Le multi-joueur reste une exigence de premier ordre, pas un embellissement post-migration.
2. Les connexions player doivent rester isolees par chemin DB et par write lease.
3. L'ajout/suppression dynamique de joueurs via Setup doit rester explicable des la conception du pool.

---

## Opportunites specifiques a Go

Le portage Go n'est pas qu'une reecriture isometrique. Certaines choses deviennent possibles ou naturelles en Go :

1. **Distribution simplifiee** : un binaire principal, avec facade web embarquee ou distribuee a part selon le packaging retenu, peut reduire fortement la complexite d'installation, meme si certains outils auxiliaires comme ffprobe/ffmpeg peuvent rester necessaires selon le perimetre retenu.

2. **Concurrence native pour le backfill** : en Python, le backfill de plusieurs joueurs est sequentiel (un seul process, un seul writer). En Go, on peut lancer N goroutines de backfill en parallele (un writer par DB player, independants entre eux), ce qui divise le temps de backfill initial par N.

3. **SSE/WebSocket natif** : au lieu du polling `/jobs/{id}` toutes les 2 secondes, le serveur Go peut pousser le statut de sync en temps reel via SSE. Cela ameliore l'UX "sync en cours" sans complexite excessive.

4. **Detection automatique de data races** : `go test -race` detecte automatiquement les courses de donnees entre goroutines. Particulierement utile pour le pool de connexions et le write lease.

5. **Temps de demarrage** : Go demarre en ~50ms vs ~2-5s pour Python+uvicorn+imports. L'experience "lance l'app" est significativement meilleure.

6. **Cross-compilation triviale** : `GOOS=linux GOARCH=amd64 go build` produit un binaire Linux depuis Windows, sans VM ni Docker.

7. **Tests de performance integres** : `go test -bench` est natif. Les benchmarks de regression sur les requetes critiques s'integrent naturellement dans la CI.

**Attention** : ces opportunites ne justifient pas a elles seules la migration. Elles sont des bonus a capturer pendant le portage, pas des raisons de demarrer le portage.

---

## Adaptation pour developpeur solo

Ce plan est ecrit dans un style "programme d'entreprise". Pour un developpeur solo, certaines pratiques doivent etre simplifiees :

### Ce qui est surdimensionne et peut etre allegre

1. **Shadow mode complet** (Phase 1.4 du complement) : remplacer par une comparaison manuelle sur 10-20 requetes representant les golden values. Un `diff` JSON des reponses Python vs Go suffit. Pas besoin d'un proxy transparent.
2. **Soak test 2 semaines** (Phase 5.1) : remplacer par 2-3 cycles de sync reels sur les vrais joueurs. Si 3 syncs passent sans divergence, c'est suffisant.
3. **8 registres anti-oubli** : fusionner en une seule checklist dans un fichier `.ai/go_migration/GO_MIGRATION_CHECKLIST.md`. 8 fichiers separes c'est de la bureaucratie pour une personne.
4. **Rapport d'ecarts documente** : un commentaire dans le code ou un TODO suffit. Pas besoin d'un document formel par phase.

### Ce qui reste obligatoire meme en solo

1. **Golden values** : non negociable. Sans ca, on ne sait pas si le Go fait la meme chose.
2. **Tests de parite** : au minimum pour les 6 algorithmes metier (performance score, LUSR, sessions, citations, killer/victim, weapon parser).
3. **Sprint 0 POC** : 2 jours, non negociable. C'est le filtre le moins cher.
4. **Checklist pre-suppression Python** : avant de supprimer un module Python, verifier qu'il y a un equivalent Go teste.
5. **Discipline de branche** : le Go vit dans son worktree/branche. Pas de melange avec le dev Python courant.

---

# COMPLEMENT : COUVERTURE METIER, FONCTIONNELLE ET TECHNIQUE

> Sections ajoutees le 2026-04-13 pour completer le plan initial avec les aspects
> manquants : inventaire fonctionnel, algorithmes metier, catalogue SQL, risques
> identifies, erreurs corrigees.

---

## Errata et corrections sur le plan initial

### 1. Prerequis de lancement : corpus Go gele

Ce plan Go reste en attente tant que le corpus de reference Go n'est pas termine et gele.

Une fois ce prerequis valide, la reference contractuelle de depart du portage Go devient explicitement :

1. la facade V7 (consommateur web) ;
2. les contrats HTTP/OpenAPI exposes par le backend Python ;
3. les fixtures, golden values et suites de validation associees ;
4. le schema DuckDB et les comportements metier deja stabilises.

La migration Go remplace Python derriere cette reference. Elle ne reouvre pas un nouveau chantier frontend ou produit.

### 2. L'estimation d'effort est sous-evaluee

Le plan initial estime "4 a 7 mois pour 1 ingenieur backend". C'est optimiste compte tenu de :
- 94 champs SyncScope a reproduire fidelement
- 35 migrations DuckDB a rendre idempotentes en Go
- 11 mixins sync engine (certains avec de la logique film/bitstream complexe)
- 2 algorithmes de scoring non triviaux (performance relative percentile + TrueSkill2 adapte)
- ~120 arguments CLI a reconstituer
- 28+ endpoints API a porter avec middleware, injection de dependances, rate limiting
- ~550 fichiers de tests a traduire ou reconstituer
- Gestion des accents/i18n (14 langues dans metadata)

**Estimation revisee** : 6 a 10 mois pour 1 ingenieur backend senior a temps plein. 4 a 6 mois avec 2 ingenieurs. Le risque principal n'est pas la quantite de code mais la densite des invariants metier a verifier.

### 3. La section SPNKr est surdimensionnee

Le plan initial consacre ~20% de sa longueur a debattre de SPNKr. En realite, `src/ports/api.py` definit deja un `HaloAPIPort` abstrait. Le vrai travail n'est pas "migrer SPNKr" mais :
1. Implementer `HaloAPIPort` en Go (HTTP client direct vers les endpoints 343i)
2. Gerer les memes patterns : retry exponentiel, rate limit 60 req/min, circuit breaker sur 3 echecs consecutifs
3. Gerer le meme cycle de tokens : spartan_token + clearance_token avec refresh MSAL

SPNKr est une librairie Python tierce ; en Go, on la remplace par un client HTTP direct. La complexite reelle est dans l'auth MSAL et l'echange de tokens Halo, pas dans SPNKr elle-meme.

### 4. L'architecture hexagonale existante n'est pas mentionnee

`src/ports/` contient deja 2 interfaces abstraites :
- `HaloAPIPort` — contrat d'acces a l'API Halo
- `DataRepository` — contrat d'acces aux donnees

Ces interfaces sont la frontiere naturelle pour le portage Go. Le plan devrait s'appuyer dessus plutot que de re-decouvrir les frontieres.

### 5. Le write lease system n'est pas mentionne

`src/data/repositories/_write_lease.py` implemente une coordination process-locale par chemin DB pour eviter la collision entre ouvertures `read_write` et `read_only`. Le plan Go doit reproduire explicitement cette semantique avant tout durcissement : un seul writer logique par DB path, attente courte equivalente au Python (~5 s), warning explicite puis tentative d'ouverture quand meme si le lease persiste. Il ne faut pas presenter cela comme un verrou inter-process plus fort sans decision d'architecture separee.

### 6. Les vues materialisees ne sont pas couvertes

Le plan ne mentionne jamais les `mv_*` (materialized views dans stats.duckdb). En Python, elles sont recalculees apres chaque sync via `DROP VIEW IF EXISTS` + `CREATE VIEW`. En Go, le meme pattern idempotent doit etre reproduit.

### 7. L'archive Parquet n'est pas couverte

Le cold storage (`data/players/{gamertag}/archive/matches_*.parquet`) utilise Polars pour la lecture/ecriture. En Go, il faudra une librairie Parquet (apache/arrow-go ou parquet-go).

---

## Inventaire fonctionnel complet

### Surfaces produit (ce que l'utilisateur voit et utilise)

| # | Section | Sous-sections | Complexite | Priorite portage |
|---|---------|---------------|:----------:|:----------------:|
| 1 | **Accueil** | Hero card, Battle Pass, Challenges, Timeline, Media recents, Signaux | Moyenne | P2 |
| 2 | **Stats — Series** | 5 onglets (Win/Loss, KDA, Precision, Objectif, Forme) × 2 modes (Periode/Sessions) | Haute | P1 |
| 3 | **Stats — Historique** | Table 17 colonnes, pagination, filtres, tri, export CSV | Moyenne | P1 |
| 4 | **Stats — Match View** | 4 onglets (Scoreboard 19 col, Evenements, Statistiques, Details) | Haute | P1 |
| 5 | **Explorer** | Recherche fuzzy gamertags, filtres cascade, rencontres, Match View | Haute | P1 |
| 6 | **Profil — Carriere** | Rang actuel, historique rangs, stats saison, top rencontres | Moyenne | P1 |
| 7 | **Profil — Citations** | Commendations, medailles par categorie, frequence | Moyenne | P2 |
| 8 | **Escouade** | 13 sous-modules, 2 onglets (Synergies + Impact), selection coequipiers | Tres haute | P2 |
| 9 | **Synthese** | Solo vs Squad, heatmap, top semaine | Haute | P2 |
| 10 | **Medias** | Galerie, filtres, groupement, lightbox, like/flag | Moyenne | P3 |
| 11 | **Settings** | Langue, theme, media, Discord, backfill, about | Basse | P2 |
| 12 | **Setup/Auth** | Device Code Flow, ajout joueur, smoke test | Haute | P3 (derniere) |
| 13 | **Sync/Backfill** | Delta, full, ~120 options CLI, post-sync pipeline | Tres haute | P4 (derniere) |

### Filtres et resolution cascade

Le systeme de filtres est transversal a toutes les sections Stats/Explorer/Escouade :

```
Resolution cascade (POST /players/{slug}/filters/resolve) :

  1. Playlists disponibles (filtrees par joueur + periode)
  2. Modes de jeu (filtres par playlists selectionnees)
  3. Maps (filtrees par modes selectionnes)
  4. Paires map/mode (filtrees par maps selectionnees)

  Entree : FilterContextInput {
    date_range: [start, end] | null
    session_ids: list[str] | null
    playlist_ids: list[str] | null
    mode_ids: list[str] | null
    map_ids: list[str] | null
    outcome_filter: "all" | "wins" | "losses" | null
  }

  Sortie : FilterContextResolved {
    available_playlists: list[{id, name_fr, name_en, count}]
    available_modes: list[{id, name_fr, name_en, count}]
    available_maps: list[{id, name_fr, name_en, count}]
    match_ids: list[str]  # IDs resolus (intersection de tous les filtres)
    total_matches: int
  }
```

En Go, la resolution cascade doit reproduire exactement les memes ensembles de resultats. C'est un des premiers candidats pour un test de parite golden values.

### Sessions : deux modes de calcul

| Mode | Algorithme | Parametres |
|------|-----------|------------|
| **Gap-based** (legacy) | Coupe si > 120 min entre 2 matchs | `DEFAULT_SESSION_GAP_MINUTES = 120` |
| **Context-based** (avance) | Gap + changement de coequipiers + transition ranked/social | `SESSION_CUTOFF_HOUR = 8`, signature d'equipe |

Label de session : `"01/04/2026 14:30–15:45 (3)"` (date plage horaire + nombre de matchs).

Le mode avance a des regles subtiles :
- Si `friends_xuids` est fourni : seul un changement de **friend** declenche une nouvelle session (les randoms sont ignores)
- Si un joueur passe de ranked a social (ou inversement) : nouvelle session
- `SESSION_CUTOFF_HOUR = 8` : un match a 7:45 puis un a 8:15 = meme session, un a 8:05 puis un a 8:10 le lendemain = session differente si gap depasse

### Citations et commendations

Systeme de recompenses calculees post-match :

```
Pipeline :
  1. Charger evenements du match (kills, medailles, objectifs)
  2. Pour chaque regle dans citation_mappings (metadata.duckdb) :
     - Verifier la condition (sequence de medailles, seuils, etc.)
     - Si vraie : incrementer le compteur
  3. Stocker dans match_citations (match_id, citation_id, count)
  4. Agreger par session/saison via GROUP BY

Rules custom (src/analysis/citations/custom_rules.py) :
  - "Triple Kill" = 3 kills en < 5 secondes
  - "Clutch" = dernier joueur vivant + kill
  - "Flag Runner" = capture drapeau + N kills pendant le transport
  - Etc.
```

### Media indexing (bout en bout)

```
Pipeline :
  1. scripts/index_media.py scanne un dossier (videos .mp4/.mkv)
  2. Pour chaque fichier :
     a. Hash SHA-256 (deduplication)
     b. Extraction metadata via ffprobe (duree, resolution, codec)
     c. Matching gamertag + match via conventions de nommage ou timestamp
     d. Insert/update dans media_files (stats.duckdb)
     e. Association match dans media_match_associations
  3. Optionnel : generation thumbnails (scripts/generate_thumbnails.py)

Tables :
  - media_files : path, hash, duration_s, resolution, codec, created_at, size_bytes
  - media_match_associations : media_id, match_id, gamertag, confidence_score
```

En Go, il faut un equivalent de ffprobe. Options : appel externe a ffprobe (meme approche), ou librairie Go type `ffprobe-go`.

### PvE / Firefight (shared_pve.duckdb)

Le plan initial ne mentionne pas du tout le mode Firefight :

```
Table pve_match_stats :
  - match_id, xuid, gamertag
  - waves_completed, boss_kills
  - kills par type d'ennemi : grunt, elite, jackal, brute, hunter,
    skimmer, crawler, soldier, knight, warden
  - total_kills, deaths, damage_dealt

Pipeline sync :
  - Detecte le mode PvE via playlist_id
  - Extrait les stats PvE depuis l'API SPNKr
  - Insert dans shared_pve.duckdb (pas shared_matches_v2)
```

Ce module est plus simple que le PvP mais il ne faut pas l'oublier.

---

## Catalogue des algorithmes metier a porter

### Algorithme 1 — Performance Score (percentile relatif, v5)

**Fichier source** : `src/analysis/_performance_relative.py`

**Complexite** : Haute (statistique glissante + 10 metriques ponderees)

```
Entree :
  - row: dict (stats du match courant)
  - df_history: pl.DataFrame (50 derniers matchs du joueur)
  - had_bot_teammate: bool

Metriques et poids (somme = 1.0) :
  kpm (kills/min)                  : 0.17
  dpm_deaths (deaths/min, inverse) : 0.13
  apm (assists/min)                : 0.08
  kda ((K+A)/D)                    : 0.13
  accuracy (%)                     : 0.06
  pspm (score perso/min)           : 0.12
  dpm_damage (degats/min)          : 0.09
  rank_perf (rang vs attendu)      : 0.04
  kills_vs_expected (sigmoide)     : 0.10
  deaths_vs_expected (sigmoide inv): 0.08

Algorithme :
  1. Fenetre glissante de 50 matchs
  2. Pour chaque metrique : percentile = (nb matchs inferieurs / total) × 100
  3. Moyenne ponderee des percentiles (degradation gracieuse si metriques manquantes)
  4. Bonus coequipier bot (+5 si defaite, cap a 100)
  5. Score final : arrondi a 1 decimale

Seuils d'affichage :
  >= 75 : Excellent (vert)
  >= 60 : Bon
  >= 45 : Moyen
  >= 30 : Sous la moyenne (orange)
  < 30  : Catastrophique (rouge)
```

**Portage Go** : Statistique pure, pas de dependance externe. Implementable en Go natif. **Verifier avec golden values sur au moins 100 matchs.**

### Algorithme 2 — Skill Rating LUSR (TrueSkill2 adapte)

**Fichier source** : `src/analysis/skill_rating.py`

**Complexite** : Tres haute (mathematique sequentielle, etat persistent par playlist)

```
Parametres TrueSkill2 :
  INITIAL_MU    = 1500.0
  INITIAL_SIGMA = 350.0 (v5.3: etait 500?)
  MIN_SIGMA     = 60.0
  BETA          = 200.0   (bruit de performance)
  TAU           = 25.0    (derive naturelle par match)
  K_ELO         = 32.0    (amplitude de mise a jour)

Score composite [0,1] = moyenne ponderee :
  kills_vs_expected : 31%
  deaths_vs_expected: 28%
  win_factor        : 5%
  damage_efficiency : 23%
  accuracy_delta    : 13%

Estimation force adversaire :
  z_score = (kills_expected - match_avg_ke) / match_std_ke
  individual_mu = 1500 + 100 × z_score

Mise a jour TrueSkill :
  delta_mu = K_ELO × (composite - 0.5) × match_difficulty_weight
  new_mu = clamp(old_mu + delta_mu, borne_inf, borne_sup)

Inactivite :
  Si gap > 45 jours : sigma augmente (decay)
  INACTIVITY_SIGMA_PER_DAY, MAX_INACTIVITY_DAYS

Groupes de playlists isoles : ranked, arena, btb, tactical, social, fun

Tiers :
  Bronze  [1000-1200] 6 sous-niveaux
  Argent  [1200-1400] 6 sous-niveaux
  Or      [1400-1600] 6 sous-niveaux
  Platine [1600-1800] 6 sous-niveaux
  Diamant [1800-2000] 6 sous-niveaux
  Onyx    [2000+]     1 niveau
```

**Portage Go** : Algorithme mathematique pur sans dependance, mais la verification est critique. L'etat est sequentiel (chaque match depend du precedent). **Verifier avec golden values sur un historique complet de joueur (500+ matchs)**. Toute deviation > 0.1 point sur mu est un bug.

### Algorithme 3 — Killer/Victim Resolution

**Fichier source** : `src/analysis/killer_victim.py`

**Complexite** : Haute (resolution en 2 passes, tolerance temporelle ±5ms)

```
Passe 1 (Certaine) :
  Match 1-to-1 entre evenements kill et death dans une fenetre de ±5ms
  
Passe 2 (Estimee) :
  Cas ambigus resolus par :
  1. Frequence d'adversaire deja confirmee (passe 1)
  2. Meilleur rang dans le match (tiebreaker)
  3. Tri stable par XUID

Sortie :
  - certain_counts: {xuid: count}
  - estimated_counts: {xuid: count}
  - matrix: {killer_xuid: {victim_xuid: count}}
```

**Portage Go** : Algorithmique pure. Attention a la precision temporelle (nanosecondes dans les events).

### Algorithme 4 — Weapon Parsing (Film Bitstream)

**Fichier source** : `src/analysis/weapon_parser.py`

**Complexite** : Tres haute (parsing binaire de chunks film Halo)

```
Parsing :
  - Lecture de chunks binaires (film replay)
  - Detection des swaps d'arme via marqueurs dans le bitstream
  - Extraction : player_index (4 bits hauts octet 5), weapon_id (timeline)
  - Timestamp estime depuis frame_count

IDs speciaux :
  MELEE_WEAPON_ID    = 0xFF
  GRENADE_WEAPON_ID  = 0xFE
  VEHICLE_FILM_ID    = 2
  Autres : filmshell UBIGINT → weapon_labels dans metadata.duckdb

Reconciliation :
  weapon_kills.weapon_id peut differer du filmshell
  v_weapon_kills utilise COALESCE(reconciled_as, weapon_id) = effective_weapon_id
```

**Portage Go** : C'est le module le plus risque. Le parsing binaire est fragile et mal documente. **Recommandation : porter en dernier, conserver un bridge Python si necessaire.**

### Algorithme 5 — Participation Objective

**Fichier source** : `src/analysis/objective_participation.py`

```
Poids des assists :
  KILL_ASSIST    : 30
  MARK_ASSIST    : 20
  EMP_ASSIST     : 35
  DRIVER_ASSIST  : 25
  SENSOR_ASSIST  : 15
  FLAG_ASSIST    : 40

Ratios calcules :
  objective_ratio = objective_score / (objective_score + kill_score + assist_score)
  assist_ratio = assist_score / total_score
```

**Portage Go** : Simple, arithmetique pure.

### Algorithme 6 — Sessions (2 modes)

**Fichier source** : `src/analysis/sessions.py`

```
Mode 1 (gap-based) :
  Si gap entre 2 matchs > 120 min → nouvelle session

Mode 2 (context-based) :
  Gap + changement de signature d'equipe (friends) + transition ranked/social
  SESSION_CUTOFF_HOUR = 8

Generation du label :
  "01/04/2026 14:30–15:45 (3)" = date debut–fin (nombre matchs)
```

**Portage Go** : Moderement complexe a cause du mode context-based.

### Algorithme 7 — Spawn Detection

**Fichier source** : `src/analysis/spawn_detection.py`

**Complexite** : Haute (~700 LOC avec documentation exhaustive de la recherche)

Le module detecte les spawns et respawns des joueurs a partir des evenements film. Il analyse les patterns de position et timing pour identifier les zones de spawn et les spawn kills.

**Portage Go** : Algorithmique pure mais volumineuse. **Verifier avec golden values sur au moins 20 matchs couvrant des modes varies (CTF, Slayer, Strongholds).**

---

## Catalogue des requetes SQL critiques a porter

Les requetes suivantes sont actuellement executees via `DuckDBRepository` et ses mixins. Chacune doit avoir un equivalent Go exact.

### Requetes read-only haute frequence

| ID | Description | Tables/Vues | Complexite |
|----|-------------|-------------|:----------:|
| Q1 | Bootstrap joueur (rang, stats saison) | career_progression, match_participants | Basse |
| Q2 | Resolution gamertag | v_gamertag_lookup | Basse |
| Q3 | Filtres cascade | match_participants, match_registry, metadata i18n | Haute |
| Q4 | Match history pagine | v_match_full, match_participants | Moyenne |
| Q5 | Top coequipiers | match_participants × self-join (excl. bots) | Moyenne |
| Q6 | Matchs communs (explorer) | match_participants × 2 joueurs | Moyenne |
| Q7 | Career rank history | career_progression | Basse |
| Q8 | Top rencontres/antagonistes | v_killer_victim_full, match_participants | Haute |
| Q9 | Medailles match | medals_earned, metadata | Basse |
| Q10 | Events match (timeline) | highlight_events | Basse |
| Q11 | Media pagine | media_files, media_match_associations | Basse |
| Q12 | Weapon kills match | v_weapon_kills, weapon_labels | Moyenne |
| Q13 | Statistiques PvE | pve_match_stats | Basse |
| Q14 | Performance scores batch | player_match_enrichment | Basse |
| Q15 | LUSR history | match_skill_rank | Basse |
| Q16 | Battle Pass + Challenges | API live (pas DB) | — |

### Requetes read-write (sync/backfill)

| ID | Description | Tables | Risque |
|----|-------------|--------|:------:|
| W1 | Insert match registry | match_registry | Lock partagee |
| W2 | Insert participants | match_participants (31 colonnes) | Haute (volume) |
| W3 | Insert medailles | medals_earned | Moyenne |
| W4 | Insert highlight events | highlight_events | Moyenne |
| W5 | Insert weapon kills | weapon_kills | Haute (reconciliation) |
| W6 | Upsert xuid_aliases | xuid_aliases | Basse |
| W7 | Insert enrichment joueur | player_match_enrichment | Moyenne |
| W8 | Insert personal scores | personal_score_awards | Basse |
| W9 | Insert citations | match_citations | Basse |
| W10 | Insert career progression | career_progression | Basse |
| W11 | Insert skill rating | match_skill_rank | Basse |
| W12 | Insert PvE stats | pve_match_stats | Basse |
| W13 | Refresh materialized views | mv_* (DROP + CREATE) | Moyenne |
| W14 | Mise a jour backfill bitmask | sync_meta / match_registry | Haute (flags 22 bits) |
| W15 | Sauvegarde cache MSAL | sync_meta | Haute (auth) |

---

## Inventaire des dependances Python et equivalents Go

| Dependance Python | Fonction | Equivalent Go | Maturite |
|-------------------|----------|---------------|:--------:|
| `duckdb` | Moteur OLAP | `github.com/marcboeker/go-duckdb` | ⚠️ A valider (POC A) |
| `polars` | DataFrames/Series | SQL DuckDB natif + structs Go | N/A |
| `pydantic` v2 | Validation/serialisation | Structs Go + `go-playground/validator` | Haute |
| `fastapi` | Framework HTTP | `chi` ou `echo` ou `gin` | Haute |
| `uvicorn` | Serveur ASGI | Serveur net/http natif | Haute |
| `msal` | Auth Microsoft | Client MSAL Go ou HTTP direct | ⚠️ A valider (POC C) |
| `aiohttp` | Client HTTP async | `net/http` + goroutines | Haute |
| `pyarrow` | Parquet | `apache/arrow-go` | Haute |
| `plotly` | Graphiques | N/A (frontend React) | N/A |
| `streamlit` | UI | N/A (elimine par migration React) | N/A |
| `jwt` / `itsdangerous` | Sessions/tokens | `github.com/golang-jwt/jwt` | Haute |
| `ffprobe` (subprocess) | Metadata video | Meme approche (exec ffprobe) | Haute |
| `chromadb` (`src/ai/`) | RAG vectoriel | Hors scope (outillage dev) | N/A |

### Dependance critique : DuckDB Go driver

Le driver `go-duckdb` (github.com/marcboeker/go-duckdb) utilise CGo pour wrapper la lib C de DuckDB.

**Risques specifiques** :
- Cross-compilation Windows/Linux necessite les headers DuckDB C
- Comportement de lock : un seul writer par fichier (meme contrainte que Python)
- `ATTACH` d'une DB read-only depuis une connection read-write : a tester
- DuckDB 1.x vs 0.x : verifier la compatibilite du driver avec la version utilisee (1.4.4)
- Performance CGo : overhead d'appel FFI (normalement negligeable pour des requetes OLAP)

**Alternatives** :
- DuckDB CLI en subprocess (lent, fragile, dernier recours)
- DuckDB WASM (non pertinent pour un backend)

---

## Strategie i18n

L'application supporte 14 langues (BCP-47 : en-US, fr-FR, de-DE, es-ES, etc.) a travers :

1. `metadata.asset_translations` — traductions dynamiques des assets Halo (maps, playlists, variants)
2. `src/ui/translations.py` — traductions statiques de l'UI Streamlit
3. Frontend React — probablement i18next ou equivalent

**Pour le backend Go** :
- Les traductions dynamiques restent en DuckDB (requetes SQL avec jointure sur `asset_translations`)
- Les traductions statiques de l'UI ne concernent pas le backend (elles sont dans React)
- Le backend doit supporter le header `Accept-Language` et passer le `lang` aux requetes de resolution

**Aucune librairie i18n Go n'est necessaire** : les traductions sont dans la DB ou dans le frontend.

---

## Strategie de tests pour le portage Go

### Principe : parity-first

Chaque module Go doit prouver sa parite avec le Python correspondant **avant** de remplacer celui-ci. Les tests de parite sont plus importants que les tests unitaires classiques.

### Types de tests necessaires

| Type | Quoi | Comment |
|------|------|---------|
| **Golden values** | Comparer sortie Go vs sortie Python figee | Fixtures JSON dans `tests/fixtures/golden_values/` |
| **SQL parity** | Meme requete, meme resultat | Executer les memes SQL sur le meme corpus DuckDB |
| **Algorithm parity** | Performance score, LUSR, sessions | Entrees identiques → sorties identiques (tolerance : ε < 0.01) |
| **API contract** | Meme endpoint, meme payload JSON | Tests OpenAPI : schema + valeurs sur corpus |
| **E2E** | Frontend React appelle Go, meme comportement | Playwright existants (41 tests) |
| **Load** | Latence comparable | Benchmark p50/p95 sur les memes requetes |

### Golden values a constituer AVANT le portage

| Surface | Donnees figees necessaires |
|---------|---------------------------|
| Bootstrap | Rang, XP, gamertag, playlists disponibles |
| Filtres cascade | Pour chaque combinaison filtre → liste de match_ids |
| Career | Historique rangs, top matchs, rencontres |
| Match History | 3 pages de 50 matchs, tri par date desc |
| Explorer | Recherche "Spartan" → resultats fuzzy |
| Match View | Scoreboard 8 joueurs, events timeline, medals |
| Performance Score | 100 matchs avec score attendu (± 0.1) |
| LUSR | Historique 500 matchs avec mu/sigma attendus (± 0.1) |
| Sessions | Decoupage d'un mois en sessions (IDs + labels) |
| Escouade | Top 3 coequipiers + 13 sous-metriques |

### Consommation des tests existants

Les tests Python dans `tests/parity/` utilisent des fixtures dans `tests/fixtures/ref_player/`. Le Go doit :
1. Utiliser les **memes fichiers fixtures** (pas de reconstitution)
2. Lire les **memes golden values JSON**
3. Comparer avec les **memes tolerances**

Cela garantit une chaine de verificabilite tracable Python ↔ Go.

---

## Modele de concurrence Go vs Python

### Situation actuelle en Python

- Streamlit/FastAPI tourne en process principal
- Sync tourne soit dans le meme process (Streamlit), soit en process separe (CLI)
- Le write lease (`_write_lease.py`) protege les ecritures DuckDB
- Les lectures sont concurrentes (ATTACH read-only)
- Un sync bloquant peut provoquer des "database locked" cote lecture (retry loop)

### Modele cible en Go

```
                    ┌─────────────────────┐
                    │   go-api (net/http)  │
                    │   Request handlers   │
                    └────────┬────────────┘
                             │
                    ┌────────▼────────────┐
                    │   Connection Pool    │
                    │   Read-only pool     │ ← N connections paralleles
                    │   Write lease (1)    │ ← Semaphore global
                    └────────┬────────────┘
                             │
              ┌──────────────┼──────────────┐
              │              │              │
     ┌────────▼──┐   ┌──────▼──┐   ┌───────▼──┐
     │ metadata  │   │ shared  │   │ player/  │
     │ .duckdb   │   │ _v2     │   │ stats    │
     │ (RO)      │   │ (RO/RW) │   │ (RO/RW)  │
     └───────────┘   └─────────┘   └──────────┘
```

**Regles** :
1. Pool de connexions read-only : borne et explicite, pas illimite.
2. Write lease : 1 seul writer par DB a la fois (sync.Mutex par path, attente courte equivalente au Python plutot qu'un blocage long)
3. ATTACH shared_matches_v2 en read-only depuis chaque connexion player, une fois par connexion et pas par requete.
4. Si sync en cours : les lectures continuent mais peuvent voir un etat intermediaire (acceptable, meme comportement que Python)
5. Tous les opens `read_write` passent par un composant central unique ; pas d'ouverture writer ad hoc dans les handlers ou helpers.

### Gestion multi-joueurs

`db_profiles.json` peut contenir N joueurs. Le pool doit gerer des connexions par joueur :
- Un `map[gamertag]*PlayerPool` cree les pools a la demande (lazy init)
- Chaque pool player fait ATTACH shared une seule fois a l'init
- Les write leases sont independants par DB path (un sync sur joueur A ne bloque pas les lectures joueur B)
- Pool read-only par player borne a ~5 connexions
- Voir la section "Gestion multi-joueurs en Go" pour le detail

---

## Matrice detaillee Python → Go (mise a jour)
La version maintenue de cette matrice a ete sortie dans [MATRIX.md](MATRIX.md).

Regle de pilotage :

1. Aucun package, script, helper runtime ou surface hors scope ne doit etre touche sans statut explicite dans la matrice.
2. La strategie de portage du bitmask backfill est maintenue avec les valeurs exactes dans la matrice, pas dans ce document maitre.

## Backfill bitmask : strategie de portage

Voir [MATRIX.md](MATRIX.md) pour les valeurs exactes, les lacunes intentionnelles de numerotation et les surfaces outillage associees.

---

## Phases revisees avec detail fonctionnel

### Phase 0 — Cadrage, inventaire et corpus (2-3 semaines)

**Livrable 0.1 — Gel des contrats API** :
- Executer `openapi-typescript http://127.0.0.1:8000/api/openapi.json` et versionner
- Precondition : facade web cible gelee, golden values a jour, puis freeze du schema OpenAPI pour le lancement du chantier Go
- Documenter chaque endpoint : methode, path, payload in/out, middleware

**Livrable 0.2 — Corpus de golden values** :
- Etendre les 3 tests de parite existants a toutes les 16 surfaces read-only
- Les golden values doivent couvrir : valeurs numeriques, ordres de tri, pagination, cas limites (0 match, joueur sans medailles, match PvE)

**Livrable 0.3 — Baselines de performance** :
- Mesurer p50/p95 de chaque endpoint Python sur le corpus
- Sauvegarder comme reference : si Go est > 2× plus lent, c'est un bug

**Livrable 0.4 — POC DuckDB Go** :
- Valider go-duckdb sur Windows 10/11 et Linux
- Tester : open read-only, ATTACH, write lease, lock behavior
- Verifier explicitement : 2 writers meme path, 2 writers paths differents, et absence de migration implicite non voulue a l'ouverture
- Tester les types critiques : UBIGINT (weapon_id), TIMESTAMP WITH TIME ZONE, VARCHAR, BOOLEAN
- Tester COALESCE, CASE WHEN, GROUP BY, window functions

### Phase 1 — Socle Go read-only (4-6 semaines)

**Sprint 1.1 — Squelette HTTP** :
- `go-api/cmd/levelup-api/main.go` : server, config, healthcheck, request_id middleware
- Un mode de demo/test doit exister des le socle : fixtures stables, schemas maitrises et bypass auth si necessaire
- Routing : Chi ou Echo (pas Gin — trop opinionne pour ce cas)
- OpenAPI : generation depuis le meme schema que Python (`oapi-codegen` ou `ogen`)
- Middleware : CORS (memes origines), rate limit, logging structure (slog)
- Graceful shutdown : intercepter `os.Interrupt` / `SIGTERM`, `server.Shutdown(ctx)` avec timeout 15s, drainer les connexions DuckDB en cours

**Sprint 1.2 — Couche repository read-only** :
- `internal/platform/duckdb/pool.go` : connexion pool read-only + write lease
- `internal/platform/duckdb/queries/` : toutes les requetes Q1-Q16 du catalogue
- Tests : chaque requete comparee au golden value
- Regle d'implementation : aucun open `read_write` hors du composant central DuckDB

**Sprint 1.3 — Endpoints read-only** :
- Bootstrap (GET /bootstrap, GET /players)
- Filtres cascade (POST /players/{slug}/filters/resolve)
- Career (GET /pages/career, /career/top-matches, /career/encounters)
- Match History (POST /pages/match-history/query)
- Tests de parite endpoint par endpoint

**Sprint 1.4 — Validation de parite** :
- Executer les 10-20 golden values (requetes representant chaque endpoint) sur Go et Python
- Comparer les JSON de sortie via `diff` automatise (script de comparaison simple)
- Documenter les ecarts dans un fichier `parity_report.json`
- Gate : 0 ecart non justifie sur le corpus = passage a la phase 2
- Note solo : pas besoin d'un proxy transparent. Un script qui appelle les deux backends et compare suffit.

### Phase 2 — Parcours read-only complets (4-6 semaines)

**Sprint 2.1 — Explorer** :
- Recherche fuzzy gamertags (autocomplete depuis xuid_aliases)
- Rencontres croises (matchs communs entre 2 joueurs)
- Match View (4 onglets)
- Portage de la resolution killer/victim pour l'onglet Events

Note de perimetre : la facade V7 reste le consommateur de reference ; elle n'est pas reimplementee dans ce chantier.

**Sprint 2.2 — Stats/Series** :
- Port des 5 onglets × 2 modes (Periode/Sessions)
- Necessite le portage de sessions.py (2 modes de calcul)
- Necessite le portage de win_loss_service, timeseries_service
- Necessite le portage du performance score (algorithme 1)

**Sprint 2.3 — Accueil/Home** :
- Hero card (agglomeration career + last match)
- Battle Pass + Challenges : appels API Halo live (SPNKr equivalent)
  → Premiere utilisation du client HTTP Go vers 343i
- Timeline (5 derniers matchs)
- Media recents (3 derniers)

**Sprint 2.4 — Escouade + Synthese** :
- Top coequipiers (Q5)
- 13 sous-modules d'analyse — certains sont complexes (radar, first blood, clutch)
- Solo vs Squad breakdown
- Heatmap, top semaine

**Sprint 2.5 — Profil Citations + Medias** :
- Citations : portage du CitationEngine (regles custom)
- Medias : galerie paginee, filtres, groupement

**Gate phase 2** : 41 tests Playwright passent avec le backend Go.

### Phase 3 — Auth, session, settings, jobs (3-4 semaines)

**Sprint 3.1 — Modele de session** :
- Port des cookies httpOnly + JWT signing
- Zustand state : player context, session context
- POST /session/context

**Sprint 3.2 — Device Code Flow** :
- POC MSAL Go (ou portage HTTP direct du flux OAuth2 device code)
- Cible canonique : MSAL Go
- Compatibilite obligatoire : support refresh tokens (`SPNKR_OAUTH_REFRESH_TOKEN[_<GAMERTAG>]` + `oauth_refresh_token` dans `sync_meta`) tant que le runbook et les jobs en dependent
- Regle de priorite : MSAL par defaut, mais un refresh token deja exploitable est prioritaire sur l'ouverture d'un nouveau Device Code Flow interactif
- POST /auth/device-flow/start → user_code + verification_url
- GET /auth/device-flow/{attempt_id} → polling
- Echange access_token → spartan_token + clearance_token
- Persistance cache MSAL dans sync_meta (DuckDB write)
- Cas d'echec a couvrir : cache MSAL invalide, refresh token revoque, echec d'echange Halo, absence totale de chemin auth valide

**Sprint 3.3 — Settings** :
- GET /settings, PATCH /settings
- POST /settings/media/reset-index (destructif)
- POST /setup/players, POST /setup/smoke-test

**Sprint 3.4 — Jobs longs** :
- Modele : start → poll status → result, avec persistance hors memoire
- GET /jobs/{job_id}
- POST /sync/initial (retourne AsyncJobStatus)
- Redemarrage : `running` → `interrupted`, et exposition de `active_sync_job_id` dans le bootstrap
- Exclusivite stricte : une seule sync a la fois pour toute l'application ; si une sync est deja active, la suivante est refusee avec reference au job existant

**Gate phase 3** : onboarding complet fonctionne sans Python.

### Phase 4 — Sync, backfill, outillage (6-8 semaines)

C'est la phase la plus longue et la plus risquee.

**Prerequis avant le premier sync Go sur donnees reelles** : backup automatique de `shared_matches_v2.duckdb` et des DB player avant la premiere execution. Les fichiers DuckDB n'ont pas de WAL accessible externement ni de mecanisme de snapshot natif ; un sync Go defaillant peut corrompre les donnees sans rollback simple. Le script `scripts/backup_player.py` (ou son equivalent Go a ce stade) doit etre execute et verifie avant toute ecriture Go reelle.

**Sprint 4.1 — Moteur sync minimal** :
- Delta sync (fetch nouveaux matchs, insert dans shared + player)
- Reproduction des 11 mixins du SyncEngine en packages Go (ConnectionMixin, SchemaMixin, SharedWritesMixin, PerformanceMixin, SkillRatingMixin, CareerMixin, AggregatesMixin, WeaponKillsEngineMixin, MatchProcessingMixin, EnrichedWritesMixin, FanoutEnrichmentMixin)
- Write lease identique au Python

**Sprint 4.2 — Pipeline post-sync** :
- Performance score (algorithme 1)
- LUSR/TrueSkill (algorithme 2)
- Refresh materialized views
- Fanout enrichments

**Sprint 4.3 — Backfill complet** :
- Port de SyncScope (94 champs)
- Port du bitmask (22 bits, valeurs exactes)
- CLI : ~120 arguments
- `cmd/levelup-sync --backfill --player X --medals --force-medals`

**Sprint 4.4 — Migrations DuckDB** :
- Port du registre de migrations (35 steps)
- Idempotence garantie (schema_migrations table)
- Auto-apply au demarrage

**Sprint 4.5 — Weapon parsing** :
- Portage du parser de chunks film (binaire)
- Si trop risque : conserver un bridge Python pour cette seule fonction
- Reconciliation weapon_id

**Sprint 4.6 — PvE Firefight** :
- Sync pve_match_stats vers shared_pve.duckdb
- Detection mode PvE via playlist_id

**Sprint 4.7 — Scripts d'exploitation** :
- backup, restore, healthcheck, diagnose, check_env
- archive (Parquet read/write via arrow-go)
- index-media (ffprobe subprocess)
- populate-* (seed metadata)

**Sprint 4.8 — Notifications Discord** :
- Port des embeds Discord post-sync/backfill (tout est dans `src/utils/discord_notifier.py` — fichier unique)
- Thumbnail upload, anti-spam `discord_notified_at`
- Port notification nouvelle version
- Webhook URL configurable via `app_settings.json`
- Embeds bilingues (FR/EN) selon `discord_lang`

**Gate phase 4** : `cmd/levelup-sync --full --gamertag X --max-matches 500` produit un resultat identique a Python.

### Phase 5 — Bascule et extinction Python (2-4 semaines)

**Sprint 5.1 — Validation en conditions reelles** :
- Lancer 3 cycles de sync reels (delta) sur tous les joueurs configures avec le backend Go
- Comparer les resultats de sync avec le Python (match count, bitmask coherence, pas de regression)
- Utiliser l'app normalement pendant quelques jours (navigation, filtres, matchs)
- Si 3 syncs + utilisation normale sans divergence → bascule OK
- Note solo : pas besoin de 2 semaines formelles. Le critere est "3 cycles clean", pas un delai calendaire.

**Sprint 5.2 — Bascule progressive** :
- Feature flag par surface (cote reverse proxy ou deployment)
- Surface par surface : Career → History → Explorer → ...
- Reversibilite avant suppression finale : tant que Python n'est pas retire du trunk, la bascule ne doit pas etre irreversible

**Sprint 5.3 — Nettoyage** :
- Supprimer le code Python devenu mort
- Garder les tests de parite (ils deviennent les golden values de reference)
- Mettre a jour la documentation, Docker, CI, packaging

---

## Risques supplementaires identifies

### Risque 5 — Stabilite du schema DuckDB pendant le portage

Si le schema DuckDB evolue pendant le portage Go (nouvelle colonne, nouveau backfill flag), il faut maintenir les deux implementations en sync. **Regle** : les evolutions DuckDB restent possibles mais encadrees — toute migration de schema doit etre portee dans le Go dans la meme semaine. Pas de freeze total (le produit doit pouvoir evoluer), mais pas d'accumulation de dette non portee (voir "Strategie d'evolution du produit pendant le portage").

### Risque 6 — Derive du frontend React

Si le frontend evolue pendant le portage Go, les contrats API peuvent changer. **Regle** : les changements de contrat OpenAPI restent possibles si le Go est mis a jour dans la meme semaine. Changements client-only (cosmetic, routing, state React) ne necessitent pas de coordination. Le freeze total est remplace par un budget de dette controle (voir "Strategie d'evolution du produit pendant le portage").

### Risque 7 — CGo sur Windows

Le driver go-duckdb utilise CGo. Sur Windows, cela necessite un compilateur C (MinGW ou MSVC). Cela complique le build et le CI. **Mitigation** : tester le build Windows des le POC A, pas apres.

### Risque 8 — Mapping de types DuckDB ↔ Go

DuckDB a des types specifiques (UBIGINT, HUGEINT, TIMESTAMP WITH TIME ZONE, BLOB) dont le mapping vers Go n'est pas trivial. Le weapon_id est un UBIGINT (uint64 en Go), les timestamps necessitent time.Time avec timezone. **Mitigation** : creer un jeu de tests de types couvrant chaque type DuckDB utilise.

### Risque 9 — Latence des requetes ATTACH

En Python, chaque DuckDBRepository ATTACH `shared_matches_v2.duckdb` en read-only. En Go avec un pool de connexions, l'ATTACH doit se faire une seule fois par connexion (pas a chaque requete). Sinon la latence explose. **Mitigation** : ATTACH dans l'initialisation du pool, pas dans les handlers.

### Risque 10 — Perte de la logique "degradation gracieuse"

Beaucoup de code Python a des `if metric is None: skip` ou `try/except: return None`. Ce pattern de degradation gracieuse est critique pour l'UX (un match sans precision n'empeche pas d'afficher le reste). Le code Go doit reproduire cette robustesse, pas paniquer sur des `nil`.

---

## Decisions techniques a prendre AVANT la premiere ligne de Go (mises a jour)

1. **Driver DuckDB Go** : valider `go-duckdb` v1.x sur Windows et Linux (POC A)
2. **Framework HTTP** : Chi vs Echo (recommandation : Chi — plus proche de net/http standard)
3. **Generation OpenAPI** : `oapi-codegen` (types + server stubs depuis le schema existant)
4. **Validation payloads** : `go-playground/validator` ou validation manuelle
5. **Session/cookies** : `gorilla/sessions` ou `scs` (SCS prefere — plus moderne)
6. **JWT** : `golang-jwt/jwt` v5
7. **Logging** : `log/slog` (stdlib Go 1.21+)
8. **Config** : `envconfig` ou `viper` (envconfig prefere — plus simple)
9. **MSAL Go** : `github.com/AzureAD/microsoft-authentication-library-for-go` est un SDK Microsoft officiel avec support device code flow natif (`AcquireTokenByDeviceCode`). Le risque est plus faible que le plan initial le laissait entendre. Un test de ~50 lignes suffit a le valider dans le Sprint 0.
10. **Parquet** : `apache/arrow-go` pour le cold storage archive
11. **Tests** : `testing` stdlib + `testify/assert` pour les assertions
12. **CI** : GitHub Actions avec build matrix (Windows, Linux, amd64)
13. **Charting** : le backend retourne des series JSON, `react-plotly.js` cote frontend (aucun changement)

---

## Points de decision ouverts (a trancher avant le lancement)

| # | Question | Options | Recommandation | Impact |
|---|----------|---------|----------------|--------|
| D1 | Monorepo ou polyrepo ? | Go dans le meme repo vs repo separe | Monorepo (memes fixtures, memes golden values) | Structure repo |
| D2 | Un binaire ou plusieurs ? | `levelup-api` + `levelup-sync` + `levelup-tools` vs tout-en-un | Binaire unique avec sous-commandes (voir "Modele de deploiement") | Packaging |
| D3 | Generation OpenAPI : code-first ou schema-first ? | Generer le schema depuis le Go vs utiliser le schema Python existant | Schema-first (garantit la parite contractuelle) | DX |
| D4 | Sessions : filesystem ou Redis ? | Garder le filesystem actuel vs introduire Redis | Filesystem (pas de nouvelle dependance pour un outil solo) | Infra |
| D5 | Cache MSAL : bridge Python temporaire ou portage direct ? | Garder un microservice Python pour l'auth vs tout porter | MSAL canonique + support refresh tokens (env + sync_meta) | Complexite |
| D6 | Weapon parser : porter ou bridge ? | Reecrire le parser binaire en Go vs subprocess Python | Bridge temporaire puis portage (module le plus risque) | Risque |
| D7 | CI : build DuckDB depuis les sources ou pre-built ? | Compiler libduckdb.a vs telecharger les releases | Pre-built (plus rapide, moins fragile) | CI |
| D8 | Strategie pool `database/sql` + ATTACH ? | `sql.Conn` pinee par gamertag vs `ConnInitFunc` vs pool custom hors `database/sql` | A valider dans Sprint 0 — impacte toute l'architecture du pool DuckDB | Architecture |
| D9 | Mecanisme de feature flags pour la bascule ? | Variable d'env, champ `app_settings.json`, ou header HTTP | Variable d'env + champ `app_settings.json` (simple, pas de dependance) | Bascule |
| D10 | Observabilite : outils de monitoring ? | Prometheus + `/metrics`, ou logs structures seuls | Logs structures `slog` + endpoint `/debug/stats` (suffisant pour solo dev) | Ops |

---

## Checklist pre-lancement (a valider AVANT d'ecrire du Go)

La checklist exhaustive et maintenue est desormais dans [OPS_COMPAT_CHECKLIST.md](OPS_COMPAT_CHECKLIST.md).

Minimum incompressible :

- [ ] Schema OpenAPI versionne, golden values a jour et perimetre contractuel de depart explicitement nomme
- [ ] POC DuckDB Go valide sur Windows et Linux
- [ ] Version DuckDB go-duckdb compatible avec les fichiers Python 1.4.4 (pas de migration implicite)
- [ ] Strategie pool `database/sql` + ATTACH validee (D8)
- [ ] Toolchain CGo Windows documente et reproductible en CI
- [ ] MSAL Go valide, avec strategie explicite de support refresh tokens
- [ ] Strategie de compatibilite cache MSAL Python → Go documentee
- [ ] Mode de demo/test defini des le socle
- [ ] Modele de jobs persistants defini avant tout portage de sync/setup
- [ ] Mecanisme de feature flags choisi (D9)
- [ ] Matrice et checklist ops initialisees avant toute suppression de code Python
