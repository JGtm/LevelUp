# Common commands — LevelUp

French version: [FR/COMMANDS.md](FR/COMMANDS.md)

## Run the app

```bash
python launcher.py run
```

## Sync

```bash
python scripts/sync.py --delta --gamertag YourGamertag
python scripts/sync.py --full --gamertag YourGamertag --max-matches 500
```

## Backfill

```bash
python scripts/backfill_data.py --player YourGamertag --sessions
python scripts/backfill_data.py --player YourGamertag --citations
```

## Backup / restore

```bash
python scripts/backup_player.py --gamertag YourGamertag
python scripts/restore_player.py --gamertag YourGamertag --backup ./data/backups/YourGamertag
```

## Tests

```bash
python -m pytest -q --ignore=tests/integration
```
