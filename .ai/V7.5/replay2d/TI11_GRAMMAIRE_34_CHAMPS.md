# ti=11 « managed-objective » — la grammaire des 34 champs (asset de recherche conservé)

> Conservé le 2026-09-13 sur décision de l'utilisateur, après suppression de la branche de
> recherche `wt/ti11-cadre` (27 août 2026). Le code complet, les bancs et les logs de mesure sont
> figés sous le tag `archive/ti11-cadre` (commit `c8aaf4afb`, poussé sur origin).

## Ce que c'est

L'entité ti=11 du film Theater est le DESCRIPTEUR d'objectif du HUD (34 composants), pas l'objet
physique. Ses feuilles ont été résolues au Ghidra adresse par adresse, et le décodeur reproduit
le cadre d'image-clé bit à bit à 90,32 % (100 % sur Oddball et KOTH). Mais l'état vivant
(porteur, progression) n'est ni dans l'image-clé (état par défaut) ni dans le delta (chaînage
3,8 %, bruit) : c'est un descripteur recalculé côté client. Conclusion du 2026-08-27 : ne pas
rejouer cette piste ; le propriétaire de zone vient de ti=13, crâne et drapeau de ti=42.

## La grammaire (feuilles résolues, largeurs immédiates, aucune porte, aucune quantification)

| index | composant | déserialiseur du jeu | forme |
|---|---|---|---|
| i0 | managed-objective-timers-component | FUN_142ed5a6c -> FUN_1410d9088 | 2 x R(7) |
| i1 | managed-objective-color-component | FUN_142ed544c | 4 x R(8) RGBA (déquant [0,1]) |
| i3 | managed-objective-object-reference-component | FUN_142ed5550 | R(32) GlobalID |
| i5 | managed-objective-type-component | FUN_1410fc4a4 | R(32) |
| i12 | managed-objective-progress-component | FUN_142ed575c | R(32) |
| i13 | managed-objective-required-progress-component | FUN_142ed5844 | R(32) |
| i14 | managed-objective-state-component | FUN_142ed5948 | R(3) |
| i15 | managed-objective-parent-objective-component | FUN_142ed5674 | R(32) GlobalID |
| i16..i31 | managed-objective-sub-objective-entities-component | FUN_142ed5974 | R(32) GlobalID, 16 slots partageant le nom |

La structure est un arbre parent/enfant (i15 parent, i16-31 enfants) borné à 16 slots (offset
d'i32 prouve la borne). Les 16 slots sub-objective partagent leur nom : l'occurrence se
reconstruit depuis l'ordre du masque, comme les `rtpc` de ti=10 et les `masked-property` de ti=13.

## Où lire le détail

- `PISTE_A_ti11_color.md`, `PISTE_B_ti11_sous_entites.md`, `TI11_SPEC_10_FEUILLES.md` (ce dossier,
  repris de la branche) : conception et preuves Ghidra.
- `TI11_components_managed_objective.go.txt` (ce dossier) : le décodeur tel qu'écrit, hors
  compilation (extension .txt volontaire : rien ne le consomme).
- Tag `archive/ti11-cadre` : bancs (`keyframe_objective_fullstate_test.go`, `ti11_delta_test.go`),
  protocoles et logs (`registre_film/TI11_*`), plan `PLAN_TI11_TEST_CADRE_2026-08-27.md`.

## Si on rouvre un jour

Le chemin serait de RECALCULER l'état d'objectif côté LevelUp comme le HUD le fait (à partir
des faits publiés : zones ti=13, portages ti=42, score), pas de le lire dans ti=11. Le décodeur
ci-dessus serait alors à porter sur le profil de déchiffrage (lot I), pas à fusionner tel quel.
