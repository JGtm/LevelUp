/**
 * Tests — padPresenceRefine : L'INCERTITUDE RAMENÉE À CE QU'UN JOUEUR A PU FAIRE.
 *
 * CE QUE CE FICHIER VERROUILLE, et chaque point est une clause de la décision utilisateur du
 * 2026-08-27 (« un socle incertain alors que tous les joueurs étaient à l'autre bout de la
 * map ») :
 *  - une APPROCHE remonte `tLow` juste avant elle, et pas plus loin ;
 *  - la PREMIÈRE approche gagne : c'est le premier instant où l'arme a PU partir ;
 *  - AUCUNE approche : le socle reste PLEIN jusqu'à `tHigh`, et bascule vide À `tHigh` — la
 *    preuve d'absence garde tous ses droits, le compte à rebours aussi ;
 *  - une fenêtre JAMAIS FERMÉE (`tHigh <= tLow`) n'est pas touchée : il n'y a rien à raffiner
 *    là où aucune absence n'a été prouvée ;
 *  - un passage RAPIDE entre deux échantillons est attrapé — c'est le segment qui compte, pas
 *    les points ;
 *  - `t0`, `tHigh`, `spawns` et `cycle` ne bougent JAMAIS.
 *
 * LA LECTURE SE VÉRIFIE PAR `padStateAt`, et pas seulement sur les nombres : ce qui compte est
 * ce que l'écran affiche à une image donnée.
 */
import { describe, expect, it } from 'vitest'

import {
  PAD_APPROACH_RADIUS_M,
  PAD_BETWEEN_FRAMES,
  refinePadPresence,
} from './padPresenceRefine'
import type { ReplayTrackReady, ReplayWeaponPadReady } from '../../../lib/replay/replayNormalize'
import { padStateAt } from './weaponPadTime'

/** Le socle témoin : à l'origine, une occupation dont l'absence est prouvée à l'image 200. */
function pad(over: Partial<ReplayWeaponPadReady> = {}): ReplayWeaponPadReady {
  return {
    x: 0,
    y: 0,
    weapon: '0x0A1992BC',
    spawns: [0],
    presence: [{ t0: 0, tLow: 100, tHigh: 200 }],
    ...over,
  }
}

/** Une trace : une suite de positions datées, comme le film les réplique. */
function track(points: { t: number; x: number; y: number }[]): ReplayTrackReady {
  return { slot: 1, team: -1, points }
}

/** Un joueur qui reste à `x` mètres du socle pendant toute la fenêtre. */
function loin(x: number): ReplayTrackReady {
  return track([
    { t: 0, x, y: 0 },
    { t: 300, x, y: 0 },
  ])
}

const seul = (pads: readonly ReplayWeaponPadReady[]) => pads[0].presence[0]

describe('refinePadPresence — une approche remonte la borne, et rien d’autre', () => {
  it('APPROCHE UNIQUE : `tLow` monte à la dernière image AVANT le passage', () => {
    // Le joueur part de 10 m et atteint le socle à l'image 150 : il entre dans le rayon de 2 m
    // peu avant, et la borne se pose sur la dernière image pleine qui précède cette entrée.
    const t = track([
      { t: 100, x: 10, y: 0 },
      { t: 150, x: 0, y: 0 },
    ])
    const occ = seul(refinePadPresence([pad()], [t]))
    expect(occ.tLow).toBeGreaterThan(100)
    expect(occ.tLow).toBeLessThan(150)
    // 10 m parcourus en 50 images : le rayon de 2 m est atteint à 4/5 du trajet, soit ~140.
    expect(occ.tLow).toBe(139)
    // Ce qui compte à l'écran : plein AVANT, incertain À PARTIR de la borne.
    const raffine = refinePadPresence([pad()], [t])[0]
    expect(padStateAt(raffine, 138)).toBe('full')
    expect(padStateAt(raffine, 139)).toBe('uncertain')
    expect(padStateAt(raffine, 200)).toBe('empty')
  })

  it('les bornes que la mesure publie NE BOUGENT PAS : t0, tHigh, spawns, cycle', () => {
    const cycle = { medianS: 40, p10S: 40, p90S: 40, gaps: 2, missing: 0 }
    const source = pad({ spawns: [0, 300], cycle })
    const out = refinePadPresence([source], [track([{ t: 120, x: 0, y: 0 }, { t: 130, x: 0, y: 0 }])])
    expect(out[0].spawns).toEqual([0, 300])
    expect(out[0].cycle).toEqual(cycle)
    expect(seul(out).t0).toBe(0)
    expect(seul(out).tHigh).toBe(200)
  })

  it('APPROCHES MULTIPLES : la PREMIÈRE gagne — c’est là que l’arme a pu partir', () => {
    const tot = track([
      { t: 100, x: 10, y: 0 },
      { t: 120, x: 0, y: 0 },
      { t: 140, x: 10, y: 0 },
    ])
    const tard = track([
      { t: 100, x: 20, y: 0 },
      { t: 180, x: 0, y: 0 },
    ])
    const seule = seul(refinePadPresence([pad()], [tot]))
    const deux = seul(refinePadPresence([pad()], [tard, tot]))
    expect(deux.tLow).toBe(seule.tLow)
    expect(deux.tLow).toBeLessThan(150)
  })

  it('la borne ne DESCEND jamais : une approche avant `tLow` ne rend pas d’incertitude', () => {
    // Le joueur est sur le socle dès l'image 0, bien avant la fenêtre : rien à raffiner.
    const colle = track([
      { t: 0, x: 0, y: 0 },
      { t: 300, x: 0, y: 0 },
    ])
    expect(seul(refinePadPresence([pad()], [colle])).tLow).toBe(100)
  })
})

describe('refinePadPresence — personne n’est passé : le socle reste PLEIN', () => {
  it('AUCUNE APPROCHE : plein jusqu’à la preuve d’absence, vide À la preuve d’absence', () => {
    const out = refinePadPresence([pad()], [loin(50)])
    const raffine = out[0]
    // C'est le cas de l'utilisateur : l'ancienne lecture disait « incertain » de 100 à 199.
    expect(padStateAt(pad(), 150)).toBe('uncertain')
    expect(padStateAt(raffine, 100)).toBe('full')
    expect(padStateAt(raffine, 150)).toBe('full')
    expect(padStateAt(raffine, 199)).toBe('full')
    // ET LE BASCULEMENT VIDE N'A PAS BOUGÉ : `tHigh` garde tous ses droits.
    expect(padStateAt(raffine, 200)).toBe('empty')
    expect(padStateAt(raffine, 500)).toBe('empty')
  })

  /**
   * LA LAME EST RÉELLE, ET ELLE DOIT RESTER FINE (revue adversariale du 2026-08-27). L'image de
   * lecture est FRACTIONNAIRE (`useReplayPlayback` avance de `dt × fps` sans arrondi) : la
   * borne ne peut donc pas être posée « entre deux images » au sens où rien ne l'atteindrait.
   * Ce qui est vérifiable, et ce qui compte, c'est que la lame reste étroite — 1/64 d'image,
   * ~1,6 ms de film — et que le sentinel « jamais vidé » ne soit jamais déclenché.
   */
  it('la lame « personne n’est passé » est ÉTROITE, et ne réveille pas le sentinel', () => {
    expect(PAD_BETWEEN_FRAMES).toBeGreaterThan(0)
    expect(PAD_BETWEEN_FRAMES).toBeLessThanOrEqual(1 / 64)
    const occ = seul(refinePadPresence([pad()], [loin(50)]))
    expect(occ.tLow).toBe(occ.tHigh - PAD_BETWEEN_FRAMES)
    // `tHigh > tLow` reste vrai : c'est cette inégalité qui garde la bascule « vide » vivante.
    expect(occ.tHigh).toBeGreaterThan(occ.tLow)
    // Et AUCUNE image entière de la fenêtre ne se lit « incertain ».
    const raffine = refinePadPresence([pad()], [loin(50)])[0]
    for (let f = 100; f < 200; f++) expect(padStateAt(raffine, f)).toBe('full')
  })

  it('un joueur qui reste JUSTE au-delà du rayon ne change rien', () => {
    const out = refinePadPresence([pad()], [loin(PAD_APPROACH_RADIUS_M + 0.01)])
    expect(padStateAt(out[0], 150)).toBe('full')
  })

  it('un joueur JUSTE dans le rayon, lui, compte', () => {
    const out = refinePadPresence([pad()], [loin(PAD_APPROACH_RADIUS_M - 0.01)])
    expect(seul(out).tLow).toBe(100)
    expect(padStateAt(out[0], 150)).toBe('uncertain')
  })
})

describe('refinePadPresence — ce que la règle ne touche pas', () => {
  it('une fenêtre JAMAIS FERMÉE (`tHigh <= tLow`) reste intacte', () => {
    const jamais = pad({ presence: [{ t0: 0, tLow: 3464, tHigh: 3464 }] })
    const out = refinePadPresence([jamais], [loin(50)])
    expect(out[0]).toBe(jamais)
    expect(seul(out)).toEqual({ t0: 0, tLow: 3464, tHigh: 3464 })
    expect(padStateAt(out[0], 4000)).toBe('full')
  })

  it('un socle SANS occupation traverse sans dommage', () => {
    const vide = pad({ spawns: [], presence: [] })
    expect(refinePadPresence([vide], [loin(1)])[0]).toBe(vide)
  })

  it('sans trace, ou sans socle, la liste d’origine est rendue TELLE QUELLE', () => {
    const pads = [pad()]
    expect(refinePadPresence(pads, [])).toBe(pads)
    expect(refinePadPresence([], [loin(1)])).toEqual([])
  })

  it('une trace VIDE ne fait pas tomber la passe', () => {
    expect(padStateAt(refinePadPresence([pad()], [track([])])[0], 150)).toBe('full')
  })

  it('une trace à UN SEUL point compte quand ce point est sur le socle', () => {
    // Une vie d'une seule image répliquée, pile sur le socle : c'est une approche, et la boucle
    // par segments ne peut pas la voir (il n'y a pas de paire). Le cas isolé existe pour elle.
    const unique = track([{ t: 150, x: 0, y: 0 }])
    expect(seul(refinePadPresence([pad()], [unique])).tLow).toBe(149)
    const loinUnique = track([{ t: 150, x: 50, y: 0 }])
    expect(padStateAt(refinePadPresence([pad()], [loinUnique])[0], 150)).toBe('full')
  })
})

describe('refinePadPresence — le SEGMENT, pas les points', () => {
  it('un passage RAPIDE entre deux échantillons est attrapé', () => {
    // Deux échantillons seulement, à 20 m de part et d'autre du socle : AUCUN n'est dans le
    // rayon, mais la trajectoire passe exactement dessus. C'est le défaut que le test aux seuls
    // points laisserait passer — un joueur véhiculé traverse une arène entre deux images.
    const traverse = track([
      { t: 140, x: -20, y: 0 },
      { t: 160, x: 20, y: 0 },
    ])
    const auxPoints = traverse.points.every(
      (p) => Math.hypot(p.x, p.y) > PAD_APPROACH_RADIUS_M,
    )
    expect(auxPoints, 'aucun échantillon n’est dans le rayon').toBe(true)
    const occ = seul(refinePadPresence([pad()], [traverse]))
    expect(occ.tLow).toBeGreaterThanOrEqual(140)
    expect(occ.tLow).toBeLessThan(150)
  })

  /**
   * LE FILET DU CLIP (revue adversariale du 2026-08-27). Ce cas ne vérifiait que l'état à
   * l'image 150, et il passait AVEC comme SANS la coupure à la fenêtre : sans clip, `tLow`
   * devient `tHigh`, ce que `padStateAt` lit comme le sentinel « jamais vidé » — donc « plein »
   * à 150 aussi, et le cas ne voyait rien. Ce qui DISCRIMINE est l'après : un socle correctement
   * raffiné bascule VIDE à `tHigh`, un socle tombé dans le sentinel reste plein pour toujours.
   */
  it('un segment qui ne touche le socle qu’APRÈS la fenêtre ne compte pas', () => {
    const apres = track([
      { t: 200, x: 20, y: 0 },
      { t: 260, x: 0, y: 0 },
    ])
    const raffine = refinePadPresence([pad()], [apres])[0]
    expect(padStateAt(raffine, 150)).toBe('full')
    // LES DEUX ASSERTIONS QUI MORDENT : la borne est la lame « personne n'est passé », et le
    // socle bascule bien vide à la preuve d'absence au lieu de se figer plein.
    expect(raffine.presence[0].tLow).toBe(200 - PAD_BETWEEN_FRAMES)
    expect(padStateAt(raffine, 250)).toBe('empty')
  })
})

describe('refinePadPresence — plusieurs socles, plusieurs occupations', () => {
  it('chaque occupation est raffinée pour elle-même', () => {
    const deux = pad({
      spawns: [0, 300],
      presence: [
        { t0: 0, tLow: 100, tHigh: 200 },
        { t0: 300, tLow: 400, tHigh: 500 },
      ],
    })
    // Un seul passage, dans la SECONDE fenêtre : la première reste pleine, la seconde bascule.
    const t = track([
      { t: 400, x: 10, y: 0 },
      { t: 450, x: 0, y: 0 },
    ])
    const out = refinePadPresence([deux], [t])[0]
    expect(padStateAt(out, 150)).toBe('full')
    expect(padStateAt(out, 199)).toBe('full')
    expect(padStateAt(out, 200)).toBe('empty')
    expect(out.presence[1].tLow).toBeGreaterThan(400)
    expect(padStateAt(out, 449)).toBe('uncertain')
  })

  it('deux socles éloignés ne se contaminent pas', () => {
    const ici = pad()
    const laBas = pad({ x: 40, y: 40 })
    const t = track([
      { t: 100, x: 10, y: 0 },
      { t: 150, x: 0, y: 0 },
    ])
    const [a, b] = refinePadPresence([ici, laBas], [t])
    expect(padStateAt(a, 150)).toBe('uncertain')
    // Le socle éloigné est bien RAFFINÉ — c'est tout l'objet de la règle : personne n'est passé
    // dessus, il reste plein sur toute la fenêtre et ne bascule qu'à la preuve d'absence.
    expect(padStateAt(b, 150)).toBe('full')
    expect(padStateAt(b, 199)).toBe('full')
    expect(padStateAt(b, 200)).toBe('empty')
  })
})
