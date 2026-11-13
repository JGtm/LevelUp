/**
 * replayPreferences.ts — LE SEUL ENDROIT qui touche localStorage pour les préférences du
 * rejeu. Patron NÉ dans useReplaySound (son : coupé/volume) : clé simple, lecture sous
 * try — navigation privée ou storage refusé dégradent en silence vers la valeur par
 * défaut, jamais une erreur affichée ni un plantage.
 *
 * CENTRALISÉ ICI (CLAUDE.md règle 6, « à la 3e copie, centraliser ») : le tiroir de
 * réglages (décision utilisateur du 16/08) ajoute au son persisté les calques, la vitesse
 * et le filtre de catégories — quatre à six clés qui partagent EXACTEMENT ce patron.
 * useReplaySound et useReplaySettings l'importent tous les deux plutôt que de le
 * recopier une troisième fois chacun de leur côté.
 */
import { DEFAULT_EXPORT_FORMAT_ID, EXPORT_FORMAT_IDS, type ExportFormatId } from '../export/exportFormats'
import { DEFAULT_FEED_FILTER, FEED_FILTER_IDS, type FeedFilterId } from '../model/feedFilterLogic'
import { normalizeTurns, QUARTER_TURNS, type QuarterTurns } from '../model/orientation'
import { VIEW_PRESET_IDS, type ViewPresetId } from './viewPresetLogic'

/** Lit un booléen persisté ; absent ou storage indisponible -> `fallback`. */
export function readStoredFlag(key: string, fallback: boolean): boolean {
  try {
    const raw = localStorage.getItem(key)
    return raw === null ? fallback : raw === 'true'
  } catch {
    return fallback
  }
}

/**
 * Lit un nombre persisté, accepté seulement si `isValid` le confirme (bornes ou
 * appartenance à un ensemble autorisé — ex. les multiplicateurs de vitesse) ; absent,
 * invalide ou storage indisponible -> `fallback`.
 */
export function readStoredNumber(
  key: string,
  fallback: number,
  isValid: (value: number) => boolean,
): number {
  try {
    const raw = localStorage.getItem(key)
    if (raw === null) return fallback
    const value = Number(raw)
    return Number.isFinite(value) && isValid(value) ? value : fallback
  } catch {
    return fallback
  }
}

/**
 * Lit un CHOIX persisté parmi un ensemble fermé (ex. la lecture de la carte de chaleur :
 * présence ou éliminations) ; absent, hors liste ou storage indisponible -> `fallback`.
 *
 * MÊME EXIGENCE QUE `readStoredNumber` : une valeur écrite par une AUTRE version de l'app —
 * ou par n'importe qui, `localStorage` est ouvert — ne doit jamais entrer dans l'état. Ce
 * qui n'est pas dans la liste n'existe pas.
 */
export function readStoredChoice<T extends string>(
  key: string,
  fallback: T,
  allowed: readonly T[],
): T {
  try {
    const raw = localStorage.getItem(key)
    return raw !== null && (allowed as readonly string[]).includes(raw) ? (raw as T) : fallback
  } catch {
    return fallback
  }
}

/**
 * LES ABONNÉS D'UNE CLÉ, et pourquoi ils sont apparus (2026-08-18, item R3.7).
 *
 * Jusqu'ici UN SEUL composant lisait chaque préférence : l'état React local du hook suffisait.
 * Les FICHES COMPACTES cassent cette hypothèse — la bascule vit dans le tiroir (sous le
 * canvas), les fiches vivent dans une AUTRE colonne de la page. Deux `useState` initialisés
 * du même `localStorage` ne se parlent pas : la bascule aurait bougé sans que les fiches
 * changent, jusqu'au prochain rechargement.
 *
 * L'ÉVÉNEMENT `storage` DU NAVIGATEUR NE RÉPOND PAS À CE BESOIN : il ne se déclenche QUE
 * pour les AUTRES onglets, jamais pour celui qui écrit. C'est le piège classique, et c'est
 * pour cela que ce registre existe plutôt qu'un abonnement à `window`.
 */
const abonnes = new Map<string, Set<(value: string) => void>>()

/** subscribePreference — écoute une clé ; rend la fonction qui se désabonne. */
export function subscribePreference(key: string, fn: (value: string) => void): () => void {
  const set = abonnes.get(key) ?? new Set()
  abonnes.set(key, set)
  set.add(fn)
  return () => set.delete(fn)
}

/** Persiste une préférence (booléen ou nombre déjà tourné en chaîne, JSON pour une liste). */
export function persistPreference(key: string, value: string): void {
  try {
    localStorage.setItem(key, value)
  } catch {
    /* stockage refusé (navigation privée) : la préférence ne survit pas, le réglage marche. */
  }
  // Les abonnés sont prévenus MÊME si le stockage a refusé : le réglage doit marcher dans
  // la session, c'est sa survie au rechargement qui est perdue, pas son effet.
  for (const fn of abonnes.get(key) ?? []) fn(value)
}

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
 * L'ORIENTATION DE LA CARTE (`model/orientation`) est un réglage PAR CARTE : retenue dans CE
 * navigateur sous `replay-orientation.<mapId>`, jamais côté serveur. Une carte dont l'utilisateur
 * a tourné la vue s'ouvre ainsi toujours tournée, sans toucher aux autres. La lecture est
 * VALIDÉE contre les quatre positions : absente, hors liste ou écrite par une autre version ->
 * 0 quart de tour.
 */
export const ORIENTATION_KEY_PREFIX = 'replay-orientation.'

/** Les valeurs écrites : le nombre de quarts de tour, en chaîne (`readStoredChoice` compare des chaînes). */
const ORIENTATION_VALUES: readonly string[] = QUARTER_TURNS.map(String)

/** La clé de stockage de l'orientation d'une carte. */
export function orientationKey(mapId: string): string {
  return `${ORIENTATION_KEY_PREFIX}${mapId}`
}

/** L'orientation retenue pour cette carte, ou 0 quart de tour. */
export function readOrientation(mapId: string): QuarterTurns {
  return normalizeTurns(Number(readStoredChoice(orientationKey(mapId), '0', ORIENTATION_VALUES)))
}

/** Retient l'orientation choisie pour cette carte. Stockage refusé : le choix vaut pour la session. */
export function persistOrientation(mapId: string, turns: QuarterTurns): void {
  persistPreference(orientationKey(mapId), String(turns))
}
