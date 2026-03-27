# Grenade Throws & Melee Detection — Technical Reference

> **Source de vérité** : méthode d'Andy Curtis (acurtis166), reverse engineering des frame packets
> filmshell Halo Infinite. Ce document décrit le protocole binaire de référence, les écarts avec
> l'implémentation actuelle de LevelUp, et les pistes d'amélioration envisageables.
>
> Dernière mise à jour : 2026-03-27

---

## Table of Contents

1. [Vue d'ensemble](#vue-densemble)
2. [Lancer de grenade — protocole acurtis](#lancer-de-grenade--protocole-acurtis)
   - [Marker et structure binaire](#marker-et-structure-binaire)
   - [Types de grenades (IDs 32 bits)](#types-de-grenades-ids-32-bits)
   - [Extraction du player index](#extraction-du-player-index)
3. [Melee — protocole acurtis](#melee--protocole-acurtis)
   - [Marker et ancre](#marker-et-ancre)
   - [Discrimination par type byte](#discrimination-par-type-byte)
   - [Offsets weapon selon type](#offsets-weapon-selon-type)
   - [Champs extraits](#champs-extraits)
4. [Écarts avec le code actuel](#écarts-avec-le-code-actuel)
5. [Pistes d'amélioration et stratégies à évaluer](#pistes-damélioration-et-stratégies-à-évaluer)

---

## Vue d'ensemble

Les filmshell Halo Infinite contiennent des **frame packets** (`PacketType.FRAME`, type 0, ~258 bytes,
~60 fps) qui encodent l'état de jeu image par image. Deux marqueurs distincts permettent de détecter
les actions physiques de grenade et melee directement dans ces flux binaires, indépendamment des
événements API (CoreStats, highlight events).

Ces deux marqueurs sont **orthogonaux** au marqueur fire events (`0b10100100110`) déjà utilisé dans
LevelUp pour l'attribution des kills par arme.

---

## Lancer de grenade — protocole acurtis

### Marker et structure binaire

- **Marker** : `0x4c0c00` (24 bits), non nécessairement aligné sur un octet.
- **Recherche** : scan bit-à-bit sur les données brutes du chunk, identique à la méthode
  `bits.findall(marker, bytealigned=False)` utilisée pour les fire events.
- **Structure après le marker** :

```
[marker 24 bits] [weapon_id 32 bits] ... [47 bits de padding] ... [player_index 5 bits]
```

Étapes d'extraction :
1. Localiser le marker `0x4c0c00` dans le chunk.
2. Lire les **32 bits suivants** → `weapon_id` (identifiant grenade 32 bits).
3. Valider le `weapon_id` contre la liste des types connus (voir ci-dessous).
4. Avancer de **47 bits** depuis la fin du `weapon_id`.
5. Extraire le **champ player_index de 5 bits**.

### Types de grenades (IDs 32 bits)

Andy utilise les IDs **32 bits** (pas les IDs filmshell 64 bits). Ces valeurs sont stables entre
les matchs et sont celles que l'on retrouve dans d'autres contextes de parsing Halo.

| Type | ID 32 bits |
|------|-----------|
| Fragmentation Grenade | `0xB0171062` |
| Plasma Grenade | `0xC0E34C44` |
| Shock Grenade | `0x3B2567D4` |
| Spike Grenade | `0x9212E428` |

> **Note** : Ces IDs 32 bits sont les **premiers 4 octets** des WIDs 64 bits filmshell. L'ID
> 64 bits frag (`0xb6dbead842c9679f`) est distinct et considéré moins fiable dans d'autres contextes
> que ces 32 bits. Seuls les 4 types ci-dessus ont été confirmés.

### Extraction du player index

Le champ player_index (5 bits) identifie le joueur qui lance la grenade. Il est cohérent avec le
`player_index` mappé via `PLAYER_METADATA` (méthode `detect_pi_from_metadata`).

---

## Melee — protocole acurtis

### Marker et ancre

- **Marker** : `0b10100110010` (11 bits), non nécessairement aligné sur un octet.

  > ⚠️ Ce marker est **différent** du marker fire events de LevelUp (`0b10100100110`).
  > Les bits 3 et 4 (en partant de la droite) diffèrent — il ne s'agit pas du même type d'événement.

- **Ancre** : avancer de **3 bits** dans le marker pour atterrir à l'octet `0x34` ou `0x35`.
  Cette ancre sert de point de référence pour tous les offsets relatifs ci-dessous.

### Discrimination par type byte

À **bit offset +76** depuis l'ancre, un `uint8` encode le type d'attaque :

| Valeur | Signification |
|--------|--------------|
| `0x42` | Melee non-propulsé (épée énergie non activée, ou arme normale) — miss/lunge |
| `0x47` | Marteau gravitationnel — powered hit ou miss |
| `0x60` | Épée énergie — powered hit ou melee non-propulsé hit |

Toute valeur en dehors de `{0x42, 0x47, 0x60}` doit être rejetée (false positive).

### Offsets weapon selon type

L'offset du `weapon_id` dans la structure varie selon le type byte. Ces offsets sont relatifs à
l'ancre :

```python
_WEAPON_OFFSETS = {
    0x42: [88],        # Non-powered miss (sword non-activated, ou arme normale)
    0x47: [86],        # Gravity hammer powered hit ou miss
    0x60: [
        101,           # Energy sword powered hit
        103,           # Non-powered hit
    ],
}
```

Pour `0x60` (épée énergie), les deux offsets `101` et `103` doivent être testés séquentiellement ;
celui qui produit un WID valide est retenu.

### Champs extraits

Tous les offsets sont relatifs à l'**ancre** (marker + 3 bits) :

| Champ | Offset (bits) | Taille | Description |
|-------|--------------|--------|-------------|
| `player_index` | 20 | variable | Index du joueur qui attaque |
| `weapon_id` | voir `_WEAPON_OFFSETS` | 32 bits | Arme utilisée |
| `weapon_variant` | suit `weapon_id` | présent si applicable | Variant cosmétique (après weapon_id si l'arme en possède un) |
| `animation_type` | nibble avant `weapon_id` | 4 bits | Type d'animation (nibble précédant le weapon_id) |
| `type` | 76 | 8 bits | Discriminant d'attaque (`0x42`/`0x47`/`0x60`) |

> **Interprétation hit/miss** : Andy note que la distinction "hit" vs "miss" pourrait en réalité
> encoder si le joueur a effectué une **lunge** (mouvement d'attaque). La sémantique exacte n'est
> pas confirmée définitivement.

---

## Écarts avec le code actuel

### 1. Marker grenade : inconnu du pipeline actuel

Notre pipeline ne contient aucune référence au marker `0x4c0c00`. Les grenades sont actuellement
traitées par **inférence de médailles** dans `_weapon_kills_repo.py` :

```python
# Approche actuelle (inférence)
is_grenade = any(m in GRENADE_MEDALS for m in medals_nearby)
# GRENADE_MEDALS = {"Sticky Fingers", "Grenadier", "Boom!", "Kong", "Stick", "Grenade Stick"}
```

Conséquences :
- On ne détecte que les lancers qui **obtiennent une médaille** (et donc généralement un kill).
- Le type de grenade (Frag / Plasma / Shock / Spike) est **inconnu**.
- Les lancers sans kill sont **invisibles**.
- Faux positifs possibles si une médaille grenade arrive dans une fenêtre de ±500 ms d'un kill
  par une autre arme.

### 2. Marker melee : différent du marker fire events

Notre marker universel (`0b10100100110`) capture les événements fire (tir d'arme). Le marker melee
d'Andy (`0b10100110010`) est distinct — il capture les **animations d'attaque melee**.

Actuellement, les melees sont inférés via médailles :

```python
# Approche actuelle (inférence)
is_melee = any(m in MELEE_MEDALS for m in medals_nearby)
# MELEE_MEDALS = {"Pummel", "Assassination", "Back Smack", "Melee", "Quigley", "Ninja", "Pancake"}
```

Conséquences :
- Aucune distinction entre **melee de base** et **arme melee propulsée** (épée/marteau).
- Les swings dans le vide (miss ou lunge) sont **invisibles**.
- `weapon_id` melee dans `weapon_kills` est toujours le sentinel `MELEE_WEAPON_ID = 1`,
  sans information sur l'arme réelle.

### 3. Format des IDs de grenade : 32 bits vs 64 bits

Notre `WEAPON_ID_MAP` utilise des WIDs **64 bits** (`bytes[8]` → UBIGINT). Andy confirme que pour
les grenades, les IDs 32 bits sont **plus stables** dans d'autres contextes. Le WID 64 bits frag
actuellement listé dans `_weapon_data.py` (`0xb6dbead842c9679f`) est marqué `# (unconfirmed)`.

Les IDs 32 bits d'Andy ne correspondent pas directement au schéma actuel de `WEAPON_ID_MAP` : ils
ne contiennent pas le suffix `42c9679f` et ne font pas 8 octets.

### 4. Granularité des données

| Dimension | Approche actuelle | Protocole acurtis |
|-----------|-------------------|------------------|
| Lancers sans kill | Non capturés | Capturés |
| Type de grenade | Inconnu | Frag / Plasma / Shock / Spike |
| Swings melee manqués | Non capturés | Capturés |
| Arme du melee | Sentinel générique (1) | weapon_id réel |
| Melee basic vs propulsé | Indistinguible | Distingué par type byte |

### 5. Couverture joueurs

Le scan fire events actuel (Section 2) ne couvre que le **joueur POV** (player_index = 1 dans le
film). Formula A couvre les autres via snapshot. Les markers grenade et melee d'Andy extraient le
`player_index` directement depuis le packet — la couverture multi-joueurs est donc potentiellement
**native** pour ces deux événements, sous réserve que tous les player_index soient présents dans
les frame packets (à valider).

---

## Pistes d'amélioration et stratégies à évaluer

### Piste A — Scanner de lancers de grenade (priorité : moyenne)

**Principe** : ajouter `scan_grenade_throws(chunk_data, estimate_ts)` dans
`src/analysis/_weapon_scanners.py`, sur le modèle de `scan_fire_events_b5`.

Points à évaluer :
- **Faux positifs** : valider le taux de faux positifs du marker `0x4c0c00` sur un échantillon
  de chunks (quelques matchs diversifiés). Le marker 24 bits est plus long que le marker 11 bits
  fire events → a priori moins de collisions.
- **Densité** : mesurer combien de throw events on récupère par match et les comparer aux stats
  API `GrenadeKills` (lower bound) et aux médailles grenade.
- **Remplacement ou complément** : à terme, les throw events pourraient remplacer l'inférence
  médaille pour les kills grenade (meilleure précision sur le type). Transition progressive :
  garder les deux en parallèle pendant une période de validation.

**Bénéfice potentiel** :
- Grenade throw rate par type (Plasma > Frag ? Shock throw frequency ?)
- Grenade efficiency : ratio kills / throws
- Stats de team : qui contribue le plus aux grenades de zone ?

### Piste B — Scanner de swings melee (priorité : basse)

**Principe** : ajouter `scan_melee_swings(chunk_data, estimate_ts)` en cherchant
`0b10100110010` et en appliquant la discrimination par type byte.

Points à évaluer :
- **Ambiguïté hit/lunge** : Andy lui-même indique que la sémantique exacte n'est pas confirmée.
  Avant d'afficher ces données dans l'UI, il faudra valider sur des cas connus (replay + stats API).
- **Offsets multiples pour `0x60`** : la logique de fallback entre offsets `101` et `103` introduit
  une complexité de validation supplémentaire.
- **Volume de données** : le melee est relativement rare par rapport aux fire events. Impact DB
  limité.

**Bénéfice potentiel** :
- Melee swing accuracy (hits / swings)
- Distinction épée énergie activée vs melee basique (pour les stats d'assassinats)
- Animations lunge vs attaque normale

### Piste C — Enrichissement de `weapon_kills` pour les grenades (priorité : haute si Piste A validée)

Une fois le scanner grenade validé, les entrées `weapon_kills` avec `weapon_id = GRENADE_WEAPON_ID`
(sentinel 0) pourraient être enrichies avec le type réel de grenade.

Options de schéma :
1. **Nouvelle colonne `grenade_type VARCHAR`** dans `weapon_kills` : nullable, renseignée
   uniquement quand `weapon_id = GRENADE_WEAPON_ID`.
2. **Nouveaux `weapon_id` distincts** par type de grenade (ex. `FRAG_GRENADE_ID = 10`,
   `PLASMA_GRENADE_ID = 11`, …) pour préserver la cohérence du champ `weapon_id`.
3. **Nouvelle table `grenade_throws`** dédiée (lancers bruts, sans lien nécessaire avec un kill) —
   la plus expressive mais aussi la plus de travail d'intégration UI.

L'option 2 est probablement la plus compatible avec le pipeline existant (même sentinel pattern,
même aggregations).

### Piste D — Validation croisée (indispensable avant toute écriture DB)

Avant d'intégrer ces scanners dans le pipeline de sync, effectuer une validation sur un
**ensemble de matchs contrôlés** :

1. Choisir 10–20 matchs avec des kills grenade/melee connus (via `medals_earned` +
   `weapon_kills`).
2. Extraire les throw/swing events via les nouveaux scanners.
3. Vérifier que :
   - Chaque kill grenade connu a au moins un throw event dans une fenêtre temporelle raisonnable
     (ex. −3000 ms à −100 ms avant le kill).
   - Le `player_index` des throw events correspond bien au killer.
   - Le faux positif rate (throws détectés sans kill ni médaille grenade) est interprétable
     (lancers légitimes sans kill).
4. Documenter les résultats dans `.ai/thought_log.md` avant tout merge.

### Piste E — Harmonisation des IDs grenade 32 bits / 64 bits

Évaluer si les IDs 32 bits d'Andy (`0xB0171062`, etc.) correspondent aux 4 premiers octets des
WIDs 64 bits actuels dans `WEAPON_ID_MAP`. Si oui, on peut dériver les IDs 32 bits depuis les
entrées existantes sans créer de nouvelle structure.

Le WID frag 64 bits actuel `0xb6dbead842c9679f` a pour premiers 4 octets `b6dbead8` — ce qui ne
correspond **pas** à `0xB0171062`. Deux hypothèses :
- Le WID 64 bits frag actuel est incorrect (il est déjà marqué `# (unconfirmed)`).
- Les grenades ont deux représentations filmshell différentes selon le contexte (inventory
  snapshot vs throw event).

Cette ambiguïté doit être levée avant toute intégration.

---

*Référence externe : [commentaire acurtis166 sur GitHub](https://github.com/acurtis166/SPNKr/issues) — grenade throws et melee frame packets.*
