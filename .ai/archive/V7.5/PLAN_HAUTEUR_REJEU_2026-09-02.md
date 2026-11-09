# PLAN — Le rejeu tient dans l'écran (compaction de la tête + hauteur élastique)

> Branche : `wt/hauteur-rejeu` (worktree dédié `LevelUp-wt-hauteur-rejeu`), depuis `feat/v75`.
> Demande utilisateur du 2026-09-02 : « le canevas doit toujours être visible dans son
> entièreté niveau hauteur — ça englobe la map et le rejeu mais aussi les contrôles de
> lecture ». Plus : « la partie sous la L1 pourrait être compactée un peu », et le bouton
> « Retour au match » fait parfois doublon avec le fil d'Ariane, et son libellé est faux
> quand on n'est jamais passé par la fiche.

## Le constat mesuré (2026-09-02)

Budget vertical pour voir bandeau + terrain + frise + contrôles :

| Zone | px |
|---|---|
| NavL1 + gap du shell | 64 |
| Fil d'Ariane (`pt-4 pb-2`) | 44 |
| `py-3` + titre « Rejeu 2D » + ligne de rappel (`min-h-6`) | 68 |
| `space-y-4` | 16 |
| ReplayScoreBanner | ~60 |
| Carte : bord + `p-3` + canvas 480 + transport ~207 + `p-3` | ~713 |
| **Total** | **~965 px de viewport** |

Confrontation (Chrome Windows, ~110-120 px de chrome navigateur) :

- 1080p @ 100 % → ~960 px : **juste en dessous** ;
- 1080p @ 125 % (défaut de la plupart des portables) → ~768 px : **il manque 200 px** ;
- 1080p @ 150 % → ~640 px ; 1366x768 → ~660 px.

Ça ne tient donc pas sur la configuration la plus répandue.

## Décisions prises avant d'écrire une ligne

- **D1 — La compaction et l'élasticité s'additionnent, elles ne s'opposent pas.** Sur les
  ~965 px, le canvas est le SEUL élément élastique : le bloc transport (~207 px) est
  incompressible, ce sont des commandes. Chaque pixel rendu par la tête devient un pixel
  de terrain.
- **D2 — RÉVISÉE le 2026-09-02 (contestation utilisateur, acceptée). Le terrain GRANDIT.**

  *Décision initiale, abandonnée* : plafonner à 480, la valeur d'alors, pour mettre l'export
  hors du chantier. *Contestation* : « pourquoi ça grandirait jamais ? on a des grandes images
  de map, c'est dommage de pas pouvoir l'agrandir ». *Vérification* : l'argument export ne
  tenait qu'à moitié. Les dimensions de sortie dépendent DÉJÀ de `devicePixelRatio`
  (`useReplayExport.ts` : `dpr = devicePixelRatio × exportRenderScale`) — un écran 2x exporte
  déjà deux fois plus grand qu'un écran 1x. L'export n'a jamais été canonique, et le stabiliser
  coûte une ligne, pas un chantier.

  *Décision retenue*, en trois pièces :

  1. **`CANVAS_HEIGHT_CEILING = 720`** — un plafond DUR, qui ne borne que la mémoire des quatre
     calques statiques (~41 Mio à 900x720 en densité 2, contre ~28 Mio aujourd'hui).
  2. **`usefulHeight(bounds, width, pad)`** (`replayLogic.ts`) — le plafond qui compte, calculé
     PAR CARTE. `canvasScale` prend le plus petit des deux rapports : passé le point où la
     largeur devient limitante, un pixel de hauteur de plus n'agrandit plus la carte, il ajoute
     une bande vide. Ce point dépend du ratio de la scène, donc aucune constante ne pouvait
     l'exprimer. `useReplayView` retient `min(offre de l'écran, hauteur utile)`.
  3. **`exportScaleFor(h) = min(960 / h, 2)`** — le facteur de suréchantillonnage cesse d'être
     constant. 960 est la ligne mesurée « double 1004x960 » du tableau de `useReplayView.ts`.
     Une toile de 480 est doublée (le comportement d'avant, au pixel près), une toile de 720
     multipliée par 1,33 : **les deux sortent la même vidéo**. L'export devient donc INVARIANT à
     la taille d'écran, ce qu'il n'était pas avant ce chantier.

  Conséquence : la crainte initiale est non seulement levée, elle est retournée — l'export est
  plus stable après qu'avant.
- **D3 — `CANVAS_HEIGHT_MIN = 360`.** Sous ce seuil la carte n'est plus lisible : on laisse
  alors défiler plutôt que rétrécir indéfiniment.
- **D4 — Quantification 8 px + debounce 120 ms, obligatoires.** Aujourd'hui `view` ne change
  que sur la largeur, qui est stable. Une hauteur dérivée du viewport rend le
  redimensionnement de fenêtre continu : sans garde, chaque tick recuirait les 4 calques
  statiques (dont le sol, 45 000 cellules). Ce n'est pas une bombe RAM — chaque cuisson
  REMPLACE la précédente (la référence est réaffectée, l'ancienne toile est collectée) et le
  nombre de calques est fixé à 4 ; c'est `CANVAS_HEIGHT_CEILING` qui borne leur taille, à
  ~41 Mio en densité 2 contre ~28 aujourd'hui. Mais c'est une tempête CPU/GC qui figerait
  l'onglet.
- **D5 — Le bouton garde sa place, il change de nom.** Le fil d'Ariane porte `← Retour`
  (`router.history.back()` : l'historique, donc n'importe quoi) ; le bouton est un `Link`
  vers une destination fixe. Deux choses différentes sous le même mot. On le nomme par sa
  destination — « Fiche du match » / « Match details » — ce qui est vrai qu'on en vienne ou
  non. Pas de libellé conditionné à l'historique : il clignoterait selon le parcours.
- **D6 — Le doublon de chaîne est supprimé à la source.** `replay.tsx` calcule
  `buildMatchHeadingStr(map_ui, mode_ui)` pour le fil d'Ariane, et `ReplayMatchRecall`
  rappelle la MÊME fonction sur les MÊMES arguments : « Slayer sur Streets » est imprimé
  deux fois à 50 px d'écart. Le rappel se réduit à ce qu'il ajoute vraiment : date et
  playlist.

## Contrainte dure repérée avant de coder

`ReplayCanvas.tsx` fait **665 lignes pour un plafond de 665**
(`placementFamily.guard.test.ts`). Le fichier ne peut pas gagner UNE ligne. La logique de
hauteur vit donc dans un hook dédié, et l'appel qui le branche doit rendre plus de lignes
qu'il n'en prend (l'effet `ResizeObserver` actuel en fait 10, l'appel du hook en fera 2).

## Étape 1 — Compaction de la tête

- [x] 1.1 `MatchBreadcrumb` : props optionnelles `leaf` / `detail` / `action` (additives —
      les deux appels de `MatchViewPage` restent inchangés).
- [x] 1.2 `replay.tsx` : suppression du bloc titre (h1 + bouton + ligne de rappel) ; tout
      passe dans la ligne du fil d'Ariane. Le `h1` devient le segment feuille « Rejeu 2D »
      (sémantique préservée, icône conservée).
- [x] 1.3 `ReplayMatchRecall` : ne rend plus que date + playlist (D6). Props `mapUI`/`modeUI`
      et l'import `buildMatchHeadingStr` supprimés.
- [x] 1.4 i18n rejeu : `back` = « Fiche du match » / « Match details » (D5).
- [x] 1.5 Tests : `ReplayMatchRecall.test.tsx` mis à jour ; les tests de `MatchViewPage`
      passent sans modification (preuve que 1.1 est bien additif).
- Gate : `npm run typecheck` + `npx vitest run src/features/match-replay src/features/match-view`

## Étape 2 — Hauteur élastique

- [x] 2.1 `useReplayView.ts` : `CANVAS_HEIGHT_MIN` / `CANVAS_HEIGHT_MAX` (D2/D3) ;
      `useReplayView` prend `height` en entrée et le pose dans `canvasView`.
- [x] 2.2 Nouveau `useReplayViewport.ts` : `{ width, height }` depuis un seul conteneur —
      `ResizeObserver` pour la largeur, `window.resize` pour la hauteur (une fenêtre
      rétrécie EN HAUTEUR seulement ne change pas la largeur du conteneur : sans ce second
      écouteur, rien ne se passerait), quantification + debounce (D4), clamp.
- [x] 2.3 `ReplayCanvas.tsx` : l'effet `ResizeObserver` cède la place à l'appel du hook ;
      les 8 `CANVAS_HEIGHT` deviennent la hauteur vive. Le fichier doit RÉTRÉCIR.
- [x] 2.4 Tests : `useReplayViewport.test.ts` (clamp haut/bas, quantification, debounce,
      écoute de `window.resize`).
- Gate : `npm run typecheck` + `npx vitest run src/features/match-replay` (dont le
  garde-rail de taille) + `npm run lint`

## Hors périmètre (acté avec l'utilisateur)

- **Zoom / pan navigable** — reporté. Ce chantier donne toute la place que la MISE EN PAGE
  permet ; le zoom est ce qui permettrait d'aller au-delà. À rouvrir quand le besoin réel sera
  « lire le détail » et non « ça ne tient pas » — c'est-à-dire après le gate visuel des
  étapes 1 à 3, qui dira si la question se pose encore.
- ~~Relever le plafond pour exploiter les grands écrans~~ — **TRAITÉ à l'étape 3** (D2 révisée).

## Découvertes (à ne pas traiter ici)

## Journal

### 2026-09-02 — Étape 1 CLOSE (compaction de la tête)

Gate passé dans la session : `npm run typecheck` EXIT=0 (`tsc -b`), puis
`npx vitest run src/features/match-replay src/features/match-view` →
**164 fichiers, 2305 tests, 0 échec**.

Fait :

- `MatchHeader.tsx` — `MatchBreadcrumb` accueille `leaf` / `detail` / `action`, tous
  optionnels. Les deux appels de `MatchViewPage` n'ont PAS été touchés et leurs tests
  passent : la preuve que l'ajout est bien additif, pas une réécriture déguisée.
- `replay.tsx` — le bloc titre (h1 + rappel + bouton, ~84 px) disparaît ; son contenu monte
  sur la ligne du fil. `py-3` devient `pb-3` : le fil porte déjà son `pb-2`.
- `ReplayMatchRecall.tsx` — ne compose plus « mode sur carte » ; l'import
  `buildMatchHeadingStr`, les props `mapUI`/`modeUI`/`locale` et le conteneur `<div>` sont
  supprimés (le composant rend un fragment, ses morceaux deviennent des enfants directs du
  flex du fil). Le doublon de chaîne est donc mort à la source, pas masqué.
- `i18n.ts` — `back` : « Retour au match » → « Fiche du match » / « Match details » (D5).
- `ReplayMatchRecall.test.tsx` — réécrit. Deux tests NEUFS gardent les décisions plutôt que
  le rendu : « ne redit PAS le mode sur la carte » (interdit la réapparition du doublon) et
  « avec une date, le point médian la sépare de ce qui précède » (la feuille du fil est
  désormais toujours devant, la ponctuation change donc de condition).

Mesure : la ligne du fil passe de 44 à ~52 px (le bouton `h-7` la fait grandir de 8 px), et
la page rend 68 px de bloc titre + 16 px de `space-y-4`. **Gain net ~76 px.**

Report assumé : le COMMIT de l'étape attend l'autorisation de l'utilisateur (règle du dépôt
« demander avant tout commit ») — c'est le seul item de la clôture qui reste ouvert, et il
ne dépend pas de moi. Le reste de l'étape est fait et vérifié.

### 2026-09-02 — Étape 2 CLOSE (hauteur élastique)

Gate passé dans la session : `npm run typecheck` EXIT=0 ;
`npx vitest run src/features/match-replay src/features/match-view` → **165 fichiers,
2318 tests, 0 échec** (13 tests neufs) ; `npm run lint` EXIT=0, **0 erreur**, 23
avertissements tous PRÉEXISTANTS (`react-hooks/incompatible-library` sur des `useReactTable`)
et aucun dans un fichier touché ici.

Fait :

- `useReplayView.ts` — `CANVAS_HEIGHT` devient `CANVAS_HEIGHT_MAX` (480, valeur inchangée)
  et gagne `CANVAS_HEIGHT_MIN` (360). `useReplayView` reçoit `height` : `renderWidth` et
  `canvasView` en dépendent désormais tous les deux — donc le dessin ET le survol lisent la
  même projection, ce qui était l'invariant à ne pas casser.
- `useReplayViewport.ts` (neuf) — `fitCanvasHeight` (pure : quantifier puis borner) et le
  hook de mesure. Deux sources d'événements : `ResizeObserver` sur le conteneur pour la
  largeur, `resize` de la fenêtre pour la hauteur — une fenêtre rétrécie EN HAUTEUR seule ne
  change pas la largeur du conteneur, l'observateur ne se déclencherait pas.
- L'espace libre est mesuré DANS LE CADRE QUI DÉFILE (`overflow-y` de l'ancêtre, trouvé par
  style calculé — pas par la balise `main`), pas dans la fenêtre. Pris depuis la fenêtre,
  `rect.top` dépend du défilement : descendre dans la page aurait agrandi le terrain, donc la
  page, donc le défilement. Rapporté au conteneur de défilement et à son `scrollTop`, le
  résultat est le même à toutes les positions.
- `ReplayCanvas.tsx` — l'effet `ResizeObserver` (11 lignes) cède la place à un appel de hook
  (2 lignes) ; les 8 `CANVAS_HEIGHT` deviennent `viewH`, ajouté aux dépendances de `draw`.
  `useState` disparaît de l'import React (plus aucun état local). **Le fichier passe de 665 à
  656 lignes** : il était PILE à son plafond, il rend 9 lignes.

Vérifié sur pièces : `containerRef` est bien l'élément dont `offsetHeight` vaut
`chrome + terrain` (il enveloppe `p-3` → canvas + `ReplayTransport`), et son `rect.top` est
déjà sous le bandeau de score. Le calcul du chrome par soustraction est donc exact sans
mesurer séparément la frise.

Découverte (non traitée, hors périmètre) : `AppShell` utilise `h-screen` (= `100vh`). Le
passage à `h-dvh` corrigerait la barre d'adresse mobile ; c'est global, sans rapport avec le
terrain, et n'a pas été touché.

Report assumé, identique à l'étape 1 : le COMMIT attend l'autorisation de l'utilisateur.

### 2026-09-02 — Étape 3 CLOSE (le terrain grandit — révision de D2)

Déclenchée par la contestation utilisateur de D2, vérifiée puis acceptée : le plafond à 480
était une prudence mal placée (cf. D2 révisée ci-dessus).

- [x] 3.1 `replayLogic.usefulHeight` — le plafond par carte, inverse exact de `fitWidth`.
- [x] 3.2 `useReplayView` — reçoit `freeHeight` (une OFFRE) et rend `renderHeight`, qui est
      `min(offre, usefulHeight)`. La frontière est nette : le viewport mesure la place, la vue
      décide du cadrage — elle seule connaît les bornes de la scène.
- [x] 3.3 `useReplayViewport` — rend `freeHeight`, plafonnée à `CANVAS_HEIGHT_CEILING`. Le
      chrome n'est plus déduit d'une hauteur recopiée mais MESURÉ dans le DOM
      (`canvasRef.getBoundingClientRect().height`) : une valeur recopiée dit ce qu'on croit
      avoir demandé, le DOM dit ce qui est peint — à la première divergence, la mesure suivante
      aurait cru que le chrome changeait de taille et serait partie en oscillation.
- [x] 3.4 `exportScaleFor` + câblage dans `useReplayExport` — une seule ligne changée à
      l'endroit où le facteur était posé en dur ; `canvas.clientHeight` était déjà en portée.
- [x] 3.5 Tests : 6 sur `usefulHeight` (dont l'aller-retour avec `fitWidth`, qui prouve qu'il
      n'y a de bande vide sur aucun axe), 3 sur `exportScaleFor` (dont « sort la MÊME hauteur de
      vidéo quelle que soit la toile »), et le test du hook qui exige désormais que l'offre
      DÉPASSE l'ancienne constante sur un grand écran.

Gate : `npx tsc -b --force` (cache purgé) EXIT=0 ;
`vitest run src/features/match-replay src/features/match-view` → **165 fichiers, 2325 tests,
0 échec** ; `npm run lint` EXIT=0, 0 erreur, 23 avertissements préexistants dont aucun dans un
fichier touché. `ReplayCanvas.tsx` : 656 lignes (plafond 665).

Périmètre NON étendu : le zoom/pan reste hors chantier. Ce lot rend la place que la mise en
page peut donner ; le zoom est ce qui permettrait d'aller AU-DELÀ, et c'est un autre travail
(la projection devient déplaçable, donc le survol, les infobulles et les quatre calques
statiques suivent — c'est là, et seulement là, que la recuisson deviendrait un vrai sujet).
