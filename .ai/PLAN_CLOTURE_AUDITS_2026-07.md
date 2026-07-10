# PLAN — Clôture de la campagne d'audits 2026-07 (post-vérification finale)

> Date de rédaction : 2026-07-06. Exécutant prévu : Opus. Superviseur : Guillaume.
> Source : [AUDIT_VERIF_FINALE_2026-07-06.md](AUDIT_VERIF_FINALE_2026-07-06.md) (findings
> VF-1 … VF-15 — elle FAIT FOI pour le détail ; ce plan les ordonne en lots exécutables).
> Plan parent : [PLAN_TRAITEMENT_AUDITS_2026-07.md](PLAN_TRAITEMENT_AUDITS_2026-07.md)
> (ce plan-ci NE le remplace pas : il le TERMINE — dernier kilomètre avant le merge main).
>
> **Contrat d'exécution : skill `plan-execution`, intégralement.** Rappels non négociables :
> ordre strict des lots, un lot commencé est terminé, statuts [x]/[~]/[!] justifiés,
> vérifier sur pièces AVANT de coder et AVANT de cocher (les lignes citées ont pu bouger —
> la session J/J9 travaillait encore au moment de la rédaction), zéro fix opportuniste
> hors lot (→ §Découvertes), entrée thought_log + mise à jour de CE fichier dans le même
> commit que chaque clôture de lot.
>
> **Branche** : `refactor/audits-2026-07` (continuité — 1 tâche = 1 branche, N commits).
> Préfixe commits : `cloture(Vx):`. JAMAIS de push main sans accord (auto-deploy prod).
>
> **Pré-requis P0** : la session J en cours est terminée et son travail commité (J2/J3/J7/J9
> livrés au moment de l'audit ; J4/J6 restent différés measure-first — NE PAS les rouvrir
> ici). Working tree propre avant de démarrer V1.
>
> **PIÈGE D'OUTILLAGE (leçon VF-2)** : pour tout gate front, purger le cache incrémental
> avant de conclure : `Remove-Item -Recurse -Force node_modules\.tmp` puis
> `npm run typecheck`. Un `tsc -b` à cache chaud peut rendre un FAUX VERT. Côté Go :
> `-p 1` + filtre `^--- FAIL:` + code de sortie (acquis 2026-07-03).

## Décisions PRÉ-TRANCHÉES (ne pas re-questionner ; veto utilisateur possible avant le lot)

| # | Décision | Choix |
|---|---|---|
| DC-1 | `GET /jobs/{job_id}` | Mettre sous `RequireAuth` (no-op single-user/démo, cohérent lot S) ET rendre `newJobID()` aléatoire (crypto/rand, préfixe date lisible OK) |
| DC-2 | Types Node des 2 garde-rails front | `/// <reference types="node" />` en tête des 2 fichiers guard (PAS d'ajout global de "node" à tsconfig.app.json — ne pas exposer les globals Node au code navigateur) |
| DC-3 | Erreurs `ManifestLocale` | Typer les props `locale` en `ManifestLocale` À LA SOURCE (remonter le type), pas de cast aux bords |
| DC-4 | Hook Prestige | CÂBLER (pas retirer) : Prestige est acté ON (C7/ADR 0005). Implémenter le stub + brancher le chemin engine. Si l'investigation révèle une redondance réelle avec un autre chemin vivant, repli = retirer stub+call et documenter — mais décision par défaut : câbler |
| DC-5 | Trio `sync/writes.go` + `insertHighlightEventsFromData` | SUPPRIMER (fonctions + tests + entrées allowlist ON CONFLICT devenues sans objet). Les tests `concurrent_upsert_*` qui caractérisent l'ancien UPSERT ART : supprimer aussi SI leur seul sujet est le code supprimé (vérifier sur pièces ; le tripwire no_art_patterns reste le garde du pattern) |
| DC-6 | `coverage.html` / `coverage_baseline.txt` | Dé-tracker coverage.html (`git rm --cached` + .gitignore). `coverage_baseline.txt` : VÉRIFIER s'il est consommé par un script/CI (coverage_filter.sh) — si oui le garder, sinon dé-tracker aussi |
| DC-7 | `ORDER BY r.start_time` brut Q29/Q29Bulk | Migrer les 2 sites vers `StartTimeCanonicalSQL("r")` (mêmes 50 matchs pour les 2 variantes = pas de divergence bulk/unitaire) + étendre le garde-rail H1 d'un second regex `ORDER BY\s+\w*\.?start_time` (allowlist migrations) |
| DC-8 | Ratchet « routes nues » | LIVRER le test chi.Walk → allowlist datée des routes publiques (login, callback, health, session/context, SPA, static). C'est le garde-rail qui aurait attrapé /jobs (règle 6 : pas de fix sans garde-rail) |

## Vue d'ensemble (ordre d'exécution)

| Lot | Thème | Findings | Effort |
|---|---|---|---|
| V1 | Gate cassé : typecheck front | VF-2 | 0,5 j |
| V2 | Bug fonctionnel : hook Prestige post-sync | VF-1 | 0,5-1 j |
| V3 | Sécurité : /jobs + ratchet routes nues | VF-3, VF-15 | 0,5 j |
| V4 | Code mort + allowlists mortes | VF-5, VF-6, VF-9 | 0,5 j |
| V5 | Garde-rail halowaypoint + docs/commentaires inversés | VF-7, VF-8, VF-12, VF-10 | 0,5 j |
| V6 | Tracker/journal/DETTE + vérification finale §5 du plan parent | VF-4 | 0,5-1 j |
| V7 | Résiduel qualité (petits fixes bornés) | VF-11, VF-13, VF-14 | 0,5 j |
| V8 | Contrat front↔back (types main vs openapi) | découverte A2 §7 généralisée | 0,5-1 j |
| V9 | Données de prod (audit sur copie backup, puis correctifs) | dettes data connues hors audits | 1-2 j |
| V10 | Exploitation : restore-test restic + checklist deploy (PAS d'alerting uptime — décision user 2026-07-06) | angle ops | 0,5-1 j |
| — | GATE HUMAIN (revue visuelle utilisateur) puis PLAN DE MERGE | — | — |

Ordre : V1→V7 (code) → V8 → V10a (restore backup = fournit la copie prod) → V9a (audit
data sur la copie) → GATE HUMAIN → PLAN DE MERGE → V9b-e + V10c (post-merge, sur prod).

---

## LOT V1 — Réparer le typecheck front (BLOQUANT : gate transversal rouge)

Objectif : `npm run typecheck` = 0 erreur, CACHE PURGÉ, et le rester.

- [x] V1a — VF-2 : les 6 TS2345 `ManifestLocale` (DC-3) résolus À LA SOURCE, 0 cast.
  Props/paramètres `locale` remontés de `string` → `ManifestLocale` :
  `SessionMultiSelect.tsx` (prop + `getTexts` + default param du test),
  `HomeCitationsNearCompletion.tsx` (prop `NearCompletionTile`),
  `LeaderboardBlock.tsx` (prop de la row — couvre 419/502/544 par 1 seul typage),
  `MediaPage.tsx` (`buildSessionGroups` + `buildGroups`). Tous les call-sites source
  tirent du store `appShellStore.locale` (déjà `'fr' | 'en'`) → typecheck sans cast.
- [x] V1b — VF-2 : `queryKeys.mediaMatchCandidates` — signature élargie à
  `filePath: string | null` (variante « clé STABLE »). Pour un `filePath` non-null la clé
  reste byte-identique (rien inséré/retiré) ; le null n'est jamais fetché (`enabled:!!filePath`).
  Commentaire ajouté dans keys.ts.
- [x] V1c — VF-2 : `/// <reference types="node" />` en 1re ligne de
  `calendar.guard.test.ts` et `keys.guard.test.ts` (DC-2). EFFET DE BORD constaté (→ Découvertes,
  traité car bloquant le gate) : la directive tire `@types/node` dans TOUT le programme tsc
  (elle n'est PAS file-scopée), ce qui bascule `setTimeout` sur la surcharge Node dans
  `CoverFlowModal.tsx:492` (retour `Timeout` ≠ `number`). Corrigé en 1 ligne (`window.setTimeout`
  → `setTimeout`, runtime identique navigateur) + commentaire. Typecheck = 0 (cache purgé).
- [x] V1d — Piège `tsc -b` incrémental documenté dans `delivery-checklist` §2 Frontend
  (purge `node_modules\.tmp` avant conclusion, symétrique du `-p 1` §1) + critère go/no-go.
- [x] V1e — VF-16 : baseline rebaselinée SUR PIÈCES. Le log CI a été mal diagnostiqué au
  départ (« déplacements lot K ») : sur les 537 tests top-level absents, 427 sont des
  RELOCATIONS lot K (func existe encore, ex. `TestGetLUSRChain`→`internal/sync/skill`) et
  110 sont des SUPPRESSIONS légitimes documentées — chacune tracée à son commit
  (`8daee9fed` G2 home legacy ×48, `d4343dce4` D1b ×20, `25f9c3581` G5 media-notif ×11,
  `bb1ba3422` G3 session-compare ×9, `75e57c6e4` L4 ContractValidate ×8, kda/E8/F3/D1e/S…).
  0 disparition orpheline, 0 fail dans la capture courante, 0 package perdu (sauf
  `internal/api/gen` supprimé en G1 + `cmd/get-token` sans tests). Rebaseline = retrait
  SUBTRACTIF pur des 688 pairs absentes (diff = 5546 lignes, 0 insertion, LF préservé,
  aucun sous-test Windows-only réintroduit). Gate rejoué localement (extraction exacte du
  script vs capture courante) : 0 missing → exit 0.
- [x] V1f — `delivery-checklist` §0 : item « vérifier les runs CI de la branche
  (`gh run list --branch`) avant de déclarer un lot clos » ajouté (leçon VF-16).

Gate V1 : `Remove-Item -Recurse -Force node_modules\.tmp ; npm run typecheck` → exit 0 ;
`npm run lint` → 0 erreur ; `npm run test` (hors sandbox) → vert ; **run CI complet VERT
sur la branche après push** (`gh run watch` / `gh run list`).

## LOT V2 — Câbler le hook Prestige post-sync (bug fonctionnel majeur)

Objectif : `prestige.RunPostSyncHook` tourne réellement après ingestion sur les chemins
sync (HTTP + auto-sync + V2), OU décision de repli documentée (DC-4).

- [x] V2a — Cartographie sur pièces faite (cf. journal ci-dessous). 4 chemins établis :
  (1) HTTP initial → `newEngineFor` → engine.run():713 ; (2) HTTP delta + (3) auto-sync +
  watcher → `scheduler.BuildEngine` → engine.run():713 ; (4) V2 cycle → `RunPostSyncForV2`
  (NE passe PAS par engine.run(), gap confirmé) → hook explicite Phase 6. Invariant
  deadlock-free respecté : V1 fire pendant le lease tenu (instance directe non-lease) ;
  V2 fire hors fenêtre RW (lease relâché en fin de RunPostSync). Identifiant = playerSlug
  (= user_id des défis ; réel PlayerSlug==Gamertag, cf. config_players.go:142/180).
- [x] V2b — `SyncHandler.WithPrestigeHook` stocke le hook (champ struct), `newEngineFor`
  câble `SyncEngine.WithPrestigeHook`. Scheduler : `AutoSyncScheduler.WithPrestigeHook` +
  câblage `BuildEngine` (couvre auto-sync + HTTP delta + watcher). V2 :
  `CycleOrchestratorImpl.WithPrestigeHook` + invocation Phase 6 par joueur réussi. DI dans
  cmd/server/main.go (bundle via `reg.PrestigeBundle()`) → scheduler + `SyncV2WiringDeps`.
  Stub `return h` + `TODO(prestige-agent)` retirés (le commentaire « Stub temporaire »
  n'était plus sur la ligne 158 mais sur la méthode ligne 226 — retiré là).
- [x] V2c — Gardes livrées, DEUX niveaux, chacune vérifiée MORDANTE (régression simulée →
  rouge, puis revert) : (1) unitaire — `internal/sync/engine_prestige_hook_test.go` (hook
  stocké + invoqué au pattern run():713) + `internal/sync/v2/cycle_prestige_hook_test.go`
  (hook invoqué par joueur réussi via spy, skip si post-sync échoue) ; (2) câblage —
  `scheduler/auto_sync_build_engine_test.go` (golden `HasPrestigeHook`) +
  `handlers/sync_handler_align_test.go` (`newEngineFor` porte le hook). Inspecteur
  `SyncEngine.HasPrestigeHook()` ajouté (engine_introspect.go).
- [x] V2d — DOCS MAJ : `prestige/sync_hook.go` (RunPostSyncHook décrit le câblage réel des
  4 chemins) + `wire/prestige_setup.go` (RunPostSync) ; entrée §7 du plan parent (D1f BUG
  latent) annotée `[TRAITÉ V2 2026-07-06]`.

Journal V2 (2026-07-06) : VF-1 confirmé — hook mort sur TOUS les chemins (pas seulement HTTP :
le SyncEngine.WithPrestigeHook n'avait aucun caller prod). Découverte MAJEURE non anticipée par
le plan : le pipeline V2 (cycle orchestrator, moteur de sync par défaut ADR 0027) NE passe PAS
par engine.run() → le hook engine ne l'aurait jamais couvert. Wiring V2 dédié ajouté
(CycleOrchestratorImpl Phase 6). DC-4 (câbler) appliqué, pas de repli. Gates V2 tous verts.

Gate V2 : `go build ./... && go vet ./...` ; `go test ./internal/prestige/... ./internal/api/handlers/... ./internal/scheduler/...` ;
`go test -tags=integration -p 1 ./internal/sync/... ./internal/api/...` → exit 0 ;
grep `TODO(prestige-agent)` → 0.

## LOT V3 — Sécurité : /jobs + ratchet routes nues

- [x] V3a — VF-3/DC-1 : `registerJobsHuma` sous `RequireAuth` (server_apiv1.go:490 — la
  ligne avait bougé de 485). Fait via `newHumaAPI(r.With(RequireAuth(cfg.DemoMode,
  cfg.AuthMode)))` : humachi adosse l'API Huma au sous-routeur gardé (héritage middleware
  confirmé sur pièces, cf. `humacore.NewAPI` + registerGamertagHuma). NoStore : pas
  présent sur cette route à l'origine → non ajouté (respect « conserver s'il y est »). Cas
  `V3 GET /jobs/{job_id}` ajouté à `TestLotS_GuardedRoutes_AnonymousUnauthorized`
  (`guard_s_test.go`) : 401 anonyme, PASS. Le package `handlers` ne peut pas appeler
  `registerJobsHuma` (cycle d'import api↔handlers) → mount minimal répliqué (garde
  court-circuite avant le handler, idem cas S existants).
- [x] V3b — DC-1 : `newJobID()` → `job_<YYYYMMDD>_<hex16>` (crypto/rand 16 octets =
  128 bits). Consommateurs vérifiés sur pièces (grep JobID/Split/Atoi/Parse → 0 parsing
  du timestamp) : seul usage ordonnant = tiebreaker `Store.List` quand StartedAt nil (cas
  dégénéré, ordre lexical grossièrement chronologique conservé par le préfixe date).
  Fallback horloge sur échec crypto/rand loggé (slog.Error, jamais d'ID vide silencieux).
  Commentaire stale `store_list_test.go:19` (« newJobID = UnixNano ») corrigé. Test
  `jobid_test.go` : format + unicité + 0 collision sur 10 000 générations → PASS.
- [x] V3c — DC-8 : ratchet `internal/api/bare_routes_ratchet_test.go`. Approche par
  MARQUAGE des middlewares (nom runtime). Une 1re tentative COMPORTEMENTALE (boot enforcement
  `DemoMode=false` + composition de chaîne + requête anonyme) a été ABANDONNÉE après échec
  CI : le boot enforcement wire des services nil → `os.Exit(1)` de validation TOML +
  nil-deref dans `NewRouter` sur Linux, crashant tout le binaire de test `internal/api`
  (VF observé en direct sur `e703d6dc7` : Go Coverage + Baseline rouges). Approche
  livrée, robuste : boot du routeur en mode DÉMO (propre, 0 dépendance réelle) ; en démo les
  gardes lot S sont NO-OP au runtime MAIS le closure du middleware reste dans la chaîne
  `chi.Walk` ; on l'identifie par `runtime.FuncForPC(...).Name()` (contient `RequireAuth`,
  `RequireAdmin`, `RequirePlayerOwnership`, `LoopbackOnly` — stable, OS-indépendant). Route
  gardée = chaîne contient un de ces marqueurs ; sinon → allowlist datée 2026-07-07
  (liveness, static toutes-méthodes, référentiels assets/titres, bootstrap/players/
  changelog/help/feed-version, directory/gamertags, auth device-flow + POST auth/session).
  Self-check anti-rot (leçon V4d) : 0 entrée d'allowlist morte/inutile. MORDANT prouvé
  2 sens : (1) jobs dégardé → rouge « GET /api/v1/jobs/{job_id} » ; (2) entrée d'allowlist
  bidon → rouge « entrée d'allowlist MORTE ». Limite documentée : `catalog/*` dep-gated
  (absent du routeur démo) → non couvert (référentiel GET, table §1) ; SPA `/*` (NotFound
  non walkable) et `/auth/xbox/*` racine (mode xbox) hors surface.
- [x] V3d — VF-15 : `.ai/LOT_S_ROUTE_GUARD_TABLE.md` rafraîchi : +`GET /jobs/{job_id}` (§2,
  gardé RequireAuth, l.490), `GET /session` (302) → `POST /session/context` (l.302, libellé
  réel + note CSRF), `/gamertags?q=` → `GET /directory/gamertags/search` (l.888), lignes
  re-pointées partout (§1-§5 post-J2/K), +`/static/*`, +note MountAdminMonitoringRoutes
  (/admin/monitoring/jobs). Section « Garde-rail automatisé (V3c) » ajoutée : la table est
  désormais adossée au ratchet (c'est lui qui mord, plus le grep manuel).

Gate V3 : `go build ./... && go vet ./...` → exit 0 ; `go test ./internal/api/...
./internal/platform/jobs/...` → exit 0 ; `go test -tags=integration -p 1
./internal/api/...` → exit 0 (0 `^--- FAIL:`) ; ratchet V3c mordant 2 sens (ci-dessus) ;
`golangci-lint --new-from-rev=HEAD` → 0 issue nouvelle (4 issues résiduelles = dette
baseline pré-existante server.go/server_apiv1.go, hors périmètre V3).

## LOT V4 — Code mort + allowlists mortes + artefacts

- [x] V4a — VF-5/DC-5 : supprimé `insertHighlightEventsFromData`
  (engine_highlight_events.go) + ses 2 tests (`TestInsertHighlightEventsFromData_*` dans
  `highlight_events_orchestration_test.go`). Sibling `ProcessHighlightEvents` (chemin
  batch/replay vivant) + son test `TestProcessHighlightEvents_ZeroEvents…` conservés,
  helper `makeBenignZlibChunk` conservé. Header fichier + commentaires stale (engine.go,
  engine_postsync.go) mis à jour (fonction supprimée).
- [x] V4b — VF-5/DC-5 : supprimé le trio `InsertRegistryIfNotExists`/`InsertParticipants`
  (+ `insertParticipantRow`)/`InsertMedals` (sync/writes.go) — 0 caller prod (import
  OpenSpartan routé via `persist.SharedPersister` depuis E1, vérifié sur pièces
  openspartan_import_service.go:342). Tests : retiré chirurgicalement les 5 tests du trio
  de `writes_test.go` (import `time` mort retiré) ; supprimé en entier les 3 fichiers
  `concurrent_upsert_stress_test.go`/`concurrent_upsert_property_test.go`/
  `concurrent_multiplayer_e2e_test.go` (sujet unique = InsertParticipants supprimé).
  Allowlists : retiré l'entrée morte `writes.go` d'`allowlistArtPatterns` (plus aucun
  ON CONFLICT DO UPDATE) ; corrigé la justification `match_registry` de
  `shared_write_guard_test.go` (MarkWeaponKillsDone, pas InsertRegistry) + retiré les
  entrées mortes `match_participants`/`medals_earned`. `UpsertSharedCSRs` NON touché.
  Baseline CI mise à jour (52 lignes retirées, 13 pairs, retrait subtractif pur).
- [x] V4c — VF-6 : purgé les entrées mortes : `sentinel_test.go` `internal/api/registry.go`
  (×2 : allowedEnvReaders + allowedDuckDBWriters, fichier supprimé au lot K, remplacé par
  wire/registry_auth.go déjà listé) ; `no_art_patterns_test.go` allowlistRawDelete
  `skill_rating_postsync_persist.go` (compactMatchSkillRankSuperseded SUPPRIMÉE + fichier
  déplacé sync/skill/, doublement mort) ; `no_attach_on_social_test.go`
  `social_persister_combined.go` (spéculative « si présent », jamais créée).
- [x] V4d — VF-6 : `TestAllowlistJustifiesEverything` couvre AUSSI `allowlistRawDelete`
  (DELETE FROM table protégée) ; self-check « entrée d'allowlist = fichier existant »
  ajouté à sentinel_test.go (3 maps chemins) et no_attach_on_social_test.go. Les 3
  self-checks PROUVÉS mordants (entrée bidon → rouge → retirée).
- [x] V4e — VF-9/DC-6 : `git rm --cached apps/go-api/coverage.html` + .gitignore.
  `coverage_baseline.txt` GARDÉ : consommé par le ratchet CI (scripts/coverage_check.sh,
  ci.yml:237, working-directory apps/go-api) ET scripts/check_coverage_ratchet.sh.
- [x] V4f — VF-12 (partiel, mécanique) : retiré la clé morte `discord_notify_new_media`
  de la fixture `discord_extra_test.go:51` (jamais lue par LoadNotifyConfig depuis G5 ;
  discord.go ne lit que sync + new_version). Test conservé.

Gate V4 : `go build ./... && go vet ./...` → exit 0 ; `go test ./...` → exit 0 ;
`go test -tags=integration -p 1 ./internal/sync/... ./internal/persist/...
./internal/platform/auth/... ./internal/platform/duckdb/...` → exit 0 (anti-ART après
suppression des tests upsert) ; grep de chaque symbole supprimé → 0 (code) ; self-checks
V4d mordants prouvés. TOUS PASSÉS (2026-07-07).

## LOT V5 — Garde-rail halowaypoint + docs/commentaires inversés

- [x] V5a — VF-7 : livré `internal/archlint/no_halowaypoint_literal_test.go` : interdit
  le littéral `halowaypoint` dans les .go non-test hors allowlist PAR FICHIER (27 entrées,
  datée 2026-07-07, décroissante) + self-check d'existence ET de présence du littéral
  (leçon V4d/VF-6). Zones vérifiées sur pièces par grep : `sync/haloclient/` (5),
  `platform/halo/` (5), `platform/auth/` (2), `domain/title/auth_descriptor.go`,
  `assets/{fetcher_gamecms,kinds}.go`, `games/halo_infinite/adapter_asset_urls.go`,
  `games/halo_5/client.go`, `halotest/fake_server.go`, `cmd/*` (9), `scripts/warm_bp_assets`.
  Morsure prouvée 2 sens (littéral bidon hors allowlist → RED ; retiré → GREEN) + self-check
  prouvé (entrée bidon → RED). But atteint : FIGER la liste (pas 0 aujourd'hui).
- [x] V5b — VF-8 : purgé `LEVELUP_CONTRACT_VALIDATE` de `docs/CONFIGURATION.md`,
  `docs/FR/CONFIGURATION.md` (les DEUX) et `.env.local.example` (bloc entier). Grep tracked
  hors `.ai/` → 0 (le middleware source n'existe déjà plus sur la branche depuis L4).
- [x] V5c — VF-12 : commentaires stale réécrits : `engine_fetch.go` (chemin batch unique,
  + 3 comments inline `insertFetchedMatch`), `engine_highlight_events.go:160` (processMatch →
  appelant convergence/replay ; la fonction morte `insertHighlightEventsFromData` a bien été
  supprimée par V4a — constaté), mentions `processMatch` (`backfill_personal_scores.go`,
  `engine.go:100`, `csr_shared_backfill.go`, `csr_writes.go` — `processMatch` = 0 restant),
  `engine.go:175` (RunBackfillLUSR → RunBackfillLUSRDryRun v2), header
  `session_compare_service.go` (décrit l'infra session-summary partagée + orphelin l.22 retiré),
  orphelin `ReassociateMedia` fin `media_service_upload.go` supprimé, `ops/healthcheck.go:8`
  (fmt.Println → slog), `eslint.config.js` (warn Phase 0 → error depuis I5), `.golangci.yml`
  (header 5 args/60 stmts/12 → 7/80/15 effectifs).
- [x] V5d — VF-10 : `//nolint:gocyclo` mensonger retiré de `startSessionPurgeLoop`
  (golangci confirme : aucun gocyclo sur cette fonction) ; fragment orphelin `server.go:126-127`
  réparé (phrase NewRouter complétée) ; nolint nu `player_repos_test.go:112` justifié (liste
  DDL plate) ; historique `sync_root_freeze_test.go:21` complété (112→106→88→80) ; bilan
  K3d/K2a du plan parent annoté (4 → 5 fichiers, server_apiv1.go nommé) ; exemption fichier
  ajoutée en tête de `server_apiv1.go` (assembleur DI séquentiel ~1290 L + condition de re-découpe).

Gate V5 : `go build/vet ./...` exit 0 ; `go test ./internal/archlint/...` vert (nouveau
ratchet inclus, morsure 2 sens prouvée) ; `go test sync/api/service/ops` exit 0 ;
grep `LEVELUP_CONTRACT_VALIDATE` tracked hors `.ai/` → 0 ; grep `processMatch` prod → 0,
`insertFetchedMatch`/`RunBackfillLUSR` résiduels tous explicitement « legacy/supprimé D1b »
(mentions historiques OK). Commit `cloture(V5):` + entrée thought_log.

## LOT V6 — Tracker, journal, dette assumée, vérification finale

Objectif : le plan parent redevient la source de vérité, puis SA dernière action (jamais
faite) est exécutée.

- [x] V6a — VF-4 : plan parent. Statuts J restés à jour (18f3c7ee7 tient). I2 → `[x]`
  (I2b complet, hashes d3fdead37/8b80c7b46/974ac5a33) ; I4 → `[x/~]` + sous-puce (ii)
  EncounterSplitBars annotée [TRAITÉ, 7f70297b8/652232dfa] ; paragraphe « RESTE (7) » de K3f
  purgé (contradiction interne) + « BILAN SESSION /goal » annoté [BILAN PÉRIMÉ] ; P1/P2
  statués `[x]` (vérif git : branche part de 3aef23396 propre) ; §6 Journal H/I/J/K/L/M/N
  ajouté (comptes/gates/hashes, dont Gate H comptes 97/33).
- [x] V6b — VF-4 : thought_log — entrée de clôture du chantier V (V1-V6, hashes, CI verte)
  ajoutée EN TÊTE.
- [x] V6c — VF-4 : DETTE_ASSUMEE — entrée « E7 » corrigée (renommée D1a→D2 ; VRAI E7 DDL
  bootstrap ajouté, condition b23/b25) ; résidus K réels ajoutés (§1 : K1a-cœur, K1b-legacy,
  K1d/K1h/K1j/K1k/K1l-reste, K1n, K2b-drain infaisable, K3a/K3b, K3f-décisions) ; §4 J
  rafraîchi (J2/J3/J7/J9 livrés ; restent J1(2)/J4/J6 + J5) ; N3(b/c/e) ajoutés (§6) ;
  footer tranché (§8 « Livrées depuis la rédaction ») + §9 « Clôture V » (V9/V10/GATE/MERGE).
- [x] V6d — VÉRIFICATION FINALE §5 exécutée (4 audits relus en diagonale, 4 passes // ).
  **0 ORPHELIN** — matrice §5 confirmée exhaustive (0 ligne à ajouter). BILAN FINAL rédigé
  en §8 du plan parent (173+29+25+CR findings couverts, 0 orphelin, différés documentés,
  gates finaux, prêt GATE HUMAIN + merge après V7/V8). Écarts traçabilité mineurs → §7.
- [x] V6e — Mémoire projet `project_plan_traitement_audits_2026_07.md` mise à jour (état
  clôture V1-V6, hashes, pointeurs plan de clôture + audit VF + DETTE §8/§9).

Gate V6 : relecture croisée — 0 item du plan parent sans statut ; 0 `[!]` sans
justification ; chaque report présent dans DETTE_ASSUMEE ; bilan final rédigé et remis.

## LOT V7 — Résiduel qualité (bornés, faibles enjeux)

- [x] V7a — VF-11 : `XboxLoginPage.tsx:390,:422` migrés vers `t('common.auth.username_label')`
  / `login_pending` / `login_action` (clés déjà présentes, FR identiques ; aucun ajout
  manifest ni rebuild requis).
- [x] V7b — VF-13/DC-7 : `Q29HistoryForAvg` (:248) ET `Q29HistoryForAvgBulkTpl` (:296)
  migrés en `var` + `ORDER BY `+`StartTimeCanonicalSQL("r")`+` DESC NULLS LAST` (identiques
  entre eux ; test bulk==unitaire `TestGetHistoryForAvgBulk_EqualsSinglePerXUID` re-passé).
  Garde-rail `no_raw_start_time_literal_test.go` étendu d'un 2e regex
  `ORDER BY \w+\.start_time([^_]|$)` (table-qualifié = toujours brut ; ne mord PAS
  l'alias nu issu du fragment canonique — queries_match.go:400 vérifié ; ni start_time_utc).
  Allowlist fichier GELÉE (20 fichiers cmd/diag/backfill/seed préexistants, keyée par
  fichier comme H1 ; queries_match.go volontairement absent). Morsure prouvée 2 sens.
- [x] V7c — VF-14 : VERDICT per-copie = 2 variantes (leçon H7), PAS 4 doublons —
  Variant A 80/65/50/35 (SessionSummaryCard, PlayerDetailPanel) = `perfScale` existant ;
  Variant B 75/60/45/30 (mapPerfVsHistoryChart, squadSessionTimelineChart) = nouveau
  `perfSessionScale`. 5e copie découverte par le garde-rail : `SessionBriefing/tier.ts`
  `getScoreTier` (Variant A) → dérive maintenant de `perfScale`. Garde-rail grep
  `perf-tier.guard.test.ts` (modèle calendar.guard) interdit toute redéfinition locale
  `perfTierToken`/échelle recopiée hors instances.ts. Morsure prouvée 2 sens.
- [x] V7d — VF-14 : `formatDateShort` `'fr-FR'` = verrou chart DÉLIBÉRÉ CONSERVÉ (décision
  I2b l.945 ferme + DETTE_ASSUMEE §2 I2b). Motif : rendu `DD/MM` numérique pur,
  locale-invariant FR/EN ; threader la locale introduirait un ordre MM/DD en 'en-US' sans
  gain i18n. Justification renforcée sur place (plus de flou) — pas de threading.
- [x] V7e — VF-15 : entrée `groups.go` de l'allowlist Huma datée
  `TODO(expiry:2026-10-01)` (migration non triviale : path params + flux invitation).
- [x] V7f — lint préexistants : `match_history_service.go:107` goconst `"loss"` →
  `duelLabelLoss`/`duelLabelWin` (même package) ; `halo_ranks_loader.go` gocyclo
  `LoadRankCatalog` 16>15 → extraction `enrichRankCatalogXP` + type `rankAcc` promu.
  Les 2 issues disparues de `golangci-lint --new-from-rev=main` ; 0 nouvelle sur mes fichiers.

Gate V7 : ✅ front (cache purgé) typecheck 0 / lint 0 err (68 warn baseline) / vitest
2073 pass (garde-rail V7c inclus) ; ✅ Go build+vet 0, `go test duckdb/archlint/service/sync`
0 FAIL, intégration `-p 1` duckdb 0 FAIL ; ✅ `golangci-lint --new-from-rev=main` : V7f
disparus, 0 nouvelle issue sur fichiers touchés (22 issues résiduelles = dette branche
préexistante hors périmètre).

## LOT V8 — Contrat front↔back (généraliser la découverte A2 §7)

Objectif : plus aucun type de réponse écrit à la main dans `apps/web/src` qui diverge du
contrat réel (chaque divergence = un `undefined` silencieux à l'écran).

- [x] V8a — Inventaire (2026-07-07) : balayage complet des types de réponse hand-written
  consommés par `api.get/post`, comparés à `generated.ts`/openapi.yaml + structs Go.
  Livrable : `.ai/INVENTAIRE_V8A_TYPES_FRONT_BACK.md` (4 divergences confirmées, toutes
  Career ; ~33 hand-written conservés = view-models/hors-Huma/richer, allowlistés V8d).
- [x] V8b — Cas PROUVÉ corrigé (2026-07-07). Réalité à l'écran : la section « Top matchs »
  ne s'affichait JAMAIS au 1er chargement (`data.top_matches_preview` absent du Go →
  EmptyStateCard) ; « voir tout » lisait `fullTopMatches.items` (undefined, réel
  `{best_matches, worst_matches}`) → toujours vide. Idem encounters (`.items` vs
  `{teammates, enemies}`). Fix : `types.ts` (retrait des 2 champs fantômes de
  `CareerPageResponse` + ré-export généré de `CareerTopMatchesResponse`/`CareerEncountersResponse`) ;
  `CareerPage`/`CareerTopMatchesTable`/`CareerEncountersSection`/`queries.ts` réalignés sur
  les endpoints dédiés (fetch d'entrée de page) et les shapes DTO réelles ; `start_time`
  ajouté à `TopMatchDTO` Go (dispo dans `TopMatchRawRow`) + openapi + `generate-types` pour
  préserver la colonne date. Vérifié : test vitest `CareerTopMatches.contract.test.tsx`
  monte CareerPage avec la réponse au shape RÉEL et prouve le rendu non-vide (map + bouton
  « voir tout »). Revue visuelle finale = GATE HUMAIN.
- [x] V8c — Migration/allowlist statuées par type (2026-07-07). La MAJORITÉ de `types.ts`
  était déjà ré-exportée du contrat (batches antérieurs). Les ~33 restants hand-written
  sont soit des view-models composites (sous-types sans schéma), soit des endpoints hors
  OpenAPI Huma, soit des types PLUS RICHES que le généré (cas L1 : ré-export cru destructif —
  ex. `SessionContextResponse`, `CareerPageResponse`, `CompareResponse`). Choix documenté
  par type dans l'inventaire ; conservés + verrouillés par l'allowlist décroissante V8d.
- [x] V8d — Garde-rail livré (2026-07-07) : `apps/web/src/lib/api/response-types.guard.test.ts`
  (modèle `keys.guard.test.ts`), interdit toute nouvelle `interface/type *Response` manuelle
  hors `generated.ts` + allowlist décroissante datée. Morsure prouvée 2 sens (ajout hors
  allowlist → rouge ; entrée d'allowlist orpheline → rouge self-check).

Gate V8 : typecheck (cache purgé) 0 ; lint 0 erreur ; vitest 242 fichiers / 2076 pass /
0 fail (garde-rail V8d + test contrat V8b inclus) ; Go build + `go test ./internal/api/...`
`./internal/service/...` verts (drift OpenAPI sans MISSING nouveau : `start_time` ajouté des
2 côtés) ; garde-rail V8d mord (2 sens). Revue visuelle Career = GATE HUMAIN.

## LOT V9 — Données de prod (audit d'abord, correctifs ensuite)

Objectif : les dettes DATA connues cessent d'être des mémoires — elles sont quantifiées
sur une copie réelle, puis corrigées. AUCUNE écriture directe sur prod hors fenêtre
convenue avec l'utilisateur (règle projet : prévenir avant toute op prod).

- [x] V9a — Audit read-only sur la copie (snapshot restic 9e96ed20, 2026-06-27) LIVRÉ :
  `.ai/AUDIT_DATA_PROD_2026-07-07.md` (chiffré par titre). Verdict : la copie est PROPRE
  sur les 3 dettes DATA connues (correctifs déjà déployés avant le backup). (1) TZ décalés
  Infinite = **0** (max T0 apparent 118s < 120s ; ~964 = chiffre historique déjà corrigé) ;
  H5 = N/A (first_joined_time NULL 100%). (2) is_ranked faux = **0** (Infinite 34/34 sur
  playlist classée flaggés ; H5 0 CSR-porteur non flaggé). (3) orphelins/doublons = 0 SAUF
  medals_earned H5 = 2149 orphelins (bruit ingestion, Découvertes). (4) colonnes DROP
  différé toutes présentes (PME known_teammates_count/friends_xuids ×8 DB ; media_files
  discord_notified ×2 shared_social). (5) watermark LUSR = dernière ligne (pas de désync de
  tête) ; 23 gaps d'intérieur (Madina 7 / JGtm 10 / Choco 6) = résidu EP ~0,8% hors V9.
  (6) counts = V10a (Inf 1780/26577, H5 3032/24208).
- [x] V9b — Correctif TZ : l'outil `cmd/backfill_first_joined_tz` a DÉJÀ un mode `--commit`
  (pas seulement diag) → aucune implémentation requise. Validé sur copie : dry-run =
  « Matchs décalés : 0 » → **0 à corriger** (déjà appliqué en prod avant le backup).
  Aucun re-backfill LUSR nécessaire (0 match affecté). Aucune écriture copie, aucun
  `.pristine`. PLAN DE MERGE : rien à rejouer en prod pour la TZ.
- [x] V9c — Correctif is_ranked : fix CODE import-time DÉJÀ présent
  (`openspartan_import_service.go::writeOneMatch` l.317-320 : RankRecap présent ⟹
  is_ranked=true) + test unitaire DÉJÀ présent (`openspartan_import_service_test.go:261-270`,
  re-passé vert). Backfill historique = migration boot `shared_backfill_is_ranked_and_season`
  + seed autoritatif ranked-playlists. Validé sur copie : **0 mismarqué** (rien à backfiller,
  aucun CSR/LUSR à recalculer). Aucune écriture copie. PLAN DE MERGE : rien à rejouer en prod.
- [x] V9d — Rebuild append-only des DROP différés PLANIFIÉ (non exécuté — décision
  opérateur) : section dédiée dans `.ai/AUDIT_DATA_PROD_2026-07-07.md` §V9d (tables PME ×8 +
  media_files ×2, recette ADR 0026 via `append_only_rebuild.go` + CHECKPOINT shared_social,
  procédure serveur arrêté, vérifs post, à combiner avec étape 2 « répétition générale » du
  PLAN DE MERGE).
- [x] V9e — `weapon_kills_v3` chiffré + reco ESCALADÉE (décision utilisateur) : v3 n'existe
  NI en prod NI sur la branche (uniquement worktree non mergé `feat/weapon-attribution-v3`) ;
  0 table `%v3%` dans la copie. Servi en prod = v2 `weapon_kills`, couverture Infinite
  **1194/1780 = 67,1%** (v3 corrigerait l'ATTRIBUTION churn, pas la couverture ; algo
  inachevé). **RECO par défaut : retirer** (supprimer branche+worktree ; dead-code museum
  évité). Aucun changement code. Détail §V9e du rapport.
- [x] V9f — Cluster « TODO P4 ADR 0006 retirer *100 » : 6 occurrences (pas ~10) datées en
  bloc `TODO(expiry:2026-12-31)` (prestige/evaluator.go, squad_service.go,
  timeseries_service_{tabs,buckets}.go, session_compare_stat_helpers.go, stats_service.go).
  Reste hors périmètre (migration unité canonique 0..1 = chantier à part). Lint
  `TestNoExpiredTODO` re-passé vert.

Gate V9 : ✅ rapport V9a remis AVANT tout (rien à corriger — copie propre). Code touché =
V9f COMMENTAIRES SEULS + dating TODO (V9b/V9c déjà en place avant le lot). `go build` +
`go vet` = 0 ; tests prestige/service/archlint/openspartan = 0 FAIL ; intégration `-p 1`
sync+service = 0 FAIL (V9c import re-vérifié). Aucune écriture sur la copie prod, aucun
`.pristine` (0 à corriger).

## LOT V10 — Exploitation (SANS alerting uptime — décision utilisateur 2026-07-06)

- [x] V10a — Test de RESTAURATION restic PROUVÉ (2026-07-07). Config découverte : repo
  `/opt/levelup/restic-repo` (disque VPS), password `/opt/levelup/.restic-password`,
  timer systemd `levelup-restic-backup.timer` (04:00 UTC), scope `data/titles` (2 titres)
  + `data/auth` + config JSON. Méthode (a) : restic LOCAL 0.18.1 via `sftp:lvelup:` +
  password injecté depuis le VPS → restore de `latest` (9e96ed20, 2026-06-27) vers
  `C:\Users\Guillaume\Downloads\Scripts\LevelUp-prod-copy\` (HORS repo git, survit à la
  session — consommée par V9a). 109 fichiers / 734.832 MiB. Toutes les DB ouvertes en RO
  (outil Go `cmd/tmpdbq`, pas de duckdb CLI local). Counts de référence pour V9a :
  Infinite shared 1780 matchs / 26577 participants (2021-11→2026-06-25) ; Infinite
  metadata 123 maps / 167 medals / 35 playlists / 9702 asset_tr ; Infinite pve 20 ;
  joueurs Infinite Madina 1182 / JGtm 958 / Chocoboflor 490 / XxDaemon 22 (mv_player_matches) ;
  H5 shared 3032 matchs / 24208 participants (2015-2023) ; H5 metadata 13 tables ;
  H5 social 33 tables ; H5 Madina 1424 matchs ; 9 tokens auth. Runbook :
  `docs/RUNBOOK_RESTORE_TEST.md`. ALARME reportée en Découvertes (backup auto jamais
  produit — voir V10-D1).
- [x] V10b — Checklist de déploiement EXÉCUTABLE écrite : `docs/RUNBOOK_DEPLOY_CHECKLIST.md`
  (structure pré-deploy / deploy / post-deploy / rollback, cases à cocher). Chaque item
  vérifié sur pièces dans le repo avant écriture : regen-demo NON destructif (rm des seuls
  stubs JSON fantômes — pas warehouse/players, incident 2026-06-05, `deploy.sh` §2a) ;
  garde anti-crash-loop `pgrep -f '[b]ackfill'` (`deploy.yml`) ; CHECKPOINT shared_social
  (ADR 0022) ; `legacy_source_used_*` sous `/debug/vars` clé `levelup`
  (`internal/observability/legacy_source.go`) ; `/debug/vars` admin-only
  (`server_apiv1.go` sous RequireAuth+RequireAdmin) ; port 8000 `/health`
  (`deploy.sh` §4) ; logs par catégorie `/opt/levelup/data/logs/*.log` (23 fichiers
  vérifiés VPS) ; rollback `git revert -m 1` + critère GO/NO-GO migrations irréversibles.
  Emplacement TODO explicite « DATE DE MISE EN PROD DE D1A » présent dans la checklist
  (arme D2 ≥ 7 j). Relue à blanc une fois (dry-run de lecture : chaque étape exécutable
  telle quelle).
- [!] V10c — POST-merge (différé par conception : nécessite la prod post-merge sous charge
  réelle). Hors périmètre de l'exécutant V10ab. Fenêtre d'observation runtime — lire
  `duckdb_pool_stats` + `duckdb_budgets` sous charge réelle (débloque J1(2)) et statuer
  J4/J6 (measure-first) avec des chiffres. À traiter après le merge (Gate J définitif).

Gate V10 (partiel — V10c différé post-merge) : restauration PROUVÉE (DB ouvertes +
counts consignés ci-dessus) ; 2 runbooks écrits ; checklist rejouée à blanc une fois.
V10c = chiffres consignés au plan parent après merge (Gate J définitif).

---

## GATE HUMAIN — revue visuelle utilisateur (AVANT merge, rien oublier)

Checklist consolidée de TOUTES les vérifications visuelles promises par la campagne
(à dérouler par Guillaume sur le dev local, FR puis EN via le switch locale) :

**PASSE 1 EFFECTUÉE PAR GUILLAUME LE 2026-07-07** — résultats ci-dessous ; anomalies
regroupées dans le LOT GH (à corriger, puis RE-PASSE ciblée sur les items GH).

- [x] A1 — CORRIGÉ (GH-1) : il n'existe PAS d'onglet « Forme ». La page Séries
  temporelles (`/players/$playerSlug/stats/timeseries`) expose 3 onglets :
  **Synthèse** (EN « Summary », défaut), **Distributions** (EN « Distributions »),
  **Progression** (EN « Progression ») — cf. `TimeseriesPage.tsx:41-43` +
  `manifests/timeseries.toml` `timeseries.tabs.*`. Les courbes de
  `TimeseriesFormCharts.tsx` (FDA / KDA value trend, « Durée de vie moyenne » / Avg
  life, « Assistances » / Assists) sont rendues dans l'onglet **Synthèse** (défaut,
  `TimeseriesPage.summary.tsx:132-170`) ET réutilisées dans **Progression**
  (`TimeseriesPage.progression.tsx`). RE-VÉRIF UTILISATEUR : Séries temporelles →
  onglet **Synthèse** (puis **Progression**), contrôler ces 3 courbes en FR puis EN.
  (Le libellé « Forme » vient d'un ancien pilote 6-onglets — commentaire historique
  résiduel `timeseries.toml:7`, sans onglet correspondant ; noté en Découvertes.)
- [x] A2 — Top matches OK (dominance flags « Débandade » etc. présents). RÉSERVE
  consignée (PAS bloquant) : doutes sur l'attribution des flags de dominance pour les
  parties À OBJECTIFS — l'archi n'est pas en place, prévue v7.1 → à nettoyer à la mise
  en place de la v7.1 (GH-2, backlog).
- [x] V8 — Top matches/rencontres s'affichent. QUESTION ouverte : les nouveaux badges
  créés lors de la page Relations y figurent-ils ? → GH-3 (investiguer, répondre, câbler
  si trou simple).
- [x] A5 — Tooltips ECharts OK.
- [x] Gate F — Pages H5 OK.
- [!] I1/I2 — **EN incomplet** (vérifié UI en anglais) : nav **L1 full FR** ; nav **L2
  full FR** (boutons « Analyse », breadcrumb « Retour ») ; drawer du scoreboard :
  **RÉGRESSION — les images des médailles ne s'affichent plus** + noms de médailles
  en FR ; Match View section « Match flow » : cards en FR. → GH-4/GH-5/GH-7.
  DEMANDE UX (asset drawer, tab bordure droite) : médailles à description TRONQUÉE →
  retirer la description, garder image + nom (la description existe déjà en tooltip
  au survol). → GH-6.
- [x] I4 — Ascension/MatchView/PrestigeSquadProgress EN OK.
- [x] G3 — Session-detail OK (nav L2 FR = GH-4).
- [!] Bloc « CSR rankings (current season) » : les noms de playlists restent en FR
  en UI EN. Cause probable : `AugmentWithActiveRankedCSRs` câblé locale="fr" statique
  au DI (H8, wire/registry_pages_explorer.go:126). → GH-8.
- [!] Match View, HEADER (haut de page) en UI EN : titre mode+map (« … on … »), date,
  durée, playlist. → GH-9 (extension superviseur 2026-07-08).
- [ ] Smoke général : à RE-passer après le LOT GH (Home, Career, Squad, Explorer,
  Sessions — FR et EN, un joueur Infinite + un H5).

## LOT GH — Corrections du GATE HUMAIN (2026-07-07, décidées par Guillaume)

Objectif : purger les anomalies vues en re-passe 1 du gate humain. Exécutant : Opus
piloté. Périmètre fermé GH-1..GH-8 ; découvertes → §Découvertes.

- [x] GH-1 — FAIT : pas d'onglet « Forme ». `TimeseriesFormCharts.tsx` exporte 3
  courbes (KdaValueTrend/AvgLifeTrend/AssistsTrend) rendues dans l'onglet **Synthèse**
  (défaut) et réutilisées dans **Progression** (les 3 onglets réels : Synthèse /
  Distributions / Progression, `TimeseriesPage.tsx:41-43`). Item A1 ci-dessus corrigé
  avec noms FR/EN exacts + chemin de nav pour re-vérification.
- [x] GH-2 — FAIT : entrée « §10. Attribution des flags de dominance — parties À
  OBJECTIFS (v7.1) » ajoutée à `.ai/V7/DETTE_ASSUMEE_2026-Q3.md` (réserve utilisateur
  2026-07-07, non bloquant, doc-only).
- [x] GH-3 — RÉPONSE FACTUELLE : **non câblé, et PAS un trou simple** (design gap →
  consigné en Découvertes, non corrigé). Faits sur pièces :
  (1) `TopMatchDTO` (`domain/career.go:194-205`) ne porte AUCUN champ badge — seulement
  `outcome_code`/`outcome_label`. Le tableau `CareerTopMatchesTable.tsx` ne rend qu'une
  pill d'issue (victoire/défaite/égalité, `outcomeBadgeVariant`) ; aucun badge
  rivalité/némésis/dominance.
  (2) La page Relations (`features/palmares/RelationsRivalryCards.tsx`) a introduit des
  CARTES de rivalité (frise des duels + écart de frags cumulé + taux de victoire par
  rival, type `RelationRivalry.duels[]`) — des cartes AGRÉGÉES par adversaire, pas des
  badges par match. Aucune donnée par-match adverse n'existe sur `/pages/career/top-matches`
  (une ligne de top match ne porte pas l'identité de l'adversaire).
  (3) Câbler la rivalité sur les top matchs exigerait : jointure backend (chaque top
  match → adversaire dominant + son état de rivalité), nouveau champ `TopMatchDTO`, rendu
  front — soit une feature (backend + DTO + front, ~1-2 j), pas une réutilisation de
  composant. Hors périmètre GH → Découvertes.
- [x] GH-4 — FAIT. Système : manifest typé `common.toml` → `generated/common.ts`
  (`node scripts/build_i18n_manifests.mjs`). Ajout de 40 clés `common.nav.*` (sections +
  onglets + aria) et `common.filters.*`/`common.period.*` (pill_label, context_aria,
  matches_count ICU plural, incompatible_tooltip, prefix_named, custom_label, dialog_aria,
  from/to, empty_title, presets 7d/30d/90d/all). `navL1Sections.tsx` : `label`→`labelKey`
  (interfaces L1Section/L1Tab) ; `NavL1.tsx` (SplitButton + inline + aria « Onglets
  {section} » via var ICU), `NavL1MobileMenu.tsx` (+ aria « Sections »), `NavL2.tsx`
  (CAREER_TABS/CAREER_TABS_H5/COMMUNITY_TABS `labelKey` + NavTabBar résout via `t`),
  `FilterOmnibar.tsx` (aria Filtres/Contexte, compteur « N matchs » ICU plural),
  `_filter_pills/FiltresPill.tsx` (span + tooltip incompat.), `PeriodePill.tsx` (trigger,
  Du/Au, dialog aria, empty title, presets), `_hooks.ts` (PERIOD_PRESETS `labelKey`),
  breadcrumb « Retour » = `MatchHeader.tsx` (match-view i18n `back`, locale threadée).
  Citations EN = « Commendations » (cohérent citations/commendations). Tests adaptés
  locale-explicite (`NavL1MobileMenu.test` labelKey ; `FilterOmnibar.test` +
  `FiltresPill.test` pin `locale:'fr'`) — 0 assertion supprimée. Manifests régénérés
  (generated/*.ts commités).
- [x] GH-5 — FAIT. (a) CAUSE RACINE (sur pièces) : NON une refonte I2/K1g/F3. La médaille
  H5 (sprite, pas de PNG par-médaille) s'affichait vide car le drawer
  (`PlayerDetailPanel.tsx` MedalsSection) rendait un `<img src={image_url}>` BRUT et
  `MedalImageURL` renvoie "" pour H5 (`games/halo_5/adapter_asset_urls.go:132`). Le commit
  `b2ed57f36` (2026-06-21, « rendu sprite médaille title-agnostic sur 4 surfaces ») a migré
  4 surfaces vers `MedalIcon` + champs sprite MAIS a OMIS le drawer + n'a jamais ajouté les
  champs sprite à `PlayerMedalRow`. Infinite (drawer) marchait (PNG). Fix : champs sprite
  ajoutés à `domain.PlayerMedalRow` (Go) + `PlayerMedalRow` (shim TS) ; `indexBulkMedalsByXUID`
  passe par `static.MedalImage(slug, id)` (sprite H5 / PNG Infinite) comme le résumé ; drawer
  migré vers `<MedalIcon>`. Test `PlayerDetailPanel.test.tsx` (fixture Infinite = img src non
  vide + fixture H5 sprite = role=img) — PROUVÉ rouge sur le code cassé (re-cassé
  temporairement : test H5 échoue, Infinite passe), puis restauré. **Portée : DRAWER
  UNIQUEMENT** (PlayerMedalRow consommé nulle part ailleurs ; les 4 autres surfaces
  utilisent déjà MedalIcon). (b) Noms FR sous EN : `lookupMedalMeta`
  (`match_view_repo_medals.go`) hardcodait un COALESCE FR-first sans locale → migré vers le
  helper canonique locale-aware `medalLabelDescCoalesceSQL(ctxkeys.Locale(ctx))` +
  `medalTranslationJoinsSQL` (EN n'injecte jamais name_fr). Corrige AUSSI la grille de
  médailles du résumé (même helper). FR par défaut préservé (ctxkeys.Locale défaut "fr").
- [x] GH-6 — FAIT. `features/asset-drawer/AssetCard.tsx` : la description tronquée
  (`<p class="line-clamp-2">`) n'est plus rendue pour `kind === 'medals'` (image + nom
  seulement). La description COMPLÈTE reste au survol via le `title` du conteneur
  (« nom — description », déjà présent). Maps/armes inchangés. Test `AssetCard.test.tsx`
  (médaille : pas de `<p>` description mais title conteneur la contient ; map : description
  toujours rendue).
- [x] GH-7 — FAIT (FRONT, clés bilingues). Les « cards » = badges d'impact
  (`MatchImpactBadgesBar`). Le titre venait de `b.label` = backend `BadgeFR`
  (`analysis/match_impact.go`, FR-only ; le moteur analysis reste inchangé — pas de
  refonte). Le badge porte déjà un `BadgeKey` stable et les DESCRIPTIONS étaient déjà
  bilingues front. Ajout d'un map `impactBadgeNames: Record<string,string>` à la i18n
  match-view (FR = libellés serveur actuels à l'octet, EN ajouté) ; le composant rend
  `t.impactBadgeNames[b.key] ?? b.label` (fallback = libellé serveur pour clé inconnue).
  Les graphes du Match flow (Dominance/Cadence/KD cumulé) utilisent déjà `t` (bilingue).
- [ ] GH-8 — CSR rankings : noms de playlists locale-aware. Le param locale de
  `AugmentWithActiveRankedCSRs` est STATIQUE ("fr") au boot (H8). Fix : résoudre la
  locale PAR REQUÊTE (ctx) sur ce chemin, OU servir les deux noms (name_fr/name_en) et
  choisir côté front — prendre le pattern le plus cohérent avec l'existant (citations).
  Vérifier aussi le chemin sync ("en" statique — correct pour la persistance ? statuer).
  **FAIT** : la closure DI `newExplorerCSRProvider` (`registry_pages_explorer.go:126`) est
  construite au boot mais EXÉCUTÉE par requête avec son ctx → `"fr"` statique remplacé par
  `ctxkeys.Locale(c)`. Le bloc Explorer CSR sert donc les noms de playlist dans la locale
  de la requête (EN = NameEN via `AugmentWithActiveRankedCSRs`). CHOIX = résolution par
  requête (pattern ctx, pas de double-nom : l'augment prend déjà un param locale). SITE
  SYNC (`sync/career.go:206`, `"en"`) STATUÉ CORRECT : c'est de la PERSISTANCE
  (`SaveCSRSnapshots`), le nom canonique EN est le bon défaut persistant ; le bloc Explorer
  flagué lit la voie LIVE (augment per-requête, désormais localisée), pas les snapshots. La
  ré-localisation à la lecture des snapshots persistés (GetCSRSnapshots, autre surface) =
  hors périmètre, notée.
- [ ] GH-9 — (EXTENSION superviseur 2026-07-08, retour utilisateur direct) Match View,
  SECTION HEADER (haut de page) non traduit sous UI EN. Vu en anglais :
  « Assassin en équipe on Bazaar » / « 06 avr. 2026, 23:40 » / « 9m 53s » / « Partie
  rapide ». Trois défauts distincts : (1) titre mode+map = label mode FR composé avec
  « on » EN = famille du bug de normalisation cross-langue (skill `halo-modes`, mémoire
  « Slayer on Forest sur Forêt ») ; source = header servi par `builders_header.go`
  (mode_name FR issu des traductions metadata) → résolution PAR LOCALE de requête
  (X-LevelUp-Locale → ctx ; EN = nom canonique API, FR = asset_translations) ;
  (2) date « 06 avr. 2026, 23:40 » figée FR sous EN → formatter front du header
  MatchViewPage (formatDate sans locale threadée / 'fr-FR' résiduel) ; (3) « Partie
  rapide » (playlist) = même famille que GH-8 (playlist locale-aware) — vérifier si même
  chemin de données ou site distinct. À traiter AVEC GH-7/GH-8 (mêmes fichiers match_view).
  **FAIT** (voie Infinite = `applyMatchHeaderMetaLabels`, `match_view_builders_header.go`,
  rendue locale-aware via `ctxkeys.Locale(ctx)` threadé depuis `buildMatchHeader`) :
  (1) map+mode : EN → MapNameEN + mode dérivé de `pair_name` EN (`ResolveModeUI(pair, nil)`)
  ; FR → MapNameFR + ModeNameFR (comportement historique préservé à l'octet). Comme map ET
  mode deviennent cohérents en langue, le compose front `buildMatchHeadingStr` (« on »/
  « sur ») n'est plus mixte. `NormalizeModeLabel` strippe toujours « on/sur <map> ».
  (2) date : `formatDateFRLong` conservé (voie canonique + tests) ; nouveau
  `formatDateLong(t, locale)` (mois EN/FR) utilisé par le header ; FR identique à l'octet.
  (3) playlist : EN → PlaylistName brut, FR → PlaylistNameFR. Test Go
  `match_view_header_locale_test.go` (EN vs FR). LIMITE : la voie H5 CANONIQUE
  (`buildCanonicalHeader`, labels via `assetLabelAndID` FR-first + `formatDateFRLong`) reste
  FR — chantier locale de l'adapter canonique distinct → Découvertes.

Gate GH : front purge cache + typecheck 0 + lint 0 err + vitest 0 fail ; si Go touché
(GH-5b/GH-7/GH-8) : build+vet+tests packages + `-tags=integration -p 1 ./internal/api/...`
exit 0 ; CI de branche verte ; RE-PASSE visuelle utilisateur sur GH-1/4/5/6/7/8.

## GATE HUMAIN — RE-PASSE 2 (Guillaume, 2026-07-08 soir, UI EN, serveur rebuildé)

Validés : GH-1 (couleurs perf OK), GH-4 (sauf Saison), GH-5, GH-6, GH-7, GH-8 (Explorer),
GH-9 (header). Anomalies restantes/nouvelles → LOTS GH2-A (bugs) et GH2-B (i18n) :

- [!] Omnibar : le filtre **Saison** a été oublié par GH-4 (encore FR). → GH2-B1
- [!] BUG FONCTIONNEL : bouton L2 « View matches » → ERREUR depuis la page Timeseries
  (vérifier les autres pages partageant ce L2). Suspect : refactor nav GH-4. → GH2-A1
- [!] Match View : onglets « Général » et « Détails » non traduits ; titre
  « Antagonistes » non traduit ; drawer : tooltip des CITATIONS non traduit. → GH2-B2
- [!] Carrière, bloc « Rankings » : playlists non traduites — surface symétrique de
  GH-8 (lecture des snapshots CSR PERSISTÉS — la découverte GH est confirmée par
  l'utilisateur). → GH2-B3
- [!] Accueil « Recent playlists » : un UUID s'affiche en 2e position (JGtm) —
  résolution de nom d'asset manquante. → GH2-A3
- [!] Accueil « Recent sessions » : « Solo »/« Escouade » hardcodés, « Matchs » FR,
  outcomes FR, « Durée de la session » FR. → GH2-B4
- [!] Accueil « Highlights » (KPI cards) : noms de cartes et playlists FR. → GH2-B5
- [!] Tuiles de matchs : noms de médailles et CITATIONS en FR (tooltips inclus) ;
  « Recent media » : noms de cartes FR. → GH2-B6
- [!] BUG FONCTIONNEL + i18n : popup de réassociation de médias ENTIÈREMENT FR et
  affiche « loading error ». Suspect bug : V1b (clé mediaMatchCandidates). → GH2-A2 + GH2-B7
- Motif de fond : l'ACCUEIL sert des libellés FR-canoniques backend — le report
  F4-site-2 (« analysis/home_locale.go littéraux FR » → K1) devient bloquant sous UI EN.
  Traiter par locale de requête (pattern GH-9), résolution UNIQUE au point d'entrée du
  builder Home plutôt que N patchs.

## LOT GH2-A — Bugs fonctionnels re-passe 2 (PRIORITÉ, avant GH2-B)

- [x] GH2-A1 — L2 « View matches » en erreur depuis Timeseries. CAUSE (PAS GH-4 :
  labels seuls) = préexistant : `MatchViewRepo` lit les faits shared via un SNAPSHOT
  Parquet immuable (`SnapshotPreferredSharedReader`), tandis que `/filters/match-ids`
  (liste du bouton) lit le shared LIVE. Un match présent en live mais absent du snapshot
  courant (récent / exclu comme "partial" au cut) → `GetMatchMeta` `sql.ErrNoRows` → 404.
  Reproduit sur JGtm (match `9a2241c5…` : 106 matchs live hors snapshot v10). FIX :
  fallback snapshot→live per-requête (`match_view_repo.go` `forceLive`) — les ~18 lectures
  shared match-immutables basculent sur le live si le match manque au snapshot. Test
  `TestGetMatchMeta_SnapshotMissFallsBackToLive` (rouge sur code cassé). Vérifié live :
  match `9a2241c5…` passe de 404 à 200 avec scoreboard/médailles complets. Toutes les
  pages partageant ce L2 (History/Sessions/Squad) utilisent la même chaîne → couvertes.
- [x] GH2-A2 — Popup réassociation média « loading error ». CAUSE (PAS V1b) = préexistant
  (feature HLS) : pour une vidéo, la galerie sert `file_path` sous forme d'URL servable
  pointant sur le playlist HLS (`…/media/files/JGtm/hls/<stem>/master.m3u8`). Le handler
  `/media/match-candidates` (et `/media/associate`) ne dépouillait PAS le préfixe → le
  lookup `media_files` ne matchait ni `file_path` (préfixe URL en trop) ni `file_name`
  (`basename`=`master.m3u8` ≠ `<stem>.mkv`) → `ErrNoRows` → 500. FIX : helper
  `mediaServableURLToStoredPath` (strip du préfixe → chemin relatif stocké) appliqué aux
  2 handlers (`media.go`), factorisé avec `urlToFilePath` (`media_paths.go`). Tests
  handler (candidates + associate, rouge sur code cassé). Vérifié live : 500 → 200 +
  5 candidats. (i18n de la popup = GH2-B7, hors lot.)
- [x] GH2-A3 — Accueil « Recent playlists » : UUID en 2e position (JGtm), UI EN. CAUSE :
  la playlist `96f32b0a-f89b-4507-83b1-bc07dd458dfa` (FR « Arène delta : Héritage ») n'a
  PAS d'entrée EN dans `asset_translations` ; `resolvePlaylistNameForLocale` retombe alors
  sous EN sur le `match_registry.playlist_name` brut = le playlist_id UUID (FR résout, EN
  non). FIX display (frontend) : `HomeRecentPlaylistsCard` détecte un playlist_name UUID
  et affiche le libellé neutre localisé existant `common.home.unknown_playlist`
  (FR « Sélection inconnue » / EN « Unknown playlist ») — JAMAIS d'UUID brut. Test vitest
  (rouge sur code cassé). Backfill data requis pour résoudre le vrai nom EN → §Découvertes.

## LOT GH2-B — i18n re-passe 2 (backend locale-aware + oublis front)

- [x] GH2-B1 — Omnibar : filtre Saison bilingue (complète GH-4). FAIT : 3 littéraux FR
  de `SaisonPill.tsx` (trigger « Saison », « Toutes saisons », folding « + N saisons
  sans matchs ») → clés `common.filters.season_pill_label/season_all/season_empty_fold`
  (ICU plural). Test EN vs FR (`SaisonPill.test.tsx`, locale épinglée + bloc EN,
  0 assertion perdue).
- [x] GH2-B2 — Match View : onglets « Général »/« Détails » + « Antagonistes » →
  i18n match-view (`tabGeneral/tabDetails/antagonistTitle`, TABS→labelKey ;
  `MatchAntagonistChart` via `t`). Tooltip citations du drawer = BACKEND
  (`title={c.description ?? c.name}` ← Q26j) : nom locale-aware via la colonne
  existante `citation_name_display_en` ; description SANS source EN (seed FR-only) →
  masquée sous EN (principe GH-5b : EN n'injecte jamais de FR), tooltip = nom EN.
  Test Go EN vs FR (`TestCitationsRepo_LoadMatchCitationsRich_LocaleAware`).
- [x] GH2-B3 — Carrière « Rankings » : schéma réel = `player_csr_snapshots` persiste
  UN nom (canonique EN, sync locale "en" — statué GH-8). Fix au chokepoint UNIQUE de
  lecture (`enrichCSRPlaylistNames` ; tous les lecteurs passent par
  `GetCSRSnapshots` → career_service:166) : FR via asset_translations +
  `resolvePlaylistNameForLocale`, EN garde le nom persisté. Lecteur symétrique de la
  même page (`enrichLUSRPlaylistNames`, FR forcé) corrigé aussi
  (`PreferredLangsForLocale(ctxkeys.Locale)`). Test EN vs FR
  (`TestEnrichCSRPlaylistNames_LocaleAware`).
- [x] GH2-B4 — Accueil « Recent sessions » : FRONT (`HomeSessionCarousel.tsx`) —
  Solo/Escouade, Équipe/Perso, « N match(s) », outcomes, FDA, « Durée de la
  session », aria → clés `home.sessions.*` (TOUTES existaient dans home.toml mais
  n'étaient pas câblées ; coquille FR `losses_count` « # Defeats » corrigée).
  `dominant_playlist/mode` backend déjà locale-aware. Test EN vs FR
  (`HomeSessionCarousel.test.tsx`).
- [x] GH2-B5 — Accueil « Highlights » : titres de cartes = TitleKey manifest (déjà
  bilingues) ; les composés map · mode (Detail best_underdog/kda_peak/killing_spree,
  Value favorite_map) étaient `labelFR` (FR-first) →
  `labelForLocale`, locale threadée en PARAMÈTRE
  (`BuildHighlightsFromCanonical(rows, locale)` — analysis reste pur, résolution
  unique au point d'entrée GetHomePage). Tests EN vs FR
  (`TestSliceBestKillingSpreeCanonical_LocaleAware`,
  `TestSliceFavoriteMapCanonical_LocaleAware`).
- [x] GH2-B6 — Tuiles de matchs (accueil) : médailles (noms + tooltips) →
  `resolveMedalLabels` migré sur le helper canonique GH-5b
  (`medalLabelDescCoalesceSQL(ctxkeys.Locale)`) ; citations → Q26j locale-aware
  (2 scanners HomeRepo + CitationsRepo) ; commendations natives H5 (slot
  TopCitations) → `loadCommendationDefsFromMetadata` locale-aware (parité
  halo5_commendation_defs.go) ; « Recent media » : cartes →
  `enrichMediaMapTranslations` locale-aware. Tests EN vs FR
  (`TestResolveMedalLabels_LocaleAware`, `TestHomeRepo_LoadMatchCitations_LocaleAware`,
  `TestEnrichMediaMapTranslations_LocaleAware`).
- [x] GH2-B7 — Popup réassociation média : le dictionnaire `matchPicker`
  (i18n-modals.ts) existait FR+EN mais n'était PAS câblé (`MediaMatchPicker.tsx` FR
  en dur, erreur = clé leaderboard empruntée). Câblé intégralement (titres, Capture,
  Fermer, Fenêtre, compteur, Chargement, erreur dédiée, état vide, badge actuel,
  confirmation, Annuler/Confirmer/Application, équipes/spectateurs/lobby) ; clés
  mortes purgées (minutesSuffix/reassociationError/unknownError/youSuffix) +
  variantes titleAssociate/confirmTitleAssociate ajoutées. Test EN vs FR
  (`MediaMatchPicker.test.tsx`, 5 tests).
- Architecture : locale résolue UNE fois par requête (`ctxkeys.Locale(ctx)`, posée
  par le middleware title.go) côté repos/service ; `analysis/` reste pur (locale en
  paramètre depuis GetHomePage). Pas d'explosion de périmètre.

Gate GH2 (A puis B) : front purge cache + typecheck 0 + lint 0 err + vitest 0 fail ;
Go build+vet+tests + `-tags=integration -p 1 ./internal/api/... ./internal/service/...`
exit 0 si backend touché ; CI verte ; re-passe 3 utilisateur ciblée.

### RE-PASSE 3 UTILISATEUR — checklist ciblée (après GH2-B, UI EN puis FR)

1. Omnibar (toute page filtrée) : pill « Season » (EN) / « Saison » (FR), popover
   « All seasons » + folding « + N seasons without matches ».
2. Match View : onglets « General »/« Details », section « Antagonists » ; drawer
   scoreboard → survol des CITATIONS : nom EN, tooltip = nom EN (la DESCRIPTION
   n'existe qu'en FR — sous EN elle est volontairement absente, pas un bug).
3. Carrière, bloc « Rankings » : noms de playlists EN sous UI EN, FR sous FR
   (idem le bloc LUSR de la même page).
4. Accueil « Recent sessions » : « Squad »/« Solo », « N matches », outcomes EN,
   « Session duration », « Team »/« Self ».
5. Accueil « Highlights » : détails « map · mode » des tuiles (Plus belle victoire,
   Pic FDA, Série) dans la langue de l'UI.
6. Tuiles de matchs (accueil) : noms + tooltips des médailles ET citations dans la
   langue de l'UI (y compris joueur H5 : commendations natives) ; rail « Recent
   media » : noms de cartes localisés.
7. Popup de réassociation média (galerie ou accueil) : tout EN sous UI EN (titre,
   Window, boutons, états vide/erreur).

### RE-PASSE 3 — RÉSULTATS (Guillaume, 2026-07-09/10) → LOT GH3

Validés : 2 (Match View onglets/Antagonists/citations drawer), 3 (Rankings+LUSR),
4 (Recent sessions), 5 (Highlights), 6 (tuiles, sauf a/b ci-dessous), 7 (popup, sauf
libellés des suggestions). Résiduels → LOT GH3 :

- [!] GH3-1 — Omnibar Saison : la LISTE des saisons affiche les NOMS FR sous UI EN
  (source = catalogue saisons ; résoudre par locale).
- [!] GH3-2 — Le bouton/label « Analyse » reste tel quel sous UI EN — choisir le bon
  terme EN (« Analyze » pour le bouton d'action, « Analysis » pour une section de nav)
  et vérifier LES DEUX sites (bouton omnibar/L2 + section nav L1 le cas échéant).
- [!] GH3-3 — Tuiles de matchs : légendes rendement/résistance en FR sous EN
  (« 327 dégâts/frag », « 263 dégâts/mort ») → localiser.
- [!] GH3-4 — Popup réassociation : les SUGGESTIONS affichent playlist et mode en FR
  sous UI EN (endpoint match-candidates ou rendu front) → locale-aware. NOTE PERF
  consignée (hors périmètre) : chargement des suggestions lent.
- [~] 6b clarifié : les descriptions de citations n'existent qu'en FR (data) — sous EN,
  tooltip = nom seul (comportement GH2-B2 voulu) ; sous FR la description s'affiche.

## LOT GH3 — Traîne re-passe 3 (4 correctifs bornés)

> Exécution : agent GH3 (stoppé à la toute fin, gates Go déjà passés) ; finalisation,
> re-gates et commit par le superviseur (2026-07-10). Périmètre inchangé.

- [x] GH3-1 noms de saisons locale-aware (Omnibar popover) — `seasons_catalog.go`
  (résolution par locale) + plomberie `field_mappings.go` + `wire/registry.go` ; tests
  handlers EN/FR ajoutés.
- [x] GH3-2 terme EN : la valeur EN était l'orthographe britannique « Analyse » →
  « Analyze » (bouton + 2 messages ICU du popover filtres) ; manifests régénérés.
- [x] GH3-3 légendes « dégâts/frag » / « dégâts/mort » → clés
  `common.match_card.dmg_per_kill/death` (EN « dmg/kill » / « dmg/death ») dans le
  composant PARTAGÉ combat-yield (Home, tuile match, KpiGrid, Synthesis couverts d'un
  coup) ; tests locale.
- [x] GH3-4 playlist/mode des suggestions de réassociation locale-aware
  (`media_repo_filters.go`, réutilise la résolution GH2-B6) + tests EN/FR.

Gate GH3 PASSÉ (vérificateur, 2026-07-10) : go build/vet/tests handlers+service+duckdb
= 0 (l'agent avait aussi passé l'intégration -p 1 api/service/duckdb avant arrêt) ;
front purge cache : tsc 0, lint 0 err, vitest 245 fichiers / 2091 pass / 0 fail ;
CI = commit dédié. Re-passe 4 éclair utilisateur : Omnibar saisons EN, bouton
« Analyze », légendes tuiles EN, suggestions popup EN (+ tooltips citations EN de GH4).

## LOT GH4 — Descriptions EN des citations/commendations (nouveau, 2026-07-10)

Origine : re-passe 3 point 6b + Découverte GH2-B(a). Les descriptions de citations
n'existent qu'en FR (data) → sous UI EN, le tooltip du drawer retombe sur le nom seul
(masquage GH2-B2/B6 voulu, faute de source EN). L'utilisateur veut les descriptions EN.
Source proposée : la page Halopedia des commendations Halo 5. Périmètre EXÉCUTANT GH4 =
DATA (seed + migration + read) + tooltip citations UNIQUEMENT ; NE PAS toucher les
fichiers FRONT des tuiles/omnibar/popup (lot GH3 en cours).

CARTOGRAPHIE (GH4-1, sur pièces) — deux systèmes DISTINCTS :

- **Système A — `citation_mappings`** (le « Citations/Commendations » anneau doré,
  app-authored, PARTAGÉ Infinite+H5-dérivé). Table dans `metadata.duckdb` PAR TITRE
  (title-owned, isolée par chemin ADR 0008 ; colonnes sans title_id). Colonnes :
  `citation_name_norm` (PK), `citation_name_display` (FR), `citation_name_display_en`
  (EN nom, migration `add_citation_name_display_en`), `description` (**FR SEULE, PAS de
  `description_en`**), `mapping_type`, `category`, `image_path`, `tier_targets`, …
  Seed = `internal/ops/seed_citation_data.go::defaultCitationMappings()` (88 règles, FR
  écrit-main) + map `citationDisplayEN` (Norm→nom EN) → `SeedCitationMappings`
  (`ops/seed.go`), lancé par la CLI `levelup data seed citation-mappings` (PAS au boot ;
  même mécanique que `citation_name_display_en`, non encore déployé en prod car GH2
  pré-merge). Commentaire code : « les citations Infinite sont des COPIES de
  commendations Halo 5 (seul le calcul de progression diffère) » → la source EN
  autoritative des descriptions EST la commendation Halo 5 officielle.
  Tooltip / read-paths (chemin GH2-B2, « Q26j ») :
    1. `citations_repo.go::loadCitationMappingMeta` (Q26j) → Match View drawer/summary ;
    2. `home_repo_medals_citations.go::loadCitationMappingMeta` (Q26j) → tuiles accueil ;
    3. `citations_repo.go::LoadCitationMappings` (Q34) → page Citations catalogue.
  #1 et #2 masquent `description=""` sous EN (GH2-B2/B6) ; #3 sert le FR brut. C'est ici
  que la description EN doit être servie.
- **Système B — `commendation_definitions`** (H5 NATIF, metadata.duckdb halo_5,
  `platform/duckdb/halo5/halo5_commendation_defs.go` + `loadCommendationDefsFromMetadata`).
  Peuplé par `cmd/h5-metadata-fetch` depuis l'API Metadata OFFICIELLE
  (`www.haloapi.com/metadata/h5/metadata/commendations`, clé `LEVELUP_HALOAPI_KEY`,
  `Accept-Language` honoré). La table PORTE DÉJÀ `description_en`/`description_fr`
  (seedCommendations l.526-531) MAIS les read-paths ne SÉLECTIONNENT PAS la description
  (retour = ID/Name/IconURL/Category/TierTargets). Surface = slot TopCitations d'un
  joueur H5 (GH2-B6). Hors périmètre tooltip-description in-scope (le report utilisateur
  vise le système A) → noté Découvertes.

- [x] GH4-1 — CARTOGRAPHIE consignée ci-dessus (2 systèmes, sources, stockage, 3
  read-paths, point de masquage). Vérifiée sur pièces.
- [x] GH4-2 — H5 VOIE OFFICIELLE : clé `LEVELUP_HALOAPI_KEY` (`.env.local`) VALIDE —
  `GET /metadata/h5/metadata/commendations` = HTTP 200, 121 commendations EN avec
  descriptions (ex. « Spartan Slayer » → « Take out enemy Spartans »). Source autoritative
  pour les citations Système A copiées de H5 (mode de jeu, multijoueur, véhicules,
  éliminations, grenades). NB : ces descriptions sont RÉPLIQUÉES dans la map Go committée
  (pas une dépendance runtime à l'API — même robustesse que citation_name_display_en).
- [x] GH4-3 — CONTRE-VÉRIF Halopedia OK (WebFetch `List_of_Halo_5:_Guardians_commendations`).
  L'endpoint metadata NE porte PAS les commendations « Spartan Company » (Flag 'Em Down,
  Grand Theft, Power Play, Vandalism, Too Fast For You, No Hard Feelings, Look Ma No Pin…)
  ni certaines firefight → Halopedia les fournit/confirme. Matching : 88/88 citations ont
  une description EN. Provenance : ~57 API H5 officielle (verbatim), 15 Spartan Company
  (Halopedia + trad fidèle du FR), 24 maîtrises d'armes Infinite (trad fidèle, idiome
  officiel), 4 Infinite-only (avenger, sentinel/brute/skimmer slayer, trad fidèle). 0
  non-matchée. Attribution : contenu officiel du jeu, usage projet perso — OK.
- [x] GH4-4 — Map `citationDescriptionEN` (Norm→EN, 88 entrées) livrée dans
  `seed_citation_data.go`, symétrique de `citationDisplayEN` + helper
  `citationDescriptionENOr` (`seed.go`, nil si absente ⇒ pas de FR injecté). Data committée
  en Go. Garde-rail `seed_citation_en_test.go` : complétude (toute citation a une
  description EN) + pas d'entrée morte — PASSÉ (prouve 88/88).
- [x] GH4-5 — SCHÉMA + CÂBLAGE livrés : migration title-owned `add_citation_description_en`
  (ALTER `citation_mappings` ADD `description_en` ; ajoutée à `canonicalOrder`) ; seed
  écrit `description_en` (INSERT+UPDATE) ; Q26j + Q34 sélectionnent `description_en` ; les 3
  read-paths (drawer `citations_repo`, tuiles `home_repo`, catalogue `LoadCitationMappings`)
  servent la description EN sous UI EN (dispo → EN ; absente → nom seul, fallback GH2-B2
  conservé, jamais de FR). Tests Go EN vs FR : `TestCitationsRepo_LoadMatchCitationsRich_LocaleAware`
  + `TestHomeRepo_LoadMatchCitations_LocaleAware` (description EN sous EN) +
  `TestCitationsRepo_LoadCitationMappings_DescriptionLocaleAware` (fallback nom-seul
  prouvé). Backfill dev = re-run `levelup data seed citation-mappings` (data non committée) ;
  prod = même commande post-merge (voir PLAN DE MERGE étape V9d/6).

Gate GH4 : Go build+vet 0 ; tests packages touchés 0 ; `-tags=integration -p 1
./internal/platform/duckdb/... ./internal/api/...` exit 0 (schéma/lecture touchés) ;
1 test EN-vs-FR (description EN servie sous EN, nom seul si absente, FR sous FR) ;
commit(s) `cloture(GH4):` FR + Co-Authored-By ; push + CI verte. Données binaires
(metadata.duckdb) NON committées → livrer l'OUTIL (seed) + documenter l'exécution
(dev exécutée, prod post-merge).

## LOT GH5 — Résiduels re-passe 4 du gate humain (nouveau, 2026-07-10)

Origine : re-passe 4 du gate humain (post-GH3/GH4). Deux incohérences inter-surfaces.
Exécutant GH5 piloté. Périmètre FERMÉ = GH5-1, GH5-2 ; découvertes → §Découvertes.
Décisions superviseur PRÉ-TRANCHÉES (ne pas re-questionner) :
  - GH5-1 : ordre des sélecteurs de saison = RÉCENT-EN-HAUT (DESC) PARTOUT (aligner
    Omnibar/Explorer sur la Carrière CSR, déjà DESC via `sortCSRSeasonsDesc`).
  - GH5-2 : libellé du filtre « Experience Type » locale-aware côté BACKEND, la Value
    reste FR INCHANGÉE (miroir exact de GH3-1 saisons — « PVP non classé » est le pivot
    des 3 mécanismes cascade/substring : ne JAMAIS toucher la Value).

- [x] GH5-1 — Ordre des saisons DESC (récent d'abord) dans les sélecteurs Omnibar/Explorer.
  Le tri VISIBLE est établi front par `useSeasons()` (fieldMappings.ts:260, était
  `displayOrder` ASC). Approche retenue = tri par `startDate` DESC (récence réelle) :
  le `SeasonEntry` front ne porte PAS de champ `source`, donc keyer sur la date réelle
  place les saisons « DB-only » (DisplayOrder synthétique `maxOrder+10*(i+1)`,
  seasons_catalog.go:246) à leur JUSTE place chronologique (approche (a) du superviseur)
  et évite le piège du DESC-sur-displayOrder naïf. Données dev (serveur :8000) : 14 saisons
  TOML (S1→S13 + Winter Update), 0 DB-only aujourd'hui ; `display_order` strictement
  monotone avec `start_date` → startDate-DESC ≡ displayOrder-DESC pour le TOML, robuste
  au futur (une S14+ Waypoint arriverait DB-only et se placerait à sa date réelle en tête).
  PIÈGE traité (découvert on-pièces) : `prevSeason`/`nextSeason` (findSeasonAt.ts, consommés
  par PeriodSessionRail `SeasonRail`) étaient array-index → un flip global du selecteur
  partagé les aurait INVERSÉS ; rendus ORDRE-INDÉPENDANTS (voisin chronologique calculé par
  startDate). `useActiveSeason` (findActiveSeason) déjà ordre-indépendant (recherche par
  fenêtre) → sûr, inchangé. `SaisonPill` préserve l'ordre d'entrée dans chaque partition
  (available count>0 / unavailable count=0) → DESC conservé. `CareerHighlightMatchesSection`
  (dropdown saisons) hérite DESC = cohérent avec la décision. Tests DESC ajoutés
  (comparateur pur + prev/next ordre-indépendant + ordre DOM SaisonPill).
- [x] GH5-2 — Filtre « Experience Type » : Label locale-aware backend, Value FR inchangée.
  Les 3 libellés `expTypePVPUnranked`/`expTypePVPRanked`/`expTypePVE` (constantes FR,
  match_history_service.go:50-52) étaient Label ET Value des options (buildExperienceOptions
  filters_options.go:326 ; emptyResolved filters_service.go:459), sans locale → FR sous EN
  dans le FiltresPill. Décision = localiser le LABEL, garder la VALUE FR. Réalisation :
  mapping Go `value_FR → label_EN` (backend-authored, à côté des constantes FR) + helper
  `experienceLabelForLocale` ; localisation appliquée au POINT D'ENTRÉE ctx-aware
  `FiltersService.Resolve` (seul chemin prod surfaçant ces options à l'UI ; l'autre caller
  `filterMatchHistoryRows:407` jette `resolved` via `_ = resolved`). CHOIX documenté : PAS
  de threading de `locale` à travers la fonction PURE `ResolveFiltersFromRows`
  (~40 call-sites de tests) — post-projection unique à l'entrée = même résultat, risque nul,
  0 test cassé (la couche pure produit le canonique FR, le service projette vers la locale
  de requête). Libellés EN choisis = « Ranked PvP » / « Unranked PvP » / « PvE » (garde la
  distinction PvP/PvE des 3 options, cohérent avec le vocabulaire « Ranked »/« Unranked »
  des manifests session/synthesis/career). Value FR intacte → cascade `EXPERIENCE_TO_CASCADE`
  + matchers substring `applyExperienceFilter` (Go) et `experienceCounts` (useLocalFilterBar)
  inchangés, commentaires de contrat ajoutés à chacun. Test Go EN-vs-FR (Label localisé,
  Value FR dans les DEUX).

Gate GH5 PASSÉ (2026-07-10) : ✅ front cache purgé — typecheck 0 ; lint 0 err (68 warn
baseline, aucun sur mes fichiers) ; vitest 245 fichiers / 2099 pass / 14 skip / 0 fail
(dont GH5-1 : fieldMappings comparateur DESC, findSeasonAt prev/next ordre-indépendant,
SaisonPill ordre DOM). ✅ Go build 0 + vet 0 ; `go test ./internal/service/... ./internal/api/...`
0 FAIL ; `-tags=integration -p 1 -timeout 900s ./internal/api/... ./internal/service/...`
exit 0 (0 `^--- FAIL:`, tous `ok`), dont GH5-2 `TestFiltersService_Resolve_ExperienceLabelsLocaleAware`
(Label EN sous EN, Value FR dans les 2 locales, chemins buildExperienceOptions + emptyResolved).
Aucun manifest i18n ni `generated/*.ts` touché (GH5-2 = libellés EN Go-authored ; GH5-1 = TS pur).

Journal GH5 (2026-07-10) : GH5-1 — flip du sélecteur PARTAGÉ `useSeasons` révèle sur pièces
un couplage non cité par le plan : `prevSeason`/`nextSeason` (findSeasonAt.ts) étaient
array-index et consommés par `PeriodSessionRail` → un flip naïf aurait INVERSÉ les boutons
saison précédente/suivante du rail. Rendus ordre-indépendants (voisin chronologique recalculé
par startDate sur copie). Tri visible keyé sur `startDate` DESC (pas `displayOrder`) : le
`SeasonEntry` front n'a pas de champ `source`, la date réelle place les DB-only à leur juste
récence (0 DB-only en dev aujourd'hui — 14 saisons TOML S1→S13 + Winter ; robuste au futur).
GH5-2 — 2e caller de `ResolveFiltersFromRows` (`filterMatchHistoryRows`) jette `resolved`
(`_ = resolved`) → le SEUL chemin prod surfaçant les options d'expérience à l'UI est
`FiltersService.Resolve`. Choix : localisation du Label au point d'entrée ctx-aware (post-projection
unique) plutôt que threading de `locale` dans la fonction PURE `ResolveFiltersFromRows`
(~40 call-sites de tests) — même résultat, risque nul, Value FR intacte (cascade/substring),
commentaires de contrat posés aux 3 sites couplés (EXPERIENCE_TO_CASCADE, applyExperienceFilter,
experienceCounts). Écart documenté vs la lettre « thread jusqu'au builder » : la décision
superviseur (Label locale-aware backend, Value FR) est respectée à l'identique.

## PLAN DE MERGE & DÉPLOIEMENT (le big-bang est inévitable — le rendre sûr)

Constat : les merges intermédiaires (après B, G, K) n'ont pas été faits ; ~130 commits
partiront d'un coup et push main = deploy prod AUTO, SANS kill-switch V1 (retiré D1c).
Séquence OBLIGATOIRE, dans l'ordre :

1. **CI de branche VERTE** (V1e) + gate global final local : `go build && go vet &&
   go test ./...` puis `-tags=integration -p 1 -timeout 900s ./...` exit 0 ; front cache
   purgé + typecheck + lint + vitest.
2. **Répétition générale sur la copie prod** (celle de V10a) : pointer un serveur local
   sur la copie (`data/` restauré), booter le binaire de la branche → vérifier que les
   migrations/vues au boot passent (PME view sans colonnes mortes, `_latest`,
   deprecatedPlayerAggregates, prestige/halo5 extraits), smoke des pages sur données
   réelles. C'est LA répétition du deploy — aucune surprise de migration ne doit rester.
3. **Gate live-sync manuel** (dette D1c) : un sync delta + un backfill court sur le dev
   local avec tokens réels, les 6 joueurs — AVANT le merge (c'est l'utilisateur ou une
   session avec réseau/tokens qui le fait).
4. **GATE HUMAIN** ci-dessus coché.
5. **Fenêtre de merge** : prévenir (deploy auto), heure calme, utilisateur disponible ;
   `git checkout main && git pull && git merge refactor/audits-2026-07` (merge commit,
   pas de squash — l'historique par lot est la traçabilité) ; push.
6. **Post-deploy immédiat** (checklist V10b) : logs boot (migrations OK), port répond,
   pages clés, `legacy_source_used` visible, **NOTER LA DATE D1A au plan parent**,
   surveiller le premier auto-sync complet.
   - **SEED CITATIONS (GH2-B2 + GH4)** : `citation_name_display_en` ET `description_en`
     sont peuplés par le seed CLI, PAS au boot. Lancer `levelup data seed citation-mappings`
     (conteneur one-off, serveur arrêté — écriture metadata.duckdb Infinite, mono-process
     ADR 0013 ; NE PAS écraser tout le metadata.duckdb, le seed est SELECT-then-write
     idempotent, respecte `asset_translations` UGC runtime). Sans ce run, les noms/descriptions
     EN des citations restent vides en prod (lecteur retombe sur nom FR / nom seul — pas de
     régression, mais objectif GH4 non atteint). Les migrations `add_citation_name_display_en`
     + `add_citation_description_en` (colonnes) passent, elles, au boot.
7. **Rollback documenté AVANT le merge** : le deploy étant auto, le rollback =
   `git revert -m 1 <merge>` + push (redéploie l'ancien état applicatif). RISQUE DB à
   évaluer en répétition (étape 2) : lister les migrations NON réversibles du diff
   (rebuilds, drops de vues) — si une migration rend la DB incompatible avec l'ancien
   binaire, le rollback exige AUSSI la restauration restic → c'est le critère GO/NO-GO
   de l'étape 2.
8. D2 (ADR 0023 Phase 5) : chantier séparé `refactor/adr0023-phase5`, déclenché par la
   télémétrie `legacy_source_used` ≥7 j après la date notée en (6).

## Campagnes futures (hors clôture — jamais auditées à ce jour, acté 2026-07-06)

- **Accessibilité** (contrastes tokens, clavier, ARIA — `lib/accessibility` déjà prêt).
- **Perf front** (bundle, code-split, poids ECharts, cache TanStack).
- **Périmètre sécurité réseau** (rate-limiting login/callback, en-têtes CSP/HSTS,
  posture cookies) — défensif uniquement.
- **E2E navigateur** (Playwright) pour rendre le GATE HUMAIN répétable.
- (Alerting uptime : écarté par décision utilisateur 2026-07-06 — pas de système
  de monitoring externe souhaité à ce stade.)

## Découvertes hors périmètre (à remplir pendant l'exécution — NE PAS traiter)

- **[V1 — TRAITÉE car bloquait le gate] DC-2 a un effet de bord non anticipé.** Une
  directive `/// <reference types="node" />` dans un fichier inclus au programme tsc
  N'EST PAS file-scopée : elle tire `@types/node` dans TOUT le programme (`tsconfig.app.json`
  `include: ["src"]` couvre les `.guard.test.ts`). Conséquence : `setTimeout`/`ReturnType<typeof setTimeout>`
  bascule sur la surcharge Node (`Timeout`) pour tout le code, cassant `CoverFlowModal.tsx:492`
  (`window.setTimeout` → `number`). L'intention de DC-2 (« ne pas exposer les globals Node
  au code navigateur ») est partiellement contredite par le mécanisme choisi. Corrigé a
  minima dans le périmètre (1 ligne CoverFlowModal + commentaire) pour débloquer le gate.
  Piste future (hors lot) si on veut vraiment isoler node : sortir les 2 guard-tests dans
  un projet tsconfig séparé, ou remplacer la directive par un typage local de `process`/`node:*`.

- **[V4 — commentaires « carte de navigation » stale, TRAITÉS car mécaniquement liés à
  V4a]** `engine.go:747` et `engine_postsync.go:237` listaient/citaient
  `insertHighlightEventsFromData` (fonction supprimée en V4a) → doc inversée introduite
  par ma suppression. Corrigés dans le lot (non un fix opportuniste : conséquence directe
  du retrait). Le reste du balayage commentaires stale reste V5c.
- **[V4 — hors périmètre, NON traité] `scripts/coverage_baseline.txt`** est un 2e fichier
  baseline versionné (distinct de `apps/go-api/coverage_baseline.txt`), consommé par
  `scripts/check_coverage_ratchet.sh` (ratchet per-function). VF-9/DC-6 ne visait que le
  fichier apps/go-api → laissé tel quel (légitimement versionné).
- **[V4 — hors périmètre, NON traité] entrée `weapon_kills` de `shared_write_guard_test.go`**
  décrit encore « InsertWeaponKills (DELETE+INSERT sérialisé…) » : InsertWeaponKills est
  passé append-only (génération, plus de DELETE, cf. writes.go). Justification légèrement
  stale mais l'entrée reste NÉCESSAIRE (InsertWeaponKills écrit toujours weapon_kills) —
  correction cosmétique reportée (pas dans le périmètre VF-5 qui ne cite que :54,64).

- **[V8 — 3 divergences front↔back hors cas A2, NON traitées]** Trouvées pendant le
  balayage V8a (détail : `.ai/INVENTAIRE_V8A_TYPES_FRONT_BACK.md` §divergences additionnelles).
  Hors du cas PROUVÉ A2 (Career), donc consignées et non corrigées (règle 7 ; aucune ne
  bloque le gate V8). À trancher (décision produit/backend) :
  1. `CompareResponse.privacy_warning` + `.player_b_partial` (lus `ComparePage.tsx:433-434`,
     absents du Go `domain.CompareResponse` + openapi) → LATENT ACTIF : bannière privacy et
     hint partiel jamais rendus. `CompareResponse` est allowlisté V8d (dette verrouillée).
  2. `NormalizedPlayerStats.is_local_sample` : présent Go+openapi, absent de l'interface
     front → champ backend non lisible (sens inverse). Sous-type, hors garde-rail nommé.
  3. `RecentMediaItem` (sous-type `HomePageResponse.recent_media`) : front déclare 15 champs,
     Go n'en sérialise que 3 → LATENT DORMANT (recent_media non consommé dans l'UI Home,
     référencé `[]` en tests seuls). Sous-type, hors garde-rail nommé.
  Fixer (1) exige de décider add-au-Go vs retrait-front ; (2)/(3) = alignement de sous-types
  view-model (élargir le garde-rail aux sous-types = chantier distinct).

- **[V10-D1 — hors périmètre, NON traité — ALARME PROD] Le backup restic automatique n'a
  JAMAIS produit de snapshot.** Le timer `levelup-restic-backup.timer` s'exécute bien à
  04:00 UTC mais le service échoue avec `status=203/EXEC` : le script à
  `/opt/levelup/scripts/restic-backup.sh` a les permissions `-rw-r--r--` (bit exécutable
  ABSENT) → systemd ne peut pas l'exécuter. Les 3 snapshots existants (dont `latest`
  9e96ed20 du 2026-06-27) ont TOUS été créés par des invocations MANUELLES (`bash
  restic-backup.sh`). Conséquence : `latest` a 10 j au 2026-07-07 et aucun backup
  automatique n'existe. Correctif (VPS, hors périmètre lecture-seule V10 — à faire par
  l'utilisateur ou une session write) : `chmod +x /opt/levelup/scripts/restic-backup.sh`.
  NOTE : `deploy.sh` fait `git reset --hard` qui écrase le fichier depuis git — vérifier
  que le repo versionne le bit +x (le script `scripts/restic-backup.sh` doit être
  exécutable dans git, sinon le chmod sera reperdu au prochain deploy).
- **[V10-D2 — hors périmètre, NON traité] Repo restic single-disk sans copie off-VPS
  prouvée.** `scripts/RESTIC_BACKUP.md` recommande de copier `.restic-password` + le repo
  hors-VPS (sinon irrécupérable si le disque est perdu) mais aucune preuve que c'est fait.
  La copie locale `LevelUp-prod-copy/` produite par V10a est de fait la première
  validation off-VPS d'une restauration. Résilience hors-site = chantier séparé (non une
  clôture d'audit).
- **[V10 — note doc] `docs/BACKUP_RESTORE.md` décrit un 2e mécanisme (scheduler in-app
  `pkg/duckdbbackup`, Parquet-staged, `cmd/restore`) DISTINCT du backup prod réel (timer
  systemd + `restic-backup.sh`, fichiers `.duckdb` bruts).** Le scheduler in-app est
  désactivé (`backup_enabled` défaut `false`). Deux docs coexistent sans se référencer —
  `RUNBOOK_RESTORE_TEST.md` cible explicitement le mécanisme réellement en prod et note la
  distinction. Consolidation doc = hors périmètre.

- **[GH-1 — hors périmètre, NON traité] Commentaire historique stale `timeseries.toml:6-7`.**
  L'en-tête du manifest cite « Pilote 6 onglets : KPIs, Cumul, Forme, Intensité,
  Distributions, Combat » — plan pilote ancien ; les onglets RÉELS sont Synthèse /
  Distributions / Progression (`TimeseriesPage.tsx:41-43`). C'est la source du faux item
  « onglet Forme » du GATE HUMAIN. Fix trivial (1 ligne de commentaire) mais hors périmètre
  GH-1 (qui ne vise que l'item du plan) — laissé tel quel (règle 7).

- **[GH-9 — hors périmètre, NON traité] Header Match View voie CANONIQUE (H5 live) reste FR.**
  `buildCanonicalHeader` (`match_view_canonical.go`) résout map/mode/playlist via
  `assetLabelAndID`/`canonicalModeUI` (label « FR si dispo sinon défaut », sous-système
  adapter canonique) et la date via `formatDateFRLong` — pas de séparation FR/EN ni de ctx
  locale threadé. Sous UI EN, un match servi par la voie live/H5 garde des libellés FR.
  Rendre l'adapter canonique locale-aware (AssetReference FR/EN + threading ctx) = chantier
  distinct (touche la couche canonical, pas juste le header). GH-9 a corrigé la voie
  Infinite (`applyMatchHeaderMetaLabels`), qui est celle du rapport utilisateur.

- **[GH-8 — hors périmètre, NON traité] Ré-localisation lecture snapshots CSR persistés.**
  `GetCSRSnapshots` sert `CareerPlaylistCSR.PlaylistName` persisté (« en » depuis le sync).
  Si une surface Career/saison affiche ce nom persisté directement sous UI FR, elle
  montrerait de l'EN (bug symétrique de GH-8). Le bloc Explorer flagué lit la voie LIVE
  (corrigée). Statuer/re-localiser à la lecture des snapshots = surface distincte non
  flaguée, notée.

- **[GH-3 — design gap, NON traité] Rivalité Relations absente des Top matchs Career.**
  `TopMatchDTO` ne porte aucun champ badge/adversaire ; le tableau ne rend qu'une pill
  d'issue. Les cartes de rivalité de la page Relations (`RelationsRivalryCards`) sont des
  agrégats PAR ADVERSAIRE (duels), pas des badges par match. Rapprocher les deux exige une
  feature backend (jointure top-match → adversaire dominant + état rivalité, nouveau champ
  DTO) + rendu front — ~1-2 j, pas une réutilisation de composant. Décision produit :
  backlog éventuel (afficher un contexte de rivalité sur les top matchs). Non corrigé.

- **[GH2-A3 — CAUSE DATA + BACKFILL prod, à exécuter hors dev]** Le nom UUID vient d'un
  trou de traduction : la playlist `96f32b0a-f89b-4507-83b1-bc07dd458dfa` a une entrée
  FR dans `metadata.asset_translations` (« Arène delta : Héritage ») mais AUCUNE entrée
  EN → sous UI EN, la résolution retombe sur le `match_registry.playlist_name` brut = le
  playlist_id. Le fix GH2-A3 est un garde d'affichage (jamais d'UUID) ; le VRAI nom EN se
  répare en peuplant `asset_translations` (playlist, en-US) — commande existante
  `go run ./apps/go-api/cmd/populate-assets` (peupler les playlists manquantes). NE PAS
  l'exécuter sur les données dev ici (pas de tokens ; risque écriture) — à lancer côté prod
  (ou dev avec tokens réels) dans un chantier data. Vérif : `SELECT asset_id,lang,name FROM
  asset_translations WHERE asset_id='96f32b0a-…' AND asset_type='playlist'` doit avoir une
  ligne `en`/`en-US`. Le même trou explique le mode UUID `8347c528-…` vu dans le filtre
  Média (symptôme identique côté mode).

- **[GH2-A1 — DÉCOUVERTE snapshot ready-filter, NON traité]** Le snapshot immuable v10
  (watermark 2026-07-07T21:06) EXCLUT 106 matchs pourtant présents en live et COMPLETS
  (ex. `9a2241c5…` : 8 participants, 214 events, 31 médailles). Le cut n'exporte que les
  matchs « ready » (`ready_match_count`=1713 vs live=1819) ; le critère de readiness a donc
  écarté des matchs à données complètes (lag de flag readiness au moment du cut, ou seuil
  trop strict). Le fix GH2-A1 (fallback live) masque le symptôme correctement, mais la
  cause amont (readiness du snapshot cutter) mérite un audit séparé — sinon la voie live
  (plus lente, sujette au B-swap) est empruntée pour ~6 % des matchs. Hors périmètre GH2-A.

- **[GH2-A2 — DÉCOUVERTES adjacentes, NON traitées]** (a) `LoadMatchCandidatesForMedia`
  renvoie une ERREUR (→ 500) quand le lookup capture est `ErrNoRows` : après le fix c'est
  inatteignable pour un média valide, mais un média réellement introuvable resterait un 500
  « loading error » au lieu d'une réponse vide propre — durcissement défensif possible (hors
  lot). (b) L'endpoint `/media/likes` convertit l'URL servable via `urlToFilePath` (chemin
  ABSOLU), qui pour une vidéo HLS donne `basename`=`master.m3u8` — le like d'une vidéo HLS
  a probablement le même angle mort que la réassociation avant fix (à vérifier/durcir
  séparément ; non reproduit dans ce lot).

- **[GH2-B — DÉCOUVERTES, NON traitées]** (a) **Descriptions de citations : aucune
  source EN.** `citation_mappings.description` est seedée FR-only (seed_citation_data.go)
  et il n'existe pas de colonne `description_en` (contrairement au nom,
  `citation_name_display_en`). GH2-B2/B6 masquent la description sous EN (principe
  GH-5b) → tooltip = nom. La vraie traduction EN des descriptions = chantier
  data/seed (ajouter `description_en` + seed, symétrique de
  `add_citation_name_display_en`). (b) **`LoadPlaylistAssetTranslationsFR`**
  (career_repo_highlights.go:323) reste un contrat FR-nommé qui force
  `PreferredLangsForLocale("fr")` — consommé par les highlights Carrière (« Matchs
  marquants ») ; si cette surface montre des playlists FR sous UI EN à la re-passe 3,
  c'est ce site (renommer + threader la locale = petite refonte du contrat, non
  flaguée par l'utilisateur, non traitée règle 7). (c) **Clés `home.sessions.newer_aria/
  older_aria` dormantes** : home.toml porte des doublons des clés réellement câblées
  (`common.home.newer_session_aria/older_session_aria`) — purge cosmétique du manifest
  possible. (d) `OUTCOME_LABELS_FALLBACK_FR` (MediaMatchPicker) : fallback FR-only
  quand `useFieldMappings` n'a pas encore répondu — la voie primaire (outcomes TOML)
  est localisée ; résiduel transitoire acceptable, noté.

- **[GH5-2 — SURFACE SYMÉTRIQUE hors périmètre, NON traitée]** Le filtre « expérience »
  du mode « matchs » de l'Explorer/Historique est servi par un champ DISTINCT :
  `MatchHistoryQuerySummary.AvailableExperienceTypes []string` (valeurs FR canoniques,
  `computeExplorerAvailableOptions`, match_history_service.go:220/269) → openapi
  `available_experience_types` → `ExplorerPage.filterOptions.ts:42`. Sous UI EN cette surface
  montrerait les mêmes libellés FR (« PVP non classé »…). GH5-2 vise le FiltresPill de
  l'OMNIBAR (`FiltersService.Resolve`, le report utilisateur) — cette surface Explorer est un
  chemin séparé, non flaguée, hors périmètre (règle 7). Fix futur = même recette (localiser le
  Label côté service, garder la Value FR ; le champ étant `[]string` il faudrait le passer en
  `[]LabelValue` ou localiser front via un mapping value_FR→label — petite refonte du contrat).
- **[GH5-2 — micro-inefficacité pré-existante, NON traitée]** `filterMatchHistoryRows`
  (match_history_service_filters.go:407) calcule `resolved := ResolveFiltersFromRows(...)` puis
  le JETTE (`_ = resolved`) et re-filtre via `applyAllFilters` — appel entièrement inutile
  (résultat jamais lu). Pré-existant, hors périmètre GH5 (retrait de l'appel mort = nettoyage
  ultérieur).
- **[GH5-1 — pas une découverte bloquante] `prevSeason`/`nextSeason` rendus ordre-indépendants.**
  Traité DANS le lot (conséquence directe du flip de `useSeasons`, comme V1c/V4a) : ce n'est
  pas un fix opportuniste mais la correction obligatoire d'une régression que le flip aurait
  introduite sur `PeriodSessionRail`. 0 DB-only saison en dev aujourd'hui (14 TOML) ; la clé
  `startDate` rend le tri robuste si Waypoint ajoute une saison hors TOML.
