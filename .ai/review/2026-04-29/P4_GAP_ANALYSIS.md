# P4 Gap analysis — domain.*MatchRow → canonical (revue 2026-04-29)

## Synthese (5 lignes max)

L'extension reellement necessaire de `canonical.PlayerMatchRow` est **petite** (5 a 8 champs additionnels confirmes universels). La quasi-totalite des champs `*FR`, `*Label`, `*ImageURL`, `Skill*Label` Halo-specifiques doivent transiter par `TitleSemanticAdapter` (deja designe par ADR 0002) ou `TitleAssetURLAdapter` (couche 3 du plan finition multi-titres). Les valeurs brutes de skill rating (CSR/LUSR + tier code + sub-tier + delta) sont universelles FPS/PvP competitif et appartiennent au canonical via un nouveau sous-struct `Self.SkillRating` ou un `Enrichment.SkillSnapshot`. Les scores d'equipe `Team0Score/Team1Score` sont remplaces par `MatchSummary.Teams[]` (deja existant via `MatchDetail`). Les champs `Pair*` et `IsFirefight` sont remplaces par `MatchSummary.GameVariant` + `MatchType` + `IsPvE` (deja en place). Verdict : pilote home faisable avec **une seule extension canonical** (skill rating snapshot) plus injection systematique de `TitleSemanticAdapter`/`TitleAssetURLAdapter`.

## Tableau exhaustif HomeMatchRow (50 champs)

| Champ | Type Go | Cible | Justification |
|---|---|---|---|
| MatchID | string | canonical.Summary.MatchID | deja present |
| StartTime | time.Time | canonical.Summary.StartedAtUTC | deja present |
| MapID | string | canonical.Summary.Map.ID | deja present (AssetReference) |
| MapName | string | canonical.Summary.Map.DefaultLabel ou Labels["en"] | deja present |
| MapNameFR | string | canonical.Summary.Map.Labels["fr"] | deja present (loader le remplit) |
| PairID | string | A SUPPRIMER (Halo-specific concat) | redondant avec GameVariant.ID + Map.ID |
| PairName | string | A SUPPRIMER ou TitleSemanticAdapter | composite Halo-only ; reconstructible cote service via Map+GameVariant labels |
| PairNameFR | string | A SUPPRIMER ou TitleSemanticAdapter | idem PairName |
| GameVariantID | string | canonical.Summary.GameVariant.ID | deja present |
| GameVariantName | string | canonical.Summary.GameVariant.DefaultLabel | deja present |
| GameVariantNameFR | string | canonical.Summary.GameVariant.Labels["fr"] | extension de label, couvert |
| PlaylistID | string | canonical.Summary.Playlist.ID | deja present |
| PlaylistName | string | canonical.Summary.Playlist.DefaultLabel | deja present |
| PlaylistNameFR | string | canonical.Summary.Playlist.Labels["fr"] | deja present |
| IsFirefight | bool | canonical.Summary.IsPvE (alias) ou MatchType==Firefight | deja present |
| IsRanked | bool | canonical.Summary.IsRanked | deja present |
| SessionLabel | *string | canonical.Enrichment.SessionLabel | deja present |
| IsWithFriends | bool | canonical.Enrichment.IsWithFriends | deja present |
| Outcome | int | canonical.Self.Outcome (Outcome enum) | deja present (conversion int->enum dans loader) |
| TeamID | int | canonical.Self.TeamID | deja present |
| Team0Score | int | canonical.Summary.Teams[].Score | a EXPOSER : `MatchSummary` n'a pas `Teams`, seulement `MatchDetail` |
| Team1Score | int | idem | idem |
| DominanceFlag | int | canonical.Enrichment.DominanceFlag | deja present (DominanceFlag) |
| Kills | int | canonical.Self.Kills | deja present |
| Deaths | int | canonical.Self.Deaths | deja present |
| Assists | int | canonical.Self.Assists | deja present |
| KDA | *float64 | canonical.Self.KDA | deja present |
| Ratio | *float64 | canonical extension : `Self.KDR` | MANQUANT en struct (FieldKDR existe en FieldKey mais pas projete) - AJOUTER |
| Accuracy | *float64 | canonical.Self.Accuracy | deja present |
| AvgLifeSeconds | *float64 | canonical.Self.AvgLifeSeconds | deja present (champ etendu) |
| TimePlayedSecs | *int | canonical.Self.TimePlayed | deja present |
| DamageDealt | *float64 | canonical.Self.DamageDealt | deja present (typage *int dans canonical, MISMATCH a aligner) |
| DamageTaken | *float64 | canonical.Self.DamageTaken | idem mismatch type |
| TeamMMR | *float64 | canonical.Enrichment.TeamMMR | deja present |
| EnemyMMR | *float64 | canonical.Enrichment.EnemyMMR | deja present |
| PerformanceScore | *float64 | canonical.Enrichment.PerformanceScore | deja present |
| SkillRatingValue | *float64 | canonical EXTENSION : `Enrichment.SkillSnapshot.RatingValue` | universel FPS/PvP -> AJOUTER |
| SkillRatingType | string | canonical EXTENSION : `Enrichment.SkillSnapshot.RatingType` (RatingType enum) | enum deja existant -> AJOUTER champ |
| SkillTier | *string | canonical EXTENSION : `Enrichment.SkillSnapshot.TierCode` | code stable cross-titre (ex "diamond") -> AJOUTER |
| SkillSubTier | int | canonical EXTENSION : `Enrichment.SkillSnapshot.SubTier` | universel competitif -> AJOUTER |
| SkillTierLabel | *string | TitleSemanticAdapter.Ranks() ou .Outcomes() | label localise = i18n -> SEMANTIC |
| SkillRatingDelta | *float64 | canonical EXTENSION : `Enrichment.SkillSnapshot.Delta` | delta points = universel -> AJOUTER |
| SkillPlaylistGroup | *string | canonical EXTENSION : `Enrichment.SkillSnapshot.PlaylistGroup` | groupe normalise (ex "ranked-arena") universel -> AJOUTER |
| SkillRankImageURL | *string | TitleAssetURLAdapter.CSRRankImageURL(tier, subTier) | asset URL Halo-specific -> ASSET ADAPTER |
| RankInTeam | *int | canonical.Self.RankInMatch | RankInMatch existe (rang global match) ; RankInTeam est plus fin -> renommer ou ajouter `Self.RankInTeam` |
| HeadshotKills | int | canonical.Self.HeadshotKills | deja present |
| PerfectKills | int | canonical.Self.PerfectKills | deja present (champ etendu) |
| MaxKillingSpree | *int | canonical.Self.MaxKillingSpree | deja present |

## Tableau SynthesisMatchRow (14 champs)

| Champ | Type | Cible | Justification |
|---|---|---|---|
| MatchID | string | canonical.Summary.MatchID | deja present |
| StartTime | time.Time | canonical.Summary.StartedAtUTC | deja present |
| Outcome | int | canonical.Self.Outcome | deja present |
| Kills | int | canonical.Self.Kills | deja present |
| Deaths | int | canonical.Self.Deaths | deja present |
| KDA | *float64 | canonical.Self.KDA | deja present |
| IsWithFriends | bool | canonical.Enrichment.IsWithFriends | deja present |
| Accuracy | *float64 | canonical.Self.Accuracy | deja present |
| TimePlayedSecs | *int | canonical.Self.TimePlayed | deja present |
| PerformanceScore | *float64 | canonical.Enrichment.PerformanceScore | deja present |
| SessionLabel | *string | canonical.Enrichment.SessionLabel | deja present |
| IsRanked | bool | canonical.Summary.IsRanked | deja present |
| IsFirefight | bool | canonical.Summary.IsPvE | deja present |
| PlaylistName | string | canonical.Summary.Playlist.DefaultLabel | deja present |

**Verdict Synthesis** : 100% des champs deja couverts par le canonical existant. Migration sans extension.

## Tableau StatsMatchRow (24 champs)

| Champ | Type | Cible | Justification |
|---|---|---|---|
| MatchID, StartTime, Outcome | - | canonical.Summary/Self | deja present |
| Kills, Deaths, Assists | int | canonical.Self.* | deja present |
| KDA, Accuracy | *float64 | canonical.Self.* | deja present |
| PersonalScore | *int | canonical.Self.PersonalScore | deja present |
| DamageDealt, DamageTaken | *float64 | canonical.Self.* | mismatch type *int->*float64 a corriger en P4 |
| TimePlayedSeconds | *int | canonical.Self.TimePlayed | deja present |
| TeamMMR, EnemyMMR | *float64 | canonical.Enrichment.* | deja present |
| KillsExpected, DeathsExpected | *float64 | canonical EXTENSION : `Enrichment.SkillSnapshot.KillsExpected/DeathsExpected` | deja dans `MatchSkillSnapshot` mais pas dans Enrichment. Soit copier dans Enrichment, soit exposer un pointeur vers `MatchSkillSnapshot` depuis Enrichment |
| Rank | *int | canonical.Self.RankInMatch | deja present |
| IsRanked | bool | canonical.Summary.IsRanked | deja present |
| PlaylistName | string | canonical.Summary.Playlist | deja present |
| PairName | string | A SUPPRIMER (Halo-only) | reconstruit cote service depuis GameVariant + Map |
| TeamID | *int | canonical.Self.TeamID | deja present |
| PerfScoreComputed | *float64 | canonical.Enrichment.PerformanceScore | deja present (meme champ) |
| SessionID, SessionLabel | *string | canonical.Enrichment.* | deja present |
| MedalExploitScore | *float64 | canonical EXTENSION ? ou `domain.MatchMetrics` calcule | composite analytique LevelUp -> garder en domain calcule, pas canonical |
| OffensiveConversion | *float64 | idem | calcul derive (formule ratio damage/kills+assists) -> reste en analysis/ |
| DefensiveResistance | *float64 | idem | idem |

**Verdict Stats** : 1 ajout (KillsExpected/DeathsExpected dans Enrichment ou via pointeur vers MatchSkillSnapshot). Le reste est deja couvert ou doit rester calcul derive en analysis/.

## Extensions canonical proposees (minimal)

Politique additive (ADR 0005 cite dans match.go). Toutes optionnelles via pointeurs.

1. **`MatchSummary.Teams []TeamSnapshot`** (deja existe sur `MatchDetail`, manque sur `MatchSummary`) — JUSTIFICATION : Team0Score/Team1Score sont universels (tout FPS team-based affiche le score d'equipe en headline). Ajouter le snapshot leger (sans Participants) suffit.

2. **`MatchParticipant.KDR *float64`** — JUSTIFICATION : `FieldKDR` existe en FieldKey enum mais le ratio K/D n'est pas projete dans le struct. Universel FPS PvP. (Alternative : laisser le service le calculer ; mais le repo Halo le stocke deja).

3. **Nouveau sous-struct `SkillSnapshot`** dans `PlayerMatchEnrichment` :
   ```go
   type SkillSnapshot struct {
       RatingType    RatingType    // "csr" | "lusr" (enum existant)
       RatingValue   *float64
       TierCode      *string       // "diamond", "onyx", code stable cross-titre
       SubTier       *int          // 1..6 ou nil pour Onyx
       Delta         *float64      // points gagnes/perdus ce match
       PlaylistGroup *string       // "ranked-arena", normalise
       KillsExpected *float64      // depuis MatchSkillSnapshot
       DeathsExpected *float64
   }
   // PlayerMatchEnrichment {... ; SkillSnapshot *SkillSnapshot}
   ```
   JUSTIFICATION : tout titre PvP competitif expose un rating + tier + sub-tier + delta. Les LIBELLES (`SkillTierLabel = "Diamant 3"`) restent dans `TitleSemanticAdapter.Ranks()`. L'image (`SkillRankImageURL`) reste dans `TitleAssetURLAdapter.CSRRankImageURL`. Ce decoupage suit ADR 0002 strictement.

4. **`MatchParticipant.RankInTeam *int`** (optionnel) — JUSTIFICATION : `RankInMatch` existe (rang global). `RankInTeam` est utilise par MatchCard front (ranking intra-equipe). Universel team-based. Si l'on veut eviter la prolifération, le service peut le calculer cote analysis depuis le scoreboard charge.

5. **Aligner les types `DamageDealt/DamageTaken`** : actuellement `*int` dans `canonical.MatchParticipant`, `*float64` dans `domain.HomeMatchRow`. Choisir `*float64` (plus general, autorise des dommages partiels Spartan III).

**Si le scope minimal est pris (pilote home)** : seuls (1) Teams sur Summary + (3) SkillSnapshot suffisent. (2) KDR et (4) RankInTeam peuvent etre calcules cote service en passe 1, ajoutes en passe 2.

## Patterns de mapping recommandes

### Service Home (pilote P4.1)

```go
// AVANT
matches, _ := s.repo.LoadHomeMatches(ctx)            // []domain.HomeMatchRow
hero := analysis.BuildHeroCard(matches, gamertag, totalMatches)

// APRES
rows, _ := s.matchesRepo.Load(ctx, port.PlayerMatchFilters{}) // []canonical.PlayerMatchRow
labels := s.semantic.Assets()                                 // i18n FR/EN deja resolus
assetURLs := s.assetURL                                       // TitleAssetURLAdapter

hero := analysis.BuildHeroCardFromCanonical(rows, gamertag, totalMatches, labels)
recent := analysis.BuildRecentMatchesFromCanonical(rows, favoriteIDs, locale, labels, assetURLs)
// SkillTierLabel resolu via labels.Ranks().Resolve(row.Enrichment.SkillSnapshot.TierCode, row.Enrichment.SkillSnapshot.SubTier, locale)
// SkillRankImageURL resolu via assetURLs.CSRRankImageURL(tier, subTier)
```

### Service Synthesis / Stats

Migration triviale : projection 1:1 depuis `canonical.PlayerMatchRow` (aucune extension). Le service ne consomme jamais de label localise -> pas besoin de TitleSemanticAdapter pour ces deux.

## Effort estime revise (vs PLAN_ACTION.md)

| Sub-phase | Plan original | Effort revise |
|---|---|---|
| Extensions canonical (Teams sur Summary + SkillSnapshot) | non chiffre | **0.5-1 j** (additif, code clair) |
| ADR 0011 (entree d'extension Skill) | non chiffre | **2 h** |
| P4.1 pilote home (refactor + adapters injectes) | 4-6 j | **5-7 j** (gap #7 + cablage TitleAssetURLAdapter + golden tests) |
| P4.2 14 services restants | 12-18 j | **10-14 j** (Synthesis, Stats, Career, Timeseries, Compare, Engagement, Citations triviaux ; MatchView + SquadV2 deja partiels ; Media/MediaIndex independants des MatchRow) |
| P4.3 suppression legacy | 1 j | **0.5-1 j** |

**Justification revision** : la part de l'extension canonical est negligeable (1j) car la majorite des champs Halo-specifiques disparait via `TitleSemanticAdapter`/`TitleAssetURLAdapter`. Le pilote home reste le plus couteux car il porte aussi le gap #7 (titleSlug) et le cablage assets adapter, mais le gain de connaissance se diffuse mecaniquement aux 14 autres. Les services Stats/Synthesis/Compare/Timeseries sont quasi-mecaniques (~0.3-0.5 j chacun).

## Risques identifies

1. **Skill rating cross-titre** : la modelisation `SkillSnapshot` suppose que tout titre PvP a `tier+sub_tier+delta`. Halo MCC pourrait avoir un autre modele (1-50 only). Mitigation : garder tous les champs en pointeur, documenter qu'aucun n'est obligatoire.

2. **Mismatch types DamageDealt/DamageTaken `*int` vs `*float64`** : casse silencieuse possible si un service utilise deja `*int`. Mitigation : grep `Self.DamageDealt` avant changement, faire un commit dedie d'alignement.

3. **`PairName` dans tests Halo-specifiques** : suppression frappe ~15-20 tests. Mitigation : creer un helper test `pairNameFromSummary(s.GameVariant, s.Map)` pour preserver la lisibilite des fixtures.

4. **`TitleSemanticAdapter.Ranks()` peut etre vide** (cas career_rank_translations vide cite dans adapter.go). Le service home a deja un fallback rankCatalog nil. Etendre ce fallback aux Skill tier labels : si `Ranks()` ne resoud pas le tier, retomber sur `TierCode` brut.

5. **Volume des `PairName` dans mappers SQL** : la query `playerMatchesBaseSelect` (player_matches_repo.go:171) ne SELECT pas pair_name actuellement. Les services qui en dependent (citations, engagement) doivent prouver qu'ils n'en ont pas besoin OU que `GameVariant.DefaultLabel + " on " + Map.DefaultLabel` est equivalent. Mitigation : audit ciblé (`grep PairName` dans services) avant P4.2.
