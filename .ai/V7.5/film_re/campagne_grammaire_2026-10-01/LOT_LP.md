# Lot LP — image-clé jugée par le bloc de type 1 (2026-10-04)

> Plan : `.ai/PLAN_CAMPAGNE_GRAMMAIRE_2026-10-01.md` §6.1 / §6.2 « LP », reprise de la variante V3e
> de L9 (`LOT_L9.md` §4-§8, D-L9-3). Vague 2, GO de l'utilisateur du 2026-10-04. Worktree
> `LevelUp-wt-cg2-lp`, branche `feat/cg2-lp`, base `2393d7db7` (vague 1 fusionnée, `grammar-2026-10-03.2`,
> `killsource-2026-09-27`). Films lus en place (lecture seule) depuis `data/cache/film_chunks` du
> checkout principal ; 20 films (19 témoins de `config/replay_corpus.toml` + `1c4c63c2`). Ghidra en
> lecture seule (HTTP direct 127.0.0.1:8089). Aucune cuisson en lot, aucune base, aucun push.
>
> Convention : **lu** = lu dans Ghidra ; **mesuré** = compté sur les films ; **estimé** = déduit
> par un raisonnement écrit ; « sain » = paquet fermé au sens de L0 (reste nul, aucune règle de
> l'écrivain contredite). `scratchpad/` = `C:/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp/f46f71fc-4042-4343-b5c8-623c9fca58c2/scratchpad/v2-LP/`
> (non versionné : TSV, binaires, surcouche, patch, décompilations).

## 0. Statut : [!] NON RETENU (gate 2)

Le lot porté — **la marche d'image-clé de toutes les générations sous le témoin du bloc de type 1**
(règle 1, §2) — donne **+22 635 paquets sains, +475 733 records utiles sains** sur 20 films, part de
gains factices 1,1 %, fermetures factices du corpus 9 229 → 5 751 (mesuré, §3). Il **échoue au
gate 2 sur deux films** : `a349fea8` −1 sain (0 utile), `c75f33b8` −3 sains / −18 utiles. Les quatre
paquets sont instruits un par un (§4) : chacun n'était « sain » qu'en ouvrant sa liste d'événements
sur un NEW de tête que les blocs de type 1 du chunk et du chunk suivant contredisent, admis par une
bande d'archétype élargie par des déclarations d'image-clé que le bloc dit non allouées. Ce ne sont
pas des fermetures factices au sens du juge de L0 (aucune de ses règles n'est contredite) :
l'exception D2 ne s'applique pas à la lettre (plan §6.0 point 2, « le juge est celui de L0 »).
Décision demandée (D-LP-1, §8).

Le **désaveu des déclarations que le bloc dit non vivantes** (règle 2 de l'énoncé de LP, R-P3) est
**réfuté** : lu dans le jeu, le chargement d'image-clé restaure le vecteur d'entités EN ENTIER et la
garde du décodeur lit ce vecteur par slot ; mesuré, désavouer les 2 400 entrées « allouées non
vivantes » (`0x7`) coûte −6 152 sains et met quatre films de plus en baisse (§1.3, §3.2). Ce que
R-P3 mesurait sous ce nom (`bf15f7ab` slot 553) est couvert par la règle 1 (§3.3).

Commit : ce document et la sonde `lp_temoin_research_test.go` (tag `research`) seulement. Aucun
fichier de production ne change ; `grammar.Rev` et `killsource.Rev` inchangées. Le code du lot est
gardé hors dépôt (§6).

## 1. Ce qui est lu dans le jeu

### 1.1 Le chargement d'une image-clé (lu ; décompilations `scratchpad/ghidra/` et `../L9/ghidra/`)

| Fonction | Ce qu'elle fait |
|---|---|
| `FUN_1428e2a9c` | lit le bloc de type 1 (`FUN_1429883ec`) PUIS l'image-clé (`FUN_142e2bfd0`). |
| `FUN_142e2aab4` | restaure la table de datums (`FUN_142f22be8(monde+0x120)`) et le vecteur d'entités (`FUN_141f850dc(monde+0x42f0)`), appelle `FUN_142e2de40` sur chaque entrée de 200 octets, puis `FUN_1408f1730`. |
| `FUN_141f850dc` | copie le vecteur restauré entrée `i` → entrée `i` (eid en `+0`, archétype en `+4`, …), redimensionné à `max(0x1fff, n)` : TOUTES les entrées. |
| `FUN_1408f1730` | parcourt la table de datums ; pour chaque entrée `(f&1)&&(f&4)&&!(f&2)&&!(f&0x20)`, `eid = gen<<30 \| slot`, archétype lu en `slot*200+4` du vecteur `*monde+0x20`, puis `FUN_142f2e534(vue, eid)` : inscrit l'entité dans la vue (rang 1). |
| `FUN_1406cbaa0` | garde de la branche vive du décodeur : `*(uint*)(slot*200 + *(*monde)+0x20) != eid` → corps non lu. Le vecteur est indexé PAR SLOT. |
| `FUN_142e32b24` | (NEW rejoué) écrit l'entrée `slot*200` du même vecteur et y pose `+8 \|= 1`. |

### 1.2 Ce qui rend une entrée de datum allouée, vivante ou libérée (lu)

| Fonction | Effet sur l'octet de drapeaux `+0x00` |
|---|---|
| `FUN_1408f1618` (allocation) | `= 1`, génération `+0x01 = eid>>30`, `+0x04 = 1`, `+0x08..+0x18 = 0`. |
| `FUN_142f2e598` (allocateur) | `gen = (gen+1)&3`, `eid = gen<<30 \| slot` : génération 0 = 4e allocation ; le seul identifiant nul est `0xffffffff`. |
| `FUN_1408f12c4` (libération immédiate) | `\|= 2`, masque par vue `+0x08`/`+0x10` mis à 0, `FUN_140bbcf34`, puis saut à `FUN_1408f16c4`. Appelée par `FUN_1408f1244`, `FUN_1408f18d0` (NEW sur slot occupé) et `FUN_142f2f73c` cas 2 (DEL rejoué). |
| `FUN_141fdab4c` (libération multi-vues) | `\|= 2`, notifie chaque vue (`FUN_142f2f320`), et n'appelle `FUN_1408f16c4` que si le masque par vue `+0x08` est nul. |
| `FUN_142f2df60` (une vue relâche) | si le masque par vue est nul : `FUN_1408f16c4` ; sinon `\|= 8`. |
| `FUN_1408f16c4` (solde) | `&= 0xd8` : retire 1 (alloué), 2 (libéré), 4 (publié), 0x20 (écarté). |

Conséquences (estimé à partir de ces lectures) : une entrée `0x7` est une entité libérée qu'au moins
une vue tient encore ; une entrée à drapeaux 0 est soldée (génération gardée) ou jamais allouée. Une
entrée d'image-clé est écrite par une vue (`FUN_142f2e174`, une entrée par entité de SA table,
identifiant pris de la table de datums) : au bloc du même instant, son slot est ALLOUÉ (drapeau 1)
SOUS SA GÉNÉRATION. Le bit 4 (« publié ») n'a pas été localisé chez son écrivain (non lu).

### 1.3 Ce qui en découle pour les deux règles de l'énoncé

- **Règle 1 (liaison des générations, V3e)** : un en-tête n'est l'entrée d'un slot que si le bloc
  qui précède l'image-clé porte ce slot alloué sous la même génération. Toutes les générations sont
  alors des ancres ; la génération ne décide plus ni le voisin, ni le recalage, ni le saut, ni
  l'élection. **Fondée** sur §1.1-§1.2. Écart à V3e : V3e exigeait « drapeaux ≠ 0 », le lot exige le
  drapeau 1 (`FUN_1408f1618` / `FUN_1408f16c4`) ; carte v2 identique à l'octet (§3.2).
- **Règle 2 (désaveu des non-vivantes)** : le prédicat de `FUN_1408f1730` choisit les entités que la
  vue recrée ; il ne retire rien du vecteur que le décodeur lit (`FUN_141f850dc`, `FUN_1406cbaa0`).
  **Réfutée** pour la marche de trames, et mesurée nuisible (§3.2).

## 2. Ce qui a été écrit (code hors dépôt, §6)

- `grammar/keyframe_temoin.go` (neuf) : lecture des seuls drapeaux et générations du bloc
  (`lireTemoinDeDatums`, cardinal et fermeture de `LireBlocDeDatums`), index payload d'image-clé →
  bloc qui le précède dans son chunk (`blocsDesImagesCles`, `blocsDuChunk`), règle 1 (`admet`,
  `ancre`, `valide`, `generationDecidee`, `cleDeGeneration`), témoin lu une fois par payload dans la
  mémoire des marches du contexte, `MarcheDImageCle.TableDeDatums` (table à position libre sous le
  témoin).
- `grammar/keyframe_world.go` : `kfEnTeteDAncre` (gardes sans génération) ; `kfAnchorFromID` garde la
  génération 0 refusée pour la marche SANS témoin ; `motifDAncre(prev, toutesGenerations)` ;
  `suivante` passe par `r.ancre` / `generationDecidee` / `cleDeGeneration` ; commentaires de contrat
  (D-L9-8 soldée dans le patch).
- `grammar/keyframe_world_marche.go` : la mémoire porte blocs et témoins ; `marcherLaTable` passe par
  `r.valide` ; saut de largeur ouvert à toute génération admise. `MarcheDImageCle.Records` et
  `KeyframeRec` (dont `Elue`) gardent leur forme.
- `grammar/keyframe_world_preuve.go` : la mémoire est construite avec l'index des blocs ; clé de
  génération de `meilleurCandidat`.
- `grammar/keyframe_datums.go` : `candidatsDeDatumSous`, `tableSousTemoin`, `LierTableDeDatums` sous
  le témoin du bloc de son chunk.
- `grammar/type1_datums.go` : `cardinalDuBloc` (dérivation du cardinal, partagée), `datumVivant`.
- Repli nommé et compté `repli_image_cle_sans_bloc_de_datums` (`grammar/replis_du_film.go`,
  `facts/fallback/noms.go`, `facts/fallback/registre_filmdec_marche.go`,
  `replay/versement_des_replis.go`) : sans bloc lisible, la marche d'avant. Mesuré : les 686 paquets
  d'image-clé des 20 films ont leur bloc JUSTE avant eux ; le repli ne se déclenche pas.
- `grammar/keyframe_world_motif_test.go` : le motif couvre les gardes dans les deux modes ; le test
  différentiel joue aussi sous témoin (synthétique, et témoin réel des bobines).
- **Fichier de la représentation intermédiaire touché** : `grammar/keyframe_liaison.go`, une ligne
  (`TableDeDatums(pay)` → `marche.TableDeDatums(pay)`). Aucun autre.

## 3. Mesures

### 3.1 Carte de fermeture v2, 20 films, `-denominateur-fixe` (base `2393d7db7` contre lot, mesuré)

Recette L0 / vague 1 (`scratchpad/carte.sh`, `gate2.awk`). Carte de la base identique à celle de
l'intégration de la vague 1 (`integ2/carte_rev2`) hors pic et durée.

| Film | Sains avant | après | Net | Sains perdus (contredits / non fermés) | Gagnés | Au bit | dont factices | Utiles sains avant | après | Net utiles |
|---|---|---|---|---|---|---|---|---|---|---|
| 0797ce72 | 19156 | 20621 | +1465 | 1 (0 / 1) | 1466 | 1465 | 0 | 159929 | 176164 | +16235 |
| 084a804d | 4832 | 6763 | +1931 | 0 | 1931 | 1963 | 32 | 89616 | 142385 | +52769 |
| 111fa685 | 4026 | 5019 | +993 | 0 | 993 | 1004 | 12 | 42896 | 69685 | +26789 |
| 11de8353 | 5629 | 6206 | +577 | 0 | 577 | 582 | 5 | 65621 | 78974 | +13353 |
| 1c4c63c2 | 13388 | 18105 | +4717 | 1452 (229 / 1223) | 6169 | 6246 | 113 | 165965 | 307291 | +141326 |
| 396cfc92 | 23131 | 24746 | +1615 | 0 | 1615 | 1615 | 0 | 169502 | 184847 | +15345 |
| 4f77afc1 | 24082 | 27970 | +3888 | 179 (44 / 135) | 4067 | 4029 | 40 | 607244 | 728797 | +121553 |
| 50247b26 | 139 | 139 | 0 | 0 | 0 | 0 | 0 | 274 | 274 | 0 |
| 51ebbc0f | 20443 | 20825 | +382 | 10 (0 / 10) | 392 | 426 | 38 | 140617 | 144678 | +4061 |
| 60ae07c4 | 13946 | 14973 | +1027 | 0 | 1027 | 1028 | 1 | 83669 | 93250 | +9581 |
| **a349fea8** | 424 | 423 | **−1** | 1 (0 / 1) | 0 | 0 | 0 | 3654 | 3654 | 0 |
| a521164d | 692 | 692 | 0 | 0 | 0 | 0 | 0 | 78 | 78 | 0 |
| bcb6d393 | 5879 | 5879 | 0 | 0 | 0 | 0 | 0 | 35492 | 35492 | 0 |
| bf15f7ab | 28603 | 29220 | +617 | 0 | 617 | 616 | 0 | 216104 | 221853 | +5749 |
| bfecd02b | 27666 | 29382 | +1716 | 0 | 1716 | 1716 | 0 | 239045 | 258636 | +19591 |
| **c75f33b8** | 23872 | 23869 | **−3** | 3 (0 / 3) | 0 | 1 | 1 | 153997 | 153979 | **−18** |
| d9781168 | 26317 | 28573 | +2256 | 0 | 2256 | 2260 | 8 | 180854 | 201508 | +20654 |
| e5adf7b2 | 4147 | 4953 | +806 | 0 | 806 | 819 | 13 | 80067 | 103417 | +23350 |
| f75e7053 | 23673 | 23673 | 0 | 0 | 0 | 0 | 0 | 162137 | 162137 | 0 |
| fb1a1a72 | 43450 | 44099 | +649 | 0 | 649 | 654 | 5 | 325749 | 331144 | +5395 |
| **corpus** | 313495 | 336130 | **+22635** | 1646 (273 / 1373) | 24281 | 24424 | 268 (1,1 %) | 2922510 | 3398243 | **+475733** |

- Juge sur gagnés et perdus : 24 424 paquets gagnés au bit, dont 268 factices (1,1 %) ; 5 267 perdus
  au bit, dont 3 894 factices dans la référence (fermetures factices retirées) ; fermetures factices
  du corpus 9 229 → 5 751.
- Les 1 373 sains perdus « non fermés » sont TOUS des listes d'événements localisées devenues non
  localisées ; 3 894 listes non saines suivent le même chemin ; 361 listes deviennent localisées et
  saines (mesuré, `scratchpad/v2_transitions_liste.tsv`). Mécanisme : §4.
- Indicateur D1 (utiles sains / fixe consolidé recalculé, 7 823 980 : deux films lisent plus que le
  fixe de R-COMB-2) : corpus **37,4 % → 43,4 %** (variable 46,4 % → 48,6 %) ; HI_1_13_0 76,3 % →
  83,1 % ; HI_1_10_0 12,3 % → 21,4 % ; HI_1_11_0 20,9 % → 27,0 % ; HI_1_9_0 20,2 % → 24,3 % ;
  HI_1_8_0 23,3 % → 26,0 % ; HI_1_12_0, HI_1_4_1, version-31, version-33 inchangés.
- Témoins nommés : `bf15f7ab` +617 sains (R-P3 en référence : +606) ; `11de8353` +577.

### 3.2 Variantes (mesuré, même carte ; code temporaire, retiré)

| Variante | Net sains | Net utiles sains | Films en baisse |
|---|---|---|---|
| règle 1, témoin « drapeaux ≠ 0 » (V3e à la lettre) | +22 635 | +475 733 | `a349fea8` −1 / 0, `c75f33b8` −3 / −18 |
| **règle 1, témoin « drapeau 1 » (le lot)** | **+22 635** | **+475 733** | identique, carte identique à l'octet |
| règle 1 + règle 2 (désaveu des `0x7` dans `Records` et dans la table) | +16 483 | +332 282 | `a349fea8` −1, `bcb6d393` −84 / −860, `c75f33b8` −577 / −5 244, `e5adf7b2` −27 / −809, `11de8353` 0 / −194 |

Sur la base `2393d7db7` (qui porte L8), `fb1a1a72` ne perd plus (L9 : −2 / −38, D-L9-5) : +649 sains.

### 3.3 Les records déclarés, jugés par le bloc (sonde `TestLPTemoins`, mesuré)

| Classe (entrée du slot au bloc qui précède) | Référence | Lot |
|---|---|---|
| même génération, vivante (`0x5`) | 373 176 | 377 379 |
| même génération, allouée non vivante (`0x7`) | 1 492 | 2 400 |
| même génération, drapeaux 0 | 25 | 0 |
| autre génération, drapeaux 0 | 825 | 0 |

Les 850 records de D-L9-4 qui contredisaient la prémisse de V3e sont tous sur des slots non alloués
(drapeaux 0) : faux au regard de §1.2, la règle 1 les retire. Témoins de R-P3 (`TestLPSlots`) :
`bf15f7ab` slot 553 chunk 14, bloc `0x0 g0` : la référence déclarait `g1 ti20 @270696`, le lot ne
déclare rien (cas (a) de R-P3 résolu) ; `11de8353` slot 688 : aucune déclaration d'image-clé dans
aucune des deux marches (le cas (c) de R-P3 est un NEW du flux, hors de ce lot).

## 4. Les quatre paquets perdus, un par un (mesuré ; sondes `TestLPPaquets`, `TestLPBandes`, `TestLPSlots`, `TestLPTetes`)

Tous étaient des listes d'événements ouvertes par `DebutParFermeture` (NEW de tête d'où la marche
ferme le paquet) ; sous le lot, la liste n'est plus localisée. Le filtre des candidats de tête
(`candidatsDeTete`) exige que le slot du NEW soit dans la BANDE de son archétype
(`TableAnticipee.SlotDeLArchetype` : plage comblée des slots que les images-cles déclarent).

| Paquet | NEW de tête (référence) | Ce que disent les blocs | Ce qui portait la bande |
|---|---|---|---|
| `c75f33b8` 11:712, 11:716, 16:1148 | `1536 g0 ti0` | slot 1536 `drapeaux 0x0 g1` à tous les blocs des chunks 9 à 18 | la SEULE déclaration `ti0` du film : chunk 5, slot 1536 `g1` à `@527365`, bloc `0x0 g1` (slot soldé) ; la référence l'élisait dans la fenêtre où le lot lit les vrais records 1540-1549 (`ti37`, `0x5`) |
| `a349fea8` 10:932 (0 utile) | `164 g3 ti17` | slot 164 `0x0 g0` aux blocs des chunks 8 à 12 | bande `ti17` 84..291 en référence, élargie par des déclarations `ti17` sur les slots 256-272 (chunks 39-47), tous `0x0 g0` au bloc ; sous le lot 84..116 |

- Sur tout le film, la marche de référence ne lit sur le slot 1536 de `c75f33b8` que ces trois NEW
  (aucun delta, aucun DEL) ; sur le slot 164 de `a349fea8`, ce NEW et un NEW `g2 ti37` au chunk 3.
- Estimé : un NEW `1536 g0` exigerait trois allocations du slot depuis `g1` (`FUN_142f2e598`) et une
  quatrième avant l'image-clé suivante pour que le bloc suivant redise `g1`, deux fois (chunks 11 et
  16) ; un NEW `164 g3` exigerait trois allocations puis une quatrième dans le même intervalle, sans
  qu'aucun autre record de ces slots soit lu. Ces créations n'ont très probablement pas eu lieu : la
  référence fermait ces paquets en lisant comme NEW des bits que le jeu n'a pas écrits comme tels.
- Ni l'une ni l'autre n'est une fermeture factice au sens du juge de L0 (aucune de ses règles n'est
  contredite) : la règle « un NEW crée sous la génération suivant celle du slot, que le bloc suivant
  confirme » n'est pas au juge (D-LP-2).
- Population : sur les 1 373 sains perdus, le NEW de tête est, au bloc du chunk suivant, sous une
  autre génération 750 fois, libéré sous sa génération 615 fois, sans bloc suivant 8 fois. Témoin :
  sur les 45 611 listes localisées et saines des deux côtés, les têtes `DebutParFermeture` donnent
  733 « autre génération » sur 3 617 (20 %) : la classe « autre génération » n'est donc PAS une
  preuve de création fausse à elle seule (des allocations successives entre deux images-clés la
  produisent) ; seuls les quatre cas instruits ci-dessus le sont, par la chronique de leur slot.

## 5. Gates

Depuis `apps/go-api`, `GOCACHE=C:/Users/Guillaume/AppData/Local/go-build-cg2-lp`, une commande `go` à
la fois. Arbre commité = base + sonde + ce document.

| Gate | Sortie |
|---|---|
| `gofmt -l ./internal/games/halo_infinite/film/` | vide |
| `go vet ./...` | rc 0 |
| `go vet -tags=research ./internal/games/halo_infinite/film/...` | rc 0 |
| `go test ./internal/archlint/ -count=1` | ok (§9) |
| gate 2 (carte v2, 20 films) | **rouge** : `a349fea8` −1 / 0, `c75f33b8` −3 / −18 (§3, §4) |
| gate 3 (killsource json, 19 témoins, base contre lot) | 15 identiques à l'octet ; `111fa685` 2 morts, `11de8353` 1, `fb1a1a72` 1 passent de `voie=balayage/scan` à `sequentielle/marche` (avec `accord` +2/+1/+1 et les compteurs d'appariement) ; `51ebbc0f`, `bf15f7ab` : chaîne `calibration` seule. **Aucune mort apparue ni disparue, aucune valeur changée.** La voie est une sortie persistée (D-74) : sous D23, le lot retenu monterait `killsource.Rev` (backfill killsource dû). |
| gate 4 (`replay-equiv`, avant / lot alternés, 2 tours ; 3 témoins + BTB `084a804d` + `a349fea8`) | **non conclusif sous charge** (un `sed.exe` d'une autre session tourne à plein depuis le 2026-10-03, charge 36 à 77 %) : durées dans le bruit (`d9781168` 24,5 / 24,4 → 24,4 / 23,9 s ; `fb1a1a72` 29,9 / 38,7 → 27,5 / 38,6 ; `60ae07c4` 29,6 / 30,5 → 28,9 / 30,2 ; `084a804d` 99 / 127 → 102 / 116 ; `a349fea8` 123 / 137 → 127 / 156) ; pic : `d9781168` 0,36 / 0,34 → 0,41 / 0,41 Gio (+14 % / +21 %, reproduit aux deux tours), les autres ±10 % dans les deux sens. À rejouer sur machine calme avant toute fusion. |
| `replay-equiv` (5 films ci-dessus, contre les mêmes références) | étapes changées par le lot : `artifact`, `movementStates(.stats)`, `continuousFire(.stats)`, `projectiles` (`d9781168` 269 → 362), `inventory` / `loadouts` (`084a804d` 866 → 1 248 / 738 → 1 056 ; `a349fea8` 984 → 1 164 / 842 → 967), `vehicles`, `carrierMarks`, `pads`, `placements.stats`, `killRefs` (`fb1a1a72`), `killsource` (voie) ; positions, morts, identités, équipes, objectifs inchangés sur ces 5 films. Non joué sur les 15 autres (lot arrêté au gate 2). |
| compteur `repli_debut_de_liste_ferme_au_bit` (sonde `TestLPDebuts`, 20 films) | 9 069 → 5 590 ; `1c4c63c2` 6 428 → 3 079, `4f77afc1` 1 013 → 820, `0797ce72` 20 → 8 ; en hausse : `51ebbc0f` 90 → 121, `e5adf7b2` 159 → 172, `111fa685` 141 → 152, `084a804d` 472 → 481, `fb1a1a72` 39 → 44, `d9781168` 261 → 265, `11de8353` 128 → 131 ; `a349fea8` 44 → 43 ; autres inchangés |
| autres replis (`replay-equiv`, 5 films) | `repli_liaison_par_anticipation` `084a804d` 928 → 1 471, `a349fea8` 1 020 → 1 411, `60ae07c4` 441 → 486 ; `repli_identite_premier_occupant_du_siege` `084a804d` 144 → 214 ; `repli_record_desynchronise_jete` `084a804d` 2 → 0 ; `repli_image_cle_sans_bloc_de_datums` 0 partout |

Patch sous la surcouche (information) : `go test -overlay=<surcouche_lp> -run 'TestMotifDAncre|TestSuivanteEgale|TestBlocDeDatums|TestTableDeDatums|TestKeyframe'` (grammar) ok ; `facts/fallback` ok ; `replay -run 'Versement|Verse|Replis'` ok.

Non joués, le lot s'arrêtant au gate 2 et le commit ne changeant aucune sortie : G-film sur le
patch, montée de révision, `keyframe_closure.golden`, `replay-corpus-gate`, mutations, revue
adversariale de fin de lot.

## 6. Le code du lot, hors dépôt

- Patch : `scratchpad/lp_production.diff` (fichiers suivis) + `scratchpad/lp_keyframe_temoin.go.nouveau`.
- Surcouche prête : `scratchpad/surcouche_lp/overlay.json` (12 fichiers, dont le test du motif). La
  sonde de ce lot se joue sous elle :
  `go test -tags=research -overlay=<surcouche_lp/overlay.json> -run '^TestLPTemoins$' ...`.
- Reste à faire si le lot est retenu : révision (`grammar.Rev` à une valeur propre, entrée de
  `rev_chronique.go`, empreinte régénérée) ; `killsource.Rev` (D23, la voie change) ;
  `keyframe_closure.golden` et goldens à re-figer ; un test qui fait rougir une mutation de
  `temoinDeDatums.admet` (les tests différentiels du motif rejouent la même règle des deux côtés et
  ne la jugent pas) ; ligne de versement du compte du contexte de `killsource` ; gate 4 au calme.
- Variante règle 2 (rejetée) : `scratchpad/variante_regle2/`.

## 7. Écarts

- Statut [!] : aucune correction de production livrée, bien que la règle 1 soit lue dans le jeu et
  gagne nettement ; à la lettre du gate 2, deux films baissent.
- La règle 2 de l'énoncé (désaveu des non-vivantes) n'est pas portée : réfutée par la lecture du
  chargement (§1.3) et par la mesure (§3.2).
- Gate 4 non conclusif (machine chargée par un processus d'une autre session, non touché).
- Des variantes ont été mesurées avec une bascule d'environnement temporaire (`LP_VARIANTE`), retirée
  avant le commit.

## 8. Découvertes

- **D-LP-1 — Décision demandée.** Retenir la règle 1 malgré `a349fea8` −1 et `c75f33b8` −3 / −18,
  les quatre paquets étant instruits (§4) comme fermés à travers un NEW que les blocs contredisent ?
  Ou d'abord D-LP-2 (juge), puis rejouer LP sous le juge complété ? Recommandation : D-LP-2 d'abord
  si l'utilisateur tient le gate à la lettre ; le gain mesuré (+22 635 / +475 733) et la règle sont
  prêts (§6).
- **D-LP-2 — Une règle de l'écrivain absente du juge de L0** : un NEW crée sous la génération qui
  suit celle du slot (`FUN_142f2e598`), et le bloc de type 1 suivant montre le slot sous cette
  génération (ou une suivante). Non mesurée comme règle du juge ; §4 montre qu'elle ne se réduit pas
  à « même génération au bloc suivant » (allocations successives).
- **D-LP-3 — La bande d'archétype des NEW de tête (`SlotDeLArchetype`) était nourrie par des
  déclarations fausses** : sous le lot, 5 267 listes perdent leur localisation, dont 1 373 saines et
  3 894 factices (74 %) ; 615 en gagnent une, dont 361 saines. L'admission des têtes par l'allocation
  lue au bloc (L1a, « allocation de l'eid au bloc de type 1 ») est la correction générale attendue ;
  hors lot.
- **D-LP-4 — Le prédicat « vivante » n'est pas celui du décodeur** (§1.3) : `0x7` = libérée tenue
  par une vue ; son entrée d'image-clé reste au vecteur que `FUN_1406cbaa0` lit. Le commentaire de
  `type1_datums.go` (« `0x0` avec génération (libérée) » comme seule classe libérée) est incomplet
  (2 400 entrées `0x7` déclarées sur 20 films). Règle 17, à corriger avec le lot qui touchera ce
  fichier.
- **D-LP-5 — Le vecteur d'image-clé est indexé par slot** (`FUN_141f850dc` copie `i` → `i`,
  `FUN_1406cbaa0` lit `slot*200`) : confirme en partie D-L9-9 ; la marche déterministe pourrait
  compter les entrées au lieu de chercher des ancres. Non mesuré.
- **D-LP-6 — `World.BindImageCle` lit les deux bits de tête comme RANG DE VUE** (`world.go`, lot
  5.13.1) ; sous la règle 1, des records de génération 0, 2 et 3 sont déclarés (191 / 3 158 / 418
  entrées vivantes) et passent en vue inconnue (`vueDeLEspaceDeNoms`). §1.1 et L9 lisent ces bits
  comme la génération. À instruire avec la représentation intermédiaire.
- **D-LP-7 — Pic mémoire de `d9781168`** +14 % / +21 % sous charge, reproduit : cause non instruite
  (plus de records lus : `projectiles` 269 → 362, `movementStates` 2 721 → 2 724 ; ou l'index des
  blocs). À mesurer au calme.
- **D-LP-8 — La table à position libre de la sonde** (`TestLPSlots`) est la table SANS témoin même
  sous la surcouche : la table sous témoin n'existe qu'en méthode du lot.

## 9. Sonde commitée

`apps/go-api/internal/games/halo_infinite/film/internal/grammar/lp_temoin_research_test.go`
(`//go:build research`) : `TestLPTemoins`, `TestLPSlots`, `TestLPBandes`, `TestLPPaquets`,
`TestLPTetes`, `TestLPDebuts`. Elle ne dépend que de symboles de la base : sur la tête elle décrit la
marche de référence, sous la surcouche du lot la marche sous témoin. Sorties de ce lot :
`scratchpad/sonde_tete/`, `scratchpad/sonde_lp/`, `scratchpad/sonde_tetes_perdus/`,
`scratchpad/sonde_tetes_controle/`.

## 10. Contrôle indépendant (2026-10-04) : corrections exigées, NON appliquées par le lot

Recopiées à l'intégration de la vague 2 (le lot n'a pas eu de passe de corrections ; pièces du
contrôle : `scratchpad/v2-LP-ctl/`). Le statut [!] est confirmé, les chiffres du §3 et du §4 sont
reproduits à l'unité. Les points ci-dessous valent AVANT que D-LP-1 soit soumise à l'utilisateur ;
ils PRIMENT sur les passages du lot qu'ils contredisent.

1. **§1.2, bit 4** : « le bit 4 (« publié ») n'a pas été localisé chez son écrivain » est FAUX.
   `FUN_142f2e598` fait `*pbVar1 |= 4` avant `gen = (gen+1)&3`. Voie d'allocation vive, absente du
   §1.2 : `FUN_142e31ef8` appelle `FUN_142f2f634` (`|= 1` ; si le slot porte 1|2, les vues sont
   d'abord forcées à le relâcher par `FUN_142f2e710`), puis `FUN_142f2e598` (`|= 4`, génération + 1).
   `FUN_1408f1618` n'est que la voie du NEW rejoué (appelants `FUN_1408f1314`, `FUN_142f2f73c`). La
   règle 1 reste fondée : le drapeau 1 est posé par les deux voies.
2. **§5 / §6, test rouge** : le patch fait rougir `grammar/TestMarcheMemoriseeEgaleLaMarcheFraiche`
   (`keyframe_world_memoire_test.go:50`, `minibobine_000d5950` : `Refutations` 0 pour la marche
   mémorisée, 1 pour la marche fraîche) ; vert sur la base. La marche fraîche `MarcheDImageCle{preuve}`
   n'a pas de témoin, la marche mémorisée en a un.
3. **§5 / §6, test rouge** : le patch fait rougir
   `sync/killcollector/TestLeRapportDuContexteDuPontEstVerseALaPasse`
   (`repli_image_cle_sans_bloc_de_datums=26`). Les 8 mini-films de `replay/testdata` n'ont AUCUN bloc
   de type 1 avant leurs 139 images-clés ; seules les deux mini-bobines de killsource en ont
   (`000d5950` : 5, `e5adf7b2` : 1). Le « repli 0 fois » ne vaut que pour les 20 films du parc.
4. **§6, code mort** : `kfValidAnchor` (`grammar/keyframe_world.go:58`) n'a plus d'appelant de
   production sous le patch (`golangci-lint --enable-only=unused` : 1 alerte sur le lot, 0 sur la base).
5. **§6, règle 6** : `clePayload{debut: &pay[0], n: len(pay)}` passe de 1 à 4 copies en production
   (`keyframe_datums.go:199`, `keyframe_temoin.go:99` et `:175`, `keyframe_world_marche.go:96`) : helper
   et garde-rail exigés.
6. **§6, repli non compté** : `LierTableDeDatums` (`keyframe_datums.go:199`, appelé par
   `birth_loadouts.go:170`) retombe EN SILENCE sur la table sans témoin quand aucun bloc ne précède
   l'image-clé (ADR 0034 D-10) ; le compteur n'est posé que dans `memoireDesMarches.temoinDe`.
7. **§6, aucun test ne juge la règle 1** : cinq mutations restent VERTES sur `grammar`,
   `facts/killsource`, `replay` et `sync/killcollector` (hors rouges propres au lot) : `admet` sans la
   génération, `admet` sans le drapeau 1, `admet` toujours vrai, `generationDecidee` réduite à
   `gen==1`, `cleDeGeneration` rendant `gen`. Cause mesurée : les mini-films n'ont pas de bloc de type 1.
8. **D-LP-1** : « la règle et le gain sont prêts » est à nuancer — le patch fait rougir deux tests
   existants (plus les goldens de révision attendus) et ajoute une alerte de lint ; il n'est pas prêt
   à fusionner.
9. **Découverte (D-LP-9)** : le plan (§6.0 point 3, R-COMB-2) déclarait LP « nul par construction »
   pour killsource ; la mesure le contredit (la marche d'image-clé que lit killsource change ; 4 morts
   changent de voie : `111fa685` 2, `11de8353` 1, `fb1a1a72` 1). Phrase du plan amendée à
   l'intégration de la vague 2.
