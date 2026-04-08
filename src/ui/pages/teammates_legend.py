"""Panneau de légende joueurs pour la page Coéquipiers.

Pour changer de stratégie de positionnement, modifier ``_PANEL_MODE`` :

- ``"fixed"``   → ``position:fixed`` côté droit, scroll-conditionné ✅ (défaut)
- ``"sidebar"`` → ``st.sidebar``, très simple à tester
- ``"hidden"``  → désactivé (debug / comparaison visuelle)
"""

from __future__ import annotations

import logging

import streamlit as st
import streamlit.components.v1 as components

from src.ui.i18n import get_lang

logger = logging.getLogger(__name__)

# ── Stratégie ─────────────────────────────────────────────────────────────────
# Changer cette valeur pour tester une autre approche sans toucher au reste.
_PANEL_MODE: str = "fixed"

# CSS du panneau fixe — modifier uniquement ce bloc pour ajuster l'apparence.
_FIXED_STYLE: str = (
    "position:fixed;"
    "right:1.2rem;"
    "top:50%;"
    "transform:translateY(-50%);"
    "z-index:999;"
    "background:rgba(14,17,23,0.92);"
    "border:1px solid rgba(255,255,255,0.14);"
    "border-radius:10px;"
    "padding:10px 14px;"
    "backdrop-filter:blur(6px);"
    "max-width:150px;"
    "font-size:13px;"
    "color:white;"
    "line-height:1.6;"
)

_LABEL_FR = "Joueurs"
_LABEL_EN = "Players"

# JS injecté via components.html(height=0) — observe deux sentinelles DOM :
# - #llp-squad-start  : fin de l'en-tête Escouade (injecté par _render_fixed)
# - #llp-impact-end   : début de la section Impact (injecté par teammates_impact)
# Le panneau n'est visible qu'entre les deux.
_SCROLL_OBSERVER_JS: str = """<script>
(function(){
  var T=0;
  function run(){
    try{
      var d=window.parent.document;
      var p=d.getElementById('llp-fixed-panel');
      var s=d.getElementById('llp-squad-start');
      if(!p||!s){if(T++<30)setTimeout(run,150);return;}
      var e=d.getElementById('llp-impact-end');
      function upd(){
        var iH=window.parent.innerHeight||768;
        var ok=s.getBoundingClientRect().top<iH;
        if(ok&&e)ok=e.getBoundingClientRect().top>50;
        p.style.display=ok?'block':'none';
      }
      var mc=d.querySelector('[data-testid="stMain"]')||d.querySelector('.main');
      if(mc)mc.addEventListener('scroll',upd,{passive:true});
      window.parent.addEventListener('scroll',upd,{passive:true});
      upd();
    }catch(ex){if(T++<30)setTimeout(run,150);}
  }
  run();
})();
</script>"""


def render_player_legend_panel(colors_by_name: dict[str, str]) -> None:
    """Injecte le panneau de légende joueurs selon la stratégie ``_PANEL_MODE``."""
    if not colors_by_name:
        logger.debug("render_player_legend_panel: colors_by_name vide, panneau ignoré")
        return

    label = _LABEL_FR if get_lang() == "fr" else _LABEL_EN

    if _PANEL_MODE == "hidden":
        logger.debug("render_player_legend_panel: mode 'hidden', panneau désactivé")
        return

    if _PANEL_MODE == "sidebar":
        logger.debug("render_player_legend_panel: mode 'sidebar', %d joueurs", len(colors_by_name))
        _render_sidebar(colors_by_name, label)
        return

    # "fixed" (défaut)
    logger.debug("render_player_legend_panel: mode 'fixed', %d joueurs", len(colors_by_name))
    _render_fixed(colors_by_name, label)


def _render_fixed(colors_by_name: dict[str, str], label: str) -> None:
    """Panneau ``position:fixed`` c\u00f4t\u00e9 droit \u2014 visible entre Escouade et Impact.

    Le panneau d\u00e9marre cach\u00e9 (``display:none``). Un observateur JS pilote sa
    visibilit\u00e9 selon la position de deux sentinelles DOM :
    ``#llp-squad-start`` (inject\u00e9e ici) et ``#llp-impact-end`` (inject\u00e9e dans
    ``render_impact_taquinerie``).
    """
    dots = "".join(
        f'<div style="display:flex;align-items:center;gap:8px;margin:3px 0;">'
        f'<span style="width:11px;height:11px;border-radius:50%;background:{color};'
        f'display:inline-block;flex-shrink:0;"></span>'
        f'<span style="white-space:nowrap;overflow:hidden;text-overflow:ellipsis;'
        f'max-width:105px;" title="{name}">{name}</span>'
        f"</div>"
        for name, color in colors_by_name.items()
    )
    header = (
        f'<div style="font-weight:600;font-size:10px;text-transform:uppercase;'
        f'letter-spacing:.08em;color:rgba(255,255,255,.5);margin-bottom:6px;">'
        f"{label}</div>"
    )
    # Sentinelle de d\u00e9but + panneau cach\u00e9 initialement (le JS g\u00e8re la visibilit\u00e9)
    html = (
        '<div id="llp-squad-start" style="height:0;line-height:0;overflow:hidden;'
        'padding:0;margin:0;border:0;"></div>'
        f'<div id="llp-fixed-panel" style="display:none;{_FIXED_STYLE}">{header}{dots}</div>'
    )
    st.markdown(html, unsafe_allow_html=True)
    components.html(_SCROLL_OBSERVER_JS, height=0)


def _render_sidebar(colors_by_name: dict[str, str], label: str) -> None:
    """Alternative sidebar — changer ``_PANEL_MODE = 'sidebar'`` pour tester."""
    st.sidebar.markdown("---")
    st.sidebar.markdown(f"**{label}**")
    for name, color in colors_by_name.items():
        dot = (
            f'<span style="background:{color};width:10px;height:10px;'
            f'border-radius:50%;display:inline-block;margin-right:6px;"></span>'
        )
        st.sidebar.markdown(f"{dot}{name}", unsafe_allow_html=True)
