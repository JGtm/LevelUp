/**
 * usageI18n.ts — LE DICTIONNAIRE DU BLOC « usages d'équipement, armes spéciales et objectifs »
 * de la page Sessions (chantier session-usage, S3).
 *
 * PARITÉ FR/EN PAR TYPAGE : `Record<Locale, UsageText>` — une clé ajoutée d'un côté
 * casse la compilation de l'autre. Il vit à part du manifeste TOML de la page Sessions :
 * ce bloc parle le vocabulaire du FILM (familles de geste, épisodes actifs), pas celui
 * des KPI de session — même partage que `match-replay/i18n.ts` face aux field-mappings.
 *
 * DEUX RÈGLES DE VOCABULAIRE, posées par la revue de lisibilité du 2026-09-09
 * (`.ai/PLAN_SESSION_USAGE_LISIBILITE_2026-09-09.md`, D1 et D6) :
 *
 *   1. UN LIBELLÉ DE GRANDEUR EST UN NOM NU — « Camouflage », pas « Camouflages
 *      (épisodes actifs) ». La colonne des libellés fait 168 px : un qualificatif y est
 *      tronqué, donc illisible, donc inutile. Ce que le qualificatif disait vit dans
 *      l'infobulle d'en-tête de la carte (`cardHint*`), qui est lue quand la question se
 *      pose et n'occupe rien le reste du temps.
 *   2. LE MOT « SOCLE » N'ATTEINT PLUS L'ÉCRAN. C'est le vocabulaire de la MESURE (un
 *      socle d'arme est l'objet du film sur lequel une arme réapparaît) ; le lecteur, lui,
 *      parle d'armes spéciales ramassées. Le code, lui, garde `pad_*` : c'est la clé du
 *      contrat, elle ne se traduit pas.
 */
import type { Locale } from '@/lib/i18n/locale'

export interface UsageText {
  /** Titres des trois cartes de section. */
  blockEquipment: string
  blockPadControl: string
  blockObjectives: string
  /** Titre de la carte unique d'état vide (bloc indisponible ou sans film). */
  blockUnavailableTitle: string
  /**
   * Aide d'en-tête de chaque carte : TOUT ce que le corps n'écrit plus (lecture des
   * barres, trait de parité, cases de régularité, réserves de mesure). Une carte a UNE
   * aide — pas une note par forme, sinon on a juste déplacé le pavé.
   */
  cardHintEquipment: string
  cardHintPadControl: string
  cardHintObjectives: string
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
  /** Intitulés des trois colonnes de jauge (§7 : les trois dénominateurs). */
  gaugeTeamOfLobby: string
  gaugePlayerOfTeam: string
  gaugePlayerOfLobby: string
  /** Repli des deux colonnes de jauge secondaires (D4). */
  sharesShowMoreFmt: (count: number) => string
  sharesHide: string
  sharesHint: string
  /** Lignes de la grille des cadences. */
  rowMyTeam: string
  rowLobby: string
  /** Segments de la piste du lobby. */
  segTeamRest: string
  segEnemy: string
  /** Bande de régularité : le comptage écrit à droite des cases. */
  bandAboveFmt: (team: string) => string
  bandTipFmt: (index: number, share: string, parity: string) => string
  bandTipUnmeasured: (index: number) => string
  /** Textes d'honnêteté (comptes bruts en INFOBULLE, jamais dans une cellule — D2). */
  honestyFmt: (numerator: string, denominator: string) => string
  cadenceTipFmt: (who: string, metric: string, rate: string, raw: string) => string
  gaugeTipFmt: (metric: string, gauge: string, share: string, raw: string) => string
  trackTipFmt: (who: string, count: string, pct: string) => string
  notMeasured: string
  /** Notes de pied du bloc armes spéciales — les seuls CHIFFRES qui restent au pied. */
  padUnnamedFmt: (count: number) => string
  powerupLine: string
  powerupDetailFmt: (label: string, count: number) => string
  padCadenceFmt: (player: string, team: string, lobby: string) => string
  /** Libellés des grandeurs (clés du contrat). */
  metricCamo: string
  metricOvershield: string
  metricWall: string
  metricGrapple: string
  metricDropped: string
  /** Le TOTAL des ramassages d'armes spéciales — mis à part sous ses familles (D6). */
  metricPads: string
  /** Familles d'équipement déployé (mêmes clés que le manifeste du titre). */
  equipSensor: string
  equipRift: string
  equipShroud: string
  equipSeeker: string
  equipField: string
  /**
   * Le translocateur quantique, famille du BILAN D'ÉQUIPEMENT (`equipment_translocator_beacon`,
   * étape E4). PAS `equipRift` : cette clé habille `deployed_<famille>`, dont le suffixe
   * réel ne vaut jamais « rift » (le manifeste écrit `translocator_beacon` — cf. §6 du plan,
   * `deployedFamilyLabel` ne matche donc jamais ce cas, une clé morte antérieure à ce lot,
   * non traitée ici).
   */
  equipTranslocator: string
  /** Famille déployée hors catalogue : la clé reste à l'écran — un nom approchant se
   *  lirait comme une certitude (même règle que le catalogue d'armes du rejeu). */
  metricDeployedFmt: (family: string) => string
  /**
   * LES TROIS ISSUES D'UN OBJET PRIS (P1, étape E4) : le remplissage de la jauge, en
   * infobulle uniquement — jamais de chiffre affiché dans la barre (P7/E4.2).
   */
  outcomeUsed: string
  outcomeKept: string
  outcomeDropped: string
  /** L'infobulle de jauge, augmentée du détail des trois issues quand la grandeur les porte. */
  gaugeOutcomeTipFmt: (base: string, used: string, kept: string, dropped: string) => string
  /** L'infobulle de jauge, augmentée des deux repères de taux qui EXCLUENT le joueur (P7). */
  gaugeReferenceTipFmt: (base: string, teammates: string, opponents: string) => string
  /** Famille d'arme non nommée par le catalogue du titre : la clé reste à l'écran. */
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
}

export const USAGE_TEXT: Record<Locale, UsageText> = {
  fr: {
    blockEquipment: "Usages d'équipement",
    blockPadControl: 'Contrôle des armes spéciales',
    blockObjectives: 'Objectifs par rôle et par famille',
    blockUnavailableTitle: "Usages d'équipement, armes spéciales et objectifs",
    cardHintEquipment:
      "Chaque barre est ta part du total de ton équipe sur la session ; le trait vertical marque la parité, la part d'un joueur moyen (100 divisé par l'effectif). Plus bas, une case par match mesuré, dans l'ordre de la session : au-dessus, à hauteur ou en dessous de cette parité. Camouflage et surbouclier comptés ici sont les épisodes ACTIFS, mesurés par le film ; les bonus ramassés au sol, eux, restent anonymes et sont comptés dans « Contrôle des armes spéciales » — les deux ne s'additionnent jamais.",
    cardHintPadControl:
      "Chaque barre est ta part du total de ton équipe sur la session ; le trait vertical marque la parité, la part d'un joueur moyen (100 divisé par l'effectif). Chaque ramassage vient de l'événement daté du film et porte son ramasseur : un ramassage que la mesure ne sait pas attribuer n'est compté pour personne, jamais deviné. Les bonus (camouflage, surbouclier) ne sont attribuables à personne par nature : ils sont comptés à part, et jamais additionnés aux épisodes actifs du bloc « Usages d'équipement ».",
    cardHintObjectives:
      "Chaque barre est ta part du total de ton équipe sur la session ; le trait vertical marque la parité, la part d'un joueur moyen (100 divisé par l'effectif). Ce bloc se mesure hors film : il couvre plus de matchs que les deux autres. Le rôle « Tenir » se mesure en durée — ses totaux sont en minutes:secondes, ses parts restent des pourcentages.",
    measuredFmt: (m, t) => `Matchs mesurés ${m}/${t}`,
    objectivesScopeFmt: (n, t) => `Matchs avec objectifs ${n}/${t}`,
    unavailableLoadFailed: "La lecture du résumé d'usage a échoué.",
    unavailableNoMeasured: "Aucun match de cette session n'a de film mesuré.",
    viewCadences: 'Cadences par 10 minutes de jeu mesuré',
    viewShares: 'Parts et parités',
    viewRegularity: 'Régularité match par match',
    viewLobbyTrack: 'Qui ramasse les armes spéciales',
    viewRoles: 'Par rôle, toutes familles confondues',
    viewFamilies: "Ma part d'équipe, par famille de mode",
    viewSquadRoles: "Part d'équipe par joueur et par rôle",
    gaugeTeamOfLobby: 'Mon équipe dans le lobby',
    gaugePlayerOfTeam: 'Ma part dans mon équipe',
    gaugePlayerOfLobby: 'Ma part dans le lobby',
    sharesShowMoreFmt: (n) => `Voir plus (${n})`,
    sharesHide: 'Replier',
    sharesHint:
      "Les deux autres façons de rapporter la même mesure : ma part du lobby entier, et la part de mon équipe dans ce lobby. Rien n'est retiré du calcul.",
    rowMyTeam: 'Mon équipe',
    rowLobby: 'Lobby',
    segTeamRest: 'Reste de mon équipe',
    segEnemy: 'Eux (anonyme)',
    bandAboveFmt: (team) => `${team} au-dessus de la parité`,
    bandTipFmt: (i, share, parity) =>
      `Match ${i} — part d'équipe ${share} (parité de session ${parity})`,
    bandTipUnmeasured: (i) => `Match ${i} — part non mesurée`,
    honestyFmt: (n, d) => `${n} sur ${d}`,
    cadenceTipFmt: (who, metric, rate, raw) => `${who} — ${metric} : ${rate} par 10 min (${raw})`,
    gaugeTipFmt: (metric, gauge, share, raw) => `${metric} — ${gauge} : ${share} (${raw})`,
    trackTipFmt: (who, count, pct) => `${who} : ${count} ramassages (${pct})`,
    notMeasured: '—',
    padUnnamedFmt: (n) => `${n} ramassages sans joueur identifié — jamais attribués.`,
    powerupLine: 'Bonus ramassés (joueur non identifié)',
    powerupDetailFmt: (label, count) => `${label} ${count}`,
    padCadenceFmt: (p, t, l) => `Ramassages par 10 min — moi ${p} · mon équipe ${t} · lobby ${l}`,
    metricCamo: 'Camouflage',
    metricOvershield: 'Surbouclier',
    metricWall: 'Mur de protection',
    metricGrapple: 'Grappin',
    metricDropped: 'Objets lâchés',
    metricPads: 'Toutes armes spéciales',
    equipSensor: 'Capteur de menaces',
    equipRift: 'Faille quantique',
    equipShroud: 'Écran occultant',
    equipSeeker: 'Traqueur de menaces',
    equipField: 'Champ de réparation',
    equipTranslocator: 'Translocateur',
    metricDeployedFmt: (fam) => `Équipement ${fam}`,
    outcomeUsed: 'Utilisé',
    outcomeKept: "Gardé sans l'utiliser",
    outcomeDropped: 'Lâché en mourant',
    gaugeOutcomeTipFmt: (base, used, kept, dropped) =>
      `${base} — utilisé ${used} · gardé ${kept} · lâché ${dropped}`,
    gaugeReferenceTipFmt: (base, teammates, opponents) =>
      `${base} (reste de mon équipe ${teammates} utilisé · eux ${opponents} utilisé)`,
    padFamilyFmt: (key) => `Arme ${key}`,
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
  },
  en: {
    blockEquipment: 'Equipment usage',
    blockPadControl: 'Power weapon control',
    blockObjectives: 'Objectives by role and family',
    blockUnavailableTitle: 'Equipment, power weapons and objectives',
    cardHintEquipment:
      'Each bar is your share of your team total over the session; the vertical mark is parity, the share of an average player (100 divided by headcount). Below, one square per measured match, in session order: above, at, or below that parity. Camouflage and overshield counted here are the ACTIVE episodes measured by the film; power-ups picked up off the ground stay anonymous and are counted in "Power weapon control" — the two are never added together.',
    cardHintPadControl:
      'Each bar is your share of your team total over the session; the vertical mark is parity, the share of an average player (100 divided by headcount). Every pickup comes from the timed film event and carries its picker: a pickup the measurement cannot attribute is counted for nobody, never guessed. Power-ups (camouflage, overshield) are attributable to nobody by nature: they are counted separately, and never added to the active episodes of the "Equipment usage" block.',
    cardHintObjectives:
      'Each bar is your share of your team total over the session; the vertical mark is parity, the share of an average player (100 divided by headcount). This block is measured outside the film: it covers more matches than the other two. The "Hold" role is measured in duration — its totals are minutes:seconds, its shares remain percentages.',
    measuredFmt: (m, t) => `Measured matches ${m}/${t}`,
    objectivesScopeFmt: (n, t) => `Matches with objectives ${n}/${t}`,
    unavailableLoadFailed: 'Loading the usage summary failed.',
    unavailableNoMeasured: 'No match of this session has a measured film.',
    viewCadences: 'Rates per 10 minutes of measured play',
    viewShares: 'Shares and parity',
    viewRegularity: 'Match-by-match consistency',
    viewLobbyTrack: 'Who picks up the power weapons',
    viewRoles: 'By role, all families combined',
    viewFamilies: 'My team share, by mode family',
    viewSquadRoles: 'Team share by player and role',
    gaugeTeamOfLobby: 'My team in the lobby',
    gaugePlayerOfTeam: 'My share of my team',
    gaugePlayerOfLobby: 'My share of the lobby',
    sharesShowMoreFmt: (n) => `Show more (${n})`,
    sharesHide: 'Collapse',
    sharesHint:
      "The two other ways of reporting the same measurement: my share of the whole lobby, and my team's share of that lobby. Nothing is removed from the calculation.",
    rowMyTeam: 'My team',
    rowLobby: 'Lobby',
    segTeamRest: 'Rest of my team',
    segEnemy: 'Them (anonymous)',
    bandAboveFmt: (team) => `${team} above parity`,
    bandTipFmt: (i, share, parity) =>
      `Match ${i} — team share ${share} (session parity ${parity})`,
    bandTipUnmeasured: (i) => `Match ${i} — share not measured`,
    honestyFmt: (n, d) => `${n} of ${d}`,
    cadenceTipFmt: (who, metric, rate, raw) => `${who} — ${metric}: ${rate} per 10 min (${raw})`,
    gaugeTipFmt: (metric, gauge, share, raw) => `${metric} — ${gauge}: ${share} (${raw})`,
    trackTipFmt: (who, count, pct) => `${who}: ${count} pickups (${pct})`,
    notMeasured: '—',
    padUnnamedFmt: (n) => `${n} pickups with no identified player — never attributed.`,
    powerupLine: 'Power-ups picked up (player not identified)',
    powerupDetailFmt: (label, count) => `${label} ${count}`,
    padCadenceFmt: (p, t, l) => `Pickups per 10 min — me ${p} · my team ${t} · lobby ${l}`,
    metricCamo: 'Camouflage',
    metricOvershield: 'Overshield',
    metricWall: 'Drop wall',
    metricGrapple: 'Grappleshot',
    metricDropped: 'Dropped objects',
    metricPads: 'All power weapons',
    equipSensor: 'Threat sensor',
    equipRift: 'Quantum rift',
    equipShroud: 'Shroud screen',
    equipSeeker: 'Threat seeker',
    equipField: 'Repair field',
    equipTranslocator: 'Translocator',
    metricDeployedFmt: (fam) => `Equipment ${fam}`,
    outcomeUsed: 'Used',
    outcomeKept: 'Kept, not used',
    outcomeDropped: 'Dropped when killed',
    gaugeOutcomeTipFmt: (base, used, kept, dropped) =>
      `${base} — used ${used} · kept ${kept} · dropped ${dropped}`,
    gaugeReferenceTipFmt: (base, teammates, opponents) =>
      `${base} (rest of my team ${teammates} used · them ${opponents} used)`,
    padFamilyFmt: (key) => `Weapon ${key}`,
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
  },
}

// ─── Libellés dérivés du dictionnaire (clés machine du contrat → texte) ──────────

/**
 * Le libellé d'une famille d'ÉQUIPEMENT DÉPLOYÉ (clé du manifeste du titre, suffixe de
 * `deployed_<famille>`).
 *
 * 2e copie ASSUMÉE du dictionnaire `placementFamily` de `match-replay/i18n.ts` (import
 * croisé interdit par le ratchet lint-cross-feature-imports), au même titre que
 * `USAGE_METRIC_TOKENS` l'est de `USAGE_GROUP_TOKENS` : mêmes noms d'un écran à l'autre.
 * À la 3e copie : centraliser dans `components/` et poser le garde-rail (CLAUDE.md n°6).
 *
 * `wall` n'est PAS ici : la grandeur `deployed_wall` a son propre libellé de premier rang
 * (`metricWall`), parce que le mur est la seule famille que le contrat range dans une
 * grandeur nommée plutôt que dans l'ensemble ouvert.
 */
export function deployedFamilyLabel(family: string, t: UsageText): string {
  switch (family) {
    case 'sensor':
      return t.equipSensor
    case 'rift':
      return t.equipRift
    case 'shroud':
      return t.equipShroud
    case 'seeker':
      return t.equipSeeker
    case 'field':
      return t.equipField
    default:
      return t.metricDeployedFmt(family)
  }
}

/**
 * equipmentBilanFamilyLabel — le libellé d'une famille du BILAN D'ÉQUIPEMENT
 * (`equipment_<famille>`, étape E4), la clé étant celle du RÉSUMÉ Go
 * (`replay.EquipmentOutcomeFamilies`, vocabulaire des POSES : `translocator_beacon`,
 * `shroud_screen`, `threat_seeker`, `repair_field`, PAS les alias courts de rendu
 * `deployedFamilyLabel` ('rift'/'shroud'/'seeker'/'field') qui ne matchent AUCUNE
 * clé réelle de `deployed_<famille>` (§6 du plan — bug préexistant, non traité ici).
 *
 * Familles reconnues : les huit de `equipmentOutcomeStems` (Go). Une famille NEUVE
 * du bilan (manifeste étendu) garde sa clé à l'écran — jamais un nom approchant.
 */
export function equipmentFamilyLabel(family: string, t: UsageText): string {
  switch (family) {
    case 'wall':
      return t.metricWall
    case 'sensor':
      return t.equipSensor
    case 'translocator_beacon':
      return t.equipTranslocator
    case 'shroud_screen':
      return t.equipShroud
    case 'threat_seeker':
      return t.equipSeeker
    case 'repair_field':
      return t.equipField
    case 'powerup_camo':
      return t.metricCamo
    case 'powerup_overshield':
      return t.metricOvershield
    default:
      return t.metricDeployedFmt(family)
  }
}

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
