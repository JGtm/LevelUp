/**
 * Tests — SaisonPill : popover avec folding "+N saisons sans matchs".
 *
 * Couvre :
 *   - Trigger label (collapsed) selon activeSeason
 *   - Saisons avec count > 0 visibles d'office
 *   - Saisons avec count === 0 repliées sous <details><summary>
 *   - Fallback : pas de seasonCounts → toutes les saisons visibles, pas de folding
 *   - onSelectSeason appelé avec la bonne SeasonEntry au click
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, within } from '@testing-library/react'

import type { SeasonEntry } from '@/lib/i18n/fieldMappings'
import { useAppShellStore } from '@/stores/appShellStore'
import { SaisonPill } from './SaisonPill'

// Libellés résolus par clé i18n (GH2-B1) — locale 'fr' épinglée pour les
// assertions historiques FR ; le bloc « i18n EN » bascule explicitement en 'en'.
beforeEach(() => {
  useAppShellStore.setState({ locale: 'fr' })
})

function makeSeasons(): SeasonEntry[] {
  return [
    {
      id: 'season1',
      label: 'Heroes of Reach',
      shortLabel: 'S1',
      startDate: new Date('2021-12-08T00:00:00Z'),
      endDate: new Date('2022-05-03T00:00:00Z'),
      displayOrder: 10,
    },
    {
      id: 'season2',
      label: 'Lone Wolves',
      shortLabel: 'S2',
      startDate: new Date('2022-05-03T00:00:00Z'),
      endDate: new Date('2022-11-08T00:00:00Z'),
      displayOrder: 20,
    },
    {
      id: 'season3',
      label: 'Echoes Within',
      shortLabel: 'S3',
      startDate: new Date('2023-03-07T00:00:00Z'),
      endDate: new Date('2023-06-20T00:00:00Z'),
      displayOrder: 30,
    },
    {
      id: 'season4',
      label: 'Infection',
      shortLabel: 'S4',
      startDate: new Date('2023-06-20T00:00:00Z'),
      endDate: new Date('2023-10-17T00:00:00Z'),
      displayOrder: 40,
    },
    {
      id: 'season6',
      label: 'Spirit of Fire',
      shortLabel: 'S6',
      startDate: new Date('2024-01-30T18:00:00Z'),
      endDate: new Date('2024-04-30T18:00:00Z'),
      displayOrder: 60,
    },
  ]
}

describe('SaisonPill — trigger', () => {
  it('affiche "Saison" quand aucune saison active', () => {
    render(
      <SaisonPill
        open={false}
        onToggle={vi.fn()}
        onClose={vi.fn()}
        seasons={makeSeasons()}
        activeSeason={null}
        onSelectSeason={vi.fn()}
      />,
    )
    expect(screen.getByRole('button', { name: /Saison/ })).toBeInTheDocument()
  })

  it('affiche "S6 — Spirit of Fire" quand S6 est active', () => {
    const seasons = makeSeasons()
    const s6 = seasons.find((s) => s.id === 'season6')!
    render(
      <SaisonPill
        open={false}
        onToggle={vi.fn()}
        onClose={vi.fn()}
        seasons={seasons}
        activeSeason={s6}
        onSelectSeason={vi.fn()}
      />,
    )
    expect(screen.getByText(/S6 — Spirit of Fire/)).toBeInTheDocument()
  })
})

describe('SaisonPill — popover folding', () => {
  it('replie les saisons à count=0 sous "+ N saisons sans matchs"', () => {
    const seasons = makeSeasons()
    // S6=42, S4=12, S2=8 visibles ; S1=0, S3=0 repliées
    const seasonCounts = [
      { season_id: 'season1', count: 0 },
      { season_id: 'season2', count: 8 },
      { season_id: 'season3', count: 0 },
      { season_id: 'season4', count: 12 },
      { season_id: 'season6', count: 42 },
    ]
    render(
      <SaisonPill
        open={true}
        onToggle={vi.fn()}
        onClose={vi.fn()}
        seasons={seasons}
        activeSeason={null}
        seasonCounts={seasonCounts}
        onSelectSeason={vi.fn()}
      />,
    )

    const dialog = screen.getByRole('dialog', { name: /Choix de la saison/ })

    // S2/S4/S6 visibles d'office avec leur count
    expect(within(dialog).getByText('Lone Wolves')).toBeInTheDocument()
    expect(within(dialog).getByText('Infection')).toBeInTheDocument()
    expect(within(dialog).getByText('Spirit of Fire')).toBeInTheDocument()
    expect(within(dialog).getByText('(42)')).toBeInTheDocument()
    expect(within(dialog).getByText('(12)')).toBeInTheDocument()
    expect(within(dialog).getByText('(8)')).toBeInTheDocument()

    // Le summary indique 2 saisons indisponibles
    expect(within(dialog).getByText(/\+ 2 saisons sans matchs/)).toBeInTheDocument()

    // S1 et S3 ne sont PAS visibles avant click sur le summary (présentes dans le DOM
    // mais cachées par <details>). On vérifie quand même qu'elles sont rendues
    // (HTML5 <details> garde le DOM, juste affiché en display:none).
    expect(within(dialog).getByText('Heroes of Reach')).toBeInTheDocument()
    expect(within(dialog).getByText('Echoes Within')).toBeInTheDocument()
  })

  it('sans seasonCounts : toutes les saisons visibles sans folding', () => {
    render(
      <SaisonPill
        open={true}
        onToggle={vi.fn()}
        onClose={vi.fn()}
        seasons={makeSeasons()}
        activeSeason={null}
        onSelectSeason={vi.fn()}
      />,
    )
    const dialog = screen.getByRole('dialog')
    expect(within(dialog).getByText('Heroes of Reach')).toBeInTheDocument()
    expect(within(dialog).getByText('Echoes Within')).toBeInTheDocument()
    // Aucun folding
    expect(within(dialog).queryByText(/sans matchs/)).not.toBeInTheDocument()
  })

  it('click sur une saison appelle onSelectSeason avec la bonne entry', () => {
    const onSelectSeason = vi.fn()
    const onClose = vi.fn()
    render(
      <SaisonPill
        open={true}
        onToggle={vi.fn()}
        onClose={onClose}
        seasons={makeSeasons()}
        activeSeason={null}
        onSelectSeason={onSelectSeason}
      />,
    )

    fireEvent.click(screen.getByText('Spirit of Fire'))
    expect(onSelectSeason).toHaveBeenCalledTimes(1)
    expect(onSelectSeason.mock.calls[0][0].id).toBe('season6')
    expect(onClose).toHaveBeenCalledTimes(1)
  })
})

describe('SaisonPill — ordre récent-en-haut (GH5-1)', () => {
  it("préserve l'ordre des saisons fourni (récent en tête) dans le popover", () => {
    // useSeasons trie récent-d'abord (DESC) ; la pill n'inverse pas — elle rend
    // dans l'ordre reçu. On fournit un tableau DESC (S6 → S1) et on vérifie que
    // le DOM sort dans cet ordre.
    const desc = [...makeSeasons()].reverse() // S6, S4, S3, S2, S1
    render(
      <SaisonPill
        open={true}
        onToggle={vi.fn()}
        onClose={vi.fn()}
        seasons={desc}
        activeSeason={null}
        onSelectSeason={vi.fn()}
      />,
    )
    const dialog = screen.getByRole('dialog')
    const buttons = within(dialog).getAllByRole('button')
    // Sans seasonCounts ni onClear : les boutons du dialog = uniquement les rows saison.
    expect(buttons[0].textContent).toContain('Spirit of Fire') // S6, plus récente
    expect(buttons[buttons.length - 1].textContent).toContain('Heroes of Reach') // S1, plus ancienne
  })
})

describe('SaisonPill — onClear', () => {
  it("le bouton 'Toutes saisons' est cliquable quand une saison est active", () => {
    const seasons = makeSeasons()
    const s6 = seasons.find((s) => s.id === 'season6')!
    const onClear = vi.fn()
    const onClose = vi.fn()
    render(
      <SaisonPill
        open={true}
        onToggle={vi.fn()}
        onClose={onClose}
        seasons={seasons}
        activeSeason={s6}
        onSelectSeason={vi.fn()}
        onClear={onClear}
      />,
    )

    const btn = screen.getByRole('button', { name: 'Toutes saisons' })
    fireEvent.click(btn)
    expect(onClear).toHaveBeenCalledTimes(1)
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it("le bouton 'Toutes saisons' est désactivé quand aucune saison n'est active", () => {
    render(
      <SaisonPill
        open={true}
        onToggle={vi.fn()}
        onClose={vi.fn()}
        seasons={makeSeasons()}
        activeSeason={null}
        onSelectSeason={vi.fn()}
        onClear={vi.fn()}
      />,
    )
    const btn = screen.getByRole('button', { name: 'Toutes saisons' })
    expect(btn).toBeDisabled()
  })
})

describe('SaisonPill — i18n EN (GH2-B1)', () => {
  it('rend le trigger, "All seasons" et le folding en anglais sous locale en', () => {
    useAppShellStore.setState({ locale: 'en' })
    const seasonCounts = [
      { season_id: 'season1', count: 0 },
      { season_id: 'season2', count: 8 },
      { season_id: 'season3', count: 0 },
      { season_id: 'season4', count: 12 },
      { season_id: 'season6', count: 42 },
    ]
    render(
      <SaisonPill
        open={true}
        onToggle={vi.fn()}
        onClose={vi.fn()}
        seasons={makeSeasons()}
        activeSeason={null}
        seasonCounts={seasonCounts}
        onSelectSeason={vi.fn()}
        onClear={vi.fn()}
      />,
    )
    // Trigger EN (pas de saison active).
    expect(screen.getByRole('button', { name: /Season/ })).toBeInTheDocument()
    const dialog = screen.getByRole('dialog', { name: /Season selector/ })
    // Bouton clear EN.
    expect(within(dialog).getByRole('button', { name: 'All seasons' })).toBeInTheDocument()
    // Folding EN (2 saisons à count=0), plus aucun libellé FR.
    expect(within(dialog).getByText(/\+ 2 seasons without matches/)).toBeInTheDocument()
    expect(within(dialog).queryByText(/sans matchs/)).not.toBeInTheDocument()
    expect(within(dialog).queryByText('Toutes saisons')).not.toBeInTheDocument()
  })
})
