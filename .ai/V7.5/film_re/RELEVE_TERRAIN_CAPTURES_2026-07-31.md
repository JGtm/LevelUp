# RELEVE TERRAIN — captures Cheat Engine du 2026-07-31

> Une capture sans releve est infalsifiable, donc sans valeur. Ce fichier EST l'oracle des
> deux captures continues de la soiree. A conserver AVEC les CSV.
>
> Outillage : `filmdec_full_capture.lua`, hook @ 7FF6911CCD11 (dispatch des composants),
> buffer 2 097 152 records de 40 o. Record : eid(4) typeIndex(4) compIndex(4) param4(4)
> bitCursor(4) skipCount(4) signature(16).

---

## Fichiers produits

| fichier | contenu | taille |
|---|---|---|
| `deser_table.tsv` | table des deserialiseurs — 50 archetypes, 1068 composants | 47 Ko |
| `archetype_vtables.tsv` | descripteur + vtable des 50 archetypes (adresses Ghidra) | 1,6 Ko |
| `filmdec_full_696a9d7c_strongholds_nomad.csv` | 1 205 704 records | 70,7 Mo |
| `filmdec_full_530820e5_ctf_catalyst.csv` | 988 752 records | 57,9 Mo |

Controle de la table des deserialiseurs, PASSE : archetype 35 = 64 composants ET son
`vtable[0x60]` = `0x140F44C38`. Les deux volets sont tombes. La structure n'a pas bouge.

---

## Capture 1 — `696a9d7c` · Strongholds:Arena · Vagabond (Nomad)

**Identite PROUVEE** — pas supposee. 401 signatures de 16 octets echantillonnees sur toute
la longueur de la capture, cherchees telles quelles dans les chunks :

| film | trouvees / 401 | taux |
|---|---|---|
| **696a9d7c** | 397 | **99,0 %** |
| 64e8adfa (temoin) | 3 | 0,7 % |
| 000d5950 (temoin) | 2 | 0,5 % |

Ce sont les temoins qui donnent sa valeur au resultat : une signature de 16 octets pourrait
etre banale, ils prouvent qu'elle ne l'est pas.

### Releve a l'oeil (utilisateur)

| temps film | observation | borne dans le flux |
|---|---|---|
| 0:48 | **flyguy8773 capture la base B** | record #105 523 |
| 1:30 | une equipe controle **les trois bases**, score 21 pour l'autre | record #211 604 |
| 3:10 | **score 69 - 30** | record #419 300 |
| 5:34 | **controle des trois bases par l'equipe de flyguy8773** | record #825 733 |

Les bornes sont des BORNES SUPERIEURES : le compteur est lu apres l'annonce, la latence de
saisie n'est pas mesuree.

### Calibration index de record <-> temps du film

| ancre | t (s) | record | records/s |
|---|---|---|---|
| base B | 48 | 105 523 | 2 198 |
| 3 bases | 90 | 211 604 | 2 351 |
| score 69-30 | 190 | 419 300 | 2 207 |
| 3 bases (2) | 334 | 825 733 | 2 472 |

Ecart au debit median ~7 %. L'index de record est une horloge APPROCHEE du temps de film :
bon pour situer a quelques secondes, PAS pour dater finement. La datation exacte passe par
l'alignement de signature.

### Ventilation finale — 1 205 704 records, 2 095 entites

`ti=35`:809 350 · `ti=37`:147 005 · `ti=42`:67 887 · `ti=41`:63 152 · `ti=4`:34 110 ·
`ti=10`:26 006 · `ti=12`:18 903 · `ti=47`:14 777 · `ti=43`:11 972 · `ti=13`:4 571 ·
`ti=5`:2 908 · `ti=6`:2 716 · `ti=2`:1 312 · `ti=21`:702 · `ti=38`:327 · `ti=45`:6

**`ti=23` (zones) : ABSENT. `ti=11` (objectifs) : ABSENT.** Zero dispatch sur 1 205 704
records, alors que QUATRE evenements d'objectif sont releves a l'oeil sur la meme periode.

---

## Capture 2 — `530820e5` · CTF:Arena · Catalyst

**Identite PROUVEE** : 400/401 = **99,8 %** ; temoins `696a9d7c` 0,0 % et `64e8adfa` 1,0 %.

### Releve a l'oeil : ABSENT — a dire franchement

Aucune observation n'a ete relevee pendant cette lecture. La capture n'a donc PAS d'oracle
humain. Elle n'est pas pour autant sans valeur : trois oracles independants existent hors
ligne pour ce film, et ils sont documentes.

1. le **chunk type-3 (footer d'events)** est EN CACHE (`chunk_26.bin`) — il porte les events
   `type_hint=10` (interactions de drapeau) avec xuid acteur et `time_ms` precis ;
2. le **detecteur de capture `tiers==6`** (burst FRAME re-transmettant les 6 records de
   l'echelle de score) — mesure 0 manque / 0 faux positif sur 4 matchs ;
3. le stat par joueur **`FlagCaptures`** en base.

Refaire la lecture avec un releve a l'oeil reste souhaitable si l'occasion se represente,
notamment pour SURBOUCLIER et CAMOUFLAGE (les deux equipements qui manquent a la table des
capacites, qui n'en connait que 4 sur 11) : ces deux-la n'ont PAS d'oracle hors ligne.

### Ventilation finale — 988 752 records, 1 364 entites

`ti=35`:627 657 · `ti=37`:158 415 · `ti=42`:108 464 · `ti=41`:33 525 · `ti=4`:29 054 ·
`ti=43`:7 415 · `ti=10`:7 485 · `ti=47`:5 624 · `ti=12`:2 926 · `ti=5`:2 495 · `ti=6`:2 340 ·
`ti=13`:1 246 · `ti=2`:1 103 · `ti=38`:363 · `ti=21`:308 · **`ti=11`:162** · `ti=27`:128 ·
`ti=26`:18 · `ti=45`:10 · `ti=9`:8 · `ti=34`:4 · `ti=25`:2

**`ti=11` (objectifs) : 162 dispatches. `ti=23` (zones) : ABSENT.**

Six archetypes n'apparaissent QUE sur ce film et jamais sur le Strongholds : `ti=9`, `ti=25`,
`ti=26`, `ti=27`, `ti=34`, `ti=38`.

---

## LE RESULTAT DE LA SOIREE — ce que la comparaison des deux etablit

`ti=11` est dispatche en CTF (162 fois) et jamais en Strongholds (0 sur 1,2 M). L'absence
n'est donc **ni un defaut de capture, ni un chemin de code inaccessible : elle est propre au
mode**. Le drapeau du CTF est une entite du flux de replication ; les bases du Strongholds
n'en sont pas une — en tout cas pas sous l'indice 23.

Ce constat est CONFIRME PAR UNE SECONDE CHAINE, independante et anterieure
(`.ai/archive/V7/RESEARCH_THEATER_RE.md` §M / M-ter, juin 2026) :

- les evenements de mode a objectif (`type_hint=10`) vivent dans le **chunk type-3**, pas
  dans la replication ;
- le score continu vit dans **TYPE_2** (le snapshot ~20 s), comme un varint dont l'offset
  DERIVE d'un match a l'autre — d'ou la regle « lecture ancree sur un marqueur local par
  match, jamais d'offset en dur » ;
- ce document supposait que l'identite de la zone (A/B/C) serait accessible « via la machine
  d'etat zone (replication) ». **Notre capture refute cette piste** : la zone n'est pas non
  plus dans la replication. C'est un resultat NEGATIF neuf, et il evite de perdre une session
  a chercher la.

Consequence pratique : pour les objectifs, chercher dans les IMAGES-CLES et le footer
d'events, pas dans le flux delta. Les deux sont deja en cache — donc decodable hors ligne,
sans jeu et sans Cheat Engine.

## Cas non resolu que la capture 2 vise directement

`530820e5` est un **CTF:Arena** (a 3). Le document de juin classe ce cas « NON cracke » : la
capture discrete n'y est pas materialisee comme champ stable dans le snapshot 20 s, elle est
« stockee en accumulateurs de flag-carry ». Les 162 dispatches de `ti=11`, lisibles grace aux
34 composants de la table des deserialiseurs prise le meme soir, sont le cote replication de
cette machine d'etat. C'est la piece qui manquait.
