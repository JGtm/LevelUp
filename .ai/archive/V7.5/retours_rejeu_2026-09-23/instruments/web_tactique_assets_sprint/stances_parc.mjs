// Sonde (lecture seule) : fréquence des états de mouvement publiés (`stances[]`) sur les
// documents de rejeu du cache, UN DOCUMENT À LA FOIS en mémoire (lecture séquentielle).
// Rend : documents par schéma, documents portant au moins un intervalle, intervalles et
// durée cumulée par genre, et la part de la durée « étiquetée » par genre.
import { readdirSync, readFileSync } from 'node:fs'
import { join } from 'node:path'

const DIR = 'C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/replays/halo_infinite'
const fichiers = readdirSync(DIR).filter((f) => /^[0-9a-f]{8}\.json$/.test(f)).sort()
const parSchema = new Map()
const parGenre = new Map()
let avecStances = 0
let avecSprint = 0
let docs68 = 0
for (const f of fichiers) {
  let d
  try { d = JSON.parse(readFileSync(join(DIR, f), 'utf8')) } catch { continue }
  parSchema.set(d.schemaVersion, (parSchema.get(d.schemaVersion) ?? 0) + 1)
  if (d.schemaVersion === 68) docs68++
  const st = Array.isArray(d.stances) ? d.stances : []
  if (st.length > 0) avecStances++
  if (st.some((s) => s.kind === 'sprint')) avecSprint++
  const dt = d.frameIntervalMs || 100
  for (const s of st) {
    const g = parGenre.get(s.kind) ?? { n: 0, ms: 0 }
    g.n++
    g.ms += Math.max(0, s.t1 - s.t0 + 1) * dt
    parGenre.set(s.kind, g)
  }
  d = null
}
const total = [...parGenre.values()].reduce((a, g) => a + g.ms, 0)
console.log(`documents lus : ${fichiers.length} ; par schéma : ${[...parSchema].map(([k, v]) => `${k}:${v}`).join(' ')}`)
console.log(`documents avec au moins un intervalle : ${avecStances} ; avec au moins un sprint : ${avecSprint} (schéma 68 : ${docs68})`)
for (const [k, g] of [...parGenre].sort((a, b) => b[1].ms - a[1].ms)) {
  console.log(`  ${k.padEnd(12)} intervalles=${String(g.n).padStart(6)}  durée=${(g.ms / 1000).toFixed(0).padStart(7)} s  part=${((100 * g.ms) / total).toFixed(1)} %  médiane≈${(g.ms / g.n / 1000).toFixed(2)} s/intervalle (moyenne)`)
}
