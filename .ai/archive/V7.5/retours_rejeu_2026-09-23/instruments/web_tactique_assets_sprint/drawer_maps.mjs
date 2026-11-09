// Sonde (lecture seule) : reproduit le catalogue « Cartes » de l'Asset Drawer tel que le
// serveur le sert, À PARTIR DE LA COPIE de metadata.duckdb.
//
// Chaîne reproduite (fichier:ligne du dépôt @ 43a01721e) :
//   1. boot : MetadataRepo.ListMapsByTitle(ctx, "halo_infinite", "")      server.go:278
//      SQL recopié À L'IDENTIQUE de metadata_repo_assets_list.go:41-66 (search = '')
//   2. StaticAssetMetaRepo.filterAssets : Contains(lower(NameEN|NameFR), q) asset_meta_static.go:84-97
//   3. AssetService.ListMaps : ImageURL = hiAssetURL.MapImageURL(NameEN) ; "" => exclu
//      asset_service.go:55-59 ; adapter_asset_urls.go:138-163 (uuidRe, suffixes, fichier présent)
//   4. client : dedupeAssetsById (assetDrawerLogic.ts:29) puis libellé = FR ? name_fr||name_en : name_en
//      (AssetCard.tsx:15)
//
// Usage : node drawer_maps.mjs [recherche]
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

// 1. SQL de boot, recopié de metadata_repo_assets_list.go:41-66 avec search = ''.
const boot = sql(`
  SELECT asset_id, name_en, name_fr, name_canonical
  FROM (
    SELECT DISTINCT ON (m.map_asset_id)
           m.map_asset_id                                     AS asset_id,
           COALESCE(at_en.name, m.name_canonical, '')         AS name_en,
           COALESCE(at_fr.name, '')                           AS name_fr,
           COALESCE(m.name_canonical, '')                     AS name_canonical
    FROM maps_catalog m
    LEFT JOIN asset_translations at_en
        ON at_en.asset_id   = m.map_asset_id
       AND at_en.asset_type = 'map'
       AND at_en.lang       = 'en-US'
    LEFT JOIN asset_translations at_fr
        ON at_fr.asset_id   = m.map_asset_id
       AND at_fr.asset_type = 'map'
       AND at_fr.lang       = 'fr-FR'
    WHERE m.title_slug = 'halo_infinite'
      AND COALESCE(m.name_canonical, '') NOT LIKE '% - %'
    ORDER BY m.map_asset_id, at_en.name
  )
  ORDER BY name_canonical, name_en`)

// 3. Répertoire d'images (server.go:545 WithMapImagesDir -> static/maps/halo_infinite).
const dir = join(REPO, 'static', 'maps', 'halo_infinite')
const images = new Set(
  readdirSync(dir)
    .filter((f) => ['.jpg', '.png'].includes(extname(f).toLowerCase()))
    .map((f) => basename(f, extname(f))),
)
const uuidRe = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i
const suffixes = [' Heavies', ' Sentry Defense', ' Firefight']
function imageURL(nameEN) {
  const n = nameEN.trim()
  if (n === '' || uuidRe.test(n)) return ''
  if (suffixes.some((s) => n.endsWith(s))) return ''
  return images.has(n) ? `/static/maps/halo_infinite/${n}` : ''
}

// 2. + 3. filtre de recherche puis exclusion sans image.
const filtre = (m) => q === '' || m.name_en.toLowerCase().includes(q) || m.name_fr.toLowerCase().includes(q)
const servis = boot.filter(filtre).map((m) => ({ ...m, image: imageURL(m.name_en) })).filter((m) => m.image !== '')

// 4. dédoublonnage client par id (sans effet si les id sont distincts).
const vus = new Set()
const client = servis.filter((m) => (vus.has(m.asset_id) ? false : (vus.add(m.asset_id), true)))

const libelleFR = (m) => (m.name_fr ? m.name_fr : m.name_en)
function groupes(liste, cle) {
  const g = new Map()
  for (const m of liste) {
    const k = cle(m)
    if (!g.has(k)) g.set(k, [])
    g.get(k).push(m)
  }
  return [...g.entries()].filter(([, v]) => v.length > 1).sort((a, b) => b[1].length - a[1].length)
}

console.log(`recherche=${JSON.stringify(q)}`)
console.log(`lignes SQL (boot, search='') : ${boot.length}`)
console.log(`  dont name_canonical = uuid : ${boot.filter((m) => uuidRe.test(m.name_canonical)).length}`)
console.log(`après filtre de recherche     : ${boot.filter(filtre).length}`)
console.log(`servies (avec image)          : ${servis.length}`)
console.log(`après dedupeAssetsById        : ${client.length}  (ids distincts : ${new Set(client.map((m) => m.asset_id)).size})`)
const dFR = groupes(client, libelleFR)
const dImg = groupes(client, (m) => m.image)
console.log(`libellés FR en double         : ${dFR.length} noms, ${dFR.reduce((s, [, v]) => s + v.length - 1, 0)} cartes en trop`)
console.log(`images en double              : ${dImg.length} images, ${dImg.reduce((s, [, v]) => s + v.length - 1, 0)} cartes en trop`)
for (const [nom, v] of dFR) {
  console.log(`  ${v.length}x ${JSON.stringify(nom)}`)
  for (const m of v) console.log(`     ${m.asset_id}  name_canonical=${JSON.stringify(m.name_canonical)}  en=${JSON.stringify(m.name_en)} fr=${JSON.stringify(m.name_fr)}`)
}
