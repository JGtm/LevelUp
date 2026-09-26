# PLAN — Fix timestamps médias + ré-association post-timezone

**Date de création :** 2026-04-24  
**Branche cible :** `fix/media-timestamps` (depuis `feat/v7-assets-abstraction`)  
**Statut :** 📋 À implémenter

---

## 1. Contexte et root causes

### 1.1 Root cause A — DST dans `AssociateMediaWithMatches`

`IndexMedia` ([ops/media.go:94](../apps/go-api/internal/ops/media.go)) ouvre DuckDB via
`sql.Open("duckdb", targetPath)` **directement**, en bypassant `OpenReadWrite` du package
`platform/duckdb`. Conséquence : le `SET TimeZone` appliqué par `connInitFn` n'est **jamais**
exécuté sur cette connexion.

La requête d'association ([ops/media.go:157–168](../apps/go-api/internal/ops/media.go)) compare :

```sql
ABS(DATEDIFF('minute', mf.capture_start_utc, mr.start_time)) <= tolerance
```

- `capture_start_utc` → `TIMESTAMPTZ` : UTC réel (correct)
- `mr.start_time` → `TIMESTAMP` naïf : stocké en heure Paris par la session Python

Sans `SET TimeZone='Europe/Paris'`, DuckDB interprète les `TIMESTAMP` naïfs comme **UTC**,
introduisant un décalage de **+1 h en hiver (CET)** ou **+2 h en été (CEST)**. Le DST n'est
**jamais** le même d'un match à l'autre → les associations ratent de façon aléatoire selon la
saison du match.

**Note DST et ré-indexation d'archives :** `SET TimeZone='Europe/Paris'` utilise la base
IANA — DuckDB résout l'offset **en fonction de la date du timestamp traité**, pas de la date
d'exécution. Ré-indexer aujourd'hui une vidéo du 12 décembre utilisera automatiquement CET
(+1 h), non CEST. Cette propriété est valide pour les nouvelles associations comme pour la
ré-association d'archives.

**Deuxième problème dans le SQL actuel :** la comparaison est uniquement avec `start_time`.
Une capture faite à la 10e minute d'un match de 12 min a un delta de 10 min par rapport à
`start_time` — elle serait ratée avec une tolérance de 5 min. La correction utilise la
fenêtre complète `[start_time, end_time]` + un petit buffer d'imprécision (voir §3.3).

**Ce bug existe depuis le portage Go de l'indexeur.** Le fix timezone du 23 avril (commit
`f4a0c464`) a corrigé l'affichage mais pas cette couche d'association.

### 1.2 Root cause B — `mtime` côté serveur = heure d'upload

Dans `insertMediaFile` ([ops/media.go:332](../apps/go-api/internal/ops/media.go)) :

```go
t := fi.ModTime().UTC()   // mtime après os.WriteFile → heure d'upload sur serveur
captureAt = &t
```

`os.WriteFile` crée le fichier à l'instant de l'upload → `mtime` est l'heure d'upload, pas
l'heure de capture. En local, le fichier est copié depuis la Xbox/PC et le `mtime` reflète
l'heure de capture. Sur un serveur, c'est toujours l'heure du transfert.

Pour OBS, le nom de fichier ne contient pas de datetime lisible. Seul `file.lastModified`
(mtime du fichier côté client, avant upload) est fiable.

### 1.3 Impact cumulé

Les associations dans `media_match_associations` créées **avant le fix DST** sont potentiellement
fausses (décalage ±1 h/2 h dans la fenêtre de tolérance). Il faut une ré-association complète
avec backup préalable.

---

## 2. Architecture des changements

```
domain/media.go
  UploadedFile        + CaptureTimeUnix *int64       (client-provided mtime)
  MediaIndexOptions   + Timezone string               (propagé jusqu'aux ops)
                      + CaptureTimes map[string]*int64 (basename → unix ts)
                      ToleranceMin renommé BufferMin    (2 min par défaut — buffer autour de la fenêtre match)
  ReassociateRequest  NEW
  ReassociateResult   NEW

ops/media.go
  sanitizeMediaTimezone(tz string) string             NEW (copie locale de platform/duckdb)
  parseCaptureTimeFromFilename(name string, loc *time.Location) *time.Time  NEW
  insertMediaFile     signature étendue (captureTimeUnix *int64, loc *time.Location)
  IndexMedia          SET TimeZone après sql.Open ; loc calculé depuis opts.Timezone
                      CaptureTimes transmis par basename à insertMediaFile
  AssociateMediaWithMatches  signature + timezone string ; SET TimeZone appliqué
                              SQL : BETWEEN (start_time - buffer) AND (end_time + buffer)

api/handlers/media.go
  parseUploadedFiles  lire champ form "capture_times" (JSON []int64)
  PostReassociateMedia  NEW handler + route enregistrée

service/media_service.go
  MediaService        + champ timezone string (sanitizé à l'injection dans NewMediaService)
  NewMediaService     + paramètre timezone string — appel sanitizeMediaTimezone ici et stocker
  UploadMedia         remplir CaptureTimes + Timezone dans MediaIndexOptions
  ReassociateMedia    NEW (backup + vide + relance)

api/registry.go
  ServiceRegistry     + champ timezone string
  NewServiceRegistry  initialiser timezone depuis cfg.UserTimezone
  Media()             passer r.timezone à NewMediaService
  MediaUpload()       passer r.timezone à NewMediaService

cmd/levelup/cmd_data.go
  runIndexMedia       ajouter Timezone: cfg.UserTimezone dans MediaIndexOptions

platform/settings/store.go
  MediaToleranceMinutes renommé à MediaBufferMinutes (json: "media_buffer_minutes")
  défaut 10 → 2 (sémantique change : buffer autour de [start, end] vs distance depuis start)

port/services.go
  MediaService interface + ReassociateMedia(ctx, req) (*ReassociateResult, error)

apps/web/src/
  composant upload    envoyer champ "capture_times" (file.lastModified / 1000)
```

---

## 3. Plan d'implémentation détaillé

### Phase 1 — Propagation timezone dans les ops *(bloquant pour phases 2 et 3)*

#### Étape 1.1 — `domain/media.go` : étendre `MediaIndexOptions` et `UploadedFile`

```go
type UploadedFile struct {
    OriginalName    string
    Data            []byte
    CaptureTimeUnix *int64 // mtime du fichier côté client (secondes Unix), optionnel
}

type MediaIndexOptions struct {
    PlayerDBPath        string
    SharedSocialDBPath  string
    SharedMatchesDBPath string
    CapturesDir         string
    ForceRescan         bool
    BufferMin           int    // buffer en minutes autour de [start_time, end_time] — 0 → défaut 2 min
    Gamertag            string
    Timezone     string            // IANA (ex: "Europe/Paris") — SET TimeZone à l'ouverture
    CaptureTimes map[string]*int64 // basename → unix ts client (optionnel, depuis upload)
}
```

Ajouter également les types de la ré-association :

```go
// ReassociateRequest configure une ré-association forcée des médias.
type ReassociateRequest struct {
    DBPath              string
    SharedSocialDBPath  string
    SharedMatchesDBPath string
    BufferMin           int // 0 → défaut 2 min
}

// ReassociateResult résume le résultat de la ré-association.
type ReassociateResult struct {
    BackupTable  string   `json:"backup_table"`          // nom de la table snapshot
    DeletedAssoc int      `json:"deleted_associations"`  // lignes supprimées
    NewAssoc     int      `json:"new_associations"`      // nouvelles associations créées
    Errors       []string `json:"errors,omitempty"`
}
```

#### Étape 1.2 — `ops/media.go` : `sanitizeMediaTimezone` + `SET TimeZone`

Ajouter `sanitizeMediaTimezone` — copie locale de `sanitizeTimezone` de `platform/duckdb/db.go`
(évite une dépendance croisée ; à consolider dans un `tzutil` partagé plus tard) :

```go
// sanitizeMediaTimezone valide un nom de timezone IANA pour éviter l'injection SQL.
// Retourne "" si la valeur contient des caractères non autorisés.
func sanitizeMediaTimezone(tz string) string {
    for _, c := range tz {
        switch {
        case c >= 'A' && c <= 'Z':
        case c >= 'a' && c <= 'z':
        case c >= '0' && c <= '9':
        case c == '/' || c == '_' || c == '-' || c == '+':
        default:
            return ""
        }
    }
    return tz
}
```

Dans `IndexMedia`, après `sql.Open` et avant `ensureMediaTables` :

```go
tz := sanitizeMediaTimezone(opts.Timezone)
if tz != "" {
    if _, err := db.Exec("SET TimeZone = '" + tz + "'"); err != nil {
        slog.Warn("IndexMedia: SET TimeZone échoué, DST possiblement incorrect",
            "timezone", tz, "err", err)
    } else {
        slog.Debug("IndexMedia: SET TimeZone appliqué", "timezone", tz)
    }
}
```

#### Étape 1.3 — `ops/media.go` : `AssociateMediaWithMatches` — nouveau SQL + timezone

Nouvelle signature :

```go
func AssociateMediaWithMatches(db *sql.DB, sharedMatchesPath string, bufferMin int, timezone string) (int, error)
```

Le paramètre s'appelle désormais `bufferMin` (buffer d'imprécision) et non plus `toleranceMin`
pour refléter la nouvelle sémantique. Valeur par défaut : **2 min**.

En tête de fonction, appliquer `SET TimeZone` (idempotent si déjà fait par `IndexMedia`) :

```go
if tz := sanitizeMediaTimezone(timezone); tz != "" {
    if _, err := db.Exec("SET TimeZone = '" + tz + "'"); err != nil {
        slog.Warn("AssociateMediaWithMatches: SET TimeZone échoué",
            "timezone", tz, "err", err)
    }
}
slog.Debug("AssociateMediaWithMatches: démarrage",
    "shared_matches_path", sharedMatchesPath,
    "buffer_min", bufferMin,
    "timezone", timezone)
```

**Nouveau SQL — fenêtre `[start_time − buffer, end_time + buffer]` :**

```go
q := fmt.Sprintf(`
    INSERT OR IGNORE INTO media_match_associations (media_file_id, match_id, delta_seconds)
    SELECT
        mf.id,
        mr.match_id,
        ABS(DATEDIFF('second', mf.capture_start_utc, mr.start_time)) AS delta_s
    FROM media_files mf
    JOIN shared_matches.match_registry mr
        ON mf.capture_start_utc
               BETWEEN (mr.start_time - INTERVAL '%d minutes')
                   AND (mr.end_time   + INTERVAL '%d minutes')
    WHERE mf.id NOT IN (SELECT media_file_id FROM media_match_associations)
`, bufferMin, bufferMin)
```

Sémantique : la capture doit se situer **dans la fenêtre du match** (de `start_time` à
`end_time`), avec un buffer de quelques minutes pour absorber les captures du scoreboard
post-match ou un léger décalage d'horodatage. `end_time` est disponible dans `match_registry`
(`start_time + duration_seconds`).

Log à la sortie :

```go
slog.Info("AssociateMediaWithMatches: terminé",
    "associations_created", n, "buffer_min", bufferMin, "timezone", timezone)
```

Mettre à jour l'appel dans `IndexMedia` :

```go
assoc, err := AssociateMediaWithMatches(db, opts.SharedMatchesDBPath, opts.BufferMin, opts.Timezone)
```

#### Étape 1.4 — `service/media_service.go` : injecter `timezone`

```go
type MediaService struct {
    repo     port.MediaRepository
    timezone string // IANA sanitizé (empty string si invalide ou absent)
}

// NewMediaService sanitise la timezone à l'injection pour que s.timezone soit toujours sûr
// à utiliser directement dans les requêtes SQL sans re-validation.
func NewMediaService(repo port.MediaRepository, timezone string) *MediaService {
    return &MediaService{repo: repo, timezone: ops.SanitizeMediaTimezone(timezone)}
}
```

> **Action** : exporter `sanitizeMediaTimezone` → `SanitizeMediaTimezone` dans `ops/media.go`
> pour l'appeler depuis `service/`. Elle y reste car c'est un détail d'implémentation ops.
> Le service ne re-sanitise **jamais** `s.timezone` après ce point.

Dans `ReassociateMedia` et `UploadMedia`, utiliser `s.timezone` directement :

```go
if s.timezone != "" {
    _, _ = db.ExecContext(ctx, "SET TimeZone = '"+s.timezone+"'")
}
```

#### Étape 1.5 — `api/registry.go` : propager `timezone` vers `NewMediaService`

Ajouter `timezone string` comme champ de `ServiceRegistry` et l'initialiser depuis `cfg.UserTimezone` :

```go
type ServiceRegistry struct {
    resolve       PlayerResolver
    provider      auth.TokenProvider
    assetResolver assets.Resolver
    timezone      string // IANA depuis cfg.UserTimezone, transmis aux services qui en ont besoin
}

func NewServiceRegistry(cfg *config.AppConfig, provider auth.TokenProvider) *ServiceRegistry {
    return &ServiceRegistry{
        resolve:  ...,  // inchangé
        provider: provider,
        timezone: cfg.UserTimezone,
    }
}
```

Mettre à jour les deux factory methods :

```go
func (r *ServiceRegistry) Media(ctx context.Context, slug string) (port.MediaService, error) {
    pdb, err := r.resolve(ctx, slug)
    ...
    return service.NewMediaService(duckdb.NewMediaRepo(pdb), r.timezone), nil
}

func (r *ServiceRegistry) MediaUpload(...) {
    ...
    svc := service.NewMediaService(duckdb.NewMediaRepo(pdb), r.timezone)
    ...
}
```

#### Étape 1.6 — `cmd/levelup/cmd_data.go` : ajouter `Timezone` dans `runIndexMedia`

Appel direct à `ops.IndexMedia` hors service — doit transmettre la timezone :

```go
result, err := ops.IndexMedia(ops.MediaIndexOptions{
    ...
    BufferMin:    *tolMin,
    Timezone:     cfg.UserTimezone,
})
```

#### Étape 1.7 — `platform/settings/store.go` : renommer `MediaToleranceMinutes` → `MediaBufferMinutes`

```go
// Avant (sémantique : distance max depuis start_time)
MediaToleranceMinutes int `json:"media_tolerance_minutes"` // défaut 10

// Après (sémantique : buffer autour de [start_time, end_time])
MediaBufferMinutes    int `json:"media_buffer_minutes"`    // défaut 2
```

Changer le défaut dans `defaultSettings()` de 10 à 2. Ce changement casse la rétrocompat du
json `app_settings.json` — la clé `media_tolerance_minutes` sera ignorée. Ajouter une
migration one-shot de lecture de l'ancienne clé dans `Apply()` si elle est présente.
    if f.CaptureTimeUnix != nil {
        ts := f.CaptureTimeUnix
        captureTimes[f.OriginalName] = ts
    }
}

idxResult, err := ops.IndexMedia(ops.MediaIndexOptions{
    PlayerDBPath:        req.DBPath,
    SharedSocialDBPath:  req.SharedSocialDBPath,
    SharedMatchesDBPath: req.SharedMatchesDBPath,
    CapturesDir:         req.CapturesDir,
    BufferMin:           tol,
    Timezone:            s.timezone,
    CaptureTimes:        captureTimes,
})
```

---

### Phase 2 — Endpoint de ré-association avec backup

#### Étape 2.1 — `port/services.go` : étendre l'interface `MediaService`

```go
type MediaService interface {
    GetMediaPage(ctx context.Context, req domain.MediaPageRequest) (*domain.MediaPageResponse, error)
    SetMediaLike(ctx context.Context, req domain.MediaLikeRequest) (*domain.MediaLikeResponse, error)
    UploadMedia(ctx context.Context, req domain.UploadRequest) (*domain.UploadResult, error)
    ReassociateMedia(ctx context.Context, req domain.ReassociateRequest) (*domain.ReassociateResult, error)
}
```

#### Étape 2.2 — `service/media_service.go` : implémenter `ReassociateMedia`

Séquence : backup → count → delete → associate.

```go
func (s *MediaService) ReassociateMedia(
    ctx context.Context,
    req domain.ReassociateRequest,
) (*domain.ReassociateResult, error) {
    tol := req.BufferMin
    if tol <= 0 {
        tol = 2
    }

    targetPath := req.DBPath
    if req.SharedSocialDBPath != "" {
        targetPath = req.SharedSocialDBPath
    }

    db, err := sql.Open("duckdb", targetPath)
    if err != nil {
        return nil, fmt.Errorf("ReassociateMedia: ouverture DB: %w", err)
    }
    defer db.Close()

    // s.timezone est déjà sanitizé par NewMediaService — utiliser directement
    if s.timezone != "" {
        _, _ = db.ExecContext(ctx, "SET TimeZone = '"+s.timezone+"'")
    }
    backupTable := fmt.Sprintf("media_match_associations_bak_%d", ts)

    slog.InfoContext(ctx, "ReassociateMedia: création backup",
        "backup_table", backupTable, "target_db", targetPath)

    if _, err := db.ExecContext(ctx,
        "CREATE TABLE "+backupTable+" AS SELECT * FROM media_match_associations",
    ); err != nil {
        return nil, fmt.Errorf("ReassociateMedia: backup: %w", err)
    }

    var deletedCount int
    _ = db.QueryRowContext(ctx,
        "SELECT COUNT(*) FROM media_match_associations",
    ).Scan(&deletedCount)

    if _, err := db.ExecContext(ctx, "DELETE FROM media_match_associations"); err != nil {
        return nil, fmt.Errorf("ReassociateMedia: delete: %w", err)
    }

    slog.InfoContext(ctx, "ReassociateMedia: associations supprimées, relance",
        "deleted", deletedCount, "buffer_min", tol, "timezone", s.timezone)

    newAssoc, assocErr := ops.AssociateMediaWithMatches(db, req.SharedMatchesDBPath, tol, s.timezone)
    result := &domain.ReassociateResult{
        BackupTable:  backupTable,
        DeletedAssoc: deletedCount,
        NewAssoc:     newAssoc,
    }
    if assocErr != nil {
        result.Errors = append(result.Errors, fmt.Sprintf("association: %v", assocErr))
        slog.ErrorContext(ctx, "ReassociateMedia: association échouée", "err", assocErr)
    }

    slog.InfoContext(ctx, "ReassociateMedia: terminé",
        "backup_table", backupTable,
        "deleted", deletedCount,
        "new_associations", newAssoc,
        "errors", len(result.Errors))

    return result, nil
}
```

> Note : `sanitizeMediaTimezone` est définie dans `ops/media.go`. Le service l'appelle via un
> helper local identique (ou extraire en package `tzutil` partagé).

#### Étape 2.3 — `api/handlers/media.go` : handler `PostReassociateMedia`

Route : `POST /api/v1/players/{player_slug}/media/reassociate`

```go
// PostReassociateMedia vide media_match_associations (avec backup), puis relance l'association.
// POST /api/v1/players/{player_slug}/media/reassociate
func (h *MediaHandler) PostReassociateMedia(w http.ResponseWriter, r *http.Request) {
    if h.newUpload == nil {
        writeError(w, http.StatusNotImplemented, "upload_not_configured", "upload factory non configurée")
        return
    }

    slug := chi.URLParam(r, "player_slug")
    svc, _, _, dbPath, sharedSocialDBPath, sharedMatchesDBPath, err := h.newUpload(r.Context(), slug)
    if err != nil {
        writeError(w, http.StatusNotFound, "player_not_found", err.Error())
        return
    }

    var body struct {
        BufferMin int `json:"buffer_min"`
    }
    _ = json.NewDecoder(r.Body).Decode(&body) // corps optionnel

    req := domain.ReassociateRequest{
        DBPath:              dbPath,
        SharedSocialDBPath:  sharedSocialDBPath,
        SharedMatchesDBPath: sharedMatchesDBPath,
        BufferMin:           body.BufferMin,
    }

    result, err := svc.ReassociateMedia(r.Context(), req)
    if err != nil {
        slog.ErrorContext(r.Context(), "reassociate: erreur service",
            "player", slug, "err", err)
        writeError(w, http.StatusInternalServerError, "reassociate_error", err.Error())
        return
    }

    slog.InfoContext(r.Context(), "reassociate: OK",
        "player", slug,
        "backup_table", result.BackupTable,
        "deleted", result.DeletedAssoc,
        "new", result.NewAssoc)

    writeJSON(w, http.StatusOK, result)
    BumpMediaFeedVersion()
}
```

> Enregistrer la route dans le router — chercher `PostUploadMedia` dans `router.go` /
> `registry.go` pour le pattern exact.

---

### Phase 3 — Timestamps fiables à l'upload

#### Étape 3.1 — `ops/media.go` : filename datetime parser Xbox

```go
import "regexp"

// xboxFilenameRe matche le pattern Xbox :
// "Halo Infinite 2024.11.15 - 21.30.45.01.mp4"
// Groupe 1=année 2=mois 3=jour 4=heure 5=min 6=sec
var xboxFilenameRe = regexp.MustCompile(
    `(\d{4})\.(\d{2})\.(\d{2}) - (\d{2})\.(\d{2})\.(\d{2})`)

// parseCaptureTimeFromFilename tente d'extraire la datetime depuis le nom de fichier.
// Retourne nil si aucun pattern connu n'est trouvé.
// La datetime est interprétée comme heure locale (loc), puis convertie en UTC.
func parseCaptureTimeFromFilename(name string, loc *time.Location) *time.Time {
    if loc == nil {
        return nil
    }
    if m := xboxFilenameRe.FindStringSubmatch(name); m != nil {
        year  := mustAtoi(m[1])
        month := mustAtoi(m[2])
        day   := mustAtoi(m[3])
        hour  := mustAtoi(m[4])
        min   := mustAtoi(m[5])
        sec   := mustAtoi(m[6])
        if year == 0 {
            return nil
        }
        t := time.Date(year, time.Month(month), day, hour, min, sec, 0, loc).UTC()
        return &t
    }
    return nil
}

// mustAtoi convertit une string en int, retourne 0 en cas d'erreur.
func mustAtoi(s string) int {
    n, _ := strconv.Atoi(s)
    return n
}
```

#### Étape 3.2 — `ops/media.go` : `insertMediaFile` — chaîne de priorité

Nouvelle signature :

```go
func insertMediaFile(db *sql.DB, path, hash, playerSlug string, captureTimeUnix *int64, loc *time.Location) error
```

Logique de résolution de `capture_start_utc` :

```go
var captureAt *time.Time

// Priorité 1 : datetime parsée depuis le nom de fichier (Xbox)
if t := parseCaptureTimeFromFilename(filepath.Base(path), loc); t != nil {
    captureAt = t
    slog.Debug("insertMediaFile: datetime extraite du nom de fichier",
        "file", filepath.Base(path), "capture_start_utc", *t)
}

// Priorité 2 : mtime client fourni par le navigateur (file.lastModified)
if captureAt == nil && captureTimeUnix != nil && *captureTimeUnix > 0 {
    t := time.Unix(*captureTimeUnix, 0).UTC()
    captureAt = &t
    slog.Debug("insertMediaFile: datetime depuis file.lastModified client",
        "file", filepath.Base(path), "capture_start_utc", t)
}

// Priorité 3 : mtime filesystem (dernier recours — incorrect sur serveur)
if captureAt == nil {
    fi, _ := os.Stat(path)
    if fi != nil {
        t := fi.ModTime().UTC()
        captureAt = &t
        slog.Debug("insertMediaFile: datetime depuis mtime filesystem (fallback)",
            "file", filepath.Base(path), "capture_start_utc", t)
    }
}
```

#### Étape 3.3 — `ops/media.go` : propager `loc` et `CaptureTimes` dans `IndexMedia`

Après le bloc `SET TimeZone`, calculer `loc` une seule fois :

```go
var loc *time.Location
if opts.Timezone != "" {
    if l, err := time.LoadLocation(opts.Timezone); err == nil {
        loc = l
    } else {
        slog.Warn("IndexMedia: timezone invalide pour filename parser",
            "timezone", opts.Timezone, "err", err)
    }
}
```

Dans la boucle de scan, récupérer le timestamp client par basename :

```go
var clientTs *int64
if opts.CaptureTimes != nil {
    clientTs = opts.CaptureTimes[filepath.Base(path)]
}
if err := insertMediaFile(db, path, hash, opts.Gamertag, clientTs, loc); err != nil {
    ...
}
```

#### Étape 3.4 — `api/handlers/media.go` : lire le champ `capture_times`

Dans `parseUploadedFiles`, après la boucle sur `headers` :

```go
var captureTimes []int64
if raw := r.FormValue("capture_times"); raw != "" {
    if err := json.Unmarshal([]byte(raw), &captureTimes); err != nil {
        slog.Warn("upload: capture_times invalide, ignoré", "err", err)
        captureTimes = nil
    } else {
        slog.Debug("upload: capture_times reçu", "count", len(captureTimes))
    }
}
for i := range out {
    if i < len(captureTimes) && captureTimes[i] > 0 {
        ts := captureTimes[i]
        out[i].CaptureTimeUnix = &ts
    }
}
```

#### Étape 3.5 — Frontend `apps/web/src/` : envoyer `capture_times`

Chercher le call site `formData.append('files'` dans les composants TypeScript pour localiser
le composant upload. Ajouter **avant** le `fetch` :

```typescript
const captureTimes = Array.from(files).map(f => Math.floor(f.lastModified / 1000));
formData.append('capture_times', JSON.stringify(captureTimes));
```

---

## 4. Logging — couverture complète

| Localisation | Niveau | Message |  Champs slog |
|---|---|---|---|
| `IndexMedia` début | `Debug` | `"IndexMedia: démarrage"` | `captures_dir`, `buffer_min`, `timezone`, `force_rescan` |
| `IndexMedia` SET TimeZone OK | `Debug` | `"IndexMedia: SET TimeZone appliqué"` | `timezone` |
| `IndexMedia` SET TimeZone KO | `Warn` | `"IndexMedia: SET TimeZone échoué, DST possiblement incorrect"` | `timezone`, `err` |
| `IndexMedia` scan terminé | `Debug` | `"IndexMedia: scan répertoire"` | `scanned`, `new_files`, `skipped` |
| `IndexMedia` fin | `Info` | `"IndexMedia: terminé"` | `scanned`, `new_files`, `associated`, `errors` |
| `AssociateMediaWithMatches` début | `Debug` | `"AssociateMediaWithMatches: démarrage"` | `shared_matches_path`, `buffer_min`, `timezone` |
| `AssociateMediaWithMatches` fin OK | `Info` | `"AssociateMediaWithMatches: terminé"` | `associations_created` |
| `AssociateMediaWithMatches` erreur | `Error` | `"AssociateMediaWithMatches: erreur SQL"` | `err` |
| `insertMediaFile` filename parser | `Debug` | `"insertMediaFile: datetime extraite du nom de fichier"` | `file`, `capture_start_utc` |
| `insertMediaFile` client ts | `Debug` | `"insertMediaFile: datetime depuis file.lastModified client"` | `file`, `capture_start_utc` |
| `insertMediaFile` mtime fallback | `Debug` | `"insertMediaFile: datetime depuis mtime filesystem (fallback)"` | `file`, `capture_start_utc` |
| `ReassociateMedia` backup | `Info` | `"ReassociateMedia: création backup"` | `backup_table`, `target_db` |
| `ReassociateMedia` backup KO | `Error` | `"ReassociateMedia: backup échoué"` | `err` |
| `ReassociateMedia` delete+relance | `Info` | `"ReassociateMedia: associations supprimées, relance"` | `deleted`, `buffer_min`, `timezone` |
| `ReassociateMedia` association KO | `Error` | `"ReassociateMedia: association échouée"` | `err` |
| `ReassociateMedia` fin | `Info` | `"ReassociateMedia: terminé"` | `backup_table`, `deleted`, `new_associations`, `errors` |
| `PostReassociateMedia` OK | `Info` | `"reassociate: OK"` | `player`, `backup_table`, `deleted`, `new` |
| `PostReassociateMedia` KO | `Error` | `"reassociate: erreur service"` | `player`, `err` |
| `parseUploadedFiles` capture_times | `Debug` | `"upload: capture_times reçu"` | `count` |
| `parseUploadedFiles` capture_times KO | `Warn` | `"upload: capture_times invalide, ignoré"` | `err` |

---

## 5. Tests — couverture complète

### 5.1 `apps/go-api/internal/ops/` — étendre `media_backup_cgo_test.go`

> Tests sur DuckDB réel (tag `cgo`). Utiliser `t.TempDir()` pour tous les fichiers temporaires.

#### Parser filename

| Fonction de test | Ce qu'elle vérifie |
|---|---|
| `TestParseCaptureTimeFromFilename_XboxPattern` | Filename Xbox classique → datetime UTC correcte |
| `TestParseCaptureTimeFromFilename_XboxPattern_CET` | Heure Paris hiver → UTC = Paris − 1 h |
| `TestParseCaptureTimeFromFilename_XboxPattern_CEST` | Heure Paris été → UTC = Paris − 2 h |
| `TestParseCaptureTimeFromFilename_OBSPattern` | Filename OBS sans datetime → `nil` |
| `TestParseCaptureTimeFromFilename_Empty` | Nom vide → `nil` |
| `TestParseCaptureTimeFromFilename_NilLoc` | `loc == nil` → `nil` sans panic |
| `TestParseCaptureTimeFromFilename_PartialMatch` | Pattern incomplet → `nil` |

#### `insertMediaFile` — chaîne de priorité

| Fonction de test | Ce qu'elle vérifie |
|---|---|
| `TestInsertMediaFile_XboxFilenameUsed` | Filename Xbox → `capture_start_utc` = datetime du nom |
| `TestInsertMediaFile_ClientTimestampUsed` | `captureTimeUnix` fourni + filename OBS → timestamp client utilisé |
| `TestInsertMediaFile_FallbackMtime` | Pas de client ts, pas de match filename → `capture_start_utc` ≈ mtime |
| `TestInsertMediaFile_XboxPriorityOverClient` | Filename Xbox ET `captureTimeUnix` → Xbox gagne |
| `TestInsertMediaFile_ZeroClientTimestamp_Ignored` | `captureTimeUnix = 0` → traité comme absent |

#### `AssociateMediaWithMatches` — DST

| Fonction de test | Ce qu'elle vérifie |
|---|---|
| `TestAssociateMediaWithMatches_WithTimezone_CET` | `start_time` naïf hiver, capture pendant le match → associé avec timezone |
| `TestAssociateMediaWithMatches_WithTimezone_CEST` | Même chose en été (décalage +2 h) → associé |
| `TestAssociateMediaWithMatches_WithoutTimezone_MissesMatch` | Sans `SET TimeZone` → le même cas rate (régression guard) |
| `TestAssociateMediaWithMatches_EmptySharedPath` | `sharedMatchesPath == ""` → retourne `0, nil` |
| `TestAssociateMediaWithMatches_CaptureBeforeStart_InBuffer` | Capture 1 min avant `start_time` → dans le buffer → associé |
| `TestAssociateMediaWithMatches_CaptureAfterEnd_InBuffer` | Capture 1 min après `end_time` → dans le buffer → associé |
| `TestAssociateMediaWithMatches_CaptureWayBeforeStart_Rejected` | Capture 5 min avant `start_time` (hors buffer 2 min) → non associé |
| `TestAssociateMediaWithMatches_CaptureAfterEnd_OutOfBuffer` | Capture 5 min après `end_time` → non associé |
| `TestAssociateMediaWithMatches_ReindexOldArchive_CETCorrect` | Ré-indexation aujourd'hui d'une vidéo de décembre → CET appliqué (pas CEST) |

**Setup pour les tests DST :**

```
-- Match CEST (été) de 10 min :
match_registry.start_time = '2024-07-15 21:30:00'  (naïf Paris été)
match_registry.end_time   = '2024-07-15 21:40:00'  (naïf Paris été)

-- Capture au milieu du match :
media_files.capture_start_utc = '2024-07-15 19:35:00Z'  (UTC = 21h35 Paris CEST)
```

Sans `SET TimeZone` : DuckDB traite `21:30` naïf comme UTC → capture `19:35Z` hors de
la fenêtre `[21:30Z, 21:40Z]` → non associé.  
Avec `SET TimeZone='Europe/Paris'` : `21:30 Paris été = 19:30Z`, `21:40 = 19:40Z` → capture
`19:35Z` dans `[19:30Z, 19:40Z]` → associé.

**Vérification de la propriété ré-indexation d'archives :**

Pour `TestAssociateMediaWithMatches_ReindexOldArchive_CETCorrect` : insérer un match de
décembre (`start_time='2024-12-12 20:30:00'`, durée 10 min), une capture `'2024-12-12 19:35:00Z'`
(= 20h35 CET). Exécuter le test en avril (CEST). Avec `SET TimeZone='Europe/Paris'`, DuckDB
doit résoudre décembre comme CET → `20:30 CET = 19:30Z` → capture `19:35Z` dans la fenêtre → associé.
Démontre que l'offset est basé sur la date du timestamp, pas la date d'exécution.

#### `IndexMedia` — intégration timezone

| Fonction de test | Ce qu'elle vérifie |
|---|---|
| `TestIndexMedia_WithTimezone_Propagated` | `opts.Timezone` → SET TimeZone appliqué → association correcte |
| `TestIndexMedia_CaptureTimes_UsedByBasename` | `opts.CaptureTimes["clip.mp4"] = &ts` → `insertMediaFile` reçoit le bon ts |

#### `sanitizeMediaTimezone`

| Fonction de test | Ce qu'elle vérifie |
|---|---|
| `TestSanitizeMediaTimezone_Valid` | `"Europe/Paris"` → retourné intact |
| `TestSanitizeMediaTimezone_UTC` | `"UTC"` → retourné intact |
| `TestSanitizeMediaTimezone_Injection` | `"UTC'; DROP TABLE"` → `""` |
| `TestSanitizeMediaTimezone_Empty` | `""` → `""` |

---

### 5.2 `apps/go-api/internal/service/` — étendre `media_service_test.go`

> `mockMediaRepo` à compléter si nécessaire pour les nouvelles méthodes de l'interface.

| Fonction de test | Ce qu'elle vérifie |
|---|---|
| `TestNewMediaService_TimezoneStored` | `NewMediaService(repo, "Europe/Paris")` → `s.timezone == "Europe/Paris"` |
| `TestMediaService_ReassociateMedia_CreatesBackup` | Table `media_match_associations_bak_*` créée |
| `TestMediaService_ReassociateMedia_DeletesOldAssoc` | `media_match_associations` vide après appel |
| `TestMediaService_ReassociateMedia_DefaultBuffer2` | `BufferMin=0` → 2 min passés à l'association |
| `TestMediaService_ReassociateMedia_CustomBuffer` | `BufferMin=5` → 5 min passés |
| `TestMediaService_ReassociateMedia_BackupConflict` | Table backup déjà existante → erreur remontée (edge case) |
| `TestMediaService_ReassociateMedia_SharedSocialPriority` | `SharedSocialDBPath` non vide → utilisé comme DB cible |
| `TestMediaService_UploadMedia_XboxFilenameTimestamp` | Fichier Xbox → `capture_start_utc` correct en DB |
| `TestMediaService_UploadMedia_ClientTimestampPropagated` | `UploadedFile.CaptureTimeUnix` → présent dans `CaptureTimes` |
| `TestMediaService_UploadMedia_TimezonePropagated` | `s.timezone` → `MediaIndexOptions.Timezone` rempli |
| `TestNewMediaService_InvalidTimezone_StoredEmpty` | `NewMediaService(repo, "bad;tz")` → `s.timezone == ""` (sanitisé à l'injection) |

### 5.2b `apps/go-api/internal/api/` — `registry_test.go`

| Fonction de test | Ce qu'elle vérifie |
|---|---|
| `TestServiceRegistry_TimezoneFromCfg` | `cfg.UserTimezone` → stocké dans `ServiceRegistry.timezone` |
| `TestServiceRegistry_Media_PassesTimezoneToService` | `Media()` → `NewMediaService` reçoit `r.timezone` |

### 5.2c `apps/go-api/internal/platform/settings/` — `store_test.go`

| Fonction de test | Ce qu'elle vérifie |
|---|---|
| `TestSettingsStore_MediaBufferMinutes_Default` | Défaut = 2 |
| `TestSettingsStore_MediaBufferMinutes_Legacy_Tolerance_Ignored` | Ancienne clé `media_tolerance_minutes` dans JSON → ignorée, défaut 2 appliqué |

---

### 5.3 `apps/go-api/internal/api/handlers/` — étendre `media_test.go`

> `mockMediaService` doit implémenter `ReassociateMedia`. Ajouter :
>
> ```go
> reassocResult *domain.ReassociateResult
> reassocErr    error
> func (m *mockMediaService) ReassociateMedia(_ context.Context, _ domain.ReassociateRequest) (*domain.ReassociateResult, error) {
>     return m.reassocResult, m.reassocErr
> }
> ```

| Fonction de test | Ce qu'elle vérifie |
|---|---|
| `TestReassociateHandler_OK` | POST `/reassociate` → 200 + `ReassociateResult` JSON |
| `TestReassociateHandler_PlayerNotFound` | `newUpload` retourne erreur → 404 |
| `TestReassociateHandler_ServiceError` | Service retourne erreur → 500 |
| `TestReassociateHandler_DefaultBuffer` | Corps vide → `BufferMin=0` passé au service |
| `TestReassociateHandler_CustomBuffer` | Corps `{"buffer_min": 5}` → `BufferMin=5` |
| `TestReassociateHandler_InvalidBody_Ignored` | Corps JSON invalide → requête traitée (body optionnel) |
| `TestReassociateHandler_BumpsMediaFeedVersion` | Succès → `mediaFeedVersion` incrémenté |
| `TestReassociateHandler_NoUploadFactory` | `h.newUpload == nil` → 501 |
| `TestParseUploadedFiles_WithCaptureTimes` | Champ `capture_times` JSON → `CaptureTimeUnix` remplis |
| `TestParseUploadedFiles_InvalidCaptureTimes_Ignored` | JSON invalide → pas d'erreur, `CaptureTimeUnix` nil |
| `TestParseUploadedFiles_CaptureTimes_LengthMismatch` | Plus de timestamps que de fichiers → extras ignorés, pas de panic |
| `TestParseUploadedFiles_CaptureTimes_ZeroValue` | `capture_times=[0]` → `CaptureTimeUnix` laissé nil |

---

## 6. Vérification manuelle post-implémentation

1. **Test CET (hiver)** : uploader `Halo Infinite 2024.01.20 - 20.15.30.01.mp4`.
   Vérifier dans `media_files` : `capture_start_utc = '2024-01-20 19:15:30Z'` (Paris − 1 h).
   Vérifier que le match `start_time = '2024-01-20 20:15:30'` est associé.

2. **Test CEST (été)** : uploader `Halo Infinite 2024.07.15 - 21.30.00.01.mp4`.
   Vérifier : `capture_start_utc = '2024-07-15 19:30:00Z'` (Paris − 2 h). Match → associé.

3. **Test OBS** : uploader `OBS 2024-07-15 21-30-00.mp4` (pas de pattern Xbox).
   `file.lastModified = 1721079000` (21h30 Paris été).
   Vérifier : `capture_start_utc = '2024-07-15 19:30:00Z'`. Match → associé.

4. **Test ré-association** : `POST /api/v1/players/jgtm/media/reassociate`.
   Réponse : `backup_table` non vide, `deleted_associations >= 0`, `new_associations >= 0`.
   Vérifier la table backup dans la DB via DuckDB CLI :
   `SELECT COUNT(*) FROM media_match_associations_bak_...`.

5. **`go test ./apps/go-api/...`** → tous les tests passent, 0 régression.

---

## 7. Décisions et périmètre

| Décision | Justification |
|---|---|
| Buffer 2 min (configurable) pour toutes les associations | Un match dure 2–12 min : le buffer couvre uniquement l'imprécision du timestamp (scoreboard post-match, décalage horodatage). La fenêtre réelle `[start_time, end_time]` couvre la durée du match. |
| Suppression de la tolérance 20 min Python | Cette valeur compensait l'imprécision du mtime côté local ; avec `file.lastModified` et le filename parser la précision est < 1 min, un buffer 20 min créerait des faux positifs entre matchs consécutifs. |
| `end_time` utilisé dans le JOIN | Disponible dans `match_registry`. Rend le SQL sémantiquement correct : « la capture est-elle pendant ce match ? » plutôt que « est-elle proche du début du match ? » |
| Filename parser interprète la datetime en timezone locale | Xbox encode l'heure locale affichée à l'écran lors de la capture |
| `SanitizeMediaTimezone` exportée dans `ops/` | Le service en a besoin dans `NewMediaService` pour sanitiser à l'injection. Exporter évite la duplication sans créer de package `tzutil` prématurément. |
| Sanitisation dans `NewMediaService` (pas dans les méthodes) | `s.timezone` est toujours propre après le constructeur. Les méthodes `ReassociateMedia`, `UploadMedia` font confiance au champ — pas de re-validation. |
| `ServiceRegistry.timezone string` | `ServiceRegistry` ne stocke pas `cfg` : ajouter un champ `timezone` initialisé depuis `cfg.UserTimezone` dans `NewServiceRegistry`. |
| `media_tolerance_minutes` → `media_buffer_minutes` (défaut 2) | Sémantique changée : l'ancienne valeur mesurait la distance depuis `start_time` ; la nouvelle mesure le buffer autour de `[start_time, end_time]`. Un défaut de 10 n'a plus de sens. Rupture de rétrocompat JSON documentée. |
| Pas d'EXIF in-memory | Xbox et OBS couverts par filename + `file.lastModified` ; EXIF ajoute une dépendance cgo et ne couvre pas OBS |
| Backup nommé par timestamp Unix | Reproductible, lisible, collision quasi-impossible |
| `CaptureTimes` indexé par basename (pas chemin complet) | L'original name du form et le chemin disque ont le même basename après `safeDestPath` |

**Hors périmètre :** nettoyage automatique des tables backup, EXIF in-memory,
extraction de `sanitizeTimezone` en package partagé, endpoint de listage des tables backup.
