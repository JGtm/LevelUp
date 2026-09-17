/**
 * Tests du bloc « Arme favorite » du briefing Explorer.
 *
 * Deux contrats : le NOMBRE d'armes suit `slots` (deux ou une, chacune tenant sur UNE
 * ligne — nom, barre et compteur côte à côte, gate visuel du 2026-09-17),
 * et l'HABILLAGE est celui des voisines de la rangée — la carte, empilé comme seul
 * (décision utilisateur au gate visuel du 2026-09-17). Le stub i18n renvoie la clé : on
 * contrôle la structure, pas la traduction.
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

function weaponsBlock(): ExplorerBriefingWeapons {
  return {
    entries: [
      { label: 'Fusil de combat', kills: 40, class: 'shoulder' },
      { label: 'Pistolet', kills: 25, class: 'sidearm' },
    ],
    measured_kills: 65,
    scope_kills: 120,
  }
}

/** Pistes des barres de frags (le fond gris sous la barre teintée), une par arme rendue. */
function bars(container: HTMLElement): number {
  return container.querySelectorAll('[class*="bg-muted-foreground/15"]').length
}

/** En-tête bordurée de BriefingSectionCard — la carte est là si et seulement si elle existe. */
function cardHeader(container: HTMLElement): Element | null {
  return container.querySelector('[class*="border-b"]')
}

describe('FavoriteWeaponBlock — nombre d’armes selon la place', () => {
  it('deux emplacements : deux armes, chacune avec sa barre', () => {
    const { container } = renderWithProviders(
      <FavoriteWeaponBlock weapons={weaponsBlock()} slots={2} t={t} locale="fr" />,
    )
    const text = container.textContent ?? ''
    expect(text).toContain('Fusil de combat')
    expect(text).toContain('Pistolet')
    expect(bars(container)).toBe(2)
  })

  it('un emplacement : la première arme seule, avec sa barre', () => {
    const { container } = renderWithProviders(
      <FavoriteWeaponBlock weapons={weaponsBlock()} slots={1} t={t} locale="fr" />,
    )
    const text = container.textContent ?? ''
    expect(text).toContain('Fusil de combat')
    expect(text).not.toContain('Pistolet')
    expect(bars(container)).toBe(1)
  })

  it('tient chaque arme sur UNE ligne : nom, barre et compteur côte à côte', () => {
    const { container } = renderWithProviders(
      <FavoriteWeaponBlock weapons={weaponsBlock()} slots={2} t={t} locale="fr" />,
    )
    const lignes = Array.from(container.querySelectorAll('li'))
    expect(lignes).toHaveLength(2)
    const attendu = [
      { label: 'Fusil de combat', kills: '40' },
      { label: 'Pistolet', kills: '25' },
    ]
    lignes.forEach((ligne, i) => {
      expect(ligne.textContent).toContain(attendu[i].label)
      expect(ligne.textContent).toContain(attendu[i].kills)
      // La barre vit DANS la ligne, plus en dessous.
      expect(ligne.querySelectorAll('[class*="bg-muted-foreground/15"]')).toHaveLength(1)
      // Rangée flex, jamais empilée — et jamais une grille (D10, piège du test DP-3).
      expect(ligne.className).toContain('flex')
      expect(ligne.className).not.toContain('flex-col')
      expect(ligne.className).not.toContain('grid')
    })
  })
})

describe('FavoriteWeaponBlock — la carte des voisines, dans les deux montages', () => {
  it('porte le titre dans son en-tête bordurée, à une comme à deux armes', () => {
    for (const slots of [1, 2] as const) {
      const { container } = renderWithProviders(
        <FavoriteWeaponBlock weapons={weaponsBlock()} slots={slots} t={t} locale="fr" />,
      )
      const header = cardHeader(container)
      expect(header).not.toBeNull()
      expect(header?.textContent).toContain('explorer.briefing.weapons_title')
    }
  })

  it('ne rend plus la note de couverture, même quand tout n’est pas mesuré', () => {
    // measured_kills (65) < scope_kills (120) : l'ancienne note aurait paru ici.
    const { container } = renderWithProviders(
      <FavoriteWeaponBlock weapons={weaponsBlock()} slots={2} t={t} locale="fr" />,
    )
    expect(container.textContent ?? '').not.toContain('weapons_coverage')
  })

  it('ne rend rien quand le payload ne porte aucune arme', () => {
    const vide: ExplorerBriefingWeapons = { entries: [], measured_kills: 0, scope_kills: 120 }
    const { container } = renderWithProviders(
      <FavoriteWeaponBlock weapons={vide} slots={2} t={t} locale="fr" />,
    )
    expect(container.textContent ?? '').toBe('')
  })
})
