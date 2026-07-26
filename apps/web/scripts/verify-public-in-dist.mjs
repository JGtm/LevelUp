#!/usr/bin/env node
// verify-public-in-dist.mjs — garde-rail build : chaque fichier de public/ doit
// se retrouver dans dist/ apres `vite build` (Vite copie public/ tel quel a la
// racine du dist). Si un fichier manque, le build DOIT echouer avec la liste
// des manquants — prevention d'une regression FUTURE (option `copyPublicDir`
// desactivee, chemin public/ deplace/renomme, etc.), pas le correctif d'un bug
// prod deja survenu : aucune investigation n'a confirme d'image absente en
// production. L'hypothese initiale (.dockerignore strippant public/ du
// contexte de build) etait fausse : un motif .dockerignore est ancre a la
// racine du contexte, `*.png` ne traverse pas les sous-dossiers (cf.
// .dockerignore + thought_log 2026-07-26).
//
// Generique par construction : aucune enumeration de fichiers, d'extensions ni
// de slug — la verite est le contenu reel de public/.
//
// Usage : node scripts/verify-public-in-dist.mjs [publicDir] [distDir]
// (defauts : ./public et ./dist relatifs au cwd — apps/web).
import { readdirSync, existsSync } from 'node:fs'
import { join, relative, resolve, sep } from 'node:path'
import process from 'node:process'

const publicDir = resolve(process.argv[2] ?? 'public')
const distDir = resolve(process.argv[3] ?? 'dist')

/** Liste recursive des fichiers d'un dossier, en chemins relatifs a `base`. */
function listFilesRecursive(dir, base) {
  const files = []
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const full = join(dir, entry.name)
    if (entry.isDirectory()) {
      files.push(...listFilesRecursive(full, base))
    } else if (entry.isFile()) {
      files.push(relative(base, full))
    }
  }
  return files
}

if (!existsSync(publicDir)) {
  // public/ absent = contexte de build ampute (c'est exactement le mode de
  // panne qu'on veut attraper) — jamais un cas nominal.
  console.error(`verify-public-in-dist: dossier public introuvable: ${publicDir}`)
  process.exit(1)
}
if (!existsSync(distDir)) {
  console.error(`verify-public-in-dist: dossier dist introuvable: ${distDir} (lancer \`npm run build\` avant)`)
  process.exit(1)
}

const publicFiles = listFilesRecursive(publicDir, publicDir)
const missing = publicFiles.filter((rel) => !existsSync(join(distDir, rel)))

if (missing.length > 0) {
  console.error(
    `verify-public-in-dist: ECHEC — ${missing.length} fichier(s) de public/ absent(s) de dist/ :`,
  )
  for (const rel of missing) {
    console.error(`  - ${rel.split(sep).join('/')}`)
  }
  console.error(
    'Cause probable : option copyPublicDir desactivee dans la config Vite, ' +
      'public/ deplace/renomme, ou fichier ajoute apres les negations ' +
      '!apps/web/public/** de .dockerignore (verifier ce fichier).',
  )
  process.exit(1)
}

console.log(
  `verify-public-in-dist: OK — ${publicFiles.length} fichier(s) de public/ presents dans dist/.`,
)
