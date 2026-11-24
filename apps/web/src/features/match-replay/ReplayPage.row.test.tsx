/**
 * Tests — LA RANGÉE DES ÉQUIPES du Cinéma et du Plein cadre, dans la vraie page.
 *
 * Monte la vraie page, la vraie toile (contexte 2D enregistreur) et un document à deux camps AVEC calque de
 * score ; la largeur du corps du cadre est posée par le test (jsdom ne met rien en page) et suivie par un
 * observateur de taille simulé. Ce que ces tests prouvent :
 *  - la rangée est posée quand la règle (`dispositionEquipes`) la retient : le score est AU CENTRE de la rangée
 *    (emplacement `rangee`), il n'est plus sur la toile, et le fil des éliminations passe SUR LE PLAN (la colonne
 *    n'est jamais affichée, son fil complet reste monté) ;
 *  - sinon (fenêtre trop étroite, largeur pas mesurée, chacun pour soi, Standard) les équipes sont dans la colonne
 *    et le score est dans l'en-tête, sur la toile seulement si l'en-tête est trop étroit pour le loger ;
 *  - passer de la rangée au repli (la fenêtre rétrécit) et basculer de cadre ne remontent NI la toile NI la
 *    cellule du fil.
 */
import { fireEvent, screen, within } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { REPLAY_TEXT } from './i18n/i18n'
import { largeurRequiseEnteteScore } from './model/cotesEnteteScore'
import { largeurRequiseRangee, largeurRequiseRangeeGrandeEquipe, MARGE_BASCULE_PX } from './model/rowLayout'
import { goToFrame, unstubReplayPage } from './test/replayPageHarness'
import {
  bandeaux,
  colonneDesEquipes,
  installerLesDoublesDeLaRangee,
  L4,
  LARGEUR_REQUISE_ENTETE_SCORE_PX,
  LGE,
  match,
  mount,
  rangee,
  redimensionner,
} from './test/replayRowKit'

vi.mock('@tanstack/react-router', async (importOriginal) =>
  (await import('./test/replayPageRouter')).routerMock(await importOriginal<typeof import('@tanstack/react-router')>()),
)

const T = REPLAY_TEXT.fr

beforeEach(installerLesDoublesDeLaRangee)

afterEach(() => {
  unstubReplayPage()
})

describe('ReplayPage : la rangée des équipes, 4 contre 4', () => {
  it.each(['cinema', 'plein-cadre'] as const)('%s : la rangée est posée, le score est à son centre et plus sur la toile', async (frame) => {
    const { container } = await mount(match(4), L4)
    goToFrame(frame)
    const ligne = rangee(container)
    expect(ligne).not.toBeNull()
    expect(ligne?.getAttribute('data-replay-teams-row')).toBe('fiches')
    expect(ligne?.querySelectorAll('[data-replay-player-card]')).toHaveLength(8)
    const [banniere, ...autres] = bandeaux(container)
    expect(autres).toHaveLength(0)
    expect(banniere.getAttribute('data-replay-score-banner')).toBe('rangee')
    expect(ligne?.querySelector('[data-replay-teams-row-score]')?.contains(banniere)).toBe(true)
    expect(container.querySelector('section')?.contains(banniere)).toBe(false)
    expect(container.querySelector('[data-replay-body] > [data-replay-teams-row]')).not.toBeNull()
  })

  it('la colonne ne montre plus les équipes : le fil complet reste monté (caché), et la fiche du joueur s’ouvre en fenêtre, hors de la colonne', async () => {
    const { container } = await mount(match(4), L4)
    goToFrame('cinema')
    expect(colonneDesEquipes(container)).toBeNull()
    expect(screen.getByTestId('replay-feed-slot')).toBeInTheDocument()
    fireEvent.click(within(rangee(container) as HTMLElement).getByRole('button', { name: 'Ouvrir la fiche de Joueur5' }))
    expect(within(rangee(container) as HTMLElement).getByRole('button', { name: 'Ouvrir la fiche de Joueur5' })).toHaveAttribute('aria-pressed', 'true')
    expect(screen.getByTestId('replay-feed-slot')).not.toHaveAttribute('aria-hidden')
    const fenetre = screen.getByRole('region', { name: T.playerPanel.labelFmt('Joueur5') })
    expect(fenetre).toHaveAttribute('data-replay-fiche-popover')
    expect(container.contains(fenetre)).toBe(false)
  })

  it('« Prendre le point de vue » dans la fiche : le point de vue passe au joueur ouvert, et le bouton disparaît', async () => {
    const { container } = await mount(match(4), L4)
    goToFrame('cinema')
    const fiches = within(rangee(container) as HTMLElement)
    fireEvent.click(fiches.getByRole('button', { name: 'Ouvrir la fiche de Joueur5' }))
    expect(fiches.getByRole('button', { name: 'Ouvrir la fiche de Joueur0' }).style.borderColor).toBe('var(--ring)')
    fireEvent.click(screen.getByRole('button', { name: T.playerPanel.viewpoint.label }))
    // La bordure de point de vue quitte le joueur de la page ; la fiche ouverte garde la sienne (bordure de sélection).
    expect(fiches.getByRole('button', { name: 'Ouvrir la fiche de Joueur0' }).style.borderColor).toBe('var(--border)')
    expect(screen.queryByRole('button', { name: T.playerPanel.viewpoint.label })).toBeNull()
  })

  it('le joueur de la page est le joueur suivi : sa fiche a la bordure de suivi', async () => {
    const { container } = await mount(match(4), L4)
    goToFrame('cinema')
    const fiches = within(rangee(container) as HTMLElement)
    expect(fiches.getByRole('button', { name: 'Ouvrir la fiche de Joueur0' }).style.borderColor).toBe('var(--ring)')
    expect(fiches.getByRole('button', { name: 'Ouvrir la fiche de Joueur1' }).style.borderColor).toBe('var(--border)')
  })

  it('Plein cadre : la marge de corps de 4 px fait tenir la rangée 40 px plus tôt qu’au Cinéma, au pixel près', async () => {
    const demande = largeurRequiseRangee(4, 'plein-cadre')
    expect(demande).toBe(L4 - 40)
    const pose = await mount(match(4), demande)
    goToFrame('plein-cadre')
    expect(rangee(pose.container)).not.toBeNull()
    pose.unmount()
    const repli = await mount(match(4), demande - 1)
    goToFrame('plein-cadre')
    expect(rangee(repli.container)).toBeNull()
    repli.unmount()
    // La même largeur ne suffit pas au Cinéma.
    const cinema = await mount(match(4), demande)
    goToFrame('cinema')
    expect(rangee(cinema.container)).toBeNull()
  })

  it('trop étroit (un pixel de moins que la demande) : le repli, équipes en lignes dans la colonne, score dans l’en-tête', async () => {
    const { container } = await mount(match(4), L4 - 1)
    goToFrame('cinema')
    expect(rangee(container)).toBeNull()
    expect(colonneDesEquipes(container)).not.toBeNull()
    const [banniere, ...autres] = bandeaux(container)
    expect(autres).toHaveLength(0)
    expect(banniere.getAttribute('data-replay-score-banner')).toBe('entete')
    expect(container.querySelector('section')?.contains(banniere)).toBe(false)
    expect(container.querySelector('[data-replay-header="cinema"] [data-replay-header-cell="centre"]')?.contains(banniere)).toBe(true)
  })

  it.each(['cinema', 'plein-cadre'] as const)('%s, repli : le score est dans l’en-tête tant que le corps loge sa cellule de 440 px, sur la toile une fois la marge de bascule dépassée', async (frame) => {
    const requis = largeurRequiseEnteteScore(frame)
    const { container } = await mount(match(4), requis)
    goToFrame(frame)
    expect(rangee(container)).toBeNull()
    expect(bandeaux(container).map((b) => b.getAttribute('data-replay-score-banner'))).toEqual(['entete'])
    expect(container.querySelector(`[data-replay-header="${frame}"][data-replay-header-score]`)).not.toBeNull()
    redimensionner(requis - MARGE_BASCULE_PX - 1)
    expect(bandeaux(container).map((b) => b.getAttribute('data-replay-score-banner'))).toEqual(['scene'])
    expect(container.querySelector('section')?.contains(bandeaux(container)[0])).toBe(true)
    expect(container.querySelector('[data-replay-header-score]')).toBeNull()
  })

  it('rangée posée : l’en-tête garde sa ligne de 52 px, sans cellule de score', async () => {
    const { container } = await mount(match(4), L4)
    goToFrame('cinema')
    expect(container.querySelector('[data-replay-header-score]')).toBeNull()
    expect(bandeaux(container).map((b) => b.getAttribute('data-replay-score-banner'))).toEqual(['rangee'])
  })

  it('largeur du corps non mesurée : le repli', async () => {
    const { container } = await mount(match(4), 0)
    goToFrame('cinema')
    expect(rangee(container)).toBeNull()
    expect(colonneDesEquipes(container)).not.toBeNull()
  })

  it('le Standard : équipes en lignes dans sa colonne, score dans l’en-tête, jamais de rangée', async () => {
    const { container } = await mount(match(4), 4000)
    expect(rangee(container)).toBeNull()
    expect(colonneDesEquipes(container)).not.toBeNull()
    expect(container.querySelectorAll('[data-replay-player-card]')).toHaveLength(8)
    expect(bandeaux(container).map((b) => b.getAttribute('data-replay-score-banner'))).toEqual(['entete'])
  })

  it('le Standard étroit (sous la place de la cellule du score) : le score est sur la toile, l’en-tête tient sur une ligne', async () => {
    const { container } = await mount(match(4), LARGEUR_REQUISE_ENTETE_SCORE_PX - 1)
    expect(bandeaux(container).map((b) => b.getAttribute('data-replay-score-banner'))).toEqual(['scene'])
    expect(container.querySelector('section')?.contains(bandeaux(container)[0])).toBe(true)
    const entete = container.querySelector('[data-replay-header="standard"]') as HTMLElement
    expect(entete).not.toBeNull()
    expect(entete.querySelector('[data-replay-header-cell="centre"]')).toBeNull()
    expect(entete.querySelector('[data-replay-header-cell="gauche"]')).not.toBeNull()
    expect(entete.querySelector('[data-replay-header-cell="droite"]')).not.toBeNull()
    redimensionner(LARGEUR_REQUISE_ENTETE_SCORE_PX)
    expect(bandeaux(container).map((b) => b.getAttribute('data-replay-score-banner'))).toEqual(['entete'])
    expect(container.querySelector('[data-replay-header="standard"] [data-replay-header-cell="centre"]')).not.toBeNull()
  })

  it('chacun pour soi (un seul camp) : le repli, même sur un corps très large', async () => {
    const { container } = await mount(match(8, 1), 4000)
    goToFrame('cinema')
    expect(rangee(container)).toBeNull()
    expect(colonneDesEquipes(container)).not.toBeNull()
  })
})

describe('ReplayPage : la rangée des équipes en Grande équipe', () => {
  it('12 contre 12 : six emplacements de deux fiches par camp, le score au centre', async () => {
    const { container } = await mount(match(12), LGE)
    goToFrame('cinema')
    const ligne = rangee(container) as HTMLElement
    expect(ligne.getAttribute('data-replay-teams-row')).toBe('grande-equipe')
    expect(ligne.querySelectorAll('section')).toHaveLength(2)
    for (const camp of ligne.querySelectorAll('section')) expect(camp.children).toHaveLength(6)
    expect(ligne.querySelectorAll('[data-replay-player-card]')).toHaveLength(24)
    expect(bandeaux(container).map((b) => b.getAttribute('data-replay-score-banner'))).toEqual(['rangee'])
  })

  it('trop étroit pour la Grande équipe : le repli, équipes en paires dans la colonne, score dans l’en-tête', async () => {
    const { container } = await mount(match(12), largeurRequiseRangeeGrandeEquipe('plein-cadre') - 1)
    goToFrame('plein-cadre')
    expect(rangee(container)).toBeNull()
    expect(colonneDesEquipes(container)).not.toBeNull()
    expect(bandeaux(container).map((b) => b.getAttribute('data-replay-score-banner'))).toEqual(['entete'])
    expect(container.querySelector('[data-replay-header="plein-cadre"] [data-replay-header-cell="centre"]')?.contains(bandeaux(container)[0])).toBe(true)
  })
})

describe('ReplayPage : la rangée ne remonte ni la toile ni le fil', () => {
  it('Standard, Cinéma, Plein cadre, retour : la même toile et la même cellule du fil, rangée posée', async () => {
    const { container } = await mount(match(4), L4)
    const toile = container.querySelector('canvas')
    const fil = screen.getByTestId('replay-feed-slot')
    for (const frame of ['cinema', 'plein-cadre', 'standard', 'plein-cadre', 'cinema', 'standard'] as const) {
      goToFrame(frame)
      expect(container.querySelector('canvas')).toBe(toile)
      expect(screen.getByTestId('replay-feed-slot')).toBe(fil)
    }
  })

  it('la fenêtre rétrécit puis s’élargit au Cinéma : rangée, repli, rangée, sans remonter la toile ni le fil', async () => {
    const { container } = await mount(match(4), L4)
    goToFrame('cinema')
    const toile = container.querySelector('canvas')
    const fil = screen.getByTestId('replay-feed-slot')
    expect(rangee(container)).not.toBeNull()

    redimensionner(L4 - MARGE_BASCULE_PX - 1)
    expect(rangee(container)).toBeNull()
    expect(colonneDesEquipes(container)).not.toBeNull()
    expect(bandeaux(container).map((b) => b.getAttribute('data-replay-score-banner'))).toEqual(['entete'])
    expect(container.querySelector('canvas')).toBe(toile)
    expect(screen.getByTestId('replay-feed-slot')).toBe(fil)

    redimensionner(L4)
    expect(rangee(container)).not.toBeNull()
    expect(colonneDesEquipes(container)).toBeNull()
    expect(bandeaux(container).map((b) => b.getAttribute('data-replay-score-banner'))).toEqual(['rangee'])
    expect(container.querySelector('canvas')).toBe(toile)
    expect(screen.getByTestId('replay-feed-slot')).toBe(fil)
  })
})

describe('ReplayPage : la marge de bascule de la rangée et du score', () => {
  const emplacementsDuScore = (c: HTMLElement) => bandeaux(c).map((b) => b.getAttribute('data-replay-score-banner'))

  it.each(['cinema', 'plein-cadre'] as const)('%s, 4 contre 4 : 1290, 1275, 1290 px ne font pas clignoter la rangée (seuil 1286 px)', async (frame) => {
    const { container } = await mount(match(4), 1290)
    goToFrame(frame)
    expect(rangee(container)).not.toBeNull()
    const toile = container.querySelector('canvas')
    redimensionner(1275)
    expect(rangee(container)).not.toBeNull()
    expect(colonneDesEquipes(container)).toBeNull()
    expect(emplacementsDuScore(container)).toEqual(['rangee'])
    redimensionner(1290)
    expect(rangee(container)).not.toBeNull()
    expect(container.querySelector('canvas')).toBe(toile)
  })

  it('la rangée posée ne se défait que sous la demande moins 24 px, et ne revient qu’à la demande', async () => {
    const { container } = await mount(match(4), L4)
    goToFrame('cinema')
    redimensionner(L4 - MARGE_BASCULE_PX)
    expect(rangee(container)).not.toBeNull()
    redimensionner(L4 - MARGE_BASCULE_PX - 1)
    expect(rangee(container)).toBeNull()
    redimensionner(L4 - 1)
    expect(rangee(container)).toBeNull()
    redimensionner(L4)
    expect(rangee(container)).not.toBeNull()
  })

  it('une colonne ne devient pas rangée sur la marge : monté à 1275 px, il faut 1286 px', async () => {
    const { container } = await mount(match(4), 1275)
    goToFrame('cinema')
    expect(rangee(container)).toBeNull()
    redimensionner(1285)
    expect(rangee(container)).toBeNull()
    redimensionner(1286)
    expect(rangee(container)).not.toBeNull()
  })

  it('le score reste dans l’en-tête entre 796 et 819 px, et ne remonte de la toile qu’à 820 px', async () => {
    const { container } = await mount(match(4), LARGEUR_REQUISE_ENTETE_SCORE_PX)
    goToFrame('cinema')
    expect(emplacementsDuScore(container)).toEqual(['entete'])
    redimensionner(LARGEUR_REQUISE_ENTETE_SCORE_PX - MARGE_BASCULE_PX)
    expect(emplacementsDuScore(container)).toEqual(['entete'])
    redimensionner(LARGEUR_REQUISE_ENTETE_SCORE_PX - MARGE_BASCULE_PX - 1)
    expect(emplacementsDuScore(container)).toEqual(['scene'])
    redimensionner(LARGEUR_REQUISE_ENTETE_SCORE_PX - 1)
    expect(emplacementsDuScore(container)).toEqual(['scene'])
    redimensionner(LARGEUR_REQUISE_ENTETE_SCORE_PX)
    expect(emplacementsDuScore(container)).toEqual(['entete'])
  })
})
