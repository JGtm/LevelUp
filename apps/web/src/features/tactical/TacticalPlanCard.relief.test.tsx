/**
 * TacticalPlanCard — le fond Relief, ses traits et la coupe par étage (item D.4 du plan du fond Relief).
 *
 * Ce que ces tests cadenassent : la pilule « Fond » à trois valeurs (Plan, Rendu, Relief) qui écrit la préférence du
 * rejeu ; les traits du relief retenus PAR FOND ; la pilule « Étage » (rangs, sans unité) ; la coupe appliquée au
 * fond (la toile du relief remplace l'`<img>`) et aux zones nommées SEULES (liste fermée) ; la mention de légende des
 * calques entiers ; la carte sans maillage (pas de pilule « Étage ») ; le repli sur le plan, journalisé.
 *
 * Les fichiers du relief et le calculateur sont remplacés par des doubles (les décisions se testent, pas le calcul) ;
 * `peindreLePlan` est moqué pour lire ce que le plan donne au peintre (zones filtrées, traits).
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, screen, within } from '@testing-library/react'

import type { TacticalRaster } from '@/lib/api/types'
import { RELIEF_VIEW_TEXT } from '@/components/map-relief/text'
import { renderWithProviders } from '@/test/render-utils'

import { getTacticalText } from './i18n'
import { peindreLePlan } from './planPaint'
import { canvasDuRelief, donneesDuRelief, frameDuRelief, sidecarDuRelief, ZONES_A_HAUTEUR } from './test/reliefPlan'
import { TacticalPlanCard, type ReglagesDuPlan } from './TacticalPlanCard'

const mocks = vi.hoisted(() => ({
  sidecar: { value: undefined as unknown },
  files: vi.fn(),
  render: vi.fn(),
}))

vi.mock('./planPaint', () => ({ peindreLePlan: vi.fn() }))
vi.mock('./queries', async (importOriginal) => {
  const actual = await importOriginal<typeof import('./queries')>()
  return {
    ...actual,
    useTacticalMapBackgroundFrame: () => null,
    useTacticalMapBackgroundPlan: () => null,
    useTacticalMapBackgroundImage: () => ({ url: 'blob:fond-plan', variante: 'plan' }),
    useTacticalMapPlanImage: () => ({ planImage: 'plan.webp', pending: false }),
    useTacticalMapBackgroundSidecarQuery: () => ({ data: mocks.sidecar.value, isPending: false }),
  }
})
vi.mock('./queriesRelief', async (importOriginal) => ({
  ...(await importOriginal<typeof import('./queriesRelief')>()),
  useTacticalBackgroundElement: () => 'image-du-fond',
}))
vi.mock('@/lib/replay/reliefQueries', () => ({ useReliefFiles: mocks.files }))
vi.mock('@/lib/replay/useReliefRender', () => ({ useReliefRender: mocks.render }))
// Les jetons ne se résolvent pas sans feuille de style : des encres fixes suffisent à ce que les traits se composent.
vi.mock('@/lib/replay/useReliefInks', () => ({ useReliefInks: () => ({ base: '#6b7078', outline: '#ffffff' }) }))
vi.mock('@/lib/replay/relief/reliefBackground', () => ({
  ReliefCanvases: class {
    reliefImage = () => canvasDuRelief()
    maskedImage = () => canvasDuRelief()
  },
}))

const t = getTacticalText('fr')
const T = RELIEF_VIEW_TEXT.fr

const LECTURE: TacticalRaster = {
  map_id: 'streets',
  question: 'morts',
  qui: 'moi',
  bornes: { min_x: 0, max_x: 100, min_y: 0, max_y: 50, valide: true },
  pas_m: 10,
  echelle: { p50: 1, p95: 5, borne: 5, n_cellules: 1, symetrique: false },
  cellules: [{ col: 2, lig: 1, valeur: 3, brut: 3, matchs: 4, matchs_victoire: 2, matchs_defaite: 2, centre_x: 25, centre_y: 15 }],
  grappes: [],
  matchs_filtres: 50,
  matchs_retenus: 45,
  matchs_victoire: 20,
  matchs_defaite: 25,
  matchs_en_attente: 0,
  points_ignores: 0,
  zones: ZONES_A_HAUTEUR,
}

const REGLAGES: ReglagesDuPlan = {
  question: 'morts',
  onQuestionChange: () => {},
  qui: 'moi',
  onQuiChange: () => {},
  escouadeDisponible: false,
  escouadeEnAttente: false,
  spawn: '',
  onSpawnChange: () => {},
  vies: 'depart',
  onViesChange: () => {},
  grappes: [],
  zonesVisibles: true,
  onZonesVisiblesChange: () => {},
}

/** La carte porte un relief (`relief`) et ses données (`donnees`), ou aucun. */
function carte({ relief = true, donnees = donneesDuRelief() as ReturnType<typeof donneesDuRelief> | null, error = null as unknown } = {}) {
  mocks.sidecar.value = sidecarDuRelief(relief)
  mocks.files.mockReturnValue({ data: relief ? donnees : null, blob: relief && donnees ? new Blob() : null, loading: false, error })
}

function rendre() {
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
      selected={null}
      onPointSelect={() => {}}
      contributions={[]}
      survol={null}
      onSurvol={() => {}}
      choisie={null}
      onChoisir={() => {}}
    />,
  )
}

const contexteFactice = { clearRect: vi.fn(), drawImage: vi.fn(), strokeRect: vi.fn() } as unknown as CanvasRenderingContext2D

/** Ce que le plan a donné au peintre à son dernier tracé. */
function dernierePeinture() {
  const peinture = vi.mocked(peindreLePlan).mock.calls.at(-1)?.[3]
  if (!peinture) throw new Error('le plan n’a rien peint')
  return peinture
}
const volumesPeints = () => dernierePeinture().zones.map((z) => z.volumeIndex)

beforeEach(() => {
  window.localStorage.clear()
  mocks.files.mockReset()
  mocks.render.mockReset().mockReturnValue({ status: 'ready', info: { width: 40, height: 10, hasMesh: true }, frame: frameDuRelief(), error: null })
  carte()
  vi.spyOn(HTMLCanvasElement.prototype, 'clientWidth', 'get').mockReturnValue(200)
  vi.spyOn(HTMLCanvasElement.prototype, 'clientHeight', 'get').mockReturnValue(100)
  vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue(contexteFactice as never)
})

afterEach(() => {
  vi.restoreAllMocks()
  vi.mocked(peindreLePlan).mockClear()
})

describe('la pilule « Fond »', () => {
  it('propose Plan, Rendu, Relief, avec le plan retenu par défaut', () => {
    rendre()
    const pilule = screen.getByTestId('tactical-plan-fond-pilule')
    expect(within(pilule).getAllByRole('button').map((b) => b.textContent)).toEqual(['Plan', 'Rendu', 'Relief'])
    expect(within(pilule).getByRole('button', { name: 'Plan' })).toHaveAttribute('aria-pressed', 'true')
  })

  it('choisir le Relief l’écrit dans la préférence du rejeu et pose la toile du relief à la place de l’image', () => {
    rendre()
    expect(screen.queryByTestId('tactical-plan-surface')).toBeNull()
    fireEvent.click(within(screen.getByTestId('tactical-plan-fond-pilule')).getByRole('button', { name: 'Relief' }))
    expect(window.localStorage.getItem('replay-background-style')).toBe('relief')
    expect(screen.getByTestId('tactical-plan-surface')).toBeInTheDocument()
    // L'image reste montée (pas de clignotement) mais invisible : le relief la remplace, il ne s'y superpose pas.
    expect(screen.getByTestId('tactical-plan-frame').querySelector('img')).toHaveStyle({ visibility: 'hidden' })
  })

  it('l’infobulle de « Relief » dit la part tirée du maillage (D-16)', () => {
    rendre()
    expect(screen.getByRole('button', { name: 'Relief' }).title).toContain('46 % de la carte')
  })

  it('carte sans relief : Plan et Rendu seulement, ni traits ni étage', () => {
    carte({ relief: false })
    rendre()
    const pilule = screen.getByTestId('tactical-plan-fond-pilule')
    expect(within(pilule).getAllByRole('button').map((b) => b.textContent)).toEqual(['Plan', 'Rendu'])
    expect(screen.queryByTestId('tactical-plan-traits')).toBeNull()
    expect(screen.queryByTestId('tactical-plan-etage')).toBeNull()
  })
})

describe('les traits du relief', () => {
  it('éteints sur le Plan : aucun trait au peintre', () => {
    rendre()
    expect(screen.getByTestId('tactical-plan-traits')).toHaveAttribute('aria-pressed', 'false')
    expect(dernierePeinture().reliefLines ?? null).toBeNull()
  })

  it('allumés d’office sur le Relief : le peintre les reçoit avec le calage du relief', () => {
    window.localStorage.setItem('replay-background-style', 'relief')
    rendre()
    expect(screen.getByTestId('tactical-plan-traits')).toHaveAttribute('aria-pressed', 'true')
    expect(dernierePeinture().reliefLines?.calibration.widthPx).toBe(40)
  })

  it('un geste est retenu PAR FOND : éteindre sur le Relief ne touche pas le Plan', () => {
    window.localStorage.setItem('replay-background-style', 'relief')
    rendre()
    fireEvent.click(screen.getByTestId('tactical-plan-traits'))
    expect(screen.getByTestId('tactical-plan-traits')).toHaveAttribute('aria-pressed', 'false')
    expect(window.localStorage.getItem('replay-relief-lines.relief')).toBe('false')
    fireEvent.click(within(screen.getByTestId('tactical-plan-fond-pilule')).getByRole('button', { name: 'Plan' }))
    expect(screen.getByTestId('tactical-plan-traits')).toHaveAttribute('aria-pressed', 'false')
    expect(window.localStorage.getItem('replay-relief-lines.plan')).toBeNull()
  })
})

describe('la pilule « Étage » et la coupe', () => {
  it('propose Tout puis un rang par étage, sans unité', () => {
    rendre()
    const pilule = screen.getByTestId('tactical-plan-etage')
    expect(within(pilule).getAllByRole('button').map((b) => b.textContent)).toEqual(['Tout', 'Bas', 'Étage 1', 'Étage 2'])
    expect(within(pilule).getByRole('button', { name: 'Tout' })).toHaveAttribute('aria-pressed', 'true')
  })

  it('« Tout » : toutes les zones nommées sont peintes, et la légende ne dit rien', () => {
    rendre()
    expect(volumesPeints()).toEqual([1, 2])
    expect(screen.queryByTestId('tactical-plan-sans-hauteur')).toBeNull()
  })

  it('un étage ne garde que les zones dont la tranche de hauteur croise sa bande', () => {
    rendre()
    fireEvent.click(within(screen.getByTestId('tactical-plan-etage')).getByRole('button', { name: 'Bas' }))
    expect(volumesPeints()).toEqual([1])
    fireEvent.click(within(screen.getByTestId('tactical-plan-etage')).getByRole('button', { name: 'Étage 2' }))
    expect(volumesPeints()).toEqual([2])
  })

  it('« Tout » rend toutes les zones', () => {
    rendre()
    const pilule = screen.getByTestId('tactical-plan-etage')
    fireEvent.click(within(pilule).getByRole('button', { name: 'Étage 2' }))
    fireEvent.click(within(pilule).getByRole('button', { name: 'Tout' }))
    expect(volumesPeints()).toEqual([1, 2])
  })

  it('sous coupe, la chaleur (sans hauteur) est nommée dans la légende : elle reste sur tous les étages', () => {
    rendre()
    fireEvent.click(within(screen.getByTestId('tactical-plan-etage')).getByRole('button', { name: 'Étage 1' }))
    const mentions = screen.getByTestId('tactical-plan-sans-hauteur')
    expect(mentions).toHaveTextContent('Chaleur : ce calque ne porte pas la hauteur, il reste affiché sur tous les étages.')
  })

  it('la coupe module le fond : la toile du relief remplace l’image, sur un fond Plan comme sur un fond Rendu', () => {
    rendre()
    fireEvent.click(within(screen.getByTestId('tactical-plan-etage')).getByRole('button', { name: 'Étage 1' }))
    expect(screen.getByTestId('tactical-plan-surface')).toBeInTheDocument()
    expect(screen.getByTestId('tactical-plan-frame').querySelector('img')).toHaveStyle({ visibility: 'hidden' })
  })

  it('une autre carte repart de « Tout » (la coupe n’est jamais persistée)', () => {
    const { unmount } = rendre()
    fireEvent.click(within(screen.getByTestId('tactical-plan-etage')).getByRole('button', { name: 'Étage 1' }))
    unmount()
    rendre()
    expect(within(screen.getByTestId('tactical-plan-etage')).getByRole('button', { name: 'Tout' })).toHaveAttribute('aria-pressed', 'true')
    expect(volumesPeints()).toEqual([1, 2])
  })
})

describe('carte sans maillage de navigation', () => {
  beforeEach(() => carte({ donnees: donneesDuRelief(true) }))

  it('pas de pilule « Étage » : jamais une commande qui ne commanderait rien', () => {
    rendre()
    expect(screen.queryByTestId('tactical-plan-etage')).toBeNull()
  })

  it('le fond Relief et ses traits restent proposés', () => {
    rendre()
    expect(within(screen.getByTestId('tactical-plan-fond-pilule')).getByRole('button', { name: 'Relief' })).toBeInTheDocument()
    expect(screen.getByTestId('tactical-plan-traits')).toBeInTheDocument()
  })

  it('l’infobulle de « Relief » ne cite pas de part de maillage', () => {
    rendre()
    expect(screen.getByRole('button', { name: 'Relief' }).title).not.toContain('%')
  })

  it('aucune zone n’est filtrée', () => {
    rendre()
    expect(volumesPeints()).toEqual([1, 2])
  })
})

describe('repli sur le plan, journalisé', () => {
  it('un relief illisible : plus de choix « Relief », le plan reste retenu, le journal le dit', () => {
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
    window.localStorage.setItem('replay-background-style', 'relief')
    carte({ donnees: null, error: new Error('version de schéma inconnue') })
    rendre()
    const pilule = screen.getByTestId('tactical-plan-fond-pilule')
    expect(within(pilule).queryByRole('button', { name: 'Relief' })).toBeNull()
    expect(within(pilule).getByRole('button', { name: 'Plan' })).toHaveAttribute('aria-pressed', 'true')
    expect(screen.queryByTestId('tactical-plan-surface')).toBeNull()
    expect(warn).toHaveBeenCalledWith(expect.stringContaining('[relief-view]'))
  })

  it('une panne du calculateur : même repli', () => {
    vi.spyOn(console, 'warn').mockImplementation(() => {})
    mocks.render.mockReturnValue({ status: 'failed', info: null, frame: null, error: new Error('worker') })
    window.localStorage.setItem('replay-background-style', 'relief')
    rendre()
    expect(within(screen.getByTestId('tactical-plan-fond-pilule')).getByRole('button', { name: 'Plan' })).toHaveAttribute('aria-pressed', 'true')
    expect(screen.queryByTestId('tactical-plan-surface')).toBeNull()
  })
})

describe('le texte des commandes', () => {
  it('sont dites avec le dictionnaire partagé des vues 2D', () => {
    rendre()
    expect(screen.getByRole('group', { name: T.backgroundLabel })).toBeInTheDocument()
    expect(screen.getByRole('group', { name: T.floorLabel })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: T.lines })).toBeInTheDocument()
  })
})
