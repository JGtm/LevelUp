/**
 * SquadObjectiveSection.test.tsx — l'objectif de l'onglet Emprise : ordre et disposition des
 * blocs (rapport de force à même la section, fiches dans leur propre section), masquage sans
 * objectif, bloc placeholder sous le minimum, fiches du 22/09 sans reste du camp. Le rapport de
 * force par soirée est dans « Soirées précédentes » (SquadEmprisePage.soirees.test.tsx).
 */
import { describe, expect, it, vi } from 'vitest'
import { render, screen, within } from '@testing-library/react'

import type { SquadFormesBlock } from '@/lib/api/types'

import { SquadObjectiveSection } from './SquadObjectiveSection'
import { block0709, block2209, history0709 } from './objectif.fixtures'

const options: unknown[] = []
vi.mock('echarts-for-react', () => ({
  default: (props: { option: unknown }) => {
    options.push(props.option)
    return <div data-testid="echarts-mock" />
  },
}))

function renderSection(block: SquadFormesBlock | undefined) {
  return render(
    <SquadObjectiveSection
      block={block}
      matchHistory={history0709()}
      medalDigest={[]}
      mainPlayerLabel="JGtm"
      locale="fr"
    />,
  )
}

describe('SquadObjectiveSection', () => {
  it('07/09 : les blocs dans l’ordre du débrief, les deux premiers côte à côte, les fiches dans leur section', async () => {
    renderSection(block0709())
    const section = screen.getByTestId('squad-objective-section')
    expect(within(section).getByText('Objectif')).toBeInTheDocument()
    const ids = ['objective-balance', 'objective-fil', 'objective-sheets']
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
    expect(nodes[2].className).not.toContain('border')
    // Le graphe au fil de la session est tracé (sept matchs) ; le rapport de force par soirée n'est
    // plus dans cette section.
    expect(await screen.findAllByTestId('echarts-mock')).toHaveLength(1)
    expect(screen.queryByTestId('objective-evenings')).toBeNull()
  })

  it('rapport de force : un cadre par famille, « Bases · 4 matchs », actions sous Prendre / Défendre / Tenir', () => {
    renderSection(block0709())
    const bases = screen.getByTestId('objective-balance-family-zones_strongholds')
    expect(bases.textContent).toContain('Bases')
    expect(bases.textContent).toContain('4 matchs')
    for (const role of ['Prendre', 'Défendre', 'Tenir']) expect(within(bases).getByText(role)).toBeInTheDocument()
    // jsdom ne mesure rien : les valeurs sont masquées dans leur segment, aucune ligne au-dessus de la barre.
    expect(screen.queryByTestId('objective-balance-repli-zones_strongholds-time_in_zones_seconds')).toBeNull()
    const labels = screen.getByTestId('objective-balance-row-zones_strongholds-time_in_zones_seconds').querySelectorAll<HTMLElement>('[data-fit-label]')
    expect([...labels].map((l) => l.textContent)).toEqual(['17 min 48 · 54 %', '15 min 26'])
  })

  it('22/09 : fiches JGtm (4 drapeaux capturés, 3 volés), zéro atténué, rôle dominant, pied de fiche', () => {
    renderSection(block2209())
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
    // Deux matchs à objectif : pas de barres au fil de la session — le bloc placeholder
    // (EmptyStateNotice tiretée), titre en gras puis la cause.
    const fil = screen.getByTestId('objective-fil-note')
    expect(fil.textContent).toContain('2 matchs à objectif ce soir')
    expect(fil.textContent).toContain('Sous le minimum de trois, la carte se masque.')
    expect(fil.querySelector('.border-dashed')).not.toBeNull()
    expect(screen.queryByTestId('echarts-mock')).toBeNull()
  })

  it('aucun match à objectif dans le périmètre : la section se retire, intertitre compris', () => {
    const b = block0709()
    b.matches = b.matches!.map((m) => ({ ...m, objective: undefined }))
    const { container } = renderSection(b)
    expect(container).toBeEmptyDOMElement()
    const { container: c2 } = renderSection(undefined)
    expect(c2).toBeEmptyDOMElement()
  })
})
