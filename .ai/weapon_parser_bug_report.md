# Rapport de bugs — Weapon Parser & Plan de correction

> Rédigé le 2026-03-11 | Branche : main | Scope : weapon_id=0 / NULL en DB

---

## 1. Contexte

Lors d'une analyse des `weapon_kills` de JGtm, 10 kills avec `weapon_id=0` ont été
identifiés manuellement en regardant les films. Résultat : plusieurs armes réelles
(Rayon de sentinelle arcane, Rushdown Hammer, Needler Pinpoint) et des corps-à-corps
sont stockés avec `weapon_id=0` (sentinelle grenade) au lieu de leur vrai hex.

Par ailleurs, 55 `weapon_id` avec le suffixe `42c9679f` présents dans `weapon_kills`
pour JGtm ne sont pas dans `WEAPON_ID_MAP` (dont 4 confirmés comme variantes du Sidekick
et 1 comme variante du MA40).

---

## 2. Architecture du pipeline (rappel)

```
SPNKr API → chunks film
    ↓
_weapon_kills_repo.load_player_kills_for_match()   ← classe les kills is_melee / is_grenade
    ↓
weapon_parser.correlate_kills_to_weapons()         ← corrige + corrèle fire events
    │
    ├─ is_melee=True  → _make_sentinel_result(MELEE_WEAPON_ID=1)
    ├─ is_grenade=True → _make_sentinel_result(GRENADE_WEAPON_ID=0)
    └─ sinon          → _match_kill_to_fire_event()
                            ↓
                        _scan_fire_events_bitstring()  ← cherche fire events dans chunk
                            ↓
                        weapon_id = best["weapon_bytes"] → int
                        weapon_id = None si aucun candidat
    ↓
_weapon_kills_repo.insert_weapon_kill_rows()       ← r.get("weapon_id") → NULL si None
```

---

## 3. Bugs identifiés

### Bug A — Faux positifs `is_grenade` par médaille (CRITIQUE)

**Fichier** : `src/data/repositories/_weapon_kills_repo.py:345-346`
**Aussi** : `_weapon_kills_repo.py:404-405` (batch `load_all_kills_for_match`)

```python
"is_melee":   any(m in MELEE_MEDALS for m in nearby),   # fenêtre ±500ms
"is_grenade": any(m in GRENADE_MEDALS for m in nearby),  # fenêtre ±500ms
```

**Problème** : la classification repose uniquement sur les médailles dans une
fenêtre de ±500ms autour du kill. Trois faux positifs observés :

| Situation | Cause du faux positif |
|-----------|----------------------|
| Kill arme normale + médaille "Boom!" obtenue ±500ms pour un autre kill | `is_grenade=True` → weapon_id=0 |
| Corps-à-corps sans médaille melee + médaille grenade proche | `is_grenade=True` au lieu de `is_melee=True` |
| Kill grenade "Dynamo" → CORRECT mais joueur voit une arme en main | Normal — attribution correcte, confusion visuelle |

**Cas observés** :
- Rows 1, 2, 7 (Rayon de sentinelle arcane) → `weapon_id=0` au lieu du vrai hex
- Rows 3, 6 (corps-à-corps avec Commando Impact / Mangler) → `weapon_id=0` au lieu de `1`

**Priorité** : HAUTE — affecte potentiellement des centaines de kills

---

### Bug B — Hammer (et armes mêlée sans fire event) non résolus (HAUTE)

**Fichier** : `src/analysis/weapon_parser.py:384-389` (`_match_kill_to_fire_event`)
**Fichier** : `src/analysis/_weapon_data.py:155-156` (`MELEE_MEDALS`)

```python
MELEE_MEDALS: frozenset[str] = frozenset(
    {"Pummel", "Assassination", "Back Smack", "Melee", "Quigley"}
)
```

**Problème** : Le Gravity Hammer et le Rushdown Hammer sont des armes de mêlée
qui ne génèrent **aucun fire event** (pas de projectile). La médaille obtenue par
un kill Hammer n'est pas dans `MELEE_MEDALS` (c'est "Skullcase", "Pound Town", etc.).

Résultat :
- `is_melee=False` (pas de médaille melee)
- `is_grenade=False` (pas de médaille grenade)
- `_match_kill_to_fire_event` appelé → aucun candidat → `weapon_id=None` → NULL en DB

**Cas observé** : Row 5 (Rushdown Hammer) → `weapon_id=0` (probablement NULL coercé
ou kill précédent grenade qui masque le vrai kill).

**Armes concernées** : Gravity Hammer, Rushdown Hammer, Diminisher of Hope,
Energy Sword et variantes (dans la mesure où elles ne tirent pas).

---

### Bug C — Variantes avec suffixe non-standard silencieusement ignorées (MOYENNE)

**Fichier** : `src/analysis/weapon_parser.py:223`

```python
# _scan_fire_events_bitstring
if weapon_int not in WEAPON_IDS_INT and weapon_bytes[4:] != COMMON_WEAPON_SUFFIX:
    continue  # ← drop silencieux
```

**Fichier** : `src/analysis/weapon_parser.py:100`

```python
# scan_formula_a
if suffix != COMMON_WEAPON_SUFFIX and data[ws_c : ws_c + 8] not in WEAPON_ID_MAP:
    continue  # ← drop silencieux
```

**Problème** : Toute variante d'arme dont le suffixe est ≠ `42c9679f` ET qui n'est
pas encore dans `WEAPON_ID_MAP` est complètement invisible au parser. Aucun log,
aucun compteur — le fire event est silencieusement ignoré.

**Impact** : si Arcane Sentinel Beam ou Needler Pinpoint ont un suffixe propre
(comme les variantes Sword `8978aa7a` / `1ec48c7a` ou Hammer `a730e49f` / `d8d07ca1`),
leurs fire events ne sont jamais capturés → `weapon_id=None` même quand le parser
est invoqué.

---

### Bug D — Mappings `WEAPON_ID_MAP` incomplets (5 hex confirmés manquants)

**Fichier** : `src/analysis/_weapon_data.py`

Identifiés par visionnage des films de JGtm :

| Hex (big-endian) | Arme | Kills JGtm | Statut |
|------------------|------|:----------:|--------|
| `0x91EB16DE42C9679F` | Mk51 Sidekick (variante) | 49 | À ajouter |
| `0xEDFF0E9642C9679F` | Mk51 Sidekick (variante) | 34 | À ajouter |
| `0xB1EB695E42C9679F` | Mk51 Sidekick (variante) | 23 | À ajouter |
| `0xF951480042C9679F` | Mk51 Sidekick (variante) | 7 | À ajouter |
| `0xF55C4BD242C9679F` | MA40 AR (variante) | 6 | À ajouter |

Note : les noms exacts des skins/variantes sont à confirmer (playlist ou cosmétique).
Pour l'instant les classer en `"Mk51 Sidekick (alt2/3/4/5)"` et `"MA40 AR (alt2)"`.

**Partiellement identifiés** (film visionné, hex inconnu) :
- Rayon de sentinelle arcane → hex manquant, probablement suffixe non-standard
- Needler Pinpoint → hex manquant, probablement suffixe non-standard

---

## 4. Plan de correction détaillé

### Étape 1 — Ajouter les 5 hex confirmés dans `WEAPON_ID_MAP` [IMMÉDIAT]

**Fichier** : `src/analysis/_weapon_data.py`
**Complexité** : triviale
**Risque** : nul

Ajouter dans la section `# ── Variantes / skins` :
```python
bytes.fromhex("91eb16de42c9679f"): "Mk51 Sidekick (alt2)",
bytes.fromhex("edff0e9642c9679f"): "Mk51 Sidekick (alt3)",
bytes.fromhex("b1eb695e42c9679f"): "Mk51 Sidekick (alt4)",
bytes.fromhex("f951480042c9679f"): "Mk51 Sidekick (alt5)",
bytes.fromhex("f55c4bd242c9679f"): "MA40 AR (alt2)",
```

Puis relancer le backfill `--weapon-kills` sur les matchs concernés pour re-attribuer
ces kills avec le bon weapon_id.

---

### Étape 2 — Corriger la priorité `is_melee` / `is_grenade` [PRIORITÉ HAUTE]

**Fichier** : `src/data/repositories/_weapon_kills_repo.py:338-348` et `395-406`
**Complexité** : faible
**Risque** : moyen (modifier le comportement de classification)

**Corrections à appliquer** :

#### 2a — `is_melee` prime sur `is_grenade`

Si un kill a à la fois une médaille melee ET une médaille grenade à ±500ms
(cas marginal mais possible), `is_melee` doit gagner :

```python
is_melee_val = any(m in MELEE_MEDALS for m in nearby)
is_grenade_val = any(m in GRENADE_MEDALS for m in nearby) and not is_melee_val
```

#### 2b — Réduire la fenêtre grenade à ±300ms

Les médailles grenade arrivent typiquement très proches du kill. Une fenêtre de
±500ms génère des faux positifs quand un autre kill (arme normale) se produit
juste avant/après un kill grenade du même joueur.

```python
MEDAL_WINDOW_MELEE_MS = 500   # inchangé
MEDAL_WINDOW_GRENADE_MS = 300  # réduit de 500 → 300
```

#### 2c — Enrichir `MELEE_MEDALS` avec les médailles Hammer

Ajouter les médailles spécifiques aux kills par Gravity Hammer :
```python
MELEE_MEDALS: frozenset[str] = frozenset({
    "Pummel", "Assassination", "Back Smack", "Melee", "Quigley",
    # Hammer kills — sans fire event, doivent passer par MELEE_WEAPON_ID
    "Skullcase", "Pound Town",   # à vérifier contre les données réelles
})
```

> **Action préalable** : requêter `medals_earned` pour les matchs Rushdown Hammer
> confirmés et lister les médailles obtenues dans ±500ms des kills identifiés.

---

### Étape 3 — Logger les variants inconnus au lieu de les dropper [PRIORITÉ MOYENNE]

**Fichier** : `src/analysis/weapon_parser.py:220-224`
**Complexité** : faible
**Risque** : nul (lecture seule)

Remplacer le `continue` silencieux par un log `DEBUG` pour aider à trouver
les hex des nouvelles variantes :

```python
# Avant le continue dans _scan_fire_events_bitstring
if weapon_int not in WEAPON_IDS_INT and weapon_bytes[4:] != COMMON_WEAPON_SUFFIX:
    logger.debug(
        "variant inconnue filtrée : %s (suffix=%s)",
        weapon_bytes.hex(), weapon_bytes[4:].hex()
    )
    continue
```

Idem dans `scan_formula_a:100`.

Cela permettra, lors du prochain `--weapon-kills` backfill, de collecter les hex
des variantes inconnues directement dans les logs.

---

### Étape 4 — Identifier les hex manquants (Sentinel Arcane, Needler Pinpoint)

**Méthode recommandée** (après Étape 3) :

1. Relancer le backfill `--weapon-kills` sur les matchs concernés avec log level DEBUG
2. Filtrer les lignes `variant inconnue filtrée` dans les logs
3. Comparer les suffixes aux patterns existants (Sword variants, Hammer variants)
4. Confirmer en re-visionnant le film au timestamp correspondant
5. Ajouter dans `WEAPON_ID_MAP` avec le nom confirmé

**Alternative** : chercher dans les data repos communautaires Halo Infinite
(OpenSpartan, Grunt.API) les hash d'assets pour "Arcane Sentinel Beam" et
"Needler Pinpoint".

---

### Étape 5 — Backfill de re-classification [APRÈS corrections 1+2]

Une fois les Étapes 1 et 2 appliquées, relancer le backfill sur tous les matchs
ayant des `weapon_kills` avec `weapon_id=0` ou `weapon_id IS NULL` :

```bash
# Identifier les match_ids à re-processer
# (tous les matchs JGtm avec au moins 1 kill weapon_id=0 ou NULL)
python scripts/backfill_data.py --weapon-kills --player JGtm --force-weapon-kills
```

> Ajouter l'option `--force-weapon-kills` si elle n'existe pas encore, pour
> forcer le re-traitement même si `weapon_kills` existe déjà pour le match.

---

## 5. Métriques d'impact estimé (JGtm seul)

| Bug | Kills affectés | % du total |
|-----|:--------------:|:----------:|
| weapon_id=0 (GRENADE sentinel) | 339 | 5.7% |
| weapon_id=1 (MELEE sentinel) | 661 | 11.2% |
| weapon_id=NULL | 1 | <0.1% |
| Hex manquants (5 confirmés) | ~123 | 2.1% |
| **Total potentiellement mal classifié** | **~1124** | **~19%** |

---

## 6. Ordre d'exécution recommandé

```
[1] Étape 1 — Ajouter 5 hex dans WEAPON_ID_MAP          (15 min, risque nul)
[2] Étape 3 — Logger les variants inconnus               (30 min, risque nul)
[3] Étape 2a — Priorité is_melee > is_grenade            (1h, test requis)
[4] Étape 2b — Fenêtre grenade 500→300ms                 (30 min, test requis)
[5] Étape 2c — Enrichir MELEE_MEDALS (Hammer)            (après requête médailles)
[6] Étape 4 — Collecter hex Sentinel/Needler via logs    (après Étape 3 + backfill)
[7] Étape 5 — Backfill re-classification complète        (après tout le reste)
```

---

## 7. Fichiers modifiés par ce plan

| Fichier | Étapes | Type |
|---------|--------|------|
| `src/analysis/_weapon_data.py` | 1, 2c | Ajout mappings + médailles |
| `src/analysis/weapon_parser.py` | 3 | Ajout logs |
| `src/data/repositories/_weapon_kills_repo.py` | 2a, 2b | Fix classification |
| `scripts/backfill_data.py` | 5 | Option `--force-weapon-kills` |
