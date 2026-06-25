# PLAN_WEAPON_TAXONOMY.md — Registre d'armes canonique en BDD (passage principal, multi-titres)

> Rédigé le 2026-06-23. Successeur de `.ai/PLAN_WEAPON_FAMILY_CANONICAL.md` (qui ne traitait QUE la famille, en TOML mémoire).
> Branche cible : feature branch dédiée (chantier conséquent, indépendant des 4 surfaces H5 déjà livrées).
>
> **Vision (user, 2026-06-23)** : un **registre d'armes canonique EN BDD**, qui est le **passage PRINCIPAL** de l'app pour
> connaître/résoudre les armes (pas un TOML « à côté »). Il porte, par arme : les **IDs multiples** (un même weapon a
> plusieurs ids selon la source — ex. Halo Infinite : id de module ET id de chunk de film ; H5 : stock id), le **nom propre
> du jeu** + **traductions** (FR/EN), l'**origine/faction**, la **famille** (rôle cross-titre), le **type de dégât**, et un
> espace **extensible** (`extra` JSON) pour des infos futures (ex. TTK qu'on trouve sur certaines commus — « un jour peut-être »).
> Centralise et remplace les lookups épars actuels (`weapon_labels` metadata + `analysis/weapon_data.go` filmshell + stock ids H5).
>
> **Classifications VÉRIFIÉES (2026-06-23)** contre **halopedia.org + wiki.halo.fr** (4 agents, §6) — sources citées par arme.
>
> **Scope = FONDATION (données + resolver, passage principal). UI = DIFFÉRÉE** : le user n'a pas encore défini la narration /
> le surfaçage. Les surfaces candidates sont listées (§9) mais NON construites ici.

---

## 0. TL;DR

1. **Registre BDD** (3 tables) = source unique + passage principal de la résolution d'arme :
   - `weapons` : 1 ligne par arme par titre (clé canonique + nom + class + family + faction + damage_type + `extra` JSON).
   - `weapon_ids` : N ids par arme (id_kind ∈ module/film_chunk/stock_id/filmshell…) → gère les IDs multiples.
   - `weapon_families` : référentiel des familles cross-titre (clé → libellés FR/EN).
2. **Resolver** = passage principal : `(titre, id_kind, id_value) → weapon_key → {nom, FR, class, family, faction, damage_type, extra}`.
   Remplace progressivement `weapon_labels` + `weapon_data.go` (migration anti-corruption, §8).
3. **5 dimensions** par arme : `class` (poing/épaule/lourde + melee/grenade = MANIPULATION), `role` (automatic/precision/
   sniper/shotgun/sidearm/power/special/melee/grenade = FONCTION DE COMBAT), `family` (identité précise cross-titre),
   `faction` (humaine/covenante/forerunner/parias), `damage_type` (ballistic/plasma/hardlight/spike/shock/explosive…). Extensible.
4. **Append-only + `extra` JSON** → extensible sans migration (TTK, cadence, portée… le jour venu).
5. **Table de classification VÉRIFIÉE** §6 (halopedia + wiki.halo.fr, sourcée par arme).
6. **UI = différée** (narration TBD). On livre la fondation (P1→P5), pas le surfaçage.

---

## 1. Pourquoi un registre BDD (et pas un TOML mémoire)

Le user veut que ce soit le **passage principal**, pas un complément. Raisons :
- **IDs multiples** : un weapon a plusieurs identités selon la source (Infinite : id de module CMS + id de chunk de film ;
  H5 : stock id ; filmshell hash). Aujourd'hui la résolution est éparpillée (`weapon_labels` par titre, `weapon_data.go`
  filmshell Infinite, stock ids H5 ad hoc). Un registre **unifie** : tout id → un `weapon_key` canonique.
- **Extensibilité** : `extra` JSON + append-only permettent d'ajouter TTK, cadence, etc. sans migration ni casser les lecteurs.
- **Requêtable** : « kills par faction/famille » devient un JOIN, pas une reconstruction front.
- **Multi-titres** : la `family` cross-titre (battle_rifle Infinite ↔ battle_rifle H5) vit naturellement comme dimension du registre.

Le TOML reste comme **source de seed** (versionné Git, revue produit), mais la BDD est le **runtime** (passage principal).

---

## 2. Les 5 dimensions

| Dimension | Valeurs | Note |
|---|---|---|
| `class` | sidearm (poing), shoulder (épaule), heavy (lourde), melee, grenade | **MANIPULATION** (prise en main). Demandée par le user (poing/épaule/lourde). |
| `role` | automatic, precision, sniper, shotgun, sidearm, power, special, melee, grenade | **FONCTION DE COMBAT** (« pression vs précision »). Ajoutée 2026-06-23 : donne le regroupement large même quand la `family` est unique. class & role coïncident pour poing/mêlée/grenade, **divergent** pour épaule (precision vs automatic) et lourde (sniper/shotgun/power/special). |
| `family` | identité précise cross-titre (`battle_rifle`, `assault_rifle`, `dmr`, `sniper_rifle`, `rocket_launcher`…) | **valeur multi-titre** : compare la MÊME arme entre jeux (BR75 HINF ↔ Battle Rifle H5). Souvent unique (forerunner H5) → c'est `role` qui regroupe. |
| `faction` | human, covenant, forerunner, banished | **par ORIGINE de conception**, pas le porteur (Épée = covenant même si portée par les Parias). Parias absents de H5. |
| `damage_type` | ballistic/kinetic, plasma, hardlight, spike, shock/voltaic, explosive, incendiary, particle_beam, gravitic… | rempli **quand documenté** ; extensible. |

> Faction — règle tranchée (confirmée halopedia) : `banished` = uniquement les armes de **design Jiralhanae** (Skewer,
> Mangler, Shock Rifle, Disruptor, Ravager + Fuel Rod SPNKr modifié). Les armes covenant portées par les Parias (Plasma
> Pistol, Needler, Épée, Marteau, Pulse/Stalker/Vestige Carbine) restent `covenant`. Cindershot/Heatwave = `forerunner`.

---

## 3. Schéma BDD (registre = passage principal)

DB : `metadata.duckdb` (référentiel global, déjà le foyer des `weapon_labels` / `career_ranks`). Title-scopé par colonne
`title_slug` (le registre est un référentiel partagé multi-titres, pas une donnée per-match).

```sql
-- 1) Registre canonique : 1 ligne par arme par titre (append-only, written_at)
CREATE TABLE weapons (
  weapon_key   VARCHAR NOT NULL,   -- slug canonique stable, ex. "hinf_br75", "h5_battle_rifle"
  title_slug   VARCHAR NOT NULL,   -- titre de CETTE entrée
  name         VARCHAR NOT NULL,   -- nom propre du jeu (EN), ex. "BR75"
  name_fr      VARCHAR,            -- traduction FR officielle (wiki.halo.fr)
  class        VARCHAR,            -- sidearm|shoulder|heavy|melee|grenade (MANIPULATION)
  role         VARCHAR,            -- automatic|precision|sniper|shotgun|sidearm|power|special|melee|grenade (FONCTION DE COMBAT)
  family_key   VARCHAR,            -- FK weapon_families (cross-titre) ; NULL = non regroupé
  faction      VARCHAR,            -- human|covenant|forerunner|banished
  damage_type  VARCHAR,            -- ballistic|plasma|hardlight|spike|shock|explosive|... (nullable)
  manufacturer VARCHAR,            -- fabricant in-lore (traçabilité, ex. "Misriah Armory")
  extra        JSON,               -- EXTENSIBLE : ttk, fire_rate, range... (futur, nullable)
  written_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP
  -- PK technique append-only : (weapon_key) lu via vue weapons_latest (MAX written_at)
);

-- 2) IDs multiples par arme (la remarque clé du user)
CREATE TABLE weapon_ids (
  weapon_key   VARCHAR NOT NULL,   -- FK weapons
  title_slug   VARCHAR NOT NULL,
  id_kind      VARCHAR NOT NULL,   -- module | film_chunk | stock_id | filmshell | cms_asset | ...
  id_value     VARCHAR NOT NULL,   -- string (couvre UBIGINT > 2^63 ET UUID)
  written_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (title_slug, id_kind, id_value)  -- un id résout vers UN weapon_key
);

-- 3) Référentiel des familles cross-titre
CREATE TABLE weapon_families (
  family_key VARCHAR PRIMARY KEY,
  name_en    VARCHAR NOT NULL,
  name_fr    VARCHAR NOT NULL
);
```

- **DÉCISION D'IMPLÉMENTATION (P2, 2026-06-23)** : finalement **PK simple + `INSERT OR IGNORE`** (pas append-only `_latest`),
  comme `weapon_labels.go` / `career_ranks` / `mode_name_tr`. Justification : c'est un **référentiel STATIQUE** seedé au boot
  (zéro writer concurrent, zéro UPDATE per-match) → **hors périmètre du bug ART #23046**, qui ne frappe que les tables d'état
  mutées sous pression concurrente. L'append-only/`_latest` n'apporterait rien ici sauf de la complexité de lecture. PK :
  `weapons` = (title_slug, weapon_key) ; `weapon_ids` = (title_slug, id_kind, id_value) ; `weapon_families` = (family_key).
  Une re-classification rare = migration corrective nommée (comme `fix_super_fiesta_fr_label`). L'extensibilité voulue (TTK…)
  passe par `extra` JSON, **pas** par l'append-only.
- **`id_value` en VARCHAR** : certains weapon_id filmshell dépassent 2^63 (cf. `weapon_data.go`) et les ids H5/CMS sont des
  UUID/strings → on stocke en string (filmshell = décimal via `strconv.FormatUint`), cast au besoin.

---

## 4. Resolver Go (passage principal)

`internal/games/weaponregistry/` (algos + repo lecture ; pas de logique métier titre) :
```go
type Weapon struct {
    Key         string
    TitleSlug   string
    Name        string
    NameFR      string
    Class       string  // sidearm|shoulder|heavy|melee|grenade
    FamilyKey   string
    Faction     string  // human|covenant|forerunner|banished
    DamageType  string
    Manufacturer string
    Extra       map[string]any
}

type Registry interface {
    // PASSAGE PRINCIPAL : résout n'importe quel id (module/film/stock) → l'arme canonique.
    ByID(titleSlug, idKind, idValue string) (Weapon, bool)
    ByKey(weaponKey string) (Weapon, bool)
    Family(familyKey string) (FamilyLabels, bool)
}
```
- Chargé au boot (cache en mémoire depuis `weapons_latest` + `weapon_ids` + `weapon_families`), invalidé au re-seed.
- `ByID` insensible à la casse pour les noms ; exact pour les ids numériques.
- Dégradation : id inconnu → `(_, false)` (le caller affiche le label brut / « Autres », jamais crash).
- Log boot : `weapon_registry_loaded` (weapons_count, ids_count, families_count, unmapped_warn).

---

## 5. (déplacé) — voir §6 pour la table vérifiée.

---

## 6. Table de classification VÉRIFIÉE (halopedia.org + wiki.halo.fr, 2026-06-23)

> Sources : 4 agents de recherche, pages individuelles + catégories des deux wikis. `faction` = ORIGINE de conception.

### 6.1. Halo Infinite

| Arme | class | family | faction | damage_type | name_fr | manufacturer |
|---|---|---|---|---|---|---|
| BR75 | shoulder | battle_rifle | human | ballistic | Fusil de combat | Misriah Armory |
| M392 Bandit | shoulder | dmr | human | ballistic | Bandit | Sevine Arms |
| MA40 AR | shoulder | assault_rifle | human | ballistic | Fusil d'assaut | Misriah Armory |
| MA5K Avenger | shoulder | smg | human | ballistic | Avenger | Misriah Armory |
| VK78 Commando | shoulder | commando | human | ballistic | Commando | Vakara GesmbH |
| S7 Sniper | heavy | sniper_rifle | human | ballistic | Fusil de précision S7 | Misriah Armory |
| CQS48 Bulldog | shoulder | shotgun | human | ballistic | Bulldog | Misriah Armory |
| MLRS-2 Hydra | heavy | hydra | human | explosive | Hydra | Chalybs Defense Solutions |
| M41 SPNKr | heavy | rocket_launcher | human | explosive | Lance-roquettes | Misriah Armory |
| Fuel Rod SPNKr | heavy | rocket_launcher | banished | explosive | Lance-roquettes Fuel Rod | Banished (SPNKr modifié) |
| Mk50 Sidekick | sidearm | magnum | human | ballistic | Sidekick | Emerson Tactical Systems |
| Plasma Pistol | sidearm | plasma_pistol | covenant | plasma | Pistolet à plasma | Iruiru Armory (Rohakadu) |
| Needler | shoulder | needler | covenant | spike (crystalline) | Needler | Lodam Armory |
| Sentinel Beam | heavy | sentinel_beam | forerunner | hardlight | Laser de Sentinelle | Ferrarius Assembler Vats |
| Energy Sword | melee | energy_sword | covenant | plasma | Épée à énergie | Merchants of Qikost |
| Gravity Hammer | melee | gravity_hammer | covenant | gravitic | Marteau antigravité | Sacred Promissory |
| Skewer | heavy | skewer | banished | spike | Empaleur | Flaktura Workshop |
| Cindershot | heavy | cindershot | forerunner | hardlight | Crémator | Ferrarius Assembler Vats |
| Heatwave | heavy | heatwave | forerunner | hardlight | Calcineur | Ferrarius Assembler Vats |
| Ravager | shoulder | ravager | banished | plasma (incendiary) | Ravageur | Veporokk Workshop |
| Shock Rifle | heavy | shock_rifle | banished | shock | Fusil électrique | Sicatt Workshop |
| Disruptor | sidearm | disruptor | banished | shock | Disrupteur | Sicatt Workshop |
| Mangler | sidearm | mangler | banished | spike | Déchiqueteur | Ukala Workshop |
| Pulse Carbine | shoulder | pulse_carbine | covenant | plasma | Carabine à impulsion | Lodam Armory |
| Stalker Rifle | shoulder | stalker_rifle | covenant | plasma | Fusil traqueur | Merchants of Qikost |
| Vestige Carbine | shoulder | carbine | covenant | plasma | Vestige Carbine | (restaurateurs Sangheili) |
| Frag Grenade | grenade | frag_grenade | human | explosive | Grenade à fragmentation | Misriah Armory |
| Plasma Grenade | grenade | plasma_grenade | covenant | plasma | Grenade à plasma | — |
| Dynamo Grenade | grenade | dynamo_grenade | banished | shock | Grenade Dynamo | — |

> Exclus (mécaniques mêlée / médailles, pas des armes) : Assassination, Back Smack, Melee, Pummel, Ninja, Stick(y),
> Sandwich/Mythic Sandwich, Pancake, Kong, Quigley, Mutilator, Grenadier. **« Diminisher of Hope » = Gravity Axe (mêlée
> perso d'Escharum), PAS une variante Skewer** (corrigé via halopedia).

### 6.2. Halo 5: Guardians

| Arme | class | family | faction | damage_type | name_fr | manufacturer |
|---|---|---|---|---|---|---|
| Assault Rifle (MA5D) | shoulder | assault_rifle | human | ballistic | Fusil d'assaut | Misriah Armory |
| Battle Rifle (BR55HB) | shoulder | battle_rifle | human | ballistic | Fusil de combat | Misriah Armory |
| DMR (M395B) | shoulder | dmr | human | ballistic | DMR | Misriah Armory |
| Magnum (M6H2) | sidearm | magnum | human | ballistic | Magnum | Misriah Armory |
| SMG (M20) | shoulder | smg | human | ballistic | Mitraillette | Misriah Armory |
| Shotgun (M45D) | shoulder | shotgun | human | ballistic | Fusil à pompe | Misriah Armory |
| Sniper Rifle (SRS99-S5) | heavy | sniper_rifle | human | ballistic | Fusil de précision | Misriah Armory |
| Rocket Launcher (M41D) | heavy | rocket_launcher | human | explosive | Lance-roquettes | Misriah Armory |
| Grenade Launcher (M319) | heavy | grenade_launcher | human | explosive | Lance-grenades | Misriah Armory |
| Railgun (ARC-920) | heavy | railgun | human | kinetic+explosive | Railgun | Acheron Security |
| SAW (M739) | shoulder | saw | human | ballistic | SAW | Misriah Armory |
| Hydra (MLRS-1) | heavy | hydra | human | explosive | Hydra | Chalybs Defense Solutions |
| Spartan Laser (M6/E) | heavy | spartan_laser | human | hardlight | Laser Spartan | Misriah Armory |
| Carbine (Mosa) | shoulder | carbine | covenant | spike (radioactive) | Carabine | Iruiru Armory |
| Energy Sword | melee | energy_sword | covenant | plasma | Épée à énergie | Merchants of Qikost |
| Plasma Pistol | sidearm | plasma_pistol | covenant | plasma | Pistolet à plasma | (legacy covenant) |
| Brute Plasma Rifle | shoulder | plasma_rifle | covenant | plasma | Fusil à plasma brute | Sacred Promissory |
| Fuel Rod Cannon | heavy | fuel_rod | covenant | explosive | Canon à combustible | Forge of Raansek |
| Storm Rifle | shoulder | storm_rifle | covenant | plasma | Fusil Storm | Lodam Armory |
| Beam Rifle | heavy | beam_rifle | covenant | particle_beam | Fusil de sniper covenant | Merchants of Qikost |
| Plasma Caster | heavy | plasma_caster | covenant | plasma | Canon plasma | Merchants of Qikost |
| Needler | shoulder | needler | covenant | spike (crystalline) | Needler | Lodam Armory |
| Gravity Hammer | melee | gravity_hammer | covenant | gravitic | Marteau antigravité | Sepulo'tez Workshop |
| Light Rifle (Z-250) | shoulder | light_rifle | forerunner | hardlight | Fusil léger | Ferrarius Assembler Vats |
| Binary Rifle (Z-750) | heavy | binary_rifle | forerunner | particle_beam | Fusil binaire | Ferrarius Assembler Vats |
| Boltshot (Z-110) | sidearm | boltshot | forerunner | hardlight | Pistolet à particules | Ferrarius Assembler Vats |
| Incineration Cannon (Z-390) | heavy | incineration_cannon | forerunner | hardlight (antiparticle) | Canon incendiaire | Ferrarius Assembler Vats |
| Sentinel Beam | heavy | sentinel_beam | forerunner | particle_beam | Laser de Sentinelle | Ferrarius Assembler Vats |
| Suppressor (Z-130) | shoulder | suppressor | forerunner | hardlight | Éradicateur | Ferrarius Assembler Vats |
| Scattershot (Z-180) | shoulder | scattershot | forerunner | hardlight | Répercuteur | Ferrarius Assembler Vats |

### 6.3. Familles cross-titre (la valeur multi-titre)

Présentes DANS LES DEUX titres (comparaison directe) : `battle_rifle`, `assault_rifle`, `dmr`, `sniper_rifle`,
`rocket_launcher`, `magnum`, `shotgun`, `smg`, `hydra`, `plasma_pistol`, `needler`, `energy_sword`, `gravity_hammer`,
`sentinel_beam`, `carbine` (Vestige HINF ↔ Carbine H5 — à confirmer côté produit). Distinctes volontairement :
`commando` (HINF) ≠ `light_rifle` (H5) — marksman des deux factions, ne PAS fusionner.

Mono-titre : HINF parias (`skewer`, `mangler`, `shock_rifle`, `disruptor`, `ravager`, `cindershot`, `heatwave`,
`pulse_carbine`, `stalker_rifle`/dmr-covenant) ; H5 (`saw`, `railgun`, `spartan_laser`, `grenade_launcher`, `storm_rifle`,
`plasma_rifle`, `fuel_rod`, `beam_rifle`, `plasma_caster`, `boltshot`, `binary_rifle`, `incineration_cannon`,
`suppressor`, `scattershot`, `light_rifle`).

---

## 7. Source des IDs (le `weapon_ids`)

| Titre | id_kind | Source dans le code |
|---|---|---|
| halo_infinite | `filmshell` | `analysis/weapon_data.go` `WeaponIDToName` (uint64 filmshell hash) |
| halo_infinite | `cms_asset` / `module` | `weapon_labels` metadata (weapon_id UBIGINT) + assets CMS |
| halo_infinite | `film_chunk` | id de chunk de film (à extraire du décodeur film si distinct du filmshell) |
| halo_5 | `stock_id` | `KillerWeaponStockId` (events H5) + metadata officiel (h5-metadata-fetch) |

> P2 du plan = **réconcilier** ces ids vers les `weapon_key` (un weapon = N ids). C'est le cœur « passage principal » :
> aujourd'hui chaque source résout indépendamment ; le registre unifie.

---

## 8. Migration « passage principal » (remplacer les lookups épars)

1. Le registre seedé, on bascule les **points de lecture** weapon → registre :
   - `match_view` weapon donut, synthesis/timeseries/squad top-weapons : résolvent `id → weapon_key → name/family/faction`.
   - `analysis/weapon_data.go::ResolveWeaponFromFilmshell` → délègue au registre (puis le map hardcodé devient un seed).
   - `weapon_labels` metadata → le registre devient la source ; `weapon_labels` peut rester en vue de compat le temps de la bascule.
2. **Anti-corruption** : bascule progressive derrière le resolver (un seul point de vérité), pas un big-bang des callers.
3. Dégradation : id non mappé → label brut (jamais de perte d'affichage).

---

## 9. Surfaçage UI — DIFFÉRÉ (narration TBD)

**Ne PAS construire.** Candidats quand la narration sera cadrée : donut « kills par faction », « kills par classe »,
regroupement du breakdown par-arme **par famille** + **comparaison cross-titre** (« 2× plus de kills `sniper_rifle` que la
moyenne H5 »), filtre Explorer par famille/faction. Honnêteté : armes non mappées → « Autres » explicite.

---

## 10. Tests

- Loader registre (DuckDB :memory:) : seed + `ByID`/`ByKey`/`Family` ; id inconnu → false ; casse insensible sur noms.
- Cohérence CI : tout `family_key` cité ∈ `weapon_families` ; `class`/`faction`/`damage_type` ∈ enums ; aucun `id_value`
  mappé à 2 `weapon_key` (unicité). Rapport couverture `mapped/total` par titre.
- Append-only : re-seed = nouvelle génération, `weapons_latest` voit la dernière ; pas d'UPDATE indexé (garde anti-ART).
- Migration : un caller basculé sur le registre donne le MÊME label qu'avant pour les armes connues (golden parity).

---

## 11. Phases

| Phase | Contenu | Done si |
|---|---|---|
| **P0 — Plan** (ce doc) | Schéma BDD + table vérifiée §6 + cadrage UI-différé | validé user |
| **P1 — Vérif data** | halopedia + wiki.halo.fr (FAIT 2026-06-23, §6 sourcée) | table figée |
| **P2 — Schéma + seed** | 3 tables (migration metadata `add_weapon_registry`, PK simple) + seed Go depuis §6 (42 familles + 59 armes + 36 filmshell ids Infinite) | **FAIT** (tables seedées, tests verts `weapon_registry_test.go`) |
| **P2bis — ids H5** | réconciliation `stock_id` H5 (catalogue officiel `weapon_labels` metadata H5) dans `weapon_ids` | **FAIT** (35 stock_ids + variantes, tests verts) |
| **P3 — Resolver + tests** | package `weaponregistry` + repo lecture + tests | **FAIT** (`ByID`/`ByKey`/`Family`, tests verts, log boot weapon_registry_loaded) |
| **P4 — Enrichir les lecteurs** | les donuts/« outils de destruction » marchent DÉJÀ pour LES DEUX titres (weapon_kills + résolution de nom via `weapon_labels` par titre : filmshell HINF, stock_id H5). P4 = leur AJOUTER les nouvelles dims (class/role/family/faction) du registre + unifier la résolution (fallback `weapon_labels` pour le catalogue complet : grenades/Spartan/véhicules hors seed curé) | **DIFFÉRÉ** : le surfaçage des nouvelles dims dépend de la narration UI (non décidée). Le registre **enrichit**, il ne remplace pas la résolution de nom existante. |
| **P5 — (DIFFÉRÉ) UI** | donut/comparaison selon narration user | hors-scope tant que narration TBD |

**Livré = P0→P3 + P2bis** (fondation + passage principal seedé + resolver, sans UI). **P4 + P5 = différés, pilotés par la narration UI** (décision user 2026-06-23).

---

## 12. Documents liés
- `.ai/PLAN_WEAPON_FAMILY_CANONICAL.md` — prédécesseur (famille seule, TOML).
- `internal/analysis/weapon_data.go` — roster + ids filmshell Infinite (source `weapon_ids`).
- `config/titles/halo_5/mappings/asset_labels_fr.toml` + `cmd/h5-metadata-fetch` — labels + stock ids H5.
- ADR 0026 (append-only) ; ADR 0011 (canonical vs semantic) ; ADR 0008 (metadata.duckdb référentiels).
- Sources lore : halopedia.org + wiki.halo.fr (citées par arme dans l'historique de recherche 2026-06-23).
