# Installation Guide — LevelUp

French version: [FR/INSTALL.md](FR/INSTALL.md)

> Complete guide to install and configure LevelUp on your machine.

---

## Windows — Recommended install (no technical knowledge required)

LevelUp ships a one-click launcher that automates the entire setup.
**You do not need to know what Python is.**

### Step 1 — Download LevelUp

Go to the GitHub project page → green **Code** button → **Download ZIP**.
Extract the folder wherever you like (e.g. Desktop or `C:\LevelUp\`).

> If you know Git, you can also clone:
> ```bash
> git clone https://github.com/JGtm/LevelUp_with_SPNKr.git
> ```

### Step 2 — Double-click `LevelUp.bat`

The launcher does **everything automatically**:

1. Looks for Python on your PC
2. If not found → downloads and installs it via `winget` (Windows 10/11 — answer `Y`)
3. Creates an isolated environment (`.venv`)
4. Installs all dependencies
5. Starts the dashboard and opens your browser at `http://localhost:8501`

> **First launch**: 2–5 minutes (downloads). Subsequent launches: a few seconds.

### Step 3 — Setup Wizard in the browser

On the first launch, LevelUp detects it is not yet configured and shows a **guided wizard**.
Choose your path:

#### 🎮 Xbox Express (recommended — 2 steps)

The simplest path. The only unavoidable requirement: create a free Azure application
(Microsoft mandates this for access to the official Halo Infinite API).

**Wizard step 1 — Create an Azure application (free, no charge)**

> Azure is Microsoft's cloud service that manages Xbox authentication.
> LevelUp needs a dedicated "access key" per user.
> Registration is free; LevelUp uses no paid Azure services.

1. Go to [portal.azure.com](https://portal.azure.com) — sign in with your Microsoft/Xbox account
2. Search for **Microsoft Entra ID** → **App registrations** → **New registration**
3. Fill in:
   - Name: `LevelUp Halo`
   - Account type: *Personal Microsoft accounts only*
   - Redirect URI → **Web** → `http://localhost:8501`
4. Click **Register**
5. On the **Overview** page: copy the **Application (client) ID** (format `xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx`)
6. Go to **Certificates & secrets** → **New client secret** → give it a name → **Add**
   → copy the **Value** column immediately (it disappears if you navigate away)
7. Go to **API permissions** → **Add a permission** → **Microsoft Graph**:
   add `offline_access` and `User.Read`

Paste the Client ID and the Value into the wizard → LevelUp saves everything automatically.

**Wizard step 2 — Xbox sign-in with one click**

Click **"Sign in with Xbox"** → a Microsoft window opens → sign in with your Xbox account
→ LevelUp automatically retrieves your gamertag and XUID, creates your profile,
and stores the OAuth token in your database.

#### ☁️ Azure Manual (advanced — 3 steps)

Same Azure setup as above, but the refresh token is obtained manually
(use this if the automatic Xbox flow causes issues, e.g. reverse proxy):

```bash
python scripts/spnkr_get_refresh_token.py
```

This script opens a browser, authenticates you, and displays the token to copy into `.env.local`.

### Step 4 — Smoke test (automatic check on 20 matches)

After Xbox sign-in, the wizard automatically runs a **3-phase smoke test**:

| Phase | What happens |
|-------|--------------|
| 📡 Phase 1 — Sync | Synchronize 20 matches from the Halo API |
| ⚙️ Phase 2 — Enrichment | Compute scores, sessions, citations, LUSR/CSR, killer/victim pairs |
| 🔍 Phase 3 — Verification | Full integrity check of all tables (see below) |

**Checked tables (all required):**

| Table | Database | What is validated |
|-------|----------|-------------------|
| `match_registry` | shared | count > 0 |
| `match_participants` | shared | count > 0 + kills/deaths not NULL |
| `medals_earned` | shared | count > 0 |
| `killer_victim_pairs` | shared | count > 0 |
| `highlight_events` | shared | count > 0 (film clips) |
| `xuid_aliases` | shared | count > 0 |
| `player_match_enrichment` | player | count > 0 + session_id not NULL |
| `performance_score` | shared (via match_participants) | computed score > 0 |
| `match_citations` | player | count > 0 |
| `match_skill_rank` (LUSR/CSR) | player | count > 0 + LUSR/CSR present |
| `sessions` | player | count > 0 |
| `sync_meta` | player | count > 0 |
| shared↔player consistency | cross | counts consistent |

If any check fails, the test offers to **retry**. When everything is green, two options:

- **⚙️ Full sync** → navigates to the Settings page to fetch your complete history (recommended)
- **📊 Dashboard (20 matches)** → goes directly to the dashboard with the already-synced matches

---

## macOS / Linux

`LevelUp.bat` is Windows-only. On macOS/Linux:

1. **Install Python 3.10+** if not already available
   - macOS: `brew install python@3.12` or download from [python.org](https://www.python.org/downloads/)
   - Linux: `sudo apt install python3.12 python3.12-venv` (Ubuntu/Debian) or equivalent
2. In the extracted folder, create a virtual environment and install:
   ```bash
   python3 -m venv .venv
   source .venv/bin/activate
   pip install -e ".[spnkr]"
   ```
3. Start LevelUp:
   ```bash
   python launcher.py run
   ```
4. The browser opens at `http://localhost:8501` — the **Setup Wizard** appears and guides you
   through the same steps (Azure + Xbox sign-in + smoke test) as on Windows.

> **Note:** The automatic Python installer (`winget`) only works on Windows 10/11.
> On macOS/Linux you manage Python yourself, but everything after that (wizard, sync, dashboard)
> is fully cross-platform.

---

## Developer install

### Requirements
- Python 3.10+ (recommended: 3.12)
- Git

```bash
git clone https://github.com/JGtm/LevelUp_with_SPNKr.git
cd LevelUp_with_SPNKr

# Create virtual environment
python -m venv .venv

# Activate (Windows PowerShell)
.venv\Scripts\Activate.ps1
# Activate (Linux/macOS)
source .venv/bin/activate

# Full install (with dev tools)
pip install -e ".[dev,spnkr]"
```

### Environment healthcheck

```bash
python scripts/check_env.py
# or
python launcher.py doctor
```

### Tests

```bash
# Full suite
python -m pytest

# Excluding integration tests (faster)
python -m pytest --ignore=tests/integration

# Specific file
python -m pytest tests/test_duckdb_repository.py -v
```

### Update

```bash
git pull origin main
python launcher.py setup --update
```

See [CONFIGURATION.md](CONFIGURATION.md) for Azure token configuration.

---

## Docker install

### Requirements
- Docker Desktop installed
- Docker Compose v2 available (`docker compose version`)

### Pre-requisites: configuration files

Before the first `docker compose up`, make sure these files exist:

```bash
# If db_profiles.json does not exist yet
echo '{"profiles": {}}' > db_profiles.json

# If app_settings.json does not exist yet
echo '{}' > app_settings.json
```

> **Why?** Docker bind-mount creates a *directory* (not a file) if the source does not exist,
> which would crash the app.

### Start with Docker Compose

```bash
# Build and start
docker compose up --build

# In the background
docker compose up -d

# View logs
docker compose logs -f

# Stop
docker compose down
```

### Docker volumes

| Host path | Container path | Description |
|-----------|----------------|-------------|
| `./data` | `/app/data` | DuckDB v5 data (read/write) |
| `./db_profiles.json` | `/app/db_profiles.json` | Player profiles |
| `./app_settings.json` | `/app/app_settings.json` | Application settings |

---

## Troubleshooting

### Full diagnostic

```bash
python launcher.py doctor
```

### "Module not found" error

```bash
python launcher.py setup
```

### DuckDB version error

```bash
python -c "import duckdb; print(duckdb.__version__)"
# Must be >= 1.4.0
python launcher.py setup --update
```

### Expired OAuth token

In the app → **Settings** → **Xbox connection** → **Reconnect**.
The token is stored in `data/players/<gamertag>/stats.duckdb` (table `sync_meta`).

### Permission Denied (Windows / PowerShell)

```powershell
# Allow PowerShell scripts (once only)
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
```

---

## Folder structure after install

```
LevelUp/
├── .venv/                         # Python virtual environment
├── data/
│   ├── players/
│   │   └── MyGamertag/
│   │       └── stats.duckdb       # Per-player enrichments
│   └── warehouse/
│       ├── metadata.duckdb        # Reference data (maps, medals…)
│       └── shared_matches.duckdb  # Shared match data (centralised)
├── .env.local                     # Azure tokens (created by the wizard)
├── db_profiles.json               # Player profiles (created by the wizard)
└── ...
