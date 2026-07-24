/**
 * Tests — jauge « progression vers le rang max » (career.02), title-agnostic.
 *
 * Couvre : résolution title-agnostic du libellé du rang max (Infinite « Héros »,
 * Halo 5 « SR 152 », repli générique), rendu par titre (compteur X/N par titre),
 * et masquage quand le titre ne déclare pas la capability `career`.
 */
import { beforeEach, describe, it, expect } from 'vitest'
import { screen } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'
import type { HeroProgress } from '@/lib/api/types'
import { CareerHeroGaugeChart, heroMaxRankName } from './CareerChartsSection.gauges'

function hero(over: Partial<HeroProgress>): HeroProgress {
  return {
    xp_total_required: 9_319_350,
    xp_remaining: 8_527_380,
    percentage: 8.5,
    current_rank: 12,
    total_ranks: 272,
    ...over,
  }
}

function setTitle(capabilities: string[]) {
  useAppShellStore.setState({
    currentTitleSlug: 'unit',
    availableTitles: [
      { slug: 'unit', name: 'Unit', status: 'active', capabilities, is_default: false, effective_hp_to_kill: 225 },
    ] as unknown as ReturnType<typeof useAppShellStore.getState>['availableTitles'],
  })
}

describe('heroMaxRankName (résolution title-agnostic)', () => {
  it('Halo Infinite : « Héros » (fr) / « Hero » (en) depuis le payload', () => {
    const h = hero({ max_rank_name_fr: 'Héros', max_rank_name_en: 'Hero' })
    expect(heroMaxRankName(h, 'fr')).toBe('Héros')
    expect(heroMaxRankName(h, 'en')).toBe('Hero')
  })

  it('Halo 5 : « SR 152 » depuis le payload (aucun littéral Infinite)', () => {
    const h = hero({ total_ranks: 152, max_rank_name_fr: 'SR 152', max_rank_name_en: 'SR152' })
    expect(heroMaxRankName(h, 'fr')).toBe('SR 152')
    expect(heroMaxRankName(h, 'en')).toBe('SR152')
  })

  it('repli générique quand la source ne fournit pas le nom du rang max', () => {
    const h = hero({ max_rank_name_fr: undefined, max_rank_name_en: undefined })
    expect(heroMaxRankName(h, 'fr')).toBe('le rang max')
    expect(heroMaxRankName(h, 'en')).toBe('max rank')
  })
})

describe('CareerHeroGaugeChart', () => {
  beforeEach(() => {
    setTitle(['career'])
  })

  it('rendu Infinite : titre interpolé « Progression vers Héros » + compteur X/272', () => {
    renderWithProviders(
      <CareerHeroGaugeChart
        heroProgress={hero({ current_rank: 122, total_ranks: 272, max_rank_name_fr: 'Héros' })}
        locale="fr"
        intlLocale="fr-FR"
      />,
    )
    expect(screen.getByText('Progression vers Héros')).toBeInTheDocument()
    expect(screen.getByText('122/272')).toBeInTheDocument()
  })

  it('rendu Halo 5 : compteur X/152 (borne du titre, pas le fallback 272)', () => {
    renderWithProviders(
      <CareerHeroGaugeChart
        heroProgress={hero({ current_rank: 111, total_ranks: 152, max_rank_name_fr: 'SR 152' })}
        locale="fr"
        intlLocale="fr-FR"
      />,
    )
    expect(screen.getByText('111/152')).toBeInTheDocument()
  })

  it('masqué quand le titre ne déclare pas la capability `career`', () => {
    setTitle(['matchmaking']) // titre partiel, sans career
    renderWithProviders(
      <CareerHeroGaugeChart
        heroProgress={hero({})}
        locale="fr"
        intlLocale="fr-FR"
      />,
    )
    expect(screen.queryByText('Progression vers le rang max')).not.toBeInTheDocument()
  })
})
