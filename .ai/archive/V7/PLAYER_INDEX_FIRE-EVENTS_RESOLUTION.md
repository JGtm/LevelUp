# Résolution player_index dans les fire events — Rapport définitif

> Dernière mise à jour : **2026-03-19**
> Statut : **MISE À JOUR — b5>>4 validé (inv134), sentinel API validé (inv135), structure melee NS partiellement validée (inv136)**
> Scripts : `inv131`, `inv132`, `inv133`, `inv134_b5_pi_attribution.py`, `inv135_sentinel_nongun.py`, `inv136_melee_marker.py`

---

## ⛔ CE QUI NE FONCTIONNE PAS — NE PAS RÉESSAYER

---

### ~~Théorie 1 — byte[1] encode le player_index~~
**Date invalidée : ~2026-03-09 (FINDINGS inv#1–125) — confirmée 2026-03-18**

**Affirmation (FINDINGS) :** `byte[1] = (pi << 5) | 0x06`, donc `0x26 = pi=1 = joueur POV`.
Les fire events seraient "POV only" car tous ont `byte[1]=0x26`.

**> FAUX.**

`byte[1] = 0x26` est un **marqueur structurel fixe identique pour tous les joueurs**.
La formule `(1 << 5) | 0x06 = 0x26` est une coïncidence numérique.
Scan exhaustif des 8 variants (`0x26, 0x46, 0x66, 0x86, 0xa6, 0xc6, 0xe6, 0x06`) :
**seul `0x26` donne des résultats** (52–1177 events selon le match).
Les 7 autres variants retournent **0 résultat**.

Cette erreur d'interprétation a contaminé toute la recherche inv#1–125 et produit
la conclusion fausse "POV only / T0 impossible / Architecturally final".

---

### ~~Théorie 2 — byte[0] (top-3-bits) encode le player_index~~
**Date invalidée : 2026-03-16**

**Affirmation :** `pi = byte[0] >> 5` (analogie avec Section 1 Formula A).
Scan des 8 variants de `byte[0]` (`0x0d, 0x2d, 0x4d, 0x6d, 0x8d, 0xad, 0xcd, 0xed`).

**> FAUX.**

Seul `byte[0] = 0x0d` donne des events valides.
Top-3-bits de `byte[0]` = 0 pour **55/58** events valides analysés.
`byte[0]` est lui aussi fixe — il ne discrimine pas les joueurs.

---

### ~~Théorie 3 — fire_counter discrimine le joueur~~
**Date invalidée : 2026-03-17**

**Affirmation :** le fire_counter (byte[4]) serait par joueur (repart à 0 par vie).

**> FAUX.**

Le fire_counter est un compteur global monotone partagé entre tous les joueurs.
Il ne peut pas servir de discriminant.

---

### ~~Théorie 4 — Les fire events sont POV-only~~
**Date invalidée : 2026-03-18 — confirmation acurtis**

**Affirmation (FINDINGS inv#1–125, Section 3) :**
> *"T0 opponent fire events | Impossible | Server asymmetric replication:
> opponent weapon states not transmitted to client. Architecturally final."*

**> ABSOLUMENT FAUX.**

Le serveur réplique les fire events de **tous les joueurs** dans chaque chunk.
Match `147ffd4d` (Super Fiesta, 10 joueurs) : **1177 fire events** pour **9 joueurs**,
confirmé indépendamment par acurtis (baseline : 1178).

**Ne jamais réutiliser cette conclusion.**

---

### ~~Théorie 5 — `POV_PLAYER_INDEX = 1`~~
**Date invalidée : 2026-03-18**

**Affirmation (weapon_parser.py:45) :** le joueur POV a player_index=1.

**> FAUX.**

Le joueur POV a **pi=0** dans le système fire_seq (`fire_seq % n_players = 0`).
`_build_marker(1) = 0x26` fonctionnait par coïncidence — pas parce que pi=1 est le POV.
La constante `POV_PLAYER_INDEX = 1` est une relique de l'erreur Théorie 1.

---

## ⚠️ CE QUI EST APPROXIMATIF — FALLBACK ACCEPTABLE SEULEMENT

---

### `fire_seq % n_players` = player_index (inv#132)
**Statut : APPROXIMATIF — fonctionne sur 8 joueurs, échoue si n_players variable ou incorrect**

Validé 7/7 sur d9329229 (8 joueurs fixes). Échoue sur a974fdeb (9 joueurs, n_eff difficile
à déterminer). Déclassé en fallback suite à la découverte de b5>>4 (inv134).

**Limite principale :** `n_players` doit être exact. Impossible à calculer de façon fiable
si un joueur rejoint/quitte en cours de match ou si le compte est ambigu.

---

### Formula A — Section 1 (snapshots d'état arme)
**Statut : fallback — utilisé quand aucun fire event disponible**

Formula A lit le weapon_id depuis les snapshots d'état arme (Section 1, `[20 00 02 pb]`).
`pi = pb >> 5` (3 bits hauts du payload byte).

**Limites :**
- Retourne des **skin IDs** (cosmétiques) non répertoriés dans `weapon_labels` → `None` en DB
- **Aliasing pi** en grands matchs : plusieurs joueurs partagent le même `pb >> 5` (inv#110)
- Résultat `None` si aucun snapshot trouvé pour ce pi dans le chunk courant/voisin
- **Système de numérotation distinct** de fire_seq — pas directement compatible

**Usage correct :** fallback uniquement quand `b5>>4` ne peut pas résoudre
(joueur sans pi résolu dans `xuid_to_pi`, ou match sans chunks en cache).

---

### map_b2_to_player — NS cross-reference
**Statut : remplacé — ne pas utiliser pour la production**

Cross-référence `fire_seq` → `pi` via Section 1 timeline (cherche quel pi a la même arme
au même timestamp dans les snapshots). Produit des **drops** (events non résolus si
le snapshot n'est pas disponible) et des **erreurs** si deux joueurs ont la même arme
au même moment.

**Remplacé par** `fire_seq % n_players` (zéro drop, déterministe).

---

### Global claim-and-remove sans filtre pi
**Statut : BUGGÉ — produisait de la cross-attribution**

Ancien comportement de `correlate_kills_global` : pool unique de tous les events,
claim-and-remove par proximité temporelle sans filtrage par joueur.

**Symptôme observé (match `82f3af9f`) :** 4 fire events de FreshKalvin203 attribués
aux kills de JGtm (×1), NewFlipBobCat (×2), alpal capone (×1).

**Supprimé le 2026-03-18** — remplacé par le filtre `ev["player_index"] == killer_pi`.

---

## ✅ CE QUI FONCTIONNE — SOLUTION DÉFINITIVE

---

### `b5 >> 4` = player_index (inv#134)
**Date de validation : inv#134 — 2026-03-19 — 3 matchs, 282 kills**

#### Formule

```
Structure fire event (nibble-shifted bitstream) :
  marker 11b : 0b10100100110  (0d 26 — fixe, universel tous joueurs)
  event_start = marker_pos + 3
  b1 [+0..+7]   : 0x26 (fixe)
  b2 [+8..+15]  : fire_seq
  b3 [+16..+23] : 0x40 (fixe)
  b4 [+24..+31] : fire_counter (uint8, wraparound — NE PAS utiliser pour dédup)
  b5 [+32..+39] : (player_index << 4) | slot
  weapon [+40..+103] : 8 bytes big-endian

player_index = b5 >> 4
slot         = b5 & 0x03   (1 = primary, 3 = secondary)
```

Valeurs b5 observées : `{3, 17, 19, 33, 35, 49, 51, 65, 81, 83, 97, 99, 113, 115}`
= 8 joueurs (pi 0–7) × 2 slots. Pi=0 = joueur POV/enregistreur.

#### Validation (inv#134)

| Match | Mode | Kills | Conf (<500ms) | Taux |
|-------|------|-------|---------------|------|
| a974fdeb | Quick Play | 87 | 73 | 84% |
| f2f81265 | Quick Play | 98 | 87 | 89% |
| d9329229 | Quick Play | 97 | 92 | 95% |

#### Points critiques

- **Déduplication** : par proximité `byte_pos` (< 2 bytes = même event physique).
  Ne **pas** déduper par `(fire_counter, weapon)` : fire_counter wrappe à 255 et
  supprime des events légitimes sur armes automatiques (bug confirmé sur KIN92/MA40 AR).
- **Mapping pi → xuid** : via `detect_pi_from_metadata()` (PLAYER_METADATA payload ~25KB).
  POV player absent de PLAYER_METADATA (pi=0 filtré) → reçoit pi=0 par fallback.
- **Kills sans fire event** : gap > 500ms ou no_event = kill grenade/melee probable.
  ~10.6% des kills sur 3 matchs (30/282). Traité par sentinel API (inv135).

#### Implémentation

Script de référence : `scripts/experimental/inv134_b5_pi_attribution.py`

**Statut production : MIGRÉ (2026-03-19) ✅**

`scan_fire_events_all` utilise désormais `scan_fire_events_b5` — `fire_seq % n_players` supprimé.
Migration requise dans ces fichiers :

| Fichier | Changement |
|---------|-----------|
| `src/analysis/_weapon_scanners.py` | Remplacer `scan_fire_events_bitstring` : extraire b5 à chaque event, retourner `player_index = b5 >> 4`. Dédup par `byte_pos` proximity (≤2 bytes), pas par `(fire_counter, weapon)`. |
| `src/analysis/weapon_parser.py` | `scan_fire_events_all` : supprimer le paramètre `n_players` et la ligne `ev["player_index"] = ev["fire_seq"] % n_players`. Le `player_index` vient désormais du scan b5. |
| `src/data/services/weapon_extraction_service.py` | Supprimer `n_players = len(all_participants) or 8` passé à `scan_fire_events_all`. |

**Ce qui NE change PAS :**
- `correlate_kills_global` : le filtre `ev["player_index"] == killer_pi` reste identique.
- `detect_pi_from_metadata` : le mapping xuid→pi reste identique.
- `PLAYER_METADATA` : toujours la source du mapping pi→xuid.

---

### Sentinel grenade/melee via counts API (inv#135)
**Date de validation : inv#135 — 2026-03-19 — 3 matchs, 282 kills, gun_diff=+0**

#### Problème

b5>>4 attribue une arme à ~89% des kills (gap <500ms). Les ~11% restants (gap ≥500ms
ou aucun fire event) correspondent aux kills grenade et melee, mais on ne peut pas
déterminer par le seul film lequel est lequel, ni par quel kill individuel.

#### Données disponibles

`match_participants` contient `grenade_kills` et `melee_kills` (totaux par joueur par match)
fournis par l'API Halo Infinite. Ces totaux sont exacts et servent de quota.

#### Algorithme sentinel

```
Pour chaque joueur :
  1. Attribuer un fire event (b5>>4) à chaque kill → gap en ms (ou None si aucun event)
  2. Trier les kills par gap décroissant (None = infini = plus incertain)
  3. Marquer les grenade_kills premiers comme GRENADE
  4. Marquer les melee_kills suivants comme MELEE
  5. Les kills restants gardent l'arme du fire event (gun kill confirmé)
```

Principe : les kills les plus éloignés d'un fire event sont les plus probables d'être
non-gun. Le quota API borne exactement le nombre de sentinels posés — pas de surcompensation.

#### Validation (inv#135)

| Match | Kills | gun_diff | Toutes lignes OK ? |
|-------|-------|----------|--------------------|
| a974fdeb | 87 | **+0** | 8/8 ✅ |
| f2f81265 | 98 | **+0** | 8/8 ✅ |
| d9329229 | 97 | **+0** | 8/8 ✅ |
| **TOTAL** | **282** | **+0** | **24/24 ✅** |

Script de référence : `scripts/experimental/inv135_sentinel_nongun.py`

#### Limites connues

- On ne sait pas distinguer grenade vs melee par le film — l'ordre GRENADE avant MELEE
  dans le tri par gap est une convention arbitraire. Les counts totaux sont exacts,
  pas l'assignation individuelle grenade/melee par kill.
- Si un joueur n'a aucun chunk en cache, les gaps sont tous None → les N premiers kills
  (par ordre chronologique croissant) seront marqués sentinel, ce qui est incorrect.
  Mitigé par le fait que sans chunks on n'a de toute façon pas d'attribution film.
- Le seuil GAP_CONF_MS=500ms est empirique. Un kill gun avec une pause de 600ms sera
  incorrectement marqué sentinel si le quota n'est pas épuisé par des gaps plus grands.

---

## 🔬 EN COURS D'INVESTIGATION — RÉSULTATS PARTIELS

---

### Melee events dans la couche NS (inv#136)
**Date : 2026-03-19 — match a974fdeb (9 events confirmés)**
**Statut : STRUCTURE CONFIRMÉE sur 1 match / NON GÉNÉRALISABLE en production**

#### Ce qui est confirmé

La structure melee existe bien dans la couche nibble-shiftée (NS), distincte des fire events.
La formule `player_index = b5 >> 4` est valide pour les melee events (même byte, même offset).

Structure melee (offsets depuis `mel_start` dans la couche NS) :

```
[0]  b0      : (b0 & 0x07) == 0x03   (low nibble = 3 — lead melee, haut nibble variable)
[1]  b1      : CONSTANTE PAR MATCH   (0x40 pour a974fdeb) ← seul discriminant melee/fire
[2]  b2      : compteur incrémental par joueur
[3]  b3      : 0x20                  (CONSTANT — discriminant structurel vs fire en NS)
[4]  b4      : 0x00                  (CONSTANT)
[5]  b5ctx   : 0x00                  (CONSTANT)
[6]  b6      : 0x0d                  (lead byte d'un fire event intégré)
[7]  b7      : 0x26                  (b1 fire event = fixe universel)
[8]  b5_ml   : (pi << 4) | slot      → player_index = b5_ml >> 4  ✓
[9]  b9      : 0x40–0x43             (b3 fire event, mask 0xFC == 0x40)
[10] fc      : fire_counter
[11] b5_fire : b5 du fire event associé (arme tenue par la victime ?)
[12:20] weapon_id (8 bytes big-endian, même format que fire events)
```

La structure contient un **fire event intégré** aux octets [6:20] (b6=0x0d, b7=0x26, b5_fire, weapon).
Il s'agit probablement du dernier fire event de la victime ou d'un contexte de synchronisation.

#### Problème fondamental — superposition fire/melee dans la couche NS

Un fire event situé exactement **6 bytes avant** `mel_start` satisfait toutes les mêmes
contraintes structurelles (`b3=0x20, b6=0x0d, b7=0x26, b9&0xFC=0x40`) parce que son
weapon (à l'offset +6 depuis sa propre start) tombe exactement à l'offset +12 depuis
`mel_start`.

Le byte `b1` est le **seul discriminant fiable** entre les deux types d'events.

#### Validation (inv#136 — a974fdeb, b1=0x40)

| Pi | Events melee trouvés | API melee_kills | Commentaire |
|----|--------------------|-----------------|-------------|
| 1  | 1 | 3 | sous-détection |
| 3  | 2 | 3 | sous-détection |
| 4  | 3 | 0 | faux positifs probables |
| 5  | 3 | 4 | sous-détection |
| 2,6,7 | 0 | 1+5+2=8 | non détectés |
| **Total** | **9** | **18** | **50% détection** |

Pi formula `b5>>4` confirmée cohérente avec `detect_pi_from_metadata` pour les 9 events trouvés.

#### Blocages identifiés

**Blocage 1 — b1 match-specific** : La valeur b1 est une constante par match (0x40 pour a974fdeb)
mais sa règle de calcul est inconnue. Sans elle, impossible de distinguer melee de fire.

| Match | Observation | b1 connu ? |
|-------|-------------|-----------|
| a974fdeb | 9 events avec b1=0x40 | ✓ (empirique) |
| d9329229 | 0 events — b3 ≠ 0x20 pour tous les candidats | ✗ |
| f2f81265 | 318 événements avec filtre fort seul (sans b1) | ✗ |

**Blocage 2 — b3 non constant sur d9329229** : La contrainte `b3=0x20` élimine l'intégralité
des candidats melee sur d9329229. Soit b3 varie selon la version du protocole, soit une
autre constante discrimine ces matchs.

#### Conclusion

**Ne pas utiliser inv136 en production.** inv135 (sentinel API) reste la seule solution
robuste pour identifier les kills melee/grenade.

La formule `pi = b5>>4` dans les melee events est validée structurellement mais ne peut
pas être exploitée sans connaître b1 a priori.

**Piste suivante** : Déterminer si b1 est lié à un champ accessible dans le manifest film
(playlist, map, version protocole). Si b1 est dérivable du contexte match, le blocage tombe.

Script de référence : `scripts/experimental/inv136_melee_marker.py`

---

### `fire_seq % n_players` = player_index (inv#132 — DÉCLASSÉ)
**Statut : solution précédente, remplacée par b5>>4**
**Date de validation originale : inv#132 (7/7 points) — implémenté en production le 2026-03-18**

#### Formule

```
byte[2]  =  fire_seq  =  player_index  +  life_number × n_players

→  player_index  =  fire_seq  %  n_players
```

`fire_seq` est le byte d'index `[2]` dans la structure nibble-shiftée d'un fire event
(noté `b2_stream` dans le code avant cette découverte).

#### Points de validation (inv#132)

| fire_seq (b2) | n_players | pi = b2 % n | Confirmé par |
|---|---|---|---|
| 1 | 8 | **1** | NS timeline cross-ref |
| 3 | 8 | **3** | NS timeline cross-ref |
| 6 | 8 | **6** | NS timeline cross-ref |
| 27 | 8 | **3** (27 % 8) | NS timeline cross-ref |
| 28 | 8 | **4** (28 % 8) | NS timeline cross-ref |
| 34 | 8 | **2** (34 % 8) | NS timeline cross-ref |
| 46 | 8 | **6** (46 % 8) | NS timeline cross-ref |

**7/7 ✓**

#### Invariants

- **Joueur POV** : `pi = 0` dans le système fire_seq (et dans PLAYER_METADATA — exclu par `detect_pi_from_metadata` qui filtre `pi != 0`)
- **`life_number`** : s'incrémente à chaque mort du joueur → `fire_seq` croît mais `% n_players` reste stable pour le même joueur
- **`n_players`** = nombre de joueurs humains du match (hors bots `bid(`)

#### Implémentation production (2026-03-18)

```python
# weapon_parser.py — scan_fire_events_all()
# Un seul scan avec le marqueur fixe (pi=1 donne 0x26 par coïncidence)
events = scan_fire_events_bitstring(chunk_data, 1, ts_fn)
for ev in events:
    ev["player_index"] = ev["fire_seq"] % n_players
```

```python
# weapon_extraction_service.py — _run_scan_phase()
n_players = len(all_participants) or 8
events = scan_fire_events_all(..., n_players=n_players)

# _process_match_inner() — POV player pi=0
if xuid and xuid.isdigit():
    pi_map.setdefault(int(xuid), 0)
```

```python
# _global_correlation.py — correlate_kills_global()
# Filtre par pi du tueur — chaque kill ne consomme QUE ses propres events
killer_pi = xuid_to_pi.get(kill["xuid"])
candidates = [
    (i, ev) for i, ev in enumerate(available)
    if (t_ms - KILL_WINDOW_MS) <= ev["timestamp_ms"] <= t_ms
    and (killer_pi is None or ev.get("player_index") == killer_pi)
]
```

---

---

---

## Ce que le film binaire NE permet PAS de détecter (état 2026-03-19)

### Kills grenade — non identifiables depuis le film

Il n'existe pas, à ce jour, de marqueur binaire identifié dans les chunks film qui
permette de détecter un lancer de grenade ou un kill grenade.

- Les fire events (`0d 26 ...`) correspondent aux tirs d'armes uniquement.
- Aucune structure analogue n'a été trouvée pour les grenades.
- **Conséquence :** un kill grenade n'a pas de fire event associé dans les chunks.
  Il apparaît donc comme `no_event` ou grand gap dans l'attribution film.

### Kills melee — non identifiables depuis le film (à ce jour)

Les kills melee ne produisent pas de fire event identifiable avec les patterns actuels.
Il est possible qu'un marqueur existe dans le binaire (le protocole réplique l'état
des joueurs en temps réel), mais il n'a pas encore été trouvé ni documenté.

- **Ce qui est connu :** l'utilisateur indique qu'une identification des melees dans
  les données brutes est théoriquement possible, mais la méthode n'est pas encore établie.
- **Ce qui est fait en attendant :** sentinel via counts API (voir section inv#135).

### Solution de contournement : sentinel via counts API

Puisqu'on ne peut pas identifier grenade/melee individuellement depuis le film,
on utilise les totaux `grenade_kills` + `melee_kills` de `match_participants` (API)
comme quota pour marquer les kills les plus incertains (plus grand gap film) :

- Pas de sur-compensation : le quota API borne exactement le nombre de sentinels.
- Résultat validé : `gun_diff = +0` sur 282 kills / 3 matchs.
- Limite : on ne peut pas assigner avec certitude QUEL kill individuel est grenade
  vs melee — seuls les totaux par joueur sont fiables.

---

## Cas edge documentés

### TypeRsamurai — pi=9 dans PLAYER_METADATA, absent de b5>>4

Sur match `a974fdeb` (9 joueurs humains), TypeRsamurai apparaît avec `pi=9` dans
PLAYER_METADATA. Or b5>>4 ne retourne que des valeurs 0–7 sur ce match (8 slots).
Résultat : 0 fire event trouvé pour TypeRsamurai, son unique kill → no_event → sentinel
grenade (API confirme : `grenade_kills=1`).

Hypothèse : joueur arrivé après le début du match (slot 9 dans PLAYER_METADATA mais
aucun slot fire event attribué au-delà de 7). Non confirmé, cas isolé.

**Comportement attendu du code :** pi=9 → pool vide → kill → no_event → sentinel. Correct.

---

### fire_counter wraparound — bug de dédup

Le `fire_counter` (b4, byte [+24..+31]) est un uint8 qui boucle à 255.
Sur un match long avec une arme automatique (ex. MA40 AR), le même `fire_counter`
revient toutes les ~256 rafales (~25 secondes à pleine cadence).

**Bug confirmé :** dédup par `(player_index, fire_counter, weapon_bytes)` supprimait les
occurrences ultérieures → KIN92 (19 kills MA40 AR) perdait des events légitimes,
avg gap gonflé à 3 secondes au lieu de 3ms.

**Fix :** dédup par proximité `byte_pos` uniquement (< 2 bytes = même event physique).

---

### Kills melee/grenade avec fire event <500ms avant

Sur 3 matchs, ~18% des kills non-gun API (45/282) ont un fire event dans la fenêtre
<500ms. Cela ne signifie **pas** que ces kills ont été mal détectés — cela signifie que
le joueur tirait juste avant de finir au poing ou à la grenade. Le film capture
correctement le dernier tir, mais le killing blow est non-gun.

Le sentinel inv135 corrige cela en utilisant le quota API comme vérité terrain.

---

## Récapitulatif chronologique

| Date | Événement |
|---|---|
| 2026-03-09 | FINDINGS inv#1–125 clôturés avec la fausse conclusion "POV only" |
| ~2026-03-16 | Début investigation : scan variants b0/b1 → marker fixe confirmé |
| 2026-03-16 | Théories 1, 2, 3 invalidées par scan exhaustif |
| 2026-03-17 | Théorie b2_stream via NS cross-ref (map_b2_to_player) : partielle, drops |
| 2026-03-17 | inv132 : formule `fire_seq % n_players` découverte, validée 7/7 |
| 2026-03-18 | Théorie 4 ("POV only") invalidée — confirmation acurtis |
| 2026-03-18 | Fix implémenté en production (`scan_fire_events_all` + `correlate_kills_global`) |
| 2026-03-18 | ERRATUM posé dans `FINDINGS_weapon_extraction_EN*.md` |
| 2026-03-19 | inv134 : b5>>4 découvert (doc acurtis 2026-03-18), validé 3 matchs 282 kills |
| 2026-03-19 | fire_seq%n déclassé en "approximatif" — b5>>4 devient solution définitive |
| 2026-03-19 | Fix dédup : par byte_pos proximity, pas par (fire_counter, weapon) |
| 2026-03-19 | inv135 : sentinel grenade/melee via counts API pour combler les 10.6% restants |
| 2026-03-19 | inv136 : structure melee NS confirmée sur a974fdeb (b3=0x20, pi=b5>>4 valide) |
| 2026-03-19 | inv136 : blocage identifié — b1 match-specific, b3 non constant sur d9329229 |
| 2026-03-19 | inv136 : production reste sur inv135 sentinel — inv136 en investigation ouverte |
