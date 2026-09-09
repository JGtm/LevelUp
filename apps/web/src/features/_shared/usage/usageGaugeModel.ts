/**
 * usageGaugeModel.ts — LA FORME « écart à la parité » (jauge) du bloc « usages d'équipement,
 * armes spéciales et objectifs » : sa valeur, son trait de parité, la pile des trois issues et
 * les deux repères de taux, ses deux textes.
 *
 * Extrait de `session-detail/usageLogic.ts` le 2026-09-09 (étape E5.1bis, scission de taille —
 * CLAUDE.md n°5, 680 L avant scission) au moment du déménagement du bloc vers
 * `features/_shared/usage/` (étape E5.1, PLAN_EQUIPEMENT_GACHIS_2026-09-09.md).
 *
 * `undefined` N'EST PAS ZÉRO : le contrat (domain/session_usage.go) omet un champ
 * quand son dénominateur est nul ou son scope vide (0/0 n'est pas 0 %). Chaque
 * projection rend alors « non mesuré » (`null` → tiret), jamais un zéro inventé.
 *
 * Pur : aucun React, aucune couleur en dur, aucune lecture de store — les libellés
 * viennent de `usageI18n`, les encres arrivent par callbacks de l'appelant.
 */
import type { SemanticToken } from '@/lib/accessibility'
import type { SessionUsageOutcomes } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import { formatUsageCount, formatUsagePct } from './usageFormat'
import type { UsageText } from './usageI18n'

/**
 * Une jauge 0..100 % : sa valeur, son trait de parité, ses deux textes.
 *
 * PLUS D'ÉTENDUE MIN/MAX (revue de lisibilité 2026-09-09, D3). Le rail portait un second
 * trait fin, sans légende, disant l'écart entre le meilleur et le pire match — la BANDE DE
 * RÉGULARITÉ de la même carte dit exactement cela, case par match, et se lit. Deux formes
 * pour une information, dont une muette : celle qui ne parle pas a été retirée du modèle,
 * pas seulement cachée du rendu (règle « 0 code mort »).
 *
 * `honestyText` (le compte brut) reste dans le MODÈLE mais ne sort plus en cellule : il
 * n'apparaît que dans `tooltip`. Un pourcentage doublé de sa fraction dans une colonne de
 * 3 lignes triple la charge de lecture sans rien ajouter (D2).
 */
export interface UsageGaugeModel {
  key: string
  valuePct: number | null
  parityPct: number | null
  valueText: string
  honestyText: string
  tooltip: string
  /**
   * LA PILE DES TROIS ISSUES (P1, P6, étape E4) — remplit la TRANCHE (0..`valuePct`),
   * jamais le reste du rail. Absent = rendu STRICTEMENT INCHANGÉ, un seul aplat
   * (même contrat que `ValueGridCell.segments`, E1) : toute grandeur sans
   * `SessionUsageOutcomes` (armes spéciales, objectifs, familles hors bilan) n'a
   * jamais porté cette clé et continue de rendre exactement comme avant.
   */
  segments?: UsageGaugeOutcomeSegment[]
  /**
   * LES DEUX REPÈRES DE TAUX DANS LA TRANCHE (P7, décision produit du §3.2) : des
   * MARQUES, jamais un chiffre affiché (E4.2) — le texte vit dans `tooltip`. Chacun
   * est un pourcentage RELATIF À LA TRANCHE (même dénominateur que `segments`,
   * PAS le rail entier) : `null` quand la référence n'a pas de scope à camp connu
   * (P7 exclut le joueur — session FFA, aucune référence d'équipe ne se calcule),
   * jamais un 0 % inventé.
   */
  teammatesRatePct: number | null
  opponentsRatePct: number | null
}

/** L'ordre canonique des trois issues : utilisé → gardé → lâché, la gamme ordinale bon /
 * neutre / mauvais de la table §3.1 du plan équipement (décision S10 du master plan,
 * 2026-09-09 : la vue match et ce bloc empilaient dans deux ordres différents). */
const OUTCOME_ORDER = ['used', 'kept', 'dropped'] as const
export type UsageOutcomeKind = (typeof OUTCOME_ORDER)[number]

/** Le jeton d'une issue — table normative §3.1 du plan. */
const OUTCOME_TOKENS: Record<UsageOutcomeKind, SemanticToken> = {
  used: 'divergent-pos',
  dropped: 'divergent-neg',
  kept: 'divergent-neutral',
}

function outcomeKindLabel(kind: UsageOutcomeKind, t: UsageText): string {
  switch (kind) {
    case 'used':
      return t.outcomeUsed
    case 'dropped':
      return t.outcomeDropped
    case 'kept':
      return t.outcomeKept
  }
}

export interface UsageGaugeOutcomeSegment {
  key: UsageOutcomeKind
  /** Fraction 0..1 DE LA TRANCHE (used+kept+dropped = 100 %), jamais du rail entier. */
  fraction: number
  token: SemanticToken
  label: string
}

/**
 * buildOutcomeSegments — la pile utilisé → gardé → lâché (ordre §3.1 / S10), UNIQUEMENT
 * les issues NON NULLES (« une famille sans troisième issue rend deux segments »,
 * E4.6) — un segment à fraction 0 ne se dessine pas. `outcomes` absent, ou dont la
 * somme des trois issues est nulle (aucun objet mesuré) : PAS de pile — le rendu
 * simple (un seul aplat) reste la vérité, jamais une pile à une seule couleur.
 *
 * EXPORTÉE depuis le 2026-09-09 (E5.8, PLAN_EQUIPEMENT_GACHIS_2026-09-09) :
 * `usageCountsModel.ts` (variante comptes, Synthèse/Escouade) réutilise la MÊME pile —
 * `EquipmentUsageFamilyLine`/`EquipmentUsagePlayerLine` portent les mêmes champs
 * `used`/`kept`/`dropped` que `SessionUsageOutcomes` (structurellement compatibles),
 * donc aucune seconde copie de cette fonction (CLAUDE.md n°6).
 */
export function buildOutcomeSegments(
  outcomes: SessionUsageOutcomes | null | undefined,
  t: UsageText,
): UsageGaugeOutcomeSegment[] | undefined {
  if (outcomes == null) return undefined
  const total = outcomes.used + outcomes.kept + outcomes.dropped
  if (total <= 0) return undefined
  const segments = OUTCOME_ORDER.filter((kind) => outcomes[kind] > 0).map((kind) => ({
    key: kind,
    fraction: outcomes[kind] / total,
    token: OUTCOME_TOKENS[kind],
    label: outcomeKindLabel(kind, t),
  }))
  return segments
}

/** Une ligne de jauges : la grandeur, et ses trois dénominateurs (§7 du handoff). */
export interface UsageGaugeRowModel {
  key: string
  label: string
  gauges: UsageGaugeModel[]
  /**
   * Cette ligne est le TOTAL des lignes qui la précèdent (D6) — la grille la sépare d'un
   * filet et la pose en teinte secondaire. Le seul cas aujourd'hui : « toutes armes
   * spéciales », qui est exactement la somme de ses familles ; affichée comme leur égale,
   * elle se lisait comme une grandeur de plus.
   */
  isTotal?: boolean
}

/** Le sous-ensemble « parts » commun aux grandeurs, familles d'arme et rôles. */
export interface UsageSharesLike {
  player_total: number
  team_total?: number
  lobby_total: number
  team_share_of_lobby_pct?: number
  player_share_of_team_pct?: number
  player_share_of_lobby_pct?: number
}

export interface UsageGaugeRowInput {
  key: string
  label: string
  shares: UsageSharesLike
  isDuration?: boolean
  teamParityPct: number | null | undefined
  lobbyParityPct: number | null | undefined
  teamOfLobbyParityPct: number | null | undefined
  /** Cette grandeur est le total des lignes qui la précèdent (cf. `UsageGaugeRowModel`). */
  isTotal?: boolean
  /**
   * Les trois issues DU JOUEUR et les deux repères de taux (étape E4, contrat étendu
   * en E3). Absent pour toute grandeur hors bilan d'équipement (armes spéciales,
   * objectifs) — jamais posé à zéro.
   */
  outcomes?: SessionUsageOutcomes | null
  t: UsageText
  locale: Locale
}

/**
 * buildGaugeRow — les trois jauges d'une grandeur : mon équipe dans le lobby, ma part
 * dans mon équipe, ma part dans le lobby. Chaque jauge porte sa part (l'axe) et son
 * compte brut (l'honnêteté, en infobulle seulement — D2). Une part absente du contrat
 * rend une jauge VIDE au tiret — jamais un 0 %.
 *
 * L'ORDRE DES JAUGES EST UN CONTRAT DE RENDU : la 2e (`player-of-team`) est celle que la
 * grille montre seule quand le repli est fermé — `PRIMARY_GAUGE_INDEX`, `UsageForms`.
 *
 * `outcomes` NE S'APPLIQUE QU'AUX DEUX JAUGES DE PART DU JOUEUR (`player-of-team`,
 * `player-of-lobby`) — jamais à `team-of-lobby` : les trois issues sont une grandeur
 * DU JOUEUR (domain.SessionUsageOutcomes ne porte que « mine »), quand la première
 * jauge mesure la part de MON ÉQUIPE dans le lobby, une population entière sans
 * porteur individuel.
 */
export function buildGaugeRow(input: UsageGaugeRowInput): UsageGaugeRowModel {
  const { shares, t, locale } = input
  const isDur = input.isDuration === true
  const count = (v: number | null | undefined) => formatUsageCount(v, locale, isDur)
  const segments = buildOutcomeSegments(input.outcomes, t)
  const teammatesRatePct = input.outcomes?.teammates_used_rate_pct ?? null
  const opponentsRatePct = input.outcomes?.opponents_used_rate_pct ?? null

  const gauge = (
    key: string,
    gaugeLabel: string,
    valuePct: number | null | undefined,
    parityPct: number | null | undefined,
    numerator: number | null | undefined,
    denominator: number | null | undefined,
    withOutcomes: boolean,
  ): UsageGaugeModel => {
    const valueText = formatUsagePct(valuePct, locale)
    const honestyText = t.honestyFmt(count(numerator), count(denominator))
    let tooltip = t.gaugeTipFmt(input.label, gaugeLabel, valueText, honestyText)
    const gaugeSegments = withOutcomes ? segments : undefined
    if (gaugeSegments != null && input.outcomes != null) {
      tooltip = t.gaugeOutcomeTipFmt(
        tooltip,
        count(input.outcomes.used),
        count(input.outcomes.kept),
        count(input.outcomes.dropped),
      )
    }
    const gaugeTeammatesRatePct = withOutcomes ? teammatesRatePct : null
    const gaugeOpponentsRatePct = withOutcomes ? opponentsRatePct : null
    if (gaugeTeammatesRatePct != null || gaugeOpponentsRatePct != null) {
      tooltip = t.gaugeReferenceTipFmt(
        tooltip,
        formatUsagePct(gaugeTeammatesRatePct, locale),
        formatUsagePct(gaugeOpponentsRatePct, locale),
      )
    }
    return {
      key,
      valuePct: valuePct ?? null,
      parityPct: parityPct ?? null,
      valueText,
      honestyText,
      tooltip,
      segments: gaugeSegments,
      teammatesRatePct: gaugeTeammatesRatePct,
      opponentsRatePct: gaugeOpponentsRatePct,
    }
  }

  return {
    key: input.key,
    label: input.label,
    isTotal: input.isTotal,
    gauges: [
      gauge(
        'team-of-lobby',
        t.gaugeTeamOfLobby,
        shares.team_share_of_lobby_pct,
        input.teamOfLobbyParityPct,
        shares.team_total,
        shares.lobby_total,
        false,
      ),
      gauge(
        'player-of-team',
        t.gaugePlayerOfTeam,
        shares.player_share_of_team_pct,
        input.teamParityPct,
        shares.player_total,
        shares.team_total,
        true,
      ),
      gauge(
        'player-of-lobby',
        t.gaugePlayerOfLobby,
        shares.player_share_of_lobby_pct,
        input.lobbyParityPct,
        shares.player_total,
        shares.lobby_total,
        true,
      ),
    ],
  }
}
