# Plan — BlobStoragePathPrefix pour les chunks film

> **Mise à jour 24 avr. 2026** — Les phases 1 et 3 ont été implémentées (avec une approche
> différente du plan initial). Le gap highlight_events est également comblé. Ce document
> décrit l'état réel du code et les tests restant à écrire.

## Contexte

Le commit Grunt `1e9efc4` (19 avr. 2026) ajoute `BlobStoragePathPrefix?: string` sur `AssetBase`.
Initialement identifié comme un risque futur, ce pattern est **déjà implémenté** dans notre code.

---

## État de l'implémentation (code réel)

### Structure manifest — différente du plan initial

`BlobStoragePathPrefix` est au niveau **racine** du manifest (pas dans `CustomData`).
Les chunks utilisent `FileRelativePath` (toujours relatif — plus de gestion URL absolue/relative) :

```go
// halo_client.go — structure réelle
type filmManifest struct {
    BlobStoragePathPrefix string `json:"BlobStoragePathPrefix"`
    CustomData            struct {
        FilmMajorVersion int         `json:"FilmMajorVersion"`
        Chunks           []filmChunk `json:"Chunks"`
    } `json:"CustomData"`
}

type filmChunk struct {
    Index            int    `json:"Index"`
    ChunkType        int    `json:"ChunkType"`
    FileRelativePath string `json:"FileRelativePath"`
    // ...
}
```

### `buildChunkURL` — remplace `resolveChunkURL` du plan

Plus simple : `FileRelativePath` est toujours relatif, pas besoin de distinguer absolu/relatif.

```go
func buildChunkURL(blobPrefix, fileRelativePath string) string {
    name := strings.TrimLeft(fileRelativePath, "/")
    if name == "" {
        return blobPrefix
    }
    if blobPrefix != "" && blobPrefix[len(blobPrefix)-1] != '/' {
        return blobPrefix + "/" + name
    }
    return blobPrefix + name
}
```

### `fetchFilmManifest` — factorisation partagée

Factorisée et utilisée par les deux méthodes :
- `GetMatchFilm` → chunks `ChunkType=2` (weapon scanner)
- `GetHighlightEventsChunk` → chunk `ChunkType=3` (highlight events)

### Pipeline highlight events — implémenté en Go ✅

**Anciennement un gap fonctionnel, maintenant comblé :**

```
GetHighlightEventsChunk (ChunkType=3)
  → analysis.ParseHighlightEvents (zlib decompress + scan binaire)
  → InsertHighlightEvents (INSERT OR IGNORE, idempotent)
  → InsertKillerVictimPairsFromEvents (via analysis.ComputeKillerVictimPairs)
  → MarkEventsLoaded (events_loaded = TRUE, MBitEvents dans backfill_completed)
```

Câblé dans `processHighlightEvents` → appelé depuis `processMatch` via `opts.WithHighlightEvents`.

---

## Travail restant — Tests

### Tests unitaires `buildChunkURL`

**Fichier** : `apps/go-api/internal/sync/halo_client_extra_test.go`

| Test | Cas | Vérification |
|------|-----|-------------|
| `TestBuildChunkURL_EmptyRelativePath` | Prefix seul, path vide | Retourne le prefix tel quel |
| `TestBuildChunkURL_BasicConcatenation` | `prefix="https://blob.example.com"`, `path="chunks/c0.bin"` | `"https://blob.example.com/chunks/c0.bin"` |
| `TestBuildChunkURL_TrailingSlashOnPrefix` | Prefix avec `/` final + path avec `/` initial | Pas de double slash |
| `TestBuildChunkURL_PathWithLeadingSlash` | `path="/chunks/c0.bin"` | Slash initial supprimé |
| `TestBuildChunkURL_EmptyPrefix` | `prefix=""`, `path="chunks/c0.bin"` | Retourne `"chunks/c0.bin"` (erreur détectée plus tard par downloadBlob) |

### Tests d'intégration `GetMatchFilm`

| Test | Cas | Ce que ça protège |
|------|-----|------------------|
| `TestGetMatchFilm_BasicPrefix` | Manifest avec prefix + 1 chunk relatif **(chemin nominal)** | Téléchargement correct via `buildChunkURL` |
| `TestGetMatchFilm_MultiChunk` | Prefix + N chunks | Tous les chunks résolus |
| `TestGetMatchFilm_FilmAbsent` | 404 sur le manifest | Retour `(nil, false, nil)` |
| `TestGetMatchFilm_DownloadFails` | Manifest OK, `downloadBlob` échoue | Erreur remontée avec chunk index + match_id |

### Tests d'intégration `GetHighlightEventsChunk`

| Test | Cas | Ce que ça protège |
|------|-----|------------------|
| `TestGetHighlightEventsChunk_Found` | Manifest avec chunk ChunkType=3 | Retourne les données + filmMajorVersion |
| `TestGetHighlightEventsChunk_NoChunk` | Manifest sans ChunkType=3 | Retour `(nil, 0, false, nil)` |
| `TestGetHighlightEventsChunk_FilmAbsent` | 404 | Retour `(nil, 0, false, nil)` |

### Non-régression pipeline weapon kills

> **Objectif** : garantir que `buildChunkURL` ne casse pas le pipeline complet film → `weapon_kills`.
> Depuis l'implémentation des highlight events en Go, `getKillsForPlayer` lira des données
> réelles pour les nouveaux matchs — la dépendance est maintenant entièrement gérée en Go.

**Tests déjà présents — à NE PAS redoubler :**

| Fichier | Couverture |
|---------|-----------|
| `weapon_scanner_test.go` | `FindFramePositions`, `ScanFormulaA`, `ScanFireEventsB5`, `ComputeConfidence`, `CountKillsByWeapon` |
| `weapon_parser_test.go` | `FindChunkAtTime`, `BuildWeaponTimelines`, `ScanFireEventsAll`, `ScanFireEvents` |
| `weapon_correlation_test.go` | `CorrelateKillsGlobal` (melee, grenade, fire event, fallback formulaA/timeline), `AttributionFromEvent` |
| `weapon_reconciliation_test.go` | `CountConfidentAttributions`, `ComputeSurplus`, `FindBestSurplus`, `AssignSentinels`, `ReconcileAPIAggregates` |
| `backfill_weapons_test.go` | `getKillsForPlayer`, `getXuidToPI`, `InsertWeaponKills`, `MarkWeaponKillsDone`, `attributionsToRows` |
| `highlight_events_test.go` | `InsertHighlightEvents` (empty/insert/idempotent), `InsertKillerVictimPairsFromEvents`, `MarkEventsLoaded` |

**Tests à ajouter** — fichier : `apps/go-api/internal/sync/backfill_weapons_test.go`

| Test | Scénario | Ce que ça protège |
|------|----------|------------------|
| `TestBackfillWeaponKillsForMatch_NoFilm` | `GetMatchFilm` retourne `false` | Retour `(false, nil)`, aucun write |
| `TestBackfillWeaponKillsForMatch_WithHighlightEvents` | highlight_events seedés en DB + chunks binaires de fixture | **Non-régression** : pipeline complet fonctionne quand `highlight_events` est alimenté par Go |
| `TestBackfillWeaponKillsForMatch_EmptyHighlightEvents` | highlight_events vide pour le match | `weapon_kills` vide mais pas d'erreur (comportement intentionnel) |

> **Note** : `buildTestChunkWithKills()` est un helper qui construit des octets avec
> `FormulaAPattern = []byte{0x20, 0x00, 0x02}` et `FrameMarker = []byte{0xA0, 0x7B, 0x42}`
> (constantes de `weapon_scanner.go`).

---

## Périmètre

- **In scope** : `filmManifest` / `GetMatchFilm` / `GetHighlightEventsChunk` / `buildChunkURL`.
- **Out of scope** : autres assets UGC (maps, game variants) — pas concernés par `buildChunkURL`.
- **Aucune migration DB** — changement purement dans la couche client HTTP.

---

## Branche

`feat/v7-assets-abstraction` (implémentation déjà mergée sur cette branche)
