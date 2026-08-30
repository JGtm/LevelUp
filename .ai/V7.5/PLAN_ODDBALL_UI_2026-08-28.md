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
- [!] 2.1 Schéma `ObjectiveObjectLife` = {en, family, fr, pts[], t0, t1} ; coverage =
      {declared, lives, motionless, outOfAxis, points, scanned}. AUCUN délai de respawn / borne
      de ré-apparition publié (vérifié `generated.ts` + `ObjectiveObjectLife` schema).
- [!] 2.2 Donnée MANQUANTE → aucun compteur (ne pas fabriquer : le geste demandait « s'il y a »).
      Le gap entre deux vies libres pourrait être un portage OU un respawn — indistinguables sans
      donnée fiable (oracle porteur réfuté phase D4). Statué au CR ; branchement = lot Go si le
      film publiait un jour la borne de ré-apparition.

## Étape 3 — Le crâne LIBRE ne s'affiche pas (investiguer + corriger, périmètre rendu)
Retour : « investiguer pourquoi le crâne libre ne s'affiche pas alors que la donnée existe ».
- [x] 3.1 Écarté SUR PIÈCES : (a) pas OFF par défaut — `objectiveObjects.paint` est appelé
      INCONDITIONNELLEMENT dans `draw()` ; (b) pas de toggle manquant — le calque est ungated
      (intentionnel) ; (c) pas de filtre de mode — `normalize` mappe `raw.objectiveObjects`
      direct, `useMatchReplay` le sert tel quel ; (d) pas de garde de schéma ; (e) CAUSE = le
      glyphe : disque r=5 NEUTRE + cerne muted-fg mince, il se dissolvait sur un fond de carte
      photographique (exactement le problème du drapeau AVANT son liseré).
- [~] 3.2 Corrigé par l'ÉTAPE 1 (périmètre rendu) : liseré à l'encre du fond + disque agrandi +
      orbites → le crâne libre se détache de la carte comme le drapeau. Résidu hors code : si
      l'artefact servi n'a PAS été re-cuit (pas d'`objectiveObjects`), rien ne s'affiche — data/
      déploiement, à vérifier au gate visuel.

## Étape 4 — Message inter-manche = style victoire/défaite, SANS l'accent gauche
Retour : « le message fin de manche doit avoir le même style que le texte de défaite ou victoire
(SANS l'accent sur le côté gauche, faut le virer de ce style) ».
- [x] 4.1 `borderLeft: 6px solid tint.accent` retiré du `TeamPanel` de `ReplayVictoryOverlay`.
- [x] 4.2 `ReplayRoundBreakOverlay` = panneau neutre + titre (`OVERLAY_NEUTRAL_PANEL`+`OVERLAY_TITLE`).
- [x] 4.3 `replayOverlayStyles.ts` centralise (corps/panneau neutre/titre) + garde-rail
      `replayOverlayStyles.guard.test.ts` (le littéral de titre ne vit que dans le module).
- [x] 4.4 Tests overlays verts (33) ; les tests victoire n'assoient pas sur l'accent (vérifié).
Gate : typecheck OK ; vitest 3 fichiers (33) OK ; eslint fichiers touchés 0 erreur/0 warning.

## Étape 5 — Son « manche terminée » FR/EN câblé
Retour utilisateur + stub déjà écrit dans `replaySound.ts`.
- [x] 5.1 Copiés : `round_over_fr.wav` (1,749 s) / `round_over_en.wav` (1,707 s) — 48 kHz PCM
      16-bit stéréo, format canonique du dépôt (aucune ré-encodage nécessaire).
- [x] 5.2 `roundOverSound.ts` : `roundOverSoundEvents(doc, locale)` daté sur `roundTransitions`
      (même mesure que l'overlay), stem par locale ; branché dans `buildSoundTimeline`
      (catégorie objective) `if (locale)`, stub remplacé.
- [x] 5.3 `locale` threadé : `useReplaySound` (7e param) → `buildSoundTimeline` ; `ReplayCanvas`
      passe `locale`.
- [x] 5.4 Garde-rail : `ROUND_OVER_SOUND_STEMS` re-exporté + ajouté à `referenced` ET `longs`
      (durée ≤6 s, non retronqué).
- [x] 5.5 `roundOverSound.test.ts` (bascule, locale FR/EN, manche unique, garde d'horloge).
Gate : typecheck OK ; vitest 4 fichiers son (101) OK ; eslint fichiers touchés 0 erreur
(seul avert. `objectiveObjects` pré-existant).
Réserve : les 2 wav ne sont PAS renormalisés à -16 LUFS (recette maison) — copie brute demandée ;
niveau à re-mesurer au gate d'écoute si nécessaire (CR).

## Découvertes (à ne pas traiter — noter seulement)
- `ReplayCanvas.draw()` n'a pas `objectiveObjects` dans son tableau de dépendances (avert.
  exhaustive-deps PRÉ-EXISTANT, dette gelée) : latent, non traité (hors périmètre). En pratique
  inoffensif — `draw` se recrée sur `doc`, dont dépend `objectiveObjects`.

## Journal
- 2026-08-28 : plan écrit et commité avant toute modification (contrat plan-execution).
- 2026-08-28 : étape 1 close — glyphe crâne centralisé + habillé comme le drapeau.
- 2026-08-28 : étapes 2 (respawn = data manquante) et 3 (crâne libre = salience, corrigé par
  étape 1) statuées sur pièces, aucun code neuf (pas de fabrication).
- 2026-08-28 : étape 4 close — accent gauche retiré, message inter-manche aligné, style centralisé.
- 2026-08-28 : étape 5 close — son « manche terminée » FR/EN câblé (piste, locale-aware), assets
  copiés + garde-rail. Réserve loudness (copie brute, non renormalisée).
