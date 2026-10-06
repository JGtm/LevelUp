# LOT VA, ÉTAPE V2 — La fin de la vue A fixe le début de la vue B (2026-10-05)

> Campagne de grammaire, lot VA. Décisions de l'utilisateur du 2026-10-04 : « Lire la vue A jusqu'au
> bout », puis « Oui, lance » sur les règles (1) à (3) ci-dessous. Plan de conception :
> `scratchpad/va-conception/PLAN_LOT_VUE_A.md` (non versionné) et sa vérification adverse ; étape V1 :
> `LOT_VA_V1.md` (`e6ec7abd4`, `3bacfadeb`).
>
> Corrections du contrôle indépendant de `8354c0d43` (§14) intégrées le 2026-10-05.
>
> Worktree `LevelUp-wt-cg3-vuea`, branche `feat/cg3-vue-a`, tête de départ `3bacfadeb` (V1), base du
> lot `87cdfa761` (= `origin/feat/v75`, contrôlé par `git fetch` au début de l'étape et avant les
> gates : pas d'avance). Référence de mesure : binaires de `87cdfa761` (ceux de V1, dont la sortie est
> identique à l'octet à `87cdfa761`).
>
> Conventions : **lu** = lu dans le jeu (Ghidra, `HaloInfinite.exe` HI_1_13_0, lecture seule) ;
> **mesuré** = compté par un outil sur les films ; **estimé** = dérivé d'une mesure par une hypothèse
> écrite. « Sain » = paquet fermé que le juge de L0 ne contredit pas (colonne `ferme` de la carte v2).

## 0. Statut

**[x] retenue après les corrections du contrôle indépendant (§14), sous une réserve à lever par le
pilote : le gate de corpus rend rc = 1, instruit (3 `FAUX` de V-3, `[FILET]`), sans `MANQUE`.**
La fin de la vue A lue (E) fixe désormais le début de la vue B aux trois sites du localisateur
(cuisson, marche des morts d'objet, marche de `killsource`), selon les décisions (1) à (3). La loi du
bit nul devant la tête de la vue B, que la première version appliquait aussi aux candidats de tête,
est retirée : hors plan, non couverte par une décision (§14, correction 2).

- Carte v2, 20 films, contre `87cdfa761` : **313 542 → 355 198 paquets sains (+41 656)**, records
  utiles sains **2 923 597 → 3 474 013 (+550 416)** ; D1 corpus 44,4 %, HI_1_13_0 91,2 %. **354 sains
  perdus en brut, tous de classe E** (la marche depuis E bute sur un composant non porté ou en vue C —
  liste par composant au §5.1), instruits paquet par paquet.
- Gate 2 : **tenu sans exception** — 16 films en hausse, 4 films illisibles inchangés à l'unité, aucun
  film en baisse ; aucune exception D2.
- `killsource` : **aucune mort, valeur ni voie ne change** (20 films) : `killsource.Rev` reste
  `killsource-2026-09-27`, aucun backlog du parc à demander.
- Marche des morts d'objet : étape `vehicles` publiée mesurée aussi sur les deux films ÉGALE à
  véhicules (§6.1) — `4f77afc1` : 0 mort retirée, 5 lectures d'occupation retirées, instruites comme
  pertes de classe E ; `bfecd02b` : aucune retirée.
- Gate de corpus : **rc = 1** (avant et après les fusions de `feat/v75` `28c542b33` et `65c99b669`, §7.1 et §7.2) — 16 / 19 sans `FAUX` (4 « ok », 12 `PERTE`), 0 `MANQUE`, 3 `FAUX` de V-3 (instruits : tous les tirs
  « hors vie » sont des tirs de véhicule couverts par un trajet du même slot dans ce véhicule ; le banc
  ne compte pas les trajets comme des vies), `[FILET]` des postures (fausse continuité retirée, oracle
  physique). Son admission (rc 1) est un geste du pilote ; la revue adversariale de fin de lot reste à
  faire.
- Gates de code (rejoués après chaque fusion, dernière : `30d94a60b`, §7.2) : gofmt vide, vet (et `-tags=research`) rc=0, archlint ok, G-film rc=0 (21 paquets),
  golangci-lint 0 issue ; **13 / 13 mutations ROUGES** (dont la neuve, m14, juge PRÉFIXE) ;
  vérifications (a) à (g) de la représentation intermédiaire tenues.
- Révisions : `grammar-2026-10-06.3` (`.2` pour la première version) ; `killsource`, `objectives`,
  `profile`, `source`, `SchemaVersion` constants.


## 1. Ce qui est lu dans le jeu

| Fait | Lu où | Porté dans |
|---|---|---|
| Le tick s'écrit `cfg:1 · vue A · 0 · vue B · vue C`, bout à bout : l'écrivain 0 écrit la vue A (`142f2c578 CALL FUN_142f2c050`), puis UN bit de valeur 0 (`142f2c57d XOR R8D,R8D`, `142f2c583 CALL FUN_1406d49c4` : `*(w+0x30) = *(w+0x30)*2 \| param_3`) ; la vue B est écrite dans un tampon à part (`142f2c56e CALL FUN_142f2cc78`) ; `FUN_14299d2c8` écrit le bit de configuration (`14299d333`, `14299d33e`) puis recopie les tampons au bit près (`FUN_1406d5d14`, boucle `14299d34c..14299d395`) | `va_ghidra/ASM_142f2c3b0.txt`, `ASM_14299d2c8.txt`, `FUN_1406d49c4.c` (versionnés à cette étape ; relus de la recherche LS) | en-tête de `localisateur.go` |
| Le lecteur (`FUN_142987460`) lit le bit de configuration, puis la vue A (`FUN_14076a1c4` : `{R(1) ; 0 → fin ; FUN_14080a9d4}`), puis la vue B au bit qui suit. Aucune recherche, aucune signature | `FUN_142987460`, `FUN_14076a1c4` (V1, `va_ghidra/FUN_14076a1c4.c`) | `debutParLaVueA` |
| Conséquence : le début de la vue B est le bit qui suit le terminateur de la vue A, et le bit qui précède la TÊTE de la vue B vaut 0. Devant tout autre record, le bit précédent est le dernier bit du record d'avant (records contigus, `FUN_14076b9c8`) | les deux lignes ci-dessus | `debutParLaVueA`, `precedeDuTerminateur` |
| Version de chaque genre de message : celle du film en rejeu (`film + 0xCB208`), native `DAT_14474cd90` ; règle des deux classes (ÉGALE / PRÉFIXE), décision du 2026-10-04 | `FUN_141102ed0`, `FUN_1428e1c64` (V1) | `vue_a_versions.go` (V1, inchangé) |

**Non lu, et non utilisé** : rien de neuf. Le slot 123 et le composant unique de la signature restent
des MESURES (convention de production, inchangée là où la fin de la vue A ne décide pas).

## 2. La règle de production

Pour un paquet delta dont la tête annonce une liste d'événements, la vue A déjà lue par la lecture
unique (`rangerLaTete`, V1) décide (`debutParLaVueA`, `localisateur.go`) :

| Classe de la table du film | Vue A lue jusqu'au terminateur, ≥ 1 message | Sinon |
|---|---|---|
| ÉGALE (HI_1_12_0, HI_1_13_0) | **E**, toujours (`lecture.DebutParVueA`) — même quand la signature du slot 123 trouverait une autre position, même quand la marche depuis E bute ensuite sur un composant non porté : la loi du jeu prime, revenir à la signature serait une convention (décision (1)) | chemin d'avant, à l'identique |
| PRÉFIXE (HI_1_9_0 à HI_1_11_0) | **E** seulement si la marche de la vue B depuis E ferme le paquet sans règle de l'écrivain contredite (`lectureDEssai(…).Fermee`, juge de L0, monde restauré) ; sinon chemin d'avant (décision (2)) | chemin d'avant |
| ILLISIBLE | — | chemin d'avant |

Une vue A lue en partie (genre non porté, charge refusée, bit de configuration à 0, payload épuisé)
n'est JAMAIS utilisée (décision (3)) : `debutParLaVueA` exige `Porte`.

**Les trois sites du localisateur** suivent la même règle :

- cuisson : `debutDeLaVueBDeCuisson` (`marche_trames.go`) reçoit la vue A rangée — pas de seconde
  lecture — puis `localiserLaListe` à défaut ;
- marche des morts d'objet (`marchDebut`, `object_deaths_march.go`) et marche de `killsource`
  (`walk.go`) : `DebutDeLaVueB` (`localisateur.go`) lit la vue A par la lecture unique (troisième
  appelant permis de `lireLaVueA`, garde-rail archlint mis à jour) sous la grammaire de vue A du film
  (`VueADuFilm` : `fc.grammaireDeLaVueA()` pour la marche des morts, `VueADuFilmSousCarte(f.src,
  carte)` pour `killsource`, portée par sa calibration), puis `LocaliserBoucleDeRecords` dans l'ordre
  `SignaturePuisLargeurLibre` à défaut.
- la calibration du cadre de la marche des morts (`trialFrameConfig` → `marchStartOf`) juge chaque
  largeur d'identifiant par le taux de paquets que la SIGNATURE localise : la signature dépend de
  cette largeur, la fin de la vue A n'en dépend pas et localiserait les mêmes paquets sous toutes les
  largeurs. Elle garde le localisateur seul (`VueADuFilm{}`), écrit dans la doc de `marchStartOf`.

**La loi du bit nul devant la tête (D-LSR-3, lue) n'est PAS appliquée aux candidats de tête**
(correction 2 du contrôle, §14). La première version de l'étape l'appliquait aux candidats NEW de tête
de la cuisson (`debutParChaine`, `debutParFermetureRangee`) : c'était une règle neuve du
localisateur, que le plan classait hors lot (`PLAN_LOT_VUE_A.md`, D-VA-6) et qu'aucune des
décisions (1) à (3) ne couvre. Elle est retirée ; les candidats sont ceux d'avant le lot, à
l'identique. Reste de cette version : le test du bit qui précède une position n'a qu'une implantation
(`precedeDuTerminateur`, `localisateur.go`), que les deux étages du localisateur (signature stricte,
repli à largeur libre) appellent, sortie identique ; le garde-rail
`archlint/film_localisateur_unique_test.go` interdit le test brut `BitAt(_, x-1)` dans `grammar` hors
de lui (une troisième copie ne revient pas).

## 3. Ce qui change dans le code

Production (`film/internal/`) :

- `grammar/localisateur.go` : en-tête réécrit (la fin de la vue A d'abord, la table des trois sites,
  la règle des classes) ; `debutParLaVueA` (la décision) ; `VueADuFilm` et `VueADuFilmSousCarte` (la
  grammaire de vue A du film pour les marches qui n'ouvrent pas de contexte) ; `DebutDeLaVueB` (les deux
  marches qui lisent les morts) ; `precedeDuTerminateur` (seule implantation du test du bit qui précède
  une position ; `marchLocateStrict` et `marchLocateFallback` l'appellent, sortie identique).
  `debutParLaVueA` n'exige qu'une vue A portée (correction 4, §14 : la condition « au moins un
  message » était inatteignable ; la raison est écrite à la fonction).
- `grammar/marche_trames.go` : `marcherLePaquet` prend la grammaire de vue A du contexte et passe par
  `debutDeLaVueBDeCuisson` (la fin de la vue A rangée, sinon `localiserLaListe`) — la fonction vit
  ici parce que le pilotage de la marche des trames ne vit que dans ce fichier (garde-rail
  `marche_trames_unique_test.go`) ; en-tête mis à jour.
- `grammar/debut_de_liste.go` : doc de `localiserLaListe` seulement (la liste dont la vue A n'a pas
  décidé). Les candidats de tête sont ceux d'avant le lot (la loi du bit nul retirée, correction 2,
  §14).
- `grammar/object_deaths_march.go`, `grammar/object_deaths.go` : `marchDebut` reçoit la grammaire de
  vue A (`DebutDeLaVueB`) ; `ScanMarchFacts` passe celle du film ; `marchStartOf` (calibration)
  garde le localisateur seul, la raison écrite ; en-tête (mécanisme 2).
- `facts/killsource/calibrate.go`, `walk.go`, `decode.go` : la calibration porte `VueA`
  (`VueADuFilmSousCarte(f.src, carte)`), `runWalk` reçoit la calibration (5 paramètres) et part de
  `DebutDeLaVueB` ; en-tête de `walk.go`. Chronique : complément à révision constante.
- `grammar/frame_closure_detail.go` : `PaquetDeCarte.ListeLue` ; `ListeLocalisee` exclut
  `DebutParVueA` (vérification (c)).
- `grammar/lecture/paquet.go` : `DebutParVueA` EN QUEUE de `DebutDeVueB` (valeurs stables), doc : un
  début LU n'est pas une récupération.
- `grammar/vue_a_lecture.go`, `vue_a_versions.go`, `marche_trames_rangs.go` : docs (le troisième
  appelant de la lecture unique ; la décision par classe ; la marche par rangs part de la fin de la vue
  A lue ou d'un début localisé).
- `grammar/rev.go`, `rev_chronique.go`, `testdata/grammar_rev.golden` : `grammar-2026-10-06.2`, puis
  `grammar-2026-10-06.3` (corrections du contrôle, §10).

Instruments et garde-rails :

- `archlint/film_localisateur_unique_test.go` : l'essai de position reconnaît `precedeDuTerminateur(_, s)` ;
  le test brut `BitAt(_, x-1)` est interdit dans `grammar` hors de `precedeDuTerminateur`, que les deux
  étages du localisateur appellent ; vecteurs `TestGardeRailBitDeTeteVecteurs`.
- `archlint/film_vue_a_lecteur_unique_test.go` : `DebutDeLaVueB` est le troisième appelant permis de
  `lireLaVueA`.
- `research/cmd_fermeture` (carte v2) : la colonne `liste` vaut `lue` pour `DebutParVueA` ; compte
  « listes d evenements / lues (fin de la vue A) » dans `fermeture_ecrivain.tsv`.
- Recopies du pilotage (gate 5) : `cmMarcher` et `rnMarcher` partent par défaut de
  `debutDeLaListeSous` (= `debutDeLaVueBDeCuisson` sous la grammaire du film) ; la branche « sans
  localisateur fourni » de `cmPaquetDe`, devenue inatteignable, est retirée.
- Sondes `research` : `grammar/va_v2_research_test.go` (`TestVAV2Pertes`, `TestVAV2Vehicules`),
  `replaybuild/va_v2_etapes_research_test.go` (`TestVAV2Etapes`, et `VA_FAITS` : les faits des films
  du gate de corpus hors du corpus de `replay-equiv`) ; corrections du contrôle :
  `facts/killsource/va_v2_profil_research_test.go` (`TestVAV2ProfilCalibre` : le profil que la cuisson
  pose, en JSON) et `grammar/va_v2_corr_research_test.go` (`TestVAV2CorrMorts` : la marche des morts
  sous ce profil, chaque lecture que la fin de la vue A retire ou ajoute, instruite paquet par paquet).

Tests (vecteurs d'après l'écrivain) :

- `grammar/debut_par_vue_a_test.go` (neuf) : E > S, signature fictive DANS la vue A (un Script dont la
  charge porte un bit nul puis une signature stricte) → E, film ÉGALE, cuisson et marches ; E < S, la
  marche depuis E bute sur un masque hors archétype avant une signature → E pour ÉGALE, le localisateur
  à l'identique pour PRÉFIXE ; PRÉFIXE prouvé par la fermeture → E ; vue A lue en partie (bit de
  configuration à 0, genre 85 non porté, film sans table) → le localisateur à l'identique ; vrai
  paquet `bcb6d393` 1:204 (quarante Script) → 5605 par la vue A ; marche des morts sur la bobine
  réelle : localise exactement ce que `DebutDeLaVueB` localise, et plus que le localisateur seul ; juge
  de la preuve PRÉFIXE (correction 3, §14, `TestUnFilmAncienNePrendPasUneFinDeVueAFermeeAuBitSeulement`) :
  la marche depuis E ferme au bit près en contredisant l'ordre de l'écrivain (`InvariantOrdreVueB`)
  → E pour ÉGALE, le localisateur à l'identique (et pas E) pour PRÉFIXE.
- Les vecteurs de `debut_de_liste_test.go`, `debut_de_liste_masque_test.go` et
  `debut_par_fermeture_test.go` que la première version réécrivait selon la loi du bit nul, et le test
  de cette loi (`TestLaTeteDeLaVueBSuitUnBitNul`), sont retirés avec elle (correction 2) : ces trois
  fichiers sont ceux de V1.
- `frame_closure_detail_test.go` : par bobine, ≥ 1 liste lue en plus de ≥ 1 localisée et ≥ 1 non
  localisée (`ks_000d5950` : 119 / 1 / 353 ; `ks_e5adf7b2` : 40 / 1 / 13).
- `marche_trames_test.go` : docs de `rangerUnAvec` et de `TestLaMarcheDepuisLaTeteNeTraversePasUneVueANonVide`.
- `localisateur_test.go` : `marchDebut` à quatre paramètres.
- `killsource/testdata/minibobine.golden` : 460 / 473 → 473 / 473 paquets à événements localisés.



## 4. Table par film (carte v2, gate 2)

Carte v2 (`research/cmd_fermeture -mode v2 -denominateur-fixe`, table ECS figée de la campagne,
`-plafond-gib 4`), base `87cdfa761` (`scratchpad/cg3-V1/carte_base`, identique à l'octet à V1) contre
la tête corrigée (`scratchpad/cg3-V2c/carte_tete`). Gate 2 par `integ2/gate2.awk`. **Mesuré** :
`fermeture_paquets.tsv` de la tête corrigée est identique à l'octet à la pièce `carte_E` de la
première version (la surcouche m04, « fin de la vue A seule ») ; les autres fichiers de la carte ne
diffèrent que par le pic mémoire et la durée.

| Film | Build | Classe | Sains base | Sains V2 | Net | Perdus bruts (tous E) | dont contredits | Gagnés | Gains au bit (factices) | Pertes au bit | Utiles sains base | Utiles sains V2 | Net utiles | Utiles sains perdus bruts |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| `0797ce72` | HI_1_13_0 | egale | 19156 | 20772 | +1616 | 4 | 0 | 1620 | 1610 (1) | 8 | 159929 | 176835 | +16906 | 29 |
| `396cfc92` | HI_1_13_0 | egale | 23131 | 25954 | +2823 | 94 | 0 | 2917 | 2911 (0) | 96 | 169502 | 196024 | +26522 | 782 |
| `4f77afc1` | HI_1_13_0 | egale | 24106 | 31015 | +6909 | 103 | 0 | 7012 | 6310 (11) | 297 | 607967 | 822451 | +214484 | 1401 |
| `51ebbc0f` | HI_1_13_0 | egale | 20444 | 26877 | +6433 | 8 | 0 | 6441 | 6364 (1) | 16 | 140632 | 197932 | +57300 | 53 |
| `bcb6d393` | HI_1_12_0 | egale | 5880 | 7119 | +1239 | 4 | 0 | 1243 | 1241 (1) | 6 | 35502 | 46309 | +10807 | 10 |
| `bf15f7ab` | HI_1_13_0 | egale | 28603 | 29914 | +1311 | 2 | 0 | 1313 | 1305 (0) | 2 | 216104 | 228138 | +12034 | 8 |
| `bfecd02b` | HI_1_13_0 | egale | 27668 | 31067 | +3399 | 7 | 0 | 3406 | 3386 (0) | 9 | 239064 | 276442 | +37378 | 44 |
| `c75f33b8` | HI_1_13_0 | egale | 23872 | 27272 | +3400 | 3 | 0 | 3403 | 3316 (0) | 7 | 153997 | 180599 | +26602 | 26 |
| `d9781168` | HI_1_13_0 | egale | 26317 | 35774 | +9457 | 66 | 0 | 9523 | 9305 (0) | 95 | 180854 | 263867 | +83013 | 524 |
| `f75e7053` | HI_1_13_0 | egale | 23673 | 24517 | +844 | 59 | 0 | 903 | 898 (0) | 62 | 162137 | 168248 | +6111 | 436 |
| `fb1a1a72` | HI_1_13_0 | egale | 43457 | 46308 | +2851 | 4 | 0 | 2855 | 2821 (0) | 7 | 325815 | 351322 | +25507 | 15 |
| `50247b26` | version-31 | illisible | 139 | 139 | +0 | 0 | 0 | 0 | 0 (0) | 0 | 274 | 274 | +0 | 0 |
| `60ae07c4` | HI_1_8_0 | illisible | 13948 | 13948 | +0 | 0 | 0 | 0 | 0 (0) | 0 | 83685 | 83685 | +0 | 0 |
| `a349fea8` | version-33 | illisible | 424 | 424 | +0 | 0 | 0 | 0 | 0 (0) | 0 | 3654 | 3654 | +0 | 0 |
| `a521164d` | HI_1_4_1 | illisible | 692 | 692 | +0 | 0 | 0 | 0 | 0 (0) | 0 | 78 | 78 | +0 | 0 |
| `084a804d` | HI_1_10_0 | prefixe | 4837 | 4992 | +155 | 0 | 0 | 155 | 115 (0) | 0 | 89755 | 93780 | +4025 | 0 |
| `111fa685` | HI_1_10_0 | prefixe | 4026 | 4090 | +64 | 0 | 0 | 64 | 59 (0) | 0 | 42896 | 44298 | +1402 | 0 |
| `11de8353` | HI_1_9_0 | prefixe | 5631 | 5768 | +137 | 0 | 0 | 137 | 121 (0) | 0 | 65658 | 68563 | +2905 | 0 |
| `1c4c63c2` | HI_1_10_0 | prefixe | 13389 | 14245 | +856 | 0 | 0 | 856 | 521 (11) | 4 | 165987 | 187431 | +21444 | 0 |
| `e5adf7b2` | HI_1_11_0 | prefixe | 4149 | 4311 | +162 | 0 | 0 | 162 | 145 (0) | 0 | 80107 | 84083 | +3976 | 0 |
| **corpus** | | | 313542 | 355198 | +41656 | 354 | 0 | 42010 | 40428 (25) | 609 | 2923597 | 3474013 | +550416 | 3328 |

Lecture du gate 2 (« aucun film en baisse, NET par film, paquets sains ET records utiles sains ») :
**tenu, sans exception**. 16 films en hausse sur les deux comptes, les 4 films illisibles inchangés à
l'unité (la fin de la vue A n'y joue pas, et les candidats de tête sont ceux d'avant). Aucune exception
D2 n'est demandée : les 14 fermetures factices que la première version retirait sur `60ae07c4`,
`a349fea8` et `a521164d` restent ce qu'elles étaient en base (§14, correction 2).

- Part factice des gains au bit : 25 / 40 428 (0,1 %).
- D1 (part fixe, `integ2/d1.awk`, dénominateur fixe recalculé, fixe 7 823 078) : tête corrigée 44,4 %
  sur le corpus, 91,2 % sur HI_1_13_0, 31,3 % sur HI_1_12_0, 21,9 % sur HI_1_11_0, 13,4 % sur
  HI_1_10_0, 21,1 % sur HI_1_9_0 — les valeurs de la première version à la décimale ; HI_1_8_0,
  HI_1_4_1, version-31, version-33 inchangés.

### 4.1 Décomposition par règle (première version, pour mémoire)

La première version portait deux règles neuves ; deux cartes de plus en retiraient une chacune par
surcouche (`scratchpad/cg3-V2/decomp.sh`, même base, même gate 2) :

| Variante | Sains | Net | Perdus bruts | Gagnés | Factices / gains au bit | Utiles sains (net) | Films en baisse |
|---|---|---|---|---|---|---|---|
| fin de la vue A seule (= **V2 corrigée**, identique à l'octet) | 355 198 | +41 656 | 354 (tous classe E) | 42 010 | 25 / 40 428 | 3 474 013 (+550 416) | **0** |
| loi du bit nul seule | 312 835 | −707 | 844 | 137 | 17 / 154 | 2 916 306 (−7 291) | 18 |
| première version (les deux) | 354 796 | +41 254 | 752 | 42 006 | 29 / 40 428 | 3 473 088 (+549 491) | 3 (D2) |

Tout le gain vient de la fin de la vue A. La loi du bit nul seule fait baisser 18 films ; avec la fin
de la vue A, elle coûtait 402 sains et faisait baisser les trois films illisibles. Elle est retirée
(§14, correction 2) : V2 est désormais la première ligne.


## 5. Les 354 sains perdus, instruits paquet par paquet

**Mesuré** : les 354 paquets perdus par la tête corrigée sont exactement les 354 pertes de classe E de
la première version (même liste film:paquet, même cause d'arrêt dans la carte, `comm` vide). Leur
instruction est rejouée sur le code corrigé par la sonde `va_v2_research_test.go` (`TestVAV2Pertes`,
tag `research`) : la marche de la carte v2 (`rnMarcher`, qui suit la cuisson) ; pour chaque perdu, sur
le monde d'avant lui, la vue A lue, le début V2 et comment, le début que la règle d'AVANT rendrait sur
ce même monde (S0) et son bit précédent, les verdicts des marches d'essai, le point d'arrêt de la marche
depuis le début V2 ; puis le verdict que la marche rend au paquet.

**Contrôle croisé** : 354 / 354 perdus instruits ; le verdict et la cause rendus par la sonde sont ceux
de la carte (0 désaccord) ; S0 est rendu par la règle d'avant le lot sur le monde de la tête.

| Classe | Perdus | Utiles sains perdus | Instruction |
|---|---|---|---|
| **E** — la fin de la vue A décide (films ÉGALE) | 354 | 3 328 | la vue A est lue jusqu'à son terminateur dans les 354 ; la marche depuis E ne ferme pas ; depuis S0 elle fermait (monde de la tête). E < S0 dans les 354 ; S0 venait de la signature (268), de la fermeture par NEW de tête (82) ou de la chaîne (4). La marche depuis E s'arrête AVANT S0 dans 350 cas (la loi dit E, elle ne dit pas si S0 est une frontière de record) et passe S0 dans 4 (S0 tombe dans un record lu depuis E : contredit). S0 était lui-même précédé d'un bit à 1 dans 24 cas (D-VAV2-7). **Elles ne se réparent pas en revenant à la signature** (décision (1)) : la cure est la grammaire des records de tête de la vue B (VB-têtes) |

Les classes L (loi du bit nul) et C (cascade de liaison posée à un début que la loi contredit) de la
première version n'existent plus : elles venaient de la loi, retirée.

### 5.1 Les pertes E, par composant (pour les lots suivants)

Cause du premier arrêt de la marche depuis E (colonne `cause` de la carte ; identique ligne à ligne à
la première version) :

| Composant / arrêt | Perdus | Films |
|---|---|---|
| `ti=43 i19 device-position-animation-name-component` | 139 | `f75e7053` 43, `4f77afc1` 11, `396cfc92` 85 |
| `ti=12 i16 managed-navpoint-override-flags` | 61 | `f75e7053` 12, `bfecd02b` 4, `4f77afc1` 39, `396cfc92` 4, `0797ce72` 1, `fb1a1a72` 1 |
| `vue C : terminateur hors cadre` | 44 | `bcb6d393` 1, `4f77afc1` 4, `396cfc92` 1, `d9781168` 37, `fb1a1a72` 1 |
| `vue B : sortie par rejet` | 37 | `4f77afc1` 31, `51ebbc0f` 1, `d9781168` 5 |
| `ti=10 i2 managed-object-navpoint-component` | 19 | `bcb6d393` 1, `4f77afc1` 3, `51ebbc0f` 5, `d9781168` 10 |
| `ti=45 i0 matchflow-sequence-data-component` | 19 | `bcb6d393` 1, `f75e7053` 1, `bfecd02b` 3, `c75f33b8` 3, `bf15f7ab` 2, `51ebbc0f` 1, `396cfc92` 4, `0797ce72` 1, `d9781168` 2, `fb1a1a72` 1 |
| `vue C : bloc 0xbc (desalignement)` | 12 | `d9781168` 12 |
| `ti=12 i18 managed-navpoint-position-offset` | 7 | `f75e7053` 3, `4f77afc1` 1, `51ebbc0f` 1, `0797ce72` 1, `fb1a1a72` 1 |
| `ti=43 i21 device-position-group-component` | 6 | `4f77afc1` 6 |
| `ti=10 i22 managed-object-interaction-filter-component` | 3 | `4f77afc1` 3 |
| `ti=35 i59 biped-spartan-ability-non-predicted-state` | 2 | `4f77afc1` 2 |
| `ti=10 i18 managed-object-networked-property-component` | 1 | `4f77afc1` 1 |
| `ti=12 i22 managed-navpoint-visual-state-groups-component-2` | 1 | `4f77afc1` 1 |
| `ti=43 i35 device-animation-layer-state-component` | 1 | `bcb6d393` 1 |
| `ti=43 i39 device-machine-flags-component` | 1 | `4f77afc1` 1 |
| `ti=45 i1 matchflow-focus-data-component` | 1 | `0797ce72` 1 |


Lecture : `ti=43 i19 device-position-animation-name` (139, dont 85 sur `396cfc92` et 43 sur
`f75e7053`) et `ti=12 i16 managed-navpoint-override-flags` (61, dont 39 sur `4f77afc1`) font 56 % des
pertes E, comme la vérification adverse du plan l'annonçait ; les arrêts en vue C (56) et les sorties
de vue B par rejet (37, slot inconnu du monde) suivent. `ti=10 i2`, `ti=45 i0`, `ti=12 i18`,
`ti=43 i21` complètent. Ce sont des composants non portés ou des records de tête non traversés, pas
des fautes de la vue A.


## 6. `killsource` (gate 3) et la marche des morts

**`cmd/killsource json`**, 20 films (19 témoins + `1c4c63c2`, carte Refuge), binaire de `87cdfa761`
contre binaire de la tête corrigée : **19 JSON identiques à l'octet** ; sur `c75f33b8`, une seule ligne
diffère, `concordance.enregistrements_lus_par_les_deux_voies` 5 → 6 (`Stats.Redundant`, diagnostic non
persisté). Les 20 JSON sont identiques à l'octet à ceux de la première version : la marche de
`killsource` n'utilise pas les candidats de tête. **Aucune mort, aucune valeur, aucune voie ne
change.** `replay-equiv` : l'étape `killRefs` est identique partout ; l'étape `killsource` diverge sur
16 films parce qu'elle hache tout le `Result` (compteurs `Stats.PacketsLocated`,
`Replis.LocalisationsALargeurLibre`, `Replis.IndicesHorsRoster`, non persistés dans
`match_kill_events`) ; elle est identique à la première version sur les 20.

**`killsource.Rev` reste `killsource-2026-09-27`** (D23 : aucune sortie persistée ne change) :
complément de chronique, golden régénéré à révision constante. **Aucun backlog du parc n'est ouvert.**
La mini-bobine de `killsource` (`testdata/minibobine.golden`) passe de 460 / 473 à 473 / 473 paquets à
événements localisés (ligne de diagnostic).

### 6.1 La marche des morts d'objet (vérification (d))

La marche à huit vues (`ScanMarchFacts`, `object_deaths_march.go`) part de la fin de la vue A quand
elle décide (V2 explicite). Elle n'utilise pas les candidats de tête : l'étape `vehicles` de
`replay-equiv` est identique à la première version sur les 20 films.

**Films PRÉFIXE** (corpus de `replay-equiv`) : étape `vehicles` divergente contre la base sur 6 films
(`084a804d`, `111fa685`, `11de8353`, `1c4c63c2`, `53ce4390`, `e5adf7b2`), identique sur les 14 autres ;
valeur de l'étape (sonde `TestVAV2Etapes`) : seuls `Deaths`, `Occupancy` et `DeathStats` changent, par
ajout seulement :

| Film | Morts de véhicule (base → V2) | Lectures d'occupation (base → V2) | Retirées |
|---|---|---|---|
| `084a804d` | 35 → 35 | 113 → 127 (+14) | 0 |
| `111fa685` | 4 → 4 | 8 → 9 (+1) | 0 |
| `11de8353` | 15 → **16** (+1 : slot 783 `ti=40`, `Mort`, `t = 2354459518` µs) | 26 → 34 (+8) | 0 |
| `1c4c63c2` | 4 → 4 | 24 → 47 (+23) | 0 |
| `53ce4390` | 3 → 3 | 29 → 60 (+31) | 0 |
| `e5adf7b2` | 17 → 17 | 63 → 75 (+12) | 0 |

**Films ÉGALE à véhicules (correction 1 du contrôle).** Le corpus de `replay-equiv` n'en contient
aucun : ses films ÉGALE sont des cartes sans `ti=40` aux images-clés, leur étape `vehicles` est vide en
base comme en V2. Les deux films ÉGALE à véhicules du gate de corpus, `4f77afc1` (Flood Gulch) et
`bfecd02b` (Snowbound), sont mesurés à part : **étape `vehicles` PUBLIÉE**, valeur écrite par
`TestVAV2Etapes` (cuisson du harnais sous la racine du gate, `VA_FAITS` = les faits du gate, binaires
de `87cdfa761` et de la tête) :

| Film | Morts `ti=40` (base → V2) | Retirées / ajoutées | Lectures d'occupation (base → V2) | Retirées / ajoutées |
|---|---|---|---|---|
| `4f77afc1` | 74 → 79 | **0** / 5 | 144 → 318 | **5** / 179 |
| `bfecd02b` | 1 → 1 | 0 / 0 | 7 → 8 | 0 / 1 |

L'affirmation de la première version (« uniquement par ajout, aucune lecture de la base ne disparaît »)
est donc **fausse sur `4f77afc1`** : cinq lectures d'occupation disparaissent. Aucune mort publiée ne
disparaît.

**Le contexte compte, et c'est la réserve du contrôle.** La sonde du contrôle (`ScanMarchFacts` sous
`NewFilmContextForMap` et la carte, sans autre pose) mesurait 87 → 87 morts dont 22 retirées et 22
ajoutées, 318 → 304 occupations dont 75 retirées, et sur `bfecd02b` 106 → 104 dont 8 retirées. Ma
sonde reproduit ces nombres à l'identique dans ce contexte. Mais l'étape publiée marche sous le profil
que `killsource` calibre sur le film (`Result.ProfilCalibre`, posé par `replay.poserProfilPuisCarte` :
génération stricte, traversée, largeur d'axe absolue), puis les largeurs d'objet du monde de la carte
et les largeurs MPP relues (`gwInstallMPPWidths`). Sous ce contexte reconstruit
(`killsource/va_v2_profil_research_test.go` écrit le profil calibré, identique à l'octet sous la base
et sous la tête ; `grammar/va_v2_corr_research_test.go` le pose), la sonde rend **exactement** les
comptes de l'étape publiée sur les deux films (74 → 79, 0 / 5 ; 144 → 318, 5 / 179 ; 1 → 1 ; 7 → 8,
0 / 1), et sur `11de8353` (PRÉFIXE) les 15 → 16 morts de `replay-equiv` (occupations 27 → 33 contre
26 → 34 publiées : les largeurs MPP de ce build ne sont pas relues, la cuisson y pose celles qu'elle
calibre ailleurs, que la sonde ne reconstruit pas). Les nombres du contrôle décrivent une marche que la
production ne fait pas ; la réserve est levée par cette mesure.

**Les cinq lectures retirées, instruites paquet par paquet** (même monde des deux côtés : la marche
restaure le monde après chaque paquet ; « S » = le localisateur `SignaturePuisLargeurLibre`, début de
la base ; « E » = la fin de la vue A, début de V2) :

| Instant (µs) | Lecture retirée | Bit du record (base) | E | S | Marche depuis E : records, fin, arrêt | Vie relue ailleurs en V2 |
|---|---|---|---|---|---|---|
| 7723140170 | slot 2120 gen 0, branche libre (`Attached=false`) | 1490 | 26 | 1450 | 22 records, fin au bit 3517 (passe S sans qu'aucun record n'y commence), désynchronisée sur `ti=0` slot 1038 à `i0 game-engine-team-mapping-component` | non |
| 8268681033 | slot 4873 gen 3, branche libre | 2935 | 231 | 2656 | 3 records, fin au bit 346 < S, arrêt `ti=0` slot 512 à `i0 game-engine-team-mapping-component` | non |
| 8286033133 | slot 622 gen 1, branche libre | 4775 | 323 | 4693 | 4 records, fin au bit 800 < S, arrêt `ti=0` slot 516 à `i0` | oui |
| 8341387343 | slot 729 gen 1, à bord du slot 965, siège 52 | 1678 | 26 | 1615 | 3 records, fin au bit 182 < S, arrêt `ti=0` slot 518 à `i0` | non |
| 8477226401 | slot 700 gen 1, à bord du slot 5668, sans siège | 2761 | 556 | 2734 | 3 records, fin au bit 712 < S, arrêt `ti=0` slot 533 à `i0` | oui |

Instruction : **pertes de classe E (décision (1))**. Dans les cinq paquets, la vue A est lue jusqu'à son
terminateur, E < S, et la lecture retirée était lue par la base dans la vue B à partir de S. La marche
depuis E s'arrête avant S quatre fois (sur le composant `i0 game-engine-team-mapping-component` de
l'archétype `ti=0`, non traversé), et une fois passe S sans y trouver de frontière de record (S
contredit par les records lus depuis E). Revenir à la signature serait la convention que la décision
(1) écarte ; la cure est la grammaire des records de tête de la vue B (lot VB-têtes, D-VAV2-4). Les
valeurs retirées sont elles-mêmes peu vraisemblables — trois lectures sans parent, un siège 52 (champ
de six bits ; les sièges mesurés sur les parents véhicule de ce film sont dans {0, 1, 2},
`types.VehicleOccupancy`), un parent au slot 5668 — mais c'est une observation, pas le motif du
classement. Les 5 morts et 179 occupations ajoutées (180 lectures avant déduplication) sont toutes lues en vue B,
dans des paquets où S est absent (la base ne localisait pas la liste), avant E (la base partait dans
la vue A : 134 lectures) ou après E.

`repli_localisation_largeur_libre` (journal « repli déclenché » de `replay-equiv`, 20 films) : 25 686 →
**19 972** (identique à la première version) ; il baisse là où E localise, il est inchangé sur les 4
films illisibles. Registre des replis, sommes sur les 20 films :

| Repli | Base | Première version | V2 corrigée |
|---|---|---|---|
| `repli_localisation_largeur_libre` | 25 686 | 19 972 | 19 972 |
| `repli_debut_de_liste_ferme_au_bit` | 8 047 | 5 240 | 7 053 |
| `repli_liaison_par_anticipation` | 7 861 | 7 454 | 7 639 |
| `repli_identite_premier_occupant_du_siege` | 1 313 | 1 341 | 1 341 |
| `repli_episode_occupation_par_trou_de_position` | 89 | 59 | 59 |
| `repli_episode_borne_par_la_vie_suivante` | 49 | 29 | 29 |
| `repli_deadstate_hors_bande_bipede` | 257 | 264 | 264 |
| `repli_deadstate_indice_hors_roster` | 219 | 221 | 221 |

(La première version donnait certains de ces comptes sur les seuls films changés ; ceux-ci sont sur les
20 films.)


## 7. Gates (sorties exactes, tête corrigée)

Environnement : `GOCACHE=C:/Users/Guillaume/AppData/Local/go-build-cg3-vuea`, CGO msys64 ucrt64, une
commande go à la fois ; films lus en place (`data/cache/film_chunks`, lecture seule) ; racine factice
et copie du parc au scratchpad (`scratchpad/cg3-V1/repo`, `scratchpad/cg3-V1/parc`). Pièces :
`scratchpad/cg3-V2c/`. Les gates 1 à 11 ci-dessous ont tourné sur la tête corrigée AVANT fusion (base
`87cdfa761`) ; le `git fetch` qui les suivait a trouvé `origin/feat/v75` avancé à `28c542b33` : fusion
et gates rejoués au §7.1.

1. `gofmt -l ./internal/ ./cmd/` : vide.
2. `go vet ./...` : rc=0. `go vet -tags=research ./internal/games/halo_infinite/film/... ./internal/replaybuild/` : rc=0.
3. `go test ./internal/archlint/ -count=1` : `ok  levelup/go-api/internal/archlint 43.718s`.
4. G-film `go test ./internal/games/halo_infinite/film/... ./internal/replaybuild/... ./internal/sync/killcollector/... -count=1 -timeout 30m` :
   rc=0, 21 paquets `ok` (`scratchpad/cg3-V2c/gfilm.txt`), dont `grammar`, `facts/killsource`,
   `facts/objectives`, `replay`, `types`, `replaybuild`, `sync/killcollector`.
5. `golangci-lint` (`GOLANGCI_LINT_CACHE` au scratchpad) : recette CI `--new-from-merge-base=origin/main ./...` : `0 issues.` ; paquets touchés
   (`film/internal/grammar/...`, `film/internal/facts/...`, `film/research/cmd_fermeture/...`,
   `internal/archlint/...`, `internal/replaybuild/...`), toutes les issues : `0 issues.` ; sous
   `--build-tags=research --new-from-rev=8354c0d43` (grammar, facts, replaybuild) : `0 issues.`
6. Carte v2, 20 films, `-plafond-gib 4` : rc=0, 20 films, 0 échec ; `fermeture_paquets.tsv` identique
   à l'octet à `cg3-V2/carte_E`. Gate 2 : §4.
7. `cmd/killsource json`, 20 films : rc=0 partout ; 19 JSON identiques à l'octet à la base,
   `c75f33b8` : une ligne (§6) ; 20 identiques à la première version.
8. `replay-equiv` (20 films, racine factice) : « BILAN : 0 identique(s), 20 different(s) » contre les
   références non re-figées (rc=1, attendu). Base contre tête, étape par étape : divergent `artifact`
   (20 films), `movementStates` (16), `movementStates.stats` (16), `killsource` (16, compteurs non
   persistés, §6), `continuousFire.stats` (16), `continuousFire` (13), `vehicles` (6, §6.1) ;
   **identiques** : `objectives`, `killRefs`, `deaths`, et toutes les autres étapes. Contre la première
   version : `vehicles` et `killsource` identiques sur les 20 ; divergent `artifact` (20),
   `movementStates.stats` et `continuousFire.stats` (10), `movementStates` (6 : `084a804d`, `111fa685`,
   `11de8353`, `1c4c63c2`, `60ae07c4`, `e5adf7b2`), `continuousFire` (5) — les films où la loi retirée
   jouait. Fixtures de contrat web régénérées (`REPLAY_CONTRACT_UPDATE=1 … -run ContractFixtures
   -update`) : 8 fixtures et le manifeste, chaînes de révision.
9. `replay-corpus-gate --reference=base --base=87cdfa761 --parc-root <copie> --work-root <scratchpad>` :
   **rc=1**. Tableau :

```
temoin       famille          base(87cdfa761)   HEAD    gains   pertes   chang.      duree base   cache  statut
bcb6d393     ctf_mono_manche      78     78       23       18        3      4.97s cache  A+F    PERTE
fb1a1a72     ctf_multi_manche     78     78       48       25        3     12.94s cache  A+F    PERTE
d9781168     oddball              78     78       66       75        4      9.44s cache  A+F    PERTE
c75f33b8     assaut_bombe         78     78       44       44        4      6.19s cache  A+F    PERTE
bf15f7ab     slayer               78     78       27       21        3      5.92s cache  A+F    PERTE
51ebbc0f     deux_manches         78     78       49       35        3      6.79s cache  A+F    PERTE
084a804d     vehicules            78     78      118       28       10   1m12.36s cache  A+F    FAUX
0797ce72     region_index_2_bits     78     78       30       13        3      7.37s cache  A+F    PERTE
111fa685     version_39           78     78       96       17        5     24.01s cache  A+F    FAUX
e5adf7b2     version_40_build_1_11     78     78       79       34        6     25.48s cache  A+F    PERTE
60ae07c4     version_37           78     78        0        0        0     16.49s cache  A+F    ok
a349fea8     version_33_sans_identification     78     78        0        0        0   1m25.52s cache  A+F    ok
a521164d     version_33_build_1_4_1     78     78        0        0        0     23.11s cache  A+F    ok
11de8353     version_38_build_1_9_0     78     78       76        7       29     24.61s cache  A+F    PERTE
50247b26     version_31_sans_identification     78     78        0        0        0     45.77s cache  A+F    ok
bfecd02b     vehicules_v41_utilisateur     78     78       67       25        6      9.92s cache  A+F    PERTE
4f77afc1     equipement_origine_utilisateur     78     78      276      112       17     53.61s cache  A+F    FAUX
396cfc92     strongholds_zones     78     78       33       17        3     11.25s cache  A+F    PERTE
f75e7053     koth_collines        78     78       35       14        2      5.65s cache  A+F    PERTE

```

   Banc de vérité : **16 / 19 « ok »**, **0 « MANQUE »**, 3 « FAUX ».
   - Les trois films illisibles qui rendaient `MANQUE` dans la première version (`60ae07c4`,
     `a349fea8`, `a521164d`) sont « ok », sans gain ni perte : « P-1 paquets fermés au bit près » y est
     celui de la base.
   - `FAUX` (`4f77afc1` V-3 112 → 263, `084a804d` 92 → 119, `111fa685` 5 → 15) : les valeurs de la
     première version. Rejoué sur les artefacts de cette tête (`scratchpad/cg3-V2/v3.jq`) : les instances
     « tir » de V-3 sont 252, 105 et 10, et **toutes sont des tirs de véhicule couverts par un trajet du
     même slot dans ce véhicule** (252 / 252, 105 / 105, 10 / 10), comme dans la première version (même
     instruction : le banc prend les vies dans les seules pistes bipèdes, D-VAV2-3).
   - `[FILET]` : 485 lignes (420 de postures). Oracle physique rejoué (`oracle_corpus.js`,
     `postures_corpus.tsv`) : sprint 479 469 → 452 963 images (37 073 retirées, 10 545 ajoutées),
     escalade 39 191 → 29 854, saut dérivé 79 412 → 80 556 ; vitesse horizontale médiane des images de
     sprint retirées 1,90 à 2,34 sur 15 films contre 2,72 à 2,83 en sprint commun (et 1,91 à 2,24 hors
     posture) ; l'escalade la plus longue de `c75f33b8` tombe de 1 960 images à 8 : **fausse continuité
     retirée**, comme dans la première version. Trous du tir continu (`holesOpenViewB`, `holesKind`,
     `holesBlockBC`) et trous de rafales : reclassés, instruits par famille, pas paquet par paquet.
     Lignes `[FILET]` des véhicules identiques à la première version ; sur `4f77afc1`, la durée du
     véhicule 772 tombe de 3 539 à 2 137 images : sa mort est lue en V2 (slot 772, 7 588 058 591 µs,
     §6.1).
   - Télémétrie : `grammarRev` `grammar-2026-10-03.5 -> grammar-2026-10-06.3`, `profileRev`
     `profile-2026-09-17.3 -> profile-2026-10-06` (19 témoins).
10. Sondes : `TestVAV1Tete` (20 films) : `va_v1_tete.tsv` identique à celui de V1 et de la première
    version ; `TestVAV2Pertes` (354 perdus, §5) ; `TestVAV2Etapes` et `TestVAV2CorrMorts` (§6.1).
11. Mutations : §8.

### 7.1 Après la fusion de `feat/v75` (`28c542b33`)

`origin/feat/v75` a avancé pendant l'étape : `28c542b33` (« fix(rejeu) : tourelles grises, tirs depuis
la bouche des armes, cercle de retour du drapeau », `replay.SchemaVersion` 78 → 79). Fusionné
(`feat/v75` a raison) après le commit des corrections. Conflits : les fixtures de contrat web
(`replay_schema_78_*` supprimées par `feat/v75`, `replay_schema_79_*` créées) — prises de `feat/v75`
puis régénérées par la commande du dépôt : les 8 identiques à celles de `28c542b33` une fois les chaînes
`grammar-…` et `profile-…` neutralisées. `.ai/thought_log.md` : fusion automatique, les deux entrées
gardées. Aucun fichier du décodeur (`film/internal/grammar`, `facts/killsource`, `profile`, `source`,
`film/research`) n'est touché par `28c542b33`.

Gates rejoués sur la tête fusionnée (`scratchpad/cg3-V2c/fusion/`) :

1. `gofmt` vide ; `go vet ./...` rc=0 ; `go vet -tags=research` rc=0 ; archlint `ok`.
2. G-film : rc=0, 21 paquets `ok`.
3. `golangci-lint` : recette CI, paquets touchés, et `--build-tags=research --new-from-rev=8354c0d43` :
   `0 issues.` aux trois.
4. Carte v2 : `fermeture_paquets.tsv` identique à l'octet à celle d'avant la fusion (gate 2 : §4,
   inchangé).
5. `cmd/killsource json`, 20 films : rc=0, 20 JSON identiques à l'octet à ceux d'avant la fusion.
6. `replay-equiv` contre la NOUVELLE base (`28c542b33`, binaire construit depuis `git archive`) : mêmes
   étapes divergentes qu'avant la fusion contre `87cdfa761` (`artifact` 20, `movementStates` 16,
   `movementStates.stats` 16, `killsource` 16, `continuousFire.stats` 16, `continuousFire` 13,
   `vehicles` 6 — les 6 mêmes films) ; `objectives`, `killRefs`, `deaths` identiques. Contre la tête
   d'avant la fusion : seule l'étape `artifact` diverge.
7. `replay-corpus-gate --reference=base --base=28c542b33` (base cuite par le gate) : **rc=1**, même
   verdict qu'avant la fusion — 16 / 19 « ok », 0 `MANQUE`, 3 `FAUX` de V-3 (92 → 119, 5 → 15,
   112 → 263 ; tirs « hors vie » 105, 10 et 252, tous des tirs de véhicule couverts par un trajet), 485
   lignes `[FILET]`. Tableau :

```
temoin       famille          base(28c542b33)   HEAD    gains   pertes   chang.      duree base   cache  statut
bcb6d393     ctf_mono_manche      79     79       23       18        3      5.26s cuite  A+F    PERTE
fb1a1a72     ctf_multi_manche     79     79       48       25        3     13.75s cuite  A+F    PERTE
d9781168     oddball              79     79       66       75        4      9.51s cuite  A+F    PERTE
c75f33b8     assaut_bombe         79     79       44       44        4      6.29s cuite  A+F    PERTE
bf15f7ab     slayer               79     79       27       21        3      5.87s cuite  A+F    PERTE
51ebbc0f     deux_manches         79     79       49       35        3      6.74s cuite  A+F    PERTE
084a804d     vehicules            79     79      118       28       10   1m11.99s cuite  A+F    FAUX
0797ce72     region_index_2_bits     79     79       30       13        3      8.38s cuite  A+F    PERTE
111fa685     version_39           79     79       96       17        5     24.14s cuite  A+F    FAUX
e5adf7b2     version_40_build_1_11     79     79       79       34        6     26.72s cuite  A+F    PERTE
60ae07c4     version_37           79     79        0        0        0     19.23s cuite  A+F    ok
a349fea8     version_33_sans_identification     79     79        0        0        0   1m43.06s cuite  A+F    ok
a521164d     version_33_build_1_4_1     79     79        0        0        0     21.64s cuite  A+F    ok
11de8353     version_38_build_1_9_0     79     79       76        7       29     21.31s cuite  A+F    PERTE
50247b26     version_31_sans_identification     79     79        0        0        0     45.66s cuite  A+F    ok
bfecd02b     vehicules_v41_utilisateur     79     79       67       25        6      8.79s cuite  A+F    PERTE
4f77afc1     equipement_origine_utilisateur     79     79      276      112       17     48.21s cuite  A+F    FAUX
396cfc92     strongholds_zones     79     79       33       17        3     11.55s cuite  A+F    PERTE
f75e7053     koth_collines        79     79       35       14        2      5.41s cuite  A+F    PERTE

```

8. Mutations rejouées : **13 / 13 ROUGES**, chacune sur le test attendu
   (`scratchpad/cg3-V2c/fusion/mutations.txt`).

### 7.2 Après la seconde fusion de `feat/v75` (`65c99b669`)

`origin/feat/v75` a de nouveau avancé : `65c99b669` (relecture de V1, goldens `killsource` régénérés).
Fusionné (`30d94a60b`, `feat/v75` a raison) ; conflits résolus et décrits au message de fusion
(archlint, doc de `vue_a_versions.go`, `grammar_rev.golden`, fixtures de contrat). `git fetch` refait
avant ce rapport (2026-10-06) : `origin/feat/v75` = `65c99b669`, pas d'avance.

Gates rejoués sur la tête fusionnée `30d94a60b` (`scratchpad/cg3-V2c/fusion2/`, `fusion2.sh`) :

1. `gofmt` vide ; `go vet ./...` rc=0 ; `go vet -tags=research` rc=0 ; archlint `ok`.
2. G-film : rc=0, 21 paquets `ok`.
3. `golangci-lint` : recette CI (`--new-from-merge-base=origin/main`), paquets touchés, et
   `--build-tags=research --new-from-rev=8354c0d43` : `0 issues.` aux trois.
4. Carte v2 : `fermeture_paquets.tsv` et les 13 autres tables identiques à l'octet à celles d'avant
   la fusion ; `fermeture_films.tsv` identique hors les colonnes `pic_octets` et `duree_ms` (mesures
   d'exécution). Gate 2 : §4, inchangé.
5. `cmd/killsource json`, 20 films : rc=0, 20 JSON identiques à l'octet à ceux d'avant la fusion.
6. `replay-equiv`, tête : les 20 TSV d'étapes identiques à l'octet à ceux de la tête d'avant la
   fusion (aucune étape ne change, `artifact` compris). Contre la NOUVELLE base (`65c99b669`, binaire
   construit depuis `git archive`, arbre vérifié identique à `git archive 65c99b669`) : exactement les
   mêmes couples (film, étape) divergents qu'au §7.1 (`artifact` 20, `movementStates` 16,
   `movementStates.stats` 16, `killsource` 16, `continuousFire.stats` 16, `continuousFire` 13,
   `vehicles` 6 — `084a804d`, `111fa685`, `11de8353`, `1c4c63c2`, `53ce4390`, `e5adf7b2`) ;
   `objectives`, `killRefs`, `deaths` identiques.
7. `replay-corpus-gate --reference=base --base=65c99b669` (base cuite par le gate) : **rc=1**, même
   verdict, même tableau (gains, pertes, changements et statuts identiques témoin par témoin) qu'au
   §7.1 — 4 « ok », 12 `PERTE`, 3 `FAUX` de V-3 (`084a804d`, `111fa685`, `4f77afc1`), 0 `MANQUE`,
   485 lignes `[FILET]`. Seule la télémétrie diffère : la base porte désormais
   `grammar-2026-10-06` et `profile-2026-10-06` (V1 est dans `feat/v75`).
8. Mutations rejouées : **13 / 13 ROUGES**, chacune sur le test attendu
   (`scratchpad/cg3-V2c/fusion2/mutations.txt`) ; les copies mutées ne diffèrent de la tête que par
   la mutation (vérifié, 2 à 4 lignes de `diff` chacune).

Incident d'exécution, sans effet sur les résultats retenus : le script des gates, laissé orphelin par
la fin de la session précédente, a été suspendu une nuit pendant le `replay-equiv` de la base (un film
« 12h33m », puis 16 enfants morts au lancement, code `0xC0000142`). Ce passage
(`re_base_echec.log`, `re_base_tsv_echec/`) est écarté ; `replay-equiv` de la base a été relancé seul
après la fin du script, une commande à la fois : 20 films, 0 échec (`re_base.log`).


## 8. Mutations (`-overlay`, suite entière du paquet ; ROUGE attendu)

Script `scratchpad/cg3-V2c/mutations.sh` (surcouches `mut/*/overlay.json`, portées sur les sources
corrigées), résultat `mutations.txt` : **13 / 13 ROUGES**, chacune sur le test attendu. La
surcouche de m14 visait d'abord un dossier renommé (échec de mise en place, rc 1 sans test) : corrigée et
rejouée seule, ROUGE sur son test.

| Mutation | Verdict | Tests en échec |
|---|---|---|
| m01_E_ignore | ROUGE | TestFrameClosureDetailleeRendLaCarteDeFrameClosure, TestLaFinDeLaVueADUnFilmRecentEstLeDebutDeLaVueB, TestLaFinDeLaVueAPrimeQuandLaMarcheButeEnsuite, TestLaMarcheDesMortsPartDeLaFinDeLaVueA, TestUnFilmAncienNePrendPasUneFinDeVueAFermeeAuBitSeulement, TestUnFilmAncienPrendLaFinDeSaVueAQuandElleFermeLePaquet, TestUnVraiPaquetPartDeLaFinDeSaVueA |
| m02_signature_reprise_apres_arret | ROUGE | TestLaFinDeLaVueAPrimeQuandLaMarcheButeEnsuite, TestLaMarcheDesMortsPartDeLaFinDeLaVueA, TestUnFilmAncienNePrendPasUneFinDeVueAFermeeAuBitSeulement, TestUnVraiPaquetPartDeLaFinDeSaVueA |
| m03_preuve_retiree_films_anciens | ROUGE | TestFrameClosureDetailleeRendLaCarteDeFrameClosure, TestFrameClosureRatchet, TestLaFinDeLaVueAPrimeQuandLaMarcheButeEnsuite, TestUnFilmAncienNePrendPasUneFinDeVueAFermeeAuBitSeulement |
| m05_E_sur_lecture_arretee | ROUGE | TestFrameClosureDetailleeRendLaCarteDeFrameClosure, TestFrameClosureRatchet, TestLaFermetureDeLaStructureNeDescendPas, TestScanObjectDeathsSurBobineReelle, TestUneVueALueEnPartieNeDecideRien |
| m06_E_malgre_configuration_a_0 | ROUGE | TestLeBitDeConfigurationAZeroArreteLaVueA, TestUneVueALueEnPartieNeDecideRien |
| m07_signature_avant_E | ROUGE | TestLaFinDeLaVueADUnFilmRecentEstLeDebutDeLaVueB, TestLaFinDeLaVueAPrimeQuandLaMarcheButeEnsuite, TestUnFilmAncienPrendLaFinDeSaVueAQuandElleFermeLePaquet |
| m08_killsource_sans_E | ROUGE | TestGoldenMiniBobine |
| m09_marche_des_morts_sans_E | ROUGE | TestLaMarcheDesMortsPartDeLaFinDeLaVueA |
| m10_liste_lue_comptee_localisee | ROUGE | TestFrameClosureDetailleeRendLaCarteDeFrameClosure |
| m11_cuisson_sans_E | ROUGE | TestFrameClosureDetailleeRendLaCarteDeFrameClosure, TestLaFinDeLaVueADUnFilmRecentEstLeDebutDeLaVueB, TestLaFinDeLaVueAPrimeQuandLaMarcheButeEnsuite, TestUnFilmAncienNePrendPasUneFinDeVueAFermeeAuBitSeulement, TestUnFilmAncienPrendLaFinDeSaVueAQuandElleFermeLePaquet, TestUnVraiPaquetPartDeLaFinDeSaVueA |
| m14_juge_prefixe_ferme_au_bit | ROUGE | TestUnFilmAncienNePrendPasUneFinDeVueAFermeeAuBitSeulement |
| m12_test_brut_du_bit_precedent (disque) | ROUGE | TestLocalisateurDeBoucleUnique |
| m13_vue_A_lue_hors_des_appelants_permis (disque) | ROUGE | TestLectureDeLaVueAUnique |


Règles visées : m01 E ignoré ; m02 signature reprise quand la marche depuis E bute (film ÉGALE) ;
m03 preuve de fermeture retirée pour les films PRÉFIXE ; m05 E pris sur une lecture arrêtée ; m06 E
pris malgré le bit de configuration à 0 ; m07 signature essayée avant E ; m08 killsource sans E ; m09
marche des morts sans E ; m10 liste lue comptée comme localisée ; m11 cuisson sans E ; m12 test brut
du bit précédent hors de `precedeDuTerminateur` (garde-rail sur disque, fichier temporaire retiré) ;
m13 la vue A lue hors des appelants permis (idem) ; **m14 (neuve, correction 3) le juge PRÉFIXE
affaibli en `FermeeAuBit`** (la mutation MC du contrôle). La mutation m04 de la première version (bit
nul retiré des candidats) n'a plus d'objet : c'est désormais le code. La mutation MF du contrôle
(garde « au moins un genre » retirée) n'a plus d'objet : la garde est retirée (correction 4).


## 9. Vérifications de la représentation intermédiaire (session levelup-57)

| | Vérification | Statut | Preuve |
|---|---|---|---|
| (a) | Tête donnée aux canaux (`VueA.Etat`, genres, continuation) IDENTIQUE paquet par paquet | [x] | `rangerLaTete`, `teteDe`, `listeAnnoncee` et la passe des têtes ne changent pas. Sonde `TestVAV1Tete` rejouée sur le code corrigé (20 films) : `va_v1_tete.tsv` identique à celui de V1 et de la première version (`diff` vide) |
| (b) | Vue A lue UNE fois ; étendues de `lecture.Paquet` inchangées hors genres portés ; préambule hors rangs sans changement | [x] | En cuisson, `debutDeLaVueBDeCuisson` reçoit la vue A déjà lue par `rangerLaTete` ; la preuve PRÉFIXE (`lectureDEssai`) part de E, au rang 1, sans relire la vue A. `p.VueA` rangée inchangée (sonde (a)). Les deux marches qui lisent les morts n'ont pas de structure : elles lisent la vue A une fois par paquet, par la lecture unique (`DebutDeLaVueB`, troisième appelant permis, garde-rail archlint, mutation m13 ROUGE). Préambule `DecodeFrameViewsCurseur`, `candidatsDeTete` (balayage depuis p = 0) et leurs consommateurs : non touchés (la loi du bit nul retirée, ils sont ceux de V1) |
| (c) | `DebutParVueA` dans `lecture.DebutDeVueB`, ADR 0037 IR-6 amendé ; la fin de la vue A est une LECTURE, `frame_closure_detail` ne la compte pas comme localisée | [x] | `DebutParVueA` en queue de l'énumération (valeurs existantes stables), doc « lecture, pas récupération » ; `PaquetDeCarte.ListeLue`, `ListeLocalisee` l'exclut (mutation m10 ROUGE) ; ADR 0037 : IR-3 (dernière phrase) et IR-6 (paragraphe neuf, EN ; la phrase sur les candidats de tête dit désormais que la récupération ne leur applique pas le bit nul) |
| (d) | Marche à huit vues de production : morts, véhicules, killsource inchangés sauf ce que V2 change et instruit ; `repli_localisation_largeur_libre` avant / après | [x] | Il n'y a pas de canal des morts (2.7.a non fusionné). `killsource` : aucune ligne persistée ne change (§6). Morts bipèdes (`deaths`) identiques. Véhicules (§6.1) : films PRÉFIXE, +1 mort et +89 lectures d'occupation, aucune retirée ; films ÉGALE à véhicules (étape publiée, correction 1) : `4f77afc1` +5 morts, aucune retirée, +179 lectures d'occupation et **5 retirées, instruites comme pertes de classe E** ; `bfecd02b` +1 lecture d'occupation, aucune retirée. `repli_localisation_largeur_libre` : **25 686 → 19 972** sur 20 films (il baisse là où E localise ; inchangé sur les 4 films illisibles) |
| (e) | `film_context.go` à 500 lignes | [x] | Non touché (500 lignes) ; la grammaire de vue A du contexte est lue par `fc.grammaireDeLaVueA()` (V1) |
| (f) | `consumeVueA` supprimé ; préambule gardé et écrit | [x] | Supprimé en V1, ratchet intact ; préambule inchangé (V1 §5) |
| (g) | `TestLaMarcheRangeUneVueANonPortee` et `distribuer_tetes_test.go` : genre 5 porté, raison écrite | [x] | Fait en V1 ; en V2 la doc du test renommé (`TestLaMarcheDepuisLaTeteNeTraversePasUneVueANonVide`) et de `rangerUnAvec` dit que la marche partie de la tête ne traverse pas une vue A non vide et que la vue B d'un paquet à événements part de `debutDeLaVueBDeCuisson` ; `distribuer_tetes_test.go` non touché |

Fichiers de la RI touchés à cette étape (à soumettre à levelup-57 avant fusion) :
`lecture/paquet.go` (`DebutParVueA` + docs), `marche_trames.go` (`debutDeLaVueBDeCuisson`, appel, en-tête),
`marche_trames_rangs.go` (doc), `marche_trames_test.go` (docs), `frame_closure_detail.go`
(`ListeLue`), `frame_closure_detail_test.go`, `docs/adr/0037-film-intermediate-representation.md`
(IR-3, IR-6). NON touchés : `movement_states.go` (d'où l'écart du compteur, §11), le cœur et la doc
de `distribuer.go`, `distribuer_tetes.go`, `film_context.go`.


## 10. Révisions

- `grammar.Rev` : `grammar-2026-10-06` (V1) → `grammar-2026-10-06.2` (première version) →
  **`grammar-2026-10-06.3`** (corrections du contrôle : la sortie de la cuisson change, la loi retirée ;
  rang suivant sans trou, aucune autre branche locale ne porte la valeur : `feat/ri-etape2`
  `grammar-2026-10-05`, `feat/cg2-ln` `grammar-2026-10-04`, `feat/cg2-lt` `.03.4`, `feat/cg2-ls`
  `.03.3`). Une valeur neuve plutôt que `.2` réécrite : des artefacts ont été cuits sous `.2` (pièces
  de la première version), une même révision ne doit pas nommer deux sorties. Chronique écrite.
- `killsource.Rev` : **constante** `killsource-2026-09-27` (§6, D23) ; complément de chronique mis à
  jour (`.3`), golden régénéré à révision constante.
- `objectives.Rev` : **constante** `objectives-2026-09-27` (étape `objectives` de `replay-equiv`
  identique sur les 20 films) ; complément dans `objectives/rev.go` mis à jour, golden régénéré.
- `profile.Rev` (`profile-2026-10-06`), `source.Rev`, `replay.SchemaVersion` : inchangés par le lot (`SchemaVersion` 78 à `87cdfa761`, 79 depuis la fusion de `28c542b33`, qui le monte pour son compte). Le
  contenu cuit change (états de mouvement, tir continu, véhicules) ; précédent de la campagne (vague 2,
  L3a, L4a) : `SchemaVersion` reste, la recuisson suit la montée de `grammar.Rev` (geste de
  l'utilisateur, D7).
- Régénérés par les commandes du dépôt (portes à deux verrous, `LEVELUP_UPDATE_*`) :
  `grammar_rev.golden`, `killsource_rev.golden`, `objectives_rev.golden`, `types/testdata/shapes.golden`
  (ligne des révisions), fixtures web `replay_schema_78_*` puis, après la fusion, `replay_schema_79_*`, et `manifest.json`.

## 11. Écarts

- **La loi du bit nul était un écart au plan non déclaré, et elle est retirée** (correction 2, §14).
  La première version l'appliquait aux candidats NEW de tête ; le plan la classait hors lot
  (`PLAN_LOT_VUE_A.md` l. 158-160 : « le lot n'en fait pas une règle neuve du localisateur » ;
  D-VA-6 : « hors lot ») et aucune décision du 2026-10-04 ne la couvre. Elle était la seule cause des
  3 films en baisse au gate 2 et des 3 `MANQUE` du gate de corpus. Faits établis, pour la décision à
  venir : ses 14 pertes sur les films illisibles étaient des fermetures factices au sens de la loi (bit
  à 1 devant S0 dans les 14 cas, vérifié par le contrôle) ; elle n'est lue que dans HI_1_13_0 et la
  première version l'appliquait aussi aux films illisibles HI_1_8_0, HI_1_4_1 et version-33. La
  réintroduire demande une décision datée de l'utilisateur (à quels films, et si l'exception D2 vaut
  pour ses fermetures factices retirées).
- **`movement_states.go` non touché (consigne), d'où un compteur dont le sens s'élargit.**
  `movementStateScanner.Trame` compte `EventPacketsNewRecordStart` pour tout début autre que
  `DebutParSignature` : les listes parties de la fin de la vue A y entrent désormais, alors que sa doc
  (`types.MovementStateStats`) dit « liste qui commence à un record NEW de tête ». Le compteur est
  transporté par les faits (`replay/filmfacts_*`), il n'est pas publié dans le document. Correction
  d'une ligne (exclure `DebutParVueA`, comme LN le faisait) proposée à la représentation
  intermédiaire (levelup-57), propriétaire du fichier ; non faite ici.
- **La carte v2 (instrument du gate 2) ne pose pas la carte du match** (`grammar.ContexteDeFilm`) :
  les positions à index des genres 5 et 6 y arrêtent la vue A (rien n'est deviné), là où la cuisson de
  production, qui pose la carte, la lit jusqu'au bout et part de E. Le gate 2 sous-estime donc V2 sur
  ces paquets ; `killsource json`, `replay-equiv` et le gate de corpus tournent avec la carte. Non
  mesuré séparément.
- **La première version affirmait la marche des morts « uniquement par ajout »** sans avoir mesuré
  de film ÉGALE à véhicules (le corpus de `replay-equiv` n'en contient pas) ; c'était faux sur
  `4f77afc1` (5 lectures d'occupation retirées). Corrigé et instruit (§6.1).
- **`killsource.Rev` ne monte pas**, contre l'attente de la consigne (« monte si une mort, une valeur
  ou une voie change ») : aucune ne change (§6). Aucun backlog du parc n'est à demander.
- **Gain et pertes plus grands que l'estimation du plan** (+41 656 sains contre +27 000 composés ;
  354 pertes E contre ~291) : le plan composait le crochet de LN (E seulement là où la signature
  échouait) ; V2 fait primer E partout (décision (1)) et prouve E sur les films PRÉFIXE (décision (2)).
- **Le gate de corpus rend rc = 1** (§7) : 3 `FAUX` de la règle V-3 (instruits : tous les tirs « hors
  vie » sont des tirs de véhicule couverts par un trajet du même slot dans ce véhicule ; le banc prend
  les vies dans les pistes bipèdes, qui s'arrêtent à la montée), et des lignes `[FILET]` (postures :
  fausse continuité retirée, oracle physique de R3 ; trous reclassés, instruits par famille, pas paquet
  par paquet). L'admission d'un rc 1 est un geste du pilote (plan §10 ; précédent : vague 2,
  `87cdfa761`).
- **Revue adversariale de fin de lot** (plan §8 : lot de localisation) : non faite dans cette étape ;
  à lancer par le pilote avant fusion.


## 12. Découvertes

- **D-VAV2-1** — `EventPacketsNewRecordStart` compte les listes lues jusqu'à la fin de la vue A
  (§11) ; à corriger dans `movement_states.go` par la RI.
- **D-VAV2-2** — La carte v2 ne pose pas la carte du match : un instrument qui mesure la cuisson
  devrait la poser (`-catalogue`/cartes), base et tête ensemble.
- **D-VAV2-3** — Le banc de vérité prend les vies dans les seules pistes bipèdes (`indexerVies`) :
  un joueur en véhicule n'y a pas de vie, et chaque tir de véhicule lu pendant un trajet est compté
  « hors vie » (V-3 : 101 instances déjà en base sur `4f77afc1`, 78 sur `084a804d`, toutes couvertes
  par un trajet). Plus la lecture des trajets progresse, plus V-3 monte à tort. Le banc devrait
  couvrir les vies par les trajets.
- **D-VAV2-4** — Les pertes E (354) désignent les records de tête de la vue B que le décodeur ne
  traverse pas encore : `ti=43 i19 device-position-animation-name` (139), `ti=12 i16
  managed-navpoint-override-flags` (61), arrêts en vue C (56), rejets de slot inconnu (37),
  `ti=10 i2` (19), `ti=45 i0` (19), `ti=12 i18` (7), `ti=43 i21` (6) — liste pour le lot VB-têtes. La
  marche des morts y ajoute `ti=0 i0 game-engine-team-mapping-component` (les 5 occupations retirées
  de `4f77afc1`, §6.1).
- **D-VAV2-5** — Sur les films ÉGALE, les paquets SANS liste d'événements gagnent aussi (première
  version : sains 227 749 → 250 498, +22 749), par les records NEW désormais lus en tête de liste
  (liaisons du monde).
- **D-VAV2-6** — L'étape `killsource` de `replay-equiv` hache tout le `Result`, compteurs de
  diagnostic compris (`Stats.PacketsLocated`, `Stats.Redundant`, `Replis`) : elle diverge sans
  qu'aucune ligne persistée ne change. Une étape « lignes persistées » dirait D23 directement.
- **D-VAV2-7** — 24 pertes E partaient en base d'un début S0 lui-même précédé d'un bit à 1 (loi de la
  tête contredite) : ces 24 « sains » de la base sont, au sens de la loi, des fermetures factices.
- **D-VAV2-8** — (première version) Cascade de liaison (`e5adf7b2`, slot 1096) : six sains de la base
  reposaient sur un NEW lu à un début que la loi contredit. Sans la loi, ces six paquets restent sains.
- **D-VAV2-9** — Le corpus de `replay-equiv` n'a aucun film ÉGALE à véhicules : son étape `vehicles`
  ne voit pas la fin de la vue A sur les films récents. `4f77afc1` et `bfecd02b` (gate de corpus) le
  couvriraient.
- **D-VAV2-10** — Une sonde de la marche des morts qui n'ouvre que `NewFilmContextForMap` ne mesure
  pas l'étape publiée : la cuisson pose le profil que `killsource` calibre (génération stricte,
  traversée, largeur d'axe absolue) et les largeurs MPP. Écart mesuré sur `4f77afc1` : 87 contre 74
  morts `ti=40` en base. Les sondes de cette étape le reconstruisent (`VA_PROFILS`).


## 13. Pièces

Corrections du contrôle, scratchpad `scratchpad/cg3-V2c/` (non versionné) : `carte_tete/`, `gate2.tsv`,
`gate2_perdus.txt`, `p.txt`, `table_films.md`, `ks_tete/`, `re_tete_tsv/`, `re_div_base.tsv`,
`re_div_v2.tsv`, `replis_tete.tsv`, `etapes_base/`, `etapes_tete/`, `veh/` (`diffveh.sh`),
`profils_tete/`, `profils_base/`, `corr_morts/` (contexte du contrôle), `corr_morts2/` et `corr_morts4/`
(contexte de la cuisson), `corr_morts3/` (`11de8353`), `sonde_pertes/`, `sonde_tete/`, `gate_work/`,
`gate.log`, `gate_table.txt`, `va_v2c_corpus_gate.json`, `postures_corpus.tsv`, `mut/`,
`mutations.txt`, `gfilm.txt`, `archlint.txt`, `vet.txt`, `vet_research.txt`, `lint_ci.txt`,
`lint_paquets.txt`, `lint_research.txt`, `apres.log`. Scripts : `env.sh`, `construire.sh`,
`carte.sh`, `ks.sh`, `re.sh`, `gate.sh`, `etapes.sh`, `apres.sh`, `mutations.sh`. Première version :
`scratchpad/cg3-V2/` (dont `carte_E/`, `v3.jq`, `oracle_corpus.js`, `replis.sh`). Base :
`scratchpad/cg3-V1/` (`carte_base/`, `ks_base/`, `re_base_tsv/`, `replis_base2.tsv`, binaires
`bin/base/`, `src_base/`). Contrôle : `scratchpad/cg3-V2-ctl/`. Seconde fusion (§7.2) : `scratchpad/cg3-V2c/fusion2/` (`fusion.log`, `carte_tete/`, `ks_tete/`, `re_tete_tsv/`, `re_base_tsv/`, `re_div_base.tsv`, `re_div_pre.tsv`, `gate.log`, `gate_table.txt`, `mutations.txt`), script `fusion2.sh`. Versionnés :
`va_ghidra/ASM_142f2c3b0.txt`, `ASM_14299d2c8.txt`, `FUN_1406d49c4.c`.


## 14. Corrections du contrôle indépendant (2026-10-05)

Contrôle de `8354c0d43` (parent `3bacfadeb`), pièces `scratchpad/cg3-V2-ctl/`. Quatre corrections,
toutes appliquées, aucune autre.

| # | Correction demandée | Verdict sur pièces | Action |
|---|---|---|---|
| 1 | Instruire la marche des morts sur les films ÉGALE à véhicules ; corriger §6 et §9(d) (« aucune retirée ») ; mesurer l'étape publiée sur `4f77afc1` et `bfecd02b` | **Fondée.** Le corpus de `replay-equiv` n'a aucun film ÉGALE à véhicules (vérifié : étape `vehicles` vide). Étape publiée mesurée : `4f77afc1` 0 mort retirée, 5 occupations retirées ; `bfecd02b` aucune retirée. Les nombres de la sonde du contrôle (22 morts, 75 et 8 occupations retirées) sont reproduits à l'identique dans son contexte, mais ce contexte n'est pas celui de la cuisson (profil calibré par `killsource` non posé) ; dans le contexte reconstruit, la sonde rend exactement l'étape publiée | §6.1 réécrit, §9(d) corrigé ; les 5 retraits instruits paquet par paquet comme pertes de classe E (décision (1)) ; sondes `TestVAV2ProfilCalibre`, `TestVAV2CorrMorts`, `VA_FAITS` |
| 2 | Loi du bit nul des candidats de tête : écart au plan non déclaré ; la retirer ou obtenir une décision datée ; la déclarer au §11 | **Fondée.** `PLAN_LOT_VUE_A.md` la classe hors lot ; aucune décision du 2026-10-04 ne la couvre ; aucune décision datée de l'utilisateur n'est disponible à cette étape | **Retirée** (`debut_de_liste.go` revient à V1 hors une doc ; test de la loi et vecteurs réécrits retirés). Carte v2 identique à l'octet à `carte_E` : +41 656 sains, aucun film en baisse, aucune exception D2 ; gate de corpus sans `MANQUE`. Déclarée au §11. `grammar-2026-10-06.3` |
| 3 | Test qui fige le juge de la preuve PRÉFIXE (« sans règle de l'écrivain contredite ») ; la mutation MC restait VERTE | **Fondée** (rapport du contrôle : MC VERTE sur grammar et killsource ; aucun test de la première version ne pose une marche fermée au bit près qui contredit l'écrivain sous un film PRÉFIXE) | `TestUnFilmAncienNePrendPasUneFinDeVueAFermeeAuBitSeulement` : la marche depuis E ferme au bit près en contredisant l'ordre de l'écrivain ; ÉGALE → E, PRÉFIXE → le localisateur, pas E. Mutation m14 (= MC) : ROUGE sur ce test seul |
| 4 | Garde `len(a.Genres) == 0` inatteignable dans `debutParLaVueA` (MF VERTE, mutant équivalent) : la retirer ou écrire pourquoi | **Fondée.** Relu : `marchDebut` et `runWalk` n'appellent qu'avec `BitAt(pay, 1) != 0`, la cuisson qu'avec `listeAnnoncee` ; une vue A lue depuis le bit 1 et portée compte alors au moins un genre (`lireLaVueA` lit le genre après chaque bit de continuation à 1 ; `rangerLaVueA` rend `VueTerminee` pour une vue portée) | Garde retirée ; la raison est écrite à la fonction (une vue A vide portée finirait d'ailleurs au bit 2, le début de la vue B d'un paquet sans événement) |

Vérification sur pièces le 2026-10-06, tête `30d94a60b` (après les deux fusions de `feat/v75`) : les
quatre corrections sont présentes — `debutParLaVueA` (`localisateur.go`) sans la garde des genres,
raison écrite ; `candidatsDeTete` et `localiserLaListe` (`debut_de_liste.go`) identiques à
`87cdfa761` hors une doc (la loi du bit nul ne reste qu'au localisateur, où elle était déjà en base) ;
`TestUnFilmAncienNePrendPasUneFinDeVueAFermeeAuBitSeulement` présent, m14 ROUGE sur lui seul ; §6.1,
§9(d) et §11 corrigés. Gates rejoués : §7.2.
