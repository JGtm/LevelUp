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
- D-B1 : écran de victoire pour les matchs à 2 ÉQUIPES uniquement. FFA, > 2 équipes,
  DNF : pas d'écran (noter en Découvertes si un cas fréquent apparaît). Égalité
  (outcome_code 1) : panneau « Égalité » neutre, sans logo, couleurs de thème.
- D-B2 : vainqueur déterminé par `header.outcome_code` du joueur de la page (1 = égalité,
  2 = victoire, 3 = défaite — confirmé par les tests Explorer) + son camp
  (`is_me`→`team_side` du scoreboard) : 2 → son équipe gagne, 3 → l'autre. Pas de
  déduction par le score timeline.
- D-B3 : couleur de l'écran = cascade d'IDENTITÉ (`teamColorResolver` de
  `features/match-view/teamColor.ts` : team_color backend → couleur officielle par
  team_id → token allié/adverse). C'est une EXCEPTION assumée à la décision D1 de la
  page rejeu (tokens seuls) — décision utilisateur du 2026-08-26 (« même couleur que
  les en-têtes du scoreboard Match View »). La documenter dans l'en-tête du composant.
  Le TEXTE reste en `--foreground` (jamais teinté), comme au scoreboard.
- D-B4 : score final affiché via `readScoreBanner` évalué à la borne de fin (réutilise
  la logique existante) ; s'il rend `null` (mode sans compteur), l'écran s'affiche sans
  ligne de score.
- D-B5 : pas de bouton de fermeture. L'écran est dérivé de la position de lecture :
  visible quand `frame ≥ borne de fin` (borne atteinte en lecture OU frise tirée au
  bout) ; recommencer ou remonter la frise le fait disparaître. Il COIFFE le canvas
  (absolute), il ne remplace pas le terrain.
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

- [ ] B1. Constantes d'outcome : chercher une constante front existante pour les codes
      1/2/3 (`grep -ri "outcome.*=.*2" apps/web/src/lib`). Si aucune : les nommer dans
      `apps/web/src/lib/halo/outcomes.ts` (nouveau, avec commentaire source « tests
      Explorer + API Halo ») — pas de nombre magique dans la logique.
- [ ] B2. NOUVEAU `victoryLogic.ts` (pur) :
      `readVictory(scoreboard, outcomeCode) → { teamID, teamSide, ally } | { tie: true } | null`
      selon D-B1/D-B2. Détermination des 2 camps par les `team_side` distincts du
      scoreboard (exactement 2, sinon null) ; camp du joueur = ligne `is_me`.
- [ ] B3. NOUVEAU `victoryLogic.test.ts` : victoire de mon camp, victoire adverse,
      égalité, FFA (pas de team_side) → null, 3 équipes → null, outcome_code absent →
      null, scoreboard sans `is_me` → null.
- [ ] B4. Recette de teinte : dans `features/match-view/teamColor.ts`, NOUVEAU
      `teamTintStyles(color)` rendant `{ background, border, accent }` (color-mix oklab
      22 % / 55 % / pleine — les trois littéraux actuels de `MatchScoreboard.tsx`).
      Migrer l'en-tête d'équipe de `MatchScoreboard.tsx` dessus DANS LE MÊME COMMIT +
      garde-rail (`teamTint.guard.test.ts` : le littéral `22%, transparent` interdit
      dans `features/` hors `teamColor.ts`) — règle CLAUDE.md n°6.
- [ ] B5. NOUVEAU `ReplayVictoryOverlay.tsx` : monté par la page au-dessus du canvas
      (conteneur `relative` de la section carte, overlay `absolute inset-0` centré,
      voile `bg` du thème semi-opaque). Contenu : logo (`teamLogoPath`) en filigrane
      grand format derrière le panneau, panneau avec `teamTintStyles`, titre
      « Victoire — {resolveTeamLabel} » (ou « Égalité »), ligne de score final (D-B4).
      Visible selon D-B5. `aria-live="polite"`, `role="status"`. En-tête : documenter
      l'exception D1 (D-B3).
- [ ] B6. i18n : clés FR + EN dans `match-replay/i18n.ts` (`victoryTitleFmt`,
      `tieTitle`, aria du panneau). FR sans anglicismes.
- [ ] B7. Test composant `ReplayVictoryOverlay.test.tsx` : rendu victoire (logo, label,
      couleur passée), rendu égalité (ni logo ni couleur d'équipe), masqué avant la
      borne, masqué quand `readVictory` rend null.

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

## Clôture (après Lot B)

- [ ] C1. Suite complète : `npm run test` (tout apps/web), typecheck purgé, lint.
- [ ] C2. Grep anti-régression : `formatClock(` hors helper D-A2 (aucun appel sur l'axe
      brut), littéral de recette hors `teamColor.ts`.
- [ ] C3. Entrée `.ai/thought_log.md` (date, décisions D-A1..D-B6, mesures, résultat
      des gates).
- [ ] C4. Commits sur `wt/replay-cadrage-victoire` (préfixe `feat(v7.5-rejeu):`), un
      commit par lot minimum. NE PAS merger dans `feat/v75`, NE PAS pousser sans
      instruction du pilote.
- [ ] C5. Gate visuel : à la main de l'UTILISATEUR (jamais l'agent) — témoins à ouvrir :
      000d5950 (victoire au score, T0 connu), e94163af (clamp début), 606d9844 (T0
      inconnu — début plein film mais fin cadrée + écran), 64e8adfa (fini au temps —
      vérifier le vainqueur affiché vs 2-2 au score). Le CR final liste ces 4 URLs
      locales `/…/matches/{id}/replay`.

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
- **Lot A, 2026-08-26 — invariant utile pour le Lot B** : quand la fenêtre existe et que le
  début n'est pas clampé, l'instant AFFICHÉ du fil retombe exactement sur `event_time_ms` de
  l'API — le recalage `+t0−originMs` de `killFeedLogic` est annulé par le retrait de
  `startMs`. Ce n'est pas un défaut : les deux chemins disent la même horloge de match.
