"""Parsing des arguments CLI pour le backfill.

Ce module centralise la définition de tous les arguments CLI,
séparant la logique de parsing de la logique métier.
"""

from __future__ import annotations

import argparse


def create_argument_parser() -> argparse.ArgumentParser:
    """Crée le parser d'arguments pour le CLI backfill.

    Returns:
        Parser configuré avec tous les arguments.
    """
    parser = argparse.ArgumentParser(
        description="Backfill des données manquantes pour DuckDB",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog=_get_usage_examples(),
    )

    # ── Sélection du joueur ──
    parser.add_argument(
        "--player",
        type=str,
        default=None,
        help="Gamertag du joueur (ignoré si --all)",
    )
    parser.add_argument(
        "--all",
        action="store_true",
        help="Traiter tous les joueurs DuckDB",
    )

    # ── Options générales ──
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Mode dry-run (ne fait que lister les matchs sans données)",
    )
    parser.add_argument(
        "--max-matches",
        type=int,
        default=None,
        help="Nombre maximum de matchs à traiter par joueur (défaut: tous)",
    )
    parser.add_argument(
        "--requests-per-second",
        type=int,
        default=5,
        help="Rate limiting API (défaut: 5 req/s)",
    )
    parser.add_argument(
        "--detection-mode",
        choices=["or", "and"],
        default="or",
        help=(
            "Mode de détection des matchs manquants: "
            "'or' = manque AU MOINS UNE donnée (défaut, compatible), "
            "'and' = manque TOUTES les données (strict, évite re-traitement)"
        ),
    )

    # ── Options de données à backfill ──
    parser.add_argument("--medals", action="store_true", help="Backfill les médailles")
    parser.add_argument("--events", action="store_true", help="Backfill les highlight events")
    parser.add_argument("--skill", action="store_true", help="Backfill les stats skill/MMR")
    # --force-skill : ajouté v5.1 pour forcer le re-téléchargement des données
    # skill/MMR (team_mmr, enemy_mmr, kills/deaths expected/stddev) de TOUS
    # les matchs, y compris ceux déjà peuplés. Utile après correction du bug
    # enemy_mmr (était ignoré avant v5.1).
    parser.add_argument(
        "--force-skill",
        action="store_true",
        help="Force le rescan skill/MMR pour TOUS les matchs",
    )
    parser.add_argument(
        "--personal-scores",
        action="store_true",
        help="Backfill les personal score awards",
    )
    parser.add_argument(
        "--performance-scores",
        action="store_true",
        help="Calculer les scores de performance manquants",
    )
    parser.add_argument(
        "--force-performance-scores",
        action="store_true",
        help="Force le calcul des scores de performance pour TOUS les matchs",
    )
    parser.add_argument("--aliases", action="store_true", help="Mettre à jour les aliases XUID")
    parser.add_argument(
        "--all-data",
        action="store_true",
        help="Backfill toutes les données",
    )

    # ── Accuracy ──
    parser.add_argument(
        "--accuracy",
        action="store_true",
        help="Backfill la précision (accuracy) pour les matchs avec accuracy NULL",
    )
    parser.add_argument(
        "--force-accuracy",
        action="store_true",
        help="Force la récupération de accuracy pour TOUS les matchs",
    )

    # ── Shots ──
    parser.add_argument(
        "--shots",
        action="store_true",
        help="Backfill shots_fired/shots_hit pour les matchs où ils sont NULL",
    )
    parser.add_argument(
        "--force-shots",
        action="store_true",
        help="Force la mise à jour de shots_fired/shots_hit pour TOUS les matchs",
    )

    # ── Enemy MMR ──
    parser.add_argument(
        "--enemy-mmr",
        action="store_true",
        help="Backfill enemy_mmr pour les matchs avec enemy_mmr NULL",
    )
    parser.add_argument(
        "--force-enemy-mmr",
        action="store_true",
        help="Force la récupération de enemy_mmr pour TOUS les matchs",
    )

    # ── Assets ──
    parser.add_argument(
        "--assets",
        action="store_true",
        help="Récupérer les noms (playlist, map, pair, game_variant) via Discovery UGC",
    )
    parser.add_argument(
        "--force-assets",
        action="store_true",
        help="Force la récupération des noms pour TOUS les matchs",
    )

    # ── Aliases ──
    parser.add_argument(
        "--force-aliases",
        action="store_true",
        help="Force la ré-extraction des aliases pour tous les matchs",
    )
    parser.add_argument(
        "--force-medals",
        action="store_true",
        help="Force le rescan de TOUS les matchs pour les médailles",
    )

    # ── Participants ──
    parser.add_argument(
        "--participants",
        action="store_true",
        help="Backfill les participants de match (table match_participants)",
    )
    parser.add_argument(
        "--force-participants",
        action="store_true",
        help="Force la ré-extraction des participants pour tous les matchs",
    )
    parser.add_argument(
        "--participants-scores",
        action="store_true",
        help="Backfill score et rang des participants",
    )
    parser.add_argument(
        "--participants-kda",
        action="store_true",
        help="Backfill kills, deaths, assists des participants",
    )
    parser.add_argument(
        "--participants-shots",
        action="store_true",
        help="Backfill shots_fired/shots_hit des participants",
    )
    parser.add_argument(
        "--force-participants-shots",
        action="store_true",
        help="Force shots pour tous les participants de tous les matchs",
    )
    parser.add_argument(
        "--participants-damage",
        action="store_true",
        help="Backfill damage_dealt/damage_taken des participants",
    )
    parser.add_argument(
        "--force-participants-damage",
        action="store_true",
        help="Force damage pour tous les participants de tous les matchs",
    )
    parser.add_argument(
        "--participants-avg-life",
        action="store_true",
        help="Backfill avg_life_seconds (durée de vie moyenne) des participants",
    )
    parser.add_argument(
        "--force-participants-avg-life",
        action="store_true",
        help="Force avg_life_seconds pour tous les participants de tous les matchs",
    )

    # ── Killer/victim ──
    parser.add_argument(
        "--killer-victim",
        action="store_true",
        help="Backfill les paires killer/victim depuis highlight_events",
    )

    # ── End time ──
    parser.add_argument(
        "--end-time",
        action="store_true",
        help="Remplir end_time (start_time + time_played_seconds)",
    )
    parser.add_argument(
        "--force-end-time",
        action="store_true",
        help="Recalculer end_time pour tous les matchs",
    )

    # ── Sessions ──
    parser.add_argument(
        "--sessions",
        action="store_true",
        help=(
            "Backfill session_id, session_label ET colonnes amis "
            "(is_with_friends, known_teammates_count, friends_xuids) pour les matchs stables ≥ 4h. "
            "Requiert friends_defaults.json pour is_with_friends."
        ),
    )
    parser.add_argument(
        "--force-sessions",
        action="store_true",
        help=(
            "Recalculer sessions + colonnes amis même si session_id déjà rempli "
            "(utile pour rétro-remplir is_with_friends sur les anciens matchs)"
        ),
    )

    # ── Teammates signature ──
    parser.add_argument(
        "--teammates-sig",
        action="store_true",
        help=(
            "Reconstruire teammates_signature + is_with_friends depuis shared_matches "
            "pour les matchs où teammates_signature IS NULL."
        ),
    )
    parser.add_argument(
        "--force-teammates-sig",
        action="store_true",
        help="Recalculer teammates_signature + is_with_friends pour TOUS les matchs.",
    )

    # ── Citations ──
    parser.add_argument(
        "--citations",
        action="store_true",
        help="Calculer et insérer les citations pour les matchs existants",
    )
    parser.add_argument(
        "--force-citations",
        action="store_true",
        help="Recalculer les citations même si déjà présentes",
    )

    # ── Participants enrich (V5 final) ──
    parser.add_argument(
        "--participants-enrich",
        action="store_true",
        help="Backfill colonnes étendues (headshot_kills, kda, etc.) + MMR dans shared.match_participants",
    )
    parser.add_argument(
        "--force-participants-enrich",
        action="store_true",
        help="Force le backfill des colonnes étendues même si déjà remplies",
    )

    # ── Granulaire MMR (v5.2) ──────────────────────────────────────────────
    parser.add_argument(
        "--team-mmr",
        action="store_true",
        help="Backfill team_mmr si NULL (granulaire v5.2)",
    )
    parser.add_argument(
        "--force-team-mmr",
        action="store_true",
        help="Force rechargement team_mmr pour TOUS les matchs",
    )
    parser.add_argument(
        "--mmr",
        action="store_true",
        help="= --team-mmr + --enemy-mmr (group MMR)",
    )
    parser.add_argument(
        "--force-mmr",
        action="store_true",
        help="= --force-team-mmr + --force-enemy-mmr",
    )

    # ── Granulaire Expected (v5.2) ─────────────────────────────────────────
    parser.add_argument(
        "--kills-expected",
        action="store_true",
        help="Backfill kills_expected/stddev si NULL",
    )
    parser.add_argument(
        "--deaths-expected",
        action="store_true",
        help="Backfill deaths_expected/stddev si NULL",
    )
    parser.add_argument(
        "--assists-expected",
        action="store_true",
        help="Backfill assists_expected/stddev si NULL",
    )
    parser.add_argument(
        "--expected",
        action="store_true",
        help="= --kills-expected + --deaths-expected + --assists-expected",
    )
    parser.add_argument("--force-kills-expected", action="store_true")
    parser.add_argument("--force-deaths-expected", action="store_true")
    parser.add_argument("--force-assists-expected", action="store_true")
    parser.add_argument(
        "--force-expected",
        action="store_true",
        help="Force rechargement de toutes les expected values",
    )

    # ── Combat granulaire (v5.2) ───────────────────────────────────────────
    parser.add_argument(
        "--damage",
        action="store_true",
        help="Backfill damage_dealt/damage_taken si NULL",
    )
    parser.add_argument(
        "--avg-life",
        action="store_true",
        help="Backfill avg_life_seconds si NULL",
    )
    parser.add_argument(
        "--combat",
        action="store_true",
        help="= --accuracy + --shots + --damage",
    )
    parser.add_argument("--force-damage", action="store_true")
    parser.add_argument("--force-avg-life", action="store_true")
    parser.add_argument(
        "--force-combat",
        action="store_true",
        help="= --force-accuracy + --force-shots + --force-damage",
    )

    # ── Kills détaillés (v5.2) ─────────────────────────────────────────────
    parser.add_argument(
        "--grenade-kills",
        action="store_true",
        help="Backfill grenade_kills si NULL",
    )
    parser.add_argument(
        "--melee-kills",
        action="store_true",
        help="Backfill melee_kills si NULL",
    )
    parser.add_argument(
        "--power-weapon-kills",
        action="store_true",
        help="Backfill power_weapon_kills si NULL",
    )
    parser.add_argument(
        "--headshot-kills",
        action="store_true",
        help="Backfill headshot_kills si NULL",
    )
    parser.add_argument(
        "--max-spree",
        action="store_true",
        help="Backfill max_killing_spree si NULL",
    )
    parser.add_argument(
        "--kills-detail",
        action="store_true",
        help="= --grenade-kills + --melee-kills + --power-weapon-kills + --headshot-kills",
    )
    parser.add_argument("--force-grenade-kills", action="store_true")
    parser.add_argument("--force-melee-kills", action="store_true")
    parser.add_argument("--force-power-weapon-kills", action="store_true")
    parser.add_argument("--force-headshot-kills", action="store_true")
    parser.add_argument("--force-max-spree", action="store_true")
    parser.add_argument(
        "--force-kills-detail",
        action="store_true",
        help="Force rechargement de tous les kills détaillés",
    )

    # ── Divers granulaires (v5.2) ──────────────────────────────────────────
    parser.add_argument(
        "--kda-recalc",
        action="store_true",
        help="Recalcule kda si NULL",
    )
    parser.add_argument(
        "--time-played",
        action="store_true",
        help="Backfill time_played_seconds si NULL",
    )
    parser.add_argument("--force-kda-recalc", action="store_true")
    parser.add_argument("--force-time-played", action="store_true")

    # ── Team scores (match_registry) ──────────────────────────────────────
    parser.add_argument(
        "--team-scores",
        action="store_true",
        help="Peuple team_0_score/team_1_score dans match_registry où NULL",
    )
    parser.add_argument(
        "--force-team-scores",
        action="store_true",
        help="Force le rechargement des team scores même s'ils sont déjà présents",
    )
    parser.add_argument(
        "--btb-only",
        action="store_true",
        help="Limite --team-scores aux matchs BTB objectifs (CTF/TC/Stockpile) avec score corrompu (>100)",
    )
    parser.add_argument(
        "--arena-only",
        action="store_true",
        help="Limite --team-scores aux matchs non-BTB objectifs avec score corrompu (>100)",
    )

    # ── Mode category (match_registry — local, sans API) ──────────────────
    parser.add_argument(
        "--mode-category",
        action="store_true",
        help=(
            "Recalcule mode_category dans match_registry depuis pair_name "
            "(local, sans appel API). Corrige les ~1289 matchs avec mode_category=NULL."
        ),
    )
    parser.add_argument(
        "--force-mode-category",
        action="store_true",
        help="Force le recalcul de mode_category pour TOUS les matchs (même déjà renseignés)",
    )

    # ── Nettoyage structurel des DBs joueurs ──────────────────────────────
    parser.add_argument(
        "--cleanup-player-dbs",
        action="store_true",
        help=(
            "Supprime les vues cassées (v_highlight_events, v_match_participants, "
            "v_match_stats, v_medals_earned) et tables legacy (match_participants) "
            "dans chaque stats.duckdb joueur. Opération locale, sans API."
        ),
    )

    # ── Groupe Core Stats (v5.2) ───────────────────────────────────────────
    parser.add_argument(
        "--core-stats",
        action="store_true",
        help="= accuracy, shots, damage, avg-life, kills-detail, kda-recalc, time-played",
    )
    parser.add_argument(
        "--force-core-stats",
        action="store_true",
        help="Force rechargement de tous les core-stats",
    )

    # ── PVE / Firefight (v5.2) ─────────────────────────────────────────────
    parser.add_argument(
        "--pve-stats",
        action="store_true",
        help="Backfill stats PVE (Firefight) → shared_pve.duckdb (v5.2)",
    )
    parser.add_argument(
        "--force-pve-stats",
        action="store_true",
        help="Force le backfill PVE même si déjà tenté (MatchBits.PVE_STATS ignoré)",
    )

    # ── Weapon kills (v5.5) ────────────────────────────────────────────────
    parser.add_argument(
        "--weapons",
        action="store_true",
        help="Backfill kills par arme → shared.weapon_kills (v5.5)",
    )
    parser.add_argument(
        "--force-weapons",
        action="store_true",
        help="Force le backfill armes même si MatchBits.WEAPON_KILLS déjà posé",
    )
    parser.add_argument(
        "--force-no-film",
        action="store_true",
        help=(
            "Re-tenter les matchs marqués WEAPON_KILLS_NO_FILM (film 404/expiré). "
            "Implique --weapons. Utile si les films Halo redeviennent disponibles."
        ),
    )

    # ── LUSR / CSR — LevelUp Skill Rank (v5.2) ────────────────────────────
    parser.add_argument(
        "--lusr",
        action="store_true",
        help=(
            "Calculer le LUSR (LevelUp Skill Rank) pour les matchs non classés "
            "sans rating (calcul local TrueSkill 2, pas d'API)"
        ),
    )
    parser.add_argument(
        "--force-lusr",
        action="store_true",
        help="Recalculer et réécrire le LUSR pour TOUS les matchs non classés depuis zéro",
    )
    parser.add_argument(
        "--csr",
        action="store_true",
        help=(
            "Backfill le CSR (Competitive Skill Rating) depuis l'API pour les matchs classés "
            "(nécessite un appel API skill)"
        ),
    )
    parser.add_argument(
        "--force-csr",
        action="store_true",
        help="Forcer le re-fetch CSR pour tous les matchs classés même si déjà présents",
    )
    parser.add_argument(
        "--fetch-csr",
        action="store_true",
        help=(
            "Récupère le CSR actuel du joueur via get_playlist_csr (Ranked Arena + Slayer) "
            "et le stocke dans skill_history. Utilisé pour seeder le LUSR initial."
        ),
    )
    # ── Détection des bots ────────────────────────────────────────────────────
    parser.add_argument(
        "--bot-detection",
        action="store_true",
        help=(
            "Détecte les matchs où un coéquipier était un bot IA (xuid LIKE 'bid(%%') "
            "et met à jour had_bot_teammate dans player_match_enrichment."
        ),
    )

    # ── Détection domination (Steaktacular) ───────────────────────────────────
    parser.add_argument(
        "--dominance",
        action="store_true",
        help=(
            "Calcule dominance_flag dans player_match_enrichment : "
            "1=domination totale (notre équipe a Steaktacular), "
            "2=humiliation totale (équipe ennemie a Steaktacular)."
        ),
    )
    parser.add_argument(
        "--force-dominance",
        action="store_true",
        help="Recalculer dominance_flag pour tous les matchs (même ceux déjà calculés).",
    )

    # ── Aliases XUID → Gamertag depuis highlight_events ───────────────────────
    parser.add_argument(
        "--aliases-from-events",
        action="store_true",
        help=(
            "Backfille xuid_aliases depuis highlight_events.raw_json. "
            "Corrige les XUIDs manquants (l'API stats Halo ne retourne pas PlayerGamertag). "
            "Opération locale, sans appel API."
        ),
    )
    parser.add_argument(
        "--force-aliases-from-events",
        action="store_true",
        help="Met à jour TOUS les aliases depuis events, même les XUIDs déjà présents.",
    )

    # ── Gestion des métadonnées (citations) ───────────────────────────────────
    parser.add_argument(
        "--enable-pve-citations",
        action="store_true",
        help=(
            "Active les citations PVE désactivées (brute_slayer, skimmer_slayer) "
            "dans metadata.duckdb. Opération locale, app doit être arrêtée."
        ),
    )

    parser.add_argument(
        "--skill-rank",
        action="store_true",
        help="= --lusr + --csr (alias unifié pour backfill LUSR + CSR en une commande)",
    )
    parser.add_argument(
        "--force-skill-rank",
        action="store_true",
        help="= --force-lusr + --force-csr (recalcule tout depuis zéro)",
    )

    # ── Détection version SPNKr stale (v5.4) ──────────────────────────────
    parser.add_argument(
        "--detect-stale-events",
        action="store_true",
        help=(
            "Détecte les matchs syncés avec une version SPNKr obsolète "
            "(highlight_events potentiellement corrompus). Affiche un rapport "
            "sans modifier les données."
        ),
    )

    # ── Career Rank XP total (local, sans API) ───────────────────────────────────
    parser.add_argument(
        "--xp-total",
        action="store_true",
        help=(
            "Recalcule xp_total et xp_for_next_rank dans career_progression "
            "pour tous les joueurs, depuis les vraies valeurs metadata.duckdb. "
            "Corrige l'ancienne formule approximative hardcodée. Local, sans API."
        ),
    )

    # ── Médaille Vengeur (custom — revenge kill) ─────────────────────────────
    parser.add_argument(
        "--avenger",
        action="store_true",
        help=(
            "Calcule et insère la médaille Vengeur (ID 9000000001) dans medals_earned. "
            "Un vengeur = tuer l'ennemi responsable de sa mort précédente. "
            "Opération locale, sans API. Requiert killer_victim_pairs peuplé."
        ),
    )
    parser.add_argument(
        "--force-avenger",
        action="store_true",
        help="Recalcule la médaille Vengeur pour TOUS les matchs (écrase l'existant).",
    )

    # ── Référentiel médailles (metadata.duckdb, one-shot) ────────────────────
    parser.add_argument(
        "--medal-metadata",
        action="store_true",
        help=(
            "Peuple medal_definitions dans metadata.duckdb depuis les JSON. "
            "Opération one-shot, locale, sans API."
        ),
    )

    # ── Badges narrative comeback (local, v6.2) ───────────────────────────────
    parser.add_argument(
        "--comeback-badges",
        action="store_true",
        dest="comeback_badges",
        help=(
            "Calcule les badges narrative Remontada / Débandade / Contre-Remontada "
            "depuis les kill-events (highlight_events). Local, sans API. "
            "Requiert events_loaded = TRUE pour chaque match."
        ),
    )
    parser.add_argument(
        "--force-comeback-badges",
        action="store_true",
        dest="force_comeback_badges",
        help="Re-traite les matchs déjà badgés (valeurs 3-5) en plus des non badgés.",
    )

    return parser


def _get_usage_examples() -> str:
    """Retourne les exemples d'usage pour l'aide CLI."""
    return """
Exemples:
    # Backfill toutes les données pour un joueur
    python scripts/backfill_data.py --player SpartanC --all-data

    # Mode strict (pas de re-téléchargement si partiellement rempli)
    python scripts/backfill_data.py --player SpartanC --all-data --detection-mode and

    # Backfill uniquement les médailles
    python scripts/backfill_data.py --player SpartanC --medals

    # Calculer les scores de performance manquants
    python scripts/backfill_data.py --player SpartanC --performance-scores

    # Calculer le LUSR (local, sans API)
    python scripts/backfill_data.py --player SpartanC --lusr

    # Recalculer le LUSR depuis zéro
    python scripts/backfill_data.py --player SpartanC --force-lusr

    # Backfill le CSR depuis l'API (matchs classés)
    python scripts/backfill_data.py --player SpartanC --csr

    # LUSR + CSR en une seule commande (option recommandée)
    python scripts/backfill_data.py --player SpartanC --skill-rank

    # Tout recalculer depuis zéro (LUSR + CSR)
    python scripts/backfill_data.py --player SpartanC --force-skill-rank

    # Médaille Vengeur (kill de vengeance — local, sans API)
    python scripts/backfill_data.py --avenger
    python scripts/backfill_data.py --force-avenger

    # Backfill pour tous les joueurs
    python scripts/backfill_data.py --all --all-data

    # Mode dry-run (liste seulement)
    python scripts/backfill_data.py --player SpartanC --dry-run

    # Recalculer mode_category dans match_registry (local, sans API — ~1289 matchs)
    python scripts/backfill_data.py --mode-category

    # Recalculer mode_category pour TOUS les matchs
    python scripts/backfill_data.py --force-mode-category

    # Nettoyer vues cassées et tables legacy dans les DBs joueurs
    python scripts/backfill_data.py --cleanup-player-dbs

    # Backfill participants avec stats NULL (274 matchs, besoin API)
    python scripts/backfill_data.py --all --participants --force-participants

Workaround OR — Exécution par étapes:
    python scripts/backfill_data.py --player SpartanC --medals
    python scripts/backfill_data.py --player SpartanC --sessions
    python scripts/backfill_data.py --player SpartanC --participants-kda
    """
