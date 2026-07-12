# Archive squash — baseline player v1

> Chantier N4 — plan `.ai/PLAN_MIGRATION_SQUASH_BASELINE_2026-07.md`.
> Date du squash : 2026-07-12. Commit d'origine (état PRÉ-squash des sources) :
> `9296496c9ad4e3c554d3f0d4371a878d99a6e8ac`.

## Ce que remplace la baseline

`create_baseline_player_v1` (`apps/go-api/internal/games/halo_infinite/migrations/steps_player_baseline.go`)
émet, sur une DB player VIERGE, le schéma cumulé « à plat » des **33 steps title-owned
contigus** ci-dessous (borne figée M3a : bloc `create_base_player_schema` →
`player_append_only_csr_snapshots_v1` ; le 1er step GLOBAL suivant,
`player_append_only_match_citations_v1`, N'EST PAS traversé — DM-4).

Ordre canonique (order.go, pré-squash) :

1. create_base_player_schema
2. add_engagement_score_columns_to_player_match_enrichment
3. create_engagement_coefficients_table
4. repair_engagement_coefficients_primary_key
5. add_engagement_pace_columns_to_player_match_enrichment
6. create_engagement_response_bins_table
7. add_bot_teammate_column
8. add_career_progression_sequence
9. add_career_identity_assets
10. add_career_banner_image
11. add_career_last_fetch_status
12. add_challenge_snapshots
13. add_challenge_snapshots_render_columns
14. add_challenge_snapshots_display_path
15. add_battlepass_snapshots
16. add_dominance_flag_column
17. add_media_like_columns
18. add_media_capture_start_utc
19. add_performance_score
20. add_player_performance_indexes
21. add_pme_session_label
22. add_pme_session_index
23. add_skill_rating_table
24. fix_mv_session_stats_varchar
25. add_match_exclusion_flag
26. add_player_privacy_state
27. drop_media_from_player_db
28. add_player_achievements
29. fix_match_citations_schema
30. cleanup_spartan_customization_garbage_urls
31. add_msr_measurement_matches_remaining
32. player_add_expected_win_prob
33. player_append_only_csr_snapshots_v1  ← SENTINELLE DM-5

## Provenance des sources (fonctions Go pré-squash)

- `playerBaseSteps()` (`steps_player_base.go`) : steps 1, 7-31.
- `playerSteps()` (`steps_player.go`) : steps 2-6 (famille engagement).
- `playerMatchSkillRankSteps()` (`steps_player_match_skill_rank.go`) : step 32.
- `appendOnlyMiscSteps()` (`steps_appendonly_misc.go`) : step 33 (no-op sur DB vierge :
  `player_csr_snapshots` n'existe pas encore dans le bloc → aucune table produite).

Les 4 fichiers dans `source/presquash_*.go` sont les versions EXACTES (git `HEAD` ci-dessus)
avant retrait. `git log` conserve l'historique complet ; cette archive est une commodité d'audit.

## Preuve zéro-perte

`player_block_golden.snapshot` = `SchemaSnapshot` du schéma cumulé des 33 steps, capturé de
l'historique réel AVANT retrait. Le test d'intégration
`TestSquashInvariant_PlayerBaselineEquivalent`
(`squash_invariant_test.go`) prouve à chaque CI que `SchemaSnapshot(baseline) == golden`
octet pour octet. Les steps player POSTÉRIEURS à la borne étant inchangés, l'égalité au
niveau bloc implique l'égalité du provisioning player complet (preuve compositionnelle).

## DM-5 (équivalence ledger)

Une player DB EXISTANTE (prod) porte déjà les 33 noms dans `schema_migrations`. Le champ
`SupersededByAll` de la baseline liste ces 33 noms ; le runner (`registry.go`,
`supersededBaselineSatisfied`) considère la baseline comme SATISFAITE dès que la sentinelle
(`player_append_only_csr_snapshots_v1`) est présente → l'enregistre SANS rejouer son DDL.
Tests : `TestDM5_SupersededBaselineSkipsDDLWhenSentinelApplied` /
`TestDM5_BaselineDDLRunsOnVirginDB` (`internal/migration/squash_dm5_test.go`).
