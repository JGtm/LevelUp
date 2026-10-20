# Vérification adverse V-WEB-4b

Cadre : dépôt `C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration`, branche `feat/v75`.
Le tip réel au moment de la vérification est `081871f09` (2026-09-05 22:07) et non `736ccf3c3`,
mais `git diff --stat 736ccf3c3..HEAD` sur les 11 fichiers audités (ci.yml, lefthook.yml,
vite.config.ts, playwright.config.ts, zoneStatesPaint.ts, canvasInk.ts + guard,
carrierPosition.guard, placementFamily.guard, ReplayCanvas.tsx, e2e/) rend **vide** : les
constats portent sur exactement le même code. Lecture seule, aucun test exécuté.

## Constat 1 — les 2 specs de rasterisation n'ont jamais tourné en CI : TIENT

- Ce que j'ai vérifié :
  - `.github/workflows/ci.yml:12-52` — le workflow n'a QUE deux déclencheurs : `push` (branches
    `main, feat/**, feature/**, fix/**, hotfix/**, refactor/**, perf/**, docs/**, chore/**,
    integration/**`) et `pull_request: branches: [main]`. `grep -n "workflow_dispatch\|schedule:"`
    sur `ci.yml` : **aucune sortie**. Pas de troisième porte.
  - `ci.yml:524` — `if: github.event_name == 'pull_request'` sur le job `e2e-react`. Le job est
    donc muet sur tout push de branche, y compris `feat/v75`.
  - Autre job ? `grep -rn "playwright" .github/workflows/` ne rend que `ci.yml` (l.523, 542, 607,
    625-626). Les 7 autres workflows (deploy, gitleaks, release, shared-social-gate, sync-labels,
    test-deploy-precheck, triage-feedback) n'en contiennent aucune trace. Le job homonyme
    `e2e-regen-demo-precheck` (`test-deploy-precheck.yml:204`) est une **simulation Docker de
    `rm -rf`/bind-mounts**, il ne lance pas Playwright — malgré son nom.
  - Hook local ? `lefthook.yml` lu **intégralement** (5 commandes pre-commit : gitleaks, gofmt,
    go-vet, check-merge-conflict, docs-fr-sync ; 9 commandes pre-push : 4 linters JS, knip,
    contract-ratchet, go-vet-cgo, govulncheck, shared-social-gate, rappel-gate-push).
    **Zéro occurrence de `playwright` ou `e2e`.**
  - PR sur la période, toutes refs confondues :
    `git log --oneline v7.3.0..HEAD --all --grep="Merge pull request" | wc -l` → **0**.
    Le dernier merge de PR du dépôt est `ad62a5650` (2026-07-18, PR #64), **antérieur** à
    `v7.3.0` (`a2719a68c`, 2026-08-04).
  - Les specs sont bien prises par le projet lancé : `playwright.config.ts:41` n'exclut du projet
    `chromium` que `/.*\.visual\.spec\.ts/` ; `replay-explosion-raster.spec.ts` et
    `replay-muzzle-raster.spec.ts` sont des `.spec.ts` ordinaires — elles TOURNERAIENT si le job
    tournait. Sur 31 specs `e2e/`, 2 seulement mentionnent le rejeu (`grep -lni "replay\|rejeu"`),
    et les 2 `*.visual.spec.ts` ne sont dans aucun projet lancé par la CI.

- Ce qui confirme : le mécanisme est établi ligne à ligne, pas par déduction — la seule
  condition d'exécution est `github.event_name == 'pull_request'`, et aucun autre point du dépôt
  n'invoque Playwright. Le fichier lui-même (`replay-explosion-raster.spec.ts:22-24`) documente
  qu'il est irremplaçable en vitest et qu'il **n'a besoin d'aucun serveur** — donc rien de
  technique ne s'opposait à le mettre dans `frontend`.

- Deux imprécisions de l'auditeur, sans effet sur le verdict :
  1. **La fenêtre n'est pas « depuis v7.3.0 »** : `git log --diff-filter=A` date la création des
     deux specs au **2026-08-15** (`2e1b67a8b`, `07e4fd189`), soit 11 jours APRÈS le tag. Le
     chiffre exact est **331 commits** touchant `match-replay` depuis leur création (et non 373
     depuis v7.3.0). L'ordre de grandeur et la conclusion sont inchangés.
  2. **« 0 Merge pull request » n'est pas strictement « 0 PR »** : une PR ouverte vers `main` puis
     fermée sans merge aurait déclenché `e2e-react`. Je ne peux pas l'exclure hors ligne (aucune
     ref `refs/pull/` locale). Mais c'est un angle mort de l'auditeur, pas une réfutation : la
     partie porteuse du constat — *le job ne se déclenche sur aucun des 331 pushes de la
     campagne* — se lit directement dans `ci.yml:524` et ne dépend d'aucune inférence.

- Conséquence réelle reformulée : les 589 lignes de la seule preuve de rendu du dépôt sont
  derrière une porte (`pull_request` vers `main`) que le mode branche unique n'ouvre jamais, et
  aucun autre workflow ni hook ne les rejoue.

## Constat 2 — `ZONE_ALPHA_ORDER`, constante vivante par son seul test : RÉFUTÉ

- Ce que j'ai vérifié :
  - `zoneStatesPaint.ts:108-114` — la déclaration :
    `export const ZONE_ALPHA_ORDER = [0, ZONE_UNDER_CAPTURE_FILL_ALPHA, ZONE_HELD_FILL_ALPHA,
    ZONE_ACTIVE_FILL_ALPHA, ZONE_CAPTURE_FILL_ALPHA] as const`
  - `grep` (ripgrep, sur `apps/web/src` seul) : 6 occurrences — déclaration (l.108), commentaire
    (l.54, l.97), import du test (l.33), et les 2 lignes de l'assertion (l.519-520). **Zéro
    consommateur de production : ce point de l'auditeur est exact.**

- Ce que l'auditeur n'a pas vu — et qui casse la qualification :
  1. **Ce n'est pas un tableau de littéraux.** Les 4 valeurs non nulles sont des **références** à
     des constantes déclarées 50 lignes plus haut et **toutes employées en production** :
     `ZONE_HELD_FILL_ALPHA` et `ZONE_ACTIVE_FILL_ALPHA` dans `paintOwnerFill` (l.219-222),
     `ZONE_CAPTURE_FILL_ALPHA` dans la progression (l.200), `ZONE_UNDER_CAPTURE_FILL_ALPHA`
     l.219. L'assertion `expect(ZONE_ALPHA_ORDER[i]).toBeGreaterThan(...[i-1])` porte donc sur un
     **invariant réel entre quatre constantes vivantes**, pas sur un tableau clos sur lui-même.
     La phrase-clé du constat — « il ne peut échouer que si quelqu'un édite ce tableau même,
     c'est-à-dire jamais par accident » — est **fausse sur pièces** : régler
     `ZONE_HELD_FILL_ALPHA` à 0,5 (au-dessus de `ACTIVE` = 0,42) sans toucher au tableau rend le
     test rouge.
  2. **L'accident visé s'est déjà produit une fois, et la source le date.**
     `zoneStatesPaint.ts:49-54` : « LES DEUX REMPLISSAGES D'APPARTENANCE ONT ÉTÉ RENFORCÉS le
     2026-08-25 (item D-R, retour utilisateur « la teinte est trop discrète ») : tenue 0,22 ->
     0,30, active 0,30 -> 0,42. L'écart ENTRE LES DEUX se creuse en même temps qu'ils montent —
     sans quoi renforcer la zone tenue aurait effacé ce qui distingue la colline active ». C'est
     exactement la classe d'édition — un réglage remonté seul, sous gate visuel utilisateur — que
     le test attrape, et l'auditeur cite le commentaire l.97-105 sans traiter ce qu'il dit.
  3. **Un `expect(result).toEqual(result)` déguisé serait `expect(sorted(x)).toEqual(sorted(x))`.**
     Ici le test asserte un ordre que rien dans le code ne garantit : les 5 valeurs sont
     indépendantes, aucun tri ne les produit.
  4. **Ce n'est pas un « dead code museum »** au sens de l'anti-pattern n° 1 (« conserver du code
     mort au cas où ») : rien de mort n'est conservé — les 4 valeurs référencées sont toutes
     peintes à l'écran. `ZONE_ALPHA_ORDER` est une **couture de test** assumée et documentée
     (« EXPORTÉE POUR ÊTRE TESTÉE », l.97-98), catégorie différente d'une constante débranchée.
  - Reste vrai : le test **ne couvre pas** un échange actif/tenu *à l'intérieur* de
    `paintOwnerFill` (l.219-222). Mais aucun test ne couvre tout ; ce que le voisin immédiat
    (`zoneStatesLayer.test.ts:505-514`) couvre déjà par mesure de rendu, c'est la hiérarchie
    teinte/progression. Une non-couverture n'est pas une auto-validation.

- Conséquence réelle reformulée : le test n'est pas une tautologie mais un invariant d'ordre sur
  quatre opacités de production que le dernier réglage visuel a effectivement fait bouger ; seul
  demeure un point mineur de style (un export réservé aux tests), qui ne justifie ni la
  qualification « auto-validation » ni la gravité P1.

## Constat 3 — `ReplayCanvas.tsx` : le seul test le concernant compte ses lignes : TIENT

- Ce que j'ai vérifié :
  - `placementFamily.guard.test.ts:300-304` — le corps exact :
    `it('ReplayCanvas.tsx reste sous son plafond', () => { const src = readFileSync(...);
    expect(src.split('\n').length - 1).toBeLessThanOrEqual(665) })`. Confirmé.
  - `wc -l` : `ReplayCanvas.tsx` = **664**, `placementFamily.guard.test.ts` = **305**. Le
    commentaire du cliquet court de l.146 à l.298, soit **153 des 305 lignes** du fichier —
    l'ordre de grandeur annoncé (150) est juste.
  - Exercice indirect, cherché de trois façons, toutes négatives :
    * `grep -rn "ReplayCanvas" src/ e2e/ --include=*.test.ts --include=*.test.tsx --include=*.spec.ts`
      → 6 lignes, **aucun import et aucun rendu** : une entrée d'allowlist qui vise un AUTRE
      fichier (`flagCarries.guard.test.ts:46` → `'ReplayCanvasTips.tsx'`), 3 commentaires
      (`placementDroppedWitness.test.ts:109`, `replayLogic.test.ts:267`, `useZoneStates.test.ts:4`)
      et les 2 lignes du cliquet.
    * Spec e2e de route : `grep -lni "replay\|rejeu" e2e/*.spec.ts e2e/visual/*.spec.ts` sur les
      31 specs ne rend QUE les 2 specs de rasterisation — qui posent un canevas par
      `page.setContent` et **ne montent jamais le composant**.
    * Test d'ordre par lecture de source : `grep -rn "drawGeometryLayer" --include=*.test.*` →
      **aucune sortie**. Aucun test, même par grep de fichier, n'observe la pile de calques.
  - La composition existe bien où l'auditeur la situe : `ReplayCanvas.tsx:359+` enchaîne
    `ctx.drawImage(mapImage)` / `drawGeometryLayer` / heatmap / zones / objectifs /
    `drawProjectilesLayer` / `weaponPads.paint` / `groundWeapons.paint` /
    `drawEquipmentPlacementsLayer` … chacun sous sa bascule (`showZones`, `showPlacements`, …).
  - Le cliquet est-il documenté comme mesure transitoire datée ? **Daté oui, transitoire non** :
    l.146-148 le rattache au « registre des reports (2026-08-16) », mais l'annonce explicitement
    comme permanent — « Le plafond n'est pas un idéal, c'est un CLIQUET : il ne remonte jamais ».
    Aucune date cible de retrait, aucun critère de sortie. Il ne bénéficie donc pas de
    l'exemption « kill-switch daté » de CLAUDE.md n° 11.

- Ce qui confirme : le journal du cliquet démontre lui-même qu'il pilote 18 extractions
  successives (861 → 665 lignes) — il fait travailler la structure du fichier, mais il n'affirme
  jamais rien sur ce que le fichier DESSINE. Une bascule inversée ou un calque non appelé laisse
  la CI verte.

- Conséquence réelle reformulée : la seule assertion de la CI sur le fichier qui ordonne la scène
  est sa longueur, et elle est hébergée dans un fichier dont le nom annonce les familles de pose.

## Constat 4 — `carrierPosition.guard.test.ts` : un 6e hook échapperait : TIENT

- Ce que j'ai vérifié, en lisant les 4 cas :
  - l.29-38 : `CALQUES_PORTEURS` = 5 noms écrits à la main, `LECTEURS_PURS` = 2. Confirmé.
  - Cas 1 (l.52) itère `CALQUES_PORTEURS` ; cas 2 (l.61) itère `LECTEURS_PURS` ; cas 3 (l.70)
    itère `[...CALQUES_PORTEURS, ...LECTEURS_PURS]` — **3 cas sur 4 sur listes manuelles**.
  - **Cas 4 (l.82-91), que l'auditeur soupçonnait de rattraper le coup : il ne le rattrape pas.**
    Il scanne bien `readdirSync(__dirname)`, mais son prédicat est
    `lire(f).includes('vehiclePositionAt')` (l.86) — **uniquement `vehiclePositionAt`**, jamais
    `buildPlayerPosAt`. Un 6e hook `useReplayXCarrier.ts` qui appelle `buildPlayerPosAt` sans
    toucher `vehiclePositionAt` traverse les quatre cas.
  - Second garde-rail voisin ? `livesPosition.guard.test.ts:38-52` scanne le dossier mais cherche
    `livesByXuid` — pas `buildPlayerPosAt`. Il ne ferme pas le trou.
  - `grep buildPlayerPosAt` sur `apps/web/src` : hors tests, **2 fichiers seulement** —
    `carrierPosition.ts:41,115` et `livesPosition.ts:97-98` (la définition). La reproduction de
    l'auditeur est exacte.
  - Le disclaimer du fichier (l.20-22) **ne couvre pas** ce cas : il ne concède qu'« un huitième
    lecteur qui naîtrait *sans jamais nommer aucune de ces fonctions* ». Le scénario de l'auditeur
    NOMME `buildPlayerPosAt` et passe quand même.

- Une nuance qui pèse sur le correctif, pas sur le constat : `buildPlayerPosAt` est un export
  **légitime et sanctionné**, `livesPosition.guard.test.ts:50` désignant nommément « ou
  `buildPlayerPosAt` (le bipède seul) » comme porte autorisée. L'inversion proposée par l'auditeur
  (interdire `buildPlayerPosAt(` partout sauf 2 fichiers) contredirait donc ce garde voisin : le
  prédicat à inverser doit viser les *hooks de porteur*, pas l'appel lui-même.

- Conséquence réelle reformulée : la régression que le fichier existe pour bloquer (« un porteur
  embarqué retraverserait le décor en ligne droite ») repasse dès qu'un 6e calque de porteur naît,
  et c'est précisément le régime de la campagne — 5 ajoutés le même jour (2026-09-05, l.6-9).

## Constat 5 — `canvasInk.guard.test.ts` promet six encres et en vérifie une : TIENT (gravité → P3)

- Ce que j'ai vérifié :
  - `canvasInk.ts:20-28` : `InkVar` déclare bien **6** membres — `--muted-foreground`, `--border`,
    `--foreground`, `--background`, `--card`, `--replay-label-stroke`.
  - `canvasInk.guard.test.ts:15` : `const REPLAY_INKS = ['--replay-label-stroke'] as const`.
    **1 entrée**. La liste est écrite à la main, non dérivée du type : la panne décrite (une 2e
    encre `--replay-*` ajoutée à `InkVar` sans être ajoutée à `REPLAY_INKS`) passerait. Ce noyau
    du constat tient.
  - `readInk` en chemin dégradé (`canvasInk.ts:35-42`) :
    `grep -rn "readInk" --include=*.test.*` ne rend **que la ligne de commentaire l.6** du garde.
    Aucun test n'exerce le retour `''`. Confirmé.
  - Autre garde couvrant les 5 restantes ? `fxInk.guard.test.ts:66-76` vérifie
    `SERVED_FX_TINTS` + `--replay-fx-core`, pas les tokens de base ;
    `replayOverlayStyles.guard.test.ts` ne lit pas `globals.css`. **Aucun**.

- Ce que l'auditeur n'a pas vu, et qui impose d'abaisser la gravité :
  1. **La restriction de portée est explicite dans le code même du garde**, `canvasInk.guard.test.ts:14` :
     « Les encres déclarées par `InkVar` (canvasInk.ts) **qui appartiennent au rejeu lui-même** ».
     Il n'y a **qu'une seule** encre propre au rejeu aujourd'hui (`--replay-label-stroke`, seule
     `--replay-*` de `InkVar`) : la couverture réelle est **1/1** du périmètre déclaré, pas 1/6.
     Le décalage est entre le docstring d'en-tête l.4-6 (« chaque `InkVar` ») et le commentaire
     l.14 — une doc trop large, pas un garde qui rate ce qu'il vise.
  2. **Les 5 autres sont des tokens de base du système de design**, pas des encres du rejeu, et
     chacune est déclarée **deux fois** dans `globals.css` (thème clair + sombre) — mesuré :
     `--muted-foreground` 2, `--border` 2, `--foreground` 2, `--background` 2, `--card` 2. Leur
     disparition ne perdrait pas « un contour de nom » : elle casserait toute l'application, bien
     avant le canvas. Le scénario de panne « en silence » ne s'applique pas à elles.

- Conséquence réelle reformulée : le garde couvre exactement ce qu'il déclare couvrir, mais par
  une liste manuelle non dérivée du type — donc il ne s'étendra pas tout seul à la prochaine
  encre propre au rejeu ; le défaut est une doc d'en-tête trop large plus un garde non
  auto-extensible, pas un trou de 5 encres non gardées. Gravité P1 → **P3**.

## Constat 6 — aucun seuil de couverture vitest : TIENT

- Ce que j'ai vérifié :
  - `apps/web/vite.config.ts`, bloc `coverage` : `provider: 'v8'`, `reporter: ['text','lcov','html']`,
    `reportsDirectory`, `include`, `exclude` — **aucune clé `thresholds`**.
    `grep -n "thresholds" apps/web/vite.config.ts apps/web/package.json` : **aucune sortie**.
  - `ci.yml:127-131` : la CI l'écrit elle-même — « P3.8 (revue **2026-04-29**) : activation Vitest
    coverage en CI. **Pas de ratchet pour l'instant (baseline a etablir une fois par main).** Le
    rapport HTML est uploade en artefact pour debug. » La date du 2026-04-29 est confirmée à la
    ligne près, et la note n'a jamais été levée (fichier identique entre `736ccf3c3` et HEAD).
  - L'artefact est bien produit et jamais relu : `ci.yml:133-140` (`upload-artifact`,
    `retention-days: 7`), aucun step ne le consomme.
  - Ratchet ailleurs ? `ls tools/` : `check-generated-types-fresh.mjs`, `knip-ratchet.mjs`,
    `lint-contract-ratchet.mjs`, `lint-cross-feature-imports.mjs`, `lint-no-hardcoded-colors.mjs`,
    `lint-no-hardcoded-fields.mjs` — **aucun ratchet de couverture web**. Les deux seuls existants
    sont Go : `apps/go-api/coverage_baseline.txt` (69.0, `ci.yml:407-411`) et
    `scripts/check_coverage_ratchet.sh` (shared_social, ADR 0021), aucun des deux ne lit
    `apps/web/coverage/`.
  - Le gate de non-régression de tests est Go-only : `.ai/baselines/tests_pre_migration.jsonl`
    contient du JSONL `go test` (`{"Package":"levelup/go-api/cmd/admin"}`) et
    `grep -c "apps/web\|vitest"` y rend **0**. Aucun équivalent web ne recense les tests attendus.

- Une réserve mineure, qui n'entame pas le constat : la suppression d'un test n'est pas *toujours*
  invisible. Le ratchet knip (`tools/knip-ratchet.mjs`, `THRESHOLDS = { files: 0, exports: 0,
  types: 0 }`, en CI sans condition de chemin, `ci.yml:97-98`) rougirait si le test supprimé était
  le **seul consommateur** d'un export — c'est le cas de `zoneStatesLayer.test.ts` vis-à-vis de
  `ZONE_ALPHA_ORDER`. Cette prise est fortuite et étroite : elle ne dit rien de la suppression
  d'un `it()`, d'un `describe.skip`, ni d'un test dont les cibles ont d'autres consommateurs.

- Conséquence réelle reformulée : la CI calcule un chiffre de couverture web, l'archive 7 jours et
  n'en tire aucun verdict ; la note « baseline à établir » est en place depuis 16 mois de
  calendrier projet.

## Bilan : 5 tiennent, 1 réfuté, 1 requalifié

- **RÉFUTÉ (1)** : constat 2 (P1-1, `ZONE_ALPHA_ORDER`). Le tableau agrège **quatre constantes de
  production** peintes par `paintOwnerFill`/la progression ; un réglage d'opacité remonté seul —
  l'accident exact que la source date au 2026-08-25 (tenue 0,22 → 0,30, active 0,30 → 0,42) —
  rend le test rouge. Ce n'est ni une auto-validation ni du code mort.
- **TIENT (5)**, dont **1 requalifié** : constat 5 (P1-5, `canvasInk`) → **P3** : le garde couvre
  1/1 de son périmètre déclaré (l.14, « les encres qui appartiennent au rejeu lui-même »), les
  5 autres `InkVar` étant des tokens de base doublement déclarés dans `globals.css` ; le défaut
  réel se réduit à une liste manuelle non dérivée du type et à un docstring d'en-tête trop large.
- Constats 1, 3, 4 et 6 tiennent **sans réserve de substance**. Deux imprécisions de chiffre ou de
  périmètre relevées et sans effet : la fenêtre du constat 1 est « depuis le 2026-08-15,
  331 commits » et non « depuis v7.3.0, 373 commits » (les specs n'existaient pas au tag), et
  « 0 merge de PR » n'exclut pas formellement une PR ouverte-puis-fermée — mais le mécanisme
  porteur (`ci.yml:524`) se lit directement, sans inférence.
