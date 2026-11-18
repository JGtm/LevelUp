/**
 * Tests — LA FICHE DU JOUEUR EN FENÊTRE au Cinéma et au Plein cadre, dans la vraie page.
 *
 * Monte la vraie page, la vraie toile (contexte 2D enregistreur) et un match à deux camps ; la largeur du corps du
 * cadre et les rectangles de la fiche ouverte et de la toile sont posés par le test (jsdom ne met rien en page).
 * Ce que ces tests prouvent :
 *  - un clic sur une fiche de la rangée, ou sur une ligne de la colonne de repli, ouvre une fenêtre en portail,
 *    couche d'Échap, qui porte le corps commun de la fiche ; la colonne ne se montre jamais en rangée ;
 *  - la fenêtre est NON BLOQUANTE : un geste sur le plan ou sur la barre ne la ferme pas, la lecture continue ;
 *    elle se ferme par la croix, par Échap (avant le Plein cadre) ou en recliquant la même fiche ; une autre fiche
 *    change de joueur ;
 *  - elle s'ancre bord gauche sur la fiche (par la droite près du bord droit de la page) et sa hauteur est bornée
 *    par le bas du plan ;
 *  - le Standard garde sa fiche en colonne, et la toile n'est jamais remontée.
 */
import { act, fireEvent, screen, within } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { REPLAY_TEXT } from './i18n/i18n'
import { COTES_FENETRE_FICHE } from './model/cotesFenetreFiche'
import { largeurRequiseRangee } from './model/rowLayout'
import { goToFrame, mountReplayPage, stubReplayPage, unstubReplayPage } from './test/replayPageHarness'
import { scoreboardRow } from './test/scoreboardRow'
import { testReplayDoc } from './test/testDoc'

vi.mock('@tanstack/react-router', async (importOriginal) =>
  (await import('./test/replayPageRouter')).routerMock(await importOriginal<typeof import('@tanstack/react-router')>()),
)

const T = REPLAY_TEXT.fr
const PP = T.playerPanel
const L4 = largeurRequiseRangee(4)

/** La largeur du corps du cadre, les rectangles de la fiche ouverte et de la toile que le test pose. */
let largeurCorps = 0
let rectFiche = { left: 140, right: 400, top: 10, bottom: 70 }
let basDuPlan = 500
const observateurs: { cible: Element; rappel: () => void }[] = []

class ObservateurDeTaille {
  private readonly rappel: () => void
  constructor(rappel: () => void) {
    this.rappel = rappel
  }
  observe(cible: Element) {
    observateurs.push({ cible, rappel: this.rappel })
  }
  unobserve() {}
  disconnect() {
    for (let i = observateurs.length - 1; i >= 0; i--) if (observateurs[i].rappel === this.rappel) observateurs.splice(i, 1)
  }
}

const rect = (r: Partial<DOMRect>): DOMRect => ({ x: 0, y: 0, width: 0, height: 0, top: 0, left: 0, right: 0, bottom: 0, toJSON: () => ({}), ...r })

beforeEach(() => {
  window.localStorage.clear()
  largeurCorps = L4
  rectFiche = { left: 140, right: 400, top: 10, bottom: 70 }
  basDuPlan = 500
  observateurs.length = 0
  stubReplayPage()
  vi.stubGlobal('ResizeObserver', ObservateurDeTaille)
  vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockImplementation(function (this: HTMLElement) {
    if (this.hasAttribute('data-replay-body')) return rect({ width: largeurCorps, right: largeurCorps })
    if (this.getAttribute('aria-pressed') === 'true' && this.hasAttribute('data-replay-player-card')) return rect(rectFiche)
    if (this.tagName === 'CANVAS') return rect({ bottom: basDuPlan })
    // Le reste du cadre a une taille : sans elle la toile n'aurait aucune largeur et le clavier du lecteur serait inerte.
    return rect({ width: 960, height: 540, right: 960, bottom: 540 })
  })
})

afterEach(() => {
  unstubReplayPage()
})

/** Un match à deux camps de `parCamp` places ; tous les joueurs sont vivants de l'image 0 à l'image 290. */
function match(parCamp: number) {
  const n = parCamp * 2
  const equipe = (i: number) => Math.floor(i / parCamp)
  const serie = (teamId: number, v: number) => {
    const points = [{ t: 0, v: 0 }, { t: 10, v }]
    return { teamId, rounds: [{ round: 0, points }], total: points }
  }
  const doc = testReplayDoc({
    frameCount: 300,
    frameIntervalMs: 100,
    originMs: 0,
    roster: Array.from({ length: n }, (_, i) => ({ xuid: `X${i}`, filmIndex: i, seat: i % parCamp, name: `Joueur${i}`, team: equipe(i) })),
    tracks: Array.from({ length: n }, (_, i) => ({
      slot: 512 + i, team: equipe(i), xuid: `X${i}`, startFrame: 0, endFrame: 290, points: [{ t: 0, x: i, y: i, sh: 1, hp: 1 }],
    })),
    scoreTimeline: { teams: [serie(0, 7), serie(1, 3)], players: [] },
  })
  const context = {
    header: { map_ui: 'Detachment', mode_ui: 'Assassin en équipe', playlist_label: 'Partie rapide', start_time_label: '10 nov. 2026' },
    scoreboard: Array.from({ length: n }, (_, i) => scoreboardRow(`X${i}`, `Joueur${i}`, equipe(i) === 0 ? 'a' : 'b', { is_me: i === 0 })),
    highlight_events: [],
    media: null,
    source_unavailable: false,
  }
  return { doc, context }
}

async function mount(largeur: number) {
  largeurCorps = largeur
  const { doc, context } = match(4)
  return mountReplayPage(doc, context)
}

const carte = (nom: string) => screen.getByRole('button', { name: PP.cardOpenFmt(nom) })
const fenetre = (nom: string) => screen.queryByRole('region', { name: PP.labelFmt(nom) })
const cadre = (c: HTMLElement) => c.querySelector('[data-replay-frame]')?.getAttribute('data-replay-frame')
const colonne = (c: HTMLElement) => c.querySelector<HTMLElement>('[data-replay-column]')
const frappe = (key: string) => fireEvent.keyDown(document.body, { key })

describe('ReplayPage : la fiche en fenêtre, rangée posée', () => {
  it.each(['cinema', 'plein-cadre'] as const)('%s : un clic sur une fiche de la rangée ouvre une fenêtre en portail, couche d’Échap, avec le corps commun', async (frame) => {
    const { container } = await mount(L4)
    goToFrame(frame)
    fireEvent.click(carte('Joueur5'))
    const f = fenetre('Joueur5') as HTMLElement
    expect(f).not.toBeNull()
    expect(container.contains(f)).toBe(false)
    expect(f.parentElement).toBe(document.body)
    expect(f).toHaveAttribute('data-replay-escape-layer')
    expect(f.style.width).toBe(`${COTES_FENETRE_FICHE.largeurPx}px`)
    for (const action of ['follow', 'solo', 'hide'] as const) {
      expect(within(f).getByRole('button', { name: PP.action[action] })).toHaveAttribute('aria-pressed', 'false')
    }
    expect(within(f).getByText(T.frames.escapeKey)).toBeInTheDocument()
    expect(within(f).getByRole('button', { name: PP.close })).toBeInTheDocument()
    expect(carte('Joueur5')).toHaveAttribute('aria-pressed', 'true')
  })

  it('la colonne ne se montre jamais, fiche ouverte ou non ; le fil complet garde son nœud, non masqué', async () => {
    const { container } = await mount(L4)
    goToFrame('cinema')
    const fil = screen.getByTestId('replay-feed-slot')
    expect(colonne(container)?.getAttribute('data-replay-column')).toBe('cachee')
    fireEvent.click(carte('Joueur5'))
    expect(colonne(container)?.getAttribute('data-replay-column')).toBe('cachee')
    expect(screen.getByTestId('replay-feed-slot')).toBe(fil)
    expect(fil).not.toHaveAttribute('aria-hidden')
    expect(container.querySelector('[data-replay-roster-column]')).toBeNull()
  })

  it('une autre fiche change de joueur sans fermer ; la même fiche, rouverte, ferme', async () => {
    await mount(L4)
    goToFrame('cinema')
    fireEvent.click(carte('Joueur5'))
    fireEvent.click(carte('Joueur2'))
    expect(fenetre('Joueur5')).toBeNull()
    expect(fenetre('Joueur2')).not.toBeNull()
    expect(document.querySelectorAll('[data-replay-fiche-popover]')).toHaveLength(1)
    expect(carte('Joueur2')).toHaveAttribute('aria-pressed', 'true')
    expect(carte('Joueur5')).toHaveAttribute('aria-pressed', 'false')
    fireEvent.click(carte('Joueur2'))
    expect(document.querySelector('[data-replay-fiche-popover]')).toBeNull()
  })

  it('la croix ferme la fenêtre', async () => {
    await mount(L4)
    goToFrame('cinema')
    fireEvent.click(carte('Joueur5'))
    fireEvent.click(within(fenetre('Joueur5') as HTMLElement).getByRole('button', { name: PP.close }))
    expect(document.querySelector('[data-replay-fiche-popover]')).toBeNull()
    expect(carte('Joueur5')).toHaveAttribute('aria-pressed', 'false')
  })

  it('Échap ferme la fenêtre ; au Plein cadre, la frappe suivante sort du cadre (une frappe, une couche)', async () => {
    const { container } = await mount(L4)
    goToFrame('plein-cadre')
    fireEvent.click(carte('Joueur5'))
    frappe('Escape')
    expect(document.querySelector('[data-replay-fiche-popover]')).toBeNull()
    expect(cadre(container)).toBe('plein-cadre')
    frappe('Escape')
    expect(cadre(container)).not.toBe('plein-cadre')
  })

  it('un geste sur le plan ou sur la barre ne ferme pas la fenêtre ; la lecture continue, Espace reste actif', async () => {
    const { container } = await mount(L4)
    goToFrame('cinema')
    fireEvent.click(carte('Joueur5'))
    const toile = container.querySelector('canvas') as HTMLCanvasElement
    fireEvent.pointerDown(toile)
    fireEvent.mouseDown(toile)
    fireEvent.click(toile)
    fireEvent.wheel(toile, { deltaY: -100 })
    fireEvent.click(screen.getByRole('button', { name: 'Lecture' }))
    expect(fenetre('Joueur5')).not.toBeNull()
    expect(screen.getByRole('button', { name: 'Pause' })).toBeInTheDocument()
    frappe(' ')
    expect(screen.getByRole('button', { name: 'Lecture' })).toBeInTheDocument()
    expect(fenetre('Joueur5')).not.toBeNull()
  })

  it('Suivre, Seul et Masquer agissent depuis la fenêtre, avec leurs exclusions', async () => {
    await mount(L4)
    goToFrame('cinema')
    fireEvent.click(carte('Joueur5'))
    const bouton = (action: 'follow' | 'solo' | 'hide') => within(fenetre('Joueur5') as HTMLElement).getByRole('button', { name: PP.action[action] })
    fireEvent.click(bouton('follow'))
    expect(bouton('follow')).toHaveAttribute('aria-pressed', 'true')
    fireEvent.click(bouton('solo'))
    expect(bouton('solo')).toHaveAttribute('aria-pressed', 'true')
    fireEvent.click(bouton('hide'))
    expect(bouton('hide')).toHaveAttribute('aria-pressed', 'true')
    expect(bouton('solo')).toHaveAttribute('aria-pressed', 'false')
    expect(bouton('follow')).toHaveAttribute('aria-pressed', 'false')
  })

  it('la fiche ouverte de la rangée prend le fond clair et la bordure claire (jetons)', async () => {
    await mount(L4)
    goToFrame('cinema')
    fireEvent.click(carte('Joueur5'))
    expect(carte('Joueur5').style.borderColor).toBe('var(--foreground)')
    expect(carte('Joueur5').style.background).toContain('var(--foreground)')
    expect(carte('Joueur2').style.borderColor).not.toBe('var(--foreground)')
  })

  it('la toile ne se remonte ni à l’ouverture, ni au changement de joueur, ni à la fermeture', async () => {
    const { container } = await mount(L4)
    goToFrame('cinema')
    const toile = container.querySelector('canvas')
    fireEvent.click(carte('Joueur5'))
    fireEvent.click(carte('Joueur2'))
    fireEvent.click(carte('Joueur2'))
    expect(container.querySelector('canvas')).toBe(toile)
    expect(container.querySelectorAll('canvas')).toHaveLength(1)
  })
})

describe('ReplayPage : la fiche en fenêtre, repli (équipes en colonne)', () => {
  it.each(['cinema', 'plein-cadre'] as const)('%s : un clic sur une ligne de la colonne ouvre la fenêtre ; la colonne garde équipes et fil', async (frame) => {
    const { container } = await mount(L4 - 1)
    goToFrame(frame)
    const fil = screen.getByTestId('replay-feed-slot')
    const equipes = container.querySelector('[data-replay-roster-column]')
    fireEvent.click(carte('Joueur5'))
    const f = fenetre('Joueur5') as HTMLElement
    expect(f).not.toBeNull()
    expect(container.contains(f)).toBe(false)
    expect(carte('Joueur5')).toHaveAttribute('aria-pressed', 'true')
    expect(colonne(container)?.getAttribute('data-replay-column')).toBe('affichee')
    expect(container.querySelector('[data-replay-roster-column]')).toBe(equipes)
    expect(screen.getByTestId('replay-feed-slot')).toBe(fil)
    expect(fil).not.toHaveAttribute('aria-hidden')
    expect(container.querySelector('[data-replay-fiche-column]')).toBeNull()
  })

  it('changer de ligne change de joueur ; Échap, la croix et la même ligne ferment', async () => {
    await mount(L4 - 1)
    goToFrame('cinema')
    fireEvent.click(carte('Joueur5'))
    fireEvent.click(carte('Joueur2'))
    expect(fenetre('Joueur5')).toBeNull()
    expect(fenetre('Joueur2')).not.toBeNull()
    frappe('Escape')
    expect(fenetre('Joueur2')).toBeNull()
    fireEvent.click(carte('Joueur2'))
    fireEvent.click(within(fenetre('Joueur2') as HTMLElement).getByRole('button', { name: PP.close }))
    expect(fenetre('Joueur2')).toBeNull()
    fireEvent.click(carte('Joueur2'))
    fireEvent.click(carte('Joueur2'))
    expect(fenetre('Joueur2')).toBeNull()
  })

  it('la fenêtre ouverte survit au passage de la rangée au repli : elle se ré-ancre sur la ligne', async () => {
    await mount(L4)
    goToFrame('cinema')
    fireEvent.click(carte('Joueur5'))
    largeurCorps = L4 - 1
    act(() => {
      for (const { cible, rappel } of [...observateurs]) if (cible.hasAttribute('data-replay-body')) rappel()
    })
    expect(fenetre('Joueur5')).not.toBeNull()
    expect(document.querySelectorAll('[data-replay-fiche-popover]')).toHaveLength(1)
    expect(carte('Joueur5')).toHaveAttribute('aria-pressed', 'true')
  })
})

describe('ReplayPage : où la fenêtre se pose', () => {
  it('bord gauche aligné sur la fiche, 8 px dessous', async () => {
    await mount(L4)
    goToFrame('cinema')
    rectFiche = { left: 140, right: 400, top: 10, bottom: 70 }
    fireEvent.click(carte('Joueur5'))
    const f = fenetre('Joueur5') as HTMLElement
    expect(f.style.left).toBe('140px')
    expect(f.style.top).toBe('78px')
  })

  it('près du bord droit de la page, elle s’aligne par la droite de la fiche et ne sort pas', async () => {
    await mount(L4)
    goToFrame('cinema')
    const largeurPage = window.innerWidth
    rectFiche = { left: largeurPage - 120, right: largeurPage - 20, top: 10, bottom: 70 }
    fireEvent.click(carte('Joueur5'))
    const f = fenetre('Joueur5') as HTMLElement
    const left = parseFloat(f.style.left)
    expect(left).toBe(largeurPage - 20 - COTES_FENETRE_FICHE.largeurPx)
    expect(left + COTES_FENETRE_FICHE.largeurPx).toBeLessThanOrEqual(largeurPage - 8)
  })

  it('la hauteur est bornée par le bas du plan, moins la marge', async () => {
    await mount(L4)
    goToFrame('cinema')
    basDuPlan = 400
    rectFiche = { left: 140, right: 400, top: 10, bottom: 70 }
    fireEvent.click(carte('Joueur5'))
    expect((fenetre('Joueur5') as HTMLElement).style.maxHeight).toBe(`${400 - 70 - 8 - 8}px`)
  })
})

describe('ReplayPage : le Standard garde sa fiche en colonne', () => {
  it('un clic sur une ligne remplace la colonne par la fiche : aucune fenêtre', async () => {
    const { container } = await mount(4000)
    fireEvent.click(carte('Joueur5'))
    expect(container.querySelector('[data-replay-fiche-column]')).not.toBeNull()
    expect(document.querySelector('[data-replay-fiche-popover]')).toBeNull()
    expect(screen.queryByRole('button', { name: PP.close })).toBeNull()
  })
})
