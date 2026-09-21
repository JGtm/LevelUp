/**
 * StackedTrack — LA piste épaisse horizontale : une barre de 32 px segmentée, chaque segment
 * portant son écriture quand il a la place de la tenir, et toujours son infobulle.
 *
 * HISSÉE DANS `components/charts/` LE 2026-09-21 (lot K du plan d'ajustements supplémentaires
 * pré-v7.5). Elle est née la veille dans `features/explorer/` sous le nom
 * `ExplorerStackedTrack` pour deux blocs de l'encart cible (« Part des assistances »,
 * « Répartition des résultats ») ; la vue « Part de chaque équipe » de la page match (5.A) est
 * la troisième posée dessus, et une feature n'importe pas d'une autre
 * (`tools/lint-cross-feature-imports.mjs`). Le composant ne connaît ni assistance, ni équipe,
 * ni famille d'équipement : il reçoit des largeurs déjà calculées et des textes déjà écrits.
 *
 * LA PISTE N'EST PAS FORCÉMENT PLEINE. Les segments somment à 100 % pour une répartition
 * (V/N/D), et à moins que cela quand l'appelant les mesure sur une ÉCHELLE COMMUNE à plusieurs
 * pistes (5.A : la borne est le plus gros total, pas le total de la ligne) — le fond `bg-muted`
 * qui reste est alors la part de l'échelle non employée, et c'est exactement ce qui se lit.
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
  /**
   * Nom accessible du segment (`role="img"`), et le focus clavier qui va avec. ABSENT = le
   * segment n'est qu'une forme, son texte se lit au survol — le cas des pistes de l'encart
   * cible, dont la légende voisine porte déjà les valeurs.
   */
  ariaLabel?: string
}

export function StackedTrack({
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
              role={seg.ariaLabel ? 'img' : undefined}
              aria-label={seg.ariaLabel}
              tabIndex={seg.ariaLabel ? 0 : undefined}
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
