# Plan — ADR 0030 (agregats persist) + ADR 0031 (frontiere source de donnees par titre)

> **STATUT : COMPLETE (2026-07-11)** — 2 ADRs rediges (Proposed), index CLAUDE.md + MT-27
> a jour, journal a jour. Chantier DOC-ONLY : aucun fichier de code modifie (verifie).
> Livrables code (allowlist D-3, ratchet lecture D-4, batch opaque D-1/D-5, httpx D-2) =
> ACTES dans les ADRs, implementation en lots futurs (hors perimetre de ce plan).
> Commits : ADR 0030 (54c181b4c), ADR 0031 (a8c8feb5e), cloture (ce commit).
> Deplace vers `.ai/V7/` au commit de cloture.

## Contexte

Discussion DDD → constat en 3 explorations :
1. **Invariants ART proteges au runtime seulement** : `internal/persist/` (BatchBuilder, Persisters INSERT-only, WAL) tient les invariants ADR 0019/0026, mais rien au compile-time — batchs mutables post-Submit, bypass possibles via `platform/duckdb.OpenReadWrite()` brut, cache process-wide, ecritures post-sync directes (`internal/sync/writes.go`, `career.go`, `engagement.go`, `performance.go`). Et **aucun garde-rail sur la lecture brute des tables append-only** (convention `_latest` non verifiee — seul trou sans filet).
2. **Frontiere client API floue** : le client HTTP Infinite (`internal/sync/halo_client*.go`, ~1380 L) vit dans la couche sync ; le client H5 (`internal/games/halo_5/client.go`, zero import LevelUp) est propre mais duplique ~350 L de retry/backoff/rate-limit (constantes identiques 4/800ms/10s, `HTTPError` declare 2x).
3. **Pas d'interface « source de donnees par titre »** : V2 cycle orchestrator (ADR 0027, statut *Proposé*) est mono-titre Infinite ; H5 contourne via `livesync.Runner`. Deux orchestrations coexistent sans couture commune — cout direct pour un 3e titre.

**Cadrage utilisateur (decide)** : livrable = ADRs d'abord (execution code planifiee apres) ; frontiere clarifiee EN INTERNE (packages nets), pas d'extraction repo.

**Branche** : `docs/adr-aggregates-title-boundary` creee depuis `refactor/audits-2026-07` (reference l'etat de la branche d'audit, ex. lint TODO(expiry) commit 2de54a072). Jamais sur main.

## Decisions actees (a inscrire telles quelles — aucune decision ouverte a l'execution)

### ADR 0030 — Persist write aggregates (compile-time enforcement)
- **D-1 Mecanisme** : privatiser les champs de `MatchBatch`/`SharedBatch`/`PlayerBatch` dans `internal/persist` (types opaques, construction uniquement via `BatchBuilder`). Pas de nouveau package.
- **D-2 Pilote** : famille **PlayerEnrichment** d'abord (plus petite surface : `post_sync_enrichment_persister.go` ; c'est la que vivent les bypass reels ; invariant stage merge-on-read le plus subtil). SharedMatch = generalisation ulterieure ; la sequence de lease cross-DB est renvoyee a ADR 0027.
- **D-3 Bypass `OpenReadWrite`** : allowlist datee + test ratchet (pattern deja institutionnalise, cf. `no_raw_outcome_literal_test.go` + lint TODO(expiry)). Allowlist initiale = etat courant, chaque entree datee, ajouts interdits.
- **D-4 Garde-rail lecture `_latest`** : test grep/AST interdisant `FROM <table_append_only>` hors persist/vues (meme harnais que `no_art_patterns_test.go`). Ecarter le revoke DB (DuckDB embarque : pas d'ACL multi-roles — justification a ecrire). Le helper de lecture type = role de D-MV2 (ADR 0025), referencer sans dupliquer.
- **D-5 Immutabilite post-Submit** : transfert de propriete (`Build()`/`Submit()` one-shot, builder invalide apres usage) — l'erreur devient impossible a ecrire, zero cout memoire.
- **Positionnement obligatoire vs ADR 0013** : 0013 a rejete le « strict compile-time guard via signature changes » (cascade `port.DBExecutor` dans les repos metier). 0030 encapsule la construction de batch A L'INTERIEUR d'un seul package — pas de cascade. Paragraphe explicite requis, sinon contradiction apparente.

### ADR 0031 — Title data-source boundary & sync mutualization
- **D-1 Packages client** : deplacer `halo_client*.go` vers `internal/games/halo_infinite/client/` (symetrie halo_5, esprit ADR 0012). Core partage dans `internal/platform/httpx` (infra, pas domaine). Regle ecrite + guard import archlint : les clients titre n'importent QUE stdlib + `x/time/rate` + `platform/httpx` (lui-meme leaf).
- **D-2 Core HTTP minimal** (~150 L) : retry + backoff expo + `rate.Limiter` + `HTTPError` unifie ; auth par titre via `RequestDecorator func(*http.Request)`. Pas de client generique a options. (De-duplication constatee, pas abstraction speculative — la regle des 3 occurrences ne s'y oppose pas.)
- **D-3 Interface `TitleSyncRunner`** calquee sur l'existant : `TitleSlug() string ; RunCycle(ctx, players []Profile) (CycleReport, error)`, enregistrement capability-based (`HandlesTitle`). Ecarter explicitement l'interface fine niveau source (forcerait Infinite a produire `canonical.MatchSummary` qu'il saute — chantier de 0027, pas de 0031).
- **D-4 Delta/watermark** : extraire un `KnownSet`/KnownLoader partage (impl V2 existante) ; H5 remplace son `isKnown` local. Un seul concept « matchs connus par (title, player) ».
- **D-5 Articulation orchestrations** : cible = V2 parametre `titleSlug` (renvoi a 0027, que 0031 **amende** — noter son statut *Proposé* et proposer passage « Accepté (amendé) »). `livesync.Runner` = adaptateur transitoire derriere D-3. Interdiction d'une 3e architecture. **Aucune promesse de migration H5→V2 dans 0031** (ressort de 0027).
- **Registre MT** : MT-03 est deja pris (world leaderboard, done). Creer **MT-27** « interface source de donnees par titre » dans le registre.

## Etapes (ordre strict, gate avant l'etape suivante — contrat skill `plan-execution`)

### Etape 1 — Rediger `docs/adr/0030-persist-write-aggregates.md` (EN-only, pas d'emojis) — CLOSE 2026-07-11
- [x] Statut *Proposed* ; Context : bug ART #23046, invariants runtime 0019/0026, les 3 vecteurs de bypass avec chemins de fichiers, trou lecture `_latest` (verifies sur pieces : batch.go champs exportes, builder.go:185 Build, queue.go:167 Submit, db.go:286 OpenReadWrite ~25 sites, sync/{writes,career,engagement,performance}.go)
- [x] Decisions D-1..D-5 avec options ecartees et justifications (revoke DB ecarte D-4 ; DTO/MarshalJSON note D-1)
- [x] Paragraphe de positionnement vs ADR 0013 (pas de cascade de signatures — encapsulation mono-package)
- [x] References croisees exactes : ADR 0019, 0026, 0013, 0025/D-MV2, `internal/persist/doc.go` (section « Hors scope MatchBatch »), `no_art_patterns_test.go` + `append_only_state_guard_test.go` + `no_raw_outcome_literal_test.go` + `todo_expiry_test.go`
- [x] Consequences + esquisse d'execution (pilote PlayerEnrichment, generalisation SharedMatch) marquee « future lots, hors ADR »
- **Gate** : PASSE — fichier existe ; grep refs 0013=7/0019=5/0026=5 > 0 ; zero emoji (seuls des tirets cadratins en titres, comme ADR 0026) ; 15 chemins de fichier cites verifies existants.

### Etape 2 — Rediger `docs/adr/0031-title-data-source-boundary.md` (EN-only, pas d'emojis) — CLOSE 2026-07-11
- [x] Statut *Proposed* ; Context : dualite d'orchestration (V2/CycleOrchestrator vs livesync.Runner), duplication HTTP **re-chiffree sur pieces** : Infinite `halo_client_http.go`=219 L (deja migre en sous-package `haloclient/`, pas racine sync — DECOUVERTE) ; H5 `client.go`=450 L dont ~160 L retry/backoff/rate/HTTPError (doGet 301-354, doGetBinary 361-415, waitRetry 419-431, parseRetryAfterSeconds 436-450) ; constantes IDENTIQUES (4/800ms/10s) verifiees ; `HTTPError` declare 2x (haloclient:62 + halo_5:58) ; client H5 zero-import LevelUp confirme (stdlib + x/time/rate)
- [x] Decisions D-1..D-5 avec options ecartees (interface fine niveau source ecartee → 0027)
- [x] Section « Amends ADR 0027 » (statut Proposé → proposer Accepté amendé ; l'ADR ne modifie PAS le statut de 0027 lui-meme = decision humaine)
- [x] Sequencement note : client move + httpx AVANT Phase 1.6 (pool auth) — `RequestDecorator` = point d'accroche du pool par titre ; multi-titre V2 reste gate par Phases 1.5/1.6 (ADR 0025) ; INTERDICTION d'une 3e architecture ; AUCUNE promesse migration H5->V2
- **Gate** : PASSE — fichier existe ; refs 0027=15/0012=3/0025=5/0008=2 > 0 ; zero emoji ; 6 chemins cites verifies ; chiffres de duplication mesures par `wc -l` (219/450) et `sed` sur les constantes, non repris de rapports d'agents.

### Etape 3 — Mises a jour d'index et journal — CLOSE 2026-07-11
- [x] `CLAUDE.md` : 0030/0031 ajoutes a la liste des ADRs (section « Decisions architecturales »)
- [x] `.ai/V7/PLAN_MULTITITRE_INDEX.md` : entree MT-27 (interface source par titre → ADR 0031), Registre B, marquee « nouvel axe 2026-07 hors audit initial »
- [x] `.ai/thought_log.md` : entree datee [2026-07-11] (date reelle d'execution ; le plan indiquait [2026-07-03], placeholder du redacteur — cf. Decouvertes), statut Complété, decision principale (2 ADRs, pilote PlayerEnrichment, frontiere client interne), prochaines etapes
- **Gate** : PASSE — `grep 0030 CLAUDE.md`, `grep MT-27 .ai/V7/PLAN_MULTITITRE_INDEX.md`, `grep "2026-07-11" .ai/thought_log.md` non vides ; `git status` = seulement docs/adr/, CLAUDE.md, .ai/ (aucun code). Commit autonome (mission : pleine autonomie, commit au fil).

## Regles d'execution
- Statuts d'item : `[x]` fait / `[~]` couvert ailleurs (reference) / `[!]` non traite (justification ecrite). Aucune case vide a la cloture.
- Zero fix opportuniste hors perimetre — consigner dans « Decouvertes » ci-dessous.
- Reprise de session : lire ce fichier + `.ai/thought_log.md` ; l'avancement = etat des cases.

## Verification finale
- Docs uniquement : pas de tests Go a lancer. Relire chaque ADR contre la checklist ADR existante (format des ADRs 0026/0027 comme modeles de structure).
- Verifier qu'aucun fichier de code n'a ete modifie (`git status` : seulement docs/adr/, CLAUDE.md, .ai/).

## Hors perimetre (suites recommandees, a planifier APRES acceptation des ADRs)
Ordre d'execution recommande : (1) pilote 0030-D2 PlayerEnrichment (lot dedie apres cloture du lot E — eviter le churn sur `builder.go` pendant l'audit) avec golden byte-identique ; (2) 0031-D1/D2 pure move (`git mv`, zero diff logique) + httpx, commit separe du cablage ; (3) 0031-D3/D4 interface + KnownSet ; (4) generalisation 0030 SharedMatch. V2 multi-titre gate par Phases 1.5/1.6.

## Decouvertes
(consignees pendant l'execution — NON traitees, hors perimetre doc-only)

1. **Client Infinite deja en sous-package `internal/sync/haloclient/`** (le plan supposait
   `internal/sync/halo_client*.go` a la racine sync). ~2386 L non-test ; core retry/backoff
   dans `halo_client_http.go` (219 L). La cible ADR 0031-D1
   (`internal/games/halo_infinite/client/`) reste valide ; le move sera un `git mv` du
   sous-package, pas de fichiers epars.
2. **Tension D-1 (champs prives) vs serialisation JSON du WAL** : `encoding/json` ne marshale
   pas les champs non-exportes ; l'implementation du batch opaque devra posseder la
   serialisation (DTO prive ou `MarshalJSON`/`UnmarshalJSON` custom, WAL byte-compatible,
   couvert par `batch_roundtrip_test.go`). Note dans l'ADR 0030 (D-1) ; pas un blocage.
3. **`HTTPError` declare 3x** (pas 2x) : `haloclient/halo_client.go:62`, `halo_5/client.go:58`,
   ET `internal/presence/rest_client.go:140`. Les 2 clients Halo sont la cible D-2 ; le client
   presence est une surface distincte (REST Xbox presence), hors perimetre 0031.
4. **`OpenReadWrite` = ~25 sites non-test** (mesure pour l'allowlist initiale D-3, a seeder au
   lot d'implementation) : legitimes (persister, provider_writer, pool, schema bootstrap) +
   repos post-sync. Non traite ici (doc-only).
5. **Lint `TODO(expiry:YYYY-MM-DD)`** = `internal/archlint/todo_expiry_test.go` (pas un lint
   externe) — referencable pour le pattern de dette datee (D-3).
6. **Gate etape 3 du plan grep `"2026-07-03"`** : date placeholder du redacteur. Execution
   reelle 2026-07-11 → entree thought_log datee honnetement 2026-07-11. Intention du gate
   (entree existe) satisfaite ; le grep litteral `2026-07-03` du plan est perime, remplace
   par `2026-07-11` dans le gate ci-dessus.
7. **Statut de l'ADR 0027 non modifie** : 0031 PROPOSE le passage « Accepté (amendé) » mais
   n'edite pas 0027 (engagement d'architecture long terme = decision humaine). Verification
   utilisateur requise (cf. rapport).
