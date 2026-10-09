/**
 * explosionFx.test.ts — CE QUE L'EXPLOSION ÉMET, ET CE QU'ELLE COÛTE.
 *
 * Trois propriétés se testent ici : elle est une FONCTION DU TEMPS (donc un retour en
 * arrière redonne exactement la même image — le contraire de l'animation à état de la
 * référence), elle est BORNÉE (item 3.4 : le coût d'image ne doit jamais faire tomber la
 * lecture), et elle ne peint que les encres du thème.
 */
import { describe, expect, it } from 'vitest'

import { drawExplosion, EXPLOSION_MS, type ExplosionInk, type ExplosionShape } from './explosionFx'
import { count, recordingContext, valuesOf } from '../test/recordingContext'

const INK: ExplosionInk = { fire: 'FIRE', core: 'CORE', smoke: 'SMOKE' }

const shape = (over: Partial<ExplosionShape> = {}): ExplosionShape => ({
  x: 200,
  y: 150,
  ageMs: 0,
  seed: 4242,
  k: 1,
  reduced: false,
  ...over,
})

function trace(over: Partial<ExplosionShape> = {}, ink: ExplosionInk = INK) {
  const { ops, ctx } = recordingContext()
  drawExplosion(ctx, shape(over), ink)
  return ops
}

describe('l’explosion de grenade', () => {
  it('est une FONCTION DU TEMPS : deux lectures du même âge donnent la même image', () => {
    // C'est LA différence avec la référence Csstat, qui anime à état : ici la lecture
    // recule et saute au curseur, une explosion à état afficherait n'importe quoi.
    const a = JSON.stringify(trace({ ageMs: 420 }))
    const b = JSON.stringify(trace({ ageMs: 420 }))
    expect(a).toBe(b)
  })

  it('deux germes différents donnent deux images différentes', () => {
    const a = JSON.stringify(trace({ ageMs: 300, seed: 1 }))
    const b = JSON.stringify(trace({ ageMs: 300, seed: 2 }))
    expect(a).not.toBe(b)
  })

  it('ne dessine RIEN hors de sa fenêtre : une explosion finie ne coûte pas une opération', () => {
    expect(trace({ ageMs: EXPLOSION_MS + 1 })).toEqual([])
    expect(trace({ ageMs: -1 })).toEqual([])
  })

  it('sans encre de feu, rien n’est peint — jamais une couleur inventée', () => {
    expect(drawAndCount({ fire: '', core: 'CORE', smoke: 'SMOKE' })).toBe(0)
  })

  it('ne peint QUE les trois encres du thème', () => {
    for (const age of [0, 200, 700, 1_300]) {
      const ops = trace({ ageMs: age })
      const couleurs = new Set(
        ops
          .filter((o) => o.op === 'set strokeStyle' || o.op === 'set fillStyle' || o.op === 'addColorStop')
          .map((o) => (o.op === 'addColorStop' ? o.args[1] : o.args[0]))
          .filter((c): c is string => typeof c === 'string' && !c.startsWith('[object')),
      )
      for (const c of couleurs) {
        expect(['FIRE', 'CORE', 'SMOKE', 'transparent'], `âge ${age} peint ${c}`).toContain(c)
      }
    }
  })

  it('BORNE le coût d’image : 19 dégradés et 228 primitives au pire (item 3.4)', () => {
    // MESURE, pas estimation. Les « opérations lourdes » sont les dégradés radiaux : ce sont
    // elles qui coûtent, pas les traits. Le maximum est cherché sur toute la timeline, et
    // il est FIGÉ ici — c'est ce qui empêche une explosion enrichie de passer sans qu'on
    // s'en aperçoive.
    let pireGradients = 0
    let pireTotal = 0
    for (let age = 0; age <= EXPLOSION_MS; age += 20) {
      const ops = trace({ ageMs: age })
      pireGradients = Math.max(pireGradients, count(ops, 'createRadialGradient'))
      pireTotal = Math.max(pireTotal, ops.length)
    }
    expect(pireGradients).toBe(19)
    // 227 avant le 2026-08-15 : le plateau du flash ajoute UNE borne de couleur à son unique
    // dégradé, et rien d'autre. Le nombre de dégradés, lui, n'a pas bougé.
    expect(pireTotal).toBe(228)
  })

  it('ne compose JAMAIS en additif : c’est ce qui la rendait invisible sur le thème clair', () => {
    // RÉGRESSION VERROUILLÉE (même verrou que l'éclair de bouche). Les quatre phases chaudes
    // composaient en `lighter` : sur un fond quasi blanc, `dst + src·α` écrête à 255 et ne
    // change AUCUN pixel — flash, boule de feu, onde de choc et éclats étaient absents du
    // thème clair, poussière comprise dans le décompte, il ne restait qu'elle. Le comptage
    // de pixels vit en Chromium réel (e2e/replay-explosion-raster.spec.ts) ; ici on interdit
    // simplement le retour de l'opérateur.
    for (const age of [0, 40, 180, 320, 700, 900, 1_300]) {
      expect(valuesOf(trace({ ageMs: age }), 'globalCompositeOperation'), `âge ${age}`).not.toContain(
        'lighter',
      )
    }
  })

  it('s’éteint : il reste moins d’opérations à la fin qu’au début', () => {
    // Les deux âges sont RELATIFS à la timeline : la durée a déjà changé une fois
    // (1,4 -> 2,4 s le 2026-08-17), un âge en dur aurait cessé d'être « la fin ».
    expect(trace({ ageMs: EXPLOSION_MS - 50 }).length).toBeLessThan(trace({ ageMs: 100 }).length)
  })

  it('sous mouvement réduit : une empreinte fixe, aucune particule', () => {
    const ops = trace({ ageMs: 500, reduced: true })
    expect(count(ops, 'createRadialGradient')).toBe(0)
    expect(count(ops, 'arc')).toBe(2)
    // Et elle ne varie pas avec l'âge : rien ne bouge, c'est le contrat.
    expect(JSON.stringify(trace({ ageMs: 100, reduced: true }))).toBe(
      JSON.stringify(trace({ ageMs: 1_200, reduced: true })),
    )
  })

  it('encadre son rendu d’un save/restore — aucun état ne fuit sur le calque suivant', () => {
    const ops = trace({ ageMs: 300 })
    expect(ops[0]?.op).toBe('save')
    expect(ops[ops.length - 1]?.op).toBe('restore')
  })

  it('le flash ne dure qu’un instant : il n’est plus là au tiers de la timeline', () => {
    // LE FLASH SE COMPTE, IL NE SE DEVINE PAS. Comparer le NOMBRE TOTAL de dégradés à deux
    // instants ne mesurait pas le flash mais l'essaim entier — et l'étirement de la timeline
    // (2026-08-17) l'a montré : à 500 ms il reste désormais PLUS de braises vivantes qu'au
    // départ de dégradés, alors que le flash, lui, est bien éteint. Sa signature est le
    // PLATEAU : il est le seul dégradé à trois bornes de couleur, les autres en ont deux.
    const flashes = (age: number) => {
      const ops = trace({ ageMs: age })
      return count(ops, 'addColorStop') - 2 * count(ops, 'createRadialGradient')
    }
    expect(flashes(0)).toBe(1)
    expect(flashes(EXPLOSION_MS / 3)).toBe(0)
  })
})

function drawAndCount(ink: ExplosionInk): number {
  const { ops, ctx } = recordingContext()
  drawExplosion(ctx, shape({ ageMs: 100 }), ink)
  return ops.length
}
