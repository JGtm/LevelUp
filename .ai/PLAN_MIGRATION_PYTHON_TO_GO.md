> [!WARNING]
> DOCUMENT DE CADRAGE — a valider avant execution.
> Les decisions D1-D7 doivent etre tranchees, le POC initial (2 jours) doit passer, et la migration React/FastAPI doit etre stabilisee avant de demarrer la Phase 1.

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

## Objectif

Remplacer progressivement le runtime Python de LevelUp par un backend et des outils Go, sans casser les invariants metier existants : DuckDB v6, auth Halo, sync/backfill, logique analytique et parcours produits deja engages via React.

Ce plan part d'un constat simple : la migration Streamlit -> FastAPI/React est deja entamee dans le depot. Une migration Python -> Go ne doit donc pas devenir une seconde reecriture big bang. Elle doit s'appuyer sur les contrats API, les fixtures de parite et les parcours deja identifies.

## Resume executif

- La migration complete vers Go est faisable, mais elle doit etre menee comme un programme de remplacement progressif, pas comme une reecriture simultanee.
- Le frontend React/TypeScript doit rester en place. Il n'y a pas de raison de refaire aussi la couche web.
- DuckDB doit rester la source de verite. Revoir en meme temps le schema, le moteur et le runtime multiplierait inutilement le risque.
- Les ecrans read-only doivent etre portes avant l'auth, les jobs et le moteur de sync.
- Les calculs aujourd'hui portes par Polars ne doivent pas etre recodes "a l'intuition" en Go. Il faut basculer soit vers SQL explicite, soit vers des pipelines Go verifies contre des golden values.
- Le programme doit utiliser une approche strangler au moment de l'integration et de la bascule : Python et Go cohabitent a ce niveau, meme si l'implementation locale se fait tranquillement dans un worktree dedie.

## Hypothese de travail : worktree dedie

Ce plan suppose explicitement que l'implementation se fait dans un worktree dedie. Cela change la discipline de travail locale, mais pas les criteres de validation avant integration.

## Ce que cela change vraiment

1. Il n'est pas necessaire de garder le projet localement fonctionnel a chaque commit intermediaire dans le worktree.
2. Il est acceptable de casser provisoirement les imports, le build, les scripts ou une partie des parcours pendant un gros refactor.
3. Il est acceptable de faire des deplacements massifs de packages, de renommer large, ou de remplacer des frontieres entieres sans maintenir une compatibilite locale temporaire.
4. Les bridges temporaires, feature flags, shadow mode et rollback ne sont pas des obligations de confort pendant le dev local ; ce sont des obligations d'integration et de bascule.

## Ce que cela ne change pas

1. Avant merge, revue formelle ou bascule, le lot doit revenir a un etat testable et explicable.
2. Les contrats React, les golden values et la parite metier restent la reference obligatoire.
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
3. Tout ce qui touche les endpoints deja consommes par React.
4. Tout ce qui touche sync, backfill, smoke test, backup, restore ou media indexing.

## Decision de cadrage

### Ce que le plan assume

- Le frontend React/TypeScript deja en cours reste la facade cible.
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
7. Notifications Discord : le backend Go doit etre capable d'envoyer des embeds Discord (webhook) apres sync/backfill, exactement comme le Python actuel (`src/utils/discord_notifier.py`, `_discord_embed.py`, `_discord_queries.py`). Inclure : embeds bilingues, thumbnail upload, anti-spam via `discord_notified_at`, notifications new version.

## Decisions techniques a prendre avant la premiere ligne de Go

1. Valider un driver DuckDB compatible Windows/Linux et le comportement de lock associe.
2. Choisir le socle HTTP Go et la strategie de validation des payloads.
3. Choisir la forme des contrats de charting : series JSON, points, buckets, annotations, jamais des widgets Python encodes.
4. Choisir la strategie de session et de cookies pour remplacer le modele actuel.
5. Choisir la strategie de logging, trace, request_id et observabilite.
6. Decider si le cache de tokens Halo est porte nativement en Go des la phase auth, ou transite provisoirement par un bridge Python a eteindre ensuite.
7. Decider la strategie de generation OpenAPI et de types frontend.

## Comment etre sur de ne rien oublier

Tu ne peux pas le garantir par memoire ou par intuition. Il faut remplacer la memoire humaine par des registres obligatoires, des tests de couverture et des gates de suppression.

## Registre anti-oubli obligatoire

1. Tenir une matrice Python -> Go vivante pour chaque package, script et point d'entree.
2. Donner a chaque element exactement un statut : a migrer, a garder temporairement, a encapsuler derriere un bridge, a retirer, ou hors scope.
3. Tenir un registre des dependances externes : DuckDB, SPNKr, auth Halo, MSAL, ffprobe, Discord, systeme de thumbnails, CI, OS cibles.
4. Tenir un registre des contrats HTTP : route, payload, consommateur React, golden test associe, proprietaire.
5. Tenir un registre des acces DB : quelles tables et vues sont lues, ecrites, migrees, verrouillees, et par quel binaire.
6. Tenir un registre des jobs de fond : sync, backfill, smoke test, media indexing, reindex, cron, retries, side effects.
7. Tenir un registre des scripts d'exploitation : sync, restore, backup, check_env, diagnostic, migration, release, Docker, CI.
8. Tenir un registre de decommission : ce qui peut etre supprime, ce qui doit cohabiter, et ce qui bloque encore la suppression d'un morceau Python.

## Regle simple

Si un fichier Python n'apparait dans aucun registre, il est considere comme non couvert et la migration n'est pas assez mature pour le toucher.

## Routine de couverture obligatoire avant chaque phase

1. Verifier qu'aucun package Python actif n'est sans statut explicite.
2. Verifier qu'aucun endpoint React critique ne depend d'un comportement Python non documente.
3. Verifier qu'aucune ecriture DuckDB n'existe sans plan de migration idempotent.
4. Verifier qu'aucun script d'exploitation Python critique n'est oublie hors backlog.
5. Verifier qu'aucune variable d'environnement, secret ou cache disque n'est implicitement requise par un service Python restant.
6. Verifier qu'aucune tache de fond ne depend encore d'un rerun UI ou d'un processus Python non inventorie.
7. Verifier qu'aucune suppression de code Python n'est faite sans test de parite, runbook et rollback.

## Definition operationnelle de rien oublier

Tu peux considerer que la couverture est suffisante seulement si :

1. Chaque dossier Python encore vivant a un proprietaire, un statut et un remplacant cible.
2. Chaque commande utilisateur ou operateur a une contrepartie Go, ou une decision explicite de maintien temporaire en Python.
3. Chaque integration externe a une strategie documentee : migration, bridge temporaire ou maintien assume.
4. Chaque contrat de page React est relie a des golden values et a un test de parite.
5. Chaque chemin de production et de maintenance peut etre rejoue sans connaissance implicite du repo.

## Matrice Python -> Go initiale

Cette matrice est volontairement non exhaustive. Compte tenu des deux grosses migrations en cours sur le depot, elle sert a couvrir les angles morts majeurs des maintenant, pas a pretendre fournir un inventaire final complet du premier coup.

### Surfaces prioritaires deja visibles

1. apps/api/
  Statut : a remplacer progressivement par go-api.
  Strategie : port direct des routes et schemas utiles au frontend React.
2. src/data/repositories/
  Statut : a porter partiellement puis largement en Go.
  Strategie : extraire les requetes critiques d'abord, SQL-first quand possible.
3. src/data/services/
  Statut : a porter par parcours metier, pas en bloc.
  Strategie : career, history, explorer, match view d'abord.
4. src/analysis/
  Statut : a auditer avant portage.
  Strategie : ne porter que ce qui ne peut pas vivre proprement en SQL DuckDB.
5. src/auth/
  Statut : a garder temporairement puis a porter.
  Strategie : bridge ou adapter temporaire, puis portage Go des flux retenus.
6. spnkr_pr/
  Statut : a isoler immediatement, a porter plus tard.
  Strategie : anti-corruption layer, puis remplacement progressif des appels utiles.
7. src/app/
  Statut : a analyser puis distribuer entre Go et suppression.
  Strategie : `src/app/` contient la logique applicative Streamlit (page routing, filtres cascade, KPIs, state management, sidebar, cache control, player provisioning, session keys — ~25 fichiers). La majorite de cette logique est deja portee dans `apps/api/` (FastAPI) et `apps/web/` (React). Le portage Go doit identifier ce qui reste uniquement dans `src/app/` et n'a pas d'equivalent API/React (notamment `_filters_cascade.py`, `_filters_friends.py`, `player_provisioning.py`). Le reste est du code Streamlit mort une fois la migration React terminee.
8. src/utils/ (partiellement)
  Statut : a auditer module par module.
  Strategie : `discord_notifier.py`, `_discord_embed.py`, `_discord_queries.py` et `_discord_media.py` sont a porter en Go (webhook HTTP + embed JSON). `radar_chart.py`, `teammates_views.py` et autres helpers UI seront supprimes avec Streamlit.

### Surfaces a traiter tard ou a remplacer plutot qu'a porter

1. src/data/sync/
  Statut : a porter tardivement.
  Strategie : une fois les couches read-only, settings et auth stabilisees.
2. scripts/sync.py et scripts/backfill_data.py
  Statut : a reconstituer en Go a la fin du coeur backend.
  Strategie : repartir des usages reels, pas d'une traduction ligne a ligne.
3. scripts/backup_player.py, scripts/restore_player.py, scripts/check_env.py, scripts/diagnose_player_db.py
  Statut : a recenser puis a reimplementer selectivement.
  Strategie : prioriser ce qui est necessaire a l'exploitation et au support.
4. launcher.py, LevelUp.sh, LevelUp.bat, packaging/
  Statut : a adapter, pas a porter tel quel.
  Strategie : les faire converger vers les nouveaux binaires Go et le frontend conserve.

### Surfaces qui ne doivent pas etre "portees vers Go" au sens litteral

1. src/ui/, streamlit_app.py, streamlit_app_v7.py
  Statut : a retirer via la migration React, pas a reimplementer en Go.
2. apps/web/
  Statut : a conserver.
  Strategie : stabiliser les contrats, pas changer de pile web.
3. tests/parity/, fixtures, golden values
  Statut : a conserver comme reference.
  Strategie : les enrichir si besoin, jamais les traiter comme du legacy a supprimer trop tot.

### Surfaces encore floues et volontairement laissees non exhaustives

1. scripts auxiliaires de maintenance, d'investigation et de migration ponctuelle.
2. sous-modules peu utilises de src/utils/ et de src/ports/.
3. outillage documentaire et scripts de release non encore relies a la future chaine Go.
4. src/app/ — couche applicative Streamlit (~25 fichiers : routing, filtres, state, sidebar, KPIs). La plupart sera supprimee avec Streamlit ; les logiques metier non dupliquees dans apps/api/ doivent etre identifiees en Phase 0.
5. Notifications Discord — `src/utils/discord_notifier.py` et ses sous-modules (`_discord_embed.py`, `_discord_queries.py`, `_discord_media.py`). Fonctionnalites : embed bilingue post-sync/backfill, notification de nouveaux medias (avec thumbnail GIF/image, anti-spam via `discord_notified_at`), notification de nouvelle version. Le webhook URL est configurable via `app_settings.json`.

Regle : toute zone absente de cette matrice initiale doit etre ajoutee avant qu'un lot de migration la modifie profondement.

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
2. Ouvrir `metadata.duckdb` en read-only, executer `SELECT * FROM career_ranks LIMIT 5`
3. Ouvrir `shared_matches_v2.duckdb` en read-only, executer une requete bootstrap joueur (Q1 du catalogue)
4. Tester ATTACH de shared depuis une connexion player
5. Verifier les types critiques : UBIGINT → uint64, TIMESTAMP WITH TIME ZONE → time.Time, VARCHAR, BOOLEAN
6. Tester le lock : ouvrir en read-write, tenter une seconde connexion read-write → observer le comportement
7. Compiler et executer sur Windows (CGo + MinGW ou MSVC)

**Jour 2 — HTTP + MSAL** :
1. Monter un `net/http` minimal avec un handler `/health` qui retourne le nombre de matchs en DB
2. Ajouter un handler GET `/api/bootstrap` qui lit les memes donnees que le Python
3. Comparer le JSON de sortie avec la golden value Python
4. Tester `github.com/AzureAD/microsoft-authentication-library-for-go` : instancier un `PublicClientApplication`, appeler `AcquireTokenByDeviceCode()`, verifier que le user_code + verification_url arrivent

**Gate Sprint 0** :
- ✅ DuckDB Go lit les 3 types de DB sans erreur sur Windows
- ✅ Les types UBIGINT/TIMESTAMP sont correctement mappes
- ✅ CGo compile sur Windows sans contorsion excessive
- ✅ Un endpoint HTTP retourne un JSON coherent avec le Python
- ✅ MSAL Go device code flow fonctionne (au moins jusqu'au user_code)
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
3. Brancher le frontend React sur des endpoints Go en mode shadow ou via feature flag au moment de l'integration ; dans le worktree, un remplacement plus direct est acceptable tant qu'il est reverifie avant merge.
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

1. Porter les services de page en respectant les contrats React deja en place.
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
2. Garder Python en rollback court terme, jamais comme double canonique durable.
3. Mesurer erreurs, latence, ecarts de calcul et issues de jobs.
4. Retirer progressivement les endpoints Python devenus redondants.
5. Supprimer les scripts Python restants seulement apres soak test de plusieurs cycles reels.

Note : le rollback court terme est une exigence de branche integree ou de production. Il n'a pas besoin d'exister a chaque instant dans le worktree de developpement.

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
15. Le frontend React continue a fonctionner sans modification de code pendant tout le portage.
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
16. Lancer le portage Go pendant que la migration React est encore en cours (deux migrations simultanees = double risque d'incoherence contractuelle).

## Gates Go/No-Go recommandes

### Gate 1 - Faisabilite technique minimale

Passer ce gate seulement si :

1. DuckDB Go est valide sur les OS cibles.
2. Les lectures read-only ont une parite demonstrable.
3. L'equipe accepte une migration par strangler et non par big bang.

### Gate 2 - Viabilite produit

Passer ce gate seulement si :

1. Les parcours Career, History, Explorer et Match View tournent en Go avec parite acceptable.
2. Le frontend React consomme les endpoints Go sans changement de semantique.
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

**Consequence d'un arret** : revenir sur Python+FastAPI sans dette. Le travail Go reste dans une branche archivee. Les golden values et les POC enrichissent le projet meme sans migration.

---

## Modele de deploiement cible

Le plan decrit **quoi** porter mais jamais **comment** le resultat final est lance et deploye. Cette section comble ce manque.

### Architecture runtime cible

```
Utilisateur
    |
    v
[levelup-api]          ← Binaire Go unique
  ├── Sert l'API REST (net/http)
  ├── Sert les fichiers statiques React (embed via go:embed ou serveur de fichiers)
  ├── Healthcheck, metrics, logging
  └── Gere les jobs longs (sync, backfill) en goroutines internes
    |
    v
[DuckDB files]         ← Meme layout qu'aujourd'hui (data/warehouse/, data/players/)
```

### Lancement

- **Local Windows** : `levelup-api.exe serve` (remplace `python launcher.py run`)
- **Local Linux** : `./levelup-api serve`
- **Docker** : `docker run -v ./data:/data levelup-api serve` (un seul binaire, image FROM scratch ou distroless)
- **Scripts CLI** : `levelup-api sync --delta --gamertag X` (remplace `python scripts/sync.py`)
- **Outils** : `levelup-api tools backup --gamertag X` (remplace `python scripts/backup_player.py`)

### Un binaire ou plusieurs ?

Recommandation : **un seul binaire avec sous-commandes** (pattern `kubectl`-like). Avantages : une seule chose a distribuer, pas de confusion sur les versions, Go supporte bien les sous-commandes via `cobra` ou `urfave/cli`.

```
levelup-api serve              → lance le serveur HTTP
levelup-api sync --delta       → lance un sync delta
levelup-api sync --full        → lance un sync complet
levelup-api tools backup       → backup DB joueur
levelup-api tools restore      → restore DB joueur
levelup-api tools healthcheck  → diagnostic integrite
levelup-api tools seed         → seed metadata
levelup-api version            → affiche la version
```

### Frontend React

2 options :
1. **Embed via `go:embed`** : les fichiers build React sont embarques dans le binaire Go. Zero dependance externe. Distribution = 1 fichier. C'est l'option recommandee pour un outil solo.
2. **Servi separement** : nginx/caddy sert le React, Go sert l'API. Plus standard pour du cloud, mais ajoute une dependance.

### Service systeme

- **Windows** : `levelup-api.exe` peut etre lance au demarrage via Task Scheduler, un raccourci startup, ou un service NSSM. Le fichier `LevelUp.bat` actuel est remplace par un lancement direct du binaire.
- **Linux** : fichier systemd unit dans `packaging/`, analogue au service actuel mais sans Python/venv.

---

## Migration des donnees utilisateurs existants

Un utilisateur qui utilise LevelUp aujourd'hui a des fichiers DuckDB crees par le Python actuel. Le passage a Go ne doit pas casser leur installation.

### Compatibilite DuckDB

- Le driver `go-duckdb` utilise `libduckdb` (la lib C de DuckDB). Il faut que la version de libduckdb dans go-duckdb soit compatible avec les fichiers crees par DuckDB 1.4.4 (Python).
- **DuckDB n'a PAS de garantie de compatibilite binaire entre versions majeures** (ex: un fichier DuckDB 0.x ne s'ouvre pas en 1.x). Il faut imperativement que go-duckdb utilise DuckDB 1.x.
- **Validation** : pendant le Sprint 0, ouvrir un fichier DuckDB cree par le Python actuel avec go-duckdb et verifier qu'il s'ouvre sans erreur, que les types/schema sont intacts, et qu'aucune migration implicite ne se declenche.

### Cache MSAL

- Le cache MSAL est serialise en JSON dans la table `sync_meta` de chaque `stats.duckdb` via `SerializableTokenCache`. Ce JSON est un format MSAL standard.
- **MSAL Go** utilise le meme format de serialisation (`ExportAuthResult` / `ImportAuthResult`). Le cache devrait etre compatible directement.
- **Validation** : pendant le Sprint 0, deserialiser un cache MSAL cree par Python MSAL avec MSAL Go et verifier que le token refresh fonctionne.

### Sessions HTTP

- Les sessions actuelles utilisent des fichiers filesystem dans `data/sessions/` (serialises par l'API FastAPI Python).
- **Strategie** : au premier demarrage du Go, les sessions existantes sont invalidees. L'utilisateur devra se re-logger. C'est acceptable pour un outil perso (pas pour un SaaS).
- Alternative : lire le format de session FastAPI et le migrer. Non recommande (complexe pour un gain minime).

### Configuration

- `db_profiles.json`, `app_settings.json`, `.env.local` : format JSON/env, lisibles tels quels depuis Go.
- Aucune migration de config necessaire.

### Runbook de transition pour les utilisateurs

```
1. Telecharger le nouveau binaire levelup-api (ou levelup-api.exe)
2. L'installer a cote du repo existant (meme dossier data/)
3. Lancer `levelup-api serve` → il ouvre les memes fichiers DuckDB
4. Se re-authentifier (device code flow) au premier lancement
5. Verifier que les donnees sont intactes (memes matchs, memes stats)
6. Supprimer l'ancien Python/venv quand tout fonctionne
```

---

## Strategie d'evolution du produit pendant le portage

Le plan initial implique un gel des contrats OpenAPI, des migrations DuckDB et du frontend pendant 6-10 mois. C'est irrealiste pour un projet personnel actif. Halo Infinite va continuer a evoluer (nouvelles saisons, maps, modes).

### Regles pragmatiques

1. **Changements schema DuckDB** : autorises si la migration Go correspondante est ajoutee dans la meme semaine. Le Go ne doit jamais etre a plus d'une semaine de retard sur le schema.
2. **Nouveaux endpoints API** : autorises si le contrat OpenAPI est mis a jour et un golden value est ajoute simultanement. Le Go les implemente au prochain sprint de la phase en cours.
3. **Changements frontend React** : autorises s'ils sont client-only (pas de modification du contrat backend). Si un nouveau contrat backend est necessaire, il doit etre ajoute en Python ET documente pour portage Go.
4. **Nouvelles features metier** : implementees d'abord en Python (c'est la reference), puis portees en Go comme partie du backlog normal. Le Go ne doit jamais "rattraper" une feature surprise — elle doit etre planifiee.
5. **Corrections de bugs** : toujours en Python d'abord (c'est la prod). Le Go herite du fix via les golden values mis a jour.

### Discipline de synchronisation

Chaque semaine, verifier :
- [ ] Le schema OpenAPI Go est a jour par rapport au Python
- [ ] Les golden values refletent l'etat actuel du Python
- [ ] Aucune migration DuckDB Python n'est sans equivalent Go planifie

---

## Gestion multi-joueurs en Go

`db_profiles.json` peut contenir N joueurs. Chaque joueur a sa propre DB (`data/players/{gamertag}/stats.duckdb`). Cela a un impact direct sur le pool de connexions.

### Architecture du pool

```
ConnectionManager
  ├── metadata_pool       → 1 pool read-only, shared global
  ├── shared_pool         → 1 pool read-only + 1 write lease (sync.Mutex)
  ├── shared_pve_pool     → 1 pool read-only + 1 write lease
  └── player_pools        → map[gamertag]*PlayerPool
       ├── "PlayerA" → read-only pool + write lease + ATTACH shared (RO)
       ├── "PlayerB" → read-only pool + write lease + ATTACH shared (RO)
       └── "PlayerC" → ...
```

### Regles

1. Les pools player sont crees **a la demande** (lazy init au premier acces) et fermes proprement au shutdown.
2. Chaque pool player fait ATTACH de shared_matches_v2 en read-only une seule fois a l'init (pas a chaque requete — sinon la latence explose).
3. Un sync sur PlayerA n'impacte pas les lectures sur PlayerB (write leases sont par DB path).
4. Si un joueur est ajoute via Setup, son pool est cree dynamiquement.
5. Si un joueur est supprime, son pool est ferme et les fichiers DuckDB peuvent etre archives.
6. Taille de pool read-only par player : bornee a ~5 connexions (un outil perso n'a pas besoin de plus).

---

## Opportunites specifiques a Go

Le portage Go n'est pas qu'une reecriture isometrique. Certaines choses deviennent possibles ou naturelles en Go :

1. **Distribution zero-dependency** : un seul binaire statique avec le frontend React embarque (`go:embed`). Plus besoin de Python, venv, pip, ou d'instructions d'installation complexes. Le "onboarding" utilisateur passe de "installe Python, cree un venv, pip install, configure Azure, lance streamlit" a "telecharge levelup-api.exe, lance-le".

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
3. **8 registres anti-oubli** : fusionner en une seule checklist dans un fichier `.ai/GO_MIGRATION_CHECKLIST.md`. 8 fichiers separes c'est de la bureaucratie pour une personne.
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

### 1. Incoherence avec la migration React/FastAPI declaree canonique

Le document `.ai/MIGRATION_MASTER.md` declare la migration Streamlit -> FastAPI/React **canonique** (41 tests Playwright passants, 8 slices verifices). Le backend de reference est FastAPI Python. Ce plan Go ne contredit pas cette decision, mais il faut etre explicite :

**La migration Go ne commence qu'APRES la stabilisation complete de la migration React/FastAPI.** Le backend FastAPI est la reference contractuelle. Le Go le remplace, il ne le double pas.

### 2. L'estimation d'effort est sous-evaluee

Le plan initial estime "4 a 7 mois pour 1 ingenieur backend". C'est optimiste compte tenu de :
- 96 champs SyncScope a reproduire fidelement
- 36 migrations DuckDB a rendre idempotentes en Go
- 8 mixins sync engine (certains avec de la logique film/bitstream complexe)
- 2 algorithmes de scoring non triviaux (performance relative percentile + TrueSkill2 adapte)
- Plus de 80 arguments CLI a reconstituer
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

`src/data/repositories/_write_lease.py` implementge un semaphore global par chemin DB pour eviter la corruption DuckDB multi-writer. Le plan Go doit reproduire ce mecanisme exactement (timeout 30s, un seul writer par DB a la fois).

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
| 13 | **Sync/Backfill** | Delta, full, 80+ options CLI, post-sync pipeline | Tres haute | P4 (derniere) |

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
1. Pool de connexions read-only : illimite (goroutine-safe via sync.Pool ou equivalent)
2. Write lease : 1 seul writer par DB a la fois (sync.Mutex par path, timeout 30s)
3. ATTACH shared_matches_v2 en read-only depuis chaque connexion player
4. Si sync en cours : les lectures continuent mais peuvent voir un etat intermediaire (acceptable, meme comportement que Python)

### Gestion multi-joueurs

`db_profiles.json` peut contenir N joueurs. Le pool doit gerer des connexions par joueur :
- Un `map[gamertag]*PlayerPool` cree les pools a la demande (lazy init)
- Chaque pool player fait ATTACH shared une seule fois a l'init
- Les write leases sont independants par DB path (un sync sur joueur A ne bloque pas les lectures joueur B)
- Pool read-only par player borne a ~5 connexions
- Voir la section "Gestion multi-joueurs en Go" pour le detail

---

## Matrice detaillee Python → Go (mise a jour)

### Packages Python actifs et statut de portage

| Package | Fichiers | LOC approx | Statut | Cible Go | Difficulte |
|---------|:--------:|:----------:|--------|----------|:----------:|
| `apps/api/` | ~30 | ~3000 | A remplacer | `go-api/internal/api/` | Moyenne |
| `src/data/repositories/` | ~15 | ~4000 | A porter | `go-api/internal/platform/duckdb/` | Haute |
| `src/data/services/` | ~8 | ~1500 | A porter | `go-api/internal/{domain}/` | Moyenne |
| `src/data/sync/` | ~20 | ~5000 | A porter (P4) | `go-api/internal/sync/` | Tres haute |
| `src/data/migration/` | ~40 | ~2000 | A porter | `go-api/internal/platform/migrations/` | Haute |
| `src/analysis/` | ~12 | ~3000 | A porter | `go-api/internal/analysis/` | Tres haute |
| `src/auth/` | ~6 | ~1000 | A porter | `go-api/internal/auth/` | Haute |
| `src/app/` | ~25 | ~2000 | A analyser + distribuer | Majoritairement supprime (Streamlit) | Basse |
| `src/ports/` | 2 | ~200 | Interfaces de reference | Interfaces Go equivalentes | Basse |
| `src/config.py` | 1 | ~150 | A porter | `go-api/internal/platform/config/` | Basse |
| `src/ui/` | ~40 | ~6000 | A supprimer (React) | N/A | N/A |
| `src/ai/` | ~6 | ~1200 | Hors scope | Reste Python ou separe | N/A |
| `src/utils/` (Discord) | ~4 | ~600 | A porter | `go-api/internal/platform/discord/` | Basse |
| `scripts/` | ~25 | ~3000 | A reconstituer | `go-api/cmd/` | Haute |
| `spnkr_pr/` | ~8 | ~800 | A remplacer | Client HTTP Go direct | Moyenne |
| `launcher.py` | 1 | ~500 | A remplacer | `go-api/cmd/levelup-api/` | Moyenne |

**Total a porter** : ~25 000 LOC Python → ~25-35 000 LOC Go. Le boilerplate Go (error handling explicite `if err != nil`, struct marshaling/unmarshaling, SQL row scanning) est significativement plus verbeux que Python+Pydantic+FastAPI. La logique pure est plus concise en Go, mais les handlers HTTP et l'acces DB compensent largement.

### Scripts et outillage

| Script Python | Usage | Portage Go | Priorite |
|---------------|-------|------------|:--------:|
| `scripts/sync.py` | Sync delta/full | `cmd/levelup-sync/` | P4 |
| `scripts/backfill_data.py` | Backfill selectif (80+ flags) | `cmd/levelup-sync/ --backfill` | P4 |
| `scripts/backup_player.py` | Backup DB joueur | `cmd/levelup-tools backup` | P3 |
| `scripts/restore_player.py` | Restore DB joueur | `cmd/levelup-tools restore` | P3 |
| `scripts/healthcheck_db.py` | Diagnostic integrite | `cmd/levelup-tools healthcheck` | P3 |
| `scripts/index_media.py` | Indexation videos | `cmd/levelup-tools index-media` | P3 |
| `scripts/check_env.py` | Validation environnement | `cmd/levelup-tools check-env` | P2 |
| `scripts/diagnose_player_db.py` | Debug schemas | `cmd/levelup-tools diagnose` | P3 |
| `scripts/post_sync_compute.py` | Post-sync pipeline | Integre dans sync | P4 |
| `scripts/archive_season.py` | Archivage Parquet | `cmd/levelup-tools archive` | P4 |
| `scripts/populate_*.py` | Seed metadata | `cmd/levelup-tools seed` | P2 |
| `launcher.py` | Orchestrateur principal | `cmd/levelup-api/` | P1 |

---

## Backfill bitmask : strategie de portage

Le systeme de bitmask (22 bits) est central pour le sync :

```
Bit 0  : medals            (1)
Bit 1  : events            (2)
Bit 2  : skill             (4)
Bit 3  : personal_scores   (8)
Bit 5  : accuracy          (32)
Bit 6  : shots             (64)
Bit 7  : enemy_mmr         (128)
Bit 8  : assets            (256)
Bit 9  : participants      (512)
Bit 10 : participants_scores (1024)
Bit 11 : participants_kda  (2048)
Bit 12 : participants_shots (4096)
Bit 13 : participants_damage (8192)
Bit 14 : aliases           (16384)
Bit 15 : participants_avg_life (32768)
Bit 19 : killer_victim     (524288)
Bit 20 : pve_stats         (1048576)
Bit 21 : weapon_kills      (2097152)
Bit 22 : weapon_kills_no_film (4194304)
```

**Attention** : les bits 4, 16, 17, 18 sont absents (lacunes intentionnelles dans la numerotation). Le portage Go doit reproduire exactement les memes valeurs de bits car les bitmasks sont persistees en DB.

En Go, utiliser des constantes `iota`-like mais avec valeurs explicites :

```go
const (
    BackfillMedals           = 1 << 0   // 1
    BackfillEvents           = 1 << 1   // 2
    BackfillSkill            = 1 << 2   // 4
    BackfillPersonalScores   = 1 << 3   // 8
    // Bit 4 intentionnellement absent
    BackfillAccuracy         = 1 << 5   // 32
    // ... etc, valeurs EXACTES
)
```

---

## Phases revisees avec detail fonctionnel

### Phase 0 — Cadrage, inventaire et corpus (2-3 semaines)

**Livrable 0.1 — Gel des contrats API** :
- Executer `openapi-typescript http://127.0.0.1:8000/api/openapi.json` et versionner
- Freeze du schema OpenAPI (pas de changement cote Python pendant le portage)
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
- Tester les types critiques : UBIGINT (weapon_id), TIMESTAMP WITH TIME ZONE, VARCHAR, BOOLEAN
- Tester COALESCE, CASE WHEN, GROUP BY, window functions

### Phase 1 — Socle Go read-only (4-6 semaines)

**Sprint 1.1 — Squelette HTTP** :
- `go-api/cmd/levelup-api/main.go` : server, config, healthcheck, request_id middleware
- Routing : Chi ou Echo (pas Gin — trop opinionne pour ce cas)
- OpenAPI : generation depuis le meme schema que Python (`oapi-codegen` ou `ogen`)
- Middleware : CORS (memes origines), rate limit, logging structure (slog)
- Graceful shutdown : intercepter `os.Interrupt` / `SIGTERM`, `server.Shutdown(ctx)` avec timeout 15s, drainer les connexions DuckDB en cours

**Sprint 1.2 — Couche repository read-only** :
- `internal/platform/duckdb/pool.go` : connexion pool read-only + write lease
- `internal/platform/duckdb/queries/` : toutes les requetes Q1-Q16 du catalogue
- Tests : chaque requete comparee au golden value

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
- POST /auth/device-flow/start → user_code + verification_url
- GET /auth/device-flow/{attempt_id} → polling
- Echange access_token → spartan_token + clearance_token
- Persistance cache MSAL dans sync_meta (DuckDB write)

**Sprint 3.3 — Settings** :
- GET /settings, PATCH /settings
- POST /settings/media/reset-index (destructif)
- POST /setup/players, POST /setup/smoke-test

**Sprint 3.4 — Jobs longs** :
- Modele : start → poll status → result
- GET /jobs/{job_id}
- POST /sync/initial (retourne AsyncJobStatus)

**Gate phase 3** : onboarding complet fonctionne sans Python.

### Phase 4 — Sync, backfill, outillage (6-8 semaines)

C'est la phase la plus longue et la plus risquee.

**Sprint 4.1 — Moteur sync minimal** :
- Delta sync (fetch nouveaux matchs, insert dans shared + player)
- Reproduction des 8 mixins du SyncEngine en packages Go
- Write lease identique au Python

**Sprint 4.2 — Pipeline post-sync** :
- Performance score (algorithme 1)
- LUSR/TrueSkill (algorithme 2)
- Refresh materialized views
- Fanout enrichments

**Sprint 4.3 — Backfill complet** :
- Port de SyncScope (96 champs)
- Port du bitmask (22 bits, valeurs exactes)
- CLI : 80+ arguments
- `cmd/levelup-sync --backfill --player X --medals --force-medals`

**Sprint 4.4 — Migrations DuckDB** :
- Port du registre de migrations (36 steps)
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
- Port des embeds Discord post-sync/backfill (`src/utils/discord_notifier.py`, `_discord_embed.py`)
- Port des notifications media (`_discord_media.py`) : thumbnail upload, anti-spam `discord_notified_at`
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
- Rollback : re-pointer vers Python en < 1 min

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
| D5 | Cache MSAL : bridge Python temporaire ou portage direct ? | Garder un microservice Python pour l'auth vs tout porter | Portage direct (MSAL Go existe) | Complexite |
| D6 | Weapon parser : porter ou bridge ? | Reecrire le parser binaire en Go vs subprocess Python | Bridge temporaire puis portage (module le plus risque) | Risque |
| D7 | CI : build DuckDB depuis les sources ou pre-built ? | Compiler libduckdb.a vs telecharger les releases | Pre-built (plus rapide, moins fragile) | CI |

---

## Checklist pre-lancement (a valider AVANT d'ecrire du Go)

- [ ] Migration React/FastAPI terminee et stable en production (plus de changements de contrats)
- [ ] Schema OpenAPI versionne et gele
- [ ] Corpus de golden values couvrant les 16 surfaces read-only
- [ ] Baselines de performance mesurees et documentees (p50/p95 par endpoint)
- [ ] POC A valide (DuckDB Go read-only + ATTACH + types UBIGINT/TIMESTAMP)
- [ ] POC C valide (MSAL Go device code flow ou preuve que le bridge est viable)
- [ ] Build CGo Windows valide (go-duckdb compile et passe les tests sur Windows)
- [ ] Decisions D1-D7 tranchees
- [ ] Worktree dedie cree, branche dediee, registres anti-oubli initialises
- [ ] Aucune autre migration majeure en cours
- [ ] Aucune autre migration majeure en cours
