# R_LOC — localisation des paquets à événements : R-LS (D-67) et R-L1 (c) (2026-10-02)

> Chantier `loc` de la campagne de grammaire (préfixe `r_loc`). Worktree temporaire
> `LevelUp-wt-cg-loc`, HEAD détachée sur `fe18bf67c`. Recherche seule : aucun fichier suivi modifié,
> aucune base, aucune cuisson, `grammar.Rev` inchangé (`grammar-2026-09-27.3`). Films lus en place,
> un à la fois, plafond 4 Gio (pics de 91 à 368 Mio).
>
> Conventions : **mesuré** = compté sur les films par la sonde ou l'outil ; **établi** = lu dans
> Ghidra ET confirmé par la mesure ; **estimé** / **supposé** = dit comme tel. « Sain » = paquet
> fermé que le juge `cmContredit` (décision D2) ne contredit pas ; « factice » = fermé contredit.
> Pourcentages sur dénominateur **fixe** (maximum des records utiles lus par film, sur toutes les
> marches mesurées de la campagne ET de ce chantier, §6.0 du plan) et **variable**.

## Corrections du 2026-10-02 (ajoutées après coup ; le texte d'origine ci-dessous n'est pas réécrit)

Sources : verdicts adverses du chantier loc (`VERIFICATIONS_ADVERSES_R.md`, « Chantier loc »),
`CRITIQUE_COMPLETUDE_R.md` (points 7, 8, 10, 14, 21), `SURCOUCHE_UNIQUE.md` §1.3, §4.1, §4.6 et
`R_COMB_2.md` §6-§7.

1. **« 99,5 % des signatures acceptées »** (§2.2) : recalculé à 64 950 / 65 342 = **99,4 %** pour
   `ls` (99,78 % pour `ls2`).
2. Liste des slots liés incomplète : il manque 3072 (`0797ce72`, 2 chunks), 6528 (`1c4c63c2`,
   6 chunks) et 4 (`e60aaf06`, 1 chunk) ; `e60aaf06` est rangé à tort parmi les « 123 seul ». Les
   comptes sont pris après `lierLeChunkAuMonde` sur toutes les images-clés du chunk, pas « au début de
   chunk ».
3. **Repli à largeur libre** : « −16 673 paquets sains » est la perte BRUTE de sains ; le solde net
   est 276 316 → 262 175, soit **−14 141** (même correction pour R-LOC-5 et §5).
4. **Pourcentages « variables »** : ce sont des fermés BRUTS rapportés aux utiles lus ; en sains,
   HI_1_13_0 80,3 % → 82,9 %.
5. L'ordre retenu (`ls2`) n'est mesuré que par la sonde ; la copie de recherche de `cmd_fermeture`
   réalise l'ordre `ls`. Rejoué depuis sous la surcouche unique (bascule `CAMPAGNE_RLOC_LS=1`) : carte
   `ls` identique 20 / 20 (26 421 non localisés, 296 755 fermés).
6. **Killsource** : « aucune valeur publiée ne change » est inexact : la voie (`read_path`) est
   PERSISTÉE (`internal/sync/killcollector/collector_batch.go:72`) ; montée de `killsource.Rev` et
   backfill killsource DUS. Tag, statut, crédit et origine ne changent pas. Pour `51ebbc0f`, seule
   l'égalité des comptes et des voies est vérifiable.
7. **Films de killsource (critique point 8)** : les 28 films sont les 19 témoins + les 9 de l'enquête ;
   `1c4c63c2` (corpus) et `81c02726` en sont exclus faute de carte lisible. R-COMB-2 retrouve 229 morts
   du `scan` à la marche sur les 19 témoins. Rejoué sous la surcouche unique sur 3 films de l'enquête :
   voies identiques à `rloc_killsource_comparaison.tsv` ; les 25 autres non rejoués.
8. **Signature par NOM (critique point 7)** : la sonde identifie l'archétype par le nom
   `high-frequency` (`rlocArchetypesHF`), contre la règle de R-HOM. Mesuré par R-COMB-2 : sous L8,
   toutes les signatures tombent sur `ti=4`, aucune sur `ti=3` ; le prédicat du lot s'écrira par TABLE
   de composant.
9. **Surcouche** : `r_loc_overlay/` n'était pas inerte : elle remplaçait sans condition le
   localisateur des deux sites. Dans la surcouche unique, la règle passe derrière
   `CAMPAGNE_RLOC_LS=1`.
10. **Vue A (§4)** : les statistiques de genres et l'oracle portent sur la région `ls` (26 421), pas
    `ls2` (26 427) ; « HI_1_13_0 de 73,8 % à au plus 75,8 % » mélange deux bases (+1,8 point sur `ls`,
    non mesuré sur `ls2`) ; 16 066 + 10 350 = 26 416, pas 26 421 ; la « borne saine » de l'oracle
    n'est pas un majorant (45 % de gains factices) ; « table des gestionnaires construite à
    l'exécution » = supposé fort ; `FUN_140bbd474` a une seconde table (`DAT_145225910`) non recopiée.
11. **D-79 / « bornes » (critique point 21)** : LS dépasse l'oracle (ii) de BIS_1 ; en combinaison
    (C11, R-COMB-2), LS rend +34 222 sains marginaux. Un oracle en référence n'est pas une borne.
12. **Constaté par la surcouche unique, non instruit** : sous LS, l'alerte de santé « dead-states à tag
    `jpt!` hors du roster » de `6b0e6f0f` passe de 10 à 42 ; cette note ne la mentionne pas.
13. Gate de la note : `archlint` rouge à `fe18bf67c` (`TestNoExpiredTODO`, hors chantier), soldé depuis.

## 0. En bref

1. **D-67 vérifié sur pièces.** Les deux localisateurs (`grammar.marchLocateStrict` et la copie
   `killsource.locateStrict` / `locateFallback`) comparent bien le slot au littéral `123`. Le
   registre de chacun des 30 films lus a un seul archétype à composant unique nommé `high-frequency`
   (`ti=4`). Dans les modes à objectif porté, le monde lie d'AUTRES slots à cet archétype (124 à
   129, 133, 134, 135, 304 selon les films), et le slot 123 n'y est lié que dans une partie des
   chunks (par exemple 11 chunks sur 38 pour `d9781168`).
2. **Carte de fermeture, 20 films.** La signature acceptée sur tout slot `high-frequency` fait
   tomber les paquets à événements non localisés (région (ii)) de **48 720 à 26 421** (−45,8 %).
   Mais **l'ordre de l'enquête (« 123 strict -> autre slot high-frequency -> … ») perd sur un
   film** : `1c4c63c2` (HI_1_10_0) passe de 13 540 à 12 691 paquets fermés sains (−849), parce que
   la signature préempte la fermeture par NEW de tête que `debutDeLaListe` essayait avant. L'ordre
   **`ls2`** (123 strict -> fermeture par NEW de tête -> signature high-frequency) ne fait baisser
   **aucun film** : +18 119 paquets gagnés (36 contredits, 0,2 %), 2 perdus (1 sain, 0 record
   utile) ; records utiles fermés sains **2 573 107 -> 2 742 546** (+169 439).
3. **Killsource, 28 films** (9 de l'enquête + 19 témoins ; `1c4c63c2` exclu, carte inconnue) : sous
   surcouche, **1 080 kills** passent du `scan` à la `marche` (1 591 -> 511 kills `scan`), aucun dans
   l'autre sens, **aucune valeur publiée ne change** (tag, statut, crédit identiques sur les
   4 366 morts), aucune mort n'apparaît ni ne disparaît. Les trois films de l'enquête redonnent
   exactement ses chiffres : 157/167, 180/209, 25/27 = **362/403**.
4. **R-L1 (c), établi dans Ghidra** : la vue A est une concaténation SANS préfixe de longueur de
   messages déjà sérialisés (`FUN_142f2c050` recopie des tampons par `FUN_1406d60f4`, chacun écrit
   par `FUN_140bbd474` : `1`, genre `R(7)`, trois références gardées, charge du genre par la
   `vtable[0x60]` du gestionnaire), suivie du terminateur `0` ; la vue B commence au bit suivant.
   **Elle se lit jusqu'au bout par grammaire, mais seulement en lisant la charge de CHAQUE genre
   présent** : aucun raccourci de longueur n'existe.
5. **Région (ii) après LS** : **26 427** paquets restent non localisés sur les 20 films (`ls2`).
   Leurs genres de tête : 41 genres distincts ; 56,9 % ont un genre de tête sans aucune grammaire
   de charge dans le dépôt, 43,1 % un genre dont seule la TÊTE est lue en Go (le tir, genre 36,
   9 160 paquets à lui seul). La lecture actuelle (`consumeVueA`) en localise **0** par
   construction. L'oracle « premier départ qui ferme » en localise au plus 16 066 (60,8 %), mais
   45 % de ses gains supplémentaires sont factices : **borne saine estimée +8 826 paquets fermés
   sains, +83 609 records utiles sains** (corpus 20 films).

## 1. Protocole

**Sondes** (neuves, tag `research`, ligne 1) :
- `apps/go-api/internal/games/halo_infinite/film/internal/grammar/r_loc_ls_research_test.go`
  (460 lignes) : `TestRLocLocalisateur`. La marche est celle de la campagne (`cmMarcher`, même
  monde, même pilotage, même classement que la carte v2). Seul le localisateur de début de liste
  change, par la variante `tete`. Chaque paquet fermé passe au juge `cmContredit` (le même que
  `cmJuge`).
- `apps/go-api/internal/games/halo_infinite/film/internal/grammar/r_loc_vuea_research_test.go`
  (174 lignes) : l'oracle de fin de vue A et son contrôle (§4.3).

Variantes du localisateur :

| Variante | Règle |
|---|---|
| `ref` | 123 strict ; sinon fermeture par NEW de tête (`debutParFermeture`) ; chaîne de tête si 123 trouvé. Égale `debutDeLaListe` : la sonde appelle aussi la production sur chaque paquet à événements, **0 écart** sur 30 films (colonne `ecarts_ref_production`). |
| `ls` | 123 strict ; sinon signature stricte (35 bits, composant unique, génération du profil) sur un AUTRE slot que le monde lie à un archétype `high-frequency` (lu par NOM : seul composant `high-frequency`) ; sinon fermeture par NEW de tête. C'est l'ordre de l'enquête, et c'est ce que fait la surcouche de `marchLocateStrict`. |
| `ls2` | 123 strict ; sinon fermeture par NEW de tête ; sinon signature high-frequency (+ chaîne de tête). |
| `ls+libre` | `ls`, puis repli à largeur libre du slot 123 (`marchLocateFallback`) avant la fermeture. |
| `libre` | 123 strict, puis repli à largeur libre (contrôle : le repli seul). |
| `ls+vueA` | `ls`, puis l'oracle de fin de vue A (§4.3) sur ce qui reste non localisé. |

**Surcouche** (`r_loc_overlay/`, méthode D-48/D10, outillage de MESURE seulement) : copies de
`grammar/object_deaths_march.go` (`marchLocateStrict` en deux passes dans une seule boucle :
première signature 123, sinon première signature sur un autre slot high-frequency) et de
`facts/killsource/walk.go` (`locateStrict`, même règle ; `locateRecordsAvecVerdict` et le repli
inchangés). `overlay.json` remplace les deux fichiers de CE worktree.

**Commandes** (depuis `apps/go-api`, `GOCACHE` dédié, CGO) :
- `RLOC_RACINE=<LevelUp>/data/cache/film_chunks RLOC_FILMS=<30 ids> RLOC_SORTIE=<scratchpad>
  go test -tags=research -count=1 -timeout 240m -run '^TestRLocLocalisateur$' ./internal/games/halo_infinite/film/internal/grammar/`
  — passe 1 (cinq variantes, contrôle 300 paquets par film) : `PASS` 2 111 s ; passe 2
  (`RLOC_VARIANTES=ls,ls2 RLOC_CONTROLE=0`) : `PASS` 450 s.
- `go build -overlay=<r_loc_overlay/overlay.json> ./cmd/killsource` (et la même sans surcouche),
  puis `killsource json <id> -carte <carte> -cache <LevelUp>/data/cache` sur 28 films.
- `go build -tags=research -overlay=… ./internal/games/halo_infinite/film/research/cmd_fermeture`,
  puis `-mode v2` sur les 20 films.

**Films** : les 20 du corpus (19 témoins de `config/replay_corpus.toml` + `1c4c63c2`), les 9 films
de l'enquête présents dans le cache (`8f7f5806`, `6b0e6f0f`, `9c0ec856`, `72b0a25e`, `d1a18847`,
`e60aaf06`, `a765f61d`, `c46ef9d1`, `af3af607`), et `81c02726`.

**Contrôles (mesurés, tous tenus)** :
- la marche `ref` de la sonde rend, sur les 20 films, la carte v2 de la campagne À L'IDENTIQUE
  (paquets fermés, listes non localisées, utiles lus, utiles fermés ; 20/20) : 284 704 paquets
  fermés, 48 720 listes non localisées, 2 588 167 / 5 961 028 utiles, 264 757 hors cadre ;
- l'instrument officiel `cmd_fermeture -mode v2` compilé SOUS SURCOUCHE rend, film par film, les
  mêmes quatre comptes que la variante `ls` de la sonde (20/20 identiques) ;
- `ref` = `debutDeLaListe` paquet par paquet (0 écart sur 30 films).

## 2. R-LS — vérification sur pièces

### 2.1 Code (lu le 2026-10-02 sur `fe18bf67c`)

- `grammar/object_deaths_march.go:50` : `marchSignatureSlot = uint32(123)` ; `marchSignature123`
  (l. 115) exige `rec.Slot == marchSignatureSlot`, 35 bits, un composant ; `marchLocateFallback`
  (l. 139) exige aussi `rec.Slot == marchSignatureSlot`.
- `facts/killsource/walk.go` : `signature123` compare `rec.Slot == 123` en littéral ;
  `locateFallback` aussi. Les deux copies portent la même règle (confirmé).
- **Écart entre les deux sites, non signalé par l'enquête** : la cuisson
  (`debutDeLaListe`, `debut_de_liste.go:41`, appelée par `movement_states.go:287`,
  `frame_closure.go:237` et `frame_closure_detail.go:186`) n'appelle QUE `marchLocateStrict`, puis
  la fermeture par NEW de tête ; elle n'a PAS de repli à largeur libre. Le repli n'existe que dans
  `marchLocalise` (marche des morts d'objet, `ScanMarchFacts`) et dans killsource. Les deux sites
  n'ont donc pas le même ordre aujourd'hui, et un « même localisateur » (LU) devra le dire.

### 2.2 Registre et images-clés (mesuré, `rloc_registre.tsv`)

- Sur les 30 films (9 builds, de version-31 à HI_1_13_0) : un seul archétype a pour unique
  composant `high-frequency` : `ti=4`. Un autre archétype porte un composant de ce nom :
  `ti=3 i1` (sur 2 composants), l'homonyme de D-49 ; il n'entre pas dans la règle (lecture par nom
  ET composant unique).
- Slots que le monde de la marche lie à `ti=4` au début d'au moins un chunk (slot : chunks) :

| Film | Mode (corpus ou enquête) | Slots `high-frequency` |
|---|---|---|
| `8f7f5806` | Ranked:Oddball, Live Fire | 123:17 127:23 128:19 (sur 60) |
| `a765f61d` | Ranked:Oddball, Live Fire | 123:19 127:23 128:21 |
| `6b0e6f0f` | variante `4640310c`, Refuge | 123:14 126:13 127:13 128:13 129:12 |
| `9c0ec856` | Ranked:Oddball, Recharge | 123:17 124:12 |
| `c46ef9d1` | Ranked:Oddball, Recharge | 123:19 124:19 125:24 |
| `af3af607` | Ranked:Oddball, Lattice | 123:23 124:20 125:22 |
| `d9781168` | Oddball:Arena, Dredge | 123:11 133:11 134:15 (sur 38) |
| `c75f33b8` | Assault:One Bomb, Curfew | 123:5 127:14 135:4 (sur 25) |
| `51ebbc0f` | Oddball:Arena, Banished Narrows | 123:12 304:14 |
| `60ae07c4` | Ranked:Oddball, Live Fire (HI_1_8_0) | 123:21 127:21 |
| `1c4c63c2` | HI_1_10_0, mode non déclaré | 123:15 126:12 127:13 128:13 129:14 6528:6 |
| `0797ce72` | Super Fiesta, Live Fire | 123:23 3072:2 |
| 18 autres films (CTF à deux drapeaux, Slayer, BTB, Strongholds, KotH, Banished Narrows hors Oddball) | | 123 seul |

  Les slots de l'enquête (124, 126 à 129) sont retrouvés ; d'autres s'y ajoutent (125, 133, 134,
  135, 304). **Le slot 123 n'est lié à `ti=4` que dans une partie des chunks** des films à objectif
  porté (5 chunks sur 25 pour `c75f33b8`) : ce que le slot 123 porte ailleurs n'est pas instruit.
- Les signatures acceptées par `ls` tombent sur ces slots dans 99,5 % des cas. Exceptions
  (`rloc_slots_hf.tsv` contre `rloc_registre.tsv`, mesuré) : `1c4c63c2` 359 paquets sur des slots
  qu'aucun début de chunk ne lie à `ti=4` (2048 ×161, 4096 ×137, 3904 ×43, …), `e5adf7b2` 22
  (slot 2048), `6b0e6f0f` 10 (slot 0), `8f7f5806` 1. Ces liaisons viennent du monde en cours de
  chunk (table de datums, anticipation) ; leur justesse n'est pas instruite (découverte §6).

## 3. R-LS — effet mesuré

### 3.1 Carte de fermeture, 20 films (`rloc_agregats_groupe_variante.tsv`, `rloc_par_build_corpus20.tsv`)

Dénominateur fixe du corpus : 6 784 607 records utiles (maximum par film sur les marches de bis 1
à 4 et de ce chantier ; il MONTE avec LS, D-42 : `d9781168` passe de 253 988 à 316 761).

| Variante | Non localisés (région (ii)) | Fermés | Fermés sains | Utiles fermés sains | Utiles lus | Hors cadre | Gagnés (contredits) | Perdus (dont sains) | Films en baisse : paquets sains / utiles sains | Sains / fixe | Variable |
|---|---|---|---|---|---|---|---|---|---|---|---|
| `ref` | 48 720 | 284 704 | 276 316 | 2 573 107 | 5 961 028 | 264 757 | — | — | — | 37,9 % | 43,4 % |
| `ls` | 26 421 | 296 755 | 293 457 | 2 758 570 | 6 284 483 | 275 530 | 18 430 (8) | 6 379 (1 982) | **1** / 0 | 40,7 % | 44,1 % |
| **`ls2`** | **26 427** | **302 821** | **294 402** | **2 742 546** | 6 203 648 | 268 368 | **18 119 (36)** | **2 (1)** | **0 / 0** | **40,4 %** | 44,5 % |
| `ls+libre` | 9 334 | 281 514 | 279 871 | 2 432 417 | 6 373 370 | 302 658 | 21 019 (264) | 24 209 (18 098) | 10 / 7 | 35,9 % | 38,2 % |
| `libre` | 31 240 | 268 835 | 262 175 | 2 241 825 | 6 048 130 | 292 602 | 2 616 (275) | 18 485 (16 673) | 12 / 9 | 33,0 % | 37,2 % |
| `ls+vueA` (oracle) | 10 350 | 313 005 | 302 283 | 2 842 179 | 6 370 216 | 274 491 | 34 725 (7 420) | 6 424 (2 045) | 1 / 0 | 41,9 % | 45,0 % |

Lecture (mesuré) :
- **La seule baisse de `ls` est `1c4c63c2`** : fermés sains 13 540 -> 12 691, perdus 6 043 dont
  1 758 sains (567 utiles). Ses records utiles sains montent pourtant (164 875 -> 187 297) : la
  perte est en paquets. Cause (mesurée par les passes, `rloc_passes.tsv`) : en `ref`, 8 188 paquets
  y sont localisés par la fermeture par NEW de tête ; sous `ls`, la signature high-frequency les
  prend d'abord (16 531 paquets) et la marche qui en part ferme moins souvent que celle partie du
  NEW de tête. Sous `ls2`, `1c4c63c2` monte : 14 053 sains (+513), 176 154 utiles sains (+11 279).
- **Le gate « aucune baisse sur aucun film » de §6.0 n'est tenu que par `ls2`.** `ls2` ne retire
  rien de ce que `ref` localise : 2 paquets perdus sur le corpus (1 sain, 0 record utile).
- **Le repli à largeur libre du slot 123 est nuisible dans la marche de la carte** : `libre` seul
  perd 18 485 paquets (16 673 sains) et fait baisser 12 films ; `4f77afc1` passe de 22 910 à
  11 051 fermés. Il ne faut PAS l'ajouter à `debutDeLaListe`. (Sa valeur dans killsource et
  `marchLocalise`, où il existe déjà, n'est pas remesurée ici.)
- **« Hors cadre » monte** sous `ls2` (264 757 -> 268 368) : des paquets désormais localisés
  avancent jusqu'à la vue C et y sont arrêtés, même déplacement de cause que D-64.
- Les films sans autre slot high-frequency (CTF à deux drapeaux, Slayer, BTB, Strongholds, KotH)
  ne bougent pas d'un paquet sous `ls2` ; seuls 6 films du corpus changent (`d9781168`,
  `c75f33b8`, `51ebbc0f`, `60ae07c4`, `1c4c63c2`, et `e5adf7b2` en localisation seulement).

Par build, records utiles fermés sains sur le dénominateur fixe (corpus 20 films) :

| Build | `ref` | `ls` | `ls2` | `ls+vueA` |
|---|---|---|---|---|
| HI_1_13_0 (fixe 2 931 799) | 68,6 % | 74,0 % | **73,8 %** | 75,8 % |
| HI_1_10_0 | 16,2 % | 17,4 % | 16,8 % | 18,4 % |
| HI_1_8_0 | 28,9 % | 30,5 % | 30,4 % | 30,8 % |
| autres builds | inchangés | | | |

Variable HI_1_13_0 : 80,6 % -> 83,2 %. Le fixe HI_1_13_0 de bis 3 (2 759 700) est dépassé : LS lit
plus loin que toutes les marches antérieures (D-42).

Films par film (`rloc_par_film_ref_ls_ls2.tsv`), témoins à objectif porté, fermés sains
`ref -> ls2` : `d9781168` 26 214 -> 36 873 ; `c75f33b8` 22 855 -> 25 743 ; `51ebbc0f` 9 759 ->
13 137 ; `60ae07c4` 13 828 -> 14 476 ; `1c4c63c2` 13 540 -> 14 053. CTF (`bcb6d393`, `fb1a1a72`,
`084a804d`, `4f77afc1`) : inchangés.

Films de l'enquête (9 films, `ls2`) : non localisés 39 422 -> 8 956 ; fermés sains 296 976 ->
364 866 ; utiles fermés sains 2 414 094 -> 3 036 253 ; 68 028 gagnés dont 174 contredits ; 0 perdu.
`8f7f5806` : 41 329 -> 56 394 sains ; `6b0e6f0f` 36 181 -> 45 475. Les films de Banished Narrows
(`72b0a25e`, `e60aaf06`) et le témoin `d1a18847` ne bougent pas (aucun autre slot high-frequency).

### 3.2 Killsource (`rloc_killsource_comparaison.tsv`, `rloc_killsource_morts.tsv`)

Surcouche de `walk.go` (ordre « 123 strict -> autre slot high-frequency -> repli libre du slot
123 » ; killsource n'a pas de fermeture par NEW de tête, donc `ls` = `ls2` sur ce site). Morts
appariées par (instant, victime) :

| Groupe | Morts | `scan` réf. | `scan` LS | scan -> marche | marche -> scan | Tag / statut / crédit changés | Morts apparues / disparues |
|---|---|---|---|---|---|---|---|
| 9 films de l'enquête | 1 619 | 1 042 | 191 | 851 | 0 | 0 / 0 / 0 | 0 / 0 |
| 19 témoins | 2 747 | 549 | 320 | 229 | 0 | 0 / 0 / 0 | 0 / 0 |
| **Total 28 films** | **4 366** | **1 591** | **511** | **1 080** | **0** | **0** | **0** |

- Origine « les deux vérités concordent » (= `credit-concordant`) : `8f7f5806` 167 -> 10,
  `6b0e6f0f` 209 -> 29, `9c0ec856` 27 -> 2 : **362 / 403 rendus à la marche**, exactement le
  chiffre de l'enquête (§3.1 de l'enquête).
- Témoins à objectif porté : `d9781168` 105 -> 2, `c75f33b8` 57 -> 9, `60ae07c4` 80 -> 2 ;
  **`51ebbc0f` 52 -> 52, inchangé** alors que la carte y gagne (+3 378 sains) : sortie killsource
  identique à l'octet (bloc `sante` comparé). Non instruit ; hypothèse (supposée) : le monde de
  killsource (première déclaration de chaque slot) ne lie pas le slot 304 à `ti=4` au moment des
  paquets, ou la seconde cause de l'enquête (objet transitoire de Banished Narrows) bloque la
  marche avant les bipèdes.
- CTF à deux drapeaux, Slayer, BTB : 0 changement.
- **Delta killsource à déclarer pour le lot** : provenance (`voie`) de 1 080 morts sur ces 28 films,
  aucune valeur. `1c4c63c2` et `81c02726` ne sont pas joués : leur carte n'est pas connue, et la
  commande refuse sans carte (règle du 2026-09-27).

## 4. R-L1 (c) — la vue A se lit-elle jusqu'au bout par grammaire ?

### 4.1 L'écrivain et le lecteur (Ghidra `HI_1_13_0`, lecture seule ; extraits dans `r_loc_ghidra/`)

| Maillon | Adresse | Ce qu'il fait (lu) |
|---|---|---|
| enregistreur par tick | `FUN_142f2c3b0` | écrivain 0 : `FUN_142f2c050(w)` puis `FUN_1406d49c4(w)` (le terminateur `0`) ; écrivain 1 : `FUN_142f2cc78` (vue B) ; écrivain 2 : tampons de contrôle (vue C) puis terminateur. Les trois sont recopiés bout à bout dans le paquet (T3 §1.1). |
| vue A | `FUN_142f2c050` | boucle sur 2 047 entrées de 16 octets de `DAT_14521d920` (`+0` genre, `+4` décalage, `+8` nombre de BITS) ; pour chaque entrée de taille > 0 : `FUN_14080ada0(genre)` (nom de débogage, aucune écriture) puis `FUN_1406d60f4(w, _, 0x1451f9920 + décalage, nbits)` = **recopie de bits brute, sans préfixe** (désassemblage `142f2c106`..`142f2c127`). |
| pré-sérialisation d'un message | `FUN_140bbd474` (appelé par `FUN_140bbd1a4`) | écrit dans le tampon `0x1451f9920` : 8 bits `0x80 | genre` (= `1` puis `R(7)` genre), puis 3 × (bit de garde, handle si présent par `FUN_1406d5110`), puis `FUN_1424d80bc` ; range `genre`, `décalage`, `nombre de bits` dans la table. La longueur n'est **jamais écrite dans le flux**. |
| charge | `FUN_1424d80bc` | `vtable[0x60]` du gestionnaire du genre (`*(obj+0x210 + genre*8)`) ; si `FUN_14076cea8()`, un bit et le mot `0x0bcddcba` (forme de débogage, faux sur les 20 films, D-16). |
| lecteur de la vue A | `FUN_14076a1c4` | `{ R(1) ; 0 -> fin ; FUN_14080a9d4 }` (déjà porté par `consumeVueA`, lot 5.14). |
| lecteur d'un message | `FUN_14080a9d4` | `R(7)` genre ; si `< 0x7b` : 3 références gardées (`FUN_1406cf008`, `FUN_1406d3140`), puis la charge par la `vtable[0x68]` du gestionnaire, sur une structure de `vtable[0x10]()` octets ; aucune charge lue si cette taille vaut 0. |

**Établi** : la fin de la vue A est le début de la vue B (aucun bit entre le terminateur et le
premier record), et rien dans le flux ne donne la longueur d'un message. Lire la vue A jusqu'au
bout est donc POSSIBLE par grammaire, mais exige la grammaire de charge (`vtable[0x68]`) de chaque
genre rencontré. La table des gestionnaires est construite à l'exécution
(`DAT_144e61d88 + 0x210`) : le lien genre -> désérialiseur ne se lit pas dans l'image statique
(la numérotation du catalogue de `event_types_catalogue_test.go` n'est pas celle de la trame).

### 4.2 Ce qui reste non localisé après LS (mesuré, `rloc_genres_agreges_ls.tsv`)

- Corpus 20 films : **26 421** (`ls`) / **26 427** (`ls2`) paquets à événements non localisés,
  contre 48 720. Par build (`ls2`) : HI_1_10_0 7 996, HI_1_13_0 6 476, version-33 4 041,
  version-31 2 257, HI_1_11_0 1 707, HI_1_9_0 1 641, HI_1_4_1 1 291, HI_1_8_0 735, HI_1_12_0 283.
- Genre de tête (seul lisible sans charge), corpus : 41 genres distincts ; **56,9 %** des paquets
  ont un genre de tête sans aucune grammaire de charge en Go ; **43,1 %** un genre dont seule la
  tête est lue (`36` tir 9 160, `21` 1 322, `9` 826, …). Genres sans grammaire les plus fréquents :
  `0` (3 842), `82` (2 029), `15` (1 962), `5` (1 554), `38` (1 140), `80` (808), `7` (757).
- Films de l'enquête : 8 951 restants (`ls`) ; 48,9 % en tir (`36`, 4 377).
- **Lecture actuelle** : `consumeVueA` s'arrête au premier corps (`Porte = false`) : 0 paquet de
  la région (ii) n'est localisé par elle (par construction, `frame_vue_messages.go:75`).

### 4.3 La lecture de la vue A les localiserait-elle ? (oracle, mesuré ; gain estimé)

L'oracle (`r_loc_vuea_research_test.go`) prend, sur un paquet encore non localisé, la première
position `e >= 13` précédée d'un bit nul, qui porte un en-tête lisible (DELTA décodé sur un slot
lié, génération du profil, ou NEW de la bande annoncée) et d'où la marche complète ferme la vue C
au bit près ; au plus 256 marches d'essai par paquet. C'est la famille de positions que rendrait
la lecture complète de la vue A ; ce n'est pas un localisateur.

**Contrôle** (300 premiers paquets localisés par la signature 123, par film) : sur les paquets que
`ref` ferme sains (corpus), le premier départ qui ferme EST la signature 2 666 fois, une chaîne de
records qui en part tombe exactement sur la signature 447 fois, NON cohérente 125 fois, aucun
départ dans le plafond 40 fois : **cohérence 95,0 %** (3 113 / 3 278). Films de l'enquête :
1 504 / 1 571 = 95,7 %.

Résultat sur la région (ii) restante (`ls+vueA` contre `ls`, corpus) :

| | Paquets |
|---|---|
| un départ qui ferme est trouvé | 16 066 (60,8 % des 26 421) |
| aucun départ (dont plafond atteint) | 10 350 (8 303) |
| gains supplémentaires de paquets fermés | +16 295, dont 7 412 contredits (45 %) |
| fermés sains en plus | +8 826 (293 457 -> 302 283) |
| records utiles sains en plus | +83 609 (2 758 570 -> 2 842 179) |

Par build, l'oracle trouve presque tout sur HI_1_13_0 (6 071 / 6 476), une majorité sur HI_1_10_0
(6 355 / 7 985), et presque rien sur version-31 (14 / 2 257), version-33 (53 / 4 040), HI_1_4_1
(35 / 1 291) : sur ces builds la marche ne ferme quasiment jamais, quel que soit le départ.

**Réponse** : oui par construction (établi), à condition de porter la charge des genres présents ;
le gain est **borné** (estimé) par l'oracle à environ +8 800 paquets sains et +84 000 records utiles
sains sur le corpus, soit HI_1_13_0 de 73,8 % (`ls2`) à au plus 75,8 % sur le dénominateur fixe.
La réserve sur la borne est forte : 45 % de gains factices, et un contrôle à 95 % (et non 100 %).

## 5. Impact sur les lots

- **LS** : l'ordre de l'enquête ne passe pas le gate sur le site de la cuisson. Il faut deux règles
  sur deux sites, ou une seule règle qui garde la fermeture par NEW de tête avant la signature
  high-frequency :
  - cuisson (`debutDeLaListe`) : 123 strict -> fermeture par NEW de tête -> signature
    high-frequency (+ chaîne de tête) ; mesuré : 0 film en baisse, +18 119 / −2 paquets,
    +169 439 records utiles sains ;
  - killsource et `marchLocalise` (pas de fermeture par NEW de tête) : 123 strict -> signature
    high-frequency -> repli libre du slot 123 ; mesuré : 1 080 morts rendues à la marche, aucune
    valeur changée.
  - **Pas de repli à largeur libre dans `debutDeLaListe`** (−16 673 paquets sains, 12 films en
    baisse).
  - L'effet sur la marche des morts d'objet (`ScanMarchFacts`) n'est pas mesuré ici.
  - Le lot LU (unification) doit donc porter un paramètre d'ordre, ou garder deux appelants
    distincts d'un même cœur ; « posé UNE fois » reste vrai pour le test de la signature, pas pour
    l'ordre.
  - Le dénominateur fixe monte (HI_1_13_0 : 2 931 799) : à recalculer à la vague (D-42).
- **L1b** (région (ii)) : sa borne de bis 1 (+11 540 nets) a été mesurée sur une région de 48 720
  paquets ; LS en localise 45,8 %. Le reste (26 427) ne se lit qu'avec les grammaires de charge des
  genres de la vue A : c'est un chantier de grammaire d'événements (41 genres de tête mesurés,
  table des gestionnaires construite à l'exécution), pas un correctif de localisateur. Borne estimée
  +8 826 paquets sains / +83 609 utiles sains, à 45 % de gains factices près.

## 6. Découvertes (consignées, non traitées)

- **R-LOC-1** Les deux sites du localisateur n'ont déjà pas le même ordre : la cuisson n'a pas le
  repli libre, killsource et `marchLocalise` l'ont (§2.1). L'enquête et D-67 parlent d'une seule
  règle.
- **R-LOC-2** Le slot 123 n'est lié à `ti=4` que dans une partie des chunks des films à objectif
  porté (5 sur 25 pour `c75f33b8`, 11 sur 38 pour `d9781168`) ; ce qu'il porte ailleurs, et ce que
  la signature stricte 123 y lit, n'est pas instruit.
- **R-LOC-3** Signatures high-frequency sur des slots qu'aucun début de chunk ne lie à `ti=4`
  (`1c4c63c2` 2048, 4096, 3904… : 359 paquets ; `e5adf7b2` 22) : liaisons prises en cours de
  chunk, justesse non instruite.
- **R-LOC-4** `51ebbc0f` : la carte gagne +3 378 paquets sains sous LS, killsource ne change pas
  d'un octet (§3.2).
- **R-LOC-5** Le repli à largeur libre du slot 123 dégrade la carte quand on l'ajoute à la cuisson
  (§3.1, `4f77afc1` 22 910 -> 11 051) ; sa valeur dans killsource, où il existe, n'est pas
  remesurée sous la grammaire actuelle.
- **R-LOC-6** `ls` fait mieux que `ls2` en records utiles sains sur plusieurs films HI_1_13_0
  (`d9781168` 281 804 contre 277 374, `c46ef9d1` 470 497 contre 459 514) : l'ordre entre fermeture
  par NEW de tête et signature high-frequency n'est pas décidé par la grammaire ; `ls2` est le seul
  ordre mesuré sans baisse.
- **R-LOC-7** `archlint` est ROUGE sur l'arbre de base `fe18bf67c`, hors de ce chantier :
  `TestNoExpiredTODO` signale `internal/api/handlers/json_huma_coverage_test.go:34`
  `TODO(expiry:2026-10-01)` échu (le TODO est présent tel quel à `fe18bf67c`). Tous les autres tests
  d'`archlint` passent (§7).

## 7. Gate (rejoué le 2026-10-02 depuis `apps/go-api` du worktree)

- `gofmt -l` sur `r_loc_ls_research_test.go`, `r_loc_vuea_research_test.go` et les deux copies de
  `r_loc_overlay/` : vide.
- `go vet -tags=research ./internal/games/halo_infinite/film/internal/grammar/` : exit 0.
- `go vet -tags=research -overlay=<r_loc_overlay/overlay.json>` sur `grammar`,
  `facts/killsource`, `cmd/killsource`, `research/cmd_fermeture` : exit 0.
- `go test -count=1 ./internal/archlint/` : **FAIL**, uniquement `TestNoExpiredTODO` (TODO échu le
  2026-10-01 dans `internal/api/handlers/json_huma_coverage_test.go`, fichier non touché, R-LOC-7) ;
  `go test -count=1 -skip '^TestNoExpiredTODO$' ./internal/archlint/` : `ok` (29 s), ratchets de
  taille et de tag `research` compris.
- Sondes : 460 et 174 lignes, `//go:build research` en ligne 1, noms sans suffixe de garde.
- `git status` : seuls des fichiers neufs (sondes, `r_loc_overlay/`, `r_loc_ghidra/`,
  `r_loc_tsv/`, cette note).

## 8. Fichiers

- Note : `.ai/V7.5/film_re/campagne_grammaire_2026-10-01/R_LOC.md`.
- Sondes : `apps/go-api/internal/games/halo_infinite/film/internal/grammar/r_loc_ls_research_test.go`,
  `…/r_loc_vuea_research_test.go`.
- Surcouche : `.ai/V7.5/film_re/campagne_grammaire_2026-10-01/r_loc_overlay/` (`object_deaths_march.go`,
  `walk.go`, `overlay.json` aux chemins de CE worktree : à réécrire pour un autre worktree).
- Ghidra : `.ai/V7.5/film_re/campagne_grammaire_2026-10-01/r_loc_ghidra/` (décompilations de
  `FUN_142f2c050`, `FUN_142f2c3b0`, `FUN_140bbd474`, `FUN_140bbd1a4`, `FUN_1424d80bc`,
  `FUN_14076a1c4`, `FUN_14080a9d4`, `FUN_14080ada0`, `FUN_1406d60f4`, `FUN_14076ba54`,
  `FUN_14076be40`, désassemblage de `FUN_142f2c050`, références aux tables).
- TSV : `.ai/V7.5/film_re/campagne_grammaire_2026-10-01/r_loc_tsv/` — brutes de la passe 1
  (`rloc_variantes`, `rloc_passes`, `rloc_slots_hf`, `rloc_genres`, `rloc_oracle`, `rloc_controle`,
  `rloc_registre`), de la passe 2 (`*_passe2_ls2`), agrégats (`rloc_agregats_groupe_variante`,
  `rloc_par_build_corpus20`, `rloc_par_film_ref_ls_ls2`, `rloc_genres_agreges_ls`,
  `rloc_denominateur_fixe_max`), killsource (`rloc_killsource_comparaison`,
  `rloc_killsource_morts`), carte officielle sous surcouche (`rloc_carte_v2_surcouche_ls_films`).
