# LevelUp Commendations — Full Reference

French version: [FR/CITATIONS_REFERENCE.md](FR/CITATIONS_REFERENCE.md)

This document is the **source of truth** to rebuild `citation_mappings` if needed.
Last update: February 21, 2026 — 55 commendations (51 enabled, 4 disabled).

---

## Architecture

### Storage

| Item | Location | Purpose |
|------|----------|---------|
| Rules registry | `metadata.duckdb` → `citation_mappings` | 1 row = 1 commendation (type, source, status) |
| Computed results | player `stats.duckdb` → `match_citations` | Value per match × commendation |
| Raw PvE stats | `shared_pve.duckdb` → `pve_match_stats` | Enemy-type kills per match |
| Images | `static/commendations/halo_5_guardians/*.png` | 158 PNG files |
| FR medals | `static/medals/medals_fr.json` | 169 entries `{medal_id: "Nom FR"}` |

### Populate script

```bash
# Normal mode (UPSERT, removes obsolete rows)
python scripts/populate_citation_mappings.py

# ⚠️ Reset mode (DROP TABLE + full rebuild)
python scripts/populate_citation_mappings.py --reset
```

### `citation_mappings` schema

```sql
CREATE TABLE citation_mappings (
    citation_name_norm    VARCHAR PRIMARY KEY,
    citation_name_display VARCHAR NOT NULL,
    mapping_type          VARCHAR NOT NULL,      -- medal | stat | award | custom | pve_stat | composite
    medal_id              BIGINT,
    medal_ids             VARCHAR,
    stat_name             VARCHAR,
    award_name            VARCHAR,
    award_category        VARCHAR,
    custom_function       VARCHAR,
    composite_children    VARCHAR,
    confidence            VARCHAR,
    notes                 VARCHAR,
    enabled               BOOLEAN DEFAULT TRUE,
    created_at            TIMESTAMP,
    updated_at            TIMESTAMP
);
```

---

## Calculation methods by `mapping_type`

### `medal` — Medal counting

Source: `shared_matches.duckdb` → `medals_earned` (`medal_name_id`, `count`)

- If `medal_ids` is set (CSV): **sum** of counts for each `medal_name_id`
- Else: single `medal_id` count

### `stat` — Direct column read

Source: `shared_matches.duckdb` → `match_participants` (column = `stat_name`)

Value = `int(match_stats[stat_name])` per match.

### `pve_stat` — PvE kill stat

Source: `shared_pve.duckdb` → `pve_match_stats` (column = `stat_name`)

Same idea as `stat`, but PvE stats are merged into `match_stats` via `load_match_pve_stats()`.

### `award` — PersonalScoreAward counting

Source: player `stats.duckdb` → `personal_score_awards` (`award_name`, `award_count`)

Value = `sum(award_count)` where `award_name` matches exactly.

### `custom` — Custom Python function

Source: `src/analysis/citations/custom_rules.py` → `CUSTOM_FUNCTIONS`

Each function receives `(df: pl.DataFrame, awards: dict, highlight_events: list)` and returns an `int`.

### `composite` — Children aggregation

No per-match value (returns 0). At aggregation time, a raw sum of all enabled `composite_children` values.

---

## Detailed custom functions

| Function | Logic | Used stats/tables |
|----------|-------|-------------------|
| `compute_bulldozer` | Slayer matches (regex `slayer\|assassin` on `playlist_name`/`game_variant_name`) with `kda > 8.0`. Excludes Firefight/BTB. Returns 0 or 1. | `match_participants.kda`, `match_registry.playlist_name`, `match_registry.game_variant_name` |
| `compute_wins_ctf` | Wins (`outcome == 2`) in CTF (pattern `ctf\|capture.*drapeau\|drapeau.*neutre\|neutral.*flag`) | `match_participants.outcome`, `match_registry.playlist_name` |
| `compute_wins_slayer` | Wins in Slayer (pattern `slayer\|assassin`) | same |
| `compute_wins_strongholds` | Wins in Strongholds (pattern `stronghold\|bases`) | same |
| `compute_wins_firefight` | Wins in Firefight (pattern `firefight\|baptême\|bapteme`) | same |
| `compute_annexion_forcee` | Precise mode: scans `highlight_events`, counts sequences of 3+ “mode” (zone capture) events without a “death”. Fallback: `zone_captures // 3` from awards. ~90% reliable. | `highlight_events`, `personal_score_awards` |
| `compute_flag_em_down` | Sum awards: `"Porteur arrêté"` + `"Flag Carrier Kill"` + `"Flag Carrier Killed"` | `personal_score_awards` |
| `compute_hijack` | Sum awards containing `Hijacked`, `Hijack`, `Skyjack` (case-insensitive) | `personal_score_awards` |
| `compute_vandalism` | Sum awards containing `Destroyed`, `Destruction` | `personal_score_awards` |
| `compute_wraith_destroyer` | `awards["Wraith Destroyed"] + awards["Apparition Destroyed"]` | `personal_score_awards` |
| `compute_mongoose_destroyer` | `awards["Mongoose Destroyed"]` | `personal_score_awards` |
| `compute_warthog_destroyer` | `awards["Warthog Destroyed"] + awards["Rocket Warthog Destroyed"]` | `personal_score_awards` |

---

## Full inventory

Note: `Display` values are currently in French because they match the stored/UI naming.

### GROUP 1 — Game mode (11)

| # | Norm | Display | Type | Source | Tiers | Master | Image |
|---|------|---------|------|--------|-------|--------|-------|
| 1 | `a la charge` | À la charge | award | `"Zone capturée"` | 10, 20, 30, 50, **100** | 100 | `H5G_citation_%C3%80_la_charge.png` |
| 2 | `annexion forcee` | Annexion forcée | custom | `compute_annexion_forcee` | 3, 6, 9, 15, **30** | 30 | `H5G_citation_Annexion_forc%C3%A9e.png` |
| 3 | `assistant` | Assistant | stat | `assists` | 25, 50, 75, 125, **250** | 250 | `H5G_citation_Assistant.png` |
| 4 | `bulldozer` | Bulldozer | custom | `compute_bulldozer` | 3, 6, 9, 15, **30** | 30 | `H5G_citation_Bulldozer.png` |
| 5 | `defenseur du drapeau` | Défenseur du drapeau | award | `"Porteur tué"` | 10, 20, 30, 50, **100** | 100 | `H5G_citation_D%C3%A9fenseur_du_drapeau.png` |
| 6 | `je te tiens !` | Je te tiens ! | award | `"Drapeau ramené"` | 10, 20, 30, 50, **100** | 100 | `H5G_citation_Je_te_tiens_%21.png` |
| 7 | `partie prenante` | Partie prenante | award | `"Zone sécurisée"` | 10, 20, 30, 50, **100** | 100 | `H5G_citation_Partie_prenante.png` |
| 8 | `sus au porteur du drapeau` | Sus au porteur du drapeau | award | `"Porteur tué"` | 10, 20, 30, 50, **100** | 100 | `H5G_citation_Sus_au_porteur_du_drapeau.png` |
| 9 | `victoire au drapeau` | Victoire au drapeau | custom | `compute_wins_ctf` | 5, 10, 15, 25, **50** | 50 | `H5G_citation_Victoire_au_drapeau.png` |
| 10 | `victoire en assassin` | Victoire en assassin | custom | `compute_wins_slayer` | 5, 10, 15, 25, **50** | 50 | `H5G_citation_Victoire_en_Assassin.png` |
| 11 | `victoire en bases` | Victoire en bases | custom | `compute_wins_strongholds` | 5, 10, 15, 25, **50** | 50 | `H5G_citation_Victoire_en_Bases.png` |

### GROUP 2 — Weapon (2)

| # | Norm | Display | Type | Source (medal_id) | FR medal name | Tiers | Master | Image |
|---|------|---------|------|-------------------|--------------|-------|--------|-------|
| 12 | `ecrasement` | Écrasement | medal | `221693153` | Écrasement | 10, 20, 30, 50, **100** | 100 | `H5G_citation_%C3%89crasement.png` |
| 13 | `pilote` | Pilote | medal | `3169118333` | Violence routière | 10, 20, 30, 50, **100** | 100 | `H5G_citation_Pilote.png` |

### GROUP 3 — Multiplayer (9)

| # | Norm | Display | Type | Source | FR medal(s) | Tiers | Master | Image |
|---|------|---------|------|--------|-------------|-------|--------|-------|
| 14 | `assassin` | Assassin | medal | `548533137` | Par derrière | 5, 10, 15, 25, **50** | 50 | `H5G_citation_Assassin.png` |
| 15 | `carnage de spartans` | Carnage de Spartans | medal | `2780740615,4261842076,418532952,1486797009,710323196,1720896992,2567026752,2875941471` | Folie meurtrière, Massacre, Émeute, Carnage, Cauchemar, Croque-mitaine, Croque-mort, Démon | 3, 6, 9, 15, **30** | 30 | `H5G_citation_Carnage_de_Spartans.png` |
| 16 | `combat rapproche` | Combat rapproché | stat | `melee_kills` | — | 10, 20, 30, 50, **100** | 100 | `H5G_citation_Combat_rapproch%C3%A9.png` |
| 17 | `combattant opportuniste` | Combattant opportuniste | medal | `medal_ids` | Double frag, Triple frag, Massacre, Quelle tuerie, Carnage, Boucherie, Meurtre mort détruire | 10, 20, 30, 50, **100** | 100 | `H5G_citation_Combattant_opportuniste.png` |
| 18 | `multifrag` | Multifrag | medal | `622331684` | Double frag | 3, 6, 9, 15, **30** | 30 | `H5G_citation_Multifrag.png` |
| 19 | `pugilat` | Pugilat | stat | `melee_kills` | — | 5, 10, 15, 25, **50** | 50 | `H5G_citation_Pugilat.png` |
| 20 | `tir a la tete` | Tir à la tête | stat | `headshot_kills` | — | 10, 20, 30, 50, **100** | 100 | `H5G_citation_Tir_%C3%A0_la_t%C3%AAte.png` |
| 21 | `tueur de spartans` | Tueur de Spartans | stat | `kills` | — | 20, 40, 60, 100, **200** | 200 | `H5G_citation_Tueur_de_Spartans.png` |
| 22 | `œil de lynx` | Œil de lynx | medal | `1512363953` | Parfait | 10, 20, 30, 50, **100** | 100 | `H5G_citation_%C5%92il_de_lynx.png` |

### GROUP 4 — Spartan Companies (15)

> Note: `"player vs everything"` belongs to the PvE group (`pve_stat`), not Spartan Companies.

| # | Norm | Display | Type | Source | Tiers | Master | Image |
|---|------|---------|------|--------|-------|--------|-------|
| 23 | `flag 'em down` | Sors les drapeaux | custom | `compute_flag_em_down` | 1000, 2000, 3000, 4800, **9700** | 9700 | `H5G_citation_Flag_%27em_down.png` |
| 24 | `grand theft` | Vol à la tire | custom | `compute_hijack` | 200, 400, 600, 960, **1940** | 1940 | `H5G_citation_Grand_Theft.png` |
| 25 | `helping hand` | Coup de main | stat | `assists` | 20000, 40000, 60000, 96000, **194400** | 194400 | `H5G_citation_Helping_Hand.png` |
| 26 | `i'm just perfect` | Zéro défaut | medal | `1512363953` (Parfait) | 2000, 4000, 6000, 9600, **19400** | 19400 | `H5G_citation_I%27m_just_perfect.png` |
| 27 | `lawnmower` | Tondeuse | medal | `221693153` (Écrasement) | 500, 1000, 1500, 2400, **4900** | 4900 | `H5G_citation_Lawnmower.png` |
| 28 | `look ma no pin` | Regarde maman, sans goupille | stat | `grenade_kills` | 4000, 8000, 12000, 19200, **38900** | 38900 | `H5G_citation_Look_ma_no_pin.png` |
| 29 | `lucky` | Lucky | medal | `medal_ids: 3905838030,3091261182` (La chance + Chargeur vide) | 400, 800, 1200, 1920, **3880** | 3880 | `H5G_citation_Lucky.png` |
| 30 | `no hard feelings` | Sans rancune | stat | `kills` | 50000, 100000, 150000, 240000, **486000** | 486000 | `H5G_citation_No_Hard_Feelings.png` |
| 31 | `positive contribution` | Positive contribution | custom | `compute_bulldozer` | 300, 600, 900, 1440, **2960** | 2960 | `H5G_citation_Positive_contribution.png` |
| 32 | `power play` | Coup de force | stat | `power_weapon_kills` | 10000, 20000, 30000, 48000, **97200** | 97200 | `H5G_citation_Power_play.png` |
| 33 | `road trip` | Virée sur la route | medal | `3169118333` (Violence routière) | 3000, 6000, 9000, 14400, **29200** | 29200 | `H5G_citation_Road_Trip.png` |
| 34 | `sting like a bee` | Pique comme une abeille | stat | `melee_kills` | 5000, 10000, 15000, 24000, **48600** | 48600 | `H5G_citation_Sting_like_a_bee.png` |
| 35 | `the reaper` | Le faucheur | medal | `2625820422` (Frag d'outre-tombe) | 500, 1000, 1500, 2400, **4850** | 4850 | `H5G_citation_The_Reaper.png` |
| 36 | `too fast for you` | Trop rapide pour toi | medal | `2123530881` (Revirement) | 2000, 4000, 6000, 9600, **19400** | 19400 | `H5G_citation_Too_fast_for_you.png` |
| 37 | `vandalisme` | Vandalisme | custom | `compute_vandalism` | 1200, 2400, 3600, 5760, **11640** | 11640 | `H5G_citation_Vandalism.png` |

### GROUP 5 — Vehicles (7)

| # | Norm | Display | Type | Source | Tiers | Master | Image |
|---|------|---------|------|--------|-------|--------|-------|
| 38 | `destructeur d'apparitions` | Destructeur d'apparitions | custom | `compute_wraith_destroyer` | 3, 6, 9, 15, **30** | 30 | `H5G_citation_Destructeur_d%27apparitions.png` |
| 39 | `destructeur de banshees` | Destructeur de banshees | award | `DESTROYED_BANSHEE` | 3, 6, 9, 15, **30** | 30 | `H5G_citation_Destructeur_de_banshees.png` |
| 40 | `destructeur de ghosts` | Destructeur de ghosts | award | `DESTROYED_GHOST` | 5, 10, 15, 25, **50** | 50 | `H5G_citation_Destructeur_de_ghosts.png` |
| 41 | `destructeur de mongooses` | Destructeur de mongooses | custom | `compute_mongoose_destroyer` | 5, 10, 15, 25, **50** | 50 | `H5G_citation_Destructeur_de_mongooses.png` |
| 42 | `destructeur de scorpions` | Destructeur de scorpions | award | `DESTROYED_SCORPION` | 1, 3, 5, 7, **10** | 10 | `H5G_citation_Destructeur_de_scorpions.png` |
| 43 | `destructeur de warthogs` | Destructeur de warthogs | custom | `compute_warthog_destroyer` | 5, 10, 15, 25, **50** | 50 | `H5G_citation_Destructeur_de_warthogs.png` |
| 44 | `destructeur de wasps` | Destructeur de wasps | award | `DESTROYED_WASP` | 3, 6, 9, 15, **30** | 30 | `H5G_citation_Destructeur_de_wasps.png` |

### GROUP 5b — Grenades (2)

| # | Norm | Display | Type | Source (medal_id) | Tiers | Master | Image |
|---|------|---------|------|-------------------|-------|--------|-------|
| 45 | `frag_grenade` | Grenade à fragmentation | medal | `2648272972` (Grenadier) | 5, 10, 15, 25, **50** | 50 | `H5G_citation_Grenade_%C3%A0_fragmentation.png` |
| 46 | `plasma_grenade` | Grenade à plasma | medal | `3655682764` (Collage/Stick) | 2, 4, 6, 10, **20** | 20 | `H5G_citation_Grenade_%C3%A0_plasma.png` |

### GROUP 7 — PvE Firefight (10, including 4 disabled)

| # | Norm | Display | Type | Source (pve_match_stats) | Enabled | Tiers | Master | Image |
|---|------|---------|------|--------------------------|---------|-------|--------|-------|
| 47 | `tueur de grognards` | Tueur de Grognards | pve_stat | `grunt_kills` | ✅ | 10, 20, 30, 50, **100** | 100 | `H5G_citation_Tueur_de_Grognards.png` |
| 48 | `tueur d'elites` | Tueur d'Élites | pve_stat | `elite_kills` | ✅ | 10, 20, 30, 50, **100** | 100 | `H5G_citation_Tueur_d%27%C3%89lites.png` |
| 49 | `tueur de rapaces` | Tueur de Rapaces | pve_stat | `jackal_kills` | ✅ | 5, 10, 15, 25, **50** | 50 | `H5G_citation_Tueur_de_Rapaces.png` |
| 50 | `tueur de chasseurs` | Tueur de Chasseurs | pve_stat | `hunter_kills` | ✅ | 2, 4, 6, 10, **20** | 20 | `H5G_citation_Tueur_de_Chasseurs.png` |
| 51 | `tueur de sentinelles` | Tueur de sentinelles | pve_stat | `sentinel_kills` | ❌ | 5, 10, 15, 25, **50** | 50 | `H5G_citation_Tueur_de_sentinelles.png` |
| 52 | `like a boss` | Comme un Boss | pve_stat | `boss_kills` | ✅ | 250, 500, 750, 1200, **2400** | 2400 | `H5G_citation_Like_a_boss.png` |
| 53 | `player vs everything` | Éliminations Firefight | pve_stat | `total_enemy_kills` | ✅ | 200, 400, 600, 960, **1940** | 1940 | `H5G_citation_Player_vs_Everything.png` |
| 54 | `tueur de brutes` | Tueur de Brutes | pve_stat | `brute_kills` | ❌ | — | — | *(no H5G image)* |
| 55 | `tueur de skimmers` | Tueur de Skimmers | pve_stat | `skimmer_kills` | ❌ | — | — | *(no H5G image)* |
| 56 | `tueur de marines` | Tueur de Marines | pve_stat | `marine_kills` | ❌ | 20, 40, 60, 100, **200** | 200 | `H5G_citation_Tueur_de_r%C3%A9pliques_de_Marines.png` |

### GROUP 8 — Composites (3)

| # | Norm | Display | Type | Children | Calculation |
|---|------|---------|------|----------|-------------|
| 57 | `covenant_destroyer` | Destructeur de Covenants | composite | `grunt_slayer`, `elite_slayer`, `jackal_slayer`, `hunter_slayer`, `like_a_boss`, `brute_slayer`, `skimmer_slayer` | Count of mastered children (not raw sum). |
| 58 | `grenade_mastery` | Maîtrise des grenades | composite | `frag_grenade`, `plasma_grenade` | Count of mastered children. |
| 59 | `vehicle_mastery` | Maîtrise de véhicule | composite | `splatter`, `driver`, `wraith_destroyer`, `banshee_destroyer`, `ghost_destroyer`, `mongoose_destroyer`, `scorpion_destroyer`, `warthog_destroyer`, `wasp_destroyer` | Count of mastered children. |

---

## Reference medal IDs

| medal_id | FR name | EN name | Used by |
|----------|---------|---------|---------|
| `221693153` | Écrasement | Splatter | `ecrasement`, `lawnmower` |
| `548533137` | Par derrière | Back Smack | `assassin` |
| `622331684` | Double frag | Double Kill | `multifrag`, `combattant opportuniste` |
| `1430343434` | Boucherie | Killtacular | `combattant opportuniste` |
| `1486797009` | Carnage | Killtrocity | `combattant opportuniste` |
| `1512363953` | Parfait | Perfection | `œil de lynx`, `i'm just perfect` |
| `2063152177` | Triple frag | Triple Kill | `combattant opportuniste` |
| `2123530881` | Revirement | Comeback | `too fast for you` |
| `2137071619` | Quelle tuerie | Overkill | `combattant opportuniste` |
| `2242633421` | Meurtre mort détruire | Killimanjaro | `combattant opportuniste` |
| `2625820422` | Frag d'outre-tombe | From the Grave | `the reaper` |
| `2648272972` | Grenadier | Grenadier | `frag_grenade` |
| `3091261182` | Chargeur vide | Empty Magazine | `lucky` |
| `3169118333` | Violence routière | Wheelman | `pilote`, `road trip` |
| `3655682764` | Collage | Stick | `plasma_grenade` |
| `3905838030` | La chance | Lucky | `lucky` |
| `4261842076` | Massacre | Killamanjaro | `combattant opportuniste` |

---

## Unmapped custom functions

| Function | Reason |
|----------|--------|
| `compute_wins_firefight` | Registered in `CUSTOM_FUNCTIONS` but not referenced by any commendation. Candidate for a future “Firefight win” commendation. |

---

## Intentional duplicates

Some commendations share the same data sources but have very different tier targets:

| Source | Commendations (tiers) |
|--------|------------------------|
| `assists` | `assistant` (250), `helping hand` (194400) |
| `kills` | `tueur de spartans` (200), `no hard feelings` (486000) |
| `melee_kills` | `combat rapproche` (100), `pugilat` (50), `sting like a bee` (48600) |
| medal `221693153` | `ecrasement` (100), `lawnmower` (4900) |
| medal `3169118333` | `pilote` (100), `road trip` (29200) |
| medal `1512363953` | `œil de lynx` (100), `i'm just perfect` (19400) |
| `compute_bulldozer` | `bulldozer` (30), `positive contribution` (2960) |

---

## Disabled commendations and reasons

| Norm | Reason |
|------|--------|
| `tueur de sentinelles` | Disabled on request — sentinels not tracked |
| `tueur de brutes` | No H5G image, enemy type not present in Halo 5 |
| `tueur de skimmers` | No H5G image, enemy type not present in Halo 5 |
| `tueur de marines` | Allies, not a meaningful positive commendation |

---

## Plan: remove the H5G JSON dependency

Goal: remove dependency on `data/wiki/halo5_commendations_fr.json`. The UI should read everything from `citation_mappings`.

### Columns to add to `citation_mappings`

| Column | Type | Meaning |
|--------|------|---------|
| `image_path` | `VARCHAR` | Relative PNG path (`static/commendations/halo_5_guardians/...`) |
| `category` | `VARCHAR` | `Game mode`, `Multiplayer`, `Weapon`, `Spartan Companies`, `Enemy` |
| `description` | `VARCHAR` | French description (for now) |
| `tier_targets` | `VARCHAR` | CSV targets, e.g. `"10,20,30,50,100"` |

### Phases

1. Schema: `ALTER TABLE` to add 4 columns
2. Data: hardcode the 4 fields in `populate_citation_mappings.py`
3. UI: read `citation_mappings` instead of JSON
4. Cleanup: remove JSON files and related loaders

---

## Name normalization

Function `_normalize_name(s: str) -> str` in `src/ui/commendations.py`:

```python
base = " ".join(str(s or "").strip().lower().split())
return "".join(ch for ch in unicodedata.normalize("NFKD", base) if not unicodedata.combining(ch))
```

1. Strip + lowercase + collapse whitespace
2. NFKD decomposition
3. Remove combining characters (accents)

Example: `"À la charge"` → `"a la charge"`.
