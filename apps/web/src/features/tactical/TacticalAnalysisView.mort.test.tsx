/**
 * TacticalAnalysisView — le PARCOURS de la lecture des coéquipiers, de bout en bout (lectures moquées) :
 *
 *   sans sélection de tuile   la vue de la zone (C) au-dessus de la liste, aucune requête de séquence ;
 *   clic sur une tuile        l'encadré de la mort en bas de la colonne, le bilan de zone se retire, la
 *                             séquence est demandée pour CETTE mort (match, victime, instant du match) ;
 *   séquence servie           la barre d'étapes ; une étape ouvre la fiche « À 7:20 » ;
 *   séquence en 404           aucune barre, aucun message : la vue D n'existe pas là où le rejeu n'est pas servi ;
 *   re-clic ou Échap          retour à la vue de la zone ;
 *   un frag                   pas de séquence (elle ne se demande que pour une mort).
 */
import type React from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, screen } from '@testing-library/react'

import type { TacticalCelluleReponse, TacticalRaster } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'
import { renderWithProviders } from '@/test/render-utils'

import { getTacticalText } from './i18n'
import { BILAN, FRAG_ABATTU_EN_RETOUR, MORT_COMPLETE, SEQUENCE } from './mort.fixtures'
import { TacticalAnalysisView } from './TacticalAnalysisView'

vi.mock('@/lib/api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/api/client')>()
  return { ...actual, api: { ...actual.api, getBlob: () => Promise.reject(new Error('pas de fond')) } }
})
const useTacticalRaster = vi.fn()
const useTacticalCellule = vi.fn()
const useTacticalSequence = vi.fn()
vi.mock('./queries', async (importOriginal) => {
  const actual = await importOriginal<typeof import('./queries')>()
  return {
    ...actual,
    useTacticalRaster: (...a: unknown[]) => useTacticalRaster(...a),
    useTacticalCellule: (...a: unknown[]) => useTacticalCellule(...a),
    useTacticalSequence: (...a: unknown[]) => useTacticalSequence(...a),
  }
})
vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    Link: ({ params, search, children, ...rest }: { params: { matchId: string }; search?: Record<string, string>; children?: React.ReactNode } & Record<string, unknown>) => (
      <a href={`/m/${params.matchId}/replay?${new URLSearchParams(search).toString()}`} {...(rest as Record<string, unknown>)}>
        {children}
      </a>
    ),
  }
})
vi.mock('@/lib/i18n/fieldMappings', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/i18n/fieldMappings')>()
  return { ...actual, useOutcomeMapping: () => undefined }
})
vi.mock('@/lib/title-routing', () => ({ useTitleSlug: () => 'halo_infinite' }))

const RASTER: TacticalRaster = {
  map_id: 'streets', question: 'morts', qui: 'moi',
  bornes: { min_x: 0, max_x: 100, min_y: 0, max_y: 50, valide: true },
  pas_m: 10, echelle: { p50: 1, p95: 5, borne: 5, n_cellules: 1, symetrique: false },
  cellules: [{ col: 2, lig: 1, valeur: 14, brut: 14, matchs: 9, matchs_victoire: 5, matchs_defaite: 4, centre_x: 25, centre_y: 15 }],
  grappes: [], matchs_filtres: 50, matchs_retenus: 45, matchs_victoire: 20, matchs_defaite: 25, matchs_en_attente: 0, points_ignores: 0,
}
const DETAIL: TacticalCelluleReponse = {
  contributions: [MORT_COMPLETE, FRAG_ABATTU_EN_RETOUR],
  zone: { nom_fr: 'Place centrale', nom_en: 'Central Square' },
  bilan: BILAN,
}

function rendre() {
  return renderWithProviders(
    <TacticalAnalysisView
      playerSlug="JGtm"
      carte={{ mapId: 'streets', origine: 'url' }}
      mapName="Ruelles"
      locale="fr"
      t={getTacticalText('fr')}
      matchIds={['m1']}
      coequipiers={[]}
    />,
  )
}

const tuiles = () => screen.getAllByTestId('tactical-rejeu-tile')
const derniereDemande = () => useTacticalSequence.mock.calls.at(-1)?.[2]

beforeEach(() => {
  useAppShellStore.setState({
    locale: 'fr', userTimezone: 'UTC', currentTitleSlug: 'halo_infinite',
    availableTitles: [{ slug: 'halo_infinite', name: 'Halo Infinite', capabilities: ['replay'] }] as never,
  })
  useTacticalRaster.mockReturnValue({ data: RASTER, isPending: false, isError: false, isPlaceholderData: false })
  useTacticalCellule.mockReturnValue({ data: DETAIL, isPending: false, isError: false })
  useTacticalSequence.mockReturnValue({ data: undefined, isError: false })
})

afterEach(() => {
  vi.clearAllMocks()
})

describe('le parcours : zone, mort choisie, étapes, retour', () => {
  it('sans tuile choisie : la vue de la zone, aucune séquence demandée', () => {
    rendre()
    expect(screen.getByTestId('tactical-zone-bilan')).toBeInTheDocument()
    expect(screen.queryByTestId('tactical-mort-detail')).toBeNull()
    expect(derniereDemande()).toBeNull()
  })

  it('le bilan ne dit jamais sur combien de morts l’équipe a été lue', () => {
    rendre()
    expect(screen.getByTestId('tactical-zone-bilan').textContent).not.toMatch(/lue sur|read on/i)
  })

  it('clic sur une tuile : l’encadré de la mort, le bilan se retire, la séquence est demandée pour cette mort', () => {
    rendre()
    fireEvent.click(tuiles()[0])
    expect(screen.getByTestId('tactical-mort-detail')).toHaveTextContent(/Instant de la mort · .* · 7:20/)
    expect(screen.queryByTestId('tactical-zone-bilan')).toBeNull()
    expect(derniereDemande()).toEqual({ match_id: 'm1', victim_xuid: '111', time_ms: 462_000 })
  })

  it('séquence en 404 (rejeu non servi) : ni barre d’étapes ni message', () => {
    rendre()
    fireEvent.click(tuiles()[0])
    expect(screen.queryByTestId('tactical-sequence-etapes')).toBeNull()
    expect(screen.queryByRole('alert')).toBeNull()
    expect(screen.getByTestId('tactical-mort-fiche')).toBeInTheDocument()
  })

  it('séquence servie : la barre ; une étape ouvre la fiche « À 7:20 » ; « Instantané » revient à la fiche de la mort', () => {
    useTacticalSequence.mockReturnValue({ data: SEQUENCE, isError: false })
    rendre()
    fireEvent.click(tuiles()[0])
    const etapes = screen.getAllByTestId('tactical-sequence-etape')
    expect(etapes).toHaveLength(8)
    fireEvent.click(etapes[4])
    expect(screen.getByTestId('tactical-etape-fiche')).toHaveTextContent('À 7:20')
    expect(screen.queryByTestId('tactical-mort-fiche')).toBeNull()
    fireEvent.click(screen.getByRole('button', { name: 'Instantané' }))
    expect(screen.getByTestId('tactical-mort-fiche')).toBeInTheDocument()
    expect(screen.queryByTestId('tactical-etape-fiche')).toBeNull()
  })

  it('re-clic sur la tuile : retour à la vue de la zone ; Échap aussi', () => {
    rendre()
    fireEvent.click(tuiles()[0])
    fireEvent.click(tuiles()[0])
    expect(screen.getByTestId('tactical-zone-bilan')).toBeInTheDocument()
    expect(screen.queryByTestId('tactical-mort-detail')).toBeNull()
    fireEvent.click(tuiles()[0])
    expect(screen.getByTestId('tactical-mort-detail')).toBeInTheDocument()
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(screen.getByTestId('tactical-zone-bilan')).toBeInTheDocument()
  })

  it('une étape choisie ne survit pas à un autre choix : la nouvelle mort s’ouvre sur son instantané', () => {
    useTacticalSequence.mockReturnValue({ data: SEQUENCE, isError: false })
    rendre()
    fireEvent.click(tuiles()[0])
    fireEvent.click(screen.getAllByTestId('tactical-sequence-etape')[3])
    fireEvent.click(tuiles()[1])
    fireEvent.click(tuiles()[0])
    expect(screen.getByTestId('tactical-mort-fiche')).toBeInTheDocument()
  })

  it('un frag choisi : « Instant du frag », aucune séquence demandée', () => {
    rendre()
    fireEvent.click(tuiles()[1])
    expect(screen.getByTestId('tactical-mort-detail')).toHaveTextContent('Instant du frag')
    expect(derniereDemande()).toBeNull()
  })
})
