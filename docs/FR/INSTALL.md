# Guide d'installation — LevelUp

Version anglaise : [../INSTALL.md](../INSTALL.md)

> Guide complet pour installer et configurer LevelUp sur votre machine.

---

## Windows — Installation locale recommandée

Ce dépôt de migration Go n'embarque plus de lanceurs en un clic.
Le point d'entrée standard est désormais `make dev`.

### Étape 1 — Télécharger LevelUp

Rendez-vous sur la page GitHub du projet → bouton vert **Code** → **Download ZIP**.
Extrayez le dossier où vous voulez (ex. Bureau ou `C:\LevelUp\`).

> Si vous connaissez Git, vous pouvez aussi cloner :
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

Au premier lancement, LevelUp détecte qu'il n'est pas encore configuré et affiche un **wizard guidé**.
Choisissez votre parcours :

#### Xbox Express (recommandé — 2 étapes)

**v6 — Aucune configuration Azure requise.** LevelUp embarque son propre client ID.

**Étape 1 — Saisir votre gamertag**

Tapez votre gamertag Xbox dans le wizard. LevelUp crée automatiquement votre profil local.

**Étape 2 — S'authentifier via Device Code**

Le wizard affiche un code court et l'URL `https://xbox.com/activate`.

1. Ouvrez [https://xbox.com/activate](https://xbox.com/activate) dans votre navigateur
2. Entrez le code affiché dans le wizard
3. Connectez-vous avec votre compte Microsoft/Xbox

C'est tout — LevelUp récupère votre XUID, complète votre profil et persiste le refresh token
OAuth dans le store de tokens unique (`data/auth/watcher_tokens/{xuid}.json`,
voir [ADR 0023](../adr/0023-auth-tokens-single-source.md)), puis lance le smoke test.

#### Onboarding avancé (headless / CLI)

Utilisez ce parcours quand le wizard interactif n'est pas accessible (serveur, headless, reverse proxy).
Le joueur doit déjà être déclaré dans `db_profiles.json` avec son `xuid` avant de lancer ces
commandes (le store de tokens est adressé par xuid). Les tokens vont directement dans le store unique — il n'y a
**aucune manipulation de `.env.local`** (voir [ADR 0023](../adr/0023-auth-tokens-single-source.md)).

```bash
# Device Code Flow dans le navigateur, refresh token écrit dans le store
cd apps/go-api && go run ./cmd/token-capture/ <Gamertag>

# Ou importer un refresh token obtenu ailleurs (lu depuis stdin)
cd apps/go-api && go run ./cmd/token-import/ <Gamertag>
```

Après capture/import, redémarrez le serveur : le Pool d'auth trouve le token dans le store et
fonctionne immédiatement.

#### Note pour les forks / développeurs

Le client ID embarqué est lié à l'Azure App Registration de ce projet.
Si vous forkez LevelUp, créez votre propre Azure App Registration (gratuite) et définissez :

```env
# .env.local
LEVELUP_OAUTH_CLIENT_ID=your_own_client_id
```

Voir [CONFIGURATION.md](CONFIGURATION.md) pour le déroulé complet de l'inscription Azure.
Cette variable d'environnement prend le pas sur l'ID embarqué. Notez que `.env.local` ne sert qu'à la config
(client ID) : ce n'est **pas** un store de credentials — les refresh tokens vivent dans le store de tokens
(voir [ADR 0023](../adr/0023-auth-tokens-single-source.md)).

#### Fournisseur d'authentification — SISU (défaut) vs MSAL

Le Device Code Flow d'onboarding utilise par défaut le fournisseur **SISU** : le flux
device-code natif Xbox, qui ne requiert **aucune app Azure** — c'est pourquoi « Xbox Express »
ci-dessus ne demande aucune configuration Azure. Un repli config-only (`app_settings.json` :
`"auth_provider": "msal"`) bascule sur le fournisseur MSAL (client ID Azure embarqué/le vôtre)
si l'endpoint natif Xbox venait à casser. Aucun bouton d'UI — laissez vide (`sisu`) sauf
besoin explicite de MSAL. Voir [CONFIGURATION.md](CONFIGURATION.md) pour le détail.

### Étape 5 — Smoke test (vérification automatique sur 20 matchs)

Après la connexion Xbox, le wizard lance automatiquement un **smoke test en 3 phases** :

| Phase | Ce qui se passe |
|-------|-----------------|
| Phase 1 — Sync | Synchronisation de 20 matchs depuis l'API Halo |
| Phase 2 — Enrichissement | Calcul des scores, sessions, citations, LUSR/CSR, paires killer/victim |
| Phase 3 — Vérification | Contrôle d'intégrité complet de toutes les tables (voir ci-dessous) |

**Tables vérifiées (toutes obligatoires) :**

| Table | Base | Ce qui est validé |
|-------|------|-------------------|
| `match_registry` | shared | count > 0 |
| `match_participants` | shared | count > 0 + kills/deaths non NULL |
| `medals_earned` | shared | count > 0 |
| `killer_victim_pairs` | shared | count > 0 |
| `highlight_events` | shared | count > 0 (clips filmés) |
| `xuid_aliases` | shared | count > 0 |
| `player_match_enrichment` | player | count > 0 + session_id non NULL |
| `performance_score` | shared (via match_participants) | score calculé > 0 |
| `match_citations` | player | count > 0 |
| `match_skill_rank` (LUSR/CSR) | player | count > 0 + LUSR/CSR présents |
| `sessions` | player | count > 0 |
| `sync_meta` | player | count > 0 |
| Cohérence shared↔player | croisé | counts cohérents |

Si un check échoue, le test propose de **relancer**. Quand tout est vert, deux choix :

- **Sync complète** → navigue vers la page Paramètres pour récupérer tout votre historique (recommandé)
- **Dashboard (20 matchs)** → accède directement au dashboard avec les matchs déjà synchronisés

---

## macOS / Linux

Le workflow local est identique à celui de Windows :

1. Installez Go 1.26+, Node.js + npm, et GNU Make
2. Installez Air :
   ```bash
   go install github.com/air-verse/air@latest
   ```
3. Installez les dépendances frontend :
   ```bash
   cd apps/web && npm install && cd ../..
   ```
4. Démarrez la stack :
   ```bash
   make dev
   ```

Ouvrez ensuite http://localhost:5173 et complétez le wizard dans l'application.

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

Le driver DuckDB requiert CGO. Voir [testing.md](../testing.md) pour la matrice complète
(chemin rapide CGO=0, ratchet de couverture, Windows MinGW).

```bash
# Go — suite complète avec DuckDB (CGO)
cd apps/go-api
CGO_ENABLED=1 LEVELUP_DEMO_MODE=true go test ./... -timeout 5m -count=1

# Go — sous-ensemble rapide sans DuckDB
CGO_ENABLED=0 go test ./internal/domain/... ./internal/analysis/... ./contracttest/... -count=1

# Frontend
cd apps/web && npm run typecheck && npm test
```

### Mise à jour

```bash
git pull origin main
cd apps/web && npm install && cd ../..
go install github.com/air-verse/air@latest
```

Voir [CONFIGURATION.md](CONFIGURATION.md) pour la configuration des tokens Azure.

---

## Installation Docker

### Prérequis
- Docker Desktop installé
- Docker Compose v2 disponible (`docker compose version`)

### Prérequis : fichiers de configuration

Avant le premier `docker compose up`, assurez-vous que ces fichiers existent :

```bash
# Si db_profiles.json n'existe pas encore
echo '{"profiles": {}}' > db_profiles.json

# Si app_settings.json n'existe pas encore
echo '{}' > app_settings.json
```

> **Pourquoi ?** Docker bind-mount crée un *dossier* (pas un fichier) si la source n'existe pas,
> ce qui crasherait l'app.

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

### Volumes Docker

| Chemin hôte | Chemin conteneur | Description |
|-------------|------------------|-------------|
| `./data` | `/app/data` | Données DuckDB (lecture/écriture) |
| `./db_profiles.json` | `/app/db_profiles.json` | Profils joueurs |
| `./app_settings.json` | `/app/app_settings.json` | Paramètres applicatifs |

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

### Erreur de version DuckDB

```bash
cd apps/go-api && CGO_ENABLED=1 go test ./... -count=1
```

### Token OAuth expiré

Dans l'app → **Paramètres** → **Connexion Xbox** → **Reconnecter** (relance le flux Device Code
et rafraîchit le token dans le store). Pour les joueurs headless, relancez
`cd apps/go-api && go run ./cmd/token-capture/ <Gamertag>`. Le refresh token est persisté dans
`data/auth/watcher_tokens/{xuid}.json` (store de tokens unique, voir
[ADR 0023](../adr/0023-auth-tokens-single-source.md)).

### Permission Denied (Windows / PowerShell)

```powershell
# Autoriser les scripts PowerShell (une seule fois)
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
```

---

## Structure des dossiers après installation

```
LevelUp/
├── apps/
│   ├── go-api/                      # Backend Go (API + sync + CLI sous cmd/)
│   └── web/                         # Frontend React/Vite
├── data/
│   ├── auth/
│   │   └── watcher_tokens/
│   │       └── {xuid}.json          # Store de tokens OAuth/MSAL (ADR 0023)
│   ├── players/
│   │   └── MyGamertag/
│   │       └── stats.duckdb         # Enrichissements par joueur
│   └── warehouse/
│       ├── metadata.duckdb          # Référentiels (maps, médailles…)
│       └── shared_matches_v2.duckdb # Matchs partagés (centralisé)
├── db_profiles.json                 # Profils joueurs (créé par le wizard)
├── app_settings.json                # Paramètres applicatifs
└── .env.local                       # Config optionnelle (ex. LEVELUP_OAUTH_CLIENT_ID pour les forks)
```

---

## Prochaines étapes

1. [Configuration Azure détaillée](CONFIGURATION.md)
2. [Synchroniser vos matchs](SYNC_GUIDE.md)
3. [Explorer le dashboard](../../README.md)
