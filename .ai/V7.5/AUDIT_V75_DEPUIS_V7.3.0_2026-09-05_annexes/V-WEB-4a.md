# Vérification adverse V-WEB-4a

## Constat 1 — Deux horloges m:ss dans l'onglet Chronologie : TIENT (P0 maintenu)

- Ce que j'ai vérifié :
  - `apps/go-api/internal/analysis/replay/score_timeline.go:48-70` — `scoreClock` ne connaît que
    trois champs (`intervalMS`, `frames`, `originMS`) et `frameOf` fait `t = (timeMS - c.originMS) / c.intervalMS`.
    Le commentaire du champ (`:54-57`) l'écrit lui-même : « la grille compte depuis le premier paquet
    de POSITION : sans cette soustraction, toute la courbe glisse de 3,6 s à 50,8 s ». **`t0FilmMs`
    n'apparaît nulle part dans ce fichier.**
  - `apps/go-api/internal/analysis/replay/t0_film.go:298` — `t0 := originMs + cov.MarginMs`. Le coup
    d'envoi est PUBLIÉ comme valeur (`doc.T0FilmMs`, `document.go:740`), jamais utilisé pour décaler la
    grille. Le producteur ne cale donc PAS la série de score sur le coup d'envoi mesuré.
  - Côté client : `_scoreCurve.ts:102` (`c.frame * frameIntervalMs`) et `:123` (`p.t * frameIntervalMs`) ;
    même chose pour les barres, `_scoreEvents.ts:118` (`ms: p.t * frameIntervalMs`). Aucun terme
    correctif. `MatchScoreCurveChart.tsx:159-164` pose `min: 0`, `max: curve.durationMs`,
    `formatter: (v) => formatClock(v)` ; ses `Props` (`:52-67`) ne portent NI `t0_ms` NI l'en-tête.
  - Axe A : `MatchKDCumulChart.tsx:104-115` filtre et trie sur `event_time_ms`, et `:304`
    `axisLabel: { …, formatter: (v: number) => formatMmSs(v) }`. Ces `event_time_ms` sont recalés sur le
    gameplay par `apps/go-api/internal/service/match_view_data_loaders.go:549-565`
    (`d.events = timeline.CorrectEventRaws(d.events, tl)`).
  - `grep -rn "t0Ms|t0_ms" apps/web/src --include=*.ts --include=*.tsx | grep -v test` → **zéro** occurrence
    sous `features/match-view/` ; tout est dans `features/match-replay/`.
  - `grep -rn "displayClockMs" apps/web/src | grep -v test` → 6 fichiers consommateurs, **tous** dans
    `features/match-replay/` (`ReplayKillFeed`, `ReplayPlacementTip`, `ReplayScoreBanner`,
    `useReplayClock`, `useReplayExport`, `useReplayTimeline`). Zéro en Match View.
  - `config/titles/halo_infinite/mappings/regulation.toml:273-279` — seuls `"Slayer"` et
    `"Team Snipers"` valent `hidden`. Sur tout mode à objectif, `MatchViewTabChronology.tsx:84-121`
    monte bien les deux blocs l'un sous l'autre.
- Ce qui confirme (et que l'auditeur n'a PAS vu, mais qui va dans son sens) :
  - **Le dépôt possède déjà le recalage et ne l'applique qu'au rejeu.** `replayWindow.ts:118-129` fait
    `startT0Ms = doc.t0FilmMs ?? apiT0Ms` puis `startMs = max(0, startT0Ms − originMs)`, et
    `displayClockMs` (`:138-140`) retranche ce `startMs` pour que « le coup d'envoi se lise 0:00 ». La
    Match View n'appelle ni l'un ni l'autre. Ce n'est donc pas une méconnaissance du problème dans le
    dépôt : c'est la même donnée, le même schéma, et deux traitements.
  - **La fausse prémisse est verrouillée par un test.** `MatchScoreCurveChart.test.tsx:121-127` nomme son
    `it` « borne chaque série **au coup d'envoi** et à la fin du match » et assert
    `expect(opt.series[0].data[0]).toEqual([0, 0])`. Le malentendu de `_scoreCurve.ts:110` est donc
    recopié dans l'oracle : une correction de l'axe ferait rougir ce test-là, ce qui aggrave la dette
    plutôt que de l'atténuer.
  - Magnitude : le SIGNE varie, ce que l'auditeur n'explicite pas. Témoins du dépôt —
    `replayWindow.ts:26` : `e94163af` originMs 39 772 > t0_ms 35 238 → **+4,5 s** ;
    `origin.go:41-45` : `000d5950` originMs 3 604, avec un T0 de 18 à 28 s (`domain/match_view.go:155-160`)
    → **−14 à −24 s**. « jusqu'à ±20 s » est donc soutenu par les mesures du dépôt.
  - Nuance à porter au constat, sans le renverser : le point tracé `[0,0]` n'est pas FAUX (le score vaut
    bien 0 à la frame 0 comme au coup d'envoi) ; ce qui est faux, c'est l'ÉTIQUETTE de l'axe et
    l'affirmation de `_scoreCurve.ts:110` / du test. Et la population concernée est étroite (artefact
    présent ET mode hors Slayer/Team Snipers).
- Conséquence réelle reformulée : sur tout mode à objectif doté d'un artefact, l'abscisse « 0m00s » de la
  courbe/des barres de score désigne le premier paquet de position du film et non le coup d'envoi, alors
  que « 0m00s » des frags cumulés, juste au-dessus, désigne bien le coup d'envoi — écart mesuré de −24 s
  à +4,5 s selon le match, non signalé et figé par un test qui affirme le contraire.

## Constat 2 — `MatchKDCumulChart.buildOption` 260 L : TIENT (P1), une formulation à corriger

- Ce que j'ai vérifié :
  - `MatchKDCumulChart.tsx` : `const buildOption = useCallback(` en L95, fonction fléchée L96, dernier `}`
    du corps L354, `},` L355, tableau de dépendances L356-366, `)` L367. Corps = **L96-355 = 260 lignes**.
    Le compte de l'auditeur est **exact au numéro de ligne près**.
  - `MatchCadenceChart.tsx` : `useCallback` L56, fonction L57, `}` L217, `},` L218, deps L219, `)` L220.
    Corps = **L57-218 = 162 lignes**. Exact également.
  - `grep -n "exemption|eslint-disable|complexity|max-lines|seuil"` sur les deux fichiers → **0 résultat**.
    L'en-tête de `MatchKDCumulChart.tsx:1-18` décrit la fonctionnalité, jamais une dérogation à R5.
- Ce qui doit être corrigé dans l'énoncé :
  - « **sans aucun test qui l'exerce** » est FAUX pour `MatchKDCumulChart`. `ChartCard.tsx:141` calcule
    `buildOption(series)` dans un `useMemo` à chaque rendu non vide, et `MatchCombatCtfOverlay.test.tsx:61-91`
    monte le composant deux fois avec 3 kills et un scoreboard réel : **les 260 lignes s'exécutent**
    (cumuls, stagger anti-collision et sa boucle `while`, résolution de ton, overlay CTF). Un plantage
    ou une boucle non bornée serait attrapé. La phrase du constat 3 « sans exercer une seule de ses 260
    lignes » est, elle, factuellement fausse. Ce qui manque est l'ORACLE, pas l'exécution.
  - Pour `MatchCadenceChart`, en revanche, la version forte tient : le seul test qui le nomme
    (`MatchViewTabs.test.tsx:107`) le **remplace par un stub** (`vi.mock('./MatchCadenceChart', …)`).
    Zéro ligne exécutée, zéro assertion.
- Conséquence réelle reformulée : deux fonctions de 260 L et 162 L au-dessus du seuil R5 sans commentaire
  d'exemption, dont l'une n'est exécutée par aucun test et l'autre exécutée sans qu'aucune assertion ne
  porte sur ce qu'elle calcule.

## Constat 3 — `MatchCombatCtfOverlay.test.tsx` : TIENT (gravité → P2), cause mal attribuée

- Ce que j'ai vérifié : le fichier entier (124 L). Ligne 21 :
  `<div data-testid="echarts-stub">{JSON.stringify(option).slice(0, 80)}</div>`. Quatre `it` (L61, 77, 93, 109),
  chacun avec une seule assertion `expect(screen.getByTestId('echarts-stub')).toBeInTheDocument()`.
  Les faits bruts de l'auditeur sont exacts.
- Ce que l'auditeur n'a pas vu :
  1. **La troncature n'est pas la cause du défaut.** Aucune des 4 assertions ne LIT le contenu du stub :
     elles cherchent l'élément, pas son texte. Capturer l'option entière ne changerait strictement rien
     au pouvoir de détection de ce fichier. Le titre du constat (« le mock tronque l'option à 80
     caractères ») désigne donc un détail inerte comme mécanisme.
  2. **« auto-validant » est le mauvais label.** Rien ne se vérifie soi-même ici. Les quatre `it`
     s'appellent « rend **sans crash** … » — ils annoncent un test de fumée et en délivrent un ; deux
     d'entre eux (`objectiveEvents={null}`, L77 et L109) gardent réellement le chemin nul.
  3. **Incohérence interne de l'audit.** Le tableau « Constats écartés » exonère
     `MatchViewPage.test.tsx` / `MatchViewTabs.test.tsx` au motif qu'« elles testent ce qu'elles
     annoncent … pas d'auto-validation ». Le même critère, appliqué ici, exonère ces quatre `it`.
  4. **Ce n'est pas le seul mock tronquant** : `MatchNarrativeSection.test.tsx:19` fait
     `JSON.stringify(option).slice(0, 60)`. Le chiffre « Tests auto-validants : 1 fichier » est donc un
     artefact de la définition retenue.
  5. Le patron « option entière » existe bien ailleurs et n'est pas rare :
     `MatchScoreCurveChart.test.tsx:26`, `MatchScoreEventsChart.test.tsx:31`,
     `MatchAssistChart.test.tsx:24`, `MatchPositionsHeatmap.test.tsx:11`.
- Ce qui survit : l'overlay de captures CTF n'a effectivement aucun oracle — c'est un test **faible /
  fumigène**, du même grade que `MatchKillDistanceSection.test.tsx` que l'audit classe « faible », et non
  un test auto-validant. P1 est surcoté pour un défaut de cette nature.
- Conséquence réelle reformulée : les repères de captures CTF ne sont vérifiés par aucune assertion — le
  seul test qui les monte ne garantit que l'absence de plantage, ce qu'il annonce honnêtement dans ses
  quatre titres.

## Constat 4 — R6 format `MmSSs` : TIENT sur le compte (gravité → P2), RÉFUTÉ sur la « doc inversée »

- Ce que j'ai vérifié :
  - `grep -rn "padStart(2, '0')}s\`" apps/web/src --include=*.ts --include=*.tsx | grep -v test` →
    6 lignes : `components/ui/match-card.tsx:43` (avec une espace : `m ` — format différent),
    `match-view/MatchImpactBadgesBar.tsx:81`, `match-view/MatchKDCumulChart.tsx:81`,
    `match-view/_chartSeries.ts:27`, `synthesis/SynthesisBipolaireChart.tsx:52` (conditionnel),
    `lib/formatters/duration.ts:104` (canonique). **Trois copies dans `match-view/`, aucun garde-rail** :
    le fait matériel tient.
- Ce que l'auditeur n'a pas vu — **la citation de `_scoreCurve.ts:33` est tronquée par l'ellipse** :
  - Texte réel, `_scoreCurve.ts:33-34` : « L'instant en **M:SS** a UN foyer dans le dépôt (`lib/formatters`) :
    **la carte et la frise du rejeu** l'appellent toutes deux, aucune ne le réécrit. »
  - L'audit cite « l'instant en M:SS a UN foyer dans le dépôt (`lib/formatters`) **…** aucune ne le
    réécrit », et l'ellipse supprime précisément le SUJET de « aucune ». La phrase ne dit pas « personne
    dans le dépôt ne réécrit M:SS » : elle dit que **ces deux surfaces-là** ne le réécrivent pas. Et
    c'est vrai : `_scoreCurve.ts:35` réexporte `formatClockMMSS`, `useReplayTimeline.ts:123` l'appelle.
  - Second glissement : le commentaire porte sur **M:SS**, pas sur **MmSSs**. Il n'affirme rien sur la
    famille incriminée. L'accusation d'anti-pattern n°9 (« doc inversée ») ne tient sur aucun des deux
    plans.
- Ce que l'auditeur n'a pas exploité — la sémantique documentée invalide le cadrage « factorisation
  abandonnée » (anti-pattern n°8) :
  - `formatDurationMShort` (`duration.ts:98-105`) prend des **SECONDES** et rend `'-'` pour `0`/`null`.
    Or `MatchImpactBadgesBar.formatTime` (`:76-82`) prend des **ms** et rend **`null`** à ≤ 0 ;
    `MatchKDCumulChart.formatMmSs` (`:78-82`) prend des **ms** et doit rendre `'0m00s'` (étiquette d'axe) ;
    `_chartSeries.formatBinSeconds` (`:24-28`) prend des secondes mais rend `'0m00s'`. Deux des trois
    diffèrent d'UNITÉ, les trois diffèrent au zéro : aucune n'est substituable.
  - Le dépôt a déjà tranché ce cas exact et l'a DOCUMENTÉ : `duration.ts:30-43` (`formatClockMMSS`)
    explique « Une DURÉE nulle n'existe pas … Un INSTANT nul, lui, est le coup d'envoi ». Il manque donc
    la variante « instant » de `MmSSs`, pas une migration abandonnée. C'est un manque de complétion,
    pas une factorisation laissée en plan.
- Conséquence réelle reformulée : trois écritures du littéral `MmSSs` cohabitent dans `match-view/` sans
  garde-rail parce que la variante « instant » du helper canonique n'a pas été créée — mais aucun
  commentaire du dépôt n'affirme le contraire, et les trois copies ne sont pas interchangeables avec
  l'existant.

## Constat 5 — Allowlist `match-view=>match-replay` : TIENT (P1)

- Ce que j'ai vérifié :
  - `tools/lint-cross-feature-imports.mjs:87-99`, texte intégral lu. Il dit bien « Reciproque, et
    **STRICTEMENT bornee au chargement de l'artefact** » et se clôt sur « **cette exception ne couvre QUE
    le chargement de l'artefact** ». Aucune ambiguïté à exploiter, aucune seconde justification ailleurs
    dans le fichier.
  - `grep -rn "@/features/match-replay" apps/web/src/features/match-view --include=*.ts --include=*.tsx | grep -v test`
    → **9 lignes**. Trois sont `useMatchReplay` (`MatchScoreCurveChart.tsx:39`,
    `MatchScoreEventsChart.tsx:45`, `MatchImpactBadgesBar.tsx:38`) = couvertes. **Six ne le sont pas** :
    `equipmentKillBadges.ts:26` (`equipmentUsageLogic`), `:27` (`rosterLogic` — `buildPlayers`,
    `indexBySlot`, `playerName`), `:28` (type `replayNormalize`), `MatchImpactBadgesBar.tsx:37`
    (`REPLAY_TEXT`), `MatchViewTabChronology.tsx:9` et `:10` (deux sections React).
  - Chronologie des dates — décisive pour l'accusation de doc périmée :
    `git log -L 87,99:tools/lint-cross-feature-imports.mjs` → commentaire écrit le **2026-08-18**
    (`bb6eb7694`) ; `MatchViewTabChronology` reçoit les deux sections le **2026-08-25** (`9b5cbe116`) ;
    `equipmentKillBadges.ts` et l'import `REPLAY_TEXT` arrivent le **2026-08-30** (`260910c4d`).
    Les six imports hors borne sont donc TOUS postérieurs à la justification, qui n'a pas été rouverte.
- Ce qui confirme : j'ai cherché une exemption de repli (un second commentaire, une allowlist par module,
  un `// eslint-disable`) — il n'y en a aucune. Le lint ne raisonne qu'en paires `A=>B` : la paire étant
  déjà autorisée pour la query, il reste vert quoi qu'on importe ensuite.
- Conséquence réelle reformulée : la seule frontière écrite entre la Match View et le rejeu affirme ne
  couvrir que le chargement de l'artefact, alors que six imports sur neuf — dont `rosterLogic` et deux
  sections React — l'ont franchie dans les douze jours qui ont suivi sa rédaction, sans qu'aucun gate ne
  puisse le signaler.

## Constat 6 — `xuidMeta` / `teamColor` / `teamSeriesColor` sans test de comportement : RÉFUTÉ

- Ce que j'ai vérifié :
  - `apps/web/src/features/match-replay/ReplayTeams.test.tsx:12` importe `resolveXuidMeta` depuis
    `@/features/match-view/xuidMeta`, et le `it` de `:798-808` fait :
    `const board = [sbRow('A','Alpha','t0'), sbRow('B','Bravo','t1')]` → `resolveXuidMeta(board, 'A')` →
    `expect(headerOf('Équipe Eagle').style.borderLeft).toContain('var(--ac-team-ally)')` et
    `expect(headerOf('Équipe Cobra').style.borderLeft).toContain('var(--ac-team-enemy)')`.
    **C'est exactement la contre-épreuve que l'audit déclare inexistante** : inverser
    `team_side === allyTeam` en `!==` (`xuidMeta.ts:35`) rend `A` adverse et `B` allié, les deux couleurs
    s'échangent, le test rougit. L'affirmation « il resterait vert si la comparaison était inversée » est
    donc fausse.
  - `MatchScoreCurveChart.test.tsx:130-135` assert `opt.series[0].lineStyle.color === 'var(--ac-team-ally)'`
    et `series[1] === 'var(--ac-team-enemy)'`. Le montage (`:17-35`) ne mocke QUE `useMatchReplay` et
    `resolveToken`/`tokenCssVar` : la chaîne réelle `resolveXuidMeta` → `allyOfTeamId` →
    `teamSeriesColor` s'exécute et est assertée sur ses deux branches colorées. Idem
    `MatchScoreEventsChart.test.tsx:31` (option entière capturée).
  - `teamColor.ts` : `ReplayKillFeed.test.tsx:190-214` monte un scoreboard
    `{ team_side:'t0', team_color:'var(--team-témoin)' }` et assert
    `expect(icon.getAttribute('style')).toContain('var(--team-témoin)')` — c'est la tête de cascade de
    `teamColorResolver` (`teamColor.ts:36-52`, `team_color` du backend prioritaire) vérifiée sur pièce.
  - Le sous-constat « `xuidMeta.guard.test.ts:24` autorise un `xuidMeta.test.ts` inexistant » est
    matériellement vrai (`ls` : seuls `xuidMeta.ts` et `xuidMeta.guard.test.ts`), mais **inerte** :
    `ALLOWED` n'exempte que des noms de fichier du balayage grep. Un nom en trop n'affaiblit aucun garde,
    il ne dispense rien qui existe.
- Ce que l'auditeur n'a pas vu : sa reproduction (`ls apps/web/src/features/match-view/ | grep -E "^(xuidMeta|teamColor|teamSeriesColor)"`)
  ne cherche les tests QUE dans le dossier du helper. Les trois modules sont importés par
  `features/match-replay/` (allowlist `match-replay=>match-view`, `lint-cross-feature-imports.mjs:81-86`)
  et c'est là que vivent leurs oracles. La méthode de mesure produit mécaniquement le constat.
- Ce qui subsiste, très en deçà de l'énoncé : la branche neutre `teamSeriesColor(null)` /
  `teamTokenCssVar(null)`, les dosages `teamTintStyles` (22 % / 55 %) et `meXUIDOf` n'ont, eux, aucune
  assertion trouvée. C'est un manque ponctuel, pas « aucun test de comportement ».
- Conséquence réelle reformulée : la définition d'« allié » et l'encre de série d'équipe SONT couvertes
  par des oracles qui rougiraient sur une inversion — seules trois branches marginales (encre neutre,
  dosages de teinte, `meXUIDOf`) restent sans assertion.

## Bilan : 3 tiennent, 1 réfuté, 2 requalifiés

- **Tiennent** : constat 1 (P0 maintenu, renforcé par deux pièces que l'audit n'avait pas —
  `replayWindow.ts:118-140` montre que le dépôt sait faire le recalage et ne l'applique qu'au rejeu, et
  `MatchScoreCurveChart.test.tsx:121` verrouille la fausse prémisse) ; constat 2 (P1, avec « aucun test
  qui l'exerce » à corriger en « aucun oracle » pour `MatchKDCumulChart`, et confirmé sans réserve pour
  `MatchCadenceChart`, entièrement mocké) ; constat 5 (P1, confirmé sur les dates : les six imports hors
  borne sont postérieurs de 7 à 12 jours à la justification).
- **Réfuté** : constat 6 — `ReplayTeams.test.tsx:798-808` porte précisément la contre-épreuve de
  l'inversion, `MatchScoreCurveChart.test.tsx:134-135` assert les deux branches de `teamSeriesColor`,
  `ReplayKillFeed.test.tsx:190-214` celle de `teamColorResolver`. La reproduction proposée par l'audit ne
  regardait que le dossier du helper.
- **Requalifiés** : constat 3 → **P2** (les faits sont exacts mais la troncature à 80 caractères n'est pas
  le mécanisme — aucune assertion ne lit le stub ; « auto-validant » est le mauvais label pour quatre `it`
  qui s'annoncent « rend sans crash », et le critère d'exonération que l'audit applique aux tests de
  placement les exonérerait aussi) ; constat 4 → **P2** (les trois copies existent, mais l'accusation de
  « doc inversée » repose sur une citation dont l'ellipse supprime le sujet de « aucune » et confond
  M:SS avec MmSSs, et la divergence d'unité/de zéro fait de ce cas un helper à COMPLÉTER, non une
  factorisation abandonnée).
