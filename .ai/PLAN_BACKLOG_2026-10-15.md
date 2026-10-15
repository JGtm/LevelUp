# Plan — backlog du 2026-10-15 (items 4, 5, 6 de `.ai/BACKLOG.md`)

Origine : demande du user du 2026-10-15 (« ok pour 4, 5 et 6 »). Items 1 et 2 (courbe de score
Lattice, carte Harvest) : en dernier, hors de ce plan (séquence de pré-release). Item 3 (restes
de la démo) : conversation dédiée, avec l'audit de sécurité.

Contrat d'exécution : skill `plan-execution` (fait foi). Branche `feat/backlog-2026-10-15`,
worktree `LevelUp-wt-backlog-1015`, partie de `origin/feat/v75` `a9a9ac78a`. Superviseur : session
principale. Exécutants : Sonnet. Fusion dans `feat/v75` : geste du user. Push `main` : jamais.

Ordre strict : E1 → E2 → E3 → E4. Une étape est close quand son gate est vert, chaque item
statué (`[x]` / `[~]` réf / `[!]` justifié), ce fichier à jour, une entrée au thought_log, un
commit préfixé par l'étape. Découvertes hors périmètre : section 3, non traitées.

Environnement : `GOCACHE=%LOCALAPPDATA%\go-build-backlog1015`,
`GOLANGCI_LINT_CACHE=%LOCALAPPDATA%\golangci-backlog1015`. Jamais `go build ./...`. Un seul
`go` à la fois. Clôture : `scripts/disk-hygiene.ps1 -Lot backlog1015 -Apply`.

## 1. Décisions (fermes)

- **D-1** Le garde `archlint/campaign_exclusion_guard_test.go` est ÉTENDU, pas doublé : critère
  élargi aux tables `killer_victim_pairs`, `match_kill_events(_latest)` et `medals_earned`,
  racines élargies à `internal/sync`, `internal/ops`, `cmd`.
- **D-2** Nouvelle catégorie de dispense « outil opérateur » réservée à `cmd/diag_*`,
  `cmd/audit_*` et aux backfills qui réécrivent le match lu lui-même (la Campagne n'y change
  aucun agrégat). Tout lecteur de `internal/sync` ou `internal/ops` qui produit une donnée
  servie à l'UI (dérivé, agrégat, snapshot) est CORRIGÉ, pas dispensé, sauf catégorie existante
  prouvée (mono-match, ensemble fourni par l'appelant, prédicat classé, sémantique).
- **D-3** Exclusion = helpers existants (`analysis.SQLExcludeCampaign*`, `excludeCampaign*` de
  `platform/duckdb/campaign_exclusion.go`). Aucun nouveau mécanisme, aucune comparaison de slug.
- **D-4** E1 est un diagnostic : si la cause est une donnée du poste (sync partielle locale),
  on clôt avec le constat ; si c'est un défaut de code de taille S, il est corrigé en E1 ;
  au-delà, STOP et retour au user.
- **D-5** Lecture prod pour E1 : requête en LECTURE seule sur le VPS, le superviseur prévient le
  user avant (pas d'écriture, pas de redémarrage).
- **D-6** Item 6 : `migration.SplitSQL` (limite documentée) et la CLI de migration d'une player
  DB désignée sont RETIRÉS du backlog sans traitement (aucun besoin, proposé le 15/10, validé).
- **D-7** `title_slug` de `GET /squads` : supprimé du contrat (OpenAPI + `make generate-types`
  + appelants web), pas de rétrocompatibilité (paramètre sans effet).

## 2. Étapes

### E1 — Matchs sans tableau des scores (item 5, diagnostic) — S

Pièces : `.ai/V7.5/PLAN_BACKLOG_2026-09-26.md` DA-5 ; témoins `000d5950`, `28c9b538`,
`4f77afc1`, `94a28b8b` (vue match `scoreboard_empty`).

- [x] E1.1 Sur une COPIE de `shared_matches_v2.duckdb` du poste : pour chaque témoin, présence
  dans `match_registry`, nombre de lignes de `match_participants` (vue `_latest` si append-only),
  date d'insertion ; résultat en tableau au journal.
- [x] E1.2 Recenser localement tous les matchs du registre sans participant (compte, période,
  titre, joueurs suivis concernés).
- [x] E1.3 Lire le chemin d'écriture des participants (sync + rattrapage) et nommer la cause
  (sync interrompue, rattrapage manquant, filtre) avec `fichier:ligne`.
- [x] E1.4 Même compte qu'E1.2 en prod (D-5). Fait par le superviseur le 2026-10-15 (journal).
- [x] E1.5 Conclusion selon D-4 : constat clos, ou correctif S avec test rouge puis vert, ou STOP.
- [x] E1.6 Mettre à jour l'entrée du backlog (retirée si close, réécrite sinon). La sous-partie
  « ERROR au boot de la démo sur `halo_5/warehouse/metadata.duckdb` » passe dans l'audit de
  sécurité/démo (`[~]`).

Gate : requêtes rejouables collées au journal ; si correctif, `go test` des paquets touchés.

### E2 — Lecteurs sans exclusion de la Campagne (item 4) — M

Pièces : `archlint/campaign_exclusion_guard_test.go` + `_dispenses_test.go` ;
`platform/duckdb/kill_events_source.go:71-85` (`QKillsBetweenPlayers`, `…Borne`) ;
`platform/duckdb/queries_citations.go:38` (`Q36aMedalTotals`, lu par `citations_repo.go:113` et
`medals_repo.go:92`) ; `api/wire/post_sync_deltas_snapshot.go:322` (`loadEarnedMedalIDs`) ;
liste des 56 lecteurs au journal B4 de `.ai/V7.5/PLAN_BACKLOG_2026-09-26.md` (l. 2351-2380).

- [ ] E2.1 Test de comportement rouge puis vert (tag `integration`, joué `halo_5` ET
  `halo_infinite`, comme `campaign_exclusion_behavior_test.go`) puis correction de
  `QKillsBetweenPlayers` et `QKillsBetweenPlayersBorne` (duel A/B sans frag de Campagne).
- [ ] E2.2 Idem pour `Q36aMedalTotals` (totaux de médailles, citations).
- [ ] E2.3 Idem pour `loadEarnedMedalIDs` (médaille de Campagne non « déjà obtenue »).
- [ ] E2.4 Étendre le garde selon D-1 ; mutation vérifiée (retirer une exclusion d'E2.1 → rouge).
- [ ] E2.5 Statuer les 17 lecteurs de `internal/sync` (corriger ou dispense D-2, avec raison
  et `fichier:ligne`).
- [ ] E2.6 Statuer les 10 lecteurs de `internal/ops`.
- [ ] E2.7 Statuer les 29 lecteurs de `cmd`.
- [ ] E2.8 Tout autre entrant révélé par l'extension (nouvelles tables) statué de même ; un
  entrant hors de toute catégorie = STOP, retour au superviseur.
- [ ] E2.9 Ratchet `no_slug_comparison_test.go` vert ; aucune comparaison de slug ajoutée.

Gate : `go test ./internal/archlint/ ./internal/platform/duckdb/... ./internal/api/wire/` + les
paquets corrigés, puis `go test -tags=integration -p 1` sur les mêmes paquets.

### E3 — Petites dettes Go (item 6) — S

- [ ] E3.1 `campaign_repo.go` `LoadAxisSamples` (l. ~278-286) et la requête sur `match_registry`
  (l. ~364-373) : temps canonique (fragment partagé, règle n°8) pour filtre et tri ; test.
- [ ] E3.2 `engagement_score_repo_queries.go` `ListRecentPvPMatchIDs` (l. ~309-312) : temps
  canonique ; erreur de `Scan` (l. ~85 du bloc, `if err == nil`) et `continue` sur erreur sans
  log (l. ~34-38) → remontée ou `slog.WarnContext` + compteur ; test.
- [ ] E3.3 `GET /squads` : retrait de `title_slug` (D-7), `make generate-types`,
  `make check-types`, appelants web mis à jour.
- [ ] E3.4 D-6 appliqué dans `.ai/BACKLOG.md`.

Gate : `go test` des paquets touchés, `go test ./internal/api/...` (contrat OpenAPI golden),
`make check-types`, `make test-web` si un fichier web change.

### E4 — Clôture

- [ ] E4.1 `make go-api-lint` (ratchet), `make check-types`.
- [ ] E4.2 Revue adversariale du diff (skill `adversarial-review`, contexte frais).
- [ ] E4.3 `.ai/BACKLOG.md` : items 4, 5, 6 retirés ou réécrits ; `.ai/REGISTRE_REPORTS.md`
  alimenté ; thought_log.
- [ ] E4.4 Push de la branche, CI verte (tout rouge se répare).
- [ ] E4.5 Fusion dans `feat/v75` : demandée au user.

## 3. Découvertes

- E1 : `partial_reasons` ne distingue pas « liste vide » de « chargeur en erreur » (`service/match_view_data_loaders.go:262-269`, `:670-674`) : une panne de lecteur s'affiche comme un match sans tableau des scores. Évolution de contrat, à décider avec le user.
- E1 : `match_registry.participants_loaded` et `player_count` ne sont plus posés par le chemin courant (FALSE / 0 sur des matchs complets) : colonnes mortes ou à fiabiliser, à trancher.

## 4. Journal

- 2026-10-15 : plan écrit, worktree créé sur `a9a9ac78a`.
- 2026-10-15 (E1, exécutant Sonnet) : E1.1, E1.2, E1.3, E1.5, E1.6 faits ; E1.4 laissé au
  superviseur. Méthode : copie de `shared_matches_v2.duckdb` de chaque titre (principal, 16:22, pas de
  `.wal`) dans le scratchpad, interrogée par `cmd/diag_q` (CLI testé, lecture seule ; pas de CLI
  `duckdb` sur le poste). `match_participants` n'est pas append-only (pas de vue `_latest`).
  - **E1.1 — tableau des témoins (Halo Infinite ; aucun n'est dans Halo 5)**

    | match | registre | début UTC | playlist | 1re sync (par, quand) | lignes `match_participants` | lignes lues par Q12 | `highlight_events` |
    |---|---|---|---|---|---|---|---|
    | `000d5950` | oui | 2026-03-08 20:43 | Quick Play | JGtm, 2026-03-08 21:55 | 8 | 8 | 230 |
    | `4f77afc1` | oui | 2026-07-24 20:20 | Big Team Battle | JGtm, 2026-07-24 20:41 | 36 | 28 | 844 |
    | `94a28b8b` | oui | 2026-08-27 20:51 | Quick Play | Chocoboflor, 2026-08-27 21:02 | 10 | 9 | 193 |
    | `28c9b538` | oui | 2026-09-01 19:53 | Quick Play | JGtm, 2026-09-01 20:52 | 8 | 8 | 240 |

    « Q12 » = `Q12MatchScoreboard` (`platform/duckdb/queries_match.go:54-122`) : le filtre `NOT (kills,
    deaths, assists, personal_score tous à 0 ET au moins un non NULL)` écarte les lignes à zéro
    (4f77afc1 : 8 écartées, 94a28b8b : 1). Les quatre ont donc un tableau des scores NON vide sur
    les données de ce poste, ainsi que des événements. Même résultat dans le cut Parquet servi
    (`data/titles/halo_infinite/warehouse/snapshots/v…180/shared/`, créé le 2026-10-14) : 8, 36, 10, 8
    participants et 230, 844, 193, 240 événements. Requête :
    `SELECT substr(match_id,1,8), count(*), sum(CASE WHEN NOT (COALESCE(kills,0)=0 AND COALESCE(deaths,0)=0
    AND COALESCE(assists,0)=0 AND COALESCE(personal_score,0)=0 AND (kills IS NOT NULL OR deaths IS NOT
    NULL OR assists IS NOT NULL OR personal_score IS NOT NULL)) THEN 1 ELSE 0 END) FROM match_participants
    WHERE match_id LIKE '000d5950%' OR … GROUP BY 1`.
  - **E1.2 — recensement local** : Halo Infinite 9 230 matchs au registre, **0 sans participant** ;
    Halo 5 3 032 matchs, **0 sans participant** ; 0 participant orphelin (sans registre) côté Halo
    Infinite. Aucun joueur suivi concerné, aucune période. Cas voisin, distinct : matchs à participants
    TOUS à zéro (Q12 les écarte, donc `scoreboard_empty` à juste titre) : Halo Infinite 83 (2021-11 à
    2026-09-28 ; 43 Ranked Arena, 12 Ranked Slayer, 6 Ranked Doubles, 5 BTB…, matchs de 16 à 309 s,
    abandons/quitté tôt), Halo 5 115 (2015 à 2021, rosters de 1 à 7). Aucun témoin n'en fait partie.
    Autre constat : `participants_loaded` est à FALSE et `player_count` à 0/2 sur les témoins alors que
    les participants existent : ces deux colonnes ne sont jamais posées par le chemin d'écriture courant
    (`persist/shared_persister.go:146-200` n'écrit pas `participants_loaded`) — ne pas s'y fier. Le bit
    participants de `backfill_completed` (512) est faux sur 157 matchs HI (et les 3 032 de H5, qui n'ont pas
    ce bit) tout en ayant leurs participants : même remarque.
  - **E1.3 — chemin d'écriture** : `SharedPersister.Persist` (`persist/shared_persister.go:66-141`)
    insère registre, participants, médailles, événements… dans UNE transaction, `ROLLBACK` si l'une
    échoue : un match au registre sans participants ne peut pas naître d'une sync interrompue.
    `WithParticipants` vaut `true` par défaut (`domain/sync.go:67`) et le moteur extrait les
    participants à chaque fetch (`sync/engine_fetch.go:105-107`) ; aucun site ne le passe à `false`.
    Rattrapage : `FindMatchesMissingParticipantBits` (`sync/backfill.go:82-115`) ; H5 : rosters
    incomplets reconstruits par `cmd/h5-roster-topup` (sujet distinct, rosters partiels, pas vides).
    Côté lecture, `goLoad` avale toute erreur d'un chargeur de la vue match en `slog.WarnContext`
    (`service/match_view_data_loaders.go:262-269`) et `detectPartialMatchData` pose `scoreboard_empty`
    quand la liste est vide, sans distinguer « vide » et « chargeur en erreur » (`:670-674`).
    **Cause du constat du 27/09 : non établie, non reproductible.** Ce qui est prouvé : la donnée est
    complète sur ce poste aujourd'hui et l'écriture est atomique ; la vue du 27/09 rapportait AUSSI
    `events_empty` alors que les 193 à 844 événements existent : deux sources indépendantes vides en
    même temps pointent vers un lecteur partagé indisponible ou une source de données différente au
    moment de la sonde (serveur de worktree, lecteur sous charge), pas vers une donnée manquante.
    Les journaux serveur du 27/09 n'existent plus (`data/logs/` ne garde que `backfill_killsource`).
  - **E1.5 — D-4** : cause = environnement de la sonde, pas un défaut de données ni de code prouvé :
    constat clos, aucun code modifié, aucun test. Une amélioration (distinguer chargeur en erreur de
    liste vide dans `partial_reasons`) serait une évolution de contrat, pas un correctif S : section 3.
  - **E1.6** : entrée du backlog réécrite (constat local clos, reste le compte prod, retirée par le
    superviseur si E1.4 donne 0) ; la sous-partie « ERROR au boot de la démo » (`halo_5/metadata.duckdb`)
    est passée dans l'entrée `[demo]` du backlog, pour l'audit sécurité/démo `[~]`.
  - **E1.4 — requête prod à jouer par le superviseur** (lecture seule, sur une COPIE de
    `/opt/levelup/data/titles/{halo_infinite,halo_5}/warehouse/shared_matches_v2.duckdb`, ou
    `duckdb -readonly` si aucun process ne la tient RW ; une fois par titre). Testée ici sur les copies
    locales (HI : 9230 / 0 / 83 / NULL / NULL / 4 / 0 ; H5 : 3032 / 0 / 115 / NULL / NULL / 0 / 0) :
    ```sql
    WITH n AS (
      SELECT r.match_id,
             COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC') AS t,
             r.first_sync_by,
             (SELECT COUNT(*) FROM match_participants p WHERE p.match_id = r.match_id) AS n_part,
             (SELECT COUNT(*) FROM match_participants p WHERE p.match_id = r.match_id
                AND NOT (COALESCE(p.kills,0)=0 AND COALESCE(p.deaths,0)=0 AND COALESCE(p.assists,0)=0
                         AND COALESCE(p.personal_score,0)=0
                         AND (p.kills IS NOT NULL OR p.deaths IS NOT NULL OR p.assists IS NOT NULL
                              OR p.personal_score IS NOT NULL))) AS n_q12
      FROM match_registry r)
    SELECT COUNT(*) AS registre,
           COUNT(*) FILTER (WHERE n_part = 0) AS sans_participant,
           COUNT(*) FILTER (WHERE n_part > 0 AND n_q12 = 0) AS participants_tous_a_zero,
           MIN(t) FILTER (WHERE n_part = 0) AS depuis, MAX(t) FILTER (WHERE n_part = 0) AS jusqua,
           COUNT(*) FILTER (WHERE match_id LIKE '000d5950%' OR match_id LIKE '28c9b538%'
                              OR match_id LIKE '4f77afc1%' OR match_id LIKE '94a28b8b%') AS temoins_au_registre,
           COUNT(*) FILTER (WHERE n_part = 0 AND (match_id LIKE '000d5950%' OR match_id LIKE '28c9b538%'
                              OR match_id LIKE '4f77afc1%' OR match_id LIKE '94a28b8b%')) AS temoins_sans_participant
    FROM n;
    ```
    Lecture : `sans_participant` = le compte d'E1.2 (attendu 0 ; si > 0, `depuis`/`jusqua` donnent la
    période, et `first_sync_by` se ventile par `GROUP BY` sur `n_part = 0`). Les 4 témoins ne sont
    probablement pas au registre prod (matchs de comptes du poste) : 0 y est normal.
- 2026-10-15 (E1.4, superviseur) : copies des deux `shared_matches_v2.duckdb` de prod
  (`scp` depuis `/opt/levelup/data/titles/*/warehouse/`, pas de `.wal`, checkpoint de 04:00), requête
  du journal jouée par `cmd/diag_q` : Halo Infinite 2014 / **0** / 1 / NULL / NULL / 4 / 0 ; Halo 5
  3032 / **0** / 115 / NULL / NULL / 0 / 0. Aucun match sans participant en prod ; les quatre témoins
  y sont, avec participants. Entrée du backlog retirée. **E1 close.**
