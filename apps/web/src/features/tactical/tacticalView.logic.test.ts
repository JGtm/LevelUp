import { describe, expect, it } from 'vitest'

import type { BornesMonde } from '@/lib/api/types'

import { getTacticalText } from './i18n'
import {
  cellFromClick,
  pageTitle,
  ratioSafe,
  sourceForQuestion,
  statusMessages,
  unitForQuestion,
} from './tacticalView.logic'

const tFr = getTacticalText('fr')
const tEn = getTacticalText('en')

const BORNES: BornesMonde = { min_x: 0, max_x: 100, min_y: 0, max_y: 50, valide: true }

// ─── Titre de la vue ───────────────────────────────────────────────────────────

describe('pageTitle — « Plan de <carte> — <question> »', () => {
  it('assemble le nom de carte et le libellé FR de la question', () => {
    expect(pageTitle(tFr, 'Aquarius', 'morts')).toBe('Plan de Aquarius — Où je meurs')
  })

  it('assemble le nom de carte et le libellé EN de la question', () => {
    expect(pageTitle(tEn, 'Aquarius', 'morts')).toBe('Plan of Aquarius — Where I die')
  })

  it('change de libellé avec la question, à carte fixe', () => {
    expect(pageTitle(tFr, 'Aquarius', 'kills')).toBe('Plan de Aquarius — Où je tue')
    expect(pageTitle(tFr, 'Aquarius', 'routes')).toBe('Plan de Aquarius — Mes routes de spawn')
  })
})

// ─── Unité par question ────────────────────────────────────────────────────────

describe('unitForQuestion — une unité distincte par question', () => {
  it.each([
    ['morts', 'morts par match'],
    ['kills', 'frags par match'],
    ['gagne', 'engagements par match'],
    ['temps', 'secondes par match'],
    ['routes', 'passages par match'],
    ['isole', 'morts isolées par match'],
  ] as const)('%s -> %s', (question, unite) => {
    expect(unitForQuestion(tFr, question)).toBe(unite)
  })
})

// ─── Source de la mesure ───────────────────────────────────────────────────────

describe('sourceForQuestion — artefact de rejeu vs journal des morts', () => {
  it('« temps » et « routes » exigent l’artefact de rejeu', () => {
    expect(sourceForQuestion(tFr, 'temps')).toBe(tFr.sourceReplay)
    expect(sourceForQuestion(tFr, 'routes')).toBe(tFr.sourceReplay)
  })

  it('les autres questions lisent le journal des morts', () => {
    expect(sourceForQuestion(tFr, 'morts')).toBe(tFr.sourceJournal)
    expect(sourceForQuestion(tFr, 'kills')).toBe(tFr.sourceJournal)
    expect(sourceForQuestion(tFr, 'gagne')).toBe(tFr.sourceJournal)
    expect(sourceForQuestion(tFr, 'isole')).toBe(tFr.sourceJournal)
  })
})

// ─── Messages de statut ────────────────────────────────────────────────────────

describe('statusMessages — bandeaux « en attente » / « non disponible »', () => {
  it('aucun message quand les deux compteurs sont à zéro', () => {
    expect(statusMessages(tFr, 0, 0)).toEqual([])
  })

  it('seulement « en attente » quand matchsNonCuisables est nul', () => {
    const messages = statusMessages(tFr, 3, 0)
    expect(messages).toHaveLength(1)
    expect(messages[0]).toBe(tFr.statusPending(3))
  })

  it('seulement « non disponible » quand matchsEnAttente est nul', () => {
    const messages = statusMessages(tFr, 0, 2)
    expect(messages).toHaveLength(1)
    expect(messages[0]).toBe(tFr.statusUnavailable(2))
  })

  it('LES DEUX coexistent : cuisson en cours ET matchs jamais cuisables', () => {
    const messages = statusMessages(tFr, 5, 1)
    expect(messages).toEqual([tFr.statusPending(5), tFr.statusUnavailable(1)])
  })
})

// ─── Cellule sous un clic ──────────────────────────────────────────────────────

describe('cellFromClick — la cellule (col, row) sous un clic canvas', () => {
  it('coin haut-gauche du canvas -> cellule (0, 0)', () => {
    expect(cellFromClick(0, 0, 200, 100, BORNES, 10)).toEqual({ col: 0, row: 0 })
  })

  it('centre du canvas -> la cellule du centre du monde', () => {
    // Canvas 200x100 px pour un monde 100x50 m, pas de 10 m : centre du monde =
    // (50, 25), donc col = 5, row = 2 (pas de 10 -> colonnes 0..9, lignes 0..4).
    expect(cellFromClick(100, 50, 200, 100, BORNES, 10)).toEqual({ col: 5, row: 2 })
  })

  it('un clic hors canvas ne rend rien', () => {
    expect(cellFromClick(-1, 0, 200, 100, BORNES, 10)).toBeNull()
    expect(cellFromClick(0, 101, 200, 100, BORNES, 10)).toBeNull()
  })

  it('des bornes invalides ne rendent rien', () => {
    expect(cellFromClick(10, 10, 200, 100, { ...BORNES, valide: false }, 10)).toBeNull()
  })

  it('un pas de grille nul ou négatif ne rend rien', () => {
    expect(cellFromClick(10, 10, 200, 100, BORNES, 0)).toBeNull()
  })
})

// ─── ratioSafe ──────────────────────────────────────────────────────────────────

describe('ratioSafe — une proportion, jamais une division par zéro', () => {
  it('divise normalement', () => {
    expect(ratioSafe(3, 12)).toBe(0.25)
  })

  it('rend 0 sur un dénominateur nul ou négatif', () => {
    expect(ratioSafe(3, 0)).toBe(0)
    expect(ratioSafe(3, -1)).toBe(0)
  })
})
