/**
 * La TUILE « Taux d'échange » du bandeau de KPI.
 *
 * Ce que ces tests cadenassent (correction W3, revue du 2026-09-06 — la tuile n'avait
 * AUCUN test : supprimer le masquage plein-historique ou inverser le signe de l'écart
 * passait sans qu'aucune assertion ne bouge) :
 *
 *   - rien du tout quand la section est absente du contrat ;
 *   - les trois grandeurs ensemble : valeur, « N vengées sur M », écart en points ;
 *   - l'écart et sa flèche se TAISENT sur tout l'historique ;
 *   - le signe de l'écart suit bien le sens de la comparaison.
 */
import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import type { TeammatesPageResponse } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'

import { SquadContext } from './SquadContext'
import { SquadEchangeKpi } from './SquadEchangeKpi'
import { couverture, echangeDe } from './squadEchange.fixtures'

beforeEach(() => useAppShellStore.setState({ locale: 'fr' }))
afterEach(() => useAppShellStore.setState({ locale: 'fr' }))

function rendreAvec(pageData: TeammatesPageResponse | null) {
  return renderWithProviders(
    <SquadContext.Provider
      value={{
        selectedRows: [],
        confirmedGamertags: [],
        pageData,
        playerSlug: 'moi',
        currentPlayerXuid: 'x1',
      }}
    >
      <SquadEchangeKpi />
    </SquadContext.Provider>,
  )
}

const page = (echange: TeammatesPageResponse['echange']): TeammatesPageResponse =>
  ({ echange }) as TeammatesPageResponse

describe('SquadEchangeKpi', () => {
  it('ne rend RIEN quand la section est absente du contrat', () => {
    const { container } = rendreAvec(page(undefined))
    expect(container.textContent).toBe('')
  })

  it('ne rend RIEN sans pageData', () => {
    const { container } = rendreAvec(null)
    expect(container.textContent).toBe('')
  })

  it('sert la valeur ET le compte brut — jamais le taux seul', () => {
    rendreAvec(page(echangeDe({ couverture: couverture(18, 45), habituel: couverture(40, 100) })))
    expect(screen.getByTestId('kpi-primary').textContent).toContain('40')
    const secondaire = screen.getByTestId('kpi-secondary').textContent ?? ''
    expect(secondaire).toContain('18')
    expect(secondaire).toContain('45')
  })

  it('MASQUE l’écart et la flèche quand le périmètre couvre tout l’historique', () => {
    // Supprimer le `pleinHistorique ?` afficherait « ±0 pts vs habituel » — une
    // tautologie présentée comme une mesure.
    rendreAvec(page(echangeDe({ matchs_total: 60, matchs_habituel: 60 })))
    expect(screen.queryByTestId('squad-echange-kpi-delta')).toBeNull()
    expect(screen.queryByTestId('kpi-trend')).toBeNull()
  })

  it('affiche un écart POSITIF et une flèche montante au-dessus de l’habituel', () => {
    rendreAvec(page(echangeDe({ couverture: couverture(27, 45), habituel: couverture(40, 100) })))
    expect(screen.getByTestId('squad-echange-kpi-delta').textContent).toContain('+20 pts')
    expect(screen.getByTestId('kpi-trend').getAttribute('data-trend')).toBe('above')
  })

  it('affiche un écart NÉGATIF et une flèche descendante en dessous de l’habituel', () => {
    // Inverser le signe de la soustraction fait tomber ce test.
    rendreAvec(page(echangeDe({ couverture: couverture(9, 45), habituel: couverture(40, 100) })))
    const delta = screen.getByTestId('squad-echange-kpi-delta').textContent ?? ''
    expect(delta).toContain('20 pts')
    expect(delta.includes('+')).toBe(false)
    expect(screen.getByTestId('kpi-trend').getAttribute('data-trend')).toBe('below')
  })

  it('pose la réserve « échantillon faible » AVEC la valeur, sans la cacher', () => {
    rendreAvec(page(echangeDe({ couverture: couverture(8, 8, 3), habituel: couverture(40, 100) })))
    expect(screen.getByTestId('kpi-primary').textContent).toContain('100')
    expect(screen.getByTestId('kpi-secondary').textContent).toMatch(/échantillon faible/i)
  })
})
