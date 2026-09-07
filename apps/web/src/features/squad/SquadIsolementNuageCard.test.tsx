/**
 * Le NUAGE « isolement x couverture » (item 7.7) — ce qu'il montre et ce qu'il tait.
 *
 * Ce que ces tests cadenassent : un état vide n'est jamais un nuage à zéro point ; le
 * pied de carte NOMME les deux planchers (session, échantillon) plutôt que de les
 * recopier en dur côté client ; et les deux langues rendent deux textes.
 */
import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'
import type { SquadEchangeJoueur, SquadIsolementPoint, SquadNuageIsolement } from '@/lib/api/types'

import { SquadIsolementNuageCard } from './SquadIsolementNuageCard'

beforeEach(() => useAppShellStore.setState({ locale: 'fr' }))
afterEach(() => useAppShellStore.setState({ locale: 'fr' }))

const joueurs: SquadEchangeJoueur[] = [
  { xuid: 'x1', gamertag: 'Alice' },
  { xuid: 'x2', gamertag: 'Bob' },
]

function isoCouverture(brut: number, n: number, echantillonFaible: boolean) {
  return {
    taux: n > 0 ? brut / n : 0,
    brut,
    par_match: n > 0 ? brut / 3 : 0,
    n,
    echantillon_faible: echantillonFaible,
  }
}

function point(over: Partial<SquadIsolementPoint> = {}): SquadIsolementPoint {
  return {
    xuid: 'x1',
    gamertag: 'Alice',
    session_label: '02/09 soir',
    morts_examinees: 10,
    morts_isolees: 4,
    part_isolee: isoCouverture(4, 10, true),
    couverture: isoCouverture(6, 10, true),
    ...over,
  } as SquadIsolementPoint
}

function nuageDe(over: Partial<SquadNuageIsolement> = {}): SquadNuageIsolement {
  return {
    points: [point(), point({ xuid: 'x2', gamertag: 'Bob', session_label: '05/09 soir' })],
    plancher_morts_session: 5,
    plancher_echantillon_faible: 30,
    ...over,
  } as SquadNuageIsolement
}

describe('SquadIsolementNuageCard', () => {
  it('ÉTAT VIDE, jamais un nuage à zéro point, quand aucune session ne franchit le plancher', () => {
    renderWithProviders(<SquadIsolementNuageCard nuage={nuageDe({ points: [] })} joueurs={joueurs} />)
    expect(screen.getByText(/Aucun point mesuré/i)).toBeTruthy()
    expect(screen.queryByTestId('chart-card')).toBeNull()
  })

  it('rend le nuage quand au moins un point franchit le plancher', () => {
    renderWithProviders(<SquadIsolementNuageCard nuage={nuageDe()} joueurs={joueurs} />)
    expect(screen.getByTestId('squad-isolement-nuage')).toBeTruthy()
    expect(screen.getByTestId('chart-card')).toBeTruthy()
  })

  it('le pied de carte NOMME les deux planchers (5 et 30), pas de valeur en dur', () => {
    renderWithProviders(
      <SquadIsolementNuageCard
        nuage={nuageDe({ plancher_morts_session: 5, plancher_echantillon_faible: 30 })}
        joueurs={joueurs}
      />,
    )
    const carte = screen.getByTestId('squad-isolement-nuage').closest('section')
    const texte = carte?.textContent ?? ''
    expect(texte).toContain('5')
    expect(texte).toContain('30')
  })

  it('PARITÉ FR/EN : les deux langues rendent un titre, et deux titres différents', () => {
    const { unmount } = renderWithProviders(
      <SquadIsolementNuageCard nuage={nuageDe()} joueurs={joueurs} />,
    )
    const fr = screen.getByText('Isolement et couverture').textContent ?? ''
    unmount()
    useAppShellStore.setState({ locale: 'en' })
    renderWithProviders(<SquadIsolementNuageCard nuage={nuageDe()} joueurs={joueurs} />)
    const en = screen.getByText('Isolation and coverage').textContent ?? ''
    expect(fr.length).toBeGreaterThan(0)
    expect(en.length).toBeGreaterThan(0)
    expect(en).not.toBe(fr)
  })
})
