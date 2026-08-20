# Plan Backup DuckDB — pkg/duckdbbackup + Restic

> Rédigé le 2026-05-25. Révisé pour architecture générique (`pkg/`).
> Basé sur l'audit de `internal/ops/backup.go`, `internal/scheduler/data_health_check.go`
> et `cmd/server/main.go`.

---

## 1. Objectifs

| Objectif | Détail |
|---|---|
| **Package générique réutilisable** | `pkg/duckdbbackup/` — zéro import `internal/`, copiable dans tout projet Go+DuckDB |
| **Intégré au serveur** | Goroutine dans `main.go`, même pattern que `HealthScheduler` |
| **Pas de snapshot inutile** | Fingerprint `mtime+size` : si aucune DB n'a changé → 0 snapshot restic créé |
| **Warehouse + players** | Couvre toutes les DBs via une fonction de découverte injectée par LevelUp |
| **Restore simple** | `restic restore latest` → lecture Parquet ou `COPY FROM` |

---

## 2. Prérequis : Restic

### Installation Windows

```powershell
# Option A — Winget
winget install restic.restic

# Option B — Scoop
scoop install restic

# Option C — Binaire direct
# https://github.com/restic/restic/releases → restic_0.xx.x_windows_amd64.zip
# → extraire restic.exe dans PATH (ex: C:\tools\)
```

### Initialiser le repo restic une seule fois

```powershell
$env:RESTIC_REPOSITORY  = "D:\Backups\levelup"
$env:RESTIC_PASSWORD_FILE = "D:\Backups\levelup.key"  # fichier contenant le mot de passe

restic init
```

### Variables d'environnement

| Variable | Exemple | Obligatoire |
|---|---|---|
| `RESTIC_REPOSITORY` | `D:\Backups\levelup` | Oui si backup activé |
| `RESTIC_PASSWORD_FILE` | `D:\Backups\levelup.key` | Recommandé (vs `RESTIC_PASSWORD`) |
| `RESTIC_PASSWORD` | `monsecret` | Alternative à `PASSWORD_FILE` |
| `LEVELUP_BACKUP_ENABLED` | `true` | Non (défaut: `false`) |
| `LEVELUP_BACKUP_DIR` | `D:\Backups\levelup-staging` | Non (défaut: `{repo_root}/data/backups`) |
| `LEVELUP_BACKUP_INTERVAL` | `6h` | Non (défaut: `6h`) |
| `LEVELUP_BACKUP_KEEP_DAILY` | `7` | Non (défaut: `7`) |
| `LEVELUP_BACKUP_KEEP_WEEKLY` | `4` | Non (défaut: `4`) |
| `LEVELUP_BACKUP_KEEP_MONTHLY` | `12` | Non (défaut: `12`) |

---

## 3. Architecture

```
pkg/duckdbbackup/                 ← GÉNÉRIQUE — zéro import levelup/go-api/internal/
  config.go                       ← Config struct + FromEnv()
  target.go                       ← type Target{Key, Path string}
  manifest.go                     ← Manifest — fingerprints JSON par Target
  exporter.go                     ← DuckDB → Parquet+zstd (seule dep: duckdb-go/v2)
  restic.go                       ← ResticClient — wrapper exec.Command("restic", ...)
  scheduler.go                    ← Scheduler — goroutine + ticker, orchestration

apps/go-api/internal/ops/
  backup_service.go               ← ADAPTATEUR LevelUp — ~30 lignes
                                    construit []Target depuis PathResolver
                                    + délègue à pkg/duckdbbackup.New()
  backup.go                       ← existant : corriger bug read-only (voir §6)
```

### Principe de réutilisation

Le package générique ne sait pas qu'il protège des bases Halo. Il reçoit une liste de
`Target{Key, Path}` via une fonction de découverte injectée par le projet hôte :

```go
// pkg/duckdbbackup — interface publique principale
func New(cfg Config, discover func() ([]Target, error)) *Scheduler
```

**Dans LevelUp :**
```go
// internal/ops/backup_service.go
func NewLevelUpBackupScheduler(cfg BackupConfig, pr *title.PathResolver) *duckdbbackup.Scheduler {
    return duckdbbackup.New(cfg.toPkgConfig(), func() ([]duckdbbackup.Target, error) {
        return discoverLevelUpDBs(pr)  // ~15 lignes, PathResolver
    })
}
```

**Dans un autre projet Go+DuckDB :**
```go
duckdbbackup.New(cfg, func() ([]duckdbbackup.Target, error) {
    return []duckdbbackup.Target{
        {Key: "main",     Path: "/data/myapp.duckdb"},
        {Key: "analytics", Path: "/data/analytics.duckdb"},
    }, nil
})
```

---

## 4. Détail du package `pkg/duckdbbackup/`

### 4.1 `target.go`

```go
package duckdbbackup

// Target décrit une base DuckDB à sauvegarder.
type Target struct {
    Key  string // nom court (manifest + logs) : "shared_matches_v2", "player:Chocoboflor"
    Path string // chemin absolu vers le fichier .duckdb
}

// Result résume le résultat d'un cycle de backup.
type Result struct {
    SnapshotID   string            // ID restic du snapshot créé ("" si skip)
    Skipped      bool              // true si aucune DB modifiée
    ExportedDBs  []string          // Keys des DBs effectivement exportées
    Duration     time.Duration
}
```

### 4.2 `config.go`

```go
package duckdbbackup

// Config centralise tous les paramètres du backup.
// Instanciable manuellement ou via FromEnv() pour les projets qui lisent les
// variables d'environnement standard (RESTIC_*, LEVELUP_BACKUP_*).
type Config struct {
    Enabled          bool
    BackupDir        string        // dossier staging local
    Interval         time.Duration // entre deux cycles (défaut: 6h)
    KeepDaily        int           // rétention restic (défaut: 7)
    KeepWeekly       int           // (défaut: 4)
    KeepMonthly      int           // (défaut: 12)
    ResticBin        string        // chemin binaire restic (défaut: "restic" dans PATH)
    ResticRepo       string        // RESTIC_REPOSITORY
    ResticPassword   string        // RESTIC_PASSWORD
    ResticPwdFile    string        // RESTIC_PASSWORD_FILE
    CompressionLevel int           // Zstd 1-22 (défaut: 9)
}

// FromEnv construit une Config depuis les variables d'environnement standard.
// Préfixe env : BACKUP_* (générique) — le projet hôte peut aussi construire
// manuellement depuis son propre préfixe.
func FromEnv() Config
```

### 4.3 `manifest.go`

```go
package duckdbbackup

// Manifest persiste les fingerprints du dernier backup par Target.
// Stocké dans {BackupDir}/.manifest.json
type Manifest struct {
    LastBackupAt time.Time               `json:"last_backup_at"`
    Databases    map[string]fingerprint  `json:"databases"`
    path         string
}

type fingerprint struct {
    Path           string    `json:"path"`
    Mtime          time.Time `json:"mtime"`
    SizeBytes      int64     `json:"size_bytes"`
    LastBackedUpAt time.Time `json:"last_backed_up_at"`
}

func LoadManifest(path string) (*Manifest, error)           // crée vide si absent
func (m *Manifest) HasChanged(t Target) (bool, error)       // os.Stat → compare mtime+size
func (m *Manifest) MarkSaved(t Target) error                // met à jour le fingerprint
func (m *Manifest) Save() error                             // écrit le JSON
```

**Pourquoi `mtime + size_bytes` ?**
DuckDB met à jour le `mtime` du fichier à chaque checkpoint (toute écriture valide).
Si `mtime` et `size` n'ont pas changé depuis le dernier backup → la DB est intacte.
C'est O(1) (un seul appel `os.Stat`), pas de connexion à la DB nécessaire.

### 4.4 `exporter.go`

```go
package duckdbbackup

// ExportTarget exporte toutes les tables BASE TABLE d'une DB cible en Parquet+zstd
// dans outputDir/. Utilise une connexion ?access_mode=read_only.
// Retourne le nombre de tables exportées.
func ExportTarget(ctx context.Context, t Target, outputDir string, compressionLevel int) (int, error)
```

Réutilise la logique de `backup.go` (`exportTableToParquet`, `listBaseTables`) mais
**sans les types LevelUp** et avec la correction du mode read-only.

### 4.5 `restic.go`

```go
package duckdbbackup

type ResticClient struct {
    cfg Config
}

func NewResticClient(cfg Config) *ResticClient

// IsAvailable vérifie que le binaire restic est dans le PATH.
func (r *ResticClient) IsAvailable() bool

// EnsureInit initialise le repo si absent (restic init — idempotent).
func (r *ResticClient) EnsureInit(ctx context.Context) error

// Backup crée un snapshot de stagingDir. Retourne l'ID du snapshot.
func (r *ResticClient) Backup(ctx context.Context, stagingDir string) (string, error)

// Forget applique la politique de rétention + prune.
func (r *ResticClient) Forget(ctx context.Context) error

// env() fusionne os.Environ() + RESTIC_REPOSITORY + RESTIC_PASSWORD[_FILE].
// Passé à exec.Command via cmd.Env pour ne pas polluer l'env du process principal.
func (r *ResticClient) env() []string
```

### 4.6 `scheduler.go`

```go
package duckdbbackup

type Scheduler struct {
    cfg      Config
    discover func() ([]Target, error)
    restic   *ResticClient
}

func New(cfg Config, discover func() ([]Target, error)) *Scheduler

// Run démarre la boucle périodique. Appel en goroutine.
// Premier cycle immédiat au démarrage (même comportement que HealthScheduler).
func (s *Scheduler) Run(ctx context.Context)

// RunOnce exécute un cycle unique (testable sans goroutine).
func (s *Scheduler) RunOnce(ctx context.Context) (*Result, error)
```

**Algorithme d'un cycle :**

```
1. discover() → []Target (liste dynamique des DBs)
2. Pour chaque Target :
     manifest.HasChanged(t) ?
       oui → exporter vers staging/{t.Key}/, manifest.MarkSaved(t)
       non → skip
3. Si 0 Target exporté → log "aucune modification" + return (0 snapshot)
4. restic.EnsureInit(ctx)
5. restic.Backup(ctx, stagingDir) → snapshotID
6. restic.Forget(ctx)
7. manifest.Save()
8. return Result{SnapshotID, ExportedDBs, Duration}
```

---

## 5. Adaptateur LevelUp (`internal/ops/backup_service.go`)

```go
package ops

// NewLevelUpBackupScheduler crée un Scheduler configuré pour LevelUp.
// Toute la logique réside dans pkg/duckdbbackup ; cet adaptateur ne fait
// que construire la liste des Target depuis PathResolver.
func NewLevelUpBackupScheduler(cfg BackupConfig, pr *title.PathResolver) *duckdbbackup.Scheduler {
    return duckdbbackup.New(cfg.toPkgConfig(), func() ([]duckdbbackup.Target, error) {
        return discoverLevelUpDBs(pr)
    })
}

// discoverLevelUpDBs liste toutes les DBs DuckDB du projet :
// 4 warehouse fixes + 1 par joueur dans data/titles/halo_infinite/players/.
func discoverLevelUpDBs(pr *title.PathResolver) ([]duckdbbackup.Target, error) {
    slug := title.DefaultSlug
    targets := []duckdbbackup.Target{
        {Key: "shared_matches_v2", Path: pr.SharedDBPath(slug)},
        {Key: "metadata",          Path: pr.MetadataDBPath(slug)},
        {Key: "shared_pve",        Path: pr.SharedPVEDBPath(slug)},
        {Key: "shared_social",     Path: pr.SharedSocialDBPath(slug)},
    }
    // Joueurs : scanner le dossier players/
    playersDir := filepath.Join(pr.TitleDataDir(slug), "players")
    entries, err := os.ReadDir(playersDir)
    if err != nil {
        return targets, nil // pas de joueur = pas bloquant
    }
    for _, e := range entries {
        if !e.IsDir() {
            continue
        }
        targets = append(targets, duckdbbackup.Target{
            Key:  "player:" + e.Name(),
            Path: pr.PlayerDBPath(slug, e.Name()),
        })
    }
    return targets, nil
}
```

---

## 6. Corrections `internal/ops/backup.go`

**Bug à corriger en priorité** (ligne 62) :

```go
// AVANT — ouvre en RW, conflit si le serveur tourne avec la même DB
db, err := sql.Open("duckdb", opts.PlayerDBPath)

// APRÈS — read-only, compatible avec toute connexion déjà ouverte
db, err := sql.Open("duckdb", opts.PlayerDBPath+"?access_mode=read_only")
```

`backup.go` peut rester tel quel pour les usages CLI existants.
`pkg/duckdbbackup/exporter.go` réimplémente `ExportTarget` avec le mode read-only
intégré dès le départ — pas de copier-coller, juste l'extraction propre.

---

## 7. Modifications `config.go` + `main.go`

### `internal/config/config.go`

```go
// BackupConfig est la vue LevelUp de la config backup.
// Convertie en duckdbbackup.Config via toPkgConfig().
type BackupConfig struct {
    Enabled      bool
    BackupDir    string
    Interval     time.Duration
    KeepDaily    int
    KeepWeekly   int
    KeepMonthly  int
}

func loadBackupConfig(repoRoot string) BackupConfig  // lit LEVELUP_BACKUP_*

func (c BackupConfig) toPkgConfig() duckdbbackup.Config // + lit RESTIC_* depuis env
```

### `cmd/server/main.go`

Après le `healthScheduler` (~ligne 674) :

```go
if cfg.Backup.Enabled {
    backupSched := ops.NewLevelUpBackupScheduler(cfg.Backup, pr)
    schedulerWG.Add(1)
    go func() {
        defer schedulerWG.Done()
        backupSched.Run(schedulerCtx)
    }()
    slog.Info("backup: scheduler démarré", "interval", cfg.Backup.Interval)
} else {
    slog.Debug("backup: désactivé (LEVELUP_BACKUP_ENABLED non défini)")
}
```

---

## 8. Séquence d'exécution (cycle complet)

```
T+0h  boot → BackupScheduler.Run() démarre → premier cycle immédiat
      manifest absent → toutes DBs = "modifiées"
      Export 9 DBs → staging/
      restic backup staging/  → snapshot #1
      restic forget --prune
      manifest.Save()

T+6h  tick → 2 DBs modifiées (shared + Chocoboflor)
      Export 2 DBs → staging/ (écrase les anciens Parquet de ces 2 DBs)
      restic backup staging/  → snapshot #2 (dédup : seuls les nouveaux blocs stockés)
      restic forget --prune
      manifest.Save()

T+12h tick → 0 DB modifiée depuis T+6h
      log "backup: aucune modification, cycle ignoré"
      return → 0 snapshot créé ✓
```

---

## 9. Restore

```powershell
# Lister les snapshots
restic -r D:\Backups\levelup snapshots

# Restaurer le dernier snapshot
restic -r D:\Backups\levelup restore latest --target D:\Temp\restore

# Réimporter une table dans DuckDB
# COPY match_registry FROM 'D:\Temp\restore\shared_matches_v2\match_registry_*.parquet';

# Restaurer un snapshot passé
restic -r D:\Backups\levelup restore <snapshot-id> --target D:\Temp\restore-old
```

---

## 10. Points d'attention

**Connexions DuckDB concurrent** : `ExportTarget` ouvre toujours en `?access_mode=read_only`.
Compatible avec les connexions RW déjà ouvertes dans le même process Go.

**Restic absent** : `ResticClient.IsAvailable()` vérifie le PATH au boot du scheduler.
Si absent → `slog.Warn` + scheduler désactivé proprement, pas de crash.

**Repo restic non initialisé** : `EnsureInit()` est appelé avant le premier `Backup()`.
Idempotent — sans effet si le repo existe déjà.

**Taille staging** : le dossier staging contient les derniers Parquet de chaque DB.
Espace estimé : 10–30 % de la taille totale des `.duckdb` (Parquet+zstd compresse très bien).
Restic déduplique les blocs entre snapshots — l'espace restic croît lentement.

**Timeouts par étape** :
- Export Parquet (toutes DBs) : 10 min
- `restic backup` : 5 min
- `restic forget --prune` : 2 min

---

## 11. Checklist d'implémentation

- [ ] **`pkg/duckdbbackup/target.go`** — types `Target`, `Result`
- [ ] **`pkg/duckdbbackup/config.go`** — `Config`, `FromEnv()`
- [ ] **`pkg/duckdbbackup/manifest.go`** — `Manifest`, fingerprints JSON
- [ ] **`pkg/duckdbbackup/exporter.go`** — `ExportTarget()` read-only
- [ ] **`pkg/duckdbbackup/restic.go`** — `ResticClient`, `IsAvailable`, `EnsureInit`, `Backup`, `Forget`
- [ ] **`pkg/duckdbbackup/scheduler.go`** — `Scheduler`, `New`, `Run`, `RunOnce`
- [ ] **`internal/ops/backup_service.go`** — adaptateur LevelUp (~30 lignes)
- [ ] **`internal/ops/backup.go`** — corriger bug read-only ligne 62
- [ ] **`internal/config/config.go`** — `BackupConfig`, `loadBackupConfig()`
- [ ] **`cmd/server/main.go`** — wiring (~8 lignes après healthScheduler)
- [ ] **Tests** — `manifest_test.go` (HasChanged, mock os.Stat via interface) + `scheduler_test.go` (RunOnce sur fixtures, skip si restic absent)
- [ ] **`.env.local.example`** — documenter les variables LEVELUP_BACKUP_* + RESTIC_*

---

## 12. Démarrage rapide (une fois implémenté)

```powershell
# .env.local
LEVELUP_BACKUP_ENABLED=true
LEVELUP_BACKUP_INTERVAL=6h
RESTIC_REPOSITORY=D:\Backups\levelup
RESTIC_PASSWORD_FILE=D:\Backups\levelup.key

# Initialiser restic (une seule fois)
restic -r D:\Backups\levelup init

# Le backup démarre automatiquement avec le serveur
air
```
