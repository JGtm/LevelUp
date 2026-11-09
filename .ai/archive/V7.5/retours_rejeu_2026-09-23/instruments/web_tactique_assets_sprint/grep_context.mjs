// Sonde (lecture seule) : dans un transcript, cherche une regex dans le TEXTE des messages
// (utilisateur ET assistant, blocs texte seulement, résultats d'outil exclus sauf --outils) et
// imprime un extrait autour de chaque occurrence, avec l'horodatage et le rôle. Lecture en flux.
// Usage : node grep_context.mjs <fichier.jsonl> <regex> [depuis] [jusqu'à] [rayon=300] [--outils]
import { createReadStream } from 'node:fs'
import { createInterface } from 'node:readline'

const args = process.argv.slice(2)
const outils = args.includes('--outils')
const [fichier, motif, depuis = '0000', jusqua = '9999', rayonTxt = '300'] = args.filter((a) => a !== '--outils')
const re = new RegExp(motif, 'gi')
const rayon = Number(rayonTxt)

function blocs(message) {
  const c = message?.content
  if (typeof c === 'string') return [{ kind: 'text', t: c }]
  if (!Array.isArray(c)) return []
  const out = []
  for (const b of c) {
    if (b?.type === 'text' && typeof b.text === 'string') out.push({ kind: 'text', t: b.text })
    else if (outils && b?.type === 'tool_result') {
      const t = typeof b.content === 'string' ? b.content : JSON.stringify(b.content)
      out.push({ kind: 'tool_result', t })
    } else if (outils && b?.type === 'tool_use') out.push({ kind: `tool_use:${b.name}`, t: JSON.stringify(b.input) })
  }
  return out
}

const rl = createInterface({ input: createReadStream(fichier, { encoding: 'utf8' }), crlfDelay: Infinity })
for await (const ligne of rl) {
  let o
  try { o = JSON.parse(ligne) } catch { continue }
  if (o.type !== 'user' && o.type !== 'assistant') continue
  const ts = o.timestamp ?? ''
  if (ts.slice(0, 10) < depuis || ts.slice(0, 10) > jusqua) continue
  for (const b of blocs(o.message)) {
    re.lastIndex = 0
    let m
    let dernier = -Infinity
    while ((m = re.exec(b.t)) !== null) {
      if (m.index - dernier < rayon) continue
      dernier = m.index
      const a = Math.max(0, m.index - rayon)
      const z = Math.min(b.t.length, m.index + m[0].length + rayon)
      console.log(`=== ${ts} ${o.type}/${b.kind}\n…${b.t.slice(a, z).replace(/\s+/g, ' ')}…\n`)
      if (m[0].length === 0) re.lastIndex++
    }
  }
}
