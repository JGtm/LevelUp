/**
 * skullPresence.test.ts — LA RÈGLE DE PRÉSENCE DU CRÂNE, et surtout ses CONTRE-ÉPREUVES.
 *
 * Le bug corrigé : le crâne au repos sur son socle (vie instant-unique, t0 == t1) clignotait une
 * image puis disparaissait. La règle le TIENT au repos — mais seulement quand une PRISE le
 * corrobore, jamais au point de chute-dans-le-vide. Ces deux propriétés sont l'objet des tests :
 *   - un repos-socle suivi d'une prise est tenu sur TOUT le trou (pas une image) ;
 *   - une chute-dans-le-vide (repos suivi d'une VIE de respawn) rend `absent`, PAS un fantôme.
 */
import { describe, expect, it } from 'vitest'

import { skullPresenceAt, skullSocle } from './skullPresence'

import type { ReplayObjectiveObjectReady, ReplaySkullCarry } from './replayNormalize'

/** La position réelle du socle dans l'artefact d9781168 : centre de carte. */
const SOCLE = { x: 3.6, y: 0.97 }

function life(
  t0: number, t1: number, pts: Array<{ t: number; x: number; y: number }>,
): ReplayObjectiveObjectReady {
  return { family: 'ball', en: 'Oddball', fr: 'Crâne', t0, t1, pts }
}

/** Une vie instant-unique (repos) : un seul point, t0 == t1. */
function rest(t: number, x: number, y: number): ReplayObjectiveObjectReady {
  return life(t, t, [{ t, x, y }])
}

function carry(t0: number, t1: number): ReplaySkullCarry {
  return { closed: true, t0, t1, xuid: 'xuid(1)' }
}

describe('skullPresenceAt — précédence du portage', () => {
  it('rend {carried} quand une carry couvre l image, MÊME si une vie s y superpose', () => {
    const lives = [life(10, 20, [{ t: 10, x: 5, y: 5 }])] // vie fictive superposée
    const carries = [carry(10, 20)]
    expect(skullPresenceAt(lives, carries, 15)).toEqual({ state: 'carried' })
  })
})

describe('skullPresenceAt — vie active (comportement historique)', () => {
  const lives = [life(10, 12, [{ t: 10, x: 0, y: 0 }, { t: 11, x: 1, y: 0 }, { t: 12, x: 2, y: 0 }])]

  it('rend {free} au point ÉMIS, rolling:true tant qu un point suit', () => {
    expect(skullPresenceAt(lives, [], 10)).toEqual({ state: 'free', at: { x: 0, y: 0 }, rolling: true })
    expect(skullPresenceAt(lives, [], 11)).toEqual({ state: 'free', at: { x: 1, y: 0 }, rolling: true })
  })

  it('rolling:false au dernier point de la vie', () => {
    expect(skullPresenceAt(lives, [], 12)).toEqual({ state: 'free', at: { x: 2, y: 0 }, rolling: false })
  })
})

describe('skullPresenceAt — TENUE jusqu à la prise, et sa contre-épreuve', () => {
  // Une vie qui finit en P=(2,0) à t1=12.
  const vie = life(10, 12, [{ t: 10, x: 0, y: 0 }, { t: 11, x: 1, y: 0 }, { t: 12, x: 2, y: 0 }])

  it('TIENT le crâne à P quand le prochain début est une PRISE', () => {
    const carries = [carry(20, 25)]
    // Frames du trou (12, 20) : tenu à P, immobile.
    expect(skullPresenceAt([vie], carries, 15)).toEqual({ state: 'free', at: { x: 2, y: 0 }, rolling: false })
    expect(skullPresenceAt([vie], carries, 19)).toEqual({ state: 'free', at: { x: 2, y: 0 }, rolling: false })
  })

  it('CONTRE-ÉPREUVE : si le prochain début est une VIE, le trou est {absent}', () => {
    const respawn = life(20, 22, [{ t: 20, x: 9, y: 9 }])
    expect(skullPresenceAt([vie, respawn], [], 15)).toEqual({ state: 'absent' })
    expect(skullPresenceAt([vie, respawn], [], 19)).toEqual({ state: 'absent' })
  })
})

describe('skullPresenceAt — LE BUG : repos-socle tenu sur tout le trou', () => {
  const socle = rest(20, SOCLE.x, SOCLE.y) // instant-unique au socle
  const carries = [carry(30, 40)]

  it('rend {free} au socle à l image d émission (vie active)', () => {
    expect(skullPresenceAt([socle], carries, 20)).toEqual({ state: 'free', at: SOCLE, rolling: false })
  })

  it('TIENT le socle sur TOUT le trou (21..29), pas une seule image', () => {
    for (const f of [21, 25, 29]) {
      expect(skullPresenceAt([socle], carries, f)).toEqual({ state: 'free', at: SOCLE, rolling: false })
    }
  })

  it('bascule {carried} dès la prise', () => {
    expect(skullPresenceAt([socle], carries, 30)).toEqual({ state: 'carried' })
  })
})

describe('skullPresenceAt — PIÈGE void-drop : jamais un fantôme au point de chute', () => {
  const CHUTE = { x: 8, y: 8 } // hors socle
  const chute = rest(10, CHUTE.x, CHUTE.y) // instant à la position de mort
  const socle = rest(20, SOCLE.x, SOCLE.y) // respawn au socle
  const carries = [carry(30, 40)] // reprise après respawn

  it('la fenêtre de la chute (11..19) est {absent}, PAS {free, at: chute}', () => {
    for (const f of [11, 15, 19]) {
      const p = skullPresenceAt([chute, socle], carries, f)
      expect(p).toEqual({ state: 'absent' })
      expect(p).not.toEqual({ state: 'free', at: CHUTE, rolling: false })
    }
  })

  it('la fenêtre du socle tenu (21..29) est {free, at: socle}', () => {
    for (const f of [21, 25, 29]) {
      expect(skullPresenceAt([chute, socle], carries, f)).toEqual({ state: 'free', at: SOCLE, rolling: false })
    }
  })
})

describe('skullPresenceAt — bornes', () => {
  const lives = [life(10, 12, [{ t: 10, x: 0, y: 0 }, { t: 12, x: 2, y: 0 }])]

  it('avant la 1re émission : {absent}', () => {
    expect(skullPresenceAt(lives, [], 5)).toEqual({ state: 'absent' })
  })

  it('après le dernier événement sans prise suivante : {absent}', () => {
    expect(skullPresenceAt(lives, [], 100)).toEqual({ state: 'absent' })
  })

  it('avec carries:[] (artefact pré-schéma-23), la présence retombe sur la vie active seule', () => {
    // Dégradation : aucune prise ne suit jamais → aucun maintien, comportement historique.
    expect(skullPresenceAt(lives, [], 11)).toEqual({ state: 'free', at: { x: 0, y: 0 }, rolling: true })
    expect(skullPresenceAt(lives, [], 13)).toEqual({ state: 'absent' })
  })
})

describe('skullPresenceAt — GARDE fixture d9781168 (frames concrètes)', () => {
  // Les quatre void-drops mesurés : une chute instant-unique HORS socle juste avant la fenêtre,
  // puis le respawn instant-unique AU socle. Sans prise dans la fenêtre → elle doit être absente
  // (le prochain début est la VIE de respawn), et l instant-socle doit répliquer au socle.
  const drops: Array<{ chute: number; windowMid: number; socle: number }> = [
    { chute: 1033, windowMid: 1050, socle: 1075 },
    { chute: 4772, windowMid: 4800, socle: 4813 },
    { chute: 5543, windowMid: 5560, socle: 5584 },
    { chute: 6678, windowMid: 6700, socle: 6719 },
  ]
  const lives = drops.flatMap((d) => [rest(d.chute, 8, 8), rest(d.socle, SOCLE.x, SOCLE.y)])

  it('chaque fenêtre de chute rend {absent} (aucun fantôme à la position de mort)', () => {
    for (const d of drops) {
      expect(skullPresenceAt(lives, [], d.windowMid)).toEqual({ state: 'absent' })
    }
  })

  it('chaque instant-socle réplique {free} au socle', () => {
    for (const d of drops) {
      expect(skullPresenceAt(lives, [], d.socle)).toEqual({ state: 'free', at: SOCLE, rolling: false })
    }
  })
})

describe('skullSocle — le socle est le MODE des vies-instant', () => {
  it('rend la position RÉCURRENTE, ignore les chutes ponctuelles', () => {
    // Socle deux fois (le crâne y réapparaît), une chute isolée ailleurs.
    const lives = [rest(10, 3.6, 0.97), rest(50, 3.6, 0.97), rest(90, -22, 9)]
    expect(skullSocle(lives)).toEqual({ x: 3.6, y: 0.97 })
  })

  it('rend null sans récurrence (une vie-instant isolée = plutôt une chute qu un socle)', () => {
    expect(skullSocle([rest(10, -22, 9)])).toBeNull()
    expect(skullSocle([])).toBeNull()
  })
})

describe('skullPresenceAt — le crâne au REPOS est sur son socle', () => {
  it('pose le crâne sur le socle AVANT sa première émission (au lieu d absent)', () => {
    const lives = [life(100, 102, [{ t: 100, x: 5, y: 5 }, { t: 102, x: 6, y: 5 }])]
    expect(skullPresenceAt(lives, [], 0, SOCLE)).toEqual({ state: 'free', at: SOCLE, rolling: false })
  })

  it('sans socle (null, défaut), reste absent — comportement historique inchangé', () => {
    const lives = [life(100, 102, [{ t: 100, x: 5, y: 5 }, { t: 102, x: 6, y: 5 }])]
    expect(skullPresenceAt(lives, [], 0, null)).toEqual({ state: 'absent' })
    expect(skullPresenceAt(lives, [], 0)).toEqual({ state: 'absent' })
  })

  it('PIÈGE void-drop : pendant le cooldown, le crâne est sur le SOCLE, jamais au point de chute', () => {
    // chute dans le vide (instant à un pit) puis respawn au socle (vie) ; entre les deux = cooldown.
    const lives = [rest(10, -22, 9), rest(30, SOCLE.x, SOCLE.y)]
    expect(skullPresenceAt(lives, [], 20, SOCLE)).toEqual({ state: 'free', at: SOCLE, rolling: false })
    // contre-épreuve : sans socle, absent — mais JAMAIS un fantôme au pit (-22, 9).
    expect(skullPresenceAt(lives, [], 20, null)).toEqual({ state: 'absent' })
  })
})
