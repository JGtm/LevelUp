/**
 * La carte « Riposte » — le récit que six blocs racontaient chacun leur tour.
 *
 * Ce que ces tests cadenassent (D19, 2026-09-21) :
 *   - la PHRASE DE LECTEUR est TOUJOURS rendue (l'ancien « Constat » se taisait sous 5
 *     points d'écart, faisant disparaître la seule carte en français) ;
 *   - le chiffre d'appel dit le taux ET le délai, cohérents avec la couverture servie ;
 *   - l'écart se TAIT sur tout l'historique (tautologie, pas mesure) ;
 *   - une SEULE soirée rend UN SEUL bâton — plus aucun repli en liste ;
 *   - les deux replis sont FERMÉS par défaut.
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
  it('rend TOUJOURS la phrase de lecteur, chiffrée', () => {
    renderWithProviders(<SquadRiposteCard echange={avecSoirees([0.3, 0.1])} />)
    const phrase = screen.getByTestId('squad-riposte-phrase').textContent ?? ''
    // 19,4 % ≈ une fois sur cinq, délai médian 2,4 s.
    expect(phrase).toMatch(/une fois sur 5/i)
    expect(phrase).toContain('2,4')
    expect(phrase).toMatch(/5 secondes/)
  })

  it('le chiffre d’appel et son écart sont cohérents avec la couverture servie', () => {
    renderWithProviders(<SquadRiposteCard echange={avecSoirees([0.3])} />)
    expect(screen.getByTestId('squad-riposte-taux').textContent ?? '').toContain('19,4')
    // 19,4 % contre un habituel de 20,0 % : un point sous l'habituel.
    const ecart = screen.getByTestId('squad-riposte-ecart').textContent ?? ''
    expect(ecart).toMatch(/1 pts/)
    expect(ecart).toMatch(/sous l’habituel|sous l'habituel/)
    expect(screen.getByTestId('squad-riposte-delai-median').textContent ?? '').toContain('2,4')
  })

  it('TAIT l’écart quand le filtre couvre tout l’historique (tautologie)', () => {
    const plein = echangeDe({ matchs_total: 60, matchs_habituel: 60 })
    renderWithProviders(<SquadRiposteCard echange={plein} />)
    expect(screen.queryByTestId('squad-riposte-ecart')).toBeNull()
    // La phrase, elle, reste : le fait ne dépend pas de l'écart.
    expect(screen.getByTestId('squad-riposte-phrase')).toBeTruthy()
  })

  it('UNE SEULE SOIRÉE rend un graphe, jamais une liste de définitions', () => {
    const { container } = renderWithProviders(<SquadRiposteCard echange={avecSoirees([0.3])} />)
    expect(container.querySelector('[data-testid="chart-card"]')).toBeTruthy()
    expect(container.querySelector('dl')).toBeNull()
    expect(container.querySelector('[data-testid="chart-card-empty"]')).toBeNull()
  })

  it('ouvre ses deux replis FERMÉS', () => {
    renderWithProviders(<SquadRiposteCard echange={avecSoirees([0.3])} />)
    for (const id of ['squad-riposte-fold-delai', 'squad-riposte-fold-matrice']) {
      const repli = screen.getByTestId(id) as HTMLDetailsElement
      expect(repli.open).toBe(false)
    }
  })

  it('PARITÉ FR/EN : les deux langues rendent une phrase, et deux phrases différentes', () => {
    const { unmount } = renderWithProviders(<SquadRiposteCard echange={avecSoirees([0.3])} />)
    const fr = screen.getByTestId('squad-riposte-phrase').textContent ?? ''
    unmount()
    useAppShellStore.setState({ locale: 'en' })
    renderWithProviders(<SquadRiposteCard echange={avecSoirees([0.3])} />)
    const en = screen.getByTestId('squad-riposte-phrase').textContent ?? ''
    expect(fr.length).toBeGreaterThan(0)
    expect(en).not.toBe(fr)
    expect(en.includes('squad.riposte.')).toBe(false)
  })
})
