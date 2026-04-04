"""Tests unitaires pour src/utils/strings.py.

Couvre :
- is_uuid_like : détection UUID complet / partiel / négatifs
"""

from __future__ import annotations

import pytest

from src.utils.strings import is_uuid_like


class TestIsUuidLike:
    """Tests directs sur src.utils.strings.is_uuid_like."""

    def test_full_uuid(self) -> None:
        assert is_uuid_like("a446725e-b281-414c-a21e-123456789abc") is True

    def test_partial_uuid_three_groups(self) -> None:
        assert is_uuid_like("a446725e-b281-414c") is True

    def test_partial_uuid_two_groups(self) -> None:
        assert is_uuid_like("a446725e-b281") is True

    def test_partial_uuid_one_group(self) -> None:
        assert is_uuid_like("a446725e") is True

    def test_uppercase_accepted(self) -> None:
        assert is_uuid_like("A446725E-B281-414C-A21E-123456789ABC") is True

    def test_map_name_rejected(self) -> None:
        assert is_uuid_like("Aquarius") is False

    def test_playlist_name_rejected(self) -> None:
        assert is_uuid_like("Quick Play") is False

    def test_mode_name_rejected(self) -> None:
        assert is_uuid_like("Arena:4v4 Slayer") is False

    def test_empty_string_rejected(self) -> None:
        assert is_uuid_like("") is False

    def test_numeric_string_rejected(self) -> None:
        assert is_uuid_like("12345") is False

    def test_uuid_with_trailing_space_rejected(self) -> None:
        # Le regex ne matche pas si la chaîne contient un espace
        assert is_uuid_like("a446725e ") is False
