"""Gestion des paramètres utilisateur (persistés).

Objectif:
- Déplacer les "paramètres" hors sidebar (onglet Paramètres)
- Permettre une exécution sur NAS/Docker avec un fichier de config monté

Le chemin est configurable via LEVELUP_SETTINGS_PATH.
"""

from __future__ import annotations

import contextlib
import json
import logging
import os
import threading
from typing import Any, Literal

from pydantic import BaseModel, ConfigDict, Field, field_validator, model_validator

logger = logging.getLogger(__name__)


def get_settings_path() -> str:
    """Retourne le chemin du fichier app_settings.json."""
    override = os.environ.get("LEVELUP_SETTINGS_PATH")
    if override and str(override).strip():
        return str(override).strip()
    repo_root = os.path.dirname(os.path.dirname(os.path.dirname(__file__)))
    return os.path.join(repo_root, "app_settings.json")


class AppSettings(BaseModel):
    """Paramètres utilisateur persistés en JSON.

    Pydantic v2 gère automatiquement la coercion des types (str→bool, str→int, etc.)
    et ignore les clés inconnues (extra='ignore') comme ``_comment``.
    """

    model_config = ConfigDict(extra="ignore", str_strip_whitespace=True, frozen=True)

    # Discord
    discord_notifications_enabled: bool = False
    discord_webhook_url: str = ""  # Fallback si DISCORD_WEBHOOK_URL absent de l'env
    discord_notify_sync: bool = True  # Notif après chaque sync
    discord_notify_backfill: bool = True  # Notif après chaque backfill
    discord_notify_new_version: bool = True  # Notif lors d'une nouvelle version majeure (vX.Y)
    discord_notify_new_media: bool = True  # Notif quand de nouveaux médias sont indexés
    last_notified_version: str = ""  # Version lors du dernier envoi de notif déploiement

    # Médias
    media_enabled: bool = True
    media_screens_dir: str = ""  # Déprécié: migration vers media_captures_base_dir
    media_videos_dir: str = ""  # Déprécié: migration vers media_captures_base_dir
    media_captures_base_dir: str = ""  # Base unique: base_dir/{gamertag}/ pour captures
    media_tolerance_minutes: int = Field(default=3, ge=0)
    media_indexing_interval_hours: int = Field(default=4, ge=0)  # 0 = run unique au démarrage
    media_reindex_after_sync: bool = True  # Réindexer les médias après chaque sync
    media_watcher_enabled: bool = True  # Watcher inotify Linux (remplace le thread périodique)
    media_watcher_debounce_seconds: int = Field(default=5, ge=1, le=60)  # Délai avant indexation

    # UX
    refresh_clears_caches: bool = False

    # Source
    prefer_spnkr_db_if_available: bool = True

    # SPNKr (API → DB) : rafraîchissement automatique
    spnkr_refresh_on_start: bool = True
    spnkr_refresh_on_manual_refresh: bool = True
    spnkr_refresh_match_type: Literal["all", "matchmaking", "custom", "local"] = "matchmaking"
    spnkr_refresh_max_matches: int = Field(default=200, ge=1)
    spnkr_refresh_with_highlight_events: bool = False

    # Backfill après synchronisation
    spnkr_refresh_with_backfill: bool = False  # Backfill complet après sync
    spnkr_refresh_backfill_medals: bool = False
    spnkr_refresh_backfill_events: bool = False
    spnkr_refresh_backfill_skill: bool = False
    spnkr_refresh_backfill_personal_scores: bool = False
    spnkr_refresh_backfill_performance_scores: bool = True  # Par défaut activé
    spnkr_refresh_backfill_aliases: bool = False
    spnkr_refresh_backfill_lusr: bool = True  # LUSR local (sans API) — par défaut activé
    spnkr_refresh_backfill_weapons: bool = False  # v5.5 : kills par arme depuis films SPNKr

    # Fichiers (overrides optionnels)
    aliases_path: str = ""
    profiles_path: str = ""

    # Profil joueur (bannière/rang) — télécharge les assets quand les tokens sont dispo
    profile_assets_download_enabled: bool = True
    profile_assets_auto_refresh_hours: int = Field(default=24, ge=0)  # 0 = désactivé

    # Profil joueur (auto depuis API Waypoint via SPNKr)
    profile_api_enabled: bool = True
    profile_api_auto_refresh_hours: int = Field(default=6, ge=0)

    profile_banner: str = ""  # URL https://... ou chemin local
    profile_emblem: str = ""  # URL https://... ou chemin local
    profile_backdrop: str = ""  # URL https://... ou chemin local
    profile_nameplate: str = ""  # URL https://... ou chemin local
    profile_service_tag: str = ""  # ex: "SPTA"
    profile_id_badge_text_color: str = ""  # ex: "#FFFFFF"
    profile_rank_label: str = ""  # ex: "Diamond III" / "Héros" / etc.
    profile_rank_subtitle: str = ""  # ex: "CSR 1540" / "Saison 5" / etc.

    # Architecture de données v4
    repository_mode: Literal["duckdb"] = "duckdb"
    enable_duckdb_analytics: bool = False

    # Doppler secrets (opt-in)
    doppler_enabled: bool = False  # Charger les secrets depuis Doppler au démarrage
    doppler_project: str = ""  # Nom du projet Doppler (ex: "levelup")
    doppler_config: str = ""  # Config Doppler (ex: "dev", "prod")

    # Tailscale funnel (opt-in)
    tailscale_funnel_enabled: bool = False  # Exposer l'app via Tailscale au démarrage

    # Authentification
    auth_method: Literal["refresh_token", "msal"] = (
        "refresh_token"  # refresh_token = OAuth v2 direct ; msal = Device Code Flow Azure
    )

    # Internationalisation
    lang: Literal["fr", "en"] = "fr"  # Langue de l'UI
    discord_lang: Literal["fr", "en"] = "fr"  # Langue des messages Discord
    cli_lang: Literal["fr", "en"] = "fr"  # Langue des scripts CLI

    # Affichage
    user_timezone: str = "Europe/Paris"  # Timezone pour les dates/heures affichées (IANA)

    # Carrière — top/pire matchs
    career_top_exclude_btb: bool = False  # Exclure les matchs BTB du top 10 meilleurs/pires

    # Affichage — records historiques
    show_records: bool = (
        False  # Afficher les barres de records sur les graphes Escouade (expérimental)
    )

    # Affichage — modes de jeu
    normalize_mode_labels: bool = (
        True  # Supprimer les préfixes redondants (ex: BTB:Slayer → Assassin)
    )

    # --- Validators ---

    @model_validator(mode="before")
    @classmethod
    def _strip_none_values(cls, data: Any) -> Any:
        """Supprime les valeurs None pour laisser les défauts Pydantic s'appliquer."""
        if isinstance(data, dict):
            return {k: v for k, v in data.items() if v is not None}
        return data

    @field_validator("spnkr_refresh_match_type", mode="before")
    @classmethod
    def _normalize_match_type(cls, v: Any) -> str:
        """Normalise le type de match en minuscules avec fallback."""
        s = str(v).strip().lower()
        return s if s in {"all", "matchmaking", "custom", "local"} else "matchmaking"

    @field_validator("auth_method", mode="before")
    @classmethod
    def _normalize_auth_method(cls, v: Any) -> str:
        """Normalise la méthode d'auth avec fallback sur 'refresh_token'."""
        s = str(v).strip().lower()
        return s if s in {"refresh_token", "msal"} else "refresh_token"

    @field_validator("lang", "discord_lang", "cli_lang", mode="before")
    @classmethod
    def _normalize_lang(cls, v: Any) -> str:
        """Normalise la langue en minuscules avec fallback sur 'fr'."""
        s = str(v).strip().lower()
        return s if s in {"fr", "en"} else "fr"

    @field_validator("user_timezone", mode="before")
    @classmethod
    def _validate_timezone(cls, v: Any) -> str:
        """Valide la timezone via ZoneInfo (standard IANA) avec fallback 'Europe/Paris'."""
        from zoneinfo import ZoneInfo, ZoneInfoNotFoundError

        s = str(v or "").strip()
        if not s:
            return "Europe/Paris"
        try:
            ZoneInfo(s)
            return s
        except (ZoneInfoNotFoundError, KeyError, Exception):
            return "Europe/Paris"

    @field_validator("repository_mode", mode="before")
    @classmethod
    def _normalize_repo_mode(cls, _v: Any) -> str:
        """Seul mode supporté : duckdb."""
        return "duckdb"

    @field_validator(
        "media_tolerance_minutes",
        "profile_assets_auto_refresh_hours",
        "profile_api_auto_refresh_hours",
        mode="before",
    )
    @classmethod
    def _clamp_non_negative(cls, v: Any, info: Any) -> int:
        """Convertit en int ≥ 0 avec fallback sur le défaut du champ."""
        try:
            return max(0, int(v))
        except (ValueError, TypeError):
            return cls.model_fields[info.field_name].default

    @field_validator("spnkr_refresh_max_matches", mode="before")
    @classmethod
    def _clamp_positive(cls, v: Any, info: Any) -> int:
        """Convertit en int ≥ 1 avec fallback sur le défaut du champ."""
        try:
            return max(1, int(v))
        except (ValueError, TypeError):
            return cls.model_fields[info.field_name].default


def _apply_legacy_migrations(data: dict) -> dict:
    """Transformations de schéma one-time appliquées uniquement au chargement disque.

    N'est jamais appelé sur model_copy ou patch_settings.
    Pour les migrations futures : ajouter un bloc ici, jamais un @model_validator.
    """
    # Migration media_captures_base_dir (anciens champs media_screens_dir / media_videos_dir)
    if not data.get("media_captures_base_dir") and (
        data.get("media_screens_dir") or data.get("media_videos_dir")
    ):
        from pathlib import Path

        paths = [
            Path(p)
            for p in [data.get("media_screens_dir", ""), data.get("media_videos_dir", "")]
            if p
        ]
        with contextlib.suppress(ValueError, TypeError):
            common = paths[0].parent
            for p in paths[1:]:
                common = Path(os.path.commonpath([str(common), str(p)]))
            if str(common).strip():
                data["media_captures_base_dir"] = str(common)
    return data


def _settings_bak_path(path: str) -> str:
    """Retourne le chemin du fichier de backup (.json.bak)."""
    return path + ".bak"


def _try_parse_settings(path: str, label: str) -> AppSettings | None:
    """Tente de lire et valider un fichier settings. Retourne None en cas d'échec."""
    try:
        with open(path, encoding="utf-8") as f:
            obj = json.load(f) or {}
    except FileNotFoundError:
        return None
    except json.JSONDecodeError as exc:
        logger.warning("Settings %s : JSON invalide (%s)", label, exc)
        return None
    except Exception as exc:
        logger.error("Settings %s : lecture impossible (%s)", label, exc)
        return None

    if not isinstance(obj, dict):
        logger.warning("Settings %s : contenu inattendu (type=%s)", label, type(obj).__name__)
        return None

    try:
        return AppSettings.model_validate(_apply_legacy_migrations(obj))
    except Exception as exc:
        logger.error("Settings %s : validation échouée (%s)", label, exc)
        return None


def load_settings() -> AppSettings:
    """Charge les paramètres depuis le fichier JSON.

    Cascade de récupération :
    1. app_settings.json (principal)
    2. app_settings.json.bak (backup du dernier état valide)
    3. AppSettings() — defaults, sans écriture sur disque
    """
    import traceback as _tb

    _caller = _tb.extract_stack()[-2]
    path = get_settings_path()
    bak_path = _settings_bak_path(path)

    # ① Fichier principal
    if os.path.exists(path):
        settings = _try_parse_settings(path, "principal")
        if settings is not None:
            logger.info(
                "Settings chargées depuis %s [appelé depuis %s:%d]",
                path,
                _caller.filename.split("\\")[-1].split("/")[-1],
                _caller.lineno,
            )
            return settings
        # Fichier corrompu/illisible → tenter le backup
        logger.warning("Settings principal illisible — tentative backup %s", bak_path)
    else:
        # Première utilisation : créer un fichier vide proprement
        try:
            os.makedirs(os.path.dirname(path) or ".", exist_ok=True)
            _atomic_write(path, "{}\n")
        except Exception as exc:
            logger.warning("Impossible de créer %s : %s", path, exc)
        return AppSettings()

    # ② Backup
    if os.path.exists(bak_path):
        settings = _try_parse_settings(bak_path, "backup")
        if settings is not None:
            logger.warning(
                "Settings récupérées depuis le backup %s — le fichier principal sera restauré",
                bak_path,
            )
            # Restaurer le principal depuis le backup (écriture atomique)
            try:
                import shutil

                shutil.copy2(bak_path, path)
                logger.info("Fichier principal restauré depuis backup")
            except Exception as exc:
                logger.error("Restauration backup → principal échouée : %s", exc)
            return settings
        logger.error("Settings backup %s aussi illisible", bak_path)
    else:
        logger.warning("Pas de backup disponible (%s)", bak_path)

    # ③ Defaults — ne pas écrire sur disque (ne pas écraser un fichier potentiellement récupérable)
    logger.error(
        "Impossible de charger les settings depuis %s ni son backup — defaults appliqués",
        path,
    )
    return AppSettings()


def _atomic_write(path: str, content: str) -> None:
    """Écrit `content` dans `path` de façon atomique (même filesystem).

    - Linux/Debian : rename() → vraiment atomique, même fichier ouvert ailleurs
    - Windows NTFS : quasi-atomique ; retry sur PermissionError (fichier verrouillé)
    - Le fichier temporaire est créé dans le même dossier pour garantir le même filesystem.
    """
    import shutil
    import tempfile

    dir_path = os.path.dirname(path) or "."
    os.makedirs(dir_path, exist_ok=True)

    fd, tmp_path = tempfile.mkstemp(dir=dir_path, prefix=".settings_", suffix=".tmp")
    try:
        try:
            with os.fdopen(fd, "w", encoding="utf-8") as f:
                f.write(content)
                f.flush()
                if os.name != "nt":  # fsync utile sur Linux, superflu sur Windows NTFS
                    os.fsync(f.fileno())
        except Exception:
            with contextlib.suppress(OSError):
                os.close(fd)
            raise

        # Remplacement atomique — retry multiple sur Windows (verrou antivirus / Streamlit)
        _replaced = False
        if os.name == "nt":
            for _attempt, _delay in enumerate([0.05, 0.1, 0.2, 0.5], start=1):
                try:
                    os.replace(tmp_path, path)
                    _replaced = True
                    break
                except PermissionError:
                    import time as _time

                    _time.sleep(_delay)
            if not _replaced:
                os.replace(tmp_path, path)  # dernière tentative — lève si toujours bloqué
        else:
            os.replace(tmp_path, path)
        tmp_path = None  # Succès : ne pas supprimer dans finally

    finally:
        if tmp_path is not None and os.path.exists(tmp_path):
            with contextlib.suppress(OSError):
                os.unlink(tmp_path)

    # Backup du dernier bon état (non-fatal)
    bak_path = _settings_bak_path(path)
    try:
        shutil.copy2(path, bak_path)
    except Exception as exc:
        logger.warning("Backup settings échoué (non-bloquant) : %s", exc)


# ---------------------------------------------------------------------------
# V3 — sources de vérité unifiées
# ---------------------------------------------------------------------------

_SENTINEL = object()  # Valeur sentinelle « aucun changement »
_WRITE_LOCK: threading.Lock = threading.Lock()  # Sérialise les I/O disque (multi-tab)
_PROCESS_CACHE: dict[str, str | None] = {"last_content": None}  # Dédup par contenu


def _write_settings(settings: AppSettings) -> tuple[bool, str]:
    """Sauvegarde atomique + thread-safe. Usage interne uniquement.

    L'API publique est patch_settings().
    """
    with _WRITE_LOCK:
        try:
            content = json.dumps(settings.model_dump(), ensure_ascii=False, indent=2)
            if content == _PROCESS_CACHE["last_content"]:
                return True, ""  # dédup — contenu identique
            _atomic_write(get_settings_path(), content)
            _PROCESS_CACHE["last_content"] = content

            if __debug__:
                _verify = _try_parse_settings(get_settings_path(), "post-write verify")
                if _verify is not None:
                    for _field in ("show_records", "lang", "discord_notify_sync"):
                        _w = getattr(settings, _field, None)
                        _d = getattr(_verify, _field, None)
                        if _w != _d:
                            logger.error(
                                "_write_settings MISMATCH: field=%s wrote=%r disk=%r",
                                _field,
                                _w,
                                _d,
                            )

            return True, ""
        except Exception as exc:
            logger.error("_write_settings échoué : %s", exc)
            return False, f"Impossible d'écrire {get_settings_path()}: {exc}"


def patch_settings(key: str, value: Any) -> tuple[AppSettings, bool, str]:
    """Met à jour un champ dans session_state et persiste sur disque.

    Toujours lire depuis session_state — jamais depuis un paramètre local.
    Retourne (nouvel_objet, succès, message_erreur).
    """
    import streamlit as st

    current: AppSettings | None = st.session_state.get("app_settings")
    if current is None or not isinstance(current, AppSettings):
        logger.debug("patch_settings: session_state vide, chargement depuis disque")
        current = load_settings()

    if getattr(current, key, _SENTINEL) == value:
        return current, True, ""  # Aucun changement, pas d'écriture inutile

    updated = current.model_copy(update={key: value})
    st.session_state["app_settings"] = updated  # Mettre à jour AVANT l'écriture
    ok, err = _write_settings(updated)
    if not ok:
        logger.error("patch_settings: rollback %s=%r — %s", key, value, err)
        st.session_state["app_settings"] = current  # Rollback
    else:
        logger.debug("patch_settings: %s=%r persisté", key, value)
    return updated, ok, err


def save_settings(settings: AppSettings) -> tuple[bool, str]:
    """Sauvegarde les paramètres dans le fichier JSON de façon atomique.

    Conservé pour compatibilité avec les scripts CLI.
    Dans le code UI, préférer patch_settings().
    """
    return _write_settings(settings)
