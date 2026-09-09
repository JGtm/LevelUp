import { describe, expect, it } from 'vitest'

import { tacticalIntensity } from '@/lib/replay/heatPaint'
import type { BornesMonde, CelluleTactique, EchelleTactique } from '@/lib/api/types'

import { getTacticalText } from './i18n'
import {
  cellFromClick,
  pageTitle,
  planCanvasView,
  ratioSafe,
  sourceForQuestion,
  statusMessages,
  tacticalGridFromRaster,
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

  // L'ADRESSE EST ANCRÉE SUR L'ORIGINE DU MONDE, PAS SUR LES BORNES DE LA LECTURE — même
  // convention que `CelluleTactique.col/lig` côté serveur (`tactical.Grille.Cellule`).
  // Une adresse relative aux bornes ne retrouvait aucune cellule serveur dès que la
  // carte n'était pas calée sur (0, 0), c'est-à-dire sur toutes les cartes réelles.
  it('rend une adresse ancrée sur l’origine du monde, bornes négatives comprises', () => {
    const bornes: BornesMonde = { min_x: -20, max_x: -10, min_y: -8, max_y: -4, valide: true }
    // Coin haut-gauche du canvas = (-20, -8) monde ; pas de 2 m -> (-10, -4).
    expect(cellFromClick(0, 0, 200, 80, bornes, 2)).toEqual({ col: -10, row: -4 })
    // Centre du canvas = (-15, -6) monde -> colonne -8, ligne -3.
    expect(cellFromClick(100, 40, 200, 80, bornes, 2)).toEqual({ col: -8, row: -3 })
  })

  it('suit le pas publié : la même position rend une adresse différente à 0,5 et à 2 m', () => {
    expect(cellFromClick(100, 50, 200, 100, BORNES, 0.5)).toEqual({ col: 100, row: 50 })
    expect(cellFromClick(100, 50, 200, 100, BORNES, 2)).toEqual({ col: 25, row: 12 })
  })
})

// ─── planCanvasView — la projection monde -> canvas du calque de chaleur ─────────

describe('planCanvasView — le cadrage du calque de chaleur', () => {
  it('pose l’origine du canvas sur (min_x, min_y), pas sur l’origine du monde', () => {
    const bornes: BornesMonde = { min_x: -20, max_x: -10, min_y: -8, max_y: -4, valide: true }
    // 200 px pour 10 m -> 20 px/m ; la colonne 0 du monde (x = 0) tomberait à 400 px.
    expect(planCanvasView(bornes, 200)).toEqual({ topLeftWorld: { x: 400, y: 160 }, scale: 20 })
  })

  it('rend une origine nulle quand les bornes partent de (0, 0)', () => {
    expect(planCanvasView(BORNES, 200)).toEqual({ topLeftWorld: { x: 0, y: 0 }, scale: 2 })
  })

  it('ne rend rien sur des bornes invalides ou un canvas vide', () => {
    expect(planCanvasView({ ...BORNES, valide: false }, 200)).toBeNull()
    expect(planCanvasView(BORNES, 0)).toBeNull()
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

// ─── tacticalGridFromRaster — la grille de peinture (noyau lib/replay/heatPaint, Q7) ──────

/** Une cellule serveur minimale — les champs que `tacticalGridFromRaster` ne lit pas
 *  (`brut`, `matchs*`, `centre_*`) sont posés à 0, hors-sujet pour cette conversion. */
function celluleDe(col: number, lig: number, valeur: number): CelluleTactique {
  return { col, lig, valeur, brut: valeur, centre_x: 0, centre_y: 0, matchs: 3, matchs_defaite: 0, matchs_victoire: 0 }
}

const ECHELLE: EchelleTactique = { p50: 2, p95: 8, n_cellules: 5, borne: 8, symetrique: false }

describe('tacticalGridFromRaster — bornes + pas -> grille de peinture', () => {
  it('assemble dimensions, cellules et échelle depuis le raster serveur', () => {
    const g = tacticalGridFromRaster([celluleDe(1, 2, 6)], BORNES, 10, ECHELLE)
    expect(g).not.toBeNull()
    expect(g).toMatchObject({
      cell: 10,
      nx: 10, // (100 - 0) / 10
      ny: 5, // (50 - 0) / 10
      minX: 0,
      minY: 0,
      lo: 2,
      hi: 8,
      filled: 5,
    })
    expect(g!.cells).toEqual([{ col: 1, row: 2, value: 6 }])
  })

  it('des bornes invalides ne rendent rien', () => {
    expect(tacticalGridFromRaster([], { ...BORNES, valide: false }, 10, ECHELLE)).toBeNull()
  })

  it('un pas de grille nul ou négatif ne rend rien', () => {
    expect(tacticalGridFromRaster([], BORNES, 0, ECHELLE)).toBeNull()
  })

  it('SNAPSHOT LÉGER — intensités sur une petite grille, sans canvas (entrée cellules)', () => {
    const g = tacticalGridFromRaster(
      [celluleDe(0, 0, 0), celluleDe(1, 0, 2), celluleDe(2, 0, 6)],
      { min_x: 0, max_x: 3, min_y: 0, max_y: 1, valide: true },
      1,
      ECHELLE,
    )!
    // p50 = 2, p95 = 8 (ECHELLE) : valeur 0 (jamais atteinte) -> null ; 2 -> bas d'échelle
    // (0) ; 6 -> aux deux tiers de l'échelle — même règle que `heatIntensity` (entrée points).
    const intensites = g.cells.map((c) => tacticalIntensity(g, c.value))
    expect(intensites).toEqual([null, 0, 2 / 3])
  })
})

// instantToFrame / TACTICAL_REPLAY_FRAME_INTERVAL_MS (lot M1, conversion mécanique instant ->
// frame) sont retirées le 2026-09-08 (lot M1b) avec leurs tests : la conversion vit désormais
// dans `lib/replay/replayLogic.resolveTacticalReplayInstant` (+ `msToFrames`), testée là-bas.
