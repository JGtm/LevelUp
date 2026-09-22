/**
 * sessionRange.logic — les décisions PURES de la carte « Portée des engagements » de la
 * colonne de session (lot W, D23-4 : le NUAGE DE LA PÉRIODE À UN SEUL JOUEUR).
 *
 * CE QUI A CHANGÉ AU LOT W. Le lot O dessinait un bâton par match de la session, et posait
 * ses bandes de rôle sur les TIERS DE LA SESSION ELLE-MÊME : chaque soirée redéfinissait ses
 * propres rôles, donc deux soirées n'étaient pas comparables et la question du lecteur
 * — « est-ce que ce soir je joue plus loin que d'habitude ? » — n'avait aucun repère dans le
 * graphe. Le nuage remplace le bâton : l'axe porte la PÉRIODE DE RÉFÉRENCE du joueur
 * (`range_reference`, lot U), la session s'y lit en surbrillance, et les bandes sont assises
 * sur les SEUILS SERVIS — les mêmes que l'Escouade, calculés une fois côté Go.
 *
 * LES SEUILS NE SE RECALCULENT PAS ICI (D23-4). `role_low_m` / `role_high_m` arrivent du
 * contrat ; absents (sous trois matchs pleins), il n'y a PAS de bandes — jamais de bandes
 * fabriquées côté web, qui divergeraient de celles de l'Escouade.
 *
 * LES DÉCISIONS COMMUNES SONT IMPORTÉES, PAS RECOPIÉES (CLAUDE.md n°6) : plancher de mesure,
 * ordre des matchs, étiquettes « #N · carte », projection en points et fenêtre glissante
 * viennent de `features/squad/squadRangeRoles.logic` (import cross-feature
 * `session-detail=>squad`, déjà allowlisté).
 *
 * Pur : aucun React, aucune couleur en dur.
 */
import {
  FENETRE_ROLE,
  PLANCHER_MESURE,
  TAILLE_POINT_MIN,
  categoriesMatchs,
  ordonnerProfils,
  seriesPortee,
  type PointPortee,
  type RoleDePortee,
  type SeriePortee,
  type SeuilsRoles,
} from '@/features/squad/squadRangeRoles.logic'
import type { MatchRangeBlock, MatchRangeProfile, RangeReferenceBlock } from '@/lib/api/types'

export { FENETRE_ROLE, PLANCHER_MESURE, TAILLE_POINT_MIN }
export type { PointPortee, RoleDePortee, SeriePortee, SeuilsRoles }

/** Le nuage prêt à peindre : ses points, ses bandes, et la fenêtre de la session. */
export interface NuagePortee {
  /** Les matchs de l'axe X, du plus ancien au plus récent. */
  profils: MatchRangeProfile[]
  /** Les étiquettes « #N · carte » de l'axe X. */
  categories: string[]
  /** La série du joueur consulté — `null` quand rien n'est mesuré. */
  serie: SeriePortee | null
  /** Les bandes de rôle, SERVIES ou absentes. Jamais recalculées. */
  seuils: SeuilsRoles | null
  /** Indices INCLUSIFS des matchs de la session dans `profils` — `null` en repli. */
  surbrillance: { debut: number; fin: number } | null
  /** `true` = nuage de période ; `false` = repli sur les seuls matchs de la session. */
  periode: boolean
  /** Les `match_id` de la session affichée — l'encre pleine des points. */
  idsSession: ReadonlySet<string>
}

/**
 * seuilsServis lit les deux bornes de rôle du contrat. Le serveur les sert ENSEMBLE ou pas
 * du tout (sous trois matchs pleins) : une seule des deux présente serait une bande sans
 * l'autre, donc rien.
 */
export function seuilsServis(reference: RangeReferenceBlock | null | undefined): SeuilsRoles | null {
  const bas = reference?.role_low_m
  const haut = reference?.role_high_m
  if (bas == null || haut == null) return null
  return { bas, haut }
}

/**
 * La série du joueur consulté. Le scope Sessions ne sert QU'UN joueur par profil (lots N2 et
 * U) ; `meLabel` ne départage donc que le cas dégradé d'un serveur qui en servirait
 * plusieurs, par gamertag insensible à la casse (le slug de route est en minuscules, le
 * titre écrit le gamertag comme il l'entend).
 */
function serieConsultee(series: SeriePortee[], meLabel: string): SeriePortee | null {
  if (series.length === 0) return null
  if (series.length === 1) return series[0]
  const cible = meLabel.toLowerCase()
  return series.find((s) => s.gamertag.toLowerCase() === cible) ?? series[0]
}

/** Les identifiants des matchs de la session affichée — la fenêtre à surligner. */
function idsDeLaSession(session: MatchRangeBlock | null | undefined): Set<string> {
  return new Set((session?.profiles ?? []).map((p) => p.match_id))
}

/**
 * nuagePortee compose le nuage à peindre.
 *
 * NOMINAL : l'axe est la période de référence (`reference`), la session s'y surligne.
 * REPLI (référence absente — vieux serveur, ou période tautologique : la référence se
 * réduirait aux matchs de la session) : l'axe est la session seule, sans bandes et sans
 * surbrillance — il n'y a alors aucune population où situer la soirée.
 */
export function nuagePortee(
  reference: RangeReferenceBlock | null | undefined,
  session: MatchRangeBlock | null | undefined,
  meLabel: string,
): NuagePortee {
  const idsSession = idsDeLaSession(session)
  const profilsReference = reference?.profiles ?? []
  const periode = profilsReference.length > 0
  const profils = ordonnerProfils(periode ? profilsReference : (session?.profiles ?? []))
  const categories = categoriesMatchs(profils)
  const serie = serieConsultee(seriesPortee(profils, [meLabel]), meLabel)

  let surbrillance: { debut: number; fin: number } | null = null
  if (periode) {
    const positions = profils
      .map((p, i) => (idsSession.has(p.match_id) ? i : -1))
      .filter((i) => i >= 0)
    if (positions.length > 0) {
      surbrillance = { debut: positions[0], fin: positions[positions.length - 1] }
    }
  }

  return {
    profils,
    categories,
    serie,
    seuils: periode ? seuilsServis(reference) : null,
    surbrillance,
    periode,
    idsSession,
  }
}

/** `true` quand le point appartient à la session affichée (encre pleine). */
export function pointDeLaSession(nuage: NuagePortee, point: PointPortee): boolean {
  return !nuage.periode || nuage.idsSession.has(point.matchId)
}
