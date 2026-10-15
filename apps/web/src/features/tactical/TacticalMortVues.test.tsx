/**
 * Les VUES de la lecture des coéquipiers à la mort, rendues :
 *
 *   C  la zone         bilan dans le bloc du bas de la colonne (barre des quatre statuts, rose, médianes),
 *                      nuage des coéquipiers sur le plan ;
 *   A  l'instantané    bloc du bas « Mort · date · instant » (même hauteur que le bilan), cercle du rayon,
 *                      coéquipiers pleins ou creux sur le plan ;
 *   D  la séquence     barre d'étapes cliquables, scène de l'étape, fiche « À 7:20 ».
 *
 * Aucun échange (« tueur abattu », « abattu en retour ») nulle part : décision du user du 2026-10-15.
 *
 * L'horloge affichée est celle du REJEU (coup d'envoi retranché) : l'instant du match de la fixture
 * (462 000 ms, « 7:42 » si on oubliait le décompte) et l'instant affiché (440 000 ms) diffèrent exprès.
 */
import type React from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, screen, within } from '@testing-library/react'

import type { CelluleTactique, TacticalCelluleReponse, TacticalRaster } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'
import { renderWithProviders } from '@/test/render-utils'

import { peindreLaScene } from './coequipiersPaint'
import { getTacticalText } from './i18n'
import { BILAN, FRAG, MORT_COMPLETE, SEQUENCE } from './mort.fixtures'
import { peindreLePlan } from './planPaint'
import { TacticalPlanCard, type ReglagesDuPlan } from './TacticalPlanCard'
import { TacticalZoneCard } from './TacticalZoneCard'
import { peindreLaCouche } from './tueursPaint'
import type { VueDeMort } from './vueDeMort'

vi.mock('./planPaint', () => ({ peindreLePlan: vi.fn() }))
vi.mock('./tueursPaint', () => ({ peindreLaCouche: vi.fn() }))
vi.mock('./coequipiersPaint', () => ({ peindreLaScene: vi.fn() }))
vi.mock('./queries', async (importOriginal) => {
  const actual = await importOriginal<typeof import('./queries')>()
  return { ...actual, useTacticalMapBackgroundFrame: () => null, useTacticalMapBackgroundUrl: () => null }
})
vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    Link: ({
      params,
      search,
      children,
      ...rest
    }: {
      params: { titleSlug: string; playerSlug: string; matchId: string }
      search?: Record<string, string>
      children?: React.ReactNode
    } & Record<string, unknown>) => {
      const qs = search ? `?${new URLSearchParams(search).toString()}` : ''
      return (
        <a href={`/${params.titleSlug}/players/${params.playerSlug}/matches/${params.matchId}/replay${qs}`} {...(rest as Record<string, unknown>)}>
          {children}
        </a>
      )
    },
  }
})
vi.mock('@/lib/i18n/fieldMappings', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/i18n/fieldMappings')>()
  return { ...actual, useOutcomeMapping: () => undefined }
})
vi.mock('@/lib/title-routing', () => ({ useTitleSlug: () => 'halo_infinite' }))

const t = getTacticalText('fr')

const CELLULE: CelluleTactique = {
  col: 2, lig: 1, valeur: 14, brut: 14, matchs: 9, matchs_victoire: 5, matchs_defaite: 4, frags: 3, morts: 14, centre_x: 25, centre_y: 15,
}
const LECTURE: TacticalRaster = {
  map_id: 'streets', question: 'morts', qui: 'moi',
  bornes: { min_x: -20, max_x: 80, min_y: -20, max_y: 30, valide: true },
  pas_m: 10, echelle: { p50: 1, p95: 5, borne: 5, n_cellules: 1, symetrique: false },
  cellules: [CELLULE], grappes: [],
  matchs_filtres: 50, matchs_retenus: 45, matchs_victoire: 20, matchs_defaite: 25, matchs_en_attente: 0, points_ignores: 0,
}
const REGLAGES: ReglagesDuPlan = {
  question: 'morts', onQuestionChange: () => {}, qui: 'moi', onQuiChange: () => {},
  escouadeDisponible: false, escouadeEnAttente: false, spawn: '', onSpawnChange: () => {}, grappes: [],
}

const DETAIL: TacticalCelluleReponse = {
  contributions: [MORT_COMPLETE, FRAG],
  zone: { nom_fr: 'Place centrale', nom_en: 'Central Square' },
  bilan: BILAN,
}

function vue(extra: Partial<VueDeMort> = {}): VueDeMort {
  return { index: 0, contribution: MORT_COMPLETE, sequence: null, etape: null, onEtape: () => {}, ...extra }
}

function rendreZone(props: { question?: 'morts' | 'isole' | 'kills' | 'solde' | 'temps'; vue?: VueDeMort | null } = {}) {
  return renderWithProviders(
    <TacticalZoneCard
      t={t}
      locale="fr"
      playerSlug="JGtm"
      question={props.question ?? 'morts'}
      signee={false}
      pasM={10}
      cellule={CELLULE}
      detail={{ data: DETAIL, isPending: false, isError: false }}
      bilan={DETAIL.bilan}
      vue={props.vue ?? null}
    />,
  )
}

function rendrePlan(v: VueDeMort | null, contributions = [MORT_COMPLETE, FRAG]) {
  return renderWithProviders(
    <TacticalPlanCard
      t={t}
      locale="fr"
      playerSlug="JGtm"
      mapId="streets"
      titre="Ruelles"
      question="morts"
      lecture={LECTURE}
      etat="pret"
      inconnus={null}
      etiquette={null}
      reglages={REGLAGES}
      selected={{ col: 2, row: 1 }}
      onPointSelect={() => {}}
      contributions={contributions}
      survol={null}
      onSurvol={() => {}}
      choisie={v ? v.index : null}
      onChoisir={() => {}}
      vue={v}
    />,
  )
}

function derniereScene() {
  const appels = vi.mocked(peindreLaScene).mock.calls
  return appels[appels.length - 1]?.[1]
}

beforeEach(() => {
  useAppShellStore.setState({
    locale: 'fr',
    userTimezone: 'UTC',
    currentTitleSlug: 'halo_infinite',
    availableTitles: [{ slug: 'halo_infinite', name: 'Halo Infinite', capabilities: ['replay'] }] as never,
  })
  vi.spyOn(HTMLCanvasElement.prototype, 'clientWidth', 'get').mockReturnValue(400)
  vi.spyOn(HTMLCanvasElement.prototype, 'clientHeight', 'get').mockReturnValue(200)
  vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue({
    clearRect: vi.fn(), strokeRect: vi.fn(), beginPath: vi.fn(), arc: vi.fn(), stroke: vi.fn(),
  } as unknown as CanvasRenderingContext2D)
})

afterEach(() => {
  vi.restoreAllMocks()
  vi.mocked(peindreLePlan).mockClear()
  vi.mocked(peindreLaCouche).mockClear()
  vi.mocked(peindreLaScene).mockClear()
})

describe('vue C — la zone, dans le bloc du bas', () => {
  it('lecture « morts » : barre des quatre statuts, rose, phrase, médianes, « N sur M »', () => {
    const { container } = rendreZone()
    const bilan = screen.getByTestId('tactical-zone-bilan')
    const statuts = within(bilan).getByTestId('tactical-bilan-statuts')
    expect(statuts).toHaveTextContent('4 dans le rayon')
    expect(statuts).toHaveTextContent('6 au-delà')
    expect(statuts).toHaveTextContent('1 hors de vue')
    expect(statuts).toHaveTextContent('3 aucun')
    expect(within(bilan).getByTestId('tactical-bilan-barre').children).toHaveLength(4)
    expect(within(bilan).getByTestId('tactical-rose').querySelectorAll('line')).toHaveLength(6)
    expect(bilan).toHaveTextContent('5 sur 14 par la droite')
    expect(bilan).toHaveTextContent('médiane 16 m')
    expect(bilan).toHaveTextContent(/Coéquipier le plus proche.*médiane 21 m/)
    expect(bilan.textContent).not.toMatch(/abattu/i)
    // Aucun « non mesuré » à l'écran, et le bilan est dans le bloc du bas, APRÈS la liste.
    expect(container.textContent).not.toMatch(/non mesur/i)
    const liste = screen.getByTestId('tactical-rejeu-list')
    expect(liste.compareDocumentPosition(bilan) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
  })

  it('lecture « isole » : la même vue ; lectures de frags : aucun bilan', () => {
    const { unmount } = rendreZone({ question: 'isole' })
    expect(screen.getByTestId('tactical-zone-bilan')).toBeInTheDocument()
    unmount()
    rendreZone({ question: 'kills' })
    expect(screen.queryByTestId('tactical-zone-bilan')).toBeNull()
  })

  it('le bloc du bas garde la même hauteur avec ou sans tuile choisie (aucun saut de mise en page)', () => {
    const { unmount } = rendreZone()
    const sans = screen.getByTestId('tactical-zone-bloc').className
    unmount()
    rendreZone({ vue: vue() })
    const avec = screen.getByTestId('tactical-zone-bloc').className
    expect(avec).toBe(sans)
    expect(avec).toContain('h-[196px]')
  })

  it('lectures sans mort ni frag (temps) : aucun bilan', () => {
    rendreZone({ question: 'temps' })
    expect(screen.queryByTestId('tactical-zone-bilan')).toBeNull()
  })

  it('sur le plan : le nuage des coéquipiers et sa légende', () => {
    rendrePlan(null)
    const scene = derniereScene()
    expect(scene?.marques.every((m) => m.role === 'nuage')).toBe(true)
    expect(scene?.marques).toHaveLength(2)
    expect(screen.getByTestId('tactical-plan-formes')).toHaveTextContent('Coéquipiers en jeu à chaque mort')
    expect(screen.queryByTestId('tactical-sequence-etapes')).toBeNull()
  })
})

describe('vues A et B — la mort choisie', () => {
  it('l’encadré s’ouvre en bas de la colonne, après la liste, et le bilan de zone se retire', () => {
    rendreZone({ vue: vue() })
    expect(screen.queryByTestId('tactical-zone-bilan')).toBeNull()
    const liste = screen.getByTestId('tactical-rejeu-list')
    const detail = screen.getByTestId('tactical-mort-detail')
    expect(liste.compareDocumentPosition(detail) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
  })

  it('titre « Mort · date · instant » sur l’horloge du REJEU (7:20, jamais 7:42)', () => {
    rendreZone({ vue: vue() })
    const detail = screen.getByTestId('tactical-mort-detail')
    expect(detail).toHaveTextContent(/Mort · 12\/10\/2026 · \d{2}:\d{2} · 7:20/)
    expect(detail.textContent).not.toContain('7:42')
    // Le bouton de rejeu ouvre au même instant que la tuile : t = instant du match, horloge « match ».
    const lien = within(detail).getByRole('link')
    expect(lien.getAttribute('href')).toContain('t=462000')
    expect(lien.getAttribute('href')).toContain('clock=match')
  })

  it('fiche : le tueur et ses distances, les coéquipiers ● ○, en attente, résumé de l’équipe', () => {
    rendreZone({ vue: vue() })
    const fiche = screen.getByTestId('tactical-mort-fiche')
    expect(fiche).toHaveTextContent('Hollow Vex · BR75')
    expect(fiche).toHaveTextContent('15 m')
    const lignes = fiche.textContent ?? ''
    expect(lignes).toContain('Kestrel 77')
    expect(lignes).toContain('12 m')
    expect(lignes).toContain('27 m')
    expect(fiche.querySelector('[data-forme="plein"]')).not.toBeNull()
    expect(fiche.querySelector('[data-forme="creux"]')).not.toBeNull()
    expect(fiche).toHaveTextContent('Brisk Otter · en attente de réapparition')
    expect(screen.getByTestId('tactical-mort-detail')).toHaveTextContent('1 coéquipier dans le rayon de 18 m, 1 au-delà, 1 hors jeu.')
  })

  it('un frag choisi : « Frag · date · instant » et la victime', () => {
    rendreZone({ vue: vue({ index: 1, contribution: FRAG }) })
    expect(screen.getByTestId('tactical-mort-detail')).toHaveTextContent(/Frag · .* · 7:28/)
    expect(screen.getByTestId('tactical-mort-fiche')).toHaveTextContent('Victime · Cible')
  })

  it('sur le plan : le cercle du rayon, le mort, les coéquipiers pleins et creux', () => {
    rendrePlan(vue())
    const scene = derniereScene()
    expect(scene?.cercle).toEqual({ centre: { x: 10, y: 10 }, rayonM: 18 })
    expect(scene?.marques.map((m) => [m.role, m.plein])).toEqual([
      ['mort', true],
      ['coequipier', true],
      ['coequipier', false],
    ])
    const legende = screen.getByTestId('tactical-plan-formes')
    expect(legende).toHaveTextContent('Mort')
    expect(legende).toHaveTextContent('Coéquipier dans le rayon')
    expect(legende).toHaveTextContent('Coéquipier hors du rayon')
    expect(legende).toHaveTextContent('Rayon du radar (18 m)')
    expect(legende.querySelector('[data-forme="mort"]')).not.toBeNull()
    expect(legende.querySelector('[data-forme="creux"]')).not.toBeNull()
  })


  it('sans séquence servie (rejeu absent ou 404) : pas de barre d’étapes', () => {
    rendrePlan(vue())
    expect(screen.queryByTestId('tactical-sequence-etapes')).toBeNull()
  })
})

describe('vue D — la séquence', () => {
  it('la barre : « Instantané » puis huit étapes de −4,0 s à +3,0 s, cliquables', () => {
    const onEtape = vi.fn()
    rendrePlan(vue({ sequence: SEQUENCE, onEtape }))
    const barre = screen.getByTestId('tactical-sequence-etapes')
    expect(within(barre).getByRole('button', { name: 'Instantané' })).toHaveAttribute('aria-pressed', 'true')
    const etapes = within(barre).getAllByTestId('tactical-sequence-etape')
    expect(etapes.map((b) => b.textContent)).toEqual(['−4,0 s', '−3,0 s', '−2,0 s', '−1,0 s', '0,0 s', '+1,0 s', '+2,0 s', '+3,0 s'])
    fireEvent.click(etapes[2])
    expect(onEtape).toHaveBeenCalledWith(2)
    fireEvent.click(within(barre).getByRole('button', { name: 'Instantané' }))
    expect(onEtape).toHaveBeenCalledWith(null)
  })

  it('une étape choisie : la scène de l’étape, la couche des tueurs se retire, légende de la séquence', () => {
    rendrePlan(vue({ sequence: SEQUENCE, etape: 4 }))
    const scene = derniereScene()
    expect(scene?.cercle?.centre).toEqual({ x: 10, y: 10 })
    expect(scene?.traines.length).toBeGreaterThan(0)
    expect(scene?.marques.map((m) => m.role).sort()).toEqual(['coequipier', 'mort', 'tueur'])
    expect(screen.queryByTestId('tactical-plan-tueurs')).toBeNull()
    expect(screen.getByTestId('tactical-plan-formes')).toHaveTextContent('traîne : 4 s précédentes')
    expect(screen.getByTestId('tactical-plan-formes')).toHaveTextContent('Tueur')
    const choisie = within(screen.getByTestId('tactical-sequence-etapes')).getAllByTestId('tactical-sequence-etape')[4]
    expect(choisie).toHaveAttribute('aria-pressed', 'true')
  })

  it('la fiche d’étape : horloge du rejeu, compte dans le rayon, distance de chaque acteur', () => {
    rendreZone({ vue: vue({ sequence: SEQUENCE, etape: 4 }) })
    const fiche = screen.getByTestId('tactical-etape-fiche')
    expect(fiche).toHaveTextContent('À 7:20')
    expect(fiche).toHaveTextContent('1 sur 1 dans le rayon de 18 m')
    expect(fiche).toHaveTextContent('Hollow Vex')
    expect(fiche).toHaveTextContent('15 m')
    expect(fiche).toHaveTextContent('Kestrel 77')
    expect(fiche).toHaveTextContent('12 m')
    expect(screen.queryByTestId('tactical-mort-fiche')).toBeNull()
  })

  it('une étape avant le coup d’envoi : l’horloge est celle du rejeu, plancher à 0:00 côté serveur', () => {
    const seq = { ...SEQUENCE, etapes: SEQUENCE.etapes!.map((e, i) => (i === 0 ? { ...e, instant_affiche_ms: 0 } : e)) }
    rendreZone({ vue: vue({ sequence: seq, etape: 0 }) })
    expect(screen.getByTestId('tactical-etape-fiche')).toHaveTextContent('À 0:00')
  })
})

describe('UNE seule légende sous le plan', () => {
  it('une forme = une entrée, les deux couches ensemble, dans l’ordre Mort · Tueurs · coéquipiers · rayon', () => {
    rendrePlan(vue())
    const legendes = screen.getAllByTestId('tactical-plan-formes')
    expect(legendes).toHaveLength(1)
    const formes = [...legendes[0].querySelectorAll('[data-forme]')].map((n) => n.getAttribute('data-forme'))
    expect(formes).toEqual(['mort', 'losange', 'rond', 'plein', 'creux', 'cercle'])
    expect(new Set(formes).size).toBe(formes.length)
    expect(screen.queryByTestId('tactical-tueurs-legend')).toBeNull()
  })

  it('les lectures de frags : victimes seules', () => {
    rendrePlan(null, [FRAG])
    const formes = [...screen.getByTestId('tactical-plan-formes').querySelectorAll('[data-forme]')].map((n) => n.getAttribute('data-forme'))
    expect(formes).toEqual(['rond'])
  })
})
