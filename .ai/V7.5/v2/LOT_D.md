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

## Reste à faire

- Tâche D-II (D.6 à D.14) : non commencée. Elle démarre sur message du superviseur, après la
  revue adversariale de D-I.
