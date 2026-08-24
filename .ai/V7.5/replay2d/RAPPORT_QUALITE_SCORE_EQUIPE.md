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

(en cours)
