# Sync Guide — LevelUp

French version: [FR/SYNC_GUIDE.md](FR/SYNC_GUIDE.md)

> How to synchronize your Halo Infinite matches with LevelUp.

## Sync Architecture

LevelUp v5.1 uses a **Shared Matches** architecture: match data is centralized in a shared database, while personal enrichments stay in the player's own database.

```
SPNKr API (Halo Infinite)
        │
        ▼
DuckDBSyncEngine
├── api_client.py      # Async API wrapper
├── transformers.py    # JSON → DuckDB rows
└── engine.py          # Orchestrator
        │
        ├─→ New match → shared_matches.duckdb
        │   ├── match_registry         (common match data)
        │   ├── match_participants     (all players, 31 columns incl. MMR)
        │   ├── highlight_events       (film events)
        │   ├── medals_earned          (medals)
        │   └── xuid_aliases           (xuid→gamertag mapping)
        │
        └─→ Enrichment → players/{gamertag}/stats.duckdb
            ├── player_match_enrichment (performance_score, session_id)
            ├── personal_score_awards   (objective awards)
            └── sync_meta              (sync state)
```

---

## Basic Commands

### Delta Sync (Incremental)

Fetches only new matches since the last sync:

```bash
python scripts/sync.py --delta --player YourGamertag
```

**Advantages:**
- Fast (a few seconds)
- Ideal for daily use
- Doesn't overload the API

### Full Sync (Complete)

Fetches all matches up to a limit:

```bash
python scripts/sync.py --full --player YourGamertag --max-matches 500
```

**When to use:**
- First import
- Recover missing history
- After a long period without syncing

### Adding a New Player

```bash
# By gamertag
python scripts/sync.py --add-player SpartanC

# By XUID
python scripts/sync.py --add-player 2533274823110022

# Add + immediately run a full sync
python scripts/sync.py --add-player SpartanC --full --max-matches 500
```

### Sync with Backfill

After syncing, you can automatically fill in missing data:

```bash
# Full backfill (all missing data)
python scripts/sync.py --delta --player YourGamertag --with-backfill

# Only compute missing performance scores
python scripts/sync.py --delta --player YourGamertag --backfill-performance-scores

# Only compute citations (local, no API call)
python scripts/sync.py --delta --player YourGamertag --with-citations
```

---

## Sync Options

| Option | Description | Default |
|--------|-------------|---------|
| `--player` | Player to sync (gamertag or XUID) | All players |
| `--add-player` | Add/update a player in `db_profiles.json` | — |
| `--delta` | Incremental mode (new matches only) | No |
| `--full` | Complete mode (all matches up to limit) | No |
| `--max-matches` | Max number of matches | 200 |
| `--match-type` | Match type (`all`, `matchmaking`, `custom`) | `matchmaking` |
| `--with-assets` | Download missing assets (medals, maps) | No |
| `--with-backfill` | Run a full backfill after sync | No |
| `--with-citations` | Compute citations after sync (local, no API) | No |
| `--backfill-performance-scores` | Compute missing performance scores | No |
| `--rebuild-cache` | Rebuild the MatchCache | No |
| `--apply-indexes` | Apply optimized indexes | No |
| `--stats` | Display DB statistics | No |
| `--no-discord` | Disable Discord notification for this run | No |
| `--verbose` | Verbose mode | No |

**Important:** All data is always fetched for each synchronized match:
- Stats (kills, deaths, assists, KDA, accuracy, etc.)
- Medals
- Personal scores
- Performance score
- Highlight events (kills/deaths from films)
- Skill/MMR (per-match skill data)
- XUID → Gamertag aliases

---

## Backfill (Standalone)

Use `backfill_data.py` to enrich data independently of sync:

### Common Commands

```bash
# All data for one player
python scripts/backfill_data.py --player YourGamertag --all-data

# All data for all players
python scripts/backfill_data.py --all --all-data

# Sessions + friend detection
python scripts/backfill_data.py --player YourGamertag --sessions

# Citations
python scripts/backfill_data.py --player YourGamertag --citations

# LUSR skill rating (local TrueSkill 2, no API)
python scripts/backfill_data.py --player YourGamertag --lusr

# CSR from API (ranked matches)
python scripts/backfill_data.py --player YourGamertag --csr

# LUSR + CSR combined
python scripts/backfill_data.py --player YourGamertag --skill-rank

# Performance scores
python scripts/backfill_data.py --player YourGamertag --performance-scores

# Core stats (accuracy, shots, damage, kills detail, KDA, etc.)
python scripts/backfill_data.py --player YourGamertag --core-stats

# PvE / Firefight stats
python scripts/backfill_data.py --player YourGamertag --pve-stats

# Dry-run (list only, no changes)
python scripts/backfill_data.py --player YourGamertag --dry-run
```

### Backfill Options Reference

| Option | Description | Needs API |
|--------|-------------|-----------|
| `--all-data` | Backfill all data types | Yes |
| `--medals` | Backfill medals | Yes |
| `--events` | Backfill highlight events | Yes |
| `--skill` | Backfill MMR/skill data | Yes |
| `--personal-scores` | Backfill personal score awards | Yes |
| `--performance-scores` | Compute performance scores | No |
| `--accuracy` | Backfill accuracy | Yes |
| `--shots` | Backfill shots_fired/shots_hit | Yes |
| `--damage` | Backfill damage_dealt/damage_taken | Yes |
| `--combat` | = accuracy + shots + damage | Yes |
| `--core-stats` | = combat + kills detail + KDA + time played | Yes |
| `--sessions` | Compute sessions + friend detection | No |
| `--citations` | Compute match citations | No |
| `--lusr` | Compute LUSR rating (local TrueSkill 2) | No |
| `--csr` | Backfill CSR from API (ranked) | Yes |
| `--skill-rank` | = lusr + csr | Mixed |
| `--pve-stats` | Backfill Firefight/PvE stats | Yes |
| `--participants` | Backfill match participants | Yes |
| `--killer-victim` | Backfill killer/victim pairs | No |
| `--aliases` | Update XUID aliases | Yes |
| `--assets` | Fetch names (playlist, map, game variant) | Yes |
| `--team-scores` | Populate team scores in match_registry | No |
| `--mode-category` | Recompute mode_category (local) | No |
| `--bot-detection` | Detect bot teammates | No |
| `--cleanup-player-dbs` | Remove legacy views/tables from player DBs | No |
| `--dry-run` | List only, no changes | — |

Most options have a `--force-*` variant to reprocess already-filled data.

---

## Sync via the Dashboard

### Sidebar Button

The dashboard displays:
- **Last sync**: Date and time
- **Matches**: Number of synced matches
- **Sync button**: Triggers a delta sync
- **Full button**: Triggers a full sync

### Start the stack before syncing

```bash
make dev
```

Then trigger a delta sync from the UI, or run:

```bash
python scripts/sync.py --delta --gamertag YourGamertag
```

---

## Troubleshooting

### Environment Check

```bash
python scripts/check_env.py
```

### Rate Limiting

If you receive a 429 error:

1. Wait a few minutes
2. Reduce `--max-matches`
3. Use `--delta` instead of `--full`

### Expired Token

```
Error: Authentication failed
```

**Solution:**
```bash
python scripts/spnkr_get_refresh_token.py
```

### Career Rank Not Synced (Player-gated Warning)

Some Halo Waypoint endpoints (career rank, customization) return 403 if the Spartan token
doesn't belong to the targeted player. Symptom in logs:

```
WARNING — No player token for 'YourGamertag' — career rank skipped.
```

**Solution:** add a per-player token in `.env.local`:

```env
SPNKR_OAUTH_REFRESH_TOKEN_YOURGAMERTAG=your_xbox_live_refresh_token
```

See [CONFIGURATION.md](CONFIGURATION.md#azure--spnkr) for details.

---

## Best Practices

| Usage | Frequency | Command |
|-------|-----------|---------|
| Active player | Daily | `--delta` |
| Casual player | Weekly | `--delta` |
| First import | Once | `--full --max-matches 1000` |

### Before a Gaming Session

```bash
python scripts/sync.py --delta --player YourGamertag
```

### After a Gaming Session

```bash
# Sync + full backfill
python scripts/sync.py --delta --player YourGamertag --with-backfill

# Or just performance scores
python scripts/sync.py --delta --player YourGamertag --backfill-performance-scores
```
