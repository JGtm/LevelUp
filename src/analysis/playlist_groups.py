"""Groupes de playlists Halo Infinite pour le LUSR.

Ce fichier est conçu pour être facilement modifiable indépendamment
de l'algorithme de rating. Chaque groupe a son propre état TrueSkill
indépendant, permettant un rating précis par contexte de jeu.

Les 6 groupes couvrent les grandes familles de modes de Halo Infinite :
- ranked   (1.00) : Matchmaking élo officiel
- arena    (0.80) : 4v4 compétitif non classé
- tactical (0.70) : Modes précision / one-shot
- btb      (0.60) : 12v12 chaotique
- social   (0.40) : Casual / armes aléatoires
- fun      (0.15) : Party modes

Pour modifier les groupes (préfixes, playlists, poids), éditer ce seul fichier.
L'algorithme LUSR dans skill_rating.py n'a pas à être touché.
"""

from __future__ import annotations

from dataclasses import dataclass


@dataclass(frozen=True)
class PlaylistGroupConfig:
    """Configuration d'un groupe de playlist pour le LUSR."""

    weight_factor: float
    """Facteur multiplicateur sur l'amplitude des mises à jour TrueSkill.
    1.0 = impact plein (ranked), 0.15 = impact minimal (fun).
    """

    pairs_prefixes: tuple[str, ...] = ()
    """Préfixes de pair_name pour auto-détection (ex: 'Ranked:', 'Arena:').
    Vérification : pair_name.startswith(prefix).
    """

    playlist_names: tuple[str, ...] = ()
    """Noms de playlist exacts pour auto-détection complémentaire.
    Utilisés si aucun préfixe pair_name ne correspond.
    """

    label_fr: str = ""
    """Label français pour l'UI."""


# =============================================================================
# Groupes de playlists Halo Infinite
# =============================================================================
#
# Ordre d'évaluation dans get_playlist_group() :
#   1. Correspondance pair_name (préfixes) — prioritaire
#   2. Correspondance playlist_name (noms exacts)
#   3. DEFAULT_PLAYLIST_GROUP ("social")
#
# Clés dict dans l'ordre du plus au moins "sérieux" :
#   ranked → arena → tactical → btb → social → fun
# Cet ordre influe sur la priorité (premier match gagne).

PLAYLIST_GROUPS: dict[str, PlaylistGroupConfig] = {
    # ── Ranked (Compétitif officiel) ─────────────────────────────────────────
    # Tous les modes avec élo officiel Halo. La mise à jour TrueSkill a le
    # poids le plus fort : ces matchs comptent vraiment.
    "ranked": PlaylistGroupConfig(
        weight_factor=1.00,
        pairs_prefixes=("Ranked:",),
        playlist_names=(
            "Ranked Arena",
            "Ranked Slayer",
            "Ranked Team Doubles",
            "HCS",
            "HCS Open",
            "Ranked FFA",
        ),
        label_fr="Classé",
    ),
    # ── Arène (4v4 compétitif non classé) ────────────────────────────────────
    # Modes 4v4 sans élo officiel mais format compétitif. Stratégie et
    # coordination équipe comptent. Poids élevé mais inférieur au ranked.
    "arena": PlaylistGroupConfig(
        weight_factor=0.80,
        pairs_prefixes=("Arena:", "QuickPlay:", "TeamDoubles:"),
        playlist_names=(
            "Quick Play",
            "Squad Battle",
            "Team Doubles",
            "Open Slayer",
            "Team Slayer",
        ),
        label_fr="Arène",
    ),
    # ── Tactique (one-shot / précision) ──────────────────────────────────────
    # Modes où la mécanique de précision prime sur l'endurance (snipers, SWAT).
    # Contexte de jeu très différent → rating indépendant, poids intermédiaire.
    "tactical": PlaylistGroupConfig(
        weight_factor=0.70,
        pairs_prefixes=("Tactical:", "Snipers:"),
        playlist_names=(
            "Tactical Slayer",
            "Team Snipers",
            "Sniper's Paradise",
            "SWAT",
            "Precision Slayer",
        ),
        label_fr="Tactique",
    ),
    # ── Grande équipe (BTB, 12v12) ────────────────────────────────────────────
    # Modes 12v12 chaotiques. La contribution individuelle est diluée dans
    # le chaos de masse → poids modéré, mais rating distinct de l'arène.
    "btb": PlaylistGroupConfig(
        weight_factor=0.60,
        pairs_prefixes=("BTB:", "BigTeam:"),
        playlist_names=(
            "Big Team Battle",
            "Big Team Social",
            "Big Team Heavies",
            "BTB Heavies",
        ),
        label_fr="Grande équipe",
    ),
    # ── Social (casual / armes aléatoires) ───────────────────────────────────
    # Modes où le hasard (Fiesta, Rumble Pit) ou les règles simplifiées
    # réduisent la mesurabilité du skill. Poids faible.
    "social": PlaylistGroupConfig(
        weight_factor=0.40,
        pairs_prefixes=("Fiesta:", "FFA:", "Social:"),
        playlist_names=(
            "Rumble Pit",
            "Fiesta",
            "Super Fiesta",
            "Land Grab",
            "Attrition",
        ),
        label_fr="Social",
    ),
    # ── Fun (party modes) ────────────────────────────────────────────────────
    # Modes de détente avec règles radicalement différentes (Infection,
    # Grifball, Husky Raid). Résultats peu corrélés au skill réel → poids minimal.
    "fun": PlaylistGroupConfig(
        weight_factor=0.15,
        pairs_prefixes=("Infection:", "ActionSack:", "Grifball:", "Husky:"),
        playlist_names=(
            "Infection",
            "Action Sack",
            "Grifball",
            "Husky Raid",
            "Escalation Slayer",
        ),
        label_fr="Fun",
    ),
}

# Groupe assigné si aucun préfixe ni nom de playlist ne correspond.
DEFAULT_PLAYLIST_GROUP: str = "social"


# =============================================================================
# Fonction de détection
# =============================================================================


def get_playlist_group(playlist_name: str | None, pair_name: str | None) -> str:
    """Détermine le groupe de playlist d'un match.

    Priorité :
    1. Correspondance pair_name (préfixes) — la plus fiable
    2. Correspondance playlist_name (nom exact)
    3. Groupe par défaut ("social")

    Args:
        playlist_name: Nom de la playlist (ex: "Quick Play", "Big Team Battle").
        pair_name: Nom de la paire (ex: "Ranked:Slayer on Aquarius").

    Returns:
        Clé du groupe (ex: "ranked", "arena", "tactical", "btb", "social", "fun").
    """
    pn = (pair_name or "").strip()
    pl = (playlist_name or "").strip()

    for group_key, cfg in PLAYLIST_GROUPS.items():
        # 1. Vérification des préfixes de pair_name (prioritaire)
        for prefix in cfg.pairs_prefixes:
            if pn.startswith(prefix):
                return group_key
        # 2. Vérification des noms de playlist exacts
        if pl in cfg.playlist_names:
            return group_key

    return DEFAULT_PLAYLIST_GROUP
