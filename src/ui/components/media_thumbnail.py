"""Composant Thumbnail média – affichage statique, survol animé, lightbox optionnelle."""

from __future__ import annotations

import base64
import hashlib
import io
import json
import logging
from functools import lru_cache
from pathlib import Path

import streamlit as st

from src.ui.components.media_lightbox import build_lightbox_html

logger = logging.getLogger(__name__)

# Taille max pour embarquer en data URI (éviter iframe trop lourd)
# Les GIF de captures Halo peuvent être volumineux → limites assouplies
MAX_STATIC_BYTES = 3_000_000  # ~3 MB pour miniature statique (dont GIF)
MAX_HOVER_BYTES = 4_000_000  # ~4 MB pour version survol (GIF animé)
MAX_LIGHTBOX_BYTES = 5_000_000  # ~5 MB pour lightbox (sinon on affiche la miniature)


@lru_cache(maxsize=256)
def _gif_first_frame_png_bytes(gif_path_str: str) -> bytes | None:
    """Extrait la première frame d'un GIF et la retourne en PNG binaire."""
    gif_path = Path(gif_path_str)
    try:
        from PIL import Image
    except ImportError:
        return None
    try:
        with Image.open(gif_path) as img:
            img.seek(0)
            frame = img.convert("RGBA")
            buf = io.BytesIO()
            frame.save(buf, format="PNG", optimize=True)
            data = buf.getvalue()
            if len(data) > MAX_STATIC_BYTES:
                return None
            return data
    except Exception as exc:
        logger.debug("GIF first frame %s: %s", gif_path, exc)
        return None


def _gif_first_frame_to_data_uri(gif_path: Path) -> str | None:
    """Extrait la première frame d'un GIF et retourne une data URI PNG (statique, ne s'anime pas)."""
    data = _gif_first_frame_png_bytes(str(gif_path.resolve()))
    if data is None:
        return None
    b64 = base64.standard_b64encode(data).decode("ascii")
    return f"data:image/png;base64,{b64}"


def load_native_thumbnail_source(
    static_path: Path | str,
    *,
    hover_path: Path | str | None = None,
) -> bytes | str | None:
    """Retourne une source native Streamlit pour afficher une miniature sans iframe.

    Pour les miniatures GIF utilisées comme aperçu vidéo, on retourne la première
    frame statique en PNG quand elle est disponible. Sinon, on retombe sur le
    chemin du fichier image tel quel.
    """
    path = Path(static_path)
    if not path.exists():
        return None

    is_gif_thumbnail = path.suffix.lower() == ".gif" and hover_path is not None
    if is_gif_thumbnail:
        first_frame = _gif_first_frame_png_bytes(str(path.resolve()))
        if first_frame is not None:
            return first_frame

    return str(path)


def _path_to_data_uri(path: Path, max_bytes: int, mime: str) -> str | None:
    """Lit un fichier et retourne une data URI, ou None si fichier absent / trop gros."""
    if not path.exists():
        return None
    try:
        data = path.read_bytes()
        if len(data) > max_bytes:
            return None
        b64 = base64.standard_b64encode(data).decode("ascii")
        return f"data:{mime};base64,{b64}"
    except Exception as e:
        logger.debug("Data URI %s: %s", path, e)
        return None


def _mime_for_path(path: Path, kind: str) -> str:
    ext = (path.suffix or "").lower()
    if kind == "video":
        if ext in (".webm",):
            return "video/webm"
        if ext in (".mkv",):
            return "video/x-matroska"
        return "video/mp4"
    if ext in (".png",):
        return "image/png"
    if ext in (
        ".jpg",
        ".jpeg",
    ):
        return "image/jpeg"
    if ext in (".webp",):
        return "image/webp"
    return "image/png"


def build_thumbnail_html(  # noqa: PLR0913
    *,
    static_src: str,
    hover_src: str | None,
    lightbox_src: str,
    kind: str,
    width: int = 200,
    height: int = 200,
    element_id: str,
    include_lightbox: bool = True,
) -> str:
    """Construit le HTML complet : zone thumbnail (static + survol) + lightbox optionnelle.

    - Affichage par défaut : static_src.
    - Au survol uniquement : chargement et affichage de hover_src (GIF ne charge qu'au hover).
    - Ratio uniforme : boîte fixe 16:9 avec aspect-ratio, object-fit: cover.
    - Au clic : ouverture du lightbox si ``include_lightbox=True``.
    """
    container_id = f"media-thumb-{element_id}"
    static_id = f"thumb-static-{element_id}"
    hover_id = f"thumb-hover-{element_id}"

    safe_static = static_src.replace("\\", "\\\\").replace('"', "&quot;").replace("<", "&lt;")
    # hover_src n'est pas mis dans le src initial : on l'injecte au mouseenter pour que le GIF ne s'anime qu'au survol
    hover_js = json.dumps(hover_src) if hover_src else "null"
    hover_img = (
        f'<img id="{hover_id}" src="" alt="" style="width:100%;height:100%;object-fit:cover;object-position:center;position:absolute;top:0;left:0;display:none;" />'
        if hover_src
        else ""
    )

    overlay_id = f"media-lightbox-{element_id}" if include_lightbox else None
    lightbox_html = ""
    if include_lightbox:
        lightbox_html = build_lightbox_html(
            media_src=lightbox_src,
            kind=kind,
            element_id=element_id,
        )

    # Ratio 16:9 pour toutes les captures (grille alignée), contenu recadré
    return _build_thumbnail_container_html(
        container_id=container_id,
        width=width,
        static_id=static_id,
        safe_static=safe_static,
        hover_img=hover_img,
        lightbox_html=lightbox_html,
        hover_id=hover_id,
        overlay_id=overlay_id,
        hover_js=hover_js,
        include_lightbox=include_lightbox,
    )


def _build_thumbnail_container_html(  # noqa: PLR0913
    *,
    container_id: str,
    width: int,
    static_id: str,
    safe_static: str,
    hover_img: str,
    lightbox_html: str,
    hover_id: str,
    overlay_id: str | None,
    hover_js: str,
    include_lightbox: bool,
) -> str:
    """Génère le HTML complet du conteneur thumbnail + lightbox + JS."""
    cursor_style = "pointer" if include_lightbox else "default"
    overlay_lookup = f"document.getElementById('{overlay_id}')" if overlay_id else "null"
    click_handler = (
        "  c.addEventListener('click', function() {\n    overlay.style.display = 'flex';\n  });\n"
        if include_lightbox
        else ""
    )
    return f"""<style>html,body{{margin:0;padding:0;overflow:hidden;}}</style>
<div id="{container_id}" class="media-thumb-container" style="
    width: 100%;
    max-width: {width}px;
    aspect-ratio: 16 / 9;
    position: relative;
    overflow: hidden;
    border-radius: 8px;
    cursor: {cursor_style};
    background: #1a1a1a;
    flex-shrink: 0;
    margin: 0 auto;
">
    <img id="{static_id}" src="{safe_static}" alt="" style="width:100%;height:100%;object-fit:cover;object-position:center;display:block;" />
    {hover_img}
</div>
{lightbox_html}
<script>
(function() {{
    var c = document.getElementById('{container_id}');
    var s = document.getElementById('{static_id}');
    var h = document.getElementById('{hover_id}');
    var overlay = {overlay_lookup};
    var hoverSrc = {hover_js};
    if (!c) return;
    c.addEventListener('mouseenter', function() {{
        if (h && hoverSrc) {{
            h.src = hoverSrc;
            s.style.display = 'none';
            h.style.display = 'block';
        }}
    }});
    c.addEventListener('mouseleave', function() {{
        if (h) {{
            h.src = '';
            h.style.display = 'none';
            s.style.display = 'block';
        }}
    }});
{click_handler}}})();
</script>
"""


def render_media_thumbnail(  # noqa: PLR0913
    *,
    static_path: Path | str,
    hover_path: Path | str | None = None,
    full_media_path: Path | str | None = None,
    kind: str = "image",
    width: int = 200,
    height: int = 200,
    media_id: str | None = None,
    include_lightbox: bool = True,
    height_iframe: int | None = None,
) -> None:
    """Affiche un thumbnail média dans Streamlit (dimensions fixes, survol animé, lightbox optionnelle).

    Args:
        static_path: Chemin de la miniature statique (affichée par défaut).
        hover_path: Chemin de la version « survol » (GIF animé ou image). Si absent, pas de changement au survol.
        full_media_path: Chemin du média plein (lightbox). Si absent ou fichier trop gros, utilise la miniature.
        kind: "image" ou "video" (pour le lecteur dans le lightbox).
        width: Largeur du thumbnail (px).
        height: Hauteur du thumbnail (px).
        media_id: Identifiant unique (pour clés Streamlit / ids HTML). Généré si absent.
        include_lightbox: Active l'overlay embarqué et l'encodage du média complet.
        height_iframe: Hauteur de l'iframe HTML (défaut: height + marge).
    """
    static_path = Path(static_path)
    if not static_path.exists():
        st.caption(f"Fichier absent : {static_path.name}")
        return

    element_id = media_id or hashlib.md5(str(static_path.resolve()).encode()).hexdigest()[:12]
    static_uri = _resolve_static_thumbnail_uri(static_path, hover_path)
    if not static_uri:
        _render_large_thumbnail_fallback(static_path, width)
        return

    hover_uri = _resolve_hover_thumbnail_uri(hover_path)
    lightbox_uri, lightbox_kind = _resolve_lightbox_media(
        static_uri=static_uri,
        static_path=static_path,
        full_media_path=full_media_path,
        kind=kind,
        include_lightbox=include_lightbox,
    )
    html = build_thumbnail_html(
        static_src=static_uri,
        hover_src=hover_uri,
        lightbox_src=lightbox_uri,
        kind=lightbox_kind,
        width=width,
        height=height,
        element_id=element_id,
        include_lightbox=include_lightbox,
    )
    iframe_h = height_iframe if height_iframe is not None else height + 24
    st.components.v1.html(html, height=iframe_h)


def _resolve_static_thumbnail_uri(static_path: Path, hover_path: Path | str | None) -> str | None:
    """Construit la data URI de la miniature statique, avec support GIF première frame."""
    is_gif_thumbnail = str(static_path).lower().endswith(".gif") and hover_path
    static_uri: str | None = None
    if is_gif_thumbnail:
        static_uri = _gif_first_frame_to_data_uri(static_path)
    if static_uri:
        return static_uri
    return _path_to_data_uri(static_path, MAX_STATIC_BYTES, _mime_for_path(static_path, "image"))


def _render_large_thumbnail_fallback(static_path: Path, width: int) -> None:
    """Affiche un fallback natif Streamlit si la data URI est trop volumineuse."""
    try:
        st.image(str(static_path), width=width)
    except Exception:
        st.caption(f"Miniature trop volumineuse : {static_path.name}")


def _resolve_hover_thumbnail_uri(hover_path: Path | str | None) -> str | None:
    """Construit la data URI de la miniature affichée au survol, si présente."""
    if not hover_path:
        return None

    resolved_hover = Path(hover_path)
    if not resolved_hover.exists():
        return None
    return _path_to_data_uri(
        resolved_hover,
        MAX_HOVER_BYTES,
        "image/gif"
        if str(resolved_hover).lower().endswith(".gif")
        else _mime_for_path(resolved_hover, "image"),
    )


def _resolve_lightbox_media(
    *,
    static_uri: str,
    static_path: Path,
    full_media_path: Path | str | None,
    kind: str,
    include_lightbox: bool,
) -> tuple[str, str]:
    """Retourne la source et le type média à injecter dans la lightbox."""
    lightbox_uri = static_uri
    lightbox_kind = "image"
    if not include_lightbox:
        return lightbox_uri, lightbox_kind

    full_path = Path(full_media_path) if full_media_path else static_path
    lightbox_uri = _path_to_data_uri(
        full_path,
        MAX_LIGHTBOX_BYTES,
        _mime_for_path(full_path, kind),
    )
    if not lightbox_uri:
        return static_uri, "image"
    return lightbox_uri, kind
