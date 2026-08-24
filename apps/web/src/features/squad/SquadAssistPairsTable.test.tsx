/**
 * SquadAssistPairsTable — le tableau « qui assiste qui » et son BANDEAU DE COUVERTURE.
 *
 * Ce que ces tests cadenassent : la couverture s'affiche TOUJOURS (y compris quand il
 * n'y a aucune paire, où elle est justement ce qui distingue « aucune entraide » de
 * « rien mesuré »), la part se calcule sur le total SERVEUR, et « Éliminations volées »
 * est bien la colonne produit en FR — pas un anglicisme.
 */
import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { fireEvent, screen, within } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import type { SquadAssistPair, SquadAssistPairs } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'

import { SquadAssistPairsTable } from './SquadAssistPairsTable'

const pair = (over: Partial<SquadAssistPair> = {}): SquadAssistPair => ({
  assist_xuid: 'X_A',
  assist_gamertag: 'Alice',
  killer_xuid: 'X_B',
  killer_gamertag: 'Bob',
  assist_count: 3,
  stolen_count: 1,
  ...over,
})

const block = (over: Partial<SquadAssistPairs> = {}): SquadAssistPairs => ({
  matches_measured: 4,
  matches_total: 10,
  total_assists: 4,
  pairs: [pair()],
  ...over,
})

beforeEach(() => useAppShellStore.setState({ locale: 'fr' }))
afterEach(() => useAppShellStore.setState({ locale: 'fr' }))

describe('SquadAssistPairsTable', () => {
  it('affiche le bandeau de couverture « mesuré sur N des M matchs »', () => {
    renderWithProviders(<SquadAssistPairsTable block={block()} />)
    const banner = screen.getByTestId('squad-assist-pairs-coverage')
    expect(banner.textContent).toContain('4')
    expect(banner.textContent).toContain('10')
  })

  it('garde la couverture AFFICHÉE sans aucune paire — sinon « aucune » se lirait « rien mesuré »', () => {
    renderWithProviders(<SquadAssistPairsTable block={block({ pairs: [], total_assists: 0 })} />)
    expect(screen.getByTestId('squad-assist-pairs-coverage')).toBeTruthy()
    expect(screen.getByText(/Aucune assistance entre membres/i)).toBeTruthy()
  })

  it('traite `pairs: null` (tableau nullable du contrat) comme une liste vide', () => {
    renderWithProviders(
      <SquadAssistPairsTable block={{ ...block(), pairs: null } as unknown as SquadAssistPairs} />,
    )
    expect(screen.getByText(/Aucune assistance entre membres/i)).toBeTruthy()
  })

  it('rend les 5 colonnes, dont « Éliminations volées » en FR', () => {
    renderWithProviders(<SquadAssistPairsTable block={block()} />)
    const headText = screen.getByRole('table').querySelector('thead')?.textContent ?? ''
    for (const label of [
      'Assistant',
      'Bénéficiaire',
      'Assistances',
      'Part',
      'Éliminations volées',
    ]) {
      expect(headText).toContain(label)
    }
    // Pas d'anglicisme dans l'en-tête FR.
    expect(headText).not.toContain('Stolen')
  })

  it('calcule la part sur le TOTAL SERVEUR, pas sur la somme des lignes visibles', () => {
    // Deux lignes à 3 et 1, mais le total serveur vaut 8 : les parts affichées sont
    // 37,5 % et 12,5 % — jamais 75 % / 25 %.
    renderWithProviders(
      <SquadAssistPairsTable
        block={block({
          total_assists: 8,
          pairs: [pair({ assist_count: 3 }), pair({ assist_gamertag: 'Carol', assist_count: 1 })],
        })}
      />,
    )
    const text = screen.getByRole('table').querySelector('tbody')?.textContent ?? ''
    expect(text).toContain('37,5')
    expect(text).toContain('12,5')
    expect(text).not.toContain('75,0')
  })

  it('trie sur clic d\'en-tête (éliminations volées décroissantes au premier clic)', () => {
    renderWithProviders(
      <SquadAssistPairsTable
        block={block({
          total_assists: 10,
          pairs: [
            pair({ assist_gamertag: 'Alice', assist_count: 6, stolen_count: 0 }),
            pair({ assist_gamertag: 'Carol', assist_count: 4, stolen_count: 4 }),
          ],
        })}
      />,
    )
    const rowsBefore = screen.getByRole('table').querySelectorAll('tbody tr')
    expect(within(rowsBefore[0] as HTMLElement).getByText('Alice')).toBeTruthy()

    fireEvent.click(screen.getByRole('button', { name: /Trier par Éliminations volées/i }))
    const rowsAfter = screen.getByRole('table').querySelectorAll('tbody tr')
    expect(within(rowsAfter[0] as HTMLElement).getByText('Carol')).toBeTruthy()
  })

  it('bascule les libellés en anglais avec la locale EN', () => {
    useAppShellStore.setState({ locale: 'en' })
    renderWithProviders(<SquadAssistPairsTable block={block()} />)
    const headText = screen.getByRole('table').querySelector('thead')?.textContent ?? ''
    expect(headText).toContain('Stolen kills')
    expect(headText).toContain('Beneficiary')
  })
})
