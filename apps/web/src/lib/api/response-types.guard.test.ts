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
  'BrowserLoginStartResponse',  // GET /auth/xbox/login (2026-11-02, L21.00) — route montée sous la capacité LoginViaBrowser seulement, donc hors du document généré (comme /desktop/strings) ; à migrer si le générateur monte un jour les routes de bureau
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

// Types d'API écrits à la main pour des routes que le contrat généré ne porte pas (2026-11-03, lot L22) : `openapi-gen`
// décrit le routeur du profil démo, qui ne monte aucune route sous capacité desktop ou hébergée (import depuis le site,
// export vers l'application de bureau, rapport de diagnostic, connexion par le navigateur). Faire entrer ces routes dans
// le contrat demande de séparer le routeur de contrat du routeur de démonstration : six garde-rails comparent
// aujourd'hui les deux dans les deux sens (routes nues, routes documentées et enregistrées, document partagé, absence
// hors du profil desktop). Liste DÉCROISSANTE : un type qui entre dans le contrat sort d'ici.
const MANUAL_CAPABILITY_TYPES = [
  { name: 'ExportToken', file: 'features/settings/exportForDesktopQueries.ts', route: 'POST /export/desktop/token' },
  { name: 'SiteImportStatus', file: 'features/settings/siteImportQueries.ts', route: 'GET et POST /import/site' },
  { name: 'SiteImportRequest', file: 'features/settings/siteImportQueries.ts', route: 'POST /import/site' },
  { name: 'DiagnosticReport', file: 'components/shell/diagnosticReportQueries.ts', route: 'POST /desktop/diagnostic' },
  { name: 'ReplayRegenStatus', file: 'components/replay-regen/queries.ts', route: 'GET /desktop/replay-regen et POST .../start|pause|resume' },
] as const

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

// --- Extension (plan architecture, étape 7.4) : TOUS les schémas, pas seulement les *Response ---
//
// Un type écrit à la main sous le nom d'un schéma du contrat dit une chose que le contrat dit autrement :
// c'est la classe de bug A2 (champ lu côté front, absent côté Go), pour chaque nom et pas seulement pour
// les réponses. Dans `lib/api/types.ts`, toute déclaration manuelle homonyme d'un schéma est donc interdite
// hors de l'ALLOWLIST des réponses (ci-dessus) et de KEPT_MANUAL_HOMONYMS (ci-dessous). Une déclaration
// dérivée du contrat (`components['schemas'][…]`, y compris `Omit<…> & { … }`) n'est pas manuelle. Hors de
// `types.ts`, le nombre de déclarations homonymes est plafonné à sa valeur du 2026-11-13 (dette antérieure,
// à résorber, jamais relevée).
const DECLARATION_RE = /^(?:export\s+)?(interface|type)\s+(\w+)/gm
const NEXT_DECLARATION_RE = /\n(?:export\s+|type\s|interface\s|const\s|function\s)/

/** Plafond des homonymes manuels hors `lib/api/types.ts` (mesuré le 2026-11-13). */
const HOMONYMS_OUTSIDE_TYPES_CEILING = 78

// Types NON *Response de `lib/api/types.ts` laissés à la main (étape 7.4, 2026-11-13). Liste décroissante :
// chaque entrée dit pourquoi le contrat généré ne peut pas être ré-exporté tel quel, et reste tant que la cause tient.
const KEPT_MANUAL_HOMONYMS: Readonly<Record<string, string>> = {
  PeriodInput:
    "producteur : le contrat décrit la forme de RÉPONSE (champs requis, DTO Go sans omitempty) ; le store et le lien ?f= portent des fenêtres partielles",
  SessionsInput: "producteur, même cause que PeriodInput (picked_* absents tant que rien n'est choisi)",
  FilterContextInput:
    "producteur : period, sessions et cascade facultatifs côté front, requis dans le contrat ; les rendre requis change les corps envoyés et le format du lien ?f=",
  EngagementTimeseriesRequest: 'porte FilterContextInput (producteur)',
  MatchHistoryQueryRequest:
    "porte FilterContextInput ; le contrat omet picked_solo_session_labels, que le serveur lit (domain/match_history.go)",
  ExplorerMatchesQueryRequest:
    "porte FilterContextInput ; le contrat omet 17 champs de requête que le serveur lit (tris, paliers, plages de dates, scopes)",
  ExplorerPlayerQueryRequest:
    "le contrat porte un champ filters que la structure Go n'a pas, et omet page et target_xuid qu'elle lit (domain/explorer.go)",
  CompareRequest:
    "le contrat exige title_slug, que la structure Go n'a pas et que le front n'envoie pas (domain/compare.go)",
  MediaItemRow:
    "view-model normalisé (null au lieu d'absent, section et compteurs par défaut), fabriqué par normalizeMediaItem ; le schéma MediaItem du contrat est la forme brute",
}

interface ManualDeclaration {
  file: string
  name: string
}

/** Déclarations manuelles (interface, ou alias qui ne dérive pas du contrat) dont le nom est un schéma. */
function manualSchemaHomonyms(
  files: string[],
  schemaNames: ReadonlySet<string>,
  root: string,
): ManualDeclaration[] {
  const out: ManualDeclaration[] = []
  for (const file of files) {
    const src = readFileSync(file, 'utf8')
    for (const m of src.matchAll(DECLARATION_RE)) {
      const [, kind, name] = m
      if (!schemaNames.has(name)) continue
      if (kind === 'type') {
        const rest = src.slice(m.index + m[0].length)
        const stop = rest.search(NEXT_DECLARATION_RE)
        const body = stop === -1 ? rest : rest.slice(0, stop)
        if (body.includes("components['schemas']")) continue
      }
      out.push({ file: file.slice(root.length + 1).replaceAll('\\', '/'), name })
    }
  }
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

  it('types manuels de routes sous capacité : déclarés là où la liste le dit, et absents du contrat généré', () => {
    const generatedSchemas = new Set(extractContractSurface(readFileSync(join(srcRoot, 'lib/api/generated.ts'), 'utf8')).schemas)
    for (const t of MANUAL_CAPABILITY_TYPES) {
      const src = readFileSync(join(srcRoot, t.file), 'utf8')
      expect(
        new RegExp(String.raw`export\s+(interface|type)\s+${t.name}\b`).test(src),
        `${t.name} (${t.route}) n'est plus déclaré à la main dans ${t.file} : retirer l'entrée de MANUAL_CAPABILITY_TYPES.`,
      ).toBe(true)
      expect(
        generatedSchemas.has(t.name),
        `${t.name} (${t.route}) existe maintenant dans le contrat généré : le ré-exporter (type X = components['schemas'][...]) et retirer l'entrée de MANUAL_CAPABILITY_TYPES.`,
      ).toBe(false)
    }
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

  // --- Extension 7.4 : tous les noms de schéma ------------------------------------------
  const typesFile = join(srcRoot, 'lib/api/types.ts')
  const homonymsInTypes = manualSchemaHomonyms([typesFile], schemas, srcRoot).map((d) => d.name)

  it('lib/api/types.ts : aucune déclaration manuelle homonyme d’un schéma hors des deux listes', () => {
    const unexpected = homonymsInTypes.filter((n) => !ALLOWLIST.has(n) && !(n in KEPT_MANUAL_HOMONYMS)).sort()
    expect(
      unexpected,
      `Déclaration(s) manuelle(s) homonyme(s) d'un schéma du contrat : ${unexpected.join(', ')}. ` +
        `Ré-exporter le schéma (type X = components['schemas']['X']), ou le dériver d'un schéma qui le porte ` +
        `réellement ; sinon l'ajouter à KEPT_MANUAL_HOMONYMS avec la cause écrite.`,
    ).toEqual([])
  })

  it('KEPT_MANUAL_HOMONYMS décroissante : aucune entrée périmée', () => {
    const stale = Object.keys(KEPT_MANUAL_HOMONYMS)
      .filter((n) => !homonymsInTypes.includes(n))
      .sort()
    expect(
      stale,
      `Entrée(s) de KEPT_MANUAL_HOMONYMS sans déclaration manuelle correspondante dans types.ts : ${stale.join(', ')}. ` +
        `Retirer la ligne.`,
    ).toEqual([])
  })

  it('hors lib/api/types.ts : le nombre de déclarations homonymes d’un schéma ne dépasse pas le plafond daté', () => {
    const outside = manualSchemaHomonyms(
      files.filter((f) => f !== typesFile && !f.endsWith(join('lib', 'api', 'generated.ts'))),
      schemas,
      srcRoot,
    )
    expect(
      outside.length,
      `${outside.length} déclaration(s) homonyme(s) hors types.ts pour un plafond de ${HOMONYMS_OUTSIDE_TYPES_CEILING} :\n  ` +
        outside.map((d) => `${d.file} ${d.name}`).join('\n  '),
    ).toBeLessThanOrEqual(HOMONYMS_OUTSIDE_TYPES_CEILING)
  })

  it('auto-preuve : la détection voit interface et alias manuels, ignore les dérivations du contrat', () => {
    const detect = (code: string): string[] => {
      const names = new Set(['Foo'])
      const found: string[] = []
      for (const m of code.matchAll(DECLARATION_RE)) {
        const [, kind, name] = m
        if (!names.has(name)) continue
        if (kind === 'type') {
          const rest = code.slice(m.index + m[0].length)
          const stop = rest.search(NEXT_DECLARATION_RE)
          if ((stop === -1 ? rest : rest.slice(0, stop)).includes("components['schemas']")) continue
        }
        found.push(name)
      }
      return found
    }
    expect(detect('export interface Foo { a: 1 }')).toEqual(['Foo'])
    expect(detect('export type Foo = { a: 1 }')).toEqual(['Foo'])
    expect(detect("export type Foo = components['schemas']['Foo']")).toEqual([])
    expect(detect("export type Foo = Omit<components['schemas']['Foo'], 'a'> & { b: 1 }")).toEqual([])
    expect(detect("export type Foo =\n  components['schemas']['Bar']\n\nexport interface Baz {}")).toEqual([])
    expect(detect('export interface Other { a: 1 }')).toEqual([])
  })
})
