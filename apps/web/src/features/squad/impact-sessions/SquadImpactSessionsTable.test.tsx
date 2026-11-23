import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'

import type { Locale } from '@/lib/i18n/locale'

import { getSquadText } from '../i18n'
import { impactSessionsFixture } from './impactSessions.fixtures'
import { SquadImpactSessionsTable } from './SquadImpactSessionsTable'

const INK: Record<string, string> = {
  JGtm: 'var(--ac-squad-player-1)',
  joueur_117: 'var(--ac-squad-player-2)',
  joueur_165: 'var(--ac-squad-player-3)',
}
const inkOf = (gt: string) => INK[gt] ?? 'var(--ac-muted)'

function renderTable(props: { picked?: readonly string[]; locale?: Locale; data?: typeof impactSessionsFixture | null } = {}) {
  const locale = props.locale ?? 'fr'
  const ui = (picked: readonly string[]) => (
    <SquadImpactSessionsTable
      data={props.data === undefined ? impactSessionsFixture : props.data}
      pickedSessionLabels={picked}
      inkOf={inkOf}
      locale={locale}
      impact={getSquadText(locale).impact}
      emptyMessage="Aucune donnée"
    />
  )
  const utils = render(ui(props.picked ?? []))
  return { ...utils, rerenderWith: (picked: readonly string[]) => utils.rerender(ui(picked)) }
}

const rowOf = (label: string | RegExp) => screen.getByText(label).closest('tr') as HTMLElement
const toggleOf = (label: string | RegExp) => within(rowOf(label)).getByRole('button')

describe('SquadImpactSessionsTable', () => {
  it('met « Toutes les sessions » en tête, repliée, puis une ligne par session de la plus récente à la plus ancienne', () => {
    renderTable()
    const rows = screen.getAllByTestId(/^squad-impact-row/)
    expect(rows).toHaveLength(5)
    expect(within(rows[0]).getByText('Toutes les sessions')).toBeInTheDocument()
    expect(within(rows[0]).getByRole('button')).toHaveAttribute('aria-expanded', 'false')
    expect(within(rows[1]).getByText('14/11/2026 20:05–23:10')).toBeInTheDocument()
    expect(within(rows[4]).getByText('31/10/2026 22:00–22:45')).toBeInTheDocument()
  })

  it('affiche le nombre de matchs, le bilan et le net par match de chaque joueur', () => {
    renderTable()
    const row = rowOf('14/11/2026 20:05–23:10')
    expect(within(row).getByText('5 matchs')).toBeInTheDocument()
    expect(row.textContent).toContain('31') // 3 victoires, 1 défaite
    expect(within(row).getByText('+0,5')).toBeInTheDocument() // JGtm : 2,5 points sur 5 matchs
  })

  it('un joueur sans match sur la session affiche un tiret au lieu d’un net', () => {
    renderTable()
    const row = rowOf('31/10/2026 22:00–22:45')
    expect(within(row).getAllByText('—')).toHaveLength(2)
  })

  it('sans sélection, ouvre la session la plus récente, sans teinte', () => {
    renderTable()
    expect(toggleOf('14/11/2026 20:05–23:10')).toHaveAttribute('aria-expanded', 'true')
    expect(toggleOf('10/11/2026 21:00–22:30')).toHaveAttribute('aria-expanded', 'false')
    expect(rowOf('14/11/2026 20:05–23:10').className).not.toContain('bg-info')
    expect(screen.getByRole('table', { name: 'Rôles d’impact de la session 14/11/2026 20:05–23:10' })).toBeInTheDocument()
  })

  it('ouvre et teinte la session sélectionnée, suffixe de compte volatil toléré', () => {
    renderTable({ picked: ['10/11/2026 21:00–22:30 (1)'] })
    expect(toggleOf('10/11/2026 21:00–22:30')).toHaveAttribute('aria-expanded', 'true')
    expect(rowOf('10/11/2026 21:00–22:30').className).toContain('bg-info')
    expect(toggleOf('14/11/2026 20:05–23:10')).toHaveAttribute('aria-expanded', 'false')
    expect(rowOf('14/11/2026 20:05–23:10').className).not.toContain('bg-info')
  })

  it('plusieurs sessions sélectionnées s’ouvrent toutes', () => {
    renderTable({ picked: ['14/11/2026 20:05–23:10 (5)', '07/11/2026 18:30–19:40 (2)'] })
    expect(toggleOf('14/11/2026 20:05–23:10')).toHaveAttribute('aria-expanded', 'true')
    expect(toggleOf('07/11/2026 18:30–19:40')).toHaveAttribute('aria-expanded', 'true')
    expect(toggleOf('10/11/2026 21:00–22:30')).toHaveAttribute('aria-expanded', 'false')
  })

  it('un clic sur la ligne la déplie et la replie ; plusieurs lignes restent ouvertes en même temps', async () => {
    renderTable()
    await userEvent.click(screen.getByText('10/11/2026 21:00–22:30'))
    expect(toggleOf('10/11/2026 21:00–22:30')).toHaveAttribute('aria-expanded', 'true')
    expect(toggleOf('14/11/2026 20:05–23:10')).toHaveAttribute('aria-expanded', 'true')
    await userEvent.click(screen.getByText('10/11/2026 21:00–22:30'))
    expect(toggleOf('10/11/2026 21:00–22:30')).toHaveAttribute('aria-expanded', 'false')
  })

  it('Entrée et Espace déplient la ligne au clavier, une seule fois chacune', async () => {
    renderTable()
    toggleOf('07/11/2026 18:30–19:40').focus()
    await userEvent.keyboard('{Enter}')
    expect(toggleOf('07/11/2026 18:30–19:40')).toHaveAttribute('aria-expanded', 'true')
    await userEvent.keyboard(' ')
    expect(toggleOf('07/11/2026 18:30–19:40')).toHaveAttribute('aria-expanded', 'false')
  })

  it('la matrice déplie un match par colonne (tous les matchs de la session) et une colonne par rôle', () => {
    renderTable()
    const matrix = screen.getByRole('table', { name: /Rôles d’impact de la session 14\/11/ })
    const headers = within(matrix).getAllByRole('columnheader')
    // joueur + 5 matchs + 11 rôles + score
    expect(headers).toHaveLength(1 + 5 + 11 + 1)
    expect(headers[1]).toHaveTextContent('1')
    expect(headers[5]).toHaveTextContent('5')
    expect(headers[1].querySelector('span[title^="Match 1"]')).not.toBeNull()
    expect(within(matrix).getAllByRole('row')).toHaveLength(1 + 3)
  })

  it('met un point quand le joueur n’a aucun rôle sur un match, et les icônes sinon', () => {
    renderTable()
    const matrix = screen.getByRole('table', { name: /Rôles d’impact de la session 14\/11/ })
    const jgtm = within(matrix).getAllByRole('row')[1]
    // match 1 : premier sang + bourreau ; match 2 : finisseur ; match 4 et 5 : aucun rôle
    expect(within(jgtm).getAllByTestId('squad-impact-cell')).toHaveLength(3)
    expect(within(jgtm).getAllByText('·')).toHaveLength(2)
  })

  it('porte le trait d’issue de chaque match : vert pour une victoire, rouge pour une défaite', () => {
    renderTable()
    const matrix = screen.getByRole('table', { name: /Rôles d’impact de la session 14\/11/ })
    const marks = within(matrix)
      .getAllByRole('columnheader')
      .slice(1, 6)
      .map((h) => h.querySelector('span')?.getAttribute('style') ?? '')
    expect(marks[0]).toContain('outcome-win')
    expect(marks[2]).toContain('outcome-loss')
    expect(marks[4]).toContain('outcome-dnf')
  })

  it('surligne le meilleur et le pire score, avec couronne au premier et banane au dernier négatif', () => {
    renderTable()
    const matrix = screen.getByRole('table', { name: /Rôles d’impact de la session 14\/11/ })
    const rows = within(matrix).getAllByRole('row')
    const scoreOf = (i: number) => rows[i].querySelector('td:last-child') as HTMLElement
    expect(scoreOf(3).textContent).toBe('+3,5') // joueur_165
    expect(scoreOf(3).getAttribute('style')).toContain('outcome-win')
    expect(within(scoreOf(3)).getByTitle('Champion (rang #1)')).toBeInTheDocument()
    expect(scoreOf(2).textContent).toBe('−0,5') // joueur_117
    expect(scoreOf(2).getAttribute('style')).toContain('outcome-loss')
    expect(within(scoreOf(2)).getByTitle(/Maillon faible/)).toBeInTheDocument()
    expect(scoreOf(1).getAttribute('style')).toBeNull() // JGtm, au milieu
  })

  it('marque le dernier d’un passager clandestin quand son score n’est pas négatif', () => {
    const [first, ...rest] = impactSessionsFixture.sessions ?? []
    const scores = [3, 1, 0]
    const data = {
      ...impactSessionsFixture,
      sessions: [{ ...first, players: (first.players ?? []).map((p, i) => ({ ...p, points: scores[i] })) }, ...rest],
    }
    renderTable({ data })
    const matrix = screen.getByRole('table', { name: /Rôles d’impact de la session 14\/11/ })
    const rows = within(matrix).getAllByRole('row')
    expect(within(rows[1]).getByTitle('Champion (rang #1)')).toBeInTheDocument()
    expect(within(rows[3]).getByTitle(/Passager clandestin/)).toBeInTheDocument()
    expect(within(matrix).queryByTitle(/Maillon faible/)).toBeNull()
  })

  it('surligne un compte de rôle : le plus grand compte d’un rôle qui coûte des points est le pire', async () => {
    renderTable()
    await userEvent.click(screen.getByText('Toutes les sessions'))
    const totals = screen.getByRole('table', { name: 'Rôles d’impact sur toutes les sessions affichées' })
    const headers = within(totals).getAllByRole('columnheader')
    // pas de colonnes de matchs : joueur + 11 rôles + score
    expect(headers).toHaveLength(1 + 11 + 1)
    const rows = within(totals).getAllByRole('row')
    // premier sang : JGtm 2, joueur_117 2, joueur_165 2 -> égalité, aucun surlignage ; top_killer : 2, 1, 2 -> pire joueur_117
    const topKillerCol = 1 + 10 // joueur puis rôles dans l'ordre : top_killer est le dernier rôle
    const cellsOf = (r: HTMLElement) => r.querySelectorAll('td')
    expect(cellsOf(rows[2])[topKillerCol - 1].getAttribute('style')).toContain('outcome-loss')
    expect(cellsOf(rows[1])[topKillerCol - 1].getAttribute('style')).toContain('outcome-win')
  })

  it('« Toutes les sessions » somme les matchs des sessions affichées', () => {
    renderTable()
    expect(within(rowOf('Toutes les sessions')).getByText('12 matchs · totaux par rôle')).toBeInTheDocument()
  })

  it('une nouvelle sélection remet l’état ouvert à zéro', async () => {
    const { rerenderWith } = renderTable()
    await userEvent.click(screen.getByText('07/11/2026 18:30–19:40'))
    expect(toggleOf('07/11/2026 18:30–19:40')).toHaveAttribute('aria-expanded', 'true')
    rerenderWith(['10/11/2026 21:00–22:30 (3)'])
    expect(toggleOf('07/11/2026 18:30–19:40')).toHaveAttribute('aria-expanded', 'false')
    expect(toggleOf('10/11/2026 21:00–22:30')).toHaveAttribute('aria-expanded', 'true')
  })

  it('un clic de l’utilisateur survit à un nouveau rendu de même sélection', async () => {
    const { rerenderWith } = renderTable()
    await userEvent.click(screen.getByText('07/11/2026 18:30–19:40'))
    rerenderWith([])
    expect(toggleOf('07/11/2026 18:30–19:40')).toHaveAttribute('aria-expanded', 'true')
  })

  it('rend le message vide sans session', () => {
    renderTable({ data: null })
    expect(screen.getByTestId('squad-impact-sessions')).toHaveTextContent('Aucune donnée')
    expect(screen.queryByRole('table')).toBeNull()
  })

  it('parle anglais : en-têtes, nombres et séparateur décimal', () => {
    renderTable({ locale: 'en' })
    expect(screen.getByText('All sessions')).toBeInTheDocument()
    expect(screen.getByRole('columnheader', { name: 'Record' })).toBeInTheDocument()
    expect(within(rowOf('14/11/2026 20:05–23:10')).getByText('+0.5')).toBeInTheDocument()
  })
})
