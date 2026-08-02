# PROTOCOLE DE VÉRIFICATION — nommer les emplacements d'arme au Théâtre

> Écrit le 2026-08-02. Branche `feat/re-mode-score`.
> Table source : `.ai/V7.5/dumps/forge_zones/emplacements_a_observer.txt`.
> Contexte : état de l'art `.ai/ETAT_DE_L_ART_FORGE_PALETTE_ZONES.md` §Q1.0-decies.

## Pourquoi ce document existe

Deux corrections successives de l'utilisateur ont remis le protocole d'aplomb.

1. **« Les socles sont juste des images sans aspect particulier en jeu. »** Demander
   « lequel de ces deux socles est-ce ? » n'a pas de réponse. La mesure le confirme : sur
   Vagabond les deux socles portent la MÊME variante de caisse et la MÊME cadence. Le seul
   observable est **l'objet qui apparaît dessus**.
2. **« Ça peut varier selon les modes de jeu. »** Vérifié, et c'est vrai deux fois :
   - le mode change la **cadence** : les mêmes positions sur Fragmentation valent 120 s en
     variante de base et **45 s** en variante Heavies ;
   - et surtout, **tous** les matchs récents de l'utilisateur sur Cliffhanger, Catalyst et
     Launch Site sont en **Super Fiesta**, où l'armement est randomisé. Ces cartes-là sont
     donc inutilisables telles quelles.

D'où ce document : cinq matchs **en mode normal**, datés à la minute, avec les positions,
les cadences et l'hypothèse à confirmer ou à réfuter.

## Ce qu'il faut nommer

Quatre `Representation Name` couvrent **285 socles sur 199 cartes**, plus un cas à part.
Nommer les cinq nomme tout.

| identifiant | socles | cartes |
|---|---:|---:|
| `-1351408675` | 117 | 20 |
| `-1412311642` | 98 | 27 |
| `-245254093` | 46 | 21 |
| `-219174009` | 24 | 11 |
| `493070541` *(cas à part : tag `weap` direct, sans Representation Name)* | 3 | 1 |

## Les cinq observations

Cadence retenue partout : **90 à 150 s** — le régime des objets de puissance. Les socles à
30-45 s sont écartés, ce sont vraisemblablement des armes de base.

### 1. `-1412311642` — hypothèse : LANCE-ROQUETTES

| | |
|---|---|
| **Carte** | Empyrean |
| **Quand** | **2026-07-28 à 22:18** (heure de Paris) |
| **Mode** | Team Slayer:Arena — Quick Play |
| **Durée** | 10 min 24 s · film `ac03413d` |
| **Où** | **deux socles côte à côte, 9,3 m d'écart, à la même hauteur** — (−4,6 · −43,7 · 142,5) et (+4,7 · −43,7 · 142,5) |
| **Cadence** | 150 s |

Hypothèse issue du relevé Vagabond : l'utilisateur y atteste un lance-roquettes sur le socle
bas, qui porte ce même identifiant. **Si Empyrean rend autre chose, l'hypothèse « le
Representation Name porte l'identité de l'objet » est réfutée** — c'est le test le plus
important des cinq.

Témoin de repli, même identifiant : **Argyle, 2026-07-17 à 22:35**, Team Slayer:Arena.

### 2. `-1351408675` — hypothèse : AUCUNE (117 socles en jeu)

| | |
|---|---|
| **Carte** | Argyle |
| **Quand** | **2026-07-23 à 21:55** (heure de Paris) |
| **Mode** | Team Slayer:Arena — Quick Play |
| **Durée** | 9 min 01 s · film `946034f3` |
| **Où** | **deux socles diamétralement opposés autour du centre, 18,3 m d'écart, même hauteur** — (+9,0 · −2,0 · 50,6) et (−8,9 · +2,0 · 50,6) |
| **Cadence** | 120 s |

C'est l'observation la plus rentable : elle nomme **117 socles sur 20 cartes** d'un coup.

### 3. `-219174009` — hypothèse : AUCUNE

| | |
|---|---|
| **Carte** | Chasm |
| **Quand** | **2026-07-07 à 22:46** (heure de Paris) |
| **Mode** | CTF:Arena Neutral Flag — Quick Play |
| **Durée** | 7 min 39 s · film `e8d384c7` |
| **Où** | **deux socles éloignés, 39,5 m d'écart, même hauteur** — (−49,8 · −52,9 · −138,7) et (−84,6 · −71,6 · −138,7) |
| **Cadence** | 120 s |

### 4. `-245254093` — hypothèse : CAMOUFLAGE ACTIF

| | |
|---|---|
| **Carte** | Insolence |
| **Quand** | **2026-07-24 à 22:06** (heure de Paris) |
| **Mode** | BTB:Total Control — Big Team Battle |
| **Durée** | 11 min 22 s · film `5676a9ba` |
| **Où** | **les deux socles à cadence longue** — (−24,8 · +8,1 · 68,8) et (−33,8 · −32,8 · 65,2). Ignorer les deux autres socles de la carte, à 30 s |
| **Cadence** | 120 s |

Hypothèse issue du relevé Vagabond : l'utilisateur y atteste un camouflage sur le socle
haut, qui porte cet identifiant. **Réserve honnête** : le mode BTB est moins « standard »
que Team Slayer ; en cas de doute, le témoin de référence reste Vagabond lui-même —
**2026-07-17 à 21:19**, Strongholds:Arena, film `696a9d7c` (celui déjà utilisé pour l'oracle
de zone), socle haut à (144,9 · 53,3 · 54,8), 120 s.

### 5. `493070541` — hypothèse : AUCUNE, et c'est le plus simple

| | |
|---|---|
| **Carte** | Catalyst |
| **Quand** | **2026-07-07 à 22:34** (heure de Paris) |
| **Mode** | Team Slayer:Arena — Quick Play |
| **Durée** | 10 min 30 s · film `ab004e99` |
| **Où** | **une paire symétrique 13 m d'écart** — (11,3 · −6,5 · 22,9) et (11,3 · +6,5 · 22,9) ; **plus un socle isolé à l'opposé, 1,4 m plus haut** — (−12,1 · 0,0 · 24,3) |
| **Cadence** | 90 s |

Ce type n'est pas une entrée de palette : c'est **le tag `weap` lui-même** (14 088 octets).
Ce qui apparaît là nomme donc directement une arme, sans intermédiaire. Et la paire porte
la variante `banished_shock` tandis que l'isolé porte `banished_plasma` : **ils devraient
donc être deux armes différentes**. C'est une seconde prédiction falsifiable.

## Ce qu'une réponse doit contenir

Pour chaque socle : **l'arme ou l'objet de puissance qui y apparaît**. Rien d'autre n'est
nécessaire — ni la position, ni l'heure, elles sont déjà fixées ici.

Un désaccord est aussi utile qu'un accord : il réfute l'hypothèse que le
`Representation Name` porte l'identité de l'objet, et il faudra alors chercher l'identité
ailleurs (le troisième mot d'identité non identifié, offsets 688/732, reste candidat).

## Cartes ÉCARTÉES, et pourquoi

| carte | raison |
|---|---|
| Cliffhanger, Launch Site, Catalyst *(matchs de juillet)* | tous en **Super Fiesta** — armement randomisé |
| Fragmentation, Scarr | dernier match en **2025**, hors fenêtre Théâtre plausible |
| Solitude | dernier match le 2026-06-03, plus ancien qu'Insolence pour le même identifiant |
| tous les socles à 30-45 s | régime d'arme de base, pas d'objet de puissance |
