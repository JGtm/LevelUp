# Lot L2 — les composants `device-*` de `ti=43` lus dans le jeu, et sa réparation (2026-10-02)

> Lot L2 du plan `.ai/V7.5/PLAN_CAMPAGNE_GRAMMAIRE_2026-10-01.md` (§6.2 L2, §6.3 D19), sous le contrat
> `plan-execution`, dans le cadre de la décision du 2026-10-02 au soir : **corrections d'abord,
> uniquement générales, lues dans le jeu** (aucun réglage par film, par carte ni par version).
>
> Worktree `LevelUp-wt-cg-l2`, branche `feat/cg-l2`, base `af6e93e23` (= `feat/campagne-grammaire`
> = `feat/v75`, lot L0 fusionné). Films lus en place, un à la fois (`data/cache/film_chunks` du
> checkout principal, lecture seule). Aucune base ouverte en écriture, aucune cuisson en lot.
> Convention : **mesuré** = compté par l'outil sur les films ; **établi** = déduit du jeu lu (Ghidra,
> lecture seule, HTTP `127.0.0.1:8089`, image `HaloInfinite.exe` HI_1_13_0) ; **estimé / supposé** =
> hypothèse écrite.

## 0. Statut

**[!] non retenu en l état (contrôle du 2026-10-03, §9).** Le port des 23 composants est fidèle au jeu
(relu par le contrôle) et le gate 2 tient sur les 21 films quand L2 est mesuré seul (§5). Mais la
mesure que D19 exigeait n'était pas faite : rejouée (§9.3, C3), **la contribution marginale du L2 de
production dans C11 reste en baisse sur `1c4c63c2` : −447 paquets sains / −8 520 records utiles
sains** (44 979 -> 44 532 ; −476 / −9 160 avec la copie de recherche de R-COMB-2 sur la même base).
La perte est instruite paquet par paquet : ce n'est pas une fermeture factice retirée (exception D2),
c'est une perte de paquets sains causée par un faux en-tête NEW `ti=43` pris comme tête de liste au
SECOND RANG de `debutParFermeture`, que la réparation (preuve par chaîne) ne vise pas (D-L2-12). Le
gate 2 « aucun film en baisse » n'étant jamais assoupli, **le code du lot est retiré** par le commit
qui suit celui des corrections (revert par commit) ; les corrections C1, C2, C5, C6 restent dans
l'historique (commit « campagne(grammaire) L2: corrections du controle ») pour la reprise du lot.

| Item | Statut | En une ligne |
|---|---|---|
| Port des 22 lecteurs `i19`..`i40` (+ `i18` déplacé) | [!] | fidèle au jeu (contrôle) ; retiré avec le lot (gate 2 de D19 en défaut, §9.3) |
| Réparation (D19) | [!] | règle de l'écrivain juste, mais elle ne vise que `debutParChaine` : la perte de C11 passe par le second rang de `debutParFermeture` (§9.3) |
| Règle des copies (`FUN_140d580d0`) | [!] | lecteur unique et garde-rail retirés avec le lot ; la recopie à trois sites (minuteurs) préexiste à L2 et revient : dette notée (D-L2-13), hors périmètre |
| `ecs_table.tsv`, garde G4 | [!] | retirés avec le lot |
| Vecteurs d'après l'écrivain | [x] | V31c (N = 8) et masques dense court / épars non croissant ajoutés (C1, C2) ; K3 et K4 ROUGES (§9.1) ; retirés avec le lot |
| Révision | [!] | `grammar-2026-10-02.2` retirée avec le lot (`grammar.Rev` revient à `grammar-2026-10-02`) |
| Gates §6.0 1, 2, 3 + replay-equiv + mutations, L2 seul | [x] | §6, §9.5 |
| Gate 2 en combinaison (D19, C11) | [!] | `1c4c63c2` −447 / −8 520 (§9.3) |
| replay-corpus-gate (§6.0 point 6) | [x] | changements de contenu instruits au paquet (C4, §9.4) ; banc de vérité 19 / 19 ok |
| `capture.go` | [~] | non touché : aucune valeur `device-*` n'est captée (D-L2-10) |
| Corrections du contrôle C1 à C6 | [x] | §9 |

## 1. Ce qui est lu dans le jeu

Image : `HaloInfinite.exe` HI_1_13_0, la même que celle des notes du 16-17/09 et de T7
(`NOTE_3_6_TI43_GRAMMAIRES_A/B/C/D_2026-09-17.md`, `T7_dispositifs_ti43_moteur.md` §1-§2). Relus ce
jour (décompilé et désassemblage, sorties sous `scratchpad/L2/ghidra/`) : `FUN_142f02a48` (`i31` :
`*(+0x5a0) = N ; if (N < 9) {...N x FUN_1408f0ac4(.., 1)...; return 1} else return 0`, `142f02b1e CMP
R9D,0x8`), `FUN_143206ae0` (`i36` : deux entrées, `FUN_1408f0ac4(.., 0)` sous la porte, `+0x2c += 3`
inconditionnel), `FUN_142f02c94` / `FUN_142ba78dc` / `FUN_140d580d0` (`i37` : `142ba790d CALL
FUN_1406d84b4` après `MOV [RSP+0x20],EBX` = la même largeur `n`), `FUN_141076f68` / `FUN_143206f24`
(`i35` : `if (0 < w) Q(14)`), `FUN_142e2da44` (l'écrivain du masque : boucle `i < *(desc+0x4320)`,
`7 < compte` => dense `R(1)=1` + `R(64)`, sinon `R(1)=0` + `R(3)` compte + index croissants).

| Composant | Lecteur (`descripteur + 0x40`) | Grammaire portée |
|---|---|---|
| `i18` position | `FUN_140bef320` | `Q(14)` + `R(1)` (inchangé, déplacé) |
| `i19` | `FUN_1410156e4` | `R(32)` + `Q(10)` |
| `i20` | `FUN_142f02d04` | `R(256)` |
| `i21` | `FUN_1407f0678` (écrivain `FUN_142f05cd0`) | `R(32)` + porte [`R(8)`] |
| `i22`, `i23` | `FUN_14100d310`, `FUN_14100d2d0` | `Q(14)` |
| `i24`, `i29` | `FUN_141167910`, `FUN_14116fcb0` | `FUN_1408f0ac4` catégorie 1 |
| `i25`, `i40` | `FUN_142f02c54`, `FUN_141fd7bc0` | `Q(8)` |
| `i26` | `FUN_142f029e4` | 2 x `R(32)` |
| `i27` | `FUN_140bee524` | `R(1)` + `R(6)` |
| `i28`, `i30`, `i33` | `FUN_142f02bec`, `FUN_142f02c20`, `FUN_142f02b7c` | `R(1)` |
| `i31` | `FUN_142f02a48` | `R(8)` N ; N > 8 : échec (arrêt propre) ; sinon N x `FUN_1408f0ac4` cat. 1 |
| `i32` | `FUN_142f02bcc` | `R(5)` |
| `i34` | `FUN_140f44104` -> `FUN_143206e48` (écrivain `FUN_142f056a4`) | A ; 8 x [G ; `R(32)` + `Q(14)` x 2 + `Q(10)` + `Q(14)` + `R(2)`] |
| `i35` | `FUN_141076f68` -> `FUN_143206f24` (écrivain `FUN_142f0570c`) | A ; 8 x [G ; `Q(10)` w ; code de w != 0 : `Q(14)`] |
| `i36` | `FUN_142f02bb0` -> `FUN_143206ae0` | 2 x [G ; `FUN_1408f0ac4` cat. 0 ; `R(3)`] |
| `i37` | `FUN_142f02c94` -> `FUN_142ba78dc` | 2 x [G ; `Q(10)` + `Q(10)` + `R(5)` + `Q(10)`] |
| `i38` | `FUN_142f02d28` | `Q(18)` |
| `i39` | `FUN_14107bb68` | `R(9)` |

Le niveau déclaré par le film n'est lu par aucun de ces lecteurs (trois paramètres) et
`vtable[0x10]` rend faux pour tous ces descripteurs (T7 §1.1) : **dans l'exécutable lu, tout film
dont le registre déclare l'un de ces noms est lu par ce lecteur, quel que soit son niveau**. C'est
ce que fait le décodeur (dispatch par nom depuis le registre du film).

**Les trois familles du registre (D6).** Le port s'applique à tout film qui déclare ces noms.
Établi pour la famille HI_1_10_0 à HI_1_13_0 (registre identique, empreinte `a5d95bfd`, T7 §3) et
l'exécutable lu. Pour HI_1_8_0 = HI_1_9_0 et HI_1_4_1 = v31 = v33 : **supposé** (même nom, même
niveau dans le registre du film ; aucun exécutable de ces builds). Aucune restriction n'a été
nécessaire : le gate 2 tient sur tous les films (§5) ; mesuré sur ces familles : `60ae07c4` +2,
`11de8353` +3, `50247b26` +1 sains, `a521164d` et `a349fea8` 0, aucune perte.

## 2. Ce qu'il fallait réparer : l'échec sur HI_1_10_0 instruit paquet par paquet

Mesuré sur le port seul, sans réparation (`l2_tsv/net_sains_port_seul_sans_reparation.tsv`, juge
de L0) : **un seul film en baisse, `084a804d` −3 paquets sains / −105 records utiles sains** (4
sains devenus contredits, 1 gagné) ; `1c4c63c2` +8 / 0 (le −28 / −2 de R-COMB-2 était mesuré sous
l'ancien juge à trois invariants).

**Les quatre paquets `084a804d` 2:154, 2:158, 2:162, 2:166** (et 2:150, non fermé) — détail
`l2_tsv/084a804d_2_150-158_avant_apres_port_seul.diff` :
- avant : le localisateur pose le début à 7 014 (2:154) ; la liste `DELTA 123 ...` se lit et le
  paquet ferme sans règle contredite ;
- port seul : `debutParChaine` trouve, 198 bits plus tôt, un en-tête `NEW slot=6947 eid=0x1b23
  ti=43` dont la traversée (désormais portée) tombe **au bit près** sur 7 014, et prend la liste à
  ce NEW. Son masque vaut `0x800001022010000` : bit 59, alors que `ti=43` a 41 composants. Le
  paquet devient « fermé au bit, masque au-delà de l'archétype » : contredit.
- Établi par l'écrivain : `FUN_142e2da44` ne pose aucun bit `i >= *(desc+0x4320)`. Cet en-tête n'a
  pas été écrit par le jeu. Le même faux NEW (même slot, même eid, même écart de 198 bits) précède
  le début localisé dans les cinq paquets : ce sont les bits de la fin de la vue A, pas un record.
- Avant le port, la traversée du faux NEW désynchronisait sur le premier `device-*` non porté, et
  `pasDEssai` le refusait par accident. Le port retire l'accident ; la règle de l'écrivain le
  remplace par une raison.

**Les gains factices de HI_1_10_0 (BIS_2 : 382 / 401 ; R-COMB-2 D-105)** — détail de douze paquets
de `1c4c63c2` (`l2_tsv/1c4c63c2_gains_factices_detail.txt`) : tous ont `strict=-1` (le localisateur
ne trouve rien) ; `debutParFermeture` prend au **second rang** (repli
`repli_debut_de_liste_ferme_au_bit`) un candidat `NEW ti=43` dont le masque est aléatoire
(`0xbc914910078b7df6`, `0x20bf1d827b5db27c`, `0x836419172dea6b7`…) ou épars non croissant. Avant le
port, ces candidats désynchronisaient ; ils se traversent maintenant et ferment parfois au bit près
— toujours contredits. Le juge de L0 les exclut (D2) : ce ne sont pas des gains. Mesuré par
composant et par build (`l2_tsv/l2_dispositifs.tsv`) : sur HI_1_9_0, HI_1_10_0 et HI_1_11_0, les
records `ti=43` des paquets non sains portent les 23 composants à des comptes presque égaux
(140 à 175 par composant sur HI_1_10_0, dont 103 à 138 comme dernier record lu) : la signature de
masques lus au hasard ; sur HI_1_13_0, la répartition est celle d'objets réels (`i18` 110 557 sains,
`i21` 669, `i22`/`i23`/`i34`/`i35` 240, `i39` 507). **Conclusion : l'échec sur HI_1_10_0 n'est pas
une grammaire `device-*` fausse ; ce sont de faux en-têtes NEW `ti=43` que deux chemins de tête de
liste acceptaient.** Ce que la mesure n'établit pas : qu'un vrai NEW `ti=43` se lise juste sur
HI_1_10_0 (aucun NEW `ti=43` dans un paquet sain de ce build, mesuré ; les deltas `i22`/`i23` y sont
lus dans 39 / 40 paquets sains).

## 3. Ce qui change (production)

1. `grammar/components_device_ti43.go` (neuf, 216 lignes) : le maillon `consumeComposantsDispositif`
   (entre `consumeNavpointComponent` et `consumeComposantsVueBM4b`), un `case` par composant avec
   son lecteur du jeu ; `i18` y est déplacé (`dispatch_biped.go`). `i31` au-delà de 8 rend
   `ported = false` (statut `partiel` de la table, comme les filtres de `ti=12`).
2. `grammar/debut_de_liste.go`, `pasDEssai` : un pas NEW ou delta dont `MasqueNonEcrit` n'est pas
   `InvariantAucun` est refusé. Règle de `FUN_142e2da44`, la même que L0.7 juge dans la fermeture ;
   seule la preuve par chaîne change. Le second rang de `debutParFermeture` (D-A) n'est pas touché.
3. `grammar/bit_leaf_readers.go` : `consume140d580d0` (lecteur unique de `FUN_140d580d0`) ; les trois
   copies existantes (`decodeGameEngineRoundTimer`, `consumeGameEngineCampaignTimer`,
   `consumePlayerSoftKillTimer`) l'appellent, mêmes bits ; `consume142ba78dc` (pour `i37`) aussi.
   Garde-rail : `lecteur_140d580d0_guard_test.go` (toute fonction de production dont la doc cite
   `FUN_140d580d0` ou `FUN_142ba78dc` doit appeler le lecteur unique).
4. `testdata/ecs_table.tsv` : `ti=43 i18` à `i40` (statut, adresse, grammaire, `bits_typ` — entier
   pour les largeurs fixes, `variable` sinon —, source, confiance) ; `ecs_widths_guard_test.go` :
   123 -> 138 largeurs fixes.
5. Révisions : `grammar.Rev` `grammar-2026-10-02.2` (entrée de `rev_chronique.go`, empreinte
   `b7f04d51…`) ; `killsource.Rev` **inchangée** (§6.3), golden à révision constante (`a0a59c83…`) ;
   `types/testdata/shapes.golden` (ligne des révisions) ; fixtures de contrat
   `replay_schema_76_*.json.gz` + `manifest.json` : identiques hors chaîne `grammar-2026-10-02.2`
   (vérifié par `jq -S` et substitution, 8 / 8). `source.Rev`, `profile.Rev`, `objectives.Rev`,
   `replay.SchemaVersion` : inchangés.

## 4. Vecteurs et mutations

Vecteurs construits d'après l'écrivain (T7 §6.1) : `TestLesDispositifsLisentCeQueLEcrivainEcrit`
(V35a-c, V34a-b, V21a-b, V19, V22, V23, V24a-c, V29a, V31a-c, V36, V37, V39 ; chaque vecteur suivi de
64 uns, consommation exacte exigée) ; `TestUneChaineQuiTraverseUnMasqueNonEcritNeProuveRien` (NEW et
delta, chacune des trois règles du masque, chaque règle avec son témoin écrit par le jeu ; C2, §9.1) ; `TestFUN140d580d0AUnSeulLecteur`.

Mutations (`l2_tsv/mutations.sh`, `l2_tsv/mutations.txt`, `go test -overlay`, base verte) :
**12 / 12 ROUGES** — M1 (règle du masque retirée, pas NEW), M2 (idem, pas delta), M3 (`i35` sans
condition), M4 (`i31` borné), M5 (`i36` catégorie 1), M6 (`i37` sans 3e `Q`), M7 (`i34` un `Q(14)`
de moins), M8 (`i21` sans porte), M9 (`i24`/`i29` catégorie 0), M11 (`i36` sans `R(3)`), M12 (maillon
débranché), M10bis (troisième copie de `FUN_140d580d0`, mutation EN PLACE puis fichier rendu à
l'octet). M10 par overlay reste VERTE **par construction** : le garde-rail lit les sources sur
disque (`go/parser`), pas la compilation (D-L2-7). Le contrôle a trouvé deux mutations vertes (K3, K4) ; corrigées (C1, C2), elles
sont ROUGES (§9.1).

## 5. Gate 2 — carte de fermeture v2, par film

Commande : `cmd_fermeture -mode v2 -denominateur-fixe r_comb2_tsv/r_comb2_denominateurs.tsv
-paquets -top 0` sur les 20 films + `81c02726`, binaire de la base (`af6e93e23`) contre binaire du
lot, table de chaque côté (`l2_tsv/carte.sh`). Comparaison paquet par paquet (`net_sains.awk` de L0,
`juge.awk`). « Sains » = fermés (L0) sans règle contredite. Les TSV par paquet (65 Mo) restent au
scratchpad (`scratchpad/L2/carte_avant`, `carte_apres`).

| Film | Build | Sains avant | Sains après | Net | Sains perdus | Utiles sains avant | Utiles sains après | Net utiles | Au bit gagnés | dont factices | Perdus au bit |
|---|---|---|---|---|---|---|---|---|---|---|---|
| 0797ce72 | HI_1_13_0 | 19 152 | 19 152 | 0 | 0 | 159 920 | 159 920 | 0 | 0 | 0 | 0 |
| 084a804d | HI_1_10_0 | 4 773 | 4 779 | +6 | 0 | 88 036 | 88 175 | +139 | 38 | 37 | 0 |
| 111fa685 | HI_1_10_0 | 4 012 | 4 012 | 0 | 0 | 42 601 | 42 601 | 0 | 5 | 5 | 0 |
| 11de8353 | HI_1_9_0 | 5 612 | 5 615 | +3 | 0 | 65 230 | 65 267 | +37 | 30 | 29 | 0 |
| 1c4c63c2 | HI_1_10_0 | 13 333 | 13 341 | +8 | 0 | 165 404 | 165 404 | 0 | 359 | 354 | 12 |
| 396cfc92 | HI_1_13_0 | 22 819 | 26 904 | +4 085 | 0 | 166 984 | 201 591 | +34 607 | 4 087 | 2 | 0 |
| 4f77afc1 | HI_1_13_0 | 22 260 | 22 759 | +499 | 0 | 550 289 | 564 920 | +14 631 | 502 | 26 | 0 |
| 50247b26 | version-31 | 139 | 140 | +1 | 0 | 274 | 298 | +24 | 1 | 0 | 0 |
| 51ebbc0f | HI_1_13_0 | 9 759 | 9 787 | +28 | 0 | 58 585 | 58 819 | +234 | 27 | 0 | 0 |
| 60ae07c4 | HI_1_8_0 | 13 802 | 13 804 | +2 | 0 | 82 730 | 82 746 | +16 | 1 | 1 | 0 |
| a349fea8 | version-33 | 420 | 420 | 0 | 0 | 3 595 | 3 595 | 0 | 0 | 0 | 0 |
| a521164d | HI_1_4_1 | 692 | 692 | 0 | 0 | 78 | 78 | 0 | 0 | 0 | 0 |
| bcb6d393 | HI_1_12_0 | 5 830 | 17 126 | +11 296 | 0 | 35 143 | 115 213 | +80 070 | 11 297 | 1 | 0 |
| bf15f7ab | HI_1_13_0 | 28 465 | 28 466 | +1 | 0 | 214 974 | 214 982 | +8 | 1 | 0 | 0 |
| bfecd02b | HI_1_13_0 | 26 380 | 26 382 | +2 | 0 | 226 529 | 226 548 | +19 | 0 | 0 | 0 |
| c75f33b8 | HI_1_13_0 | 22 854 | 22 855 | +1 | 0 | 144 430 | 144 438 | +8 | 1 | 0 | 0 |
| d9781168 | HI_1_13_0 | 26 210 | 26 211 | +1 | 0 | 180 052 | 180 060 | +8 | 6 | 5 | 0 |
| e5adf7b2 | HI_1_11_0 | 4 123 | 4 125 | +2 | 0 | 79 457 | 79 497 | +40 | 11 | 11 | 0 |
| f75e7053 | HI_1_13_0 | 23 416 | 25 038 | +1 622 | 0 | 160 042 | 172 591 | +12 549 | 1 623 | 1 | 0 |
| fb1a1a72 | HI_1_13_0 | 22 276 | 22 397 | +121 | 0 | 161 566 | 162 535 | +969 | 118 | 0 | 0 |
| **corpus (20)** | | **276 327** | **294 005** | **+17 678** | **0** | **2 585 919** | **2 729 278** | **+143 359** | 18 107 | 472 | 12 |
| `81c02726` (témoin) | HI_1_13_0 | 15 184 | 18 763 | +3 579 | 0 | 123 525 | 157 363 | +33 838 | 3 580 | 1 | 0 |

- « Sains perdus » ventilés (« devenus contredits » / « devenus non fermés ») : **0 / 0 sur tous les
  films**. Aucun film en baisse ni en paquets sains ni en records utiles sains.
- Juge des invariants joué sur les GAGNÉS : 472 fermetures au bit gagnées contredisent une règle
  (2,6 % des 18 107) — 97 à 100 % des gains au bit de HI_1_9_0, HI_1_10_0, HI_1_11_0, HI_1_8_0 ;
  0 à 5 % ailleurs. Règles : masque au-delà de l'archétype 259, sortie de vue B par rejet 173,
  épars non croissant 31, ordre de la vue B 9 (`l2_tsv/juge_detail_gagnes_perdus.txt`). Elles ne
  comptent pas comme gain (D2).
- Juge sur les PERDUS : les 12 paquets perdus au bit (`1c4c63c2` 49:610, 704, 796, 816, 818, 824,
  826, 832, 848, 850, 870, 912) étaient tous **contredits** avant (« masque au-delà de l'archétype »,
  0 record utile, liste localisée et fermée au bit près) et deviennent « liste non localisée » : fermetures
  factices retirées (exception D2), aucun sain.
- 43 paquets contredits deviennent sains (requalification : la liste reprend au début localisé au lieu d un faux NEW de tête ;
  la réparation seule en requalifie 33).
- Témoins nommés : `bcb6d393` +11 296 / +80 070 (HI_1_12_0 : 26,7 % -> 78,3 % des paquets) ;
  `81c02726` +3 579 / +33 838 (records utiles sains : fixe 76,6 % -> 97,5 %, variable 79,9 % -> 98,4 %).
- Dénominateur fixe (`l2_tsv/fermeture_denominateurs_*.tsv`, inchangé : aucune marche du lot ne lit
  plus que le maximum consolidé) ; part fixe des utiles sains, corpus : 33,3 % -> 35,2 %
  (2 585 919 -> 2 729 278 sur 7 758 290) ; variable : 43,4 % -> 45,6 % (lus 5 963 499 -> 5 990 548). `111fa685` lit 3 records utiles de moins (245 556 -> 245 553), sans effet sur ses
  sains.
- Réparation seule, sans le port (`l2_tsv/net_sains_reparation_seule.tsv`, mesuré) : +33 sains /
  +770 utiles sains sur 8 films, 0 perdu (D-L2-2).

## 6. Sorties des autres gates

### 6.1 Gate 1 (depuis `apps/go-api`, `GOCACHE=C:/Users/Guillaume/AppData/Local/go-build-cg-l2`)

| Gate | Sortie |
|---|---|
| `gofmt -l ./internal/games/halo_infinite/film/ ./cmd/` | vide |
| `go vet ./internal/games/halo_infinite/film/...` | rc 0, aucune sortie |
| `go vet -tags=research ./internal/games/halo_infinite/film/...` | rc 0, aucune sortie |
| `go test ./internal/archlint/` | `ok` (70,8 s) |
| G-film `go test ./internal/games/halo_infinite/film/... ./internal/replaybuild/... ./internal/sync/killcollector/... -count=1 -timeout 30m` | 19 paquets `ok`, rc 0 (après régénération des goldens de §3.5) |
| `golangci-lint run --new-from-rev=af6e93e23 ./internal/games/halo_infinite/film/internal/grammar/...` | `0 issues.` |
| Mutations | 12 / 12 rouges (§4) |

### 6.2 Gate 2 : §5.

### 6.3 Gate 3 — killsource

`cmd/killsource json <film> -carte <carte> -cache <checkout>/data/cache -catalogue
map_quant_bounds.json` sur les 19 témoins de `config/replay_corpus.toml` (`l2_tsv/ks_jouer.sh`),
binaire de la base (overlay des sources de `af6e93e23`) contre binaire du lot : **18 / 19 identiques
à l'octet** ; `bfecd02b` diffère sur trois valeurs du bloc `sante` seulement
(`l2_tsv/killsource_bfecd02b.diff`) : `killsource_candidates_total` 95 -> 96,
`killsource_unexplained_self` 5 -> 6, `taux_inexpliques` 0,1368 -> 0,1458. **Aucune mort, aucune
ligne publiée (82), aucune voie, aucun verdict ne change.** Cause (estimée, non instruite au
record) : la marche lit au-delà d'un record `ti=43` qui l'arrêtait et y trouve un candidat de plus,
inexpliqué, non publié. Même constat que R-COMB-2 (« un compteur de santé sur `bfecd02b` »).
`killsource.Rev` **n'est pas montée** : la sortie persistée (`match_kill_events`) ne change pas ; un
compteur de diagnostic n'appelle pas de redécodage du parc (règle supposée, non instruite — plan
§6.0 point 3). Golden régénéré à révision constante. `1c4c63c2` et `81c02726` : sans carte lisible,
non joués (D-67).

### 6.4 replay-equiv (recette L0)

Racine factice `scratchpad/L2/repo` (copie des 20 films du corpus d'équivalence, de
`data/titles/halo_infinite/reference` et de `config/`, références d'équivalence ; toutes vérifiées
identiques à l'octet au worktree et au checkout par `diff -rq`). Binaire de la base (overlay) et du
lot, digests conservés (`-out-dir`), comparés étape par étape.

Bilans : base `BILAN : 0 identique(s), 20 different(s)` (18 min 54 s) et lot idem (19 min 04 s) contre
les références du dépôt — **attendu** : les références n'ont pas été re-figées après L0 (LOT_L0 §7.1,
mêmes cinq étapes). Le verdict du lot se lit **base contre lot**, digests à digests
(`l2_tsv/replay_equiv_etapes_divergentes.tsv`, `replay_equiv_comptes.tsv`) : **20 / 20 films
diffèrent, sur 9 étapes au plus sur 61** ; les 52 autres étapes (positions, morts, identités,
équipes, objectifs, équipement, armes au sol…) sont identiques à l'octet sur les 20 films.

| Étape | Films | Ce qui change |
|---|---|---|
| `artifact`, `continuousFire.stats` | 20 | compteurs du tir continu et révision de calque |
| `movementStates.stats` | 18 | compteurs des états de mouvement (records lus, listes non localisées, liaisons oubliées) |
| `vehicles` | 10 | objet unique (compte 1 -> 1) ; sur les deux films cuits, seul `repli_physique_de_type_de_vehicule_supposee` bouge (+1 sur `084a804d`) |
| `killsource` | 8 | objet unique (compte 1 -> 1) ; sur les deux films cuits, seul `repli_deadstate_hors_bande_bipede` bouge (`084a804d` 25 -> 27, `000d5950` 4 -> 11) ; aucune mort publiée ne change |
| `movementStates` | 7 | états lus en plus : `000d5950` 4 680 -> 4 684, `084a804d` 12 552 -> 12 553, `1c4c63c2` 10 381 -> 10 390, `696a9d7c` 5 175 -> 5 177, `bcb6d393` 2 613 -> 2 616 (+ `7344d24f`, `01e1f945` à compte égal) |
| `continuousFire` | 7 | rafales : `000d5950` 8 -> 20, `bcb6d393` 1 -> 4, `1c4c63c2` 196 -> 198, `7344d24f` 6 -> 7 ; contenu seul sur `01e1f945`, `084a804d`, `696a9d7c` |
| `birthLoadouts(.stats)` | 3 / 4 | une dotation de naissance fermée de plus : `084a804d` 15 -> 16, `1c4c63c2` 27 -> 28, `a349fea8` 10 -> 11 |

Instruction sur deux films cuits un par un (`cmd/replay-build`, base contre lot, faits du dossier
d'équivalence, racine factice ; `scratchpad/L2/banc/`) : chemins JSON changés (hors chaîne de
révision) — `084a804d` : `coverage.continuousFire` (22 valeurs), `coverage.stances` (12),
`coverage.fallbacks` (10 : `repli_deadstate_hors_bande_bipede` 25 -> 27,
`repli_debut_de_liste_ferme_au_bit` 457 -> 494, `repli_liaison_par_anticipation` 920 -> 924,
`repli_physique_de_type_de_vehicule_supposee` 8 914 -> 8 915, `repli_record_desynchronise_jete`
29 -> 27), `coverage.birthLoadouts` (8) ; `000d5950` : `bursts` (7 -> 15 rafales), `coverage.continuousFire`,
`coverage.stances` (listes non localisées 263 -> 221, liaisons oubliées 2 053 -> 350),
`coverage.fallbacks` (`repli_deadstate_hors_bande_bipede` 4 -> 11, `repli_liaison_par_anticipation`
13 -> 14). Aucune piste (`tracks`), aucun tir hors rafale, aucune identité ne change sur ces deux
films. Cause : les records `ti=43` se lisent jusqu'au bout, les paquets qui s'arrêtaient sur eux
ferment ou lisent plus loin (§5).

Durées et pics (avant -> après, `l2_tsv/replay_equiv_durees_pics.txt`) : semblables à ± 10 %, sauf
`084a804d` 1 min 47 -> 2 min 10 (+21 %, pic 1,05 -> 1,08 Gio) et `696a9d7c` +20 % ; la passe
« avant » a tourné pendant les mutations et la carte (machine chargée), les écarts ne sont pas
attribuables au lot sans remesure. Le plafond du gate 4 ne s'applique pas à L2 (§6.0).

### 6.5 replay-corpus-gate (gate 6, joué en plus de la liste de la mission)

`replay-corpus-gate --reference=base --base=af6e93e23 --parc-root <copie> --work-root <scratchpad>
--json` depuis le worktree, binaire du lot (rapport `l2_tsv/corpus_gate.json`, sortie
`l2_tsv/corpus_gate_verdict.txt`). Écarts nécessaires, comme L0 : base explicite ; `--parc-root` sur
une COPIE du parc au scratchpad (base partagée et `metadata.duckdb` identiques à l'octet au checkout,
films et manifestes des 19 témoins) — la gate écrit son cache de base et son verrou sous le parc.
Première exécution tuée par la limite de durée des tâches de fond de la session (30 min, 18 bases
cuites) ; le worktree de base qu'elle laissait a été retiré (`git worktree remove`, sans `--force`,
arbre propre) ; seconde exécution détachée, base relue du cache : **rc 1**, `git worktree list`
propre ensuite.

- **Banc de vérité : 19 / 19 « ok »** (aucun oracle — kills, morts, assistances, score, équipes,
  vies — ni aucune classe de violation en défaut). `P-1` = les paquets fermés de la carte
  (`bcb6d393` 17 126, `fb1a1a72` 22 397, …).
- **Statut PERTE sur 14 témoins** (5 « ok » : `c75f33b8`, `bf15f7ab`, `bfecd02b`, `396cfc92`,
  `f75e7053`) : 188 métriques en gain, 57 en perte, 11 changements. Les pertes, instruites :
  - compteurs de trous du tir continu (`holesNotClosing` sur 12 témoins, `holesKind` 8,
    `holesBlockBC` 4, `holeRuns` 3, `innerHoles`, `heldHoleMs`, `burstsWithHole`, `noTrack`) : des
    paquets qui s'arrêtaient en vue B sur un `device-*` lisent maintenant jusqu'à la vue C et y
    échouent ; le trou change de classe. Mesuré sur `bcb6d393` : `holesNotClosing` 3 254 -> 4 035
    (+781) pour +11 296 paquets fermés, à rapprocher des « 780 paquets qui avancent jusqu'à la vue C
    et y sont arrêtés hors cadre » de BIS_2 §5.4 ;
  - compteurs de la marche des états (`forgottenBindings` 8, `refusedNews` 6,
    `refusedNewFalseReads` 5, `refusedNewUndecided` 2) et des dotations (`noDisplayable` 2,
    `noWeaponComponent` 1) : plus de records lus, plus de records jugés ;
  - **une valeur de contenu** : `bcb6d393` `stances/duree-totale` 7 184 -> 7 132, tout entière sur le
    slot 548, dont le dernier sprint finit à 2 181 au lieu de 2 235 (artefacts de base et de tête
    comparés, `scratchpad/L2/banc/`) ; changements de même nature sur `4f77afc1` (slots 558, 697,
    735 : durées 1 355 -> 1 328, 1 258 -> 1 256, 830 -> 814 ; +8 sprints, +7 sauts). Cause
    (estimée, non instruite paquet par paquet) : les paquets de ces films arrêtés par `i35` (HI_1_12_0)
    ou `i21` (`4f77afc1`) sont lus ; l'état de sprint, qui courait jusqu'au prochain paquet lu, est
    observé terminé plus tôt (règle « états à l'instant »). Aucun oracle ne départage.
- Le gate de corpus reste donc ROUGE dans ses propres termes (« perte » = sens de la métrique) ;
  aucune de ces pertes n'est un oracle en défaut. À trancher par l intégrateur (§7).

## 7. Recouvrements et écarts

- Écart : la réparation touche `debut_de_liste.go` (marche), hors de la liste de fichiers du plan
  pour L2. Elle est la condition du gate 2 (§2), générale et lue dans l'écrivain du masque.
- Écart : `capture.go` non touché (D-L2-10).
- Écart : le lecteur unique `FUN_140d580d0` migre trois copies hors de `ti=43` (minuteurs de manche,
  de campagne, de mise à mort douce), mêmes bits ; le titre de L3a annonce « helper `FUN_140d580d0` »
  : recouvrement à fusionner par l'intégrateur (mêmes noms que T7 §5, D-L2-8).
- Écart : `grammar.Rev` en `.2` et non en suffixe propre au lot (D-L2-9).
- Écart : killsource change sur un compteur de santé d'un film, `killsource.Rev` gardée (§6.3).
- Écart : `replay-corpus-gate` rc 1, statut PERTE sur 14 témoins (compteurs de trous et de marche, une durée de sprint sur `bcb6d393`, trois sur `4f77afc1`), banc de vérité 19 / 19 ok (§6.5). Le gate de corpus « sans perte » (§6.0 point 6) n est pas tenu à la lettre.
- Écart : le hook `pre-commit` (lefthook, `go vet` du module) tourne avec l environnement du hook, pas avec le `GOCACHE` dédié du lot.
- Mesuré au contrôle (§9.3) : la contribution marginale dans C11 reste en baisse sur `1c4c63c2`
  (−447 / −8 520) ; le lot n est pas retenu.

## 8. Découvertes

- **D-L2-1** L'« échec de L2 sur HI_1_10_0 » (BIS_2 382 / 401, R-COMB-2 D-105) n'est pas une
  grammaire `device-*` fausse : (a) la perte saine (`084a804d`, 4 paquets) vient d'un faux en-tête
  NEW `ti=43` (bits de la vue A) accepté comme tête de liste par la preuve par chaîne ; (b) les gains
  factices sont des têtes prises au second rang de `debutParFermeture` sur de faux NEW `ti=43` à
  masque aléatoire. Établi pour (a) par l'écrivain du masque, mesuré pour (b).
- **D-L2-2** La preuve par chaîne acceptait déjà, sans L2, des têtes dont le masque contredit
  l'écrivain : la réparation seule requalifie 33 paquets en sains (8 films, +770 utiles sains), 0 perdu.
- **D-L2-3** Sur HI_1_9_0 à HI_1_11_0, les records `ti=43` des paquets non sains portent les 23
  composants à comptes presque égaux (lectures au hasard) ; aucun NEW `ti=43` dans un paquet sain de
  HI_1_10_0. La validité du port pour un VRAI NEW `ti=43` de ce build n'a pas de témoin.
- **D-L2-4** (MESURÉ le 2026-10-03, §9.3 ; était estimé) Marginal dans C11 : −447 / −8 520 ; mécanisme : les mêmes faux NEW `ti=43` de tête
  (second rang), qui lient leur slot dans le monde.
- **D-L2-5** Le second rang de `debutParFermeture` (D-A) prend des têtes dont le masque contredit
  l'écrivain ; avec L2 il se déclenche plus souvent (comptes au §6.4). Le restreindre ferait perdre
  les 29 paquets de D-L0-2 ; non traité.
- **D-L2-6** `i31` échoue encore (N > 8) sur 11 paquets (`11de8353` 7, `1c4c63c2` 3, `60ae07c4` 1) :
  lectures probablement désalignées en amont (estimé).
- **D-L2-7** Un garde-rail qui lit les sources sur disque (`go/parser`) est invisible aux mutations
  par `-overlay` : il faut muter en place et rendre le fichier.
- **D-L2-8** Recouvrement L2 / L3a sur le lecteur `FUN_140d580d0`.
- **D-L2-9** `revision.ParserRevision` n'admet qu'un rang numérique, et `VerifierRangs` exige des
  rangs consécutifs : chaque lot de la vague prend `grammar-2026-10-02.2` ; l'intégrateur renumérote.
- **D-L2-10** Le plan cite `capture.go` pour L2 ; aucun composant `device-*` n'a de consommateur de
  valeur : rien à capter.
- **D-L2-11** Les fixtures de contrat (8 mini-films) ne changent qu'à la chaîne de révision : aucun
  de ces films ne lit un `device-*` qui modifierait le document.
- **D-L2-12** (mesuré, §9.3) Le second rang de `debutParFermeture` lit une liste CONTREDITE et lie
  ses NEW au monde. Un faux NEW `ti=43` que le port rend traversable y devient tête de liste ; un faux
  NEW `ti=45` lu derrière lui lie le slot 736, le vrai NEW du bipède 736 ne remplace pas cette
  liaison, et au moins 505 paquets sains de `1c4c63c2` (C11) se désynchronisent. La réparation de
  `debutParChaine` ne touche pas ce chemin. Reprise de L2 : la correction doit porter sur ce que le
  second rang lie (ou sur son admission), règle générale à fonder ; le retrait naïf des candidats à
  masque non écrit au second rang perd 23 sains sur `e5adf7b2` (oracle du contrôle).
- **D-L2-13** Sans L2, `FUN_140d580d0` reste recopié à trois sites (minuteurs de manche, de
  campagne, de mise à mort douce) : dette de la règle 6 antérieure au lot, hors périmètre.

## 9. Contrôle du 2026-10-03 : corrections C1 à C6

| Correction | Statut | Ce qui est fait |
|---|---|---|
| C1 vecteur V31c (N = 8) | [x] | `00001000` + 8 références de catégorie 1, porte attendue vraie, consommation exacte ; K3 ROUGE |
| C2 masques dense court et épars non croissant | [x] | NEW et delta, chaque règle avec son témoin écrit par le jeu ; K4 ROUGE |
| C3 marginale de L2 dans C11 | [x] mesurée, **[!] en défaut** | `1c4c63c2` −447 / −8 520 ; instruite au paquet (§9.3) : le lot n'est pas retenu |
| C4 valeurs de contenu au paquet | [x] | chaque changement classé sur pièces (§9.4) |
| C5 histoire hors des commentaires | [x] | deux en-têtes de tests réécrits en contrat ; histoire ci-dessous (§9.2) |
| C6 exactitudes | [x] | 216 lignes (§3) ; colonne de `consumeComposantsVueBM4b` ; « SEPT maillons » réécrit |

### 9.1 C1, C2 : vecteurs et mutations

- `components_device_ti43_test.go` : V31c, `N = 8` suivi de huit références de catégorie 1 (nulles et
  non nulles, sonde à 0 et à 1). Le jeu admet `N = 8` : `FUN_142f02a48` fait `if ((int)N < 9)`
  (`142f02b1e CMP R9D,0x8 ; JLE`, relu par le contrôle).
- `debut_de_liste_masque_test.go` : un monde dont les archétypes 2 et 4 portent huit composants d'un
  bit, pour que les index restent DANS l'archétype ; par pas (NEW, delta) un masque dense de sept
  composants et un masque épars à index non croissants, chacun avec son témoin (dense de huit, épars
  croissant) que la chaîne doit accepter : le témoin prouve que seul le masque refuse le cas.
- Mutations rejouées sur toute la suite `grammar` (`l2_tsv/mutations_controle.sh`, `.txt`, `-overlay`) :
  **16 / 16 ROUGES** — K1 à K12 du contrôle, dont **K3** (`n >= capaciteMoniteurs`, rouge sur V31c) et
  **K4** (seule la règle hors archétype, rouge sur « NEW, dense de sept composants ») ; en plus K4n et
  K4d (K4 sur un seul pas), K4e (épars non croissant accepté) et K4c (dense court accepté), chacune
  rouge sur le cas du pas et de la règle visés.

### 9.2 C5, C6 : commentaires

Histoire retirée du code (règle 17), consignée ici :
- `lecteur_140d580d0_guard_test.go` : `FUN_140d580d0` (deux `Q(n)` puis `R(5)`) était recopié à trois
  sites (minuteurs de manche, de campagne et de mise à mort douce) ; le lot L2 en ajoutait un
  quatrième par `FUN_142ba78dc` (`i37` de `ti=43`) ; un seul lecteur depuis L2. L'en-tête dit
  maintenant le contrat : chaque lecteur de production qui cite l'une des deux fonctions appelle
  `consume140d580d0` ou `consume142ba78dc`.
- `ecs_widths_guard_test.go` : 2026-10-02 (lot L2) : 123 -> 138 largeurs FIXES, 66 gardées
  inchangées ; quinze lignes `ti=43` entrent par le haut (`i19` à `i40` à largeur constante depuis
  `non_porte`, et `i18` `device-position-component`, porté de longue date mais sans `bits_typ`) ; les
  huit autres sont gardées par le flux. Le commentaire dit maintenant ce que les constantes comptent.
- `dispatch_object.go` : colonne réalignée ; « Cette exemption vaut pour les SEPT maillons de la
  chaîne » était ambigu depuis M4b (la chaîne a dix maillons) : sept dépassent 80 lignes et portent
  `funlen` (mesuré : `consumeNavpointComponent` 32 lignes, `consumeComposantsDispositif` 48,
  `consumeComposantsVueBM4b` 22) ; réécrit, avec « Chaque maillon dépasse 80 lignes » devenu « Les
  sept premiers ». La phrase historique « le lot 2.7 l'a coupé en SEPT maillons » est vraie et reste.

### 9.3 C3 : la marginale de L2 dans C11, rejouée (mesuré)

**Surcouche unique reconstruite sur `af6e93e23`.** Chaque fichier de `surcouche_unique_postj12/` est
fusionné à trois voies (`git merge-file`, `ours` = `af6e93e23`, `base` = `df228c24c`, `theirs` = la
copie de la surcouche) : 8 fusions sans conflit ; L0 avait changé `traverse.go` et `frame_infer.go`,
et la fusion n'y ajoute que le delta de la surcouche (15 et 2 lignes, identiques au delta
d'origine). `rcomb2_leviers.go` est repris tel quel. Deux binaires de `TestRComb2` (tags
`research,campagne_overlay`) : TÊTE (code du lot) et BASE (sources de `af6e93e23` pour les dix
fichiers que L2 change, tests de L2 retirés). Passe 2 (C11 = sans L7, `CAMPAGNE_RCOMB2_RETIRES=L7`),
mêmes variables que R-COMB-2 (`l2_tsv/c11_lancer.sh`). Le L2 DE PRODUCTION se lit dans la
configuration `full-L2` du binaire de tête (le levier de recherche est éteint, la production lit les
`device-*`) ; C11 sans L2 est `full-L2` du binaire de base ; la copie de recherche de R-COMB-2 est
`full` du binaire de base. « Sains » = fermés au sens de L0 (au bit près, aucune règle de l'écrivain
contredite) ; le juge à trois invariants de la sonde n'en contredit aucun. Contrôle : la référence
redonne la carte v2 (`1c4c63c2` 13 333 / 13 341, `084a804d` 4 773 / 4 779).

| Film | C11 sans L2 | C11 + L2 de production | **Marginale** | Sans l'oracle L9 | C11 + L2 de recherche (base) |
|---|---|---|---|---|---|
| `084a804d` | 25 481 / 689 456 | 25 484 / 689 504 | **+3 / +48** | +3 / +48 | +2 / +18 |
| `111fa685` | 13 274 / 279 551 | 13 275 / 279 575 | **+1 / +24** | +1 / +24 | +1 / +24 |
| `1c4c63c2` | 44 979 / 953 041 | 44 532 / 944 521 | **−447 / −8 520** | −446 / −8 502 | −476 / −9 160 |

(paquets sains / records utiles sains ; `l2_tsv/c11_hi_1_10_0.tsv`). **La baisse persiste : au sens
de D19, L2 n'est pas réparé.** (À la base `df228c24c` et au juge à trois invariants, R-COMB-2
mesurait −916 / −16 120 ; la base et le juge ont changé avec L0, d'où l'écart.)

**Instruction paquet par paquet** (`1c4c63c2`, `full-L2`, sonde `l2_tsv/sonde_c3.diff`, détail
`l2_tsv/c11_1c4c63c2_sains_perdus.tsv`) : 571 sains perdus, tous devenus NON fermés, 124 gagnés.
- 501 perdus ont la même cause : `ti=45 i0 matchflow-sequence-data-component` ; le record qui
  désynchronise est, dans les 501, un delta du slot 736 lu sous l'archétype `ti=45`. Dans la base, le
  même delta est lu sous `ti=35` (bipède), sans désynchronisation. Chunks 55 (461), 54 (55), 56.
- L'origine est le paquet 54:396 (`l2_tsv/c11_1c4c63c2_paquet_54_396.tsv`). Base : sain, liste prise
  au bit 260 (un delta), 24 records utiles. Tête : le localisateur strict ne trouve rien ; la liste
  est prise au bit 2 368 sur `NEW slot=3459 ti=43`, fermée au bit près mais contredite (« masque
  au-delà de l'archétype ») : c'est la définition du second rang de `debutParFermeture` (le chemin de
  la composition LS × L1a qui l'y appelle n'est pas tracé : estimé). Les records lus de cette liste :
  `NEW 3459 ti=43`, `NEW 442 ti=47`, **`NEW 736 ti=45`** — le slot 736 est lié à `ti=45`.
- Au paquet 54:1970, le vrai `NEW slot=736 ti=35` est lu (paquet sain) mais les deltas du slot 736
  restent lus sous `ti=45` à partir de 54:1972 : un NEW qui contredit une entité vivante ne remplace
  pas la liaison (règle de `frame_infer.go`, établie par lecture ; la liaison n'est pas tracée par la
  sonde).
- Les 70 autres : 4 lisent aussi 736 sous `ti=45`, 43 sont dans les chunks 54 à 56 (sortie par rejet,
  masque ; même contamination, estimé), 23 ailleurs (chunks 8, 52, 53, 59, 65 ; non instruits).

**Verdict D2** : ces paquets étaient SAINS dans la base (fermés, aucune règle contredite) ; ils sont
perdus parce qu'une liste contredite, prise au second rang, lie un faux NEW au monde. Ce n'est pas
une fermeture factice retirée : la perte n'est pas admissible. La réparation de L2 (`debutParChaine`)
ne vise pas ce chemin. En production seule, le même mécanisme est actif : les listes prises au
second rang passent de 457 à 494 sur `084a804d`, de 6 039 à 6 377 sur `1c4c63c2`, de 991 à 1 009 sur
`4f77afc1`, de 4 à 5 sur `bcb6d393` (sonde C4) ; il ne fait pas baisser la carte de L2 seul, mais
c'est lui qui porte les changements publiés de HI_1_10_0 (§9.4).

### 9.4 C4 : les valeurs de contenu, instruites au paquet (mesuré)

Méthode : `cmd/replay-build` de base et de tête, construits avec une sonde en surcouche
(`l2_tsv/sonde_c4.diff`, jamais dans le code) qui écrit, pour chaque paquet de la marche des états
de mouvement, la classe de sa liste (sans liste, strict, chaîne, fermeture au premier rang, au
second rang, non localisée) et le verdict de sa vue C ; pour chaque lecture d'état retenue et chaque
transition de saut dérivée, son paquet ; pour chaque dotation de naissance, ce qui la confirme.
Racine factice au scratchpad (films identiques à l'octet au cache du checkout, `config/` identique
au worktree), faits du gate (`bcb6d393`, `4f77afc1`) et du dossier d'équivalence (HI_1_10_0).
Contrôle : les `stances` produits sont identiques à ceux du gate (`bcb6d393` base et tête,
`4f77afc1` tête). Détail : `l2_tsv/c4_analyse.txt`, scripts `c4_cuire.sh`, `c4_analyser.sh`.

| Film | Changement publié | Paquet qui le porte | Base | Tête | Classe |
|---|---|---|---|---|---|
| `bcb6d393` | slot 548, sprint t1 2 235 -> 2 181 | 12:1334, sprint levé | non localisée (aucune levée lue : l'intervalle courait jusqu'à la fin de la vie) | fermeture rang 1, fermé | **paquet sain** |
| `bcb6d393` | slot 541, sprint 1 762-1 763 (neuf, non relevé par l'exécutant) | 10:1104 | non localisée | rang 1, fermé | **paquet sain** |
| `4f77afc1` | slot 558, sprint 10 208-10 255 -> 10 208-10 219 + 10 247-10 255 | 56:454, sprint levé | non localisée | rang 1, fermé | **paquet sain** |
| `4f77afc1` | slot 564, sprint 10 219-10 221 (neuf) | 56:454 | non localisée | rang 1, fermé | **paquet sain** |
| `4f77afc1` | slot 697, sprint 8 069-8 086 -> 8 069-8 071 + 8 074-8 086 | 45:782, sprint levé | rang 1, fermé, liste au bit 7 864 | rang 1, fermé, liste au bit 236 (ses premiers records sont lus) | **paquet sain** |
| `4f77afc1` | slot 718, sprint 8 071-8 080 (neuf) | 45:782 | idem | idem | **paquet sain** |
| `4f77afc1` | slot 735, sprint 8 166-8 241 -> 8 166-8 221 + 8 231-8 241 | 46:480, sprint levé | non localisée | rang 1, fermé | **paquet sain** |
| `4f77afc1` | slot 735, saut 8 221-8 226 -> 8 221-8 225 ; saut 8 248-8 253 retiré | 46:478 (strict), 46:642 (sans liste) | non fermés (sortie par rejet) | fermés | **paquets sains** |
| `4f77afc1` | sprints neufs 539, 603, 747 ; sauts neufs 676, 678 | 19:16, 19:14, 46:254, 31:920, 31:978 | non localisée, sans liste non fermé, rang 1 (liste au bit 5 656) | rang 1 ou sans liste, fermés (747 : liste au bit 123) | **paquets sains** |
| `084a804d` | dotation de naissance 15 -> 16 (`closed` ; non publiée : `noDisplayable` 15 -> 16) | création 43:198, slot 558 | non confirmée | confirmée par un delta du slot 1 536, `ti=43`, que le port décode | **ni l'un ni l'autre** : paquet NON LOCALISÉ dans les deux marches ; le record qui confirme n'est jugé par aucune fermeture |
| `1c4c63c2` | dotation de naissance 27 -> 28 (non publiée : `noDisplayable` 27 -> 28) | création 54:1940, slot 733 | non confirmée | confirmée par un delta du slot 4 637, `ti=43` | **ni l'un ni l'autre** (idem) |
| `1c4c63c2` | slot 640, accroupi 2 802-2 805 (neuf) | 16:2276 | non localisée | rang 2, contredit (sortie par rejet), liste au bit 2 743 | **liste contredite (second rang)** |
| `1c4c63c2` | slot 737, escalade 5 138-5 396 ajoutée, 10 568-10 617 retirée | 5:654 (escalade, sprint levé) | rang 2 | rang 2, contredit (sortie par rejet) | **liste contredite (second rang)** |
| `1c4c63c2` | sauts neufs 745 (5 226-5 231), 746 (10 469-10 473) | transitions attribuées à 29:166 et 55:644 (sans liste, fermés des deux côtés) ; échantillons de vitesse neufs de 29:192, 32:154, 55:702, 58:84 | non localisée ou rang 2 | rang 2, contredits (masque au-delà de l'archétype) | **liste contredite (second rang)** |
| `084a804d` | lectures 12 552 -> 12 553, aucun intervalle publié ne change | 31:980 (escalade, slot 733) | non localisée | rang 2, contredit | liste contredite (second rang), sans effet publié |

Lectures retenues sans effet publié sur `4f77afc1` : 516 (48:850, sans liste, tête « sortie par
rejet »), 583 et 656 (rang 2, contredits), 678 (31:1028 et 31:1040, rang 1, sains). Sur `bcb6d393`,
557 (16:1212, rang 1, sain). **Bilan** : sur `bcb6d393` et `4f77afc1` (gate 6), tout changement publié
vient d'un paquet SAIN (premier rang ou sans liste, fermé) ; sur HI_1_10_0, tout changement publié de
la marche des états vient d'une liste CONTREDITE prise au second rang (le mécanisme de §9.3), et les
deux dotations neuves sont confirmées par un record `ti=43` d'un paquet non localisé (non publiées).

### 9.5 Gates rejoués (corrections C1, C2, C5, C6 ; aucun code de marche changé)

| Gate | Sortie |
|---|---|
| `gofmt -l` film et cmd | vide |
| `go vet` film, `go vet -tags=research` film | rc 0, rc 0 |
| G-film (film/..., replaybuild, killcollector, `-count=1`) | 19 paquets `ok`, rc 0 |
| `go test ./internal/archlint/` | `ok` (44,7 s) |
| Mutations | 16 / 16 rouges (§9.1) |
| Carte v2 (`1c4c63c2`, `084a804d`, `111fa685`, `bcb6d393`, `4f77afc1`), binaire des corrections | `fermeture_paquets.tsv` identique paquet par paquet à celui de l'exécutant et à celui du contrôle |
| killsource | non rejoué : aucun code de marche ne change (tests et commentaires) |

Le commit suivant retire le code du lot : l'arbre de code revient à `af6e93e23` ; ses gates sont
dans son message.
