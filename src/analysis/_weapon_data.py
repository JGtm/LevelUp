"""Données statiques armes Halo Infinite — IDs filmshell et timings.

Séparé de weapon_parser.py pour respecter la limite de 500 lignes par module.
"""

from __future__ import annotations

# ══════════════════════════════════════════════════════════════════════════════
# Weapon IDs (8 bytes, format filmshell Andy Curtis)
# ══════════════════════════════════════════════════════════════════════════════

WEAPON_ID_MAP: dict[bytes, str] = {
    # ══════════════════════════════════════════════════════════════════════
    # CONFIRMÉES (liste maître 2026-03-11)
    # ══════════════════════════════════════════════════════════════════════
    bytes.fromhex("6acdc44d42c9679f"): "Bandit Evo",  # pragma: allowlist secret
    bytes.fromhex("2b1824d542c9679f"): "BR75",  # pragma: allowlist secret  (= BR75 Ranked, même ID)
    bytes.fromhex("230447b142c9679f"): "Cindershot",  # pragma: allowlist secret
    bytes.fromhex("b619d84a42c9679f"): "CQS48 Bulldog",  # pragma: allowlist secret
    bytes.fromhex("84bd29ed42c9679f"): "Disruptor",  # pragma: allowlist secret
    bytes.fromhex("9d6aaed242c9679f"): "Fuel Rod SPNKr",  # pragma: allowlist secret
    bytes.fromhex("841ac5e542c9679f"): "Gravity Hammer",  # pragma: allowlist secret
    bytes.fromhex("2ac9c2ff42c9679f"): "Heatwave",  # pragma: allowlist secret
    bytes.fromhex("71ab0a2c42c9679f"): "M41 SPNKr",  # pragma: allowlist secret
    bytes.fromhex("2fb21c8742c9679f"): "M392 Bandit",  # pragma: allowlist secret
    bytes.fromhex("48c19d2d42c9679f"): "MA40 AR",  # pragma: allowlist secret
    bytes.fromhex(
        "f5c335dfe7232c0f"
    ): "MA5K Avenger",  # pragma: allowlist secret  (NB: tables 1&2 indiquent 0x...0B — à reverifier)
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
    # ══════════════════════════════════════════════════════════════════════
    # NON CONFIRMÉES — grenades (IDs non vérifiés en filmshell)
    # ══════════════════════════════════════════════════════════════════════
    bytes.fromhex("b6dbead842c9679f"): "Frag Grenade",  # pragma: allowlist secret
    bytes.fromhex("c1e1bab042c9679f"): "Plasma Grenade",  # pragma: allowlist secret
    bytes.fromhex("3ad55da442c9679f"): "Dynamo Grenade",  # pragma: allowlist secret
}

# ══════════════════════════════════════════════════════════════════════════════
# Dérivés indexés par uint64 (pour stockage DB et lookups rapides)
# ══════════════════════════════════════════════════════════════════════════════

# IDs sentinelles (réservés, ne collident pas avec les uint64 film)
MELEE_WEAPON_ID: int = 1
GRENADE_WEAPON_ID: int = 0
VEHICLE_WEAPON_ID: int = 2

# bytes → int
WEAPON_BYTES_TO_INT: dict[bytes, int] = {
    k: int.from_bytes(k, byteorder="big") for k in WEAPON_ID_MAP
}

# int → name (pour résolution à l'affichage)
WEAPON_INT_TO_NAME: dict[int, str] = {
    int.from_bytes(k, byteorder="big"): v for k, v in WEAPON_ID_MAP.items()
}

# name → int (reverse — pour migration backfill)
WEAPON_NAME_TO_INT: dict[str, int] = {v: k for k, v in WEAPON_INT_TO_NAME.items()}

# int → set (pour validation rapide)
WEAPON_IDS_INT: set[int] = set(WEAPON_INT_TO_NAME)

# ══════════════════════════════════════════════════════════════════════════════
# Fusion de variantes : nom filmshell → nom canonique
# Appliquer après attribution pour regrouper sous la même citation.
# ══════════════════════════════════════════════════════════════════════════════

WEAPON_FUSION_MAP: dict[str, str] = {
    # Bandit : deux références filmshell pour le même canon
    "M392 Bandit": "Bandit Evo",
    # SPNKr : variante Fuel Rod comptée avec le SPNKr de base
    "Fuel Rod SPNKr": "M41 SPNKr",
}

# Fusion par weapon_id (int → int canonique)
WEAPON_FUSION_MAP_ID: dict[int, int] = {
    WEAPON_NAME_TO_INT[k]: WEAPON_NAME_TO_INT[v]
    for k, v in WEAPON_FUSION_MAP.items()
    if k in WEAPON_NAME_TO_INT and v in WEAPON_NAME_TO_INT
}

# Traductions EN → FR des noms canoniques filmshell
# Utilisé dans l’UI pour afficher les noms localisés.
WEAPON_NAME_FR: dict[str, str] = {
    "Bandit Evo": "Bandit EVO",
    "BR75": "BR75",
    "Cindershot": "Crémateur",
    "CQS48 Bulldog": "CQS48 Bulldog",
    "Disruptor": "Disrupteur",
    "Energy Sword": "Épée à énergie",
    "Gravity Hammer": "Marteau antigravité",
    "Heatwave": "Calcineur",
    "M41 SPNKr": "M41 SPNKr",
    "MA40 AR": "MA40 AR",
    "Mangler": "Déchiqueteur",
    "Mk51 Sidekick": "MK50 Sidekick",
    "MLRS-2 Hydra": "Hydra",
    "Mutilator": "Mutilateur",
    "Needler": "Needler",
    "Plasma Pistol": "Pistolet à plasma",
    "Pulse Carbine": "Carabine à impulsion",
    "Ravager": "Ravageur",
    "S7 Sniper": "S7 Sniper",
    "Sentinel Beam": "Rayon de Sentinelle",
    "Shock Rifle": "Fusil électrique",
    "Skewer": "Empaleur",
    "Stalker Rifle": "Fusil traqueur",
    "VK78 Commando": "VK78 Commando",
    "Vestige Carbine": "Carabine Vestige",
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
    "MA40 AR": (650, 500),
    "BR75": (650, 500),
    "Bandit Evo": (650, 500),
    "M392 Bandit": (650, 500),
    "CQS48 Bulldog": (650, 500),
    "VK78 Commando": (650, 500),
    "Vestige Carbine": (650, 500),
    "Shock Rifle": (650, 500),
    "Shock Rifle (Ranked)": (650, 500),
    "Mangler": (650, 500),
    "Pulse Carbine": (650, 500),
    "Disruptor": (650, 5000),
    "Heatwave": (650, 2000),
    "Needler": (650, 2000),
    "MLRS-2 Hydra": (650, 2000),
    "Ravager": (650, 5000),
    "S7 Sniper": (900, 300),
    "Stalker Rifle": (900, 300),
    "Mutilator": (900, 300),
    "Skewer": (900, 3000),
    "Sentinel Beam": (900, 500),
    "Cindershot": (900, 5000),
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
    "Dynamo Grenade": (950, 1500),
}

# Timing indexé par weapon_id (int) — pour lookups dans weapon_parser
WEAPON_TIMING_BY_ID: dict[int, tuple[int, int]] = {
    WEAPON_NAME_TO_INT[name]: timing
    for name, timing in WEAPON_TIMING.items()
    if name in WEAPON_NAME_TO_INT
}

# ══════════════════════════════════════════════════════════════════════════════
# Résolution weapon_id → nom d'affichage (appelé côté UI uniquement)
# ══════════════════════════════════════════════════════════════════════════════

# IDs à exclure des graphes « armes les plus utilisées »
EXCLUDED_WEAPON_IDS: frozenset[int] = frozenset(
    {MELEE_WEAPON_ID, GRENADE_WEAPON_ID, VEHICLE_WEAPON_ID}
)


def resolve_weapon_display(weapon_id: int | None, lang: str = "fr") -> str | None:
    """Résout un weapon_id en nom d'affichage localisé.

    Chaîne : weapon_id → WEAPON_INT_TO_NAME → WEAPON_FUSION_MAP → WEAPON_NAME_FR.
    Retourne None si weapon_id est None (kill non résolu).
    """
    if weapon_id is None:
        return None
    # Sentinelles
    if weapon_id == MELEE_WEAPON_ID:
        return "Corps à corps" if lang == "fr" else "Melee"
    if weapon_id == GRENADE_WEAPON_ID:
        return "Grenade"
    if weapon_id == VEHICLE_WEAPON_ID:
        return "Véhicule" if lang == "fr" else "Vehicle"
    # Nom filmshell
    name = WEAPON_INT_TO_NAME.get(weapon_id)
    if name is None:
        return f"weapon_{weapon_id}"
    # Fusion variantes
    canonical = WEAPON_FUSION_MAP.get(name, name)
    # Traduction
    if lang == "fr":
        return WEAPON_NAME_FR.get(canonical, canonical)
    return canonical
