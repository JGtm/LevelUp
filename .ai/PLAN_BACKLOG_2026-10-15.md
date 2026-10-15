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

- [x] E2.1 Test de comportement rouge puis vert (tag `integration`, joué `halo_5` ET
  `halo_infinite`, comme `campaign_exclusion_behavior_test.go`) puis correction de
  `QKillsBetweenPlayers` et `QKillsBetweenPlayersBorne` (duel A/B sans frag de Campagne).
- [x] E2.2 Idem pour `Q36aMedalTotals` (totaux de médailles, citations).
- [x] E2.3 Idem pour `loadEarnedMedalIDs` (médaille de Campagne non « déjà obtenue »).
- [x] E2.4 Étendre le garde selon D-1 ; mutation vérifiée (retirer une exclusion d'E2.1 → rouge).
- [x] E2.5 Statuer les 17 lecteurs de `internal/sync` (corriger ou dispense D-2, avec raison
  et `fichier:ligne`).
- [x] E2.6 Statuer les 10 lecteurs de `internal/ops`.
- [x] E2.7 Statuer les 29 lecteurs de `cmd`.
- [x] E2.8 Tout autre entrant révélé par l'extension (nouvelles tables) statué de même ; un
  entrant hors de toute catégorie = STOP, retour au superviseur.
- [x] E2.9 Ratchet `no_slug_comparison_test.go` vert ; aucune comparaison de slug ajoutée.
- [x] E2.10 (ajouté par le superviseur le 2026-10-15, revue du CR) Les 8 corrections de
  `internal/sync` et `internal/ops` lisent le titre par `ctxkeys.TitleSlug(ctx)`, qui retombe
  SILENCIEUSEMENT sur `halo_infinite` (aucune exclusion) si le contexte n'en porte pas. Pour
  chacune : énumérer TOUS les chemins d'appel de production jusqu'au point d'entrée (sync
  coordinateur, scheduler, handlers HTTP de recalcul, `cmd/levelup`, post-sync, rattrapages,
  dont `BatchComputePerformanceScores` exporté) et prouver par `fichier:ligne` que le titre est
  posé. Tout chemin sans titre : le poser au point d'entrée (ou passer le titre en paramètre),
  avec un test qui passe par ce point d'entrée sur `halo_5`. Aucun chemin Halo 5 ne doit
  dépendre du repli.
- [x] E2.11 Suite du retrait de `cmd/lusr_v2_phase0` : compilation des paquets `cmd/` qui
  importaient `internal/sync` (`go vet` ciblé), puis recherche des exports de `internal/sync`
  devenus sans appelant hors tests ; un orphelin est supprimé avec ses tests (règle n°7).

Gate : `go test ./internal/archlint/ ./internal/platform/duckdb/... ./internal/api/wire/` + les
paquets corrigés, puis `go test -tags=integration -p 1` sur les mêmes paquets.

### E3 — Petites dettes Go (item 6) — S

- [x] E3.1 `campaign_repo.go` `LoadAxisSamples` (l. ~278-286) et la requête sur `match_registry`
  (l. ~364-373) : temps canonique (fragment partagé, règle n°8) pour filtre et tri ; test.
- [x] E3.2 `engagement_score_repo_queries.go` `ListRecentPvPMatchIDs` (l. ~309-312) : temps
  canonique ; erreur de `Scan` (l. ~85 du bloc, `if err == nil`) et `continue` sur erreur sans
  log (l. ~34-38) → remontée ou `slog.WarnContext` + compteur ; test.
- [x] E3.3 `GET /squads` : retrait de `title_slug` (D-7), `make generate-types`,
  `make check-types`, appelants web mis à jour.
- [x] E3.4 D-6 appliqué dans `.ai/BACKLOG.md`.

Gate : `go test` des paquets touchés, `go test ./internal/api/...` (contrat OpenAPI golden),
`make check-types`, `make test-web` si un fichier web change.

### E4 — Vue match : « lecture en échec » distincte de « pas de données » — M

Origine : découverte E1 n°1, à traiter « proprement et solidement » (user, 2026-10-15).
Pièces : `service/match_view_data_loaders.go:262-269` (`goLoad` : erreur → `slog.Warn`, liste
vide), `:670-674` (`detectPartialMatchData`) ; cache des vues (ADR 0036) ; bandeau « match
partiel » côté web.

Décisions fermes :
- **D-8** Un chargeur en erreur ne produit PAS une liste vide indiscernable : la vue porte une
  raison distincte par source (`<source>_unavailable`, à côté des `<source>_empty` existantes),
  l'erreur est journalisée `slog.ErrorContext` (source, match_id, err) AVANT la dégradation.
- **D-9** Réponse toujours 200 partielle (dégradation gracieuse), mais une vue construite avec au
  moins une source `_unavailable` N'EST PAS mise en cache (ni cache serveur, ni en-têtes HTTP de
  cache permettant la réutilisation) : la panne ne survit pas au retour du chargeur.
- **D-10** Le web affiche un bandeau distinct « Données momentanément indisponibles, réessayez
  plus tard » / « Data temporarily unavailable, please try again later » (i18n FR+EN, tokens de
  couleur), différent du bandeau « match partiel ». Erreur ET vide simultanés : le bandeau
  « indisponible » prime.

- [x] E4.1 Inventaire sur pièces : toutes les sources chargées par `goLoad`, toutes les raisons
  `partial_reasons` émises, leurs lecteurs (Go, OpenAPI, web, tests E2E), le chemin de cache.
- [x] E4.2 Service : erreur ≠ vide (D-8) ; tests unitaires avec un chargeur en erreur et un
  chargeur vide, rouges avant.
- [x] E4.3 Cache : D-9, test qui prouve qu'une vue en erreur n'est pas servie depuis le cache
  au second appel une fois le chargeur rétabli, rouge avant.
- [x] E4.4 Contrat : OpenAPI (raisons documentées), `make generate-types`, golden à jour.
- [x] E4.5 Web : bandeau D-10 dans la logique (`*_logic.ts` / hook, pas dans le composant),
  i18n FR+EN, test vitest.
- [x] E4.6 Les autres raisons `partial_reasons` existantes gardent leur sens (non-régression).

Gate : `go test` des paquets touchés + `./internal/api/...`, `make check-types`, `make test-web`
(fichiers de la vue match), recette navigateur de la vue match (cas normal + chargeur coupé
simulé en test, pas sur les bases réelles).

### E5 — Registre des matchs : colonnes d'intégrité non remplies — M

Origine : découverte E1 n°2, à traiter « proprement et solidement » (user, 2026-10-15).
Pièces : `match_registry.participants_loaded` (DDL `sync/schema.go:207`, seul écrivain
`openspartan/mapper/mapper.go:69`), `match_registry.player_count` (oracle d'intégrité du roster,
« fix #10 », migration `games/halo_infinite/migrations/steps_shared_core.go:602-604`, écrit par
`games/halo_5/ingest/collect.go:47`, colonne de `persist/shared_persister.go:168`, lu par
`migration/steps_shared.go:275`), bit `participants` (512) de `backfill_completed`
(`sync/backfill_flags.go:141`) faux sur 157 matchs Halo Infinite qui ont leurs participants.

Décisions fermes :
- **D-11** Colonne LUE (lecteur de production identifié) → remplie à l'INSERTION par le chemin
  `persist` (INSERT-only, ADR 0019/0030) pour CHAQUE titre qui dispose de la donnée ; titre sans
  la donnée → valeur « inconnue » documentée, jamais une valeur fausse.
- **D-12** Colonne NON LUE → retirée du code (écrivains, DDL des bases neuves, fixtures, colonnes
  de seed démo) selon la règle n°7. La colonne physique des bases existantes : retirée seulement
  par un mécanisme de migration déjà éprouvé sur `match_registry` ; sinon STOP (décision user).
- **D-13** Lignes existantes : réparation uniquement par une recette sûre contre l'ART (ADR 0026,
  `migration/append_only_rebuild.go` ou équivalent existant). Si seul un `UPDATE` sur
  `match_registry` le permet : STOP, décision user (vecteur ART, cf. `cmd/h5-roster-topup:332`).
- **D-14** Bit 512 : établir qui le lit (sélection de rattrapage ?) et ce que coûte un faux
  négatif (re-téléchargement inutile) ; le poser là où les participants sont écrits.

- [ ] E5.1 Inventaire sur pièces, par titre (HI, H5, import OpenSpartan) : écrivains, lecteurs,
  date de la régression (`git log -S`), comptes sur une copie locale ET sur une copie prod
  (le superviseur fournit la copie prod). Rapport au journal, puis STOP pour verdict du
  superviseur (application de D-11 à D-14 colonne par colonne).
- [ ] E5.2 `participants_loaded` : traitement selon le verdict.
- [ ] E5.3 `player_count` : traitement selon le verdict.
- [ ] E5.4 Bit 512 : traitement selon le verdict.
- [ ] E5.5 Lignes existantes selon D-13.
- [ ] E5.6 Tests : persist (tag `integration`, OBLIGATOIRE), rouge avant ; ratchets ART verts
  (`no_art_patterns_test.go`, allowlist inchangée).

Gate : `go test` + `go test -tags=integration -p 1` sur `./internal/persist/... ./internal/sync/...
./internal/games/... ./internal/migration/...` (paquets touchés). Revue adversariale Opus du
diff E5 (lot persist/sync).

### E6 — Clôture

- [ ] E6.1 `make go-api-lint` (ratchet), `make check-types`.
- [ ] E6.2 Revue adversariale du diff complet (skill `adversarial-review`, contexte frais, Opus).
- [ ] E6.3 `.ai/BACKLOG.md` : items 4, 5, 6 retirés ou réécrits ; `.ai/REGISTRE_REPORTS.md`
  alimenté ; thought_log.
- [ ] E6.4 Push de la branche, CI verte (tout rouge se répare).
- [ ] E6.5 Fusion dans `feat/v75` : demandée au user.

## 3. Découvertes

- E1 : `partial_reasons` ne distingue pas « liste vide » de « chargeur en erreur » (`service/match_view_data_loaders.go:262-269`, `:670-674`) : une panne de lecteur s'affiche comme un match sans tableau des scores. → intégré au plan : étape E4 (user, 2026-10-15).
- E1 : `match_registry.participants_loaded` et `player_count` ne sont plus posés par le chemin courant (FALSE / 0 sur des matchs complets) : colonnes mortes ou à fiabiliser. → intégré au plan : étape E5 (user, 2026-10-15).
- E2 : `sync.FindMatchesMissingParticipantBits` (`internal/sync/backfill.go:86`) n'a aucun appelant de production (seulement `backfill_integration_test.go:219`, `:238`) : code mort au sens de la règle n°7, dispensé le temps du garde, à retirer avec ses tests. Non traité.
- E2 : `cmd/lusr_v2_phase0` SUPPRIMÉ (décision du user, règle n°7) ; E2.11 : aucun export de `internal/sync` devenu orphelin (15 constantes de calibrage réexportées par `skill_reexport.go` ne servent plus que `cmd/diag_lusr_player/replay.go` : dette de façade, à trancher avec cet outil). Fait.
- E2 : `sync.RecomputeAfterARTRebuild` (`recompute_after_art_rebuild.go:46`) n'a aucun appelant de production (seulement ses tests) : code mort, non traité.
- E2 : les valeurs DÉJÀ calculées pour les joueurs Halo 5 (sessions, score de performance, LUSR, modèle des assistances, engagement, dates de paliers) l'ont été Campagne comprise ; les chargeurs corrigés ne les recalculent pas d'eux-mêmes. À décider avec le user au déploiement ; rien n'est lancé.
- E2 : le commentaire de `games/halo_5/ingest/registry.go:46` (« engagement `loadMatchesForEngagement` exclut ces matchs » faute de `end_time`) décrivait un effet de bord, pas une exclusion de la Campagne : sans objet depuis la correction. Non traité.
- E3 (superviseur) : `SquadUsualContexts(ctx, roster, titleSlug)` garde un argument `titleSlug` ignoré par l'implémentation (`wire/prestige_lazy_service_squad.go:173-178`) ; depuis E3.3 le handler passe `""`. Paramètre mort de l'interface, à retirer (règle n°7). Non traité.
- E4 (exécutant) : les 16 autres chargeurs de `goLoad` (enrichment, objective score, kill_sources, kill_assists, assist_pairs, kv_pairs, skill_rank, shared_csrs, encounters, encounter_stats, encounter_assists, media, expected_stats, bulk_medals, bulk_weapons, hist_avg, + kill_distances et citations câblés sous condition) et le chargeur `eventsRepo.Load` (manuel) restent best-effort : un échec ne laisse que le journal (désormais WARN, inchangé), aucune raison dans la vue. D-8 est appliqué aux QUATRE sources qui portaient déjà une `<source>_empty` (le plan dit « à côté des `<source>_empty` existantes ») ; étendre aux autres demande un code par source et une chaîne web chacune. Non traité.
- E4 (exécutant) : `translatePartialReason` (`MatchViewPage.tsx`) porte quatre codes que Go n'émet plus jamais (`combat_narrative_unavailable`, `citations_unavailable`, `media_unavailable`, `accuracy_damage_taken_native_unavailable`) : code mort côté web (règle n°7), et `citations_unavailable` / `media_unavailable` auraient pu être pris pour des pannes de lecture (d'où l'ensemble FERMÉ `isUnavailableReason`). Ses libellés sont des ternaires FR/EN en ligne, pas dans `i18n.ts` (antérieur à E4). Non traité.
- E4 (exécutant) : `CachedHighlightEventsRepo` (`platform/duckdb/highlight_events_cache.go`) n'a aucun appelant de production (le câblage de la vue match utilise `NewHighlightEventsRepo` non décoré) : code mort. `match_view_data_loaders.go` fait 699 lignes (> 500, dette gelée, +1 ligne). Non traités.

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
- 2026-10-15 (E2, exécutant Sonnet) : E2.1 à E2.9 faits, gate vert. Cache `go-build-backlog1015`, un seul `go` à la fois.
  - **Corrections avec test rouge puis vert** (halo_5 rouge, halo_infinite neutre ; sorties dans `$TEMP/e2/rouge-*.log`) :
    | lecteur | test | rouge (halo_5) |
    |---|---|---|
    | `QKillsBetweenPlayers(Borne)` via `CompareRepo.GetEncounterStats`, `ExplorerRepo.GetKillerVictimBetween` (`kill_events_source.go`, jeton résolu `compare_repo.go:254`, `explorer_repo.go:103`) | `platform/duckdb/campaign_exclusion_e2_behavior_test.go` | 12 frags / 8 morts au lieu de 2 / 1 |
    | `Q36aMedalTotals` (`queries_citations.go`, `citations_repo.go:113`, `medals_repo.go:92`) | idem | `map[100:5 200:3]` au lieu de `map[100:1]` |
    | `loadEarnedMedalIDs` (`wire/post_sync_deltas_snapshot.go`, param `titleSlug` ajouté) | `wire/campaign_exclusion_medals_behavior_test.go` | 2 médailles au lieu de 1 |
    | `sync.loadAssistsSamples`, `loadHistoryForPerf`, `loadSessionMatchRowsDirect` (jeton), `loadMatchesForEngagement` | `sync/session_recalc_test.go` (4 tests) | 2 au lieu de 1 / `[arena1 camp1]` |
    | `skill.loadLUSRMatchData`, `loadShadowMatches` | `sync/skill/campaign_exclusion_test.go` | 2 au lieu de 1 |
    | `ops.loadCrossingMatches` (param `titleSlug`), `selectRecentMatchIDs` (param `titleSlug`) | `ops/milestone_dates_cgo_test.go` | 2 au lieu de 1 |
    Total dans `sync`+`ops` : 8 corrections (plafond 12/15 non atteint). Titre pris dans le contexte (`ctxkeys.TitleSlug(ctx)`, convention du moteur de sync) ou le paramètre déjà passé ; helpers `analysis.SQLExcludeCampaign*`, aucune comparaison de slug. Fixture `skill_v2_shadow_test.go` : colonne `game_variant_id` ajoutée.
  - **Garde étendu (D-1)** : critère = `match_participants|mv_player_matches|killer_victim_pairs|match_kill_events(_latest)|medals_earned` ET `xuid|_xuid = ?`/`IN (` ; racines + `internal/sync`, `internal/ops`, `cmd`. Sonde avant statut : 86 entrants (dont 3 déjà corrigés, 17 sync/10 ops/29 cmd attendus, plus nouveaux : sync 22, ops 11, duckdb 12, analysis 1). Mutation : `+ campaignExclusionToken` retiré de `killsBetweenPlayersSQL` -> `kill_events_source.go:killsBetweenPlayersSQL (ligne 88)` rouge (`mutation-garde.log`), retirée (grep `MUTANT E2` vide).
  - **Tableau des 56 lecteurs du journal B4 et entrants** (catégories de `campaign_exclusion_guard_dispenses_test.go`, preuves `fichier:ligne` dans chaque raison) :
    - sync corrigés (6) : assists_model, performance_helpers, session_recalc, engagement, skill_rating_loaders, skill_v2_shadow.
    - sync mono-match (10) : citations (loadMatchStats, loadMedalsForMatch), citations_weapons_source, comeback, enrichment_discipline, enrichments (2 bots), friends_recompute, medal_exploit_loader, snapshot_readiness_eval (décision du superviseur : la valeur ne dépend que du match lu).
    - sync sémantique, prédicat classé (2) : csr_backfill, csr_shared_backfill.
    - sync ensemble fourni / sélection de travail (4) : backfill.go (3), engine_backfills.loadAllMatchIDsForPlayer ; `FindMatchesMissingParticipantBits` y reste (code mort noté en section 3).
    - ops corrigés (2) : milestone_dates, seed_demo.selectRecentMatchIDs. ops sémantique (6) : archive (3), data_quality_examples, seed_demo_anonymize_check, seed_demo_corpus (classé), seed_demo_replay_picks. ops ensemble fourni (2) : sharedTablesWhere, loadDemoCorpusStats.
    - cmd (29 attendus, 28 dispensés « outil opérateur » ; `lusr_v2_phase0` SUPPRIMÉ sur décision du user, `seed-medal` inclus) : diag_* et audit_* par nom, backfill_* et h5-roster-* par UPDATE/INSERT du match lu.
    - entrants révélés par l'extension, hors liste des 56 : sync `loadMedalsForMatch`, `loadWeaponKillsFromSource`, `loadRankedMatchesForCSRBackfill`, `computeAndPersistH5DisciplineAwards`, `ComputeMedalExploitByMatch` ; ops `loadRealIdentities`, `loadDemoReplayCandidates` ; analysis `AnnuaireKillFeedLocaliserSQL` ; duckdb (12) : explorer médailles x2, Q26l, Q26h, medals_by_xuid, Q32d, Q32e, QTacticalMates, hijack, Q27, Q14 (tous ensemble fourni ou mono-match, preuves par appelant). Aucun entrant hors catégorie.
  - **Retrait de `cmd/lusr_v2_phase0`** (décision du user) : répertoire, entrée de `no_raw_start_time_literal_test.go`, mention de `skill_chain_provider.go` retirés ; ADR 0024 (historique) laissé.
  - **Gate** (logs `$TEMP/e2/gate-unit-{a,b}.log`, `gate-integ-{a,b}.log`) : sans tag, `archlint`, `platform/duckdb/...`, `api/wire`, `sync/...`, `ops/...` tous `ok` ; avec `-tags=integration -p 1`, mêmes paquets tous `ok`. Ratchets `no_slug_comparison_test.go`, `TestSyncRootPackageFrozen` (aucun fichier racine ajouté dans `internal/sync/`), `TestMatchRegistryFixturesDeTestAlignees` verts.
  - **E2 close.**
- 2026-10-15 (E2.10-E2.11, exécutant Sonnet, reprise après coupure API) :
  - **E2.10, méthode** : `ctxkeys.TitleSlug` retombe sur Infinite sans titre. Deux protections : (1) nouveaux helpers `analysis.SQLExcludeCampaignByMatchIDCtx` / `SQLResolveCampaignExclusionCtx` (`analysis/campaign_exclusion.go`) : titre posé -> forme title-aware, titre ABSENT -> forme title-agnostic (`SQLExcludeAll...`, tous les GUID de Campagne), jamais le repli sur Infinite ; les 6 lecteurs `sync` les utilisent ; reconnus par le garde. (2) titre de la BASE posé aux points d'entrée qui n'en portaient pas (jamais un littéral : `e.titleSlug` du moteur, `pdb.TitleSlug` du profil, `p.TitleSlug` du joueur). Tests : `TestCampaignExclusion_*` rejoués avec un 3e cas « contexte SANS titre » (rouge avec l'ancien repli : `rouge-e210.log`, 6 tests rouges ; vert après), `TestCampaignExclusion_TitreDesBasesPrimeSurLeCtx` (ctx entrant Infinite, base Halo 5 : mutant sur `pinTitle` et `titledCtx` -> rouge, `mutant-pins.log`).
  - **Tableau lecteur -> chemins de production -> preuve du titre**
    | lecteur | chemins jusqu'au point d'entrée | titre |
    |---|---|---|
    | `loadAssistsSamples` | `batchComputePlayerAssistsModel` <- `RunBackfillAssistsModel` (`assists_model.go:331` pinTitle, appelé par `cmd/levelup/cmd_backfill.go:702`) ; post-sync `runScoringStepsWithDB` (`engine_postsync_scoring.go:51` pinTitle) ; `BackfillEnrichmentFromShared` (`enrichment_backfill.go:139`) <- `cmd/h5-enrich/main.go:110`, `games/halo_5/livesync/wire.go:90` | posé (pinTitle ; `WithTitleSlug(halo5.TitleSlug)`) ; sinon repli title-agnostique |
    | `loadHistoryForPerf` | `batchComputePerformanceScores` <- `RunBackfillPerf` (`engine_backfills.go:345` pinTitle), post-sync (`engine_postsync_scoring.go:51`), `MatchRecomputer.RecomputeAfterExclusion` (`match_recomputer.go`, `titledCtx`, titre du profil fourni par `wire/registry_pages_home.go:186` `.WithTitle(pdb.TitleSlug)` ; ce chemin n'en posait AUCUN : corrigé), `BatchComputePerformanceScores` exporté <- `service/openspartan_post_import_service.go:269` (`postImportCtx` `:95`, `:130`), `BackfillEnrichmentFromShared` (`:115`, mêmes entrées que ci-dessus) | posé (corrigé pour l'exclusion de match) |
    | `loadSessionMatchRowsDirect` | `appendSessionsInline` <- `recalculateSessionsInline` <- post-sync (`engine_postsync_scoring.go:73`, pinTitle `:51`), `BackfillEnrichmentFromShared` (`enrichment_backfill.go:108`) ; `RecalculatePlayerSessions` <- `api/handlers/settings.go:557` (ctx `context.Background()` SANS titre : corrigé, `WithTitleSlug(…, p.TitleSlug)`), `openspartan_post_import_service.go:240` (`postImportCtx`) | posé |
    | `loadMatchesForEngagement` | `batchComputeEngagementScores` <- `RunBackfillEngagementScores` (`engine_backfills.go:74` pinTitle ; `cmd/levelup/cmd_backfill.go:293`, `cmd_engagement_coefs.go:97`, `service/backfill_orchestrator.go:164` en `context.Background()` : corrigé par pinTitle), post-sync (`engine_postsync_scoring.go:51`), `BackfillEnrichmentFromShared` (`enrichment_backfill.go:122`) | posé |
    | `loadLUSRMatchData` | `BatchComputeLUSRWithMedals` <- `BatchComputeLUSR` <- post-sync `runLUSRV1UnderRead` (`engine_postsync_scoring.go:184` pinTitle), `MatchRecomputer` (titre corrigé), `RecomputeAfterARTRebuild` (`recompute_after_art_rebuild.go:76`, aucun appelant hors tests : découverte) ; `RunFormulaSim` <- `engine_backfills.go:168` (pinTitle) <- `cmd/levelup/cmd_backfill.go:546` ; `BatchComputeLUSRPreview` sans appelant | posé |
    | `loadShadowMatches` | `RunLUSRV2ShadowOwnerOnly` <- post-sync `runSkillRatingSteps` (`engine_postsync_scoring.go:143` pinTitle + `:162` `WithTitleSlug`), `RecomputeLUSRCanonicalForPlayer` <- `engine_backfills.go:214-217` (`WithTitleSlug(e.titleSlug)` déjà), `openspartan_post_import_service.go:171` (`postImportCtx`), `cmd/h5-lusr-backfill/main.go:108` et `cmd/h5-lusr-smoke/main.go:66` (`WithTitleSlug(halo5)`), `cmd/recompute_perfnote` (titre constant Infinite, `main.go:46`), `games/halo_5/livesync/wire.go:90` ; `cmd/lusr_v2_canonical_backfill/main.go:118` et `cmd/lusr_v2_replay/main.go:96` en `context.Background()` (binaires Infinite) ; `ScanLUSRGaps` <- `scheduler/data_health_lusr.go:71` et `wire/registry_lusr_gaps.go:35,165` (`WithTitleSlug`) | posé ; seuls les binaires Infinite sans titre retombent, sans Campagne à exclure |
    | `ops.loadCrossingMatches` | paramètre `titleSlug` de `BackfillMilestoneDates` (`milestone_dates.go:234`) <- `cmd/backfill_milestone_dates/main.go:35,74` (`filepath.Base(--data-root)` = slug du dossier du titre lu) | donnée de la base lue |
    | `ops.selectRecentMatchIDs` | paramètre `opts.TitleSlug` (`seed_demo.go:446,706`) <- `validateSeedDemoOpts` (`seed_demo.go:591`, défaut `DefaultSlug` si vide) <- `SeedDemo`/`EmitDemoManifest`, options de `SeedDemoMulti` (`seed_demo_multititle.go:159,172`) ; titre = celui des options (`opts.TitleSlug`, aussi utilisé pour les chemins de sortie `seed_demo.go:320-324`) | option du seed (défaut Infinite documenté) |
  - **Points d'entrée corrigés** : `SyncEngine.pinTitle` (5 chemins backfill + 3 du post-sync), `MatchRecomputer.WithTitle`/`titledCtx` (exclusion d'un match), `handlers/settings.go:557`. Test d'entrée sur halo_5 : `TestCampaignExclusion_TitreDesBasesPrimeSurLeCtx` (rouge par mutant) ; le chemin HTTP de `settings.go:557` n'a pas de test d'entrée dédié (handler lourd) : couvert par le cas « sans titre » des lecteurs, qui exclut désormais la Campagne même sans titre.
  - **E2.11** : `go vet` des 29 paquets `cmd/` qui importent `internal/sync` : sans diagnostic. Les 24 exports consommés par l'ancien outil (`Beta`, `InitialMU`, `InitialSigma`, `MaxInactivityDays`, `InactivityThresholdDay`, `InactivitySigmaPerDay`, `MaxSigma`, `MinSigma`, `IndividualMUAlpha`, `KElo`, `MinRating`, `Tau`, `MinMatchesForAccuracyDelta`, `AccuracyHistorySize`, `DefaultOpponentSigma`, `CompositeWeights`, `GetLUSRChain`, `MetricKey*`) : tous encore utilisés hors tests. Les 15 constantes de calibrage sont réexportées par `internal/sync/skill_reexport.go` et ne sont consommées hors `skill` que par `cmd/diag_lusr_player/replay.go` (outil de diagnostic conservé) : AUCUN orphelin au sens de la règle 7 (consommateur vivant) ; leur réexport dans `sync` ne sert que cet outil, dette de façade à trancher avec lui. `GetLUSRChain`, `CompositeWeights`, `MetricKey*` : consommateurs de production multiples. Rien supprimé.
  - **Gates E2.10-11** : unitaire (`archlint`, `platform/duckdb/...`, `api/wire`, `analysis`, `api/handlers`, `sync/...`, `ops/...`) `ok` ; intégration `-p 1` : première passe rouge sur 30+ tests `sync`/`skill` dont les fixtures `match_registry` n'avaient pas `game_variant_id` (le repli title-agnostique émet toujours la clause : la colonne existe en prod, fixtures alignées : `performance_integration_test`, `lusrdb_helpers_test`, `recompute_after_art_rebuild_test`, `sync_pipeline_fixture_test`, `skill_rating_loaders_test`, `skill_rating_dryrun_test`), puis `sync/...` + `archlint` verts ; autres paquets verts (`g2-integ-a/b/c.log`).
- 2026-10-15 (E3, exécutant Sonnet) : E3.1 à E3.4 faits, gate vert (sorties dans `$TEMP/e3-*.log`).
  - **E3.1 / E3.2, tests rouges avant** (`platform/duckdb/temps_canonique_e3_test.go`, tag `integration`, 6 tests, tous rouges avant correction, verts après). Fixture : import à `start_time` NULL, `start_time` naïf divergent de `start_time_utc`, ligne hors fenêtre au temps canonique.
    | test | rouge avant | vert après |
    |---|---|---|
    | `LoadAxisSamples` radar | `[9 3 2]` au lieu de `[1 2 3]` | `[1 2 3]` |
    | `LoadAxisSamples` composante LUSR (`loadLUSRMatchIDsInWindow`) | `[0.9 0.3 0.2]` au lieu de `[0.1 0.2 0.3]` | ok |
    | `ListRecentPvPMatchIDs` (limite 4 et 2) | `[mD mC mB]`, `[mC mB]` | `[mA mB mC mD]`, `[mC mD]` |
    | `LoadEventsForMatch` ligne illisible | aucune erreur, ligne sautée | erreur remontée |
    | `LoadAllCoefficients` ligne illisible | aucune erreur, ligne sautée | erreur remontée |
    | kills synthétiques illisibles | journal vide | `WARN` avec `err=` |
  - **Temps canonique** : `StartTimeCanonicalSQL` (délégué de `analysis.SQLStartTimeCanonical`) pour le filtre ET le tri de `LoadAxisSamples` et de `loadLUSRMatchIDsInWindow` (alias vide), et pour le filtre `IS NOT NULL` et le tri de `ListRecentPvPMatchIDs`. Les deux fichiers sortent de `rawOrderByStartTimeAllowlist` (archlint, plus aucun `ORDER BY x.start_time` brut) : l'allowlist rétrécit.
  - **Erreurs, choix remontée vs log** (`engagement_score_repo_queries.go`) : REMONTÉE pour `LoadEventsForMatch`, `loadSyntheticKillEvents`, `LoadTeamXUIDs`, `LoadAllCoefficients`, `ListRecentPvPMatchIDs` (un `Scan` en échec ne doit pas produire une liste tronquée qui ressemble à un résultat : tous les appelants (`port.EngagementScoreRepo`, services engagement/timeseries) traitent déjà l'erreur de ces méthodes, qui retournent `error`). LOG `slog.WarnContext` puis dégradation pour deux sites best-effort dont la dégradation est documentée : kills synthétiques (`serr`, repli sur les médailles seules, l'erreur était ignorée) et effectifs d'équipe (`_ = ...Scan`, repli sur 0). Ajout, dans `loadLUSRMatchIDsInWindow`, du contrôle `sharedRows.Err()` manquant (même bloc, même cause : erreur de lecture avalée). Pas de compteur : aucun compteur existant sur ces chemins, le log suffit.
  - **E3.3, preuve que `title_slug` est sans effet** : le handler le passait comme 3e argument à `SquadUsualContexts`, que `LazyPrestigeService` ignore (`_ string`) et remplace par `pdb.TitleSlug` (`wire/prestige_lazy_service_squad.go:173-178`) ; le web n'envoie jamais ce paramètre (`prestige.ts:478`, `prestige.paths.test.ts:145`). Retiré de `listMySquadsInput` ; le handler passe `""` (la méthode d'interface partagée avec le service non paresseux garde sa signature : hors périmètre). Contrat : `make openapi-gen` puis `make generate-types` ; diff = `title_slug` supprimé des paramètres de requête de `listMySquads` (`openapi.yaml` -5 lignes, `generated.ts` -1). Aucun appelant web à nettoyer (aucun ne l'envoyait).
  - **E3.4** : entrée « [go/hygiène] » du backlog supprimée (4 lignes : 2 traitées, 2 retirées par D-6).
  - **Gate** : `go test` sans tag puis `-tags=integration -p 1` sur `archlint`, `platform/duckdb`, `api/...` : tous `ok` (golden OpenAPI `TestOpenAPIYAMLIsUpToDate` inclus dans `internal/api`) ; `make check-types` sans diagnostic ; `make test-web` : 919 fichiers passés, 9598 tests.
  - **E3 close.**
- 2026-10-15 (E4, exécutant Sonnet) : E4.1 à E4.6 faits, gate vert, recette navigateur `[!]` (voir plus bas).
  - **E4.1, inventaire sur pièces (écrit avant de coder)**
    - Sources de `loadMatchViewDataParallel` (`service/match_view_data_loaders.go`) : par `goLoad` (WARN, erreur avalée, valeur
      zéro) : stats, enrichment, scoreboard, objective score, medals, events, kill_sources, kill_assists, assist_pairs, kv_pairs,
      skill_rank, shared_csrs, encounters, encounter_stats, encounter_assists (`match_view_encounter_assists.go`), media,
      expected_stats, bulk_medals, bulk_weapons, hist_avg ; conditionnelles : kill_distances, citations, rich citations ; hors
      `goLoad` : `eventsRepo.Load` (`g.Go` manuel, WARN, repli sur la conversion de `events`), awards et friendsExtras (après
      `g.Wait`). Seules quatre sources alimentent `partial_reasons` : stats, scoreboard, events, medals.
    - `partial_reasons` ÉMISES par Go : uniquement `detectPartialMatchData` (`:655-690`) : `scoreboard_empty`, `events_empty`,
      `player_stats_empty` (stats nil ou outcome 0), `medals_empty` (0 médaille ET bit PBitMedals absent). Posées dans
      `buildMatchViewFromData` (`IsPartial = len > 0`). Autre usage du mot `partial_reasons` : colonne
      `player_match_enrichment.partial_reasons` de la préparation des snapshots (vocabulaire distinct, `sync/snapshot`) : sans lien
      avec la vue, non touchée.
    - LUES : web `MatchViewPage.tsx` (`translatePartialReason`, bandeau), `lib/api/types.ts` (type, commentaire) ; aucun test
      E2E Playwright (`apps/web/e2e/` : seul `compare-bars.spec.ts` contient « partial », sans rapport) ; tests Go :
      `match_view_dominance_test.go:434-510` (5 tests de `detectPartialMatchData`, inchangés et verts).
    - CHEMIN DE CACHE COMPLET de la vue match : (1) AUCUN cache serveur de la réponse : le service est reconstruit à chaque requête
      (`wire.MatchView`, `ServiceFactory`), aucun cache de réponse, aucun `singleflight` sur ce chemin ; `CachedHighlightEventsRepo`
      existe mais n'est pas câblé (la vue utilise `NewHighlightEventsRepo`) ; `player_read_cache` (ADR 0036 I3) ne porte ni la vue
      ni ses sources ; (2) AUCUN en-tête HTTP de cache sur `GET /players/{slug}/matches/{id}` (pas d'ETag, pas de Cache-Control :
      `middleware.CacheMaxAge` n'est monté nulle part, `cache_http.go` ne sert que les blobs/assets) ; (3) lecture de source :
      `SnapshotPreferredSharedReader` (singleton par titre, cache de queriers `:memory:` et negative-cache par VERSION de cut,
      repli live) : cache de CONNEXION, pas de résultat, hors périmètre ; (4) le SEUL cache qui réutilise une vue : TanStack Query
      côté web, `useMatchView` `staleTime` 10 min (`features/match-view/queries.ts`) : une vue dégradée y restait 10 minutes ;
      pas de service worker, pas de persistance de requêtes, pas de préchargement de `matchView` (`lib/query/prefetch.ts` ne le
      touche pas). Le cache n'est donc pas « côté serveur » comme le suppose la lettre de D-9, mais l'intention (la panne ne survit
      pas au retour du chargeur) est portée par le cache web ; décision : traiter (4) en E4.5 (`staleTime` fonction de la vue) et poser
      `no-store` côté HTTP en garde-fou. Aucune raison existante ne change de sens.
  - **E4.2 / D-8** : quatre raisons nouvelles, à côté des `_empty` : `scoreboard_unavailable`, `events_unavailable`,
    `player_stats_unavailable`, `medals_unavailable` (constantes `domain.PartialReason*Unavailable`, `domain.IsUnavailableReason`,
    `MatchViewResponse.HasUnavailableSource`). Une source en échec n'émet JAMAIS aussi son `_empty` (`matchPartialReasons`,
    `service/match_view_source_failures.go`, fichier neuf ; `detectPartialMatchData` inchangée). `sourceFailures.goLoad` : journal
    `slog.ErrorContext(ctx, "match_view: source indisponible", "source", ..., "match_id", ..., "err", ...)` AVANT la dégradation (WARN si
    l'erreur est `context.Canceled` : requête abandonnée, pas panne de la source). Périmètre : les quatre sources portant déjà une
    `_empty` (voir section 3 pour les autres). Tests `service/match_view_unavailable_test.go` : 4 sources x (erreur, vide), panne +
    vide simultanés, journal. Rouge avant (mutant : `if false && d.failures.has(src)`, équivalent à l'ancien comportement) : 4 cas
    « erreur », « panne et vide simultanés », « la panne ne survit pas » en échec ; vert après.
  - **E4.3 / D-9** : (a) service : `TestMatchView_LaPanneNeSurvitPasAuRetourDuChargeur` (même service, chargeur du tableau des scores
    rétabli entre deux appels : 2e vue complète, tableau servi, aucune raison) : aucun cache serveur de la vue n'existe (inventaire),
    il fige donc l'invariant ; rouge avec le mutant ci-dessus (le 1er appel n'y est plus « en échec ») ; (b) HTTP :
    `handlers/match_view_test.go` `TestMatchViewHandler_CacheControl_VueEnEchec` : `Cache-Control: no-store` sur une vue avec
    source en panne, AUCUN en-tête sur vue complète, vue à source vide seule et code homonyme structurel (`media_unavailable`) ; rouge
    avant (2 cas en échec avec `if false &&`) ; (c) web : `queries.matchView.test.tsx` : retour sur la page après une vue en panne =
    nouvel appel et vue complète servie ; vue complète = un seul appel ; rouge avant (`staleTime` fixe 10 min : le test « vue en panne »
    échoue), vert après. `matchViewStaleTime` testé dans `matchPartial.logic.test.ts`.
  - **E4.4 contrat** : `make openapi-gen` + `make generate-types`. Diff : `partial_reasons` documenté (`doc:` : `_empty` vs `_unavailable`,
    vue transitoire, no-store) ; en-tête de réponse `Cache-Control` (string, optionnel) sur `getMatchView` 200. `generated.ts` régénéré
    (2 hunks), `types.ts` (commentaire des codes). `TestOpenAPIYAMLIsUpToDate` (`internal/api`) vert.
  - **E4.5 web** : `features/match-view/matchPartial.logic.ts` (`selectMatchPartialBanner` : `unavailable` prime sur `partial`, ensemble
    fermé des quatre codes ; `matchViewStaleTime`), bandeau dans `MatchViewPage.tsx` (tokens `destructive` : `border-destructive/50
    bg-destructive/10 text-destructive`, aucun hex), i18n `pageUnavailableBanner` FR « Données momentanément indisponibles, réessayez
    plus tard » / EN « Data temporarily unavailable, please try again later » (parité typée dans `MatchViewText`). Tests vitest :
    logique, page (4 : panne seule, panne + vide, vide seul inchangé, complète), cache (2).
  - **E4.6** : les cinq tests `TestDetectPartialMatchData_*` inchangés et verts ; test de page « source vide seule » garde le bandeau
    partiel et son texte ; `translatePartialReason` intact.
  - **Gate** : `go test ./internal/domain/ ./internal/service/` ok (première passe de `internal/service` en échec, non reproduit : 7
    rejeux `-count=1` verts, cause non établie) ; `go test -count=1 ./internal/api/... ./internal/archlint/` ok (api 25.9 s, handlers
    10.5 s, wire, archlint 73.8 s) ; `go vet` service/domain/handlers sans diagnostic ; `make check-types` sans diagnostic ; `make
    test-web` : 921 fichiers passés (5 ignorés), 9614 tests ; eslint des fichiers touchés sans diagnostic.
  - **Recette navigateur : `[!]`** : `http://127.0.0.1:8000/health` répond 200 mais ce serveur n'est pas un build de cette branche
    (aucun binaire de ce worktree lancé) : il ne servirait ni le bandeau ni l'en-tête. Le cas « chargeur coupé » est de toute façon
    simulé en test (interdit sur les bases réelles) ; le cas normal et le rendu des bandeaux sont couverts par les tests de page. À
    faire par le superviseur sur un build de la branche.
