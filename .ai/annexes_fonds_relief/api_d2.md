# Bibliothèque web du relief (D.2) : ce que D.3, D.4 et D.6 branchent

Compte rendu de l'item D.2 du plan `PLAN_FONDS_RELIEF_2026-11-18.md`, relu par le pilote le 2026-11-18
(`23ec703e5e`, branche `feat/fonds-relief-d2`). Point d'entrée : `apps/web/src/lib/replay/relief/index.ts`.

## API

- Lecture : `parseReliefData(raw)` (erreurs `ReliefError` `data-version` / `data-invalid`) ;
  `readReliefImage(blob, calibration, heightMeta, loaders?)` ; `decodeReliefPixels(image, calibration, meta)`
  (contrôle de la conception §6.1) ; `hasNoMesh(data)`.
- Traits et repères : `traceLineSet(path, lines, toScreen, closed?)` ; `pixelToWorld`, `worldToPixel`.
- Calcul pur (aussi dans le worker) : `shadeRelief`, `lightForTurns(turns)`, `reliefRamp(base)`,
  `buildTintTable`, `tintRelief`, `rgbToCss`, `buildFloorStack`, `computeSlice(input, cut, {plan?})`,
  `cutWeight`, `filterLinesByKeep`.
- Étages, suivi, infobulle : `floorRanks(levels)` (rang 0 = « Bas », libellés dans les dictionnaires),
  `nearestRank`, `rankHeight`, `snapCut`, `quantizeCut`, `bearingFloor(mesh, x, y, z)`, `FOLLOW_INITIAL`,
  `stepFollow(state, sample, {mesh, levels})`, `meshShareOf(data, image?)` (D-16, `null` sans maillage).
- Moteur : `createReliefComputer({workerFactory?, forceInline?, startTimeoutMs?})` →
  `.mode()`, `.load({image, data})`, `.render({turns, cut, base, lines, masks})` → `ReliefFrame | null`,
  `.dispose()`. `ReliefFrame = { width, height, cut, image (RGBA), alpha (masque du fond Rendu), planAlpha
  (masque du fond Plan), lines: { steps, contours, walls } }`.

## Branchements

- Fichiers : image par `api.getBlob(<route ?variante=relief>)`, données par `api.get(<route
  background.relief.json>)` (lot C).
- Worker : `createReliefComputer()` le crée seul et bascule sur le calcul synchrone s'il est indisponible.
- `load` CÈDE `image.heights` et `image.flags` au worker : ne plus les relire ensuite.
- La dernière demande de rendu gagne ; une demande remplacée rend `null`.
- `base` : la teinte du jeton `map-relief-base` (D.5) résolue en hex ; `base: null` = masques et traits
  seulement, pour les fonds Plan et Rendu.
- Dessin : image et masques au calage du fond, donc par `drawMapBackground` et `rotateCanvasAbout` ; traits
  par `traceLineSet` avec la projection de la vue, liseré avec `closed = true`.
- `turns` (quarts de tour) ne change que la direction de la lumière, jamais la géométrie.
- Toute `ReliefError` (`isReliefError`) → repli sur le Plan, avec un journal.
- `stepFollow` attend le temps du REJEU en millisecondes ; un recul ou un saut de plus de 1 s est traité comme
  un saut de lecture. D.6 décide si le délai de 300 ms suit la vitesse de lecture.
- Coupe arrondie au quart d'unité ; défauts `floors.merge` 0,4 et `floors.tolerance` 0,3 si le JSON ne les
  porte pas.
