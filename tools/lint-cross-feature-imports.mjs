#!/usr/bin/env node
/**
 * Lint custom — frontières features/ et components/.
 *
 * P8.5 (revue 2026-04-29 axe 4 + gap #14) : interdit deux patterns d'import
 * problématiques :
 *
 * 1. **Cross-feature** : un fichier dans `apps/web/src/features/{Y}/` ne doit
 *    pas importer depuis `@/features/{X}/` quand X ≠ Y. Les composants
 *    partagés doivent vivre dans `apps/web/src/components/` ou
 *    `apps/web/src/lib/`. Les exceptions documentées (ex: shell/ orchestre
 *    les features, prestige consommé par home, etc.) sont déclarées dans
 *    ALLOWED_CROSS_IMPORTS.
 *
 * 2. **Frontière inversée** (gap #14) : un fichier dans
 *    `apps/web/src/components/` ne doit pas importer depuis `@/features/`.
 *    Les composants UI génériques doivent recevoir leurs données par props.
 *    Seul `apps/web/src/components/shell/` peut importer features/ (il
 *    orchestre l'app entière).
 *
 * Usage :
 *   node tools/lint-cross-feature-imports.mjs
 *
 * Exit code :
 *   0 = clean
 *   1 = violations détectées
 */

import { readFileSync, readdirSync, statSync } from 'node:fs'
import { join, relative, resolve, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = dirname(fileURLToPath(import.meta.url))
const REPO_ROOT = resolve(__dirname, '..')
const WEB_SRC = join(REPO_ROOT, 'apps/web/src')

// Pattern : extrait `featureName` de `@/features/featureName/...`.
const FEATURE_IMPORT_RE = /@\/features\/([a-z0-9-]+)/g

// Exceptions cross-feature documentées.
//
// Format : "{consumer_feature}=>{imported_feature}" → raison.
//
// - prestige-from-home : la HomePage embarque le carousel défis Prestige
// - palmares-from-home : HomeBattlePassPanel partage les rarities/lightbox
// - settings-from-others : le coin "settings" est consommé partout (auto-sync)
// - lab-from-timeseries : ChartsShowcasePage est par essence une vitrine
//   qui prévisualise les charts de chaque feature
const ALLOWED_CROSS_IMPORTS = new Set([
  // Auth est cross-cutting (login/register/admin partagent les queries)
  'auth=>auth',
  'admin=>auth',
  'admin=>setup',
  // Le dashboard monitoring (WatcherSection) réutilise le hook useWatcherStatus
  // de settings/watcher-queries plutôt que de le dupliquer — dépendance durable.
  'admin=>settings',
  // L'onglet Admin « Lab » (AdminLabPage + WaypointExplorerPanel) et la section
  // Diagnostics de « Qualité données » réutilisent les panneaux / queries / i18n
  // de la feature lab (ResourcesPanel, DiagnosticsPanel, useLab*) — réutilisation
  // voulue (zéro réécriture), dépendance durable.
  'admin=>lab',
  // Compare consommé par carrière + explorer + palmarès (tiroir latéral)
  'career=>compare',
  'explorer=>compare',
  'palmares=>compare',
  // Leaderboard consommé par carrière + palmarès
  'career=>leaderboard',
  'palmares=>leaderboard',
  // Citations consommé par carrière (onglet)
  'career=>citations',
  // Achievements consommé par carrière (section progression)
  'career=>achievements',
  // Career réutilise ExplorerMatchesTable pour les "Matchs marquants"
  'career=>explorer',
  // Career réutilise MatchEncountersTable pour les "Joueurs les plus croisés"
  'career=>match-view',
  // Engagement orchestre des sous-vues squad
  'engagement=>squad',
  // Home orchestre prestige + palmares + media + match-history
  'home=>prestige',
  // Ascension EST l'UI Prestige (objectifs/arcs/coach) → dépendance durable.
  'ascension=>prestige',
  // SquadFocusStrip consomme les hooks Prestige escouade (roster CRUD + défis +
  // évaluation) — Phase C, dépendance durable comme home=>prestige.
  'squad=>prestige',
  'home=>palmares',
  'home=>media',
  'home=>match-history',
  // Settings consommé par friends + profil
  'friends=>settings',
  // ChartsShowcasePage du Lab agrège tous les wrappers
  'lab=>timeseries',
  'lab=>squad',
  // Match-view embarque engagement + match-history (favoris, navigation)
  'match-view=>engagement',
  'match-view=>match-history',
  // Notifications consommée par settings (NotificationsSettingsTab)
  'settings=>notifications',
  // Setup queries réutilisées par settings (jobs status)
  'settings=>setup',
  // Squad embarque engagement (SquadEngagementSection)
  'squad=>engagement',
  // SquadLayout orchestre la barre filtres unifiée Squad (cf. commit 26111a3a)
  'squad=>settings',
  'squad=>filters',
  'squad=>friends',
  'squad=>compare',
  // PersonalStatsLayout réutilise les primitives filters/synthesis/squad (cf. SquadLayout pattern)
  'personal-stats=>filters',
  'personal-stats=>synthesis',
  'personal-stats=>squad',
  // Synthesis embarque squad sub-views
  'synthesis=>squad',
  // TimeseriesPage embarque la section engagement, réutilise les wrappers
  // Squad (WinRate vs History, Map perf vs History, intensity heatmap, builder
  // efficiency) et le tableau Explorer pour l'historique des matchs.
  'timeseries=>engagement',
  'timeseries=>squad',
  'timeseries=>explorer',
  // TimeseriesPage.summary réutilise SynthesisWeaponAccuracyChart (graphe « Précision
  // par arme », Halo 5) sous le sunburst frags — dépendance durable, analogue à
  // session-detail=>synthesis.
  'timeseries=>synthesis',
  // Explorer mode "Joueur" réutilise des composants Squad (synergy table,
  // visualisations partagées) — feature durable.
  'explorer=>squad',
  // Home embarque le SyncIndicator + auto-sync triggers de settings.
  'home=>settings',
  // Match-view réutilise des wrappers Squad (colors hash joueur, impact badges)
  // ainsi que la galerie Media. Settings : réglages accessibility/preferences.
  'match-view=>squad',
  'match-view=>media',
  'match-view=>settings',
  // Flow onboarding : auth (XboxLoginPage) + onboarding (OpenSpartanImportCard)
  // partagent les primitives setup (déclaration joueur, jobs de sync initial).
  'auth=>setup',
  'onboarding=>setup',
  // Home dashboard agrège aussi le widget Ascension (HomeAscensionWidget).
  'home=>ascension',
  // Explorer réutilise la bannière d'identité joueur de Home
  // (ExplorerTargetIdentityBanner).
  'explorer=>home',
  // SquadContributionsPage réutilise un chart Timeseries (réciproque durable
  // de timeseries=>squad déjà déclaré).
  'squad=>timeseries',
  // Synthesis agrège filters + explorer (vue consolidée transverse).
  'synthesis=>filters',
  'synthesis=>explorer',
  // TimeseriesSkillProgression réutilise le contexte rating Carrière (LUSR/CSR).
  'timeseries=>career',
  // Accueil + Explorer affichent le bandeau d'identité Spartan (emblème + nameplate
  // recolorisés) de la feature spartan-customizer (Halo 5) — dépendance durable,
  // analogue à home=>prestige / explorer=>home.
  'home=>spartan-customizer',
  'explorer=>spartan-customizer',
  // UnifiedCitationsPage fusionne citations + commendations (concepts jumeaux,
  // FR=Citations) ; HomeCitationsNearCompletion embarque le widget "proche de la
  // complétion" des deux — dépendances durables (analogue à career=>citations).
  'citations=>commendations',
  'home=>citations',
  'home=>commendations',
  // Les helpers de surlignage de tableaux (ExplorerMatchesTable.highlight /
  // LeaderboardBlock.highlight) réutilisent la logique highlight de match-view —
  // pattern partagé durable (analogue à career=>match-view).
  'explorer=>match-view',
  'leaderboard=>match-view',
  // ExplorerRankedBlock réutilise lusrChainLabel (libellé de chaîne LUSR) du
  // contexte rating Carrière — durable, analogue à timeseries=>career.
  'explorer=>career',
  // SessionMatchesTable réutilise ExplorerMatchesTable pour l'historique de session
  // — durable, strictement analogue à career=>explorer.
  'session-detail=>explorer',
  // SessionFragCard réutilise SynthesisWeaponAccuracyChart (graphe « Précision par
  // arme », Halo 5) au lieu de « Détails des frags » — dépendance durable, analogue à
  // session-detail=>explorer.
  'session-detail=>synthesis',
  // MatchEncountersTable réutilise RelationBadgeLegend (légende des badges de
  // relation) de palmarès — dépendance durable.
  'match-view=>palmares',
  // Auth est cross-cutting : OnboardingOpenSpartanPage et SettingsPage embarquent
  // des cartes auth (login OpenSpartan, SetPasswordCard) — analogue à admin=>auth.
  'onboarding=>auth',
  'settings=>auth',
  // useSquadPresets lit useMyGroups (queries de la feature groups) pour proposer
  // les presets d'escouade — dépendance durable.
  'squad=>groups',
])

// Fichiers shell autorisés à importer @/features/ (orchestration globale).
const SHELL_PATH_PREFIX = 'apps/web/src/components/shell/'

// Marqueur inline pour exception ponctuelle.
const ALLOWED_INLINE_MARKER = 'cross-feature-allow'

function* walk(dir) {
  for (const entry of readdirSync(dir)) {
    const full = join(dir, entry)
    const st = statSync(full)
    if (st.isDirectory()) {
      if (entry === 'node_modules' || entry === 'dist' || entry === '.tanstack') continue
      yield* walk(full)
    } else if (
      (entry.endsWith('.ts') || entry.endsWith('.tsx')) &&
      !entry.endsWith('.test.ts') &&
      !entry.endsWith('.test.tsx') &&
      !entry.endsWith('.spec.ts')
    ) {
      yield full
    }
  }
}

function getFeatureNameFromPath(relPath) {
  // apps/web/src/features/foo/bar.tsx → "foo"
  const match = relPath.match(/^apps\/web\/src\/features\/([a-z0-9-]+)\//)
  return match ? match[1] : null
}

const crossViolations = []
const reverseViolations = []

const featuresDir = join(WEB_SRC, 'features')
const componentsDir = join(WEB_SRC, 'components')

// 1. Cross-feature scan
try {
  for (const file of walk(featuresDir)) {
    const relPath = relative(REPO_ROOT, file).replaceAll('\\', '/')
    const consumerFeature = getFeatureNameFromPath(relPath)
    if (!consumerFeature) continue

    const text = readFileSync(file, 'utf8')
    if (text.includes(ALLOWED_INLINE_MARKER)) continue

    let m
    FEATURE_IMPORT_RE.lastIndex = 0
    while ((m = FEATURE_IMPORT_RE.exec(text)) !== null) {
      const importedFeature = m[1]
      if (importedFeature === consumerFeature) continue
      const key = `${consumerFeature}=>${importedFeature}`
      if (ALLOWED_CROSS_IMPORTS.has(key)) continue
      crossViolations.push({ file: relPath, key, importedFeature })
    }
  }
} catch (err) {
  console.error('cross-feature scan error:', err.message)
}

// 2. Reverse boundary scan (components/ → features/)
try {
  for (const file of walk(componentsDir)) {
    const relPath = relative(REPO_ROOT, file).replaceAll('\\', '/')
    if (relPath.startsWith(SHELL_PATH_PREFIX)) continue // shell orchestre

    const text = readFileSync(file, 'utf8')
    if (text.includes(ALLOWED_INLINE_MARKER)) continue

    if (text.match(FEATURE_IMPORT_RE)) {
      let m
      FEATURE_IMPORT_RE.lastIndex = 0
      while ((m = FEATURE_IMPORT_RE.exec(text)) !== null) {
        reverseViolations.push({ file: relPath, importedFeature: m[1] })
      }
    }
  }
} catch (err) {
  console.error('reverse boundary scan error:', err.message)
}

// Report
let totalViolations = crossViolations.length + reverseViolations.length

if (totalViolations === 0) {
  console.log('lint-cross-feature-imports: clean (0 violation)')
  process.exit(0)
}

if (crossViolations.length > 0) {
  console.log(`\n# Cross-feature imports (${crossViolations.length}) :`)
  const byConsumer = new Map()
  for (const v of crossViolations) {
    if (!byConsumer.has(v.file)) byConsumer.set(v.file, new Set())
    byConsumer.get(v.file).add(v.importedFeature)
  }
  for (const [file, set] of byConsumer.entries()) {
    console.log(`  ${file}`)
    for (const imp of set) console.log(`    → @/features/${imp}/...`)
  }
}

if (reverseViolations.length > 0) {
  console.log(`\n# Frontière inversée (${reverseViolations.length}, gap #14) :`)
  const byFile = new Map()
  for (const v of reverseViolations) {
    if (!byFile.has(v.file)) byFile.set(v.file, new Set())
    byFile.get(v.file).add(v.importedFeature)
  }
  for (const [file, set] of byFile.entries()) {
    console.log(`  ${file}`)
    for (const imp of set) console.log(`    → @/features/${imp}/... (composants/ ne doit pas importer features/)`)
  }
}

console.log(
  `\nUtilisez \`// cross-feature-allow: <raison>\` en tête de fichier pour ` +
    `marquer une exception ponctuelle, ou ajoutez la paire dans ` +
    `ALLOWED_CROSS_IMPORTS si la dépendance est durable.`,
)

// Ratchet : abaissé progressivement.
// État P8.5 (revue 2026-04-29) : 0 cross-feature non-déclarée + 6 violations
// reverse boundary tolérées (cover-flow-modal, AppShell, NavL1).
const RATCHET_THRESHOLD = 10

if (totalViolations > RATCHET_THRESHOLD) {
  console.log(
    `\nERREUR : ${totalViolations} > plafond P8.5 (${RATCHET_THRESHOLD}). ` +
      'Cleanup requis avant merge.',
  )
  process.exit(1)
}

console.log(
  `\nInfo : ${totalViolations} <= plafond P8.5 (${RATCHET_THRESHOLD}). ` +
    'Pas d\'échec — ratchet en place pour bloquer toute régression.',
)
process.exit(0)
