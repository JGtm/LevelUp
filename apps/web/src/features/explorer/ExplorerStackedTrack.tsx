/**
 * ExplorerStackedTrack — LA piste épaisse horizontale de l'encart cible : une barre de
 * 32 px segmentée, chaque segment portant son écriture quand il a la place de la tenir,
 * et toujours son infobulle.
 *
 * Deux blocs la posent (maquette du 2026-09-21, propositions 1.A et 2.A transposées à
 * l'horizontale) : « Part des assistances » (une piste par sens, échelle linéaire commune)
 * et « Répartition des résultats » (une piste empilée V/N/D). D'où le composant partagé
 * plutôt que deux balisages jumeaux.
 *
 * Rendu DOM/CSS, pas ECharts : le wrapper `BarStackedChart` ne porte ni étiquette DANS le
 * segment, ni trait de parité, ni épaisseur de barre imposée, et son rendu canvas
 * n'exposerait ni les comptes ni l'ordre aux tests. C'est le pattern déjà en service sur
 * `_shared/usage/UsageLobbyTrack` — le deuxième mécanisme de l'app, pas un troisième.
 */
import { Tooltip } from '@/components/ui/tooltip'

/** La part de la PISTE qu'un segment doit peser pour porter son écriture. */
const LABEL_MIN_FRACTION_PCT = 9

export interface StackedTrackSegment {
  key: string
  /** Largeur en % de la piste (0..100) — déjà calculée par l'appelant. */
  widthPct: number
  /** Couleur de l'aplat (valeur résolue ou variable CSS d'un jeton). */
  color: string
  /** Écriture posée dans le segment s'il pèse assez ; sinon reprise en légende. */
  label: string
  tooltip: string
}

export function ExplorerStackedTrack({
  segments,
  ariaLabel,
  parityPct,
  parityLabel,
  testId,
}: {
  segments: StackedTrackSegment[]
  ariaLabel: string
  /** Position (0..100 %) du trait de parité vertical ; absent → pas de trait. */
  parityPct?: number | null
  parityLabel?: string
  /** Préfixe des `data-testid` de segment (`<testId>-<key>`). */
  testId: string
}) {
  return (
    <div
      className="relative flex h-8 w-full overflow-hidden rounded-sm bg-muted"
      role="img"
      aria-label={ariaLabel}
      data-testid={testId}
    >
      {segments.map((seg) => (
        // La largeur est portée par l'ITEM (calc en %), jamais par un flex-grow sur le
        // contenu d'un Tooltip : son ancre inline-flex se dimensionnerait au texte et non
        // au compte (piège documenté sur UsageLobbyTrack).
        <div
          key={seg.key}
          className="mr-[2px] h-full last:mr-0"
          style={{ width: `calc(${seg.widthPct}% - 2px)` }}
        >
          <Tooltip content={seg.tooltip} className="h-full w-full">
            {/* `text-white` : une écriture posée SUR un aplat, question de contraste dans
                le segment et non couleur sémantique (même usage que UsageLobbyTrack). */}
            <div
              className="flex h-full w-full cursor-help items-center justify-center overflow-hidden whitespace-nowrap px-1 text-2xs font-semibold text-white"
              style={{ backgroundColor: seg.color }}
              data-testid={`${testId}-${seg.key}`}
            >
              {seg.widthPct >= LABEL_MIN_FRACTION_PCT ? seg.label : ''}
            </div>
          </Tooltip>
        </div>
      ))}
      {parityPct != null && (
        <span
          className="pointer-events-none absolute inset-y-0 w-px bg-foreground"
          style={{ left: `${parityPct}%` }}
          role="img"
          aria-label={parityLabel ?? ''}
          data-testid={`${testId}-parity`}
        />
      )}
    </div>
  )
}
