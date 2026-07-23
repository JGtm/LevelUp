/**
 * Tests — MedalsView (rendu GÉNÉRIQUE des super-sections).
 * Vérifie : chaque super-section présente dans le view-model est rendue avec son
 * libellé, ses catégories et ses médailles (aucun nom de section en dur) ; état
 * vide quand aucune super-section.
 */
import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MedalsView, type MedalCategoryView, type MedalsViewModel } from './MedalsView'
import type { MedalSummaryItem } from '@/lib/api/types'

function item(name: string): MedalSummaryItem {
  return {
    medal_id: Math.floor(Math.random() * 1e9),
    name,
    description: '',
    difficulty: 'Normal',
    difficulty_key: 'normal',
    difficulty_rank: 0,
    category: 'x',
    super_section: 's',
    personal_score: 0,
    count: 1,
    sort: 0,
  }
}

function category(key: string, label: string, medalName: string): MedalCategoryView {
  return {
    key,
    label,
    pct: 50,
    isMastered: false,
    masteryLabel: '1/2 obtenues',
    totalAwardedLabel: '3 au total',
    items: [item(medalName)],
  }
}

describe('MedalsView', () => {
  it('rend GÉNÉRIQUEMENT toutes les super-sections présentes (libellés, catégories, médailles)', () => {
    const vm: MedalsViewModel = {
      superSections: [
        { key: 'classics', label: 'Classiques', categories: [category('spree', 'Série', 'Tueur en série')] },
        { key: 'other', label: 'Autres', categories: [category('skill', 'Talent', 'Perfection')] },
      ],
    }
    render(<MedalsView vm={vm} locale="fr" emptyTitle="vide" emptyDescription="rien" />)
    expect(screen.getByText('Classiques')).toBeTruthy()
    expect(screen.getByText('Autres')).toBeTruthy()
    expect(screen.getByText('Série')).toBeTruthy()
    expect(screen.getByText('Talent')).toBeTruthy()
    expect(screen.getByText('Tueur en série')).toBeTruthy()
    expect(screen.getByText('Perfection')).toBeTruthy()
  })

  it('view-model vide → carte état vide (emptyTitle)', () => {
    render(
      <MedalsView
        vm={{ superSections: [] }}
        locale="fr"
        emptyTitle="Aucune médaille à afficher"
        emptyDescription="Aucune médaille ne correspond au filtre sélectionné."
      />,
    )
    expect(screen.getByText('Aucune médaille à afficher')).toBeTruthy()
  })
})
