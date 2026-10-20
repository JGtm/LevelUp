// sweep_vehicle_weapon_tags.mjs — L'INSTRUMENT QUI ECRIT `vehicle_weapon_tags_observed.json`.
//
// Retours du rejeu 2026-09-23. Pose au lot L1.5 (revue RR-L1-06) a cote du garde-rail CLIENT des
// tables d'armes de vehicule ; DEPLACE au lot M4a (sans changer une ligne de mesure) a cote du
// garde-rail GO du registre `config/titles/halo_infinite/mappings/vehicle_weapons.toml`
// (`internal/games/mappings/loader_vehicle_weapons_test.go`), quand les tables client ont ete
// remplacees par ce registre publie dans le document.
//
// LECTURE SEULE : il lit les documents de rejeu publies (`<depot>/data/cache/replays/halo_infinite`,
// fichiers `*.json` hors `*.derived.json`), n'ouvre aucune base, n'ecrit que la fixture.
//
// CE QU'IL COMPTE, par tag d'arme de VEHICULE (`Shot.w` au gabarit `0x<weap>00000000`) :
//  - `shots` : tous les tirs publies sous ce tag ;
//  - `shotsWithVehicle` : ceux qui portent `v` (slot du vehicule) — « N tirs (dont M en vehicule) » ;
//  - `docs` : les documents (8 premiers caracteres) qui en publient au moins un ;
//  - `carriers` : pour chaque tir a `v`, la vie du vehicule `v` qui COUVRE l'instant du tir
//    (`t0 <= t <= t1max`), cle `famille:chassis` (`?` pour une famille vide) ; une vie introuvable
//    est comptee sous `?:?`.
//
// Usage (depuis apps/go-api) :
//   node internal/games/mappings/testdata/vehicle_weapons/sweep_vehicle_weapon_tags.mjs <dossier du parc> [--check]
// Le dossier est OBLIGATOIRE (un worktree n'a pas de `data/` : passer celui du checkout qui porte
// le parc) ; `--check` compare au lieu d'ecrire (code de sortie 1 sur un ecart).
import { readdirSync, readFileSync, writeFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

const HERE = dirname(fileURLToPath(import.meta.url))
const FIXTURE = join(HERE, 'vehicle_weapon_tags_observed.json')
const GENERATED_AT = '2026-09-23'

const args = process.argv.slice(2)
const check = args.includes('--check')
const dir = args.find((a) => !a.startsWith('--'))
if (dir === undefined) {
  console.error('usage : node sweep_vehicle_weapon_tags.mjs <dossier data/cache/replays/halo_infinite> [--check]')
  process.exit(2)
}

const isVehicleWeaponTag = (w) => typeof w === 'string' && w.endsWith('00000000')

/** La vie du véhicule `slot` qui couvre l'instant `t`, ou `undefined`. */
function lifeAt(vehicles, slot, t) {
  return vehicles.find((v) => v.slot === slot && v.t0 <= t && t <= (v.t1max ?? v.t1 ?? Infinity))
}

const files = readdirSync(dir).filter((f) => f.endsWith('.json') && !f.includes('derived')).sort()
const byTag = new Map()
const schemaVersions = new Set()
for (const f of files) {
  const doc = JSON.parse(readFileSync(join(dir, f), 'utf8'))
  schemaVersions.add(doc.schemaVersion)
  const vehicles = doc.vehicles ?? []
  for (const s of doc.shots ?? []) {
    if (!isVehicleWeaponTag(s.w)) continue
    let e = byTag.get(s.w)
    if (!e) {
      e = { tag: s.w, shots: 0, shotsWithVehicle: 0, docs: new Set(), carriers: {} }
      byTag.set(s.w, e)
    }
    e.shots++
    e.docs.add(f.slice(0, 8))
    if (s.v === undefined) continue
    e.shotsWithVehicle++
    const life = lifeAt(vehicles, s.v, s.t)
    const key = life ? `${life.family || '?'}:${life.chassis ?? '?'}` : '?:?'
    e.carriers[key] = (e.carriers[key] ?? 0) + 1
  }
}

const tags = [...byTag.values()]
  .sort((a, b) => b.shots - a.shots || a.tag.localeCompare(b.tag))
  .map((e) => ({
    tag: e.tag,
    shots: e.shots,
    shotsWithVehicle: e.shotsWithVehicle,
    docs: [...e.docs].sort(),
    carriers: Object.fromEntries(Object.entries(e.carriers).sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0]))),
  }))

const out = {
  generatedAt: GENERATED_AT,
  source:
    'instrument apps/go-api/internal/games/mappings/testdata/vehicle_weapons/sweep_vehicle_weapon_tags.mjs '
    + '(tirs au gabarit 0x<weap>00000000 ; porteur = vie du vehicule v qui COUVRE l instant, t0 <= t <= t1max)',
  corpus: {
    documents: files.length,
    schemaVersions: [...schemaVersions].sort((a, b) => a - b),
    dir: `data/cache/replays/halo_infinite (${files.length} documents, lecture seule)`,
  },
  tags,
}
const text = `${JSON.stringify(out, null, 2)}\n`

if (check) {
  const current = readFileSync(FIXTURE, 'utf8')
  if (current !== text) {
    console.error('fixture differente de la sortie de l instrument')
    process.exit(1)
  }
  console.log(`fixture a jour (${files.length} documents, ${tags.length} tags)`)
} else {
  writeFileSync(FIXTURE, text)
  console.log(`fixture ecrite (${files.length} documents, ${tags.length} tags)`)
}
