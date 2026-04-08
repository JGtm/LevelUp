"""LevelUp - Analyse des statistiques Halo Infinite."""

from importlib.metadata import PackageNotFoundError
from importlib.metadata import version as _pkg_version

try:
    __version__ = _pkg_version("levelup-halo")
except PackageNotFoundError:
    __version__ = "dev"
