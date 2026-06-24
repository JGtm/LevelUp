# Plan refonte title-agnostic — parité H5 (audit ultracode 2026-06-24)

Issu du workflow `h5-title-agnostic-audit` (6 dimensions, 47 findings). Branche `feat/h5-enrichment-parity`.

## Racine causale
1. **Divergence sync** : HINF → `sync.SyncEngine` (a `WithPostSyncRunner`) ; H5 → `livesync.Runner.RunDelta` (aucun hook progression/prestige/career) → couche Ascension morte pour H5.
2. **`defaultProgressionTitleSlug()` = `halo_infinite`** figé au boot dans handlers + post-sync background, au lieu de `ctxkeys.TitleSlug(ctx)` / `pdb.TitleSlug` par requête.

## Chantiers (ordre de dépendance)
- **C0 — Capabilities** (prérequis) : ajouter `CapabilityKey` `progression.tracking`, `prestige.engine`, `coach.proactive`, `notifications.delta`, `career.persist`. Déclarer dans `capabilities.toml` (HINF supported, H5 not_exposed→bascule au fil). Fichiers : `internal/games/adapter.go:39-60`, `capabilities.go:13-32`.
- **C1 — Hook post-sync dans le runner H5** (fix dominant) : `Deps.RunProgression` appelé dans `RunDelta` après `PostScore` si inserted>0, gaté `progression.tracking`. `runner.go:51-74/:154`, `wire.go::newHalo5Runner`, modèle `sync/engine.go:206`, `api/post_sync_deltas.go`.
- **C2 — Titre par requête** : éradiquer `defaultProgressionTitleSlug()` (`post_sync_deltas.go:115-121`), handlers lisent `pdb.TitleSlug`/`ctxkeys.TitleSlug`. `server.go:574-1557`, `sync_handler.go:427-433` (bgCtx), `progression_backfill_provider.go:35`.
- **C3 — career_progression H5** : `PersistPerMatchSR` (frère de `PersistPerMatchCSR`) dans `PostScore` → `InsertCareerProgressionPartial` depuis carnage `XpInfo.SpartanRank/TotalXP`. Gaté `career.persist`. `csr_match.go`, `dto_carnage.go:97`, `career_sr.go::applySpartanRank`, `adapter_data.go:270`.
- **C4 — Succès Xbox H5** : schéma `xbox_achievement_definitions` en metadata commune (gaté `achievements`) + inverser `metadata_test.go:77-91` ; CLI `RunForTitleDB` (pas `RunForDB`) ; gate `runAchievementsSync` ; warmer threade `titleID` (`achievements.go:278`) ; catégorisation `achievement_categories_halo_5.go` (workflow). XboxTitleID h5=219630713 déjà OK.
- **C5 — Config H5** (zéro Go) : créer `config/titles/halo_5/milestones/catalog.toml` (omettre `combat_*` data-limited). Lu auto par `RegisterMilestonesSeedMigration`.
- **C6 — Constantes partagées** : `225.0` câblé (`skill_rating.go:329` computeCombatYield → `games.EffectiveHpToKill`, garder 225 figé QUE sur sous-chemin LUSR `:245`) ; perf chain title-aware (`skill_config.go:181`→`GetLUSRChainForTitle`) ; Steaktacular ID par titre (`comeback.go:36`) ; URL Waypoint par titre ; mode_category resolver (large, différable).
- **C7 — Slug literals résiduels** (value low) : `world_player_stats_repo.go:54`, garder les défauts légitimes (DefaultSlug, adapter identity).

## Quick wins (small+high) en premier : C2.1 (bgCtx title sync_handler:427), C6.1 (225 param), C2.3, C4.2 (RunForTitleDB), C4.4 (warmer titleID), C5 (catalog.toml).

## Risques
- HINF byte-identique : 225 figé sur LUSR `:245` ; tests régression HINF avant merge.
- Prestige bundle mono-titre (`prestige_setup.go:57`) : NE PAS monter pour H5 tant que `prestige.engine: not_exposed` (large, différer).
- Notifications delta : gater chaque bloc par capability (sinon routes mortes H5 /season-pass /citations).
- Data-limited H5 : omettre `combat_*`/accuracy des milestones jusqu'à calibration.
- ADR 0008 : jamais `RunForDB`/`DefaultSlug` pour provisionner metadata H5.

Détail complet : sortie workflow `wf_d9fb4281-d47`.
