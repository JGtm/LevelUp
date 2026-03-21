"""Composant UI Streamlit — Device Code Flow MSAL (setup wizard).

La logique métier est dans ``src/utils/msal_device_flow.py`` et
``src/ui/xbox_oauth.py`` (testables sans Streamlit).
"""

from __future__ import annotations

import logging
import queue
import threading

import streamlit as st

from src.app.session_keys import SK

logger = logging.getLogger(__name__)

# Clés session_state internes (module-private)
_DC_FLOW_KEY = SK.DC_FLOW
_DC_APP_KEY = SK.DC_MSAL_APP
_DC_QUEUE_KEY = SK.DC_RESULT_QUEUE
_DC_CLIENT_ID_KEY = SK.DC_CLIENT_ID


def check_dc_queue() -> dict | None:
    """Vérifie la queue du device flow sans bloquer. Retourne le résultat si prêt."""
    q = st.session_state.get(_DC_QUEUE_KEY)
    if q is None:
        return None
    try:
        return q.get_nowait()
    except queue.Empty:
        return None


def reset_device_flow() -> None:
    """Nettoie l'état du device flow en cours."""
    for key in (_DC_FLOW_KEY, _DC_APP_KEY, _DC_QUEUE_KEY, _DC_CLIENT_ID_KEY):
        st.session_state.pop(key, None)


def start_device_flow(client_id: str) -> None:
    """Initie le device flow MSAL et démarre le thread de polling."""
    from src.utils.msal_device_flow import (
        DeviceFlowError,
        acquire_token_blocking,
        initiate_device_flow,
    )

    result, app = initiate_device_flow(client_id)
    q: queue.Queue[dict] = queue.Queue(maxsize=1)

    def _poll() -> None:
        try:
            token = acquire_token_blocking(app, result._flow)
            q.put({"refresh_token": token})
            logger.info("Device flow thread : token obtenu (client_id=%s…)", client_id[:8])
        except DeviceFlowError as exc:
            logger.warning("Device flow thread : erreur=%s — %s", exc.code, exc.detail)
            q.put({"error": exc.code, "detail": exc.detail})
        except Exception as exc:  # noqa: BLE001
            logger.error("Device flow thread : exception inattendue: %s", exc)
            q.put({"error": "unknown", "detail": str(exc)})

    thread = threading.Thread(target=_poll, daemon=True, name="msal-device-flow")
    thread.start()

    st.session_state[_DC_APP_KEY] = app
    st.session_state[_DC_FLOW_KEY] = result
    st.session_state[_DC_QUEUE_KEY] = q
    st.session_state[_DC_CLIENT_ID_KEY] = client_id
    logger.info("Device flow initié (client_id=%s… user_code=%s)", client_id[:8], result.user_code)
