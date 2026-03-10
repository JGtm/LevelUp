"""Données statiques armes Halo Infinite — IDs filmshell et timings.

Séparé de weapon_parser.py pour respecter la limite de 500 lignes par module.
"""

from __future__ import annotations

# ══════════════════════════════════════════════════════════════════════════════
# Weapon IDs (8 bytes, format filmshell Andy Curtis)
# ══════════════════════════════════════════════════════════════════════════════

WEAPON_ID_MAP: dict[bytes, str] = {
    # ── Armes confirmées (liste maître 2026-03-10) ───────────────────────
    bytes.fromhex("6acdc44d42c9679f"): "Bandit Evo",  # pragma: allowlist secret
    bytes.fromhex("2b1824d542c9679f"): "BR75",  # pragma: allowlist secret
    bytes.fromhex("230447b142c9679f"): "Cindershot",  # pragma: allowlist secret
    bytes.fromhex("b619d84a42c9679f"): "CQS48 Bulldog",  # pragma: allowlist secret
    bytes.fromhex("84bd29ed42c9679f"): "Disruptor",  # pragma: allowlist secret
    bytes.fromhex("9d6aaed242c9679f"): "Fuel Rod SPNKr",  # pragma: allowlist secret
    bytes.fromhex("841ac5e542c9679f"): "Gravity Hammer",  # pragma: allowlist secret
    bytes.fromhex("2ac9c2ff42c9679f"): "Heatwave",  # pragma: allowlist secret
    bytes.fromhex("71ab0a2c42c9679f"): "M41 SPNKr",  # pragma: allowlist secret
    bytes.fromhex("2fb21c8742c9679f"): "M392 Bandit",  # pragma: allowlist secret
    bytes.fromhex("48c19d2d42c9679f"): "MA40 AR",  # pragma: allowlist secret
    bytes.fromhex("f5c335dfe7232c0b"): "MA5K Avenger",  # pragma: allowlist secret
    bytes.fromhex("80977ba542c9679f"): "Mangler",  # pragma: allowlist secret
    bytes.fromhex("767db96d42c9679f"): "MLRS-2 Hydra",  # pragma: allowlist secret
    bytes.fromhex("f408190f42c9679f"): "Mk51 Sidekick",  # pragma: allowlist secret
    bytes.fromhex("d791556542c9679f"): "Mutilator",  # pragma: allowlist secret
    bytes.fromhex("b7262ca1c8fb11d0"): "Mythic Sandwich",  # pragma: allowlist secret
    bytes.fromhex("b533957e42c9679f"): "Needler",  # pragma: allowlist secret
    bytes.fromhex("c354294642c9679f"): "Plasma Pistol",  # pragma: allowlist secret
    bytes.fromhex("30484ea642c9679f"): "Pulse Carbine",  # pragma: allowlist secret
    bytes.fromhex("c30d87c742c9679f"): "Ravager",  # pragma: allowlist secret
    bytes.fromhex("0a1992bc42c9679f"): "S7 Sniper",  # pragma: allowlist secret
    bytes.fromhex("880fe0bc42c9679f"): "Sandwich",  # pragma: allowlist secret
    bytes.fromhex("a0955e9e42c9679f"): "Sentinel Beam",  # pragma: allowlist secret
    bytes.fromhex("9387a8b942c9679f"): "Shock Rifle",  # pragma: allowlist secret
    bytes.fromhex("1a22fee642c9679f"): "Shock Rifle (Ranked)",  # pragma: allowlist secret
    bytes.fromhex("0d20c46942c9679f"): "Skewer",  # pragma: allowlist secret
    bytes.fromhex("daf193c742c9679f"): "Stalker Rifle",  # pragma: allowlist secret
    bytes.fromhex("3e07021742c9679f"): "Vestige Carbine",  # pragma: allowlist secret
    bytes.fromhex("fd98554c42c9679f"): "VK78 Commando",  # pragma: allowlist secret
    # ── Energy Sword family ───────────────────────────────────────────────
    bytes.fromhex("4ff3937e42c9679f"): "Energy Sword",  # pragma: allowlist secret
    bytes.fromhex("4ff3937e8978aa7a"): "Duelist Energy Sword",  # pragma: allowlist secret
    bytes.fromhex("4ff3937e1ec48c7a"): "Elite Bloodblade",  # pragma: allowlist secret
    bytes.fromhex("0c55765f7a9376a0"): "Infected Energy Sword",  # pragma: allowlist secret
    # ── Gravity Hammer family ─────────────────────────────────────────────
    bytes.fromhex("841ac5e5a730e49f"): "Diminisher of Hope",  # pragma: allowlist secret
    bytes.fromhex("841ac5e5d8d07ca1"): "Rushdown Hammer",  # pragma: allowlist secret
    # ── Grenades (Formula A / état inventaire) ────────────────────────────
    bytes.fromhex("b6dbead842c9679f"): "Frag Grenade",  # pragma: allowlist secret
    bytes.fromhex("c1e1bab042c9679f"): "Plasma Grenade",  # pragma: allowlist secret
    bytes.fromhex("6683257c42c9679f"): "Spike Grenade",  # pragma: allowlist secret
    bytes.fromhex("3ad55da442c9679f"): "Dynamo Grenade",  # pragma: allowlist secret
    bytes.fromhex("ab879f1e42c9679f"): "Spike Grenade (alt)",  # pragma: allowlist secret
    bytes.fromhex("e3a0a51842c9679f"): "Dynamo Grenade (alt)",  # pragma: allowlist secret
    bytes.fromhex("18e1fea042c9679f"): "Dynamo Grenade (proj)",  # pragma: allowlist secret
    # ── Variantes / skins (suffixe standard confirmé) ────────────────────
    bytes.fromhex("7e53b3c642c9679f"): "Pulse Carbine (alt)",  # pragma: allowlist secret
    bytes.fromhex("04e7f00b42c9679f"): "Plasma Pistol (alt)",  # pragma: allowlist secret
    bytes.fromhex("3d34488542c9679f"): "Heatwave (alt)",  # pragma: allowlist secret
    bytes.fromhex("f5ef3bdb42c9679f"): "Stalker Rifle (alt)",  # pragma: allowlist secret
    bytes.fromhex("fcc6aa7642c9679f"): "Shock Rifle (alt)",  # pragma: allowlist secret
    bytes.fromhex("7deb133f42c9679f"): "Mangler (alt)",  # pragma: allowlist secret
    bytes.fromhex("cb30ec5e42c9679f"): "Disruptor (alt)",  # pragma: allowlist secret
    bytes.fromhex("2b1d61e442c9679f"): "Ravager (alt)",  # pragma: allowlist secret
    bytes.fromhex("7a11aeef42c9679f"): "Skewer (alt)",  # pragma: allowlist secret
    bytes.fromhex("c2a6d5e042c9679f"): "Cindershot (alt)",  # pragma: allowlist secret
    bytes.fromhex("1f6ae65542c9679f"): "MLRS-2 Hydra (alt)",  # pragma: allowlist secret
    bytes.fromhex("6d32c7dc42c9679f"): "Dynamo Grenade (state)",  # pragma: allowlist secret
}

# ══════════════════════════════════════════════════════════════════════════════
# Médailles indiquant un kill melee ou grenade
# ══════════════════════════════════════════════════════════════════════════════

MELEE_MEDALS: frozenset[str] = frozenset(
    {"Pummel", "Assassination", "Back Smack", "Melee", "Quigley"}
)

GRENADE_MEDALS: frozenset[str] = frozenset(
    {"Sticky Fingers", "Grenadier", "Boom!", "Kong", "Stick", "Grenade Stick"}
)

# ══════════════════════════════════════════════════════════════════════════════
# Timing par classe d'arme : (swap_ms, travel_max_ms)
# swap_ms      = délai physique minimum pour changer d'arme avant le kill
# travel_max_ms = délai max projectile/blast pour compter comme ce kill
# ══════════════════════════════════════════════════════════════════════════════

WEAPON_TIMING: dict[str, tuple[int, int]] = {
    "Mk51 Sidekick": (400, 300),
    "Plasma Pistol": (450, 300),
    "Plasma Pistol (alt)": (450, 300),
    "MA40 AR": (650, 500),
    "BR75": (650, 500),
    "Bandit Evo": (650, 500),
    "M392 Bandit": (650, 500),
    "M392 Bandit (alt)": (650, 500),
    "CQS48 Bulldog": (650, 500),
    "VK78 Commando": (650, 500),
    "Vestige Carbine": (650, 500),
    "Shock Rifle": (650, 500),
    "Shock Rifle (Ranked)": (650, 500),
    "Shock Rifle (alt)": (650, 500),
    "Mangler": (650, 500),
    "Mangler (alt)": (650, 500),
    "Pulse Carbine": (650, 500),
    "Pulse Carbine (alt)": (650, 500),
    "Disruptor": (650, 5000),
    "Disruptor (alt)": (650, 5000),
    "Heatwave": (650, 2000),
    "Heatwave (alt)": (650, 2000),
    "Needler": (650, 2000),
    "MLRS-2 Hydra": (650, 2000),
    "MLRS-2 Hydra (alt)": (650, 2000),
    "Ravager": (650, 5000),
    "Ravager (alt)": (650, 5000),
    "S7 Sniper": (900, 300),
    "Stalker Rifle": (900, 300),
    "Stalker Rifle (alt)": (900, 300),
    "Mutilator": (900, 300),
    "Skewer": (900, 3000),
    "Skewer (alt)": (900, 3000),
    "Sentinel Beam": (900, 500),
    "Cindershot": (900, 5000),
    "Cindershot (alt)": (900, 5000),
    "M41 SPNKr": (1100, 2000),
    "Fuel Rod SPNKr": (1100, 2000),
    "Gravity Hammer": (1100, 1400),
    "Diminisher of Hope": (1100, 1400),
    "Rushdown Hammer": (1100, 1400),
    "Energy Sword": (1100, 1400),
    "Duelist Energy Sword": (1100, 1400),
    "Elite Bloodblade": (1100, 1400),
    "Infected Energy Sword": (1100, 1400),
    "Frag Grenade": (950, 1650),
    "Plasma Grenade": (950, 1350),
    "Spike Grenade": (950, 1550),
    "Spike Grenade (alt)": (950, 1550),
    "Dynamo Grenade": (950, 1500),
    "Dynamo Grenade (alt)": (950, 1500),
    "Dynamo Grenade (proj)": (950, 1500),
}
