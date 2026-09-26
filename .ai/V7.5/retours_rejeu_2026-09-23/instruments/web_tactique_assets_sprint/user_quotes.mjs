// Sonde (lecture seule) : extrait d'un transcript de session Claude Code les messages TAPÉS par
// l'utilisateur (type "user", contenu texte, hors résultats d'outil et hors messages injectés),
// filtrés par mots-clés et par fenêtre de dates. Lecture en flux, ligne à ligne.
// Usage : node user_quotes.mjs <fichier.jsonl> <regex> [depuis YYYY-MM-DD] [jusqu'à YYYY-MM-DD]
import { createReadStream } from 'node:fs'
import { createInterface } from 'node:readline'

const [, , fichier, motif, depuis = '0000', jusqua = '9999'] = process.argv
const re = new RegExp(motif, 'i')

function texteDe(message) {
  const c = message?.content
  if (typeof c === 'string') return c
  if (!Array.isArray(c)) return ''
  return c.filter((b) => b && b.type === 'text' && typeof b.text === 'string').map((b) => b.text).join('\n')
}

const rl = createInterface({ input: createReadStream(fichier, { encoding: 'utf8' }), crlfDelay: Infinity })
let n = 0
for await (const ligne of rl) {
  if (!ligne.includes('"type":"user"')) continue
  let o
  try { o = JSON.parse(ligne) } catch { continue }
  if (o.type !== 'user' || o.isMeta || o.toolUseResult !== undefined) continue
  const ts = o.timestamp ?? ''
  if (ts.slice(0, 10) < depuis || ts.slice(0, 10) > jusqua) continue
  const t = texteDe(o.message)
  if (!t || t.startsWith('<') || /tool_result/.test(t)) continue
  if (!re.test(t)) continue
  n++
  const court = t.length > 1500 ? t.slice(0, 1500) + ' […]' : t
  console.log(`=== ${ts} (uuid ${o.uuid})\n${court}\n`)
}
console.log(`-- ${n} message(s)`)
