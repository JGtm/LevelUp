/**
 * Les QUATRE VUES de la lecture des coéquipiers à la mort, rendues :
 *
 *   C  la zone         bilan au-dessus de la liste (barre des quatre statuts, rose, « Tueur abattu en 3 s : N sur M »,
 *                      médiane), nuage des coéquipiers sur le plan ;
 *   A  l'instantané    encadré « Instant de la mort · date · instant » en bas de la colonne de droite, cercle du
 *                      rayon, coéquipiers pleins ou creux sur le plan ;
 *   B  l'échange       bulle « Tueur abattu 1,8 s après la mort », trait du frag du coéquipier ;
 *   D  la séquence     barre d'étapes cliquables, scène de l'étape, fiche « À 7:20 ».
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
import { BILAN, FRAG_ABATTU_EN_RETOUR, MORT_COMPLETE, SEQUENCE } from './mort.fixtures'
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
  contributions: [MORT_COMPLETE, FRAG_ABATTU_EN_RETOUR],
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

function rendrePlan(v: VueDeMort | null, contributions = [MORT_COMPLETE, FRAG_ABATTU_EN_RETOUR]) {
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

describe('vue C — la zone, au-dessus de la liste', () => {
  it('lecture « morts » : barre des quatre statuts, rose, phrase, médianes, « N sur M »', () => {
    const { container } = rendreZone()
    const bilan = screen.getByTestId('tactical-zone-bilan')
    const statuts = within(bilan).getByTestId('tactical-bilan-statuts')
    expect(statuts).toHaveTextContent('dans le rayon · 4')
    expect(statuts).toHaveTextContent('au-delà du rayon · 6')
    expect(statuts).toHaveTextContent('hors de vue · 1')
    expect(statuts).toHaveTextContent('aucun en jeu · 3')
    expect(within(bilan).getByTestId('tactical-bilan-barre').children).toHaveLength(4)
    expect(within(bilan).getByTestId('tactical-rose').querySelectorAll('line')).toHaveLength(6)
    expect(bilan).toHaveTextContent('5 sur 14 par la droite')
    expect(bilan).toHaveTextContent('distance médiane 16 m')
    const chiffres = within(bilan).getAllByTestId('tactical-bilan-chiffre').map((n) => n.textContent)
    expect(chiffres[0]).toMatch(/Tueur abattu en 3 s.*5\s*sur 14/)
    expect(chiffres[1]).toMatch(/Coéquipier le plus proche.*21 m\s*médiane/)
    // Aucun « non mesuré » à l'écran, et le bilan est AVANT la liste.
    expect(container.textContent).not.toMatch(/non mesur/i)
    const liste = screen.getByTestId('tactical-rejeu-list')
    expect(bilan.compareDocumentPosition(liste) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
  })

  it('lecture « isole » : la même vue ; lectures de frags : le miroir « abattu en retour » seul', () => {
    const { unmount } = rendreZone({ question: 'isole' })
    expect(screen.getByTestId('tactical-zone-bilan')).toBeInTheDocument()
    unmount()
    rendreZone({ question: 'kills' })
    expect(screen.queryByTestId('tactical-zone-bilan')).toBeNull()
    expect(screen.getByTestId('tactical-bilan-frags')).toHaveTextContent(/Abattu en retour en 3 s.*2\s*sur 9/)
  })

  it('lectures sans mort ni frag (temps) : aucun bilan', () => {
    rendreZone({ question: 'temps' })
    expect(screen.queryByTestId('tactical-zone-bilan')).toBeNull()
    expect(screen.queryByTestId('tactical-bilan-frags')).toBeNull()
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

  it('titre « Instant de la mort · date · instant » sur l’horloge du REJEU (7:20, jamais 7:42)', () => {
    rendreZone({ vue: vue() })
    const detail = screen.getByTestId('tactical-mort-detail')
    expect(detail).toHaveTextContent(/Instant de la mort · 12\/10\/2026 · \d{2}:\d{2} · 7:20/)
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
    expect(fiche).toHaveTextContent('1 coéquipier dans le rayon de 18 m, 1 au-delà, 1 hors jeu.')
  })

  it('un frag choisi : « Instant du frag », la victime et l’échange', () => {
    rendreZone({ vue: vue({ index: 1, contribution: FRAG_ABATTU_EN_RETOUR }) })
    expect(screen.getByTestId('tactical-mort-detail')).toHaveTextContent(/Instant du frag · .* · 7:28/)
    expect(screen.getByTestId('tactical-mort-fiche')).toHaveTextContent('Victime · Cible')
    expect(screen.getByTestId('tactical-mort-fiche')).toHaveTextContent('Abattu en retour 1,5 s après le frag')
  })

  it('sur le plan : le cercle du rayon, le mort, les coéquipiers pleins et creux, le trait du frag', () => {
    rendrePlan(vue())
    const scene = derniereScene()
    expect(scene?.cercle).toEqual({ centre: { x: 10, y: 10 }, rayonM: 18 })
    expect(scene?.marques.map((m) => [m.role, m.plein])).toEqual([
      ['mort', true],
      ['coequipier', true],
      ['coequipier', false],
    ])
    expect(scene?.traits).toHaveLength(1)
    const legende = screen.getByTestId('tactical-plan-formes')
    expect(legende).toHaveTextContent('Mort')
    expect(legende).toHaveTextContent('Coéquipier dans le rayon')
    expect(legende).toHaveTextContent('Coéquipier hors du rayon')
    expect(legende).toHaveTextContent('Rayon du radar (18 m)')
    expect(legende.querySelector('[data-forme="mort"]')).not.toBeNull()
    expect(legende.querySelector('[data-forme="creux"]')).not.toBeNull()
  })

  it('B — la bulle de l’échange sur le plan', () => {
    rendrePlan(vue())
    const bulle = screen.getByTestId('tactical-echange')
    expect(bulle).toHaveTextContent('Tueur abattu 1,8 s après la mort')
    expect(bulle).toHaveTextContent('par Kestrel 77, à 27 m du tueur')
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
    expect(screen.queryByTestId('tactical-echange')).toBeNull()
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

describe('la tuile porte le badge de riposte, court, sur la ligne 3', () => {
  it('« abattu 1,8 s » sur une mort dont le tueur a été abattu, « abattu en retour » sur un frag', () => {
    rendreZone()
    const badges = screen.getAllByTestId('tactical-rejeu-riposte')
    expect(badges.map((b) => b.textContent)).toEqual(['abattu 1,8 s', 'abattu en retour'])
    for (const b of badges) expect(b.closest('[data-testid="tactical-rejeu-l3"]')).not.toBeNull()
  })
})

describe('UNE seule légende sous le plan', () => {
  it('une forme = une entrée, les deux couches ensemble, dans l’ordre Mort · Tueurs · coéquipiers · frag · rayon', () => {
    rendrePlan(vue())
    const legendes = screen.getAllByTestId('tactical-plan-formes')
    expect(legendes).toHaveLength(1)
    const formes = [...legendes[0].querySelectorAll('[data-forme]')].map((n) => n.getAttribute('data-forme'))
    expect(formes).toEqual(['mort', 'losange', 'rond', 'plein', 'creux', 'trait', 'cercle'])
    expect(new Set(formes).size).toBe(formes.length)
    expect(screen.queryByTestId('tactical-tueurs-legend')).toBeNull()
  })

  it('les lectures de frags : victimes seules', () => {
    rendrePlan(null, [FRAG_ABATTU_EN_RETOUR])
    const formes = [...screen.getByTestId('tactical-plan-formes').querySelectorAll('[data-forme]')].map((n) => n.getAttribute('data-forme'))
    expect(formes).toEqual(['rond'])
  })
})
