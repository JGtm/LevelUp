# Guide Backup & Restore — LevelUp

Version anglaise : [../BACKUP_RESTORE.md](../BACKUP_RESTORE.md)

LevelUp stocke toutes ses données dans des fichiers DuckDB sous `data/titles/<slug>/`. Il existe **deux chemins de sauvegarde complémentaires** :

| Chemin | Ce qui est protégé | Format | Piloté par |
|--------|--------------------|--------|-----------|
| **Snapshots restic automatiques** (recommandé) | Toutes les DuckDB de tous les titres (warehouse + par joueur) | Parquet+Zstd en staging, puis un snapshot `restic` | scheduler `pkg/duckdbbackup` (intégré au serveur), `cmd/backup-once`, restauré via `cmd/restore` |
| **Export par joueur (manuel)** | Une seule `stats.duckdb` de joueur | Fichiers Parquet+Zstd sur disque | `levelup backup` / `levelup restore` |

Un one-shot dédié, `levelup restore-csr`, restaure les CSR historiques par-match depuis un backup DuckDB legacy.

---

## Arborescence des données (`data/titles/<slug>/`)

Tous les chemins sont résolus par le `PathResolver` de `internal/domain/title`. Pour le titre par défaut `halo_infinite` :

```
data/titles/halo_infinite/
├── warehouse/
│   ├── shared_matches_v2.duckdb   # matchs/participants/médailles partagés
│   ├── metadata.duckdb            # référentiels (rangs, citations, ...)
│   ├── shared_pve.duckdb          # stats Firefight
│   └── shared_social.duckdb       # likes/favoris/état social
├── players/
│   └── <Gamertag>/
│       ├── stats.duckdb           # DB d'enrichissement joueur
│       ├── archive/               # archives Parquet froides (`levelup archive`)
│       └── captures/              # médias capturés localement
└── backups/
    └── <Gamertag>/                # exports Parquet par joueur (manuels)
```

Le scheduler restic automatique découvre chaque titre sous `data/titles/` et protège chaque DB warehouse plus chaque `players/<Gamertag>/stats.duckdb`. Les clés de backup sont préfixées par le slug (`<slug>:shared_matches_v2`, `<slug>:metadata`, `<slug>:shared_pve`, `<slug>:shared_social`, `<slug>:player:<Gamertag>`).

---

## 1. Snapshots restic automatiques

### Fonctionnement

Le serveur exécute un scheduler de backup (`ops.NewLevelUpBackupScheduler`, câblé dans `cmd/server/main.go`). À chaque cycle :

1. Découverte de toutes les cibles DuckDB sous `data/titles/`.
2. Ignore les DB inchangées (manifest d'empreintes `.manifest.json` dans le dossier staging).
3. `PRAGMA integrity_check` sur les DB modifiées (une DB dégradée est tout de même sauvegardée, avec un warning).
4. Export des `BASE TABLE` de chaque DB modifiée en Parquet+Zstd vers une arborescence staging.
5. Création d'un unique snapshot `restic` du dossier staging, puis application de la politique de rétention (`restic forget --prune`).

Le premier cycle s'exécute immédiatement au démarrage, puis tous les `interval` (défaut `6h`). Si le binaire `restic` est absent du `PATH`, le scheduler logue un warning et se désactive.

### Configuration

Le comportement vient de `app_settings.json` ; les chemins machine viennent des variables d'environnement.

| Réglage (`app_settings.json`) | Défaut | Signification |
|-------------------------------|--------|---------------|
| `backup_enabled` | `false` | Active le scheduler périodique |
| `backup_interval` | `6h` | Durée Go entre deux cycles |
| `backup_keep_daily` | `7` | `restic forget --keep-daily` |
| `backup_keep_weekly` | `4` | `restic forget --keep-weekly` |
| `backup_keep_monthly` | `12` | `restic forget --keep-monthly` |

| Variable d'environnement | Défaut | Signification |
|--------------------------|--------|---------------|
| `RESTIC_REPOSITORY` | _(non défini)_ | Emplacement du dépôt restic (requis pour sauvegarder) |
| `RESTIC_PASSWORD` | _(non défini)_ | Mot de passe du dépôt |
| `RESTIC_PASSWORD_FILE` | _(non défini)_ | Fichier contenant le mot de passe |
| `RESTIC_BIN` | `restic` | Chemin du binaire `restic` |
| `LEVELUP_BACKUP_DIR` | `data/backups` | Répertoire de staging local |

Quand ni `RESTIC_PASSWORD` ni `RESTIC_PASSWORD_FILE` n'est défini, restic est invoqué avec `--insecure-no-password` (dépôt non chiffré — adapté à un usage local mono-utilisateur).

La rétention est une **enveloppe globale unique** sur tous les titres (un seul snapshot, une seule politique de rétention couvre les DB de chaque titre, par conception).

### Lancer un snapshot manuellement

```bash
cd apps/go-api
go run ./cmd/backup-once
```

Cela exécute exactement un cycle de façon synchrone et affiche l'id du snapshot, la durée et la liste des DB exportées. C'est un no-op (`Skipped`) si rien n'a changé depuis le dernier cycle.

### Lister et restaurer les snapshots

```bash
cd apps/go-api

# Lister les snapshots disponibles (id, date, hôte)
go run ./cmd/restore --list

# Restaurer le snapshot le plus récent dans data/restore/<YYYY-MM-DD>/
go run ./cmd/restore

# Restaurer le snapshot le plus récent d'un jour donné
go run ./cmd/restore --date 2026-05-25

# Restaurer un snapshot précis par id court
go run ./cmd/restore --snapshot 6ba84d2b

# Restaurer dans un répertoire personnalisé
go run ./cmd/restore --output /tmp/restore/

# Simuler sans rien écrire
go run ./cmd/restore --dry-run
```

Par défaut, `restore` écrit une arborescence miroir sous `data/restore/<date>/` (`{slug}/{db}.duckdb`, `{slug}/players/{gamertag}/stats.duckdb`) pour inspecter avant tout remplacement.

**Restauration en place sur la production** — écrase les fichiers DuckDB live :

```bash
go run ./cmd/restore --live          # demande confirmation
```

`--live` est incompatible avec `--output`. **Arrêter d'abord le serveur Go** — les DB ne doivent pas être ouvertes pendant leur remplacement.

---

## 2. Export par joueur (`levelup backup` / `restore`)

Exporte la `stats.duckdb` d'un seul joueur en fichiers Parquet+Zstd autonomes (portables, lisibles par tout outil DuckDB/Parquet). Indépendant de restic.

### Backup

```bash
cd apps/go-api

# Sortie par défaut : data/titles/<slug>/backups/<Gamertag>/
go run ./cmd/levelup backup --gamertag VotreGamertag

# Titre, répertoire de sortie et niveau de compression personnalisés (Zstd 1-22, défaut 9)
go run ./cmd/levelup backup \
  --gamertag VotreGamertag \
  --title halo_infinite \
  --output-dir ./mes_backups \
  --compression-level 15
```

Chaque table est écrite en `<table>_<timestamp>.parquet`, plus un `backup_metadata_<timestamp>.json` décrivant les lignes/tailles.

### Restore

```bash
cd apps/go-api

# Restaurer toutes les tables depuis un répertoire de backup
go run ./cmd/levelup restore \
  --gamertag VotreGamertag \
  --backup-dir ./data/titles/halo_infinite/backups/VotreGamertag

# Restaurer des tables précises, en remplaçant les données existantes
go run ./cmd/levelup restore \
  --gamertag VotreGamertag \
  --backup-dir ./mes_backups \
  --tables player_match_enrichment,match_citations \
  --replace

# Inspecter sans écrire
go run ./cmd/levelup restore --gamertag VotreGamertag --backup-dir ./mes_backups --dry-run
```

| Flag | Effet |
|------|-------|
| `--title` | Slug du titre cible (défaut `halo_infinite`) |
| `--tables T1,T2` | Restaurer uniquement ces tables (défaut : toutes) |
| `--replace` | `DROP TABLE` avant import (sinon les lignes sont ajoutées) |
| `--dry-run` | Lister sans modifier |

---

## 3. Restaurer les CSR historiques (`levelup restore-csr`)

Récupération one-shot et idempotente des valeurs CSR par-match depuis un backup DuckDB legacy (ex. un `shared_matches_v2.duckdb` extrait d'un snapshot). Prévu pour le cas où les CSR ont été écrasés par LUSR avant la mise en place du garde-fou SQL.

```bash
cd apps/go-api

# Inspecter le schéma legacy et compter sans écrire
go run ./cmd/levelup restore-csr \
  --gamertag VotreGamertag \
  --backup /chemin/vers/legacy/shared_matches_v2.duckdb \
  --dry-run

# Appliquer (overwrite supprime les LUSR fautifs sur les matchs concernés)
go run ./cmd/levelup restore-csr \
  --gamertag VotreGamertag \
  --backup /chemin/vers/legacy/shared_matches_v2.duckdb \
  --mode overwrite
```

| Flag | Effet |
|------|-------|
| `--title` | Slug du titre cible (défaut `halo_infinite`) |
| `--backup` | Chemin vers le `.duckdb` legacy (attaché en lecture seule) |
| `--mode preserve\|overwrite` | `overwrite` supprime les LUSR fautifs sur les matchs à restaurer ; `preserve` les conserve |
| `--dry-run` | Inspecter schéma et comptages uniquement |

La commande attache le backup en lecture seule, localise la table source des CSR, et réinsère les CSR dans `match_skill_rank` avec `ON CONFLICT DO NOTHING`.

---

## Notes pratiques

- **Restaurer avant d'écraser les données live** : préférer une restauration en staging (`go run ./cmd/restore`), inspecter, puis n'utiliser `--live` qu'avec le serveur arrêté.
- **Migration vers une nouvelle machine** : copier le dépôt restic (ou le dossier Parquet par joueur) et restaurer sur l'hôte cible avec les mêmes variables d'environnement.
- **Intégrité** : le scheduler enregistre les résultats de `PRAGMA integrity_check` dans le `.manifest.json` du staging ; une DB dégradée est tout de même snapshotée avec un warning, pour ne jamais perdre une copie récupérable.

> Note legacy : les anciens scripts Python `scripts/backup_player.py` / `scripts/restore_player.py` ont été supprimés. Utiliser les commandes Go ci-dessus.
