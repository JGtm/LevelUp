/**
 * ReplayExportDialog — LE DIALOGUE D'EXPORT : la plage, le son, et ce qui se passe pendant.
 *
 * # POURQUOI UN DIALOGUE, ET PAS UN SIMPLE BOUTON
 *
 * L'enregistrement temps réel n'avait rien à demander : il filmait ce qu'on lui montrait, et
 * s'arrêtait quand on le lui disait. L'export, lui, décide TOUT à l'avance — de quelle image à
 * quelle image, avec ou sans son — et ne rend la main qu'une fois le fichier prêt. Ces choix
 * doivent donc se poser AVANT, et une seule fois.
 *
 * # CE QUE CE COMPOSANT NE DÉCIDE PAS
 *
 * Rien du calcul. Il ne connaît ni encodeur, ni toile, ni piste sonore : il rend un état
 * (`ReplayExport`) et appelle `run`. Le bornage lui-même est délégué — `clampExportBounds`
 * vit dans `replayExportPlan.ts`, avec le reste de l'arithmétique de l'export, parce qu'une
 * borne qui se corrige dans un composant est une borne qu'on ne peut pas tester.
 *
 * # LES DEUX CURSEURS NE PEUVENT PAS SE CROISER
 *
 * Ils sont bornés à la fenêtre de gameplay ET l'un à l'autre : tirer la fin avant le début
 * pousse le début, pas l'inverse d'un cran. Un intervalle vide n'est pas une erreur à
 * signaler, c'est un geste qu'on empêche — il n'y a rien à dire à l'utilisateur.
 *
 * # PENDANT LE CALCUL, LE DIALOGUE NE SE FERME PAS
 *
 * Il devient une barre de progression et un bouton « Annuler ». Fermer laisserait un export
 * tourner sans que rien ne le dise, et c'est exactement ce qu'on ne veut pas d'un traitement
 * qui dure des minutes.
 */
import { useState } from 'react'

import { REPLAY_TEXT, type ReplayLocale } from './i18n'
import { clampExportBounds, type ExportBounds } from './replayExportPlan'
import type { ReplayExport } from './useReplayExport'

interface Props {
  exporter: ReplayExport
  locale: ReplayLocale
  onClose: () => void
}

export function ReplayExportDialog({ exporter, locale, onClose }: Props) {
  const t = REPLAY_TEXT[locale]
  const domain = exporter.defaultBounds()
  const [bounds, setBounds] = useState<ExportBounds>(domain)
  const [withSound, setWithSound] = useState(true)
  const running = exporter.state.running

  // L'HORLOGE DU MATCH, PAS CELLE DU FILM : la même règle que la barre de lecture. Elle est
  // portée par l'export, qui seul tient le document et la fenêtre de gameplay.

  const setStart = (v: number) => setBounds((b) => clampExportBounds({ ...b, startFrame: v }, domain))
  const setEnd = (v: number) => setBounds((b) => clampExportBounds({ ...b, endFrame: v }, domain))

  return (
    <div
      role="dialog"
      aria-label={t.exportDialogTitle}
      className="absolute inset-x-3 bottom-3 z-20 rounded-lg border border-border bg-card p-4 shadow-lg"
    >
      <p className="text-sm font-semibold text-foreground">{t.exportDialogTitle}</p>
      {running ? (
        <ExportProgress exporter={exporter} locale={locale} />
      ) : (
        <>
          <div className="mt-3 flex flex-col gap-2">
            <BoundSlider
              label={t.exportFrom}
              value={bounds.startFrame}
              domain={domain}
              clock={exporter.clockOf(bounds.startFrame)}
              onChange={setStart}
            />
            <BoundSlider
              label={t.exportTo}
              value={bounds.endFrame}
              domain={domain}
              clock={exporter.clockOf(bounds.endFrame)}
              onChange={setEnd}
            />
          </div>
          <label className="mt-3 flex cursor-pointer items-center gap-2 text-[12.5px] text-foreground">
            <input
              type="checkbox"
              checked={withSound}
              onChange={(e) => setWithSound(e.currentTarget.checked)}
              className="size-3.5 cursor-pointer accent-primary"
            />
            {t.exportWithSound}
          </label>
          <p className="mt-2 text-[11.5px] text-muted-foreground">{t.exportLengthFmt(exporter.lengthClock(bounds))}</p>
          <p className="mt-1 text-[11.5px] text-muted-foreground">{t.exportRunningHint}</p>
          <div className="mt-3 flex items-center justify-end gap-2">
            <button
              type="button"
              onClick={onClose}
              className="h-8 cursor-pointer rounded-full px-3 text-[12.5px] font-medium text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground"
            >
              {t.exportClose}
            </button>
            <button
              type="button"
              onClick={() => void exporter.run(bounds, { sound: withSound })}
              className="h-8 cursor-pointer rounded-full bg-primary px-4 text-[12.5px] font-semibold text-primary-foreground transition-colors hover:bg-primary/90"
            >
              {t.exportStart}
            </button>
          </div>
        </>
      )}
    </div>
  )
}

/** Un curseur de borne, avec son horloge de match à droite. */
function BoundSlider({
  label,
  value,
  domain,
  clock,
  onChange,
}: {
  label: string
  value: number
  domain: ExportBounds
  clock: string
  onChange: (v: number) => void
}) {
  return (
    <label className="flex items-center gap-3 text-[12.5px] text-foreground">
      <span className="w-12 shrink-0 text-muted-foreground">{label}</span>
      <input
        type="range"
        min={domain.startFrame}
        max={domain.endFrame}
        value={value}
        onChange={(e) => onChange(Number(e.currentTarget.value))}
        aria-label={label}
        className="h-1.5 min-w-0 flex-1 cursor-pointer appearance-none rounded-full bg-muted accent-primary"
      />
      <span className="w-12 shrink-0 text-right font-mono tabular-nums text-muted-foreground">{clock}</span>
    </label>
  )
}

/**
 * La progression, et le SEUL geste disponible pendant le calcul.
 *
 * La barre est un simple bloc à largeur variable : une balise `<progress>` se style
 * différemment dans chaque navigateur, et l'export n'a pas besoin d'un composant de plus.
 */
function ExportProgress({ exporter, locale }: { exporter: ReplayExport; locale: ReplayLocale }) {
  const t = REPLAY_TEXT[locale]
  const { done, total, pct } = exporter.state
  return (
    <div className="mt-3">
      <div
        role="progressbar"
        aria-valuenow={Math.round(pct)}
        aria-valuemin={0}
        aria-valuemax={100}
        aria-label={t.exportDialogTitle}
        className="h-1.5 w-full overflow-hidden rounded-full bg-muted"
      >
        <div className="h-full rounded-full bg-primary transition-[width]" style={{ width: `${pct}%` }} />
      </div>
      <p className="mt-2 font-mono text-[11.5px] tabular-nums text-muted-foreground">
        {t.exportProgressFmt(done, total)}
      </p>
      <div className="mt-3 flex justify-end">
        <button
          type="button"
          onClick={exporter.cancel}
          className="h-8 cursor-pointer rounded-full bg-secondary px-4 text-[12.5px] font-medium text-secondary-foreground transition-colors hover:bg-secondary/80"
        >
          {t.exportCancel}
        </button>
      </div>
    </div>
  )
}
