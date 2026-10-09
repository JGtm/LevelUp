/**
 * textesSansPersonne.test.ts — GARDE : aucun possessif, pronom de personne ni impératif de la 2e
 * personne (« reviens », « vérifie », « connecte » ; « connect it » en anglais) dans les textes des Séries temporelles › Usages, de toute la
 * page Escouade (`squad/i18n.ts`, les `*Strings.ts` de `features/squad/`, le sélecteur
 * `squadPresets.i18n.ts`, les jeux des cartes d'objectif `formes/i18n.ts` et `formes/cardsI18n.ts`,
 * le manifeste `squad.toml`), des cartes de la page Sessions et de la Vue match (titres, intertitres, légendes,
 * aides ⓘ, infobulles, messages), en français comme en anglais. Pour la Vue match, s'y ajoutent les
 * littéraux de phrase de ses sources (`features/match-view/`). Le joueur est désigné par son
 * gamertag, les groupes par « Équipe », « Adversaire », « Reste de l'équipe » ; « camp » n'est
 * jamais écrit (le champ de données `camp` peut le rester).
 *
 * Toutes les chaînes des textes sont collectées, celles des fonctions comprises (appelées avec des
 * arguments d'échantillon : 0, 1, 2, un texte, une part d'équipement, un objet dont chaque champ
 * est un texte). Les manifestes sont lus ligne à ligne : `timeseries.toml`, `session.toml`,
 * `match_view.toml` et `squad.toml` en entier, et dans `synthesis.toml` le bloc `synthesis.weapon_range.*`, que lit
 * la section « Portée » de l'onglet Usages. Mot entier seulement (« mesuré », « nombre » passent) ;
 * liste blanche vide.
 */
import { readdirSync, readFileSync } from 'node:fs'
import { resolve } from 'node:path'

import { describe, expect, it } from 'vitest'

import { REPLAY_TEXT } from '@/features/match-replay/i18n/i18n'
import { MATCH_VIEW_TEXT } from '@/features/match-view/i18n'
import { MATCH_EMPRISE_TEXT } from '@/features/match-view/matchEmpriseText'
import { COORDINATION_TEXT } from '@/features/session-detail/coordinationI18n'
import { SESSION_CARD_TEXT } from '@/features/session-detail/sessionEmpriseText'
import { EMPRISE_TEXT } from '@/features/squad/emprise/empriseStrings'
import { PLACEMENT_TEXT } from '@/features/squad/emprise/placementStrings'
import { FORMES_CARDS_TEXT } from '@/features/squad/formes/cardsI18n'
import { FORMES_TEXT } from '@/features/squad/formes/i18n'
import { EN_TEXT as SQUAD_EN, FR_TEXT as SQUAD_FR } from '@/features/squad/i18n'
import { OBJECTIF_TEXT } from '@/features/squad/objectif/objectifStrings'
import { getSquadFocusText } from '@/features/squad/squadFocusStrings'
import { SQUAD_PRESETS_STRINGS } from '@/features/squad/squadPresets.i18n'
import { getSquadRangeRolesText } from '@/features/squad/squadRangeRolesStrings'

import { EMPRISE_TEXT_SOLO, OBJECTIF_TEXT_SOLO, USAGES_TEXT } from './usagesText'

const NOT_LETTER_BEFORE = '(?<![\\p{L}\\p{N}])'
const NOT_LETTER_AFTER = '(?![\\p{L}\\p{N}])'
const BANNED: Record<'fr' | 'en', RegExp> = {
  fr: new RegExp(`${NOT_LETTER_BEFORE}(ma|mes|mon|moi|me|je|j(?=['’])|m(?=['’])|t(?=['’])|notre|nos|nous|ta|tes|ton|toi|tu|te|vous|votre|vos|reviens|vérifie|connecte|camp|camps)${NOT_LETTER_AFTER}`, 'iu'),
  en: new RegExp(`${NOT_LETTER_BEFORE}(my|our|we|us|me|your|you|connect[ ]+it)${NOT_LETTER_AFTER}`, 'iu'),
}

/** Un code de locale (`en-US`, champ `intlLocale` du jeu de l'Escouade) n'est pas un texte affiché. */
const LOCALE_TAG = /^[a-z]{2,3}-[A-Z]{2}$/

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

/**
 * Les textes des cartes de la page Sessions. Du jeu de l'Escouade (`squad`), la page lit trois parties :
 * `performanceCharts` (Répartition des frags, `SquadFragBreakdownCard`), `weaponKills` (Outils de
 * destruction, `SessionToolsCard`) et `empty` (état vide de ces deux cartes) ; seules ces trois
 * parties sont vérifiées, le reste appartient à la page Escouade.
 */
type SessionView = (typeof SESSION_CARD_TEXT)['fr']['full']
const sessionView = (v: SessionView) => ({
  ...v,
  squad: { performanceCharts: v.squad.performanceCharts, weaponKills: v.squad.weaponKills, empty: v.squad.empty },
})
const SESSION_TEXTS = Object.fromEntries(
  Object.entries(SESSION_CARD_TEXT).map(([locale, t]) => [
    locale,
    { full: sessionView(t.full), compact: sessionView(t.compact), compactCards: t.compactCards },
  ]),
)

/** La carte « Usage d'équipements, par joueur » de la Vue match : la seule partie du dictionnaire du rejeu qu'elle lit. */
const MATCH_EQUIPMENT_USAGE_TEXT = { fr: REPLAY_TEXT.fr.equipmentUsage, en: REPLAY_TEXT.en.equipmentUsage }

/** Le jeu entier de la page Escouade, ses libellés de focus et ceux des « Rôles de portée ». */
const SQUAD_TEXT = { fr: SQUAD_FR, en: SQUAD_EN }
const SQUAD_FOCUS_TEXT = { fr: getSquadFocusText('fr'), en: getSquadFocusText('en') }
const SQUAD_RANGE_ROLES_TEXT = { fr: getSquadRangeRolesText('fr'), en: getSquadRangeRolesText('en') }

const TEXTS = {
  SQUAD_TEXT,
  SQUAD_FOCUS_TEXT,
  SQUAD_RANGE_ROLES_TEXT,
  SQUAD_PRESETS_STRINGS,
  FORMES_TEXT,
  FORMES_CARDS_TEXT,
  EMPRISE_TEXT,
  EMPRISE_TEXT_SOLO,
  PLACEMENT_TEXT,
  OBJECTIF_TEXT,
  OBJECTIF_TEXT_SOLO,
  USAGES_TEXT,
  SESSION_TEXTS,
  COORDINATION_TEXT,
  MATCH_VIEW_TEXT,
  MATCH_EMPRISE_TEXT,
  MATCH_EQUIPMENT_USAGE_TEXT,
} as const

describe('aucun possessif ni pronom de personne', () => {
  for (const [name, byLocale] of Object.entries(TEXTS)) {
    for (const locale of ['fr', 'en'] as const) {
      it(`${name}.${locale}`, () => {
        const strings: string[] = []
        collect((byLocale as Record<string, unknown>)[locale], strings)
        expect(strings.length).toBeGreaterThan(0)
        expect(strings.filter((s) => !LOCALE_TAG.test(s) && BANNED[locale].test(s))).toEqual([])
      })
    }
  }

  it('manifeste session.toml (fr, en)', () => {
    expect(offendingManifestLines('session.toml', 'session.')).toEqual([])
  })

  it('manifeste timeseries.toml (fr, en)', () => {
    expect(offendingManifestLines('timeseries.toml', 'timeseries.')).toEqual([])
  })

  it('manifeste synthesis.toml, section « Portée » des Séries temporelles (synthesis.weapon_range.*)', () => {
    expect(offendingManifestLines('synthesis.toml', 'synthesis.weapon_range.')).toEqual([])
  })

  it('manifeste match_view.toml (fr, en)', () => {
    expect(offendingManifestLines('match_view.toml', 'match_view.')).toEqual([])
  })

  it('manifeste common.toml, messages de la synchronisation initiale (common.initial_sync.*)', () => {
    expect(offendingManifestLines('common.toml', 'common.initial_sync.')).toEqual([])
  })

  it('manifeste squad.toml (fr, en), en entier', () => {
    expect(offendingManifestLines('squad.toml', '')).toEqual([])
  })

  it('littéraux de phrase des sources de la Vue match (features/match-view, hors tests)', () => {
    expect(offendingSourcePhrases(resolve(process.cwd(), 'src', 'features', 'match-view'))).toEqual([])
  })
})

/** Une liste de classes (« flex items-center gap-2 ») n'est pas une phrase. */
function isClassList(s: string): boolean {
  const tokens = s.trim().split(/\s+/)
  return tokens.every((t) => /^[a-z0-9!:\-[\]./%#()_,&>*=~+]+$/.test(t)) && tokens.some((t) => /[-:[]/.test(t))
}

/**
 * Les littéraux de PHRASE d'un dossier de sources (`.ts` / `.tsx`, tests exclus) : chaînes entre
 * guillemets, parties fixes des gabarits, textes JSX — ceux qui portent un blanc et un mot, hors
 * listes de classes, fragments de code (« a.b », « a_b ») et commentaires. Rend `fichier: phrase` pour chaque phrase fautive (FR ou EN).
 */
function offendingSourcePhrases(dir: string): string[] {
  const offending: string[] = []
  for (const name of readdirSync(dir)) {
    if (!/\.tsx?$/.test(name) || /\.test\.tsx?$/.test(name)) continue
    const code = readFileSync(resolve(dir, name), 'utf8')
      .replace(/\/\*[\s\S]*?\*\//g, ' ')
      .replace(/(^|[^:'"`])\/\/.*$/gm, '$1')
    const literals = [
      ...[...code.matchAll(/'((?:[^'\\\n]|\\.)*)'/g)].map((m) => m[1]),
      ...[...code.matchAll(/"((?:[^"\\\n]|\\.)*)"/g)].map((m) => m[1]),
      ...[...code.matchAll(/`((?:[^`\\]|\\.)*)`/g)].flatMap((m) => m[1].split(/\$\{[^}]*\}/)),
      ...[...code.matchAll(/>([^<>{}]+)</g)].map((m) => m[1]),
    ]
    for (const s of literals) {
      if (!/\s/.test(s.trim()) || !/\p{L}{2,}/u.test(s) || isClassList(s) || /[A-Za-z][._][A-Za-z]/.test(s)) continue
      if (BANNED.fr.test(s) || BANNED.en.test(s)) offending.push(`${name}: ${s.trim()}`)
    }
  }
  return offending
}
