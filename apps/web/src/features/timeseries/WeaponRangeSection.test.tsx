/**
 * WeaponRangeSection.test — la section « Portée ».
 *
 * Ce que ces tests verrouillent : les quatre tuiles SANS dénominateur, les DEUX tuiles
 * d'entame qui n'apparaissent QUE si l'entame est mesurée (jamais un zéro — décision D5), la
 * carte « Portée par arme » en demi-largeur avec sa voisine `aside`, sa légende (encres, position en pied), le tableau
 * dépliable (les deux côtés, le tiret du côté non mesuré), et le retrait complet de la
 * section quand rien n'est publiable.
 *
 * Le graphe est testé PUR (`components/charts/weaponRangeChart.test.ts`) — ici ECharts est
 * mocké (jsdom ne peint pas de canvas).
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { screen, within } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import { tokenCssVar } from '@/lib/accessibility'
import type { SynthesisWeaponRange, WeaponRangeSide } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'

import { WeaponRangeSection } from './WeaponRangeSection'

vi.mock('echarts-for-react', () => ({
  default: () => <div data-testid="echarts-mock" />,
}))

const side = (o: Partial<WeaponRangeSide>): WeaponRangeSide => ({
  measured: 10,
  p10: 5,
  median: 7,
  p90: 10,
  // min_m / max_m : ajoutés au contrat le 2026-09-17 (profil d'armes du Face-à-face).
  // Publiés pour l'infobulle, JAMAIS tracés — la Synthèse les ignore. Ils encadrent
  // simplement le bâton p10 -> p90 de cette fixture.
  min_m: 2,
  max_m: 14,
  above_pct: 30,
  level_pct: 50,
  below_pct: 20,
  ...o,
})

const RANGE: SynthesisWeaponRange = {
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

/** Le texte de la tuile (`AccentCard`) dont le LIBELLÉ est `label` — valeur et dénominateur
 *  compris. Remonter au conteneur est la seule façon de lier une valeur à SA tuile. */
function flatCardOf(label: string): string {
  const card = screen.getByText(label).closest('div.rounded-lg')
  expect(card).not.toBeNull()
  return flat(card!.textContent)
}

function textOf(re: RegExp): string[] {
  return screen
    .getAllByText((_, node) => (node ? re.test(flat(node.textContent)) : false))
    .map((n) => flat(n.textContent))
}

beforeEach(() => {
  useAppShellStore.setState({ locale: 'fr' })
})

describe('WeaponRangeSection — rendu nominal', () => {
  // « Hauteur d'engagement » a quitté la page (V5 du plan PLAN_TIMESERIES_USAGES_EMPRISE_2026-10-05) :
  // « Portée par arme » reste en demi-largeur dans sa grille à deux colonnes (D11) ; la seconde
  // colonne reçoit la carte passée en `aside`.
  it('pose la carte `aside` dans la même rangée que « Portée par arme »', () => {
    renderWithProviders(
      <WeaponRangeSection range={RANGE} aside={<section aria-label="Voisine">voisine</section>} />,
    )
    const card = screen.getByRole('region', { name: 'Portée par arme' })
    const voisine = screen.getByRole('region', { name: 'Voisine' })
    expect(voisine.parentElement).toBe(card.parentElement)
  })

  it('affiche « Portée par arme » seule, en demi-largeur, avec son graphe', () => {
    renderWithProviders(<WeaponRangeSection range={RANGE} />)
    const card = screen.getByRole('region', { name: 'Portée par arme' })
    expect(card).toBeInTheDocument()
    expect(screen.queryByRole('region', { name: "Hauteur d'engagement" })).not.toBeInTheDocument()
    // Une ChartCard : le canvas lui-même est chargé en `lazy`, on pince la carte.
    expect(screen.getAllByTestId('chart-card')).toHaveLength(1)
    expect(card.closest('.lg\\:grid-cols-2')).not.toBeNull()
  })

  it('les quatre tuiles portent leur valeur, SANS dénominateur de couverture', () => {
    renderWithProviders(<WeaponRangeSection range={RANGE} />)
    expect(screen.getByText('Portée médiane des frags du joueur')).toBeInTheDocument()
    expect(screen.getByText('Portée médiane des morts du joueur')).toBeInTheDocument()
    // Retour utilisateur du 2026-10-07 : ni « N frags mesurés sur M », ni « N morts mesurées
    // sur M », ni « 1,5 s avant le frag · N frags mesurés ».
    expect(screen.queryByText(/mesurés sur|mesurées sur|frags mesurés/)).not.toBeInTheDocument()
    // LA VALEUR, PAS SEULEMENT SON LIBELLÉ : chaque tuile porte SA médiane. Sans ces deux
    // lignes, deux valeurs échangées (frags <-> morts) passeraient inaperçues.
    expect(flatCardOf('Portée médiane des frags du joueur')).toContain('7,4 m')
    expect(flatCardOf('Portée médiane des morts du joueur')).toContain('11,8 m')
    expect(screen.getByText("Distance d'entame médiane")).toBeInTheDocument()
    expect(screen.getByText('Entame → frag')).toBeInTheDocument()
    expect(textOf(/^Distance réduite avant le frag dans 61 % des frags$/).length).toBeGreaterThan(0)
    // Distance signée : le signe DIT le sens (la distance se ferme), il n'est pas décoratif.
    expect(textOf(/^-1,7 m$/).length).toBeGreaterThan(0)
  })

  it('la légende nomme ses séries, SANS mention de position', () => {
    // Les mentions « (bâton du haut) » / « (bâton du bas, l'arme est celle du tueur) » ont
    // été retirées le 2026-09-09 : elles doublaient un ordre déjà lisible sur le graphe et
    // faisaient une ligne de légende deux fois plus longue que la légende.
    renderWithProviders(<WeaponRangeSection range={RANGE} />)
    // `within` la légende : « Frags du joueur » nomme AUSSI un groupe de colonnes du tableau.
    const legend = screen.getByRole('list', { name: 'Légende' })
    expect(within(legend).getByText('Frags du joueur')).toBeInTheDocument()
    expect(within(legend).getByText('Morts du joueur')).toBeInTheDocument()
    expect(screen.queryByText(/bâton du haut/)).not.toBeInTheDocument()
    expect(screen.queryByText(/bâton du bas/)).not.toBeInTheDocument()
  })

  it('la légende est rendue en PIED de son graphe, par le composant commun', () => {
    // « Ça ne suit pas la nomenclature des autres graphes » (2026-09-09) : les deux légendes
    // passe par <ChartLegend>, posé dans le pied de carte de sa ChartCard.
    renderWithProviders(<WeaponRangeSection range={RANGE} />)
    const legends = screen.getAllByTestId('chart-legend')
    expect(legends).toHaveLength(1)
    for (const legend of legends) {
      expect(legend.className).toContain('justify-center')
      expect(legend.closest('[data-testid="chart-card-legend"]')).not.toBeNull()
    }
  })

  it('les pastilles de la légende de PORTÉE portent l’encre de leur côté', () => {
    // Lot 6, item 6.0d. Sans ce test, échanger les deux `tokenCssVar` de `RangeLegend` laissait la suite verte : la légende
    // aurait annoncé les frags à l'encre des morts, et rien n'aurait mordu — alors que c'est
    // la légende qui dit au lecteur quel bâton est lequel.
    renderWithProviders(<WeaponRangeSection range={RANGE} />)
    const legend = screen.getByRole('list', { name: 'Légende' })
    const swatch = (name: string) =>
      within(legend).getByText(name).parentElement!.querySelector('span[aria-hidden]') as HTMLElement
    // Famille des stats de combat (2026-09-17) : les mêmes encres que partout dans l'app.
    expect(swatch('Frags du joueur').style.backgroundColor).toBe(tokenCssVar('stat-kills'))
    expect(swatch('Morts du joueur').style.backgroundColor).toBe(tokenCssVar('stat-deaths'))
  })

  it('les deux tuiles de portée portent l’accent de leur côté', () => {
    // Le filet de 3 px en tête de tuile est le SEUL rappel de couleur entre la tuile et son
    // bâton : les valeurs sont déjà épinglées, l'ENCRE ne l'était pas (lot 6, item 6.0d).
    // Échanger `accent={KILLS_TOKEN}` et `accent={DEATHS_TOKEN}` laissait la suite verte.
    renderWithProviders(<WeaponRangeSection range={RANGE} />)
    const accentOf = (label: string) =>
      (screen.getByText(label).closest('div.rounded-lg')!.firstElementChild as HTMLElement).style
        .backgroundColor
    expect(accentOf('Portée médiane des frags du joueur')).toBe(tokenCssVar('stat-kills'))
    expect(accentOf('Portée médiane des morts du joueur')).toBe(tokenCssVar('stat-deaths'))
  })

  it('le tableau groupe ses colonnes : mes frags D’ABORD, mes morts ensuite', () => {
    renderWithProviders(<WeaponRangeSection range={RANGE} />)
    const groups = Array.from(
      screen.getByRole('table').querySelectorAll('thead tr:first-child th'),
    ).map((th) => flat(th.textContent))
    // La première cellule est le coin vide au-dessus de la colonne « Arme ».
    expect(groups).toEqual(['', 'Frags du joueur', 'Morts du joueur'])
  })

  it('ne publie plus ni la ligne « sous le seuil » ni la note de couverture', () => {
    // Retirées le 2026-09-09 (demande utilisateur) : deux paragraphes de texte gris sous la
    // carte, qui répétaient une réserve déjà portée par les dénominateurs de chaque tuile.
    renderWithProviders(<WeaponRangeSection range={RANGE} />)
    expect(screen.queryByText(/Sous le seuil/)).not.toBeInTheDocument()
    expect(screen.queryByText(/couverture partielle/)).not.toBeInTheDocument()
  })

  it('le tableau déplié redit les deux côtés, avec un tiret là où rien n’est mesuré', () => {
    renderWithProviders(<WeaponRangeSection range={RANGE} />)
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

describe('WeaponRangeSection — dégradations', () => {
  it('entame mesurée mais AUCUN frag apparié : la distance d’entame reste, le delta disparaît', () => {
    // Les deux absences disent deux choses différentes : `opening` absent = aucune entame
    // mesurée ; `opening.delta` absent = aucune entame appariée à son frag.
    renderWithProviders(
      <WeaponRangeSection
        range={{ ...RANGE, opening: { median_m: 9.1, measured_kills: 618 } }}
      />,
    )
    expect(screen.getByText("Distance d'entame médiane")).toBeInTheDocument()
    expect(screen.queryByText('Entame → frag')).not.toBeInTheDocument()
    expect(screen.queryByText(/Distance réduite avant le frag/)).not.toBeInTheDocument()
  })

  it('sans entame mesurée, les DEUX tuiles d’entame disparaissent (jamais un zéro)', () => {
    renderWithProviders(<WeaponRangeSection range={{ ...RANGE, opening: undefined }} />)
    expect(screen.queryByText("Distance d'entame médiane")).not.toBeInTheDocument()
    expect(screen.queryByText('Entame → frag')).not.toBeInTheDocument()
    // Les deux tuiles de portée, elles, restent : la portée ne dépend pas de l'entame.
    expect(screen.getByText('Portée médiane des frags du joueur')).toBeInTheDocument()
  })

  it('un côté sans aucune mesure : sa tuile de portée DISPARAÎT, jamais « 0,0 m »', () => {
    // Scope « morts seulement » (un filtre où le joueur n'a fragué personne) : le service ne
    // retire le bloc que si les DEUX côtés sont vides, et sert le côté absent à zéro. Une
    // tuile « 0,0 m » se lirait « il frague au contact » — la MÊME doctrine que l'entame (D5).
    const { container } = renderWithProviders(
      <WeaponRangeSection
        range={{
          ...RANGE,
          weapons: [{ ...RANGE.weapons![0], kills: undefined }],
          median_kills_m: 0,
          measured_kills: 0,
          total_kills: 0,
          below_threshold_kills: [],
          opening: undefined,
        }}
      />,
    )
    expect(screen.queryByText('Portée médiane des frags du joueur')).not.toBeInTheDocument()
    expect(screen.getByText('Portée médiane des morts du joueur')).toBeInTheDocument()
    expect(flat(container.textContent)).not.toContain('0,0 m')
  })

  it('l’autre côté vide : la tuile des morts disparaît, celle des frags reste', () => {
    renderWithProviders(
      <WeaponRangeSection
        range={{
          ...RANGE,
          weapons: [{ ...RANGE.weapons![0], deaths: undefined }],
          median_deaths_m: 0,
          measured_deaths: 0,
          total_deaths: 0,
          below_threshold_deaths: [],
        }}
      />,
    )
    expect(screen.getByText('Portée médiane des frags du joueur')).toBeInTheDocument()
    expect(screen.queryByText('Portée médiane des morts du joueur')).not.toBeInTheDocument()
  })

  it('entame qui ÉLOIGNE : le signe + est écrit, il n’est pas décoratif', () => {
    // Miroir du cas nominal (delta négatif) : `signDisplay: 'exceptZero'` doit écrire le
    // signe des DEUX côtés, sans quoi « 1,7 m » ne dirait pas si la distance s'ouvre ou se ferme.
    renderWithProviders(
      <WeaponRangeSection
        range={{
          ...RANGE,
          opening: { median_m: 9.1, measured_kills: 618, delta: { median_m: 1.7, closing_share_pct: 38.6, n: 574 } },
        }}
      />,
    )
    expect(flatCardOf('Entame → frag')).toContain('+1,7 m')
  })

  it('sans bloc du tout, la section entière se retire', () => {
    const { container: nul } = renderWithProviders(<WeaponRangeSection range={undefined} />)
    expect(nul).toBeEmptyDOMElement()
    const { container: vide } = renderWithProviders(<WeaponRangeSection range={null} />)
    expect(vide).toBeEmptyDOMElement()
  })

  it('TOUTES les armes sous le seuil : la section reste, les graphes cèdent la place à leur raison', () => {
    // Cas nominal (consigne du pilote, revue du lot 4) : `weapons` vide MAIS des compteurs
    // et des médianes bien réels, et les deux listes « sous le seuil » remplies.
    renderWithProviders(
      <WeaponRangeSection
        range={{
          ...RANGE,
          weapons: [],
          below_threshold_kills: [{ weapon_key: 'hinf_hydra', label: 'Hydra', label_en: 'Hydra', measured: 6 }],
          below_threshold_deaths: [{ weapon_key: 'hinf_ravager', label: 'Ravageur', label_en: 'Ravager', measured: 5 }],
        }}
      />,
    )
    // Les tuiles restent : les médianes globales ne dépendent pas du seuil de publication.
    expect(screen.getByText('Portée médiane des frags du joueur')).toBeInTheDocument()
    // Aucun graphe, mais une phrase qui DIT pourquoi — jamais un canevas vide sans mot.
    expect(screen.queryAllByTestId('chart-card')).toHaveLength(0)
    // LA PORTÉE dit son seuil.
    expect(
      screen.getByText(/Aucune arme n'atteint 8 frags ou 8 morts sur cette période/),
    ).toBeInTheDocument()
    // Le tableau disparaît aussi : il n'aurait aucune ligne à redire.
    expect(screen.queryByText('Voir en tableau')).not.toBeInTheDocument()
    expect(screen.queryByRole('table')).not.toBeInTheDocument()
  })

  it('en anglais, libellés et nombres suivent la locale', () => {
    useAppShellStore.setState({ locale: 'en' })
    renderWithProviders(<WeaponRangeSection range={RANGE} />)
    expect(screen.getByText("Median range of the player's kills")).toBeInTheDocument()
    expect(screen.getByText('BR75 Battle Rifle')).toBeInTheDocument()
    // LES DISTANCES AUSSI suivent la locale : « 7.4 m » et non « 7,4 m ». Sans cette ligne,
    // un formateur figé sur fr-FR passerait le test anglais.
    expect(flatCardOf("Median range of the player's kills")).toContain('7.4 m')
    expect(flatCardOf("Median range of the player's deaths")).toContain('11.8 m')
  })
})
