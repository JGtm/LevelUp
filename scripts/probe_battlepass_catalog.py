#!/usr/bin/env python3
"""Sonde le catalogue battle pass live et le persiste localement."""

from __future__ import annotations

import argparse
import asyncio
import json
import logging
import sys
from dataclasses import dataclass
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(REPO_ROOT))

from src.data.battlepass_probe import probe_battlepass_catalog

logger = logging.getLogger(__name__)


@dataclass(frozen=True)
class PlayerProfile:
    """Profil joueur minimal requis pour la collecte battle pass."""

    name: str
    db_path: Path
    xuid: str


def main() -> int:
    args = _build_parser().parse_args()
    logging.basicConfig(
        level=logging.DEBUG if args.verbose else logging.INFO,
        format="%(levelname)s %(message)s",
    )

    try:
        profile = _resolve_player_profile(args.player)
    except ValueError as exc:
        logger.error("%s", exc)
        return 2

    result = asyncio.run(
        probe_battlepass_catalog(
            profile.db_path,
            player_name=profile.name,
            xuid=profile.xuid,
            lang=args.lang,
            output_dir=args.output_dir,
        )
    )

    print(json.dumps(_build_summary(result), indent=2, ensure_ascii=False))
    return 0


def _build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description="Sonde tous les battle pass connus pour un joueur et les persiste localement.",
    )
    parser.add_argument(
        "--player",
        help="Nom du joueur tel qu'il apparaît dans db_profiles.json. Défaut: premier profil Halo Infinite.",
    )
    parser.add_argument(
        "--lang",
        default="fr",
        help="Langue de capture logique pour le manifeste local.",
    )
    parser.add_argument(
        "--output-dir",
        help="Dossier de sortie pour les snapshots JSON. Défaut: data/investigation/battlepass/<joueur>/.",
    )
    parser.add_argument(
        "--verbose",
        action="store_true",
        help="Active les logs détaillés.",
    )
    return parser


def _resolve_player_profile(player_name: str | None) -> PlayerProfile:
    profiles = _load_profiles()
    if not profiles:
        raise ValueError("Aucun profil Halo Infinite trouvé dans db_profiles.json.")
    if player_name is None:
        return profiles[0]
    for profile in profiles:
        if profile.name.casefold() == player_name.casefold():
            return profile
    known = ", ".join(profile.name for profile in profiles)
    raise ValueError(f"Joueur introuvable: {player_name}. Profils disponibles: {known}")


def _load_profiles() -> list[PlayerProfile]:
    config_path = REPO_ROOT / "db_profiles.json"
    if not config_path.exists():
        return []
    payload = json.loads(config_path.read_text(encoding="utf-8"))
    profiles_root = payload.get("profiles") or {}
    halo_profiles = profiles_root.get("halo_infinite") or {}
    resolved: list[PlayerProfile] = []
    for player_name, config in halo_profiles.items():
        if not isinstance(config, dict):
            continue
        raw_db_path = str(config.get("db_path") or "").strip()
        xuid = str(config.get("xuid") or "").strip()
        if not raw_db_path or not xuid:
            continue
        resolved.append(
            PlayerProfile(
                name=player_name,
                db_path=(REPO_ROOT / raw_db_path).resolve(),
                xuid=xuid,
            )
        )
    return resolved


def _build_summary(result) -> dict[str, object]:
    return {
        "output_dir": str(result.output_dir),
        "operations_snapshot": str(result.operations_snapshot_path),
        "manifest": str(result.manifest_path),
        "counts": {
            "operations": result.operation_count,
            "player_snapshots_inserted": result.snapshot_count,
            "tracks": result.track_count,
            "items": result.item_count,
            "track_assets": result.track_asset_count,
            "item_assets": result.item_asset_count,
            "currency_assets": result.currency_asset_count,
        },
        "missing_assets": {
            "tracks": list(result.missing_track_assets),
            "items": list(result.missing_item_assets),
            "repo_currencies": list(result.repo_missing_currency_assets),
        },
        "external_currency_paths": list(result.external_currency_paths),
    }



if __name__ == "__main__":
    raise SystemExit(main())
