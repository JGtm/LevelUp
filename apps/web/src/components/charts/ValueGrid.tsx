/**
 * ValueGrid — LA GRILLE DE VALEURS : lignes = individus, colonnes = grandeurs, une échelle par
 * colonne. Le rendu de `valueGridModel.ts`, et rien d'autre.
 *
 * DOM ET CSS, PAS ECHARTS, et c'est un choix mesuré. Ce que la forme exige, c'est que les noms
 * restent alignés d'une colonne à l'autre — un individu doit se lire EN LIGNE. C'est un problème
 * de MISE EN PAGE, que `grid` résout en une déclaration ; cinq `grid` ECharts côte à côte se
 * battraient contre lui (hauteurs de rangée calculées séparément, axes catégoriels redondants),
 * pour un graphe qui n'a ni zoom, ni survol de série, ni animation. Marques à angle droit
 * (`border-radius: 0`), conformément à la préférence dataviz du dépôt.
 *
 * CE COMPOSANT NE CALCULE RIEN : bornes, longueurs, graduations, totaux et filets de groupe
 * viennent tous du modèle. Il ne connaît ni les joueurs, ni les équipes, ni les grandeurs — deux
 * features l'emploient sur deux jeux de données sans le savoir l'une de l'autre.
 *
 * L'INFOBULLE EST AU SURVOL *ET* AU FOCUS CLAVIER : chaque barre est focusable et porte son
 * texte en `aria-label`, donc la valeur reste atteignable sans souris comme au lecteur d'écran.
 * L'ensemble défile HORIZONTALEMENT dans son propre conteneur — jamais le corps de la page.
 */
import { Fragment } from 'react'

import { Tooltip } from '@/components/ui/tooltip'

import type { ValueGridModel } from './valueGridModel'

/** Largeur de la colonne des noms, et largeur mini d'une colonne de valeurs (px). */
const NAME_WIDTH = 152
const COLUMN_MIN = 126
/** Gouttière entre colonnes (px) — reprise dans le calcul de largeur mini de la grille. */
const COLUMN_GAP = 14
/** Largeur réservée au nombre écrit à droite de chaque barre, gouttière comprise (px). */
const VALUE_WIDTH = 38
const VALUE_GAP = 8

interface Props {
  model: ValueGridModel
  /** Libellé de la colonne des noms. Absent = en-tête vide (le mock retenu). */
  rowHeaderLabel?: string
}

export function ValueGrid({ model, rowHeaderLabel }: Props) {
  const { rows, columns, cells, separators } = model
  const gridStyle = {
    gridTemplateColumns: `${NAME_WIDTH}px repeat(${columns.length}, minmax(${COLUMN_MIN}px, 1fr))`,
    minWidth: NAME_WIDTH + columns.length * (COLUMN_MIN + COLUMN_GAP),
    columnGap: COLUMN_GAP,
  }

  return (
    <div className="overflow-x-auto">
      <div className="grid items-center gap-y-[3px]" style={gridStyle}>
        <div className="border-b border-transparent" aria-hidden={!rowHeaderLabel}>
          <span className="text-3xs font-semibold uppercase tracking-wider text-muted-foreground">
            {rowHeaderLabel ?? ''}
          </span>
        </div>
        {columns.map((col) => (
          <div
            key={col.key}
            className="mb-1 flex items-baseline gap-1.5 whitespace-nowrap border-b border-border pb-1.5 text-3xs font-semibold uppercase tracking-wider"
          >
            <span className="truncate">{col.label}</span>
            {col.totalText != null && (
              <span className="ml-auto font-medium normal-case tracking-normal text-muted-foreground tabular-nums">
                {col.totalText}
              </span>
            )}
          </div>
        ))}

        {rows.map((row, r) => (
          <Fragment key={row.key}>
            {separators.includes(r) &&
              Array.from({ length: columns.length + 1 }, (_, i) => (
                <div key={`sep-${row.key}-${i}`} className="my-1 h-px bg-border" />
              ))}
            <div
              className={`flex items-center gap-[7px] overflow-hidden whitespace-nowrap text-xs${row.emphasis ? ' font-semibold' : ''}`}
              title={row.hint}
            >
              {row.accent && (
                <span
                  className="h-3 w-[3px] flex-none"
                  style={{ backgroundColor: row.accent }}
                  aria-hidden="true"
                />
              )}
              <span className="truncate">{row.label}</span>
            </div>
            {columns.map((col, c) => {
              const cell = cells[r][c]
              return (
                <div key={col.key} className="flex items-center" style={{ gap: VALUE_GAP }}>
                  <Tooltip content={cell.tooltip} className="w-full">
                    <div
                      className="relative h-[11px] w-full min-w-[40px] bg-muted"
                      tabIndex={0}
                      role="img"
                      aria-label={cell.tooltip}
                    >
                      <div
                        className="absolute left-0 top-0 h-full"
                        style={{
                          width: `${cell.fraction * 100}%`,
                          backgroundColor: cell.color,
                        }}
                      />
                    </div>
                  </Tooltip>
                  <span
                    className="text-right text-3xs text-muted-foreground tabular-nums"
                    style={{ minWidth: VALUE_WIDTH }}
                  >
                    {cell.text}
                  </span>
                </div>
              )
            })}
          </Fragment>
        ))}

        <div />
        {columns.map((col) => (
          <div
            key={col.key}
            className="relative mt-1 h-[15px] border-t border-border text-3xs text-muted-foreground tabular-nums"
          >
            <span className="absolute left-0 top-0.5">{col.axis[0]}</span>
            {/* Le milieu se cale sur le milieu du RAIL, pas de la cellule : la cellule porte
                aussi le nombre écrit à droite de la barre. */}
            <span
              className="absolute top-0.5 -translate-x-1/2"
              style={{ left: `calc((100% - ${VALUE_WIDTH + VALUE_GAP}px) / 2)` }}
            >
              {col.axis[1]}
            </span>
            <span className="absolute top-0.5" style={{ right: VALUE_WIDTH + VALUE_GAP }}>
              {col.axis[2]}
            </span>
          </div>
        ))}
      </div>
    </div>
  )
}
