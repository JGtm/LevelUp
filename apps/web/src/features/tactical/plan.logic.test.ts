/**
 * plan.logic — l'aide ⓘ, la légende verticale, la boîte et l'état de la carte du plan (plan
 * Tactique v2, L5.2).
 *
 * Ce que ces tests cadenassent :
 *   - l'ⓘ dit les dénominateurs, la source, le PAS de la grille et le plancher, plus ce que la
 *     lecture ajoute (« victoires − défaites », « morts seul ») et les zones hors cadre ;
 *   - la rampe de légende est VERTICALE, construite depuis la rampe PEINTE, le bas de la rampe
 *     (valeur basse, ou négative) en bas ;
 *   - la boîte du plan garde le rapport du fond et ne dépasse pas la hauteur que la fenêtre lui laisse ;
 *   - l'image du fond suit la fenêtre du zoom ;
 *   - l'échec prime sur tout ; une carte hors du filtre ou absente se dit, jamais une attente.
 */
import { describe, expect, it } from 'vitest'

import type { TacticalRaster } from '@/lib/api/types'
import { mapFrame } from '@/lib/replay/heatPaint'

import { getTacticalText } from './i18n'
import { boiteDuPlan, cadrageDuFond, etatDuPlan, infoDuPlan, legendeDuPlan, rampeVerticale } from './plan.logic'
import { aspectDuPlan } from './tacticalLecture.logic'
import { repereDuPlan } from './tacticalView.logic'

const t = getTacticalText('fr')
const tEn = getTacticalText('en')

const LECTURE: TacticalRaster = {
  map_id: 'streets',
  question: 'morts',
  qui: 'moi',
  bornes: { min_x: 0, max_x: 100, min_y: 0, max_y: 50, valide: true },
  pas_m: 10,
  echelle: { p50: 1, p95: 5, borne: 5, n_cellules: 2, symetrique: false },
  cellules: [
    { col: 2, lig: 1, valeur: 3, brut: 3, matchs: 4, matchs_victoire: 2, matchs_defaite: 2, centre_x: 25, centre_y: 15 },
    { col: 5, lig: 2, valeur: 5, brut: 5, matchs: 6, matchs_victoire: 3, matchs_defaite: 3, centre_x: 55, centre_y: 25 },
  ],
  matchs_filtres: 50,
  matchs_retenus: 45,
  matchs_victoire: 20,
  matchs_defaite: 25,
  points_ignores: 0,
}

describe('infoDuPlan — l’aide ⓘ du titre', () => {
  it('les matchs lus, la source, le pas de la grille et le plancher par zone', () => {
    expect(infoDuPlan(t, 'fr', 'morts', LECTURE, 2)).toBe(
      '45 matchs · journal des morts · grille 10 m · 3 matchs distincts par zone',
    )
  })

  it('un pas fractionnaire dans la langue de la page ; en anglais, les mots anglais', () => {
    expect(infoDuPlan(t, 'fr', 'morts', { ...LECTURE, pas_m: 0.5 }, 2)).toContain('grille 0,5 m')
    expect(infoDuPlan(tEn, 'en', 'morts', LECTURE, 2)).toBe(
      '45 matches · death log · 10 m grid · 3 distinct matches per zone',
    )
  })

  it('une lecture d’artefact cite les artefacts de rejeu', () => {
    expect(infoDuPlan(t, 'fr', 'temps', LECTURE, 2)).toContain('artefacts de rejeu')
  })

  it('« victoires − défaites » : les deux dénominateurs, de chaque côté', () => {
    expect(infoDuPlan(t, 'fr', 'gagne', LECTURE, 2)).toContain(
      ' · 20 victoires et 25 défaites, 3 matchs distincts de chaque côté',
    )
  })

  it('« morts seul » : la règle avec la ou les portées, puis ce qu’elle écarte par construction', () => {
    const info = infoDuPlan(t, 'fr', 'isole', { ...LECTURE, rayons_radar_m: [18, 24], morts_equipe_a_terre: 3 }, 2)
    expect(info).toContain(
      ' · seul : aucun coéquipier visible, ou le plus proche au-delà de 18 m ou 24 m (portée du radar)',
    )
    expect(info).not.toContain('portée de radar connue')
    expect(info).toContain(" · 3 morts écartées, aucun coéquipier en mesure d'accompagner")
  })

  it('des zones servies hors du cadre du fond : dites, jamais avalées', () => {
    expect(infoDuPlan(t, 'fr', 'morts', LECTURE, 1)).toContain(' · 1 zone hors du cadre du fond sur 2')
    expect(infoDuPlan(t, 'fr', 'morts', LECTURE, 2)).not.toContain('hors du cadre')
  })
})

describe('legendeDuPlan — les bornes de la rampe', () => {
  const format = (n: number) => String(n).replace('.', ',')

  it('lecture simple : de 0 au p95, l’unité à part', () => {
    expect(legendeDuPlan(LECTURE.echelle, 'morts par match', format)).toEqual({
      lo: '0',
      hiNombre: '5',
      unite: 'morts par match',
      hi: '5 morts par match',
      mode: 'intensity',
    })
  })

  it('lecture signée : de − borne à + borne, rampe divergente', () => {
    const l = legendeDuPlan({ p50: 0, p95: 0, borne: -2.5, n_cellules: 3, symetrique: true }, 'frags − morts par match', format)
    expect(l.lo).toBe('− 2,5')
    expect(l.hiNombre).toBe('+ 2,5')
    expect(l.mode).toBe('divergent')
  })
})

describe('rampeVerticale — la rampe de légende, depuis la rampe peinte', () => {
  const RAMPE = Array.from({ length: 64 }, (_, i) => `rgba(${i},0,0,0.5)`)

  it('verticale, du bas (valeur basse) vers le haut (valeur haute)', () => {
    const css = rampeVerticale(RAMPE)
    expect(css.startsWith('linear-gradient(0deg, ')).toBe(true)
    expect(css).toContain('rgba(0,0,0,0.5) 0%')
    expect(css.endsWith('rgba(63,0,0,0.5) 100%)')).toBe(true)
  })

  it('rampe vide : aucun dégradé', () => {
    expect(rampeVerticale([])).toBe('none')
  })
})

describe('boiteDuPlan — le cadre du fond', () => {
  it('au rapport du fond, jamais plus haut que la hauteur que la fenêtre lui laisse', () => {
    expect(boiteDuPlan(2, 800)).toEqual({ aspectRatio: 2, width: '100%', maxWidth: '1600px' })
    expect(boiteDuPlan(0.5, 560)).toEqual({ aspectRatio: 0.5, width: '100%', maxWidth: '280px' })
  })

  // Constat du 2026-09-09 : sur Illusion, un canvas de 1 070 x 13 375 px (rapport 0,08) tiré des
  // bornes servies. Le repère d'une carte à fond est son CALAGE, et le canvas remplit la boîte :
  // avec le calage publié d'Illusion (`ctf_illusion.json`), le cadre garde le rapport du fond et
  // tient dans la hauteur laissée, quelles que soient les bornes.
  it('Illusion : le cadre suit le calage du fond, jamais les bornes servies', () => {
    const fond = mapFrame({ metersPerPixel: 0.031, originX: -25.751434532165526, originY: 35.164999237060556, widthPx: 1711, heightPx: 2224 })
    const bornesEtirees = { min_x: -25, max_x: 8.17, min_y: -380, max_y: 35, valide: true }
    const repere = repereDuPlan(fond, bornesEtirees, 2)
    expect(repere).not.toBeNull()
    const aspect = aspectDuPlan(fond, repere)
    expect(aspect).toBeCloseTo(1711 / 2224, 6)
    const hauteurMax = 800
    const boite = boiteDuPlan(aspect, hauteurMax)
    // La boîte prend toute la largeur disponible (1 070 px au constat) sous `maxWidth`, au rapport du fond.
    const largeur = Math.min(1070, Number.parseInt(boite.maxWidth, 10))
    expect(largeur / aspect).toBeLessThanOrEqual(hauteurMax + 1)
  })
})

describe('cadrageDuFond — l’image du fond suit la fenêtre du zoom', () => {
  const scene = { minX: 0, maxX: 100, minY: 0, maxY: 50 }

  it('à 1x (fenêtre = scène) : le cadre entier', () => {
    expect(cadrageDuFond(scene, scene)).toEqual({ left: '0%', top: '0%', width: '100%', height: '100%' })
  })

  it('à 2x sur le quart haut-droit : image doublée, décalée de la moitié vers la gauche, collée en haut', () => {
    expect(cadrageDuFond(scene, { minX: 50, maxX: 100, minY: 25, maxY: 50 })).toEqual({
      left: '-100%',
      top: '0%',
      width: '200%',
      height: '200%',
    })
  })

  it('à 2x sur le quart bas-gauche : décalée d’une hauteur de fenêtre vers le haut', () => {
    expect(cadrageDuFond(scene, { minX: 0, maxX: 50, minY: 0, maxY: 25 })).toEqual({
      left: '0%',
      top: '-100%',
      width: '200%',
      height: '200%',
    })
  })
})

describe('etatDuPlan — ce qui se pose sur le fond', () => {
  it('l’échec (lecture, périmètre, composition) prime sur tout', () => {
    expect(etatDuPlan('hors_filtre', 'echec')).toBe('echec')
    expect(etatDuPlan('attente', 'echec')).toBe('echec')
  })

  it('carte hors du filtre : le dire ; aucune carte ouvrable : rien', () => {
    expect(etatDuPlan('hors_filtre', 'attente')).toBe('hors_filtre')
    expect(etatDuPlan('aucune', 'attente')).toBe('sans_carte')
  })

  it('sinon l’état de la lecture', () => {
    expect(etatDuPlan('attente', 'attente')).toBe('attente')
    expect(etatDuPlan('url', 'relecture')).toBe('relecture')
    expect(etatDuPlan('defaut', 'pret')).toBe('pret')
  })
})
