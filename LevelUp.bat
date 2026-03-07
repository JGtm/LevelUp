@echo off
chcp 65001 >nul 2>&1
setlocal EnableDelayedExpansion

:: ============================================================================
:: LevelUp - Lanceur Windows
::
:: Double-cliquez sur ce fichier pour lancer LevelUp.
:: Au premier lancement, l'environnement est configuré automatiquement.
:: ============================================================================

title LevelUp - Halo Infinite Dashboard

:: Se placer dans le dossier du script
cd /d "%~dp0"

:: ── Chercher Python dans .venv (déjà configuré) ──
if exist ".venv\Scripts\python.exe" (
    .venv\Scripts\python.exe launcher.py run
    goto :end
)

:: ── Premier lancement : setup automatique ──
echo.
echo  ╔══════════════════════════════════════════╗
echo  ║     LevelUp - Premier lancement          ║
echo  ╚══════════════════════════════════════════╝
echo.

:: Chercher Python sur le système
set "PY="

:: Essayer py (Windows Python Launcher)
where py >nul 2>&1
if %ERRORLEVEL% equ 0 (
    for /f "tokens=*" %%i in ('py -3.12 -c "import sys; print(sys.executable)" 2^>nul') do set "PY=%%i"
    if not defined PY (
        for /f "tokens=*" %%i in ('py -3 -c "import sys; print(sys.executable)" 2^>nul') do set "PY=%%i"
    )
)

:: Essayer python dans le PATH
if not defined PY (
    where python >nul 2>&1
    if %ERRORLEVEL% equ 0 (
        for /f "tokens=*" %%i in ('python -c "import sys; print(sys.executable)" 2^>nul') do set "PY=%%i"
    )
)

:: Chercher dans les emplacements standards
if not defined PY (
    for %%P in (
        "%LOCALAPPDATA%\Programs\Python\Python312\python.exe"
        "%LOCALAPPDATA%\Programs\Python\Python313\python.exe"
        "%LOCALAPPDATA%\Programs\Python\Python311\python.exe"
    ) do (
        if exist %%P (
            set "PY=%%~P"
            goto :found_python
        )
    )
)

:: Python non trouvé → proposer winget
if not defined PY (
    echo   Python non trouve sur le systeme.
    echo.
    echo   LevelUp necessite Python 3.10 ou superieur.
    echo   Voulez-vous installer Python 3.12 automatiquement ?
    echo.
    choice /C ON /N /M "   [O]ui / [N]on : "
    if !ERRORLEVEL! equ 2 (
        echo   Installez Python depuis https://www.python.org/downloads/
        pause
        exit /b 1
    )
    winget install --id Python.Python.3.12 --scope user --accept-source-agreements --accept-package-agreements
    if !ERRORLEVEL! neq 0 (
        echo   Echec. Installez Python depuis https://www.python.org/downloads/
        pause
        exit /b 1
    )
    set "PATH=%LOCALAPPDATA%\Programs\Python\Python312\;%LOCALAPPDATA%\Programs\Python\Python312\Scripts\;%PATH%"
    set "PY=%LOCALAPPDATA%\Programs\Python\Python312\python.exe"
)

:found_python
echo   Python: !PY!

:: Créer le venv et installer les dépendances
echo   Creation de l'environnement...
"!PY!" -m venv .venv
if %ERRORLEVEL% neq 0 (
    echo   Erreur lors de la creation du venv.
    pause
    exit /b 1
)

.venv\Scripts\python.exe -m pip install --upgrade pip -q
.venv\Scripts\python.exe -m pip install -e ".[spnkr]" -q
if %ERRORLEVEL% neq 0 (
    echo   Erreur lors de l'installation des dependances.
    pause
    exit /b 1
)

echo   OK - Environnement pret.
echo.

:: Lancer le dashboard
.venv\Scripts\python.exe launcher.py run

:end
