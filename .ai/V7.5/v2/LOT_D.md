# Lot D — Modèle web du rejeu — journal d'exécution

Plan : `.ai/PLAN_V2_REJEU_FILM_2026-09-05.md`, lot D. Worktree `LevelUp-wt-v2-web-modele`,
branche `feat/v2-web-modele`, base `a21fd77f4`. Contrat : skill `plan-execution`.

---

## Tâche D-I (horloges) — CLOSE le 2026-09-06

### Items

- [x] **D.1 (P0-7) — un seul axe temporel pour l'onglet Chronologie.** `commit c141e30b0`.
  `apps/web/src/lib/replay/matchClock.ts` porte les conversions entre les trois axes (match,
  film, gameplay), avec le contrat de `apps/go-api/internal/domain/match_view.go` (champ
  `T0Ms`, lu sur pièces) recopié dans l'en-tête. La courbe de score ET les barres d'instants
  (`_scoreCurve.ts`, `_scoreEvents.ts` — même bloc de l'onglet, même défaut, cf. « Décisions »)
  passent par elle ; `header.t0_ms` descend de `MatchViewPage` jusqu'aux deux cartes. Le test
  qui verrouillait la prémisse « borné au coup d'envoi » assert désormais les abscisses
  attendues, calculées à la main. Garde-rail `matchClock.guard.test.ts`.
- [x] **D.2 (P0-5) — un seul verdict d'horloge pour la page de rejeu.** `commit 4fb7de28e`.
  `features/match-replay/model/replayClock.ts` ; les cinq sites cités par le registre le
  consomment (`replayWindow`, `killFeedLogic`, `replayMediaLogic`, `presenceFeed`,
  `seatLogic`). Garde-rail `model/replayClock.guard.test.ts` : plus aucune lecture de
  `originMs` dans la feature ni dans la route, hors de l'horloge.
- [x] **D.3 (J2, résidu P0-6) — un seul recalage, une seule horloge visible.** `commit
  a3f01e4ab`. `killFx.ts` et `replaySound.ts` reçoivent les kills DÉJÀ recalés
  (`killsOfFeed(feedEntries)`) et ne rejouent plus `alignFeed` ; `ReplayCanvas` ne reçoit plus
  ni `kills` ni `t0Ms`. Les marques de la frise écrivent leur instant avec le formateur du
  rejeu (troncature) au lieu de celui de `lib/formatters` (arrondi) ; garde-rail
  `replayClockFormat.guard.test.ts`.
- [x] **D.4 (J3, J5) — deux codes morts supprimés.** `commit 198ab7e6c`. `replaySchemaLogic.ts`
  + son garde + la mention qu'en faisait `deltaLayersContract.guard.test.ts` ; `roundAtFrame`,
  son type `RoundReading` et ses six assertions.
- [x] **D.5 (N1, N2, N3) — géométrie extraite et sous oracle.** `commit 10aa884c3`.
  `_kdCumul.ts` (31 cas à oracle avec `_cadence.ts`) et `_cadence.ts` extraits sur le patron de
  `_scoreCurve.ts` ; `MatchCombatCtfOverlay.test.tsx` réécrit sur l'option ECharts entière et
  les abscisses des deux conventions de placement ; `formatClockMShort` aux quatre sites du
  format `MmSSs` + garde-rail `clockMShort.guard.test.ts`.

### Gate D-I

| Gate | Commande exacte | Dernière ligne |
|---|---|---|
| Typecheck (cache forcé) | `cd apps/web && npx tsc -b --force` | (aucune sortie) `EXIT=0` |
| Lint | `npm --prefix apps/web run lint` | `✖ 27 problems (0 errors, 27 warnings)` |
| Vitest (périmètre du lot) | `cd apps/web && node_modules/.bin/vitest run --pool=forks src/features/match-replay src/features/match-view src/lib/replay src/lib/formatters` | `Tests  2883 passed (2883)` |
| Vitest (suite web entière) | `cd apps/web && node_modules/.bin/vitest run --pool=forks` | `Tests  6287 passed \| 14 skipped (6301)` |
| Couleurs | `node tools/lint-no-hardcoded-colors.mjs` | `lint-no-hardcoded-colors: clean (0 violation)` |
| Imports croisés | `node tools/lint-cross-feature-imports.mjs` | `Info : 7 <= plafond P8.5 (7). Pas d'échec` |
| Build de production | `npm --prefix apps/web run build` | `✓ built in 2.73s` (2 198 modules) |
| Vitest + couverture | `cd apps/web && node_modules/.bin/vitest run --pool=forks --coverage` | `Lines : 78.03% ( 19263/24685 )` |

Les 27 avertissements de lint sont préexistants (`react-hooks/incompatible-library` sur les
tableaux TanStack, `squad/` et `explorer/`) ; il y en avait 28 au commit de base — le lot en
retire un, n'en ajoute aucun. Zéro test skippé ajouté, aucun garde assoupli.

**Specs de rasterisation (Playwright).** `npx playwright --version` → `1.62.1`, mais le
binaire Chromium manquait : installé (`npx playwright install chromium`) pour pouvoir jouer le
gate. Résultat, `cd apps/web && npx playwright test e2e/replay-explosion-raster.spec.ts
e2e/replay-muzzle-raster.spec.ts --reporter=line` → **`2 failed`, `1 passed`** :

- `replay-muzzle-raster.spec.ts` : **vert**.
- `replay-explosion-raster.spec.ts` : **rouge, cause PRÉEXISTANTE et non causée par ce lot.**
  Le harnais compte les imports de VALEUR de `replayDraw.ts` et en exige 7 ; le fichier en
  porte 6. Mesuré des deux côtés : `git show a21fd77f4:…/replayDraw.ts | grep -E "^import[[:space:]]+" | grep -vc "^import type"`
  → `6`, et `6` également à HEAD. Le lot D ne touche pas `replayDraw.ts` (`git diff a21fd77f4
  --stat -- …/replayDraw.ts` : vide). Le ratchet est donc désaccordé depuis avant ce lot —
  cohérent avec M1 du registre : ces deux specs n'ont JAMAIS tourné en CI (job gaté
  `pull_request`). **Non traité ici** : le fichier relève de F.6, qui les met en CI (règle 7,
  périmètre fermé). Correctif d'une ligne : passer `moduleSource('replayDraw.ts', 7)` à `6`
  après avoir vérifié que les dépendances injectées par `harnais()` suffisent encore.

**Pouvoir de détection des garde-rails, vérifié sur pièces.** Un garde qui ne rougirait sur
rien ne garderait rien : chacun a été confronté aux fichiers du commit de BASE, où le défaut
qu'il vise existait.

- `matchClock.guard.test.ts` : le motif de conversion trouve les **5** sites de
  `git show a21fd77f4:…/_scoreCurve.ts` et `_scoreEvents.ts` (lignes 86, 102, 123 et 85, 118),
  et **0** à HEAD.
- `model/replayClock.guard.test.ts` : le motif `(?<![Cc]lock)\.originMs` **ATTRAPE** les cinq
  fichiers de base (`killFeedLogic`, `presenceFeed`, `replayMediaLogic`, `replayWindow`,
  `seatLogic`) et **aucun** à HEAD.
- `clockMShort.guard.test.ts` : le littéral `MmSSs` **ATTRAPE** les quatre fichiers de base
  (`MatchImpactBadgesBar`, `MatchKDCumulChart`, `_chartSeries`, `SynthesisBipolaireChart`),
  **pas** `components/ui/match-card.tsx` (format « 2m 05s », avec une espace — exclusion
  voulue), et **aucun** à HEAD.
- `replayClockFormat.guard.test.ts` : son second cas ré-affirme que le formateur du rejeu
  TRONQUE (`Math.floor(ms / 1000)` présent dans `replayLogic.ts`), sans quoi l'arbitrage
  n'aurait plus d'objet.

**Surveillance CI : ABANDONNÉE sur consigne du superviseur.** La branche est poussée
(`158138ded`). Le quota GitHub étant épuisé (403 secondaire sur l'API Actions, partagé avec les
lots exécutés en parallèle), le verdict par job sera constaté par le superviseur. Dernière
observation avant la coupure, sur le run `33996542564` : **8 jobs verts sur 9** — `Go Build +
Test` (ubuntu et windows), `Go Lease Enforcement (ADR 0013)`, `OpenAPI Lint`, `Go Contract
Test`, `Frontend (TypeScript + Vite build)` (toutes ses étapes ✓, y compris `Ratchet knip`,
`Lint multi-titres` et `Run tests (Vitest) + coverage`), `Go Lint (golangci-lint)` — le neuvième
(`Go Coverage + Baseline`) était encore `in_progress`, et `E2E React (Playwright)` `skipped`
(job gaté `pull_request`, cf. M1). Le lot ne touche **aucun** fichier Go
(`git diff a21fd77f4 --name-only | grep -c "^apps/go-api/"` → `0`), donc rien de ce qu'il change
ne peut peser sur le job restant.

---

## Décisions prises

1. **L'axe commun de l'onglet Chronologie est celui de `event_time_ms`, ancré sur
   `header.t0_ms` (D.1).** Deux coups d'envoi coexistent : celui de l'API (estimé des
   `first_joined_time`) et celui du film (`doc.t0FilmMs`, mesuré, que le rejeu 2D préfère pour
   son horloge de lecture). C'est le premier qui ancre l'axe, et il ne peut pas en être
   autrement : c'est l'ancre que le producteur a retranchée pour fabriquer `event_time_ms`. S'y
   substituer déplacerait « Frags cumulés », la série qui est déjà juste. `t0FilmMs` est porté
   à côté dans `MatchClock`, pour qui affiche une horloge de LECTURE.
2. **`_scoreEvents.ts` est traité avec `_scoreCurve.ts` (D.1).** Le plan cite
   `MatchViewTabChronology.tsx:84-121`, qui couvre les DEUX rendus du même bloc (barres ou
   courbe selon `score_timeline_kind`), et le registre nomme explicitement `_scoreEvents.ts:118`
   au même titre. Corriger un seul des deux aurait laissé le défaut dans la branche sœur.
3. **Sans origine publiée, le bloc « Score dans le temps » ne rend plus rien (D.1).** L'écart
   entre les deux axes vaut alors 3,6 à 50,8 s d'inconnu, et le producteur REFUSE lui-même de
   publier une origine qu'il ne peut pas établir. Mesure du parc local (106 artefacts servis
   sous `data/cache/replays/halo_infinite/`) : 5 n'ont pas d'`originMs`, et les 5 portent
   `coverage.originResolved: false` — ils étaient donc DÉJÀ écartés en amont par
   `filmClockTrusted`. **Aucune carte ne disparaît de l'écran.**
4. **Politique de repli unique de la page de rejeu (D.2) : « l'origine est celle que
   l'ARTEFACT PUBLIE ; sans elle, la page n'a pas d'horloge et aucune surface ne place quoi que
   ce soit sur l'axe du film ».** Une seule exception, NOMMÉE et MESURÉE : le fil des
   éliminations, seule surface à disposer d'une seconde source (appariement des kills aux fins
   de vie, médianes mesurées +3 678 / +10 589 / +39 856 ms sur trois témoins, à 20-70 ms de
   l'origine publiée quand les deux existent). Rien d'autre ne mesure : une seconde surface qui
   s'inventerait une origine réintroduirait exactement le défaut P0-5.
   Conséquences pour un artefact sans origine (les 5 du parc, déjà en mode dégradé — la fenêtre
   de gameplay y est nulle) : la piste Médias se tait au lieu de poser ses captures à l'origine
   zéro (elles pouvaient se retrouver à ~40 s du frag qu'elles montrent), et un relais de siège
   n'est plus daté — l'affichage retombe sur le repli déjà documenté dans `seatLogic`, chacun
   garde son siège.
5. **L'horloge exige `frameIntervalMs` (D.2).** Les 106 artefacts du parc le publient, toujours
   à 100 ms. L'exiger ne retire donc rien et supprime deux replis divergents qui traînaient
   (60 images/s dans `replayLogic`, 100 ms dans `seatLogic`).
6. **Le formateur retenu pour le rejeu est celui qui TRONQUE (D.3).** C'est la convention d'un
   lecteur, c'est ce que lisent toutes les surfaces visibles du rejeu, et c'est donc le sens qui
   ne déplace aucun pixel. L'arrondi (`formatClockMMSS`) reste celui du reste du dépôt, où il
   est le bon choix pour une étiquette d'axe. Exception au garde-rail : `ReplayMediaLightbox`,
   qui date un CLIP vidéo et non le match.
7. **`formatClockMShort` est une variante manquante, pas une quatrième copie (D.5).** Le dépôt
   avait déjà tranché ce partage pour `M:SS` (`formatDurationMMSS` / `formatClockMMSS`) : une
   DURÉE nulle n'existe pas, un INSTANT nul est le coup d'envoi. Chacun des quatre sites garde
   chez lui la seule décision qui lui appartient — l'unité d'entrée (secondes ou ms) ou ce qu'il
   fait d'une absence (`null` pour un badge sans instant, `Xs` sous la minute en synthèse).

---

## Découvertes (hors périmètre, non traitées)

1. **`e2e/replay-explosion-raster.spec.ts` est rouge depuis avant ce lot** (détail au gate
   ci-dessus). Relève de F.6.
2. **`SynthesisBipolaireChart` rendait « 1m60s »** sur une durée de vie moyenne comprise entre
   119,5 et 120 s : l'ancien code prenait `Math.floor(v/60)` minutes et `Math.round(v % 60)`
   secondes, deux arrondis incohérents. Le passage par le formateur canonique arrondit d'abord
   la valeur entière, ce qui supprime la sortie invalide (seul écart de sortie du lot en dehors
   des corrections voulues, et il ne concerne qu'un intervalle d'un demi-seconde).
3. **`REGISTRE_REPORTS.md` ligne 449** porte l'entrée `EXPECTED_REPLAY_SCHEMA_VERSION` dont la
   condition de reprise (« À LA CLÔTURE DU LOT D … supprimer la constante ET son garde ») est
   exécutée par D.4. L'entrée reste à amender par le superviseur : un exécuteur n'écrit que son
   journal de lot et le thought_log.
4. **`ReplayCanvas.tsx` est à 661 lignes** pour un plafond de 665
   (`placementFamily.guard.test.ts`), contre 664 au commit de base : le lot rend 3 lignes de
   marge, il n'en consomme aucune. D-II (D.11) remplace ce cliquet par `max-lines` eslint.

---

---

## Tâche D-II, premier temps (D.6, D.7, D.8) — CLOS le 2026-09-06

### Items

- [x] **D.6 — la jointure devient une fonction pure testée.** `commit 0ff61938d`.
  `model/replayModel.ts` porte `buildReplayModel(doc, matchView, settings)` (identité, marques,
  horloge, fenêtre de gameplay, roster, fil recalé, médias, score final, countdown) ;
  `model/useReplayModel.ts` ne fait que la mémoïser. La route passe de **395 à 316 lignes** et
  de **treize `useMemo` à trois** — restent le fond de carte et les zones nommées (trois AUTRES
  requêtes, hors de cette jointure) et le son de fin de partie (locale-aware). 20 cas à oracle
  écrit à la main.
- [x] **D.7 — l'ordre des calques devient une donnée testée.** `commit 246f3bbf2`.
  `replayCompose.ts` porte `LAYER_ORDER` (les 25 calques, du sol vers le sujet), `sceneLayers`
  (la condition de chacun : interrupteur du tiroir OU absence de matière) et `composeScene` (la
  boucle, qui rend la liste de ce qu'elle a PEINT). 25 cas au contexte enregistreur. Côté
  canvas, `draw` passe de **222 à 22 lignes** ; la table de liaison devient `buildScene`.
- [x] **D.8 — la position de lecture publiée sort de l'arbre React.** `commit 20c8abe9e`.
  `model/playbackStore.ts` : le canvas publie, la page lit (`usePlaybackFrame`,
  `useSyncExternalStore`), un magasin par page. La copie React de la route et la prop de
  rappel `onFrameChange` disparaissent. 14 cas.
- [x] **D.8 — `useReplayDrawer` : vérifié, il RESTE.** Son unique consommateur est
  `ReplayCanvas` (`grep -rn useReplayDrawer apps/web/src` : une importation, un appel, deux
  mentions dans le garde de taille). Il groupe ~120 lignes que le canvas ne peut pas absorber —
  il est à 651 lignes pour un plafond de 665. La condition de l'item (« disparaît si plus rien
  ne le justifie ») est donc vérifiée et NON remplie : le justificatif tient.
- [!] **D.8 — « `ReplayTransport` et `ReplaySettingsDrawer` remontent frères » : NON TRAITÉ,
  bloqué sur une dépendance mesurée.** Les deux sont déjà frères dans le DOM (enfants de la
  même `div` racine du canvas) ; les promouvoir en frères de `<ReplayCanvas/>`, montés par la
  route, exige de faire remonter avec eux l'état dont ils vivent :
  - `ReplaySettingsDrawer` reçoit `drawer.panel`, dont les neuf `available.*` viennent de neuf
    hooks de CALQUE qui vivent dans le canvas (`calloutZones`, `placements`, `weaponPads`,
    `groundWeapons`, `flags`, `vipCrown`, `skullCarrier`, `bombCarrier`, `vehicles`) ;
  - `ReplayTransport` reçoit `playback`, `timeline`, `sound`, `capture`, `settings` et
    `clockRef`, tous produits par des hooks du canvas.
  Les faire remonter, c'est démonter le composant-dieu — le contenu de D.9 et D.10, pas d'un
  item de câblage. Et le tiroir est positionné en `absolute` par rapport à la `div` racine du
  canvas : changer son parent change son bloc conteneur, donc sa position à l'écran, sous une
  contrainte « aucun changement visuel » que les deux specs de rasterisation ne couvrent pas
  (elles peignent des primitives hors DOM). **Question ouverte pour le superviseur** : est-ce à
  rattacher à D.9/D.10, ou à traiter comme un item propre avec un gate visuel dédié ?

### Gate du premier temps de D-II

| Gate | Commande exacte | Dernière ligne |
|---|---|---|
| Typecheck (cache forcé) | `cd apps/web && npx tsc -b --force` | (aucune sortie) `TSC_EXIT=0` |
| Lint | `npm --prefix apps/web run lint` | `✖ 27 problems (0 errors, 27 warnings)` |
| Vitest (dossiers du lot) | `cd apps/web && node_modules/.bin/vitest run --pool=forks src/features/match-replay src/features/match-view src/lib/replay` | `Tests  2871 passed (2871)` |
| Vitest (suite web entière) | `cd apps/web && node_modules/.bin/vitest run --pool=forks` | `Tests  6339 passed \| 14 skipped (6353)` |
| Couleurs | `node tools/lint-no-hardcoded-colors.mjs` | `lint-no-hardcoded-colors: clean (0 violation)` |
| Imports croisés | `node tools/lint-cross-feature-imports.mjs` | `Info : 7 <= plafond P8.5 (7). Pas d'échec` |
| Build de production | `npm --prefix apps/web run build` | `✓ built in 4.11s` |
| Rasterisation (témoin) | `cd apps/web && npx playwright test e2e/replay-explosion-raster.spec.ts e2e/replay-muzzle-raster.spec.ts --reporter=line` | `3 passed` |

Les 27 avertissements de lint sont ceux de la baseline : le premier temps en a introduit puis
retiré deux familles (voir « Décisions »), il n'en laisse aucun.

**Le témoin de rasterisation est vert AVANT et APRÈS chaque item.** Il a fallu réparer son
harnais pour cela, et la dérive était double, pas simple : `drawGrenadeRestLayer` avait quitté
`replayDraw.ts` pour `grenadeRestLayer.ts` (3 imports de valeur, et non 7). Le correctif local
est dans `apps/web/e2e/replay-explosion-raster.spec.ts` — deux lignes, à réconcilier au merge
avec le lot F, qui met ces specs en CI (F.6).

### Décisions prises

8. **Le modèle est la jointure ARTEFACT × VUE MATCH, et rien d'autre (D.6).** Le fond de carte
   et les zones nommées viennent de trois autres requêtes : les y faire entrer aurait demandé
   d'élargir la signature à des données qui ne participent pas de cette jointure. Le son de fin
   de partie dépend de la LANGUE (voix d'annonceur) : il reste à la page.
9. **Une seule mémo pour toute la jointure (D.6).** Les trois entrées sont des résultats de
   requêtes dont l'identité ne change qu'au chargement ; la page se re-rend ~6,7 fois par
   seconde pendant la lecture et aucun de ces rendus ne refait le calcul — exactement comme les
   treize mémos remplacés.
10. **`players` est exposé par le modèle bien que `ReplayTeams` en rebâtisse un (D.6).** Le
    champ a un consommateur interne (les lignes d'entrée/sortie) ; la seconde construction est
    la même fonction pure sur les mêmes entrées, donc le même résultat — mesuré en vérification
    adverse (V-WEB-1 constat 2), et sa migration relève du lot des canoniques.
11. **Le contrat de calque n'invente pas une troisième convention (D.7).**
    `LayerPaint = (ctx, frame, dpr)` est la signature la plus large des onze `paint` que les
    hooks exposent déjà : cinq lisent la densité de pixels, six l'ignorent, et TypeScript
    accepte les seconds là où le premier est attendu. Les dix calques restants sont des
    fonctions libres à signature riche (`drawTracksLayer`, `drawShotsLayer`…) : le canvas les
    LIE à cette convention plutôt que de les réécrire — leur signature est ce qui les rend
    testables une par une, et l'uniformiser aurait coûté leurs tests sans rien apporter.
12. **`buildScene` porte une exemption R5 assumée (D.7).** 179 lignes, mais c'est une TABLE :
    une entrée par calque, aucun embranchement, aucun ordre. La découper par famille donnerait
    trois listes de vingt liaisons à tenir en phase pour la même table. Le cliquet de taille du
    canvas a d'ailleurs mordu pendant le lot (666 > 665) : les justifications d'ORDRE ont alors
    migré dans `LAYER_ORDER`, où elles ont désormais leur seul foyer — le fichier retombe à
    651 lignes.
13. **Le magasin RETIENT la position publiée au lieu de lire la cellule vivante (D.8), et ce
    n'est pas un doublon.** `useSyncExternalStore` exige un instantané STABLE entre deux
    notifications : lire la cellule que la boucle avance à 60 Hz ferait lire à React une valeur
    différente à chaque appel, ce qu'il signale comme une boucle. « Où en est le tracé » et
    « ce que la page affiche » sont deux grandeurs, séparées par un bridage à 150 ms qui vit
    chez celui qui peint.
14. **La cellule de dessin reste un `useRef` du canvas (D.8).** La faire porter par le magasin
    et la passer en prop a été essayé : le compilateur React refuse de mémoïser un composant
    qui reçoit une cellule de ref par prop (14 avertissements neufs, `Cannot access refs during
    render`). Le magasin ne porte donc que la publication — ce qui est aussi ce que le point 13
    impose.

### Découvertes (premier temps de D-II)

5. **La position de lecture ne se remet pas à zéro d'un match à l'autre.** Ni avant ce lot (la
   route gardait `useState(0)`, le canvas son `frameRef`), ni après. Naviguer d'un rejeu à un
   autre sans démonter la route laisserait la position précédente. Non traité : le corriger
   serait un changement de comportement, hors de la contrainte de ce lot.
6. **`e2e/replay-explosion-raster.spec.ts` portait DEUX dérives, pas une** (détail au gate).
   Le correctif d'une ligne annoncé n'aurait pas suffi.

---

## Tâche D-II, second temps — PARTIEL, arrêté après D.10 le 2026-09-06

### Items

- [x] **D.9 — cinq canoniques, cinq garde-rails, toutes les copies migrées.** Quatre commits.
  - `32607ff66` **K3, `replayView.ts`** : le type du cadrage était redéclaré 8 fois, et le
    passage monde → canvas réécrit en dépaquetant ses quatre champs — 29 sites dans 22
    fichiers, plus 7 pour l'échelle. `CanvasView`, `projectTo(view, p)` et `scaleOf(view)`
    prennent le cadrage ENTIER : il n'y a plus d'ordre à retenir entre `width` et `height`,
    deux nombres du même type que le compilateur ne peut pas départager. 36 sites migrés.
    Exception structurelle : `replayLogic.layerOffset` garde l'écriture longue (il DÉFINIT
    `worldToCanvas`, passer par `replayView` ferait un cycle).
  - `15613a2b7` **K1, `buildLivesBySlot` / `lifeOfSlotAt`** : quatre calques réécrivaient
    byte pour byte l'index des vies par slot et la relecture « la vie qui couvre l'image ».
    Les deux rejoignent `buildLivesByXuid` dans `livesPosition.ts` ; le garde-rail existant
    couvre désormais les deux index.
  - `bc57c117c` **K2, `matchSides.ts`** : « quelle est MON équipe » était écrite quatre fois
    — trois copies, plus une canonique cachée dans le module des SONS d'objectif, un foyer
    que personne ne va chercher pour peindre une onde ; « l'équipe de chaque xuid » deux
    fois. Le garde est resserré sur la FORMULE et non sur ses ingrédients : chercher la ligne
    `is_me` reste permis pour en lire autre chose.
  - `803daf16a` **K4, `carriedGlyphPulse.ts` et K5, `replaySpans.ts`** : la respiration des
    trois glyphes portés (le risque n'était pas la désynchronisation — les trois modes ne
    co-occurrent jamais — mais la dérive silencieuse) ; et le prédicat « cet intervalle
    couvre cette image », écrit DOUZE fois en deux orthographes (et non dix : `zoneSound.ts`
    en portait deux de plus que le registre n'en comptait).
- [x] **D.10 (M4) — le garde du porteur dérive sa liste de la source.** `commit 5fe9d1957`.
  Il nommait cinq calques à la main ; il lit maintenant le code — tout module qui se donne un
  résolveur de position (`const posOf = …`) est un lecteur de porteur et doit le prendre au
  résolveur commun, la version mémoïsée s'il est monté dans React, le constructeur sinon.
- [x] **D.10 (M5) — le garde des encres dérive sa liste du type.** `commit 5fe9d1957`. Il
  promettait « chaque `InkVar` » et n'en vérifiait QU'UNE sur six. Deux exigences désormais
  distinguées : toute encre du type doit être déclarée (sans quoi `readInk` rend `''` et le
  canvas peint avec l'encre précédente), et celles propres au rejeu doivent l'être dans les
  DEUX thèmes — elles n'ont pas de valeur héritée sur laquelle retomber.
- [!] **D.10 (les « 11 faux hooks ») — NON TRAITÉ, et la mesure invite à trancher avant.**
  Le décompte d'abord : ils sont QUINZE, pas onze, à ne porter ni état ni effet
  (`useReplayAbilityFx`, `useReplayBombBlast`, `useReplayBombCarrier`, `useReplayFx`,
  `useReplayGrenadeRest`, `useReplayInks`, `useReplayObjectiveObjects`, `useReplayPlacements`,
  `useReplaySkullCarrier`, `useReplayTiming`, `useReplayView`, `useReplayVipCrown`,
  `useSlotIdentity`, `useTeamCascades`, et `useReplayTimeline` qui, lui, appelle bien deux
  vrais hooks). Le fond ensuite : **leur logique est DÉJÀ pure et déjà testée ailleurs.**
  Témoin mesuré, `useReplayVipCrown` (66 L, 3 mémos) — il ne contient qu'un `useCarrierPosAt`,
  un objet de style mémoïsé et un `useCallback` qui appelle `drawVipCrown`, fonction pure
  testée dans `vipCrownLayer.test.ts` ; `useSlotIdentity` (211 L, 11 mémos) ne porte que deux
  boucles. Ce que ces hooks contiennent, c'est de la MÉMOÏSATION et du câblage.
  Les convertir en fonctions pures aurait donc trois effets, et aucun n'est celui visé :
  (1) aucune testabilité gagnée — la logique est déjà hors React ; (2) une quinzaine de
  `useMemo` déplacés dans `ReplayCanvas`, qui est à 651 lignes pour un plafond de 665 et que
  D.11 veut faire maigrir ; (3) un risque de perte de mémoïsation sur une boucle à 60 images
  par seconde, sous une contrainte « aucun changement de comportement » qu'aucun gate ne
  mesure.
  **Question au superviseur** : le constat vise-t-il le NOM (`use*` promet un état qu'ils
  n'ont pas — on renommerait alors en `buildX` + `useX` mince, patron de D.6) ou la
  STRUCTURE (les convertir vraiment) ? Le premier est un déplacement de mémo sans risque ; le
  second est celui que la mesure déconseille.
- [ ] **D.11, D.12, D.13, D.14 — non commencés.** Arrêt propre demandé par le superviseur en
  cas de saturation de contexte plutôt qu'un travail bâclé (cf. « Reste à faire »).

### Gate du second temps (partiel)

| Gate | Commande exacte | Dernière ligne |
|---|---|---|
| Typecheck (cache forcé) | `cd apps/web && npx tsc -b --force` | (aucune sortie) `TSC_EXIT=0` |
| Lint | `npm --prefix apps/web run lint` | `✖ 27 problems (0 errors, 27 warnings)` |
| Vitest (suite web entière) | `cd apps/web && node_modules/.bin/vitest run --pool=forks` | `Tests  6354 passed \| 14 skipped (6368)` |
| Couleurs | `node tools/lint-no-hardcoded-colors.mjs` | `lint-no-hardcoded-colors: clean (0 violation)` |
| Imports croisés | `node tools/lint-cross-feature-imports.mjs` | `Info : 7 <= plafond P8.5 (7). Pas d'échec` |
| Build de production | `npm --prefix apps/web run build` | `✓ built in 1.94s` |
| Rasterisation (témoin) | `cd apps/web && npx playwright test e2e/replay-explosion-raster.spec.ts e2e/replay-muzzle-raster.spec.ts --reporter=line` | `3 passed` |

`lint:colors` n'existe pas encore comme script npm : c'est l'objet de D.12, non commencé. Le
lint canonique est joué directement, avec sa commande, ci-dessus.

**Le témoin de rasterisation est vert APRÈS CHAQUE ITEM** (K3, K1, K2, K4+K5, M4+M5). Il a
fallu l'étendre une fois : K3 change les imports de `grenadeRestLayer`, donc le harnais reçoit
désormais `replayView` dans sa portée — deux lignes de plus au correctif local, à réconcilier
avec le lot F.

### Décisions prises (second temps)

15. **`projectTo(view, p)` plutôt qu'un projecteur pré-lié (D.9, K3).** Une seule forme, prise
    à 36 sites, sans allocation de closure dans une boucle de dessin. Les cinq calques qui
    projettent en boucle gardent un raccourci local `const px = (p) => projectTo(view, p)` :
    c'est de la brièveté au point d'appel, pas une seconde règle de projection.
16. **Les gardes de K2 et K5 visent la FORMULE, pas ses ingrédients (D.9).** La première
    version interdisait `find((r) => r.is_me)` et `new Map<string, number>()` : elle attrapait
    quatre usages parfaitement légitimes (lire l'issue du match sur la ligne « moi », indexer
    l'indice de film d'un siège). Un garde qui crie faux se désactive ; les motifs ont été
    resserrés sur la formule entière.
17. **`covers` est fermé aux DEUX bouts, et le module le dit (D.9, K5).** `t1` est la dernière
    image où l'état vaut, pas la première où il ne vaut plus : un portage d'une seule image a
    `t0 === t1` et doit se peindre. C'est la convention du film, et l'écrire est ce qui rend
    la treizième copie impossible à écrire faux.
18. **Un garde dérivé vérifie AUSSI qu'il dérive quelque chose (D.10).** Les deux gardes
    refondus portent un cas qui échoue si la dérivation rend une liste vide — sans quoi une
    union renommée ou une convention abandonnée les rendrait verts et inertes, ce qui est
    exactement le défaut qu'ils corrigent.

### Découvertes (second temps)

7. **Le prédicat d'intervalle était écrit DOUZE fois, pas dix** : `zoneSound.ts` en portait
   deux que le registre n'avait pas comptées (K5).
8. **`MatchPadControlSection.tsx` lit le camp du joueur SANS le parser** (`board.find((r) =>
   r.is_me)?.team_side ?? null`, un `team_side` brut) : une troisième lecture du tableau de
   score, dans un composant de la Match View hébergé sous `match-replay/`. Hors du périmètre
   de K2 (qui vise le camp NUMÉRIQUE), non traitée — elle relève du même nettoyage que N4/D.13.
9. **`heatmapLayer.ts` porte déjà une fonction locale `scaleOf`** qui prend une grandeur du
   monde, pas le cadrage : l'import de la canonique s'y qualifie (`scaleOf as viewScale`)
   plutôt que de renommer une fonction dont le nom est juste chez elle.

## Reste à faire

- **D.10 (les faux hooks)** : bloqué sur la question ci-dessus — le NOM ou la STRUCTURE.
- **D.11** (max-lines 500 `skipComments`, retrait du cliquet de lignes brutes, arborescence par
  responsabilité, `README.md`), **D.12** (`lint:colors` en CI, 9 copies partielles supprimées),
  **D.13** (les modules que l'allowlist croisée dément descendent dans `lib/replay/`),
  **D.14** (merge de `feat/v2-capabilities`, route gatée par la capability `replay`) : non
  commencés. **Prochain item : D.11.**
- **D.15** (promotion de `ReplayTransport` et `ReplaySettingsDrawer` en frères du canvas) :
  différé hors chantier par le superviseur, après un gate visuel validé par l'utilisateur.

