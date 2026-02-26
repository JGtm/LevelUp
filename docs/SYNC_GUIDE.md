# Sync Guide — LevelUp

French version: [FR/SYNC_GUIDE.md](FR/SYNC_GUIDE.md)

This guide covers the recommended sync commands for Halo Infinite matches.

## Delta sync (daily)

Fetches only new matches since the last sync:

```bash
python scripts/sync.py --delta --gamertag YourGamertag
```

## Full sync (first import)

Fetches match history up to a limit:

```bash
python scripts/sync.py --full --gamertag YourGamertag --max-matches 500
```

## After sync: backfill (optional)

Use the backfill CLI to compute missing enrichments:

```bash
python scripts/backfill_data.py --player YourGamertag --sessions
python scripts/backfill_data.py --player YourGamertag --citations
```

## Troubleshooting

- Start with: `python scripts/check_env.py`
