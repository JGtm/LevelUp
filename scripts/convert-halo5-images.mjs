#!/usr/bin/env node
// Convertit les images PNG de Halo 5 du front (apps/web/public/titles/halo_5) en WebP et retire les PNG.
//
// Pourquoi : ces images pesaient 41 Mo dans l'installeur de bureau (L23). Les formes de codage sont
// choisies par famille, parce que les masques du personnalisateur Spartan ne sont PAS des images
// ordinaires : `recolor.ts` lit R = zone primaire, G = zone secondaire, B = zone tertiaire, et
// alpha = max(R, G, B). Un codage avec perte (YUV 4:2:0) fait baver les couleurs pures les unes sur les
// autres et salit le noir hors zone ; il ne convient qu'aux grandes plaques (mesure du 2026-11-04 : 0,74 %
// des pixels s'ecartent de plus de 16 niveaux apres recoloration, contre 3,5 % pour les emblemes et 16 %
// pour leurs vignettes).
//
//   spartan/nameplates/*.png          1296x272, RGB    -> WebP avec perte, qualite 85 (-87 %)
//   wallpaper_*.png                   1600x450, RGB    -> WebP avec perte, qualite 85
//   spartan/emblems/*.png             256x256, RGBA    -> WebP SANS perte (masque a canaux + alpha)
//   spartan/*_thumb_raw/*.png         vignettes RGB    -> WebP SANS perte (masques a canaux, trop petits)
//
// Hors perimetre : `teams/` (le chemin `/titles/{slug}/teams/{id}.png` est partage avec Halo Infinite,
// dont les logos restent en PNG), tout ce qui est sous `halo_infinite/`, les fonds de carte du rejeu.
//
// Idempotent : un PNG n'est converti que si son WebP manque ou est plus ancien ; une fois converti, le
// PNG est supprime (git garde l'historique), donc relancer sans PNG ne fait rien. `--keep` garde les PNG
// (apercu, comparaison). Sortie : poids avant/apres par famille.
//
// Outil : ffmpeg du PATH (ou LEVELUP_FFMPEG_DIR), encodeur libwebp.
//
// Usage : node scripts/convert-halo5-images.mjs [--keep] [--dir <racine>] [--jobs <n>]

import { spawn } from 'node:child_process'
import { readdir, stat, unlink } from 'node:fs/promises'
import os from 'node:os'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const QUALITE_AVEC_PERTE = '85'

/** Familles dans l'ordre de specificite ; la premiere regle dont `test` accepte le chemin relatif gagne. */
export const REGLES = [
  { nom: 'plaques', test: (p) => p.startsWith('spartan/nameplates/'), args: ['-quality', QUALITE_AVEC_PERTE] },
  { nom: 'fonds', test: (p) => /^wallpaper_[^/]+\.png$/.test(p), args: ['-quality', QUALITE_AVEC_PERTE] },
  { nom: 'emblemes', test: (p) => p.startsWith('spartan/emblems/'), args: ['-lossless', '1', '-compression_level', '6'] },
  { nom: 'vignettes', test: (p) => /^spartan\/[a-z]+_thumb_raw\//.test(p), args: ['-lossless', '1', '-compression_level', '6'] },
]

/** Regle applicable a un chemin relatif (separateurs `/`), ou null (non converti). */
export function regleDe(cheminRelatif) {
  if (!cheminRelatif.endsWith('.png')) return null
  return REGLES.find((r) => r.test(cheminRelatif)) ?? null
}

async function lister(dossier) {
  const sortie = []
  for (const e of await readdir(dossier, { withFileTypes: true })) {
    const p = path.join(dossier, e.name)
    if (e.isDirectory()) sortie.push(...(await lister(p)))
    else sortie.push(p)
  }
  return sortie
}

function ffmpegExe() {
  const dir = process.env.LEVELUP_FFMPEG_DIR
  return dir ? path.join(dir, process.platform === 'win32' ? 'ffmpeg.exe' : 'ffmpeg') : 'ffmpeg'
}

function encoder(source, cible, args) {
  return new Promise((resolve, reject) => {
    // bgra : libwebp fait lui-meme la conversion de couleur (pas le swscale de ffmpeg) ; un PNG opaque
    // donne un WebP sans plan alpha.
    const p = spawn(ffmpegExe(), ['-v', 'error', '-y', '-i', source, '-c:v', 'libwebp', '-pix_fmt', 'bgra', ...args, '-frames:v', '1', cible], {
      stdio: ['ignore', 'ignore', 'pipe'],
    })
    let erreur = ''
    p.stderr.on('data', (d) => (erreur += d))
    p.on('error', reject)
    p.on('close', (code) => (code === 0 ? resolve() : reject(new Error(`ffmpeg a echoue (${code}) sur ${source} : ${erreur.trim()}`))))
  })
}

async function principal() {
  const argv = process.argv.slice(2)
  const garder = argv.includes('--keep')
  const val = (cle, defaut) => (argv.includes(cle) ? argv[argv.indexOf(cle) + 1] : defaut)
  const racine = path.resolve(val('--dir', path.join(path.dirname(fileURLToPath(import.meta.url)), '..', 'apps/web/public/titles/halo_5')))
  const jobs = Math.max(1, Number(val('--jobs', Math.min(6, os.cpus().length))))

  const taches = []
  for (const abs of await lister(racine)) {
    const rel = path.relative(racine, abs).split(path.sep).join('/')
    const regle = regleDe(rel)
    if (regle) taches.push({ abs, rel, regle, cible: abs.replace(/\.png$/, '.webp') })
  }

  const bilan = new Map()
  const ligne = (nom) => {
    if (!bilan.has(nom)) bilan.set(nom, { fichiers: 0, avant: 0, apres: 0 })
    return bilan.get(nom)
  }
  let suivant = 0
  const ouvrier = async () => {
    while (suivant < taches.length) {
      const t = taches[suivant++]
      const src = await stat(t.abs)
      const cibleExiste = await stat(t.cible).catch(() => null)
      if (!cibleExiste || cibleExiste.mtimeMs < src.mtimeMs) await encoder(t.abs, t.cible, t.regle.args)
      const apres = (await stat(t.cible)).size
      if (apres === 0) throw new Error(`WebP vide : ${t.cible}`)
      const l = ligne(t.regle.nom)
      l.fichiers += 1
      l.avant += src.size
      l.apres += apres
      if (!garder) await unlink(t.abs)
    }
  }
  await Promise.all(Array.from({ length: jobs }, ouvrier))

  const mo = (n) => (n / 1048576).toFixed(2).padStart(7)
  let totAvant = 0
  let totApres = 0
  for (const [nom, l] of bilan) {
    totAvant += l.avant
    totApres += l.apres
    console.log(`${nom.padEnd(10)} ${String(l.fichiers).padStart(4)} fichiers  ${mo(l.avant)} Mo -> ${mo(l.apres)} Mo`)
  }
  console.log(`${'total'.padEnd(10)} ${String(taches.length).padStart(4)} fichiers  ${mo(totAvant)} Mo -> ${mo(totApres)} Mo`)
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  principal().catch((e) => {
    console.error(e.message)
    process.exit(1)
  })
}
