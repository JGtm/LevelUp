/**
 * replayColumns.ts — décodeur de la forme EN COLONNES du document de rejeu.
 *
 * L'encodage est un encodage de TRANSPORT : le serveur (`replayview.EncodeColumns`) réécrit le
 * JSON du document servi quand la requête demande `encoding=columns`, et ce module est le SEUL
 * endroit qui le défait, avant `validateReplayDocument` et `normalizeReplayDocument`. Le reste de
 * l'application voit les mêmes objets qu'avec la forme normale.
 *
 * # Forme (contrat complet : en-tête de `replayview/columns.go`)
 *
 *  - la racine porte `$enc: 1` ; ce module ne décode que si `$enc` est présent, puis le retire ;
 *  - une liste d'objets devient `{ $c: N, k: [clés], c: [colonnes] }`, une liste de n-uplets
 *    numériques `{ $c: N, a: arité, c: [colonnes] }` ; l'opération est récursive ;
 *  - une colonne est un tableau nu de valeurs, ou `{ v, s?, d?, a? }` : `v` ne porte que les
 *    valeurs PRÉSENTES ; `d = 1` dit que `v` porte des différences successives ; `s` est le nombre de
 *    décimales (les entiers de `v` se divisent par 10^s) ; `a` dit où la valeur manque (longueurs
 *    alternées présent / absent, ou chaîne de 0 et de 1). Une valeur ABSENTE n'est ni 0 ni `null` :
 *    la clé n'existe pas dans l'objet rendu.
 *
 * La division par 10^s est exacte (entier sûr divisé par une puissance de dix représentable : le
 * résultat est le double le plus proche du jeton décimal d'origine) : les valeurs reviennent
 * identiques à celles de la forme normale, à `-0` près, qui devient `0`.
 */

/** Version de la forme en colonnes que ce décodeur sait lire. */
const COLUMNS_FORM_VERSION = 1

type Obj = Record<string, unknown>

function isObj(v: unknown): v is Obj {
  return v !== null && typeof v === 'object' && !Array.isArray(v)
}

/**
 * decodeReplayColumns rend le document de transport sous sa forme normale. Un document sans
 * `$enc` est rendu tel quel (même référence) : la forme normale reste servie telle quelle quand
 * l'encodeur du serveur refuse ou n'est pas branché.
 */
export function decodeReplayColumns<T>(raw: T): T {
  if (!isObj(raw) || !('$enc' in raw)) return raw
  const { $enc, ...rest } = raw
  if ($enc !== COLUMNS_FORM_VERSION) {
    throw new Error(`replayColumns: forme en colonnes inconnue ($enc = ${String($enc)})`)
  }
  return decodeValue(rest) as T
}

function decodeValue(v: unknown): unknown {
  if (Array.isArray(v)) {
    // Même règle que l'encodeur : une liste dont le premier élément n'est ni objet ni liste est intacte.
    if (v.length === 0 || typeof v[0] !== 'object' || v[0] === null) return v
    return v.map(decodeValue)
  }
  if (isObj(v)) {
    if ('$c' in v) return decodeTable(v)
    const out: Obj = {}
    for (const k in v) out[k] = decodeValue(v[k])
    return out
  }
  return v
}

interface DecodedColumn {
  vals: unknown[]
  /** null : toutes les lignes portent une valeur. */
  present: Uint8Array | null
}

function decodeColumn(col: unknown, rows: number): DecodedColumn {
  if (Array.isArray(col)) return { vals: col.map(decodeValue), present: null }
  if (!isObj(col) || !Array.isArray(col.v)) throw new Error('replayColumns: colonne mal formée')
  const scale = typeof col.s === 'number' ? col.s : 0
  const delta = col.d === 1
  let vals: unknown[] = col.v
  if (delta) {
    const acc: number[] = new Array(vals.length)
    let sum = 0
    for (let i = 0; i < vals.length; i++) {
      sum += vals[i] as number
      acc[i] = sum
    }
    vals = acc
  }
  if (scale > 0) {
    const div = 10 ** scale
    const out: number[] = new Array(vals.length)
    for (let i = 0; i < vals.length; i++) out[i] = (vals[i] as number) / div
    vals = out
  } else if (!delta) {
    vals = vals.map(decodeValue)
  }
  return { vals, present: col.a === undefined ? null : decodePresence(col.a, rows) }
}

function decodePresence(a: unknown, rows: number): Uint8Array {
  const present = new Uint8Array(rows)
  if (typeof a === 'string') {
    for (let i = 0; i < rows; i++) present[i] = a.charCodeAt(i) - 48
    return present
  }
  if (!Array.isArray(a)) throw new Error('replayColumns: absence mal formée')
  let pos = 0
  let on = true
  for (const run of a as number[]) {
    if (on) present.fill(1, pos, pos + run)
    pos += run
    on = !on
  }
  return present
}

function decodeTable(t: Obj): unknown[] {
  const rows = t.$c as number
  if (!Number.isInteger(rows) || rows < 0 || !Array.isArray(t.c)) throw new Error('replayColumns: table mal formée')
  const cols = t.c.map((c) => decodeColumn(c, rows))
  const out: unknown[] = new Array(rows)
  if (t.a !== undefined) {
    for (let i = 0; i < rows; i++) {
      const row: unknown[] = new Array(cols.length)
      for (let j = 0; j < cols.length; j++) row[j] = cols[j].vals[i]
      out[i] = row
    }
    return out
  }
  const keys = t.k as string[]
  const next = new Array<number>(cols.length).fill(0)
  for (let i = 0; i < rows; i++) {
    const row: Obj = {}
    for (let j = 0; j < cols.length; j++) {
      const c = cols[j]
      if (c.present === null) row[keys[j]] = c.vals[i]
      else if (c.present[i]) row[keys[j]] = c.vals[next[j]++]
    }
    out[i] = row
  }
  return out
}
