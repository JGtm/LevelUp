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

- [ ] 3.1 `cmd/snapshot-world-leaderboard` : flag `-restore-best` — pour chaque
      (saison, playlist) dont le lot servi est strictement plus pauvre qu'un lot historique
      (critères D1), ré-INSÉRER le meilleur lot avec `fetched_at` frais via
      `InsertWorldCSRSnapshot` (append-only, aucune suppression). Idempotent : skip si le
      lot servi est déjà le meilleur. Dry-run par défaut, `-execute` pour écrire.
- [ ] 3.2 Exécution locale (serveur ARRÊTÉ — writer unique dblease), puis relance serveur.
- [ ] 3.3 Test : `:memory:` — lot dégradé servi + lot sain historique → restore → la vue
      `_latest` sert le contenu sain ; re-run → no-op.

**Gate Lot 3** :
```
diag_q ... "SELECT count(*), sum(CASE WHEN COALESCE(l.xuid,'')<>'' THEN 1 ELSE 0 END), count(s.gamertag) \
  FROM world_csr_leaderboard_latest l LEFT JOIN world_player_season_stats_latest s \
  ON lower(s.gamertag)=lower(l.gamertag) AND s.season_id=l.season_id AND s.playlist_id=l.playlist_id \
  WHERE l.season_id='csrseason13-2' AND l.playlist_id='edfef3ac-9cbe-4fa2-b949-8f29deafd483'"
```
Attendu : 100 lignes, 100 xuid, ≥ 80 enrichies. Sonde HTTP authentifiée (cookie signé HMAC
avec `LEVELUP_SESSION_SECRET` de `.env.local`) : ≥ 80 entrées avec `match_count`.

## Lot 4 — UI honnête + contrat robuste (full-stack, moyen)

- [ ] 4.1 Backend : saison/playlist absents du GET page → réponse 200 vide (plus jamais le
      500 `leaderboard_error` pour paramètres manquants) ; test handler httptest.
- [ ] 4.2 Backend : `LeaderboardCatalog` expose les playlists PAR SAISON (couples réels
      mesurés en base) en PLUS des listes plates (compat) ; `make generate-types`.
- [ ] 4.3 Front `LeaderboardBlock` : appliquer D2 (seuil 25 % + bandeau i18n FR+EN dans le
      manifest commun) ; sélecteur playlist filtré par saison sélectionnée,
      `effectivePlaylist` retombe sur la 1re playlist DE la saison ; tests vitest.
- [ ] 4.4 Vérifier au passage la découverte `total_local` absent du JSON (tag ?) — corriger
      si trivial, sinon consigner en §Découvertes.

**Gate Lot 4** : `make check-types && make test-web` ; sonde authentifiée sans paramètres →
HTTP 200 corps vide structuré ; capture visuelle = MAIN AU USER (URL exacte fournie :
`http://localhost:5173/t/halo_infinite/players/JGtm/community`).

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

## Protocole de reprise

Relire ce plan + la dernière entrée `thought_log.md` ; l'avancement fait foi par les cases
ci-dessus ; reprendre au premier item non `[x]`/`[~]`/`[!]` ; ne jamais paralléliser deux lots.
