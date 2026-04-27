/**
 * WeaponsTable — tableau armes utilisees par le squad (chunk S9).
 *
 * 1 ligne par arme, colonnes par joueur (kills) + total. Slider min kills
 * cote front filtre les armes a faible volume.
 */
import { useMemo, useState } from 'react'

import type { WeaponsTableRow } from '../types'

export interface WeaponsTableProps {
  rows: WeaponsTableRow[]
  /** Ordre stable des colonnes joueurs. */
  squadOrder: string[]
  /** Labels d'en-tete deja localises par le caller. */
  labels: {
    weapon: string
    total: string
    minKills: (n: number) => string
    grenadeMelee: string
  }
  /** Min kills initial. */
  defaultMinKills?: number
}

export function WeaponsTable({
  rows,
  squadOrder,
  labels,
  defaultMinKills = 0,
}: WeaponsTableProps) {
  const [minKills, setMinKills] = useState(defaultMinKills)

  const filtered = useMemo(
    () => rows.filter((r) => r.total >= minKills),
    [rows, minKills],
  )

  if (rows.length === 0) {
    return null
  }

  // Le max sert a borner le slider sans depasser la realite des donnees.
  const maxTotal = Math.max(...rows.map((r) => r.total))

  return (
    <div className="flex flex-col gap-3" data-testid="weapons-table">
      <div className="flex items-center gap-3">
        <label className="text-sm text-muted-foreground" htmlFor="weapons-min-kills">
          {labels.minKills(minKills)}
        </label>
        <input
          id="weapons-min-kills"
          type="range"
          min={0}
          max={maxTotal}
          step={1}
          value={minKills}
          onChange={(e) => setMinKills(Number(e.target.value))}
          className="flex-1"
          data-testid="weapons-table-slider"
        />
      </div>
      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead className="bg-muted border-b">
            <tr>
              <th className="px-3 py-2 text-left">{labels.weapon}</th>
              {squadOrder.map((gt) => (
                <th key={gt} className="px-3 py-2 text-center">
                  {gt}
                </th>
              ))}
              <th className="px-3 py-2 text-center">{labels.total}</th>
            </tr>
          </thead>
          <tbody className="divide-y">
            {filtered.map((row) => (
              <tr key={`${row.weapon_id}-${row.is_grenade_melee ? 'gm' : 'w'}`}>
                <td className="px-3 py-2">
                  {row.label ?? `#${row.weapon_id}`}
                  {row.is_grenade_melee && (
                    <span className="ml-2 text-xs text-muted-foreground">
                      ({labels.grenadeMelee})
                    </span>
                  )}
                </td>
                {squadOrder.map((gt) => (
                  <td key={gt} className="px-3 py-2 text-center">
                    {row.kills_by_xuid[gt] ?? 0}
                  </td>
                ))}
                <td className="px-3 py-2 text-center font-medium">{row.total}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
