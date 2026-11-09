// Sonde (lecture seule) : TOUS les messages tapés par l'humain dans un transcript — messages
// « user » texte ET commandes mises en file pendant un tour (attachment queued_command,
// origin.kind = human), dédoublonnés par texte+minute. Lecture en flux.
// Usage : node human_all.mjs <fichier.jsonl> <regex> [depuis ISO] [jusqu'à ISO]
import { createReadStream } from 'node:fs'
import { createInterface } from 'node:readline'

const [, , fichier, motif = '.', depuis = '0000', jusqua = '9999'] = process.argv
const re = new RegExp(motif, 'i')
const vus = new Set()
const out = []

function texteDe(message) {
  const c = message?.content
  if (typeof c === 'string') return c
  if (!Array.isArray(c)) return ''
  return c.filter((b) => b && b.type === 'text' && typeof b.text === 'string').map((b) => b.text).join('\n')
}

const rl = createInterface({ input: createReadStream(fichier, { encoding: 'utf8' }), crlfDelay: Infinity })
for await (const ligne of rl) {
  let o
  try { o = JSON.parse(ligne) } catch { continue }
  const ts = o.timestamp ?? ''
  if (ts < depuis || ts > jusqua) continue
  let t = ''
  let source = ''
  if (o.type === 'attachment' && o.attachment?.type === 'queued_command' && o.attachment?.origin?.kind === 'human') {
    t = String(o.attachment.prompt ?? '')
    source = 'file'
  } else if (o.type === 'user' && !o.isMeta && o.toolUseResult === undefined) {
    t = texteDe(o.message)
    source = 'direct'
    if (!t || t.startsWith('<') || t.startsWith('This session is being continued')) continue
  } else continue
  if (!re.test(t)) continue
  const cle = t.trim() + '|' + ts.slice(0, 16)
  if (vus.has(cle)) continue
  vus.add(cle)
  out.push(`=== ${ts} [${source}]\n${t.length > 1200 ? t.slice(0, 1200) + ' […]' : t}\n`)
}
console.log(out.join('\n'))
console.log(`-- ${out.length} message(s)`)
