/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail anti-anglicismes (I15, 2026-07-24 — CLAUDE.md règle n°1 : « UI FR
 * sans anglicismes ») — 3e occurrence documentée du pattern « kills/assists
 * oubliés en FR » (règle projet n°6 : à la 3e occurrence, centraliser +
 * garde-rail qui interdit l'ancien littéral). Ce test interdit la
 * RÉ-introduction, côté FRANÇAIS uniquement, des anglicismes purgés par I15 :
 * PB, kill(s), assist(s), streak, win rate, leaderboard.
 *
 * PÉRIMÈTRE — fichiers traités par I15 + complément I15-bis (2026-07-24, lot
 * dédié à la dette listée comme point de vigilance dans le rapport I15) :
 * pas un audit exhaustif du corpus i18n (dette pré-existante plus large hors
 * périmètre, cf. rapports I15/I15-bis) :
 *   - 5 dictionnaires hand-written (features/<feature>/i18n.ts) : notifications,
 *     ascension, match-view, squad, _shared/SessionBriefing.
 *   - 9 manifests TOML (lib/i18n/manifests/*.toml) : profile, squad, common,
 *     palmares, admin, match_view, timeseries, explorer, session (les 3
 *     derniers ajoutés par I15-bis — anciennement exclus, dette résorbée).
 *
 * EXCLU DÉLIBÉRÉMENT — `coaching_tips.toml` (~80 entrées, registre esport) :
 * relecture éditoriale dédiée faite le 2026-07-24 (hors I15/I15-bis, périmètre
 * séparé), avec correction directe des anglicismes clairs (verbes anglais bruts,
 * termes ayant un équivalent FR déjà établi dans l'app) et conservation
 * documentée du jargon esport assumé (strafe, clutch, carry, trade, peek, ADS,
 * hipfire, aim assist, POV, stack, snowball, loadout, combo, timing, momentum,
 * push, aggro, feed, rat, miss, sticky, drill, aimbot, solo queue, callout,
 * stat-padder…). Ce jargon assumé ferait échouer FORBIDDEN_PATTERNS/l'esprit du
 * garde-rail s'il était scanné ici — ne pas ajouter ce manifest à la liste
 * ci-dessus sans une repasse dédiée qui distingue jargon assumé et anglicisme
 * réel (quelques cas restent ambigus, arbitrage utilisateur via Notion).
 *
 * MÉTHODE — on résout la valeur FR réellement servie à l'utilisateur (jamais
 * de regex sur le fichier source brut, qui mélangerait FR et EN) :
 *   - dicts hand-written : on appelle le getter/l'objet avec la locale 'fr'
 *     (même résolution que le runtime applicatif), puis on parcourt
 *     récursivement les valeurs. Les valeurs FONCTIONS (templates avec
 *     paramètres, ex. `(n) => \`Match ${n}\``) ne sont PAS invoquées : audit
 *     manuel (I15) confirmé qu'aucune ne contient de littéral interdit à ce
 *     jour. Si un futur template embarque un anglicisme, ce garde-rail ne le
 *     verra pas — limitation connue, à lever si le besoin apparaît.
 *   - manifests TOML : parse avec `@iarna/toml` (même lib que le script de
 *     build `build_i18n_manifests.mjs`), on ne scanne QUE la feuille `fr`.
 *   - Jetons d'interpolation ICU `{nom_du_param}` STRIPPÉS avant le test de
 *     pattern : ce sont des noms de variables techniques (jamais rendus tels
 *     quels à l'utilisateur), pas de la prose. Cas réel trouvé par ce
 *     garde-rail (I15) : `notif.rival_encounter.body` = « Tu as recroisé
 *     {gamertag} : {kills} frags / {deaths} morts. » — le paramètre technique
 *     `{kills}` reflète le nom de clé backend, le texte VISIBLE dit "frags" ;
 *     sans ce stripping, le garde-rail produirait un faux positif sur son
 *     propre nom de variable.
 *
 * EXCEPTIONS (explicites, commentées ci-dessous) :
 *   - Abréviations `assist.`/`Assist.` (ambigües mais lisibles en contexte de
 *     colonne étroite) : tolérées par décision I15 explicite (2026-07-24) —
 *     `common.toml:common.match_card.assists`, `match-view:sbColAssists` et
 *     `match_view.toml:match_view.cards.assists_vs_expected`.
 *   - `match_view.toml:match_view.summary.*` (kpi_kills, kpi_deaths,
 *     kpi_assists, kpi_kda, kpi_accuracy, kpi_personal_score, kpi_avg_life,
 *     kpi_max_spree, kpi_headshots, kpi_perfect_kills, expected_label,
 *     delta_label) : DÉCOUVERTE I15 (2026-07-24), confirmée et traitée par
 *     I15-bis (2026-07-24) — grep exhaustif (y compris accès dynamiques par
 *     template de clé) confirmant AUCUNE référence ailleurs dans
 *     apps/web/src : section entière morte (Onglet Résumé superseded),
 *     supprimée du manifest (FR+EN). Plus d'exception nécessaire ici.
 *   - `badge` et `playlist` ne sont volontairement PAS dans la liste des
 *     patterns interdits (mots jugés assimilés / cas au cas par cas ailleurs)
 *     — rien à exempter pour eux.
 *   - Noms propres (Kamikaze, Top Gun) : ne matchent aucun des patterns
 *     interdits, aucune exception nécessaire.
 */
import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'
// Import nommé (pas de default export dans les types @iarna/toml) : évite une
// dépendance à esModuleInterop, non activé dans tsconfig.app.json.
import { parse as parseTOML } from '@iarna/toml'

import { getNotificationsText } from '@/features/notifications/i18n'
import { getAscensionText } from '@/features/ascension/i18n'
import { MATCH_VIEW_TEXT } from '@/features/match-view/i18n'
import { getSquadText } from '@/features/squad/i18n'
import { getBriefingTexts } from '@/features/_shared/SessionBriefing/i18n'

// cwd du process vitest = apps/web (racine du projet front) — même convention
// que lib/i18n/locale.ratchet.test.ts (resolve(process.cwd(), 'src')) : on
// évite `__dirname`, non fiable en ESM sans shim explicite dans ce projet.
const MANIFESTS_DIR = join(process.cwd(), 'src', 'lib', 'i18n', 'manifests')

// Patterns interdits (I15). Insensible à la casse : un anglicisme reste un
// anglicisme en début de phrase ("Kills", "Streak"...).
const FORBIDDEN_PATTERNS: { name: string; re: RegExp }[] = [
  { name: 'PB', re: /\bPB\b/i },
  { name: 'kill(s)', re: /\bkills?\b/i },
  { name: 'assist(s)', re: /\bassists?\b/i },
  { name: 'streak', re: /\bstreak\b/i },
  { name: 'win rate', re: /\bwin rate\b/i },
  { name: 'leaderboard', re: /\bleaderboard\b/i },
]

// Exceptions explicites : clé = `${source}:${path}`. Chaque entrée est datée
// et justifiée dans le commentaire d'en-tête ci-dessus.
const EXCEPTIONS = new Set<string>([
  'match-view:sbColAssists', // "Assist." — abréviation colonne étroite, tolérée (I15 2026-07-24)
  'common.toml:common.match_card.assists', // "assist." — idem
  'match_view.toml:match_view.cards.assists_vs_expected', // "Assist." — idem
])

interface Offender {
  source: string
  path: string
  value: string
  pattern: string
}

// Retire les jetons d'interpolation ICU (`{name}`, `{n, number}`, et les
// `{n, plural, one {# match} other {# matchs}}` imbriqués) avant le test de
// pattern — cf. commentaire d'en-tête MÉTHODE. Itère jusqu'à stabilité : une
// passe ne retire que les accolades les plus internes, donc un pluriel
// imbriqué a besoin de plusieurs passes pour disparaître entièrement.
function stripInterpolationTokens(value: string): string {
  let current = value
  let next = current.replace(/\{[^{}]*\}/g, ' ')
  while (next !== current) {
    current = next
    next = current.replace(/\{[^{}]*\}/g, ' ')
  }
  return current
}

function collectStrings(
  value: unknown,
  path: string,
  out: { path: string; value: string }[],
): void {
  if (typeof value === 'string') {
    out.push({ path, value })
    return
  }
  if (typeof value === 'function') {
    // Templates paramétrés non invoqués — cf. commentaire d'en-tête (MÉTHODE).
    return
  }
  if (Array.isArray(value)) {
    value.forEach((v, i) => collectStrings(v, `${path}[${i}]`, out))
    return
  }
  if (value && typeof value === 'object') {
    for (const [k, v] of Object.entries(value as Record<string, unknown>)) {
      collectStrings(v, path ? `${path}.${k}` : k, out)
    }
  }
}

function scanDict(source: string, frValue: unknown): Offender[] {
  const strings: { path: string; value: string }[] = []
  collectStrings(frValue, '', strings)
  const offenders: Offender[] = []
  for (const { path, value } of strings) {
    const stripped = stripInterpolationTokens(value)
    for (const { name, re } of FORBIDDEN_PATTERNS) {
      if (re.test(stripped)) {
        const key = `${source}:${path}`
        if (!EXCEPTIONS.has(key)) {
          offenders.push({ source, path, value, pattern: name })
        }
      }
    }
  }
  return offenders
}

// Flatten manifest TOML : ne garde que la feuille `fr` d'un noeud {fr, en}.
function flattenManifestFr(obj: unknown, prefix: string, out: Map<string, string>): void {
  if (!obj || typeof obj !== 'object') return
  const rec = obj as Record<string, unknown>
  if (typeof rec.fr === 'string' && typeof rec.en === 'string') {
    out.set(prefix, rec.fr)
    return
  }
  for (const [k, v] of Object.entries(rec)) {
    flattenManifestFr(v, prefix ? `${prefix}.${k}` : k, out)
  }
}

function scanManifest(fileName: string): Offender[] {
  const raw = readFileSync(join(MANIFESTS_DIR, fileName), 'utf8')
  const parsed = parseTOML(raw)
  const flat = new Map<string, string>()
  flattenManifestFr(parsed, '', flat)
  const offenders: Offender[] = []
  const source = fileName
  for (const [key, value] of flat) {
    const stripped = stripInterpolationTokens(value)
    for (const { name, re } of FORBIDDEN_PATTERNS) {
      if (re.test(stripped)) {
        const excKey = `${source}:${key}`
        if (!EXCEPTIONS.has(excKey)) {
          offenders.push({ source, path: key, value, pattern: name })
        }
      }
    }
  }
  return offenders
}

function formatOffenders(offenders: Offender[]): string {
  return offenders
    .map((o) => `${o.source}:${o.path} — "${o.value}" (pattern: ${o.pattern})`)
    .join('\n')
}

describe('garde-rail anti-anglicismes FR (I15 + I15-bis, périmètre = fichiers traités)', () => {
  it('dictionnaires hand-written : aucun anglicisme interdit côté FR', () => {
    const offenders = [
      ...scanDict('notifications', getNotificationsText('fr')),
      ...scanDict('ascension', getAscensionText('fr')),
      ...scanDict('match-view', MATCH_VIEW_TEXT.fr),
      ...scanDict('squad', getSquadText('fr')),
      ...scanDict('session-briefing', getBriefingTexts('fr')),
    ]
    expect(offenders, `Anglicismes détectés :\n${formatOffenders(offenders)}`).toEqual([])
  })

  it('manifests TOML (profile/squad/common/palmares/admin/match_view/timeseries/explorer/session) : aucun anglicisme interdit côté FR', () => {
    const offenders = [
      ...scanManifest('profile.toml'),
      ...scanManifest('squad.toml'),
      ...scanManifest('common.toml'),
      ...scanManifest('palmares.toml'),
      ...scanManifest('admin.toml'),
      ...scanManifest('match_view.toml'),
      ...scanManifest('timeseries.toml'),
      ...scanManifest('explorer.toml'),
      ...scanManifest('session.toml'),
    ]
    expect(offenders, `Anglicismes détectés :\n${formatOffenders(offenders)}`).toEqual([])
  })
})
