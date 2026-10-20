# Audit Multi-Titres — LevelUp

> Date : 2026-04-22

---

## 1. Architecture globale

Le système multi-titres a été introduit au Sprint 44. Le principe :
- Un `TitleRegistry` (Go) est la source de vérité des titres supportés
- Un `PathResolver` gère les chemins physiques `data/titles/{slug}/`
- Un middleware `TitleExtractor` injecte le `title_slug` dans chaque requête HTTP via le contexte
- La session serveur (`SessionData.CurrentTitleSlug`) persiste le titre courant
- Le frontend transmet le titre via le header `X-LevelUp-Title` et gère un store Zustand

---

## 2. Ce qui est terminé ✅

| Zone | Fichiers |
|---|---|
| Registre + PathResolver | `apps/go-api/internal/domain/title/registry.go` |
| Middleware `TitleExtractor` | `apps/go-api/internal/api/middleware/title.go` |
| Pool DuckDB title-aware (clé `title_slug:gamertag`) | `apps/go-api/internal/platform/duckdb/pool.go` |
| Config v3 (`db_profiles.json`) + `ResolvePlayer` | `apps/go-api/internal/config/player_resolver.go` |
| Session `POST /session/context` | `apps/go-api/internal/api/handlers/session_context.go` |
| Bootstrap domain (`current_title_slug`, `available_titles`) | `apps/go-api/internal/domain/bootstrap.go` |
| `SessionData.CurrentTitleSlug` + `JobMeta.TitleSlug` | `apps/go-api/internal/domain/session.go`, `domain/job.go` |
| Compare + Leaderboard title-aware | `service/compare_service.go`, `service/leaderboard_service.go` |
| FanoutService + LabService propagent le titre | `service/fanout_service.go`, `service/lab_service.go` |
| Setup `POST /setup/players` injecte le titre | `apps/go-api/internal/api/handlers/setup.go` |
| Store Zustand `switchTitle()` + `isTitleSwitching` | `apps/web/src/stores/appShellStore.ts` |
| Header `X-LevelUp-Title` dans le client HTTP | `apps/web/src/lib/api/client.ts` |
| Types TS `TitleSummary`, `BootstrapResponse.available_titles` | `apps/web/src/lib/api/types.ts` |
| CLIs `cmd/levelup` et `cmd/refresh-metadata` avec flag `--title` | `apps/go-api/cmd/levelup/`, `cmd/refresh-metadata/main.go` |
| Tests multititle | `domain/title/multititle_test.go`, `handlers/multititle_test.go` |

Le système est **cohérent et fonctionnel pour un seul titre actif** (`halo_infinite`). Dès qu'un second titre est ajouté au registre, les gaps ci-dessous deviennent des régressions.

---

## 3. Gaps — plan priorisé

### ❌ Critique (bloque le multi-titre réel)

#### G1 — Pas de sélecteur de titre dans l'UI

**Problème** : `switchTitle()` et `isTitleSwitching` existent dans `appShellStore.ts` mais ne sont jamais appelés depuis un composant. `AppShellHeader` affiche le titre courant en badge statique lecture seule.

**Fix** : Créer un `TitleSwitcher` dans `AppShellHeader` — dropdown visible uniquement si `available_titles.length > 1`, badge statique sinon. Appelle `switchTitle(slug)` → `POST /session/context`.

---

#### G2 — `BootstrapService.Build()` ne filtre pas les joueurs par titre

**Fichier** : `apps/go-api/internal/service/bootstrap_service.go` L52

**Problème** : `cfg.LoadPlayers()` sans argument retourne tous les joueurs de tous les titres. Si `currentTitleSlug = "halo_mcc"`, la liste `available_players` contient quand même des joueurs Halo Infinite.

**Fix** :
```go
// Avant
players, err := cfg.LoadPlayers()

// Après
players, err := cfg.LoadPlayers(currentTitleSlug)
```

---

#### G3 — `"halo_infinite"` hardcodé dans `career_service` et `stats_service`

**Fichiers** :
- `apps/go-api/internal/service/career_service.go` L380 — `resolveCurrentSeason`
- `apps/go-api/internal/service/stats_service.go` L347 — `resolveCurrentSeason`

**Problème** : `titleID := "halo_infinite"` en dur au lieu de lire le titre depuis le `PlayerDB` ou le contexte.

**Fix** : Utiliser `pdb.TitleSlug` (déjà disponible dans le `PlayerDB` passé en paramètre) :
```go
// Avant
titleID := "halo_infinite"

// Après
titleID := pdb.TitleSlug
```

---

#### G4 — Type `SessionContextResponse` désynchronisé côté TypeScript

**Fichier** : `apps/web/src/lib/api/types.ts`

**Problème** : Le type Go `domain.SessionContextResponse` expose `AvailableTitles []TitleSummary`, mais l'interface TypeScript correspondante ne possède pas ce champ. Après un `POST /session/context`, le frontend ne peut pas mettre à jour la liste des titres disponibles.

**Fix** :
```typescript
export interface SessionContextResponse {
  current_title_slug: string
  available_titles?: TitleSummary[]  // ← manquant
}
```

---

### ⚠️ Mineur (dette, ne bloque pas aujourd'hui)

#### G5 — `PlayerSummary` sans `title_slug`

**Problème** : Si deux titres partagent un gamertag identique, `available_players` dans le bootstrap ne permet pas de les distinguer côté frontend.

**Fix** : Ajouter `TitleSlug string` dans `domain.PlayerSummary` (Go) + `title_slug?: string` dans le type TypeScript.

---

#### G6 — `resolveRealPlayer` charge cross-titres

**Fichier** : `apps/go-api/internal/config/player_resolver.go` L54

**Problème** : `cfg.LoadPlayers()` sans filtre — recherche globale par slug sans tenir compte du titre courant du contexte.

**Fix** : Passer le `titleSlug` du contexte en premier critère de recherche.

---

#### G7 — Dead code `setLastPlayerSlug` deprecated

**Fichier** : `apps/web/src/stores/settingsDraftStore.ts` L121-L129

**Problème** : La méthode marquée `@deprecated` hardcode `halo_infinite` comme clé et n'est appelée nulle part. C'est du code mort.

**Fix** : Supprimer la méthode et son usage éventuel.

---

#### G8 — CLIs `migrate-static-maps` et `populate-assets` hardcodent `"halo_infinite"`

**Fichiers** :
- `apps/go-api/cmd/migrate-static-maps/main.go` L120
- `apps/go-api/cmd/populate-assets/main.go` L354

**Problème** : Le flag `--title-id` est implémenté dans `cmd/refresh-metadata` mais absent de ces deux CLIs qui passent `"halo_infinite"` en dur.

**Fix** : Ajouter le flag `--title-id` (défaut `halo_infinite`) et le propager aux appels `UpsertMapImageRegistry` / `FetchAsset`.

---

## 4. Ordre d'implémentation suggéré

| Priorité | Gap | Effort estimé |
|---|---|---|
| 1 | G3 — hardcode career/stats services | ~5 min (2 lignes) |
| 2 | G2 — bootstrap filtre joueurs par titre | ~10 min (1 ligne + test) |
| 3 | G4 — type TS SessionContextResponse | ~5 min (1 champ) |
| 4 | G1 — TitleSwitcher UI | ~1h (nouveau composant) |
| 5 | G5 — PlayerSummary + title_slug | ~30 min (modèle Go + TS) |
| 6 | G6 — resolveRealPlayer title-aware | ~20 min |
| 7 — | G7 — dead code store | ~5 min |
| 8 | G8 — CLIs title flag | ~30 min (×2 CLIs) |
