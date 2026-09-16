# La bande de slots bipede : ce que l'executable en dit (2026-09-16)

> Lot 3.5, item 3.5.1 du plan `.ai/PLAN_DECODEUR_FILM_2026-09-13.md` — instruction bornee,
> AUCUN correctif. Source unique : `HaloInfinite.exe` (Ghidra, LECTURE SEULE, HTTP direct
> `127.0.0.1:8089`, image base `140000000`, projet du 2026-06-04) et le depot. Aucun film n'a
> ete decode pour cette note (voie de decodage tenue ailleurs). Adresses et valeurs COLLEES.
>
> Instrument / Ghidra lecture seule — rien n'a ete renomme ni commente dans le projet Ghidra.

---

## 1. La reponse en cinq lignes

1. La bande d'un bipede n'est PAS deduite : l'executable la DECLARE. Un bipede est alloue
   dans la classe 1 du repartiteur d'entites, **base `0x200` = 512, cardinal `0x100` = 256**,
   soit **`[512, 767]`** — deux entiers de `.rdata`, `0x143cefd80` et `0x143cefd84`.
2. Le vehicule est la classe voisine : base `0x300` = 768, cardinal 256, soit `[768, 1023]`.
   Le depot MESURE deja ces deux bases sans les avoir nommees (§3).
3. Ces deux bornes sont des **constantes de code** : la table est en `.rdata`
   (`143606000..1443951ff`), elle n'a que deux sites de LECTURE dans tout le binaire et
   **aucun site d'ecriture**. Elles ne dependent ni de la carte, ni de la taille de la table
   d'objets, ni d'une configuration de partie.
4. Le seul cardinal DYNAMIQUE du modele est celui de la classe 4 (le vivier general, base
   `0x500` = 1280) et des domaines d'encodage 0, 1, 7 et 8 : il suit `DAT_144706100`, la
   taille courante de la table d'entites, qui ne fait que CROITRE. La croissance est
   explicitement refusee a toute classe autre que la 4 (`142f2f1e6 CMP R14D, 0x4` /
   `142f2f1ea JNZ`). **Un bipede ne peut donc pas obtenir un slot au-dessus de 767.**
5. Consequence directe sur la question posee : `[512, 7808]` et `[512, 8064]` ne sont pas des
   variantes de build. La borne basse 512 est la VRAIE base ; la borne haute est un artefact
   du releve, pas une donnee du jeu. **Verdict (ii)** — ligne au registre des reports, §6.

---

## 2. Le modele, tel que l'executable l'ecrit

### 2.1 La table des classes d'allocation — `.rdata`, immuable

`FUN_142f2fc08(type, &base, &cardinal, &classe)` traduit le TYPE d'un objet en classe, puis
lit la paire dans une table de `.rdata` a `0x143cefd78` (`142f2fc4f` pour la base,
`142f2fd85` pour le cardinal). Octets releves a `0x143cefd78` (48 o) :

```
00000000 00020000 | 00020000 00010000 | 00030000 00010000 | 00040000 00010000
00050000 ff1a0000 | ffffffff 00000000
```

| Classe | Base | Cardinal | Bande | Adresse de la base | Adresse du cardinal |
|---|---|---|---|---|---|
| 0 | `0x000` = 0 | `0x200` = 512 | `[0, 511]` | `0x143cefd78` | `0x143cefd7c` |
| **1** | **`0x200` = 512** | **`0x100` = 256** | **`[512, 767]`** | **`0x143cefd80`** | **`0x143cefd84`** |
| 2 | `0x300` = 768 | `0x100` = 256 | `[768, 1023]` | `0x143cefd88` | `0x143cefd8c` |
| 3 | `0x400` = 1024 | `0x100` = 256 | `[1024, 1279]` | `0x143cefd90` | `0x143cefd94` |
| 4 | `0x500` = 1280 | `DAT_144706100 - 1280` | `[1280, table-1]` | `0x143cefd98` | (calcule) |

Le cardinal de la classe 4 n'est PAS lu dans la table : `FUN_142f2fc08` le calcule
(`if (uVar3 == 4) cardinal = DAT_144706100 - base;`). C'est la seule classe elastique.

`get_xrefs_to 0x143cefd78` rend exactement deux entrees, toutes deux dans `FUN_142f2fc08`,
toutes deux en LECTURE (`142f2fc48` DATA, `142f2fc4f` READ) ; `0x143cefd7c` une seule
(`142f2fd85` READ). Segment `.rdata` : aucune ecriture possible.

### 2.2 Le type 0x23 est le bipede, le type 0x28 le vehicule

Le branchement de `FUN_142f2fc08` sur `param_1` (le type), trace a la main :

- `param_1 = 0x23` (35) : `0x23 >= 0x1a` et `< 0x27` ; ni `0x26`, ni `< 0x21` ; `!= 0x21` et
  `0x23 - 0x22 = 1 != 0` ; puis `0x23 - 0x23 = 0` -> `LAB_142f2fc46` avec **classe 1**.
- `param_1 = 0x28` (40) : `>= 0x27`, `<= 0x2c`, ni `0x2c` ni `0x27` -> **classe 2**.
- `param_1 = 0x29` (41) : -> **classe 3**.

Deuxieme lecture, independante, qui confirme la paire : le sequenceur de references dans le
flux (`FUN_1406d5110`, ecrivain ; `FUN_1406d3140`, lecteur) traite `0x23` et `0x28` ENSEMBLE
et eux seuls. Extrait du desassemblage de l'ecrivain :

```
1406d51fe  CALL 0x1405d5d08        ; type de l'entite depuis son handle
1406d5203  CMP  EAX, 0x23          ; bipede ?
1406d52f9  CALL 0x1405d5d08
1406d52fe  CMP  EAX, 0x28          ; vehicule ?
...
1406d523e  CMP  byte ptr [0x144706104], 0x0
1406d524b  MOV  ESI, dword ptr [0x1451f98f0]   ; base    = 0x200 = 512
1406d5251  MOV  EDI, dword ptr [0x1451f98f4]   ; cardinal = 0x200 = 512
```

Quand le bit de porte vaut 1 (type `0x23` ou `0x28`), la reference est encodee sur le
domaine 4, **base 512 cardinal 512, soit `[512, 1023]`** — l'UNION exacte de la classe 1
(bipede) et de la classe 2 (vehicule). Les numeros `0x23` = 35 et `0x28` = 40 sont ceux des
archetypes `ti=35` et `ti=40` du film : l'enumeration de types de l'executable et l'index
d'archetype de l'ECS sont **la meme numerotation**.

### 2.3 La largeur d'un champ de reference est `ceil(log2(cardinal))`

`FUN_1406d310c(n)` rend `ceil(log2(n))` (bit de poids fort, +1 si le reste est non nul).
L'ecrivain `FUN_1406d5110` ecrit `(handle & 0x3fffffff) - base` sur cette largeur, puis
`R(2)` de generation (`handle >> 30`). Le lecteur `FUN_1406d3140` fait l'inverse et rend
`base + R(largeur)`. D'ou, par domaine :

| Domaine | Base | Cardinal | Largeur | Bande |
|---|---|---|---|---|
| 0, 1 | 512 | `DAT_144706100 - 512` | 13 au defaut | `[512, table-1]` |
| 2 | 512 | 256 | 8 | `[512, 767]` |
| 3 | 768 | 256 | 8 | `[768, 1023]` |
| **4** | **512** | **512** | **9** | **`[512, 1023]`** (bipede + vehicule) |
| 5 | 1024 | 256 | 8 | `[1024, 1279]` |
| 6 | 0 | 512 | 9 | `[0, 511]` |
| 7, 8 | 0 | `DAT_144706100` | 13 au defaut | `[0, table-1]` |

Ce tableau est ECRIT par `FUN_140d10bb0` (9 entrees, boucle `iVar3 = 0..8`), qui pose
`DAT_144706104 = 1` puis remplit `DAT_1451f98d0` (base) / `DAT_1451f98d4` (cardinal) par pas
de 8 octets, avec des LITTERAUX (`0x200`, `0x100`, `0x300`, `0x400`) et `DAT_144706100` pour
les quatre domaines elastiques. Valeur statique relevee a `0x144706100` : `ff 1f 00 00` =
**0x1FFF = 8191** — d'ou la largeur 13 par defaut, et d'ou le `kfTableCap = 8192` que le
depot porte en dur (`filmdec/keyframe_world.go:19`).

Les deux entrees du domaine 4 (`0x1451f98f0`, `0x1451f98f4`) n'ont AUCUN site d'ecriture hors
`FUN_140d10bb0` : `get_xrefs_to` rend deux lectures chacune (`1406d524b` / `1406d5251` cote
ecrivain, `1406d3305` / `1406d330b` cote lecteur). Seuls `0x1451f98d4` (domaine 0),
`0x1451f98dc` (domaine 1), `0x1451f990c` (domaine 7) et `0x1451f9914` (domaine 8) recoivent
des ecritures apres coup, depuis `FUN_1408f1618` (`1423503d3..eb`) et `FUN_142f2f0cc`
(`142f2f292..2b1`).

### 2.4 La croissance est interdite a la classe du bipede

`FUN_142f2f0cc` est l'allocateur : il demande sa bande a `FUN_142f2fc08`, cherche un slot
libre dans `[base, base + cardinal - 1]`, et s'il n'en trouve pas :

```
142f2f1dd  CMP  byte ptr [0x144706104], 0x0   ; table de domaines active ?
142f2f1e4  JZ   0x142f2f1f0                   ; non -> on peut agrandir
142f2f1e6  CMP  R14D, 0x4                     ; classe == 4 (vivier general) ?
142f2f1ea  JNZ  0x142f2f2cf                   ; non -> ECHEC, pas d'agrandissement
```

et, seulement pour la classe 4 :

```
142f2f292  MOV  dword ptr [0x1451f990c], EAX  ; domaine 7 cardinal = taille+1
142f2f299  MOV  dword ptr [0x1451f98dc], ...  ; domaine 1 cardinal = taille - 0x1ff
142f2f2a1  MOV  dword ptr [0x1451f98d4], ESI  ; domaine 0 cardinal = taille - 0x1ff
142f2f2aa  MOV  dword ptr [0x1451f9914], ...  ; domaine 8 cardinal
142f2f2b1  MOV  dword ptr [0x144706100], ...  ; taille de la table d'entites
```

`FUN_1408f1618` (enregistrement d'un handle recu, chemin de replication) fait les memes cinq
ecritures quand le slot annonce depasse la table courante. Les deux chemins ne font que
CROITRE la table et ne touchent jamais les classes 0 a 3.

**Corollaire** : dans le modele de l'executable, un bipede occupe toujours un slot de
`[512, 767]`, et un vehicule un slot de `[768, 1023]`, quelle que soit la taille de la table.
La taille de la table ne change QUE la largeur des champs des domaines 0, 1, 7 et 8.

### 2.5 Le film dit lui-meme si la table de domaines est active

`DAT_144706104` — la bascule qui fait passer de « base 0, cardinal `DAT_144706100` » a la
table par domaine — est ecrite au debut du traitement d'un paquet de trame :

```
14298749f  CALL 0x1406cf008                   ; R(1) sur le lecteur de bits du paquet
1429874ab  MOV  byte ptr [0x144706104], AL
```

C'est l'amorce de paquet que le depot documente deja (`filmdec/frame_records.go:47-55`,
`PacketPreambleBits`). Le film est donc autoportant sur ce point : il porte le bit qui dit
quel modele de reference lire. Rien ici n'appelle un « profil par build ».

---

## 3. Ce que le depot mesure deja, et qui colle

| Piece du depot | Ce qu'elle dit | Lecture a la lumiere de l'exe |
|---|---|---|
| `RAPPORT_LOT_H_VERSIONS_2026-09-13.md` §D5 | `[512, 643..767]` sur les versions 31, 33, 37, 41 | Exactement la classe 1 : base 512, jamais au-dela de 767 |
| `film_re/V7_DESTRUCTION_EVENEMENT_2026-09-02.md:83` | bande `ti=40` de 13 a 67 slots, **minimum 768** | Exactement la base de la classe 2 |
| `film_re/V7_DESTRUCTION_EVENEMENT_2026-09-02.md:251` | `ti=40` « base + 256..302, soit les slots 768..814 quand la base vaut 512 » | La base 512 et le decalage 256 sont les deux bases de `.rdata` |
| `film_re/ETAT_VEHICULES_2026-08-31.md:159` | vehicules « 768-1023 » | La classe 2 entiere |
| `film_re/V5_ETAT_OCCUPATION_2026-09-02.md:154` | bande bipede = 103 slots, bande vehicule = 47 | Sous les 256 de chaque classe |
| `film_re/SONDAGE_E2_BIPEDE_INDEX_2026-09-08.md` §2.2 | premier record bipede de `d9781168` : **slot 515** | Dans `[512, 767]` |
| `filmdec/keyframe_world.go:19` | `kfTableCap = 8192` | La valeur statique de `DAT_144706100` + 1 (`0x1FFF` + 1), pas une borne de bipede |
| `filmdec/offline_biped.go:60` | `bipedSlotBits = 13` | La largeur des domaines 0/1/7/8, pas celle d'un slot de bipede (9 bits au domaine 4, 8 au domaine 2) |
| `filmdec/frame_records.go:39-44` | `IDLowBits` « valeur de RUNTIME, `FUN_1406d3140 -> FUN_1406d310c` sur `DAT_1451f98d0/d4` » | Exact : c'est `ceil(log2(cardinal du domaine))`, et seuls 0/1/7/8 bougent |
| `filmdec/offline_biped_band.go:161-190` | `bipedSlotBand` = union des `ti=35` vus aux images-cle, **puis comblement de tous les trous entre min et max, sans borne de largeur** | Le comblement n'a aucune borne : un seul record aberrant emporte toute la bande |

Les deux bases mesurees par le depot (512 pour `ti=35`, 768 pour `ti=40`) sont les deux
litteraux de `.rdata`. Le recoupement est complet dans les deux sens.

---

## 4. La cause de `[512, 7808]` et `[512, 8064]`

Ce qui est ETABLI :

1. **7808 et 8064 ne sont pas des slots de bipede.** La classe 1 s'arrete a 767 et ne croit
   jamais (§2.4). Aucune donnee de build, de carte ou de partie ne peut deplacer cette borne.
2. **La borne basse 512 est vraie**, et c'est une constante de `.rdata`, pas une mesure.
3. **Le releve du depot n'a aucune borne haute** : `bipedSlotBand` prend le max des slots
   `ti=35` vus et comble tout entre min et max (`offline_biped_band.go:186-190`) ; cote
   `killsource`, `timeline.bipedRange()` (`killsource/world.go:127-143`) fait le meme min/max
   nu. Le seul garde-fou en amont est `kfValidAnchor`, dont la seule borne haute est
   `slot < kfTableCap = 8192` (`keyframe_world.go:105`) — c'est-a-dire le domaine ENTIER de
   la table d'entites, pas la bande du bipede. Un unique record accepte a tort a un slot de
   la queue du domaine suffit a porter `bipHi` a 7808 ou 8064, et le comblement rend alors la
   bande dense sur `[512, 8064]` : la porte ne filtre plus rien (constat §D5 du rapport H,
   `hors_plage_bipede` tombe a 0 ou 1).
4. La forme des deux valeurs est coherente avec un ancrage fautif plutot qu'avec une entite :
   `7808 = 0x1E80` et `8064 = 0x1F80` ont leurs **sept bits de poids faible nuls** et sont a
   384 et 128 du plafond `0x2000` ; `kfValidAnchor` exige en outre `slot > prevSlot`, ce qui
   ne laisse survivre, parmi les faux ancrages, que ceux de la queue du domaine.

Ce qui n'est PAS etabli ici, et pourquoi : la position exacte du record fautif dans le
payload, et donc la demonstration que c'est bien un faux ancrage de `kfScanNext` plutot
qu'un record vrai d'un autre archetype lu avec un `ti` decale, demandent d'ouvrir les trois
films concernes (`11de8353`, `e5adf7b2`, `111fa685`). **Le decodage de film est hors du
perimetre de cette session** (voie tenue par un autre agent). Ce point est la condition de
reprise ecrite au registre.

Meme raisonnement pour la borne basse `128` relevee sur `a26dbcdb`, `443426df`, `084a804d` et
`5676a9ba` (`[128, 757..767]`) : 128 est en dessous de la base 512 de la classe 1, donc ce
n'est pas davantage un slot de bipede ; c'est la meme famille de defaut, par le bas.

---

## 5. Ce que le correctif du lot 3.5 aura a sa disposition (aucune ligne ecrite ici)

- Une bande de bipede **bornee par construction** : `[512, 767]`, et une bande de vehicule
  `[768, 1023]` — deux constantes de l'executable, a poser comme telles (avec leur
  provenance Ghidra), pas comme une mesure du film.
- Le releve du film garde son role : il dit QUELS slots de `[512, 767]` sont occupes. Ce
  qu'il n'a pas a decider, c'est l'INTERVALLE.
- Un negatif mesurable, gratuit, qui vaut garde-rail : tout slot `ti=35` hors `[512, 767]`
  (ou `ti=40` hors `[768, 1023]`) est un record a REFUSER et a COMPTER, jamais a integrer a
  la bande. Sur les films sains, le compteur doit valoir 0 ; sur `11de8353`, `e5adf7b2` et
  `111fa685` il vaudra exactement le nombre de records fautifs — ce qui repond du meme coup
  a la question laissee ouverte au §4.
- Deux replis du registre de repli tombent avec ce correctif :
  `repli_bande_bipede_comblee` (`filmdec/offline_biped_band.go`) et
  `repli_deadstate_hors_bande_bipede` (`killsource/walk.go`), tous deux fleches « lot 3.5 »
  au plan (registre des replis, lignes 5058 et 5081).

---

## 6. Verdict 3.5.1

**(ii) — constante de code.** Les bornes de la bande de slots du bipede sont deux entiers de
`.rdata` de `HaloInfinite.exe` (`0x143cefd80` = `0x200`, `0x143cefd84` = `0x100`), lus par
`FUN_142f2fc08` et jamais ecrits ; la classe du bipede est exclue de la seule croissance de
table que l'allocateur autorise (`142f2f1e6 CMP R14D, 0x4`). Il n'y a donc **pas d'entree de
profil** a creer pour cette bande : elle ne depend d'aucune donnee de build, de carte ni de
partie. La variabilite observee d'un film a l'autre est un defaut de RELEVE cote decodeur,
pas une divergence de build.

Ligne portee au registre des reports (`.ai/V7.5/REGISTRE_REPORTS.md`) avec sa condition de
reprise : le correctif est un lot M3, et l'identification du record fautif demande un
decodage de film.

---

## 7. Adresses citees (toutes verifiees dans cette session)

| Symbole / adresse | Role |
|---|---|
| `FUN_142f2fc08` (`0x142f2fc08`) | type d'objet -> (base, cardinal, classe) |
| `0x143cefd78` .. `0x143cefd98` | table des classes d'allocation, `.rdata`, immuable |
| `142f2fc4f`, `142f2fd85` | les deux seules lectures de cette table |
| `FUN_142f2f0cc` (`0x142f2f0cc`) | allocateur d'entite ; `142f2f1e6 CMP R14D, 0x4` = la classe 4 seule peut agrandir |
| `142f2f292`, `142f2f299`, `142f2f2a1`, `142f2f2aa`, `142f2f2b1` | les cinq ecritures de croissance |
| `FUN_1408f1618` (`0x1408f1618`) | enregistrement d'un handle ; memes cinq ecritures a `1423503d3..eb` |
| `FUN_140d10bb0` (`0x140d10bb0`) | remplit la table des 9 domaines d'encodage ; `DAT_144706104 = 1` |
| `DAT_1451f98d0` / `DAT_1451f98d4` | base / cardinal du domaine 0 (`.data`, pas de 8 o) |
| `DAT_1451f98f0` / `DAT_1451f98f4` | domaine 4 = `[512, 1023]`, bipede + vehicule, jamais reecrit |
| `DAT_144706100` | taille de la table d'entites ; valeur statique `0x1FFF` = 8191 |
| `DAT_144706104` | bascule « table de domaines active », ecrite a `1429874ab` depuis un `R(1)` du paquet |
| `FUN_1406d310c` (`0x1406d310c`) | `ceil(log2(n))` — la largeur d'un champ de reference |
| `FUN_1406d3140` (`0x1406d3140`) | lecteur d'une reference : `base + R(largeur)` puis `R(2)` generation |
| `FUN_1406d5110` (`0x1406d5110`) | ecrivain symetrique ; `1406d5203 CMP EAX, 0x23` / `1406d52fe CMP EAX, 0x28` |
| `FUN_1405d5d08` (`0x1405d5d08`) | type d'une entite depuis son handle |
| `FUN_142987460` (`0x142987460`) | processeur de trame ; `1429874ab` ecrit la bascule depuis le film |
