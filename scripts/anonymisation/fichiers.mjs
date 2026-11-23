// fichiers.mjs - quels fichiers l'outil lit, et comment il les decode.
//
// Seuls les fichiers SUIVIS par git sont candidats. Sont toujours exclus : les fichiers generes
// (routeTree.gen.ts, generated.ts, *_gen.go), les sommes de controle et verrous (go.sum, lockfiles),
// les archives et fixtures compressees, les binaires (par extension, puis par octet nul), tout
// fichier qui n'est pas de l'UTF-8 valide, et le dossier de l'outil lui-meme.

import { execFileSync } from 'node:child_process';
import fs from 'node:fs';
import path from 'node:path';

export const EXCLUSIONS_DEFAUT = [
  'scripts/anonymisation/**',
  '**/routeTree.gen.ts',
  '**/generated.ts',
  '**/*_gen.go',
  '**/*.gen.ts',
  '**/go.sum',
  '**/package-lock.json',
  '**/pnpm-lock.yaml',
  '**/yarn.lock',
  '**/Cargo.lock',
  '**/*.gz',
  '**/*.zst',
  '**/*.zip',
  '**/*.bin',
  '**/*.duckdb',
  '**/*.duckdb.wal',
  '**/*.wal',
  '**/*.sig',
];

const EXTENSIONS_BINAIRES = /\.(webp|png|jpe?g|gif|ico|bmp|ogg|wav|mp3|mp4|webm|woff2?|ttf|otf|pdf|blob|mvar|exe|dll|so|a|o)$/i;

/** Convertit un motif glob (`**`, `*`, `?`) en expression sur des chemins en `/`. */
export function globVersRegex(motif) {
  let re = '';
  for (let i = 0; i < motif.length; i++) {
    const c = motif[i];
    if (c === '*' && motif[i + 1] === '*') {
      i += 1;
      if (motif[i + 1] === '/') {
        i += 1;
        re += '(?:.*/)?';
      } else re += '.*';
    } else if (c === '*') re += '[^/]*';
    else if (c === '?') re += '[^/]';
    else re += c.replace(/[.+^${}()|[\]\\]/g, '\\$&');
  }
  return new RegExp(`^${re}$`);
}

/** Lit un fichier de chemins : une ligne `+motif` (inclusion) ou `-motif` (exclusion) ; `#` = commentaire. */
export function lireListeChemins(texte) {
  const inclure = [];
  const exclure = [];
  for (const brut of texte.split(/\r?\n/)) {
    const l = brut.trim();
    if (l === '' || l.startsWith('#')) continue;
    if (l.startsWith('+')) inclure.push(l.slice(1).trim());
    else if (l.startsWith('-')) exclure.push(l.slice(1).trim());
    else throw new Error(`liste de chemins : la ligne "${l}" ne commence ni par + ni par -`);
  }
  return { inclure, exclure };
}

/** Les chemins suivis par git sous `racine`, filtres par inclusions et exclusions. */
export function choisirFichiers(racine, { inclure = [], exclure = [] } = {}) {
  const suivis = execFileSync('git', ['ls-files', '-z'], { cwd: racine, maxBuffer: 1 << 30 })
    .toString('utf8')
    .split('\0')
    .filter(Boolean);
  const inc = inclure.map(globVersRegex);
  const exc = [...EXCLUSIONS_DEFAUT, ...exclure].map(globVersRegex);
  const retenus = [];
  const ignores = { exclusion: 0, binaire: 0 };
  for (const f of suivis) {
    if (exc.some((r) => r.test(f)) || (inc.length > 0 && !inc.some((r) => r.test(f)))) {
      ignores.exclusion++;
    } else if (EXTENSIONS_BINAIRES.test(f)) {
      ignores.binaire++;
    } else {
      retenus.push(f);
    }
  }
  return { retenus, ignores };
}

/** Decode un fichier. Rend { texte, bom } ou { ignore: 'binaire' | 'utf16' | 'non-utf8' }. */
export function decoder(octets) {
  if (octets.length >= 2 && ((octets[0] === 0xff && octets[1] === 0xfe) || (octets[0] === 0xfe && octets[1] === 0xff))) {
    return { ignore: 'utf16' };
  }
  if (octets.subarray(0, 8000).includes(0)) return { ignore: 'binaire' };
  const bom = octets.length >= 3 && octets[0] === 0xef && octets[1] === 0xbb && octets[2] === 0xbf;
  try {
    const texte = new TextDecoder('utf-8', { fatal: true }).decode(bom ? octets.subarray(3) : octets);
    return { texte, bom };
  } catch {
    return { ignore: 'non-utf8' };
  }
}

/** Reencode : meme BOM qu'a la lecture. */
export function encoder(texte, bom) {
  const corps = Buffer.from(texte, 'utf8');
  return bom ? Buffer.concat([Buffer.from([0xef, 0xbb, 0xbf]), corps]) : corps;
}

/** Vrai si `enfant` est `parent` ou se trouve dessous (chemins absolus, casse ignoree sous Windows). */
export function estSous(parent, enfant) {
  const rel = path.relative(path.resolve(parent), path.resolve(enfant));
  const meme = rel === '' || (!rel.startsWith('..') && !path.isAbsolute(rel));
  return meme;
}

export function lireOctets(racine, relatif) {
  return fs.readFileSync(path.join(racine, relatif));
}

const EXTENSIONS_DOCS = /\.(md|txt|tsv|html?|log|csv|diff|patch|out|err|jsonl|svg|cff|rst)$/i;
const EXTENSIONS_CODE_SOUS_AI = /\.(go|ts|tsx|mjs|js|sh|ps1|lua|py|rs|sql|awk|toml|ya?ml)$/i;

/** 'docs' ou 'code' : le genre qui choisit les entrees de table a appliquer (drapeaux docs/code).
 *  Documents : texte de pilotage (md, txt, tsv, html, journaux...), tout ce qui est sous `.ai/` ou
 *  `docs/` hors scripts, et les fichiers sans extension en capitales (LICENSE, NOTICE). */
export function genreDeFichier(f) {
  if (EXTENSIONS_DOCS.test(f)) return 'docs';
  if (EXTENSIONS_CODE_SOUS_AI.test(f)) return 'code';
  if (f.startsWith('.ai/') || f.startsWith('docs/')) return 'docs';
  if (/(^|\/)[A-Z][A-Z0-9_-]*$/.test(f)) return 'docs';
  return 'code';
}
