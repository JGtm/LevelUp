/**
 * zone.logic — la logique PURE de la zone sélectionnée : la zone présélectionnée (la plus chaude),
 * ses coordonnées, sa valeur et sa sous-ligne, le modèle de chaque mini-tuile « Rejeu » et la
 * position de l'étiquette du nom de zone sur le plan.
 *
 * LE BADGE DE PLACEMENT EST TRONQUÉ AU MÈTRE, jamais arrondi : 17,9 m reste « près · 17 m » — la
 * portée du radar est la frontière, et un arrondi ferait lire « 18 m » d'une mort dite « près ».
 * Il ne se pose que sur une MORT (un frag, une entrée ou une réapparition n'ont pas de placement).
 */
import type { CelluleTactique, TacticalCelluleReponse, TacticalContribution } from '@/lib/api/types'
import { intlLocale } from '@/lib/formatters'
import type { Locale } from '@/lib/i18n/locale'
import { formatClock, OUVERTURE_AVANT_EVENEMENT_MS } from '@/lib/replay/replayLogic'

import type { TacticalText } from './i18n'
import {
  rectSelection,
  repereAspect,
  type FenetreDuPlan,
  type RepereTactique,
  type TacticalQuestion,
} from './tacticalView.logic'
import { distanceEnMetres } from './tueurs.logic'

/** Au-delà de ce point de la largeur, l'étiquette du nom de zone passe à GAUCHE de la cellule. */
const ETIQUETTE_BASCULE = 0.55
/** Marges hautes et basses du cadre où l'étiquette se cale sur la cellule au lieu de la centrer. */
const ETIQUETTE_BORD = 0.04

/**
 * zoneLaPlusChaude — la zone présélectionnée (D10) : la plus grande valeur (absolue sur une lecture
 * signée), puis le plus de matchs distincts.
 */
export function zoneLaPlusChaude(cellules: readonly CelluleTactique[], signee: boolean): CelluleTactique | null {
  let meilleure: CelluleTactique | null = null
  let score = -Infinity
  for (const c of cellules) {
    const s = signee ? Math.abs(c.valeur) : c.valeur
    if (s > score || (s === score && meilleure !== null && c.matchs > meilleure.matchs)) {
      meilleure = c
      score = s
    }
  }
  return meilleure
}

/** Le choix de l'utilisateur est d'ÉCARTER la zone : la présélection de la plus chaude ne revient pas. */
export const ZONE_ECARTEE = 'ecartee' as const

/** Ce que l'utilisateur a décidé : une cellule, l'écart de toute zone, ou rien (présélection). */
export type ChoixDeZone = { col: number; row: number } | typeof ZONE_ECARTEE | null

/** zoneAffichee — la zone du plan : le choix, sinon la plus chaude ; aucune une fois écartée. */
export function zoneAffichee(
  choix: ChoixDeZone,
  chaude: { col: number; row: number } | null,
): { col: number; row: number } | null {
  if (choix === ZONE_ECARTEE) return null
  return choix ?? chaude
}

/**
 * choixApresClic — le choix après un clic sur le plan : un clic sur la zone déjà affichée, ou loin de
 * toute cellule servie (`cliquee` nulle), l'ÉCARTE ; un clic sur une autre cellule la choisit.
 */
export function choixApresClic(
  affichee: { col: number; row: number } | null,
  cliquee: { col: number; row: number } | null,
): ChoixDeZone {
  if (cliquee === null) return ZONE_ECARTEE
  if (affichee !== null && affichee.col === cliquee.col && affichee.row === cliquee.row) return ZONE_ECARTEE
  return cliquee
}

/**
 * Distance maximale, en pas de grille, entre un clic et le CENTRE d'une cellule servie pour que le
 * clic la retienne : la chaleur lissée déborde de quelques dixièmes de pas autour de chaque cellule
 * (`chaleurLissee.ts`), et un clic sur ce débord vise la cellule qui l'a peint.
 */
const CLIC_PORTEE_PAS = 1

/**
 * choixDuClic — l'adresse que retient un clic sur le plan (`point`, en mètres monde) : la cellule
 * SERVIE qui le contient, sinon la cellule servie dont le centre est le plus proche à moins d'un
 * pas ; `null` sinon (loin de toute cellule : `choixApresClic` y lit l'écart de la zone).
 */
export function choixDuClic(
  cellules: readonly CelluleTactique[],
  point: { x: number; y: number },
  pasM: number,
): { col: number; row: number } | null {
  if (!(pasM > 0)) return null
  const col = Math.floor(point.x / pasM)
  const row = Math.floor(point.y / pasM)
  if (cellules.some((c) => c.col === col && c.lig === row)) return { col, row }
  let meilleure: { col: number; row: number } | null = null
  let distance = CLIC_PORTEE_PAS * pasM
  for (const c of cellules) {
    if (Math.abs(c.col - col) > 1 || Math.abs(c.lig - row) > 1) continue
    const d = Math.hypot((c.col + 0.5) * pasM - point.x, (c.lig + 0.5) * pasM - point.y)
    if (d <= distance) {
      distance = d
      meilleure = { col: c.col, row: c.lig }
    }
  }
  return meilleure
}

/** decimale — un nombre à une décimale, séparateur de la langue (« 1,8 »). */
export function decimale(v: number, locale: Locale): string {
  return new Intl.NumberFormat(intlLocale(locale), { minimumFractionDigits: 1, maximumFractionDigits: 1 }).format(v)
}

/** Un nombre au plus au dixième, signe moins typographique. */
function metres(v: number, locale: Locale): string {
  const n = new Intl.NumberFormat(intlLocale(locale), { maximumFractionDigits: 1 }).format(Math.abs(v))
  return v < 0 ? `−${n}` : n
}

/** coordonneesDeZone — « x −14…−12 m · y 4…6 m » : l'étendue monde de la cellule au pas servi. */
export function coordonneesDeZone(t: TacticalText, col: number, lig: number, pasM: number, locale: Locale): string {
  return t.zoneCoords(
    metres(col * pasM, locale),
    metres((col + 1) * pasM, locale),
    metres(lig * pasM, locale),
    metres((lig + 1) * pasM, locale),
  )
}

/** valeurAffichee — la valeur de la zone ; une lecture signée porte son signe (« + », « − »). */
export function valeurAffichee(valeur: number, signee: boolean, locale: Locale): string {
  const n = new Intl.NumberFormat(intlLocale(locale), { maximumFractionDigits: 2 }).format(Math.abs(valeur))
  if (!signee || valeur === 0) return valeur < 0 ? `−${n}` : n
  return valeur > 0 ? `+ ${n}` : `− ${n}`
}

/**
 * sousLigneDeZone — « N matchs distincts », précédé des victoires et défaites (« victoires −
 * défaites ») ou des frags et morts (« solde »).
 */
export function sousLigneDeZone(t: TacticalText, question: TacticalQuestion, c: CelluleTactique): string {
  const matchs = t.zoneMatches(c.matchs)
  if (question === 'gagne') return `${t.zoneWinsLosses(c.matchs_victoire, c.matchs_defaite)} · ${matchs}`
  if (question === 'solde') return `${t.zoneKillsDeaths(c.frags ?? 0, c.morts ?? 0)} · ${matchs}`
  return matchs
}

/** dateEtHeure — « 02/09/2026 · 00:30 » au fuseau du joueur. */
function dateEtHeure(iso: string, timezone: string, locale: Locale): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  const loc = intlLocale(locale)
  const date = new Intl.DateTimeFormat(loc, { timeZone: timezone, day: '2-digit', month: '2-digit', year: 'numeric' })
  const heure = new Intl.DateTimeFormat(loc, { timeZone: timezone, hour: '2-digit', minute: '2-digit', hourCycle: 'h23' })
  return `${date.format(d)} · ${heure.format(d)}`
}

/** Ce que la mini-tuile d'une contribution affiche. */
interface ModeleDeTuile {
  mode?: string
  score?: string
  date: string
  /** L'instant sur l'horloge AFFICHÉE du rejeu (« 7:20 ») ; absent quand le serveur ne le publie pas. */
  instant?: string
  fait: string
  /** Le fait découpé autour du gamertag (« Tué par » · « Rival » · «  »), pour que seul le gamertag tronque. */
  faitDecoupe: { avant: string; gamertag?: string; apres: string }
  /** L'arme, ou la catégorie de la source traduite ; absente sinon. Seule partie qui se tronque. */
  arme?: string
  badge?: string
  /** La distance à l'autre joueur (« à 23 m ») ; absente quand un des deux points manque. */
  distance?: string
  /** Le texte complet de la tuile, pour l'infobulle. */
  titre: string
  rejeu: { disponible: boolean; search: { t: string; clock: 'match' | 'film' }; libelle: string }
}

function faitDe(t: TacticalText, c: TacticalContribution): string {
  switch (c.face) {
    case 'mort':
      return c.autre_gamertag ? t.tileKilledBy(c.autre_gamertag) : t.tileDeath
    case 'frag':
      return c.autre_gamertag ? t.tileKilled(c.autre_gamertag) : t.tileFrag
    case 'entree':
      return t.tileEntry
    case 'reapparition':
      return t.tileRespawn
    default:
      return ''
  }
}

/** Marque posée à la place du gamertag pour découper la phrase du fait sans la dupliquer. */
const MARQUE_GAMERTAG = '\u0001'

/** faitDecoupeDe — le fait de la face autour du gamertag de l'autre joueur ; sans gamertag, tout en `avant`. */
function faitDecoupeDe(t: TacticalText, c: TacticalContribution): { avant: string; gamertag?: string; apres: string } {
  const phrase = c.face === 'mort' ? t.tileKilledBy : c.face === 'frag' ? t.tileKilled : null
  if (!phrase || !c.autre_gamertag) return { avant: faitDe(t, c), apres: '' }
  const [avant = '', apres = ''] = phrase(MARQUE_GAMERTAG).split(MARQUE_GAMERTAG)
  return { avant, gamertag: c.autre_gamertag, apres }
}

export function armeDe(t: TacticalText, c: TacticalContribution): string | undefined {
  if (c.arme_label) return c.arme_label
  return c.categorie_source ? t.tileCategories[c.categorie_source] : undefined
}

/** distanceDe — « à 23 m » (« < 1 m » sous le mètre) : la distance entre l'événement et l'autre joueur. */
function distanceDe(t: TacticalText, c: TacticalContribution, locale: Locale): string | undefined {
  const d = distanceEnMetres(c)
  if (d === 0) return t.tileDistanceLt1
  return d === null ? undefined : t.tileDistance(new Intl.NumberFormat(intlLocale(locale)).format(d))
}

function badgeDe(t: TacticalText, c: TacticalContribution, locale: Locale): string | undefined {
  if (c.face !== 'mort' || !c.placement) return undefined
  const d = c.placement.distance_m
  if (d === undefined || d === null) return c.placement.seul ? t.tileAlone : undefined
  const texte = d < 1 ? '< 1' : new Intl.NumberFormat(intlLocale(locale)).format(Math.floor(d))
  return c.placement.seul ? t.tileAloneAt(texte) : t.tileNearAt(texte)
}

/**
 * modeleDeTuile — la mini-tuile d'une contribution : mode, score (« X - Y », mon camp d'abord),
 * issue (`issue`, le mot du titre), date au fuseau du joueur, instant, fait, arme, badge, et le lien
 * de rejeu à l'instant (porté seulement si l'artefact existe).
 */
export function modeleDeTuile(
  t: TacticalText,
  c: TacticalContribution,
  locale: Locale,
  timezone: string,
  issue: string | null,
): ModeleDeTuile {
  // L'instant écrit est celui que l'horloge du rejeu affichera à l'événement (`instant_affiche_ms`,
  // calculé par le serveur depuis le coup d'envoi) : jamais `instant_ms`, qui compte le countdown.
  const instant = c.instant_affiche_ms == null ? undefined : formatClock(c.instant_affiche_ms)
  const fait = faitDe(t, c)
  const arme = armeDe(t, c)
  const badge = badgeDe(t, c, locale)
  const distance = distanceDe(t, c, locale)
  const date = dateEtHeure(c.match_started_at, timezone, locale)
  const titre = [c.mode_label, c.score_label, issue, date, instant, fait, distance, arme, badge].filter(Boolean).join(' · ')
  return {
    mode: c.mode_label || undefined,
    score: c.score_label || undefined,
    date,
    instant,
    fait,
    faitDecoupe: faitDecoupeDe(t, c),
    arme,
    badge,
    distance,
    titre,
    rejeu: {
      disponible: c.replay_available,
      search: { t: String(c.instant_ms), clock: c.clock === 'film' ? 'film' : 'match' },
      libelle: instant
        ? t.tileOpenReplay(instant, OUVERTURE_AVANT_EVENEMENT_MS / 1000)
        : t.tileOpenReplayNoInstant(OUVERTURE_AVANT_EVENEMENT_MS / 1000),
    },
  }
}

/**
 * infobulleDuPoint — le texte court du point survolé sur le plan : le gamertag de l'autre joueur,
 * l'arme si elle est connue, la distance si les deux points le sont.
 */
export function infobulleDuPoint(t: TacticalText, c: TacticalContribution, locale: Locale): string {
  return [c.autre_gamertag, armeDe(t, c), distanceDe(t, c, locale)].filter(Boolean).join(' · ')
}

/**
 * positionEtiquette — où poser le nom de zone sur le plan, en fractions du cadre : à côté de la
 * cellule choisie, à droite tant qu'elle est dans la partie gauche du plan, à gauche au-delà ;
 * centrée verticalement sauf contre les bords. `null` quand la cellule tombe hors du cadre, ou
 * hors de la FENÊTRE visible quand le plan est grossi.
 */
export function positionEtiquette(
  selected: { col: number; row: number },
  repere: RepereTactique,
  fenetre: FenetreDuPlan = repere,
): { left: string; top: string; transform: string } | null {
  // Le cadre de la cellule pour un canvas de largeur 1 : ses coordonnées sont des fractions de la
  // largeur, et la hauteur du cadre vaut 1 / rapport.
  const r = rectSelection(selected, repere, 1, fenetre)
  if (!r) return null
  const cx = r.x + r.size / 2
  const cy = (r.y + r.size / 2) * repereAspect(fenetre)
  if (cx < 0 || cx > 1 || cy < 0 || cy > 1) return null
  const droite = cx < ETIQUETTE_BASCULE
  const left = droite ? r.x + r.size : r.x
  let ty = '-50%'
  if (cy < ETIQUETTE_BORD) ty = '0'
  else if (cy > 1 - ETIQUETTE_BORD) ty = '-100%'
  return {
    left: `${(left * 100).toFixed(2)}%`,
    top: `${(cy * 100).toFixed(2)}%`,
    transform: `translate(${droite ? '4px' : 'calc(-100% - 4px)'}, ${ty})`,
  }
}

/**
 * titreDeZone — le titre de la carte de zone : « Zone sélectionnée » sans sélection ou tant que le
 * détail n'a pas répondu ; puis le nom en jeu de la zone (déjà dans la langue de la page), ou « Zone sans
 * nom » quand aucune zone ne la nomme.
 */
export function titreDeZone(
  t: TacticalText,
  aUneCellule: boolean,
  reponse: TacticalCelluleReponse | undefined,
): string {
  if (!aUneCellule || !reponse) return t.zoneTitle
  if (!reponse.zone) return t.zoneUnnamed
  return reponse.zone.nom
}
