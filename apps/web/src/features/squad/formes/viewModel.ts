/**
 * viewModel.ts — CE QUE LES DIX-NEUF CARTES PARTAGENT : le bloc, les deux
 * dictionnaires, les formats de la langue courante, l'escouade avec ses encres,
 * et la façon de nommer un match et une arme.
 *
 * Construit UNE FOIS par la section et passé aux cartes : dix-neuf cartes qui
 * reconstruiraient chacune leur table d'armes et leur escouade recalculeraient
 * dix-neuf fois la même chose, et finiraient par diverger sur un détail (l'ordre
 * des joueurs, l'encre d'un coéquipier).
 */
import type {
  SquadFormesBlock,
  SquadFormesMatch,
  SquadFormesWeapon,
} from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import type { FormesCardsText } from './cardsI18n'
import { squadPlayerInk } from './colors'
import { formatCount, formatMatchTime, formatPct, formatSigned } from './format'
import type { FormesText } from './i18n'
import { allMatches, measuredMatches } from './model/access'
import { weaponIndex } from './model/pads'

/** Un membre de l'escouade affichée : son identité et son encre. */
export interface FormesSquadMember {
  xuid: string
  label: string
  ink: string
}

export interface FormesViewModel {
  block: SquadFormesBlock
  t: FormesText
  ct: FormesCardsText
  locale: Locale
  mainXuid: string
  squad: FormesSquadMember[]
  weapons: Record<string, SquadFormesWeapon>
  measured: SquadFormesMatch[]
  matches: SquadFormesMatch[]
  fmtPct: (v: number) => string
  fmtSigned: (v: number) => string
  fmtCount: (v: number, isDuration?: boolean) => string
  /** « 19:22 · Bastion » — l'identité courte d'un match. */
  matchLabel: (m: SquadFormesMatch) => string
  /** L'heure seule, pour l'axe de la bande. */
  matchTime: (m: SquadFormesMatch) => string
  /** La carte du match (sous-libellé). */
  matchMap: (m: SquadFormesMatch) => string
  /** Le nom d'une arme de socle, ou la réserve « arme non cataloguée ». */
  weaponLabel: (key: string) => string
}

/** Le libellé d'un membre : son gamertag, le nom de la page pour le joueur, son xuid en dernier. */
function nameOf(
  xuid: string,
  gamertag: string | undefined,
  mainXuid: string,
  mainPlayerLabel: string | undefined,
): string {
  if (gamertag != null && gamertag !== '') return gamertag
  if (xuid === mainXuid && mainPlayerLabel != null && mainPlayerLabel !== '') return mainPlayerLabel
  return xuid
}

export function buildFormesViewModel(
  block: SquadFormesBlock,
  t: FormesText,
  ct: FormesCardsText,
  locale: Locale,
  /**
   * Le nom du joueur de la page, tel que la RÉPONSE DE PAGE le porte
   * (`main_player`). Dernier filet contre un XUID à l'écran : le bloc le nomme
   * déjà, mais un scope dont aucun participant ne porte son gamertag l'avait
   * laissé sans nom (défaut mesuré le 2026-09-13).
   */
  mainPlayerLabel?: string,
): FormesViewModel {
  const mainXuid = block.main_xuid ?? ''
  const squad: FormesSquadMember[] = (block.squad ?? []).map((p, index) => ({
    xuid: p.xuid,
    // Un joueur sans gamertag garde son identifiant : aucune vie anonyme, jamais
    // un « inconnu » à l'écran — et le joueur de la page, lui, a toujours son nom.
    label: nameOf(p.xuid, p.gamertag, mainXuid, mainPlayerLabel),
    ink: squadPlayerInk(index),
  }))
  const weapons = weaponIndex(block)
  return {
    block,
    t,
    ct,
    locale,
    mainXuid,
    squad,
    weapons,
    measured: measuredMatches(block),
    matches: allMatches(block),
    fmtPct: (v) => formatPct(v, locale),
    fmtSigned: (v) => formatSigned(v, locale),
    fmtCount: (v, isDuration) => formatCount(v, locale, isDuration),
    matchLabel: (m) => {
      const time = formatMatchTime(m.start_time, locale)
      return m.mode_label ? `${time} · ${m.mode_label}` : time
    },
    matchTime: (m) => formatMatchTime(m.start_time, locale),
    matchMap: (m) => m.map_label ?? '',
    weaponLabel: (key) => {
      const label = weapons[key]?.label
      return label != null && label !== '' ? label : t.common.unknownWeapon
    },
  }
}
