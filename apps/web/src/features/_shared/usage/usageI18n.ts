/**
 * usageI18n.ts — LE DICTIONNAIRE DU BLOC « usages d'équipement, armes spéciales et objectifs »
 * (chantier session-usage, S3).
 *
 * Déménagé de `session-detail/usageI18n.ts` vers ici le 2026-09-09 (étape E5.1,
 * PLAN_EQUIPEMENT_GACHIS_2026-09-09.md) : le bloc devient importable par `features/synthesis`
 * et `features/squad` sans violer `lint-cross-feature-imports` — déplacement pur, contenu
 * inchangé.
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
  /**
   * LES NIVEAUX D'ARME (2026-09-14) : le titre de la rangée, les cinq libellés de niveau et
   * les trois notes de mesure. Le niveau vient de la CARTE (emplacement Forge croisé au socle)
   * et de l'équipement de départ du film — jamais du nom ni du rôle de l'arme.
   */
  blockPadTiers: string
  cardHintPadTiers: string
  /**
   * LES TROIS NIVEAUX RENDUS (D2 amendee, 2026-09-21). `bonus` et `non_classe` ont quitte
   * cette table avec leurs lignes : les socles de BONUS sont des equipements (camouflage et
   * surbouclier sont deja comptes en `equipment_powerup_*` dans « Usages d'equipement »), et
   * les prises sans emplacement identifie passent en infobulle (`padTierUnclassifiedFmt`).
   */
  padTierLabels: Record<'base' | 'terrain' | 'puissance', string>
  /** Matchs mesurés dont le film n'a publié AUCUN socle (le mode n'en allume aucun). */
  padTierNoPadsFmt: (n: number) => string
  /** Matchs à socles dont la carte n'est pas dans la référence : niveaux non établis. */
  padTierUnmeasuredFmt: (n: number) => string
  /** Matchs à départs aléatoires : pas d'arme de base à distinguer. */
  padTierRandomStartsFmt: (n: number) => string
  /** Le dépliable des armes de base, fermé par défaut (D2) : « Armes de base (N) ». */
  padTierBaseToggleFmt: (n: number) => string
  /** Les prises dont l'emplacement n'a pas été identifié — en INFOBULLE, plus en ligne (D2). */
  padTierUnclassifiedFmt: (n: number) => string
  blockObjectives: string
  /** Titre de la carte unique d'état vide (bloc indisponible ou sans film). */
  blockUnavailableTitle: string
  /**
   * Aide d'en-tête de chaque carte : TOUT ce que le corps n'écrit plus (lecture des
   * barres, trait de parité, cases de régularité, réserves de mesure). UNE aide par
   * carte — pas une note par forme sous le graphe, sinon on a juste déplacé le pavé.
   */
  cardHintPadControl: string
  cardHintObjectives: string
  /** Les trois blocs d'équipement (2026-09-13) : chacun porte SON aide — phrases de
   *  l'ancienne aide unique, redistribuées vers le bloc qu'elles décrivent. */
  cardHintCadences: string
  cardHintShares: string
  cardHintRegularity: string
  /** La même couverture, en PIED de carte et en phrase (2026-09-13, demande
   *  utilisateur) : le bandeau des quatre cartes d'équipement ne porte plus de compteur,
   *  la couverture s'écrit UNE fois par rangée sous la carte de gauche.
   *  PHRASE = ACCORD EN NOMBRE : « Mesuré sur 1 match sur 1 » au singulier (constaté
   *  écrit « 1 matchs » sur une session d'un seul match, capture du 2026-09-13). */
  measuredFooterFmt: (measured: number, total: number) => string
  /** « Matchs avec objectifs N/M » — le bloc 3 a son propre scope (hors films). */
  objectivesScopeFmt: (withObjectives: number, total: number) => string
  /** Raisons du bloc indisponible (contrat : unavailable_reason machine).
   *  `unsupported` N'A PAS DE LIBELLE, et c'est voulu : depuis le 2026-09-05 ce cas
   *  MASQUE le bloc au lieu de l'annoncer (un titre sans decodeur de film n'aura jamais
   *  de resume d'usage — la carte etait un bloc mort). Cf. usageAvailability. */
  unavailableLoadFailed: string
  /**
   * LES ÉTATS VIDES D'UN BLOC DANS UNE RANGÉE (D8, 2026-09-21) : le bloc RESTE affiché et
   * NOMME sa cause — une rangée amputée d'une carte se lit comme un bug. `unavailableLoadFailed`
   * et `emptyNoFilm` couvrent les deux causes déjà connues ; les deux suivantes distinguent
   * « le film est lu, mais ce mode n'allume rien » (Super Fiesta) de « rien n'est mesuré ».
   */
  emptyNoFilm: string
  emptyNoPads: string
  emptyNoObjectives: string
  /**
   * LE TITRE COURT DE CHAQUE ÉTAT VIDE (2026-09-22). L'état vide canonique de l'app
   * (`EmptyStateNotice`, `components/ui/empty-state.tsx`) se lit sur DEUX lignes : un titre
   * en gras (`text-sm font-semibold text-foreground`) puis sa description en gris. Les
   * phrases ci-dessus sont les DESCRIPTIONS ; ces quatre-là sont les titres. Une par cause,
   * toujours : deux causes, deux phrases (D8) vaut aussi pour la ligne du haut, sans quoi
   * le titre redirait « aucune donnée » à la place de ce qui manque vraiment.
   */
  emptyTitleNoFilm: string
  emptyTitleNoPads: string
  emptyTitleNoObjectives: string
  emptyTitleLoadFailed: string
  /** Titres des vues à l'intérieur des cartes. */
  viewCadences: string
  viewShares: string
  viewRegularity: string
  viewLobbyTrack: string
  viewRoles: string
  viewFamilies: string
  viewSquadRoles: string
  /**
   * Les prises NETTES de drapeau : le TITRE DE LA VUE, et lui seul.
   *
   * SES SIX PHRASES EXPLICATIVES SONT PARTIES le 2026-09-21 (D1) : l'écart brut/net, la
   * fenêtre de jonglage, les ouvertures lues, la couverture et le périmètre d'équipe
   * faisaient cinq paragraphes de méthode sous deux chiffres. La jauge du rôle « prendre »
   * dit la grandeur ; le reste n'est plus écrit.
   */
  viewFlagGrabsNet: string
  /** Intitulés des trois colonnes de jauge (§7 : les trois dénominateurs). */
  gaugeTeamOfLobby: string
  gaugePlayerOfTeam: string
  gaugePlayerOfLobby: string
  /** Lignes de la grille des cadences. */
  rowMyTeam: string
  rowLobby: string
  /** Segments de la piste du lobby. */
  segTeamRest: string
  segEnemy: string
  /**
   * LA LÉGENDE DE LA TEXTURE (D7, 2026-09-21) : plein = rapporté à mon équipe, hachuré =
   * rapporté au lobby ; sur les pistes, la hachure neutre désigne les adversaires. Elle
   * est POSÉE SOUS CHAQUE FORME qui porte la texture, au lieu d'être une phrase de
   * l'infobulle d'en-tête que personne n'ouvrait.
   */
  hatchLegendSolid: string
  hatchLegendLobby: string
  hatchLegendEnemy: string
  /**
   * LA LÉGENDE DE LA BANDE DE RÉGULARITÉ (2026-09-21) : les quatre encres, écrites UNE
   * fois sous les bandes. « X/X au-dessus de la parité » était répété à droite de chaque
   * ligne ; il ne reste que le compte « X/X », l'explication vit ici.
   */
  bandLegendAbove: string
  bandLegendNear: string
  bandLegendBelow: string
  bandLegendUnmeasured: string
  bandTipFmt: (index: number, share: string, parity: string) => string
  bandTipUnmeasured: (index: number) => string
  /** Textes d'honnêteté (comptes bruts en INFOBULLE, jamais dans une cellule — D2). */
  honestyFmt: (numerator: string, denominator: string) => string
  cadenceTipFmt: (who: string, metric: string, rate: string, raw: string) => string
  gaugeTipFmt: (metric: string, gauge: string, share: string, raw: string) => string
  trackTipFmt: (who: string, count: string, pct: string) => string
  notMeasured: string
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

  // ─── Variante COMPTES (P9, PLAN_EQUIPEMENT_GACHIS_2026-09-09, E5/E6) : Synthèse et
  // Escouade — axe en objets pris, aucun pourcentage dans les barres, aucun trait de
  // parité. Les libellés de grandeur (familles) et les trois issues réutilisent le
  // dictionnaire ci-dessus (equipmentFamilyLabel, outcomeUsed/Kept/Dropped) — ce
  // bloc n'ajoute que ce qui est propre à l'axe en comptes et aux deux donuts.
  /** Aide d'en-tête des deux cartes en variante comptes (pas de trait de parité ici). */
  cardHintEquipmentCounts: string
  cardHintWeaponCounts: string
  /** Titre de la carte donut, équipement puis armes spéciales, solo vs escouade. */
  viewEquipmentPartsSolo: string
  viewEquipmentPartsSquad: string
  viewWeaponPartsSolo: string
  viewWeaponPartsSquad: string
  /** Valeur de barre : le compte, jamais un pourcentage (P9). */
  countTakenFmt: (n: string) => string
  countPickupsFmt: (n: string) => string
  /** Graduation de fin d'axe (comptes) : le nombre reste seul en 0 et au milieu. */
  axisEquipmentTakenFmt: (n: string) => string
  /** Infobulle de base d'une barre en comptes (avant l'ajout des issues/repères,
   *  qui réutilise gaugeOutcomeTipFmt/gaugeReferenceTipFmt ci-dessus). */
  countsTipFmt: (label: string, value: string) => string
  /** La part « moi » du donut — même mot sur les deux pages (P10/P11, §3.3). */
  donutMe: string
  /** Sous-total « moi + mes amis » — absent quand aucun ami suivi n'est présent. */
  donutSquadSubtotal: string
}

export const USAGE_TEXT: Record<Locale, UsageText> = {
  fr: {
    blockEquipment: "Usages d'équipement",
    blockPadControl: 'Contrôle des armes spéciales',
    blockPadTiers: 'Contrôle des armes par niveau',
    cardHintPadTiers:
      "Les mêmes prises de socle, rangées par niveau d'arme. Le niveau vient de la CARTE : l'emplacement que le fichier de la carte pose — râtelier ou socle de puissance — confirmé par le socle du match ; et du film pour les armes de début de vie. Jamais du nom de l'arme : la même Hydra est de terrain sur une carte et de puissance sur une autre. Survolez un niveau pour le détail par arme.",
    padTierLabels: {
      base: 'Armes de base',
      terrain: 'Armes de terrain',
      puissance: 'Armes de puissance',
    },
    padTierNoPadsFmt: (n) =>
      `Sur ${n} match${n > 1 ? 's' : ''}, le mode n'allume aucun socle : il n'y a rien à classer.`,
    padTierUnmeasuredFmt: (n) =>
      `Sur ${n} match${n > 1 ? 's' : ''}, les emplacements de la carte ne sont pas dans la référence : le niveau n'a pas pu être établi.`,
    padTierRandomStartsFmt: (n) =>
      `Sur ${n} match${n > 1 ? 's' : ''}, les équipements de début de vie sont tirés au sort : pas d'arme de base à distinguer.`,
    padTierBaseToggleFmt: (n) => `Armes de base (${n})`,
    padTierUnclassifiedFmt: (n) => `${n} prises sur un emplacement non identifié.`,
    blockObjectives: 'Objectifs par rôle et par famille',
    blockUnavailableTitle: "Usages d'équipement, armes spéciales et objectifs",
    cardHintCadences:
      "Chaque valeur est un nombre PAR MATCH MESURÉ : le total de la ligne divisé par le nombre de matchs mesurés de la session. Camouflage et surbouclier comptés ici sont les épisodes ACTIFS, mesurés par le film ; les bonus ramassés au sol, eux, restent anonymes et sont comptés dans « Contrôle des armes spéciales » — les deux ne s'additionnent jamais.",
    cardHintShares:
      "Chaque barre est ta part du total de ton équipe sur la session ; le trait vertical marque la parité, la part d'un joueur moyen (100 divisé par l'effectif).",
    cardHintRegularity:
      "Une case par match mesuré, dans l'ordre de la session : au-dessus, à hauteur ou en dessous de la parité d'équipe de ce match-là.",
    cardHintPadControl:
      "Chaque barre est ta part du total de ton équipe sur la session ; le trait vertical marque la parité, la part d'un joueur moyen (100 divisé par l'effectif). Chaque ramassage vient de l'événement daté du film et porte son ramasseur : un ramassage que la mesure ne sait pas attribuer n'est compté pour personne, jamais deviné.",
    cardHintObjectives:
      "Chaque barre est ta part du total de ton équipe sur la session ; le trait vertical marque la parité, la part d'un joueur moyen (100 divisé par l'effectif). Ce bloc se mesure hors film : il couvre plus de matchs que les deux autres. Le rôle « Tenir » se mesure en durée — ses totaux sont en minutes:secondes, ses parts restent des pourcentages.",
    measuredFooterFmt: (m, t) => `Mesuré sur ${m} match${m > 1 ? 's' : ''} sur ${t}`,
    objectivesScopeFmt: (n, t) => `Matchs avec objectifs ${n}/${t}`,
    unavailableLoadFailed: "La lecture du résumé d'usage a échoué.",
    emptyNoFilm: "Aucun film décodé sur cette sélection.",
    emptyNoPads: "Aucun socle d'arme dans les modes de cette sélection.",
    emptyNoObjectives: "Aucun objectif dans les modes de cette sélection.",
    emptyTitleNoFilm: 'Aucune mesure',
    emptyTitleNoPads: 'Aucune prise de socle',
    emptyTitleNoObjectives: "Aucune mesure d'objectif",
    emptyTitleLoadFailed: 'Lecture impossible',
    viewCadences: 'Cadences par match',
    viewShares: 'Parts et parités',
    viewRegularity: 'Régularité match par match',
    viewLobbyTrack: 'Qui ramasse les armes spéciales',
    viewRoles: 'Par rôle, toutes familles confondues',
    viewFamilies: "Ma part d'équipe, par famille de mode",
    viewSquadRoles: "Part d'équipe par joueur et par rôle",
    viewFlagGrabsNet: 'Prises nettes de drapeau',
    gaugeTeamOfLobby: 'Mon équipe dans le lobby',
    gaugePlayerOfTeam: 'Ma part dans mon équipe',
    gaugePlayerOfLobby: 'Ma part dans le lobby',
    rowMyTeam: 'Mon équipe',
    rowLobby: 'Lobby',
    segTeamRest: 'Reste de mon équipe',
    segEnemy: 'Équipe adverse',
    hatchLegendSolid: "Plein : rapporté à mon équipe",
    hatchLegendLobby: 'Hachuré : rapporté au lobby',
    hatchLegendEnemy: 'Hachure neutre : les adversaires',
    bandLegendAbove: 'Au-dessus de la parité',
    bandLegendNear: 'Au niveau de la parité',
    bandLegendBelow: 'Sous la parité',
    bandLegendUnmeasured: 'Non mesuré',
    bandTipFmt: (i, share, parity) =>
      `Match ${i} — part d'équipe ${share} (parité de session ${parity})`,
    bandTipUnmeasured: (i) => `Match ${i} — part non mesurée`,
    honestyFmt: (n, d) => `${n} sur ${d}`,
    cadenceTipFmt: (who, metric, rate, raw) => `${who} — ${metric} : ${rate} par match (${raw} au total)`,
    gaugeTipFmt: (metric, gauge, share, raw) => `${metric} — ${gauge} : ${share} (${raw})`,
    trackTipFmt: (who, count, pct) => `${who} : ${count} ramassages (${pct})`,
    notMeasured: '—',
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
    roleTake: 'Prendre',
    roleDefend: 'Défendre',
    roleHold: 'Tenir',
    roleUnknownFmt: (key) => `Rôle ${key}`,
    familyCtf: 'Drapeau',
    familyKoth: 'Colline du roi',
    familyStrongholds: 'Bases',
    familyOddball: 'Crâne',
    familyStockpile: 'Stockage',
    familyExtraction: 'Extraction',
    familyVip: 'VIP',
    familyUnknownFmt: (key) => `Famille ${key}`,
    cardHintEquipmentCounts:
      "Chaque barre est le nombre d'objets pris sur la période ; son remplissage dit ce qui en a été fait. Les deux repères dans la barre sont des taux qui t'excluent : le trait plein est le taux du reste de ton équipe, le pointillé celui de eux.",
    cardHintWeaponCounts:
      "Chaque barre est le nombre de prises de socle d'arme sur la période. Le tir n'est pas mesuré à ce grain : la barre est un compte simple, pas ce qui a été fait de la prise.",
    viewEquipmentPartsSolo: "Ma part de l'équipement du lobby",
    viewEquipmentPartsSquad: "Notre part de l'équipement du lobby",
    viewWeaponPartsSolo: 'Ma part des armes spéciales du lobby',
    viewWeaponPartsSquad: 'Notre part des armes spéciales du lobby',
    countTakenFmt: (n) => `${n} pris`,
    countPickupsFmt: (n) => `${n} prises`,
    axisEquipmentTakenFmt: (n) => `${n} objets pris`,
    countsTipFmt: (label, value) => `${label} — ${value}`,
    donutMe: 'Moi',
    donutSquadSubtotal: 'Mon escouade',
  },
  en: {
    blockEquipment: 'Equipment usage',
    blockPadControl: 'Power weapon control',
    blockPadTiers: 'Weapon control by level',
    cardHintPadTiers:
      'The same pad pickups, sorted by weapon level. The level comes from the MAP: the spot the map file places — rack or power pedestal — confirmed by the match pad; and from the film for spawn weapons. Never from the weapon name: the same Hydra is a map weapon on one map and a power weapon on another. Hover a level for the per-weapon detail.',
    padTierLabels: {
      base: 'Starting weapons',
      terrain: 'Map weapons',
      puissance: 'Power weapons',
    },
    padTierNoPadsFmt: (n) =>
      `In ${n} match${n > 1 ? 'es' : ''}, the mode lights no pad at all: there is nothing to sort.`,
    padTierUnmeasuredFmt: (n) =>
      `In ${n} match${n > 1 ? 'es' : ''}, the map's weapon spots are not in the reference: the level could not be established.`,
    padTierRandomStartsFmt: (n) =>
      `In ${n} match${n > 1 ? 'es' : ''}, spawn loadouts are handed out at random: there is no starting weapon to single out.`,
    padTierBaseToggleFmt: (n) => `Base weapons (${n})`,
    padTierUnclassifiedFmt: (n) => `${n} pickups on an unidentified spot.`,
    blockObjectives: 'Objectives by role and family',
    blockUnavailableTitle: 'Equipment, power weapons and objectives',
    cardHintCadences:
      'Each value is a count PER MEASURED MATCH: the row total divided by the number of measured matches in the session. Camouflage and overshield counted here are the ACTIVE episodes measured by the film; power-ups picked up off the ground stay anonymous and are counted in "Power weapon control" — the two are never added together.',
    cardHintShares:
      'Each bar is your share of your team total over the session; the vertical mark is parity, the share of an average player (100 divided by headcount).',
    cardHintRegularity:
      "One square per measured match, in session order: above, at, or below that match's team parity.",
    cardHintPadControl:
      'Each bar is your share of your team total over the session; the vertical mark is parity, the share of an average player (100 divided by headcount). Every pickup comes from the timed film event and carries its picker: a pickup the measurement cannot attribute is counted for nobody, never guessed.',
    cardHintObjectives:
      'Each bar is your share of your team total over the session; the vertical mark is parity, the share of an average player (100 divided by headcount). This block is measured outside the film: it covers more matches than the other two. The "Hold" role is measured in duration — its totals are minutes:seconds, its shares remain percentages.',
    measuredFooterFmt: (m, t) => `Measured on ${m} match${m === 1 ? '' : 'es'} out of ${t}`,
    objectivesScopeFmt: (n, t) => `Matches with objectives ${n}/${t}`,
    unavailableLoadFailed: 'Loading the usage summary failed.',
    emptyNoFilm: 'No decoded film in this selection.',
    emptyNoPads: 'No weapon pad in the modes of this selection.',
    emptyNoObjectives: 'No objective in the modes of this selection.',
    emptyTitleNoFilm: 'Nothing measured',
    emptyTitleNoPads: 'No pad pickup',
    emptyTitleNoObjectives: 'No objective measured',
    emptyTitleLoadFailed: 'Could not load',
    viewCadences: 'Rate per match',
    viewShares: 'Shares and parity',
    viewRegularity: 'Match-by-match consistency',
    viewLobbyTrack: 'Who picks up the power weapons',
    viewRoles: 'By role, all families combined',
    viewFamilies: 'My team share, by mode family',
    viewSquadRoles: 'Team share by player and role',
    viewFlagGrabsNet: 'Net flag grabs',
    gaugeTeamOfLobby: 'My team in the lobby',
    gaugePlayerOfTeam: 'My share of my team',
    gaugePlayerOfLobby: 'My share of the lobby',
    rowMyTeam: 'My team',
    rowLobby: 'Lobby',
    segTeamRest: 'Rest of my team',
    segEnemy: 'Opposing team',
    hatchLegendSolid: 'Solid: measured against my team',
    hatchLegendLobby: 'Hatched: measured against the lobby',
    hatchLegendEnemy: 'Neutral hatch: the opponents',
    bandLegendAbove: 'Above parity',
    bandLegendNear: 'At parity',
    bandLegendBelow: 'Below parity',
    bandLegendUnmeasured: 'Not measured',
    bandTipFmt: (i, share, parity) =>
      `Match ${i} — team share ${share} (session parity ${parity})`,
    bandTipUnmeasured: (i) => `Match ${i} — share not measured`,
    honestyFmt: (n, d) => `${n} of ${d}`,
    cadenceTipFmt: (who, metric, rate, raw) => `${who} — ${metric}: ${rate} per match (${raw} total)`,
    gaugeTipFmt: (metric, gauge, share, raw) => `${metric} — ${gauge}: ${share} (${raw})`,
    trackTipFmt: (who, count, pct) => `${who}: ${count} pickups (${pct})`,
    notMeasured: '—',
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
    cardHintEquipmentCounts:
      'Each bar is the number of items taken over the period; its fill says what became of them. The two marks in the bar are rates that exclude you: the solid mark is the rate of the rest of your team, the dotted one is theirs.',
    cardHintWeaponCounts:
      "Each bar is the number of power weapon pad pickups over the period. Firing isn't measured at this grain: the bar is a plain count, not what became of the pickup.",
    viewEquipmentPartsSolo: "My share of the lobby's equipment",
    viewEquipmentPartsSquad: "Our share of the lobby's equipment",
    viewWeaponPartsSolo: "My share of the lobby's power weapons",
    viewWeaponPartsSquad: "Our share of the lobby's power weapons",
    countTakenFmt: (n) => `${n} taken`,
    countPickupsFmt: (n) => `${n} pickups`,
    axisEquipmentTakenFmt: (n) => `${n} items taken`,
    countsTipFmt: (label, value) => `${label} — ${value}`,
    donutMe: 'Me',
    donutSquadSubtotal: 'My squad',
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

