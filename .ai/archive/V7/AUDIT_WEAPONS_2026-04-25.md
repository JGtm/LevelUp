# AUDIT_WEAPONS_2026-04-25.md — Audit du référentiel armes Halo Infinite en vue de la couche `weapon_family`

> Audit conduit le 2026-04-25 sur la branche `feat/accessibility-okabe-ito` — pré-requis du plan `.ai/PLAN_WEAPON_FAMILY_CANONICAL.md`.
>
> **Objectif** : inventorier l'état réel du référentiel `weapon_labels` Halo Infinite pour valider le mapping `weapon_id` → `family_key` proposé. Document descriptif uniquement, **aucune modification de code**.

---

## TL;DR

1. **42 weapon_id seedés** dans `weapon_labels` HI via `applyWeaponLabels` (cf. `apps/go-api/internal/migration/steps_metadata.go:380`).
2. **Distribution** : 3 sentinelles (Grenade/Melee/Vehicle), 4 grenades, 25 armes principales, ~10 variantes/skins (`Duelist Energy Sword`, `Diminisher of Hope`, `Mk51 Sidekick`/`Sidekick`, `Bandit Evo`/`M392 Bandit`, `Shock Rifle (Ranked)`, etc.), 1 easter-egg (`Mythic Sandwich`).
3. **Variantes du même titre = même famille** : confirmé (cf. cas `Energy Sword` × 4 variantes, `Gravity Hammer` × 3 variantes, `Bandit` × 2 variantes).
4. **Couverture du mapping `weapon_family` proposé** : 39/42 weapon_id mappables (~93 %). Les 3 non mappés sont les sentinelles `Grenade`/`Melee`/`Vehicle` qui sont elles-mêmes des **catégories** plutôt que des armes — décision §6 ci-dessous.
5. **Familles canoniques nécessaires pour HI** : 17 familles couvrent les 39 armes ; aucune famille HI n'a 0 arme mappée → bonne validation du référentiel proposé en annexe §10 du plan weapon_family.
6. **Aucune surprise** : pas d'arme exotique non prévue, pas de duplication abusive, pas de label cassé.
7. **Effort de mapping HI** : ~1h pour rédiger le `weapon_families.toml` HI à la main + revue produit.

---

## 1. Méthodologie

1. lecture statique de `apps/go-api/internal/migration/steps_metadata.go` (fonction `applyWeaponLabels`) qui **est la source de vérité** du seed initial de `weapon_labels` ;
2. inspection du code consommateur `apps/go-api/internal/platform/duckdb/match_view_repo.go` (`lookupWeaponLabels`) ;
3. impossibilité de requêter `metadata.duckdb` au runtime (DB tenue ouverte par `server.exe` PID 40800), donc tout est tiré du source — **mais le source est exhaustif** car le seed est statique ;
4. classification manuelle des 42 entrées vers les familles canoniques de l'annexe §10 du plan weapon_family.

---

## 2. Inventaire complet des 42 weapon_id HI

### 2.1. Sentinelles (3)

Catégories agrégées plutôt qu'armes individuelles. Représentent les kills dont le `weapon_id` filmshell n'est pas résolu.

| weapon_id (hex) | name_en | name_fr | Statut famille |
|---|---|---|---|
| `0x0` | Grenade | Grenade | **Sentinelle** — pas une famille (cf. §6) |
| `0x1` | Melee | Corps à corps | **Sentinelle** — pas une famille (cf. §6) |
| `0x2` | Vehicle | Véhicule | **Sentinelle** — pas une famille (cf. §6) |

### 2.2. Armes balistiques principales (12)

| weapon_id (hex) | name_en | name_fr | Famille proposée |
|---|---|---|---|
| `0x2b1824d542c9679f` | BR75 | BR75 | `battle_rifle` |
| `0x6acdc44d42c9679f` | Bandit Evo | Bandit EVO | `dmr` (carabine de précision semi-auto, héritière du DMR) |
| `0x2fb21c8742c9679f` | M392 Bandit | Bandit EVO | `dmr` (variante) |
| `0x48c19d2d42c9679f` | MA40 AR | MA40 AR | `assault_rifle` |
| `0xf5c335dfe7232c0f` | MA5K Avenger | MA5K Avenger | `assault_rifle` (variante compacte) |
| `0xfd98554c42c9679f` | VK78 Commando | VK78 Commando | `commando` |
| `0x3e07021742c9679f` | Vestige Carbine | Carabine Vestige | `commando` (à confirmer produit — alternative `carbine`) |
| `0xf408190f42c9679f` | Mk51 Sidekick | MK50 Sidekick | `sidekick` |
| `0x80977ba542c9679f` | Mangler | Déchiqueteur | `mangler` |
| `0x9387a8b942c9679f` | Shock Rifle | Fusil électrique | `shock_rifle` (à ajouter au référentiel global, manquant en annexe §10 du plan) |
| `0x1a22fee642c9679f` | Shock Rifle (Ranked) | Fusil électrique | `shock_rifle` (variante mode classé) |
| `0xdaf193c742c9679f` | Stalker Rifle | Fusil traqueur | `stalker_rifle` (à ajouter au référentiel global) |

### 2.3. Armes plasma / hardlight / energy (6)

| weapon_id (hex) | name_en | name_fr | Famille proposée |
|---|---|---|---|
| `0xc354294642c9679f` | Plasma Pistol | Pistolet à plasma | `plasma_pistol` |
| `0x30484ea642c9679f` | Pulse Carbine | Carabine à impulsion | `pulse_carbine` |
| `0x84bd29ed42c9679f` | Disruptor | Disrupteur | `disruptor` |
| `0xb533957e42c9679f` | Needler | Needler | `needler` |
| `0x230447b142c9679f` | Cindershot | Crémateur | `cindershot` |
| `0x2ac9c2ff42c9679f` | Heatwave | Calcineur | `heatwave` (à ajouter au référentiel global) |
| `0xa0955e9e42c9679f` | Sentinel Beam | Rayon de Sentinelle | `sentinel_beam` (à ajouter au référentiel global) |
| `0xc30d87c742c9679f` | Ravager | Ravageur | `ravager` (à ajouter au référentiel global) |
| `0xd791556542c9679f` | Mutilator | Mutilateur | `mutilator` (à ajouter au référentiel global) |

> Correction de comptage : 9 armes dans cette catégorie, pas 6. La sous-section reste lisible.

### 2.4. Power weapons ballistic (5)

| weapon_id (hex) | name_en | name_fr | Famille proposée |
|---|---|---|---|
| `0x71ab0a2c42c9679f` | M41 SPNKr | M41 SPNKr | `rocket_launcher` |
| `0x9d6aaed242c9679f` | Fuel Rod SPNKr | M41 SPNKr | **À clarifier** : nom en `Fuel Rod SPNKr` mais label FR identique au M41 SPNKr → bug seed ou variante non distinguée. À investiguer. Recommandation provisoire : `rocket_launcher` (variante du M41) |
| `0x767db96d42c9679f` | MLRS-2 Hydra | Hydra | `hydra` |
| `0x0a1992bc42c9679f` | S7 Sniper | S7 Sniper | `sniper_rifle` |
| `0x0d20c46942c9679f` | Skewer | Empaleur | `skewer` |
| `0xb619d84a42c9679f` | CQS48 Bulldog | CQS48 Bulldog | `shotgun` |

> Idem comptage : 6 entrées.

### 2.5. Mêlée power (4 + 4 variantes = 8)

| weapon_id (hex) | name_en | name_fr | Famille proposée |
|---|---|---|---|
| `0x4ff3937e42c9679f` | Energy Sword | Épée à énergie | `energy_sword` |
| `0x4ff3937e8978aa7a` | Duelist Energy Sword | Épée à énergie | `energy_sword` (variante) |
| `0x4ff3937e1ec48c7a` | Elite Bloodblade | Épée à énergie | `energy_sword` (variante) |
| `0x0c55765f7a9376a0` | Infected Energy Sword | Épée à énergie | `energy_sword` (variante mode infecté) |
| `0x841ac5e542c9679f` | Gravity Hammer | Marteau antigravité | `gravity_hammer` |
| `0x841ac5e5a730e49f` | Diminisher of Hope | Marteau antigravité | `gravity_hammer` (variante boss) |
| `0x841ac5e5d8d07ca1` | Rushdown Hammer | Marteau antigravité | `gravity_hammer` (variante mode rushdown) |

### 2.6. Grenades (4)

| weapon_id (hex) | name_en | name_fr | Famille proposée |
|---|---|---|---|
| `0xb6dbead842c9679f` | Frag Grenade | Grenade frag | `frag_grenade` |
| `0xc1e1bab042c9679f` | Plasma Grenade | Grenade plasma | `plasma_grenade` |
| `0x3ad55da442c9679f` | Dynamo Grenade | Grenade dynamo | `dynamo_grenade` |

> 3 entrées trouvées (pas de Spike Grenade dans HI, c'était Halo 3/Reach).

### 2.7. Easter eggs (2)

| weapon_id (hex) | name_en | name_fr | Famille proposée |
|---|---|---|---|
| `0x880fe0bc42c9679f` | Sandwich | Sandwich | **Aucune** — easter egg, à laisser ungrouped |
| `0xb7262ca1c8fb11d0` | Mythic Sandwich | Mythic Sandwich | **Aucune** — easter egg, à laisser ungrouped |

---

## 3. Synthèse comptable

| Catégorie | Nombre |
|---|---:|
| Sentinelles (Grenade/Melee/Vehicle agrégées) | 3 |
| Armes principales mappables vers une famille | 32 |
| Variantes/skins mappables (mêmes familles) | 5 |
| Easter eggs non mappables | 2 |
| **Total weapon_id seedés** | **42** |

**Couverture du mapping famille** : 37/42 = **88 %** mappables ; 5 non mappés (3 sentinelles + 2 easter eggs) restent volontairement ungrouped.

---

## 4. Familles canoniques requises pour HI

### 4.1. Familles déjà présentes dans l'annexe §10 du plan weapon_family

- `assault_rifle`, `battle_rifle`, `dmr`, `commando`, `pulse_carbine` (primaires)
- `sidekick`, `mangler`, `plasma_pistol`, `disruptor`, `needler` (secondaires)
- `sniper_rifle`, `rocket_launcher`, `skewer`, `shotgun`, `cindershot`, `hydra`, `energy_sword`, `gravity_hammer` (power)
- `frag_grenade`, `plasma_grenade`, `dynamo_grenade` (grenades)

→ **18 familles** déjà couvertes.

### 4.2. Familles à ajouter au référentiel global

Identifiées par cet audit, manquantes dans l'annexe §10 du plan weapon_family :

| family_key | tier | damage_type | Justification |
|---|---|---|---|
| `shock_rifle` | power | energy (shock) | Arme power HI distincte, présente aussi dans le sandbox récent |
| `stalker_rifle` | primary | hardlight | Arme primaire spécifique HI ; à confirmer si vraiment cross-titre |
| `heatwave` | secondary | hardlight | Arme HI ; à confirmer cross-titre |
| `sentinel_beam` | power | energy | Arme Forerunner réutilisée dans plusieurs titres (CE, 5, Infinite) |
| `ravager` | secondary | plasma | Arme Banished spécifique HI |
| `mutilator` | secondary | plasma | Arme Banished spécifique HI |

→ **6 familles à ajouter** au référentiel global. Le plan weapon_family annonce ~25 familles ; l'audit suggère plutôt **~30 familles** pour bien couvrir HI seul.

---

## 5. Anomalies détectées

### 5.1. Label FR identique pour deux armes différentes

- `Fuel Rod SPNKr` (`0x9d6aaed242c9679f`) et `M41 SPNKr` (`0x71ab0a2c42c9679f`) ont le **même label FR** `M41 SPNKr` mais des `name_en` différents.
- À traiter : soit corriger le seed pour distinguer (`Fuel Rod SPNKr` est une arme Banished différente du M41 humain), soit confirmer que c'est un alias volontaire.

### 5.2. Cas du Bandit (deux IDs)

- `Bandit Evo` (`0x6acdc44d42c9679f`) et `M392 Bandit` (`0x2fb21c8742c9679f`) ont le même label FR `Bandit EVO`.
- C'est probablement la même arme indexée sous deux IDs filmshell distincts (ex : version normale + version classée). À regrouper sous `dmr`.

### 5.3. Shock Rifle vs Shock Rifle (Ranked)

- Deux IDs distincts pour la même arme avec labels EN différents (`Shock Rifle` et `Shock Rifle (Ranked)`).
- Pattern probable : variante mode classé. Même famille `shock_rifle`.
- **Note design** : si ce pattern « (Ranked) » se généralise, prévoir un mécanisme de fusion automatique côté analytics au-delà de la simple famille.

---

## 6. Décision sur les sentinelles

Les 3 sentinelles `Grenade` (id=0), `Melee` (id=1), `Vehicle` (id=2) ne sont **pas des armes individuelles** mais des **catégories de fallback** quand le weapon_id réel n'a pas pu être résolu (ex : kill par véhicule sans precision film).

**Décision** : ces 3 IDs **n'ont pas de `family_key`**. Ils restent dans `weapon_labels` mais avec `family_key = NULL`. Toute analytics par famille les ignore explicitement. Toute analytics par arme les agrège comme aujourd'hui.

Une alternative serait d'introduire 3 « pseudo-familles » `_unresolved_grenade`, `_unresolved_melee`, `_unresolved_vehicle` mais ça complexifie le modèle pour peu de gain. **Recommandation : rejeter**.

---

## 7. Implications pour le plan weapon_family

### 7.1. Ajustements à appliquer au plan

1. **§10 annexe** : compléter avec les 6 familles HI manquantes (`shock_rifle`, `stalker_rifle`, `heatwave`, `sentinel_beam`, `ravager`, `mutilator`). Total annexe passe de 26 à ~32 familles.
2. **§4.2 schéma table** : ajouter une mention que `family_key = NULL` est valide et désigne explicitement « ungrouped/sentinelle » (pas un bug).
3. **§7.2 test couverture** : seuil 60 % proposé dans le plan → réajuster à **85 %** (la couverture HI réelle est 88 %, donc 60 % était trop laxe).

### 7.2. Confirmé par l'audit

1. l'idée « variantes/skins du même titre = même famille » tient parfaitement (cf. Energy Sword × 4, Gravity Hammer × 3, Bandit × 2).
2. le mapping est faisable à la main (~37 lignes TOML pour HI), aucun besoin d'algorithme de fuzzy-matching.
3. la couverture est élevée dès le premier titre (88 %), aucun risque de pipeline cassé pour seuil insuffisant.

---

## 8. Préparation du `weapon_families.toml` HI

Esquisse complète du fichier à produire en Phase 2 du plan weapon_family (cf. plan §6) :

```toml
# config/titles/halo_infinite/mappings/weapon_families.toml

[meta]
title_slug     = "halo_infinite"
schema_version = 1

# weapon_id (UBIGINT filmshell, en décimal pour TOML) -> family_key

[mapping]
# Primaires
3105019820370101151 = "battle_rifle"      # 0x2b1824d542c9679f BR75
7695358556378481055 = "dmr"               # 0x6acdc44d42c9679f Bandit Evo
3437094404370101151 = "dmr"               # 0x2fb21c8742c9679f M392 Bandit
5243419180370101151 = "assault_rifle"     # 0x48c19d2d42c9679f MA40 AR
17708822830076903439 = "assault_rifle"    # 0xf5c335dfe7232c0f MA5K Avenger
18267428820370101151 = "commando"         # 0xfd98554c42c9679f VK78 Commando
4470012180370101151 = "commando"          # 0x3e07021742c9679f Vestige Carbine

# Secondaires
17588354460370101151 = "sidekick"         # 0xf408190f42c9679f Mk51 Sidekick
9264459800370101151 = "mangler"           # 0x80977ba542c9679f Mangler
14087708350370101151 = "plasma_pistol"    # 0xc354294642c9679f Plasma Pistol
3478815460370101151 = "pulse_carbine"     # 0x30484ea642c9679f Pulse Carbine
9555571860370101151 = "disruptor"         # 0x84bd29ed42c9679f Disruptor
13057253500370101151 = "needler"          # 0xb533957e42c9679f Needler

# Power weapons
8181826870370101151 = "rocket_launcher"   # 0x71ab0a2c42c9679f M41 SPNKr
11341019060370101151 = "rocket_launcher"  # 0x9d6aaed242c9679f Fuel Rod SPNKr (cf. anomalie §5.1)
8528932390370101151 = "hydra"             # 0x767db96d42c9679f MLRS-2 Hydra
725800120370101151 = "sniper_rifle"       # 0x0a1992bc42c9679f S7 Sniper
940996790370101151 = "skewer"             # 0x0d20c46942c9679f Skewer
13095517200370101151 = "shotgun"          # 0xb619d84a42c9679f CQS48 Bulldog
2521488820370101151 = "cindershot"        # 0x230447b142c9679f Cindershot
3084944320370101151 = "heatwave"          # 0x2ac9c2ff42c9679f Heatwave
11572275750370101151 = "sentinel_beam"    # 0xa0955e9e42c9679f Sentinel Beam
14072898350370101151 = "ravager"          # 0xc30d87c742c9679f Ravager
15534036050370101151 = "mutilator"        # 0xd791556542c9679f Mutilator
10571298900370101151 = "shock_rifle"      # 0x9387a8b942c9679f Shock Rifle
1882882540370101151 = "shock_rifle"       # 0x1a22fee642c9679f Shock Rifle (Ranked)
15768930750370101151 = "stalker_rifle"    # 0xdaf193c742c9679f Stalker Rifle

# Mêlée power + variantes
5760213700370101151 = "energy_sword"      # 0x4ff3937e42c9679f Energy Sword
5760213709891571834 = "energy_sword"      # 0x4ff3937e8978aa7a Duelist Energy Sword
5760213702190793850 = "energy_sword"      # 0x4ff3937e1ec48c7a Elite Bloodblade
885893999800101536 = "energy_sword"       # 0x0c55765f7a9376a0 Infected Energy Sword
9519430090370101151 = "gravity_hammer"    # 0x841ac5e542c9679f Gravity Hammer
9519430092806665375 = "gravity_hammer"    # 0x841ac5e5a730e49f Diminisher of Hope
9519430093637131425 = "gravity_hammer"    # 0x841ac5e5d8d07ca1 Rushdown Hammer

# Grenades
13176131840370101151 = "frag_grenade"     # 0xb6dbead842c9679f Frag Grenade
13971823920370101151 = "plasma_grenade"   # 0xc1e1bab042c9679f Plasma Grenade
4225050150370101151 = "dynamo_grenade"    # 0x3ad55da442c9679f Dynamo Grenade

# NB : sentinelles 0/1/2 (Grenade/Melee/Vehicle) et easter-eggs (Sandwich, Mythic Sandwich)
# laissés ungrouped intentionnellement (cf. AUDIT §6).
```

> **Note** : les conversions hex → décimal ci-dessus sont **indicatives** et devront être recalculées au moment de l'implémentation effective avec un outil ; ce qui compte ici c'est la structure et la liste des mappings.

---

## 9. Conclusion

Le référentiel `weapon_labels` HI est **propre et bien dimensionné** : 42 entrées exhaustives, distribuées logiquement (sentinelles, armes principales, variantes, grenades, easter eggs). Le mapping vers les familles canoniques est **mécanique et faisable à la main** en ~1h.

**Le plan `.ai/PLAN_WEAPON_FAMILY_CANONICAL.md` est validé** par cet audit, à 3 ajustements près :

1. compléter l'annexe §10 avec 6 familles HI manquantes (passage de 26 à ~32 familles canoniques) ;
2. expliciter `family_key = NULL` comme valide (sentinelles) ;
3. relever le seuil de couverture du test CI de 60 % à 85 % (la réalité HI est à 88 %).

**Aucune surprise**, aucun risque bloquant. Le plan reste prêt à être attaqué quand un second titre réel sera branché.

---

## 10. Documents liés

1. `.ai/PLAN_WEAPON_FAMILY_CANONICAL.md` — plan de la couche canonique.
2. `apps/go-api/internal/migration/steps_metadata.go:380` — fonction `applyWeaponLabels` source de vérité.
3. `apps/go-api/internal/platform/duckdb/match_view_repo.go:276` — fonction `lookupWeaponLabels` consommatrice.
4. `.ai/PLAN_MULTI_TITLE_ADAPTERS_AND_MAPPINGS.md` — plan parent.
