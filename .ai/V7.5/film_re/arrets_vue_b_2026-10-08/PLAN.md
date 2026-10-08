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
| E2 | `ti=10 i22 managed-object-interaction-filter-component` (299) | [x] | (ce commit) | `FUN_140dbdf5c` = `FUN_140dbe400(v = 1 < param_4)`, le bloc de filtres seul (comme `ti=12 i5`, `i6`) ; carte v2 contre E1 : +89 sains, +2 252 utiles, 0 perdu, aucun film en baisse ; arrêt suivant `ti=10 i23 flags` (201) ; `grammar-2026-10-08.3` |
| E3 | `ti=35 i59 biped-spartan-ability-non-predicted-state` (250) | [ ] | | fichiers de levelup-57 interdits |
| E4 | `ti=11 i4 managed-objective-interaction-filter-component` (165) | [ ] | | |
| E5 | suivants par fréquence si le temps le permet (`ti=10 i18`..`i21`, `ti=12 i17`, `ti=45 i1`, …) | [ ] | | |

## Gates du lot entier (base `a4515e66c`)

| # | Gate | Statut | Sortie |
|---|---|---|---|
| G1 | carte v2 base contre tête, gate 2 | [ ] | |
| G2 | gate 3 : `killsource json` (19 témoins + `1c4c63c2`) | [ ] | |
| G3 | `TestGoldenFilms` | [ ] | |
| G4 | gate de corpus sur une copie du parc, chaque FAUX / PERTE instruit, 0 MANQUE | [ ] | |
| G5 | gofmt, vet (normal, research, integration), archlint | [ ] | |
| G6 | golangci-lint 0 issue | [ ] | |
| G7 | mutations rouges | [ ] | |
| G8 | baseline des tests (aucun test retiré ni renommé) | [ ] | |
| G9 | `make gate-push` (TMP court dédié) | [ ] | |
| G10 | push + CI | [ ] | |

## Périmètre (2) et (3) : en attente du feu vert du pilote

| # | Item | Statut |
|---|---|---|
| P2 | message de dégâts (genre 0) mal lu, fin de vue A trop tôt (découverte 34 de levelup-57) | [ ] feu vert du pilote reçu (après (1)) |
| P3 | NEW de bipède lu dans une trame non fermée (`bf15f7ab`, slot 553, découverte 45) | [ ] feu vert du pilote reçu (après (2)) |

## Journal

- 2026-10-08 : plan écrit ; carte v2 de base (binaire de `a4515e66c`, 1 min 21 pour 20 films) :
  causes identiques à la tête d'intégration du lot précédent (`ti=12 i21` 679, `i22` 534, `i20` 435,
  `ti=10 i22` 292, `ti=35 i59` 250, `ti=11 i4` 165, `ti=12 i17` 92).

- E1 : lecteur `FUN_140dbe1bc` trouvé par la table de noms `143d07f00` (huit pointeurs) -> accesseur `14064c620` (index en `descripteur + 8`) -> descripteur `143d081c8 - 0x28` : écrivain `142edb178` (slot `+0x10`), lecteur `140dbe1bc` (après le thunk `14076ce9c`). R(1) présence (`FUN_1406cf008`) ; présent : R(32) puis `FUN_140dbe25c(v = 1)` = bloc de filtres `FUN_140dbe400` (déjà porté, [consumeFilterSet]), R(32), un R(32) par filtre présent, K entrées d'ordre de 3 bits. Écrivain `FUN_142c94dd4` dans le même ordre. Statut `partiel` (tag 15 arrête, comme `i2`..`i6`). Carte v2 contre la base : +6 617 sains, 0 perdu ; des 1 675 paquets arrêtés sur `i20`..`i25`, 1 167 deviennent sains, 216 s'arrêtent sur `ti=11 i4`, 210 sortent par rejet. Rotation R0 faite au même rang.
- 2026-10-08 : feu vert du pilote pour (2) puis (3), à traiter après (1) ; liste complète des fichiers interdits reçue (lot LK de levelup-57) ; annonces à levelup-57 seulement pour les passes de plus de 15 min.

- E2 : nom `143c94b40` -> accesseur `141177ff0` -> descripteur (accesseur en `143d09308`) : niveau `141179610` (`MOV EAX, 2`), écrivain `142edb250` (`etat + 0x68`, saut vers `FUN_142c7023c`), lecteur `140dbdf5c` : `FUN_140dbe400(etat + 0x68, flux, v = 1 < param_4)`, la même forme que `ti=12 i5`/`i6` et que `ti=11 i4` (`140dbe170`, `etat + 0x48`, écrivain `142edb5cc`). Le commentaire du dépôt qui disait la queue de `FUN_142c7023c` « de largeur inconnue » (appel virtuel par tag) parlait de l écrivain du bloc de filtres, porté depuis le lot 5.1.1 (`consumeFilterSet`). Carte v2 contre E1 : +89 sains, 0 perdu.

## Découvertes

Voir RAPPORT.md.
