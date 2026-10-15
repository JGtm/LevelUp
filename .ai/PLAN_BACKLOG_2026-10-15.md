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
- [x] E4.7 (ajout du superviseur, 2026-10-15, « propre et solide ») :
  - [x] E4.7a TOUT chargeur de `goLoad` en erreur (hors annulation de requête, qui reste WARN) : `slog.ErrorContext` (source,
    match_id, err) AVANT la dégradation, et vue « en panne partielle » par un code générique unique `section_unavailable` pour les
    sources sans `_unavailable` dédié (la source est dans le log, pas dans le contrat) ; même règle no-store / refetch ; même
    bandeau web ; tests rouge puis vert. Aucune ligne nette dans `match_view_data_loaders.go` (699 L gelées).
  - [x] E4.7b Les 4 codes web que Go n'émet plus (`combat_narrative_unavailable`, `citations_unavailable`, `media_unavailable`,
    `accuracy_damage_taken_native_unavailable`) supprimés du web (traduction, types, tests, i18n) après grep du dépôt entier ;
    découverte correspondante de la section 3 retirée.
  - [x] E4.7c Test de `internal/service` en échec une fois : rejeu `-count=20`, cause nommée ou note en section 3.
- [x] E4.8 (ajout du superviseur, 2026-10-15) : faire remonter l'erreur des méthodes de repo/adapter de la vue match qui l'avalent
  (le service les classe alors par E4.7a / dédiée), adapter les autres appelants (log ERROR avant dégradation), `eventsRepo.Load` /
  awards / friendsExtras dans le contrat ; « table absente / titre sans la donnée » distinguée par erreur typée ou capability
  existante, jamais par un nil générique ; tests rouge puis vert. Découpé le 2026-10-15 après le STOP d'inventaire (> 12) :
  - [x] E4.8a Lot A : #1 `GetMatchEvents`/`lireEvenementsDuMatch`, #2 `GetMatchKillSources`, #3 `GetMatchKillAssists`, #4 `GetMatchKVPairs`,
    #12 `GetMatchObjectiveScore`, #13 `loadMatchObjectiveStats`, #14 `loadMatchBombStats`, + `eventsRepo.Load` et awards (service).
  - [x] E4.8b Lot B (exécutant neuf) : #5-#11 (`GetMatchMedia`, `GetMatchExpectedStats`, `GetHistoryForAvg`/`Bulk`, `GetMatchSkillRank`,
    `GetMatchSharedCSRs`, `GetMatchBulkMedals`, `GetMatchBulkWeaponKills`+`bulkWeaponKillsFromSource`), #15 `GetPlayerAssistsModel`,
    #16 à vérifier (`GetPlayerMatchStats`, `GetMatchEnrichment`), les 3 sites de lignes ignorées sans compteur (#17 : `scanAssistPairs`,
    `lookupMedalMeta`, `lookupLabelsByID`) et le changement du port `FriendsExtrasResolver` (port interne, autorisé : pas de contrat d'API).
  - [x] E4.8c Garde-rail archlint (AST ou grep, modèle des ratchets existants) : interdit dans `platform/duckdb/match_view_repo*.go` un
    `return nil, nil` / valeur zéro + `nil` dans une branche `if err != nil`, et un `continue` dans une branche `if err != nil` sans appel
    `slog`/compteur ; allowlist VIDE à la fin de E4.8b, mutation vérifiée.
  - [x] E4.8-inventaire : STOP déclenché (> 12), inventaire au journal ; découpage validé par le superviseur (a / b / c ci-dessus).

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

- [x] E5.1 Inventaire sur pièces, par titre (HI, H5, import OpenSpartan) : écrivains, lecteurs,
  date de la régression (`git log -S`), comptes sur une copie locale ET sur une copie prod
  (le superviseur fournit la copie prod). Rapport au journal, puis STOP pour verdict du
  superviseur (application de D-11 à D-14 colonne par colonne).
**Verdict du superviseur (2026-10-15, sur l'inventaire E5.1)** :
- **V-1 `participants_loaded` → D-12, RETIRÉE** : aucun écrivain Go, aucun lecteur. Code ET colonne
  physique (V-4).
- **V-2 `player_count` → D-12, RETIRÉE** : aucun lecteur réel (pas d'oracle malgré « fix #10 »),
  valeurs historiques fausses (13 sur 1 544 égales au roster), écrivain H5 jamais exercé. Pas de
  remplissage rétroactif (oracle tautologique). Un vrai oracle de roster serait une fonctionnalité
  neuve, hors de ce plan (proposée au user, non bloquante). Code, vues, export, démo, TSV ET
  colonne physique + index orphelin `idx_registry_player_count` (V-4).
- **V-3 Bit 512 → D-14, GARDÉ** : un lecteur de production (backfill admin HI). Écrivain HI à
  l'insertion correct, inchangé. Import OpenSpartan : pose le bit 512 à l'insertion (et le
  `BackfillCompleted: 1` du mapper, qui est le bit médailles, est nommé/commenté pour ce qu'il est).
  H5 : rien (aucun lecteur). La condition « participants » du scope `AllData` reste.
- **V-4 Lignes et schéma existants → D-13, UNE reconstruction** par la recette éprouvée sur
  `match_registry` (`migration/table_swap.go` `swapTableTx`, précédent
  `shared_recompute_mode_category_v1` du 2026-10-09, et/ou `widen_match_registry_team_scores`) :
  une étape de migration unique qui, sur les bases partagées, retire `participants_loaded`,
  `player_count` et `idx_registry_player_count`, et pose le bit 512 sur les matchs qui ont des
  participants, seulement pour les titres dont le manifeste/capability porte ce bit (aucune
  comparaison de slug). Aucun `UPDATE`. Idempotente, rejouable, empreintes des vues vérifiées.
  Le bit skill (4) manquant sur les mêmes 157 matchs n'est PAS posé (donnée non vérifiable depuis
  la base partagée) : découverte.

- [x] E5.2 V-1 côté code : DDL `sync/schema.go`, fixtures, champ et ligne du mapper OpenSpartan,
  tests ; grep final à 0 hors migration de retrait et archives.
- [x] E5.3 V-2 côté code : champ `domain.MatchRegistryRow.PlayerCount`, écrivain H5
  (`ingest/collect.go`, `capture.go`), seed démo et `persist/demo_seed_columns.go`, étape de
  migration d'ajout (retirée de `order.go` seulement si le mécanisme de migration le permet sans
  casser l'historique ; sinon conservée et neutralisée par l'étape de retrait, justifié), DDL,
  vues `mv_player_matches` / `v_match_full`, export Parquet du snapshot (le lecteur de snapshot
  doit TOLÉRER un cut ancien qui porte encore la colonne : test), colonne TSV de
  `cmd/mapfond-inventaire`. Grep final à 0 hors migration de retrait et archives.
- [ ] E5.4 V-3 : import OpenSpartan pose le bit 512 à l'insertion (`toSyncRegistry`) ; test.
- [ ] E5.5 V-4 : étape de migration de reconstruction, tests (base aux deux colonnes + index +
  157 lignes sans bit → après : colonnes et index absents, bits posés, autres colonnes et index
  identiques, vues valides ; second passage = no-op). **Répétition sur les COPIES prod** fournies
  par le superviseur (`scratchpad/e5_prod/{halo_infinite,halo_5}`, sur une copie de la copie) :
  comptes avant/après, liste des index, durée, mémoire ; résultats au journal.
- [ ] E5.6 Tests : persist/migration (tag `integration`, OBLIGATOIRE), rouge avant ; ratchets ART
  verts (`no_art_patterns_test.go`, allowlist inchangée, `criticalMatchTables`/`guardedSharedTables`
  respectées).

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
- E4 (exécutant) : les chargeurs de `goLoad` sans `_empty` ne portent plus de raison dédiée mais le code générique `section_unavailable` (E4.7a). Reste hors contrat : `eventsRepo.Load` (`g.Go` manuel, repli sur la conversion de `events`), awards et friendsExtras ; et plusieurs méthodes du repo (`GetMatchKillSources`, `GetMatchKillAssists`, ...) avalent elles-mêmes leur erreur (`return nil, nil` + WARN) avant d'atteindre le service : une panne de ces sources reste invisible de la vue. Non traité.
- E4 (exécutant) : `CachedHighlightEventsRepo` (`platform/duckdb/highlight_events_cache.go`) n'a aucun appelant de production (le câblage de la vue match utilise `NewHighlightEventsRepo` non décoré) : code mort. `match_view_data_loaders.go` fait 699 lignes (> 500, dette gelée, +1 ligne). Non traités.

- E4.7c : un échec unique de `go test ./internal/service/` (avec `./internal/domain/`, cache froid) pendant E4, non reproduit : `go test -count=20 -run "TestMatchView_|TestDetectPartialMatchData|TestMatchViewService" ./internal/service/` vert puis 3 passes complètes vertes. Cause non établie ; si elle revient : `go test -count=1 ./internal/service/ 2>&1 | grep -E "^(--- FAIL|panic)"` pour relever le nom.

- E4.8a : `loadAwardsForScoreboard`, `MatchViewService.WithAwardsRepo` et le champ `awardsRepo` n'ont aucun appelant de production : code mort (règle n°7), non traité. `match_bomb_stats` n'est pas exporté dans le snapshot (`ops/snapshot_export.go`) : sur un match servi du snapshot, les statistiques d'Assaut sont absentes de la vue (comportement antérieur inchangé, désormais constaté explicitement dans `loadMatchBombStats`) ; à corriger par l'export de la table ou une lecture live, hors E4.
- E4.8b : l'AST du garde (E4.8c) a relevé trois sites HORS inventaire, traités parce que l'allowlist est vide : `GetMatchNeighbors` et `GetMatchNeighborsFiltered` (voisins vides sur lecteur indisponible ou requête en échec) et `LookupMedalMetaByName` (WARN + map vide). Restent HORS garde (ils ne renvoient pas d'`error`) et non traités : `lookupMapImageURL`, `ResolveAssetNamesBulk` ignoré par `_` et la résolution de nom d'asset de `GetMatchMeta` (`match_view_repo.go:169`, `:305-333`), `lookupWeaponMeta`/`resolveWeaponMeta` (libellés d'armes), `loadKnownModesEN` ; côté service, `computeExpectedAssists` rend `nil` en silence quand `GetAssistsCoef` échoue, et `GetMatchNeighbors`/`GetMatchNeighborsFiltered` (service) rendent encore des voisins vides + nil sur erreur du repo (journal passé en ERREUR, comportement inchangé). Le garde ne couvre pas non plus `if err == nil && ...` (forme du lecteur illisible silencieux : `match_view_repo.go:333`, `lookupMapImageURL`).
- E4.8b : `friendsByXUID` (`wire/registry_pages.go`) incluait TOUS les profils du titre, `auth_only` compris (aucune player DB de stats) : avec une panne remontée, chaque vue contenant un tel ami serait passée en `section_unavailable`. Corrigé dans le même lot (`friendProfilesByXUID`, test), parce que le changement de port rendait la différence absence/panne observable. Reste : un profil SUIVI dont la player DB n'a jamais été créée ou migrée (ami onboardé, jamais synchronisé) est désormais une panne dite pour ses vues ; à surveiller au premier déploiement (journal ERROR `friends_extras: open player db failed` / `load ... failed`).

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
  - **Recette navigateur : `[!]`** : le serveur local (`:8000`) sert `feat/v75`, pas cette branche : il ne montrerait ni le bandeau ni
    l'en-tête. Reportée au gate visuel du user après fusion (décision du superviseur, 2026-10-15). Le cas « chargeur coupé » est de
    toute façon simulé en test (interdit sur les bases réelles) ; le rendu des bandeaux est couvert par les tests de page.
- 2026-10-15 (E4.7, exécutant Sonnet, sur demande du superviseur) : gate vert.
  - **E4.7a** : `goLoad` libre supprimé ; `sourceFailures.goLoadSection` (même journal ERROR source/match_id/err, WARN si requête abandonnée)
    pour les 17 chargeurs sans raison dédiée : code unique `domain.PartialReasonSectionUnavailable = "section_unavailable"`, ajouté à
    `IsUnavailableReason` (donc no-store HTTP, `staleTime` 0, bandeau « indisponible ») et au web (`UNAVAILABLE_REASONS`). Une capability non
    supportée (`games.ErrCapabilityNotSupported`, ex. distances de frag d'un titre sans film) n'est PAS une panne (Debug, aucune raison).
    Ordre : sources dédiées, puis `section_unavailable`. `match_view_data_loaders.go` : 699 -> 689 lignes (aucune ligne nette ajoutée).
    Tests rouge puis vert (mutation : `section` jamais posé) : `TestMatchView_ChargeurSansRaisonDediee_SectionUnavailable` (code, journal
    ERROR `source=kv_pairs`, 2e appel complet), `TestMatchView_PanneDedieeEtGenerique` ; verts : capability non supportée, handler no-store
    sur `section_unavailable`, `matchPartial.logic` et page (bandeau « indisponible »). Contrat : doc de `partial_reasons` (openapi.yaml +
    generated.ts, 1 ligne chacun) régénérés.
  - **E4.7b** : grep du dépôt hors `.ai` (Go, OpenAPI, web, e2e, config) : aucun émetteur des 4 codes. Supprimés de `translatePartialReason`
    (ces libellés étaient des ternaires en ligne, aucune clé `i18n.ts`), des tests et du type ; le test « code homonyme » devient « code inconnu ».
    Découverte correspondante retirée de la section 3.
  - **E4.7c** : voir section 3 (non reproduit, repro notée).
  - **Gate** : `go vet` ; `go test -count=1 ./internal/domain/ ./internal/api/... ./internal/archlint/` ok ; `internal/service` ok (x20 ciblé + 3 complets) ;
    `make generate-types` ; `make check-types` sans diagnostic ; `make test-web` 921 fichiers, 9 616 tests ; eslint propre.
- 2026-10-15 (E4.8, exécutant Sonnet) : inventaire seul, STOP (plus de 12 méthodes). Aucun fichier de code modifié. Aucun `persist`/`sync`.
  **Méthodes qui avalent leur erreur (`platform/duckdb/match_view_repo*.go`), 17 sites hors service :**
  | # | méthode (fichier:ligne) | avale | autres appelants à vérifier |
  |---|---|---|---|
  | 1 | `GetMatchEvents` (`_noms.go:49`) + `lireEvenementsDuMatch` (`:83`, WARN, `ok=false`) | lecteur partagé indisponible -> `nil, nil` ; requête Q21 en échec -> WARN | à grepper |
  | 2 | `GetMatchKillSources` (`_extras.go:36-45`) | `nil, nil` + WARN (lecteur, Q21b) | idem |
  | 3 | `GetMatchKillAssists` (`_extras.go:83-92`) | `nil, nil` + WARN (lecteur, Q21c) | idem |
  | 4 | `GetMatchKVPairs` (`_extras.go:139-150`) | `nil, nil` + WARN (lecteur, Q20) | idem |
  | 5 | `GetMatchMedia` (`_extras.go:265-278`) | `nil, nil` (lecteur et Q24, `nolint:nilerr`) | idem |
  | 6 | `GetMatchExpectedStats` (`_extras.go:332,345`) | `nil, nil` « colonnes absentes ou match introuvable » (mélange absence et panne) | idem |
  | 7 | `GetHistoryForAvg` (`_extras.go:360-371`) | `nil, nil` + WARN (jumelle `GetHistoryForAvgBulk` `:411-431`, appelée ailleurs) | idem |
  | 8 | `GetMatchSkillRank` (`_neighbors_skill.go:119,132`) | `nil, nil` `nolint:nilerr` | idem |
  | 9 | `GetMatchSharedCSRs` (`_neighbors_skill.go:183,189,222`) | `nil, nil` ; « table absente : match non-ranked ou avant migration » = absence légitime à distinguer | idem |
  | 10 | `GetMatchBulkMedals` (`_medals.go:193,199`) | `nil, nil` `nolint:nilerr` | idem |
  | 11 | `GetMatchBulkWeaponKills` (`_weapons.go:140,146,199`) + `bulkWeaponKillsFromSource` (`_weapons_source.go:68`) | `nil, nil` ; `continue` sur ligne illisible (`:199`) | idem |
  | 12 | `GetMatchObjectiveScore` (`_scoreboard.go:317,322,334`) | `0, nil` sur erreur de requête | idem |
  | 13 | `loadMatchObjectiveStats` (`_scoreboard.go:227-242`) | vue absente ou erreur -> WARN, section dégradée ; `continue` sur Scan | via `GetMatchScoreboard` |
  | 14 | `loadMatchBombStats` (`_scoreboard.go:358-375`) | idem (colonnes absentes) | via `GetMatchScoreboard` |
  | 15 | `GetPlayerAssistsModel` (`_extras.go:477,486`) | `nil, nil` ; pas appelée par un chargeur de la vue (`WithExpectedAssists`), à écarter ou à traiter | à grepper |
  | 16 | `GetPlayerMatchStats` (`:377`), `GetMatchEnrichment` (`:411,420`) | renvoient une structure vide : à vérifier si c'est `sql.ErrNoRows` (absence) ou toute erreur | à grepper |
  | 17 | lignes illisibles : `scanAssistPairs` (`_assist_pairs.go:180`), `lookupMedalMeta`, `lookupLabelsByID` (`if err == nil`, `continue`) | ligne ignorée sans compteur | internes |
  Dans le service, hors contrat : `eventsRepo.Load` (`g.Go` manuel, WARN, repli sur la conversion de `events`), `loadAwardsForScoreboard`
  (`match_view_data_loaders.go` ~600-610, WARN si pas `ErrCapabilityNotSupported`), `friendsExtras` (`~432`, fan-out, résultat sans erreur dans
  la signature `port.FriendsExtrasResolver` : il faudrait changer le port).
  Déjà corrects (erreur remontée) : `GetMatchScoreboard`, `GetMatchMedals`, `GetMatchEncounters`, `GetMatchEncounterStats`,
  `GetMatchEncounterAssists`.
  **Avis** : l'étape dépasse largement le plafond de 12 (14 méthodes de la vue + 3 sites du service + 1 changement de port), chaque
  méthode demande l'audit de ses autres appelants (`GetHistoryForAvg*`, `GetMatchKVPairs`, `GetMatchSharedCSRs`... sont partagées avec
  d'autres pages) et la distinction « table absente » (au moins 5 cas). Proposition de découpage : (A) sources du bandeau et du kill
  feed (#1-#4, #12-#14, + awards/eventsRepo) ; (B) restes (#5-#11, #15-#17) et le port `FriendsExtrasResolver`.
- 2026-10-15 (E4.8a, exécutant Sonnet) : lot A fait, gate vert. Aucun `persist`/`sync`. Aucune ligne nette ajoutée à
  `match_view_data_loaders.go` (699 -> 668 L).
  - **Méthodes traitées (erreur REMONTÉE)** : #1 `GetMatchEvents` (lecteur et Q21 ; `lireEvenementsDuMatch` ne renvoie plus `ok`), #2
    `GetMatchKillSources`, #3 `GetMatchKillAssists`, #4 `GetMatchKVPairs` (lecteur et requête), #12 `GetMatchObjectiveScore`, #13
    `loadMatchObjectiveStats`, #14 `loadMatchBombStats` (journal passé de WARN à ERROR). Service : `eventsRepo.Load` (code déplacé dans
    `loadCanonicalEvents`, `match_view_source_failures.go`) dit `section_unavailable` ; `events` garde `events_unavailable`.
  - **Autres appelants audités (grep du dépôt)** : chacune de ces méthodes n'a qu'UN appelant de production, le chargeur correspondant de
    `loadMatchViewDataParallel` (qui classe l'erreur : E4.7a / dédiée) ; `GetMatchScoreboard` idem. Les autres occurrences sont des
    tests (adaptés) et le `noopMatchViewRepo` de `port`. `GetMatchEvents` homonymes (`MatchEventsService`, Halo 5) sont d'autres
    méthodes, non concernées. Aucun appelant ne comptait sur le `nil, nil`.
  - **Mécanisme de « section secondaire en panne »** : `domain.SectionDegradedError` (`domain/section_degraded.go`, `IsSectionDegraded`) :
    un lecteur rend sa donnée valide ET cette erreur (ici `GetMatchScoreboard` : lignes servies, stats d'objectif / d'Assaut en panne).
    `sourceFailures.goLoad` l'envoie dans le code générique au lieu de `scoreboard_unavailable`. Contrat de port documenté
    (`port/repository.go`). RÉUTILISABLE par le lot B.
  - **Absence légitime, distinguée EXPLICITEMENT (pas par un nil)** : (1) `GetMatchObjectiveScore` : table `personal_score_awards` du
    joueur absente = 0, constatée par sonde du catalogue (`information_schema`), toute défaillance de la sonde ou de la lecture
    remonte ; (2) `loadMatchBombStats` : `match_bomb_stats_latest` absente du lecteur = absence constatée par `relationExists`
    (`platform/duckdb/relation_probe.go`, helper neuf), WARN, pas d'erreur. MOTIF : le lecteur SNAPSHOT n'exporte PAS
    `match_bomb_stats` (`ops/snapshot_export.go` : ni `sharedSnapshotTables` ni `sharedSnapshotMatchKeyedRaw`) ; sans la sonde, toute
    vue d'un match servi du snapshot sur un titre à `film.bomb_stats` aurait été marquée en panne. Pour tous les autres : kill feed
    (`match_kill_events_latest`), objectifs (`match_objective_stats_latest`), `highlight_events` existent dans les deux titres (vérifié sur
    une copie de `halo_5/shared_matches_v2.duckdb` : table et vue présentes) et dans le snapshot : leur absence est une base non migrée
    = panne, qui remonte ; une relation vide rend « pas de donnée » sans erreur (test).
  - **Tests, rouge puis vert** (rouge obtenu en remettant temporairement la version HEAD des trois fichiers de repo, puis restaurée ;
    `platform/duckdb/match_view_repo_erreurs_remontees_test.go`, tag `integration`) : `GetMatchEvents` (vue `highlight_events` retirée),
    les trois lectures du kill feed (vue `match_kill_events_latest` retirée), `GetMatchObjectiveScore` (absence -> 0 sans erreur ;
    table présente sans vue `_latest` -> erreur), stats d'Assaut (vue de schéma incompatible -> `SectionDegradedError` + ERROR + lignes
    servies) ; test G1 adapté (`match_view_scoreboard_objective_degrade_test.go` : lignes servies ET `SectionDegradedError` ET ERROR).
    Verts avant et après (garde) : relation vide sans erreur ; vue d'Assaut absente -> pas d'erreur. Service (`match_view_unavailable_test.go`,
    rouge par mutation) : section secondaire du tableau -> `section_unavailable` et tableau servi, `eventsRepo.Load` en échec dit /
    capability non supportée non dite, kill sources / assistances / score d'objectif en panne -> `section_unavailable`. Bout en bout
    service + HTTP (`handlers/match_view_unavailable_e2e_test.go`, vrai service, repo factice) : panne d'une source -> 200 +
    `section_unavailable` + `no-store`, puis rétablie -> vue complète sans en-tête. Pas de test e2e à travers un vrai DuckDB jusqu'au HTTP :
    `service` dépend de `platform/duckdb` (cycle d'import pour un test interne) ; la chaîne est couverte maillon par maillon.
  - **Fixtures corrigées (masquées par l'ancien `nil, nil`)** : `shared.highlight_events` du fixture `player_repos_test.go` n'avait pas la
    colonne `raw_json` lue par Q21 (`TestMatchViewRepo_GetMatchEvents_Empty` passait en avalant un Binder Error) ; les fixtures
    `match_kill_events_latest` des tests legacy n'ont pas les colonnes de Q21b/Q21c (`publishable`...) : le test d'état vide utilise le schéma
    de production (`migration.EnsureMatchKillEvents`).
  - **Awards `[~]`** : `loadAwardsForScoreboard` et `WithAwardsRepo` n'ont AUCUN appelant de production (ni la vue ni le câblage) : ils ne
    peuvent pas dégrader la vue ; code mort (règle n°7), non modifié ici, voir section 3.
  - **Gate** : `go vet` (+ `-tags=integration` sur `platform/duckdb`) sans diagnostic ; `go test -count=1` sur `domain`, `port`, `service`,
    `platform/duckdb`, `api/...`, `archlint` : ok ; `go test -count=1 -tags=integration ./internal/platform/duckdb/` ok. Web non touché
    (pas de `make check-types` / `test-web`), contrat OpenAPI inchangé (golden `internal/api` vert).
  - **POUR L'EXÉCUTANT DU LOT B (E4.8b), tout ce qu'il doit savoir** :
    1. Méthode : une par une, grepper TOUS les appelants (`grep -rn "\bNom(" --include=*.go apps/go-api`) AVANT de changer le
       comportement ; si un autre appelant comptait sur `nil, nil`, l'adapter (log ERROR avant dégradation, jamais un nouvel avalement).
       Beaucoup sont partagées : `GetHistoryForAvg`/`Bulk` (`_extras.go:~346-420`, aussi appelées par d'autres pages/services),
       `GetMatchSharedCSRs`, `GetMatchSkillRank`, `GetMatchBulkMedals`/`BulkWeaponKills` (+ `bulkWeaponKillsFromSource`,
       `_weapons_source.go:68`), `GetMatchMedia` (`_extras.go:~262`), `GetMatchExpectedStats`, `GetPlayerAssistsModel`.
    2. Le service classe tout seul : une erreur d'un chargeur `goLoadSection` devient `section_unavailable`, sans rien changer côté
       service (sauf `friendsExtras`, voir 5). Section secondaire d'une lecture qui rend sa donnée principale : renvoyer la donnée ET
       `&domain.SectionDegradedError{Section: ..., Err: errors.Join(...)}` (modèle : `GetMatchScoreboard`, `loadMatchObjectiveStats`).
    3. « Absence légitime » : jamais un `nil` déduit d'une erreur. Soit sonde du catalogue (`relationExists(ctx, db, nom)`,
       `platform/duckdb/relation_probe.go`), soit `games.ErrCapabilityNotSupported` (le service la traite en absence). Candidats : `GetMatchSharedCSRs`
       (« table absente : match non classé » = absence constatée par sonde, pas toute erreur), `GetMatchExpectedStats` (mélange absence et
       panne : séparer `sql.ErrNoRows` — match ou joueur sans ligne = absence — d'une erreur de requête/colonne absente), `GetMatchSkillRank`
       (`resolveMatchRatingType` : `if err == nil`), `GetPlayerAssistsModel`. ATTENTION au lecteur SNAPSHOT
       (`ops/snapshot_export.go`/`snapshot_read.go`) : il ne porte que `match_registry, match_participants, medals_earned, highlight_events,
       killer_victim_pairs, match_csrs, match_objective_stats, match_kill_events` (+ vues `_latest` reconstruites) : toute lecture d'une
       autre relation par `sharedRead()` échoue sur un match servi du snapshot (découverte E4.8a : `match_bomb_stats`). Avant de faire
       remonter une erreur de relation, vérifier qu'elle est dans le snapshot ; sinon sonde + absence constatée, et le noter en section 3.
    4. Fixtures : un test qui passait en avalant une erreur peut cacher un fixture incomplet (vu deux fois en E4.8a) : corriger le fixture,
       pas le code. Les tests `integration` du paquet `platform/duckdb` durent ~4 min (`-tags=integration`).
    5. `FriendsExtrasResolver` (`port`, appelé dans `buildMatchViewFromData`, `match_view_data_loaders.go` ~419-432, APRÈS le
       `g.Wait` donc hors `sourceFailures`) : ajouter une erreur au port (interne, autorisé), la classer `section_unavailable` via
       `d.failures.failSection` (nil-safe, gère la capability) AVANT que `matchPartialReasons(d)` ne soit calculé, adapter l'implémentation
       (`wire.buildFriendsExtrasResolver`, fan-out : une panne d'une DB amie = erreur de section, pas une absence) et ses tests. Aucune ligne
       nette sur `match_view_data_loaders.go` (668 L) : logique dans `match_view_source_failures.go` ou un fichier voisin.
    6. Les 3 sites « ligne ignorée sans compteur » (#17 : `scanAssistPairs:180`, `lookupMedalMeta:103`, `lookupLabelsByID:88`) et le `continue`
       de `GetMatchBulkWeaponKills:199` : une ligne illisible est une défaillance (ERROR + erreur remontée ou `SectionDegradedError`), pas un
       `continue` muet. `GetPlayerMatchStats`/`GetMatchEnrichment` (#16) : vérifier que la structure vide n'est rendue que pour `sql.ErrNoRows`.
    7. Gate habituel + `go test -count=1 -tags=integration ./internal/platform/duckdb/` ; mutation rouge par méthode (remettre le fichier HEAD
       le temps de lancer les tests neufs). Puis E4.8c (garde-rail archlint) avec allowlist vide.

- 2026-10-15 (E4.8b + E4.8c, exécutant Sonnet) : lot B et garde-rail faits, gate vert. Aucun `persist`/`sync`. `match_view_data_loaders.go` 668 -> 601 L
  (aucune ligne nette ajoutée : le bloc des amis est sorti dans `service/match_view_friends_extras.go`).
  - **Méthodes traitées (erreur REMONTÉE)** : `GetMatchMedia`, `GetMatchExpectedStats`, `GetHistoryForAvg` + `GetHistoryForAvgBulk`, `GetMatchSkillRank`
    (+ `resolveMatchRatingType` : (string, error)), `GetMatchSharedCSRs`, `GetMatchBulkMedals`, `GetMatchBulkWeaponKills` (+ la ligne illisible de
    `tallyMatchKillSources` : `continue` + ERROR devenu erreur remontée, car un compte d'armes tronqué sous-estimerait des frags sans le dire),
    `GetPlayerAssistsModel`, `GetPlayerMatchStats`, `GetMatchEnrichment` ; hors inventaire, relevés par le garde : `GetMatchNeighbors`,
    `GetMatchNeighborsFiltered`, `LookupMedalMetaByName`. Libellés : `lookupMedalMeta` -> (map, error), `lookupLabelsByID` -> (map, error),
    `lookupMedalCitationLabels` -> (map, error). `scanAssistPairs:180` : le `continue` est la ligne de PORTÉE seule (colonnes de paire NULL, état
    nominal « mesuré, zéro assistant »), pas une ligne illisible : `[~]`, aucun changement. Port `FriendsExtrasResolver` : `(map, error)`.
  - **Mécanismes de l'absence légitime (aucun nil déduit d'une erreur)** : (1) `sql.ErrNoRows` testé AVANT `err != nil` (joueur sans ligne : stats,
    stats attendues, enrichissement, rang, modèle d'assists, voisins hors scope ; match absent du registre : repli sur le type brut du rang) ;
    (2) sonde `relationExists` : `match_csrs_latest` (lecteur sans CSR), `v_weapon_kills` (relation PROPRE AU TITRE, absente du fichier d'un titre à
    décodeur de film), `citation_mappings` (référentiel HINF seul, absent de la metadata Halo 5 : `lookupMedalCitationLabels` sonde avant de lire) ;
    (3) base sociale non configurée (`SharedSocial == nil`) pour les médias ; (4) amis : xuid sans profil lisible = ignoré (`friendProfilesByXUID`
    écarte le joueur courant et les `auth_only`). Vérifié snapshot : `match_participants`, `match_registry`, `medals_earned`, `match_csrs`,
    `match_kill_events` y sont ; `v_weapon_kills` est title-owned (`sharedSnapshotTitleOwnedRaw`) d'où la sonde. Relations de la player DB
    (`match_skill_rank_latest`, `player_match_enrichment_latest`, `player_assists_model`) : le target player de Halo 5 hérite du schéma HINF
    complet (`steps_player_test.go`), leur absence est donc une base non migrée = panne.
  - **Section secondaire** : libellés de médailles illisibles -> `GetMatchMedals` / `GetMatchBulkMedals` rendent les médailles (libellé replié sur
    l'identifiant) ET `*domain.SectionDegradedError` (`medal_labels`) ; le service dit `section_unavailable`, pas `medals_unavailable`.
  - **Appelants adaptés** : un seul appelant de production par méthode du repo (le chargeur de la vue) SAUF : `GetMatchSkillRank` /
    `GetMatchEnrichment` / `GetPlayerAssistsModel` (`friends_extras_loader.go`, désormais erreurs jointes et remontées au service),
    `GetPlayerAssistsModel` (`match_view_builders_summary.go:126` : ERROR avant le repli populationnel, valeur de repli servie ; `expected_assists.go:66`
    et `teammates/teammates_squad_expected_fda.go:84` : WARN -> ERROR ; `platform/duckdb/squad_v2_adapter.go:291` renvoie déjà l'erreur),
    `GetHistoryForAvgBulk` (`match_view_data_loaders.go` : `if err == nil` muet -> `failSection` dans `match_view_friends_extras.go`),
    `GetMatchBulkWeaponKills` (`attachTopWeapons`, `match_view_repo_weapons_source.go` : WARN -> ERROR, arme favorite non calculée),
    `lookupMedalCitationLabels` (`home_repo_medals_citations.go`, `medal_definitions_repo.go` : ERROR puis libellés repliés, comportement voulu),
    `LookupMedalMetaByName` (`match_view_killfeed_medals.go` : WARN -> ERROR, le feed garde le nom brut), voisins (`service/match_view_service.go` :
    WARN -> ERROR, voisins vides inchangés). `friendsExtras` : erreur classée `section_unavailable` par `failSection` avant `matchPartialReasons`.
  - **E4.8c** : `archlint/match_view_repo_erreurs_remontees_test.go` (AST) : dans une branche `if <erreur> != nil` de `platform/duckdb/match_view_repo*.go`,
    interdit un `return` dont la dernière valeur est `nil` (fonction renvoyant `error`) et un `continue` sans `slog.*` ni compteur. Allowlist VIDE.
    Détecteur testé sur sources fabriquées (mauvaises : 2 manquements ; saines : 0). Mutation vérifiée puis retirée : `return nil, nil` réintroduit dans
    `GetHistoryForAvg` -> rouge (`match_view_repo_extras.go:361`) ; `continue` réintroduit dans `tallyMatchKillSources` -> rouge. Le garde a relevé
    13 sites avant correction, dont 3 hors inventaire.
  - **Tests rouge puis vert** : repo (`platform/duckdb/match_view_repo_erreurs_remontees_lot_b_test.go`, tag `integration`) : rouge obtenu en remettant
    à HEAD `match_view_repo.go`, `_extras.go`, `_neighbors_skill.go`, `_weapons_source.go` (10 tests en échec), puis par mutation ciblée des retours
    d'erreur de `_weapons.go`, `_medals.go`, `medal_citation_fallback.go` (médailles, libellés, citations, armes SQL), tout restauré. Service
    (`match_view_unavailable_lot_b_test.go`) : mutation des deux `failSection` de `match_view_friends_extras.go` -> 2 tests rouges ; journal ERROR du
    modèle d'assists -> rouge par mutation du niveau. Loader des amis : tests réécrits (le résolveur remonte l'ouverture ratée, une absence n'est pas une
    panne) ; rouge par non-compilation sur l'ancienne signature. Bout en bout service + HTTP (`handlers/match_view_unavailable_lot_b_e2e_test.go`) : 8 sources
    du lot B (rang, CSR partagés, médias, stats attendues, médailles, armes, historique, amis) -> 200 + `section_unavailable` + `no-store`, puis rétablie ->
    vue complète sans en-tête. Un e2e à travers un vrai DuckDB jusqu'au HTTP reste impossible (cycle d'import `service` <-> `platform/duckdb` pour un test
    interne au paquet duckdb) : chaîne couverte maillon par maillon, comme au lot A. Wire : `TestFriendProfilesByXUID`.
  - **Fixtures corrigées (masquées par l'ancien `nil, nil`)** : le seed `seedPlayerSchema` n'avait pas `match_participants.backfill_bits` (lue par Q17 :
    `TestMatchViewRepo_GetPlayerMatchStats_NotFound` passait en avalant un Binder Error). Corrigé selon la recette des cliquets de seed : colonne ajoutée au
    seed, `Q17PlayerMatchStats` enrôlée dans `seedParityCarte`, entrée `backfill_bits` retirée de `ratchetColonnesHorsPerimetre`.
  - **Gate** : `go vet` (+ `-tags=integration` sur `platform/duckdb`) sans diagnostic ; `go test -count=1` sur `domain`, `port`, `service/...`,
    `platform/duckdb`, `api/...`, `archlint` : ok ; `go test -count=1 -tags=integration ./internal/platform/duckdb/` ok ; `golangci-lint --new-from-rev=HEAD`
    sur les paquets touchés : 0 issue. Web non touché, contrat OpenAPI inchangé (golden `internal/api` vert) : pas de `make check-types` / `test-web`.
  - **E4 CLOSE** : E4.1 à E4.8c statués `[x]` ; recette navigateur de la vue match `[!]` (voir plus haut : le serveur local sert `feat/v75`), reportée au
    gate visuel de l'utilisateur.
- 2026-10-15 (E5.1, exécutant Sonnet, LECTURE SEULE, aucun code modifié) : inventaire de `participants_loaded`, `player_count` et du bit 512
  de `backfill_completed`. Méthode : copies locales (`data/titles/{slug}/warehouse/shared_matches_v2.duckdb` copiées dans le scratchpad) et copies
  prod fournies par le superviseur (`e5_prod/`, 20:17), interrogées par `cmd/diag_q` compilé une fois (lecture seule) ; aucune base vivante ouverte.
  Prod HI = 2 014 matchs, local HI = 9 230 (7 216 de plus, tous Go-era), H5 = 3 032 identiques en local et en prod. **Le bit 512 vit dans
  `match_registry.backfill_completed` (base PARTAGÉE), pas dans une player DB** (`sync/writes.go:289-291` ; à ne pas confondre avec `PBitMedals`
  = 512 de `match_participants.backfill_bits`, `backfill_flags.go:39`). Colonne `participants_loaded` : ABSENTE du DDL de migration
  `create_base_shared_schema` (`steps_shared_core.go:33-70`) et des bases H5 ; présente seulement dans `sync/schema.go:207` (HI).
  - **E5.1-a — écrivains (par donnée x titre)**

    | donnée | HI (moteur de sync Go) | H5 | import OpenSpartan | autres |
    |---|---|---|---|---|
    | `participants_loaded` | AUCUN. Le DDL (`sync/schema.go:207`) est là depuis `b58cec2d4` (2026-04-15) ; `persistMatchRegistry` (`persist/shared_persister.go:155-200`) ne la liste pas, donc DEFAULT FALSE | colonne inexistante | `mapper.go:69` pose `ParticipantsLoaded: true` sur le type du MAPPER (`mapper/rows.go:39`) ; `toSyncRegistry` (`service/openspartan_import_adapters.go:41-72`) ne la recopie pas et `domain.MatchRegistryRow` n'a pas ce champ : la valeur meurt avant l'INSERT | fixture `sync/testutil/fixture.go:65` ; aucune migration, aucun backfill, aucun cmd ne l'écrit |
    | `player_count` | AUCUN (`domain.MatchRegistryRow.PlayerCount` reste 0 ; colonne listée `shared_persister.go:168`, valeur `:198`, donc 0) | `games/halo_5/ingest/collect.go:44-47` : `len(carnage.PlayerStats)` (`halo_5/capture.go:342-347`), ajouté par `955fac1f0` (2026-06-26) ; aucune ligne des données n'exerce ce code (voir comptes) | `mapper.go:41` `reg.PlayerCount = len(parts)` sur le type du mapper (`rows.go:45`) ; PERDU par `toSyncRegistry` (idem) | démo : `ops/seed_demo_synthetic_shared.go:115-121` (taille du roster fabriqué) ; migration d'ajout `steps_shared_core.go:600-604` ; DDL `steps_shared_core.go:60` et `sync/schema.go:213` |
    | bit 512 | `sync/collect.go:329-331` (`applyCompletionBitsToBatch`, appelé `collect.go:244`) : posé si `len(batch.Shared.Participants) > 0`, avant le Submit (INSERT-only) | NON posé (`backfill_completed` NULL sur les 3 032 lignes ; `ingest/collect.go` ne l'a jamais posé ; aucun lecteur H5) | NON posé : `toSyncRegistry` ne porte pas `BackfillCompleted` (le mapper pose `1`, qui est le bit MÉDAILLES, `mapper.go:68`), donc NULL à l'INSERT ; jamais exercé dans les données | `cmd/levelup/cmd_reset_bitmasks.go:155-166` (UPDATE `backfill_completed |= 512` pour les matchs qui ont des participants ; sous-commande `reset-bitmasks --apply`, câblée `main.go:125`) |

  - **E5.1-b — lecteurs de PRODUCTION**
    - `participants_loaded` : **aucun**. Recherche exhaustive Go (SQL en chaînes compris), `openapi.yaml`, web, TOML : le seul autre texte est `docs/archive`.
    - `player_count` : **aucun oracle d'intégrité n'existe**. Le « fix #10 » (`955fac1f0`) a écrit la colonne et déclaré « re-backfill = ops data séparée » ; aucun code ne
      compare `player_count` aux lignes de `match_participants` (grep exhaustif). Lecteurs réels : (1) vue `mv_player_matches` (`migration/steps_shared.go:275`) et
      `v_match_full` (`SELECT mr.*`) = RECOPIE, aucun lecteur nommé de la colonne derrière ; (2) export Parquet du snapshot (`ops/snapshot_export.go:197`,
      `SELECT * FROM match_registry`) = recopie ; (3) `cmd/mapfond-inventaire/lecture.go:114` : colonne TSV `joueurs_max` d'un inventaire de cartes hors ligne (informatif,
      fausse aujourd'hui) ; (4) `cmd/diag_expected_assists/main.go:83` : corrélation de diagnostic jetable ; (5) démo : `persist/demo_seed_columns.go:36`
      (liste de parité avec le persister, ratchet `demo_seed_columns_test.go:70`) et `ops/seed_demo_synthetic_shared.go:120` = écrivains, pas lecteurs.
    - bit 512 : **un lecteur de production, HI seul** : `sync/backfill.go:348-353` (`doneGuard("participants")`, `backfill.go:174-182`) quand `scope.Participants` est vrai,
      condition jointe en `OR` aux autres (`backfill.go:387`). `scope.Participants` n'est vrai que par `AllData` (`scope.go:137`), et `handlers/backfill.go:156-166,209` pose `AllData`
      quand la requête `POST /backfill/start` n'a AUCUN scope explicite (cas du bouton admin, `web/src/features/settings/queries.ts:112`). La liste `missing`
      (`service/backfill_orchestrator.go:77`) nourrit : `BackfillEventsForMatches(missing, ...)` (`:192-197`) puis `ReplayHighlightEventsForMatches` : pour CHAQUE id, `clearEventsLoaded`
      (UPDATE `events_loaded=FALSE`, `sync/events_replay.go:147,228-233`) puis re-téléchargement du film ; et `BackfillPersonalScoreAwardsForMatches(missing)` (`:244-253`) : un
      `GetMatchStats` par id (`sync/backfill_personal_scores.go:53-57`). Aucun exécuteur « participants » n'existe dans l'orchestrateur : rien ne pose jamais le bit 512 après coup.
      Hors ça : `cmd/levelup/cmd_reset_bitmasks.go` (réparation), `cmd/diag_bitmask_coverage` (diagnostic). `sync.FindMatchesMissingParticipantBits` et
      `NewBackfillFlagSet` (`backfill_cli.go:27`, le flag `--participants`) n'ont aucun appelant de production. Le readiness du snapshot (`snapshot_readiness.go:125`) ne lit PAS le bit 512
      (seulement armes, film absent, PVE).

  - **E5.1-c — date et cause**
    - `participants_loaded` et `player_count` (HI) : jamais écrits par le code Go. Dernière ligne les portant : première synchro du 2026-04-06 22:03 (prod) ; première ligne Go : 2026-05-06.
      Le DDL a été porté de l'ère Python au commit `b58cec2d4` (2026-04-15, « moteur sync delta Go ») sans écrivain (`git log -S participants_loaded` : ce commit et une correction de
      fixture du 2026-05-06 ; `-S ParticipantsLoaded` : le mapper `adedcc6a3` du 2026-05-18, valeur perdue à la recopie). Ce n'est pas une régression du code Go : c'est une
      donnée qu'il n'a jamais produite. L'écrivain Python n'est pas dans le dépôt (déduit des données : 1 382 lignes `participants_loaded` vrai, toutes synchronisées avant le 2026-04-07).
    - `player_count` (H5) : écrivain ajouté le 2026-06-26 (`955fac1f0`), APRÈS l'ingestion de la totalité du parc H5 (créé le 2026-06-23) ; aucun match H5 ingéré depuis.
    - bit 512 (HI) : vraie régression, FERMÉE. Le chemin batch Collect->Persist est devenu le défaut le 2026-05-24 (`82d0aa1a2`) sans poser les bits de complétude ; `ddfd9e124`
      (2026-05-31, « backfill_completed fiable sur le chemin batch », qui cite le re-traitement infini des backfills) les pose. Preuve dans les données : les 157 matchs sans
      bit ont tous `first_sync_at` entre 2026-05-25 11:59 et 2026-05-30 11:46 (78 le 25, 77 le 26, 2 le 30) ; dès le 2026-06-03, 11 sur 11 portent le bit. Mêmes 157 en local et en prod.
      Les mêmes 157 n'ont pas non plus le bit skill (4) : même cause, hors périmètre (découverte).

  - **E5.1-d — comptes** (jointure `match_registry` x `COUNT(*)` de `match_participants` par match ; mois = `COALESCE(start_time_utc, start_time AT TIME ZONE 'UTC')`)

    | | HI local | HI prod | H5 (local = prod) |
    |---|---|---|---|
    | matchs au registre | 9 230 | 2 014 | 3 032 |
    | sans aucun participant | 0 | 0 | 0 |
    | `participants_loaded` vrai / faux | 1 382 / 7 848 | 1 382 / 632 | colonne absente |
    | `player_count` > 0 / = 0 / NULL | 1 544 / 7 686 / 0 | 1 544 / 470 / 0 | 0 / 0 / 3 032 |
    | `player_count` = nb réel de participants | 13 | 13 | sans objet |
    | `player_count` < nb réel (sur les > 0) | 1 531 (aucun >) | 1 531 (aucun >) | sans objet |
    | bit 512 posé / non posé | 9 073 / 157 | 1 857 / 157 | 0 / 3 032 |
    | `backfill_completed` NULL | 0 | 0 | 3 032 |
    | index sur `player_count` | `idx_registry_player_count` (legacy, aucun code ne le crée) | idem | aucun |

    Croisements : `participants_loaded` vrai implique `player_count` > 0 (1 382 sur 1 382), jamais l'inverse (162 lignes `player_count` de 2 à 6 et `participants_loaded` faux, 2026-02 / 03).
    Même posé, `player_count` ne vaut le roster réel que 13 fois sur 1 544 (rosters réels de 4 à 36) : les valeurs historiques ne sont PAS la taille du roster API, donc inutilisables
    comme oracle même conservées. Prod HI par mois : `participants_loaded` vrai de 2021-11 à 2026-04 (2026-01 : 179/179, 2026-02 : 33/173, 2026-03 : 142/184, 2026-04 : 12/86),
    ZÉRO à partir de 2026-05 (2026-05 : 0/86 ; 2026-06 : 0/65 ; 2026-07 : 0/174 ; 2026-09 : 0/32) ; `player_count` > 0 suit la même coupure (+ les 162 lignes de février-mars). Bit 512 : posé
    partout sauf les 157 de la fenêtre du 25 au 30 mai (répartis sur les mois de début de match 2021-11, 2026-02 à 2026-05). Local : mêmes coupures (les 7 216 matchs locaux en plus sont Go-era :
    `participants_loaded` faux, `player_count` 0, bit posé hors fenêtre). H5 par année (2015 : 364, 2016 : 605, 2017 : 308, 2018 : 1 149, 2019 : 422, 2020 : 103, 2021 : 65, 2023 : 16) :
    `player_count` NULL et bit 512 non posé partout, rosters réels de 1 à 21.
    Requêtes à rejouer (le bit vit dans la base partagée : aucune player DB à ouvrir) :
    `WITH p AS (SELECT match_id, count(*) n FROM match_participants GROUP BY 1) SELECT COALESCE(mr.participants_loaded::VARCHAR,'NULL'), (COALESCE(mr.backfill_completed,0)&512)>0, p.n IS NOT NULL, count(*), sum(CASE WHEN mr.player_count=p.n THEN 1 ELSE 0 END) FROM match_registry mr LEFT JOIN p USING(match_id) GROUP BY 1,2,3`
    (sur H5, sans `participants_loaded`) ; mois : `strftime(COALESCE(mr.start_time_utc, mr.start_time AT TIME ZONE 'UTC'),'%Y-%m')` ; fenêtre : `SELECT date_trunc('day', first_sync_at), (COALESCE(backfill_completed,0)&512)>0, count(*) FROM match_registry WHERE first_sync_at >= '2026-05-20' GROUP BY 1,2`.

  - **E5.1-e — coût du statu quo**
    - `participants_loaded` : nul (aucun lecteur). `player_count` : nul côté produit (aucun oracle) ; trompe seulement la colonne `joueurs_max` d'un outil hors ligne ; la promesse « oracle d'intégrité
      du roster » de `domain/match_rows.go:74-79` et de `halo_5/mapping_carnage.go:57` est une affirmation sans lecteur.
    - bit 512 : le bouton admin « backfill » sans scope explicite re-sélectionne à CHAQUE lancement les 157 matchs HI de la fenêtre (le bit n'est jamais posé après coup), donc jusqu'à 157 re-téléchargements
      de film avec remise à zéro de `events_loaded` (UPDATE) et 157 `GetMatchStats` pour les scores personnels, répartis par joueur concerné (prod : JGtm 77, Chocoboflor 70, XxDaemonGamerxX 10).
      Établi par lecture du code et par comptage ; aucun job lancé. Incertain : une part de ces 157 est peut-être déjà sélectionnée par une autre condition `OR` (médailles, assets...) qui dépend de la player DB.
  - **E5.1-f — recettes de réparation sans `UPDATE` sur `match_registry`**
    - `match_registry` n'est PAS append-only (absente de `tablesProtegees`, `no_art_patterns_test.go:20-24` ; pas de vue `_latest`) : la recette de l'ADR 0026 (`append_only_rebuild.go`) ne s'y applique pas
      (clé fonctionnelle = PK `match_id`, lue partout, 2 vues dépendantes). Elle EST « critique » au sens des règles : table « match-of-record » de la base partagée (écriture per-match = persist INSERT-only,
      CLAUDE.md ART n°1), listée dans `criticalMatchTables` (`no_art_patterns_test.go:466-475`, tripwire contre `UPDATE ... FROM (VALUES ...)`) et dans `guardedSharedTables` (`shared_write_guard_test.go:31-36`,
      écrivains hors persist en allowlist). Des `UPDATE` row-by-row sérialisés sous lease y existent en production (`persist/events_completion_persister.go:179,212,333`, `sync/killcollector/registry_flags.go:53`,
      `sync/pve.go:321`, `persist/registry_names_persister.go`, `persist/t0_film_persister.go:66`) : sur des colonnes NON indexées. `idx_registry_player_count` fait de `player_count` une colonne INDEXÉE : un UPDATE y est
      le vecteur nommé par l'ADR 0026 (cf. `registry_names_persister.go:25`, `steps_shared_recompute_mode_category.go:11`).
    - Recette applicable, éprouvée sur `match_registry` : **échange par reconstruction `swapTableTx`** (`internal/migration/table_swap.go:62`), telle qu'employée par `shared_recompute_mode_category_v1`
      (2026-10-09, `steps_shared_recompute_mode_category.go`) : table reconstruite à côté avec le DDL exact et UNE colonne remplacée par une expression, échange en transaction, index reposés (ou
      retirés), gardes avant COMMIT (cardinalité, DDL, empreinte des vues dépendantes hors colonne). Coût : relecture de la table entière (2 014 lignes en prod HI, 9 230 en local) ; 7 index + `mv_player_matches` +
      `v_match_full` à reposer. Elle répare `backfill_completed` (`backfill_completed | 512` pour les matchs qui ont des participants) comme elle retirerait `player_count` / `participants_loaded`.
      Variantes pour retirer une colonne : `migration.AlterColumnTypeIfNeeded` / `steps_shared_social_media_files_drop_liked.go` (index relevés, DROP, DROP COLUMN, recréés) et
      `widen_match_registry_team_scores` (précédent d'ALTER sur `match_registry` avec index secondaires). Alternatives sanctionnées par le code mais en `UPDATE` : `levelup reset-bitmasks --apply` (UPDATE set-based d'une
      colonne non indexée, hors des gardes car sous `cmd/`) ; un persister row-by-row dans `internal/persist` calqué sur `events_completion_persister.go`. D-13 : ces deux dernières sont des `UPDATE` sur `match_registry`, donc décision du superviseur.
  - **E5.1 STATUÉ `[x]`.** STOP : verdict du superviseur attendu avant E5.2 (recommandations au rapport de l'exécutant).
- 2026-10-15 (E5.2, exécutant) : `participants_loaded` retirée du code. DDL de secours `sync/schema.go` (ligne de colonne), fixture d'intégration
  `sync/testutil/fixture.go`, champ `ParticipantsLoaded` du type du mapper OpenSpartan (`mapper/rows.go`) et son affectation (`mapper/mapper.go`), assertion
  de `mapper_test.go` (garde `EventsLoaded`/`MedalsLoaded`). La DDL de migration `create_base_shared_schema` ne la portait pas. Grep Go = 0 ; hors Go : seulement
  `docs/archive`, le journal et deux TSV de recherche archivés (`.ai/V7.5/replay2d/registre_film/oracle_lotA*.tsv`, en-têtes d'export). Colonne physique : E5.5.
  `go vet ./internal/openspartan/...`, `go vet -tags=integration ./internal/sync/testutil/`, `go test -count=1 ./internal/openspartan/...` verts. **E5.2 STATUÉ `[x]`.**
- 2026-10-15 (E5.3, exécutant) : `player_count` retirée du code. Champ `domain.MatchRegistryRow.PlayerCount` ; écrivain H5 (`captureParticipants` ne rend
  plus le compte, paramètre `playerCount` de `ingest.CollectMatchBatch` retiré, test) ; INSERT `persist/shared_persister.go` et contrat
  `persist/demo_seed_columns.go` ; seeder démo `ops/seed_demo_synthetic_shared.go` ; champ et affectation du mapper OpenSpartan (sa valeur mourait déjà à la
  recopie) ; DDL de base `create_base_shared_schema` et DDL de secours `sync/schema.go` ; vue `mv_player_matches` (`migration/steps_shared.go`, `v_match_full`
  est un `SELECT mr.*`) ; colonne TSV `joueurs_max` de `cmd/mapfond-inventaire` ; `cmd/diag_expected_assists` corrèle désormais avec le roster RÉEL
  (`COUNT(*)` de `match_participants` par match). **Étape d'ajout `add_player_count_to_match_registry` SUPPRIMÉE** (registre HI + `canonicalOrder`) : le
  mécanisme le permet sans casser l'historique (précédent `shared_objective_score_v1`, créateur supprimé ; la ligne du ledger des bases existantes reste
  inerte, `runSteps` ne parcourt que les steps enregistrés ; `TestCanonicalCoversGlobalAndTitle` vert). Fixtures de test réalignées (5 DDL), entrées
  `player_count` retirées du gel `archlint/match_registry_ddl_types_test.go` (le gel rétrécit). Commentaires faux corrigés (`halo_5/mapping_carnage.go`,
  `cmd/h5-roster-topup`). Lecteur de snapshot : **rouge puis vert** — avec l'ancienne vue, un cut neuf (sans la colonne) échouait
  (`Binder Error: Table "mr" does not have a column named "player_count"`, `TestOpenSnapshotShared_Fidelity_integration`) ; nouveau test
  `TestOpenSnapshotShared_CutAncienAvecColonnesRetirees_integration` : un cut ANCIEN qui porte encore `player_count` et `participants_loaded` est servi
  (vues lisibles, `mv_player_matches` sans la colonne). Grep Go restant : ce test (voulu), l'alias de sortie retiré, et le constat daté du gel archlint
  (mesure du 2026-09-16, laissé tel quel). `go vet` (avec et sans tag) + `go test` persist, halo_5, halo_infinite/migrations, migration, openspartan, archlint,
  domain, ops (avec et sans tag), sync/snapshot verts. **E5.3 STATUÉ `[x]`.**
