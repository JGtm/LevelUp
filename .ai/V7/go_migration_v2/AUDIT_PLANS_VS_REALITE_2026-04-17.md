# Audit Plans vs Realite - Go Migration et No-Streamlit

> Date : 2026-04-17
> Auteur : GitHub Copilot
> Nature : audit documentaire et technique croise, sans modification de code runtime

## Objectif

Ce document repond a une question precise :

1. qu'est-ce qui est reellement fait dans les chantiers go-migration et no-streamlit ;
2. qu'est-ce qui reste vraiment a faire ;
3. quels documents sont fiables, trompeurs, ou simplement obsoletes.

L'audit distingue volontairement trois categories :

1. travail reellement implemente ;
2. travail implemente mais documente de facon incomplete ou contradictoire ;
3. artefacts historiques qui ne doivent plus etre lus comme du backlog produit actif.

## Methode

La revue a ete menee en croisant :

1. les documents de pilotage ;
2. le thought log ;
3. le code runtime ;
4. la CI ;
5. les routes, handlers, stores et points d'entree reels.

Ordre de confiance applique pendant l'audit :

1. code runtime et CI ;
2. thought log ;
3. roadmap detaillee ;
4. checklist de suivi.

En pratique, des documents peuvent etre utiles pour le pilotage, mais ils ne sont pas consideres comme sources de verite si le code ou la CI disent le contraire.

## Resume executif

### Ce que la seconde passe confirme

1. Le repo LevelUp-go-migration est avance techniquement, mais il n'est pas en etat de bascule production fermee. Le vrai reste a faire se concentre sur la fermeture du gate Sprint 36, la suppression des exemptions de contrat, et la finalisation du Sprint 44 multi-titres.
2. Le repo LevelUp-no-streamlit a bien bascule vers React/FastAPI comme runtime actif. Le principal reste a faire n'est pas fonctionnel, mais documentaire et de cleanup legacy.
3. Le plus gros risque de lecture aujourd'hui n'est pas l'absence de travail, mais la derive documentaire entre plans, checklists, thought logs et etat reel du code.

### Verdict court

1. Go migration : avancee reelle, mais pas cloturee.
2. No-streamlit : migration produit active et globalement terminee, mais documentation desynchronisee.
3. Les documents les moins fiables sont la checklist Go v2, le MIGRATION_MASTER no-streamlit pour ses liens, et le project_map no-streamlit.

## Partie I - LevelUp-go-migration

### 1. Ce qui est confirme comme reellement fait

#### 1.1 CI Go et quality gates reelles

La CI Go n'est pas embryonnaire. Elle contient bien :

1. build et tests Go ;
2. lint golangci-lint ;
3. job de couverture avec seuil ;
4. E2E React.

Preuves :

1. `.github/workflows/ci.yml` contient `go-build`, `go-lint`, `go-coverage`, `go-contract-test`, `go-golden-test`, `e2e-react`.
2. Le seuil de couverture 50% est bien defini dans `.github/workflows/ci.yml` via `Go Coverage (seuil 50%)` et un check `min 50%`.

Conclusion : certaines lectures automatiques anterieures qui concluaient a une absence de job Playwright ou de couverture Go etaient fausses.

#### 1.2 Sprint 41 n'est pas vide

Le plan detaille laisse S41 en cases ouvertes, mais le code montre qu'une partie substantielle existe deja :

1. healthcheck HTTP enrichi et route exposee ;
2. colonnes scoreboard clefs presentes dans le domaine, l'OpenAPI et les requetes ;
3. pipeline `weapon_kills` et vue `v_weapon_kills` deja branches cote migration/sync.

Cela ne veut pas dire que S41 est ferme a 100%, mais il ne faut pas le lire comme un sprint integralement a faire.

#### 1.3 Sprint 44 a deja depasse le stade purement documentaire

Le multi-titres est entame dans le runtime :

1. `TitleExtractor` est branche dans le routeur ;
2. `CurrentTitleSlug` existe dans la session et le bootstrap ;
3. `db_profiles.json` v3 title-aware est lu ;
4. `PathResolver` et `TitleRegistry` existent ;
5. une migration de namespace avec manifest JSON existe ;
6. le frontend React transporte deja le titre courant et peut appeler `POST /session/context` pour switcher.

Preuves typiques :

1. `apps/go-api/internal/api/server.go`
2. `apps/go-api/internal/api/middleware/title.go`
3. `apps/go-api/internal/domain/session.go`
4. `apps/go-api/internal/config/config.go`
5. `apps/go-api/internal/ops/migrate/migrate.go`
6. `apps/web/src/stores/appShellStore.ts`
7. `apps/web/src/lib/api/client.ts`

Conclusion : le Sprint 44 est reellement en cours, pas juste planifie.

### 2. Ce qui reste vraiment a faire

#### 2.1 Gate Sprint 36 non fermee

Le document de bascule production n'est pas vert. Quatre criteres restent explicitement ouverts :

1. `parity_check.py = 0 diff sur 24 endpoints` ;
2. `15 specs Playwright = vert` ;
3. `Onboarding E2E (auth -> home) = vert` ;
4. `Couverture Go >= 50%` dans le runbook de bascule.

Nuance importante :

1. la CI contient bien un garde-fou de couverture a 50% ;
2. mais le gate documentaire de bascule ne l'annonce pas comme valide ;
3. donc il reste au minimum un probleme de preuve de fermeture, meme si une partie de la realite technique est deja en place.

Verdict : la migration Go n'est pas "prete prod" au sens du gate Sprint 36.

#### 2.2 Exemptions de contrat encore presentes

Le test de contrat Go admet encore des divergences explicites dans `notYetImplemented` :

1. `GET /api/v1/players/{*}/pages/citations` ;
2. `GET /api/v1/players/{*}/pages/commendations` ;
3. `GET /api/v1/players/{*}/pages/media` ;
4. `GET /api/v1/players/{*}/pages/synthesis` ;
5. `POST /api/v1/players/{*}/pages/match-history/export` ;
6. `GET /api/v1/directory/gamertags/search`.

Interpretation :

1. certains endpoints existent des deux cotes, mais pas avec la meme methode ;
2. d'autres restent conditionnels ou incomplets ;
3. le contrat Go n'est donc pas encore totalement substituable au contrat de reference.

Verdict : c'est un vrai reste a faire technique, pas juste un probleme de documentation.

#### 2.3 Sprint 44 avance, mais contrat encore intermediaire

Le chantier multi-titres est deja tres entame, mais plusieurs criteres de sortie restent non fermes.

Trois points ressortent :

1. le routage OpenAPI ne porte pas encore `title_slug` dans les paths ; le runtime passe par `X-LevelUp-Title` (middleware `TitleExtractor`) puis par `sess.CurrentTitleSlug` en fallback. Nuance : cette strategie header-plus-session est potentiellement un choix d'architecture assume dans l'ADR Sprint 44 (voir `ADR_S44_MULTI_TITLE_NAMESPACE.md`), pas forcement un ecart a corriger. A confirmer en croisant avec la doc ADR avant de trancher ;
2. `POST /session/context` ne renvoie pas encore le bootstrap complet annonce par la doc de Sprint 44 ;
3. `JobMeta` reste une `map[string]any`, alors que la doc vise une structure plus durcie.

Conclusion : le multi-titres est reellement implante a la verticale, mais pas encore durci jusqu'au contrat final annonce. Le point (1) merite une verification ADR avant d'etre comptabilise comme dette technique.

> **Mise à jour Sprint 49 (2026-07-25)** : les 3 points sont désormais résolus :
> 1. ADR confirmée définitive — le routage header+session est le choix d'architecture assumé (pas dette) ;
> 2. `POST /session/context` retourne désormais `available_titles` + `current_title_slug` ;
> 3. `JobMeta` converti en struct typée avec `TitleSlug string` + `Extra map[string]any`.

#### 2.4 La checklist Go v2 est obsolete comme outil de suivi

La checklist v2 continue d'annoncer un programme tres en retard, jusqu'a parler de Sprint 5 a ouvrir, alors que :

1. le roadmap v2 marque S29-S41 termines, sauf S36 ;
2. le thought log documente des travaux bien au-dela ;
3. le code et la CI confirment une base beaucoup plus avancee.

Verdict : il faut cesser de lire cette checklist seule comme source de verite.

### 3. Ce qui etait trompeur mais ne constitue pas un "reste a faire" reel

#### 3.1 L'absence de Playwright CI etait un faux negatif

La CI comporte bien un job `e2e-react`.

#### 3.2 Sprint 41 n'est pas un sprint vierge

Une partie du scoreboard, du healthcheck et du pipeline weapons existe deja.

#### 3.3 Le repo n'est pas dans un etat "docs only"

Le thought log est en partie trop optimiste, mais il ne fabrique pas ex nihilo des sprints inexistants : plusieurs briques annoncees sont bien presentes dans le code.

### 4. Priorites actionnables cote Go

#### Priorite 1 - Fermer la realite de bascule Sprint 36

1. executer et documenter la parite 24 endpoints ;
2. valider les specs Playwright et l'onboarding E2E ;
3. resynchroniser `BASCULE_GO.md` avec les preuves reelles.

#### Priorite 2 - Supprimer les exemptions de contrat

1. realigner methodes GET/POST ;
2. fermer `match-history/export` ;
3. clarifier le statut de la recherche gamertag en mode demo et en mode reel.

#### Priorite 3 - Finir proprement Sprint 44

1. trancher entre routage par header/session et routage path title-aware ;
2. faire converger `POST /session/context` vers le contrat de re-bootstrap annonce ;
3. structurer les metadonnees jobs et les garanties d'isolement inter-titres.

## Partie II - LevelUp-no-streamlit

### 1. Ce qui est confirme comme reellement fait

#### 1.1 Le runtime actif est bien React/FastAPI

Le point d'entree n'est plus Streamlit.

Preuves :

1. `src/utils/launcher_startup.py` declare `_launch_react` comme point d'entree principal ;
2. ce launcher demarre `uvicorn` pour `apps/api` et `npm run dev` pour `apps/web` ;
3. `streamlit_app.py` est marque `ARCHIVED` et indique explicitement que le front actif est React/Vite.

Verdict : la migration produit active vers React/FastAPI est bien en place.

#### 1.2 Le thought log confirme la fin de migration produit

Le thought log documente :

1. le passage canonical des slices React ;
2. la fin de migration React/FastAPI ;
3. le fait que le cleanup Streamlit residuel est optionnel.

Verdict : le noyau produit actuel ne semble pas etre dans un etat de migration inachevee.

### 2. Ce qui reste vraiment a faire

#### 2.1 MIGRATION_MASTER pointe vers les mauvais fichiers

Le document maitre React/FastAPI est trompeur sur ses liens :

1. il affirme que la migration est terminee ;
2. mais il reference systematiquement `migration/*.md` (chemin relatif, minuscules, sans espace) dans ses liens markdown, alors que les vrais documents vivent sous `.ai/V7/Migration React/` (majuscule et espace dans le nom de dossier).

Verification ponctuelle : les liens `[migration/DECISIONS.md](migration/DECISIONS.md)`, `[migration/FUNCTIONAL_SPECS.md](...)`, `[migration/API_CONTRACTS.md](...)`, `[migration/SLICES.md](...)` etc. referencent tous un sous-dossier `migration/` qui n'existe pas a cote de `MIGRATION_MASTER.md`. Les fichiers cibles existent bien (`API_CONTRACTS.md`, `DECISIONS.md`, `FUNCTIONAL_SPECS.md`, `INVARIANTS.md`, `NATIVE_COMPONENTS.md`, `PARITY_MATRIX.md`, `PLAN_MIGRATION_FASTAPI_REACT.md`, `SLICES.md`, `AUDIT_CODEBASE.md`) mais sous `.ai/V7/Migration React/`.

Effet : un lecteur externe peut croire que la base documentaire detaillee n'existe pas ou a disparu, ce qui est faux. Tous les liens markdown du document maitre sont casses.

Verdict : vraie dette documentaire, a corriger rapidement. Deux options : soit renommer le dossier en `migration/` a plat (rupture de chemin, mais plus portable), soit corriger tous les liens vers `V7/Migration%20React/*.md` (url-encode de l'espace).

#### 2.2 project_map est desynchronise du runtime reel

Le `project_map.md` de no-streamlit decrit encore un etat Streamlit v5.7 et conserve Streamlit comme points d'entree, alors que le runtime actif est React/FastAPI.

Effet :

1. confusion pour les agents et pour les reprises de contexte ;
2. mauvais ordre de lecture ;
3. mauvaise hierarchisation entre produit actif, archive Streamlit et corpus Go documentaire.

Verdict : vraie dette documentaire structurante.

#### 2.3 `SPRINT_EXPLORATION.md` manque dans les deux repos

Les `CLAUDE.md` des deux repos (go-migration et no-streamlit) instruisent tous les agents IA de consulter `.ai/SPRINT_EXPLORATION.md` avant toute action. Or le fichier est absent des deux cotes :

1. `LevelUp-go-migration/.ai/SPRINT_EXPLORATION.md` : MISSING ;
2. `LevelUp-no-streamlit/.ai/SPRINT_EXPLORATION.md` : MISSING.

Ce n'est donc pas un probleme specifique a no-streamlit, c'est un ecart transversal : les deux `CLAUDE.md` prescrivent une lecture obligatoire d'un document inexistant.

Verdict : ce n'est pas un manque produit, mais c'est un trou de gouvernance documentaire partage par les deux repos. Deux reparations possibles : soit creer le fichier dans chaque repo, soit retirer la reference des deux `CLAUDE.md`.

#### 2.4 Cleanup legacy Streamlit encore optionnel

Le thought log garde explicitement un reliquat optionnel : suppression de `streamlit_app.py`, `streamlit_app_v7.py` et des pages Streamlit pures qui ne servent plus au produit actif.

Verdict : ce n'est pas un blocage de migration, mais c'est un reste a faire legitime si l'objectif est un repo plus net.

### 3. Ce qui ne doit plus etre lu comme backlog actif

#### 3.1 `IMPL_V7.md` est un plan historique de cockpit Streamlit

Ce document est encore utile pour comprendre la logique V7 historique, mais il ne decrit pas le produit actif React/FastAPI.

Indices clairs :

1. branche cible `v7/cockpit` ;
2. creation de `streamlit_app_v7.py` ;
3. phases B2/C centrees sur sidebar, shell Streamlit et hubs Streamlit.

Conclusion : ses cases restantes ne doivent pas etre interpretees comme du "reste produit" courant pour no-streamlit.

#### 3.2 Le corpus Go dans no-streamlit n'est pas du backlog runtime du repo

Le dossier `.ai/go_migration_v2/` existe bien dans no-streamlit, mais il s'agit d'un corpus documentaire de migration Go, pas d'une fonctionnalite runtime active de ce repo.

Conclusion : il ne faut pas confondre presence documentaire et chantier d'implementation local.

### 4. Priorites actionnables cote no-streamlit

#### Priorite 1 - Resynchroniser la documentation d'entree

1. corriger les liens de `MIGRATION_MASTER.md` ;
2. remettre `project_map.md` en phase avec React/FastAPI comme front canonique.

#### Priorite 2 - Clarifier le statut du legacy Streamlit

1. soit garder explicitement les fichiers archives avec une politique de conservation ;
2. soit supprimer le residuel inutile encore mentionne comme optionnel.

#### Priorite 3 - Fermer le trou documentaire agentique

1. ajouter ou retirer de la documentation obligatoire la reference a `SPRINT_EXPLORATION.md` dans les deux repos (go-migration et no-streamlit) de maniere coordonnee, puisque les deux `CLAUDE.md` portent la meme prescription.

## Partie III - Constat transversal sur la gouvernance documentaire

### 1. Probleme principal

Le probleme le plus saillant n'est pas un manque uniforme d'implementation, mais une derive entre :

1. documents de pilotage ;
2. code reel ;
3. CI ;
4. thought logs ;
5. runtime actif.

### 2. Regle de lecture recommandee

Pour les revues futures, l'ordre recommande est :

1. runtime reel et points d'entree ;
2. CI ;
3. tests/contract tests ;
4. thought log ;
5. roadmap ;
6. checklist.

### 3. Documents a traiter avec prudence

#### A ne pas lire seuls

1. `LevelUp-go-migration/.ai/go_migration_v2/GO_MIGRATION_CHECKLIST.md`
2. `LevelUp-no-streamlit/.ai/MIGRATION_MASTER.md`
3. `LevelUp-no-streamlit/.ai/project_map.md`

#### Plutot fiables pour l'etat reel

1. `LevelUp-go-migration/.github/workflows/ci.yml`
2. `LevelUp-go-migration/apps/go-api/internal/api/contract_test.go`
3. `LevelUp-go-migration/docs/BASCULE_GO.md`
4. `LevelUp-go-migration/.ai/thought_log.md`
5. `LevelUp-no-streamlit/src/utils/launcher_startup.py`
6. `LevelUp-no-streamlit/streamlit_app.py`
7. `LevelUp-no-streamlit/.ai/thought_log.md`

## Synthese finale

### Go migration

1. Le chantier est reel et deja tres avance.
2. Il reste du travail technique important avant de pouvoir parler de fermeture propre : gate de bascule, exemptions de contrat, fin de Sprint 44.
3. La checklist v2 ne doit plus etre utilisee comme tableau de bord principal sans resynchronisation forte.

### No-streamlit

1. Le produit actif est bien React/FastAPI.
2. Je ne vois pas de reste a faire critique equivalent a un backlog de migration fonctionnelle non realise.
3. Le principal reste a faire est documentaire, plus un cleanup legacy optionnel.

### Recommendation pratique

Si un seul plan d'action doit sortir de cet audit, il devrait etre :

1. cote Go : finir les preuves de substituabilite ;
2. cote no-streamlit : nettoyer la facade documentaire ;
3. cote gouvernance : aligner les documents d'entree sur la verite du runtime, pas l'inverse.

## Limites de l'audit

1. Aucun test n'a ete execute pendant cette passe ; l'audit repose sur la lecture croisee des plans, du code, du runtime declare et de la CI.
2. L'audit ne pretend pas recalculer la couverture Go reelle a partir de `coverage.out` ; il constate seulement l'existence du garde-fou CI et l'absence de fermeture documentaire du gate Sprint 36.
3. Les statuts Sprint 42 et Sprint 43 n'ont pas ete revalidees case par case ; la conclusion porte sur les ecarts structurels les plus fiables, pas sur chaque sous-tache du plan.
