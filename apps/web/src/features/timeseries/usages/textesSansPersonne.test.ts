/**
 * textesSansPersonne.test.ts — GARDE : aucun possessif ni pronom de personne dans les textes des
 * Séries temporelles › Usages et de l'Escouade › Emprise (titres, intertitres, légendes, aides ⓘ,
 * infobulles), en français comme en anglais. Le joueur est désigné par son gamertag, le camp par
 * « Camp », l'autre par « Adversaire », le reste par « Reste du camp ».
 *
 * Toutes les chaînes des textes sont collectées, celles des fonctions comprises (appelées avec des
 * arguments d'échantillon : 0, 1, 2, un texte, une part d'équipement, un objet dont chaque champ
 * est un texte). Les manifestes sont lus ligne à ligne : `timeseries.toml` en entier, et dans
 * `synthesis.toml` le bloc `synthesis.weapon_range.*`, que lit la section « Portée » de l'onglet
 * Usages. Mot entier seulement (« mesuré », « nombre » passent) ; liste blanche vide.
 */
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

import { describe, expect, it } from 'vitest'

import { EMPRISE_TEXT } from '@/features/squad/emprise/empriseStrings'
import { PLACEMENT_TEXT } from '@/features/squad/emprise/placementStrings'
import { OBJECTIF_TEXT } from '@/features/squad/objectif/objectifStrings'

import { EMPRISE_TEXT_SOLO, OBJECTIF_TEXT_SOLO, USAGES_TEXT } from './usagesText'

const NOT_LETTER_BEFORE = '(?<![\\p{L}\\p{N}])'
const NOT_LETTER_AFTER = '(?![\\p{L}\\p{N}])'
const BANNED: Record<'fr' | 'en', RegExp> = {
  fr: new RegExp(`${NOT_LETTER_BEFORE}(ma|mes|mon|moi|notre|nos|nous|ta|tes|ton|toi|tu|te|vous|votre|vos)${NOT_LETTER_AFTER}`, 'iu'),
  en: new RegExp(`${NOT_LETTER_BEFORE}(my|our|we|us|me|your|you)${NOT_LETTER_AFTER}`, 'iu'),
}

/** Lignes `fr = "…"` / `en = "…"` fautives d'un manifeste ; `section` : préfixe des clés vérifiées. */
function offendingManifestLines(file: string, section: string): string[] {
  const toml = readFileSync(resolve(process.cwd(), 'src', 'lib', 'i18n', 'manifests', file), 'utf8')
  const offending: string[] = []
  let inScope = false
  for (const line of toml.split(/\r?\n/)) {
    const header = /^\[([^\]]+)\]$/.exec(line)
    if (header) inScope = header[1].startsWith(section)
    const m = /^(fr|en) = "(.*)"$/.exec(line)
    if (inScope && m && BANNED[m[1] as 'fr' | 'en'].test(m[2])) offending.push(line)
  }
  return offending
}

/** Un objet dont chaque champ lu est un texte (arguments objets des infobulles). */
const TEXT_OBJECT = new Proxy({}, { get: (_, key) => (key === Symbol.toPrimitive ? () => 'X' : 'X') })
/** 'used' : les fonctions qui indexent une part d'équipement (servi / gardé / lâché). */
const SAMPLES: unknown[] = [0, 1, 2, 'X', 'used', TEXT_OBJECT]

function collect(v: unknown, out: string[]): void {
  if (typeof v === 'string') {
    out.push(v)
  } else if (typeof v === 'function') {
    let evaluated = false
    for (const sample of SAMPLES) {
      try {
        collect((v as (...a: unknown[]) => unknown)(...Array(8).fill(sample)), out)
        evaluated = true
      } catch {
        // échantillon inadapté à cette fonction : le suivant
      }
    }
    if (!evaluated) throw new Error(`fonction de texte non évaluable : ${v.toString().slice(0, 80)}`)
  } else if (Array.isArray(v)) {
    v.forEach((x) => collect(x, out))
  } else if (v && typeof v === 'object') {
    Object.values(v).forEach((x) => collect(x, out))
  }
}

const TEXTS = {
  EMPRISE_TEXT,
  EMPRISE_TEXT_SOLO,
  PLACEMENT_TEXT,
  OBJECTIF_TEXT,
  OBJECTIF_TEXT_SOLO,
  USAGES_TEXT,
} as const

describe('aucun possessif ni pronom de personne', () => {
  for (const [name, byLocale] of Object.entries(TEXTS)) {
    for (const locale of ['fr', 'en'] as const) {
      it(`${name}.${locale}`, () => {
        const strings: string[] = []
        collect((byLocale as Record<string, unknown>)[locale], strings)
        expect(strings.length).toBeGreaterThan(0)
        expect(strings.filter((s) => BANNED[locale].test(s))).toEqual([])
      })
    }
  }

  it('manifeste timeseries.toml (fr, en)', () => {
    expect(offendingManifestLines('timeseries.toml', 'timeseries.')).toEqual([])
  })

  it('manifeste synthesis.toml, section « Portée » des Séries temporelles (synthesis.weapon_range.*)', () => {
    expect(offendingManifestLines('synthesis.toml', 'synthesis.weapon_range.')).toEqual([])
  })
})
