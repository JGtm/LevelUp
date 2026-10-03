# NOTE 5.3 — LES ETATS DE MOUVEMENT DU SPARTAN DANS LE FILM

> Lot 5.3 (post-chantier, RECHERCHE puis port si prouve). Branche `feat/decfilm-53`, base
> `6e86db356`. Question de l'utilisateur (2026-09-19) : « on a les evenements de joueurs comme
> les slide, crouch, sprint et saut ? ».
>
> **ETAT : point 1 (L'ECRIVAIN) FAIT. Point 2 (PREUVE SUR FILM) EN COURS — sa conclusion de
> cadence a ete RETIREE le 2026-09-21 (§ 2 quater), l'instrument n'etant pas etalonne.** L'ecrivain aux § 1 a 2.9,
> D1 sur mini-bobines au § 2.4 bis, **la mesure sur `bfecd02b` et `4f77afc1` au § 2 ter**.
> Le port (5.3.3) N'EST PAS LANCE : il attend le retour du pilote. Aucune base DuckDB n'a ete
> ouverte a aucun moment ; les films ont ete lus UN A LA FOIS.

---

## PASSATION 5.7 — A LIRE EN ENTIER AVANT DE TOUCHER UN OCTET (2026-09-21)

> Ecrite par l executant sortant a ~760 k jetons. Dix points. **La vraie livraison du lot est la
> PORTE DE SPECULATION** (5.7.4) : elle corrige un defaut qui gonflait `stances[]` d un facteur 19
> a 79. Le reste est de la recherche, et elle s arrete a un endroit PRECIS que le point (2) donne
> a l adresse pres.

### (1) L ETAT EXACT

Branche `feat/decfilm-57`, worktree `LevelUp-wt-decfilm-57`, base `7af38c44d`. Arbre propre, rien
pousse.

| sha | ce qu il apporte |
|---|---|
| `07ea5cf29` | 5.7.1 + 5.7.2 : l ecrivain (sprint = bit 45, `i59` = capacite, `i55` = union), la mesure, et la decouverte de la porte polluee. **Aucun octet de production** |
| `3830ba3a5` | fusion de `feat/recherche-decodeur-film` (lot 5.8, web seul) |
| `c234b9589` | 5.7.2 (suite) : les cinq instants Theater NON livres, avec la raison. Documents |
| `8706fdb3f` | **5.7.3 — PORT** : les quatre charges de l union `ti=35 i55` (D1 fermee). `grammar-2026-09-21.3`, `killsource-2026-09-21.3` |
| `ee183ce39` | **5.7.4 — PORT** : la porte de speculation. `grammar-2026-09-21.4`, `killsource-2026-09-21.4` |
| `5af778ac7` | 5.7.5 : oracles physiques, scores, chaine de donnees. **Aucun octet de production** |

**CE QUI EST PORTE — TROIS CHOSES, ET RIEN D AUTRE.**

1. **LA PORTE DE SPECULATION** (`observateur.go` : `neutraliserEtatsDeMouvement`, appelee par les
   deux neutralisations existantes ; `object_deaths_march.go` : les trois localisateurs la
   declarent). Deux garde-rails dans `etats_mouvement_porte_guard_test.go`.
2. **L UNION D ETAT PHYSIQUE D `i55`** (`components_biped_posture.go`), quatre charges, largeurs
   toutes relues chez l ecrivain.
3. Rien d autre. `SchemaVersion` reste **65**.

**CE QUI N EST PAS PORTE, ET C EST DELIBERE** : ni `sprint`, ni `jump`, ni `clamber` dans
`stances[].kind` ; les etiquettes 4, 5 et 6 d `i59` (leur grammaire est LUE — § 4 D3 — mais la
decision du 5.3.3-c de s arreter tient) ; `PlayerGameEventSmall` (type 82).

### (2) LE SPRINT — LA CHAINE REPREND EXACTEMENT ICI

**LE CONSOMMATEUR DE DEPART, ETABLI** :

```
transition_conditions_is_sprinting_tlg          (graphe d animation : une CONDITION D ETAT)
SpartanAbilityIsSprinting @1436f7170            (enregistre par FUN_140fe6664, avec quatre soeurs)
  -> FUN_142a0c70c :  obj = FUN_140477618(&poignee, 1)
                      return (*(u64*)(obj + 0x8b8) >> 0x2d) & 1     <- BIT 45
```

**LA QUESTION QUI RESTE, ET ELLE EST UNIQUE : QUI POSE LE BIT 45 ?**

**LES SEULS CANDIDATS RESTANTS** — les fonctions qui assignent le mot ENTIER depuis un REGISTRE.
Les deux premieres sont a lire en premier parce qu elles posent le mot ET testent un de ses bits,
donc elles le RECALCULENT :

| fonction | instruction | note |
|---|---|---|
| **`FUN_1409aac4c`** | `1409aadec  MOV [RBX+0x8b8], R8` | teste aussi `0x400` deux fois et `BTS 0xf` — **COMMENCER ICI** |
| **`FUN_140775a24`** | `14077639d  MOV [RDI+0x8b8], RAX` | **teste le bit 45 lui-meme** (`140775e13`) — **PUIS ICI** |
| `FUN_1406730c4` | `140673280` et `140673299`, `MOV [RAX+0x8b8], RCX` | |
| `FUN_1406c7ad4` | `1406c7b45  MOV [RAX+0x8b8], RDX` | |
| `FUN_1406c9b1c` | `1406ca467  MOV [RSI+0x8b8], RCX` | l applicateur d etat replique — son seul autre contact est le **bit 54** |
| `FUN_1407184ac` | `14071922f  MOV [R14+0x8b8], RCX` | |
| `FUN_140776790` | `140776b5d  MOV [RSI+0x8b8], RCX` | |
| `FUN_140803c54` | `140803dc3  MOV [RSI+0x8b8], RDX` | |
| `FUN_140805064` | `140805089  MOV [RCX+0x8b8], RDX` | |
| `FUN_1408dcd7c` | `1408dce51` et `1408dce94`, `MOV [RDI+0x8b8], RDX` | |
| `FUN_140970614` | `14097069f  MOV [RSI+0x8b8], RCX` | |
| `FUN_1409ab28c` | `1409ab555  MOV [RDI+0x8b8], RAX` | |
| `FUN_140a19150` | `140a194c6  MOV [RBX+0x8b8], RCX` | |
| `FUN_140a1e2e4` | `140a1e305  MOV [R11+0x8b8], RAX` | |
| `FUN_140adfa5c` | `140adfb01` et `140adfb68`, `MOV [RBP+0x8b8], RDX/R8` | |

Deux ecrivains du mot sont ECARTES PAR LEUR FORME : `FUN_1405659f0` (`1405666fe`, immediat `0xf`)
et `FUN_140b39604` (`140b39738`, `MOV word ptr` — seize bits, donc bits 0 a 15). **Il reste
QUINZE fonctions, une lecture par fonction.**

**CE QUI EST ECARTE, ET LA MESURE QUI L ECARTE** (ne pas refaire) :

| mecanisme | mesure |
|---|---|
| ecriture partielle a l octet | `[x+0x8bd]` porte le bit 45 : **0 reference** dans l image (`0x8bc` en a **89**, `0x8be` **42**) |
| `BTS`/`BTR` a rang immediat | **26** sur ce mot (14 `BTS`, 12 `BTR`), rangs 8, 9, 0xb, 0xd, 0xe, 0xf, 0x11, 0x12, 0x16, 0x18, 0x1a, 0x1d, 0x1e — **aucun a 0x2d** |
| masque calcule | aucun des assignateurs ne charge `1<<45` (`0x200000000000`) ; le masque n apparait qu en `TEST` ; **0** `BTS`/`BTC` a rang REGISTRE, **0** `SHLX` sur cet offset |
| copie de structure | les SEULES copies de 16 octets a `0x8b8` sont `FUN_141576070` et `FUN_1415761e0` — les constructeurs de copie du **widget d interface `TwoToneMeterQuad`** (objet `0x918` o, vtable `PTR_FUN_143849e88`, enregistre par `FUN_1400eed50`, chaine `fui_meter_two_tone_quad_widget`). **COLLISION D OFFSET ENTRE DEUX CLASSES, pas une copie d unite** |
| la chaine d etat replique | **aucun des NEUF applicateurs** de `FUN_1406c9b1c` ne touche `obj+0x8b8` : `FUN_140a10970`, `FUN_140c85028`, `FUN_1406c72e8`, `FUN_140a10c54`, `FUN_140a10b64`, `FUN_140a10a7c`, `FUN_1404d4c28`, `FUN_140a10998`, `FUN_1406ca5f0` |

Et le seul contact de `FUN_1406c9b1c` lui-meme est le **bit 54**, dont la source est un champ de
l objet VIVANT (`FUN_140719698(obj) + 0x120` / `+0x121`), pas du tampon replique.

**14 SITES DE TEST DU BIT 45**, utiles pour borner le sens du drapeau : `FUN_140775a24`,
`FUN_140776c4c`, `FUN_1407754cc`, `FUN_1407fa018`, `FUN_1407fba24`, `FUN_1407fcb30`,
`FUN_140800ea8`, `FUN_1406de83c`, `FUN_14060f81c`, `FUN_140611238`, `FUN_1406735e0`,
`FUN_1406dba04`, `FUN_140897b6c`, `FUN_14089dd7c`.

### (3) LE SAUT — LA CHAINE A FAIRE, ET PAR QUEL BOUT

**L ETAT AERIEN EST UNE CLASSE, ET ELLE EST NOMMEE.** L image ne porte que trois chaines
`c_biped_*` :

| classe | chaine | enregistrement reflechi | descripteur |
|---|---|---|---|
| `c_biped_ground_state` | `143e2b730` | `FUN_1431be7cc` | — |
| **`c_biped_airborne_state`** | **`143e2bfc0`** | **`FUN_1432226c0`** | `_DAT_144815240` |
| `c_biped_vehicle_state` | `143e2bfd8` | `FUN_143222af4` | `_DAT_144815260` |

`c_biped_airborne_state` fait **0x6c octets** : six booleens (`0x0c`, `0x0d`, `0x0e`, `0x0f`,
`0x10`, `0x11`), des flottants (`0x14`, `0x18`, `0x1c`, `0x38`, `0x3c`, `0x40`, `0x44`, `0x64`,
`0x68`), deux vec3 (`0x20`, `0x2c`), deux champs `0xc010001` (`0x48`, `0x54`), deux shorts
(`0x60`, `0x62`).

**LA QUESTION A POSER, DANS CET ORDRE** :

1. **Qui INSTANCIE / transitionne vers `c_biped_airborne_state` ?** Point de depart : les xrefs du
   descripteur reflechi (`_DAT_144815240`) et la vtable de la classe ; puis remonter aux appelants
   du constructeur. C est le maillon qui manque.
2. **Quelle donnee declenche la transition ?** Si c est un champ replique, il est forcement dans
   les CINQ composants denses (cf. point 4) — donc position, vitesse, visee, tick ou bouclier.
3. **`i55` est le DISCRIMINANT de l union d etat physique, et ses lecteurs sont nommes** :

```
FUN_142f0293c  thunk : FUN_141015c90(obj + 0x12b4)  [0 bit]  puis FUN_142f1f630
FUN_142f1f630  tag = R(2)  puis FUN_141fd997c(tag, ctx)
FUN_141fd997c  POSE l octet de genre dst+0x2c = 1, 2, 3 et appelle :
               tag 0 -> FUN_142f265dc   (aucun octet de genre pose)
               tag 1 -> FUN_142f25a3c   (genre 1)
               tag 2 -> FUN_142f263ac   (genre 2)
               tag 3 -> FUN_142f264f4   (genre 3)
```

La correspondance tag -> classe **n est PAS etablie** : aucune chaine ne s attache a l octet de
genre, et la mesure ne tranche pas (47 lectures liees au bipede au mieux). C est en remontant
depuis la CLASSE (point 1) qu on la nommera, pas en notant le tag.

**LA VERITE TERRAIN EST DEJA POSEE, ET ELLE EST SOLIDE** : **H = 0,85 m**, duree de montee
**0,467 s** (`bfecd02b`) et **0,466 s** (`4f77afc1`), pic **x 10,7** et **x 3,9** au-dessus de ses
voisins. Elle sert a VALIDER tout candidat qu on trouvera, et elle a deja servi a calibrer
l echelle d `i1` (cf. point 6).

### (4) LES CANDIDATS NOTES — ET CEUX QUI NE L ONT PAS ETE

**NOTES : 43 champs sur `bfecd02b` (1 886 instants), 54 sur `4f77afc1` (9 310).** Relus a leur
`StartBit` sur les records RETENUS, ce qui leur donne un slot ET un instant.

`i18.mot32.porte` et ses **32 bits** · `i57.etiquette=<v>` (v = tag − 1), `i57.sous=<v>`,
`i57.reference` · `i59.tag=<v>`, `i59.interne=<v>` · `i55.tag=0..3` · `i54.amorce`, `i54.second` ·
`i29.accroupi`.

| etiquette | meilleur candidat | precision | rappel |
|---|---|---|---|
| amorce de saut (`bfecd02b`, 136) | `i57.etiquette=-1` | **11,3 %** | 25,7 % |
| amorce de saut (`4f77afc1`, 830) | `i59.tag=0` / `i57.etiquette=-1` | **11,9 %** | 20,7 % |
| plateau 2,75-2,99 (`bfecd02b`, 111) | `i57.etiquette=-1` | **13,8 %** | 39,6 % |
| plateau 2,50-2,74 (`bfecd02b`, 50) | `i59.tag=2` | 1,0 % | 6,0 % |

Seuil de nommage : 90 % dans les DEUX sens. **Jamais atteint, sur aucune des six notations.** Les
11-14 % valent environ sept fois le hasard (les fenetres de saut couvrent 1,5 % du film) : il y a
un signal faible, pas une identite.

**CE QUI N A PAS ETE NOTE, ET C EST LA PREMIERE CHOSE A FAIRE** :

1. **LE MASQUE DE 32 DRAPEAUX DE `PlayerGameEventSmall`** (type 82, **578** en tete de paquet sur
   `bfecd02b`). Il n est PAS porte. Grammaire relevee, deux largeurs manquantes :
   `FUN_14080add8` = `FUN_14080b30c` [0 bit] + `FUN_14080ae70` [`R(32)` + `R(8)` +
   **`FUN_14080b1b8(ev+0x10)`** + **`FUN_14080b034(ev+0x78)`**, ces deux-la NON relevees] +
   `FUN_14080ae28` = **32 x `R(1)` empaquetes un a un**. Le pont vers le slot existe deja et il est
   valide : une reference de **domaine 4** + **512** (`zoom_events.go`, 6/6 contre Theater a moins
   de 1,2 s ; et 512 = `0x200` = la base de la categorie 4 de `FUN_140d10bb0`). **C est le seul
   canal DENSE et non explore.**
2. **LES CHARGES d `i55`** : le port du 5.7.3 publie le handle `R(15)`, la direction `R(19)`, le
   mot de 32 bits et le sous-genre — **aucun n a ete note comme candidat**, seul le tag l a ete.
3. **`i60 simulation-state` et `i63 biped-action`** : jamais relus comme candidats.
4. **LES CINQ COMPOSANTS DENSES EUX-MEMES.** La visee (`i21`, 65,29 %) et le bouclier (`i5`,
   36,31 %) n ont jamais ete confrontes aux etiquettes. Si un etat par instant existe, il est
   la — c est le point (4) du recensement qui le dit.

### (5) LES INSTRUMENTS, ET LES DEUX ORACLES QUI LES GARDENT

Tous `_test.go` sous `//go:build research`, paquet `grammar`, aucun octet de production.

| fichier | role |
|---|---|
| `mouvement_5_7_posture_research_test.go` | **LA PASSE PARTAGEE** `m57Passe` (marche a 3 vues, carte installee, oracle de contenu) + le test de la montee |
| `mouvement_5_7_sejour_research_test.go` | sejours et matrice de transitions des tags d `i55` |
| `mouvement_5_7_aval_research_test.go` | ce qui suit `i55` dans le record (clef = le NOM, jamais l index) |
| `mouvement_5_7_phase_research_test.go` | **L ORACLE DE LA PORTE** : `HOOK` contre `Trace.Comps`, par phase |
| `mouvement_5_7_retenus_research_test.go` | relecture a `StartBit` (controle : 0 ecart de largeur sur 75 977) |
| `mouvement_5_7_balayage_research_test.go` | **`ScanMovementStates`, la fonction de PRODUCTION** |
| `mouvement_5_7_sprint_research_test.go` | distribution de vitesse au sol, globale et par vie |
| `mouvement_5_7_5_oracles_research_test.go` | **H et les plateaux** (verite terrain physique) |
| `mouvement_5_7_5_candidats_research_test.go` | scores precision/rappel **et le recensement de DENSITE** |

```bash
export GOCACHE=/c/Users/Guillaume/Downloads/Scripts/LevelUp-wt-decfilm-57/.gocache
export PATH=/c/msys64/ucrt64/bin:$PATH CGO_ENABLED=1
cd apps/go-api
MOUV57_FILM=<worktree>/data/cache/film_chunks/bfecd02b \
MOUV57_CARTE=snowbound \
MOUV57_BORNES='C:\Users\Guillaume\Downloads\Scripts\LevelUp-wt-decfilm-57\data\titles\halo_infinite\reference\map_quant_bounds.json' \
  go test -tags=research -count=1 -v -timeout 60m \
    -run '^TestMouvement57' ./internal/games/halo_infinite/film/internal/grammar/
```

`4f77afc1` se joue avec `MOUV57_CARTE='flood gulch'` (cf. piege 2).

**DEUX ORACLES, ET ILS SONT OBLIGATOIRES AVANT TOUTE CONCLUSION.**

1. **L ETALON DE CONTENU** : `i21` doit rester au-dessus de 50 % (l instrument REFUSE de publier en
   dessous). References : `bfecd02b` `i0` 85,5 · `i1` 77,5 · `i21` 65,3 · `i25` 97,1 % pour
   **97 345** records `ti=35` (6 desyncs) ; `4f77afc1` 72,6 · 63,8 · 69,6 · 98,2 % pour **321 335**
   (56 desyncs).
2. **`Trace.Comps` COMME ORACLE DE LA PORTE** : ce que la porte publie doit EGALER ce que les
   records rendus declarent, par NOM de composant. References (`bfecd02b` / `4f77afc1`) : `i29`
   **76 / 236**, `i55` **52 / 287**, `i62` **56 / 230**, `i54` **321 / 1 033**, `i18`
   **117 / 515**, `i1` **79 471 / 220 844**. Une inegalite = une porte qui publie des essais.

### (6) LES PIEGES — CHACUN A COUTE UNE MESURE FAUSSE

1. **LES LARGEURS D AXE DE LA CARTE S INSTALLENT A LA MAIN** : `NewFilmContextForMap` PUIS
   `PoserLargeursObjetDuMondeDepuisDecoupage`. Sans le second geste, `i0` lit cinq bits de trop
   par record et tout ce qui suit est du bruit (lecon du 5.3.5).
2. **LA CARTE DE `4f77afc1` EST `flood gulch`, PAS `cliffhanger`.** Sous la mauvaise carte la
   marche rend **22 713** records `ti=35` au lieu de **321 335**. La carte se lit dans le manifeste
   du film, jamais par defaut.
3. **TOUTE PORTE DE PUBLICATION NEUVE DOIT S INSCRIRE DANS `neutraliserEtatsDeMouvement`** (ou
   dans les neutralisations qui l appellent), sinon elle publiera les essais d alignement de
   `marchLocateStrict` et de `deltaBodyTrial` — facteur **14 a 152** selon la rarete du composant.
   C est le defaut que ce lot a corrige, et il ne casse AUCUN test quand il revient : il publie
   davantage, et « davantage » ressemble a « mieux ».
4. **LES UNITES D `i1` : DEUX CHOSES DIFFERENTES.** Le facteur **0,240** du 5.3.5 est le rapport
   (deplacement de la meme vie) / (vitesse decodee) ; il a valide la DISPERSION, jamais l ECHELLE
   (« aucune unite supposee »). C est **H = 0,85 m** qui valide l echelle, parce qu une hauteur de
   saut est une grandeur absolue du jeu. Ne pas confondre les deux, et ne pas « corriger » la
   vitesse par 0,24.
5. **`Trace.Mask` N EST POSE QUE SUR LE CHEMIN NEW/IMAGE-CLE.** Compter les composants par le
   masque sur le chemin delta rend **63** la ou la porte tire **3 908** fois. Compter par **NOM**
   dans `Trace.Comps`. (Premiere version de la sonde d aval : 52 au lieu de 3 784.)
6. **LES FIXTURES DE CONTRAT REJOUENT DES ENTREES FIGEES** (`inputs_<film>.bin.gz`,
   `REPLAYINPUTS24`) : un changement de decodeur ne les traverse PAS tant qu on n a pas refige les
   entrees par leur porte (`GoldenBuildsRegenerate` + `REPLAY_FILM_CACHE`, et
   `GoldenInputsRegenerate` + `REPLAY_FILM_DIR` pour `000d5950`), puis les goldens d assemblage,
   puis les fixtures. Deux portes en serie, et la premiere est facile a oublier.
7. **LES HEREDOCS BASH ECHOUENT** sur du contenu a accents et backticks : passer par un fichier
   Python ou par l outil d ecriture.
8. **`golangci-lint` refuse de tourner en parallele** : isoler `GOLANGCI_LINT_CACHE`.

### (7) HORS PERIMETRE — CONSIGNE AU § 4, NE PAS TRAITER

**D3 (5.7)** les etiquettes 4, 5 et 6 d `i59`, desormais portables avec des feuilles que le depot
a deja (la decision du 5.3.3-c de s arreter tient) · **D4 (5.7)** le masque de 32 drapeaux de
`PlayerGameEventSmall` · **D5 (5.7)** `MobilityActionHook`, la porte HISTORIQUE d `i54`, reste
vivante pendant les essais et les calques de capacite en heritent · **D6 (5.7)** le producteur du
bit 45 (c est le point 2 ci-dessus) · **D10 (5.3)** `marchViews = 8` contre trois vues ·
**D11 (5.3)** 24 % des paquets a liste pleine non localises. D1 et D2 (5.7) sont TRAITEES.

### (8) L ENVIRONNEMENT

Worktree `C:/Users/Guillaume/Downloads/Scripts/LevelUp-wt-decfilm-57`, branche `feat/decfilm-57`.
`GOCACHE=<worktree>/.gocache`, `PATH=/c/msys64/ucrt64/bin:$PATH`, `CGO_ENABLED=1`,
`GOLANGCI_LINT_CACHE=<worktree>/.golangci-cache`.

**AUCUNE BASE DuckDB. UN FILM A LA FOIS.** Films autorises : `bfecd02b` (snowbound) et `4f77afc1`
(flood gulch). Jonctions `film_chunks` et `film_manifests` posees (**1 598 entrees**), JAMAIS
retirees, jamais de `git worktree remove`. `replay-corpus-gate` INTERDIT ; le gate avec decodage
est `replay-equiv -films bcb6d393` SANS `-update`. Corpus des 19 et re-figeage des references :
geste du pilote, a la fin. Ghidra en lecture seule sur `127.0.0.1:8089`
(`/decompile_function`, `/search_instructions`, `/get_xrefs_to`, `/read_memory`,
`/search_strings`) — **ne PAS appeler `/disassemble_function`, il rend 162 Mo**.

### (9) LE FORMAT DU COMPTE RENDU

Court, par question, avec **les tableaux chiffres et leurs DENOMINATEURS**. Dire ce qui est
mesure, ce qui est refute, ce qui n est pas fait. Un negatif s ecrit avec ses chiffres ; une
hypothese abandonnee se nomme. **Arret sur toute perte.** Et quand une mesure contredit le depot,
c est l INSTRUMENT qu on suspecte en premier — ce lot en est la demonstration : la « decision de
valeur » du 5.7.4 n etait qu une porte mal inscrite.

### (10) LA DOCTRINE DE L UTILISATEUR — ELLE FAIT AUTORITE, ET ELLE A DEJA CORRIGE CE LOT DEUX FOIS

1. **« TOUT EST ENREGISTRE DANS LE FILM. »** Le Theater sait quand un joueur se met a courir rien
   qu en lisant le film ; il ne refait pas le match en live, ca rendrait impossible la lecture d un
   match ancien. **Consequence de methode** : « aucun deserialiseur n ecrit ce champ » n est JAMAIS
   une conclusion — c est un maillon manquant.
2. **LA CHAINE SE REMONTE A REBOURS, DEPUIS LE CONSOMMATEUR.** Ce qui affiche / anime / decide ->
   la memoire qu il lit -> qui l ecrit -> d ou vient la donnee -> quel composant, quel champ,
   quelle largeur. **Une lecture d ecrivain ne remonte pas un flux de donnees** : c est l erreur
   du 5.7.1, et le 5.7.5 la corrige (la chaine de l accroupi, complete, en est la demonstration :
   `FUN_142ed42a8` -> `etat+0x7e8`/`0x7ec` -> `FUN_140a10970` sous le bit 29 du masque ->
   `FUN_140c60e1c`/`FUN_1408b2230` -> objet).
3. **JAMAIS DE NEGATIF.** Un candidat absent est « non trouve, voici les scores » ; une chaine qui
   se perd est « non trouve a `<adresse>` », avec la liste de ce qui reste a lire.
4. **LES ORACLES PHYSIQUES DU JEU SONT DES VERITES TERRAIN** : tous les Spartans sautent la meme
   hauteur et courent a la meme vitesse. On etiquette PAR LA PHYSIQUE d abord, on cherche le champ
   ensuite.
5. **UN MAILLON PAR LECTURE**, but ecrit avant. Le quota se tient comme ca.

---

## PASSATION 5.3.3 — A LIRE EN ENTIER AVANT DE TOUCHER UN OCTET (2026-09-21)

> Ecrite a la main du pilote sortant, a ~770 k jetons. Dix points. Le lot est un lot de
> GRAMMAIRE, et sa methode a ete CORRIGEE EN ROUTE : cinq hypotheses ont ete refutees, quatre
> par la mesure, une par l ecrivain. **Ne les refais pas** — le point 6 les nomme.

### (1) L ETAT EXACT

Branche `feat/decfilm-53`, worktree `LevelUp-wt-decfilm-53`, base du lot `6e86db356`, fusion
`a289e9c1a` du lot 5.4 (`5fd6f02c3`, `grammar-2026-09-20.2`). Arbre propre. Rien pousse.

| sha | ce qu il apporte |
|---|---|
| `6901752f9` `d0e3f42c1` `97f14743d` | 5.3.1 : l ecrivain des etats, D1 mesuree sur mini-bobines |
| `40213547e` `76cd4f714` | 5.3.1 bis/ter : canal d evenements, `i56`, l enum, l objection tranchee |
| `7fccdecb8` `9a4962173` `3e81a2a20` `6405237fe` | 5.3.2 : preuve sur film, conclusion de cadence RETIREE, etalon Rosette, hypothese du pilote confirmee |
| `834527f99` `2cea5cb56` | 5.3.3.0 : liste des fautifs corrigee, journal de fusion |
| `9d94dc8b3` `613849532` `4fe52d984` `552db474b` | 5.3.3.1 a .4 : quatre negatifs (monde, largeur d ID, base d ID, preambule) |
| `2a018adf1` | 5.3.3.5 : `i60`, gate TENU en mesure |

**CE QUI EST PORTE** : rien de neuf. **AUCUN OCTET DE PRODUCTION N A ETE TOUCHE DE TOUT LE
LOT.** Tous les fichiers Go ajoutes sont des `_test.go` sous `//go:build research`, plus le
paquet `film/research/mouvement/` (hors couche, declare dans `archlint`).

**CE QUI N EST PAS PORTE, ET C EST LE TRAVAIL** : le point (2).

### (2) LE CHANGEMENT DE PRODUCTION A FAIRE — `SimStateComplet`

`i60 simulation-state` est **deja porte integralement**, queue comprise (lot R7-b, 2026-08-17).
Ce qui le fait desyncer est un DRAPEAU :
`internal/games/halo_infinite/film/internal/grammar/profil_balayage.go:136`
(`SimStateComplet bool`, defaut `false`), rendu tel quel comme `ported` par
`dispatch_biped.go:112`.

Son **critere de bascule est ECRIT** dans son commentaire : « que le chemin absolu d `i0` tire
ses trois largeurs de la CARTE du match ».

**LE CHANGEMENT JUSTE** : lier le drapeau a la PRESENCE des largeurs de carte, dans
`ResolveProfile` (`internal/games/halo_infinite/film/internal/grammar/profile.go:40`,
signature `ResolveProfile(film *source.Film, entry *profile.MapQuantEntry) profile.Profile`) —
vrai quand `entry != nil`, faux sinon.

**NE PAS basculer le defaut GLOBAL** : `NewFilmContext` (auto-detecte, sans carte) sert les
enveloppes `ScanFilm*(dir)`, ou les largeurs d axe ne viennent PAS de la carte. Un defaut
global casserait ces appelants — c est exactement ce que le critere interdit.

**GATES DE CE COMMIT** : `grammar.Rev` monte par l EMPREINTE (racine `film/internal/grammar/`)
avec son entree de CHRONIQUE (`rev_chronique.go`, la derniere est `grammar-2026-09-20.2`, donc
`.3`) ; ratchet 0.A.3 (`keyframe_closure.golden`) sans ligne en BAISSE ; `replay-equiv
--films=4f77afc1` SANS `-update`, **0 perte hors artifact** ; et l ORACLE DE CONTENU :
**records `ti=35` >= 11 228 et `i21` ~ 64 %** (mesures ci-dessous).

**MESURE DEJA FAITE, a reproduire** (carte `snowbound`, drapeau leve sur le profil de la
marche) : trames saines 40,5 % -> **40,6 %** · records `ti=35` 11 150 -> **11 228** · `i21`
64,3 % -> **64,2 %** · desyncs reelles 3 130 -> **3 087** · **fautif `i60` 38 -> 0** · `i29` lu
7 -> **14** · `i62` lu 6 -> **14**.

### (3) LA LECTURE GHIDRA A FAIRE — LE SERIALISEUR DE TRAME DELTA

**Point de depart** : les APPELANTS de `FUN_1406d3140` en categorie **7** dans la boucle
d ECRITURE (le pendant de `readRecordID`). Ghidra est joignable en lecture seule sur
`127.0.0.1:8089` (`/decompile_function?address=0x...`, `/get_xrefs_to`, `/read_memory`) ;
`HaloInfinite.exe` est analyse, 311 103 fonctions.

**CE QU ON CHERCHE** : les CHEMINS DE RECORD que `DecodeFrameRecords` ne modelise pas. Le
decodeur connait `recNew`, `recDel`, `recDelta` et `recEnd` ; **12 316 paquets sur 25 958
echouent des leur PREMIER record**, avec un slot qui n existe pas (4 568 slots distincts, dont
**0,6 %** seulement vus dans un record sain — ce sont des identifiants lus dans du bruit).

**LES QUATRE NEGATIFS DEJA ACQUIS — NE PAS LES REFAIRE** : ce n est ni l amorcage du monde, ni
la largeur d ID (13, categorie 7, confirmee chez l ecrivain), ni la base d ID (0 pour la
categorie 7), ni le preambule (`0xA0` ecrase les deux populations, 98,34 % contre 99,79 %).

### (4) `i59` ET `i57` — CE QUE L ECRIVAIN DIT DEJA

`ecs_table.tsv` (`internal/grammar/testdata/`) les declare `partiel`, et donne la raison :

- **`i57 biped-spartan-ability`** (`FUN_142f02810`) : `R(2)` etiquette ; etiquette 1 -> `R(2)` +
  `R(24)` reference ; **etiquette 3 -> `FUN_142f262d4`, gardee par des octets d etat RUNTIME**.
  Une desync PROPRE y est la bonne reponse, et elle doit rester ECRITE.
- **`i59 biped-spartan-ability-non-predicted-state`** (`FUN_142f02994`) : etiquette + corps ;
  etiquette 3 = evenement de grappin PAR PAIRES a 0,150 s, position absolue quantifiee aux
  largeurs de la CARTE, queue `R(3)` sous `param_4 = 2`. `ported=false` sur `Zero3 != 0` ou
  `Inner` hors `{1,2}`.

**METHODE** : une seule lecture d ecrivain par composant, but ecrit avant. Chercher **jusqu ou
l ecrivain rend le composant calculable DEPUIS LE FLUX** ; ce qui depend d un octet RUNTIME
reste une desync propre, et cela se dit dans la note plutot que de se contourner.

Mesures actuelles (bfecd02b, sans le drapeau) : `i59` fautif **19** fois, `i57` **12** fois.

### (5) LA MESURE FINALE DES ETATS

Instrument : `grammar/mouvement_5_3_2d_*` (quatre fichiers, voir point 7).

```bash
export GOCACHE=/c/Users/Guillaume/Downloads/Scripts/LevelUp-wt-decfilm-53/.gocache
export PATH=/c/msys64/ucrt64/bin:$PATH CGO_ENABLED=1
cd apps/go-api
MOUV532D_FILM=<...>/data/cache/film_chunks/bfecd02b \
MOUV532D_CARTE=snowbound \
MOUV532D_BORNES='C:\Users\Guillaume\Downloads\Scripts\LevelUp-wt-decfilm-53\data\titles\halo_infinite\reference\map_quant_bounds.json' \
MOUV532D_SIMSTATE=1 \
  go test -tags=research -count=1 -v -timeout 60m \
    -run '^TestMouvement532Trame$' ./internal/games/halo_infinite/film/internal/grammar/
```

`MOUV532D_IDLOW` existe aussi (rejouer la calibration de largeur d ID ; **ne pas s en servir
pour corriger quoi que ce soit**, cf. point 6).

**A RENDRE** : trames saines · desyncs par composant · **cadence de `i29` par slot
(records/s)** · intervalles d accroupi par slot (progression > 0,5) · `i62` glissade lue
(compte + intervalles) · `i54` datee (instants) · `i18` et `i55` ventiles · et **les 5 instants
par etat en TEMPS DE BARRE THEATER** (accroupi, glissade, action de mobilite, montee/saut), sur
`bfecd02b`.

**L ORACLE DE CONTENU EST OBLIGATOIRE A CHAQUE PASSE** : `i21` doit rester autour de 64 % et
les records `ti=35` au-dessus de 11 228. L instrument REFUSE deja de publier une cadence si
`i21` tombe sous 50 % — **ne desactive pas cette garde**, elle a deja evite une catastrophe
(point 6).

### (6) LES PIEGES RENCONTRES — CHACUN A COUTE UNE MESURE FAUSSE

1. **`DesyncAt = 0` est une SENTINELLE, pas un index.** Sur le chemin delta,
   `DecodeFrameRecords` pose `EntityTrace{DesyncAt: 0}` **sans jamais poser `TypeIndex`** quand
   le test de generation echoue. Lu naivement, cela donne « archetype 0, composant 0 » : j ai
   publie « `ti=0 i0` est le verrou, 13 463 echecs » — **c etait faux**. Discriminant :
   `len(Trace.Comps) == 0`.
2. **Le monde ne doit pas etre remis a neuf par chunk** — les liaisons slot -> archetype des
   chunks precedents sont perdues (37,7 % -> 40,5 % de trames saines une fois corrige).
3. **`bpkCalibre` mesure une FERMETURE, pas une largeur.** Il rend `IDLowBits = 9` a 97,5 % de
   trames exactes ; a cette largeur les records `ti=35` tombent de **11 150 a 1**. Une largeur
   trop petite satisfait la fermeture de facon degeneree (profondeur 1,02 record/paquet). **Un
   oracle de fermeture ne prouve jamais une largeur : il lui faut un oracle de CONTENU.**
4. **`poserBasculeDInstrument` ecrit dans `profilDInstrument`**, le profil du HARNAIS — que les
   marches qui tiennent leur profil du contexte de film n utilisent PAS. Le drapeau se pose sur
   `cfg.Profil.Grammaire`.
5. **Le ratchet de taille (500 lignes) a refuse l instrument DEUX fois.** La reponse est la
   scission par DEPLACEMENT PUR ; `plafondsParFichier` est datee et fermee, **ne pas y ajouter
   de ligne**.
6. **Les heredocs bash echouent sur du contenu a accents et backticks** dans cet environnement :
   passer par un fichier `.py` ecrit a part, ou par l outil d ecriture.

### (7) LES INSTRUMENTS, ET CE QUE CHACUN FAIT

| fichier | role |
|---|---|
| `grammar/mouvement_5_3_2_research_test.go` + `_tableaux_` | la lecture delta par chercheur d ancres, et ses tableaux (5.3.2) |
| `grammar/mouvement_5_3_2b_research_test.go` | porteurs par image, oracle d accroupissement, vitesse en m/s (`DecodeVelocityMagnitude`, loi log/exp 0,03-350) |
| `grammar/mouvement_5_3_2c_evenements_research_test.go` | recensement des types d evenements en tete (`PacketHeadEventType`) + etalonnage du lecteur de masque |
| `grammar/mouvement_5_3_2d_trame_research_test.go` | **LA MARCHE DE REFERENCE** : `DecodeFrameRecords`, carte, drapeau, etalon `i21` |
| `grammar/mouvement_5_3_2d_couverture_research_test.go` | rejets : slots inconnus contre generations avancees |
| `grammar/mouvement_5_3_2d_preambule_research_test.go` | octet de tete et taille, rejetes contre sains |
| `grammar/mouvement_i55_d1_research_test.go` | D1 sur les sept mini-bobines |
| `film/research/mouvement/` + `cmd_mouvement` | la chaine du descripteur sur l exe (cibles, vocabulaire) |

### (8) HORS PERIMETRE — CONSIGNE AU §4, NE PAS TRAITER

- **Les bases par categorie pour les EVENEMENTS** : la table de `FUN_140d10bb0` vaut aussi pour
  les references d evenements (`zoomSlotBase = 512` = la base de la categorie 4, validation
  croisee du § 2decies). Un lot d evenements pourrait PORTER ces bases au lieu de les mesurer.
- **`D5` mode 0 du vehicule** (lot 5.4), **D2** (conventions de `deser_addr` dans
  `ecs_table.tsv`), **D4** (les deux numerotations de types d evenement), **D6** (`i49` : la
  table dit 3 bits, l ecrivain en lit 2 ou 4), **D8** (`scanRecordDirs` ne modelise que six
  composants), **D9** (le domaine mesure des champs d `i54`).
- **Le PORT des etats dans le document** (schema 65, web) : il se decide APRES la mesure finale,
  et pas avant.

### (9) L ENVIRONNEMENT

Worktree `C:/Users/Guillaume/Downloads/Scripts/LevelUp-wt-decfilm-53`, branche
`feat/decfilm-53`. `export GOCACHE=<worktree>/.gocache`, `PATH=/c/msys64/ucrt64/bin:$PATH`,
`CGO_ENABLED=1`, `GOLANGCI_LINT_CACHE=<worktree>/.golangci-cache`.

**AUCUNE BASE DuckDB tant que le backfill tourne** — demander au pilote avant tout
`replay-facts-export`. **UN FILM A LA FOIS.** Films autorises a ce jour : `bfecd02b`
(Snowbound, carte `snowbound`) et `4f77afc1` (Flood Gulch). Jonctions `film_chunks` et
`film_manifests` posees (1 598 entrees), **jamais retirees**, jamais de `git worktree remove`.
Corpus gate complet, `replay-equiv` sur 20 films et re-figeage : **par le pilote, a la fin**.

Gates sans decodage a chaque commit : `gofmt -l ./internal ./cmd` · `go build ./...` ·
`go vet ./...` et `go vet -tags=research ./internal/games/halo_infinite/film/...` ·
`go test -count=1` sur `halo_infinite/...`, `archlint`, `replaybuild`, `replaydoc`,
`replayview`, `contracttest`, `api` · `golangci-lint run ./internal/games/halo_infinite/film/...`.

### (10) LE FORMAT DU COMPTE RENDU

Court, par composant, avec **les tableaux chiffres** et leurs DENOMINATEURS. Dire ce qui est
mesure, ce qui est refute, et ce qui n est pas fait. **Un negatif s ecrit** ; une hypothese
abandonnee se nomme. **Arret sur toute perte.** Et quand une mesure contredit le depot, c est
l instrument qu on suspecte en premier — c est la lecon de `zoom_events.go` (« sept campagnes
ont conclu "aucun evenement de zoom" a cause d un decalage d UN bit ») et celle de ce lot.


---

## 0. LA REPONSE COURTE, TELLE QUE L'ECRIVAIN LA DONNE

| Geste | Le film l'ecrit-il ? | Ou | Forme |
|---|---|---|---|
| **Accroupi** | **OUI** | `ti=35 i29` | booleen + fraction 0..1. **MESURE : uniquement a l'IMAGE-CLE** (99 % des records d'image-cle, 0 % des records delta) — donc un etat tous les ~18 s, pas par image (§ 2ter.1) |
| **Glissade** | **GRAMMAIRE ACQUISE, DONNEE INACCESSIBLE** | `ti=35 i62` | booleen + direction/intensite + 2 fractions. **MESURE : lue 0 fois sur les deux films** — `i62` est DERRIERE le bloquant `i60 simulation-state-component`. Porter `i60` est le pre-requis chiffre (§ 2ter.3) |
| **Sprint** | **OUI, mais sans son nom** | `ti=35 i54 biped-mobility-action-component` — l'ACTION DE MOBILITE, dont le corps est PARTAGE avec l'evenement de fil 43 `initiate_mobility_action` (§ 2.7) | EVENEMENT date : flag1 = « une action est transmise », puis un identifiant et une transformation. **Quelle** action reste a nommer (§ 2.8) ; repli mesurable par la vitesse `i1` |
| **Saut** | **PAS SOUS SON NOM — ni composant, ni evenement de joueur** | negatif mesure (§ 2.7) : seuls `ai_jump` (78) et `AILand` (72), prefixes AI. **Candidats vivants** : le mot de 32 bits optionnel d'`i18 unit-control +0x544` (§ 2.9.1, deja decode et jete), `i55` (§ 2.4), `i63`, et la composante verticale d'`i1` | Theater rejoue l'animation depuis un ETAT REPLIQUE, pas depuis les entrees (§ 2.9.4). Le bit de saut, s'il existe, est dans le mot de 32 bits — **mesurable en 5.3.2, non nommable avant** |

**LE NEGATIF EST MESURE, PAS SUPPOSE** (§ 3) : sur les **64 composants** de l'archetype bipede
(`ti=35`), **aucun** ne porte « sprint » ni « jump » dans son nom ; et sur le pool **complet**
des chaines de l'image, **aucun** nom de composant ne les porte non plus, alors que le moteur
emploie ces deux mots abondamment ailleurs (71 chaines « sprint », 94 « jump »).

---

## 1. LA METHODE, ET CE QUI LA REND PUBLIABLE

Chaine du descripteur du lot 3.7 (`NOTE_3_7_REAPPARITION_2026-09-17.md` § 7), rejouee telle
quelle : nom du composant -> accesseur `LEA reg,[rip+chaine] ; RET` -> slot unique de `.rdata`
= `descripteur + 0x18` -> ecrivain `descripteur + 0x40`. Chaque pas exige l'UNICITE.

**GARDE DE PUBLICATION REUTILISEE, PAS RECOPIEE** : `reapparition.Calibrer` (exporte par ce
lot) rejoue la chaine sur les six temoins du depot ; une seule concordance qui rate et la
passe ne publie rien.

```
== CALIBRATION (6 temoins) ==
  OK  managed-player-back-button-scoreboard-flair-component     ecrivain 0x142ed5af4
  OK  managed-navpoint-sub-type-component                       ecrivain 0x1410e0cac
  OK  managed-navpoint-radial-progress                          ecrivain 0x140fc8d14
  OK  device-position-animation-name-component                  ecrivain 0x1410156e4
  OK  managed-navpoint-manual-timer-initial-duration-component  ecrivain 0x142ed5194
  OK  managed-navpoint-manual-timer-current-duration-component  ecrivain 0x142ed512c
```

**6 / 6.** Les cinq cibles du lot sont ensuite resolues sans un echec, et les cinq adresses
d'ecrivain retombent **a l'octet** sur celles que le decodeur porte deja — la chaine lit bien
cette image-la.

| cible | descripteur | ECRIVAIN | compagnon `+0x28` |
|---|---|---|---|
| `ti=35 i29` unit-crouch | `143d062a8` | **`142ed42a8`** | `142ed9e90` |
| `ti=35 i62` biped-slide | `143d0ca68` | **`142f02978`** | `142f054b8` |
| `ti=35 i54` biped-mobility-action | `143d0c9c0` | **`1408f0264`** | `142f053f8` |
| `ti=35 i55` biped-posture-physics | `143d0cd98` | **`142f0293c`** | `142f05478` |
| `ti=35 i1` object-translational-velocity | `143d0c8d0` | **`14076d45c`** | `14320ca04` |

Ghidra etait joignable pendant ce lot (`127.0.0.1:8089`, `HaloInfinite.exe` analyse,
311 103 fonctions) et a servi **en lecture seule** pour les decompilations et les
desassemblages ci-dessous. Son index de chaines est vide : le balayage de vocabulaire du § 3
est fait par l'instrument Go, sur les octets des sections.

---

## 2. CE QUE LE JEU ECRIT, CHAMP PAR CHAMP

### 2.1 `ti=35 i29 unit-crouch-component` — ACCROUPI : un booleen ET une fraction

`FUN_142ed42a8`, 89 octets, lu en entier :

```
142ed42bc CALL 1406cf008                         R(1)   -> [unite + 0x7e8]   (octet)
142ed42c1 MOVSS XMM3,[143cd8374]                 borne haute
142ed42d1 XORPS XMM2,XMM2                        borne basse = 0.0
142ed42d9 MOV [RSP+0x20],0xa                     largeur = 10 bits
142ed42e7 CALL 1406d84b4                         dequantification
142ed42f3 MOVSS [RDI+0x7ec],XMM0                 -> [unite + 0x7ec]   (float32)
```

`DAT_143cd8374` lu dans l'image : `00 00 80 3f` = **1.0f**.

**VERDICT** : le film ecrit, A CHAQUE IMAGE ou le composant est porte, **un booleen « accroupi »
et une fraction quantifiee sur 10 bits dans [0.0, 1.0]** (1 024 paliers) — la PROGRESSION de
l'accroupissement, pas seulement son etat binaire. C'est un ETAT, pas un evenement : les
intervalles « accroupi de t0 a t1 » se reconstruisent par changement de front du booleen.

Le decodeur lit les deux et les JETTE (`consumeUnitCrouch`, `unit_weaponstate.go`).

### 2.2 `ti=35 i62 biped-slide-component` — GLISSADE : un booleen, une direction, deux fractions

`FUN_142f02978` (mince) -> `FUN_142f26ce8`, lu en entier :

| bits | -> | ce que c'est |
|---|---|---|
| `R(1)` | `[etat + 0x00]` octet | **« en glissade »**. A 0, le composant s'arrete la |
| `R(1)` puis `R(19)` + `R(10)` | `[etat + 0x04]` | direction (cubemap 19 bits) + magnitude (10 bits) : **le vecteur de la glissade** |
| `R(8)` dequant `[0.0, 1.0]` | `[etat + 0x10]` float | fraction |
| `R(8)` dequant `[0.0, 1.0]`, **seulement si `param_4 >= 1`** | `[etat + 0x14]` float | fraction ; **defaut 1.0** quand la porte est fermee (`XMM3` porte encore la borne haute) |
| `R(8)` | `[etat + 0x02]` (stocke en mot) | petit entier 0..255 |

La borne des deux dequantifications est le meme `143cd8374` = **1.0f**.

**VERDICT** : la glissade est un ETAT par image, avec sa direction et son intensite. Le
decodeur porte la grammaire **au bit pres** et jette toutes les valeurs.

**CORRECTION DE L'ETAT DES LIEUX DU BRIEF** : « biped-slide = nom present dans l'exe, alias non
rattache (ligne -1) » est FAUX. La ligne `-1` de `ecs_table.tsv` est l'ALIAS d'orthographe
accepte par le dispatch ; le composant reel est **`ti=35 i62`, statut `porte`**, ligne 751 de
la table.

### 2.3 `ti=35 i54 biped-mobility-action-component` — UNE INITIATION, PAS UN ETAT

`FUN_1408f0264` : `R(1)` flag1 -> `[0x1295]`, `R(1)` flag2 -> `[0x1296]`, puis si flag1 un
handle a largeur variable (`FUN_1408f0ac4`, categorie 0) et le corps `FUN_1408f02c8`.

Corps, champs relus un par un dans la decompilation :

| bits | -> champ | ce que c'est |
|---|---|---|
| `R(1)` ; si 1 `R(10)` sinon `0xFFFFFFFF` | `+0x08` | **identifiant optionnel sur 10 bits** (1 024 valeurs), sentinelle « aucun » |
| `R(1)` ; si 0 : position absolue + avant/haut | — | l'ancrage de l'action |
| `R(96)` brut | `+0x0c` | vecteur 3 flottants |
| position (porte de pleine precision) | `+0x3c` | |
| 3 x (3 x `R(12)`) | `+0x48` | |
| 2 x `R(24)` | `+0x6c`, `+0x78` | |
| 3 x `R(12)` | `+0x84` | |
| 2 x `R(10)` | `+0x90`, `+0x94` | deux flottants |
| `R(1)` | `+0xa1` | |
| `R(7)` | `+0x98` | entier 0..127 |
| `R(2)` | `+0x9c` | **enumere a 4 valeurs** |
| `R(1)` | `+0x9f` | |

Total mesure : **365 a 447 bits** selon les portes internes.

Le vocabulaire de l'image nomme ce composant : `initiate_mobility_action` (`143c97470`) et
`biped-initiate-mobility-action: relevance = %5.3f` (`143e0c500`). **C'est une INITIATION
d'action**, transmise avec la transformation du bipede au moment ou elle commence — donc un
EVENEMENT date, pas un niveau replique. Les candidats a l'identite de l'action sont `+0x08`
(10 bits), `+0x98` (7 bits) et `+0x9c` (2 bits) ; **aucun n'est nomme dans l'image**, leur sens
se tranche par la mesure (§ 4).

Le decodeur lit flag1 et flag2, les publie par `MobilityActionHook` — **sans aucun
consommateur** — et jette le reste.

### 2.4 `ti=35 i55 biped-posture-physics-component` — UN TAG DE 2 BITS QUI OUVRE QUATRE CHARGES

`FUN_142f0293c` resout `bipede + 0x12b4` puis appelle `FUN_142f1f630`, qui lit **`R(2)`** et
passe la valeur a `FUN_141fd997c`. Celle-ci ecrit un octet de discriminant en `+0x2c` de la
cible et appelle un lecteur DIFFERENT par valeur :

| tag | discriminant ecrit | lecteur | consomme des bits ? |
|---|---|---|---|
| 0 | — | `FUN_142f265dc` | **OUI** (`1406cf008`, `1406d310c`, ...) |
| 1 | `1` | `FUN_142f25a3c` | **OUI** (`1406d310c`, `14076dc04`, `14076e494`) |
| 2 | `2` | `FUN_142f263ac` | **OUI** (`14076dc04`, `14076e494`, `1408f0ac4`) |
| 3 | `3` | `FUN_142f264f4` | **OUI** (`14076dc04`, `14076e494`) |

Les charges ecrivent des vecteurs a 3 flottants et des handles a sentinelle `0xffff` /
`0xffffffff` : c'est un **ancrage physique** (« contre quoi le bipede se tient »), pas un
libelle de posture. Le vocabulaire de l'image le confirme du cote animation :
`biped_ground_transient_posture`, `biped_posture_animation`, `character_posture`.

> **DECOUVERTE D1 (§ 6)** : `consumeBipedPosturePhysics` fait `br.Skip(2)` — il lit le tag et
> **ne consomme AUCUNE des quatre charges**, alors que les quatre en consomment chez le jeu.
> **MESUREE le 2026-09-20 sur les sept mini-bobines : § 2.4 bis.**

### 2.4 bis LA MESURE DE D1 — SEPT MINI-BOBINES, AUCUN FILM DU CACHE

Instrument : `grammar/mouvement_i55_d1_research_test.go` (tag `research`, aucun octet de
production). Le tag est relu DIRECTEMENT dans le payload a `CompResult.StartBit`, la position
que la marche de production publie — donc sans crochet d'observation, donc sans toucher
`observateur.go` ni `components_probe.go` (qui feraient bouger `grammar.Rev`).

**CE QUE LES MINI-BOBINES PERMETTENT, ET CE QU'ELLES INTERDISENT.** Leur `PROVENANCE.txt` est
formel : chunk de REGISTRE + douze paquets d'IMAGE-CLE + PIED, et **aucun paquet de
replication**. Le chemin DELTA du bipede n'y est donc pas exercable — mesure a l'appui,
`ScanFilmBipedPositions` refuse les sept (« aucun slot biped (ti=35) dans les keyframes »). La
mesure se fait sur le chemin d'IMAGE-CLE, qui marche les memes composants par la meme boucle et
le meme `case`.

| bobine | records `ti=35` bornes | `i55` franchi | t0 | t1 | t2 | t3 |
|---|---|---|---|---|---|---|
| a521164d | 205 | 77 | 64 | 6 | 2 | 5 |
| 60ae07c4 | 236 | 50 | 40 | 6 | 2 | 2 |
| 11de8353 | 255 | 214 | 170 | 16 | 10 | 18 |
| 111fa685 | 214 | 158 | 119 | 12 | 12 | 15 |
| e5adf7b2 | 237 | 182 | 135 | 24 | 8 | 15 |
| bcb6d393 | 137 | 134 | 97 | 10 | 9 | 18 |
| fb1a1a72 | 80 | 80 | 60 | 5 | 9 | 6 |
| **TOTAL** | **1 364** | **895** | **685** | **79** | **52** | **79** |

**LE TAG EST NON NUL 210 FOIS SUR 895 (23,5 %).** Ce n'est donc pas un negatif : le `Skip(2)`
saute bel et bien, une fois sur quatre, une charge que le jeu lit.

**CONTROLE INTERNE (necessaire).** Si le curseur avait derive AVANT `i55`, les deux bits relus
seraient du bruit — et du bruit rend quatre valeurs equiprobables. L'ecart a l'uniforme vaut
**1 270** pour 895 lectures (un bruit en rendrait environ 3) : les deux bits sont un champ
structure, lu au bon endroit. Corroboration : `i29`, `i54` et `i55` sont declares par
**exactement les memes 895 records** (65,6 %), et `i62` par **aucun** — le masque d'un record
d'image-cle n'est pas du hasard.

**CE QUE CHAQUE TAG COUTE CHEZ LE JEU** (feuilles relevees au desassemblage ; les deux
occurrences d'un meme `ADD [reg+0x2c], n` sont les deux branches d'UNE lecture) :

| tag | lecteur | feuilles consommatrices | cout plancher |
|---|---|---|---|
| 0 | `FUN_142f265dc` | `1406cf008` x3 = 3 x R(1) ; une largeur variable via `1406d310c` | **≥ 3 bits + un champ a largeur variable** |
| 1 | `FUN_142f25a3c` | trois largeurs variables via `1406d310c` ; `14076e494` position ; `14076dc04(0x13)` = R(19) | **≥ 19 bits + position + 3 champs variables** |
| 2 | `FUN_142f263ac` | `14076e494` position ; `14076dc04` ; `1406cf008` = R(1) ; `1408f0ac4` handle ; une R(32) | **≥ 52 bits + position** |
| 3 | `FUN_142f264f4` | `1406cf008` x3 ; `14076e494` position ; `14076dc04(0x13)` = R(19) | **≥ 22 bits + position** |

`14076e494` est la position du port (`consumeE494Position`) : porte de pleine precision puis
`R(96)` brut, ou le corps quantifie. **Aucun des quatre tags ne coute zero bit.**

**« POURQUOI LA MARCHE RESTE-T-ELLE ALIGNEE ? » — SUR CES BOBINES, ELLE NE L'EST PAS, ET ELLE NE
L'A JAMAIS ETE.** L'oracle de fermeture le dit : sur les 1 364 records `ti=35` bornes, **6
ferment** (0,4 %), et **AUCUN de ces 6 n'a franchi `i55`**. La marche du bipede s'arrete plus
loin sur `i60 simulation-state-component`, non porte (c'est exactement ce que le golden
`keyframe_closure.golden` fige depuis le lot 0.A.3) — donc rien, sur le chemin d'image-cle,
n'a jamais verifie le curseur au-dela de `i55`. Il n'y a pas de paradoxe a expliquer ici : il
n'y avait pas d'alignement prouve.

**CE QUI RESTE OUVERT, ET C'EST 5.3.2.** La bit-exactitude du corpus concerne le chemin DELTA,
que les mini-bobines ne portent pas. Trois hypotheses, toutes mesurables sur un film entier :
(a) `i55` n'est pas declare dans les masques delta ; (b) il l'est, et son tag y est toujours 0
avec une charge de tag 0 nulle en pratique ; (c) il l'est avec des tags non nuls, et la marche
delta derive sans qu'aucun oracle actuel ne le voie. **Tant que ce compte n'est pas fait,
aucune conclusion de posture ne doit s'appuyer sur `i55`.**

### 2.5 `ti=35 i1 object-translational-velocity` — LA VITESSE, DEJA DECODEE ET JETEE

`FUN_14076d45c` : `R(1)` ; si pleine precision `R(96)` brut (3 flottants), sinon `R(19)`
direction + `R(10)` magnitude. C'est la voie du SAUT par derivation (composante verticale) et
du SPRINT par seuil de vitesse au sol.

### 2.6 `ti=35 i56 biped-spartan-ability-energy` — L'ENERGIE DE LA CAPACITE D'ARMURE, ET NON LE SPRINT

Le catalogue d'aout (`RECAP_STATS_EXPLOITABLES.md:229`, `HANDOFF_FILM_EXTRACTION_EXTERNAL_DEV.md:595`)
range `i56` sous « crouch / sprint / slide / mobilite ». **Le lecteur du depot dit autre chose, et
il a ete relu au desassemblage** (`ability_energy.go`, deser `FUN_140fc1410` -> `FUN_140fc147c`) :

```
R(3) masque ; puis, POUR CHAQUE BIT ARME, R(7)   -> [bipede + 0x12ea + i]
bit a 0 -> valeur par defaut 0x7F, AUCUN bit lu
cout total : 3 + 7 x popcount(masque) = 3 a 24 bits
```

C'est **la jauge des TROIS EMPLACEMENTS DE CHARGE de la capacite d'armure**, compagnon d'`i48`
`biped-desired-ability-set` (la capacite SELECTIONNEE). Le sprint de Halo Infinite ne consomme
aucune energie — le propulseur, le grappin, le repulseur, le mur, le camo et le surbouclier si.
**`i56` mesure donc l'usage d'EQUIPEMENT, pas le sprint** ; le classer sous « sprint » est une
erreur du catalogue d'aout, que la presente note corrige.

> Le fichier du depot portait deja la lecon : « le DECOMPILE MENT ICI » — Ghidra supprimait le
> bloc froid qui lit les 7 bits, et un portage anterieur en avait conclu « 3 bits, bit-exact ».
> Seul le desassemblage fait foi.

### 2.7 LE CANAL D'EVENEMENTS — UN SEUL CORPS POUR DEUX CANAUX, ET UN CANAL MESURE VIDE

Le film porte, a cote des composants, une liste d'EVENEMENTS. Deux types y touchent au
mouvement (`GRAMMAIRE_EVENTS_FILM_2026-08-30.md`, annexe A, table reconstruite depuis le
registrar `FUN_140e453b4`) :

| type de fil | nom | tampon | lecteur `vtable+0x68` |
|---|---|---|---|
| **43** | `initiate_mobility_action` | 164 octets | `0x142ef8f04` |
| **78** | `ai_jump` | 28 octets | `0x142ef8df0` |

> **CORRECTION DE LECTURE** : les nombres « 164 » et « 28 » sont la **TAILLE DU TAMPON DE
> RECEPTION** (`vtable+0x10`), pas un nombre d'occurrences — l'en-tete de l'annexe A le dit
> mot pour mot. Aucun comptage de corpus ne se lit dans cette table.

**LE FAIT DECISIF : L'EVENEMENT 43 ET LE COMPOSANT `i54` PARTAGENT LE MEME CORPS.**

```c
undefined1 FUN_142ef8f04(..., longlong param_3, undefined8 param_4) {
  uVar1 = FUN_1406cf008(param_4);          // R(1)  -> +0x9d   == flag1
  *(undefined1 *)(param_3 + 0x9d) = uVar1;
  *(undefined1 *)(param_3 + 0x9e) = 0;     //          +0x9e   == flag2, FORCE A 0
  FUN_1408f02c8(param_3, param_4);         // LE MEME CORPS QUE i54
  return 1;
}
```

`FUN_1408f02c8` n'a **que deux appelants** (xrefs Ghidra) : `FUN_1408f0264` (le deser du
composant `i54`) et `FUN_142ef8f04` (le lecteur de l'evenement 43). Meme structure, memes
champs, meme enum — **ce qui se decode une fois sert les deux canaux**.

**MAIS LE CANAL D'EVENEMENTS EST MESURE VIDE POUR CE TYPE.** Le lot R5 (2026-09-03) :
« Treize types suspects ont ZERO tete sur les 325 160 paquets » — dont 42 et 43. Le lot R7,
ecrit exactement pour lever le doute en marchant la LISTE ENTIERE de chaque paquet, conclut
(`RAPPORT_R7_TRAME_COMPLETE_2026-09-03.md`) :

| type | verdict R7 | mesure |
|---|---|---|
| 42 `biped_dodge` | **ABSENT du film** | 0 tete pour 30,3 attendues |
| 43 `initiate_mobility_action` | **ABSENT du film** | 0 tete pour 16,3 attendues |

**CONSEQUENCE POUR LE LOT** : la voie vivante de l'action de mobilite est le **COMPOSANT
`i54`**, pas l'evenement. C'est coherent avec ce que le depot a deja mesure sur `i54` — « le
temoin sans capacite porte quand meme 631 evenements » (`ecs_table.tsv`, ligne 743) : l'action
de mobilite circule, mais dans le flux d'etat.

> Confirmation croisee du champ-a-champ : le lot R7 portait DEJA la grammaire du type 43
> (`r7_charges_lot5_research_test.go:70`) et elle finit par `Skip(1 + 7 + 2 + 1)` — exactement
> les `+0xa1`, `+0x98`, `+0x9c`, `+0x9f` du § 2.3, releves independamment. Deux lectures, une
> grammaire.

**LE SAUT DU JOUEUR N'EST PAS UN EVENEMENT — NEGATIF MESURE SUR LA TABLE ENTIERE.** Sur les 123
types, les seuls au vocabulaire du saut sont **`ai_jump` (78)** et **`AILand` (72)**, tous deux
prefixes `AI`. Le vocabulaire de l'image qui les entoure est celui de la NAVIGATION DES BOTS :
`AI Jump Action: relevance = %5.3f`, `ai_clamber_from_jump`, `ai_clamber_max_jump_height`,
`Bot_EnablePathlessMeleeJump`, `BotTuning_JumpUpExceedsMaxHeightAddedCost`,
`HKAI_TRAVERSAL_TYPE_JUMP` (une categorie de traversee du moteur de navigation Havok). Aucun
type `biped_jump` ni `player_jump` n'existe. **Le saut du joueur reste donc a deriver de `i1`**
(composante verticale), et c'est ce que 5.3.2 mesure.

### 2.8 L'ENUM DE L'ACTION DE MOBILITE — LES TROIS CANDIDATS, ET CE QUI LES DEPARTAGERA

Le corps partage porte trois champs susceptibles de nommer l'action :

| champ | largeur | forme | lecture |
|---|---|---|---|
| `+0x08` | `R(1)` puis `R(10)` | sentinelle `0xFFFFFFFF` quand absent ; plage `FUN_1406d310c(0x400)` = 1 024 | **index de definition** d'action, le plus probable |
| `+0x98` | `R(7)` | entier nu 0..127 | |
| `+0x9c` | `R(2)` | entier nu 0..3, masque `& 0x03` | **candidat n°1 pour l'enum**, voir ci-dessous |

**CE QUI DESIGNE `+0x9c`** : la table d'actions d'entree du moteur (`143d03c40`, § 3) aligne
**exactement quatre** actions de mobilite consecutives — `Sprint` (`143d03c40`), `Thruster`
(`143d03c48`), `Clamber` (`143d03c58`), `Slide` (`143d03c60`) — et `+0x9c` a exactement quatre
valeurs. Ce n'est PAS une preuve : c'est une coincidence de cardinal, et la table d'entree est
une table de LIAISON DE COMMANDES, pas forcement l'enum reseau.

**AUCUN DES TROIS CHAMPS N'EST NOMME DANS L'IMAGE** : le balayage du pool complet des chaines
(§ 3) ne rend aucune etiquette attachee a ces offsets. Les nommer demande donc l'une des deux
voies suivantes, et la premiere est de loin la moins chere :

1. **LA VENTILATION SUR FILM (5.3.2)** — croiser, par joueur et par instant, la valeur de
   `+0x9c` (et de `+0x08`) avec la vitesse au sol d'`i1`, l'etat d'`i62` (glissade) et la
   variation d'altitude. Une classe qui coincide avec « vitesse > marche et pas de glissade »
   est le sprint ; une classe qui coincide avec `i62` actif est la glissade ; une classe qui
   coincide avec une montee franche en z contre un mur est l'escalade ; le reste est le
   propulseur, recoupable par `i56` (§ 2.6) et `i48`.
2. la chasse au CONSOMMATEUR du champ dans l'executable (quel code du jeu branche sur
   `bipede + 0x1294`), plus couteuse et sans garantie.

**LA VENTILATION DE L'ENUM PAR JOUEUR S'AJOUTE DONC AU TABLEAU DE 5.3.2.**

### 2.9 L'OBJECTION DE L'UTILISATEUR — THEATER REJOUE L'ESCALADE, DONC LE FILM PORTE DE QUOI LA DECLENCHER

**L'objection est juste, et elle vaut mieux que ma formulation du § 2.7.** Theater rejoue
l'animation de saut et d'escalade : le joueur s'agrippe au rebord. Quelque chose declenche cela.
L'hypothese proposee : les ENTREES repliquees (les bits d'action de la structure de controle
d'unite, comme dans les Halo precedents). Les quatre candidats ont ete lus CHEZ L'ECRIVAIN.

#### 2.9.1 `ti=35 i18 unit-control-component` — DEUX INDEX BORNES ET UN MOT DE 32 BITS

`FUN_141017084`, relu champ par champ (decompilation + desassemblage du site d'appel de queue) :

| bits | -> champ | forme |
|---|---|---|
| `R(1)` porte | — | a 0 : `+0x548` et `+0x726` recoivent la sentinelle `0xFFFF`, rien d'autre n'est lu |
| `R(5)` | `+0x548` (ushort) | **borne : `> 0x20` fait ECHOUER la lecture** (`return 0`) |
| `R(1)` puis `R(6)` | `+0x726` (ushort) | meme borne `<= 0x20`, meme sentinelle |
| `R(1)` puis `R(32)` | **`+0x544` (dword)** | optionnel ; defaut **0** quand la porte est fermee (`LEA R8,[RBP+0x544]`, `FUN_14080d69c`) |

**CE QUE CELA DIT.** `+0x548` et `+0x726` sont des **INDEX** : plage 0..32, sentinelle `0xFFFF`,
et une valeur hors plage arrete la lecture. Ce ne sont pas des drapeaux d'action. En revanche
**`+0x544` est un mot de 32 bits optionnel, de defaut 0** — c'est **exactement la forme d'un
champ de bits de commande**, et c'est le SEUL champ de cette forme sur tout l'archetype bipede.

**CE QUE CELA NE DIT PAS.** L'ecrivain ne le nomme pas : `FUN_14080d69c` est la feuille
generique « porte + valeur », aucune constante ni enumere ne l'accompagne, et le balayage du
pool des chaines (§ 3) ne rend aucune etiquette attachee a cet offset. Un mot de 32 bits a
defaut 0 est aussi bien un horodatage, un compteur, une graine ou un handle.

> Le glose d'`ecs_table.tsv` pour `i18` — « les ENTREES DE COMMANDE de l'unite (ce que le joueur
> appuie) » — porte deja la mention **« Non mesure »**. Cette note ne la confirme ni ne
> l'infirme : elle la REDUIT a un champ precis, `+0x544`, et la rend mesurable.

**LE DECODEUR LE LIT DEJA ET LE JETTE** (`consumeUnitControl` -> `consumeOpt32`). La ventilation
de ses 32 bits ne coute donc aucun octet de grammaire : c'est une sonde, pas un port.

#### 2.9.2 Les trois autres candidats, ecartes chez l'ecrivain

| composant | ce que l'ecrivain ecrit | verdict |
|---|---|---|
| `i19 unit-actor-control` | des handles resolus en RAM | **DEJA REFUTE par le depot** comme pont vers le joueur (`ecs_table.tsv`) |
| `i25 unit-command-tick` | `R(10)` = le NUMERO d'image d'entree | le film garde le TICK de commande **sans** le champ de boutons qui l'accompagnerait |
| `i49 biped-control-context` | `R(w)` avec **w = 4 si pleine precision, sinon 2** (`DAT_145121140`), puis `R(1)` -> `ctx+0xa33` / `+0xa36` | un CONTEXTE etroit, pas une pression de bouton. **Au passage : le glose « 3 bits (5 valeurs) » d'`ecs_table.tsv` est FAUX** — l'ecrivain lit 2 ou 4 bits selon un reglage de processus (D6) |

**Aucun champ de bits d'action replique n'a ete trouve sur le bipede**, hors le mot de 32 bits
d'`i18`.

#### 2.9.3 `i55` : les quatre tags sont-ils la machine d'etat de posture ?

L'hypothese de l'utilisateur (au sol / aerien / accroupi / escalade) a ete confrontee a l'image.

**L'ENUMERE DE POSTURE DU MOTEUR EXISTE, ET IL NE TIENT PAS SUR DEUX BITS.** Le cluster
`0x1437d6870` porte, dans l'ordre, l'enumere d'etat de personnage de Havok :

```
HK_CHARACTER_ON_GROUND · HK_CHARACTER_JUMPING · HK_CHARACTER_IN_AIR
HK_CHARACTER_CLIMBING  · HK_CHARACTER_FLYING  · HK_CHARACTER_USER_STATE_0..2
```

**CINQ etats au moins, donc trois bits au minimum.** Le tag d'`i55` en a deux : il ne PEUT pas
porter cet enumere. C'est un negatif de cardinal, pas une opinion.

**CE QUE LES QUATRE CHARGES DISENT VRAIMENT.** Elles ecrivent des vecteurs a trois flottants et
des handles a sentinelle (`0xffff` / `0xffffffff`), et le tag `2` lit en plus un HANDLE
(`FUN_1408f0ac4`) — c'est-a-dire une REFERENCE D'ENTITE. La forme est celle d'un **ANCRAGE** :
« contre quoi, et ou, le bipede se tient » — rien (t0), un ancrage simple (t1), **un ancrage a
une ENTITE** (t2), un ancrage a une POSITION du monde (t3). L'intuition « ca sent l'accrochage »
vise juste sur cette moitie-la : s'agripper a un rebord EST un ancrage. Mais l'etat aerien, lui,
n'y tient pas : il n'a pas de vecteur d'ancrage. **Nommer les quatre tags reste une mesure de
5.3.2** (§ 2.4 bis : le tag est non nul 23,5 % du temps).

#### 2.9.4 CE QUE LE GRAPHE D'ANIMATION DU JEU DIT DU DECLENCHEMENT

Le pool des chaines porte les noeuds du graphe d'animation, et ils tranchent la question du
MECANISME :

```
EntryNode_objects_animation_graphs_transition_conditions_is_sprinting_tlg
EntryNode_objects_animation_graphs_transition_conditions_is_not_sprinting_tlg
EntryNode_objects_animation_graphs_transition_conditions_is_airborne_tlg
EntryNode_objects_animation_graphs_transition_conditions_is_leap_airborne_tlg
EntryNode_objects_animation_graphs_character_sprint_ang
EntryNode_objects_animation_graphs_character_airborne_default_ang
```

Les transitions du graphe sont gardees par des **CONDITIONS D'ETAT** — `is_sprinting`,
`is_airborne` — et non par des pressions de bouton. Les accesseurs de script vont dans le meme
sens : `SpartanAbilityIsSprinting`, `SpartanAbilityGetSprintFraction`,
`SpartanAbilityIsClambering`, `IsAirborne`, `Unit_IsAirborne`.

**CONCLUSION DU § 2.9, ET CORRECTION DU § 2.7.** Theater ne rejoue pas des ENTREES : il rejoue
un ETAT REPLIQUE, et cet etat est precisement ce que le lot 5.3 a trouve — l'accroupissement et
sa progression (`i29`), la glissade et son vecteur (`i62`), l'ACTION DE MOBILITE avec sa
transformation d'ancrage (`i54`, 365 a 447 bits : c'est la pose de l'accrochage), la posture
physique et son ancrage (`i55`), l'action en cours (`i63`). **L'utilisateur a raison sur le
fond — le film porte de quoi rejouer l'escalade — et le mecanisme est l'etat, pas l'entree.**

La formulation a corriger est celle du saut : **« le saut n'a pas d'evenement » reste vrai et
mesure (§ 2.7), mais il ne faut pas en conclure que le film n'en sait rien.** Il reste UN
candidat de forme « entree » — le mot de 32 bits d'`i18 +0x544` — et trois candidats d'etat
(`i55`, `i63`, la composante verticale d'`i1`). Tous sont mesurables a la voie libre, aucun
n'est nommable avant.

## 2 ter. LA PREUVE SUR FILM (5.3.2) — DEUX FILMS, UN A LA FOIS, AUCUNE BASE OUVERTE

> Voie libre du pilote le 2026-09-21. `bfecd02b` (Snowbound, Team Slayer) puis `4f77afc1`
> (Flood Gulch). Instrument : `grammar/mouvement_5_3_2*_research_test.go`, tag `research`.
> Aucune base DuckDB, aucun artefact, aucun corpus.

### 2ter.1 LE FAIT STRUCTURANT : DEUX CANAUX, DEUX CADENCES

> **CONCLUSION RETIREE LE 2026-09-21 — VOIR LE § 2 QUATER.** Les chiffres de ce paragraphe
> restent (ils ont ete mesures), mais leur LECTURE — « les etats ne voyagent qu'a
> l'image-cle » — est FAUSSE : le depot documente en trois endroits un etat bipede PAR TICK
> dans les deltas de type 0, accroupissement compris. Le « 0,0 % » ci-dessous est un defaut
> d'instrument, et le § 2quater.5 nomme le suspect.

La marche de production ne lit pas les composants de mouvement sur le chemin delta — et ce
n'est pas un choix, c'est une limite : `scanRecordDirs` ne modelise que `i1`, `i2`, `i3`, `i4`,
`i5` et `i21`, et s'arrete au premier composant hors de cette liste (**D8**). L'instrument
rejoue donc la VRAIE boucle de composants (`traverseComponentLoopFrom`) sur les records delta,
et la meme sur les records d'image-cle.

| | `bfecd02b` delta | `bfecd02b` image-cle | `4f77afc1` delta | `4f77afc1` image-cle |
|---|---|---|---|---|
| records `ti=35` | **162 444** (90 slots) | **207** | **378 661** (253 slots) | **995** |
| `i1` vitesse | 90,1 % | 24,2 % | 86,3 % | 26,4 % |
| `i18` unit-control | **0,0 %** | **99,0 %** | **0,0 %** | **98,8 %** |
| `i29` unit-crouch | **0,0 %** | **99,0 %** | **0,0 %** | **98,8 %** |
| `i54` mobility-action | **0,3 %** (464) | 99,0 % | **0,5 %** (1 867) | 98,8 % |
| `i55` posture-physics | **0,0 %** | **99,0 %** | **0,0 %** | **98,8 %** |
| `i62` biped-slide | 0,0 % (2 au masque) | 0,0 % | 0,0 % (2 au masque) | 0,0 % |

**CE QUE CELA CHANGE POUR LE PRODUIT.** L'accroupissement, la posture et le mot de controle
sont **echantillonnes a l'image-cle**, soit une fois toutes les ~18 s par bipede — pas par
image. Des « intervalles d'etat par joueur » a la cadence de l'image sont donc **impossibles**
pour ces champs : ce que le film permet, c'est un ETAT AU MOMENT DE L'IMAGE-CLE. Seule
l'action de mobilite est datee finement, parce qu'elle voyage en delta.

### 2ter.2 D1 TRANCHEE SUR FILM — ET SANS CONSEQUENCE MESURABLE

| | chemin delta | image-cle | tags | NON NULS |
|---|---|---|---|---|
| `bfecd02b` | `i55` sur **0** des 162 444 records | 205/207 | 0:138 1:15 2:23 3:29 | **67 (32,7 %)** |
| `4f77afc1` | `i55` sur **0** des 378 661 records | 983/995 | 0:730 1:70 2:89 3:94 | **253 (25,7 %)** |

Les deux films confirment la mesure des mini-bobines (23,5 %) : **le `Skip(2)` saute bien, une
fois sur quatre, une charge que le jeu lit** — et il ne le fait QUE sur le chemin d'image-cle,
jamais en delta.

**ET POURTANT RIEN NE CASSE, POUR UNE RAISON MESUREE** : la marche d'image-cle du bipede
s'arrete de toute facon avant d'avoir quoi que ce soit a verifier. Les bloquants, comptes :

```
bfecd02b : i60 simulation-state-component 171 · i57 biped-spartan-ability 18 · i59 …-non-predicted-state 17   (marche complete 1/207)
4f77afc1 : i60 simulation-state-component 859 · i59 …-non-predicted-state 68 · i57 biped-spartan-ability 58   (marche complete 10/995)
```

Aucun oracle de fermeture ne s'exerce au-dela d'`i55`. **D1 est donc reelle, documentee, et
inoffensive tant que la marche s'arrete a `i60`** — elle deviendra bloquante le jour ou `i60`
sera porte. C'est une dette datee, pas un incident.

### 2ter.3 LA GLISSADE EST INACCESSIBLE — ET C'EST UN PRE-REQUIS CHIFFRE, PAS UNE IMPASSE

`i62 biped-slide` est lu **0 fois** sur les deux films. La raison est mesuree, et ce n'est PAS
que le film n'en parle pas : **`i62` est DERRIERE le bloquant**. `i60` arrete la marche, et
`i62` vient apres. Sur le chemin delta, `i62` n'apparait au masque que 2 fois par film.

**Pour publier la glissade, il faut d'abord porter `i60 simulation-state-component`** (et,
accessoirement, `i57` et `i59`). Le cout est chiffre : 171 + 859 records bloques sur les deux
films. Tant que ce n'est pas fait, la glissade n'est pas publiable — et dire « le film ne la
porte pas » serait faux.

### 2ter.4 LE MOT DE 32 BITS N'EST PAS UN CHAMP DE BOUTONS — D7 EST REFUTEE

`i18 +0x544` ne voyage qu'a l'image-cle : **35 records sur 205** (`bfecd02b`) et **180 sur
983** (`4f77afc1`). Ventilation bit a bit, chaque bit contre la composante verticale de la
vitesse au record suivant du meme slot :

| film | bits allumes | frequence par bit | montee qui suit | plancher |
|---|---|---|---|---|
| `bfecd02b` | **32 / 32** | 2,9 % a 14,3 % | 25 % a 100 % (n <= 5) | 5,8 % |
| `4f77afc1` | **32 / 32** | 7,2 % a 20,6 % | 21 % a 62 % | 8,9 % |

**AUCUN BIT MORT, AUCUN BIT DOMINANT.** Un champ de bits de commande aurait la signature
inverse : la plupart des bits jamais allumes (le jeu n'a pas 32 actions), et un ou deux bits
tres frequents (avancer, tirer). Ici les 32 bits sont allumes dans la meme fourchette etroite
et aucun ne predit la montee mieux que ses voisins. **C'est la signature d'un mot OPAQUE a
forte entropie** — jeton, horodatage, hachage — pas d'un champ de boutons.

**D7 est donc REFUTEE, et avec elle la derniere piste « entree repliquee » du saut.** Il ne
reste aucun candidat de forme « bouton » sur l'archetype bipede. Le § 2.9.4 tenait : Theater
rejoue un ETAT, pas des entrees.

> Reserve ecrite : la cadence d'image-cle (~18 s) rend de toute facon impossible d'apparier un
> bit a un saut, qui dure environ une seconde. Meme si un bit de saut existait dans ce mot, ce
> canal ne permettrait pas de le dater. Le negatif ci-dessus ne repose pas sur cette reserve —
> il repose sur l'absence de bit mort — mais elle le double.

### 2ter.5 L'ACTION DE MOBILITE, MESUREE — ET L'ORACLE DE VITESSE QUI REFUTE LE SPRINT

| | `bfecd02b` | `4f77afc1` |
|---|---|---|
| initiations (`flag1`) | **453** sur 9 slots | **1 792** sur 63 slots |
| `flag2` | 0 | 3 |
| identifiant 10 bits present | **0** | **0** |
| `+0x9c` R(2) | 0:342 · **2:111** | 0:1 623 · **2:169** |
| `+0x98` R(7) | 0:342 · 1:74 · 3:37 | 0:1 535 · 1:73 · 3:56 · 5:53 · 8:21 · 10:18 · 13:19 · 15:17 |

**TROIS RESULTATS NETS.**

1. **L'identifiant de 10 bits n'est JAMAIS transmis** — 0 fois sur 2 245 initiations. Le champ
   `+0x08` reste a sa sentinelle : **il ne porte pas l'identite de l'action**, contrairement a
   ce que le § 2.8 donnait pour le candidat le plus probable. Rayé.
2. **`+0x9c` ne prend que DEUX valeurs, 0 et 2** — jamais 1 ni 3. Ce n'est donc pas un enumere
   a quatre actions ; c'est un drapeau a deux etats loge dans deux bits. L'hypothese
   « Sprint / Thruster / Clamber / Slide sur 2 bits » (§ 2.8) est **refutee par les valeurs**.
3. **`+0x98` porte 3 valeurs sur un film et 8 sur l'autre**, avec 0 tres dominant (75 % et
   86 %). C'est le seul champ qui se comporte comme un discriminant d'action — et son domaine
   depend du film, donc du contenu de la partie.

**L'ORACLE DE VITESSE REFUTE LE SPRINT.** La magnitude quantifiee (monotone en vitesse) au
moment de l'initiation, contre les records ordinaires :

| classe | `bfecd02b` p10 / median / p90 | `4f77afc1` p10 / median / p90 |
|---|---|---|
| action de mobilite | 55 / **125** / 164 (n = 449) | 94 / **130** / 189 (n = 1 725) |
| debout | 136 / **211** / 245 (n = 145 875) | 129 / **217** / 250 (n = 324 920) |

**L'action de mobilite se produit PLUS LENTEMENT que la marche ordinaire, sur les deux films.**
Un sprint irait plus vite. Ces initiations ne sont donc pas des sprints : le profil est celui
d'un geste ou l'on RALENTIT — s'agripper a un rebord (escalade), ou amorcer une poussee.
**C'est la premiere mesure du lot qui dit ce que `i54` n'est PAS, et elle est franche.**

### 2ter.6 LE SPRINT — NON TRANCHE, ET DIT COMME TEL

La distribution de la magnitude debout est **resserree** (p10 136, mediane 211, p90 245 sur
`bfecd02b`) : **aucune seconde bosse** ne s'y detache. Le sprint ne se lit donc pas comme une
classe separee de cette seule distribution. Deux raisons possibles, non departagees ici : la
quantification (log/exp) ecrase le haut de la plage, ou la vitesse de sprint n'est pas assez
distante de la marche pour se voir sans dequantification. **Conclusion : non tranche.** Ce qui
le trancherait : dequantifier la magnitude en m/s et comparer aux vitesses connues du jeu.

### 2ter.7 LES CINQ INSTANTS PAR ETAT, POUR L'OEIL DE L'UTILISATEUR

Sur `bfecd02b` (Snowbound, Team Slayer), en **TEMPS DE BARRE THEATER** = temps film depuis le
debut, `mm:ss`. Le `slot` est l'entite, c'est-a-dire UNE VIE — l'attribution vie -> joueur est
le travail de l'index de `replaybuild` et n'est PAS faite ici (le document n'a pas ete cuit :
aucune base ouverte). Les instants sont espaces d'au moins dix secondes pour ne pas donner cinq
fois le meme geste.

| etat | 5 instants (temps de barre Theater) |
|---|---|
| **ACCROUPI** | `02:40` (slot 518) · `03:00` (542) · `03:40` (547) · `04:20` (552) · `04:40` (559) |
| **GLISSADE** | **AUCUN** — inaccessible, voir § 2ter.3 |
| **ACTION DE MOBILITE** | `00:39` (516) · `01:00` (513) · `01:20` (518) · `01:37` (523) · `01:48` (515) |
| **MONTEE (candidat saut, `dirZ > 0,60`)** | `00:19` (512) · `00:36` (518) · `00:46` (517) · `00:57` (515) · `01:08` (521) |

> Les instants « accroupi » viennent des images-cles (cadence ~18 s) ; les « action de
> mobilite » et les « montee » viennent du chemin delta, donc de l'image exacte.

### 2ter.8 CE QUE L'INSTRUMENT A APPRIS SUR LUI-MEME

Deux defauts de harnais ont ete trouves par des ECARTS DE COMPTE, et ils sont consignes parce
que le prochain instrument les referait :

1. **`traverseComponentLoopFrom` ne pose pas `EndBit`** — c'est `TraverseEntity` qui le fait
   apres elle. Sans cette ligne, la fin du DERNIER composant d'un record vaut 0 et ce composant
   n'est jamais lu : `i54`, plus haut index de 459 records, etait vu **5 fois au lieu de 487**.
2. **Le registre d'un film peut nommer un composant SANS le suffixe `-component`** (le dispatch
   accepte les deux orthographes). Comparer au seul nom long fait manquer des composants.

Dans les deux cas, c'est la mesure separee du MASQUE — le denominateur — qui a revele l'ecart.
Un instrument qui ne publie que ses trouvailles ne peut pas se corriger lui-meme.

## 2 quater. CORRECTION DE CAP (2026-09-21) — LA CONCLUSION « IMAGE-CLE SEULEMENT » EST RETIREE

> L'utilisateur, qui fait autorite sur le film, a corrige : ces etats sont connus A L'INSTANT
> PRECIS, « on sait a la milliseconde ce que le joueur fait, on sait meme quand il zoome ». Il a
> raison, et le depot le documentait deja en trois endroits. **La conclusion du § 2ter.1 est
> RETIREE.** Ce paragraphe dit ce qui la remplace, et ce qui reste a faire.

### 2quater.1 CE QUE LE DEPOT DISAIT DEJA, ET QUE LE § 2 TER A CONTREDIT

| source | ce qu'elle dit |
|---|---|
| `RECAP_STATS_EXPLOITABLES.md` § TIER 3 | « ETAT BIPED PAR FRAME (sante, bouclier, velocite, **CROUCH**, position, aim, munitions) — source : **deltas type-0 (~60 fps)** + snapshots keyframe type-2 (~18-20 s) » |
| `HANDOFF_FILM_EXTRACTION_EXTERNAL_DEV.md` l. 29 | « **per-tick** biped state (health, shield, velocity, **crouch**, position, aim, ammo) » |
| `RECETTE_DECODAGE_FILM_CHUNKS.md` | la table des deserialiseurs : `i18 unit-control FUN_141017084`, `i29 unit-crouch FUN_142ed42a8`, `i21 aiming FUN_14076df7c` |

**L'accroupissement est par tick dans les deltas de type 0.** Le « `i29` a 0,0 % des records
delta » du § 2ter.1 est donc un **DEFAUT D'INSTRUMENT**, pas un fait du film, et il est retire.

### 2quater.2 LE MODELE D'UN ETAT PAR INSTANT QUI EXISTE DEJA : LE ZOOM

`zoom_events.go` lit l'etat de lunette **a l'instant**, et il le fait dans la LISTE
D'EVENEMENTS en tete des paquets delta — pas dans la trame de composants :

```
[1 bit config] [ ( 1 [R(7) type] [3 references gardees] [charge] )* 0 ] [trame de records]
```

`unit_zoom` (type 21) : ~400 000 occurrences sur 1 367 films ; charge `R(2)` = le palier de
lunette + 1 ; pont vers le joueur par la premiere reference (domaine 4) **+ 512 = le slot du
bipede** (63 index sur 64 tombent sur un slot reel, contre 0 sur 64 pour toute autre base) ;
valide contre Theater (6 entrees en lunette sur 6, a moins de 1,2 s).

**ET SA LECON, QUI EST EXACTEMENT LA MIENNE** : « Sept campagnes de mesure ont conclu "aucun
evenement de zoom dans la bobine" parce qu'elles lisaient le type a `payload[0] & 0x7F` — elles
ignoraient le bit de configuration et decalaient donc TOUT d'un bit. » **Quand une mesure rend
zero, le suspect numero un est l'instrument.**

### 2quater.3 CE QUE LE RECENSEMENT DES EVENEMENTS DONNE SUR `bfecd02b` — MESURE ETALONNEE

Par `PacketHeadEventType`, la porte du depot (jamais une arithmetique refaite a la main) :
**31 232 paquets delta, 5 274 portent un evenement en tete (16,9 %), 20 types distincts.**

| type | nom | en tete | part |
|---|---|---|---|
| 36 | `action_weapon_fire` | 2 611 | 49,5 % |
| 82 | `PlayerGameEventSmall` | 578 | 11,0 % |
| 15 | `Script` | 378 | 7,2 % |
| **21** | **`unit_zoom`** | **320** | **6,1 %** |
| 0 | `damage_aftermath` | 277 | 5,3 % |
| 38 | `weapon_reload` | 229 | 4,3 % |
| 9 | `biped_pickup` | 142 | 2,7 % |
| 39 | `biped_throw_initiate` | 76 | 1,4 % |

**Le temoin passe** : `unit_zoom` est bien la, 320 fois, sur ce film. Le canal des instants est
vivant et lisible. Les types 42, 43, 72 et 78 sont a **0 en tete** — et c'est un PLANCHER, pas
un negatif : ce scanner ne lit que le PREMIER evenement de chaque liste.

**`PlayerGameEventSmall` (type 82, 578 occurrences) est le candidat a instruire** : un
evenement de joueur generique, frequent, dont la charge n'est pas portee.

### 2quater.4 L'ETALONNAGE N'A PAS PU SE FAIRE PAR LA PORTE DE PRODUCTION — MESURE A CONSIGNER

Le balayage bipede de production rend **ZERO record** sur `bfecd02b`, `DropSaturated` a vrai
comme a faux, profil MPP du build installe ou non, alors que la marche directe en lit 162 444
avec la meme bande de slots et le meme decoupage d'`i0`. `ScanBipedPositions` a donc une
condition d'entree que ce film ne remplit pas — vraisemblablement les bornes de carte, que
`QuantaOnly` dispense de FOURNIR mais dont le filtre de saturation depend encore.

Deux controles ont malgre tout ete faits :

1. **Le filtre `RequireTag1` n'est PAS la cause.** Sans lui : 162 487 records au lieu de
   162 444, et `i29` passe de 0 a **1**. Le crible d'ancre n'ecarte donc pas la population
   cherchee.
2. **Le gradient du masque est coherent** — `i0` 100 %, `i25` 100 %, `i1` 90,3 %, `i21` 62,4 %,
   `i5` 32,2 % — et le dispatch de production est alle AU BOUT de 162 482 records sur 162 487.

### 2quater.5 CE QUI RESTE, ET C'EST PRECIS

**LE SUSPECT NOMME** : `walkDeltaBipedPayload` est un **CHERCHEUR D'ANCRES** — il balaie le
payload bit a bit et retient ce qui RESSEMBLE a un en-tete de bipede. Ce n'est pas le decodeur
de trame. Le depot en a un vrai : **`DecodeFrameRecords` (`frame_records.go`)**, qui consomme
le preambule du paquet puis lit les records DANS L'ORDRE. `event_list.go` precise sa limite :
il saute la liste d'evenements, donc il fonctionne sur les paquets a liste vide (octet de tete
`0x80..0xBF`) et rate ceux qui portent un evenement — soit, sur `bfecd02b`, **83,1 % des
paquets** (25 958 sur 31 232).

**LA PROCHAINE MESURE, ET ELLE N'EST PAS FAITE ICI** : re-ventiler `ti=35` avec
`DecodeFrameRecords` sur ces 83,1 % de paquets, etalonner sur `i21`/`i0`/`i1`, puis rendre pour
`i29` la cadence reelle et les intervalles d'accroupissement par slot. **Aucune conclusion de
cadence ne doit etre tiree avant cet etalonnage** — celle du § 2ter.1 ne l'a pas ete, et elle
etait fausse.

**A LIRE AVANT**, sur demande de l'utilisateur : `RE_EXE_GHIDRA_FINDINGS.md`,
`PLAN_FILM_ECS_DECODER.md`, et le `thought_log` des 2026-06-03/04/05 (archive Q2).

## 2 quinquies. LE VRAI DECODEUR DE TRAME, ET L ETALON QUI TRANCHE (2026-09-21)

### 2quin.1 LA MESURE PAR `DecodeFrameRecords`

Troisieme instrument (`mouvement_5_3_2d`), sur le decodeur de trame du depot : preambule de
paquet, puis les records **dans l ordre**, contre un `World` amorce par les images-cles du
chunk. Domaine : les paquets a liste d evenements VIDE (`pay[0]&0x40 == 0`), soit **25 958 des
31 232 paquets delta** de `bfecd02b`.

| | valeur |
|---|---|
| paquets cadres | 25 958 |
| trames decodees **sans erreur** | **9 786 (37,7 %)** |
| records de trame | 24 940, dont **10 060 de `ti=35`** |
| etalon | `i0` 62,5 % · `i1` 55,5 % · **`i21` 64,5 %** · **`i25` 95,2 %** |
| `i18` / `i29` / `i54` / `i55` / `i62` | **4 · 1 · 8 · 3 · 1** (soit 0,0 % a 0,1 %) |
| intervalles d accroupi | **0** sur 37 slots |

**DEUX DECODEURS INDEPENDANTS CONVERGENT.** Le chercheur d ancres (162 444 records) et le
decodeur de trame (10 060 records de `ti=35`) donnent la meme reponse : dans la trame de
composants des paquets delta, l accroupissement n est pas la.

**ET LA MESURE PORTE SA PROPRE RESERVE** : 37,7 % de trames decodees sans erreur, ce n est pas
un cadrage sain. Prise seule, elle ne suffirait pas.

### 2quin.2 L ETALON QUI TRANCHE — ET IL N EST NI L UN NI L AUTRE DE MES INSTRUMENTS

Le depot porte une VERITE TERRAIN, obtenue par capture live (Cheat Engine) et consignee dans
`components_position_i0.go` :

> « sur les **15 529 records du masque `{i0,i1,i21,i25}`** dont l oracle de POSITION Rosette
> donne la longueur vraie (**113 bits**), la seule largeur de `i0` pour laquelle les desers
> PORTES de `i1` et de `i21` consomment exactement leurs largeurs vraies (**31 et 25 bits**)
> est **47 bits** : `i1` tombe juste sur 100,0 % des records et `i21` sur 100,0 %. »

Et `unit_weaponstate.go` : « l oracle de position Rosette donne **unit-command-tick = 10 bits
constants** ».

**LE RECORD BIPEDE DELTA DOMINANT EST DONC `{i0, i1, i21, i25}` — 47 + 31 + 25 + 10 = 113 bits.
Position, velocite, visee, tick de commande. L ACCROUPISSEMENT N Y EST PAS.**

Ce n est pas mon instrument qui le dit : c est une capture live du jeu, sur 15 529 records, et
mes deux lecteurs independants retrouvent exactement ce masque (`i0` 100 %, `i25` 100 %, `i1`
90,3 %, `i21` 62,4 % au chercheur d ancres).

### 2quin.3 CE QUE CELA VEUT DIRE, ET CE QUE CELA NE VEUT PAS DIRE

> **AMENDE LE 2026-09-21 PAR LE § 2 SEXIES.** Le record delta DOMINANT porte bien quatre
> composants (l oracle Rosette reste vrai), mais le record RARE en porte plus — et c est lui
> qui porte l accroupissement. La conclusion « l accroupissement n est pas dans la trame » est
> RETIREE : il y est, derriere les largeurs fausses de `ti=0 i0` et des trois `partiel` du
> bipede.

**CE QUE CELA VEUT DIRE** : la phrase « per-tick biped state (health, shield, velocity,
**crouch**, position, aim, ammo) » de `RECAP_STATS_EXPLOITABLES` et du handoff decrit le
VOCABULAIRE de l archetype bipede — ce que le film PEUT porter — et non ce que le record delta
porte a chaque tick. Le record delta dominant porte quatre composants, pas sept.

**CE QUE CELA NE VEUT PAS DIRE** : que le film ignore l accroupissement a l instant.
L utilisateur a raison sur le fond — Theater le rejoue — et le depot montre DEJA ou un etat par
instant se loge quand il n est pas dans la trame : **le canal d evenements**. `unit_zoom` en est
la preuve vivante (~400 000 occurrences, pont vers le slot par domaine 4 + 512, valide contre
Theater 6/6). L accroupissement par instant est donc a chercher **la**, pas dans les masques.

### 2quin.4 `PlayerGameEventSmall` — LE CANDIDAT, ET CE QUE SON ECRIVAIN DIT

578 occurrences en tete sur `bfecd02b` (11,0 % des paquets a evenement), deuxieme type le plus
frequent apres le tir. Son lecteur (`vtable+0x68` = `FUN_14080add8`) :

```c
FUN_14080b30c(param_3);              // initialisation, 0 bit
*(undefined4 *)(param_3 + 0xa0) = 0; // remise a zero d un champ
FUN_14080ae70(param_3, param_4);     // R(32) -> param_3[0]
FUN_14080ae28(param_4);              // une seconde lecture
```

La charge commence donc par **un mot de 32 bits**, suivi d une seconde lecture non encore
relevee. **Aucun sous-type n est nomme a ce stade** : le nommer demande de relever
`FUN_14080ae28` et de ventiler le mot sur le film. C est la prochaine mesure, et elle n est pas
faite ici.

### 2quin.5 LES CINQ INSTANTS, ET POURQUOI ILS NE SONT PAS RE-CALCULES

Les instants d accroupi du § 2ter.7 viennent des images-cles, et ils restent valides pour ce
qu ils sont : les seuls instants d accroupissement que le depot sait dater aujourd hui. Le
decodeur de trame n en produit **aucun** (0 intervalle sur 37 slots), et il serait malhonnete de
publier une liste vide comme un progres. Des instants d accroupi A LA CADENCE DU JEU viendront
du canal d evenements, quand il sera lu.

## 2 sexies. L HYPOTHESE DU PILOTE EST CONFIRMEE : L ACCROUPI PAR INSTANT EST DANS LA TRAME, DERRIERE UNE LARGEUR FAUSSE

> Mesure du 2026-09-21 sur `bfecd02b`, toujours par `DecodeFrameRecords`. Le pilote a pose la
> bonne question : **un delta ne porte `i29` QUE quand l accroupi CHANGE**, donc ces records
> sont rares — et si ce sont AUSSI ceux qui echouent, l accroupi est bien la.

### 2sex.1 LA POPULATION EN ECHEC, ET CE QU ELLE PORTE

**16 172 records desynchronises**, dont seulement **44 de `ti=35`** (0,3 %). La comparaison qui
decide n est pas un compte mais un RAPPORT — la part des masques portant chaque composant parmi
les records EN ECHEC, contre cette meme part parmi les records SAINS :

| composant | part des ECHECS `ti=35` | part des SAINS | facteur |
|---|---|---|---|
| `i18 unit-control` | **18,18 %** (8) | 0,05 % (5) | **x 364** |
| `i29 unit-crouch` | **9,09 %** (4) | 0,04 % (4) | **x 227** |
| `i54 biped-mobility-action` | **22,73 %** (10) | 0,10 % (10) | **x 227** |
| `i55 biped-posture-physics` | **13,64 %** (6) | 0,07 % (7) | **x 195** |
| `i62 biped-slide` | **18,18 %** (8) | 0,03 % (3) | **x 606** |

**LES CINQ COMPOSANTS DE MOUVEMENT SONT SUR-REPRESENTES D UN FACTEUR 200 A 600 DANS LES
ECHECS.** Le « `i29` a 0,0 % » des mesures precedentes etait donc une mesure de la population
SAINE — et la population saine est, par construction, la population PAUVRE : les records
`{i0,i1,i21,i25}` de 113 bits que l oracle Rosette decrit. Les records RICHES, ceux qui portent
un changement d etat, sont precisement ceux qui cassent.

**L hypothese du pilote est confirmee : l accroupissement par instant EST dans la trame.**

### 2sex.2 LE COMPOSANT FAUTIF, NOMME, PAR ARCHETYPE

> **CE TABLEAU EST FAUX ET REMPLACE PAR LE § 2 SEPTIES (2026-09-21).** Les 13 463 « `ti=0 i0` »
> etaient des REJETS DE GENERATION, pas des composants fautifs : sur le chemin delta,
> `DesyncAt = 0` est une sentinelle de rejet et `TypeIndex` n y est jamais pose. La
> sur-representation du § 2sex.1, elle, tient et se renforce.

| archetype | echecs | composant fautif dominant |
|---|---|---|
| **`ti=0`** (game-engine) | **13 686** (84,6 %) | **`i0` : 13 463** — le verrou principal |
| `ti=2` | 424 | `i15` : 328 |
| `ti=5` | 422 | `i22` : 367 |
| `ti=10` | 315 | — |
| **`ti=35`** (bipede) | **44** | **`i59` : 18 · `i60` : 16 · `i57` : 10** |

Cote bipede, les trois fautifs sont exactement les trois composants que `ecs_table.tsv` declare
**`partiel`** : `i57 biped-spartan-ability`, `i59 biped-spartan-ability-non-predicted-state`,
`i60 simulation-state`. Un record qui porte l accroupissement porte aussi ces composants-la, et
la marche casse sur eux — pas sur `i29`.

**LA CHAINE CAUSALE, ECRITE** : `ti=0 i0` casse dans 13 463 paquets → le reste du paquet n est
jamais atteint → les records bipedes RICHES, qui viennent apres dans la trame, sont perdus →
`i29` parait absent. Ce n est pas le film qui se tait, c est la marche qui n arrive pas jusqu a
lui.

### 2sex.3 CE QUE CELA FAIT DU LOT

**L accroupissement par instant n est PAS un item de recherche : c est un ITEM DE GRAMMAIRE**,
et sa liste de travaux est nommee, dans l ordre du rendement :

1. **`ti=0 i0`** — 13 463 echecs a lui seul, 83 % de tous les echecs du film. Tant qu il n est
   pas bit-exact, aucune trame riche ne se lit.
2. **`ti=35 i57`, `i59`, `i60`** — les trois `partiel` du bipede, 44 echecs sur ce film, mais ce
   sont eux qui gardent l acces a `i29`, `i18`, `i54`, `i55` et `i62`.
3. `ti=2 i15` (328) et `ti=5 i22` (367), secondaires.

**CE QUI EST DONC RETIRE** : la conclusion du § 2quin.3 selon laquelle « le record delta
dominant porte quatre composants, donc l accroupissement n y est pas ». Le record DOMINANT en
porte quatre — l oracle Rosette reste vrai — mais le record RARE en porte plus, et c est lui qui
compte pour l accroupissement. Les deux enonces ne se contredisent pas ; le second manquait.

### 2sex.4 CE QUI N A PAS ETE FAIT, ET POURQUOI

Le pilote demandait aussi (2) l ecrivain du masque delta — comment le jeu decide d inclure `i29`
— et (3) la ventilation du `R(32)` de `PlayerGameEventSmall`. **Ni l un ni l autre n est fait.**
Le resultat de (1) les deprioritise : il n y a plus de mystere sur l endroit ou vit
l accroupissement par instant, donc plus besoin de l ecrivain du masque pour le trouver, ni du
canal d evenements comme piste de remplacement. Ce qui reste est un travail de largeurs, et il
commence par `ti=0 i0`.

## 2 septies. CORRECTION DU § 2 SEXIES — `ti=0 i0` N ETAIT PAS LE VERROU, ET VOICI LA VRAIE LISTE

> Ouverture du lot de grammaire 5.3.3. Premiere action : re-verifier le classement des echecs
> AVANT de toucher une largeur. Elle a trouve une erreur de lecture, et la liste change.

### 2sept.1 L ERREUR, ET SA CAUSE EXACTE

Le § 2sexies donnait `ti=0 i0 game-engine-team-mapping` a **13 463 echecs**, « le verrou
principal ». **C EST FAUX.** Sur le chemin delta, `DecodeFrameRecords` pose, quand le test de
generation echoue :

```go
rec.Trace = EntityTrace{DesyncAt: 0, EndBit: br.BitPos()}
rec.DesyncAt = 0
break   // aucun composant n est lu, et rec.TypeIndex n est JAMAIS pose
```

`DesyncAt = 0` y est une **SENTINELLE de rejet**, pas un index de composant ; et `TypeIndex`
reste a sa valeur nulle, c est-a-dire **0**. Mon histogramme lisait donc « archetype 0,
composant 0 » la ou le decodeur disait « je n ai meme pas regarde ce record ».

**LE DISCRIMINANT QUI MANQUAIT** : un rejet de generation ne franchit AUCUN composant
(`len(Trace.Comps) == 0`). Avec ce test, la ventilation se separe proprement.

### 2sept.2 LA VENTILATION CORRIGEE, ET LA CORRECTION DE HARNAIS QU ELLE A ENTRAINEE

Seconde erreur trouvee dans la foulee : le monde (`World`) etait remis a neuf **a chaque
chunk**, donc les liaisons slot -> archetype posees par les chunks precedents etaient perdues.
Le monde persiste desormais sur tout le film.

| | avant correction | apres |
|---|---|---|
| trames decodees sans erreur | 9 786 (37,7 %) | **10 512 (40,5 %)** |
| records `ti=35` | 10 060 | **11 150** |
| **rejets de GENERATION** (monde, pas grammaire) | 13 463 | **12 316** |
| **desynchronisations REELLES de grammaire** | 2 709 | **3 130** |

### 2sept.3 LA VRAIE LISTE DES COMPOSANTS FAUTIFS, PAR RENDEMENT

| archetype | echecs reels | composant fautif dominant |
|---|---|---|
| `ti=2` (game-engine) | 442 | **`i15 managed-engine-timers-component` : 331** |
| `ti=5` (joueur) | 429 | **`i22 player-aim-assist-component` : 371** |
| `ti=10` (managed-object) | 354 | **`i5 managed-object-navpoint-component` : 228** |
| `ti=0` | 265 | (disperse) |
| `ti=18` | 145 | — |
| **`ti=35` (bipede)** | **69** | **`i60 simulation-state` : 38 · `i59` : 19 · `i57` : 12** |

**`game-engine-team-mapping` n apparait plus.** Les trois `partiel` du bipede, eux, tiennent :
ce sont bien `i57`, `i59` et `i60` qui gardent l acces aux composants de mouvement.

### 2sept.4 CE QUI NE BOUGE PAS : LA SUR-REPRESENTATION

Elle se RENFORCE apres correction, ce qui est le meilleur signe qu elle est reelle :

| composant | part des ECHECS `ti=35` | part des SAINS | facteur |
|---|---|---|---|
| `i18 unit-control` | **15,94 %** | 0,13 % | **x 123** |
| `i29 unit-crouch` | **15,94 %** | 0,13 % | **x 123** |
| `i55 biped-posture-physics` | **24,64 %** | 0,13 % | **x 190** |

La conclusion du § 2sexies tient donc **entierement sur ce point** : l accroupissement par
instant est dans la trame, dans les records riches, et les records riches echouent. Seule la
LISTE DES COMPOSANTS A CORRIGER etait fausse.

### 2sept.5 CE QUI RESTE LE PREMIER OBSTACLE, ET CE N EST PAS UNE GRAMMAIRE

**12 316 rejets de generation contre 3 130 desynchronisations reelles** : les quatre cinquiemes
des trames perdues le sont parce que le monde ne connait pas encore la liaison slot ->
archetype, pas parce qu une largeur est fausse. Le monde s amorce aux images-cles ET aux
records NEW des deltas — mais un record NEW n est lie que si sa marche est PROPRE, et une trame
qui casse tot n en lie aucun. C est un amorcage circulaire.

**AUCUNE LARGEUR N A ETE TOUCHEE.** Corriger une grammaire avant d avoir leve cet amorcage
reviendrait a mesurer le gain sur une population de trames que le monde mutile encore.

### 2sept.6 L ORDRE DE TRAVAIL QUE CETTE MESURE IMPOSE

1. **L AMORCAGE DU MONDE** — 12 316 trames, quatre fois le total des desyncs de grammaire.
   Question a instruire : d ou la production tire-t-elle ses liaisons (les `world_dump` que
   `frame_records.go` mentionne), et peut-on les charger avant la premiere trame ?
2. `ti=2 i15 managed-engine-timers` (331), `ti=5 i22 player-aim-assist` (371),
   `ti=10 i5 managed-object-navpoint` (228).
3. `ti=35 i60`, `i59`, `i57` (69 au total) — les gardiens des composants de mouvement, et la
   cible finale du lot 5.3.

## 2 octies. ETAPE (1) — L AMORCAGE DU MONDE N EST PAS LA CAUSE, ET C EST MESURE

> Lot de grammaire, etape (1). Deux corrections tentees, et une mesure qui les refute toutes
> les deux. **Aucune largeur touchee, aucune perte.**

### 2oct.1 LES DEUX CORRECTIONS TENTEES

1. **Le monde persiste sur tout le film** (deja fait au 5.3.3.0) : 37,7 % -> 40,5 % de trames
   saines.
2. **Liaison par SLOT, generation neutralisee** : `BindWildcard` au lieu de `BindFull` pour
   chaque entite d image-cle — la porte que `world.go` prevoit pour « une liaison slot ->
   archetype dont la GENERATION est INCONNUE ». C est exactement la forme demandee (« la
   liaison ne doit pas dependre d une marche propre »).

**RESULTAT : AUCUN CHANGEMENT. Pas un chiffre ne bouge.** 25 958 paquets cadres, 10 512 trames
saines (40,5 %), 12 316 rejets, 3 130 desyncs reelles, memes fautifs, meme etalon.

### 2oct.2 CE QUE LA MESURE DE COUVERTURE DIT, ET POURQUOI ELLE TRANCHE

Si la generation etait en cause, les slots rejetes seraient les memes que ceux vus dans les
records sains. Mesure :

| | valeur |
|---|---|
| slots DISTINCTS rejetes | **4 568** |
| dont vus AUSSI dans un record sain | **28 (0,6 %)** |
| slots les plus rejetes | 137 (102), 1041 (82), 2616 (63), 3933 (58), 5268 (54) — **tous inconnus** |

**99,4 % des slots rejetes n apparaissent JAMAIS dans un record sain.** Ce ne sont donc ni des
generations qui avancent, ni des entites que l amorcage aurait oubliees : **4 568 slots
distincts, etales de 137 a 5 268, c est plus d entites que n en porte un match.** Ce sont des
identifiants LUS DANS DU BRUIT.

### 2oct.3 LA CONCLUSION, ET ELLE DEPLACE LE PROBLEME

`DecodeFrameRecords` rend la main au PREMIER record en echec. Les 12 316 rejets sont donc
12 316 paquets dont le **PREMIER** record echoue deja son test de generation — avec un slot qui
n existe pas. **Le cadrage de ces paquets est faux des le premier bit de trame**, et tout ce
qui suit est du bruit.

**L amorcage du monde n est donc PAS la cause, et la cible « rejets < 1 000 » n est pas
atteignable par lui.** Le vrai sujet est le CADRAGE : pourquoi, sur 60 % des paquets a liste
d evenements vide, la trame ne commence-t-elle pas ou le preambule le dit ?

Pistes que la mesure designe, aucune instruite ici :

- le preambule vaut `DefaultPacketPreambleBits = 2` pour tous ces paquets ; `event_list.go`
  dit que ces 2 bits sont `[config][continuation=0]` — le filtre `pay[0]&0x40 == 0` teste bien
  le bit de continuation, mais rien ne garantit que le preambule soit de 2 bits pour TOUS ;
- `bpkCalibre` (`biped_pickup_research_test.go`) BALAYE deja `IDLowBits` sur les paquets a
  liste vide pour trouver la largeur qui maximise le taux de trames exactes : **le depot a
  donc deja un instrument de calibration de cadrage**, et il faudrait le rejouer sur ce film
  avant toute grammaire.

### 2oct.4 AUCUNE PERTE, ET RIEN N EST TOUCHE

Le seul octet modifie est dans un `_test.go` sous tag `research` (`BindFull` -> `BindWildcard`,
plus la mesure de couverture). `grammar.Rev`, `facts.Rev`, le ratchet 0.A.3 et les fixtures
sont inchanges par construction. Gates sans decodage verts.

## 2 nonies. LE CADRAGE : `bpkCalibre` DIT 9, L ECRIVAIN DIT 13 — ET C EST L ECRIVAIN QUI A RAISON

> Etape (1) du cadrage. La calibration a ete jouee, puis ARBITREE par l ecrivain, et
> l arbitrage a evite un correctif qui aurait detruit le decodage des bipedes en rendant tous
> les gates verts.

### 2non.1 LA CALIBRATION DU DEPOT, SUR `bfecd02b`

`TestBipedPickupCalibration` (`bpkCalibre`), balayage d `IDLowBits` sur les paquets a liste
vide, 3 000 paquets :

| `IDLowBits` | trames EXACTES | profondeur (record/paquet) |
|---|---|---|
| **9** | **97,5 %** | **1,02** |
| 10 | 5,3 % | 1,46 |
| 11 | 85,2 % | 1,04 |
| 12 | 11,2 % | 2,08 |
| **13 (defaut)** | **5,1 %** | 1,63 |
| 14 | 8,2 % | 1,28 |
| 15 | 28,3 % | 1,05 |
| 16 | 17,2 % | 1,15 |

Temoins de decalage au cadrage retenu : +0 bit **79,0 %**, +1 **0,0 %**, +2 **0,1 %**, +3
**0,0 %**. **Le cadrage au bit 2 est donc juste** — ce n est pas la position de la trame qui est
en cause.

### 2non.2 CE QUE `IDLowBits = 9` FAIT VRAIMENT, MESURE

| | defaut 13 | calibre 9 |
|---|---|---|
| trames decodees sans erreur | 10 512 (40,5 %) | **25 182 (97,0 %)** |
| rejets de generation | 12 316 | **405** |
| desyncs reelles | 3 130 | **371** |
| **records `ti=35`** | **11 150** | **1** |
| etalon `i21` | 64,3 % | **0,0 %** |

**97 % de trames « saines » et UN SEUL record de bipede.** L etalon `i21` a REFUSE de publier —
la garde posee au paragraphe 2 quinquies a fait exactement son travail. Un correctif adopte sur
le seul taux de trames aurait rendu tous les gates verts en detruisant le decodage des joueurs.

### 2non.3 L ARBITRAGE : L ECRIVAIN, ET IL EST DEJA DANS LE DEPOT

`readRecordID` porte `FUN_1406d3140(_, _, 7, _)` — **categorie 7** de la table de plages du jeu.
Et `varwidth.go` ecrit, apres relecture de `FUN_140d10bb0` :

> « `W` ne vaut 13 que pour les categories 0, 1, **7** et 8 — celles dont la plage derive de
> `DAT_144706100` (0x1FFF, et `bitLen(0x1DFF) == bitLen(0x1FFF) == 13`). »

`varWidthRange(7)` retombe sur `varWidthDefaultRange = 0x1FFF`, donc `varWidthBits(7) = 13`.
**LA LARGEUR D ID BAS EST 13, ET ELLE VIENT DE L ECRIVAIN.** Ce n est ni une constante « qui
marche » ni une ligne de profil : c est une categorie de la table du jeu.

### 2non.4 POURQUOI `bpkCalibre` REND 9, ET CE QUE CELA APPREND DE L ORACLE

Son critere d exactitude est « la trame consomme le payload a moins d un octet pres ». **Une
largeur TROP PETITE le satisfait de facon degeneree** : la profondeur tombe a **1,02
record/paquet** — un record par paquet, la ou un paquet delta a 60 Hz en porte plusieurs
dizaines. La trame « ferme » parce qu elle lit un seul gros record et s arrete, pas parce
qu elle est juste.

**LECON, ET ELLE VAUT POUR TOUT LE CHANTIER** : un oracle de FERMETURE ne prouve pas une
largeur ; il lui faut un oracle de CONTENU (ici : le compte de records par paquet, et la
presence d `i21`). `bpkCalibre` reste valide pour ce qu il fait — departager des cadrages a
largeur EGALE — mais il ne peut pas arbitrer une largeur.

### 2non.5 CONSEQUENCE POUR L ETAPE (2)

**IL N Y A RIEN A CORRIGER DANS `frame_records.go`.** La largeur d ID y est deja celle de
l ecrivain. L etape (2) telle qu elle etait prevue — « corriger le cadrage » — **n a pas
d objet**, et aucun octet de production n a ete touche.

**LA CAUSE DES 12 316 REJETS RESTE DONC OUVERTE**, et deux hypotheses sont eliminees pour de
bon : ce n est pas l amorcage du monde (paragraphe 2 octies), ce n est pas la largeur d ID.
Ce qui reste a instruire, dans l ordre : le PREAMBULE (les 2 bits valent-ils 2 bits pour TOUS
les paquets a liste vide ?), et surtout **`IDBase`** — `FUN_1406d3140` rend
`(queue << 30) | (base + valeur)` avec une base de 0x200 / 0x300 / 0x400 **selon la
categorie**, et `varwidth.go` dit explicitement que cette base **n est pas portee** et que la
changer « se juge au gate de decodage ». Une base fausse decale TOUS les identifiants — c est
exactement le symptome mesure : des slots qui n existent pas.

## 2 decies. `IDBase` — LA TABLE DU JEU, LUE EN ENTIER, ET UN NEGATIF QUI FERME LA PISTE

> Troisieme hypothese du cadrage, instruite chez l ecrivain. **Negatif : le port etait deja
> exact. Aucun octet de production touche.** Et la lecture rend au passage une table complete
> que le depot n avait pas, plus une validation croisee du zoom.

### 2dec.1 L ECRIVAIN DE LA TABLE, LU EN ENTIER

`FUN_1406d3140` lit sa base et sa plage dans une table indexee par categorie :

```c
uVar7 = DAT_144706100;                            // plage par defaut
if (DAT_144706104 != '\0') {
    uVar8 = (&DAT_1451f98d0)[param_3 * 2];        // BASE de la categorie
    uVar7 = (&DAT_1451f98d4)[param_3 * 2];        // PLAGE de la categorie
}
```

Et `FUN_140d10bb0` REMPLIT cette table, categorie par categorie (`piVar2` = la plage,
`piVar2[-1]` = la base, boucle `iVar3` de 0 a 8) :

| categorie | BASE | PLAGE | largeur `bitLen(plage)` |
|---|---|---|---|
| 0 | `0x200` | `0x1DFF` | 13 |
| 1 | `0x200` | `0x1DFF` | 13 |
| 2 | `0x200` | `0x100` | 8 |
| 3 | **`0x300`** | `0x100` | 8 |
| 4 | `0x200` | `0x200` | 9 |
| 5 | **`0x400`** | `0x100` | 8 |
| 6 | `0` | `0x200` | 9 |
| **7** | **`0`** | **`0x1FFF`** | **13** |
| 8 | `0` | `0x1FFF` | 13 |

Les PLAGES concordent exactement avec `varWidthRange` du depot. **Les BASES, que `varwidth.go`
declarait explicitement NON PORTEES, sont desormais lues** — et elles valent bien 0x200 / 0x300
/ 0x400, mais **pas pour toutes les categories** : les categories 6, 7 et 8 ont une base NULLE.

### 2dec.2 LE NEGATIF : LE PORT ETAIT DEJA EXACT

`readRecordID` porte la categorie **7**. Sa base est **0**, sa largeur **13**. Or
`DefaultFrameConfig()` rend deja `IDLowBits: 13, IDBase: 0`.

**LES DEUX VALEURS DU PORT SONT CELLES DE L ECRIVAIN.** Il n y a rien a corriger, aucune mesure
a refaire, aucun gate a jouer : changer `IDBase` reviendrait a s ecarter de l ecrivain. La piste
est FERMEE, et c est un negatif ecrit, pas un abandon.

**TROIS HYPOTHESES SONT DESORMAIS ELIMINEES** pour les 12 316 rejets : l amorcage du monde
(§ 2 octies), la largeur d ID (§ 2 nonies), la base d ID (ici). Toutes trois par l ecrivain ou
par la mesure, aucune par lassitude.

### 2dec.3 VALIDATION CROISEE GRATUITE : LA BASE DU ZOOM ETAIT UNE MESURE, ELLE EST MAINTENANT UNE GRAMMAIRE

`zoom_events.go` porte `zoomSlotBase = 512`, obtenue par FORCE BRUTE : « base 512 : 63 index
sur 64 tombent sur un slot bipede reellement vu dans le film (98 %) ; bases 0, 256, 768, 1024 :
0 sur 64 ». La premiere reference d `unit_zoom` est de **domaine 4**.

**Et la categorie 4 de la table du jeu a pour base `0x200` = 512.**

La constante que sept campagnes avaient cherchee empiriquement est donc la base de sa categorie,
lue chez l ecrivain. Ce n est pas une coincidence a 1 chance sur 4 : c est la confirmation que
le « domaine » d une reference d evenement EST la categorie de `FUN_1406d3140`, et que la table
ci-dessus vaut pour les references d evenements comme pour les identifiants de record.

**CE QUE CELA OUVRE, ET QUI N EST PAS DE CE LOT** : les domaines 7 et 8 d `unit_zoom` ont une
base de 0 et une largeur de 13 ; les domaines 3 et 5, des bases 0x300 et 0x400. Un lot
d evenements pourrait porter ces bases au lieu de les mesurer.

### 2dec.4 SUITE, DITE COMME CONVENU

La cause des 12 316 rejets reste ouverte, et les trois suspects nommes sont tombes. **Je passe
donc a `ti=35 i60`, `i59`, `i57` sur la population SAINE actuelle** (11 150 records de bipede,
etalon `i21` a 64,3 %), en le disant : ces trois composants gardent l acces a `i29`, `i18`,
`i54`, `i55` et `i62`, et ils sont mesures fautifs 38, 19 et 12 fois.

## 2 undecies. LE PREAMBULE NE DISTINGUE RIEN — ET `ti=35 i60` A UN PREDICAT ENTIEREMENT LISIBLE

### 2und.1 LE PREAMBULE : QUATRIEME HYPOTHESE, QUATRIEME NEGATIF

Mesure sans hypothese sur `bfecd02b` : la distribution de l octet de TETE et de la TAILLE des
paquets, **10 512 sains contre 12 316 rejetes**.

| octet de tete | sains | rejetes |
|---|---|---|
| `0x80` | 139 (1,32 %) | **0** |
| `0x88` | 1 | 1 |
| `0x89` | 28 (0,27 %) | 20 (0,16 %) |
| `0x8A` | 7 | 5 |
| **`0xA0`** | **10 337 (98,34 %)** | **12 290 (99,79 %)** |

**AUCUNE valeur n est propre aux rejetes.** `0xA0` ecrase les deux populations ; la seule
valeur exclusive (`0x80`, 139 paquets) est propre aux SAINS. Les tailles ne separent pas
davantage : les rejetes sont seulement un peu plus GROS (129 paquets de moins de 64 octets
contre 1 964 chez les sains), ce qui s explique sans hypothese — un gros paquet porte plus de
records, donc plus d occasions de casser.

**LA PISTE DU PREAMBULE EST FERMEE.** Quatre hypotheses eliminees pour les 12 316 rejets :
amorcage du monde, largeur d ID, base d ID, preambule. Aucune par lassitude.

### 2und.2 `ti=35 i60 simulation-state` — L ECRIVAIN, ET LA QUEUE N EST PAS UN MYSTERE

`ecs_table.tsv` le declare `partiel` : « structure connue ; **la queue depend d un predicat sur
les vecteurs decodes** ». Lecture de `FUN_142ED6D88` :

```c
FUN_140c1e79c(param_2);                               // (R1[R19] + R8)
cVar1 = FUN_140501798(param_1 + 0xb, param_1 + 0xe);  // LE PREDICAT
if (cVar1 != '\0') {
    FUN_14076e494(param_2, param_1 + 0x11, 0x10, 0, 0, 0);   // la QUEUE : position, axe 16 bits
    ...
}
```

**LE PREDICAT, LU EN ENTIER** (`FUN_140501798`), sur les deux vec3 deja decodes :

```
orthonormes(v1, v2) :=
      | ‖v1‖² − 1.0 | < 0.001   et fini
  et  | ‖v2‖² − 1.0 | < 0.001   et fini
  et  | v1·v2 − 0.0 | < 0.001   et fini
```

Constantes relues dans l image, aucune devinee :

| adresse | valeur | role |
|---|---|---|
| `DAT_143cd8370` | **0.0f** | le produit scalaire vise |
| `DAT_143cd8374` | **1.0f** | la norme visee (la meme constante qu au § 2.1) |
| `DAT_143cd8380` | **`0x7FFFFFFF`** | masque de valeur absolue |
| `DAT_143cd84bc` | **0.001f** | l epsilon |

C est un **test d orthonormalite a 10⁻³** : la queue n est lue que si les deux vecteurs forment
une base orthonormee valide.

**CE QUE CELA CHANGE, ET C EST DECISIF** : le predicat porte sur des valeurs **DEJA DECODEES DU
FLUX**, pas sur un octet d etat RAM. Il est donc **entierement calculable par le decodeur**, et
`i60` peut devenir bit-exact — contrairement a `i57`, dont `ecs_table` dit que son etiquette 3
est « gardee par des octets d etat RUNTIME : desync PROPRE ».

**CHEMIN DE PORT, ECRIT** : decoder les deux vec3 (ils le sont deja : `4 x R(16)` puis
`4 x R(16)` du corps), evaluer `orthonormes`, et ne lire la queue `FUN_14076e494(..., 0x10)`
que si le predicat tient. Aucune constante « qui marche » : les quatre valeurs viennent de
l image.

## 2 duodecies. `ti=35 i60` — LE GATE EST TENU, ET LA CAUSE N ETAIT PAS LA GRAMMAIRE

### 2duo.1 CE QUE LA LECTURE A TROUVE AVANT D ECRIRE UNE LIGNE

`i60 simulation-state` est **DEJA PORTE INTEGRALEMENT**, queue comprise, depuis le lot R7-b du
2026-08-17. Le predicat d orthonormalite du § 2undecies y est meme documente comme « vrai par
construction » : `FUN_140c1e79c` decode une direction unitaire, puis `FUN_1406d8678` construit
un vecteur PERPENDICULAIRE (produit vectoriel, rotation de Rodrigues, normalisation) — une base
orthonormee construite satisfait les trois tests, donc la queue est TOUJOURS lue.

Ce qui fait desynchroniser `i60` n est donc pas une grammaire manquante : c est
**`br.p.Grammaire.SimStateComplet`**, un drapeau de profil a `false` par defaut, que
`dispatch_biped.go` rend tel quel comme valeur de `ported`.

Et ce drapeau porte son critere de bascule, ECRIT : « que le chemin absolu d `i0` tire ses
trois largeurs de la CARTE du match ».

### 2duo.2 LA MESURE, CARTE FOURNIE ET DRAPEAU LEVE

`bfecd02b` ouvert par `NewFilmContextForMap` avec `snowbound` (axes 15/15/17), drapeau pose sur
le profil de la marche :

| | avant | apres | gate |
|---|---|---|---|
| trames saines | 10 512 (40,5 %) | **10 546 (40,6 %)** | |
| records `ti=35` | 11 150 | **11 228** | **>= 11 150 OK** |
| etalon `i21` | 64,3 % | **64,2 %** | **~64 % OK** |
| desyncs reelles | 3 130 | **3 087** | |
| **fautif `i60`** | **38** | **0** | **-> 0 OK** |
| `i29` lu | 7 | **14** | x2 |
| `i62` lu | 6 | **14** | x2 |

**LES TROIS CRITERES DU GATE SONT TENUS**, et l acces aux composants de mouvement DOUBLE.

### 2duo.3 UN DEFAUT DE BRANCHEMENT, DIT PLUTOT QUE CACHE

La premiere tentative a pose le drapeau par `poserBasculeDInstrument`, qui ecrit dans
`profilDInstrument` — le profil du HARNAIS, que cette marche n utilise pas (elle tient le sien
du contexte de film). Elle n a donc rien deplace, et **ce n etait pas un resultat, c etait un
defaut de branchement**. Le drapeau se pose sur `cfg.Profil.Grammaire`.

### 2duo.4 LE CHANGEMENT DE PRODUCTION, SPECIFIE MAIS NON FAIT

**Basculer le defaut global a `true` serait FAUX** : `NewFilmContext` (auto-detecte, sans carte)
sert les enveloppes `ScanFilm*(dir)`, et sur ce chemin les largeurs d axe ne viennent PAS de la
carte — c est precisement ce que le critere interdit. Un defaut global casserait ces appelants.

**LE CHANGEMENT JUSTE** : lier `SimStateComplet` a la PRESENCE des largeurs de carte, dans
`ResolveProfile` — vrai quand une entree de carte est fournie, faux sinon. C est une ligne de
profil qui depend d une config, exactement la forme que la regle impose, et non une constante
« qui marche ».

**NON FAIT DANS CE TOUR**, et c est assume : le changement touche la production, donc il
entraine `grammar.Rev` par empreinte, une entree de chronique, le ratchet 0.A.3 et
`replay-equiv --films=4f77afc1`. Il se fait d un seul tenant, pas en fin de budget.

---

## 2 terdecies. LE PORT DE `SimStateComplet` (5.3.3-a) — LA BASCULE SUIT LA CARTE, ET LA CUISSON N EN VOIT RIEN

> Point (a) du lot 5.3.3, execute le 2026-09-21. Le § 2duo.4 specifiait le changement ; celui-ci
> le POSE, et il le mesure — y compris ce qu il ne change pas.

### 2ter-d.1 CE QUI EST PORTE, ET OU

Deux portes, une regle, ecrite une fois dans `grammaireSousCarte`
(`grammar/profil_balayage.go`) : **la grammaire d un profil qui porte les largeurs d axe de la
carte declare `i60` complet.**

| porte | site | temoin de la presence de la carte |
|---|---|---|
| construction du contexte sous catalogue | `NewFilmContextForMap` (`film_context.go`) | `resolveI0Layout(forced, entry) != nil` |
| installation des largeurs sur un profil deja construit | `ProfilDeBalayage.PoserLargeursObjetDuMondeDepuisDecoupage` | le decoupage passe la garde des axes non nuls |

**LA SECONDE PORTE N EST PAS UN DOUBLON, ELLE EST OBLIGATOIRE.** `replay.poserProfilPuisCarte`
remplace le profil **ENTIER** du contexte par celui que `killsource` a calibre
(`Result.ProfilCalibre`) ; une bascule posee a la seule construction y serait EFFACEE sans un
mot, et la production n aurait rien gagne. La seconde porte est celle de
`killsource.ProfilDeDepartPourCarte` et de l installateur de `replay`, et elle survit a ce
remplacement parce qu elle voyage avec les largeurs.

Le DEFAUT GLOBAL reste faux (`grammaireDuProfil`), comme le § 2duo.4 l exigeait :
`NewFilmContext` sert les enveloppes `ScanFilm*(dir)` et les instruments, ou les largeurs ne
viennent pas de la carte.

Garde-rail : `grammar/simstate_carte_test.go` — cinq cas de profil (defaut global, contexte sans
carte, carte sans largeurs, entree nulle, carte avec largeurs) plus les deux sorties du geste
d installation. Un defaut global leve, ou une porte perdue, le fait rougir.

### 2ter-d.2 LE GATE DE CONTENU, TENU SANS AUCUNE VARIABLE D ENVIRONNEMENT

La marche de reference relancee sur `bfecd02b` / `snowbound` **sans `MOUV532D_SIMSTATE`** — la
bascule vient desormais du code de production, par la carte :

| | passation (drapeau a la main) | 5.3.3-a (bascule portee) | gate |
|---|---|---|---|
| cadres delta a liste vide | 25 958 | 25 958 | |
| trames saines | 10 546 (40,6 %) | **10 546 (40,6 %)** | |
| records de trame | 28 185 | **28 185** | |
| records `ti=35` | 11 228 | **11 228** | **>= 11 228 OK** |
| etalon `i21` | 64,2 % | **64,2 %** | **~64 % OK** |
| etalon `i0` / `i1` / `i25` | — | 62,5 % / 55,6 % / 94,6 % | |
| rejets de GENERATION | 12 316 | **12 325** | |
| desyncs REELLES | 3 087 | **3 087** | |
| dont `ti=35` | 31 | **31** (1,0 %) | |
| **fautif `i60`** | 0 | **0** (absent de la liste) | **-> 0 OK** |
| `i29` lu | 14 | **14** | |
| `i62` lu | 14 | **14** | |

Les fautifs de `ti=35` sont desormais **`i59` (19)** et **`i57` (12)**, et rien d autre : c est
exactement le perimetre du point (c).

### 2ter-d.3 CE QUE LA CUISSON DE PRODUCTION EN VOIT : RIEN, ET C EST MESURE

La question n est pas rhetorique : la bascule entre dans le chemin de `killsource` (donc dans la
calibration), et une bascule de grammaire qui entre dans la calibration peut deplacer des
positions. **Elle ne les deplace pas.** A/B par `replay-build`, cache de faits **vide a chaque
passe** (sans quoi la seconde passe relit les faits de la premiere et rend un faux « identique »
en 172 ms — piege rencontre) :

| film | carte | bascule abaissee | bascule levee | verdict |
|---|---|---|---|---|
| `000d5950` | `cliffhanger` | `bcb2a510…` | `bcb2a510…` | **BIT A BIT IDENTIQUE** |
| `bcb6d393` | `cliffhanger` | `af57c5ea…` | `af57c5ea…` | **BIT A BIT IDENTIQUE** |

`replay-equiv -films bcb6d393` le confirme etape par etape : la **SEULE** etape que la bascule
deplace est le digest de `killsource` — et ce digest porte la VALEUR du profil calibre, ou la
bascule vit maintenant ; son compte (1) et les octets du kill-feed ne changent pas.

**CE QUE CELA VEUT DIRE, ET CE QUE CELA NE VEUT PAS DIRE.** Ce que la bascule ouvre est la
LECTURE DE LA TRAME — la ou `i60` fermait la traversee du bipede avant `i61-63`. Les calques
publies aujourd hui ne consomment rien de ce qui est derriere (D8 : le balayage bipede de
production s arrete a `i21`), donc l artefact ne bouge pas. Le port des etats dans le document
est un autre lot, et il se decide apres la mesure finale.

### 2ter-d.4 REVISIONS

`grammar.Rev` : `grammar-2026-09-20.2` -> **`grammar-2026-09-21`**, avec son entree de chronique.
`facts.Rev` : `killsource-2026-09-20` -> **`killsource-2026-09-21`**, mecaniquement (elle hache
la valeur de la precedente). L entree de chronique des faits DIT ce qu un backlog de redecodage
rapporterait pour cette revision : **rien**, mesure ci-dessus. `SchemaVersion` NE MONTE PAS.

Goldens refiges : `grammar_rev.golden`, `facts_rev.golden`, `types/testdata/shapes.golden`, et
les 8 fixtures de contrat `replay_schema_64_*.json.gz` + leur manifeste — **dont le diff
decompresse ne porte QUE les deux chaines de revision** (verifie champ par champ sur
`bcb6d393` : 35 valeurs, toutes des `grammarRev` / `factsRev` / `layers[*]`).

Ratchet 0.A.3 (`keyframe_closure.golden`) : **aucune ligne en baisse** — il passe inchange.

### 2ter-d.5 DECOUVERTE, NON TRAITEE (consignee au § 6)

`replay-equiv -films bcb6d393` rend un ECART **PRE-EXISTANT** sur l etape `positions` : compte
IDENTIQUE (110 004), sha different de la reference figee. **Il n est pas de ce lot** : il
apparait a l identique quand la bascule est abaissee (meme sha obtenu), donc il vient de la base
`5fd6f02c3` — les references d equivalence ont ete refigees pour la derniere fois a `6e86db356`
(cloture 5.2), AVANT la fusion du lot 5.4. Le re-figeage est un geste du pilote, a la fin.

---

## 2 quaterdecies. L ECRIVAIN DE LA TRAME (5.3.3-b) — LA TRAME A TROIS BOUCLES, ET LE DEPOT LE SAVAIT DEJA

> Point (b) du lot 5.3.3, 2026-09-21. Question posee : quels CHEMINS DE RECORD
> `DecodeFrameRecords` ne modelise-t-il pas, et qu est-ce que le CODE designe comme cause des
> 12 316 rejets au premier record ? Reponse : aucun chemin ne manque au decodeur de records —
> c est le PAQUET qui a trois boucles, et l instrument de 5.3.2 n en lisait qu une.

### 2quat-d.1 LE FRAME-PROCESSEUR, LU EN ENTIER (`FUN_142987460`)

```
DAT_144706104 = R(1)                                   // le drapeau de configuration
pour vue dans 0..2 :                                   // TROIS vues, dans l ordre
    vtable[0x60](vue, 0xa00 - total, sortie + total*0xc0, &n)      // HORS BANDE : aucun bit lu
    vtable[0x40](vue, etat, LECTEUR, 0xa00 - total, sortie + total*0xc0, &n)   // FUN_1406cd128
    total += n
pour vue dans 0..2 : pour chaque record de la plage de la vue : vtable[0x48](vue, record)  // APPLIQUER
FUN_1406d07b0(sortie, total, 0)
```

**QUATRE FAITS, ET CHACUN DIT QUELQUE CHOSE.**

1. **UN SEUL bit d amorce dans cette fonction** — le drapeau de configuration
   (`DAT_144706104`). Le SECOND bit de l amorce de 2 bits du port n est pas ici, et il n avait
   pas a y etre : c est le BIT DE CONTINUATION de la liste d evenements, deja modelise dans
   `grammar/event_list.go` (`[1 bit config][( 1 [R(7) type] ... )* 0][trame de records]`). Le
   « second bit non localise dans le desassemblage » du § DefaultPacketPreambleBits est donc
   localise : c est le bit que l instrument lui-meme teste en `pay[0]&0x40`.
2. **LA BOUCLE DE RECORDS EST APPELEE TROIS FOIS SUR LE MEME LECTEUR.** Chaque vue a sa propre
   fin de trame (son record de type 0). `DecodeFrameRecords` rend la main au PREMIER type 0 :
   il lit la vue 0, et rien de plus.
3. **`vtable[0x60]` PRODUIT DES RECORDS SANS LIRE UN BIT** (aucun lecteur en parametre) : une
   passe HORS BANDE, depuis l etat local de la vue. Ces records existent dans la trame appliquee
   et ne sont dans AUCUN flux — un decodeur hors ligne ne peut pas les voir, et n a pas a les
   chercher.
4. **CAPACITE 0xa00 (2 560) records pour les trois vues**, et un abandon propre par
   `FUN_1406cd3a8` quand elle est atteinte (`if ((capacite <= deja_lus) && (type != 0))`). Ce
   n est pas un chemin de flux : c est la borne du tampon de l appelant.

### 2quat-d.2 LA BOUCLE ELLE-MEME (`FUN_1406cd128`), ET SES DEUX BRANCHES

L en-tete est confirme une troisieme fois, et a l identique du port : prefixe de type
`R(1) -> DELTA` sinon `R(2) in {0,1,2,3}` ; puis l id par la categorie **7**
(`low = R(ceilLog2(DAT_1451f990c)) + DAT_1451f9908`, `tag = R(2)` en bits 30-31).

**IL Y A DEUX BOUCLES DANS LA FONCTION**, selon le global `DAT_14474cd78` :

| | `== 0` | `!= 0` |
|---|---|---|
| lecture de l id | `FUN_1406d3140(_, lecteur, 7, &id)` | la MEME chose, EN LIGNE (memes `DAT_1451f9908/990c`) |
| type 1 (NEW) | `[R(8) garde] FUN_141f86704` | `FUN_1406cbaa0(type, id, ...)` |
| type 2 (DEL) | `[R(8) garde] R(32)` | idem |
| type 3 (DELTA) | garde puis `FUN_141f86b58` | idem |
| sortie | record RANGE dans le tableau (pas de 0xc0) | applique, rien de range |

Les deux lisent la MEME grammaire de bits. Le port en melange les deux moities (le stockage de
la premiere, le dispatch de la seconde) — sans consequence de flux.

**LA GARDE DU DELTA, CHEZ L ECRIVAIN** : `entree = table_de_la_vue + (id & 0x3fffffff) * 0xa0` ;
le record n est decode QUE si `*(uint *)(entree + 8) == id` (l eid ENTIER, generation comprise)
ET `*(short *)(entree + 2) == type`. Sinon la fonction rend 2 et **la boucle s ARRETE** — le jeu
abandonne le paquet, exactement comme `DecodeFrameRecords`. La table est celle de LA VUE
(`vue + 0x38`), pas un monde global.

**L ECRIVAIN D UNE REFERENCE D ENTITE** (`FUN_1406d2464`, categorie 0) confirme la symetrie de
l id : `W(1)` de presence, puis `W(largeur) = (id & 0x3fffffff) - base` — **la base est
SOUSTRAITE a l ecriture** et ajoutee a la lecture, comme le port le fait —, puis `W(2)` du tag.
Et le meme `DAT_144706104` y choisit entre la table et le global : **le bit d amorce n est pas un
bit mort, il SELECTIONNE la table de largeurs d id.** Il vaut 1 sur 100 % du corpus.

### 2quat-d.3 LA CAUSE DES REJETS, DESIGNEE PAR LE CODE — ET ELLE ETAIT DANS L INSTRUMENT

`DecodeFrameRecords` est le port FIDELE de `FUN_1406cd128` : **une** vue. Le port fidele de
`FUN_142987460` existe DEJA dans le depot, et depuis l origine : **`grammar.DecodeFrameViews`**
(`frame_harvest.go`), dont l en-tete dit mot pour mot « The offline decoder previously read only
ONE loop from bit 0 — potentially missing views 1..N ». Sa valeur de PRODUCTION est **HUIT** vues
(`killsource.Options.Views = 8`, `grammar.marchViews = 8`). Et les 16,9 % de paquets a liste
pleine ont eux aussi leur porte dans le depot : `marchLocateStrict` localise le debut de la trame
par la SIGNATURE du premier record (un delta du slot 123, long de 35 bits, a composant unique —
candidat unique et vrai sur 690 paquets sur 690).

**LA MARCHE DE REFERENCE DE 5.3.2 N UTILISE NI L UN NI L AUTRE** : un `DecodeFrameRecords` par
paquet, depuis le bit 2, et un monde lie par les seules images-cles. Elle lisait donc une vue sur
trois et jetait un paquet sur six.

**LA MESURE LE DIT EN TROIS CHIFFRES** (`bfecd02b`, `mouvement_5_3_3b_vues_research_test.go`) :

| constat | mesure |
|---|---|
| paquets delta du film | **31 232** |
| a liste VIDE (lus par la reference) | 25 958 (83,1 %) |
| a liste PLEINE (**jetes** par la reference) | **5 274 (16,9 %)** |
| paquets « sains » gardant **>= 24 bits NON LUS** apres la fin de la vue 0 | **9 006 sur 10 191 (88,4 %)** |
| poursuite de la lecture apres cette fin | **4 869 fins propres (54,1 %)**, 9 992 records de plus |

**ET LES SLOTS REJETES NE SONT PAS DU BRUIT** — c est le point que la passation laissait ouvert.
Sur 4 474 slots distincts pour 12 441 rejets : seuls **47,1 % ne sont vus qu UNE fois**, **24,0 %
apparaissent dans un record NEW** et **23,2 % dans un record sain** ; le plus rejete est le slot
**123** — la signature du premier record du jeu — **365 fois**. Un identifiant lu dans du bruit
uniforme sur 2^13 slots ne revient pas 365 fois. Ce sont de vraies entites qu un monde lie par
les seules images-cles n a jamais liees.

### 2quat-d.4 CE QUE LA MARCHE DU JEU RAPPORTE, MESURE

Meme film, meme monde, meme profil ; `DecodeFrameViews` + `marchLocateStrict` sur les paquets a
liste pleine :

| | reference 5.3.2 (1 vue, liste vide seule) | 3 vues (ce que le CODE dit) | 8 vues (valeur du depot) | 16 vues |
|---|---|---|---|---|
| records lus | 28 185 | 117 753 | **124 828** | 125 010 |
| records `ti=35` | 11 228 | **31 530** | 30 279 | 30 281 |
| etalon `i0` | 62,5 % | 65,2 % | 63,3 % | 63,3 % |
| etalon `i1` | 55,6 % | 59,7 % | 57,4 % | 57,4 % |
| **etalon `i21`** | 64,2 % | **67,3 %** | 67,2 % | 67,2 % |
| etalon `i25` | 94,6 % | **93,4 %** | 92,6 % | 92,6 % |
| **`i29` accroupi lu** | 14 | 89 | **110** | 110 |
| **`i62` glissade lue** | 14 | 89 | **119** | 119 |
| paquets a liste pleine localises | 0 | 4 147 (78,6 %) | 4 006 (76,0 %) | 4 006 (76,0 %) |

**L ORACLE DE CONTENU EST CONSERVE** (`i21` 67 %, `i25` 93 %) et l acces aux etats de mouvement
est multiplie par **6 a 8,5**. `ti=35` passe de 11 228 a **31 530**.

**8 ET 16 SONT INDISCERNABLES** : la marche sature avant la huitieme vue. **3 rend le meilleur
etalon** (`i25` 93,4 % contre 92,6 ; `ti=35` 31 530 contre 30 279) — et 3 est ce que le
frame-processor deroule. La valeur 8 lit 7 000 records de plus pour un etalon legerement plus
sale : elle depasse la trame. **Consigne pour la suite : la mesure canonique se fait a TROIS
vues, 8 reste comme controle de sensibilite.**

### 2quat-d.5 CE QUI N EST PAS CORRIGE, ET POURQUOI

**Aucun octet de production n est touche par ce point.** `frame_records.go` n avait pas d ecart a
combler : il porte `FUN_1406cd128` fidelement, et le porteur de `FUN_142987460` existait deja a
cote de lui. La lecon est celle du point (10) de la passation, appliquee a la lettre : **quand
une mesure contredit le depot, c est l instrument qu on suspecte en premier.**

**DECOUVERTES, CONSIGNEES AU § 6, NON TRAITEES** :

- **`marchViews = 8` contre TROIS vues chez le frame-processor.** Le depot lit au-dela de la
  trame. Mesure ci-dessus : 8 rapporte 7 000 records de plus et degrade `i25` de 0,8 point.
  Reduire la constante a 3 est un lot de production (elle porte les empreintes gelees de
  `killsource` et de la marche des morts d objet).
- **Les 24,0 % de paquets a liste pleine que la signature ne localise PAS** (1 268 sur 5 274).
  La porte existe, son taux est de 76 a 79 % ; le complement demande la grammaire de charge des
  types d evenement — c est le lot d evenements deja consigne.
- **La table d entites est celle de LA VUE** (`vue + 0x38`, pas de 0xa0). Le decodeur hors ligne
  tient UN monde pour les trois. C est une approximation assumee tant que les trois vues
  partagent l espace de slots observe ; un lot qui voudrait la lever devra mesurer si un meme
  slot porte deux archetypes selon la vue.

---

## 2 quindecies. `i59` ET `i57` (5.3.3-c) — L ETIQUETTE VAUT `brut + 1`, ET LA DESYNC D `i57` EST DEFINITIVE

> Point (c) du lot 5.3.3, 2026-09-21. Une lecture d ecrivain par composant, but ecrit avant.

### 2quin-d.1 `i59` — LA LOI DE L ETIQUETTE, LUE A L OCTET

**BUT DE LA LECTURE** : le port du corps `tag==3` rend `ported=false` sur `Inner` hors {1,2}. Que
dit l ecrivain de cette valeur interne, et combien de branches a-t-il vraiment ?

`FUN_142f21c0c` — le lecteur d etiquette appele en tete de `FUN_142f25e90` — fait EXACTEMENT
ceci :

```
*(reader + 0x2c) += 3                   // le compteur de bits avance de TROIS, et de trois seulement
*param_3 = (octet de tete >> 5) + 1     // la valeur RANGEE est brut + 1
```

`FUN_142f25e90` dispatche ensuite sur la valeur RANGEE :

| valeur rangee | brut | ce que l ecrivain lit apres l en-tete commun |
|---|---|---|
| **0** | — | **INATTEIGNABLE** (`brut + 1 >= 1`) : branche morte du point de vue du flux |
| 1 | 0 | quatre mots a -1 ; `FUN_1407f08bc(p+0x76)` ; retour |
| 2 | 1 | deux mots a -1 ; `FUN_1408f0ac4(p+2, categorie 5)` ; `FUN_1407f08bc(p+0x76)` ; retour |
| 3 | 2 | `FUN_1408f0ac4(p, categorie 0)` ; `FUN_1408f0ac4(p+2, cat. 5)` ; **trois** `FUN_142f26e9c` ; `FUN_14076dc04(0x18)` ; une queue courte |
| 4 et 5 | 3, 4 | deux mots a -1 ; `FUN_1408f0ac4(cat. 5)` ; **un** `FUN_142f26e9c` ; ... |
| 6 | 5 | `FUN_1407f08bc(p+0x1e)` ; **porte LUE DANS LE FLUX** (`*(short*)(p+0x1e) == -1`) -> `FUN_1408f0ac4(cat. 5)` ou rien ; `FUN_1408f0ac4(cat. 0)` ; **deux** `FUN_142f26e9c` ; `R(1)` ; `FUN_14076dc04` ; retour |
| 7 et 8 | 6, 7 | **RIEN** : l ecrivain sort du `switch` par son `return` |

En-tete commun a toutes les valeurs non nulles, avant le `switch` :
`FUN_142f26e40(p+0x14, p+0x16, param_4)` puis `FUN_14297ea84`.

**TROIS CORRECTIONS DU DEPOT, ET ELLES SONT PORTEES DANS LA GODOC** :

1. **`AbilityNonPredictedState.Inner` porte le BRUT**, pas l etiquette. Les deux constantes du
   port (`anchorInnerLight = 1`, `anchorInnerHeavy = 2`) designent donc les etiquettes **2 et 3**
   de l ecrivain. Le depot les lisait comme « les etiquettes 1 et 2 ».
2. **L ecrivain a SIX etiquettes (1 a 6), le port en modelise DEUX.** Le « `Inner` hors {1,2} »
   n est donc pas une forme inconnue : ce sont quatre branches ECRITES et non portees, plus deux
   valeurs (bruts 6 et 7) **qui ne portent aucune charge propre**.
3. **La porte `FUN_1407f08bc` est LUE AU MAUVAIS ENDROIT** par le port : il la lit AVANT son
   `switch`, la ou l ecrivain la lit DANS ses etiquettes 1 et 2. C est sans effet sur les deux
   branches portees, et c est le premier obstacle a en porter une troisieme.

### 2quin-d.2 `i57` — LA DESYNC EST DEFINITIVE, ET MAINTENANT BORNEE EXACTEMENT

**BUT DE LA LECTURE** : la branche `tag==3` rend `ported=false` des son premier bit a 1. Le
depot dit « gardee par des octets d etat RUNTIME ». Lesquels, et sont-ils vraiment hors du flux ?

`FUN_142f262d4`, lu en entier, rend TROIS cas et eux seuls :

```
a = R(1) -> dst[0]
si a != 0 :
    FUN_14297ea84                              // R(6)
    si (dst[2] & 1) == 0 : saut a la queue
    sinon : c = R(1), puis
        c == 0                      -> FUN_142f04664(dst+4, br, 0, param_3)
        c == 1 et (dst[2] & 0x10)   -> FUN_1406d3140 (un id d entite) PUIS FUN_142f04664
        c == 1 et !(dst[2] & 0x10)  -> FUN_1406d3140 SEUL, puis saut a la queue
t = R(1) -> dst[1]
si t != 0 : FUN_14076e494(dst+0x18, 0x10, 0, param_3, 0)      // la MEME queue qu i60
```

**LE VERDICT EST NEGATIF, ET IL EST DEFINITIF** : `dst[2]` n est ECRIT PAR AUCUNE lecture de
cette fonction — seuls `dst[0]` et `dst[1]` le sont. Il n est donc derivable ni du flux, ni de
l ecrivain, qui gate sur le MEME octet. Tant qu aucun autre composant ne replique cet octet, la
desync propre EST la bonne reponse, et c est desormais ecrit avec sa raison exacte plutot qu en
gros.

### 2quin-d.3 LE COUT DU MANQUE, MESURE — ET IL DECIDE

Marche a TROIS vues, paquets a liste pleine localises, `bfecd02b` :

| | mesure |
|---|---|
| records `ti=35` | **31 530** |
| dont desynchronises | **38 (0,12 %)** |
| composant fautif | **`i59` 25 · `i57` 13** |
| tags externes d `i59` | 0:2 675 · 1:529 · 2:484 · **3:359** |
| corps `tag==3` parcourus | 359, dont **21 complets (5,8 %)** |
| tags d `i57` | 0:2 848 · 2:543 · 1:522 · **3:416** |

**DECISION, ET ELLE EST ASSUMEE : ON NE PORTE PAS PLUS LOIN DANS CE LOT.** Le manque coute
**0,12 %** des records de bipede. Le porter exige d etablir bit-exactement `FUN_142f26e40`,
`FUN_1408f0ac4` (categories 0 ET 5) et `FUN_1407f08bc` — trois largeurs qu aucune lecture n a
encore rendues — et de DEPLACER la porte du port, ce qui bougerait le curseur sur les corps qui
aboutissent aujourd hui : ceux-la publient `grappleLines[]` au document (schema 8). Un gain de
0,12 % contre un risque sur une sortie publiee, c est un lot en soi, pas une fin de lot.

**CE QUI EST LIVRE A LA PLACE** : la loi et le perimetre, FIGES. `i59_etiquette_loi_test.go`
epingle (a) la loi `brut + 1`, (b) l inatteignabilite de l etiquette 0, (c) pour les HUIT valeurs
brutes, la consommation de bits du port et son verdict de portage. Les six valeurs non modelisees
y sont figees comme NON PORTEES : un lot qui en portera une devra mettre le tableau a jour
DELIBEREMENT.

### 2quin-d.4 REVISIONS

**`grammar.Rev` NE MONTE PAS** — elle reste `grammar-2026-09-21`, la revision de CE lot. Le
changement est de la GODOC seule (deux blocs de commentaire) plus un fichier de test : aucun bit
n est lu autrement, aucune sortie ne peut changer. L empreinte, elle, hache les OCTETS de la
couche : le golden est donc refige AVEC LA MEME REVISION, ce que sa porte prevoit explicitement.
`facts.Rev` hache la VALEUR de la revision : elle ne bouge pas. Aucune fixture de contrat a
refiger.

### 2quin-d.5 REPORT, CONSIGNE AU § 6

**PORTER LES QUATRE ETIQUETTES ECRITES D `i59` (1, 4, 5, 6) ET LES DEUX VALEURS SANS CHARGE
(bruts 6 et 7).** Pre-requis, dans l ordre : (1) la largeur de `FUN_142f26e40` et de
`FUN_14297ea84` ; (2) celle de `FUN_1408f0ac4` en categories 0 et 5 (la table des domaines
d `event_list.go` donne 13 bits pour la categorie 0 et 8 pour la 5 — a confirmer chez
l ecrivain, plus le tag de 2 bits) ; (3) celle de `FUN_1407f08bc` (le port lit porte + R(8), et
l etiquette 6 en relit la valeur comme un SHORT teste a -1) ; (4) le deplacement de la porte
dans le `switch`, avec un temoin sur les corps qui aboutissent aujourd hui.

---

## 2 sexdecies. LA MESURE FINALE DES ETATS (5.3.4) — DEUX FILMS, ET UN NEGATIF QUI TIENT

> Point (d) du lot 5.3, 2026-09-21. La marche est celle du JEU (trois vues, paquets a liste
> pleine localises) ; les valeurs viennent des portes de publication posees par ce point.

### 2sex-d.1 CE QUE CE POINT A DU PORTER AVANT DE MESURER

Cinq deserialiseurs LISAIENT leurs champs et les JETAIENT. Ce point leur donne une porte de
publication, et rien d autre — `etats_mouvement_hooks.go`, un hook, une enumeration stable, un
publieur, sur l idiome de `components_probe.go` :

| composant | ce qui est publie | bits lus |
|---|---|---|
| `i29 unit-crouch` | le booleen d accroupissement, le quantum de PROGRESSION R(10) | inchanges |
| `i62 biped-slide` | la porte, le bloc quantifie (direction R(19), magnitude R(10)), trois fractions R(8) | inchanges |
| `i55 biped-posture-physics` | le tag de posture R(2) | inchanges |
| `i18 unit-control` (tete) | la porte, le premier index R(5), la porte du second, le second R(6) | inchanges |
| `i1 object-translational-velocity` | le mode, la porte, la direction R(19), le mot d echelle R(10) | inchanges |

**LE SLOT VOYAGE AVEC LA VALEUR**, et c est ce qui rend la mesure PAR VIE possible : le publieur
est une methode du LECTEUR, qui porte le slot de capture pose par la boucle de records. Sans lui
une « cadence par slot » ne serait qu un total.

`grammar.Rev` ne monte pas : elle reste `grammar-2026-09-21`, la revision de ce lot — aucun bit
n est lu autrement et la production n attache aucun observateur. Le golden d empreinte est
refige a revision egale. Le litteral `unit-control-component`, passe a quatre copies, est
centralise en `compUnitControl` (regle 6).

### 2sex-d.2 LES DEUX FILMS, TABLEAU PAR TABLEAU

| | `bfecd02b` (Snowbound, 8 joueurs) | `4f77afc1` (Flood Gulch, 24 joueurs) |
|---|---|---|
| paquets lus · dont a liste pleine localises | 30 105 · 4 147 | 30 175 · 12 117 |
| vues franchies | 18 983 | 16 326 |
| records `ti=35` · desynchronises | 31 530 · **38 (0,12 %)** | 23 026 · **57 (0,25 %)** |
| composant fautif | `i59` 25 · `i57` 13 | `i59` 42 · `i57` 15 |
| duree couverte | 521,9 s | 1 203,7 s |
| **`i29` accroupi** : lectures · slots | **848 · 69** | **23 704 · 187** |
| dont booleen POSE | 179 (21,1 %) | 7 120 (30,0 %) |
| dont progression > 0,5 | 196 (23,1 %) | 7 416 (31,3 %) |
| cadence du slot le plus actif | 0,268 rec/s (slot 537) | 1,971 rec/s (slot 659) |
| intervalles (progression > 0,5) | mediane **14,65 s** · p90 89,5 · max 163,6 | mediane **6,77 s** · p90 43,9 · max 230,5 |
| **`i62` glissade** : lectures · porte ouverte | **787 · 155 (19,7 %)** | **21 011 · 6 151 (29,3 %)** |
| intervalles de glissade | mediane **30,31 s** · p90 102,9 · max 155,5 | mediane **9,04 s** · p90 50,7 · max 278,0 |
| **`i54` action de mobilite** : lectures · amorce posee | 4 451 · 1 079 (24,2 %) | 43 798 · 12 391 (28,3 %) |
| **`i55` posture** : lectures · tags | 4 286 · 0:2 795 1:540 2:524 3:427 | 33 558 · 0:17 388 1:5 817 2:5 601 3:4 752 |
| **`i18` tete** : lectures · porte posee | 6 804 · 1 964 (28,9 %) | — |
| **`i1` vitesse** : lectures · dequantifiees | 10 239 · 3 291 | 59 942 · 16 787 |

**CE QUE CES CHIFFRES DISENT, ET C EST LE FOND DU LOT** : l accroupi et la glissade sont LUS, par
vie et a la milliseconde, sur les deux films. Les quatre tags de `i55` se distribuent comme une
machine d etat (un tag dominant a 52 % et trois autres a 13-17 %), et la meme forme se retrouve
sur les deux films — ce n est pas du bruit. Et `i54` porte son drapeau d amorce sur ~une lecture
sur quatre, sur les deux films.

**LA POPULATION DEPEND DU MODE, ET FORTEMENT** : un BTB a 24 joueurs rend **28 fois** plus de
lectures d accroupi qu une arene a 8 (23 704 contre 848), et des intervalles deux fois plus
courts. Une mesure faite sur un seul film d arene sous-estime donc massivement la matiere
disponible.

### 2sex-d.3 LES CINQ INSTANTS PAR ETAT, EN TEMPS DE BARRE THEATER

Temps film depuis le premier paquet delta lu, `mm:ss`, un instant par episode (deux lectures a
moins de deux secondes sur le meme slot ne comptent qu une fois).

**`bfecd02b`** (Snowbound — le match de JGtm ; roster lu DANS le film : Tataaannn, JGtm,
Chocoboflor, Draconewt, MEK1906, Madina97294, SHN Lups99, indahoopty8751) :

| etat | cinq instants |
|---|---|
| **ACCROUPI** | 02:21 (slot 531) · 02:40 (slot 537) · 02:53 (slot 1112) · 02:54 (slot 907) · 03:30 (slot 531) |
| **GLISSADE** | 00:56 (slot 907) · 01:14 (slot 907) · 01:23 (slot 907) · 02:54 (slot 907) · 02:57 (slot 1173) |
| **ACTION DE MOBILITE** | 00:00 · 00:12 · 00:35 · 00:38 · 00:40 |
| **MONTEE (candidat saut)** | 01:57 (slot 7807) · 02:00 (slot 530) · 02:15 (slot 531) · 02:57 (slot 1173) · 03:00 (slot 541) |

**`4f77afc1`** (Flood Gulch, BTB 24 joueurs) :

| etat | cinq instants |
|---|---|
| **ACCROUPI** | 01:56 (slot 521) · 01:56 (slot 530) · 02:05 (slot 521) · 02:12 (slot 521) · 02:13 (slot 5165) |
| **GLISSADE** | 01:55 (slot 521) · 01:56 (slot 530) · 01:59 (slot 527) · 02:05 (slot 521) · 02:08 (slot 535) |
| **ACTION DE MOBILITE** | 01:48 · 01:50 · 01:53 · 01:55 · 01:57 |
| **MONTEE (candidat saut)** | 01:51 (slot 527) · 01:55 (slot 530) · 01:58 (slot 527) · 02:03 (slot 521) · 02:21 (slot 521) |

**LES INSTANTS SONT NOMMES PAR SLOT, PAS PAR JOUEUR, ET C EST DIT** : le roster sort du film sans
aucune base (`ReadPlayerTable` sur le `chunk_00`), mais le JOIN slot -> joueur demande le registre
d identite (record de creation de bipede -> joueur), qui vit dans `killsource` et dans la cuisson
du rejeu. Le faire ici aurait voulu dire recopier ce registre dans un instrument.

**`i54` N A PAS DE SLOT** : son hook (`MobilityActionHook`, anterieur a ce lot) ne porte que ses
deux drapeaux. Ses instants sont donc dates et non attribues. Un lot qui voudrait l attribuer
n a qu a le faire passer par la porte de ce lot.

### 2sex-d.4 LE NEGATIF : L ORACLE DE VITESSE N EST PAS UTILISABLE, ET LE SPRINT RESTE NON TRANCHE

**MESURE, SUR LES DEUX FILMS** — vitesse au sol des records de BIPEDE, `DecodeVelocity` sur le
chemin quantifie :

| | `bfecd02b` | `4f77afc1` |
|---|---|---|
| lectures dequantifiees | 3 291 | 16 787 |
| mediane | 13,31 m/s | 7,89 m/s |
| p90 · max | 149,0 · 343,2 m/s | 139,1 · 346,9 m/s |
| part a **>= 11 m/s** | **1 728 / 3 291 (52,5 %)** | **7 699 / 16 787 (45,9 %)** |

**C EST IMPOSSIBLE POUR UN SPARTAN**, qui marche a ~4,5 m/s et sprinte a ~7. Un Spartan sur deux
a plus de 11 m/s, et le maximum est celui d une balle (la loi du dequantificateur porte jusqu a
350 m/s). **L ORACLE EST DONC REFUTE, PAS LE SPRINT** : soit la loi log/exp ne s applique pas a
cette largeur sur le chemin delta, soit la lecture reste attribuee a tort. **LE SPRINT RESTE NON
TRANCHE**, comme au § 2ter.6, et il le reste pour une raison MESUREE et non par defaut
d instrument.

**LA MEME PRUDENCE VAUT POUR LES « MONTEES »** : 28 a 32 % des lectures dequantifiees portent une
composante verticale > 2 m/s, avec des valeurs jusqu a 178 m/s. Le CANDIDAT SAUT est donc lu,
mais son seuil est un seuil d INSTRUMENT, et la meme incertitude de dequantification le porte.
Les instants de montee ci-dessus sont a lire comme des candidats, pas comme des sauts etablis.

### 2sex-d.5 UNE DECOUVERTE DE HARNAIS, CONSIGNEE

**`decodeInferLoop` NE POSE PAS LE SLOT DE CAPTURE.** `decodeDelta` appelle
`poserSlotDeCapture` ; la boucle d inference que `DecodeFrameViews` emploie ne le fait PAS pour un
record NEW. Une valeur publiee depuis un record NEW herite donc du slot du record PRECEDENT, ou
de zero au premier record du paquet. Mesure de l effet : sans filtre, le slot 0 portait a lui
seul 3 035 des 7 947 lectures d accroupi de `bfecd02b`, et la vitesse « du bipede » montait a
343 m/s parce qu elle venait de projectiles.

**L INSTRUMENT S EN PROTEGE PAR UN FILTRE** (« ne garder que les lectures d un slot LIE au
bipede »), qui ecarte 7 099 lectures d `i29`, 3 036 d `i62` et 19 886 d `i1` sur `bfecd02b`.
**LA CORRECTION DE FOND EST UN LOT DE PRODUCTION** : elle touche l attribution des ECHANTILLONS
DE POSITION du meme chemin, donc une sortie publiee. Consignee au § 6.

---

## 2 septdecies. LA LOI DE `i1` (5.3.5) — ELLE ETAIT EXACTE, ET L INSTRUMENT LISAIT LA CARTE D A COTE

> Point (1) de la reprise du 2026-09-21. **CE SECTION CORRIGE LES CHIFFRES DES § 2quaterdecies,
> § 2quindecies ET § 2sexdecies** : ils ont ete mesures sans installer les largeurs d axe de la
> carte, et ils sous-estiment tout d un facteur 3 a 65. Les valeurs de ce § font foi.

### 2sept-d.1 L ECRIVAIN, LU A L OCTET — ET LE PORT EST EXACT

```
FUN_14076d45c  (l entree d i1)
    b = R(1) ; FUN_14076d4d0(lecteur, dst, b*2)
FUN_14076d4d0  (le repartiteur, PARTAGE avec i62)
    mode 0 (b == 0) : FUN_14076d528(..., min = DAT_143cd88f8, max = DAT_143cd88fc, 10, 0x13)
    mode 2 (b == 1) : FUN_1406d676c(..., 0x60)        -> vec3 BRUT de 96 bits
FUN_14076d528  (le vec3 a precision dynamique)
    g = R(1) ; si g != 0 -> VECTEUR CONSTANT (*PTR_DAT_14474c2f0), AUCUN bit de charge
    sinon : dir = R(0x13 = 19) ; FUN_1406d8288(dir, &u, 19)   -> unitaire cubemap
            m   = FUN_14076d6dc(lecteur, min, max, 10)        -> LE SCALAIRE, LU APRES
            out = u * m
FUN_14076d6dc  (la loi du scalaire)
    raw = R(w) ; n = 1 << w
    raw == 0    -> min
    raw >= n-1  -> max
    sinon       -> exp(raw*step + 0.5*step) - (1 - min),   step = log((1 - min) + max) / n
```

**CONSTANTES RELUES DANS LE BINAIRE** (`/read_memory`, float32 little-endian) :

| symbole | octets | valeur |
|---|---|---|
| `DAT_143cd88f8` | `8fc2f53c` | **0,029999999** (min) |
| `DAT_143cd88fc` | `0000af43` | **350,0** (max) |
| `DAT_143cd8374` | `0000803f` | 1,0 (le 1 de « 1 - min ») |
| `DAT_143cd84b0` | `0000003f` | 0,5 (le demi-pas) |

**VERDICT : `DecodeVelocityMagnitude` TRANSCRIT CETTE LOI TERME POUR TERME**, ses trois
constantes valent celles-ci, l ordre direction-puis-scalaire est le bon et la polarite de la
porte l est aussi (bit a 1 = vecteur constant, zero bit de charge). **LA LOI N ETAIT PAS LA
CAUSE.**

### 2sept-d.2 LA CAUSE, ET C EST ENCORE L INSTRUMENT

**LES LARGEURS D AXE DE LA CARTE N ETAIENT PAS INSTALLEES.** `NewFilmContextForMap` pose la
bascule de grammaire et le decoupage impose, mais PAS le descripteur world-object : en production
c est `replay.installWorldObjectPrecision` qui le fait, juste apres le constructeur. Aucun
instrument de ce lot — ni celui de 5.3.2, ni les miens — ne faisait ce second geste. Le chemin
absolu d `i0` lisait donc ses trois axes aux largeurs de **`cliffhanger` (13/13/14)** sur un film
de **`snowbound` (15/15/17)** : **cinq bits de trop par record**, et tout ce qui suit dans le
record est du bruit.

**L EFFET, MESURE SUR `bfecd02b`** (meme marche, meme monde, meme profil ; seules les largeurs
changent) :

| | sans les largeurs | **avec les largeurs** | facteur |
|---|---|---|---|
| records `ti=35` | 31 530 | **97 447** | x3,1 |
| records `ti=35` DESYNCHRONISES | 38 (0,12 %) | **3 (0,00 %)** | /13 |
| composant fautif | `i59` 25 · `i57` 13 | **`i59` 3, `i57` ZERO** | |
| etalon `i0` | 63,3 % | **85,5 %** | |
| etalon `i1` | 57,4 % | **77,5 %** | |
| etalon `i21` | 67,2 % | **65,2 %** | |
| etalon `i25` | 92,6 % | **97,0 %** | |
| `i1` lectures dequantifiees | 3 291 | **60 783** | x18,5 |
| positions `i0` captees | 25 495 | **96 638** | x3,8 |
| `i29` lectures (slot bipede) | 848 | **1 407** | x1,7 |
| `i62` lectures (slot bipede) | 787 | **1 507** | x1,9 |

**TROIS DESYNCS SUR 97 447 RECORDS**, et `i57` n en cause plus AUCUNE : la « desync propre » de
`i57` que le § 2quindecies chiffrait a 13 etait elle-meme un artefact de largeur. Le verdict
GRAMMATICAL de `i57` (le gate `dst[2]` n est pas derivable du flux) reste vrai ; son COUT est nul
sur ce film.

### 2sept-d.3 L ORACLE INDEPENDANT : LA VITESSE DECODEE SUIT LE DEPLACEMENT

Pour chaque paire de positions successives d une MEME vie : deplacement par seconde, divise par
la vitesse decodee du meme intervalle. **Aucune unite n est supposee — c est la DISPERSION qui
tranche.**

| film | paires appariees | p10 | mediane | p90 | **dispersion p90/p10** | verdict |
|---|---|---|---|---|---|---|
| `bfecd02b` | 59 557 | 0,180 | **0,240** | 0,307 | **1,7** | ETROITE |
| `4f77afc1` | 178 180 | 0,124 | **0,236** | 0,288 | **2,3** | ETROITE |

**LE FACTEUR D UNITE EST LE MEME SUR LES DEUX FILMS (0,240 et 0,236)** — deux cartes, deux modes,
deux builds. Un decodage faux ne rendrait pas deux fois la meme constante. **LA VITESSE DE `i1`
EST DONC LUE JUSTE**, et l aberration du § 2sexdecies.4 (347 m/s, un bipede sur deux au-dessus de
11 m/s) est entierement imputable aux largeurs manquantes.

### 2sept-d.4 LA DISTRIBUTION EN m/s — ET LE SPRINT EST REFUTE COMME OBSERVABLE PAR LA VITESSE

| film | lectures | p10 | mediane | p90 | max |
|---|---|---|---|---|---|
| `bfecd02b` | 60 783 | 1,14 | **2,26 m/s** | 2,88 | 343 |
| `4f77afc1` | 214 284 | 0,82 | **2,27 m/s** | 2,90 | 347 |

Histogramme de la vitesse AU SOL, pas de 1 m/s (`bfecd02b`) :
`0:4 843 · 1:15 960 · 2:37 283 · 3:2 335 · 4:51 · 5:13 · 6:11 · 7:7 · ... · >=25:179`

**UN SEUL MAXIMUM LOCAL, A 2-3 m/s** — 61,3 % de la population sur `bfecd02b`, 56,1 % sur
`4f77afc1`. **AUCUNE SECONDE BOSSE.** Au-dela de 4 m/s la population s effondre (51 lectures sur
60 783, soit 0,08 %), et ce qui reste au-dessus de 25 m/s (179 et 3 668) est la queue des
vehicules et des projectiles montes sur des slots recycles.

**LE SPRINT EST DONC REFUTE COMME OBSERVABLE PAR LA VITESSE**, et c est un negatif MESURE, pas un
defaut d instrument : la loi est exacte, l oracle la valide a deux films, et la distribution n a
qu un mode. Si le sprint existe dans le film, il n est PAS un deuxieme regime de vitesse — il
faut le chercher dans un ETAT (un drapeau), pas dans une grandeur.

### 2sept-d.5 LE SAUT — UNE FORME LISIBLE, MAIS PAS UNE PREUVE

Impulsion = la composante verticale devient positive puis negative sur la MEME vie.

| | `bfecd02b` | `4f77afc1` |
|---|---|---|
| impulsions · slots | 1 817 · 59 | 10 494 · 209 |
| duree (mediane) | 0,234 s | 0,534 s |
| pic de montee (mediane) | 0,367 m/s | 0,699 m/s |
| **impulsions a pic >= 3 m/s** | **270** | **2 805** |
| duree de celles-la (mediane) | **0,632 s** | **1,567 s** |

**LE PIC MEDIAN EST SOUS 1 m/s** : l immense majorite des « impulsions » est le clapotis vertical
d un Spartan qui marche, pas un saut. En ne gardant que les pics >= 3 m/s, `bfecd02b` rend une
duree mediane de **0,632 s** — exactement l ordre d un saut de Halo —, mais `4f77afc1` rend
**1,567 s** avec un p90 a 48 s : **la signature ne tient pas d un film a l autre.**

**VERDICT : LE SAUT N EST PAS PROUVE.** Il est LU (la composante verticale est decodee juste) mais
sa segmentation en episodes repose sur DEUX SEUILS D INSTRUMENT (le pic de 3 m/s, le signe de
vz), et le resultat n est pas stable. **Publier un tel etat serait publier un seuil comme une
donnee** — c est ce que la doctrine du chantier interdit (la grammaire prime, une heuristique est
un repli compte). Report au § 6.

### 2sept-d.6 CE QUE 5.3.5 CHANGE POUR LE PORT

`stances[]` ne peut porter que ce qui est **LU** : `crouch` (`i29`, un booleen du flux),
`slide` (`i62`, la porte du flux) et `mobility` (`i54`, son drapeau d amorce). **`sprint` et
`jump` n y entrent pas** — le premier est refute, le second n est pas prouve. Les deux sont au
§ 6 avec leurs chiffres.

---

## 2 duodevicies. LE PORT AU DOCUMENT (5.3.6) — `stances[]`, SCHEMA 65, ET TROIS GENRES SEULEMENT

> Point (2) de la reprise du 2026-09-21, sur DECISION UTILISATEUR. Le calque est livre ; le sprint
> et le saut n y sont pas, et c est une mesure (§ 2septdecies).

### 2duodev.1 LA FORME PUBLIEE

`stances[]` — UN INTERVALLE PAR (VIE, GENRE) : `{slot, kind, t0, t1}`, sur le meme axe que
`Point.T`. Trois genres, et ce sont les seuls que le film ECRIT :

| genre | composant | ce qui est lu |
|---|---|---|
| `crouch` | `ti=35 i29 unit-crouch-component` | le booleen d accroupissement (+ la progression R(10), lue et gardee dans la lecture) |
| `slide` | `ti=35 i62 biped-slide-component` | la porte de tete du composant |
| `mobility` | `ti=35 i54 biped-mobility-action-component` | le drapeau d amorce |

**PAS DE `players[].stances[]`, ET C EST DELIBERE** : le document n a pas de `players[]` — chaque
calque par vie est un tableau RACINE keye par slot (`equipmentEpisodes`, `grappleLines`,
`groundWeapons`...). La granularite demandee (par vie/slot) est celle-la ; s en ecarter aurait
cree une seconde forme pour la meme chose.

`coverage.stances` publie les denominateurs : `scanned`, `absent`, `records`, `desyncs`, `reads`,
`intervals`, `byKind`, `lives`, `tracksTotal`, `dropped`, `eventPacketsUnlocated`, `mapWidths`.

### 2duodev.2 LE CHEMIN COMPLET, ET LE CHOIX DE MARCHE QUI LE REND POSSIBLE

```
grammar.ScanMovementStates   (movement_states.go, marche du FRAME-PROCESSEUR, 3 vues)
    -> types.MovementStateRead[]           (les TRANSITIONS, datees, par vie)
    -> FilmInputs.MovementStates           (l etage de balayage, etape `movementStates`)
    -> codec des faits REPLAYINPUTS24      (les fixtures d entrees les portent)
    -> Options.MovementStates              (applyTo)
    -> replay.buildStances                 (document_stances.go, le PLIEUR d `episodeAccum`)
    -> doc.Stances + doc.Coverage.Stances   (schema 65)
    -> replaydoc.Stance / StanceCoverage    (jumeaux stocke/servi + parite)
    -> zod + ReplayDocumentReady            (frontiere web)
    -> fiche du joueur                      (un mot, FR/EN)
```

**LE BALAYAGE N EMPLOIE PAS LA MARCHE DES AUTRES CANAUX DE CAPACITE, ET C EST MESURE.** Celle-la
(`walkDeltaBipedRecords`) est un CHERCHEUR D ANCRES : la sonde de production la joue sur
`bfecd02b` et elle annonce `i29` **ZERO** fois et `i62` **UNE** fois sur **162 444** records —
exactement le defaut que le lot 5.3.2 avait constate, parce qu elle ne retient que la population
pauvre `{i0,i1,i21,i25}`. La marche du frame-processeur (`DecodeFrameViews`, trois vues) en rend
**97 447** records `ti=35` dont **3** desynchronises.

**LE PLIEUR D INTERVALLES N EST PAS RECOPIE** (regle 6) : `episodeAccum` d
`equipment_episodes.go` est employe TEL QUEL — memes fenetres de vie, meme cloture a la mort,
meme bornage — et sa sortie convertie. Une seconde machine a etats aurait diverge de la premiere.

### 2duodev.3 CE QUE LE BALAYAGE DE PRODUCTION MESURE (`bfecd02b`, Snowbound)

| | mesure |
|---|---|
| records `ti=35` · desynchronises | **97 447 · 3** |
| lectures RETENUES | **7 941** — `crouch` 2 490 (495 posees, 19,9 %) · `mobility` 2 964 (758, 25,6 %) · `slide` 2 487 (386, 15,5 %) |
| slots distincts | **101** |
| paquets decodes | 29 308, dont 5 274 a liste d evenements (3 350 localises, **1 924 non localises**) |
| lectures ECARTEES (slot non lie au bipede) | 7 849 |
| doublons dedupliques (re-parcours du chemin d inference) | 2 563 |
| largeurs d axe employees | **[15 15 17]** — celles de la carte, publiees dans la couverture |

**LES DEUX COMPTEURS D ECART SONT PUBLIES, ET C EST LE POINT** : `dropped` (7 849) dit le prix de
l attribution partielle du chemin d inference (D13), `eventPacketsUnlocated` (1 924) la part du
film que la signature ne localise pas. Sans eux, « 7 941 lectures » ne se jugerait pas.

### 2duodev.4 LA CHECKLIST DE MONTEE DE SCHEMA, ITEM PAR ITEM

| item | etat |
|---|---|
| `SchemaVersion` 64 -> 65 | fait |
| chronique `document_chronicle.go` v65 | fait (ce qui monte, ce qui ne monte pas, les deux negatifs) |
| `structure_test.go` : la raison ecrite + la garde | fait |
| `document_shape.golden` | refige (empreinte `9f8aa1d5`, schema 65) |
| jumeaux `replaydoc` / `replayview` + parite | faits (`Stance`, `StanceCoverage`, deux convertisseurs ; `byKind` RECOPIEE) |
| plafonds justifies | `document_chronicle.go` 1825 -> 1863 et `structure_test.go` 1215 -> 1221 (exception ECRITE, meme commit que `SchemaVersion`) ; `film_scan.go` 504 -> 472 par DEPLACEMENT PUR (`film_scan_mouvement.go`) |
| surface citee hors decodeur | 260 -> **262** avec sa justification datee (`replay.Stance`, `replay.StanceCoverage`) |
| `layers` | un calque `stances` sous `grammar.Rev` |
| codec des faits | `REPLAYINPUTS23` -> **`REPLAYINPUTS24`**, chronique v24 ; 8 fixtures d entrees refigees PAR LEUR PORTE (re-decodage des 8 films) |
| goldens d assemblage | 8 refiges |
| fixtures de contrat | 8 renommees `replay_schema_65_*` + manifeste ; les 8 de la v64 supprimees ; 2 717 376 o sur un plafond de 3 145 728 |
| formes de `film/types` | `MovementStateRead` et `MovementStateStats` inscrites dans `formesFigees`, golden refige |
| contrat | `wantReplayDocumentFields` 60 -> **61** + son entree de chronique ; `stances` decrit des DEUX cotes |
| zod web + frontiere | `replayDocumentSchema`, `replayNormalize`, `replayReadyTypes`, et les deux listes de tableaux nullables (`stances`, `coverage.stances.mapWidths`) |
| OpenAPI **EN DERNIER** + `make generate-types` | faits, dans cet ordre |
| web MINIMAL | un mot sur la ligne du nom de la fiche, FR/EN, `text-muted-foreground` (token semantique) |

**LE LIBELLE DE `mobility` EST « ACTION », PAS « ESCALADE »** : le film transmet qu une action de
mobilite est AMORCEE, il ne dit pas LAQUELLE — l enum a trois candidats et le domaine mesure de
ses champs contredit l hypothese a quatre valeurs (§ 2.8 et D9). Nommer « Escalade » aurait ete
choisir a la place de la mesure. FR/EN : `Accroupi`/`Crouched`, `Glissade`/`Slide`,
`Action`/`Action`.

### 2duodev.5 LE GATE DE DECODAGE, ET CE QU IL A DIT

`replay-equiv -films bcb6d393`, cache de faits vide a chaque passe. **Avant re-figeage** : ECART
sur 13 etapes sur 57 — et la lecture est nette :

- **2 etapes NEUVES** (`movementStates`, `movementStates.stats`), inserees entre `bombReads` et
  `grenades` ;
- **9 etapes DECALEES D UN RANG** : chacune lit les valeurs attendues de sa voisine precedente
  (`grenades` obtient le sha de `movementStates`, `projectiles` celui de
  `movementStates.stats`...). Decalage POSITIONNEL pur, pas une regression ;
- `positions` : l ecart **PRE-EXISTANT** (D15), au sha IDENTIQUE a celui mesure au lot 5.3.3-a
  bascule abaissee ;
- `killsource` : le sha du profil calibre, deja constate au lot 5.3.3-a ;
- `artifact` : **attendu** — le document monte au schema 65.

**APRES RE-FIGEAGE de ce seul film** : `1 identique, 0 different`, et la passe suivante le
confirme (`identique`). Le re-figeage des **19 autres** films reste le geste du pilote a la fin
du chantier — leur reference est structurellement perimee par les deux etapes neuves, pas par une
regression.

---

---

## 3 bis. LOT 5.7 — LE SPRINT EST UN DRAPEAU RAM, LE SAUT A UN SEUL CANDIDAT, ET LA PORTE DES ETATS PUBLIE LES ESSAIS DE LA MARCHE

> Lot 5.7, 2026-09-21, demande utilisateur : « le SPRINT et le SAUT du Spartan a l instant, lus
> dans le film ». Base `7af38c44d` (schema 65, lots 5.1 a 5.6 fusionnes), branche
> `feat/decfilm-57`. Doctrine tenue : l ecrivain dans Ghidra d abord, la mesure ensuite avec
> l oracle de contenu et les largeurs d axe de la carte installees ; un film a la fois ; aucune
> base DuckDB.
>
> **CE LOT S ARRETE SUR UNE DECISION DE VALEUR, ET ELLE EST AU § 5.7.4.** Ce qui suit dit ce qui
> est prouve, ce qui est refute, ce qui est porte, et pourquoi la question du saut ne peut PAS
> etre tranchee tant que la decision n est pas prise.

### 5.7.1 L ECRIVAIN — QUATRE LECTURES, ET CE QU ELLES TRANCHENT

#### 5.7.1.a LE SPRINT N EST PAS UN CHAMP REPLIQUE : C EST LE BIT 45 DES DRAPEAUX D UNITE

Le jeu a son propre accesseur, et c est lui qui repond. `FUN_140fe6664` enregistre CINQ
fonctions de script d un coup, avec leur chaine et leur implementation :

| chaine | adresse | implementation |
|---|---|---|
| `SpartanAbilityIsSprinting` | `1436f7170` | `142a0c70c` |
| `SpartanAbilityGetSprintFraction` | `1436f7150` | `142a0c6e0` |
| `SpartanAbilityIsClambering` | `1436f7130` | `142a0c6e8` |
| `SpartanAbilityIsEvading` | `1436f7118` | `142a0c6f4` |
| `SpartanAbilityIsSliding` | `1436f7100` | `142a0c700` |

`FUN_142a0c70c`, lu en entier, tient en une ligne :

```
obj = FUN_140477618(&poignee, 1)                 // resolution de la poignee en objet
return (*(u64*)(obj + 0x8b8) >> 0x2d) & 1        // BIT 45 du mot de drapeaux de l unite
```

**LE SPRINT EST DONC UN BIT DE DRAPEAU D UNITE EN RAM, a l offset 0x8b8 de l objet, rang 45.**
Reste a savoir si le film le replique. Quatre negatifs, tous mesures sur l image :

1. **AUCUNE instruction de l image ne touche `[x+0x8bd]`** — l octet qui porte le bit 45
   (`search_instructions`, 0 resultat). Les octets voisins, eux, en ont : `0x8bc` 89 references,
   `0x8be` 42. Le bit 45 n est donc jamais adresse a l octet.
2. **Aucun `BTS` ni `BTR qword ptr [x+0x8b8], 0x2d`.** Il y a 14 `BTS` et 12 `BTR` sur ce mot,
   aux rangs 8, 9, 0xb, 0xd, 0xe, 0xf, 0x11, 0x12, 0x16, 0x18, 0x1a, 0x1d, 0x1e — **jamais
   0x2d**.
3. **Aucune des 16 fonctions qui assignent le mot ENTIER** (`MOV qword ptr [x+0x8b8], REG`) ne
   charge le masque `1 << 45` (`0x200000000000`). Le masque existe, mais uniquement dans des
   LECTURES (`TEST qword ptr [x+0x8b8], RAX` — `FUN_140775a24`, `FUN_140776c4c`, `FUN_1407fa018`,
   `FUN_140800ea8`, `FUN_1406de83c`, ...).
4. **AUCUN deserialiseur de `ti=35` n ecrit a `obj+0x8b8`.** Les offsets d objet que les
   deserialiseurs du bipede ecrivent, relus un par un :

   | composant | offset dans l objet |
   |---|---|
   | `i16 object-physics-flags` | `0x4dc` (un `uint` de drapeaux) |
   | `i18 unit-control` | `0x544` (mot 32 b), `0x548`, `0x726` (deux index) |
   | `i29 unit-crouch` | `0x7e8` (booleen), `0x7ec` (fraction) |
   | `i63 biped-action` | `0xaa8` |
   | `i54 biped-mobility-action` | `0x11f8`, `0x1295`, `0x1296` |
   | `i62 biped-slide` | `0x129c` |
   | **`i55 biped-posture-physics`** | **`0x12b4`** |
   | `i57 biped-spartan-ability` | `0x12e4` |
   | `i59 …-non-predicted-state` | `0x1324` |

   Et l APPLIQUEUR d etat replique — `FUN_1406c9b1c`, la fonction qui recopie le tampon decode
   (`puVar5`, aux offsets ci-dessus) vers l objet vivant (`uVar13`) — ne touche `obj+0x8b8`
   qu UNE fois, au **bit 54** (`| 0x40000000000000`), et sa source est un champ de l objet
   VIVANT (`FUN_140719698(obj) + 0x120` / `+0x121`), pas un champ du tampon.

**VERDICT : le drapeau de sprint n est pas replique.** Ce n est pas « le film ne le porte pas »
au sens ou on aurait cherche et pas trouve : c est l ecrivain qui dit que ce bit n a pas de
chemin d ecriture depuis le flux. Et cela CONCORDE avec le negatif mesure du lot 5.3.5 (mediane
2,26 m/s, un seul mode a 2-3 m/s, 0,08 % au-dela de 4 m/s) : les deux bouts tiennent le meme
verdict.

#### 5.7.1.b LES SIX ETIQUETTES D `i59` SONT CELLES D UNE CAPACITE SPARTIATE, ET L IMAGE N EN NOMME AUCUNE

`FUN_142f2679c` (le tag externe d `i59`), relu a l octet :

```
v = R(FUN_1406d310c(4)) = R(2)
*(int*)(etat + 0x1324) = v - 1                   // -1 = aucune
if (v - 1 == 2) FUN_142f25e90(etat + 0x1338, ...)  // le corps lourd, POUR CETTE VALEUR SEULE
```

Trois lecteurs de ce meme champ nomment ce qu il est :

- `FUN_142f020a4` teste `*(int*)(obj + 0x1324) == 2` avant d appeler `FUN_142f24e34(obj+0x1338)` ;
- `FUN_142f0db08` s en sert d INDEX : `lVar2 = FUN_14049d3a0(obj)` (le gestionnaire de capacites),
  puis `lVar1 = lVar2 + (*(int*)(obj+0x1324)) * 4` et un appel virtuel `vtable+0x120` sur la
  definition resolue depuis `lVar1 + 0x1c` ;
- l APPLIQUEUR d `i59` — `FUN_140de19b4`, appele par `FUN_1406c9b1c` sous le bit 26 du masque de
  changement — parcourt `mgr+0x1c` a `mgr+0x28`, c est-a-dire **TROIS emplacements de capacite**,
  et passe `etat+0x1324` a `vtable+0x218` de CHACUN.

**Donc le tag externe d `i59` designe un EMPLACEMENT DE CAPACITE SPARTIATE (trois, plus « aucune »
a -1), et les six etiquettes internes sont la machine d etat de la capacite qui occupe cet
emplacement** — ce que le lot 5.3.3-c avait deja mesure par ailleurs (paires a 0,150 s = le
grappin). Aucune des six n est le sprint, aucune n est le saut.

**ET AUCUNE N EST NOMMEE DANS L IMAGE.** L octet d etiquette vit a `obj + 0x1386` (base
`0x1338` + `0x4e`) : `search_instructions` rend **0 reference** a `0x1386` comme a `0x1387`. Le
jeu n y accede que par le pointeur de composant, et aucune chaine ne s y attache. Nommer les six
demanderait de descendre dans la classe concrete de chaque capacite (son `vtable+0x218`) — un lot
en soi, et consigne comme tel.

**TROIS FAITS D ECRIVAIN QUE LA NOTE DU 5.3.3-c N AVAIT PAS**, releves en relisant
`FUN_142f25e90` en entier :

| etiquette | ce que l ecrivain lit apres l en-tete commun |
|---|---|
| 1 | quatre mots a -1 ; `FUN_1407f08bc(p+0x76)` ; retour |
| 2 | deux mots a -1 ; `FUN_1408f0ac4(p+2, cat. 5)` ; `FUN_1407f08bc(p+0x76)` ; retour |
| **3** | `FUN_1408f0ac4(p, cat. 0)` ; `FUN_1408f0ac4(p+2, cat. 5)` ; **trois** `FUN_142f26e9c` ; `FUN_14076dc04(p+0x1a, 0x18)` ; **puis une QUEUE DE 9 BITS** et `FUN_140809d94(p+0x1d, DAT_144976b50, v9 - 1)` |
| **4 et 5** | deux mots a -1 ; `FUN_1408f0ac4(p+2, cat. 5)` ; **un** `FUN_142f26e9c` ; **`FUN_14076e494(p+0xd, 0x10, ...)`** ; `FUN_14076dc04` ; **la MEME queue de 9 bits** |
| **6** | `FUN_1407f08bc(p+0x1e)` ; **porte LUE DANS LE FLUX** (`*(short*)(p+0x1e) == -1`) -> `FUN_1408f0ac4(cat. 5)` ou deux mots a -1 ; `FUN_1408f0ac4(p, cat. 0)` ; **deux** `FUN_142f26e9c` ; `R(1)` ; `FUN_14076dc04` ; **PAS de queue de 9 bits** |
| 7 et 8 (bruts 6 et 7) | rien : l ecrivain sort par son `return` |

Le port du depot ne modelise que les etiquettes 2 et 3 (ses `anchorInnerLight = 1` et
`anchorInnerHeavy = 2`, valeurs BRUTES), et sa queue `Tail9` est exactement la queue de 9 bits
de l etiquette 3. **Les etiquettes 4, 5 et 6 sont desormais portables avec des feuilles que le
depot a DEJA** (`consume1408f0ac4` en categories 0 et 5, `consumeSimStateHandleTail` pour
`FUN_14076e494(..., 0x10)`, `consumeAbilityAnchorVec` pour `FUN_142f26e9c`, `R(24)` pour
`FUN_14076dc04(..., 0x18)`) — mais le lot 5.3.3-c a DECIDE de ne pas aller plus loin, et ce lot
ne revient pas sur cette decision : c est consigne au § 6, pas traite ici.

#### 5.7.1.c `i57` EST LA CHARGE DE LA CAPACITE, ET LE DEPOT LE LISAIT JUSTE

`FUN_142f268c4` confirme le port au bit : `param_4 < 2` -> `R(FUN_1406d310c(4)) = R(2)` et
`etat[3] = v - 1` ; `param_4 >= 2` -> `FUN_142f21cf0` qui lit la MEME largeur ; puis
`etat[3] == 0` -> `FUN_142f25d78`, `etat[3] == 2 && param_4 > 1` -> `FUN_142f262d4`. Rien a
corriger.

Ce que l APPLIQUEUR en fait le nomme : `FUN_140f8f300(mgr, etat+0x12e4)` boucle sur les TROIS
emplacements de capacite et, pour chacun, lit l octet `etat+0x12e4+6+emplacement` — **sentinelle
`0x7F`**, la signature de la jauge `R(7)` par charge d `i56` — puis ecrit un FLOTTANT dans la
capacite (`*(float*)(cap + 6*8) = fraction`). `i56` et `i57` sont donc bien le couple
jauge/etat d une capacite d armure, et ni l un ni l autre n est le sprint.

#### 5.7.1.d L ETAT AERIEN EXISTE, IL A TROIS CLASSES, ET `i55` EST SON DISCRIMINANT

L image ne connait que **TROIS** classes d etat de bipede — ce sont les seules chaines
`c_biped_*` du binaire :

| classe | chaine | enregistrement (table de champs reflechie) |
|---|---|---|
| `c_biped_ground_state` | `143e2b730` | `FUN_1431be7cc` |
| **`c_biped_airborne_state`** | `143e2bfc0` | `FUN_1432226c0` |
| `c_biped_vehicle_state` | `143e2bfd8` | `FUN_143222af4` |

Et `i55 biped-posture-physics-component` a exactement quatre voies. `FUN_142f0293c` appelle
`FUN_141015c90(obj + 0x12b4)` (0 bit, une mise a neuf) puis `FUN_142f1f630`, qui lit `R(2)` et
passe le tag a `FUN_141fd997c`. **`FUN_141fd997c` N EST PAS UNE « RESOLUTION D ETAT, 0 BIT LU »
COMME LE DISAIT LA GLOSE DU DEPOT : c est le REPARTITEUR D UNE UNION DISCRIMINEE.** Il POSE UN
OCTET DE GENRE — `*(u8*)(dst + 0x2c)` = **1, 2, 3** pour les tags 1, 2, 3 — et appelle un lecteur
de charge DIFFERENT par tag ; le tag 0 prend une quatrieme voie qui ne pose pas l octet.

C est la decouverte **D1** du lot 5.3, confirmee chez l ecrivain et chiffree : les quatre charges
lisent de **2 a plus de 120 bits**, et le depot les sautait toutes (§ 5.7.3 pour la grammaire
portee, largeur par largeur).

**LA CORRESPONDANCE TAG -> CLASSE N EST PAS ETABLIE, ET ELLE NE PEUT PAS L ETRE PAR L IMAGE** :
aucune chaine ne s attache a l octet de genre. Elle se tranchait par la MESURE — et c est la que
le lot bute (§ 5.7.2).

#### 5.7.1.e LES DEUX AUTRES CANDIDATS, POUR MEMOIRE

**`i18 +0x544`** : rien de neuf, et D7 reste refutee. Chez l appliqueur, `FUN_1406c9b1c` charge
`*(uint*)(etat + 0x544)` dans une case de pile que `FUN_1409986f8` REECRIT immediatement (celle-ci
prend sa valeur de `FUN_1407f21b4`, pas du mot d `i18`) ; aucun consommateur nomme. Le mot reste
ce que la mesure de 5.3.2 disait : opaque, a forte entropie, aucun bit predictif.

**`PlayerGameEventSmall` (type 82, 578 en tete sur `bfecd02b`)** : la grammaire descend d un
niveau par rapport a la note du 5.3.2 ter.

```
FUN_14080add8 :
    FUN_14080b30c(ev)           0 bit   (ev[0] = 0xffffffff ; ev+8 = 0 ; FUN_14080b428(ev+0x10))
    *(u32*)(ev + 0xa0) = 0      0 bit
    FUN_14080ae70(ev, lecteur) : R(32) -> ev[0]
                                 R(8)  -> ev+8
                                 FUN_14080b1b8(ev + 0x10, lecteur)    largeur NON RELEVEE
                                 FUN_14080b034(ev + 0x78, lecteur)    largeur NON RELEVEE
    FUN_14080ae28(lecteur, ..., ev + 0xa0) : 32 x R(1), empaquetes UN A UN dans un u32
```

**La seconde lecture que la note declarait « non relevee » est donc un MASQUE DE 32 DRAPEAUX, lu
bit a bit** — exactement la forme d un champ d actions. C est le candidat vivant le plus
prometteur pour le saut, et il n est PAS porte ici : deux feuilles restent inconnues, et le canal
d evenements est hors du perimetre de l enum `kind`. Consigne au § 6.

### 5.7.2 LA MESURE — ET LE DEFAUT D INSTRUMENT QU ELLE A TROUVE

> `bfecd02b` (Snowbound, Team Slayer), UN film, aucune base DuckDB, aucun artefact.
> Instruments : `grammar/mouvement_5_7_*_research_test.go` (cinq fichiers, tag `research`).

#### 5.7.2.a L ORACLE DE CONTENU, REPRODUIT A L IDENTIQUE

Marche du jeu (`DecodeFrameViews`, TROIS vues, paquets a liste pleine par `marchLocateStrict`,
largeurs d axe de la carte installees) :

| | mesure |
|---|---|
| paquets lus / a liste pleine localises / vues franchies | 29 308 / 3 350 / 12 507 |
| records `ti=35` | **97 447** |
| dont desynchronises | **3** (0,00 %), fautif `i59` |
| etalon `i0` · `i1` · `i21` · `i25` | **85,5 % · 77,5 % · 65,2 % · 97,0 %** |

Ce sont EXACTEMENT les chiffres du lot 5.3.5 : l instrument de 5.7 part du meme point.

#### 5.7.2.b LE TEST DU SAUT SUR LA PORTE BRUTE : AUCUN TAG NE PREDIT UNE MONTEE

`i55` par la porte de publication : **3 908 lectures**, dont **1 260 sur un slot lie au bipede**,
sur **70 slots**. Tags : 0 -> 882 (70,0 %) · 1 -> 138 (11,0 %) · 2 -> 119 (9,4 %) · 3 -> 121
(9,6 %).

Test de la montee (fenetre de deux ticks = 40 000 us, MEME vie) :

| tag | lectures | appariees | vz > 0 | vz >= 1 m/s |
|---|---|---|---|---|
| 0 | 882 | 247 (28,0 %) | 146 (59,1 %) | 68 (27,5 %) |
| 1 | 138 | 21 (15,2 %) | 15 (71,4 %) | 9 (42,9 %) |
| 2 | 119 | 34 (28,6 %) | 22 (64,7 %) | 12 (35,3 %) |
| 3 | 121 | 25 (20,7 %) | 16 (64,0 %) | 8 (32,0 %) |

**Aucun tag n approche les 90 % exiges.** Et la CONTRE-PREUVE est ecrasante : sur **5 546 montees
franches** (vz >= 1 m/s) parmi 60 783 lectures d `i1`, **95 seulement (1,7 %)** sont precedees
d une lecture d `i55` de la meme vie dans la fenetre — **98,3 % ne le sont pas**.

Deux tests qui ne dependent pas de l appariement disent la meme chose. **Sejour** (temps jusqu a
la lecture d `i55` suivante de la meme vie) : medianes **2,44 · 1,98 · 2,76 · 2,97 s** — le meme
ordre de grandeur pour les quatre, alors qu un etat aerien dure quelques dixiemes de seconde.
**Matrice des transitions** : chaque tag va vers 0 dans 57 a 74 % des cas, c est-a-dire
proportionnellement a la part de 70 % du tag 0 — **la signature de tirages INDEPENDANTS, pas
d une machine d etat.**

#### 5.7.2.c POURQUOI CES CHIFFRES NE MESURENT PAS LE FILM : LA PORTE PUBLIE LES ESSAIS

C est la lecon de `zoom_events.go` appliquee a l envers : quand la mesure contredit l ecrivain,
on suspecte l instrument. Le compte des lectures a ete confronte au compte des COMPOSANTS QUE LES
RECORDS RENDUS DECLARENT (`Trace.Comps`, clef = le NOM de registre, pas l index).

> Le tableau ci-dessous est mesure APRES le port de grammaire du § 5.7.3, d ou les 3 784 lectures
> d `i55` la ou le § 5.7.2.b en annonce 3 908. L ecart est de 3 % et le facteur de pollution est
> le meme a un point pres : la conclusion ne depend pas de la passe.

| composant | dans les records RENDUS | porte, phase `marchLocateStrict` | porte, phase `DecodeFrameViews` | facteur |
|---|---|---|---|---|
| `object-translational-velocity-*` (`i1`) | **75 488** | 13 221 | 82 401 | **x 1,27** |
| `biped-mobility-action-component` (`i54`) | **321** | 3 059 | 1 555 | **x 14** |
| `biped-slide-component` (`i62`) | **56** | 2 656 | 1 398 | **x 72** |
| `unit-control-component` (`i18`) | **101** | 4 678 | 2 614 | **x 72** |
| `biped-posture-physics-component` (`i55`) | **52** | 2 359 | 1 425 | **x 73** |
| `unit-crouch-component` (`i29`) | **60** | 6 358 | 2 776 | **x 152** |

**LE MECANISME EST NOMME, ET IL EST DANS LE DEPOT.** La porte de publication tire depuis
`traverseComponentLoop`, et DEUX chemins de la marche appellent cette boucle sur des alignements
CANDIDATS dont ils jettent ensuite la quasi-totalite :

- **`marchLocateStrict`** — la localisation du premier record d un paquet a liste d evenements :
  elle essaie des offsets jusqu a ce qu un decodage tienne ;
- **`deltaBodyTrial`** (`frame_chain_infer.go`) — l inference de chaine, qui decode un corps
  candidat **CONTRE CHAQUE ARCHETYPE du registre** (`for ti := range c.w.Reg.Archetypes`) sous
  budget d essais, et ne garde que l alignement gagnant.

Pour un composant declare sur 77 % des records, le bruit d essais pese 1,27 fois le signal et ne
se voit pas. Pour un composant declare sur moins d un record sur mille, il pese **de 14 a 152
fois** le signal. Les trois « tests » du § 5.7.2.b mesuraient donc, pour l essentiel, des
lectures faites a des positions de bit que le decodeur a lui-meme refusees — d ou la signature de
tirages independants.

#### 5.7.2.d LA POPULATION RETENUE, MESUREE PROPREMENT — ET ELLE EST MINUSCULE

Instrument `mouvement_5_7_retenus_research_test.go` : AUCUN hook pendant la marche ; les records
RENDUS sont gardes, et chaque composant d etat est relu **a son `StartBit`**, celui que la boucle
de composants a consigne. Controle d exactitude : **0 ECART DE LARGEUR sur 75 977 relectures** —
la relecture consomme au bit ce que la boucle a consomme.

| composant | lectures sur les records RETENUS | slots |
|---|---|---|
| `unit-crouch-component` (`i29`) | **60** | 27 |
| `biped-slide-component` (`i62`) | **56** | 33 |
| `biped-posture-physics-component` (`i55`) | **52** | 28 |
| `biped-mobility-action-component` (`i54`) | **321** | 39 |
| `object-translational-velocity-*` (`i1`) | 75 488 | 80 |

**LE SAUT NE PEUT DONC PAS ETRE TRANCHE SUR CE FILM.** Des 52 lectures d `i55`, 30 portent un
slot lie au bipede et **20 ont une vitesse tenue** : tag 0 -> 15, tag 1 -> 1, tag 2 -> 3,
tag 3 -> 1. Un denominateur de UN n est pas une mesure. **Le candidat de l ecrivain (l union
d etat physique, dont l une des voies est l etat aerien) reste donc NON TRANCHE — pour une raison
mesuree, et pas pour un manque d essai.**

**ET LES CINQ INSTANTS PAR ETAT EN TEMPS DE BARRE THEATER NE SONT PAS LIVRES, DELIBEREMENT.** Le
brief les demande pour le sprint, le saut et l escalade. Aucun des trois n est etabli : le sprint
est refute chez l ecrivain, l escalade n est pas nommee, et le saut n a pas de tag. Sortir cinq
instants d un tag qu on ne sait pas nommer donnerait a un horodatage l autorite d une preuve —
c est exactement la faute que le § 2 quinquies du 5.3 a evitee en refusant de publier une liste
vide comme un progres. Les instants des TROIS etats DEJA nommes (accroupi, glissade, action de
mobilite) sont livres depuis le 5.3.4, et ils sont a re-lire sous la decision du § 5.7.4 : leur
population aussi vient de la porte polluee.

### 5.7.3 LE PORT DE GRAMMAIRE — LES QUATRE CHARGES D `i55` (D1 FERMEE)

Ce que le depot lisait : `R(2)` et rien de plus. Ce que l ecrivain lit, largeur par largeur —
toutes relues chez lui, la seule feuille neuve etant `FUN_14080bd28` (`R(15)`, masque `& 0x7fff`)
et la largeur de `FUN_14076dc04` etant lue au DESASSEMBLAGE de ses trois sites d appel
(`142f263e5` `LEA R9D,[RBX+0x13]` avec RBX=0 ; `142f2658b` `MOV R9D,0x13` ; `1431c357d`
`MOV R9D,0x13`) :

```
tag = R(2)
tag 0  FUN_142f265dc : FUN_141015cb0 [0 bit] ; g = R(1) ; si g == 0 :
                         k = R(2)
                         k == 1      : R(15) + R(1) + R(1) + R(2)
                         k == 2 ou 3 : R(15) + FUN_1431c3538
tag 1  FUN_142f25a3c : v0 = R(2) ; si v0 != 0 :
                         R(15) + R(2) + R(2) + [queue e494 LEVEL=0x10] + R(19)
tag 2  FUN_142f263ac : [queue e494 LEVEL=0x10] + R(19)
                       g = R(1) ; g == 0 -> R(32) ; g != 0 -> FUN_1408f0ac4(cat. 0)
                       R(15)
tag 3  FUN_142f264f4 : R(15)
                       g = R(1) ; si g != 0 : [queue e494 LEVEL=0x10] + R(19)
                       R(32) + R(1) + R(1)

FUN_1431c3538 = R(2) + R(1) + R(32) + R(1)[si != 0 : R(19)]
```

`[queue e494 LEVEL=0x10]` est `consumeSimStateHandleTail`, le lecteur que le depot porte deja
pour la queue d `i60`. Les deux branches de la porte du tag 2 designent la MEME chose par deux
chemins — un identifiant brut de 32 bits, ou une reference d entite resolue (`FUN_1408e04c8` la
recopie vers `dst+0x18` sans lire un bit) : la signature d un OBJET PORTEUR.

**MESURE, ET C EST L ORACLE DE CONTENU QUI AUTORISE LE COMMIT** :

| | avant | apres |
|---|---|---|
| records `ti=35` | 97 447 | **97 345** (-0,10 %) |
| desyncs | 3 (`i59`) | **6** (`i59` 5, `i57` 1) |
| etalon `i0` | 85,5 % | **85,5 %** |
| etalon `i1` | 77,5 % | **77,5 %** |
| etalon `i21` | 65,2 % | **65,3 %** |
| etalon `i25` | 97,0 % | **97,1 %** |

**LE COMPTE NE MONTE PAS, ET C EST DIT.** On aurait attendu l inverse : `i55` est le 56e des 64
composants du bipede, donc sauter sa charge decale `i56` a `i63` et tronque la queue du record.
Deux mesures expliquent pourquoi le gain est nul. (1) `i55` n est declare que sur **52** des
97 345 records de ce film, dont 50 (96,2 %) portent bien un composant d index superieur : le
manque coutait donc la queue de **cinquante** records, pas de mille. (2) Les essais d alignement
de `deltaBodyTrial` dependent de la largeur d `i55` : la corriger redistribue quelques alignements
gagnants, dans les deux sens. L ecart est de **un pour mille** et l etalon ne bouge pas : la
correction est NEUTRE en couverture et JUSTE en grammaire — c est l ecrivain qui tranche, pas le
compte.

**REVISIONS ET GOLDENS.** `grammar.Rev` -> `grammar-2026-09-21.3` (entree de chronique) ;
`facts.Rev` -> `killsource-2026-09-21.3` (entree de chronique, et elle DIT que le backlog de
redecodage n est pas sans objet cette fois : les records du bipede ne ferment plus aux memes
bits) ; `replay.SchemaVersion` **INCHANGEE a 65** (aucun champ neuf). Ratchet G4 : 121/66 ->
**120/66** — `i55` QUITTE le controle des largeurs entieres au lieu d y changer de colonne (sa
`bits_typ` devient « variable »), et la raison est datee dans la constante. `facts/rev.go`
passait 500 lignes en accueillant son entree : la chronique sort dans
`facts/rev_chronique.go` par **deplacement pur** (510 -> 66 + 455), exactement comme la couche
`grammar`, et le test de chronique lit desormais les deux fichiers.

**LES HUIT FIXTURES DE CONTRAT REFIGEES NE PORTENT AUCUN CHANGEMENT DE CONTENU** : diff des JSON
DECOMPRESSES = **72 lignes, les deux chaines de revision SEULES**, `coverage.stances` identique
a l octet sur les huit films. `replay-equiv -films bcb6d393` SANS `-update` : **3 etapes
divergentes sur 57** — `movementStates` (4 469 -> 4 671 lectures), `movementStates.stats`, et
`artifact` (1 912 592 -> 1 912 635 octets). **Les 54 autres sont IDENTIQUES** : positions,
`killsource`, objectifs, inventaire, equipement, sons, vehicules. Aucune perte.

> ECART ENTRE LES DEUX CHEMINS DE CUISSON, CONSIGNE : le constructeur de fixtures de contrat
> porte les largeurs d axe PAR DEFAUT (`coverage.stances.mapWidths = [13, 13, 14]`) et son
> `movementStates` ne bouge pas ; `replay-equiv` installe les largeurs de la carte du match et le
> sien bouge. Les deux mesures sont vraies, elles ne decrivent pas le meme decodage.

### 5.7.4 (PERIME PAR LE § 5.7.4 CI-DESSOUS) LA DECISION DE VALEUR TELLE QU ELLE A ETE POSEE

> Le pilote a tranche le 2026-09-21 : l issue 1 (filtrer la porte) N EST PAS un choix de valeur,
> c est une correction. Ce paragraphe garde ses CHIFFRES, qui restent la mesure du defaut ; sa
> conclusion (« trois issues, le choix appartient a l utilisateur ») est PERIMEE.

**LE CALQUE `stances[]` DU SCHEMA 65 REPOSE SUR LA PORTE POLLUEE.** `movement_states.go` emploie
exactement la meme marche que l instrument de 5.7 — `marchLocateStrict` puis `DecodeFrameViews`
avec la porte armee — et sa deduplication `(slot, genre, instant)` plus son filtre de slot lie ne
retirent que la part des essais qui tombe sur un slot non lie ou sur le meme instant. Le lot 5.3.6
publie, sur `bfecd02b`, **2 490 lectures d accroupi, 2 487 de glissade et 2 964 d action de
mobilite** ; les records que le decodeur RETIENT en portent **60, 56 et 321**.

Ce n est pas une erreur de grammaire et ce n est pas un seuil deguise : c est une frontiere
d instrument. Trois issues, et le choix appartient a l utilisateur :

1. **FILTRER LA PORTE** : ne publier que les lectures des records retenus. C est la sortie juste,
   et elle divise les comptes de `stances[]` par environ quarante. Les intervalles publies
   changent, `coverage.stances` change, le contenu cuit change — **donc bien au-dela de l enum
   `kind`**, ce que le contrat de ce lot lui interdit de decider seul.
2. **RETIRER `stances[]`** en attendant : le calque a ete livre sur une mesure que ce lot vient
   d invalider.
3. **LE GARDER TEL QUEL** en le disant a la couverture : c est defendable si l on considere qu un
   essai d alignement qui DECODE proprement un accroupi a une position de bit plausible reste un
   accroupi — mais ce n est pas prouve, et personne ne l a mesure.

**ET C EST CE MEME DEFAUT QUI BLOQUE LA QUESTION DU SAUT** : nommer l une des quatre voies de
l union d `i55` demande une population, et la population retenue est de 52 lectures sur un film.
Tant que la decision (1) n est pas prise, aucune mesure de saut ne sera autre chose qu une mesure
d essais.


### 5.7.4 LA PORTE EST CORRIGEE A LA SOURCE — ET LE DEPOT AVAIT DEJA LA DOCTRINE, LE NOM, ET LE MECANISME

> Arbitrage du pilote, 2026-09-21 : « publier depuis une porte qui tire sur des alignements jetes
> est un defaut de CORRECTION, pas un choix de valeur ». Le § 5.7.4 qui precede est donc
> PERIME dans sa conclusion (« trois issues, le choix appartient a l utilisateur ») et conserve
> pour ses chiffres. L issue retenue est la premiere : filtrer a la source.

#### 5.7.4.a LE DEFAUT ETAIT UNE OMISSION D INSCRIPTION, PAS UNE FRONTIERE MANQUANTE

La recherche du § 5.7.2.c nommait deux chemins speculatifs. La relecture du depot en dit plus, et
mieux : **le mecanisme existait deja, sous son nom, avec sa doctrine ecrite**.

```go
// observateur.go, depuis le lot 2.2
// POURQUOI CES DEUX-LA, ET POURQUOI TEMPORAIREMENT. Les chemins d INFERENCE essaient une lecture
// sur des bits qu ils abandonneront peut-etre ; UNE LECTURE SPECULATIVE N EST PAS UNE LECTURE
func (o *Observation) neutraliserCaptures() func() { ... PosCaptureHook, UnitRefHook = nil, nil }
```

Cinq chemins speculatifs la declarent depuis 2026-08 : `repairUnportedComponent`,
`inferUnboundArchetype`, `inferChainArchetype`, `validatedResync`, et les deux essais de
`frame_harvest.go`. **La porte des etats de mouvement, posee au lot 5.3.4, ne s y etait jamais
inscrite** — et le LOCALISATEUR de paquet (`marchLocateStrict` et ses deux voisines) ne declarait
RIEN, alors qu il traverse l entite pour de vrai a chaque offset essaye.

C est donc une omission a deux endroits, et elle explique tout — y compris pourquoi les positions
ne bougeaient pas : `PosCaptureHook` etait inscrit, la porte des etats non.

#### 5.7.4.b LA CORRECTION, ET POURQUOI ELLE EST DANS LA MARCHE

`neutraliserEtatsDeMouvement` est **la porte unique**. Les deux neutralisations existantes
l appellent ; les trois localisateurs l appellent directement. Il n y a AUCUN filtre aval et
aucune logique par calque — un filtre par calque devrait redecouvrir, apres coup, quel alignement
la marche a retenu, c est-a-dire refaire le travail de la marche et diverger d elle des qu elle
change. Ici c est la marche elle-meme qui dit « cette lecture est un essai », et elle seule le
sait.

Ce qui n est PAS eteint, et c est assume : `MobilityActionHook`, la porte HISTORIQUE d `i54`
(celle qui ne porte pas le slot). Elle alimente les balayages de capacite, dont les sorties sont
figees par leurs propres references ; l inscrire deplacerait ces calques sans qu aucune mesure ne
l ait demande. Consigne au § 4 du plan (D5 (5.7)).

#### 5.7.4.c LA PORTE REND EXACTEMENT CE QUE LES RECORDS RETENUS DECLARENT — CONTROLE SUR DEUX FILMS

Confrontation, lecture par lecture, au compte des `Trace.Comps` des records RENDUS par
`DecodeFrameViews` (clef = le NOM de registre, tous archetypes confondus) :

| composant | `bfecd02b` porte / declare | `4f77afc1` porte / declare |
|---|---|---|
| `unit-crouch-component` (`i29`) | **76 / 76** | **236 / 236** |
| `biped-posture-physics-component` (`i55`) | **52 / 52** | **287 / 287** |
| `biped-slide-component` (`i62`) | **56 / 56** | **230 / 230** |
| `biped-mobility-action-component` (`i54`) | **321 / 321** | **1 033 / 1 033** |
| `unit-control-component` (`i18`) | **117 / 117** | **515 / 515** |
| `object-translational-velocity-*` (`i1`) | **79 471 / 79 471** | **220 844 / 220 844** |

**ZERO lecture fantome, sur les douze mesures.** Et la reconciliation ferme des deux cotes :
`bfecd02b` 76 + 56 + 321 = **453** = 400 retenues + 53 ecartees ; `4f77afc1`
236 + 230 + 1 033 = **1 499** = 1 430 + 68 + 1 doublon.

#### 5.7.4.d LE BALAYAGE DE PRODUCTION, AVANT ET APRES

`ScanMovementStates` appele directement (la fonction de production, sous les largeurs d axe de la
carte du match) :

| | `bfecd02b` avant | apres | `4f77afc1` avant | apres |
|---|---|---|---|---|
| records `ti=35` | 97 345 | 97 345 | 321 335 | 321 335 |
| desyncs | 6 | 6 | 56 | 56 |
| **lectures retenues** | **7 463** | **400** | **112 592** | **1 430** |
| ecartees (slot non lie) | 7 940 | **53** | 44 123 | **68** |
| doublons | 2 399 | **0** | 47 670 | **1** |
| accroupi | 2 340 (474 posees) | **52 (6)** | 33 977 (10 459) | **191 (76)** |
| glissade | 2 303 (341) | **47 (2)** | 35 195 (10 603) | **222 (82)** |
| action de mobilite | 2 820 (731) | **301 (262)** | 43 420 (13 039) | **1 017 (837)** |
| vies portant une lecture | 95 / 96 / 98 | **24 / 26 / 29** | 273 / 277 / 275 | **41 / 38 / 73** |

Facteur global : **x 18,7** sur `bfecd02b`, **x 78,7** sur `4f77afc1`. Les DOUBLONS, qui etaient
2 399 et 47 670, tombent a 0 et 1 : c etait la signature du defaut — le meme (slot, genre,
instant) publie par plusieurs essais.

Ecarts entre transitions consecutives d une meme vie, medianes : accroupi 1,001 -> **2,185 s**,
glissade 1,452 -> **1,885 s**, mobilite 0,817 -> **0,017 s** (`bfecd02b`). La mobilite passe a UN
TICK, ce qui est la forme attendue d une amorce d action ; les cadences « autour de la seconde »
d avant etaient celles du localisateur, pas du jeu.

#### 5.7.4.e CE QUI NE BOUGE PAS, PROUVE AU DOCUMENT ET A L ETAPE

**A L ETAPE** — `replay-equiv -films bcb6d393`, SANS `-update` : **3 etapes divergentes sur 57**.
`movementStates` (compte 4 469 -> **726**), `movementStates.stats`, `artifact` (1 912 592 ->
1 910 083 o). **Les 54 autres sont IDENTIQUES a l octet** : `positions`, `killsource`, les tirs,
les pistes, les objectifs, l inventaire, l equipement, les sons, les vehicules.

**AU DOCUMENT** — diff des JSON decompresses des fixtures de contrat, apres re-figeage des
ENTREES par leur porte :

| film | `stances[]` | `reads` | `dropped` | `byKind` |
|---|---|---|---|---|
| `bcb6d393` | 69 -> **18** | 4 469 -> 726 | 5 460 -> 139 | 26/38/5 -> 1/15/2 |
| `000d5950` | 95 -> **49** | 7 004 -> 1 873 | 4 643 -> 128 | 17/66/12 -> 0/49/0 |
| `e5adf7b2` | 945 -> **51** | 31 141 -> 721 | 16 646 -> 37 | 297/361/287 -> 5/44/2 |
| `fb1a1a72` | 89 -> **8** | 8 974 -> 306 | 4 852 -> 92 | 33/38/18 -> 0/8/0 |

**AUCUNE AUTRE CLE DU DOCUMENT NE BOUGE** sur les quatre films controles : seuls `stances`,
`coverage.stances`, `layers` et `coverage.decoder` (les deux dernieres etant les chaines de
revision). Les `tracks`, les `shots`, le kill-feed, les zones : identiques a l octet.

**POURQUOI LES FIXTURES N AVAIENT PAS BOUGE AU PREMIER RE-FIGEAGE, ET CE QUE CELA CORRIGE**
(remplace D2 (5.7), qui parlait a tort de « largeurs par defaut ») : le constructeur de fixtures
de contrat n ouvre AUCUN film — il rejoue les fixtures d ENTREES figees (`inputs_<film>.bin.gz`,
codec `REPLAYINPUTS24`) via `chargerGoldenBuild`. Un changement de decodeur ne les traverse donc
que si l on re-fige les entrees PAR LEUR PORTE (`GoldenBuildsRegenerate` avec
`REPLAY_FILM_CACHE`, plus `GoldenInputsRegenerate` pour `000d5950`), ce qui re-decode les huit
films. C est fait, et cela fait entrer d un coup les deux corrections du lot : la grammaire d
`i55` (5.7.3, d ou les `records` qui bougent de quelques unites) et la porte (5.7.4, d ou
`stances[]`).

#### 5.7.4.f CE QUE LES SEIZE INTERVALLES DE `bcb6d393` DISENT, ET CE QU IL FAUT EN LIRE

Une couverture partielle est un RESULTAT, et `StanceCoverage` le dit par ses propres compteurs.
Sur `bcb6d393`, 726 lectures retenues donnent 18 intervalles sur 12 vies ; sur deux films
l accroupi tombe a ZERO intervalle. Ce n est pas « personne ne s accroupit » : un intervalle
demande une lecture POSEE **et** une lecture LEVEE dans la meme vie publiee, et sur `bfecd02b` il
n y a que **6 lectures posees** d accroupi sur 52. Le film transmet ces composants tres
rarement dans les records que la marche retient — et c est maintenant un fait mesure au lieu d un
chiffre gonfle par les essais.

#### 5.7.4.g LES DEUX ORACLES REPOSES SUR LA POPULATION PROPRE

**SPRINT — UN SEUL MODE, SUR LES DEUX FILMS.** Vitesse au sol des records retenus, casiers de
0,5 m/s :

| | `bfecd02b` (55 044 lectures, 37 vies) | `4f77afc1` (169 981 lectures, 143 vies) |
|---|---|---|
| mediane | **2,263 m/s** | **2,280 m/s** |
| p90 / p99 | 2,878 / 3,510 | 2,879 / 7,109 |
| au-dela de 4 m/s | 41 (**0,07 %**) | 3 508 (**2,06 %**) |
| **modes locaux** | **1** — 2,5-3,0 m/s (17 916) | **1** — 2,5-3,0 m/s (59 011) |
| rapport p95/mediane par vie | 1,04 a **1,47** | 1,01 a **5,22** |

La queue de `4f77afc1` (BTB 24 joueurs) n est PAS une seconde bosse : c est un PLATEAU quasi plat
de 4 a 12,5 m/s (150 a 390 lectures par casier sur 24 casiers). Un sprint ferait une bosse ; un
parc de vehicules fait un plateau. Le negatif du 5.3.5 tient donc sur la population propre, et il
tient mieux : **un seul mode local sur chacun des deux films**.

**SAUT — TOUJOURS PAS DE TAG.** Les quatre tags d `i55`, sur les lectures retenues liees au
bipede : `bfecd02b` 20 lectures (18/1/0/1), `4f77afc1` 47 lectures (24/9/5/9). Contre-preuve :
sur 4 869 puis 16 362 montees franches de vz, **3 puis 5** sont precedees d une lecture d `i55` de
la meme vie dans les deux ticks — **99,9 % et 100,0 % ne le sont pas**. Aucun tag ne predit, et
les denominateurs (9 lectures au mieux par tag) disent pourquoi le contraire ne serait pas
mesurable.

#### 5.7.4.h REVISIONS ET GARDE-RAILS

`grammar.Rev` -> `grammar-2026-09-21.4`, `facts.Rev` -> `killsource-2026-09-21.4`, chroniques
ecrites ; `replay.SchemaVersion` **INCHANGEE a 65** — la FORME ne change pas, le CONTENU oui.
Re-figeage : golden de grammaire, golden des faits, golden des formes de types, 7 fixtures
d entrees + celle de `000d5950`, 7 goldens d assemblage, 8 fixtures de contrat + manifeste.

**DEUX GARDE-RAILS, parce que le defaut garde est une OMISSION** (et une omission ne casse rien :
elle publie davantage, et « davantage » ressemble a « mieux ») :
`etats_mouvement_porte_guard_test.go` epingle (a) que les TROIS neutralisations eteignent la porte
et la restaurent a l identique, cas `nil` compris, et (b) sur la SOURCE, que les trois
localisateurs appellent `neutraliserEtatsDeMouvement()()` — avec, dans le message d echec, le
chiffre que l omission coutait.


### 5.7.5 LES DEUX ORACLES PHYSIQUES, LES SCORES, ET LA CHAINE DE DONNEES — AVEC L ENDROIT EXACT OU ELLE SE PERD

> Deux corrections de l utilisateur, qui font autorite, et qui ont change la methode :
>
> 1. « Le Theater sait quand un joueur se met a courir rien qu en lisant le film ; il ne refait
>    pas le match en live, ca rendrait impossible la lecture d un match ancien ; tout est
>    enregistre dans le film. » Donc « le bit 45 n est ecrit par aucun deserialiseur de `ti=35` »
>    ne peut pas etre la fin de l histoire.
> 2. « Tous les Spartans sautent la meme hauteur (petite variable) et courent a la meme vitesse ;
>    on a la velocite et la position en Z, ca permet de controler quand un saut ou un sprint est
>    entame. » Deux ORACLES PHYSIQUES exacts, donc une verite terrain avant tout candidat.
>
> Tout ce qui suit est mesure sur la population PROPRE du 5.7.4, un film a la fois, aucune base.

#### 5.7.5.a LE SAUT A UNE HAUTEUR, ET ELLE EST LA MEME SUR DEUX FILMS

Un episode aerien commence quand la vitesse verticale TENUE depasse 0,5 m/s et finit quand elle
repasse dessous ; la hauteur montee est l integrale de `vz` sur cette phase. (Pourquoi `vz` et non
le Z des positions : `PositionSample.Vec` n est absolu que sous un accumulateur de monde, qu aucun
balayage n installe — sur le chemin delta c est un delta borne. La verticale de `i1`, elle, a son
unite VALIDEE par l oracle independant du 5.3.5.)

| | `bfecd02b` | `4f77afc1` |
|---|---|---|
| episodes aeriens / vies | 841 / 45 | 4 464 / 182 |
| hauteur mediane · p90 · max | 0,079 · 0,841 · 2,69 m | 0,168 · 0,876 · 7,51 m |
| **H (pic au-dessus de 0,3 m)** | **0,85 m** | **0,85 m** |
| episodes dans le casier du pic | **123** contre 15 et 8 (**x 10,7**) | **653** contre 244 et 94 (**x 3,9**) |
| etiquetes a +/- 10 % de H | **136** (55,5 % des episodes >= 0,3 m) | **830** (47,7 %) |
| **duree mediane de ceux-la** | **0,467 s** | **0,466 s** |
| micro-episodes sous 0,3 m ecartes | 596 | 2 724 |

**L ORACLE DE L UTILISATEUR TIENT EXACTEMENT.** H = 0,85 m et 0,466 s sur deux films, deux cartes,
deux formats de partie (8 joueurs contre 24), avec un pic dix fois plus haut que ses voisins sur
le premier. Le casier [0,0-0,1[ (453 et 1 577 episodes) est le pas de quantification de `vz`
integre sur un tick, pas des sauts : le seuil de 0,3 m qui l ecarte est LU dans la mesure.

**ET C EST UNE CALIBRATION GRATUITE DE L UNITE.** Le 5.3.5 validait `DecodeVelocityMagnitude` par
la DISPERSION de son rapport au deplacement, sans jamais valider son ECHELLE (« aucune unite
supposee »). Une hauteur de saut de 0,85 m, identique sur deux films, est une grandeur ABSOLUE du
jeu : elle dit que les m/s de `i1` sont bien des m/s.

#### 5.7.5.b LE SPRINT A DEUX PLATEAUX REPRODUCTIBLES — ET LA SEPARATION N EST PAS FRANCHE

Segmentation de la vitesse au sol TENUE en plateaux (valeur tenue >= 0,5 s a 5 % pres),
histogramme PONDERE PAR LA DUREE :

| | `bfecd02b` | `4f77afc1` |
|---|---|---|
| plateaux / vies / duree cumulee | 287 / 39 / 245 s | 2 349 / 169 / 2 158 s |
| **Vm** | **2,12 m/s** (37,2 s, 15,2 %) | **2,12 m/s** (257,1 s, 11,9 %) |
| **Vs** | **2,88 m/s** (103,5 s, 42,3 %) | **2,88 m/s** (962,0 s, 44,6 %) |
| **rapport Vs/Vm** | **1,35** | **1,35** |

**Deux bosses, aux MEMES valeurs, sur deux films** — c est exactement ce que « les vitesses sont
constantes par doctrine du jeu » predit, et le rapport 1,35 est l ordre du multiplicateur de
sprint d Infinite.

**MAIS LA SEPARATION N EST PAS CELLE DE DEUX PICS.** Entre 1,75 et 3,00 m/s la duree cumulee monte
de facon continue (165 · 257 · 190 · 505 · 962 s sur `4f77afc1`) : c est UNE masse dominante avec
une epaule, pas deux modes disjoints. **Lequel des deux est le sprint n est donc PAS etabli par la
vitesse seule** — 2,88 pourrait etre la marche avant et 2,12 le deplacement lateral (plus lent
dans Halo), le sprint etant alors le petit relief a 3,25-3,75 m/s (13 s sur 2 158, soit 0,6 %).
Les deux lectures restent ouvertes, et c est dit comme tel.

#### 5.7.5.c LES SCORES : AUCUN CANDIDAT NOMME, ET LE PLAFOND EST STRUCTUREL

Chaque champ replique du bipede est relu A SON `StartBit` sur les records RETENUS (technique
validee au § 5.7.2.d, 0 ecart de largeur sur 75 977 relectures), ce qui lui donne un slot et un
instant. Puis on note : PRECISION = part des instants du candidat qui tombent sur l etiquette,
RAPPEL = part des etiquettes couvertes. Un candidat n est nomme qu au-dela de 90 % dans les DEUX
sens.

Candidats : les 32 bits du mot d `i18` plus sa porte, les etiquettes d `i57`, le tag externe et
l etiquette interne d `i59`, les quatre tags d `i55`, les deux drapeaux d `i54`, l accroupi d
`i29`. **43 candidats notables sur `bfecd02b` (1 886 instants), 54 sur `4f77afc1` (9 310).**

| etiquette | meilleur candidat | precision | rappel |
|---|---|---|---|
| amorce de saut (`bfecd02b`, 136 etiquettes) | `i57.etiquette=-1` | 11,3 % | 25,7 % |
| amorce de saut (`4f77afc1`, 830 etiquettes) | `i59.tag=0` / `i57.etiquette=-1` | 11,9 % | 20,7 % |
| plateau 2,75-2,99 (`bfecd02b`, 111) | `i57.etiquette=-1` | 13,8 % | 39,6 % |
| plateau 2,50-2,74 (`bfecd02b`, 50) | `i59.tag=2` | 1,0 % | 6,0 % |

**AUCUN CANDIDAT NOMME**, sur aucune des six notations, sur aucun des deux films. Les meilleurs
scores (11-14 % de precision) sont environ sept fois le hasard — les 136 fenetres de saut couvrent
1,5 % du film — donc il y a un signal faible, et il est loin d une identite.

**ET LE RECENSEMENT DE DENSITE DIT POURQUOI, ET CE N EST PAS STATISTIQUE.** Sur les 97 345 records
`ti=35` retenus de `bfecd02b`, CINQ composants seulement sont denses :

| composant | part des records retenus |
|---|---|
| `unit-command-tick-component` (`i25`) | **97,14 %** |
| `object-position-dynamic-precision-component` (`i0`) | **85,54 %** |
| `object-translational-velocity-dynamic-precision-component` (`i1`) | **77,55 %** |
| `unit-desired-aiming-vector-component` (`i21`) | **65,29 %** |
| `object-shield-vitality-component` (`i5`) | **36,31 %** |
| `unit-active-camo-state` · `biped-spartan-ability` (`i57`) | 0,75 % · **0,64 %** |
| `biped-mobility-action` (`i54`) · `unit-control` (`i18`) | **0,33 %** · **0,10 %** |
| `unit-crouch` (`i29`) · `biped-posture-physics` (`i55`) | **0,06 %** · **0,05 %** |

**Le record delta du bipede porte cinq choses : le tick de commande, la position, la vitesse, la
visee et le bouclier.** Tout le reste est sous un record sur cent. Un etat connu A CHAQUE INSTANT
ne peut pas vivre dans un champ transmis 0,05 % du temps — le plafond des scores du volet C est
donc structurel, et il etait previsible.

C est la REPRODUCTION INDEPENDANTE de l oracle Rosette du § 2 quinquies (capture live a Cheat
Engine : 15 529 records du masque `{i0, i1, i21, i25}`, longueur vraie 113 bits) — par le
decodeur seul cette fois, et avec le bouclier en plus.

#### 5.7.5.d LA CHAINE DE DONNEES, MAILLON PAR MAILLON

**ACCROUPI — CHAINE COMPLETE, ET ELLE VALIDE LA METHODE.**

```
i29 unit-crouch-component
  deser FUN_142ed42a8 : R(1) -> etat+0x7e8 (booleen) ; R(10) -> etat+0x7ec (fraction)
  applicateur FUN_1406c9b1c (l applicateur d etat replique) appelle
      FUN_140a10970(_, objet, etat, masque) :
          si (masque & 0x20000000)                 <- LE BIT 29 DU MASQUE = L INDEX DU COMPOSANT
              FUN_140c60e1c(objet, *(u8*)(etat + 0x7e8))     le booleen
              FUN_1408b2230(objet, *(u32*)(etat + 0x7ec))    la fraction
  -> objet vivant -> conditions du graphe d animation
```

Cinq maillons, aucun trou : **`i29` EST la source de l accroupissement, pas un echo.** Et le bit
du masque de changement est l INDEX DU COMPOSANT, ce qui donne un moyen general de relier un
applicateur a son composant.

**SPRINT — LA CHAINE SE PERD AU PRODUCTEUR, ET VOICI L ADRESSE.**

Du consommateur vers l amont :

```
transition_conditions_is_sprinting_tlg  (graphe d animation, condition d ETAT)
SpartanAbilityIsSprinting @1436f7170  ->  impl FUN_142a0c70c
    obj = FUN_140477618(&poignee, 1)
    return (*(u64*)(obj + 0x8b8) >> 0x2d) & 1          <- LE BIT 45 DU MOT DE DRAPEAUX
14 sites de TEST du bit 45 releves (FUN_140775a24, FUN_140776c4c, FUN_1407754cc, FUN_1407fa018,
  FUN_1407fba24, FUN_1407fcb30, FUN_140800ea8, FUN_1406de83c, FUN_14060f81c, FUN_140611238,
  FUN_1406735e0, FUN_1406dba04, FUN_140897b6c, FUN_14089dd7c)
??? QUI POSE LE BIT 45 : NON TROUVE
```

Quatre mecanismes cherches, chacun avec sa mesure :

1. **ecriture partielle a l octet** : `[x+0x8bd]` porte le bit 45 — **0 reference dans l image**
   (les octets voisins en ont : `0x8bc` 89, `0x8be` 42).
2. **`BTS`/`BTR` a rang immediat** : **26** sur ce mot (14 `BTS`, 12 `BTR`), aux rangs 8, 9, 0xb,
   0xd, 0xe, 0xf, 0x11, 0x12, 0x16, 0x18, 0x1a, 0x1d, 0x1e — **aucun a 0x2d**.
3. **masque calcule** : aucun des **17** assignateurs du mot ENTIER ne charge `1 << 45`
   (`0x200000000000`) ; le masque n apparait qu en LECTURE (`TEST`), et aucun `BTS`/`BTC` a rang
   REGISTRE ni `SHLX` ne porte sur cet offset.
4. **copie de structure** : les SEULES copies de 16 octets a l offset `0x8b8` sont
   `FUN_141576070` et `FUN_1415761e0` — et ce sont les constructeurs de copie du **widget
   d interface `TwoToneMeterQuad`** (objet de 0x918 octets, vtable `PTR_FUN_143849e88`,
   enregistre par `FUN_1400eed50` avec la chaine `fui_meter_two_tone_quad_widget`). **Collision
   d offset entre deux classes, pas une copie d unite.** Piste fermee, et son adresse est ecrite.

Et le maillon qui aurait porte la reponse est controle : **aucun des NEUF applicateurs de la
chaine d etat replique ne touche `obj+0x8b8`** — `FUN_140a10970`, `FUN_140c85028`,
`FUN_1406c72e8`, `FUN_140a10c54`, `FUN_140a10b64`, `FUN_140a10a7c`, `FUN_1404d4c28`,
`FUN_140a10998`, `FUN_1406ca5f0` : zero reference sur les neuf. Le seul contact de
`FUN_1406c9b1c` lui-meme est le **bit 54**, et sa source est un champ de l objet VIVANT
(`FUN_140719698(obj) + 0x120` / `+0x121`), pas du tampon replique.

**NON TROUVE, ET LA VOIE NON EXPLOREE EST NOMMEE** : l un des 17 assignateurs du mot entier
(`FUN_1406730c4`, `FUN_1406c7ad4`, `FUN_1407184ac`, `FUN_140775a24`, `FUN_140776790`,
`FUN_140803c54`, `FUN_140805064`, `FUN_1408dcd7c`, `FUN_140970614`, `FUN_1409aac4c`,
`FUN_1409ab28c`, `FUN_140a19150`, `FUN_140a1e2e4`, `FUN_140adfa5c`, `FUN_1405659f0`,
`FUN_140b39604`, `FUN_1406c9b1c`), depuis un registre dont la provenance n a pas ete remontee.
Un maillon par lecture : c est le programme du lot suivant, et il commence par `FUN_1409aac4c`
(qui pose le mot ET teste le bit 0x400) et `FUN_140775a24` (qui pose le mot ET teste le bit 45).

**AERIEN — MEME POINT D ARRET.** `c_biped_airborne_state` (`143e2bfc0`) est l une des trois
classes d etat de bipede, et sa table de champs reflechie (`FUN_1432226c0` : 6 booleens,
8 flottants, 2 vec3, 2 shorts sur 0x6c octets) est celle d une structure LOCALE. Aucun des neuf
applicateurs ne l alimente depuis un champ replique.

#### 5.7.5.e CE QUE TOUT CELA DIT, ET CE QU IL NE DIT PAS

**L UTILISATEUR A RAISON SUR LE FOND, ET LA MESURE LE MONTRE PLUTOT QUE DE LE CONTREDIRE.** Ce que
le film transmet par instant, ce sont cinq choses : le tick de commande, la position, la vitesse,
la visee, le bouclier. Et le SAUT s y lit avec une nettete qu on peut chiffrer — un pic a
**0,85 m** dix fois plus haut que ses voisins, de duree **0,466 s**, identique sur deux films. Le
Theater n a pas besoin d un bit de saut : il a la vitesse. C est aussi ce que dit le graphe
d animation, dont les transitions sont gardees par des CONDITIONS D ETAT (`is_sprinting`,
`is_airborne`) et non par des champs.

**CE QUI N EST PAS ETABLI, ET QUI N EST PAS PUBLIE.** Que le bit 45 soit CALCULE a la relecture
plutot que transmis reste une hypothese : son producteur n a pas ete trouve, et « non trouve a ces
adresses » n est pas « n existe pas ». Le sprint n a donc ni source repliquee nommee ni signature
de vitesse franche. Et le saut, bien qu il ait une signature physique excellente, se lit par un
SEUIL sur une integrale — l etiqueter dans le document publierait un seuil comme une donnee, ce
que la doctrine du chantier interdit. **Rien n est porte.**

**CE QU IL FAUDRAIT POUR PORTER LE SAUT**, ecrit pour que la decision soit possible : (1) remonter
le producteur du bit 45 ou de l etat aerien (un maillon par lecture, liste ci-dessus), ce qui
donnerait une source et non un seuil ; OU (2) si l on accepte la derivation, un accord mesure
entre l etiquette physique et un signal du jeu — par exemple les medailles ou les evenements que
le film porte deja — pour que le seuil soit VALIDE contre autre chose que lui-meme. Aucune des
deux n est faite.

### 5.7.6 CE QUI N EST PAS PUBLIE, ET POURQUOI

**AUCUNE MONTEE DE SCHEMA. `stances[].kind` NE GAGNE NI `sprint` NI `jump` NI `clamber`.**

- `sprint` : REFUTE chez l ecrivain (§ 5.7.1.a) et par la vitesse (5.3.5). Le publier serait
  publier un seuil comme une donnee.
- `jump` : NON TRANCHE, avec ses chiffres (§ 5.7.2.d). Le seul candidat vivant est l une des
  quatre voies de l union d `i55`, et la population ne permet pas de dire laquelle.
- `clamber` : l escalade n est pas nommee. `i54` reste « Action » / « Action », comme au 5.3.6.

Web inchange : aucun libelle ajoute, aucun zod touche, aucune OpenAPI.

## 3. LE NEGATIF, MESURE DEUX FOIS

**(a) Sur l'archetype.** Les 64 composants de `ti=35` (`ecs_table.tsv`) : aucun nom ne contient
« sprint », « jump », « clamber », « vault » ni « airborne ». Les seuls noms de geste sont
`i29 unit-crouch`, `i54 biped-mobility-action`, `i55 biped-posture-physics`, `i62 biped-slide`.

**(b) Sur l'image entiere.** Balayage du pool COMPLET des chaines des sections de donnees
(`reapparition.ChainesContenant`, instrument de ce lot) :

| mot | chaines dans l'image | dont noms de composant |
|---|---|---|
| sprint | 71 | **0** |
| jump | 94 | **0** |
| crouch | 66 | 1 (`unit-crouch-component`) |
| slide | 60 | 1 (`biped-slide-component`) |
| clamber | 34 | **0** |
| mantle | 0 | 0 |
| vault | 14 | **0** |
| thrust | 37 | **0** |
| posture | 14 | 1 (`biped-posture-physics-component`) |
| airborne | 68 | **0** |
| mobility | 10 | 1 (`biped-mobility-action-component`) |

Le moteur PARLE de sprint et de saut — `SpartanAbilityIsSprinting`,
`SpartanAbilityGetSprintFraction`, `SpartanAbilityIsClambering`, `IsAirborne`,
`Unit_IsAirborne`, `HK_CHARACTER_JUMPING`, `CharacterPhysicsModeClambering`, et une table
d'actions d'entree (`143d03c40`) qui aligne `Sprint`, `Thruster`, `Clamber`, `Slide` — **mais
aucun de ces mots ne nomme un composant replique**. Le sprint et le saut ne sont pas ecrits
sous leur nom dans le film ; s'ils y sont, ils y sont sous une autre forme (identifiant
d'action de `i54`, tag de `i55`, action de `i63`, ou derivation de la vitesse `i1`).

> Reserve ecrite : le balayage ne voit que l'ASCII a un octet. Un mot present uniquement en
> UTF-16 y echapperait. Le comptage « stance » (963) est domine par `instance` — sous-chaine,
> pas mot ; il n'est pas utilise comme preuve.

---

## 4. CE QUE LA PREUVE SUR FILM DOIT TRANCHER (point 2, en attente de voie libre)

1. `i29` : distribution du booleen et de la fraction ; les intervalles accroupis tiennent-ils ?
2. `i62` : le booleen de glissade donne-t-il des intervalles courts (< 2 s) coherents avec une
   vitesse elevee decroissante ?
3. `i54` : combien d'evenements par joueur et par match, et **LA VENTILATION DE L'ENUM** —
   `+0x9c` (2 bits) et `+0x08` (10 bits) croises avec la vitesse au sol d'`i1`, l'etat de
   glissade d'`i62` et la variation d'altitude, pour NOMMER les classes (sprint / escalade /
   glissade / propulseur). Voir § 2.8 : c'est la voie la moins chere pour nommer l'enum.
4. `i55` : le compte et la distribution du tag sont FAITS sur les mini-bobines (§ 2.4 bis) ; ce
   qui reste est le chemin DELTA — `i55` y est-il declare, avec quels tags, et la marche y
   ferme-t-elle apres l'avoir franchi ?
5. `i1` : la composante verticale signe-t-elle le saut (vz > 0 puis < 0) ? La vitesse au sol
   separe-t-elle sprint et marche par un seuil net ?
6. **`i18 unit-control +0x544`** : ventiler les **32 bits un par un** contre la
   composante verticale d'`i1`, l'etat d'`i29` (accroupi), celui d'`i62` (glissade) et les
   initiations d'`i54`. **Un bit de saut se signerait par lui-meme** : il s'allume une
   image AVANT que `vz` ne devienne positif. Le champ est deja decode et jete
   (`consumeOpt32`) : la sonde ne coute aucun octet de grammaire.

---

## 5. INSTRUMENTS, REJOUABLES

```bash
export GOCACHE=.../.gocache PATH=/c/msys64/ucrt64/bin:$PATH CGO_ENABLED=1
cd apps/go-api
MOUV_EXE="D:/SteamLibrary/steamapps/common/Halo Infinite/HaloInfinite.exe" \
  go run -tags=research ./internal/games/halo_infinite/film/research/cmd_mouvement
MOUV_EXE="..." go run -tags=research ./internal/games/halo_infinite/film/research/cmd_mouvement \
  -vocabulaire -plafond=25
```

```bash
# La mesure de D1, sur les sept mini-bobines — aucun film du cache, aucune base.
go test -tags=research -count=1 -v -run TestMouvementI55D1 \
  ./internal/games/halo_infinite/film/internal/grammar/
```

- `film/research/mouvement/` — cibles du lot, vocabulaire, rapport.
- `grammar/mouvement_i55_d1_research_test.go` — la mesure de D1 (`_test.go`, donc HORS de
  l'empreinte de la couche grammaire : `grammar.Rev` ne bouge pas).
- `film/research/reapparition/` — deux ajouts SEULEMENT : `Executable.ChainesContenant`
  (le pool complet des chaines, que l'en-tete de `univers.go` appelait deja) et l'export de
  `Calibrer` (la garde de publication, reutilisee au lieu d'etre recopiee).

---

## 6. DECOUVERTES HORS PERIMETRE (consignees, NON traitees)

- **D1 (5.3)** — `consumeBipedPosturePhysics` (`ti=35 i55`) saute les quatre charges du tag de
  2 bits, que le jeu lit toutes. **MESUREE sur les sept mini-bobines (§ 2.4 bis) : le tag est
  NON NUL 210 fois sur 895 franchissements (23,5 %)**, et aucun des quatre tags ne coute zero
  bit chez le jeu. La question « pourquoi la marche reste-t-elle alignee ? » ne se pose pas sur
  ces bobines — elle ne l'est pas : 6 fermetures sur 1 364 records, **0** parmi ceux qui
  franchissent `i55`. RESTE OUVERT pour le chemin DELTA, mesure de 5.3.2.
- **D2 (5.3)** — les commentaires de portage de `i54` et `i62` donnent comme « descripteur »
  l'adresse du SLOT de nom (`descripteur + 0x18`) : `143d0c9d8` au lieu de `143d0c9c0`,
  `143d0ca80` au lieu de `143d0ca68`. Aucune grammaire n'est fausse ; c'est la meme
  incoherence de convention que la decouverte de bord du lot 3.7 (§ 7.3).
- **D3 (5.3)** — `MobilityActionHook` (`observateur.go`) n'a toujours aucun consommateur. **Il prend son sens
  au § 2.7** : c'est le seul point d'ecoute existant de l'action de mobilite, et le canal
  d'evenements qui porterait la meme chose est mesure VIDE.
- **D4 (5.3)** — **LES DEUX TABLES DE TYPES D'EVENEMENT DU DEPOT NE SONT PAS INDEXEES PAREIL, ET
  C'ETAIT UNE QUESTION OUVERTE.** `event_types_catalogue_test.go` (types 50..127, base
  `0x144724A90`) porte son propre avertissement : « valable SEULEMENT si les deux espaces
  d'index coincident (non etabli) ». **Ils ne coincident pas, et l'arithmetique de la colonne
  « Objet » de l'annexe A le montre sans Ghidra** : l'objet du type de fil 43 est `0x144724d18`,
  soit `0x144724A90 + 81*8` — donc le SLOT 81 ; celui du type 78 est `0x144724d50`, soit le
  SLOT 88. Or `eventTypeNames[81]` vaut bien `initiate_mobility_action` et `eventTypeNames[88]`
  vaut `ai_jump`. **Le piege est reel** : nommer un type de FIL avec cette table rend
  « MusicTrigger » pour ce qui est `ai_jump`. Les instruments de recherche qui comptent
  (`lot1_tirs`, `r7_grammaire`, `r7_charges_lot5`) utilisent la bonne table, celle du
  registrar. **NON TRAITEE** : un lot d'outillage renommerait la carte en `eventSlotNames` et
  poserait le garde-rail.
- **D5 (5.3)** — **LE CATALOGUE D'AOUT RANGE `i56` SOUS « SPRINT », ET C'EST FAUX** (§ 2.6) :
  `biped-spartan-ability-energy` est la jauge des trois emplacements de charge de la capacite
  d'armure (masque `R(3)` + `R(7)` par charge armee, defaut `0x7F`), donc de l'EQUIPEMENT. Deux
  documents le propagent (`RECAP_STATS_EXPLOITABLES.md:229`,
  `HANDOFF_FILM_EXTRACTION_EXTERNAL_DEV.md:595`). **NON TRAITEE** : les corriger demande de
  toucher deux documents hors perimetre de ce lot ; la presente note fait foi en attendant.
- **D6 (5.3)** — **`ecs_table.tsv` DONNE A `i49 biped-control-context` « 3 bits (5 valeurs) » ;
  L'ECRIVAIN EN LIT 2 OU 4.** `FUN_14107166c` calcule sa largeur depuis `DAT_145121140` (le
  reglage de pleine precision du processus) : `w = 4` s'il vaut 1, `w = 2` sinon, puis `R(1)`.
  Le port du depot (`consumeBipedControlContext`) est juste ; c'est la TABLE qui ment. **NON
  TRAITEE** (regle 7) : une ligne de table, a corriger par le lot qui reprendra `ecs_table.tsv`
  avec son garde-rail (voir D2).
- **D7 (5.3)** — **UN MOT DE 32 BITS OPTIONNEL, DEJA DECODE ET JETE, N'A AUCUN SENS ETABLI.**
  `i18 unit-control +0x544` (§ 2.9.1) est le seul champ de tout l'archetype bipede ayant la forme
  d'un champ de bits de commande. `consumeUnitControl` le consomme par `consumeOpt32` et
  l'abandonne. **NON TRAITEE ICI, mais c'est un item de 5.3.2** : sa ventilation bit a bit est la
  mesure la moins chere du lot, et elle tranche la question du saut.
  **REFUTEE LE 2026-09-21 (§ 2ter.4)** : les 32 bits sont TOUS allumes, dans une fourchette
  etroite (7,2 a 20,6 % sur `4f77afc1`), et aucun ne predit la montee mieux que ses voisins. Un
  champ de boutons aurait des bits MORTS et un ou deux bits dominants. C'est un mot opaque a
  forte entropie. **Il ne reste aucun candidat de forme « entree » sur le bipede.**
- **D8 (5.3)** — **LE BALAYAGE BIPEDE DE PRODUCTION NE LIT QUE SIX COMPOSANTS, ET S'ARRETE AU
  PREMIER AUTRE.** `scanRecordDirs` (`offline_aim.go`) modelise `i1`, `i2`, `i3`, `i4`, `i5` et
  `i21`, puis rend la main (« composant non modelise -> curseur non fiable »). Tout ce qui est
  au-dela d'`i21` — donc TOUS les composants de mouvement — n'est JAMAIS lu sur le chemin delta
  en production. Ce n'est pas un defaut : ce balayage ne cherche que les positions et les
  directions. Mais cela veut dire qu'un port qui voudrait publier l'action de mobilite devra
  faire marcher la boucle COMPLETE sur les records delta, ce que 5.3.2 a prouve faisable
  (100 % de marches completes sur 541 105 records). **NON TRAITEE** : c'est le premier item de
  chiffrage de 5.3.3.
- **D9 (5.3)** — **`i54` PORTE DEUX CHAMPS DONT LE DOMAINE MESURE CONTREDIT L'HYPOTHESE DE
  L'ECRIVAIN.** L'identifiant de 10 bits n'est transmis **0 fois sur 2 245 initiations**, et
  `+0x9c` ne prend que **deux** valeurs (0 et 2), jamais 1 ni 3. L'hypothese « enumere a quatre
  actions » du § 2.8 est refutee par les valeurs ; seul `+0x98` se comporte en discriminant, et
  son domaine varie d'un film a l'autre (3 valeurs contre 8). **NON TRAITEE** : nommer les
  classes demande de croiser `+0x98` avec la carte et le geste, ce qui est un lot en soi.

- **D10 (5.3) — `marchViews = 8` CONTRE TROIS VUES CHEZ LE FRAME-PROCESSEUR.** `FUN_142987460`
  deroule exactement trois boucles ; le depot en deroule huit. Mesure : 8 vues rapportent 7 000
  records de plus mais degradent l etalon `i25` de 0,8 point et rendent 1 251 records `ti=35` de
  MOINS ; 16 vues est indiscernable de 8. **NON TRAITEE** : reduire la constante est un lot de
  production (empreintes gelees de `killsource` et de la marche des morts d objet).
- **D11 (5.3) — 24 % DES PAQUETS A LISTE PLEINE NE SE LOCALISENT PAS.** `marchLocateStrict` en
  localise 76 a 79 % ; le complement demande la grammaire de charge des types d evenement.
  **NON TRAITEE** — c est le lot d evenements deja consigne.
- **D12 (5.3) — LA TABLE D ENTITES EST CELLE DE LA VUE.** La garde du delta lit
  `vue + 0x38 + slot*0xa0` ; le port tient UN monde pour les trois vues. **NON TRAITEE** :
  approximation assumee, a lever seulement apres avoir mesure si un meme slot porte deux
  archetypes selon la vue.
- **D13 (5.3) — `decodeInferLoop` NE POSE PAS LE SLOT DE CAPTURE.** Une valeur publiee depuis un
  record NEW herite du slot du record precedent, ou de zero au premier record du paquet. Sans
  filtre, le slot 0 portait 3 035 des 7 947 lectures d `i29` de `bfecd02b`. **NON TRAITEE** :
  c est un lot de PRODUCTION, le meme chemin attribuant les echantillons de position.
- **D14 (5.3) — L ORACLE DE VITESSE EST REFUTE, ET AVEC LUI LA MESURE DU SPRINT.** Un bipede sur
  deux a plus de 11 m/s et le maximum est 347 m/s. **NON TRAITEE**, et le SPRINT reste NON
  TRANCHE pour une raison mesuree ; le remede est un oracle verifie contre les positions
  successives d une meme vie.
- **D15 (5.3) — L ETAPE `positions` DE L EQUIVALENCE A DERIVE SUR `bcb6d393` AVANT CE LOT.** Meme
  sha avec la bascule de 5.3.3-a abaissee : l ecart vient de la base `5fd6f02c3`. **NON TRAITEE**
  — le re-figeage est un geste du pilote.
- **PORTER LES QUATRE ETIQUETTES ECRITES D `i59` (1, 4, 5, 6) ET LES DEUX SANS CHARGE (bruts 6
  et 7).** Pre-requis dans l ordre au § 2quindecies.5. **NON TRAITEE** : 0,12 % des records de
  bipede, contre un risque sur `grappleLines[]`.

- **D16 (5.3) — LE SPRINT EST REFUTE COMME OBSERVABLE PAR LA VITESSE** (5.3.5). Loi de `i1`
  exacte (ecrivain + constantes relues), oracle du deplacement valide sur deux films (dispersion
  1,7 et 2,3 ; facteur d unite 0,240 et 0,236), et la distribution au sol n a **qu un seul mode,
  a 2-3 m/s** (61,3 % et 56,1 %) : au-dela de 4 m/s il reste 0,08 % de la population.
  **NON TRAITEE** : si le sprint est dans le film, c est un ETAT (un drapeau), pas un second
  regime de vitesse. A chercher dans les composants d etat, pas dans une grandeur.
- **D17 (5.3) — LE SAUT EST LU MAIS PAS PROUVE** (5.3.5). La composante verticale est decodee
  juste ; sa segmentation en episodes repose sur deux seuils d instrument, et la signature ne
  tient pas d un film a l autre (duree mediane des impulsions a pic >= 3 m/s : 0,632 s sur
  `bfecd02b`, 1,567 s sur `4f77afc1`, p90 48 s). **NON TRAITEE** : publier cet etat publierait un
  seuil comme une donnee. Le remede est une PRECONDITION lue (etre au sol), pas un seuil ajuste.
- **D18 (5.3) — LES INSTRUMENTS DU LOT N INSTALLAIENT PAS LES LARGEURS D AXE DE LA CARTE**
  (5.3.5). `NewFilmContextForMap` pose la bascule et le decoupage impose, mais le descripteur
  world-object est installe SEPAREMENT en production (`replay.installWorldObjectPrecision`).
  Sans ce second geste, `i0` lit ses axes aux largeurs de `cliffhanger` sur toutes les autres
  cartes. Effet mesure sur `bfecd02b` : records `ti=35` 31 530 -> **97 447**, desyncs 38 -> **3**,
  etalon `i0` 63,3 -> **85,5 %**, lectures `i1` x18,5. **TRAITEE DANS LES INSTRUMENTS** (les
  trois posent desormais le descripteur) ; **consignee parce qu elle invalide les chiffres des
  § 2quaterdecies a § 2sexdecies**, corriges au § 2septdecies. Un garde-rail d instrument
  manquerait encore : rien n empeche le prochain d oublier ce geste.
