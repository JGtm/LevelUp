# Comment fonctionne le parser d'armes

> État : branche `analysis/weapon-parser-rewrite` — 2026-03-11

---

## Le problème à résoudre

Le jeu (via l'API Halo) donne le **nombre** de kills par type d'arme pour un match,
mais pas le détail kill par kill. Pour savoir "avec quoi JGtm a tué à t=3:24", la
seule source est le **film** du match — un fichier binaire téléchargeable depuis
les serveurs Xbox.

Le parser lit ce fichier binaire et relie chaque kill à son arme.

**Architecture en 3 couches — qui fait quoi**

| Couche | Fichier | Rôle |
|--------|---------|------|
| **Parser** | `weapon_parser.py` | Lit le film, corrèle kills↔fire events, retourne une liste de dicts en mémoire. **N'écrit jamais en base.** |
| **Service d'extraction** | `weapon_extraction_service.py` | Orchestre le parser pour tous les joueurs du match, applique la réconciliation API, produit les `kill_rows` finaux. **N'écrit pas non plus en base.** |
| **Repository** | `_weapon_kills_repo.py` → `insert_weapon_kill_rows()` | Seul responsable de l'écriture dans la table `weapon_kills` (shared_matches.duckdb). Appelé par le service après que celui-ci a finalisé les données. |

Le parser **ne produit pas** les weapon_ids. Les weapon_ids (8 octets) sont des
identifiants binaires fixés par le jeu et encodés tels quels dans le film. Le
rôle du parser est de :
1. **Lire** ces identifiants dans le film
2. **Les faire correspondre** à un nom d'arme via le dictionnaire statique
   `WEAPON_ID_MAP` (construit à partir d'investigations manuelles sur le film)
3. **Les associer** à un kill précis via la fenêtre temporelle

La découverte ou la mise à jour des weapon_ids est un travail d'**investigation
film** séparé (acurtis166, etc.) — pas une responsabilité du parser.

---

## Les deux sources d'information dans le film

Le film d'un match est découpé en **chunks** d'environ 19 secondes. Chaque chunk
contient deux types de données utiles :

### Section 1 — L'état des joueurs (Formula A)
Un snapshot périodique qui dit : "à cet instant, le joueur N tient telle arme".
C'est une photo d'état, pas un événement. Elle se met à jour quand le joueur
change d'arme ou ramasse une arme.

### Section 2 — Les tirs du POV (fire events)
Chaque fois que le **POV** (le joueur dont c'est le film) appuie sur la gâchette,
ça génère un event qui contient :
- L'instant du tir (en ms)
- L'arme utilisée (weapon_id 8 octets)
- L'indice du joueur (player_index)
- Un vecteur de visée encodé en octahedral 3D→2D

Les fire events se trouvent dans le **layer nibble-shifté** de chaque chunk
de réplication (REPLICATION_DATA). Ce layer s'obtient ainsi :

```python
nibble_shifted = bytes(
    (data[i] << 4 | data[i + 1] >> 4) & 0xFF
    for i in range(len(data) - 1)
)
```

**Structure d'un fire event** (positions dans le layer nibble-shifté) :

| Offset | Valeur | Description |
|--------|--------|-------------|
| [0] | `0x0D` ou `0x05` | Lead byte |
| [1] | `(player_index << 5) \| 0x06` | Ex. POV (idx=1) = `0x26`, joueur 2 = `0x46` |
| [2] | variable | `b2_stream` — discriminateur dual-stream |
| [3] | `0x40`–`0x43` | Constante (filtre : `byte[3] & 0xFC == 0x40`) |
| [4] | `0`–`248` (step 8) | `fire_counter` — compteur de tir (0–127 puis reset) ; peut sauter des valeurs (frames perdues) |
| [5] | variable | `b5_correlated` — corrélé à `b2_stream` |
| [6–13] | 8 octets | `weapon_id` |
| [14] | `0`–`7` | Octant de visée (encodage octahedral) |
| [15–16] | `uint16` | Magnitude dans l'octant |

**Méthode de scan alternative** (acurtis166) : chercher le pattern de bits
`0b101_0010_0110` (correspond à `0xD26` / `0x526` selon le lead byte) dans le
layer nibble-shifté, puis valider que les 64 bits à l'offset +40 bits (`_WEAPON_OFFSET`)
correspondent à un `weapon_id` connu dans l'enum `Weapon`.

**Armes automatiques — dual-stream** : BR75, MA40 AR et similaires génèrent
**deux** entrées par tir avec des valeurs `b2_stream` différentes mais le même
`fire_counter`. Les armes semi-auto (Bandit, Stalker, Commando) produisent **une
seule** entrée. Clé de déduplication : `(weapon_id, fire_counter)` par chunk.

**Bits après weapon_id** :
- 2ème bit = `0` pour les tirs en milieu de rafale, `1` pour le dernier tir (ex. BR75 : séquence 0-0-1)
- 3ème bit ≈ hit/miss (`0`=touché, `1`=raté), non fiable à 100% ; un second bit
  plus loin dans la structure confirme

**Limitation fondamentale** : la Section 2 ne contient que les tirs du POV.
Les adversaires et coéquipiers ne tirent pas dans "son" film — ou du moins pas
de manière fiable et continue. Confirmé expérimentalement :
même avec les 8 XUID et player_index de tous les joueurs résolus, seul l'index 1
(le recorder) produit des fire events.

> **Décision de conception** : les adversaires ne seront **pas traités**.
> Ni la Section 2 (fire events) ni la Section 1 (snapshots Formula A) ne fournissent
> leurs données d'armes de façon exploitable depuis le film d'un autre joueur.
> Tenter de les couvrir produit 63 % de NULL en base (cf. tableau ci-dessous).
> Seuls le POV et les coéquipiers dont le player_index est résolu sont dans le périmètre.

---

Note : les variantes d'une même arme (ex. BR75 Ranked, S7 Flexfire)
partagent le **même weapon_id** que l'arme de base.

**Liste des armes confirmées** (source : acurtis166, février-mars 2026) :

| Arme | weapon_id (hex) |
|------|----------------|
| Bandit Evo | `6ACDC44D42C9679F` |
| BR75 (= BR75 Ranked) | `2B1824D542C9679F` |
| Cindershot | `230447B142C9679F` |
| CQS48 Bulldog | `B619D84A42C9679F` |
| Diminisher of Hope | `841AC5E5A730E49F` |
| Disruptor | `84BD29ED42C9679F` |
| Duelist Energy Sword | `4FF3937E8978AA7A` |
| Elite Bloodblade | `4FF3937E1EC48C7A` |
| Energy Sword | `4FF3937E42C9679F` |
| Fuel Rod SPNKr | `9D6AAED242C9679F` |
| Gravity Hammer | `841AC5E542C9679F` |
| Heatwave | `2AC9C2FF42C9679F` |
| Infected Energy Sword | `0C55765F7A9376A0` |
| M392 Bandit | `2FB21C8742C9679F` |
| M41 SPNKr | `71AB0A2C42C9679F` |
| MA40 AR | `48C19D2D42C9679F` |
| MA5K Avenger | `F5C335DFE7232C0F` |
| Mangler | `80977BA542C9679F` |
| Mk51 Sidekick | `F408190F42C9679F` |
| MLRS-2 Hydra | `767DB96D42C9679F` |
| Mutilator | `D791556542C9679F` |
| Mythic Sandwich | `B7262CA1C8FB11D0` |
| Needler | `B533957E42C9679F` |
| Plasma Pistol | `C354294642C9679F` |
| Pulse Carbine | `30484EA642C9679F` |
| Ravager | `C30D87C742C9679F` |
| Rushdown Hammer | `841AC5E5D8D07CA1` |
| S7 Sniper | `0A1992BC42C9679F` |
| Sandwich | `880FE0BC42C9679F` |
| Sentinel Beam | `A0955E9E42C9679F` |
| Shock Rifle | `9387A8B942C9679F` |
| Shock Rifle (Ranked) | `1A22FEE642C9679F` |
| Skewer | `0D20C46942C9679F` |
| Stalker Rifle | `DAF193C742C9679F` |
| Vestige Carbine | `3E07021742C9679F` |
| VK78 Commando | `FD98554C42C9679F` |

## Structure binaire d'un chunk (packet header)

Chaque chunk du film est découpé en **paquets** de taille variable, chacun précédé
d'un en-tête de **16 octets** (acurtis166, mars 2026). Cette structure permet
d'indexer toutes les sections du film sans lire l'intégralité des données.

| Champ | Type | Description |
|-------|------|-------------|
| Type | `uint16le` | Type du paquet |
| byte2 | `uint8` | — |
| byte3 | `uint8` | — |
| Size | `uint32le` | Taille des données du paquet (octets) |
| Timestamp | `uint64le` | Horodatage en microsecondes |

Types de paquets connus :

| Valeur | Nom | Note |
|--------|-----|------|
| 0 | `FRAME` | Frame de données |
| 1 | `START_CHUNK` | Début de chunk |
| 2 | `TYPE_2` | — |
| 6 | `TYPE_6` | — |
| 7 | `END_CHUNK` | Fin de chunk — marque la fin de l'itération |
| 8 | `PLAYER_METADATA` | Métadonnées joueur |
| 10 | `TYPE_10` | Intercalé avec les frames |
| 12 | `TYPE_12` | — |

---

## Résolution du player_index

Chaque joueur dans le film est repéré par un `player_index` (0–31). Pour associer
un XUID (format `xuid(1234567890123456)`) à son index dans le film (acurtis166) :

```python
from bitstring import Bits

def get_player_index(bits: Bits, player_id: str) -> int:
    """Retourne le player_index d'un joueur en cherchant son XUID dans le film."""
    if player_id.startswith("bid"):
        return -1  # bots non supportés
    term = Bits(uintle=int(player_id[5:-1]), length=64)
    position = bits.find(term, bytealigned=False)
    if not position:
        raise ValueError(f"Player {player_id} not found")
    # Les 5 bits précédant la première occurrence du XUID = player_index
    return bits[position[0] - 5 : position[0]].uint
```

Règles d'application :
- L'index est **stable** entre les chunks → peut être mis en cache après le 1er chunk
- Utiliser le `join_time` du payload stats pour **ignorer les chunks antérieurs** à
  l'arrivée du joueur dans la partie
- Sauter les joueurs déjà résolus (optimisation : stopper après la première occurrence)

**Option B — Packet `PLAYER_METADATA` (type 8)** :
Le service actuel (`_resolve_player_indices`) tente d'abord d'extraire les
associations `pi → xuid` depuis le packet de type 8 (`PLAYER_METADATA`, ~25KB)
présent dans le premier chunk, via `detect_pi_from_metadata`. C'est plus rapide
que la méthode acurtis car il évite le scan bitstring sur le chunk complet (~700KB).
La méthode acurtis (ci-dessus) n'est utilisée qu'en **fallback** si le packet
METADATA est absent ou ne couvre pas tous les joueurs.

> À vérifier : fiabilité de l'option B sur les replays anciens / matchs avec
> joueurs en retard de connexion (le packet peut être incomplet).

---

## Les deux chemins d'attribution

Pour chaque joueur dans un match, le pipeline choisit un chemin selon que c'est
le POV ou non.

### Chemin A — POV (Section 2, fire events)
**Pour qui** : le joueur élu "propriétaire" du film via méthode privée et non détaillée par Microsoft.

**Comment** :
1. Les kill_times (`time_ms`) sont chargés **depuis la DB** (`highlight_events`) avant
   tout téléchargement de chunks — ils servent à la fois à filtrer les chunks utiles
   et comme timestamps de référence pour la corrélation
2. On scanne tous les fire events du film pour `player_index = 1` (le POV),
   en accumulant une liste globale sur tous les chunks
3. Pour chaque kill à t=T, on cherche le dernier fire event **non encore réclamé**
   dans `[T−5s, T]`
4. L'arme de ce fire event devient l'arme du kill

**Fiabilité** : élevée. Fenêtre de 5s pour capturer les armes à delayed damage
(Cindershot, Ravager, etc.). Un delta court = haute confiance, un delta long =
confiance réduite (peut-être un tir raté juste avant ?).

### Chemin B — Coéquipiers T1 (Section 1, snapshots)
**Pour qui** : les coéquipiers dont on retrouve le `player_index` dans le film.
**Hors périmètre** : les adversaires — leurs données ne sont pas accessibles.

**Aucun fire event disponible pour les joueurs T1 non-POV.** La Section 2 n'encode
pas les tirs des autres joueurs. Même en résolvant correctement le `player_index`
de tous les coéquipiers, scanner le film à leur recherche ne retourne rien. Il est
inutile de tenter de les récupérer dans l'état actuel — il n'y a tout simplement
rien d'exploitable dans le film pour eux côté fire events.

**Comment (Section 1 uniquement)** :

La Section 1 contient des événements **Formula A** (`scan_formula_a`) : à chaque
changement d'arme (swap ou ramassage), le film enregistre un snapshot
`[20 00 02 pb ... wid:8B]` où `pi = pb >> 5`. Ces événements sont scannés pour
reconstruire une timeline à granularité chunk.

`build_weapon_timeline` produit pour chaque chunk :
- `timeline[chunk_idx][pi]` = **dernière** arme vue pour ce `pi` (état en fin de chunk)
- `swap_pis[chunk_idx]` = ensemble des `pi` ayant eu **> 1 arme distincte** dans ce
  chunk (détection de swap intra-chunk)

Attribution T1 pour un kill à `t_ms` :
1. Trouver le chunk couvrant `t_ms` via `find_chunk_at_time`
2. Lire `timeline[chunk][pi]` → weapon_id
3. Si `pi in swap_pis[chunk]` : `confidence = "medium"` (swap détecté dans la fenêtre
   ~19s, impossible de savoir si avant ou après le kill)
4. Sinon : `confidence = "high"` (arme connue) ou `"low"` (hex inconnu)

**Cas limite** : si `timeline[chunk]` est vide pour ce `pi`, fallback sur
`timeline[chunk - 1]` (le chunk précédent).

**Fiabilité** :
- `confidence=high` : le joueur n'a pas changé d'arme pendant le chunk — attribution
  solide
- `confidence=medium` : un swap Formula A a été détecté dans le même chunk (~19s).
  L'arme stockée est la **dernière** tenue dans ce chunk, pas nécessairement celle
  du kill. Ce cas est le principal vecteur d'imprécision T1.
- `confidence=low` : arme tenue identifiée (hex présent) mais non référencée dans
  `WEAPON_ID_MAP`

**Amélioration possible** : les événements Formula A ont une position en octets dans
le chunk. En les combinant avec `build_frame_estimator` (interpolation de timestamp
par position octet), on pourrait estimer à quelle milliseconde le swap a eu lieu et
comparer à `kill_t` → passer de `medium` à `high` si le swap est après le kill, ou
confirmer l'arme pré-swap si le swap est avant.

---

## Comment on identifie les armes (le mapping hex → nom)

Chaque arme dans le film est représentée par **8 octets** (un identifiant binaire).
La grande majorité des armes "standard" partagent le même suffixe (`42c9679f` pour
les 4 derniers octets). Les armes spéciales (variantes d'Energy Sword, Gravity Hammer)
ont des suffixes différents.

`WEAPON_ID_MAP` dans `_weapon_data.py` associe ces 8 octets au nom de l'arme.
Seules les armes **confirmées par investigation directe sur le film** sont dans ce
dictionnaire. Un hex inconnu reste inconnu — le kill est stocké avec `confidence=low`
et son ID numérique brut.

**Principe clé : tout hex est récupérable, connu ou non.**
Puisque tous les fire events partagent le même format et que le weapon_id est
toujours aux octets [6–13], le parser peut extraire et stocker cet ID même s'il
n'est pas dans `WEAPON_ID_MAP`. Un weapon_id inconnu aujourd'hui peut être résolu
plus tard (investigation film, nouvelle entrée dans le dictionnaire) sans
nécessiter un re-scan du film — l'ID brut est déjà en base. C'est exactement ce
que fait le parser actuel : les 15 377 kills `confidence=low` en base ont tous un
`weapon_id` numérique stocké, prêts à être nommés dès que l'hex est identifié.

**Exception — les sentinelles : `weapon_id = 0`, `1`, `2` ne viennent pas du film.**
Ces trois valeurs sont des **IDs artificiels** assignés par la logique de
détection (médailles), pas des weapon_ids lus dans le film :

| Valeur | Constante | Signification |
|--------|-----------|---------------|
| `0` | `GRENADE_WEAPON_ID` | Kill attribué à une grenade (via médailles) |
| `1` | `MELEE_WEAPON_ID` | Kill attribué à un coup de mêlée (via médailles) |
| `2` | `VEHICLE_WEAPON_ID` | Kill attribué à un véhicule |

Un vrai weapon_id film converti en `uint64` est toujours un très grand nombre
(ex. `0x6ACDC44D42C9679F` ≈ 7.7 × 10¹⁸). Les valeurs 0, 1 et 2 ne peuvent
donc pas apparaître naturellement dans le film — elles ne collisionnent jamais
avec de vrais IDs. Si `weapon_id = 0` apparaît en base, c'est que la détection
par médaille a classifié ce kill comme grenade, **pas** qu'un hex `000...0` a
été lu dans le film.

### Note de conception : faut-il stocker plusieurs weapon_ids par kill ?

**Intuition** : les sentinelles sont des fallbacks. Si demain la détection s'améliore,
les lignes avec `weapon_id = 0/1` en base ne servent plus à rien — on a écrasé des
données potentielles avec une approximation.

**Contre-argument initial :** appliquer une hiérarchie de confiance à l'écriture —
sentinelle autorisée uniquement sur `NULL` ou `confidence=low`, jamais sur `medium`
ou `high`. Ce fix résout Step 4b sans changer le schéma.

**Limite de ce contre-argument :** même avec la hiérarchie, on écrase quand même
le hex film d'un kill `confidence=low`. Si ce hex est identifié plus tard
("ah, c'était du Needler"), la ligne en base dit "melee" et l'information film est
perdue — il faudrait re-parser le film.

**Approche masque (proposition retenue) :**
Ajouter une colonne `reconciled_as UBIGINT` (NULL par défaut) distincte de `weapon_id`.

| Colonne | Contenu | Modifiable ? |
|---------|---------|--------------|
| `weapon_id` | Hex brut lu dans le film. Jamais écrasé. | ❌ non |
| `reconciled_as` | Sentinelle assignée par la réconciliation API (`0/1/2`). | ✅ oui, réversible |

L'attribution effective pour les requêtes = `COALESCE(reconciled_as, weapon_id)`.

**Avantages :**
- Zéro perte de données film : le hex reste toujours récupérable
- Réversible : `UPDATE SET reconciled_as = NULL WHERE ...` si la détection s'améliore
- Traçabilité : on distingue "détecté dans le film" vs "élu par l'API"
- Compatible avec la résolution future d'hex inconnus sans re-parser

**Coût :** toutes les requêtes doivent utiliser `COALESCE(reconciled_as, weapon_id)`.
Ce risque d'oubli est contenu en créant une vue :
```sql
CREATE VIEW v_weapon_kills AS
SELECT *, COALESCE(reconciled_as, weapon_id) AS effective_weapon_id
FROM weapon_kills;
```
Les consommateurs (UI, agrégats) lisent `v_weapon_kills.effective_weapon_id`,
jamais directement `weapon_id` seul.

**Règle d'éligibilité pour `reconciled_as` (inchangée par rapport à la hiérarchie) :**

| `confidence` (valeur string Python) | `reconciled_as` peut être positionné ? |
|-------------------------------------|----------------------------------------|
| `"high"` | ❌ jamais |
| `"medium"` | ❌ jamais |
| `"low"` | ✅ oui — hex inconnu, l'API est plus fiable |
| `"none"` | ✅ oui — aucune info film (pas de fire event / pas de snapshot) |

> **Note** : la valeur `"none"` (string Python) correspond à ce qu'on appelait
> `NULL` dans les discussions conceptuelles. En base DuckDB, la colonne `confidence`
> stocke la string `'none'` — jamais un SQL `NULL`. Ne pas confondre les deux.

---

## Les sentinelles (melee, grenade, véhicule)

### Détection actuelle : médailles

La méthode actuelle utilise les **médailles** obtenues dans les 500ms autour du
kill pour classifier les sentinelles :

| Médaille présente | → Attribution |
|-------------------|---------------|
| Pummel, Back Smack, Ninja, Assassination, Pancake… | `weapon_id = 1` (melee) |
| Sticky Fingers, Grenadier, Boom!, Stick… | `weapon_id = 0` (grenade) |
| — (aucune médaille spécifique) | Chemin normal (fire event / snapshot) |

### Événements melee dans le film (acurtis166, mars 2026)

> **POV uniquement.** Les events melee partagent le même layer nibble-shifté et
> la même structure d'encodage que les fire events. Puisque les fire events sont
> confirmés POV-only, les events melee le sont très probablement aussi — non
> exploitables pour les coéquipiers T1 ni pour les adversaires.

Les coups de mêlée ont leur propre type d'event dans le film, identifié par le
marqueur `0xd340`. Ils partagent les **mêmes weapon_ids** que les fire events
et ajoutent un champ **animation type** (`5` ou `d`) qui distingue les deux
animations melee possibles selon l'arme :

| Arme | Animation `5` | Animation `d` |
|------|---------------|---------------|
| BR75 | Weapon toe bas→haut | Weapon toe d→g |
| Energy Sword (toutes variantes) | Slash diag. haut-g | Stab d→g |
| Gravity Hammer / Rushdown / Diminisher / Mythic Sandwich | Smash A | Smash B |
| Mk51 Sidekick | Main gauche | Grip arme |
| Mangler | Bayonette g→d | Bayonette haut-d→bas-g |
| MA40 AR / Cindershot / Bulldog / Heatwave / Bandit Evo / Hydra / Commando / Stalker / Pulse Carbine / Sniper / Sentinel Beam / Vestige | Coude droit | Weapon toe |
| Needler | Fond arme | Stab aiguille |
| Skewer / Ravager | Slash bayonette | Stab bayonette |

Le bouton "fire" des armes melee (Hammer, Sword) utilise les mêmes weapon_ids
avec le marqueur `0xd340` mais peut avoir une structure légèrement différente
(non entièrement caractérisée).

**Opportunité pour le nouveau parser** : les events melee du film permettraient
d'attribuer les kills melee directement sans dépendre des médailles.

---

## La réconciliation API

L'API Halo fournit, par match et **par joueur** (table `match_participants`), les
colonnes `grenade_kills` et `melee_kills`. Ces agrégats de fin de match sont
disponibles pour **tous** les joueurs — POV comme coéquipiers.

Le parser actuel ne les utilise que pour le POV (la réconciliation est absente du
chemin T1). Pour le nouveau parser, ce signal est exploitable pour les coéquipiers
aussi : si le parser attribue 3 kills grenade à un coéquipier alors que l'API en
compte 1, il y a surestimation.

Mécanisme actuel (POV uniquement) :

- **Trop de kills arme** → on rétrograde les moins sûrs de `high` à `medium`
- **Pas assez de kills melee/grenade** → on reclassifie les kills les plus incertains
  en melee/grenade pour combler le manque (Step 4b)
- **Pas assez de kills arme** → on promeut des `medium` en `high`

**Point de friction** : le Step 4b puise dans les kills les moins certains, et
les kills T1 à hex inconnu (`confidence=low`) sont les premiers candidats. Résultat :
des kills réels avec une arme inconnue peuvent être reclassifiés en grenade/melee
pour "équilibrer les comptes" avec l'API.

**Opportunité pour le nouveau parser** : étendre la validation grenade/melee via
`match_participants.grenade_kills` / `melee_kills` aux coéquipiers aussi, pas
uniquement au POV.

---

## Pourquoi il y a des weapon_id NULL en base

Un kill NULL signifie que le parser n'a trouvé **aucune info** sur l'arme :
- Chemin POV : aucun fire event dans les 5s avant le kill (vehicule, edge case)
- Chemin T1 : le film n'a encodé aucun snapshot pour ce joueur dans ce chunk

Le tableau ci-dessous montre la réalité en base (85 247 kills au total) :

| État | Kills | % |
|------|------:|--:|
| NULL — pas d'info | 54 313 | 63.7% |
| Hex inconnu (`conf=low`) | 15 377 | 18.0% |
| Arme identifiée (`conf=high`) | 10 631 | 12.5% |
| Melee / Grenade sentinelle | 4 447 | 5.2% |

Les NULL viennent quasi-exclusivement du chemin T1 pour les adversaires — c'est
pour ça qu'ils sont exclus du périmètre du nouveau parser : le film d'un joueur
n'encode ni les fire events ni les snapshots arme des joueurs adverses de façon
exploitable.

---

## Zones de confiance POV — seuils par arme

La fonction `_get_confidence(weapon_id, delta_ms)` applique une logique à 3 zones
basée sur `WEAPON_TIMING_BY_ID[weapon_id] = (swap_ms, travel_max_ms)` :

| Zone | Condition | `confidence` | Signification |
|------|-----------|:------------:|---------------|
| A | `delta_ms < swap_ms` | `high` | Swap physiquement impossible dans cette fenêtre |
| B | `swap_ms ≤ delta_ms ≤ travel_max` | `medium` | Fenêtre ambiguë — l'arme a pu changer |
| C | `delta_ms > travel_max` | `low` | Delayed damage — l'arme du fire event est suspecte |

**Valeurs par arme (extraites de `WEAPON_TIMING` / `_weapon_data.py`)** :

| Classe d'armes | `swap_ms` | `travel_max_ms` |
|----------------|----------:|----------------:|
| Sidekick | 400 | 300 |
| Plasma Pistol | 450 | 300 |
| AR, BR75, Bandit, Bulldog, Commando, Shock Rifle, Mangler… | 650 | 500 |
| Heatwave, Needler, Hydra | 650 | 2000 |
| Ravager, Disruptor, Cindershot | 650 | **5000** |
| Sniper, Stalker, Mutilator | 900 | 300 |
| Skewer | 900 | 3000 |
| SPNKr, Fuel Rod, Gravity Hammer, Energy Sword (toutes) | 1100 | 1400 |
| M41 SPNKr, Fuel Rod SPNKr | 1100 | 2000 |
| *Défaut (arme inconnue)* | 650 | 2000 |

**Conséquence pour le rewrite** : les seuils sont par-arme, donc l'attribution de
confiance ne peut être calculée qu'après avoir résolu le `weapon_id`. Pour les kills
`confidence=low`, la règle est : `delta_ms > travel_max_ms` pour cette arme — le
feu était probablement le bon mais le projectile a voyagé très longtemps.

### Zone B — mécanisme W2 check (`_check_zone_b_swap`)

Quand un kill tombe en Zone B (`confidence=medium`), une vérification supplémentaire
cherche si le POV a **changé d'arme immédiatement après** le kill :

```python
post_swap = [
    ev for ev in fire_events_all
    if kill_t < ev["timestamp_ms"] <= kill_t + swap_ms
    and ev["weapon_bytes"] != best["weapon_bytes"]
]
```

Logique : si dans `[kill_t, kill_t + swap_ms]` un fire event **d'une arme différente**
existe, c'est que le joueur a swappé *après* le kill — preuve que l'arme W1 (celle
du fire event candidat) était bien l'arme du kill. On retient alors W2 (l'arme
post-swap) comme arme du kill et on remonte la confiance à `high`.

> **Attention** : la logique retient l'arme **W2** (l'arme après le swap), pas W1.
> L'hypothèse implicite est que le joueur a swappé vers W2 *à cause* du kill
> (ex. il vidait son chargeur, il a switch immédiatement). Cette hypothèse n'est
> pas toujours correcte — si le joueur avait déjà commencé à swapper avant le kill,
> W1 serait la bonne réponse. À réévaluer dans le rewrite.

---

## Optimisations identifiées pour le rewrite

### 1. Déduplication fire event → un seul kill (claim-and-remove)

**Bug actuel :** `_match_kill_to_fire_event` fait `max(candidates, by=timestamp)` sans
marquer les fire events comme utilisés. Deux kills consécutifs dans une fenêtre de 5s
peuvent tous deux "piquer" le même fire event — notamment pour les armes AOE (Hammer,
Cindershot) qui peuvent produire deux kills presque simultanés.

**Solution avec `highlight_events`** : les timestamps des kills sont connus à la ms via
`highlight_events.time_ms`. Trier les kills par timestamp croissant, puis pour chaque kill :
1. Chercher le dernier fire event **non encore réclamé** dans `[kill_t - 5000, kill_t]`
2. Le marquer comme réclamé (le sortir du pool) une fois attribué

Ce mécanisme s'applique au **POV et aux coéquipiers T1** via le même timestamp de
référence — `highlight_events` est disponible pour tous les joueurs d'un match.

Pour T1, l'apport est différent : au lieu de "quel chunk couvre ce kill" (fenêtre ~19s),
on sait exactement `kill_t` → le snapshot actif est "le dernier état enregistré avant
`kill_t`", plus précis qu'une granularité chunk.

### 2. Cross-chunk — architecture deux-phases

La clé est la séparation stricte entre **phase de scan** et **phase de corrélation**.

**`_scan_player_chunks`** (service) :
```python
all_events: list[dict] = []
for _idx, (chunk_data, start_ms, dur_ms) in sorted(chunks.items()):
    packets = index_chunk(chunk_data)
    all_events.extend(
        scan_fire_events(chunk_data, player_index, start_ms, dur_ms, packets=packets)
    )
all_events.sort(key=lambda e: e["timestamp_ms"])
return all_events
```

Tous les chunks sont parcourus **en ordre croissant**, les fire events sont accumulés
dans une liste plate unique, puis triés par `timestamp_ms` absolu. Le résultat est
une liste globale couvrant l'intégralité du match, sans notion de frontière de chunk.

**`correlate_kills_to_weapons`** (parser) reçoit cette liste globale et, pour chaque
kill à `kill_t`, filtre :
```python
candidates = [ev for ev in fire_events_all
              if (kill_t - KILL_WINDOW_MS) <= ev["timestamp_ms"] <= kill_t]
```

Un kill à `t = 19 050 ms` (début chunk N+1) trouvera sans problème un fire event
à `t = 18 800 ms` (fin chunk N), car les deux sont dans la même liste plate.

> **Règle de conception** : conserver l'architecture deux-phases (scan global →
> corrélation globale). Ne pas traiter les chunks un par un dans `correlate`.

### 3. Skip des chunks sans kill

Calculer un ensemble `needed: set[int]` en filtrant
les chunks de la métadonnée film par rapport aux `kill_times_ms` :

```python
needed: set[int] = set()
for kill_t in kill_times_ms:
    window_start = kill_t - KILL_WINDOW_MS
    for ch in chunks_meta:
        if ch.chunk_type.value != 2:
            continue
        ch_start = ch.chunk_start_time_offset_milliseconds
        ch_end = ch_start + ch.duration_milliseconds
        if ch_end >= window_start and ch_start <= kill_t:
            needed.add(ch.index)
```

Seuls les chunks dont la fenêtre temporelle `[ch_start, ch_end]` chevauche
`[kill_t - 5000ms, kill_t]` sont téléchargés. Les chunks sans aucun kill dans
leur fenêtre sont ignorés — téléchargement et parsing.

> **Règle de conception** : les `kill_times_ms` proviennent de `highlight_events`,
> chargés avant le téléchargement des chunks.

---

## Résumé des limites actuelles

| Limite | Impact |
|--------|--------|
| Section 2 (fire events) uniquement pour le POV | Les coéquipiers ont une moins bonne couverture ; les adverses sont exclus |
| Film n'encode pas les adverses | → 63% de NULL actuels, justifie leur exclusion du périmètre |
| 279 hex inconnus (T1 low confidence) | → 18% des kills "identifiés" mais sans nom |
| Step 4b reclassifie des hex inconnus en grenade/melee | → faux mélanges dans les stats grenade/arme du POV |
| `WEAPON_ID_MAP` à 36 armes confirmées | → tout hex hors map reste inconnu |
