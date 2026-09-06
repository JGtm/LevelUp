/**
 * SynthesisWeaponRangeSection.test — la section « Portée des engagements ».
 *
 * Ce que ces tests verrouillent : les quatre tuiles AVEC leur dénominateur, les DEUX tuiles
 * d'entame qui n'apparaissent QUE si l'entame est mesurée (jamais un zéro — décision D5), la
 * ligne « sous le seuil » qui NOMME les armes écartées et disparaît quand il n'y en a pas, la
 * note de couverture, le tableau dépliable (les deux côtés, le tiret du côté non mesuré), et
 * le retrait complet de la section quand rien n'est publiable.
 *
 * Les deux graphes sont testés PURS (`_weaponRangeChart.test.ts`,
 * `_weaponElevationChart.test.ts`) — ici ECharts est mocké (jsdom ne peint pas de canvas).
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { screen, within } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import type { WeaponRangeSide } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'

import { SynthesisWeaponRangeSection } from './SynthesisWeaponRangeSection'
import type { WeaponRangeBlock } from './weaponRange_logic'

vi.mock('echarts-for-react', () => ({
  default: () => <div data-testid="echarts-mock" />,
}))

const side = (o: Partial<WeaponRangeSide>): WeaponRangeSide => ({
  measured: 10,
  p10: 5,
  median: 7,
  p90: 10,
  above_pct: 30,
  level_pct: 50,
  below_pct: 20,
  ...o,
})

const RANGE: WeaponRangeBlock = {
  weapons: [
    {
      weapon_key: 'hinf_br75',
      label: 'Fusil de combat BR75',
      label_en: 'BR75 Battle Rifle',
      kills: side({ measured: 281, p10: 7.1, median: 13.6, p90: 24.9, above_pct: 37, level_pct: 49.1, below_pct: 13.9 }),
      deaths: side({ measured: 402, p10: 8.9, median: 16.4, p90: 29.7, above_pct: 10, level_pct: 52, below_pct: 38 }),
    },
    {
      // Arme mesurée d'un seul côté : un seul bâton au graphe, des tirets au tableau.
      weapon_key: 'hinf_commando',
      label: 'Commando VK78',
      label_en: 'VK78 Commando',
      kills: side({ measured: 64, p10: 6.3, median: 11.9, p90: 21.4 }),
    },
  ],
  median_kills_m: 7.4,
  median_deaths_m: 11.8,
  measured_kills: 1214,
  total_kills: 1602,
  measured_deaths: 1087,
  total_deaths: 1455,
  below_threshold_kills: [
    { weapon_key: 'hinf_hydra', label: 'Hydra', label_en: 'Hydra', measured: 6 },
    { weapon_key: 'hinf_disruptor', label: 'Disrupteur', label_en: 'Disruptor', measured: 4 },
  ],
  below_threshold_deaths: [{ weapon_key: 'hinf_ravager', label: 'Ravageur', label_en: 'Ravager', measured: 5 }],
  opening: {
    median_m: 9.1,
    measured_kills: 618,
    delta: { median_m: -1.7, closing_share_pct: 61.4, n: 574 },
  },
}

/** Espaces fines/insécables des formats FR : on compare sur du texte normalisé. */
const flat = (s: string | null | undefined) => (s ?? '').replace(/[\s\u00A0\u202F]+/g, ' ').trim()

function textOf(re: RegExp): string[] {
  return screen
    .getAllByText((_, node) => (node ? re.test(flat(node.textContent)) : false))
    .map((n) => flat(n.textContent))
}

beforeEach(() => {
  useAppShellStore.setState({ locale: 'fr' })
})

describe('SynthesisWeaponRangeSection — rendu nominal', () => {
  it('affiche la carte, son compte d’armes et les deux graphes', () => {
    renderWithProviders(<SynthesisWeaponRangeSection range={RANGE} />)
    const card = screen.getByRole('region', { name: 'Portée par arme — mes frags et mes morts' })
    expect(card).toBeInTheDocument()
    expect(flat(card.querySelector('h3')?.textContent)).toContain('2 armes')
    // Deux ChartCard : le canvas lui-même est chargé en `lazy`, on pince la carte.
    expect(screen.getAllByTestId('chart-card')).toHaveLength(2)
  })

  it('les quatre tuiles portent leur valeur ET leur dénominateur', () => {
    renderWithProviders(<SynthesisWeaponRangeSection range={RANGE} />)
    expect(screen.getByText('Portée médiane de mes frags')).toBeInTheDocument()
    expect(textOf(/^1 214 frags mesurés sur 1 602$/).length).toBeGreaterThan(0)
    expect(screen.getByText('Portée médiane de mes morts')).toBeInTheDocument()
    expect(textOf(/^1 087 morts mesurées sur 1 455$/).length).toBeGreaterThan(0)
    expect(screen.getByText("Distance d'entame médiane")).toBeInTheDocument()
    expect(textOf(/^1,5 s avant le frag · 618 frags mesurés$/).length).toBeGreaterThan(0)
    expect(screen.getByText('Entame → frag')).toBeInTheDocument()
    expect(textOf(/^Vous fermez la distance dans 61 % des frags$/).length).toBeGreaterThan(0)
    // Distance signée : le signe DIT le sens (la distance se ferme), il n'est pas décoratif.
    expect(textOf(/^-1,7 m$/).length).toBeGreaterThan(0)
  })

  it('les deux légendes nomment les séries ET leur position dans le graphe', () => {
    renderWithProviders(<SynthesisWeaponRangeSection range={RANGE} />)
    expect(textOf(/^Mes frags \(bâton du haut\)$/).length).toBeGreaterThan(0)
    expect(textOf(/^Mes morts \(bâton du bas, l'arme est celle du tueur\)$/).length).toBeGreaterThan(0)
    expect(screen.getByText("d'en haut (> +1 m)")).toBeInTheDocument()
    expect(screen.getByText('à niveau')).toBeInTheDocument()
    expect(screen.getByText("d'en bas (< −1 m)")).toBeInTheDocument()
  })

  it('nomme les armes écartées par le seuil, des deux côtés, et publie la note de couverture', () => {
    renderWithProviders(<SynthesisWeaponRangeSection range={RANGE} />)
    expect(
      textOf(/^Sous le seuil de 8 mesures — frags : Hydra \(6\), Disrupteur \(4\) · morts : Ravageur \(5\)$/).length,
    ).toBeGreaterThan(0)
    expect(
      screen.getByText(/Ne compte que les frags et les morts dont la position du tueur ET de la victime/),
    ).toBeInTheDocument()
  })

  it('le tableau déplié redit les deux côtés, avec un tiret là où rien n’est mesuré', () => {
    renderWithProviders(<SynthesisWeaponRangeSection range={RANGE} />)
    expect(screen.getByText('Voir en tableau')).toBeInTheDocument()
    const table = screen.getByRole('table')
    const commando = within(table).getByText('Commando VK78').closest('tr')
    expect(commando).not.toBeNull()
    const cells = Array.from(commando!.querySelectorAll('td')).map((c) => flat(c.textContent))
    expect(cells[0]).toBe('Commando VK78')
    expect(cells.slice(1, 7)).toEqual(['64', '6,3 m', '11,9 m', '21,4 m', '30 %', '20 %'])
    // Les six colonnes du côté « morts » sont vides pour cette arme, pas à zéro.
    expect(cells.slice(7)).toEqual(['—', '—', '—', '—', '—', '—'])
  })
})

describe('SynthesisWeaponRangeSection — dégradations', () => {
  it('entame mesurée mais AUCUN frag apparié : la distance d’entame reste, le delta disparaît', () => {
    // Les deux absences disent deux choses différentes : `opening` absent = aucune entame
    // mesurée ; `opening.delta` absent = aucune entame appariée à son frag.
    renderWithProviders(
      <SynthesisWeaponRangeSection
        range={{ ...RANGE, opening: { median_m: 9.1, measured_kills: 618 } }}
      />,
    )
    expect(screen.getByText("Distance d'entame médiane")).toBeInTheDocument()
    expect(textOf(/^1,5 s avant le frag · 618 frags mesurés$/).length).toBeGreaterThan(0)
    expect(screen.queryByText('Entame → frag')).not.toBeInTheDocument()
    expect(screen.queryByText(/Vous fermez la distance/)).not.toBeInTheDocument()
  })

  it('sans entame mesurée, les DEUX tuiles d’entame disparaissent (jamais un zéro)', () => {
    renderWithProviders(<SynthesisWeaponRangeSection range={{ ...RANGE, opening: undefined }} />)
    expect(screen.queryByText("Distance d'entame médiane")).not.toBeInTheDocument()
    expect(screen.queryByText('Entame → frag')).not.toBeInTheDocument()
    // Les deux tuiles de portée, elles, restent : la portée ne dépend pas de l'entame.
    expect(screen.getByText('Portée médiane de mes frags')).toBeInTheDocument()
  })

  it('sans arme sous le seuil, la ligne « sous le seuil » n’est pas rendue', () => {
    renderWithProviders(
      <SynthesisWeaponRangeSection
        range={{ ...RANGE, below_threshold_kills: [], below_threshold_deaths: null }}
      />,
    )
    expect(screen.queryByText(/Sous le seuil/)).not.toBeInTheDocument()
    // La note de couverture, elle, reste : la mesure est partielle dans tous les cas.
    expect(screen.getByText(/couverture partielle/)).toBeInTheDocument()
  })

  it('un seul côté sous le seuil : ce demi-énoncé seul, jamais « morts : » à vide', () => {
    renderWithProviders(
      <SynthesisWeaponRangeSection range={{ ...RANGE, below_threshold_deaths: [] }} />,
    )
    expect(
      textOf(/^Sous le seuil de 8 mesures — frags : Hydra \(6\), Disrupteur \(4\)$/).length,
    ).toBeGreaterThan(0)
    expect(screen.queryByText(/morts : /)).not.toBeInTheDocument()
  })

  it('sans bloc du tout, la section entière se retire', () => {
    const { container: nul } = renderWithProviders(<SynthesisWeaponRangeSection range={undefined} />)
    expect(nul).toBeEmptyDOMElement()
    const { container: vide } = renderWithProviders(<SynthesisWeaponRangeSection range={null} />)
    expect(vide).toBeEmptyDOMElement()
  })

  it('TOUTES les armes sous le seuil : la section reste, les graphes cèdent la place à leur raison', () => {
    // Cas nominal (consigne du pilote, revue du lot 4) : `weapons` vide MAIS des compteurs
    // et des médianes bien réels, et les deux listes « sous le seuil » remplies.
    renderWithProviders(
      <SynthesisWeaponRangeSection
        range={{
          ...RANGE,
          weapons: [],
          below_threshold_kills: [{ weapon_key: 'hinf_hydra', label: 'Hydra', label_en: 'Hydra', measured: 6 }],
          below_threshold_deaths: [{ weapon_key: 'hinf_ravager', label: 'Ravageur', label_en: 'Ravager', measured: 5 }],
        }}
      />,
    )
    // Les tuiles restent : les médianes globales ne dépendent pas du seuil de publication.
    expect(screen.getByText('Portée médiane de mes frags')).toBeInTheDocument()
    expect(textOf(/^1 214 frags mesurés sur 1 602$/).length).toBeGreaterThan(0)
    // Aucun graphe, mais une phrase qui DIT pourquoi — jamais un canevas vide sans mot.
    expect(screen.queryAllByTestId('chart-card')).toHaveLength(0)
    expect(
      screen.getByText(/Aucune arme n'atteint le seuil de 8 mesures sur cette période/),
    ).toBeInTheDocument()
    // Les armes écartées sont nommées, la note de couverture reste, le tableau disparaît
    // (il n'aurait aucune ligne à redire).
    expect(
      textOf(/^Sous le seuil de 8 mesures — frags : Hydra \(6\) · morts : Ravageur \(5\)$/).length,
    ).toBeGreaterThan(0)
    expect(screen.getByText(/couverture partielle/)).toBeInTheDocument()
    expect(screen.queryByText('Voir en tableau')).not.toBeInTheDocument()
    expect(screen.queryByRole('table')).not.toBeInTheDocument()
  })

  it('en anglais, libellés et nombres suivent la locale', () => {
    useAppShellStore.setState({ locale: 'en' })
    renderWithProviders(<SynthesisWeaponRangeSection range={RANGE} />)
    expect(screen.getByText('Median range of my kills')).toBeInTheDocument()
    expect(textOf(/^1,214 measured kills out of 1,602$/).length).toBeGreaterThan(0)
    expect(screen.getByText('BR75 Battle Rifle')).toBeInTheDocument()
  })
})
