# Plan — Lecture allégée du contexte de match pour la page de rejeu (2026-10-20)

Contrat d'exécution : skill `plan-execution` (ordre strict, gate par étape, statut de chaque item,
zéro fix hors périmètre). Statuts : `[x]` fait, `[~]` couvert ailleurs (référence), `[!]` non
traité (justification) ; aucune case vide à la clôture. Reprise de session : lire ce fichier (cases cochées) puis
`git log --oneline -10` dans le worktree.

## Objectif et critère de succès

La page de rejeu attend aujourd'hui la vue complète du match (`GET /players/{slug}/matches/{id}`) :
le serveur calcule tous les onglets (rencontres sur l'historique, Emprise, Isolement, radar,
citations, résumé, extras des amis) alors que le rejeu n'en lit que quatre morceaux. Une route
dédiée ne charge et ne construit que ces morceaux, avec LES MÊMES builders que la vue complète.

Succès :
1. Ce que la page de rejeu affiche est identique à aujourd'hui, en particulier la frise des frags
   et des morts de chaque joueur, ses médailles, le fil des éliminations (arme, assistant,
   victime), les fiches (noms, camps, K/D/A, médailles, entrées et sorties en cours de match),
   l'écran de fin, la piste des médias.
2. Prouvé par test : pour un même match, chaque morceau servi par la route allégée est égal
   (comparaison profonde) au morceau correspondant de la vue complète.
3. Mesure au navigateur (local, même match que la mesure du 2026-10-20) : la requête de contexte
   coûte nettement moins que la vue complète (1,3-1,5 s), objectif < 0,5 s.

## Ce que le rejeu lit de la vue du match (inventaire, à confirmer par le compilateur, étape 1)

| Morceau | Lecteurs | Sources serveur |
|---|---|---|
| `header` (champs : `t0_ms`, `start_time`, `playable_duration_seconds`, `outcome_code`, `score_mine`, `score_theirs`, `score_timeline_kind`, `mode_category`, `map_ui`, `mode_ui`, `playlist_label`, `start_time_label`, autres à confirmer) | `replayModel` (horloge, fenêtre, score final), route (fil d'Ariane, rappel), `ReplayTeams`, écran de fin, son de fin | meta (Q13), stats joueur (Q17), enrichment (Q18), scoreboard (Q12) + `apply*` de l'en-tête |
| `team_tab.scoreboard` (lignes ; champs à confirmer : `xuid`, `gamertag`, `is_me`, `is_bot`, `team_side`, `outcome`, `kills`, `deaths`, `assists`, `score`, `medals`, `rank`, `joined_in_progress`, `left_in_progress`, `first_joined_time`, `last_leave_time`, nom d'équipe) | identité, allégeance, marques, roster, fiches | scoreboard (Q12), bulk medals, bulk weapons, CSR, `applyTeamNames` |
| `combat_tab.highlight_events` (kills ET médailles) | **frise des frags/morts par joueur, médailles**, fil des éliminations | events (Q21), events canoniques, kv pairs (synthèse si events vides), kill sources (Q21b), kill assists (Q21c), recalage T0, `decorateKillFeed`, `decorateMedalEvents` |
| `media_tab` | piste Médias de la frise | media (Q24) + transformation chemins → URLs du handler (`WithMediaURLs`) |

## Décisions (tranchées)

- D1. Route : `GET /players/{player_slug}/matches/{match_id}/replay/context`, montée dans le groupe
  du rejeu (mêmes portes : `CapReplay`, `ReplayGate`, participation). Un titre sans rejeu ne
  l'appelle jamais (la page n'est pas montée).
- D2. Forme de la réponse : `{ header, scoreboard, highlight_events, media }` — chaque morceau a
  le TYPE EXACT de son homologue dans `MatchViewResponse` (`MatchViewHeader`, `[]MatchScoreboardRow`,
  `[]MatchHighlightEvent`, `MatchMediaTab`). Pas de type allégé côté Go : la parité est structurelle.
  Amendement (2026-10-20, décision superviseur sur la Découverte 2) : + `source_unavailable` (booléen,
  vrai quand une lecture de la liste fermée D4 a échoué), pour garder la politique de la vue
  (nouvel essai 30 s, fraîcheur nulle) ; politique commune dans `lib/analysis-pending/matchRefreshPolicy.ts`.
- D3. Zéro copie de logique : l'assemblage de l'en-tête (les `apply*`) et la préparation
  (recalage T0, expected assists si utile) sont EXTRAITS de `buildMatchViewFromData` en fonctions
  partagées appelées par les deux chemins. La vue complète garde un comportement identique.
- D4. Chargements de la route allégée = ceux dont dépend un des quatre morceaux, et eux seuls.
  Liste fermée à l'étape 2 à partir de l'inventaire typé de l'étape 1 ; tout chargement dont
  dépend un champ LU par le rejeu est conservé, même s'il coûte.
- D5. Côté web, la page de rejeu lit la route allégée (nouveau hook, clé dans `lib/query/keys.ts`) ;
  `useMatchView` n'est plus appelé par la page de rejeu. Pas de réutilisation opportuniste du
  cache de la vue complète (deux formes, deux clés).
- D6. Le modèle du rejeu et ses composants sont typés sur un type `ReplayMatchContext` (les
  quatre morceaux) au lieu de `MatchViewResponse` : le compilateur garantit qu'aucun lecteur ne
  dépend d'un champ que la route ne sert pas.

## Branche

Worktree dédié `../LevelUp-wt-replay-contexte`, branche `feat/replay-contexte` (depuis `feat/v75`,
préfixe `feat/` pour que la CI tourne). Fusion dans `feat/v75` sur décision de l'utilisateur :
une PR de release v8.0.1 est ouverte depuis `feat/v75`.

## Étapes

### Étape 1 — Inventaire typé côté web (moyen)
- [x] 1.1 Définir `ReplayMatchContext` (dans `lib/replay/`) : `header: MatchViewHeader`,
      `scoreboard: MatchScoreboardRow[]`, `highlight_events: MatchHighlightEvent[]`,
      `media: MatchMediaTab`.
- [x] 1.2 Remplacer `MatchViewResponse` par `ReplayMatchContext` dans les signatures de
      `features/match-replay/**` et `lib/replay/**` qui reçoivent la vue (`replayModel`,
      `useReplayModel`, médias, et tout autre consommateur trouvé par `grep -rn MatchViewResponse`).
      La route adapte provisoirement la vue complète vers ce type (fonction pure testée).
- [x] 1.3 Lister dans ce plan (section Inventaire) les champs d'en-tête et de ligne de scoreboard
      effectivement lus (grep des accès sur les valeurs typées + compilation).
- Gate : `cd apps/web && npx tsc -b --force` sans erreur ; `npx vitest run src/features/match-replay src/lib/replay src/routes` vert.
- Journal (2026-10-20) : `ReplayMatchContext` + adaptateur `replayContextFromMatchView`
  (`lib/replay/replayMatchContext.ts`, testé) ; `buildReplayModel` / `useReplayModel` et la route
  typés sur le contexte. Seuls `replayModel.ts` et `useReplayModel.ts` recevaient la vue entière ;
  les autres lecteurs prenaient déjà un morceau typé (`MatchScoreboardRow[]`, `Pick<MatchViewHeader, …>`,
  `MatchHighlightEvent[]`, `MatchMediaTab`). Gate : `tsc -b --force` exit 0 ; vitest 267 fichiers /
  3 708 tests verts (4 fichiers sautés, préexistants), exit 0.

### Étape 2 — Service et route Go (lourd)
- [x] 2.1 Domaine : `domain.MatchReplayContext` (D2).
- [x] 2.2 Extraire de `buildMatchViewFromData` les fonctions partagées (D3) ; vue complète inchangée.
- [x] 2.3 `MatchViewService.GetReplayContext` : meta bloquant + participation (comme `GetMatchView`),
      chargement parallèle RESTREINT (liste fermée D4, écrite ici avant de coder), assemblage
      par les fonctions partagées et les builders existants.
- [x] 2.4 Port `MatchViewService.GetReplayContext` ; handler (décode, appelle, applique la même
      transformation des médias que `GetMatchView`) ; montage D1 ; `openapi.yaml` régénéré
      (`make generate-types` côté web).
- [x] 2.5 Journal : `slog` sur l'échec de meta comme `GetMatchView` ; sections de durée nommées.
- Gate : `go build ./cmd/server` ; `go vet ./internal/service/ ./internal/api/...` ;
  `go test ./internal/service/ ./internal/api/...` vert.
- Journal (2026-10-20) : `domain/match_replay_context.go` (quatre morceaux aux types de la vue
  complète + `SourceUnavailable` non sérialisé, d'où `Cache-Control: no-store` comme la vue).
  Fonctions partagées extraites dans `service/match_view_shared_build.go` : `loadMatchMeta`
  (absence → 404, panne → 500, journal identique), `prepareMatchViewEvents` (recalage T0),
  `assembleMatchHeader` (favori + `buildMatchHeader` + tous les `apply*`), `isMatchFavorite`,
  `decorateHighlightEvents` (arme, assistant, victime, équipe, médailles) ; `highlightEventList`
  extrait de `buildCombatTabFull`. `GetReplayContext` + chargement restreint (liste D4) dans
  `service/match_view_replay_context.go` ; lignes par `buildTeamTabFull` (rencontres, paires et
  extras à nil) + `applyTeamNames`, médias par `buildMediaTab`. Handler
  `handlers/match_view_replay_context.go` (même `transformMatchMediaURLs`, erreurs par
  `matchViewServiceError`, désormais partagé avec `handleGetMatchView`) ; monté dans le groupe
  `CapReplay` + `ReplayGate` + `participated` (`server_apiv1.go`). Sections de durée
  `replay_context_meta` / `replay_context_assemble` + métrique `match_replay_context_get`.
  `openapi.yaml` régénéré (`go run ./cmd/openapi-gen`, +1 chemin, +1 schéma) puis
  `generated.ts` (`npm run generate-types`). Gate : `go build ./cmd/server` exit 0 ; `go vet`
  exit 0 ; `go test ./internal/service/ ./internal/api/...` exit 0 (6 paquets ok) ; vitest
  `src/lib/api` (gardes de contrat) 39/39 ; `node tools/lint-contract-ratchet.mjs` exit 0.

### Étape 3 — Preuves (moyen)
- [x] 3.1 Test de parité service (repo factice riche : kills avec arme et assistant, médailles,
      joueur entré et sorti en cours de match, bot, médias, T0 non nul, synthèse kv quand events
      vides) : `GetReplayContext` == morceaux correspondants de `GetMatchView` (`reflect.DeepEqual`).
- [x] 3.2 Test « pas de chargement lourd » : le repo factice compte les appels ; aucun de
      `GetMatchEncounters`, `GetMatchEncounterStats`, `GetMatchEncounterAssists`, `GetHistoryForAvg`,
      `GetMatchExpectedStats` (sauf si D4 l'exige), citations, Emprise, Isolement, friends extras.
- [x] 3.3 Test handler `httptest` : 200 + forme JSON ; 404 match inconnu ; participation refusée.
- Gate : tests 3.1-3.3 verts ; `go test -count=1 ./internal/service/ ./internal/api/...`.
- Journal (2026-10-20) : `service/match_view_replay_context_test.go` — repo factice qui compte
  ses appels et rend des copies (la vue complète modifie le scoreboard chargé en place).
  3.1 `TestGetReplayContext_PariteAvecLaVueComplete` : deux cas (kills dans les events ; events
  sans kill → synthèse kv), `reflect.DeepEqual` des QUATRE morceaux entiers, plus des témoins qui
  prouvent que la fixture exerce arme, assistant, médaille résolue, recalage T0 (15 s), countdown
  écarté, entrée/sortie en cours, bot, rang, médailles de ligne, médias. Mutation vérifiée : sans
  `decorateHighlightEvents`, le test rougit sur « événements ». `TestGetReplayContext_ChampsDesChargementsOmis` :
  avec un ami suivi et un modèle d'attendus, les lignes diffèrent (prouvé) et sont égales une
  fois retirés les sept champs de la liste D4. 3.2 `TestGetReplayContext_AucunChargementLourd` :
  chaque compteur est d'abord vu > 0 sur la vue complète, puis 0 sur le contexte (rencontres ×3,
  historique, attendus, modèle d'assists, paires d'assistance, score d'objectif, médailles Q14,
  citations, events canoniques, Isolement, feuille de l'Emprise, extras des amis). Erreurs
  (absence, non-participation avant chargement, panne) et contexte transitoire testés aussi.
  3.3 `handlers/match_view_replay_context_test.go` : 200 + exactement quatre clés + médias réécrits
  + pas de Cache-Control ; 404 `match_not_found`, 404 `match_not_participant`, 500
  `replay_context_error` ; `no-store` sur source en panne et analyse en cours. Gate : `go test
  -count=1 ./internal/service/ ./internal/api/...` exit 0 (6 paquets) ; `go test ./internal/archlint/
  ./internal/domain/ ./internal/port/` exit 0.

### Étape 4 — Branchement web (rapide)
- [x] 4.1 Hook `useMatchReplayContext` (clé `queryKeys.matchReplayContext`), page de rejeu
      branchée dessus ; adaptateur provisoire de l'étape 1.2 supprimé s'il n'a plus d'appelant.
- [x] 4.2 `replayLoadState` : `identityPending` lit la nouvelle requête.
- Gate : `npx tsc -b --force`, `npx eslint` sur les fichiers touchés, vitest `match-replay` + `lib/replay` + `routes` vert ; `grep -n useMatchView` dans la route de rejeu : aucun résultat.
- Journal (2026-10-20) : `useMatchReplayContext` dans `lib/replay/queries.ts` (clé
  `matchReplayContext`, classée title-scopée dans `keys.title-slug.guard.test.ts` ; fraîcheur
  nulle et rechargement 60 s pendant l'analyse du film, comme la vue complète). L'adaptateur
  `replayContextFromMatchView` n'a plus d'appelant : supprimé, remplacé par
  `normalizeReplayMatchContext` (tableaux `null` du transport → vides, testé) et
  `replayContextStaleTime`. La route ne lit plus `useMatchView` ; `identityPending` lit
  `isPending` du contexte ; commentaires de `replayLoadState` et de la route remis à jour.
  Gate : `tsc -b --force` exit 0 ; `eslint` (9 fichiers touchés) exit 0 ; vitest
  `match-replay` + `lib/replay` + `routes` + `lib/api` + `lib/query` + `match-view` : 331 fichiers
  / 4 305 tests verts, exit 0 (premier passage rouge : la nouvelle clé n'était pas classée par le
  garde title-slug, corrigé) ; `grep -n useMatchView` sur la route : aucun résultat (exit 1) ;
  `npx knip` exit 0 ; `lint-cross-feature-imports` exit 0.

### Étape 5 — Vérification et clôture (rapide)
- [x] 5.1 (superviseur) Mesure au navigateur (instance Chrome déjà connectée, une seule page) : durées des
      requêtes de la page, capture à l'affichage ; comparaison visuelle avec la capture d'avant
      (fiches, frise, médailles, fil).
- [x] 5.2 (superviseur) `adversarial-review` du diff (lot à risque : parité de données).
- Journal 5.1-5.2 (2026-10-21) : mesure au navigateur, local, match du 2026-10-20 (servi par le live),
  après fusion dans `feat/v75` (`6eb8ffae8`) : contexte 91 ms / 83 Kio (vue complète : 1,3-1,5 s / 117 Kio,
  4,9 s avant le bornage de Q23b) ; page affichée d'un bloc à 0,9 s après le chargement (3,5 s avec la vue
  complète bornée, 5,5 s avant). Capture identique à celle d'avant (en-tête, bandeau, pions, frise des frags,
  morts et médailles, fiches). Revue adversariale ronde 1 : relecteur parité/accès 0 constat recevable
  (14 conditions) ; relecteur couverture 6 constats, 5 corrigés (`7c8cd7596`, 15 mutations rouges), 1 consigné.
  Ronde 2 non lancée : les corrections ne touchent que des tests, chacune prouvée par mutation.
- [x] 5.3 `.ai/thought_log.md` ; commit(s) ; push `feat/replay-contexte` ; CI verte.
- Gate : CI de la branche verte (`gh run list --branch feat/replay-contexte`).
- Journal 5.3 (2026-10-20) : entrée `.ai/thought_log.md` ; un commit par étape (1 à 4) puis ce
  commit de clôture ; push de `feat/replay-contexte`. Verdict de la CI : relevé par l'exécutant
  après le push (rapport au superviseur), non réécrit ici pour ne pas relancer la CI.

## Inventaire (rempli à l'étape 1)

Méthode : grep des accès (`.champ`, `'champ'` dans un `Pick`) sur tous les fichiers non-test de
`features/match-replay/**`, `lib/replay/**`, la route de rejeu et les modules de `features/match-view`
qu'ils importent (`_momentum`, `xuidMeta`, `teamColor`, `teamSeriesColor`, `format`, `MatchHeader`),
puis lecture du contexte de chaque occurrence pour écarter les homonymes du document de rejeu
(`rank` des grenades, `kills`/`deaths` des pistes, `medals` d'une entrée du fil).

- **En-tête** (lu) : `t0_ms`, `start_time`, `playable_duration_seconds`, `outcome_code`,
  `score_mine`, `score_theirs`, `score_timeline_kind`, `mode_category`, `map_ui`, `mode_ui`,
  `playlist_label`, `start_time_label`. L'en-tête est servi ENTIER (toutes ses sources sont
  conservées, D4) : aucun champ d'en-tête ne diffère de la vue complète.
- **Ligne de scoreboard** (lu) : `xuid`, `gamertag`, `is_me`, `is_bot`, `team_side`, `team_name`,
  `joined_in_progress`, `left_in_progress`, `first_joined_time`, `last_leave_time`, `kills`,
  `deaths`, `assists` (compteurs de la fiche, `ReplayCountersBadge`).
- **Ligne de scoreboard, JAMAIS lus par le rejeu** (aucune occurrence) : `kda`, `accuracy`,
  `shots_*`, `damage_*`, `average_life`, `avg_life_seconds`, `*_kills` (headshot, grenade, melee,
  power weapon, perfect, assassination, ground pound, shoulder bash), `max_killing_spree`,
  `score`, `rank`, `outcome`, `top_weapon_*`, `offensive_conversion`, `defensive_resistance`,
  `damage_per_kill`, `damage_per_death`, `expected_kills`, `expected_deaths`, `expected_assists`,
  `locally_estimated`, `weapon_kills`, `medals`, `performance_score`, `had_bot_teammate`,
  `skill_rank`, `objective`, `team_color`. Les médailles des fiches et de la frise viennent des
  ÉVÉNEMENTS (`highlight_events`), pas de la ligne.
- **Événements** (lu) : `event_type`, `event_time_ms`, `actor_xuid`, `actor_gamertag`,
  `actor_team_id`, `victim_xuid`, `victim_gamertag`, `weapon_key`, `weapon_label`,
  `weapon_image_url`, `weapon_image_tinted`, `headshot`, `killer_damage_pct`, `assist_state`,
  `assist_gamertag`, `assist_team_id`, `assist_damage_pct`, `medal_name`, `medal_label`,
  `medal_description`, `medal_image_url`. Servis entiers (toutes leurs sources conservées).
- **Médias** : `media_items[]` (identifiant, nature, instants de capture, chemins) : servis entiers.

### D4 — liste fermée des chargements de `GetReplayContext` (écrite avant 2.3)

Conservés (chacun alimente un des quatre morceaux) :
1. `GetMatchMeta` (bloquant) — en-tête, recalage T0, durée.
2. `IsParticipant` (`assertParticipation`) — garde, comme `GetMatchView`.
3. `GetPlayerMatchStats` (Q17) — en-tête (issue, score, enrichissement).
4. `GetMatchEnrichment` (Q18) — en-tête (performance).
5. `GetMatchScoreboard` (Q12) — lignes, en-tête, équipe du tueur dans le fil.
6. `GetMatchEvents` (Q21) — événements.
7. `GetMatchKVPairs` — synthèse des kills quand les événements n'en portent pas ; victime du fil.
8. `GetMatchKillSources` (Q21b) — arme du kill.
9. `GetMatchKillAssists` (Q21c) — assistant du kill.
10. `LookupMedalMetaByName` (dans `decorateMedalEvents`) — identité des médailles du fil.
11. `GetMatchBulkMedals`, `GetMatchBulkWeaponKills` — `medals` / `weapon_kills` des lignes (non lus,
    mais lectures bornées au match : gardées pour que la ligne reste égale à la vue complète).
12. `GetMatchSkillRank`, `GetMatchSharedCSRs` — `skill_rank` des lignes (même raison).
13. `GetMatchMedia` (Q24) — médias.
14. `IsMatchFavorite`, `IsAvailable` du rejeu, signal d'analyse en cours, référentiel `team_colors`
    (`applyTeamNames`) — en-tête et `team_name` (déjà dans les `apply*` partagés).

Omis (n'alimentent aucun champ lu par le rejeu) :
- `GetMatchEncounters`, `GetMatchEncounterStats`, `GetMatchEncounterAssists` (rencontres),
  `GetMatchAssistPairs` (paires d'assistance), `GetMatchObjectiveScore` (radar), `GetMatchMedals`
  (Q14 : résumé, citations), events canoniques (cadence, rôles), distances de kill, citations,
  Emprise, Isolement, outils d'armes, radar.
- `GetHistoryForAvg`, `GetMatchExpectedStats`, `GetPlayerAssistsModel` + coefficients
  populationnels (`computeExpectedAssists`), `friendsExtras` (fan-out vers les bases des amis),
  `GetHistoryForAvgBulk` : ils ne posent sur les lignes que `performance_score`,
  `had_bot_teammate`, `skill_rank` (amis suivis), `expected_assists`, `expected_kills`,
  `expected_deaths`, `locally_estimated` — aucun lu par le rejeu (inventaire ci-dessus).

Conséquence pour la parité (3.1) : en-tête, événements et médias sont égaux À L'IDENTIQUE. Les
lignes sont égales à l'identique quand le match n'a ni ami suivi ni modèle d'attendus ; sinon elles
diffèrent EXACTEMENT sur les sept champs ci-dessus, et le test le prouve dans les deux sens.

## Découvertes (hors périmètre, non traitées)

- **Synthèse kv sur l'horloge du film quand T0 ≠ 0** (vue complète ET contexte, comportement
  inchangé par ce lot) : `applyKVSynthesisIfNeeded` synthétise les kills depuis `d.kvPairs`, qui
  n'est PAS recalé T0 (seule la copie `kvPairsFeed` l'est). Sur un match sans kill dans les events
  (Halo 5) à T0 non nul, les kills synthétisés gardent l'instant du film (countdown inclus) alors
  que les médailles sont recalées, et la victime (appariée sur `kvPairsFeed` corrigé) ne s'y pose
  pas. Constaté par la fixture de 3.1 (cas « synthèse », T0 = 15 s). À vérifier sur un match Halo 5
  réel (T0 y est-il jamais non nul ?) avant toute correction.
  Décision utilisateur 2026-10-20 : non corrigé. Mesuré sur les bases locales (copie de la prod) :
  Halo 5 3 032 matchs dont 2 754 en synthèse mais T0 > 0 sur 0 ; Halo Infinite 9 244 matchs dont
  8 681 avec T0 > 0 mais 0 en synthèse ; intersection 0. À reprendre si un titre réunit synthèse et
  T0 non nul.
- [x] **Nouvel essai automatique en cas de panne** (traité : amendement D2 du 2026-10-20,
  `source_unavailable` + `matchRefreshPolicy`, cf. décisions) : la vue complète portait `partial_reasons`, d'où un
  nouvel essai toutes les 30 s côté web quand une lecture avait échoué ; le contexte (forme D2) ne
  porte pas ces raisons. Le serveur pose `no-store` sur un contexte transitoire et le web garde la
  fraîcheur nulle + le rechargement de 60 s pendant l'analyse du film, mais une panne de lecture
  n'est plus réessayée toute seule toutes les 30 s sur la page de rejeu. À trancher : ajouter un
  drapeau au contrat (amende D2) ou s'en tenir là.
- **(P2) `applyTeamNames` hors de la preuve de parité** (revue adversariale ronde 1, 2026-10-20) :
  `apps/go-api/internal/service/match_view_replay_context.go:147` appelle
  `applyTeamNames` (`match_view_teams.go:42`), qui ne fait quelque chose que si l'adapter d'assets
  du titre résout noms et couleurs d'équipe (Halo 5). Le service factice de la parité n'expose pas
  ces résolveurs : retirer l'appel du contexte ne ferait rougir aucun test. Non traité : Halo 5 n'a
  pas de rejeu (`CapReplay`), la page qui lit ce contexte n'est pas montée pour lui.

## Revue adversariale, ronde 1 (2026-10-20)

Relecteur parité / accès : aucun constat recevable (14 conditions tiennent). Relecteur couverture :
6 constats, 5 corrigés, 1 consigné (Découvertes, `applyTeamNames`).
- [x] P1 en-tête : `replayFixtureService` câble table réglementaire, manches, règle du score dans
  le temps, issues du titre, rejeu, analyse en cours, taxonomie, favoris ; `assertHeaderExercised`
  exige prolongation, clé d'issue, score en manches, `score_timeline_kind`, `replay_available`,
  `analysis_pending` à causes `[kill_source]` seules (l'ordre rejeu → analyse compte), catégorie,
  favori. Mutations : chacun des sept `apply*` retiré, le favori forcé à faux, et l'ordre
  rejeu/analyse inversé font rougir le test (9 mutations, 9 rouges).
- [x] P1 `anyFailed`, branche des sources marquées : `TestGetReplayContext_TableauDeScoreEnPanne`
  (service) et `TestReplayContextHandler_TableauDeScoreEnPanne_NoStore` (vrai service + vrai
  handler : `source_unavailable` vrai + `no-store`, puis rétabli). Mutation `return f.section` :
  les deux rougissent.
- [x] P2 `actor_team_id` exigé sur un kill (`assertFeedExercised`) ; mutation `scoreboard: nil`
  dans `decorateHighlightEvents` : rouge.
- [x] P2 `TestGetReplayContext_AucunChargementLourd` : compteur de distances de kill câblé et
  `GetHistoryForAvgBulk` dans la liste lourde (ami suivi sans attendus), chacun vu > 0 sur la vue
  complète ; mutations qui ajoutent l'une ou l'autre lecture au contexte : rouge.
- [x] P2 ratchet `TestReplayRoutesMountedUnderCapabilityGate` étendu à `MountReplayContext` (même
  règle : `RequireCapability(..., CapReplay)` sur un bloc ancêtre) ; mutation qui sort la ligne du
  groupe du rejeu : rouge (`server_apiv1.go:806`).
- [~] `applyTeamNames` : consigné aux Découvertes (P2), non corrigé sur décision superviseur.
