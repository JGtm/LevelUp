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
import { formatClock } from '@/lib/replay/replayLogic'

import type { TacticalText } from './i18n'
import { rectSelection, repereAspect, type RepereTactique, type TacticalQuestion } from './tacticalView.logic'

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
  const loc = locale === 'en' ? 'en-GB' : 'fr-FR'
  const date = new Intl.DateTimeFormat(loc, { timeZone: timezone, day: '2-digit', month: '2-digit', year: 'numeric' })
  const heure = new Intl.DateTimeFormat(loc, { timeZone: timezone, hour: '2-digit', minute: '2-digit', hourCycle: 'h23' })
  return `${date.format(d)} · ${heure.format(d)}`
}

/** Ce que la mini-tuile d'une contribution affiche. */
interface ModeleDeTuile {
  mode?: string
  score?: string
  date: string
  instant: string
  fait: string
  /** L'arme, ou la catégorie de la source traduite ; absente sinon. Seule partie qui se tronque. */
  arme?: string
  badge?: string
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

function armeDe(t: TacticalText, c: TacticalContribution, locale: Locale): string | undefined {
  const arme = locale === 'en' ? c.arme_label_en || c.arme_label : c.arme_label || c.arme_label_en
  if (arme) return arme
  return c.categorie_source ? t.tileCategories[c.categorie_source] : undefined
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
  const instant = formatClock(c.instant_ms)
  const fait = faitDe(t, c)
  const arme = armeDe(t, c, locale)
  const badge = badgeDe(t, c, locale)
  const date = dateEtHeure(c.match_started_at, timezone, locale)
  const titre = [c.mode_label, c.score_label, issue, date, instant, fait, arme, badge].filter(Boolean).join(' · ')
  return {
    mode: c.mode_label || undefined,
    score: c.score_label || undefined,
    date,
    instant,
    fait,
    arme,
    badge,
    titre,
    rejeu: {
      disponible: c.replay_available,
      search: { t: String(c.instant_ms), clock: c.clock === 'film' ? 'film' : 'match' },
      libelle: t.tileOpenReplay(instant),
    },
  }
}

/**
 * positionEtiquette — où poser le nom de zone sur le plan, en fractions du cadre : à côté de la
 * cellule choisie, à droite tant qu'elle est dans la partie gauche du plan, à gauche au-delà ;
 * centrée verticalement sauf contre les bords. `null` quand la cellule tombe hors du cadre.
 */
export function positionEtiquette(
  selected: { col: number; row: number },
  repere: RepereTactique,
): { left: string; top: string; transform: string } | null {
  // Le cadre de la cellule pour un canvas de largeur 1 : ses coordonnées sont des fractions de la
  // largeur, et la hauteur du cadre vaut 1 / rapport.
  const r = rectSelection(selected, repere, 1)
  if (!r) return null
  const cy = (r.y + r.size / 2) * repereAspect(repere)
  const droite = r.x + r.size / 2 < ETIQUETTE_BASCULE
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
 * détail n'a pas répondu ; puis le nom en jeu de la zone dans la langue de la page, ou « Zone sans
 * nom » quand aucune zone ne la nomme.
 */
export function titreDeZone(
  t: TacticalText,
  locale: Locale,
  aUneCellule: boolean,
  reponse: TacticalCelluleReponse | undefined,
): string {
  if (!aUneCellule || !reponse) return t.zoneTitle
  if (!reponse.zone) return t.zoneUnnamed
  return locale === 'en' ? reponse.zone.nom_en : reponse.zone.nom_fr
}
