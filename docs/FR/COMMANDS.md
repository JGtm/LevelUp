# Commandes utiles — LevelUp

English version: [../COMMANDS.md](../COMMANDS.md)

> Aide-mémoire de la stack actuelle : backend Go (`apps/go-api`) + frontend React/Vite (`apps/web`).
> L'outillage d'exploitation est le CLI `levelup` (`apps/go-api/cmd/levelup`). Les cibles `make`
> sont dans le `Makefile` racine. L'accès DuckDB exige CGO (voir [Tests](#tests)).

---

## Lancement

```bash
make dev          # API Go (air, :8000) + frontend Vite (:5173) — Ctrl+C arrête tout
make go-api-dev   # API Go seule (hot-reload air)
make web          # Frontend seul (Vite, :5173)
make stop         # Arrête les serveurs dev (kill par port, API + 5173)
make restart      # stop + dev
```

Ouvrir http://localhost:5173 une fois `make dev` lancé.

---

## Build

```bash
make go-api-build   # CGO_ENABLED=1 go build -> apps/go-api/bin/server
make install-web    # npm install dans apps/web
make generate-types # Types TypeScript depuis apps/go-api/api/openapi.yaml
make check-types    # tsc -b (typecheck seul)
```

---

## CLI `levelup`

Compilé depuis `apps/go-api/cmd/levelup`. Lancer via `go run` (CGO requis) ou builder un binaire.
Utiliser `LEVELUP_REPO_ROOT` pour pointer le repo de données (auto-détecté si absent).

```bash
cd apps/go-api
CGO_ENABLED=1 go run ./cmd/levelup <commande> [options]
# Aide par commande :
CGO_ENABLED=1 go run ./cmd/levelup <commande> --help
```

### Synchronisation (API Halo)

```bash
# Sync delta — nouveaux matchs uniquement
go run ./cmd/levelup sync-delta --gamertag MonGamertag
go run ./cmd/levelup sync-delta --all --max-matches 25
# options : --match-type all|matchmaking|custom|local  --rps N  --token-pool-size N

# Sync complète — parcourt les N derniers matchs API, insère les manquants (comble les trous)
go run ./cmd/levelup sync-full --gamertag MonGamertag --max-matches 500

# Backfill des achievements Xbox (admin one-shot)
go run ./cmd/levelup sync-achievements --all [--dry-run]
```

### Backfill (local Go ; CSR/weapons nécessitent des tokens Halo)

```bash
go run ./cmd/levelup backfill --gamertag X --citations        [--force]
go run ./cmd/levelup backfill --all          --lusr           [--force]
go run ./cmd/levelup backfill --gamertag X --perf             [--force]
go run ./cmd/levelup backfill --gamertag X --engagement-scores
go run ./cmd/levelup backfill --gamertag X --csr             [--force]   # tokens Halo
go run ./cmd/levelup backfill --all          --shared-csr     [--dry-run] # tokens Halo
go run ./cmd/levelup backfill --all          --weapons        [--force]   # film CDN
go run ./cmd/levelup backfill --gamertag X --citations-recompute-all
```

### Backup / restore

```bash
go run ./cmd/levelup backup  --gamertag X [--output-dir D] [--compression-level 9]
go run ./cmd/levelup restore --gamertag X --backup-dir D [--replace] [--dry-run] [--tables T1,T2]
go run ./cmd/levelup restore-csr --gamertag X --backup PATH [--dry-run] [--mode preserve|overwrite]
```

### Référentiels / seed / migration

```bash
go run ./cmd/levelup seed career-ranks | citation-mappings | medals | rank-translations
go run ./cmd/levelup seed-demo            # génère les données démo anonymisées (data/demo/)
go run ./cmd/levelup migrate              # migre les données vers le namespace multi-titres
go run ./cmd/levelup add-title --name "Halo MCC" [--slug s] [--capabilities matchmaking,media] [--xbox-id X] [--steam-id S]
```

### Médias

```bash
go run ./cmd/levelup index-media --gamertag X [--force-rescan] [--buffer-min N]
```

### Diagnostic & ops

```bash
go run ./cmd/levelup healthcheck [--verbose]
go run ./cmd/levelup diagnose --db PATH [--verbose]
go run ./cmd/levelup check-env
go run ./cmd/levelup gate-check [--gamertag X] [--json]
go run ./cmd/levelup compare-db --go-db PATH --python-db PATH [--json]
```

### Prestige — analyseur de tuning de la grammaire coach

Analyseur en LECTURE SEULE (jamais d'ouverture RW). Produit des **recommandations**
d'ajustement de la grammaire de synthèse du coach
(`config/coach_advisor/synthesis_grammar.toml`) à partir de la télémétrie Prestige
(taux de complétion par métrique de grammaire). L'application reste **manuelle** : un
humain lit le rapport et édite le TOML — aucune PR automatique, aucun override runtime.

```bash
# Tous les joueurs d'un titre (défaut halo_infinite), rapport texte :
go run ./cmd/prestige-tuning-analyze
# Un seul joueur, sortie JSON :
go run ./cmd/prestige-tuning-analyze --player JGtm --format json
# Seuils personnalisés (règle : complétion < min-completion sur >= min-sample défis coach acceptés) :
go run ./cmd/prestige-tuning-analyze --min-completion 0.30 --min-sample 50 --source coach
# flags : --format text|json  --player SLUG|GAMERTAG  --title SLUG
#         --min-completion 0..1  --min-sample N  --source coach|user|pilot_mode  --grammar PATH
```

Sous `--min-sample` : « données insuffisantes » (aucune reco sur du bruit). Une métrique
de télémétrie absente de la grammaire est signalée comme orpheline (dérive de nommage /
défi legacy).

### Maintenance (serveur arrêté pour les rebuilds ART/alias)

```bash
go run ./cmd/levelup rebuild-pme-art --all | --gamertag X   # reconstruit l'index ART player_match_enrichment
go run ./cmd/levelup consolidate-aliases                    # merge xbox_aliases dans shared.xuid_aliases
go run ./cmd/levelup recompute-friends [--dry-run]          # recompute is_with_friends sur les player DBs
go run ./cmd/levelup replay-events --gamertag X             # re-parse les highlight events
go run ./cmd/levelup reset-bitmasks                         # reset des bits de backfill skill/participants/PVE
go run ./cmd/levelup engagement-coefs [--with-scores]      # recompute des coefficients d'engagement
```

### Notifications

```bash
go run ./cmd/levelup notify-version --version v1.2.3
go run ./cmd/levelup notify-sync --gamertag X --op sync_delta --duration 120s [--matches N]
```

Liste complète : `go run ./cmd/levelup help`.

---

## Tests

### Go (voir [../testing.md](../testing.md))

```bash
# Rapide, sans DuckDB (CGO off)
make go-api-test
# ou directement :
cd apps/go-api && CGO_ENABLED=0 go test ./internal/domain/... ./internal/analysis/... ./contracttest/... -count=1

# Suite complète avec DuckDB (CGO on — toolchain C / MinGW requis sur Windows)
cd apps/go-api && CGO_ENABLED=1 LEVELUP_DEMO_MODE=true go test ./... -timeout 5m -count=1

make go-api-coverage   # rapport de couverture
make go-api-lint       # go vet
```

### Frontend (`apps/web`)

```bash
make test-web        # vitest run
make test-e2e        # Playwright (nécessite `make dev` en cours)
make test-e2e-ui     # Playwright en mode UI
# ou via npm dans apps/web :
npm run test:run
npm run test:coverage
npm run lint
```

---

## Variables d'environnement

| Variable | Rôle |
|----------|------|
| `LEVELUP_REPO_ROOT` | Racine du repo de données (auto-détectée si absente) |
| `LEVELUP_API_PORT` | Port de l'API Go (défaut `8000`) |
| `LEVELUP_DEMO_MODE` | Mode démo (utilisé par les cibles de test) |
| `LEVELUP_NOTIFY_VERSIONS` | Mettre à `1` pour activer les notifs de version en prod |
| `DISCORD_WEBHOOK_URL` | Webhook Discord (prévaut sur `app_settings.json`) |
| `CGO_ENABLED` | Doit valoir `1` pour tout build/test touchant DuckDB |

---

## Chemins des données

```
data/
  warehouse/metadata.duckdb         # référentiels (maps, playlists, médailles)
  warehouse/shared_matches_v2.duckdb # matchs/médailles/events/aliases partagés
  warehouse/shared_pve.duckdb       # stats Firefight
  players/{gamertag}/stats.duckdb   # enrichissements par joueur
  players/{gamertag}/archive/       # archives Parquet
db_profiles.json                    # profils joueurs (multi-titres)
app_settings.json                   # paramètres app
.env.local                          # tokens Azure / secrets
```

Voir [ARCHITECTURE_V6.md](../ARCHITECTURE_V6.md) pour le modèle de données complet.
