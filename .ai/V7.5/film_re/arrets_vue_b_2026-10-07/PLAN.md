# PLAN — Composants où la marche depuis la fin de la vue A bute (lot arrêts vue B, 2026-10-07)

> Exécution du handoff `.ai/HANDOFF_COMPOSANTS_BLOQUANTS_VUE_B_2026-10-06.md` sous le contrat
> `plan-execution`. GO de l'utilisateur le 2026-10-07. Worktree `LevelUp-wt-grammaire-arrets-vue-b`,
> branche `feat/grammaire-arrets-vue-b`, base `feat/v75` = `879f31bbf`. Aucune fusion, aucune cuisson.

## Méthode par composant (§4 du handoff)

Lecture du déserialiseur dans le jeu (Ghidra lecture seule, `HaloInfinite.exe` HI_1_13_0) ;
portage ; vecteur de test d'après l'écrivain ; `ecs_table.tsv` ; `grammar.Rev` + empreinte
régénérée ; carte v2 (20 films) précédent état retenu contre tête ; gate 2 (aucun film en baisse
de paquets sains). Un composant qui fait baisser un film : instruit paquet par paquet, corrigé si la
cause est dans la lecture, sinon retiré et consigné. Un commit par composant retenu.

## Items

| # | Composant | Statut | Commit | Note |
|---|---|---|---|---|
| C1 | `ti=43` `i18`..`i40` (reprise de L2, `b12eb7692`) | [x] | `ec9897101` | carte v2 contre `879f31bbf` : +19 797 sains, 0 perdu, aucun film en baisse ; `grammar-2026-10-07` |
| C2 | `ti=12 i16 managed-navpoint-override-flags` | [x] | `05ab869d0` | `FUN_140ebf834` = `R(5)` ; carte v2 contre C1 : +110 sains, 0 perdu, aucun film en baisse ; `grammar-2026-10-07.2` |
| C3 | `ti=45 i0 matchflow-sequence-data-component` | [x] | `0962d0970` | `FUN_14101cdd8` = `R(4)` + 4 x `R(32)` ; carte v2 contre C2 : +861 sains, 0 perdu, aucun film en baisse ; mini-bobine : +8 annonces (perte (a) du §2.2 regagnée) ; `grammar-2026-10-07.3` |
| C4 | `ti=10 i2 managed-object-navpoint-component` (et `i3` a `i17`, même nom) | [x] | `e480f6dbb` | `FUN_14107cea4` = `R(32)` ; carte v2 contre C3 : +1 481 sains, 0 perdu, aucun film en baisse ; `grammar-2026-10-07.4` |
| C5 | `ti=12 i18 managed-navpoint-position-offset` | [x] | `05b70670a` | `FUN_140f04f68` = garde `FUN_14076f91c` + `FUN_14076e524` niveau 0x10 (`lireE494`) ; carte v2 contre C4 : +33 sains, 0 perdu, aucun film en baisse ; `grammar-2026-10-07.5` |
| D  | point (d) du §2.2 (instruit si la lecture l éclaire, pas un lot) | [~] | — | non éclairé : aucun des cinq paquets ne change entre base et tête (RAPPORT §3) |

## Gates du lot entier (§5)

| # | Gate | Statut | Sortie |
|---|---|---|---|
| G1 | carte v2 base `879f31bbf` contre tête du lot, gate 2 | [x] | +22 282 sains, +216 058 utiles, 0 perdu, 0 film en baisse |
| G2 | gate 3 : `killsource json` sur les 19 témoins, base contre tête | [x] | 17 identiques ; 3 ne diffèrent que par la ligne de diagnostic `calibration` (non persistée) ; `killsource.Rev` constante |
| G3 | `TestGoldenFilms` | [x] | ok, 4 films |
| G4 | gate de corpus (`replay-corpus-gate --reference=base`), chaque FAUX/PERTE instruit, 0 MANQUE | [x] | rc 1 ; banc 18 ok, 0 MANQUE, 1 FAUX instruit (RAPPORT §2.1) ; PERTE de filet instruites (§2.2) |
| G5 | gofmt, vet (+ `-tags=research`), archlint | [x] | gofmt vide ; vet, research, integration rc 0 ; archlint ok |
| G6 | golangci-lint 0 issue (`--new-from-rev=879f31bbf`) | [x] | 0 issues |
| G7 | mutations rouges sur chaque nouveau lecteur | [x] | 28 / 28 rouges |
| G8 | baseline des tests renommés/supprimés | [x] | aucun test retiré ni renommé |
| G9 | `make gate-push` (TMP court dédié) | [x] | EXIT_GATEPUSH=0 (lint 0 issue, web vert, baseline 9 533 / 9 533 présents, 0 échec) |
| G10 | push + CI | [~] | push après ce commit ; état CI dans le message de clôture au pilote |

## Correctif D9 (GO de l'utilisateur du 2026-10-07)

| # | Item | Statut | Sortie |
|---|---|---|---|
| D9.1 | Compter l'événement sur les documents réels (parc 126 + 19 témoins) avant tout code | [x] | parc schéma 86 : 14 dans 6 films ; témoins base `12b8fb3df` : 18 dans 5 (RAPPORT §7.2) |
| D9.2 | Règle trouvée à la source, corrigée en un seul endroit, règle générale | [x] | `qualifierContre` / `armeEnMain` (RAPPORT §7.1) |
| D9.3 | `grep` des autres consommateurs (facts, killsource, web) | [x] | aucun autre chemin ne fabrique le faux événement |
| D9.4 | Test rouge avant / vert après ; mutation rouge | [x] | 2 tests ; mutations 5 / 5 rouges |
| D9.5 | Révisions (rangs de travail) | [x] | `SchemaVersion` 83, `grammar.Rev` `.6` ; à renuméroter (RAPPORT §7.5) |
| D9.6 | Gate de corpus, chaque FAUX / PERTE instruit, 0 MANQUE | [x] | banc 19 / 19 ok, 0 FAUX, 0 MANQUE ; 5 PERTE de filet instruites (§7.3) |
| D9.7 | Recompte après sur les 19 témoins : 0 | [x] | 0 ; le reste du document identique hors révisions |
| D9.8 | gofmt, vet, golangci-lint, tests des paquets, archlint | [x] | §7.6 |
| D9.9 | `make gate-push` | [!] | EXIT 2, instruit sans correction (causes hors périmètre) : lint 0, web vert ; deux paquets non touchés coupés à 300 s puis `ok` seuls (313 s, 310 s) ; un test de durées instable sous charge (2 / 3) ; baseline complète (RAPPORT §7.7) |
| D9.10 | push, CI | voir le message de clôture au pilote | le push suit ce commit |

## Journal

- 2026-10-07 : plan écrit ; carte v2 de base (binaire de `879f31bbf`, ~2 min pour 20 films).
- C1 : port de L2 repris sur la tête (le maillon s insère entre M4b et `ti=40` ; `i37` passe par le lecteur de minuteur unique `lireMinuteur142ba78dc`, venu depuis L2 ; la règle du masque de `pasDEssai` était déjà en tête, lot LT). Ghidra relu : `i19` `FUN_1410156e4`, `i21` `FUN_1407f0678` / `FUN_1407f08bc` / `FUN_1407f08f8`. Carte v2 : 0 perte, la perte de `1c4c63c2` de L2 (second rang) ne se reproduit pas (+13). Retenu.
- Mutations C1 et C2 jouées (`tsv/mutations.sh`, overlay) : 20 / 20 ROUGES.
- C2 : lecteur `FUN_140ebf834` -> `FUN_140ebf854` (`R(5)` vers `etat+0x70c`), écrivain `142ed0e2c` (cinq bits du même mot), trouvés par nom -> `getName` -> descripteur (slot après le thunk `FUN_14076ce9c`). Carte v2 contre C1 : +110 sains (64 arrêtés sur `i16` en C1), 0 perdu ; les autres records `i16` avancent et s arrêtent sur `i18` (63 -> 318) et `i17` (92). Retenu.
- C3 : lecteur `FUN_14101cdd8` (`FUN_14101d200` = `R(4)` valeur - 1, puis quatre `R(32)`), écrivain `FUN_142edbf94` (`FUN_1407ebac4` = octet + 1 sur quatre bits, puis les quatre mots). Carte v2 contre C2 : +861 sains (dont `1c4c63c2` +779), 0 perdu ; 50 gagnés arrêtés sur `ti=45 i0` en C2, 714 sortaient par rejet. Mini-bobine `000d5950` : la marche du paquet 2:712 passe le slot 122 et lit les huit annonces d emplacement vide des bipèdes 512 à 519 (`heldWeaponChanges` 6 -> 14, records bipèdes 29 511 -> 29 519) : c est la perte (a) du §2.2 du handoff, regagnée. Goldens mis à jour (changement de décodage déclaré). G-film 21 ok. Mutations S1-S3 ROUGES. Retenu.
- C4 : le registre de `ti=10` pose SEIZE descripteurs sous le nom `managed-object-navpoint-component` (`i2` à `i17`) ; une seule table (accesseur de nom `14064c7d0`, table `143c971d8`), un seul lecteur `FUN_14107cea4` = `R(32)` vers `etat+0x14+4*index` (`index = *(descripteur+8)`), écrivain `142edb304` (32 bits du même mot). Les seize lignes sont portées (G4 157 -> 173). Carte v2 contre C3 : +1 481 sains (`1c4c63c2` +796, `111fa685` +248, `e5adf7b2` +165), 0 perdu ; arrêts suivants : `ti=10 i22` (291), `ti=10 i18`..`i21`. G-film 21 ok. Mutations O1-O2 ROUGES. Retenu.
- C5 : lecteur `FUN_140f04f68` : garde de pleine précision `FUN_14076f91c` (0 bit) ; sous la garde `FUN_1411b259c` = `R(96)`, sinon `FUN_14076e524` niveau `0x10` (`MOV R9D,0x10` en `140f04f80`, `CALL 140f04f8b`) : la forme de `FUN_14076e494`, portée par `lireE494` (le portage unique de la position ; site ajouté à la table du ratchet `lecteur_position_ratchet_test.go` et aux flux `lecteur_position_sites_test.go`). Écrivain `142edb0b4` -> `141f860b0` -> `FUN_1407eb61c` niveau `0x10`. Carte v2 contre C4 : +33 sains, 0 perdu ; les records `ti=12` vont jusqu aux `visual-state-groups` (`i20` 433, `i21` 675, `i22` 532) et `i17` (92). G-film 21 ok. Mutations P1-P3 ROUGES (P2, garde retirée, VERTE au premier passage : vecteur sous la garde ajouté, `TestLeDecalageDuMarqueurLitSousLaGarde`, puis ROUGE). Retenu.

## Découvertes

Voir RAPPORT.md §4 (D1 à D8).
