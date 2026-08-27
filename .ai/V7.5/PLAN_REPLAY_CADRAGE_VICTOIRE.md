# PLAN — Rejeu 2D : cadrage T0/fin réelle + écran de victoire

> Statut : ACTIF — exécution sous contrat `plan-execution` (ordre strict, gates, statuts
> `[x]`/`[~]`/`[!]`, zéro fix hors périmètre).
> Branche : `wt/replay-cadrage-victoire` (worktree `LevelUp-wt-replay-cadrage`), fusion
> cible : `feat/v75` (mode branche unique v7.5).
> Périmètre : **frontend uniquement** (`apps/web`). Aucun changement Go, aucun artefact
> à re-cuire, aucun schéma.
> Rédigé le 2026-08-26 après mesures sur pièces (4 artefacts témoins × base).

## Objectif

1. **Cadrer la lecture du rejeu 2D sur le match réel** : démarrer au début du gameplay
   (T0) et s'arrêter à la fin déclarée du match, en rognant le countdown pré-match et la
   queue post-game (~6 s de film).
2. **Écran de victoire** à l'instant de la fin : logo d'équipe en grand, « Victoire —
   Équipe X », score final, habillage couleur identité (recette 22 %/55 % du scoreboard
   de la Match View).

Critère de succès : sur les témoins listés en fin de plan, la lecture démarre au coup
d'envoi, s'arrête à l'annonce de victoire, l'écran s'affiche avec le bon vainqueur, le
bon logo, la bonne couleur ; tous les gates verts ; comportement actuel intact quand les
données manquent.

## Fondations mesurées (NE PAS re-dériver, c'est vérifié sur pièces)

- Axe du rejeu : frame 0 = premier paquet de position du film ; `doc.originMs` (schéma
  ≥ 4) = instant de la frame 0 sur l'horloge du film (zéro = `start_time`).
- `matchView.header.playable_duration_seconds` est calculé côté Go comme
  `duration_seconds − t0_ms/1000` quand T0 est connu, sinon durée API
  (`match_view_builders_header.go::headerGameplayDurationSeconds`). Conséquence :
  **fin du match sur l'axe du rejeu = `t0_ms + playable_duration_seconds×1000 − originMs`**
  — formule valide MÊME quand `t0_ms` est absent (l'ancrage se compense), erreur ≤ 1 s
  (troncature seconde).
- Début du gameplay sur l'axe du rejeu = `max(0, t0_ms − originMs)` (clamp : sur
  e94163af le premier paquet arrive APRÈS le T0).
- Le film ne garde que ~5-6 s après la fin déclarée (mesuré : 6,1 s / 6,1 s / 6,1 s /
  5,2 s sur les 4 témoins) — l'outro complète n'y est pas.
- Le dernier point du score timeline tombe 1,5-1,6 s AVANT la fin déclarée sur les 3
  matchs gagnés au score ; sur un match fini au temps (64e8adfa) il tombe 133 s avant —
  **ne jamais utiliser le score timeline comme borne de fin**.
- Valeurs témoins pour les tests (artefacts `data/cache/replays/halo_infinite/`,
  interval 100 ms) :

| Témoin | t0_ms | originMs | playable header (s) | frameCount | start attendu (frame) | fin attendue (frame) |
|---|---|---|---|---|---|---|
| 000d5950 | 18 465 | 3 604 | 478 | 4 985 | ~149 | ~4 929 |
| e94163af | 35 238 | 39 772 | 180 | 1 813 | 0 (clamp) | ~1 755 |
| 606d9844 | absent | 6 890 | 235 | 2 342 | 0 (repli) | ~2 281 |
| 64e8adfa | 29 069 | 10 516 | 810 | 8 337 | ~186 | ~8 286 |

(Recalculer précisément dans les tests avec la formule — les « ~ » viennent des
arrondis ; ne pas copier ces valeurs sans les recalculer.)

## Décisions produit TRANCHÉES (ne pas rouvrir en cours d'exécution)

- D-A1 : la queue post-game (~6 s) est rognée entièrement — la lecture ET la frise
  s'arrêtent à la fin déclarée. Pas d'option pour la voir.
- D-A2 : l'horloge AFFICHÉE (bandeau + fil) est recalée sur le gameplay :
  `affiché = ms − startMs`, plancher 0. L'axe interne (frames, replayMs) ne change pas.
- D-A3 : sans `originMs` (artefact pré-v4) ou sans `playable_duration_seconds` → aucun
  cadrage, aucun écran de victoire : comportement strictement actuel. `t0_ms` absent →
  début frame 0, la fin et l'écran restent actifs.
> **AMENDEMENT DU 2026-08-26 (décision utilisateur, en cours de Lot B) — L'ÉCRAN PREND LE
> POINT DE VUE DU JOUEUR DE LA PAGE.** Il n'est plus « l'écran du vainqueur » mais SON écran,
> comme dans le jeu : le TITRE est son verdict (`header.outcome_label`, déjà localisé côté
> serveur — aucun titre i18n local pour ces trois cas), et l'HABILLAGE (logo, couleur, nom
> d'équipe) est celui de SON équipe en victoire COMME en défaite. D-B1, D-B2, D-B3 et D-B5
> ci-dessous sont amendés en conséquence ; les cas SANS écran et la ligne de score ne changent
> pas.

- D-B1 (amendé) : écran de fin pour les matchs à 2 ÉQUIPES uniquement. FFA, > 2 équipes,
  DNF (code 4) : pas d'écran (noter en Découvertes si un cas fréquent apparaît). Égalité
  (outcome_code 1) : panneau neutre, sans logo ni couleur d'équipe, couleurs de thème —
  titre = `outcome_label` (« Égalité »), et la ligne de score reste.
- D-B2 (amendé) : issue déterminée par `header.outcome_code` du joueur de la page
  (1 = égalité, 2 = victoire, 3 = défaite — confirmé par les tests Explorer) + son camp
  (`is_me`→`team_side` du scoreboard). La lecture rend TROIS choses distinctes : l'issue,
  SON équipe (qui habille l'écran, toujours), et l'équipe qui gagne (2 → la sienne, 3 →
  l'autre). Pas de déduction par le score timeline. Le TITRE ne se fabrique pas côté front :
  c'est `header.outcome_label`, le même mot que la Match View.
- D-B3 (amendé) : couleur de l'écran = cascade d'IDENTITÉ (`teamColorResolver` de
  `features/match-view/teamColor.ts` : team_color backend → couleur officielle par
  team_id → token allié/adverse), appliquée à l'équipe DU JOUEUR DE LA PAGE. C'est une
  EXCEPTION assumée à la décision D1 de la page rejeu (tokens seuls) — décision utilisateur
  du 2026-08-26 (« même couleur que les en-têtes du scoreboard Match View »). La documenter
  dans l'en-tête du composant. Le TEXTE reste en `--foreground` (jamais teinté), comme au
  scoreboard.
- D-B4 : score final affiché via `readScoreBanner` évalué à la borne de fin (réutilise
  la logique existante) ; s'il rend `null` (mode sans compteur), l'écran s'affiche sans
  ligne de score.
- D-B5 (amendé) : pas de bouton de fermeture. L'écran est dérivé de la position de lecture :
  visible quand `frame ≥ borne de fin` (borne atteinte en lecture OU frise tirée au
  bout) ; recommencer ou remonter la frise le fait disparaître. Il COIFFE le canvas
  (absolute), il ne remplace pas le terrain. AJOUT D'EXÉCUTION : il est transparent au
  pointeur (`pointer-events-none`) — la frise vit sous le voile, et le seul geste qui fait
  disparaître l'écran serait sinon celui que le voile bloque.
- D-B6 : libellés i18n FR **et** EN (`Record<Locale, T>` — parité par typage), aucun
  anglicisme FR. Logo via `teamLogoPath` (`/titles/{slug}/teams/{id}.png`), `onError`
  → masqué proprement (l'écran reste valable sans logo).

## Lot A — Cadrage T0 / fin réelle

Tous les fichiers sous `apps/web/src/features/match-replay/` sauf mention.

- [x] A1. NOUVEAU `replayWindow.ts` (logique pure, sans React) :
      `replayWindow(doc, header) → { startFrame, endFrame, startMs, endMs } | null`.
      Entrées : `doc.originMs`, `doc.frameIntervalMs`, `doc.frameCount`,
      `header.t0_ms`, `header.playable_duration_seconds`. Règles : formules des
      Fondations ; `null` selon D-A3 ; clamps `startFrame ≥ 0`,
      `endFrame ≤ frameCount−1`, et `null` si `endFrame ≤ startFrame` (donnée
      incohérente → repli comportement actuel). En-tête de fichier : pourquoi la borne
      de fin n'utilise JAMAIS le score timeline (mesure 64e8adfa).
- [x] A2. NOUVEAU `replayWindow.test.ts` : les 4 témoins du tableau (valeurs recalculées
      exactes), + cas `originMs` absent → null, + `playable` absent → null, +
      `t0_ms` absent → start 0 mais fenêtre active, + incohérence (end ≤ start) → null.
- [x] A3. `useReplayPlayback.ts` : accepter la fenêtre (prop `window: … | null` dans
      `ReplayPlaybackOptions`). `null` → bornes actuelles (0, frameCount−1) ; sinon la
      boucle s'arrête à `endFrame` (même mécanique qu'aujourd'hui, borne différente),
      `restart()` et le rembobinage de `togglePlay()` ramènent à `startFrame`, et la
      PREMIÈRE lecture démarre à `startFrame`. Adapter l'en-tête du fichier.
- [x] A4. Frise de lecture : borne basse = `startFrame` (prop du composant transport —
      repérer le `<input type="range">` via `sliderRef`/`ReplayTransport` et son `min`).
      Un scrub ne peut plus sortir de la fenêtre.
- [x] A5. Horloge affichée (D-A2) : NOUVEAU helper pur `displayClockMs(ms, window)` dans
      `replayWindow.ts` (soustrait `startMs`, plancher 0 ; `window` null → identité).
      Recenser TOUS les affichages d'horloge de la page (`formatClock(` dans
      `match-replay/` — bandeau `ReplayScoreBanner`, fil `ReplayKillFeed`, transport,
      fiches le cas échéant) et les faire passer par le helper. Aucun affichage
      d'instant ne doit rester sur l'axe brut du rejeu.
- [x] A6. `ReplayCanvas.tsx` + page `replay.tsx` : la page calcule la fenêtre UNE fois
      (`useMemo` sur `replayWindow(doc, header)`) et la passe au canvas (qui la relaie à
      la lecture et au transport) et aux composants d'horloge. Pas de recalcul par
      composant.
- [x] A7. Tests de lecture : compléter `useReplayPlayback.test.tsx` — démarrage à
      `startFrame`, arrêt à `endFrame`, restart → `startFrame`, fenêtre null →
      comportement d'aujourd'hui inchangé.

Gate Lot A (obligatoire avant Lot B) :
```
cd apps/web
Remove-Item -Recurse -Force node_modules\.tmp   # anti faux-vert tsc
npm run typecheck
npm run test -- --run src/features/match-replay
npm run lint
```
Critère : 0 erreur, tests A2/A7 verts, aucun test existant cassé.

### Journal d'exécution — Lot A (2026-08-26, clos)

Gate passé dans cette session, codes de sortie relevés un par un (pas de sortie filtrée) :
`EXIT_TYPECHECK=0` · `EXIT_TEST=0` (77 fichiers, 1190 tests, 0 échec) · `EXIT_LINT=0`
(21 warnings, tous antérieurs au lot et hors fichiers touchés).

Valeurs recalculées des 4 témoins (arrondi à l'image la plus proche — tronquer coûtait une
image de gameplay à chaque bout) :

| Témoin | startMs | endMs | startFrame | endFrame | queue rognée |
|---|---|---|---|---|---|
| 000d5950 | 14 861 | 492 861 | 149 | 4 929 | 5,5 s |
| e94163af | 0 (clamp) | 175 466 | 0 | 1 755 | 5,7 s |
| 606d9844 | 0 (sans T0) | 228 110 | 0 | 2 281 | 6,0 s |
| 64e8adfa | 18 553 | 828 553 | 186 | 8 286 | 5,0 s |

Deux écarts de FORME par rapport à la lettre du plan, aucun sur le fond :

1. La prop s'appelle **`playWindow`**, pas `window` : dans `ReplayCanvas.tsx` une prop
   déstructurée nommée `window` masquerait le `window` global, dont le fichier lit
   `devicePixelRatio` et `matchMedia`. Le nom est le même partout pour ne pas avoir deux
   vocabulaires.
2. **Dixième extraction imposée par le cliquet de taille** : le canvas était PILE à 706
   lignes. L'horloge affichée et la publication bridée de l'image partent dans
   `useReplayClock.ts` ; le cliquet descend à **697** (`placementFamily.guard.test.ts`).

Deux compléments non listés mais imposés par A4/A5 (une frise cadrée dont les marques ne
suivent pas ment) : `ReplayLeadMarks` + `useLeadMarks` prennent la fenêtre pour l'ÉCHELLE des
marques de retournement ET pour leur infobulle ; `ReplayCanvasTips` + `ReplayPlacementTip` la
prennent pour l'instant du lâcher d'objet.

Recensement A5 — tous les affichages d'instant du dossier passent par `displayClockMs` :
bandeau (`ReplayScoreBanner`), horloge de la barre + total (`useReplayClock`), fil ×3
(`ReplayKillFeed`, recalé une fois au montage de la rangée), marques de retournement
(`ReplayLeadMarks.markClock`), infobulle de pose (`ReplayPlacementTip`). Restent bruts : la
définition de `formatClock` (`replayLogic.ts`) et les DURÉES (`formatSeconds`, âges de
lecture), qui ne datent rien.

## Lot B — Écran de victoire

- [x] B1. Constantes d'outcome : chercher une constante front existante pour les codes
      1/2/3 (`grep -ri "outcome.*=.*2" apps/web/src/lib`). Si aucune : les nommer dans
      `apps/web/src/lib/halo/outcomes.ts` (nouveau, avec commentaire source « tests
      Explorer + API Halo ») — pas de nombre magique dans la logique.
      RÉSOLU PAR L'EXISTANT : `apps/web/src/lib/outcome.ts` (`outcomeCodeToValue`, 2=win /
      3=loss / 1=tie / 4=dnf, défaut `null` hors contrat) est déjà LA source unique du
      dépôt, et elle porte son garde-rail (`outcome.guard.test.ts`, qui interdit tout
      mapping code→issue recopié ailleurs). Aucun fichier créé : en écrire un aurait été
      la 6e copie que ce garde-rail existe pour empêcher.
- [x] B2. NOUVEAU `victoryLogic.ts` (pur) :
      `readVictory(scoreboard, outcomeCode) → { teamID, teamSide, ally } | { tie: true } | null`
      selon D-B1/D-B2. Détermination des 2 camps par les `team_side` distincts du
      scoreboard (exactement 2, sinon null) ; camp du joueur = ligne `is_me`.
      FORME AMENDÉE (point de vue du joueur, 2026-08-26) : la lecture rend
      `{ outcome: 'win'|'loss'|'tie', mine: VictoryTeam | null, winner: VictoryTeam | null }`
      ou `null`. `mine` habille l'écran quelle que soit l'issue ; l'égalité rend les deux
      équipes à `null`, ce qui rend sa neutralité impossible à contourner au rendu.
- [x] B3. NOUVEAU `victoryLogic.test.ts` : victoire de mon camp, victoire adverse,
      égalité, FFA (pas de team_side) → null, 3 équipes → null, outcome_code absent →
      null, scoreboard sans `is_me` → null. + le cas qui porte l'amendement : en défaite,
      `mine` reste MON équipe et `winner` désigne l'autre.
- [x] B4. Recette de teinte : dans `features/match-view/teamColor.ts`, NOUVEAU
      `teamTintStyles(color)` rendant `{ background, border, accent }` (color-mix oklab
      22 % / 55 % / pleine — les trois littéraux actuels de `MatchScoreboard.tsx`).
      Migrer l'en-tête d'équipe de `MatchScoreboard.tsx` dessus DANS LE MÊME COMMIT +
      garde-rail (`teamTint.guard.test.ts` : le littéral `22%, transparent` interdit
      dans `features/` hors `teamColor.ts`) — règle CLAUDE.md n°6.
- [x] B5. NOUVEAU `ReplayVictoryOverlay.tsx` : monté par la page au-dessus du canvas
      (conteneur `relative` de la section carte, overlay `absolute inset-0` centré,
      voile `bg` du thème semi-opaque). Contenu AMENDÉ (2026-08-26) : logo
      (`teamLogoPath`) de MON équipe en filigrane grand format derrière le panneau,
      panneau avec `teamTintStyles` de MA couleur, titre = `header.outcome_label` servi
      par le backend, sous-titre = `resolveTeamLabel` de MON équipe, ligne de score final
      (D-B4). Égalité : panneau neutre, titre seul. Visible selon D-B5.
      `aria-live="polite"`, `role="status"`. En-tête : documenter l'exception D1 (D-B3)
      et le point de vue du joueur.
- [x] B6. i18n : clés FR + EN dans `match-replay/i18n.ts` (aria du panneau, libellé de la
      ligne de score). FR sans anglicismes. AMENDÉ : PAS de clé de titre — le verdict est
      `header.outcome_label`, déjà localisé côté serveur (les clés `victoryTitleFmt` /
      `tieTitle` écrites en début de lot ont été retirées, règle « 0 code mort »).
- [x] B7. Test composant `ReplayVictoryOverlay.test.tsx` : rendu victoire (logo, label,
      couleur passée), rendu égalité (ni logo ni couleur d'équipe), masqué avant la
      borne, masqué quand `readVictory` rend null. + les cas de l'amendement : DÉFAITE
      habillée de MON équipe (jamais l'emblème adverse), verdict backend écrit tel quel,
      libellé absent → pas d'écran, abandon → pas d'écran.

Gate Lot B :
```
cd apps/web
Remove-Item -Recurse -Force node_modules\.tmp
npm run typecheck
npm run test -- --run src/features/match-replay src/features/match-view
npm run lint
```
Critère : 0 erreur ; garde-rail B4 vert ; grep hex : aucun `#RRGGBB` introduit dans
`features/` (les hex d'équipe restent dans `lib/halo/teamNames.ts`, référentiel toléré).

### Journal d'exécution — Lot B (2026-08-26, clos)

Gate passé dans cette session, codes de sortie relevés un par un : `EXIT_TYPECHECK=0` ·
`EXIT_TEST=0` (105 fichiers, 1457 tests, 0 échec, sur `match-replay` + `match-view`) ·
`EXIT_LINT=0` (21 warnings, le même compte qu'au Lot A, tous antérieurs et hors fichiers
touchés). Grep hex : aucun `#RRGGBB` dans les fichiers non-test touchés.

**B1 n'a rien créé, et c'est le bon résultat.** La recherche a trouvé `lib/outcome.ts`,
source unique déjà en place ET déjà gardée (`outcome.guard.test.ts` interdit tout mapping
code→issue recopié hors de ce fichier). `victoryLogic` consomme `outcomeCodeToValue` :
aucune comparaison numérique n'apparaît dans le lot, le garde-rail existant reste vert.

**La lecture d'égalité exige elle aussi DEUX camps** (précision de D-B1, tranchée à
l'exécution) : le panneau annonce la fin d'un affrontement à deux camps, et l'afficher sur
un FFA laisserait croire que les huit joueurs ont fini à égalité. Un cas de test le fixe.

**Un cas de non-rendu de plus, non prévu par le plan : `outcome_label` absent.** Depuis
l'amendement, le titre EST ce libellé ; sans lui, le panneau n'annoncerait rien tout en
masquant le terrain. Le backend le garantit non vide au schéma — la garde n'existe que pour
la fenêtre où la Match View n'est pas encore chargée.

**L'overlay laisse passer les clics (`pointer-events-none`), et ce n'était pas prévu par le
plan.** La frise de lecture vit DANS `ReplayCanvas`, donc sous le voile : un overlay opaque
au pointeur aurait enfermé l'utilisateur dans l'écran de victoire, puisque le seul geste qui
le fait disparaître (remonter la frise, D-B5) est précisément celui que le voile aurait
bloqué. Le panneau n'a aucun élément interactif — rien ne se perd.

**Un élargissement de signature, hors lettre du plan mais imposé par B5** :
`teamColorResolver` prend désormais `readonly MatchScoreboardRow[]`. Le résolveur ne fait
que lire, et l'overlay tient son scoreboard en lecture seule. Généralisation stricte, aucun
appelant affecté.

**Découpe de la recette (B4)** : `teamTintStyles` rend les trois COULEURS (fond 22 %,
trait 55 %, accent plein) et laisse les ÉPAISSEURS au point d'appel — 2 px sous l'en-tête du
scoreboard, 4 px sur son bord gauche, 6 px sur le liseré du panneau de victoire : trois
rôles, trois mesures, une seule teinte. Le garde-rail ne surveille que le littéral du fond
(22 %) : le 55 % est un motif d'habillage général du dépôt (badges, pastilles Explorer) sans
rapport avec une couleur d'équipe, et l'interdire ferait rougir des surfaces innocentes.
Preuve du garde-rail relevée en session : littéral réintroduit dans `MatchScoreboard.tsx` →
test ROUGE (`offenders: ["src/features\\match-view\\MatchScoreboard.tsx"]`) ; sonde retirée
→ test VERT, tests du scoreboard inchangés.

**L'AMENDEMENT DU POINT DE VUE est arrivé en cours de lot** (décision utilisateur du
2026-08-26, alors que B5/B6/B7 étaient écrits et verts) : l'écran n'est plus celui du
vainqueur mais celui du JOUEUR DE LA PAGE. Trois conséquences, toutes reprises avant commit —
le titre passe de la clé i18n locale à `header.outcome_label` (les deux clés locales de titre
sont supprimées, pas laissées mortes), `readVictory` sépare l'issue / MON équipe / le
vainqueur, et l'habillage suit `mine` au lieu de `winner`. Le lot n'a jamais été committé sous
l'ancien contrat.

**Ce que l'overlay rend, dans les quatre cas** (mesuré par `ReplayVictoryOverlay.test.tsx`) :
victoire → titre « Victoire » (mot du backend), sous-titre « Équipe Eagle », logo
`/titles/halo_infinite/teams/0.png` en filigrane, teinte Eagle `#3B9DFF` (22 %/55 %/plein),
score final « 50 - 30 » ; DÉFAITE → titre « Défaite », et TOUT LE RESTE IDENTIQUE (Eagle,
son logo, sa couleur — un test vérifie que le rouge Cobra n'apparaît nulle part), score
« 50 - 30 » toujours des deux camps ; égalité → titre « Égalité », panneau neutre
(`bg-card`, `border-border`), ni logo ni couleur d'équipe, score conservé ; lecture nulle
(FFA, > 2 camps, abandon, code ou libellé absent, hors cadrage) → DOM vide, pas même un voile.

## Clôture (après Lot B)

- [x] C1. Suite complète : `npm run test` (tout apps/web), typecheck purgé, lint.
      PASSÉ PAR LE PILOTE (2026-08-26) : TSC=0, VITEST=0 (4784 passés / 14 skips
      antérieurs), LINT=0.
- [x] C2. Grep anti-régression : `formatClock(` hors helper D-A2 (aucun appel sur l'axe
      brut), littéral de recette hors `teamColor.ts`.
      PASSÉ (2026-08-26) : les 3 `formatClock` restants du fil lisent un `replayMs` déjà
      recalé au montage de la rangée ; `useReplayClock` recale avant d'écrire ; le littéral
      `22%, transparent` ne vit que dans `teamColor.ts` + son guard.
- [x] C3. Entrée `.ai/thought_log.md` (date, décisions D-A1..D-B6, mesures, résultat
      des gates). ÉCRITE (2026-08-26, en tête du journal).
- [x] C4. Commits sur `wt/replay-cadrage-victoire` (préfixe `feat(v7.5-rejeu):`), un
      commit par lot minimum. FAIT : `3a29f2e2f` (lot A), `50d24e848` (lot B), + commit
      de clôture (docs). NE PAS merger dans `feat/v75`, NE PAS pousser sans
      instruction du pilote.
- [ ] C5. Gate visuel : à la main de l'UTILISATEUR (jamais l'agent) — témoins à ouvrir :
      000d5950 (victoire au score, T0 connu), e94163af (clamp début), 606d9844 (T0
      inconnu — début plein film mais fin cadrée + écran), 64e8adfa (fini au temps —
      vérifier le vainqueur affiché vs 2-2 au score). Le CR final liste ces 4 URLs
      locales `/…/matches/{id}/replay`.

## Lot C — Sons de fin de partie (ajouté le 2026-08-27, vote utilisateur)

Sélection VOTÉE par l'utilisateur (2026-08-27, `Desktop/Halo Infinite - Sons armes/_fin_partie/vote_fin_partie.json`) ;
assets déjà extraits, identifiés (transcription locale) et normalisés par le pilote :
`_fin_partie/livraison/*.wav` — voix −16 LUFS, musiques −18 LUFS (décision utilisateur
« audible sans hurler »), plafond −1 dBTP, gain linéaire pur (recette du catalogue).

Décisions TRANCHÉES :
- D-C1 : à la fin du match (la LECTURE atteint `endFrame` — le même instant que l'écran),
  jouent ENSEMBLE : une voix d'annonceur (selon l'issue) + la fanfare (selon l'issue).
  Un seul déclenchement par arrivée en fin ; PAS de son quand on ATTEINT la fin par la
  frise (le son du rejeu ne vit que pendant la lecture — cohérent avec useReplaySound).
- D-C2 : issue → fichiers. Victoire : `end_victory_voice_{loc}_*` + `end_victory_music_01` ;
  défaite : `end_defeat_voice_{loc}_*` + `end_defeat_music_01` ; égalité :
  `end_tie_voice_{loc}_*` + `end_tie_music_01`. Plusieurs prises → tirage aléatoire à
  chaque déclenchement. `{loc}` = locale de l'UI (fr/en) — PREMIÈRE entrée locale-aware
  du catalogue son.
- D-C3 : FFA (écran absent, D-B1) : le SON joue quand même si le joueur de la page GAGNE
  (outcome_code 2) — FR `end_winner_voice_fr_01` (« Vainqueur »), EN : repli
  `end_victory_voice_en_01` (« Winner » isolé introuvable dans le pack EN — documenté).
  FFA perdu / égalité FFA : rien. Musique du FFA gagné : `end_victory_music_01`.
- D-C4 : le son de fin respecte le réglage existant (coupé par défaut, volume des
  réglages) — aucun son si le son du rejeu est désactivé.
- D-C5 : plafond `SOUND_CUT_MAX_S` relevé à 12.0 s (fichier le plus long : 11,67 s) avec
  mise à jour du COMMENTAIRE (la règle « plafond = plus long fichier livré » est
  préservée) ; le fondu de sortie de 0,25 s reste inchangé.

- [x] CS1. Copier les 14 wav de `_fin_partie/livraison/` vers
      `apps/web/static/sounds/halo_infinite/` (noms inchangés).
      CHEMIN CORRIGÉ SUR PIÈCES : `apps/web/static/` n'existe pas. Les sons du rejeu vivent
      à la RACINE du dépôt, `static/sounds/halo_infinite/` — c'est ce que résout déjà
      `replaySoundAssets.guard.test.ts` (`REPO_ROOT` = 5 niveaux au-dessus de la feature) et
      c'est là que sont les 44 fichiers existants. 44 -> 58.
- [x] CS2. NOUVEAU `endMatchSound.ts` (pur, testé) : table des fichiers par issue ×
      locale (voix, prises multiples) + fanfares ; `endMatchSounds(outcome, ffa, locale,
      rand)` → liste de chemins à jouer (`rand` injecté pour des tests déterministes).
      Issue dérivée de la MÊME lecture que l'écran (`victoryLogic.readVictory` /
      outcome_code pour le FFA) — pas de re-décodage parallèle.
      Signature du plan tenue à la lettre, plus deux fonctions que le câblage impose :
      `endMatchSoundSpec(scoreboard, outcomeCode, locale)` (la lecture, appuyée sur
      `readVictory`) et `endMatchSoundStems(spec)` (TOUTES les prises, pour le préchargement).
- [x] CS3. Déclenchement : quand la boucle de lecture passe à l'arrêt SUR endFrame
      (transition `ended` de `useReplayPlayback`), publier l'événement de fin ; le
      lecteur audio joue les fichiers de CS2 (voix et musique en même temps). Une seule
      fois par arrivée ; « Recommencer » réarme.
- [x] CS4. Manifeste + garde-rail : les 14 stems entrent dans la liste que
      `replaySoundAssets.guard.test.ts` rejoue contre le dossier (un stem sans fichier
      ou un fichier sans stem = test rouge).
- [x] CS5. D-C5 : plafond à 12.0 + commentaire mis à jour dans `replayAudio.ts`.
- [x] CS6. Tests : sélection (issue × locale × FFA × tirage), déclenchement unique,
      silence quand le son est coupé.

Gate Lot C : purge `node_modules\.tmp` ; `npm run typecheck` ; `npm run test -- --run
src/features/match-replay` ; `npm run lint` — codes de sortie vérifiés un à un.
Commit unique : `feat(v7.5-rejeu): lot C — sons de fin de partie (annonceur + fanfare)`.

### Journal d'exécution — Lot C (2026-08-27, clos)

Gate passé dans cette session, codes de sortie relevés un par un (`node_modules\.tmp` purgé
avant) : `EXIT_TYPECHECK=0` · `EXIT_TEST=0` (80 fichiers, 1258 tests, 0 échec sur
`src/features/match-replay` — le Lot A en relevait 77/1190 sur le même filtre) ·
`EXIT_LINT=0` (21 warnings, le MÊME compte qu'aux lots A et B ; le seul du dossier porte sur
`ReplayFeedName.tsx`, que le lot ne touche pas).

**Ce que joue chaque fin**, mesuré par `endMatchSound.test.ts` (voix + fanfare, toujours
ensemble) :

| Issue | Deux camps FR | Deux camps EN | FFA (aucun camp lisible) |
|---|---|---|---|
| Victoire | `end_victory_voice_fr_01\|02` + `end_victory_music_01` | `end_victory_voice_en_01` + `end_victory_music_01` | `end_winner_voice_fr_01` (FR) / `end_victory_voice_en_01` (EN) + `end_victory_music_01` |
| Défaite | `end_defeat_voice_fr_01\|02` + `end_defeat_music_01` | `end_defeat_voice_en_01\|02` + `end_defeat_music_01` | rien |
| Égalité | `end_tie_voice_fr_01\|02` + `end_tie_music_01` | `end_tie_voice_en_01` + `end_tie_music_01` | rien |

`|` = deux prises, tirage à chaque déclenchement (`rand` injecté ; production `Math.random`).
Abandon (code 4), code absent, en-tête pas encore chargé : rien.

**LE DRAPEAU `ffa` DIT « AUCUNE ÉQUIPE À NOMMER », PAS SEULEMENT « FFA »** (précision tranchée
à l'exécution). Il est vrai dès que `readVictory` ne rend rien : FFA, plus de deux camps, ou
scoreboard sans ligne `is_me`. Dans ces trois cas seule la VICTOIRE sonne, et par la réplique
qui ne nomme personne — « Vainqueur » dit vrai sans rien supposer, là où « Défaite » et
« Égalité » supposent un affrontement à deux camps. Un cas de test le fixe.

**« ARRIVER » N'EST PAS « Y ÊTRE » — c'est ce qui fait tenir D-C1.** La condition du
déclenchement est `from < endFrame` où `from` est l'image d'AVANT le pas d'animation : une
lecture qui franchit la borne annonce, une frise tirée jusqu'au bout (qui pose déjà le curseur
SUR la borne) n'annonce rien. L'unicité en découle sans compteur ni drapeau : la boucle
s'arrête là, et en repartir passe soit par un rembobinage (« Recommencer », « Lecture » sur un
rejeu terminé) soit par une position en deçà de la borne. Quatre cas dans
`useReplayPlayback.test.tsx`.

**LA CONCLUSION OBÉIT AUSSI AU SILENCE D'AVANCE RAPIDE**, et ce n'était pas écrit dans D-C4 :
au-delà de `SOUND_MAX_SPEED`, le panneau de réglages AFFIRME « son coupé par la vitesse ». Y
faire jouer une fanfare rendrait cette phrase fausse à l'écran. Le son coupé et le volume
(D-C4) sont respectés par construction : la conclusion passe par le même `ReplayAudioPlayer`.

**LES PRISES SONT PRÉCHARGÉES AVEC LA PISTE, tirage compris** — trois fichiers au lieu d'un.
Le tirage n'a lieu qu'à l'arrivée en fin, et `ReplayAudioPlayer.play` rend le silence (puis
charge) sur un buffer absent : précharger la seule prise tirée aurait donné un premier écran
de fin muet, à chaque match.

**Le plafond du lecteur et la borne des sons d'événement ont CESSÉ d'être le même nombre.**
`SOUND_CUT_MAX_S` passe de 4,0 à 12,0 (D-C5) : sans autre garde, une explosion de grenade
re-livrée à 9 s ne ferait plus rougir personne. `replaySoundAssets.guard.test.ts` porte donc
désormais `LONG_MAX_S = 4.0` pour les sons d'ÉVÉNEMENT, plus un cas qui tient la règle de
D-C5 elle-même : le plafond reste au-dessus du plus long fichier livré ET à moins d'une
seconde de lui (11,667 s, `end_tie_music_01`). Un plafond qui décolle ne protège plus.

**Onzième extraction imposée par le cliquet de taille** : `ReplayCanvas.tsx` était PILE à 697
et le lot y fait entrer une prop (la fin de partie, relayée au lecteur et à la lecture). Les
cinq mémos d'effets précalculés — tirs, « ! » du tireur, morts, fins de vol, grappin — partent
dans `useReplayFx.ts` ; ils ne dépendent que du film, ne lisent ni thème ni cadrage et ne
dessinent rien. Noms inchangés, pas une ligne du tracé ne bouge. Le cliquet descend à **691**.

**Preuve du garde-rail relevée en session** : `end_winner_voice_fr_01.wav` retiré du dossier →
`EXIT=1`, 2 tests rouges (`expected [ 'end_winner_voice_fr_01' ] to deeply equal []` sur
« chaque stem du manifeste a son fichier .wav », plus le cas de plafond qui lit toutes les
durées) ; fichier remis → `EXIT=0`, 14 tests verts.

**Aucune string UI nouvelle**, donc aucune clé i18n : le lot n'ajoute ni réglage ni libellé.
La LANGUE, elle, entre pour la première fois dans le catalogue sonore — c'est la prop `locale`
de la page rejeu, la même que le fil, les fiches et le tiroir.

Extension future consignée (HORS lot C) : fins multi-équipes par couleur — 8 répliques
FR (« Partie terminée, l'équipe bleue/rouge/cyan/mauve/verte/citron/jaune/orange est
déclarée vainqueur ») et 8 EN (« Game over — <color> team wins ») identifiées dans les
packs annonceur ; correspondance constructible via `team_id` → couleur officielle
(`lib/halo/teamNames.ts`). À traiter avec l'écran multi-équipes s'il voit le jour.

## Découvertes hors périmètre

(À consigner ici pendant l'exécution — ne rien corriger.)

- **Lot A, 2026-08-26 — `ReplayKillFeed.tsx` est PILE au seuil.** 498 lignes avant le lot,
  500 après (une ligne d'import, une prop). La prochaine addition impose une extraction :
  candidat naturel, les trois formes de rangée (`FeedLine` / `DeathLine` / `KillLine`) dans
  un fichier voisin, en emportant `FEED_ROW` et les constantes de pictogramme avec elles
  (sans quoi l'import serait circulaire). Non traité : hors périmètre du lot.
- **Lot A, 2026-08-26 — `killFeedLogic.ts` pèse 541 lignes**, au-dessus du seuil de 500 de
  CLAUDE.md. Dette ANTÉRIEURE au lot, non aggravée (le fichier n'est pas touché).
- **Lot A, 2026-08-26 — cinq fichiers de test du dossier dépassent 500 lignes**
  (`ReplayTeams.test.tsx` 960, `replaySound.test.ts` 741, `ReplayKillFeed.test.tsx` 656,
  `zoneStatesLayer.test.ts` 574, `useReplayWeaponPads.test.ts` 550). Dette antérieure.
- **Lot B, 2026-08-26 — `i18n.ts` du rejeu est à 534 lignes**, au-dessus du seuil de 500.
  Le fichier était DÉJÀ à 530 avant le lot (dette antérieure : la découpe du 2026-08-18 avait
  sorti le contrat, et les deux tables l'ont regarni depuis). Le lot y ajoute 4 lignes — 2
  clés × 2 langues, indivisibles. La découpe naturelle serait par BLOC (fil, fiches, réglages)
  plutôt qu'une nouvelle extraction transverse. Non traité : hors périmètre.
- **Lot B, 2026-08-26 — la ligne de score de l'overlay dépend de `readScoreBanner`**, donc
  du côté allié : un scoreboard sans `xuidMeta` fait disparaître le score alors que le
  VAINQUEUR, lui, reste affiché (il ne dépend que de `outcome_code` et de `is_me`). Ce n'est
  pas incohérent — le titre affirme moins que deux nombres — mais c'est une asymétrie à
  connaître si un témoin montre un panneau sans chiffres.
- **Lot C, 2026-08-27 — le son reste MUET après un rechargement de page tant qu'on n'a pas
  rebasculé le bouton deux fois.** La préférence `replay-sound-on` est restaurée à `true` au
  montage, mais le `ReplayAudioPlayer` ne naît QUE dans `toggle()` (politique d'autoplay) :
  `playerRef` est donc nul, le battement se tait, et le premier clic du bouton — qui semble
  « activer » — coupe en réalité une préférence déjà à `true`. Dette ANTÉRIEURE au lot (elle
  vaut pour tous les sons du rejeu, pas seulement la fin de partie) ; elle touche aussi la
  conclusion sonore, qui ne partira pas sur un rejeu ouvert par un rechargement. Le correctif
  naturel serait un premier geste QUELCONQUE de la page qui crée le lecteur quand la
  préférence est déjà à `true`. Non traité : hors périmètre.
- **Lot C, 2026-08-27 — `i18n.ts` du rejeu reste à 534 lignes** (découverte du lot B). Le lot C
  n'y touche pas : il n'ajoute aucune string UI.
- **Lot A, 2026-08-26 — invariant utile pour le Lot B** : quand la fenêtre existe et que le
  début n'est pas clampé, l'instant AFFICHÉ du fil retombe exactement sur `event_time_ms` de
  l'API — le recalage `+t0−originMs` de `killFeedLogic` est annulé par le retrait de
  `startMs`. Ce n'est pas un défaut : les deux chemins disent la même horloge de match.
