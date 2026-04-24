# Plan — BlobStoragePathPrefix pour les chunks film

## Contexte

Le commit Grunt `1e9efc4` (19 avr. 2026) ajoute un champ `BlobStoragePathPrefix?: string` sur
l'interface `AssetBase`, ce qui inclut `FilmAsset`. Ce champ indique un préfixe de blob storage
à concaténer avec les chemins relatifs présents dans `Files.FileRelativePaths` (et potentiellement
dans d'autres champs de `CustomData`).

### Hypothèse actuelle dans notre code

`filmChunk.ChunkURL` est toujours une URL absolue pre-signed Azure Blob. `downloadBlob` passe
directement cette URL sans auth header (pre-signed = auto-signée).

```go
// halo_client.go
type filmChunk struct {
    ChunkURL string `json:"ChunkUrl"`  // URL absolue aujourd'hui
    ...
}

data, err := c.downloadBlob(ctx, chunk.ChunkURL)
```

### Risque

Si Halo migre le format du manifest film vers des chemins relatifs + `BlobStoragePathPrefix`
(pattern déjà observé dans d'autres assets UGC), `ChunkURL` deviendrait un chemin relatif
(ex. `chunks/match-uuid-xxxx/chunk-0.bin`) et `downloadBlob` échouerait silencieusement
avec une erreur URL malformée ou un 404 sans message clair.

Ce n'est **pas un bug aujourd'hui** — c'est une hypothèse fragile non défendue par le code.

---

## Plan d'implémentation

### Phase 1 — Enrichir le struct + détecter le format (défensif)

**Fichier** : `apps/go-api/internal/sync/halo_client.go`

1. Ajouter `BlobStoragePathPrefix` dans `filmManifest` au niveau `CustomData` :

```go
type filmManifest struct {
    CustomData struct {
        BlobStoragePathPrefix string      `json:"BlobStoragePathPrefix"`
        FilmChunks            []filmChunk `json:"FilmChunks"`
    } `json:"CustomData"`
}
```

2. Ajouter une fonction helper `resolveChunkURL(prefix, rawURL string) (string, error)` :

```go
// resolveChunkURL construit l'URL finale d'un chunk.
// Si rawURL est déjà absolu, le prefix est ignoré.
// Si rawURL est relatif, il est concaténé au prefix.
// Retourne une erreur si rawURL est relatif et prefix est vide.
func resolveChunkURL(prefix, rawURL string) (string, error) {
    if strings.HasPrefix(rawURL, "http://") || strings.HasPrefix(rawURL, "https://") {
        return rawURL, nil
    }
    if strings.TrimSpace(prefix) == "" {
        return "", fmt.Errorf("resolveChunkURL: ChunkUrl %q est relatif mais BlobStoragePathPrefix est absent", rawURL)
    }
    return strings.TrimRight(prefix, "/") + "/" + strings.TrimLeft(rawURL, "/"), nil
}
```

3. Remplacer l'appel direct dans `GetMatchFilm` :

```go
for i, chunk := range chunks {
    resolvedURL, err := resolveChunkURL(manifest.CustomData.BlobStoragePathPrefix, chunk.ChunkURL)
    if err != nil {
        return nil, false, fmt.Errorf("GetMatchFilm chunk %d(%s): %w", i, matchID, err)
    }
    data, err := c.downloadBlob(ctx, resolvedURL)
    ...
}
```

### Phase 2 — Tests

**Fichier** : `apps/go-api/internal/sync/halo_client_extra_test.go` (ou nouveau `film_blob_test.go`)

#### Tests unitaires `resolveChunkURL`

| Test | Cas | Vérification |
|------|-----|-------------|
| `TestResolveChunkURL_AbsoluteURL_NoPrefix` | URL absolue, prefix vide | URL retournée telle quelle |
| `TestResolveChunkURL_AbsoluteURL_WithPrefix` | URL absolue, prefix non vide | URL retournée telle quelle — prefix ignoré |
| `TestResolveChunkURL_RelativeWithPrefix` | Chemin relatif + prefix → URL absolue | Concaténation correcte |
| `TestResolveChunkURL_RelativeWithPrefix_TrailingSlash` | Prefix avec slash final + path avec slash initial | Pas de double slash dans l'URL résultante |
| `TestResolveChunkURL_RelativeWithoutPrefix` | Chemin relatif, prefix vide | Erreur explicite avec le rawURL dans le message |
| `TestResolveChunkURL_EmptyURL` | `rawURL` vide | Erreur explicite |

#### Tests d'intégration `GetMatchFilm`

| Test | Cas | Ce que ça protège |
|------|-----|------------------|
| `TestGetMatchFilm_AbsoluteURLs_NoPrefix` | Manifest sans prefix, ChunkUrl absolues **(comportement actuel)** | **Non-régression** — chemin nominal inchangé après la PR |
| `TestGetMatchFilm_UsesBlobPrefix_SingleChunk` | Prefix + 1 ChunkUrl relatif | Nouveau format pris en charge |
| `TestGetMatchFilm_UsesBlobPrefix_MultiChunk` | Prefix + N ChunkUrl relatifs | Tous les chunks résolus, pas seulement le premier |
| `TestGetMatchFilm_AbsoluteURLs_PrefixPresent` | Prefix non vide + ChunkUrl absolues | Prefix ignoré sur les absolus (format mixte) |
| `TestGetMatchFilm_RelativeURL_NoPrefixInManifest` | Pas de prefix, ChunkUrl relatif | Erreur remontée avec match_id + index dans le message |

### Phase 3 — Logging

Le `slog.Warn` ne doit **pas** être optionnel — c'est le seul signal d'alerte opérationnel si
l'API bascule.

Deux points de log obligatoires :

**A. Log par chunk au moment de la résolution** (dans la boucle, remplace le log unique pre-boucle) :

```go
for i, chunk := range chunks {
    resolvedURL, err := resolveChunkURL(manifest.CustomData.BlobStoragePathPrefix, chunk.ChunkURL)
    if err != nil {
        // Log explicite avant de retourner l'erreur — visible dans les traces sync.
        slog.ErrorContext(ctx, "GetMatchFilm: impossible de résoudre l'URL du chunk",
            "match_id", matchID,
            "chunk_index", i,
            "raw_url", chunk.ChunkURL,
            "prefix", manifest.CustomData.BlobStoragePathPrefix,
            "err", err,
        )
        return nil, false, fmt.Errorf("GetMatchFilm chunk %d(%s): %w", i, matchID, err)
    }
    // Log warn si résolution via prefix (format non nominal).
    if resolvedURL != chunk.ChunkURL {
        slog.WarnContext(ctx, "GetMatchFilm: chunk résolu via BlobStoragePathPrefix",
            "match_id", matchID,
            "chunk_index", i,
            "resolved_url", resolvedURL,
        )
    }
    data, err := c.downloadBlob(ctx, resolvedURL)
    ...
}
```

**Pourquoi par chunk et pas pre-boucle :** un manifest en format mixte (certains chunks absolus,
d'autres relatifs) serait invisible avec un log unique. Le warn par chunk garantit la traçabilité
exacte.

**B. Test de vérification du logging** : utiliser `slog/slogtest` ou un handler de capture pour
vérifier que :
- Le `slog.Warn` est émis exactement une fois par chunk relatif résolu
- Le `slog.Error` est émis si `resolveChunkURL` échoue
- Aucun log n'est émis sur le chemin nominal (URLs absolues, pas de prefix)

### Phase 4 — Non-régression pipeline weapon kills

> **Objectif** : garantir que la résolution d'URL (`resolveChunkURL`) ne casse pas
> le pipeline complet film → `weapon_kills`.

**Clarification architecture** (important pour cibler les bons tests) :

- Les `highlight_events` ne sont **pas** parsés depuis le film binaire en Go.
- **Le backfill events n'est pas du tout implémenté en Go** : `warnUnimplemented` dans
  `internal/api/handlers/backfill.go` (l.155) le confirme — quand `scope.Events` est actif,
  Go détecte les matchs avec `events_loaded = false` mais n'exécute rien et émet un warning.
- Conséquence directe : `getKillsForPlayer` lit `highlight_events WHERE event_type='Killed'`
  pour alimenter `CorrelateKillsGlobal`. Si la table est vide, zero attributions → `weapon_kills` vide.
- **C'est un gap fonctionnel** : sans implémentation du backfill events, le pipeline weapon kills
  dépend de données historiques Python ou est inopérant sur les nouveaux matchs.
- Le film binaire nourrit **uniquement** `weapon_kills` via `BackfillWeaponKillsForMatch`.

**Tests déjà présents (non-régression micro, `analysis/`)** — à NE PAS redoubler :

| Fichier | Couverture |
|---------|-----------|
| `weapon_scanner_test.go` | `FindFramePositions`, `ScanFormulaA`, `ScanFireEventsB5`, `ComputeConfidence`, `CountKillsByWeapon` |
| `weapon_parser_test.go` | `FindChunkAtTime`, `BuildWeaponTimelines`, `ScanFireEventsAll`, `ScanFireEvents` |
| `weapon_correlation_test.go` | `CorrelateKillsGlobal` (melee, grenade, fire event, fallback formulaA/timeline), `AttributionFromEvent` |
| `weapon_reconciliation_test.go` | `CountConfidentAttributions`, `ComputeSurplus`, `FindBestSurplus`, `AssignSentinels`, `ReconcileAPIAggregates` |
| `backfill_weapons_test.go` | `getKillsForPlayer`, `getXuidToPI`, `InsertWeaponKills`, `MarkWeaponKillsDone`, `attributionsToRows` |

Ces tests couvrent chaque maillon de la chaîne sur des entrées triviales. Ce qui manque : un test
bout-en-bout qui passe de vrais octets binaires jusqu'à l'insertion en DB.

**Tests à ajouter** — fichier : `apps/go-api/internal/sync/backfill_weapons_test.go`
(tag `//go:build integration`)

| Test | Scénario | Ce que ça protège |
|------|----------|------------------|
| `TestBackfillWeaponKillsForMatch_NoFilm` | `GetMatchFilm` retourne `false` (film absent) | Retour `(false, nil)`, aucun write en DB |
| `TestBackfillWeaponKillsForMatch_AbsoluteURLs` | Mock retourne des chunks binaires de fixture (URLs absolues, sans prefix) | **Non-régression chemin actuel** — même résultat après introduction de `resolveChunkURL` |
| `TestBackfillWeaponKillsForMatch_WithBlobPrefix` | Mock retourne les mêmes chunks via prefix + URL relative | Même résultat en DB que le cas AbsoluteURLs |
| `TestBackfillWeaponKillsForMatch_RelativeURL_NoPrefix` | Mock retourne un chunk relatif sans prefix dans le manifest | Erreur remontée, `weapon_kills` non modifiée |

**Structure du test de non-régression (AbsoluteURLs)** :

```go
// TestBackfillWeaponKillsForMatch_AbsoluteURLs est le test de non-régression
// central : il vérifie que l'introduction de resolveChunkURL ne change pas
// le comportement observé sur les URLs absolues (chemin 100% actuel).
func TestBackfillWeaponKillsForMatch_AbsoluteURLs(t *testing.T) {
    db := openWeaponDB(t)
    // Seeder highlight_events avec 2 kills pour "xuid1"
    db.Exec(`INSERT INTO highlight_events VALUES ('m1', 'xuid1', 'Killed', 5000)`)
    db.Exec(`INSERT INTO highlight_events VALUES ('m1', 'xuid1', 'Killed', 10000)`)
    db.Exec(`INSERT INTO match_registry (match_id) VALUES ('m1')`)

    // Mock retournant des chunks binaires de fixture (octets FormulaA connus)
    mock := &mockHaloClient{
        filmChunks: map[int]filmChunkData{
            0: {Data: buildTestChunkWithKills(5000, 10000), Offset: 0, Duration: 15000},
        },
        filmPresent: true,
    }

    found, err := BackfillWeaponKillsForMatch(context.Background(), mock, db, "m1", "xuid1")
    if err != nil {
        t.Fatalf("erreur inattendue: %v", err)
    }
    if !found {
        t.Fatal("expected film found")
    }

    // Vérifier que weapon_kills a bien été alimentée (non-régression)
    var count int
    db.QueryRow("SELECT COUNT(*) FROM weapon_kills WHERE match_id='m1' AND xuid='xuid1'").Scan(&count)
    if count == 0 {
        t.Fatal("weapon_kills non alimentée après BackfillWeaponKillsForMatch")
    }
}
```

> **Note sur les fixtures binaires** : `buildTestChunkWithKills()` est un helper de test qui
> construit des octets avec le pattern `FormulaAPattern = []byte{0x20, 0x00, 0x02}` et
> `FrameMarker = []byte{0xA0, 0x7B, 0x42}` connus de `weapon_scanner.go`. Ce helper
> appartient au package `sync_test` et dépend de constantes exportées depuis `analysis`.

---

## Périmètre

- **In scope** : `filmManifest` / `GetMatchFilm` / `downloadBlob` uniquement.
- **Out of scope** : les autres assets UGC (maps, game variants) — leur téléchargement ne passe
  pas par `downloadBlob` et ne souffre pas du même problème.
- **Aucune migration DB** — changement purement dans la couche client HTTP.
- **Aucun changement d'interface** `HaloClient` — `GetMatchFilm` garde la même signature.

---

## Effort estimé

| Phase | Complexité |
|-------|-----------|
| Phase 1 — struct + resolveChunkURL + GetMatchFilm | ~30 lignes |
| Phase 2 — 4 tests unitaires | ~80 lignes |
| Phase 3 — log warn | ~5 lignes |

Branche suggérée : `fix/film-blob-storage-prefix`
