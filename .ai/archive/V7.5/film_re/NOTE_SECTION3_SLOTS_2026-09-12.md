# L'enregistrement de slot de `chunk_00` : grammaire complete, gamertag, ordre, et le verdict sur la personnalisation

Date : 2026-09-12, phase 2. Suite directe de `NOTE_SECTION3_CHUNK00_2026-09-12.md` (phase 1), dont
elle ferme les questions ouvertes n°1 (equipe et gamertag), n°2 (pont slot vers index de joueur) et
corrige deux affirmations. Travail **hors ligne, lecture seule** : desassemblage statique de
`HaloInfinite.exe` (Ghidra, instance partagee, aucun renommage ni sauvegarde) + mesures sur les
`chunk_00` du cache local. Aucun code de production modifie, aucun commit.

Instruments (tous sous garde d'environnement, sautes en CI) :

| fichier | ce qu'il mesure |
|---|---|
| `apps/go-api/internal/analysis/filmdec/section3_slot_grammar_research_test.go` | la grammaire complete d'un enregistrement de slot, la fermeture de longueur, le gamertag, le plancher de bruit |
| `apps/go-api/internal/analysis/filmdec/section3_slot_ordre_research_test.go` | l'ordre des enregistrements contre le `player_index` de production, la recherche de l'equipe, la contradiction texte du 30/08, le bloc de queue |
| `apps/go-api/internal/analysis/filmdec/section3_slot_perso_research_test.go` | l'hypothese « bloc de personnalisation » : remplissage, stabilite par joueur, discrimination, prefixe commun, forme des identifiants |
| `apps/go-api/internal/analysis/filmdec/section3_slot_builds_research_test.go` | le profil par build sur les 1 351 films du cache (offset d'en-tete, decalage de longueur, survie du nom) |

---

## Resume execute

1. **VERDICT SUR LA PERSONNALISATION — le format la RESERVE, le film ne la PORTE PAS.** La
   structure existe et elle est nommee dans l'executable : `FUN_1407ec27c` ecrit
   `variantName`, `styleName`, `themeName`, `coatingName`, `markerName`, `regionOverrideName`,
   `actionPose`, `region`, `permutation`, `model_region`, `model_permutation`, sur la structure
   `sub+0xcc0` — la MEME que l'ecrivain de film recopie en brut sur `0x39e0` bits, avec
   **fermeture arithmetique exacte** (`0x738 + 4 = 0x73C = 1 852 octets`). Mais dans les films :
   **0 octet non nul sur 81 488** (44 enregistrements, 6 films, 2 builds). Le bloc est ecrit
   INTEGRALEMENT A ZERO. Et aucune autre zone de l'enregistrement ne porte de contenu stable par
   joueur : le prefixe commun d'une zone chez un meme joueur d'un match a l'autre vaut **2 a 16
   octets** sur 447, 702 et 668. **Verdict : hypothese REFUTEE pour le contenu, PROUVEE pour la
   structure.** Le Theater ne retrouve pas les tenues dans `chunk_00`.
2. **Le second volet de l'hypothese est REFUTE, et sa vraie cause est mesuree.** « Les deux
   classes de longueur sont avec / sans le bloc de personnalisation » : faux, ce bloc est de
   largeur FIXE et toujours ecrit. Les deux classes viennent des **trois listes prefixees par
   leur longueur** : `N ∈ {0, 702}` octets et `M ∈ {0, 167}` mots de 32 bits (mesure : 7 fois 0,
   37 fois 702/167 sur 44 enregistrements). Fermeture : `702 × 8 + 167 × 32 = 10 960` bits, et
   l'ecart mesure entre la plus courte longue (27 603) et la plus courte courte (16 611) vaut
   **10 992 = 10 960 + 32**, les 32 bits etant la difference de longueur des deux gamertags.
   Aucun ajustement.
3. **LA GRAMMAIRE EST COMPLETE ET ELLE FERME.** Quatorze champs lus au desassemblage de
   `FUN_1407edea8`, chacun verifie sur pieces (largeur lue dans le code de son ecrivain). La
   grammaire PREDIT la longueur totale d'un enregistrement a partir de quatre nombres lus dans le
   flux. Mesure : **38/38** sur les 6 films temoins, **560/567** sur 76 films (les 7 restants sont
   encadres par une position parasite du balayage, comptee a part depuis).
4. **LE GAMERTAG EST LU, ET IL Y A DEUX CHAMPS DE TEXTE.** Le champ `sub+0xc14` (`FUN_1407ece18`,
   unites de 16 bits MSB d'abord, terminees par NUL, 16 au plus) rend les 44 gamertags des
   6 films. Contrôle positif du 30/08 PASSE : `whiteknight2519` (`0x13CCA6`) et `LORD PEINX13`
   (`0x13F7AF`) tombent dans l'enregistrement du bon XUID. Le second champ est le bloc de queue
   `sub+0x1400` (44 octets = 22 unites, recopie BRUT donc petit-boutiste) : **640/640 blocs
   portent le gamertag de leur propre enregistrement**, sur 76 films.
5. **LA CONTRADICTION DU 30/08 EST TRANCHEE : deux champs de texte, aucune lecture fortuite.**
   Rejeu du balayage UTF-16LE aligne sur l'octet : **18 touches, 9 dans le champ de nom, 9 dans le
   bloc de queue, 0 hors grammaire.** Les positions relatives « tres differentes » du 30/08 (1 021
   et ~27 700 bits) etaient l'un et l'autre champ.
6. **L'EQUIPE N'EST PAS DANS LA PARTIE DECODEE DE L'ENREGISTREMENT.** Les neuf champs courts
   candidats sont **CONSTANTS sur les 44 enregistrements** : `b2=0`, `f10=183`, `f14=0`, `f6=-1`,
   `f8=0`, `f7=0`, `repr=0`, `u32Tete=0` ; seul `f1` prend deux valeurs, et ce sont exactement les
   9 enregistrements a listes vides contre 35 a listes pleines — c'est un drapeau de presence, pas
   une equipe. **Aucun champ ne partage un roster en deux moities egales, sur 0/6 films.** Question
   fermee par la negative.
7. **L'ORDRE DES ENREGISTREMENTS EST LE `player_index` DE PRODUCTION.** Sur 76 films portant un
   document de rejeu : **605/637 slots ont rang == filmIndex**, **72/76 films en coincidence
   totale**, et surtout **76/76 films ou `filmIndex - rang` est CONSTANT**. Les 4 films en
   desaccord sont donc une TRONCATURE DE TETE du lecteur (la grappe perd ses premiers
   enregistrements), pas une permutation. Une table explicite peut remplacer l'inference par le
   fil des morts.
8. **UNE PREMISSE DE LA PHASE 1 EST FAUSSE : le cache porte SEPT builds, pas deux.** `TestD1Builds`
   classe les 1 351 films en 13 groupes portant `HI_1_13_0`, `HI_1_12_0`, `HI_1_11_0`, `HI_1_10_0`,
   `HI_1_9_0`, `HI_1_8_0`, `HI_1_4_1`, plus 5 films sans section d'identification. La verification
   demandee sur trois builds a donc porte sur **sept**.
9. **LA GRAMMAIRE SE TRANSPOSE D'UN BUILD A L'AUTRE PAR UNE SEULE CONSTANTE**, et le XUID comme le
   nom survivent partout : **11 550 / 11 728 enregistrements** rendent un gamertag imprimable, tous
   builds confondus. Detail par build en section D.
10. **Le decalage d'en-tete par build se ferme arithmetiquement, trois fois.** L'offset de la chaine
    de build recule de **16 644** octets en `HI_1_11_0`, **16 648** en `HI_1_10_0`/`HI_1_9_0`/
    `HI_1_8_0`, **16 668** en `HI_1_4_1`. Or `16 640 = 0x4100` est exactement **un bloc de registre
    ECS**, et les restes valent `4 × 1`, `4 × 2`, `4 × 7` — soit 4 octets par entree manquante de
    la table par type, dont `TestD1Builds` mesure les cardinaux 123, 122, 117 contre 124. Les trois
    ecarts tombent sans ajustement.

---

## A. La grammaire d'un enregistrement de slot, champ par champ

`FUN_1407ecb08(slot, writer)` ecrit l'en-tete (phase 1, rappel) puis delegue a
`FUN_1407edea8(sub = slot+0x18, writer)`. Le desassemblage de cette derniere, releve ligne a ligne,
donne l'ordre d'ecriture ; la largeur de chaque champ est lue **dans le code de son ecrivain**, pas
supposee.

| # | source | largeur | ecrivain | verifie sur pieces |
|---|---|---|---|---|
| — | `slot+0x00/01/02` | 1+1+1 bits | `FUN_1406d49c4` | phase 1 |
| — | `slot+0x04` | 32 bits | inline | phase 1 |
| — | `slot+0x08` | 2 bits | inline | phase 1 |
| — | `slot+0x09` | 48 bits | `FUN_1406d60f4(…,0x30)` | phase 1 |
| — | `slot+0x10` | **64 bits = LE XUID** | `FUN_1406d6498(…,0x40)` | phase 1 |
| 1 | `sub+0x000` | **masque de presence**, prefixe de **11 bits** (rang du bit haut, moins 1) puis ce nombre de bits | `FUN_1407ecd78` → `FUN_1424ccf94` (`MOV ECX,0xb`) | oui |
| 2 | `sub+0x100` / `+0x108` | **12 bits** = N, puis **N octets** | `FUN_1411b1a24` (`MOV ECX,0xc`) puis `FUN_1406d60f4(…, N<<3)` | oui |
| 3 | `sub+0x908` / `+0x910` | **8 bits** = M, puis **M mots de 32 bits** | `FUN_1411b198c` (`0x8`) puis `FUN_1406d60f4(…, M<<5)` | oui |
| 4 | `sub+0xc48` | `0x340` bits = **104 octets** bruts | `FUN_1406d60f4` | oui |
| 5 | `sub+0xc14` | **chaine UTF-16, 16 unites au plus** : unites de 16 bits MSB d'abord, arret APRES l'unite nulle | `FUN_1407ece18` (`TEST DI,DI ; JNZ`) | oui |
| 6 | `sub+0xc38` | `0x80` bits = **16 octets** bruts | `FUN_1406d60f4` | oui |
| 7 | `sub+0xcb0` | **32 bits**, etiquete `desired-representation` (`DAT_143686818`) | `FUN_1407edaf4` (`MOV ECX,0x20`) | oui |
| 8 | `sub+0xcb8` | **64 bits** | `FUN_1406d60f4(…,0x40)` | oui |
| 9 | `sub+0xc12` | **10 bits** (ushort) | `FUN_1407edcc4` (`MOV ECX,0xa`) | oui |
| 10 | `sub+0xc36` | **14 bits** (ushort) | `FUN_1407edd3c` (`MOV ECX,0xe`) | oui |
| 11 | `sub+0xc35` | **6 bits, VALEUR + 1** (char signe : -1 representable) | `FUN_1407eddb4` (`MOVSX ; INC ; MOV ECX,0x6`) | oui |
| 12 | `sub+0xc10` | **8 bits** | inline | oui |
| 13 | `sub+0xc34` | **7 bits** (octet) | `FUN_1407ede30` (`MOV ECX,0x7`) | oui |
| 14 | `sub+0xc11 & 1` | **1 bit** | inline | oui |
| 15 | `sub+0xcc0` | `0x39e0` bits = **1 852 octets** bruts — **LA PERSONNALISATION** | `FUN_1406d60f4` | oui |
| 16 | `sub+0x1400` | `0x160` bits = **44 octets** bruts — **LE SECOND CHAMP DE NOM** | `FUN_1406d60f4` (tail call) | oui |
| — | `slot+0x1448` | 32 bits | inline | phase 1 |

### A.1 Les fermetures arithmetiques de la structure

```
0xc14 + 16 unites x 2 o = 0xc34   le champ qui suit la chaine commence pile a sa fin
0xcc0 + 1 852           = 0x13fc  le bloc de personnalisation s'arrete juste avant 0x1400
slot+0x18 + 0x1430      = 0x1448  l'offset du dernier u32 ; 0x1448 + 4 = 0x1450, le pas de la
                                  table. AUCUN TROU dans la structure.
```

### A.2 G-CLO : la grammaire predit la longueur, et la prediction est exacte

Oracle 100 % interne : la grammaire predit la longueur TOTALE d'un enregistrement a partir de
quatre nombres lus dans le flux (compte du masque, N, M, longueur du nom). Cette prediction doit
valoir EXACTEMENT l'ecart mesure jusqu'au debut de l'enregistrement suivant ; une seule largeur
fausse casse la prediction.

| corpus | ecarts mesurables | egaux a la prediction | positions parasites ecartees |
|---|---|---|---|
| 6 films temoins (2 builds) | 38 | **38** | 0 |
| 76 films a document de rejeu | 567 | **560** | — |

Les 7 non-fermetures sont encadrees par une position **parasite** du balayage : un motif d'en-tete
fortuit a l'interieur d'un vrai enregistrement, reconnaissable sans rien supposer de la grammaire
(son champ de nom ne rend pas de texte imprimable). L'instrument les compte desormais a part, et le
lecteur canonique (`s3sChaine`, qui avance du pas PREDIT) les enjambe par construction.

### A.3 G-REF et G-NEG : le gamertag, et le plancher de bruit

Reference externe : le `roster[].name` des documents de rejeu du depot, lui-meme issu de
`match_participants`.

| corpus | noms compares a l'oracle | egaux | decalages voisins essayes | touches |
|---|---|---|---|---|
| 6 films temoins | 8 (un seul film a un document de rejeu) | **8** | 1 408 | **44** |
| 76 films a document de rejeu | 636 | **616** | 20 576 | **639** |

Les deux gamertags du 30/08 tombent dans l'enregistrement du bon XUID. Les 20 ecarts des 76 films
sont des positions **parasites** du balayage (`TestSection3SlotGamertag` part du balayage brut, pas
du lecteur chaine) et les 4 films a troncature de tete.

Contrôle negatif **mesure** (methode, regle 4) : la meme lecture de chaine tentee aux
32 decalages de bit voisins (-16..+16, hors 0) rend un texte imprimable **44 fois sur 1 408**, soit
un seul decalage par enregistrement (44/44 exactement, et 639 pour 640 enregistrements sur les
76 films) — et c'est le decalage d'un octet, previsible : une suite d'unites `00 XX` relue un octet
plus loin rend `XX 00`, donc encore de l'ASCII. **Le bruit n'est pas diffus, il est structurel et
unique** : 3,1 % des decalages voisins, tous le meme.

### A.4 G-T44 : le bloc de queue est un second champ de nom, et son offset depend du build

L'offset du nom dans le bloc est **cherche**, pas suppose : le test balaie les 22 positions paires
et retient celle ou la relecture rend exactement le gamertag de l'enregistrement.

| build | enregistrements | offset du nom dans le bloc de 44 octets |
|---|---|---|
| `HI_1_13_0` | 632 | **0** (632/632) |
| `HI_1_12_0` | 8 | **12** (8/8) |
| **total** | **640** | **640/640 portent le gamertag** |

Le bloc precede ou suit le nom de trois u32 (`ffffffff`, `ff000000`, un petit entier) selon le
build. **Contrôle discriminant** : relu en gros-boutiste, le bloc rend **0/44** gamertags — un seul
ordre d'octets marche, donc la lecture n'est pas fortuite. Et cet ordre est bien celui que
l'ecrivain impose : `FUN_1406d60f4` recopie l'image memoire (donc UTF-16LE), la ou
`FUN_1407ece18` ecrit des scalaires de 16 bits MSB d'abord.

### A.5 G-TXT : la contradiction du 30/08, tranchee

Rejeu exact de la lecture du 30/08 (balayage du corps a la recherche de suites UTF-16LE imprimables
alignees sur l'octet, 4 caracteres minimum), chaque touche etant situee dans l'enregistrement qui la
contient puis confrontee aux PLAGES des deux champs de texte — une appartenance, pas une tolerance.

| corpus | touches | dans le champ de nom | dans le bloc de queue | hors grammaire |
|---|---|---|---|---|
| 6 films temoins | 18 | **9** | **9** | **0** |

La reponse a la question ouverte n°1 est donc : **il y a plusieurs champs de texte, et aucune des
deux lectures du 30/08 n'etait fortuite.** Le balayage aligne sur l'octet n'accroche pas le DEBUT
d'un champ (les champs sont a une position quelconque du flux, et l'un est en unites
gros-boutistes) : il demarre au premier octet ou la donnee decalee tombe dans l'ASCII imprimable,
c'est-a-dire quelque part a l'interieur du champ. C'est pourquoi le 30/08 voyait `blpp` (la fin de
`VitaminA1688` decalee d'un bit), `rndrh` (`Madina97294`) ou `rphfn` (`AlliedLace98437`).

---

## B. LE VERDICT SUR LA PERSONNALISATION

### B.1 La structure : PROUVEE par l'executable

`FUN_140969c54` est le serialiseur **DELTA** de la meme structure que l'ecrivain de film : memes
offsets `+0xc10`, `+0xc12`, `+0xc14`, `+0xc34`, `+0xc36`, `+0xc38`, `+0xc48`, `+0xcb0`, `+0xcb8`,
et il appelle le meme `FUN_1407ecd00` sur la base. Mais la ou l'ecrivain de film recopie `+0xcc0`
en brut, lui le passe champ par champ :

```
140969edf: LEA RDX,[RSI + 0xcc0]
140969ee9: CALL 0x1407ec27c
```

et `FUN_1407ec27c(writer, cust)` ecrit des champs NOMMES, les noms etant passes en clair a
l'ecrivain de u32 `FUN_1407edaf4(writer, nom, valeur)` :

| nom du champ | indice | octet |
|---|---|---|
| `variantName` | `param_2[0x4f]` | `0x13C` |
| `model_region` / `model_permutation` | `param_2[2+2i]` / `[3+2i]` | boucle sur `param_2[0]` paires |
| `styleName` | `param_2[0x50]` | `0x140` |
| `themeName` | `param_2[0x1cc]` | `0x730` |
| `coatingName` | `param_2[0x1cd]` | `0x734` |
| `actionPose` | `param_2[0x1ce]` | `0x738` |

Le pool de chaines qui les porte (`0x143686770`..`0x1436868a0`) aligne, dans l'ordre : `unarmed`,
`variantName`, `styleName`, `regionOverrideName`, `themeName`, `coatingName`, `markerName`,
`desired-representation`, `variant-name`, `queued-replay-mission`, `unknown`, `region`,
`permutation`, `model_permutation`, `model_region`. C'est le vocabulaire d'une tenue Halo Infinite.

**Quatre fermetures arithmetiques, sans ajustement :**

```
0x738 + 4                  = 0x73C = 1 852   la largeur EXACTE du bloc brut de l'ecrivain de film
0x14C + 24 x 0x24          = 0x4AC           24 attaches d'armure (FUN_1407ebf44)
0x4AC + 7 x 0x58           = 0x714           7 objets a 8 couples region/permutation (FUN_1407eda5c)
puis 0x730 theme, 0x734 revetement, 0x738 pose
```

24 attaches, 7 objets, un theme, un revetement, une pose. **La structure est prouvee.**

### B.2 Le contenu : REFUTE par les films

P-ZER, le contrôle ecrit en premier parce qu'une structure prouvee mais vide invalide d'avance
toute lecture de ses champs. 44 enregistrements, 6 films, 2 builds :

| zone | octets non nuls | sur | enregistrements ou la zone est non vide |
|---|---|---|---|
| masque `sub+0x000` | 3 082 | 14 915 | 35 |
| liste d'octets `sub+0x108` | 13 737 | 25 974 | 35 |
| liste de mots `sub+0x910` | 7 503 | 24 716 | 35 |
| bloc de 104 o `sub+0xc48` | **0** | 4 576 | **0** |
| bloc de 16 o `sub+0xc38` | **0** | 704 | **0** |
| **PERSONNALISATION `sub+0xcc0`** | **0** | **81 488** | **0** |
| bloc de queue `sub+0x1400` | 744 | 1 936 | 44 |

**Le bloc de personnalisation est ecrit integralement a zero dans les 44 enregistrements.** Les
champs nommes valent tous `00000000`, les 24 attaches d'armure sont vides (0/24 sur chaque
enregistrement), les 7 objets aussi.

### B.3 Et ce n'est pas ailleurs non plus

P-STA / P-DIS / P-PRE, sur les memes 44 enregistrements. Trois joueurs sont vus dans au moins deux
films (`JGtm` 5 films, `Madina97294` 4, `Chocoboflor` 2).

| zone | STABLE par joueur | empreintes distinctes | dont partagees | prefixe commun le plus court |
|---|---|---|---|---|
| masque | 0/3 | 31 | 1 | 8 o sur 442 |
| liste d'octets | 0/3 | 32 | 2 | 2 o sur 702 |
| liste de mots | 0/3 | 32 | 2 | 0 o sur 668 |
| bloc de 104 o | 3/3 | **1** | **1** | 104/104 |
| bloc de 16 o | 3/3 | **1** | **1** | 16/16 |
| PERSONNALISATION | 3/3 | **1** | **1** | 1852/1852 |
| bloc de queue | 0/3 | 43 | 0 | 0 o sur 44 |

Lecture : les trois zones a `1 empreinte partagee par 36 xuids` sont celles remplies de zeros —
**P-STA seul ne prouve rien, une zone de zeros le passe**, ce que P-DIS revele. Et les zones
reellement remplies ne sont **pas** stables par joueur : leur prefixe commun chez un meme joueur
d'un match a l'autre s'arrete au bout de **0 a 16 octets**. Aucune zone de l'enregistrement ne
porte une tenue en clair.

### B.4 Ce que la partie variable contient quand meme

P-TAG, sur la liste de mots de 32 bits (`sub+0x910`, 167 mots quand elle est pleine) : **6 179 mots
lus, 635 valeurs distinctes, dont 305 a `0xFFFFFFFF`** (la marque « absent » d'un identifiant de
tag), **4 451 inferieurs a 256** et **1 423 hauts**. La tete de la liste est reconnaissable et
partiellement stable par joueur (`JGtm` : `00008593 ffffffff 42c9679f` sur 5 films ;
`Madina97294` : `cc614892 ffffffff 42c9679f` sur 4 ; `42c9679f` est commun a presque tous les
joueurs). **Mesure, pas conclusion** : la zone melange des valeurs a forme d'identifiant, une
majorite de petits entiers et au moins une constante partagee. Son role n'est pas etabli.

### B.5 Verdict chiffre

| enonce | verdict |
|---|---|
| « L'enregistrement de slot RESERVE une structure de personnalisation (armure, revetements, theme, pose, regions/permutations) » | **PROUVE** — noms de champs dans l'exe + 4 fermetures arithmetiques |
| « Le film PORTE la personnalisation des joueurs » | **REFUTE** — 0 octet non nul sur 81 488 |
| « Les deux classes de longueur sont avec / sans le bloc de personnalisation » | **REFUTE** — bloc de largeur fixe toujours ecrit ; les classes viennent des trois listes prefixees (fermeture a 10 992 = 10 960 + 32 bits) |
| « Une autre zone de l'enregistrement porte la tenue » | **REFUTE** — prefixe commun par joueur de 0 a 16 octets sur 447/702/668 |
| « Les identifiants du bloc se retrouvent dans les catalogues du jeu » | **NON TESTE** — sans objet : le bloc est vide. Voir « ce qui reste ouvert » n°2 |

Consequence pour le chantier decodeur : **le rejeu 2D ne peut pas afficher la tenue d'epoque d'un
joueur a partir de `chunk_00`.** Le Theater du jeu la retrouve ailleurs (service ou cache local), ce
qui est coherent avec le fait que le bloc est reserve mais vide dans le flux enregistre.

---

## C. L'equipe et l'ordre des slots

### C.1 G-EQP : l'equipe n'est pas dans la partie decodee

Deux lentilles distinctes (methode, regle 7), toutes deux ecrites avant la mesure : (a) confrontation
a un vecteur d'equipes de reference releve dans `match_participants` ; (b) oracle interne — un champ
d'equipe doit partager chaque roster en deux moities de taille egale.

Distribution des neuf champs candidats sur les **44** enregistrements des 6 films :

| champ | valeurs observees | partage un roster en deux moities egales |
|---|---|---|
| `b2` (2 b) | `0` x44 | 0/6 films |
| `f10` (10 b) | `183` x44 | 0/6 |
| `f14` (14 b) | `0` x44 | 0/6 |
| `f6` (6 b, signe) | `-1` x44 | 0/6 |
| `f8` (8 b) | `0` x44 | 0/6 |
| `f7` (7 b) | `0` x44 | 0/6 |
| `f1` (1 b) | `0` x8, `1` x36 | 0/6 |
| `repr` (32 b) | `0` x44 | 0/6 |
| `u32Tete` (32 b) | `0` x44 | 0/6 |

Huit champs sur neuf sont **constants sur tout le corpus** : ils ne peuvent porter aucune
information par joueur. Le neuvième, `f1`, prend deux valeurs, et ce sont exactement les 9
enregistrements a listes vides (`N = M = 0`) contre 35 a listes pleines : c'est un **drapeau de
presence des listes**, pas une equipe. **L'equipe n'est donc pas dans les 78 bits de champs courts
du sous-enregistrement.** Elle est dans le masque de presence, dans l'une des deux listes, ou
absente de `chunk_00`. La question ouverte n°1 de la phase 1 est fermee **par la negative**, sur
piece.

Note de methode : la lentille (a) ne discrimine rien ici (un champ constant a 0 « coincide » avec
l'equipe 0 sur la moitie des slots, mecaniquement). C'est la lentille (b), interne, qui tranche —
illustration de la regle 7 : deux lentilles qui posent la meme question ne valent qu'une.

### C.2 G-IDX : l'ordre des enregistrements EST le `player_index` de production

Reference externe : le `roster[].filmIndex` des documents de rejeu deja produits par le depot
(`resolvePlayerIndices`, aujourd'hui reconstruit par le fil des morts). Corpus : les **76 films** du
cache qui portent a la fois un `chunk_00` et un document de rejeu.

| mesure | resultat |
|---|---|
| slots ou `rang == filmIndex` | **605 / 637** |
| films ou la coincidence est TOTALE | **72 / 76** |
| films ou `filmIndex - rang` est **CONSTANT** | **76 / 76** |

La troisieme ligne est la bonne question et elle est decisive : dans les 4 films en desaccord, tous
les rangs sont decales du **meme** nombre (+2, +3, +4, +7). Ce n'est pas une permutation, c'est une
**troncature de tete** du lecteur — la grappe terminale perd ses premiers enregistrements quand le
balayage ne les accroche pas. **L'ordre de la table de `chunk_00` est celui du `player_index`.**

Gain disponible : une table explicite `rang -> xuid` remplace l'inference par le fil des morts
(donnee a 77 % d'accord contre l'oracle killsource). Il reste a rendre le lecteur robuste a la
troncature de tete avant d'en faire quoi que ce soit de production — **hors perimetre de ce lot**.

---

## D. Le profil par build, sur les 1 351 films du cache

La phase 1 avait conclu « le cache ne contient plus que deux builds » et verifie la table des slots
sur deux. **C'est faux.** Une premisse negative se re-teste a chaque reprise (methode, erreur A) ;
celle-la ne l'avait pas ete. `TestD1Builds` classe les 1 351 films en **13 groupes** (build, version,
cardinal de la table par type) portant **7 builds distincts**.

`TestSection3SlotProfilBuilds` mesure, sur tout le cache et sans liste de films ecrite a la main :

| build | films | offset de la chaine de build | ecart a `0x0CB414` | ecart `mesure - predit` sur la longueur d'un enregistrement | films ou l'ecart est unique | gamertags imprimables |
|---|---|---|---|---|---|---|
| `HI_1_13_0` | 1 123 | `0x0CB414` | 0 | **0** x7 447 | 1 087/1 123 | 8 585/8 669 |
| `HI_1_12_0` | 146 | `0x0CB414` | 0 | **0** x1 165 | **146/146** | 1 311/1 326 |
| `HI_1_11_0` | 39 | `0x0C7310` | **-16 644** | **-2 880** x798 | 31/39 | 846/855 |
| `HI_1_10_0` | 26 | `0x0C730C` | **-16 648** | **-2 880** x481 | 21/26 | 513/517 |
| `HI_1_9_0` | 1 | `0x0C730C` | **-16 648** | **-4 320** x23 | 1/1 | 24/24 |
| `HI_1_8_0` | 10 | `0x0C730C` | **-16 648** | **-4 320** x127 | 9/10 | 138/139 |
| `HI_1_4_1` | 1 | `0x0C72F8` | **-16 668** | **+1 600** x23 | 1/1 | 24/24 |
| sans identification | 5 | absente | — | **+1 600** x113 | 4/5 | 119/119 |

Trois faits s'y lisent, et ce sont des donnees de PROFIL pour le chantier decodeur :

1. **La grammaire du slot se transpose par UNE SEULE CONSTANTE par build** : -2 880 bits (360
   octets) pour `HI_1_11_0` et `HI_1_10_0`, -4 320 bits (540 octets) pour `HI_1_9_0` et `HI_1_8_0`,
   +1 600 bits (200 octets) pour `HI_1_4_1`. La constante est la meme sur TOUS les enregistrements
   d'un film (criterion ecrit d'avance : un changement de build change la partie FIXE, donc le meme
   ecart partout). Les films a plusieurs ecarts sont ceux ou le balayage ajoute une position
   parasite ; l'instrument les ecarte deja par le champ de nom, il en reste sur les gros rosters.
2. **La tete de l'enregistrement et le champ de nom sont stables a travers les sept builds** :
   **11 550 / 11 728** enregistrements rendent un gamertag imprimable, tous builds confondus
   (98,5 %). Le XUID a 85 bits de l'en-tete et le champ `sub+0xc14` ne bougent pas.
3. **L'en-tete du fichier, lui, se decale — et le decalage se ferme arithmetiquement.**
   `16 640 = 0x4100` est **exactement un bloc de registre ECS** (la phase 1 en compte 50 dans
   l'exe courant), et les restes valent `4 × 1`, `4 × 2`, `4 × 7`, soit 4 octets par entree
   manquante de la table par type — dont `TestD1Builds` mesure les cardinaux **123, 122, 117**
   contre **124** sur `HI_1_13_0`/`HI_1_12_0` :

   ```
   HI_1_11_0             : 16 640 + 4 x 1 = 16 644   (124 - 123 = 1)
   HI_1_10_0/9_0/8_0     : 16 640 + 4 x 2 = 16 648   (124 - 122 = 2)
   HI_1_4_1              : 16 640 + 4 x 7 = 16 668   (124 - 117 = 7)
   ```

   Trois fermetures independantes, sans ajustement. Les builds anciens ont donc **49 blocs de
   registre au lieu de 50** et une table par type plus courte. La carte d'en-tete de la phase 1 est
   valide pour `HI_1_12_0` et `HI_1_13_0` **seulement** ; ailleurs, tout ce qui suit le registre se
   lit relativement a la chaine de build, pas a `0x0CB414`.

---

## E. Ce qui est ETABLI, ce qui est HYPOTHESE, ce qui est REFUTE

### Etabli

1. La grammaire complete du sous-enregistrement de slot : 16 champs, chaque largeur verifiee dans le
   code de son ecrivain, trois fermetures d'offset sans trou (A, A.1).
2. La grammaire predit la longueur totale d'un enregistrement : 38/38 sur 6 films, 560/567 sur
   76 films, les 7 restants imputables a une position parasite du balayage (A.2).
3. Le gamertag est a `sub+0xc14`, en unites de 16 bits MSB d'abord terminees par NUL, 16 au plus.
   616/636 contre l'oracle externe sur 76 films, 8/8 sur le seul film temoin qui en a un, et les
   deux gamertags du 30/08 tombent au bon XUID (A.3).
4. Un SECOND champ de nom existe : le bloc brut `sub+0x1400`, 44 octets, en UTF-16 petit-boutiste.
   640/640 sur 76 films ; 0/44 en gros-boutiste (A.4).
5. Tout texte UTF-16 aligne sur l'octet du corps appartient a l'un des deux champs : 18/18, 0 hors
   grammaire (A.5). Question ouverte n°1 du 30/08 fermee.
6. La structure de personnalisation `sub+0xcc0` est nommee dans l'executable et sa largeur ferme
   exactement sur `0x73C = 1 852` (B.1).
7. Cette structure est ecrite INTEGRALEMENT A ZERO dans les films : 0 octet non nul sur 81 488
   (B.2).
8. Les deux classes de longueur viennent des trois listes prefixees, pas d'un bloc optionnel :
   fermeture a `10 992 = 702 × 8 + 167 × 32 + 32` (resume n°2).
9. Les neuf champs courts du sous-enregistrement sont constants sur le corpus ; aucun ne porte
   l'equipe (C.1).
10. L'ordre des enregistrements est celui du `player_index` de production : `filmIndex - rang`
    constant sur 76/76 films (C.2).
11. Le cache porte 7 builds ; la grammaire du slot y tient a une constante pres, et le XUID comme
    le nom survivent partout (D).
12. Le decalage d'en-tete par build vaut un bloc de registre plus 4 octets par entree manquante de
    la table par type : trois fermetures sans ajustement (D.3).

### Hypothese (une seule chaine, ou non tranche)

1. **Le role des trois listes prefixees.** Elles portent le seul contenu variable de
   l'enregistrement, mais ni le masque de 2 048 bits, ni les 702 octets, ni les 167 mots ne sont
   stables par joueur. La tete de la liste de mots l'est partiellement (B.4). Lecture la plus
   economique, **non prouvee** : une description d'etat de session (dotation, drapeaux de presence)
   melangee a des identifiants.
2. **Le champ de 48 bits `slot+0x09` est un jeton de session** (phase 1, inchange).
3. **Les trois u32 du bloc de queue** (`ffffffff`, `ff000000`, un petit entier) : ordre variable par
   build, role non etabli.
4. **`desired-representation` (`sub+0xcb0`)** : le nom vient de l'executable, la valeur est nulle
   sur les 44 enregistrements. Aucune lecture possible.

### Refute

1. **« Le film porte la personnalisation des joueurs. »** REFUTE : 0/81 488 octets non nuls.
2. **« Les deux classes de longueur sont avec / sans bloc de personnalisation. »** REFUTE : le bloc
   est de largeur fixe et toujours ecrit.
3. **« Une autre zone de l'enregistrement porte la tenue du joueur. »** REFUTE : prefixe commun par
   joueur de 0 a 16 octets.
4. **« L'equipe est dans l'un des champs courts du sous-enregistrement. »** REFUTE : huit champs
   constants, le neuvième est un drapeau de presence des listes.
5. **« Les deux positions de gamertag du 30/08 sont peut-etre une lecture fortuite. »** REFUTE : les
   deux sont des champs de texte de la grammaire, et 18/18 touches sont expliquees.
6. **« Le cache local ne contient plus que deux builds. »** (phase 1, section G.3) REFUTE : sept
   builds, 13 groupes, 1 351 films.
7. **« La carte d'en-tete de `chunk_00` vaut pour tous les films. »** REFUTE : elle vaut pour
   `HI_1_12_0` et `HI_1_13_0` ; ailleurs l'en-tete est plus court de 16 644 a 16 668 octets.

---

## F. Commandes pour rejouer

Les instruments sont sous garde d'environnement ; sans la variable ils sont sautes, donc la CI reste
verte. Les chemins sont **en style Windows** (`C:/...`) : un chemin de style Git Bash (`/c/...`)
fait echouer l'ouverture.

```bash
cd apps/go-api
C="C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks"
R="C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/replays/halo_infinite"
B="C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/backups/pre-restauration-cle-2026-08-20/replays_local/halo_infinite"
T="$C/000d5950;$C/00162144;$C/00502e52;$C/0014603f;$C/007d53a4;$C/02784ce1"

# A.2 / A.3 : la grammaire ferme, et le gamertag sort (38/38 et 44/44)
CHUNK00_FILMS="$T" go test ./internal/analysis/filmdec/ \
  -run 'TestSection3SlotGrammaire|TestSection3SlotGamertag' -v -timeout 30m

# A.4 / A.5 : le second champ de nom et la contradiction du 30/08 (640/640 et 18/18)
CHUNK00_FILMS="$T" go test ./internal/analysis/filmdec/ \
  -run 'TestSection3SlotBlocQueue|TestSection3SlotTexte' -v -timeout 30m

# B : LE VERDICT PERSONNALISATION (P-ZER puis P-STA/P-DIS puis P-PRE puis P-CHP/P-TAG)
CHUNK00_FILMS="$T" go test ./internal/analysis/filmdec/ \
  -run 'TestSection3SlotPerso' -v -timeout 30m
CHUNK00_FILMS="$T" go test ./internal/analysis/filmdec/ \
  -run 'TestSection3SlotListes' -v -timeout 30m

# C.1 : l'equipe n'est dans aucun champ court
CHUNK00_FILMS="$T" CHUNK00_REPLAYS="$R;$B" CHUNK00_EQUIPES="000d5950=0,1,0,0,1,1,1,0" \
  go test ./internal/analysis/filmdec/ \
  -run 'TestSection3SlotEquipe|TestSection3SlotChampsCourts' -v -timeout 30m

# C.2 : l'ordre des slots contre le player_index de production, sur les 76 films a document
# de rejeu. La liste se construit par intersection, elle n'est pas ecrite a la main :
L=""; for f in "$R"/*.json; do b=$(basename "$f" .json); case "$b" in *.derived) continue;; esac
  [ -f "$C/$b/chunk_00.bin" ] && L="$L;$C/$b"; done
CHUNK00_FILMS="${L#;}" CHUNK00_REPLAYS="$R" go test ./internal/analysis/filmdec/ \
  -run 'TestSection3SlotIndex|TestSection3SlotGrammaire|TestSection3SlotBlocQueue' -v -timeout 40m

# D : le profil par build, sur les 1 351 films du cache (~4 min)
CHUNK00_CORPUS="$C" go test ./internal/analysis/filmdec/ \
  -run TestSection3SlotProfilBuilds -v -timeout 60m

# D (releve film par film, un film par build)
CHUNK00_FILMS="$C/000d5950;$C/0014603f;$C/00ba2e1c;$C/084a804d;$C/11de8353;$C/0a247154;$C/a521164d" \
  go test ./internal/analysis/filmdec/ -run TestSection3SlotProfilExemples -v -timeout 30m

# le classement des builds dont la section D part
D1_BUILD_DIR="$C" go test ./internal/analysis/filmdec/ -run TestD1Builds -v -timeout 60m
```

Le vecteur d'equipes de `CHUNK00_EQUIPES` se relit dans `shared_matches_v2.duckdb` (**lecture
seule** ; le fichier peut etre tenu par le serveur, dans ce cas ne pas forcer), dans l'ordre du
`filmIndex` :

```sql
SELECT substr(match_id,1,8), string_agg(team_id, ',')
  FROM match_participants WHERE xuid NOT LIKE 'bid(%' GROUP BY 1;
```

Gates passes : `gofmt -l` net, `go vet ./internal/analysis/filmdec/` net,
`go test ./internal/analysis/filmdec/` sans garde **ok**, `go test ./internal/archlint/` **ok** — le
ratchet des variables de paquet de `filmdec` n'est pas touche (les instruments n'introduisent que
des `const` et des locales).

---

## G. Decouvertes hors perimetre — notees, NON TRAITEES (regle 7)

1. **Le decalage de 8 octets de `parseRegistry`** : connu depuis la phase 1, **non tranche**, pas
   touche par ce lot.
2. **`TestD1Builds` compte 124 entrees de table par type sur `HI_1_13_0` la ou l'ecrivain en ecrit
   123** (`MOV R9D,0xf60`). Les ecarts RELATIFS entre builds (123, 122, 117) sont coherents et
   c'est eux que la section D utilise ; la difference d'un cran vient de l'heuristique de
   `lireEntete` (elle remonte tant que la valeur tient sur 16 bits) et n'est pas tranchee.
3. **Le lecteur perd la tete de la table sur 4 films sur 76** (C.2). C'est le seul defaut connu du
   lecteur canonique. Le corriger demande de comprendre pourquoi le balayage n'accroche pas les
   premiers enregistrements de ces films — non traite.
4. **5 films du cache n'ont pas de section d'identification** et se comportent comme `HI_1_4_1`
   (ecart +1 600). Leur build reste inconnu.

---

## Ce qui reste ouvert

1. **Ou est l'equipe ?** Pas dans les champs courts (C.1). Restent le masque de presence de
   2 048 bits, les 702 octets et les 167 mots. Le geste suivant est de suivre le CONSOMMATEUR
   (methode, regle 1) : trouver le desserialiseur de `FUN_1407ecd00` et lire qui compare le
   resultat a des constantes.
2. **A quoi servent les 167 mots de 32 bits ?** Leur forme melange identifiants (`0xFFFFFFFF`,
   valeurs hautes) et petits entiers (B.4). Le contrôle « les identifiants se retrouvent dans les
   catalogues du jeu » n'a pas ete execute : il exige le corpus `gamefiles` (lecture de
   l'installation locale, dizaines de minutes, tag `//go:build gamefiles`) et il n'a de sens qu'une
   fois quelques mots identifies comme des identifiants de tags. **A planifier, pas a deviner.**
3. **Pourquoi la personnalisation est-elle reservee et vide ?** Deux branches a departager par une
   mesure (methode, erreur E) : (a) le jeu ne la serialise que dans un autre contexte (le
   serialiseur DELTA `FUN_140969c54`, c'est-a-dire la replication en ligne, ou elle est ecrite
   champ par champ) ; (b) elle est remplie a la lecture par le Theater depuis une autre source.
   Un film de partie personnalisee ou un `chunk` ulterieur pourrait trancher.
4. **La constante par build de la longueur d'un enregistrement** est mesuree (D) mais pas
   EXPLIQUEE : quel champ a change de taille entre `HI_1_11_0` et `HI_1_12_0` (+360 octets) puis
   entre `HI_1_8_0` et `HI_1_11_0` (+180 octets) ? Le candidat naturel est le bloc de
   personnalisation lui-meme, qui aurait grandi avec le systeme de tenues — a verifier sur un
   executable de ces builds, qu'on n'a pas.
5. **Le lecteur canonique n'est pas robuste a la troncature de tete** (G.3) ni aux builds anciens
   (il faudrait calibrer la constante sur le film). Les deux sont des travaux de decodeur, pas de
   rétro-ingenierie.
6. **Le joueur manquant de `00162144`** (phase 1, D.2) : toujours non tranche.
