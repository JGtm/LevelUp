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
| C2 | `ti=12 i16 managed-navpoint-override-flags` | [x] | (ce commit) | `FUN_140ebf834` = `R(5)` ; carte v2 contre C1 : +110 sains, 0 perdu, aucun film en baisse ; `grammar-2026-10-07.2` |
| C3 | `ti=45 i0 matchflow-sequence-data-component` | [ ] | | |
| C4 | `ti=10 i2 managed-object-navpoint-component` | [ ] | | |
| C5 | `ti=12 i18 managed-navpoint-position-offset` | [ ] | | |
| D  | point (d) du §2.2 (instruit si la lecture l'éclaire, pas un lot) | [ ] | | |

## Gates du lot entier (§5)

| # | Gate | Statut | Sortie |
|---|---|---|---|
| G1 | carte v2 base `879f31bbf` contre tête du lot, gate 2 | [ ] | |
| G2 | gate 3 : `killsource json` sur les 19 témoins, base contre tête | [ ] | |
| G3 | `TestGoldenFilms` | [ ] | |
| G4 | gate de corpus (`replay-corpus-gate --reference=base`), chaque FAUX/PERTE instruit, 0 MANQUE | [ ] | |
| G5 | gofmt, vet (+ `-tags=research`), archlint | [ ] | |
| G6 | golangci-lint 0 issue (`--new-from-rev=879f31bbf`) | [ ] | |
| G7 | mutations rouges sur chaque nouveau lecteur | [ ] | |
| G8 | baseline des tests renommés/supprimés | [ ] | |
| G9 | `make gate-push` (TMP court dédié) | [ ] | |
| G10 | push + CI | [ ] | |

## Journal

- 2026-10-07 : plan écrit ; carte v2 de base (binaire de `879f31bbf`, ~2 min pour 20 films).
- C1 : port de L2 repris sur la tête (le maillon s insère entre M4b et `ti=40` ; `i37` passe par le lecteur de minuteur unique `lireMinuteur142ba78dc`, venu depuis L2 ; la règle du masque de `pasDEssai` était déjà en tête, lot LT). Ghidra relu : `i19` `FUN_1410156e4`, `i21` `FUN_1407f0678` / `FUN_1407f08bc` / `FUN_1407f08f8`. Carte v2 : 0 perte, la perte de `1c4c63c2` de L2 (second rang) ne se reproduit pas (+13). Retenu.
- Mutations C1 et C2 jouées (`tsv/mutations.sh`, overlay) : 20 / 20 ROUGES.
- C2 : lecteur `FUN_140ebf834` -> `FUN_140ebf854` (`R(5)` vers `etat+0x70c`), écrivain `142ed0e2c` (cinq bits du même mot), trouvés par nom -> `getName` -> descripteur (slot après le thunk `FUN_14076ce9c`). Carte v2 contre C1 : +110 sains (64 arrêtés sur `i16` en C1), 0 perdu ; les autres records `i16` avancent et s arrêtent sur `i18` (63 -> 318) et `i17` (92). Retenu.

## Découvertes

(aucune pour l'instant)
