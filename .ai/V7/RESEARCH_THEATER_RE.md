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

**Résultat (Slayer 000d5950, offsets PROPRES)** :
- TYPE_2 payload **byte 813** = `team0_score × 4` → `score_team0 = payload[813] >> 2` — **26/26 exact**.
- TYPE_2 payload **byte 823** = `team1_score` (×1) — **25/26** (un ±1 de jitter de frontière).
- `SUM` ne matche aucun offset → deux compteurs par équipe, pas une somme.
- ATTENTION bug d'outillage : `filmx extract` écrivait sa bannière `[raw] N bytes` sur **stdout**
  → +~19 octets parasites de longueur variable par chunk. Le premier relevé "byte 832/842" était gonflé
  par la bannière. Corrigé (bannière → stderr, `main.go` patché, `filmx.exe` rebuildé) ; payload propre
  commence par `a0 00 00 00 00 00 00 0b`. Toujours extraire `1>out 2>NUL`.

**Généralisation (workflow 4 agents, vérifié 2026-06-02)** :
- **CONFIRMÉ sur 3 matchs Slayer** : le score vit dans l'en-tête game-state fixe de TYPE_2 (~bytes 810-850),
  MAIS **l'offset ET le scaling glissent par match** (structure bit-packée précédée d'un petit préfixe
  variable ; le bloc score coulisse de quelques bits). 000d5950 = 813(×4)/823(×1) ; c7d40d45 = 846(×1)/836(×1) ;
  d7e0d591 = bit6782(×1)/byte837 u16BE(×32). Tous 26/26 ou 30/30 exact. **Règle : scanner une fenêtre
  (~bytes 810-850, widths bit 6-12 BE, scalings ×1/×2/×4/×32) par match**, ne pas hardcoder un offset.
- **Strongholds 7344d24f : SOLVED** (grâce au lead utilisateur « 2 zones tenues = +1/s, contesté = 0 »). Le score
  allié (team0=JGtm, WIN) est à TYPE_2 **byte 842** = un **varint à continuation** après le marqueur fixe `7b 60`
  (byte 839) : `vg = byte842 si (byte842 & 0x40)==0, sinon ((byte842 & 0x3f)<<8)|byte843` ; **score = vg × 4.099**
  (quanta ~4 pts). Reconstruit à ±1 des 4 ancres (53.3/119.9/123.0/168.1 vs 54/120/124/167) ; le palier 295-359s
  retombe EXACTEMENT (vg figé à 30 sur 4 keyframes consécutifs). Pourquoi les 8 recherches précédentes échouaient :
  c'est un **varint dont l'offset dérive** (~1 byte au chunk 21), pas un champ fixe → un scan d'offset fixe ne peut
  pas l'attraper ; il fallait l'insight mécanique (intégrale d'un taux qui *stalle*) pour savoir quelle forme chercher.
  Taux 3-zones k non résolvable à 20s (aucune fenêtre 20s entièrement à 3 zones). Le score continu EST donc bien
  dans TYPE_2 — le mur §M-bis était un artefact de la méthode (scan fixe vs varint dérivant).
- **CTF : Neutral Flag (à 5) CRACKÉ ; Arena (à 3) non matérialisé** (workflow 5 matchs). **Ancre invariante = le
  token 12-bit `0x7B6`** (MSB-first, content-scan bytes [840,905)) — les « marqueurs » 7b60 / 07b6 / 3db0 vus par
  match sont le MÊME token à des phases de bit différentes (record bit-packé démarrant à un sous-octet variable).
  Captures = **deux compteurs par équipe, petits, +1/capture (3 bits, sans scaling)** dans anchor_bit+[60..175],
  mais leur distance à l'ancre **dérive par match** (région clock/points variable entre) → décodage = **SCAN** :
  chercher les ≤2 colonnes monotones (pas de pas >+1, mutuellement exclusives). Vérifié : Neutral 0f9550e5 (5-0),
  80a07955 (0-5), dcf44b35 (5-4). **Arena à 3 (64e8adfa, 53ce4390) : NON cracké** — la capture discrète n'est PAS
  matérialisée comme champ stable dans le snapshot 20s (stockée en accumulateurs de flag-carry) ; 64e8 ne contient
  même pas le token 0x7B6. Résolution à 3 : per-frame, events CAPTURE explicites, ou le **stat per-joueur nommé
  `FlagCaptures`** (cf. §O).

**Pattern général confirmé** : le score vit dans le bloc game-state TYPE_2 (~bytes 810-895) mais souvent comme
**varint / champ bit-packé à offset dérivant** (Strongholds byte 842 varint ; CTF layout multi-era ; Slayer scaling
×1/×2/×4 par match). D'où la règle : **lecture varint ancrée sur un marqueur local par match**, pas un offset hardcodé.

**CTF — TIMELINE DES CAPTURES : RÉSOLU (workflow 2026-06-02)**. Une capture de drapeau déclenche un **burst FRAME**
qui re-transmet la **table objectif COMPLÈTE = 6 records de l'échelle de score** co-occurrents dans une frame :
préfixes `a4 00 00 00`, `03 48 00 00 01`, `06 90 00 00 02`, `0d 20 00 00 05`, `1a 40 00 00 0a`, `34 80 00 00 15`
(la valeur gauche double ; préfixes **constants inter-match**, seul l'octet d'instance final varie). **Détecteur =
frame avec `tiers==6`** (≥3× baseline ; `tiers≥4` sur-compte). Distinct des bursts de mort (`38 68 e4`) / respawn
(`2d 20 71 22`). **COUNT + TIMING = rock-solid** : 0 manque / 0 faux positif sur 4 matchs (counts = DB exactement :
0f9550e5=5, 53ce4390=3, dcf44b35=9, 64e8adfa=5), ms-précis. Ancre or validée : 53ce4390 capture TheRaeSide à
t=656554 (burst 1081o) = th=10 t=656558 team1. **ÉQUIPE par capture** : fiable via l'**event th=10 `b55`(team)
coïncident ±5ms** — MAIS seulement si le **chunk type-3 footer (events) est en cache** (53ce4390 footer chunk_40 →
3/3 avec équipe ; dcf44b35/64e8adfa footer non-caché → timing OK mais équipe partielle/inconnue). Le compteur 0x7B6
n'est PAS fiable pour l'équipe par-capture (multi-era). **Prod : (1) `tiers==6` = détecteur universel de capture ;
(2) pour l'équipe, recâbler le cache film pour RETENIR le chunk type-3 footer** (le scanner ne télécharge que les
chunks type-2 → footer absent → c'est pourquoi l'équipe manque). Outils : tmp_film_explore/{ctfsig,ctfcap,t2score}.

**Modes objectif — statut score + events (workflow 2026-06-02)** :
- **KOTH : score SOLVED** (2 variantes, ancre 0x7B6 @byte839). (A) points (606d9844 105-8) : `t2[anchor+14]`=total,
  `t2[anchor+13]>>4`=team1, team0=total−team1. (B) Ranked captures (0a247154 4-2) : `t2[anchor+12]`=meter team0,
  `t2[anchor+16]`=meter team1, **score = meter/5** (5 ticks = 1 capture). Validés DB exactement (105-8, 4-2).
- **Strongholds : score SOLVED** (byte842 varint×4.099) ; **moments de capture = PARTIEL**. PAS de burst discret
  (accrual continu) ; reconstruits via l'**inflexion de la pente du score** (dérivée du nb de zones tenues) à **~20s**,
  équipe **inférée** du signe de la pente. Zone-ID non récupérable (b36 = slot joueur, pas zone). Structure de contrôle
  + palier = solides ; captures discrètes ms-précises = non.
- **Oddball : carry-timeline SOLVED, score ≈ ±1-3% (gagnant), exact RÉSISTE** (footer re-fetché, vérifié 2026-06-02).
  Footer caché → events th=10 = **heartbeats de possession du crâne (~5s)** : slot + équipe + horloge ms + xuid, PAS
  d'octet score. Carry timeline (qui porte, quand) reconstruite ; score = **intégrale du temps de possession** (rate
  ~1 pt/s) → **gagnant à ±1-3%** (24db 202 vs 200 ; d97 202 vs 196) ; le perdant sur-prédit (+12-22% : carries contestées
  finissant en mort + lead 5s). Flux **LOSSY** (ticks droppés en carry contesté) → pas exact. Le ladder TYPE_2 (off+88
  après 0x7B6) = accumulateur **GLOBAL** (croît indépendamment de l'équipe) → le score per-team n'est PAS un champ propre.
  Équipe via `xuid→team_id` DB (le champ team d'evdump est faux sur 24db). Exact final = DB.
- **DISTINCTION CLÉ (continu vs discret)** : seul **CTF** a un burst de capture DISCRET (capture = +1 → re-transmit de
  la table 6-tiers). Strongholds/KOTH/Oddball **accrèdent en CONTINU** → pas de burst par-capture ; les « moments » =
  inflexions de pente (~20s). Plus fin = décoder l'octet controlling-team par-FRAME (deeper RE, non fait).
- **BLOQUEUR UNIVERSEL + fix** : le chunk **type-3 footer (events th=10 + team b55 propre) n'est PAS caché** (scanner
  = type-2 only) ; re-fetchable du CDN blob_prefix (no auth, http 200). Un seul recâblage → events + team par-event
  fiables sur TOUS les modes objectif.

- **Kill-feed/arme dans TYPE_2 : NON définitif** — scan bit-level exhaustif des 26 keyframes vs séquences de
  kills récents = best 15/24 (bruit). Le kill feed à l'écran est de l'**UI client éphémère** reconstruite depuis
  le flux d'events, PAS persistée dans le snapshot. Les weapon-ids (suffixe `0x42c9679f`) ne vivent QUE dans la
  région entités de TYPE_2 (~bytes 24400-71000) + les fire events des paquets FRAME. **Arme-du-kill = corréler
  chaque kill (evdump 50: t, killer) avec son fire event le plus proche (`filmx fire` → pi+slot+weapon)** —
  décodeur viable, déjà présent dans les chunks TYPE_2.

**Pourquoi TYPE_2 et pas REPLICATION_DATA_START** : TYPE_2 est le snapshot d'état-de-jeu par chunk. Le score
vit dans un **bloc game-state à taille fixe dans les ~1500 premiers octets**, byte-aligné. Le bloc score
s'arrête vers byte ~907 puis zero-padding ; le reste de TYPE_2 (taille variable, 138-147 Ko) = état entités.
La taille de TYPE_2 oscille (pas un log append-only) → c'est bien un snapshot, pas un journal d'events.

**Outils** : `tmp_film_explore/scorefind/` (modes `exact` et `mono`+ancres, glob `kf*.bin`). **Câblage scanner différé v2.**

### N. Positions joueurs + immobilité + spawn — DÉCODÉ (vérifié 2026-06-02, Slayer 000d5950)

**Carte entité→joueur : SOLVED** (le mur de la session). Via les events highlight (chunk-27), le couple
`(team, b36-slot)` identifie les 8 joueurs sans conflit sur 205 events kill+death :
team0 b36={0:…0416, 1:…0022, 2:…0284321, 3:…4760703} ; team1 b36={0:…2097883, 1:…5845110, 2:…4178793711, 3:…7245250}.
Cross-confirmé : la corrélation gaps-de-tir↔morts a indépendamment épinglé **fire-pi=2 = xuid …0022** (p=0.006).

**Immobilité CONFIRMÉE (modèle absence rejeté) — 3 preuves indépendantes** :
1. Le snapshot ne rétrécit PAS quand des joueurs meurent (le nb d'entités *monte* avec deadCount, r=+0.59 ;
   absence prédirait −1). Les morts génèrent des entités transitoires (armes lâchées, ragdolls, effets).
2. Les **records joueurs** (cf. ci-dessous) sont à **~8 constants** par keyframe (mean 8.24, mode 8), découplés de
   aliveCount (4-8) → les morts gardent leur record (figés à leur position de mort).
3. **Figé-puis-téléport OBSERVÉ directement** (joueur …0022, mort t=62936) : burst de frame de MORT @62932.9ms
   (1050B = 4.3× la baseline, +4 blocs full-state re-transmis, à ±3ms de la mort) → **intervalle figé** 63-72s
   (aucun burst, aucun tir) → burst de RESPAWN @72058.2ms (1289B, à ±2ms de l'event t=72060). Délai respawn
   event = 7606ms (∈ 8-9s prédit). Triple-bracket (frame-burst + event + gap de tir).

**Records de position joueurs — TROUVÉS** (région dynamique TYPE_2, AVANT le manifeste d'objets statiques dont le
landmark byte-identique est (−6.60,5.42,−2.14)). Chaque record est délimité par un **« comb » P24** (motif binaire
1⁸0¹⁶ ×4), un comb par joueur, espacés ~310-345 bits. Position = **triplet float32-LE (x,y,z) à (combStartBit − 177
− 96)** dans les records full-state ; les records delta omettent la position absolue (compression delta). Validation
forte : kf02 (début de match) = 8 records en **deux clusters 4+4** = signature spawn 2 équipes de 4 :
team A ~(43.7,9,0.5) (high-ground), team B ~(−4,16,−1.2). Positions bornées (x:−2..35, y:−24..25, z:−2.8..1.7).
NB : le suffixe arme `42 c9 67 9f` (BE) + marqueur `13 71` + tag `03 44 0c` = famille d'objets **statiques**
(weapon-spawns), PAS les joueurs (payload identique sur les 26 keyframes) — ne pas confondre.

**Per-frame positions DÉCODÉES + figé-puis-téléport PROUVÉ avec coordonnées (phase-4, ancré sur l'image FilmShell)** :
- Décodeur : **float32 BIG-ENDIAN à offsets BIT/nibble** (HNIB=4) dans la région type-0 FRAME concaténée. En-tête de
  frame = ancre 32-bit `0xA07B4200` + octet TICK (+0x08/frame, ~16-17ms/frame). Delta-compressé (seuls les joueurs
  qui changent sont ré-émis) ; chaque coord précédée de bits-flag `1` (no-change). Cross-validé : un track atteint
  exactement 17.13 = la valeur exemple de l'écran FilmShell → repère/échelle confirmés.
- **Figé-puis-téléport (joueur …0022) PROUVÉ** : triplet position du corps = **(−4.77, −9.00, 4.06)** — isolé du
  décor statique car 0/26 frames AVANT la mort, apparaît au burst de mort f177@62946ms puis **figé bit-identique
  f177-201** puis SILENCE → **téléport au respawn = (35.07, 17.10, 50.00)** (émerge au burst respawn f723@72058ms),
  saut **~66 unités 3D**, fenêtre **~9.1s** (mort 62936 / respawn 70542 + settle). Exactement le modèle utilisateur.
- **Header ECS parsé** (chunk_00) : 42 archétypes, 264 composants nommés (slots 260o = [256o nom ASCII][u32]). Position
  joueur = archétype avatar/biped #35 (high-frequency, dynamic-precision) pos0=`object-position-dynamic-precision-component`.
  Globals #0 : pos0 team-mapping / pos1 shared-team-lives / pos2 current-state. Tiers `low-frequency`/`high-frequency`.
  Les ids FMT/SUB/BASE de FilmShell ne sont PAS sérialisés dans le header (enum moteur compile-time).

**Reste bloqué** : **tracks denses TOUS-joueurs attribués** — la compression delta fait que l'index P n'est pas à un
offset fixe (seuls les changeants ré-émis) + la continuité casse au gap de mort (corps silencieux). On isole un joueur
*spécifique* via une ancre temporelle d'event (sa mort), pas les 8 génériquement. Débloquer = RE complet du champ P
delta + l'enum de composants moteur (hors header).

Outils (stdlib, tmp_film_explore/) : delimscan, extractfinal, posfixed, firemap, framedump, framergn, immob,
exacttrk, postrk2, emerge (per-frame), hdrschema (registre ECS). **Câblage scanner différé v2.**

### O. Films = Lua/HavokScript embarqué — schéma de stats NOMMÉES (dend/acurtis, partagé utilisateur 2026-06-02)

Les films embarquent du **bytecode HavokScript** (magic `0x1b4c7561` = `\x1bLua`) + des chemins `.lua` (UTF-8) des
scripts de game-mode (CaptureTheFlag, Helpers/MPSpawnEvents, ParcelLibrary, globals/global_stats…). dend : « ce qu'on
voit n'est pas forcément du mouvement 'clair' mais des **instructions moteur** » → prudence : une partie du flux
bit-packé peut être du script, pas de l'état pur (mais les positions float32 FilmShell sont du vrai état — mouvement
lisse). **Implication majeure** : les stats/score sont des **stats NOMMÉES** de premier ordre dans le moteur. Noms
extraits du bytecode (= schéma de référence) :
- Objectif/CTF : `FlagCaptures`, `FlagCaptureAssists`, `FlagCarriersKilled`, `FlagReturns`, `FlagSecures`,
  `KillsAsFlagCarrier`, `KillsAsFlagReturner`, `ObjectivesCompleted` ; objets `Blue/Red/Neutral Flag Stand`.
- Combat : `SpartanKills`, `SpartanDeaths`, `SpartanAssists`, `PersonalScore`, `PowerWeaponKills`, `GrenadeKills`,
  `BestKillingSpree`, `AverageLifeDuration`, `DamageTaken`, `AccuracyPercentage`.
- PvE : `HunterKills`, `JackalKills`, `SkimmerKills`, `MarineKills`, `SentinelKills`.
- Scripts clés : `parcel_center_score_board_ui.lua` (scoreboard), `HandlePlayerSpawnOnClient` /
  `DeathCamHandleTeleport_Client` / `MPSpawnEventsStartup` (spawn/respawn = **events scriptés** → corrobore notre
  modèle immobilité+téléport), `ToggleMainObjectiveScoreboardClient`, `SendHUDEvent`.

**Piste robuste** : la capture CTF = le stat nommé `FlagCaptures` (per joueur). Si le film matérialise un bloc de
stats per-joueur (clé = stat-id mappé par la table de noms du header ECS), c'est la source ROBUSTE des captures ET de
tout (kills, assists, PersonalScore, kills PvE) — meilleure que reconstruire depuis le game-state. À chasser après le
parse du header (en cours, phase-4). Ce n'est PAS un décodeur clé-en-main (dend lui-même évite le bytecode HavokScript)
mais un schéma de noms + un pointeur vers où vit le score.

### Synthèse « fiabiliser l'arme du kill » (combinaison des veines)

- **Kills à l'arme à feu** : weapon id du fire event (existant) + crosscheck géométrique via aim vector (§J, nouveau) + médaille weapon-specific horodatée (§H, nouveau, ex. Snipe/No Scope→sniper).
- **Kills grenade** : marqueur `0x4c0c00` → type de grenade (§C, validé).
- **Kills melee** : event melee `0xd340` → weapon id + animation type (§K).
- Le kill-feed (killer→victime) reste la jointure temporelle `time_ms` (§E), l'arme s'y greffe par les 3 points ci-dessus, PAS par une donnée stockée dans le type-3.

### P. Théorie B — crosscheck géométrique aim·LOS + bit hit-location : DEAD-END (3 verrous, vérifié 2026-06-02)

Test de l'hypothèse « au kill, le vecteur de visée du tueur pointe la position de la victime ; + un bit
head/body/miss dans le fire event → confirme le tir/tueur et désambiguïse les fire events concurrents ».
Ground-truth : v2-HIGH `weapon_kills` (000d5950, killer xuid + weapon_id + time_ms) ⨯ deaths th=20 (victime).
Outils créés (stdlib, tmp_film_explore/) : `firetime` (date chaque fire event par le FRAME le contenant + dump
aim40 + tail40), `exactbit`/`findvals`/`burstpos`/`regiondump`/`recwalk` (décodage de records per-frame).

**Verrou 1 — le vecteur de visée du fire event est 2-DDL seulement (z toujours = 0) → LOS 3D impossible.**
Le décodage cubemap (§J) ne résout qu'UNE coordonnée dans la face ; l'axe orthogonal est **forcé à 0**
(acurtis : « secondary coordinate non décodée, erreur ~45° aux coutures »). VÉRIFIÉ exhaustivement : **100 %**
des aim décodés sur 000d5950 ET 0014603f ont `z=0.000`, et le vecteur claque sur l'axe dominant de la face
(une coord ±0.99, l'autre petite, la 3e = 0). Pire, en plein duel (trade-kill t=188296, pi=7 vs pi=5 qui se
tirent dessus) les DEUX décodent `(0.18,-0.98,0)` — quasi identiques. Un test d'angle LOS exige ~quelques degrés
de précision pour départager des fire events ; un vecteur avec un axe annulé et un biais massif vers la face 4
(chunk_04 : 19/22 à y<-0.7 ; chunk_10 : 22/22) ne peut pas le faire. **Le champ aim40 a une entropie directionnelle
trop faible pour un LOS.**

**Verrou 2 — la position du TUEUR (origine du rayon LOS) n'est pas récupérable.** Le décodeur phase-4 (§N)
n'isole une position que via une **ancre d'event** (la mort du joueur, qui déclenche un burst full-state).
La victime, oui : confirmé, mort de …0022 t=62936 → emerge frame 176 donne (-9.000, 4.060, -4.771) exact
(uint32 0xc1100086/0x4081ebef/0xc098a910), persistant 5-6 frames. Mais le TUEUR ne meurt pas → aucune ancre.
Et le format per-frame est un **flux delta réseau** : les 3 coords d'un même joueur ne sont PAS contiguës
(frame 177 victime : x@relBit 1645, y@2195 [+550 bits], z@2370 [+175 bits] — interleavées avec d'autres entités),
et les offsets glissent par frame (delta-compression). Le clustering « triplet le plus serré » (`burstpos`)
produit des faux triplets (les floats plausibles sont communs). Sur le trade-kill (2 morts simultanées, frame 496
burst 1824B = 9×) emerge sort 7 coords mais aucune ne re-localise en triplet stable en frame 497 → impossible
d'attribuer proprement les 2 triplets aux 2 victimes, encore moins de récupérer un tueur vivant.

**Verrou 3 — pas de ground-truth pour un bit headshot.** Halo Infinite **n'a PAS de médaille Headshot**
(vérifié metadata medal_definitions : seules Snipe/No Scope/Counter-snipe/Nade Shot/Perfect/Perfection existent,
toutes arme- ou style-spécifiques, pas « headshot générique »). Donc même si un bit head/body existait dans le
fire event, on ne pourrait pas l'étalonner. Les bits post-arme (`tail40`) varient par tir mais sans label de
vérité ; le « bf/hit/confirm » que j'avais d'abord lu en bits[0:32] était en fait **les bits de l'aim** (§J lit
35 bits juste après l'arme) → mon premier étiquetage était faux, re-corrigé.

**Démenti d'un piège** : la prémisse « KEY ENABLER » (pos2=forward-and-up siège à quelques champs APRÈS la position
dans le MÊME record avatar → étendre le walk pour lire aim+vitalité) **ne tient pas** contre le bitstream réel :
le per-frame n'est pas un dump de struct ECS plat mais de la réplication réseau delta-compressée bit-packée ;
« lire les champs suivants » donne du garbage (exposants énormes) car ce ne sont pas des champs alignés 32 bits
consécutifs (recwalk frame 177 : field+1..+28 = NaN/1e30). L'ordre de déclaration des composants dans l'archétype
#35 est réel (header ECS) mais ne se traduit PAS en adjacence binaire dans le delta per-frame.

**VERDICT : dead-end pour le crosscheck géométrique aim·LOS et pour le bit hit-location.** Les 3 jambes du test
sont chacune cassées indépendamment. Ce qui RESTE utile et confirmé : (a) la **position de la VICTIME au kill**
est décodable via son burst de mort (utile pour heatmaps de morts, contexte spatial du kill — pas pour l'arme) ;
(b) l'arme du kill reste la corrélation fire-event temporelle existante (§G/§E) + médaille weapon-specific
horodatée (§H) + melee/grenade (§C/§K). L'aim40 peut servir un replay 2D grossier (orientation approximative),
pas une preuve géométrique de l'auteur du kill. Débloquer Théorie B exigerait : un décodage cubemap COMPLET
(2e coordonnée de face, hors travaux acurtis) ET la RE complète du champ position delta per-frame pour TOUS les
joueurs vivants (hors header ECS) — soit la « Rosetta Stone » des schémas .module, non disponible offline.

### Q. Théories C (projectile) + A (vitality) : DEAD-END / BLOQUÉ ; conclusion arme-de-kill (vérifié 2026-06-02)

**Théorie C — tracking de projectile : DEAD-END.** Les projectiles tirés (roquette 71ab0a2c, aiguille b533957e,
skewer 0d20c469, cindershot 230447b1, heatwave 2ac9c2ff, ravager c30d87c7) **ne sont PAS des entités per-frame
trackées** portant weapon-id + propriétaire. Sur 2 matchs à fort usage projectile (000d5950 36 kills proj ; 1eee300e
77), chaque occurrence d'un weapon-id projectile dans le FRAME = un **fire event** (déjà dans weapon_kills) ou un
snapshot 1-frame d'arme tenue — JAMAIS un vol multi-frame launcher→victime (M41 0/1199 frames en chunk_02 malgré des
kills roquette). Les grenades = marqueurs de jet **épars 1-frame** (object-id, pas weapon-id), pas un arc tracké. Aucun
burst spawn/despawn au moment du kill. Cause : les weapon-ids ne vivent QUE dans (a) le snapshot REPLICATION_DATA_START
+ (b) les fire events des paquets FRAME.

**Théorie A — vitality-drop = instant exact : BLOQUÉ + sans gain.** L'agent a planté (pas de sortie structurée). Mais
même mur que §P : le per-frame est delta-packé (pas un struct plat) → la vie n'est lisible qu'au burst full-state ;
or le burst de mort = déjà l'instant que le highlight event marque → **aucun gain de timing** par rapport à ce qu'on a.

**CONCLUSION GLOBALE arme-de-kill** : les 3 angles « outside the box » (aim·LOS, projectile, vitality) sont tous
structurellement morts. **Il N'EXISTE PAS de signal d'arme-de-kill au-delà de la corrélation fire-event** (+ marqueurs
grenade/melee + horodatage µs). Le plan v3 (`.ai/PLAN_WEAPON_ATTRIBUTION_V3.md`) **EST le plafond** : plancher ~82%
(P1 grenade/melee + P3 canon, vérifiés) ; ~90% UNIQUEMENT si P2 (timing µs des fire-events) récupère les soft —
aucun levier additionnel n'existe. Recherche d'angles créatifs : **close.**

### R. Strongholds — OWNER par zone via objet-zone (position fixe + champ owner) : DEAD-END (vérifié 2026-06-02)

Hypothèse (dernière avenue accessible) : une zone Strongholds est un OBJET à position FIXE (les zones ne bougent
pas) portant un champ OWNER (neutre/team0/team1) qui CHANGE aux captures. Signature recherchée : record ~3× (3 zones)
à **position constante inter-chunks** + **petit champ owner variable**, validable contre la timeline GT (la contrainte
décisive = Zone B : team0 c6-15 → team1 c16-24 → team0 c25-31, soit `IIIII00000000001111111110000000`).

**Méthode** (outils throwaway `tmp_film_explore/{zonepos,zonebit,zonefld,zoneanch,zoneanch2,gtscan,gtscan2,gtclean,
zonerec,floatarr,tripcount}`, TYPE_2 propre via `filmx extract 2`, 31 chunks) :
1. **Triplets float32 à position constante inter-chunks** : le scan byte-aligné ne trouve RIEN (positions non
   byte-alignées) ; le scan **bit-level LE** trouve **un petit ensemble de ~5-9 objets statiques** présents dans les
   31/31 chunks. Position-like (carte) : `(2.330,-13.756,-3.002)`, `(2.877,39.164,-4.605)`, `(-48.021,-2.865,-3.002)`,
   `(2.268,-48.021,-2.865)`. Le reste = constantes structurelles (`(32.5,32.5,32.5)`, triplets z=50.502 = skybox/bbox).
   Chaque triplet est un **record d'objet discret** (triplet + 4e composante ~0 + zéros — pas un tableau de géométrie
   continue). Donc ce sont de **vraies positions d'objets statiques** (candidats zones plausibles).
2. **Champ owner co-localisé ?** NON. Pour CHACUN des 4 candidats, les octets/bits AUTOUR du triplet sont
   **bit-identiques sur les 31 chunks** (ex. `200940200d40201140201540a2165cc1801940c0a21cd983201940e0` constant) —
   c'est le **manifeste d'objets statiques** (= famille weapon-spawn déjà connue §N, payload figé par keyframe).
   Scan ancré (relocalise le triplet par chunk, gère la dérive d'offset) sur fenêtre **±4000 bits (±500 o)**, widths
   1-3, tolérance 2 chunks de bruit : **0 hit** correspondant au retour Zone-B (A en c6-15 & c25-31, B≠A en c16-24).
3. **Champ owner en composant parallèle (clé = object-id) ?** Inutile : l'object-id du manifeste statique est lui-même
   constant inter-chunks → le rechercher ne renvoie que le même bloc figé, aucun compagnon mutable.
4. **Owner = champ plain dans l'en-tête game-state TYPE_2 à offset fixe ?** NON. L'en-tête game-state EST à offset
   byte FIXE (marqueur score `7b 60` @byte839 vérifié sur 31/31, score allié reconstruit 0→205 monotone, cohérent §M-ter).
   Scan `gtclean` (retour propre Zone-B, ≤1 bruit) sur **toute la région fixe [835,2500]** : **0 hit**. En relâchant à
   ≤3 bruit/[835,3000], le « meilleur » candidat (byte 1449.6, w=1, col `0000000000000001101110110000000`) place bien
   sa bande à 1 sur c16-24 MAIS : (a) byte 1449 est **dans la soupe de compteurs qui DÉRIVE** (les octets glissent de
   colonne entre chunks — dump 1405-1460 le montre), (b) bruit c18=0/c22=0, (c) ne « revient » pas proprement. = **faux
   positif de la même classe que le bit-compteur monotone déjà démenti** (cf. PROVEN NEGATIVES). Au-delà du bloc score
   (~byte 907), la structure devient variable/dérivante → un scan d'offset fixe n'a plus de sens.

**VERDICT** : l'owner par-zone N'EST PAS un champ accessible — ni co-localisé à une position d'objet constante
(records statiques figés, 0 champ mutable à ±500 o), ni un champ plain 3-phases dans l'en-tête game-state à offset
fixe. Il est derrière le **schéma de record d'entité bit-packé / le composant controlling-team par-FRAME** (réplication
delta, offsets dérivants), qui exige le **schéma off-film** (.module / runtime-tagviewer) — le même mur que §M-bis/§N/§P.
Le score continu reste reconstructible (byte842 varint×4.099, §M-ter) mais **l'attribution zone→équipe à T n'est pas
récupérable offline → `objective_id` reste NULL**. Toutes les avenues in-film pour « qui tient quelle base à T » sont
**épuisées**.
