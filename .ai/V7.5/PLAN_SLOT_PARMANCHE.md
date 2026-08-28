# PLAN — Résolution slot→joueur consciente de l'image (rejeu 2D multi-manche)

Branche : `wt/slot-parmanche` (base `aa1245fa0`). Lot FRONT-END de correctness.

## Problème (établi sur pièces, ne pas re-diagnostiquer)

`rosterLogic.indexBySlot` construit une `Map<slot, valeur>` en balayant TOUTES les vies de
TOUS les joueurs — DERNIER GAGNANT. Un slot de biped est RÉATTRIBUÉ entre manches
(multi-manche) : slot X = joueur A en manche 0, joueur B en manche 1-2. La Map s'effondre →
un slot montre un seul nom/couleur pour tout le match. Symptômes utilisateur (Oddball 3
manches) : « deux DinoR00 et pas de SHROOM », « le point de SHROOM n'apparaît qu'à partir de
1:06 ». Les 5 dérivés collapsent : `nameBySlot`, `colorBySlot`, `sideBySlot`, `markBySlot`,
`indexBySlot` générique.

## Fix

Rendre la résolution slot→joueur CONSCIENTE DE L'IMAGE : au lieu d'une Map figée, un index
`slot -> [vies triées]` et un résolveur `ownerAtFrame(slot, frame)` = la vie qui occupe ce
slot à cette frame. Puis dérivés frame-aware : nom/couleur/marque/camp d'un slot À UNE FRAME.

## Invariant NON NÉGOCIABLE

Mono-manche rend EXACTEMENT le même résultat qu'avant (dans une seule manche, toutes les
vies d'un slot appartiennent au même joueur → frame-aware == dernier-gagnant). Prouvé par
test. Multi-manche corrigé : test avec CONTRE-ÉPREUVE (le collapse aurait donné le mauvais
joueur).

## Étapes

- [x] **E1 — Fondation `rosterLogic.ts`.** Ajouter `SlotOwnership` + `buildSlotOwnership(players)`
  + `ownerAtFrame(slot, frame)`. Convertir les 4 dérivés de rendu (color/side/mark/name) en
  FACTORIES de résolveurs frame-aware `(slot, frame) => valeur`. GARDER `indexBySlot`
  (agrégat, consommé par `equipmentUsageLogic` — hors périmètre canvas). Gate :
  `rosterLogic.test.ts` mis à jour + nouveaux tests mono-manche neutre PROUVÉ + multi-manche
  corrigé avec contre-épreuve (scénario « deux DinoR00 / SHROOM »).

- [x] **E2 — Hook `useSlotIdentity.ts`.** Retourner résolveurs frame-aware `(slot, frame)`.
  Convertir `distinctSlotColors` en résolveur frame-aware. Retirer l'export `slotColors` map
  (remplacé par `colorOfSlot`). Gate : `useSlotIdentity.test.ts` + typecheck.

- [x] **E3 — Marqueurs joueurs (`replayMarkers.ts`).** `MarkerStyle.colorOfSlot/markOfSlot/
  nameOfSlot` → `(slot, frame)`. Résoudre à la frame BORNÉE à la fenêtre de la vie (clamp) :
  vivant = frame ; croix de mort = fin de fenêtre (sinon régression : la croix ne se dessine
  plus). Gate : `replayMarkers.test.ts`.

- [x] **E4 — Événements (`fireMark.ts`, `replayDraw.ts`).** fireMark : `colorOfSlot(slot, frame)`
  (vie vivante à frame). killFx : `colorOfSlot(slot, e.frame)` (couleur du tueur à l'INSTANT
  du kill). Gate : `fireMark.test.ts`, `meleeStar.test.ts`.

- [x] **E5 — Équipement / capteur (`equipmentPlacementsLayer.ts`, `threatSensor.ts`,
  `equipmentZones.ts`).** `sideOfSlot`/`colorOfSlot` → `(slot, frame)`. Poseur résolu à
  `p.t0` (instant de pose, poseur vivant) ; vie proche résolue à `pingFrame`/`query.frame`.
  Ajouter `ownerFrame` à `SensorReveal` pour colorer la marque de révélation par le poseur.
  Gate : `equipmentPlacementsLayer.test.ts`, `threatSensor.test.ts`, `equipmentZones.test.ts`.

- [x] **E6 — Composants (`ReplayCanvas.tsx`, `ReplayCanvasTips.tsx`, `ReplayTeams.tsx`).**
  Passer la frame partout ; `ownerNameOf(slot, t0)` pour l'infobulle de pose ; `sideOfSlot`
  frame-aware pour le `fxScene` des fiches. Gate : typecheck.

- [x] **E7 — Gates finaux.** `npm run typecheck`, `npm run lint` (0 erreur),
  `npm run test` (dossier match-replay + suite web).

- [x] **E8 — Régression P2 (revue adversariale) : couleur des objets LÂCHÉS à la mort.** Un
  objet `origin='dropped'` (équipement déployable ET grenades) a `t0 = finVie du poseur + 1` :
  `colorOfSlot(owner, t0)` strict rend `null` → l'objet est peint en `neutral` au lieu de la
  couleur d'équipe du poseur. Régression VISIBLE PAR DÉFAUT (showDropped ON) et sur un film
  MONO-manche → viole l'invariant. Même cause pour un P3 : killFx d'un kill posthume/échange
  (`e.frame` au-delà de la fin de vie du tueur). FIX BORNÉ : `SlotOwnership.ownerAtFrameOrLast`
  (vie couvrante, sinon la vie juste précédente = le lâcheur/tueur — frame-CORRECT, PAS le
  dernier-gagnant), réservé à DEUX consommateurs de FRONTIÈRE via `colorOfSlotOrLast` :
  (1) couleur du poseur des poses (`inkOf`), (2) couleur de l'effet de mort (killFx). Marqueurs
  de vie / plaques / capteur GARDENT la résolution stricte (rien dans les trous). Corriger le
  commentaire `inkOf` (doc inversée). Gate : rosterLogic + equipmentPlacementsLayer (double
  frame-dépendant + contre-épreuve dropped@t0=finVie+1 : strict→neutre échoue, or-last→équipe
  passe) + typecheck + lint + suite match-replay.

## Découvertes (NON traitées — hors périmètre)

- `equipmentUsageLogic.ownerOfSlot` (via `indexBySlot`) : MÊME effondrement en multi-manche
  pour l'AGRÉGAT d'usage d'équipement (tractions grappin / épisodes / poses attribués au
  dernier propriétaire du slot). C'est une table de stats, pas le calque canvas → hors
  périmètre du bug rapporté. Corrigeable par résolution à l'instant du geste (grapple/episode
  ont un temps, grenades par filmIndex). NOTÉ, non traité.
- Le crâne d'Oddball n'a pas de donnée avant ~frame 254 (~25 s, 1re vie libre) : côté
  donnée/décode Go, hors de ce lot.

## Journal

- [2026-08-28] Plan écrit et commité avant toute mesure (plan-execution). Surface cartographiée
  sur pièces : 11 fichiers de production + tests. Design tranché : `indexBySlot` conservé
  (agrégat), 4 dérivés → frame-aware, frame threadée à chaque consommateur avec la frame
  CORRECTE (draw clampée pour marqueurs, `e.frame` pour kills, `t0` pour poseurs, `pingFrame`
  pour cibles capteur).
- [2026-08-28] E1 CLOS. `buildSlotOwnership`/`ownerAtFrame` + `colorResolver`/`sideResolver`/
  `markResolver`/`nameResolver` (frame-aware) ; `indexBySlot` conservé (doc clarifiée = agrégat).
  Gate `rosterLogic.test.ts` : 36/36 vert, incluant neutralité mono-manche PROUVÉE + correction
  multi-manche avec CONTRE-ÉPREUVE (« deux DinoR00 / SHROOM », le slot 512 rend SHROOM à f=25 et
  DinoR00 à f=220 — impossible avec la Map figée). Env : junction node_modules vers le worktree
  principal (deps identiques). NB thought_log non touché (texte au CR, cf. consigne du lot).
- [2026-08-28] E2 CLOS (useSlotIdentity frame-aware, `distinctColorResolver`, slotColors retirée) ;
  E3 CLOS (marqueurs, résolution clampée à la fenêtre de vie + test contre-épreuve du clamp) ;
  E4 CLOS (fireMark@frame, killFx@e.frame) ; E5 CLOS (poseur@t0, cible@pingFrame, `SensorReveal.
  ownerFrame` ; contre-épreuves poseur/cible) ; E6 CLOS (composants, tsc -b --force vert).
- [2026-08-28] E7 CLOS. Gates finaux : typecheck `tsc -b --force` EXIT=0 ; lint touché 0 erreur
  (1 warning `objectiveObjects` PRÉ-EXISTANT à aa1245fa0, hors périmètre) ; vitest match-replay
  1642/1642 ; suite web complète 5213 passed / 14 skipped / EXIT=0. LOT LIVRÉ. Avant merge :
  ADVERSARIAL-REVIEW requise (calque roster/plaques partagé par tout le rejeu). Ne PAS pousser.
- [2026-08-28] E8 CLOS. Revue adversariale : régression P2 réelle (couleur des objets LÂCHÉS à la
  mort peints en neutre au lieu de la couleur d'équipe du poseur — `t0 = finVie+1`, `ownerAtFrame`
  rend null ; visible par défaut, même mono-manche → violait l'invariant). Fix borné :
  `ownerAtFrameOrLast` + `colorOfSlotOrLast`, câblés SEULEMENT aux 2 frontières (poses `inkOf`,
  killFx). Marqueurs/vies/capteur gardent la résolution stricte. Commentaire `inkOf`/`PlacementInk`
  corrigé (doc inversée). Contre-épreuve : dropped@t0=finVie+1 strict→NEUTRE (échoue) vs
  or-last→ÉQUIPE (passe) ; unités `ownerAtFrameOrLast` (couvrante ; finVie+1→précédente ; trou
  multi-vies→précédente pas suivante ; avant 1re vie→null). Cliquet de taille ReplayCanvas re-tenu
  (679/679, commentaires condensés). Gates : tsc EXIT=0 ; lint 0 err (warning pré-existant) ;
  vitest match-replay 1650/1650. Ne PAS pousser.
