/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n°6 + lot V8d, 2026-07-07) : aucun NOUVEAU type de RÉPONSE
 * API écrit à la main dans `apps/web/src`.
 *
 * Contexte : un type de réponse hand-written qui diverge du contrat réel = un
 * `undefined` silencieux à l'écran (bug prouvé A2 : `CareerTopMatchesResponse`
 * déclaré `{ items }` alors que le backend renvoie `{ best_matches, worst_matches }`,
 * et `data.top_matches_preview` lu mais absent du contrat Go). La source de vérité
 * est `apps/web/src/lib/api/generated.ts` (dérivé d'openapi.yaml) : tout type de
 * réponse DOIT en être ré-exporté (`type X = components['schemas'][...]`).
 *
 * Ce test détecte toute déclaration MANUELLE d'un `interface *Response` ou
 * `type *Response = { ... }` (alias vers autre chose que `components['schemas']`).
 * L'ensemble détecté doit être EXACTEMENT l'allowlist ci-dessous — une allowlist
 * DÉCROISSANTE datée : chaque entrée est un reste légitime (view-model composite,
 * champ live-fetch hors OpenAPI, endpoint non encore migré Huma) à résorber.
 *
 * Morsure prouvée dans les 2 sens :
 *   1. AJOUT — un nouveau `interface FooResponse` hors allowlist → rouge (le nom
 *      apparaît dans `unexpected`).
 *   2. RETRAIT — migrer une entrée vers generated.ts sans retirer sa ligne
 *      d'allowlist → rouge (le nom apparaît dans `stale`, self-check).
 *
 * DURCISSEMENT V72-01 / H7 (2026-07-25) — le garde-rail ci-dessus est NÉGATIF (il
 * interdit les nouveaux types manuels) : il ne dit rien de la VALIDITÉ des ré-exports
 * existants. Depuis H6 le contrat est GÉNÉRÉ, donc un renommage côté Go peut rendre un
 * `components['schemas']['X']` orphelin — TypeScript le résout alors en `unknown`
 * (index d'une clé absente sur un type d'interface indexé) sans forcément casser tous
 * les appelants. Deux assertions POSITIVES sont ajoutées :
 *   3. RÉSOLUTION — tout `components['schemas']['X']` cité dans `src/` désigne un schéma
 *      RÉELLEMENT présent dans generated.ts.
 *   4. TYPES CRITIQUES — les réponses de page les plus exposées restent des ré-exports
 *      du contrat (interdiction de les re-hand-writer).
 */
import { describe, it, expect } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { join, resolve } from 'node:path'
import { extractContractSurface } from './contractSurface'

// Déclaration hand-written d'un type de réponse :
//   - `export interface XxxResponse {`  (toujours manuel)
//   - `export type XxxResponse = ` NON suivi immédiatement de `components[`
//     (un ré-export du contrat généré est autorisé, un alias local ne l'est pas)
const INTERFACE_RE = /export\s+interface\s+(\w*Response)\b/g
const TYPE_ALIAS_RE = /export\s+type\s+(\w*Response)\s*=\s*(.*)/g

// Allowlist DÉCROISSANTE (2026-07-07, lot V8d). Chaque entrée = un type de réponse
// encore défini à la main, avec sa raison. À migrer vers generated.ts au fil de l'eau.
// NE PAS ajouter d'entrée sans justification ; retirer l'entrée dès la migration.
const ALLOWLIST = new Set<string>([
  // --- lib/api/types.ts : view-models / endpoints hors OpenAPI Huma ---
  'SessionContextResponse',     // POST /session/context — non migré Huma
  'HealthResponse',             // GET /health — endpoint infra hors openapi.yaml
  'FilterMatchIdsResponse',     // POST /filters/match-ids — wrapper trivial { match_ids }
  'DeviceFlowStartResponse',    // POST /auth/device-flow/start — champ deprecated alias
  'SettingsResponse',           // GET /settings — large struct settings, non migrée
  'CareerPageResponse',         // sous-types view-model (CareerSummary/LusrSection) hors schéma
  'CareerHighlightMatchesResponse', // agrégat composite ExplorerMatchRow + cascade counts
  'PaginatedResponse',          // générique <T> — wrapper de pagination
  'HomePageResponse',           // page composite — sous-types view-model locaux
  'TeammatesPageResponse',      // page composite squad — sous-types view-model locaux
  'SynthesisPageResponse',      // page composite — sous-types view-model locaux
  'MediaPageResponse',          // page composite média
  'MediaUploadResponse',        // POST /media/upload — non migré Huma
  'MatchViewResponse',          // page composite Match View — nombreux tabs view-model
  'CompareResponse',            // POST /pages/compare — NormalizedPlayerStats live-fetch
  'BackupStatusResponse',       // GET /settings/backup/status — pkg duckdbbackup hors Huma
  'AdminSchedulerStatusResponse', // GET /admin/monitoring/scheduler — types pkg scheduler
  'AdminJobsResponse',          // GET /admin/monitoring/jobs — wrapper AsyncJobStatus[]
  // --- fichiers feature-locaux ---
  'StreaksResponse',            // features/ascension/types.ts — Ascension V2
  'RecordsResponse',            // features/ascension/types.ts
  'MilestonesResponse',         // features/ascension/types.ts
  'AdminTitlesListResponse',    // features/admin/titles/queries.ts
  'ProposalsListResponse',      // features/coach/types.ts — coach proactif
  'AcceptResponse',             // features/coach/types.ts
  'DismissResponse',            // features/coach/types.ts
  'NotificationsListResponse',  // features/notifications/queries.ts — alias NotificationListResult
  'ImportStartResponse',        // features/onboarding/queries.ts — OpenSpartan import
  'FieldMappingsResponse',      // lib/i18n/fieldMappings.ts — mappings TOML backend-driven
])

// Réponses CRITIQUES (V72-01 H7) : pages à fort trafic + endpoints d'amorçage. Elles
// DOIVENT rester des ré-exports du contrat généré (`export type X = components[...]`).
// Liste POSITIVE, croissante : on y ajoute un type dès qu'il quitte l'ALLOWLIST.
const CRITICAL_CONTRACT_TYPES = [
  'BootstrapResponse',
  'PlayersListResponse',
  'MatchHistoryPageResponse',
  'SessionPageResponse',
  'MedalsPageResponse',
  'CitationsPageResponse',
  'AchievementsPageResponse',
  'SeasonPassPageResponse',
  'TimeseriesPageResponse',
  'RelationsPageResponse',
  'LeaderboardResponse',
  'ExplorerPlayerQueryResponse',
  'ExplorerMatchesQueryResponse',
  'EngagementTimeseriesResponse',
  // Régression historique A2 (cf. en-tête) : contrat {best_matches, worst_matches}.
  'CareerTopMatchesResponse',
  'CareerEncountersResponse',
  'GamertagSearchResponse',
  'DeviceFlowStatusResponse',
  'WatcherStatusResponse',
  'LoginResponse',
  // Sorti de l'ALLOWLIST le 2026-08-03 : le type manuel déclarait `total_likers`
  // optionnel alors que le DTO Go l'émet toujours (compteur de likers lu à
  // l'écran) — exactement la classe de bug A2.
  'MediaLikeResponse',
] as const

// `export type X = components['schemas']['Y']` (le suffixe `& { … }` d'un view-model
// enrichi est toléré : la base reste le contrat).
const CONTRACT_REEXPORT_RE =
  /export\s+type\s+(\w+)\s*=\s*components\['schemas'\]\['(\w+)'\]/g
// Toute citation d'un schéma du contrat, où qu'elle soit (types.ts, features/…).
const SCHEMA_REF_RE = /components\['schemas'\]\['(\w+)'\]/g

function walk(dir: string): string[] {
  const out: string[] = []
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const full = join(dir, entry.name)
    if (entry.isDirectory()) {
      if (entry.name === 'node_modules' || entry.name === 'generated') continue
      out.push(...walk(full))
    } else if (/\.(ts|tsx)$/.test(entry.name) && !/\.test\.tsx?$/.test(entry.name)) {
      out.push(full)
    }
  }
  return out
}

function collectHandWrittenResponses(files: string[]): Set<string> {
  const found = new Set<string>()
  for (const f of files) {
    const src = readFileSync(f, 'utf8')
    for (const m of src.matchAll(INTERFACE_RE)) {
      found.add(m[1])
    }
    for (const m of src.matchAll(TYPE_ALIAS_RE)) {
      const name = m[1]
      const rhs = (m[2] ?? '').trim()
      // Un ré-export du contrat généré est la forme AUTORISÉE : on ne le compte pas.
      if (rhs.startsWith("components['schemas']") || rhs.startsWith('components["schemas"]')) {
        continue
      }
      found.add(name)
    }
  }
  return found
}

/** Schémas du contrat cités par le front : `fichier → { nom du schéma }`. */
function collectSchemaRefs(files: string[]): Map<string, Set<string>> {
  const byFile = new Map<string, Set<string>>()
  for (const f of files) {
    const src = readFileSync(f, 'utf8')
    const names = new Set<string>()
    for (const m of src.matchAll(SCHEMA_REF_RE)) names.add(m[1])
    if (names.size > 0) byFile.set(f, names)
  }
  return byFile
}

/** Ré-exports du contrat déclarés dans types.ts : `nom exporté → nom de schéma`. */
function collectContractReexports(typesTs: string): Map<string, string> {
  const out = new Map<string, string>()
  for (const m of typesTs.matchAll(CONTRACT_REEXPORT_RE)) out.set(m[1], m[2])
  return out
}

describe('garde-rail types de réponse API (V8d — source unique generated.ts)', () => {
  const srcRoot = resolve(process.cwd(), 'src')
  const files = walk(srcRoot)
  const found = collectHandWrittenResponses(files)

  it('aucun NOUVEAU interface/type *Response manuel hors allowlist', () => {
    const unexpected = [...found].filter((n) => !ALLOWLIST.has(n)).sort()
    expect(
      unexpected,
      `Type(s) de réponse hand-written non allowlistés : ${unexpected.join(', ')}. ` +
        `Ré-exporter depuis generated.ts (type X = components['schemas'][...]) ` +
        `ou, si aucun schéma openapi n'existe, ajouter au bloc ALLOWLIST daté avec justification.`,
    ).toEqual([])
  })

  it('allowlist décroissante : aucune entrée périmée (self-check)', () => {
    const stale = [...ALLOWLIST].filter((n) => !found.has(n)).sort()
    expect(
      stale,
      `Entrée(s) d'allowlist sans type hand-written correspondant (migré ? supprimé ?) : ` +
        `${stale.join(', ')}. Retirer la ligne de l'ALLOWLIST.`,
    ).toEqual([])
  })

  // --- Durcissement H7 : assertions POSITIVES sur le contrat généré -----------------
  const generated = readFileSync(join(srcRoot, 'lib/api/generated.ts'), 'utf8')
  const schemas = new Set(extractContractSurface(generated).schemas)
  const refsByFile = collectSchemaRefs(files)
  const reexports = collectContractReexports(readFileSync(join(srcRoot, 'lib/api/types.ts'), 'utf8'))

  it("extraction non vacuante (schémas du contrat, citations front, ré-exports)", () => {
    expect(schemas.size, 'schémas du contrat').toBeGreaterThan(400)
    expect(refsByFile.size, 'fichiers citant un schéma').toBeGreaterThan(0)
    expect(reexports.size, 'ré-exports de types.ts').toBeGreaterThan(200)
  })

  it('tout schéma cité par le front existe dans generated.ts (aucun ré-export orphelin)', () => {
    const orphans: string[] = []
    for (const [file, names] of refsByFile) {
      const rel = file.slice(srcRoot.length + 1).replaceAll('\\', '/')
      for (const n of [...names].sort()) {
        if (!schemas.has(n)) orphans.push(`${rel} → components['schemas']['${n}']`)
      }
    }
    expect(
      orphans,
      `Citation(s) d'un schéma ABSENT du contrat généré :\n  ${orphans.join('\n  ')}\n` +
        `TypeScript résout un index inexistant en \`unknown\` sans casser tous les appelants : ` +
        `le front lirait des champs qui n'existent pas. Repointer le ré-export vers le schéma ` +
        `réellement émis (le contrat est GÉNÉRÉ : voir apps/go-api/api/openapi.yaml).`,
    ).toEqual([])
  })

  it('les réponses CRITIQUES restent des ré-exports du contrat', () => {
    const broken = CRITICAL_CONTRACT_TYPES.filter((n) => {
      const schema = reexports.get(n)
      return schema === undefined || !schemas.has(schema)
    }).sort()
    expect(
      broken,
      `Réponse(s) critique(s) qui ne sont plus des ré-exports valides du contrat : ${broken.join(', ')}.\n` +
        `Ces types portent des pages entières : les re-hand-writer rouvre la classe de bugs A2 ` +
        `(champ lu côté front, absent côté Go). Rétablir ` +
        `\`export type X = components['schemas']['X']\` dans lib/api/types.ts.`,
    ).toEqual([])
  })
})
