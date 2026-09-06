/**
 * muzzleFlash.test.ts — CE QUE L'ÉCLAIR DE BOUCHE ÉMET, vérifié sans navigateur (contexte
 * enregistreur, cf. test/recordingContext.ts).
 *
 * Trois propriétés se testent ici et nulle part ailleurs : chaque famille a bien SA forme,
 * la couleur vient de la TEINTE de l'arme et d'aucune autre source, et le tir dont le regard
 * n'est pas lisible ne dessine AUCUNE direction.
 */
import { describe, expect, it } from 'vitest'

import type { FxInk, FxTint } from './fxInk'
import { drawMuzzleFlash, type MuzzleShape } from './muzzleFlash'
import type { ShotFamily } from './shotEffects'
import { count, recordingContext, valuesOf } from '../test/recordingContext'

/** Encres de test : une couleur reconnaissable par teinte, pour tracer d'où vient la peinture. */
const INK: FxInk = {
  tint: {
    kinetic: 'TINT-kinetic',
    plasma_cool: 'TINT-plasma_cool',
    plasma_hot: 'TINT-plasma_hot',
    forerunner: 'TINT-forerunner',
    electric: 'TINT-electric',
    needle: 'TINT-needle',
    blast: 'TINT-blast',
    neutral: 'TINT-neutral',
  },
  core: 'CORE',
}

const shape = (over: Partial<MuzzleShape> = {}): MuzzleShape => ({
  x: 100,
  y: 100,
  angle: 0,
  fade: 1,
  reduced: false,
  seed: 3,
  k: 1,
  ...over,
})

function trace(fam: ShotFamily, tint: FxTint = 'kinetic', over: Partial<MuzzleShape> = {}) {
  const { ops, ctx } = recordingContext()
  drawMuzzleFlash(ctx, fam, tint, shape(over), INK)
  return ops
}

const DRAWN: ShotFamily[] = ['ballistic', 'plasma', 'light', 'shock', 'explosive', 'needles', 'plain']

describe('l’éclair de bouche', () => {
  it('donne une signature DISTINCTE à chaque famille dessinée', () => {
    // LA SIGNATURE PORTE L'ÉTIREMENT ET LES RAYONS, et c'est une MESURE, pas une commodité :
    // la poudre et le plasma émettent exactement les mêmes primitives, dans le même ordre.
    // Ce qui les sépare est la GÉOMÉTRIE de la bouffée — la flamme est étirée dans l'axe
    // (1,7 / 0,45, valeurs de la référence Csstat), le plasma est rond et grandit.
    const sigs = new Map<string, ShotFamily[]>()
    for (const f of DRAWN) {
      const ops = trace(f)
      const sig = JSON.stringify([
        ops.map((o) => o.op),
        valuesOf(ops, 'lineWidth'),
        ops.filter((o) => o.op === 'scale' || o.op === 'createRadialGradient').map((o) => o.args),
      ])
      sigs.set(sig, [...(sigs.get(sig) ?? []), f])
    }
    expect([...sigs.values()].filter((fs) => fs.length > 1)).toEqual([])
  })

  it('encadre chaque effet d’un save/restore — aucun état ne fuit sur le calque suivant', () => {
    for (const f of DRAWN) {
      const ops = trace(f)
      expect(ops[0]?.op, f).toBe('save')
      expect(ops[ops.length - 1]?.op, f).toBe('restore')
    }
  })

  it('ne compose PAS en additif — l’additif est invisible sur un fond clair (mesuré)', () => {
    // RÉGRESSION VERROUILLÉE. L'éclair composait en `lighter` (référence Csstat, carte
    // noire). Rastérisation Chromium du 2026-08-15, thème clair, sol `--card` : écart
    // maximal de 3 valeurs sur 255 et ZÉRO pixel modifié d'au moins 8 — l'éclair y était
    // invisible par construction, à toute taille. La preuve chiffrée, elle, vit dans le
    // garde-rail de rastérisation (e2e/replay-muzzle-raster.spec.ts) : un contexte
    // enregistreur ne peut pas voir une couleur qui sature.
    for (const f of DRAWN) {
      expect(valuesOf(trace(f), 'globalCompositeOperation'), f).not.toContain('lighter')
    }
  })

  it('ne peint QUE la teinte de l’arme et le cœur — aucune couleur de joueur n’y entre', () => {
    // Correction utilisateur du 2026-08-15 : l'effet de tir ne porte pas la couleur du
    // tireur. Ce test le VERROUILLE — la fonction ne reçoit d'ailleurs aucune autre couleur.
    for (const f of DRAWN) {
      const peintes = new Set([
        ...valuesOf(trace(f, 'plasma_hot'), 'strokeStyle'),
        ...valuesOf(trace(f, 'plasma_hot'), 'fillStyle'),
        ...trace(f, 'plasma_hot')
          .filter((o) => o.op === 'addColorStop')
          .map((o) => o.args[1]),
      ])
      for (const c of peintes) {
        if (typeof c !== 'string' || c === 'transparent') continue
        // Les dégradés sont des jetons inertes du contexte enregistreur (objets, pas des
        // chaînes) : seules les couleurs littérales sont examinées.
        if (c.startsWith('[object')) continue
        expect(['TINT-plasma_hot', 'CORE'], `${f} peint ${c}`).toContain(c)
      }
    }
  })

  it('une teinte vide retombe sur le neutre du thème, jamais sur une voisine', () => {
    const ink: FxInk = { ...INK, tint: { ...INK.tint, forerunner: '' } }
    const { ops, ctx } = recordingContext()
    drawMuzzleFlash(ctx, 'ballistic', 'forerunner', shape(), ink)
    expect(valuesOf(ops, 'fillStyle')).toContain('TINT-neutral')
  })

  it('sans aucune encre, rien n’est dessiné — pas une couleur inventée', () => {
    const vide: FxInk = { tint: { ...INK.tint, kinetic: '', neutral: '' }, core: '' }
    const { ops, ctx } = recordingContext()
    drawMuzzleFlash(ctx, 'ballistic', 'kinetic', shape(), vide)
    expect(ops).toEqual([])
  })

  it('sans regard lisible : une bouffée RONDE, aucune direction inventée', () => {
    const ops = trace('ballistic', 'kinetic', { angle: null })
    // La bouffée ne tourne pas et ne s'étire pas : ni rotate, ni scale.
    expect(count(ops, 'rotate')).toBe(0)
    expect(count(ops, 'scale')).toBe(0)
    expect(count(ops, 'arc')).toBe(2) // halo + cœur
  })

  it('la MÊLÉE ne devrait jamais arriver ici, et si elle arrive elle ne ment pas', () => {
    // Le filtrage est en amont (buildShotFx). Le rendu de secours est la bouffée neutre,
    // jamais une flamme orientée qui affirmerait un tir.
    const ops = trace('melee')
    expect(count(ops, 'rotate')).toBe(0)
  })

  it('s’éteint avec l’âge : l’opacité d’un éclair vieux est plus basse', () => {
    const jeune = Math.max(...valuesOf(trace('ballistic', 'kinetic', { fade: 1 }), 'globalAlpha'))
    const vieux = Math.max(...valuesOf(trace('ballistic', 'kinetic', { fade: 0.2 }), 'globalAlpha'))
    expect(vieux).toBeLessThan(jeune)
  })

  it('sous mouvement réduit, l’intensité ne dépend plus de l’âge', () => {
    const a = valuesOf(trace('ballistic', 'kinetic', { fade: 1, reduced: true }), 'globalAlpha')
    const b = valuesOf(trace('ballistic', 'kinetic', { fade: 0.2, reduced: true }), 'globalAlpha')
    expect(a).toEqual(b)
  })

  it('l’éclair se pose DEVANT le marqueur, dans l’axe du regard', () => {
    const ops = trace('ballistic', 'kinetic', { angle: 0 })
    const t = ops.find((o) => o.op === 'translate')
    expect(t?.args[0] as number).toBeGreaterThan(100) // décalé vers +x, l'axe visé
    expect(t?.args[1]).toBe(100)
  })

  it('la lumière est un rai continu : deux passes sur le même segment', () => {
    const ops = trace('light', 'forerunner')
    expect(count(ops, 'lineTo')).toBe(1)
    expect(count(ops, 'stroke')).toBe(2)
  })

  it('l’arc électrique est BRISÉ : une polyligne, jamais un segment droit', () => {
    const ops = trace('shock', 'electric')
    expect(count(ops, 'lineTo')).toBe(5)
    const ys = ops.filter((o) => o.op === 'lineTo').map((o) => o.args[1] as number)
    expect(new Set(ys).size).toBeGreaterThan(1)
  })

  it('les aiguilles sont une GERBE : cinq brins qui s’écartent', () => {
    const ops = trace('needles', 'needle')
    expect(count(ops, 'moveTo')).toBe(5)
    expect(count(ops, 'stroke')).toBe(5)
  })

  it('l’explosif ouvre son onde AU CANON : le film ne date aucun impact', () => {
    const ops = trace('explosive', 'blast', { angle: 0, fade: 0.5 })
    const anneaux = ops.filter((o) => o.op === 'arc' && (o.args[2] as number) > 0)
    // L'anneau est centré sur la bouche (x > 100 dans l'axe), pas au bout d'une portée.
    const centre = anneaux[anneaux.length - 1]
    expect(centre?.args[0] as number).toBeLessThan(120)
  })
})
