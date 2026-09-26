// Sonde (lecture seule) : distribution des DURÉES d'intervalle par genre d'état (s), un document à la fois.
import { readdirSync, readFileSync } from 'node:fs'
import { join } from 'node:path'
const DIR = 'C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/replays/halo_infinite'
const par = new Map()
for (const f of readdirSync(DIR).filter((x) => /^[0-9a-f]{8}\.json$/.test(x))) {
  let d = JSON.parse(readFileSync(join(DIR, f), 'utf8'))
  const dt = d.frameIntervalMs || 100
  for (const s of d.stances ?? []) {
    if (!par.has(s.kind)) par.set(s.kind, [])
    par.get(s.kind).push(((s.t1 - s.t0 + 1) * dt) / 1000)
  }
  d = null
}
const q = (a, p) => a[Math.min(a.length - 1, Math.floor(p * (a.length - 1)))]
for (const [k, a] of par) {
  a.sort((x, y) => x - y)
  console.log(`${k.padEnd(12)} n=${String(a.length).padStart(6)}  min=${a[0].toFixed(1)}  p10=${q(a, 0.1).toFixed(1)}  med=${q(a, 0.5).toFixed(1)}  p90=${q(a, 0.9).toFixed(1)}  max=${a[a.length - 1].toFixed(1)} s`)
}
