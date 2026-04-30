# Référence i18n — Traductions des assets Halo

> **Source de vérité versionnée** sur l'architecture i18n du projet (mise à jour 2026-04-30).
>
> Ce document existe pour qu'un agent IA ou un développeur tombe dessus en cherchant « où sont les traductions », « quelle table contient les noms FR », « comment supporter une nouvelle langue », sans avoir à fouiller le code.
>
> Compagnon local (Claude Code) : `.claude/skills/halo-i18n/SKILL.md` (non versionné, pointe vers ce fichier).

---

## TL;DR — Carte des traductions

| Type d'asset | Source autoritative | Localisation |
|---|---|---|
| **Nom brut d'un asset** (playlist, pair, map, game_variant) | DiscoveryUGC `lang=` param | `metadata.duckdb → asset_translations` |
| **Sous-mode normalisé** (« Slayer » → « Assassin ») | Calcul Go via `NormalizeModeLabel()` | À termes : `metadata.duckdb → pair_mode_label_translations` (cf. PLAN_PLAYLISTS_CATALOG §4sexies) |
| **Catégorie de mode** (Assassin, Fiesta, BTB...) | Enum Go non traduit (constante stable) | `apps/go-api/internal/analysis/mode_category.go` (à termes : TOML versionné, cf. PLAN §4cinquies) |
| **Médaille** | Ref WaypointService | `metadata.duckdb → medal_definitions` (`name_fr`, `name_en`, `description_fr`, `description_en`) |
| **Rang de carrière** | Ref WaypointService | `metadata.duckdb → career_ranks` |
| **Surcharges manuelles** (pair_name cassé DiscoveryUGC) | Édition humaine | `metadata.duckdb → mode_pair_overrides` |
| **Traduction sous-mode EN→FR** (héritage Python) | Édition humaine | `metadata.duckdb → mode_name_tr` |

---

## Langues supportées

DiscoveryUGC accepte un `lang=` param. Liste typique Halo Infinite (à confirmer en lisant [discovery_client.go](apps/go-api/internal/platform/halo/discovery_client.go) `doGetWithLang`) :

```
en, fr, de, es-ES, es-MX, it, ja, ko, nl, pl, pt-BR, ru, zh-CN, zh-TW
```

**État courant** : l'app productive cible aujourd'hui **EN + FR seulement**. Les colonnes `_fr` de `match_registry` existent dans le schéma mais ne sont jamais peuplées (mesure 2026-04-30 : 0/1545 matchs) — la résolution FR se fait à chaud à chaque vue UI.

**État cible** (PLAN_PLAYLISTS_CATALOG §4sexies) : hydratation complète des 14 langues DiscoveryUGC dans `asset_translations` au bootstrap initial du catalogue, refresh mensuel pour les changements. Coût marginal négligeable, gain de runtime massif côté UI.

---

## Architecture cible (3 couches)

| Couche | Stockage | Multi-lang ? |
|---|---|---|
| **Nom canonique** (debug + lookup rapide) | Inline `name_canonical VARCHAR` dans chaque table catalogue (EN par défaut) | Non (1 valeur) |
| **Noms bruts traduits** | Table partagée `metadata.asset_translations` (existante) | **Oui native via colonne `lang`** |
| **Labels normalisés** (sortie `NormalizeModeLabel`) | Nouvelle table `metadata.pair_mode_label_translations(pair_asset_id, lang, label)` | **Oui native** |

Pourquoi cette décomposition :
- `name_canonical` inline évite un JOIN systématique pour les requêtes simples ou de debug.
- `asset_translations` est déjà multi-lang depuis le début — ajouter une langue = ajouter des lignes, pas de migration.
- `pair_mode_label_translations` sépare les labels **dérivés** (calcul de normalisation par langue) des noms bruts (sortie DiscoveryUGC).

---

## Anti-patterns à proscrire (revue de PR i18n)

### ❌ Recalculer la traduction FR à chaud à chaque vue UI

**Symptôme** : code qui appelle `resolve_display_mode(pair_name, lang="fr")` ou équivalent dans une boucle de rendu.

**État actuel** : `match_registry.{pair_name_fr, map_name_fr, playlist_name_fr}` ne sont jamais peuplées (0/1545 matchs vérifié 2026-04-30). Les vues UI calculent à chaud à chaque requête. Coût runtime gaspillé.

**Bon réflexe** : consulter d'abord `asset_translations` (ou `pair_mode_label_translations` à termes). Si absent, le calcul à chaud est un fallback tolérable mais à signaler comme dette.

### ❌ Créer une nouvelle table `xxx_translations`

**Symptôme** : un PR ajoute `playlist_translations`, `map_labels_fr`, ou similaire.

**Vérifier d'abord** :
1. `asset_translations(asset_id, asset_type, lang, name, description)` couvre les **noms bruts** d'asset → étendre avec un nouveau `asset_type` plutôt que créer une table.
2. `pair_mode_label_translations(pair_asset_id, lang, label)` couvre les **labels normalisés** dérivés.
3. Si ni l'un ni l'autre ne couvre le besoin, justifier dans le commit avec exemples de requêtes.

### ❌ Ajouter une colonne `name_fr` inline à une table métier

**Symptôme** : une migration ajoute `name_fr VARCHAR` à `match_registry`, `match_participants`, ou autre table métier.

**Pourquoi c'est cassé** : la colonne est calculée à chaud puis pas peuplée (cf. `match_registry.pair_name_fr` aujourd'hui), ou peuplée une fois et jamais re-fetchée si l'asset change de nom.

**Bon réflexe** : référencer l'asset_id et JOIN sur `asset_translations` au moment de la lecture, OU peupler via le pipeline catalogue qui re-fetch périodiquement (refresh mensuel).

### ❌ Hardcoder une catégorie de mode en français dans le code Go

**Symptôme** : `if category == "Assassin" { ... }` ou `case "Assassin": ...` dans la logique métier.

**Pourquoi c'est cassé** : « Assassin » est déjà la catégorie canonique (constante Go), mais c'est aussi la traduction française de « Slayer ». Confusion garantie.

**Bon réflexe** : utiliser les constantes `ModeCategoryAssassin` etc. de [mode_category.go](apps/go-api/internal/analysis/mode_category.go), jamais le string littéral.

### ❌ Supposer que `discovery_client.go` ne supporte qu'une langue

DiscoveryUGC a `doGetWithLang(ctx, ..., lang)` depuis le début. Pas besoin de wrapper, pas besoin de monkey-patch. Juste passer le bon `lang` au moment du fetch.

---

## Helpers à utiliser, pas réinventer

### Côté Go

```go
// apps/go-api/internal/analysis/
NormalizeModeLabel(raw, mapLabels...)           // sous-mode propre
InferModeCategoryFromPairName(pairName)         // catégorie enum
PairNamePrefixesForCategory(category)           // helper inverse
AllKnownPairNamePrefixes()                      // pour filtre Other
```

### Côté Python (legacy)

```python
# src/analysis/
resolve_display_mode(pair_name, mode_category, lang, tables)
infer_custom_category_from_pair_name(pair_name)
```

### SQL — patterns types

```sql
-- Lookup direct d'une traduction
SELECT name FROM asset_translations
WHERE asset_id = ? AND asset_type = 'playlist' AND lang = 'fr';

-- JOIN catalogue + traduction (pattern de lecture standard)
SELECT
  p.playlist_asset_id,
  COALESCE(t.name, p.name_canonical) AS display_name,
  p.experience
FROM playlists_catalog p
LEFT JOIN asset_translations t
  ON  t.asset_id   = p.playlist_asset_id
 AND  t.asset_type = 'playlist'
 AND  t.lang       = $1
WHERE p.title_slug = 'halo_infinite'
  AND p.is_active  = TRUE;

-- Idem pour les labels normalisés
SELECT
  d.pair_asset_id,
  COALESCE(l.label, d.name_canonical) AS mode_label,
  d.mode_category
FROM map_mode_pair_definitions d
LEFT JOIN pair_mode_label_translations l
  ON  l.pair_asset_id = d.pair_asset_id
 AND  l.lang          = $1
WHERE d.title_slug = 'halo_infinite';
```

---

## Fichiers source de référence

| Fichier | Rôle |
|---|---|
| [apps/go-api/internal/platform/halo/discovery_client.go](apps/go-api/internal/platform/halo/discovery_client.go) | `doGetWithLang` + langues supportées |
| [apps/go-api/internal/migration/steps_metadata.go](apps/go-api/internal/migration/steps_metadata.go) | Schéma `asset_translations`, `mode_name_tr`, `mode_pair_overrides`, `medal_definitions`, `career_ranks` |
| [apps/go-api/internal/analysis/mode_label.go](apps/go-api/internal/analysis/mode_label.go) | `NormalizeModeLabel` |
| [apps/go-api/internal/analysis/mode_category.go](apps/go-api/internal/analysis/mode_category.go) | `InferModeCategoryFromPairName` + constantes (à termes : loader TOML) |
| `.ai/PLAN_PLAYLISTS_CATALOG.md` §4sexies | Architecture i18n cible (multi-langues complet) |
| `.ai/PLAN_PLAYLISTS_CATALOG.md` §4cinquies | TOML mode_category + auto-détection préfixes |
| `.claude/skills/halo-modes/SKILL.md` | Skill compagnon : règles de normalisation modes (2 niveaux orthogonaux) |
| `.claude/skills/halo-i18n/SKILL.md` | Skill compagnon : pointe vers ce document |

---

## Quand consulter cette référence

- Avant d'ajouter une colonne `_fr` à une table existante
- Avant de créer une table `xxx_translations`
- Avant d'écrire une fonction qui résout un nom d'asset en français/anglais
- Avant de décider qu'une traduction « manque » (vérifier `asset_translations` d'abord)
- Avant de hardcoder le nom français d'une catégorie ou d'un mode dans la logique métier
- Avant d'ajouter une nouvelle langue à supporter (vérifier d'abord ce que DiscoveryUGC fournit nativement)
