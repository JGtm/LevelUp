# R_COMP — chantier « comp » de la campagne de grammaire : R-L3, R-HOM, R-P3 (2026-10-02)

> Recherches préalables du plan `.ai/V7.5/PLAN_CAMPAGNE_GRAMMAIRE_2026-10-01.md` §6.1 / §6.2 (R-L3, R-HOM,
> R-P3). Worktree temporaire `LevelUp-wt-cg-comp`, HEAD détachée sur `fe18bf67c`. Aucun fichier
> suivi modifié, aucune base DuckDB, aucune cuisson, aucun commit. Films en lecture seule (19
> témoins de `config/replay_corpus.toml` + `1c4c63c2` ; `81c02726` pour le seul relevé de
> registre). Ghidra : `HaloInfinite.exe` HI_1_13_0, HTTP `127.0.0.1:8089`, lecture seule
> (`search_strings`, `get_xrefs_to`, `read_memory`, `decompile_function`, `disassemble_function`,
> `get_function_by_address`, `search_byte_patterns`) ; extraits dans
> `r_comp_tsv/r_comp_ghidra_extraits.txt`.
>
> Conventions : « mesuré » = compté par une sonde sur les films ; « estimé » = calculé sans
> marche ; « établi » = prouvé (Ghidra et/ou mesure discriminante avec témoin) ; « supposé » =
> hypothèse non discriminée. « Sains » = paquets fermés sans invariant de l'écrivain contredit
> (juge `cmJuge`, décision D2) ; le chiffre brut (fermés) est publié à côté dans les TSV.
> Pourcentages : dénominateur FIXE = par film, maximum des records utiles lus sur {marches de
> `mb_variantes.tsv`, `mb3_ti3.tsv`, `mb2_ti43.tsv`, et les marches de cette note} ; VARIABLE =
> records utiles lus par la marche elle-même. Toutes les A/B en surcouche (`-overlay`, D10) sont
> de l'outillage de MESURE, hors CI, jamais une preuve de gate.

---

## Corrections du 2026-10-02 (ajoutées après coup ; le texte d'origine ci-dessous n'est pas réécrit)

Sources : verdicts adverses du chantier comp (`VERIFICATIONS_ADVERSES_R.md`, « Chantier comp »),
`CRITIQUE_COMPLETUDE_R.md` (points 6, 9, 14, 24), `SURCOUCHE_UNIQUE.md` §4.3 et `R_COMB_2.md`.

1. **R-L3, « un bit de trop dans `ti=2 i4` » (§0 et §2.2) — NON CONFIRMÉ.** La note se contredit :
   son propre TSV (`r_comp_l3_decalages.tsv`) montre qu'un décalage de −1 à l'entrée de `i5`, `i6`,
   `i7`, `i8`, `i9` OU `i10` ferme 49 / 49 et 19 / 19, comme « `i4` −1 ». Établi : un bit de trop
   ENTRE le début de `i4` et l'entrée de `i10` ; `i4` est le candidat SUPPOSÉ (porte de
   `FUN_1407f2058`), non discriminé de `i5`..`i9`, jamais testés en lecture alternative. version-31 ne
   discrimine même pas `i0`..`i4`. Une correction posée sur `i4` fermerait les records en rendant une
   valeur fausse si le bit est ailleurs (L3b attend la discrimination).
2. Dénominateur variable : il baisse sur **5** films (et non 4). Le témoin `i15 −1` ne « gagne ≤ 36 »
   pas partout : sur `1c4c63c2`, +399 paquets bruts (229 contredits), soit +130 sains.
3. **L3a « 0 sain perdu » (critique point 9)** : faux à la lettre. Dans le contexte de cette note,
   3 924 gagnés − 316 contredits = 3 608 gains sains pour un net de +3 575 : 33 sains requalifiés
   contredits (calcul awk de la critique) ; R-COMB-2 (Live Fire en production) en compte 48
   (`1c4c63c2` 41), aucun devenu non fermé. Formule juste : « 0 film en baisse nette ; sains
   requalifiés contredits, aucun perdu non fermé ». En combinaison : +12 000 sains marginaux.
4. **R-P3** : l'oracle « archétype forcé » prouve une liaison fausse vers une UNITÉ, pas vers un
   bipède (forcer `ti=40` fait au moins aussi bien, G1 366, 0 perdu) ; « bipède » repose sur les
   déclarations d'image-clé et de bloc des mêmes slots. G4 → 0 exige la COMBINAISON « `world-object`
   au jeu + lecture par index » (chacune seule laisse G4 à 157). Le cas (a) des chroniques
   (`bf15f7ab` slot 553) ressemble à une erreur de la marche d'image-clé par voisinage. G2 « autres
   ≤ 19 » : `ti=37` tous composants fait 21.
5. **Extension de L6a au site `ti=41 i0` (§5, critique point 6)** : mesurée ici sur les deux seuls
   films Live Fire, alors que la lecture « au jeu » de `world-object-i0` est globale. BIS_4 et R-COMB-2
   la mesurent sur tout le corpus : seule, 6 films en baisse ; dans C11, −494 sains sur `1c4c63c2`.
   L'extension est RETIRÉE du lot L6a (PLAN D-110).
6. **LP** (désaveu hors bloc) en combinaison (R-COMB-2) : seul +694 sains, marginal +1 281 / +23 474
   utiles sains, gate 2 tenu ; gate 3 sans objet (killsource ne lit pas le bloc de type 1).
7. R-HOM : selon le build, `simulation-state` est à `ti=35 i59/i60` et `ti=40 i42/i43` (pas seulement
   `i60/i61` et `i43/i44`). Les sains de BIS_3 cités au §3.3 ne sont pas rejugés par le juge ici.
8. **Rejoué après J12** : `TestRCompL3ImagesCles` et `TestRCompL3Delta` rendent leurs 4 TSV à l'octet
   sous la surcouche unique ; `TestRCompP3` et `TestRCompP3LiveFire` ne sont pas rejoués (supposés
   identiques). `r_comp_overlay/` est remplacée par `surcouche_unique_postj12/`.
9. DC-9 (appel accidentel de `python3`, aucun fichier) : écart consigné au PLAN D-102 ; décision au
   PLAN §6.3 D21. Gate de la note : `archlint` rouge à `fe18bf67c` (`TestNoExpiredTODO`, hors
   chantier), tenu seulement sous `-skip` ; soldé depuis.

## 0. Verdicts

| Recherche | Statut | Résumé |
|---|---|---|
| **R-L3** | **établi (par la mesure, sous D6)** | Le bassin `i15` n'est pas en cause et le registre `ti=2` est identique sur les 21 films. Les masques « tout à un » des trois vieux builds viennent d'**un bit de trop lu dans `ti=2 i4 game-engine-current-round-component`** (lecteur HI_1_13_0 `FUN_14116fc70` = porte inversée + `R(5)`). Avec un bit de moins à `i4`, **98 records d'image-clé moteur sur 98 ferment** sur HI_1_4_1, version-31 et version-33 (0 sur 98 sans), et leurs bassins redeviennent des préfixes contigus. Portage du moteur (`i11, i13, i14, i15, i16, i17`) en A/B delta : **aucun film en baisse** (0 paquet sain perdu sur 20 films). |
| **R-HOM** | **établi** (Ghidra) | 326 noms de composants recensés (registres de 21 films). **Un seul homonyme de grammaire : `high-frequency`** (`ti=3 i1` → `FUN_142ed4880` = `R(16)+R(8)+R(2)` ; `ti=4 i0` → `FUN_14076d034` = `R(8)`). L'appartenance au `ti=3` est désormais **lue** dans la fonction d'enregistrement de l'archétype (`FUN_140e460fc`, `+0x4754 = 3`). Deux autres noms à deux tables (`simulation-state-component`, `simulation-state-playback-component`, `ti=35` et `ti=40`) ont **la même grammaire** (thunks vers la même fonction). Le Go ne confond que `high-frequency` ; effet du routage correct (mesuré par BIS_3, recalculé en sains) : +1 463 paquets sains seul, +30 235 avec `low-frequency` porté. |
| **R-P3** | **partiel** : G4 établi ; G1, G3 établis quant au mécanisme, source de la liaison fausse partiellement établie ; G2 établi comme symptôme | Aucun des quatre « coupables » n'est une largeur fausse de son composant. **G1 (694) et G3 (241)** : deltas de **bipède** lus sous une liaison fausse du slot (`ti=20` / `ti=14`) ; **G2 (658)** : un `DEL` qui est un en-tête mal lu après un décalage antérieur (92,8 % de leurs eid ne sont alloués par aucun bloc) ; **G4 (161)** : la position d'objet du monde `ti=41 i0` lue aux largeurs de la plage de la carte quel que soit l'index lu (exception `consumeObjectPositionMonde`) — **G4 157 + 3 → 0** avec la lecture du jeu par index (Live Fire, contexte de production). |

Gate R-P3 (les quatre comptes, classe « aucune allocation », hors cadre, marche de référence,
20 films) : **694 / 658 / 241 / 161** reproduits à l'identique ; meilleurs mécanismes mesurés au §4.5.

---

## 1. Matériel

Sondes (toutes dans `apps/go-api/internal/games/halo_infinite/film/internal/grammar/`, < 500 lignes,
tag en ligne 1) :

| Fichier | Tag | Objet |
|---|---|---|
| `r_comp_registres_research_test.go` | `research` | registre de chaque film (ti, i, nom, niveau) + empreintes |
| `r_comp_l3_research_test.go` | `research && campagne_overlay` | R-L3, marche delta (carte v2 + juge) sous 8 variantes du moteur |
| `r_comp_l3_kf_research_test.go` | `research && campagne_overlay` | R-L3, fermeture des records d'image-clé du moteur + recherche de décalage + hypothèses `i0..i4` |
| `r_comp_p3_research_test.go` | `research && campagne_overlay` | R-P3, diagnostic des coupables + A/B (oracles et règles de désaveu) + gate des quatre comptes |
| `r_comp_p3_livefire_research_test.go` | `research && campagne_overlay` | R-P3 G4, Live Fire, contexte de production, position d'objet du monde « au jeu » × lecture par index |
| `r_comp_p3_slot_research_test.go` | `research && campagne_overlay` | R-P3, chronique d'un slot (images-clés, bloc de type 1, records, NEW refusés) |

Surcouche : `campagne_grammaire_2026-10-01/r_comp_overlay/` = copie à l'identique des trois fichiers de
`mesures_bis2_overlay/` (base `fe18bf67c`, même tête que la campagne), `overlay.json` réécrit vers ce
worktree. Recensement Ghidra : script bash `r_comp_tsv/r_comp_hom_recension.sh` (curl, lecture seule).

Commandes (depuis `apps/go-api`, `GOCACHE` dédié, une commande `go` à la fois, `-count=1`,
plafond `filmproc` 4 Gio, films un par un, sorties dans `r_comp_tsv/`) :

```
CAMPAGNE_RACINE=<LevelUp>/data/cache/film_chunks CAMPAGNE_SORTIE=<...>/r_comp_tsv CAMPAGNE_FILMS=<20 ids>
go test -tags=research -count=1 -run '^TestRCompRegistres$' ./internal/games/halo_infinite/film/internal/grammar/      (+ 81c02726)
go test -tags=research,campagne_overlay -overlay=<r_comp_overlay/overlay.json> -count=1 -timeout 120m \
  -run '^TestRCompL3ImagesCles$|^TestRCompL3Delta$|^TestRCompP3$' ./internal/.../grammar/
# G4 : CAMPAGNE_CATALOGUE=<LevelUp>/data/titles/halo_infinite/reference/map_quant_bounds.json
#      CAMPAGNE_CARTES="0797ce72=Live Fire;60ae07c4=Live Fire - Ranked"
#      CAMPAGNE_BORNES_sgh_interlock=<plages 0, 2, 3 de mesures_bis2_tsv/mb2_regions_live_fire.tsv>  -run '^TestRCompP3LiveFire$'
# chronique : CAMPAGNE_SLOT=<film>:<slot> -run '^TestRCompP3ChroniqueDuSlot$'
```

Durées mesurées : registres 7 s ; images-clés 21 s ; delta L3 ~25 min (8 marches × 20 films) ;
P3 808 s (8 marches × 20 films) ; Live Fire 14 s ; chronique 1 à 48 s par slot. Pics mémoire ≤ 400 Mio.

---

## 2. R-L3 — bassin `i15` « tout à un » sur HI_1_4_1, version-31, version-33

### 2.1 Registre `ti=2` (méthode §6.2 (1)) — mesuré

`r_comp_registres_empreintes.tsv` (noms + niveaux, FNV-1a par archétype) : **`ti=2` = 18 entrées,
empreinte `cb77b82e` identique sur les 21 films** (HI_1_4_1, version-31, version-33, HI_1_8_0 à
HI_1_13_0, `81c02726`) ; `i14` niveau 2, `i16` niveau 4, les autres niveau 1. `ti=0` : 27 entrées
`65cae118` sur 20 films, version-31 diffère (`b95634fe`, 26 entrées : deux `forge-engine-baking-data`
de niveau 1 au lieu de trois de niveau 2). **Le registre n'explique pas la panne** (déjà dit par
T7-3, confirmé ici sur les trois builds visés).

### 2.2 L'oracle de fermeture des images-clés du moteur — mesuré, cause établie

Un record d'image-clé est un état complet et BORNÉ. Avec les six grammaires du moteur (`i11` `R(128)`,
`i13` `R(13)+n×R(1)`, `i14` niveau 2, `i15` bassin, `i16` `R(7)+R(1)`, `i17` `R(8)` ; T7 §2.6-2.7) la
fermeture est l'oracle (`r_comp_l3_images_cles.tsv`, records `ti=2`) :

| Build (films) | référence (rien de porté) | `moteur` (grammaire HI_1_13_0) | `moteur` + `i4` un bit de moins |
|---|---:|---:|---:|
| HI_1_8_0 … HI_1_13_0 (17 films) | 0 / 604 | **588 / 604** | 2 / 604 |
| HI_1_4_1 `a521164d` | 0 / 19 | **0** / 19 | **19 / 19** |
| version-31 `50247b26` | 0 / 30 | **0** / 30 | **30 / 30** |
| version-33 `a349fea8` | 0 / 49 | **0** / 49 | **49 / 49** |

Témoins décalés d'un bit après le bassin (`moteur, i15 ±1`) : **0 fermeture sur 715 records**
(`ti=0` et `ti=2`). Les 16 records non fermés des builds récents (15 sur `51ebbc0f`, 1 sur `1c4c63c2`)
ne ferment sous aucun décalage unique : non élucidés.

**Localisation du bit (recherche de décalage + hypothèses)** (`r_comp_l3_decalages.tsv`) : sur
`a349fea8` et `a521164d`, retirer un bit à l'entrée de n'importe quel composant de `i5` à `i10` ferme
49/49 et 19/19 ; à l'entrée de `i0` … `i4`, 48/49 et 18/19 seulement (un record dont une porte se lit
ailleurs). Hypothèses jouées en lecture alternative d'UN composant :

| Hypothèse (vieux builds) | `a349fea8` | `a521164d` | `50247b26` | 17 films récents |
|---|---:|---:|---:|---:|
| `i0` lu un bit plus tôt | 48/49 | 18/19 | 30/30 | 0/604 |
| `i2 current-state` en `R(2)` | 48/49 | 18/19 | 30/30 | 0/604 |
| `i3 game-finished` absent | 48/49 | 18/19 | 30/30 | 0/604 |
| `i4` un bit de moins | **49/49** | **19/19** | 30/30 | 2/604 |
| `i4` = `R(5)` sans porte | **49/49** | **19/19** | 30/30 | 2/604 |
| `i4` = porte + `R(4)` | **49/49** | **19/19** | 30/30 | 2/604 |

**Établi (mesure, D6)** : sur ces trois builds, `ti=2 i4 game-engine-current-round-component` est
écrit avec **un bit de moins** que ne le lit `FUN_14116fc70` de HI_1_13_0 (`FUN_1407f2058` : `R(1)`
porte ; si 0, `R(5)`). Les trois formes candidates sont **indiscernables** sur ces films (porte
toujours à 0 : une manche présente). Lecture Ghidra des vieux builds impossible (un seul exécutable).
Masques du bassin sous la bonne lecture : préfixes contigus 94, vide 3, autre 1 (sur 98) — le
« tout à un » de la note 3.7 était l'UNION des masques de records mal alignés.

Conséquence hors bassin : `i4` est commun à `ti=0`, `ti=1`, `ti=2` ; sur les trois vieux builds, tout
record moteur qui porte `i4` se décale d'un bit dès `i5` (composants portés compris) en production.

### 2.3 A/B delta du portage (méthode §6.2 (2)) — mesuré

`r_comp_l3_delta.tsv` (8 variantes × 20 films), `r_comp_l3_resume.tsv` (synthèse). Variante
`moteur` contre `reference`, paquets sains :

| Film | Build | Δ sains | Δ utiles sains | gagnés (contredits) | **perdus sains** | % fixe réf → moteur | % variable réf → moteur |
|---|---|---:|---:|---:|---:|---|---|
| `a521164d` | HI_1_4_1 | **0** | 0 | 1 (1) | **0** | 0,04 → 0,04 | 0,04 → 0,04 |
| `50247b26` | version-31 | **0** | 0 | 0 | **0** | 0,08 → 0,08 | 0,09 → 0,09 |
| `a349fea8` | version-33 | **+4** | +59 | 4 (0) | **0** | 0,73 → 0,74 | 0,77 → 0,76 |
| `60ae07c4` | HI_1_8_0 | +144 | +939 | 145 (1) | 0 | 28,94 → 29,27 | 30,89 → 30,84 |
| `11de8353` | HI_1_9_0 | +9 | +230 | 12 (2) | 0 | 25,58 → 25,67 | 26,27 → 26,31 |
| `084a804d` | HI_1_10_0 | +30 | +805 | 41 (11) | 0 | 15,89 → 16,03 | 16,43 → 16,43 |
| `111fa685` | HI_1_10_0 | +2 | +22 | 3 (1) | 0 | 17,10 → 17,11 | 17,35 → 17,33 |
| `1c4c63c2` | HI_1_10_0 | +147 | +323 | 424 (237) | 0 | 20,22 → 20,26 | 21,20 → 21,16 |
| `e5adf7b2` | HI_1_11_0 | +1 | +1 | 1 (0) | 0 | 26,68 → 26,68 | 27,44 → 27,44 |
| `bcb6d393` | HI_1_12_0 | +49 | +349 | 50 (1) | 0 | 25,11 → 25,36 | 28,89 → 28,93 |
| 10 films HI_1_13_0 | HI_1_13_0 | +4 à +992 | +9 à +30 735 | 3 243 (62) | 0 | cf. TSV | cf. TSV |
| **Total 20 films** | | **+3 575** | **+53 521** | **3 924 (316)** | **0** | | |

**Verdict R-L3 : aucun film en baisse** (paquets sains ; le dénominateur VARIABLE baisse de 0,01 à
0,04 point sur 4 films parce que le portage lit plus loin — effet D-42, pas une perte). Le témoin
`moteur, i15 −1` perd 518 paquets sains sur `bf15f7ab` et gagne ≤ 36 ailleurs (hasard) : la
grammaire `i15` est discriminée. Sur les trois vieux builds, `i4` corrigé ne change pas la marche
delta (+0) : ces films ferment < 1 % pour d'autres causes (vue C hors cadre), l'effet de `i4` y est
masqué. `1c4c63c2` : 237 des 424 gains sont contredits (factices) ; sains +147.

---

## 3. R-HOM — composants homonymes

### 3.1 Méthode — Ghidra, lecture seule

1. Union des noms de composant des registres des 21 films : **326 noms** (`r_comp_registres.tsv`).
2. Par nom : chaîne ASCII exacte (`search_strings ^nom$`) → xref DATA (accesseur de nom,
   `vtable[0x08]`) → xrefs DATA vers l'accesseur = **tables** (table = `vtable + 0x08`) → lecteur =
   `vtable[0x28]`, ou `[0x30]` si c'est le thunk `FUN_14076ce9c` (`r_comp_hom_tables_binaire.tsv`).
3. **Archétype de chaque table, lu dans le jeu** : l'objet de composant statique (`.data`) porte la
   vtable ; il est enregistré par `FUN_14064dd28(archétype + 8, index, &objet, ...)` dans une fonction
   d'enregistrement qui pose `*(archétype + 0x4754) = ti` (54 fonctions, 299 appels). Sur les 147
   lignes à `ti` constant résolues, le nom de la table enregistrée en `(ti, index)` **concorde avec le
   registre du film** `0797ce72` 147 fois sur 147 (2 index au-delà du registre : enregistrements
   conditionnels compactés, `FUN_14064ce0c` passe un drapeau d'activation par composant)
   (`r_comp_hom_enregistrement.tsv`, `r_comp_hom_enregistrement_ti.tsv`).

### 3.2 Recensement — établi

| Catégorie | Noms | Détail |
|---|---:|---|
| une table | 314 | dont 2 où `vtable[0x28]` est le lecteur direct (`*-dynamic-precision`) |
| une table par tableau de noms | 8 | `managed-navpoint-visual-state-groups-component-0..7` : accesseur indexé `0x14064c620` (`names[this+8]`, tableau `0x143d07f00`), table unique `0x143d081c8`, lecteur `0x140dbe1bc` |
| absent de HI_1_13_0 | 1 | `equipment-charges-used-component` (`ti=37 i27`, version-31 seulement ; HI_1_13_0 porte `equipment-charges-remaining-component`) |
| **deux tables** | **3** | ci-dessous |

| Nom | `(ti, i)` | Table | Lecteur | Grammaire | Le Go |
|---|---|---|---|---|---|
| `high-frequency` | `ti=3 i1` | `0x143d07af0` | `FUN_142ed4880` | `R(16)+R(8)+R(2)` = 26 bits | **CONFOND** : lit `R(8)` (`dispatch_item.go:122`, routage par nom) |
| `high-frequency` | `ti=4 i0` | `0x143d06a60` | `FUN_14076d034` | `R(8)` | juste |
| `simulation-state-component` | `ti=35 i60` / `ti=40 i43` | `0x143d0c938` / `0x143d0b358` | thunks `0x142f02434` / `0x142f02444` | les deux sautent à `FUN_142ed6d88` (champ `+0xa48` / `+0x850`) : **même grammaire** | juste (une lecture par nom) |
| `simulation-state-playback-component` | `ti=35 i61` / `ti=40 i44` | `0x143d0c988` / `0x143d0b268` | thunks `0x142f02454` / `0x142f02464` | les deux sautent à `FUN_142ed6d20` : **même grammaire** | juste |

Archétypes lus : `FUN_140e460fc` (`+0x4754 = 3`) enregistre `i0` = `low-frequency` (table
`0x143d07b40`, `FUN_142ed4aec`) et `i1` = `high-frequency` table `0x143d07af0` ; `FUN_140e462d8`
(`+0x4754 = 4`) enregistre `i0` = `high-frequency` table `0x143d06a60` ; `FUN_14064c7d8` (`0x23` = 35)
enregistre `simulation-state*` en 60/61 ; `FUN_14064ce0c` (`0x28` = 40) en 43/44. **L'appartenance
de la table 26 bits à `ti=3`, « déduite » au plan §6.2 L8, est établie.**

### 3.3 Effet mesuré d'un routage correct

Mesures de BIS_3 (`mesures_bis3_tsv/mb3_ti3.tsv`, surcouche, 20 films), recalculées en sains
(fermés − fermés contredits) ; aucune nouvelle marche n'était nécessaire, le crochet `b3CrochetTi3`
route déjà par archétype (`typeIndex == 3`) :

| Variante | Fermés | Sains | Δ sains | Gagnés / perdus | Gagnés contredits |
|---|---:|---:|---:|---:|---:|
| référence | 284 704 | 276 316 | — | — | — |
| routage seul (`ti3-i1-26`) | 286 155 | 277 779 | **+1 463** | +1 462 / −11 | 9 |
| `low-frequency` seul (`ti3-i0`) | 285 130 | 276 677 | +361 | +836 / −410 | 64 |
| les deux (`ti3-i0+i1-26`) | 315 312 | 306 912 | **+30 596** (+30 235 sur `ti3-i0`) | +30 618 / −10 | 34 |
| témoin `ti=4 i0` lu sur 26 bits | 34 399 | 12 707 | −263 609 | +6 065 / −256 370 | 4 628 |

Par film (sains, `les deux`) : `fb1a1a72` 22 279 → 42 038, `51ebbc0f` 9 759 → 19 782, `c75f33b8`
22 855 → 23 670 ; `1c4c63c2` 13 540 → 13 539 (**−1 sain**, dû au portage de `i0`, pas au routage :
même valeur sous `ti3-i0`) ; aucun autre film ne bouge en sains.

---

## 4. R-P3 — coupables résiduels des décalages (D-56)

### 4.1 Gate reproduit — mesuré

`r_comp_p3.tsv`, marche de référence, table `dernier_composant_x_classe` de `b3Diag`, classe
« aucune allocation », colonne hors cadre : **G1 `DELTA ti=20 … i1 spawn-filter-weight` 694**
(`111fa685` 320, `bf15f7ab` 172, `11de8353` 76, `1c4c63c2` 68, `e5adf7b2` 58) ; **G2 `DEL ti=0 …`
658** (19 films) ; **G3 `DELTA ti=14 … i1 crew-marked-objects` 241** (`11de8353`) ; **G4 `DELTA
ti=41 … i2 object-forward-and-up` 161** (`0797ce72` 158, `60ae07c4` 3). Identiques à MESURES_BIS_3.

### 4.2 Les lecteurs eux-mêmes sont justes — établi (Ghidra)

- `ti=20 i1` : `FUN_142ed70b8` → `FUN_1406d84b4(…, n = 0x10, 1, 0)`, qui consomme exactement `n`
  bits (désassemblage `142ed70df MOV [RSP+0x20],0x10`) : `R(16)`, comme le Go.
- `ti=20 i0` : `FUN_142ecf744` : `R(2)` cas ; 1 → niveau < 2 `FUN_1407f2058` (`R(1)`[0 → `R(5)`]),
  sinon `FUN_142b67e34` = `R(9)` ; **2 → `FUN_142b67f08` (`R(1)`[0 → `R(13)`]) puis `R(6)`** ;
  3 → `R(32)` + `FUN_14076e494(0x10)` + `R(3)` + `R(4)`·n×`R(32)` + `FUN_1407f1ff4` (`R(1)`[0 →
  `R(5)`]). Le Go lit le cas 2 par `readQuantStat(1)` (sonde, 13 ou 9 bits, PLUS deux bits de
  queue) : **écart réel** (découverte DC-4), mais **sans rôle dans G1** : les 742 records `ti=20`
  coupables portent tous `i0` cas 0 ou `i0` absent ; A/B « cas 2 du jeu » : gate identique, +1/−1
  paquet (contredits).
- `ti=14 i1` : `FUN_142ed421c` = `R(1)` ; si 1 → `FUN_1406d3140(R8 = 7)` (identifiant) : le Go
  désynchronise proprement sur 1. Les records G3 lisent `i1` sur 1 bit (porte 0) : conforme.
- `ti=41 i2` : `FUN_14076e278` → `FUN_140c5f938(p4 = 0)` → `FUN_140c5fa84` + `FUN_140c5f9c8`
  (porte, `R(19)`, `R(8)`) sauf si `DAT_145121140 == 1` (variante moteur, D-32). Les records G4 lisent
  28 ou 9 bits : conforme à la branche normale.

### 4.3 Le mécanisme — établi (mesure discriminante)

**Masques impossibles.** Les 742 records `ti=20` et 275 records `ti=14` coupables ont **tous** un
masque portant des bits au-delà des 3 composants de l'archétype (`0x2200003` = {0, 1, 21, 25},
`0x2000003`, `0x102200003`, …) — `r_comp_p3_tables.tsv` tables `g1_ti20`, `g3_ti14`. {0, 1, 21, 25}
est le masque type d'une UNITÉ (`ti=35` : position, vitesse, `unit-desired-aiming-vector`,
`unit-command-tick`). Le traverseur Go ignore en silence les bits de masque au-delà du dernier
composant de l'archétype (découverte DC-1) : il lit `i0 + i1` (18 bits) et rend la main au milieu
d'un record de bipède.

**Oracle « archétype forcé »** (on lie, avant le paquet, l'eid de chaque record à masque impossible
à `ti=35`, resp. `ti=40`) : G1 694 → 415 (resp. 366), **G3 241 → 0**, G2 658 → 637 (634) ;
+853 / −30 paquets (sains perdus 28, tous sur `51ebbc0f` en `ti=35`) ; `bf15f7ab` +597 paquets
(G1 172 → 0), `11de8353` +175 sains (G3 → 0), `1c4c63c2` +72 (G1 68 → 0). `111fa685` ne se répare
pas (320 → 300) : ses deltas de bipède échouent plus loin même sous `ti=35` (film à 17 % de fermeture).

**Établi** : G1 et G3 sont des deltas de bipède lus sous une liaison fausse du slot, pas une largeur
de `spawn-filter-weight` ni de `crew-marked-objects`.

### 4.4 La source de la liaison fausse — chroniques de slot (mesuré)

Le diagnostic compte 1 017 records coupables (742 `ti=20` + 275 `ti=14`, un par paquet ; le gate de
`b3Diag` les compte par premier rejet d'eid, d'où 694 + 241). Huit slots en portent 1 015
(`g1_g3_slot`) : `111fa685` 554 (319), `11de8353` 688 (273), `bf15f7ab` 553 (173), `11de8353` 610
(94) et 611 (11), `1c4c63c2` 562 (77) et 591 (10), `e5adf7b2` 553 (58) — tous dans la plage des
bipèdes joueurs (550-700) ; les 2 autres sont deux slots `ti=14` à 1 record. Chroniques
`r_comp_p3_chronique_<film>_<slot>.tsv` :

1. **Déclaration d'image-clé `ti=20` sur un slot que le bloc de type 1 du MÊME chunk dit non vivant**
   (`bf15f7ab` 553 chunk 14 ; `11de8353` 610 chunk 18 ; `e5adf7b2` 553) : bloc « gen 0, vivante
   false, masque vide » ; l'image-clé du chunk suivant déclare `ti=35` tête 1, bloc vivant gen 1,
   pont `ti=35`. L'écrivain ne déclare que des entités allouées : la déclaration est fausse ou
   périmée. **A/B « désaveu image-clé hors bloc »** (retirer à l'ouverture du chunk une liaison
   d'image-clé dont le bloc du même chunk dit le slot non vivant ; 2 342 liaisons retirées sur 20
   films) : **G1 694 → 388**, G2 658 → 650, +699 / −8 paquets, **0 sain perdu**, **+694 sains**,
   +7 800 utiles sains (`bf15f7ab` +606 sains, 93,40 % → 95,85 % sur dénominateur fixe, 94,44 % →
   96,32 % variable ; `11de8353` +54 ; `e5adf7b2` +33 ; `1c4c63c2` +1).
2. **Déclaration d'image-clé `ti=20` sur un slot vivant** (`111fa685` 554 chunk 9, `1c4c63c2` 562
   chunk 14, et l'anticipation qui la propage au chunk précédent, `111fa685` chunk 8) : dans ces
   images-clés, **plusieurs slots consécutifs du pool des bipèdes** sont déclarés `ti=20` (554-557 ;
   559-563) avec des records de 344 à 1 400 bits, alors que l'image-clé suivante (et les précédentes
   sur `1c4c63c2`) les déclare `ti=35` (≈ 2 600 bits) sous la même génération. **Non discriminé** :
   marche d'image-clé qui lit faux ces records (ancre ou `TI` lus à tort), ou slots réellement
   réutilisés par des filtres de réapparition à l'instant de l'image-clé puis recréés en bipède à
   la même génération (NEW non lu). Reste G1 = 388 (320 + 68).
3. **NEW à masque impossible** (`11de8353` 688 chunk 29, paquet 92 : `NEW gen 1 ti=14 masque
   0x8000000`, paquet hors cadre) : `BindFull` pose `ti=14` ; le vrai NEW du bipède (paquet 320,
   `ti=35`, paquet fermé) est ensuite **REFUSÉ** par la marche parce que le slot est « vivant »
   (`neufsRefuses`, `contreditUneEntiteVivante`, lot L7 ; D-13 : le jeu crée sur une entrée occupée,
   `FUN_1408f18d0`). A/B « désaveu NEW à masque impossible » : G3 → 0 mais −522 paquets (257 sains)
   ailleurs : **règle rejetée** ; la règle propre est celle de L7 (accepter le NEW sur slot occupé),
   non mesurée ici.

**G2 = symptôme** (`g2_del`, `g2_del_premier_non_del_avant`) : 635 des 684 `DEL` coupables (92,8 %)
visent un eid qu'aucun bloc n'alloue → l'en-tête `DEL` est lui-même mal lu. Le record lu avant la
chaîne de `DEL` : `DELTA ti=40` 326 (tous sur Live Fire `0797ce72` 181 et `60ae07c4` 145 = T4-C3,
L6a), `DELTA ti=20` à masque impossible 76 (= G1), `DELTA ti=21 i16 flock-position` 66 (= L6b),
`NEW ti=41` 33, `DELTA ti=35` 24, autres ≤ 19. Sous lecture par index (Live Fire, contexte de
production, §4.5) : G2 `0797ce72` 242 → 31, `60ae07c4` 173 → 26.

**G4 — établi** : `ti=41 i0 object-position-component` = `FUN_14076e29c` → `FUN_14076e420(0x10)` →
`FUN_14076e524`, qui lit l'index de plage puis les largeurs de LA LIGNE DE L'INDEX LU (preuve de L6a,
T4-C3) ; le Go lit cette position par l'exception datée `consumeObjectPositionMonde`
(`lecteur_position_exceptions.go:35`), qui lit l'index puis les largeurs de la plage de la carte
quel que soit l'index. L'A/B « lecture par index » de BIS_3 ne passait pas par ce site : G4 y restait
157. Live Fire, contexte de production (`r_comp_p3_livefire.tsv`) :

| Film | Variante | Fermés | Sains | Perdus sains | G2 | **G4** |
|---|---|---:|---:|---:|---:|---:|
| `0797ce72` | production | 19 124 | 19 088 | — | 242 | **157** |
| | + lecture par index | 22 390 | 22 368 | 1 | 43 | 157 |
| | + world-object au jeu | 19 125 | 19 088 | 0 | 243 | 157 |
| | + world-object au jeu + lecture par index | 22 884 | **22 863** | 1 | 31 | **0** |
| `60ae07c4` | production | 13 965 | 13 828 | — | 173 | **3** |
| | + lecture par index | 15 722 | 15 618 | 33 | 26 | 3 |
| | + world-object au jeu + lecture par index | 15 688 | 15 590 | **63** | 26 | **0** |

Sur `0797ce72`, la position d'objet du monde lue par index ajoute **+495 sains** à la lecture par
index seule ; sur `60ae07c4` elle en **retire 28** (déjà vu par D-55 : le site `world-object-i0` au
jeu perd sur ce film).

### 4.5 Gate R-P3 — les quatre comptes

| Mécanisme mesuré | G1 | G2 | G3 | G4 | Δ sains (20 films) | Perdus sains | Nature |
|---|---:|---:|---:|---:|---:|---:|---|
| référence | 694 | 658 | 241 | 161 | — | — | — |
| `ti=20 i0` cas 2 du jeu | 694 | 658 | 241 | 161 | 0 | 0 | lecteur (nul) |
| oracle du pont du bloc | 694 | 658 | 241 | 161 | 0 | 0 | oracle (le pont est vide ou ambigu sur ces slots) |
| oracle archétype forcé `ti=35` | 415 | 637 | **0** | 161 | +823 | 28 | oracle |
| oracle archétype forcé `ti=40` | 366 | 634 | **0** | 161 | +838 | 0 | oracle |
| **désaveu image-clé hors bloc** | **388** | 650 | 241 | 161 | **+694** | **0** | règle réaliste |
| désaveu NEW à masque impossible | 694 | 651 | **0** | 161 | +282 | **257** | règle rejetée |
| désaveu des deux | 388 | 643 | 0 | 161 | +976 | 257 | — |
| Live Fire : world-object au jeu + index (2 films, contexte de production) | — | −358 | — | **−160 → 0** | +5 537 contre « production » (`0797ce72` +3 775, `60ae07c4` +1 762) ; contre « lecture par index » seule : +495 et −28 | 64 | lecteur (L6a étendu) |

Restes non attribués après ces mesures : G1 388 (`111fa685` 320, `1c4c63c2` 68 : §4.4 point 2) ; G2 :
la part qui suit un `DELTA ti=40` hors Live Fire et les queues ≤ 33 ; G3 et G4 : 0 sous leurs
mécanismes.

---

## 5. Impact sur les lots

- **L3** : R-L3 rend « aucun film en baisse » pour le portage HI_1_13_0 du moteur (gate par film
  tenu sur 20 films, sains). Ajouter au lot, sous D6 : `i4 current-round` un bit plus court sur
  HI_1_4_1, version-31, version-33 (98/98 images-clés), avec une condition **mesurable par film**
  (la branche sur le build est interdite) — candidat : la fermeture des records d'image-clé du
  moteur sous chaque forme, décidée au premier record borné du film. Vecteurs : V15a/V15b/V16/V17 de
  T7 §6.2 ; témoin `i15 ±1` (0/715).
- **L8** : prérequis R-HOM levé ; « grammaire de `i1` probable » devient **établie** (table lue dans
  l'enregistrement de `ti=3`). Le routage se fait par ARCHÉTYPE (ou par table), pas par nom.
  `ecs_table.tsv` `ti=3 i1` porte `FUN_14076d034` : faux (DC-2).
- **L6a** : étendre le lot au site `ti=41 i0` (`consumeObjectPositionMonde`) : sans lui, G4 reste 157
  sur `0797ce72` ; avec lui +495 sains de plus sur ce film, −28 sur `60ae07c4` (à juger au gate par
  film, avec D4).
- **L9 / L1 / L7** (marche) : G1 et G3 sont des liaisons fausses de slots de bipède. La règle
  « désaveu d'une déclaration d'image-clé que le bloc de type 1 du même chunk dit non vivante »
  (+694 sains, 0 perdu) demande la lecture du bloc de type 1 en production (comme L1, gate 4,
  D11) ; le reste (§4.4 point 2) et le NEW refusé sur slot occupé (§4.4 point 3, L7) relèvent de la
  marche d'image-clé et de L7. Aucun de ces gains n'est inclus dans les bornes de L1/L7/L9 publiées.
- **L0** : la carte devrait classer « masque au-delà de l'archétype » comme invariant violé à la
  LECTURE (DC-1) ; le libellé `DEL ti=0` de `dernier_composant_x_classe` désigne un en-tête mal lu,
  pas un `DEL` (92,8 %).

---

## 6. Découvertes (consignées, non traitées)

- **DC-1** `traverseComponentLoop` ignore en silence les bits de masque au-delà du dernier composant
  de l'archétype : un masque impossible (l'écrivain ne pose que des bits < nombre de composants) ne
  désynchronise pas le record. 1 017 records coupables G1/G3 en portent un.
- **DC-2** `ecs_table.tsv` : `ti=3 i1 high-frequency` porte `deser_addr = FUN_14076d034` (c'est le
  lecteur de `ti=4 i0`) ; le bon est `FUN_142ed4880`. La ligne `ti=4 i0` dit « KEYFRAME : 26 bits
  (distinct) » : la table statique de `ti=4` n'a qu'un lecteur `R(8)` ; les 26 bits sont ceux de
  `ti=3 i1` (à revoir).
- **DC-3** `equipment-charges-used-component` (`ti=37 i27`, version-31) n'existe pas dans
  HI_1_13_0 ni dans le dispatch Go (renommé `-remaining`) : tout record `ti=37` qui le porte
  désynchronise sur version-31.
- **DC-4** `spawn-filter-type` cas 2 : le Go lit `readQuantStat(1)` (sonde + 13/9 bits + 2 bits de
  queue) puis `R(6)` ; le jeu lit `R(1)` [0 → `R(13)`] puis `R(6)` (`FUN_142b6ee08` →
  `FUN_142b67f08`). Mesuré presque jamais exercé (+1/−1 paquet).
- **DC-5** La fonction d'enregistrement statique des archétypes (`FUN_14064dd28`, `+0x4754 = ti`,
  54 fonctions) donne la table de chaque `(ti, index)` : un garde-fou « dispatch Go par table » est
  constructible hors ligne (147/147 concordances). Index compactés quand un enregistrement
  conditionnel est inactif (`ti=0` forge, `ti=41 i21/i22`, `ti=23 i33`).
- **DC-6** Images-clés moteur non fermées sous la grammaire complète : 15/41 sur `51ebbc0f`, 1/68 sur
  `1c4c63c2`, aucun décalage unique ne les ferme.
- **DC-7** Les compteurs de masques de bassin relevés dans la marche delta incluent les lectures
  d'essai des localisateurs (`debutDeLaListe`, `marchLocateStrict`) : un crochet de composant voit
  aussi les marches d'essai ; toute mesure par crochet doit en tenir compte (non publiées ici).
- **DC-8** Gate `go test ./internal/archlint/` : `TestNoExpiredTODO` est ROUGE par le calendrier
  (`internal/api/handlers/json_huma_coverage_test.go:34`, `TODO(expiry:2026-10-01)`, échu au
  2026-10-02), fichier hors campagne ; tous les autres tests d'`archlint` sont verts.
- **DC-9** Un appel accidentel `python3 -` à script VIDE a été lancé pendant une édition (aucun
  code Python écrit ni exécuté, aucun fichier produit) ; signalé pour transparence.

---

## 7. Gate (depuis `apps/go-api` du worktree)

| Commande | Résultat |
|---|---|
| `gofmt -l internal/games/halo_infinite/film/internal/grammar/r_comp_*` | vide |
| `go vet -tags=research ./internal/games/halo_infinite/film/internal/grammar/` | ok |
| `go vet -tags=research,campagne_overlay -overlay=<r_comp_overlay/overlay.json> ./internal/games/halo_infinite/film/internal/grammar/` | ok |
| `go test -count=1 ./internal/archlint/` | **FAIL** : seul `TestNoExpiredTODO` (DC-8, échéance calendaire dans un fichier hors campagne) |
| `go test -count=1 -skip '^TestNoExpiredTODO$' ./internal/archlint/` | ok |
| sondes : `TestRCompRegistres`, `TestRCompL3ImagesCles`, `TestRCompL3Delta`, `TestRCompP3`, `TestRCompP3LiveFire`, `TestRCompP3ChroniqueDuSlot` | ok |

`grammar.Rev` inchangé (`grammar-2026-09-27.3`) : aucun fichier de production touché.

## 8. TSV (`r_comp_tsv/`)

`r_comp_registres.tsv`, `r_comp_registres_empreintes.tsv` ; `r_comp_l3_images_cles.tsv`,
`r_comp_l3_decalages.tsv`, `r_comp_l3_delta.tsv`, `r_comp_l3_delta_tables.tsv`,
`r_comp_l3_resume.tsv` ; `r_comp_hom_tables_binaire.tsv`, `r_comp_hom_enregistrement.tsv`,
`r_comp_hom_enregistrement_ti.tsv`, `r_comp_hom_composants.tsv`, `r_comp_hom_recension.sh` ;
`r_comp_p3.tsv`, `r_comp_p3_tables.tsv`, `r_comp_p3_livefire.tsv`, `r_comp_p3_livefire_tables.tsv`,
`r_comp_p3_chronique_<film>_<slot>.tsv` (6) ; `r_comp_ghidra_extraits.txt`.
