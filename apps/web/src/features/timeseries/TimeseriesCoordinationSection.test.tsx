/**
 * Coordination des Séries temporelles (lot Q) — la projection PURE et les trois états de
 * la carte « Appui reçu » (la carte « Riposte » a quitté la page : plan
 * PLAN_TIMESERIES_USAGES_EMPRISE_2026-10-05, V5).
 *
 * Ce que ces tests cadenassent :
 *   - la projection d'une soirée (taux en %, bâton creux, dénominateur), et le refus
 *     d'inventer un taux sur une soirée sans dénominateur ;
 *   - la carte CONSERVÉE, nommée, quand le bloc est indisponible (D8),
 *     et la section entièrement retirée quand le titre ne sert aucun bloc ;
 *   - D22-verbosité : les chiffres d'appel sont rendus, aucune phrase de lecteur.
 *
 * `echarts-for-react` est mocké (canvas jsdom instable), comme partout sur cette page.
 */
import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'

import type { CoordinationBlock, CoordinationSessionPoint, Couverture } from '@/lib/api/types'

import { TimeseriesCoordinationSection } from './TimeseriesCoordinationSection'
import {
  coordinationDessinable,
  habituelOuTaux,
  moyenneGlissante,
  pariteOuRien,
  serieDeSoirees,
} from './timeseriesCoordination.logic'

vi.mock('echarts-for-react', () => ({
  default: () => <div data-testid="echarts-mock" />,
}))

function couverture(brut: number, n: number, faible = false): Couverture {
  return { brut, n, taux: n > 0 ? brut / n : 0, par_match: 0, echantillon_faible: faible }
}

function soiree(label: string, mesFrags: number, faible = false): CoordinationSessionPoint {
  return {
    session_label: label,
    matches_measured: 3,
    matches_total: 3,
    appui: {
      on_me_prepare: couverture(Math.round(mesFrags / 2), mesFrags, faible),
      ma_part_des_appuis: couverture(6, 30),
    },
  }
}

function bloc(over: Partial<CoordinationBlock> = {}): CoordinationBlock {
  return {
    available: true,
    matches_measured: 24,
    matches_total: 30,
    appui: {
      on_me_prepare: couverture(41, 100),
      ma_part_des_appuis: couverture(27, 100),
      parity_pct: 25,
    },
    sessions: [soiree('12/09/2025 21:00–23:10 (5)', 10), soiree('14/09/2025 20:00–21:00 (2)', 4, true)],
    ...over,
  }
}

describe('timeseriesCoordination.logic', () => {
  it('projette une soirée en taux POURCENTS, son bâton creux et son dénominateur', () => {
    const s = serieDeSoirees(bloc().sessions ?? [], (p) => p.appui.on_me_prepare)
    expect(s.valuesPct).toEqual([50, 50])
    expect(s.hollow).toEqual([false, true])
    expect(s.volumes).toEqual([10, 4])
  })

  it('n’invente AUCUN taux sur une soirée sans dénominateur', () => {
    const s = serieDeSoirees([soiree('20/09/2025 20:00 (1)', 0)], (p) => p.appui.on_me_prepare)
    expect(s.valuesPct).toEqual([null])
  })

  it('ne dessine pas de parité quand le serveur ne la mesure pas', () => {
    expect(pariteOuRien(undefined)).toBeNull()
    expect(pariteOuRien(0)).toBeNull()
    expect(pariteOuRien(25)).toBe(25)
  })

  it('prend l’habituel de la période de RÉFÉRENCE quand le serveur le mesure, le taux sinon', () => {
    const c = couverture(54, 100)
    expect(habituelOuTaux(48.4, c)).toBe(48.4)
    expect(habituelOuTaux(undefined, c)).toBe(54)
    expect(habituelOuTaux(0, c)).toBe(54)
  })

  it('calcule la tendance sur les bâtons PLEINS, et rien tant que la fenêtre est incomplète', () => {
    const serie = {
      valuesPct: [10, 20, 30, 40, 50],
      hollow: [false, false, false, false, false],
      volumes: [9, 9, 9, 9, 9],
    }
    expect(moyenneGlissante(serie)).toEqual([null, null, 20, 30, 40])
    // Une soirée creuse (échantillon faible) ne nourrit pas la moyenne : la fenêtre
    // devient incomplète et la tendance se tait plutôt que de mentir.
    expect(moyenneGlissante({ ...serie, hollow: [false, false, true, false, false] })).toEqual([
      null,
      null,
      null,
      null,
      null,
    ])
    expect(moyenneGlissante({ ...serie, valuesPct: [10, null, 30, 40, 50] })[3]).toBeNull()
  })

  it('ne se dit dessinable qu’avec un bloc disponible ET au moins une soirée', () => {
    expect(coordinationDessinable(bloc())).toBe(true)
    expect(coordinationDessinable(bloc({ sessions: [] }))).toBe(false)
    expect(coordinationDessinable(bloc({ available: false }))).toBe(false)
    expect(coordinationDessinable(undefined)).toBe(false)
  })
})

describe('TimeseriesCoordinationSection', () => {
  it('rend « Appui reçu » seule et ses chiffres d’appel — sans phrase de lecteur ni couverture', async () => {
    render(<TimeseriesCoordinationSection block={bloc()} locale="fr" />)
    expect(screen.getByRole('region', { name: 'Appui reçu' })).toBeInTheDocument()
    expect(screen.queryByRole('region', { name: 'Riposte' })).not.toBeInTheDocument()
    // Le canvas est chargé en `lazy` par ChartCard : il arrive après la suspense.
    expect(await screen.findAllByTestId('echarts-mock')).toHaveLength(1)
    expect(screen.getAllByText('41 %').length).toBeGreaterThan(0)
    expect(screen.queryByText(/mesurés sur/)).not.toBeInTheDocument()
  })

  it('CONSERVE la carte quand le bloc est indisponible : un état vide nommé', () => {
    render(
      <TimeseriesCoordinationSection
        block={bloc({ available: false, unavailable_reason: 'Aucun film décodé' })}
        locale="fr"
      />,
    )
    expect(screen.getAllByText('Aucun film décodé')).toHaveLength(1)
    expect(screen.queryAllByTestId('echarts-mock')).toHaveLength(0)
    expect(screen.getByRole('region', { name: 'Appui reçu' })).toBeInTheDocument()
  })

  it('ne rend RIEN quand la page ne sert aucun bloc de coordination', () => {
    const { container } = render(<TimeseriesCoordinationSection block={undefined} locale="fr" />)
    expect(container).toBeEmptyDOMElement()
  })
})
