# ADR 0039 — Per-match summaries of the kill feed, written with the pass

Status: accepted (2026-10-21) — implementation in progress (plan `.ai/PLAN_PERF_PAGES_V801_2026-10-21.md`, steps 7 to 9): until it lands, readers still read the feed and the guard of Decision 8 does not exist yet.
Relates to: ADR 0019 (Collect → Persist), ADR 0026 (append-only + `_latest`; adds a fifth
current-version mechanism, "generation decided by a head"), ADR 0030 D-4 (raw reads guard),
ADR 0034 D-6 (one revision per thing that can change), ADR 0036 I2 (bounded windows) and its
Consequences ("materialising the `_latest` state waits for production evidence": this ADR is that
change, see Context).

## Vocabulary

- **Feed**: `match_kill_events`, one row per death; its view `match_kill_events_latest` keeps the
  latest **pass** (`decode_pass`) of each match.
- **Summary**: an aggregate of one pass of the feed, stored in a summary table (S0, S1, S2).
- **Head**: the one-row-per-match summary (S0). Its **generation** (`summary_pass`) decides which
  rows of S1 and S2 are current, exactly as `match_usage_films` decides for `match_usage_players`.
- **Row-level reader**: a reader that needs individual deaths (instants, coordinates, exact
  medians, per-death journals). It keeps reading the feed.

## Context

Every pass rewrites all the deaths of a match (ADR 0026), and the view keeps the latest pass with
the window `QUALIFY decode_pass = FIRST_VALUE(decode_pass) OVER (PARTITION BY match_id ORDER BY
written_at DESC, id DESC)` (`migration/steps_shared_kill_events.go`). Request-path readers count
those rows again on every page load: frags between two players are recounted by 7 readers, assists
per pair by 5, frags per weapon by 7, and "does this match have a readable feed" by 8. That is the
situation rule 6 of the project forbids (at the third copy of a pattern, centralise it behind a
guard), and every page pays the window and the per-row filters of the feed.

Measured on a copy of the production databases on 2026-10-21 (bench of `scripts/bench-pages.sh`,
2 DuckDB threads, 512 MB): after compaction of the superseded passes, the recounts still cost
250 to 1 080 ms on the match view (encounter statistics and encounter assists over the player's
history), 0.83 s on Compare, 55 to 205 ms (assisted frags) and 60 to 150 ms (favourite weapon) on
Home. Before compaction (7.97 M raw feed rows for 0.42 M latest rows), the same reads cost 1 to 4 s.
Compaction (now automatic, ADR 0026 amendment) removes the superseded passes; it does not remove the
recounting nor the seven copies. This ADR does.

## Decision

1. **The rule.** An aggregate of the feed that decomposes by match is computed ONCE, when its pass
   is written, and stored in a summary table. A reader in `platform/duckdb`, `api`, `service` or
   `analysis` reads the summary, never the feed, unless it is a row-level reader listed by query in
   a dated allowlist that can only shrink (Decision 8).

2. **Write points.** `insertKillEventRows` (`internal/persist/kill_events_persister.go`) is the only
   runtime writer of the feed; the film producer (`KillSourcePersister.PersistPass`), the credit
   producer (`persistCreditKillEvents`) and their callers (primary sync of every title, events
   completion, kill-source collector, `BatchBuilder`, backfill CLI) go through it. It computes the
   summaries from the rows of the pass AFTER the normalisations of `killEventArgs` (`""` → NULL,
   `SourceTag == 0` → NULL) and inserts them in the same transaction, with the same Go timestamp as
   the pass. The insertion of S0, S1 and S2 lives in ONE function, `insertKillSummaryRows`, called by
   `insertKillEventRows`, by the synthetic demo seed (`ops/seed_demo_synthetic_shared.go`, which
   inserts feed rows directly and must write their summaries too) and by the catch-up (Decision 5).
   Two historic one-shot migrations also inserted feed rows (`steps_shared_kill_events_credit_base`,
   `steps_shared_kill_events_from_pairs`): they never write summaries; the catch-up covers their
   passes.
   The summary function is total and returns no error: the only failures are database errors,
   which would fail the feed insert as well. In the primary sync the pass shares the match's
   transaction (`SharedPersister`): a database failure while inserting summaries drops that match
   from the cycle, as a feed failure does today. Every writer has an error-injection test.

3. **Three families.** Every summary row carries `id BIGINT` (own sequence, primary key),
   `match_id`, `summary_pass`, `summary_rev`, `src_decode_pass` and `written_at`; each table has the
   index (`match_id`, `written_at`). `S0_latest` keeps, per match, the head with the latest
   (`written_at`, `id`); `S1_latest` and `S2_latest` join their rows to `S0_latest` on (`match_id`,
   `summary_pass`) and expose the head's `publishable`. The views are recreated after any
   `ADD COLUMN` (a `SELECT *` view freezes its columns, `steps_shared_usage_summary.go`).
   - **S0 `match_kill_heads`**, one row per match and generation: `publishable` (a field of the
     batch, `KillSourceBatch.Publishable`, constant over the pass by construction), `n_rows`,
     `n_assist_known`, `n_killer_xuid` (rows with `feed_killer_xuid` not NULL), `n_src_killer` (rows
     with `source_tag` and `feed_killer_xuid` not NULL).
   - **S1 `match_kill_pairs`**, grain (`match_id`, `feed_killer_xuid`, `victim_xuid`, `source_tag`):
     `kills`. NULL killer, NULL victim and NULL source are kept as groups (weapon coverage counts
     killer-less rows, Home's film frags count bot victims, pair readers count source-less rows).
     Non-publishable matches are kept: pair and weapon readers do not filter on `publishable`.
     Pair readers sum over `source_tag`, weapon readers sum over `victim_xuid`. No gamertag: no
     aggregate reader reads one.
   - **S2 `match_assist_pairs`**, only rows with `publishable AND assist_known`, grain (`match_id`,
     `assist_xuid`, `assist_gamertag`, `feed_killer_xuid`, `feed_killer_gamertag`, and `victim_xuid`
     only when `assist_xuid` and `feed_killer_xuid` are both NULL): `n`, `n_stolen`
     (`assist_damage_pct > killer_damage_pct`), `sum_assist_pct` and `n_assist_pct` (an average is
     `sum / n`, NULL when `n = 0`, never 0 %), `n_low` (`< 25`), `n_mid` (`25..50`), `n_high`
     (`> 50`). Rows with no named assistant are kept (the Coordination block counts "measured,
     nobody assisted"); readers that need a named assistant or killer filter S2. Gamertags are part
     of the grain because the match view separates bot groups by gamertag.

4. **Contracts the summaries rely on.** A named assistant implies `assist_known` and an assist part
   implies an assistant gamertag: enforced at write time by `validateKillEventAssist`, and checked
   on the production copy of 2026-10-21 for every latest row, historic migrations included (0
   exceptions). `publishable` is a batch field. Sizes on that copy: S1 245 547 rows and S2 86 901
   for 419 858 latest feed rows (Halo Infinite), S1 96 178 for 268 337 (Halo 5). The gain is not
   fewer rows; it is that readers stop paying the feed's window and per-row filters.

5. **Revision and recomputation.** `summary_rev` changes whenever the pure function, its grain, its
   normalisations or the tier bounds (`domain.AssistTierLowMaxPct` / `AssistTierMidMaxPct`) change:
   a fingerprint test of the function, on the model of `decoder_rev_fingerprint_test.go`, fails
   otherwise. Readers never filter on `summary_rev`; the views serve the latest generation. A head
   is missing when the latest pass of a match has no head; it is stale when its `summary_rev` is not
   the current one, or when it summarises a pass other than the latest (a newer pass was written
   without its summaries). One SQL text defines this (`persist.PendingKillSummaryMatches`) for the
   catch-up and the gauge. `levelup backfill-match-summaries --title <slug>` writes the missing and stale
   generations from the latest pass read in the database (second adapter of the same pure function,
   kept equal to the in-memory one by a parity test), through `insertKillSummaryRows`, server
   stopped (same pattern as `backfill-vehicle-takes`). A revision bump is applied to the whole fleet
   before serving: recomputation on the local databases, then copy to the server (release sequence).

6. **A missing summary is a release gate.** A per-match reader that finds no head shows the match
   as not measured (the "analysis pending" state where the page has one), never as zero. A reader
   that sums over many matches cannot show that state: a match without a head is absent from the
   sum. The gauge "matches whose latest pass has no head, or a stale one" is therefore a release
   gate: it must be 0 on the databases copied to production, it is logged at WARN by the data
   health check when non-zero, outside that check's warnings total and published as the expvar
   `levelup.kill_summaries_pending` (computed asynchronously, not at boot, because it reads the feed's
   window over the whole table), and in-line writes keep it at 0 afterwards.

7. **Bounded reads (ADR 0036 I2 applies to the S views).** Before the first reader switches, the
   bounded-window test (`exigerFenetresBornees`) runs on the S views under a constant `VARCHAR[]`
   list of matches. Amended on 2026-10-21 with the measurement (`platform/duckdb/resumes_fenetres_test.go`):
   the filter does NOT cross the head's window when a daughter view (S1, S2) is read under a list,
   and the remedy "bind the list on the head view too" does not carry over, because the head's
   window sits inside the daughter view (a semi-join on the head view leaves the inner window
   whole; a daughter read raw is forbidden by ADR 0026). The head's window is therefore accepted
   under a list: it sees one row per match and generation (about 6,500 compacted rows on
   production), never the daughters' rows, and the automatic compaction bounds it. A read of ONE
   match (`match_id = ?`) and S0 itself stay bounded. The accepted window is a line of the I2
   exceptions of ADR 0036, and the test above holds its size. A reader reads an S view with a
   `match_id` list, or is a documented exception:
   Home's favourite weapon (filtered by killer over the player's history; S1 by killer is a small
   scan) and `/monitoring/weapon-coverage` (whole fleet, operator only). The head has one row per
   match and generation, so an unpushed head window costs little. A correlated `EXISTS` on S0
   (Tactical coverage) copies the list inside the `EXISTS`, as the current query does.

8. **Guard.** An archlint test, scoped like `TestNoRawAppendOnlyReads` (`platform/duckdb`, `api`,
   `service`, `analysis`; `sync`, `cmd`, `scheduler`, `ops`, `migration` and `persist` are outside
   it), fails when the identifier `KillEventsCanonicalTable` or the literal
   `match_kill_events_latest` (comments stripped) appears in a query that is not in the dated
   allowlist of row-level readers: Q20 `GetMatchKVPairs`, Q21b `GetMatchKillSources`, Q21c
   `GetMatchKillAssists`, Q32e `LoadSquadKillLog`, `qSoloFrags`, the `highlight_events` fallbacks
   (`loadSyntheticKillEvents`, Q32c), the measured-kill join (`kill_measured.go`), kill distance,
   weapon range, match range rows, solo lives, the Tactical readers, the Halo 5 events source and
   the name directory (`analysis/identity.go`, names only). A permanent coherence test checks, for
   every match of a corpus, that the sum of `kills` in S1 equals the row count of the latest pass.

9. **Housekeeping.** The three tables are in the compaction registry (`dernierePasse` for S0, the
   parent view's generation for S1 and S2, view signature `INNER JOIN match_kill_heads_latest ON
   match_id AND summary_pass`), in the ART guards' protected tables
   (`sync/no_art_patterns_test.go`, `sync/append_only_state_guard_test.go`), in the shared snapshot
   export (`ops/snapshot_export.go`) and in the demo seed (identity columns anonymised,
   `TestSharedTablesDeclareEveryIdentityColumn`). One generic migration creates them, ordered after
   the `steps_shared_kill_events_*` migrations, and `EnsureMatchKillEvents` ensures them; titles
   without a film get empty tables, and readers keep returning `ErrCapabilityNotSupported` where
   they do today. No branch on the title slug. Summary code lives in new files
   (`kill_events_persister.go` is above the 500-line limit).

## Readers served

| Family | Readers |
|---|---|
| S0 | Tactical coverage (`QTacticalUnivers.mesure`), solo non-publishable journal (`qSoloJournalNonPubliable`), match view assist scope (Q21d), squad measured matches (Q32d), assist-carrying matches of Relations (Q28c) and Home (Q26l), match range denominator (`matchRangeKillsTotalSQL`), Emprise read flags (`EventsRead`, `LoadJournalWeaponKills` `Read`) |
| S1 pairs | match view encounter statistics (Q23b), Career top encounters (Q26) and rivals (Q27), Relations (Q28), rival timeline (Q30), Compare and Explorer frags between players (`QKillsBetweenPlayers*`), Home film frags (Q26l) |
| S1 weapons | match view weapons (`qMatchKillSourceWeapons`), Explorer top weapons and Home favourite weapon (`weaponKillsFromSourceForPlayer`), `LoadWeaponKillsAggregated`, `LoadKillSourceCategoryKills`, Emprise vehicles (`qSquadVehicleFrags`) and special weapons (`LoadJournalWeaponKills`), weapon coverage, citations weapons (`sync/citations_weapons_source.go`) |
| S2 | match view assist pairs (Q21d), Relations and match view encounter assists (Q28c), Home assisted tiers (Q26l), Coordination support (`QCoordinationAppuis`), squad assist pairs (Q32d) |

Sync-side readers of the feed (`sync/enrichment_discipline.go`, `sync/engagement.go`,
`sync/events_replay.go` and others) are outside this decision and outside the guard.

## Rollout

Migration and summary writes ship together; the catch-up runs on the local databases (both titles)
until the gauge is 0, then the databases are copied to production with the release. Existing shared
snapshots lack the new tables and are refused until the next cut (live reads meanwhile, the
documented fallback). A database restored from before this ADR has no heads: the catch-up is
mandatory before serving it. A code rollback leaves the tables unread and is harmless.

## How to add a reader or an aggregate

- **A reader.** Is the figure an aggregate of the feed that decomposes by match? If not, read the
  feed and add the query to the allowlist of Decision 8, with its reason. If it is, read
  `S0_latest`, `S1_latest` or `S2_latest`, bound by a `match_id` list (Decision 7), apply the head's
  `publishable` when the figure names deaths, and keep the reader's duration section (ADR 0036 I6).
- **An aggregate or a column.** Change the pure function (one place), bump `summary_rev` (the
  fingerprint test says so), add the column through a migration that recreates the views, update
  the compaction signature, the snapshot export and the demo seed if a table is added, and
  recompute the fleet locally before the release (Decision 5).

## Consequences

- Every pass writes three more tables in its transaction (cost measured by the pass cost
  integration test before the first release).
- Pages that aggregate history read summaries filtered by `match_id`, whatever the number of passes.
- The summary tables grow by one generation per pass, like the feed; the automatic compaction of
  superseded passes keeps them lean.

## Alternatives considered

- **Bounded windows only (ADR 0036 I2).** Kept for row-level readers, but it leaves seven copies of
  the same count and a window per read.
- **A cache invalidated at sync (ADR 0036 I3).** Hides the cost after the first load, not the
  recount nor the copies.
- **Summaries computed by a separate post-sync job.** Opens a window where the feed and the
  summaries disagree and needs a marker; writing in the pass's transaction makes them agree.
- **`killer_victim_pairs`.** One row per frag with duplicates, deleted then inserted, without
  `publishable`, source or assistant, fed by the API: not a summary of the feed.
