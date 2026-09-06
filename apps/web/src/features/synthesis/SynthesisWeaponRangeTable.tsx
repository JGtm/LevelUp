/**
 * SynthesisWeaponRangeTable — LE MÊME CONTENU QUE LES DEUX GRAPHES, EN CHIFFRES.
 *
 * Déplié depuis le `<details>` de la carte « Portée par arme ». Extrait de la section pour
 * la garder sous le plafond de 500 lignes : c'est une vue complète (douze colonnes, deux
 * côtés) qui n'a rien à voir avec la mise en page de la carte.
 */
import type { WeaponRangeSide } from '@/lib/api/types'

import type { WeaponRangeLine } from './_weaponRangeChart'
import type { RangeFormats, SynthesisKey, Translate } from './weaponRangeText'

/**
 * Les six colonnes d'un côté. Seule la première change de libellé (« Mesurés » au masculin
 * pour les frags, « Mesurées » pour les morts — le français accorde, l'anglais non).
 */
const SIDE_COLUMNS: SynthesisKey[] = [
  'synthesis.weapon_range.table_p10',
  'synthesis.weapon_range.table_median',
  'synthesis.weapon_range.table_p90',
  'synthesis.weapon_range.table_above',
  'synthesis.weapon_range.table_below',
]
const COLUMNS: SynthesisKey[][] = [
  ['synthesis.weapon_range.table_measured_kills', ...SIDE_COLUMNS],
  ['synthesis.weapon_range.table_measured_deaths', ...SIDE_COLUMNS],
]

/**
 * Le tableau est STATIQUE et natif, à dessein : il redit, chiffre par chiffre, les lignes du
 * graphe qui le précède, DANS LE MÊME ORDRE. Le rendre triable le désynchroniserait de la
 * lecture d'à côté, qui est justement un continuum du contact à la longue portée.
 */
export function SynthesisWeaponRangeTable({
  lines,
  t,
  f,
}: {
  lines: WeaponRangeLine[]
  t: Translate
  f: RangeFormats
}) {
  const cell = (side: WeaponRangeSide | null, read: (s: WeaponRangeSide) => string) =>
    side ? read(side) : '—'
  const sideCells = (side: WeaponRangeSide | null) => [
    cell(side, (s) => f.count(s.measured)),
    cell(side, (s) => f.distance(s.p10)),
    cell(side, (s) => f.distance(s.median)),
    cell(side, (s) => f.distance(s.p90)),
    cell(side, (s) => f.percent(s.above_pct)),
    cell(side, (s) => f.percent(s.below_pct)),
  ]
  return (
    <table className="min-w-[720px] border-collapse text-xs tabular-nums">
      <thead>
        <tr>
          <th className="px-2.5 py-1" />
          <th className="px-2.5 pt-1 text-center font-medium text-muted-foreground" colSpan={6}>
            {t('synthesis.weapon_range.side_kills')}
          </th>
          <th className="px-2.5 pt-1 text-center font-medium text-muted-foreground" colSpan={6}>
            {t('synthesis.weapon_range.side_deaths')}
          </th>
        </tr>
        <tr className="text-muted-foreground">
          <th className="whitespace-nowrap border-b border-border px-2.5 py-1 text-left font-medium">
            {t('synthesis.weapon_range.table_weapon')}
          </th>
          {COLUMNS.map((side, sideIndex) =>
            side.map((key) => (
              <th
                key={`${sideIndex}-${key}`}
                className="whitespace-nowrap border-b border-border px-2.5 py-1 text-right font-medium"
              >
                {t(key)}
              </th>
            )),
          )}
        </tr>
      </thead>
      <tbody>
        {lines.map((line) => (
          <tr key={line.weaponKey}>
            <td className="whitespace-nowrap border-b border-border px-2.5 py-1 text-left">
              {line.label}
            </td>
            {[...sideCells(line.kills), ...sideCells(line.deaths)].map((value, i) => (
              <td
                key={i}
                className="whitespace-nowrap border-b border-border px-2.5 py-1 text-right"
              >
                {value}
              </td>
            ))}
          </tr>
        ))}
      </tbody>
    </table>
  )
}

