# Référentiel complet des Citations LevelUp

> **Source de vérité** pour reconstruire `citation_mappings` en cas de perte.
> Dernière mise à jour : 21 février 2026 — 55 citations (51 activées, 4 désactivées).

---

## Architecture

### Stockage

| Élément | Emplacement | Rôle |
|---------|-------------|------|
| Référentiel des règles | `metadata.duckdb` → `citation_mappings` | 1 ligne = 1 citation, avec type, source, état |
| Résultats calculés | `stats.duckdb` (joueur) → `match_citations` | Valeur par match × citation |
| Stats PVE brutes | `shared_pve.duckdb` → `pve_match_stats` | Kills par type d'ennemi par match |
| Images | `static/commendations/h5g/*.png` | 158 fichiers PNG |
| Médailles FR | `static/medals/medals_fr.json` | 169 entrées `{medal_id: "Nom FR"}` |

### Script de peuplement

```bash
# Mode normal (UPSERT, supprime les obsolètes)
python scripts/populate_citation_mappings.py

# ⚠️ Mode reset (DROP TABLE + recréation complète)
python scripts/populate_citation_mappings.py --reset
```

### Schéma de la table `citation_mappings`

```sql
CREATE TABLE citation_mappings (
    citation_name_norm    VARCHAR PRIMARY KEY,  -- clé de matching (lowercase sans accents)
    citation_name_display VARCHAR NOT NULL,      -- nom affiché en FR
    mapping_type          VARCHAR NOT NULL,      -- medal | stat | award | custom | pve_stat | composite
    medal_id              BIGINT,                -- ID médaille unique (type medal)
    medal_ids             VARCHAR,               -- IDs séparés par virgule (type medal, multi)
    stat_name             VARCHAR,               -- colonne match_participants (type stat/pve_stat)
    award_name            VARCHAR,               -- nom exact du PersonalScoreAward (type award)
    award_category        VARCHAR,               -- objective | vehicle
    custom_function       VARCHAR,               -- nom fonction dans custom_rules.py (type custom)
    composite_children    VARCHAR,               -- JSON array de citation_name_norm (type composite)
    confidence            VARCHAR,               -- high | medium | low
    notes                 VARCHAR,               -- description libre
    enabled               BOOLEAN DEFAULT TRUE,
    created_at            TIMESTAMP,
    updated_at            TIMESTAMP
);
```

---

## Méthodes de calcul par `mapping_type`

### `medal` — Comptage de médailles

**Source** : `shared_matches.duckdb` → `medals_earned` (colonnes `medal_name_id`, `count`)

- Si `medal_ids` renseigné (CSV) : **somme** des `count` de chaque `medal_name_id`
- Sinon `medal_id` unique : `count` de la médaille

### `stat` — Lecture directe d'une colonne

**Source** : `shared_matches.duckdb` → `match_participants` (colonne = `stat_name`)

Valeur = `int(match_stats[stat_name])` par match.

### `pve_stat` — Lecture d'un kill PVE

**Source** : `shared_pve.duckdb` → `pve_match_stats` (colonne = `stat_name`)

Même principe que `stat`, mais les données PVE sont fusionnées dans `match_stats` par le moteur via `load_match_pve_stats()`.

### `award` — Comptage d'un PersonalScoreAward

**Source** : `stats.duckdb` (joueur) → `personal_score_awards` (colonne `award_name`, `award_count`)

Valeur = `sum(award_count)` où `award_name` matche exactement.

### `custom` — Fonction Python personnalisée

**Source** : `src/analysis/citations/custom_rules.py` → dictionnaire `CUSTOM_FUNCTIONS`

Chaque fonction reçoit `(df: pl.DataFrame, awards: dict, highlight_events: list)` et retourne un entier.

### `composite` — Agrégation d'enfants

**Calcul** : pas de valeur par match (retourne 0). À l'agrégation, somme brute des valeurs de tous les `composite_children` activés.

---

## Fonctions custom détaillées

| Fonction | Logique | Stats/tables utilisées |
|----------|---------|----------------------|
| `compute_bulldozer` | Matchs Slayer/Assassin (regex `slayer\|assassin` sur `playlist_name` ou `game_variant_name`) avec `kda > 8.0`. Exclut Firefight/BTB. Retourne 0 ou 1. | `match_participants.kda`, `match_registry.playlist_name`, `match_registry.game_variant_name` |
| `compute_wins_ctf` | Victoires (`outcome == 2`) en CTF (pattern `ctf\|capture.*drapeau\|drapeau.*neutre\|neutral.*flag`) | `match_participants.outcome`, `match_registry.playlist_name` |
| `compute_wins_slayer` | Victoires en Slayer (pattern `slayer\|assassin`) | idem |
| `compute_wins_strongholds` | Victoires en Strongholds (pattern `stronghold\|bases`) | idem |
| `compute_wins_firefight` | Victoires en Firefight (pattern `firefight\|baptême\|bapteme`) | idem |
| `compute_annexion_forcee` | **Mode précis** : parcourt `highlight_events`, compte les séquences de 3+ événements "mode" (capture zone) sans "death". **Fallback** : `zone_captures // 3` depuis awards. ~90% fiable. | `highlight_events`, `personal_score_awards` |
| `compute_flag_em_down` | Somme awards : `"Porteur arrêté"` + `"Flag Carrier Kill"` + `"Flag Carrier Killed"` | `personal_score_awards` |
| `compute_hijack` | Somme awards contenant `Hijacked`, `Hijack`, `Skyjack` (case-insensitive) | `personal_score_awards` |
| `compute_vandalism` | Somme awards contenant `Destroyed`, `Destruction` | `personal_score_awards` |
| `compute_wraith_destroyer` | `awards["Wraith Destroyed"] + awards["Apparition Destroyed"]` | `personal_score_awards` |
| `compute_mongoose_destroyer` | `awards["Mongoose Destroyed"]` | `personal_score_awards` |
| `compute_warthog_destroyer` | `awards["Warthog Destroyed"] + awards["Rocket Warthog Destroyed"]` | `personal_score_awards` |

---

## Inventaire complet des citations

### GROUPE 1 — Mode de jeu (11)

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

### GROUPE 2 — Arme (2)

| # | Norm | Display | Type | Source (medal_id) | Médaille FR | Tiers | Master | Image |
|---|------|---------|------|-------------------|-------------|-------|--------|-------|
| 12 | `ecrasement` | Écrasement | medal | `221693153` | Écrasement | 10, 20, 30, 50, **100** | 100 | `H5G_citation_%C3%89crasement.png` |
| 13 | `pilote` | Pilote | medal | `3169118333` | Violence routière | 10, 20, 30, 50, **100** | 100 | `H5G_citation_Pilote.png` |

### GROUPE 3 — Multijoueur (9)

| # | Norm | Display | Type | Source | Médaille(s) FR | Tiers | Master | Image |
|---|------|---------|------|--------|----------------|-------|--------|-------|
| 14 | `assassin` | Assassin | medal | `548533137` | Par derrière | 5, 10, 15, 25, **50** | 50 | `H5G_citation_Assassin.png` |
| 15 | `carnage de spartans` | Carnage de Spartans | medal | `2780740615,4261842076,418532952,1486797009,710323196,1720896992,2567026752,2875941471` | Folie meurtrière, Massacre, Émeute, Carnage, Cauchemar, Croque-mitaine, Croque-mort, Démon | 3, 6, 9, 15, **30** | 30 | `H5G_citation_Carnage_de_Spartans.png` |
| 16 | `combat rapproche` | Combat rapproché | stat | `melee_kills` | — | 10, 20, 30, 50, **100** | 100 | `H5G_citation_Combat_rapproch%C3%A9.png` |
| 17 | `combattant opportuniste` | Combattant opportuniste | medal | `medal_ids` | Double frag, Triple frag, Massacre, Quelle tuerie, Carnage, Boucherie, Meurtre mort détruire | 10, 20, 30, 50, **100** | 100 | `H5G_citation_Combattant_opportuniste.png` |
| 18 | `multifrag` | Multifrag | medal | `622331684` | Double frag | 3, 6, 9, 15, **30** | 30 | `H5G_citation_Multifrag.png` |
| 19 | `pugilat` | Pugilat | stat | `melee_kills` | — | 5, 10, 15, 25, **50** | 50 | `H5G_citation_Pugilat.png` |
| 20 | `tir a la tete` | Tir à la tête | stat | `headshot_kills` | — | 10, 20, 30, 50, **100** | 100 | `H5G_citation_Tir_%C3%A0_la_t%C3%AAte.png` |
| 21 | `tueur de spartans` | Tueur de Spartans | stat | `kills` | — | 20, 40, 60, 100, **200** | 200 | `H5G_citation_Tueur_de_Spartans.png` |
| 22 | `œil de lynx` | Œil de lynx | medal | `1512363953` | Parfait | 10, 20, 30, 50, **100** | 100 | `H5G_citation_%C5%92il_de_lynx.png` |

### GROUPE 4 — Spartan Companies (15)

> Note : `"player vs everything"` appartient au groupe PVE (pve_stat), pas SC.

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

### GROUPE 5 — Véhicules (7, catégorie "Arme")

| # | Norm | Display | Type | Source | Tiers | Master | Image |
|---|------|---------|------|--------|-------|--------|-------|
| 38 | `destructeur d'apparitions` | Destructeur d'apparitions | custom | `compute_wraith_destroyer` | 3, 6, 9, 15, **30** | 30 | `H5G_citation_Destructeur_d%27apparitions.png` |
| 39 | `destructeur de banshees` | Destructeur de banshees | award | `DESTROYED_BANSHEE` | 3, 6, 9, 15, **30** | 30 | `H5G_citation_Destructeur_de_banshees.png` |
| 40 | `destructeur de ghosts` | Destructeur de ghosts | award | `DESTROYED_GHOST` | 5, 10, 15, 25, **50** | 50 | `H5G_citation_Destructeur_de_ghosts.png` |
| 41 | `destructeur de mongooses` | Destructeur de mongooses | custom | `compute_mongoose_destroyer` | 5, 10, 15, 25, **50** | 50 | `H5G_citation_Destructeur_de_mongooses.png` |
| 42 | `destructeur de scorpions` | Destructeur de scorpions | award | `DESTROYED_SCORPION` | 1, 3, 5, 7, **10** | 10 | `H5G_citation_Destructeur_de_scorpions.png` |
| 43 | `destructeur de warthogs` | Destructeur de warthogs | custom | `compute_warthog_destroyer` | 5, 10, 15, 25, **50** | 50 | `H5G_citation_Destructeur_de_warthogs.png` |
| 44 | `destructeur de wasps` | Destructeur de wasps | award | `DESTROYED_WASP` | 3, 6, 9, 15, **30** | 30 | `H5G_citation_Destructeur_de_wasps.png` |

### GROUPE 6 — PVE Firefight (10, dont 4 désactivées)

| # | Norm | Display | Type | Source (pve_match_stats) | Enabled | Tiers | Master | Image |
|---|------|---------|------|--------------------------|---------|-------|--------|-------|
| 45 | `tueur de grognards` | Tueur de Grognards | pve_stat | `grunt_kills` | ✅ | 10, 20, 30, 50, **100** | 100 | `H5G_citation_Tueur_de_Grognards.png` |
| 46 | `tueur d'elites` | Tueur d'Élites | pve_stat | `elite_kills` | ✅ | 10, 20, 30, 50, **100** | 100 | `H5G_citation_Tueur_d%27%C3%89lites.png` |
| 47 | `tueur de rapaces` | Tueur de Rapaces | pve_stat | `jackal_kills` | ✅ | 5, 10, 15, 25, **50** | 50 | `H5G_citation_Tueur_de_Rapaces.png` |
| 48 | `tueur de chasseurs` | Tueur de Chasseurs | pve_stat | `hunter_kills` | ✅ | 2, 4, 6, 10, **20** | 20 | `H5G_citation_Tueur_de_Chasseurs.png` |
| 49 | `tueur de sentinelles` | Tueur de sentinelles | pve_stat | `sentinel_kills` | ❌ | 5, 10, 15, 25, **50** | 50 | `H5G_citation_Tueur_de_sentinelles.png` |
| 50 | `like a boss` | Comme un Boss | pve_stat | `boss_kills` | ✅ | 250, 500, 750, 1200, **2400** | 2400 | `H5G_citation_Like_a_boss.png` |
| 51 | `player vs everything` | Éliminations Firefight | pve_stat | `total_enemy_kills` | ✅ | 200, 400, 600, 960, **1940** | 1940 | `H5G_citation_Player_vs_Everything.png` |
| 52 | `tueur de brutes` | Tueur de Brutes | pve_stat | `brute_kills` | ❌ | — | — | *(pas d'image H5G)* |
| 53 | `tueur de skimmers` | Tueur de Skimmers | pve_stat | `skimmer_kills` | ❌ | — | — | *(pas d'image H5G)* |
| 54 | `tueur de marines` | Tueur de Marines | pve_stat | `marine_kills` | ❌ | 20, 40, 60, 100, **200** | 200 | `H5G_citation_Tueur_de_r%C3%A9pliques_de_Marines.png` |

### GROUPE 7 — Composite (1)

| # | Norm | Display | Type | Enfants | Calcul |
|---|------|---------|------|---------|--------|
| 55 | `destructeur de covenants` | Destructeur de Covenants | composite | `tueur de grognards`, `tueur d'elites`, `tueur de rapaces`, `tueur de chasseurs`, `like a boss`, `tueur de brutes`, `tueur de skimmers` | Somme brute des valeurs agrégées de tous les enfants activés. Pas d'image propre. |

---

## Medal IDs de référence

| medal_id | Nom FR | Nom EN | Citations |
|----------|--------|--------|-----------|
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
| `3091261182` | Chargeur vide | Empty Magazine | `lucky` |
| `3169118333` | Violence routière | Wheelman | `pilote`, `road trip` |
| `3905838030` | La chance | Lucky | `lucky` |
| `4261842076` | Massacre | Killamanjaro | `combattant opportuniste` |

---

## Fonctions custom non mappées

| Fonction | Raison |
|----------|--------|
| `compute_wins_firefight` | Enregistrée dans `CUSTOM_FUNCTIONS` mais aucune citation n'y fait référence. Candidate pour une future citation "Victoire en Firefight". |

---

## Doublons intentionnels

Certaines citations partagent la même source de données avec des tiers très différents (Mode de jeu = cibles courtes, SC = cibles de compagnie Spartan) :

| Source | Citations (Tiers) |
|--------|-------------------|
| `assists` | `assistant` (250), `helping hand` (194400) |
| `kills` | `tueur de spartans` (200), `no hard feelings` (486000) |
| `melee_kills` | `combat rapproche` (100), `pugilat` (50), `sting like a bee` (48600) |
| `medal 221693153` (Écrasement) | `ecrasement` (100), `lawnmower` (4900) |
| `medal 3169118333` (Violence routière) | `pilote` (100), `road trip` (29200) |
| `medal 1512363953` (Parfait) | `œil de lynx` (100), `i'm just perfect` (19400) |
| `compute_bulldozer` | `bulldozer` (30), `positive contribution` (2960) |

---

## Citations désactivées et raisons

| Norm | Raison |
|------|--------|
| `tueur de sentinelles` | Désactivée à la demande — sentinelles non suivies |
| `tueur de brutes` | Pas d'image H5G, type d'ennemi non présent dans H5 |
| `tueur de skimmers` | Pas d'image H5G, type d'ennemi non présent dans H5 |
| `tueur de marines` | Alliés, pas pertinent comme citation positive |

---

## Plan de découplage du JSON H5G

### Objectif

Supprimer la dépendance à `data/wiki/halo5_commendations_fr.json`. L'UI lit tout depuis `citation_mappings`.

### Colonnes à ajouter à `citation_mappings`

| Colonne | Type | Contenu |
|---------|------|---------|
| `image_path` | `VARCHAR` | Chemin relatif PNG (`static/commendations/h5g/...`) |
| `category` | `VARCHAR` | `Mode de jeu`, `Multijoueur`, `Arme`, `Spartan Companies`, `Ennemi` |
| `description` | `VARCHAR` | Description FR |
| `tier_targets` | `VARCHAR` | CSV des seuils, ex: `"10,20,30,50,100"` |

### Phases

1. **Schema** : `ALTER TABLE` pour ajouter les 4 colonnes
2. **Data** : Hardcoder les 4 champs dans les tuples de `populate_citation_mappings.py`
3. **UI** : Lire `citation_mappings` au lieu du JSON pour construire les items
4. **Cleanup** : Supprimer les JSON + fonctions de chargement associées

---

## Normalisation des noms

Fonction `_normalize_name(s: str) -> str` dans `src/ui/commendations.py` :

```python
base = " ".join(str(s or "").strip().lower().split())
return "".join(ch for ch in unicodedata.normalize("NFKD", base) if not unicodedata.combining(ch))
```

1. Strip + lowercase + collapse whitespace
2. Décomposition NFKD (sépare accents)
3. Supprime les caractères combinants (accents)

Exemples : `"À la charge"` → `"a la charge"`, `"Œil de lynx"` → `"œil de lynx"` (œ préservé car non décomposé en NFKD).
