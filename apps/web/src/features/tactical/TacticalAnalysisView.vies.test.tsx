/**
 * TacticalAnalysisView — la bascule « Départ / Toutes les vies » : le paramètre `vies` rejoint la
 * lecture (donc sa clé de cache), « Départ » ne change rien au corps posté, et changer de valeur
 * remet le filtre de grappe à « Toutes » (l'identifiant d'une grappe n'existe que sous sa valeur).
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, screen } from '@testing-library/react'

import type { TacticalRaster } from '@/lib/api/types'
import { renderWithProviders } from '@/test/render-utils'

import type { CarteEffective } from './cockpit.logic'
import { getTacticalText } from './i18n'
import { TacticalAnalysisView } from './TacticalAnalysisView'

vi.mock('@/lib/api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/api/client')>()
  return { ...actual, api: { ...actual.api, getBlob: () => Promise.reject(new Error('aucun fond')) } }
})

const useTacticalRaster = vi.fn()
const useTacticalCellule = vi.fn<(...args: unknown[]) => { data?: unknown; isPending: boolean }>(() => ({
  data: undefined,
  isPending: true,
}))
vi.mock('./queries', async (importOriginal) => {
  const actual = await importOriginal<typeof import('./queries')>()
  return {
    ...actual,
    useTacticalRaster: (...args: unknown[]) => useTacticalRaster(...args),
    useTacticalCellule: (...args: unknown[]) => useTacticalCellule(...args),
  }
})
vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return { ...actual, Link: ({ children }: { children?: React.ReactNode }) => <a href="/rejeu">{children}</a> }
})

const t = getTacticalText('fr')
const LUE: CarteEffective = { mapId: 'streets', origine: 'url' }

const RASTER: TacticalRaster = {
  map_id: 'streets',
  question: 'morts',
  qui: 'moi',
  bornes: { min_x: 0, max_x: 100, min_y: 0, max_y: 50, valide: true },
  pas_m: 10,
  echelle: { p50: 1, p95: 5, borne: 5, n_cellules: 1, symetrique: false },
  cellules: [{ col: 5, lig: 2, valeur: 5, brut: 5, matchs: 6, matchs_victoire: 3, matchs_defaite: 3, centre_x: 55, centre_y: 25 }],
  grappes: [{ id: 'g1', nom: 'Base Rouge', matchs: 10, x: 10, y: 10 }],
  matchs_filtres: 50,
  matchs_retenus: 45,
  matchs_victoire: 20,
  matchs_defaite: 25,
  matchs_en_attente: 0,
  points_ignores: 0,
}

function monter(raster: TacticalRaster = RASTER) {
  useTacticalRaster.mockReturnValue({ data: raster, isPending: false, isError: false })
  return renderWithProviders(
    <TacticalAnalysisView
      playerSlug="JGtm"
      carte={LUE}
      mapName="Ruelles"
      locale="fr"
      t={t}
      matchIds={['m1', 'm2']}
      coequipiers={[]}
    />,
  )
}

type Params = { spawn?: string; vies?: string }
const dernierParams = () => useTacticalRaster.mock.calls.at(-1)?.[2] as Params
const dernierParamsCellule = () => useTacticalCellule.mock.calls.at(-1)?.[3] as Params | undefined

afterEach(() => {
  vi.clearAllMocks()
})

describe('TacticalAnalysisView — la bascule « Départ / Toutes les vies »', () => {
  it('« Départ » par défaut : la bascule est là, et la lecture ne porte aucun paramètre `vies`', () => {
    monter()
    expect(screen.getByRole('button', { name: t.livesStart })).toHaveAttribute('aria-pressed', 'true')
    expect(screen.getByRole('button', { name: t.livesAll })).toHaveAttribute('aria-pressed', 'false')
    expect(dernierParams().vies).toBeUndefined()
  })

  it('« Toutes les vies » : le paramètre `vies` rejoint la lecture du plan et le détail de la zone', () => {
    monter()
    fireEvent.click(screen.getByRole('button', { name: t.livesAll }))
    expect(screen.getByRole('button', { name: t.livesAll })).toHaveAttribute('aria-pressed', 'true')
    expect(dernierParams().vies).toBe('toutes')
    expect(dernierParamsCellule()?.vies).toBe('toutes')
  })

  it('changer de valeur remet le filtre de grappe à « Toutes » : l’identifiant n’existe que sous sa valeur', () => {
    monter()
    fireEvent.change(screen.getByRole('combobox', { name: t.pillRespawn }), { target: { value: 'g1' } })
    expect(dernierParams().spawn).toBe('g1')
    fireEvent.click(screen.getByRole('button', { name: t.livesAll }))
    expect(dernierParams().spawn).toBeUndefined()
    expect(screen.getByRole('combobox', { name: t.pillRespawn })).toHaveValue('')
    // Un clic sur la valeur déjà active ne touche à rien.
    fireEvent.change(screen.getByRole('combobox', { name: t.pillRespawn }), { target: { value: 'g1' } })
    fireEvent.click(screen.getByRole('button', { name: t.livesAll }))
    expect(dernierParams().spawn).toBe('g1')
  })

  it('revenir à « Départ » ne poste de nouveau aucun `vies`', () => {
    monter()
    fireEvent.click(screen.getByRole('button', { name: t.livesAll }))
    fireEvent.click(screen.getByRole('button', { name: t.livesStart }))
    expect(dernierParams().vies).toBeUndefined()
  })

  it('sans grappe, la bascule reste proposée sur une lecture d’artefact et se tait sur une lecture de la base', () => {
    const { unmount } = monter({ ...RASTER, grappes: [] })
    expect(screen.queryByTestId('tactical-plan-vies')).toBeNull()
    fireEvent.change(screen.getByRole('combobox', { name: t.pillReading }), { target: { value: 'temps' } })
    expect(screen.getByTestId('tactical-plan-vies')).toBeInTheDocument()
    unmount()
  })

  it('les mots de la bascule, en français et en anglais', () => {
    const en = getTacticalText('en')
    expect([t.pillLives, t.livesStart, t.livesAll]).toEqual(['Vies', 'Départ', 'Toutes les vies'])
    expect([en.pillLives, en.livesStart, en.livesAll]).toEqual(['Lives', 'Start', 'All lives'])
    expect(t.livesHint).toContain('Toutes les vies')
    expect(en.livesHint).toContain('All lives')
  })
})
