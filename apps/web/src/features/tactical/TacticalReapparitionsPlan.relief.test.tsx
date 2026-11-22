/**
 * TacticalReapparitionsPlan — le fond Relief et la coupe par étage sur le plan de la lecture « Réapparitions
 * adverses » (item D.4 du plan du fond Relief) : les MÊMES commandes que le plan de placement, et les zones nommées
 * (teinte et étiquettes) sont le second calque qui porte une hauteur — la coupe les filtre.
 *
 * La couche des zones est remplacée par un double qui expose la liste qu'elle reçoit ; les fichiers du relief et le
 * calculateur le sont aussi (la décision se teste, pas le calcul).
 */
import { fireEvent, screen, within } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { TacticalRaster } from '@/lib/api/types'
import { normalizeCalloutZones } from '@/lib/replay/calloutsPaint'
import { renderWithProviders } from '@/test/render-utils'

import { getTacticalText } from './i18n'
import { canvasDuRelief, donneesDuRelief, frameDuRelief, sidecarDuRelief, ZONES_A_HAUTEUR } from './test/reliefPlan'
import { TacticalReapparitionsPlan } from './TacticalReapparitionsPlan'

const FOND = { originX: 0, originY: 1, widthM: 4, heightM: 1 }
const mocks = vi.hoisted(() => ({
  sidecar: { value: undefined as unknown },
  files: vi.fn(),
  render: vi.fn(),
}))

vi.mock('./useHauteurDuPlan', () => ({ useHauteurDuPlan: () => 500 }))
vi.mock('./queries', async (importOriginal) => {
  const actual = await importOriginal<typeof import('./queries')>()
  return {
    ...actual,
    useTacticalMapBackgroundFrame: () => FOND,
    useTacticalMapBackgroundPlan: () => null,
    useTacticalMapBackgroundImage: () => ({ url: 'blob:fond-plan', variante: 'plan' as const }),
    useTacticalMapPlanImage: () => ({ planImage: 'plan.webp', pending: false }),
    useTacticalMapBackgroundSidecarQuery: () => ({ data: mocks.sidecar.value, isPending: false }),
  }
})
vi.mock('./queriesRelief', async (importOriginal) => ({
  ...(await importOriginal<typeof import('./queriesRelief')>()),
  useTacticalBackgroundElement: () => 'image-du-fond',
}))
vi.mock('./TacticalReapparitionsZones', () => ({
  TacticalReapparitionsZones: ({ zones }: { zones: readonly { volumeIndex: number }[] }) => (
    <div data-testid="zones-recues">{zones.map((z) => z.volumeIndex).join(',')}</div>
  ),
}))
vi.mock('@/lib/replay/reliefQueries', () => ({ useReliefFiles: mocks.files }))
vi.mock('@/lib/replay/useReliefRender', () => ({ useReliefRender: mocks.render }))
vi.mock('@/lib/replay/useReliefInks', () => ({ useReliefInks: () => ({ base: '#6b7078', outline: '#ffffff' }) }))
vi.mock('@/lib/replay/relief/reliefBackground', () => ({
  ReliefCanvases: class {
    reliefImage = () => canvasDuRelief()
    maskedImage = () => canvasDuRelief()
  },
}))

const t = getTacticalText('fr')

const LECTURE: TacticalRaster = {
  map_id: 'streets',
  question: 'morts',
  qui: 'moi',
  bornes: { min_x: 0, max_x: 4, min_y: 0, max_y: 1, valide: true },
  pas_m: 1,
  echelle: { p50: 1, p95: 5, borne: 5, n_cellules: 1, symetrique: false },
  cellules: [],
  grappes: [],
  matchs_filtres: 50,
  matchs_retenus: 45,
  matchs_victoire: 20,
  matchs_defaite: 25,
  matchs_en_attente: 0,
  points_ignores: 0,
}

const ZONES = normalizeCalloutZones(ZONES_A_HAUTEUR)

function carte(relief: boolean) {
  mocks.sidecar.value = sidecarDuRelief(relief)
  mocks.files.mockReturnValue({ data: relief ? donneesDuRelief() : null, blob: relief ? new Blob() : null, loading: false, error: null })
}

const PROPOSEES = [
  { volumeIndex: 1, nom: 'Base' },
  { volumeIndex: 2, nom: 'Pont' },
]

function rendre(choisie: number | null = null) {
  return renderWithProviders(
    <TacticalReapparitionsPlan
      t={t}
      locale="fr"
      playerSlug="JGtm"
      mapId="streets"
      titre="Ruelles"
      lecture={LECTURE}
      etat="pret"
      inconnus={null}
      modele={null}
      zones={ZONES}
      pilule={null}
      zonesProposees={PROPOSEES}
      choisie={choisie}
      onChoisir={() => {}}
      plan={{ href: null, decide: true }}
    />,
  )
}

const zonesRecues = () => screen.getByTestId('zones-recues').textContent

beforeEach(() => {
  window.localStorage.clear()
  mocks.files.mockReset()
  mocks.render.mockReset().mockReturnValue({ status: 'ready', info: { width: 40, height: 10, hasMesh: true }, frame: frameDuRelief(), error: null })
  carte(true)
  vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue({ clearRect: vi.fn(), drawImage: vi.fn() } as never)
})

afterEach(() => vi.restoreAllMocks())

describe('TacticalReapparitionsPlan — le fond Relief et la coupe par étage', () => {
  it('pose les trois commandes du fond : Plan, Rendu, Relief, traits et étage', () => {
    rendre()
    const fond = screen.getByTestId('tactical-plan-fond-pilule')
    expect(within(fond).getAllByRole('button').map((b) => b.textContent)).toEqual(['Plan', 'Rendu', 'Relief'])
    expect(screen.getByTestId('tactical-plan-traits')).toBeInTheDocument()
    expect(screen.getByTestId('tactical-plan-etage')).toBeInTheDocument()
  })

  it('« Tout » : toutes les zones nommées sont à la couche', () => {
    rendre()
    expect(zonesRecues()).toBe('1,2')
  })

  it('un étage ne laisse à la couche que les zones dont la tranche de hauteur croise sa bande', () => {
    rendre()
    const etage = screen.getByTestId('tactical-plan-etage')
    fireEvent.click(within(etage).getByRole('button', { name: 'Bas' }))
    expect(zonesRecues()).toBe('1')
    fireEvent.click(within(etage).getByRole('button', { name: 'Étage 2' }))
    expect(zonesRecues()).toBe('2')
  })

  it('la coupe module le fond : la toile du relief remplace l’image', () => {
    rendre()
    fireEvent.click(within(screen.getByTestId('tactical-plan-etage')).getByRole('button', { name: 'Étage 1' }))
    expect(screen.getByTestId('tactical-plan-surface')).toBeInTheDocument()
  })

  describe('la zone tenue choisie que la coupe retire du plan', () => {
    const select = () => screen.getByTestId('tactical-reapp-zone-select') as HTMLSelectElement
    const optionsDuSelect = () => within(select()).getAllByRole('option').map((o) => o.textContent)

    it('sans coupe, aucune zone n’est dite hors de l’étage', () => {
      rendre(1)
      expect(optionsDuSelect()).toEqual(['Aucune', 'Base', 'Pont'])
      expect(select().value).toBe('1')
      expect(select().closest('label')).not.toHaveAttribute('title')
    })

    it('la coupe retire la zone choisie : la pilule la dit hors de l’étage, grisée et expliquée, sans défaire le choix', () => {
      rendre(1)
      fireEvent.click(within(screen.getByTestId('tactical-plan-etage')).getByRole('button', { name: 'Étage 2' }))
      expect(zonesRecues()).toBe('2')
      expect(optionsDuSelect()).toEqual(['Aucune', t.reappHeldOffFloor('Base'), 'Pont'])
      expect(select().value).toBe('1')
      expect(select().className).toContain('text-muted-foreground')
      expect(select().closest('label')).toHaveAttribute('title', t.reappHeldOffFloorHint)
    })

    it('une zone choisie qui reste sur l’étage n’est pas annoncée hors de l’étage', () => {
      rendre(2)
      fireEvent.click(within(screen.getByTestId('tactical-plan-etage')).getByRole('button', { name: 'Étage 2' }))
      expect(select().className).not.toContain('text-muted-foreground')
      expect(select().closest('label')).not.toHaveAttribute('title')
    })

    it('lever la coupe rend la zone : la mention disparaît', () => {
      rendre(1)
      const etage = within(screen.getByTestId('tactical-plan-etage'))
      fireEvent.click(etage.getByRole('button', { name: 'Étage 2' }))
      fireEvent.click(etage.getByRole('button', { name: 'Tout' }))
      expect(optionsDuSelect()).toEqual(['Aucune', 'Base', 'Pont'])
      expect(select().closest('label')).not.toHaveAttribute('title')
    })
  })

  it('carte sans relief : ni traits ni étage, les zones restent entières', () => {
    carte(false)
    rendre()
    expect(screen.queryByTestId('tactical-plan-traits')).toBeNull()
    expect(screen.queryByTestId('tactical-plan-etage')).toBeNull()
    expect(zonesRecues()).toBe('1,2')
  })
})
