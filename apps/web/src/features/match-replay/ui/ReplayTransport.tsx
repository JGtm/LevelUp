/**
 * ReplayTransport — LA BARRE DE LECTURE COMPLÈTE du rejeu : la frise à pistes (ReplayTimelineTracks)
 * et, dessous, trois zones sur une rangée.
 *
 *  - À GAUCHE, ce qui se règle : lecture automatique, son (ReplaySoundControls), vitesse (ReplaySpeedMenu) ;
 *  - AU MILIEU, le transport : recommencer, saut de -10 s, lecture / pause (le seul bouton plein de la
 *    barre), saut de +10 s ;
 *  - À DROITE, ce qui sort du rejeu (ReplayCaptureButtons : capture, enregistrement ou export) puis les
 *    icônes de cadre et de plein écran du système (ReplayFrameButtons).
 *
 * LE TRANSPORT EST AU CENTRE EXACT du bloc : `flex-1` sur les DEUX flancs leur donne la même largeur
 * quoi qu'ils contiennent, donc le groupe du milieu y reste quand une commande latérale apparaît ou
 * disparaît. Un `justify-between` à trois enfants le décentrerait.
 *
 * L'ÉTAT NE VIT PAS ICI : la lecture vit dans `useReplayPlayback` ; l'image courante, l'horloge, la
 * vitesse et le son restent à la toile, qui les remet par `ReplayTransportCommands`. L'horloge est dans la
 * frise, pas dans cette rangée. Les raccourcis clavier sont câblés par la toile ; les noms des boutons
 * les rappellent entre parenthèses. Un bouton en icône garde son nom en `aria-label` et en `title`.
 *
 * PENDANT UN EXPORT, LES COMMANDES DE LECTURE SONT NEUTRALISÉES : la boucle d'export écrit `frameRef`
 * image par image, et la boucle de lecture, les sauts et le glisser de la frise écrivent le MÊME ref.
 */
import type { ReactNode } from 'react'

import { isCaptureBusy } from '../export/captureBusy'
import { REPLAY_TEXT } from '../i18n/i18n'
import { SKIP_SECONDS } from '../layers/replayCanvasConfig'
import { ReplaySoundControls } from '../sound/ReplaySoundControls'
import { ReplayAutoPlayToggle } from './ReplayAutoPlayToggle'
import { ReplayCaptureButtons } from './ReplayCaptureButtons'
import { ReplayFrameButtons } from './ReplayFrameButtons'
import { ReplayPlayButton } from './ReplayPlayButton'
import { ReplayRoundButton } from './ReplayRoundButton'
import { ReplaySpeedMenu } from './ReplaySpeedMenu'
import { ReplayTimelineTracks } from './ReplayTimelineTracks'
import { RestartIcon } from './ReplayTransportIcons'
import type { ReplayTransportCommands } from './replayTransportCommands'

/** Les commandes de la toile, moins le zoom (la croix flottante le fait) et le plan tourné (le retour passe par la section Vue). */
type ReplayTransportProps = Omit<ReplayTransportCommands, 'zoom' | 'autoRotation' | 'thinBar'> & {
  /** La bande des titres de la frise, fournie par le cadre (`canvasSlotsOf`) : elle lit la page, pas la toile. */
  antagonistBand?: ReactNode
}

export function ReplayTransport({
  playing, onTogglePlay, onRestart, onSeekBy, clockRef, timeline, autoPlay, onToggleAutoPlay,
  speed, onSetSpeed, sound, capture, locale, fullscreen, frameControl, antagonistBand,
}: ReplayTransportProps) {
  const busy = isCaptureBusy(capture)
  const t = REPLAY_TEXT[locale]
  return (
    // LE SOCLE SOMBRE sépare le lecteur de la carte ; il va bord à bord, chaque pixel qu'il ne prend pas
    // devient du terrain (`useReplayViewport.freeSpaceFor` déduit le chrome par soustraction).
    <div className="mt-3 bg-background/40 px-3 pb-2.5 pt-3">
      {/* LA FRISE AUSSI est neutralisée : son curseur écrit `frameRef`. `pointer-events-none` bloque le
          geste, `aria-hidden` la retire des technologies d'assistance le temps du calcul. */}
      <div className={busy ? 'pointer-events-none opacity-40' : undefined} aria-hidden={busy}>
        <ReplayTimelineTracks {...timeline} clockRef={clockRef} antagonistBand={antagonistBand} />
      </div>

      <div className="mt-1.5 flex items-center gap-3.5">
        <div className="flex flex-1 items-center gap-2">
          <ReplayAutoPlayToggle autoPlay={autoPlay} onToggle={onToggleAutoPlay} locale={locale} />
          <ReplaySoundControls sound={sound} locale={locale} />
          <ReplaySpeedMenu speed={speed} onSetSpeed={onSetSpeed} locale={locale} />
        </div>

        <div className="flex shrink-0 items-center gap-1.5">
          <ReplayRoundButton onClick={onRestart} label={`${t.restart} (R)`} ghost disabled={busy}>
            <RestartIcon />
          </ReplayRoundButton>
          <ReplayRoundButton onClick={() => onSeekBy(-SKIP_SECONDS)} label={`${t.skipBackFmt(SKIP_SECONDS)} (←)`} bordered disabled={busy}>
            <span className="text-[10.5px] font-medium tabular-nums">−{SKIP_SECONDS}</span>
          </ReplayRoundButton>
          {/* LE SEUL BOUTON PLEIN DE LA BARRE. */}
          <ReplayPlayButton playing={playing} onTogglePlay={onTogglePlay} disabled={busy} locale={locale} sizeClass="h-10 w-10" />
          <ReplayRoundButton onClick={() => onSeekBy(SKIP_SECONDS)} label={`${t.skipForwardFmt(SKIP_SECONDS)} (→)`} bordered disabled={busy}>
            <span className="text-[10.5px] font-medium tabular-nums">+{SKIP_SECONDS}</span>
          </ReplayRoundButton>
        </div>

        <div className="flex flex-1 items-center justify-end">
          <ReplayCaptureButtons capture={capture} locale={locale} />
          <ReplayFrameButtons frameControl={frameControl} fullscreen={fullscreen} locale={locale} />
        </div>
      </div>
    </div>
  )
}
