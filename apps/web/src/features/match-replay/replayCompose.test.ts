/**
 * replayCompose.test.ts — L'ORDRE DE LA SCÈNE ET SES BASCULES, au contexte enregistreur.
 *
 * CE QUE CES CAS PROTÈGENT, ET QUE RIEN NE PROTÉGEAIT (registre 2026-09-05, M3) : la
 * composition du rejeu vivait dans une fonction de deux cents lignes du canvas, et le SEUL
 * test qui nommait ce fichier comptait ses lignes. Une inversion de calques — les morts sous
 * le sol, la chaleur par-dessus les joueurs — ou un interrupteur du tiroir qui cesse de
 * couper ne se voyait qu'à l'oeil, sur un rejeu, par quelqu'un qui savait quoi chercher.
 *
 * LE CONTEXTE ENREGISTREUR EST LA PREUVE. Chaque peintre témoin écrit son nom DANS le
 * contexte (`fillText(id, frame, dpr)`) : le journal du contexte dit donc l'ordre RÉEL des
 * gestes, pas seulement ce que la fonction prétend rendre — et il dit du même coup que
 * l'image et la densité de pixels sont bien transmises à chaque calque.
 */
import { describe, expect, it } from 'vitest'

import {
  composeScene,
  LAYER_ORDER,
  sceneLayers,
  type LayerPaint,
  type ReplayLayerId,
  type ReplayScene,
} from './replayCompose'

/** Le contexte témoin : il n'enregistre que ce que les calques lui écrivent. */
function contexteEnregistreur(): { ctx: CanvasRenderingContext2D; journal: string[] } {
  const journal: string[] = []
  const ctx = {
    fillText: (texte: string, x: number, y: number) => journal.push(`${texte}@${x}/${y}`),
  } as unknown as CanvasRenderingContext2D
  return { ctx, journal }
}

/** Un peintre témoin : il écrit son nom, l'image et la densité dans le contexte. */
function temoin(id: ReplayLayerId): LayerPaint {
  return (ctx, frame, dpr) => ctx.fillText(id, frame, dpr)
}

const TOUT_ALLUME: ReplayScene['toggles'] = {
  zones: true,
  shotFx: true,
  placements: true,
  killFx: true,
}

const TOUT_A_PEINDRE: ReplayScene['has'] = {
  background: true,
  floor: true,
  heat: true,
  zoneNames: true,
  objectivesCooked: true,
  projectiles: true,
  placements: true,
  fireMarks: true,
  shotFx: true,
  grenades: true,
  zoneStates: true,
  objectivePulses: true,
  killFx: true,
}

function scene(over: Partial<ReplayScene> = {}): ReplayScene {
  return {
    toggles: { ...TOUT_ALLUME },
    has: { ...TOUT_A_PEINDRE },
    paint: Object.fromEntries(LAYER_ORDER.map((id) => [id, temoin(id)])) as ReplayScene['paint'],
    ...over,
  }
}

/** Les noms des calques réellement peints, dans l'ordre où le CONTEXTE les a reçus. */
function peints(s: ReplayScene, frame = 42, dpr = 2): string[] {
  const { ctx, journal } = contexteEnregistreur()
  const rendus = composeScene(ctx, sceneLayers(s), frame, dpr)
  // Le journal du contexte et la liste rendue doivent dire la MÊME chose : sans quoi la
  // valeur de retour serait une déclaration d'intention et non un constat.
  expect(journal.map((e) => e.split('@')[0])).toEqual(rendus)
  return rendus
}

describe('sceneLayers — l’ordre de la scène', () => {
  it('rend les vingt-cinq calques, dans l’ordre déclaré, sans doublon ni oubli', () => {
    const layers = sceneLayers(scene())
    expect(layers.map((l) => l.id)).toEqual([...LAYER_ORDER])
    expect(new Set(LAYER_ORDER).size).toBe(LAYER_ORDER.length)
  })

  it('PEINT du fond vers le sujet : le sol d’abord, les morts en dernier', () => {
    const ordre = peints(scene({ has: { ...TOUT_A_PEINDRE, floor: false } }))
    expect(ordre[0]).toBe('fond-carte')
    expect(ordre[ordre.length - 1]).toBe('morts')
    // Les repères qui décident de la lisibilité : le terrain sous les joueurs, les joueurs
    // sous les événements.
    expect(ordre.indexOf('chaleur')).toBeLessThan(ordre.indexOf('trajectoires'))
    expect(ordre.indexOf('zones-nommees')).toBeLessThan(ordre.indexOf('trajectoires'))
    expect(ordre.indexOf('vehicules')).toBeLessThan(ordre.indexOf('trajectoires'))
    expect(ordre.indexOf('trajectoires')).toBeLessThan(ordre.indexOf('tirs'))
    expect(ordre.indexOf('tirs')).toBeLessThan(ordre.indexOf('morts'))
  })

  it('transmet À CHAQUE calque l’image courante et la densité de pixels', () => {
    const { ctx, journal } = contexteEnregistreur()
    composeScene(ctx, sceneLayers(scene()), 1_337, 3)
    expect(journal.every((e) => e.endsWith('@1337/3'))).toBe(true)
  })
})

describe('sceneLayers — les deux replis du sol, exclusifs', () => {
  it('pose l’IMAGE calée quand la carte en a une, et pas les props Forge', () => {
    const ordre = peints(scene())
    expect(ordre).toContain('fond-carte')
    expect(ordre).not.toContain('sol-forge')
  })

  it('retombe sur les props Forge quand l’image manque', () => {
    const ordre = peints(scene({ has: { ...TOUT_A_PEINDRE, background: false } }))
    expect(ordre).toContain('sol-forge')
    expect(ordre).not.toContain('fond-carte')
  })

  it('ne pose AUCUN sol quand ni l’image ni les props n’existent', () => {
    const ordre = peints(scene({ has: { ...TOUT_A_PEINDRE, background: false, floor: false } }))
    expect(ordre).not.toContain('fond-carte')
    expect(ordre).not.toContain('sol-forge')
  })
})

describe('sceneLayers — chaque interrupteur du tiroir coupe SES calques, et eux seuls', () => {
  /** Ce que chaque bascule doit retirer de la scène, exactement. */
  const COUPE: Array<[keyof ReplayScene['toggles'], ReplayLayerId[]]> = [
    ['zones', ['zones-nommees']],
    // Un seul geste éteint l'éclair de bouche ET le « ! » du tireur : c'est le même événement.
    ['shotFx', ['marques-de-tir', 'tirs']],
    ['placements', ['poses-equipement']],
    ['killFx', ['morts']],
  ]

  it.each(COUPE)('« %s » coupé retire exactement %s', (bascule, attendus) => {
    const complet = peints(scene())
    const coupe = peints(scene({ toggles: { ...TOUT_ALLUME, [bascule]: false } }))
    expect(complet.filter((id) => !coupe.includes(id))).toEqual(attendus)
  })

  it('LA MESURE NE S’ÉTEINT PAS AVEC LE DESSIN : couper les morts ne touche pas la chaleur', () => {
    // `killFx` alimente la lecture « éliminations » de la carte de chaleur, qui n'est pas un
    // effet : l'interrupteur du tiroir n'éteint que le calque des morts.
    const coupe = peints(scene({ toggles: { ...TOUT_ALLUME, killFx: false } }))
    expect(coupe).toContain('chaleur')
    expect(coupe).not.toContain('morts')
  })
})

describe('sceneLayers — un calque sans matière ne s’ouvre pas', () => {
  /** Ce que chaque absence de matière doit retirer, exactement. */
  const SANS: Array<[keyof ReplayScene['has'], ReplayLayerId[]]> = [
    ['heat', ['chaleur']],
    ['zoneNames', ['zones-nommees']],
    ['objectivesCooked', ['objectifs-cuits']],
    ['projectiles', ['projectiles']],
    ['placements', ['poses-equipement']],
    ['fireMarks', ['marques-de-tir']],
    ['shotFx', ['tirs']],
    ['grenades', ['grenades']],
    ['zoneStates', ['etat-zones']],
    ['objectivePulses', ['pulses-objectif']],
    ['killFx', ['morts']],
  ]

  it.each(SANS)('sans %s, %s ne se peint pas', (matiere, attendus) => {
    const complet = peints(scene())
    const sans = peints(scene({ has: { ...TOUT_A_PEINDRE, [matiere]: false } }))
    expect(complet.filter((id) => !sans.includes(id))).toEqual(attendus)
  })

  it('les calques SANS condition se peignent toujours, même sur une scène vide de matière', () => {
    const vide = Object.fromEntries(
      Object.keys(TOUT_A_PEINDRE).map((k) => [k, false]),
    ) as unknown as ReplayScene['has']
    expect(peints(scene({ has: vide }))).toEqual([
      'socles-armes',
      'armes-au-sol',
      'vehicules',
      'trajectoires',
      'gestes-capacite',
      'fin-de-vol',
      'drapeaux',
      'objets-objectif',
      'couronne-vip',
      'crane-porte',
      'bombe-portee',
      'deflagration',
    ])
  })
})

describe('composeScene — la boucle', () => {
  it('ne peint RIEN quand tout est éteint, et ne lève pas', () => {
    const { ctx, journal } = contexteEnregistreur()
    const eteints = sceneLayers(scene()).map((l) => ({ ...l, on: false }))
    expect(composeScene(ctx, eteints, 0, 1)).toEqual([])
    expect(journal).toEqual([])
  })

  it('respecte l’ordre de la LISTE qu’on lui donne, pas celui de la déclaration', () => {
    // C'est ce qui permet à `sceneLayers` d'être la seule source de l'ordre : la boucle, elle,
    // n'a aucune opinion.
    const { ctx, journal } = contexteEnregistreur()
    const inverse = [...sceneLayers(scene())].reverse()
    composeScene(ctx, inverse, 7, 1)
    expect(journal[0]).toBe('morts@7/1')
  })
})
