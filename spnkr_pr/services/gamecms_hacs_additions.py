"""Additions to spnkr/services/gamecms_hacs.py — inventory item metadata.

One new method: get_item_metadata.
In the actual PR it is appended directly into GameCmsHacsService in gamecms_hacs.py.
"""

from spnkr.responses import JsonResponse
from spnkr.services.base import BaseService

# In the actual PR: from spnkr.models.gamecms_hacs import InventoryItem, OperationRewardTrackMeta
from spnkr_pr.models.gamecms_hacs_additions import InventoryItem, OperationRewardTrackMeta

_HOST = "https://gamecms-hacs.svc.halowaypoint.com"


class GameCmsHacsServiceExtension(BaseService):
    """Extension of GameCmsHacsService adding inventory item metadata and
    operation reward track metadata.

    In the actual PR these methods are merged directly into GameCmsHacsService.
    """

    async def get_item_metadata(
        self,
        inventory_item_path: str,
    ) -> JsonResponse[InventoryItem]:
        """Get metadata for an inventory item (name, description, preview image path…).

        The ``inventory_item_path`` is any ``.json`` path returned by the economy
        service in customization fields, e.g.:
        - ``"Inventory/Armor/Cores/013-001-olympus-c13d0f2f.json"``
        - ``"Inventory/Armor/Coatings/007-001-...json"``
        - ``"Inventory/Weapons/Cores/...json"``

        Use the returned ``common_data.display_path.display_path.media_url`` with
        ``get_image()`` to fetch the item's preview image bytes.

        Args:
            inventory_item_path: Relative path to the item JSON on gamecms_hacs.
                Leading slashes are stripped automatically.

        Returns:
            Parsed inventory item metadata including id, title, quality, and
            the image path needed for ``get_image()``.
        """
        url = f"{_HOST}/hi/Waypoint/file/{inventory_item_path.lstrip('/')}"
        resp = await self._get(url)
        return JsonResponse(resp, lambda data: InventoryItem(**data))

    async def get_operation_reward_track(
        self,
        reward_track_path: str,
    ) -> JsonResponse[OperationRewardTrackMeta]:
        """Récupère les métadonnées d'un reward track d'opération (battlepass).

        Utilise ``summary_image_path`` du résultat avec ``get_image()`` pour
        afficher l'artwork de l'opération en cours.

        Args:
            reward_track_path: Chemin relatif retourné par l'API Economy,
                ex. ``"Operations/Season6Operations-1.json"``.

        Returns:
            Métadonnées minimales incluant ``summary_image_path``.
        """
        url = f"{_HOST}/hi/Progression/file/{reward_track_path.lstrip('/')}"
        resp = await self._get(url)
        return JsonResponse(resp, lambda data: OperationRewardTrackMeta(**data))

    async def get_image_bytes(self, relative_path: str) -> bytes:
        """Wrapper pour récupérer les octets d'une image GameCMS.

        Args:
            relative_path: Chemin relatif passable à ``get_image()``.

        Returns:
            Données brutes de l'image (PNG/JPEG).
        """
        url = f"{_HOST}/hi/images/file/{relative_path.lstrip('/')}"
        resp = await self._get(url)
        return await resp.read()
