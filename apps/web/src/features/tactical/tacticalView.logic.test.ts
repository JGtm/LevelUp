import { describe, expect, it } from 'vitest'

import { tacticalIntensity } from '@/lib/replay/heatPaint'
import type { BornesMonde, CelluleTactique, EchelleTactique } from '@/lib/api/types'

import { getTacticalText } from './i18n'
import {
  celluleDuClic,
  grilleDuPlan,
  pageTitle,
  planEmptyReason,
  planEmptyText,
  planLegend,
  questionSansCellule,
  ratioSafe,
  rectSelection,
  repereAspect,
  repereDuPlan,
  sourceForQuestion,
  statusMessages,
  unitForQuestion,
  vueContain,
  vueDuPlan,
} from './tacticalView.logic'

const tFr = getTacticalText('fr')
const tEn = getTacticalText('en')


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

// ─── planEmptyReason — CE QUE LE PLAN VIDE DIT, ET IL DOIT DIRE VRAI ────────────

describe('planEmptyReason — trois causes de plan vide, trois messages', () => {
  it('rien n’est vide quand au moins une cellule est peinte', () => {
    expect(planEmptyReason(4, 38, 38)).toBeNull()
    // Une seule cellule suffit : le plan est maigre, pas vide.
    expect(planEmptyReason(1, 38, 38)).toBeNull()
  })

  it('aucun match dans le filtre : le périmètre est vide, pas la mesure', () => {
    expect(planEmptyReason(0, 0, 0)).toBe('aucun-match')
  })

  it('des matchs mais aucun mesuré : « pas assez de matchs mesurés » reste vrai', () => {
    expect(planEmptyReason(0, 0, 38)).toBe('aucune-mesure')
  })

  // LE DÉFAUT DU POINT 21 : sur Illusion, 38 matchs filtrés et mesurés, aucune cellule
  // au-dessus du plancher. Le message « pas assez de matchs mesurés » était FAUX.
  it('des matchs mesurés mais dispersés : densité insuffisante, jamais « pas assez de matchs »', () => {
    expect(planEmptyReason(0, 38, 38)).toBe('densite')
    expect(planEmptyReason(0, 3, 38)).toBe('densite')
  })
})

describe('planEmptyText — le titre et la description de chaque cause', () => {
  it('reprend le message existant pour « aucune mesure »', () => {
    expect(planEmptyText(tFr, 'aucune-mesure', 0, 0.5)).toEqual({
      title: tFr.planEmptyTitle,
      description: tFr.planEmptyDescription,
    })
  })

  it('la densité insuffisante cite les matchs mesurés, le plancher et le pas essayé', () => {
    const densite = planEmptyText(tFr, 'densite', 38, 2)
    expect(densite.title).toBe(tFr.planEmptyDensityTitle)
    expect(densite.description).toContain('38 matchs mesurés')
    expect(densite.description).toContain('3 matchs distincts')
    expect(densite.description).toContain('2 m')
    // Ce que la page ne doit PLUS dire quand les matchs sont là.
    expect(densite.title).not.toBe(tFr.planEmptyTitle)
  })

  it('la densité insuffisante existe aussi en anglais', () => {
    const densite = planEmptyText(tEn, 'densite', 38, 2)
    expect(densite.title).toBe(tEn.planEmptyDensityTitle)
    expect(densite.description).toContain('38 measured matches')
  })

  it('un périmètre vide garde son propre message', () => {
    expect(planEmptyText(tFr, 'aucun-match', 0, 0.5)).toEqual({
      title: tFr.planEmptyNoMatchTitle,
      description: tFr.planEmptyNoMatchDescription,
    })
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

// ─── Conformité à la maquette 034b1915 (lot F, 2026-09-13) ────────────────────

const fmt = (n: number) => String(n)

describe('planLegend — les deux bornes de la rampe et le mode de rampe', () => {
  it('lecture de quantile : de 0 au p95, unité sur la borne haute seulement', () => {
    const echelle: EchelleTactique = {
      p50: 0.4, p95: 1.2, borne: 0, symetrique: false, n_cellules: 30,
    }
    expect(planLegend(echelle, 'morts par match', fmt)).toEqual({
      lo: '0',
      hi: '1.2 morts par match',
      mode: 'intensity',
    })
  })

  it('lecture signée : de −borne à +borne, rampe divergente', () => {
    const echelle: EchelleTactique = {
      p50: 0.2, p95: 0.9, borne: 0.9, symetrique: true, n_cellules: 30,
    }
    expect(planLegend(echelle, 'écart V − D', fmt)).toEqual({
      lo: '− 0.9',
      hi: '+ 0.9 écart V − D',
      mode: 'divergent',
    })
  })
})

describe('questionSansCellule — « Mes routes de spawn » n’a pas de cellule', () => {
  it('vrai pour les routes, faux pour les cinq autres lectures', () => {
    expect(questionSansCellule('routes')).toBe(true)
    for (const q of ['morts', 'kills', 'gagne', 'temps', 'isole'] as const) {
      expect(questionSansCellule(q)).toBe(false)
    }
  })
})



// ─── LA PROJECTION DU PLAN (correctif du 2026-09-13) ──────────────────────────
//
// Le cadre du fond d'Illusion, tel que l'API le publie : 1711 x 2224 px a 0,031 m/px,
// origine monde (-25,751 ; 35,165) — soit 53,04 x 68,94 m.
const FOND_ILLUSION = {
  originX: -25.751434532165526,
  originY: 35.164999237060556,
  widthM: 1711 * 0.031,
  heightM: 2224 * 0.031,
}

// Les bornes que la lecture publiait pour les MEMES donnees : 30 x 36 m, une fenetre qui ne
// couvre que 40 % du fond. C'est tout l'ecart entre les deux reperes.
const BORNES_MORTS: BornesMonde = { min_x: -14, max_x: 16, min_y: -20, max_y: 16, valide: true }

const ECHELLE: EchelleTactique = {
  p50: 0.105, p95: 0.167, borne: 0.167, symetrique: false, n_cellules: 54,
}

describe('repereDuPlan — le repere est celui du FOND, pas celui des donnees', () => {
  it('prend le cadre du fond quand la carte en a un', () => {
    const r = repereDuPlan(FOND_ILLUSION, BORNES_MORTS, 2)!
    expect(r.minX).toBeCloseTo(-25.751, 3)
    expect(r.maxX).toBeCloseTo(-25.751 + 53.041, 3)
    expect(r.maxY).toBeCloseTo(35.165, 3)
    expect(repereAspect(r)).toBeCloseTo(53.041 / 68.944, 3)
  })

  it("retombe sur les bornes des cellules quand la carte n'a pas de fond fige", () => {
    const r = repereDuPlan(null, BORNES_MORTS, 2)!
    expect(r).toEqual({ minX: -14, maxX: 16, maxY: 16, minY: -20, pasM: 2 })
  })

  it('refuse un pas nul et des bornes inexploitables', () => {
    expect(repereDuPlan(FOND_ILLUSION, BORNES_MORTS, 0)).toBeNull()
    expect(repereDuPlan(null, { ...BORNES_MORTS, valide: false }, 2)).toBeNull()
  })
})

describe('grilleDuPlan — les cellules d’index NEGATIF sont peintes', () => {
  // LE DEFAUT FERME : le serveur adresse ses cellules sur l'origine du monde, donc en
  // nombres signes, et le peintre n'enumere que 0..nx / 0..ny. Sur Illusion, 11 cellules
  // etaient dessinees sur les 54 servies.
  const cellules: CelluleTactique[] = [
    { col: -7, lig: -10, valeur: 0.12, brut: 5, centre_x: -13, centre_y: -19, matchs: 3, matchs_victoire: 0, matchs_defaite: 0 },
    { col: -1, lig: -7, valeur: 0.16, brut: 9, centre_x: -1, centre_y: -13, matchs: 4, matchs_victoire: 0, matchs_defaite: 0 },
    { col: 7, lig: 7, valeur: 0.11, brut: 4, centre_x: 15, centre_y: 15, matchs: 3, matchs_victoire: 0, matchs_defaite: 0 },
  ]

  it('replace les trois cellules DANS la grille, indices positifs', () => {
    const r = repereDuPlan(FOND_ILLUSION, BORNES_MORTS, 2)!
    const grid = grilleDuPlan(cellules, r, ECHELLE)!
    expect(grid.filled).toBe(3)
    for (const c of grid.cells) {
      expect(c.col).toBeGreaterThanOrEqual(0)
      expect(c.row).toBeGreaterThanOrEqual(0)
      expect(c.col).toBeLessThan(grid.nx)
      expect(c.row).toBeLessThan(grid.ny)
    }
  })

  it('inverse Y : un Y monde ELEVE tombe dans une ligne HAUTE de l’image', () => {
    const r = repereDuPlan(FOND_ILLUSION, BORNES_MORTS, 2)!
    const grid = grilleDuPlan(cellules, r, ECHELLE)!
    const haut = grid.cells.find((c) => c.value === 0.11)! // centre_y = +15, proche du haut
    const bas = grid.cells.find((c) => c.value === 0.12)! // centre_y = -19, plus bas
    expect(haut.row).toBeLessThan(bas.row)
  })

  it('ignore une cellule hors du cadre du fond, jamais rabattue sur un bord', () => {
    const r = repereDuPlan(FOND_ILLUSION, BORNES_MORTS, 2)!
    const loin: CelluleTactique = { ...cellules[0], col: 900, lig: 900 }
    expect(grilleDuPlan([loin], r, ECHELLE)).toBeNull()
  })
})

describe('celluleDuClic — le clic rend l’adresse SERVEUR, et le cadre s’y repose', () => {
  it('aller-retour : cliquer au centre d’une cellule rend son adresse serveur', () => {
    const r = repereDuPlan(FOND_ILLUSION, BORNES_MORTS, 2)!
    const grid = { width: 1000, height: Math.round(1000 / repereAspect(r)) }
    // Centre monde de la cellule serveur (-1, -7) : (-1, -13).
    const x = ((-1 - r.minX) / (r.maxX - r.minX)) * grid.width
    const y = ((r.maxY - -13) / (r.maxY - r.minY)) * grid.height
    expect(celluleDuClic(x, y, grid, r)).toEqual({ col: -1, row: -7 })
  })

  it('hors du canvas : aucune cellule', () => {
    const r = repereDuPlan(FOND_ILLUSION, BORNES_MORTS, 2)!
    expect(celluleDuClic(-1, 10, { width: 100, height: 100 }, r)).toBeNull()
    expect(celluleDuClic(10, 101, { width: 100, height: 100 }, r)).toBeNull()
  })

  it('le cadre de selection tombe sur la MEME case que la peinture', () => {
    const r = repereDuPlan(FOND_ILLUSION, BORNES_MORTS, 2)!
    const grid = grilleDuPlan(
      [{ col: -1, lig: -7, valeur: 0.16, brut: 9, centre_x: -1, centre_y: -13, matchs: 4, matchs_victoire: 0, matchs_defaite: 0 }],
      r,
      ECHELLE,
    )!
    const vue = vueDuPlan(r, 1000)!
    const rect = rectSelection({ col: -1, row: -7 }, r, 1000)!
    expect(rect.x).toBeCloseTo(grid.cells[0].col * r.pasM * vue.scale, 6)
    expect(rect.y).toBeCloseTo(grid.cells[0].row * r.pasM * vue.scale, 6)
    expect(rect.size).toBeCloseTo(r.pasM * vue.scale, 6)
  })

  it('rien a encadrer hors du cadre, ou sans selection', () => {
    const r = repereDuPlan(FOND_ILLUSION, BORNES_MORTS, 2)!
    expect(rectSelection(null, r, 1000)).toBeNull()
    expect(rectSelection({ col: 900, row: 900 }, r, 1000)).toBeNull()
  })
})

describe('vueContain — la vignette ne deforme pas la carte', () => {
  it('une seule echelle, la plus contraignante, et des marges de centrage', () => {
    const r = repereDuPlan(FOND_ILLUSION, BORNES_MORTS, 2)!
    // Cadre 16:9 sur un monde en hauteur : c'est la HAUTEUR qui contraint.
    const vue = vueContain(r, 320, 180)!
    expect(vue.scale).toBeCloseTo(180 / (r.maxY - r.minY), 6)
    expect(vue.topLeftWorld.y).toBeCloseTo(0, 6)
    expect(vue.topLeftWorld.x).toBeGreaterThan(0)
  })
})

describe('grilleDuPlan — une lecture SIGNEE garde ses valeurs negatives', () => {
  const cellules: CelluleTactique[] = [
    { col: -1, lig: -7, valeur: -0.8, brut: -4, centre_x: -1, centre_y: -13, matchs: 4, matchs_victoire: 0, matchs_defaite: 4 },
    { col: 1, lig: -7, valeur: 0.8, brut: 4, centre_x: 3, centre_y: -13, matchs: 4, matchs_victoire: 4, matchs_defaite: 0 },
  ]
  const signee: EchelleTactique = { p50: 0.4, p95: 0.8, borne: 0.8, symetrique: true, n_cellules: 2 }

  it('peint les deux cotes autour du zero (0 et 1 aux extremites de la rampe)', () => {
    const r = repereDuPlan(FOND_ILLUSION, BORNES_MORTS, 2)!
    const grid = grilleDuPlan(cellules, r, signee)!
    expect(grid.signee).toBe(true)
    expect(tacticalIntensity(grid, -0.8)).toBe(0)
    expect(tacticalIntensity(grid, 0.8)).toBe(1)
    // Le zero n'est pas une mesure : autant gagne que perdu, la cellule reste vide.
    expect(tacticalIntensity(grid, 0)).toBeNull()
  })

  it('une lecture NON signee efface toujours les valeurs <= 0 (cellule jamais atteinte)', () => {
    const r = repereDuPlan(FOND_ILLUSION, BORNES_MORTS, 2)!
    const quantile: EchelleTactique = { p50: 0.4, p95: 0.8, borne: 0, symetrique: false, n_cellules: 2 }
    const grid = grilleDuPlan(cellules, r, quantile)!
    expect(tacticalIntensity(grid, -0.8)).toBeNull()
  })
})
