# Le profil par build du decodeur de film : 186 explique, les builds anciens mesures, le lecteur de roster repare

Date : 2026-09-12, phase 4. Suite de `NOTE_SECTION3_CHUNK00_2026-09-12.md` (phase 1),
`NOTE_SECTION3_SLOTS_2026-09-12.md` (phase 2) et `NOTE_EQUIPE_FILM_2026-09-12.md` (phase 3),
dont elle ferme les trois questions restees ouvertes. Travail **hors ligne, lecture seule** :
desassemblage statique de `HaloInfinite.exe` (Ghidra, instance partagee, aucun renommage, aucune
sauvegarde, aucune analyse relancee) + mesures sur les `chunk_00` et les paquets de type 2 du
cache local. Aucun code de production modifie, aucun commit.

Instruments (tous sous garde d'environnement, sautes en CI) :

| fichier | ce qu'il mesure |
|---|---|
| `apps/go-api/internal/analysis/filmdec/profil_entete_research_test.go` | la decomposition d'un record ti=9 par le modele du DEPOT, et pourquoi il ne peut pas atteindre 186 |
| `apps/go-api/internal/analysis/filmdec/profil_fermeture_research_test.go` | la DERIVATION de 186 par le lecteur d'etat complet, avec ses controles de tailles de tampon |
| `apps/go-api/internal/analysis/filmdec/profil_builds_research_test.go` | la table de profil par build, et l'oracle d'equipe rejoue sur les builds anciens |
| `apps/go-api/internal/analysis/filmdec/profil_roster_research_test.go` | le diagnostic differentiel du lecteur de la table des slots, sa correction et sa preuve |
| `apps/go-api/internal/analysis/filmdec/profil_roster_bilan_research_test.go` | la sonde ciblee par XUID et la comptabilite complete de la table |

---

## Resume execute

1. **186 EST EXPLIQUE, ET LA SOMME FERME SANS AUCUN AJUSTEMENT.**
   `186 = 108 + 32 + 14 + 32`, toutes largeurs lues dans l'executable : en-tete par entite du
   lecteur d'ETAT COMPLET (`FUN_142e2bfd0`), mot de taille `n1`, etat par defaut de ti=9
   (`FUN_1410d7540`, prefixe de version a zero), mot de taille `n2`. Le premier composant suit
   immediatement : **la boucle d'etat complet n'a AUCUN masque de presence**. Mesure :
   **186 sur 2 424 records ti=9 sur 2 424**, 6 films, 2 builds.
2. **LE MODELE DU DEPOT POUR LA TABLE D'IMAGE-CLE EST REFUTE, ET L'ECART EST CHIFFRE.**
   `TraverseEntity` lit un record NEW (en-tete de 64 bits, etat par defaut, MASQUE, composants).
   Sa borne superieure arithmetique est `64 + 22 + 65 = 151` : **35 bits trop court pour 186**,
   quelle que soit la donnee. Mesure : le modele place i0 a **82 sur 80 records sur 80**, et le
   masque relu a cette position rend la forme absurde « aucun composant present » (compte = 0 sur
   80/80). **La table d'image-cle n'est pas ecrite par le lecteur de record NEW.**
3. **LE DECALAGE 186 SE DERIVE A L'IDENTIQUE SUR CINQ BUILDS** (`HI_1_4_1`, `HI_1_8_0`,
   `HI_1_10_0`, `HI_1_11_0`, `HI_1_13_0`). La grammaire de la TRAME est donc STABLE la ou celle
   de `chunk_00` ne l'est pas (transposition de -4 320 a +1 600 bits selon le build). Ce qui
   bouge dans la trame est `n2` (88 contre 136) et la longueur du record (459 contre 460).
   Oracle externe rejoue sur les builds anciens : **9 films sur 10 en accord TOTAL avec
   `match_participants.team_id`, 168 slots sur 176**, dont six Grandes batailles a **24/24**, et
   **0 touche sur 320 decalages voisins** — le dixieme etant le temoin negatif FFA.
4. **LE LECTEUR DE LA TABLE DES SLOTS EST REPARE, ET LA CAUSE TIENT EN UN CHAMP.** Le balayage
   exigeait le champ de 2 bits de `slot+0x08` NUL ; `FUN_1407ecb08` l'ecrit comme un octet
   SIGNE sur 2 bits, et la sonde l'a mesure a **1** sur des enregistrements reels. Un
   enregistrement invisible DOUBLE l'ecart au voisin, ce qui faisait perdre au regroupement
   toute la tete de la table : `1c4c63c2` rendait **11** enregistrements au lieu de 24,
   `a26dbcdb` en rendait **3**. Apres correction, sur les **86 films** du cache a plus de
   16 joueurs : **1 890 ecarts sur 1 892 ferment** a une seule constante par film, et l'oracle
   d'equipe passe de 4 a **9 films sur 10** en accord total.
5. **DEUX FERMETURES ARITHMETIQUES GRATUITES CONFIRMENT L'EN-TETE DE 108 BITS.** Les deux mots
   de taille ne sont pas des nombres quelconques : `n1 = 12` sur **2 424 records sur 2 424**,
   soit exactement la taille de la structure que `FUN_1410d7540` remplit (`uint` a `+0`, `uint`
   a `+4`, `bool` a `+8`) ; et `n2 = 136 = 0x88` sur les films du build courant, soit exactement
   le `memset(dst, 0, 0x88)` de `vtable[0x88] = FUN_141071a58`. **Un en-tete mal dimensionne
   lirait deux mots de 32 bits quelconques** : le controle est fort, il est gratuit, et il passe.

---

## A. La derivation de 186 — la chaine, terme a terme

### A.1 Ce que le modele du depot dit, et pourquoi il ne peut pas marcher

Le depot lit un record d'image-cle comme un **record NEW** : `keyframe_record_walk.go` lit un
en-tete de 64 bits `[id:32][field:26][ti:6]` puis appelle `TraverseEntity`
(`traverse.go:1082`), qui joue l'etat par defaut de l'archetype, un bit de porte, le **masque de
presence**, puis les composants presents.

Les trois largeurs ont ete relues dans l'executable le 2026-09-12 :

| terme | adresse | largeur lue | preuve |
|---|---|---|---|
| etat par defaut de ti=9 | `FUN_1410d7540` (`vtable+0x60`) | **14 ou 22 bits** | `FUN_1406cf008` = R(1) (`*(param+0x2c) += 1`) ; si 1 -> R(8) ; puis R(6), R(6), R(1) |
| masque de presence | `FUN_1406d7610` | **4 a 46, ou 65** | R(1) ; si le bit vaut 1 -> R(64) ; sinon R(3) = compte, puis compte x R(6) |
| premier composant | `FUN_140f581e8` -> `FUN_1407ef804` | 4 bits | `ADD dword ptr [RCX+0x2c],0x4` |

D'ou la borne : `64 + 22 + 65 = 151`. **186 est 35 bits au-dela**, et aucune donnee ne peut
combler l'ecart — c'est de l'arithmetique, pas une mesure. `TestProfilEnteteTI9Bornes` publie les
trois candidats du dossier :

| en-tete candidate | i0 atteignable | 186 |
|---|---|---|
| 47 (fork chasewoodhams) | `[65, 134]` | **au-dessus de la borne de 52 bits** |
| 64 (`keyframeHeaderBits`) | `[82, 151]` | **au-dessus de la borne de 35 bits** |
| 108 (`keyframeFullStateHeaderBits`) | `[126, 195]` | **ATTEIGNABLE** |

Et la mesure le confirme sur les films (`TestProfilEnteteTI9Derivation`, 80 records ti=9 de
6 films) : le bit de version de l'etat par defaut vaut **0 sur 80/80** (donc 14 bits), le modele
place i0 a **82 sur 80/80**, l'ecart a 186 vaut **104 sur 80/80**, et `TraverseEntity` ne place
JAMAIS de composant 0 (`i0reel = -1` sur 80/80) — parce que le masque relu a la position 78 dit
« compte = 0 », c'est-a-dire aucun composant present, ce qui n'est pas une lecture plausible pour
un record de 459 bits.

### A.2 Le vrai lecteur, et la somme qui ferme

La table d'image-cle est lue par `FUN_142e2bfd0`, le lecteur d'**ETAT COMPLET** dont le depot
porte deja la forme depuis le lot R7-d (`keyframe_fullstate_loop.go`) **sans l'avoir jamais
confrontee a une position de champ**. Relecture du 2026-09-12 :

```
FUN_142e2bfd0 — en-tete PAR ENTITE, 108 bits :
    R(32)  id                      -> puVar12[0]
    R(32)  typeIndex               -> puVar12[1]   (teste != 0xffffffff)
    R(32)                          -> puVar12[3]
    R(4)   FUN_142e29cf8           (verifie : `*(param_1+0x2c) += 4`)
    R(8)                           -> *(puVar12+9)
  puis, si typeIndex != 0xffffffff :
    R(32)  n1     si n1 > 0 : (**(vtable+0x60))(..., param_1, 0)   = L'ETAT PAR DEFAUT
    R(32)  controle   UNIQUEMENT si FUN_14076cea8() est vrai
    R(32)  n2     si n2 > 0 : (**(vtable+0x88))(...)  puis FUN_1428e2b68 -> FUN_142e2c690
                  FUN_142e2c690 = LA BOUCLE des 64 entrees nommees, SANS masque de presence
```

La vtable de ti=9 a ete relue **octet a octet** a `0x1436fff28` (trouvee par les references
croisees sur `0x1410d7540`, puis verifiee : `vtable+0x60 = 0x1410d7540`, la valeur que
`KEYFRAME_ARCHETYPE_DEFAULTSTATE_TABLE.md` donne pour ti=9). Les deux autres slots appeles entre
l'etat par defaut et la boucle ne consomment **aucun bit**, et c'est lu, pas suppose :

| slot | adresse | contenu | bits |
|---|---|---|---|
| `vtable+0x30` | `0x14111fd9c` | `XORPS XMM0,XMM0 ; MOV RAX,RDX ; MOVUPS [RDX],XMM0 ; MOVUPS [RDX+0x10],XMM0 ; MOV dword [RDX],1 ; RET` | **0** |
| `vtable+0x88` | `0x141071a58` | `memset(dst,0,0x88)` puis des constantes ; ne recoit PAS le lecteur de bits | **0** |

**LA SOMME :**

```
  108   en-tete par entite            FUN_142e2bfd0
+  32   n1                            R(32), teste > 0
+  14   etat par defaut de ti=9       FUN_1410d7540, prefixe de version a 0 : R(1)=0 ; R(6) ; R(6) ; R(1)
+  32   n2                            R(32), teste > 0
= 186   premier composant             managed-player-team-designator-component, R(4)
```

**Aucun ajustement.** Mesure `TestProfilFermeture186` : **186 sur 2 424 records ti=9 sur 2 424**,
sur les 6 films du lot (14 bits d'etat par defaut sur 2 424/2 424).

### A.3 Le mot de controle est absent, et c'etait une prediction

`FUN_14076cea8` lit un drapeau RUNTIME (`DAT_144c23326` si `FUN_1404f2b4c()` est vrai, sinon
`DAT_1450e24e8`) : **indecidable statiquement**. S'il etait vrai, le mot de controle de 32 bits
serait ecrit entre l'etat par defaut et `n2`, et le premier composant tomberait a **218**.
L'oracle de la phase 3 a balaye les 456 decalages possibles et n'en a retenu **qu'un seul, 186**
— donc le drapeau est faux dans ces films. Le depot portait deja cette mesure **par une autre
voie** : `filmComponentCorruptionCheck` est a `false` par defaut dans `traverse.go`. Deux chemins
sans etape commune, meme reponse.

### A.4 Le controle qui ne coute rien : les deux mots de taille

Si l'en-tete de 108 bits est la bonne, `n1` et `n2` ne sont pas des nombres quelconques : ce sont
les deux TAILLES DE TAMPON que le jeu alloue pour l'archetype (`vtable[0x20]` et `vtable[0x10]`
dans `FUN_1408f1aa4`). Elles doivent donc etre **constantes par archetype**. Un en-tete decale
d'un seul bit lirait deux mots quelconques.

| mesure | valeur |
|---|---|
| `n1` sur les records ti=9 | **12**, sur **2 424 / 2 424** |
| taille de la structure remplie par `FUN_1410d7540` | `uint @ +0`, `uint @ +4`, `bool @ +8` -> **12 octets** |
| `n2` sur les films du build courant (`HI_1_13_0`) | **136 = 0x88**, sur 584/584 |
| `memset` de `vtable+0x88` (`FUN_141071a58`) | `memset(dst, 0, **0x88**)` |

**Deux fermetures independantes, sans ajustement.** Et `n2` vaut **88** sur les films de
`HI_1_12_0` : c'est une **donnee de profil par build**, pas un bruit (section B).

### A.5 La generalite : la meme derivation sur tous les archetypes

`TestProfilFermetureTousArchetypes` applique la derivation a **TOUS** les records de la table
d'image-cle, sans rien savoir de ti=9. Sur les 6 films (95 000 records environ), **13 archetypes
sur 34 rendent `n1` ET `n2` constants** :

| ti | `n1` | `n2` | etat par defaut porte |
|---|---|---|---|
| 2, 4, 15, 18, 19, 25, 34, 45 | **0** | constant (2712, 2, 692, 896, 256, 1160, 1640, 28) | 0 bit (stubs) — et `n1 = 0` veut dire que le jeu ne lit PAS l'etat par defaut |
| 5 | **2** | **252** | 7 bits |
| 9 | **12** | 88 / 136 | 14 bits |
| 22 | **12** | **12** | 0 bit (stub) |
| 29 | **1** | **128** | 0 bit |
| 47 | **4** | variable | 0 bit |

Les 21 archetypes a `n2` non constant sont **exactement** ceux dont l'etat par defaut est de
largeur VARIABLE et non entierement resolue (ti=35 bipede : 62 largeurs distinctes ; ti=37, 42,
43 : plusieurs dizaines). C'est le comportement attendu : `n2` se lit APRES l'etat par defaut, et
une largeur d'etat par defaut fausse le fait atterrir sur des bits quelconques. **`n2` est donc,
gratuitement, un DETECTEUR de largeur d'etat par defaut fausse** — un oracle interne pour la suite
du chantier decodeur, qui n'existait pas avant.

### A.6 La donnee de profil, nommee

Pour l'objet `Keyframe KeyframeLayout` de `.ai/ARCHITECTURE_CIBLE_DECODEUR_FILM_2026-09-12.md`
section 5 (« largeur d'en-tete PAR TYPE d'entite »), la valeur a porter n'est pas un nombre unique
mais une **regle** :

```
debutDesComposants(ti) = 108 + 32 + largeurEtatParDefaut(ti) + 32       si n1 > 0
                       = 108 + 32 + 0                          + 32     si n1 == 0
                       (+ 32 si le drapeau de controle du build est actif)
```

soit **172 + largeurEtatParDefaut(ti)**. Pour ti=9 : `172 + 14 = 186`. La note de la phase 3
disait « un decodeur qui s'appuierait sur 186 sans calibrer devra le re-mesurer par build » :
c'est **resolu** — 186 se DERIVE, et ce qui varie par build est la largeur de l'etat par defaut,
pas l'en-tete.

---

## B. La table de profil par build

Corpus : les 10 films demandes — trois par build pour `HI_1_8_0` (`0a247154`, `1950c59b`,
`2ce58582`), `HI_1_10_0` (`084a804d`, `108a4e4c`, `111fa685`) et `HI_1_11_0` (`00ba2e1c`,
`06dfe6d9`, `0d0bc019`), plus le film unique `HI_1_4_1` (`a521164d`), et `000d5950` en temoin
`HI_1_13_0`. Instrument `profil_builds_research_test.go`.

### B.1 LA TABLE — une ligne par build, chaque valeur avec sa preuve

| build | films | i0 DERIVE | `n1` | `n2` | etat par defaut ti=9 | longueur du record ti=9 | transposition du slot (`mesure - predit`) | lecteur canonique de `chunk_00` |
|---|---|---|---|---|---|---|---|---|
| `HI_1_4_1` | 1 | **186** | **12** | **88** | **14 bits** | **459** | **+1 600** | **1 / 24** |
| `HI_1_8_0` | 3 | **186** | **12** | **88** | **14 bits** | **459** | **-4 320** | **1 / 8** |
| `HI_1_10_0` | 3 | **186** | **12** | **88** | **14 bits** | **459** | **-2 880** | **1 / 24** |
| `HI_1_11_0` | 3 | **186** | **12** | **88** | **14 bits** | **459** | **-2 880** | **1 / 22-24** |
| `HI_1_13_0` | 1 | **186** | **12** | **136** | **14 bits** | **460** | **0** | **8 / 8** |

Preuve de chaque colonne : `i0 DERIVE` est la somme `108 + 32 + etat par defaut + 32` calculee
sur le flux du film (section A) ; `n1` et `n2` sont les deux mots de 32 bits lus aux positions
que cette somme designe ; l'etat par defaut est joue par le deserialiseur porte
(`consumeKeyframeDefaultState`), son bit de version etant lu dans le flux ; la longueur du record
est l'ecart au record ti=9 suivant ; la transposition est `(debut suivant - debut) - longueur
predite par la grammaire du slot`, mesuree sur les enregistrements a gamertag imprimable ; la
derniere colonne compare le lecteur canonique `s3sChaine` au balayage brut.

### B.2 CE QUI NE BOUGE PAS D'UN BUILD A L'AUTRE — et c'est le resultat principal

1. **Le decalage du designateur d'equipe vaut 186 sur les CINQ builds essayes**, et il n'est pas
   constate : il est **DERIVE** par la meme somme partout (`108 + 32 + 14 + 32`). La grammaire de
   la trame d'etat est donc **STABLE** la ou celle de `chunk_00` ne l'est pas.
2. **`n1 = 12` et l'etat par defaut de ti=9 fait 14 bits sur les cinq builds.** Le prefixe de
   version est a zero partout.
3. **Le champ lui-meme lit des equipes.** Les vecteurs bruts a 186 sont des partages nets :
   `[1 1 1 2 1 2 2 2]` en arene, et **12-12** sur toutes les Grandes batailles des builds
   anciens (`084a804d`, `108a4e4c`, `111fa685`, `00ba2e1c`, `0d0bc019`, `a521164d`).

### B.3 CE QUI BOUGE, ET DE COMBIEN

| ce qui change | `HI_1_4_1` a `HI_1_11_0` | `HI_1_13_0` |
|---|---|---|
| `n2` (taille du tampon d'etat de l'archetype ti=9) | **88** | **136 = 0x88** |
| longueur d'un record ti=9 | **459 bits** | **460 bits** |
| transposition de la grammaire du slot | -4 320 / -2 880 / +1 600 selon le build | 0 |

`n2` est la seule valeur de la trame qui bouge, et elle bouge **avec l'executable** : la valeur
`136` est exactement le `memset(dst, 0, 0x88)` de `vtable[0x88]` de l'exe courant. Les builds
anciens allouaient **88** octets pour le meme etat. Le bit de plus dans la longueur du record
(459 -> 460) est coherent : l'archetype `managed-player` a gagne de l'etat entre `HI_1_11_0` et
`HI_1_13_0`.

**Chaque difference est une ligne de profil** : un decodeur qui vise plusieurs builds doit
parametrer `n2` et la longueur du record, pas le decalage du designateur.

### B.4 L'ORACLE EXTERNE, REJOUE SUR LES BUILDS ANCIENS

`match_participants.team_id` lu en **lecture seule** dans la sauvegarde
`data/backups/pre-chaine-2026-09-09/shared_matches_v2.duckdb` (le fichier de production n'a pas
ete ouvert), passe a l'instrument comme un ENSEMBLE `xuid:equipe` — l'instrument apparie par
XUID, il ne recoit aucun ordre.

| film | build | mode | accord a `d = 186` |
|---|---|---|---|
| `0a247154` | `HI_1_8_0` | Ranked King of the Hill | **8 / 8** |
| `2ce58582` | `HI_1_8_0` | Ranked Strongholds | **8 / 8** |
| `084a804d` | `HI_1_10_0` | BTB Heavies CTF | **24 / 24** |
| `108a4e4c` | `HI_1_10_0` | BTB Fiesta Slayer | **24 / 24** |
| `1950c59b` | `HI_1_8_0` | **Arena FFA Slayer** | **0 / 8 — TEMOIN NEGATIF** : le film lit `[0 0 0 0 0 0 0 0]`, « aucune equipe », la base fabrique un camp par joueur |
| **total** | | | **4 / 5 films en accord TOTAL, 64 / 72 slots** |

Le temoin negatif se reproduit a l'identique sur `HI_1_8_0` : c'est le meme comportement que les
deux FFA de la phase 3, sur un build anterieur de plus d'un an. Deux Grandes batailles a **24/24**
sur `HI_1_10_0` ferment la question d'echelle sur les builds anciens.

**Cinq films restent a cardinaux differents** (`111fa685`, `00ba2e1c`, `06dfe6d9`, `0d0bc019`,
`a521164d`) : le lecteur de `chunk_00` y rend 21 a 23 rangs contre 24 a 25 entites ti=9. Ce n'est
pas un desaccord d'equipe, c'est la question 3 — traitee en section C, apres quoi ces cinq films
sont rejoues (section C.4).

### B.5 UNE DECOUVERTE DE LA MESURE : le lecteur canonique de la table des slots est MORT sur les builds anciens

La phase 2 disait « la grammaire du slot se transpose par une seule constante par build ». La
mesure d'ici ajoute une consequence qu'elle n'avait pas relevee : **le lecteur canonique
`s3sChaine`, qui avance du pas PREDIT, rend UN SEUL enregistrement sur les 9 films de build
ancien** (contre 8/8 sur `HI_1_13_0`). C'est mecanique — il avance d'un pas faux de 2 880 a
4 320 bits des le premier enregistrement — mais cela n'avait jamais ete mesure, et cela veut dire
que **toute lecture de la table des slots sur un film d'avant `HI_1_12_0` passe aujourd'hui par
le balayage brut**, pas par la grammaire. Le balayage, lui, tient : 8, 21 a 24 enregistrements
selon le film.

---

## C. Le lecteur de la table des slots sur les gros rosters — cause et correction

Le defaut est consigne deux fois sans etre traite : phase 2 section G.3, puis phase 3
section I.4 (« la cause n'est pas traitee », contournee par un repli). Instruments de ce lot :
`profil_roster_research_test.go` et `profil_roster_bilan_research_test.go`.

### C.1 LA CAUSE, isolee par une mesure differentielle puis designee par une sonde

Quatre causes candidates ont ete ecrites AVANT la mesure : `D1` l'en-tete exige
(`booleens 1/0/0`, `u32 nul`, `champ de 2 bits nul`), `D2` le filtre de `s3rGrappe`
(`xuid == borne basse`, `jeton nul`), `D3` la grappe terminale (seuil de 40 000 bits), `D4` la
plage de XUID. La mesure releve UN critere a la fois (`TestProfilRosterCause`) :

| film | attendu (entites ti=9) | origine | sans regroupement | sans booleens | sans `u32` nul | **sans 2 bits nuls** | sans jeton non nul |
|---|---|---|---|---|---|---|---|
| `1c4c63c2` | 24 | **11** | 24 | 49 | 26 | **24** | 11 |
| `111fa685` | 24 | **23** | 24 | 47 | 26 | **24** | 23 |
| `00ba2e1c` | 24 | 22 | 23 | 42 | 31 | 22 | 22 |

`D4` est ecarte : aucune valeur hors plage n'est en cause. `D2` est ecarte : lever le filtre du
jeton ne change rien. Lever les booleens ou le `u32` nul fait exploser le compte (39 a 57) —
ce sont des faux positifs, pas des enregistrements. **Un seul critere restitue exactement le
compte attendu : le champ de 2 bits.**

La SONDE le confirme nominativement (`TestProfilRosterXuidIntrouvable` : chaque XUID du roster
de la base est cherche dans le flux SANS aucune contrainte d'en-tete, puis les 85 bits qui le
precedent sont releves) :

```
1c4c63c2  xuid 2535450607961405  bit 11137668 : REJETE — b=1/0/0 u32=0 deux=1 token=ad00c8b3a1a3
111fa685  xuid 2535454874175468  bit 10812405 : REJETE — b=1/0/0 u32=0 deux=1 token=f160bab591ef
```

**Deux enregistrements parfaitement conformes sur tous les autres champs, rejetes par le seul
`deux = 1`.** Et l'ecrivain donne raison a la mesure : `FUN_1407ecb08` ecrit `slot+0x08` comme un
**octet SIGNE sur 2 bits** — son domaine est `-2..1`, pas `{0}`. Exiger ce champ nul etait une
regularite observee sur des petits rosters, pas une contrainte du format.

### C.2 POURQUOI LE DEFAUT ETAIT CATASTROPHIQUE, ET PAS PROPORTIONNE

Un enregistrement invisible ne coute pas un enregistrement : il **double l'ecart** au voisin. Or
`s3rGrappe` remonte la grappe terminale tant que l'ecart reste sous 40 000 bits. Sur `1c4c63c2`,
l'ecart double vaut **50 442 bits** — mesure — et la remontee s'arrete la : **la table perd TOUTE
SA TETE, 11 enregistrements lus au lieu de 24.** C'est l'effet de levier qui explique les comptes
a 11 de la phase 3, la ou le nombre d'enregistrements reellement invisibles est de 1.

La sonde mesure aussi, sur les memes films, **14 XUID de la base ABSENTS du flux** (0 occurrence,
avec ou sans contrainte d'en-tete). Ceux-la ne sont pas un defaut de lecture : `chunk_00` porte le
roster **a l'instant ou le chunk est ecrit**, et un joueur arrive en cours de partie n'y est pas.
C'est la raison, mesuree, pour laquelle `match_participants` ne peut pas servir d'oracle de compte.

### C.3 LA CORRECTION, ET SA PREUVE

La correction de l'instrument tient en une ligne (`profilRosterCritCorrigee` : le critere
`deux == 0` est retire), plus le filtre de parasite deja etabli par la phase 2 (une position
parasite est un motif d'en-tete fortuit A L'INTERIEUR d'un enregistrement, reconnaissable a son
champ de nom non imprimable). **Le seuil de regroupement n'est PAS touche** : une fois le critere
corrige, plus aucun ecart intra-table ne le depasse.

La preuve `P1` est un **oracle 100 % interne**, et c'est celui de la phase 2 : **la grammaire
predit l'ecart**. Chaque ecart doit valoir la longueur PREDITE de l'enregistrement plus UNE SEULE
constante par film (celle du build). Un enregistrement saute ou une position parasite casse cette
egalite ; la mesure ne depend d'aucune base.

| corpus | films | ecarts fermes a une seule constante | enregistrements lus |
|---|---|---|---|
| les 6 Grandes batailles de la phase 3 + 5 temoins | 11 | **entre 7/7 et 23/23 — 11 films sur 11 sans trou ni parasite** | 8 a 24 |
| **tous les films du cache a plus de 16 joueurs en base** | **86** | **1 890 / 1 892** | **1 978** (contre 1 851 avant) |

**1 890 ecarts sur 1 892 ferment**, soit **2 ecarts aberrants** sur 2 films (`b1bcbe24`,
`1c5c10cc`), publies comme residu en C.5. Aucun film ne depasse la borne de 32 que l'ecrivain
impose (`0x28A00 / 0x1450`) : le maximum lu est **24**.

Le gain, film par film, n'est pas marginal :

| film | avant | apres | entites ti=9 |
|---|---|---|---|
| `a26dbcdb` | **3** | **24** | 24 |
| `d8c7d880` | **8** | **24** | 24 |
| `1c4c63c2` | **11** | **24** | 24 |
| `b4035a67` | 11 | 24 | 24 |
| `443426df` | 12 | 24 | 24 |
| `93fcc545` | 14 | 24 | 24 |
| `16c3cdbd` | 15 | 24 | 24 |

**22 films gagnent des enregistrements, 13 en perdent** (ce sont les parasites que le filtre de
nom retire), 51 sont inchanges.

### C.4 CE QUE LA CORRECTION DEBLOQUE : l'oracle d'equipe ferme sur les gros rosters

La phase 3 avait du compter a part quatre Grandes batailles « pour cardinaux differents ». Avec
l'ordre des rangs pris dans le lecteur CORRIGE (`TestProfilBuildsOracleCorrige`, l'ordre restant
celui du flux — positions de bit croissantes, rien n'est reordonne) :

| film | build | accord a `d = 186` | voisins |
|---|---|---|---|
| `1c4c63c2` | `HI_1_10_0` | **24 / 24** (etait « compte a part ») | 0/32 |
| `111fa685` | `HI_1_10_0` | **24 / 24** (etait « compte a part ») | 0/32 |
| `03af54c3` | sans identification | **24 / 24** | 0/32 |
| `213a87dc` | `HI_1_10_0` | **24 / 24** | 0/32 |
| `084a804d`, `108a4e4c` | `HI_1_10_0` | **24 / 24** | 0/32 |
| `0a247154`, `2ce58582` | `HI_1_8_0` | **8 / 8** | 0/32 |
| `000d5950` | `HI_1_13_0` | **8 / 8** | 0/32 |
| `1950c59b` | `HI_1_8_0` | **0 / 8 — temoin negatif FFA** | 0/32 |
| **total** | | **9 / 10 films en accord TOTAL, 168 / 176 slots** | **0 touche sur 320** |

### C.5 LE RESIDU, PUBLIE

1. **Deux ecarts aberrants sur 1 892** (`b1bcbe24` : un dernier ecart de 38 852 bits ;
   `1c5c10cc` : un ecart de 27 382 bits). Non tranches — 0,1 % des ecarts, sur 2 films des 86.
2. **Le compte lu vaut le compte d'entites ti=9 sur 59 films sur 86**, et l'ecart tient dans
   `[-1, +1]` sur **75 sur 86** (dont 16 a `-1`), dans `[-5, +4]` sur les 86. **Ce n'est pas une
   erreur de lecture** : `P1` ferme sur ces films, donc la table est lue entierement. Les deux
   comptes mesurent deux choses differentes — la table est le roster a l'ecriture de `chunk_00`,
   la trame compte les joueurs PRESENTS a l'image-cle. Le bilan
   (`TestProfilRosterBilan`) ventile : sur 11 films, **239 XUID lus connus de la base, 6 lus
   inconnus d'elle, 23 de la base absents du flux**.
3. **`HI_1_11_0` n'est ferme que par la voie INTERNE, et il faut le dire.** Les trois films de
   ce build (`00ba2e1c`, `06dfe6d9`, `0d0bc019`) restent a cardinaux differents meme apres
   correction — leur table porte 21 a 24 enregistrements pour 24 a 25 entites ti=9, et la sonde
   montre que les XUID manquants sont ABSENTS du flux. L'oracle EXTERNE n'a donc jamais pu y
   etre applique terme a terme. Ce qui tient sur ce build : la derivation de 186 (identique aux
   autres), `n1 = 12`, `n2 = 88`, l'etat par defaut a 14 bits, et le partage **12-12** lu a 186
   sur les trois films. C'est une chaine sur deux, pas deux.
4. **La reattribution de slot en cours de match** (phase 3, C.6) n'a PAS ete retrouvee dans
   `chunk_00` : la table est ecrite une fois, ses enregistrements sont contigus et fermes par la
   grammaire, et aucun film du lot n'y montre de trou une fois le critere corrige. La
   reattribution est un phenomene de la TRAME, pas de la table. La consequence pratique de la
   phase 3 reste entiere : **un decodeur lit le designateur par SLOT, pas par rang.**

### C.6 Le controle d'echelle : la distribution des comptes devient lisible

`TestProfilRosterCorpus` sur **400 films** du cache (lecture de `chunk_00` seule) :

| compte d'enregistrements | 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | 10 | 11 | 12 | 15 | 16 | 19 | 21 | 22 | 23 | 24 | 25 |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| **AVANT** | 8 | 2 | 4 | 8 | 1 | 7 | 18 | **299** | 23 | 1 | 1 | 1 | 1 | 1 | 1 | 1 | 2 | 5 | 14 | 2 |
| **APRES** | 2 | — | — | 2 | — | 1 | 14 | **351** | — | — | — | — | — | 1 | 1 | 1 | 1 | 4 | **22** | — |

Les comptes de 9 a 15 disparaissent completement (ils valaient 27 films : c'etaient des
parasites), les arenes se concentrent sur **8** (299 vers 351) et les Grandes batailles sur **24**
(14 vers 22). **0 film au-dela de la borne de 32** que l'ecrivain impose.

---

## D. Ce qui est PROUVE, ce qui est HYPOTHESE, ce qui est REFUTE

### Prouve (deux chaines sans etape commune, ou fermeture arithmetique)

1. **`186 = 108 + 32 + 14 + 32`** : en-tete par entite de `FUN_142e2bfd0`, mot de taille `n1`,
   etat par defaut de ti=9 (`FUN_1410d7540`), mot de taille `n2`. Toutes largeurs lues dans
   l'executable ; somme verifiee sur **2 424 records ti=9 sur 2 424**.
2. **La table d'image-cle est ecrite par le lecteur d'ETAT COMPLET, et sa boucle de composants
   n'a AUCUN masque de presence.** Chaine 1 : le desassemblage. Chaine 2 : l'arithmetique du
   modele NEW plafonne a 151.
3. **`n1` et `n2` sont les deux tailles de tampon de l'archetype** : `n1 = 12` = la structure de
   `FUN_1410d7540` ; `n2 = 136 = 0x88` = le `memset` de `vtable[0x88]`. Deux fermetures
   independantes.
4. **`vtable+0x30` et `vtable+0x88` de ti=9 consomment 0 bit** (octets relus / decompile).
5. **Le decalage 186 se DERIVE a l'identique sur cinq builds** (`HI_1_4_1`, `HI_1_8_0`,
   `HI_1_10_0`, `HI_1_11_0`, `HI_1_13_0`) : l'etat par defaut de ti=9 fait 14 bits partout et
   `n1 = 12` partout.
6. **L'equipe se lit a 186 sur les builds anciens** : **9 films sur 10 en accord TOTAL avec
   `match_participants.team_id`, 168 slots sur 176**, dont **six Grandes batailles a 24/24**, et
   **0 touche sur 320 decalages voisins**.
7. **Le temoin negatif FFA se reproduit sur `HI_1_8_0`** (`1950c59b` : `[0 0 0 0 0 0 0 0]`).
8. **La cause du defaut du lecteur de roster est le critere `slot+0x08 == 0`** : le champ est un
   octet SIGNE sur 2 bits (ecrivain `FUN_1407ecb08`) et vaut **1** sur des enregistrements
   reels, nommes, avec leur XUID et leur gamertag.
9. **La correction ferme la grammaire** : **1 890 ecarts sur 1 892** valent la longueur predite
   plus une seule constante par film, sur les **86 films** du cache a plus de 16 joueurs en base.
10. **Le lecteur canonique `s3sChaine` rend UN SEUL enregistrement sur tous les builds anterieurs
    a `HI_1_12_0`** (mesure sur 9 films) : la lecture de la table sur un film ancien passe
    aujourd'hui par le balayage, pas par la grammaire.

### Hypothese (une seule chaine, ou non tranche)

1. **Le drapeau de controle par build.** `FUN_14076cea8` lit une globale runtime : la mesure dit
   qu'il est faux dans tous les films du cache (sinon i0 serait a 218), mais rien ne garantit
   qu'un build futur ne l'active pas. Un decodeur doit donc pouvoir lire les deux formes.
2. **Le sens de `n2 = 88` contre `136`.** `136` ferme exactement sur le `memset` de l'exe courant ;
   `88` est la valeur des builds anciens et n'a pas d'executable pour la confirmer. Lecture la
   plus economique : la taille du tampon d'etat de `managed-player` a grandi de 48 octets.
3. **Les 2 ecarts aberrants sur 1 892** (`b1bcbe24`, `1c5c10cc`) : non expliques.
4. **L'ecart de compte entre la table et la trame** (`-1` sur 16 films) : la lecture la plus
   economique est l'arrivee en cours de partie, appuyee par la sonde (les XUID manquants sont
   ABSENTS du flux), mais aucune mesure ne le prouve directement.

### Refute

1. **« Le record d'image-cle a un en-tete de 64 bits suivi d'un masque de presence. »** REFUTE
   pour la table d'image-cle : la borne est 151, le champ est a 186, et le masque relu a la
   position que ce modele designe rend « aucun composant present » sur 80 records sur 80.
2. **« L'en-tete par entite vaut 47 bits » (fork chasewoodhams).** REFUTE : 186 est 52 bits
   au-dela de la borne atteignable avec 47.
3. **« Il faudra re-mesurer 186 par build. »** (phase 3, ouvert n°1) REFUTE : la somme est la
   meme sur cinq builds, et ce qui bouge est `n2` et la longueur du record.
4. **« Le lecteur de roster perd des enregistrements a cause du seuil de regroupement. »** REFUTE
   comme cause PREMIERE : le seuil n'est qu'un amplificateur. La cause est le critere
   `slot+0x08 == 0`, et une fois celui-ci corrige le seuil n'est plus jamais atteint.
5. **« `match_participants` donne le nombre d'enregistrements attendu. »** REFUTE : **23 XUID de
   la base sont ABSENTS du flux** sur 11 films, avec ou sans contrainte d'en-tete.
6. **« La reattribution de slot se voit dans `chunk_00`. »** REFUTE sur ce corpus : la table est
   contigue et fermee par la grammaire, sans trou.

---

## E. Commandes pour rejouer

Les instruments sont sous garde d'environnement ; sans la variable ils sont sautes, donc la CI
reste verte. Les chemins sont **en style Windows** (`C:/...`) : un chemin de style Git Bash
(`/c/...`) fait echouer l'ouverture.

```bash
cd apps/go-api
C="C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks"

# A : la derivation de 186 — le modele du depot et pourquoi il ne ferme pas
F="000d5950 0014603f 02b172ab 03af54c3 213a87dc 1950c59b"
L=""; for f in $F; do L="$L;$C/$f"; done; L="${L#;}"
CHUNK00_FILMS="$L" go test ./internal/analysis/filmdec/ \
  -run 'TestProfilEnteteTI9' -v -timeout 30m

# A.2 / A.4 / A.5 : la FERMETURE a 186, les deux mots de taille, la generalite par archetype
CHUNK00_FILMS="$L" go test ./internal/analysis/filmdec/ \
  -run 'TestProfilFermeture' -v -timeout 30m

# B : la table de profil par build (les 10 films du lot + un temoin HI_1_13_0)
F="0a247154 1950c59b 2ce58582 084a804d 108a4e4c 111fa685 00ba2e1c 06dfe6d9 0d0bc019 a521164d 000d5950"
L=""; for f in $F; do L="$L;$C/$f"; done; L="${L#;}"
CHUNK00_FILMS="$L" go test ./internal/analysis/filmdec/ \
  -run 'TestProfilBuildsTrame' -v -timeout 40m

# B.4 / C.4 : l'oracle externe, avec le lecteur d'origine puis avec le lecteur CORRIGE
CHUNK00_FILMS="$L" CHUNK00_XUID_EQUIPES="$ORACLE" go test ./internal/analysis/filmdec/ \
  -run 'TestProfilBuildsOracle$|TestProfilBuildsOracleCorrige' -v -timeout 60m

# C.1 : le diagnostic differentiel et la sonde ciblee
F="00ba2e1c 036c102a 03af54c3 06a883f7 1c4c63c2 213a87dc 111fa685 06dfe6d9 0d0bc019 a521164d 000d5950"
L=""; for f in $F; do L="$L;$C/$f"; done; L="${L#;}"
CHUNK00_FILMS="$L" go test ./internal/analysis/filmdec/ \
  -run 'TestProfilRosterCause|TestProfilRosterEcarts' -v -timeout 40m
CHUNK00_FILMS="$L" CHUNK00_XUID_EQUIPES="$ORACLE" go test ./internal/analysis/filmdec/ \
  -run 'TestProfilRosterXuidIntrouvable|TestProfilRosterBilan' -v -timeout 40m

# C.3 : LA PREUVE (P1 fermeture de la grammaire, P2 cardinal)
CHUNK00_FILMS="$L" go test ./internal/analysis/filmdec/ \
  -run 'TestProfilRosterCorrige' -v -timeout 40m

# C.6 : le controle d'echelle (400 films, ~20 s)
CHUNK00_CORPUS="$C" CHUNK00_CORPUS_MAX=400 go test ./internal/analysis/filmdec/ \
  -run TestProfilRosterCorpus -v -timeout 60m
```

`CHUNK00_XUID_EQUIPES` est de la forme `<prefixe>=<xuid>:<equipe>,...`, blocs separes par `;`.
**L'ordre n'y est pas lu** : l'instrument apparie par XUID. Il se construit en **lecture seule**
depuis la sauvegarde (le fichier de production peut etre tenu par le serveur — dans ce cas ne pas
forcer) :

```sql
ATTACH 'data/backups/pre-chaine-2026-09-09/shared_matches_v2.duckdb' AS s (READ_ONLY);
SELECT substr(p.match_id,1,8) AS pref,
       string_agg(p.xuid || ':' || COALESCE(CAST(p.team_id AS VARCHAR),'NULL'), ',') AS roster
  FROM s.match_participants p GROUP BY 1;
```

La liste des films a plus de 16 joueurs se construit par intersection de
`SELECT substr(match_id,1,8) FROM match_participants GROUP BY 1 HAVING count(*) >= 17` avec les
repertoires de `film_chunks/` : **96 prefixes au cache, 86 exploitables** ; le controle C.3 y
tourne en 3 min 16 s.

Gates passes : `gofmt -l` net, `go vet ./internal/analysis/filmdec/` net,
`go test ./internal/analysis/filmdec/` sans garde **ok**, `go test ./internal/archlint/` **ok** —
le ratchet des variables de paquet de `filmdec` n'est pas touche (les instruments n'introduisent
que des `const`, des types et des fonctions de test).

---

## F. Decouvertes hors perimetre — notees, NON TRAITEES (regle 7)

1. **Le decalage de 8 octets de `parseRegistry`** : connu depuis la phase 1, **NON TRANCHE**, pas
   touche par ce lot.
2. **Le modele de record d'image-cle de la PRODUCTION est faux** (`keyframe_record_walk.go`,
   `TraverseEntity` appele avec un en-tete de 64 bits et un masque). C'est la cause de fond des
   deraillements constates par les lots R3, R4 et R5 (« le deserialiseur du corps d'un record
   d'image-cle n'est resolu nulle part »), et le dossier portait deja la bonne forme depuis R7-d
   (`keyframe_fullstate_loop.go`) sans l'avoir branchee. **Non traite : c'est du code de
   production, hors perimetre de ce lot.**
3. **`n2` est un DETECTEUR gratuit de largeur d'etat par defaut fausse.** Sur les 13 archetypes
   dont l'etat par defaut est de largeur fixe, `n1` et `n2` sont constants ; sur les 21 autres
   (ti=35 bipede, 37, 42, 43...), `n2` prend des dizaines de valeurs — parce qu'il est lu APRES
   l'etat par defaut. C'est un oracle interne pour la suite du chantier decodeur. **Non exploite
   ici.**
4. **La production ecrit `Team: -1` en dur** (`replay/build.go:577`) et documente « l'equipe n'est
   pas dans le film » (`replay/document.go:577`, `:365`). Refute depuis la phase 3, et ce lot
   ajoute que la lecture tient sur cinq builds. **Aucun code de production touche.**
5. **Le pied (type 3), `b37`/`b38` contre `b55`** : contradiction datee de l'archive, toujours
   non tranchee (phase 3, I.1).

---

## Ce qui reste ouvert

1. **Brancher la lecture en production.** Les trois pieces sont maintenant la : la derivation de
   l'offset (`172 + etat par defaut`), le lecteur de roster qui tient sur les gros rosters et sur
   les builds anciens, et le pont `rang -> xuid`. Il reste a ecrire le code de production, ce que
   ce lot n'a pas fait.
2. **Les 2 ecarts aberrants** de `b1bcbe24` et `1c5c10cc` (0,1 % des ecarts mesures).
3. **Le drapeau de controle par build** (`FUN_14076cea8`) : mesure faux partout, mais un decodeur
   robuste doit detecter les deux formes plutot que cabler 186.
4. **`HI_1_9_0` et `HI_1_12_0`** : le lot a essaye cinq builds sur sept sur la table de profil.
   `HI_1_9_0` (1 film au cache) n'a pas ete essaye sur la trame ; `HI_1_12_0` l'a ete sur la table
   des slots (`06a883f7`) mais pas dans la table de la section B.
5. **Le nom d'un designateur** : `brut - 1 = team_id` est mesure sur cinq builds ; relier
   `team_id 0` a `First` cote affichage demande une mesure de plus (phase 3, ouvert n°3).
