/**
 * replayPreferences.ts — LA PORTE des préférences du rejeu : leurs clés, leurs défauts et leurs lectures validées.
 * Le patron de stockage (`lib/replay/preferenceStore` : clé simple, lecture sous try — navigation privée ou storage
 * refusé dégradent en silence vers la valeur par défaut, jamais une erreur affichée ni un plantage) y est réexporté.
 *
 * CENTRALISÉ ICI (CLAUDE.md règle 6, « à la 3e copie, centraliser ») : le tiroir de
 * réglages (décision utilisateur du 16/08) ajoute au son persisté les calques, la vitesse
 * et le filtre de catégories — quatre à six clés qui partagent EXACTEMENT ce patron.
 * useReplaySound et useReplaySettings l'importent tous les deux plutôt que de le
 * recopier une troisième fois chacun de leur côté.
 */
import { DEFAULT_EXPORT_FORMAT_ID, EXPORT_FORMAT_IDS, type ExportFormatId } from '../export/exportFormats'
import { DEFAULT_FEED_FILTER, FEED_FILTER_IDS, type FeedFilterId } from '../model/feedFilterLogic'
import { DEFAULT_MAP_BACKGROUND_VARIANT, MAP_BACKGROUND_VARIANTS, type MapBackgroundVariant } from '@/lib/replay/mapBackgroundVariant'
import { normalizeTurns, QUARTER_TURNS, type QuarterTurns } from '@/lib/replay/orientation'
import { VIEW_PRESET_IDS, type ViewPresetId } from './viewPresetLogic'
import {
  persistPreference,
  readStoredChoice,
  readStoredFlag,
  readStoredNumber,
  removePreference,
  subscribePreference,
} from '@/lib/replay/preferenceStore'

// Le patron de stockage (lecture sous try, abonnés par clé) vit dans `lib/replay/preferenceStore`, partagé avec les
// autres vues 2D ; ce module reste l'unique porte des préférences du rejeu et le réexporte.
export { persistPreference, readStoredChoice, readStoredFlag, readStoredNumber, removePreference, subscribePreference }

/**
 * LE FORMAT DE L'EXPORT VIDÉO (plan « formats vidéo standard », décision D4, 2026-09-16) :
 * retenu dans CE navigateur, jamais côté serveur. La lecture est VALIDÉE contre le catalogue
 * (`export/exportFormats.ts`) : absente, inconnue ou écrite par une autre version -> 1080p.
 */
export const EXPORT_FORMAT_KEY = 'replay-export-format'

/** Le format retenu, ou le défaut du catalogue. */
export function readExportFormat(): ExportFormatId {
  return readStoredChoice(EXPORT_FORMAT_KEY, DEFAULT_EXPORT_FORMAT_ID, EXPORT_FORMAT_IDS)
}

/** Retient le format choisi. Stockage refusé : le choix vaut pour la session, sans erreur. */
export function persistExportFormat(id: ExportFormatId): void {
  persistPreference(EXPORT_FORMAT_KEY, id)
}

/**
 * LE FILTRE DU FIL DES ÉLIMINATIONS (D-4 du plan « maquettes du rejeu ») : retenu dans CE
 * navigateur, jamais côté serveur. La lecture est VALIDÉE contre la liste fermée des filtres :
 * absente, inconnue ou écrite par une autre version -> Tout. Le filtre mémorisé qu'un match ne
 * propose pas (Objectifs sur un Assassin) n'est pas effacé : il se lit tel quel, et c'est
 * `effectiveFeedFilter` qui retombe sur Tout pour ce match seulement.
 */
export const FEED_FILTER_KEY = 'replay-feed-filter'

/** Le filtre du fil retenu, ou Tout. */
export function readFeedFilter(): FeedFilterId {
  return readStoredChoice(FEED_FILTER_KEY, DEFAULT_FEED_FILTER, FEED_FILTER_IDS)
}

/** Retient le filtre choisi. Stockage refusé : le choix vaut pour la session, sans erreur. */
export function persistFeedFilter(id: FeedFilterId): void {
  persistPreference(FEED_FILTER_KEY, id)
}

/**
 * LE FIL DES ÉLIMINATIONS POSÉ SUR LE PLAN (option « Fil des éliminations » du panneau Vue) : un OUI/NON retenu
 * dans CE navigateur, jamais côté serveur, lu et écrit comme les autres interrupteurs par `usePersistedFlag`
 * (`readStoredFlag` ci-dessus : absent ou stockage refusé -> la valeur par défaut). AFFICHÉ par défaut : c'est
 * le fil que le Cinéma et le Plein cadre montrent quand la rangée de fiches est posée. L'option n'a d'effet que
 * là où ce fil peut se poser (le panneau ne la propose pas ailleurs) ; la clé et sa valeur valent pour tous les cadres.
 */
export const PLAN_FEED_KEY = 'replay-show-plan-feed'
export const PLAN_FEED_DEFAULT = true

/**
 * LE PRÉRÉGLAGE DE VUE CHOISI en dernier (plan « maquettes du rejeu », lot 3) : retenu dans CE
 * navigateur. La vue active se DÉDUIT de l'état des calques (`activeView`) ; cette valeur ne
 * sert qu'à départager deux préréglages égaux sur les calques d'un match. Les clés `replay-*`
 * des calques restent lues telles quelles. La lecture est VALIDÉE contre la liste fermée :
 * absente, inconnue (« personnalise » compris, il se déduit et ne se retient pas) ou écrite
 * par une autre version -> `undefined`.
 */
export const VIEW_PRESET_KEY = 'replay-view-preset'

/** Le préréglage retenu, ou `undefined`. */
export function readViewPreset(): ViewPresetId | undefined {
  return readStoredChoice<ViewPresetId | ''>(VIEW_PRESET_KEY, '', VIEW_PRESET_IDS) || undefined
}

/** Retient le préréglage choisi. Stockage refusé : le choix vaut pour la session, sans erreur. */
export function persistViewPreset(id: ViewPresetId): void {
  persistPreference(VIEW_PRESET_KEY, id)
}

/**
 * L'ORIENTATION DE LA CARTE (`lib/replay/orientation`) est un réglage PAR CARTE : retenue dans CE
 * navigateur sous `replay-orientation.<mapId>`, jamais côté serveur. Une carte dont l'utilisateur
 * a tourné la vue s'ouvre ainsi toujours tournée, sans toucher aux autres.
 *
 * TROIS ÉTATS : aucun choix retenu (« Auto », règle `autoTurns`), ou un des quatre nombres de
 * quarts de tour (0 compris : un 0 retenu est un choix, pas une absence). La lecture est VALIDÉE
 * contre les quatre positions : absente, hors liste ou écrite par une autre version -> aucun choix.
 * Les valeurs écrites sont celles d'avant l'Auto, lues telles quelles.
 */
export const ORIENTATION_KEY_PREFIX = 'replay-orientation.'

/** Les valeurs écrites : le nombre de quarts de tour, en chaîne (`readStoredChoice` compare des chaînes). */
const ORIENTATION_VALUES: readonly string[] = QUARTER_TURNS.map(String)

/** La clé de stockage de l'orientation d'une carte. */
export function orientationKey(mapId: string): string {
  return `${ORIENTATION_KEY_PREFIX}${mapId}`
}

/** L'orientation retenue pour cette carte, ou `null` quand aucun choix n'est retenu (Auto). */
export function readOrientation(mapId: string): QuarterTurns | null {
  const raw = readStoredChoice(orientationKey(mapId), '', ORIENTATION_VALUES)
  return raw === '' ? null : normalizeTurns(Number(raw))
}

/** Retient l'orientation choisie pour cette carte. Stockage refusé : le choix vaut pour la session. */
export function persistOrientation(mapId: string, turns: QuarterTurns): void {
  persistPreference(orientationKey(mapId), String(turns))
}

/** Efface le choix retenu pour cette carte : elle revient en Auto. */
export function clearOrientation(mapId: string): void {
  removePreference(orientationKey(mapId))
}
/**
 * LE STYLE DU FOND DE CARTE (`lib/replay/mapBackgroundVariant`) : `plan` (défaut, la vue plan à hauteurs),
 * `rendu` ou `relief` (le terrain teinté par hauteur), un réglage GLOBAL — il vaut pour toutes les cartes et les trois vues 2D (rejeu,
 * Tactique, « Occupation du terrain »). Retenu dans CE navigateur, jamais côté serveur. La lecture est
 * VALIDÉE contre les trois valeurs : absente, inconnue ou écrite par une autre version -> `plan` (le défaut).
 * Le choix `plan` ne s'applique qu'à un fond qui porte une image plan, le choix `relief` qu'à un fond qui porte
 * un relief lisible : sinon le plan, puis le rendu, sont servis.
 */
export const BACKGROUND_STYLE_KEY = 'replay-background-style'

/** Le style de fond retenu, ou le plan (le défaut). */
export function readBackgroundStyle(): MapBackgroundVariant {
  return readStoredChoice(BACKGROUND_STYLE_KEY, DEFAULT_MAP_BACKGROUND_VARIANT, MAP_BACKGROUND_VARIANTS)
}

/** Retient le style choisi. Stockage refusé : le choix vaut pour la session, sans erreur. */
export function persistBackgroundStyle(style: MapBackgroundVariant): void {
  persistPreference(BACKGROUND_STYLE_KEY, style)
}
