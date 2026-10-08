# PLAN — Arrêts de la vue B, suite (lot 2026-10-08)

> Suite du lot `.ai/V7.5/film_re/arrets_vue_b_2026-10-07/` (rapport §4, découvertes D1 à D5), sous le
> contrat `plan-execution`. GO de l'utilisateur le 2026-10-08. Worktree
> `LevelUp-wt-grammaire-arrets-vue-b-2`, branche `feat/grammaire-arrets-vue-b-2`, base `a4515e66c`
> (`feat/v75` : `grammar-2026-10-08`, `SchemaVersion` 89, `killsource-2026-10-07.2`, faits 10).
> Méthode, gates et garde-fous : handoff `.ai/HANDOFF_COMPOSANTS_BLOQUANTS_VUE_B_2026-10-06.md`
> §4 à §6. Aucune fusion, aucune cuisson du parc.

## Méthode par composant

Lecture du désérialiseur dans le jeu (Ghidra lecture seule, `HaloInfinite.exe` HI_1_13_0, HTTP direct
`127.0.0.1:8089`) ; portage ; vecteur de test d'après l'écrivain ; `ecs_table.tsv` ; un rang
`grammar.Rev` (`grammar-2026-10-08.2`, `.3`, …) ; un commit ; carte v2 (20 films) de l'état retenu
précédent contre la tête, gate 2 (aucun film en baisse) AVANT le composant suivant. Un composant qui
fait baisser un film : cause instruite ; hors de la lecture : retiré et consigné.

## Items — périmètre (1)

| # | Composant | Statut | Commit | Note |
|---|---|---|---|---|
| R0 | Rotation de la chronique : entrées `grammar-2026-10-03` et suivantes versées dans `rev_chronique_archive_8.go` | [x] | E1 | au premier rang ; listes `fichiersDeChroniqueGrammar` et `horsCouche` étendues |
| E1 | `ti=12 i20`..`i27 managed-navpoint-visual-state-groups-component-0..7` (1 675 arrêts) | [x] | `ec54f6fd5` | `FUN_140dbe1bc` = R(1) présence + R(32) + `FUN_140dbe25c(v=1)` ; carte v2 contre la base : +6 617 sains, +131 689 utiles, 0 perdu, aucun film en baisse ; `grammar-2026-10-08.2` |
| E2 | `ti=10 i22 managed-object-interaction-filter-component` (299) | [x] | `a89373059` | `FUN_140dbdf5c` = `FUN_140dbe400(v = 1 < param_4)`, le bloc de filtres seul (comme `ti=12 i5`, `i6`) ; carte v2 contre E1 : +89 sains, +2 252 utiles, 0 perdu, aucun film en baisse ; arrêt suivant `ti=10 i23 flags` (201) ; `grammar-2026-10-08.3` |
| E3 | `ti=35 i59 biped-spartan-ability-non-predicted-state` (250, et `i58` 1) | [x] | `eed9347c1` | corps `FUN_142f25e90` relu dans le jeu, ses huit étiquettes portées (la grammaire mesurée du 2026-08-16 remplacée) ; carte v2 contre E2 : +268 sains, +6 670 utiles, 0 perdu, aucun film en baisse ; images-clés `ti=35` (20 films) : 544 + 32 arrêts sans la portée, 574 + 29 avec, 0 et 0 après ; aucun fichier interdit touché ; `grammar-2026-10-08.4` |
| E4 | `ti=11 i4 managed-objective-interaction-filter-component` (382 à l entrée, 165 en base) | [x] | `ced5b3995` | `FUN_140dbe170` = `FUN_140dbe400(v = 1 < param_4)`, le bloc de filtres seul ; carte v2 contre E3 : +6 312 sains, +80 762 utiles, 0 perdu, aucun film en baisse ; `grammar-2026-10-08.5` |
| E5a | `ti=10 i23 managed-object-flags-component` (201) | [x] | `d23a7bdab` | `FUN_1410d9b5c` -> `FUN_140f72efc` = R(2) ; écrivain `142edb23c` -> `FUN_142ed0ec8` ; carte v2 contre E4 : +4 957 sains, +108 064 utiles, 0 perdu, aucun film en baisse ; `grammar-2026-10-08.6` |
| E5b | `ti=12 i17 managed-navpoint-object-marker` (92) | [x] | `3b01c5d59` | `FUN_141169e68` = R(32) ; écrivain `142edb084` -> `FUN_1407edaf4` ; carte v2 contre E5a : +140 sains, +3 768 utiles, 0 perdu, aucun film en baisse ; `grammar-2026-10-08.7` |
| E5c | `ti=10 i18`..`i21 managed-object-networked-property-component` (40) | [x] | `7f6e69d69` | `FUN_142ed5358` = R(32) vers `etat + 0x54 + 4 * index` ; écrivain `142edb3a4` ; carte v2 contre E5b : +1 sain, 0 perdu, aucun film en baisse ; les 40 paquets avancent puis s arrêtent ailleurs (20 rejets, 9 terminateurs hors cadre, 8 fins de payload, D-E5c) ; `grammar-2026-10-08.8` |
| E5d | `ti=45 i1 matchflow-focus-data-component` (16) | [x] | `705c71dda` | `FUN_141167744` = R(6) + R(4), signés ; écrivain `142edbda4` ; carte v2 contre E5c : +10 sains, +57 utiles, 0 perdu, aucun film en baisse ; `grammar-2026-10-08.9` |
| E5e | `ti=12 i13 managed-navpoint-top-progress` (7) | [x] | `de6866f54` | `FUN_142ed51d8` = R(8) quantifié ; écrivain `142edb134` -> `FUN_142ed18e8` ; carte v2 contre E5d : 0 sain gagné ni perdu, aucun film en baisse ; les 7 paquets avancent jusqu à `i15` (1) et `i19` (4) ; `grammar-2026-10-08.10` |
| E5f | `ti=12 i15 managed-navpoint-bottom-progress` (4 à l entrée) | [x] | `9a8ae5058` | `FUN_142ed4fe4` = R(8) quantifié, le lecteur de `i13` ; écrivain `142edadf0` -> `FUN_142ed18e8` ; carte v2 contre E5e : 0 sain gagné ni perdu ; les 4 paquets finissent en fin de payload (D-E5c) ; `grammar-2026-10-08.11` |

## Gates du lot entier (base `a4515e66c`)

| # | Gate | Statut | Sortie |
|---|---|---|---|
| G1 | carte v2 base contre tête, gate 2 | [x] | +19 440 sains (487 188 -> 506 628), +349 211 utiles, 0 perdu, 0 film en baisse (`tsv/gate2_lot_contre_base.tsv`) |
| G2 | gate 3 : `killsource json` (19 témoins + `1c4c63c2`) | [x] | 7 identiques, 6 diagnostic `calibration` seul, 7 avec des morts qui passent du balayage à la marche (81 morts, contenu identique) et `0797ce72` un candidat de santé de plus ; `read_path` persisté : `killsource-2026-10-08` (rotation de la chronique : archive_2) ; `tsv/gate3_killsource.tsv` |
| G3 | `TestGoldenFilms` | [x] | 4 / 4 ok après régénération de `fccc61cd.golden` : seul le score de l oracle de `calibration` change (1205 -> 1211) |
| G4 | gate de corpus sur une copie du parc, chaque FAUX / PERTE instruit, 0 MANQUE | [!] | banc 17/19 ok ; FAUX 084a804d admis par le pilote ; MANQUE 0797ce72 (P2, image-clé) soumis à l utilisateur ; PERTE instruites (RAPPORT §4) |
| G5 | gofmt, vet (normal, research, integration), archlint | [x] | vide ; rc 0 ×3 ; archlint ok |
| G6 | golangci-lint 0 issue | [x] | 0 issues (film et module) |
| G7 | mutations rouges | [x] | 16 / 16 |
| G8 | baseline des tests (aucun test retiré ni renommé) | [x] | un test remplacé, absent de la baseline ; 5 ajoutés |
| G9 | `make gate-push` (TMP court dédié) | [x] | par étapes : lint 0, web vert, baseline 9 532 / 9 532, 0 échec |
| G10 | push + CI | [~] | push autorisé par le pilote ; état de la CI dans le message de clôture |

## Périmètre (2) et (3) : en attente du feu vert du pilote

| # | Item | Statut |
|---|---|---|
| P2 | message de dégâts (genre 0) mal lu, fin de vue A trop tôt (découverte 34 de levelup-57) | [x] `39510278d` : genre 0 bien lu (FUN_1407f15a4 = port) ; cause dans la vue B : world-object i0 porte posée lue aux largeurs de la carte (exception GA2-5) ; correctif hors portée (table DÉFAUT), accord levelup-57 + utilisateur ; fccc61cd +10 949 sains, 599 / 635 trames à genre 0 fermées ; 20 films inchangés ; `grammar-2026-10-08.12` |
| P3 | NEW de bipède lu dans une trame non fermée (`bf15f7ab`, slot 553, découverte 45) | [x] `e72132295` : le NEW est lu dans une trame FERMÉE et refusé par `contreditUneEntiteVivante` (slot lié en dur à `ti=20` gén. 0 par l image-clé, DEL jamais lu) ; règle : un NEW refusé est lié en fin de trame quand la fermeture prouve son en-tête (`neufs_prouves.go`) ; `bf15f7ab` 14:1094-1230 : 69 / 69 trames fermées (30 avant) ; carte v2 +1 046 sains, 0 perdu ; `rendParLAncrage` inchangé ; `grammar-2026-10-08.13` |

## Journal

- 2026-10-08 : plan écrit ; carte v2 de base (binaire de `a4515e66c`, 1 min 21 pour 20 films) :
  causes identiques à la tête d'intégration du lot précédent (`ti=12 i21` 679, `i22` 534, `i20` 435,
  `ti=10 i22` 292, `ti=35 i59` 250, `ti=11 i4` 165, `ti=12 i17` 92).

- E1 : lecteur `FUN_140dbe1bc` trouvé par la table de noms `143d07f00` (huit pointeurs) -> accesseur `14064c620` (index en `descripteur + 8`) -> descripteur `143d081c8 - 0x28` : écrivain `142edb178` (slot `+0x10`), lecteur `140dbe1bc` (après le thunk `14076ce9c`). R(1) présence (`FUN_1406cf008`) ; présent : R(32) puis `FUN_140dbe25c(v = 1)` = bloc de filtres `FUN_140dbe400` (déjà porté, [consumeFilterSet]), R(32), un R(32) par filtre présent, K entrées d'ordre de 3 bits. Écrivain `FUN_142c94dd4` dans le même ordre. Statut `partiel` (tag 15 arrête, comme `i2`..`i6`). Carte v2 contre la base : +6 617 sains, 0 perdu ; des 1 675 paquets arrêtés sur `i20`..`i25`, 1 167 deviennent sains, 216 s'arrêtent sur `ti=11 i4`, 210 sortent par rejet. Rotation R0 faite au même rang.
- 2026-10-08 : feu vert du pilote pour (2) puis (3), à traiter après (1) ; liste complète des fichiers interdits reçue (lot LK de levelup-57) ; annonces à levelup-57 seulement pour les passes de plus de 15 min.

- E2 : nom `143c94b40` -> accesseur `141177ff0` -> descripteur (accesseur en `143d09308`) : niveau `141179610` (`MOV EAX, 2`), écrivain `142edb250` (`etat + 0x68`, saut vers `FUN_142c7023c`), lecteur `140dbdf5c` : `FUN_140dbe400(etat + 0x68, flux, v = 1 < param_4)`, la même forme que `ti=12 i5`/`i6` et que `ti=11 i4` (`140dbe170`, `etat + 0x48`, écrivain `142edb5cc`). Le commentaire du dépôt qui disait la queue de `FUN_142c7023c` « de largeur inconnue » (appel virtuel par tag) parlait de l écrivain du bloc de filtres, porté depuis le lot 5.1.1 (`consumeFilterSet`). Carte v2 contre E1 : +89 sains, 0 perdu.

- E3 : lecteur `FUN_142f02994` -> `FUN_142f2679c` (R(2), corps si 3) -> `FUN_142f25e90` relu entier : `FUN_142f21c0c` (R(3) + 1), préfixe `FUN_142f26e40` (`FUN_1408f0ac4` catégorie 1, puis `FUN_142f04664(c = référence présente)` : sans référence, `FUN_14076e494` au niveau 0x10), `FUN_14297ea84` (R(6)), puis la branche de l étiquette (1 à 6 ; 7 et 8 sans charge). Écrivain `FUN_142f05660` -> `FUN_142f27930` -> `FUN_142f272ac` dans le même ordre. La grammaire mesurée lisait les mêmes bits sur les corps à portes fermées (Zero3 = porte de référence + porte de position + index ; Mid7 = R(6) + porte de la première référence). Publication : une ancre de grappin n est publiée que si sa position est lue à un index de plage (`PosCarte`) ; `consume142f04664` rend sa position. Mini-bobine instruite par sonde en surcouche : trames 3:636 et 3:722 passent le slot 513 et se ferment (prouvées) ; +1 record bipède ; les records des slots 1525 à 1535 y sont des `ti=37`, que la passe des armes au sol rendait aussi en `ti=42` (`worldObjects_ti42` 54 -> 53) : correct (règle de 2.7.d). Le ratchet de fermeture des images-clés (golden interdit) ne baisse pas. Carte v2 contre E2 : +268 sains, 0 perdu.

- E4 : nom `143c95338` -> accesseur `141177f90` -> descripteur (accesseur en `143d090d0`) : niveau `141179610` (2), écrivain `142edb5cc` (`etat + 0x48`, saut vers `FUN_142c7023c`), lecteur `140dbe170` = `FUN_140dbe400(etat + 0x48, flux, v = 1 < param_4)`. Le commentaire « queue d appel virtuel de largeur inconnue » (`components_managed_objective.go`, `dispatch_biped.go`, `objective_scan.go`) portait sur l écrivain ; corrigé. Carte v2 contre E3 : +6 312 sains, 0 perdu.

- (2) : chargeDegatsApres confrontée au lecteur `FUN_1407f15a4` et à l écrivain `FUN_142f19d34` (descripteur `143d0f978`, domaines 1,1,7 en `14080a018`) : identique. 4 films de killsource : 635 trames à genre 0 non fermées ; 543 avec S0 = E ; 11 où un début plus loin ferme, la marche depuis E y lit 6 à 8 records cohérents sur 2 700 à 3 900 bits (E est le vrai début) ; recherche exhaustive des fins de message : aucune chaîne ne mène au début localisé. Dernier record avant l échec : 437 fois `ti=38` (i0 + i2) dans `fccc61cd`, i0 precHigh 0, porte 1, 53 bits : l exception datée world-object i0 lisait les largeurs de la carte malgré la porte. Niveau de la ligne de table : immédiat `0x10` (`LEA R8D,[R9+0x10]` en `14076e2bc`), aucune dépendance au registre. Correctif hors portée (accord levelup-57 et utilisateur) ; deux baisses ti=37 d images-clés justifiées dans le golden. 78919882 : échecs après des records `ti=1`/`ti=0` du moteur (slot 4), non instruits plus avant.
- (3) en cours : `bf15f7ab` slot 553 : le NEW `ti=35` gén. 1 de 14:1094 est lu dans une trame FERMÉE (et non « non fermée ») ; il est refusé par `contreditUneEntiteVivante` car le monde tient le slot lié en dur à `ti=20` gén. 0 (image-clé 14:6) sans DEL lu. 829 refus sur les 20 films (`p3_refus_20films.txt`).

- Gates du lot : RAPPORT §4. Gate de corpus en 4 passes sur une copie du parc (scratchpad) ; bisections par `--base` sur `0797ce72` et `084a804d`.

## Découvertes

Voir RAPPORT.md.
