/**
 * ReplayCaptureButtons — CE QUI SORT DU REJEU : la capture d'image, l'enregistrement ou l'export
 * vidéo, et le panneau d'export. Partagé par les barres de lecture.
 *
 * DEUX FORMATS. La barre complète (Standard) les range dans un cartouche arrondi et colore ce qui écrit un fichier ;
 * la barre d'une ligne (`compact`) les pose en boutons ordinaires, au format des autres commandes (`compactButton`),
 * sans cartouche ni fond propre. La progression, le panneau d'export et le repli d'enregistrement sont les mêmes.
 *
 * Le cartouche est NOMMÉ (`aria-label` et `title` portent le nom de chaque bouton, aucun libellé
 * n'est écrit) et colore seulement ce qui écrit un fichier. L'ouverture du panneau d'export vit
 * ici : c'est le bouton d'export qui la commande.
 *
 *  - L'EXPORT HORS TEMPS RÉEL prend la place de l'enregistrement quand le navigateur sait encoder :
 *    deux boutons qui font presque la même chose seraient un piège à clic. L'enregistrement reste
 *    le repli là où l'encodage manque, et n'est rendu que si le navigateur sait filmer une toile
 *    (une commande grisée laisserait croire à une panne réparable).
 *  - LE BOUTON D'EXPORT SE RÉ-ÉLARGIT pour porter la progression pendant le calcul. Sans cette
 *    règle, un clic qui referme le panneau fait perdre à la fois le retour et le bouton
 *    « Annuler », pendant plusieurs minutes.
 *  - LE CONTENEUR EST L'ANCRE DU PANNEAU (`relative`, cartouche ou non) : le panneau se cale en `bottom-full right-0`
 *    sur le premier ancêtre positionné ; monté ailleurs, il se calerait sur la carte entière du
 *    rejeu, où `overflow-hidden` le découperait.
 */
import { useState, type CSSProperties } from 'react'

import { isCaptureBusy } from '../export/captureBusy'
import { ReplayExportDialog } from '../export/ReplayExportDialog'
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import type { ReplayCapture } from '../export/useReplayCapture'
import { COMPACT_BUTTON_CLASS, COMPACT_BUTTON_SQUARE, COMPACT_BUTTON_WIDENABLE } from './compactButton'
import { CameraIcon, ExportIcon, ExportSpinner } from './ReplayTransportIcons'

/** Ce que le bouton d'export affiche pendant un export : le pourcentage dès qu'il a un sens, une ellipse avant. */
function exportBadge(exporter: ReplayCapture['videoExport']): string {
  if (!exporter || exporter.state.phase !== 'encode') return '…'
  return `${Math.round(exporter.state.pct)} %`
}

/** Ce qui change d'un format à l'autre : le conteneur, les classes et les cotes de chaque bouton. */
interface ButtonLook {
  container: string
  image: string
  exportBtn: string
  record: (recording: boolean) => string
  /** Les cotes de la capture et de l'enregistrement (carré), et celles de l'export (qui s'élargit pour son pourcentage). */
  square: CSSProperties | undefined
  widenable: CSSProperties | undefined
}

/** Le cartouche coloré de la barre complète. */
const FULL_LOOK: ButtonLook = {
  container: 'relative flex items-center gap-1.5 rounded-full border border-border bg-muted/40 p-1',
  image: 'inline-flex h-7 cursor-pointer items-center justify-center rounded-full bg-secondary px-2.5 text-[12.5px] font-medium text-secondary-foreground transition-colors hover:bg-secondary/80',
  exportBtn: 'inline-flex h-7 cursor-pointer items-center justify-center gap-1.5 rounded-full bg-primary px-2.5 text-[12.5px] font-semibold tracking-[0.03em] text-primary-foreground transition-colors hover:bg-primary/90',
  record: (recording) =>
    `inline-flex h-7 cursor-pointer items-center justify-center rounded-full px-2.5 text-[12.5px] font-semibold tracking-[0.03em] transition-colors ${
      recording
        ? 'bg-destructive text-destructive-foreground hover:bg-destructive/90'
        : 'bg-destructive/90 text-destructive-foreground hover:bg-destructive'
    }`,
  square: undefined,
  widenable: undefined,
}

/** Deux boutons ordinaires, sans cartouche : l'export s'élargit pour porter son pourcentage, l'enregistrement garde sa pastille rouge. */
const COMPACT_LOOK: ButtonLook = {
  container: 'relative flex items-center',
  image: `${COMPACT_BUTTON_CLASS} cursor-pointer text-muted-foreground`,
  exportBtn: `${COMPACT_BUTTON_CLASS} cursor-pointer gap-1.5 text-[12.5px] font-medium tabular-nums text-muted-foreground`,
  record: () => `${COMPACT_BUTTON_CLASS} cursor-pointer text-destructive`,
  square: COMPACT_BUTTON_SQUARE,
  widenable: COMPACT_BUTTON_WIDENABLE,
}

interface ReplayCaptureButtonsProps {
  capture: ReplayCapture
  locale: ReplayLocale
  /** Le format de la barre d'une ligne : deux boutons ordinaires au lieu du cartouche. */
  compact?: boolean
}

export function ReplayCaptureButtons({ capture, locale, compact = false }: ReplayCaptureButtonsProps) {
  const look = compact ? COMPACT_LOOK : FULL_LOOK
  const [exportOpen, setExportOpen] = useState(false)
  const busy = isCaptureBusy(capture)
  const t = REPLAY_TEXT[locale]
  return (
    <div className={look.container}>
      <button
        type="button"
        onClick={capture.captureImage}
        aria-label={t.captureImage}
        title={t.captureImage}
        style={look.square}
        className={look.image}
      >
        <CameraIcon />
      </button>
      {capture.videoExport?.supported && (
        <button
          type="button"
          onClick={() => setExportOpen((v) => !v)}
          aria-expanded={exportOpen}
          aria-label={t.exportVideo}
          title={busy ? t.exportRunningHint : t.exportHint}
          style={look.widenable}
          className={look.exportBtn}
        >
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
          style={look.square}
          className={look.record(capture.recording)}
        >
          {capture.recording ? (
            <span className="h-2.5 w-2.5 rounded-[2px] bg-current" />
          ) : (
            <span className="h-2.5 w-2.5 rounded-full bg-current" />
          )}
        </button>
      )}
      {exportOpen && capture.videoExport && (
        <ReplayExportDialog exporter={capture.videoExport} locale={locale} onClose={() => setExportOpen(false)} />
      )}
    </div>
  )
}
