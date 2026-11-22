/**
 * Banc de test du relief : un contexte 2D qui n'exécute rien et note chaque appel et chaque affectation,
 * pour lire la géométrie émise sans pixel. Aucune dépendance au rejeu (le dossier est de `lib/`).
 */

/** Une opération enregistrée : le nom de la méthode (ou `set <propriété>`) et ses arguments. */
export interface RecordedOp {
  op: string
  args: unknown[]
}

/** recordingContext rend un faux contexte 2D et la liste des opérations qu'il a reçues. */
export function recordingContext(): { ops: RecordedOp[]; ctx: CanvasRenderingContext2D } {
  const ops: RecordedOp[] = []
  const state: Record<string, unknown> = {}
  const proxy = new Proxy(
    {},
    {
      get(_t, prop) {
        if (typeof prop !== 'string') return undefined
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
