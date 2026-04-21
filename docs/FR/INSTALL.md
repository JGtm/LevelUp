# Guide d'Installation - LevelUp

> Guide complet pour installer et configurer LevelUp sur votre machine.

## Installation locale recommandée (Windows)

Ce dépôt Go n'embarque plus de lanceurs `LevelUp.bat` ou `LevelUp.sh`.
Le point d'entrée standard est désormais `make dev`.

### Étape 1 — Télécharger LevelUp

Rendez-vous sur la page GitHub du projet → bouton vert **Code** → **Download ZIP**.
Extrayez le dossier où vous voulez (ex. Bureau ou `C:\LevelUp\`).

> Vous pouvez aussi cloner avec Git si vous savez l'utiliser :
> ```bash
> git clone https://github.com/JGtm/LevelUp_with_SPNKr.git
> ```

### Étape 2 — Installer l'outillage local

1. Installez Go 1.26+ et Node.js sur votre machine
2. Installez Air pour le hot reload Go :
   ```bash
   go install github.com/air-verse/air@latest
   ```
3. Installez les dépendances frontend :
   ```bash
   cd apps/web && npm install && cd ../..
   ```

### Étape 3 — Démarrer l'application

```bash
make dev
```

Cela démarre l'API Go sur le port 8000 et le frontend Vite sur http://localhost:5173.

### Étape 4 — Setup Wizard dans le navigateur

Au premier lancement, LevelUp détecte qu'il n'est pas configuré et affiche un **wizard guidé**.
Choisissez votre parcours :

#### 🎮 Xbox Express (recommandé — 2 étapes)

Le parcours le plus simple.

**Étape 1 du wizard — Application Azure (automatique ou manuelle)**

> Azure est le service cloud Microsoft qui gère l'authentification Xbox.
> LevelUp a besoin d'une « clé d'accès » propre à chaque utilisateur.
> L'inscription est gratuite ; LevelUp n'utilise aucun service payant Azure.

**Option A — Automatique (recommandée) : avec Azure CLI**

Si [Azure CLI](https://aka.ms/installazurecli) est installé sur votre machine, LevelUp
crée l'application automatiquement **sans visiter le portail Azure** pendant le wizard.

**Option B — Manuelle : sans Azure CLI**

Si Azure CLI n'est pas installé, LevelUp ouvre portal.azure.com et vous demande de saisir
uniquement le **Client ID** (pas de client secret requis) :

1. Allez sur [portal.azure.com](https://portal.azure.com) — connectez-vous avec votre compte Microsoft/Xbox
2. Cherchez **Microsoft Entra ID** → **App registrations** → **New registration**
3. Remplissez :
   - Nom : `LevelUp Halo`
   - Type de compte : *Personal Microsoft accounts only*
   - Laisser Redirect URI vide → **Register**
4. Sur la page **Overview** : copiez l'**Application (client) ID** (format `xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx`)
5. Dans **Authentication** → **Advanced settings** → mettez **Allow public client flows** à **Yes** → **Save**

Collez uniquement le Client ID dans le wizard (plus de secret requis).

**Étape 2 du wizard — Connexion Xbox en 1 clic**

Cliquez sur **"Se connecter avec Xbox"** → une fenêtre Microsoft s'ouvre → connectez-vous
avec votre compte Xbox → LevelUp récupère automatiquement votre gamertag et XUID,
crée votre profil et stocke le token OAuth dans votre base de données.

#### ☁️ Azure manuel (avancé — 3 étapes)

Même configuration Azure qu'au-dessus, mais le refresh token est obtenu manuellement
(à utiliser si le flux Xbox automatique pose problème, ex. reverse proxy) :

```bash
python scripts/spnkr_get_refresh_token.py --device-code
```

Ce script affiche un code court à entrer sur https://microsoft.com/devicelogin et sauvegarde
le token automatiquement dans `.env.local`.

### Étape 5 — Smoke test (vérification automatique sur 20 matchs)

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
- Go 1.26+
- Node.js + npm
- GNU Make
- Git

```bash
git clone https://github.com/JGtm/LevelUp_with_SPNKr.git
cd LevelUp_with_SPNKr
cd apps/web && npm install && cd ../..
go install github.com/air-verse/air@latest
make dev
```

### Vérification de l'environnement

```bash
curl http://127.0.0.1:8000/health
make go-api-test
cd apps/web && npm run typecheck
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
cd apps/web && npm install && cd ../..
go install github.com/air-verse/air@latest
```

Voir [CONFIGURATION.md](CONFIGURATION.md) pour la configuration des tokens Azure.

### 3. Ajouter un joueur via CLI (si le wizard n'est pas utilisé)

```bash
python scripts/sync.py --add-player MonGamertag
```

Cette commande crée automatiquement l'entrée dans `db_profiles.json` et le dossier `data/players/MonGamertag/`.

### 4. Premier Lancement

```bash
make dev
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
- Expose le healthcheck FastAPI sur `/api/v1/health`

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
curl http://127.0.0.1:8000/health
make go-api-test
cd apps/web && npm run typecheck
```

### Erreur "Module not found"

```bash
cd apps/web && npm install
```

### Erreur DuckDB (version incorrecte)

```bash
cd apps/go-api && go test ./... -count=1
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
