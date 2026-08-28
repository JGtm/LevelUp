# PLAN — Écran de fin de match : réapparition + habillage (retours utilisateur, 2026-08-28, lot 2)

Branche `feat/v75`. Contrat `plan-execution`. Périmètre : `apps/web/src/features/match-replay/`.
INTERDITS : pas de Go, pas de nouvelle string sans FR+EN, tokens couleur uniquement.

Retours à traiter :
1. « je n'ai plus le message qui indique la défaite ou victoire » (régression).
2. « le message, la musique et la voix se jouent dès que le score atteint le montant requis
   ou que le chrono arrive au bout » — distinct de la fin du film (queue d'outro).
3. Logo de l'équipe du joueur : conservé, mais « il doit respecter les choix de couleur
   d'équipe choisie par le user ».
4. Seul le STATUT (victoire / défaite / égalité) dans un bloc de couleur, sans accent gauche ;
   nom d'équipe et score = texte libre.
5. Les textes « manche terminée » prennent le même affichage que le statut.

## Étape 1 — Cause de la disparition du message (sur pièces, avant tout code)
- [x] 1.1 Le rendu est dérivé de `frame >= playWindow.endFrame` ; `frame` est publié par
      `useReplayClock.tick`, BRIDÉ à 150 ms. La boucle (`useReplayPlayback`) peint la dernière
      image puis s'arrête : si ce dernier `tick` tombe dans la fenêtre de bridage (cas le plus
      fréquent — le pas de fin arrive en moyenne 8 ms après le précédent à 60 fps), l'image de
      fin n'est JAMAIS publiée et l'écran ne se rend pas. Le son, lui, part par `onEnded`
      (chemin distinct) : d'où « le son oui, le message non ».
- [x] 1.2 Correction : la dernière image de la fenêtre passe le bridage (publication forcée).

## Étape 2 — Instant de déclenchement (retour 2) — vérification, pas de code neuf
- [x] 2.1 `playWindow.endFrame` = `t0_ms + playable_duration_seconds` (fin DÉCLARÉE du jeu),
      bornée par la fin du film ; la queue de 5-6 s du film est HORS fenêtre (`replayWindow.ts`).
      C'est déjà « score atteint / chrono au bout », pas la fin du film. Rien à changer :
      message (étape 1) et son (`onEnded`) partagent cette borne.

## Étape 3 — Couleur : les choix de l'utilisateur, plus l'identité officielle
- [x] 3.1 L'écran de fin portait la couleur d'IDENTITÉ (Eagle bleu / Cobra rouge — exception
      assumée à la décision D1). Retour 3 : c'est le réglage utilisateur qui fait foi →
      `team-ally` (token surchargeable), comme tout le reste de la page rejeu.
- [x] 3.2 Le LOGO est une silhouette monochrome : teinté au même token (masque CSS), avec
      sonde de chargement (un masque en 404 ne masque plus rien → aucun aplat parasite).

## Étape 4 — Le statut seul dans le bloc, nom et score en texte libre
- [x] 4.1 `replayOverlayStyles.ts` : le bloc porte désormais le statut ET SA TYPOGRAPHIE.
- [x] 4.2 `ReplayVictoryOverlay` : bloc = statut ; nom d'équipe et score sortis du bloc.
- [x] 4.3 Pas d'accent latéral gauche (déjà retiré au lot 1, invariant conservé).

## Étape 5 — « Manche N terminée » = même affichage que le statut
- [x] 5.1 `ReplayRoundBreakOverlay` consomme le bloc neutre de l'étape 4.

## Gates
- [x] `npx tsc --noEmit` (web) ; `npx vitest run` sur les fichiers touchés ; `npx eslint` idem.

## Journal
- 2026-08-28 : plan écrit avant modification. Cause de la régression établie sur pièces
  (bridage 150 ms de la publication d'image), pas une régression de style du lot 1.
- 2026-08-28 : lot clos. Gates : typecheck 0, eslint 0 (7 fichiers), vitest match-replay
  1639/1639. Gate VISUEL non passe par l'agent (l'app locale exige une authentification Xbox) :
  a l'oeil de l'utilisateur.
