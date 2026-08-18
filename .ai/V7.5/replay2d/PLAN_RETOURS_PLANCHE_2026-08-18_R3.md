# Plan — Retours de la planche R3 (bilan utilisateur du 2026-08-18, apres-midi) -> lots

> Ecrit le 2026-08-18 par la session de pilotage. Bilan verbatim en annexe. Contrat `plan-execution`.
> Un worktree frere web `../LevelUp-wt-r3-visuels` (`wt/r3-visuels`), base `feat/v75` `f80b9dfd7`+.
> Valides sans action : A2, A5, A6, B1, B3, B4, B5, B7, D0-D4, E1, F1, W2, W3. Questions : frag 3,335 s
> GARDEE (D3 « Parfait »), mur = arc GARDE (W1) + couleur `warning` (R2-5 valide).

## Decisions tranchees (superviseur, a partir du bilan)

1. Tout en tokens (`color-tokens`) ; les items VALIDES ne bougent pas ; ce qui est « a proposer » va sur la
   planche (recette scratchpad `assemble.cjs`/`app.js`/`fxbundle`, items ajoutes seulement) ; les items
   valides de la planche sont REPLIES par defaut (verdict `[VALIDE]` -> section repliee, deplie au clic).
2. Production (verdicts rendus) : R2-1 joueur actif = token `success` en plus de la forme ; R2-4 Dynamo =
   variante 2 ; R2-5 mur = token `warning` ; A4 : la nappe Dynamo persiste >= 2,5 s au sol (mesurer la
   duree actuelle, la porter a 2,5 s, sans changer le son) ; A1 : croix de mort plus petites, plus epaisses,
   toujours rouges (token `danger`/`destructive` existant), persistance plus longue (proposer 2 durees sur la
   planche, livrer la plus courte par defaut) ; cercle d'altitude a la couleur du pion du joueur ; A3 : la
   bascule « effet de mort » est dans le tiroir de reglages du rejeu (verifier, sinon l'y mettre), OFF par
   defaut ; A8/R2-2 : heatmap = plus d'OPACITE (pas de quantile) et rampe bleu -> rouge -> violet aux
   extremes rares (tokens : construire la rampe a partir de tokens semantiques existants, jamais d'hex ; si
   aucun token violet n'existe, le dire et proposer) ; R2-3 : melee fatale = ETOILE (forme dessin anime), pas
   une croix ; C1 : REGRESSION — le fil des morts ne defile plus (tout compacte en fin de match, illisible)
   -> restaurer le defilement/la hauteur (V5 « une ligne » ne doit PAS supprimer le scroll vertical de la
   liste), retirer le glyphe « joueur actif » du fil, colorer en `success` le joueur actif et ses amis dans
   le fil ; P1/P2 : socles = UNE seule version : point « disponible » + icone de l'arme EN DESSOUS, blanche
   remplie avec contour noir (tokens : `foreground`/`background` du theme — pas d'hex ; contour = encre du
   fond), compteur de reapparition AU-DESSUS du point, meme style, SEULEMENT le compteur (aucune mediane
   ni ecart) ; B2/R2-7 : variante COMPACTE de la fiche = pas de perte d'info sauf : munitions de l'arme en
   main seulement, zone active du joueur retiree, armes + grenades + equipements sur UNE ligne — livree
   comme OPTION (bascule tiroir), la validee reste le defaut ; R2-6 ecran de dissimulation : proposition
   seule, semi-transparent ou pions au-dessus (famille absente : rien de branche).
3. Sons D5 : le traqueur de menaces prend le SON du capteur de menaces (`EQUIPMENT_PLACEMENT_SOUND_STEMS`) ;
   translocateur et champ de reparation : CHERCHER les fichiers ailleurs que la bibliotheque livree
   (archive Desktop `Halo Infinite - Sons armes/`, manifeste d'extraction, tags `eqip`/`snd!` du jeu si un
   index existe dans `.ai/V7.5/`) — trouve => encode et branche ; sinon negatif ecrit avec les chemins fouilles.

## Lot R3-V (un agent, worktree `wt/r3-visuels`)
- [ ] R3.1 A1 croix de mort + cercle d'altitude ; A3 bascule tiroir ; R2-1 success ; planche : 2 durees.
- [ ] R3.2 A4 Dynamo variante 2 en production + persistance >= 2,5 s (mesure avant/apres).
- [ ] R3.3 A8/R2-2 heatmap opacite + rampe bleu/rouge/violet en tokens.
- [ ] R3.4 R2-3 melee = etoile ; R2-5 mur = `warning`.
- [ ] R3.5 C1 fil des morts : defilement restaure (regression), glyphe actif retire, `success` actif+amis.
- [ ] R3.6 P1/P2 socles : version unique (point + icone dessous blanche/contour + compteur dessus).
- [ ] R3.7 B2/R2-7 fiche compacte en option (bascule) ; R2-6 proposition seule.
- [ ] R3.8 D5 sons : traqueur = capteur ; translocateur + champ : recherche, encodage si trouve, sinon negatif.
- [ ] R3.9 Planche : items valides replies, propositions ajoutees, fumee 0 erreur ; plan statue.
Gate : typecheck/lint/vitest match-replay verts ; aucune couleur en dur ; i18n FR+EN ; planche 0 erreur.

## Annexe — bilan utilisateur verbatim (2026-08-18, apres-midi)

(colle par le superviseur : voir message utilisateur ; points cles ci-dessus reprennent chaque verdict)

## Journal du plan
- 2026-08-18 — plan ecrit, agent lance (`wt/r3-visuels`).
