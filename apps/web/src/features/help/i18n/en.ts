/** Textes de l'aide en anglais (contrat : `contract.ts`). */
import type { HelpText } from './contract'

export const EN_TEXT = {
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
    search: {
      placeholder: 'Search a term or a definition…',
      sectionsLabel: 'Glossary sections',
      emptyTitle: 'No results',
      emptyDescription:
        'No term matches your search. Try a different wording.',
    },
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
              'Relative 0–100 score measuring the contribution of a player in a match vs their own history. Computed from 13 weighted metrics — each metric is converted to a percentile rank over the player\'s past matches. Requires at least 10 historical matches to activate. Bot-resistant: if bots were on the player\'s team, a corrective bonus is applied based on the MMR gap.',
            formula:
              'Score = Σ(percentile_rank × weight) / Σ(active weights), clamped 0–100\n\nMain weights:\n0.14 × kills/min · 0.11 × KDA · 0.10 × deaths/min (inverted)\n0.10 × personal score/min · 0.09 × kills vs expected\n0.09 × offensive conversion · 0.07 × deaths vs expected\n0.06 × damage/min · 0.06 × heroic medals · 0.05 × defensive resistance\n0.04 × accuracy · 0.04 × rank vs expected',
            example:
              '80/100 in a match = the performance was in the top 20 % of the player\'s own historical performances across those metrics.',
          },
          {
            term: 'Offensive Conversion',
            definition:
              'Offensive efficiency: measures how much damage is needed to convert a kill. The {{HP}} coefficient is the total health needed to down a Spartan in the current title and adjusts automatically to the selected game (on Halo Infinite: 90 base HP + 135 shields = 225). Above 1.0 = kills are finished with less damage than a full Spartan\'s health (accuracy, headshots that skip the shield). Below = damage is wasted (unconverted assists, poor follow-up).',
            formula:
              'Offensive conversion = {{HP}} × (kills + assists/3) / damage_dealt\n\nElite reference (gauge): 0.90',
            example:
              'With the {{HP}} HP baseline: many kills for little damage gives a high conversion (> 1.0); lots of damage for few kills, a low one (< 1.0).\n\nWorked example (Halo Infinite, 225 HP): 10 kills, 6 assists, 2 800 damage → 225 × (10 + 2) / 2 800 ≈ 0.96, above the elite reference (0.90).',
          },
          {
            term: 'Defensive Resistance',
            definition:
              'Measures how much damage is absorbed per death. Uses the same baseline: {{HP}} total health to down a Spartan in the current title (on Halo Infinite: 90 base HP + 135 shields = 225). Above 1.0 = death occurs after absorbing more than a full Spartan\'s health (good resilience, forces enemies to commit a full magazine). Below = death occurs early in engagements, often from surprise or poor positioning.',
            formula:
              'Defensive resistance = damage_taken / ({{HP}} × deaths)\n\nElite reference (gauge): 1.65',
            example:
              'With the {{HP}} HP baseline: absorbing more than one Spartan\'s health before dying gives a resistance > 1.0; dying early, a resistance < 1.0.\n\nWorked example (Halo Infinite, 225 HP): 5 deaths, 1 400 damage taken → 1 400 / (225 × 5) ≈ 1.24, i.e. 1.24× a Spartan\'s health absorbed per death.',
          },
          {
            term: 'Combat profile',
            definition:
              "Three independent descriptors summarising play style (labels shown in French), calibrated on the distribution of the world's best players.\n\n• Offensive (damage→kill conversion): from Dispersé (dispersed) to Chirurgical (surgical).\n• Defensive (damage absorbed per death): from Fragile to Inébranlable (unshakable).\n• Activity (absolute engagement): from Passif (passive) to Agressif (aggressive).\n\nActivity compares the player's event pace to their lobby's average player — an ABSOLUTE engagement, unlike the Engagement Score which compares the player to their own norm. The top bands match world-leader level: it's normal not to reach them. Shown from 15 matches with data.",
            formula:
              "Offensive (avg_oc) — thresholds 0.78 / 0.81 / 0.85 / 0.90.\nDefensive (avg_dr) — thresholds 1.20 / 1.35 / 1.50 / 1.65.\nActivity — mean(player_pace / lobby_pace), thresholds 0.80 / 0.92 / 1.08 / 1.25 (1.0 = lobby pace).",
            example:
              "OC 0.87 → Précis (precise) · DR 1.29 → Exposé (exposed) · ratio 0.99 → Mesuré (measured): a precise finisher who dies a bit fast and engages at the lobby average.",
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
        title: 'Participation Profile',
        entries: [
          {
            term: 'Impact',
            definition:
              'Normalised offensive conversion on shared squad matches. Measures offensive efficiency: how much damage is needed to convert a kill. The {{HP}} coefficient represents total Spartan health in the current title (on Halo Infinite: 90 base HP + 135 shields = 225). Above 0.90 (elite reference): efficient conversion. Below: wasted damage or unconverted assists.',
            formula: 'Impact = {{HP}} × (kills + assists/3) / damage dealt\nElite reference: 0.90',
            example: 'Worked example (Halo Infinite, 225 HP): 10 kills, 6 assists, 2 800 damage → 225 × 12 / 2 800 ≈ 0.96, above the elite reference.',
          },
          {
            term: 'Combat',
            definition:
              'Combat effectiveness on shared matches: rewards high-quality kills (headshots, perfect kills) and aiming accuracy. Headshots and perfect kills each count as half a kill, then the total is multiplied by an accuracy factor.',
            formula: 'Combat = (kills + ½ headshots + ½ perfect kills) × (1 + accuracy × 0.4)',
            example: '8 kills, 4 headshots, 1 perfect kill, 45% accuracy → (8 + 2 + 0.5) × 1.18 ≈ 12.4.',
          },
          {
            term: 'Survival',
            definition:
              'Normalised defensive resistance on shared matches. Measures ability to absorb damage before dying. Uses the same baseline as Impact: {{HP}} total health to down a Spartan in the current title (225 on Halo Infinite). Above 1.65 (elite reference): engagements are survived well beyond a full life. Below 1.0: death occurs early in most exchanges.',
            formula: 'Survival = damage taken / ({{HP}} × deaths)\nElite reference: 1.65',
            example: '5 deaths, 1 800 damage taken → 1 800 / 1 125 ≈ 1.60: close to the elite reference (1.65).',
          },
          {
            term: 'Support',
            definition:
              'Assist contribution on shared matches. Each assist is weighted by 50 — the same weight Halo assigns assists in the personal score — allowing Support to be compared with kill contribution on a common scale.',
            formula: 'Support = assists × 50',
            example: '12 assists → Support = 600.',
          },
          {
            term: 'Score',
            definition:
              'Residual personal score after subtracting direct kill and assist contribution. Captures added value from medals, streaks and game-mode actions (point defences, bonus captures, etc.) not counted in the other axes.',
            formula: 'Score = personal score − (kills × 100) − (assists × 50) − objective score, ≥ 0',
            example: 'Personal score 2 400, 8 kills, 4 assists, 200 objective pts → 2 400 − 800 − 200 − 200 = 1 200.',
          },
          {
            term: 'Objective',
            definition:
              'Objective participation measured PER OPPORTUNITY: only objective matches (Flag, Zones, King of the Hill, Oddball, Stockpile, Extraction, VIP) count — Slayer matches no longer dilute the value, and the axis disappears when no objective match is in scope. Each mode combines its weighted actions (captures, grabs, returns, secures…) and the time spent on the objective (flag carrying, zone occupation…), normalized by the number of matches of that mode and calibrated against real benchmarks: a player at their mode\'s P80 level scores 80/100.',
            formula:
              'Per mode: r = min(1.25; 0.65 × actions/P80 + 0.35 × objective time/P80)\nIndex = average of r weighted by each mode\'s match count',
            example:
              '3 Flag matches at P80 (r = 1.0) + 1 average Zones match (r = 0.5) → (3 × 1.0 + 0.5) / 4 ≈ 0.88 → 70/100.',
          },
        ],
      },
      {
        title: 'Match Narrative Badges',
        entries: [
          {
            term: 'How Badges Are Computed',
            definition:
              'Badges are determined from the score curve reconstructed kill-by-kill for each match. Only Win or Loss matches receive a badge (Tie and DNF are excluded). Matches with bot teammates may be excluded based on the configured preferences. Default sensitivity: "standard" (leadPct = 40 %, comebackPct = 35 %).',
            formula:
              'Thresholds (standard sensitivity):\n• leadThreshold = final_max_score × 0.40\n• comebackThreshold = final_max_score × 0.35\n\nDetection priority:\n1. Counter-Comeback (highest priority)\n2. Comeback\n3. Collapse\n4. Domination\n5. Humiliation',
            example:
              'Final score 50–32 (final_max_score = 50):\nleadThreshold = 50 × 0.40 = 20 kills lead\ncomebackThreshold = 50 × 0.35 = 17.5 kills gap before a reversal triggers',
          },
          {
            term: 'Domination',
            definition:
              'Victory in which the team maintained a significant and consistent lead throughout the match, never truly threatened. A badge of total control.',
            formula:
              'Conditions: Win + max player lead ≥ leadThreshold\n(and no comeback swing ≥ comebackThreshold)',
            example:
              'Final 50–28. The team always led by 20+ kills → Domination. If at some point the enemy closed to −5, that\'s too narrow to trigger another badge.',
          },
          {
            term: 'Humiliation',
            definition:
              "Defeat in which the enemy maintained an overwhelming lead from start to finish. The team never had a real chance to turn things around.",
            formula:
              'Conditions: Loss + max enemy lead ≥ leadThreshold\n(and no comeback swing ≥ comebackThreshold)',
            example:
              'Final 32–50. The enemy always led by 20+ kills → Humiliation.',
          },
          {
            term: 'Comeback',
            definition:
              'Victory after being significantly behind. The team was in a losing position (deficit ≥ threshold) at some point before turning the match around and winning.',
            formula:
              'Conditions: Win + the enemy had a lead ≥ comebackThreshold before the end of the match',
            example:
              'Final 50–45. At mid-match the team was −18 kills behind. The match was turned around → Comeback.',
          },
          {
            term: 'Collapse',
            definition:
              'Defeat after being in a strong position. The team led significantly at some point before falling apart and losing.',
            formula:
              'Conditions: Loss + the team had a lead ≥ comebackThreshold before the end of the match',
            example:
              'Final 45–50. At mid-match the team led by +18 kills. The enemy caught up → Collapse.',
          },
          {
            term: 'Counter-Comeback',
            definition:
              'The rarest badge: victory after a double reversal. The team led, then was overtaken and fell behind, before reclaiming the advantage and winning. Both teams had a significant lead at some point.',
            formula:
              'Conditions: Win + the team had a lead ≥ comebackThreshold at some point\n+ the enemy also had a lead ≥ comebackThreshold at another point\n(highest priority — detected before Comeback)',
            example:
              'The team led +20 in the first half. The enemy came back and overtook it by −18. The team pushed back and won 50–47 → Counter-Comeback.',
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
              'With 5 matches played since the last sync, delta sync fetches only those 5 matches, leaving the 500 already stored untouched.',
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
        title: 'Progression & Gamification',
        entries: [
          {
            term: 'Objectives',
            definition:
              "LevelUp's challenge system, separate from Halo's Mission Control challenges. Each objective targets a Halo metric (kills, KDA, accuracy, damage…) over a defined time window — daily, weekly, monthly, or free.\n\nObjectives are organised into arcs: thematic sequences that structure a progressive journey. An arc groups several related objectives (e.g. \"improve accuracy over 30 days\" followed by \"sustain KDA ≥ 3 the following week\").\n\nTwo evaluation modes:\n• Threshold — reach the target in a single match (e.g. \"20 kills in one match\").\n• Cumulative — accumulate across the full window (e.g. \"500 kills in a month\").\n\nTwo creation modes:\n• Free — the metric, target, and window are chosen freely.\n• Guided — LevelUp suggests an objective calibrated to the player's history.\n\nObjectives can be individual (stored in the player profile) or squad-level (shared across squad members). Squad challenges offer two dynamics:\n• Collective — every member contributes toward a shared target (e.g. \"1 000 kills combined this week\").\n• Competitive — members race on the same metric and rank against each other.\n\nEach completed objective awards Prestige Points (PP) based on its tier (Normal, Heroic, Legendary, Mythic).",
            example:
              'A free objective is created: "Average 3 000 damage per match over the next 10 matches", cumulative mode, Heroic tier. Once reached, the PP is earned and the objective moves to Completed in the journey.',
          },
          {
            term: 'Prestige',
            definition:
              "LevelUp's own progression layer, independent of the official Halo rank. Each completed objective awards Prestige Points (PP) that scale with the objective's tier.\n\nPP accumulate in the player profile and determine the Prestige tier:\n• Normal (grey) — first objectives\n• Heroic (blue) — consistent engagement\n• Legendary (purple) — advanced mastery\n• Mythic (gold) — elite level\n\nThe PP Leaderboard (Community sub-tab in Palmares) compares a player's PP against players in their squad and relations — derived automatically from shared match data, no manual friend management.\n\nAccess: navigation bar → Objectives for challenges and journey; Palmares → Community for the leaderboard.",
            example:
              'With 12 Heroic and 3 Legendary objectives completed, the total PP ranks second in the squad leaderboard, with a "Legendary" badge on the profile.',
          },
        ],
      },
      {
        title: 'Ascension & Progression',
        entries: [
          {
            term: 'Streak',
            definition:
              "A run of consecutive days or sessions satisfying a condition (playing every day, hitting a performance threshold, etc.). While the streak is active, a Prestige Points multiplier boosts completed objectives. Breaking the streak resets it to zero, unless a shield protects the gap.\n\nFour types: daily play, daily performance, weekly play, weekly KDA threshold.",
            example:
              "7 days played in a row → daily_play streak = 7. PP multiplier goes from ×1.0 to ×1.3. A day missed without a shield → streak breaks and restarts at 0.",
          },
          {
            term: 'Personal Best',
            definition:
              "Highest value ever reached on a given metric within a time window (30 days, 90 days, or all-time). Updated automatically after each match: when the PB is beaten, the previous record moves to history.",
            example:
              "The 30-day PB on \"kills per match\" was 22. A score of 25 is reached → new PB 25, the old one (22) slides into history with its timestamp.",
          },
          {
            term: 'Milestone',
            definition:
              "Achievement threshold crossed automatically and kept permanently (e.g. 1,000 cumulative kills, 100 wins, 50 Killing Spree medals). Unlike objectives, nothing needs to be activated — the milestone unlocks the moment the condition is met.",
            example:
              "The 10,000th lifetime kill is reached → the \"10K Veteran\" milestone is marked earned, with the date and triggering match.",
          },
          {
            term: 'Progression Arc',
            definition:
              "A coherent sequence of thematically linked objectives, designed to guide a targeted improvement over several weeks. An arc typically chains 3–8 steps (challenges) of increasing difficulty.",
            example:
              "The \"Accuracy\" arc may chain: (1) accuracy ≥ 45 % over 10 matches, (2) ≥ 50 % over 20 matches, (3) sustain ≥ 50 % for 30 days.",
          },
          {
            term: 'Moment Card',
            definition:
              "Retrospective card for a completed objective, archived in the Achievements tab. Contains the objective name, the value reached, match count, date and tier earned. The visual trace of a past accomplishment.",
          },
          {
            term: 'Contextual Pattern',
            definition:
              "Recurring signal detected by the Pattern Engine according to game context (mode, map, or squad composition). Highlights systematic over- or under-performance in certain configurations.",
            example:
              "\"On BTB solo, the win rate is 62 %; in a full squad it drops to 41 %.\" → contextual pattern by_squad with weakness signal.",
          },
          {
            term: 'Behavioral Pattern',
            definition:
              "Signal detected on in-game behaviour: tilt (degraded perf after losses), session fatigue (drop after N matches), engagement drop, accuracy plateau, performance ceiling. Used as a proactive coach alert.",
            example:
              "After 3 consecutive losses, average accuracy drops 8 points → tilt pattern with medium severity.",
          },
          {
            term: 'Calibrated Lever',
            definition:
              "Action prioritised by the Pattern Engine, with a target axis, current value, target value, time horizon and estimated LUSR impact. Distinct from \"LUSR Leverage\": this is a concrete action recommendation, not a mathematical component.",
          },
          {
            term: 'LUSR Leverage',
            definition:
              "LUSR component identified as having the largest personal improvement margin (gap between the current average and the player's top 20 %, or the target for the next tier). Working on this lever maximises impact on the overall LUSR.",
            example:
              "The \"accuracy\" component sits at 0.42 while the personal top 20 % is 0.58 — a strong lever (+38 % margin). The \"Start a campaign\" button launches an objective focused on that axis.",
          },
          {
            term: 'Engagement Tier',
            definition:
              "Play regularity level computed over the last 30 days from average matches per day and the longest gap between sessions. Four tiers: low, regular, high, intense. Used by the coach to calibrate suggestion pacing.",
          },
          {
            term: 'Play Style',
            definition:
              "Behavioural signature derived from the FK/FD ratio (First Kills / First Deaths). Four styles: opportunistic_finisher (high FK, low FD), overextended (high FK and FD), hyper_engaged (very high FK and FD), passive (low FK and FD).",
            example:
              "FK = 18, FD = 7 over the last 50 matches → ratio 2.57 → opportunistic_finisher style.",
          },
          {
            term: 'PP Multiplier',
            definition:
              "Coefficient applied to Prestige Points earned on an objective, scaling with the length of the most relevant active streak. Grows by tiers (1×, 1.1×, 1.3×, 1.5×, 2×…) then caps.",
            example:
              "A Heroic objective is completed (200 base PP) while the daily streak is at 14 → ×1.5 multiplier → 300 PP credited.",
          },
          {
            term: 'Pilot Mode',
            definition:
              "Mode where LevelUp automatically assigns calibrated objectives (3 daily, 5 weekly, 2 monthly) instead of the player, based on profile. Toggleable. Complementary to free objectives created manually.",
          },
          {
            term: 'Proactive Coach',
            definition:
              "Component that continuously proposes actions, objectives or campaigns based on detected patterns and calibrated levers. Toggleable in settings (notifications). Does not spam: only surfaces on a significant signal change.",
          },
        ],
      },
      {
        title: 'Navigation & Organisation',
        entries: [
          {
            term: 'Sessions',
            definition:
              'Automatic grouping of consecutive matches separated by less than 2 hours of inactivity. A session represents a continuous "gaming evening". Session analysis shows how performance evolves within a session (fatigue, warm-up, etc.).',
            example:
              '5 matches played between 8 PM and 10:30 PM → 1 session.\nA 6th match at 1 AM → a new, separate session.',
          },
          {
            term: 'Squad',
            definition:
              'A lineup of teammates analyzed together: Squad pages compute aggregated stats on matches played together (synergies, contributions, intensity heatmap). Several squads can be saved, named and reloaded.\n\nNot to be confused with Groups: a squad is an analysis lineup, it grants no data access.',
          },
          {
            term: 'Groups',
            definition:
              'Family/friend circles that define which databases can be accessed to explore stats. Inviting a member (via their Xbox sign-in) opens mutual access to each other’s data.\n\nNot to be confused with Squad: a group manages data ACCESS; a squad is a teammate analysis lineup.',
          },
          {
            term: 'Explorer',
            definition:
              'Drilldown view of all matches with cascade filters: map, mode, playlist, outcome, date, session. Enables fine-grained analysis and navigation to each match detail view.',
          },
          {
            term: 'Color Accessibility',
            definition:
              'Setting in the Accessibility tab (Settings) that switches the entire interface to the Okabe-Ito (2008) color palette. This palette uses 8 colors distinguishable by the three main types of color blindness (protanopia, deuteranopia, tritanopia). The preference is stored locally and does not affect other players.',
            example:
              'If the red/green performance bars look the same, the Okabe-Ito palette makes them distinguishable: the same performance levels are then encoded in sky blue, bluish green, yellow, orange and vermillion — distinct for all vision types.',
          },
          {
            term: 'Palmares',
            definition:
              'Section grouping rankings (local leaderboard by playlist/season), relationships (frequent allies, nemeses, frequent victims), player-vs-player comparison, and the Halo season pass.',
          },
        ],
      },
    ],
  },
} satisfies HelpText
