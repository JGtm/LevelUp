/**
 * ValueAxis.tsx — L'AXE GRADUÉ SOUS CHAQUE FORME, et son intitulé.
 *
 * POURQUOI IL EST OBLIGATOIRE (correction de l'artefact 2ec1b8eb) : une barre
 * sans axe ne se lit pas — deux barres de même longueur dans deux formes
 * différentes passent pour deux valeurs égales. Chaque forme du bloc pose donc
 * son axe, aligné SUR LA PISTE et pas sur la carte : l'axe doit border ce qu'il
 * mesure, jamais la colonne des libellés.
 *
 * L'ALIGNEMENT SE DÉDUIT DE LA GRILLE DE LA FORME. Les formes n'ont pas toutes
 * le même nombre de colonnes (la jauge double en a quatre, la piste deux) : la
 * colonne élastique (`1fr`) EST la piste, et l'axe se place exactement là. Une
 * position écrite en dur a donné, sur la jauge double, cinq graduations tassées
 * dans la colonne des dénominateurs — le défaut est visible sur la première
 * capture du lot et c'est ce qui a motivé ce calcul.
 *
 * Cinq graduations (0, ¼, ½, ¾, fin), la première alignée à gauche, la dernière
 * à droite, les autres centrées sur leur trait.
 */
const TICKS = 4
/** La gouttière des grilles de formes (`gap-3` de Tailwind). */
const GAP = '0.75rem'

export interface ValueAxisProps {
  /** Le `grid-template-columns` de la forme : l'axe s'aligne sur sa piste. */
  columns: string
  min: number
  max: number
  /** L'intitulé sous l'axe (ce que mesure la graduation). */
  title?: string
  /** Le format d'une graduation. Défaut : le nombre tel quel. */
  format?: (value: number) => string
}

/** L'index de la colonne élastique — la piste. Défaut : la deuxième colonne. */
function trackColumnIndex(columns: string[]): number {
  const index = columns.findIndex((c) => c.endsWith('fr'))
  return index >= 0 ? index : Math.min(1, columns.length - 1)
}

/** Le décalage à gauche de l'intitulé : les colonnes qui précèdent la piste. */
function titleOffset(columns: string[], trackIndex: number): string {
  const parts = columns.slice(0, trackIndex)
  return `calc(${[...parts, ...parts.map(() => GAP), GAP].join(' + ')})`
}

export function ValueAxis({ columns, min, max, title, format }: ValueAxisProps) {
  const parts = columns.trim().split(/\s+/)
  const trackIndex = trackColumnIndex(parts)
  const fmt = format ?? ((v: number) => String(Math.round(v)))
  return (
    <>
      <div className="grid gap-3" style={{ gridTemplateColumns: columns }}>
        {parts.slice(0, trackIndex).map((_, i) => (
          <div key={`lead-${i}`} aria-hidden="true" />
        ))}
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
                <span
                  className={`absolute top-[3px] whitespace-nowrap ${align}`}
                  style={{ left: `${ratio * 100}%` }}
                >
                  {fmt(value)}
                </span>
              </span>
            )
          })}
        </div>
        {parts.slice(trackIndex + 1).map((_, i) => (
          <div key={`trail-${i}`} aria-hidden="true" />
        ))}
      </div>
      {title != null && (
        <div
          className="mt-0.5 text-3xs text-muted-foreground"
          style={{ marginLeft: titleOffset(parts, trackIndex) }}
        >
          {title}
        </div>
      )}
    </>
  )
}
