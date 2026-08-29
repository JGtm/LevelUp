/**
 * ReplayExportDialog — LE PANNEAU D'EXPORT : la plage, le son, et ce qui se passe pendant.
 *
 * # POURQUOI UN PANNEAU, ET PAS UN SIMPLE BOUTON
 *
 * L'enregistrement temps réel n'avait rien à demander : il filmait ce qu'on lui montrait, et
 * s'arrêtait quand on le lui disait. L'export, lui, décide TOUT à l'avance — de quelle image à
 * quelle image, avec ou sans son — et ne rend la main qu'une fois le fichier prêt. Ces choix
 * doivent donc se poser AVANT, et une seule fois.
 *
 * # IL EST ANCRÉ AU BOUTON, ET C'EST UNE DÉCISION
 *
 * Le panneau s'ouvre juste au-dessus du bouton qui l'a appelé, aligné à droite sur lui : le
 * geste et sa réponse au même endroit. Il l'était mal jusqu'au 2026-08-28 — déclaré `absolute`
 * sans ancêtre positionné dans la barre, il se calait par ACCIDENT sur la carte entière du
 * rejeu, cinq cents lignes plus loin, et recouvrait la frise. Le cartouche des commandes de
 * sortie porte désormais `relative` : l'ancrage est écrit, plus subi.
 *
 * # CE QUE CE COMPOSANT NE DÉCIDE PAS
 *
 * Rien du calcul. Il ne connaît ni encodeur, ni toile, ni piste sonore : il lit un état
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
 * # PENDANT LE CALCUL, LE PANNEAU NE SE FERME PAS TOUT SEUL
 *
 * Il devient une progression et un bouton « Annuler ». Et si l'utilisateur le referme quand
 * même par le bouton, c'est CELUI-CI qui porte alors la progression (cf. `ReplayTransport`) :
 * aucun état de la page ne doit perdre à la fois le retour et le moyen d'annuler.
 */
import { useState } from 'react'

import { REPLAY_TEXT, type ReplayLocale } from './i18n'
import type { ReplayText } from './i18nContract'
import { formatClock } from './replayLogic'
import { clampExportBounds, type ExportBounds } from './replayExportPlan'
import type { ReplayExport, ReplayExportState } from './useReplayExport'

interface Props {
  exporter: ReplayExport
  locale: ReplayLocale
  onClose: () => void
}

/** L'export travaille-t-il ? Les deux phases actives, et elles seules. */
export function isExportBusy(state: ReplayExportState): boolean {
  return state.phase === 'prepare' || state.phase === 'encode'
}

export function ReplayExportDialog({ exporter, locale, onClose }: Props) {
  const t = REPLAY_TEXT[locale]
  const domain = exporter.defaultBounds()
  const [bounds, setBounds] = useState<ExportBounds>(domain)
  const [withSound, setWithSound] = useState(true)
  const { state } = exporter

  const setStart = (v: number) => setBounds((b) => clampExportBounds({ ...b, startFrame: v }, domain))
  const setEnd = (v: number) => setBounds((b) => clampExportBounds({ ...b, endFrame: v }, domain))

  return (
    <div
      role="dialog"
      aria-label={t.exportDialogTitle}
      className="absolute bottom-full right-0 z-20 mb-2 w-[22rem] max-w-[calc(100vw-2rem)] rounded-lg border border-border bg-card p-4 shadow-lg"
    >
      <p className="text-sm font-semibold text-foreground">{t.exportDialogTitle}</p>
      {/* L'ÉCHEC SE DIT. Sans cette ligne, une panne d'encodeur faisait disparaître la barre
          sans un mot et sans fichier — impossible de distinguer « ça a échoué » de « il ne
          s'est rien passé ». */}
      {state.phase === 'failed' && (
        <p role="alert" className="mt-2 text-[12.5px] text-destructive">
          {t.exportFailed}
          {state.message ? ` (${state.message})` : ''}
        </p>
      )}
      {/* LA FIN SE DIT AUSSI, avec le nom du fichier : le clip part dans les téléchargements,
          où rien ne rattache un `rejeu-...mp4` au geste qui vient d'être fait. */}
      {state.phase === 'done' && (
        <p role="status" className="mt-2 break-all text-[12.5px] text-foreground">
          {t.exportDoneFmt(state.filename ?? '')}
        </p>
      )}
      {isExportBusy(state) ? (
        <ExportProgress exporter={exporter} locale={locale} />
      ) : (
        <ExportForm
          t={t}
          bounds={bounds}
          domain={domain}
          exporter={exporter}
          sound={{ on: withSound, set: setWithSound }}
          onBound={{ start: setStart, end: setEnd }}
          onClose={onClose}
        />
      )}
    </div>
  )
}

/** Le formulaire : les deux bornes, le son, et les deux gestes. */
function ExportForm({
  t, bounds, domain, exporter, sound, onBound, onClose,
}: {
  t: ReplayText
  bounds: ExportBounds
  domain: ExportBounds
  exporter: ReplayExport
  sound: { on: boolean; set: (v: boolean) => void }
  onBound: { start: (v: number) => void; end: (v: number) => void }
  onClose: () => void
}) {
  return (
    <>
      <div className="mt-3 flex flex-col gap-2">
        <BoundSlider
          label={t.exportFrom}
          value={bounds.startFrame}
          domain={domain}
          clock={exporter.clockOf(bounds.startFrame)}
          onChange={onBound.start}
        />
        <BoundSlider
          label={t.exportTo}
          value={bounds.endFrame}
          domain={domain}
          clock={exporter.clockOf(bounds.endFrame)}
          onChange={onBound.end}
        />
      </div>
      <label className="mt-3 flex cursor-pointer items-center gap-2 text-[12.5px] text-foreground">
        <input
          type="checkbox"
          checked={sound.on}
          onChange={(e) => sound.set(e.currentTarget.checked)}
          className="size-3.5 cursor-pointer accent-primary"
        />
        {t.exportWithSound}
      </label>
      <p className="mt-2 text-[11.5px] text-muted-foreground">
        {t.exportLengthFmt(exporter.lengthClock(bounds))}
      </p>
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
          onClick={() => void exporter.run(bounds, { sound: sound.on })}
          className="h-8 cursor-pointer rounded-full bg-primary px-4 text-[12.5px] font-semibold text-primary-foreground transition-colors hover:bg-primary/90"
        >
          {t.exportStart}
        </button>
      </div>
    </>
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
 * DEUX PHASES QUI NE DISENT PAS LA MÊME CHOSE. En `prepare` — polices, logo, décodage des sons,
 * mixage hors ligne, encodage de la piste — aucun compte d'images n'a de sens, rien n'est encore
 * encodé : la barre est INDÉTERMINÉE et le texte dit ce qu'on fait. C'est la phase qui affichait
 * « Image 0 / 18000 » sans bouger pendant plusieurs secondes, c'est-à-dire exactement l'instant
 * où l'on se demande si ça a planté. En `encode`, le compte et l'estimation deviennent vrais.
 *
 * La barre est un simple bloc à largeur variable : une balise `<progress>` se style
 * différemment dans chaque navigateur, et l'export n'a pas besoin d'un composant de plus.
 */
function ExportProgress({ exporter, locale }: { exporter: ReplayExport; locale: ReplayLocale }) {
  const t = REPLAY_TEXT[locale]
  const { phase, done, total, pct, etaMs } = exporter.state
  const preparing = phase === 'prepare'
  return (
    <div className="mt-3">
      <div
        role="progressbar"
        aria-valuenow={preparing ? undefined : Math.round(pct)}
        aria-valuemin={0}
        aria-valuemax={100}
        aria-label={t.exportDialogTitle}
        className="h-1.5 w-full overflow-hidden rounded-full bg-muted"
      >
        <div
          className={`h-full rounded-full bg-primary ${preparing ? 'animate-pulse' : 'transition-[width]'}`}
          style={{ width: preparing ? '100%' : `${pct}%` }}
        />
      </div>
      <p className="mt-2 text-[11.5px] text-muted-foreground">
        {preparing ? (
          t.exportPreparing
        ) : (
          <>
            <span className="font-mono tabular-nums">{t.exportProgressFmt(done, total)}</span>
            {etaMs !== undefined ? <span> — {t.exportEtaFmt(formatClock(etaMs))}</span> : null}
          </>
        )}
      </p>
      {/* CE QUE L'UTILISATEUR VOIT BOUGER DERRIÈRE LE PANNEAU, et qui l'inquiéterait sinon. Ce
          texte vivait dans le FORMULAIRE jusqu'au 2026-08-28 : sa clé s'appelle pourtant
          `exportRunningHint`, et il ne s'affichait justement pas pendant le calcul. */}
      <p className="mt-1 text-[11.5px] text-muted-foreground">{t.exportRunningHint}</p>
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
