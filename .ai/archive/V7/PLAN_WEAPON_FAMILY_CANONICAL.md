# PLAN_WEAPON_FAMILY_CANONICAL.md — Plan d'introduction d'une couche `weapon_family` canonique cross-titres

> Plan rédigé le 2026-04-25, sur branche `feat/accessibility-okabe-ito` (note : sera commité dans un PR dédié distinct de l'accessibilité).
>
> Document complémentaire à `.ai/PLAN_MULTI_TITLE_ADAPTERS_AND_MAPPINGS.md` qui en avait acté l'idée comme « hors scope, à faire quand un second titre réel arrivera ». Ce plan documente le **cadrage à froid** pour pouvoir l'implémenter rapidement le moment venu, sans rouvrir le débat conceptuel.
>
> **État** : plan documentaire uniquement. Pas d'implémentation tant que :
> 1. la Phase A du plan multi-titres n'est pas démarrée ;
> 2. un second titre réel (Halo 5, MCC, ODST…) n'est pas validé en pipeline produit.

---

## TL;DR

1. **Concept** : `weapon_family` = identifiant canonique cross-titres (ex : `battle_rifle`, `assault_rifle`, `sniper_rifle`, `rocket_launcher`) qui regroupe les armes équivalentes au-delà des `weapon_id` filmshell par-titre.
2. **Stockage** : nouvelle table metadata `weapon_families` (référentiel global, partagé) + colonne `family_key` ajoutée à `weapon_labels` de chaque titre.
3. **Source de vérité** : un **TOML global** `config/canonical/weapon_families.toml` (versionné Git) liste les familles canoniques + leurs libellés FR/EN + leur tier (`primary`, `secondary`, `power`, `melee`, `grenade`, `vehicle`).
4. **Mapping `weapon_id` → `family_key`** : par-titre, dans `config/titles/{slug}/mappings/weapon_families.toml`. Le mapping est partiel par défaut (NULL = arme « ungrouped », son label remonte tel quel).
5. **Exposition** : `TitleSemanticAdapter.WeaponFamilies()` + résolveur `WeaponFamilyOf(weapon_id)`. Aucun changement à l'API HTTP existante en MVP ; un endpoint `/weapon-families` est introduit quand un second titre branche dessus.
6. **Hors scope** : harmonisation des stats par famille (kills par famille cross-titres) — c'est de l'analytics produit, viendra après.
7. **Effort estimé** : 2–3 jours-personne. Bloqué par : Phase A multi-titres + arrivée d'un second titre.

---

## 1. Pourquoi cette couche

### 1.1. Observation produit

Les armes Halo se répètent largement d'un titre à l'autre, avec variantes mineures de noms :

| Famille canonique | Halo Infinite | Halo 5 | Halo Reach | Halo CE |
|---|---|---|---|---|
| `battle_rifle` | BR75 | BR85 | (absent) | (absent) |
| `assault_rifle` | MA40 AR | MA5D AR | MA37 AR | MA5B AR |
| `sniper_rifle` | S7 Sniper | SRS99-S5 AM | SRS99-AM | SRS99C-S2 AM |
| `rocket_launcher` | M41 SPNKR | M41A2 SPNKR | M41 SSM | M41 SSR |
| `plasma_pistol` | Plasma Pistol | Plasma Pistol | Plasma Pistol | Plasma Pistol |
| `energy_sword` | Energy Sword | Energy Sword | Energy Sword | Energy Sword |

Sans couche canonique, comparer « kills au BR » entre Infinite et Halo 5 demande une jointure ad hoc à chaque endroit. Avec, c'est `WHERE family_key = 'battle_rifle'` partout.

### 1.2. Pourquoi pas tout de suite

1. Aucun second titre n'est encore branché → la valeur produit est nulle aujourd'hui.
2. Phase A du plan multi-titres pas commencée → on construirait sur du sable (le `TitleSemanticAdapter` n'existe pas encore).
3. Le mapping `weapon_id` → `family_key` est laborieux (~150 armes par titre) et demande une décision produit titre par titre.

Donc : **plan posé maintenant**, **implémentation plus tard**.

---

## 2. Périmètre

### 2.1. Inclus

1. Modélisation conceptuelle : qu'est-ce qu'une `weapon_family` ?
2. Schéma de stockage : table DuckDB + colonne FK + TOML global + TOML par titre.
3. Liste initiale des familles canoniques (~25 familles couvrant les armes de Halo Infinite).
4. Plan de mapping `weapon_id` → `family_key` pour HI (point de départ).
5. Articulation avec `TitleSemanticAdapter` du plan multi-titres.
6. Plan de tests + non-régression.
7. Plan de bascule progressive (3 phases).

### 2.2. Exclu

1. Implémentation effective (bloquée par Phase A multi-titres).
2. Analytics dérivées (kills par famille, stats agrégées) — viendra après dans un plan séparé.
3. Familles spécifiques à un titre unique (ex : armes Forerunner de H5) → restent uniquement dans le titre concerné, sans famille canonique.
4. Mapping pour titres autres que HI tant que les corpus ne sont pas validés.
5. UI dédiée « comparaison cross-titres par famille » — viendra avec le second titre réel.

---

## 3. Modélisation conceptuelle

### 3.1. Définition

Une **`weapon_family`** est un regroupement d'armes équivalentes en termes de **rôle gameplay**, indépendamment du titre :

1. silhouette (zoom-snipe, balayage rapide, anti-véhicule…) ;
2. tier (primaire / secondaire / power / melee / grenade / vehicle) ;
3. type de dégâts (ballistic, plasma, hardlight, energy, kinetic).

Deux armes sont dans la même famille si **un joueur les utilise dans la même situation tactique**, pas si leur modèle 3D se ressemble.

Exemples :
- BR Halo Infinite + BR Halo 5 = `battle_rifle` (même rôle, même tier, ballistic)
- BR Halo Infinite + AR Halo Infinite = familles différentes (`battle_rifle` vs `assault_rifle`, rôle différent)
- BR Halo Infinite + Storm Rifle = familles différentes (ballistic vs plasma, dégâts différents)

### 3.2. Granularité

**Niveau retenu** : moyenne. Ni trop fin (pas une famille par variante mineure : `BR75` et `BR85` = même famille `battle_rifle`), ni trop large (pas une famille « ranged » qui mélangerait BR + Sniper + Rocket).

Heuristique : **si la communauté Halo nomme deux armes par le même surnom au quotidien (« BR », « Sniper », « Rocket »), elles sont dans la même famille**.

### 3.3. Cas limites tranchés

| Cas | Décision | Raison |
|---|---|---|
| Variantes du même titre (BR75 vs MK50) | Familles différentes si tier/comportement diffère, sinon même famille | Le contenu DLC qui n'est qu'un reskin = même famille |
| Dual-wield d'une arme | Même famille, métadonnée `dual_wieldable: true` portée par le titre | Le rôle gameplay reste le même |
| Power weapon vs Variant (Sniper vs Halo CE Sniper) | Même famille `sniper_rifle` | Rôle identique, juste régulation différente |
| Forerunner armes (H5) | Familles dédiées (`hardlight_repeater`, `binary_rifle`, …) | Rôle gameplay distinct des armes ballistic/plasma |
| Armes uniques (Skewer, Sentinel Beam) | Famille dédiée si réutilisée dans 2+ titres ; sinon pas de famille | Une famille à 1 titre n'a pas d'intérêt canonique |

---

## 4. Schéma de stockage

### 4.1. Référentiel global : `weapon_families`

Nouvelle table dans une **DB metadata globale** (pas une DB par titre, car le référentiel est partagé) :

```sql
-- data/warehouse/canonical_metadata.duckdb (nouvelle DB globale)
CREATE TABLE weapon_families (
  family_key      VARCHAR PRIMARY KEY,    -- "battle_rifle", "sniper_rifle", ...
  tier            VARCHAR NOT NULL,       -- "primary" | "secondary" | "power" | "melee" | "grenade" | "vehicle"
  damage_type     VARCHAR,                -- "ballistic" | "plasma" | "hardlight" | "energy" | "kinetic"
  dual_wieldable  BOOLEAN DEFAULT FALSE,
  introduced_at   VARCHAR,                -- "halo_ce" (titre où la famille apparaît)
  schema_version  INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE weapon_family_translations (
  family_key  VARCHAR NOT NULL,
  locale      VARCHAR NOT NULL,           -- "en" | "fr"
  label       VARCHAR NOT NULL,
  description VARCHAR,
  PRIMARY KEY (family_key, locale),
  FOREIGN KEY (family_key) REFERENCES weapon_families(family_key)
);
```

Cette DB **n'appartient à aucun titre** ; elle est globale comme `app_settings.json` ou `db_profiles.json` (cf. ADR_S44).

### 4.2. Mapping par titre : extension de `weapon_labels`

Pour chaque titre, ajouter une colonne FK :

```sql
ALTER TABLE weapon_labels ADD COLUMN family_key VARCHAR;
-- pas de FK contrainte SQL stricte (DBs séparées physiquement),
-- mais le loader Go vérifie la cohérence au boot.
```

Une `weapon_id` de Halo Infinite peut donc pointer vers une `family_key` canonique. Si elle ne pointe sur rien (NULL), l'arme est « ungrouped » et ses kills sont comptés à part dans toute future analytics par famille.

### 4.3. Source de vérité : TOML global

```
config/canonical/weapon_families.toml         # référentiel global (familles + labels)
config/titles/halo_infinite/mappings/weapon_families.toml  # mapping weapon_id -> family_key
```

Le seed des tables DuckDB se fait au boot (ou via `tools/seed-weapon-families.go`) en lisant ces TOML. La DB est un cache requêtable, le TOML est la source vérité Git.

### 4.4. Schéma `weapon_families.toml` global

```toml
[meta]
schema_version = 1

# === Primaires balistiques ===

[families.assault_rifle]
tier            = "primary"
damage_type     = "ballistic"
labels          = { en = "Assault Rifle",  fr = "Fusil d'assaut" }
description     = { en = "Fully automatic ballistic primary.",
                    fr = "Fusil balistique automatique." }
introduced_at   = "halo_ce"

[families.battle_rifle]
tier            = "primary"
damage_type     = "ballistic"
labels          = { en = "Battle Rifle",   fr = "Fusil de combat" }
description     = { en = "Burst-fire mid-range ballistic primary.",
                    fr = "Fusil balistique tir en rafale, moyenne portée." }
introduced_at   = "halo_2"

[families.dmr]
tier            = "primary"
damage_type     = "ballistic"
labels          = { en = "DMR",            fr = "DMR" }
description     = { en = "Semi-auto precision ballistic primary.",
                    fr = "Fusil balistique semi-auto de précision." }
introduced_at   = "halo_reach"

# === Secondaires ===

[families.magnum]
tier            = "secondary"
damage_type     = "ballistic"
labels          = { en = "Magnum",         fr = "Magnum" }
introduced_at   = "halo_ce"

[families.plasma_pistol]
tier            = "secondary"
damage_type     = "plasma"
dual_wieldable  = true
labels          = { en = "Plasma Pistol",  fr = "Pistolet à plasma" }
introduced_at   = "halo_ce"

# === Power weapons ===

[families.sniper_rifle]
tier            = "power"
damage_type     = "ballistic"
labels          = { en = "Sniper Rifle",   fr = "Fusil de précision" }
introduced_at   = "halo_ce"

[families.rocket_launcher]
tier            = "power"
damage_type     = "ballistic"
labels          = { en = "Rocket Launcher", fr = "Lance-roquettes" }
introduced_at   = "halo_ce"

[families.energy_sword]
tier            = "power"
damage_type     = "energy"
labels          = { en = "Energy Sword",   fr = "Épée à énergie" }
introduced_at   = "halo_2"

# ... voir annexe §10 pour la liste exhaustive (~25 familles)
```

### 4.5. Schéma `weapon_families.toml` par titre

```toml
# config/titles/halo_infinite/mappings/weapon_families.toml

[meta]
title_slug     = "halo_infinite"
schema_version = 1

# weapon_id (UBIGINT filmshell) -> family_key
# Une arme absente de cette table = ungrouped (NULL)

[mapping]
1234567890 = "battle_rifle"     # BR75
2345678901 = "assault_rifle"    # MA40 AR
3456789012 = "sniper_rifle"     # S7 Sniper
4567890123 = "rocket_launcher"  # M41 SPNKR
5678901234 = "energy_sword"     # Energy Sword
# ... ~80–120 armes mappées sur les ~150 weapon_id de HI
```

### 4.6. Validation au chargement

Au boot Go (Phase A multi-titres + extension weapon_families) :

1. parser le TOML global → vérifier unicité `family_key`, présence locales `en` + `fr`, `tier` dans enum, `damage_type` dans enum ;
2. parser les TOML par-titre → vérifier que tout `family_key` cité existe dans le global ;
3. logger `weapon_families_loaded` avec `families_count`, `mapped_weapons_count`, `unmapped_weapons_count` par titre ;
4. erreur de validation = refus du seed (mais l'API tourne quand même, juste sans famille canonique).

---

## 5. Articulation avec le plan multi-titres

### 5.1. `TitleSemanticAdapter` étendu

```go
type TitleSemanticAdapter interface {
    // ... (existant : Fields, Assets, Outcomes)

    // Nouveaux :
    WeaponFamilies() WeaponFamilySet
    WeaponFamilyOf(weaponID int64) (canonical.FamilyKey, bool)
}
```

`WeaponFamilySet` est partagé entre tous les titres (lit depuis la DB globale `canonical_metadata.duckdb`), `WeaponFamilyOf` est titre-spécifique (lit depuis `weapon_labels.family_key` du titre courant).

### 5.2. Endpoint HTTP

Nouveau, **introduit seulement quand un second titre branche dessus** :

```
GET /api/v1/weapon-families?locale=fr
```

Réponse :

```json
{
  "schema_version": 1,
  "locale": "fr",
  "families": {
    "battle_rifle": {
      "label": "Fusil de combat",
      "description": "Fusil balistique tir en rafale, moyenne portée.",
      "tier": "primary",
      "damage_type": "ballistic",
      "introduced_at": "halo_2",
      "available_in_titles": ["halo_2", "halo_3", "halo_infinite"]
    }
  }
}
```

Le champ `available_in_titles` est dérivé au runtime : un titre apparaît si au moins une de ses armes pointe vers cette famille.

### 5.3. Pas de bascule du Match View en MVP

Le Match View continue d'afficher le `weapon_label` brut (« BR75 » et pas « Fusil de combat »). La famille canonique n'apparaît **que** :

1. dans des futures analytics cross-titres ;
2. dans un éventuel filtre Explorer « Top kills par famille d'arme » ;
3. dans des badges de comparaison (« Tu utilises 2x plus le BR que la moyenne Halo 5 »).

Ces surfaces produit sont **hors scope de ce plan**.

---

## 6. Plan de bascule en 3 phases

### Phase 1 — Référentiel global (1j)

1. créer `data/warehouse/canonical_metadata.duckdb` avec les 2 tables `weapon_families` + `weapon_family_translations` ;
2. créer `config/canonical/weapon_families.toml` avec ~25 familles canoniques (annexe §10) ;
3. script `tools/seed-weapon-families.go` qui lit le TOML et seed la DB ;
4. tests unitaires + golden TOML.

### Phase 2 — Mapping HI (1j)

1. ajouter colonne `family_key` à `weapon_labels` de HI (migration DuckDB) ;
2. créer `config/titles/halo_infinite/mappings/weapon_families.toml` avec mapping `weapon_id` → `family_key` (~80–120 lignes, tableau de mapping construit en cross-référençant la DB existante) ;
3. script `tools/seed-weapon-families-mapping.go --title halo_infinite` ;
4. tests : pour chaque arme HI mappée, vérifier que la famille existe dans le référentiel global ; logguer le ratio `mapped/total`.

### Phase 3 — Adapter + endpoint (0.5–1j)

1. étendre `TitleSemanticAdapter` avec `WeaponFamilies()` + `WeaponFamilyOf()` ;
2. handler `/api/v1/weapon-families` derrière flag `WEAPON_FAMILIES_API_ENABLED=false` (off par défaut) ;
3. tests unitaires sur les adapters + endpoint.

**Total : 2.5–3 jours-personne**, sans changement UI ni régression.

---

## 7. Tests

### 7.1. Unitaires

| Composant | Couverture |
|---|---|
| Loader TOML global | parsing + 5 fixtures invalides (locale manquante, tier inconnu, family_key dupliquée…) |
| Loader TOML par-titre | parsing + 3 fixtures invalides (family_key inconnue, weapon_id non numérique, doublons) |
| Resolver `WeaponFamilyOf` | arme mappée → famille correcte, arme non mappée → `(_, false)`, titre inconnu → erreur |
| Endpoint `/weapon-families` | locale connue/inconnue (fallback en), titre filter optionnel |

### 7.2. Cohérence cross-titres

Test CI : pour **chaque titre** ayant un mapping `weapon_families.toml`, vérifier :

1. tout `family_key` cité existe dans `config/canonical/weapon_families.toml` ;
2. aucun `weapon_id` n'est mappé à 2 familles différentes ;
3. taux de couverture `mapped/total` ≥ 60 % (sinon warning ; en dessous de 30 %, échec).

### 7.3. Non-régression

Aucun changement attendu sur le golden parity HI (Match View affiche toujours `weapon_label` brut). Si une régression apparaît, c'est qu'un caller a commencé à consommer la famille canonique → alerte.

---

## 8. Logging

| Événement | Niveau | Champs |
|---|---|---|
| `weapon_families_loaded` | Info | `families_count`, `schema_version` |
| `weapon_families_mapping_loaded` | Info | `title_slug`, `mapped_weapons_count`, `unmapped_weapons_count`, `coverage_pct` |
| `weapon_families_validation_failed` | Error | `file`, `family_key` ou `weapon_id`, `reason` |
| `weapon_family_lookup_missing` | Warn (rate-limited) | `title_slug`, `weapon_id` |

Même politique de rate-limit que les `field_lookup_missing` (cf. plan multi-titres §8.3).

---

## 9. Risques

| Risque | Probabilité | Impact | Mitigation |
|---|:---:|:---:|---|
| Mapping `weapon_id` → `family_key` controversé (« est-ce que le Stalker Rifle est un DMR ou un Sniper ? ») | Haut | Moyen | Décisions tracées dans le TOML par-titre avec commentaires + revue produit obligatoire |
| Couverture < 60 % d'une seule arme par-titre invalide tout le pipeline | Faible | Moyen | Test CI avec seuil + warning avant échec (cf. §7.2) |
| Ajout d'une famille casse un consommateur tiers | Faible | Faible | Schema versioning N et N-1 (cf. plan multi-titres §10.5) |
| Drift entre TOML global et table DuckDB | Moyen | Moyen | Le TOML est source-of-truth ; reseed automatique au boot si hash diffère |

---

## 10. Annexe — Liste initiale des familles canoniques (~25)

> Couvrant Halo CE → Halo Infinite, basée sur la communauté et le wiki Halopedia.

### Primaires (`tier = "primary"`)

| family_key | EN | FR | damage_type |
|---|---|---|---|
| `assault_rifle` | Assault Rifle | Fusil d'assaut | ballistic |
| `battle_rifle` | Battle Rifle | Fusil de combat | ballistic |
| `dmr` | DMR | DMR | ballistic |
| `carbine` | Carbine | Carabine | ballistic |
| `commando` | Commando Rifle | Fusil Commando | ballistic |
| `pulse_carbine` | Pulse Carbine | Carabine Pulse | plasma |
| `storm_rifle` | Storm Rifle | Fusil Tempête | plasma |
| `bandit` | Bandit | Bandit | ballistic |

### Secondaires (`tier = "secondary"`)

| family_key | EN | FR | damage_type |
|---|---|---|---|
| `magnum` | Magnum | Magnum | ballistic |
| `sidekick` | Sidekick | Sidekick | ballistic |
| `mangler` | Mangler | Mangler | ballistic |
| `plasma_pistol` | Plasma Pistol | Pistolet à plasma | plasma |
| `disruptor` | Disruptor | Disrupteur | plasma |
| `needler` | Needler | Needler | plasma |

### Power (`tier = "power"`)

| family_key | EN | FR | damage_type |
|---|---|---|---|
| `sniper_rifle` | Sniper Rifle | Fusil de précision | ballistic |
| `rocket_launcher` | Rocket Launcher | Lance-roquettes | ballistic |
| `skewer` | Skewer | Skewer | ballistic |
| `shotgun` | Shotgun | Fusil à pompe | ballistic |
| `cindershot` | Cindershot | Cindershot | hardlight |
| `hydra` | Hydra MLRS | Hydra | ballistic |
| `energy_sword` | Energy Sword | Épée à énergie | energy |
| `gravity_hammer` | Gravity Hammer | Marteau gravitationnel | kinetic |

### Mêlée et grenades (`tier = "melee" | "grenade"`)

| family_key | EN | FR | damage_type |
|---|---|---|---|
| `frag_grenade` | Frag Grenade | Grenade à fragmentation | ballistic |
| `plasma_grenade` | Plasma Grenade | Grenade à plasma | plasma |
| `dynamo_grenade` | Dynamo Grenade | Grenade Dynamo | energy |
| `spike_grenade` | Spike Grenade | Grenade à piques | ballistic |

**Total annexe : 26 familles**, à ajuster en revue produit avant Phase 1.

---

## 11. Documents liés

1. `.ai/PLAN_MULTI_TITLE_ADAPTERS_AND_MAPPINGS.md` — plan parent (canonique services + adapters + TOML).
2. `.ai/go_migration_v2/ADR_S44_MULTI_TITLE_NAMESPACE.md` — namespace par titre.
3. `apps/go-api/internal/platform/duckdb/match_view_repo.go` — point d'extension actuel pour `lookupWeaponLabels`.
4. `tools/mappings/CHANGELOG.md` (à créer) — log des bumps de schema_version.
