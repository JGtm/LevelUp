#!/usr/bin/env node
/**
 * Lint custom — frontières features/ et components/.
 *
 * Trois frontières, plafond 0 (aucune violation tolérée, aucun seuil à relever) :
 *
 * 1. **Cross-feature** : un fichier dans `apps/web/src/features/{Y}/` ne doit
 *    pas importer depuis `features/{X}/` quand X ≠ Y, que l'import soit écrit
 *    par l'alias (`@/features/X/...`) ou par un chemin relatif (`../../X/...`).
 *    Les composants partagés vivent dans `apps/web/src/components/` ou
 *    `apps/web/src/lib/`. Les dépendances durables sont déclarées dans
 *    ALLOWED_CROSS_IMPORTS ; une exception ponctuelle porte le marqueur
 *    `cross-feature-allow: <raison>` SUR la ligne de l'import (ou sur le
 *    commentaire `//` qui la précède immédiatement), jamais sur le fichier.
 *
 * 2. **`_shared` est un socle, pas une feature** : `features/_shared/` est lu par
 *    toutes les features et ne doit donc dépendre d'aucune. Tout import d'une
 *    autre feature depuis `_shared` est une violation (aucune paire déclarable) ;
 *    sans cela `squad -> _shared -> squad` forme un cycle que ni le lint ni
 *    le compilateur ne voient. Les features importent `_shared` librement.
 *
 * 3. **Frontière inversée** (gap #14) : un fichier dans
 *    `apps/web/src/components/` ne doit pas importer depuis `features/`.
 *    Les composants UI génériques reçoivent leurs données par props.
 *    Seul `apps/web/src/components/shell/` peut importer features/ (il
 *    orchestre l'app entière). Les imports de `features/_shared` depuis
 *    `components/` sont comptés comme les autres ; la liste
 *    REVERSE_SHARED_ALLOWED nomme les fichiers qui en gardent un en attendant
 *    que leur dépendance descende.
 *
 * Deux dérives sont aussi des erreurs : une entrée d'ALLOWED_CROSS_IMPORTS
 * (ou de REVERSE_SHARED_ALLOWED) qu'aucun import ne sert plus, et un marqueur
 * `cross-feature-allow` qui ne couvre aucun import hors allowlist (ou sans raison).
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

// Le socle partagé : lu par toutes les features, ne dépend d'aucune.
const SHARED_FEATURE = '_shared'

// Une cible de feature : `features/<nom>` suivi du chemin du module importé. Le chemin est
// gardé pour qu'une exception puisse porter sur un module nommé plutôt que sur une
// feature entière (cf. ALLOWED_CROSS_IMPORTS ci-dessous).
const FEATURE_TARGET_RE = /^features\/([a-z0-9_-]+)((?:\/[A-Za-z0-9._-]+)*)$/

// Un spécificateur d'import entre guillemets : alias `@/...` ou chemin relatif. Les
// importations statiques, dynamiques (`import('...')`) et les réexportations (`export ...
// from '...'`) y passent toutes.
const QUOTED_SPECIFIER_RE = /(['"`])((?:@\/|\.{1,2}\/)[^'"`\s]*)\1/g

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
  'admin=>auth',
  'admin=>setup',
  // Le dashboard monitoring (WatcherSection) réutilise le hook useWatcherStatus
  // de settings/watcher-queries plutôt que de le dupliquer — dépendance durable.
  'admin=>settings',
  // Le rejeu 2D lit les réglages d'instance des sons d'armes (variation, distance)
  // via useSettings — même source que la section d'admin qui les édite, zéro
  // duplication de query. Dépendance durable (2026-08-16, chantier sons-rejeu).
  'match-replay=>settings',
  // L'assistant de premier lancement propose le réglage de l'archive des films par le MÊME
  // panneau que la page des réglages (même formulaire, même mutation) : dépendance durable
  // (2026-11-02, chantier Desktop L14).
  'setup=>settings',
  // Leaderboard consommé par palmarès
  'palmares=>leaderboard',
  // Achievements consommé par carrière (section progression)
  'career=>achievements',
  // Career réutilise ExplorerMatchesTable pour les "Matchs marquants"
  'career=>explorer',
  // Amis par profil joueur (plan 2026-09-15, D1/D2) : la page « Amis et groupes »
  // (features/groups) embarque la section d'amis, et la vue match colore les amis
  // depuis usePlayerFriends — la liste a quitté GET /settings (admin-only) pour
  // /players/{slug}/friends. Dépendances durables (2026-09-16).
  'groups=>friends',
  'match-view=>friends',
  // Career réutilise MatchEncountersTable pour les "Joueurs les plus croisés"
  'career=>match-view',
  // Le rejeu 2D pose les kills de la Match View sur sa propre horloge : il réutilise la
  // COLLECTE des kills (`_momentum.collectKillEvents`), les deux cascades de couleur d'équipe
  // (`teamColor` pour les surfaces, `teamSeriesColor` pour les séries) et l'index des joueurs
  // (`xuidMeta`) plutôt que d'en écrire une seconde version. Dépendance durable et voulue — la
  // dupliquer donnerait deux définitions de « ce qui est un kill » et deux couleurs pour la
  // même équipe. QUATRE MODULES NOMMÉS depuis le 2026-09-06 (v2 D.13) : la paire entière
  // autorisait 60 fichiers de la Match View, et le commentaire n'en citait déjà que trois sur
  // les quatre réellement importés.
  'match-replay=>match-view/_momentum',
  'match-replay=>match-view/teamColor',
  'match-replay=>match-view/teamSeriesColor',
  'match-replay=>match-view/xuidMeta',
  // Reciproque. Le CHARGEMENT de l'artefact n'y figure plus : `queries.ts` est descendu dans
  // `lib/replay/` le 2026-09-06 (v2 D.13) avec le document lui-meme (`replayNormalize`,
  // `replayReadyTypes`), sa logique de lecture (`replayLogic`), le roster (`rosterLogic`) et
  // ses deux dependances pures — la fermeture a ete verifiee : aucun de ces sept modules
  // n'importe quoi que ce soit de `features/`. L'ancienne justification disait que le hook
  // « ne peut pas descendre sans emmener replayNormalize » : c'est exactement ce qui a ete
  // fait, et les cinq imports de query ont disparu.
  //
  // CE QUI RESTE, ET POURQUOI, module par module :
  //  - la SECTION « Usage d'equipements, par joueur » est une vue du document de rejeu montee
  //    par la Match View ; son contenu (dictionnaire, calques) vit dans la feature du rejeu — la
  //    descendre dans `lib/` y emmenerait un calque et l'i18n, c'est-a-dire deplacerait la
  //    feature au lieu de partager de la logique ;
  //  - `i18n/i18n` : la barre de faits marquants nomme les usages d'equipement avec le
  //    vocabulaire du rejeu (`REPLAY_TEXT[locale].equipmentUsage`) — un second dictionnaire
  //    donnerait deux libelles pour la meme chose ;
  //  - `model/equipmentUsageLogic` : `equipmentKillBadges` y lit `EPISODE_FAMILIES`, les deux
  //    familles dont l'usage forme un episode. Le module ne peut pas descendre dans `lib/`
  //    (il depend d'un calque), la constante ne peut pas se recopier (regle n° 6).
  'match-view=>match-replay/MatchEquipmentUsageSection',
  'match-view=>match-replay/i18n/i18n',
  'match-view=>match-replay/model/equipmentUsageLogic',
  //  - hooks de zoom / deplacement et leur controle (2026-09-21, ajustements pre-v7.5) :
  //    « Occupation du terrain » se zoome et se deplace avec EXACTEMENT les hooks du rejeu
  //    2D (contrat : une ReplayBounds, un CanvasView, une toile). Les recopier serait la
  //    troisieme copie du geste de zoom ; les descendre dans `lib/` emmenerait le modele
  //    de scene du rejeu. Dependance durable, meme raison que les sections ci-dessus.
  'match-view=>match-replay/hooks/useReplayZoom',
  'match-view=>match-replay/hooks/useReplayDrag',
  'match-view=>match-replay/hooks/useReplayWheelZoom',
  'match-view=>match-replay/hooks/useReplayZoomKeys',
  'match-view=>match-replay/ui/ReplayZoomControl',
  //  - les memes hooks et la meme commande pour le plan de l'onglet Tactique (2026-10-07,
  //    ajustements UI) : le plan se zoome et se deplace exactement comme le rejeu 2D et
  //    « Occupation du terrain ». Meme raison que ci-dessus : une troisieme copie du geste
  //    est interdite, et les descendre dans `lib/` emmenerait le modele de scene du rejeu.
  'tactical=>match-replay/hooks/useReplayZoom',
  'tactical=>match-replay/hooks/useReplayDrag',
  'tactical=>match-replay/hooks/useReplayWheelZoom',
  'tactical=>match-replay/hooks/useReplayZoomKeys',
  'tactical=>match-replay/ui/ReplayZoomControl',
  //  - la règle de HAUTEUR de la toile du rejeu 2D pour le plan de l'onglet Tactique (2026-10-17) : le
  //    plan tient dans l'écran selon la même règle (plancher, plafond, marge basse, pas, chrome
  //    déduit — légende et barre de lecture comptées). Une copie serait la deuxième définition de
  //    « la hauteur que l'écran laisse à une carte » ; la descendre dans `lib/` emmènerait les
  //    constantes du cadrage du rejeu. Dépendance durable, même raison que les hooks de zoom.
  'tactical=>match-replay/hooks/useReplayViewport',
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
  // ChartsShowcasePage du Lab agrège tous les wrappers
  'lab=>timeseries',
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
  'squad=>filters',
  'squad=>friends',
  // TimeseriesPage embarque la section engagement, réutilise les wrappers
  // Squad (WinRate vs History, Map perf vs History, intensity heatmap, builder
  // efficiency) et le tableau Explorer pour l'historique des matchs.
  'timeseries=>engagement',
  'timeseries=>squad',
  'timeseries=>explorer',
  // Home embarque le SyncIndicator + auto-sync triggers de settings.
  'home=>settings',
  // Match-view réutilise des wrappers Squad (colors hash joueur, impact badges)
  // ainsi que la galerie Media.
  'match-view=>squad',
  'match-view=>media',
  // Flow onboarding : auth (XboxLoginPage) + onboarding (OpenSpartanImportCard)
  // partagent les primitives setup (déclaration joueur, jobs de sync initial).
  'auth=>setup',
  'onboarding=>setup',
  // Le device-flow a deux écrans (connexion `XboxLoginPage`, assistant `StepDeviceCode`) et
  // UNE table « code d'échec posé par le serveur → message » (`deviceFlowFailureKey`). La
  // seconde copie, locale à l'assistant, avait dérivé : trois de ses clés n'étaient jamais
  // émises et les codes réels tombaient sur le message générique (2026-10-16, plan sécurité
  // L11-6). Module nommé seulement, dépendance durable.
  'setup=>auth/loginError',
  // Home dashboard agrège aussi le widget Ascension (HomeAscensionWidget).
  'home=>ascension',
  // Explorer réutilise la bannière d'identité joueur de Home
  // (ExplorerTargetIdentityBanner).
  'explorer=>home',
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
  // `squad/colors` (SQUAD_MAIN_PLAYER_TOKEN + SQUAD_TEAMMATE_COLOR_TOKENS) est la
  // SOURCE UNIQUE des couleurs par joueur : la même personne doit porter la même
  // couleur sur la carrière (XP history), la galerie média et la page session que
  // sur l'escouade. Dupliquer la table de tokens la ferait diverger (CLAUDE.md
  // règle 6) — dépendances durables, analogues à match-view=>squad.
  'career=>squad',
  'media=>squad',
  // SessionIntensityProfile réutilise le constructeur d'option ECharts
  // `squad/charts/squadIntensityProfileChart` (courbe d'intensité) plutôt que de le
  // recopier — durable, analogue à session-detail=>explorer.
  'session-detail=>squad',
  // La page Sessions monte l'Emprise solo des Séries temporelles (« Mes prises dans mon camp »,
  // « Mes vies », « Équipement pris », leurs modèles `usages.logic` et leurs textes solo) plutôt que de
  // les recopier : même bloc `SoloEmpriseBlock`, deux pages — durable, analogue à session-detail=>squad
  // (plan PLAN_SESSIONS_EMPRISE_2026-10-06, D2).
  'session-detail=>timeseries',
  // La Vue match montre « Isolement, par joueur » avec la LIGNE de la carte des Series temporelles
  // (`usages/LivesNearTeammateRow`, son axe et sa legende), le meme modele (`usages.logic`) et les
  // memes textes (`usagesText`, `usagesCardsText`) : une carte, trois pages — durable, analogue a
  // session-detail=>timeseries.
  'match-view=>timeseries',
  // --- Déclarées le 2026-11-13 (étape 7.1) : ces dépendances existaient déjà, couvertes
  // soit par le plafond de 7 du lint, soit par un marqueur posé sur le fichier entier
  // (le marqueur ne couvre plus qu'UNE ligne). Chacune est nommée par module : elle
  // n'ouvre pas la feature voisine.
  // La page Escouade, la Tactique et les Tendances montent le MÊME sélecteur de composition et
  // lisent la MÊME sélection d'escouade (clé de stockage, libellés, couleurs de joueur) : une
  // seconde copie donnerait deux sélections et deux palettes d'identité pour la même personne.
  'tactical=>squad/SquadCompositionPicker',
  'tactical=>squad/useCompositionOptions',
  'tendances=>squad/SquadCompositionPicker',
  'tendances=>squad/useCompositionOptions',
  'tendances=>squad/exactComposition',
  'tendances=>squad/squadSelectionStorage',
  'tendances=>squad/i18n',
  'tendances=>squad/colors',
  // Une seule table de noms et de couleurs de chaînes LUSR dans l'app (celle de la Carrière).
  'tendances=>career/lusr-chains',
  // L'onglet Entraînement de l'Ascension agrège les cartes du coach et lit le réglage du mode
  // proactif par le hook que la section d'administration qui l'édite lit aussi ; la
  // progression Prestige lit la liste d'amis du joueur (même source que la page Amis).
  'ascension=>coach',
  'ascension=>settings/queries',
  'ascension=>friends/queries',
  // Les liens profonds de l'Ascension vers les Séries temporelles se construisent avec
  // l'encodeur UNIQUE du paramètre ?f= (filterLink).
  'ascension=>filters/filterLink',
  // Les deux donuts du Versus de l'Explorateur sont ceux de Palmarès (relations, écart de frags).
  'explorer=>palmares/RelationWinRateDonut',
  'explorer=>palmares/CumulativeFragGapChart',
  // Le tri client des lignes de l'Explorateur sert aussi les tableaux de rencontres, du
  // classement de match et de l'historique des synergies : une seule règle de tri.
  'match-view=>explorer/explorerMatchesClientSort',
  'squad=>explorer/explorerMatchesClientSort',
])

// Fichiers de `components/` qui importent encore `features/_shared` (frontière inversée) :
// chaque entrée tombe quand la dépendance descend dans `components/` ou `lib/`, et une
// entrée qu'aucun import ne sert plus fait échouer le lint.
const REVERSE_SHARED_ALLOWED = new Set([
  // Les quatre modules d'assistance (barre de répartition, teinte, échange, textes) vivent
  // dans `features/_shared/assists/` ; la carte de match les monte. Déclaré le 2026-11-13.
  'apps/web/src/components/ui/match-card-assisted-frags.tsx',
])

// Fichiers shell autorisés à importer @/features/ (orchestration globale).
const SHELL_PATH_PREFIX = 'apps/web/src/components/shell/'

// Marqueur d'exception ponctuelle : `cross-feature-allow: <raison>`, sur la ligne de
// l'import ou sur le commentaire `//` qui la précède immédiatement.
const ALLOWED_INLINE_MARKER = 'cross-feature-allow'
const MARKER_WITH_REASON_RE = /cross-feature-allow:\s*\S/

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
  // apps/web/src/features/foo/bar.tsx → "foo" ; features/_shared/x.ts → "_shared"
  const match = relPath.match(/^apps\/web\/src\/features\/([a-z0-9_-]+)\//)
  return match ? match[1] : null
}

/**
 * Cible `features/` d'un spécificateur d'import, ou null. L'alias `@/features/X/...` et le
 * chemin relatif qui y aboutit (`../../X/...`, `../../../features/X/...`) donnent le même
 * résultat : écrire l'import autrement ne le sort pas du contrôle.
 */
function featureTarget(file, specifier) {
  let fromSrc
  if (specifier.startsWith('@/')) {
    fromSrc = specifier.slice(2)
  } else {
    fromSrc = relative(WEB_SRC, resolve(dirname(file), specifier)).replaceAll('\\', '/')
  }
  const m = fromSrc.match(FEATURE_TARGET_RE)
  return m ? { feature: m[1], modulePath: m[2] ?? '' } : null
}

const isCommentLine = (line) => /^\s*(\/\/|\/\*|\*)/.test(line)

/**
 * Index (0-based) de la ligne du marqueur qui couvre l'import écrit à `idx`, ou -1 : la
 * ligne elle-même, ou le bloc de commentaires `//` qui la précède sans ligne vide.
 */
function markerLineFor(lines, idx) {
  if (MARKER_WITH_REASON_RE.test(lines[idx])) return idx
  for (let i = idx - 1; i >= 0 && /^\s*\/\//.test(lines[i]); i--) {
    if (MARKER_WITH_REASON_RE.test(lines[i])) return i
  }
  return -1
}

const crossViolations = []
const sharedViolations = []
const reverseViolations = []
const deadMarkers = []
// Entrées d'ALLOWED_CROSS_IMPORTS réellement servies pendant le balayage : sert à détecter les
// dérogations mortes (déclarées mais jamais appariées à un import réel).
const usedAllowEntries = new Set()
const usedReverseEntries = new Set()
let markedImports = 0

const featuresDir = join(WEB_SRC, 'features')
const componentsDir = join(WEB_SRC, 'components')

// Tous les imports d'un fichier qui visent `features/` : [{ idx, target }].
function featureImports(file, lines) {
  const found = []
  lines.forEach((line, idx) => {
    if (isCommentLine(line)) return
    QUOTED_SPECIFIER_RE.lastIndex = 0
    let m
    while ((m = QUOTED_SPECIFIER_RE.exec(line)) !== null) {
      const target = featureTarget(file, m[2])
      if (target) found.push({ idx, target })
    }
  })
  return found
}

// Marqueurs d'un fichier : ceux qui couvrent un import hors allowlist sont consommés ; les
// autres sont signalés (le marqueur d'un fichier entier ne couvre plus rien).
function reportDeadMarkers(relPath, lines, consumed) {
  lines.forEach((line, idx) => {
    if (!line.includes(ALLOWED_INLINE_MARKER) || consumed.has(idx)) return
    deadMarkers.push({ file: relPath, line: idx + 1 })
  })
}

// 1 et 2. features/ : cross-feature et socle `_shared`
try {
  for (const file of walk(featuresDir)) {
    const relPath = relative(REPO_ROOT, file).replaceAll('\\', '/')
    const consumerFeature = getFeatureNameFromPath(relPath)
    if (!consumerFeature) continue

    const lines = readFileSync(file, 'utf8').split('\n')
    const consumedMarkers = new Set()

    for (const { idx, target } of featureImports(file, lines)) {
      const importedFeature = target.feature
      // Importer le socle est libre ; se lire soi-même aussi.
      if (importedFeature === consumerFeature || importedFeature === SHARED_FEATURE) continue

      const importedModule = `${importedFeature}${target.modulePath}`
      const where = `${relPath}:${idx + 1}`

      if (consumerFeature !== SHARED_FEATURE) {
        // Deux formes d'exception durable : la PAIRE (`a=>b`, toute la feature) et le MODULE
        // NOMMÉ (`a=>b/dossier/module`), la seule qui dise la vérité quand une feature
        // n'emprunte que trois modules à sa voisine. Un module autorisé n'ouvre pas sa
        // feature ; ajouter un import ailleurs dans la même voisine fait rougir le lint.
        const paire = `${consumerFeature}=>${importedFeature}`
        const module = `${consumerFeature}=>${importedModule}`
        if (ALLOWED_CROSS_IMPORTS.has(paire)) {
          usedAllowEntries.add(paire)
          continue
        }
        if (ALLOWED_CROSS_IMPORTS.has(module)) {
          usedAllowEntries.add(module)
          continue
        }
      }

      const marker = markerLineFor(lines, idx)
      if (marker >= 0) {
        consumedMarkers.add(marker)
        markedImports++
        continue
      }
      const violation = { file: relPath, where, importedFeature: importedModule }
      if (consumerFeature === SHARED_FEATURE) sharedViolations.push(violation)
      else crossViolations.push(violation)
    }
    reportDeadMarkers(relPath, lines, consumedMarkers)
  }
} catch (err) {
  console.error('cross-feature scan error:', err.message)
  process.exit(2)
}

// 3. Frontière inversée (components/ → features/)
try {
  for (const file of walk(componentsDir)) {
    const relPath = relative(REPO_ROOT, file).replaceAll('\\', '/')
    if (relPath.startsWith(SHELL_PATH_PREFIX)) continue // shell orchestre

    const lines = readFileSync(file, 'utf8').split('\n')
    const consumedMarkers = new Set()

    for (const { idx, target } of featureImports(file, lines)) {
      if (target.feature === SHARED_FEATURE && REVERSE_SHARED_ALLOWED.has(relPath)) {
        usedReverseEntries.add(relPath)
        continue
      }
      const marker = markerLineFor(lines, idx)
      if (marker >= 0) {
        consumedMarkers.add(marker)
        markedImports++
        continue
      }
      reverseViolations.push({
        file: relPath,
        where: `${relPath}:${idx + 1}`,
        importedFeature: `${target.feature}${target.modulePath}`,
      })
    }
    reportDeadMarkers(relPath, lines, consumedMarkers)
  }
} catch (err) {
  console.error('reverse boundary scan error:', err.message)
  process.exit(2)
}

const deadAllowEntries = [...ALLOWED_CROSS_IMPORTS].filter((entry) => !usedAllowEntries.has(entry))
const deadReverseEntries = [...REVERSE_SHARED_ALLOWED].filter((entry) => !usedReverseEntries.has(entry))

function printGroup(title, violations, hint) {
  if (violations.length === 0) return
  console.log(`\n# ${title} (${violations.length}) :`)
  for (const v of violations) console.log(`  ${v.where}  → ${v.importedFeature}${hint ? `  (${hint})` : ''}`)
}

printGroup('Cross-feature imports', crossViolations)
printGroup('Socle _shared important une feature', sharedViolations, '_shared ne dépend d\'aucune feature')
printGroup('Frontière inversée (gap #14)', reverseViolations, 'components/ ne doit pas importer features/')

if (deadMarkers.length > 0) {
  console.log(`\n# Marqueurs ${ALLOWED_INLINE_MARKER} qui ne couvrent aucun import (${deadMarkers.length}) :`)
  for (const m of deadMarkers) console.log(`  ${m.file}:${m.line}`)
  console.log(
    `\nUn marqueur se pose sur la ligne de l'import (ou le commentaire // qui la précède) ` +
      `et porte une raison : \`${ALLOWED_INLINE_MARKER}: <raison>\`.`,
  )
}

if (deadAllowEntries.length > 0 || deadReverseEntries.length > 0) {
  console.log(`\n# Dérogations mortes (${deadAllowEntries.length + deadReverseEntries.length}) :`)
  for (const entry of [...deadAllowEntries, ...deadReverseEntries]) console.log(`  ${entry}`)
  console.log(
    '\nCes entrées ne correspondent plus à aucun import réel. Les retirer (avec leur ' +
      'commentaire devenu orphelin).',
  )
}

const total =
  crossViolations.length +
  sharedViolations.length +
  reverseViolations.length +
  deadMarkers.length +
  deadAllowEntries.length +
  deadReverseEntries.length

if (total > 0) {
  console.log(
    `\nERREUR : ${total} violation(s), plafond 0. Descendre la dépendance dans components/ ou lib/, ` +
      `ou déclarer une dépendance durable dans ALLOWED_CROSS_IMPORTS (hors _shared), ou marquer ` +
      `l'import \`// ${ALLOWED_INLINE_MARKER}: <raison>\`.`,
  )
  process.exit(1)
}

console.log(`lint-cross-feature-imports: clean (0 violation, ${markedImports} import(s) marqué(s) par ligne)`)
process.exit(0)
