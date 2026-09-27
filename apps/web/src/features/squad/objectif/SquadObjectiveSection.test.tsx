/**
 * SquadObjectiveSection.test.tsx — la section « Objectif » de Contributions (lot L3) : ordre et
 * disposition des quatre cartes, masquage sans objectif, notes sous le minimum, fiches du 22/09.
 */
import { describe, expect, it, vi } from 'vitest'
import { render, screen, within } from '@testing-library/react'

import type { SquadFormesBlock } from '@/lib/api/types'

import { SquadObjectiveSection } from './SquadObjectiveSection'
import { block0709, block2209, history0709, history0709Evenings, history2209Evenings } from './objectif.fixtures'

const options: unknown[] = []
vi.mock('echarts-for-react', () => ({
  default: (props: { option: unknown }) => {
    options.push(props.option)
    return <div data-testid="echarts-mock" />
  },
}))

function renderSection(block: SquadFormesBlock | undefined, withHistory: 'soir0709' | 'soir2209' | 'none') {
  return render(
    <SquadObjectiveSection
      block={block}
      matchHistory={history0709()}
      objectiveHistory={
        withHistory === 'soir0709' ? history0709Evenings() : withHistory === 'soir2209' ? history2209Evenings() : undefined
      }
      medalDigest={[]}
      mainPlayerLabel="JGtm"
      locale="fr"
    />,
  )
}

describe('SquadObjectiveSection', () => {
  it('07/09 : les quatre cartes dans l’ordre du débrief, les deux premières côte à côte', async () => {
    renderSection(block0709(), 'soir0709')
    const section = screen.getByTestId('squad-objective-section')
    expect(within(section).getByText('Objectif')).toBeInTheDocument()
    const ids = ['objective-balance', 'objective-fil', 'objective-sheets', 'objective-evenings']
    const nodes = ids.map((id) => screen.getByTestId(id))
    for (let i = 1; i < nodes.length; i++) {
      expect(nodes[i - 1].compareDocumentPosition(nodes[i]) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
    }
    // Rapport de force | au fil de la session : la même rangée à deux colonnes.
    expect(nodes[0].parentElement).toBe(nodes[1].parentElement)
    expect(nodes[0].parentElement?.className).toContain('lg:grid-cols-2')
    // Les deux graphes sont tracés (sept matchs, onze soirées).
    expect(await screen.findAllByTestId('echarts-mock')).toHaveLength(2)
    expect(screen.getByTestId('objective-evenings-note').textContent).toContain('Les 10 soirées précédentes')
    expect(screen.getByTestId('objective-evenings-note').textContent).toContain('B : Bases, D : Drapeau')
  })

  it('rapport de force : un cadre par famille, « Bases · 4 matchs », actions sous Prendre / Défendre / Tenir', () => {
    renderSection(block0709(), 'none')
    const bases = screen.getByTestId('objective-balance-family-zones_strongholds')
    expect(bases.textContent).toContain('Bases')
    expect(bases.textContent).toContain('4 matchs')
    for (const role of ['Prendre', 'Défendre', 'Tenir']) expect(within(bases).getByText(role)).toBeInTheDocument()
    // jsdom ne mesure rien : chaque valeur passe au repli, jamais perdue (S3).
    const repli = screen.getByTestId('objective-balance-repli-zones_strongholds-time_in_zones_seconds')
    expect(repli.textContent).toContain('17 min 48 · 54 %')
    expect(repli.textContent).toContain('15 min 26')
  })

  it('22/09 : fiches JGtm (4 drapeaux capturés, 3 volés), zéro atténué, rôle dominant, pied de fiche', () => {
    renderSection(block2209(), 'soir2209')
    const jg = screen.getByTestId('objective-sheet-xj')
    expect(within(jg).getByText('JGtm')).toBeInTheDocument()
    expect(screen.getByTestId('objective-sheet-line-xj-flag_captures').textContent).toContain('4')
    expect(screen.getByTestId('objective-sheet-line-xj-flag_steals').textContent).toContain('3')
    expect(screen.getByTestId('objective-sheet-line-xc-flag_captures').getAttribute('data-zero')).toBe('true')
    expect(within(jg).getByText('Rôle dominant')).toBeInTheDocument()
    expect(within(jg).getByText('Rôle dominant').nextElementSibling?.textContent).toBe('Tenir')
    expect(jg.textContent).toContain('1 min 05')
    expect(screen.getByTestId('objective-sheet-rest').textContent).toContain('Reste du camp')
    // Deux matchs à objectif : ni courbe au fil de la session, ni point ce soir — les notes.
    expect(screen.getByTestId('objective-fil-note').textContent).toContain('2 matchs à objectif ce soir')
    expect(screen.getByTestId('objective-evenings-note').textContent).toContain('24 soirées de la composition sur 49')
    expect(screen.queryByTestId('echarts-mock')).toBeNull()
  })

  it('aucun match à objectif dans le périmètre : la section se retire, intertitre compris', () => {
    const b = block0709()
    b.matches = b.matches!.map((m) => ({ ...m, objective: undefined }))
    const { container } = renderSection(b, 'soir0709')
    expect(container).toBeEmptyDOMElement()
    const { container: c2 } = renderSection(undefined, 'none')
    expect(c2).toBeEmptyDOMElement()
  })

  it('sans historique de la composition (aucun coéquipier sélectionné) : pas de carte soirée après soirée', () => {
    renderSection(block0709(), 'none')
    expect(screen.queryByTestId('objective-evenings')).toBeNull()
    expect(screen.getByTestId('objective-sheets')).toBeInTheDocument()
  })
})
