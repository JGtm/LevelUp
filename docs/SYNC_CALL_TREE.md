# Arbre d'appels — Synchronisation

Trace complète du chemin d'exécution déclenché par le bouton de synchronisation dans la sidebar,
du clic utilisateur jusqu'au `st.rerun()` final.

Fichiers clés : `streamlit_app.py`, `src/ui/sync.py`, `src/ui/_sync_duckdb_ops.py`,
`src/data/sync/engine.py`.

Phases appliquées : A (statut sync), B (HEAD-first delta), C (token_scope), D (mtime conditionnel),
E (cleanup), F (sync_metadata player DB), G.1 (event loop unique), G.3 (dead code supprimé),
H (transactions batch shared), I (_run_post_sync_pipeline extraction), J (shared R/O fanout).

```text
[sidebar_sync_button] (streamlit_app.py)
  │
  ├─ st.session_state[IS_SYNCING] = True
  ├─ save_filter_preferences()
  │
  └─ sync_all_players_duckdb()          (src/ui/sync.py)
       │
       ├─ Lecture db_profiles.json → liste des joueurs
       ├─ SyncLock(timeout=0)  (filelock data/.sync.lock)
       │
       └─ asyncio.run(_sync_all_players_loop_async())  ← G.1: un seul event loop
            │  (src/ui/_sync_duckdb_ops.py)
            │
            ├─ event=async_loop_start players=N
            ├─ _activate_sync_mode()
            │     ├─ begin_sync_mode()  (flag threading.Event sur le repo)
            │     ├─ get_cached_repository_st.clear()
            │     ├─ release_all_db_connections()
            │     └─ gc.collect() + sleep(0.3)
            │
            ├─ ∀ joueur dans profiles:
            │     │
            │     └─ await sync_player_duckdb_async()  ← G.1: await (pas asyncio.run)
            │                │
            │                ├─ SyncOptions(max_matches=200, parallel_fetch=15, parallel_matches=10)
            │                │
            │                └─ _execute_sync() → _run_sync_engine()
            │                      │
            │                      └─ DuckDBSyncEngine.__init__()
            │                           │  ↳ 3 connexions DuckDB: player(RW), shared(RW), pve(RW)
            │                           │  ↳ auto-résolution XUID (sync_meta → v_gamertag_lookup)
            │                           │
            │                           └─ engine.sync_delta(options)
            │                                │
            │                                └─ _sync_internal()
            │                                     │
            │                                     ├─ get_tokens_from_env() si tokens=None
            │                                     ├─ event=token_resolved scope=any|player ← C
            │                                     │
            │                                     ├─ HEAD check (delta mode) ← B.1
            │                                     │     ↳ API(count=1) vs DB latest → short-circuit si identique
            │                                     │
            │                                     ├─ _load_existing_match_ids()  ← B.3
            │                                     │     ↳ LEFT JOIN shared × player_match_enrichment × personal_score_awards
            │                                     │
            │                                     ├─ create_api_client(rps=15) → SPNKrAPIClient
            │                                     │
            │                                     ├─ _process_matches()
            │                                     │    │
            │                                     │    ├─ BEGIN TRANSACTION sur shared ← H.1
            │                                     │    │
            │                                     │    ├─ BOUCLE par batch de 25:
            │                                     │    │     ├─ get_match_history(start, count=25)
            │                                     │    │     ├─ Filtrage delta (arrêt au 1er known)
            │                                     │    │     └─ asyncio.gather(*_bounded(mid) for mid in new_ids)
            │                                     │    │           │                    ↑ Semaphore(fetch_slots)
            │                                     │    │           │
            │                                     │    │           └─ _process_single_match()
            │                                     │    │                 ├─ CHECK shared.match_registry
            │                                     │    │                 ├─ si connu → _process_known_match()
            │                                     │    │                 │     ↳ backfill sélectif (participants, events, medals)
            │                                     │    │                 │     ↳ _write_player_enrichments()
            │                                     │    │                 └─ si nouveau → _process_new_match()
            │                                     │    │                       ↳ extract + _insert_new_match_shared()
            │                                     │    │                       ↳ _save_player_data_new_match()
            │                                     │    │                       ↳ weapon_kills si activé
            │                                     │    │
            │                                     │    ├─ _maybe_batch_commit() tous les N matchs ← H.1
            │                                     │    │     ↳ COMMIT player + COMMIT shared + BEGIN TRANSACTION shared
            │                                     │    │
            │                                     │    └─ finally: COMMIT final sur shared ← H.1
            │                                     │
            │                                     └─ _run_post_sync_pipeline() ← I
            │                                           ├─ _refresh_aggregates_async() (si inserted > 0)
            │                                           ├─ _run_career_rank_if_needed()
            │                                           ├─ _save_sync_metadata() + conn.commit()
            │                                           ├─ _run_post_sync_compute() (si inserted > 0)
            │                                           │     ├─ citations → run_in_executor (thread)
            │                                           │     ├─ batch_compute_performance_scores()
            │                                           │     ├─ backfill_sessions_for_player()
            │                                           │     └─ _compute_dominance_post_sync()
            │                                           ├─ _detach_shared_from_player_conn()
            │                                           ├─ _run_lusr_post_sync()
            │                                           └─ _enrich_other_registered_players() (fan-out)
            │                                                 ↳ shared R/O (J) + ∀ autre joueur:
            │                                                     ├─ batch_compute_performance_scores()
            │                                                     ├─ backfill_sessions_for_player()
            │                                                     └─ backfill_citations_for_player()
            │
            ├─ os.utime() sur player DB + shared DB (invalidation cache mtime, si ok=True) ← D
            └─ _deactivate_sync_mode()

  ├─ st.session_state[IS_SYNCING] = False
  ├─ _send_sync_discord_notification()
  ├─ invalidate_after_sync()  (bust cache_buster + supprime clés filtre)
  └─ st.rerun()
```
