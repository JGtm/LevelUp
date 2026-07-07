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

- [ ] V9a — Audit read-only sur la COPIE restaurée par V10a (jamais sur les fichiers que
  le serveur tient — mono-process) : quantifier (1) matchs `first_joined_time` décalés
  (~964 attendus, requête du diag `backfill_first_joined_tz`) ; (2) matchs import
  OpenSpartan à `is_ranked` faux (signal RankRecap) ; (3) lignes orphelines/doublons
  match_registry↔participants↔medals ; (4) présence physique des colonnes à DROP différé
  (DEC-6 `known_teammates_count`/`friends_xuids`, G5 `discord_notified`) ; (5) watermarks
  LUSR v2 vs lignes réelles (désync connue). Livrable : rapport chiffré par titre.
- [ ] V9b — Correctif TZ `first_joined_time` : backfill dédié (outil existant à vérifier :
  `backfill_first_joined_tz`) + RE-BACKFILL LUSR des joueurs affectés
  (`RecomputeLUSRCanonicalForPlayer`, jamais v1) — d'abord sur la copie (validation),
  puis sur prod en fenêtre convenue.
- [ ] V9c — Correctif `is_ranked` OpenSpartan : fix à l'import (signal RankRecap) +
  backfill des matchs existants + re-calcul CSR/LUSR impactés. Même méthode copie→prod.
- [ ] V9d — Planifier LE rebuild append-only qui exécute les DROP différés (recette
  ADR 0026) : colonnes DEC-6 + `discord_notified` + toute colonne morte confirmée par
  V9a(4). Date/fenêtre à convenir — le rebuild N'EST PAS fait en autonomie.
- [ ] V9e — Statuer `weapon_kills_v3` (shadow jamais promu) : promouvoir ou retirer —
  DÉCISION UTILISATEUR à escalader avec les chiffres V9a en main.
- [ ] V9f — Cluster « TODO P4 ADR 0006 retirer *100 » (§7 parent) : re-confirmer qu'il
  reste hors périmètre (migration d'unité canonique = chantier à part) et le dater
  `TODO(expiry:)` en bloc pour qu'il ne soit plus indolore.

Gate V9 : rapport V9a remis AVANT tout correctif ; chaque backfill validé sur copie
(counts avant/après) puis rejoué sur prod en fenêtre annoncée ; intégration `-p 1` verte
si du code d'import a changé (V9c touche sync/).

## LOT V10 — Exploitation (SANS alerting uptime — décision utilisateur 2026-07-06)

- [ ] V10a — Test de RESTAURATION restic : restaurer le dernier snapshot prod dans un
  dossier local isolé, ouvrir chaque DB restaurée (duckdb CLI RO), vérifier counts
  plausibles (match_registry, players, media). Documenter la procédure exacte en runbook
  (elle n'existe qu'à l'état de backup jamais rejoué). Cette copie SERT AUSSI à V9a et à
  la répétition générale du merge.
- [ ] V10b — Checklist de déploiement EXÉCUTABLE (fichier `docs/RUNBOOK_DEPLOY_CHECKLIST.md`
  ou script) consolidant les pièges connus aujourd'hui dispersés en mémoires : regen-demo
  (rm avant seed), lock backfill → crash-loop, CHECKPOINT shared_social, vérif
  `legacy_source_used` post-deploy, NOTER LA DATE D1A (arme D2), vérif port 8000/logs
  par catégorie, `/debug/vars` accessible admin.
- [ ] V10c — POST-merge : fenêtre d'observation runtime — lire `duckdb_pool_stats` +
  `duckdb_budgets` sous charge réelle (débloque J1(2)) et statuer ENFIN J4/J6 (measure-first)
  avec des chiffres. Clôt la boucle « mesurer d'abord » du lot J.

Gate V10 : restauration prouvée (DB ouvertes + counts) ; checklist rejouée à blanc une
fois ; V10c = chiffres consignés au plan parent (Gate J définitif).

---

## GATE HUMAIN — revue visuelle utilisateur (AVANT merge, rien oublier)

Checklist consolidée de TOUTES les vérifications visuelles promises par la campagne
(à dérouler par Guillaume sur le dev local, FR puis EN via le switch locale) :

- [ ] A1 — Timeseries tab **Forme** : échelle de perf non inversée (perfScale).
- [ ] A2 — Career **Top matches** : badges outcome corrects (win/loss/tie/dnf), dates
  au bon format locale — ET les données s'affichent (croisé V8b).
- [ ] A5 — Tooltips ECharts échappés : `MatchWeaponCharts` (nom d'arme + valeur + %
  préservés post-conversion formatter), squad heatmap/efficiency, OutcomeSequenceTape.
- [ ] Gate F — Pages H5 : Médias, Explorer, Match View (pas de fuite Infinite : lien
  Waypoint absent, CSR vide propre, labels corrects).
- [ ] I1/I2 — Onboarding (XboxLoginPage, StepDeviceCode, StepInitialSync, RegisterPage,
  OpenSpartanImportCard) + MatchScoreboard + heatmaps activité **en EN** (aucun résidu FR).
- [ ] I4 — AscensionProfileTab / MatchViewPage / PrestigeSquadProgress **en EN**.
- [ ] G3 — Session-detail intact (l'infra partagée session-summary n'a pas régressé).
- [ ] Smoke général : Home, Career, Squad, Explorer, Sessions — FR et EN, un joueur
  Infinite + un joueur H5.

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
