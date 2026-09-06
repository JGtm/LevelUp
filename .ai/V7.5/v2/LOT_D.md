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


---

## Tâche D-II, troisième temps (D.11 à D.14) — CLOSE le 2026-09-06

### Items

- [x] **D.11 (décision utilisateur 4) — le seuil de taille se mesure en lignes de code, et
  l'arborescence suit les responsabilités.** Neuf commits (`929fa368d`, `37f16c909`,
  `db78b6013`, `9773fda3d`, `72388f2a2`, `eed3a98cd`, `59776f181`, `5178a35bb`, `62e107729`).
  - **La règle.** `max-lines` ESLint à 500 avec `skipComments: true` et `skipBlankLines: true`,
    en `error`, sur `src/**/*.{ts,tsx}` (`apps/web/eslint.config.js`). R5 avait un énoncé et
    aucun outil côté web.
  - **Le cliquet part.** Les 160 lignes de `placementFamily.guard.test.ts` qui portaient le
    plafond de lignes BRUTES de `ReplayCanvas.tsx` (<= 665) et son journal de 17 extractions
    sont supprimées ; le garde des FAMILLES de pose, seul objet déclaré du fichier, reste
    entier. Les **26 en-têtes** qui citaient ce cliquet comme raison d'exister nomment
    désormais `max-lines` (une référence à un test supprimé est une doc morte).
  - **L'arborescence.** Huit dossiers, un commit chacun : `i18n/` (3 sources), `sound/` (25),
    `export/` (10), `settings/` (8), `layers/` (62), `ui/` (31), `hooks/` (11), `model/` (58),
    plus `test/` (doubles partagés). Déplacements PURS par `git mv` ; la racine ne garde que
    ce que la Match View consomme, traité en D.13.
  - **La règle de classement est écrite** (`features/match-replay/README.md`, huit critères
    dans un ordre, le premier qui s'applique gagne) **et gardée**
    (`test/arborescence.guard.test.ts` : un module dont une fonction reçoit un
    `CanvasRenderingContext2D` vit dans `layers/`, trois exceptions nommées).
- [x] **D.12 (M6) — le lint couleur canonique entre en CI, ses copies partielles sortent.**
  `commit fde1dbf34`. Script `lint:colors` (`apps/web/package.json`) + un step au job
  `frontend` (`.github/workflows/ci.yml`, une seule ligne ajoutée au fichier). Cinq blocs de
  garde qui relisaient la SOURCE partent (`fxInk`, `heatmapLayer`, `weaponPads`,
  `flagCarries`, `placementDropped`).
- [x] **D.13 (N4) — la frontière rejeu / Match View devient vraie, module par module.**
  `commit 7aa7ebf4f`. Sept modules descendent dans `lib/replay/` (document, normalisation,
  logique de lecture, roster, deux dépendances pures, chargement) ; le lint accepte une
  exception au niveau du MODULE ; les deux paires du rejeu deviennent huit modules nommés.
  Imports croisés hors tests : 9 avant, 4 après.
- [x] **D.14 — merge du lot C, puis la page de rejeu passe la porte de titre `replay`.**
  `merge b093fe6fc` + `commit d21998b85`. Un seul conflit (`.ai/thought_log.md`), résolu en
  gardant les deux entrées. Aucun conflit sur le code : le lot C n'a touché aucun fichier du
  rejeu. Quatre cas de test, dont un qui lit la route.

### Gate final (D.11-D.14)

| Gate | Commande exacte | Dernière ligne |
|---|---|---|
| Typecheck (cache forcé) | `cd apps/web && npx tsc -b --force` | (aucune sortie) `TSC_EXIT=0` |
| Lint (dont `max-lines` en error) | `npm --prefix apps/web run lint` | `✖ 27 problems (0 errors, 27 warnings)` |
| Couleurs (script neuf) | `npm --prefix apps/web run lint:colors` | `lint-no-hardcoded-colors: clean (0 violation)` |
| Imports croisés | `node tools/lint-cross-feature-imports.mjs` | `Info : 7 <= plafond P8.5 (7). Pas d'échec` |
| Vitest (suite web entière) | `cd apps/web && node_modules/.bin/vitest run --pool=forks` | `Tests  6376 passed \| 14 skipped (6390)` |
| Build de production | `npm --prefix apps/web run build` | `✓ built in 1.84s` (2 207 modules) |
| Rasterisation (témoin) | `cd apps/web && npx playwright test e2e/replay-explosion-raster.spec.ts e2e/replay-muzzle-raster.spec.ts --reporter=line` | `3 passed` |

Les 27 avertissements de lint sont ceux de la baseline (28 au commit de base). Zéro test
skippé ajouté ; aucun garde assoupli sans justification datée. **La rasterisation a été rejouée
après CHACUN des douze commits** : 3/3 à chaque fois.

**Pouvoir de détection, vérifié par mutation (quatre gardes neufs ou refondus).**

- `carriedGlyphPulse.guard.test.ts` (migré sur le balayage récursif) : la sinusoïde plantée
  dans `sound/zoneSound.ts` le fait rougir — `expected [ 'sound/zoneSound.ts' ] to deeply
  equal []`. Le balayage plat ne l'aurait plus vue.
- `test/arborescence.guard.test.ts` : une fonction de peinture ajoutée à `model/killFx.ts` le
  fait rougir en nommant le fautif et le README.
- `lib/replay/crossFeatureBoundary.guard.test.ts` : réintroduire la paire nue
  `match-view=>match-replay` dans l'allowlist le fait rougir.
- `replay.gate.test.tsx` (quatrième cas) : retirer la porte `replay` de la route le fait rougir.

**Preuve M6, mesurée et non supposée.** Un hex planté tour à tour dans chacun des **20 fichiers
cibles** des cinq gardes supprimés fait rougir le lint canonique (20/20) ; une classe Tailwind
aussi (3/3 éprouvés). Le gate global couvre donc strictement plus que ce qu'il remplace.

### Décisions prises (troisième temps)

19. **Les exemptions `max-lines` sont datées et nommées, jamais silencieuses (D.11).** 22
    fichiers portent `/* eslint-disable max-lines -- 2026-09-06 (lot v2 D.11, décision
    utilisateur 4) : <raison>. */` en tête, en trois familles :
    - **table de données** (une entrée par clé, aucun embranchement — la découper répartirait
      la même table sur plusieurs fichiers à tenir en phase) : `lib/api/types.ts`,
      `features/{squad,match-view,help,settings,ascension}/i18n.ts`,
      `features/match-replay/i18n/i18n.ts` ;
    - **suite de tests** d'un même module, qui partagent leur montage :
      `match-replay/ui/ReplayTeams.test.tsx`, `match-replay/sound/replaySound.test.ts`,
      `match-view/MatchHeader.test.tsx`, `home/HomePage.test.tsx`,
      `media/CoverFlowModal.test.tsx` ;
    - **hors périmètre du lot D** — l'exemption DATE la dette, elle ne l'absout pas :
      `synthesis/SynthesisPage.tsx`, `timeseries/TimeseriesFormCharts.tsx`,
      `media/CoverFlowModal.tsx`, `squad/SquadLayout.tsx`,
      `palmares/PalmaresRelationsPage.tsx`, `palmares/SeasonPassPage.tsx`,
      `match-view/MatchScoreboard.tsx`, `home/HomePage.tsx`, `components/ui/match-card.tsx`,
      `components/shell/PeriodSessionRail.tsx`.

    Quatre fichiers sont exemptés PAR LA CONFIG, faute de pouvoir porter un commentaire
    durable : les trois générés (`lib/api/generated.ts`, `lib/i18n/generated/common.ts`,
    `lib/i18n/generated/admin.ts` — un `eslint-disable` y serait effacé à la prochaine
    génération) et `features/explorer/ExplorerMatchesTable.tsx`, gelé par la consigne pour le
    merge du lot C. Aucun fichier n'a été découpé : le seul du rejeu qui dépassait est une
    table de traduction.
20. **La règle de classement de l'arborescence est un ORDRE, pas une liste (D.11).** i18n ->
    sound -> export -> settings -> layers -> ui -> hooks -> model : le premier critère qui
    s'applique gagne, ce qui donne à chaque fichier une place et une seule. Les quatre domaines
    transverses l'emportent sur la forme du fichier — `ReplaySoundControls.tsx` est un
    composant React et vit dans `sound/`, parce qu'on le cherche avec les sons.
21. **« Fx » ne veut pas dire calque (D.11).** `shotFx`, `killFx`, `grenadeFx`, `playerCardFx`
    CALCULENT ce qu'un effet montre et ne touchent jamais la toile : ils vivent dans `model/`.
    Le critère retenu — recevoir un `CanvasRenderingContext2D` — est le seul des huit qui se
    lise dans le code, donc le seul qu'un test puisse tenir.
22. **Onze gardes balayaient `readdirSync(__dirname)` : l'arborescence les aurait rendus VERTS
    ET INERTES (D.11).** `test/featureFiles.ts` rend la liste RÉCURSIVE de la feature (le patron
    `walk` était déjà recopié deux fois — la troisième copie était interdite, R6), porte les
    ancres `racineWeb()` / `racineDuDepot()` pour les neuf gardes qui lisent un fichier Go, un
    TOML ou la feuille de style, et `fichierNomme()` pour les six qui lisaient une liste de
    fichiers par chemin. Aucun garde ne compte plus ses `..`.
23. **Le témoin de rasterisation cherche ses modules par NOM (D.11, D.13).**
    `e2e/_helpers/replaySource.ts` balaie `features/match-replay/` ET `lib/replay/` : les specs
    lisaient des chemins écrits en dur et tombaient sur un ENOENT à chaque déplacement — le
    seul gate qui dise « aucun pixel n'a bougé » ne pouvait pas dépendre d'une arborescence
    qu'on refond. Ce correctif REMPLACE celui du premier temps (comptage d'imports de valeur),
    à réconcilier avec le lot F (F.6) au merge.
24. **Les quatre assertions de couleur sur le RENDU restent (D.12), et c'est un écart assumé au
    périmètre littéral.** Le registre comptait 9 copies ; cinq relisaient la SOURCE (supprimées,
    le lint canonique couvre leurs 20 fichiers cibles, mesuré), quatre vérifient le HTML PRODUIT
    (`ReplayKillFeed`, `ReplayScoreBanner`, `ReplayTimelineTracks`, `playerCardFx`). Un lint
    statique ne voit jamais une couleur qui arrive par la DONNÉE — `team_color` vient du
    document. Les supprimer aurait retiré un pouvoir de détection que le gate global ne rend
    pas : la preuve demandée (« le lint canonique couvre leurs fichiers cibles ») ne peut pas
    être faite pour celles-là.
25. **Sept modules descendent dans `lib/replay/`, et pas huit (D.13).** La fermeture a été
    vérifiée AVANT de bouger : `replayReadyTypes`, `replayNormalize`, `replayLogic`,
    `changeRefine`, `playerMarks`, `rosterLogic`, `queries` n'importent rien de `features/`.
    `equipmentUsageLogic` a été laissé : il dépend d'un CALQUE
    (`layers/equipmentPlacementsLayer`), le descendre aurait fait suivre le calque et ses
    quatre modules de forme — déplacer la feature au lieu de partager de la logique. C'est
    exactement l'argument que l'ancienne justification opposait à `queries`, et il ne tenait
    plus pour lui une fois `replayNormalize` descendu.
26. **Le plafond du ratchet P8.5 reste à 7 (D.13).** Les sept violations non déclarées sont
    étrangères au rejeu (`filters`, `palmares`, `explorer`) et les corriger serait un fix hors
    périmètre (règle 7). Ce qui redescend, c'est la PORTÉE de l'exception : de deux features
    entières (370 + 60 fichiers) à huit modules nommés. Le lint affiche désormais le module
    fautif et non plus la feature.
27. **L'état « ce titre ne propose pas de rejeu » n'est PAS recopié dans le dictionnaire du
    rejeu (D.14).** Le lot C porte déjà le libellé `replay` en FR et en EN dans la table de
    `FeatureUnavailable` (`lib/capabilities/`). En écrire un second dans `REPLAY_TEXT` donnerait
    deux textes pour la même phrase — la duplication que ce lot passe son temps à fermer. La
    parité FR/EN exigée est tenue, et vérifiée par le test (deux cas de langue).
28. **La route porte DEUX portes imbriquées (D.14).** `matchmaking` (le match n'existe pas pour
    ce titre) puis `replay` (le titre a des matchs, aucune chaîne de rejeu). Elles ne disent pas
    la même chose et le lecteur doit lire la bonne.

### Découvertes (troisième temps, hors périmètre, non traitées)

10. **`MatchPadControlSection.tsx` lit toujours le camp du joueur sans le parser** (constat 8 du
    second temps). Le fichier est resté à la racine de la feature pour D.13, il n'a pas été
    touché.
11. **Sept violations cross-feature non déclarées subsistent**, toutes étrangères au rejeu :
    `ascension -> filters/filterLink` (2), `explorer -> palmares/RelationWinRateDonut` et
    `/CumulativeFragGapChart`, `match-view -> explorer/explorerMatchesClientSort` (2) et
    `squad -> explorer/explorerMatchesClientSort`. Le lint les nomme maintenant module par
    module, ce qui rend leur traitement mécanique pour le lot qui touchera ces fichiers.
12. **Les autres paires de l'allowlist restent des paires** (une cinquantaine). Le mécanisme par
    module existe désormais pour toutes ; seules les deux du rejeu ont été converties, le reste
    est hors périmètre.
13. **`lib/` n'est contrôlé par aucun lint de frontière.** Le ratchet P8.5 vérifie
    `features/ -> features/` et `components/ -> features/`, mais pas `lib/ -> features/`. La
    fermeture des sept modules descendus a donc été vérifiée À LA MAIN avant le déplacement.
    Un `lib/ -> features/` interdit par le lint serait le garde-rail manquant.

### Reste à faire

- **D.10 (les faux hooks)** : toujours bloqué sur la question du second temps — le NOM ou la
  STRUCTURE.
- **D.15** (promotion de `ReplayTransport` et `ReplaySettingsDrawer` en frères du canvas) :
  différé hors chantier par le superviseur, après un gate visuel validé par l'utilisateur.

---

## Corrections après revue adversariale R1 — CLOSES le 2026-09-06

Verdict R1 (HEAD relu `bb379d10b`) : l'invariant de calcul tient — 25/25 calques identiques en
ordre ET en condition, jointure des 13 mémos, replis d'horloge, canoniques, magasin de lecture.
Ce qui ne tenait pas : **deux câblages neufs sans témoin** (C1, C2), **cinq affirmations
inexactes** (C3, C5, C6, C7, C8), **un cinquième changement visible hors contrat** (C4), et
**douze commentaires orphelins** (C9). Onze points, tous statués ci-dessous, chacun prouvé par
la mutation du verdict rejouée ROUGE puis VERTE.

### Points statués

- [x] **C1 (P1) — le câblage de l'axe commun a un témoin.** `commit 28d850866`. La correction
  P0-7 tenait à trois lignes (`t0Ms={header.t0_ms}` dans la page, `t0Ms={t0Ms}` sur les deux
  lectures du bloc de score) que la revue a retirées une à une sans un seul rouge.
  `MatchViewTabChronology.t0.test.tsx` (3 cas) et `MatchViewPage.t0.test.tsx` (1 cas) couvrent
  les deux niveaux : le CÂBLAGE (espions sur l'onglet et sur les deux graphes) et l'EFFET (sur
  le graphe RÉEL, la marque de l'image 400 tombe à 22 000 ms — `400 × 100 − 18 000`, calculé à
  la main — et recule du countdown entier quand la prop disparaît).
  **Mutation M4, rouge puis verte** : `expected undefined to be 30000`, aux deux sites.
- [x] **C2 (P1) — les calques se nomment eux-mêmes.** `commit a3cd9d85d`. Les ONZE calques
  câblés par un hook portent leur `id` à côté de leur `paint` (`NamedLayerPainter`), et
  `bindPainters` dérive la liaison : le canvas ne nomme plus aucun d'eux, la faute mesurée par
  la revue n'est plus ÉCRIVABLE. Le type de retour porte l'union exacte des ids passés — un
  calque oublié fait rougir le compilateur. Les QUATORZE fermetures restantes sont confrontées
  à une table écrite à la main (`sceneBinding.guard.test.ts`, 5 cas), découpée depuis `paint: {`
  et non depuis le fichier — le bloc `has:` juste au-dessus porte les mêmes clés pour des
  booléens.
  **Mutations rouges puis vertes** : le swap couronne/crâne de la revue (M3,
  `expected 339 to be -1`) et l'interversion de deux cuissons (`l'entrée chaleur ne peint pas
  cuit(heatRef.current)`).
- [x] **C3 (P2) — l'ordre des calques est confronté à un oracle.** `commit 9169fcb94`. Les 25
  ids sont écrits à la main dans le test ; la table de la source leur est confrontée.
  **Mutation M2, rouge puis verte** : le swap `chaleur` / `zones-nommees`.
- [x] **C4 (P2, décision du superviseur) — cinquième exception documentée, code inchangé.**
  `commit f3aa5aa05`. L'infobulle native des marques de la frise affiche l'instant TRONQUÉ
  (« 1:05 » là où la base écrivait « 1:06 »), sur environ une marque sur deux. C'est la
  résolution du résidu P0-6 : la même mort se datait autrement dans l'infobulle et dans le fil,
  sur le même écran. Consignée en tête de `formatClock` (`lib/replay/replayLogic.ts`) et dans
  la liste des exceptions ci-dessous.
- [x] **C5 (P2) — le lint couleur voit `oklch`/`rgba`, et balaie `lib/replay/`.**
  `commit ec5a4f391`. Troisième motif (oklch, oklab, lch, rgb/rgba, hsl/hsla, hwb) ;
  `src/lib/replay/**` ajouté au balayage ; ouverture de bloc de commentaire (`/*`, `/**`)
  tolérée comme l'étaient déjà `//` et `*`. `color(` est VOLONTAIREMENT absent du motif : c'est
  aussi un nom de variable de rendu du dépôt (`valueGridModel.ts`), et un motif qui crie faux
  se désactive. Les onze littéraux préexistants que le motif révèle portent une exception
  nommée et datée ligne à ligne (dix voiles neutres d'ombre ou de contour — exception déjà
  prévue par la doctrine `color-tokens` — et un helper qui convertit en `rgba` une couleur déjà
  résolue). Aucun rendu ne change : ce sont des commentaires.
  **Mutation M10, rouge puis verte** : `oklch(...)` + `rgba(...)` dans `layers/fxInk.ts`
  (2 violations) ; `oklch(...)` dans `lib/replay/rosterLogic.ts` (1 violation).
- [x] **C6 (P2) — le garde d'horloge attrape la lecture qualifiée.** `commit bc16a9b04`. Le
  motif exigeait que `frameIntervalMs` suive immédiatement l'opérateur : il attrapait la forme
  historique et laissait passer `paliers[0].t * clock.frameIntervalMs`, la seule que le nouveau
  code rend naturelle. **Mutation M8, rouge puis verte.**
- [x] **C7 (P2, décision du superviseur) — la mesure exacte remplace l'approximation du
  journal.** `commit f3aa5aa05`. L'arrondi avant conversion change **30 valeurs en SIX fenêtres
  d'une demi-seconde** (`[59,5;60)`, `[119,5;120)`, `[179,5;180)`, `[239,5;240)`, `[299,5;300)`,
  `[359,5;360)`), et non « un intervalle d'une demi-seconde » comme l'écrivait la découverte 2
  du D-I — **cette affirmation du journal était fausse et est corrigée ici**. Cinq fenêtres font
  disparaître une sortie invalide (« 1m60s ») ; la première change de FORME (« 60s » →
  « 1m00s »). Consignée au site de la conversion.
- [x] **C8 (P2) — l'exemption `max-lines` de l'Explorer rentre chez elle.** `commit d943b485e`.
  Sa justification (« gelé pour le merge du lot C ») avait survécu au merge, présent dans la
  branche depuis D.14 le même jour. Elle passe en tête de fichier, datée, avec un critère de
  retrait mesurable (la table de colonnes sort dans un `_columns.ts`). La config ne garde que
  les fichiers GÉNÉRÉS, qui ne peuvent pas porter de commentaire durable.
- [x] **C9 (P2) — douze JSDoc orphelins supprimés.** `commit 1cb46136e`. Chacun décrivait une
  déclaration rendue canonique par D.9 (`CanvasView` local ×11, `alphaOf` ×3, `ScoreboardSide`)
  et se retrouvait collé au-dessus de la déclaration suivante. Aucun n'est remplacé : leur
  contenu vit dans le foyer canonique.
- [x] **Réserve 9 — la citation du contrat Go.** `commit bc16a9b04`. `matchClock.ts` présentait
  `msFilm = event_time_ms + t0_ms − originMs` comme « le contrat Go, mot pour mot ». Le Go
  n'écrit que `msFilm = event_time_ms + t0_ms` : il ramène sur l'axe du MATCH, il ne voit pas le
  film. La formule reste (elle ramène en plus sur l'axe des IMAGES), la citation disparaît.
- [x] **Note hors constat — consignée, non corrigée.** `MatchCadenceChart.tsx` porte deux
  libellés FR en dur (`name: 'Temps'`, `formatter: 'Pic'`), préexistants et inchangés sur le
  fond : découverte 14 ci-dessous.

### Gate de clôture des corrections

| Gate | Commande exacte | Dernière ligne |
|---|---|---|
| Typecheck (cache forcé) | `cd apps/web && npx tsc -b --force` | (aucune sortie) `TSC_EXIT=0` |
| Lint | `npm --prefix apps/web run lint` | `✖ 27 problems (0 errors, 27 warnings)` |
| Couleurs (motif étendu) | `npm --prefix apps/web run lint:colors` | `lint-no-hardcoded-colors: clean (0 violation)` |
| Imports croisés | `node tools/lint-cross-feature-imports.mjs` | `Info : 7 <= plafond P8.5 (7). Pas d'échec` |
| Vitest (suite web entière) | `cd apps/web && node_modules/.bin/vitest run --pool=forks` | `Tests  6385 passed \| 14 skipped (6399)` |
| Build de production | `npm --prefix apps/web run build` | `✓ built in 18.37s` (2 207 modules) |
| Rasterisation (témoin) | `cd apps/web && npx playwright test e2e/replay-explosion-raster.spec.ts e2e/replay-muzzle-raster.spec.ts --reporter=line` | `3 passed` |

La suite est passée de 6 376 à 6 385 tests (+9 : 4 pour C1, 5 pour C2). Les 27 avertissements
de lint sont ceux de la baseline. Rasterisation 3/3 après chacun des huit commits.

### Les SIX exceptions du lot (le contrat en comptait quatre)

1. `SynthesisBipolaireChart` ne rend plus « 1m60s » (sortie invalide corrigée).
2. Les quatre sites `MmSSs` passent par le formateur canonique (sorties identiques, vérifié).
3. Le bloc « Score dans le temps » ne rend rien sans origine publiée (5 artefacts du parc, déjà
   écartés en amont par `filmClockTrusted` — aucune carte ne disparaît).
4. La piste Médias se tait sur un artefact sans origine, au lieu de poser ses captures à zéro.
5. **(C4, nouvelle)** L'infobulle native des marques de la frise TRONQUE l'instant au lieu de
   l'arrondir : une seconde de moins sur environ une marque sur deux.
6. **(C7, nouvelle)** `SynthesisBipolaireChart` écrit « 1m00s » au lieu de « 60s » sur
   `[59,5;60)` — changement de forme, et non correction d'une invalidité, dans une des six
   fenêtres mesurées.

### Découvertes (revue R1, hors périmètre, non traitées)

14. **`MatchCadenceChart.tsx` porte deux libellés FR en dur** (`name: 'Temps'` sur l'axe X,
    `formatter: 'Pic'` sur l'étiquette du markPoint). Préexistants ; le lot les a re-écrits en
    les déplaçant dans `cadenceOption`/`cadenceSeries`, sans changer le fond. Manquement à la
    règle n° 1 (parité FR/EN) à traiter par le lot qui touchera ce graphe.
15. **Un flake de la suite web, non reproductible.** Un run complet sur trois a rendu
    `1 failed | 6384 passed` sans nommer le cas dans la sortie capturée ; les deux runs suivants
    (dont celui du gate ci-dessus) sont verts à `6385 passed`. La machine était chargée (184 s
    contre 115 s). Signalé pour surveillance, aucune cause identifiée.
16. **Onze littéraux de couleur préexistants** portent désormais une exception `color-allow`
    nommée : dix voiles neutres (ombres d'infobulle ECharts, contours de lisibilité) et un
    helper de conversion. Ils deviennent visibles au gate, ce qui est le point ; les porter sur
    un token de voile, quand il existera, reste à faire.
