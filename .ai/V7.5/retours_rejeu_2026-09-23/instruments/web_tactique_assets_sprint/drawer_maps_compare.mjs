// Sonde (lecture seule) : compare trois grains du catalogue « Cartes » de l'Asset Drawer
// sur la COPIE de metadata.duckdb, après la même chaîne serveur que drawer_maps.mjs
// (filtre de recherche, exclusion sans image, dédoublonnage client par id).
//   actuel   : SQL @ 43a01721e  (DISTINCT ON (m.map_asset_id), commit 044751026 / D15, 2026-09-13)
//   avantD15 : SQL @ 044751026^ (DISTINCT ON (m.name_canonical), 2026-05-01)
//   propose  : SQL actuel + regroupement par URL d'image résolue (grain d'AFFICHAGE),
//              représentant = nom canonique résolu (non-uuid) d'abord, puis nom FR non vide,
//              puis asset_id (ordre total, déterministe).
// Usage : node drawer_maps_compare.mjs [recherche]
import { execFileSync } from 'node:child_process'
import { readdirSync } from 'node:fs'
import { extname, basename, join } from 'node:path'

const SP = 'C:/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp-go-migration/5e1d6f3f-b6be-4233-b361-e8a896dd728d/scratchpad'
const REPO = 'C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration'
const DB = join(SP, 'db', 'metadata.duckdb')
const q = (process.argv[2] ?? '').toLowerCase()

function sql(query) {
  const out = execFileSync(join(SP, 'diag_q.exe'), [DB, query], { encoding: 'utf8', maxBuffer: 64 << 20 })
  const lignes = out.split(/\r?\n/).filter((l) => l !== '' && !/^\(\d+ rows?\)$/.test(l))
  const entete = lignes.shift().split('\t')
  return lignes.map((l) => Object.fromEntries(l.split('\t').map((v, i) => [entete[i], v])))
}

const JOINS = `
    FROM maps_catalog m
    LEFT JOIN asset_translations at_en
        ON at_en.asset_id = m.map_asset_id AND at_en.asset_type = 'map' AND at_en.lang = 'en-US'
    LEFT JOIN asset_translations at_fr
        ON at_fr.asset_id = m.map_asset_id AND at_fr.asset_type = 'map' AND at_fr.lang = 'fr-FR'
    WHERE m.title_slug = 'halo_infinite'
      AND COALESCE(m.name_canonical, '') NOT LIKE '% - %'`

const actuel = sql(`
  SELECT asset_id, name_en, name_fr, name_canonical FROM (
    SELECT DISTINCT ON (m.map_asset_id)
           m.map_asset_id AS asset_id,
           COALESCE(at_en.name, m.name_canonical, '') AS name_en,
           COALESCE(at_fr.name, '') AS name_fr,
           COALESCE(m.name_canonical, '') AS name_canonical
    ${JOINS}
    ORDER BY m.map_asset_id, at_en.name)
  ORDER BY name_canonical, name_en`)

const avantD15 = sql(`
  SELECT DISTINCT ON (m.name_canonical)
         m.map_asset_id AS asset_id,
         COALESCE(at_en.name, m.name_canonical, '') AS name_en,
         COALESCE(at_fr.name, '') AS name_fr,
         COALESCE(m.name_canonical, '') AS name_canonical
  ${JOINS}
  ORDER BY m.name_canonical, at_en.name`)

const dir = join(REPO, 'static', 'maps', 'halo_infinite')
const images = new Set(readdirSync(dir).filter((f) => ['.jpg', '.png'].includes(extname(f).toLowerCase())).map((f) => basename(f, extname(f))))
const uuidRe = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i
const suffixes = [' Heavies', ' Sentry Defense', ' Firefight']
const imageURL = (n0) => {
  const n = n0.trim()
  if (n === '' || uuidRe.test(n) || suffixes.some((s) => n.endsWith(s))) return ''
  return images.has(n) ? `/static/maps/halo_infinite/${n}` : ''
}

function servir(lignes) {
  const filtre = (m) => q === '' || m.name_en.toLowerCase().includes(q) || m.name_fr.toLowerCase().includes(q)
  const servis = lignes.filter(filtre).map((m) => ({ ...m, image: imageURL(m.name_en) })).filter((m) => m.image !== '')
  const vus = new Set()
  return servis.filter((m) => (vus.has(m.asset_id) ? false : (vus.add(m.asset_id), true)))
}

function parImage(liste) {
  const rang = (m) => [uuidRe.test(m.name_canonical) ? 1 : 0, m.name_fr === '' ? 1 : 0, m.asset_id]
  const cmp = (a, b) => {
    const ra = rang(a), rb = rang(b)
    for (let i = 0; i < ra.length; i++) if (ra[i] !== rb[i]) return ra[i] < rb[i] ? -1 : 1
    return 0
  }
  const g = new Map()
  for (const m of liste) {
    const cur = g.get(m.image)
    if (!cur || cmp(m, cur) < 0) g.set(m.image, m)
  }
  return [...g.values()]
}

const libFR = (m) => (m.name_fr ? m.name_fr : m.name_en)
function bilan(nom, liste) {
  const parLib = new Map(), parImg = new Map()
  for (const m of liste) {
    parLib.set(libFR(m), (parLib.get(libFR(m)) ?? 0) + 1)
    parImg.set(m.image, (parImg.get(m.image) ?? 0) + 1)
  }
  const dLib = [...parLib.values()].filter((n) => n > 1)
  const dImg = [...parImg.values()].filter((n) => n > 1)
  const pire = Math.max(1, ...parImg.values())
  console.log(`${nom.padEnd(9)} cartes=${String(liste.length).padStart(3)}  images distinctes=${String(parImg.size).padStart(3)}  ` +
    `libellés FR en double=${dLib.length} (+${dLib.reduce((s, n) => s + n - 1, 0)})  ` +
    `images en double=${dImg.length} (+${dImg.reduce((s, n) => s + n - 1, 0)})  pire cas=${pire}x`)
  return parImg
}

console.log(`recherche=${JSON.stringify(q)}`)
const a = servir(actuel)
const b = servir(avantD15)
const p = parImage(a)
bilan('actuel', a)
bilan('avantD15', b)
bilan('propose', p)

// Ventilation des doublons ACTUELS : dus à D15 (homonymes à nom canonique résolu, que
// l'ancien DISTINCT ON (name_canonical) fusionnait) vs lignes à nom canonique = uuid.
let d15 = 0, uuid = 0
const g = new Map()
for (const m of a) {
  if (!g.has(m.image)) g.set(m.image, [])
  g.get(m.image).push(m)
}
for (const v of g.values()) {
  if (v.length < 2) continue
  const resolus = v.filter((m) => !uuidRe.test(m.name_canonical))
  const nomsResolus = new Set(resolus.map((m) => m.name_canonical))
  d15 += resolus.length - nomsResolus.size
  uuid += v.filter((m) => uuidRe.test(m.name_canonical)).length - (resolus.length === 0 ? 1 : 0)
}
console.log(`cartes en trop (actuel) : ${d15} dues à D15 (homonymes résolus) + ${uuid} dues à un nom canonique uuid`)
// Libellés FR divergents pour une même image (le représentant choisit l'un d'eux).
for (const [img, v] of g) {
  const libs = new Set(v.map(libFR))
  if (libs.size > 1) console.log(`  image ${img.split('/').pop()} : ${[...libs].map((l) => JSON.stringify(l)).join(' / ')} -> retenu ${JSON.stringify(libFR(p.find((m) => m.image === img)))}`)
}
