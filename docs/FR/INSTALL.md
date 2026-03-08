# Guide d'Installation - LevelUp

> Guide complet pour installer et configurer LevelUp sur votre machine.

## Installation recommandée (Windows — grand public)

LevelUp fournit un lanceur tout-en-un qui automatise l'intégralité de l'installation.
**Vous n'avez pas besoin de savoir ce qu'est Python.**

### Étape 1 — Télécharger LevelUp

Rendez-vous sur la page GitHub du projet → bouton vert **Code** → **Download ZIP**.
Extrayez le dossier où vous voulez (ex. Bureau ou `C:\LevelUp\`).

> Vous pouvez aussi cloner avec Git si vous savez l'utiliser :
> ```bash
> git clone https://github.com/JGtm/LevelUp_with_SPNKr.git
> ```

### Étape 2 — Double-cliquer sur `LevelUp.bat`

Le lanceur fait **tout automatiquement** :

1. Cherche Python sur votre PC
2. Si absent → le télécharge et l'installe via `winget` (Windows 10/11 — vous répondez `O`)
3. Crée un environnement isolé (`.venv`)
4. Installe toutes les dépendances
5. Lance le dashboard et ouvre votre navigateur sur `http://localhost:8501`

> **Au premier lancement** : 2–5 minutes (téléchargements). Les suivants : quelques secondes.

### Étape 3 — Setup Wizard dans le navigateur

Au premier lancement, LevelUp détecte qu'il n'est pas configuré et affiche un **wizard guidé**.
Choisissez votre parcours :

#### 🎮 Xbox Express (recommandé — 2 étapes)

Le parcours le plus simple. Seule contrainte inévitable : créer une application Azure gratuite
(Microsoft l'exige pour l'accès à l'API Halo Infinite officielle).

**Étape 1 du wizard — Créer une application Azure (gratuit, aucun frais)**

> Azure est le service cloud Microsoft qui gère l'authentification Xbox.
> LevelUp a besoin d'une « clé d'accès » propre à chaque utilisateur.
> L'inscription est gratuite ; LevelUp n'utilise aucun service payant Azure.

1. Allez sur [portal.azure.com](https://portal.azure.com) — connectez-vous avec votre compte Microsoft/Xbox
2. Cherchez **Microsoft Entra ID** → **App registrations** → **New registration**
3. Remplissez :
   - Nom : `LevelUp Halo`
   - Type de compte : *Personal Microsoft accounts only*
   - Redirect URI → **Web** → `http://localhost:8501`
4. Cliquez **Register**
5. Sur la page **Overview** : copiez l'**Application (client) ID** (format `xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx`)
6. Allez dans **Certificates & secrets** → **New client secret** → donnez un nom → **Add**
   → copiez immédiatement la colonne **Value** (elle disparaît si vous naviguez ailleurs)
7. Allez dans **API permissions** → **Add a permission** → **Microsoft Graph** :
   ajoutez `offline_access` et `User.Read`

Collez le Client ID et la Value dans le wizard → LevelUp sauvegarde tout automatiquement.

**Étape 2 du wizard — Connexion Xbox en 1 clic**

Cliquez sur **"Se connecter avec Xbox"** → une fenêtre Microsoft s'ouvre → connectez-vous
avec votre compte Xbox → LevelUp récupère automatiquement votre gamertag et XUID,
crée votre profil et stocke le token OAuth dans votre base de données.

#### ☁️ Azure manuel (avancé — 3 étapes)

Même configuration Azure qu'au-dessus, mais le refresh token est obtenu manuellement
(à utiliser si le flux Xbox automatique pose problème, ex. reverse proxy) :

```bash
python scripts/spnkr_get_refresh_token.py
```

Ce script ouvre un navigateur, vous authentifie et affiche le token à copier dans `.env.local`.

### Étape 4 — Smoke test (vérification automatique sur 20 matchs)

Après la connexion Xbox, le wizard lance automatiquement un **smoke test en 3 phases** :

| Phase | Ce qui se passe |
|-------|----------------|
| 📡 Phase 1 — Sync | Synchronisation de 20 matchs depuis l'API Halo |
| ⚙️ Phase 2 — Enrichissement | Calcul des scores, sessions, citations, LUSR/CSR, paires killer/victim |
| 🔍 Phase 3 — Vérification | Contrôle d'intégrité de toutes les tables (voir ci-dessous) |

**Tables vérifiées (toutes obligatoires) :**

| Table | Base | Ce qui est validé |
|-------|------|-------------------|
| `match_registry` | shared | count > 0 |
| `match_participants` | shared | count > 0 + kills/deaths non NULL |
| `medals_earned` | shared | count > 0 |
| `killer_victim_pairs` | shared | count > 0 |
| `xuid_aliases` | shared | count > 0 |
| `player_match_enrichment` | player | count > 0 + session_id non NULL |
| `performance_score` | shared (via match_participants) | score calculé > 0 |
| `match_citations` | player | count > 0 |
| `match_skill_rank` (LUSR/CSR) | player | count > 0 + LUSR/CSR présents |
| `sessions` | player | count > 0 |
| `highlight_events` | shared | count > 0 (clips filmés) |
| `sync_meta` | player | count > 0 |
| Cohérence shared↔player | croisé | counts cohérents |

Si un check échoue, le test propose de **relancer**. Si tout est vert, deux choix s'offrent :

- **⚙️ Sync complète** → navigue vers la page Paramètres pour récupérer tout votre historique (recommandé)
- **📊 Dashboard (20 matchs)** → accède directement au dashboard avec les matchs déjà synchronisés

---

## Installation pour développeurs

### Prérequis
- Python 3.10+ (recommandé : 3.12)
- Git

```bash
git clone https://github.com/JGtm/LevelUp_with_SPNKr.git
cd LevelUp_with_SPNKr

# Créer l'environnement virtuel
python -m venv .venv

# Activer (Windows PowerShell)
.venv\Scripts\Activate.ps1
# Activer (Linux/macOS)
source .venv/bin/activate

# Installation complète (avec outils de dev)
pip install -e ".[dev,spnkr]"
```

### Vérification de l'environnement

```bash
# Healthcheck complet
python scripts/check_env.py

# Ou via le lanceur
python launcher.py doctor
```

### Tests

```bash
# Suite complète
python -m pytest

# Hors intégration (plus rapide)
python -m pytest --ignore=tests/integration

# Un fichier spécifique
python -m pytest tests/test_duckdb_repository.py -v
```

### Mise à jour

```bash
git pull origin main
python launcher.py setup --update
```

Voir [CONFIGURATION.md](CONFIGURATION.md) pour la configuration des tokens Azure.

### 3. Ajouter un joueur via CLI (si le wizard n'est pas utilisé)

```bash
python scripts/sync.py --add-player MonGamertag
```

Cette commande crée automatiquement l'entrée dans `db_profiles.json` et le dossier `data/players/MonGamertag/`.

### 4. Premier Lancement

```bash
python launcher.py run
```

---

## Installation Docker

### Prérequis
- Docker Desktop installé
- Docker Compose v2 disponible (`docker compose version`)
- Fichier `db_profiles.json` à la racine du projet (créé automatiquement si absent)

### Prérequis : fichiers de configuration

Avant le premier `docker compose up`, assurez-vous que ces fichiers existent. Sinon, créez-les :

```bash
# Si db_profiles.json n'existe pas encore
echo '{"profiles": {}}' > db_profiles.json

# Si app_settings.json n'existe pas encore
echo '{}' > app_settings.json
```

> **Pourquoi ?** Docker bind-mount crée un *dossier* (pas un fichier) si la source n'existe pas, ce qui crasherait l'app.

### Lancer avec Docker Compose

```bash
# Construire et démarrer
docker compose up --build

# En arrière-plan
docker compose up -d

# Voir les logs
docker compose logs -f

# Arrêter
docker compose down
```

### Architecture de l'image

L'image Docker :
- Installe les dépendances via `pip install -e ".[spnkr]"` (pyproject.toml), incluant SPNKr + aiohttp pour la synchronisation API
- Embarque les données de référence minimales (traductions playlists, wiki commendations)
- Tourne en utilisateur non-root (`appuser`, UID 10001)
- Expose le healthcheck Streamlit sur `/_stcore/health`

### Configuration Docker

`docker-compose.yml` monte les volumes suivants :

| Volume hôte | Chemin conteneur | Description |
|-------------|-----------------|-------------|
| `./data` | `/app/data` | Données DuckDB v5 (lecture/écriture) |
| `./db_profiles.json` | `/app/db_profiles.json` | Profils joueurs |
| `./app_settings.json` | `/app/app_settings.json` | Paramètres applicatifs |

### Variables d'Environnement Docker

| Variable | Défaut | Description |
|----------|--------|-------------|
| `LEVELUP_ROOT` | `/app` | Racine du projet (détection pyproject.toml) |
| `LEVELUP_DATA` | `%APPDATA%/LevelUp` ou `./data` | Répertoire des données |
| `LEVELUP_DEFAULT_GAMERTAG` | *(vide)* | Gamertag par défaut pour mode headless |

---

## Dépannage

### Diagnostic complet

```bash
python launcher.py doctor
```

### Erreur "Module not found"

```bash
python launcher.py setup
```

### Erreur DuckDB (version incorrecte)

```bash
python -c "import duckdb; print(duckdb.__version__)"
# Doit être >= 1.4.0
python launcher.py setup --update
```

### Problème de token OAuth expiré

Aller dans l'app → **Paramètres** → **Connexion Xbox** → **Reconnecter**.
Le token est stocké dans `data/players/<gamertag>/stats.duckdb` (table `sync_meta`).

### Permission Denied (Windows / PowerShell)

```powershell
# Autoriser les scripts PowerShell (une seule fois)
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
```

---

## Structure des Dossiers Après Installation

```
LevelUp/
├── .venv/                         # Environnement virtuel Python
├── data/
│   ├── players/
│   │   └── MonGamertag/
│   │       └── stats.duckdb       # Enrichissements joueur
│   └── warehouse/
│       ├── metadata.duckdb        # Référentiels (maps, médailles…)
│       └── shared_matches.duckdb  # Matchs partagés (centralisé)
├── .env.local                     # Tokens Azure (créé par le wizard)
├── db_profiles.json               # Profils joueurs (créé par le wizard)
└── ...
```

---

## Prochaines Étapes

1. [Configuration Azure détaillée](CONFIGURATION.md)
2. [Synchroniser vos matchs](SYNC_GUIDE.md)
3. [Explorer le dashboard](../README.md#utilisation)
