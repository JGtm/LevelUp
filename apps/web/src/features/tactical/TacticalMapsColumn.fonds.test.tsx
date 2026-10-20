/**
 * La colonne « Cartes jouées » et ses fonds (plan backlog 2026-10-16, E7, D-10) : le chargement
 * de l'onglet ne coûtait PAS DEUX REQUÊTES PAR CARTE OUVRABLE (le calage, puis l'image).
 *
 * Ce que ces tests cadenassent, sur une grille de 46 cartes ouvrables (le relevé du 16/10 :
 * 96 requêtes = 4 fixes + 2 x 46) :
 *   - AUCUNE requête de calage (`GET .../background`) par vignette : le calage vient de la carte ;
 *   - AUCUNE image demandée tant qu'une vignette n'est pas entrée dans le champ de vision ;
 *   - une vignette visible demande SON image, une seule fois, à la bonne URL ;
 *   - une carte sans calage (sans fond figé) ne demande jamais d'image, même visible ;
 *   - le mini-plan se pose sur le calage porté par la carte (cadre du fond), sans requête.
 */
import { act, screen } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import type { TacticalMapCard } from '@/lib/api/types'
import { renderWithProviders } from '@/test/render-utils'

import { getTacticalText } from './i18n'
import { TacticalMapsColumn } from './TacticalMapsColumn'

vi.mock('@tanstack/react-router', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@tanstack/react-router')>()),
  Link: (await import('@/test/linkDouble')).LinkDouble,
}))

const get = vi.fn()
const getBlob = vi.fn()
vi.mock('@/lib/api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/api/client')>()
  return { ...actual, api: { ...actual.api, get: (p: string) => get(p), getBlob: (p: string) => getBlob(p) } }
})

// ─── IntersectionObserver commandable : rien n'est visible tant que le test ne le dit pas ──────

type Rappel = (entrees: Array<Pick<IntersectionObserverEntry, 'isIntersecting'>>) => void
const observes = new Map<Element, Rappel>()

class FauxObservateur {
  private readonly cb: Rappel
  constructor(cb: Rappel) {
    this.cb = cb
  }
  observe = (el: Element) => void observes.set(el, this.cb)
  disconnect = () => undefined
}

/** Fait entrer la vignette `mapId` dans le champ de vision. */
function entrer(mapId: string) {
  const tuile = screen.getByTestId(`tactical-map-${mapId}`).parentElement as Element
  const rappel = observes.get(tuile)
  if (!rappel) throw new Error(`vignette ${mapId} non observée`)
  act(() => rappel([{ isIntersecting: true }]))
}

const CALAGE = { metersPerPixel: 1, originX: 0, originY: 60, widthPx: 60, heightPx: 60, convention: 'xy' }
const N = 46

function carte(i: number, avecFond: boolean): TacticalMapCard {
  return {
    map_id: `carte-${i}`,
    map_name: `Carte ${i}`,
    map_name_fr: `Carte ${i}`,
    matchs: 100 - i,
    victoires: 50,
    defaites: 40,
    sous_plancher: false,
    ...(avecFond ? { calage_fond: CALAGE } : {}),
  }
}

const GRILLE = Array.from({ length: N }, (_, i) => carte(i, i !== 2))
const t = getTacticalText('fr')

function rendre(cartes: TacticalMapCard[] = GRILLE) {
  return renderWithProviders(
    <TacticalMapsColumn
      cartes={cartes}
      plancher={10}
      carteActive=""
      playerSlug="JGtm"
      locale="fr"
      t={t}
      onSelect={vi.fn()}
      enRelecture={false}
    />,
  )
}

beforeEach(() => {
  observes.clear()
  get.mockReset()
  getBlob.mockReset()
  get.mockRejectedValue(new Error('aucune requête de calage attendue'))
  getBlob.mockResolvedValue(new Blob(['png']))
  vi.stubGlobal('IntersectionObserver', FauxObservateur)
  vi.spyOn(URL, 'createObjectURL').mockImplementation(() => `blob:test/${Math.random()}`)
})

describe('TacticalMapsColumn — fonds', () => {
  it('46 vignettes montées : aucune requête de calage, aucune image hors champ de vision', async () => {
    rendre()
    expect(await screen.findByTestId('tactical-map-carte-0')).toBeInTheDocument()
    expect(screen.getAllByTestId(/^tactical-map-carte-\d+$/)).toHaveLength(N)
    expect(get).not.toHaveBeenCalled()
    expect(getBlob).not.toHaveBeenCalled()
  })

  it('une vignette qui entre dans le champ de vision demande SON image, une fois', async () => {
    rendre()
    entrer('carte-0')
    entrer('carte-1')
    expect(await screen.findByTestId('tactical-map-fond-carte-0')).toBeInTheDocument()
    expect(await screen.findByTestId('tactical-map-fond-carte-1')).toBeInTheDocument()
    expect(getBlob.mock.calls.map((c) => c[0])).toEqual([
      '/players/JGtm/tactical/carte-0/background.png',
      '/players/JGtm/tactical/carte-1/background.png',
    ])
    expect(get).not.toHaveBeenCalled()
    // Une seconde entrée ne redemande rien (même clé de cache).
    entrer('carte-0')
    expect(getBlob).toHaveBeenCalledTimes(2)
  })

  it('une carte sans calage (sans fond figé) ne demande jamais d’image, même visible', () => {
    rendre()
    entrer('carte-2')
    expect(getBlob).not.toHaveBeenCalled()
    expect(screen.queryByTestId('tactical-map-fond-carte-2')).toBeNull()
  })

  it('le mini-plan se pose sur le calage porté par la carte, sans requête de calage', async () => {
    const avecPlan: TacticalMapCard = {
      ...carte(0, true),
      // Bornes INVALIDES : sans le cadre du fond, le repère est introuvable et rien n'est peint.
      bornes: { min_x: 0, max_x: 0, min_y: 0, max_y: 0, valide: false },
      pas_m: 6,
      echelle: { p50: 1, p95: 3, borne: 3, n_cellules: 1, symetrique: false },
      cellules: [
        { col: 2, lig: 3, valeur: 2, brut: 2, centre_x: 15, centre_y: 21, matchs: 3, matchs_defaite: 2, matchs_victoire: 1 },
      ],
    }
    const sansCalage: TacticalMapCard = { ...avecPlan, map_id: 'carte-9', calage_fond: undefined }
    rendre([avecPlan, sansCalage])
    expect(await screen.findByTestId('tactical-map-calque-carte-0')).toBeInTheDocument()
    expect(screen.queryByTestId('tactical-map-calque-carte-9')).toBeNull()
    expect(get).not.toHaveBeenCalled()
  })
})
