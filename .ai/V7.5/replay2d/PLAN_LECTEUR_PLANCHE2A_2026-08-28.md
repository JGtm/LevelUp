# Plan — Lecteur du rejeu : frise a pistes, sauts, menu vitesse, raccourcis (planche 2a)

> Ecrit le 2026-08-28 par la session pilote. Contrat d'execution : skill `plan-execution`
> (ordre strict, un lot a la fois, statuts `[x]`/`[~]`/`[!]`, zero fix hors perimetre).
> Grille appliquee : skill `plan-review`. Spec produit :
> `.ai/V7.5/replay2d/SPEC_LECTEUR_PLANCHE2A_2026-08-28.md` (memo Claude Design + decisions
> utilisateur — elle PREVAUT sur toute intuition de mise en forme).
>
> Branche : `wt/lecteur` (worktree dedie `C:\Users\Guillaume\Downloads\Scripts\LevelUp-wt-lecteur`),
> base feat/v75 @ 86b9087c9. 1 commit par lot sur wt/lecteur. AUCUN merge vers feat/v75,
> AUCUN push — autorisation utilisateur attendue apres gate visuel.

## Objectif et critere de succes

La barre de lecture du rejeu passe au design valide (planche 2a) : sauts -/+10 s, frise a
remplissage + 4 pistes (Toi / Allies / Dominance / Medias vide), vitesse en menu, volume en
popover, pastilles de sortie nommees, raccourcis clavier. Succes : gates techniques verts
(tsc, vitest match-replay + routes, ESLint 0 sur fichiers touches, cliquet canvas <= 691),
comportements existants preserves (fil de kills inchange a l'ecran, regles son du 25/08,
bornes de gameplay), gate visuel utilisateur en fin de chantier.

## Prerequis (lot 0)

- [x] 0-1 `npm install` dans `apps/web` du worktree ; si tsc echoue sur les types generes :
      `make generate-types` depuis la racine du worktree.
      FAIT : npm install exit 0 ; `generated.ts` present ; `npx tsc -p apps/web --noEmit` VERT
      sans regeneration.
- [x] 0-2 Verifier sur pieces les ancres (le code a pu bouger) : `useReplayPlayback.ts`
      (onScrub/rewind/step), `ReplayCanvas.tsx` a 691 lignes pile, `ReplayTransport.tsx`,
      `ReplayKillFeed.tsx` (props kills/medals/t0Ms/doc), `replay.tsx` (memos kills/
      medalEvents), `useLeadMarks.ts`, `i18nContract.ts`/`i18n.ts` (cles leadChange*),
      `useReplaySettings.ts` (SPEED_MULTIPLIERS), `useReplaySound.ts` (mutedBySpeed,
      soundPlaysAtSpeed), `playerMarks.ts` (PlayerMarkKind = 'me' | 'friend'),
      `killFeedLogic.ts` (ReplayFeedEntry : key/replayMs/kill/death/medal).
      TOUTES VERIFIEES, deux ecarts par rapport au plan (cf. Decouvertes) : `ReplayCanvas.tsx`
      fait 692 lignes (pas 691) et le cliquet est DEJA ROUGE a la base ; `ReplayKillFeed` porte
      en plus les props `nowMs`/`playWindow`/`scoreboard`/`xuidMeta`/`marks`/`colorOf`, que le
      lot 1 conserve (seules kills/medals/t0Ms/doc tombent).
      Baseline mesuree avant tout code (vitest `src/features/match-replay src/routes`, hors
      sandbox) : 99 fichiers, 1502 tests, 1 SEUL rouge = le cliquet de taille.

## LOT 1 — Mecanique de lecture + donnee partagee (fondations)

- [x] 1-1 Patch `useReplayPlayback.ts` : `seekBy`, `stepFrames` (met en PAUSE d'abord),
      `seekTo` borne, helper local `writeCursor` (value + `--played`) appele par la boucle
      `step`, `seekTo`, `rewind`, `onScrub` et l'effet de pose initiale (spec §2). En-tete
      du fichier complete (2-3 lignes, pas un roman).
      `writeCursor` est un `useCallback([startFrame, endFrame])` (il entre dans les deps de la
      boucle) et borne le pourcentage a 0..100. La pose initiale l'appelle AUSSI quand elle ne
      repositionne pas — sinon `--played` reste vide au montage et la frise s'affiche creuse.
- [x] 1-2 Mise a jour `useReplayPlayback.test.tsx` : seekBy borne aux deux bouts,
      stepFrames pause + borne, `--played` suit seekTo/rewind/scrub.
      11 cas ajoutes (29 au total) : conversion secondes->images, bornage des deux commandes,
      pause du pas d'image, et `--played` sur les QUATRE chemins (saut, recommencer, glisse,
      boucle) — le champ est attache a la ref rendue par le hook.
- [x] 1-3 Fil aligne remonte d'un niveau : `replay.tsx` calcule
      `feedEntries = useMemo(() => buildFeedEntries(kills, medalEvents, t0Ms, data), ...)` ;
      `ReplayKillFeed` recoit `entries` (props kills/medals/t0Ms/doc supprimees, memo
      interne supprime) ; le rendu du fil est STRICTEMENT inchange a l'ecran.
- [x] 1-4 Mise a jour `ReplayKillFeed.test.tsx` (construit `entries` en amont) ; vitest
      routes vert (la page change de props).
- [x] 1-5 Cles i18n ajoutees (spec §3, FR ET EN, parite par typage) — les retraits lead*
      attendent le lot 3 (le composant vit encore).
      13 cles, contrat + deux tables. Ecarts de forme assumes : `2×` (typographie du depot)
      la ou la spec ecrit « 2x », et « REC » conserve en FR — c'est la spec qui le fixe, et
      c'est la convention universelle d'un bouton d'enregistrement.
- [x] 1-G Gate : `npx tsc -p apps/web --noEmit` (ou `make check-types`) VERT ; vitest cible
      `useReplayPlayback` + `ReplayKillFeed` + routes VERT. Commit lot 1.
      MESURE : tsc exit 0 ; vitest 5 fichiers / 95 tests verts ; ESLint 0 sur les 7 fichiers
      touches (avance du gate lot 3, pour ne pas accumuler).

## LOT 2 — Logique pure des pistes

- [ ] 2-1 `replayTimelineTracks.ts` : types `TrackKill`/`TrackDeath` (key, replayMs, xuid),
      `DominanceSpan`, `ReplayMediaItem`, `PlacedMedia` ; `trackScale(playWindow,
      frameCount)` ; `buildEventTracks(entries, marks, frameIntervalMs, scale, fmt)` →
      { own, allies } (regles produit spec §5) ; `buildDominance(changes, scale)` ;
      `placeMedia(media, frameIntervalMs, scale)`. Fichier <= 500 L, fonctions <= 80 L.
- [ ] 2-2 `replayTimelineTracks.test.ts` : les 5 cas de la spec §8 + bornes de fenetre +
      cles stables.
- [ ] 2-3 `SKIP_SECONDS = 10` dans `replayCanvasConfig.ts` (commentaire : convention
      lecteur video, libelle porte la duree).
- [ ] 2-G Gate : tsc VERT ; vitest `replayTimelineTracks` VERT. Commit lot 2.

## LOT 3 — Composants, cablage, suppression LeadMarks

- [ ] 3-1 `ReplayTimelineTracks.tsx` : frise (input range habille `--played`) + 4 pistes
      etiquetees (trackYou/trackAllies/trackDominance/mediaTrack), THUMB_PX = 16, encres
      par tokens (`tokenCssVar`), etat vide medias (`mediaEmpty`), infobulles horloge.
      Styles pseudo-elements du range dans `globals.css` (classe dediee, vars du theme,
      patron replay-feed-row) — AUCUN hex dans features/.
- [ ] 3-2 `ReplaySpeedMenu.tsx` (spec §6) + `ReplaySoundControls.tsx` remplace (volume en
      popover, regles du 25/08 conservees) + `useReplayShortcuts.ts` (spec §7).
- [ ] 3-3 `ReplayTransport.tsx` remplace (spec §6) : sauts -/+10 s, horloge tabular-nums
      sans font-mono, `timeline` (objet unique), pastilles nommees, REC conditionnel,
      aria conserves.
- [ ] 3-4 `useReplayTimeline.ts` (13e extraction) : assemblage complet (spec §4), appel
      `useReplayShortcuts` inclus. `ReplayCanvas.tsx` : prop `feedEntries`, appel du hook,
      passe `timeline` — et RESTE <= 691 lignes (extraire davantage si necessaire, jamais
      relever le cliquet).
- [ ] 3-5 SUPPRESSION `ReplayLeadMarks.tsx` + `ReplayLeadMarks.test.tsx` ; le type de
      retour de `useLeadMarks` demenage dans `useLeadMarks.ts` ; cles `leadChange`/
      `leadChangeAtFmt` retirees du contrat et des DEUX tables ; aucun import orphelin
      (`grep ReplayLeadMarks` = 0 hors historique).
- [ ] 3-6 Tests : `ReplayTimelineTracks.test.tsx`, `ReplaySpeedMenu.test.tsx`,
      `useReplayShortcuts.test.ts`, mise a jour `ReplayTransport.test.tsx` (spec §8).
- [ ] 3-G Gate : tsc VERT ; vitest `src/features/match-replay` COMPLET vert ; vitest
      `src/routes` vert ; ESLint 0 sur tous les fichiers touches ; cliquet
      `placementFamily.guard` VERT ; `grep -r "#[0-9a-fA-F]\{6\}"` muet sur les nouveaux
      fichiers ; `grep -ri archivo apps/web/src` muet ; aucune classe Tailwind couleur.
      Commit lot 3.

## LOT 4 — Cloture

- [ ] 4-1 Journal : entree datee dans `.ai/V7.5/replay2d/thought_log_replay.md` (decisions,
      chiffres de tests, deviations assumees) ; statuts du present plan tous poses.
- [ ] 4-2 Report REGISTRE : `.ai/V7.5/REGISTRE_REPORTS.md` — « Medias du rejeu phase 2 »
      (endpoint { id, kind, replayMs, durationMs, thumbUrl, url, label } par match/joueur,
      recalage t0/displayClockMs comme le fil ; condition de reprise : decision utilisateur
      apres livraison planche 2a) et « Lightbox medias + cles mediaOpen/mediaClose/
      mediaPausedHint » livrees avec la phase 2. Noter aussi : fichiers originaux de la
      session Claude Design non recuperes — si l'utilisateur les fournit, diff de
      reconciliation avant le gate visuel.
- [ ] 4-3 Commit final (docs) sur wt/lecteur. PAS de merge, PAS de push.

## Ce que ce plan NE fait PAS (perimetre ferme)

- Pas d'endpoint medias, pas de `ReplayMediaLightbox`, pas de cles i18n d'ouverture media.
- Pas d'Archivo ni d'aucune declaration de police.
- Aucun changement de comportement du fil de kills a l'ecran, des regles de son du
  2026-08-25, des bornes de gameplay, du tiroir de reglages.
- Aucun changement d'artefact, de contrat API, de SchemaVersion.
- Aucune retouche hors des fichiers listes ; toute decouverte va en « Decouvertes ».

## Environnement d'execution (pieges connus)

- vitest se lance HORS sandbox (echec connu en sandbox).
- `make check-types` depuis la RACINE du worktree ; npm install prealable (lot 0).
- Windows/PowerShell : pas de `&&` en PowerShell 5.1 ; preferer bash pour les chaines.
- Pas d'emojis dans les fichiers versionnes ; UI FR sans anglicismes ; commentaires de
  code dans la voix du depot (francais, majuscules sur les invariants).

## Protocole de reprise

Lire ce plan (statuts), la derniere entree du thought_log replay2d, `git log --oneline -5`
sur wt/lecteur. Reprendre a la premiere case non statuee du lot ouvert.

## Decouvertes (a consigner, ne pas traiter)

- (pilote, 2026-08-28) `FeedClock` du fil garde `font-mono` — l'abandon de font-mono ne
  vaut que pour la barre (spec §6) ; harmonisation eventuelle hors perimetre.
- (executeur, 2026-08-28, lot 0) LE CLIQUET DE TAILLE EST DEJA ROUGE A LA BASE feat/v75
  @ 86b9087c9 : `ReplayCanvas.tsx` mesure 692 lignes pour un plafond de 691
  (`placementFamily.guard.test.ts` : `src.split('\n').length - 1 <= 691`). Ce n'est donc pas
  une regression du present lot — elle preexiste. Le lot 3 la resorbe par construction
  (retrait des props sliderRef/minFrame/maxFrame/onScrub/leadMarks + extraction
  `useReplayTimeline`), et le gate exige le cliquet VERT : le plafond n'est PAS releve.
- (executeur, 2026-08-28, lot 1) `npm install` (lot 0) a REECRIT `apps/web/package-lock.json`
  en retirant 30 blocs `libc` (metadonnees de plateforme des binaires optionnels rollup/esbuild) :
  derive de la version locale de npm, sans rapport avec le lot, et potentiellement nuisible sur
  la CI Linux (ces champs filtrent glibc/musl). Le fichier a ete RESTAURE et n'entre dans aucun
  commit du chantier. A traiter separement si l'equipe veut aligner la version de npm.
- (executeur, 2026-08-28, lot 0) `ReplayKillFeed` porte six props que le plan ne cite pas
  (`nowMs`, `playWindow`, `scoreboard`, `xuidMeta`, `marks`, `colorOf`) : elles restent, la
  remontee ne concerne que l'assemblage du fil (kills/medals/t0Ms/doc -> `entries`).
