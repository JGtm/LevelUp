---
name: halo-modes
description: Normalisation des libellés de mode Halo Infinite (grammaires de pair_name, sous-mode vs catégorie parente, paquet analysis/modelabel, taxonomie injectée). À invoquer avant de toucher à un libellé de mode, à un filtre de catégorie ou à mode_category, côté Go comme côté web.
---

# Skill : halo-modes — Normalisation des modes de jeu Halo Infinite

> La grammaire des `pair_name` est celle de Halo Infinite. Mais `analysis.NormalizeModeLabel`
> est partagé par tous les titres : Halo 5 passe par `ResolveModeUIWithVariant` (repli sur
> `game_variant`). Seule la CATÉGORIE (table de préfixes) est propre à Halo Infinite.
>
> Elle n'arrive dans `analysis/`, `service/` et les handlers que par `analysis.ModeTaxonomy`,
> injectée par `api/wire` (`haloInfiniteModeTaxonomy`, `api/wire/registry_media.go`). Ne jamais
> importer `games/halo_infinite` depuis ces couches (CLAUDE.md, multi-titre). Une taxonomie
> vide ne classe rien : c'est la dégradation prévue pour un titre sans catégories.

## 2 niveaux ORTHOGONAUX — ne pas confondre

| Niveau | Fonction Go | Exemple |
|---|---|---|
| **Sous-mode** (label affiché) | `analysis.NormalizeModeLabel()` | `"Arena:Slayer on Bazaar"` → `"Slayer"` |
| **Catégorie parente** (filtre UI) | `modelabel.InferCategory()` (façade : `InferModeCategoryFromPairName()`) | `"Arena:Slayer on Bazaar"` → `"Assassin"` |

Pour afficher un mode, appeler les résolveurs de `analysis/mode_label.go` plutôt que
`NormalizeModeLabel` seul :

- `ResolveModeUI` / `ResolveModeUIIn` : libellé du `pair_name`, dans la langue de la requête pour `…In`.
- `ResolveModeUIWithVariant` / `…In` : pair d'abord, sinon `game_variant` (Halo 5). Source unique de cette convention.
- `ExtractKnownMode` : retrouve un mode connu dans un libellé déjà normalisé (« Tactical Slayer » → « Slayer »).

## Catégories custom (constantes Go)

Les constantes sont dans `analysis/modelabel` (`Category*`) ; `games/halo_infinite` les
ré-exporte en alias (`ModeCategory*`).

| Constante | Préfixes pair_name couverts |
|---|---|
| `ModeCategoryAssassin` | Arena, Tactical, Assault, Community |
| `ModeCategoryFiesta` | Fiesta, Castle Wars |
| `ModeCategorySuperFiesta` | Super Fiesta |
| `ModeCategoryHuskyRaid` | Husky Raid, Super Husky Raid |
| `ModeCategoryBTB` | BTB, BTB Heavies |
| `ModeCategoryRanked` | Ranked |
| `ModeCategoryFirefight` | Firefight, Gruntpocalypse |
| `ModeCategoryOther` | Event + tout préfixe inconnu |

Ces constantes sont des CLÉS : valeurs stockées dans `match_registry.mode_category` et clés de
filtre. Ne jamais les afficher telles quelles : le libellé affiché passe par le manifest web
(`media.toolbar.mode_categories.*` dans `lib/i18n/manifests/media.toml`, une valeur par langue).

**Super Fiesta et Husky Raid sont des catégories distinctes** (l'ancien cockpit les
regroupait sous « Fiesta »). Ne pas revenir en arrière.

## Fichiers source

- `apps/go-api/internal/analysis/mode_label.go` — `NormalizeModeLabel(raw, mapLabels...)`, résolveurs `ResolveModeUI*`, `ExtractKnownMode`
- `apps/go-api/internal/analysis/modelabel/category.go` — LA règle de catégorie (`InferCategory`, `PrefixesForCategory`, `KnownPrefixes`, table des préfixes). Paquet feuille, importable par sync et migration.
- `apps/go-api/internal/analysis/modelabel/container.go` — jetons de CONTENEUR de la grammaire (`SplitContainer`, `IsContainer`), liste UNIQUE
- `apps/go-api/internal/analysis/modelabel/strip.go` — `StripMapSuffix` (regex unique du retrait « on / sur carte ») ; `extract.go` — règle d'appariement de `ExtractKnownMode`
- `apps/go-api/internal/games/halo_infinite/mode_category.go` — façade : mêmes fonctions sous les noms historiques du titre (`InferModeCategoryFromPairName`, `PairNamePrefixesForCategory`, `AllKnownPairNamePrefixes`)
- `apps/go-api/internal/analysis/match_filter.go` — `ModeTaxonomy` (taxonomie injectée)
- Garde : `apps/go-api/internal/sync/mode_category_single_rule_test.go` (une seule règle de catégorie dans le dépôt)

## Miroir web

`apps/web/src/lib/halo/modeLabel.ts` (`normalizeModeLabel`) reprend la normalisation côté web.
Attention : il diverge du Go sur les identités de playlist et la forme inversée
(`Super Fiesta:Slayer` rend « Slayer » côté web, « Super Fiesta » côté Go). Constat connu,
non corrigé.

## Format pair_name — DEUX grammaires

```
"Arena:Slayer on Bazaar"   → conteneur="Arena", sous-mode="Slayer"
"BTB:CTF on Highpower"     → conteneur="BTB", sous-mode="CTF"
"Assassin : Classé"        → format FR avec " : " espacé → partie avant le séparateur
"Husky Raid"               → sans ":" → tester le label complet comme préfixe
```

Forme INVERSÉE `"Mode:Conteneur [qualificatif] on Carte"` (l'API la produit autant que l'autre) :

```
"Slayer:Arena on Live Fire"             → sous-mode="Slayer"            (reste vide → le mode est à gauche)
"CTF:Arena Neutral Flag on Cliffhanger" → sous-mode="Neutral Flag CTF"  (qualificatif recollé DEVANT le mode)
"Slayer:Arena Super Fiesta on Chasm"    → sous-mode="Super Fiesta"      (qualificatif = identité de playlist → identité)
```

Règle de recollage : si la partie droite commence par un jeton de conteneur (`Arena`, `BTB`,
`BTB Heavies`, `Ranked`, `Fiesta`, `Super Fiesta`, `Tactical`, `Community`, `Assault`, `Event`,
`Husky Raid`, `Super Husky Raid`, `Castle Wars`, `Doubles` — mot entier, insensible à la casse,
le plus long gagne), le reste après le conteneur est `rest` : vide → mode = gauche ; identité
de playlist → identité canonique ; sinon mode = `rest + " " + gauche` (l'ordre des clés de
`mode_name_tr` : `Neutral Flag CTF`, `Tactical Slayer`, `Fiesta CTF`). Sans conteneur ni à
gauche ni en tête à droite → partie droite (`Infection:Alpha Zombies` → `Alpha Zombies`).

## Règles de normalisation (NormalizeModeLabel)

1. Strip " sur {map}" / " on {map}" (noms de map connus en priorité). La regex ne couvre que
   `on` et `sur` (français et anglais) ; le comportement pour une autre langue n'est pas vérifié.
2. Extraction depuis pair_name (découpe sur le DERNIER `:`) :
   - format FR `" : "` → partie avant ;
   - identité de playlist à gauche (Super Fiesta, Husky Raid, Super Husky Raid) → identité
     canonique, testée AVANT la règle du conteneur (ces identités sont aussi des conteneurs) ;
   - `"Conteneur:Mode"` → partie après `:` ;
   - forme inversée `"Mode:Conteneur [qualificatif]"` → règle de recollage ci-dessus.
3. Strip générique ` sur/on .+` résiduel
4. Strip ` - Forge` et ` - Ranked`

## Traductions (metadata.duckdb)

Les traductions des noms de mode sont dans `metadata.duckdb → mode_name_tr (mode_en, lang, name)` :
une ligne par langue de `config/locales.toml`, `mode_en` est le nom anglais, clé de jointure.

## Helpers inverses

```go
// Obtenir les préfixes d'une catégorie (pour construire un WHERE)
PairNamePrefixesForCategory("Fiesta") // → {"Fiesta", "Castle Wars"} (ordre non garanti)

// Tous les préfixes connus (pour filtre "Other" = NOT IN)
AllKnownPairNamePrefixes()
```
