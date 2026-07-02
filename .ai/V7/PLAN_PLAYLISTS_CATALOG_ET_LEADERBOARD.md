# PLAN — Catalogue playlists dynamique + Leaderboard sans trous

> Créé 2026-07-02. Déclencheur : comparaison avec LeafApp_Infinite (iBotPeaches).
> Deux constats validés sur pièces : (1) notre catalogue de playlists classées est une
> photo figée à la main (4 actives, « mesure 2026-05 ») au lieu d'être fetché en direct ;
> (2) les trous du classement mondial viennent d'une re-résolution PeopleHub redondante
> alors que le xuid est DÉJÀ dans le snapshot.
>
> Contrat d'exécution : skill `plan-execution` (ordre strict, périmètre fermé, gate par
> étape, aucune case vide à la clôture, zéro fix hors périmètre). Direction actée par
> l'utilisateur : **rester en direct (tokens Xbox), pas de proxy grunt.dotapi**.

## Objectif & critère de succès

- **A (catalogue)** : `playlists_catalog.is_active` / `is_ranked` reflètent l'état RÉEL du
  jeu, rafraîchi automatiquement depuis l'API Halo, sans édition manuelle d'un slice Go.
  Succès = la page classement et la page player listent **toutes** les playlists classées
  actuellement actives (≥ ce que montre LeafApp), triées, sans dépendre des snapshots
  scrapés comme source de vérité.
- **B (leaderboard)** : réduction mesurable des « trous » d'enrichissement du top-N.
  Succès = pour une playlist donnée, le taux de joueurs enrichis (stats non vides) passe
  de l'état actuel à ≥ 90 % (le résiduel = comptes à historique privé, 403 attendu).

## Branche Git cible

`feat/leaderboard-xuid-et-catalogue-dynamique` (une branche, commits séquentiels par
étape). Ne PAS travailler sur `main` (push main = deploy prod auto).

## Ordre des étapes (ROI décroissant / risque croissant)

B1 → A1 → A2 → B2 → A3 → C1 → C2. Chaque étape est close (gate passé) avant la suivante.
C2 (saisons) est indépendante de A/B et peut être avancée si priorité produit.

---

## Étape B1 — Leaderboard : arrêter de jeter le xuid Waypoint, supprimer PeopleHub

**RÉALITÉ VÉRIFIÉE (2026-07-02, sur pièces)** : le scraper remplit DÉJÀ
`domain.LeaderboardEntry.XUID` depuis Waypoint `__NEXT_DATA__`
(leaderboard_scraper.go:117), mais `InsertWorldCSRSnapshot`
(leaderboard_world_repo.go:629-638) NE le persiste PAS — la table
`world_csr_leaderboard_snapshots` n'a pas de colonne xuid, et un commentaire
(repo:57-58) prétend à tort que Waypoint ne publie pas de xuid (DOC INVERSÉE). La Phase C
re-résout donc via PeopleHub (single-token, 1,6 s/joueur, fragile 429) — 1er point de
perte, évitable en cessant de jeter le xuid déjà en main. Effort réel ~1 j (migration
append-only incluse), PAS ½ j.

**Périmètre fermé** :
- [x] Migration : ajouter colonne `xuid VARCHAR` à `world_csr_leaderboard_snapshots` via la
      recette append-only (ADR 0026 + `internal/migration/append_only_rebuild.go`) ; MAJ la
      vue `world_csr_leaderboard_latest` pour exposer xuid.
- [x] `InsertWorldCSRSnapshot` (repo:629-638) : ajouter `xuid` à l'INSERT (`e.XUID`).
- [x] Corriger le commentaire inversé (repo:57-58) ET le read display (repo:62 : `'' AS
      xuid` → vraie colonne) — débloque `isLocalXUID` sur le classement mondial.
- [x] `WorldSeasonGamertags` (world_leaderboard_cron.go:302) → `WorldSeasonPlayers()
      []{Gamertag, XUID}` lu depuis la table (xuid inclus).
- [x] Enrichisseur : pré-seed `a.xuidByGamertag` depuis les xuid du snapshot AVANT
      `AggregatePlayer` (aggregator.go:196 le trouvera en cache → resolver jamais appelé).
      Fallback PeopleHub UNIQUEMENT si xuid absent (lignes pré-migration, non re-scrapées).
- [x] Log : `slog.DebugContext(ctx, "world_enrich: xuid depuis snapshot", ...)` + compteur
      expvar `world_enrich_xuid_from_snapshot` (mesure du court-circuit).

**Décision produit tranchée** : PAS de backfill xuid des vieux snapshots — le prochain
scrape quotidien les remplit ; entre-temps fallback PeopleHub. Acceptable.

**Gate (commandes exactes)** :
- `cd apps/go-api && go build ./... && go test ./internal/service/ ./internal/platform/duckdb/ -run 'World|Leaderboard|Aggregat'`
- `go test -tags=integration ./internal/platform/duckdb/ -run 'World'` (migration + INSERT anti-ART).
- Test ajouté : un joueur avec xuid pré-seedé n'appelle JAMAIS le resolver (mock resolver, assert 0 call).

**Livrable indépendant** : oui. Commit `fix(leaderboard): persister le xuid Waypoint et court-circuiter PeopleHub`.

---

## Étape A1 — RÉVISÉE : source = Waypoint FetchCatalog (manifest ABANDONNÉ)

**DÉCOUVERTE 2026-07-02 (recherche offline AVANT toute sonde live)** : l'approche manifest
est INVALIDE — `discovery-infiniteugc/hi/manifests/builds/{build}/game` renvoie
maps/modes/variants mais son `PlaylistLinks` est **VIDE** (source : OpenSpartan wiki). Aucune
sonde live nécessaire ; le blocker « tokens watcher » disparaît.

**Source directe autoritative TROUVÉE, déjà scrapée** : le `__NEXT_DATA__` de la page
classement Waypoint expose `playlists` = le menu déroulant des playlists CLASSÉES ACTIVES
(**7** dans la fixture : Snipers, Doubles, Slayer, Legacy, Arena, Tactical, 1v1 Showdown —
vs **4** flaggées actives en dur = CAUSE RACINE confirmée), avec `playlistId` +
`displayName` + date de rotation. `LeaderboardScraper.FetchCatalog` (leaderboard_scraper.go:186)
renvoie DÉJÀ `(seasons, playlists []WaypointRef{ID,DisplayName})` — mais `playlists` est JETÉE
par son unique appelant (snapshot-world-leaderboard:147 `refs, _, err`). Idem `seasons` (→ C2).

**Décision produit tranchée** : `is_active` = présent dans `FetchCatalog().playlists` ;
`is_ranked` = TRUE (dropdown Waypoint = classé par construction). Playlists de
`rankedplaylists.All()` ABSENTES du dropdown → `is_active=FALSE` (retirées, conservées pour
l'historique CSR). Zéro dérivation depuis les matchs, zéro manifest, zéro réseau nouveau
(FetchCatalog est déjà appelé par le cron leaderboard).

**Périmètre fermé** :
- [~] Sonde/manifest : ABANDONNÉ (source déjà validée par la fixture leaderboard_sample.html).
      Toute l'implémentation passe en A2.

**Blocker** : AUCUN.

---

## Étape A2 — Le cron découvre les playlists ACTIVES réelles (fix page classement)

**RE-CADRAGE (sur pièces)** : la page classement dérive sa liste de playlists des SNAPSHOTS
(`SELECT DISTINCT playlist_id FROM world_csr_leaderboard_latest`, leaderboard_world_repo.go:267),
PAS de `playlists_catalog`. Or le cron ne scrapait que `rankedplaylists.Active()` (4 en dur)
→ seules 4 playlists avaient des snapshots → la page n'en montrait que 4. LE fix = faire
scraper au cron les playlists réellement actives (7) découvertes sur Waypoint.

**Périmètre fermé** :
- [x] `LeaderboardScraper.FetchActivePlaylists(ctx, ref)` → mappe la portion `playlists` de
      `FetchCatalog` en `[]domain.WorldPlaylistRef{AssetID, DisplayName}` (halo).
- [x] Port `LeaderboardScraperPort.FetchActivePlaylists` + `WorldLeaderboardCron.discoverActivePlaylists`
      (fallback statique si erreur/vide → jamais zéro playlist). `runOnceForTitle` scrape les
      playlists découvertes, plus les 4 en dur.
- [x] Rafraîchi à chaque cycle du cron (déjà quotidien). Multi-titre : hérite du gate
      `CapWorldLeaderboard` de `RunOnce` (titre sans cap → skip ; fallback statique sinon).
- [!] MAJ `playlists_catalog.is_active` (metadata) depuis Waypoint : DIFFÉRÉ. Justification :
      la page classement ne lit PAS le catalogue (elle dérive des snapshots) ; le cron n'a
      pas de writer metadata.duckdb ; le seul consommateur de `playlists_catalog.is_active`
      (FiltersService/catalog_repo) est traité en A3 (migration des consommateurs). Pas requis
      pour le fix utilisateur. Contrainte ART notée pour A3 : `playlists_catalog` sans index
      secondaire (ratchet `playlists_catalog_no_index_test.go`) → UPDATE-or-INSERT only.
- [!] `IsRanked()`/`Active()` lisent le catalogue : DIFFÉRÉ → A3 (migration consommateurs).

**Gate (passé)** :
- `go build ./...` OK ; `gofmt -l` propre ; `go test ./internal/scheduler/ ./internal/platform/halo/`
  vert, dont `TestWorldLeaderboardCron_DiscoversActivePlaylists` (scrape les découvertes, pas
  les statiques) + `TestWorldLeaderboardCron_FallbackStaticPlaylists`.

---

## Étape A3 — Source active dynamique côté consommateurs (EXÉCUTÉE)

**Valeur marginale RÉELLE (mesurée sur pièces)** : FAIBLE mais livrée. La page player affiche
DÉJÀ les rangs de toutes les playlists JOUÉES via `GetPlayerCSRs` (career.go:204). L'augment
n'ajoute que les playlists actives NON-JOUÉES (prompts « Non classé »). A3 étend cet augment
aux playlists actives RÉELLES (7) au lieu des 4 en dur.

**Conception validée (source active dynamique SANS contention metadata)** :
- Le cron (A2) écrit déjà les playlists actives réelles dans `world_csr_leaderboard_latest`
  (shared). C'est LA source dynamique à lire — PAS `playlists_catalog` (metadata.duckdb =
  writer mono-process, contention avec la sync ; ADR 0013/0016). Écrire is_active depuis le
  cron = À ÉVITER.
- Helper `activeRankedPlaylists(ctx, sharedReader, titleSlug, seasonID)` :
  `SELECT DISTINCT playlist_id FROM world_csr_leaderboard_latest WHERE title_slug=? AND season_id=?`
  → `rankedplaylists.Lookup(id)` pour nom/queue/input ; **fallback `rankedplaylists.Active()`**
  si reader nil / vide / erreur (sûr par construction).
- Threader un reader shared (nil-safe) : `SyncEngine` a `sharedProvider`/`sharedDBPath` ;
  `runCSRSnapshotSync` (2 call-sites engine_postsync.go:101 & :453) → `syncPlayerCSRs` →
  `augmentWithActiveRankedCSRs`. Signature élargie d'un `sharedReader`.

**Périmètre fermé** :
- [x] `SyncEngine.activeRankedPlaylists(ctx)` : lit les playlists du DERNIER batch de
      `world_csr_leaderboard_snapshots` (season-agnostic — le format saison Waypoint diffère
      de `e.csrSeasonID`) via `e.sharedProvider` (RO) ; `rankedplaylists.Lookup` pour le
      libellé, playlist hors référence conservée (catalogue-first complète). **Fallback
      `Active()`** si provider nil / table vide / erreur (nil-safe, jamais < historique).
- [x] Threadé : `runCSRSnapshotSync` → `syncPlayerCSRs` → `augmentWithActiveRankedCSRs`
      (param `activePlaylists`; vide → `Active()`). 3 call-sites de test mis à jour.
- [~] Sélecteur page classement lit une source dynamique : DÉJÀ le cas (dérive des snapshots
      que A2 remplit avec les 7 actives). Rien à changer.
- [!] Paralléliser la boucle CSR par-playlist : NON traité (optionnel, hors périmètre du fix ;
      la boucle est best-effort séquentielle, N≈7 playlists → coût négligeable). Optimisation
      pure, à ouvrir séparément si besoin.

**Gate (passé)** :
- `go build ./...` OK ; `gofmt -l` propre ; `go test ./internal/sync/ -run 'CSR|Playlist|Career|Augment|SeedPlaylists'` vert,
  dont `TestAugmentWithActiveRankedCSRs_UsesProvidedList` (l'augment interroge la liste fournie,
  playlist hors référence incluse) + tests existants (fallback `Active()` via `nil`).

---

## Étape B2 — Leaderboard : service-record au lieu d'agrégation par-match

**Pourquoi** : 1 appel/joueur au lieu de milliers ; supprime les points d'attrition 2/3
(history + match-stats). Approche LeafApp `serviceRecord`.

**Périmètre fermé** :
- [x] **Vérifier l'existant** : `platform/halo/compare_provider.go` fetche déjà le service
      record (`FetchServiceRecord` lifetime, `FetchSeasonServiceRecord` saison→count). Endpoint
      `/hi/players/{p}/Matchmade/servicerecord?seasonId=[&isRanked=]`. La réponse porte des
      Subqueries `PlaylistAssetIds` → un filtre `playlistAssetId` existe (parité SPNKr).
- [!] Remplacer `collectPlayerMatches`/`getMatch` par un fetch service-record par (saison,
      playlist). **BLOQUÉ (règle 3 — validation live token-gated requise, PAS un report
      momentum)** : `FetchSeasonServiceRecord` ne lit AUJOURD'HUI que `MatchesCompleted` ; la
      forme complète du service-record FILTRÉ PAR `playlistAssetId` (CoreStats par playlist)
      n'est **pas prouvée** contre l'API. Bâtir tout l'agrégateur de STATS sur une forme non
      vérifiée = imprudent → sonde live d'abord (compte watcher, endpoint public). Design
      turnkey ci-dessous ; agrégateur/tests intacts tant que non validé.
- [!] 403 (privé) = joueur gardé + stats vides + flag ; erreur par-joueur non-fatale.
      Dépend de l'item ci-dessus (même refonte agrégateur). NB : le non-fatal par-JOUEUR
      existe déjà (Run/errgroup) ; c'est le non-fatal par-MATCH qui manque, mooté par le swap.

**Design validé (turnkey pour session dédiée, après sonde live)** :
1. `sync.HaloAPIClient.GetSeasonPlaylistServiceRecord(ctx, xuid, seasonID, playlistID)` →
   GET `{haloStatsHost}/hi/players/xuid(N)/Matchmade/servicerecord?seasonId=&playlistAssetId=`
   (mirroir `GetPlaylistCsr`), parse CoreStats → `domain.WorldServiceRecord`. 404/403→(nil,nil).
2. `PooledHaloClient.GetSeasonPlaylistServiceRecord` via `doPublic`.
3. `analysis.WorldStatsFromServiceRecord(...)` → `WorldPlayerSeasonStats`. **Mapping validé** :
   KDA = Kills+Assists/3−Deaths (Σ per-match = linéaire, exact) ; Accuracy =
   (ShotsHit/ShotsFired)×MatchCount (approx de Σ per-match% ; la lecture passe kda/accuracy
   BRUTS, sans ÷match_count — vérifié worldPlayerStatsQuery) ; tie/dnf=0 (non fournis) ;
   kills/deaths/assists/damage/playtime/medals = agrégat (= sommes, exact).
4. Nouvelle interface source `WorldServiceRecordSource` (remplace `WorldMatchSource`) ;
   `AggregatePlayer` boucle sur `cfg.RankedPlaylists` : 1 SR/(joueur,playlist), émet une ligne
   si MatchesCompleted>0. Supprimer `collectPlayerMatches`/`getMatch`/cache/singleflight
   (code mort). Réécrire les fakes des 4 tests (fakeMatchSource→fakeServiceRecordSource).

**RÉSULTAT (sonde live 2026-07-02) — design service-record NON VIABLE** :
- Format saison VALIDÉ : `Csr/Seasons/CsrSeason13-2.json` (chemin CMS ; le format Waypoint
  `csrseason13-2` → 404). Saison SEULE : JGtm = 367 matchs, CoreStats complets. ✔
- **`playlistAssetId` NON SUPPORTÉ** : sur les 16 playlists classées, AUCUNE ne renvoie de
  données malgré 367 matchs saison. Le service-record ne donne QUE l'agrégat par SAISON (toutes
  playlists confondues), pas par playlist. → impossible de peupler `world_player_season_stats`
  (clé saison×playlist) depuis cet endpoint. **Hypothèse B2 originale invalidée.**
- Artefacts de sonde (endpoint `GetSeasonPlaylistServiceRecord`, `domain.WorldServiceRecord`,
  `cmd/probe-service-record`) SUPPRIMÉS (code mort, règle 7) — finding préservé git + ce journal.

**PIVOT B2 → hardening par-match (EXÉCUTÉ)** : le seul moyen d'avoir le par-playlist reste
l'agrégation par-match ; on la REND ROBUSTE au lieu de la remplacer.
- [x] `collectPlayerMatches` : un match illisible (403/404/timeout après retries) est IGNORÉ
      (`continue`) au lieu d'annuler tout le joueur — LE fix des trous. Compteur expvar
      `world_enrich.match_skipped`.
- [x] Erreur d'historique APRÈS collecte partielle → stats conservées (avant : `return nil,err`
      jetait le partiel) ; échec dès la 1re page → erreur remontée (signal préservé).
- [x] Dichotomie offset en échec → scan linéaire (fallback) au lieu d'abandon.
- [~] Non-fatal par-JOUEUR : déjà en place (Run/errgroup).

**INCIDENT TOKEN (résolu)** : 1re version de la sonde sans persistance du RT roté → rotation
JGtm non sauvée. Vérifié ensuite : JGtm auth OK (RT a survécu), persistance corrigée avant
suppression. RAS.

**Gate (passé)** : `go build ./...` OK ; `gofmt` propre ; `go test ./internal/service/ -run
'World|Enrich|Aggregat|Retry|SkipsUnreadable'` vert, dont `TestAggregate_SkipsUnreadableMatch`
(match 404 ignoré, joueur garde les autres + compteur incrémenté).

---

## Étape C1 — Frontend : tri + affichage placement précédent (FULL-STACK, non exécutée)

**RE-CADRAGE (sur pièces)** : ce n'est PAS du polish frontend — les 2 items exigent des
DONNÉES BACKEND non exposées aujourd'hui par `CareerCSRRank` (`apps/web/src/lib/api/types`,
rendu par `CareerRankingBlock.tsx`). Le classement (LeaderboardBlock) a DÉJÀ tri + trends +
rank_delta ; C1 concerne la page PLAYER.

**Périmètre fermé** :
- [!] Trier `is_active` d'abord : NON exécuté. Nécessite d'ajouter un flag `is_active` par
      playlist à la réponse CSR carrière (career_service + type TS). Full-stack.
- [!] Placement saison précédente en delta : NON exécuté. `CareerCSRRank` ne porte pas la
      valeur N-1. Deux voies : (a) backend expose `prev_value/prev_tier` par playlist
      (join player_csr_snapshots saison N-1) ; (b) frontend-only : `useCareerCSRs` current +
      previous (availableSeasons[1]) + join par playlist_id + delta UI. (b) est faisable sans
      backend mais reste une vraie feature (2 queries, i18n, tokens).
- [!] i18n FR/EN + tokens : dépend des items ci-dessus.

**Raison du non-go (delivery-checklist)** : full-stack + gate vitest hors sandbox, à exécuter
en contexte frais — précipiter à grande profondeur de session risquerait la page player
elle-même. Design (a)/(b) prêt. Aucun blocage EXTERNE (exécutable en session dédiée).

**Gate** :
- `make check-types && make test-web`

---

## Étape C2 — Saisons : persister + surfacer la liste (numéro + sous-saison)

**Pourquoi** : LeafApp affiche les saisons avec numéro + nom de sous-saison (« X-Infinite »).
Waypoint expose déjà cette liste et notre scraper la parse (`Seasons []{SeasonID,
DisplayName}`, scraper:247,262-265) mais la jette. Coût faible, valeur : sélecteur de
saison correct + placement « saison précédente » nommé.

**Périmètre fermé** :
- [x] **Enquête** (fixture leaderboard_sample.html) : `seasonId` = identifiant CSR
      (`csrseason13-2`, `csrseason12-1`…) ; `displayName` = nom d'Operation. Mapping réel :
      `csrseason13-2`→"Infinite", `csrseason12-1`→"Shadows" (FR "Ombres"),
      `csrseason11-1`→"Last Stand" (FR "Dernier bastion"). Les `translations` par locale sont
      DANS le payload Waypoint. « X-Infinite » = `csrseasonX-Y` (n° saison CSR) + Operation.
- [!] Persister la liste des saisons (table `season_catalog` ou colonnes) : NON exécuté —
      full-stack, contexte frais requis. NB : le scraper renvoie DÉJÀ seasons via FetchCatalog
      (jeté chez l'appelant), donc la persistance = capter cette portion + upsert ART-safe.
- [~] Surfacer via sélecteur : DÉJÀ partiellement fait — `useLeaderboardCatalog` renvoie
      `seasons[].display_name` (leaderboard) et `CareerRankingBlock` a `availableSeasons`. C2
      n'ajoute que la persistance autoritative des noms+traductions Waypoint.
- [!] Multi-titre gate capability : dépend de la persistance (item ci-dessus).

**Raison du non-go (delivery-checklist)** : idem C1 — full-stack + vitest hors sandbox, à
faire en contexte frais. Enquête (le cœur de ta question saisons) = LIVRÉE ci-dessus.

**Gate (quand exécuté)** :
- `cd apps/go-api && go test ./internal/platform/halo/ ./internal/platform/duckdb/ -run 'Season|Leaderboard'`
- `make check-types && make test-web`

---

## Découvertes (fixes hors périmètre repérés — À NE PAS traiter ici)

- **Doc inversée** (leaderboard_world_repo.go:57-58) : commentaire « Waypoint ne publie pas
  de xuid » contredit par scraper:117 qui le parse. Traité DANS B1 (même code).
- **Seasons Waypoint déjà scrapées mais inexploitées** : `parseLeaderboardPage` capture
  `Seasons []{SeasonID, DisplayName, translations}` (scraper:247,262-265). La fixture
  montre `csrseason13-2` → DisplayName « Infinite » + traductions FR complètes (Ombres,
  Dernier bastion…) : « X-Infinite » = n° saison CSR + nom d'Operation. Ouvert en C2.
- **Manifest ABANDONNÉ (A1 pivot, 2026-07-02)** : `discovery-infiniteugc/.../game`
  PlaylistLinks VIDE (OpenSpartan wiki). Remplacé par `FetchCatalog().playlists` déjà scrapé
  (7 playlists actives vs 4 en dur = cause racine). A1/A2 révisées. Aucun réseau nouveau.
- **`FetchCatalog().playlists` jeté** (snapshot-world-leaderboard:147 `refs, _, err`) : la
  liste autoritative des playlists classées actives est fetchée puis ignorée. Traité en A2.
- **`playlists_catalog` sans index secondaire** (ratchet `playlists_catalog_no_index_test.go`)
  : UPDATE OK sans index ; ne JAMAIS recréer idx_playlists_catalog_active (corruption ART).

## Protocole de reprise de session

- Avancement = cases cochées de ce fichier + entrées `.ai/thought_log.md`.
- État final session : B1 [x d58528501] · A2 [x dfce681f4] · A3 [x fddf71f5c] · B2 [x
  7a49ab0a0 — pivot hardening après sonde live prouvant playlistAssetId non supporté] ·
  C1 [!] frontend (voie b : 2e query + delta + NOUVELLE clé i18n regen + vitest) ·
  C2 [x enquête saisons ; persistance [!] full-stack]. Tests verts, tout poussé.
- Reprendre en session FRAÎCHE (toolchain web) : C1 (voie b, previous-delta page player —
  nécessite édition career.toml + regénération i18n + vitest hors sandbox), puis C2
  (persistance seasons Waypoint). Aucun blocage externe ; friction = toolchain frontend.
- Avant de coder une étape : rouvrir les fichiers cibles (le code a pu bouger).

## Statut de clôture

Aucune case vide autorisée : `[x]` fait / `[~]` couvert ailleurs (réf) / `[!]` non traité
(justification). Entrée thought_log obligatoire par étape close.
