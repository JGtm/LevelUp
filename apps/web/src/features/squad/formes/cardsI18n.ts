/**
 * cardsI18n.ts — LES NEUF CARTES SOLO de l'artefact 2ec1b8eb (Séries temporelles) : titre,
 * note de pied, sous-titres, et les deux paragraphes de constat qui se calculent. Les cartes
 * du contexte escouade ont été retirées au lot L5.4 du plan
 * PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26 (onglet Emprise).
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

/** Les neuf cartes solo, dans l'ordre de l'artefact. */
export type FormesCardKey =
  | 'equipmentShares'
  | 'equipmentByMatch'
  | 'equipmentSpread'
  | 'padsGapSolo'
  | 'padsShareSolo'
  | 'padsWeaponGrid'
  | 'objectivesGapRole'
  | 'objectivesSharesByFamily'
  | 'objectivesRawGrid'

export interface FormesCardText {
  title: string
  note: string
}

export interface FormesCardsText {
  cards: Record<FormesCardKey, FormesCardText>
  subtitles: {
    myShareByMatch: string
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
    },
  },
  en: {
    subtitles: {
      myShareByMatch: 'Match by match — my share of the lobby',
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
    },
  },
}
