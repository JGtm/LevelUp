/**
 * zone.logic — la zone sélectionnée et sa mini-tuile « Rejeu » (plan Tactique v2, L6.1).
 *
 * Ce que ces tests cadenassent :
 *   - la zone présélectionnée est la plus chaude : |valeur| sur une lecture signée, puis le plus
 *     de matchs distincts ;
 *   - les coordonnées, la valeur signée, la sous-ligne par lecture ;
 *   - la tuile : le fait par face, l'arme (ou la catégorie traduite, ou rien), le badge de
 *     placement TRONQUÉ au mètre (« < 1 » en deçà), jamais sur un frag, la date au fuseau du
 *     joueur, le texte complet ;
 *   - l'étiquette du nom de zone, posée du côté où elle tient.
 */
import { describe, expect, it } from 'vitest'

import type { CelluleTactique, TacticalContribution } from '@/lib/api/types'

import { getTacticalText } from './i18n'
import {
  choixDuClic,
  coordonneesDeZone,
  modeleDeTuile,
  positionEtiquette,
  sousLigneDeZone,
  valeurAffichee,
  zoneLaPlusChaude,
} from './zone.logic'

const t = getTacticalText('fr')
const tEn = getTacticalText('en')

function cellule(col: number, valeur: number, matchs: number, extra: Partial<CelluleTactique> = {}): CelluleTactique {
  return { col, lig: 1, valeur, brut: valeur, matchs, matchs_victoire: 2, matchs_defaite: 2, centre_x: 0, centre_y: 0, ...extra }
}

function contribution(extra: Partial<TacticalContribution> = {}): TacticalContribution {
  return {
    match_id: 'm1',
    instant_ms: 83_400,
    clock: 'match',
    xuid: '2533274000000001',
    match_started_at: '2026-09-01T22:30:00Z',
    replay_available: true,
    face: 'mort',
    autre_gamertag: 'Rival',
    mode_label: 'Assassin',
    score_label: '50 - 42',
    ...extra,
  }
}

describe('zoneLaPlusChaude (D10)', () => {
  it('lecture simple : la plus grande valeur, puis le plus de matchs distincts', () => {
    expect(zoneLaPlusChaude([cellule(1, 2, 3), cellule(2, 5, 3), cellule(3, 5, 6)], false)?.col).toBe(3)
  })

  it('lecture signée : la plus grande valeur ABSOLUE, négative comprise', () => {
    expect(zoneLaPlusChaude([cellule(1, 0.4, 3), cellule(2, -0.9, 3)], true)?.col).toBe(2)
  })

  it('égalité de valeur : le plus de matchs distincts l’emporte même s’il vient d’abord, sinon la première', () => {
    expect(zoneLaPlusChaude([cellule(1, 5, 6), cellule(2, 5, 3)], false)?.col).toBe(1)
    expect(zoneLaPlusChaude([cellule(1, 5, 6), cellule(2, 5, 6)], false)?.col).toBe(1)
  })

  it('égalité de valeur absolue sur une lecture signée : le plus de matchs distincts', () => {
    expect(zoneLaPlusChaude([cellule(1, -0.9, 6), cellule(2, 0.9, 3)], true)?.col).toBe(1)
    expect(zoneLaPlusChaude([cellule(1, 0.9, 3), cellule(2, -0.9, 6)], true)?.col).toBe(2)
  })

  it('aucune cellule : aucune zone', () => {
    expect(zoneLaPlusChaude([], false)).toBeNull()
  })
})

describe('coordonneesDeZone et valeurAffichee', () => {
  it('x et y de la cellule, en mètres, signe moins typographique', () => {
    expect(coordonneesDeZone(t, -7, 2, 2, 'fr')).toBe('x −14…−12 m · y 4…6 m')
    expect(coordonneesDeZone(t, 3, 0, 0.5, 'fr')).toBe('x 1,5…2 m · y 0…0,5 m')
  })

  it('une lecture signée porte son signe, une lecture simple non', () => {
    expect(valeurAffichee(0.25, true, 'fr')).toBe('+ 0,25')
    expect(valeurAffichee(-0.25, true, 'fr')).toBe('− 0,25')
    expect(valeurAffichee(0.25, false, 'fr')).toBe('0,25')
  })
})

describe('sousLigneDeZone', () => {
  const c = cellule(1, 1, 4, { frags: 3, morts: 1 })
  it('les matchs distincts, et ce que la lecture ajoute', () => {
    expect(sousLigneDeZone(t, 'morts', c)).toBe('4 matchs distincts')
    expect(sousLigneDeZone(t, 'gagne', c)).toBe('2 victoires, 2 défaites · 4 matchs distincts')
    expect(sousLigneDeZone(t, 'solde', c)).toBe('3 frags, 1 mort · 4 matchs distincts')
  })
})

describe('modeleDeTuile — la mini-tuile d’une contribution', () => {
  it('une mort : tuée par l’autre joueur, l’arme dans la langue de la page', () => {
    const m = modeleDeTuile(t, contribution({ arme_label: 'Fusil de combat', arme_label_en: 'BR75' }), 'fr', 'UTC', 'Victoire')
    expect(m.fait).toBe('Tué par Rival')
    expect(m.arme).toBe('Fusil de combat')
    expect(modeleDeTuile(tEn, contribution({ arme_label: 'Fusil de combat', arme_label_en: 'BR75' }), 'en', 'UTC', 'Win').arme).toBe('BR75')
  })

  it('à défaut d’arme, la catégorie traduite ; une catégorie non traduite ne s’écrit pas', () => {
    expect(modeleDeTuile(t, contribution({ categorie_source: 'Headshot' }), 'fr', 'UTC', null).arme).toBe('tir à la tête')
    expect(modeleDeTuile(t, contribution({ categorie_source: 'Bullet' }), 'fr', 'UTC', null).arme).toBeUndefined()
  })

  it('un frag, une entrée, une réapparition : leur fait', () => {
    expect(modeleDeTuile(t, contribution({ face: 'frag', autre_gamertag: 'Cible' }), 'fr', 'UTC', null).fait).toBe('A tué Cible')
    expect(modeleDeTuile(t, contribution({ face: 'entree', autre_gamertag: undefined }), 'fr', 'UTC', null).fait).toBe('Entrée dans la zone')
    expect(modeleDeTuile(t, contribution({ face: 'reapparition', autre_gamertag: undefined }), 'fr', 'UTC', null).fait).toBe('Réapparition')
  })

  it('le badge de placement : seul sans distance, seul ou près TRONQUÉ au mètre, « < 1 » en deçà', () => {
    const badge = (placement: TacticalContribution['placement']) =>
      modeleDeTuile(t, contribution({ placement }), 'fr', 'UTC', null).badge
    expect(badge({ seul: true })).toBe('seul')
    expect(badge({ seul: true, distance_m: 30.7 })).toBe('seul · 30 m')
    expect(badge({ seul: false, distance_m: 17.9 })).toBe('près · 17 m')
    expect(badge({ seul: false, distance_m: 0.4 })).toBe('près · < 1 m')
  })

  it('jamais de badge sur un frag', () => {
    const m = modeleDeTuile(t, contribution({ face: 'frag', placement: { seul: true } }), 'fr', 'UTC', null)
    expect(m.badge).toBeUndefined()
  })

  it('la date et l’heure au fuseau du joueur', () => {
    expect(modeleDeTuile(t, contribution(), 'fr', 'Europe/Paris', null).date).toBe('02/09/2026 · 00:30')
    expect(modeleDeTuile(t, contribution(), 'fr', 'UTC', null).date).toBe('01/09/2026 · 22:30')
  })

  it('le texte complet (infobulle) et le lien de rejeu à l’instant', () => {
    const m = modeleDeTuile(
      t,
      contribution({ arme_label: 'Fusil de combat', placement: { seul: true, distance_m: 30.7 } }),
      'fr',
      'UTC',
      'Victoire',
    )
    expect(m.instant).toBe('1:23')
    expect(m.titre).toBe('Assassin · 50 - 42 · Victoire · 01/09/2026 · 22:30 · 1:23 · Tué par Rival · Fusil de combat · seul · 30 m')
    expect(m.rejeu).toEqual({ disponible: true, search: { t: '83400', clock: 'match' }, libelle: 'Ouvrir le rejeu à 1:23' })
  })
})

describe('positionEtiquette — le nom de zone à côté de la cellule', () => {
  const repere = { minX: 0, maxX: 100, minY: 0, maxY: 50, pasM: 10 }

  it('à gauche du plan : l’étiquette se pose à DROITE de la cellule', () => {
    expect(positionEtiquette({ col: 2, row: 1 }, repere)).toEqual({
      left: '30.00%',
      top: '70.00%',
      transform: 'translate(4px, -50%)',
    })
  })

  it('à droite du plan : à GAUCHE de la cellule', () => {
    expect(positionEtiquette({ col: 8, row: 1 }, repere)).toEqual({
      left: '80.00%',
      top: '70.00%',
      transform: 'translate(calc(-100% - 4px), -50%)',
    })
  })

  it('hors du cadre : aucune étiquette', () => {
    expect(positionEtiquette({ col: 40, row: 1 }, repere)).toBeNull()
  })

  it('plan grossi 2x sur la moitié gauche : l’étiquette suit la fenêtre', () => {
    const fenetre = { minX: 0, maxX: 50, minY: 12.5, maxY: 37.5 }
    // Cellule (2, 1) : x 20..30, y 10..20 → centre (25, 15), à la moitié de la fenêtre en largeur.
    expect(positionEtiquette({ col: 2, row: 1 }, repere, fenetre)).toEqual({
      left: '60.00%',
      top: '90.00%',
      transform: 'translate(4px, -50%)',
    })
  })

  it('cellule hors de la fenêtre visible : aucune étiquette', () => {
    const fenetre = { minX: 0, maxX: 50, minY: 25, maxY: 50 }
    expect(positionEtiquette({ col: 8, row: 1 }, repere, fenetre)).toBeNull()
  })
})

describe('choixDuClic — le clic retient une cellule SERVIE', () => {
  const servie = (col: number, lig: number): CelluleTactique => ({
    col,
    lig,
    valeur: 1,
    brut: 1,
    matchs: 3,
    matchs_victoire: 0,
    matchs_defaite: 0,
    centre_x: 0,
    centre_y: 0,
  })
  const cellules = [servie(2, 3), servie(4, 3)]

  it('dans une cellule servie : elle', () => {
    expect(choixDuClic(cellules, { x: 5, y: 7 }, 2)).toEqual({ col: 2, row: 3 })
  })

  it('sur le débord de la chaleur, à moins d’un pas d’un centre servi : la cellule la plus proche', () => {
    // (3,3) n'est pas servie : le clic va au centre servi le plus proche, (2,3) en (5,7) ou (4,3) en (9,7).
    expect(choixDuClic(cellules, { x: 6.9, y: 7 }, 2)).toEqual({ col: 2, row: 3 })
    expect(choixDuClic(cellules, { x: 7.1, y: 7 }, 2)).toEqual({ col: 4, row: 3 })
  })

  it('loin de toute cellule servie : rien', () => {
    expect(choixDuClic(cellules, { x: 5, y: 12 }, 2)).toBeNull()
    expect(choixDuClic(cellules, { x: 5, y: 7 }, 0)).toBeNull()
  })
})
