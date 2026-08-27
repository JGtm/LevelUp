/**
 * ReplayTransport — LA BARRE DE LECTURE du rejeu. Refonte validée le 2026-08-28 (planche 2a),
 * après « là ça fait basic de fou » : la barre disait onze commandes du même poids, sans
 * hiérarchie ni matière.
 *
 * CE QUI CHANGE, ET POURQUOI :
 *
 *  - LA FRISE DEVIENT UNE TABLE DE MONTAGE (ReplayTimelineTracks) : au-dessus du curseur, tes
 *    éliminations et tes morts, celles de tes alliés, la DOMINANCE, les MÉDIAS. On voit la
 *    forme du match avant de l'avoir lu. Le curseur reste le même `input[type=range]` piloté
 *    par la boucle de dessin — seul son habillage change.
 *  - UN SEUL BOUTON PLEIN, la lecture, en rond de 44 px. Tout le reste est fantôme ou bordé :
 *    la hiérarchie se lit avant les icônes.
 *  - LES SAUTS ±10 s encadrent la lecture (demande du 2026-08-27) : c'est le geste le plus
 *    fréquent d'un rejeu qu'on analyse, et il n'existait qu'en tirant la frise à la main.
 *  - LA VITESSE PASSE EN MENU (ReplaySpeedMenu) : quatre boutons pour un réglage occupaient la
 *    place de quatre commandes.
 *  - LES SORTIES SONT MISES EN AVANT (demande du 2026-08-28) : « Image » et « REC » deviennent
 *    deux pastilles NOMMÉES dans leur propre cartouche, REC en `destructive`. Ce sont les
 *    seules commandes colorées de la barre — elles écrivent un fichier, les autres non.
 *  - L'HORLOGE PERD LE MONOSPACE et gagne la taille : `tabular-nums` suffit à la stabiliser au
 *    défilement, et elle devient l'ancre visuelle de la barre au lieu d'un détail gris.
 *  - LES RACCOURCIS CLAVIER (useReplayShortcuts) sont câblés par le canvas ; les libellés des
 *    boutons les rappellent entre parenthèses.
 *
 * L'ÉTAT NE VIT TOUJOURS PAS ICI : la LECTURE vit dans `useReplayPlayback` ; l'image courante,
 * l'horloge, la vitesse et le son restent au canvas. Les icônes gardent leur libellé en
 * aria-label/title — un symbole sans nom serait une régression d'accessibilité.
 */
import type { ComponentProps, RefObject } from 'react'

import { REPLAY_TEXT, type ReplayLocale } from './i18n'
import { SKIP_SECONDS } from './replayCanvasConfig'
import { ReplaySoundControls } from './ReplaySoundControls'
import { ReplaySpeedMenu } from './ReplaySpeedMenu'
import { ReplayTimelineTracks } from './ReplayTimelineTracks'
import { SlidersIcon } from './SlidersIcon'
import type { ReplayCapture } from './useReplayCapture'
import type { ReplaySound } from './useReplaySound'

interface ReplayTransportProps {
  playing: boolean
  onTogglePlay: () => void
  onRestart: () => void
  /** Le saut ±10 s, en secondes signées (cf. useReplayPlayback.seekBy). */
  onSeekBy: (seconds: number) => void
  /** L'horloge est écrite par la boucle de dessin (textContent), pas par React. */
  clockRef: RefObject<HTMLSpanElement | null>
  /** La frise et ses pistes, en UN objet (même motif que `leadMarks` avant elle). */
  timeline: ComponentProps<typeof ReplayTimelineTracks>
  speed: number
  onSetSpeed: (speed: number) => void
  sound: ReplaySound
  capture: ReplayCapture
  locale: ReplayLocale
  settingsOpen: boolean
  onToggleSettings: () => void
  settingsButtonRef: RefObject<HTMLButtonElement | null>
}

export function ReplayTransport({
  playing, onTogglePlay, onRestart, onSeekBy, clockRef, timeline,
  speed, onSetSpeed, sound, capture, locale,
  settingsOpen, onToggleSettings, settingsButtonRef,
}: ReplayTransportProps) {
  const t = REPLAY_TEXT[locale]
  return (
    // LE SOCLE SOMBRE sépare le lecteur de la carte : sans lui, la barre flottait sur le même
    // fond que le canvas et paraissait posée là par accident.
    <div className="mt-3 rounded-md bg-background/40 px-3.5 pb-3.5 pt-3">
      <ReplayTimelineTracks {...timeline} />

      <div className="mt-5 flex items-center gap-3.5">
        {/* L'HORLOGE D'ABORD, en grand : « où j'en suis » avant « ce que je peux faire ». */}
        <span
          ref={clockRef}
          className="min-w-[6.5rem] text-[21px] font-medium tabular-nums tracking-[-0.02em]"
          aria-label={t.time}
        />

        <div className="flex items-center gap-1.5">
          <RoundButton onClick={onRestart} label={`${t.restart} (R)`} ghost>
            <RestartIcon />
          </RoundButton>
          <RoundButton onClick={() => onSeekBy(-SKIP_SECONDS)} label={`${t.skipBackFmt(SKIP_SECONDS)} (←)`} bordered>
            <span className="text-[10.5px] font-medium tabular-nums">−{SKIP_SECONDS}</span>
          </RoundButton>
          {/* LE SEUL BOUTON PLEIN DE LA BARRE. Le nom accessible dit ce que le CLIC va faire,
              l'icône dit où l'on en est (patron d'état inchangé). */}
          <button
            type="button"
            onClick={onTogglePlay}
            aria-label={playing ? t.pause : t.play}
            title={`${playing ? t.pause : t.play} (Espace)`}
            className="inline-flex h-11 w-11 cursor-pointer items-center justify-center rounded-full bg-primary text-primary-foreground transition-colors hover:bg-primary/90 active:bg-primary/80"
          >
            {playing ? <PauseIcon /> : <PlayIcon />}
          </button>
          <RoundButton onClick={() => onSeekBy(SKIP_SECONDS)} label={`${t.skipForwardFmt(SKIP_SECONDS)} (→)`} bordered>
            <span className="text-[10.5px] font-medium tabular-nums">+{SKIP_SECONDS}</span>
          </RoundButton>
        </div>

        {/* LES RÉGLAGES DE LECTURE : le son, puis la vitesse. Des réglages, pas des commandes. */}
        <div className="flex items-center gap-2">
          <ReplaySoundControls sound={sound} locale={locale} />
          <ReplaySpeedMenu speed={speed} onSetSpeed={onSetSpeed} locale={locale} />
        </div>

        <div className="flex-1" />

        {/* CE QUI SORT DU REJEU, dans son propre cartouche et NOMMÉ. Le bouton d'enregistrement
            ne se rend pas quand le navigateur ne sait pas filmer une toile (décision 7) : une
            commande grisée laisserait croire à une panne réparable. */}
        <div className="flex items-center gap-1.5 rounded-full border border-border bg-muted/40 p-1">
          <button
            type="button"
            onClick={capture.captureImage}
            aria-label={t.captureImage}
            title={t.captureImage}
            className="inline-flex h-8 cursor-pointer items-center gap-1.5 rounded-full bg-secondary px-3 text-[12.5px] font-medium text-secondary-foreground transition-colors hover:bg-secondary/80"
          >
            <CameraIcon />
            {t.captureImageShort}
          </button>
          {capture.recordingSupported && (
            <button
              type="button"
              onClick={capture.toggleRecording}
              aria-pressed={capture.recording}
              aria-label={capture.recording ? t.stopRecording : t.recordVideo}
              title={t.recordHint}
              className={`inline-flex h-8 cursor-pointer items-center gap-1.5 rounded-full px-3.5 text-[12.5px] font-semibold tracking-[0.03em] transition-colors ${
                capture.recording
                  ? 'bg-destructive text-destructive-foreground hover:bg-destructive/90'
                  : 'bg-destructive/90 text-destructive-foreground hover:bg-destructive'
              }`}
            >
              {capture.recording ? (
                <span className="h-2.5 w-2.5 rounded-[2px] bg-current" />
              ) : (
                <span className="h-2.5 w-2.5 rounded-full bg-current" />
              )}
              {capture.recording ? t.stopRecordingShort : t.recordVideoShort}
            </button>
          )}
        </div>

        {/* LES RÉGLAGES FERMENT LA BARRE, tout à droite — là où tous les lecteurs les mettent. */}
        <button
          ref={settingsButtonRef}
          type="button"
          onClick={onToggleSettings}
          aria-expanded={settingsOpen}
          aria-label={t.settingsButton}
          title={t.settingsButton}
          className={`inline-flex h-9 w-9 cursor-pointer items-center justify-center rounded-full transition-colors ${
            settingsOpen ? 'bg-accent text-accent-foreground' : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground'
          }`}
        >
          <SlidersIcon />
        </button>
      </div>
    </div>
  )
}

/**
 * Un bouton rond de la barre : fantôme (recommencer) ou bordé (les sauts). La forme ronde vient
 * de la planche 1a — validée le 2026-08-28 contre les rectangles de la première passe.
 */
function RoundButton({
  onClick, label, children, ghost, bordered,
}: {
  onClick: () => void
  label: string
  children: React.ReactNode
  ghost?: boolean
  bordered?: boolean
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-label={label}
      title={label}
      className={`inline-flex h-[34px] w-[34px] cursor-pointer items-center justify-center rounded-full transition-colors ${
        bordered ? 'border border-input hover:bg-accent' : ''
      } ${ghost ? 'text-muted-foreground hover:bg-accent hover:text-accent-foreground' : ''}`}
    >
      {children}
    </button>
  )
}

/** Icône lecture : le triangle. Décorative — le libellé vit sur le bouton. */
function PlayIcon() {
  return (
    <svg viewBox="0 0 16 16" className="h-[17px] w-[17px]" fill="currentColor" aria-hidden="true">
      <path d="M4.5 2.8a.8.8 0 0 1 1.2-.7l8 5.2a.8.8 0 0 1 0 1.4l-8 5.2a.8.8 0 0 1-1.2-.7z" />
    </svg>
  )
}

/** Icône pause : les deux barres. */
function PauseIcon() {
  return (
    <svg viewBox="0 0 16 16" className="h-[17px] w-[17px]" fill="currentColor" aria-hidden="true">
      <rect x="3.5" y="2.5" width="3.4" height="11" rx="1" />
      <rect x="9.1" y="2.5" width="3.4" height="11" rx="1" />
    </svg>
  )
}

/** Icône appareil photo : le boîtier et son objectif. */
function CameraIcon() {
  return (
    <svg
      viewBox="0 0 16 16"
      className="h-[15px] w-[15px]"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.4"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d="M1.8 5.2h2.6l1.1-1.7h4.9l1.1 1.7h2.7v7.3H1.8z" />
      <circle cx="8" cy="8.9" r="2.4" />
    </svg>
  )
}

/** Icône recommencer : la flèche qui reboucle vers le début. */
function RestartIcon() {
  return (
    <svg
      viewBox="0 0 16 16"
      className="h-[15px] w-[15px]"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.6"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d="M2.5 8a5.5 5.5 0 1 0 1.6-3.9" />
      <path d="M4.1 1.6v2.9H7" />
    </svg>
  )
}
