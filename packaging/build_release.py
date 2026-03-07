"""Build script pour créer une release portable LevelUp.

Crée un zip contenant :
- Python Embeddable (~15 MB)
- Le code source LevelUp
- LevelUp.bat (lanceur double-clic)

Usage :
    python packaging/build_release.py [--version 5.5.0]
"""

from __future__ import annotations

import argparse
import hashlib
import io
import logging
import shutil
import stat
import tarfile
import zipfile
from pathlib import Path
from urllib.request import urlopen

logger = logging.getLogger(__name__)

REPO_ROOT = Path(__file__).resolve().parent.parent
PYTHON_EMBED_VERSION = "3.12.10"
PYTHON_EMBED_URL = (
    f"https://www.python.org/ftp/python/{PYTHON_EMBED_VERSION}"
    f"/python-{PYTHON_EMBED_VERSION}-embed-amd64.zip"
)
# MD5 officiel depuis python.org/downloads/release/python-31210/
PYTHON_EMBED_MD5 = "fe8ef205f2e9c3ba44d0cf9954e1abd3"  # pragma: allowlist secret

# Fichiers/dossiers à inclure dans la release
INCLUDE_FILES = [
    "launcher.py",
    "streamlit_app.py",
    "LevelUp.bat",
    "pyproject.toml",
    "db_profiles.json",
    "app_settings.example.json",
    "run.sh",
]

# Fichiers pour la release Unix (macOS / Linux) — sans LevelUp.bat
INCLUDE_FILES_UNIX = [
    "launcher.py",
    "streamlit_app.py",
    "LevelUp.sh",
    "pyproject.toml",
    "db_profiles.json",
    "app_settings.example.json",
    "run.sh",
]

INCLUDE_DIRS = [
    "src",
    "static",
    "scripts",
]

# Scripts shell à marquer exécutables dans le tarball
_EXECUTABLE_SUFFIXES = {".sh"}


def _read_version() -> str:
    """Lit la version depuis pyproject.toml."""
    toml_path = REPO_ROOT / "pyproject.toml"
    for line in toml_path.read_text(encoding="utf-8").splitlines():
        if line.strip().startswith("version"):
            return line.split("=")[1].strip().strip('"')
    return "0.0.0"


def _download_python_embed(target_dir: Path) -> None:
    """Télécharge et extrait Python Embeddable avec vérification d'intégrité."""
    logger.info("Téléchargement Python Embeddable %s...", PYTHON_EMBED_VERSION)
    print(f"  → Téléchargement Python Embeddable {PYTHON_EMBED_VERSION}...")
    resp = urlopen(PYTHON_EMBED_URL, timeout=120)  # noqa: S310 — URL fixe python.org
    raw = resp.read()

    # Vérification d'intégrité MD5 (hash officiel python.org)
    actual_md5 = hashlib.md5(raw).hexdigest()  # noqa: S324
    if actual_md5 != PYTHON_EMBED_MD5:
        msg = (
            f"Intégrité Python Embeddable compromise : "
            f"MD5 attendu {PYTHON_EMBED_MD5}, obtenu {actual_md5}"
        )
        logger.error(msg)
        raise ValueError(msg)
    logger.info("Intégrité vérifiée (MD5 OK)")

    data = io.BytesIO(raw)

    python_dir = target_dir / "python"
    python_dir.mkdir(parents=True, exist_ok=True)

    with zipfile.ZipFile(data) as zf:
        zf.extractall(python_dir)

    # Activer pip : décommenter import site dans python312._pth
    pth_files = list(python_dir.glob("python*._pth"))
    for pth in pth_files:
        content = pth.read_text(encoding="utf-8")
        content = content.replace("#import site", "import site")
        pth.write_text(content, encoding="utf-8")

    logger.info("Python Embeddable extrait dans %s", python_dir)
    print(f"  ✓ Python Embeddable extrait ({python_dir})")


def _copy_project(target_dir: Path, include_files: list[str] | None = None) -> None:
    """Copie les fichiers du projet."""
    print("  → Copie des fichiers du projet...")
    files = include_files if include_files is not None else INCLUDE_FILES

    for f in files:
        src = REPO_ROOT / f
        if src.exists():
            shutil.copy2(src, target_dir / f)

    for d in INCLUDE_DIRS:
        src_dir = REPO_ROOT / d
        if not src_dir.exists():
            continue
        dst_dir = target_dir / d
        shutil.copytree(
            src_dir,
            dst_dir,
            ignore=shutil.ignore_patterns("__pycache__", "*.pyc", "*.pyo", ".git"),
        )

    # Créer les dossiers de données vides
    (target_dir / "data" / "players").mkdir(parents=True, exist_ok=True)
    (target_dir / "data" / "warehouse").mkdir(parents=True, exist_ok=True)

    print("  ✓ Fichiers copiés")


def _create_portable_bat(target_dir: Path) -> None:
    """Crée un LevelUp.bat adapté au mode portable (Python Embeddable)."""
    bat_content = r"""@echo off
chcp 65001 >nul 2>&1
setlocal EnableDelayedExpansion

title LevelUp - Halo Infinite Dashboard
cd /d "%~dp0"

set "PYTHON=%~dp0python\python.exe"

:: Premier lancement : installer les dépendances
if not exist ".deps_installed" (
    echo.
    echo  Installation des dependances ^(premier lancement^)...
    echo.
    "%PYTHON%" -m ensurepip --upgrade -q 2>nul
    "%PYTHON%" -m pip install --upgrade pip -q
    "%PYTHON%" -m pip install -e ".[spnkr]" -q
    if !ERRORLEVEL! neq 0 (
        echo   Erreur lors de l'installation.
        pause
        exit /b 1
    )
    echo OK > .deps_installed
    echo   OK - Dependances installees.
    echo.
)

"%PYTHON%" launcher.py run
"""
    (target_dir / "LevelUp.bat").write_text(bat_content, encoding="utf-8")
    print("  ✓ LevelUp.bat portable créé")


def _create_zip(target_dir: Path, version: str) -> Path:
    """Crée le zip final (Windows portable)."""
    zip_name = f"LevelUp-v{version}-win64-portable"
    zip_path = REPO_ROOT / "dist" / f"{zip_name}.zip"
    zip_path.parent.mkdir(parents=True, exist_ok=True)

    print(f"  → Création de {zip_path.name}...")

    with zipfile.ZipFile(zip_path, "w", zipfile.ZIP_DEFLATED) as zf:
        for file in sorted(target_dir.rglob("*")):
            if file.is_file():
                arcname = f"{zip_name}/{file.relative_to(target_dir)}"
                zf.write(file, arcname)

    size_mb = zip_path.stat().st_size / (1024 * 1024)
    print(f"  ✓ {zip_path.name} ({size_mb:.1f} MB)")
    return zip_path


def _create_unix_tarball(target_dir: Path, version: str) -> Path:
    """Crée l'archive tar.gz pour macOS/Linux avec permissions exécutables sur les .sh."""
    tar_name = f"LevelUp-v{version}-unix"
    tar_path = REPO_ROOT / "dist" / f"{tar_name}.tar.gz"
    tar_path.parent.mkdir(parents=True, exist_ok=True)

    print(f"  → Création de {tar_path.name}...")

    with tarfile.open(tar_path, "w:gz") as tf:
        for file in sorted(target_dir.rglob("*")):
            if not file.is_file():
                continue
            arcname = f"{tar_name}/{file.relative_to(target_dir)}"
            info = tf.gettarinfo(str(file), arcname=arcname)
            # Scripts shell doivent être exécutables
            if file.suffix in _EXECUTABLE_SUFFIXES:
                info.mode = stat.S_IFREG | 0o755
            else:
                info.mode = stat.S_IFREG | 0o644
            with file.open("rb") as fh:
                tf.addfile(info, fh)

    size_mb = tar_path.stat().st_size / (1024 * 1024)
    print(f"  ✓ {tar_path.name} ({size_mb:.1f} MB)")
    return tar_path


def _build_win64(version: str) -> Path:
    """Construit la release portable Windows."""
    build_dir = REPO_ROOT / "dist" / f"_build_win64_v{version}"
    if build_dir.exists():
        shutil.rmtree(build_dir)
    build_dir.mkdir(parents=True)

    _download_python_embed(build_dir)
    _copy_project(build_dir)
    _create_portable_bat(build_dir)
    artifact = _create_zip(build_dir, version)

    shutil.rmtree(build_dir)
    return artifact


def _build_unix(version: str) -> Path:
    """Construit la release source Unix (macOS / Linux)."""
    build_dir = REPO_ROOT / "dist" / f"_build_unix_v{version}"
    if build_dir.exists():
        shutil.rmtree(build_dir)
    build_dir.mkdir(parents=True)

    _copy_project(build_dir, include_files=INCLUDE_FILES_UNIX)
    artifact = _create_unix_tarball(build_dir, version)

    shutil.rmtree(build_dir)
    return artifact


def main() -> int:
    """Point d'entrée du build."""
    logging.basicConfig(level=logging.INFO, format="%(levelname)s: %(message)s")
    ap = argparse.ArgumentParser(description="Build LevelUp portable release")
    ap.add_argument("--version", default=None, help="Version (défaut: depuis pyproject.toml)")
    ap.add_argument(
        "--platform",
        choices=["win64", "unix"],
        default="win64",
        help="Plateforme cible (défaut: win64)",
    )
    args = ap.parse_args()

    version = args.version or _read_version()

    print("=" * 60)
    print(f"📦 BUILD RELEASE — LevelUp v{version} [{args.platform}]")
    print("=" * 60)

    if args.platform == "win64":
        artifact = _build_win64(version)
        print("  L'utilisateur n'a qu'à extraire le zip et double-cliquer LevelUp.bat")
    else:
        artifact = _build_unix(version)
        print("  L'utilisateur n'a qu'à extraire l'archive et lancer ./LevelUp.sh")

    print("\n" + "=" * 60)
    print("✅ BUILD TERMINÉ")
    print("=" * 60)
    print(f"\n  Release: {artifact}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
