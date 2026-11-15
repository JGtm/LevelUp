# A2 - Relecture de désassemblage (lecture seule, analyse statique)

## Identité de l'exécutable et concordance des adresses

- Programme ouvert : `HaloInfinite.exe`, projet Ghidra « HI », pid 30192 (`GET /mcp/instance_info`), PE x86-64, base `0x140000000`, 311 103 fonctions, taille mappée 95 404 712 octets, `SizeOfImage` 0x5b00000.
- Horodatage PE (lu en mémoire statique, `0x140000190`) : `TimeDateStamp` = 1775692580 = 2026-04-08 23:56:20 UTC. Création du projet Ghidra : 2026-06-04. Chemin d'import : `D:\SteamLibrary\...\HaloInfinite.exe` (le fichier n'existe plus à cet emplacement : pas d'empreinte SHA calculée).
- Concordance avec le dépôt : OUI. `FUN_14076e524` décompilé (lu) fait exactement ce que dit l'en-tête de `libs/film/internal/grammar/lecteur_position.go` : porte `FUN_1406cf008` (1 bit), index sur `DAT_144632be0` bits si porte = 0, table PAR INDEX `DAT_1445ccbe0 + (index*0x20 + NIVEAU)*0xc`, table DEFAUT `DAT_1445cc9e0 + NIVEAU*0xc` pour index -1, puis `FUN_140cc5128` (trois axes). `FUN_14076e494`, `FUN_14076e420`, `FUN_14076e29c`, `FUN_142f262d4`, `FUN_1406cbaa0`, `FUN_1406caad8`, `FUN_1406cf548` etc. ont la forme décrite par le dépôt. C'est donc le build dont le dépôt cite les adresses (celui que le dépôt appelle HI_1_13_0, majeure 0x29). Aucune recherche « par la forme » n'a été nécessaire.
- Méthode : API HTTP locale du greffon, points d'accès de lecture seulement (`decompile_function`, `disassemble_function`, `read_memory`, `get_xrefs_to`, `get_function_xrefs`, `get_function_callers`, `search_strings`, `get_metadata`, `get_current_program_info`). Rien renommé, commenté, typé, importé ; aucun script ; aucun outil de débogage. Les sorties brutes sont sous `C:\t\marche\A2\d\` (une décompilation par fichier) ; scripts jetables `gen.py`, `dom.py`, `cmp.py`, `bits.py` dans le même dossier.
- Convention : LU = vu dans la décompilation ou le désassemblage ; DÉDUIT = conséquence que j'en tire. `R(n)` = lecture de n bits. Piège de lecture rencontré : le décompilateur perd des arguments de registre (ex. le niveau de `FUN_14076e524`, le 4e argument de `FUN_140be9b88`, la destination `&dst[2]` de `FUN_14297ea84`) ; chaque fois, j'ai relu le désassemblage pour trancher.

---

## Point 1 - Fiche P : position par index de région

Adresses lues : `FUN_14076e524`, `FUN_1406cf008`, `FUN_140cc5128`, `FUN_14076e494`, `FUN_14076e420`, `FUN_14076e29c`, `FUN_141f85880`, `FUN_140be9a14`, `FUN_140be9b88`, `FUN_140be9c78`, `FUN_140be9d1c`, `FUN_1406d310c`, `FUN_140365eb0`, `FUN_140de9a1c`, `FUN_140de9aa8`, `FUN_140770640` (écrivain), `FUN_140ee7270` (flock-position), `FUN_140fb8af0` (flock-destination), `FUN_1408096f8` (genre 5), `FUN_1410f03b4` (genre 6).

### Lecture (LU)
```
FUN_14076e524(dst, lecteur, idxOut, NIVEAU)        // NIVEAU = immédiat du site d'appel
  g = R(1)                                         // FUN_1406cf008 : vrai si le bit vaut 1
  si g == 0 : idx = R(DAT_144632be0)               // index lu dans le flux
  sinon     : idx = 0xffffffff                     // ligne DEFAUT
  ligne = (idx == -1) ? DEFAUT[NIVEAU] : PARINDEX[idx*0x20 + NIVEAU]    // 3 entiers de 4 octets, pas 0xc
  pour axe 0..2 : q[axe] = R(ligne[axe])           // FUN_140cc5128, largeurs de LA LIGNE
  valeur = q*pas + borne_min + pas*0.5             // pas = (max-min)/2^largeur, bornes de la ligne idx (DAT_14462cbe0 + idx*24)
```
Oui : un index de région puis trois axes aux largeurs de LA LIGNE DE CET INDEX (le jeu ne regarde jamais « la région jouée » : il n'y a pas de notion de région jouée dans la lecture).

### Combien de bits d'index, d'où vient le nombre (LU)
`DAT_144632be0` est écrit à trois endroits :
- `FUN_140be9a14` (chargement de carte, appelé par `FUN_140be9890`) : `1` si la carte déclare exactement UNE plage (`CMP ECX,1 ; JZ 1423c6b40`), sinon `FUN_1406d310c(compte)` = ceilLog2(compte) (compte BRUT de plages, `*(DAT_144976b60+0x7bc)`, y compris les plages invalides ; compte 0 donne 0 bit). Vérifié : n=2 -> 1, n=3 -> 2, n=4 -> 2.
- `FUN_140de9a1c` (déchargement de la carte, via `FUN_140aa3814`) : `FUN_1406d310c(0x400)` = 10 ; table PAR INDEX mise à zéro (`memset 0x60000` = 1024 régions x 0x20 niveaux x 0xc) ; bornes de toutes les régions = bornes par défaut (`DAT_143b8c6b8`, +/-20000).
- `FUN_140365eb0` (init statique) : 10 également, mêmes zéros.
Donc la valeur est celle de la carte chargée, pas une constante.

### Remplissage de la table (LU)
`FUN_140be9a14` parcourt les plages de la carte (tableau à `DAT_144976b60+0x7ac`, pas 0xdc, bornes à +0x44) ; l'indice `i` est la POSITION dans ce tableau (il avance même pour une plage invalide). Pour chaque plage dont l'AABB est valide (`FUN_140be9d1c` : min < max sur les trois axes) : bit posé dans le masque `DAT_1445ccb60`, bornes copiées dans `DAT_14462cbe0 + i*0x18`, puis 32 appels `FUN_140be9b88(niveau, bornes, ..., destination = DAT_1445ccbe0 + i*0x180 + niveau*0xc)` (destination vérifiée au désassemblage : `MOV R9,R15 ; ADD R15,0xc`). La table DEFAUT est remplie par la même loi sur les bornes `DAT_143b8c6b8` (+/-20000).
Loi `FUN_140be9b88` : pas `q = FUN_140be9c78(niveau)` = `C * 2^(16-niveau)` (niveau <= 16) ou `C / 2^(niveau-16)` avec `C` = `0x3c088889` (1/120) ; si `q < 1e-4` (`0x38d1b717`) : 26 bits partout ; sinon par axe : étendue < `2q * 2^22` ? `n = ceil(étendue/(2q))` : `n = 0x400000` ; largeur = min(26, ceilLog2(n)). Identique à `profile.LargeursAxeDuNiveau` (relu : `loi_largeurs.go:117`). Au niveau 0x10 sur +/-20000 : 22 bits ; chaque niveau en moins retire 1 bit tant que la loi n'est pas plafonnée.

### Index sans ligne (LU + DÉDUIT)
- LU : le lecteur ne teste JAMAIS le masque des plages valides (`DAT_1445ccb60`) ; ce masque ne sert qu'à l'ÉCRIVAIN (`FUN_140770640` : un index dont le bit n'est pas posé retombe sur -1 / porte posée).
- LU : une ligne jamais remplie (plage invalide, ou index >= nombre de plages mais < 2^N) reste à zéro (`memset` du déchargement).
- DÉDUIT : le lecteur lit alors 0 bit par axe (largeur 0) : la position consomme `1 + N` bits au total, les trois axes ne coûtent rien. La valeur numérique est sans signification (`FUN_140cc5128` avec largeur 0 décale de `64 & 63 = 0` et rend les 32 bits bas de la fenêtre) mais aucun bit de plus n'est consommé. Le jeu ne signale pas d'erreur et ne retombe pas sur « la région jouée ».
- Ce que fait le témoin (`pos_codes` / `point`, d'après la fiche) pour une boîte non listée : « lit comme l'espace de jeu » (largeurs de la région jouée). Ce n'est PAS ce que fait le jeu pour un index sans ligne (0 bit). Écart de comportement sans doute sans effet sur le corpus (index non déclarés rares), à garder en tête si une mesure en révèle un.

### Variantes (LU)
- Position d'objet ti 36 à 43 composant 0 : `FUN_14076e29c(…)` = `FUN_14076e420(dst, lecteur, 0x10)` : `precHigh = R(1)` ; `FUN_14076e494(..., p6 = precHigh ? &DAT_143b8c6d0 : 0)` ; `FUN_14076e3e4(precHigh)` (0 bit si precHigh = 0) ; `FUN_140492128` (test de finitude, 0 bit) puis `R(2)` (`FUN_14076e304`) si fini. `FUN_14076e494` : `FUN_14076f91c()` (garde de pleine précision, globale `DAT_144e61ea0` ou `DAT_145121140 == 1`, 0 bit) vrai -> `FUN_1411b259c` = R(96) brut ; sinon p6 == 0 -> `FUN_14076e524(NIVEAU)` ; sinon `FUN_141f85880` : largeurs de la loi sur les bornes +/-100 (`DAT_143b8c6d0`), trois axes, SANS porte ni index (au niveau 0x10 : 3 x 14). La lecture par index de région vaut donc pour ces composants, aux largeurs de la ligne de l'index lu.
- `flock-position` ti=21 composant 16 : `FUN_140ee7270` : `FUN_14076f91c()` ? R(96) : `FUN_14076e524(NIVEAU = 0x10)`. Donc : porte R(1) ; si 0 : index sur `DAT_144632be0` bits ; trois axes aux largeurs de la ligne de l'index (ou 22/22/22 porte posée). C'est ce que lit le témoin. Notre `consumeFlockPosition` (index figé à 1 bit, axes à `min(26, 6 + niveau du registre)`) ne correspond PAS au jeu.
- `flock-destination` ti=21 composants 2 à 11 : `FUN_140fb8af0` : `R(1)`, puis la même lecture que ci-dessus (`FUN_14076f91c` ? R(96) : `FUN_14076e524(0x10)`), puis `R(2)` si la version du composant > 1. Même défaut que le flock-position dans notre `consumeFlockDestination` (ancien lecteur au niveau du registre) : même classe de correction.
- Messages de genre 5 et 6 : `FUN_1408096f8` (`projectile_detonate`) appelle, après `FUN_14076f91c`, `FUN_14076e524` avec `R9D = 0xf` (désassemblage 140809775) ; `FUN_1410f03b4` (`projectile_impact_effect`) avec `R9D = 0xc` (1410f044d). Précision réduite = ligne de niveau plus bas dans la MÊME table par index : `0x10 - niveau` bits de moins par axe (1 bit pour le genre 5, 4 bits pour le genre 6), tant que la loi n'est pas plafonnée (DÉDUIT de `FUN_140be9c78` / `FUN_140be9b88`). Le témoin (`16 - précision`, `saturating_sub`) et notre `niveauDetonation = 0xf` / `niveauImpact = 0xc` sont cohérents avec le jeu.

VERDICT P : confirme la fiche (la position lit un index de région puis trois axes aux largeurs de la ligne de CET index ; notre `consumeFlockPosition`, `consumeFlockDestination` et la restriction « région jouée » des genres 5 et 6 divergent du jeu).

---

## Point 2 - Fiche A : `biped-spartan-ability`, tag 3

Adresses lues : `FUN_142f268c4`, `FUN_142f262d4`, `FUN_14297ea84`, `FUN_140f03dfc`, `FUN_142f04664`, `FUN_140c1e924`, `FUN_1406d3140`, `FUN_14076e494`.

### La question exacte (LU, désassemblage)
`FUN_142f262d4(dst = RCX->RDI, lecteur = RDX->RBX, p3 = R8B)` :
```
142f262f5  CALL FUN_140f03dfc        // initialise dst : dst[0]=0, dst[1]=0, dst[2]=0, dst[3]=0, ...
142f262fd  CALL FUN_1406cf008        // a = R(1) -> dst[0]
142f26308  LEA RSI,[RDI+2]
142f2630f  MOV R8,RSI                // <- 3e argument = &dst[2]
142f26312  CALL FUN_14297ea84        // lit R(6) et l'ECRIT dans *(byte*)(dst+2)
142f2631a  TEST byte ptr [RSI],1     // la porte teste dst[2], c'est-à-dire LES 6 BITS QUE LE FLUX VIENT DE DONNER
```
`FUN_14297ea84(lecteur, p2, byte *out)` : `R(6)`, rangé dans `*out`. Le décompilateur masque la destination (il écrit `FUN_14297ea84(param_2)`), le désassemblage montre `R8 = RDI+2`.
Réponse : la porte `dst[2] & 1` porte sur les 6 bits lus dans le flux ; `FUN_140f03dfc` met bien `dst[2]` à zéro, mais EN AMONT de la lecture (premier appel de la fonction), puis `FUN_14297ea84` l'écrase. Le commentaire du lot 5.13.3 de `components_biped_spartan.go` (« dst[2] reste zéro, la porte est toujours vraie ») est donc FAUX : la porte dépend du flux.

### Suite complète de lectures (LU) - tag 3 après `tag = R(2)` (`FUN_142f268c4`)
```
a = R(1)                                   // dst[0]
si a :
    flags = R(6)                           // dst[2]  (FUN_14297ea84)
    si flags & 1 :
        c = R(1)
        si c == 0 :
            FUN_142f04664(dst+4, c=0) = FUN_14076e494(NIVEAU 0x10)   // une position (garde R(96) ou FUN_14076e524)
        sinon (c == 1) :
            FUN_1406d3140(catégorie 0)     // entier à largeur variable : 13 bits (plage 0x1dff) + 2 bits de queue
            si flags & 0x10 :
                FUN_142f04664(dst+4, c=1) = R(2) ; 3 x R(13) (FUN_140c1e924, largeur R9D = 0xd) ; R(1) ; si 1 : R(16)
            sinon : rien d'autre
b = R(1)                                   // dst[1]
si b : FUN_14076e494(dst+0x18, NIVEAU 0x10)  // seconde position
```
(Ordre vérifié au désassemblage : après la porte, le test `c != 0 && (flags & 0x10) == 0` met `BPL = 0`, lit l'entier variable, puis saute le corps `FUN_142f04664` ; `c != 0 && (flags & 0x10)` lit l'entier variable puis le corps ; `c == 0` saute l'entier variable et lit le corps.)

### Écart avec le témoin sur la branche c = 1 (LU + DÉDUIT)
Le témoin lit `skip(11 + 2)` = 13 bits. Le jeu appelle `FUN_1406d3140` avec la catégorie 0 : valeur = `FUN_1406d310c(DAT_1451f98d4)` bits, où l'init statique pose `DAT_1451f98d4 = DAT_144706100 - 0x200 = 0x1fff - 0x200 = 0x1dff`, soit 13 bits de valeur, PLUS 2 bits de queue = 15 bits. Notre `varWidthBits(0)` = 13 + `readVarWidthInt` 2 = 15 : notre code est d'accord avec le jeu, c'est le témoin qui diffère de 2 bits sur c = 1. Les trois occurrences mesurées par A1 sont toutes en c = 0 : la branche c = 1 n'a pas été exercée par la mesure. Réserve : `DAT_144706100` / `DAT_1451f98d4` sont aussi réécrits à la croissance de la table d'entités (`FUN_1408f1618`, `FUN_142f2f0cc` : `DAT_144706100 = slot + 1`, `DAT_1451f98d4 = slot - 0x1ff`) ; je n'ai pas pu établir statiquement si cette réécriture se produit en lecture d'un film avant ces lectures. Si elle se produisait, la largeur de la catégorie 0 dépendrait de l'état de la table (ce que le témoin 11 + 2 pourrait refléter) : à trancher par une occurrence c = 1 mesurée.

VERDICT A : confirme la fiche (la porte `dst[2] & 1` porte sur les 6 bits lus ; la branche gardée est atteignable). Réserve : sur c = 1 le jeu lit 13 + 2 = 15 bits, pas 13 comme le témoin.

---

## Point 3 - Fiche T : type d'une entité

Adresses lues : `FUN_1406cbaa0`, `FUN_1406caad8`, `FUN_1406cb5f0`, `FUN_1408f1aa4`, `FUN_1408f2150`, `FUN_1408f0d58`, `FUN_1408f1618`.

### Garde (LU)
Dans `FUN_1406cbaa0` cas 3 (mise à jour) : refus (code `3 - (état != 2)`, ZÉRO bit lu) si `eid == 0xffffffff`, ou `(eid & 0x3fffffff) >= nombre d'entrées` (table `*(*plVar3+0x20)`, pas 200 = 0xc8), ou `*(uint *)(slot*200 + base) != eid`. `FUN_1406caad8` répète la même garde (`!= param_2` -> 3) puis lit le type en `entrée + 4` (`puVar18[1]` : descripteur `*(vue+0x18) + 8 + type*8`). `FUN_1406cb5f0` (recherche d'une entrée) compare aussi `*(uint*)(slot*200 + t) == eid`. Donc la garde compare l'identifiant COMPLET : emplacement (30 bits bas) ET génération (2 bits de tête), car `eid` n'est jamais masqué dans la comparaison.

### Création, suppression, réallocation (LU)
- Création (`FUN_1408f1aa4` : `R(6)` = type, puis lecture du corps ; l'écriture de l'entrée est dans `FUN_1408f2150`) : `entrée[0] = eid` (génération comprise), `entrée[1] = type`, 0 dans l'état, `entrée[3] = 0xffffffff`, `entrée[0x24] = -1`, `entrée[0x25]/[0x26]` = taille et tampon de création, `[0x30]/[0x2e]` ; bit de vie posé dans le masque `param_1+0x50`. La table grandit si le slot dépasse sa taille (min 0x1fff).
- Suppression (`FUN_1406cbaa0` cas 2 -> `FUN_1408f0d58`) : retrouve l'entrée par `FUN_1406cb5f0` (eid complet), appelle le handler de suppression du type et `FUN_1408f0e98` (nettoyage de l'objet moteur). Dans ces fonctions je n'ai VU AUCUNE écriture qui efface `entrée[0]` ni `entrée[1]` : l'entrée garde son eid et son type jusqu'à la réallocation du slot (DÉDUIT : un delta portant l'ancien eid d'un slot supprimé mais pas réalloué passerait encore la garde ; non vérifié ailleurs).
- Réallocation sous une nouvelle génération (création avec un eid dont les 2 bits de tête ont changé) : `FUN_1408f2150` écrase `entrée[0]` par le nouvel eid et `entrée[1]` par le nouveau type. Un delta portant l'ANCIENNE génération échoue alors la garde (rejet, 0 bit). Un delta portant la nouvelle génération AVANT toute création vue échoue aussi.
- Génération : suivie à part dans une table de pas 0x18 (`*(byte*)(+1)` = `eid >> 30`, `FUN_1408f1618`).

### Conséquence pour notre code (DÉDUIT)
`World.ArchetypeForSlot(slot)` (clé = slot seul) ne reproduit pas la garde : le jeu lie le type à l'eid complet et le remplace à chaque création du slot.

VERDICT T : confirme la fiche.

---

## Point 4 - Fiche H : table d'entités au début d'un morceau

Adresses lues : `FUN_142e2bfd0` (lecteur d'état complet, appelé par `FUN_1428e2a9c`), `FUN_142e29cf8`, `FUN_142f2e174`, `FUN_142f30610`, `FUN_142f2c658`, `FUN_142f2c754`, `FUN_1406d5110`, `FUN_142f2f0cc`, `FUN_1408f1618`.

### Les fonctions citées par la fiche ne sont pas le dump du tableau (LU)
`FUN_142f2e174` parcourt la table de vues (pas 0xa0, masque `vue+0x58`) et produit un MOT de 32 bits par entité vivante (`génération << 30 | genre << 23 | slot & 0x1fff`) ; `FUN_142f2c658` les sérialise : `FUN_142f303bc` (kind 1), `FUN_142f304a8` (kind 2), `FUN_142f30610` (kind 3). `FUN_142f30610` appelle `FUN_142f2c754(writer, 3, eid, archetype)` qui écrit : (si contrôle de corruption `FUN_14076cea8`) 32 bits `0xf0c3a57e` ; 1 bit (`FUN_1406d49c4`) ; 2 bits de kind si kind != 3 ; l'eid par `FUN_1406d5110` (index sur `ceilLog2(plage)` bits, `-base`, puis 2 bits de génération) ; si kind != 3 et le contrôle est actif : 8 bits d'archétype ; puis `FUN_140769e08` (R(1) + R(7) sélecteur de base) et l'état. C'est le format des ENREGISTREMENTS de la vue B (create / delete / update), PAS un en-tête `[id:32][type:32]`. L'affirmation de `keyframe_datums.go` (« FUN_142f2e174 / FUN_142f30610 : l'image-clé est le dump de la table, en-tête `[id:32][field:26][ti:6]` ») n'est pas soutenue par ces fonctions.

### La table d'entités du morceau est lue par `FUN_142e2bfd0` (LU)
```
si version du format > 7 : DAT_144706104 = R(1)              // le [1] initial du témoin
boucle sur les entrées du vecteur (pas 200 octets = 0x32 uint),
       tant que (nombre lu < DAT_144dbfc90) :
    entrée[0] = R(32)          // eid, génération dans les 2 bits de poids fort
    entrée[1] = R(32)          // type ; 0xffffffff = emplacement vide
    entrée[3] = R(32)          // poignée d'objet (0xffffffff = aucune)
    FUN_142e29cf8             -> R(4)  (drapeaux)
    entrée+9  = R(8)
    si entrée[1] != 0xffffffff :
        n = R(32)                                  // taille de la structure de création
        si n > 0 : DAT_144e61ea0 = 1 ; lecteur de création du type (vtable+0x60) ;
                   si contrôle de corruption : R(32) ; DAT_144e61ea0 = 0
        m = R(32)                                  // taille de l'état
        si m > 0 : lecteur d'état du type (vtable+0x88) ; FUN_1428e2b68
```
Concordance avec le témoin (`snapshot.rs`) : id 32, type 32, 32 (poignée), 4 bits, 8 bits = 108 bits ; emplacement vide = 108 bits, rien après (le type est testé contre `0xffffffff`) ; vivant : `[n:32]` + création, `[m:32]` + état. CONFIRMÉ.

### Couvre-t-elle tous les emplacements ? (LU + DÉDUIT)
LU : la boucle parcourt les entrées DANS L'ORDRE DE L'INDEX, sans index écrit dans le flux, vivantes ou vides, bornée par le nombre d'entrées du vecteur et par `DAT_144dbfc90` (compteur d'entités, mis à `slot + 1` par `FUN_1408f2150` quand la table grandit ; vaut 0 dans l'image statique). DÉDUIT : la table couvre TOUS les emplacements de 0 à `DAT_144dbfc90 - 1`, les vides pesant 108 bits ; le NOMBRE d'entrées n'est pas écrit dans le flux (il vient de l'état du lecteur) : un décodeur hors ligne doit le déduire (le témoin cherche l'en-tête de chaque entrée, notre `TableDeDatums` aussi). Je n'ai pas relu le côté ÉCRITURE de cette boucle (borne exacte écrite) : non vérifié.

### Remarque utile pour la fiche C (LU)
Pendant la lecture de la structure de CRÉATION d'une entrée du tableau, `DAT_144e61ea0 = 1` : la garde `FUN_14076f91c` est vraie, donc toute position lue par `FUN_14076e494` / `FUN_140ee7270` / `FUN_140fb8af0` y est en R(96) brut, pas quantifiée. L'état (`m`) est lu garde à 0. Dans le chemin NEW par enregistrement (`FUN_1408f1aa4`), je n'ai vu aucune pose de cette garde.

VERDICT H : confirme la fiche (forme de l'entrée : 32 / 32 / 32 / 4 / 8 = 108 bits, génération dans les deux bits de poids fort de l'identifiant, emplacement vide = type `0xffffffff` ; la table couvre tous les emplacements dans l'ordre de l'index). Correction de citation : le lecteur est `FUN_142e2bfd0` ; `FUN_142f2e174` / `FUN_142f30610` sont l'écrivain des enregistrements de la vue B.

---

## Point 5 - Fiche V : têtes des messages de la vue des événements

Adresses lues : `FUN_140e453b4` (enregistrement des 123 descripteurs : genre `g` à `DAT_144e61d88 + 0x210 + g*8`), vtable de chaque descripteur (`+0x58` = domaine de la poignée d'indice i, `+0x68` = lecteur de charge), `FUN_14080a9d4` (tête de lecture, via le dépôt), `FUN_1406d3140`, `FUN_140d10bb0`.

### Méthode (LU)
Pour chacun des 123 genres : lecture mémoire de l'adresse du descripteur (extraite de `FUN_140e453b4`), de sa vtable, du pointeur `vtable+0x58`, puis exécution à la main (script `dom.py`, sous-ensemble d'instructions `test edx,edx / sub edx,1 / lea eax,[rdx+n] / jz / jnz / ret`) des trois appels i = 0, 1, 2. Aucune instruction inconnue ; chaque fonction est une cascade de 3 comparaisons. Contrôle : genre 5 -> `[5, 8, 7]`, lecteur de charge `+0x68` = `FUN_1408096f8` (cohérent avec le dépôt).

### Résultat (LU)
Comparaison des 123 triplets du jeu avec (a) `tableDesGenresVueA` du dépôt (`vue_a_genres.go`) et (b) `handle_kinds` du témoin (`events.rs`) : AUCUN écart sur les 123 genres. Donc, pour les genres demandés :

| genre | jeu | notre code | témoin |
|---|---|---|---|
| 0 | 1,1,7 | 1,1,7 | 1,1,7 |
| 5 | 5,8,7 | 5,8,7 | 5,8,7 |
| 6 | 5,8,7 | 5,8,7 | 5,8,7 |
| 9 | 2,8,7 | 2,8,7 | 2,8,7 |
| 15 | 8,8,7 | 8,8,7 | 8,8,7 |
| 36 | 1,8,7 | 1,8,7 | 1,8,7 |
| 75 | 1,8,7 | 1,8,7 | 1,8,7 |
| 82 | 0,8,7 | 0,8,7 | 0,8,7 |
| 104 | 0,0,7 | 0,0,7 | 0,0,7 |
| 105 | 0,0,7 | 0,0,7 | 0,0,7 |

(L'encodage est le même de part et d'autre : le chiffre est la catégorie de `FUN_1406d3140`.)

### Domaine -> largeur (LU)
`FUN_140d10bb0` (init, pose `DAT_144706104 = 1`) : catégories 0 et 1 : plage `DAT_144706100 - 0x200` (= 0x1dff, `DAT_144706100` = 0x1fff dans l'image) -> 13 bits ; 2 : 0x100 -> 8 ; 3 : 0x100 -> 8 ; 4 : 0x200 -> 9 ; 5 : 0x100 -> 8 ; 6 : 0x200 -> 9 ; 7 et 8 : 0x1fff -> 13 ; chaque poignée porte 2 bits de queue (génération) ; la catégorie 1 lit un bit de sonde et bascule sur l'entrée 4 (9 bits) s'il vaut 1. Identique au témoin (`handle()`) et à `varwidth.go`. Réserve : `DAT_144706100` / `DAT_1451f98d4` sont réécrits à la croissance de la table d'entités (cf. point 2), non élucidé pour un film.

### Portée du constat (DÉDUIT)
L'exécutable ouvert est celui qui joue la majeure 0x29 (HI_1_13_0, `FUN_1428e219c`). Sa table n'a AUCUN écart avec notre code : on ne peut donc pas imputer la divergence mesurée sur HI_1_12_0 (genres 82, 0, 15, 9, 36 ; début de vue B +48 / +64 / +84) à une table de poignées de CE build. Pour HI_1_12_0, l'exécutable ouvert ne permet pas de conclure : un film 12_0 a été écrit par l'écrivain de CE build-là (majeure 0x28, non jouée par cet exécutable), dont la table des poignées (ou les charges par genre) a pu différer ; seul un second exécutable (12_0) trancherait. Ce que je peux dire : si l écart 12_0 vient de la table des poignées, ce n'est pas un défaut de recopie de la table du build courant.

VERDICT V : confirme notre code actuel (pour ce build : 123 genres sur 123 identiques au jeu, au témoin et au dépôt). Pour HI_1_12_0 : indécidable avec cet exécutable.

---

## Point 6 - Fiche K : vue de contrôle (vue C)

Adresses lues : `FUN_1406cf548` (boucle), `FUN_1406d0388` (kind 0), `FUN_142f29b38` (kind 1), `FUN_142f29e54` (kind 2), `FUN_1406cdc04`, `FUN_142f2972c`.

### Boucle (LU)
```
tant que R(1) == 1 :                    // bit de continuation (0 -> fin, code 0)
    si compteur >= plafond : code 3
    kind = R(2)
    kind 0 -> FUN_1406d0388 ; kind 1 -> FUN_142f29b38 ; kind 2 -> FUN_142f29e54
    kind 3 -> AUCUN handler : 0 bit lu, la boucle enchaîne (iVar5 reste à 0)
```
`FUN_1406cdc04` : `R(1)` ; si 1 : `R(7)` ; sinon rend 0xff.

### Kinds 0, 1, 2 (LU)
- kind 0 (`FUN_1406d0388`) : 1re opération = `FUN_1406cdc04` (R(1) [+ R(7)]), puis `R(5)` (index du joueur 0..31) ; la suite (blocs a et b) est déjà portée par le dépôt.
- kind 1 (`FUN_142f29b38`) : 1re opération = `FUN_1406cdc04` ; `FUN_1406d0f0c(*param_2, ref)` ; puis `R(5)` (index, garde `> 0x1f`).
- kind 2 (`FUN_142f29e54`) : 1re opération = `FUN_1406cdc04` ; puis `FUN_142f2972c` : `R(1)` ; si 1 un bloc de longueur variable (`FUN_1406d0788(…, 0xb, 2)`) ; puis `R(5)` et `R(8)`.
Donc OUI : après les 2 bits de `kind`, un drapeau suivi de 7 bits (référence) est lu AVANT l'index du joueur pour les kinds 0, 1 et 2 (pour le kind 2, d'autres lectures, un drapeau et un bloc éventuel, s'intercalent entre ce couple et le `R(5)`). Et le kind 3 lit ZÉRO bit, comme le porte déjà `kindVueCNeant` de `frame_vue_controle.go:35`. Notre `lireEntreeDeControle` (kind 0) lit bien `R(1)`, `R(7)` si 1, puis `R(5)`.

VERDICT K : confirme la fiche (drapeau + 7 bits avant l'index pour tous les kinds portés par un handler, kind 3 = 0 bit) ; notre code actuel est conforme pour le kind 0.

---

## Fiche C - ce que j'ai croisé en passant (non traité, sous le quart de temps)

- Garde de pleine précision : `FUN_14076f91c` = (`DAT_144e61ea0 != 0` ou `DAT_145121140 == 1`). `DAT_144e61ea0` est posé à 1 autour des lectures de CRÉATION des entrées du tableau d'entités du morceau (point 4) : les positions y sont en R(96). `DAT_145121140` n'a que des lectures (2 pages d'xrefs de lecteurs) ; sa valeur de film n'est pas établie ici.
- Chemin NEW par enregistrement (`FUN_1408f1aa4`) : type = `R(6)`, lecteur de création = vtable+0x60 de l'archétype avec `(taille, tampon, lecteur, 1)` ; je n'y ai pas vu de pose de la garde.
- `flock-destination` (ti=21 composants 2 à 11, `FUN_140fb8af0`) : lit `R(1)`, `FUN_14076f91c ? R(96) : FUN_14076e524(0x10)`, puis `R(2)` si version > 1 : même lacune de table par index que la fiche P (notre `consumeFlockDestination` utilise l'ancien lecteur au niveau du registre). Probable contribuant à « ti=21 c.2 (create) 48 paquets » de la fiche C.
- ti=3, 47, 13, 10, 4, 22 : non relus.

## Choses que je n'ai PAS établies

- Statique seulement : la valeur effective de `DAT_144706100` / `DAT_1451f98d4` (largeur des poignées de catégories 0, 1, 7, 8) en lecture d'un film. L'image donne 0x1fff, mais `FUN_1408f1618` et `FUN_142f2f0cc` la réécrivent à la croissance de la table d'entités.
- La borne exacte écrite par l'écrivain de la table d'entités (point 4), et la valeur de `DAT_144dbfc90` côté lecteur au moment du morceau.
- La valeur de `DAT_145121140` pour un film (garde de pleine précision, alternative à `DAT_144e61ea0`).
- Que la suppression n'efface jamais `entrée[0]` (point 3) : seulement vérifié dans `FUN_1408f0d58`, `FUN_1408f0e98`, `FUN_1408f1244`.
