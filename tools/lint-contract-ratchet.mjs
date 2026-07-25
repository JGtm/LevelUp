#!/usr/bin/env node
/**
 * Lint custom — garde-fou anti-doublon contrat OpenAPI ↔ view-model front.
 *
 * Phase D du plan PLAN_WEB_API_TYPES_MIGRATION : un type manuel
 * (`export interface X`) dans `apps/web/src/lib/api/types.ts` ne doit PAS
 * doublonner un schéma du contrat OpenAPI (`components['schemas']['X']`, présent
 * dans `generated.ts`) — sauf s'il s'agit d'un view-model légitime (champs calculés
 * côté client : badge_image_url, *_fr/*_en, match_count_filtered…) ou d'un type
 * producteur (Input/Request construit par le front). Sinon il faut le SHIMER
 * (`export type X = components['schemas']['X']`) pour éviter la dérive contrat↔front.
 *
 * Échoue dès qu'une NOUVELLE collision (interface manuelle homonyme d'un schéma
 * contrat) apparaît hors baseline → force une décision consciente : shimer le type,
 * ou l'ajouter à BASELINE_COLLISIONS avec sa raison. La baseline ne doit que
 * DÉCROÎTRE (re-shim) ou croître par ajout justifié ; une entrée obsolète (collision
 * disparue) fait aussi échouer (hygiène).
 *
 * Usage :   node tools/lint-contract-ratchet.mjs
 * Exit code : 0 = clean · 1 = violation (nouvelle collision ou baseline obsolète)
 */

import { readFileSync } from 'node:fs'
import { join, resolve, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = dirname(fileURLToPath(import.meta.url))
const REPO_ROOT = resolve(__dirname, '..')
const TYPES_TS = join(REPO_ROOT, 'apps/web/src/lib/api/types.ts')
const GENERATED_TS = join(REPO_ROOT, 'apps/web/src/lib/api/generated.ts')

// Collisions LÉGITIMES connues (view-models enrichis OU types Input producteur).
// Post-Étape 3 + re-shim Phase D. Trié alpha.
const BASELINE_COLLISIONS = new Set([
  // V72-01 H7 (2026-07-25) : Group/GroupMember RETIRÉS de la baseline — le schéma généré
  // est devenu fidèle (tag `enum:"owner,member"` sur Role, `nullable:"false"` sur Members,
  // invariant nil → [] tenu par groupstore) et types.ts les ré-exporte du contrat.
  'AdminJobsResponse',
  'AdminLogEntry',
  'AdminSchedulerStatusResponse',
  'AsyncJobStatus',
  'BackupStatusResponse',
  // V8b (2026-07-07) : CareerEncountersResponse / CareerTopMatchesResponse retirés —
  // migrés en ré-export du contrat généré (plus de collision hand-written).
  'CareerHighlightMatchesResponse',
  'CareerHistoryPoint',
  'CareerLusrCheckpoint',
  'CareerLusrSection',
  'CareerPageResponse',
  'CareerSummary',
  'CascadeInput',
  'CompareRequest',
  'CompareResponse',
  'DeviceFlowStartResponse',
  'EngagementTimeseriesRequest',
  'ExplorerMatchRow',
  'ExplorerMatchesQueryRequest',
  'ExplorerMatchesQuerySummary',
  'ExplorerPlayerQueryRequest',
  'FilterContextInput',
  'HealthResponse',
  // 2026-07-10 (A3.5) : LabWaypointResponse retiré de la baseline — le type manuel
  // ET son endpoint /lab/waypoint sont supprimés avec le retrait du Lab (DC-9).
  'MatchCombatTab',
  'MatchEncounterBadge',
  'MatchHighlightEvent',
  'MatchHistoryQueryRequest',
  'MatchHistoryRow',
  'MatchScoreboardRow',
  'MatchTeamTab',
  'MatchViewResponse',
  'MatchWeaponKill',
  'MediaItemRow',
  'MediaLikeRequest',
  'MediaLikeResponse',
  'MediaPageResponse',
  'NormalizedPlayerStats',
  'PaginationRequest',
  'PeriodInput',
  'SchedulerSnapshot',
  'SeasonPassContentSummary',
  'SessionContextRequest',
  'SessionContextResponse',
  'SessionsInput',
  'SettingsResponse',
  'SquadIntensityProfile',
  'SquadTimeseriesPoint',
  'SquadWeaponBar',
  'TeammateKPIs',
  'TeammatesPageResponse',
])

/** Noms des `export interface X` de types.ts. */
function handWrittenInterfaceNames(typesTs) {
  const out = new Set()
  for (const m of typesTs.matchAll(/^export interface ([A-Za-z0-9_]+)/gm)) {
    out.add(m[1])
  }
  return out
}

/** Noms des schémas du contrat (clés sous components.schemas de generated.ts). */
function contractSchemaNames(generatedTs) {
  const out = new Set()
  let inSchemas = false
  let depth = 0
  for (const line of generatedTs.split('\n')) {
    if (!inSchemas) {
      if (/^ {4}schemas: \{/.test(line)) {
        inSchemas = true
        depth = 1
      }
      continue
    }
    const m = line.match(/^ {8}([A-Za-z0-9_]+)\??:/)
    if (m && depth === 1) out.add(m[1])
    depth += (line.match(/\{/g) ?? []).length - (line.match(/\}/g) ?? []).length
    if (depth <= 0) break
  }
  return out
}

const interfaces = handWrittenInterfaceNames(readFileSync(TYPES_TS, 'utf8'))
const schemas = contractSchemaNames(readFileSync(GENERATED_TS, 'utf8'))

// Sanity : l'extraction marche (sinon le garde-fou serait vacuant).
if (interfaces.size < 50 || schemas.size < 100) {
  console.error(
    `lint-contract-ratchet: extraction anormale (interfaces=${interfaces.size}, schemas=${schemas.size}). ` +
      `Le parser est cassé — vérifier la structure de types.ts / generated.ts.`,
  )
  process.exit(1)
}

const collisions = [...interfaces].filter((n) => schemas.has(n)).sort()
const novel = collisions.filter((n) => !BASELINE_COLLISIONS.has(n))
const stale = [...BASELINE_COLLISIONS].filter((n) => !collisions.includes(n)).sort()

let failed = false
if (novel.length > 0) {
  failed = true
  console.error(
    `lint-contract-ratchet: ${novel.length} interface(s) manuelle(s) doublonnant un schéma OpenAPI hors baseline :\n` +
      novel.map((n) => `  - ${n}`).join('\n') +
      `\n→ SHIMER (export type X = components['schemas']['X']) si le contrat couvre fidèlement le type,\n` +
      `  ou ajouter à BASELINE_COLLISIONS dans tools/lint-contract-ratchet.mjs avec la raison\n` +
      `  (view-model à champs calculés / type Input producteur).`,
  )
}
if (stale.length > 0) {
  failed = true
  console.error(
    `lint-contract-ratchet: ${stale.length} entrée(s) BASELINE_COLLISIONS obsolète(s) (plus de collision — type shimé/supprimé) :\n` +
      stale.map((n) => `  - ${n}`).join('\n') +
      `\n→ retirer ces noms de BASELINE_COLLISIONS.`,
  )
}

if (failed) {
  process.exit(1)
}

console.log(
  `lint-contract-ratchet: clean (${collisions.length} collisions view-model/Input baseline, ` +
    `${interfaces.size} interfaces manuelles, ${schemas.size} schémas contrat).`,
)
