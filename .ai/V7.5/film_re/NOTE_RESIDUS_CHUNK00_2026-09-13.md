# Les residus de `chunk_00`, de la table des slots et du designateur d'equipe

Date : 2026-09-13, phase 5b. Ferme les residus laisses par les phases 1 a 4
(`NOTE_SECTION3_CHUNK00_2026-09-12.md`, `NOTE_SECTION3_SLOTS_2026-09-12.md`,
`NOTE_EQUIPE_FILM_2026-09-12.md`, `NOTE_PROFIL_PAR_BUILD_2026-09-12.md`). Travail **hors ligne,
lecture seule** : desassemblage statique de `HaloInfinite.exe` (Ghidra, instance partagee, aucun
renommage, aucune sauvegarde, aucune analyse relancee) + mesures sur les 1 351 `chunk_00` et les
paquets de type 2 et 3 du cache local. **Aucun code de production modifie, aucun commit.**

Instruments (tous sous garde d'environnement, sautes en CI) :

| fichier | ce qu'il mesure |
|---|---|
| `apps/go-api/internal/analysis/filmdec/residus_vacants_research_test.go` | residu 1 : les deux ecarts aberrants, le predicat de slot VACANT, la fermeture de la grammaire sur 96 films |
| `apps/go-api/internal/analysis/filmdec/residus_slots_research_test.go` | residu 2 : ou tombe la transposition par build, quel champ change de largeur, le lecteur par grammaire calibre sur les 7 builds |
| `apps/go-api/internal/analysis/filmdec/residus_trame_research_test.go` | residus 3 et 4 : `HI_1_9_0` sur la trame, la sentinelle du drapeau de controle, la cardinalite reelle des designateurs |
| `apps/go-api/internal/analysis/filmdec/residus_pied_research_test.go` | residu 6 : les 60 octets du bloc du pied confrontes a l'equipe PROUVEE, octet par octet |

---

## Resume execute

1. **RESIDU 1 FERME, ET IL EN FERME UN AUTRE AU PASSAGE.** Les deux ecarts aberrants de la
   phase 4 sont, l'un comme l'autre, **exactement un enregistrement de slot VACANT** que le
   balayage ne peut pas voir. La longueur d'un tel enregistrement se CALCULE depuis la grammaire
   (`1 299 + perso + 352 + 32`) et vaut **13 619 bits** sur `HI_1_10_0`/`HI_1_11_0` : c'est
   **au bit pres** le surplus mesure sur les deux films, `reste 0`. La question ouverte n°3 de la
   phase 1 (« la longueur exacte d'un enregistrement de slot VIDE », estimee entre 15 700 et
   17 900 bits) est donc fermee par le meme calcul. Fermeture generale : **2 121 ecarts sur
   2 121, 96 films sur 96, zero aberrant.**
2. **RESIDU 2 FERME, ET LE CHAMP EST NOMME.** La transposition par build (`-2 880`, `-4 320`,
   `+1 600` bits) est **entierement** portee par le bloc de personnalisation `sub+0xcc0`. La
   mesure coupe l'enregistrement en deux autour du bloc de queue (localisable par le gamertag en
   UTF-16 petit-boutiste) : la moitie APRES est invariante sur les sept builds, et la moitie
   AVANT une fois retiree sa plus longue suite de bits nuls vaut **270 bits sur TOUS les builds**.
   Le bloc nul fait 1 852 / 1 492 / 1 312 / 2 052 octets selon le build, et sa variation EST la
   transposition. Aucun ajustement.
3. **LE LECTEUR PAR GRAMMAIRE EST EXACT SUR LES SEPT BUILDS**, et il se calibre **sur le film**,
   sans table de build ecrite a la main. Controle d'echelle sur les **1 351 films du cache** :
   `rsChaine` rend exactement le compte du balayage corrige sur **1 351/1 351**, la ou
   `s3sChaine` etait MORT (≤ 1 enregistrement) sur **101 films**. Et le controle gratuit qui
   n'avait jamais ete fait : **enregistrements lus + slots vacants enjambes == 32 sur 1 351 films
   sur 1 351** — la borne `0x28A00 / 0x1450` de l'ecrivain, verifiee par le lecteur sur tout le
   corpus.
4. **RESIDU 3 FERME : `HI_1_9_0` rejoint la table de profil**, et sa ligne est identique a celle
   de `HI_1_8_0` sur la trame : etat par defaut de ti=9 a 14 bits, `n1 = 12`, `n2 = 88`, record de
   459 bits, **i0 DERIVE a 186**. Table des slots : transposition `-4 320`, 24 enregistrements
   + 8 vacants = 32. Oracle externe : **accord EXACT sur les 23 rangs apparies, a un seul
   recalage sur deux essayes.** `HI_1_12_0` est ajoute a la meme table (`n2 = 136`, record 460).
5. **RESIDU 4 FERME PAR UNE MESURE DIRECTE, ET LE MOT GATE EST UNE SENTINELLE.** L'ecrivain
   d'etat complet `FUN_142e2d08c` montre que le mot de 32 bits gate par le drapeau n'est pas un
   champ : c'est la constante **`0x0FFDDCBA`**. Cherchee a tout decalage de bit dans les paquets
   de type 2 de 10 films couvrant les 7 builds : **0 occurrence**. Temoin positif du meme
   instrument : a la position ou elle tomberait, on lit `n2` sur **4 368 records sur 4 368**. Et
   le critere interne d'equipe passe a `d = 186` sur 8 films sur 10, **a `d = 218` sur 0 sur 10**.
6. **RESIDU 5 : LA CORRESPONDANCE `team_id 0` <-> `First` EST PROUVEE PAR LE CONSOMMATEUR
   D'AFFICHAGE.** La fonction de script `AddTeamDesignatorStringIdsToList`
   (`FUN_142d417c0` -> `FUN_142d40c1c`) empile, dans cet ordre, les identifiants de chaine
   **`team_0`, `team_1`, ..., `team_7`, `team_neutral`** — neuf entrees, meme cardinal et meme
   ordre que l'enumeration `mp_team_designator` (`First`..`Eighth`, `Neutral`). La numerotation
   part donc de **0**, et le premier designateur nomme est `team_0` = `First`. Cote donnees :
   **aucun film du cache ne porte plus de deux designateurs** — les trois seuls matchs a plus de
   deux `team_id` en base (deux FFA a 8, un mode a 4) lisent dans le film une valeur UNIQUE
   (`0` = aucune equipe en FFA, `1` = designateur 0 dans le mode a 4). Les designateurs `Third`
   a `Eighth` ne sont exerces par aucun film.
7. **RESIDU 6 TRANCHE, ET LA PRODUCTION LIT LE MAUVAIS OCTET.** Balayage AVEUGLE des 60 octets
   du bloc du pied, trois lectures par octet, contre l'equipe PROUVEE de l'acteur (etablie dans
   le film seul) : **`b37` et `b38` valent l'equipe sur 665 evenements sur 665**, et `b55` — celui
   que la production lit sous le nom `teamRaw` — **vaut 0 sur les 665**, donc il coincide avec
   l'equipe 0 et se trompe sur l'equipe 1, exactement une fois sur deux. Plancher de bruit
   mesure : **4 lectures parfaites sur 180 essayees**, et ce sont les deux lectures de `b37` et
   les deux de `b38`. L'archive avait raison, la production a tort.

---

## 1. Les deux ecarts aberrants : un slot VACANT, au bit pres

### 1.1 Ce que la grammaire predit pour un enregistrement VIDE

La somme est **calculee**, pas ajustee : chaque terme est une largeur deja lue dans le code de
son ecrivain (phases 1 et 2), evaluee pour des champs tous nuls.

```
   85  en-tete du slot (1+1+1+32+2+48)       FUN_1407ecb08
+  64  le XUID, nul                          FUN_1406d6498
+  11  prefixe du masque (rang du bit haut)  FUN_1424ccf94, masque vide -> rang 0
+   1  le masque reduit a un bit             ce bit vaut 0
+  12  N, nul                                FUN_1411b1a24
+   8  M, nul                                FUN_1411b198c
+ 832  le bloc de 104 octets                 sub+0xc48
+  16  la chaine reduite a son NUL           FUN_1407ece18 s'arrete APRES l'unite nulle
+ 128  le bloc de 16 octets                  sub+0xc38
+  32  `desired-representation`              sub+0xcb0
+  64  le champ de 64 bits                   sub+0xcb8
+  46  les six champs courts (10+14+6+8+7+1)
= 1 299 bits, puis le bloc de personnalisation, le bloc de queue (352) et le u32 final (32).
```

D'ou `rsVide(delta) = 1 299 + 14 816 + delta + 352 + 32`. Sur `HI_1_10_0` / `HI_1_11_0`
(`delta = -2 880`) : **13 619 bits**.

### 1.2 La mesure, sur les deux films

| film | build | slot | ecart mesure | longueur predite | surplus | intervalle | bits non nuls | multiple de `rsVide` | en-tete + XUID Xbox dans l'intervalle | predicat VACANT |
|---|---|---|---|---|---|---|---|---|---|---|
| `b1bcbe24` | `HI_1_11_0` | 21 | 38 852 | 28 113 | **+13 619** | 13 619 bits | **1** | **1 x 13 619, reste 0** | **0** | **true** |
| `1c5c10cc` | `HI_1_10_0` | 15 | 27 382 | 16 643 | **+13 619** | 13 619 bits | **1** | **1 x 13 619, reste 0** | **0** | **true** |

Les deux surplus sont **identiques au bit pres**, et egaux a la longueur predite d'un
enregistrement vide sur leur build. La branche (b) — « un enregistrement de joueur rejete » — est
**eliminee par la mesure** et non par choix : l'intervalle ne contient aucun motif d'en-tete suivi
d'un entier de la plage des XUID Xbox.

**Le seul bit non nul de l'enregistrement vacant** est a **+1 282** de son debut, soit le dernier
bit du champ de 6 bits `sub+0xc35` : il y vaut brut `1`, donc la valeur signee **0**, la ou les
enregistrements occupes portent brut `0` = **-1**. C'est le seul champ qui distingue un slot
vacant d'un bloc de zeros.

**Pourquoi le balayage ne le voit pas, et pourquoi il coute cher.** Un enregistrement vacant est
doublement invisible : ses trois booleens de tete valent `0/0/0` (le balayage exige `1/0/0`) et
son XUID vaut 0 (hors de la plage Xbox). Il ne coute donc pas un enregistrement, il **DOUBLE
l'ecart** entre ses deux voisins — le meme effet de levier que le champ de 2 bits de la phase 4.

### 1.3 La fermeture generale

`TestResidusSlotFermeture` rejoue P1 de la phase 4 avec une seule tolerance, **verifiee** : un
ecart peut porter en plus un nombre ENTIER de slots vacants, et chacun doit satisfaire le predicat
grammatical `rsVacant` A SA POSITION CALCULEE.

| corpus | ecarts | fermes | films sans aucun ecart aberrant | slots vacants trouves et verifies |
|---|---|---|---|---|
| les 96 films du cache a ≥ 17 joueurs en base | **2 121** | **2 121** | **96 / 96** | **2** |

La phase 4 fermait 1 890 ecarts sur 1 892 (86 films) ; ce lot en ferme **2 121 sur 2 121** sur
96 films. **Zero aberrant reste.**

> Note de methode. La phase 4 partait du balayage d'ORIGINE dans son instrument de fermeture ;
> ce lot part du balayage CORRIGE (critere `slot+0x08` leve, filtre de parasite par le champ de
> nom). Sans cela on melange deux causes d'ecart double — le champ de 2 bits et le slot vacant —
> et le compte d'aberrants remonte a 16 sur 14 films. Le chiffre publie ici l'est avec le
> balayage corrige, qui est le seul coherent avec le verdict de la phase 4.

---

## 2. La transposition par build : c'est le bloc de personnalisation

### 2.1 La mesure, et pourquoi elle est possible sans executable de ces builds

L'ordre d'ecriture de `FUN_1407edea8` place **tous** les champs de longueur variable AVANT le
gamertag et **tous** les champs de largeur fixe APRES lui. Or le bloc de queue `sub+0x1400` porte
le MEME gamertag, en UTF-16 **petit-boutiste** (phase 2, A.4 : 640/640). Il est donc LOCALISABLE
dans le flux sans rien supposer de ce qui le precede : on cherche les deux premieres unites du
nom deja lu (32 bits). Cela coupe l'enregistrement en deux moities mesurables separement.

Colonnes du releve : `AVANT` = (position du nom petit-boutiste) - (fin du champ de gamertag) ;
`APRES` = (debut de l'enregistrement suivant) - (position du nom petit-boutiste) ; `nulle` = la
plus longue suite de bits nuls de la moitie AVANT ; `RESTE` = `AVANT - nulle - (384 - APRES)`.

| film | build | transposition | AVANT | APRES | plus longue suite nulle | = octets | **RESTE hors bloc nul** |
|---|---|---|---|---|---|---|---|
| `000d5950` | `HI_1_13_0` | 0 | 15 086 | **384** | 14 816 | **1 852** | **270** |
| `0014603f` | `HI_1_12_0` | 0 | 15 182 | **288** | 14 816 | **1 852** | **270** |
| `00ba2e1c` | `HI_1_11_0` | **-2 880** | 12 302 | **288** | 11 936 | **1 492** | **270** |
| `084a804d` | `HI_1_10_0` | **-2 880** | 12 302 | **288** | 11 936 | **1 492** | **270** |
| `11de8353` | `HI_1_9_0` | **-4 320** | 10 766 | **384** | 10 496 | **1 312** | **270** |
| `0a247154` | `HI_1_8_0` | **-4 320** | 10 766 | **384** | 10 496 | **1 312** | **270** |
| `a521164d` | `HI_1_4_1` | **+1 600** | 16 686 | **384** | 16 416 | **2 052** | **270** |
| `03af54c3` | (sans identification) | **+1 600** | 16 686 | **384** | 16 416 | **2 052** | **270** |

Trois faits se lisent, et aucun n'a ete cherche :

1. **La moitie APRES ne prend que deux valeurs, 384 et 288, et l'ecart vaut 96 bits = 12 octets**
   — exactement l'offset du nom DANS le bloc de queue que la phase 2 avait mesure par une tout
   autre voie (0 sur `HI_1_13_0`, 12 octets sur `HI_1_12_0`). **Le bloc de queue et le u32 final
   sont donc inchanges sur les sept builds.**
2. **La plus longue suite de bits nuls vaut EXACTEMENT `s3sPersoBits = 14 816` sur le build
   courant**, c'est-a-dire les 1 852 octets du bloc de personnalisation lu dans l'executable — et
   elle recule de la transposition, bit pour bit, sur chaque autre build.
3. **Le RESTE vaut 270 bits sur les huit groupes**, c'est-a-dire
   `128 (bloc de 16 o) + 32 (repr) + 64 (q64) + 46 (six champs courts) = 270`. **Aucun autre
   champ n'a change de largeur.**

Les valeurs `nulle` de `14 852` / `11 972` / `10 532` / `16 452` qui apparaissent sur une minorite
d'enregistrements valent la valeur modale **plus 36 bits** : ce sont les enregistrements dont
`f14`, `f6`, `f8`, `f7` et `f1` sont tous nuls, ce qui prolonge la suite de zeros vers l'amont de
`14+6+8+7+1 = 36`. Leur `RESTE` vaut alors `270 - 36 = 234`. C'est un controle interne gratuit :
la somme reste exacte.

### 2.2 La largeur du bloc de personnalisation par build

| build | bloc `sub+0xcc0` | ecart au build courant | ferme sur le pas des attaches d'armure (`0x24` = 36 o) ? |
|---|---|---|---|
| `HI_1_12_0`, `HI_1_13_0` | **1 852 o** (`0x73C`) | 0 | reference : `0x738 (actionPose) + 4 = 0x73C`, 24 attaches |
| `HI_1_10_0`, `HI_1_11_0` | **1 492 o** | **-360** | **oui : 360 = 10 x 0x24** -> 14 attaches |
| `HI_1_8_0`, `HI_1_9_0` | **1 312 o** | **-540** | **oui : 540 = 15 x 0x24** -> 9 attaches |
| `HI_1_4_1`, sans identification | **2 052 o** | **+200** | **non** : 200 n'est multiple ni de `0x24` (36) ni de `0x58` (88) |

**Mesure** : la largeur du bloc par build. **Hypothese, appuyee mais non prouvee** : deux des
trois ecarts tombent exactement sur le pas des 24 attaches d'armure de `FUN_1407ebf44`, ce qui se
lit comme « le systeme de tenues a gagne 5 puis 10 attaches ». Le quatrieme (`+200` octets sur
`HI_1_4_1` et les 5 films sans section d'identification) **ne ferme sur aucun des deux pas
connus** : sur ce build la structure est plus grande, et la cause n'est pas etablie — il faudrait
un executable de ce build, qu'on n'a pas.

### 2.3 Le lecteur par grammaire, calibre sur le film

`rsChaine(d, delta)` est `s3sChaine` corrige de deux facons :

- le pas predit est corrige de la constante du build, **calibree sur le film** (`rsDelta` : la
  valeur modale de `mesure - predit` sur les ecarts du balayage corrige). Aucune table de build
  n'est ecrite a la main : la calibration vaut donc aussi pour un build inconnu — c'est
  exactement ce que montrent les 5 films sans section d'identification ;
- un slot **vacant** est enjambe de `rsVide(delta)` au lieu d'arreter le lecteur.

| film | build | delta | `s3sChaine` | `rsChaine` | balayage corrige | vacants enjambes |
|---|---|---|---|---|---|---|
| `000d5950` | `HI_1_13_0` | 0 | 8 | 8 | 8 | 24 |
| `0014603f` | `HI_1_12_0` | 0 | 8 | 8 | 8 | 24 |
| `00ba2e1c` | `HI_1_11_0` | -2 880 | **1** | **22** | 22 | 10 |
| `084a804d` | `HI_1_10_0` | -2 880 | **1** | **24** | 24 | 8 |
| `11de8353` | `HI_1_9_0` | -4 320 | **1** | **24** | 24 | 8 |
| `0a247154` | `HI_1_8_0` | -4 320 | **1** | **8** | 8 | 24 |
| `a521164d` | `HI_1_4_1` | +1 600 | **1** | **24** | 24 | 8 |
| `03af54c3` | (sans identification) | +1 600 | **1** | **24** | 24 | 8 |

**Controle d'echelle, sur les 1 351 films du cache** (`TestResidusSlotChaineCorpus`, 118 s) :

| build | films | delta(s) mesures | `s3sChaine` | `rsChaine` | balayage corrige | films au compte du balayage | films ou `s3sChaine` est MORT | vacants | **films ou lus + vacants == 32** |
|---|---|---|---|---|---|---|---|---|---|
| `HI_1_13_0` | 1 123 | `0` x1 123 | 8 596 | **9 003** | 9 003 | **1 123/1 123** | 19 | 26 933 | **1 123/1 123** |
| `HI_1_12_0` | 146 | `0` x146 | 1 312 | **1 347** | 1 347 | **146/146** | 0 | 3 325 | **146/146** |
| `HI_1_11_0` | 39 | `-2 880` x39 | 39 | **913** | 913 | **39/39** | 39 | 335 | **39/39** |
| `HI_1_10_0` | 26 | `-2 880` x26 | 26 | **576** | 576 | **26/26** | 26 | 256 | **26/26** |
| `HI_1_8_0` | 10 | `-4 320` x10 | 10 | **160** | 160 | **10/10** | 10 | 160 | **10/10** |
| (sans identification) | 5 | `+1 600` x5 | 5 | **120** | 120 | **5/5** | 5 | 40 | **5/5** |
| `HI_1_9_0` | 1 | `-4 320` x1 | 1 | **24** | 24 | **1/1** | 1 | 8 | **1/1** |
| `HI_1_4_1` | 1 | `+1 600` x1 | 1 | **24** | 24 | **1/1** | 1 | 8 | **1/1** |

Trois resultats :

1. **Le delta est UNIQUE par build sur tout le corpus** — aucune dispersion, 1 351 films.
2. **`rsChaine` egale le balayage corrige sur 1 351 films sur 1 351**, la ou `s3sChaine` etait
   mort sur **101** films (dont 19 du build courant : ce sont ceux qui portent un slot vacant
   avant la fin de leur roster).
3. **La derniere colonne est la fermeture la plus forte du lot** : le lecteur lit exactement
   **32 enregistrements** — occupes plus vacants — sur **1 351 films sur 1 351**, sans jamais
   s'arreter avant. C'est la borne `0x28A00 / 0x1450 = 32` de l'ecrivain, verifiee par le lecteur
   sur tout le cache et sur sept builds. Une seule largeur fausse la casserait.

---

## 3. `HI_1_9_0` sur la trame — la table de profil est complete

`11de8353` est le seul film de son build au cache. Le lot l'a passe a la derivation de 186 et a
l'oracle externe, et y a ajoute `HI_1_12_0` (absent de la table de la phase 4).

### 3.1 La table par build, completee

| build | films | i0 DERIVE | `n1` | `n2` | etat par defaut ti=9 | record ti=9 | transposition du slot | bloc de personnalisation |
|---|---|---|---|---|---|---|---|---|
| `HI_1_4_1` | 1 | **186** | 12 | **88** | 14 bits | **459** | **+1 600** | **2 052 o** |
| `HI_1_8_0` | 10 | **186** | 12 | **88** | 14 bits | **459** | **-4 320** | **1 312 o** |
| **`HI_1_9_0`** | **1** | **186** | **12** | **88** | **14 bits** | **459** | **-4 320** | **1 312 o** |
| `HI_1_10_0` | 26 | **186** | 12 | **88** | 14 bits | **459** | **-2 880** | **1 492 o** |
| `HI_1_11_0` | 39 | **186** | 12 | **88** | 14 bits | **459** | **-2 880** | **1 492 o** |
| **`HI_1_12_0`** | **146** | **186** | **12** | **136** | **14 bits** | **460** | **0** | **1 852 o** |
| `HI_1_13_0` | 1 123 | **186** | 12 | **136** | 14 bits | **460** | 0 | **1 852 o** |
| (sans identification) | 5 | — (non mesure sur la trame ici, sauf `03af54c3` : **186**) | 12 | 88 | 14 bits | 459 | **+1 600** | **2 052 o** |

Ce que la ligne `HI_1_12_0` ajoute : la bascule `n2 = 88 -> 136` et `record = 459 -> 460` se fait
**entre `HI_1_11_0` et `HI_1_12_0`**, au meme endroit que la bascule du bloc de personnalisation
(`1 492 -> 1 852`). Les deux changements sont donc contemporains, et ils portent sur deux
structures differentes (le tampon d'etat de `managed-player` dans la trame, la tenue dans
`chunk_00`).

### 3.2 L'oracle externe, avec recalage

La confrontation terme a terme de la phase 3 exige `nombre de rangs == nombre d'entites ti=9` et
ECARTE les films ou les deux different. C'est le cas de `11de8353` (24 rangs, 25 entites). Ce lot
essaie **tous les recalages** du vecteur des rangs sur celui des entites, et publie le nombre de
recalages en accord EXACT — ce nombre est le plancher de bruit MESURE. Le vecteur attendu est
indexe **par rang** et porte `-1` aux rangs dont le XUID n'est pas dans l'oracle, pour qu'un rang
manquant ne decale pas la suite.

| film | build | rangs lus (+vacants) | apparies | entites ti=9 | recalages en accord EXACT | essais | rangs compares |
|---|---|---|---|---|---|---|---|
| **`11de8353`** | **`HI_1_9_0`** | **24 (+8)** | **23** | **25** | **[0]** | **2** | **23** |
| `0014603f` | `HI_1_12_0` | 8 (+24) | 8 | 8 | [0] | 1 | 8 |
| `06a883f7` | `HI_1_12_0` | 24 (+8) | 23 | 24 | [0] | 1 | 23 |
| `000d5950` | `HI_1_13_0` | 8 (+24) | 8 | 8 | [0] | 1 | 8 |
| `00ba2e1c` | `HI_1_11_0` | 22 (+10) | 22 | 24 | [0] | 3 | 22 |
| `084a804d` | `HI_1_10_0` | 24 (+8) | 24 | 24 | [0] | 1 | 24 |
| `0a247154` | `HI_1_8_0` | 8 (+24) | 8 | 8 | [0] | 1 | 8 |
| `a521164d` | `HI_1_4_1` | 24 (+8) | 23 | 24 | [0] | 1 | 23 |
| `03af54c3` | (sans identification) | 24 (+8) | 24 | 24 | [0] | 1 | 24 |
| `1c5c10cc` | `HI_1_10_0` | 23 (+9) | 22 | 24 | [0] | 2 | 22 |
| `1950c59b` | `HI_1_8_0` | 8 (+24) | 8 | 8 | **[]** | 1 | 8 — **temoin negatif FFA** |
| `b1bcbe24` | `HI_1_11_0` | 23 (+9) | 23 | **20** | — | **0** | — table plus longue que la trame |
| **total** | | | | | **10/12 films a recalage UNIQUE** | **15** | **185 slots en accord** |

**10 films sur 12 rendent un recalage et un seul, et c'est toujours 0** ; les deux exceptions sont
le temoin negatif FFA (le film dit « aucune equipe », la base fabrique un camp par joueur) et un
film dont la table porte plus de rangs que la trame n'a d'entites, cas ou le recalage n'a pas de
sens. **10 recalages en accord sur 15 essayes** : le plancher est mesure, pas estime.

---

## 4. Le drapeau de controle par build : ce qu'il ecrit, et il n'est nulle part

### 4.1 Ce que l'executable dit — l'ECRIVAIN, qui n'avait pas ete lu

La phase 4 avait lu le LECTEUR d'etat complet (`FUN_142e2bfd0`) et note qu'un mot de 32 bits
« de controle » est lu entre l'etat par defaut et `n2` si `FUN_14076cea8()` est vrai. Le
symetrique, `FUN_142e2d08c`, dit **ce que ce mot contient** :

```
  (**(code **)(*plVar14 + 0x58))(...)          <- l'etat par defaut de l'archetype
  if (DAT_1450e24e8 != '\0') {
      ... ecriture de 32 bits de la valeur 0xffddcba ...
  }
  ... n2 ...
```

**Ce n'est pas un champ, c'est une SENTINELLE constante `0x0FFDDCBA`** — un marqueur de detection
de corruption. Le nom que le depot porte deja (`filmComponentCorruptionCheck`, `traverse.go`) est
donc exact.

Ce qui positionne le drapeau, sur pieces :

| cote | globale | ce qui l'ecrit |
|---|---|---|
| **ECRIVAIN** (`FUN_142e2d08c` @`142e2d508`) | `DAT_1450e24e8` | **aucune** reference d'ecriture dans l'image ; valeur statique **0** |
| **LECTEUR** (`FUN_14076cea8`) | `DAT_144c23326` si `FUN_1404f2b4c()`, sinon `DAT_1450e24e8` | **aucune** reference d'ecriture ; valeurs statiques **0** |

Les deux globales n'ont **que des lectures** dans tout le binaire (`DAT_1450e24e8` : 3 lectures,
dont l'initialiseur de l'ecrivain de film `FUN_14299b674` ; `DAT_144c23326` : 1 lecture). Ce sont
donc des interrupteurs de developpement, poses par un systeme de reglage externe au code
statique, et **a zero par defaut**. Aucun mode de jeu, aucune globale de session ne les active.

**Le predicat de choix, lui, vient du film.** `FUN_1404f2b4c` lit
`*(uint *)(index * 0x1134f0 + 0xea71c + DAT_145121d28) == 2`. Le pas `0x1134F0` est la taille de
la structure de session que `FUN_14095944c` recopie pour le film, et `+0xEA71C` est l'un des
champs que l'ecrivain du corps de `chunk_00` serialise (`FUN_1407ec560` @`1407ec828`), **sur
2 bits** (`SHL RDX,0x2` a `1407ec844`) — donc de domaine `0..3`, et la valeur `2` y est
representable. La phase 1 portait « 1 bit » pour ce champ : **correction, c'est 2 bits.**

### 4.2 La mesure sur les films

`TestResidusTrameSentinelle` cherche `0x0FFDDCBA` a **tout decalage de bit** dans les paquets de
type 2, et lit en parallele les 32 bits qui precedent immediatement le premier composant — la
place ou la sentinelle tomberait.

| film | build | paquets type 2 | **S1 : occurrences de `0x0FFDDCBA`** | **S2 (temoin positif) : les 32 bits a i0-32** | records ti=9 |
|---|---|---|---|---|---|
| `11de8353` | `HI_1_9_0` | 16 | **0** | `88` x400 | 400 |
| `0014603f` | `HI_1_12_0` | 19 | **0** | `136` x152 | 152 |
| `06a883f7` | `HI_1_12_0` | 28 | **0** | `136` x672 | 672 |
| `000d5950` | `HI_1_13_0` | 25 | **0** | `136` x200 | 200 |
| `00ba2e1c` | `HI_1_11_0` | 23 | **0** | `88` x552 | 552 |
| `084a804d` | `HI_1_10_0` | 43 | **0** | `88` x1 032 | 1 032 |
| `0a247154` | `HI_1_8_0` | 39 | **0** | `88` x312 | 312 |
| `a521164d` | `HI_1_4_1` | 13 | **0** | `88` x312 | 312 |
| `03af54c3` | (sans identification) | 24 | **0** | `88` x576 | 576 |
| `1950c59b` | `HI_1_8_0` | 20 | **0** | `88` x160 | 160 |
| **total** | **7 builds** | **250** | **0** | **4 368 / 4 368 lisent `n2`** | **4 368** |

Le negatif est publie **avec son temoin positif** (methode, regle 4) : a l'endroit exact ou la
sentinelle tomberait, on lit `n2` sur 4 368 records sur 4 368. La recherche est donc au bon
endroit, et elle ne trouve rien.

Troisieme lentille (`TestResidusTrameDeuxFormes`) : le critere interne d'equipe — domaine `1..9`
et partage en deux moities egales — applique aux DEUX formes possibles.

| decalage | films qui passent | ce qu'on y lit |
|---|---|---|
| **`d = 186`** (drapeau inactif) | **8 / 10** | des designateurs `{1, 2}`, partages 4-4 et 12-12 ; les 2 echecs sont le FFA (valeurs `0`) et un film a 25 entites (cardinal impair, le partage egal est mecaniquement impossible) |
| **`d = 218`** (drapeau actif) | **0 / 10** | `15` sur **toutes** les entites de **tous** les films — la valeur `0xF`, hors du domaine |

**Verdict** : le drapeau est **inactif dans tous les films du cache, sur les sept builds**, et
c'est MESURE (0 sentinelle sur 250 paquets, 0/10 films a `d = 218`), non plus deduit de l'absence
d'un autre decalage. Un decodeur robuste peut le detecter a cout nul : lire les 32 bits a
`i0 - 32` et comparer a `0x0FFDDCBA`.

---

## 5. Le nom d'un designateur : le consommateur d'affichage

### 5.1 La chaine, cote executable

La table `mp_team_designator` n'a **aucune reference de code** : ni son descripteur
(`0x1445c0c00`), ni son tableau de 9 noms (`0x144723da0`, dont la seule reference est le
descripteur lui-meme). Le descripteur appartient a une table de descripteurs d'enumerations de
script de pas **`0x58`** (verifie sur l'entree voisine `0x1445c0ba8`, de cardinal 2), et aucune de
ces entrees n'est referencee par du code : elles sont consommees **par nom**, a l'execution, par
le systeme de script. **Chercher « qui lit la table » par les references croisees ne peut donc
rien rendre — et c'est une mesure, pas un renoncement.**

Le consommateur se trouve en suivant le NOM de la fonction de script, pas la table. Il s'appelle
`AddTeamDesignatorStringIdsToList` (`0x1436e34f0`), et il est enregistre dans
`FUN_140ee83dc` @`140ee863d` avec le pointeur de fonction `0x142d417c0` :

```
FUN_142d417c0  ->  FUN_142d40c1c(composant, param) :
    "team_0"       -> ajoute a la liste
    "team_1"       -> ajoute a la liste
    "team_2" ... "team_7"
    "team_neutral" -> ajoute a la liste
```

**Neuf identifiants de chaine, dans cet ordre**, contre neuf noms d'enumeration
(`First`, `Second`, `Third`, `Fourth`, `Fifth`, `Sixth`, `Seventh`, `Eighth`, `Neutral` — les deux
extremites relues en memoire : `First` @`0x1436daff0`, `Neutral` @`0x1436db008`). Meme cardinal,
meme ordre, et la seconde liste **nomme la numerotation** : elle part de `team_0`.

**Verdict.** `team_id 0` -> designateur 0 -> identifiant de chaine `team_0` -> premiere entree de
`mp_team_designator`, `First`. La correspondance **est prouvee pour l'INDEXATION** par deux voies
sans etape commune (la table de l'enumeration, lue en donnees ; la liste d'identifiants de chaine,
lue en code). Ce qui **n'est PAS prouve ici** : que l'etiquette affichee de `team_0` soit
« Eagle ». Le binaire ne porte que l'identifiant `team_0` ; le libelle vit dans les fichiers de
chaines du jeu, hors de l'executable. La seule trace d'« Eagle » / « Cobra » dans le binaire est
cote variante de mode, ou `eagleStartScore` est a `variante+0x1108` et `cobraStartScore` a
`variante+0x110C` (`FUN_142c76fc0` / `FUN_142c76ef8`) — **Eagle au plus petit offset, donc en
premier**, ce qui est coherent mais n'est pas une preuve d'indexation.

### 5.2 La cardinalite reelle, cote donnees

Oracle externe (lecture seule sur la sauvegarde `data/backups/pre-chaine-2026-09-09/`) :
**aucun mode a equipes du corpus ne depasse 2 equipes.**

| playlist | `team_id` distincts | plage | matchs | joueurs |
|---|---|---|---|---|
| Big Team Battle | **2** | 0..1 | 595 | 24 a 32 |
| Quick Play | **2** | 0..1 | 1 237 | 8 a 13 |
| Ranked Arena / Slayer / Team Snipers / Squad Battle / ... | **2** | 0..1 | — | — |
| **SURVIVE THE UNDEAD** | **4** | 0..3 | 1 | 4 |
| **Rumble Pit** (FFA) | **8** | 0..7 | 3 | 8 a 11 |
| Firefight | 1 | 0 | 5 | 4 |

**La question « une Grande bataille porte-t-elle jusqu'a 8 equipes ? » est REFUTEE** : la Grande
bataille est a **deux** equipes de douze, sur 595 matchs.

Et ce que le FILM en dit (`TestResidusTrameCardinalite`), sur les trois seuls matchs du cache que
la base credite de plus de deux `team_id` :

| film | mode | `team_id` distincts en base | **valeurs brutes dans le film** |
|---|---|---|---|
| `1950c59b` | Arena FFA Slayer | 8 | **`0` x160** — une seule valeur, « aucune equipe » |
| `610363ee` | Shotty Snipe Slayer FFA | 8 | **`0` x216** — idem |
| `007d53a4` | SURVIVE THE UNDEAD | 4 | **`1` x200** — une seule valeur, designateur 0 |
| `58d09c44` | BTB One Flag CTF | 2 | `1` x648, `2` x648 |
| `7344d24f` | Strongholds Arena | 2 | `1` x120, `2` x120 |

**Mesure** : aucun film du cache ne peut porter plus de deux designateurs nommes — les seuls
candidats a plus de deux equipes lisent dans le film une valeur UNIQUE. Les designateurs `Third`
a `Eighth` **ne sont exerces par aucun film de ce corpus**, et le `team_id` multiple de la base
est, dans les trois cas, une fabrication de l'API (le meme phenomene que le temoin negatif FFA de
la phase 3, etendu ici a un mode a 4 camps).

---

## 6. Le pied de film : `b37`/`b38` portent l'equipe, `b55` ne porte rien

### 6.1 Le protocole

La contradiction etait ouverte depuis l'archive : `RESEARCH_THEATER_RE.md:534` identifie l'equipe
a `b37`/`b38`, sa ligne 624 la dit fiable en `b55`, sa ligne 645 la dit fausse ; la production lit
`b55` et le commente « NON fiable ». Elle ne pouvait pas etre tranchee faute d'equipe de reference
interne. Elle l'est maintenant : l'equipe par XUID est **prouvee dans le film seul** (rang de la
table des slots de `chunk_00` -> XUID, i-eme entite ti=9 -> designateur a 186, equipe = brut - 1).

Le balayage est **aveugle** : on teste **les 60 octets** du bloc, chacun par **trois** lectures
(valeur brute, valeur moins un, bit de poids faible), soit 180 lectures. Le bloc est celui de la
production, transcrit sans changement depuis `objectiveevents/film.go`.

### 6.2 La mesure

Corpus : 16 films de modes a objectif du cache (CTF, Strongholds, One Flag), dont **14 retenus** —
deux ecartes parce que le nombre de rangs de la table ne vaut pas le nombre d'entites ti=9, cas ou
l'equipe par rang n'a pas de sens. **688 evenements `th=10` vus, 665 confrontes.**

| lecture | accord | % |
|---|---|---|
| **`b37` brut** | **665 / 665** | **100,0** |
| **`b37` bit bas** | **665 / 665** | **100,0** |
| **`b38` brut** | **665 / 665** | **100,0** |
| **`b38` bit bas** | **665 / 665** | **100,0** |
| `b16` bit bas | 398 / 665 | 59,8 |
| `b0` bit bas | 360 / 665 | 54,1 |
| `b18` bit bas | 357 / 665 | 53,7 |
| ... (les 173 autres lectures) | ≤ 357 / 665 | ≤ 53,7 |

**Plancher de bruit MESURE : 4 lectures en accord parfait sur 180 essayees**, et ce sont les deux
lectures de `b37` et les deux de `b38` (leurs valeurs valent 0 ou 1, donc « brut » et « bit bas »
coincident). **Deux octets sur soixante portent l'equipe.**

Le detail des octets en litige :

| octet | brut == equipe | valeur-1 == equipe | bit bas == equipe | valeurs observees |
|---|---|---|---|---|
| **`b37`** | **665/665** | 0/665 | **665/665** | `1` x347, `0` x318 |
| **`b38`** | **665/665** | 0/665 | **665/665** | `1` x347, `0` x318 |
| **`b55`** (ce que la production lit) | 318/665 | 0/665 | 318/665 | **`0` x665 — une seule valeur** |
| `b36` (le « slot ») | 173/665 | 153/665 | 341/665 | `0` x187, `3` x171, `1` x158, `2` x149 |

**Verdict, en trois lignes.**

1. **`b37` et `b38` SONT l'equipe** : 665/665, valeurs `{0, 1}`, egales terme a terme a l'equipe
   prouvee par la trame. L'archive (`:534`) avait raison.
2. **`b55` ne porte rien** : il vaut `0` sur les 665 evenements des 14 films. Son accord de
   318/665 est **mecanique** — c'est le nombre d'evenements dont l'acteur est d'equipe 0. C'est
   exactement le piege de methode que la phase 2 avait deja nomme en C.1 : un champ constant a
   zero « coincide » avec l'equipe 0 sur la moitie des lignes. Le commentaire « NON fiable sur
   certains matchs » de la production decrit donc un champ **toujours** faux pour l'equipe 1, pas
   un champ intermittent.
3. **`b36` n'est pas l'index de joueur** : il prend les valeurs 0..3 de facon a peu pres uniforme
   sur des films a 8 et 24 joueurs. Ce n'est pas le `player_index`. Son role n'est pas etabli ici.

Releve brut (extrait de `TestResidusPiedDrapeaux`, `008e1bba` et `58d09c44`) :

```
t=  51677 xuid 2535405671792166 equipe prouvee 0 | b36 0 | b37..b43 00 00 00 00 00 00 00 | b55 0
t=  66008 xuid 2577506572472715 equipe prouvee 1 | b36 1 | b37..b43 01 01 00 00 00 00 00 | b55 0
t= 100302 xuid 2535415455425240 equipe prouvee 1 | b36 3 | b37..b43 01 01 00 01 00 00 00 | b55 0
t= 124522 xuid 2533274872234198 equipe prouvee 0 | b36 2 | b37..b43 00 00 00 02 00 00 00 | b55 0
```

`b40` prend 0, 1 ou 2 et ne suit pas l'equipe ; `b39`, `b41`, `b42`, `b43` sont nuls sur le releve.

**Consequence pour la production, NON TRAITEE ici (regle 7)** : `objectiveevents/film.go` lit
`b55` sous le nom `teamRaw`, et `objectiveevents/extract.go` a raison de ne pas s'y fier — mais la
bonne correction n'est pas « aller chercher l'equipe au roster », c'est **lire `b37`**. Aucun code
de production n'a ete touche par ce lot.

---

## 7. Ce qui est PROUVE, ce qui est HYPOTHESE, ce qui est REFUTE

### Prouve (fermeture arithmetique, ou deux chaines sans etape commune)

1. Un enregistrement de slot VACANT mesure `1 299 + perso(build) + 352 + 32` bits, soit **13 619**
   sur `HI_1_10_0`/`HI_1_11_0`, et c'est **exactement** le surplus des deux ecarts aberrants de la
   phase 4 (reste 0, 2 films sur 2).
2. La grammaire ferme **2 121 ecarts sur 2 121** sur les 96 films du cache a ≥ 17 joueurs, une
   fois les slots vacants enjambes et verifies par un predicat grammatical sans seuil.
3. La transposition par build est **entierement** portee par le bloc de personnalisation
   `sub+0xcc0` : la moitie APRES le bloc de queue est invariante, et le RESTE hors bloc nul vaut
   **270 bits sur les sept builds**.
4. Le lecteur par grammaire calibre sur le film rend le compte du balayage corrige sur
   **1 351 films sur 1 351**, et lit **32 slots exactement** (occupes + vacants) sur 1 351 films
   sur 1 351 — la borne de l'ecrivain, verifiee par la lecture.
5. Le delta est **unique par build** sur les 1 351 films : `0`, `-2 880`, `-4 320`, `+1 600`.
6. `HI_1_9_0` : i0 **derive** a 186 (`108 + 32 + 14 + 32`), `n1 = 12`, `n2 = 88`, record 459 bits,
   transposition `-4 320` — ligne identique a `HI_1_8_0`.
7. Le mot gate par le drapeau de controle est la **sentinelle constante `0x0FFDDCBA`**
   (`FUN_142e2d08c`), et elle est **absente des 250 paquets de type 2 examines**, sur 7 builds,
   avec temoin positif a **4 368/4 368**.
8. Le critere interne d'equipe passe a `d = 186` sur 8 films sur 10 et **a `d = 218` sur 0 sur
   10** (valeur `15` partout, hors domaine).
9. Le champ `jeu+0xEA71C` — celui que `FUN_1404f2b4c` compare a 2 — est ecrit sur **2 bits** par
   `FUN_1407ec560` (`SHL RDX,0x2`), et non 1 comme la phase 1 le portait.
10. Le consommateur d'affichage du designateur est
    `AddTeamDesignatorStringIdsToList` -> `FUN_142d40c1c`, qui empile **`team_0` .. `team_7`,
    `team_neutral`** dans cet ordre : la numerotation part de 0 et `team_0` est la premiere
    entree, en correspondance de cardinal et d'ordre avec `First`..`Eighth`, `Neutral`.
11. **`b37` et `b38` du bloc du pied valent l'equipe** : 665/665 sur 14 films, contre un plancher
    de 4 lectures parfaites sur 180 essayees.

### Hypothese (une seule chaine, ou non tranche)

1. **La cause de la variation du bloc de personnalisation.** Deux des trois ecarts ferment
   exactement sur le pas des attaches d'armure (`-360 = 10 x 0x24`, `-540 = 15 x 0x24`) ; le
   troisieme (`+200` octets sur `HI_1_4_1` et les films sans identification) ne ferme sur aucun
   pas connu. Sans executable de ces builds, la lecture « le nombre d'attaches a change » reste
   une hypothese.
2. **Le role de `b40` du bloc du pied** (0, 1 ou 2) et celui de `b36` (0..3, pas le
   `player_index`).
3. **Ce qui pourrait activer le drapeau de controle.** Les deux globales n'ont aucune ecriture
   dans l'image : interrupteurs de developpement poses hors du code statique. Rien ne prouve
   qu'aucun build ne les active ; la mesure dit seulement qu'aucun film de ce cache ne l'a.
4. **Le libelle affiche d'un designateur.** L'indexation est prouvee (`team_0` = `First` =
   designateur 0) ; que `team_0` s'affiche « Eagle » n'est pas dans l'executable.

### Refute

1. **« Les deux ecarts aberrants n'ont pas d'explication. »** (phase 4, C.5 n°1) REFUTE : un slot
   vacant, au bit pres, sur les deux films.
2. **« La longueur d'un enregistrement de slot VIDE est indeterminee, entre 15 700 et 17 900
   bits. »** (phase 1, ouvert n°3) REFUTE : elle se calcule, et elle vaut `1 683 + perso(build)`.
3. **« Le candidat naturel de la transposition par build est le bloc de personnalisation —
   a verifier sur un executable de ces builds, qu'on n'a pas. »** (phase 2, ouvert n°4) REFUTE
   comme impossible : la verification se fait **sur les films**, sans executable, en coupant
   l'enregistrement autour du bloc de queue.
4. **« Le lecteur canonique n'est pas robuste aux builds anciens ; c'est un travail de
   decodeur. »** (phase 2, ouvert n°5) REFUTE comme difficulte : une constante calibree sur le
   film et un predicat de slot vacant suffisent, et le resultat est exact sur 1 351 films.
5. **« Une Grande bataille peut porter jusqu'a huit equipes. »** REFUTE : 2 equipes sur
   595 matchs de Grande bataille ; le seul mode a plus de 2 `team_id` reels en base est le FFA, ou
   le film ne porte **aucune** equipe.
6. **« L'equipe d'un evenement du pied est a l'octet 55. »** (`objectiveevents/film.go`) REFUTE :
   `b55` vaut 0 sur 665 evenements sur 665. **`b37` et `b38` sont l'equipe, a 665/665.**
7. **« Le champ `jeu+0xEA71C` est ecrit sur 1 bit. »** (phase 1, table du corps) REFUTE : 2 bits.
8. **« Le drapeau de controle est mesure faux partout. »** — l'affirmation de la phase 4 etait une
   DEDUCTION (le balayage ne retenait que 186), pas une mesure. Elle est desormais MESUREE, et
   elle tient.

---

## 8. Commandes pour rejouer

Les instruments sont sous garde d'environnement ; sans la variable ils sont sautes, donc la CI
reste verte. Les chemins sont **en style Windows** (`C:/...`) : un chemin de style Git Bash
(`/c/...`) fait echouer l'ouverture.

```bash
cd apps/go-api
C="C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks"

# 2 : la transposition par build et le lecteur calibre (un film par build)
F="000d5950 0014603f 00ba2e1c 084a804d 11de8353 0a247154 a521164d 03af54c3"
L=""; for f in $F; do L="$L;$C/$f"; done; L="${L#;}"
CHUNK00_FILMS="$L" go test ./internal/analysis/filmdec/ \
  -run 'TestResidusSlotTransposition|TestResidusSlotChaineParBuild' -v -timeout 30m

# 2 : le controle d'echelle sur les 1 351 films du cache (~2 min)
CHUNK00_CORPUS="$C" go test ./internal/analysis/filmdec/ \
  -run TestResidusSlotChaineCorpus -v -timeout 120m

# 1 : les deux ecarts aberrants, sur pieces
CHUNK00_FILMS="$C/b1bcbe24;$C/1c5c10cc" go test ./internal/analysis/filmdec/ \
  -run TestResidusSlotAberrants -v -timeout 30m

# 1 : la fermeture de la grammaire sur les films a gros roster (96 films, ~5 s)
#     la liste se construit par intersection, elle n'est pas ecrite a la main :
go build -o /tmp/diagq ./cmd/diag_q
B="C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/backups/pre-chaine-2026-09-09/shared_matches_v2.duckdb"
L=""; for f in $(/tmp/diagq "$B" \
  "SELECT substr(match_id,1,8) FROM match_participants GROUP BY 1 HAVING count(*) >= 17" \
  | tail -n +2 | tr -d '\r'); do [ -d "$C/$f" ] && L="$L;$C/$f"; done
CHUNK00_FILMS="${L#;}" go test ./internal/analysis/filmdec/ \
  -run TestResidusSlotFermeture -v -timeout 60m

# 3 et 4 : HI_1_9_0 sur la trame, la sentinelle du drapeau, la cardinalite
F="11de8353 0014603f 06a883f7 000d5950 00ba2e1c 084a804d 0a247154 a521164d 03af54c3 1950c59b"
L=""; for f in $F; do L="$L;$C/$f"; done; L="${L#;}"
CHUNK00_FILMS="$L" go test ./internal/analysis/filmdec/ \
  -run 'TestResidusTrameSentinelle|TestResidusTrameDeuxFormes|TestResidusTrameCardinalite' \
  -v -timeout 60m
CHUNK00_FILMS="$L" CHUNK00_XUID_EQUIPES="$ORACLE" go test ./internal/analysis/filmdec/ \
  -run TestResidusTrameOracleRecale -v -timeout 60m

# 3 : la table de profil par build, completee (instrument de la phase 4, rejoue)
CHUNK00_FILMS="$C/11de8353;$C/0014603f;$C/06a883f7;$C/000d5950" \
  go test ./internal/analysis/filmdec/ -run TestProfilBuildsTrame -v -timeout 40m

# 6 : le pied — les 60 octets contre l'equipe prouvee (16 films de modes a objectif)
L=""; for f in $(/tmp/diagq "$B" "SELECT substr(match_id,1,8) FROM match_registry \
  WHERE lower(game_variant_name) LIKE '%ctf%' OR lower(game_variant_name) LIKE '%strongh%'" \
  | tail -n +2 | tr -d '\r'); do [ -d "$C/$f" ] && L="$L;$C/$f"; done
CHUNK00_FILMS="${L#;}" go test ./internal/analysis/filmdec/ \
  -run 'TestResidusPiedOctetEquipe|TestResidusPiedDrapeaux' -v -timeout 60m
```

`CHUNK00_XUID_EQUIPES` est de la forme `<prefixe>=<xuid>:<equipe>,...`, blocs separes par `;`.
**L'ordre n'y est pas lu** : l'instrument apparie par XUID. Il se construit en **lecture seule**
depuis la sauvegarde (le fichier de production peut etre tenu par le serveur — dans ce cas ne pas
forcer) :

```bash
/tmp/diagq "$B" "SELECT substr(p.match_id,1,8) AS pref,
  string_agg(p.xuid || ':' || COALESCE(CAST(p.team_id AS VARCHAR),'NULL'), ',') AS roster
  FROM match_participants p WHERE p.xuid NOT LIKE 'bid(%' GROUP BY 1"
```

La cardinalite par mode de la section 5.2 :

```bash
/tmp/diagq "$B" "WITH m AS (SELECT p.match_id, COALESCE(r.playlist_name,'?') pl,
    count(*) n, count(DISTINCT p.team_id) nt, min(p.team_id) lo, max(p.team_id) hi
  FROM match_participants p LEFT JOIN match_registry r ON r.match_id = p.match_id
  WHERE p.xuid NOT LIKE 'bid(%' GROUP BY 1,2)
 SELECT pl, nt, min(lo), max(hi), count(*), min(n), max(n)
   FROM m GROUP BY 1,2 ORDER BY nt DESC, 5 DESC"
```

Gates passes : `gofmt -l` net, `go vet ./internal/analysis/filmdec/` net,
`go test ./internal/analysis/filmdec/` sans garde **ok**, `go test ./internal/archlint/` **ok** —
le ratchet des variables de paquet de `filmdec` n'est pas touche (les instruments n'introduisent
que des `const`, des types, des fonctions de test et des locales).

---

## 9. Decouvertes hors perimetre — notees, NON TRAITEES (regle 7)

1. **Le decalage de 8 octets de `parseRegistry`** : connu depuis la phase 1, **NON TRANCHE**, pas
   touche par ce lot.
2. **La production lit `b55` pour l'equipe d'un evenement d'objectif** (`objectiveevents/film.go`,
   `extract.go`). Refute par la section 6 : il faut lire `b37`. **Code de production, hors
   perimetre.**
3. **La phase 1 porte `jeu+0xEA71C` a 1 bit** dans la table du corps de `chunk_00` ; le
   desassemblage dit 2 bits. La table du corps n'est pas re-verifiee champ par champ ici, et les
   offsets voisins (`+0xEA714`, `+0xEA718`, `+0xEA71D`, `+0xEA724`, `+0xEA728`) ne sont pas
   re-mesures. **A reprendre si quelqu'un decode le corps.**
4. **Le modele de record d'image-cle de la PRODUCTION est faux** (`keyframe_record_walk.go`) :
   consigne par la phase 4, toujours non traite.
5. **Le champ `sub+0xc35` distingue un slot vacant (valeur signee 0) d'un slot occupe (-1).**
   Observation utile pour un decodeur, non exploitee ici.
6. **19 films du build courant portent un slot vacant AVANT la fin de leur roster** (c'est ce qui
   tuait `s3sChaine` sur eux). La raison pour laquelle l'index de joueur a un trou n'est pas
   etablie.

---

## Ce qui reste ouvert

1. **La cause de la variation du bloc de personnalisation sur `HI_1_4_1`** (+200 octets, ne ferme
   sur aucun pas connu). Les cinq films sans section d'identification se comportent comme lui :
   leur build reste inconnu, et le decouvrir fermerait les deux questions d'un coup.
2. **Le libelle affiche d'un designateur** (« Eagle » / « Cobra », les couleurs d'equipe). Il vit
   dans les fichiers de chaines du jeu, pas dans l'executable : la mesure demande le corpus
   `gamefiles` (tag `//go:build gamefiles`, lecture de l'installation locale). **A planifier, pas
   a deviner.**
3. **Le role de `b36`, `b40` et des octets 39 a 43 du bloc du pied.** `b36` prend 0..3 et n'est
   pas le `player_index` ; `b40` prend 0, 1 ou 2.
4. **Brancher la lecture en production** : le pont `rang -> xuid` (phase 2), le designateur par
   slot (phase 3), le profil par build et le lecteur exact sur sept builds (phase 4 + ce lot) sont
   maintenant tous disponibles. Rien n'est branche : **aucun code de production n'a ete touche.**
5. **Le film a plus de rangs que d'entites** (`b1bcbe24` : 23 rangs, 20 entites ti=9). La table
   est le roster a l'ecriture du chunk, la trame compte les presents ; l'ecart dans ce sens-la
   (joueurs partis) n'a pas ete quantifie sur le corpus.
