/**
 * ReplayCaptureButtons — CE QUI SORT DU REJEU : la capture d'image, l'enregistrement ou l'export
 * vidéo, et le panneau d'export. Partagé par les barres de lecture.
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
 *  - LE CARTOUCHE EST L'ANCRE DU PANNEAU (`relative`) : le panneau se cale en `bottom-full right-0`
 *    sur le premier ancêtre positionné ; monté ailleurs, il se calerait sur la carte entière du
 *    rejeu, où `overflow-hidden` le découperait.
 */
import { useState } from 'react'

import { ReplayExportDialog, isExportBusy } from '../export/ReplayExportDialog'
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import type { ReplayCapture } from '../export/useReplayCapture'
import { CameraIcon, ExportIcon, ExportSpinner } from './ReplayTransportIcons'

/** Ce que le bouton d'export affiche pendant un export : le pourcentage dès qu'il a un sens, une ellipse avant. */
function exportBadge(exporter: ReplayCapture['videoExport']): string {
  if (!exporter || exporter.state.phase !== 'encode') return '…'
  return `${Math.round(exporter.state.pct)} %`
}

export function ReplayCaptureButtons({ capture, locale }: { capture: ReplayCapture; locale: ReplayLocale }) {
  const [exportOpen, setExportOpen] = useState(false)
  const busy = capture.videoExport ? isExportBusy(capture.videoExport.state) : false
  const t = REPLAY_TEXT[locale]
  return (
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
      {capture.videoExport?.supported && (
        <button
          type="button"
          onClick={() => setExportOpen((v) => !v)}
          aria-expanded={exportOpen}
          aria-label={t.exportVideo}
          title={busy ? t.exportRunningHint : t.exportHint}
          className="inline-flex h-7 cursor-pointer items-center justify-center gap-1.5 rounded-full bg-primary px-2.5 text-[12.5px] font-semibold tracking-[0.03em] text-primary-foreground transition-colors hover:bg-primary/90"
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
      {exportOpen && capture.videoExport && (
        <ReplayExportDialog exporter={capture.videoExport} locale={locale} onClose={() => setExportOpen(false)} />
      )}
    </div>
  )
}
