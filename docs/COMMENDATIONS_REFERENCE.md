# LevelUp Commendations — Full Reference

French version: [FR/CITATIONS_REFERENCE.md](FR/CITATIONS_REFERENCE.md)

This document is the **human-readable mirror** of the authoritative Go seed `internal/ops/seed_citation_data.go` (`defaultCitationMappings`). That function is the single source of truth — if this table and the seed ever disagree, the seed wins. The seed defines **88 commendations** (the `enabled = false` ones are listed but not computed).

> Stack note: this used to be rebuilt from `scripts/populate_citation_mappings.py`; that Python script and `src/ui/commendations.py` have been removed. To rebuild `citation_mappings`, run `levelup seed citation-mappings`.

---

## Storage

| Item | Location | Purpose |
|------|----------|---------|
| Rule registry | `metadata.duckdb` → `citation_mappings` | 1 row = 1 commendation. |
| Computed results | player `stats.duckdb` → `match_citations` (view `match_citations_latest`) | Value per match × commendation (append-only). |
| Raw PvE stats | `shared_pve.duckdb` → `pve_match_stats` | Enemy-type kills per match (`pve_stat`). |
| Weapon kills | `shared` `v_weapon_kills` + `metadata.weapon_labels` | `weapon_stat` per-weapon kills (`weapon_kills:<NameEN>`). |
| Images | `static/commendations/halo_5_guardians/` and `static/commendations/halo_infinite/` | PNG referenced by `image_path`. |

## Rebuild

```bash
levelup seed citation-mappings      # writes citation_mappings into metadata.duckdb
```

### `citation_mappings` schema

See `internal/ops/seed.go` (`SeedCitationMappings`):

```sql
CREATE TABLE citation_mappings (
    citation_name_norm    VARCHAR PRIMARY KEY,
    citation_name_display VARCHAR NOT NULL,
    mapping_type          VARCHAR NOT NULL DEFAULT 'medal',  -- medal|stat|pve_stat|weapon_stat|award|custom|composite
    medal_id              UBIGINT,
    medal_ids             VARCHAR,
    stat_name             VARCHAR,
    award_name            VARCHAR,
    award_category        VARCHAR,
    custom_function       VARCHAR,
    composite_children    VARCHAR,
    enabled               BOOLEAN NOT NULL DEFAULT TRUE,
    image_path            VARCHAR,
    category              VARCHAR,
    description           VARCHAR,
    tier_targets          VARCHAR,
    subcategory           VARCHAR
);
```

---

## Norm-keys, not normalization

There is **no runtime normalization function** any more (the old Python `_normalize_name` / NFKD is gone). Each commendation's `citation_name_norm` is a **fixed ASCII key** hard-coded in the Go seed (e.g. `charge`, `splatter`, `flag_defender`). Display names carry the accents (`À la charge`, `Écrasement`); the norm-key is the stable join key into `match_citations`.

---

## Full inventory (from the seed)

Tier columns are the `tier_targets` CSV; **Master** is the last (largest) tier.

### PvP — Game mode (11)

| Norm | Display | Type | Source | Tiers |
|------|---------|------|--------|-------|
| `charge` | À la charge | award | `zone_captured` (objective) | 10,20,30,50,**100** |
| `forced_annexation` | Annexion forcée | custom | `compute_annexion_forcee` | 3,6,9,15,**30** |
| `assistant` | Assistant | stat | `assists` | 25,50,75,125,**250** |
| `bulldozer` | Bulldozer | custom | `compute_bulldozer` | 3,6,9,15,**30** |
| `flag_defender` | Défenseur du drapeau | award | `carrier_killed` (objective) — **disabled** (I7) | 10,20,30,50,**100** |
| `got_you` | Je te tiens ! | award | `flag_returned` (objective) | 10,20,30,50,**100** |
| `stakeholder` | Partie prenante | award | `zone_secured` (objective) | 10,20,30,50,**100** |
| `flag_carrier_hunter` | Sus au porteur du drapeau | award | `carrier_killed` (objective) | 10,20,30,50,**100** |
| `flag_victory` | Victoire au drapeau | custom | `compute_wins_ctf` | 5,10,15,25,**50** |
| `slayer_victory` | Victoire en assassin | custom | `compute_wins_slayer` | 5,10,15,25,**50** |
| `strongholds_victory` | Victoire en bases | custom | `compute_wins_strongholds` | 5,10,15,25,**50** |

### PvP — Vehicle / Grenade (4)

| Norm | Display | Type | Source | Tiers | Subcat |
|------|---------|------|--------|-------|--------|
| `splatter` | Écrasement | medal | `221693153` | 10,20,30,50,**100** | Général |
| `driver` | Pilote | medal | `2926348688` | 10,20,30,50,**100** | Général |
| `frag_grenade` | Grenade à fragmentation | medal | `2648272972` | 5,10,15,25,**50** | Grenade |
| `plasma_grenade` | Grenade à plasma | medal | `3655682764` | 2,4,6,10,**20** | Grenade |

### PvP — Multiplayer (10)

| Norm | Display | Type | Source | Tiers |
|------|---------|------|--------|-------|
| `assassin` | Assassin | medal | `548533137` | 5,10,15,25,**50** |
| `spartan_carnage` | Carnage de Spartans | medal | `2780740615,4261842076,418532952,1486797009,710323196,1720896992,2567026752,2875941471` | 3,6,9,15,**30** |
| `close_combat` | Combat rapproché | stat | `melee_kills` | 10,20,30,50,**100** |
| `opportunist` | Combattant opportuniste | medal | `622331684,2063152177,4261842076,2137071619,1486797009,1430343434,2242633421` | 10,20,30,50,**100** |
| `multikill` | Multifrag | medal | `622331684` | 3,6,9,15,**30** |
| `melee_fighter` | Pugilat | stat | `melee_kills` | 5,10,15,25,**50** |
| `headshot` | Tir à la tête | stat | `headshot_kills` | 10,20,30,50,**100** |
| `spartan_killer` | Tueur de Spartans | stat | `kills` | 20,40,60,100,**200** |
| `eagle_eye` | Œil de lynx | medal | `1512363953` | 10,20,30,50,**100** |
| `avenger` | Vengeur | medal | `9000000001` (custom medal) | 5,15,30,55,**105** |

### PvP — Spartan Companies (15)

| Norm | Display | Type | Source | Tiers |
|------|---------|------|--------|-------|
| `flag_em_down` | Sors les drapeaux | custom | `compute_flag_em_down` | 1000,2000,3000,4800,**9700** |
| `grand_theft` | Vol à la tire | custom | `compute_hijack` | 200,400,600,960,**1940** |
| `helping_hand` | Coup de main | stat | `assists` | 20000,40000,60000,96000,**194400** |
| `im_just_perfect` | Zéro défaut | medal | `1512363953` | 2000,4000,6000,9600,**19400** |
| `lawnmower` | Tondeuse | medal | `221693153` | 500,1000,1500,2400,**4900** |
| `look_ma_no_pin` | Regarde maman, sans goupille | stat | `grenade_kills` | 4000,8000,12000,19200,**38900** |
| `lucky` | Lucky | medal | `3905838030,3091261182` | 400,800,1200,1920,**3880** |
| `no_hard_feelings` | Sans rancune | stat | `kills` | 50000,100000,150000,240000,**486000** |
| `positive_contribution` | Positive contribution | custom | `compute_bulldozer` | 300,600,900,1440,**2960** |
| `power_play` | Coup de force | stat | `power_weapon_kills` | 10000,20000,30000,48000,**97200** |
| `road_trip` | Virée sur la route | medal | `221693153` | 3000,6000,9000,14400,**29200** |
| `sting_like_a_bee` | Pique comme une abeille | stat | `melee_kills` | 5000,10000,15000,24000,**48600** |
| `the_reaper` | Le faucheur | medal | `2625820422` | 500,1000,1500,2400,**4850** |
| `too_fast_for_you` | Trop rapide pour toi | medal | `2123530881` | 2000,4000,6000,9600,**19400** |
| `vandalism` | Vandalisme | custom | `compute_vandalism` | 1200,2400,3600,5760,**11640** |

### PvP — Vehicle destroyers (7)

| Norm | Display | Type | Source | Tiers | Subcat |
|------|---------|------|--------|-------|--------|
| `wraith_destroyer` | Destructeur d'apparitions | custom | `compute_wraith_destroyer` | 3,6,9,15,**30** | Covenant |
| `banshee_destroyer` | Destructeur de banshees | award | `destroyed_banshee` (vehicle) | 3,6,9,15,**30** | Covenant |
| `ghost_destroyer` | Destructeur de ghosts | award | `destroyed_ghost` (vehicle) | 5,10,15,25,**50** | Covenant |
| `mongoose_destroyer` | Destructeur de mongooses | custom | `compute_mongoose_destroyer` | 5,10,15,25,**50** | UNSC |
| `scorpion_destroyer` | Destructeur de scorpions | award | `destroyed_scorpion` (vehicle) | 1,3,5,7,**10** | UNSC |
| `warthog_destroyer` | Destructeur de warthogs | custom | `compute_warthog_destroyer` | 5,10,15,25,**50** | UNSC |
| `wasp_destroyer` | Destructeur de wasps | award | `destroyed_wasp` (vehicle) | 3,6,9,15,**30** | UNSC |

### PvE — Firefight (10, 4 disabled)

| Norm | Display | Type | Source (`pve_match_stats`) | Enabled | Tiers |
|------|---------|------|----------------------------|---------|-------|
| `grunt_slayer` | Tueur de Grognards | pve_stat | `grunt_kills` | yes | 10,20,30,50,**100** |
| `elite_slayer` | Tueur d'Élites | pve_stat | `elite_kills` | yes | 10,20,30,50,**100** |
| `jackal_slayer` | Tueur de Rapaces | pve_stat | `jackal_kills` | yes | 5,10,15,25,**50** |
| `hunter_slayer` | Tueur de Chasseurs | pve_stat | `hunter_kills` | yes | 2,4,6,10,**20** |
| `sentinel_slayer` | Tueur de sentinelles | pve_stat | `sentinel_kills` | no | 5,10,15,25,**50** |
| `like_a_boss` | Comme un Boss | pve_stat | `boss_kills` | yes | 250,500,750,1200,**2400** |
| `player_vs_everything` | Éliminations Firefight | custom | `compute_wins_firefight` (Firefight wins, not a `pve_match_stats` column) | yes | 5,10,15,25,**50** |
| `brute_slayer` | Tueur de Brutes | pve_stat | `brute_kills` | no | 10,20,30,50,**100** |
| `skimmer_slayer` | Tueur de Skimmers | pve_stat | `skimmer_kills` | no | 10,20,30,50,**100** |
| `marine_slayer` | Tueur de Marines | pve_stat | `marine_kills` | no | 20,40,60,100,**200** |

### Weapons — UNSC (10, weapon_stat)

All `weapon_stat`, `stat_name = weapon_kills:<NameEN>`, subcategory `UNSC`.

| Norm | Display | weapon_kills | Tiers |
|------|---------|--------------|-------|
| `br75_mastery` | Maîtrise du BR75 | `BR75` | 25,50,100,200,**500** |
| `ma40_mastery` | Maîtrise du MA40 AR | `MA40 AR` | 25,50,100,200,**500** |
| `sidekick_mastery` | Maîtrise du MK50 Sidekick | `Mk51 Sidekick` | 10,25,50,100,**250** |
| `commando_mastery` | Maîtrise du VK78 Commando | `VK78 Commando` | 10,25,50,100,**250** |
| `sniper_mastery` | Maîtrise du S7 Sniper | `S7 Sniper` | 10,20,40,80,**200** |
| `spnkr_mastery` | Maîtrise du M41 SPNKr | `M41 SPNKr` | 10,20,40,80,**200** |
| `bulldog_mastery` | Maîtrise du CQS48 Bulldog | `CQS48 Bulldog` | 10,25,50,100,**250** |
| `bandit_mastery` | Maîtrise du Bandit EVO | `Bandit Evo` | 10,25,50,100,**250** |
| `hydra_mastery` | Maîtrise du MLRS-2 Hydra | `MLRS-2 Hydra` | 5,10,20,40,**100** |
| `mutilator_mastery` | Maîtrise du Mutilateur | `Mutilator` | 10,25,50,100,**250** |

### Weapons — Banished/Paria (9, weapon_stat)

Subcategory `Paria`.

| Norm | Display | weapon_kills | Tiers |
|------|---------|--------------|-------|
| `stalker_mastery` | Maîtrise du Fusil traqueur | `Stalker Rifle` | 10,25,50,100,**250** |
| `needler_mastery` | Maîtrise du Needler | `Needler` | 10,25,50,100,**250** |
| `energy_sword_mastery` | Maîtrise de l'Épée à énergie | `Energy Sword` | 10,20,40,80,**200** |
| `mangler_mastery` | Maîtrise du Déchiqueteur | `Mangler` | 10,25,50,100,**250** |
| `skewer_mastery` | Maîtrise de l'Empaleur | `Skewer` | 5,10,20,40,**100** |
| `gravity_hammer_mastery` | Maîtrise du Marteau antigravité | `Gravity Hammer` | 10,20,40,80,**200** |
| `pulse_carbine_mastery` | Maîtrise de la Carabine à impulsion | `Pulse Carbine` | 10,25,50,100,**250** |
| `ravager_mastery` | Maîtrise du Ravageur | `Ravager` | 5,10,20,40,**100** |
| `plasma_pistol_mastery` | Maîtrise du Pistolet à plasma | `Plasma Pistol` | 10,25,50,100,**250** |

### Weapons — Forerunner (5, weapon_stat)

Subcategory `Forerunner`.

| Norm | Display | weapon_kills | Tiers |
|------|---------|--------------|-------|
| `heatwave_mastery` | Maîtrise du Calcineur | `Heatwave` | 10,25,50,100,**250** |
| `cindershot_mastery` | Maîtrise du Crémateur | `Cindershot` | 10,20,40,80,**200** |
| `sentinel_beam_mastery` | Maîtrise du Rayon de Sentinelle | `Sentinel Beam` | 10,25,50,100,**250** |
| `disruptor_mastery` | Maîtrise du Disrupteur | `Disruptor` | 10,25,50,100,**250** |
| `shock_rifle_mastery` | Maîtrise du Fusil électrique | `Shock Rifle` | 10,25,50,100,**250** |

### Composites (7)

Type `composite`; value = count of mastered children.

| Norm | Display | Children |
|------|---------|----------|
| `covenant_destroyer` | Destructeur de Covenants | `grunt_slayer`, `elite_slayer`, `jackal_slayer`, `hunter_slayer`, `like_a_boss`, `brute_slayer`, `skimmer_slayer` |
| `grenade_mastery` | Maîtrise des grenades | `frag_grenade`, `plasma_grenade` |
| `vehicle_mastery` | Maîtrise de véhicule | `splatter`, `driver`, `wraith_destroyer`, `banshee_destroyer`, `ghost_destroyer`, `mongoose_destroyer`, `scorpion_destroyer`, `warthog_destroyer`, `wasp_destroyer` |
| `human_weapons_mastery` | Maîtrise des armes UNSC | the 10 UNSC `*_mastery` |
| `paria_weapons_mastery` | Maîtrise des armes Parias | the 9 Paria `*_mastery` |
| `forerunner_weapons_mastery` | Maîtrise des armes Forerunner | the 5 Forerunner `*_mastery` |
| `all_weapons_mastery` | Maîtrise en armement | `human_weapons_mastery`, `paria_weapons_mastery`, `forerunner_weapons_mastery`, `grenade_mastery` (meta) |

---

## Reference medal IDs

| medal_id | FR name | Used by |
|----------|---------|---------|
| `221693153` | Écrasement (Splatter) | `splatter`, `lawnmower`, `road_trip`, `vehicle_mastery` (via splatter) |
| `548533137` | Par derrière (Back Smack) | `assassin` |
| `622331684` | Double frag | `multikill`, `opportunist` |
| `1430343434` | Boucherie (Killtacular) | `opportunist` |
| `1486797009` | Carnage (Killtrocity) | `opportunist`, `spartan_carnage` |
| `1512363953` | Parfait (Perfection) | `eagle_eye`, `im_just_perfect` |
| `2063152177` | Triple frag | `opportunist` |
| `2123530881` | Revirement (Comeback) | `too_fast_for_you` |
| `2137071619` | Quelle tuerie (Overkill) | `opportunist` |
| `2242633421` | Meurtre mort détruire (Killimanjaro) | `opportunist` |
| `2625820422` | Frag d'outre-tombe (From the Grave) | `the_reaper` |
| `2648272972` | Grenadier | `frag_grenade` |
| `2926348688` | Pilote assist (Wheelman) | `driver` |
| `3091261182` | Chargeur vide (Empty Magazine) | `lucky` |
| `3169118333` | Violence routière | — (unreferenced since I7: `driver` → `2926348688`, `road_trip` → `221693153`) |
| `3655682764` | Collage (Stick) | `plasma_grenade` |
| `3905838030` | La chance (Lucky) | `lucky` |
| `4261842076` | Massacre (Killamanjaro) | `opportunist`, `spartan_carnage` |
| `9000000001` | Vengeur (custom medal definition) | `avenger` |

The full medal catalogue is populated from the 343i API. `SeedMedalDefinitions` (`internal/ops/seed.go`) only verifies and counts the `medal_definitions` table — it does **not** inject custom medals. The id `9000000001` (Vengeur/avenger) is not a real 343i medal: it exists only as the `MedalID` of the `avenger` citation mapping in `seed_citation_data.go`.

---

## Disabled commendations

The seed ships these with `Enabled: false` (listed, not computed):

| Norm | Why |
|------|-----|
| `sentinel_slayer` | Sentinels not tracked. |
| `brute_slayer` | Enemy type not present in Halo 5 (no image). |
| `skimmer_slayer` | Enemy type not present in Halo 5 (no image). |
| `marine_slayer` | Allies — not a meaningful positive commendation. |
| `flag_defender` | No unambiguous own-flag-defense ingestion award (`carrier_killed` = enemy-carrier kill = `flag_carrier_hunter`). Future candidate: `carrier_stopped`. |

---

## Intentional duplicates

Several commendations share the same source with very different tier targets:

| Source | Commendations (Master tier) |
|--------|------------------------------|
| `assists` | `assistant` (250), `helping_hand` (194400) |
| `kills` | `spartan_killer` (200), `no_hard_feelings` (486000) |
| `melee_kills` | `close_combat` (100), `melee_fighter` (50), `sting_like_a_bee` (48600) |
| medal `221693153` | `splatter` (100), `lawnmower` (4900), `road_trip` (29200) |
| medal `1512363953` | `eagle_eye` (100), `im_just_perfect` (19400) |
| `compute_bulldozer` | `bulldozer` (30), `positive_contribution` (2960) |

---

## Custom functions — all bound

Every custom function registered in `citations_custom.go` (`DispatchCustom`) is now referenced by a commendation. `compute_wins_firefight` was bound to `player_vs_everything` in I7 (Firefight wins), removing the last unbound rule.
