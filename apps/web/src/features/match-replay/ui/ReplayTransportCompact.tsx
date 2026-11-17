/**
 * ReplayTransportCompact — LA BARRE DE LECTURE D'UNE LIGNE du Cinéma et du Plein cadre. Mêmes commandes
 * que la barre complète (`ReplayTransport`), sur 56 px, de gauche à droite :
 *
 *  - la CAMÉRA (emplacement `camera`, voir ci-dessous) et le son ;
 *  - le transport : lecture / pause (rond plein), recommencer, −10, +10, le temps « instant / durée »,
 *    l'interrupteur de lecture automatique ;
 *  - la FRISE (`ReplayTimelineCompact`), qui prend la largeur restante ;
 *  - la vitesse ; le bouton « Plan tourné » quand l'Auto a tourné le plan (`ReplayRotatedButton`) ; le zoom − et +
 *    (la commande de zoom flottante n'est pas rendue dans ces cadres) ;
 *  - capture et export (`ReplayCaptureButtons`), icônes de cadre et plein écran du système
 *    (`ReplayFrameButtons`).
 *
 * Un séparateur vertical sépare les groupes. La barre est PILOTÉE PAR L'OBJET DE COMMANDES de la toile et
 * par lui seul : aucun état de lecture ne vit ici, et les raccourcis clavier sont ceux de la toile.
 *
 * L'EMPLACEMENT DE LA CAMÉRA : `camera` est un nœud que l'appelant fournit (le bouton « Caméra » et son
 * menu). Absent, rien n'est rendu — ni bouton inerte, ni espace — et le son occupe l'extrémité gauche. Présent,
 * il se pose à gauche du son, dans le même groupe. Le nœud est monté par la fonction de rendu de l'emplacement
 * de la toile, qui ne contient aucun hook : un état du menu vit dans le composant qu'il fournit.
 *
 * PENDANT UN EXPORT, LA LECTURE ET LA FRISE SONT NEUTRALISÉES comme dans la barre complète : la boucle
 * d'export écrit l'image courante, et la lecture, les sauts et le glisser de la frise écrivent le MÊME ref.
 * Les icônes de cadre gardent la règle de `ReplayFrameButtons`.
 */
import type { ReactNode } from 'react'

import { isCaptureBusy } from '../export/captureBusy'
import { REPLAY_TEXT } from '../i18n/i18n'
import { SKIP_SECONDS } from '../layers/replayCanvasConfig'
import { COTES_BARRE_COMPACTE as C } from '../model/cotesBarreCompacte'
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

export interface ReplayTransportCompactProps extends ReplayTransportCommands {
  /** Le bouton « Caméra » et son menu, à gauche du son ; absent = rien (cf. l'en-tête). */
  camera?: ReactNode
}

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

export function ReplayTransportCompact({
  playing, onTogglePlay, onRestart, onSeekBy, clockRef, timeline, autoPlay, onToggleAutoPlay,
  speed, onSetSpeed, sound, capture, locale, fullscreen, frameControl, zoom, autoRotation, camera,
}: ReplayTransportCompactProps) {
  const busy = isCaptureBusy(capture)
  const t = REPLAY_TEXT[locale]
  return (
    <div
      className="flex items-center bg-background/40"
      style={{ height: C.barreHauteurPx, columnGap: C.ecartPx, padding: `0 ${C.margeXPx}px` }}
      data-replay-transport="compact"
    >
      {camera}
      <ReplaySoundControls sound={sound} locale={locale} />
      <GroupSeparator />

      <ReplayPlayButton
        playing={playing}
        onTogglePlay={onTogglePlay}
        disabled={busy}
        locale={locale}
        sizeClass="shrink-0"
        style={{ width: C.lecturePx, height: C.lecturePx }}
      />
      <ReplayRoundButton onClick={onRestart} label={`${t.restart} (R)`} ghost disabled={busy}>
        <RestartIcon />
      </ReplayRoundButton>
      <ReplayRoundButton onClick={() => onSeekBy(-SKIP_SECONDS)} label={`${t.skipBackFmt(SKIP_SECONDS)} (←)`} bordered disabled={busy}>
        <span className="text-[10.5px] font-medium tabular-nums">−{SKIP_SECONDS}</span>
      </ReplayRoundButton>
      <ReplayRoundButton onClick={() => onSeekBy(SKIP_SECONDS)} label={`${t.skipForwardFmt(SKIP_SECONDS)} (→)`} bordered disabled={busy}>
        <span className="text-[10.5px] font-medium tabular-nums">+{SKIP_SECONDS}</span>
      </ReplayRoundButton>
      {/* L'instant et la durée, écrits par la boucle de dessin (textContent) : « 3:51 / 7:45 ». */}
      <span
        ref={clockRef}
        className="whitespace-nowrap text-center text-[12.5px] tabular-nums text-muted-foreground"
        style={{ minWidth: C.tempsLargeurMinPx, margin: `0 ${C.tempsMargePx}px` }}
      />
      <ReplayAutoPlayToggle autoPlay={autoPlay} onToggle={onToggleAutoPlay} locale={locale} />

      {/* LA FRISE AUSSI est neutralisée : son curseur écrit `frameRef` (cf. la barre complète). */}
      <div className={`flex min-w-0 flex-1 ${busy ? 'pointer-events-none opacity-40' : ''}`} aria-hidden={busy}>
        <ReplayTimelineCompact timeline={timeline} />
      </div>

      <ReplaySpeedMenu speed={speed} onSetSpeed={onSetSpeed} locale={locale} />
      <GroupSeparator />
      {autoRotation && <ReplayRotatedButton onKeepUpright={autoRotation.onKeepUpright} locale={locale} />}
      <div role="group" aria-label={t.zoomGroup} className="flex shrink-0 items-center" style={{ columnGap: C.ecartPx }}>
        <ReplayRoundButton onClick={zoom.zoomOut} label={t.zoomOut} ghost disabled={!zoom.canZoomOut}>
          <span className="text-[15px] leading-none">−</span>
        </ReplayRoundButton>
        <ReplayRoundButton onClick={zoom.zoomIn} label={t.zoomIn} ghost disabled={!zoom.canZoomIn}>
          <span className="text-[15px] leading-none">+</span>
        </ReplayRoundButton>
      </div>
      <GroupSeparator />
      <ReplayCaptureButtons capture={capture} locale={locale} />
      <ReplayFrameButtons frameControl={frameControl} fullscreen={fullscreen} locale={locale} />
    </div>
  )
}
