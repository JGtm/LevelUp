/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail « aucun inconnu à l'écran » (décision utilisateur du 2026-10-09) : aucune chaîne
 * d'interface, en français comme en anglais, ne dit qu'une grandeur n'est pas mesurée, pas lue,
 * pas publiée, ou qu'un match n'a pas de film. « Trop troublant, induit un manque de confiance » :
 * une donnée absente n'est pas rendue (case vide, ligne ou carte absente), une base se dit sans
 * dire ce qu'elle laisse de côté, un état vide décrit ce qui existe.
 *
 * MÉTHODE — sur le SOURCE, pour voir aussi les gabarits à paramètres que la résolution des
 * dictionnaires n'invoque pas :
 *   - fichiers `.ts` / `.tsx` de `src/` (hors tests, gabarits de test, code généré) : seul le
 *     CONTENU DES LITTÉRAUX de chaîne est examiné, commentaires retirés (un commentaire peut
 *     parler d'une absence, l'écran non) ;
 *   - manifestes TOML (`lib/i18n/manifests/*.toml`) : les seules valeurs `fr = …` / `en = …`.
 *
 * HORS PÉRIMÈTRE, justifié :
 *   - `features/admin/` et `admin.toml` — les écrans d'exploitation dont l'objet même est de
 *     compter les inconnus (couverture d'armes, trous de classement, files) ; servis au seul rôle
 *     administrateur ;
 *   - `features/legal/` — les mentions légales (« aucun service de mesure d'audience ») ne parlent
 *     pas de données de jeu ;
 *   - `lib/api/` — le contrat (types et descriptions OpenAPI générés), jamais rendu tel quel ;
 *   - les littéraux d'identifiant (minuscules sans espace : clés, `data-testid`, chemins).
 */
import { readFileSync, readdirSync, statSync } from 'node:fs'
import { join, relative, sep } from 'node:path'

import { describe, expect, it } from 'vitest'

const SRC = join(__dirname, '..', '..')
const MANIFESTS = join(SRC, 'lib', 'i18n', 'manifests')

/** Les formulations interdites : chacune dit un inconnu, une lacune ou une couverture. */
const FORBIDDEN: readonly RegExp[] = [
  /\bnon[\s-]?mesur/i,
  /\bnot[\s-]measur/i,
  /\bunmeasur/i,
  /\bsans film\b/i,
  /\bno film\b/i,
  /\bwithout (a )?film\b/i,
  /\bcouverture\b/i,
  /\bcoverage\b/i,
  /\b(aucune?|no|nothing)\s+(\S+\s+)?(mesur|measur)/i,
  /\b(frags?|kills?|matchs?|matches|vies?|lives?|usages?|soirées?|sessions?|prises?|pickups?|appuis?|distances?|positions?|portées?|ranges?|victoires?|défaites?)\s+mesur(é|ée|és|ées)\b/i,
  /\bmeasured\s+(kills?|matches|match|lives?|life|usages?|sessions?|pickups?|support|distances?|ranges?|positions?)\b/i,
  /\blecture indisponible\b|\breading unavailable\b/i,
  /\bnon publiable|\bnot publishable|\bnon publiés?\b|\bnot published\b/i,
  /\bnon lue?s? sur\b|\bnot read on\b/i,
  /\bramasseur connu|\bknown picker|\bportée de radar connue|\bknown radar range/i,
  /\bécartée?s?\s*:|\bleft out:/i,
  /\billisible\b|\bunreadable\b/i,
]

const EXCLUDED_DIRS = new Set(['generated', 'test', '__fixtures__'])

function isScannedSource(rel: string): boolean {
  if (!/\.(ts|tsx)$/.test(rel)) return false
  if (/\.(test|spec)\.tsx?$/.test(rel) || /\.fixtures?\.tsx?$/.test(rel)) return false
  if (rel.endsWith('routeTree.gen.ts')) return false
  const parts = rel.split(sep)
  if (parts.some((p) => EXCLUDED_DIRS.has(p))) return false
  if (parts[0] === 'features' && (parts[1] === 'admin' || parts[1] === 'legal')) return false
  return !(parts[0] === 'lib' && parts[1] === 'api')
}

/** Un littéral d'identifiant (clé, `data-testid`, chemin) : minuscules, sans espace. */
const IDENTIFIER = /^[a-z0-9_./{}:-]+$/

function walk(dir: string, out: string[] = []): string[] {
  for (const name of readdirSync(dir)) {
    const full = join(dir, name)
    if (statSync(full).isDirectory()) walk(full, out)
    else out.push(full)
  }
  return out
}

/**
 * Les contenus des littéraux de chaîne d'un source TS, commentaires écartés. Lecture à états :
 * code, commentaire de ligne, commentaire de bloc, chaîne '…', "…" ou `…` (une interpolation
 * `${…}` d'un gabarit coupe le littéral, son expression est relue comme du code).
 */
function stringLiterals(src: string): string[] {
  const out: string[] = []
  const templates: number[] = [] // profondeur d'accolades à laquelle chaque gabarit a été suspendu
  let depth = 0
  let i = 0
  const readQuoted = (quote: string) => {
    let s = ''
    i += 1
    while (i < src.length && src[i] !== quote) {
      if (src[i] === '\\') {
        s += src[i + 1] ?? ''
        i += 2
        continue
      }
      if (src[i] === '\n') break
      s += src[i]
      i += 1
    }
    i += 1
    out.push(s)
  }
  const readTemplate = () => {
    let s = ''
    while (i < src.length) {
      const c = src[i]
      if (c === '\\') {
        s += src[i + 1] ?? ''
        i += 2
        continue
      }
      if (c === '`') {
        i += 1
        out.push(s)
        return
      }
      if (c === '$' && src[i + 1] === '{') {
        out.push(s)
        i += 2
        templates.push(depth)
        depth += 1
        return
      }
      s += c
      i += 1
    }
    out.push(s)
  }
  while (i < src.length) {
    const c = src[i]
    const n = src[i + 1]
    if (c === '/' && n === '/') {
      while (i < src.length && src[i] !== '\n') i += 1
    } else if (c === '/' && n === '*') {
      const end = src.indexOf('*/', i + 2)
      i = end < 0 ? src.length : end + 2
    } else if (c === "'" || c === '"') {
      readQuoted(c)
    } else if (c === '`') {
      i += 1
      readTemplate()
    } else if (c === '{') {
      depth += 1
      i += 1
    } else if (c === '}') {
      depth -= 1
      i += 1
      if (templates.length > 0 && templates[templates.length - 1] === depth) {
        templates.pop()
        readTemplate()
      }
    } else {
      i += 1
    }
  }
  return out
}

/** Les valeurs `fr = "…"` / `en = "…"` d'un manifeste TOML. */
function manifestValues(src: string): string[] {
  return src
    .split('\n')
    .map((l) => /^(fr|en)\s*=\s*"(.*)"\s*$/.exec(l.trim())?.[2])
    .filter((v): v is string => v != null)
}

function offenders(texts: readonly string[]): string[] {
  return texts.filter((s) => !IDENTIFIER.test(s) && FORBIDDEN.some((re) => re.test(s)))
}

describe('aucun inconnu dans les chaînes d’interface (FR et EN)', () => {
  it('la lecture des littéraux voit les gabarits et ignore les commentaires', () => {
    const src = "// non mesuré\nconst a = 'Non mesuré' /* sans film */\nconst b = `x ${n ? 'y' : `${m} non mesurés`} z`"
    expect(stringLiterals(src)).toEqual(['Non mesuré', 'x ', 'y', '', ' non mesurés', ' z'])
    expect(offenders(stringLiterals(src))).toHaveLength(2)
  })

  it('aucun source TS/TSX ne porte une formulation d’inconnu', () => {
    const hits: string[] = []
    for (const file of walk(SRC)) {
      const rel = relative(SRC, file)
      if (!isScannedSource(rel)) continue
      for (const s of offenders(stringLiterals(readFileSync(file, 'utf8')))) hits.push(`${rel} : « ${s.trim()} »`)
    }
    expect(hits).toEqual([])
  })

  it('aucun manifeste TOML ne porte une formulation d’inconnu', () => {
    const hits: string[] = []
    for (const name of readdirSync(MANIFESTS)) {
      if (!name.endsWith('.toml') || name === 'admin.toml') continue
      for (const s of offenders(manifestValues(readFileSync(join(MANIFESTS, name), 'utf8')))) hits.push(`${name} : « ${s} »`)
    }
    expect(hits).toEqual([])
  })
})
