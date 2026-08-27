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

## AMENDEMENT DU 2026-08-28 — port de `planche2a_impl/` acte

L'utilisateur a fourni le handoff ORIGINAL de la session Claude Design en cours de lot 2 :
`.ai/V7.5/replay2d/planche2a_impl/` (7 fichiers de code + `IMPLEMENTATION.md` + le canvas
`.dc.html`). REGLE QUI PREVAUT DESORMAIS sur la spec derivee : les 5 nouveaux composants/hooks
et les 2 remplacements sont PORTES TELS QUELS, ils ne sont pas reecrits. Adaptations
obligatoires, et elles seules :

1. En-tete « DESTINATION : ... » retiree de chaque fichier (metadonnee de handoff).
2. **Medias — decision revisee** : `ReplayMediaLightbox.tsx` EST livre dans ce chantier, avec
   ses trois cles `mediaOpen`/`mediaClose`/`mediaPausedHint`. Le perimetre ferme ci-dessous est
   amende en consequence ; ne reste en phase 2 que la DONNEE (endpoint + passage de la prop).
3. i18n : les tables fr/en du §3 d'`IMPLEMENTATION.md` MOT POUR MOT (les libelles de la spec
   derivee etaient des approximations sans accents).
4. `SKIP_SECONDS` : le design l'exporte depuis `ReplayTransport.tsx` ; la constante vit dans
   `replayCanvasConfig.ts` (un export non-composant depuis un fichier de composant declenche
   `react-refresh/only-export-components`). Transport et `useReplayTimeline` l'importent de la.
5. `ReplaySpeedMenu` : la constante locale `SOUND_MAX_SPEED` est remplacee par
   `soundPlaysAtSpeed` (`replaySoundCursor.ts`, verifie sur pieces) — pas de 3e copie de la
   regle du son.
6. `ReplayMediaLightbox` : `bg-black` (classe Tailwind couleur en `features/`) devient un token.
7. Le cablage du §4 inline les memos dans le canvas : NON — l'assemblage reste dans
   `useReplayTimeline.ts` (cliquet 691, deja rouge a la base).
8. Les deux deviations playback du lot 1 tiennent (`stepFrames` met en pause ; `writeCursor`
   aussi dans `onScrub` et la pose initiale).
9. Tests ecrits CONTRE les fichiers portes, plus : clic media avec `playing` appelle
   `onRequestPause` ; etat vide medias ; menu vitesse ferme sur Echap/clic dehors ; raccourcis
   J/K/L/M/R du fichier porte.

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

- [x] 2-1 `replayTimelineTracks.ts` : types `TrackKill`/`TrackDeath` (key, replayMs, xuid),
      `DominanceSpan`, `ReplayMediaItem`, `PlacedMedia` ; `trackScale(playWindow,
      frameCount)` ; `buildEventTracks(entries, marks, frameIntervalMs, scale, fmt)` →
      { own, allies } (regles produit spec §5) ; `buildDominance(changes, scale)` ;
      `placeMedia(media, frameIntervalMs, scale)`. Fichier <= 500 L, fonctions <= 80 L.
      PORTE de `planche2a_impl/` (amendement), donc plus riche que la description ci-dessus :
      il porte AUSSI `THUMB_PX`/`trackLeft`/`trackWidth` (geometrie du curseur natif),
      `ratioOfMs` et `clipFrameCount`. Signatures reelles : `buildEventTracks(kills, deaths,
      marks, frameIntervalMs, scale, clockOf)` (deux listes DEJA reduites, pas les entries — la
      reduction se fait dans `useReplayTimeline`), `buildDominance(changes, scale)`,
      `DominanceSegment` (pas `Span`), `kind: 'image' | 'clip'`. Une version maison ecrite
      avant l'arrivee du handoff a ete REMPLACEE par le port. 236 L.
- [x] 2-2 `replayTimelineTracks.test.ts` : les 5 cas de la spec §8 + bornes de fenetre +
      cles stables. 26 cas — dont la geometrie `trackLeft`/`trackWidth` et `clipFrameCount`,
      qui n'existaient pas dans la spec derivee.
- [x] 2-3 `SKIP_SECONDS = 10` dans `replayCanvasConfig.ts` (commentaire : convention
      lecteur video, libelle porte la duree). Adaptation 4 de l'amendement : le design
      l'exportait depuis `ReplayTransport.tsx`.
- [x] 2-G Gate : tsc VERT ; vitest `replayTimelineTracks` VERT. Commit lot 2.
      MESURE : tsc exit 0 ; 26 tests verts ; ESLint 0 sur les 5 fichiers touches.
      Inclus dans ce lot : correction i18n du lot 1 (adaptation 3) — `speedMuted` devient
      'son coupé'/'sound off' (la borne est dite par les entrees qui portent la note, pas par
      le texte), et les trois cles de la lightbox sont ajoutees.

## LOT 3 — Composants, cablage, suppression LeadMarks

- [x] 3-1 `ReplayTimelineTracks.tsx` : frise (input range habille `--played`) + 4 pistes
      etiquetees (trackYou/trackAllies/trackDominance/mediaTrack), THUMB_PX = 16, encres
      par tokens (`tokenCssVar`), etat vide medias (`mediaEmpty`), infobulles horloge.
      PORT du fichier design : les pseudo-elements du range sont habilles par variantes
      Tailwind arbitraires `[&::-webkit-slider-*]` avec les vars du theme, PAS par une classe
      de `globals.css` — la feuille n'est donc pas touchee (item du plan derive caduc).
- [x] 3-2 `ReplaySpeedMenu.tsx` (spec §6) + `ReplaySoundControls.tsx` remplace (volume en
      popover, regles du 25/08 conservees) + `useReplayShortcuts.ts` (spec §7). PORTS.
      ECART DE LA SPEC DERIVEE, assume : le volume du design N'EST PAS en popover — c'est un
      curseur habille de 58 px posé a cote du haut-parleur. Le fichier valide fait foi (amendement).
      Les trois regles du 25/08 sont tenues telles quelles : `available` garde le bouton, le
      curseur ne s'escamote pas (zero + estompe + infobulle), `mutedBySpeed` estompe et explique.
- [x] 3-2b `ReplayMediaLightbox.tsx` (amendement point 2) : PORT, avec l'adaptation 6
      (`bg-black` -> `bg-muted`). Non branchee a une donnee — ouverte par la piste medias.
- [x] 3-3 `ReplayTransport.tsx` remplace (spec §6) : sauts -/+10 s, horloge tabular-nums
      sans font-mono, `timeline` (objet unique), pastilles nommees, REC conditionnel,
      aria conserves. PORT + adaptation 4 (`SKIP_SECONDS` importe de replayCanvasConfig).
- [x] 3-4 `useReplayTimeline.ts` (13e extraction) : assemblage complet (spec §4), appel
      `useReplayShortcuts` inclus. `ReplayCanvas.tsx` : prop `feedEntries`, appel du hook,
      passe `timeline` — et RESTE <= 691 lignes (extraire davantage si necessaire, jamais
      relever le cliquet).
      IL A FALLU UNE 14e EXTRACTION : avec la seule 13e, le canvas retombait a 708 (il partait
      de 692, deja au-dessus). `useReplayDrawer.ts` sort le montage du tiroir — une cinquantaine
      de lignes qui RECOPIAIENT trente bascules sans rien decider. Resultat : 674 lignes, et le
      plafond du cliquet DESCEND a 674 (patron du fichier depuis 861).
- [x] 3-5 SUPPRESSION `ReplayLeadMarks.tsx` + `ReplayLeadMarks.test.tsx` ; le type de
      retour de `useLeadMarks` demenage dans `useLeadMarks.ts` ; cles `leadChange`/
      `leadChangeAtFmt` retirees du contrat et des DEUX tables ; aucun import orphelin
      (`grep ReplayLeadMarks` = 0 hors historique).
      Le type s'appelle `ReplayLeadMarks` (etait `ReplayLeadMarksProps` — il ne decrit plus les
      props d'un composant, mais ce que le hook rend). Les renvois documentaires au composant
      supprime sont reecrits, pas laisses pendants.
- [x] 3-6 Tests : `ReplayTimelineTracks.test.tsx`, `ReplaySpeedMenu.test.tsx`,
      `useReplayShortcuts.test.ts`, mise a jour `ReplayTransport.test.tsx` (spec §8), plus les
      quatre cas de l'amendement point 9 (clic media -> `onRequestPause` ; etat vide medias ;
      menu vitesse ferme sur Echap et clic dehors ; raccourcis J/K/L/M/R).
      52 cas sur les quatre fichiers. Les deux cas « quatre boutons de vitesse » de l'ancien
      ReplayTransport.test ont migre vers `ReplaySpeedMenu.test.tsx` ; la barre garde deux cas
      (le declencheur montre la vitesse, aucun multiplicateur ne l'encombre).
- [x] 3-G Gate : tsc VERT ; vitest `src/features/match-replay` COMPLET vert ; vitest
      `src/routes` vert ; ESLint 0 sur tous les fichiers touches ; cliquet
      `placementFamily.guard` VERT ; `grep -r "#[0-9a-fA-F]\{6\}"` muet sur les nouveaux
      fichiers ; `grep -ri archivo apps/web/src` muet ; aucune classe Tailwind couleur.
      MESURE : `npm run typecheck` exit 0 ; vitest 102 fichiers / **1561 tests verts, 0 echec**
      (baseline : 1502 tests dont 1 rouge) ; ESLint 0 erreur sur `src/features/match-replay` et
      `src/routes` (2 warnings, tous deux PRE-EXISTANTS et verifies sur la base : `exhaustive-deps`
      objectiveObjects du canvas, `only-export-components` de ReplayFeedName) ; cliquet 6/6 ;
      greps hex / archivo / classes couleur muets.
      CORRECTION DE GATE : les lots 1 et 2 avaient ete valides avec `npx tsc -p apps/web --noEmit`,
      qui ne compile RIEN (`tsconfig.json` n'a que des `references`, `files: []`). La commande
      juste est `npm run typecheck` (= `tsc -b`), et c'est elle qui a revele les deux defauts
      ci-dessous. Le gate 3 la passe sur l'ensemble, lots 1 et 2 compris.

## LOT 4 — Cloture

- [x] 4-1 Journal : entree datee dans `.ai/V7.5/replay2d/thought_log_replay.md` (decisions,
      chiffres de tests, deviations assumees) ; statuts du present plan tous poses.
      Ecrite AU FIL DE L'EAU (une section par lot, dans le commit du lot) plutot qu'en une fois
      a la cloture : le contrat `plan-execution` demande une entree a CHAQUE cloture d'etape.
- [x] 4-2 Report REGISTRE : `.ai/V7.5/REGISTRE_REPORTS.md` — « Medias du rejeu phase 2 »
      REDUIT PAR L'AMENDEMENT a la seule DONNEE : endpoint { id, kind, replayMs, durationMs,
      thumbUrl, url, label } par match/joueur (recalage t0/displayClockMs comme le fil) et
      passage de la prop `media` au canvas. Le rendu — piste, placement, lightbox, cles — est
      LIVRE. Condition de reprise : decision utilisateur apres livraison planche 2a.
      Noter aussi : les fichiers originaux de la session Claude Design ONT ete recuperes en
      cours de lot 2 et portes (cf. amendement) ; le dossier `planche2a_impl/` reste dans
      `.ai/` comme reference du gate visuel.
- [x] 4-3 Commit final (docs) sur wt/lecteur. PAS de merge, PAS de push.
      DEUX entrees ajoutees au registre (medias phase 2 reduits a la donnee ; gate visuel non
      passe). Aucun merge, aucun push : la branche `wt/lecteur` attend l'autorisation
      utilisateur apres le gate visuel.

## Ce que ce plan NE fait PAS (perimetre ferme — amende le 2026-08-28)

- Pas d'endpoint medias, et aucune donnee de media n'est branchee : la piste et la lightbox
  sont livrees, la LISTE reste vide (`EMPTY_MEDIA`). La lightbox et ses trois cles NE SONT
  PLUS hors perimetre (amendement, point 2).
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
- (revue R1, 2026-08-28, lot 5 — CONSIGNE, NON CORRIGE) **LES RACCOURCIS SE TAISENT QUAND LA
  FRISE GARDE LE FOCUS.** `isTypingTarget` (useReplayShortcuts) traite tout `INPUT` comme un
  champ de saisie — le curseur de la frise en est un (`type=range`). Apres un clic sur la frise,
  elle garde le focus : Espace ne fait plus rien, et ←/→ deplacent le curseur natif d'UNE image
  au lieu de sauter 10 s. Ce n'est pas un bug de la garde (elle protege une vraie recherche
  textuelle), c'est une DECISION PRODUIT non tranchee : faut-il exclure `type=range` de la garde,
  ou rendre le focus au conteneur apres un scrub ? Condition de reprise : verdict utilisateur au
  gate visuel, quand le comportement se constate a l'ecran.
- (revue R1, 2026-08-28, lot 5 — CONSIGNE, NON CORRIGE) `useReplaySettings` rend un OBJET NEUF a
  chaque rendu (pre-existant, anterieur a ce chantier). Consequence directe : le `useMemo` de
  `useReplayDrawer` ne retient rien — ses dependances changent d'identite en permanence. Le memo
  reste (gratuit, et il devient vrai le jour ou la source amont se stabilise) et son commentaire
  le DIT desormais, plutot que de promettre une memoisation qui n'a pas lieu. Le retirer ou
  memoiser la source amont : meme lot, hors perimetre ici.
- (executeur, 2026-08-28, lot 3) **LE GATE `tsc` DU PLAN NE VERIFIE RIEN.**
  `npx tsc -p apps/web --noEmit` compile ZERO fichier : `apps/web/tsconfig.json` ne porte que
  des `references` avec `files: []`. La commande d'autorite est `npm run typecheck` (`tsc -b`),
  celle de `make check-types`. A corriger dans les plans qui recopient cette ligne.
- (executeur, 2026-08-28, lot 3) **COLLISION DE CASSE WINDOWS.** Le handoff nommait la logique
  `replayTimelineTracks.ts` et le composant `ReplayTimelineTracks.tsx` : sur un systeme de
  fichiers insensible a la casse, TypeScript refuse les deux dans le meme programme (TS1149) et
  l'import du composant resolvait vers le module de logique. La logique est renommee
  `replayTimelineTracksLogic.ts` — patron du depot (killFeedLogic, victoryLogic, scoreBannerLogic).
  A garder en tete pour tout futur couple logique/composant homonyme.
- (executeur, 2026-08-28, lot 1) `npm install` (lot 0) a REECRIT `apps/web/package-lock.json`
  en retirant 30 blocs `libc` (metadonnees de plateforme des binaires optionnels rollup/esbuild) :
  derive de la version locale de npm, sans rapport avec le lot, et potentiellement nuisible sur
  la CI Linux (ces champs filtrent glibc/musl). Le fichier a ete RESTAURE et n'entre dans aucun
  commit du chantier. A traiter separement si l'equipe veut aligner la version de npm.
- (executeur, 2026-08-28, lot 0) `ReplayKillFeed` porte six props que le plan ne cite pas
  (`nowMs`, `playWindow`, `scoreboard`, `xuidMeta`, `marks`, `colorOf`) : elles restent, la
  remontee ne concerne que l'assemblage du fil (kills/medals/t0Ms/doc -> `entries`).

## Journal des revues (2026-08-28, pilote)

- R1 (contexte frais, gates rejoues par le relecteur) : 2 P1 + 6 P2 recevables, 20
  conditions tiennent. Triage pilote : C1-C6 -> lot 5 (`95f53d6b1`) ; 3 constats consignes
  sans correction (Decouvertes).
- R2 (2e contexte frais, perimetre = lot 5 seul) : les 6 corrections TIENNENT — mutation
  reduceFeed (swap tueur/victime) : 4 tests rouges puis restauration propre ; 0 constat
  recevable ; P0+P1 : 2 -> 0. Gates : vitest match-replay 100 fichiers / 1546 / 0 echec ;
  typecheck cache purge exit 0 ; arbre propre.
- Boucle close (2 rondes, decroissance stricte). Reste : gate visuel utilisateur +
  autorisation de merge/push (CI de branche au push).
