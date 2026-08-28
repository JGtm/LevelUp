# PLAN — Corrections rejeu 2D Oddball (retours visuels utilisateur, 2026-08-28)

Branche `wt/oddball-ui` (base 53db169ce). Exécuté sous contrat `plan-execution`.
Périmètre : front-end `apps/web/src/features/match-replay/` + assets `static/sounds/`.
INTERDITS : dépôt principal / autres worktrees intouchés (sauf copie des 2 wav) ; pas de Go ;
thought_log/REGISTRE non touchés (textes au CR) ; tokens couleur only ; strings FR+EN.

Gates par étape : `npm run typecheck`, `npm run lint`, `npm run test` (vitest, garde-rail son inclus).

## Étape 1 — Le crâne (porteur + libre) rendu comme le drapeau : taille + contour/liseré
Retour : « prendre son icône et le mettre comme on a mis le drapeau (taille et contour) ».
- [x] 1.1 `skullGlyph.ts` centralise le glyphe (les deux calques l'appellent).
- [x] 1.2 Habillage drapeau : liseré encre FOND (cote 1,6 = `FLAG_OUTLINE_PAD`), disque r=5→7,
      deux orbites. Forme BOULE, pas d'icône du jeu.
- [x] 1.3 `skullCarrierLayer` + `objectiveObjectsLayer` consomment le glyphe ; `edge` → `outline`.
- [x] 1.4 `ReplayCanvas` : `outline: markInk.outline` aux deux calques.
- [x] 1.5 Tests : `skullGlyph.test.ts` neuf + MAJ des deux tests de calque (arc/fill, `outline`).
- [x] 1.6 CR : icône `contour-25.png` (tag weap `0017592c`) existe mais URL non cuite → lot Go.
Gate : typecheck OK ; vitest 3 fichiers (17) OK ; eslint fichiers touchés 0 erreur.

## Étape 2 — Petit compteur de respawn du crâne libre (si la donnée existe)
Retour : « quand il respawn un petit compteur s'il y a ».
- [ ] 2.1 Vérifier sur pièces le schéma `ObjectiveObjectLife` (délai de respawn / borne de
      ré-apparition ?).
- [ ] 2.2 Si présent → afficher le compteur ; SINON → data manquante, ne rien fabriquer, statuer
      au CR.

## Étape 3 — Le crâne LIBRE ne s'affiche pas (investiguer + corriger, périmètre rendu)
Retour : « investiguer pourquoi le crâne libre ne s'affiche pas alors que la donnée existe ».
- [ ] 3.1 Écarter sur pièces : OFF par défaut ? toggle manquant ? filtre de mode ? garde de
      schéma ? glyphe invisible (encre/token) ?
- [ ] 3.2 Corriger la cause dans le périmètre du rendu du crâne.

## Étape 4 — Message inter-manche = style victoire/défaite, SANS l'accent gauche
Retour : « le message fin de manche doit avoir le même style que le texte de défaite ou victoire
(SANS l'accent sur le côté gauche, faut le virer de ce style) ».
- [ ] 4.1 Retirer l'accent latéral gauche (`borderLeft`) de `ReplayVictoryOverlay` (TeamPanel).
- [ ] 4.2 `ReplayRoundBreakOverlay` adopte le style du panneau (carte neutre + titre), sans accent.
- [ ] 4.3 Centraliser les classes de panneau/titre partagées (≤2 copies) + garde-rail.
- [ ] 4.4 Tests overlays verts (aucun n'assied sur l'accent gauche — vérifié).

## Étape 5 — Son « manche terminée » FR/EN câblé
Retour utilisateur + stub déjà écrit dans `replaySound.ts`.
- [ ] 5.1 Copier les 2 wav de E:/ → `static/sounds/halo_infinite/round_over_fr.wav` / `_en.wav`.
- [ ] 5.2 `roundOverSound.ts` : `roundOverSoundEvents(doc, locale)` daté sur `roundTransitions`,
      stem par locale ; branché dans `buildSoundTimeline` (catégorie objective) via `locale`.
- [ ] 5.3 Threader `locale` : `useReplaySound` → `buildSoundTimeline` ; `ReplayCanvas` passe la locale.
- [ ] 5.4 Garde-rail `replaySoundAssets.guard.test.ts` : stems enregistrés (référencés + durée long).
- [ ] 5.5 Tests logique son (`roundOverSound` + timeline).

## Découvertes (à ne pas traiter — noter seulement)
- `ReplayCanvas.draw()` n'a pas `objectiveObjects` dans son tableau de dépendances (avert.
  exhaustive-deps PRÉ-EXISTANT, dette gelée) : latent, non traité (hors périmètre). En pratique
  inoffensif — `draw` se recrée sur `doc`, dont dépend `objectiveObjects`.

## Journal
- 2026-08-28 : plan écrit et commité avant toute modification (contrat plan-execution).
- 2026-08-28 : étape 1 close — glyphe crâne centralisé + habillé comme le drapeau.
