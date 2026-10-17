/**
 * TacticalAnalysisView — le PARCOURS de la lecture des coéquipiers, de bout en bout (lectures moquées) :
 *
 *   sans sélection de tuile   la vue de la zone (C) au-dessus de la liste, aucune requête de séquence ;
 *   clic sur une tuile        l'encadré de la mort en bas de la colonne, le bilan de zone se retire, la
 *                             séquence est demandée pour CETTE mort (match, victime, instant du match),
 *                             même quand la liste dit que son rejeu n'est pas construit ;
 *   séquence servie           elle se JOUE seule de −4 s à +3 s, bouton Pause / Rejouer, barre d'étapes ;
 *                             une étape met en pause et ouvre la fiche « À 7:20 » ;
 *   séquence absente          TOUJOURS un état qui dit pourquoi (une cause = un texte), jamais rien ;
 *   rejeu fermé par le garde  aucune zone, comme le lien de rejeu (fonction absente) ;
 *   re-clic ou Échap          retour à la vue de la zone ;
 *   un frag                   pas de séquence : « la séquence se lit sur les morts ».
 */
import type React from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, fireEvent, screen, within } from '@testing-library/react'

import type { TacticalCelluleReponse, TacticalRaster } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'
import { renderWithProviders } from '@/test/render-utils'

import { getTacticalText } from './i18n'
import { BILAN, FRAG, MORT_COMPLETE, SEQUENCE } from './mort.fixtures'
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
  contributions: [MORT_COMPLETE, FRAG],
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
const slot = () => screen.queryByTestId('tactical-sequence-slot')

// ─── Ce que la requête de séquence rend (forme de TanStack Query) ─────────────────────────────
const erreur = (code: string) => ({ code, message: 'x', retryable: false, status: 404 })
const servie = () => ({ data: SEQUENCE, isPending: false, isError: false, error: null })
const echec = (code: string) => ({ data: undefined, isPending: false, isError: true, error: erreur(code) })

// ─── Les images d'animation, pilotées à la main ───────────────────────────────────────────────
let file: Map<number, FrameRequestCallback>
let prochain: number
let horloge: number
function image(ms: number) {
  for (let fait = 0; fait < ms; fait += 16) {
    horloge += 16
    const callbacks = [...file.values()]
    file.clear()
    act(() => callbacks.forEach((cb) => cb(horloge)))
  }
}

beforeEach(() => {
  file = new Map()
  prochain = 1
  horloge = 1000
  vi.stubGlobal('requestAnimationFrame', (cb: FrameRequestCallback) => {
    file.set(prochain, cb)
    return prochain++
  })
  vi.stubGlobal('cancelAnimationFrame', (id: number) => file.delete(id))
  useAppShellStore.setState({
    locale: 'fr', userTimezone: 'UTC', currentTitleSlug: 'halo_infinite',
    availableTitles: [{ slug: 'halo_infinite', name: 'Halo Infinite', capabilities: ['replay'] }] as never,
  })
  useTacticalRaster.mockReturnValue({ data: RASTER, isPending: false, isError: false, isPlaceholderData: false })
  useTacticalCellule.mockReturnValue({ data: DETAIL, isPending: false, isError: false })
  useTacticalSequence.mockReturnValue(echec('replay_not_available'))
})

afterEach(() => {
  vi.clearAllMocks()
  vi.unstubAllGlobals()
})

describe('le parcours : zone, mort choisie, étapes, retour', () => {
  it('sans tuile choisie : la vue de la zone, aucune séquence demandée', () => {
    rendre()
    expect(screen.getByTestId('tactical-zone-bilan')).toBeInTheDocument()
    expect(screen.queryByTestId('tactical-mort-detail')).toBeNull()
    expect(derniereDemande()).toBeNull()
    expect(slot()).toBeNull()
  })

  it('le bilan ne dit jamais sur combien de morts l’équipe a été lue', () => {
    rendre()
    expect(screen.getByTestId('tactical-zone-bilan').textContent).not.toMatch(/lue sur|read on/i)
  })

  it('clic sur une tuile : l’encadré de la mort, le bilan se retire, la séquence est demandée pour cette mort', () => {
    rendre()
    fireEvent.click(tuiles()[0])
    expect(screen.getByTestId('tactical-mort-detail')).toHaveTextContent(/Mort · .* · 7:20/)
    expect(screen.queryByTestId('tactical-zone-bilan')).toBeNull()
    expect(derniereDemande()).toEqual({ match_id: 'm1', victim_xuid: '111', time_ms: 462_000 })
  })

  it('rejeu fermé par le garde (404 replay_not_available) : aucune zone, aucun message — comme le lien de rejeu', () => {
    rendre()
    fireEvent.click(tuiles()[0])
    expect(slot()).toBeNull()
    expect(screen.queryByTestId('tactical-sequence-etapes')).toBeNull()
    expect(screen.queryByRole('alert')).toBeNull()
    expect(screen.getByTestId('tactical-mort-fiche')).toBeInTheDocument()
  })

  it('titre sans rejeu 2D : aucune zone, même pour un frag', () => {
    useAppShellStore.setState({
      availableTitles: [{ slug: 'halo_infinite', name: 'Halo Infinite', capabilities: [] }] as never,
    })
    rendre()
    fireEvent.click(tuiles()[0])
    expect(slot()).toBeNull()
    fireEvent.click(tuiles()[1])
    expect(slot()).toBeNull()
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

  it('un frag choisi : « Frag », aucune séquence demandée, et la zone dit que la séquence se lit sur les morts', () => {
    rendre()
    fireEvent.click(tuiles()[1])
    expect(screen.getByTestId('tactical-mort-detail')).toHaveTextContent(/Frag · /)
    expect(derniereDemande()).toBeNull()
    expect(slot()).toHaveAttribute('data-etat', 'sur_les_morts')
    expect(slot()).toHaveTextContent('La séquence se lit sur les morts')
    expect(slot()).toHaveTextContent('Seules les morts ont une séquence')
  })
})

describe('une mort choisie montre TOUJOURS un état : la séquence, ou pourquoi elle manque', () => {
  const CAS: { code: string; etat: string; titre: string; description: RegExp }[] = [
    { code: 'sequence_film_not_analysed', etat: 'film_non_analyse', titre: 'Film non analysé', description: /n'a pas été analysé/ },
    { code: 'sequence_film_to_reanalyse', etat: 'film_a_reanalyser', titre: 'Film à réanalyser', description: /l'horloge dont la séquence a besoin/ },
    { code: 'sequence_map_unmeasured', etat: 'carte_sans_rayon', titre: 'Rayon du radar inconnu', description: /portée du radar de cette variante/ },
    { code: 'sequence_no_death_position', etat: 'sans_position', titre: 'Position de la mort absente', description: /position de la victime/ },
    { code: 'autre_chose_du_serveur', etat: 'inconnue', titre: 'Séquence indisponible', description: /n'a pas pu être lue/ },
  ]

  it.each(CAS)('$code : « $titre », dans un état vide du bloc de séquence', ({ code, etat, titre, description }) => {
    useTacticalSequence.mockReturnValue(echec(code))
    rendre()
    fireEvent.click(tuiles()[0])
    expect(slot()).toHaveAttribute('data-etat', etat)
    expect(slot()).toHaveTextContent(titre)
    expect(slot()?.textContent).toMatch(description)
    expect(screen.queryByTestId('tactical-sequence-etapes')).toBeNull()
    // La fiche de l’instantané reste là : seule la séquence manque.
    expect(screen.getByTestId('tactical-mort-fiche')).toBeInTheDocument()
  })

  it('une erreur réseau (pas de code) : l’état générique, jamais rien', () => {
    useTacticalSequence.mockReturnValue({ data: undefined, isPending: false, isError: true, error: new Error('Failed to fetch') })
    rendre()
    fireEvent.click(tuiles()[0])
    expect(slot()).toHaveAttribute('data-etat', 'inconnue')
  })

  it('analyse du film en cours : l’emplacement d’attente de l’app', () => {
    useTacticalSequence.mockReturnValue(echec('sequence_analysis_pending'))
    rendre()
    fireEvent.click(tuiles()[0])
    expect(slot()).toHaveAttribute('data-etat', 'analyse_en_cours')
    expect(within(slot() as HTMLElement).getByTestId('analysis-pending-slot')).toHaveTextContent("Film en cours d'analyse")
  })

  it('chargement de la séquence : une ligne de statut', () => {
    useTacticalSequence.mockReturnValue({ data: undefined, isPending: true, isError: false, error: null })
    rendre()
    fireEvent.click(tuiles()[0])
    expect(slot()).toHaveAttribute('data-etat', 'chargement')
    expect(within(slot() as HTMLElement).getByRole('status')).toHaveTextContent('Chargement de la séquence')
  })

  it('le rejeu du match non construit côté liste (replay_available faux) : la séquence est demandée quand même', () => {
    const sansRejeu = { ...MORT_COMPLETE, replay_available: false }
    useTacticalCellule.mockReturnValue({ data: { ...DETAIL, contributions: [sansRejeu, FRAG] }, isPending: false, isError: false })
    useTacticalSequence.mockReturnValue(echec('sequence_film_not_analysed'))
    rendre()
    fireEvent.click(tuiles()[0])
    expect(derniereDemande()).toEqual({ match_id: 'm1', victim_xuid: '111', time_ms: 462_000 })
    expect(slot()).toHaveTextContent('Film non analysé')
  })

  it('la zone garde la même hauteur minimale dans tous les états (le plan ne saute pas)', () => {
    useTacticalSequence.mockReturnValue(echec('sequence_film_not_analysed'))
    rendre()
    fireEvent.click(tuiles()[0])
    const classeVide = slot()?.className
    expect(classeVide).toContain('min-h-24')
    useTacticalSequence.mockReturnValue(servie())
    fireEvent.click(tuiles()[0])
    fireEvent.click(tuiles()[0])
    expect(slot()?.className).toBe(classeVide)
  })
})

describe('la séquence servie se joue seule', () => {
  const horlogeEcrite = () => screen.getByTestId('tactical-sequence-horloge')
  const bouton = () => screen.getByTestId('tactical-sequence-lecture')

  it('au clic : lecture automatique depuis −4 s, bouton Pause, huit étapes, horloge du rejeu', () => {
    useTacticalSequence.mockReturnValue(servie())
    rendre()
    fireEvent.click(tuiles()[0])
    expect(slot()).toHaveAttribute('data-etat', 'servie')
    expect(bouton()).toHaveAttribute('data-etat', 'lecture')
    expect(bouton()).toHaveTextContent('Pause')
    expect(screen.getAllByTestId('tactical-sequence-etape')).toHaveLength(8)
    // L’horloge écrite est celle du rejeu : −4 s avant 7:20.
    expect(horlogeEcrite()).toHaveTextContent('À 7:16')
    // La fiche suit l’étape en vigueur.
    expect(screen.getByTestId('tactical-etape-fiche')).toHaveTextContent('À 7:16')
  })

  it('elle avance en temps réel jusqu’à +3 s, s’arrête et propose « Rejouer »', () => {
    useTacticalSequence.mockReturnValue(servie())
    rendre()
    fireEvent.click(tuiles()[0])
    image(4100)
    // 4 s de lecture depuis −4 s : à l’instant de la mort, 7:20.
    expect(horlogeEcrite()).toHaveTextContent('À 7:20')
    image(3200)
    expect(horlogeEcrite()).toHaveTextContent('À 7:23')
    expect(bouton()).toHaveAttribute('data-etat', 'fin')
    expect(bouton()).toHaveTextContent('Rejouer')
    expect(screen.getAllByTestId('tactical-sequence-etape')[7]).toHaveAttribute('aria-pressed', 'true')
    fireEvent.click(bouton())
    expect(bouton()).toHaveAttribute('data-etat', 'lecture')
    expect(horlogeEcrite()).toHaveTextContent('À 7:16')
  })

  it('Pause arrête la tête, Reprendre la relance', () => {
    useTacticalSequence.mockReturnValue(servie())
    rendre()
    fireEvent.click(tuiles()[0])
    image(1000)
    fireEvent.click(bouton())
    expect(bouton()).toHaveAttribute('data-etat', 'pause')
    expect(bouton()).toHaveTextContent('Reprendre')
    const arret = horlogeEcrite().textContent
    image(2000)
    expect(horlogeEcrite().textContent).toBe(arret)
    fireEvent.click(bouton())
    expect(bouton()).toHaveTextContent('Pause')
  })

  it('une étape met en pause et ouvre la fiche « À 7:20 » ; « Instantané » revient à la fiche de la mort', () => {
    useTacticalSequence.mockReturnValue(servie())
    rendre()
    fireEvent.click(tuiles()[0])
    image(500)
    fireEvent.click(screen.getAllByTestId('tactical-sequence-etape')[4])
    expect(bouton()).toHaveAttribute('data-etat', 'pause')
    expect(horlogeEcrite()).toHaveTextContent('À 7:20')
    expect(screen.getByTestId('tactical-etape-fiche')).toHaveTextContent('À 7:20')
    expect(screen.queryByTestId('tactical-mort-fiche')).toBeNull()
    fireEvent.click(screen.getByRole('button', { name: 'Instantané' }))
    expect(screen.getByTestId('tactical-mort-fiche')).toBeInTheDocument()
    expect(screen.queryByTestId('tactical-etape-fiche')).toBeNull()
  })

  it('une autre mort repart du début de sa séquence', () => {
    useTacticalSequence.mockReturnValue(servie())
    rendre()
    fireEvent.click(tuiles()[0])
    fireEvent.click(screen.getAllByTestId('tactical-sequence-etape')[3])
    expect(bouton()).toHaveAttribute('data-etat', 'pause')
    fireEvent.click(tuiles()[1])
    fireEvent.click(tuiles()[0])
    expect(bouton()).toHaveAttribute('data-etat', 'lecture')
    expect(horlogeEcrite()).toHaveTextContent('À 7:16')
  })

  it('mouvement réduit : pas de lecture automatique, l’étape de la mort est affichée', () => {
    vi.stubGlobal('matchMedia', (query: string) => ({
      matches: query.includes('prefers-reduced-motion'),
      media: query, onchange: null, addEventListener: () => {}, removeEventListener: () => {},
      addListener: () => {}, removeListener: () => {}, dispatchEvent: () => false,
    }))
    useTacticalSequence.mockReturnValue(servie())
    rendre()
    fireEvent.click(tuiles()[0])
    expect(bouton()).toHaveAttribute('data-etat', 'pause')
    expect(horlogeEcrite()).toHaveTextContent('À 7:20')
    image(1000)
    expect(horlogeEcrite()).toHaveTextContent('À 7:20')
  })
})
