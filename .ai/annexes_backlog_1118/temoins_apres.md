# Témoins APRÈS de la branche `chore/830-backlog-1118` (2026-11-23)

Item I.2 du plan `.ai/PLAN_BACKLOG_2026-11-18.md`. Mêmes commandes que dans `temoins_avant.md`, rejouées sur la tête de la branche de version
à jour (une suite à la fois, `--maxWorkers=2`). Aucune correction de code.

## Table des témoins

| Témoin | Avant | Après | Verdict |
|---|---|---|---|
| 1a. vitest `src/features/match-replay` (MR) : fichiers | 390 (387 réussis, 3 sautés) | 441 (438 réussis, 3 sautés) | changé comme le plan le prévoit (pistes A à D, H, M, Q : tests neufs) |
| 1a. MR : tests | 5 521 (5 517 réussis, 4 sautés, 0 échoué) | 6 271 (6 267 réussis, 4 sautés, 0 échoué) | changé comme le plan le prévoit (hausse) |
| 1a. MR : durée | 443,7 s | 328,7 s | non comparable (charge de la machine) |
| 1b. vitest `src/components` : fichiers / tests | 141 / 1 208 réussis, 0 échoué | 150 / 1 246 réussis, 0 échoué | changé comme le plan le prévoit (hausse ; piste H.2 / H.3 : tests de fermeture des couches, M, Q) |
| 1c. vitest `src/features/squad` : fichiers | 111 | 109 | changé, explication hors plan du backlog (voir ci-dessous) |
| 1c. squad : tests | 999 réussis, 0 échoué | 1 019 réussis, 0 échoué | changé (hausse nette) |
| 2a. Go Escouade par minute : paquet | `ok` | `ok` | identique |
| 2b. 12 tests nommés du tableau avant | 12 PASS | 12 PASS | identique |
| 2c. tests du motif, au total | 12 | 14 (2 tests neufs, PASS) | changé comme le plan le prévoit (G.2) |
| 2d. synergy.937 | 79.01814300960513 | 79.01814300960513 | identique |
| 2e. synergy.59 | 78.30508474576271 | 78.30508474576271 | identique |
| 2f. synergy.zero / synergy.negatif | 0 / 0 | 0 / 0 | identique |
| 2g. valeurs de la caractérisation (principal 0,66 / 0,31 / 0,12 / 300 / 136,63 ; ami 0,23 / 0,35 / 0,35 / 48,8 / 19,21) | posées par assertions, test PASS | test PASS, assertions inchangées | identique (G.2 : « aucun chiffre des tests de caractérisation n'a changé ») |
| 3. garde-rail `dismissableLayers.guard.test.ts` | 4 tests réussis | 4 tests réussis | identique en réussite |
| 3b. comptes du balayage (591 `.tsx` balayés, 105 dans le rejeu, 5 exemptions) | 591 / 105 / 5 | non rejouable tel quel (la sonde jetable copiait l'ancien détecteur) ; le garde-rail a changé de périmètre : `components/`, `lib/` et `.ts` de `features/` en plus, 11 exemptions datées au lieu de 5 | changé comme le plan le prévoit (H.1) |
| 4a. match 38900a5e : fil 4:30 à 5:30, variantes A et B | 19 lignes chacune | NON REJOUABLE ICI | non rejouable ici |
| 4b. fiche du joueur A aux images 9700, 9786, 9900, 9940 | mort / mort / mort / mort | NON REJOUABLE ICI | non rejouable ici |

## Détail des verdicts

### 1c. Escouade : 111 fichiers avant, 109 après

La baisse de deux fichiers vient du commit `b7f5fd00f2` (2026-11-22, « impact-sessions, W ») : il supprime avec leur code
`SquadImpactScoreboard.test.tsx`, `impact/SquadImpactHistoryCard.test.tsx`, `impact/impactHistory.logic.test.ts` et
`impact/impactHistoryChart.test.ts` (quatre fichiers de test retirés avec le code qu'ils testaient). Les deux fichiers de test neufs de l'Escouade
compensent en partie (111 - 4 + 2 = 109). Le nombre de tests monte quand même de 999 à 1 019. Ce retrait n'est rattaché à aucun item du plan
du backlog : il appartient à un autre chantier (impact des sessions), daté de la veille, donc légitime comme retrait de tests avec son code, mais à ne pas
compter comme un effet du backlog.

### 2c. Deux tests Go neufs

`TestStatsParMinuteEscouade_AucunMatchRetenuEstInconnuPasZero` et `TestStatsParMinuteEscouade_AucuneCadenceConnueRendNil` sont ajoutés par le
commit `94d2f807b5` (G.2 : les taux par minute deviennent facultatifs, « inconnu » quand aucun match n'est retenu). Aucune valeur des 12 tests d'avant ne change.

### 3b. Garde-rail des couches

Le plan H.1 élargit le périmètre et ajoute des exemptions datées (`GamertagCombobox.tsx`, `useAudioUnlock.ts`, `titleTooltip.controller.ts`,
`tooltipPosition.ts`, `useDismissable.ts`, `useReplayFullscreen.ts`, plus les 5 d'avant). Le test passe : aucune exemption sans objet, aucun motif non exempté.
Le chiffre « 105 fichiers du rejeu à plafond 0 » n'est plus la mesure du garde-rail : deux fichiers du rejeu sont désormais exemptés (raison datée).
Pour rejouer les comptes à l'identique, il faudrait réécrire la sonde jetable sur le nouveau détecteur ; ce n'est pas fait ici.

### 4. Match 38900a5e : non rejouable ici

Le témoin vient d'un test jetable (`sondes/temoin_38900a5e.test.tsx.txt`) qui lit le rejeu cuit du match sur le disque (`data/cache/replays/halo_infinite/`),
absent de ce dossier de travail (il n'y a que `rasters/`), et qui exige de créer un fichier de test dans le dépôt. Pour le rejouer : le fichier de rejeu
du match (copié ou pointé par la sonde), la sonde recopiée en `.test.tsx` sous `apps/web/src/` avec ses imports mis à jour si des fonctions ont été renommées, puis `npx vitest run` dessus.
Ce que Gate A du plan attend de ce témoin : plus de ligne « mort » pour le joueur A à l'image 9785, et la fiche « a quitté la partie » aux images 9786 et 9900.

## Écarts non expliqués

Aucun écart non expliqué sur les témoins rejoués. Réserve : la baisse de fichiers de test de l'Escouade s'explique par un commit hors plan du backlog (voir 1c).
