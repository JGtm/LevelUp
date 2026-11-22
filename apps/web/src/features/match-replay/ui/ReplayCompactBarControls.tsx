/**
 * ReplayCompactBarControls — LES COMMANDES DE LA BARRE DE LECTURE D'UNE LIGNE, dans l'ordre de la barre (cf.
 * l'en-tête de `ReplayTransportCompact`) : caméra et son ; lecture, recommencer, sauts, temps, lecture
 * automatique ; la frise ; vitesse, « Plan tourné », zoom ; capture, export, cadres. Des éléments frères de la
 * rangée que `ReplayTransportCompact` tient : la rangée les mesure (`useTransportBarLines`), la frise est le seul
 * qui porte `data-replay-bar-timeline`.
 *
 * SUR DEUX LIGNES la frise passe en tête (`order`), seule sur sa ligne, toute la largeur ; les autres commandes
 * gardent leur ordre sur la seconde. Le nœud de la frise est le même dans les deux cas : jamais remonté.
 */
import type { CSSProperties, ReactNode } from 'react'

import { isCaptureBusy } from '../export/captureBusy'
import { TIMELINE_SLOT_ATTR } from '../hooks/useTransportBarLines'
import { REPLAY_TEXT } from '../i18n/i18n'
import { SKIP_SECONDS } from '../layers/replayCanvasConfig'
import { COTES_BARRE_COMPACTE as C } from '../model/cotesBarreCompacte'
import type { BarLines } from '../model/transportBarLogic'
import { ReplaySoundControls } from '../sound/ReplaySoundControls'
import { ReplayAutoPlayToggle } from './ReplayAutoPlayToggle'
import { ReplayCaptureButtons } from './ReplayCaptureButtons'
import { ReplayFrameButtons } from './ReplayFrameButtons'
import { ReplayPlayButton } from './ReplayPlayButton'
import { ReplayRotatedButton } from './ReplayRotatedButton'
import { ReplayRoundButton } from './ReplayRoundButton'
import { ReplaySpeedMenu } from './ReplaySpeedMenu'
import { ReplayTimelineCompact } from './ReplayTimelineCompact'
import { RestartIcon } from './ReplayTransportIcons'
import type { ReplayTransportCommands } from './replayTransportCommands'

/** Le séparateur vertical entre deux groupes de commandes. */
function GroupSeparator() {
  return (
    <span
      className="shrink-0 bg-border"
      style={{ width: 1, height: C.separateurHauteurPx, margin: `0 ${C.separateurMargePx}px` }}
      aria-hidden="true"
    />
  )
}

/** La place de la frise : le reste de la ligne sur une ligne ; sur deux lignes, la première, seule et pleine largeur. */
function timelineSlotStyle(lines: BarLines): CSSProperties | undefined {
  if (lines === 1) return undefined
  return { flex: '0 0 100%', order: -1, height: C.deuxLignesFriseHauteurPx, alignItems: 'center' }
}

export function ReplayCompactBarControls({
  playing, onTogglePlay, onRestart, onSeekBy, clockRef, timeline, autoPlay, onToggleAutoPlay,
  speed, onSetSpeed, sound, capture, locale, fullscreen, frameControl, zoom, autoRotation, camera, lines,
}: ReplayTransportCommands & { camera?: ReactNode; lines: BarLines }) {
  const busy = isCaptureBusy(capture)
  const t = REPLAY_TEXT[locale]
  return (
    <>
      {camera}
      <ReplaySoundControls sound={sound} locale={locale} compact />
      <GroupSeparator />

      <ReplayPlayButton
        playing={playing}
        onTogglePlay={onTogglePlay}
        disabled={busy}
        locale={locale}
        sizeClass="shrink-0"
        style={{ width: C.lecturePx, height: C.lecturePx }}
      />
      <ReplayRoundButton onClick={onRestart} label={`${t.restart} (R)`} compact disabled={busy}>
        <RestartIcon />
      </ReplayRoundButton>
      <ReplayRoundButton onClick={() => onSeekBy(-SKIP_SECONDS)} label={`${t.skipBackFmt(SKIP_SECONDS)} (←)`} compact disabled={busy}>
        <span className="text-[12.5px] font-medium tabular-nums">−{SKIP_SECONDS}</span>
      </ReplayRoundButton>
      <ReplayRoundButton onClick={() => onSeekBy(SKIP_SECONDS)} label={`${t.skipForwardFmt(SKIP_SECONDS)} (→)`} compact disabled={busy}>
        <span className="text-[12.5px] font-medium tabular-nums">+{SKIP_SECONDS}</span>
      </ReplayRoundButton>
      {/* L'instant et la durée, écrits par la boucle de dessin (textContent) : « 3:51 / 7:45 ». */}
      <span
        ref={clockRef}
        className="whitespace-nowrap text-center text-[12.5px] tabular-nums text-muted-foreground"
        style={{ minWidth: C.tempsLargeurMinPx, margin: `0 ${C.tempsMargePx}px` }}
      />
      <ReplayAutoPlayToggle autoPlay={autoPlay} onToggle={onToggleAutoPlay} locale={locale} compact />

      {/* LA FRISE AUSSI est neutralisée : son curseur écrit `frameRef` (cf. la barre complète). */}
      <div
        {...{ [TIMELINE_SLOT_ATTR]: '' }}
        className={`flex min-w-0 flex-1 ${busy ? 'pointer-events-none opacity-40' : ''}`}
        style={timelineSlotStyle(lines)}
        aria-hidden={busy}
      >
        <ReplayTimelineCompact timeline={timeline} />
      </div>

      <ReplaySpeedMenu speed={speed} onSetSpeed={onSetSpeed} locale={locale} compact />
      <GroupSeparator />
      {autoRotation && <ReplayRotatedButton onKeepUpright={autoRotation.onKeepUpright} locale={locale} />}
      <div role="group" aria-label={t.zoomGroup} className="flex shrink-0 items-center" style={{ columnGap: C.ecartPx }}>
        <ReplayRoundButton onClick={zoom.zoomOut} label={t.zoomOut} compact disabled={!zoom.canZoomOut}>
          <span className="text-[15px] leading-none">−</span>
        </ReplayRoundButton>
        <ReplayRoundButton onClick={zoom.zoomIn} label={t.zoomIn} compact disabled={!zoom.canZoomIn}>
          <span className="text-[15px] leading-none">+</span>
        </ReplayRoundButton>
      </div>
      <GroupSeparator />
      <ReplayCaptureButtons capture={capture} locale={locale} compact />
      <ReplayFrameButtons frameControl={frameControl} fullscreen={fullscreen} locale={locale} compact />
    </>
  )
}
