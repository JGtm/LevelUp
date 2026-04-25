# tools/mappings/CHANGELOG.md — historique des bumps de schema_version des TOML

> Toute modification breaking de la structure d'un TOML sous
> `config/titles/{slug}/mappings/` doit incrémenter `schema_version` dans
> le `[meta]` du fichier concerné, et apparaître dans ce CHANGELOG avec :
> - la date au format `YYYY-MM-DD`
> - le titre concerné (ou `*` si tous)
> - le before/after du schéma
> - la stratégie de migration up + down

---

## 2026-04-25 — schema_version=1 (initial)

**Titres concernés :** `halo_infinite`, `synthetic_title_b`

**Schéma initial** introduit par la Phase A du plan
[`PLAN_MULTI_TITLE_ADAPTERS_AND_MAPPINGS.md`](../../.ai/PLAN_MULTI_TITLE_ADAPTERS_AND_MAPPINGS.md).

Champs obligatoires d'un `[fields.X]` :

- `labels.en` + `labels.fr` (string, non vide)
- `storage_unit` (enum : `count` | `ratio` | `percent` | `seconds` | `milliseconds` | `""`)
- `display_unit` (même enum)
- `format` (enum : `integer` | `signed_int` | `percent_1` | `percent_2` | `kdr_2` | `duration_hms` | `seconds` | `string` | `boolean` | `datetime` | `enum`)
- `display_order` (int > 0, unique au sein du même `group`)
- `group` (string non vide)

Champs optionnels :

- `description.en` + `description.fr` (les deux ou aucun, jamais un seul)
- `icon` (string)

Conversions d'unités déclarées comme supportées par
[`internal/games/mappings/units.go`](../../apps/go-api/internal/games/mappings/units.go) :

- `ratio ↔ percent`
- `seconds ↔ milliseconds`
- identité (storage_unit == display_unit)

Toute conversion non listée fait échouer le boot du titre concerné.

---

## Procédure générale pour bumper schema_version

1. Modifier la struct `fieldsTOML` / `fieldEntryTOML` dans
   [`loader.go`](../../apps/go-api/internal/games/mappings/loader.go) si la
   structure change (ajout/suppression de champs).
2. Mettre à jour la validation correspondante dans `validateField()`.
3. Si le bump est breaking, ajouter un dispatcher dans le loader qui choisit
   la struct selon `schema_version` (support N et N-1 simultanés).
4. Créer un script `tools/mappings/migrate_v{N}_to_v{N+1}.go` exécutable qui
   transforme un TOML vN en vN+1 (et idéalement l'inverse pour rollback).
5. Régénérer les TOML existants : `go run ./tools/mappings/migrate_v1_to_v2.go`.
6. Bumper `schema_version` dans chaque TOML concerné.
7. Documenter ici l'avant/après et la stratégie de migration.
8. Garder le support de la version N-1 au moins 1 sprint pour permettre
   un rollback gracieux.
