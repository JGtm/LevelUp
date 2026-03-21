"""
Comparaison résultats acurtis vs notre parser — match 147ffd4d-3d1d-4b90-a46d-5570009f8c36

Reproduit le tableau acurtis (Shots Fired : Stats API | Film | Weapons) avec notre
propre algorithme de parsing des chunks binaires.

Colonnes :
  Gamertag  — nom du joueur
  Stats     — shots_fired extrait de l'API (match_participants)
  Film      — fire events détectés dans les chunks (NOTE : total match, voir ci-dessous)
  Weapons   — armes distinctes via Formula A Section 1 (par joueur)

NOTE Film : scan_fire_events(pi=1) capture TOUS les fire events du match.
  Somme acurtis = 1178, notre pi=1 = 1177 (diff = 1, JGtm absent de notre sync).
  La colonne Film affiche donc le total match pour le joueur dont pi=1 est résolu,
  et 0 pour les autres — non comparable individuellement avec acurtis.

Usage :
  python scripts/experimental/compare_acurtis_147ffd4d.py
"""

from __future__ import annotations

import sys
from collections import defaultdict
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT))

from rich.console import Console
from rich.table import Table

from src.analysis._weapon_data import WEAPON_INT_TO_NAME
from src.analysis.packet_index import (
    detect_pi_from_metadata,
    extract_metadata_payload,
    index_chunk,
)
from src.analysis.player_index import detect_player_indices
from src.analysis.weapon_parser import scan_fire_events, scan_formula_a
from src.utils.db import duckdb_read_only

# ── Constantes ────────────────────────────────────────────────────────────────

MATCH_ID = "147ffd4d-3d1d-4b90-a46d-5570009f8c36"
SHARED_DB = ROOT / "data" / "warehouse" / "shared_matches.duckdb"
CHUNKS_DIR = ROOT / "data" / "investigation" / "chunks" / MATCH_ID[:8]
APPROX_CHUNK_MS = 20_000

console = Console()


# ── Helpers ───────────────────────────────────────────────────────────────────


def load_participants(match_id: str) -> list[dict]:
    """Charge xuid, gamertag, shots_fired depuis match_participants.

    Résout le gamertag via (dans l'ordre) : match_participants → highlight_events
    → xuid_aliases.
    """
    with duckdb_read_only(SHARED_DB) as con:
        rows = con.execute(
            "SELECT mp.xuid, mp.gamertag, mp.shots_fired "
            "FROM match_participants mp WHERE mp.match_id = ?",
            [match_id],
        ).fetchall()

        # v6 : gamertag supprimé de highlight_events — résolution via xuid_aliases
        gt_he = con.execute(
            "SELECT DISTINCT xuid, gamertag FROM xuid_aliases WHERE gamertag IS NOT NULL",
        ).fetchall()

        gt_xa = con.execute(
            "SELECT DISTINCT xuid, gamertag FROM xuid_aliases",
        ).fetchall()

    xuid_to_gt_he = {str(r[0]): r[1] for r in gt_he}
    xuid_to_gt_xa = {str(r[0]): r[1] for r in gt_xa}

    participants = []
    for xuid, gamertag, shots_fired in rows:
        xuid_str = str(xuid)
        resolved_gt = (
            gamertag
            or xuid_to_gt_he.get(xuid_str)
            or xuid_to_gt_xa.get(xuid_str)
            or f"xuid:{xuid}"
        )
        participants.append(
            {
                "xuid": xuid_str,
                "gamertag": resolved_gt,
                "shots_fired_api": shots_fired or 0,
            }
        )
    return participants


def load_chunks(chunks_dir: Path) -> list[tuple[int, bytes]]:
    """Charge les chunks binaires triés par index."""
    result = []
    for p in sorted(chunks_dir.glob("chunk_*.bin")):
        try:
            idx = int(p.stem.split("_")[1])
            result.append((idx, p.read_bytes()))
        except (ValueError, IndexError):
            continue
    return result


def resolve_player_indices(
    chunks: list[tuple[int, bytes]],
    participants: list[dict],
) -> dict[str, int]:
    """Résout xuid_str → player_index via PLAYER_METADATA puis fallback acurtis."""
    xuid_int_map = {
        int(p["xuid"]): p["xuid"]
        for p in participants
        if p["xuid"].isdigit()
    }
    if not xuid_int_map or not chunks:
        return {}

    first_data = chunks[0][1]
    result: dict[int, int] = {}  # xuid_int → pi

    # Méthode 1 — PLAYER_METADATA packet
    packets = index_chunk(first_data)
    metadata = extract_metadata_payload(first_data, packets)
    if metadata is not None:
        pi_to_xuid = detect_pi_from_metadata(metadata, xuid_int_map)
        for pi, xuid_int in pi_to_xuid.items():
            result[xuid_int] = pi

    # Méthode 2 — fallback acurtis (bitstring XUID scan)
    missing = {xi: xs for xi, xs in xuid_int_map.items() if xi not in result}
    if missing:
        for _, chunk_data in chunks:
            pi_to_xuid = detect_player_indices(chunk_data, missing)
            for pi, xuid_int in pi_to_xuid.items():
                result[xuid_int] = pi
            missing = {xi: xs for xi, xs in missing.items() if xi not in result}
            if not missing:
                break

    # Inverser : xuid_str → pi
    return {str(xi): pi for xi, pi in result.items()}


def scan_fire_events_all_players(
    chunks: list[tuple[int, bytes]],
    n_players: int = 8,
) -> dict[int, list[dict]]:
    """Scanne les fire events pour tous les player_index sur tous les chunks."""
    events_by_pi: dict[int, list[dict]] = defaultdict(list)

    for chunk_idx, chunk_data in chunks:
        chunk_start_ms = chunk_idx * APPROX_CHUNK_MS
        packets = index_chunk(chunk_data)

        for pi in range(n_players):
            evs = scan_fire_events(
                chunk_data, pi, chunk_start_ms, APPROX_CHUNK_MS, packets=packets
            )
            for ev in evs:
                ev["player_index"] = pi
                ev["chunk_idx"] = chunk_idx
            events_by_pi[pi].extend(evs)

    return dict(events_by_pi)


def scan_weapons_formula_a(
    chunks: list[tuple[int, bytes]],
) -> dict[int, set[str]]:
    """Extrait les armes vues par joueur via Formula A (Section 1 snapshots).

    Returns: {player_index: {weapon_name, ...}}
    """
    weapons_by_pi: dict[int, set[str]] = {}
    for _chunk_idx, chunk_data in chunks:
        for _offset, pi, wid_bytes in scan_formula_a(chunk_data):
            name = WEAPON_INT_TO_NAME.get(int.from_bytes(wid_bytes, "big"))
            if name:
                weapons_by_pi.setdefault(pi, set()).add(name)
    return weapons_by_pi


def extract_weapons(fire_events: list[dict]) -> list[str]:
    """Retourne les noms d'armes distinctes triés depuis les fire events."""
    names: set[str] = set()
    for ev in fire_events:
        wb = ev.get("weapon_bytes")
        if wb:
            wint = int.from_bytes(wb, "big")
            name = WEAPON_INT_TO_NAME.get(wint) or ev.get("weapon_name")
            if name and not name.startswith("INCONNU"):
                names.add(name)
    return sorted(names)


# ── Main ──────────────────────────────────────────────────────────────────────


def main() -> None:
    console.print(f"\n[bold cyan]Match :[/] {MATCH_ID}")
    console.print(f"[bold cyan]Chunks :[/] {CHUNKS_DIR}\n")

    # 1. Participants
    participants = load_participants(MATCH_ID)
    if not participants:
        console.print("[red]Aucun participant trouvé dans match_participants.[/]")
        return
    console.print(f"[green]{len(participants)} participants chargés.[/]")

    # 2. Chunks binaires
    chunks = load_chunks(CHUNKS_DIR)
    if not chunks:
        console.print(f"[red]Aucun chunk trouvé dans {CHUNKS_DIR}[/]")
        return
    console.print(f"[green]{len(chunks)} chunks chargés (idx {chunks[0][0]}–{chunks[-1][0]}).[/]")

    # 3. Résolution player_index
    xuid_to_pi = resolve_player_indices(chunks, participants)
    n_resolved = len(xuid_to_pi)
    console.print(f"[green]{n_resolved}/{len(participants)} player_index résolus.[/]\n")

    # Diagnostic résolution
    for p in participants:
        pi = xuid_to_pi.get(p["xuid"], "?")
        console.print(f"  {p['gamertag']:20s}  xuid={p['xuid']}  pi={pi}")
    console.print()

    # 4. Scan fire events (match-level, tout via pi=1)
    events_by_pi = scan_fire_events_all_players(chunks)
    all_fire_events = [ev for evs in events_by_pi.values() for ev in evs]
    total_film = len(all_fire_events)
    match_weapons = extract_weapons(all_fire_events)

    # 5. Weapons par joueur via Formula A (Section 1) — inconnus en Fiesta
    weapons_by_pi_fa = scan_weapons_formula_a(chunks)

    # 6. Construction tableau
    title = (
        f"[bold]Slayer:Arena Super Fiesta on Bazaar[/]\n"
        f"[dim]{MATCH_ID}[/]\n"
        f"[bold]Shots Fired[/]"
    )
    table = Table(title=title, show_header=True, header_style="bold magenta")
    table.add_column("Gamertag", style="cyan", no_wrap=True)
    table.add_column("Stats", justify="right")
    table.add_column("Film", justify="right")
    table.add_column("Weapons")

    for p in sorted(participants, key=lambda x: x["gamertag"]):
        xuid = p["xuid"]
        pi = xuid_to_pi.get(xuid)
        film_events = events_by_pi.get(pi, []) if pi is not None else []
        film_count = len(film_events)

        # Weapons : fire events si pi=1 (= match total) ; Formula A sinon (peut être vide)
        if pi == 1 and film_events:
            weapons = sorted(extract_weapons(film_events))
            weapons_note = " [dim](match total)[/]"
        else:
            fa = sorted(weapons_by_pi_fa.get(pi, set())) if pi is not None else []
            weapons = fa
            weapons_note = "" if fa else ""

        weapons_str = ", ".join(weapons) + weapons_note if weapons else ""

        table.add_row(
            p["gamertag"],
            str(p["shots_fired_api"]),
            str(film_count) if film_count else "0",
            weapons_str,
        )

    # Ligne JGtm manquant
    table.add_row(
        "[dim]JGtm (absent sync)[/]",
        "[dim]383[/]",
        "[dim]?[/]",
        "[dim]—[/]",
    )

    console.print(table)

    # 7. Note sur le Film total
    console.print(
        f"\n[bold yellow]Film total match[/] : {total_film} fire events "
        f"(tous joueurs confondus via pi=1)\n"
        f"[dim]Σ acurtis = 1178, notre pi=1 = {total_film} "
        f"(différence = {1178 - total_film})[/]"
    )

    # 8. Note Fiesta / Formula A
    if weapons_by_pi_fa:
        fa_unknown = sum(
            1 for pi_weps in weapons_by_pi_fa.values()
            for name in pi_weps if name and len(name) == 16  # hex non résolu
        )
        console.print(
            f"[dim]Formula A : {sum(len(v) for v in weapons_by_pi_fa.values())} snapshots "
            f"détectés, {fa_unknown} avec weapon_id inconnu (IDs Fiesta non mappés)[/]"
        )

    # 9. Divergences
    console.print("\n[bold red]Divergences vs acurtis :[/]")
    console.print(
        "  1. [yellow]Film non attribuable par joueur[/] — scan_fire_events(pi=1) "
        "est un scanner match-level. Le marqueur _build_marker(pi=1) correspond"
        " à un bit structurel commun à tous les fire events Section 2."
    )
    console.print(
        "  2. [yellow]Weapons Fiesta non mappées[/] — Formula A retourne des IDs "
        "hors WEAPON_ID_MAP (spawn aléatoire Fiesta). Les fire events (ci-dessus, "
        "match total) retournent les bons noms."
    )
    console.print(
        "  3. [yellow]JGtm absent[/] — xuid:2533274823110022 non présent dans "
        "match_participants → sync incomplet pour ce match."
    )
    console.print(
        "\n[dim]→ Action Phase 0 : investiguer l'encodage player_index "
        "dans la Section 2 du filmshell[/]"
    )


if __name__ == "__main__":
    main()
