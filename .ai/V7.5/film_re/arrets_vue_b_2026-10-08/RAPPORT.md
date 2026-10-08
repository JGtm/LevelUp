# RAPPORT — Arrêts de la vue B, suite (lot 2026-10-08)

> Suite du lot `.ai/V7.5/film_re/arrets_vue_b_2026-10-07/` sous le contrat `plan-execution`. GO de
> l'utilisateur le 2026-10-08 ; feu vert du pilote pour (2) et (3) le même jour. Worktree
> `LevelUp-wt-grammaire-arrets-vue-b-2`, branche `feat/grammaire-arrets-vue-b-2`, base `a4515e66c`.
> Aucune fusion, aucune cuisson du parc réel. Plan et journal : `PLAN.md`.
>
> Conventions : **lu** = lu dans le jeu (Ghidra, `HaloInfinite.exe` HI_1_13_0, lecture seule, HTTP direct
> `127.0.0.1:8089`) ; « sain » = paquet fermé que le juge ne contredit pas (colonne `ferme` de la carte v2,
> 20 films du kit).

## 0. Statut

| # | Objet | Commit | Rang `grammar` | Sains gagnés | Perdus |
|---|---|---|---|---|---|
| E1 | `ti=12 i20`..`i27 managed-navpoint-visual-state-groups-component-0..7` | `ec54f6fd5` | `.2` (+ rotation archive_8) | +6 617 | 0 |
| E2 | `ti=10 i22 managed-object-interaction-filter-component` | `a89373059` | `.3` | +89 | 0 |
| E3 | `ti=35 i59 biped-spartan-ability-non-predicted-state` (corps, 8 étiquettes) | `eed9347c1` | `.4` | +268 | 0 |
| E4 | `ti=11 i4 managed-objective-interaction-filter-component` | `ced5b3995` | `.5` | +6 312 | 0 |
| E5a | `ti=10 i23 managed-object-flags-component` | `d23a7bdab` | `.6` | +4 957 | 0 |
| E5b | `ti=12 i17 managed-navpoint-object-marker` | `3b01c5d59` | `.7` | +140 | 0 |
| E5c | `ti=10 i18`..`i21 managed-object-networked-property-component` | `7f6e69d69` | `.8` | +1 | 0 |
| E5d | `ti=45 i1 matchflow-focus-data-component` | `705c71dda` | `.9` | +10 | 0 |
| E5e | `ti=12 i13 managed-navpoint-top-progress` | `de6866f54` | `.10` | 0 | 0 |
| E5f | `ti=12 i15 managed-navpoint-bottom-progress` | `9a8ae5058` | `.11` | 0 | 0 |
| P2 | world-object i0, porte posée -> table DÉFAUT hors portée | `39510278d` | `.12` | 0 (20 films) ; fccc61cd +10 949 | 0 |
| P3 | un NEW que la fermeture de sa trame prouve crée son entité | `e72132295` | `.13` | +1 046 | 0 |
| G2 | décision `killsource.Rev` | `dbdfc3320` | — | — | — |

**Lot entier contre `a4515e66c` (carte v2, 20 films) : +19 440 paquets sains (487 188 -> 506 628),
+349 211 records utiles sains, 0 perdu, aucun film en baisse** (`tsv/gate2_lot_contre_base.tsv`).

## 1. Partie (1) : composants

Méthode (D7 du lot précédent) : nom -> chaîne `.rdata` -> accesseur de nom -> descripteur ; écrivain au slot
`accesseur + 0x10`, lecteur au slot `accesseur + 0x28` (après le thunk `FUN_14076ce9c`) ; niveau en
`accesseur - 0x08`. Désassemblage local : octets lus par `read_memory`, désassemblés par `objdump`.

- **E1** `i20`..`i27` : table de noms `143d07f00` (huit pointeurs) -> accesseur `14064c620` (index en
  `descripteur + 8`, un seul descripteur `143d081a0`). Lecteur `FUN_140dbe1bc` : R(1) présence
  (`FUN_1406cf008`) ; présent : R(32) (`etat + 0x850` pour le groupe 0), puis `FUN_140dbe25c(v = 1)` : bloc de
  filtres `FUN_140dbe400` (déjà porté, `consumeFilterSet`), R(32), un R(32) par filtre présent, K entrées
  d'ordre de 3 bits. Écrivain `142edb178` -> `FUN_142c94dd4` (`FUN_142c7023c`, mot, mots, ordre). Variable,
  statut `partiel` (tag 15). Des 1 675 paquets arrêtés sur `i20`..`i25`, 1 167 deviennent sains.
- **E2** `ti=10 i22` : nom `143c94b40` -> accesseur `141177ff0` ; niveau `141179610` (2) ; lecteur
  `FUN_140dbdf5c` = `FUN_140dbe400(etat + 0x68, v = 1 < param_4)` ; écrivain `142edb250` -> `FUN_142c7023c`.
- **E3** `ti=35 i59` : `FUN_142f02994` -> `FUN_142f2679c` (R(2), corps si 3) -> `FUN_142f25e90` relu entier :
  étiquette `FUN_142f21c0c` = R(3) + 1 ; préfixe `FUN_142f26e40` = `FUN_1408f0ac4`(catégorie 1) puis
  `FUN_142f04664(c = référence présente)` (sans référence : `FUN_14076e494` niveau 0x10) ; R(6)
  (`FUN_14297ea84`) ; étiquette 1 : `FUN_1407f08bc` ; 2 : cat. 5 + `FUN_1407f08bc` ; 3 : cat. 0, cat. 5,
  3 vecteurs (`FUN_142f26e9c`, R(24)+R(12) derrière une porte), R(24), R(9) ; 4 et 5 : cat. 5, 1 vecteur,
  `FUN_14076e494` (CALL `142f2605d`), R(24), R(9) ; 6 : `FUN_1407f08bc`, cat. 5 si sa porte est fermée, cat. 0,
  2 vecteurs, R(1), R(24) ; 7 et 8 : rien. Écrivain `FUN_142f05660` -> `FUN_142f27930` -> `FUN_142f272ac`
  dans le même ordre. La grammaire MESURÉE du 2026-08-16 (Zero3, Mid7) lisait les mêmes bits sur les corps
  à portes fermées. Publication : une ancre n'est publiée que lue à un index de plage (`PosCarte`).
  Mini-bobine (sonde en surcouche) : les trames 3:636 et 3:722 passent le slot 513 et se ferment ; +1 record
  bipède ; la passe des armes au sol ne rend plus les records `ti=37` de ces trames (`worldObjects_ti42`
  54 -> 53). Images-clés `ti=35` (20 films, sonde) : 544 + 32 arrêts sans la portée, 574 + 29 avec, 0 après.
  Aucun fichier du lot LK touché.
- **E4** `ti=11 i4` : nom `143c95338` -> accesseur `141177f90` ; lecteur `FUN_140dbe170` =
  `FUN_140dbe400(etat + 0x48, v = 1 < param_4)` ; écrivain `142edb5cc`. L'« appel virtuel de largeur
  inconnue » qui l'avait laissé dehors était la queue de l'ÉCRIVAIN du bloc de filtres : commentaires corrigés.
- **E5a** `ti=10 i23` : lecteur `FUN_1410d9b5c` -> `FUN_140f72efc` = R(2) ; écrivain `142edb23c` -> `FUN_142ed0ec8`.
- **E5b** `ti=12 i17` : lecteur `FUN_141169e68` -> `FUN_14080dec4` = R(32) ; écrivain `142edb084` -> `FUN_1407edaf4`.
- **E5c** `ti=10 i18`..`i21` : lecteur `FUN_142ed5358` = R(32) vers `etat + 0x54 + 4 * index` ; écrivain
  `142edb3a4`. Les 40 paquets avancent et s'arrêtent en aval (D-E5c).
- **E5d** `ti=45 i1` : lecteur `FUN_141167744` = R(6) puis R(4), signés ; écrivain `142edbda4`.
- **E5e/E5f** `ti=12 i13`/`i15` : `FUN_142ed51d8`/`FUN_142ed4fe4` -> `FUN_1406d84b4` largeur 8 ; écrivains
  `142edb134`/`142edadf0` -> `FUN_142ed18e8`. Les paquets avancent (jusqu'à `i19`, fin de payload).

Composants d'arrêt restants (carte de la tête) : `ti=0 i18` (52), `ti=48 i0`/`i1` (23/16), `ti=0 i19` (19),
`ti=43 i31` (13, voulu), tacmap et forge (≤ 11), `ti=12 i19` (5). Hors composants : sortie par rejet 81 802,
liste non localisée 20 783, vue C (terminateur hors cadre 11 639).

## 2. Partie (2) : le message de dégâts (genre 0)

- **Le genre 0 est bien lu.** `chargeDegatsApres` suit champ par champ le lecteur `FUN_1407f15a4` et
  l'écrivain `FUN_142f19d34` (descripteur `143d0f978` ; domaines des références 1, 1, 7 en `14080a018`) ;
  `FUN_1407f2058` = R(1), R(5) si 0.
- **La fin de vue A qui le suit est juste.** 4 films de killsource : 635 trames à genre 0 non fermées (21 %,
  contre 10 % sans genre 0). 543 : l'ancienne localisation tombe sur la fin de vue A. 11 : un début plus loin
  ferme, mais la marche depuis la fin de vue A y lit d'abord 6 à 8 records cohérents sur 2 700 à 3 900 bits ;
  aucune chaîne de messages ne mène au début localisé (recherche exhaustive). Pièces :
  `tsv/p2_genre0_debuts.tsv`, `tsv/p2_dernier_record_avant_echec.tsv`, sondes `tsv/sonde_p2_*.txt`.
- **La cause est dans la vue B** : 437 fois sur 635, le dernier record lu est un `ti=38` (i0 + i2) de
  `fccc61cd` dont l'i0 a la porte posée ; l'exception datée world-object i0 (GA2-5) lisait les largeurs de
  la carte quelle que soit la porte. Chez le jeu (`FUN_14076e524`), porte à 1 -> ligne du niveau de la table
  DÉFAUT `DAT_1445cc9e0` ; l'écrivain (`FUN_140770640` -> `FUN_1407eb6a8`) prend la même ligne. Le niveau est
  l'immédiat 0x10 (`LEA R8D,[R9+0x10]` en `14076e2bc`), sans dépendance au registre ni au film.
- **Correctif hors portée** (`39510278d`, accord levelup-57 et utilisateur) : porte posée ->
  `profile.LargeursAxeParDefautDuBuild(niveauPosition)` ; precHigh, garde et portée inchangés. Carte v2 des
  20 films identique ; `fccc61cd` +10 949 paquets sains, 0 perdu ; 599 des 635 trames à genre 0 se ferment.
  Images-clés (lues hors portée) : ti=38 et ti=42 montent sur les sept bobines ; deux baisses ti=37
  (`111fa685` 1:5 slot 1714, `11de8353` 1:13 slot 2290), golden régénéré avec la justification accordée.
  Par record (ti 37/38/42/43), anciens builds : `11de8353` +515/−23, `111fa685` +420/−17, `e5adf7b2` +450/−33.
- Second motif (78919882 : échecs après des records `ti=1`/`ti=0` du moteur, slot 4) : non corrigé, non
  instruit plus avant (§5).

## 3. Partie (3) : le NEW de bipède de `bf15f7ab`

- **Hypothèse infirmée** : le NEW (`bf15f7ab` 14:1094, slot 553, `ti=35`, génération 1) est lu dans une trame
  FERMÉE. Il était refusé par `contreditUneEntiteVivante` : le slot restait lié en dur à `ti=20`
  (génération 0, image-clé 14:6) dont la suppression n'a pas été lue ; les deltas du joueur se lisaient
  ensuite sous `ti=20`. 829 refus sur les 20 films (`tsv/p3_refus_20films.txt`).
- **Règle** (`neufs_prouves.go`) : un NEW refusé dont l'en-tête est dans l'étendue que la fermeture de sa
  trame prouve (`preuveDeLaTrame`) est lié en fin de trame ; hors de cette étendue il reste refusé ; les
  marches d'essai du début de vue B ne lient rien. `rendParLAncrage` et les canaux sont inchangés.
- Effet : `bf15f7ab` 14:1094 à 14:1230, 69 trames fermées sur 69 (30 avant) ; carte v2 +1 046 sains, 0 perdu
  (`bf15f7ab` +621, `11de8353` +394). Gate de corpus de P3 seul (base `39510278d`) : `0797ce72` et
  `084a804d` « ok », rc 0.

## 4. Gates du lot (base `a4515e66c`)

| Gate | Sortie |
|---|---|
| Gate 2 (carte v2, 20 films) | +19 440 sains, 0 perdu, 0 film en baisse |
| Gate 3 (`killsource json`, 19 témoins + `1c4c63c2`) | 7 identiques, 6 diagnostic `calibration` seul, 7 avec des morts qui passent du balayage à la marche (81 morts : `1c4c63c2` 57, `60ae07c4` 20, quatre films une chacun), contenu identique ; `0797ce72` un candidat de santé de plus. `read_path` persisté : **`killsource-2026-10-08`** (`dbdfc3320`, chronique rotée en `rev_chronique_archive_2.go`) ; `tsv/gate3_killsource.tsv` |
| `TestGoldenFilms` | 4/4 ok ; `fccc61cd.golden` régénéré (score de l'oracle de `calibration` 1205 -> 1211) |
| Gate de corpus (copie du parc, 4 passes) | rc 1 ; banc **17/19 ok, 1 FAUX, 1 MANQUE** (§4.1) ; PERTE de filet instruites (§4.2) ; `tsv/gate_corpus_passe*.log.txt` |
| gofmt ; vet, vet `-tags=research`, vet `-tags=integration` (film) | vide ; rc 0 ×3 |
| archlint | ok (77 s) |
| golangci-lint (`--new-from-rev=a4515e66c`, film ; `--new-from-merge-base=origin/main`, module) | `0 issues.` |
| Mutations (`tsv/mutations.sh`) | **16/16 ROUGES** (`tsv/mutations_lot.txt`) |
| Baseline | `TestI59PerimetreDuPortParEtiquette` remplacé (absent de la baseline) ; 5 tests ajoutés |
| `make gate-push` (TMP `C:/t/vueb2`), par étapes | golangci 0 issue ; web typecheck vert ; web lint 0 erreur (26 avertissements préexistants) ; baseline en 9 tranches `-p 1` : 0 échec, `check_test_baseline.sh --from-jsonl` rc 0 (9 532 / 9 532) |
| Push, CI | message de clôture |

`objectives.Rev`, `profile.Rev`, `source.Rev`, `SchemaVersion` (89), `SchemaDesFaits` : constants.

### 4.1 FAUX et MANQUE

- **FAUX `084a804d`** : `repli_lien_prise_arme_abandonne` 0 -> 1 (bisection : E1). E1 lit quatre échanges d'arme
  de plus (slots 584, 570, 543 ×2) ; pour celui du slot 543 à t=8748, aucune position d'acteur à ±250 ms : le
  lien à l'arme au sol est abandonné comme le repli le prévoit. Fait neuf lu, sans publication fausse —
  **admis par le pilote** (2026-10-08).
- **MANQUE `0797ce72`** : P-2 « preuves contradictoires (image-clé) » 0 -> 1 (bisection : P2). Effet publié :
  4 lignes, des fins d'objets au sol (mur du slot 549 à t=1490, grappin et deux armes lâchés par le slot 541
  à t=1505) qui restent vus jusqu'à t=1561 au lieu de leur seule apparition ; pistes, vies, morts, équipes,
  ancres inchangées. Records d'image-clé lus hors portée. **Admission : décision de l'utilisateur, avant
  fusion** (soumise par le pilote).

### 4.2 PERTE de filet (aucune mesure du banc ne les couvre)

- Durées d'état (`stances/duree-totale`, 11 témoins) et compteurs d'états : même nature qu'aux lots
  précédents (fin d'un état observée plus tôt par des paquets désormais lus).
- `coverage.deathsPaths.directScan.*` (5 témoins) : les morts qui passent à la marche (gate 3).
- `11de8353` : le véhicule fantôme du slot 829 (châssis inconnu `c9467e39`, apparition hors carte, z = −191)
  disparaît ; ses 461 échantillons sont désormais ceux du véhicule 826 (`af31ab1a`, apparition au point des
  échantillons). Correction.
- `e5adf7b2` : la prise du slot 544 à t=418 disparaît ; la première vie du slot commence à t=719 (action
  hors vie de la base). Correction.
- Trous du tir continu, couverture des armes au sol et des poses (`placements.lives`), pistes de
  projectiles −1 à −7 : lectures déplacées par les paquets désormais lus.
- Aucune perte de kill, de mort, d'assistance, de score ni d'équipe (banc).

## 5. Découvertes (non traitées)

- **D-E5c** Des paquets qui s'arrêtaient sur `ti=10 i18`..`i21` ou `ti=12 i15` finissent maintenant en « fin de
  payload » (8 et 4) : cause en aval, à instruire.
- **D-P2-1** `78919882` : les trames à genre 0 qui ne se ferment pas échouent après un record `ti=1`/`ti=0` du
  moteur de partie (slot 4) ; non instruit plus avant.
- **D-P2-2** Les images-clés sont lues hors portée : l'i0 des objets du monde y diffère de la lecture du jeu
  (R(96) sous portée) — chantier LK de levelup-57.
- **D-P3-1** Les refus de NEW contre une entité vivante restent nombreux hors des trames prouvées (829 sur les
  20 films avant P3, dont 468 de génération différente) : un DEL manqué en amont est la cause probable.
- **D-1** Pendant le gate-push, `node_modules/.tmp` (jonction vers le checkout principal) a été purgé avant le
  typecheck : cache régénérable, aucun fichier versionné touché.

## 6. Ce qui reste

- Admission du MANQUE `0797ce72` : décision de l'utilisateur, avant fusion.
- Fusion dans `feat/v75` : geste de l'utilisateur ; à la fusion, renuméroter les rangs `grammar` et
  `killsource` pris entre-temps (levelup-57), refaire la carte v2 et le gate 2 sur la tête combinée ;
  `keyframe_closure.golden` : le premier fusionné garde le sien, l'autre le régénère.
- Recuisson du parc et backlog killsource : décisions de l'utilisateur.
- Composants restants (§1) et découvertes (§5).
