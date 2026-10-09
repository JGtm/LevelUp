/**
 * SquadObjectiveSection.test.tsx — l'objectif de l'onglet Emprise : ordre et disposition des
 * blocs (rapport de force à même la section, fiches dans leur propre section), masquage sans
 * objectif, blocs placeholder sous le minimum, fiches du 22/09 sans reste du camp.
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
  it('07/09 : les blocs dans l’ordre du débrief, les deux premiers côte à côte, les fiches dans leur section', async () => {
    renderSection(block0709(), 'soir0709')
    const section = screen.getByTestId('squad-objective-section')
    expect(within(section).getByText('Objectif')).toBeInTheDocument()
    const ids = ['objective-balance', 'objective-fil', 'objective-evenings', 'objective-sheets']
    const nodes = ids.map((id) => screen.getByTestId(id))
    for (let i = 1; i < nodes.length; i++) {
      expect(nodes[i - 1].compareDocumentPosition(nodes[i]) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
    }
    // Rapport de force | au fil de la session : la même rangée à deux colonnes.
    expect(nodes[0].parentElement).toBe(nodes[1].parentElement)
    expect(nodes[0].parentElement?.className).toContain('lg:grid-cols-2')
    // Le rapport de force est à même la section : ni cadre ni titre de carte (titre redondant).
    expect(nodes[0].className).not.toContain('border')
    expect(within(nodes[0]).queryByText('Rapport de force')).toBeNull()
    // Les fiches ont leur propre section, titrée, sans carte englobante.
    const sheets = screen.getByTestId('squad-objective-sheets-section')
    expect(within(sheets).getByText('Répartition de l’objectif dans l’escouade')).toBeInTheDocument()
    expect(nodes[3].className).not.toContain('border')
    // Les deux graphes sont tracés (sept matchs, onze soirées) ; aucune phrase sous le graphe,
    // les abréviations de modes sont dans la légende.
    expect(await screen.findAllByTestId('echarts-mock')).toHaveLength(2)
    expect(screen.queryByTestId('objective-evenings-note')).toBeNull()
    const legend = within(nodes[2]).getByTestId('chart-card-legend')
    expect(legend.textContent).toContain('B : Bases')
    expect(legend.textContent).toContain('D : Drapeau')
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
    // Pas de fiche du reste du camp : les trois joueurs de l'escouade seulement.
    expect(screen.queryByTestId('objective-sheet-null')).toBeNull()
    expect(screen.queryByText('Reste de l’équipe')).toBeNull()
    expect(screen.getAllByText('Rôle dominant')).toHaveLength(3)
    // Deux matchs à objectif : ni courbe au fil de la session, ni point ce soir — les blocs
    // placeholder (EmptyStateNotice tiretée), titre en gras puis la cause.
    const fil = screen.getByTestId('objective-fil-note')
    expect(fil.textContent).toContain('2 matchs à objectif ce soir')
    expect(fil.textContent).toContain('Sous le minimum de trois, la carte se masque.')
    expect(fil.querySelector('.border-dashed')).not.toBeNull()
    const evenings = screen.getByTestId('objective-evenings-note')
    expect(evenings.textContent).toContain('24 soirées de la composition sur 49')
    expect(evenings.querySelector('.border-dashed')).not.toBeNull()
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
