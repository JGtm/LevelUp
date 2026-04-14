"""Module UI - Gestion des alias, médailles et helpers d'interface."""

from src.ui.aliases import (
    display_name_from_xuid,
    get_xuid_aliases,
)
from src.ui.formatting import format_date_fr, format_mmss
from src.ui.medals import (
    get_local_medals_icons_dir,
    load_medal_name_maps,
    medal_has_known_label,
    medal_icon_path,
    medal_label,
    render_medals_grid,
)
from src.ui.profile_api import ProfileAppearance, get_profile_appearance, get_xuid_for_gamertag
from src.ui.profile_api_tokens import ensure_spnkr_tokens
from src.ui.settings import AppSettings, load_settings, patch_settings, save_settings
from src.ui.spartan_id import SpartanIdCard, render_spartan_id
from src.ui.translations import translate_pair_name, translate_playlist_name

__all__ = [
    # aliases
    "get_xuid_aliases",
    "display_name_from_xuid",
    # spartan id
    "SpartanIdCard",
    "render_spartan_id",
    # translations
    "translate_playlist_name",
    "translate_pair_name",
    # medals
    "load_medal_name_maps",
    "medal_has_known_label",
    "get_local_medals_icons_dir",
    "medal_label",
    "medal_icon_path",
    "render_medals_grid",
    # formatting
    "format_date_fr",
    "format_mmss",
    # settings
    "AppSettings",
    "load_settings",
    "patch_settings",
    "save_settings",
    # profile api
    "ProfileAppearance",
    "get_profile_appearance",
    "get_xuid_for_gamertag",
    "ensure_spnkr_tokens",
]
