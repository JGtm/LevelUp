"""Gestion des sessions web backend — cookies signés httpOnly.

La session est stockée côté serveur dans un fichier JSON par session_id.
Le cookie navigateur contient uniquement l'identifiant opaque signé avec
`itsdangerous.URLSafeTimedSerializer`.

Modèle de session :
- `SessionData` : contenu de la session serveur (jamais exposé au navigateur)
- `SessionStore` : lecture/écriture/purge des fichiers session
- `get_session` / `set_session` : helpers FastAPI pour les routes

Règles de sécurité :
- Cookie `httpOnly`, `Secure` en production, `SameSite=Lax`
- Contenu de session jamais exposé dans un body JSON
- TTL configurable (défaut : 7 jours)
- Purge des sessions expirées au démarrage et toutes les X requêtes
"""

from __future__ import annotations

import json
import time
import uuid
from dataclasses import asdict, dataclass, field
from pathlib import Path
from typing import TYPE_CHECKING, Any

import structlog
from fastapi import Cookie, Request, Response
from itsdangerous import BadSignature, SignatureExpired, URLSafeTimedSerializer

from apps.api.app.core.config import get_settings

if TYPE_CHECKING:
    pass

logger = structlog.get_logger(__name__)
_COOKIE_NAME = "levelup_session"


# ---------------------------------------------------------------------------
# Modèle de session serveur
# ---------------------------------------------------------------------------


@dataclass
class SessionData:
    """Contenu de la session serveur — JAMAIS exposé au navigateur."""

    session_id: str = field(default_factory=lambda: str(uuid.uuid4()))
    created_at: float = field(default_factory=time.time)
    last_seen_at: float = field(default_factory=time.time)
    current_player_slug: str | None = None
    locale: str = "fr"
    hints_visible: bool = True
    # Contexte auth Halo — stocké côté serveur uniquement
    auth_ready: bool = False
    _extra: dict[str, Any] = field(default_factory=dict)

    def touch(self) -> None:
        self.last_seen_at = time.time()

    def is_expired(self, ttl_seconds: int) -> bool:
        return (time.time() - self.last_seen_at) > ttl_seconds

    def to_dict(self) -> dict:
        d = asdict(self)
        d.pop("_extra", None)
        return d

    @classmethod
    def from_dict(cls, d: dict) -> SessionData:
        known = {f.name for f in cls.__dataclass_fields__.values()}  # type: ignore[attr-defined]
        extra = {k: v for k, v in d.items() if k not in known}
        filtered = {k: v for k, v in d.items() if k in known}
        obj = cls(**filtered)
        obj._extra = extra
        return obj


# ---------------------------------------------------------------------------
# Store de sessions
# ---------------------------------------------------------------------------


class SessionStore:
    """Stockage fichier des sessions — une session = un fichier JSON."""

    def __init__(self, storage_dir: str, ttl_seconds: int) -> None:
        self._dir = Path(storage_dir)
        self._dir.mkdir(parents=True, exist_ok=True)
        self._ttl = ttl_seconds

    def _path(self, session_id: str) -> Path:
        # Valide que session_id ne contient pas de traversal
        safe_id = "".join(c for c in session_id if c.isalnum() or c == "-")
        return self._dir / f"{safe_id}.json"

    def load(self, session_id: str) -> SessionData | None:
        path = self._path(session_id)
        if not path.exists():
            return None
        try:
            data = json.loads(path.read_text(encoding="utf-8"))
            session = SessionData.from_dict(data)
            if session.is_expired(self._ttl):
                self.delete(session_id)
                return None
            return session
        except Exception:
            logger.warning("session_load_error", session_id=session_id)
            return None

    def save(self, session: SessionData) -> None:
        path = self._path(session.session_id)
        path.write_text(json.dumps(session.to_dict(), default=str), encoding="utf-8")

    def delete(self, session_id: str) -> None:
        path = self._path(session_id)
        if path.exists():
            path.unlink(missing_ok=True)

    def purge_expired(self) -> int:
        """Supprime les sessions expirées. Retourne le nombre supprimé."""
        removed = 0
        for p in self._dir.glob("*.json"):
            try:
                data = json.loads(p.read_text(encoding="utf-8"))
                session = SessionData.from_dict(data)
                if session.is_expired(self._ttl):
                    p.unlink(missing_ok=True)
                    removed += 1
            except Exception:
                p.unlink(missing_ok=True)
                removed += 1
        if removed:
            logger.info("sessions_purged", count=removed)
        return removed


# ---------------------------------------------------------------------------
# Signer / vérifier les cookies
# ---------------------------------------------------------------------------


def _get_serializer() -> URLSafeTimedSerializer:
    settings = get_settings()
    return URLSafeTimedSerializer(settings.session_secret_key, salt="levelup-session")


def _sign_session_id(session_id: str) -> str:
    return _get_serializer().dumps(session_id)


def _unsign_session_id(token: str) -> str | None:
    settings = get_settings()
    try:
        return _get_serializer().loads(token, max_age=settings.session_ttl_seconds)
    except (SignatureExpired, BadSignature):
        return None


# ---------------------------------------------------------------------------
# Singleton store + helpers dépendances FastAPI
# ---------------------------------------------------------------------------


def _get_store() -> SessionStore:
    settings = get_settings()
    return SessionStore(settings.session_storage_dir, settings.session_ttl_seconds)


def get_or_create_session(
    request: Request,
    response: Response,
    session_token: str | None = Cookie(default=None, alias=_COOKIE_NAME),
) -> tuple[SessionData, bool]:
    """Dependency FastAPI : retourne (session, is_new).

    Si le cookie est absent ou invalide, crée une nouvelle session.
    Le cookie est posé dans `response` si la session est nouvelle ou rafraîchie.
    """
    settings = get_settings()
    store = _get_store()
    is_new = False
    session: SessionData | None = None

    if session_token:
        session_id = _unsign_session_id(session_token)
        if session_id:
            session = store.load(session_id)

    if session is None:
        session = SessionData()
        is_new = True

    session.touch()
    store.save(session)

    signed = _sign_session_id(session.session_id)
    response.set_cookie(
        key=_COOKIE_NAME,
        value=signed,
        httponly=True,
        samesite="lax",
        secure=settings.is_production,
        max_age=settings.session_ttl_seconds,
        path="/",
    )
    return session, is_new
