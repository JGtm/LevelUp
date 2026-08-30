# Implémentation — barre de lecture du rejeu (planche 2a)

Cible : `apps/web/src/features/match-replay/`. Je n'ai pas d'accès en écriture au dépôt monté ;
les fichiers de ce dossier sont prêts à y être déposés tels quels (chemin de destination en tête
de chaque fichier).

## 1. Fichiers à déposer

| Fichier | Action |
| --- | --- |
| `replayTimelineTracks.ts` | nouveau — logique pure (pistes, dominance, médias) |
| `ReplayTimelineTracks.tsx` | nouveau — frise + 4 pistes + curseur habillé |
| `ReplayMediaLightbox.tsx` | nouveau — média en grand, rejeu en pause |
| `ReplaySpeedMenu.tsx` | nouveau — vitesse en menu |
| `useReplayShortcuts.ts` | nouveau — clavier |
| `ReplayTransport.tsx` | **remplace** l'existant |
| `ReplaySoundControls.tsx` | **remplace** l'existant |

## 2. Patch `useReplayPlayback.ts`

Trois ajouts. Dans `ReplayPlayback`, déclarer les deux commandes :

```ts
  /** Saut de ±N SECONDES sur l'axe réel (converti en images par `baseFps`). */
  seekBy: (seconds: number) => void
  /** Image par image, borné à la fenêtre de gameplay. */
  stepFrames: (frames: number) => void
```

Dans le corps du hook, après `onScrub` :

```ts
  /**
   * SEEK ABSOLU BORNÉ À LA FENÊTRE DE GAMEPLAY : les sauts ne peuvent pas sortir du match pour
   * aller chercher le countdown d'avant-match ou la queue du film — la même règle que le scrub.
   * Le tracé est appelé ici parce qu'un saut EN PAUSE doit se voir immédiatement ; en lecture,
   * la boucle repeindra de toute façon à l'image suivante.
   */
  const seekTo = (frame: number) => {
    const next = Math.min(endFrame, Math.max(startFrame, frame))
    frameRef.current = next
    if (sliderRef.current) sliderRef.current.value = String(Math.round(next))
    soundTick(frameToMs(next, doc))
    draw()
  }

  const seekBy = (seconds: number) => seekTo(frameRef.current + seconds * baseFps)
  const stepFrames = (frames: number) => seekTo(Math.round(frameRef.current) + frames)
```

…et les retourner : `return { playing, startFrame, endFrame, sliderRef, togglePlay, restart, onScrub, seekBy, stepFrames }`.

Enfin, LE REMPLISSAGE DE LA FRISE. Dans la boucle `step`, juste après la ligne qui écrit
`sliderRef.current.value`, ajouter la variable CSS que le curseur habillé consomme (aucun rendu
React, la boucle écrit directement dans le style) :

```ts
      if (sliderRef.current) {
        sliderRef.current.value = String(Math.round(next))
        const pct = endFrame > startFrame ? ((next - startFrame) / (endFrame - startFrame)) * 100 : 0
        sliderRef.current.style.setProperty('--played', `${pct}%`)
      }
```

La même écriture est à faire dans `seekTo` et dans `rewind` (sinon un saut en pause laisse le
remplissage en arrière) — extraire un petit `writeCursor(next)` local est le plus propre.

## 3. Clés i18n à ajouter

Dans `i18nContract.ts` (`interface ReplayText`) :

```ts
  /** Sauts de la barre : le libellé porte la durée, jamais un « avance » muet. */
  skipBackFmt: (seconds: number) => string
  skipForwardFmt: (seconds: number) => string
  /** Le menu de vitesse : la valeur normale, et la note du son coupé au-delà de 2×. */
  speedNormal: string
  speedMuted: string
  /** Les pastilles nommées des sorties (la barre n'a plus la place d'un libellé long). */
  captureImageShort: string
  recordVideoShort: string
  stopRecordingShort: string
  /** Les pistes de la frise. */
  trackYou: string
  trackAllies: string
  trackDominance: string
  dominanceOfFmt: (team: string) => string
  /** La piste des médias et sa lightbox. */
  mediaTrack: string
  mediaEmpty: string
  mediaOpen: string
  mediaClose: string
  mediaPausedHint: string
```

Table `fr` :

```ts
    skipBackFmt: (s) => `Reculer de ${s} s`,
    skipForwardFmt: (s) => `Avancer de ${s} s`,
    speedNormal: 'normal',
    speedMuted: 'son coupé',
    captureImageShort: 'Image',
    recordVideoShort: 'REC',
    stopRecordingShort: 'Arrêter',
    trackYou: 'Toi',
    trackAllies: 'Alliés',
    trackDominance: 'Dominance',
    dominanceOfFmt: (team) => `${team} mène`,
    mediaTrack: 'Médias',
    mediaEmpty: 'Aucun média sur ce match',
    mediaOpen: 'Ouvrir le média',
    mediaClose: 'Fermer',
    mediaPausedHint: 'Rejeu en pause',
```

Table `en` :

```ts
    skipBackFmt: (s) => `Back ${s} s`,
    skipForwardFmt: (s) => `Forward ${s} s`,
    speedNormal: 'normal',
    speedMuted: 'sound off',
    captureImageShort: 'Image',
    recordVideoShort: 'REC',
    stopRecordingShort: 'Stop',
    trackYou: 'You',
    trackAllies: 'Allies',
    trackDominance: 'Dominance',
    dominanceOfFmt: (team) => `${team} leads`,
    mediaTrack: 'Media',
    mediaEmpty: 'No media on this match',
    mediaOpen: 'Open media',
    mediaClose: 'Close',
    mediaPausedHint: 'Replay paused',
```

## 4. Câblage dans `ReplayCanvas.tsx`

`ReplayLeadMarks` n'est plus monté par la barre : ses changements de meneur alimentent
maintenant la piste **Dominance** (`buildDominance`), qui montre les DURÉES au lieu des instants.
Le composant peut rester dans le dépôt (il est testé) ou être supprimé avec son test.

```tsx
const { playing, startFrame, endFrame, sliderRef, togglePlay, restart, onScrub, seekBy, stepFrames } =
  useReplayPlayback({ /* inchangé */ })

const scale = useMemo(() => trackScale(playWindow, doc.frameCount), [playWindow, doc.frameCount])
const tracks = useMemo(
  () => buildEventTracks(trackKills, trackDeaths, marks, frameIntervalMs, scale,
        (ms) => formatClockMMSS(displayClockMs(ms, playWindow))),
  [trackKills, trackDeaths, marks, frameIntervalMs, scale, playWindow],
)
const dominance = useMemo(() => buildDominance(leadMarks.changes, scale), [leadMarks.changes, scale])
const placedMedia = useMemo(() => placeMedia(media, frameIntervalMs, scale), [media, frameIntervalMs, scale])

useReplayShortcuts({
  togglePlay, seekBy, stepFrames, restart,
  toggleSound: sound.toggle,
  skipSeconds: SKIP_SECONDS,
  enabled: renderWidth > 0,
})

<ReplayTransport
  playing={playing}
  onTogglePlay={togglePlay}
  onRestart={restart}
  onSeekBy={seekBy}
  clockRef={clockRef}
  timeline={{
    sliderRef, minFrame: startFrame, maxFrame: endFrame, onScrub,
    own: tracks.own, allies: tracks.allies,
    dominance, allyOf: leadMarks.allyOf, labelOf: leadMarks.labelOf,
    media: placedMedia,
    playing, onRequestPause: togglePlay,
    startClock: '0:00', midClock: midClock, endClock: endClock,
    locale,
  }}
  speed={multiplier}
  onSetSpeed={setMultiplier}
  sound={sound}
  capture={capture}
  locale={locale}
  settingsOpen={settingsOpen}
  onToggleSettings={() => setSettingsOpen((v) => !v)}
  settingsButtonRef={settingsButtonRef}
/>
```

`trackKills` / `trackDeaths` : la page du rejeu construit déjà le fil (`buildFeedEntries` /
`alignFeedByOrigin` dans `ReplayKillFeed`). Le plus économique est de remonter le fil aligné
d'un niveau (page) et de le passer au canvas réduit aux trois champs de `TrackKill` /
`TrackDeath` (`key`, `replayMs`, `xuid`) — pas de second alignement, pas de donnée dupliquée.
`marks` est la `ReadonlyMap<string, PlayerMarkKind>` déjà construite par `buildPlayerMarks`.

`midClock` / `endClock` : `formatClockMMSS(displayClockMs(...))` sur le milieu et la fin de la
fenêtre, comme le bandeau.

## 5. Médias — ce qui reste à faire côté données

Le design est complet, la donnée n'existe pas encore. Ce qui manque :

- un endpoint qui rende, pour un match et un joueur, la liste `{ id, kind, replayMs, durationMs, thumbUrl, url, label }` (le type `ReplayMediaItem` est déjà écrit) ;
- l'instant `replayMs` doit être sur l'axe DU REJEU, pas sur l'horloge du jeu : le recalage `t0Ms` / `displayClockMs` est le même que pour le fil ;
- pour un clip, `thumbUrl` sert à toute la bande (une seule image répétée). Le jour où le backend publie plusieurs vignettes par clip, remplacer `thumbUrl` par `thumbUrls: string[]` dans `clipFrameCount`/la bande — c'est le seul endroit à toucher.

D'ici là la piste s'affiche vide avec « Aucun média sur ce match » (tu voulais la barre toujours
visible ; c'est l'exception assumée à la règle « pas de commande sans quoi commander »).

## 6. Typographie

La planche 2a est en **Archivo** ; le dépôt n'impose pas de famille (`globals.css` n'en déclare
aucune). Si tu veux la même typo à l'écran, c'est un ajout global, hors de cette barre :

```css
/* globals.css */
@theme { --font-sans: Archivo, ui-sans-serif, system-ui, sans-serif; }
```

Sans ça, la barre reste correcte : l'horloge et les chiffres passent de `font-mono` à
`tabular-nums`, ce qui suffit à les stabiliser au défilement.

## 7. Tests à prévoir (cliquets du dépôt)

- `replayTimelineTracks.test.ts` — pistes : un acteur sans marque n'est sur aucune piste ; les morts ne vont que sur `own` ; une marque hors fenêtre est écartée ; `buildDominance` sans changement rend `[]` ; `placeMedia` donne à un clip la largeur de sa durée.
- `ReplayTransport.test.tsx` — le test existant vérifie les libellés accessibles : garder les mêmes clés, ajouter `skipBackFmt`/`skipForwardFmt` et les pastilles nommées.
- `useReplayShortcuts.test.ts` — Espace depuis un `<input>` ne met pas en pause ; Cmd+R n'est pas capté.
