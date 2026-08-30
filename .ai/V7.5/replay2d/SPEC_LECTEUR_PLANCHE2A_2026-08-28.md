# Spec — barre de lecture du rejeu (planche 2a, Claude Design)

> Copie de travail du memo d'implementation remis par la session Claude Design le
> 2026-08-28 (source : C:\Users\Guillaume\Downloads\LECTEUR.md), annotee des DECISIONS
> UTILISATEUR prises au lancement du chantier. Les fichiers de code produits par la session
> design n'ont PAS pu etre recuperes (workspace distant en lecture seule) : cette spec + la
> grammaire visuelle du depot font foi, le gate visuel utilisateur tranche en fin de lot.

## Decisions utilisateur (2026-08-28, prevalent sur le memo)

1. **Typographie** : GARDER la police de l'app. Aucun ajout d'Archivo, aucun `--font-sans`,
   aucune `font-family` nulle part. Seul retenu du §6 : dans la barre, `font-mono` cede la
   place a `tabular-nums` pour stabiliser les chiffres.
2. **Kill feed** : le fil (colonne de droite) est CONSERVE tel quel a l'ecran. La
   proposition « remonter le fil d'un niveau » ne porte que sur la DONNEE : le fil aligne
   (`buildFeedEntries`) se calcule une fois dans la page et sert le fil ET les pistes —
   pas de second alignement, pas de duplication.
3. **Medias** : la DONNEE est reportee en phase 2 (endpoint + branchement). Phase 1 : la
   piste Medias s'affiche avec son etat vide honnete (« Aucun media sur ce match »),
   `ReplayMediaLightbox` et les cles i18n d'ouverture NE SONT PAS livrees (zero code mort).
4. **ReplayLeadMarks** : SUPPRIME avec son test (la piste Dominance montre les durees, les
   marques d'instant deviennent redondantes). Le hook `useLeadMarks` RESTE (il alimente la
   dominance en changements de meneur + libelles).

## 1. Fichiers (perimetre revise phase 1)

| Fichier | Action |
| --- | --- |
| `replayTimelineTracks.ts` | nouveau — logique pure (pistes, dominance, medias) |
| `ReplayTimelineTracks.tsx` | nouveau — frise + 4 pistes + curseur habille |
| `ReplaySpeedMenu.tsx` | nouveau — vitesse en menu |
| `useReplayShortcuts.ts` | nouveau — clavier |
| `useReplayTimeline.ts` | nouveau — assemblage cote canvas (cliquet de taille) |
| `ReplayTransport.tsx` | remplace l'existant |
| `ReplaySoundControls.tsx` | remplace l'existant |
| `ReplayMediaLightbox.tsx` | PHASE 2 (non livre ici) |

## 2. Patch `useReplayPlayback.ts` (contrat exact du memo)

Deux commandes ajoutees a `ReplayPlayback` :

```ts
  /** Saut de +/-N SECONDES sur l'axe reel (converti en images par `baseFps`). */
  seekBy: (seconds: number) => void
  /** Image par image, borne a la fenetre de gameplay. */
  stepFrames: (frames: number) => void
```

Corps du hook, apres `onScrub` :

```ts
  const seekTo = (frame: number) => {
    const next = Math.min(endFrame, Math.max(startFrame, frame))
    frameRef.current = next
    writeCursor(next)
    soundTick(frameToMs(next, doc))
    draw()
  }
  const seekBy = (seconds: number) => seekTo(frameRef.current + seconds * baseFps)
  const stepFrames = (frames: number) => seekTo(Math.round(frameRef.current) + frames)
```

REMPLISSAGE DE LA FRISE : un helper local `writeCursor(next)` ecrit `sliderRef.value` ET la
variable CSS que le curseur habille consomme :

```ts
  const pct = endFrame > startFrame ? ((next - startFrame) / (endFrame - startFrame)) * 100 : 0
  sliderRef.current.style.setProperty('--played', `${pct}%`)
```

`writeCursor` est appele : dans la boucle `step`, dans `seekTo`, dans `rewind`, dans
`onScrub` (le fill doit suivre un drag manuel), et dans l'effet de pose initiale du curseur
(sinon fill desynchronise au montage). Deviations assumees vs memo : (a) `stepFrames` met
d'abord la lecture EN PAUSE (un pas d'image sous rAF serait ecrase en 16 ms — geste de
pause par nature) ; (b) le bornage de seekTo reste celui de la fenetre de gameplay.

## 3. Cles i18n (`i18nContract.ts` interface ReplayText + tables fr/en de `i18n.ts`)

```ts
  skipBackFmt: (seconds: number) => string
  skipForwardFmt: (seconds: number) => string
  speedNormal: string
  speedMuted: string
  captureImageShort: string
  recordVideoShort: string
  stopRecordingShort: string
  trackYou: string
  trackAllies: string
  trackDominance: string
  dominanceOfFmt: (team: string) => string
  mediaTrack: string
  mediaEmpty: string
```

fr : `Reculer de ${s} s` / `Avancer de ${s} s` / 'normal' / 'son coupe au-dela de 2x' /
'Image' / 'REC' / 'Arreter' / 'Toi' / 'Allies' / 'Dominance' / `${team} mene` / 'Medias' /
'Aucun media sur ce match'.
en : `Back ${s} s` / `Forward ${s} s` / 'normal' / 'sound off above 2x' / 'Image' / 'REC' /
'Stop' / 'You' / 'Allies' / 'Dominance' / `${team} leads` / 'Media' / 'No media on this match'.
(Accents FR corrects dans le code — ce fichier de spec est volontairement sans accent.)
RETRAIT : `leadChange` + `leadChangeAtFmt` (seuls consommateurs = ReplayLeadMarks supprime).
PHASE 2 : mediaOpen / mediaClose / mediaPausedHint (avec la lightbox).

## 4. Cablage `ReplayCanvas.tsx` — sous cliquet 691

Le canvas est A SON PLAFOND (placementFamily.guard : 691). Tout l'assemblage part dans
`useReplayTimeline.ts` (13e extraction, patron useLeadMarks/useReplayFx) :

- entrees : { doc, playWindow, feedEntries, marks, leadMarks, locale, playback (sliderRef,
  startFrame, endFrame, onScrub, playing, togglePlay), soundToggle }
- dedans : trackScale, buildEventTracks (reduction des entries aux TrackKill/TrackDeath),
  buildDominance(leadMarks.changes), placeMedia(EMPTY_MEDIA phase 1), mid/endClock
  (`formatClockMMSS(displayClockMs(...))`), et l'appel `useReplayShortcuts` (togglePlay,
  seekBy, stepFrames, restart, toggleSound, skipSeconds, enabled: renderWidth > 0).
- sortie : l'objet `timeline` pret pour ReplayTransport (sliderRef, minFrame, maxFrame,
  onScrub, own, allies, dominance, allyOf, labelOf, media, playing, onRequestPause,
  startClock, midClock, endClock, locale).

`TrackKill` / `TrackDeath` : trois champs (`key`, `replayMs`, `xuid`).
`marks` : la `ReadonlyMap<string, PlayerMarkKind>` deja construite (`buildPlayerMarks`).
Le canvas recoit la prop `feedEntries` de la page, appelle useReplayTimeline + passe
`timeline` a ReplayTransport ; les props sliderRef/minFrame/maxFrame/onScrub/leadMarks
sortent de l'appel ReplayTransport (elles entrent dans `timeline`).

Fil aligne remonte d'un niveau (replay.tsx) :
`const feedEntries = useMemo(() => buildFeedEntries(kills, medalEvents, t0Ms, data), [...])`
— ReplayKillFeed recoit `entries` (ses props kills/medals/t0Ms/doc tombent), le canvas
recoit `feedEntries`. Comportement du fil INCHANGE a l'ecran.

## 5. Regles produit des pistes

- Piste « Toi » : kills des acteurs marques `me` + LEURS morts. Piste « Allies » : kills
  des acteurs marques `friend` (jamais leurs morts). Un acteur sans marque n'est sur
  AUCUNE piste. Une entree hors fenetre de gameplay est ecartee.
- Encres (tokens uniquement) : kill = `team-ally`, mort = `team-enemy`, camp inconnu de la
  dominance = encre neutre (`--border`), bandes dominance en opacite reduite.
- Dominance : segments DUREE (d'un changement de meneur au suivant, dernier jusqu'a la fin
  de fenetre) ; `buildDominance` sans changement rend `[]` ; infobulle
  `dominanceOfFmt(labelOf(teamId))` + horloge du debut de segment.
- Medias : `placeMedia` donne a un clip la largeur de sa duree (type `ReplayMediaItem`
  { id, kind, replayMs, durationMs, thumbUrl, url, label } ecrit des la phase 1) ; piste
  vide phase 1 avec libelle `mediaEmpty`.
- Alignement horizontal : meme regle THUMB_PX = 16 que feu ReplayLeadMarks (la piste d'un
  input range court de THUMB/2 a largeur - THUMB/2).
- Curseur habille : progression `--played` (token primaire via var CSS theme), reste de
  piste en encre `--border`/`--muted` ; styles pseudo-elements du range dans
  `globals.css` (classe dediee, patron `replay-feed-row`) — jamais de hex dans features/.

## 6. Transport (remplacement)

Ordre lecteur video conserve : [Recommencer] [Lecture/Pause] [-10 s] [+10 s] [horloge
tabular-nums] | frise + 4 pistes (seule zone extensible) | [menu vitesse] [son] [pastilles
sorties : Image / REC-Arreter] [reglages]. `SKIP_SECONDS = 10` (replayCanvasConfig.ts).
Libelles des sauts : skipBackFmt/skipForwardFmt en aria-label/title. Pastilles de sortie :
icone + texte court (captureImageShort/recordVideoShort/stopRecordingShort), aria-label
long conserve (captureImage/recordVideo/stopRecording existants). Regle inchangee : bouton
REC absent si `capture.recordingSupported` est faux ; aria-pressed conserves.

`ReplaySpeedMenu` : bouton compact affichant la vitesse courante (`1x`), menu deroulant
des SPEED_MULTIPLIERS [0.5, 1, 2, 4] (aria-expanded, fermeture Escape + clic dehors +
retour focus — patron du tiroir de reglages) ; note `speedNormal` sur 1x, note `speedMuted`
sur les vitesses ou le son se tait (>2x, cf. soundPlaysAtSpeed).

`ReplaySoundControls` : haut-parleur inchange (available guard, mutedBySpeed estompe +
infobulle) ; le VOLUME passe dans un petit popover ouvert au clic (meme patron que le menu
vitesse) pour rendre la largeur aux pistes ; la valeur reglee survit a la coupure
(regles du 2026-08-25 conservees).

## 7. Raccourcis clavier (`useReplayShortcuts.ts`)

Espace = lecture/pause ; fleches gauche/droite = seekBy(-/+ SKIP_SECONDS) ; `,` / `.` =
stepFrames(-1/+1) ; `m` = son ; `r` SANS modificateur = recommencer. Ignores : tout
raccourci avec Ctrl/Meta/Alt (Cmd+R passe au navigateur), toute cible input / textarea /
select / contentEditable ; inactif tant que `enabled` est faux. preventDefault uniquement
sur les touches traitees (Espace ne defile pas la page).

## 8. Tests attendus (cliquets du depot)

- `replayTimelineTracks.test.ts` : acteur sans marque sur aucune piste ; morts uniquement
  sur `own` ; entree hors fenetre ecartee ; `buildDominance` sans changement = [] ;
  `placeMedia` largeur = duree ; ordre/cles stables.
- `ReplayTimelineTracks.test.tsx` : libelles des 4 pistes ; etat vide medias ; encres par
  tokens (pas de hex).
- `ReplayTransport.test.tsx` (mise a jour) : libelles accessibles conserves + sauts
  skipBackFmt/skipForwardFmt + pastilles nommees.
- `ReplaySpeedMenu.test.tsx` : ouverture/selection/fermeture, note son coupe.
- `useReplayShortcuts.test.ts` : Espace depuis un input ne met pas en pause ; Cmd+R non
  capte ; fleche = saut.
- `useReplayPlayback.test.tsx` (mise a jour) : seekBy borne, stepFrames pause + borne,
  `--played` ecrit par writeCursor.
- `ReplayKillFeed.test.tsx` (mise a jour) : construit `entries` via buildFeedEntries.
