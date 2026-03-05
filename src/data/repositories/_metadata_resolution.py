"""Mixin de résolution des métadonnées pour DuckDBRepository.

Construit les expressions SQL et jointures pour résoudre les noms de maps,
playlists, pairs et le fallback MMR depuis les bases metadata et locales.
Les résultats sont cachés en instance (le schéma ne change pas en session).
"""

from __future__ import annotations

import logging

logger = logging.getLogger(__name__)


class MetadataResolutionMixin:
    """Résolution SQL des noms de maps/playlists et fallback MMR."""

    def _build_metadata_resolution(self, conn) -> tuple[str, str, str, str]:
        """Construit les expressions SQL et jointures pour les métadonnées.

        Le résultat est caché en instance car le schéma metadata ne change pas en session.

        Returns:
            Tuple (metadata_joins, map_name_expr, playlist_name_expr, pair_name_expr)
        """
        # Cache d'instance (v5.1 perf — 1bis.3)
        if self._metadata_resolution_cache is not None:
            return self._metadata_resolution_cache

        metadata_joins = ""
        map_name_expr = "match_stats.map_name"
        playlist_name_expr = "match_stats.playlist_name"
        pair_name_expr = "match_stats.pair_name"

        if "meta" not in self._attached_dbs:
            logger.debug("Metadata DB non attachée, pas de résolution des métadonnées")
            return metadata_joins, map_name_expr, playlist_name_expr, pair_name_expr

        try:
            # Vérifier les tables de métadonnées en une seule requête
            tables_check = conn.execute(
                "SELECT table_name FROM information_schema.tables "
                "WHERE table_schema = 'meta' AND table_name IN "
                "('maps', 'playlists', 'map_mode_pairs', 'playlist_map_mode_pairs')"
            ).fetchall()
            existing_tables = {row[0] for row in tables_check}

            has_maps = "maps" in existing_tables
            has_playlists = "playlists" in existing_tables
            has_pairs_map_mode = "map_mode_pairs" in existing_tables
            has_pairs_playlist = "playlist_map_mode_pairs" in existing_tables
            has_pairs = has_pairs_map_mode or has_pairs_playlist
            pair_table_name = (
                "map_mode_pairs"
                if has_pairs_map_mode
                else ("playlist_map_mode_pairs" if has_pairs_playlist else None)
            )

            logger.debug(
                "Résolution métadonnées: maps=%s, playlists=%s, pairs=%s (table=%s)",
                has_maps,
                has_playlists,
                has_pairs,
                pair_table_name,
            )

            if has_maps:
                metadata_joins += (
                    " LEFT JOIN meta.maps m_meta ON match_stats.map_id = m_meta.asset_id"
                )
                map_name_expr = "COALESCE(m_meta.public_name, match_stats.map_name)"

            if has_playlists:
                metadata_joins += (
                    " LEFT JOIN meta.playlists p_meta ON match_stats.playlist_id = p_meta.asset_id"
                )
                playlist_name_expr = "COALESCE(p_meta.public_name, match_stats.playlist_name)"

            if has_pairs and pair_table_name:
                metadata_joins += (
                    f" LEFT JOIN meta.{pair_table_name} pair_meta"
                    " ON match_stats.pair_id = pair_meta.asset_id"
                )
                pair_name_expr = "COALESCE(pair_meta.public_name, match_stats.pair_name)"
        except Exception as e:
            logger.warning("Erreur construction jointures métadonnées: %s", e)

        result = (metadata_joins, map_name_expr, playlist_name_expr, pair_name_expr)
        self._metadata_resolution_cache = result
        return result

    def _build_mmr_fallback(self, conn) -> tuple[str, str, str]:
        """Construit la jointure et expressions pour le fallback MMR.

        Si ``player_match_stats`` existe, utilise COALESCE pour récupérer les MMR
        depuis cette table si ``match_stats`` a des valeurs NULL.

        Le résultat est caché en instance car les tables locales ne changent pas en session.

        Returns:
            Tuple (pms_join, team_mmr_expr, enemy_mmr_expr)
        """
        # Cache d'instance (v5.1 perf — 1bis.3)
        if self._mmr_fallback_cache is not None:
            return self._mmr_fallback_cache

        pms_join = ""
        team_mmr_expr = "match_stats.team_mmr"
        enemy_mmr_expr = "match_stats.enemy_mmr"

        try:
            pms_tables = conn.execute(
                "SELECT table_name FROM information_schema.tables "
                "WHERE table_schema='main' AND table_name='player_match_stats'"
            ).fetchall()
            if pms_tables:
                pms_join = (
                    " LEFT JOIN player_match_stats pms ON match_stats.match_id = pms.match_id"
                )
                team_mmr_expr = "COALESCE(match_stats.team_mmr, pms.team_mmr)"
                enemy_mmr_expr = "COALESCE(match_stats.enemy_mmr, pms.enemy_mmr)"
        except Exception:
            pass

        result = (pms_join, team_mmr_expr, enemy_mmr_expr)
        self._mmr_fallback_cache = result
        return result
