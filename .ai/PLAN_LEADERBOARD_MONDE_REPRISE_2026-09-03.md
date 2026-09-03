# PLAN — Classement mondial : reprise du scrape, garde-fous qualité, restauration

> Date : 2026-09-03 · Branche cible : `fix/leaderboard-monde-reprise-scrape` (depuis `main`,
> worktree dédié — le worktree principal est partagé). Merge final vers `main` = DEPLOY PROD
> AUTO : prévenir l'utilisateur avant le push.
> Exécution sous le contrat du skill `plan-execution` (ordre strict, gates, statuts
> `[x]`/`[~]`/`[!]`, zéro fix hors périmètre — les découvertes vont en §Découvertes).

## Contexte prouvé (diagnostic 2026-09-03, sondes + SQL — ne pas re-vérifier sauf gate)

- La vue `world_csr_leaderboard_latest` = « dernier lot gagne » (`max(fetched_at)` par
  titre+saison+playlist). Le cycle cron du 2026-07-07 a persisté des lots dégradés
  (Arène : 86 lignes / 0 xuid / 4,7 % enrichies) qui masquent depuis les lots sains du
  2026-07-03 (100 lignes / 100 % xuid / 88 % enrichies).
- Le cron échoue depuis le 2026-07-13 (267 cycles) : la graine de découverte = dernière
  saison persistée (`csrseason13-2`), or Halo Waypoint ne sert PLUS cette saison (404
  vérifié en live, saison absente du menu déroulant actuel). Le log « auto-résolutif »
  est FAUX : la graine est un point fixe qui ne peut jamais guérir seul.
- Waypoint est VIVANT : `https://www.halowaypoint.com/halo-infinite/leaderboards` rend la
  saison courante `csrseason13-3`, `__NEXT_DATA__` intact, 100 entrées/page avec
  `player.xuid` + `player.gamertag`, 6 playlists (mêmes IDs que nos snapshots).
- La résolution gamertag→xuid multi-comptes avec respect des débits EXISTE déjà
  (`internal/worldenrich` : chaîne PeopleHub→Profil Xbox, round-robin sur 429,
  `ratebudget.ForXUID`, cache persistant). RÉUTILISER, ne rien réimplémenter.
- `world_player_season_stats` n'a PAS de colonne xuid (jointure par gamertag) — le lot
  xuid-de-bout-en-bout est DIFFÉRÉ (voir §Reports).

## Décisions tranchées avant exécution (défauts retenus, veto user possible)

- **D1 (seuils qualité au persist)** : un lot candidat pour (saison, playlist) est REFUSÉ
  si un lot précédent existe ET ( lignes_candidat < 50 % des lignes du lot servi
  OU (couverture xuid du lot servi ≥ 90 % ET couverture xuid candidat = 0 %) ).
  Plancher absolu `minEntries = 25` conservé. Refus = skip + `slog.WarnContext` +
  compteur expvar (jamais d'erreur de cycle).
- **D2 (UI colonnes enrichies)** : colonnes détaillées affichées si ≥ 25 % des lignes ont
  `match_count != null` ; sinon masquées + bandeau i18n « Stats détaillées indisponibles
  pour ce relevé » (FR+EN). Le bandeau apparaît aussi entre 25 % et 80 % (« partielles »).
- **D3 (restauration 13-2)** : OUI — ré-INSERT append-only du meilleur lot historique
  (jamais de DELETE/UPDATE, règle ART).
- **D4 (déploiement)** : lots 1-3 mergés ensemble vers `main` après gate-push ; Lot 4
  peut suivre dans le même train ou un second commit. Restauration (Lot 3) à rejouer sur
  le VPS après deploy (lecture seule sauf la fenêtre d'INSERT ; prévenir avant).

## Lot 1 — Réparer la découverte de saison + logs honnêtes (backend, moyen)

> Mécanique VÉRIFIÉE en live (2026-09-03, curl) : `GET /halo-infinite/leaderboards` rend
> `307` avec `Location: /halo-infinite/leaderboards/csrseason13-3` — le header SEUL donne
> la saison active, aucun parsing HTML nécessaire. `/leaderboards/{saison}` redirige à son
> tour vers une playlist par défaut dont la page peut rendre 500 (vécu : 6233381c), alors
> que les autres pages playlist rendent 200 avec `__NEXT_DATA__` complet (xuid inclus) —
> la tolérance par playlist existante (skip + log) couvre.

- [x] 1.1 `internal/platform/halo/leaderboard_scraper.go` : nouvelle découverte de saison
      par redirection — GET base URL avec suivi de redirection DÉSACTIVÉ
      (`CheckRedirect` → `http.ErrUseLastResponse`), lire `Location`, extraire l'ID
      `csrseason...` (format validé, sinon erreur). Passer par `netguard.Check` comme les
      autres fetchs. Corriger le commentaire « les saisons restent accessibles
      indéfiniment » (réfuté : 13-2 retirée de Waypoint).
- [x] 1.2 `internal/scheduler/world_leaderboard_cron.go` : `discoverActiveSeason` essaie le
      repli par redirection après épuisement des candidates page-graine ; reformuler le
      message « auto-résolutif » (doc inversée) en décrivant le repli réel.
- [x] 1.3 Escalade : compteur expvar « cycles consécutifs sans snapshot » ; au-delà de 3,
      le WARN devient `slog.ErrorContext` (avec le compteur en champ).
- [x] 1.4 `enrich()` : quand `c.enricher == nil`, `slog.WarnContext` PAR CYCLE
      (« enrichissement inactif — cron scrape-only ») au lieu du retour silencieux.
- [x] 1.5 Tests : httptest — page-graine 404 + base-URL 200 (fixture) → saison découverte ;
      compteur d'escalade ; warn enricher nil (pas de fixture réseau réelle).

**Gate Lot 1** :
```
cd apps/go-api && go test ./internal/platform/halo/... ./internal/scheduler/...
```
puis redémarrer le serveur air local, déclencher un cycle (boot le fait), et vérifier :
```
diag_q data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb \
  "SELECT count(*), sum(CASE WHEN COALESCE(xuid,'')<>'' THEN 1 ELSE 0 END) \
   FROM world_csr_leaderboard_snapshots WHERE season_id='csrseason13-3'"
```
Attendu : count > 0 ET xuid non vides > 0. (diag_q : build CGO msys64, DB jamais ouverte
RW pendant que le serveur tourne — read_only.)

## Lot 2 — Garde-fous qualité au persist (backend, moyen)

- [x] 2.1 helper lecture des stats du lot servi pour (titre, saison, playlist) →
      `WorldCSRServedBatchStats` + type `WorldCSRBatchStats`, extraits dans
      `leaderboard_world_batch_stats.go` (65 L — le repo à 733 L reste intact). Test
      dans `leaderboard_world_repo_test.go` (harnais existant, build tag integration).
- [x] 2.2 `scrapeAll` applique D1 par playlist AVANT insert (hors lease writer, reader RO
      court par playlist, fail-open sur lecture illisible) ; décision isolée en fonction
      pure `degradedBatchReason` dans `world_leaderboard_quality.go` (126 L) ; refus =
      skip + WARN chiffré + expvar `world_leaderboard_batch_refused_total`.
- [x] 2.3 Tests scheduler : effondrement volume refusé ; effondrement xuid refusé ;
      croissance normale / première capture acceptés ; refus partiel (1 playlist saine
      persistée quand l'autre est refusée) ; bornes de la règle en table-driven.

**Gate Lot 2** : `cd apps/go-api && go test ./internal/scheduler/... ./internal/platform/duckdb/...`
(recette CGO msys64) **+ `go test -tags=integration ./internal/platform/duckdb/ -run WorldCSR`**
— le test du helper 2.1 est sous build tag integration, la commande sans tag ne fait que le
compiler (découverte Lot 2). Les tests persist anti-ART complets restent au gate de livraison.

## Lot 3 — Restauration des lots sains (op données, rapide)

- [x] 3.1 `cmd/snapshot-world-leaderboard` : flag `-restore-best` (dry-run par défaut,
      `-execute` pour écrire), sélection du meilleur lot par (with_xuid, rows, fetched_at),
      verdict = `duckdb.DegradedBatchReason(meilleur, servi)` — la règle D1 réutilisée à
      l'envers, une seule définition (déplacée du scheduler vers
      `leaderboard_world_batch_stats.go`, deux callers). Ré-INSERT append-only,
      idempotent par construction. EN PLUS (découverte Lot 2 retenue) : le chemin de
      scrape du CLI passe par le même garde-fou que le cron (`cliBatchRefusalReason`).
- [x] 3.2 Exécution locale FAITE (serveur arrêté puis relancé) : dry-run = exactement les
      3 couples dégradés du 07/07 identifiés, 37 sains intacts ; `-execute` = 3 restaurés,
      0 échec ; re-run = « 0 à restaurer, 40 déjà au meilleur » (idempotence prouvée).
- [x] 3.3 Tests : `TestWorldCSRBestBatch_RestoreCycle` (duckdb, integration — sélection,
      verdict, restauration, append-only, idempotence) + test d'orchestration CLI qui
      verrouille que le dry-run n'écrit rien.
- [!] 3.4 (constat de gate) Enrichissement de la population restaurée : la mesure réelle
      donne (13-2, Arène) = 200 lignes / 200 xuid / 87 enrichies sur le LOT, mais le
      top-100 AFFICHÉ (limit=100) n'en joint que 34 — les stats existantes datent des
      populations scrapées début juillet, pas du top-100 du lot 11:17. Le correctif est
      OPÉRATIONNEL, pas du code : `cmd/backfill-world-player-stats -season csrseason13-2`
      (+ 13-3 dans la même fenêtre), qui cible par construction la population SERVIE
      (WorldSeasonPlayers lit la vue _latest) avec les xuid pré-seedés des snapshots
      restaurés. Fenêtre serveur arrêté ~15-40 min — DÉCISION USER sur le créneau
      (maintenant / soir / au merge). Statué [!] en attendant ce créneau.

**Gate Lot 3 (mesuré)** : SQL = 200 lignes / 200 xuid / 87 enrichies (critère ≥ 80 sur le
lot : PASSÉ ; l'attendu initial « 100 lignes » supposait le lot 11:25 — le meilleur lot
réel est le 11:17 à 200 lignes, plus riche en archive). Sonde HTTP top-100 : 100 entrées /
100 xuid / 34 match_count → critère « ≥ 80 affichées enrichies » REPORTÉ à l'item 3.4.

## Lot 4 — UI honnête + contrat robuste (full-stack, moyen)

- [x] 4.1 Backend : saison/playlist absents (blancs inclus) sur csr-world → 200 vide +
      WARN, le repo n'est même pas appelé (prouvé par mock strict) ; `Entries` jamais nil
      sur TOUS les retours anticipés (ratchet NoNilSlices — couvrait aussi un trou
      pré-existant du chemin « titre sans capability ») ; nouveau
      `handlers/leaderboard_test.go` httptest de bout en bout.
- [x] 4.2 Backend : `LeaderboardCatalogRef.PlaylistIDs` (`playlist_ids,omitempty`, saisons
      seulement) — couples réels via `SELECT DISTINCT` isolé dans
      `leaderboard_world_catalog_seasons.go` (84 L ; le repo DESCEND à 725 L, la
      décoration Enriched déplacée avec) ; couples en best-effort AVEC WARN (front
      retombe sur la liste plate), Enriched reste une erreur dure ; openapi + generated.ts
      régénérés (`openapi-gen` + `make generate-types`, checks « à jour » verts).
- [x] 4.3 Front : logique extraite dans `LeaderboardBlock.logic.ts` (seuils exportés
      `ENRICHED_COLUMNS_MIN_RATIO=0.25` / `ENRICHED_FULL_RATIO=0.8`, bornes inclusives) +
      sous-composants `LeaderboardNotes`/`LeaderboardSelector` (le .tsx redescend à
      539 L) ; sélecteurs couplés avec double dégradation (champ absent, filtrage vide) ;
      bandeaux ICU chiffrés `{enriched}/{total}` FR+EN (`common.leaderboard.
      enrichment_unavailable` / `enrichment_partial`) ; bandeau « indisponibles » masqué
      sur saison archivée (note existante équivalente) ; 5 cas composant + 12 cas bornes.
- [x] 4.4 Élucidé, AUCUN bug : le champ Go `TotalLocal` porte `json:"total"` — le nom de
      fil est `total`, présent et requis dans openapi/generated ; « total_local » n'a
      jamais existé sur le fil (ma sonde grep-ait le mauvais nom). Figé par test HTTP
      (`TestLeaderboardPage_TotalFieldNameOnTheWire`).

**Gate Lot 4 (rejoué orchestrateur)** : `tsc -b --force` 0 erreur ; vitest leaderboard
35/35 ; go service+handlers+duckdb (integration `WorldCSR|Catalog`) verts ; build vert.
Exécuteur : `make test-web` complet 3547 pass / 14 skips pré-existants hors périmètre ;
eslint périmètre 0 warning ; lint:fields 0. Sonde HTTP live sans paramètres : couverte par
le httptest de bout en bout (serveur local arrêté pendant la fenêtre backfill) ; capture
visuelle = MAIN AU USER (`http://localhost:5173/t/halo_infinite/players/JGtm/community`).

## Livraison

- [ ] Entrée `thought_log.md` par lot clos + entrée finale.
- [ ] `make gate-push` avant merge ; tests intégration persist
      (`go test -tags=integration ./...`) obligatoires (écritures sync/persist touchées).
- [ ] Skill `delivery-checklist` avant le merge ; skill `adversarial-review` sur le diff
      (lot à risque : persist).
- [ ] Prévenir AVANT push `main` (deploy prod auto). Après deploy : restauration Lot 3 sur
      le VPS (prévenir, fenêtre d'écriture courte), puis vérifier le cycle cron prod du
      lendemain 04:00 (log « cycle terminé » saison 13-3).

## Reports (registre `.ai/V7.5/REGISTRE_REPORTS.md` à mettre à jour à la clôture)

- **xuid de bout en bout** (colonne xuid sur `world_player_season_stats` via recette
  ADR 0026 — step au nom neuf + vue —, writers, jointure par xuid d'abord). Condition de
  reprise : taux de jointure < 70 % sur la saison active malgré cron vert 7 jours.
- **C3 backfill saisons passées** (inchangé, op user) — noter que Waypoint sert encore
  12-1/11-1/… mais PLUS 13-2 : le backfill 13-2 passe par la restauration Lot 3 uniquement.

## Découvertes (ne pas traiter dans ce chantier)

- `apps/web/src/lib/capabilities/capabilities.ts` : le commentaire « halo_infinite déclare
  TOUTES les capabilities » est faux (manquent `native_kill_mechanics`, `weapon_accuracy`,
  `spartan_customizer` au bootstrap).
- Waypoint a renommé deux playlists (28bfa5f4 → « RANKED 1V1 SHOWDOWN », c94cb508 →
  « RANKED LEGACY ») — la cascade de noms couvre, mais `rankedplaylists` statique vieillit.
- (Lot 1, exécuteur) `seedSeasonID = "csrseason13-2"` est une constante-graine morte (saison
  retirée du site) : la découverte par redirection pourrait devenir le chemin PREMIER au
  lieu du repli (économie de ~4-9 requêtes 404 par cycle dégradé ; s'auto-résorbe dès
  qu'un snapshot frais existe, donc non traité).
- (Lot 1, exécuteur) le cron ne garde pas `season != ""` : une implémentation du port
  rendant `("", nil)` ferait persister `season_id=''` (le vrai scraper valide — contrat
  par convention seulement).
- (Lot 1, exécuteur) issues lint pré-existantes sur les packages touchés : `goconst`
  `challenges_details.go:429`, `gocyclo` 17 sur `FetchCSRLeaderboard` et `syncPlayer`.

- (Lot 2, exécuteur) le CLI `cmd/snapshot-world-leaderboard` écrit via le même
  `InsertWorldCSRSnapshot` SANS passer par le garde-fou D1 — un run CLI dégradé peut
  toujours masquer un lot sain. TRAITÉ AU LOT 3 (le CLI est dans son périmètre).
- (Lot 2, exécuteur) le garde-fou protège ce que la vue SERT (`max(fetched_at)`), pas la
  table : un lot accepté au `fetched_at` antérieur au lot servi serait inséré sans jamais
  être servi. Sans effet aujourd'hui (le scraper pose `time.Now().UTC()`) — hypothèse à
  ne pas casser.
- (Lot 3, exécuteur) asymétrie de signatures dans `leaderboard_world_batch_stats.go`
  (`WorldCSRServedBatchStats(…, titleSlug, seasonID, playlistID)` vs
  `WorldCSRBestBatch(…, key)`) — harmonisable sur `WorldCSRBatchKey`, non fait (le Lot 2
  est commité).
- (Lot 3, exécuteur) `TestDegradedBatchReason` vit dans `scheduler` alors que la fonction
  est dans `duckdb` — symétrie source/test à rétablir un jour.
- (Lot 3, exécuteur) `-restore-best` ignore `-dry-run` (c'est `-execute` qui commande dans
  ce mode) : deux drapeaux de simulation coexistent dans l'aide, libellés explicites mais
  confusion possible pour un opérateur pressé.
- (Lot 4, exécuteur) le ratchet `TestDTOs_NoNilSlicesOnEmptyInput` n'exerce que le chemin
  halo_infinite : le retour « titre sans capability » sérialisait `entries: null` sans
  être attrapé (corrigé par construction au 4.1, le ratchet reste aveugle à ce chemin).
- (Lot 4, exécuteur) `LeaderboardResponse.total` n'est lu nulle part côté web — champ de
  contrat sans consommateur.
- (Lot 4, exécuteur) `LeaderboardCatalogRef` sert saisons ET playlists mais porte deux
  champs saison-seulement (`enriched`, `playlist_ids`) — un type dédié clarifierait.
- (op 3.4) le checkpoint du backfill (`data/world_backfill_checkpoint.json`, juillet)
  marque csrseason13-2 « déjà complète » et la SKIPPE — la reprise doit passer
  `-force -skip-existing` (ignorer le checkpoint, ne fetcher que les joueurs manquants).
- (revue) `internal/archlint` n'était dans aucun gate de lot — le littéral d'URL du Lot 1
  n'a été attrapé que par la suite complète. Les prochains gates backend qui touchent
  logs/chemins devraient inclure `./internal/archlint/...` (10 s).
- (revue, mineurs M1-M5 non traités) : garantie Entries-non-nil du chemin nominal portée
  par les repos, pas le service · `written_at` frais du restore rend la saison « fraîche »
  pour le cron ~20 h · `openSharedRW` du CLI hors provider/dblease (pré-existant, échec
  sûr par verrou) · validation Location limitée au préfixe `csrseason` · coexistence
  `-dry-run`/`-execute` dans l'aide du CLI.

## Journal d'exécution

- **2026-09-03 — Lot 1 CLOS.** Exécuteur Opus (2 passes : implémentation puis extraction
  `world_leaderboard_discovery.go`, fichiers à 451/226/454 L — seuil 500 respecté).
  Revue orchestrateur sur diff complet ; gate test/vet/build rejoué par l'orchestrateur :
  vert (`-count=1`, 0 skip). Vérification sur DONNÉES RÉELLES : binaire du worktree lancé
  contre le checkout principal (air arrêté puis restauré) — pages-graines 13-2 en 404,
  saison `csrseason13-3` obtenue PAR LE REPLI à 11:29:09, cycle terminé 11:29:28 :
  516 lignes insérées, 516/516 avec xuid, playlist Legacy (2 entrées) rejetée par le
  plancher. Observation : ce cycle n'a couvert que les 4 playlists statiques (la
  découverte de playlists actives passe par la page-graine, morte à ce moment) — le
  cycle suivant se sert de la graine 13-3 fraîchement persistée et retrouvera les
  playlists complètes ; aucun correctif requis.
- **2026-09-03 — Lot 2 CLOS.** Même exécuteur (2 passes : implémentation puis extraction
  `leaderboard_world_batch_stats.go`, le repo revenant byte-identique à HEAD). Garde-fou
  D1 : décision pure `degradedBatchReason` (cause en clair réutilisée dans le WARN),
  mesure candidate en mémoire ISO-définition avec la requête SQL du lot servi, reader RO
  court par playlist (discipline `snapshotIsFresh`), fail-open documenté. 6 tests dont
  refus partiel sur vraie shared DB migrée, candidats à `fetched_at` postérieur (sinon la
  vue servirait l'ancien lot et les tests d'acceptation ne prouveraient rien). Gate
  rejoué par l'orchestrateur : scheduler + halo + duckdb (integration -run WorldCSR) +
  vet + build, tout vert.
- **2026-09-03 — Lot 3 CLOS (3.4 statué [!], créneau user).** `-restore-best` livré (règle
  D1 centralisée `duckdb.DegradedBatchReason`, une définition / deux callers ; scrape CLI
  protégé aussi). Exécution réelle : dry-run = exactement les 3 couples du 07/07 ;
  execute = 3 restaurés 0 échec ; re-run idempotent (0 à restaurer / 40 au meilleur).
  Après restauration : 13-2 est à 100 % xuid sur TOUTES les playlists (Assassin 115
  enrichies, Duo 101, Arène 87 sur le lot). Constat de gate : le top-100 AFFICHÉ de
  l'Arène ne joint que 34 stats → enrichissement de la population servie à compléter par
  le backfill opérationnel (item 3.4, fenêtre à choisir par le user). L'ordre de mérite
  (xuid avant volume, fraîcheur en dernier) est assumé : l'archive prime, l'enrichissement
  se complète par l'outil dédié.
- **2026-09-03 — Lot 4 CLOS.** Exécuteur frais (103 outils, tous gates verts). 500→200
  vide prouvé par mock strict + httptest bout-en-bout ; catalogue par saison
  (`playlist_ids`) avec openapi/generated régénérés et checks fraîcheur verts ; front :
  logique extraite testable (`LeaderboardBlock.logic.ts`), sélecteurs couplés à double
  dégradation, bandeaux ICU chiffrés FR+EN, fichier composant ramené SOUS son niveau
  d'avant-lot (539 L) ; `total_local` élucidé = non-bug (nom de fil `total`), figé par
  test. Gates rejoués orchestrateur : tsc --force, vitest 35/35, go
  service/handlers/duckdb + integration, build. Item 3.4 (backfill) : 13-2 skippée par le
  checkpoint de juillet (« déjà complète ») → reprise `-force -skip-existing` ; 13-3 en
  cours dans la fenêtre serveur-arrêté accordée par le user.
- **2026-09-03 — Revue adversariale (contexte frais) + correctifs.** Verdict initial
  no-merge : 0 bloquant, 4 SÉRIEUX, 5 mineurs, 13 invariants conformes. Correctifs tous
  livrés et gates verts : S1 escalade ERROR des refus consécutifs par playlist (>3, reset
  sur tout lot accepté, fail-open compris — un lot écrit clôt la série) ; S2 règle D1
  plafonnée par la profondeur demandée (`candidateDepthLimit`, 0 = sans plafond — fin du
  gel cron-200 face aux archives profondes ; décision : sur saison active, le servi est
  le frais, l'archive profonde est un filet) ; S3 `-restore-best` exige `-season` nommée
  (refus de `all`, deux FATAL explicites, testé) ; S4 tri effectif dérivé au rendu
  (`resolveSort` pure) — retombe sur le rang quand la colonne triée est masquée, choix
  utilisateur préservé, aria-sort vérifié. + littéral `www.halowaypoint.com` retiré du
  message d'escalade (seul échec de la suite d'intégration complète — garde-rail
  archlint). Gates rejoués : build, 5 packages Go dont archlint, integration WorldCSR,
  vet ./..., web complet 3554 tests.

## Protocole de reprise

Relire ce plan + la dernière entrée `thought_log.md` ; l'avancement fait foi par les cases
ci-dessus ; reprendre au premier item non `[x]`/`[~]`/`[!]` ; ne jamais paralléliser deux lots.
