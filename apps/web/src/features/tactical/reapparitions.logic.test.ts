import { describe, expect, it } from 'vitest'

import type { ReplayCalloutZone as CalloutZone, TacticalReapparitionsAdverses, TacticalRepartitionZones } from '@/lib/api/types'
import { normalizeCalloutZones } from '@/lib/replay/calloutsPaint'

import {
  cadreDeLaCouche,
  cheminDeLaZone,
  construireLaLecture,
  ecartEnPoints,
  formaterLaPart,
  jaugeContrePlancher,
  lignesEtiquetees,
  opaciteDeLaTeinte,
  positionDeLEtiquette,
  puceDEcart,
  volumeDeSonde,
  zonesProposees,
} from './reapparitions.logic'
import { repereDuPlan } from './tacticalView.logic'

function repartition(
  zones: Array<[number, string, number, number]>,
  extra: Partial<TacticalRepartitionZones> = {},
): TacticalRepartitionZones {
  const total = zones.reduce((n, z) => n + z[2], 0)
  return {
    matchs: 6,
    reapparitions: total,
    sans_zone: 0,
    suffisante: true,
    zones: zones.map(([volume_index, nom, reapparitions, part]) => ({ volume_index, nom, reapparitions, part })),
    ...extra,
  }
}

function reponse(over: Partial<TacticalReapparitionsAdverses> = {}): TacticalReapparitionsAdverses {
  return {
    etat: 'servie',
    map_id: 'carte',
    matchs_filtres: 10,
    matchs_retenus: 9,
    matchs_sans_equipe: 0,
    plancher: { matchs: 3, reapparitions: 20, joueurs_pour_tenir: 2 },
    reference: repartition([
      [1, 'Hydro', 40, 0.4],
      [2, 'Longère', 30, 0.3],
      [3, 'Fosse', 30, 0.3],
    ]),
    tenue: repartition([
      [2, 'Longère', 25, 0.5],
      [1, 'Hydro', 15, 0.3],
      [4, 'Pont', 10, 0.2],
    ]),
    zone: { volume_index: 3, nom: 'Fosse' },
    ...over,
  }
}

describe('construireLaLecture', () => {
  it('sans réponse : pas de lecture', () => {
    expect(construireLaLecture(undefined, 3)).toBeNull()
  })

  it('servie : l’union des deux répartitions, triée sur la tenue, avec l’écart en points arrondis', () => {
    const l = construireLaLecture(reponse(), 3)
    expect(l?.etat).toBe('servie')
    expect(l?.zone).toEqual({ volumeIndex: 3, nom: 'Fosse' })
    expect(l?.lignes.map((x) => [x.nom, x.tenue, x.reference, x.ecart])).toEqual([
      ['Longère', 0.5, 0.3, 20],
      ['Hydro', 0.3, 0.4, -10],
      ['Pont', 0.2, 0, 20],
      ['Fosse', 0, 0.3, -30],
    ])
    expect(l?.maxTenue).toBe(0.5)
    expect(l?.maxReference).toBe(0.4)
    expect(l?.plancher).toEqual({ matchs: 3, reapparitions: 20, joueursPourTenir: 2 })
  })

  it('vide : la référence seule, triée sur elle, avec la raison du serveur et les effectifs de la tenue', () => {
    const rep = reponse({
      etat: 'vide',
      raison: 'sous_plancher_matchs',
      tenue: repartition([], { matchs: 2, reapparitions: 12, suffisante: false, raison: 'sous_plancher_matchs' }),
    })
    const l = construireLaLecture(rep, 3)
    expect(l?.etat).toBe('vide')
    expect(l?.raison).toBe('sous_plancher_matchs')
    expect(l?.lignes.map((x) => [x.nom, x.tenue, x.ecart])).toEqual([
      ['Hydro', null, null],
      ['Longère', null, null],
      ['Fosse', null, null],
    ])
    expect(l?.tenue).toEqual({ matchs: 2, reapparitions: 12, suffisante: false })
    expect(l?.maxTenue).toBe(0)
  })

  it('une raison inconnue du serveur se lit comme un manque de mesures', () => {
    const l = construireLaLecture(reponse({ etat: 'vide', raison: 'code_futur' }), 3)
    expect(l?.raison).toBe('sous_plancher_reapparitions')
  })

  it('sans zone choisie : la réponse de sonde ne sert que par sa référence, aucune zone n’est nommée', () => {
    const l = construireLaLecture(reponse(), null)
    expect(l?.etat).toBe('sans_zone')
    expect(l?.zone).toBeNull()
    expect(l?.raison).toBeNull()
    expect(l?.lignes.every((x) => x.tenue === null)).toBe(true)
    expect(l?.lignes.map((x) => x.nom)).toEqual(['Hydro', 'Longère', 'Fosse'])
  })

  it('une répartition de référence vide (sous le plancher) ne donne aucune ligne', () => {
    const rep = reponse({
      reference: repartition([], { reapparitions: 5, suffisante: false, raison: 'sous_plancher_reapparitions' }),
    })
    const l = construireLaLecture(rep, null)
    expect(l?.lignes).toEqual([])
    expect(l?.reference.suffisante).toBe(false)
  })

  it('une répartition à zones nulles (JSON null) ne fait pas échouer la lecture', () => {
    const rep = reponse({ reference: { ...repartition([]), zones: null } })
    expect(construireLaLecture(rep, null)?.lignes).toEqual([])
  })
})

describe('mesures d’affichage', () => {
  it('l’écart est celui des parts arrondies', () => {
    expect(ecartEnPoints(0.504, 0.296)).toBe(20)
    expect(ecartEnPoints(0.2, 0.4)).toBe(-20)
    expect(ecartEnPoints(0.304, 0.296)).toBe(0)
  })

  it('la puce dit le sens et la valeur absolue', () => {
    expect(puceDEcart(5)).toEqual({ sens: 'haut', texte: '▲ 5' })
    expect(puceDEcart(-3)).toEqual({ sens: 'bas', texte: '▼ 3' })
    expect(puceDEcart(0)).toEqual({ sens: 'egal', texte: '=' })
  })

  it('la teinte croît avec la part rapportée au maximum, bornée', () => {
    expect(opaciteDeLaTeinte(0, 0.5, false)).toBe(0.07)
    expect(opaciteDeLaTeinte(0.5, 0.5, false)).toBe(0.67)
    expect(opaciteDeLaTeinte(0.25, 0.5, false)).toBe(0.37)
    expect(opaciteDeLaTeinte(0.9, 0.5, false)).toBe(0.67)
    expect(opaciteDeLaTeinte(0.4, 0.4, true)).toBe(0.4)
    expect(opaciteDeLaTeinte(0.3, 0, false)).toBe(0.07)
  })

  it('les parts se lisent en pourcentage entier, « <1 % » sous le demi-point, jamais « 0 % » pour une zone qui compte', () => {
    expect(formaterLaPart(0.124, 'en')).toBe('12%')
    expect(formaterLaPart(0.124, 'fr')).toMatch(/^12\s%$/)
    expect(formaterLaPart(0.002, 'en')).toBe('<1%')
    expect(formaterLaPart(0, 'en')).toBe('0%')
  })

  it('une jauge dit sa fraction (bornée à 1) et si le plancher est atteint', () => {
    expect(jaugeContrePlancher(2, 3)).toEqual({ valeur: 2, plancher: 3, fraction: 2 / 3, atteint: false })
    expect(jaugeContrePlancher(30, 20)).toEqual({ valeur: 30, plancher: 20, fraction: 1, atteint: true })
    expect(jaugeContrePlancher(0, 0).atteint).toBe(true)
  })

  it('seules les zones servies au-dessus du seuil, hors zone tenue, portent leur nom sur le plan', () => {
    const l = construireLaLecture(reponse(), 3)
    expect(l && lignesEtiquetees(l).map((x) => x.nom)).toEqual(['Longère', 'Hydro', 'Pont'])
    const sans = construireLaLecture(reponse(), null)
    expect(sans && lignesEtiquetees(sans)).toEqual([])
    const tenue = construireLaLecture(
      reponse({ zone: { volume_index: 1, nom: 'Hydro' }, tenue: repartition([[1, 'Hydro', 10, 0.5]]) }),
      1,
    )
    expect(tenue && lignesEtiquetees(tenue)).toEqual([])
  })
})

function zone(volume_index: number, fr: string, polygon: number[][] | null, over: Partial<CalloutZone> = {}): CalloutZone {
  return {
    name: fr,
    fr,
    en: fr,
    volume_index,
    polygon,
    x: 0,
    y: 0,
    z: 0,
    z_bottom: 0,
    z_top: 1,
    ...over,
  }
}

const CARRE = [
  [0, 0],
  [10, 0],
  [10, 10],
  [0, 10],
]

describe('le catalogue de zones', () => {
  const zones = normalizeCalloutZones([
    zone(7, 'Pont', CARRE),
    zone(3, 'Ascenseur', CARRE),
    zone(5, 'Volume sans contour', null),
    zone(9, '', CARRE),
    zone(4, 'Ascenseur', CARRE),
  ])

  it('propose les zones à contour et à nom, triées par nom puis par indice', () => {
    expect(zonesProposees(zones, 'fr')).toEqual([
      { volumeIndex: 3, nom: 'Ascenseur' },
      { volumeIndex: 4, nom: 'Ascenseur' },
      { volumeIndex: 7, nom: 'Pont' },
    ])
  })

  it('la sonde est le plus petit indice choisissable, null sans zone', () => {
    expect(volumeDeSonde(zones)).toBe(3)
    expect(volumeDeSonde(normalizeCalloutZones([zone(1, 'A', null)]))).toBeNull()
    expect(volumeDeSonde([])).toBeNull()
  })

  it('le chemin inverse Y et chaîne contour, parties et trous', () => {
    const [z] = normalizeCalloutZones([
      zone(1, 'A', CARRE, {
        parts: [
          [
            [20, 0],
            [30, 0],
            [30, 5],
          ],
        ],
        holes: [
          [
            [2, 2],
            [4, 2],
            [4, 4],
          ],
        ],
      }),
    ])
    expect(cheminDeLaZone(z)).toBe('M0 0L10 0L10 -10L0 -10ZM20 0L30 0L30 -5ZM2 -2L4 -2L4 -4Z')
    const [sans] = normalizeCalloutZones([zone(2, 'B', null)])
    expect(cheminDeLaZone(sans)).toBe('')
  })
})

describe('le cadre de la couche et les étiquettes', () => {
  const fond = { originX: 10, originY: 60, widthM: 40, heightM: 30 }
  const bornes = { min_x: 0, max_x: 20, min_y: -10, max_y: 10, valide: true }

  it('le cadre du fond prime, en Y inversé ; sinon les bornes ; sinon rien', () => {
    expect(cadreDeLaCouche(fond, bornes)).toEqual({ x: 10, y: -60, width: 40, height: 30 })
    expect(cadreDeLaCouche(null, bornes)).toEqual({ x: 0, y: -10, width: 20, height: 20 })
    expect(cadreDeLaCouche(null, { ...bornes, valide: false })).toBeNull()
    expect(cadreDeLaCouche(null, undefined)).toBeNull()
  })

  it('une étiquette se pose en pourcentages du cadre, hors cadre = rien', () => {
    const repere = repereDuPlan(fond, { ...bornes }, 1)
    expect(repere).not.toBeNull()
    if (!repere) return
    // Coin haut-gauche monde (originX, originY) -> (0 %, 0 %) ; centre du cadre -> (50 %, 50 %).
    expect(positionDeLEtiquette({ x: 10, y: 60 }, repere)).toEqual({ left: '0.00%', top: '0.00%' })
    expect(positionDeLEtiquette({ x: 30, y: 45 }, repere)).toEqual({ left: '50.00%', top: '50.00%' })
    expect(positionDeLEtiquette({ x: 60, y: 45 }, repere)).toBeNull()
  })

  it('la rotation de la carte déplace l’étiquette comme le fond', () => {
    const repere = repereDuPlan(fond, { ...bornes }, 1, 1)
    expect(repere).not.toBeNull()
    if (!repere) return
    // Un quart de tour horaire : le coin haut-gauche du monde passe en haut à droite.
    expect(positionDeLEtiquette({ x: 10, y: 60 }, repere)).toEqual({ left: '100.00%', top: '0.00%' })
  })
})
