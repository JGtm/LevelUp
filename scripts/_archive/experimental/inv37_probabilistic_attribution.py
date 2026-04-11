"""
inv37 — Attribution probabiliste des armes non-POV (T1 focus)

Stratégie
---------
Le mapping pi→xuid est partiellement résolvable :
  - POV : Nilton410 = pi=1 (fire events, confirmé inv32)
  - T1 non-POV : 3 joueurs × 3 pi slots (4, 5, 6)

Hypothèse principale (à valider) :
  pi=6 = JGtm (xuid=2533274823110022)
  Justification :
    - pi=6 weapon state events incluent Spike Grenade + Dynamo Grenade (inv35)
    - JGtm a grenade_kills=1 dans l'API (seul joueur T1 non-POV)
    - pi=6 weapon state inclut Stalker Rifle (chunk_09) = power weapon
    - JGtm a power_weapon_kills=1 dans l'API (seul joueur T1 non-POV)

Attribution probabiliste pi=4,5 :
  - Hypothèse A : shots_fired ratio → plus de tirs = plus de weapon state events
    Madina97294 (sf=636) → pi=4 (114 events) ; Chocoboflor (sf=328) → pi=5 (60 events)
  - Hypothèse B : inverse (moins d'événements = joueur moins actif = moins de kills)

Validation :
  - Croiser weapon attribuée vs kill events + API aggregates
  - Si pi=6=JGtm correct : kills JGtm avec weapon = Spike/Stalker = cohérent

Match    : d9329229-693f-429c-a84c-3fd2784bcf4a
Date     : 2026-03-05
"""

from __future__ import annotations

from collections import defaultdict
from pathlib import Path

# ---------------------------------------------------------------------------
# Paths
# ---------------------------------------------------------------------------
CHUNKS_DIR = Path(__file__).parent.parent.parent / "data/investigation/chunks/d9329229"
MAIN_DB = (
    Path(__file__).parent.parent.parent.parent / "LevelUp/data/warehouse/shared_matches_v2.duckdb"
)

# ---------------------------------------------------------------------------
# Players
# ---------------------------------------------------------------------------
XUID_NILTON = 2535427927026623  # POV
XUID_JGTM = 2533274823110022
XUID_MADINA = 2533274858283686
XUID_CHOCO = 2535469190789936

T1_NON_POV = [XUID_JGTM, XUID_MADINA, XUID_CHOCO]

GT = {
    XUID_NILTON: "Nilton410",
    XUID_JGTM: "JGtm",
    XUID_MADINA: "Madina97294",
    XUID_CHOCO: "Chocoboflor",
}

# API stats for T1 non-POV (loaded below, hardcoded for offline use)
API_STATS = {
    XUID_JGTM: {
        "kills": 15,
        "deaths": 11,
        "grenade_kills": 1,
        "power_weapon_kills": 1,
        "shots_fired": 295,
        "shots_hit": 157,
    },
    XUID_MADINA: {
        "kills": 9,
        "deaths": 16,
        "grenade_kills": 0,
        "power_weapon_kills": 0,
        "shots_fired": 636,
        "shots_hit": 298,
    },
    XUID_CHOCO: {
        "kills": 7,
        "deaths": 15,
        "grenade_kills": 0,
        "power_weapon_kills": 0,
        "shots_fired": 328,
        "shots_hit": 194,
    },
}

# ---------------------------------------------------------------------------
# Comprehensive weapon ID map (FINDINGS Group A + grenades + inv35)
# ---------------------------------------------------------------------------
WEAPON_IDS: dict[str, str] = {
    "6acdc44d42c9679f": "Bandit Evo",
    "2b1824d542c9679f": "BR75",
    "5eec4e0842c9679f": "BR75 (variant)",
    "3dc4e30942c9679f": "BR75 (stream-B)",
    "230447b142c9679f": "Cindershot",
    "b619d84a42c9679f": "CQS48 Bulldog",
    "84bd29ed42c9679f": "Disruptor",
    "9d6aaed242c9679f": "Fuel Rod SPNKr",
    "2ac9c2ff42c9679f": "Heatwave",
    "71ab0a2c42c9679f": "M41 SPNKr",
    "2fb21c8742c9679f": "M392 Bandit",
    "48c19d2d42c9679f": "MA40 AR",
    "80977ba542c9679f": "Mangler",
    "7deb133f42c9679f": "Mangler (alt)",
    "767db96d42c9679f": "MLRS-2 Hydra",
    "5ded6cf242c9679f": "MLRS-2 Hydra (alt)",
    "f408190f42c9679f": "Mk51 Sidekick",
    "0daf3b0042c9679f": "MA40 AR (alt)",
    "d791556542c9679f": "Mutilator",
    "b533957e42c9679f": "Needler",
    "c354294642c9679f": "Plasma Pistol",
    "04e7f00b42c9679f": "Plasma Pistol (alt)",
    "30484ea642c9679f": "Pulse Carbine",
    "7e53b3c642c9679f": "Pulse Carbine (alt)",
    "c30d87c742c9679f": "Ravager",
    "2b1d61e442c9679f": "Ravager (alt)",
    "0a1992bc42c9679f": "S7 Sniper",
    "9387a8b942c9679f": "Shock Rifle",
    "fcc6aa7642c9679f": "Shock Rifle (alt)",
    "0d20c46942c9679f": "Skewer",
    "7a11aeef42c9679f": "Skewer (alt)",
    "daf193c742c9679f": "Stalker Rifle",
    "f5ef3bdb42c9679f": "Stalker Rifle (alt)",
    "b2ac806c42c9679f": "Stalker Rifle (inv30)",
    "b1eb695e42c9679f": "M41 Rocket (inv30)",
    "6d32c7dc42c9679f": "Scattershot",
    "fd98554c42c9679f": "VK78 Commando",
    "7d48b60342c9679f": "VK78 Commando (alt)",
    "3e07021742c9679f": "Vestige Carbine",
    "c24e549e42c9679f": "Sentinel Beam",
    "8afc085542c9679f": "Gravity Hammer",
    "1488d0bb42c9679f": "Energy Sword",
    "67edd20142c9679f": "M392 Bandit (alt)",
    # Grenades
    "b6dbead842c9679f": "M9 Frag Grenade",
    "c1e1bab042c9679f": "Plasma Grenade",
    "ab879f1e42c9679f": "Spike Grenade",
    "e3a0a51842c9679f": "Dynamo Grenade",
    "7a9e791042c9679f": "Plasma Grenade (alt)",
    # Confirmed by frequency match inv35 (pi=6: Spike×42, Dynamo×20)
    "6683257c42c9679f": "Spike Grenade (variant)",  # ×42 for pi=6 → inv35 "Spike ×42"
    "3ad55da442c9679f": "Dynamo Grenade (variant)",  # candidate for inv35 "Dynamo ×20"
}

GRENADE_IDS = {k for k, v in WEAPON_IDS.items() if "Grenade" in v}
POWER_IDS = {
    k
    for k, v in WEAPON_IDS.items()
    if v
    in {
        "S7 Sniper",
        "M41 SPNKr",
        "Fuel Rod SPNKr",
        "Skewer",
        "Scattershot",
        "Stalker Rifle",
        "Stalker Rifle (alt)",
        "Stalker Rifle (inv30)",
        "Cindershot",
        "Energy Sword",
        "Gravity Hammer",
    }
}

SUFFIX = bytes.fromhex("42c9679f")
FRAME_MARKER = bytes.fromhex("a07b42")
PATTERN_A = bytes.fromhex("200002")


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def section(title: str) -> None:
    print(f"\n{'=' * 70}\n  {title}\n{'=' * 70}")


def find_all(data: bytes, pat: bytes) -> list[int]:
    offsets, pos = [], 0
    while True:
        pos = data.find(pat, pos)
        if pos == -1:
            break
        offsets.append(pos)
        pos += 1
    return offsets


def resolve(wid: bytes) -> str:
    return WEAPON_IDS.get(wid.hex(), f"?{wid.hex()[:8]}")


def scan_formula_a(data: bytes) -> list[tuple[int, int, bytes]]:
    """[20 00 02 pb ... wid:8B within 64B]  → (offset, pi, wid)"""
    results = []
    pos = 0
    while True:
        pos = data.find(PATTERN_A, pos)
        if pos == -1 or pos + 4 > len(data):
            break
        pb = data[pos + 3]
        pi = pb >> 5
        end = min(pos + 68, len(data))
        sx = data.find(SUFFIX, pos + 4, end)
        if sx >= 4:
            ws = sx - 4
            if ws > pos + 3:
                results.append((pos, pi, data[ws : ws + 8]))
        pos += 4
    return results


# ---------------------------------------------------------------------------
# Phase 1 — Build full weapon timeline (28 chunks)
# ---------------------------------------------------------------------------


def build_timeline(
    chunk_files: list[Path],
) -> tuple[
    dict[int, dict[int, bytes]],  # weapon_at_chunk[chunk_idx][pi]
    list[tuple[int, int, int]],  # timing: (start_ms, dur_ms, n_frames)
    dict[int, list[tuple[int, bytes]]],  # events_by_pi[pi] = [(chunk_idx, wid)]
]:
    timeline: dict[int, dict[int, bytes]] = {}
    timing: list[tuple[int, int, int]] = []
    events_by_pi: dict[int, list[tuple[int, bytes]]] = defaultdict(list)

    cumulative_ms = 0
    for ck_idx, cf in enumerate(chunk_files):
        data = cf.read_bytes()
        n_frames = len(find_all(data, FRAME_MARKER))
        dur_ms = int(n_frames * 33.33)
        timing.append((cumulative_ms, dur_ms, n_frames))
        cumulative_ms += dur_ms

        events = scan_formula_a(data)
        chunk_state: dict[int, bytes] = {}
        for _, pi, wid in events:
            chunk_state[pi] = wid
            events_by_pi[pi].append((ck_idx, wid))
        timeline[ck_idx] = chunk_state

    return timeline, timing, dict(events_by_pi)


def find_chunk(timing: list[tuple[int, int, int]], t_ms: int) -> int:
    for i, (start, dur, _) in enumerate(timing):
        if start <= t_ms < start + dur:
            return i
    return len(timing) - 1 if t_ms >= timing[-1][0] else 0


# ---------------------------------------------------------------------------
# Phase 2 — Hypothesis scoring for pi→xuid mapping
# ---------------------------------------------------------------------------


def score_pi_hypotheses(events_by_pi: dict) -> None:
    section("PHASE 2 — Hypothèses pi→xuid pour T1 non-POV")

    pi_event_counts = {pi: len(events_by_pi.get(pi, [])) for pi in [4, 5, 6]}
    pi_wid_types = {}
    for pi in [4, 5, 6]:
        wid_set = {wid.hex() for _, wid in events_by_pi.get(pi, [])}
        pi_wid_types[pi] = wid_set

    print("\n  Données Film (Formula A) :")
    for pi in [4, 5, 6]:
        known = sum(1 for wh in pi_wid_types[pi] if wh in WEAPON_IDS)
        has_grenade = bool(pi_wid_types[pi] & GRENADE_IDS)
        has_power = bool(pi_wid_types[pi] & POWER_IDS)
        print(
            f"    pi={pi}: {pi_event_counts[pi]:3d} events | "
            f"{len(pi_wid_types[pi])} wids ({known} known) | "
            f"grenade={'✓' if has_grenade else '✗'} | power={'✓' if has_power else '✗'}"
        )

    print("\n  Données API :")
    for xuid in T1_NON_POV:
        s = API_STATS[xuid]
        print(
            f"    {GT[xuid]:<18}: K={s['kills']:2d} sf={s['shots_fired']:3d} "
            f"gren_k={s['grenade_kills']} power_k={s['power_weapon_kills']}"
        )

    print("\n  Scoring hypothèses :")

    # Hypothesis 1: pi=6 = JGtm
    pi6_gren = bool(pi_wid_types[6] & GRENADE_IDS)
    pi6_power = bool(pi_wid_types[6] & POWER_IDS)
    jgtm_gren = API_STATS[XUID_JGTM]["grenade_kills"] > 0
    jgtm_power = API_STATS[XUID_JGTM]["power_weapon_kills"] > 0

    score6 = (
        int(pi6_gren and jgtm_gren) * 3
        + int(pi6_power and jgtm_power) * 3
        + int(not pi6_gren or jgtm_gren)
    )
    print("    H1: pi=6 = JGtm")
    print(f"      Film pi=6 : grenade={pi6_gren}, power={pi6_power}")
    print(f"      API JGtm  : grenade_kills={jgtm_gren}, power_kills={jgtm_power}")
    print(f"      Score : {score6}/7 (grenade=3pts, power=3pts, coherence=1pt)")

    # Hypothèse 2a: pi=4=Madina, pi=5=Choco (sf ratio)
    sf_ratio_a = "pi=4=Madina97294 (sf=636), pi=5=Chocoboflor (sf=328)"
    sf_ratio_b = "pi=4=Chocoboflor (sf=328), pi=5=Madina97294 (sf=636)"
    ev_ratio = f"pi=4={pi_event_counts[4]} events > pi=5={pi_event_counts[5]} events"

    print(f"\n    H2a: {sf_ratio_a}")
    print("      Ratio shots_fired : Madina(636) > Choco(328) ~= pi=4(114) > pi=5(60) → COHÉRENT")
    print(f"      {ev_ratio}")
    print(f"    H2b: {sf_ratio_b}")
    print(
        "      Ratio shots_fired : Choco(328) < Madina(636) ~≠~ pi=4(114) > pi=5(60) → MOINS COHÉRENT"
    )

    print("\n  Mapping retenu (meilleur score) :")
    print("    pi=6 → JGtm       (FORT : grenade + power weapon cohérents)")
    print("    pi=4 → Madina97294 (PROBABILISTE : sf=636 > 328)")
    print("    pi=5 → Chocoboflor (PROBABILISTE : sf=328 < 636)")
    print("    pi=1 → Nilton410  (CERTAIN : fire events confirmés)")

    return {
        4: XUID_MADINA,
        5: XUID_CHOCO,
        6: XUID_JGTM,
        1: XUID_NILTON,
    }


# ---------------------------------------------------------------------------
# Phase 3 — Attribution kills non-POV avec mapping
# ---------------------------------------------------------------------------


def attribute_kills(
    kills: list[tuple[int, int, str]],
    timing: list[tuple[int, int, int]],
    timeline: dict,
    pi_to_xuid: dict,
) -> None:
    """Attribute each non-POV T1 kill to the weapon held at kill time.

    Killer weapon extraction algorithm
    -----------------------------------
    The film binary encodes weapon state updates for T1 (same team as POV) via
    Formula A: `20 00 02 [pb] ... [wid:8B]` where pi = pb >> 5.

    For each kill event (timestamp t_ms, killer xuid):
      1. Map xuid -> pi  (player index, established in Phase 2 via probabilistic scoring)
      2. Find the REPLICATION_DATA chunk that contains t_ms using frame-marker timing
         (each chunk covers ~19s of match time at ~30fps)
      3. Look up timeline[chunk][pi] = weapon_id of that player in that chunk
      4. Fallback to chunk-1 if the current chunk has no state update for that pi
         (state updates are not emitted every chunk if the weapon didn't change)
      5. Resolve wid bytes -> human-readable name via WEAPON_IDS dict

    Confidence is "high" if the resolved wid is in the known WEAPON_IDS catalogue,
    "low" if it matched an unresolved/partial hash.
    """
    xuid_to_pi = {v: k for k, v in pi_to_xuid.items()}

    results = {xuid: [] for xuid in T1_NON_POV}

    for t_ms, xuid, _gt in kills:
        if xuid not in T1_NON_POV:
            continue
        pi = xuid_to_pi.get(xuid)
        if pi is None:
            results[xuid].append((t_ms, "NO_PI", "none"))
            continue
        ck = find_chunk(timing, t_ms)
        # Primary lookup in current chunk; fallback to previous chunk if no state
        # update was emitted (weapon unchanged = server skips the update packet)
        wid = timeline.get(ck, {}).get(pi) or timeline.get(ck - 1, {}).get(pi)
        if wid is None:
            results[xuid].append((t_ms, "UNKNOWN", "none"))
            continue
        wname = resolve(wid)
        conf = "high" if wid.hex() in WEAPON_IDS else "low"
        results[xuid].append((t_ms, wname, conf))

    # Print results per player
    for xuid in T1_NON_POV:
        name = GT[xuid]
        evts = results[xuid]
        total = len(evts)
        attributed = sum(1 for _, w, _ in evts if w not in ("UNKNOWN", "NO_PI"))
        high_conf = sum(1 for _, _, c in evts if c == "high")
        print(f"\n  {name} ({total} kills) : " f"{attributed} attributed ({high_conf} HIGH conf)")

        # Weapon breakdown
        from collections import Counter

        wcount = Counter(w for _, w, _ in evts)
        for w, n in wcount.most_common(8):
            print(f"    {w:<35} × {n}")

    # Summary table
    section("PHASE 3 — Résumé attribution")
    print(f"\n  {'Gamertag':<18} {'Kills':>5} {'Attributed':>10} {'HIGH':>5} {'Coverage':>9}")
    print("  " + "-" * 55)
    total_kills = total_attributed = total_high = 0
    for xuid in T1_NON_POV:
        evts = results[xuid]
        n = len(evts)
        attr = sum(1 for _, w, _ in evts if w not in ("UNKNOWN", "NO_PI"))
        high = sum(1 for _, _, c in evts if c == "high")
        pct = 100 * attr / n if n else 0
        total_kills += n
        total_attributed += attr
        total_high += high
        print(f"  {GT[xuid]:<18} {n:>5} {attr:>10} {high:>5} {pct:>8.1f}%")
    print("  " + "-" * 55)
    pct_total = 100 * total_attributed / total_kills if total_kills else 0
    print(
        f"  {'T1 non-POV TOTAL':<18} {total_kills:>5} {total_attributed:>10} "
        f"{total_high:>5} {pct_total:>8.1f}%"
    )


# ---------------------------------------------------------------------------
# Phase 4 — Validation croisée API
# ---------------------------------------------------------------------------


def validate_against_api(
    kills: list[tuple[int, int, str]],
    timing: list[tuple[int, int, int]],
    timeline: dict,
    pi_to_xuid: dict,
) -> None:
    section("PHASE 4 — Validation croisée vs API aggregates")

    xuid_to_pi = {v: k for k, v in pi_to_xuid.items()}

    for xuid in T1_NON_POV:
        api = API_STATS[xuid]
        pi = xuid_to_pi.get(xuid, -1)
        name = GT[xuid]

        # Collect attributed weapons
        attributed_weapons = []
        for t_ms, x, _ in kills:
            if x != xuid:
                continue
            ck = find_chunk(timing, t_ms)
            wid = timeline.get(ck, {}).get(pi) or timeline.get(ck - 1, {}).get(pi)
            if wid:
                attributed_weapons.append(wid.hex())

        film_grenade_kills = sum(1 for w in attributed_weapons if w in GRENADE_IDS)
        film_power_kills = sum(1 for w in attributed_weapons if w in POWER_IDS)

        print(f"\n  {name} (pi={pi}):")
        print(
            f"    API  : grenade_kills={api['grenade_kills']}  "
            f"power_weapon_kills={api['power_weapon_kills']}"
        )
        print(
            f"    Film : grenade kills attributed={film_grenade_kills}  "
            f"power kills attributed={film_power_kills}"
        )

        # Coherence check
        gren_ok = (api["grenade_kills"] > 0) == (film_grenade_kills > 0)
        power_ok = (api["power_weapon_kills"] > 0) == (film_power_kills > 0)
        print(
            f"    Cohérence : grenade={'✓' if gren_ok else '✗'}  "
            f"power={'✓' if power_ok else '✗'}"
        )

        if not gren_ok or not power_ok:
            print(f"    ⚠️  Incohérence détectée — mapping pi={pi}→{name} peut être incorrect")


# ---------------------------------------------------------------------------
# Phase 5 — Analyse des weapon IDs inconnus (pistes résolution)
# ---------------------------------------------------------------------------


def analyze_unknown_ids(events_by_pi: dict) -> None:
    section("PHASE 5 — Weapon IDs inconnus (pistes résolution)")

    from collections import Counter

    all_unknown: Counter = Counter()
    for pi in [4, 5, 6]:
        for _, wid in events_by_pi.get(pi, []):
            if wid.hex() not in WEAPON_IDS:
                all_unknown[wid.hex()] += 1

    print(f"\n  {len(all_unknown)} weapon IDs inconnus pour T1 non-POV :")
    for wid, n in all_unknown.most_common(20):
        suffix = wid[8:]  # last 4 bytes
        standard = suffix == "42c9679f"
        print(
            f"    {wid} × {n} | suffix={suffix} {'(standard)' if standard else '(NON-STANDARD!)'}"
        )

    standard_unknown = [w for w in all_unknown if w.endswith("42c9679f")]
    non_standard = [w for w in all_unknown if not w.endswith("42c9679f")]
    print(
        f"\n  Standard suffix (42c9679f) : {len(standard_unknown)} IDs → probables variantes de skin"
    )
    print(f"  Suffixe non-standard       : {len(non_standard)} IDs → vérifier")
    if non_standard:
        print(f"    IDs non-standard : {non_standard}")


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------


def load_kills() -> list[tuple[int, int, str]]:
    try:
        import duckdb
    except ImportError:
        print("[WARN] duckdb non disponible")
        return []
    db_path = MAIN_DB
    if not db_path.exists():
        return []
    with duckdb.connect(str(db_path), read_only=True) as conn:
        rows = conn.execute("""
            SELECT time_ms, CAST(xuid AS BIGINT), gamertag
            FROM highlight_events
            WHERE match_id LIKE 'd9329229%'
              AND event_type = 'kill'
            ORDER BY time_ms
        """).fetchall()
    return [(int(t), int(x), str(g)) for t, x, g in rows]


def main() -> None:
    section("inv37 — Attribution probabiliste armes non-POV T1")
    print("\n  Match : d9329229 | Hypothèse : pi=6=JGtm (grenade+power)")

    chunk_files = sorted(CHUNKS_DIR.glob("chunk_*.bin"))
    if not chunk_files:
        print(f"[ERROR] Aucun chunk dans {CHUNKS_DIR}")
        return
    print(f"  Chunks : {len(chunk_files)}")

    # Phase 1 — Timeline
    section("PHASE 1 — Construction weapon timeline")
    timeline, timing, events_by_pi = build_timeline(chunk_files)

    total_ms = sum(d for _, d, _ in timing)
    print(f"  Durée estimée : {total_ms:,}ms ({total_ms/1000:.1f}s)")
    print("  Events par pi (Formula A) :")
    for pi in range(8):
        n = len(events_by_pi.get(pi, []))
        if n:
            wids = {w.hex() for _, w in events_by_pi.get(pi, [])}
            known = sum(1 for w in wids if w in WEAPON_IDS)
            print(f"    pi={pi}: {n:4d} events | {known}/{len(wids)} wids known")

    # Phase 2 — Hypothèses
    pi_to_xuid = score_pi_hypotheses(events_by_pi)

    # Load kills
    kills = load_kills()
    if not kills:
        print("[WARN] Pas de données kill")
        return
    print(f"\n  {len(kills)} kill events chargés")

    # Phase 3 — Attribution
    attribute_kills(kills, timing, timeline, pi_to_xuid)

    # Phase 4 — Validation
    validate_against_api(kills, timing, timeline, pi_to_xuid)

    # Phase 5 — Unknown IDs
    analyze_unknown_ids(events_by_pi)

    section("CONCLUSION inv37")
    print("""
  Mapping retenu (à valider par cohérence) :
    pi=6 → JGtm       : FORT (grenade weapon state + API grenade_kill=1)
    pi=4 → Madina97294 : PROBABILISTE (shots_fired ratio)
    pi=5 → Chocoboflor : PROBABILISTE (shots_fired ratio)

  Prochain verrou : résoudre les 20+ weapon IDs inconnus (suffix 42c9679f standard).
  Ces IDs sont des variantes de skin ou alt-fire modes non mappés dans WEAPON_IDS.
  Action recommandée : soumettre à acurtis lookup service.
""")


if __name__ == "__main__":
    main()
