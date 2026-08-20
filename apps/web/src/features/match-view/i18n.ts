/**
 * i18n strings — feature match-view (header refonte 2026-05-05, mock C).
 *
 * Strings UI dédiées au header (nav + actions + labels). Les libellés de
 * stats Halo (modes, maps, playlists) viennent du pipeline backend FR
 * (asset_translations / mode_name_tr) — voir buildMatchHeader Go.
 */

import type { Locale } from '@/lib/i18n/locale'

/** Alias de compat du type central `Locale` (lib/i18n/locale) : conservé car consommé dans plus de 5 fichiers du header match-view. */
export type MatchViewLocale = Locale

export interface MatchViewText {
  prevMatch: string
  nextMatch: string
  matchCounter: (n: number, total: number) => string
  copyMatchId: string
  copied: string
  copyShort: string
  copyTooltip: string
  replayShort: string
  replayTooltip: string
  markIrrelevant: string
  reactivate: string
  excludeShort: string
  excludeTooltip: string
  reactivateTooltip: string
  // Confirmation dialog d'exclusion / réactivation
  excludeConfirmTitle: string
  excludeConfirmBody: string
  reactivateConfirmTitle: string
  reactivateConfirmBody: string
  confirmAction: string
  cancelAction: string
  excludeRankedDenied: string
  excludeErrorRanked: string
  excludeErrorGeneric: string
  performance: string
  rank: string
  /** Libellé localisé de la sentinelle de tier « Placement » (phase de placement). */
  rankPlacement: string
  addFavorite: string
  removeFavorite: string
  mapUnknown: string
  pageErrorTitle: string
  pageRetry: string
  pagePartialLoad: string
  // État dédié 404 match_not_found — match absent du substrat local (pas encore
  // synchronisé, ou identifiant invalide). Remplace l'écran d'erreur générique
  // (retiré le 2026-07-25 avec le fallback LIVE du Match view, cf. BACKLOG).
  notSyncedTitle: string
  notSyncedDescription: string
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
  // Labels de mécaniques natives Halo 5 — colonnes du scoreboard (MatchScoreboard).
  labelGrenade: string
  labelAssassination: string
  labelGroundPound: string
  labelShoulderBash: string
  weaponUnknownPrefix: string
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
  // Commendations NATIVES (Halo 5) — affichées à la place des citations dérivées
  sectionNativeCommendations: string
  noNativeCommendations: string
  // Onglet Combat — charts en haut (mock match_view.09 / .10 / .11 / .12)
  combatHighlights: string
  combatKdCumulTitle: string
  combatTugOfWarTitle: string
  /**
   * LA COURBE DE SCORE, décodée du film du match (schéma 12). Elle n'apparaît que pour les
   * matchs dont un artefact de rejeu existe — d'où la note de source, qui dit à
   * l'utilisateur pourquoi cette carte est là sur ce match et pas sur le précédent.
   */
  scoreCurveTitle: string
  scoreCurveSource: string
  scoreCurveTruncated: string
  scoreCurveLead: string
  combatCadenceTitle: string
  combatKillsLabel: string
  combatDeathsLabel: string
  combatTeamLabel: string
  combatEnemyLabel: string
  // Histogramme momentum (carte Dominance) — libellés de tooltip.
  combatMomentumDelta: string
  combatMomentumCumul: string
  combatNemesisTitle: string
  combatBullyTitle: string
  combatNoNemesis: string
  combatKilledMeFmt: (n: number) => string
  combatIKilledFmt: (n: number) => string
  combatNoData: string
  // Overlay capture CTF (charts combat — câblé couche 2)
  combatCtfCaptureLabel: string
  combatCtfCaptureTooltip: (player: string, time: string) => string
  fragDiffNoData: string
  antagonistNoData: string
  impactBadgesNoData: string
  // Libellés des badges d'impact (Match flow), keyés par BadgeKey backend. Le
  // moteur analysis ne produit qu'un libellé FR (BadgeFR) → sous UI EN les cartes
  // restaient en FR (GH-7). Résolution front bilingue par clé, fallback = libellé
  // serveur pour une clé inconnue.
  impactBadgeNames: Record<string, string>
  // Breadcrumb retour (MatchBreadcrumb)
  back: string
  // Onglets de la page (GH2-B2)
  tabGeneral: string
  tabDetails: string
  // Titre du chart Antagonistes (GH2-B2)
  antagonistTitle: string
  // Sections de l'onglet Détails (titres type-1 du catalogue d'harmonisation)
  sectionFlow: string
  sectionDuels: string
  sectionEncounters: string
  // Scoreboard team header (Eagle / Cobra avec couleur team-ally/enemy)
  scoreboardTitle: string
  scoreboardNoData: string
  teamLabelFmt: (name: string) => string
  teamUnknown: string
  teamNumberedFmt: (n: number) => string
  teamMine: string
  teamEnemy: string
  // Scoreboard expander (port de match_view_scoreboard_detail.py)
  sbDetailWeapons: string
  sbDetailMedalsAndCitations: string
  sbDetailMedalsOnly: string
  sbDetailExpected: string
  sbDetailExpectedKills: string
  sbDetailExpectedDeaths: string
  sbDetailExpectedAssists: string
  sbDetailLocallyEstimated: string
  sbDetailLocallyEstimatedHint: string
  sbDetailAntagonist: string
  sbDetailNemesis: string
  sbDetailBully: string
  sbDetailLocal: string
  sbDetailLusr: string
  sbDetailCsr: string
  sbDetailBotNoteLabel: string
  sbDetailBotNoteValue: string
  sbDetailPlayerDb: string
  sbDetailSharedOnly: string
  sbDetailExplorePlayerFmt: (player: string) => string
  // Libellés colonnes scoreboard (utilisés par buildHighlightCols)
  sbColKda: string
  sbColMeleeKills: string
  sbColDamageDealt: string
  sbColDamageTaken: string
  sbColShotsHit: string
  sbColAccuracy: string
  sbColCsr: string
  sbColRank: string
  sbColScore: string
  sbColAssists: string
  sbColMaxSpree: string
  sbColHeadshots: string
  sbColPerfectKills: string
  sbColShotsFired: string
  sbColPowerWeapons: string
  sbColAvgLife: string
  sbColPlayer: string
  sbColTopWeapon: string
  // Tooltips d'en-tête de colonne (V72-04, icône ⓘ) — colonnes non évidentes.
  sbColCsrTooltip: string
  sbColLusrTooltip: string
  sbColRankTooltip: string
  sbColKdaTooltip: string
  sbColAccuracyTooltip: string
  sbColMaxSpreeTooltip: string
  sbColPerfectKillsTooltip: string
  sbColPowerWeaponsTooltip: string
  sbColMeleeKillsTooltip: string
  sbColAvgLifeTooltip: string
  sbColTopWeaponTooltip: string
  sbColOffensiveTooltip: string
  sbColDefensiveTooltip: string
  sbViewHistoryFmt: (gamertag: string) => string
  /** Format du score (séparateurs locale-sensitive : "12 345" FR / "12,345" EN). */
  sbFormatScore: (v: number) => string
  // Nav contextuelle — Phase 2c (descriptor → label compact)
  ctxRecent: string
  ctxFavorites: string
  ctxMedia: string
  ctxTopMatches: string
  ctxWithPlayerFmt: (gamertag: string) => string
  ctxSessionFmt: (date: string) => string
  ctxPeriodFromToFmt: (from: string, to: string) => string
  ctxPeriodFromFmt: (from: string) => string
  ctxPeriodToFmt: (to: string) => string
  ctxPlaylistFmt: (name: string) => string
  ctxModeFmt: (category: string) => string
  /** Compteur intégré : "Matchs récents 12/47" / "Recent matches 12/47". */
  matchCounterCtxFmt: (label: string, n: number, total: number) => string
  /** Section « Objectifs » du scoreboard (CTF/Zones/Oddball) — V72-03. `cols` :
   *  libellé + tooltip d'en-tête par clé de colonne objectif. */
  objectives: {
    title: string
    teamTotal: string
    cols: Record<string, { label: string; tooltip: string }>
  }
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
    replayShort: 'Rejeu 2D',
    replayTooltip: 'Voir le rejeu 2D de ce match (vue du dessus)',
    markIrrelevant: 'Marquer comme non pertinent',
    reactivate: 'Réactiver',
    excludeShort: 'Exclure',
    excludeTooltip: 'Exclure ce match des statistiques et analyses',
    reactivateTooltip: 'Réintégrer ce match dans les statistiques',
    excludeConfirmTitle: 'Exclure ce match ?',
    excludeConfirmBody:
      'Ce match sera marqué non pertinent et retiré des statistiques. Le score de performance et le LUSR des matchs ultérieurs seront recalculés (quelques secondes).',
    reactivateConfirmTitle: 'Réintégrer ce match ?',
    reactivateConfirmBody:
      'Ce match sera ré-intégré aux statistiques. Le score de performance et le LUSR des matchs ultérieurs seront recalculés (quelques secondes).',
    confirmAction: 'Confirmer',
    cancelAction: 'Annuler',
    excludeRankedDenied: 'Les matchs classés ne peuvent pas être exclus (CSR officiel)',
    excludeErrorRanked: 'Les matchs classés ne peuvent pas être exclus.',
    excludeErrorGeneric: "Impossible de mettre à jour l'exclusion. Réessaie plus tard.",
    performance: 'Performance',
    rank: 'Rang',
    rankPlacement: 'En placement',
    addFavorite: 'Ajouter aux favoris',
    removeFavorite: 'Retirer des favoris',
    mapUnknown: 'Map inconnue',
    pageErrorTitle: 'Match introuvable ou erreur de chargement.',
    pageRetry: 'Réessayer',
    pagePartialLoad: 'Ce match n\'a pas pu être chargé en totalité.',
    notSyncedTitle: 'Match pas encore synchronisé',
    notSyncedDescription:
      "Ce match n'est pas encore présent dans la base locale. S'il vient d'être joué, il apparaîtra ici après la prochaine synchronisation — reviens dans quelques minutes. Vérifie aussi que le lien du match est correct.",
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
    radarTooltipSupport: 'Assistances × 50.',
    radarTooltipScore: 'Score résiduel après frags (×100) et assistances (×50) : médailles et séries.',
    radarTooltipObjective: 'Participation aux objectifs du match — actions pondérées (captures, prises, retours…) + temps sur l\'objectif, calibrées par mode (P80 du mode = 80). Absent sur un match sans objectif.',
    radarTooltipGlossaryLink: '→ Glossaire',
    labelGrenade: 'Grenade',
    labelAssassination: 'Assassinat',
    labelGroundPound: 'Coup au sol',
    labelShoulderBash: 'Charge spartane',
    weaponUnknownPrefix: 'Arme inconnue',
    sectionMedia: 'Médias',
    mediaNoCaptures: 'Aucune capture',
    mediaNoCapturesDesc: 'Les screenshots et clips associés à ce match apparaîtront ici.',
    sectionMedals: 'Médailles',
    sectionCitations: 'Citations',
    newlyMastered: 'Maîtrisé !',
    noMedals: 'Aucune médaille',
    noCitations: 'Aucune citation',
    sectionNativeCommendations: 'Commendations',
    noNativeCommendations: 'Aucune commendation',
    combatHighlights: 'Faits marquants',
    combatKdCumulTitle: 'Frags cumulés',
    combatTugOfWarTitle: 'Dominance',
    scoreCurveTitle: 'Score dans le temps',
    scoreCurveSource:
      'Décodé du film du match : le score des deux camps, tel qu’il s’affichait en jeu.',
    scoreCurveTruncated:
      'Lecture du film incomplète — la courbe s’arrête avant la fin du match.',
    scoreCurveLead: 'Retournement',
    combatCadenceTitle: 'Cadence des frags',
    combatKillsLabel: 'Frags',
    combatDeathsLabel: 'Morts',
    combatTeamLabel: 'Mon équipe',
    combatEnemyLabel: 'Adversaires',
    combatMomentumDelta: 'Écart',
    combatMomentumCumul: 'Cumul',
    combatNemesisTitle: 'Némésis',
    combatBullyTitle: 'Souffre-douleur',
    combatNoNemesis: '—',
    combatKilledMeFmt: (n) => `T'a martyrisé ${n} fois`,
    combatIKilledFmt: (n) => `Victimisé ${n} fois`,
    combatNoData: 'Pas de données disponibles',
    combatCtfCaptureLabel: 'Capture',
    combatCtfCaptureTooltip: (player, time) => `${player} — capture à ${time}`,
    fragDiffNoData: 'Aucun événement de combat enregistré pour ce match.',
    antagonistNoData: 'Aucune donnée de duels disponible pour ce match.',
    impactBadgesNoData: 'Aucun badge d\'impact sur ce match.',
    impactBadgeNames: {
      first_blood: 'Premier sang',
      first_group_death: 'Première victime',
      clutch_finisher: 'Finisseur',
      last_casualty: 'Boulet',
      last_group_kill: 'Touriste',
      top_killer: 'Bourreau',
      silent_hero: 'Héros silencieux',
      false_brother: 'Faux-frère',
      top_gun: 'Top Gun',
      kamikaze: 'Kamikaze',
    },
    back: 'Retour',
    tabGeneral: 'Général',
    tabDetails: 'Détails',
    antagonistTitle: 'Antagonistes',
    sectionFlow: 'Déroulé du match',
    sectionDuels: 'Duels & confrontations',
    sectionEncounters: 'Historique des rencontres',
    scoreboardTitle: 'Tableau des scores',
    scoreboardNoData: 'Aucune donnée de tableau des scores disponible pour ce match.',
    teamLabelFmt: (name) => `Équipe ${name}`,
    teamUnknown: 'Équipe inconnue',
    teamNumberedFmt: (n) => `Équipe ${n}`,
    teamMine: 'Mon équipe',
    teamEnemy: 'Équipe adverse',
    sbDetailWeapons: 'Armes',
    sbDetailMedalsAndCitations: 'Médailles & citations',
    sbDetailMedalsOnly: 'Médailles',
    sbDetailExpected: 'Attendu vs réel',
    sbDetailLocallyEstimated: 'Estimé localement',
    sbDetailLocallyEstimatedHint: "Pas d'API de compétence pour ce titre : frags et morts attendus via un modèle local (volume ∝ durée du match), assistances via un modèle local.",
    sbDetailExpectedKills: 'Frags',
    sbDetailExpectedDeaths: 'Morts',
    sbDetailExpectedAssists: 'Assistances',
    sbDetailAntagonist: 'Antagoniste',
    sbDetailNemesis: 'Némésis',
    sbDetailBully: 'Souffre-douleur',
    sbDetailLocal: 'Données locales',
    sbDetailLusr: 'LUSR',
    sbDetailCsr: 'CSR',
    sbDetailBotNoteLabel: 'Coéquipier bot',
    sbDetailBotNoteValue: 'Au moins un bot dans l\'équipe — stats à relativiser.',
    sbDetailPlayerDb: 'Joueur enregistré',
    sbDetailSharedOnly: 'Joueur non enregistré',
    sbDetailExplorePlayerFmt: (player) => `Profil de ${player}`,
    sbColKda: 'FDA',
    sbColMeleeKills: 'Corps à corps',
    sbColDamageDealt: 'Dégâts infligés',
    sbColDamageTaken: 'Dégâts subis',
    sbColShotsHit: 'Tirs au but',
    sbColAccuracy: 'Précision',
    sbColCsr: 'CSR',
    sbColRank: 'Rang',
    sbColScore: 'Score',
    sbColAssists: 'Assist.',
    sbColMaxSpree: 'Folie meurt.',
    sbColHeadshots: 'Tirs à la Tête',
    sbColPerfectKills: 'Frags parfaits',
    sbColShotsFired: 'Tirs',
    sbColPowerWeapons: 'Armes lourdes',
    sbColAvgLife: 'Vie moy.',
    sbColPlayer: 'Joueur',
    sbColTopWeapon: 'Outil de destr.',
    sbColCsrTooltip: 'Classement compétitif en jeu (CSR) atteint sur ce match.',
    sbColLusrTooltip: 'Classement maison (LUSR) estimé pour ce match.',
    sbColRankTooltip: 'Place du joueur dans le match, selon le score.',
    sbColKdaTooltip: 'FDA = (Frags + Assistances/3) − Morts ; valorise l\'impact, pas frags/morts.',
    sbColAccuracyTooltip: 'Précision : part de tirs qui touchent la cible, en pourcentage.',
    sbColMaxSpreeTooltip: 'Plus longue série de frags enchaînés sans mourir.',
    sbColPerfectKillsTooltip: 'Frags parfaits : bouclier vidé puis tir à la tête sans rater.',
    sbColPowerWeaponsTooltip: 'Frags à l\'arme lourde ramassée sur la carte.',
    sbColMeleeKillsTooltip: 'Frags obtenus au corps à corps.',
    sbColAvgLifeTooltip: 'Durée de vie moyenne entre deux morts.',
    sbColTopWeaponTooltip: 'Arme ayant réalisé le plus de frags dans le match.',
    sbColOffensiveTooltip: 'Rendement offensif : frags et assistances obtenus par dégât infligé.',
    sbColDefensiveTooltip: 'Résistance : dégâts encaissés avant chaque mort.',
    sbViewHistoryFmt: (gamertag) => `Voir l'historique avec ${gamertag}`,
    sbFormatScore: (v) => new Intl.NumberFormat('fr-FR').format(v),
    ctxRecent: 'récents',
    ctxFavorites: 'favoris',
    ctxMedia: 'avec média',
    ctxTopMatches: 'top performances',
    ctxWithPlayerFmt: (gamertag) => `avec ${gamertag}`,
    ctxSessionFmt: (date) => `de la session du ${date}`,
    ctxPeriodFromToFmt: (from, to) => `de la période du ${from} au ${to}`,
    ctxPeriodFromFmt: (from) => `depuis le ${from}`,
    ctxPeriodToFmt: (to) => `jusqu'au ${to}`,
    ctxPlaylistFmt: (name) => `en ${name}`,
    ctxModeFmt: (category) => `en ${category}`,
    matchCounterCtxFmt: (label, n, total) => `Matchs ${label} ${n}/${total}`,
    objectives: {
      title: 'Objectifs',
      teamTotal: 'Total équipe',
      cols: {
        flag_captures: { label: 'Captures', tooltip: 'Captures de drapeau' },
        flag_returns: { label: 'Retours', tooltip: 'Retours de drapeau' },
        flag_steals: { label: 'Vols', tooltip: 'Vols de drapeau' },
        time_as_flag_carrier_seconds: {
          label: 'Temps porteur',
          tooltip: 'Temps en tant que porteur du drapeau',
        },
        zone_captures: { label: 'Captures', tooltip: 'Zones capturées' },
        zone_secures: { label: 'Sécurisées', tooltip: 'Zones sécurisées' },
        time_in_zones_seconds: { label: 'Temps en zone', tooltip: 'Temps passé en zone' },
        skull_grabs: { label: 'Récup.', tooltip: 'Récupérations du crâne' },
        time_as_skull_carrier_seconds: {
          label: 'Temps porteur',
          tooltip: 'Temps en tant que porteur du crâne',
        },
        longest_time_as_skull_carrier_seconds: {
          label: 'Meilleur temps',
          tooltip: 'Plus longue possession du crâne',
        },
        power_seeds_deposited: {
          label: 'Déposées',
          tooltip: "Graines d'énergie déposées dans la base",
        },
        power_seeds_stolen: {
          label: 'Volées',
          tooltip: "Graines d'énergie prises dans la base adverse",
        },
        power_seed_carriers_killed: {
          label: 'Porteurs éliminés',
          tooltip: 'Porteurs de graine adverses éliminés',
        },
        time_as_power_seed_carrier_seconds: {
          label: 'Temps porteur',
          tooltip: "Temps en tant que porteur d'une graine d'énergie",
        },
        successful_extractions: { label: 'Extractions', tooltip: 'Extractions réussies' },
        extraction_initiations_completed: {
          label: 'Amorçages',
          tooltip: 'Amorçages de balise menés à terme',
        },
        extraction_conversions_completed: {
          label: 'Conversions',
          tooltip: 'Balises adverses converties',
        },
        extraction_conversions_denied: {
          label: 'Conversions refusées',
          tooltip: 'Conversions adverses empêchées',
        },
        vip_kills: { label: 'VIP abattus', tooltip: 'VIP adverses éliminés' },
        times_selected_as_vip: {
          label: 'Fois VIP',
          tooltip: 'Nombre de fois désigné VIP',
        },
        kills_as_vip: { label: 'Frags en VIP', tooltip: 'Frags réalisés en étant le VIP' },
        time_as_vip_seconds: {
          label: 'Temps VIP',
          tooltip: 'Temps passé en tant que VIP',
        },
        longest_time_as_vip_seconds: {
          label: 'Meilleur temps',
          tooltip: 'Plus longue survie en tant que VIP',
        },
      },
    },
  },
  en: {
    prevMatch: 'Previous match',
    nextMatch: 'Next match',
    matchCounter: (n, total) => `Match ${n}/${total}`,
    copyMatchId: 'Copy match ID',
    copied: 'Copied',
    copyShort: 'Copy ID',
    copyTooltip: "Copy this match's unique identifier to clipboard",
    replayShort: '2D replay',
    replayTooltip: 'Watch the 2D replay of this match (top-down view)',
    markIrrelevant: 'Mark as irrelevant',
    reactivate: 'Reactivate',
    excludeShort: 'Exclude',
    excludeTooltip: 'Exclude this match from stats and analyses',
    reactivateTooltip: 'Re-include this match in stats and analyses',
    excludeConfirmTitle: 'Exclude this match?',
    excludeConfirmBody:
      'This match will be marked irrelevant and removed from stats. The performance score and LUSR of subsequent matches will be recomputed (a few seconds).',
    reactivateConfirmTitle: 'Reactivate this match?',
    reactivateConfirmBody:
      'This match will be re-included in stats. The performance score and LUSR of subsequent matches will be recomputed (a few seconds).',
    confirmAction: 'Confirm',
    cancelAction: 'Cancel',
    excludeRankedDenied: 'Ranked matches cannot be excluded (official CSR)',
    excludeErrorRanked: 'Ranked matches cannot be excluded.',
    excludeErrorGeneric: 'Could not update exclusion. Try again later.',
    performance: 'Performance',
    rank: 'Rank',
    rankPlacement: 'In placement',
    addFavorite: 'Add to favorites',
    removeFavorite: 'Remove from favorites',
    mapUnknown: 'Unknown map',
    pageErrorTitle: 'Match not found or load error.',
    pageRetry: 'Retry',
    pagePartialLoad: 'This match could not be fully loaded.',
    notSyncedTitle: 'Match not synced yet',
    notSyncedDescription:
      "This match isn't in the local database yet. If it was just played, it will show up here after the next sync — check back in a few minutes. Also double-check that the match link is correct.",
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
    radarTooltipObjective: 'Objective participation in this match — weighted actions (captures, grabs, returns…) plus time on the objective, calibrated per mode (mode P80 = 80). Hidden on non-objective matches.',
    radarTooltipGlossaryLink: '→ Glossary',
    labelGrenade: 'Grenade',
    labelAssassination: 'Assassination',
    labelGroundPound: 'Ground Pound',
    labelShoulderBash: 'Shoulder Bash',
    weaponUnknownPrefix: 'Unknown weapon',
    sectionMedia: 'Media',
    mediaNoCaptures: 'No captures',
    mediaNoCapturesDesc: 'Screenshots and clips associated with this match will appear here.',
    sectionMedals: 'Medals',
    sectionCitations: 'Commendations',
    newlyMastered: 'Mastered!',
    noMedals: 'No medals',
    noCitations: 'No commendations',
    sectionNativeCommendations: 'Commendations',
    noNativeCommendations: 'No commendations',
    combatHighlights: 'Highlights',
    combatKdCumulTitle: 'Cumulative frags',
    combatTugOfWarTitle: 'Dominance',
    scoreCurveTitle: 'Score over time',
    scoreCurveSource: 'Decoded from the match film: both teams’ score, as it showed in game.',
    scoreCurveTruncated:
      'Incomplete film reading — the curve stops before the end of the match.',
    scoreCurveLead: 'Lead change',
    combatCadenceTitle: 'Kill cadence',
    combatKillsLabel: 'Kills',
    combatDeathsLabel: 'Deaths',
    combatTeamLabel: 'My team',
    combatEnemyLabel: 'Opponents',
    combatMomentumDelta: 'Delta',
    combatMomentumCumul: 'Cumulative',
    combatNemesisTitle: 'Nemesis',
    combatBullyTitle: 'Bully target',
    combatNoNemesis: '—',
    combatKilledMeFmt: (n) => `Martyred you ${n} times`,
    combatIKilledFmt: (n) => `You victimized them ${n} times`,
    combatNoData: 'No data available',
    combatCtfCaptureLabel: 'Capture',
    combatCtfCaptureTooltip: (player, time) => `${player} — captured at ${time}`,
    fragDiffNoData: 'No combat events recorded for this match.',
    antagonistNoData: 'No duel data available for this match.',
    impactBadgesNoData: 'No impact badges for this match.',
    impactBadgeNames: {
      first_blood: 'First blood',
      first_group_death: 'First down',
      clutch_finisher: 'Finisher',
      last_casualty: 'Last casualty',
      last_group_kill: 'Latecomer',
      top_killer: 'Top killer',
      silent_hero: 'Silent hero',
      false_brother: 'False brother',
      top_gun: 'Top Gun',
      kamikaze: 'Kamikaze',
    },
    back: 'Back',
    tabGeneral: 'General',
    tabDetails: 'Details',
    antagonistTitle: 'Antagonists',
    sectionFlow: 'Match flow',
    sectionDuels: 'Duels & head-to-head',
    sectionEncounters: 'Encounter history',
    scoreboardTitle: 'Scoreboard',
    scoreboardNoData: 'No scoreboard data available for this match.',
    teamLabelFmt: (name) => `Team ${name}`,
    teamUnknown: 'Unknown team',
    teamNumberedFmt: (n) => `Team ${n}`,
    teamMine: 'My team',
    teamEnemy: 'Enemy team',
    sbDetailWeapons: 'Weapons',
    sbDetailMedalsAndCitations: 'Medals & commendations',
    sbDetailMedalsOnly: 'Medals',
    sbDetailExpected: 'Expected vs actual',
    sbDetailLocallyEstimated: 'Locally estimated',
    sbDetailLocallyEstimatedHint: 'No skill API for this title: expected kills and deaths from a local model (volume scales with match length), assists from a local model.',
    sbDetailExpectedKills: 'Kills',
    sbDetailExpectedDeaths: 'Deaths',
    sbDetailExpectedAssists: 'Assists',
    sbDetailAntagonist: 'Antagonist',
    sbDetailNemesis: 'Nemesis',
    sbDetailBully: 'Bully target',
    sbDetailLocal: 'Local data',
    sbDetailLusr: 'LUSR',
    sbDetailCsr: 'CSR',
    sbDetailBotNoteLabel: 'Bot teammate',
    sbDetailBotNoteValue: 'At least one bot on the team — stats to be taken with a grain of salt.',
    sbDetailPlayerDb: 'Tracked player (local DB)',
    sbDetailSharedOnly: 'Untracked player (shared DB only)',
    sbDetailExplorePlayerFmt: (player) => `Explore ${player}`,
    sbColKda: 'KDA',
    sbColMeleeKills: 'Melee',
    sbColDamageDealt: 'Damage dealt',
    sbColDamageTaken: 'Damage taken',
    sbColShotsHit: 'Shots hit',
    sbColAccuracy: 'Accuracy',
    sbColCsr: 'CSR',
    sbColRank: 'Rank',
    sbColScore: 'Score',
    sbColAssists: 'Assists',
    sbColMaxSpree: 'Killing spree',
    sbColHeadshots: 'Headshots',
    sbColPerfectKills: 'Perfect kills',
    sbColShotsFired: 'Shots',
    sbColPowerWeapons: 'Power weapons',
    sbColAvgLife: 'Avg. life',
    sbColPlayer: 'Player',
    sbColTopWeapon: 'Top weapon',
    sbColCsrTooltip: 'In-game competitive rank (CSR) reached this match.',
    sbColLusrTooltip: 'In-house rating (LUSR) estimated for this match.',
    sbColRankTooltip: 'Player\'s placement in the match, by score.',
    sbColKdaTooltip: 'KDA = (Kills + Assists/3) − Deaths; rewards impact, not kills/deaths.',
    sbColAccuracyTooltip: 'Accuracy: share of shots that hit the target, as a percentage.',
    sbColMaxSpreeTooltip: 'Longest run of kills without dying.',
    sbColPerfectKillsTooltip: 'Perfect kills: shields broken then a headshot with no missed shot.',
    sbColPowerWeaponsTooltip: 'Kills with power weapons picked up on the map.',
    sbColMeleeKillsTooltip: 'Kills scored in melee.',
    sbColAvgLifeTooltip: 'Average time alive between deaths.',
    sbColTopWeaponTooltip: 'Weapon with the most kills this match.',
    sbColOffensiveTooltip: 'Offensive yield: kills and assists per damage dealt.',
    sbColDefensiveTooltip: 'Resistance: damage absorbed before each death.',
    sbViewHistoryFmt: (gamertag) => `View history with ${gamertag}`,
    sbFormatScore: (v) => new Intl.NumberFormat('en-US').format(v),
    ctxRecent: 'recent',
    ctxFavorites: 'favorites',
    ctxMedia: 'with media',
    ctxTopMatches: 'top performances',
    ctxWithPlayerFmt: (gamertag) => `with ${gamertag}`,
    ctxSessionFmt: (date) => `from session of ${date}`,
    ctxPeriodFromToFmt: (from, to) => `from period ${from} to ${to}`,
    ctxPeriodFromFmt: (from) => `since ${from}`,
    ctxPeriodToFmt: (to) => `until ${to}`,
    ctxPlaylistFmt: (name) => `in ${name}`,
    ctxModeFmt: (category) => `in ${category}`,
    matchCounterCtxFmt: (label, n, total) => `${capitalize(label)} matches ${n}/${total}`,
    objectives: {
      title: 'Objectives',
      teamTotal: 'Team total',
      cols: {
        flag_captures: { label: 'Captures', tooltip: 'Flag captures' },
        flag_returns: { label: 'Returns', tooltip: 'Flag returns' },
        flag_steals: { label: 'Steals', tooltip: 'Flag steals' },
        time_as_flag_carrier_seconds: { label: 'Carrier time', tooltip: 'Time as flag carrier' },
        zone_captures: { label: 'Captures', tooltip: 'Zones captured' },
        zone_secures: { label: 'Secured', tooltip: 'Zones secured' },
        time_in_zones_seconds: { label: 'Zone time', tooltip: 'Time spent in zones' },
        skull_grabs: { label: 'Grabs', tooltip: 'Skull grabs' },
        time_as_skull_carrier_seconds: { label: 'Carrier time', tooltip: 'Time as skull carrier' },
        longest_time_as_skull_carrier_seconds: {
          label: 'Longest',
          tooltip: 'Longest skull possession',
        },
        power_seeds_deposited: { label: 'Deposited', tooltip: 'Power seeds deposited at the base' },
        power_seeds_stolen: {
          label: 'Stolen',
          tooltip: 'Power seeds taken from the enemy base',
        },
        power_seed_carriers_killed: {
          label: 'Carriers killed',
          tooltip: 'Enemy power seed carriers killed',
        },
        time_as_power_seed_carrier_seconds: {
          label: 'Carrier time',
          tooltip: 'Time as power seed carrier',
        },
        successful_extractions: { label: 'Extractions', tooltip: 'Successful extractions' },
        extraction_initiations_completed: {
          label: 'Initiations',
          tooltip: 'Extraction initiations completed',
        },
        extraction_conversions_completed: {
          label: 'Conversions',
          tooltip: 'Enemy beacons converted',
        },
        extraction_conversions_denied: {
          label: 'Conversions denied',
          tooltip: 'Enemy conversions denied',
        },
        vip_kills: { label: 'VIPs killed', tooltip: 'Enemy VIPs killed' },
        times_selected_as_vip: { label: 'Times VIP', tooltip: 'Times selected as VIP' },
        kills_as_vip: { label: 'Kills as VIP', tooltip: 'Kills while being the VIP' },
        time_as_vip_seconds: { label: 'VIP time', tooltip: 'Time spent as VIP' },
        longest_time_as_vip_seconds: {
          label: 'Longest',
          tooltip: 'Longest survival as VIP',
        },
      },
    },
  },
}

/** Capitalise la première lettre — utilisé par le builder EN. */
function capitalize(s: string): string {
  return s.length > 0 ? s.charAt(0).toUpperCase() + s.slice(1) : s
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
  if (spec.playlist_names?.length) parts.push(spec.playlist_names.join(', '))
  if (spec.mode_categories?.length) parts.push(spec.mode_categories.join(', '))
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

// Note Phase 2c (2026-05-07) : `buildDescriptorLabel` extrait dans
// `./descriptorLabel.ts` pour respecter la limite de 500 lignes/fichier
// (CLAUDE.md §5). `buildContextLabel` (au-dessus, fallback filterSpec)
// reste ici car il dépend uniquement de MATCH_VIEW_TEXT et n'est pas
// amené à grossir.
export { buildDescriptorLabel } from './descriptorLabel'
