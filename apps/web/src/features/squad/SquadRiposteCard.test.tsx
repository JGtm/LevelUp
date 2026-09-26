/**
 * La RIPOSTE — les TROIS BLOCS que six cartes racontaient chacune leur tour.
 *
 * Ce que ces tests cadenassent (D19, 2026-09-21 ; D22-verbosité, 2026-09-21 ; découpe en
 * trois blocs, 2026-09-22) :
 *   - AUCUNE phrase de lecteur : l'explication vit dans l'infobulle (i) de CHAQUE bloc ;
 *   - le donut dit le taux et ses deux parts, cohérents avec la couverture servie ;
 *   - RIEN n'est écrit sous le donut ni sous l'histogramme — ni écart, ni médiane, ni
 *     note de pied : tout est dans le graphe ou dans son (i) ;
 *   - une SEULE soirée rend UN SEUL bâton — plus aucun repli en liste ;
 *   - l’unique repli restant (la matrice) est FERMÉ par défaut.
 */
import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'
import type { SquadEchange } from '@/lib/api/types'

import { SquadRiposteCard } from './SquadRiposteCard'
import { couverture, echangeDe } from './squadRiposte.fixtures'

beforeEach(() => useAppShellStore.setState({ locale: 'fr' }))
afterEach(() => useAppShellStore.setState({ locale: 'fr' }))

/** Une section avec N soirées mesurées, habituel à 20 %. */
function avecSoirees(taux: number[]): SquadEchange {
  return echangeDe({
    couverture: couverture(99, 511, 128),
    habituel: couverture(20, 100),
    matchs_total: 128,
    matchs_habituel: 400,
    delai_median_ms: 2400,
    taux_par_session: taux.map((t, i) => ({
      session_label: `1${i}/09 22:00–23:00 (4)`,
      matchs_mesures: 4,
      dans_le_filtre: true,
      couverture: {
        taux: t,
        brut: Math.round(t * 60),
        par_match: 1,
        n: 60,
        echantillon_faible: false,
      },
    })),
  } as Partial<SquadEchange>)
}

describe('SquadRiposteCard', () => {
  it('ne rend AUCUNE phrase de lecteur (D22-verbosité) — le chiffre d’appel reste', () => {
    renderWithProviders(<SquadRiposteCard echange={avecSoirees([0.3, 0.1])} />)
    expect(screen.queryByTestId('squad-riposte-phrase')).toBeNull()
    expect(screen.getByTestId('squad-riposte-taux')).toBeTruthy()
  })

  it('le taux est cohérent avec la couverture servie', () => {
    renderWithProviders(<SquadRiposteCard echange={avecSoirees([0.3])} />)
    expect(screen.getByTestId('squad-riposte-taux').textContent ?? '').toContain('19,4')
  })

  it('n’écrit RIEN sous le donut : plus d’écart à l’habituel en toutes lettres', () => {
    renderWithProviders(<SquadRiposteCard echange={avecSoirees([0.3])} />)
    expect(screen.queryByTestId('squad-riposte-ecart')).toBeNull()
    // Le taux, lui, reste : il vit au centre du donut (et dans son texte `sr-only`).
    expect(screen.getByTestId('squad-riposte-taux')).toBeTruthy()
  })

  it('n’écrit RIEN sous l’histogramme : ni légende de médiane, ni note de pied', () => {
    renderWithProviders(<SquadRiposteCard echange={avecSoirees([0.3])} />)
    expect(screen.queryByTestId('squad-riposte-delai-median')).toBeNull()
    const delai = screen.getByTestId('squad-riposte-delai').textContent ?? ''
    expect(delai).not.toMatch(/Fenêtre de riposte/i)
    expect(delai).not.toMatch(/hachurée/i)
  })

  it('monte TROIS blocs titrés : « Morts ripostées », « Temps de riposte », « Riposte »', () => {
    renderWithProviders(<SquadRiposteCard echange={avecSoirees([0.3])} />)
    expect(screen.getByText('Morts ripostées')).toBeTruthy()
    expect(screen.getByText('Temps de riposte')).toBeTruthy()
    // Chacun des deux blocs de tête porte SON aide (i) : la méthode y a sa place.
    expect(screen.getByTestId('squad-riposte-donut-aide')).toBeTruthy()
    expect(screen.getByTestId('squad-riposte-delai-aide')).toBeTruthy()
  })

  it('UNE SEULE SOIRÉE rend un graphe, jamais une liste de définitions', () => {
    const { container } = renderWithProviders(<SquadRiposteCard echange={avecSoirees([0.3])} />)
    expect(container.querySelector('[data-testid="chart-card"]')).toBeTruthy()
    expect(container.querySelector('dl')).toBeNull()
    expect(container.querySelector('[data-testid="chart-card-empty"]')).toBeNull()
  })

  it('ouvre son UNIQUE repli FERMÉ — celui du délai a disparu avec son histogramme', () => {
    renderWithProviders(<SquadRiposteCard echange={avecSoirees([0.3])} />)
    const repli = screen.getByTestId('squad-riposte-fold-matrice') as HTMLDetailsElement
    expect(repli.open).toBe(false)
    expect(screen.queryByTestId('squad-riposte-fold-delai')).toBeNull()
  })

  it('rend DEUX GRAPHES de tête : le donut des parts et la distribution du délai', () => {
    renderWithProviders(<SquadRiposteCard echange={avecSoirees([0.3])} />)
    // Le donut porte les DEUX parts exclusives : 99 ripostées, 412 sans réponse sur 511.
    const alt = screen.getByTestId('squad-riposte-taux').textContent ?? ''
    expect(alt).toContain('99')
    expect(alt).toContain('412')
    expect(alt).toContain('511')
    expect(screen.getByTestId('squad-riposte-delai')).toBeTruthy()
  })

  it('PARITÉ FR/EN : les deux langues rendent deux titres de bloc distincts', () => {
    const { unmount } = renderWithProviders(<SquadRiposteCard echange={avecSoirees([0.3])} />)
    expect(screen.getByText('Morts ripostées')).toBeTruthy()
    unmount()
    useAppShellStore.setState({ locale: 'en' })
    renderWithProviders(<SquadRiposteCard echange={avecSoirees([0.3])} />)
    expect(screen.getByText('Riposted deaths')).toBeTruthy()
    expect(screen.getByText('Riposte time')).toBeTruthy()
    expect(screen.queryByText(/squad\.riposte\./)).toBeNull()
  })
})
