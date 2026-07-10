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
- [ ] Migration : ajouter colonne `xuid VARCHAR` à `world_csr_leaderboard_snapshots` via la
      recette append-only (ADR 0026 + `internal/migration/append_only_rebuild.go`) ; MAJ la
      vue `world_csr_leaderboard_latest` pour exposer xuid.
- [ ] `InsertWorldCSRSnapshot` (repo:629-638) : ajouter `xuid` à l'INSERT (`e.XUID`).
- [ ] Corriger le commentaire inversé (repo:57-58) ET le read display (repo:62 : `'' AS
      xuid` → vraie colonne) — débloque `isLocalXUID` sur le classement mondial.
- [ ] `WorldSeasonGamertags` (world_leaderboard_cron.go:302) → `WorldSeasonPlayers()
      []{Gamertag, XUID}` lu depuis la table (xuid inclus).
- [ ] Enrichisseur : pré-seed `a.xuidByGamertag` depuis les xuid du snapshot AVANT
      `AggregatePlayer` (aggregator.go:196 le trouvera en cache → resolver jamais appelé).
      Fallback PeopleHub UNIQUEMENT si xuid absent (lignes pré-migration, non re-scrapées).
- [ ] Log : `slog.DebugContext(ctx, "world_enrich: xuid depuis snapshot", ...)` + compteur
      expvar `world_enrich_xuid_from_snapshot` (mesure du court-circuit).

**Décision produit tranchée** : PAS de backfill xuid des vieux snapshots — le prochain
scrape quotidien les remplit ; entre-temps fallback PeopleHub. Acceptable.

**Gate (commandes exactes)** :
- `cd apps/go-api && go build ./... && go test ./internal/service/ ./internal/platform/duckdb/ -run 'World|Leaderboard|Aggregat'`
- `go test -tags=integration ./internal/platform/duckdb/ -run 'World'` (migration + INSERT anti-ART).
- Test ajouté : un joueur avec xuid pré-seedé n'appelle JAMAIS le resolver (mock resolver, assert 0 call).

**Livrable indépendant** : oui. Commit `fix(leaderboard): persister le xuid Waypoint et court-circuiter PeopleHub`.

---

## Étape A1 — Énumérateur de manifest (sonde + implémentation)

**Pourquoi** : source directe autoritative des playlists actives = les `PlaylistLinks` du
manifest de build (`discovery-infiniteugc`). L'auth Spartan+clearance et le host sont déjà
câblés ([discovery_client.go:23,226-231](../../apps/go-api/internal/platform/halo/discovery_client.go)).

**Décision produit tranchée** : `active` = présent dans le manifest de build courant ;
`ranked` = la config playlist (déjà fetchable via `GetPlaylistConfig`) porte le flag, à
défaut = CSR endpoint non-null. On NE dérive PAS des matchs (règle actée du package
`rankedplaylists`).

**Périmètre fermé** :
- [ ] **Sonde read-only d'abord** : `cmd/probe-playlist-manifest/main.go` (CLI testée, PAS
      de sonde jetable) qui fetch le manifest de build courant et dump les `PlaylistLinks`
      (AssetId, VersionId) + tente `GetPlaylistConfig` sur 1 playlist. But : valider la
      forme JSON exacte (endpoint `hi/manifests/builds/{build}/game` — à confirmer par la
      sonde ; le numéro de build vient de gamecms/discovery, à identifier dans la sonde).
- [ ] Consigner la forme validée dans le docstring (pattern « validé contre l'API
      {date} », comme le reste du package halo).
- [ ] `HaloProvider.FetchActivePlaylists(ctx) []PlaylistRef{AssetID, VersionID}` +
      `parseGameManifest` pur (testable sans réseau, fixture JSON).
- [ ] Tests : `parseGameManifest` sur fixture (cas nominal + manifest vide).

**Gate** :
- Sonde exécutée, sortie collée dans le thought_log (forme JSON réelle).
- `cd apps/go-api && go test ./internal/platform/halo/ -run 'Manifest|Playlist'`

**Blocker documenté** : la sonde nécessite des tokens Spartan valides (compte watcher). Si
tokens morts → diagnostiquer (ADR 0023), JAMAIS re-capturer. C'est le seul point réseau.

---

## Étape A2 — Rafraîchissement du catalogue depuis le manifest

**Périmètre fermé** :
- [ ] `internal/ops/catalog_refresh.go` (existant) OU service dédié : pour chaque
      `PlaylistRef` du manifest → `GetPlaylistConfig` (nom localisé + ranked) → upsert
      `playlists_catalog(uuid, name, is_active=TRUE, is_ranked, version_id)`. Les playlists
      absentes du manifest → `is_active=FALSE` (retirées du matchmaking, conservées pour
      l'historique CSR).
- [ ] Le slice hardcodé `rankedplaylists.all` devient un **fallback/seed initial**, plus la
      source de vérité runtime. `IsRanked()` lit le catalogue rafraîchi (ne pas casser les
      callers migration : garder la fonction, changer la source).
- [ ] Cron : rafraîchir à chaque démarrage + 1×/jour (réutiliser le scheduler existant).
      Kill-switch documenté (date bascule + critère retrait, règle CLAUDE.md n°11).
- [ ] Multi-titre : gate sur capability (`ranked_playlists` / discovery UGC), pas sur
      `slug=="halo_infinite"`. Titre sans discovery → no-op propre (log), pas de panic.

**Gate** :
- `cd apps/go-api && go test ./internal/ops/ ./internal/migration/ -run 'Catalog|Playlist|Ranked'`
- `go test -tags=integration ./internal/platform/duckdb/ -run 'Catalog'` (upsert anti-ART).
- Vérif manuelle : après refresh, `duckdb .../metadata.duckdb "SELECT name,is_active,is_ranked FROM playlists_catalog WHERE title_slug='halo_infinite' AND is_active"` liste > 4 playlists si le jeu en a plus.

---

## Étape A3 — Une seule source de vérité `active` + CSR sur catalogue dynamique

**Périmètre fermé** :
- [ ] Page classement : le sélecteur de playlists lit `playlists_catalog.is_active` (issu de
      A2), plus les snapshots comme données affichées — pas l'inverse
      (leaderboard_world_repo.go:267 ne doit plus être la source de la liste).
- [ ] Post-sync CSR (career.go:242) : itérer sur `rankedplaylists.Active()` → **actives du
      catalogue dynamique** (pas les 4 en dur).
- [ ] Paralléliser la boucle CSR par-playlist via le pool (rate limiter par token déjà là) ;
      borner la concurrence. Erreur par-playlist = log + continue (pas d'abort global).

**Gate** :
- `cd apps/go-api && go test ./internal/sync/ -run 'CSR|Playlist|Career'`
- grep : plus aucune itération CSR ne référence un littéral de 4 playlists en dur.

---

## Étape B2 — Leaderboard : service-record au lieu d'agrégation par-match

**Pourquoi** : 1 appel/joueur au lieu de milliers ; supprime les points d'attrition 2/3
(history + match-stats). Approche LeafApp `serviceRecord`.

**Périmètre fermé** :
- [ ] **Vérifier l'existant AVANT d'implémenter** (règle CLAUDE.md n°14) :
      `rg -n "service.?record|ServiceRecord" apps/go-api/internal/service apps/go-api/internal/platform/halo`
      — `compare_service`/`remote_stats_cache`/`privacy_provider` fetchent probablement déjà
      le service-record de joueurs arbitraires (feature Comparaison). Réutiliser, ne pas
      dupliquer.
- [ ] Remplacer `collectPlayerMatches`/`getMatch` (aggregator) par un fetch service-record
      par xuid (saison courante), mappé vers `world_player_season_stats`.
- [ ] 403 (historique privé) = joueur gardé avec CSR/rang (Phase A), stats vides + flag
      `stats_private` ; log `slog.InfoContext` (pas d'erreur), compteur expvar dédié.
- [ ] Erreur par-joueur = non-fatale (skip ce joueur, continue les autres).

**Gate** :
- `cd apps/go-api && go test ./internal/service/ -run 'World|ServiceRecord|Aggregat'`
- Mesure avant/après du taux d'enrichissement sur 1 playlist (loggé dans thought_log).

---

## Étape C1 — Frontend : tri + affichage placement précédent (polish)

**Périmètre fermé** :
- [ ] Page player/classement : trier les playlists `is_active` d'abord, puis classées
      retirées. Query keys dans `lib/query/keys.ts`, jamais inline.
- [ ] Placement saison précédente (façon LeafApp) : afficher le snapshot de la dernière
      saison clôturée en delta ↑/↓ (données déjà présentes : `player_csr_snapshots` +
      `csr_history_backfill`). Vérifier que le backfill N-1 tourne.
- [ ] Strings UI FR + EN (`i18n.ts`), labels stats via `useFieldLabel()`, zéro couleur hex
      (tokens sémantiques, skill `color-tokens`).

**Gate** :
- `make check-types && make test-web`

---

## Étape C2 — Saisons : persister + surfacer la liste (numéro + sous-saison)

**Pourquoi** : LeafApp affiche les saisons avec numéro + nom de sous-saison (« X-Infinite »).
Waypoint expose déjà cette liste et notre scraper la parse (`Seasons []{SeasonID,
DisplayName}`, scraper:247,262-265) mais la jette. Coût faible, valeur : sélecteur de
saison correct + placement « saison précédente » nommé.

**Périmètre fermé** :
- [ ] **Enquête d'abord (ne pas deviner)** : dumper les `DisplayName`/`SeasonID` réels
      scrapés et les croiser avec `csr_season_calendars` pour établir la correspondance
      (marketing « Season N » vs identifiant CSR vs « Operation »). Consigner la logique
      dans le thought_log + un commentaire de code. Objectif : comprendre « X-Infinite ».
- [ ] Persister la liste des saisons (table/colonnes metadata dédiées ; réutiliser
      `csr_season_calendars` si adéquat, sinon table `season_catalog`). INSERT-only si
      append-only.
- [ ] Exposer via un endpoint/handler + query key ; sélecteur de saison côté classement &
      page player alimenté par cette liste (nom affiché = `DisplayName`).
- [ ] Multi-titre : gate capability (saisons CSR), pas sur slug. H5 a son propre modèle
      saison (ne pas mélanger).

**Gate** :
- `cd apps/go-api && go test ./internal/platform/halo/ ./internal/platform/duckdb/ -run 'Season|Leaderboard'`
- `make check-types && make test-web` (sélecteur + i18n FR/EN).
- Enquête DisplayName collée dans le thought_log AVANT de coder la persistance.

---

## Découvertes (fixes hors périmètre repérés — À NE PAS traiter ici)

- **Doc inversée** (leaderboard_world_repo.go:57-58) : commentaire « Waypoint ne publie pas
  de xuid » contredit par scraper:117 qui le parse. Traité DANS B1 (même code).
- **Seasons Waypoint déjà scrapées mais inexploitées** : `parseLeaderboardPage` capture
  `Seasons []{SeasonID, DisplayName}` (scraper:247,262-265) — la liste numéro + nom de
  sous-saison (« X-Infinite ») que LeafApp affiche. Aujourd'hui non persistée/non surfacée.
  Ouvert comme étape C2 (validé utilisateur 2026-07-02).

## Protocole de reprise de session

- Avancement = cases cochées de ce fichier + entrées `.ai/thought_log.md`.
- Reprendre à la 1re étape non close (dans l'ordre B1 → A1 → A2 → B2 → A3 → C1).
- Avant de coder une étape : rouvrir les fichiers cibles (le code a pu bouger).

## Statut de clôture

Aucune case vide autorisée : `[x]` fait / `[~]` couvert ailleurs (réf) / `[!]` non traité
(justification). Entrée thought_log obligatoire par étape close.
