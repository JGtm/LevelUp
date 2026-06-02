# Recherche — Rétro-ingénierie Theater / Films Halo Infinite

> Document de recherche exploratoire. Mis à jour 2026-06-01.
> Sources primaires : den.dev blog + github.com/dend/blog-comments/issues/5 (94 commentaires)

---

## Résumé exécutif

Le format des fichiers film Halo Infinite est **partiellement documenté et largement exploitable**.
La communauté (dend, acurtis166, JGtm…) a déjà accompli l'essentiel du travail de RE.
**SPNKr** — librairie déjà utilisée par LevelUp — **contient déjà un module film**.
Les fichiers sont téléchargeables via l'API Waypoint, pas uniquement disponibles en local.

---

## Ce que contiennent les fichiers film

Les films ne sont **pas de la vidéo** — ce sont des métadonnées du moteur de jeu (format propriétaire Slipspace/Blam).

### Données accessibles aujourd'hui

| Donnée | Disponibilité | Notes |
|--------|:------------:|-------|
| Gamertags + XUIDs de tous les joueurs | ✅ | Chunks type 1-2 |
| Kills / morts avec timestamp précis (ms) | ✅ | Chunk type 3 |
| Médailles (>180 types) avec timestamp | ✅ | Chunk type 3 |
| Événements mode-spécifiques (captures, objectives) | ✅ | Type hint 10 |
| Vecteurs de visée (aim vectors) | ✅ | Chunks de réplication, décodage cubemap |
| Positions des joueurs | ⚠️ | Présentes mais = engine state, pas coordonnées cartésiennes brutes |
| Événements arme (tir, recharge, pickup) | ⚠️ | Partiellement décodés |
| Type d'arme par kill | ❌ | Non résolu (recherche en cours dend) |
| Trajectoires grenades | ❌ | Très basique |
| Inventaire live joueur | ❌ | Partiellement supporté |

### Types d'événements décodés (type_hint)

| Valeur | Signification |
|--------|--------------|
| 10 | Événements mode-spécifiques (flag captures, objective kills) |
| 20 | Événements de mort |
| 50 | Événements de kill |
| 51+ | Médailles (mappent aux métadonnées API) |

---

## Format binaire des fichiers film

### Structure générale

Les films sont divisés en **chunks** numérotés (`filmChunk0`, `filmChunk1`… `filmChunk22`).

| Type de chunk | Contenu |
|---|---|
| Type 1 | Métadonnées de bootstrap du jeu |
| Type 2 | Captures d'événements in-game |
| Type 3 | Résumé (summary) + événements temporels |
| Chunks de réplication | Positions, vecteurs de visée, état du moteur |

### Compression

- **Méthode** : zlib (RFC 1950) — décompression standard
- **Header identifiable** : `78 5E`
- **Python** : `zlib.decompress(compressed_data)` — c'est tout

### Structure d'un événement (chunk type 3)

```
[12 octets]  — header
[32 octets]  — Gamertag (Unicode)
[15 octets]  — padding
[ 1 octet ]  — type d'événement
[ 4 octets]  — timestamp (little-endian, millisecondes depuis début du match)
[ 3 octets]  — padding
[ 1 octet ]  — marqueur médaille
[ 3 octets]  — padding
[ 1 octet ]  — ID type de médaille
```

### Localisation des joueurs dans les chunks type 1-2

Pattern marqueur : `0x2D 0xC0`

```
[32 octets]  — Gamertag (Unicode)
[21 octets]  — padding
[ 8 octets]  — XUID
[ 2 octets]  — marqueur 0x2D 0xC0
```

### Piège critique : données non byte-aligned

Les données ne sont **pas alignées sur les limites d'octets**. Parsing au niveau du bit requis (pas hex standard).

### Timestamp — conversion

```python
# Les 4 octets sont en little-endian, inverser avant conversion
timestamp_bytes = bytes(reversed(raw_4_bytes))
timestamp_ms = int.from_bytes(timestamp_bytes, 'big')
```

### Vecteurs de visée (aim vectors) — décodage cubemap

```python
_FACE_SIZE = ...  # constante du moteur

def decode_aim_vector(encoded_aim):
    face_index, raw_coordinate = divmod(encoded_aim, _FACE_SIZE)
    p = 2.0 * (float(raw_coordinate) / (_FACE_SIZE - 1)) - 1.0
    # Projection cubemap → vecteur unitaire [x, y, z]
    # Implémentation complète : spnkr/film/replication_data.py
```

### ComponentObjects (données de réplication décompressées)

Les chunks de réplication contiennent **1033 ComponentObjects** avec des noms lisibles :
- `PlayerWaypointComponent`
- `ManagedObjectNavpointComponent`
- `WeaponStateAmmoComponent`

Header : 256 octets (nom du composant) + 4 octets (valeur).

---

## Le problème fondamental des chunks de réplication

Les chunks de réplication contiennent des bytes bruts associés à des ComponentObjects.
Mais le film ne décrit pas leur structure — il suppose que le lecteur la connaît déjà.

**Les fichiers du jeu sont la Rosetta Stone.**

```
[Film .bin]                        [Fichiers .module du jeu]

WeaponStateAmmoComponent   →       WeaponStateAmmoComponent {
?? ?? ?? ?? ??                         float current_ammo;     // 4 bytes
?? ?? ?? ?? ??                         float reserve_ammo;     // 4 bytes
?? ??                                  bool  is_reloading;     // 1 byte
                                       ...
                                   }
```

Sans le schéma, les bytes sont illisibles. Avec le schéma, chaque valeur devient
interprétable et validable contre des données connues du match.

## Le runtime-tagviewer — chaînon manquant (pas juste "utile")

Quand Theater rejoue un film, le moteur charge en mémoire **les schémas de tous les
ComponentObjects** pour pouvoir lire les chunks de réplication. Ces schémas sont donc
disponibles en mémoire le temps du replay — c'est exactement ce que runtime-tagviewer
peut exposer.

**Workflow concret :**

```
1. Film .bin  →  décompresser  →  trouver "WeaponStateAmmoComponent" à offset X
2. Lancer Theater sur ce film  →  attacher runtime-tagviewer
3. Trouver le schéma de WeaponStateAmmoComponent en mémoire
4. Appliquer le schéma aux bytes à l'offset X  →  données interprétées
5. Valider contre ce qu'on sait du match (munitions à T+Xs, etc.)
```

Les fichiers `.module` sur disque (Reclaimer / infinite-rs) sont une alternative
hors-ligne, mais le moteur en Theater est le chemin le plus direct — il a déjà résolu
toutes les dépendances de types.

**Rôles révisés des outils :**

```
runtime-tagviewer  → schémas ComponentObjects en mémoire live   ← priorité 1
Reclaimer          → même chose mais depuis les .module sur disque
infinite-rs        → référence de format bas niveau (source Rust)
Film .bin local    → données brutes à corréler avec les schémas
```

---

## API Waypoint — endpoints film

Les films sont téléchargeables directement, **pas besoin d'avoir les fichiers localement**.

```
# 1. Métadonnées + liste des chunks
GET https://discovery-infiniteugc.svc.halowaypoint.com/hi/films/matches/{MATCH_ID}/spectate

# 2. Télécharger un chunk
GET https://blobs-infiniteugc.svc.halowaypoint.com/ugcstorage/film/{FILM_ASSET_ID}/{VERSION_ID}/filmChunk{INDEX}
```

---

## Correction importante : le décalage de 30 secondes (contribution JGtm)

> *Découverte documentée dans l'issue #5 par JGtm (mai 2026)*

**Problème** : le `StartTime` de l'API Waypoint **n'est pas le vrai début du match**.

Il y a ~30 secondes de préambule non comptabilisées :
- Intro de carte
- Cinématique d'apparition
- Décompte de lancement

Ces 30s sont "pliées" dans le `StartTime` API, ce qui décale tous les timestamps film.

**Solution** : utiliser `players.first_joined_time` (pour les joueurs avec `present_at_beginning = true`) comme référence temporelle réelle.

Exemple validé (Match Fortress, 2026-03-31, match_id `41b61fb9-3d71-40b7-bde7-45682fba6d57`) :
- API StartTime → film t=0
- Premiers kill/death dans highlight_events : ~34-35s
- Mort réelle dans le film : ~4-5s
- Décalage constaté : ~30s

---

## Ce qui existe déjà

### SPNKr (librairie Python déjà dans LevelUp)

**SPNKr contient déjà un module film** — c'est la découverte clé.

```python
from spnkr.film.highlight_events import ...   # événements kill/death/médailles
from spnkr.film.replication_data import ...   # positions, aim vectors
```

Implémentation par **acurtis166** (contributeur principal de SPNKr).

### films.openspartan.com (lancé avril 2026)

Site de démonstration fonctionnel par **dend** :
- Visualisation 2D des mouvements de joueurs en temps réel
- Kill feed avec timeline
- Graphiques K/D over time
- Killer-victim counts avec jointure temporelle
- Heatmaps positionnelles (en développement)

### OpenSpartan/film-event-extractor

Outil C# open source de parsing + agrégation (actuellement vers SQLite).

---

## Données disponibles localement

Tu as accès à des fichiers `.bin` en local sur ton ordi principal — ce sont probablement les chunks film déjà téléchargés. Format identique aux chunks API (`filmChunkN`), compressés zlib.

---

## Limitations connues

- Certains matchs n'ont pas de chunks type 3 (bug API)
- Les chunks type 3 ne contiennent pas toujours les gamertags de façon cohérente
- Les positions ne sont **pas** des coordonnées cartésiennes brutes — c'est de l'engine state Slipspace
- Type d'arme par kill : non résolu
- Mécanisme des assists : non documenté

---

## Plan de recherche

SPNKr est connu et maîtrisé — hors scope. L'objectif est les chunks de réplication non parsés.

### Étape 1 — Explorer les .bin locaux (Python + ImHex)

1. Installer **ImHex** (éditeur hex avec support de patterns)
2. Décompresser les chunks zlib et dumper les chaînes lisibles
3. Lister tous les ComponentObjects présents dans les chunks de réplication
4. Identifier ceux qui semblent contenir des données de position / état joueur

### Étape 2 — Obtenir les schémas via runtime-tagviewer

1. Installer runtime-tagviewer
2. Lancer Theater sur un match dont on connaît les .bin locaux
3. Attacher runtime-tagviewer au processus HaloInfinite.exe
4. Trouver les schémas des ComponentObjects intéressants en mémoire
5. Documenter : nom du composant → liste de champs (type + taille + offset)

### Étape 3 — Corréler schémas + bytes

1. Appliquer le schéma trouvé aux bytes du .bin à l'offset connu
2. Valider les valeurs contre des données connues du match
   (position approximative, munitions, état à un timestamp précis)
3. Itérer jusqu'à avoir une interprétation cohérente

### Étape 4 — Écrire le parser Python

Une fois les schémas validés :
1. Implémenter le décodage des ComponentObjects en Python
2. Extraire un jeu de données test sur quelques matchs connus
3. Évaluer la qualité et le périmètre avant toute intégration LevelUp

### Potentiel par priorité

| Donnée cible | Dépend de | Valeur LevelUp |
|---|---|---|
| Positions joueurs (heatmaps) | Schéma PlayerWaypointComponent | Très haute |
| État arme par moment | Schéma WeaponStateAmmoComponent | Haute |
| Vecteurs de visée précis | Déjà partiellement dans SPNKr | Moyenne |
| Trajectoires projectiles | Composant inconnu à identifier | Haute |

---

## Contributeurs clés à suivre

| Pseudo | Rôle | Liens |
|--------|------|-------|
| **dend** | Blog + films.openspartan.com + OpenSpartan | [den.dev](https://den.dev) |
| **acurtis166** | Auteur SPNKr + modules film | [github.com/acurtis166/SPNKr](https://github.com/acurtis166/SPNKr) |
| **JGtm** | Découverte décalage 30s preambule | Issue #5 |

---

## Ressources

| Ressource | URL | Intérêt |
|---|---|---|
| Blog den.dev film | https://den.dev/blog/extracting-stats-film-files-halo-infinite/ | RE complet |
| Issue #5 (94 commentaires) | https://github.com/dend/blog-comments/issues/5 | Détails techniques |
| SPNKr film module | https://github.com/acurtis166/SPNKr/tree/master/spnkr/film | Code Python ready |
| films.openspartan.com | https://films.openspartan.com | Démo live |
| OpenSpartan film-event-extractor | GitHub OpenSpartan/film-event-extractor | Référence C# |

---

## Exploration locale + validation corpus — 2026-06-01

> Analyse offline des chunks en cache (`data/cache/film_chunks/`) avec un outil Go ad hoc
> (`tmp_film_explore/filmx.exe`, stdlib only, modes : `header allstr gren melee xuids he kf fire`).
> Validation sur **12 matchs** via workflow (download type-3 frais + scans + baselines adverses).
> Recoupé avec l'issue #5 de dend/acurtis166 (sauvegarde locale).

### A. Chunk HEADER (type 1) = registre de schéma ECS — non parsé par LevelUp

Le `chunk_00` décompresse en ~1.97 MB et contient un **registre de ~200+ types de composants**
(slots de 260 octets : préfixe `01 00 00 00` + nom ASCII null-paddé sur 256). C'est la table des
matières exacte de tout l'état répliqué. Confirme le « 1033 ComponentObjects » de dend. Veines
à haute valeur, jamais extraites (bloquées sur le schéma des valeurs = fichiers `.module` /
runtime-tagviewer) :

- Spatial : `object-position-component`, `object-translational-velocity-component`,
  `object-forward-and-up-component`, `unit-desired-aiming-vector-component`
- Vitalité : `object-body-vitality-component`, `object-shield-vitality-component`,
  `object-damage-sections-component`, `object-region-state-component`, `object-dead-state-component`
- Armes/abilities : `weapon-ammo-component`, `unit-grenade-counts-component`,
  `biped-spartan-ability-energy-component`, `unit-active-camo-state-component`, `player-engine-loadout-component`
- Stats/jeu : `statborg-current-round-value-stat-component` (28×),
  `statborg-finalized-rounds-values-stat-component` (28×), `managed-objective-*` (système objectifs),
  `supply-lines-*` (économie), `player-lives-remaining-component`, `player-last-betrayer-component`
- Véhicules : suite `vehicle-*` complète

Le header contient aussi le **roster joueurs** (clusters de XUIDs proches + gamertags).
Les bots ont un `bid(` ~109 octets derrière leur nom (observation dend, à recâbler).

### B. Replication (type 2) — IDs = handles d'objets, pas XUIDs

Scan plausible-XUID : valeur dominante `2251799813685248` = 2^51 (984× dans un chunk) = constante
structurelle (handle/masque), pas un XUID. Le mapping joueur↔handle vit dans le header.
Stride périodique fort détecté (autocorrélation 0.993 sur 79 octets et multiples) — structure non
identifiée. Triplets de floats plausibles présents (coordonnées monde candidates).

### C. Marqueur `0x4c0c00` = marqueur object/equipment-id GÉNÉRAL (pas « grenade »)

Validé sur corpus (12 matchs, baselines adverses). Le marqueur 24 bits est enrichi 20-70× vs hasard
(~0.4 hit attendu/chunk vs 8-59 observés) ; le `weapon_id` 32 bits est verrouillé à `marqueur+24`
(contrôle off-by-one à +23/+25 = 0 valide). **MAIS** : ce n'est pas un marqueur de lancer de grenade.
Les hits « invalides » décodent en **object-ids 32 bits stables récurrents cross-match**
(`0x18e1fea0` ×18 sur 3 matchs, `0xb2a8143a` ×56, `0xfa9eba93` ×30). Les grenades sont le
sous-ensemble étiquetté par l'allowlist {Frag `0xB0171062`, Plasma `0xC0E34C44`, Shock `0x3B2567D4`,
Spike `0x9212E428`}. Corpus : 28 lancers valides (Frag 17, Plasma 5, Shock 4 ; **Spike jamais vu**),
valides répartis sur 5/12 matchs.

- **Confiance HAUTE** qu'un hit valide = vraie grenade (faux positifs ~0 parmi les valides).
- **Confiance BASSE** sur la complétude : scan d'un seul chunk type-2 (chunk_03) → sous-comptage ;
  les markers sont sparse / dans d'autres chunks. Avant prod : scanner tous les chunks type-2 +
  décider d'énumérer le namespace object-id complet plutôt qu'une allowlist 4-grenades.

### D. Marqueur melee `0b10100110010` — NON fiable en l'état

Corpus : 109 valides / 6466 bruts = **1.69 %** (~98 % de bruit). Le filtre `typeByte ∈ {0x42,0x47,0x60}`
laisse passer des `player_index > 15` (ex. pi=23, pi=18) directement dans le « valide ». Le marqueur
11 bits sur-déclenche. Inutilisable sans marqueur plus long/spécifique + filtre `pi ≤ 15` obligatoire.
NB : dend mentionne aussi un marqueur melee `0xd340` (à explorer).

### E. Kill-feed (killer→victime) — reconstructible par jointure temporelle, ROBUSTE

> **Précision (correction utilisateur 2026-06-01)** : le *kill feed affiché à l'écran*
> (Tueur — arme — victime, fond sur les assists, couleurs allié/ennemi du joueur) est une
> fonctionnalité ajoutée des années après le lancement et est **reconstruit côté client au replay**
> — ce n'est PAS une structure stockée dans le film. Le fait que les couleurs dépendent des réglages
> locaux du joueur le confirme. Ce qu'on extrait ci-dessous (jointure kill+death) donne killer→victime
> **mais SANS l'arme**. Vérifié : le chunk type-3 ne contient **aucun weapon-id** (suffixe 64-bit
> `42c9679f` = 0 au bit près ; marqueur object-id `0x4c0c00` = 0 ; région étendue par event = state
> bit-packé sans id d'équipement). L'arme du kill vient donc obligatoirement de la replication (§G).

Chunk type-3 (highlight events), 12 matchs : **1517/1533 kills (99.0 %)** s'apparient à une death
au `time_ms` **identique** (dt≈0, la tolérance ±150ms n'est qu'une marge de jitter). Baseline nulle
(timestamps indépendants) ~6.4 % → écart ~15.5×, Z~148σ, p≪1e-100. C'est un lien structurel
(les deux markers émis par le même événement moteur), pas une proximité fortuite. dend décrit la
même méthode (« join… using a 5 ms tolerance »).

- **Pas de paire de XUID adjacente** : les XUIDs (8 octets LE) n'apparaissent que « marqués »
  (`…2d|25 c0`), 1 par event = l'acteur. 0 XUID brut, 0 co-occurrence < 320 octets dans le type-3.
- **kills > deaths dans les 12 matchs** (+20 % agrégé) : th=50 sur-compte (assist/multikill/crédits
  dupliqués) ou morts sans tueur (suicide/chute/environnement/trahison). Prod : dédupliquer par
  `(time, killer, victim)`, ne pas forcer de victime sur les ~1 % non appariés.

> **Vérif par-joueur vs API (000d5950, 2026-06-01)** : les **deaths (th=20) du film == `match_participants.deaths` API EXACTEMENT** pour les 8 joueurs (total 93 = total kills API = le score). Les **kills (th=50) sur-comptent même après dédup `(xuid,time_ms)`** (total film 101 vs 93 API ; surplus +0 à +3 par joueur). Conclusion : **le signal mort (th=20) est autoritatif** (= score), le marqueur kill (th=50) est un flux plus riche/bruité. **Reco kill-feed** : ancrer sur les DEATH events (th=20, fiables), le tueur = le th=50 au même `time_ms` (attribution last-damage). Ceci explique le comportement jeu observé : une mort en zone interdite (OOB) déclenche bien un th=20 (victime) + un th=50 (dernier attaquant crédité) — le score ET le kill feed lisent le MÊME crédit last-damage (pas une coïncidence : le feed, ajouté tard, est un rendu UI des events de crédit qui pilotaient déjà le score). L'arme affichée = la dernière arme du tueur ayant infligé des dégâts (d'où l'arme "héritée" sur les morts environnementales).

### F. Assists — pas d'event dédié

type_hint observés en type-3 : 10 (mode), 20 (death), 50 (kill), 100/150/200 (médailles). **Aucun
type d'event assist**. dend confirme : « assists are encompassed in the event envelope for a kill…
there is no dedicated assist event ». Non extractible proprement sans décoder les octets étendus
par event (record ~3485 octets/event, ~3400 non décodés) ou via l'API (`match_participants.assists`).

### G. Arme/outil du kill — ABSENT du killfeed (type-3) ; vit dans les fire events (type-2)

Dans le type-3 : suffixe weapon `42c9679f` = **0**, marqueur grenade = **0**. Le bloc d'event (60 o)
= gamertag acteur + type + `time_ms` + champs non décodés : `b[36]∈{0,1,2,3}` uniforme (flag 2 bits),
`b[37]/b[38]` binaires, `b[59]` = medal_type (deaths toujours =1 ; kills =0 sauf ~17 % portant un
**code médaille** : 108, 105, 127… → indice indirect d'outil pour les kills médaillés seulement).

L'arme vient des **fire events (type-2)** — pipeline `weapon_kills` existant. Structure acurtis :
```
0d 26 01 40 08 11  6acdc44d42c9679f  64 9b cb 14 e0 ...
                   └── weapon 8B ──┘  └ aim vector (octaédrique) ┘
```
LevelUp extrait déjà weapon + player_index (`b5>>4`) + burst (2e bit) + hit/miss (3e bit).
**NON capturé** : le **vecteur de visée** (octant + uint16, encodage octaédrique 3D→2D) et
« the target object/player is likely encoded » après le weapon id = **la victime potentiellement
encodée dans le fire event** → lierait arme→victime directement (piste ouverte, non résolue par la
communauté). L'**ammo** est lisible près des fire events / via `weapon-ammo-component`.

### Priorités exploitables sans la « Rosetta Stone » (schéma .module)

1. **Kill-feed film-natif** (jointure th50/th20 sur `time_ms`) — robuste, prêt à coder.
2. **Lancers de grenade par type** (allowlist sur marqueur object-id) — réel mais incomplet,
   scanner tous les chunks type-2.
3. **Vecteur de visée** depuis les fire events déjà parsés (octets post-weapon) — décodage octaédrique.
4. **Cible/victime dans le fire event** (« target object likely encoded ») — lierait arme→victime
   sans jointure temporelle ; piste à creuser.

Outil d'exploration : `tmp_film_explore/filmx.exe <chunk.bin> <mode>` (throwaway, hors build app).

### H. Médailles dans le type-3 = code film 1 octet b[59], mappable → name_id API (validé end-to-end)

Chaque event `is_medal` (b[55]==1) du type-3 porte, en plus de (xuid, time_ms), un **code médaille 1 octet `b[59]`**. Ce code n'est PAS le `medal_name_id` Halo (32-bit) ni `name_id & 255` (vérifié faux : 622331684 & 255 = 36, pas un code observé). C'est une **énumération film interne stable**. Le **count d'events is_medal par joueur côté film == count `medals_earned` API par joueur** (hors id synthétique LevelUp `9000000001`), ce qui permet de résoudre la bijection `b[59] → medal_name_id` par alignement des multisets par joueur. Bijection résolue sur 000d5950 (croisée avec `medals_earned` + `medal_definitions`) :

| b[59] | medal_name_id | médaille | weapon-specific ? |
|---|---|---|---|
| 0 | 622331684 | Double Kill | non (multikill) |
| 141 | 1176569867 | Yard Sale | non (style) |
| 100 | 1734214473 | Whiplash | non (équipement) |
| 108 | 4229934157 | **Snipe** | **oui → S7 Sniper** |
| 114 | 2602963073 | **No Scope** | **oui → S7 Sniper** |
| 91 | 87172902 | Odin's Raven | non |
| 105 | 2123530881 | Reversal | non |
| 75 | 1210678802 | Warrior | non (proficiency) |
| 127 | 2625820422 | From the Grave | non (style) |
| 72 | 1146876011 | Bomber | non (proficiency) |
| 85 | 3114137341 | **Bulltrue** | contexte épée (victime en lunge) |
| 101 | 3546244406 | Kong | non |
| 109 | 1512363953 | Perfect | précision (BR/no-miss) |

Autres médailles du match (ambiguës dans la bijection 1-match, à lever cross-match) : **Stick** (3655682764 → grenade plasma/spike), **Bank Shot** (2414983178 → Cindershot ricochet), **Back Smack** (548533137 → melee dos), Breacher, From the Grave, Tag & Bag, Guardian Angel, Killing Spree, Killjoy.

**Validation end-to-end** : Player4 (2533274882097883) a film b[59] {108×2, 114×1} = Snipe×2 + No Scope×1 ; `weapon_kills` le crédite de **S7 Sniper × 4 kills**. Player7 idem. → une médaille weapon-specific, **avec son `time_ms` issu du film**, confirme l'arme du kill à cet instant.

**Valeur pour la fiabilisation de l'arme** : l'API `medals_earned` donne déjà le `name_id` réel mais **pas de timestamp** (totaux only). Le film ajoute le **`time_ms` par médaille** → permet d'ancrer une médaille weapon-specific (Snipe, No Scope, Stick, Bank Shot, Bulltrue…) sur un kill précis et de **confirmer/corriger** l'attribution fire-event pour ce kill. Pipeline recommandé : identité médaille via API (`name_id`), timing via film, alignement par joueur (trivial si une seule médaille weapon-specific par joueur ; sinon via la bijection b[59]).

### I. Cible/victime dans le fire event (piste A) — localisée, NON crackée (open)

Tentative de décoder « the target object/player is likely encoded » (acurtis) après le weapon id, pour lier tireur→arme→victime sans jointure temporelle. État :

- Les octets post-weapon des fire events réels (suffixe `42c9679f`) contiennent l'**aim vector** (préfixe stable par (joueur, arme) + composante directionnelle variable ; ex. pi=6 S7 Sniper → `55 47 08 40 …` constant puis variable) suivis d'octets d'état.
- **Bloqueur** : la corrélation précise fire-event ↔ kill ↔ joueur bute sur le **chaos des player_index** (cf. [[.ai/PLAYER_INDEX_FIRE-EVENTS_RESOLUTION.md]]) : pour le kill sniper de Player4@310141ms, `weapon_kills.player_index=1`, l'ordre participants donne pi=4, et les fire events sniper de chunk_16 sont pi=6 — trois numérotations distinctes. Isoler « le tir qui tue » et lire un champ-cible reste non résolu (cohérent avec le statut ouvert côté communauté).
- **Reco** : cible plus tractable = décoder l'**aim vector octaédrique** (octant + uint16) pour un crosscheck géométrique (le tireur visait-il la victime au moment du kill ?), plutôt que le champ-cible direct.

Outils throwaway : `tmp_film_explore/filmx.exe` (modes ajoutés : `he kf fire wbits evwide medals aim`) + `apps/go-api/cmd/tmpdbq` (requête SQL DuckDB read-only, `//go:build ignore`).

### J. Vecteur de visée (aim vector) — algo acurtis porté + VALIDÉ sur nos données

acurtis166 a publié l'algo complet de décodage du aim vector (cubemap 6 faces, `_FACE_SIZE = 0xAAA8000`, payload 30 bits). Porté en Go et **self-test exact** : `decode((0x6240e840e0>>5)&0x3FFFFFFF) = (0.3556, 0.9346, 0.0)` = valeur attendue acurtis. Appliqué à nos fire events réels (suffixe 42c9679f) : on lit les **40 bits juste après le weapon id**, on applique `(w40 >> 5) & 0x3FFFFFFF`, on décode → **vecteurs unitaires propres**. Ex. Player pi=6 S7 Sniper (chunk_16) : 5 tirs consécutifs à `(-0.737, +0.676, 0.0)` (visée tenue) puis variation — physiquement cohérent.

- **Limite connue (acurtis)** : la coordonnée secondaire n'est pas décodée → z (ou la 2e coord de face) supposée 0 (centre), erreur max ~45° aux coutures.
- **LevelUp ne capture PAS** ce champ aujourd'hui (le scanner fire event ne lit que weapon + b5 + burst + hit/miss).
- **Usage pour fiabiliser l'arme** : crosscheck géométrique — au `time_ms` du kill, le vecteur de visée du tueur pointe-t-il vers la victime ? Confirme l'attribution fire-event→kill (donc l'arme). Aussi : base du replay 2D / heatmaps de visée.

### K-bis. Décodeur melee VALIDÉ (correction du marqueur) — 2026-06-01

La version §D ("non fiable") est **supersédée**. Le vrai ancrage n'est pas le marqueur 11-bit `0b10100110010` (introuvable au bit près) mais : **préfixe `101` (3 bits) suivi d'un byte anchor ∈ {0x34, 0x35}**. anchor = position du byte 0x34/0x35. Self-test sur l'exemple acurtis : anchor lit 0x34, type@+76=0x42, **weapon@+88=f408190f42c9679f** (Mk51 Sidekick), anim@+84=0xd — MATCH exact. Offsets weapon par type (relatifs à anchor) : `0x42:[88]`, `0x47:[86]`, `0x60:[101,103]`. **La validation du weapon id aux offsets est le filtre clé** qui écrase les faux positifs (sans elle, ~98 % de bruit).

Validé sur 000d5950 : **56 swings melee** weapon-validés (vs 4 melee KILLS API) — le surplus = melees sans kill (whips, swings ratés). Armes décodées correctement **avec variantes** : type=0x47 → Rushdown Hammer (`0x841ac5e5d8d07ca1`) & Diminisher of Hope (`0x841ac5e5a730e49f`) ; type=0x42 → Sentinel Beam / Mk51 Sidekick (pistol-whip). **player_index = byte@anchor+20 (low 5 bits, 0-31)** — PAS les 5 bits hauts (toujours 0) ni `>>4`. Cohérence confirmée : le Rushdown Hammer (arme de pickup) apparaît avec pi {0,1,3,4} = 4 joueurs l'ayant tenu successivement au fil du match. Le pi est le pi canonique film (même que fire-event `b5>>4`), résolu en pi↔xuid par le mapping PLAYER_METADATA déjà en prod (méthode dend : byte précédant la 1ère occurrence du XUID de chaque joueur, incrémente par joueur). Mode `filmx.exe <chunk> melee`.

### K. Melee : le weapon id est DANS l'event melee (structure acurtis affinée)

acurtis a publié la structure melee complète (marker `0xd340`, ancre 0x34/0x35, type byte à +76 ∈ {0x42 unpowered/non-melee miss, 0x47 hammer, 0x60 sword powered hit / unpowered hit}, offsets weapon par type, **weapon id + variant + animation type présents**). La colonne « animation type » (nibble avant le weapon id, `5` ou `d`) mappe à une animation **par arme** (table acurtis : ex. Energy Sword `5`=slash haut-gauche→bas-droite, `d`=stab droite→gauche ; S7 Sniper `5`=coude droit, `d`=talon). Donc, contrairement au highlight event (qui n'a pas l'arme), **l'event melee porte directement le weapon id** → identifie l'arme du kill melee (épée/marteau/crosse). Le marqueur 11-bit reste bruité (cf. §D) mais le filtre type-byte {0x42,0x47,0x60} + `pi≤15` + validation weapon id aux offsets `{0x42:[88], 0x47:[86], 0x60:[101,103]}` (relatifs à l'ancre) le fiabilise.

### L. Structure de paquets 16 octets (acurtis) — VALIDÉE, débloque la navigation du fichier

acurtis a publié le **header de paquet 16 octets** qui structure chaque chunk décompressé :

| Champ | Type |
|---|---|
| Type | uint16 LE |
| ??? | uint8 |
| ??? | uint8 |
| Size (octets payload) | uint32 LE |
| Timestamp (microsecondes) | uint64 LE |

`PacketType` : 0=FRAME, 1=REPLICATION_DATA_START, 2=TYPE_2, 6=TYPE_6, 7=CHUNK_END, 8=PLAYER_METADATA, 9=HIGHLIGHT_EVENTS_START, 10=TYPE_10, 12=BOT_METADATA. On lit `[16o header][size o payload]` en boucle jusqu'à CHUNK_END.

**Validé sur nos données** (chunk_02 replication, `filmx.exe <chunk> pkt`) : parse **parfait**, 739972/739972 octets consommés, fin exacte sur CHUNK_END. Contenu :
- **1199 FRAME packets** sur 19.98s, dt moyen **16.68ms = ~60 fps** ; chaque FRAME porte un **timestamp µs précis** (ex. 4537898738µs absolu) → remplace l'estimation par frame-markers `A0 7B 42`. Timing fire-events exact.
- **1 PLAYER_METADATA (type 8, 25124 octets)** = le roster joueurs (le "~25KB PLAYER_METADATA" du doc player_index). Contient gamertags (Akatsuki fire17, aldusbroncus…) + xuids. **Localisé** ; le layout exact pi↔xuid reste à RE (gamertags et xuids non trivialement adjacents). C'est la clé pour résoudre le chaos player_index (piste A).
- **1 BOT_METADATA (type 12)**, **REPLICATION_DATA_START** (snapshot initial 343 KB), TYPE_2 (143 KB), TYPE_10 (1199× interleavé aux frames).

**Impact** : on peut désormais indexer tout chunk sans le lire en entier, dater chaque fire event au µs via le FRAME le contenant, et cibler PLAYER_METADATA pour le mapping joueur. Le marqueur fire confirmé par acurtis = `0b101 0010 0110` (anchor +3 sur 0x26, weapon à +40, valider vs enum Weapon).

> **Note (acurtis, basse priorité)** : le suffixe `42c9679f` des weapon-ids 64-bit est répétitif/peu utile pour l'identification ; on peut ne valider que les 32 bits hauts. Le suffixe correspond peut-être à autre chose (non investigué).

> **Vérif morts-sans-tueur (000d5950)** : sur 93 deaths (th=20), **93 ont un tueur crédité** (th=50 d'un autre joueur à ±150ms), **0 mort non créditée**. Donc le surplus kills (th=50 ≈ 101-112 vs 93) n'est PAS des morts sans tueur (pas de suicide pur dans ce match) — c'est uniquement le sur-comptage th=50 (crédits redondants/assists). Confortе : ancrer le kill-feed sur les deaths (th=20), dédupliquer th=50.

### M. Événements de mode à objectifs (type_hint=10) — LOCALISÉS (CTF + Strongholds)

Exploration pure (aucune source). Le doc note depuis le début que `type_hint=10` = « événements mode-spécifiques (flag captures, objective kills) ». Confirmé sur films objectifs téléchargés (modes via `match_registry.game_variant_name`) :
- **CTF:Arena (53ce4390)** : 34 events th=10
- **Strongholds:Arena (7344d24f)** : 71 events th=10

Chaque event th=10 porte (comme les kill/death) : **xuid acteur + time_ms précis**. Structure du bloc (VALIDÉE par ground-truth utilisateur sur CTF 53ce4390, 2026-06-01) :
- `b[55]` (is_medal) = 0 ; `b[59]` = **2** (constant pour tous les th=10) — discriminant "event de mode".
- `b[36]` = **slot joueur 0-3, constant par xuid** (slot dans la structure).
- `b[37]/b[38]` = **team_id du joueur** (`00 00` = team 0, `01 01` = team 1). CONFIRMÉ : le split 4/4 des events colle exactement au roster DB (CTF 53ce4390 : team 0 = Guisy97/CnR SPACE XV/oo TAURUS oo/Oilycrusader ; team 1 = JGtm/TheRaeSide/x Atrezik/II Glad). **Correction** : ce n'était PAS un sous-type grab/capture (hypothèse infirmée par le ground-truth).

**Validé end-to-end (CTF 53ce4390, timings Theater = time_ms film)** : les events th=10 = **interactions de drapeau** (grab/return/capture) horodatées + joueur (xuid→gamertag). Mappings confirmés : grabs de TheRaeSide (flag runner team 1, 8 events) et Oilycrusader (team 0) vers ~2:13/3:04 ; double return à 5:21 par CnR SPACE XV + oo TAURUS oo ; **capture par Guisy97 à 5:41** (en cluster simultané avec les resets de drapeau). Score final 2-1 team 0 (3 captures totales, cohérent avec le timeout).

**Strongholds aussi validé (7344d24f)** : les th=10 = **captures de zone**, timings = ground-truth utilisateur (JGtm capture 0:52 et 3:16 → events portent le xuid de JGtm 2533274823110022 ; base B 1:24 ; etc.). `b[0:32]` = gamertag de l'acteur (confirmé : "Oilycrusader", "TheRaeSide", "Guisy97"…).

**Reste ouvert — le sous-type/zone n'est dans AUCUN champ lisible** : vérifié exhaustivement sur CTF (ground-truth grab/return/capture) que `b[0:32]`=gamertag, `b36`=slot, `b37/b38`=team, `b59`=2, **préfixe marqueur=0x2d (constant)** — aucun ne distingue grab/return/capture (CTF) ni la zone A/B/C (Strongholds). L'action/zone n'est donc PAS encodée dans l'event highlight ; elle nécessite la machine d'état drapeau/zone (replication). Heuristique : les CAPTURES apparaissent en **clusters simultanés** d'une même équipe (la capture reset les drapeaux/zones → burst). 

**Score continu Strongholds** — ground-truth ancré (équipe alliée JGtm) : **54 @3:16, 120 @4:55, 124 @5:59, 167 @7:48** (≈1 pt/s à 2 zones). Le compteur brut reste dans la replication (schéma-bloqué). Deux pistes : (a) recherche ciblée du compteur (54→120→124→167 aux frames correspondants) ; (b) reconstruction par intégration du taux sur la timeline de contrôle de zones — mais bloquée car la **zone (A/B/C) n'est pas dans l'event** (b36=slot, pas zone). Mode `filmx.exe <type3_chunk> evdump 10`.

### N. Weapon swaps/pickups — décodage dédié BLOQUÉ ; held-weapon dispo autrement

acurtis a publié des **dumps bruts** d'events de swap (AR↔Bandit rack) mais **aucune spec marqueur/offset** (contrairement à grenade/melee/aim qu'il a spécifiés). Le motif récurrent `c4 c0 10` de ses exemples n'est PAS un marqueur dans nos chunks : 0 occurrence byte-alignée, 1 seule au bit près (coïncidence, sans weapon id à proximité). Décoder l'event de swap from-scratch (record ~230o, 2 weapon ids bit/nibble-shiftés) est un effort multi-itérations non couvert.

**Alternative pratique (disponible MAINTENANT)** pour « tracer l'arme tenue par joueur » : les **fire events** donnent déjà l'arme tenue à chaque tir (pi + weapon id), et **LevelUp a déjà** `ScanFormulaA` (snapshots `20 00 02` par joueur) + `BuildWeaponTimelines.SwapPIs` (détection de changement d'arme). Démontré : agrégation des fire events par joueur → séquence d'armes sur le match (ex. Super Fiesta 000d5950 : chaque pi avec sa variété d'armes correctement capturée). Pour la précision du swap exact, attendre qu'acurtis publie la spec, ou RE from-scratch.

### M-bis. Score/zone-count : différentielle multi-agent — BLOQUÉ (vérifié, 2026-06-01)

Tentative exhaustive de décoder le score d'équipe Strongholds depuis la replication, avec ancres ground-truth (54@3:16, 120@4:55, 124@5:59, 167@7:48). Workflow 4 agents (théories : marqueur score u8/u16, paire adjacente, pic de taille de frame, compteur monotone) + mes propres passes (paire byte, keyframe byte/bit monotone). **Tous négatifs, haute confiance.**

**Piège attrapé par la vérif adverse (leçon clé)** : la Théorie 1 a trouvé une fenêtre récurrente `00 0a 00 00 00 0a 00 00 [00 VAL] …` où VAL lu en u16BE matchait **les 4 ancres** (00 36=54, 00 78=120, 00 7c=124, 00 a7=167) avec un suffixe monotone par chunk. Un match 4/4 — qui ressemblait au score. **MAIS** le test inverse (distribution des valeurs APRÈS le marqueur) l'a réfuté : ce marqueur apparaît 1027-1155×/chunk et le score n'y **domine jamais** l'histogramme (plat) → c'est du framing générique par-frame/par-entité, pas un composant score. Un vrai marqueur de score ferait dominer le score. Sans la vérif multi-agent, ce faux positif aurait pu passer pour la solution.

Mon « pic de frame à la capture » (chunk_03) a aussi été démenti : la frame de capture n'est que **1.05× la moyenne** (pas un pic) ; la taille de frame est dominée par les keyframes de bord de chunk, pas les captures (corrélation cible 1/4 = hasard).

**Cause racine** : payload bit-packé (entropie 4.49, 48.8% de zéros). Les valeurs de score sont de **petits octets** (0x36/0x78/0x7c/0xa7) présents **700-1150×/chunk par hasard** → taux de faux positif ≈ 1. Aucun alignement (byte/bit), narrowing temporel, ni nombre d'ancres ne peut isoler une valeur si commune. Le score vit derrière un **schéma property-id/network-GUID** dans le delta per-frame. **Convertir "bloqué" en "décodable" nécessite le schéma off-film** (layout des propriétés du GameMode Strongholds / zone-controller : property-id → bit-width → type). Pistes in-film résiduelles (faibles) : scoreboard de fin de match (peut-être typé plainement), ou diff de bit-fields entre frames successives APRÈS avoir le property-id.

### M-ter. Score : DÉBLOQUÉ — il est dans TYPE_2, byte-aligné (Slayer, vérifié 2026-06-02)

**Le score EST dans le film.** M-bis avait conclu "bloqué" parce que toutes les passes fouillaient
`REPLICATION_DATA_START` (343 Ko, bit-packé, champs qui dérivent → aucun offset fixe) et les deltas
per-frame. **Personne n'avait cherché dans le paquet TYPE_2 avec des ancres propres.**

**Le déclic (idée utilisateur)** : prendre un match **Slayer** où le score = kills, donc un compteur
qui fait **+1 à chaque mort, à des timestamps qu'on CONNAÎT**. Ça transforme 4 ancres continues estimées
(Strongholds) en **26 ancres discrètes exactes**. Ground-truth dérivé proprement des morts (th=20, 93 events,
sans doublon) : en 2 équipes le tueur est forcément l'équipe adverse → `score_team0 = morts_team1`,
`score_team1 = morts_team0` (vérifié : colle à l'API 43/50).

**Méthode** (`tmp_film_explore/scorefind/`) :
1. Extraire le payload TYPE_2 de chaque chunk gameplay (`filmx.exe <chunk> extract 2`).
2. Construire la séquence de score par frontière de chunk (cumul des morts `t < start_ms`).
3. `scorefind exact <dir> <séquence>` : recherche colonne-par-colonne (byte u8/u16 + bit 6-12 BE/LE) de
   l'offset dont la colonne sur les N keyframes matche la séquence.

**Résultat (Slayer 000d5950)** :
- TYPE_2 payload **byte 832** = `team0_score × 4` → `score_team0 = payload[832] >> 2` — **26/26 exact**.
- TYPE_2 payload **byte 842** = `team1_score` (×1) — **25/26** (un seul ±1 = jitter de frontière sur un
  kill à t≈40000 compté juste avant/après le bord du chunk).
- `SUM` ne matche aucun offset → le score est stocké en **deux compteurs par équipe**, pas en somme.

**Pourquoi TYPE_2 et pas REPLICATION_DATA_START** : TYPE_2 est le snapshot d'état-de-jeu par chunk. Le score
vit dans un **bloc game-state à taille fixe dans les ~1500 premiers octets**, byte-aligné. Le bloc score
s'arrête vers byte ~907 puis zero-padding ; le reste de TYPE_2 (taille variable, 138-147 Ko) = état entités.
La taille de TYPE_2 oscille (pas un log append-only) → c'est bien un snapshot, pas un journal d'events.

**Offset mode-dépendant** : 832/842 est l'offset *Slayer*. Les modes objectif (Strongholds/CTF) placent
le score à un autre offset (en-tête de mode différent) — `scorefind mono <dir> <min> <max> [ancres]` cherche
les compteurs monotones atteignant une plage finale + matchant des ancres. Généralisation aux modes objectif
+ chasse au **kill-feed/arme dans TYPE_2** (le kill feed à l'écran montre l'arme = état HUD, donc plausiblement
dans TYPE_2 près du score) : workflow 4 agents en cours (Strongholds 7344d24f, 2 Slayer de confirmation,
CTF 53ce4390, kill-feed 000d5950). **Câblage scanner différé v2.**

### Synthèse « fiabiliser l'arme du kill » (combinaison des veines)

- **Kills à l'arme à feu** : weapon id du fire event (existant) + crosscheck géométrique via aim vector (§J, nouveau) + médaille weapon-specific horodatée (§H, nouveau, ex. Snipe/No Scope→sniper).
- **Kills grenade** : marqueur `0x4c0c00` → type de grenade (§C, validé).
- **Kills melee** : event melee `0xd340` → weapon id + animation type (§K).
- Le kill-feed (killer→victime) reste la jointure temporelle `time_ms` (§E), l'arme s'y greffe par les 3 points ci-dessus, PAS par une donnée stockée dans le type-3.
