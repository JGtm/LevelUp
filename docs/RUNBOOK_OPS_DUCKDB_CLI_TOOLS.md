# Runbook ops — CLI tools DuckDB et serveur LevelUp

**Date** : 2026-05-22 (Phase 5 plan stabilisation)
**Audience** : opérateurs/devs qui lancent les CLI `cmd/*` pendant que le serveur tourne
**Référence** : [.ai/archive/stabilisation-2026-05-22/HANDOFF_db_open_audit_2026-05-20.md](../.ai/archive/stabilisation-2026-05-22/HANDOFF_db_open_audit_2026-05-20.md) §3

## Le problème

DuckDB ne supporte pas le partage de file-lock OS entre processus distincts. Si un CLI tool ouvre `metadata.duckdb` en RW pendant que le serveur tient son handle, l'un des deux échouera avec :

```
IO Error: Cannot open file "metadata.duckdb":
Le processus ne peut pas accéder au fichier car ce fichier est utilisé par un autre processus.
```

Ce **n'est pas un bug applicatif** — c'est une contrainte du noyau Windows + de DuckDB en single-instance-per-file. Aucun lock applicatif Go ne peut résoudre ce conflit cross-process.

## Les outils concernés

Les CLI suivants modifient des DBs **partagées** (metadata.duckdb, shared_matches_v2.duckdb, shared_social.duckdb, ou xbox_aliases.duckdb global). Ne pas les lancer si le serveur (`apps/go-api/server.exe` ou `air`) tourne :

### Metadata
- `cmd/refresh-metadata` — recharge le catalogue Halo
- `cmd/seed-weapon-labels` — labels weapons
- `cmd/seed-rank-translations` — labels rangs
- `cmd/seed-assists-model` — coefficients OLS
- `cmd/migrate-static-maps` — UUIDs map_id
- `cmd/refresh-career-ranks` — career_ranks
- `cmd/populate-career-rank-images` — assets images
- `cmd/populate-playlists-catalog` — playlists_catalog
- `levelup populate-assets` — assets génériques (subcommand of the `levelup` CLI since 2026-07-13, shipped in the prod image ; ex-`cmd/populate-assets`)

### Shared matches v2 + xuid_aliases global
- `cmd/migrate-xuid-aliases-global` — globalise xuid_aliases
- `cmd/migrate-to-shared-social` — déplace tables vers shared_social

### Backfills (toutes DBs)
- `cmd/backfill_all` / `cmd/levelup backfill --*` — recalcule LUSR/perf/citations
- `cmd/levelup backfill --csr` / `--shared-csr` — refetche CSR API
- `cmd/levelup sync-achievements` — sync Xbox achievements
- `cmd/apply_shared_migrations` (Phase 1 plan stabilisation) — applique migrations TargetShared manuellement

### Diagnostics (read-only, OK pendant serveur)
- `cmd/diag_lusr_player` — ouvert en RO, safe
- `cmd/diag_player_schemas` — ouvert en RO, safe
- `cmd/diag_bitmask_coverage` — RO
- `cmd/diag_db_health` — RO

## Procédure recommandée

### Pour les CLI modifiants (RW)

```bash
# 1. Arrêter le serveur proprement (Ctrl+C ou air kill)
#    → laisser ~5 secondes pour libération des handles Windows post-shutdown
#    → vérifier dans logs/duckdb.log qu'il n'y a pas de "shutdown_db_leak"

# 2. Lancer le CLI
cd apps/go-api
go run ./cmd/refresh-metadata
# ou
./apps/go-api/levelup-cli.exe backfill --all --lusr --force

# 3. Redémarrer le serveur
air  # ou ./apps/go-api/server.exe
```

### Pour les CLI read-only (diagnostics)

OK de les lancer pendant que le serveur tourne :

```bash
# Le serveur tourne — lance le diag dans une autre fenêtre
./apps/go-api/diag_lusr_player.exe
./apps/go-api/diag_player_schemas.exe
curl http://127.0.0.1:8000/api/v1/_diag/progression/JGtm
curl http://127.0.0.1:8000/api/v1/_diag/csr-coverage/JGtm
```

## Détection automatique d'oubli

Le serveur logge au boot un retry sur `metadata.duckdb` avec un backoff (12×500ms = 6s). Si un CLI tient le verrou plus longtemps, le serveur logge `slog.Error("ouverture metadata échouée", ...)` après 12 tentatives et fait `os.Exit(1)`.

Si tu vois cette erreur au boot serveur :

```bash
# Vérifier qu'aucun autre process Go tient un handle DuckDB
tasklist | grep -i -E "levelup|server\.exe|go\.exe"

# Si présent, le kill (proprement si possible — Ctrl+C, sinon taskkill)
taskkill /PID <pid>
```

## Pourquoi pas un check `lsof` automatique ?

Idée : faire `lsof | grep .duckdb` dans chaque CLI tool avant ouverture, avec exit gracieux si lock détecté.

**Pas implémenté** car :
1. `lsof` n'existe pas en natif sur Windows (sysinternals `handle.exe` est l'équivalent mais nécessite admin)
2. Ajoute une dépendance fragile à un binaire externe
3. Le coût opérationnel actuel est faible (un retry au boot = ~1s)

Si l'incident devient fréquent, considérer wrapper les CLI Go avec une vérification manuelle de présence d'un PID du serveur (lecture de `data/server.pid` par exemple).

## Anti-pattern : ne pas faire

- ❌ Lancer `cmd/refresh-metadata` pendant que `air` recharge → race condition garantie
- ❌ Lancer 2 CLI Go modifiants en parallèle (l'un sur metadata, l'autre sur shared) → race
- ❌ Ouvrir `data/players/{slug}/stats.duckdb` en RW depuis un script externe pendant que l'auto-sync tourne pour ce joueur
- ❌ Garder un binaire `levelup-cli.exe` ouvert (mode interactif) pendant le serveur

## Historique d'incidents

| Date | Symptôme | Cause | Fix |
|------|----------|-------|-----|
| 2026-05-20 | metadata retry 6× au boot | Race CLI `populate-playlists-catalog` × `air` | Workflow doc + Phase 2 plan stabilisation (DumpCachedLeaks) |
| 2026-05-21 | `Chocoboflor/stats.duckdb.wal` orphelin | Crash sync engine pendant un write | Restart serveur en RW (replay WAL automatique) |
