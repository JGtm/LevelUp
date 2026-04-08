"""Tests hero HTML : adornment et spartan ID dans la nameplate.

Vérifie que get_hero_html affiche l'adornment intégré à la nameplate
(spartan-id__adornment) et que la section career-rank gauche a été supprimée.
"""

from __future__ import annotations

from pathlib import Path

from src.ui.styles import get_hero_html


def _create_fake_image(tmp_dir: Path, name: str) -> str:
    """Crée un fichier PNG minimal pour les tests."""
    # PNG 1x1 transparent minimal
    png_header = (
        b"\x89PNG\r\n\x1a\n"
        b"\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01"
        b"\x08\x06\x00\x00\x00\x1f\x15\xc4\x89"
        b"\x00\x00\x00\nIDATx\x9cc\x00\x01\x00\x00\x05\x00\x01"
        b"\r\n\xb4\x00\x00\x00\x00IEND\xaeB`\x82"
    )
    path = tmp_dir / name
    path.write_bytes(png_header)
    return str(path)


class TestHeroAdornmentPriority:
    """Adornment affiché dans spartan-id__adornment (côté droit de la nameplate)."""

    def test_adornment_shown_when_available(self, tmp_path: Path) -> None:
        """Quand adornment_path est fourni, spartan-id__adornment est présent."""
        adornment = _create_fake_image(tmp_path, "adornment.png")

        html = get_hero_html(
            player_name="Spartan",
            adornment_path=adornment,
        )

        assert "spartan-id__adornment" in html
        # Bloc career-rank gauche supprimé
        assert "career-rank__adornment--primary" not in html

    def test_rank_icon_fallback_when_no_adornment(self, tmp_path: Path) -> None:
        """Sans adornment, spartan-id__adornment n'est pas rendu."""
        html = get_hero_html(
            player_name="Spartan",
            adornment_path=None,
        )

        assert "spartan-id__adornment" not in html
        assert "career-rank__icon" not in html

    def test_no_rank_section_without_data(self) -> None:
        """Sans adornment, aucun spartan-id__adornment rendu."""
        html = get_hero_html(
            player_name="Spartan",
            adornment_path=None,
        )

        assert "spartan-id__adornment" not in html
        assert "career-rank" not in html

    def test_adornment_only_no_rank_icon(self, tmp_path: Path) -> None:
        """Adornment seul → spartan-id__adornment affiché."""
        adornment = _create_fake_image(tmp_path, "adornment.png")

        html = get_hero_html(
            player_name="Spartan",
            adornment_path=adornment,
        )

        assert "spartan-id__adornment" in html
        assert "career-rank__icon" not in html

    def test_label_only_without_icons(self) -> None:
        """Aucun rank dans la signature → aucun élément career-rank rendu."""
        html = get_hero_html(
            player_name="Spartan",
        )

        assert "career-rank__label" not in html
        assert "career-rank" not in html

    def test_empty_player_name_returns_default(self) -> None:
        """Sans nom de joueur, le hero par défaut est retourné."""
        html = get_hero_html(player_name="")
        assert "LevelUp" in html
        assert "career-rank" not in html


class TestHeroSpartanId:
    """Affichage du spartan_id — career-rank supprimé, spartan_id ignoré."""

    def test_spartan_id_not_in_career_rank(self) -> None:
        """career-rank__spartan-id absent (section career-rank supprimée)."""
        html = get_hero_html(
            player_name="Spartan",
        )
        assert "career-rank__spartan-id" not in html

    def test_spartan_id_not_rendered_when_none(self) -> None:
        """Pas de section career-rank (param spartan_id supprimé depuis la v7)."""
        html = get_hero_html(
            player_name="Spartan",
        )
        assert "career-rank__spartan-id" not in html

    def test_spartan_id_escaped(self) -> None:
        """Les caractères spéciaux dans le gamertag sont échappés."""
        html = get_hero_html(
            player_name="<script>XSS</script>",
        )
        assert "<script>" not in html
        assert "&lt;script&gt;" in html
