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

- [ ] V4a — VF-5/DC-5 : supprimer `insertHighlightEventsFromData`
  (engine_highlight_events.go:163) + ses 2 tests (`highlight_events_orchestration_test.go`)
  après vérif du sibling `ProcessHighlightEvents` (chemin batch vivant).
- [ ] V4b — VF-5/DC-5 : supprimer `InsertRegistryIfNotExists`/`InsertParticipants`/
  `InsertMedals` (sync/writes.go:34/99/188) + tests associés (`writes_test.go`,
  `concurrent_upsert_*_test.go`, `concurrent_multiplayer_e2e_test.go` — vérifier sur
  pièces ce que chacun teste d'AUTRE avant suppression) + retirer les entrées ON CONFLICT
  de l'allowlist ART devenues sans objet + la justification périmée de
  `shared_write_guard_test.go:54,64`. ATTENTION : `UpsertSharedCSRs` reste VIVANT
  (csr_shared_backfill.go:149) — ne pas le toucher.
- [ ] V4c — VF-6 : purger les entrées mortes d'allowlists : `sentinel_test.go:50,:155`
  (internal/api/registry.go ×2) ; `no_art_patterns_test.go:146` (allowlistRawDelete) ;
  `no_attach_on_social_test.go:312` (social_persister_combined.go).
- [ ] V4d — VF-6 : étendre les self-checks : `TestAllowlistJustifiesEverything` couvre
  AUSSI `allowlistRawDelete` ; ajouter un self-check « toute entrée d'allowlist pointe un
  fichier existant » à sentinel_test.go et no_attach_on_social_test.go (pattern du
  tripwire ART — c'est le mécanisme qui aurait évité VF-6).
- [ ] V4e — VF-9/DC-6 : `git rm --cached apps/go-api/coverage.html` + .gitignore ;
  statuer coverage_baseline.txt (consommé par scripts/coverage_filter.sh ? → garder si oui).
- [ ] V4f — VF-12 (partiel, mécanique) : fixture morte `discord_extra_test.go:51` (clé
  `discord_notify_new_media` écrite pour rien) — nettoyer.

Gate V4 : `go build ./... && go test ./...` → exit 0 ;
`go test -tags=integration -p 1 ./internal/sync/... ./internal/persist/...` → exit 0
(anti-ART après suppression des tests upsert) ; grep de chaque symbole supprimé → 0.

## LOT V5 — Garde-rail halowaypoint + docs/commentaires inversés

- [ ] V5a — VF-7 : livrer `internal/archlint/no_halowaypoint_literal_test.go` : interdit
  le littéral `halowaypoint` hors `games/halo_infinite/`, `platform/halo/` (si existant),
  `sync/haloclient/`, `games/halo_5/client.go`, `platform/auth/` (échange tokens),
  `domain/title/auth_descriptor.go` (defaults documentés), `assets/fetcher_gamecms.go`,
  `halotest/`, `cmd/` (outils) — allowlist PAR FICHIER datée 2026-07-06, décroissante,
  self-check d'existence des entrées (leçon V4d). Le but n'est pas 0 aujourd'hui : c'est
  de FIGER la liste pour que la prochaine URL en dur hors liste soit rouge.
- [ ] V5b — VF-8 : purger `LEVELUP_CONTRACT_VALIDATE` de `docs/CONFIGURATION.md:224`,
  `docs/FR/CONFIGURATION.md:225` (les DEUX, règle bilinguisme), `.env.local.example:203`.
- [ ] V5c — VF-12 : balayage des commentaires stale (1 commit, mécanique) :
  `engine_fetch.go:29-32` (réécrire : chemin batch unique), `engine_highlight_events.go:162`
  (disparaît avec V4a — vérifier), mentions `processMatch` (`backfill_personal_scores.go:7`,
  `engine.go:100`, `csr_shared_backfill.go:5`, `csr_writes.go:5`), `engine.go:175`
  (RunBackfillLUSR), header `session_compare_service.go:1,22` (décrire l'infra
  session-summary partagée), commentaire orphelin fin de `media_service_upload.go:189-190`,
  `ops/healthcheck.go:8` (exemple fmt.Println → slog), `eslint.config.js:30-31`,
  `.golangci.yml:8,15`.
- [ ] V5d — VF-10 : débris K : retirer/corriger le `//nolint:gocyclo` mensonger de
  `server.go:490` (startSessionPurgeLoop n'en a pas besoin) + fragment orphelin
  `server.go:126-127` ; justifier le nolint nu `player_repos_test.go:112` (ou le retirer) ;
  rafraîchir l'historique `sync_root_freeze_test.go:21` (112→106→88→80) ; corriger le
  bilan K3d/K2a du plan parent (« 4 fichiers » → 5, server_apiv1.go nommé) ; ajouter un
  commentaire d'exemption fichier en tête de `server_apiv1.go` (assembleur DI séquentiel,
  1286 L, pourquoi c'est accepté + condition de re-découpe).

Gate V5 : `go test ./internal/archlint/...` vert (nouveau ratchet inclus, vérifié dans
les 2 sens) ; grep `LEVELUP_CONTRACT_VALIDATE` hors coverage/artefacts → 0 ;
grep `processMatch|insertFetchedMatch|RunBackfillLUSR` en COMMENTAIRE prod → 0 résiduel
trompeur (les mentions « historique/supprimé le » explicites sont OK).

## LOT V6 — Tracker, journal, dette assumée, vérification finale

Objectif : le plan parent redevient la source de vérité, puis SA dernière action (jamais
faite) est exécutée.

- [ ] V6a — VF-4 : plan parent — les statuts J sont DÉJÀ à jour (`18f3c7ee7`, clôture J
  pendant l'audit — vérifier qu'ils le sont restés) ; reste : I2 → `[x]` (I2b complet,
  hash) ; I4 → `[x/~]` + purger la sous-puce (ii) livrée ; purger le paragraphe
  « RESTE (7) » de K3f et rafraîchir le bloc « BILAN SESSION /goal » ; statuer P1/P2 ;
  ajouter les entrées §6 manquantes : H, I, J, K, L, M, N (format du plan, comptes/gates
  repris du thought_log).
- [ ] V6b — VF-4 : thought_log — l'entrée J est DÉJÀ posée (`18f3c7ee7`) ; reste l'entrée
  de CE chantier de clôture (et une par lot V au fil de l'eau, règle CLAUDE.md).
- [ ] V6c — VF-4 : DETTE_ASSUMEE_2026-Q3.md — corriger l'entrée « E7 » (contenu = D1a/D2)
  et ajouter le VRAI E7 (DDL bootstrap sync/schema.go, condition : après stabilisation
  b23/b25) ; ajouter les résidus K réels (K1b-legacy→D2, K1d-reste, K1h-collection,
  K1j-openspartan, K1k-factory, K1l-reste-chemins, K1n-déplacements, K2b-drain(infaisable,
  documenté), K3b-ratchet-imports-croisés, K3f-décisions-packages) ; rafraîchir §4 J
  (J3/J7 livrés ; restent J1(2)/J4/J5/J6/J9-suite) ; ajouter N3(b/c/e) ; trancher le
  footer (reco : déplacer les ✅ dans une section « Livrées depuis » en bas, garder la
  règle).
- [ ] V6d — Exécuter la VÉRIFICATION FINALE du plan parent (§5, jamais faite) : relire
  les 4 audits en diagonale, confirmer que chaque finding a un statut dans le plan parent
  OU dans DETTE_ASSUMEE OU ici ; compléter la matrice §5 si un item est orphelin ;
  produire le BILAN FINAL utilisateur (fait / différé-où / abandonné-pourquoi).
- [ ] V6e — Mémoire : mettre à jour la mémoire projet (plan traitement audits) avec l'état
  de clôture + pointeur vers ce plan et l'audit VF.

Gate V6 : relecture croisée — 0 item du plan parent sans statut ; 0 `[!]` sans
justification ; chaque report présent dans DETTE_ASSUMEE ; bilan final rédigé et remis.

## LOT V7 — Résiduel qualité (bornés, faibles enjeux)

- [ ] V7a — VF-11 : `XboxLoginPage.tsx:390,:422` → clés `common.auth.*` existantes.
- [ ] V7b — VF-13/DC-7 : Q29 + Q29Bulk → `StartTimeCanonicalSQL("r")` (les 2 dans le même
  commit, tests bulk==unitaire re-passés) + extension du garde-rail H1 (`ORDER BY` sur
  start_time nu ; allowlist migrations/ gelées).
- [ ] V7c — VF-14 : `perfTierToken` — 4e copie détectée : centraliser dans
  `lib/accessibility/scales` (ou module dédié) + étendre le garde-rail perfScale existant
  (règle 6 : helper + ratchet même commit).
- [ ] V7d — VF-14 : `lib/formatters/date.ts:52` `'fr-FR'` en dur → `intlLocale(locale)`
  (threading si nécessaire) OU justification documentée sur place si verrou chart délibéré
  (cf. décision I2 « formatDateShort verrou chart ») — statuer, pas laisser flou.
- [ ] V7e — VF-15 : dater l'entrée `groups.go` de l'allowlist Huma
  (`json_huma_coverage_test.go`) ou la traiter.
- [ ] V7f — (issu de §7 parent, pré-existant, 5 min) : lint pré-existants au merge :
  `match_history_service.go:107` goconst `"loss"` → const `duelLabelLoss` ;
  `halo_ranks_loader.go:55` gocyclo 16>15 → extraire une branche. (Nettoyage AVANT merge
  main comme le §7 le prévoyait.)

Gate V7 : gates front V1 re-passés (cache purgé) ; `go test ./internal/platform/duckdb/...`
+ garde-rails archlint verts ; `golangci-lint run --new-from-rev=main` → 0 issue.

## LOT V8 — Contrat front↔back (généraliser la découverte A2 §7)

Objectif : plus aucun type de réponse écrit à la main dans `apps/web/src` qui diverge du
contrat réel (chaque divergence = un `undefined` silencieux à l'écran).

- [ ] V8a — Inventaire : balayer `types.ts` (et tout fichier front définissant des shapes
  de RÉPONSE API à la main) et comparer champ par champ à `generated.ts`/openapi.yaml +
  aux structs Go réellement sérialisées. Livrable : tableau divergences (type, champ,
  réel vs déclaré, consommateurs).
- [ ] V8b — Corriger le cas PROUVÉ (A2 §7) : `CareerTopMatchesResponse` (`types.ts:516`,
  `{items}` vs réel `{best_matches, worst_matches}`) + `data.top_matches_preview` lu par
  `CareerPage.tsx` mais absent du `CareerPageResponse` Go → aligner le flux de données
  Career (vérifier À L'ÉCRAN que les top matches s'affichent).
- [ ] V8c — Migrer les types réponse hand-written vers des imports de `generated.ts`
  partout où le schéma openapi existe ; pour les restants, allowlist datée.
- [ ] V8d — Garde-rail (règle 6) : test vitest fs-grep (modèle `keys.guard.test.ts`)
  interdisant toute nouvelle `interface/type *Response` manuelle hors `generated.ts` +
  allowlist décroissante datée.

Gate V8 : typecheck (cache purgé) + vitest verts ; garde-rail V8d mord (vérifié 2 sens) ;
revue visuelle de la page Career au GATE HUMAIN.

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
