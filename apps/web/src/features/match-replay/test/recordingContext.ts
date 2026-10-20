/**
 * recordingContext.ts — LE CONTEXTE CANVAS ENREGISTREUR, partagé par les tests de rendu.
 *
 * POURQUOI C'EST POSSIBLE, ET POURQUOI ÇA NE COÛTE AUCUNE DÉPENDANCE. Le code de dessin du
 * rejeu n'INTERROGE jamais le canvas : il écrit dedans. Les seules opérations qui lisent
 * quelque chose (`createRadialGradient`, `createLinearGradient`) rendent un objet dont on
 * n'appelle que `addColorStop`. Un objet qui empile `{op, args}` suffit donc à observer tout
 * ce que le rendu produit. Ni `node-canvas`, ni `jsdom`, ni image de référence.
 *
 * CE QUE CES TESTS VÉRIFIENT : la GÉOMÉTRIE ÉMISE — combien de traits, quelles primitives,
 * dans quel ordre, avec quelles largeurs. Jamais un pixel : un test de pixels serait un test
 * d'anti-crénelage, et il tomberait à chaque changement de moteur sans rien dire du rendu.
 *
 * Extrait de `canvasRecording.test.ts` le 2026-08-15, quand l'éclair de bouche a eu besoin
 * du même outil (règle « ≤ 2 copies » : on centralise avant la troisième).
 */

/** Une opération enregistrée : le nom de la méthode (ou `set <propriété>`) et ses arguments. */
export interface CanvasOp {
  op: string
  args: unknown[]
}

/**
 * recordingContext rend un faux contexte 2D qui n'exécute rien et note tout.
 *
 * LES SEULES LECTURES À TRAITER sont les dégradés : le rendu attend un objet portant
 * `addColorStop`. On rend donc un jeton inerte, et l'appel reste dans la trace — c'est le
 * comportement qu'on veut, pas une émulation.
 */
export function recordingContext(): { ops: CanvasOp[]; ctx: CanvasRenderingContext2D } {
  const ops: CanvasOp[] = []
  const state: Record<string, unknown> = {}
  const proxy = new Proxy(
    {},
    {
      get(_t, prop) {
        if (typeof prop !== 'string') return undefined
        if (prop === 'createRadialGradient' || prop === 'createLinearGradient') {
          return (...args: unknown[]) => {
            ops.push({ op: prop, args })
            return { addColorStop: (...a: unknown[]) => ops.push({ op: 'addColorStop', args: a }) }
          }
        }
        if (prop in state) return state[prop]
        return (...args: unknown[]) => {
          ops.push({ op: prop, args })
        }
      },
      set(_t, prop, value) {
        if (typeof prop === 'string') {
          state[prop] = value
          ops.push({ op: `set ${prop}`, args: [value] })
        }
        return true
      },
    },
  )
  return { ops, ctx: proxy as unknown as CanvasRenderingContext2D }
}

/** count — combien de fois une primitive a été émise. */
export const count = (ops: CanvasOp[], op: string): number => ops.filter((o) => o.op === op).length

/** valuesOf rend les valeurs successives affectées à une propriété (lineWidth, globalAlpha…). */
export const valuesOf = (ops: CanvasOp[], prop: string): number[] =>
  ops.filter((o) => o.op === `set ${prop}`).map((o) => o.args[0] as number)

/**
 * diamondCentres rend le CENTRE de chaque losange tracé, dans l'ordre d'émission.
 *
 * POURQUOI CE HELPER EXISTE (A14, 2026-08-26). Les socles étaient des disques : un `arc` par
 * marque, et le compte des arcs suffisait à dire « combien de socles ». Depuis qu'ils sont des
 * LOSANGES, une marque n'émet plus d'`arc` du tout mais quatre segments — et compter des
 * `lineTo` ne dit plus rien de lisible. Ce helper reconstitue ce que les cas veulent réellement
 * observer : COMBIEN de lieux sont marqués, et OÙ.
 *
 * LA FORME EST CELLE DE `traceDiamond` : `moveTo` au sommet HAUT, puis droite, bas, gauche. Le
 * centre est donc à mi-hauteur du sommet et du point bas — c'est-à-dire de la 2e ligne. Un
 * `moveTo` qui n'est pas suivi de trois `lineTo` n'est pas un losange et est ignoré : le calque
 * trace aussi du texte et des images.
 */
export const diamondCentres = (ops: CanvasOp[]): { x: number; y: number }[] => {
  const out: { x: number; y: number }[] = []
  for (let i = 0; i < ops.length; i++) {
    if (ops[i].op !== 'moveTo') continue
    const suite = ops.slice(i + 1, i + 4)
    if (suite.length < 3 || suite.some((o) => o.op !== 'lineTo')) continue
    const sommet = ops[i].args as number[]
    const bas = suite[1].args as number[]
    out.push({ x: sommet[0], y: (sommet[1] + bas[1]) / 2 })
  }
  return out
}
