/**
 * Tests — rowLayout : la rangée de fiches tient. Les largeurs demandées sont écrites dans
 * les noms des tests : 2 × n × 115 + 270 + 2 × n × 6 + 2 × 24 (4 contre 4 = 1286, 5 contre 5 = 1528).
 */
import { describe, expect, it } from 'vitest'

import {
  colonnesGrandeEquipe,
  dispositionEquipes,
  emplacementScoreDe,
  grandeEquipe,
  largeurRequiseRangee,
  largeurRequiseRangeeGrandeEquipe,
  planFeedPosed,
  rowFits,
  rowFitsGrandeEquipe,
} from './rowLayout'
import { LARGEUR_REQUISE_ENTETE_SCORE_PX } from './cotesEnteteScore'

const DEUX_CAMPS = 2
const LARGEUR_1440 = 1440
const LARGEUR_1280 = 1280
const LARGEUR_1600 = 1600
const LARGEUR_4000 = 4000

describe('largeurRequiseRangee', () => {
  it('4 places par camp demandent 1286 px', () => {
    expect(largeurRequiseRangee(4)).toBe(1286)
  })

  it('5 places par camp demandent 1528 px', () => {
    expect(largeurRequiseRangee(5)).toBe(1528)
  })
})

describe('rowFits', () => {
  it('4 contre 4 tient à 1440 px (demande 1286 px)', () => {
    expect(rowFits({ camps: DEUX_CAMPS, placesParCamp: [4, 4], largeurCorps: LARGEUR_1440 })).toBe(true)
  })

  it('4 contre 4 ne tient PAS à 1280 px (demande 1286 px)', () => {
    expect(rowFits({ camps: DEUX_CAMPS, placesParCamp: [4, 4], largeurCorps: LARGEUR_1280 })).toBe(false)
  })

  it('5 contre 5 ne tient pas à 1440 px (demande 1528 px)', () => {
    expect(rowFits({ camps: DEUX_CAMPS, placesParCamp: [5, 5], largeurCorps: LARGEUR_1440 })).toBe(false)
  })

  it('5 contre 5 tient à 1600 px (demande 1528 px)', () => {
    expect(rowFits({ camps: DEUX_CAMPS, placesParCamp: [5, 5], largeurCorps: LARGEUR_1600 })).toBe(true)
  })

  it('3 contre 3 tient à 1280 px', () => {
    expect(rowFits({ camps: DEUX_CAMPS, placesParCamp: [3, 3], largeurCorps: LARGEUR_1280 })).toBe(true)
  })

  it('Grande équipe (12 par camp) ne tient jamais, même à 4000 px', () => {
    expect(rowFits({ camps: DEUX_CAMPS, placesParCamp: [12, 12], largeurCorps: LARGEUR_4000 })).toBe(false)
  })

  it('chacun pour soi (plus de deux camps) ne tient jamais, même à 4000 px', () => {
    expect(rowFits({ camps: 8, placesParCamp: [1, 1, 1, 1, 1, 1, 1, 1], largeurCorps: LARGEUR_4000 })).toBe(false)
  })

  it('chacun pour soi (un seul camp) ne tient jamais, même à 4000 px', () => {
    expect(rowFits({ camps: 1, placesParCamp: [8], largeurCorps: LARGEUR_4000 })).toBe(false)
  })

  it('cas limite : une largeur exactement égale à la demande (4 contre 4, 1286 px) tient', () => {
    expect(rowFits({ camps: DEUX_CAMPS, placesParCamp: [4, 4], largeurCorps: 1286 })).toBe(true)
  })

  it('cas limite : un pixel de moins que la demande (4 contre 4, 1285 px) ne tient pas', () => {
    expect(rowFits({ camps: DEUX_CAMPS, placesParCamp: [4, 4], largeurCorps: 1285 })).toBe(false)
  })

  it('camps de tailles inégales : la rangée compte le plus grand (4 et 2 demandent 1286 px)', () => {
    expect(rowFits({ camps: DEUX_CAMPS, placesParCamp: [4, 2], largeurCorps: 1286 })).toBe(true)
    expect(rowFits({ camps: DEUX_CAMPS, placesParCamp: [4, 2], largeurCorps: 1285 })).toBe(false)
  })

  it('une liste de places qui ne compte pas deux camps ne tient pas', () => {
    expect(rowFits({ camps: DEUX_CAMPS, placesParCamp: [3], largeurCorps: LARGEUR_4000 })).toBe(false)
  })
})

describe('grandeEquipe', () => {
  it('5 places par camp : forme ordinaire ; 6 places par camp : forme compacte', () => {
    expect(grandeEquipe([5, 5])).toBe(false)
    expect(grandeEquipe([6, 6])).toBe(true)
  })

  it('un seul camp plus grand suffit ; ni un camp seul ni plus de deux camps ne sont une Grande équipe', () => {
    expect(grandeEquipe([4, 12])).toBe(true)
    expect(grandeEquipe([12])).toBe(false)
    expect(grandeEquipe([12, 12, 12])).toBe(false)
  })

  it('le seuil est celui de la rangée : au-delà, elle ne se rend plus', () => {
    expect(rowFits({ camps: DEUX_CAMPS, placesParCamp: [6, 6], largeurCorps: LARGEUR_4000 })).toBe(false)
  })
})

describe('largeurRequiseRangeeGrandeEquipe', () => {
  it('6 emplacements par camp de 80 px demandent 1350 px : 2 × 6 × (80 + 6) + 270 + 2 × 24', () => {
    expect(largeurRequiseRangeeGrandeEquipe()).toBe(1350)
  })
})

describe('rowFitsGrandeEquipe', () => {
  it('12 contre 12 tient à 1440 px (demande 1350 px)', () => {
    expect(rowFitsGrandeEquipe({ camps: DEUX_CAMPS, placesParCamp: [12, 12], largeurCorps: LARGEUR_1440 })).toBe(true)
  })

  it('cas limite : 1350 px tient, 1349 px ne tient pas', () => {
    expect(rowFitsGrandeEquipe({ camps: DEUX_CAMPS, placesParCamp: [12, 12], largeurCorps: 1350 })).toBe(true)
    expect(rowFitsGrandeEquipe({ camps: DEUX_CAMPS, placesParCamp: [12, 12], largeurCorps: 1349 })).toBe(false)
  })

  it('la largeur demandée est celle du pire cas : 7 contre 7 demande autant que 12 contre 12', () => {
    expect(rowFitsGrandeEquipe({ camps: DEUX_CAMPS, placesParCamp: [7, 7], largeurCorps: 1349 })).toBe(false)
    expect(rowFitsGrandeEquipe({ camps: DEUX_CAMPS, placesParCamp: [7, 7], largeurCorps: 1350 })).toBe(true)
  })

  it('un camp de 13 places ne tient jamais, même à 4000 px', () => {
    expect(rowFitsGrandeEquipe({ camps: DEUX_CAMPS, placesParCamp: [13, 12], largeurCorps: LARGEUR_4000 })).toBe(false)
  })

  it('ce n’est pas une Grande équipe (5 par camp) : refusée, la règle ordinaire la juge', () => {
    expect(rowFitsGrandeEquipe({ camps: DEUX_CAMPS, placesParCamp: [5, 5], largeurCorps: LARGEUR_4000 })).toBe(false)
  })

  it('chacun pour soi et une liste de places qui ne compte pas deux camps sont refusés', () => {
    expect(rowFitsGrandeEquipe({ camps: 8, placesParCamp: [8, 8], largeurCorps: LARGEUR_4000 })).toBe(false)
    expect(rowFitsGrandeEquipe({ camps: 1, placesParCamp: [12], largeurCorps: LARGEUR_4000 })).toBe(false)
    expect(rowFitsGrandeEquipe({ camps: DEUX_CAMPS, placesParCamp: [12], largeurCorps: LARGEUR_4000 })).toBe(false)
  })
})

describe('colonnesGrandeEquipe', () => {
  it('un emplacement par deux places du plus grand camp, au moins un', () => {
    expect(colonnesGrandeEquipe([12, 12])).toBe(6)
    expect(colonnesGrandeEquipe([7, 7])).toBe(4)
    expect(colonnesGrandeEquipe([7, 3])).toBe(4)
    expect(colonnesGrandeEquipe([])).toBe(1)
  })
})

describe('dispositionEquipes', () => {
  it('4 contre 4 : rangée à 1440 px, colonne à 1280 px', () => {
    expect(dispositionEquipes({ camps: DEUX_CAMPS, placesParCamp: [4, 4], largeurCorps: LARGEUR_1440 })).toBe('rangee')
    expect(dispositionEquipes({ camps: DEUX_CAMPS, placesParCamp: [4, 4], largeurCorps: LARGEUR_1280 })).toBe('colonne')
  })

  it('12 contre 12 : rangée de Grande équipe à 1440 px, colonne à 1300 px', () => {
    expect(dispositionEquipes({ camps: DEUX_CAMPS, placesParCamp: [12, 12], largeurCorps: LARGEUR_1440 })).toBe('rangee-grande-equipe')
    expect(dispositionEquipes({ camps: DEUX_CAMPS, placesParCamp: [12, 12], largeurCorps: 1300 })).toBe('colonne')
  })

  it('13 places dans un camp, chacun pour soi, un seul camp, aucune largeur mesurée : la colonne', () => {
    expect(dispositionEquipes({ camps: DEUX_CAMPS, placesParCamp: [13, 13], largeurCorps: LARGEUR_4000 })).toBe('colonne')
    expect(dispositionEquipes({ camps: 8, placesParCamp: [1, 1, 1, 1, 1, 1, 1, 1], largeurCorps: LARGEUR_4000 })).toBe('colonne')
    expect(dispositionEquipes({ camps: 1, placesParCamp: [8], largeurCorps: LARGEUR_4000 })).toBe('colonne')
    expect(dispositionEquipes({ camps: DEUX_CAMPS, placesParCamp: [4, 4], largeurCorps: 0 })).toBe('colonne')
  })

  it('5 contre 5 trop étroit pour la rangée ordinaire : la colonne, jamais la forme Grande équipe', () => {
    expect(dispositionEquipes({ camps: DEUX_CAMPS, placesParCamp: [5, 5], largeurCorps: LARGEUR_1440 })).toBe('colonne')
  })
})

describe('emplacementScoreDe', () => {
  it('le Standard porte le score dans son en-tête tant que le corps loge la cellule de 440 px', () => {
    for (const largeur of [0, LARGEUR_REQUISE_ENTETE_SCORE_PX, LARGEUR_1440, 4000]) {
      expect(emplacementScoreDe('standard', 'colonne', largeur)).toBe('entete')
    }
  })

  it('le Standard, un pixel sous la place de la cellule du score : le score coiffe la toile, comme au Cinéma', () => {
    expect(emplacementScoreDe('standard', 'colonne', LARGEUR_REQUISE_ENTETE_SCORE_PX - 1)).toBe('scene')
    expect(emplacementScoreDe('standard', 'colonne', 1)).toBe('scene')
  })

  it('une seule règle de largeur pour les trois cadres, aux deux côtés du seuil', () => {
    for (const frame of ['standard', 'cinema', 'plein-cadre'] as const) {
      expect(emplacementScoreDe(frame, 'colonne', LARGEUR_REQUISE_ENTETE_SCORE_PX - 1), frame).toBe('scene')
      expect(emplacementScoreDe(frame, 'colonne', LARGEUR_REQUISE_ENTETE_SCORE_PX), frame).toBe('entete')
      expect(emplacementScoreDe(frame, 'colonne', 0), frame).toBe('entete')
    }
  })

  it('Cinéma et Plein cadre, rangée posée : le score est au centre de la rangée', () => {
    for (const frame of ['cinema', 'plein-cadre'] as const) {
      expect(emplacementScoreDe(frame, 'rangee', LARGEUR_1440)).toBe('rangee')
      expect(emplacementScoreDe(frame, 'rangee-grande-equipe', LARGEUR_1440)).toBe('rangee')
    }
  })

  it('Cinéma et Plein cadre, repli : le score est dans l’en-tête, et plus sur la toile', () => {
    for (const frame of ['cinema', 'plein-cadre'] as const) {
      expect(emplacementScoreDe(frame, 'colonne', LARGEUR_1440)).toBe('entete')
      expect(emplacementScoreDe(frame, 'colonne', LARGEUR_REQUISE_ENTETE_SCORE_PX)).toBe('entete')
    }
  })

  it('repli : un pixel de moins que la place de la cellule du score dans l’en-tête, le score reste sur la toile', () => {
    for (const frame of ['cinema', 'plein-cadre'] as const) {
      expect(emplacementScoreDe(frame, 'colonne', LARGEUR_REQUISE_ENTETE_SCORE_PX - 1)).toBe('scene')
    }
  })

  it('repli, largeur pas encore mesurée (0) : l’en-tête', () => {
    expect(emplacementScoreDe('cinema', 'colonne', 0)).toBe('entete')
  })

  it('la place demandée est la somme de ses cotes : 440 px de cellule, deux écarts, deux cellules latérales, deux marges', () => {
    expect(LARGEUR_REQUISE_ENTETE_SCORE_PX).toBe(440 + 2 * 16 + 2 * 150 + 2 * 24)
  })
})

describe('planFeedPosed', () => {
  it('le Standard ne pose jamais le fil sur le plan : sa colonne le porte', () => {
    for (const d of ['rangee', 'rangee-grande-equipe', 'colonne'] as const) expect(planFeedPosed('standard', d)).toBe(false)
  })

  it("Cinéma et Plein cadre : posé quand la rangée de fiches l'est (ordinaire ou Grande équipe), absent dans le repli", () => {
    for (const frame of ['cinema', 'plein-cadre'] as const) {
      expect(planFeedPosed(frame, 'rangee')).toBe(true)
      expect(planFeedPosed(frame, 'rangee-grande-equipe')).toBe(true)
      expect(planFeedPosed(frame, 'colonne')).toBe(false)
    }
  })
})
