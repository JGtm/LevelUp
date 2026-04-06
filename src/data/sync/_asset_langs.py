"""Langues cibles pour la récupération des traductions d'assets Halo Infinite.

Utilisé par :
- scripts/populate_asset_translations.py (Discovery UGC, Accept-Language header)
- scripts/populate_medal_metadata.py (champ translations de l'API médailles)
"""

from __future__ import annotations

# Langues supportées par l'API Halo Infinite (confirmées via champ translations médailles).
# Format BCP-47 : <langue>-<région>
TARGET_LANGS: tuple[str, ...] = (
    "en-US",
    "fr-FR",
    "de-DE",
    "es-ES",
    "es-MX",
    "it-IT",
    "ja-JP",
    "ko-KR",
    "pt-BR",
    "zh-CN",
    "zh-TW",
    "nl-NL",
    "pl-PL",
    "ru-RU",
)

# Langue de base (fallback si traduction absente)
DEFAULT_LANG = "en-US"

# Mapping BCP-47 → code court utilisé dans l'UI (get_lang() renvoie "fr", "en"…)
LANG_SHORT: dict[str, str] = {
    "en-US": "en",
    "fr-FR": "fr",
    "de-DE": "de",
    "es-ES": "es",
    "es-MX": "es",
    "it-IT": "it",
    "ja-JP": "ja",
    "ko-KR": "ko",
    "pt-BR": "pt",
    "zh-CN": "zh",
    "zh-TW": "zh",
    "nl-NL": "nl",
    "pl-PL": "pl",
    "ru-RU": "ru",
}

# Mapping inverse : code court → BCP-47 préféré (pour les lookups UI)
SHORT_TO_LANG: dict[str, str] = {
    "en": "en-US",
    "fr": "fr-FR",
    "de": "de-DE",
    "es": "es-ES",
    "it": "it-IT",
    "ja": "ja-JP",
    "ko": "ko-KR",
    "pt": "pt-BR",
    "zh": "zh-CN",
    "nl": "nl-NL",
    "pl": "pl-PL",
    "ru": "ru-RU",
}


def to_bcp47(lang_short: str) -> str:
    """Convertit un code court ('fr') en BCP-47 ('fr-FR').

    Retourne DEFAULT_LANG si le code n'est pas reconnu.
    """
    return SHORT_TO_LANG.get(lang_short, DEFAULT_LANG)
