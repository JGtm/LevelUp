# Diagnostic — Verrou `metadata.duckdb` au redémarrage Air (hot-reload, Windows)

**Date du diagnostic** : 2026-05-21
**Sévérité** : Faible côté impact (le serveur démarre), Élevée côté lisibilité (les logs ressemblent à un crash)
**Statut** : Diagnostic complet, correctif non appliqué (attente go/no-go)
**Auteur** : analyse menée à partir des logs `19:49:57 [ERROR] duckdb: ouverture DB échouée ... metadata verrouillée, nouvelle tentative...`

---

## TL;DR

Au redémarrage du serveur Go via Air (hot-reload), on observe une rafale d'environ 6 lignes `ERROR` + 6 lignes `WARN` sur `metadata.duckdb` avant que le serveur ne démarre normalement. **Aucune écriture n'est refusée** : la boucle de retry [`cmd/server/main.go:338-349`](../apps/go-api/cmd/server/main.go#L338-L349) (12 × 500 ms = 6 s) finit par réussir et le serveur sert le trafic. Mais le double log (`slog.Error` interne dans `openCachedDB` + `slog.Warn` du caller) donne l'impression que l'application explose.

**Cause racine** : l'ancien process tué par Air conserve le handle Windows sur `metadata.duckdb` quelques secondes après son exit. Pendant cette fenêtre, le nouveau process échoue à ouvrir le fichier (conflit driver DuckDB). Ce n'est pas un problème intra-process — donc le `SharedDBProvider` (ADR 0016) ne s'applique pas ici.

**Pourquoi ça arrive** :
1. Soit Air SIGKILL le binaire avant que le shutdown gracieux ait fermé toutes les conn DuckDB (timeout dépassé)
2. Soit un caller de `OpenReadWriteShared(metaPath)` ne fait pas son `Close()` apparié → refCount du pool cached reste > 0 → le `sql.DB` n'est pas réellement fermé au shutdown → handle Windows tenu jusqu'à exit du process

**Fix solide en 4 étapes** : diagnostic shutdown → audit callers → vérif Air → nettoyage logs. Détails plus bas.

---

## Symptôme observé

```
19:49:57 [INFO]  shared_matches_v2: mode sharedprovider (B-swap actif, pool inscrit via Subscribe)
19:49:57 [ERROR] duckdb: ouverture DB échouée path=...metadata.duckdb op=OpenReadWriteShared dsn=...
19:49:57 [WARN]  metadata verrouillée, nouvelle tentative... attempt=1 max=12 err="duckdb.OpenReadWriteShared(...)"
19:49:57 [ERROR] duckdb: ouverture DB échouée path=...metadata.duckdb ...
19:49:57 [WARN]  metadata verrouillée, nouvelle tentative... attempt=2 max=12 ...
...
19:49:59 [WARN]  metadata verrouillée, nouvelle tentative... attempt=6 max=12 ...
[le serveur démarre normalement après]
```

À lire correctement :
- Ce sont **des tentatives d'ouverture**, pas des refus définitifs.
- Le serveur démarre (sinon `os.Exit(1)` aurait été déclenché à `attempt == 11`).
- Le doublet `ERROR` + `WARN` par tentative vient de deux logs distincts qui se superposent.

---

## Contexte architectural

### Pool DuckDB partagé via refCount

`OpenReadWriteShared(path)` ([`apps/go-api/internal/platform/duckdb/db.go:93-107`](../apps/go-api/internal/platform/duckdb/db.go#L93-L107)) ouvre une connexion DuckDB RW mais **passe par un cache global** indexé par path :

```go
var (
    openDBsMu sync.Mutex
    openDBs   = map[string]*cachedDB{}
)

type cachedDB struct {
    db       *DB
    refCount int
}
```

- Le premier caller ouvre réellement le fichier (`openSQLDBFor`) et stocke l'entrée avec `refCount=1`.
- Les callers suivants sur le même path réutilisent le pool et incrémentent `refCount`.
- `Close()` ([`db.go:316-338`](../apps/go-api/internal/platform/duckdb/db.go#L316-L338)) décrémente `refCount`. Quand il tombe à 0, `sql.DB.Close()` est appelé et le handle OS libéré.

**Conséquence majeure** : au runtime, aucun caller métadata ne se bat pour le file lock. Ils tapent tous le même `sql.DB` cached. Le verrou de fichier n'est en jeu qu'à **l'ouverture initiale** et au **shutdown**.

### Périmètre du `SharedDBProvider` (ADR 0016)

Le Provider B-swap sert à coordonner RO↔RW **dans le même process** entre le sync engine (writer) et les HTTP handlers (readers) sur `shared_matches_v2.duckdb`. Il ne s'applique pas à `metadata.duckdb` parce que :
- `metadata.duckdb` n'a pas de schéma RO/RW concurrent — c'est le même pool RW pour tous les callers.
- Le conflit ici est **inter-process** (ancien Air ↔ nouveau Air), pas intra-process. Aucun lock applicatif Go ne peut résoudre un handle tenu par un autre PID.

### Retry actuel au boot

[`apps/go-api/cmd/server/main.go:335-349`](../apps/go-api/cmd/server/main.go#L335-L349) :

```go
const metaOpenAttempts = 12
const metaOpenDelay = 500 * time.Millisecond
var metaDB *duckdb.DB
for attempt := range metaOpenAttempts {
    metaDB, err = duckdb.OpenReadWriteShared(metaPath)
    if err == nil {
        break
    }
    if attempt == metaOpenAttempts-1 {
        slog.Error("ouverture metadata échouée", "attempts", metaOpenAttempts, "err", err)
        os.Exit(1)
    }
    slog.Warn("metadata verrouillée, nouvelle tentative...", "attempt", attempt+1, "max", metaOpenAttempts, "err", err)
    time.Sleep(metaOpenDelay)
}
```

Cette boucle EST la temporisation demandée par la promesse « ne jamais refuser une écriture, temporiser au pire ». Elle fonctionne. Le seul problème est cosmétique : `OpenReadWriteShared` loggue `slog.Error("duckdb: ouverture DB échouée", ...)` ([`db.go:144`](../apps/go-api/internal/platform/duckdb/db.go#L144)) à chaque tentative, **avant** que le caller décide quoi faire. Donc 11 lignes ERROR pour 11 retries qui n'en sont pas vraiment.

### Autres callers métadata avec leurs propres retries

L'audit montre que la temporisation est **dupliquée et incohérente** dans le code :

| Lieu | Retry | Comportement si échec |
|---|---|---|
| `cmd/server/main.go:338-349` | 12 × 500 ms (6 s) | `os.Exit(1)` |
| `internal/api/server.go:281-318` (AssetMetadata) | 3 × 500 ms (1.5 s) | `assetMetaHandler = nil`, /assets/{title}/maps renvoie `[]` |
| `internal/api/server.go:122-141` (RankCatalog) | aucun | `hiRanks = nil` puis WARN |
| `internal/api/server.go:253-271` (Seasons) | aucun | WARN |
| `internal/api/server.go:285-294` (AssetMeta in-memory) | déjà couvert ci-dessus | — |
| `internal/api/server.go:386-395` (Catalog) | aucun | WARN, handlers catalog non enregistrés |

→ La même connaissance (« metadata peut être verrouillée transitoirement au boot ») est implémentée 6 fois avec 4 stratégies différentes. **Si le pool cached est déjà ouvert par `main.go` au moment où ces callers s'exécutent**, ils profitent du cache et ne refight pas le lock. Mais c'est une fragilité d'ordre d'init — si demain on inverse l'ordre, certaines callers retombent sur le verrou avec zéro retry.

---

## Cause racine

### Hypothèse 1 — Air SIGKILL trop précoce

Le `.air.toml` configure ([`apps/go-api/.air.toml:33-41`](../apps/go-api/.air.toml#L33-L41)) :

```toml
kill_delay = "1000ms"
send_interrupt = true   # envoie SIGTERM en premier
stop_timeout = 20000    # SIGKILL après 20s si le binaire ne s'arrête pas
```

Le shutdown gracieux côté Go ([`cmd/server/main.go:533-559`](../apps/go-api/cmd/server/main.go#L533-L559)) fait :
1. `srv.Shutdown(shutdownCtx)` — fin du HTTP server (timeout configuré).
2. `schedulerWG.Wait()` avec timeout 3 s — attend que le scheduler ait fini ses `RunOnce`.
3. `duckdb.CloseAll()` — ferme tous les pools cached.
4. `closeShared()` — ferme le Provider B-swap shared.
5. `metaDB.Close()` — décrément final sur metadata.

Si l'une de ces étapes dépasse les 20 s cumulées, Air envoie SIGKILL et **aucun `defer Close()` n'est exécuté**. Le process meurt avec les handles DuckDB encore mappés en mémoire. Windows mettra plusieurs secondes à libérer ces handles orphelins → verrou au boot suivant.

À mesurer : durée réelle du shutdown gracieux. Si > 15 s, il faut soit augmenter `stop_timeout`, soit identifier le step qui traîne.

### Hypothèse 2 — Leak de refCount

Si un caller de `OpenReadWriteShared(metaPath)` oublie son `Close()` apparié (panic non rattrapé, return early, etc.), le `refCount` du pool cached reste > 0 au shutdown. Quand `metaDB.Close()` final décrémente, on tombe à 1 (par exemple) au lieu de 0 → le test `cached.refCount > 1` ([`db.go:329`](../apps/go-api/internal/platform/duckdb/db.go#L329)) est vrai → `sql.DB.Close()` n'est JAMAIS appelé → les conns ne sont pas drainées → handles tenus jusqu'à exit du process → Windows en garde une trace pour quelques secondes.

À mesurer : à la fin du shutdown, dumper `openDBs` et vérifier que `refCount == 0` (ou que la clé a été retirée du map) pour `metaPath`.

### Hypothèse 3 — Air envoie SIGKILL d'emblée sur Windows

Sur Windows, `send_interrupt = true` n'a pas la même sémantique que sur Unix (SIGTERM n'existe pas en natif). Air doit utiliser `taskkill /T /F` ou `Process.Kill()` selon la version. Possible qu'aucun signal gracieux ne soit reçu par le binaire Go. À vérifier dans le code Air upstream pour la version utilisée.

---

## Plan d'action solide (priorisé)

### Étape 1 — Instrumenter le shutdown (5 min, 5 lignes)

Ajouter dans [`cmd/server/main.go`](../apps/go-api/cmd/server/main.go) juste après `metaDB.Close()` à la ligne 559 :

```go
// Diagnostic temporaire : dumper l'état du pool cached après shutdown pour
// détecter un leak de refCount sur metadata.duckdb. Retirer une fois le
// fix validé.
if leaked := duckdb.DumpCachedLeaks(); len(leaked) > 0 {
    for k, refs := range leaked {
        slog.Warn("shutdown_db_leak", "key", k, "refCount", refs)
    }
}
```

Et exposer côté `internal/platform/duckdb/db.go` :

```go
// DumpCachedLeaks retourne les pools encore référencés (refCount > 0) après
// CloseAll. Pour diagnostic uniquement.
func DumpCachedLeaks() map[string]int {
    openDBsMu.Lock()
    defer openDBsMu.Unlock()
    out := make(map[string]int, len(openDBs))
    for k, c := range openDBs {
        out[k] = c.refCount
    }
    return out
}
```

Critère de succès : faire un `Ctrl+C` propre sur le serveur. Si aucun `shutdown_db_leak` n'apparaît, l'hypothèse 2 est invalidée → passer à l'étape 3 (Air). Si des leaks apparaissent, lister les paths concernés et passer à l'étape 2.

### Étape 2 — Audit statique des callers `OpenReadWriteShared(metaPath)`

Liste actuelle des callers à auditer (issue du `grep` du 2026-05-21) :

| Fichier:ligne | `Close()` apparié ? | Notes |
|---|---|---|
| `cmd/server/main.go:339` | OUI ([`:557`](../apps/go-api/cmd/server/main.go#L557)) | Caller principal |
| `internal/api/server.go:123` | OUI ([`:136`](../apps/go-api/internal/api/server.go#L136)) | Charge rank catalog, ferme |
| `internal/api/server.go:254` | À vérifier ([`:253-271`](../apps/go-api/internal/api/server.go#L253-L271)) | Seasons — handle gardé pour la vie du process ? |
| `internal/api/server.go:285` | OUI ([`:294`](../apps/go-api/internal/api/server.go#L294)) | AssetMetadata in-memory pattern |
| `internal/api/server.go:386` | **À VÉRIFIER** | Catalog — pas de Close visible dans le scope |
| `internal/api/prestige_setup.go:69` | À vérifier | |
| `internal/platform/lab/provider.go:59` | OUI ([`:63`](../apps/go-api/internal/platform/lab/provider.go#L63)) | `defer metaDB.Close()` |
| `internal/platform/lab/provider.go:181` | OUI ([`:185`](../apps/go-api/internal/platform/lab/provider.go#L185)) | |
| `internal/platform/duckdb/pool.go:217` | À vérifier — long-lived ? | Pool joueur, durée du process |
| `internal/sync/engine.go:229` | À vérifier | |
| `internal/sync/engine.go:252` | À vérifier | |
| `internal/sync/engine.go:547` | OUI (`defer`) | |
| `internal/sync/citations_backfill.go:54,157,261` | OUI (`defer`) | |
| `internal/service/openspartan_post_import_service.go:84` | À vérifier | |
| Plusieurs `cmd/*/main.go` | OUI (`defer`) | Tools standalone, pas d'impact runtime serveur |

Méthode :
1. Pour chaque ligne marquée « À vérifier », lire le bloc de code et tracer le `Close()` apparié sur tous les chemins (succès + erreur + early return).
2. Si pas de Close visible, vérifier que le handle est volontairement long-lived (cas du pool joueur, du prestige_setup) et qu'il a un Close au shutdown.
3. Pour chaque cas long-lived, vérifier qu'il est enregistré quelque part pour être fermé au shutdown, ou que `duckdb.CloseAll()` ([`db.go`](../apps/go-api/internal/platform/duckdb/db.go) — à localiser) couvre bien tous les pools.

Critère de succès : chaque `OpenReadWriteShared(metaPath)` a un `Close()` apparié OU est documenté long-lived avec preuve de fermeture au shutdown.

### Étape 3 — Mesurer la durée du shutdown Air

Ajouter au shutdown :

```go
shutdownStart := time.Now()
defer func() {
    slog.Info("shutdown_total_duration_ms", "ms", time.Since(shutdownStart).Milliseconds())
}()
```

Tester un cycle de hot-reload Air (modifier un fichier `.go` pour déclencher le rebuild) et lire le log de durée. Si > 15 s, identifier l'étape qui traîne :
- `srv.Shutdown` : timeout HTTP — combien ?
- `schedulerWG.Wait()` : 3 s max, OK
- `duckdb.CloseAll()` + `closeShared()` + `metaDB.Close()` : devrait être < 1 s

Si le shutdown total dépasse 15 s, soit augmenter `stop_timeout` dans `.air.toml` (passer à 30 s), soit shorten le timeout HTTP, soit isoler la goroutine qui traîne.

### Étape 4 — Nettoyer le bruit de log (uniquement après fix)

Une fois la cause racine traitée (étapes 1-3), le retry au boot devient un filet de sécurité ultra-rare. À ce moment-là :

**Option A** : Démoter `slog.Error("duckdb: ouverture DB échouée", ...)` ([`db.go:144`](../apps/go-api/internal/platform/duckdb/db.go#L144)) en `slog.Debug` puisque le caller a déjà toute l'info pour logger au bon niveau.

**Option B** : Pousser le retry dans `OpenReadWriteShared` lui-même (couche plateforme) avec détection du pattern d'erreur « file is held by another process » et retry interne silencieux. Avantage : la promesse « jamais refuser une écriture, temporiser au pire » est tenue au niveau plateforme, pas au caller. Désavantage : risque de masquer une vraie erreur si on retry trop large.

Recommandation : Option A en premier (zéro risque, gain de lisibilité immédiat). Option B seulement si on observe que d'autres callers métadata souffrent du même problème malgré le fix shutdown.

---

## Critères d'acceptation

Le fix est considéré complet quand :
- [ ] Un cycle de hot-reload Air (`touch` un `.go`) ne produit **aucun** WARN/ERROR sur metadata dans les logs du nouveau process.
- [ ] Un `Ctrl+C` sur le serveur n'émet aucun `shutdown_db_leak`.
- [ ] La durée totale du shutdown est < 15 s (mesurée via le log temporaire de l'étape 3).
- [ ] Les 6 callers métadata avec retry ad-hoc sont harmonisés (idéalement un seul point de temporisation centralisé, voir étape 4 option B).
- [ ] Une entrée `.ai/thought_log.md` documente le fix avec les chiffres avant/après.

---

## Pourquoi on n'a PAS choisi un Provider pour metadata

Tentation naturelle : « on a un Provider qui marche pour shared, on devrait en mettre un pour metadata ». Mais le Provider B-swap résout un problème différent :

| Problème | Solution |
|---|---|
| Conflit RO↔RW intra-process (sync engine vs HTTP handlers) | SharedDBProvider (ADR 0016) |
| Conflit cross-process (ancien process zombie ↔ nouveau process) | Attendre que l'OS libère le handle (retry) |
| Leak de refCount intra-process | Audit + Close apparié |

Ajouter un Provider pour metadata n'aiderait pas — il y a déjà un pool unique partagé par refCount, ce qui est l'équivalent fonctionnel du Provider pour ce cas d'usage. Le vrai fix est de garantir que le shutdown libère bien les handles.

---

## Annexes

### A — Commandes pour reproduire

```powershell
# Démarrer Air
cd apps/go-api
air

# Dans un autre terminal, modifier un fichier pour déclencher le rebuild
# (ex: ajouter un espace dans un .go)
echo " " >> internal/api/server.go

# Observer les logs du nouveau process — la rafale "metadata verrouillée"
# apparaît dans les 6 premières secondes
```

### B — Commandes d'audit

```powershell
# Lister tous les callers OpenReadWriteShared(metaPath)
# (Note : exécuté via le tool Grep, pas grep CLI)
# Pattern : OpenReadWriteShared.*MetadataDBPath|metaPath

# Pour chaque match, vérifier la présence d'un Close() apparié dans le
# même scope (lecture manuelle nécessaire — pas d'analyse statique fiable
# en Go pour ça sans go/analysis).
```

### C — Fichiers clés

- [`apps/go-api/internal/platform/duckdb/db.go`](../apps/go-api/internal/platform/duckdb/db.go) — `OpenReadWriteShared`, cache refCount, `Close`
- [`apps/go-api/cmd/server/main.go`](../apps/go-api/cmd/server/main.go) — boucle retry boot (338-349), shutdown (533-568)
- [`apps/go-api/internal/api/server.go`](../apps/go-api/internal/api/server.go) — 6 callers metadata avec stratégies divergentes
- [`apps/go-api/.air.toml`](../apps/go-api/.air.toml) — config hot-reload, `stop_timeout`
- [`docs/adr/0016-shared-db-provider-b-swap.md`](adr/0016-shared-db-provider-b-swap.md) — ADR Provider (pour comprendre le périmètre)

### D — Ce qu'il ne faut PAS faire

1. **Ajouter un Provider pour metadata** — overkill, ne résout pas le vrai problème.
2. **Augmenter le nombre de retries dans `main.go`** — masque le symptôme sans fix la cause.
3. **Supprimer le retry** — vrai filet de sécurité utile sur Windows + Air.
4. **`taskkill /F` manuellement avant chaque restart** — workaround utilisateur, pas une solution.
5. **Migrer vers un autre moteur de hot-reload** — Air n'est pas en cause si shutdown gracieux n'est pas exécuté ; le problème viendrait à nouveau avec n'importe quel outil qui SIGKILL.

### E — Historique du problème

Le fichier `.air.toml:37-41` documente déjà la conscience du problème :

```
# Arrêt forcé après ce délai (ms).
# Doit être >= shutdown gracieux Go (srv.Shutdown 15s + scheduler/watcher
# Wait ~3s = ~18s) sinon air SIGKILL avant la libération des handles DuckDB
# → fichier metadata.duckdb verrouillé au prochain démarrage sur Windows.
stop_timeout = 20000 # ms
```

Le commentaire [`api/server.go:117-121`](../apps/go-api/internal/api/server.go#L117-L121) documente aussi le problème côté caller :

```
// IMPORTANT : Close() pour décrémenter le refCount sinon le sql.DB
// reste ouvert au shutdown (le metaDB.Close() de cmd/server décrémente seulement
// d'un cran), ce qui retient le HANDLE Windows et provoque le verrou
// "metadata verrouillée" au prochain hot-reload Air.
```

Donc le problème est connu, partiellement traité, mais sans audit systématique. Ce document est l'audit qui manquait.
