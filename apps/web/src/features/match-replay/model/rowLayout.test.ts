/**
 * Tests — rowLayout : la rangée de fiches tient. Les largeurs demandées sont écrites dans
 * les noms des tests : 2 × n × 115 + 270 + 2 × n × 6 + 2 × 24 (4 contre 4 = 1286, 5 contre 5 = 1528) au Cinéma, dont la
 * marge de corps est de 24 px ; au Plein cadre elle est de 4 px (`cotesCorps.ts`), donc 40 px de moins.
 */
import { describe, expect, it } from 'vitest'

import {
  colonnesGrandeEquipe,
  dispositionAvecMemoire,
  dispositionEquipes,
  emplacementScoreDe,
  grandeEquipe,
  largeurRequiseRangee,
  largeurRequiseRangeeGrandeEquipe,
  MARGE_BASCULE_PX,
  planFeedPosed,
  type DispositionEquipes,
  rowFits,
  rowFitsGrandeEquipe,
} from './rowLayout'
import { largeurRequiseEnteteScore } from './cotesEnteteScore'

const DEUX_CAMPS = 2
const LARGEUR_1440 = 1440
const LARGEUR_1280 = 1280
const LARGEUR_1600 = 1600
const LARGEUR_4000 = 4000

describe('largeurRequiseRangee', () => {
  it('4 places par camp demandent 1286 px', () => {
    expect(largeurRequiseRangee(4, 'cinema')).toBe(1286)
  })

  it('5 places par camp demandent 1528 px', () => {
    expect(largeurRequiseRangee(5, 'cinema')).toBe(1528)
  })
})

describe('rowFits', () => {
  it('4 contre 4 tient à 1440 px (demande 1286 px)', () => {
    expect(rowFits({ frame: 'cinema', camps: DEUX_CAMPS, placesParCamp: [4, 4], largeurCorps: LARGEUR_1440 })).toBe(true)
  })

  it('4 contre 4 ne tient PAS à 1280 px (demande 1286 px)', () => {
    expect(rowFits({ frame: 'cinema', camps: DEUX_CAMPS, placesParCamp: [4, 4], largeurCorps: LARGEUR_1280 })).toBe(false)
  })

  it('5 contre 5 ne tient pas à 1440 px (demande 1528 px)', () => {
    expect(rowFits({ frame: 'cinema', camps: DEUX_CAMPS, placesParCamp: [5, 5], largeurCorps: LARGEUR_1440 })).toBe(false)
  })

  it('5 contre 5 tient à 1600 px (demande 1528 px)', () => {
    expect(rowFits({ frame: 'cinema', camps: DEUX_CAMPS, placesParCamp: [5, 5], largeurCorps: LARGEUR_1600 })).toBe(true)
  })

  it('3 contre 3 tient à 1280 px', () => {
    expect(rowFits({ frame: 'cinema', camps: DEUX_CAMPS, placesParCamp: [3, 3], largeurCorps: LARGEUR_1280 })).toBe(true)
  })

  it('Grande équipe (12 par camp) ne tient jamais, même à 4000 px', () => {
    expect(rowFits({ frame: 'cinema', camps: DEUX_CAMPS, placesParCamp: [12, 12], largeurCorps: LARGEUR_4000 })).toBe(false)
  })

  it('chacun pour soi (plus de deux camps) ne tient jamais, même à 4000 px', () => {
    expect(rowFits({ frame: 'cinema', camps: 8, placesParCamp: [1, 1, 1, 1, 1, 1, 1, 1], largeurCorps: LARGEUR_4000 })).toBe(false)
  })

  it('chacun pour soi (un seul camp) ne tient jamais, même à 4000 px', () => {
    expect(rowFits({ frame: 'cinema', camps: 1, placesParCamp: [8], largeurCorps: LARGEUR_4000 })).toBe(false)
  })

  it('cas limite : une largeur exactement égale à la demande (4 contre 4, 1286 px) tient', () => {
    expect(rowFits({ frame: 'cinema', camps: DEUX_CAMPS, placesParCamp: [4, 4], largeurCorps: 1286 })).toBe(true)
  })

  it('cas limite : un pixel de moins que la demande (4 contre 4, 1285 px) ne tient pas', () => {
    expect(rowFits({ frame: 'cinema', camps: DEUX_CAMPS, placesParCamp: [4, 4], largeurCorps: 1285 })).toBe(false)
  })

  it('camps de tailles inégales : la rangée compte le plus grand (4 et 2 demandent 1286 px)', () => {
    expect(rowFits({ frame: 'cinema', camps: DEUX_CAMPS, placesParCamp: [4, 2], largeurCorps: 1286 })).toBe(true)
    expect(rowFits({ frame: 'cinema', camps: DEUX_CAMPS, placesParCamp: [4, 2], largeurCorps: 1285 })).toBe(false)
  })

  it('une liste de places qui ne compte pas deux camps ne tient pas', () => {
    expect(rowFits({ frame: 'cinema', camps: DEUX_CAMPS, placesParCamp: [3], largeurCorps: LARGEUR_4000 })).toBe(false)
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
    expect(rowFits({ frame: 'cinema', camps: DEUX_CAMPS, placesParCamp: [6, 6], largeurCorps: LARGEUR_4000 })).toBe(false)
  })
})

describe('la marge de corps est une cote par cadre', () => {
  it('le Plein cadre demande 40 px de moins que le Cinéma : 4 px de marge propre de chaque côté au lieu de 24', () => {
    expect(largeurRequiseRangee(4, 'plein-cadre')).toBe(1286 - 40)
    expect(largeurRequiseRangee(5, 'plein-cadre')).toBe(1528 - 40)
    expect(largeurRequiseRangeeGrandeEquipe('plein-cadre')).toBe(1350 - 40)
  })

  it('la rangée tient au Plein cadre à la largeur que ce cadre demande, et pas un pixel plus bas', () => {
    const entree = (largeurCorps: number) => ({ frame: 'plein-cadre' as const, camps: DEUX_CAMPS, placesParCamp: [4, 4], largeurCorps })
    expect(rowFits(entree(1246))).toBe(true)
    expect(rowFits(entree(1245))).toBe(false)
    // Au Cinéma la même largeur ne suffit pas encore.
    expect(rowFits({ ...entree(1246), frame: 'cinema' })).toBe(false)
  })

  it('le Standard compte la marge de sa page (24 px), comme avant', () => {
    expect(largeurRequiseRangee(4, 'standard')).toBe(1286)
  })
})

describe('largeurRequiseRangeeGrandeEquipe', () => {
  it('6 emplacements par camp de 80 px demandent 1350 px : 2 × 6 × (80 + 6) + 270 + 2 × 24', () => {
    expect(largeurRequiseRangeeGrandeEquipe('cinema')).toBe(1350)
  })
})

describe('rowFitsGrandeEquipe', () => {
  it('12 contre 12 tient à 1440 px (demande 1350 px)', () => {
    expect(rowFitsGrandeEquipe({ frame: 'cinema', camps: DEUX_CAMPS, placesParCamp: [12, 12], largeurCorps: LARGEUR_1440 })).toBe(true)
  })

  it('cas limite : 1350 px tient, 1349 px ne tient pas', () => {
    expect(rowFitsGrandeEquipe({ frame: 'cinema', camps: DEUX_CAMPS, placesParCamp: [12, 12], largeurCorps: 1350 })).toBe(true)
    expect(rowFitsGrandeEquipe({ frame: 'cinema', camps: DEUX_CAMPS, placesParCamp: [12, 12], largeurCorps: 1349 })).toBe(false)
  })

  it('la largeur demandée est celle du pire cas : 7 contre 7 demande autant que 12 contre 12', () => {
    expect(rowFitsGrandeEquipe({ frame: 'cinema', camps: DEUX_CAMPS, placesParCamp: [7, 7], largeurCorps: 1349 })).toBe(false)
    expect(rowFitsGrandeEquipe({ frame: 'cinema', camps: DEUX_CAMPS, placesParCamp: [7, 7], largeurCorps: 1350 })).toBe(true)
  })

  it('un camp de 13 places ne tient jamais, même à 4000 px', () => {
    expect(rowFitsGrandeEquipe({ frame: 'cinema', camps: DEUX_CAMPS, placesParCamp: [13, 12], largeurCorps: LARGEUR_4000 })).toBe(false)
  })

  it('ce n’est pas une Grande équipe (5 par camp) : refusée, la règle ordinaire la juge', () => {
    expect(rowFitsGrandeEquipe({ frame: 'cinema', camps: DEUX_CAMPS, placesParCamp: [5, 5], largeurCorps: LARGEUR_4000 })).toBe(false)
  })

  it('chacun pour soi et une liste de places qui ne compte pas deux camps sont refusés', () => {
    expect(rowFitsGrandeEquipe({ frame: 'cinema', camps: 8, placesParCamp: [8, 8], largeurCorps: LARGEUR_4000 })).toBe(false)
    expect(rowFitsGrandeEquipe({ frame: 'cinema', camps: 1, placesParCamp: [12], largeurCorps: LARGEUR_4000 })).toBe(false)
    expect(rowFitsGrandeEquipe({ frame: 'cinema', camps: DEUX_CAMPS, placesParCamp: [12], largeurCorps: LARGEUR_4000 })).toBe(false)
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
    expect(dispositionEquipes({ frame: 'cinema', camps: DEUX_CAMPS, placesParCamp: [4, 4], largeurCorps: LARGEUR_1440 })).toBe('rangee')
    expect(dispositionEquipes({ frame: 'cinema', camps: DEUX_CAMPS, placesParCamp: [4, 4], largeurCorps: LARGEUR_1280 })).toBe('colonne')
  })

  it('12 contre 12 : rangée de Grande équipe à 1440 px, colonne à 1300 px', () => {
    expect(dispositionEquipes({ frame: 'cinema', camps: DEUX_CAMPS, placesParCamp: [12, 12], largeurCorps: LARGEUR_1440 })).toBe('rangee-grande-equipe')
    expect(dispositionEquipes({ frame: 'cinema', camps: DEUX_CAMPS, placesParCamp: [12, 12], largeurCorps: 1300 })).toBe('colonne')
  })

  it('13 places dans un camp, chacun pour soi, un seul camp, aucune largeur mesurée : la colonne', () => {
    expect(dispositionEquipes({ frame: 'cinema', camps: DEUX_CAMPS, placesParCamp: [13, 13], largeurCorps: LARGEUR_4000 })).toBe('colonne')
    expect(dispositionEquipes({ frame: 'cinema', camps: 8, placesParCamp: [1, 1, 1, 1, 1, 1, 1, 1], largeurCorps: LARGEUR_4000 })).toBe('colonne')
    expect(dispositionEquipes({ frame: 'cinema', camps: 1, placesParCamp: [8], largeurCorps: LARGEUR_4000 })).toBe('colonne')
    expect(dispositionEquipes({ frame: 'cinema', camps: DEUX_CAMPS, placesParCamp: [4, 4], largeurCorps: 0 })).toBe('colonne')
  })

  it('5 contre 5 trop étroit pour la rangée ordinaire : la colonne, jamais la forme Grande équipe', () => {
    expect(dispositionEquipes({ frame: 'cinema', camps: DEUX_CAMPS, placesParCamp: [5, 5], largeurCorps: LARGEUR_1440 })).toBe('colonne')
  })
})

describe('emplacementScoreDe', () => {
  it('le Standard porte le score dans son en-tête tant que le corps loge la cellule de 440 px', () => {
    for (const largeur of [0, largeurRequiseEnteteScore('standard'), LARGEUR_1440, 4000]) {
      expect(emplacementScoreDe('standard', 'colonne', largeur)).toBe('entete')
    }
  })

  it('le Standard, un pixel sous la place de la cellule du score : le score coiffe la toile, comme au Cinéma', () => {
    expect(emplacementScoreDe('standard', 'colonne', largeurRequiseEnteteScore('standard') - 1)).toBe('scene')
    expect(emplacementScoreDe('standard', 'colonne', 1)).toBe('scene')
  })

  it('la même règle pour les trois cadres, au seuil que dit la marge de corps de chacun', () => {
    for (const frame of ['standard', 'cinema', 'plein-cadre'] as const) {
      expect(emplacementScoreDe(frame, 'colonne', largeurRequiseEnteteScore(frame) - 1), frame).toBe('scene')
      expect(emplacementScoreDe(frame, 'colonne', largeurRequiseEnteteScore(frame)), frame).toBe('entete')
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
      expect(emplacementScoreDe(frame, 'colonne', largeurRequiseEnteteScore(frame))).toBe('entete')
    }
  })

  it('repli : un pixel de moins que la place de la cellule du score dans l’en-tête, le score reste sur la toile', () => {
    for (const frame of ['cinema', 'plein-cadre'] as const) {
      expect(emplacementScoreDe(frame, 'colonne', largeurRequiseEnteteScore(frame) - 1)).toBe('scene')
    }
  })

  it('repli, largeur pas encore mesurée (0) : l’en-tête', () => {
    expect(emplacementScoreDe('cinema', 'colonne', 0)).toBe('entete')
  })

  it('la place demandée est la somme de ses cotes : 440 px de cellule, deux écarts, deux cellules latérales, deux marges', () => {
    expect(largeurRequiseEnteteScore('cinema')).toBe(440 + 2 * 16 + 2 * 150 + 2 * 24)
  })

  it('au Plein cadre, la marge de corps de 4 px rend la place demandée plus petite de 40 px', () => {
    expect(largeurRequiseEnteteScore('plein-cadre')).toBe(440 + 2 * 16 + 2 * 150 + 2 * 4)
    expect(largeurRequiseEnteteScore('cinema') - largeurRequiseEnteteScore('plein-cadre')).toBe(40)
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

describe('dispositionAvecMemoire : la marge de bascule', () => {
  const entree = (largeurCorps: number, placesParCamp: readonly number[] = [4, 4]) => ({ frame: 'cinema' as const, camps: DEUX_CAMPS, placesParCamp, largeurCorps })

  it('la marge est de 24 px', () => {
    expect(MARGE_BASCULE_PX).toBe(24)
  })

  it('sans précédente, ou après la colonne : le choix sans mémoire, on entre dans la rangée à la largeur demandée', () => {
    for (const precedente of [null, 'colonne'] as const) {
      expect(dispositionAvecMemoire(entree(1285), precedente), String(precedente)).toBe('colonne')
      expect(dispositionAvecMemoire(entree(1286), precedente), String(precedente)).toBe('rangee')
    }
  })

  it('après la rangée : on y reste jusqu’à la largeur demandée moins 24 px (1262 px), on en sort un pixel plus bas', () => {
    expect(dispositionAvecMemoire(entree(1286), 'rangee')).toBe('rangee')
    expect(dispositionAvecMemoire(entree(1285), 'rangee')).toBe('rangee')
    expect(dispositionAvecMemoire(entree(1262), 'rangee')).toBe('rangee')
    expect(dispositionAvecMemoire(entree(1261), 'rangee')).toBe('colonne')
  })

  it('la rangée de Grande équipe a la même marge (1350 px demandés : on y reste jusqu’à 1326 px)', () => {
    expect(dispositionAvecMemoire(entree(1326, [12, 12]), 'rangee-grande-equipe')).toBe('rangee-grande-equipe')
    expect(dispositionAvecMemoire(entree(1325, [12, 12]), 'rangee-grande-equipe')).toBe('colonne')
    expect(dispositionAvecMemoire(entree(1349, [12, 12]), 'colonne')).toBe('colonne')
  })

  it('une largeur pas mesurée (0) est toujours la colonne, rangée précédente ou non', () => {
    expect(dispositionAvecMemoire(entree(0), 'rangee')).toBe('colonne')
  })

  it('la rangée précédente ne se prolonge pas à un match d’une autre forme (la marge ne sert pas de passe-droit)', () => {
    // Précédente ordinaire, match devenu Grande équipe : la marge n'ouvre pas la rangée de Grande équipe.
    expect(dispositionAvecMemoire(entree(1340, [12, 12]), 'rangee')).toBe('colonne')
    // Précédente Grande équipe, match devenu ordinaire (4 contre 4, 1270 px, sous la demande de 1286 px).
    expect(dispositionAvecMemoire(entree(1270), 'rangee-grande-equipe')).toBe('colonne')
  })

  it('séquence 1290, 1275, 1290 (4 contre 4) : la rangée ne clignote pas', () => {
    let d: DispositionEquipes | null = null
    const suite: DispositionEquipes[] = []
    for (const largeur of [1290, 1275, 1290]) {
      d = dispositionAvecMemoire(entree(largeur), d)
      suite.push(d)
    }
    expect(suite).toEqual(['rangee', 'rangee', 'rangee'])
  })

  it('séquence 1280, 1290, 1275, 1260, 1275, 1286 : on entre à 1290, on sort sous 1262, on ne ré-entre qu’à 1286', () => {
    let d: DispositionEquipes | null = null
    const suite: DispositionEquipes[] = []
    for (const largeur of [1280, 1290, 1275, 1260, 1275, 1286]) {
      d = dispositionAvecMemoire(entree(largeur), d)
      suite.push(d)
    }
    expect(suite).toEqual(['colonne', 'rangee', 'rangee', 'colonne', 'colonne', 'rangee'])
  })
})

describe('emplacementScoreDe : la marge de bascule', () => {
  const REQUIS = largeurRequiseEnteteScore('cinema')

  it('sans précédent, ou depuis la toile ou la rangée : le seuil nu (on monte dans l’en-tête à 820 px)', () => {
    for (const precedent of [null, 'scene', 'rangee'] as const) {
      expect(emplacementScoreDe('cinema', 'colonne', REQUIS - 1, precedent), String(precedent)).toBe('scene')
      expect(emplacementScoreDe('cinema', 'colonne', REQUIS, precedent), String(precedent)).toBe('entete')
    }
  })

  it('depuis l’en-tête : on y reste jusqu’à 820 moins 24 px (796 px), la toile un pixel plus bas', () => {
    for (const frame of ['standard', 'cinema', 'plein-cadre'] as const) {
      const requis = largeurRequiseEnteteScore(frame)
      expect(emplacementScoreDe(frame, 'colonne', requis - 1, 'entete'), frame).toBe('entete')
      expect(emplacementScoreDe(frame, 'colonne', requis - 24, 'entete'), frame).toBe('entete')
      expect(emplacementScoreDe(frame, 'colonne', requis - 25, 'entete'), frame).toBe('scene')
    }
  })

  it('une largeur pas mesurée reste dans l’en-tête, et la rangée posée garde la main sur la marge', () => {
    expect(emplacementScoreDe('cinema', 'colonne', 0, 'scene')).toBe('entete')
    expect(emplacementScoreDe('cinema', 'rangee', 100, 'entete')).toBe('rangee')
  })
})
