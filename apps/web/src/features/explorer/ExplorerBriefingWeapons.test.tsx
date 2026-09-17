/**
 * Tests du bloc « Arme favorite » du briefing Explorer.
 *
 * Deux contrats : la FORME suit le nombre d'emplacements libres (deux armes avec barre,
 * une arme avec barre, ou une ligne compacte sans barre), et la note de couverture ne
 * paraît que lorsque tous les frags de la sélection n'ont pas été rattachés à une arme.
 * Le stub i18n renvoie la clé — on contrôle la structure, pas la traduction.
 */
import { describe, expect, it } from 'vitest'

import { renderWithProviders } from '@/test/render-utils'
import type { ExplorerBriefingWeapons } from '@/lib/api/types'
import type { ExplorerManifestKey } from '@/lib/i18n/generated/explorer'

import { FavoriteWeaponBlock } from './ExplorerBriefingWeapons'

const t = ((key: string) => key) as (
  key: ExplorerManifestKey,
  values?: Record<string, string | number>,
) => string

function weaponsBlock(measured: number, scope: number): ExplorerBriefingWeapons {
  return {
    entries: [
      { label: 'Fusil de combat', kills: 40, class: 'shoulder' },
      { label: 'Pistolet', kills: 25, class: 'sidearm' },
    ],
    measured_kills: measured,
    scope_kills: scope,
  }
}

/** Pistes des barres de frags (le fond gris sous la barre teintée), une par arme rendue. */
function bars(container: HTMLElement): number {
  return container.querySelectorAll('[class*="bg-muted-foreground/15"]').length
}

describe('FavoriteWeaponBlock — trois formes selon la place libre', () => {
  it('deux emplacements : deux armes, chacune avec sa barre', () => {
    const { container } = renderWithProviders(
      <FavoriteWeaponBlock weapons={weaponsBlock(65, 120)} slots={2} t={t} locale="fr" />,
    )
    const text = container.textContent ?? ''
    expect(text).toContain('Fusil de combat')
    expect(text).toContain('Pistolet')
    expect(bars(container)).toBe(2)
  })

  it('un emplacement : la première arme seule, avec sa barre', () => {
    const { container } = renderWithProviders(
      <FavoriteWeaponBlock weapons={weaponsBlock(65, 120)} slots={1} t={t} locale="fr" />,
    )
    const text = container.textContent ?? ''
    expect(text).toContain('Fusil de combat')
    expect(text).not.toContain('Pistolet')
    expect(bars(container)).toBe(1)
  })

  it('aucun emplacement : forme compacte, une arme sans barre', () => {
    const { container } = renderWithProviders(
      <FavoriteWeaponBlock weapons={weaponsBlock(65, 120)} slots={0} t={t} locale="fr" />,
    )
    const text = container.textContent ?? ''
    expect(text).toContain('explorer.briefing.weapons_title')
    expect(text).toContain('Fusil de combat')
    expect(text).not.toContain('Pistolet')
    expect(bars(container)).toBe(0)
  })
})

describe('FavoriteWeaponBlock — note de couverture', () => {
  it('paraît quand une partie des frags de la sélection n’est pas rattachée à une arme', () => {
    const { container } = renderWithProviders(
      <FavoriteWeaponBlock weapons={weaponsBlock(65, 120)} slots={2} t={t} locale="fr" />,
    )
    expect(container.textContent ?? '').toContain('explorer.briefing.weapons_coverage')
  })

  it('disparaît quand tout est mesuré (cas de la source native du second titre)', () => {
    const { container } = renderWithProviders(
      <FavoriteWeaponBlock weapons={weaponsBlock(120, 120)} slots={2} t={t} locale="fr" />,
    )
    expect(container.textContent ?? '').not.toContain('explorer.briefing.weapons_coverage')
  })

  it('disparaît aussi quand la source crédite PLUS que le total de la sélection', () => {
    const { container } = renderWithProviders(
      <FavoriteWeaponBlock weapons={weaponsBlock(130, 120)} slots={0} t={t} locale="fr" />,
    )
    expect(container.textContent ?? '').not.toContain('explorer.briefing.weapons_coverage')
  })
})
