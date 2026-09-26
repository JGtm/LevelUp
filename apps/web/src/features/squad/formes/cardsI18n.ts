/**
 * cardsI18n.ts — LES DIX-NEUF CARTES de l'artefact 2ec1b8eb : titre, note de
 * pied, sous-titres, et les deux paragraphes de constat qui se calculent.
 *
 * LES NOTES DISENT LA MÉTHODE, PAS LA SOIRÉE. Celles de la maquette citaient les
 * chiffres de sa session (« le camouflage est à 79 % Madina97294 ») : ces
 * chiffres-là appartenaient à un exemple, pas au produit. Ce qui est porté ici,
 * c'est ce que chaque forme DIT et ce qu'elle ABANDONNE — la seule partie de la
 * note qui reste vraie quelle que soit la période affichée. Les chiffres, eux,
 * arrivent par les gabarits des constats, alimentés par les mesures réelles.
 *
 * Séparé de `i18n.ts` pour la taille (CLAUDE.md n°5), pas pour le sens.
 */
import type { Locale } from '@/lib/i18n/locale'

/** Les dix-neuf cartes, dans l'ordre de l'artefact. */
export type FormesCardKey =
  | 'equipmentShares'
  | 'equipmentByMatch'
  | 'equipmentSpread'
  | 'equipmentRegularity'
  | 'equipmentLobbyTrack'
  | 'equipmentSquadGrid'
  | 'equipmentSquadTrack'
  | 'padsGapSolo'
  | 'padsShareSolo'
  | 'padsWeaponGrid'
  | 'padsGapSquad'
  | 'padsTwoFriezes'
  | 'padsSquadByMatch'
  | 'padsSquadWeaponGrid'
  | 'objectivesGapRole'
  | 'objectivesSharesByFamily'
  | 'objectivesRawGrid'
  | 'objectivesGapSquad'
  | 'objectivesLobbyTrack'

export interface FormesCardText {
  title: string
  note: string
}

export interface FormesCardsText {
  cards: Record<FormesCardKey, FormesCardText>
  subtitles: {
    myShareByMatch: string
    whenTeamShare: string
    whoLobbyShare: string
  }
  familyMatchesFmt: (family: string, matches: number) => string
  families: Record<string, string>
  /** Le constat du bloc 2, alimenté par les mesures de la période. */
  weaponsConstatFmt: (v: {
    teamShare: string
    myHeavyShare: string
    heavyTeamShare: string
    precisionTeamShare: string
    lobbyParity: string
  }) => string
  weaponsConstatEmpty: string
  /** Le constat du bloc 3, alimenté par les mesures de la période. */
  objectivesConstatFmt: (v: {
    matchesWithObjective: number
    matchesTotal: number
    rolesAboveParity: number
    rolesMeasured: number
  }) => string
  objectivesConstatEmpty: string
}

export const FORMES_CARDS_TEXT: Record<Locale, FormesCardsText> = {
  fr: {
    subtitles: {
      myShareByMatch: 'Match par match — ma part du lobby',
      whenTeamShare: 'Quand — la part de mon camp, match par match',
      whoLobbyShare: 'Qui — la part de chacun dans le lobby',
    },
    familyMatchesFmt: (family, matches) =>
      `${family} — ${matches > 1 ? `${matches} matchs` : `${matches} match`}`,
    families: {
      ctf: 'Drapeau',
      zones_koth: 'Roi de la colline',
      zones_strongholds: 'Bases',
      oddball: 'Crâne',
      stockpile: 'Réserve',
      extraction: 'Extraction',
      vip: 'VIP',
    },
    weaponsConstatFmt: (v) =>
      `Mon camp prend **${v.teamShare} des prises de socle nommées**. Il tient les armes lourdes ` +
      `à ${v.heavyTeamShare} et les armes de précision à ${v.precisionTeamShare}. Moi, je rafle ` +
      `**${v.myHeavyShare} des armes lourdes du lobby** pour une parité à ${v.lobbyParity}.`,
    weaponsConstatEmpty:
      'Aucune prise de socle nommée sur cette période : les socles se lisent dans le film, et ' +
      'aucun match du scope n’en porte un décodé.',
    objectivesConstatFmt: (v) =>
      `**${v.matchesWithObjective} matchs sur ${v.matchesTotal}** portent un objectif, et pas les ` +
      'mêmes colonnes. La part les réconcilie : ramenées à **trois rôles** — prendre, défendre, ' +
      'tenir — les grandeurs de chaque mode deviennent comparables, et la parité reste la même ' +
      `quel que soit le mode. Sur cette période, mon camp est au-dessus de la parité sur ` +
      `**${v.rolesAboveParity} des ${v.rolesMeasured} rôles mesurés**.`,
    objectivesConstatEmpty:
      'Aucun match de cette période ne porte d’objectif — ce n’est pas un trou de mesure, c’est ' +
      'un ensemble de modes sans objectif.',
    cards: {
      equipmentShares: {
        title: 'Ma part, dans mon équipe et dans le lobby',
        note:
          '**Les deux dénominateurs se lisent l’un sous l’autre**, sur le même axe : la chute ' +
          'entre les deux pistes est la taille de mon équipe dans le lobby. C’est ce qui répond à ' +
          'la question « ai-je été un poids mort ? » — être à la parité de son équipe pendant que ' +
          'l’équipe est sous celle du lobby, ce n’est pas être en défaut, c’est jouer dans une ' +
          'équipe dominée sur cet axe.',
      },
      equipmentByMatch: {
        title: 'Usages d’équipement par match',
        note:
          '**Quand le jeu a changé dans la période.** Un usage qui n’apparaît que sur un match, ' +
          'c’est la carte qui le porte, pas une envie. Chaque colonne a **sa propre échelle** — un ' +
          'mur se compare à un mur. **Ce qu’elle abandonne** : au-delà d’une vingtaine de matchs ' +
          'elle demandera un repli.',
      },
      equipmentSpread: {
        title: 'Étendue et moyenne de la période',
        note:
          '**Une période en une ligne par usage** — la moyenne par match, et la dispersion autour ' +
          'd’elle. Une moyenne basse peut cacher un match à zéro et un match très haut : l’écart ' +
          'n’est pas du bruit, c’est le mode et la carte. **Ce qu’elle abandonne** : l’ordre des ' +
          'matchs — on ne voit plus QUAND le pic a eu lieu, c’est la grille au-dessus qui le dit.',
      },
      equipmentRegularity: {
        title: 'Régularité match par match',
        note:
          '**Un écart systématique se voit à la couleur d’une ligne entière**, un accident se voit ' +
          'à une case isolée. **Ce qu’elle abandonne** : le volume, et l’intensité sature à trente ' +
          'points.',
      },
      equipmentLobbyTrack: {
        title: 'Part de mon camp',
        note:
          '**Deux questions d’un coup** : combien mon camp prend du lobby (la partie colorée face ' +
          'au trait de parité) et qui le prend chez nous (les segments). Complémentaire de la ' +
          'bande ci-dessus : celle-ci dit *quand*, celle-là dit *qui*.',
      },
      equipmentSquadGrid: {
        title: 'Cadence de chacun sur la période',
        note:
          '**Le même usage comparé entre coéquipiers, sur la même base — la moyenne par match ' +
          'mesuré.** **Ce qu’elle abandonne** : la variation d’un match à l’autre, écrasée par la ' +
          'moyenne.',
      },
      equipmentSquadTrack: {
        title: 'Qui porte quel usage dans l’escouade',
        note:
          '**La répartition des rôles à l’intérieur du groupe.** **Ce qu’elle abandonne — et c’est ' +
          'important** : le dénominateur est l’escouade SEULE, pas le lobby. Il n’y a donc **pas ' +
          'de trait de parité** ici : la barre ne dit rien de l’adversaire. À lire avec ' +
          '« Part de mon camp ».',
      },
      padsGapSolo: {
        title: 'Écart à la parité, par famille d’arme',
        note:
          '**Elle ne dit pas combien j’ai raflé, elle dit quelles armes je vais chercher** — un ' +
          'profil de jeu, pas un compteur. La table famille d’arme → catégorie (lourde / précision ' +
          '/ autre) vient du registre d’armes du titre, jamais d’une liste écrite dans l’écran.',
      },
      padsShareSolo: {
        title: 'Ma part des prises de socle, et sa dispersion',
        note:
          'La jauge donne l’ensemble, la bande dessous donne le détail sans coûter une carte de ' +
          'plus. Quand ma part varie fortement d’une carte à l’autre, **le contrôle des socles ' +
          'dépend plus du terrain que du joueur**.',
      },
      padsWeaponGrid: {
        title: 'Taux de rafle par arme',
        note:
          '**La grandeur comparable entre armes et entre périodes** — un pourcentage, dénominateur ' +
          'affiché à côté. Rafler 4 lance-roquettes sur 15 apparitions n’a rien à voir avec 4 sur ' +
          '4. **Ce qu’elle abandonne** : un petit dénominateur reste fragile (une arme apparue deux ' +
          'fois donne 50 % ou 0 %), et la colonne d’occupations est là pour qu’on le voie. Les ' +
          'armes les plus rares sont repliées hors du tableau.',
      },
      padsGapSquad: {
        title: 'Écart à la parité, par famille d’arme',
        note:
          'Le rapport de force arme par arme. Une moyenne proche de la parité peut cacher deux ' +
          'faits opposés — une famille tenue, une autre perdue.',
      },
      padsTwoFriezes: {
        title: 'Détail des prises d’armes spéciales',
        note:
          '**Les deux frises dans une seule carte, l’une au-dessus de l’autre** : elles répondent ' +
          'à deux questions qui ne se posent jamais séparément — *quand* le camp a tenu les socles, ' +
          'et *qui* les a tenus. **Ce qu’elle abandonne** : les occupations de socle dont ' +
          'l’événement natif ne nomme pas le ramasseur ne sont versées à aucun camp — la barre ' +
          'porte les prises NOMMÉES, pas la totalité des socles.',
      },
      padsSquadByMatch: {
        title: 'Emprise de l’escouade, match par match',
        note:
          '**Sur quelle carte l’escouade a tenu les armes, et sur laquelle elle les a laissées.** ' +
          'Même grammaire que la piste du lobby, mais une ligne par match au lieu d’une ligne par ' +
          'axe. **Ce qu’elle abandonne** : quelle arme — c’est un bilan de contrôle, pas un ' +
          'inventaire.',
      },
      padsSquadWeaponGrid: {
        title: 'Taux de rafle de chaque coéquipier',
        note:
          '**La spécialisation.** Chaque colonne a sa propre échelle, donc chaque joueur se lit ' +
          'contre lui-même, puis les trois se comparent par les chiffres. **Ce qu’elle abandonne** : ' +
          'le dénominateur est le même pour tous (les occupations du socle), donc la somme des ' +
          'colonnes ne fait pas 100 % — le reste est allé au lobby adverse. C’est la piste ' +
          'ci-dessus qui dit ce reste.',
      },
      objectivesGapRole: {
        title: 'Écart à la parité, par rôle',
        note:
          '**Deux lignes par rôle : mon équipe, puis le lobby.** Un rôle au-dessus dans les deux ' +
          'est un trait de jeu, pas un effet d’équipe. **Ce qu’elle abandonne** : la nature de ' +
          'l’action — trois retours de drapeau et trois zones sécurisées y pèsent pareil.',
      },
      objectivesSharesByFamily: {
        title: 'Ma part par famille de mode',
        note:
          'Les colonnes RÉELLES de chaque mode, sans les fondre dans trois rôles — mais toutes ' +
          'ramenées à la même unité, la part, donc lisibles l’une sous l’autre. **La vue par rôle ' +
          'dit le profil, celle-ci dit l’usage.**',
      },
      objectivesRawGrid: {
        title: 'Objectif par mode, en valeurs brutes',
        note:
          '**La vérité brute, sans agrégat forcé.** Chaque famille de mode garde SES colonnes. ' +
          'C’est la carte à ouvrir quand une part surprend : elle donne le compte qui l’a produite. ' +
          '**À dire à l’écran** : les matchs sans objectif n’y figurent pas — ce n’est pas un trou ' +
          'de mesure.',
      },
      objectivesGapSquad: {
        title: 'Rapport de force par famille de mode',
        note:
          'Le rapport de force sur les grandeurs que le mode publie vraiment. On y lit ce que les ' +
          'trois rôles fusionnaient — une colonne perdue pendant qu’une autre est dominée.',
      },
      objectivesLobbyTrack: {
        title: 'Ce que mon camp prend de l’objectif',
        note:
          '**La carte de synthèse du bloc** : la partie colorée dit l’avantage sur le lobby, les ' +
          'segments disent le rôle de chacun. **Ce qu’elle abandonne** : l’axe « Tenir » est en ' +
          'secondes et les deux autres en actions — les trois barres ne se comparent qu’à leur ' +
          'propre parité.',
      },
    },
  },
  en: {
    subtitles: {
      myShareByMatch: 'Match by match — my share of the lobby',
      whenTeamShare: 'When — my side’s share, match by match',
      whoLobbyShare: 'Who — each player’s share of the lobby',
    },
    familyMatchesFmt: (family, matches) =>
      `${family} — ${matches > 1 ? `${matches} matches` : `${matches} match`}`,
    families: {
      ctf: 'Capture the Flag',
      zones_koth: 'King of the Hill',
      zones_strongholds: 'Strongholds',
      oddball: 'Oddball',
      stockpile: 'Stockpile',
      extraction: 'Extraction',
      vip: 'VIP',
    },
    weaponsConstatFmt: (v) =>
      `My side takes **${v.teamShare} of the named pad pickups**. It holds heavy weapons at ` +
      `${v.heavyTeamShare} and precision weapons at ${v.precisionTeamShare}. I take ` +
      `**${v.myHeavyShare} of the lobby heavy weapons** against a parity of ${v.lobbyParity}.`,
    weaponsConstatEmpty:
      'No named pad pickup on this period: pads are read from the film, and no match in scope ' +
      'carries a decoded one.',
    objectivesConstatFmt: (v) =>
      `**${v.matchesWithObjective} of ${v.matchesTotal} matches** carry an objective, and not the ` +
      'same columns. Shares reconcile them: reduced to **three roles** — take, defend, hold — each ' +
      'mode’s measures become comparable, and parity stays the same whatever the mode. On this ' +
      `period, my side is above parity on **${v.rolesAboveParity} of the ${v.rolesMeasured} ` +
      'measured roles**.',
    objectivesConstatEmpty:
      'No match on this period carries an objective — this is not a measurement gap, it is a set ' +
      'of modes without objectives.',
    cards: {
      equipmentShares: {
        title: 'My share, in my team and in the lobby',
        note:
          '**Both denominators read one under the other**, on the same axis: the drop between the ' +
          'two tracks is the size of my team within the lobby. That is what answers “was I dead ' +
          'weight?” — being at your team’s parity while the team is below the lobby’s is not a ' +
          'failing, it is playing in a team that is dominated on that axis.',
      },
      equipmentByMatch: {
        title: 'Equipment usage per match',
        note:
          '**When the game changed over the period.** An action that appears on a single match is ' +
          'the map carrying it, not a whim. Each column has **its own scale** — a wall compares to ' +
          'a wall. **What it gives up**: beyond twenty-odd matches it will need a fold.',
      },
      equipmentSpread: {
        title: 'Spread and average of the period',
        note:
          '**A period in one row per action** — the average per match, and the spread around it. A ' +
          'low average can hide a match at zero and a very high one: the gap is not noise, it is ' +
          'the mode and the map. **What it gives up**: the order of matches — you no longer see ' +
          'WHEN the peak happened; the grid above says that.',
      },
      equipmentRegularity: {
        title: 'Regularity, match by match',
        note:
          '**A systematic gap shows as the colour of a whole row**, an accident shows as a single ' +
          'cell. **What it gives up**: volume, and intensity saturates at thirty points.',
      },
      equipmentLobbyTrack: {
        title: 'My side’s share',
        note:
          '**Two questions at once**: how much my side takes from the lobby (the coloured part ' +
          'against the parity line) and who takes it among us (the segments). Complementary to the ' +
          'band above: this one says *who*, that one says *when*.',
      },
      equipmentSquadGrid: {
        title: 'Each player over the period',
        note:
          '**The same action compared across teammates, on the same basis — the average per ' +
          'measured match.** **What it gives up**: the variation from one match to the next, ' +
          'flattened by the average.',
      },
      equipmentSquadTrack: {
        title: 'Who carries which usage in the squad',
        note:
          '**How roles split inside the group.** **What it gives up — and it matters**: the ' +
          'denominator is the squad ALONE, not the lobby. There is therefore **no parity line** ' +
          'here: the bar says nothing about the other team. Read it with “My side’s share”.',
      },
      padsGapSolo: {
        title: 'Gap to parity, by weapon family',
        note:
          '**It does not say how much I took, it says which weapons I go for** — a play profile, ' +
          'not a counter. The weapon family → category table (heavy / precision / other) comes ' +
          'from the title weapon registry, never from a list written in the screen.',
      },
      padsShareSolo: {
        title: 'My share of pad pickups, and its spread',
        note:
          'The gauge gives the whole, the band below gives the detail without costing another ' +
          'card. When my share swings from map to map, **pad control depends more on the terrain ' +
          'than on the player**.',
      },
      padsWeaponGrid: {
        title: 'Take rate by weapon',
        note:
          '**The measure that compares across weapons and across periods** — a percentage, with ' +
          'its denominator next to it. Taking 4 rocket launchers out of 15 spawns is nothing like ' +
          '4 out of 4. **What it gives up**: a small denominator stays fragile (a weapon that ' +
          'appeared twice gives 50% or 0%), and the occupations column is there so you see it. The ' +
          'rarest weapons are folded out of the table.',
      },
      padsGapSquad: {
        title: 'Gap to parity, by weapon family',
        note:
          'The balance of power weapon by weapon. An average close to parity can hide two opposite ' +
          'facts — one family held, another lost.',
      },
      padsTwoFriezes: {
        title: 'Special weapon pickups in detail',
        note:
          '**Both friezes in a single card, one above the other**: they answer two questions that ' +
          'are never asked separately — *when* the side held the pads, and *who* held them. **What ' +
          'it gives up**: pad occupations whose native event does not name the picker belong to no ' +
          'side — the bar carries NAMED pickups, not the totality of the pads.',
      },
      padsSquadByMatch: {
        title: 'Squad grip, match by match',
        note:
          '**On which map the squad held the weapons, and on which it let them go.** Same grammar ' +
          'as the lobby track, but one row per match instead of one row per axis. **What it gives ' +
          'up**: which weapon — this is a control summary, not an inventory.',
      },
      padsSquadWeaponGrid: {
        title: 'Take rate of each teammate',
        note:
          '**Specialisation.** Each column has its own scale, so each player reads against ' +
          'themselves, then the numbers compare them. **What it gives up**: the denominator is the ' +
          'same for all (pad occupations), so the columns do not add up to 100% — the rest went to ' +
          'the other team. The track above says that rest.',
      },
      objectivesGapRole: {
        title: 'Gap to parity, by role',
        note:
          '**Two rows per role: my team, then the lobby.** A role above in both is a play trait, ' +
          'not a team effect. **What it gives up**: the nature of the action — three flag returns ' +
          'and three zones secured weigh the same here.',
      },
      objectivesSharesByFamily: {
        title: 'My share by mode family',
        note:
          'The REAL columns of each mode, without melting them into three roles — but all reduced ' +
          'to the same unit, the share, so they read one under the other. **The role view says the ' +
          'profile, this one says the action.**',
      },
      objectivesRawGrid: {
        title: 'Objective by mode, in raw values',
        note:
          '**The raw truth, without a forced aggregate.** Each mode family keeps ITS columns. This ' +
          'is the card to open when a share surprises you: it gives the count that produced it. ' +
          '**To say on screen**: matches without an objective are not listed — that is not a ' +
          'measurement gap.',
      },
      objectivesGapSquad: {
        title: 'Balance of power by mode family',
        note:
          'The balance of power on the measures the mode really publishes. It shows what the three ' +
          'roles were merging — one column lost while another is dominated.',
      },
      objectivesLobbyTrack: {
        title: 'What my side takes of the objective',
        note:
          '**The summary card of the block**: the coloured part says the advantage over the lobby, ' +
          'the segments say each player’s role. **What it gives up**: the “Hold” axis is in seconds ' +
          'and the other two in actions — the three bars only compare to their own parity.',
      },
    },
  },
}
