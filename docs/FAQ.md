# FAQ — LevelUp

French version: [FR/FAQ.md](FR/FAQ.md)

## Installation

### Which Python version should I use?

Python 3.10+ works; Python 3.12 is recommended.

### Imports fail / wrong interpreter

Run:

```bash
python scripts/check_env.py
```

Then ensure you created and activated the repo-level `.venv`.

## Configuration

### My token expired

Re-run:

```bash
python scripts/spnkr_get_refresh_token.py
```

## Sync

### What’s the difference between `--delta` and `--full`?

- `--delta`: only new matches (fast)
- `--full`: full history up to `--max-matches`
