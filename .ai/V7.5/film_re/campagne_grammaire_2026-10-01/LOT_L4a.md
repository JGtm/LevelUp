# Lot L4a — véhicules `ti=40` en delta : seize composants, porte posée par la loi de l'écrivain (2026-10-03)

> Lot L4a du plan `.ai/PLAN_CAMPAGNE_GRAMMAIRE_2026-10-01.md` (§6.2 L4, partie delta seulement ;
> l'image-clé L4b, LK et LM sont hors périmètre), sous le contrat `plan-execution`, dans le cadre de
> la décision du 2026-10-02 : **corrections d'abord, uniquement générales, lues dans le jeu**.
>
> Worktree `LevelUp-wt-cg-l4a`, branche `feat/cg-l4a`, base `af6e93e23` (= `feat/campagne-grammaire`
> = `feat/v75`, L0 fusionné). Films lus en place, un à la fois, plafond 4 Gio ; aucune écriture dans
> le checkout principal ; mesures sous `scratchpad/L4a/`. Aucune base ouverte en écriture, aucune
> cuisson du parc.
> Convention : **mesuré** = compté par un outil sur les films ; **établi** = lu dans l'exécutable
> (Ghidra, HTTP direct 127.0.0.1:8089, lecture seule) ; **estimé** / **supposé** = dit comme tel.

## 0. Statut

**[x] retenu.** Gate 2 tenu sur les 20 films (aucun paquet sain perdu, aucun record utile sain perdu,
sur aucun film) ; gate 3 sans aucune mort changée ; mutations 18 / 18 rouges. Les divergences de
`replay-equiv` et du gate de corpus sont toutes rattachées à un mécanisme (§6, §7) ; deux d'entre
elles changent ce que voit le rejeu (durée de vie des pièces montées, fin de vie des véhicules) et
sont à regarder par l'utilisateur à la recuisson de la vague (§9, D-L4a-1).

| Item | Statut | En une ligne |
|---|---|---|
| Composants i30-i32, i35, i36, i38-i42, i45-i47 | fait | treize lecteurs à largeur du flux, chacun chez son désérialiseur |
| i33, i34 sous la porte `+0x818` | fait | porte posée par la loi du masque (deux écrivains relus), aucun châssis consulté |
| Image-clé (état complet) | inchangée | aucun composant du véhicule lu hors `i37` ; L4b |
| Table châssis -> type de physique (D5) | non ajoutée | le delta n'en a pas besoin (décision du pilote) |
| Repli `repli_physique_de_type_de_vehicule_supposee` | retiré | critère de retrait tenu : porte lue à l'écrivain, 0 lecture supposée |
| `d9781168` 0 / −11 en marginal (R-COMB-2) | instruit | un paquet, `17:1356` ; tête de liste factice choisie par L1a / LS en C11 ; absent du gate du lot (§5) |
| Révisions | faites | `grammar-2026-10-03` ; source, killsource, objectives à révision constante |

## 1. Ce qui est lu dans le jeu

Table nom -> descripteur -> désérialiseur : `T6_vehicules_ti40.md` §1 (vérifiée adverse,
`VERIFICATIONS_ADVERSES.md` T6-C1 à C4). Relu ce jour, en décompilation et désassemblage :

| Composant | Désérialiseur | Grammaire | Bits |
|---|---|---|---|
| `i30 vehicle-auto-turret-triggers` | `FUN_142f04994` | 3 x `FUN_1406cf008` R(1) | 3 |
| `i31 vehicle-auto-turret-aiming-vector` | `FUN_14115f33c` | `FUN_14076dc04(.., 0x13)` R(19), `FUN_1404fedf8` 0 bit | 19 |
| `i32 vehicle-transformed-or-desired-open-state-changed` | `FUN_142f04b70` | R(1) ; `FUN_1406d84b4` `MOV [RSP+0x20],0x8` @142f04ba1 | 9 |
| `i33 vehicle-type-state` | `FUN_142f02474` -> `FUN_14320c4c8` | si `+0x818` : `FUN_142af27f8` R(2) v ; v = 1 ou 3 : R(6) | 2 / 8 |
| `i34 vehicle-type-physics` | `FUN_142f02498` | si `+0x818` : R(1) mode ; `FUN_140c5f938` + `FUN_14076e1c8` (lecteur existant) | variable |
| `i35 vehicle-auto-turret-target` | `FUN_142f0496c` | `FUN_1408f0ac4(+0x83c, .., 1)` | 1 / 13 / 17 |
| `i36 vehicle-sentry-state` | `FUN_142f04b34` | `FUN_1424d9a30` R(3) ; R(1) | 4 |
| `i37 vehicle-emp-timer` | `FUN_142f049dc` | R(8) (déjà porté, déplacé) | 8 |
| `i38 vehicle-weapon-set` | `14116d3cc` (hors fonction Ghidra) | octets `49 8B 48 10 48 81 C1 4C 08 00 00 E9 20 2E 56 FF` : `ADD RCX,0x84c ; JMP 0x1406d01fc` | 5 à 9 |
| `i39 vehicle-auto-turret` | `FUN_142f04884` | R(2) (`ADD [RDX+0x2c],0x2` @142f048aa) | 2 |
| `i40 vehicle-equipment-turret-parent` | `FUN_142f04a00` | `FUN_1408f0ac4(+0x834, .., 0)` | 1 / 16 |
| `i41`, `i42 vehicle-seats-override-pitch/yaw` | `FUN_142f04a4c`, `FUN_142f04ac0` | 2 x `FUN_1406d84b4` largeur 8 | 16 |
| `i45 air-drop-flight` | `FUN_142f02508` | `FUN_142af27f8` R(2) ; largeur 0xe @142f02540 ; 0x8 @142f02562 | 24 |
| `i46 warp` | `FUN_142f04bcc` | R(1) a ; R(1) b ; si a : 0x8 @142f04c1a ; si b : 0x8 @142f04c48 | 2 / 10 / 18 |
| `i47 vehicle-low-frequency` | `FUN_142f04a20` -> `FUN_1407f1ff4` -> `FUN_1407f2058` | R(1) ; si 0 : R(5) | 1 / 6 |

**La porte `+0x818` est une loi du masque (établi).** L'octet est posé par le constructeur d'état
`FUN_14058c2ec` (slot `+0x88`) à `(type de physique du tag vehi == 6)` (T6 §2, vérifié adverse). Les
DEUX écrivains du masque de présence ne posent les bits 33 et 34 que sous cet octet ; relus ce jour :

- `FUN_142f09c74` (slot `+0x78`, masque de différence) : `FUN_14320c8fc` n'est appelé que si
  `*(param_2 + 0x818) != 0`, et son résultat est OU-é dans le masque de sortie (`FUN_1406cb04c`) ; le
  parent (`FUN_142ee0348`) n'écrit que le mot 0 du masque ;
- `FUN_142e32138` (masque complet d'un ajout de pair) pose tous les bits puis appelle
  `vtable+0x90` = `FUN_142ee98ec` -> `FUN_142f13c1c`, qui REMPLACE le masque par le retour de
  `vtable+0x130` = `FUN_142f0cca0` ; celui-ci efface les bits 0x1e à 0xff, puis ne repose les bits
  33 et 34 que par `FUN_143208c18`, qui rend un masque nul si `*(état + 0x7f0 + 0x28)` (= `+0x818`)
  est nul. `FUN_142ee4fc8` (appelé ensuite) n'ajoute que le bit 25 au mot 0 ; le parent
  `FUN_142ee09a8` n'écrit que le mot 0.

Donc, dans un record lu AVEC un masque (DELTA, NEW), l'annonce de `i33` ou de `i34` prouve que
l'écrivain avait la porte posée, quel que soit le châssis. L'égalité avec la porte du lecteur repose
sur la construction symétrique de l'état par `FUN_14058c2ec` depuis le même tag : hypothèse nommée
(vérification adverse T6-C3, point 4), non tracée côté lecteur.

**Ce qui n'est PAS établi** : l'inventaire exhaustif des marquages explicites (`FUN_1406c99ac` et
ses appelants directs, `FUN_142ed0364`) relevé incomplet par la vérification adverse (T6-C3, lentille
code). Il ne change pas la loi : un marquage n'ajoute qu'une DEMANDE de comparaison, la sortie passe
par les deux écrivains ci-dessus. Réserve gardée (D-L4a-8).

## 2. Ce qui change

| Fichier | Changement |
|---|---|
| `grammar/composants_vehicule_ti40.go` (neuf, 215 l.) | dernier maillon de la chaîne de dispatch : les seize composants, `i33` / `i34` sous la loi du masque, aucun lu dans un état complet sauf `i37` |
| `grammar/composants_vue_b_m4b.go` | perd `i34` / `i37` (déplacés) ; son `default` chaîne vers le maillon véhicule |
| `grammar/lecteur.go`, `keyframe_fullstate_loop.go` | `Lecteur.etatComplet`, posé par la seule marche d'état complet (`FUN_142e2c690`, sans masque) |
| `grammar/unit_weaponstate.go` | `lireJeuDArmes` (`FUN_1406d01fc`) extrait, partagé par le bipède `i42` et le véhicule `i38` (le crochet du bipède n'est pas appelé pour le véhicule) |
| `grammar/movement_states.go`, `types/grammar_mouvement.go`, `replay/filmfacts_{encode,decode}.go` | compteur `VehicleTypePhysicsAssumed` -> `VehicleTypePhysicsByWriterLaw` (même place dans le blob des faits) |
| `facts/fallback/noms.go`, `registre_filmdec_marche.go`, `replay/film_scan_mouvement.go` | repli `repli_physique_de_type_de_vehicule_supposee` retiré (nom, entrée, versement) |
| `grammar/testdata/ecs_table.tsv` | seize lignes `ti=40` : statut `partiel` (le maillon refuse en état complet), désérialiseur, grammaire, largeur, source ; quatre sources `composants_vue_b_m4b.go:N` recalées |
| `grammar/ecs_widths_guard_test.go` | G4 : 123 -> 131 largeurs fixes (`i30` 3, `i31` 19, `i32` 9, `i36` 4, `i39` 2, `i41` 16, `i42` 16, `i45` 24) |
| `grammar/composants_vehicule_ti40_test.go` (neuf) | vecteurs d'après l'écrivain (§4) |
| `rev.go`, `rev_chronique.go`, goldens | §8 |

**L'image-clé est inchangée, par construction et par mesure.** Dans un état complet, `i33` et `i34`
sont désérialisés pour tout véhicule sous la porte du châssis (`FUN_142e2c690`, aucun masque) :
c'est L4b. Lire `i30`-`i32` ne ferait que déplacer l'arrêt de `i30` à `i33`. Le maillon refuse donc
tous les composants propres au véhicule hors `i37` quand `etatComplet` est posé : la boucle
d'image-clé s'arrête à `i30` comme avant. Mesure : `keyframe_closure.golden` inchangé (paquet
`grammar` vert), et `replay-equiv` ne bouge aucune étape d'image-clé (§6). Écart avec la recherche :
la sonde de R-VEH / R-COMB-2 appliquait son crochet aux deux marches (porte posée partout).

**Aucune table châssis -> type de physique** (D5, décision du pilote) : le delta n'en a pas besoin.

## 3. Carte de fermeture v2, avant / après, par film (gate 2)

« Avant » = `af6e93e23` (TSV identiques à ceux de L0 « après » hors pic et durée, contrôlé) ;
« après » = le lot. Même outil (`cmd_fermeture -mode v2 -paquets -denominateur-fixe
r_comb2_denominateurs.tsv`), chacun avec sa table ECS. « Sain » = fermé selon la définition de L0
(au bit près ET aucune règle de l'écrivain contredite) ; comparaison paquet par paquet
(`l4a_tsv/comparer.awk`). Dénominateur fixe consolidé : inchangé sur les 20 films (aucune marche ne
lit plus que lui).

| Film | Build | Sains avant | Sains après | Net | Gagnés sains | Perdus sains (dont contredits / non fermés) | Gains au bit | dont contredits | Utiles sains avant | Utiles sains après | Net utiles | Part fixe | Utiles lus |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0797ce72 | HI_1_13_0 | 19152 | 19152 | +0 | 0 | 0 (0 / 0) | 0 | 0 | 159920 | 159920 | +0 | 71.0 -> 71.0 % | 205685 -> 205685 |
| 084a804d | HI_1_10_0 | 4773 | 4801 | +28 | 28 | 0 (0 / 0) | 31 | 3 | 88036 | 88791 | +755 | 11.3 -> 11.4 % | 535804 -> 536274 |
| 111fa685 | HI_1_10_0 | 4012 | 4024 | +12 | 12 | 0 (0 / 0) | 15 | 3 | 42601 | 42874 | +273 | 13.0 -> 13.1 % | 245556 -> 245620 |
| 11de8353 | HI_1_9_0 | 5612 | 5619 | +7 | 7 | 0 (0 / 0) | 9 | 2 | 65230 | 65391 | +161 | 20.1 -> 20.1 % | 248287 -> 248382 |
| 1c4c63c2 | HI_1_10_0 | 13333 | 13346 | +13 | 13 | 0 (0 / 0) | 26 | 13 | 165404 | 165603 | +199 | 12.6 -> 12.6 % | 777075 -> 777242 |
| 396cfc92 | HI_1_13_0 | 22819 | 22819 | +0 | 0 | 0 (0 / 0) | 0 | 0 | 166984 | 166984 | +0 | 69.7 -> 69.7 % | 225495 -> 225495 |
| 4f77afc1 | HI_1_13_0 | 22260 | 22865 | +605 | 605 | 0 (0 / 0) | 598 | 9 | 550289 | 569385 | +19096 | 66.6 -> 68.9 % | 738490 -> 745103 |
| 50247b26 | version-31 | 139 | 139 | +0 | 0 | 0 (0 / 0) | 0 | 0 | 274 | 274 | +0 | 0.1 -> 0.1 % | 319053 -> 319252 |
| 51ebbc0f | HI_1_13_0 | 9759 | 9759 | +0 | 0 | 0 (0 / 0) | 0 | 0 | 58585 | 58585 | +0 | 26.2 -> 26.2 % | 88493 -> 88493 |
| 60ae07c4 | HI_1_8_0 | 13802 | 13802 | +0 | 0 | 0 (0 / 0) | 0 | 0 | 82730 | 82730 | +0 | 23.0 -> 23.0 % | 268352 -> 268352 |
| a349fea8 | version-33 | 420 | 420 | +0 | 0 | 0 (0 / 0) | 2 | 2 | 3595 | 3595 | +0 | 0.7 -> 0.7 % | 469451 -> 469728 |
| a521164d | HI_1_4_1 | 692 | 692 | +0 | 0 | 0 (0 / 0) | 0 | 0 | 78 | 78 | +0 | 0.0 -> 0.0 % | 181607 -> 181722 |
| bcb6d393 | HI_1_12_0 | 5830 | 5830 | +0 | 0 | 0 (0 / 0) | 0 | 0 | 35143 | 35143 | +0 | 23.7 -> 23.7 % | 121628 -> 121628 |
| bf15f7ab | HI_1_13_0 | 28465 | 28465 | +0 | 0 | 0 (0 / 0) | 0 | 0 | 214974 | 214974 | +0 | 92.3 -> 92.3 % | 227637 -> 227637 |
| bfecd02b | HI_1_13_0 | 26380 | 27108 | +728 | 728 | 0 (0 / 0) | 729 | 1 | 226529 | 232918 | +6389 | 83.0 -> 85.3 % | 254963 -> 257060 |
| c75f33b8 | HI_1_13_0 | 22854 | 22854 | +0 | 0 | 0 (0 / 0) | 0 | 0 | 144430 | 144430 | +0 | 80.6 -> 80.6 % | 155136 -> 155136 |
| d9781168 | HI_1_13_0 | 26210 | 26210 | +0 | 0 | 0 (0 / 0) | 1 | 1 | 180052 | 180052 | +0 | 55.9 -> 55.9 % | 244872 -> 244873 |
| e5adf7b2 | HI_1_11_0 | 4123 | 4146 | +23 | 23 | 0 (0 / 0) | 24 | 1 | 79457 | 80066 | +609 | 20.7 -> 20.9 % | 289357 -> 289562 |
| f75e7053 | HI_1_13_0 | 23416 | 23416 | +0 | 0 | 0 (0 / 0) | 0 | 0 | 160042 | 160042 | +0 | 83.2 -> 83.2 % | 187304 -> 187304 |
| fb1a1a72 | HI_1_13_0 | 22276 | 22276 | +0 | 0 | 0 (0 / 0) | 0 | 0 | 161566 | 161566 | +0 | 44.9 -> 44.9 % | 179254 -> 179254 |
| **corpus** | | 276327 | 277743 | **+1416** | 1416 | **0** | 1435 | 35 | 2585919 | 2613401 | **+27482** | 33.3 -> 33.7 % | 5963499 -> 5973802 |

Lecture :

- **0 sain perdu, en brut, sur les 20 films** (ni « devenu contredit », ni « devenu non fermé ») :
  « 0 sain perdu » s'écrit à la lettre. Aucun film en baisse en paquets ni en records utiles sains.
- **Juge sur les GAGNÉS** : 1 435 paquets deviennent fermés au bit près ; 35 contredisent une règle
  de l'écrivain (**2,4 % de gains factices**, détail `l4a_tsv/comparaison_details.txt` : masque
  au-delà de l'archétype 32, ordre de la vue B 8, sortie par rejet 6, masque épars non croissant 4
  — plusieurs règles par paquet) ; ils ne comptent pas comme sains. 16 paquets fermés au bit et
  contredits AVANT deviennent sains (1 416 = 1 435 − 35 + 16).
- **Juge sur les PERDUS** : aucun paquet sain avant n'est perdu ; aucun paquet fermé au bit avant
  n'est perdu non plus.
- Les 1 666 paquets que la référence arrête sur un composant `ti=40` non porté (`fermeture_bloquants`,
  22 569 records utiles en jeu) passent à 0. Devenir de ces 1 666 : 910 fermés sains, 654 sortie de
  vue B par rejet (un en-tête plus loin dans la liste), 60 terminateur hors cadre, 12 kind non porté,
  6 bloc 0xbc, 6 fin de payload, 18 sur un autre composant non porté. Le reste du gain (506 sains)
  vient de paquets que la référence n'arrêtait pas sur `ti=40` : liaisons posées par les records
  désormais lus jusqu'au bout (NEW `ti=40` compris), estimé (non ventilé paquet par paquet).
- Comparaison avec la recherche (R-COMB-2, levier L4a seul, juge à trois invariants) : mêmes gains
  sur `084a804d`, `111fa685`, `11de8353`, `bfecd02b`, `e5adf7b2` ; `4f77afc1` +605 contre +596 et
  `1c4c63c2` +13 sans perte contre +9 / 4 perdus : la recherche lisait aussi les images-clés
  porte posée et jugeait sans les règles de L0 (D-L4a-6).

## 4. Vecteurs et mutations

Vecteurs construits d'après les désérialiseurs (T6 §6.1, relus §1), `composants_vehicule_ti40_test.go` :

- `TestComposantsTi40LargeurDuFlux` : 21 vecteurs, chaque composant dans un record à masque, fin
  exacte (tampon suivi de 32 bits à 1 : une lecture trop longue se voit) ;
- `TestPorteTi40LoiDuMasque` : `i33` (v = 0, 1, 2, 3 : 2 ou 8 bits) et `i34` mode 2 ;
- `TestComposantsTi40EtatComplet` : en état complet, chacun des quinze composants s'arrête à 0 bit
  (non porté), `i37` lit ses 8 bits ; dans un record à masque, chacun lit ;
- `TestEtatCompletPoseParLaMarcheDImageCle` : un record d'image-clé synthétique (en-tête de 108 bits,
  `ti=40`, `n1 = 0`, `n2 = 1`) passé par `WalkKeyframeFullState` s'arrête à `i30` après `i37` ; la
  même liste lue avec un masque va au bout.
- G1 et G4 de la table ECS (statuts, sources, largeurs fixes) verts.

Mutations (`l4a_tsv/mutations.sh`, copie mutée + `go test -overlay`, sortie `l4a_tsv/mutations.txt`) :
**18 / 18 ROUGES** — M1 marche d'état complet sans `etatComplet`, M2 `i33`/`i34` lus en état complet
(porte supposée), M3 composants à largeur du flux lus en état complet, M4 et M18 complément de `i33`
faux, M5 à M16 une largeur ou une porte fausse par composant (`i30`, `i31`, `i32`, `i35`, `i36`,
`i39`, `i40`, `i41`/`i42`, `i45`, `i46`, `i47`, `i38` via `FUN_1406d01fc`), M17 `i37` refusé en
état complet. L'arbre non muté est vert.

## 5. Les pertes instruites

**Au gate du lot : aucune** (§3). La seule perte à instruire était celle de R-COMB-2 :

**`d9781168`, 0 / −11 records utiles sains en MARGINAL (C11 contre C11 privée de L4a).** Rejouée à
l'identique (sonde `TestRComb2` de `1a47b8c11` sous la surcouche unique, dans un worktree temporaire
retiré depuis ; `full` 309 622 et `full-L4a` 309 633 utiles sains, égaux à `r_comb2_configs.tsv`),
avec un vidage paquet par paquet (`l4a_tsv/rcomb2_instruction_d9781168.patch`, hors dépôt de
production) : un seul paquet diffère, **`17:1356`**, sain dans les deux configurations.

- Sans L4a : liste localisée au bit 526, douze records (DELTA `ti=4`, six `ti=35`, trois `ti=42`,
  deux `ti=37`), fermée, **11 records utiles**.
- Avec L4a : la tête de liste choisie est le bit **1856**, un seul record lu : un NEW `ti=40` sur le
  slot 1338 (eid `0x4000053a`), masque `{i40}`, fermé, **0 record utile**. Le bit 1856 tombe DANS le
  record #8 de la lecture à douze records (`ti=42`, bits 1746-1861).
- Le bloc de type 1 du chunk 17 dit le slot 1338 **vivant, génération 1**, et l'image-clé le déclare
  **`ti=14`** : un NEW `ti=40` sur ce slot contredit la table de datums (l'écrivain n'écrit un NEW que
  pour une entité nouvellement allouée). La lecture à un record est factice ; le juge de L0 ne porte
  pas de règle de datum (L0.8 non retenu) et la classe « sain ».
- Mécanisme : le lecteur de `i40` est juste (lu chez `FUN_142f04a00`) ; il rend fermable un candidat
  de tête qui ne l'était pas (avant L4a, `i40` non porté arrêtait ce candidat), et le choix de la
  tête des leviers de marche de C11 (L1a / LS : paquet à événements non localisé par le localisateur
  strict, `strict = -1`) prend ce candidat. En production (base + L4a), le même paquet reste « liste
  d'événements non localisée » avant et après (`fermeture_paquets.tsv`) : la perte n'existe pas sans
  L1a / LS.
- Conclusion : perte de COMBINAISON, imputable au choix de tête des leviers de marche, pas à la
  lecture de L4a. À rejuger quand L1a / LS entrent (vague 2), avec une règle de datum (D-L4a-2).

## 6. Gate 3 (killsource) et `replay-equiv`

**`cmd/killsource json`**, 19 témoins de `config/replay_corpus.toml`, binaire de la base contre
binaire du lot : **18 / 19 identiques à l'octet** ; `e5adf7b2` : une seule ligne, le diagnostic de
l'ORACLE de calibration (`PROFIL PLAT (score 523` -> `524`), qui n'est pas persisté
(`Result.Calibration`) — `l4a_tsv/killsource_e5adf7b2_diff.txt`. **Aucune mort, aucune arme, aucune
voie ne change.** `cmd/killsource sante` : 19 / 19 identiques. `killsource.Rev` ne monte pas (golden
régénéré à révision constante, §8) — même règle que L0 et R-COMB-2 pour un diagnostic (supposée, non
instruite dans le dépôt).

**`replay-equiv`** (recette de L0 : racine factice `scratchpad/L4a/repo`, les 20 films du corpus
d'équivalence copiés du parc, `data/titles/halo_infinite/reference` et `config/` identiques à l'octet
à ceux du worktree ; faits et artefacts vidés avant chaque passe ; `-out-dir` pour garder les
digests). Les références figées sont antérieures à L0 : la base elle-même rend 20 / 20 « différents »
d'elles (comme au rapport de L0) ; la comparaison qui fait foi est **base contre lot, digest par
digest** (`l4a_tsv/cmp_equiv.sh`, `replay_equiv_etapes_divergentes.tsv`).

| Étape | Films | Cause (établie sur les vidages `084a804d`, `e5adf7b2`, `50247b26`, `000d5950` ; « même mécanisme » ailleurs, estimé) |
|---|---|---|
| `artifact` | 20 | révision de calque `grammar-2026-10-03` et les étapes ci-dessous |
| `movementStates.stats` | 20 | le compteur renommé (le digest hache les noms de champs) ; sur les films à véhicules, plus de paquets et de listes lus (`084a804d` : paquets 29 043 -> 29 048, listes localisées +5, lectures `i34` 8 914 -> 8 939) |
| `killsource` | 11 | compteurs de repli de la marche des morts : `RecordsDesynchronises` baisse (`084a804d` 29 -> 2, `50247b26` 18 -> 0), `HorsBandeBipede` monte (25 -> 59, 14 -> 32) — les états de mort de véhicule lus dans des records qui ne se désynchronisent plus passent au filtre de bande ; aucun kill ne change |
| `continuousFire.stats` | 11 | plus de paquets atteints et fermés (`084a804d` fermés 4 773 -> 4 801 = la carte) |
| `vehicles` | 10 | états de mort de véhicule : `084a804d` 28 -> 35 lus, `TailDesync` 25 -> 0 (sept morts nouvelles, dans des paquets désormais lus au-delà d'un record `ti=40`) |
| `continuousFire` | 5 | rafales lues sur les paquets nouvellement fermés |
| `movementStates` | 2 | `084a804d` 12 552 -> 12 554 états |
| `birthLoadouts(.stats)` | 1 | `084a804d` dotations lues 15 -> 16 |

Les 46 autres étapes (positions, morts de joueur, identités, équipes, objectifs, équipement, image-clé,
drapeau...) sont identiques à l'octet sur les 20 films. Durées et pics semblables (± 10 %, deux films
plus rapides : `50247b26` 1 m 24 -> 1 m 07, `a521164d` 47 s -> 35 s ; pic `1c4c63c2` 1,79 -> 1,87 Gio,
`53ce4390` 0,41 -> 0,49 Gio).

## 7. Gate de corpus (`replay-corpus-gate`, information du gate 6)

`--reference=base --base=af6e93e23` explicite, `--parc-root` sur une copie du parc au scratchpad
(films et manifestes des 19 témoins, `metadata.duckdb`, `shared_matches_v2.duckdb`, copiés en
lecture), `--work-root` au scratchpad ; worktree de base créé et retiré par l'outil. Rapport :
`l4a_tsv/corpus_gate_rapport.txt`. **rc 1** : 13 témoins en PERTE, `111fa685` FAUX, 5 ok. Ce qui
fait PERTE, famille par famille :

- **Couverture du tir continu et des états** (`holesNotClosing`, `holesKind`, `holesBlockBC`,
  `burstsWithHole`, `stances.forgottenBindings`, `dropped`, `refusedNews*`) : +1 à +129 par témoin.
  Plus de paquets ATTEINTS, donc plus de trous comptés ; même famille qu'à L0.
- **`bfecd02b` `stances/durée/slot 574` 162 -> 106, `4f77afc1` quelques slots** : un intervalle de
  sprint de `bfecd02b` (slot 574, 3822-3883) se ferme à 3827 ; une lecture de plus (`reads` 2 279 ->
  2 280) dans un paquet que la base ne lisait pas. Estimé (non instruit paquet par paquet).
- **Véhicules (`4f77afc1`, `084a804d`, `111fa685`, `11de8353`, `e5adf7b2`)** : `samplesAfterEnd`
  monte (`4f77afc1` 577 -> 1 636), `par-end/unknown` baisse (110 -> 76), `vehicles/durée-totale` baisse
  sur `4f77afc1` (226 325 -> 225 331) et les durées de monte avec. Instruit sur les artefacts gardés
  (`--keep-work`) de `4f77afc1` et `11de8353` :
  - `4f77afc1` : morts de véhicule lues 34 -> 74, `TailDesync` 34 -> 0, fins « détruit » 34 -> 74 ;
  - les cinq durées en baisse sont des PIÈCES MONTÉES (`family` nulle) : 795, 812, 870, 876, 976,
    portées par 796 (Wraith), 813 (Wraith), 871 (Wraith), 877 (Falcon), 977 (Falcon). Exemple : la
    pièce 812 vivait 1123-6540 (fin inconnue) quand son porteur 813 finissait à 2537 ; elle meurt
    désormais à 2454, **au même instant que son porteur**, dont la mort est lue aussi (fin
    « inconnue » -> « détruit », dernier échantillon inchangé). Les quatre autres paires : même motif ;
  - `11de8353` : six vies passent de « inconnue » à « détruit » (780/781 Wraith et sa pièce à 682,
    793/794 Scorpion et sa pièce à 3307...), dernier échantillon inchangé, aucune durée en baisse.
  Lecture : la marche lit désormais les états de mort de véhicule écrits dans des paquets qu'elle
  abandonnait ; la couche véhicules ferme les vies à ces morts. Les échantillons d'une pièce après la
  destruction de son porteur deviennent « après la fin » (D-L4a-1).
- **Banc de vérité** : `111fa685` FAUX sur une seule ligne, `R-1 repli repli_deadstate_hors_bande_bipede
  : 0 -> 1` (le banc classe FAUX tout repli déclenché pour la première fois, D-L0-5) ; P-1 en gain sur
  sept témoins (= la carte : 4 773 -> 4 801, 22 260 -> 22 865, 26 380 -> 27 108...) ; aucun oracle
  (kills, morts, assistances, équipes, vies) ne bouge. Le repli retiré :
  `repli_physique_de_type_de_vehicule_supposee` X -> 0 sur sept témoins (`4f77afc1` 14 117, `084a804d`
  8 914, `50247b26` 5 374, `e5adf7b2` 2 502, `111fa685` 1 939, `11de8353` 326, `a349fea8` 22).

## 8. Révisions

- `grammar.Rev` : `grammar-2026-10-02` -> **`grammar-2026-10-03`** (forme du dépôt : le premier lot
  d'un jour neuf est sans rang ; l'intégrateur renumérote la vague), entrée de `rev_chronique.go`,
  empreinte `7b2b83e5…` par la commande du dépôt.
- `killsource.Rev` inchangée (`killsource-2026-09-27`), golden régénéré à révision constante
  (`4c074747…`) avec historique : l'empreinte bouge par la valeur de `grammar.Rev` et par
  `film/types` ; sortie §6.
- `source.Rev` inchangée (`23e87562…`) et `objectives.Rev` inchangée (`6e3d9a1b…`), goldens à
  révision constante avec historique : leur fermeture rencontre `film/types` (un champ renommé),
  aucune de leurs sources ne change.
- `replay.SchemaVersion` reste 76 (règle de L0) : le document change par les calques de grammaire,
  que la révision signale déjà (verdict « à recuire »).
- Régénérés : `types/testdata/shapes.golden` (révisions + champ renommé), fixtures de contrat
  `replay_schema_76_*.json.gz` + `manifest.json` (8 films : identiques hors chaîne `grammar-…`,
  vérifié par `jq -S`, fins de ligne neutralisées, substitution).

## 9. Sorties exactes des gates

Depuis `apps/go-api`, `GOCACHE=C:/Users/Guillaume/AppData/Local/go-build-cg-l4a`, une commande `go`
à la fois :

| Gate | Sortie |
|---|---|
| `gofmt -l ./internal/games/halo_infinite/film/ ./cmd/` | vide |
| `go vet ./internal/games/halo_infinite/film/...` | rc 0 |
| `go vet -tags=research ./internal/games/halo_infinite/film/...` | rc 0 |
| `go test ./internal/archlint/` | `ok` (44,0 s) |
| G-film `go test ./internal/games/halo_infinite/film/... ./internal/replaybuild/... ./internal/sync/killcollector/... -count=1 -timeout 30m` | 19 paquets `ok` (`grammar` 31,4 s, `replay` 25,1 s) |
| `go test -tags=research ./internal/games/halo_infinite/film/research/cmd_fermeture/` | `ok` |
| `golangci-lint run --new-from-rev af6e93e23` (grammar, facts, replay, types) | `0 issues.` |
| Carte v2, 20 films | §3 : 0 sain perdu, +1 416 / +27 482 |
| killsource json, 19 témoins | 18 identiques, 1 diagnostic d'oracle (§6) |
| `replay-equiv`, 20 films | base contre lot : §6 |
| `replay-corpus-gate` | rc 1, §7 |
| Mutations | 18 / 18 rouges |

## 10. Écarts

- **Image-clé hors périmètre** : les composants à largeur du flux ne sont pas lus dans un état
  complet (§2), pour garder la marche d'image-clé inchangée ; c'est L4b.
- **`i33` / `i34` en état complet** passent de « lus porte supposée » à « non portés » : sans effet
  mesuré (`keyframe_closure.golden`, carte, `replay-equiv` inchangés sur les étapes d'image-clé : la
  boucle s'arrête à `i30` avant eux).
- **Repli retiré dans le lot** : le critère de retrait de son entrée est tenu (porte lue à
  l'écrivain, 0 lecture supposée) ; l'autre session qui retire des replis n'en touche pas d'autre.
- **Compteur renommé** : `film/types` change, donc l'empreinte de `source` et d'`objectives` (révisions
  constantes) et `shapes.golden`. Garder l'ancien nom aurait gardé un nom faux.
- **Revue adversariale de fin de lot** (plan §6.0 : L4 est un lot à risque) : non faite ici, aucun
  agent disponible dans cette exécution ; à faire par le pilote ou l'intégrateur.
- **`replay-corpus-gate` rouge** (rc 1) : toutes les lignes sont rattachées §7 ; deux familles
  changent ce que montre le rejeu des véhicules (fin de vie lue, pièces montées) — à regarder par
  l'utilisateur à la recuisson de la vague.
- **Règles de l'exécution enfreintes, sans effet sur les livrables** : une édition de commentaire
  faite par `python3` en ligne (interdit « pas de Python ») — elle a converti le fichier en CRLF,
  reconverti en LF aussitôt, contrôlé par `git ls-files --eol` ; une redirection tentée vers `/` (un
  fichier temporaire mal placé) refusée par le système, rien créé.
- Deux worktrees temporaires (`scratchpad/L4a/wt_rc2` à `1a47b8c11`, `wt_base` à `af6e93e23`) pour
  l'instruction de `d9781168` et les vidages de `replay-equiv`, modifiés hors dépôt puis restaurés et
  retirés sans `--force`.

## 11. Découvertes (consignées, non traitées)

- **D-L4a-1** — Pièces montées vivantes après la destruction de leur porteur : dans la base, la
  pièce 812 de `4f77afc1` garde des échantillons de position jusqu'à 6540 alors que son porteur
  (Wraith 813) finit à 2537 ; idem pour 795, 870, 876, 976. Le lot lit la mort (porteur et pièce au
  même instant) ; les échantillons de la pièce après cette mort (`samplesAfterEnd` +1 059 sur
  `4f77afc1`) restent inexpliqués (slot republié ? épave ?). Non instruit.
- **D-L4a-2** — Tête de liste factice dans C11 (`d9781168` `17:1356`) : un NEW `ti=40` sur un slot
  que la table de datums dit vivant sous un autre archétype (`ti=14`). Une règle de datum (NEW
  seulement sur un slot nouvellement alloué, `FUN_142f2e598`) l'écarterait ; à poser au juge avant
  la vague 2 (L1a, LS) ou avec LP.
- **D-L4a-3** — Le gate de corpus classe « perte » la baisse de `vehicles/par-end/unknown` (des vies
  qui gagnent une fin lue) et la hausse des trous comptés quand plus de paquets sont atteints : le
  sens de ces axes est à revoir (même famille que D-L0-4 / D-L0-5).
- **D-L4a-4** — `repli_deadstate_indice_hors_roster` monte sur plusieurs témoins (`e5adf7b2` 25 -> 41,
  `11de8353` 44 -> 60) : états de mort de véhicule dont l'indice d'auteur est hors roster. Aucun kill
  ne change ; non instruit.
- **D-L4a-5** — En image-clé, `i30`-`i32` sont lisibles sans porte ; `i33` / `i34` exigent le
  châssis (L4b, avec LK et LM).
- **D-L4a-6** — Les chiffres « seul » de R-VEH / R-COMB-2 pour L4a ont été mesurés avec le crochet
  appliqué aussi aux images-clés (porte posée) et sous le juge à trois invariants : ils ne valent pas
  exactement pour le lot (§3).
- **D-L4a-7** — `i40 vehicle-equipment-turret-parent` porte une référence d'entité (catégorie 0) :
  le rattachement pièce -> porteur, fait aujourd'hui par voisinage de slot (`repli_tourelle_porteur
  _voisin_de_slot`, 55 -> 61 sur un témoin), pourrait la LIRE (T6 §8). Non instruit.
- **D-L4a-8** — Inventaire des marquages explicites (`FUN_1406c99ac`, `FUN_142ed0364`) toujours
  incomplet (réserve de la vérification adverse T6-C3) ; la loi repose sur les deux écrivains relus.
