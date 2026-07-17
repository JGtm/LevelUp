# PLAN_CORRECTIFS_REVUE_2026-07 — solde des findings de la revue 10 jours

> Référence findings : `.ai/REVUE_CODE_2026-07-17.md` (revue adversariale
> 24fc02f2f..HEAD — 31 confirmés, verdicts avec citations).
> Contrat d'exécution : skill `plan-execution` (ordre strict, une étape à la
> fois, AUCUN report d'item exécutable, statut obligatoire par item, vérifier
> sur pièces avant de coder ET avant de cocher, zéro fix hors périmètre —
> consigner en Découvertes).

## Objectif et critère de succès

Solder TOUS les findings confirmés de la revue (dysfonctionnels ET robustesse —
décision utilisateur 2026-07-17 : rien n'est exclu). Succès = chaque item
statué `[x]` / `[~]` (couvert ailleurs, référence) / `[!]` (justification
écrite) ; gates verts par lot ; baseline tests intacte ; aucune nouvelle issue
lint (`--new-from-merge-base=main`).

- **Branche** : `fix/revue-2026-07-correctifs` (depuis main local, 1 branche,
  N commits — 1 commit par lot minimum).
- **Worktree** : `c:\Users\Guillaume\Downloads\Scripts\LevelUp-wt-revue-correctifs`.
- **Effort estimé** : lots A/E rapides, B/D moyens, C lourd, F moyen —
  ~5-6 jours effectifs au total.
- **Exécution** : agents Opus séquentiels (JAMAIS deux builds Go en parallèle —
  corruption cache Windows). Le superviseur ne code pas : git/merges/CI/journal.

## Décisions tranchées AVANT exécution (ne pas ré-arbitrer en cours de route)

- **D1 — Sémantique crons** : `ReportCronRun` reçoit l'erreur agrégée réelle du
  cycle ; un cycle partiellement échoué = échec avec cause. Appliqué aux 5 crons
  actuellement « toujours verts » (world_leaderboard, spartan_customization,
  data_health_check, catalog_refresh, asset_name_sweep).
- **D2 — Emojis Discord** : le payload Discord est du contenu produit →
  exemption datée en commentaire dans `internal/notify/discord.go` ; la sortie
  CLI (`cmd_data.go:233`) est purgée.
- **D3 — Filets squash (M2)** : réparation par INTROSPECTION du schéma (jamais
  par sentinelle), auto-guérison au boot, idempotente (doctrine convergent
  sync). Pas de cmd manuel. Fixtures de DBs aux états intermédiaires exigées.
- **D4 — Identité** : lot A = fix ponctuel season pass + garde-rail ratchet
  (aucune 5e occurrence possible pendant le chantier) ; lot B = refactor
  « sujet en paramètre explicite » (fix profond).
- **D5 — Flush notifications** : `visibilitychange`/`pagehide` + envoi
  `keepalive` (pas `beforeunload` seul).
- **D6 — Recovery par construction (F13)** : ON LE FAIT, périmètre = handle de
  lecture player-DB n'exposant que les variantes `*Recovered`.
- **D7 — Doctrine tokens vs sujet (validée utilisateur 2026-07-17)** :
  emprunter un token VALIDE du pool pour authentifier un fetch est un design
  assumé (le pool existe pour ça) — ne jamais « corriger » l'emprunt lui-même.
  Les invariants sont : (a) le SUJET de la requête (quel joueur) vient de la
  PAGE, jamais du contexte ambiant ; (b) ne JAMAIS persister des données sous
  un xuid différent de celui pour lequel l'API les a servies ; (c) le
  budget/quota se débite au PORTEUR réel du token (B1). Cas particulier des
  endpoints ownership-scoped upstream (BP/défis : fetchables uniquement pour
  soi) : viewer ≠ sujet → servir le persisté du sujet, pas de fetch-écriture.

## Gates (commandes exactes)

- Go ciblé : `cd apps/go-api && go test ./internal/<paquets touchés>/...`
- Go intégration (lots C, F7, et clôture) :
  `cd apps/go-api && go test -tags=integration -p 1 ./internal/<filtre ANCRÉ>/...`
  (jamais de filtre non ancré — incident LOT B).
- Lint : `golangci-lint run --timeout 5m --new-from-merge-base=main` (v2.12.2).
- Web : `make check-types` ; vitest ciblé puis `make test-web` (hors sandbox).
- Clôture chantier : suite complète `go test -tags=integration -p 1 ./...` +
  `scripts/check_test_baseline.sh tests` (~13 min) + skill `delivery-checklist`.
- Chaque lot : entrée `thought_log.md` AVANT commit (règle obligatoire).

---

## LOT A — Correctness rapides (Go + web)

- [ ] A1 (ID1) `internal/api/wire/registry_auth.go:105` — corriger le SUJET du
      season pass (décision D7). ÉTAPE 1 : vérifier sur pièces la sémantique
      upstream de GetChallenges/GetBattlePass (les endpoints Waypoint BP/défis
      sont-ils ownership-scoped — fetchables uniquement pour le porteur du
      token ? indice existant : pattern « fetch live 403 → cache 24 h »).
      ÉTAPE 2 : si ownership-scoped (probable) → quand porteur des tokens ≠
      joueur de la page : PAS de fetch live ni de persist, servir les
      snapshots persistés du joueur de la page (chemin fallback existant) ;
      quand porteur = joueur de la page : comportement actuel inchangé. Si NON
      ownership-scoped → `forcePageIdentityXUID` comme ligne 53. Dans TOUS les
      cas : plus jamais d'écriture des données de Y dans la DB de X. Tests :
      les deux cas viewer=sujet / viewer≠sujet (étendre
      `registry_auth_enrich_test.go` + test du chemin service).
- [ ] A2 (ID4) — garde-rail ratchet : nouveau test dans `internal/api/wire/`
      qui recense les constructeurs de contexte appelant `enrichWithHaloTokens`
      et ÉCHOUE si l'un d'eux n'applique pas `forcePageIdentityXUID` (allowlist
      d'exemptions datées, modèle `no_slug_comparison_test.go`).
- [ ] A3 (W1) `apps/web/src/features/auth/XboxLoginPage.tsx:70` — `fetchQuery`
      avec `staleTime: 0` (lecture réseau garantie post-login). Vérifier que le
      store bootstrap consommé par la garde onboarding est bien resynchronisé
      par cette lecture. Test vitest : cache anonyme préchargé → onAuthorized →
      redirection dashboard, pas onboarding.
- [ ] A4 (M3) `internal/migration/registry.go:225-230` — le chemin DM-5
      (`recordSupersededBaseline`) exécute AUSSI l'ensure additif idempotent :
      `ensureBaselinePlayerV1AdditiveColumns` ÉTENDU aux colonnes render de
      `challenge_snapshots` (title/description/image_url/display_path) + au
      `CREATE TABLE IF NOT EXISTS engagement_response_bins`. Test : fixture DB
      « sentinelle présente, colonne expected_win_prob absente » → boot →
      colonne présente, persist LUSR OK.
- [ ] A5 (LB1) `internal/sync/engine_postsync_csr.go:36-40` — remplacer le
      `MAX(fetched_at)` global par la sélection partitionnée par playlist
      (réutiliser `world_csr_leaderboard_latest`/`_by_batch` ou MAX par
      playlist_id). Test : 2 playlists à fetched_at distincts → les 2
      retournées.
- [ ] A6 (E5) — conversions recovery restantes :
      `milestones_earned_repo.go:69` → `QueryRowRecovered` ;
      `career_progression_partial.go:86`, `career_live_repo.go:213`,
      `engagement_score_repo.go:202`, `fanout_repo.go:93` → `ExecRecovered`.
      Étendre `player_db_recovery_routing_test.go` au motif `.Exec(`.
- [ ] A7 (ME1) `cmd/cleanup_media_index/main.go:126` — LIKE avec `ESCAPE '\'`
      + échappement de `_`/`%` dans le motif (helper testé unitairement,
      aligné sur la sémantique HasPrefix de l'indexeur).
- [ ] A8 (AU1) `internal/platform/auth/refresh_user_xsts.go` — persister le RT
      rotaté IMMÉDIATEMENT après le refresh OAuth (avant XSTS), calqué sur
      `persistRotatedRT` (`access_token_store_first.go:106-108`). Test : échec
      XSTS transitoire → store contient le RT rotaté.
- [ ] A9 (AU3) `internal/platform/auth/access_token_store_first.go:58` —
      logger l'erreur de `store.Load` (`slog.ErrorContext(ctx, ..., "err", err)`)
      avant la bascule legacy ; l'échec store ne doit plus être invisible dans
      la télémétrie.
- [ ] A10 (W4) `apps/web/src/features/admin/monitoring/DetectionsPanel.tsx:49`
      — `window.prompt` retourne null (Annuler) → early-return, AUCUNE
      mutation. Test vitest.
- [ ] A11 (W3) `apps/web/src/features/home/HomeRecentPlaylistsCard.tsx:127,159-163`
      — « Non classé » / « En placement (x/y) » / « Rang » via i18n.ts (parité
      FR/EN par `Record<Locale, T>`).
- [ ] A12 (CV4) — `cmd/levelup/cmd_data.go:233` : retirer l'emoji ;
      `internal/notify/discord.go` : commentaire d'exemption daté (D2).
- [ ] A13 (CV1) `cmd/engagement-calibrate/main.go` — chemins via
      `PathResolver.PlayersRootDir`/`PlayerDBPath` ; `log.Printf`/`log.Fatalf` →
      slog ; supprimer le champ `Bins` mort et le paramètre `ref` ignoré
      d'`autoVerdict` (corriger la godoc).

**Gate A** : go test ciblés (wire, migration, sync, platform/auth,
platform/duckdb, cmd) ; `make check-types` ; vitest ciblé (auth, admin, home) ;
lint new-from-merge-base ; entrée thought_log ; commit `fix(revue): lot A ...`.

## LOT B — Identité en profondeur

- [ ] B1 (ID3) `internal/service/career_live_fetcher.go:158` — le limiteur de
      budget est clé sur le XUID DU PORTEUR DES TOKENS (compte connecté), pas
      sur le xuid de page : introduire la clé de contexte « tokensOwnerXUID »
      posée là où les tokens sont posés, et `WithLimiter(ratebudget.ForXUID(owner))`.
      Test : page X consultée par session Y → bucket Y débité.
- [ ] B2 (ID4 profond) — le sujet devient un paramètre explicite :
      `career_live_service.go:167-168` ne lit plus le sujet ambiant ; les
      lecteurs passent par `GetSpartanIdentityFor(ctx, xuid)` (ou équivalent).
      Périmètre : call-sites de `GetSpartanIdentity` uniquement. Le ratchet A2
      reste en place comme non-régression.

**Gate B** : go test service + wire ; intégration filtrée ancrée si persist
touché ; thought_log ; commit.

## LOT C — Filets squash M2 (le morceau lourd)

- [ ] C1 — réparation par introspection (D3) : au boot player-DB, détecter
      l'ancien schéma `player_csr_snapshots` (colonnes `id`/`written_at`
      absentes) → rejouer la conversion append-only (code de `37264462f`,
      réintroduit comme repair conditionnel idempotent hors-sentinelle).
- [ ] C2 — colonnes render `challenge_snapshots` : couvert par l'ensure étendu
      de A4 → statuer `[~]` avec vérification sur fixture mi-bloc.
- [ ] C3 — fixtures DBs états intermédiaires : pré-2026-05-24 (ancien
      player_csr_snapshots), fenêtre 05-24→05-28 (sentinelle sans
      expected_win_prob), pré-06-22 (sans render), pré-07-11 (sans
      response_bins). Test de convergence : boot → `OpenPlayerDB` OK, vue
      `player_csr_snapshots_latest` liée, persist LUSR et challenges OK.
- [ ] C4 — `sync/schema.go` : un échec de bind de vue au boot ne doit plus
      produire un échec permanent silencieux : log ERROR + déclenchement de la
      réparation C1 (pas de panic, pas d'avalement).

**Gate C** : `go test -tags=integration -p 1 ./internal/migration/... ./internal/persist/...`
(filtre ancré) + tests fixtures ; thought_log ; commit.

## LOT D — Efficacité (VPS 2 vCPU / 2 Go)

- [ ] D1 (E1) `internal/service/engagement_player_service.go` — mémoïser
      coef+bins par `mode_category` en début de GetTimeseries (≤4 requêtes) ;
      cacher le check d'existence de table dans le repo (une fois par
      ouverture). Test : compteur de requêtes sur 200 matchs → ≤4.
- [ ] D2 (E3) `internal/api/wire/registry_monitoring_freshness.go` — une
      requête groupée par titre (`WHERE mp.xuid IN (...) GROUP BY mp.xuid`) sur
      le shared reader ; ne plus résoudre les player-DBs (et ne JAMAIS créer de
      DB pour les profils auth_only). Test.
- [ ] D3 (E4) `internal/migration/monitoring_schema.go` + store — rétention
      CapAndSweep (pattern notifications, caps constantes nommées) sur
      `detection_events`, `detection_status_events`, `cron_runs`. Test.
- [ ] D4 (E2) `apps/web/src/features/explorer/ExplorerPage.tsx` —
      `include_briefing: mode === 'matches'` (+ query non lancée quand
      inutilisée) ; debounce 250 ms de l'input match-ID (pattern
      GamertagSearchInput). Tests vitest.

**Gate D** : go test service/wire/migration ; vitest explorer ; thought_log ;
commit.

## LOT E — UX web restants

- [ ] E1 (W2) `apps/web/src/features/match-view/MatchTugOfWarChart.tsx` —
      rétablir les tooltips item des séries kills/vagues (DEC-2 : « gamertag —
      mm:ss ») tout en conservant le comportement axis voulu par le passage à
      `'axis'` (formatter axis n'affichant que les bins, tooltips item
      par-série effectifs). Test avec mock echarts-for-react.
- [ ] E2 (W5) `apps/web/src/features/notifications/NotificationsBell.tsx` —
      flush des ids en attente sur `visibilitychange`/`pagehide` avec envoi
      `keepalive` (D5). Test.

**Gate E** : `make check-types` + vitest ciblé ; thought_log ; commit.

## LOT F — Duplications, dette, robustesse

- [ ] F1 (R2) — helper unique outcome int→valeur (`apps/web/src/lib/`), contrat
      ADR 0006, défaut EXPLICITE documenté ; migrer les 4 copies
      (ExplorerBriefing.logic, session-detail/_shared, TimeseriesPage.summary,
      SquadSynergiesPage) ; garde-rail grep interdisant les mappings locaux.
- [ ] F2 (R1) — helper « plus longue série » dans `internal/analysis/` ;
      migrer les 4 copies Go (briefing_streaks, highlights_tiles, synthesis,
      detectTilt) ; garde-rail.
- [ ] F3 (R4) — `formatSignedFixed` dans `lib/formatters/` ; migrer les 3
      copies strictes ; glyphe unique '−' (U+2212) ; garde-rail (3e copie).
- [ ] F4 (R5) — une seule fonction coalesce dans `internal/service` ; migrer.
- [ ] F5 (R7) — `deltaToken` exporté depuis `ExplorerBriefing.logic.ts`,
      2 copies supprimées.
- [ ] F6 (R8) `internal/analysis/campaign_exclusion.go` — `quotedIDList` +
      builder commun aux deux fonctions SQL.
- [ ] F7 (R6) — seeder synthétique : partager les listes de colonnes avec
      `persist` (constantes exportées ou helpers d'insert réutilisés) + test
      qui CASSE si les colonnes du seeder divergent de persist (protège la
      recette ADR 0026).
- [ ] F8 (ME3) `internal/platform/duckdb/media_repo_registry.go:205` —
      remplacer le mini-framework à closures par un helper direct appelé 3
      fois (bulk resolve conservé).
- [ ] F9 (CV5) `internal/api/wire/post_sync_deltas.go:280` — variadique →
      paramètre struct obligatoire (mise à jour mécanique des ~25 call-sites
      de test).
- [ ] F10 (LB2, décision D1) — `runOnceForTitle` retourne error ; RunOnce
      agrège et passe l'erreur réelle à `ReportCronRun` ; appliquer aux 5
      crons concernés ; tests cronstatus (échec partiel → failure visible).
- [ ] F11 (LB3) — graine de découverte de saison = dernière saison persistée
      dans les snapshots, constante `csrseason13-2` en simple fallback. Test.
- [ ] F12 (AU4) — provenance du token (famille de client) persistée dans
      `UserTokens` à l'acquisition ; préfixe RpsTicket déterministe ; retry
      conservé en filet AVEC log WARN. Test.
- [ ] F13 (E5 profond, décision D6) — handle de lecture player-DB n'exposant
      que les variantes `*Recovered` (fermeture par construction des 3 classes
      de trous du garde-rail grep). Migrer les lecteurs player-DB vers ce type.

**Gate F** : go test paquets touchés + intégration filtrée ancrée (persist/
seeder) ; vitest lib/features migrées ; lint ; thought_log ; commit.

## Clôture chantier

- [ ] Z1 — suite complète : `go test -tags=integration -p 1 ./...` (apps/go-api).
- [ ] Z2 — `scripts/check_test_baseline.sh tests` (baseline intacte).
- [ ] Z3 — `make check-types` + `make test-web` complets.
- [ ] Z4 — skill `delivery-checklist` ; statuer TOUS les items du plan ;
      section Découvertes soldée (traitée ou consignée) ; entrée thought_log de
      clôture.
- [ ] Z5 — superviseur : merge dans main + prévenir l'utilisateur AVANT push
      (push main = deploy prod) ; revue visuelle utilisateur au merge.

## Protocole de reprise de session

1. Lire ce plan (version du WORKTREE, qui fait foi pendant le chantier) : le
   premier item non statué du premier lot incomplet = point de reprise.
2. Lire les entrées thought_log du chantier (`[2026-07-*] LOT x — revue
   correctifs`).
3. `git -C <worktree> log --oneline -10` pour l'état des commits.
4. Un seul agent à la fois dans le worktree ; vérifier qu'aucun agent/build
   n'est vivant avant d'en lancer un autre (link.exe orphelins → kill).

## Découvertes (hors périmètre — consigner ici, NE PAS traiter)

(vide)
