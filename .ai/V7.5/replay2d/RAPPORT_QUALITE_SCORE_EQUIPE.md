# RAPPORT — Qualite de donnees : le « score d'equipe » de l'API n'est pas le score affiche

Lot : `PLAN_QUALITE_SCORE_EQUIPE_SYNC.md`. Branche : `wt/qualite-score` (worktree dedie,
base `feat/v75`). Date : 2026-08-24. Executeur : agent de lot.

Ce lot est un lot de DIAGNOSTIC. Il s'arrete au verdict de l'item 1.4 : aucune ecriture
DuckDB, aucun changement de sync, aucun changement d'affichage.

---

## Source unique de la colonne

`match_registry.team_0_score` / `team_1_score` sont ecrits par UN SEUL chemin cote
Halo Infinite :

`apps/go-api/internal/sync/transforms.go:128-130`

```go
// Team scores (depuis Teams[].Stats.CoreStats.Score)
t0, t1 := extractTeamScoresByID(matchJSON)
row.Team0Score = t0
row.Team1Score = t1
```

`extractTeamScoresByID` (`apps/go-api/internal/sync/transforms_helpers.go:160-193`) lit
`Teams[].Stats.CoreStats.Score` et l'indexe par `Teams[].TeamId` (0 et 1 seulement ;
au-dela, la valeur est perdue). C'est cette valeur, et elle seule, que toutes les surfaces
du §Phase 0 affichent.

Deux autres ecrivains existent, hors perimetre Infinite-sync :

| ecrivain | fichier:ligne | source |
|---|---|---|
| Import OpenSpartan | `apps/go-api/internal/openspartan/mapper/mapper.go:141,148` | colonnes du dump OpenSpartan |
| Ingest Halo 5 | `apps/go-api/internal/games/halo_5/ingest/collect.go:42-43` | carnage H5 `TeamStats[].Score` |
| Backfill ops H5 | `apps/go-api/cmd/h5-teamscore-backfill/main.go:108` | idem (UPDATE, titre H5 uniquement) |

---

## Phase 0 — Recensement des surfaces qui lisent `team_0/1_score`

### Commande de controle (rejouable)

```bash
cd C:/Users/Guillaume/Downloads/Scripts/LevelUp-wt-qualite-score
grep -rn "team_0_score\|team_1_score" apps/go-api/ cmd/ --include="*.go" | grep -v "_test.go" | sort
grep -rn "Team0Score\|Team1Score" apps/go-api/ cmd/ --include="*.go" | grep -v "_test.go" | sort
grep -rn "MyTeamScore\|EnemyTeamScore" apps/go-api/ --include="*.go" | grep -v "_test.go" | sort
grep -rn "score_label\|ScoreLabel" apps/go-api/ --include="*.go" | grep -v "_test.go" | sort
grep -rn "scoreLabel\|score_label\|own_score\|enemy_score" apps/web/src --include="*.ts" --include="*.tsx" | grep -v generated.ts
```

Sorties du 2026-08-24 : 44 occurrences SQL/commentaire, 41 occurrences de champ struct,
20 occurrences `MyTeamScore/EnemyTeamScore`, 24 occurrences `ScoreLabel` cote Go.
Le tableau ci-dessous est EXHAUSTIF sur ces sorties (aucun « etc. »).

### 0.1 — Lectures Go (chaine complete, source -> surface)

| # | lecture SQL / champ (fichier:ligne) | chaine intermediaire | surface finale |
|---|---|---|---|
| L1 | `queries_match.go:141-142` (`r.team_0_score, r.team_1_score`) | `match_view_repo.go:307-308` -> `domain.MatchMetaRaw.Team0Score/1Score` -> `service/match_view_builders_header.go:188,281-296 buildScoreLabelFromMeta` -> `MatchViewHeader.ScoreLabel` (`json:"score_label"`) | **En-tete de la page Match** |
| L2 | `queries_home_citations.go:68-69` (`COALESCE(r.team_X_score,-1)`) | `home_repo_matches.go:118-119` -> `legacymatch.HomeMatchRow.Team0Score/1Score` -> `analysis/home_locale.go:138-153 buildHomeScoreLabel` -> `domain.HomeMatchItem.ScoreLabel` (`json:"score_label,omitempty"`) | **Cartes de match de l'Accueil** (chemin legacy) |
| L3 | `player_matches_repo.go:312-313` (`COALESCE(r.team_X_score,-1)`) | `player_matches_projection.go:36-49 projectTeamScores` -> `canonical.PlayerMatchRow.Summary.Teams[]` -> `analysis/home_canonical_recent.go:222-247 buildScoreLabelCanonical` (et `home_canonical_converters.go:104-113`) -> `HomeMatchItem.ScoreLabel` | **Cartes de match de l'Accueil** (chemin canonique) |
| L4 | `queries_career.go:101-102` (`r.team_0_score, r.team_1_score`, requete Q5 historique) | `match_history_repo.go:137-149` (scan `team0/team1`) -> `applyTeamScore` (`:272-287`) -> `domain.MatchHistoryRawRow.MyTeamScore/EnemyTeamScore` -> `service/match_history_service_enrich.go:183-186` (`"%d - %d"`) -> `domain.MatchHistoryRow.ScoreLabel` | **Historique des matchs** ; re-projete en **Explorer** via `api/handlers/projections.go:25` -> `domain.ExplorerMatchesRow.ScoreLabel` |
| L5 | `queries_squad.go:110-111` (Q30, `CASE WHEN p1.team_id = 0 ...`) | `squad_repo.go:303-304` -> `domain` squad row `MyTeamScore/EnemyTeamScore` -> `teammates/teammates_service_assets.go:259-261,298` (`"%d - %d"`) -> `domain.TeammateMatchRow.ScoreLabel` (`json:"score_label,omitempty"`) | **Escouade — historique de synergie** |
| L6 | `queries_squad.go:153-154` (Q31, matchs communs avec un coequipier) | `squad_repo.go:357-358` -> meme chaine que L5 | **Escouade — matchs d'un coequipier** |
| L7 | `media_repo_filters.go:250-251,297-306` | `domain.media` `OwnScore/EnemyScore` (`json:"own_score" / "enemy_score"`) | **Selecteur de match du module Media** |
| L8 | `replay_facts_repo.go:76` (`SELECT team_0_score, team_1_score, ...`) | `port.MatchFacts.TeamScores` -> `analysis/replay/score_timeline.go:33 ScoreInput.TeamScores` -> `score_team_identity.go:41 identityByFinalScore` (preuve (a)) | **Rejeu 2D : IDENTITE des camps** (pas le score affiche — cf. 0.3) |
| L9 | `sync/comeback.go:206` (`loadTeamScoresOrKillSums`) | appele en `comeback.go:148` UNIQUEMENT dans la branche « pas de kill-event » -> `analysis.ComputeScoreMarginDominance` -> `dominance_flag` -> badges narratifs (`analysis/home_locale.go buildHomeNarrativeBadges` : dominant / humiliation / remontada / debacle) | **Badges de dominance — HORS HALO INFINITE** : `comeback.go:141-143` renvoie `0, nil` quand `ctxkeys.TitleSlug(ctx) == titlePkg.DefaultSlug`. Sur Infinite la colonne n'est JAMAIS lue par ce chemin |
| L10 | `halo5/halo5_match_history_source.go:71-72` | `h5TeamScores` (`:220-230`) -> `canonical.TeamSnapshot` | Historique **Halo 5** (meme surface que L4, source H5) |
| L11 | `migration/steps_shared.go:271-272` | vue `v_match_full` (`CREATE OR REPLACE VIEW`, `:237`) | Pas une surface : la vue re-expose la colonne, consommee par L4/L5/L6 |
| L12 | `persist/shared_persister.go:163,191` + `persist/demo_seed_columns.go:31` + `ops/seed_demo_synthetic_shared.go:117` | ecriture INSERT-only / seed demo | Pas une surface de lecture utilisateur |
| L13 | `cmd/diag_orphan_session/main.go:197`, `cmd/diag_weapons_v3/db.go:81`, `cmd/investigate_matches/main.go:42-43,152-153` | sortie console | **Outils de diagnostic** — pas d'UI |
| L14 | `sync/schema.go:181-182`, `games/halo_infinite/migrations/steps_shared_core.go:56-57`, `sync/testutil/fixture.go:53-54` | DDL | Pas une surface |

Faux positifs ecartes du recensement, pour qu'on ne les recherche pas deux fois :

- `service/squadagg/aggregates.go:177 scoreLabel(score float64)` — HOMONYME. Mappe un score
  de performance 0..100 vers « excellent / good / ... ». Aucun rapport avec le score d'equipe.
- `analysis/comeback.go:54-55,81-94` et `analysis/comeback_objective.go:50,68-69`
  (`ScoreSnapshot.Team0Score/Team1Score`) — construits depuis les kill-events / les
  `match_objective_events`, PAS depuis `match_registry`. Meme nom de champ, autre source.
- `domain/chart/antagonists.go:84-95 TeamSnapshot` — type homonyme du domaine graphique.
- `features/session-detail/SessionPerfTrend.tsx`, `features/timeseries/TimeseriesFormCharts.tsx`,
  `features/explorer/ExplorerCombatProfile.tsx` — variables `scoreLabel` locales portant un
  libelle de serie (« Score personnel »), sans lien avec le score d'equipe.

### 0.2 — Surfaces web (composant qui rend la valeur)

| # | composant (fichier:ligne) | champ consomme | provient de |
|---|---|---|---|
| W1 | `apps/web/src/features/match-view/MatchHeader.card.tsx:361,370` | `header.score_label` | L1 |
| W2 | `apps/web/src/components/ui/match-card.tsx:72,225` | `m.score_label` | L2 / L3 (cartes Accueil et Sessions) |
| W3 | `apps/web/src/features/explorer/ExplorerMatchesTable.tsx:178,649` (colonne `score_label`, triable — `explorerMatchesClientSort.ts:65`) | `row.score_label` | L4 |
| W4 | `apps/web/src/features/squad/SquadSynergyHistoryTable.tsx:283` (colonne `score_label`) | `row.score_label` | L5 / L6 |
| W5 | `apps/web/src/features/media/MediaMatchPicker.tsx:80-81` (`${own_score} - ${enemy_score}`) | `candidate.own_score/enemy_score` | L7 |
| W6 | `apps/web/src/features/session-detail/SessionMatchesTable.tsx:52,69,99` | `score_label` **desactive** (`false` dans les deux tables de colonnes, valeur forcee `''`) | — surface NEUTRALISEE |
| W7 | `apps/web/src/features/match-replay/scoreBannerLogic.ts:52` | commentaire de reference a `MatchViewHeader.score_label` ; le bandeau du rejeu 2D lit le score du FILM, pas la colonne | — pas une surface de la colonne |

### 0.3 — Visibilite de l'anomalie par l'utilisateur

Rappel du defaut mesure (LOTA_PHASE0.md) : la colonne porte, selon le mode et selon le
match, soit le score affiche a l'ecran, soit une autre unite (ticks Strongholds, secondes
de garde KOTH).

| surface | anomalie visible ? | forme prise a l'ecran |
|---|---|---|
| W1 en-tete de la page Match | **OUI** | un Strongholds affiche `193-112` au lieu de `200-126` ; un KOTH affiche `105-8` au lieu de `3-0` |
| W2 cartes Accueil / Sessions | **OUI** | idem, sur la vignette de chaque match |
| W3 colonne Score de l'Explorer | **OUI, et aggravee** | la colonne est TRIABLE : melanger des unites (`3`, `105`, `193`) rend le tri de cette colonne sans signification entre modes |
| W4 colonne Score de l'Escouade | **OUI** | idem W3 sans le tri |
| W5 selecteur Media | **OUI** | le libellé du candidat porte le mauvais score |
| L8 rejeu 2D (identite des camps) | **NON — mais fragile** | la colonne n'est pas AFFICHEE : elle sert de preuve (a) pour rattacher un slot d'entite a un camp. La preuve ne demande que des valeurs DISTINCTES, pas justes ; elle est donc robuste a l'erreur d'unite. Elle DEGRADE (bascule sur la preuve (b), somme des frags) quand les deux valeurs sont egales. Aucun affichage errone possible par ce chemin |
| L9 badges de dominance | **NON sur Halo Infinite** | chemin coupe par `comeback.go:141-143` (`DefaultSlug -> 0, nil`). Reste actif sur Halo 5 |
| L13 outils de diagnostic | non (pas d'UI) | — |
| W6 table des matchs d'une session | non (colonne desactivee) | — |

**Bilan phase 0** : 5 surfaces utilisateur affichent directement la valeur (W1..W5), toutes
avec la meme anomalie ; 1 surface (Explorer) l'aggrave par un tri inter-modes ; 2 surfaces
consomment la colonne sans l'afficher et sans risque (L8, L9-Infinite).

Statut des items : 0.1 `[x]` · 0.2 `[x]` · 0.3 `[x]`.

---

## Phase 1 — Diagnostic donnees

### 1.1 — JSON brut de l'API sur les 3 matchs fautifs

Outil : `apps/go-api/cmd/diag_matchstats_dump` (cree pour ce lot ; auth store-first
ADR 0023, AUCUNE ouverture DuckDB, AUCUNE ecriture en base). Commande exacte :

```bash
LEVELUP_REPO_ROOT=C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration \
  go run ./cmd/diag_matchstats_dump --gamertag JGtm --rps 4 \
  --out .ai/V7.5/replay2d/registre_film/api_dumps \
  7344d24f-0154-4949-80ad-e2b781c122f1 \
  606d9844-4f22-42c1-8fb6-e9d541e5ff4c \
  8076f97f-b8aa-4949-ab51-6be5221d41e8
```

Payloads conserves dans `.ai/V7.5/replay2d/registre_film/api_dumps/`.

`Teams[].Stats` contient exactement trois blocs : `CoreStats`, `PvpStats`, `ZonesStats`.
Voici les deux seuls champs qui portent un nombre comparable a un score :

| match | mode | jeu / film | DB `team_X_score` | API `CoreStats.Score` | API `ZonesStats.StrongholdScoringTicks` |
|---|---|---|---|---|---|
| `7344d24f` | Strongholds:Arena | **200 / 126** | 193 / 112 | **200 / 126** | **193 / 112** |
| `606d9844` | KOTH:Arena | **3 / 0** | 105 / 8 | **3 / 0** | **105 / 8** |
| `8076f97f` | KOTH:Arena | **0 / 3** | 78 / 105 | **0 / 3** | **78 / 105** |

**Le resultat renverse la premisse du plan.** L'API porte le score AFFICHE, et elle le
porte dans `Teams[].Stats.CoreStats.Score` — c'est-a-dire EXACTEMENT le champ que
`extractTeamScoresByID` lit deja (`transforms_helpers.go:177`). Ce que la base contient,
sur ces trois matchs, c'est l'autre champ : `ZonesStats.StrongholdScoringTicks`, au tick
pres, sur les 6 valeurs.

Le fait etabli du plan « la MEME colonne change de semantique selon le match » etait donc
une lecture de la BASE, pas de l'API. L'API est homogene ; c'est la base qui melange deux
champs. Corollaire immediat : ni l'unite KOTH ni le plafond Strongholds ne sont des
proprietes de l'API — ce sont les deux visages du meme substitut de champ.

Deux controles ecartent une coincidence numerique :

- `CoreStats.RoundsWon` vaut 1/0 sur `606d9844` et 0/1 sur `8076f97f` : le KOTH de ces deux
  matchs se joue en UNE manche, et `Score` = 3/0 et 0/3 est bien le score de colline
  affiche, pas un compte de manches gagnees.
- En Strongholds les `StrongholdScoringTicks` PAR JOUEUR valent 0 pour les 8 joueurs de
  `7344d24f` : la valeur 193/112 n'existe qu'au niveau EQUIPE. La base ne peut donc pas
  l'avoir derivee d'une somme de joueurs — elle a bien lu le champ d'equipe.

### 1.2 — Ampleur : mesure EXACTE, pas heuristique

L'item 1.2 prevoyait une heuristique par mode (« un score de KOTH affiche est <= 5 »)
parce que le plan supposait l'API muette. 1.1 ayant refute cette premisse, l'heuristique
est sans objet : l'API donne la reponse match par match. **Les 1 934 matchs de
`match_registry` qui portent un score ont donc ete re-fetches** (`diag_matchstats_dump`,
17 lots de 120, 1 934/1 934 succes, 0 erreur, ~22 min a `--rps 4`) et confrontes a la base.

```bash
# liste
diag_q <shared.duckdb> "SELECT match_id FROM match_registry WHERE team_0_score IS NOT NULL ORDER BY match_id"
# fetch
cat ids.txt | xargs -n 120 diag_matchstats_dump --gamertag JGtm --rps 4 --out <dir>
# extraction
ls *.json | xargs -n 200 jq -r -s '.[] | [.MatchId,
  ([.Teams[]|select(.TeamId==0)|.Stats.CoreStats.Score]|first // -999),
  ([.Teams[]|select(.TeamId==1)|.Stats.CoreStats.Score]|first // -999),
  ([.Teams[]|select(.TeamId==0)|.Stats.ZonesStats.StrongholdScoringTicks]|first // -999),
  ([.Teams[]|select(.TeamId==1)|.Stats.ZonesStats.StrongholdScoringTicks]|first // -999),
  (.Teams|length)] | @tsv' > api_all.tsv
# confrontation (read_csv + jointure, lecture seule)
```

**Total : 1 934 matchs · 1 853 conformes · 80 faux (4,1 %) · 1 hors comparaison** (un FFA
sans `TeamId` 0/1).

| famille de mode | n | faux | % faux | dont valeur = ticks de zone |
|---|---|---|---|---|
| **Strongholds** | 83 | **51** | **61,4 %** | 50 |
| Total Control | 124 | 16 | 12,9 % | 16 |
| Oddball | 26 | 6 | 23,1 % | 0 |
| KOTH | 56 | 3 | 5,4 % | 3 |
| CTF | 353 | 2 | 0,6 % | 0 |
| Slayer et autres | 1 248 | 2 | 0,2 % | 0 |
| Stockpile | 44 | 0 | 0 % | 0 |

Le mode reellement touche est **Strongholds** (3 matchs sur 5 faux), puis Total Control.
Le KOTH — le mode qui a declenche l'alerte — est en fait le moins touche des modes de zone.

**Et le decoupage qui compte n'est pas le mode, c'est la DATE DE SYNC :**

| fenetre de `first_sync_at` | n | faux | % faux |
|---|---|---|---|
| A. lot initial 2026-02-14 | 1 283 | 66 | 5,1 % |
| B. 2026-02-17 → 2026-04-06 | 255 | 14 | 5,5 % |
| **C. 2026-05-06 → 2026-08-19** | **396** | **0** | **0 %** |

Aucun match n'a ete synchronise entre le 2026-04-07 et le 2026-05-05 : la bascule se situe
dans ce trou. **Les 80 lignes fausses ont TOUTES ete ecrites avant le 2026-04-06 ; les 396
matchs entres depuis le 2026-05-06 sont justes a 100 %, Strongholds compris** (22 Strongholds
dans la fenetre C, 22 conformes ; 14 Strongholds dans la fenetre B, 14 faux — la coupure est
nette, cf. le detail par match dans `registre_film/score_equipe_ecarts_2026-08-24.tsv`).

Lecture : le code de sync n'a pas change (`extractTeamScoresByID` lit `CoreStats.Score`
depuis le portage Go de sprint 18, `b58cec2d4`, et la version Python qu'il porte lisait deja
`TeamId`). C'est l'API 343 qui servait le compteur brut du mode dans `CoreStats.Score` et
qui a ete corrigee entre avril et mai 2026. **Le defaut est donc CLOS A LA SOURCE ; ce qui
reste est un residu historique de 80 lignes.**

Sous-famille residuelle, 11 des 80 ne s'expliquent PAS par les ticks de zone :

| match | mode | DB | API | forme |
|---|---|---|---|---|
| `293a763e` `9fa3100c` `adb93fb7` `ca738284` `3338df70` `d9781168` | Oddball (6) | ex. 155/218 | 218/155 | **camps INVERSES**, valeurs exactes |
| `50de9126` | BTB:Sentry Defense | 165/319 | 319/165 | camps inverses |
| `1141896d` | Arena:Strongholds | 188/182 | 200/199 | ticks (182/188) **et** inverses |
| `443426df` `d1e89330` | BTB:One Flag CTF (2) | 0/2 | 0/3 | ecart de 1 sur le vainqueur |
| `f395b462` | Arena:Attrition | 2/**1950** | 2/0 | valeur aberrante |

Les 11 sont toutes dans la fenetre A (2026-02-14) et 10 sur 11 portent
`first_sync_by = Madina97294`. Mecanisme non identifie — consigne en §Decouvertes, non traite.

### 1.3 — Oracle croise : le film contre l'API

Le film Theater porte le score AFFICHE (`coverage.score.oracle = "displayed"`). Les
artefacts de rejeu presents sur disque sont **35** (le plan en annonce 39 : chiffre perime).
20 d'entre eux ont une identite de camp resolue pour LES DEUX camps, seul cas ou la
confrontation terme a terme a un sens.

| confrontation | n | resultat |
|---|---|---|
| film = API `CoreStats.Score` | **18 / 20** | accord exact |
| ecart film/API | 2 / 20 | `06dfe6d9` (film 1/2, API 1/3) et `11de8353` (film 52/87, API 65/100) — **les deux films sont tronques** et SOUS-comptent ; l'API et la base concordent entre elles |

Deux artefacts tranchent le litige de facon decisive, parce que la base y est fausse :

| match | mode | DB | API | **film** | qui a raison |
|---|---|---|---|---|---|
| `7344d24f` | Strongholds:Arena | 193/112 | 200/126 | **200/126** | API |
| `af13e2b2` | Arena:Strongholds | 3/134 | 3/200 | **3/200** | API |

Sur les deux matchs ou base et API divergent, **le film tranche pour l'API**. Les deux KOTH
`606d9844` et `8076f97f` vont dans le meme sens (film 3/(absent=0) et (absent=0)/3, soit 3/0
et 0/3 = API) mais ne comptent pas dans le denominateur des 20 : un seul de leurs camps porte
une serie, l'autre n'ayant jamais marque.

**Taux : 18/20 = 90 % d'accord exact, et 0 cas ou le film donne raison a la base contre l'API.**

### 1.4 — VERDICT

**Issue (a) : l'API porte le score affiche, et elle le porte dans le champ que la sync lit
deja — `Teams[].Stats.CoreStats.Score`.**

Argumentaire, dans l'ordre des preuves :

1. **Couverture du champ : 1 933 / 1 934 matchs** (100 % des matchs a deux camps ; le seul
   manquant est un FFA sans `TeamId` 0/1). Le champ n'est jamais absent la ou il a un sens.
2. **Concordance avec la base : 1 853 / 1 933 = 95,9 %.** Les 80 ecarts sont TOUS anterieurs
   au 2026-04-06 ; sur les 396 matchs entres depuis le 2026-05-06, la concordance est de
   **396 / 396 = 100 %**.
3. **Concordance avec le film** (oracle « displayed », independant de l'API et de la base) :
   **18 / 20** exacts, les 2 manquants etant des films tronques qui sous-comptent. Et sur les
   2 artefacts ou base et API divergent, le film donne raison a l'API, jamais a la base.
4. **Le substitut est identifie et nomme** : 69 des 80 valeurs fausses sont, au tick pres,
   `Teams[].Stats.ZonesStats.StrongholdScoringTicks`. Ce n'est pas une hypothese, c'est une
   egalite verifiee sur 138 nombres.

Les issues (b) et (c) sont **refutees** :

- **(b) « calculable par regle de mode »** — sans objet : rien n'est a calculer, il suffit de
  lire le champ. Et la regle serait de toute facon impossible a ecrire : en Strongholds les
  ticks (193) et le score (200) vivent dans la meme plage numerique, aucune heuristique de
  seuil ne les separe. C'est precisement pour cela que l'heuristique de l'item 1.2 a ete
  remplacee par une mesure.
- **(c) « seul le film le porte »** — refutee : le film ne couvre que 35 matchs, l'API couvre
  les 1 934, et les deux disent la meme chose la ou ils se recouvrent.

**Consequence de perimetre.** Le probleme n'est pas un defaut de sync a corriger : la sync
lit le bon champ et le lit correctement depuis le 2026-05-06. C'est un **residu de donnees de
80 lignes** (4,1 %), ecrit pendant que l'API 343 servait le compteur brut du mode. Un
re-fetch par le pipeline normal ne les repare PAS : `persistMatchRegistry`
(`persist/shared_persister.go:154-200`) est un `INSERT` NU, sans `ON CONFLICT` — un match
deja present n'est jamais reecrit. Il faut un backfill dedie.

Les 3 matchs fautifs du plan sont expliques par le verdict :

| match | pourquoi la base est fausse | valeur correcte |
|---|---|---|
| `7344d24f` | ecrit le 2026-02-14 (fenetre A) ; valeur = `ZonesStats.StrongholdScoringTicks` | 200 / 126 |
| `606d9844` | idem | 3 / 0 |
| `8076f97f` | idem | 0 / 3 |

Et l'enigme laissee ouverte par `LOTA_PHASE0.md` §0b.3 (« le film DEPASSE l'oracle sur
`7344d24f`, non explique ») est **resolue** : le film ne depassait rien, c'est l'oracle qui
etait le mauvais nombre. Le motif de mode signale la-bas (trois Strongholds sur quatre
portant 200 au film) etait le film ayant raison, pas un artefact du filtre de monotonie.

## Options d'implementation (chiffrees) — pour arbitrage

Contraintes communes, verifiees sur pieces :

- `match_registry` **n'est PAS append-only** : pas de vue `_latest`, pas de `written_at`.
  La correction se fait par `UPDATE`.
- Le garde-rail anti-ART (`internal/sync/no_art_patterns_test.go:390-400`, liste
  `criticalMatchTables`) **autorise** l'`UPDATE` row-by-row `WHERE match_id = ?` et
  **interdit** `UPDATE … FROM (VALUES …)` et l'`UPDATE` set-based sans placeholder.
  Precedent exact, meme table et memes deux colonnes : `cmd/h5-teamscore-backfill/main.go:108`.
- **Aucune migration de schema** n'est requise (pas de colonne nouvelle) : la contrainte
  « elargir une migration deployee = step au nom neuf » ne s'applique pas ici.
- Ecriture RW sur la shared DB → **serveur arrete** (ADR 0013 dblease, un seul writer).

| # | option | perimetre | cout | risque | ce qu'elle ne couvre pas |
|---|---|---|---|---|---|
| **1** | **Backfill cible sur la liste mesuree** | les **80** lignes de `score_equipe_ecarts_2026-08-24.tsv` | ~80 appels API (< 1 min) ; CLI ~130 L calquee sur `h5-teamscore-backfill` ; serveur arrete ~5 min | faible — la liste est nominative et la valeur cible connue | ne re-verifie pas les 1 854 autres ; si l'API bouge encore, un nouvel ecart passe inapercu |
| **2** | **Backfill de toute la fenetre pre-2026-04-06 + garde-rail** | les **1 538** lignes des fenetres A et B, comparees puis corrigees si divergentes | ~1 538 appels API (~18 min mesures) ; meme CLI + un compteur d'ecarts logue ; serveur arrete ~25 min | faible ; plus long serveur arrete | rien de connu — c'est l'option exhaustive sur la population a risque |
| **3** | correction a l'affichage | les 5 surfaces W1..W5 | — | **non viable** | **A REJETER** : la base ne porte aucun marqueur disant « ceci est un tick » ; en Strongholds ticks et score sont dans la meme plage. Rien n'est derivable a la lecture |
| **4** | statu quo documente | — | 0 | — | 80 matchs (4,1 %) gardent un score faux sur 5 surfaces utilisateur, dont 51 Strongholds ; la colonne Score de l'Explorer reste triable sur des valeurs heterogenes |

**Recommandation de l'executeur : option 1**, et l'ajout au meme lot d'un compteur de
divergence dans la sync (log `WARN` si `CoreStats.Score` differe d'un champ de repli connu)
n'est PAS recommande — le defaut est ferme cote API depuis mai, un garde-rail permanent
contre un bug tiers deja corrige serait un « compatibility guard forever ». L'option 2 se
justifie seulement si le superviseur veut une garantie exhaustive plutot qu'une correction
de la liste mesuree.

Dans les deux cas, **le film n'a pas a etre re-cuit** : les artefacts portent deja le score
affiche (`oracle = "displayed"`), et l'identite des camps du rejeu (`replay_facts_repo.go:76`)
ne demande que des valeurs DISTINCTES — les 80 corrections ne changeront l'issue d'aucune
resolution (a), sauf a rendre distinctes deux valeurs qui ne l'etaient pas, ce qui ne peut
qu'ameliorer la cascade.

## Correctif pret (phase 2 — arbitrage du 2026-08-24 : CODE SEULEMENT)

L'utilisateur a arbitre l'**option 1** (backfill cible). La CLI est ecrite ; elle **n'a ete
executee contre aucune base** — ni `--apply`, ni `--dry-run`, ni contre une copie.
L'execution est gatee « avant le tag v7.5.0 » et suivie cote superviseur. La raison de la
barriere : le `--dry-run` lui-meme fait 80 appels API et lit la shared DB, et une autre
session tient des fichiers du depot principal.

### Ce qui a ete livre

| fichier | role |
|---|---|
| `apps/go-api/internal/sync/transforms_helpers.go:178` | `extractTeamScoresByID` **exporte** en `ExtractTeamScoresByID`. La sync et le backfill lisent desormais le score d'equipe par LA MEME fonction — zero seconde implementation, donc zero re-divergence possible |
| `apps/go-api/cmd/backfill-team-scores/decide.go` | la regle de correction, pure : ni reseau, ni DuckDB, ni flags |
| `apps/go-api/cmd/backfill-team-scores/ids.go` | chargement de la liste — **seule la colonne `match_id`** sort du TSV |
| `apps/go-api/cmd/backfill-team-scores/main.go` | cablage : auth store-first, fetch, lease writer, `UPDATE` row-by-row |
| `…/decide_test.go`, `…/ids_test.go` | 24 cas table-driven, sans reseau ni base |

### Garanties inscrites dans le code

- **Le TSV ne sert que de liste.** Ses colonnes de valeurs (`api_t0`, `db_t0`, …) ne sont
  jamais lues : chaque score est re-telecharge a l'execution. Une mesure vieille de
  plusieurs semaines ne peut pas devenir une ecriture. Fige par
  `TestLoadMatchIDs_ValeursIgnorees`.
- **`CoreStats.Score` fait foi, jamais les ticks.** `TestDecide_TicksNeverWin` construit un
  payload ou les deux coexistent et echoue si le tick l'emporte — c'est exactement le bug
  d'origine, il ne peut plus revenir en silence.
- **Ecriture uniquement si different**, sinon `identique` (outil idempotent).
- **Jamais de NULL, jamais de negatif, jamais hors bornes** du `SMALLINT` de la colonne
  (`scoreMin`/`scoreMax`) ; `TestDecide_NeverWritesNilOrNegative` balaie la plage.
- **NULL n'est pas zero** : une colonne NULL est une ligne A CORRIGER, pas une ligne conforme.
- **FFA saute et le dit** : sans `TeamId` 0 ET 1, aucun camp n'est devine.
- **Forme d'ecriture** : `UPDATE match_registry SET team_0_score = ?, team_1_score = ?
  WHERE match_id = ?`, row-by-row serialise, sous lease `dblease.KindSharedMatches`. Jamais
  de `UPDATE … FROM (VALUES …)` — forme interdite par `no_art_patterns_test.go`
  (declencheur ART #23046). Un `UPDATE` qui touche un nombre de lignes different de 1 est
  une erreur, pas un succes silencieux.
- **`--dry-run` par defaut** : il lit par `OpenReadForQuery` et tourne SERVEUR ALLUME.
  `--apply` ouvre en `OpenReadWrite` et echoue si un autre process tient la base — c'est le
  comportement voulu (ADR 0013, un seul writer).

### Commandes du jour J

```bash
cd apps/go-api
export GOCACHE=<cache prive>          # une commande go a la fois

# 1. REPETITION A BLANC — serveur allume, aucune ecriture.
#    Attendu : « corriges=80 » (ce qui SERAIT corrige), « echecs=0 ».
LEVELUP_REPO_ROOT=C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration \
  go run ./cmd/backfill-team-scores --gamertag JGtm

# 2. APPLICATION — SERVEUR ARRETE (air + server.exe), sinon l'ouverture RW echoue.
LEVELUP_REPO_ROOT=C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration \
  go run ./cmd/backfill-team-scores --gamertag JGtm --apply

# 3. CONTROLE — re-jouer le dry-run : attendu « identiques=80, corriges=0 ».
LEVELUP_REPO_ROOT=C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration \
  go run ./cmd/backfill-team-scores --gamertag JGtm
```

Un seul match, pour un premier essai prudent :

```bash
go run ./cmd/backfill-team-scores --match 7344d24f-0154-4949-80ad-e2b781c122f1 --apply
# attendu : avant 193/112 -> apres 200/126
```

Deux ecarts au resume signalent un vrai probleme, pas un alea : `echecs > 0` (API ou
`UPDATE`) et `skippes` inattendu (un match qui n'etait pas FFA en phase 1 ne doit pas le
devenir). Le decompte de reference de la phase 1 est **80 a corriger, 0 skip**.

### Note PROD (VPS) — a verifier AVANT de rejouer la passe

Tout ce qui precede est mesure sur la base LOCALE. **Le residu de 80 lignes n'a pas ete
verifie sur le VPS** et il n'y est pas necessairement identique : la base de prod a sa
propre histoire de synchronisation, et c'est la DATE DE SYNC — pas le match — qui determine
si une ligne est fausse. Avant toute execution en prod :

1. Mesurer le residu la-bas plutot que de supposer la meme liste : re-jouer la confrontation
   de la phase 1 (`cmd/diag_matchstats_dump` + comparaison) sur la base du VPS, ou a defaut
   lancer le `--dry-run` de la CLI, qui est non destructif et fait le meme travail de
   comparaison.
2. Si le residu prod differe, **produire une liste prod** et la passer en `--ids-file` : ne
   pas rejouer le TSV local sur une base dont ce n'est pas l'inventaire.
3. Le `--apply` exige l'arret du service (un seul writer) : operation a annoncer a
   l'utilisateur AVANT, comme toute intervention VPS.
4. Rien n'est urgent : le defaut est clos a la source depuis mai 2026, la prod ne se degrade
   plus. La passe peut attendre une fenetre choisie.

## Gates rejoues

| gate | commande | resultat |
|---|---|---|
| Gate 0 — grep de controle | les 5 `grep` du §Phase 0 | 44 + 41 + 20 + 24 occurrences Go, 7 surfaces web ; tableau exhaustif produit |
| Lecture DuckDB non intrusive | `diag_q <shared.duckdb> "SELECT count(*) …"` avec le serveur `air`/`server.exe` EN COURS | OK — 1 934 lignes lues, aucune ouverture RW, aucun verrou pris |
| Fetch API 1.1 | `diag_matchstats_dump … <3 match_id>` | 3/3 OK (37 180 / 33 044 / 38 173 octets) |
| Fetch API 1.2 | idem sur 1 934 match_id | **1 934 / 1 934 OK, 0 erreur** |
| gofmt | `gofmt -l ./cmd/diag_matchstats_dump/` | sortie vide |
| go vet | `go vet ./cmd/diag_matchstats_dump/` | exit 0 |
| Garde-rail anti-ART | `go test ./internal/sync/ -run "ART\|Art\|NoArt\|BulkUpdate\|Pattern\|TeamScore" -count=1` | `ok levelup/go-api/internal/sync 12.624s` |

Phase 2 (CLI de backfill, sans execution) :

| gate | commande | resultat |
|---|---|---|
| build de la CLI | `go build ./cmd/backfill-team-scores/` | exit 0 |
| tests de la CLI | `go test ./cmd/backfill-team-scores/ -count=1 -v` | `ok … 0.140s` — 24 cas, dont le chargement du TSV versionne (80 ids) |
| gofmt | `gofmt -l ./cmd/backfill-team-scores/ ./internal/sync/transforms{,_helpers}.go ./internal/sync/transforms_helpers_unit_test.go` | sortie vide |
| go vet | `go vet ./cmd/backfill-team-scores/ ./internal/sync/` | exit 0 |
| Garde-rail anti-ART (2.1 touche `internal/`) | `go test ./internal/sync/ -run "NoArt\|Pattern" -count=1` | `ok … 13.245s` |
| Gel du god-package sync + ratchets | `go test ./internal/archlint/ -count=1` | `ok … 15.489s` (aucun fichier ajoute a la racine de `internal/sync/`) |
| Non-regression du renommage | `go test ./internal/sync/... -count=1` | `ok` sur les 12 paquets (`internal/sync` 44,950s) |
| Seuils CLAUDE.md | `wc -l` + scan des longueurs de fonction | 332 / 127 / 91 L par fichier, aucune fonction > 80 L |
| **Aucune execution** | — | la CLI n'a ete lancee contre AUCUNE base, ni `--apply`, ni `--dry-run`, ni sur une copie |

`GOCACHE` prive au lot (`<worktree>/.gocache`) sur toutes les commandes `go`, une seule a la
fois. Le dossier n'est pas suivi.

## Decouvertes (hors perimetre — CONSIGNEES, NON TRAITEES)

1. **11 lignes fausses que le substitut de ticks n'explique pas**, toutes du lot initial
   2026-02-14, 10/11 avec `first_sync_by = Madina97294` : 7 sont des **inversions exactes des
   deux camps** (6 Oddball + 1 BTB:Sentry Defense), 1 cumule inversion et ticks
   (`1141896d`), 2 One Flag CTF ont un score de vainqueur a 2 au lieu de 3, et
   `f395b462` (Arena:Attrition) porte **1 950** la ou l'API donne 0. Le mecanisme n'a pas ete
   identifie et ne l'a pas ete cherche au-dela : la piste « ordre du tableau `Teams[]` » et la
   piste « point de vue du joueur qui synchronise » ont ete testees et ne collent ni l'une ni
   l'autre. Une correction par backfill les repare quelle que soit la cause.
2. **`match_participants.team_id` est SAIN** sur ces memes matchs : les `Outcome` par camp de
   l'API concordent avec ceux des participants en base. L'inversion touche donc uniquement les
   deux colonnes de score, pas l'appartenance des joueurs.
3. **`f395b462` : 1 950 dans un `SMALLINT`** — la valeur passe la contrainte de type mais est
   physiquement impossible en Attrition. Aucun controle de vraisemblance n'existe a l'ecriture
   du registre.
4. **Colonne Score de l'Explorer triable sur des unites heterogenes** : meme apres correction
   des 80 lignes, trier « Score » sur un melange de Slayer (50), CTF (3) et Strongholds (200)
   n'a pas de sens produit. Ce n'est pas un bug de donnee, c'est une question de conception de
   la colonne — hors perimetre de ce lot.
5. **Le plan annonce 39 artefacts de rejeu ; il y en a 35** sur disque
   (`data/cache/replays/halo_infinite/*.json`). Chiffre a corriger dans les documents qui le
   citent.
6. **`LOTA_PHASE0.md` §0b.3 est a amender** : son « `7344d24f` : les deux hypotheses proposees
   sont REFUTEES / non explique » est desormais explique — l'oracle etait faux, pas le film.
   Le fichier n'a pas ete modifie (hors perimetre).
7. **`sync/comeback.go:141-143`** coupe le fallback marge-de-score pour `DefaultSlug` avec une
   justification datee (« ne pas relabelliser retroactivement des badges HINF »). Le
   commentaire ne porte ni date cible de retrait ni critere mesurable, contrairement au modele
   de kill-switch impose par le CLAUDE.md. Non traite.

## Statut des items du plan

| item | statut | justification |
|---|---|---|
| 0.1 recensement des lectures Go | `[x]` | 14 chaines L1..L14, §0.1 |
| 0.2 recensement des surfaces web | `[x]` | 7 entrees W1..W7, §0.2 |
| 0.3 visibilite par surface | `[x]` | §0.3, 5 surfaces visibles |
| 1.1 JSON brut API des 3 matchs | `[x]` | dumps conserves, §1.1 ; premisse du plan refutee |
| 1.2 ampleur par mode | `[x]` | mesure EXACTE sur les 1 934 matchs au lieu de l'heuristique prevue (devenue sans objet apres 1.1) ; §1.2 |
| 1.3 oracle croise film | `[x]` | 20 artefacts confrontables sur 35, 18/20 exacts ; §1.3 |
| 1.4 verdict | `[x]` | issue (a), §1.4 |
| 2.1 mutualiser l'extraction | `[x]` | `ExtractTeamScoresByID` exportee, 1 appelant de production migre, aucun fichier ajoute a la racine de `internal/sync/` |
| 2.2 CLI — entree et liste | `[x]` | `--ids-file` (defaut = TSV du lot, colonne `match_id` par son NOM) + `--match` |
| 2.3 CLI — decision et ecriture | `[x]` | `UPDATE` row-by-row sous lease `KindSharedMatches`, `--apply` requis, gardes NULL / negatif / bornes / FFA |
| 2.4 tests unitaires | `[x]` | 24 cas table-driven, sans reseau ni base ; aucun test d'ecriture necessaire (la decision est pure, l'`UPDATE` est une ligne) |
| 2.5 gates | `[x]` | tableau §Gates rejoues, partie phase 2 |
| 2.6 section « Correctif pret » | `[x]` | commandes du jour J + note prod VPS |

**Phase 2 = code seulement, par arbitrage du 2026-08-24.** La CLI existe et est testee ;
elle **n'a ete executee contre aucune base**. Aucune ecriture DuckDB, aucun changement de
sync ni d'affichage n'a ete effectue par ce lot. La seule modification de code de production
est l'EXPORT (renommage) de `extractTeamScoresByID` : aucun changement de comportement, la
sync lit toujours le meme champ de la meme facon.
