# Installation Guide — LevelUp

French version: [FR/INSTALL.md](FR/INSTALL.md)

This guide covers the recommended local setup on Windows/macOS/Linux.

## Requirements

- **Python**: 3.10+ (recommended: 3.12)
- **Git**

## Local install (recommended)

```bash
# Clone
git clone https://github.com/JGtm/LevelUp_with_SPNKr.git
cd LevelUp_with_SPNKr

# Create a virtual environment (at repo root)
python -m venv .venv

# Activate
# Windows (PowerShell)
.venv\Scripts\Activate.ps1

# Windows (cmd)
.venv\Scripts\activate.bat

# Linux/macOS
source .venv/bin/activate

# Install
pip install -e .
```

## Environment healthcheck

Run this before troubleshooting import / interpreter issues:

```bash
python scripts/check_env.py
```

## Next steps

- Configure the Halo API tokens: [CONFIGURATION.md](CONFIGURATION.md)
- Sync your data: [SYNC_GUIDE.md](SYNC_GUIDE.md)
