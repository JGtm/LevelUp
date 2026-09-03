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
 *  - UN SEUL BOUTON PLEIN, la lecture, en rond de 40 px (44 jusqu'au 2026-09-02). Tout le reste
 *    est fantôme ou bordé : la hiérarchie se lit avant les icônes.
 *  - LES SAUTS ±10 s encadrent la lecture (demande du 2026-08-27) : c'est le geste le plus
 *    fréquent d'un rejeu qu'on analyse, et il n'existait qu'en tirant la frise à la main.
 *  - LA VITESSE PASSE EN MENU (ReplaySpeedMenu) : quatre boutons pour un réglage occupaient la
 *    place de quatre commandes.
 *  - LES SORTIES SONT MISES EN AVANT (demande du 2026-08-28) : « Image » et « REC » vivent dans
 *    leur propre cartouche, REC en `destructive`. Ce sont les seules commandes colorées de la
 *    barre — elles écrivent un fichier, les autres non. Leurs LIBELLÉS sont tombés le
 *    2026-09-02 (« vire les labels Image et exporter ») : le cartouche et la couleur portent
 *    déjà le sens, `aria-label` et `title` portent le nom. UNE EXCEPTION, et elle n'est pas
 *    négociable : le bouton d'export SE RÉ-ÉLARGIT pour afficher sa progression pendant le
 *    calcul. Sans elle, on réintroduit le défaut corrigé le 2026-08-28 — un clic malheureux
 *    faisait perdre À LA FOIS le retour et le bouton « Annuler », pendant plusieurs minutes.
 *  - L'HORLOGE A QUITTÉ CETTE BARRE le 2026-09-02 : elle vit dans la frise, sous le point qui
 *    avance (cf. `ReplayTimelineTracks`). Elle y répétait, vingt pixels plus bas et dans une
 *    autre taille, ce que la frise montrait déjà.
 *  - LES RACCOURCIS CLAVIER (useReplayShortcuts) sont câblés par le canvas ; les libellés des
 *    boutons les rappellent entre parenthèses.
 *
 * L'ÉTAT NE VIT TOUJOURS PAS ICI : la LECTURE vit dans `useReplayPlayback` ; l'image courante,
 * l'horloge, la vitesse et le son restent au canvas. Les icônes gardent leur libellé en
 * aria-label/title — un symbole sans nom serait une régression d'accessibilité.
 */
import type { ComponentProps, RefObject } from 'react'

import { useState } from 'react'

import { ReplayExportDialog, isExportBusy } from './ReplayExportDialog'
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
  /**
   * LA LECTURE AUTOMATIQUE, arrivée du tiroir le 2026-09-02. Elle ne commande PAS le lecteur
   * ouvert : elle décide de son état de départ, lu une fois au montage (useReplayPlayback).
   */
  autoPlay: boolean
  onToggleAutoPlay: () => void
  /** La frise et ses pistes, en UN objet (même motif que `leadMarks` avant elle). */
  timeline: Omit<ComponentProps<typeof ReplayTimelineTracks>, 'clockRef'>
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
  playing, onTogglePlay, onRestart, onSeekBy, clockRef, timeline, autoPlay, onToggleAutoPlay,
  speed, onSetSpeed, sound, capture, locale,
  settingsOpen, onToggleSettings, settingsButtonRef,
}: ReplayTransportProps) {
  // LE DIALOGUE D'EXPORT s'ouvre depuis la barre et se pose au-dessus d'elle. Son ouverture
  // vit ICI et pas dans le canvas : c'est le bouton qui la commande, et le canvas est à son
  // plafond de taille.
  const [exportOpen, setExportOpen] = useState(false)
  // PENDANT UN EXPORT, LES COMMANDES DE LECTURE SONT NEUTRALISEES. La boucle d'export ecrit
  // `frameRef` image par image ; la boucle de lecture, les sauts et le glisse de la frise
  // ecrivent le MEME ref. Les laisser vivants permettait a l'utilisateur de corrompre son
  // propre clip d'un clic, sans que rien ne le signale.
  const busy = capture.videoExport ? isExportBusy(capture.videoExport.state) : false
  const t = REPLAY_TEXT[locale]
  return (
    // LE SOCLE SOMBRE sépare le lecteur de la carte : sans lui, la barre flottait sur le même
    // fond que le canvas et paraissait posée là par accident.
    //
    // IL VA BORD À BORD DEPUIS LE 2026-09-02 (« le lecteur a du padding ou des marges en bas, à
    // gauche et à droite, c'est à virer »). Il vivait dans le `p-3` de la carte : 12 px de carte
    // l'encadraient, et ses coins arrondis flottaient au milieu. Il est monté d'un cran dans
    // `ReplayCanvas`, ses arrondis sont tombés, et son propre rembourrage passe de 14 à 12/10 —
    // assez pour que les commandes ne touchent pas le bord, plus assez pour faire une marge.
    //
    // CE QUE ÇA RAPPORTE : chaque pixel rendu ici devient du TERRAIN, sans que personne le
    // recalcule. `useReplayViewport.freeSpaceFor` déduit le chrome par soustraction — ce que la
    // barre ne prend plus, la carte le prend.
    <div className="mt-3 bg-background/40 px-3 pb-2.5 pt-3">
      {/* LA FRISE AUSSI : son curseur ecrit `frameRef`. `pointer-events-none` bloque le geste,
          `aria-hidden` la retire des technologies d'assistance le temps du calcul. */}
      <div className={busy ? 'pointer-events-none opacity-40' : undefined} aria-hidden={busy}>
        {/* L'HORLOGE EST DANS LA FRISE, sous le point qui avance (cf. ReplayTimelineTracks) :
            elle a quitté cette rangée, où elle répétait vingt pixels plus bas ce que la frise
            montrait déjà. C'est la référence qui descend, pas une chaîne — le texte change
            soixante fois par seconde. */}
        <ReplayTimelineTracks {...timeline} clockRef={clockRef} />
      </div>

      {/* L'ÉCART AVEC LA FRISE TOMBE DE 20 À 6 px (2026-09-02) : les commandes appartiennent à
          la frise qu'elles pilotent, un blanc de 20 px en faisait deux blocs étrangers. */}
      {/* TROIS ZONES, ET LE TRANSPORT AU MILIEU DU BLOC (demande utilisateur du 2026-09-02).
          Les commandes s'alignaient à gauche, ce qui les collait au bord et laissait un grand
          vide à droite. Le centrage tient à `flex-1` sur les DEUX flancs : à base nulle et
          croissance égale, ils prennent la même largeur quoi qu'ils contiennent — le groupe du
          milieu tombe donc au centre exact, et il y reste quand une commande latérale apparaît
          ou disparaît (l'enregistrement, absent sans WebCodecs). Un `justify-between` à trois
          enfants aurait décentré le transport dès que les flancs auraient différé. */}
      <div className="mt-1.5 flex items-center gap-3.5">
        {/* À GAUCHE : ce qui se règle. La lecture automatique y a rejoint le son et la vitesse
            (« on a carrément la place pour un bouton comme YouTube ») — elle a QUITTÉ le
            tiroir, elle n'y est plus en double : deux commandes pour un même réglage invitent
            à croire qu'elles diffèrent. */}
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
              l'icône dit où l'on en est (patron d'état inchangé). */}
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

        {/* À DROITE : ce qui SORT du rejeu. `flex-1 justify-end` fait le pendant exact du flanc
            gauche — c'est ce couple qui tient le transport au centre. */}
        <div className="flex flex-1 items-center justify-end">

        {/* CE QUI SORT DU REJEU, dans son propre cartouche et NOMMÉ. Le bouton d'enregistrement
            ne se rend pas quand le navigateur ne sait pas filmer une toile (décision 7) : une
            commande grisée laisserait croire à une panne réparable. */}
        {/* `relative` : C'EST L'ANCRE DU PANNEAU D'EXPORT, et elle est écrite ici plutôt que
            subie. Sans elle, le panneau se calait sur le premier ancêtre positionné — la carte
            entière du rejeu, cinq cents lignes plus loin — et recouvrait la frise. */}
        <div className="relative flex items-center gap-1.5 rounded-full border border-border bg-muted/40 p-1">
          <button
            type="button"
            onClick={capture.captureImage}
            aria-label={t.captureImage}
            title={t.captureImage}
            className="inline-flex h-7 cursor-pointer items-center justify-center rounded-full bg-secondary px-2.5 text-[12.5px] font-medium text-secondary-foreground transition-colors hover:bg-secondary/80"
          >
            <CameraIcon />
          </button>
          {/* L'EXPORT HORS TEMPS RÉEL prend la place de l'enregistrement quand le navigateur
              sait encoder (décision D5) : deux boutons qui font presque la même chose seraient
              un piège à clic. Le repli reste offert là où WebCodecs manque. */}
          {capture.videoExport?.supported && (
            <button
              type="button"
              onClick={() => setExportOpen((v) => !v)}
              aria-expanded={exportOpen}
              aria-label={t.exportVideo}
              title={busy ? t.exportRunningHint : t.exportHint}
              className="inline-flex h-7 cursor-pointer items-center justify-center gap-1.5 rounded-full bg-primary px-2.5 text-[12.5px] font-semibold tracking-[0.03em] text-primary-foreground transition-colors hover:bg-primary/90"
            >
              {/* LE BOUTON PORTE LA PROGRESSION quand le panneau est refermé : sans cela, un
                  clic malheureux faisait perdre À LA FOIS le retour et le bouton « Annuler »,
                  pendant un calcul de plusieurs minutes. */}
              {busy ? <ExportSpinner /> : <ExportIcon />}
              {busy ? exportBadge(capture.videoExport) : null}
            </button>
          )}
          {!capture.videoExport?.supported && capture.recordingSupported && (
            <button
              type="button"
              onClick={capture.toggleRecording}
              aria-pressed={capture.recording}
              aria-label={capture.recording ? t.stopRecording : t.recordVideo}
              title={t.recordHint}
              className={`inline-flex h-7 cursor-pointer items-center justify-center rounded-full px-2.5 text-[12.5px] font-semibold tracking-[0.03em] transition-colors ${
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
            </button>
          )}
          {/* LE PANNEAU EST MONTÉ DANS LE CARTOUCHE, et c'est la seule position qui marche : son
              `bottom-full right-0` se résout sur le premier ancêtre POSITIONNÉ, et le cartouche
              est le seul à porter `relative`. Monté en frère (ce qu'il était jusqu'au
              2026-08-28), il se calait sur la carte entière du rejeu et `bottom-full` le
              plaçait au-dessus de son bord supérieur, où `overflow-hidden` le découpait : il
              était dans le DOM, et invisible à l'écran. */}
          {exportOpen && capture.videoExport && (
            <ReplayExportDialog
              exporter={capture.videoExport}
              locale={locale}
              onClose={() => setExportOpen(false)}
            />
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
    </div>
  )
}

/**
 * L'icône de LECTURE AUTOMATIQUE, sur le modèle des lecteurs vidéo : un rail et sa pastille,
 * pleins quand c'est armé, en creux sinon. Elle ne porte pas de libellé — le bouton est un
 * `role="switch"` avec son nom accessible, et la barre n'a plus la place d'un mot de plus.
 */
function AutoPlayIcon({ on }: { on: boolean }) {
  return (
    <svg viewBox="0 0 28 16" className="h-[15px] w-[26px]" aria-hidden="true">
      <rect
        x="1"
        y="3"
        width="26"
        height="10"
        rx="5"
        fill={on ? 'currentColor' : 'none'}
        stroke="currentColor"
        strokeWidth="1.5"
        opacity={on ? 0.35 : 0.6}
      />
      {/* La pastille glisse d'un bord à l'autre : l'état se lit sans couleur, donc sans
          dépendre d'une distinction que tout le monde ne fait pas. */}
      <circle cx={on ? 19 : 9} cy="8" r="3.2" fill="currentColor" />
    </svg>
  )
}

/**
 * Un bouton rond de la barre : fantôme (recommencer) ou bordé (les sauts). La forme ronde vient
 * de la planche 1a — validée le 2026-08-28 contre les rectangles de la première passe.
 */
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

/**
 * Icône d'export : la flèche qui SORT du plateau. Volontairement différente du disque rouge
 * de l'enregistrement — les deux commandes ne se remplacent pas dans l'esprit de qui regarde,
 * même si le code, lui, remplace l'une par l'autre.
 */
function ExportIcon() {
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
      <path d="M8 10.2V2.4" />
      <path d="M5.2 5.2 8 2.4l2.8 2.8" />
      <path d="M2.8 10.4v2.2a1 1 0 0 0 1 1h8.4a1 1 0 0 0 1-1v-2.2" />
    </svg>
  )
}

/**
 * exportBadge — ce que le bouton affiche pendant un export : le pourcentage dès qu'il veut dire
 * quelque chose, et une ellipse pendant la préparation (où rien n'est encore encodé).
 */
function exportBadge(exporter: ReplayCapture['videoExport']): string {
  if (!exporter || exporter.state.phase !== 'encode') return '…'
  return `${Math.round(exporter.state.pct)} %`
}

/** L'anneau qui tourne, le temps du calcul. Même gabarit que l'icône qu'il remplace. */
function ExportSpinner() {
  return (
    <svg
      viewBox="0 0 16 16"
      className="h-[15px] w-[15px] animate-spin"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
      strokeLinecap="round"
      aria-hidden="true"
    >
      <circle cx="8" cy="8" r="6" className="opacity-30" />
      <path d="M14 8a6 6 0 0 0-6-6" />
    </svg>
  )
}
