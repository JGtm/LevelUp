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
 */
import { describe, it, expect } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { join, resolve } from 'node:path'

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
  'LabWaypointResponse',        // GET /lab/waypoint — outil interne Lab
  'FilterMatchIdsResponse',     // POST /filters/match-ids — wrapper trivial { match_ids }
  'SetupStatusResponse',        // @deprecated artefact mort, à supprimer avec useSetupStatus
  'DeviceFlowStartResponse',    // POST /auth/device-flow/start — champ deprecated alias
  'SettingsResponse',           // GET /settings — large struct settings, non migrée
  'CareerPageResponse',         // sous-types view-model (CareerSummary/LusrSection) hors schéma
  'CareerHighlightMatchesResponse', // agrégat composite ExplorerMatchRow + cascade counts
  'PaginatedResponse',          // générique <T> — wrapper de pagination
  'HomePageResponse',           // page composite — sous-types view-model locaux
  'TeammatesPageResponse',      // page composite squad — sous-types view-model locaux
  'SynthesisPageResponse',      // page composite — sous-types view-model locaux
  'MediaPageResponse',          // page composite média
  'MediaLikeResponse',          // PATCH /media/likes — non migré Huma
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
  'SquadPageV2Response',        // features/squad/v2/types.ts — page squad V2
  'FieldMappingsResponse',      // lib/i18n/fieldMappings.ts — mappings TOML backend-driven
])

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
})
