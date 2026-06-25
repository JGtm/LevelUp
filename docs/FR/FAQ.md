# FAQ — Questions fréquentes

Version anglaise : [../FAQ.md](../FAQ.md)

> LevelUp est un dashboard de stats Halo. Stack : un backend Go (`apps/go-api`) et un frontend React/Vite (`apps/web`). Il n'y a plus de Python. Pour l'installation complète, voir [INSTALL.md](INSTALL.md) ; pour la synchronisation, voir [SYNC_GUIDE.md](SYNC_GUIDE.md).

## Installation & prérequis

### Que dois-je installer ?

- **Go 1.26+** (le driver DuckDB exige CGO — une toolchain C comme MinGW sous Windows est nécessaire pour la suite de tests complète)
- **Node.js + npm** (frontend)
- **GNU Make** et **Git**
- **Air** pour le hot reload Go : `go install github.com/air-verse/air@latest`

Aucun Python n'est requis. Étapes complètes : [INSTALL.md](INSTALL.md).

### Comment lancer l'application ?

Depuis la racine du dépôt :

```bash
make dev
```

Cela démarre l'API Go sur le port 8000 et le frontend Vite sur http://localhost:5173. Au premier lancement, l'assistant de configuration dans le navigateur guide la saisie du gamertag + connexion Xbox. Pour tout arrêter : `make stop` (ou `Ctrl+C` dans le terminal de `make dev`).

### Le frontend ne démarre pas (« Module not found »)

Installez les dépendances frontend :

```bash
cd apps/web && npm install
```

## Configuration & tokens

### Mon token Xbox a expiré

Ne **re-capturez pas** les tokens pour corriger un 401 transitoire — une sync verte signifie que les tokens sont bons, et le pool d'auth les rafraîchit automatiquement.

Si le refresh token est réellement expiré, reconnectez-vous dans l'app : **Paramètres → Connexion Xbox → Reconnecter** (relance le Device Code Flow). Pour les joueurs headless :

```bash
go run ./apps/go-api/cmd/token-capture/ <Gamertag>
```

Les tokens sont la source unique dans `data/auth/watcher_tokens/{xuid}.json` (voir [ADR 0023](../adr/0023-auth-tokens-single-source.md)). Le joueur doit d'abord être déclaré dans `db_profiles.json` (avec son `xuid`).

### Comment ajouter un nouveau joueur ?

1. Déclarez le joueur dans `db_profiles.json` (avec son `xuid`).
2. Onboardez via l'assistant in-app (SSO Xbox) ou, en headless, `go run ./apps/go-api/cmd/token-capture/ <Gamertag>`.
3. La synchronisation se fait automatiquement dès que le token est dans le store (voir ci-dessous).

## Synchronisation

### Dois-je lancer une commande de sync ?

Non. La sync tourne **à l'intérieur du serveur Go** : un watcher de présence déclenche une sync delta quand un joueur termine un match, et un scheduler auto-sync synchronise périodiquement chaque joueur. L'usage quotidien ne demande aucune action manuelle. Détails : [SYNC_GUIDE.md](SYNC_GUIDE.md).

Des commandes CLI manuelles existent pour le bootstrap et le rattrapage : `levelup sync-delta` / `sync-full` / `backfill` (depuis `apps/go-api/cmd/levelup`). Ne lancez pas d'outil CLI sur une DB partagée pendant que le serveur tient le handle DuckDB — arrêtez le serveur d'abord.

### Quelle différence entre sync delta et full ?

- **Delta** — ne récupère que les matchs plus récents que le dernier watermark. Rapide ; c'est le défaut du watcher et du scheduler.
- **Full** — parcourt les N derniers matchs de l'API et insère ceux qui manquent (rattrapage). À utiliser après une longue interruption, un import ou un problème de watermark.

## Données

### Où sont stockées mes données ?

Sous `data/`, dans une arborescence agnostique au titre `data/titles/{slug}/` (slug par défaut `halo_infinite`) :

```
data/
├── auth/watcher_tokens/{xuid}.json              # store de tokens OAuth/MSAL (ADR 0023)
└── titles/halo_infinite/
    ├── warehouse/
    │   ├── metadata.duckdb                       # référentiels
    │   ├── shared_matches_v2.duckdb              # données de matchs partagées
    │   └── shared_pve.duckdb                     # stats Firefight
    └── players/{gamertag}/stats.duckdb           # enrichissements par joueur
```

Schéma complet et justification : [ARCHITECTURE_V6.md](ARCHITECTURE_V6.md).

## Développement

### Comment lancer les tests ?

```bash
# Sous-ensemble rapide, sans DuckDB (sans CGO)
cd apps/go-api
CGO_ENABLED=0 go test ./internal/domain/... ./internal/analysis/... ./contracttest/... -count=1

# Suite complète avec DuckDB (requiert CGO)
CGO_ENABLED=1 LEVELUP_DEMO_MODE=true go test ./... -timeout 5m -count=1

# Frontend
cd apps/web && npm run typecheck && npm test
```

Le raccourci `make go-api-test` lance le sous-ensemble rapide. Matrice complète (MinGW Windows, ratchet de couverture) : [testing.md](../testing.md).

### Comment reporter un bug ?

Ouvrez une issue GitHub avec votre OS, le message d'erreur complet et les étapes pour reproduire. Les logs sont écrits par catégorie sous `logs/*.log`.

## Divers

### LevelUp collecte-t-il des données ?

Non. Toutes les données restent sur votre machine ; aucune télémétrie n'est envoyée.

### Le projet est-il affilié à 343 Industries ou Microsoft ?

Non. LevelUp est un projet communautaire non officiel.
