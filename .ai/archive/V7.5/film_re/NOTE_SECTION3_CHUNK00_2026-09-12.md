# La troisieme section de `chunk_00` : le film porte la table des joueurs du match

Date : 2026-09-12. Suite directe de `NOTE_CARTE_CHUNK00_2026-08-30.md`, dont elle ferme trois
questions ouvertes et corrige une prémisse du dépôt. Travail **hors ligne, lecture seule** :
décompilation statique de `HaloInfinite.exe` (Ghidra, instance partagée, aucun renommage ni
sauvegarde) + mesures sur les `chunk_00` du cache local. Aucun code de production modifié.

Instruments (tous sous garde d'environnement, sautés en CI) :

| fichier | ce qu'il mesure |
|---|---|
| `apps/go-api/internal/analysis/filmdec/section3_ecrivain_research_test.go` | la carte de `chunk_00` relue chez l'écrivain : offsets et largeurs de bits, fermeture arithmétique, décalage d'un bit, horodatage du match confronté à `match_registry` |
| `apps/go-api/internal/analysis/filmdec/section3_roster_research_test.go` | la table des 32 slots : recherche bit à bit des XUID, plancher de faux positifs mesuré, découverte du roster sans entrée externe, contrôle croisé inter-films |

---

## Résumé exécutif

1. **La section 3 n'est pas un format inconnu : c'est le même flux bit-packé que tout le reste du
   film, et sa grammaire est écrite noir sur blanc dans l'exécutable.** Deux fonctions sérialisent
   la totalité de `chunk_00` — `FUN_14299b198` pour la tête, `FUN_14299b278` pour la suite et le
   corps — et leur désassemblage donne, champ par champ, la source et **la largeur en bits**.
2. **La voie qui a marché est l'ancre indiquée par le lot D : les chaînes de build.** `HI_1_13_0`
   (`0x1436a37c0`) et `release` (`0x143690658`) mènent en deux sauts à `FUN_14299b674`, qui remplit
   le tampon de `chunk_00`. Les statistiques sur les octets ne pouvaient rien donner ; la structure
   n'était pas cachée, elle était **décalée d'un bit**.
3. **LE POINT DUR EST LÀ : après un booléen d'UN BIT à `0x0CB45C`, tout le reste du fichier est
   décalé d'un bit.** Aucune lecture alignée sur l'octet ne pouvait rendre quoi que ce soit après
   cet offset. C'est l'explication mécanique de « entropie 7,25 bits/o, aucun pas de structure,
   aucune valeur trouvée en clair » du 30/08.
4. **La section 3 se termine par une table de 32 enregistrements de `0x1450` octets, un par SLOT du
   match, et chaque enregistrement porte le XUID du joueur en clair sur 64 bits, à 85 bits de son
   début.** Mesure : **44 des 45 XUID humains de `match_participants` retrouvés exactement une fois
   sur 6 films de 2 builds, 0 faux positif sur 240 leurres de même forme.**
   *Contrôle d'échelle, sur 250 films* : le compte d'enregistrements découverts vaut **8 sur
   198 films**, **24 sur 2 films** (la taille exacte d'une « Grande bataille »), et **ne dépasse
   jamais 32** — la borne que l'écrivain impose. Le balayage ignore tout des modes de jeu ; il
   retombe seul sur les tailles d'escouade réelles.
5. **Le film permet de reconstruire le roster SANS aucune entrée externe** (oracle interne) : le
   seul motif d'en-tête `1/0/0 + u32 nul + 2 bits nuls` suivi d'un entier de 64 bits dans la plage
   des XUID Xbox rend 44 enregistrements sur 6 films, contre 1 à 2 positions parasites par film,
   toutes éliminées par deux filtres écrits d'avance. **Le pont slot → xuid, aujourd'hui reconstruit
   indirectement par le fil des morts, est disponible en lecture directe.**
6. **L'horodatage du match est dans le film.** `_time64()` est écrit à `base+0xcb790`, soit, une
   fois le décalage d'un bit appliqué, 32 bits à `0x0CB65C × 8 + 1`. Confronté à
   `match_registry.start_time_utc` : **+19 s, +29 s, +43 s** sur les trois films témoins (seuil
   écrit avant la mesure : 120 s). Contrôle négatif : **un seul décalage de bit sur dix-sept** rend
   une valeur plausible, **sur 6/6 films**, et c'est celui que l'écrivain prédit.
7. **Question ouverte n°2 du 30/08 : FERMÉE.** La table par type commence bien à `0x0CB208` et
   compte **exactement 123 entrées** (`0xF60` bits écrits depuis `base+0xCB208`). Fermeture
   arithmétique : `0x0CB208 + 123 × 4 = 0x0CB3F4` = l'offset du champ version, mesuré par une tout
   autre voie le 30/08. Aucun ajustement.
8. **Question ouverte n°4 du 30/08 : FERMÉE.** Les deux u32 de `0x0CB454` sont l'**identifiant de
   build** et le **changelist**, lus dans la structure d'informations de build de l'exécutable
   (`FUN_140b32390` → `FUN_140b32438`, globales `DAT_144e4ef50` / `DAT_144e4ef54`, la seconde
   formatée par `" changelist: %d"`).
9. **Question ouverte n°3 du 30/08 (sémantique des valeurs 1..6) : APPUYÉE, PAS PROUVÉE.** Les 123
   u32 sont produits par un **appel virtuel `vtable+0x30`** sur chacun des objets enregistrés dans
   le tableau `DAT_144e61d88+0x210`. Une valeur courte obtenue par une méthode virtuelle du
   descripteur de type est cohérente avec « version de sérialisation par type » ; la fonction qui
   la CONSOMME au rejeu n'a pas été lue, la lecture reste une hypothèse.
10. **Une prémisse du dépôt est fausse : le registre ECS commence à l'octet 8, pas à 0.** Découverte
    hors périmètre, **non traitée** (règle 7) — voir plus bas.

---

## A. La chaîne de preuve, et pourquoi elle compte double

Deux voies **sans étape commune** mènent à la même grammaire (méthode, règle 2).

**Voie 1 — l'exécutable.** Aucune mesure sur les films n'y intervient.

```
chaîne "HI_1_13_0" @0x1436a37c0
  -> xref  FUN_140b32390           construit la structure d'infos de build
  -> appel FUN_140b32438           la recopie dans DAT_144e4ef30..90
  -> xref lecteur commun de DAT_144e4ef88 (build) ET DAT_144e4ef54 (changelist)
  -> FUN_14299b674                 initialiseur de l'écrivain de film
       FUN_14051c0f4(base+0xcb524, DAT_144e4ef38, 0x20)   version
       FUN_14051c0f4(base+0xcb544, DAT_144e4ef88, 0x20)   build
       FUN_14051c0f4(base+0xcb564, DAT_144e4ef78, 0x20)   saveur
       *(u32*)(base+0xcb584) = DAT_144e4ef50              id de build
       *(u32*)(base+0xcb588) = DAT_144e4ef54              changelist
       boucle registre : 50 x 0x4100 octets  (0xcb200 / 0x4100 = 50)
       boucle table    : 123 x u32           (0x1ec / 4  = 123)
       *(u32*)(base+0xcb790) = _time64()                  horodatage
  -> FUN_14299cb5c                 tampon de 0x1E1B80 = 1 973 120 octets
  -> FUN_14299b198 + FUN_14299b278 sérialisation, largeurs en bits
  -> FUN_1407ec560                 le corps
  -> FUN_1407ecb08                 un enregistrement de slot
```

**Voie 2 — les octets des films.** Aucune adresse de l'exécutable n'y intervient : les XUID
viennent de `match_participants`, les dates de `match_registry`, et le reste est mesuré sur le
tampon inflaté.

Le test d'indépendance de la méthode : si la voie 1 est fausse (mauvaise fonction), la voie 2 ne
tombe pas — elle rendrait simplement zéro XUID et zéro date plausible. Si la voie 2 est fausse
(mauvais roster), la voie 1 tient quand même. Elles ne partagent aucune étape.

---

## B. La carte complète de `chunk_00`, lue chez l'écrivain

`FUN_14299b198(base, writer)` puis `FUN_14299b278(base, writer)`, désassemblés. Le motif est
invariable : `LEA R8,[RBX + offset_source] ; MOV R9D, nombre_de_bits ; CALL FUN_1406d60f4`.
`FUN_1406d60f4(writer, _, src, nbits)` est l'écrivain de bits générique : MSB-first, byte-swap
64 bits, curseur `*(writer+0x2c) += nbits` — **la convention de curseur déjà connue du dossier**.

Les offsets source sont strictement contigus, ce qui ferme la lecture sans ajustement :

| # | source (base-relative) | largeur écrite | = octets | offset dans le FLUX | contenu |
|---|---|---|---|---|---|
| 1 | `0x000000` | `0x20` bits | 4 | `0x000000` | u32 A — **mesuré 41 (`0x29`) sur les 3 films `HI_1_13_0`, 40 (`0x28`) sur les 3 films `HI_1_12_0`** : il SUIT LE BUILD (cf. B.2) |
| 2 | `0x000004` | `0x20` bits | 4 | `0x000004` | u32 B — **mesuré 27 sur les 6 films, les deux builds** |
| 3 | `0x000008` | `0x659000` bits | 832 000 | `0x000008` | **LE REGISTRE ECS**, 50 blocs de `0x4100` |
| 4 | `0x0CB208` | `0xF60` bits | 492 | `0x0CB208` | **LA TABLE PAR TYPE : 123 u32** |
| 5 | `0x0CB3F4` | `0x100` bits | 32 | `0x0CB3F4` | version (`6.10026.18411.0`) |
| 6 | `0x0CB414` | `0x100` bits | 32 | `0x0CB414` | build (`HI_1_13_0`) |
| 7 | `0x0CB434` | `0x100` bits | 32 | `0x0CB434` | saveur (`release`) |
| 8 | `0x0CB454` | `0x20` bits | 4 | `0x0CB454` | **identifiant de build** (`0x0004187B`) |
| 9 | `0x0CB458` | `0x20` bits | 4 | `0x0CB458` | **changelist** (`0x0086ED94`) |
| 10 | `0x0CB45C` | **1 bit** (`FUN_1406d49c4`) | — | `0x0CB45C` bit 0 | booléen — **mesuré 0 sur les 6 films** |
| 11 | `0x0CB460` | `0x800` bits | 256 | `0x0CB45C` bit 1 | champ de nom (128 car. UTF-16) — **0 octet non nul sur les 6 films** |
| 12 | `0x0CB560` | `0x800` bits | 256 | `0x0CB55C` bit 1 | second champ de nom — **0 octet non nul sur les 6 films** |
| 13 | `0x0CB660` | `0x20` bits | 4 | `0x0CB65C` bit 1 | **HORODATAGE DU MATCH** (`_time64()`) |
| 14-16 | `0x0CB664/68/6C` | `0x20` bits chacun | 12 | `0x0CB660/64/68` bit 1 | **mesurés nuls sur les 6 films** |
| 17-19 | `0x0CB670`, `0x0CC670`, `0x0CD670` | `0x8000` bits chacun | 3 × 4 096 | `0x0CB66C/0x0CC66C/0x0CD66C` bit 1 | **mesurés à 0 / 0 / 1 octet non nul** |
| 20-21 | `0x0CE670`, `0x0CE680` | `0x80` bits chacun | 2 × 16 | `0x0CE66C`, `0x0CE67C` bit 1 | 32 octets à forte entropie, propres au film (remplis dans `FUN_14299b674` depuis un objet dont `+0xd8 == 2`) |
| 22 | `0x0CE690` | `FUN_1407ec560` | le reste | **`0x0CE68C` bit 1** | **LE CORPS — la « section 3 » dense** |

**Le décalage.** Le champ 10 consomme 1 bit là où il occupe 4 octets dans la structure. La règle de
conversion est donc, pour tout champ situé après lui :

```
bit_dans_le_flux(offset_source X) = (X - 4) * 8 + 1
```

C'est la seule raison pour laquelle la section 3 paraissait amorphe. Vérification numérique :
`0x0CB45C + 512 (deux noms) + 16 (quatre u32) + 12 288 (trois blocs) + 32 (deux blocs de 16)
= 0x0CE68C`, et le 30/08 mesurait « le premier bloc de 64 octets tous non nuls » à `0x0CE698` /
`0x0CE6A8` — juste après, parce que les premiers octets du corps contiennent encore des zéros
(`27 00 2e b2 81 f5 6c 4e 40 66 00 00 …`). Les « ~12 364 octets avant la donnée dense » du 30/08
sont donc **12 848 octets de champs mesurés, dont 12 800 vides**, plus le bruit de détection.

**Pourquoi le premier octet qui diffère entre deux films tombait à `0x0CB65C` et pas avant** : les
champs 11 et 12 (512 octets de nom) sont **vides sur tous les films mesurés** ; le premier champ
propre au match est l'horodatage, qui commence au bit 1 de l'octet `0x0CB65C`. La mesure du 30/08
était juste, sa lecture (« début de la section 3 ») est à préciser : la section commence
structurellement au bit 1 de `0x0CB45C`.

### B.1 La fermeture arithmétique (contrôle W-POS, écrit avant la mesure)

Trois mesures obtenues séparément doivent coïncider par construction :

```
0x000008 + 0x659000/8  = 0x000008 + 832 000 = 0x0CB208   = début de la table par type
0x0CB208 + 0xF60/8     = 0x0CB208 + 492     = 0x0CB3F4   = offset du champ version
```

Elles se ferment **sans aucun ajustement**, et l'offset `0x0CB3F4` du champ version avait été
mesuré le 30/08 par le seul examen des octets. Le cardinal **123** n'est donc plus une coïncidence
avec la borne du dispatcher : il est écrit par l'écrivain (`MOV R9D,0xf60`) et confirmé par la
boucle de `FUN_14299b674` (`while (j*4 < 0x1ec)`).

### B.2 Le u32 de tête suit le build — et le dépôt l'avait déjà mesuré sans le savoir

Mesure : u32 A vaut **41** sur les trois films `HI_1_13_0` et **40** sur les trois films
`HI_1_12_0`. Or le commentaire de `looksZlib` (`registry.go`) consigne un balayage des 1 378
`chunk_00` du cache : le premier octet du tampon vaut « `0x29` sur 1 117 films, mais aussi `0x28`
(204 films), `0x27` (34), `0x25` (13), `0x26`/`0x1f`/`0x21` (3 chacun), `0x22` (1) ». **C'est ce
champ**, et il croît avec la version du jeu exactement comme le cardinal de la table par type
(119 → 121 → 122 → 123). Le dépôt le lisait comme « le `kind` u32 du premier slot du registre » ;
c'est en réalité un champ d'en-tête à part entière, antérieur au registre. Sa sémantique exacte
(version de format ? nombre d'archétypes ?) n'est pas établie : le compte de blocs porteurs mesuré
le 30/08 est **49**, pas 41.

### B.3 Un contrôle interne gratuit : les horodatages sont ordonnés par build

Les six horodatages lus (section E.6) tombent dans l'ordre du build, sans qu'on l'ait demandé :
`HI_1_12_0` → 2025-10-22, 2025-10-23, 2025-11-09 ; `HI_1_13_0` → 2026-02-17, 2026-03-01,
2026-03-08. Une lecture fausse ne produirait pas six dates plausibles **ni** leur ordre correct
par rapport à un champ indépendant (la chaîne de build).

---

## C. Le corps : ce qu'est la section 3

`FUN_1407ec560(writer, jeu)` sérialise une structure unique, recopiée au préalable depuis l'objet
moteur du match par `FUN_14095944c` (constructeur de copie, `0x1134F0` octets). Champ par champ,
au désassemblage :

| source (jeu-relative) | largeur | note |
|---|---|---|
| `+0x00`, `+0x04` | 3 bits, 3 bits | |
| `+0x08` | 2 bits | |
| `+0x0C` (short) | 7 bits | |
| `+0x10` | 64 bits | |
| `+0x18` | 32 bits | |
| `+0x20` (octet) | 3 bits | |
| `+0xE9510`, `+0xE9514`, `+0xE9518` | 32 bits chacun | |
| `+0xE951C` | `FUN_140b857d8` | sous-sérialiseur — **non décodé** |
| `+0xEA694` | chaîne ASCII, max 128 car. (`FUN_1407ebe7c`) | |
| `+0xEA714`, `+0xEA718`, `+0xEA71C`, `+0xEA71D`, `+0xEA724`, `+0xEA728` | 32 / 32 / 1 / 1 / 2 / 1 bits | |
| `+0xEA72C` | `FUN_1410bc140` | sous-sérialiseur — **non décodé** |
| `jeu+0x28` | booléen puis, si vrai, `FUN_140b85504` | **objet optionnel** — candidat variante de mode |
| `+0xEA810` | **chaîne ASCII, max 256 car.** | recopiée en UTF-16 dans le champ 11 de l'en-tête |
| `+0xEA910` | **chaîne ASCII, max 256 car.** | |
| `+0xEAA10` | bits (largeur en registre) | |
| `+0xEAA18` | `0x6C0` bits = 216 octets | |
| `+0xEAAF0` .. `+0x113AF0` | **boucle : 32 × `FUN_1407ecb08`, pas `0x1450`** | **LA TABLE DES SLOTS** |

Fermeture : `0x28A00 / 0x1450 = 32` exactement, et `0xEAAF0 + 0x28A00 = 0x113AF0`, la borne de fin
littérale du désassemblage (`LEA RSI,[RDI + 0x28a00]`).

---

## D. La table des 32 slots — le résultat exploitable

### D.1 La grammaire d'un enregistrement (`FUN_1407ecb08`, désassemblage)

| source (slot-relative) | largeur | écrivain |
|---|---|---|
| `+0x00` | **1 bit** | `FUN_1406d49c4` (booléen) |
| `+0x01` | **1 bit** | booléen |
| `+0x02` | **1 bit** | booléen |
| `+0x04` | **32 bits** | inline |
| `+0x08` (octet signé) | **2 bits** | inline |
| `+0x09` | **48 bits** | `FUN_1406d60f4(…, 0x30)` |
| `+0x10` | **64 bits** | `FUN_1406d6498(…, 0x40)` — **le XUID** |
| `+0x18` … `+0x1448` | variable | `FUN_1407edea8` — sous-enregistrement, **partiellement lu** |
| `+0x1448` | 32 bits | inline |

Soit **85 bits d'en-tête (1+1+1+32+2+48) avant l'entier de 64 bits**.

Le sous-enregistrement (`FUN_1407edea8`, base `sub = slot+0x18`) a été ouvert d'un cran, assez pour
expliquer sa longueur variable :

| source (sub-relative) | largeur | écrivain |
|---|---|---|
| `+0x000` .. `+0x100` | **masque de présence de 2 048 bits, préfixé par sa longueur** | `FUN_1407ecd78` : cherche le rang du bit le plus haut, écrit le compte, puis écrit ce nombre de bits un par un (`FUN_1406d49c4`) |
| `+0x100` puis `+0x108` | u32 de longueur N, puis **N octets** | `FUN_1411b1a24` puis `FUN_1406d60f4(…, N << 3)` |
| `+0x908` puis `+0x910` | u32 de longueur M, puis **M × 4 octets** | `FUN_1411b198c` puis `FUN_1406d60f4(…, M << 5)` |
| `+0xC48` | `0x340` bits = 104 octets | `FUN_1406d60f4` |
| `+0xC14` | `0x10` bits | `FUN_1407ece18` |
| `+0xC38` | `0x80` bits = 16 octets | `FUN_1406d60f4` |
| `+0xCB0` | — | `FUN_1407edaf4(writer, DAT_143686818, valeur)` — table de correspondance ? |
| `+0xCB8` | `0x40` bits | `FUN_1406d60f4` |
| `+0xC12`, `+0xC36`, `+0xC35`, `+0xC10` | variables | `FUN_1407edcc4`, `FUN_1407edd3c`, `FUN_1407eddb4`, … |

**Trois listes préfixées par leur longueur** : c'est la source mécanique des deux classes de
longueur mesurées en D.5. Aucun de ces champs n'a la forme d'un nom de joueur ; le gamertag est
plus loin dans le sous-enregistrement, **non localisé par ce lot**.

### D.2 R-POS et R-NEG — les XUID du registre sont là, et le plancher de bruit est nul

Recherche bit à bit (MSB-first), à tout décalage, depuis le début du corps jusqu'au dernier octet
non nul. Le contrôle négatif utilise **40 leurres par film tirés au hasard dans la même plage
d'XUID Xbox** (`0x0009000000000000`..`0x000A000000000000`) : même forme, même longueur, même
espace de recherche — c'est le plancher **mesuré**, pas calculé (méthode, règle 4).

| film | build | XUID humains au registre | trouvés exactement 1 fois | en-tête conforme | leurres → touches |
|---|---|---|---|---|---|
| `000d5950` | `HI_1_13_0` | 8 | **8** | 8/8 | 40 → **0** |
| `00162144` | `HI_1_13_0` | 9 | **8** | 8/8 | 40 → **0** |
| `00502e52` | `HI_1_13_0` | 8 | **8** | 8/8 | 40 → **0** |
| `0014603f` | `HI_1_12_0` | 8 | **8** | 8/8 | 40 → **0** |
| `007d53a4` | `HI_1_12_0` | 4 | **4** | 4/4 | 40 → **0** |
| `02784ce1` | `HI_1_12_0` | 8 | **8** | 8/8 | 40 → **0** |
| **total** | | **45** | **44** | **44/44** | **240 → 0** |

« En-tête conforme » = les 85 bits qui précèdent le XUID valent `1/0/0` (trois booléens), puis un
u32 **nul**, puis un champ de 2 bits **nul**. Ces cinq champs prennent la même valeur sur les
**44** enregistrements des **6** films : une position de bit arbitraire ne produirait pas cela.

**Le seul manque, et il est nommé** : `00162144`, XUID `2533274799249008` (équipe 0, rang 3),
0 touche. Ce match est aussi le seul du lot à porter un bot (`bid(16.0)`). Hypothèse non testée :
le joueur a rejoint après l'écriture de `chunk_00` (le film est écrit au démarrage du match, comme
l'horodatage le montre), ou son slot est occupé par le bot. **Non tranché.**

### D.3 R-INT — reconstruire le roster sans aucune entrée externe

Oracle interne (méthode, règle 3) : le balayage cherche le motif d'en-tête `1/0/0 + u32 nul +
2 bits nuls` suivi d'un entier de 64 bits dans la plage des XUID Xbox. Deux filtres écrits avant la
mesure éliminent les formes dégénérées (XUID égal à la borne basse de la plage, jeton de 48 bits
nul) et un regroupement de fin de flux (écart consécutif ≤ 40 000 bits, soit 1,4 fois
l'enregistrement le plus long mesuré) isole la grappe terminale.

| film | positions conformes brutes | après filtres | roster attendu |
|---|---|---|---|
| `000d5950` | 9 | **8** | 8 |
| `00162144` | 9 | **8** | 9 (cf. D.2) |
| `00502e52` | 9 | **8** | 8 |
| `0014603f` | 10 | **8** | 8 |
| `007d53a4` | 5 | **4** | 4 |
| `02784ce1` | 9 | **8** | 8 |

Le bruit est de **1 à 2 positions par film**, toutes hors de la grappe terminale ou dégénérées.
`chunk_00` suffit donc à reconstruire la liste des joueurs et leur ordre de slot.

### D.4 R-CRO — la longueur d'un enregistrement suit le JOUEUR, pas le match

Contrôle croisé, entièrement interne : si le découpage est bon, l'enregistrement d'un joueur doit
avoir la même longueur dans deux matchs différents (son nom et sa personnalisation ne changent pas
d'un match à l'autre). Un découpage faux ne produirait pas cette égalité.

| XUID | longueur | films |
|---|---|---|
| `2533274823110022` | **27 969 bits** | `000d5950`, `00162144`, `0014603f`, `02784ce1` |
| `2533274858283686` | **28 081 bits** | `00162144`, `0014603f`, `007d53a4`, `02784ce1` |

**2/2 joueurs vus dans au moins deux films ont une longueur d'enregistrement CONSTANTE**, à travers
deux builds différents. C'est le contrôle le plus fort du lot : il ne dépend d'aucune source
externe, et il vaut simultanément pour la position du XUID et pour la frontière des
enregistrements.

### D.4 bis Contrôle d'échelle : 250 films, et le compte tombe sur les tailles d'escouade réelles

Le balayage interne (D.3) passé sur **250 films du cache** (31 `HI_1_12_0`, 219 `HI_1_13_0`) :

| enregistrements découverts | films |
|---|---|
| 8 | **198** |
| 7 | 15 |
| 9 | 14 |
| 6 | 4 |
| 4 | 5 |
| 3 | 3 |
| 1 | 6 |
| 2 · 5 · 10 | 1 chacun |
| **24** | **2** |

Trois faits s'y lisent, et aucun n'a été cherché :

1. **Le mode dominant est 8** — la taille d'une partie d'arène Halo Infinite.
2. **Deux films rendent exactement 24 enregistrements** — la taille exacte d'une partie
   « Grande bataille ». Le balayage n'a aucune notion de mode de jeu : il retombe dessus seul.
3. **Aucun film ne dépasse 32**, la borne que l'écrivain impose (`0x28A00 / 0x1450 = 32`).

Les comptes faibles (1 à 6) sont attendus : les bots n'ont pas de XUID (`bid(…)` dans
`match_participants`) et certains films du cache sont incomplets. **Aucun film ne rend un compte
impossible.**

### D.5 Ce que les longueurs disent, et ce qu'elles ne disent pas

Les longueurs mesurées se répartissent en **deux classes nettes** : **16 611 à 16 739 bits**
(~2 080 octets) et **27 603 à 28 145 bits** (~3 500 octets). L'écart entre classes est d'environ
11 400 bits. Les deux classes portent des XUID valides, donc « courte » ne veut pas dire « slot
vide ». La classe n'est pas positionnelle : sur `00502e52`, courtes et longues alternent.
**Hypothèse non testée** : un bloc optionnel de personnalisation présent ou absent.

**L'ordre des enregistrements n'est pas l'ordre des équipes.** Sur `000d5950`, les équipes dans
l'ordre des slots donnent `0,1,0,0,1,1,1,0`. C'est un ordre de session (ordre de connexion ou index
de slot moteur), pas un regroupement par équipe. **Le champ d'équipe n'a pas été localisé** : il est
dans le sous-enregistrement `FUN_1407edea8`, non décodé.

**Le champ de 48 bits (`slot+0x09`) est propre au couple (match, joueur).** Pour le XUID
`2533274823110022`, il vaut `1bdc8d4edce1`, `c8ea301b67e7`, `20f2869e9b23`, `e5e6c6eb49c2`,
`8d2f53d00e52` sur cinq films : cinq valeurs différentes, à forte entropie. Lecture la plus
économique : un jeton de session de joueur. **Non prouvé.**

---

## E. Ce qui est ÉTABLI, ce qui est HYPOTHÈSE, ce qui est RÉFUTÉ

### Établi (deux chaînes indépendantes)

1. `chunk_00` est écrit par `FUN_14299b198` + `FUN_14299b278` dans un tampon de `0x1E1B80` =
   1 973 120 octets — chiffre identique à la taille inflatée mesurée le 30/08 par une tout autre
   voie.
2. Le registre ECS occupe `0x000008`..`0x0CB208` et compte **50 blocs** de `0x4100` octets.
3. La table par type commence à `0x0CB208` et compte **exactement 123 u32**. Fermeture arithmétique
   avec l'offset du champ version.
4. Les deux u32 de `0x0CB454`/`0x0CB458` sont l'identifiant de build et le changelist.
5. Un booléen d'UN BIT à `0x0CB45C` décale tout le reste du fichier d'un bit.
6. Les 32 bits à `0x0CB65C × 8 + 1` sont l'horodatage Unix du match (+19 s / +29 s / +43 s de
   `match_registry.start_time_utc` sur trois films ; un seul décalage de bit sur dix-sept rend une
   valeur plausible, sur 6/6 films, et c'est celui que l'écrivain prédit). Dates lues :
   `000d5950` 2026-03-08 20:44:14Z · `00162144` 2026-03-01 21:37:25Z · `00502e52`
   2026-02-17 20:34:09Z · `0014603f` 2025-10-23 19:48:37Z · `007d53a4` 2025-10-22 21:45:49Z ·
   `02784ce1` 2025-11-09 20:35:00Z.
7. Le corps commence au bit 1 de `0x0CE68C` et se termine par **32 enregistrements de `0x1450`
   octets**, un par slot.
8. Chaque enregistrement de slot porte un entier de **64 bits à 85 bits de son début**, et cet
   entier est **le XUID du joueur** : 44/45 sur 6 films et 2 builds, 44/44 en-têtes conformes,
   0 faux positif sur 240 leurres de même forme, longueur d'enregistrement constante par joueur à
   travers les matchs (2/2).

### Hypothèse (une seule chaîne)

1. Les valeurs 1..6 de la table par type sont une **version de sérialisation par type**. Nouvel
   appui : elles sortent d'un appel virtuel `vtable+0x30` sur le descripteur de chaque type
   (tableau `DAT_144e61d88+0x210`). Ce qui manque toujours : la fonction qui les consomme au rejeu.
2. Les deux chaînes ASCII de 256 caractères (`jeu+0xEA810`, `jeu+0xEA910`) sont des noms de
   variante / de carte. Mesuré : le champ d'en-tête qui en recopie une en UTF-16 est **vide sur les
   6 films**, donc la valeur elle-même n'a pas été observée.
3. Le champ de 48 bits de chaque slot est un jeton de session de joueur.
4. Les deux classes de longueur d'enregistrement correspondent à la présence ou à l'absence d'un
   bloc de personnalisation.
5. L'objet optionnel de `jeu+0x28` (booléen + `FUN_140b85504`) est la variante de mode.
6. Les u32 A et B de tête. **Établi** : A suit le build (41 en `HI_1_13_0`, 40 en `HI_1_12_0`, et
   le balayage du corpus consigné dans `looksZlib` montre `0x29`/`0x28`/`0x27`/`0x25`… par
   ancienneté) ; B vaut 27 sur les deux builds. **Hypothèse** : A est une version de format du
   registre, B le nombre de composants de l'archétype 0 (le 30/08 mesurait 27 slots au bloc 0).
   Ni l'un ni l'autre n'est prouvé — et A n'est PAS le nombre de blocs porteurs, qui vaut 49.

### Réfuté

1. **« Le registre ECS commence à l'octet 0 du tampon inflaté. »** FAUX : il commence à l'octet 8.
   `parseRegistry` lit depuis 0, donc son champ « kind » du slot *i* est le champ situé 8 octets
   avant le nom du slot *i*. **Cela explique d'un coup deux observations du dépôt** : « kind = 0 sur
   1 066 des 1 067 slots » (30/08, D1b) et « le niveau lu un cran plus loin » du commentaire de
   `registryBlockTail`. Découverte hors périmètre, **non traitée** — voir section G.
2. **« Le décalage d'indexation de la table par type n'est pas déterminable. »** RÉFUTÉ par
   l'écrivain : 123 entrées à `0x0CB208`.
3. **« Les deux u32 de `0x0CB454` n'ont pas de rôle établi. »** RÉFUTÉ.
4. **« Aucun pas de structure ne ressort de la section 3 (2..1024). »** La mesure était juste, la
   conclusion implicite (« pas d'enregistrement de taille fixe ») est trompeuse : il y a bien une
   table d'enregistrements de taille fixe **en mémoire** (`0x1450` octets), mais le flux est
   bit-packé et les enregistrements y ont une **longueur variable**. Aucune périodicité alignée sur
   l'octet ne pouvait apparaître.
5. **« La section 3 est un flux dont on ne sait rien lire. »** RÉFUTÉ : le roster s'en lit
   directement.

---

## F. Commandes pour rejouer

Les instruments sont sous garde d'environnement ; sans la variable ils sont sautés, donc la CI
reste verte. `CHUNK00_FILMS` est une liste de répertoires de film séparés par `;`, **en chemins
Windows** (`C:/...`) : un chemin de style Git Bash (`/c/...`) fait échouer l'ouverture.

```bash
cd apps/go-api
C="C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks"

# B / B.1 : la carte relue chez l'écrivain, la fermeture arithmétique, le décalage d'un bit
CHUNK00_FILMS="$C/000d5950;$C/00162144;$C/00502e52" \
  go test ./internal/analysis/filmdec/ -run TestSection3CarteEcrivain -v

# Contrôle W-REF : l'horodatage du film contre match_registry.start_time_utc (epoch UTC)
CHUNK00_FILMS="$C/000d5950;$C/00162144;$C/00502e52" \
CHUNK00_DEBUTS="000d5950=1773002635;00162144=1772401016;00502e52=1771360406" \
  go test ./internal/analysis/filmdec/ -run TestSection3HorodatageContreRegistre -v

# D.3 / D.4 : le roster reconstruit SANS entrée externe, et la longueur par joueur
CHUNK00_FILMS="$C/000d5950;$C/00162144;$C/00502e52;$C/0014603f;$C/007d53a4;$C/02784ce1" \
  go test ./internal/analysis/filmdec/ \
  -run 'TestSection3RosterInterne|TestSection3LongueurParJoueur' -v -timeout 30m

# D.4 bis : contrôle d'échelle sur tout un répertoire de films (250 films en ≈9 s)
CHUNK00_CORPUS="$C" CHUNK00_CORPUS_MAX=250 \
  go test ./internal/analysis/filmdec/ -run TestSection3RosterCorpus -v -timeout 40m

# D.2 : R-POS / R-NEG, avec le roster de match_participants (≈21 s pour 6 films)
CHUNK00_FILMS="…" CHUNK00_ROSTERS="000d5950=2533274980284321,2535467794760703,…;…" \
  go test ./internal/analysis/filmdec/ -run TestSection3RosterParXuid -v -timeout 30m
```

Le roster et les débuts de match se relisent dans `shared_matches_v2.duckdb` (**lecture seule** ;
le fichier peut être tenu par le serveur, dans ce cas ne pas forcer) :

```sql
SELECT substr(match_id,1,8), string_agg(xuid, ',')
  FROM match_participants WHERE xuid NOT LIKE 'bid(%' GROUP BY 1;
SELECT substr(match_id,1,8), epoch(COALESCE(start_time_utc, start_time AT TIME ZONE 'UTC'))
  FROM match_registry;
```

Gates passés : `go vet ./internal/analysis/filmdec/` net, `go test ./internal/analysis/filmdec/`
sans garde **ok**, `go test ./internal/archlint/` **ok** — le ratchet des
variables de paquet de `filmdec` n'est pas touché (les instruments n'introduisent que des `const`
et des locales).

---

## G. Découvertes hors périmètre — notées, NON TRAITÉES (règle 7)

1. **`parseRegistry` lit le registre décalé de 8 octets.** L'écrivain place le registre à l'octet 8
   et le format de slot est donc `[nom ASCII][…][u32][u32]`, pas `[u32 kind][u32 flags][nom]`.
   Conséquences vérifiées sur pièces : le nom rendu est le bon (les deux modèles font commencer le
   nom au même endroit), mais `Archetype.Flags[i]` est le champ d'un **autre** slot — ce que le
   commentaire de `registryBlockTail` décrit déjà comme « le niveau lu un cran plus loin ». Il faut
   décider si c'est un défaut à corriger ou un décalage intentionnel compensé ailleurs
   (`quantAxisWidth`, `default_state*.go`) : **la question n'est pas tranchée par cette note** et
   toucher au registre sans la trancher casserait des décodages validés.
2. **`parseRegistry` divise toujours le fichier entier par la taille d'un bloc** (déjà signalé le
   30/08, point 5). L'écrivain donne la borne exacte : **50 blocs**, `0x000008`..`0x0CB208`.
3. **Le cache local ne contient plus que deux builds.** — **CORRIGE LE 2026-09-12 (phase 2) : CETTE
   AFFIRMATION EST FAUSSE.** `TestD1Builds` (garde `D1_BUILD_DIR`) classe les 1 351 films en
   **13 groupes portant 7 builds** : `HI_1_13_0` (1 123), `HI_1_12_0` (146), `HI_1_11_0` (39),
   `HI_1_10_0` (26), `HI_1_8_0` (10), `HI_1_9_0` (1), `HI_1_4_1` (1), plus 5 films sans section
   d'identification. La mesure ci-dessous ne comptait que les films dont l'en-tete se lit a
   `0x0CB414` — or cet offset DEPEND DU BUILD (il recule de 16 644 a 16 668 octets sur les builds
   anterieurs). Verification portee a sept builds :
   `NOTE_SECTION3_SLOTS_2026-09-12.md` section D. Le texte d'origine suit, tel quel : Mesure du 2026-09-12 sur
   `data/cache/film_chunks` : 1 123 films `HI_1_13_0` et 146 films `HI_1_12_0` sur les 1 351
   répertoires. Les builds `HI_1_5_1`, `HI_1_8_0`, `HI_1_9_0`, `HI_1_10_0`, `HI_1_11_0` que la note
   du 30/08 recensait **ne sont plus dans le cache**. La vérification demandée sur trois builds
   n'est donc pas possible en local ; elle a été faite sur **deux**.
4. **Le sous-enregistrement de slot (`FUN_1407edea8`) n'est pas décodé.** C'est lui qui porte
   l'équipe, le gamertag et la personnalisation, sur `0x1430` octets source.

---

## Ce qui reste ouvert

> **Mise a jour du 2026-09-12 (phase 2)** : les points 1 et 2 sont FERMES, et le point 3 est
> explique. Le gamertag est a `sub+0xc14` (616/636 contre l'oracle externe sur 76 films) ; un
> second champ de nom est le bloc brut `sub+0x1400` (640/640) ; la contradiction des deux positions
> du 30/08 est tranchee (18/18 touches expliquees, 0 hors grammaire) ; l'equipe n'est dans AUCUN
> champ court, fermeture par la negative ; l'ordre des enregistrements EST le `player_index`
> (`filmIndex - rang` constant sur 76/76 films) ; les deux classes de longueur viennent des trois
> listes prefixees, pas d'un bloc de personnalisation optionnel (celui-la est de largeur fixe et
> ecrit a zero). Voir `NOTE_SECTION3_SLOTS_2026-09-12.md`. Le texte d'origine suit, tel quel :

1. **L'équipe et le gamertag par slot.** Ils sont dans `FUN_1407edea8` (`slot+0x18`), dont D.1
   n'ouvre que le premier cran. Prochain geste : désassembler `FUN_1407edcc4`, `FUN_1407edd3c`,
   `FUN_1407eddb4`, `FUN_1407ede30` et `FUN_1407edaf4` (ce dernier reçoit un pointeur de données,
   `DAT_143686818`, ce qui sent la table de correspondance).

   **Un appui mesuré pour cette recherche, et il explique une bizarrerie du 30/08.** Le 30/08
   relevait « trois gamertags seulement par film, espacés de ~10 ko » là où les films ont 8 joueurs.
   Les positions de début d'enregistrement mesurées ici donnent l'explication : sur `000d5950`, les
   huit enregistrements commencent aux bits 10379571, 10396310, 10413049, 10441018, 10469112,
   10497074, 10525219, 10553316, soit des **résidus modulo 8 de 3, 6, 1, 2, 0, 2, 3, 4**. Un champ
   de nom à décalage CONSTANT dans l'enregistrement n'est donc lisible en UTF-16 aligné sur l'octet
   que dans le ou les deux enregistrements dont le résidu l'y amène — les six autres noms sont
   présents mais illisibles à l'octet. **Le test qui boucle la question** : une fois le décalage du
   champ de nom connu, les huit noms doivent sortir, et chacun dans l'enregistrement du bon XUID.
   Attention cependant : les deux positions du 30/08 sur `000d5950` tombent à 1 021 bits du début
   de l'enregistrement 0 et à 23 614 bits du début de l'enregistrement 3 — **deux décalages
   relatifs très différents**, donc soit il y a plusieurs champs de texte, soit l'une des deux
   lectures est fortuite. Cette contradiction est ouverte et doit être tranchée par une mesure, pas
   par un choix (méthode, erreur E).
2. **Le pont slot → index de joueur du film.** L'ordre des enregistrements donne un index de slot
   canonique par XUID. Reste à vérifier qu'il coïncide avec le `player_index` 5 bits utilisé en
   production (`resolvePlayerIndices`, reconstruit aujourd'hui par le fil des morts). Si oui, c'est
   un gain direct : une table explicite remplace une inférence.
3. **La longueur exacte d'un enregistrement de slot VIDE.** En supposant 32 enregistrements et en
   prenant le dernier octet non nul comme fin de flux, le résidu donne 15 700 à 17 900 bits par
   enregistrement vide selon le film — compatible avec la classe courte (16 611..16 739) mais pas
   déterminé. Le fermer demanderait de décoder `FUN_1407edea8`.
4. **Le joueur manquant de `00162144`.** Deux branches à départager par une mesure :
   il a rejoint après l'écriture de `chunk_00`, ou son slot est pris par le bot.
5. **Les valeurs 1..6 de la table par type** : lire la fonction qui les CONSOMME au rejeu.
   L'artefact `chunk00_table_par_type.tsv` est prêt ; le tableau source côté exe est
   `DAT_144e61d88+0x210` (descripteurs de type, méthode virtuelle `+0x30`).
6. **Les trois blocs de 4 096 octets et les deux blocs de 16.** Mesurés quasi vides pour les
   premiers, à forte entropie pour les seconds (matériel / session). Rôle non établi.
7. **Les deux chaînes de 256 caractères du corps** n'ont jamais été observées non vides. Un film de
   partie personnalisée (variante nommée) les porterait peut-être.
