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
import { isExportBusy } from '../export/ReplayExportDialog'
import { REPLAY_TEXT } from '../i18n/i18n'
import { SKIP_SECONDS } from '../layers/replayCanvasConfig'
import { ReplaySoundControls } from '../sound/ReplaySoundControls'
import { ReplayCaptureButtons } from './ReplayCaptureButtons'
import { ReplayFrameButtons } from './ReplayFrameButtons'
import { ReplaySpeedMenu } from './ReplaySpeedMenu'
import { ReplayTimelineTracks } from './ReplayTimelineTracks'
import { AutoPlayIcon, PauseIcon, PlayIcon, RestartIcon } from './ReplayTransportIcons'
import type { ReplayTransportCommands } from './replayTransportCommands'

/** Les commandes de la toile, moins le zoom : cette barre n'en porte pas (la croix flottante le fait). */
type ReplayTransportProps = Omit<ReplayTransportCommands, 'zoom'>

export function ReplayTransport({
  playing, onTogglePlay, onRestart, onSeekBy, clockRef, timeline, autoPlay, onToggleAutoPlay,
  speed, onSetSpeed, sound, capture, locale, fullscreen, frameControl,
}: ReplayTransportProps) {
  const busy = capture.videoExport ? isExportBusy(capture.videoExport.state) : false
  const t = REPLAY_TEXT[locale]
  return (
    // LE SOCLE SOMBRE sépare le lecteur de la carte ; il va bord à bord, chaque pixel qu'il ne prend pas
    // devient du terrain (`useReplayViewport.freeSpaceFor` déduit le chrome par soustraction).
    <div className="mt-3 bg-background/40 px-3 pb-2.5 pt-3">
      {/* LA FRISE AUSSI est neutralisée : son curseur écrit `frameRef`. `pointer-events-none` bloque le
          geste, `aria-hidden` la retire des technologies d'assistance le temps du calcul. */}
      <div className={busy ? 'pointer-events-none opacity-40' : undefined} aria-hidden={busy}>
        <ReplayTimelineTracks {...timeline} clockRef={clockRef} />
      </div>

      <div className="mt-1.5 flex items-center gap-3.5">
        <div className="flex flex-1 items-center gap-2">
          <button
            type="button"
            onClick={onToggleAutoPlay}
            role="switch"
            aria-checked={autoPlay}
            aria-label={t.autoPlay}
            title={`${t.autoPlay} — ${t.autoPlayHint}`}
            className={`inline-flex h-[30px] cursor-pointer items-center gap-1.5 rounded-full px-2 text-[10.5px] font-medium transition-colors ${
              autoPlay
                ? 'bg-secondary text-secondary-foreground hover:bg-secondary/80'
                : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground'
            }`}
          >
            <AutoPlayIcon on={autoPlay} />
          </button>
          <ReplaySoundControls sound={sound} locale={locale} />
          <ReplaySpeedMenu speed={speed} onSetSpeed={onSetSpeed} locale={locale} />
        </div>

        <div className="flex shrink-0 items-center gap-1.5">
          <RoundButton onClick={onRestart} label={`${t.restart} (R)`} ghost disabled={busy}>
            <RestartIcon />
          </RoundButton>
          <RoundButton onClick={() => onSeekBy(-SKIP_SECONDS)} label={`${t.skipBackFmt(SKIP_SECONDS)} (←)`} bordered disabled={busy}>
            <span className="text-[10.5px] font-medium tabular-nums">−{SKIP_SECONDS}</span>
          </RoundButton>
          {/* LE SEUL BOUTON PLEIN DE LA BARRE. Le nom accessible dit ce que le CLIC va faire,
              l'icône dit où l'on en est. */}
          <button
            type="button"
            onClick={onTogglePlay}
            disabled={busy}
            aria-label={playing ? t.pause : t.play}
            title={`${playing ? t.pause : t.play} (${t.keySpace})`}
            className={`inline-flex h-10 w-10 items-center justify-center rounded-full bg-primary text-primary-foreground transition-colors ${
              busy ? 'cursor-not-allowed opacity-40' : 'cursor-pointer hover:bg-primary/90 active:bg-primary/80'
            }`}
          >
            {playing ? <PauseIcon /> : <PlayIcon />}
          </button>
          <RoundButton onClick={() => onSeekBy(SKIP_SECONDS)} label={`${t.skipForwardFmt(SKIP_SECONDS)} (→)`} bordered disabled={busy}>
            <span className="text-[10.5px] font-medium tabular-nums">+{SKIP_SECONDS}</span>
          </RoundButton>
        </div>

        <div className="flex flex-1 items-center justify-end">
          <ReplayCaptureButtons capture={capture} locale={locale} />
          <ReplayFrameButtons frameControl={frameControl} fullscreen={fullscreen} locale={locale} />
        </div>
      </div>
    </div>
  )
}

/** Un bouton rond de la barre : fantôme (recommencer) ou bordé (les sauts). */
function RoundButton({
  onClick, label, children, ghost, bordered, disabled,
}: {
  onClick: () => void
  label: string
  children: React.ReactNode
  ghost?: boolean
  bordered?: boolean
  /** Neutralise pendant un export : la lecture et l'export se disputeraient l'image courante. */
  disabled?: boolean
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      aria-label={label}
      title={label}
      className={`inline-flex h-[30px] w-[30px] items-center justify-center rounded-full transition-colors ${
        disabled ? 'cursor-not-allowed opacity-40' : 'cursor-pointer'
      } ${bordered ? 'border border-input hover:bg-accent' : ''} ${
        ghost ? 'text-muted-foreground hover:bg-accent hover:text-accent-foreground' : ''
      }`}
    >
      {children}
    </button>
  )
}
