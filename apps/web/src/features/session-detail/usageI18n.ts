/**
 * usageI18n.ts — LE DICTIONNAIRE DU BLOC « usages d'équipement, socles et objectifs »
 * de la page Sessions (chantier session-usage, S3).
 *
 * PARITÉ FR/EN PAR TYPAGE : `Record<Locale, UsageText>` — une clé ajoutée d'un côté
 * casse la compilation de l'autre. FR sans anglicismes (« prises de socle »,
 * « tractions de grappin », « objets lâchés au sol » — vocabulaire imposé par le
 * handoff §1). Il vit à part du manifeste TOML de la page Sessions : ce bloc parle le
 * vocabulaire du FILM (familles de geste, socles, épisodes actifs), pas celui des KPI
 * de session — même partage que `match-replay/i18n.ts` face aux field-mappings.
 */
import type { Locale } from '@/lib/i18n/locale'

export interface UsageText {
  /** Titres des trois cartes de section. */
  blockEquipment: string
  blockPadControl: string
  blockObjectives: string
  /** Titre de la carte unique d'état vide (bloc indisponible ou sans film). */
  blockUnavailableTitle: string
  /** « Matchs mesurés N/M » — TOUJOURS visible (couverture des films partielle). */
  measuredFmt: (measured: number, total: number) => string
  /** « Matchs avec objectifs N/M » — le bloc 3 a son propre scope (hors films). */
  objectivesScopeFmt: (withObjectives: number, total: number) => string
  /** Raisons du bloc indisponible (contrat : unavailable_reason machine).
   *  `unsupported` N'A PAS DE LIBELLE, et c'est voulu : depuis le 2026-09-05 ce cas
   *  MASQUE le bloc au lieu de l'annoncer (un titre sans decodeur de film n'aura jamais
   *  de resume d'usage — la carte etait un bloc mort). Cf. usageLogic.usageAvailability. */
  unavailableLoadFailed: string
  unavailableNoMeasured: string
  /** Titres des vues à l'intérieur des cartes. */
  viewCadences: string
  viewShares: string
  viewRegularity: string
  viewLobbyTrack: string
  viewRoles: string
  viewFamilies: string
  viewSquadRoles: string
  /** Intitulés d'axes et de colonnes de jauge (§7 : les trois colonnes). */
  gaugeTeamOfLobby: string
  gaugePlayerOfTeam: string
  gaugePlayerOfLobby: string
  axisSharePct: string
  /** Lignes de la grille des cadences. */
  rowMyTeam: string
  rowLobby: string
  /** Segments de la piste du lobby. */
  segTeamRest: string
  segEnemy: string
  /** Légende du trait de parité (jeton distinct, jamais une teinte de donnée). */
  parityLegend: string
  /** Bande de régularité. */
  bandCaption: string
  bandAboveFmt: (team: string, lobby: string) => string
  bandTipFmt: (index: number, share: string, parity: string) => string
  bandTipUnmeasured: (index: number) => string
  /** Textes d'honnêteté (comptes bruts À CÔTÉ d'un taux, jamais un axe). */
  honestyFmt: (numerator: string, denominator: string) => string
  cadenceTipFmt: (who: string, metric: string, rate: string, raw: string) => string
  gaugeTipFmt: (metric: string, gauge: string, share: string, raw: string) => string
  trackTipFmt: (who: string, count: string, pct: string) => string
  notMeasured: string
  /** Corollaire §4 du handoff — À L'ÉCRAN, dans les deux blocs concernés. */
  corollaryEquipment: string
  corollaryPadControl: string
  /** Notes de pied du bloc socles. */
  padUnnamedFmt: (count: number) => string
  powerupLine: string
  powerupDetailFmt: (label: string, count: number, rate: string | null) => string
  padCadenceFmt: (player: string, team: string, lobby: string) => string
  /** Libellés des grandeurs (clés du contrat). */
  metricCamo: string
  metricOvershield: string
  metricWall: string
  metricGrapple: string
  metricDropped: string
  metricPads: string
  metricDeployedFmt: (family: string) => string
  /** Familles de socle d'arme : la clé publiée est un identifiant de famille du film. */
  padFamilyFmt: (key: string) => string
  powerupCamo: string
  powerupOvershield: string
  powerupOtherFmt: (key: string) => string
  /** Rôles d'objectif (vocabulaire imposé : prendre / défendre / tenir). */
  roleTake: string
  roleDefend: string
  roleHold: string
  roleUnknownFmt: (key: string) => string
  /** Familles de mode (clés narrative). */
  familyCtf: string
  familyKoth: string
  familyStrongholds: string
  familyOddball: string
  familyStockpile: string
  familyExtraction: string
  familyVip: string
  familyUnknownFmt: (key: string) => string
  /** Divers. */
  durationNote: string
}

export const USAGE_TEXT: Record<Locale, UsageText> = {
  fr: {
    blockEquipment: "Usages d'équipement",
    blockPadControl: 'Contrôle des armes spéciales',
    blockObjectives: 'Objectifs par rôle et par famille',
    blockUnavailableTitle: "Usages d'équipement, socles et objectifs",
    measuredFmt: (m, t) => `Matchs mesurés ${m}/${t}`,
    objectivesScopeFmt: (n, t) => `Matchs avec objectifs ${n}/${t}`,
    unavailableLoadFailed: "La lecture du résumé d'usage a échoué.",
    unavailableNoMeasured: "Aucun match de cette session n'a de film mesuré.",
    viewCadences: 'Cadences par 10 minutes de jeu mesuré',
    viewShares: 'Parts et parités',
    viewRegularity: 'Régularité match par match',
    viewLobbyTrack: 'Piste du lobby — prises de socle nommées',
    viewRoles: 'Par rôle, toutes familles confondues',
    viewFamilies: "Ma part d'équipe, par famille de mode",
    viewSquadRoles: "Part d'équipe par joueur et par rôle",
    gaugeTeamOfLobby: 'Mon camp / lobby',
    gaugePlayerOfTeam: 'Joueur / son équipe',
    gaugePlayerOfLobby: 'Joueur / lobby',
    axisSharePct: 'Part (%)',
    rowMyTeam: 'Mon équipe',
    rowLobby: 'Lobby',
    segTeamRest: 'Reste de mon équipe',
    segEnemy: 'Eux (anonyme)',
    parityLegend:
      "Trait vertical : parité — la part d'un contributeur moyen (100 / effectif).",
    bandCaption: 'Une case par match mesuré, dans l’ordre de la session.',
    bandAboveFmt: (team, lobby) =>
      `Matchs au-dessus de la parité : ${team} (équipe) · ${lobby} (lobby)`,
    bandTipFmt: (i, share, parity) =>
      `Match ${i} — part d'équipe ${share} (parité de session ${parity})`,
    bandTipUnmeasured: (i) => `Match ${i} — part non mesurée`,
    honestyFmt: (n, d) => `${n} sur ${d}`,
    cadenceTipFmt: (who, metric, rate, raw) => `${who} — ${metric} : ${rate} par 10 min (${raw})`,
    gaugeTipFmt: (metric, gauge, share, raw) => `${metric} — ${gauge} : ${share} (${raw})`,
    trackTipFmt: (who, count, pct) => `${who} : ${count} prises (${pct})`,
    notMeasured: '—',
    corollaryEquipment:
      "Camouflage et surbouclier existent sous deux formes : l'épisode actif, mesuré par le film (compté ici), et le socle de bonus vidé, anonyme (compté au bloc « Contrôle des armes spéciales »). Deux objets distincts, jamais additionnés.",
    corollaryPadControl:
      "Les socles de bonus (camouflage, surbouclier) sont anonymes par mesure : un bonus s'identifie par un nom, jamais rattachable à un joueur. Ils ne sont jamais additionnés aux épisodes actifs du bloc « Usages d'équipement ».",
    padUnnamedFmt: (n) => `${n} prises de socle sans ramasseur nommé — jamais réparties.`,
    powerupLine: 'Socles de bonus vidés (anonymes)',
    powerupDetailFmt: (label, count, rate) =>
      rate == null ? `${label} ${count}` : `${label} ${count} (${rate} par 10 min)`,
    padCadenceFmt: (p, t, l) => `Cadence par 10 min — joueur ${p} · équipe ${t} · lobby ${l}`,
    metricCamo: 'Camouflages (épisodes actifs)',
    metricOvershield: 'Surboucliers (épisodes actifs)',
    metricWall: 'Murs de protection déployés',
    metricGrapple: 'Tractions de grappin',
    metricDropped: 'Objets lâchés au sol',
    metricPads: 'Prises de socle',
    metricDeployedFmt: (fam) => `Équipements déployés (${fam})`,
    padFamilyFmt: (key) => `Famille d'arme ${key}`,
    powerupCamo: 'Camouflage',
    powerupOvershield: 'Surbouclier',
    powerupOtherFmt: (key) => `Bonus ${key}`,
    roleTake: 'Prendre',
    roleDefend: 'Défendre',
    roleHold: 'Tenir',
    roleUnknownFmt: (key) => `Rôle ${key}`,
    familyCtf: 'Drapeau',
    familyKoth: 'Colline du roi',
    familyStrongholds: 'Bastions',
    familyOddball: 'Crâne',
    familyStockpile: 'Stockage',
    familyExtraction: 'Extraction',
    familyVip: 'VIP',
    familyUnknownFmt: (key) => `Famille ${key}`,
    durationNote: 'Le rôle « Tenir » se mesure en durée : ses totaux sont en minutes:secondes, ses parts restent des pourcentages.',
  },
  en: {
    blockEquipment: 'Equipment usage',
    blockPadControl: 'Power weapon control',
    blockObjectives: 'Objectives by role and family',
    blockUnavailableTitle: 'Equipment, pads and objectives',
    measuredFmt: (m, t) => `Measured matches ${m}/${t}`,
    objectivesScopeFmt: (n, t) => `Matches with objectives ${n}/${t}`,
    unavailableLoadFailed: 'Loading the usage summary failed.',
    unavailableNoMeasured: 'No match of this session has a measured film.',
    viewCadences: 'Rates per 10 minutes of measured play',
    viewShares: 'Shares and parity',
    viewRegularity: 'Match-by-match consistency',
    viewLobbyTrack: 'Lobby track — named pad pickups',
    viewRoles: 'By role, all families combined',
    viewFamilies: 'My team share, by mode family',
    viewSquadRoles: 'Team share by player and role',
    gaugeTeamOfLobby: 'My team / lobby',
    gaugePlayerOfTeam: 'Player / their team',
    gaugePlayerOfLobby: 'Player / lobby',
    axisSharePct: 'Share (%)',
    rowMyTeam: 'My team',
    rowLobby: 'Lobby',
    segTeamRest: 'Rest of my team',
    segEnemy: 'Them (anonymous)',
    parityLegend: 'Vertical mark: parity — the share of an average contributor (100 / headcount).',
    bandCaption: 'One square per measured match, in session order.',
    bandAboveFmt: (team, lobby) => `Matches above parity: ${team} (team) · ${lobby} (lobby)`,
    bandTipFmt: (i, share, parity) =>
      `Match ${i} — team share ${share} (session parity ${parity})`,
    bandTipUnmeasured: (i) => `Match ${i} — share not measured`,
    honestyFmt: (n, d) => `${n} of ${d}`,
    cadenceTipFmt: (who, metric, rate, raw) => `${who} — ${metric}: ${rate} per 10 min (${raw})`,
    gaugeTipFmt: (metric, gauge, share, raw) => `${metric} — ${gauge}: ${share} (${raw})`,
    trackTipFmt: (who, count, pct) => `${who}: ${count} pickups (${pct})`,
    notMeasured: '—',
    corollaryEquipment:
      'Camouflage and overshield exist in two forms: the active episode, measured by the film (counted here), and the emptied power-up pad, anonymous (counted in the "Power weapon control" block). Two distinct objects, never added together.',
    corollaryPadControl:
      'Power-up pads (camouflage, overshield) are anonymous by measurement: a power-up is identified by a name, never attachable to a player. They are never added to the active episodes of the "Equipment usage" block.',
    padUnnamedFmt: (n) => `${n} pad pickups without a named collector — never attributed.`,
    powerupLine: 'Emptied power-up pads (anonymous)',
    powerupDetailFmt: (label, count, rate) =>
      rate == null ? `${label} ${count}` : `${label} ${count} (${rate} per 10 min)`,
    padCadenceFmt: (p, t, l) => `Rate per 10 min — player ${p} · team ${t} · lobby ${l}`,
    metricCamo: 'Camouflages (active episodes)',
    metricOvershield: 'Overshields (active episodes)',
    metricWall: 'Drop walls deployed',
    metricGrapple: 'Grappleshot pulls',
    metricDropped: 'Objects dropped on the ground',
    metricPads: 'Pad pickups',
    metricDeployedFmt: (fam) => `Equipment deployed (${fam})`,
    padFamilyFmt: (key) => `Weapon family ${key}`,
    powerupCamo: 'Camouflage',
    powerupOvershield: 'Overshield',
    powerupOtherFmt: (key) => `Power-up ${key}`,
    roleTake: 'Take',
    roleDefend: 'Defend',
    roleHold: 'Hold',
    roleUnknownFmt: (key) => `Role ${key}`,
    familyCtf: 'Capture the Flag',
    familyKoth: 'King of the Hill',
    familyStrongholds: 'Strongholds',
    familyOddball: 'Oddball',
    familyStockpile: 'Stockpile',
    familyExtraction: 'Extraction',
    familyVip: 'VIP',
    familyUnknownFmt: (key) => `Family ${key}`,
    durationNote:
      'The "Hold" role is measured in duration: its totals are minutes:seconds, its shares remain percentages.',
  },
}

// ─── Libellés dérivés du dictionnaire (clés machine du contrat → texte) ──────────

/** Le libellé d'un rôle d'objectif (« prendre / défendre / tenir »). */
export function roleLabel(key: string, t: UsageText): string {
  switch (key) {
    case 'take':
      return t.roleTake
    case 'defend':
      return t.roleDefend
    case 'hold':
      return t.roleHold
    default:
      return t.roleUnknownFmt(key)
  }
}

/** Le libellé d'une famille de mode (clés narrative du contrat). */
export function familyLabel(key: string, t: UsageText): string {
  switch (key) {
    case 'ctf':
      return t.familyCtf
    case 'zones_koth':
      return t.familyKoth
    case 'zones_strongholds':
      return t.familyStrongholds
    case 'oddball':
      return t.familyOddball
    case 'stockpile':
      return t.familyStockpile
    case 'extraction':
      return t.familyExtraction
    case 'vip':
      return t.familyVip
    default:
      return t.familyUnknownFmt(key)
  }
}

/** Le libellé d'un socle de bonus (nom canonique `powerup_*`). */
export function powerupLabel(key: string, t: UsageText): string {
  switch (key) {
    case 'powerup_camo':
      return t.powerupCamo
    case 'powerup_overshield':
      return t.powerupOvershield
    default:
      return t.powerupOtherFmt(key)
  }
}
