> [!WARNING]
> DOCUMENT DE CADRAGE — a verrouiller avant execution.
> Les decisions structurantes deja tranchees dans ce corpus sont la source de verite. Restent a prouver techniquement le Sprint 0, la strategie de pool DuckDB et les derniers choix de bascule/outillage.

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
> - Prerequis dur : les decisions structurantes, la facade web cible, les contrats API P0/P1 et la matrice de couverture doivent etre geles avant l'ouverture effective du Sprint 0 Go. Les golden values completes doivent etre consolidees avant l'ouverture de la Phase 1.
> - La reference contractuelle de depart du portage Go est le produit actuel gele : facade web, contrats API, fixtures et suites de validation associees.
> - Aucun agent ne doit utiliser ce document seul.

## Lecture obligatoire avant toute action

1. [PLAN_MIGRATION_PYTHON_TO_GO_V2.md](PLAN_MIGRATION_PYTHON_TO_GO_V2.md) — trajectoire, phases, gates, risques, decisions.
2. [SPRINT_ROADMAP.md](SPRINT_ROADMAP.md) — decoupage lineaire de A a Z en 29 sprints (S00–S28), pour repartir les taches et suivre l'avancement.
3. [MATRIX.md](MATRIX.md) — couverture package/script/commande/bitmask, surfaces hors scope et statut de chaque zone.
4. [OPS_COMPAT_CHECKLIST.md](OPS_COMPAT_CHECKLIST.md) — compat auth/jobs/mode de test, exploitation, packaging et migration utilisateur.
5. [ZERO_PYTHON_STRATEGY.md](ZERO_PYTHON_STRATEGY.md) — objectif zero Python, inventaire module par module, strategie d'extinction SPNKr.
6. [GO_MIGRATION_CHECKLIST.md](GO_MIGRATION_CHECKLIST.md) — suivi vivant du chantier, statuts d'avancement, preuves attendues et blocages.
7. [GO_ARCHITECTURE_RULES.md](GO_ARCHITECTURE_RULES.md) — architecture hexagonale formelle, interfaces Go obligatoires, direction des dépendances, pare-feu linter `depguard`, mapping Python ports → Go interfaces.
8. Regle simple : si une surface n'apparait ni dans la matrice ni dans la checklist ops, elle est consideree comme non couverte.

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

1. [PLAN_MIGRATION_PYTHON_TO_GO_V2.md](PLAN_MIGRATION_PYTHON_TO_GO_V2.md) fixe l'ordre macro, les phases, les gates et les conditions d'ouverture/fermeture.
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
      levelup/      # binaire unique : api, sync, backfill, tools
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
      halo/
        adapter/
        canonical/
        titles/
          haloinfinite/
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

> [!IMPORTANT]
> Les règles d'architecture logicielle formelles (couches, interfaces, DI, linter) sont dans
> [GO_ARCHITECTURE_RULES.md](GO_ARCHITECTURE_RULES.md). Ce qui suit concerne la conception produit.

1. Le service Go expose des contrats de page stables, pas des details internes de tables.
2. La facade API reste orientee produit : bootstrap, history, explorer, match view, settings, sync. Elle ne reflète jamais directement les endpoints d'un titre Halo particulier.
3. L'integration Halo suit une architecture a 2 niveaux : un socle provider generique (transport, auth, rate limiting, erreurs, registre d'endpoints), puis un adaptateur par titre.
4. Chaque adaptateur de titre mappe les payloads natifs vers un modele canonique LevelUp avant toute logique metier ou exposition HTTP.
5. Les zones specifiques au jeu doivent rester isolees : auth Waypoint/XSTS, refdata, assets discovery/economy, films, skill, labels et URLs Waypoint.
6. Les calculs critiques restent cote backend, jamais reimplementes dans le frontend.
7. **Charting decouple du renderer** — Les ~80 fonctions `plot_*` de `src/visualization/` (47 fichiers, ~12K LOC) sont portees dans `domain/chart/` (logique pure, 0 import Plotly). Les builders produisent un `ChartPayload` **renderer-agnostic** (series + metadata + annotations). Un `adapter/plotly/` convertit ce payload en `PlotlyFigurePayload` JSON pour le frontend React actuel (`react-plotly.js`). Ce decouplage permet de migrer le frontend vers Recharts/Nivo/ECharts **sans modifier le backend** — il suffit d'un nouvel adapter. Voir [GO_ARCHITECTURE_RULES.md §11](GO_ARCHITECTURE_RULES.md).
8. Les jobs longs doivent avoir un modele explicite : start, status, cancel si necessaire, result, warnings, erreurs.
9. Les migrations DuckDB doivent rester idempotentes et automatisees.
10. Graceful shutdown obligatoire : intercepter `SIGTERM`/`SIGINT` (ou `os.Interrupt` sur Windows), drainer les requetes HTTP en cours, fermer proprement les connexions DuckDB, et ne jamais interrompre un sync en pleine ecriture (attendre le commit ou rollback).
11. Notifications Discord : le backend Go doit etre capable d'envoyer des embeds Discord (webhook) apres sync/backfill, exactement comme le Python actuel (`src/utils/discord_notifier.py`). Inclure : embeds bilingues, thumbnail upload, anti-spam via `discord_notified_at`, notifications new version.

## Preparation documentaire immediate pour l'API multi-titre

Avant toute implementation Go reelle sur la couche Halo :

1. figer un modele canonique Halo produit (identite, history, match, career, assets, films, erreurs) ;
2. figer une matrice de capabilities par titre et par surface produit ;
3. documenter les zones specifiques au jeu a isoler ;
4. documenter la politique de degradation si une surface est absente ou partielle sur un titre donne.

### Portee acceptable si l'on ne connait que Halo Infinite

Oui, les points 2 et 3 restent pertinents meme si l'on ne dispose aujourd'hui d'informations solides que pour Halo Infinite, mais ils doivent etre cadres de facon non speculative.

1. La matrice de capabilities ne doit pas inventer des lignes pour d'autres titres. Dans l'immediat, elle documente uniquement `halo_infinite` et les surfaces produit que nous savons reellement alimenter.
2. Son role n'est pas de predire le prochain Halo ; son role est d'eviter que le produit Go se couple implicitement a des specificites Halo Infinite non nommees.
3. La premiere version acceptable est donc une capability map mono-titre, extensible plus tard, avec des statuts simples : supporte, degrade, non expose, hors scope.
4. Toute case inconnue pour un autre titre reste absente du document ; on n'introduit aucun faux contrat juste pour "faire multi-titre".

### Pourquoi documenter aussi l'exposition via bootstrap

La definition documentaire d'une exposition minimale des capabilities dans le bootstrap reste utile meme en mono-titre.

1. Elle force a distinguer ce qui releve du contrat produit de ce qui releve du provider Halo Infinite.
2. Elle permet d'organiser proprement la degradation pendant la migration Go : une surface peut etre annoncee comme absente ou partielle sans casser le reste du produit.
3. Elle evite de hardcoder dans le frontend ou dans les handlers l'hypothese "tout Halo Infinite est toujours disponible partout".
4. Elle ne doit pas exposer les details internes des endpoints 343i ; elle expose seulement le titre courant, le provider courant et les capabilities produit utiles au consommateur.

### Decision de cadrage pour cette migration

Dans ce chantier Go, on documente donc :

1. un modele canonique Halo ;
2. une capability map initiale limitee a Halo Infinite ;
3. une forme de bootstrap cible capable d'annoncer le titre courant et les capabilities produit utiles ;
4. sans speculation sur d'autres titres tant qu'aucune information fiable n'existe.

Ces deux livrables sont desormais materialises dans :

1. [HALO_CANONICAL_MODEL.md](HALO_CANONICAL_MODEL.md)
2. [HALO_INFINITE_CAPABILITY_MAP.md](HALO_INFINITE_CAPABILITY_MAP.md)

Leur declinaison de travail est maintenant detaillee dans :

1. [HALO_BOOTSTRAP_CONTRACT.md](HALO_BOOTSTRAP_CONTRACT.md)
2. [HALO_GO_TYPE_BLUEPRINT.md](HALO_GO_TYPE_BLUEPRINT.md)
3. [HALO_INFINITE_CANONICAL_MAPPING.md](HALO_INFINITE_CANONICAL_MAPPING.md)
4. [HALO_PRODUCT_CONTRACT_ADAPTERS.md](HALO_PRODUCT_CONTRACT_ADAPTERS.md)
 5. [HALO_PROVIDER_ERROR_TAXONOMY.md](HALO_PROVIDER_ERROR_TAXONOMY.md)
 6. [OPENAPI_MVP_P0_P1.md](OPENAPI_MVP_P0_P1.md)

Avec ces deux derniers documents, le prerequis 0 documentaire est considere comme suffisant.

La regle de travail devient alors :

1. ne plus ouvrir de nouveau cycle de documentation generale avant code ;
2. lancer le Sprint 0 ;
3. ne documenter ensuite que les ecarts reels rencontres pendant les spikes, la parite ou l'implementation.

## Decisions techniques a prendre avant la premiere ligne de Go

1. ~~Valider un driver DuckDB compatible Windows/Linux et le comportement de lock associe.~~ **RÉSOLU** — voir D1 ci-dessous.
2. ~~Choisir le socle HTTP Go et la strategie de validation des payloads.~~ **RÉSOLU** — voir D2 ci-dessous.
3. ~~Choisir la forme des contrats de charting~~ : **RÉSOLU** — architecture découplée : `domain/chart/` produit des `ChartPayload` renderer-agnostic (séries + metadata + annotations), `adapter/plotly/` convertit en `PlotlyFigurePayload` pour le frontend actuel (`react-plotly.js`). Ce découplage permet de migrer le frontend vers Recharts/Nivo sans modifier le backend (nouvel adapter = suffisant). Voir [GO_ARCHITECTURE_RULES.md §11](GO_ARCHITECTURE_RULES.md).
4. ~~Choisir la strategie de session et de cookies pour remplacer le modele actuel.~~ **RÉSOLU** — voir D4 ci-dessous.
5. ~~Choisir la strategie de logging, trace, request_id et observabilite.~~ **RÉSOLU** — voir D5 ci-dessous.
6. ~~Le cache de tokens Halo est porte nativement en Go des la phase auth (pas de bridge Python).~~ **RÉSOLU** — voir D6 ci-dessous.
7. ~~Decider la strategie de generation OpenAPI et de types frontend.~~ **RÉSOLU** — voir D7 ci-dessous.

---

### D1 — Driver DuckDB Go et stratégie de concurrence

**Choix : `github.com/duckdb/duckdb-go` v2** (anciennement `marcboeker/go-duckdb`, migré officiellement le 20/10/2025, licence MIT inchangée).

| Critère | Décision |
|---------|----------|
| **Package** | `github.com/duckdb/duckdb-go` v2.4+ (DuckDB 1.4.x, parité avec Python prod 1.4.4) |
| **Interface** | `database/sql` standard — `sql.Open("duckdb", dsn)` |
| **CGO** | Requis. Libs statiques pré-compilées fournies pour Linux amd64/arm64, Windows amd64 |
| **Windows build** | MinGW gcc via msys64 (`pacman -S mingw-w64-ucrt-x86_64-gcc`), `CGO_ENABLED=1` |
| **Arrow** | Opt-in via `-tags=duckdb_arrow` (pas nécessaire pour MVP) |

**Stratégie de connexion et ATTACH :**

```text
                    ┌─────────────────────┐
                    │   NewConnector(dsn)  │
                    │  connInitFn: ATTACH  │
                    │  shared + metadata   │
                    └──────────┬──────────┘
                               │
              ┌────────────────┼────────────────┐
              │                │                │
        ┌─────▼──────┐  ┌─────▼──────┐  ┌──────▼─────┐
        │  Read Pool  │  │  Read Pool  │  │   Writer   │
        │ (sql.DB,    │  │ (sql.DB,    │  │ sync.Mutex │
        │  read_only) │  │  read_only) │  │  1 conn    │
        └────────────┘  └────────────┘  └────────────┘
```

- **Lecture** : `NewConnector(dsn + "?access_mode=read_only", initFn)` → `sql.OpenDB(connector)`. Le pool `sql.DB` gère N connexions read-only en parallèle. Le `initFn` exécute `ATTACH 'shared_matches_v2.duckdb' AS shared (READ_ONLY)` et `ATTACH 'metadata.duckdb' AS meta (READ_ONLY)` au boot de chaque connexion.
- **Écriture** : Pattern write-lease reproduisant `_write_lease.py` — un `sync.Mutex` global par DB protège un writer unique. `db.Conn(ctx)` donne une `*sql.Conn` pinnée pour la durée de la transaction. Pas de pool d'écriture (DuckDB = single writer de toute façon).
- **Idle connections** : `db.SetMaxIdleConns(0)` pour éviter que les tables temporaires persistent (recommandation upstream).
- **Close** : Appeler `db.Close()` (ou `connector.Close()`) à l'arrêt pour flusher le WAL.

**Risques identifiés :**
- CGO impose un toolchain C sur le CI (résolu : images Docker avec gcc, GitHub Actions `apt-get install build-essential`).
- Le driver est un primary DuckDB client maintenu par l'équipe DuckDB elle-même (faible risque de stale).
- La version 2 introduit des breaking changes (Arrow opt-in, JSON scanning) — à intégrer dès le Sprint 0.

---

### D2 — Socle HTTP Go et validation des payloads

**Choix : `chi` v5 (routeur) + `go-playground/validator` v10 (validation struct tags)**

| Critère | Décision | Justification |
|---------|----------|---------------|
| **Routeur** | `github.com/go-chi/chi/v5` | Léger (~600 LOC), 100% compatible `net/http`, middleware chain idiomatique, utilisé en production par Cloudflare/Heroku. Correspond aux 16 routers FastAPI existants. |
| **Validation** | `github.com/go-playground/validator/v10` | Struct tags (`validate:"required,min=1"`), custom validators, équivalent direct des validators Pydantic. |
| **Sérialisation JSON** | `encoding/json` stdlib (+ migration vers `go-json` si benchmark le justifie) | Suffisant pour le volume de données LevelUp. |
| **Erreurs HTTP** | Struct `ApiError` custom → JSON `ApiErrorSchema` | Reproduction exacte du contrat Python : `{code, message, retryable, details, field_errors}` |

**Correspondance FastAPI → chi :**

| FastAPI | chi Go |
|---------|--------|
| `app = FastAPI()` | `r := chi.NewRouter()` |
| `@app.get("/api/v1/...")` | `r.Get("/api/v1/...", handler)` |
| `app.add_middleware(CORSMiddleware)` | `r.Use(cors.Handler(cors.Options{...}))` |
| `@app.on_event("startup")` | Code dans `main()` avant `http.ListenAndServe` |
| `Depends(get_session)` | Middleware chi injecte session dans `context.Context` |
| `HTTPException(status_code=...)` | `render.Status(r, code); render.JSON(w, r, apiErr)` |

**Middleware stack (ordre = celui de Python) :**

1. `middleware.RequestID` (chi built-in — UUID par requête, header X-Request-ID)
2. `middleware.RealIP` (proxy-aware)
3. `middleware.Logger` (structuré via slog — cf. D5)
4. `cors.Handler(cors.Options{...})` (mêmes origins que Python : localhost:5173)
5. `SessionMiddleware` (custom — cf. D4)
6. `CSRFMiddleware` (validation Origin header — reproduction de `require_same_origin()`)
7. `middleware.Recoverer` (panic → 500 JSON)

**Pourquoi pas les alternatives :**
- **Stdlib seul** (`net/http` 1.22+) : le pattern routing est suffisant mais le middleware chaining demande du boilerplate. Avec 16+ routers et 7 middlewares, chi apporte un gain net.
- **Echo/Gin** : trop opinionés, créent leur propre `Context` type — friction avec l'architecture hexagonale (nos handlers prennent `http.ResponseWriter, *http.Request`).
- **Fiber** : non compatible `net/http` (basé sur fasthttp) — interdit par §1 architecture.
- **Huma** : intéressant (OpenAPI auto) mais trop jeune et couple routeur+validation+OpenAPI — on préfère composer.

---

### D4 — Sessions et cookies

**Choix : sessions fichiers JSON + signature HMAC-SHA256 du cookie**

| Critère | Décision |
|---------|----------|
| **Stockage** | Fichiers JSON dans `data/sessions/` (idem Python) |
| **Cookie** | `levelup_session`, signé HMAC-SHA256 (stdlib `crypto/hmac` + `crypto/sha256`) |
| **Format cookie** | `base64url(sessionID \| timestamp \| hmac)` — équivalent fonctionnel de `itsdangerous.URLSafeTimedSerializer` |
| **Attributs cookie** | `HttpOnly=true`, `SameSite=Lax`, `Secure=production_only`, `Max-Age=604800` (7j), `Path=/` |
| **CSRF** | Validation header `Origin` (reproduction exacte de `require_same_origin()`) |
| **Secret** | `LEVELUP_SESSION_SECRET` env var (fail-fast si absent en production) |
| **TTL** | 7 jours (configurable via `LEVELUP_SESSION_TTL`) |
| **Purge** | Au startup + goroutine `time.Ticker` toutes les 1 h (corrige bug Python : purge uniquement au startup) |
| **Écriture atomique** | `os.CreateTemp()` → `os.Rename()` (corrige bug Python : `write_text()` direct = JSON corrompu si crash mid-write) |
| **Verrouillage** | `sync.RWMutex` par session ID dans une `sync.Map` (corrige bug Python : read-modify-write sans lock = last-write-wins) |

**3 correctifs vs implémentation Python actuelle :**

| Bug Python (`apps/api/app/deps/auth.py`) | Impact | Correctif Go |
|-------------------------------------------|--------|-------------|
| `path.write_text(json.dumps(...))` — écriture directe | Crash mid-write → JSON corrompu | `os.CreateTemp(dir)` + `f.Write(data)` + `f.Close()` + `os.Rename(tmp, target)` — atomique sur POSIX et Windows (même volume) |
| `load() → touch() → save()` sans lock | 2 requêtes concurrentes → la 2e écrase les changements de la 1re | `sessionLocks sync.Map` de `*sync.RWMutex` par session ID — `RLock` pour les lectures, `Lock` pour les écritures |
| Purge uniquement dans `lifespan startup` | Fichiers expirés s'accumulent indéfiniment entre redémarrages | Goroutine `purgeLoop()` avec `time.NewTicker(1 * time.Hour)` — itère `os.ReadDir`, parse + supprime si expiré |

**Struct SessionData Go :**

```go
// internal/domain/session.go
type SessionData struct {
    ID                  string    `json:"session_id"`
    CreatedAt           time.Time `json:"created_at"`
    LastSeenAt          time.Time `json:"last_seen_at"`
    CurrentPlayerSlug   string    `json:"current_player_slug"`
    Locale              string    `json:"locale"`
    HintsVisible        bool      `json:"hints_visible"`
    AuthReady           bool      `json:"auth_ready"`
    LinkedHaloIdentity  string    `json:"linked_halo_identity"`
    ActiveSyncJobID     string    `json:"active_sync_job_id,omitempty"`
}
```

**Écriture atomique — pattern :**

```go
// internal/platform/session/filestore.go
func (s *FileStore) atomicWrite(path string, data []byte) error {
    tmp, err := os.CreateTemp(filepath.Dir(path), ".session-*.tmp")
    if err != nil { return err }
    if _, err := tmp.Write(data); err != nil {
        tmp.Close(); os.Remove(tmp.Name()); return err
    }
    if err := tmp.Close(); err != nil {
        os.Remove(tmp.Name()); return err
    }
    return os.Rename(tmp.Name(), path)
}
```

**D4-bis — Élimination de la fragmentation session/settings (dette Python)**

L'implémentation Python souffre d'une fragmentation majeure : la même notion est stockée dans jusqu'à **5 endroits** avec des noms différents et aucune source de vérité unique.

**Diagnostic Python — état actuel :**

| Concept | `app_settings.json` | `SessionData` (JSON files) | `ui_prefs.json` | `st.session_state` | `BootstrapResponse` |
|---------|:---:|:---:|:---:|:---:|:---:|
| Langue | `lang` | `locale` | `lang` | `lang` | `locale` + `settings_excerpt.lang` |
| Hints | — | `hints_visible` | `show_hints` | `_hints_visible` | `hints_visible_default` |
| Joueur courant | — | `current_player_slug` | `last_gamertag` | — | `current_player` |

Conséquences : `POST /session/context` écrit `locale` dans la session mais ne touche pas `app_settings.json`. `PATCH /settings` écrit `lang` dans `app_settings.json` mais ne touche pas la session. Le bootstrap renvoie les deux, potentiellement divergents. Le frontend devine.

**Règle Go — source de vérité unique :**

| Concept | Source de vérité Go | Anciens emplacements Python éliminés |
|---------|--------------------|--------------------------------------|
| **Langue** (`locale`) | `SessionData.Locale` | `app_settings.lang`, `ui_prefs.lang`, `st.session_state["lang"]` |
| **Hints** (`hints_visible`) | `SessionData.HintsVisible` | `ui_prefs.show_hints`, `st.session_state["_hints_visible"]` |
| **Joueur courant** | `SessionData.CurrentPlayerSlug` | `ui_prefs.last_gamertag`, `db_profiles.json` default |
| **Timezone** | `app_settings.json` (seul) | Reste inchangé — config serveur, pas préférence utilisateur |
| **Feature flags** (`media_enabled`, etc.) | `app_settings.json` (seul) | Reste inchangé — config déploiement |

**Principe : la session porte les préférences utilisateur, `app_settings.json` porte la config serveur. Pas de chevauchement.**

**Nommage unifié Go :** le terme est `locale` partout (pas `lang`). Le champ s'appelle `Locale` dans `SessionData`, `locale` dans l'API JSON, `locale` dans le Zustand store React. Zéro alias.

**API simplifiée :**

```
GET  /api/bootstrap   → { session: { locale, hints_visible, current_player_slug, ... }, config: { timezone, features, ... } }
POST /api/session      → met à jour SessionData (locale, hints, player)
PUT  /api/settings     → met à jour app_settings.json (timezone, features)
```

Plus de `settings_excerpt.lang` qui doublonne `session.locale`. Plus de `hints_visible_default` séparé de `hints_visible`. Un seul endroit par concept.

**Fichier `ui_prefs.json` : supprimé.** Tout ce qu'il contenait (`lang`, `show_hints`, `last_gamertag`) migre dans `SessionData`. Streamlit est supprimé dans la migration Go, donc `st.session_state` disparaît de facto.

**Pourquoi pas JWT :** les données de session changent fréquemment (`current_player_slug`, `locale`, `active_sync_job_id`) et nécessitent une révocation côté serveur — JWT ne convient pas.

**Pourquoi pas Redis/BoltDB :** le déploiement est single-instance, les sessions sont légères (<1 KB), le filesystem suffit. L'interface `port/SessionStore` permettra de swapper l'implémentation (Redis, DuckDB) si nécessaire.

---

### D5 — Logging, tracing, request_id et observabilité

**Choix : `log/slog` (stdlib Go 1.21+)**

| Critère | Décision |
|---------|----------|
| **Logger** | `log/slog` (stdlib) — équivalent direct de structlog |
| **Handler prod** | `slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})` |
| **Handler dev** | `slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})` |
| **Request ID** | Middleware chi `middleware.RequestID` (UUID v4) → stocké dans `context.Context` via `slog.With("request_id", id)` |
| **Header** | `X-Request-ID` propagé en réponse (idem Python `RequestIdMiddleware`) |
| **Config** | `LEVELUP_LOG_LEVEL` (INFO par défaut), `LEVELUP_LOG_JSON` (false par défaut) |
| **Rotation fichiers** | `gopkg.in/natefinished/lumberjack.v2` (si file logging requis — pattern 5MB×3 identique) |
| **Métriques futures** | Réserver un slot pour OpenTelemetry (`go.opentelemetry.io/otel`) mais ne PAS l'ajouter au Sprint 0 |

**Correspondance structlog → slog :**

| structlog (Python) | slog (Go) |
|---------------------|-----------|
| `merge_contextvars` | `slog.With()` dans le context |
| `add_log_level` | Natif (`slog.LevelInfo`, etc.) |
| `TimeStamper(iso)` | Natif (timestamp RFC3339 en JSON) |
| `JSONRenderer` | `slog.NewJSONHandler` |
| `ConsoleRenderer` | `slog.NewTextHandler` |
| `structlog.get_logger()` | `slog.Default()` ou `slog.With(attrs...)` |

**Pourquoi pas zerolog/zap :** `slog` est stdlib depuis Go 1.21, zéro dépendance, suffisant pour les besoins LevelUp. Si un benchmark montre des bottlenecks (improbable pour un dashboard), on peut brancher un handler zerolog sous slog sans changer le code appelant.

---

### D6 — Cache de tokens Halo natif en Go

**Choix : MSAL Go SDK (`github.com/AzureAD/microsoft-authentication-library-for-go`) + cache DuckDB `sync_meta`**

| Critère | Décision |
|---------|----------|
| **SDK** | `github.com/AzureAD/microsoft-authentication-library-for-go` v1.7+ (MIT, officiel Microsoft) |
| **Client type** | `public.Client` (app publique = device code flow, pas de client secret) |
| **Device Code Flow** | `client.AcquireTokenByDeviceCode(ctx, scopes)` → `DeviceCode.Result` (user_code, verification_url) → `DeviceCode.AuthenticationResult(ctx)` |
| **Silent refresh** | `client.AcquireTokenSilent(ctx, scopes, public.WithSilentAccount(account))` |
| **Cache persistence** | `public.WithCache(accessor)` — implémentation custom de `cache.ExportReplace` qui lit/écrit dans DuckDB `sync_meta` |
| **Table** | `sync_meta (key VARCHAR PK, value VARCHAR, updated_at TIMESTAMP)` — clé `msal_token_cache`, valeur = JSON sérialisé du cache MSAL |
| **Process cache** | `sync.Map` keyed par `db_path` — tokens spartan/clearance (~1h TTL) stockés en mémoire, pas en DB |
| **Échange Halo** | `access_token → spartan_token + clearance` via HTTP direct (port `_halo_exchange.py` → Go `platform/halo/exchange.go`) |

**Architecture token flow Go :**

```text
MSAL Go (public.Client)          Halo Exchange (HTTP)
   │                                    │
   ├─ AcquireTokenByDeviceCode ─►       │
   │   (user_code, url)                 │
   │                                    │
   ├─ AuthenticationResult() ─►         │
   │   access_token (Azure AD)          │
   │       │                            │
   │       └──────────────────────►  ExchangeToken()
   │                                  spartan_token
   │                                  clearance_token
   │                                    │
   ├─ AcquireTokenSilent ─►            │
   │   (refresh via MSAL cache)         │
   │                                    │
   └─ WithCache(duckdbAccessor) ─►  sync_meta (DuckDB)
       Export/Replace JSON cache
```

**Risque majeur — compatibilité de cache Python↔Go :**
- MSAL Python et MSAL Go utilisent tous deux un format JSON pour le cache, mais la structure interne diffère (clés, namespaces).
- **Décision : pas de partage de cache.** Au premier démarrage Go, si aucun cache Go n'existe, déclencher un nouveau Device Code Flow. Le cache Python n'est pas migré — les deux stacks (Python et Go) maintiennent chacune leur propre cache dans `sync_meta` avec des clés différentes (`msal_token_cache` pour Python, `msal_token_cache_go` pour Go).
- Pendant la phase de coexistence Python/Go, les deux caches cohabitent dans `sync_meta`.

**Pourquoi pas un bridge Python :** ajouter un subprocess Python pour l'auth crée une dépendance runtime sur Python — contraire à l'objectif d'autonomie du backend Go. MSAL Go est complet et officiel.

---

### D7 — Génération OpenAPI et types frontend

**Choix : spec-first avec `oapi-codegen` + `openapi-typescript`**

| Critère | Décision |
|---------|----------|
| **Approche** | **Spec-first** : l'`openapi.yaml` est la source de vérité |
| **Source initiale** | Export du `/api/openapi.json` Python actuel, nettoyé et versionné dans `docs/api/openapi.yaml` |
| **Génération Go** | `github.com/oapi-codegen/oapi-codegen` v2 — génère types Go + interfaces handlers chi |
| **Génération TS** | `openapi-typescript` (npm) — génère types TypeScript zero-runtime depuis le spec |
| **Validation runtime** | `oapi-codegen` avec middleware de validation chi (vérifie request/response contre le spec) |
| **Documentation** | `/api/docs` (Scalar UI ou Swagger UI) servi statiquement depuis le spec YAML |

**Pipeline CI :**

```text
openapi.yaml (source de vérité, versionné dans le repo)
      │
      ├──► oapi-codegen ──► internal/api/gen/      (types + interfaces Go)
      │                      ↳ handlers implémentent les interfaces
      │
  ├──► openapi-typescript ──► apps/web/src/lib/api/types.ts
      │
      └──► Scalar/Swagger UI ──► /api/docs (servi par le backend)
```

**Pourquoi spec-first et pas code-first :**
- La migration est un **portage** — le spec API existe déjà (Python FastAPI). Partir du spec existant garantit la parité.
- `oapi-codegen` génère des **interfaces Go** que les handlers doivent implémenter — le compilateur vérifie que chaque endpoint est couvert.
- Le spec devient le **contrat partagé** entre backend (Go) et frontend (TypeScript) — pas de dérive.
- Les alternatives code-first (`swaggo/swag` : commentaires Go, `huma` : framework opinonné) sont mieux adaptées aux projets greenfield, pas à un portage avec contrat existant.

**Rétrospective quand utiliser code-first :** si plus tard de nouvelles routes sont créées en Go sans équivalent Python, on peut toujours les ajouter au spec manuellement d'abord (1 minute par endpoint), puis regénérer. Le surcoût est négligeable comparé à la garantie de parité.

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

### Pas de bridge Python transitoire

SPNKr est un client HTTP simple (retry + parsing JSON) — il n'y a pas de raison d'introduire un bridge Python temporaire. Le client Go `pkg/haloapi/` est implémenté directement dès le Sprint 11. La seule exception potentielle reste le weapon parser (D6) : si le portage binaire échoue, un subprocess Python étroit est toléré pour cette seule fonction.

### Criteres de validation du client Go Halo

1. Tous les endpoints Halo utiles ont un equivalent Go teste sur fixtures.
2. 3 cycles de sync passent avec le client Go natif.
3. Les secrets et caches ne dependent plus d'un processus Python.

## POC et validation initiale

Le Sprint 0 (2 jours — voir section "Ordre de migration recommande") couvre les validations critiques :

- **DuckDB Go read-only** : ouvrir les 3 types de DB, executer des requetes representatives, verifier types et locks.
- **Client HTTP Go** : MSAL Go device code flow (`AcquireTokenByDeviceCode`), endpoint `/health` + `/bootstrap`.
- **Compatibilite** : fichiers DuckDB crees par Python ouverts par `duckdb-go`, coexistence explicite des caches MSAL Python/Go documentee (cles separees, pas de deserialisation croisee).

**Gate** : si un de ces points echoue de facon non contournable, la migration est re-evaluee.

> Note : le plan initial contenait des POC A/B/C/D separes. Ils ont ete fusionnes dans le Sprint 0 pour eviter la duplication. SPNKr est un client HTTP simple a reimplementer directement en Go — pas de bridge transitoire (voir "Pas de bridge Python transitoire").

## Ordre de migration recommande

### Sprint 0 — POC rapide (2 jours max)

Objectif : valider en 2 jours que les briques fondamentales Go fonctionnent, avant tout travail de cadrage detaille. C'est un test de faisabilite, pas un prototype. Si ca ne passe pas, le plan s'arrete la.

**Jour 1 — DuckDB Go + types** :
1. `go mod init`, ajouter `github.com/duckdb/duckdb-go`
2. **Verifier la version DuckDB embarquee par `duckdb-go`** : elle doit etre compatible avec les fichiers crees par DuckDB Python 1.4.4 (meme majeure+mineure). Si `duckdb-go` embarque une version differente, tester l'ouverture et verifier qu'aucune migration implicite du format de stockage ne se produit (`PRAGMA database_size` avant/apres).
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
5. **Valider la strategie de coexistence du cache MSAL** : MSAL Python et MSAL Go ont des formats de cache differents. Documenter les cles separees dans `sync_meta`, l'invalidation explicite eventuelle du cache Python au premier demarrage Go, et l'absence de deserialisation croisee.

**Gate Sprint 0** :
- ✅ DuckDB Go lit les 3 types de DB sans erreur sur Windows
- ✅ La version DuckDB embarquee par `duckdb-go` est compatible avec les fichiers Python 1.4.4 (pas de migration implicite)
- ✅ ATTACH fonctionne correctement avec la strategie de pool choisie (`sql.Conn`, `ConnInitFunc` ou pool custom)
- ✅ Les types UBIGINT/TIMESTAMP sont correctement mappes
- ✅ CGo compile sur Windows avec un toolchain documente et reproductible
- ✅ Un endpoint HTTP retourne un JSON coherent avec le Python
- ✅ MSAL Go device code flow fonctionne (au moins jusqu'au user_code)
- ✅ La strategie de coexistence du cache MSAL est documentee (cles separees + invalidation explicite si necessaire)
- ❌ Si un de ces points echoue de facon non contournable → re-evaluer le plan

### Phase 0 - Cadrage, inventaire et corpus

Objectif : figer la reference avant d'ecrire du Go.

Travaux :

1. Inventorier les surfaces Python a porter : API, services, repositories, auth, sync, scripts, **y compris `src/app/`** (voir ci-dessous).
2. Figer les contrats API existants cote React et les payloads metiers critiques.
3. Constituer un corpus figé pour Career, Match History, Explorer, Match View, Setup et Settings.
4. Ecrire une matrice Python -> Go par package avec proprietaire, criticite et dependances.
5. Definir des golden values chiffrées pour les parcours prioritaires.
6. Definir la capability map initiale pour `halo_infinite` uniquement, ainsi que la forme cible de son exposition dans le bootstrap produit.

Livrables :

1. Inventaire complet des modules a migrer.
2. Liste des invariants non negociables.
3. Corpus de tests rejouable sous Windows et Linux.
4. Premier Go/No-Go technique sur DuckDB et auth.
5. **Definition of Done par sprint** : pour chaque sprint des phases 1-5, definir les criteres de sortie explicites (ex : "Sprint 1.2 termine quand Q1, Q2, Q3, Q5 passent en Go avec golden values identiques a < 0.01").
6. Contrat documentaire du bootstrap : titre courant, provider courant, capabilities produit, surfaces degradees si necessaire.

### Phase 1 - Socle Go read-only

Objectif : prouver que Go peut lire les memes donnees et exposer les memes contrats sans ecrire dans les DBs.

Travaux :

1. Monter le service Go, le healthcheck, le request_id, le logging et la config.
2. Implementer bootstrap, players et resolveur de filtres.
3. Valider la compatibilite avec la facade V7 via shadow mode ou feature flag au moment de l'integration ; le frontend n'est pas un chantier a refaire.
4. Faire porter par le bootstrap les metadonnees produit necessaires : titre courant, provider courant et capability map utile au consommateur.
5. Comparer les payloads Go contre les payloads Python sur le corpus de reference.

Livrables :

1. Service Go runnable localement.
2. Endpoints read-only equivalents sur bootstrap, players, filters.
3. Bootstrap capable d'annoncer explicitement les capabilities produit sans exposer les details 343i.
4. Suite de tests de parite backend Go vs golden values.

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

### 5. SSE pour le suivi de sync en temps reel

Le frontend a besoin de savoir ou en est un sync en cours. En Python, c'est du polling (`GET /jobs/{id}`). En Go, passer aux **Server-Sent Events** (SSE) :

1. **Endpoint** : `GET /api/sync/events` — stream SSE (Content-Type: `text/event-stream`).
2. **Events a emettre** :
   - `sync:started` (job_id, gamertag, scope)
   - `sync:progress` (phase, current_match, total_matches, elapsed_ms)
   - `sync:match_processed` (match_id, enrichments_applied)
   - `sync:error` (message, is_fatal)
   - `sync:completed` (job_id, matches_synced, duration_ms)
3. **Implementation** : goroutine de sync ecrit dans un channel, le handler SSE lit le channel et flush vers le client.
4. **Fallback** : si le client ne supporte pas SSE, `GET /jobs/{id}` reste disponible (polling classique).
5. **Timeout** : couper la connexion SSE apres 30 min d'inactivite (heartbeat `ping` toutes les 15s pour garder la connexion vivante).
6. **Sprint cible** : S22 (Jobs persistants) — le SSE s'ajoute apres que le modele de jobs est en place.

### 6. Pagination cursor-based pour l'historique de matchs

Le Python utilise une pagination offset+limit. En Go, passer a une **pagination cursor-based** :

1. **Pourquoi** : l'offset-based devient lent et instable quand des matchs sont inseres pendant la navigation (l'offset se decale). Le cursor garantit une navigation stable.
2. **Contrat** : `GET /api/players/{xuid}/matches?cursor={opaque}&limit=25` → reponse avec `next_cursor` et `has_more`.
3. **Implementation du cursor** : encoder `(started_at, match_id)` en base64url comme curseur opaque. La requete SQL utilise `WHERE (started_at, match_id) < (?, ?)` pour la page suivante.
4. **Compatibilite** : le frontend React doit consommer le cursor au lieu de `page=N`. Le contrat OpenAPI garde un champ `page` optionnel pour la compatibilite temporaire, marque comme deprecated.
5. **Sprint cible** : S06 (Match History) — c'est la premiere surface qui pagine.

## Estimation d'effort

Ordre de grandeur realiste, sous reserve qu'il n'y ait pas de refonte produit parallele :

- 1 ingenieur backend principal : 7 a 10 mois (~55 000 LOC Python verifie, dont analysis=14K, sync=13K, api=12K).
- 2 ingenieurs backend + 1 support frontend : 4 a 6 mois.
- 1 seule personne a temps partiel : risque eleve de glissement au-dela de 12 mois.

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
13. Le systeme de backfill (`BACKFILL_FLAGS` historiques + `MatchBits`) est numeriquement identique entre Python et Go — pas "equivalent", identique.
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

1. Settings, session, auth et jobs tournent sans Python sur le chemin nominal.
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

1. **POC Sprint 0 echoue** : `duckdb-go` ne fonctionne pas sur Windows, ou CGo est trop fragile pour un build reproductible.
2. **Phase 1 depasse 3× l'estimation** : si 4-6 semaines prevues deviennent 15+ semaines sans parcours read-only fonctionnel.
3. **343 Industries change fondamentalement l'API Halo** : nouveau systeme d'auth, changement radical des endpoints, deprecation de l'API stats. Le portage Go cible un contrat API qui n'existe plus.
4. **DuckDB Go driver est abandonne** : le driver `duckdb-go` n'est plus maintenu et aucune alternative credible n'emerge.
5. **Le produit evolue plus vite que le portage** : si apres 3 mois les golden values les sont obsoletes parce que le produit Python a trop bouge.
6. **Fatigue / motivation** : pour un developpeur solo, 6-10 mois de portage sans feature visible est un risque reel. Si le portage devient une corvee, il vaut mieux rester sur Python+FastAPI qui fonctionne.

**Consequence d'un arret** : le worktree Go est abandonne ou archive sans merge. Il n'y a pas de "retour" a organiser apres coup puisque Python reste la baseline tant que Go n'a pas franchi les gates d'integration.

---

## Modele de deploiement cible

Le detail runtime, packaging, Docker, jobs persistants et contraintes ffprobe/ffmpeg est maintenu dans [OPS_COMPAT_CHECKLIST.md](OPS_COMPAT_CHECKLIST.md).

Principes retenus :

1. La cible retenue est un binaire unique `levelup` avec sous-commandes (`api`, `sync`, `backfill`, `tools`) ; le packaging, la CI et les exemples de commandes doivent s'aligner dessus.
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

6. **Build matrix simplifiee par OS** : le code Go reste portable, mais avec CGO + DuckDB il faut construire et tester chaque OS cible en CI ; pas de promesse de cross-compilation triviale depuis Windows.

7. **Tests de performance integres** : `go test -bench` est natif. Les benchmarks de regression sur les requetes critiques s'integrent naturellement dans la CI.

**Attention** : ces opportunites ne justifient pas a elles seules la migration. Elles sont des bonus a capturer pendant le portage, pas des raisons de demarrer le portage.

---

## Adaptation pour developpeur solo

Ce plan est ecrit dans un style "programme d'entreprise". Pour un developpeur solo, certaines pratiques doivent etre simplifiees :

### Ce qui est surdimensionne et peut etre allegre

1. **Shadow mode complet** (Phase 1.4 du complement) : remplacer par une comparaison manuelle sur 10-20 requetes representant les golden values. Un `diff` JSON des reponses Python vs Go suffit. Pas besoin d'un proxy transparent.
2. **Soak test 2 semaines** (Phase 5.1) : remplacer par 2-3 cycles de sync reels sur les vrais joueurs. Si 3 syncs passent sans divergence, c'est suffisant.
3. **8 registres anti-oubli** : fusionner en une seule checklist dans un fichier `GO_MIGRATION_CHECKLIST.md`. 8 fichiers separes c'est de la bureaucratie pour une personne.
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
- 96 champs SyncScope a reproduire fidelement
- 35 migrations DuckDB a rendre idempotentes en Go
- 12 mixins sync engine (certains avec de la logique film/bitstream complexe) + transformers/ (~2 400 LOC)
- 2 algorithmes de scoring non triviaux (performance relative percentile + TrueSkill2 adapte)
- ~120 arguments CLI a reconstituer
- 28+ endpoints API a porter avec middleware, injection de dependances, rate limiting
- ~550 fichiers de tests a traduire ou reconstituer
- Gestion des accents/i18n (14 langues dans metadata)

**Estimation revisee** : 7 a 10 mois pour 1 ingenieur backend senior a temps plein (~55 000 LOC Python verifie : analysis=14K, sync=13K, api=12K, repos+services+auth+scripts≈16K). 4 a 6 mois avec 2 ingenieurs. Le risque principal n'est pas la quantite de code mais la densite des invariants metier a verifier.

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

**Portage Go** : C'est le module le plus risque. Le parsing binaire est fragile et mal documente. **Recommandation : porter avec `encoding/binary` (Go est naturellement fort pour le parsing binaire) ; fallback subprocess Python uniquement si le portage echoue (voir D6).**

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
| W14 | Mise a jour backfill bitmask | sync_meta / match_registry | Haute (flags historiques + `MatchBits`) |
| W15 | Sauvegarde cache MSAL | sync_meta | Haute (auth) |

---

## Inventaire des dependances Python et equivalents Go

| Dependance Python | Fonction | Equivalent Go | Maturite |
|-------------------|----------|---------------|:--------:|
| `duckdb` | Moteur OLAP | `github.com/duckdb/duckdb-go` | Haute |
| `polars` | DataFrames/Series | SQL DuckDB natif + structs Go | N/A |
| `pydantic` v2 | Validation/serialisation | Structs Go + `go-playground/validator` | Haute |
| `fastapi` | Framework HTTP | `chi` + `go-playground/validator` | Haute |
| `uvicorn` | Serveur ASGI | Serveur net/http natif | Haute |
| `msal` | Auth Microsoft | Client MSAL Go ou HTTP direct | ⚠️ A valider (POC C) |
| `aiohttp` | Client HTTP async | `net/http` + goroutines | Haute |
| `pyarrow` | Parquet | `apache/arrow-go` | Haute |
| `plotly` | Construction de figures server-side (47 fichiers, ~12K LOC) | `domain/chart/` — JSON Plotly construit en Go, rendu par `react-plotly.js` frontend | **Tres haute** |
| `streamlit` | UI | N/A (elimine par migration React) | N/A |
| `itsdangerous` | Cookie de session signe | `github.com/gorilla/securecookie` ou HMAC maison compatible D4 | Haute |
| `ffprobe` (subprocess) | Metadata video | Meme approche (exec ffprobe) | Haute |
| `chromadb` (`src/ai/`) | RAG vectoriel | Hors scope (outillage dev) | N/A |

### Dependance critique : DuckDB Go driver

Le driver `duckdb-go` (`github.com/duckdb/duckdb-go`) utilise CGo pour wrapper la lib C de DuckDB.

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
- Valider `duckdb-go` sur Windows 10/11 et Linux
- Tester : open read-only, ATTACH, write lease, lock behavior
- Verifier explicitement : 2 writers meme path, 2 writers paths differents, et absence de migration implicite non voulue a l'ouverture
- Tester les types critiques : UBIGINT (weapon_id), TIMESTAMP WITH TIME ZONE, VARCHAR, BOOLEAN
- Tester COALESCE, CASE WHEN, GROUP BY, window functions

### Phase 1 — Socle Go read-only (4-6 semaines)

**Sprint 1.1 — Squelette HTTP** :
- `go-api/cmd/levelup/main.go` : server, config, healthcheck, request_id middleware
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
- Socle provider Halo sur fixtures + etats de degradation explicites pour les blocs live
- Battle Pass + Challenges : contrats et etats "auth requise / indisponible" prepares ici ; activation live reportee apres la phase auth
- Timeline (5 derniers matchs)
- Media recents (3 derniers)

**Sprint 2.4 — Escouade + Synthese** :
- Top coequipiers (Q5)
- 13 sous-modules d'analyse — certains sont complexes (radar, first blood, clutch)
- Solo vs Squad breakdown
- Heatmap, top semaine
- Regle de separation renderer/frontend : si une figure est deja construite dans React, le backend Go expose les datasets et primitives renderer-agnostic, pas un nouveau payload Plotly impose.

**Sprint 2.5 — Profil Citations + Medias** :
- Citations : portage du CitationEngine (regles custom)
- Medias : galerie paginee, filtres, groupement

**Gate phase 2** : 41 tests Playwright passent avec le backend Go.

### Phase 3 — Auth, session, settings, jobs (3-4 semaines)

**Sprint 3.1 — Modele de session** :
- Fichiers JSON dans `data/sessions/` + cookie signe HMAC-SHA256 (pas de JWT)
- `SessionData` miroir du modele Python, revocation et expiration cote serveur
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
- Reproduction des 12 mixins du SyncEngine en packages Go (ConnectionMixin, SchemaMixin, SharedWritesMixin, PerformanceMixin, SkillRatingMixin, CareerMixin, AggregatesMixin, WeaponKillsEngineMixin, MatchProcessingMixin, MatchProcessingHelpersMixin, EnrichedWritesMixin, FanoutEnrichmentMixin)
- Portage de `transformers/` (~2 400 LOC : normalisation, nettoyage, transformations batch)
- Portage de `_batch_audit.py`, `_batch_columns.py`, `_career_rank_api.py`, `_tokens.py`, `_asset_langs.py`
- Write lease identique au Python

**Sprint 4.2 — Pipeline post-sync** :
- Performance score (algorithme 1)
- LUSR/TrueSkill (algorithme 2)
- Refresh materialized views
- Fanout enrichments

**Sprint 4.3 — Backfill complet** :
- Port de SyncScope (96 champs)
- Port des `BACKFILL_FLAGS` historiques (bits 0-15) + `MatchBits` (bits 16-22), y compris le bit legacy obsolet a ne jamais reecrire
- CLI : ~120 arguments
- `levelup backfill --player X --medals --force-medals`

**Sprint 4.4 — Migrations DuckDB** :
- Port du registre de migrations (35 steps)
- Idempotence garantie (schema_migrations table)
- Auto-apply au demarrage

**Sprint 4.5 — Weapon parsing** :
- Portage du parser de chunks film (binaire)
- Si le portage echoue : subprocess Python uniquement pour le weapon parser (fallback D6)
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

**Gate phase 4** : `levelup sync --full --gamertag X --max-matches 500` produit un resultat identique a Python.

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

Le driver `duckdb-go` utilise CGo. Sur Windows, cela necessite un compilateur C (MinGW ou MSVC). Cela complique le build et le CI. **Mitigation** : tester le build Windows des le Sprint 0, pas apres.

### Risque 8 — Mapping de types DuckDB ↔ Go

DuckDB a des types specifiques (UBIGINT, HUGEINT, TIMESTAMP WITH TIME ZONE, BLOB) dont le mapping vers Go n'est pas trivial. Le weapon_id est un UBIGINT (uint64 en Go), les timestamps necessitent time.Time avec timezone. **Mitigation** : creer un jeu de tests de types couvrant chaque type DuckDB utilise.

### Risque 9 — Latence des requetes ATTACH

En Python, chaque DuckDBRepository ATTACH `shared_matches_v2.duckdb` en read-only. En Go avec un pool de connexions, l'ATTACH doit se faire une seule fois par connexion (pas a chaque requete). Sinon la latence explose. **Mitigation** : ATTACH dans l'initialisation du pool, pas dans les handlers.

### Risque 10 — Perte de la logique "degradation gracieuse"

Beaucoup de code Python a des `if metric is None: skip` ou `try/except: return None`. Ce pattern de degradation gracieuse est critique pour l'UX (un match sans precision n'empeche pas d'afficher le reste). Le code Go doit reproduire cette robustesse, pas paniquer sur des `nil`.

---

## Decisions structurantes gelees avant la premiere ligne de Go

1. **Driver DuckDB Go** : `github.com/duckdb/duckdb-go` v2.x. Le Sprint 0 prouve la compatibilite Windows/Linux, il ne re-decide pas le package.
2. **Framework HTTP** : `chi` + `go-playground/validator`.
3. **Generation OpenAPI** : spec-first avec `oapi-codegen` + `openapi-typescript`.
4. **Session/cookies** : fichiers JSON + cookie signe HMAC-SHA256 ; pas de JWT, pas de `scs`, pas de `gorilla/sessions`.
5. **Logging** : `log/slog` (stdlib Go 1.21+).
6. **Config** : struct Go natif + `os.Getenv` + parsing JSON `app_settings.json`.
7. **MSAL Go** : SDK Microsoft officiel + support refresh tokens de compatibilite (env + `sync_meta`).
8. **Parquet** : `apache/arrow-go` pour le cold storage archive.
9. **Tests** : `testing` stdlib + `testify/assert` pour les assertions.
10. **CI** : GitHub Actions avec build matrix par OS cible (Windows, Linux, amd64) ; pas de promesse de cross-build CGO magique.
11. **Charting** : architecture decouplee — `domain/chart/` produit des `ChartPayload` renderer-agnostic, `adapter/plotly/` convertit vers `PlotlyFigurePayload` uniquement pour les surfaces backend-rendered. Les figures deja assemblees dans React restent cote frontend. Voir [GO_ARCHITECTURE_RULES.md §11](GO_ARCHITECTURE_RULES.md).

---

## Points de pilotage restants

| # | Question | Recommandation actuelle | Impact |
|---|----------|-------------------------|--------|
| P1 | Structure exacte du module Go dans le monorepo | Go reste dans le meme repo, module `go-api/` ou equivalent ; ne pas re-ouvrir le debat monorepo/polyrepo | Structure repo |
| P2 | Weapon parser : portage natif ou fallback etroit ? | Porter en Go ; bridge Python uniquement si le portage echoue | Risque |
| P3 | Strategie pool `database/sql` + ATTACH | A prouver au Sprint 0 (`sql.Conn` pinee, `ConnInitFunc` ou pool custom) | Architecture |
| P4 | Feature flags de bascule | Variable d'env + champ `app_settings.json` | Bascule |
| P5 | Observabilite minimale | Logs structures `slog` + endpoint `/debug/stats` | Ops |

---

## Checklist pre-lancement (a valider AVANT d'ecrire du Go)

La checklist exhaustive et maintenue est desormais dans [OPS_COMPAT_CHECKLIST.md](OPS_COMPAT_CHECKLIST.md).

Minimum incompressible :

- [ ] Schema OpenAPI versionne, golden values a jour et perimetre contractuel de depart explicitement nomme
- [ ] POC DuckDB Go valide sur Windows et Linux
- [ ] Version DuckDB `duckdb-go` compatible avec les fichiers Python 1.4.4 (pas de migration implicite)
- [ ] Strategie pool `database/sql` + ATTACH validee (D8)
- [ ] Toolchain CGo Windows documente et reproductible en CI
- [ ] MSAL Go valide, avec strategie explicite de support refresh tokens
- [ ] Strategie de compatibilite cache MSAL Python → Go documentee
- [ ] Mode de demo/test defini des le socle
- [ ] Modele de jobs persistants defini avant tout portage de sync/setup
- [ ] Mecanisme de feature flags choisi (D9)
- [ ] Matrice et checklist ops initialisees avant toute suppression de code Python
