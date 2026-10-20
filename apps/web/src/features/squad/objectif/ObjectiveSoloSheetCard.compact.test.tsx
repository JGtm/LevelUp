/**
 * ObjectiveSoloSheetCard.compact.test.tsx — « Ma part à l'objectif » en vue compacte (tiroir de
 * comparaison de Sessions, maquette `makeSheets` avec `cp`) : le nombre au bout de chaque action
 * devient ma part du total de mon camp, le pied aussi. Témoin : la soirée du 22/09 (relevés
 * `.ai/V7.5/MESURES_SESSIONS_2026-10-06.md` §3 : Prendre 7 / 22, Défendre 10 / 33, Tenir 65 s / 138 s).
 */
import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { FORMES_TEXT } from '../formes/i18n'
import { block2209 } from './objectif.fixtures'
import { buildSoloObjectiveSheet } from './objectif.logic'
import { ObjectiveSoloSheetCard, type SoloSheetText } from './ObjectiveSoloSheetCard'

const T: SoloSheetText = {
  title: 'Ma part à l’objectif',
  info: 'La fiche du joueur affiché.',
  dominantRole: 'Rôle dominant',
  roles: { take: 'Prendre', defend: 'Défendre', hold: 'Tenir' },
  pctFmt: (v) => `${Math.round(v)} %`,
  durationFmt: (s) => `${Math.round(s)} s`,
  lineTip: (player, family, column, value, camp) => `${player} · ${family}\n${column} : ${value} des ${camp}`,
}

const sheet = buildSoloObjectiveSheet({ ...block2209(), squad: [{ xuid: 'xj', gamertag: 'JGtm' }] })!

describe('buildSoloObjectiveSheet — totaux de rôle de mon camp', () => {
  it('à côté des miens, ceux de mon camp (colonnes facultatives exclues) : Prendre 7 / 22, Défendre 10 / 33', () => {
    expect([sheet.roleTotals[0], sheet.campRoleTotals[0]]).toEqual([7, 22])
    expect([sheet.roleTotals[1], sheet.campRoleTotals[1]]).toEqual([10, 33])
  })
})

describe('ObjectiveSoloSheetCard — compact', () => {
  it('le nombre au bout d’une action devient ma part du camp ; le pied en parts', () => {
    render(
      <ObjectiveSoloSheetCard sheet={sheet} name="JGtm" familyLabel={(f) => f} columns={FORMES_TEXT.fr.columns} t={T} compact />,
    )
    const line = sheet.families[0].lines.find((l) => l.share != null)!
    expect(screen.getByTestId(`objective-solo-line-${line.key}`).textContent).toContain(T.pctFmt(line.share! * 100))
    const foot = screen.getByTestId('objective-solo-foot').textContent ?? ''
    const take = sheet.campRoleTotals[0] > 0 ? T.pctFmt((sheet.roleTotals[0] / sheet.campRoleTotals[0]) * 100) : '—'
    expect(foot).toContain(`${take} Prendre`)
  })
})
