# Vérification adverse V-WEB-1

HEAD vérifié : `081871f09` (branche `feat/v75`). Lecture seule, aucune modification du dépôt.

## Constat 1 — Quatre politiques pour « l'artefact n'a pas d'origine » : TIENT (correction : 3 politiques distinctes, pas 4)

- **Ce que j'ai vérifié :**
  - `apps/web/src/features/match-replay/replayWindow.ts:110-112` → `const originMs = doc.originMs` ; `if (originMs == null || playableSeconds == null) return null`.
  - `killFeedLogic.ts:235-239` → `const origin = doc.originMs` ; `typeof origin === 'number' && Number.isFinite(origin) ? alignFeedByOrigin(...) : alignFeedToTracks(...)`.
  - `replayMediaLogic.ts:56` → `const origin = Number.isFinite(originMs) ? (originMs as number) : 0`.
  - `presenceFeed.ts:67-69` → `if (!playWindow) return []` PUIS `const origin = Number.isFinite(doc.originMs) ? ... : 0`.
  - `seatLogic.ts:179` (dans `frameOfAbs`) → `... : 0`.
  - Atteignabilité : `grep -rn "presenceEntries(" apps/web/src | grep -v .test.` → **un seul appelant de production**, `routes/…/replay.tsx:162`, qui passe `playWindow` = `replayWindow(data, matchView?.header)` (route:146-149). Donc le `: 0` de `presenceFeed.ts:69` est **inatteignable** dans le cas « origine absente ». `seatLogic.frameOfAbs` est appelé par `buildSeats` (l. 161) → `ReplayTeams.tsx:39` : **atteignable**, aucun garde `playWindow`.
  - Même frise : route:312 `feedEntries={feedEntries}` et route:313 `media={replayMedia}` → `ReplayCanvas.tsx:569` `useReplayTimeline({doc, playWindow, feedEntries, media, ...})` → `useReplayTimeline.ts:119` `const scale = useMemo(() => trackScale(playWindow, frameCount), ...)`, puis `tracks` (l. 127, kills/morts) et `media` (l. 141) **sur le même `scale`**. `replayTimelineTracksLogic.ts:54-58` : `trackScale(null, frameCount)` rend `{from: 0, span: frameCount-1}` — le film entier.

- **Ce qui confirme (le contre-argument attendu tombe) :**
  1. **Les artefacts sans origine existent réellement, et pas seulement en legacy.**
     `grep -L "originMs" data/cache/replays/halo_infinite/*.json | wc -l` → **5** sur **106** fichiers.
     `grep -l '"originResolved":false' … | wc -l` → **5** ; `…true` → **100**.
     Deux des cinq sont en **schéma 34** (`4f77afc1`, `fb1a1a72`) — donc bien postérieurs au schéma 4 : le producteur a *refusé* de dater ces films, il ne s'agit pas d'artefacts périmés. Les trois autres sont en schéma 20/21/34. **~5 % du parc local.** La conséquence n'est pas théorique.
  2. **`0` n'est pas « une valeur correcte pour une origine nulle », c'est un repli — et c'est le producteur Go qui l'écrit.**
     `apps/go-api/internal/analysis/replay/origin.go:126-137`, en-tête d'`originMSOf` : « **ZERO N'EST PAS UNE ORIGINE NEUTRE, C'EST UN REPLI**, et il se journalise. […] sans elle, [les calques] restent decales de l'ecart reel entre le premier paquet du film et le premier paquet de position — 3,6 s a 50,8 s selon le match. » `resolveOriginMs` (l. 108-124) rend `nil` sur deux conditions mesurées (chunk illisible ; origine lue contredite par le fil des morts au-delà de la tolérance).
  3. **Le drapeau qui dirait « ne pose rien » n'est pas lu par ces sites.**
     `grep -rn "originResolved" apps/web/src | grep -v .test.` → **deux seuls sites** : `objectivesLayer.ts:285` (commentaire) et `lib/replay/scoreTimeline.ts:206`. Ni `replayMediaLogic`, ni `presenceFeed`, ni `seatLogic` ne le consultent.

- **Correction à apporter au constat :** cinq sites, mais **trois réponses distinctes** (`null` / appariement mesuré / `0`), dont **quatre atteignables**. Le titre « quatre politiques » sur-compte d'une unité : `replayMediaLogic`, `presenceFeed` et `seatLogic` donnent la *même* réponse (`0`).

- **Conséquence réelle reformulée :** sur les ~5 % d'artefacts que le producteur refuse de dater, la piste Médias et la piste Kills sont posées sur la même échelle avec deux décalages différents (0 contre l'écart mesuré par appariement), ce qui peut éloigner une capture du frag qu'elle montre de l'ordre de l'origine réelle — 3,6 s à ~40 s selon le match ; le seul cas où les deux se recollent est celui où aucune victime n'est nommée et où l'appariement retombe lui aussi sur l'horloge brute.

## Constat 2 — Le roster reconstruit 6 fois, dont 3 par rendu : RÉFUTÉ

- **Ce que j'ai vérifié :**
  `grep -rn "buildPlayers(" apps/web/src --include=*.ts --include=*.tsx | grep -v "\.test\."` → 7 lignes : la **définition** `rosterLogic.ts:90` et **6 appels** (route:162, `ReplayTeams.tsx:96`, `useSlotIdentity.ts:158`, `equipmentUsageLogic.ts:257`, `padControlLogic.ts:150`, `match-view/equipmentKillBadges.ts:55`). Le décompte est exact.

- **Ce que l'auditeur n'a pas vu :**
  1. **Il n'y a pas 6 copies de la jointure, il y en a UNE.** `buildPlayers` est défini une seule fois (`rosterLogic.ts:90-129`) et les 6 sites l'*appellent*. R6 interdit la 3ᵉ **copie d'un pattern** et exige « centraliser dans un helper » : c'est exactement l'état actuel. Appeler un helper canonique depuis 6 endroits n'est pas la dette que R6 vise — c'est son remède.
  2. **La conséquence annoncée n'est pas réalisable.** « toute évolution de la jointure […] doit être portée à 6 endroits, et un oubli produit une fiche et une ligne de fil qui ne parlent plus du même joueur » : une évolution du corps de `buildPlayers` se propage aux 6 sites sans qu'on les touche ; une évolution de sa *signature* est une erreur de compilation TypeScript aux 6 sites, pas une divergence silencieuse. L'auditeur le concède d'ailleurs lui-même (« fonction pure, mêmes entrées »).
  3. **« 3 par rendu de la page » est faux.** Les trois sites montés sur cette page sont tous mémoïsés sur des entrées stables :
     - route:157-164 : `useMemo(..., [kills, medalEvents, t0Ms, data, scoreboard, playWindow, matchView?.header])` — `scoreboard` est lui-même `useMemo(() => matchView?.team_tab.scoreboard ?? [], [matchView])` (route:99, commenté « `?? []` fabrique un tableau neuf à chaque rendu »), `kills`/`medalEvents` sont mémoïsés (route:101-104, route:127-130) ;
     - `ReplayTeams.tsx:96` : `useMemo(() => buildPlayers(doc, scoreboard), [doc, scoreboard])` ;
     - `useSlotIdentity.ts:158` : `useMemo(() => buildPlayers(doc, scoreboard ?? []), [doc, scoreboard])`.
     La route se re-rend ~6,7 fois par seconde pendant la lecture (publication de `frame` toutes les 150 ms) : **aucun des trois memos ne se recalcule**. Le compte réel est de 3 exécutions **par chargement de données**, pas par rendu.
  4. **La fonction est pure et bon marché.** Elle ne mute pas `doc` (elle construit des objets joueurs neufs, `p.lives` est un tableau neuf trié sur place), et son coût est linéaire en (roster + pistes + scoreboard).

- **Conséquence réelle reformulée :** trois exécutions d'une fonction pure et bon marché au chargement de la page au lieu d'une — un coût négligeable et aucun risque de divergence, la jointure n'ayant qu'une seule implémentation.

## Constat 3 — `alignFeed` refait deux fois en aval de la route : TIENT (gravité → P2)

- **Ce que j'ai vérifié :**
  `grep -rn "alignFeed(" apps/web/src | grep -v "\.test\."` → définition `killFeedLogic.ts:230` et **3 appels** : `killFeedLogic.ts:482` (dans `buildFeedEntries`), `killFx.ts:119`, `replaySound.ts:630`. Le commentaire route:150-154 dit bien « LE FIL ALIGNÉ, ASSEMBLÉ ICI ET NULLE PART AILLEURS […] deux recalages menés séparément peuvent diverger ». La « doc inversée » est donc littéralement établie.

- **Ce que l'auditeur n'a pas vu — les trois sites reçoivent des entrées IDENTIQUES :**
  - route:295-296 : `kills={kills}` et `t0Ms={t0Ms}` — ce sont les **mêmes** `kills`/`t0Ms` que ceux passés à `buildFeedEntries(kills, medalEvents, t0Ms, data)` route:158 ; `doc` est le même `data`.
  - `ReplayCanvas.tsx:180` `useReplaySound(doc, kills, t0Ms, ...)` et `:245` `useReplayFx(doc, kills, t0Ms, ...)` → `useReplayFx.ts:44` `buildKillFx(doc, kills ?? [], t0Ms ?? 0)` et `useReplaySound.ts:323` `buildSoundTimeline(doc, kills ?? [], t0Ms ?? 0, ...)`.
  - `alignFeed` est **déterministe et pure** : `lifeEndsOf` (l. 360-373) reconstruit des `LifeEnd` neufs (`used: false`) à chaque appel ; `alignFeedByOrigin` (l. 261-295) fait `.map((k) => ({...k, ...}))` sans muter l'entrée ; `attachDeathKinds` (l. 326-352) n'écrit que sur les `ReplayDeath` fraîchement fabriqués et lit `doc.neutralDeaths` sans le toucher.
  → **La divergence que le commentaire de la route redoute est impossible avec le câblage actuel.** Mêmes entrées, fonction pure ⇒ mêmes sorties.
- **Ce qui reste, et une nuance dans les deux sens :** le coût est un travail redondant. Il est en réalité de **4 exécutions** et non 3 — `useReplaySound.ts:330` appelle `hasSoundEvents`, qui (l. 234-235) relance `buildSoundTimeline` en entier. Mais il est **borné au chargement** : les trois sites sont sous `useMemo` à dépendances stables, donc jamais « triple […] soixante fois par seconde ». Et « un document de plusieurs milliers de kills » est une extrapolation : les kills viennent des `highlight_events` d'un match Halo, pas du film.

- **Conséquence réelle reformulée :** un recalage identique exécuté quatre fois au lieu d'une par chargement (coût CPU ponctuel) et un commentaire de route qui affirme le contraire de ce que fait le code — un piège de maintenance réel si l'on donne un jour au canvas des `kills` différents, mais aucune divergence possible aujourd'hui : la gravité relève du P2, pas du P1.

## Constat 4 — `replaySchemaLogic.ts` : 32 L, 28 commits, zéro lecteur : TIENT (mais un pilier de l'argumentaire est FAUX)

- **Ce que j'ai vérifié :**
  - `cat -n apps/web/src/features/match-replay/replaySchemaLogic.ts` → 32 lignes, un seul export : `export const EXPECTED_REPLAY_SCHEMA_VERSION = 39`.
  - `grep -rn "EXPECTED_REPLAY_SCHEMA_VERSION\|replaySchemaLogic" apps/web/src` → hors le fichier lui-même : `replaySchemaLogic.guard.test.ts` (l. 7, 22 `import { EXPECTED_REPLAY_SCHEMA_VERSION }`, 38, 41, 42) et une **mention en commentaire** dans `deltaLayersContract.guard.test.ts:22`. **Aucun `import`, `import type` ou lecture hors test.**
  - `git log --oneline v7.3.0..HEAD -- …/replaySchemaLogic.ts | wc -l` → **28** ; en tout temps → **28** également.

- **Ce qui est FAUX dans l'argumentaire :** « À défaut, R11 exige une date de retrait — il n'y en a pas. » Une décision datée existe, dans le registre que l'audit cite pourtant lui-même pour d'autres items (l. 10, 122, 184) : `.ai/V7.5/REGISTRE_REPORTS.md` **ligne 449**, entrée « `EXPECTED_REPLAY_SCHEMA_VERSION` (web) devenue tautologique… », datée *lot ui-rejeu 2e passe, 2026-08-26*, avec condition de reprise explicite : « **A LA CLOTURE DU LOT D** […] si toujours aucun lecteur a l'execution, supprimer la constante ET son garde de parite (code mort a tests verts) ». L'en-tête du fichier (l. 20-24) documente les deux issues.

- **Ce qui sauve le constat :** le lot D **est clos** — `.ai/V7.5/PLAN_OBJECTIFS_ETAT_VIVANT_2026-08.md` § « D8 — CLOTURE DU LOT » : D8.1 et D8.2 `[x]`, D8.3 `[!]`, tous les items statués. La condition de reprise a donc été franchie sans être exécutée, et l'entrée de registre n'a pas été amendée.

- **Conséquence réelle reformulée :** le code mort est confirmé (32 L, 28 commits, aucun lecteur à l'exécution), mais le grief exact n'est pas « aucune date de retrait » — c'est « une échéance datée et enregistrée le 2026-08-26 est arrivée à terme (clôture du lot D) sans que la suppression prévue soit faite ».

## Constat 5 — `lib/replay/` : factorisation abandonnée : RÉFUTÉ

- **Ce que j'ai vérifié :**
  `ls apps/web/src/lib/replay/` → `scoreTimeline.ts` (21 338 o) + `scoreTimeline.test.ts` — le fait matériel est exact.
  Imports croisés **hors tests** : match-replay → match-view = **22** (`xuidMeta` 11, `_momentum` 6, `teamColor` 3, `teamSeriesColor` 2) ; match-view → match-replay = **9** (`queries` 3, `rosterLogic` 1, `replayNormalize` 1, `i18n` 1, `equipmentUsageLogic` 1, `MatchPadControlSection` 1, `MatchEquipmentUsageSection` 1). Les chiffres 28/10 de l'audit **incluent les fichiers de test**.

- **Ce que l'auditeur n'a pas vu :**
  1. **La prémisse est fausse : `lib/replay/` n'a pas été créé « pour accueillir le modèle partagé ».** Message de `bb6eb7694` (2026-08-18) : « la logique du score passe dans lib/replay (ratchet cross-feature P8.5) » ; il énumère ce qui a migré — `normalizeScoreTimeline` et les types `*Ready`, `scoreAtFrame`, `teamSeriesFor`/`teamScoreAtFrame`, `playerCountersAt`, `roundAtFrame`, `leadChanges`, `allyOfTeamId`, `teamIdOfSide`, `filmClockTrusted`/`scoreTimelineOf`. Le dossier a reçu **toute** la charge pour laquelle il a été créé : « 1 module sur 5 » compare l'état réel à un objectif que personne n'a jamais fixé.
  2. **Une décision documentée explique précisément pourquoi les autres modules restent dans les features** — dans le fichier même que la consigne désignait, `tools/lint-cross-feature-imports.mjs` l. 82-99 :
     - `'match-replay=>match-view'` : « Le rejeu 2D […] réutilise la COLLECTE des kills (`_momentum.collectKillEvents`), la cascade de couleur d'équipe (`teamColor`) et l'index des joueurs (`xuidMeta`) plutôt que d'en écrire une seconde version. **Dépendance durable et voulue** — la dupliquer donnerait deux définitions de "ce qui est un kill" et deux couleurs pour la même équipe. »
     - `'match-view=>match-replay'` : « Le hook ne peut pas descendre dans lib/ sans y emmener `replayNormalize` (la frontiere de nullabilite du document, 46 importeurs dans match-replay) : **ce serait deplacer la feature, pas partager de la logique**. TOUTE la logique PURE du score, elle, a bien ete sortie dans lib/replay/scoreTimeline.ts (2026-08-18, ratchet P8.5) — cette exception ne couvre QUE le chargement de l'artefact. »
     Le même argumentaire figure in extenso dans le corps du commit `bb6eb7694` (« RESTE UNE EXCEPTION, ET UNE SEULE : `useMatchReplay` »).
  3. **Le garde-rail exigé par l'anti-pattern n°8 EXISTE.** `tools/lint-cross-feature-imports.mjs:348-353` : `const RATCHET_THRESHOLD = 7` ; au-delà, `ERREUR : … > plafond P8.5 … push bloqué`. L'anti-pattern n°8 vise « créer le helper canonique sans migrer les copies **ni poser le garde-rail** » — les deux conditions manquent ici : la logique visée a bien été migrée, et le cliquet est en place.

- **Conséquence réelle reformulée :** il n'y a pas de factorisation abandonnée — `lib/replay/` a reçu l'intégralité de ce pour quoi il a été créé, le reste des dépendances croisées est une décision argumentée, datée et tenue par un cliquet qui bloque toute régression ; ce que le constat décrit comme une dette est un arbitrage documenté.

## Constat 6 — `roundAtFrame` mort, gardé par 6 assertions : TIENT

- **Ce que j'ai vérifié :**
  - `grep -rn "roundAtFrame" apps/web/src --include=*.ts --include=*.tsx | grep -v currentRoundAtFrame` → **définition** `lib/replay/scoreTimeline.ts:311` (doc) et `:315` (`export function roundAtFrame`), puis **uniquement** `lib/replay/scoreTimeline.test.ts` : import l. 20, `describe` l. 111, et **6 assertions** l. 134, 138, 142, 143, 144, 150. **Zéro appelant de production**, y compris dans `match-view`, `lib/` et `routes/`.
  - `grep -rn "RoundReading" apps/web/src` → `scoreTimeline.ts:299` (interface) et `:315` (son unique usage, le type de retour de `roundAtFrame`). Type mort avec sa fonction.
  - `git show 2fbeb5fcb` (2026-08-28, « replay2d(bandeau): score par MANCHE au lieu du cumul du match ») : le diff supprime `- roundAtFrame,` de l'import et `- const reading = roundAtFrame(best, frame)`, et ajoute 15 lignes mentionnant `currentRoundAtFrame`. `git log -S"currentRoundAtFrame" -- …/roundsLogic.ts` ne rend **que ce commit**. Le remplaçant naît donc bien dans le commit qui tue le remplacé.
  - Aucune décision datée ne le garde : `grep -i "roundAtFrame\|RoundReading\|manche courante" .ai/V7.5/REGISTRE_REPORTS.md` → **aucun résultat** ; l'en-tête de `roundAtFrame` (l. 311-314) ne dit rien d'un maintien volontaire (contrairement à `replaySchemaLogic.ts`, qui, lui, l'écrit).

- **Conséquence réelle reformulée :** une seconde lecture de « quelle manche est en cours » — celle dont `currentRoundAtFrame` documente qu'elle est fausse — survit sans appelant ni décision datée dans `lib/`, l'emplacement le plus canonique du dépôt, maintenue verte par six assertions.

## Bilan : 4 tiennent, 2 réfutés, 1 requalifié

- **Tiennent :** 1 (avec correction : 3 politiques distinctes, pas 4 — le `0` de `presenceFeed.ts:69` est inatteignable), 3 (requalifié P1 → P2), 4 (avec un pilier faux : une décision datée existe, ligne 449 du registre, mais sa condition a été franchie sans exécution), 6.
- **Réfutés :** 2 (la jointure a une implémentation unique ; les trois sites de la page sont mémoïsés sur des entrées stables — coût au chargement, pas divergence, pas « par rendu »), 5 (`lib/replay/` a reçu toute la logique de score pour laquelle il fut créé ; la frontière restante est une décision documentée dans `lint-cross-feature-imports.mjs` l. 82-99 et tenue par un cliquet `RATCHET_THRESHOLD = 7`).
- **Requalifié :** 3 → P2.
