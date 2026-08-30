# Les familles de trames : confirmation positive du modele, et l'en-tete qui varie par famille

Date : 2026-08-30. Ouverture du lot 1 du plan « percer la trame »
(`../PLAN_PERCER_TRAME_FILM_2026-08-30.md`). Instruments (garde `LOT1_TRAME_FILM`, un film par
process, verrou de decodage pris) : `lot1_familles_trame_research_test.go` — trois tests, tous
sur le film calibre `000d5950`, 12 chunks de replication, monde amorce par les images-cles du
chunk (meme mecanique que `delta_walk_witness_test.go`, dont la valeur est la COMPARABILITE
interne, pas la justesse absolue).

## 1. Confirmation positive du modele de trame (le point 6 laisse ouvert par le lot D)

> AVERTISSEMENT (ajoute le meme jour) : cette section lit toutes les familles sous l'amorce
> k=2. Le « record DEL de tete » qu'elle rapporte est un ARTEFACT de ce cadrage — retracte
> plus bas (« Suite de session »). Les taux de fermeture et les comptes de slots restent des
> mesures valides DU CHAMP LU SOUS k=2.

`TestLot1FamillesTrame` rejoue `DecodeFrameRecords` (le decodeur de production) par famille de
PREMIER OCTET, reference interne = `0xA0` (la trame de tick, 80 % du corpus) :

| famille | paquets | fermes proprement | records | slots distincts du 1er record | 1er record |
|---|---|---|---|---|---|
| `0xA0` | 12 670 | 36,3 % | 37 149 | 10 | delta ti=4 (moteur de jeu) |
| `0xD2` | 245 | **34,3 %** | 428 | 12 | **DEL x245 (100 %)** |
| `0xD3` | 125 | 16,0 % | 322 | **47** | **DEL x125 (100 %)** |
| `0x89` | 103 | 41,7 % | 311 | 94 | NEW ti=41/37/38/42 (armes au sol) |
| `0xCA` | 97 | 29,9 % | 223 | 12 | NEW ti=32..35 (bipedes : spawns) |
| `0xC0/C2/C3/C4/C7` | 645 | 100 % « vides » | 0 | — | fin immediate, payload NON LU |

Verdicts (criteres ecrits avant mesure) : **L1-C1 [0xD2] TENU** (34,3 % vs seuil 18,1 %) ;
L1-C1 [0xD3] RATE de 2,1 points (16,0 %) — publie tel quel ; **L1-C2 [0xD3] TENU** (47 slots
distincts, attendu 30..70 : recoupe les « 50 identifiants » du lot D par une voie SANS etape
commune — ici on decode l'en-tete, le lot D lisait une fenetre de bits fixe) ; L1-C2 [0xD2]
RATE (12 > 10 attendus — le lot D en voyait 7 sur une fenetre plus etroite).

**Acquis structurel n1 : les trames `0xD2` et `0xD3` commencent TOUTES par un record DEL** —
la suppression d'une entite. Et ces entites sont des TRANSITOIRES : aucune n'est jamais
declaree par une image-cle (`non-lie` 245/245 et 125/125) — le profil d'un projectile ou d'un
effet, cree par NEW dans une trame anterieure et detruit a l'impact. Les « 12 slots » de
`0xD2` forment une petite bande recyclee (245 suppressions sur 12 slots).

## 2. Les « trames vides » ont des vues 2 et 3 (`TestLot1VuesMultiples`)

Le frame-processor de l'exe lit TROIS vues de replication par paquet ; le decodeur sequentiel
n'en lit qu'une. Prediction ecrite avant mesure : les familles « vides » portent leurs records
dans les vues suivantes. Mesure (`DecodeFrameViews(3)`, inference de chaine active) :
`0xC0` passe de 0 record a **454 records et >= 2 vues sur 100 % des paquets** ; `0xC2`
54 % >= 2 vues ; toutes les familles `0xC*` rendent desormais des records sur 100 % des
paquets. RESERVE MESUREE : la couverture moyenne depasse 100 % du payload sur `0xC0`/`0xD3`
(l'inference lit AU-DELA de la fin du paquet et fabrique des records fantomes) — les comptes
multi-vues sont donc CONTAMINES et ne valent que comme signal d'existence, pas comme mesure.

## 3. LA TROUVAILLE : l'en-tete de paquet varie par famille (`TestLot1AmorceParFamille`)

Le deuxieme bit d'amorce vaut 0 sur `0xA0`/`0x80` et 1 sur toutes les familles mal lues ; le
desassemblage n'etablit qu'UN bit d'amorce (le drapeau de configuration lu par
FUN_142987460), le second n'a jamais ete localise. Balayage de l'amorce k=1..16, departage par
le discriminant etabli du depot (part de masques a 1..7 composants sur les deltas lies —
84,8 % sous la bonne grammaire, 10,7 % au hasard), plancher n >= 50 :

| famille | k gagnant | masques 1..7 | n | verdict |
|---|---|---|---|---|
| `0xA0` | **2** | 99,2 % | 21 546 | temoin A1 : l'instrument est valide |
| `0xC2` | **6** | **99,3 %** | 135 | SEUL au-dessus du plancher |
| `0xD2` | **8** | 86,2 % | 80 | NET (suivant : k=10 a 66,3 %) |
| `0xD3` | 6 | 41,5 % | 82 | faible — l'en-tete de 0xD3 n'est pas un simple decalage fixe |
| `0xC0`, `0xC3`, `0xC7`, `0xE5`, `0xE9` | — | — | < 50 | NON CONCLUANT sur 12 chunks |

**Acquis structurel n2 : les familles a bit 2 = 1 portent un EN-TETE SUPPLEMENTAIRE, de
largeur PROPRE A LA FAMILLE** (4 bits de plus pour `0xC2`, 6 pour `0xD2`). Une fois cet
en-tete saute, `0xC2` se decode au niveau du flux de tick. Le premier octet n'est donc ni un
type d'evenement (lot D) ni le debut direct des records pour ces familles : il contient un
champ d'en-tete non identifie, dont la largeur varie — vraisemblablement un code a prefixe.

## Ce que cela change pour la suite du lot 1

1. Le gisement `0xD3` s'ouvre par la semantique de cet en-tete (6+ bits), pas par les offsets
   figes de `fire_events.go` : ceux-ci lisent A TRAVERS l'en-tete et le record DEL — leurs
   « champs » (arme aux bits 44..107) chevauchent des frontieres de structure, ce qui ne les
   empeche pas d'etre empiriquement stables mais interdit de les etendre.
2. La victime : si le DEL de tete supprime le projectile, la victime vit dans les records
   SUIVANTS de la meme trame (62 % d'entre eux se decodent deja sur `0xD2`).
3. Gestes suivants, dans l'ordre : (a) identifier le champ d'en-tete (Ghidra : l'autre branche
   du record-loop FUN_1406cd128, selectionnee par `DAT_14474cd78`, lit un id dont la LARGEUR
   depend du bit de configuration du paquet — piste directe) ; (b) re-balayer k par famille
   sur PLUS de chunks pour faire tomber les NON CONCLUANT ; (c) une fois l'en-tete su,
   decoder les records suivants de `0xD2`/`0xD3` et confronter au golden killsource (garde-fou
   du plan : le nouveau se valide contre l'ancien, jamais l'inverse).

## Suite de session (meme jour) — retractation, generalisation, et ce qui resiste

### RETRACTATION : le « record DEL de tete » etait un artefact de cadrage

La section 1 lit les trames 0xD2/0xD3 sous l'amorce k=2 (celle de 0xA0). Sous le cadrage
PROPRE de la famille (k=8 pour 0xD2), le premier record n'est PAS un DEL : c'est un **DELTA
sur un slot transitoire non lie** (172/245 sur 0xD2 ; 183/183 sur 0xC2 a k=6 ; 125/125 sur
0xD3 a k=6). Le « DEL » de la premiere passe etait la lecture des bits d'en-tete + debut du
vrai record a travers un mauvais cadrage. CE QUI SURVIT a la retractation : le premier record
vise une entite transitoire (jamais declaree par une image-cle), et les comptes de slots
distincts (12 / 47) restent des mesures du meme champ d'identifiant.

### GENERALISATION : le cadrage se reproduit sur un second film

`TestLot1AmorceParFamille` sur `00502e52` (meme protocole, 12 chunks) :

| famille | 000d5950 | 00502e52 | verdict |
|---|---|---|---|
| `0xA0` | k=2, 99,2 % (n=21 546) | k=2, 99,4 % (n=23 658) | temoin stable |
| `0xC2` | k=6, 99,3 % (n=135) | **k=6, 98,4 % (n=62)** | **REPRODUIT — cadrage etabli sur 2 films** |
| `0xD2` | k=8, 86,2 % (n=80) | **k=8, 98,2 % (n=57)** | **REPRODUIT — cadrage etabli sur 2 films** |
| `0xD3` | k=6, 41,5 % (n=82) | k=6, 36,5 % (n=104) | reproductible mais FAIBLE : en-tete plus complexe, OUVERT |
| `0xC0` | non concluant | k=8, 73,2 % (n=56) | a re-mesurer (un seul film concluant) |
| `0xC3`, `0xC7`, `0xE5`, `0xE9` | non concluant | non concluant | OUVERT (effectifs < 50 sur 12 chunks) |

### NEGATIFS PUBLIES (deux instruments qui ne tranchent pas)

- **Cadrage PAR PAQUET** (`TestLot1EnteteParPaquet`, critere fermeture + couverture >= 50 %) :
  les k gagnants DIVERGENT entre paquets a en-tete identique (k=7/13/15/20 sur la meme tete
  `11010010 01100001`) et 0xE9 ne rend aucun k sur 255 paquets. La fermeture propre d'une
  marche sequentielle est trop dependante des composants non portes pour cadrer un paquet
  isole — seul le discriminant de masques PAR FAMILLE est fiable.
- **Balayage avec inference de chaine** (`TestLot1InferenceParFamille`) : NON CONCLUANT par
  DEFAUT DE L'INSTRUMENT, publie tel quel — la metrique « fin de chaine propre » est gagnee
  a 100 % par des trames vides (k=2/3 : 0 record, payload non couvert), faute d'exiger la
  couverture ; et l'inference lit au-dela de la fin du payload (records fantomes). A
  re-outiller avant toute conclusion par cette voie.

### Cote exe (Ghidra, meme session)

- La boucle de records `FUN_1406cd128` porte **DEUX grammaires d'identifiant**, selectionnees
  par le global `DAT_14474cd78` : branche A = var-int domaine 7 (celle du port historique),
  branche B = largeur lue dans une table selectionnee par **le bit de configuration du paquet**
  (`DAT_144706104`, bit 0 du payload — mesure a 1 sur 100 % du corpus) : cardinal
  `DAT_1451f990c` + base `DAT_1451f9908` si bit=1, `DAT_144706100` sans base sinon. C'est la
  source RUNTIME du `IDLowBits` calibre par film (11 sur 000d5950, 14 sur la capture live).
- `FUN_1428e24bc` (chemin de restauration d'etat : il resNapshotte les entites puis rejoue une
  boucle de records) **force temporairement `DAT_14474cd78 = 0`** puis le restaure — la
  lecture NORMALE tourne donc en branche B.
- Le deuxieme bit d'amorce reste NON LOCALISE statiquement (le frame-processor ne lit qu'un
  bit ; les blocs « unreachable » du decompile ont ete desassembles : nettoyage seulement).
  L'etabli reste empirique, comme le documentait deja `frame_records.go`.

## Decouvertes hors perimetre (non traitees, regle 7)

- Le decompile Ghidra de FUN_1406cd128 revele DEUX grammaires de boucle de records,
  selectionnees par le global `DAT_14474cd78` ; dans la seconde, la largeur de l'identifiant
  de record est tiree d'une table selectionnee par LE BIT DE CONFIGURATION du paquet
  (`DAT_144706104`) — a instruire au geste (a) ci-dessus.
- `DecodeFrameViews`/inference lisent au-dela de la fin du payload sans garde (records
  fantomes, EndBit > taille) — un clamp serait a poser si l'instrument devient production.
