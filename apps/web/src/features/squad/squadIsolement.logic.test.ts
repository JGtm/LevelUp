import { describe, expect, it } from 'vitest'

import type { SquadIsolementMort, SquadIsolementRepere } from '@/lib/api/types'

import {
  delaiSecondes,
  echelleAvecBande,
  echelleLogAvecBande,
  echellesNuage,
  etatMort,
  fenetreSecondes,
  graduationsDelai,
  plafondSecondes,
  PLANCHER_DELAI_S,
  positionMort,
  positionRepere,
  ordreDessinReperes,
  repereAttenue,
  tailleRepere,
} from './squadIsolement.logic'

function couverture(brut: number, n: number, echantillonFaible: boolean, matchs = 3) {
  return {
    taux: n > 0 ? brut / n : 0,
    brut,
    par_match: matchs > 0 ? brut / matchs : 0,
    n,
    echantillon_faible: echantillonFaible,
  }
}

function mort(over: Partial<SquadIsolementMort> = {}): SquadIsolementMort {
  return {
    xuid: 'x1',
    gamertag: 'Alice',
    match_id: 'm1',
    time_ms: 10_000,
    distance_ratio: 0.6,
    hors_de_vue: false,
    vengee: true,
    hors_fenetre: false,
    delai_ms: 3_000,
    ...over,
  } as SquadIsolementMort
}

/** Une mort dont le tueur est tombé APRÈS la fenêtre : délai publié, jamais une riposte. */
function mortHorsFenetre(delaiMs: number): SquadIsolementMort {
  return mort({ vengee: false, hors_fenetre: true, delai_ms: delaiMs })
}

function repere(over: Partial<SquadIsolementRepere> = {}): SquadIsolementRepere {
  return {
    xuid: 'x1',
    gamertag: 'Alice',
    nb_morts: 10,
    mediane_distance_ratio: 0.8,
    mediane_delai_ms: 4_000,
    part_isolee: couverture(4, 10, true),
    couverture: couverture(6, 10, true),
    ...over,
  } as SquadIsolementRepere
}

describe('echelleAvecBande', () => {
  it('sans valeur, l’échelle tient sur son minimum et réserve quand même sa bande', () => {
    const e = echelleAvecBande([], 0.5, 2)
    expect(e.mesure).toBe(2)
    expect(e.bandeDebut).toBe(2.5)
    expect(e.max).toBe(3.5)
    expect(e.bandeCentre).toBe(3)
  })

  it('arrondit le haut de la zone mesurée au pas supérieur', () => {
    expect(echelleAvecBande([2.2], 0.5, 2).mesure).toBe(2.5)
    expect(echelleAvecBande([12.4], 1, 10).mesure).toBe(13)
  })

  it('la bande commence APRÈS la zone mesurée : aucune valeur réelle ne s’y pose', () => {
    const e = echelleAvecBande([3.1], 0.5, 2)
    expect(e.bandeDebut).toBeGreaterThan(e.mesure)
    expect(e.bandeCentre).toBeGreaterThan(e.bandeDebut)
    expect(e.max).toBeGreaterThanOrEqual(e.bandeCentre)
  })
})

// ─── LES TROIS ÉTATS D'UNE MORT (règle des 5 s partout, décision du 2026-09-22) ─────

describe('etatMort', () => {
  it('le tueur tombé DANS la fenêtre est une riposte', () => {
    expect(etatMort(mort({ vengee: true, delai_ms: 3_000 }))).toBe('ripostee')
  })

  it('le tueur tombé APRÈS la fenêtre n’en est pas une — et garde son délai', () => {
    expect(etatMort(mortHorsFenetre(60_000))).toBe('horsFenetre')
  })

  it('aucune riposte connue : ni l’un ni l’autre', () => {
    expect(etatMort(mort({ vengee: false, hors_fenetre: false, delai_ms: undefined }))).toBe(
      'jamais',
    )
  })

  it('les trois états sont EXCLUSIFS : le serveur tranche, le client n’arbitre pas', () => {
    const etats = [
      etatMort(mort()),
      etatMort(mortHorsFenetre(12_000)),
      etatMort(mort({ vengee: false, hors_fenetre: false, delai_ms: undefined })),
    ]
    expect(new Set(etats).size).toBe(3)
  })
})

describe('fenetreSecondes', () => {
  it('lit la fenêtre DU CONTRAT, jamais un 5 codé en dur', () => {
    expect(fenetreSecondes({ fenetre_ms: 5_000 })).toBe(5)
    expect(fenetreSecondes({ fenetre_ms: 8_000 })).toBe(8)
  })
})

describe('plafondSecondes', () => {
  it('lit le plafond DU CONTRAT, jamais un 60 codé en dur', () => {
    expect(plafondSecondes({ plafond_ms: 60_000 })).toBe(60)
    expect(plafondSecondes({ plafond_ms: 30_000 })).toBe(30)
  })
})

describe('delaiSecondes', () => {
  it('rend le délai en secondes d’une mort ripostée', () => {
    expect(delaiSecondes(mort({ delai_ms: 2_500 }))).toBe(2.5)
  })

  it('rend AUSSI le délai d’une mort hors fenêtre : c’est tout l’intérêt de la garder', () => {
    expect(delaiSecondes(mortHorsFenetre(60_000))).toBe(60)
  })

  it('rend null quand aucune riposte n’est connue — jamais zéro', () => {
    expect(
      delaiSecondes(mort({ vengee: false, hors_fenetre: false, delai_ms: undefined })),
    ).toBeNull()
  })
})

// ─── L'AXE DES DÉLAIS EST LOGARITHMIQUE ET BORNÉ (décision du 2026-09-22) ───────

describe('echelleLogAvecBande', () => {
  it('réserve la bande AU-DESSUS du plafond mesuré, et jamais en dessous', () => {
    const e = echelleLogAvecBande(PLANCHER_DELAI_S, 60)
    expect(e.min).toBe(PLANCHER_DELAI_S)
    expect(e.mesure).toBe(60)
    expect(e.bandeDebut).toBeGreaterThan(60)
    expect(e.max).toBeGreaterThan(e.bandeDebut)
  })

  it('le centre de la bande est sa moyenne GÉOMÉTRIQUE — le milieu une fois l’axe en log', () => {
    const e = echelleLogAvecBande(PLANCHER_DELAI_S, 60)
    expect(e.bandeCentre).toBeGreaterThan(e.bandeDebut)
    expect(e.bandeCentre).toBeLessThan(e.max)
    // Milieu en log : la distance log au début vaut la distance log au sommet.
    expect(Math.log(e.bandeCentre) - Math.log(e.bandeDebut)).toBeCloseTo(
      Math.log(e.max) - Math.log(e.bandeCentre),
      10,
    )
  })
})

describe('graduationsDelai', () => {
  it('nomme les durées reconnaissables, et aucune au-delà du plafond', () => {
    const e = echelleLogAvecBande(PLANCHER_DELAI_S, 60)
    expect(graduationsDelai(e)).toEqual([0.2, 0.5, 1, 2, 5, 10, 30, 60])
  })

  it('un plafond plus bas coupe les graduations qui n’ont plus de zone où se poser', () => {
    expect(graduationsDelai(echelleLogAvecBande(PLANCHER_DELAI_S, 10))).toEqual([
      0.2, 0.5, 1, 2, 5, 10,
    ])
  })
})

describe('echellesNuage', () => {
  it('l’axe des délais vient du CONTRAT, pas du maximum observé', () => {
    const e = echellesNuage([mort({ delai_ms: 3_000 }), mortHorsFenetre(30_000)], 60)
    expect(e.delai.min).toBe(PLANCHER_DELAI_S)
    expect(e.delai.mesure).toBe(60)
    expect(e.delai.bandeDebut).toBeGreaterThan(60)
  })

  it('L’AXE DES DISTANCES RESTE LINÉAIRE ET LIBRE : un ratio se lit en multiples', () => {
    const e = echellesNuage([mort({ distance_ratio: 3.1 })], 60)
    expect(e.distance.mesure).toBe(3.5)
  })
})

describe('positionMort', () => {
  const echelles = echellesNuage([mort({ distance_ratio: 1.2, delai_ms: 4_000 })], 60)

  it('pose une mort hors fenêtre à SON délai, jamais dans la bande « jamais ripostée »', () => {
    const [, y] = positionMort(mortHorsFenetre(60_000), echelles)
    expect(y).toBe(60)
    expect(y).toBeLessThan(echelles.delai.bandeDebut)
  })

  it('POSE au plancher un délai plus court que lui — un dessin, jamais une mesure', () => {
    const [, y] = positionMort(mort({ delai_ms: 90 }), echelles)
    expect(y).toBe(PLANCHER_DELAI_S)
    // La VRAIE valeur reste lisible pour l'infobulle : le plancher ne la touche pas.
    expect(delaiSecondes(mort({ delai_ms: 90 }))).toBe(0.09)
  })

  it('RESPECTE LE PLAFOND : aucun point ne peut se poser dans la bande réservée', () => {
    const [, y] = positionMort(mortHorsFenetre(200_000), echelles)
    expect(y).toBe(echelles.delai.mesure)
    expect(y).toBeLessThan(echelles.delai.bandeDebut)
  })

  it('3 s et 30 s ne tombent pas au même endroit de l’axe log — une décade les sépare', () => {
    const fraction = (v: number) =>
      (Math.log(v) - Math.log(echelles.delai.min)) /
      (Math.log(echelles.delai.max) - Math.log(echelles.delai.min))
    const [, trois] = positionMort(mortHorsFenetre(3_000), echelles)
    const [, trente] = positionMort(mortHorsFenetre(30_000), echelles)
    expect(fraction(trente) - fraction(trois)).toBeGreaterThan(0.2)
    // Et le nuage utile — sous la fenêtre de 5 s — garde plus du tiers de la hauteur,
    // alors qu'en linéaire jusqu'à 60 s il en occupait un douzième.
    expect(fraction(5)).toBeGreaterThan(0.33)
  })

  it('pose une mort mesurée à ses deux coordonnées réelles', () => {
    expect(positionMort(mort({ distance_ratio: 1.2, delai_ms: 4_000 }), echelles)).toEqual([1.2, 4])
  })

  it('pose une mort hors de vue dans la bande de distance', () => {
    const [x] = positionMort(mort({ distance_ratio: undefined, hors_de_vue: true }), echelles)
    expect(x).toBe(echelles.distance.bandeCentre)
  })

  it('pose une mort sans riposte connue dans la bande de délai', () => {
    const [, y] = positionMort(
      mort({ vengee: false, hors_fenetre: false, delai_ms: undefined }),
      echelles,
    )
    expect(y).toBe(echelles.delai.bandeCentre)
  })
})

describe('positionRepere', () => {
  const echelles = echellesNuage([mort()], 60)

  it('pose le repère sur ses deux médianes', () => {
    expect(positionRepere(repere({ mediane_distance_ratio: 0.9, mediane_delai_ms: 5_000 }), echelles))
      .toEqual([0.9, 5])
  })

  it('renvoie le repère dans les bandes quand une médiane manque', () => {
    const [x, y] = positionRepere(
      repere({ mediane_distance_ratio: undefined, mediane_delai_ms: undefined }),
      echelles,
    )
    expect(x).toBe(echelles.distance.bandeCentre)
    expect(y).toBe(echelles.delai.bandeCentre)
  })
})

describe('tailleRepere', () => {
  it('projette le volume sur la plage réelle du roster', () => {
    expect(tailleRepere(10, 10, 50)).toBeLessThan(tailleRepere(50, 10, 50))
  })

  it('min === max : taille médiane de la plage, aucun écart à montrer', () => {
    expect(tailleRepere(30, 30, 30)).toBe(tailleRepere(99, 30, 30))
  })
})

describe('repereAttenue', () => {
  it('suit la réserve d’échantillon du taux d’isolement', () => {
    expect(repereAttenue(repere({ part_isolee: couverture(4, 10, true) }))).toBe(true)
    expect(repereAttenue(repere({ part_isolee: couverture(12, 40, false) }))).toBe(false)
  })
})

// ─── ORDRE DE DESSIN DES REPÈRES (retour utilisateur du 2026-09-21) ──────────

describe('ordreDessinReperes', () => {
  it('trie par TAILLE DÉCROISSANTE : le plus petit repère finit au premier plan', () => {
    // ECharts dessine la dernière série au-dessus. Le plus GROS part donc en premier —
    // sans quoi il recouvrait entièrement le repère d'un coéquipier moins exposé.
    const tries = ordreDessinReperes([
      repere({ gamertag: 'Petit', nb_morts: 5 }),
      repere({ gamertag: 'Gros', nb_morts: 90 }),
      repere({ gamertag: 'Moyen', nb_morts: 40 }),
    ])
    expect(tries.map((r) => r.gamertag)).toEqual(['Gros', 'Moyen', 'Petit'])
    expect(tries.map((r) => r.nb_morts)).toEqual([90, 40, 5])
  })

  it('conserve l’ordre du roster à volume ÉGAL (tri stable)', () => {
    const tries = ordreDessinReperes([
      repere({ gamertag: 'Alice', nb_morts: 12 }),
      repere({ gamertag: 'Bob', nb_morts: 12 }),
    ])
    expect(tries.map((r) => r.gamertag)).toEqual(['Alice', 'Bob'])
  })

  it('ne mute pas la liste d’entrée', () => {
    const entree = [repere({ gamertag: 'A', nb_morts: 1 }), repere({ gamertag: 'B', nb_morts: 9 })]
    ordreDessinReperes(entree)
    expect(entree.map((r) => r.gamertag)).toEqual(['A', 'B'])
  })
})
