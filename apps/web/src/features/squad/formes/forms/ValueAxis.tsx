/**
 * ValueAxis.tsx — L'AXE GRADUÉ SOUS CHAQUE FORME, et son intitulé.
 *
 * POURQUOI IL EST OBLIGATOIRE (correction de l'artefact 2ec1b8eb) : une barre
 * sans axe ne se lit pas — deux barres de même longueur dans deux formes
 * différentes passent pour deux valeurs égales. Chaque forme du bloc pose donc
 * son axe, aligné SUR LA PISTE et pas sur la carte : l'axe doit border ce qu'il
 * mesure, jamais la colonne des libellés.
 *
 * Cinq graduations (0, ¼, ½, ¾, fin), la première alignée à gauche, la dernière
 * à droite, les autres centrées sur leur trait.
 */
const TICKS = 4

export interface ValueAxisProps {
  /** Le `grid-template-columns` de la forme : l'axe s'aligne sur ses colonnes. */
  columns: string
  min: number
  max: number
  /** L'intitulé sous l'axe (ce que mesure la graduation). */
  title?: string
  /** Le format d'une graduation. Défaut : le nombre tel quel. */
  format?: (value: number) => string
}

export function ValueAxis({ columns, min, max, title, format }: ValueAxisProps) {
  const labelColumn = columns.split(' ')[0]
  const trailing = columns.split(' ').length > 2
  const fmt = format ?? ((v: number) => String(Math.round(v)))
  return (
    <>
      <div className="grid gap-3" style={{ gridTemplateColumns: columns }}>
        <div aria-hidden="true" />
        <div className="relative mt-1 h-[17px] border-t border-border text-3xs tabular-nums text-muted-foreground">
          {Array.from({ length: TICKS + 1 }, (_, i) => {
            const ratio = i / TICKS
            const value = min + (max - min) * ratio
            const align = i === 0 ? '' : i === TICKS ? '-translate-x-full' : '-translate-x-1/2'
            return (
              <span key={ratio} className="contents">
                <i
                  className="absolute top-0 h-[3px] w-px bg-border"
                  style={{ left: `${ratio * 100}%` }}
                  aria-hidden="true"
                />
                <span className={`absolute top-[3px] ${align}`} style={{ left: `${ratio * 100}%` }}>
                  {fmt(value)}
                </span>
              </span>
            )
          })}
        </div>
        {trailing && <div aria-hidden="true" />}
      </div>
      {title != null && (
        <div
          className="mt-0.5 text-3xs text-muted-foreground"
          style={{ marginLeft: `calc(${labelColumn} + 0.75rem)` }}
        >
          {title}
        </div>
      )}
    </>
  )
}
