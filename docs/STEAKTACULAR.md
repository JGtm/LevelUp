# Steaktacular Medal — Reverse-Engineering Notes

> **In-game name (French):** "À table"  
> **Medal ID:** `1169390319`  
> **Internal constant:** `MEDAL_STEAKTACULAR_ID` in `src/analysis/_medal_verdicts.py`

---

## Overview

The Steaktacular medal is awarded to the **winning team** when they defeat the opposing team by a large enough margin. 343 Industries has never published the exact conditions. This document summarises the patterns inferred from **1,432 matches** with medals loaded in `shared_matches.duckdb` (corpus collected as of March 2026).

Key figures:
- **57 matches** contain the medal (~4% of all medal-loaded matches)
- **141 individual medal awards** across those matches (multiple players from the winning team receive it)
- Modes where it appears: **Slayer, CTF, Strongholds, King of the Hill** (via Assassin, Fiesta, Ranked playlists)
- Modes where it **never appears** in this corpus: **BTB** (0/493 matches), Firefight, Other

---

## Methodology

All medals for **all players** in a match are stored in `shared.medals_earned` during sync (see `extract_all_medals()` in `src/data/sync/transformers/_medals.py`). This means enemy team medals are captured too, making the dataset complete.

To determine which side of the score boundary triggers the medal, winning players were identified via `outcome = 2` (WIN) in `match_participants`, then mapped to their team's score using `match_registry.team_0_score` / `team_1_score`.

---

## Conditions by Game Mode

### Slayer (objective: 50 kills)

| | No Steaktacular | With Steaktacular |
|---|---|---|
| **Loser score range** | 18–50 | **17–30** |
| **Sample count** | 512 | 24 |

**Observed threshold: loser score ≤ 30 (at most 60% of the 50-kill objective).**

The boundary is clean: no Steaktacular was awarded with loser ≥ 31 in this corpus. The minimum observed gap is **20 kills** (winner 50, loser 30).

> **Caveat:** matches where `loser = 29` appear in both groups. The "no steak" cases are almost certainly matches where our tracked players were on the *losing* side — the medal would have been awarded to the enemy team, but was still captured in the database. The effective trigger appears to be **gap ≥ 20** (loser ≤ 30).

---

### Capture the Flag (objective: 3 captures)

| | No Steaktacular | With Steaktacular |
|---|---|---|
| **Loser score range** | 0–3 | **always 0** |
| **Sample count** | 102 | 13 |

**Rule: complete shutout — the losing team scores zero captures.**

No Steaktacular was ever observed at 3–1 or 3–2 in this corpus. The condition appears to be a perfect shutout only.

---

### Strongholds (objective: 200 points)

| | No Steaktacular | With Steaktacular |
|---|---|---|
| **Loser score range** | 4–199 | **0–17** |
| **Sample count** | 40 | 7 |

**Observed threshold: loser score ≤ 17 (≤ 8.5% of the 200-point objective).**

The boundary is between 17 (Medal present) and 21 (no medal found), representing near-total zone domination for the duration of the match. Matches where the loser reached 16 or below were without exception awarded the medal.

---

### King of the Hill / Modes with Tick-Based Scoring

5 matches were observed where the winning team had a very large tick score (1,500–4,960 time units) while the losing team had 3. These are modes where the score represents cumulative time on-hill rather than discrete objectives. The pattern matches the general rule: the loser's score is a negligible fraction (< 0.2%) of the winner's.

---

## Summary Table

| Mode | Win condition | Steaktacular trigger |
|---|---|---|
| Slayer 4v4 | First to 50 kills | Gap ≥ 20 (loser ≤ 30) |
| CTF | First to 3 captures | Loser captures = 0 |
| Strongholds | First to 200 points | Loser ≤ ~17 pts (≤ ~9%) |
| KoTH (ticks) | Cumulative time | Loser < 1% of winner score |
| BTB | — | Not observed (0/493 matches) |

---

## Notes & Limitations

- **Selection bias:** our corpus only contains matches involving tracked players. For every "no medal" row where the loser score is near the threshold, the medal may have been awarded to the *opposing* team instead — and that award *is* also recorded in `medals_earned`. The analysis using `outcome = 2` isolates confirmed winning-side observations.

- **Dynamic trigger:** the medal is likely evaluated **at the exact moment the winning team reaches their score objective**, not at end-of-game recalculation. This means whether the medal fires depends on the loser's score at that instant, not a final comparative score.

- **BTB absent:** With an objective of 100 kills in BTB Slayer, the equivalent threshold would be a gap ≥ 40. This may simply be rare in our corpus, or the BTB variant may have different or no Steaktacular logic.

- **Fiesta / Ranked:** The same thresholds appear to apply — all observed Fiesta and Ranked Steaktacular matches follow the Slayer-50 rules above (same objective, same threshold).

---

## Internal Usage

The medal is used to compute `dominance_flag` stored in `player_match_enrichment`:

| Value | Meaning |
|---|---|
| `0` — `NONE` | Normal match |
| `1` — `DOMINATION` | Our team received Steaktacular |
| `2` — `HUMILIATION` | The enemy team received Steaktacular |

See `src/analysis/_medal_verdicts.py` and `src/data/dominance_backfill.py`.
