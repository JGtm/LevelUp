// Sonde (lecture seule) : extrait d'un transcript les questions posées à l'utilisateur par l'outil
// AskUserQuestion et ses RÉPONSES (résultats d'outil), dans une fenêtre de dates. Lecture en flux.
// Usage : node ask_answers.mjs <fichier.jsonl> [depuis YYYY-MM-DD] [jusqu'à YYYY-MM-DD] [regex]
import { createReadStream } from 'node:fs'
import { createInterface } from 'node:readline'

const [, , fichier, depuis = '0000', jusqua = '9999', motif = '.'] = process.argv
const re = new RegExp(motif, 'i')
const questions = new Map()

const rl = createInterface({ input: createReadStream(fichier, { encoding: 'utf8' }), crlfDelay: Infinity })
for await (const ligne of rl) {
  if (!ligne.includes('AskUserQuestion') && !ligne.includes('tool_result')) continue
  let o
  try { o = JSON.parse(ligne) } catch { continue }
  const ts = o.timestamp ?? ''
  if (ts.slice(0, 10) < depuis || ts.slice(0, 10) > jusqua) continue
  const c = o.message?.content
  if (!Array.isArray(c)) continue
  for (const b of c) {
    if (b?.type === 'tool_use' && b.name === 'AskUserQuestion') {
      questions.set(b.id, { ts, input: b.input })
    } else if (b?.type === 'tool_result' && questions.has(b.tool_use_id)) {
      const q = questions.get(b.tool_use_id)
      const rep = typeof b.content === 'string' ? b.content : JSON.stringify(b.content)
      const qtxt = JSON.stringify(q.input)
      if (!re.test(qtxt) && !re.test(rep)) continue
      console.log(`=== Q ${q.ts}\n${qtxt.slice(0, 3000)}\n--- R ${ts}\n${rep.slice(0, 3000)}\n`)
    }
  }
}
