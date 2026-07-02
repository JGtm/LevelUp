# Skill : halo-modes — Normalisation des modes de jeu Halo Infinite

> Spécifique à Halo Infinite (`title_slug = "halo_infinite"`).
> D'autres titres futurs auront leur propre logique de normalisation — ne pas généraliser ces fonctions sans vérifier leur compatibilité multi-titres.

## 2 niveaux ORTHOGONAUX — ne pas confondre

| Niveau | Fonction Go | Fonction Python (ref) | Exemple |
|---|---|---|---|
| **Sous-mode** (label affiché) | `NormalizeModeLabel()` | `resolve_display_mode()` | `"Arena:Slayer on Bazaar"` → `"Slayer"` |
| **Catégorie parente** (filtre UI) | `InferModeCategoryFromPairName()` | `infer_custom_category_from_pair_name()` | `"Arena:Slayer on Bazaar"` → `"Assassin"` |

## Catégories custom (constantes Go)

| Constante | Préfixes pair_name couverts |
|---|---|
| `ModeCategoryAssassin` | Arena, Tactical, Assault, Community |
| `ModeCategoryFiesta` | Fiesta, Castle Wars |
| `ModeCategorySuperFiesta` | Super Fiesta (**promu** vs Python v7) |
| `ModeCategoryHuskyRaid` | Husky Raid, Super Husky Raid (**promu** vs Python v7) |
| `ModeCategoryBTB` | BTB, BTB Heavies |
| `ModeCategoryRanked` | Ranked |
| `ModeCategoryFirefight` | Firefight, Gruntpocalypse |
| `ModeCategoryOther` | Event + tout préfixe inconnu |

**Divergence Python v7/cockpit** : Super Fiesta et Husky Raid sont des catégories distinctes en Go (Python les regroupait sous "Fiesta"). Ne pas revenir en arrière.

## Fichiers source

- `apps/go-api/internal/analysis/mode_label.go` — `NormalizeModeLabel(raw, mapLabels...)`
- `apps/go-api/internal/analysis/mode_category.go` — `InferModeCategoryFromPairName(pairName)`
- Python de référence : `git show v7/cockpit:src/analysis/mode_display.py` (_PREFIX_RULES)

## Format pair_name

```
"Arena:Slayer on Bazaar"   → préfixe="Arena", sous-mode="Slayer"
"BTB:CTF on Highpower"     → préfixe="BTB", sous-mode="CTF"
"Assassin : Classé"        → format FR avec " : " espacé → catégorie avant le séparateur
"Husky Raid"               → sans ":" → tester le label complet comme préfixe
```

## Règles de normalisation (NormalizeModeLabel)

1. Strip " sur {map}" / " on {map}" (noms de map connus en priorité)
2. Extraction depuis pair_name : format FR `" : "` → partie avant ; format `"Prefix:Mode"` → partie après `:`
3. Strip générique ` sur/on .+` résiduel
4. Strip ` - Forge` et ` - Ranked`

## Traductions (metadata.duckdb)

Les traductions EN→FR des sous-modes sont dans `metadata.duckdb → mode_name_tr`.
Les surcharges de paires map/mode sont dans `metadata.duckdb → mode_pair_overrides`.

## Helpers inverses

```go
// Obtenir les préfixes d'une catégorie (pour construire un WHERE)
PairNamePrefixesForCategory("Fiesta") // → ["Fiesta", "Castle Wars"]

// Tous les préfixes connus (pour filtre "Other" = NOT IN)
AllKnownPairNamePrefixes()
```
