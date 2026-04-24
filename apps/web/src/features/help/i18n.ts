export type HelpLocale = 'fr' | 'en'

export type HelpTab = 'glossary' | 'release-notes'

export interface GlossaryEntry {
  term: string
  definition: string
  formula?: string
  example?: string
}

export interface GlossarySection {
  title: string
  entries: GlossaryEntry[]
}

export interface HelpText {
  tabs: {
    glossary: string
    releaseNotes: string
  }
  page: {
    title: string
    subtitle: string
  }
  releaseNotes: {
    loading: string
    error: string
    title: string
  }
  glossary: {
    title: string
    sections: GlossarySection[]
  }
}

const FR_TEXT: HelpText = {
  tabs: {
    glossary: 'Glossaire & Concepts',
    releaseNotes: 'Notes de version',
  },
  page: {
    title: 'Aide',
    subtitle: 'Documentation et concepts de LevelUp',
  },
  releaseNotes: {
    loading: 'Chargement des notes de version…',
    error: 'Impossible de charger les notes de version.',
    title: 'Notes de version',
  },
  glossary: {
    title: 'Glossaire & Concepts',
    sections: [
      {
        title: 'Métriques de performance',
        entries: [
          {
            term: 'LUSR',
            definition:
              "Score de compétence basé sur TrueSkill 2, adapté à Halo Infinite. C'est l'équivalent du CSR (rang classé) pour les modes non-classés : là où le CSR est attribué officiellement par Halo pour les playlists ranked, le LUSR est calculé localement par LevelUp sur tous les autres modes (arena, btb, social, fun…), en appliquant la même logique TrueSkill. Calculé séparément pour chaque groupe de playlist. Démarre à 1 500, avec une incertitude initiale (σ) de 350 qui diminue au fil des matchs. Seuls les résultats Win/Loss/Tie sont pris en compte — les DNF comptent faiblement (0.15). Un bonus de dérive s'applique en cas d'inactivité prolongée (> 1 jour) pour refléter la rouille éventuelle.",
            formula:
              'LUSR affiché = μ − 3σ\n\nΔμ = 32 × (score_composite − 0.5) × poids_groupe\nScore composite = 0.31×(kills/attendus) + 0.28×(morts attendus/morts) + 0.23×(dégâts/attendus) + 0.13×(précision delta) + 0.05×(win factor)\n\nPoids groupes : ranked 1.0 · arena 0.8 · btb 0.7 · fun 0.25\nCap : ±100 pts par match',
            example:
              'Un joueur avec μ = 1 700 et σ = 80 affiche LUSR 1 460. Après 2 semaines sans jouer, σ monte de ~13 points (1 pt/jour, plafonné à 14 jours), rendant le prochain match plus "impactant" dans les deux sens.',
          },
          {
            term: 'Score de performance',
            definition:
              "Score relatif 0–100 qui mesure votre contribution sur un match par rapport à votre propre historique. Calculé à partir de 13 métriques pondérées — chaque métrique est convertie en percentile sur vos matchs passés. Nécessite au minimum 10 matchs historiques pour être activé. Résiste aux bots : si des bots étaient dans votre équipe, un bonus correctif est appliqué selon l'écart MMR.",
            formula:
              'Score = Σ(percentile_rang × poids) / Σ(poids actifs), normalisé 0–100\n\nPoids principaux :\n0.14 × kills/min · 0.11 × KDA · 0.10 × morts/min (inversé)\n0.10 × score personnel/min · 0.09 × kills vs attendus\n0.09 × rendement offensif · 0.07 × morts vs attendues\n0.06 × dégâts/min · 0.06 × médailles héroïques\n0.05 × résistance défensive · 0.04 × précision · 0.04 × rang vs attendu',
            example:
              '80/100 sur un match = vous étiez dans le top 20 % de vos propres performances historiques sur cet ensemble de métriques.',
          },
          {
            term: 'Rendement offensif',
            definition:
              "Efficacité offensive : mesure combien de dégâts il vous faut pour convertir une élimination. Le coefficient 225 repose sur le postulat qu'un Spartan a 225 points de vie total (90 de vie de base + 135 de bouclier), convention officielle Halo Infinite. Au-dessus de 1.0 = vous éliminez avec moins de dégâts qu'attendu (précision, headshots qui sautent le bouclier). En dessous = vous gaspillez des dégâts (assists non convertis, suivi insuffisant).",
            formula:
              'Rendement offensif = 225 × (kills + assists/3) / dégâts infligés\n\nP80 de référence (données réelles) : 0.83\n(= 83 % des matchs ont un rendement ≤ 0.83)',
            example:
              '10 kills, 6 assists, 2 800 dégâts → 225 × (10 + 2) / 2 800 ≈ 0.96 : au-dessus du P80, vous convertissez efficacement.\n6 kills, 2 assists, 2 800 dégâts → 225 × (6 + 0.67) / 2 800 ≈ 0.54 : beaucoup de dégâts pour peu d\'éliminations.',
          },
          {
            term: 'Résistance défensive',
            definition:
              "Mesure combien de dégâts vous absorbez par mort. Repose sur le même postulat : un Spartan a 225 points de vie (90 de base + 135 de bouclier). Au-dessus de 1.0 = vous mourez après avoir encaissé plus qu'une vie de Spartan (bonne résistance, vous forcez les ennemis à vider leur chargeur). En dessous = vous mourrez rapidement, souvent surpris ou mal positionnés.",
            formula:
              'Résistance défensive = dégâts reçus / (225 × morts)\n\nP80 de référence (données réelles) : 1.59',
            example:
              '5 morts, 1 400 dégâts reçus → 1 400 / (225 × 5) ≈ 1.24 : au-dessus de 1.0, vous absorbez en moyenne 1.24× la vie d\'un Spartan avant de mourir.\n5 morts, 750 dégâts reçus → 750 / 1 125 ≈ 0.67 : vous mourez tôt dans les échanges.',
          },
          {
            term: 'KDA',
            definition:
              "Ratio kills/deaths/assists classique, avec un plancher à 1 mort pour éviter la division par zéro. Les assists comptent intégralement (contrairement au rendement offensif où elles valent 1/3). Métrique de référence mais sensible au volume — un joueur très actif peut avoir un KDA identique à un joueur passif.",
            formula: 'KDA = (kills + assists) / max(1, morts)',
            example:
              '15 kills, 4 assists, 6 morts → KDA = 19/6 ≈ 3.17\n0 kill, 0 assist, 0 mort → KDA = 0/1 = 0 (plancher à 1 mort)',
          },
        ],
      },
      {
        title: 'Badges narratifs de match',
        entries: [
          {
            term: 'Comment les badges sont calculés',
            definition:
              "Les badges sont déterminés à partir de la courbe de score reconstruite match par match, en analysant l'évolution du score kill par kill. Seuls les matchs Win ou Loss reçoivent un badge (Tie et DNF sont exclus). Les matchs avec des bots coéquipiers peuvent être exclus selon les préférences. Sensibilité par défaut : « standard » (leadPct = 40 %, comebackPct = 35 %).",
            formula:
              'Seuils (sensibilité standard) :\n• leadThreshold = score_final_max × 0.40\n• comebackThreshold = score_final_max × 0.35\n\nPriorité de détection :\n1. Contre-Remontada (priorité max)\n2. Remontada\n3. Débandade\n4. Domination\n5. Humiliation',
            example:
              'Match final 50–32 (score_final_max = 50) :\nleadThreshold = 50 × 0.40 = 20 kills d\'avance\ncomebackThreshold = 50 × 0.35 = 17.5 kills de retard/avance avant retournement',
          },
          {
            term: 'Domination 🏆',
            definition:
              "Victoire dans laquelle votre équipe a maintenu une avance significative et constante tout au long du match, sans jamais être réellement menacée. Badge de maîtrise totale.",
            formula:
              'Conditions : Win + avance max joueur ≥ leadThreshold\n(et jamais de retournement ≥ comebackThreshold)',
            example:
              'Score final 50–28. Votre équipe a toujours mené d\'au moins 20 kills → Domination. Si à un moment l\'adversaire avait remonté à –5 de vous, c\'est trop peu pour déclencher un autre badge.',
          },
          {
            term: 'Humiliation 💀',
            definition:
              "Défaite dans laquelle l'adversaire a maintenu une avance écrasante du début à la fin. Votre équipe n'a jamais vraiment été en position de renverser la tendance.",
            formula:
              'Conditions : Loss + avance max adverse ≥ leadThreshold\n(et jamais de retournement ≥ comebackThreshold)',
            example:
              'Score final 32–50. L\'adversaire a toujours mené de 20+ kills → Humiliation.',
          },
          {
            term: 'Remontada ⚡',
            definition:
              "Victoire après avoir été significativement menés. Votre équipe était en position défavorable (retard ≥ seuil) à un moment du match avant de retourner la situation et gagner.",
            formula:
              'Conditions : Win + l\'adversaire avait une avance ≥ comebackThreshold avant la fin du match',
            example:
              'Score final 50–45. À mi-match vous étiez à –18 kills. Vous avez renversé → Remontada.',
          },
          {
            term: 'Débandade 💔',
            definition:
              "Défaite après avoir été en position de force. Votre équipe menait significativement à un moment du match avant de s'effondrer et perdre.",
            formula:
              'Conditions : Loss + votre équipe avait une avance ≥ comebackThreshold avant la fin du match',
            example:
              'Score final 45–50. À mi-match vous meniez de +18 kills. L\'adversaire a rattrapé → Débandade.',
          },
          {
            term: 'Contre-Remontada 🔄',
            definition:
              "Badge le plus rare : victoire après un double retournement. Votre équipe a d'abord mené, puis été rattrapée et dépassée, avant de reprendre l'avantage et gagner. Les deux équipes ont eu une avance significative à un moment.",
            formula:
              'Conditions : Win + votre équipe avait une avance ≥ comebackThreshold à un moment\n+ l\'adversaire avait aussi une avance ≥ comebackThreshold à un autre moment\n(priorité maximale — détecté avant Remontada)',
            example:
              'Vous meniez +20 en première moitié. Adversaire remonte et vous dépasse de –18. Vous repartez et gagnez 50–47 → Contre-Remontada.',
          },
        ],
      },
      {
        title: 'Données & Synchronisation',
        entries: [
          {
            term: 'Synchronisation',
            definition:
              "Processus de récupération des matchs depuis l'API Waypoint de Halo et d'écriture dans la base DuckDB locale. En mode delta (par défaut), seuls les nouveaux matchs non encore enregistrés sont récupérés. Un sync complet peut être forcé depuis les Paramètres.",
            example:
              'Vous avez joué 5 matchs depuis la dernière sync. La sync delta récupère ces 5 matchs uniquement, sans retoucher aux 500 matchs déjà stockés.',
          },
          {
            term: 'Backfill',
            definition:
              "Re-calcul ou remplissage rétroactif de données manquantes sur l'historique déjà synchronisé. Utile après une mise à jour qui introduit un nouveau champ (ex. shots_fired, skill rank, médailles) : le backfill recalcule ce champ pour tous les matchs existants.",
            example:
              'Après l\'ajout du calcul de badge en v6.2, un backfill a été lancé pour calculer les badges Remontada/Débandade/Contre-Remontada sur tout l\'historique déjà synchronisé.',
          },
          {
            term: 'Normalisation des modes',
            definition:
              "Résolution d'un nom d'affichage unique à partir des variantes brutes renvoyées par l'API Waypoint. L'API peut retourner « BTB Slayer », « BTB-Slayer » ou « Big Team Battle Slayer » pour le même mode — la normalisation les unifie en « BTB — Slayer » via la table mode_pair_overrides.",
            example:
              'La table mode_pair_overrides dans metadata.duckdb contient ~29 surcharges FR/EN pour les cas ambigus.',
          },
          {
            term: 'Fréquences de rafraîchissement',
            definition: 'LevelUp utilise plusieurs niveaux de fraîcheur des données selon la page.',
            example:
              'Live (à chaque ouverture de page) : Accueil, Dernier match.\nCache query 5–10 min : Stats, Palmarès, Escouade.\nSync manuel : déclenché depuis le bouton Synchroniser dans les Paramètres.\nBackground auto : les médias sont ré-indexés après chaque sync.',
          },
        ],
      },
      {
        title: 'Navigation & Organisation',
        entries: [
          {
            term: 'Sessions',
            definition:
              "Regroupement automatique de matchs consécutifs séparés par moins de 2 heures d'inactivité. Une session représente une « soirée de jeu » continue. L'analyse des sessions permet d'étudier comment votre performance évolue au fil d'une même session (fatigue, warm-up, etc.).",
            example:
              '5 matchs joués entre 20h00 et 22h30 → 1 session.\nUn 6ème match joué à 01h00 → nouvelle session distincte.',
          },
          {
            term: 'Escouade',
            definition:
              "Groupe de joueurs synchronisés sur LevelUp qui partagent des matchs communs. Les pages Escouade calculent des stats agrégées sur les matchs joués ensemble : synergies, contributions, heatmap d'intensité. Un joueur ne fait partie de votre escouade que s'il est lui-même synchronisé dans l'app.",
          },
          {
            term: 'Explorer',
            definition:
              "Vue drilldown de tous vos matchs avec filtres en cascade : carte, mode, playlist, résultat, date, session. Permet une analyse fine et la navigation vers la vue détaillée de chaque match.",
          },
          {
            term: 'Palmarès',
            definition:
              "Section regroupant vos classements (leaderboard local par playlist/saison), vos relations (alliés fréquents, némésis, victimes), la comparaison face-à-face avec un autre joueur, et votre pass saisonnier Halo.",
          },
        ],
      },
    ],
  },
}

const EN_TEXT: HelpText = {
  tabs: {
    glossary: 'Glossary & Concepts',
    releaseNotes: 'Release Notes',
  },
  page: {
    title: 'Help',
    subtitle: 'LevelUp documentation and concepts',
  },
  releaseNotes: {
    loading: 'Loading release notes…',
    error: 'Unable to load release notes.',
    title: 'Release Notes',
  },
  glossary: {
    title: 'Glossary & Concepts',
    sections: [
      {
        title: 'Performance Metrics',
        entries: [
          {
            term: 'LUSR',
            definition:
              "Skill rating based on TrueSkill 2, adapted for Halo Infinite. Think of it as the equivalent of CSR (the official ranked rating) for unranked modes: where CSR is assigned by Halo for ranked playlists, LUSR is computed locally by LevelUp across all other modes (arena, btb, social, fun…) using the same TrueSkill logic. Computed separately for each playlist group. Starts at 1 500 with an initial uncertainty (σ) of 350 that decreases with each match. Only Win/Loss/Tie outcomes count — DNF counts weakly (0.15). An inactivity drift is applied after 1+ day without playing to reflect potential rust.",
            formula:
              'Displayed LUSR = μ − 3σ\n\nΔμ = 32 × (composite_score − 0.5) × group_weight\nComposite score = 0.31×(kills/expected) + 0.28×(expected deaths/deaths) + 0.23×(damage/expected) + 0.13×(accuracy delta) + 0.05×(win factor)\n\nGroup weights: ranked 1.0 · arena 0.8 · btb 0.7 · fun 0.25\nCap: ±100 pts per match',
            example:
              'A player with μ = 1 700 and σ = 80 displays LUSR 1 460. After 2 weeks inactive, σ rises ~13 pts (1 pt/day, capped at 14 days), making the next match more impactful in both directions.',
          },
          {
            term: 'Performance Score',
            definition:
              'Relative 0–100 score measuring your contribution in a match vs your own history. Computed from 13 weighted metrics — each metric is converted to a percentile rank over your past matches. Requires at least 10 historical matches to activate. Bot-resistant: if bots were on your team, a corrective bonus is applied based on the MMR gap.',
            formula:
              'Score = Σ(percentile_rank × weight) / Σ(active weights), clamped 0–100\n\nMain weights:\n0.14 × kills/min · 0.11 × KDA · 0.10 × deaths/min (inverted)\n0.10 × personal score/min · 0.09 × kills vs expected\n0.09 × offensive conversion · 0.07 × deaths vs expected\n0.06 × damage/min · 0.06 × heroic medals · 0.05 × defensive resistance\n0.04 × accuracy · 0.04 × rank vs expected',
            example:
              '80/100 in a match = you were in the top 20 % of your own historical performances across those metrics.',
          },
          {
            term: 'Offensive Conversion',
            definition:
              'Offensive efficiency: measures how much damage you need to convert a kill. The 225 coefficient is based on the assumption that a Spartan has 225 total health points (90 base HP + 135 shields), the official Halo Infinite convention. Above 1.0 = you finish kills with less damage than a full Spartan\'s health (accuracy, headshots that skip the shield). Below = you waste damage (unconverted assists, poor follow-up).',
            formula:
              'Offensive conversion = 225 × (kills + assists/3) / damage_dealt\n\nP80 reference (real data): 0.83\n(= 83 % of matches have conversion ≤ 0.83)',
            example:
              '10 kills, 6 assists, 2 800 damage → 225 × (10 + 2) / 2 800 ≈ 0.96: above P80, efficient conversion.\n6 kills, 2 assists, 2 800 damage → 225 × (6 + 0.67) / 2 800 ≈ 0.54: lots of damage for few kills.',
          },
          {
            term: 'Defensive Resistance',
            definition:
              'Measures how much damage you absorb per death. Uses the same assumption: a Spartan has 225 total health (90 base HP + 135 shields). Above 1.0 = you die after absorbing more than a full Spartan\'s health (good resilience, forces enemies to commit a full magazine). Below = you die early in engagements, often surprised or poorly positioned.',
            formula:
              'Defensive resistance = damage_taken / (225 × deaths)\n\nP80 reference (real data): 1.59',
            example:
              '5 deaths, 1 400 damage taken → 1 400 / (225 × 5) ≈ 1.24: above 1.0, you absorb 1.24× a Spartan\'s health before dying.\n5 deaths, 750 damage taken → 750 / 1 125 ≈ 0.67: you die early in most engagements.',
          },
          {
            term: 'KDA',
            definition:
              'Classic kills/deaths/assists ratio, with a floor of 1 death to avoid division by zero. Assists count fully (unlike offensive conversion where they count as 1/3). A reference metric but volume-sensitive — a very active player may have the same KDA as a passive one.',
            formula: 'KDA = (kills + assists) / max(1, deaths)',
            example:
              '15 kills, 4 assists, 6 deaths → KDA = 19/6 ≈ 3.17\n0 kills, 0 assists, 0 deaths → KDA = 0/1 = 0 (floor at 1 death)',
          },
        ],
      },
      {
        title: 'Match Narrative Badges',
        entries: [
          {
            term: 'How Badges Are Computed',
            definition:
              'Badges are determined from the score curve reconstructed kill-by-kill for each match. Only Win or Loss matches receive a badge (Tie and DNF are excluded). Matches with bot teammates may be excluded based on your preferences. Default sensitivity: "standard" (leadPct = 40 %, comebackPct = 35 %).',
            formula:
              'Thresholds (standard sensitivity):\n• leadThreshold = final_max_score × 0.40\n• comebackThreshold = final_max_score × 0.35\n\nDetection priority:\n1. Counter-Comeback (highest priority)\n2. Comeback\n3. Collapse\n4. Domination\n5. Humiliation',
            example:
              'Final score 50–32 (final_max_score = 50):\nleadThreshold = 50 × 0.40 = 20 kills lead\ncomebackThreshold = 50 × 0.35 = 17.5 kills gap before a reversal triggers',
          },
          {
            term: 'Domination 🏆',
            definition:
              'Victory in which your team maintained a significant and consistent lead throughout the match, never truly threatened. A badge of total control.',
            formula:
              'Conditions: Win + max player lead ≥ leadThreshold\n(and no comeback swing ≥ comebackThreshold)',
            example:
              'Final 50–28. Your team always led by 20+ kills → Domination. If at some point the enemy closed to −5, that\'s too narrow to trigger another badge.',
          },
          {
            term: 'Humiliation 💀',
            definition:
              "Defeat in which the enemy maintained an overwhelming lead from start to finish. Your team never had a real chance to turn things around.",
            formula:
              'Conditions: Loss + max enemy lead ≥ leadThreshold\n(and no comeback swing ≥ comebackThreshold)',
            example:
              'Final 32–50. The enemy always led by 20+ kills → Humiliation.',
          },
          {
            term: 'Comeback ⚡',
            definition:
              'Victory after being significantly behind. Your team was in a losing position (deficit ≥ threshold) at some point before turning the match around and winning.',
            formula:
              'Conditions: Win + the enemy had a lead ≥ comebackThreshold before the end of the match',
            example:
              'Final 50–45. At mid-match you were −18 kills behind. You reversed it → Comeback.',
          },
          {
            term: 'Collapse 💔',
            definition:
              'Defeat after being in a strong position. Your team led significantly at some point before falling apart and losing.',
            formula:
              'Conditions: Loss + your team had a lead ≥ comebackThreshold before the end of the match',
            example:
              'Final 45–50. At mid-match you led by +18 kills. The enemy caught up → Collapse.',
          },
          {
            term: 'Counter-Comeback 🔄',
            definition:
              'The rarest badge: victory after a double reversal. Your team led, then was overtaken and fell behind, before reclaiming the advantage and winning. Both teams had a significant lead at some point.',
            formula:
              'Conditions: Win + your team had a lead ≥ comebackThreshold at some point\n+ the enemy also had a lead ≥ comebackThreshold at another point\n(highest priority — detected before Comeback)',
            example:
              'You led +20 in the first half. Enemy came back and overtook you by −18. You pushed back and won 50–47 → Counter-Comeback.',
          },
        ],
      },
      {
        title: 'Data & Sync',
        entries: [
          {
            term: 'Sync',
            definition:
              'Process that fetches matches from the Halo Waypoint API and writes them to the local DuckDB database. Delta mode (default) only fetches new matches not yet recorded. A full sync can be forced from Settings.',
            example:
              'You played 5 matches since the last sync. Delta sync fetches only those 5 matches, leaving the 500 already stored untouched.',
          },
          {
            term: 'Backfill',
            definition:
              'Retroactive recalculation or population of missing data on already-synced history. Useful after an update that introduces a new field (e.g. shots_fired, skill rank, medals): backfill computes that field for all existing matches.',
            example:
              'After adding badge computation in v6.2, a backfill was run to calculate Comeback/Collapse/Counter-Comeback badges on all previously synced history.',
          },
          {
            term: 'Mode Normalisation',
            definition:
              'Resolution of a unique display name from the raw variants returned by the Waypoint API. The API may return "BTB Slayer", "BTB-Slayer" or "Big Team Battle Slayer" for the same mode — normalisation unifies them as "BTB — Slayer" via the mode_pair_overrides table.',
            example:
              'The mode_pair_overrides table in metadata.duckdb contains ~29 FR/EN overrides for ambiguous cases.',
          },
          {
            term: 'Refresh Frequencies',
            definition: 'LevelUp uses several data freshness levels depending on the page.',
            example:
              'Live (every page open): Home, Last Match.\nQuery cache 5–10 min: Stats, Palmares, Squad.\nManual sync: triggered from the Sync button in Settings.\nAuto background: media is re-indexed after every sync.',
          },
        ],
      },
      {
        title: 'Navigation & Organisation',
        entries: [
          {
            term: 'Sessions',
            definition:
              'Automatic grouping of consecutive matches separated by less than 2 hours of inactivity. A session represents a continuous "gaming evening". Session analysis lets you study how performance evolves within a session (fatigue, warm-up, etc.).',
            example:
              '5 matches played between 8 PM and 10:30 PM → 1 session.\nA 6th match at 1 AM → a new, separate session.',
          },
          {
            term: 'Squad',
            definition:
              'Group of LevelUp-synced players who share common matches. Squad pages compute aggregated stats on matches played together: synergies, contributions, intensity heatmap. A player only joins your squad if they are themselves synced in the app.',
          },
          {
            term: 'Explorer',
            definition:
              'Drilldown view of all your matches with cascade filters: map, mode, playlist, outcome, date, session. Enables fine-grained analysis and navigation to each match detail view.',
          },
          {
            term: 'Palmares',
            definition:
              'Section grouping your rankings (local leaderboard by playlist/season), your relationships (frequent allies, nemeses, frequent victims), player-vs-player comparison, and your Halo season pass.',
          },
        ],
      },
    ],
  },
}

const TEXT: Record<HelpLocale, HelpText> = { fr: FR_TEXT, en: EN_TEXT }

export function normalizeHelpLocale(locale?: string | null): HelpLocale {
  return locale === 'en' ? 'en' : 'fr'
}

export function getHelpText(locale?: string | null): HelpText {
  return TEXT[normalizeHelpLocale(locale)]
}
