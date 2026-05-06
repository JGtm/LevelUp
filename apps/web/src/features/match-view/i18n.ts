/**
 * i18n strings — feature match-view (header refonte 2026-05-05, mock C).
 *
 * Strings UI dédiées au header (nav + actions + labels). Les libellés de
 * stats Halo (modes, maps, playlists) viennent du pipeline backend FR
 * (asset_translations / mode_name_tr) — voir buildMatchHeader Go.
 */

export type MatchViewLocale = 'fr' | 'en'

export interface MatchViewText {
  prevMatch: string
  nextMatch: string
  matchCounter: (n: number, total: number) => string
  copyMatchId: string
  copied: string
  copyShort: string
  copyTooltip: string
  markIrrelevant: string
  reactivate: string
  excludeShort: string
  excludeTooltip: string
  reactivateTooltip: string
  performance: string
  rank: string
  addFavorite: string
  removeFavorite: string
  mapUnknown: string
  noRank: string
  exitContext: string
  outcomeWin: string
  outcomeLoss: string
  outcomeDraw: string
  outcomeDnf: string
  fromDate: string
  toDate: string
  // Charts résumé
  chartKdaTitle: string
  chartSpreeTitle: string
  seriesActual: string
  seriesExpected: string
  seriesHistAvg: string
  labelKills: string
  labelDeaths: string
  labelAssists: string
  labelSpree: string
  labelHeadshots: string
  labelPerfectKills: string
  noHistData: string
  duration: string
  // Radar synergie (joueur actif)
  chartSynergyRadarTitle: string
  radarAxisCombat: string
  radarAxisSurvival: string
  radarAxisSupport: string
  radarAxisScore: string
  radarAxisObjective: string
  radarAxisImpact: string
  radarTooltipImpact: string
  radarTooltipCombat: string
  radarTooltipSurvival: string
  radarTooltipSupport: string
  radarTooltipScore: string
  radarTooltipObjective: string
  radarTooltipGlossaryLink: string
  // Charts armes
  chartWeaponPieTitle: string
  labelPowerWeapon: string
  labelMelee: string
  labelOtherKills: string
  weaponUnknownPrefix: string
  weaponOtherGroup: string
  // Section médias (dans onglet Résumé)
  sectionMedia: string
  mediaNoCaptures: string
  mediaNoCapturesDesc: string
  // Résumé — médailles & citations
  sectionMedals: string
  sectionCitations: string
  newlyMastered: string
  noMedals: string
  noCitations: string
  // Onglet Combat — charts en haut (mock match_view.09 / .10 / .11 / .12)
  combatHighlights: string
  combatKdCumulTitle: string
  combatTugOfWarTitle: string
  combatCadenceTitle: string
  combatKillsLabel: string
  combatDeathsLabel: string
  combatTeamLabel: string
  combatEnemyLabel: string
  combatNemesisTitle: string
  combatBullyTitle: string
  combatNoNemesis: string
  combatKilledMeFmt: (n: number) => string
  combatIKilledFmt: (n: number) => string
  combatNoData: string
}

export const MATCH_VIEW_TEXT: Record<MatchViewLocale, MatchViewText> = {
  fr: {
    prevMatch: 'Match précédent',
    nextMatch: 'Match suivant',
    matchCounter: (n, total) => `Match ${n}/${total}`,
    copyMatchId: "Copier l'ID du match",
    copied: 'Copié',
    copyShort: 'Copier ID',
    copyTooltip: "Copier l'identifiant unique de ce match dans le presse-papier",
    markIrrelevant: 'Marquer comme non pertinent',
    reactivate: 'Réactiver',
    excludeShort: 'Exclure',
    excludeTooltip: 'Exclure ce match des statistiques et analyses',
    reactivateTooltip: 'Réintégrer ce match dans les statistiques',
    performance: 'Performance',
    rank: 'Rang',
    addFavorite: 'Ajouter aux favoris',
    removeFavorite: 'Retirer des favoris',
    mapUnknown: 'Map inconnue',
    noRank: 'Pas de rang',
    exitContext: 'Sortir du contexte',
    outcomeWin: 'Victoires',
    outcomeLoss: 'Défaites',
    outcomeDraw: 'Égalités',
    outcomeDnf: 'Non terminés',
    fromDate: 'Depuis',
    toDate: "Jusqu'au",
    chartKdaTitle: 'F/D/A : Réel vs Attendu vs Moy. hist.',
    chartSpreeTitle: 'Folie meurtrière · Tirs à la tête · Frags parfaits',
    seriesActual: 'Réel',
    seriesExpected: 'Attendu',
    seriesHistAvg: 'Hist. Moy.',
    labelKills: 'F',
    labelDeaths: 'D',
    labelAssists: 'A',
    labelSpree: 'Folie meurtrière',
    labelHeadshots: 'Tirs à la tête',
    labelPerfectKills: 'Frags parfaits',
    noHistData: 'Pas de données historiques disponibles',
    duration: 'Durée',
    chartSynergyRadarTitle: 'Radar synergie',
    radarAxisCombat: 'Combat',
    radarAxisSurvival: 'Survie',
    radarAxisSupport: 'Support',
    radarAxisScore: 'Score',
    radarAxisObjective: 'Objectif',
    radarAxisImpact: 'Impact',
    radarTooltipImpact: 'Rendement offensif — 225 × (frags + ass/3) / dégâts. P80 = 0,83.',
    radarTooltipCombat: 'Frags + tirs à la tête + frags parfaits, pondérés par la précision.',
    radarTooltipSurvival: 'Résistance défensive — dégâts / (225 × morts). P80 = 1,59.',
    radarTooltipSupport: 'Assists × 50.',
    radarTooltipScore: 'Score résiduel après frags (×100) et assists (×50) : médailles et streaks.',
    radarTooltipObjective: "Points d'objectif (PersonalScoreAwards).",
    radarTooltipGlossaryLink: '→ Glossaire',
    chartWeaponPieTitle: 'Frags par arme',
    labelPowerWeapon: 'Armes lourdes',
    labelMelee: 'Mêlée',
    labelOtherKills: 'Autres',
    weaponUnknownPrefix: 'Arme inconnue',
    weaponOtherGroup: 'Autres armes',
    sectionMedia: 'Médias',
    mediaNoCaptures: 'Aucune capture',
    mediaNoCapturesDesc: 'Les screenshots et clips associés à ce match apparaîtront ici.',
    sectionMedals: 'Médailles',
    sectionCitations: 'Citations',
    newlyMastered: 'Maîtrisé !',
    noMedals: 'Aucune médaille',
    noCitations: 'Aucune citation',
    combatHighlights: 'Faits marquants',
    combatKdCumulTitle: 'Frags / Morts cumulés',
    combatTugOfWarTitle: 'Dominance par tranche de temps',
    combatCadenceTitle: 'Cadence des frags',
    combatKillsLabel: 'Frags',
    combatDeathsLabel: 'Morts',
    combatTeamLabel: 'Mon équipe',
    combatEnemyLabel: 'Adversaires',
    combatNemesisTitle: 'Némésis',
    combatBullyTitle: 'Souffre-douleur',
    combatNoNemesis: '—',
    combatKilledMeFmt: (n) => `T'a victimisé ${n} fois`,
    combatIKilledFmt: (n) => `Tu l'as persécuté ${n} fois`,
    combatNoData: 'Pas de données disponibles',
  },
  en: {
    prevMatch: 'Previous match',
    nextMatch: 'Next match',
    matchCounter: (n, total) => `Match ${n}/${total}`,
    copyMatchId: 'Copy match ID',
    copied: 'Copied',
    copyShort: 'Copy ID',
    copyTooltip: "Copy this match's unique identifier to clipboard",
    markIrrelevant: 'Mark as irrelevant',
    reactivate: 'Reactivate',
    excludeShort: 'Exclude',
    excludeTooltip: 'Exclude this match from stats and analyses',
    reactivateTooltip: 'Re-include this match in stats and analyses',
    performance: 'Performance',
    rank: 'Rank',
    addFavorite: 'Add to favorites',
    removeFavorite: 'Remove from favorites',
    mapUnknown: 'Unknown map',
    noRank: 'No rank',
    exitContext: 'Exit context',
    outcomeWin: 'Wins',
    outcomeLoss: 'Losses',
    outcomeDraw: 'Draws',
    outcomeDnf: 'DNF',
    fromDate: 'From',
    toDate: 'To',
    chartKdaTitle: 'K/D/A: Actual vs Expected vs Hist. Avg.',
    chartSpreeTitle: 'Spree · Headshots · Perfect kills',
    seriesActual: 'Actual',
    seriesExpected: 'Expected',
    seriesHistAvg: 'Hist. Avg.',
    labelKills: 'K',
    labelDeaths: 'D',
    labelAssists: 'A',
    labelSpree: 'Killing Spree',
    labelHeadshots: 'Headshots',
    labelPerfectKills: 'Perfect kills',
    noHistData: 'No historical data available',
    duration: 'Duration',
    chartSynergyRadarTitle: 'Synergy radar',
    radarAxisCombat: 'Combat',
    radarAxisSurvival: 'Survival',
    radarAxisSupport: 'Support',
    radarAxisScore: 'Score',
    radarAxisObjective: 'Objective',
    radarAxisImpact: 'Impact',
    radarTooltipImpact: 'Offensive conversion — 225 × (kills + ass/3) / damage. P80 = 0.83.',
    radarTooltipCombat: 'Kills + headshots + perfect kills, weighted by accuracy.',
    radarTooltipSurvival: 'Defensive resistance — damage / (225 × deaths). P80 = 1.59.',
    radarTooltipSupport: 'Assists × 50.',
    radarTooltipScore: 'Residual score after kills (×100) and assists (×50): medals and streaks.',
    radarTooltipObjective: 'Objective points (PersonalScoreAwards).',
    radarTooltipGlossaryLink: '→ Glossary',
    chartWeaponPieTitle: 'Frags by weapon',
    labelPowerWeapon: 'Power weapons',
    labelMelee: 'Melee',
    labelOtherKills: 'Other',
    weaponUnknownPrefix: 'Unknown weapon',
    weaponOtherGroup: 'Other weapons',
    sectionMedia: 'Media',
    mediaNoCaptures: 'No captures',
    mediaNoCapturesDesc: 'Screenshots and clips associated with this match will appear here.',
    sectionMedals: 'Medals',
    sectionCitations: 'Commendations',
    newlyMastered: 'Mastered!',
    noMedals: 'No medals',
    noCitations: 'No commendations',
    combatHighlights: 'Highlights',
    combatKdCumulTitle: 'Cumulative Kills / Deaths',
    combatTugOfWarTitle: 'Time-window dominance',
    combatCadenceTitle: 'Kill cadence',
    combatKillsLabel: 'Kills',
    combatDeathsLabel: 'Deaths',
    combatTeamLabel: 'My team',
    combatEnemyLabel: 'Opponents',
    combatNemesisTitle: 'Nemesis',
    combatBullyTitle: 'Bully target',
    combatNoNemesis: '—',
    combatKilledMeFmt: (n) => `Killed you ${n} times`,
    combatIKilledFmt: (n) => `You killed them ${n} times`,
    combatNoData: 'No data available',
  },
}

/**
 * buildContextLabel — produit un label localisé depuis un MatchFilterSpec.
 *
 * Phase 2b : utilisé quand la cascade tombe sur l'API avec spec URL (pas de
 * filtersLabel pré-localisé dans le matchNavContext). Format compact :
 *   "Classée Arena · Victoires · Depuis 01/04/2026"
 */
import type { MatchFilterSpec } from '@/lib/match-nav/navContext'

export function buildContextLabel(
  spec: MatchFilterSpec | null | undefined,
  locale: MatchViewLocale,
): string {
  if (!spec) return ''
  const t = MATCH_VIEW_TEXT[locale]
  const parts: string[] = []
  if (spec.playlist_name) parts.push(spec.playlist_name)
  if (spec.mode_category) parts.push(spec.mode_category)
  if (spec.outcome) {
    const map: Record<string, string> = {
      win: t.outcomeWin,
      loss: t.outcomeLoss,
      draw: t.outcomeDraw,
      dnf: t.outcomeDnf,
    }
    const lbl = map[spec.outcome]
    if (lbl) parts.push(lbl)
  }
  if (spec.date_from || spec.date_to) {
    const intlLocale = locale === 'en' ? 'en-GB' : 'fr-FR'
    const fmt = (iso: string) => {
      const d = new Date(iso)
      if (isNaN(d.getTime())) return iso
      return new Intl.DateTimeFormat(intlLocale, { day: '2-digit', month: '2-digit', year: 'numeric' }).format(d)
    }
    if (spec.date_from && spec.date_to) {
      parts.push(`${fmt(spec.date_from)} → ${fmt(spec.date_to)}`)
    } else if (spec.date_from) {
      parts.push(`${t.fromDate} ${fmt(spec.date_from)}`)
    } else if (spec.date_to) {
      parts.push(`${t.toDate} ${fmt(spec.date_to)}`)
    }
  }
  if (spec.session_id) parts.push(`#${spec.session_id}`)
  return parts.join(' · ')
}
