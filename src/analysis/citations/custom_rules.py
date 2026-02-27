"""
Règles de calcul custom pour citations H5G complexes.

Ce module contient les fonctions de calcul pour les citations qui nécessitent
une logique métier complexe (filtres multiples, conditions, séquences, etc.).
"""

from typing import Any

import polars as pl


def compute_bulldozer(df: pl.DataFrame) -> int:
    """Compte les parties Slayer/Assassin avec KDA > 8 (hors Firefight/BTB).

    Filtre sur playlist_name OU game_variant_name pour couvrir les Quick Play
    qui contiennent des variants Slayer.

    Args:
        df: DataFrame du match avec colonnes playlist_name, game_variant_name, kda, etc.

    Returns:
        1 si la condition est validée, 0 sinon.
    """
    if df.is_empty():
        return 0

    # Vérifier que le match est Slayer/Assassin via playlist_name OU game_variant_name
    slayer_pattern = "(?i)slayer|assassin"
    exclude_pattern = "(?i)firefight|btb|baptême|bapteme|big team|grande bataille"

    has_playlist = "playlist_name" in df.columns
    has_variant = "game_variant_name" in df.columns

    # Filtrer : (playlist OU variant match slayer) ET PAS firefight/btb
    conditions = []
    if has_playlist:
        conditions.append(pl.col("playlist_name").str.contains(slayer_pattern))
    if has_variant:
        conditions.append(pl.col("game_variant_name").str.contains(slayer_pattern))

    if not conditions:
        return 0

    slayer_filter = conditions[0]
    for cond in conditions[1:]:
        slayer_filter = slayer_filter | cond

    exclude_conditions = []
    if has_playlist:
        exclude_conditions.append(pl.col("playlist_name").str.contains(exclude_pattern))
    if has_variant:
        exclude_conditions.append(pl.col("game_variant_name").str.contains(exclude_pattern))

    if exclude_conditions:
        exclude_filter = exclude_conditions[0]
        for cond in exclude_conditions[1:]:
            exclude_filter = exclude_filter | cond
        slayer_filter = slayer_filter & ~exclude_filter

    filtered = df.filter(slayer_filter)

    if filtered.is_empty():
        return 0

    # KDA > 8 (utiliser la colonne kda si disponible, sinon calculer)
    if "kda" in filtered.columns:
        count = filtered.filter(pl.col("kda") > 8.0).height
    else:
        count = filtered.filter((pl.col("kills") / pl.col("deaths").clip(1, None)) > 8.0).height

    return count


def compute_wins_mode(df: pl.DataFrame, mode_pattern: str) -> int:
    """Compte les victoires dans un mode donné.

    Args:
        df: DataFrame des matchs
        mode_pattern: Pattern regex pour le mode (ex: "ctf|drapeau")

    Returns:
        Nombre de victoires
    """
    if df.is_empty():
        return 0

    return df.filter(
        pl.col("playlist_name").str.contains(f"(?i){mode_pattern}") & pl.col("outcome").eq(2)
    ).height


def compute_wins_ctf(df: pl.DataFrame) -> int:
    """Victoires en Capture du drapeau."""
    return compute_wins_mode(df, "ctf|capture.*drapeau|drapeau.*neutre|neutral.*flag")


def compute_wins_firefight(df: pl.DataFrame) -> int:
    """Victoires en Firefight/Baptême du feu."""
    return compute_wins_mode(df, "firefight|baptême|bapteme")


def compute_wins_slayer(df: pl.DataFrame) -> int:
    """Victoires en Slayer/Assassin."""
    return compute_wins_mode(df, "slayer|assassin")


def compute_wins_strongholds(df: pl.DataFrame) -> int:
    """Victoires en Strongholds/Bases."""
    return compute_wins_mode(df, "stronghold|bases")


def compute_annexion_forcee(
    df: pl.DataFrame | None = None, awards: dict[str, int] | None = None, **kwargs: Any
) -> int:
    """Compte les séquences de 3+ captures de zone sans mourir.

    Condition : Capturer 3 zones d'affilée dans un match Strongholds sans mourir entre.

    ⚠️ EXTRAPOLATION : Cette implémentation est une approximation basée sur les
    highlight_events (mode + death). Les events "mode" correspondent aux captures
    de zone (Zone capturée + Zone sécurisée), mais l'API Halo ne fournit pas
    directement la citation "Annexion forcée". Le résultat peut donc différer
    légèrement de la vraie valeur in-game.

    Args:
        df: DataFrame du match (non utilisé directement).
        awards: Dict des compteurs d'awards.
        **kwargs: Peut contenir ``highlight_events`` — liste de tuples
            ``(time_ms, event_type)`` triés par temps pour ce joueur dans ce match.

    Returns:
        Nombre de séquences de 3+ captures sans mourir (estimation).
    """
    highlight_events: list[tuple[int, str]] | None = kwargs.get("highlight_events")

    if highlight_events is not None:
        # Mode "précis" via highlight_events — reste une extrapolation :
        # Les events "mode" = captures de zone, mais l'API ne distingue pas
        # les types d'objectifs (zone vs drapeau vs autre). Fiabilité ~90%.
        streak = 0
        count = 0
        for _time_ms, event_type in highlight_events:
            if event_type == "mode":
                streak += 1
                if streak >= 3 and streak % 3 == 0:
                    count += 1
            elif event_type == "death":
                streak = 0
        return count

    # Fallback : approximation grossière par awards (encore moins fiable)
    if awards is None:
        return 0

    zone_captures = (
        awards.get("zone_captured", 0)
        # Legacy fallback (données pré-migration)
        or awards.get("Zone capturée", 0)
        or awards.get("Zone Capture", 0)
    )

    if zone_captures < 3:
        return 0

    return zone_captures // 3


def compute_flag_em_down(
    df: pl.DataFrame | None = None, awards: dict[str, int] | None = None, **kwargs: Any
) -> int:
    """Compte les kills de porteur de drapeau ennemi.

    Condition : Tuer un ennemi portant le drapeau.

    Args:
        df: DataFrame des matchs (non utilisé)
        awards: Dict des compteurs d'awards
        **kwargs: Arguments supplémentaires

    Returns:
        Nombre de Flag Carrier Kills
    """
    if awards is None:
        return 0
    return (
        awards.get("runner_stopped", 0)
        # Legacy fallback (données pré-migration)
        + awards.get("Porteur arrêté", 0)
        + awards.get("Flag Carrier Kill", 0)
        + awards.get("Flag Carrier Killed", 0)
    )


def compute_hijack(
    df: pl.DataFrame | None = None, awards: dict[str, int] | None = None, **kwargs: Any
) -> int:
    """Compte le nombre total de hijacks de véhicules.

    Args:
        df: DataFrame des matchs (non utilisé)
        awards: Dict des compteurs d'awards
        **kwargs: Arguments supplémentaires

    Returns:
        Nombre total de hijacks
    """
    if awards is None:
        return 0

    total = 0
    for key, val in awards.items():
        # Technical IDs: hijacked_banshee, hijacked_ghost, etc.
        if key.startswith("hijacked_") or any(
            kw in key.lower() for kw in ("hijacked", "hijack", "skyjack")
        ):
            total += val
    return total


def compute_vandalism(
    df: pl.DataFrame | None = None, awards: dict[str, int] | None = None, **kwargs: Any
) -> int:
    """Compte le nombre de véhicules ennemis détruits.

    Args:
        df: DataFrame des matchs (non utilisé)
        awards: Dict des compteurs d'awards
        **kwargs: Arguments supplémentaires

    Returns:
        Nombre de véhicules détruits
    """
    if awards is None:
        return 0

    total = 0
    for key, val in awards.items():
        # Technical IDs: destroyed_banshee, destroyed_ghost, etc.
        if key.startswith("destroyed_") or any(
            kw in key.lower() for kw in ("destroyed", "destruction")
        ):
            total += val
    return total


def compute_wraith_destroyer(
    df: pl.DataFrame | None = None, awards: dict[str, int] | None = None, **kwargs: Any
) -> int:
    """Compte les Wraiths détruits."""
    if awards is None:
        return 0
    return (
        awards.get("destroyed_wraith", 0)
        # Legacy fallback
        + awards.get("Wraith Destroyed", 0)
        + awards.get("Apparition Destroyed", 0)
    )


def compute_mongoose_destroyer(
    df: pl.DataFrame | None = None, awards: dict[str, int] | None = None, **kwargs: Any
) -> int:
    """Compte les Mongooses détruits."""
    if awards is None:
        return 0
    return (
        awards.get("destroyed_mongoose", 0)
        # Legacy fallback
        + awards.get("Mongoose Destroyed", 0)
    )


def compute_warthog_destroyer(
    df: pl.DataFrame | None = None, awards: dict[str, int] | None = None, **kwargs: Any
) -> int:
    """Compte les Warthogs détruits."""
    if awards is None:
        return 0
    return (
        awards.get("destroyed_warthog", 0)
        + awards.get("destroyed_rocket_warthog", 0)
        # Legacy fallback
        + awards.get("Warthog Destroyed", 0)
        + awards.get("Rocket Warthog Destroyed", 0)
    )


# Registry des fonctions custom pour utilisation dynamique
CUSTOM_FUNCTIONS = {
    "compute_bulldozer": compute_bulldozer,
    "compute_wins_ctf": compute_wins_ctf,
    "compute_wins_firefight": compute_wins_firefight,
    "compute_wins_slayer": compute_wins_slayer,
    "compute_wins_strongholds": compute_wins_strongholds,
    "compute_annexion_forcee": compute_annexion_forcee,
    "compute_flag_em_down": compute_flag_em_down,
    "compute_hijack": compute_hijack,
    "compute_vandalism": compute_vandalism,
    "compute_wraith_destroyer": compute_wraith_destroyer,
    "compute_mongoose_destroyer": compute_mongoose_destroyer,
    "compute_warthog_destroyer": compute_warthog_destroyer,
}


def get_custom_function(function_name: str):
    """Récupère une fonction custom par son nom.

    Args:
        function_name: Nom de la fonction

    Returns:
        La fonction ou None si non trouvée
    """
    return CUSTOM_FUNCTIONS.get(function_name)
