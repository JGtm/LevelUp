@echo off
chcp 65001 >nul 2>&1
setlocal EnableDelayedExpansion

:: ============================================================================
:: LevelUp - Lanceur Windows
::
:: Usage : LevelUp.bat [options]
::
::   --reinstall   Force la recreation du venv (.venv supprime et recree)
::   --no-spnkr    Install legere sans les dependances API Halo (spnkr)
::   --offline     Interdit l'acces PyPI (necessite un cache pip local)
:: ============================================================================

title LevelUp - Halo Infinite Dashboard

:: Se placer dans le dossier du script
cd /d "%~dp0"

:: -- Parser les options (les autres args sont transmis a launcher.py) ----------
set OPT_REINSTALL=0
set OPT_NO_SPNKR=0
set OPT_OFFLINE=0
set "PASS_ARGS="

:parse_args
if "%~1"=="" goto args_done
if /i "%~1"=="--reinstall" ( set OPT_REINSTALL=1 & shift & goto parse_args )
if /i "%~1"=="--no-spnkr"  ( set OPT_NO_SPNKR=1  & shift & goto parse_args )
if /i "%~1"=="--offline"   ( set OPT_OFFLINE=1    & shift & goto parse_args )
set "PASS_ARGS=!PASS_ARGS! %~1"
shift
goto parse_args
:args_done

:: -- Preparer le dossier de logs -----------------------------------------------
if not exist "data\logs" mkdir "data\logs" >nul 2>&1
set "INSTALL_LOG=%~dp0data\logs\install.log"
set "PYPROJECT_HASH=%~dp0.venv\.pyproject_hash"

:: -- Flag --reinstall -----------------------------------------------------------
if !OPT_REINSTALL! equ 1 (
    if exist ".venv" (
        echo   Suppression du venv ^(--reinstall^)...
        rmdir /s /q .venv >nul 2>&1
    )
)

:: -- Venv deja present -> validation et lancement --------------------------------
if not exist ".venv\Scripts\python.exe" goto :first_launch

:: 1. Interpreteur vivant ? (peut pointer vers un Python desinstalle)
.venv\Scripts\python.exe -c "import sys; sys.exit(0)" >nul 2>&1
if !ERRORLEVEL! neq 0 (
    echo   Interpreteur du venv inaccessible ^(Python desinstalle ?^), recreation...
    rmdir /s /q .venv >nul 2>&1
    goto :first_launch
)

:: 2. Imports critiques presents ?
.venv\Scripts\python.exe -c "import streamlit, duckdb, polars" >nul 2>&1
if !ERRORLEVEL! neq 0 (
    echo   Environnement incomplet detecte, reinstallation...
    rmdir /s /q .venv >nul 2>&1
    goto :first_launch
)

:: 3. pyproject.toml modifie ? (fingerprint MD5 via certutil)
set "CURRENT_TS="
for /f "skip=1 tokens=*" %%h in ('certutil -hashfile "pyproject.toml" MD5 2^>nul') do (
    if not defined CURRENT_TS set "CURRENT_TS=%%h"
)
set "STORED_TS="
if exist "!PYPROJECT_HASH!" for /f "usebackq delims=" %%l in ("!PYPROJECT_HASH!") do set "STORED_TS=%%l"
if defined CURRENT_TS (
    if "!CURRENT_TS!" neq "!STORED_TS!" (
        echo   pyproject.toml modifie - mise a jour des dependances...
        set "INSTALL_EXTRA=.[spnkr]"
        if !OPT_NO_SPNKR! equ 1 set "INSTALL_EXTRA=."
        set "PIP_OPTS=--disable-pip-version-check"
        if !OPT_OFFLINE! equ 1 set "PIP_OPTS=!PIP_OPTS! --no-index"
        .venv\Scripts\python.exe -m pip install -e "!INSTALL_EXTRA!" !PIP_OPTS! >> "!INSTALL_LOG!" 2>&1
        if !ERRORLEVEL! equ 0 (
            echo !CURRENT_TS!>"!PYPROJECT_HASH!"
            echo   OK - Dependances a jour.
        ) else (
            echo   Mise a jour partielle ^(details : !INSTALL_LOG!^)
        )
    )
)

:: Venv OK -> lancer l'app
.venv\Scripts\python.exe launcher.py !PASS_ARGS!
if !ERRORLEVEL! neq 0 (
    echo.
    echo   [ERREUR] LevelUp s'est arrete avec le code !ERRORLEVEL!.
)
goto :end

:: -- Premier lancement ----------------------------------------------------------
:first_launch
echo.
echo  ========================================
echo       LevelUp - Premier lancement
echo  ========================================
echo.

set "PY="

:: Essayer py (Windows Python Launcher) -- versions par ordre de preference
where py >nul 2>&1
if !ERRORLEVEL! equ 0 (
    for %%v in (3.13 3.12 3.11 3.10) do (
        if not defined PY (
            for /f "tokens=*" %%i in ('py -%%v -c "import sys; print(sys.executable)" 2^>nul') do set "PY=%%i"
        )
    )
)

:: Verifier que la version trouvee est bien >= 3.10
if defined PY (
    for /f "tokens=*" %%v in ('"!PY!" -c "import sys; print(sys.version_info.minor)" 2^>nul') do (
        if %%v LSS 10 (
            echo   Python 3.%%v detecte ^(trop ancien^), recherche d'une version 3.10+...
            set "PY="
        )
    )
)

:: Essayer python / python3 dans le PATH
if not defined PY (
    for %%n in (python python3) do (
        if not defined PY (
            where %%n >nul 2>&1
            if !ERRORLEVEL! equ 0 (
                for /f "tokens=*" %%v in ('%%n -c "import sys; print(sys.version_info.minor)" 2^>nul') do (
                    if %%v GEQ 10 (
                        for /f "tokens=*" %%i in ('%%n -c "import sys; print(sys.executable)" 2^>nul') do set "PY=%%i"
                    )
                )
            )
        )
    )
)

:: Chercher dans les emplacements standards (user, system, x86)
if not defined PY (
    for %%P in (
        "%LOCALAPPDATA%\Programs\Python\Python313\python.exe"
        "%LOCALAPPDATA%\Programs\Python\Python312\python.exe"
        "%LOCALAPPDATA%\Programs\Python\Python311\python.exe"
        "%LOCALAPPDATA%\Programs\Python\Python310\python.exe"
        "%PROGRAMFILES%\Python313\python.exe"
        "%PROGRAMFILES%\Python312\python.exe"
        "%PROGRAMFILES%\Python311\python.exe"
        "%PROGRAMFILES%\Python310\python.exe"
        "%PROGRAMFILES(X86)%\Python312\python.exe"
        "%PROGRAMFILES(X86)%\Python311\python.exe"
    ) do (
        if not defined PY if exist %%P set "PY=%%~P"
    )
)

:: -- Python non trouve -> proposer installation via winget ---------------------
if not defined PY (
    echo   Python 3.10+ non trouve sur ce systeme.
    echo.
    where winget >nul 2>&1
    if !ERRORLEVEL! neq 0 (
        echo   winget non disponible sur ce systeme.
        echo.
        echo   Installez Python 3.12 manuellement :
        echo     https://www.python.org/downloads/
        echo.
        echo   Puis relancez LevelUp.bat.
        start "" "https://www.python.org/downloads/"
        goto :end
    )
    echo   Voulez-vous installer Python 3.12 automatiquement via winget ?
    echo.
    choice /C ON /N /M "   [O]ui / [N]on : "
    if !ERRORLEVEL! equ 2 (
        echo   Installez Python depuis https://www.python.org/downloads/
        goto :end
    )
    winget install --id Python.Python.3.12 --source winget --scope user --accept-source-agreements --accept-package-agreements
    if !ERRORLEVEL! neq 0 (
        echo   Echec. Installez Python depuis https://www.python.org/downloads/
        goto :end
    )
    :: Retrouver le chemin reel via py launcher
    for /f "tokens=*" %%i in ('py -3.12 -c "import sys; print(sys.executable)" 2^>nul') do set "PY=%%i"
    if not defined PY (
        for %%P in (
            "%LOCALAPPDATA%\Programs\Python\Python312\python.exe"
            "%PROGRAMFILES%\Python312\python.exe"
        ) do if not defined PY if exist %%P set "PY=%%~P"
    )
    if not defined PY (
        echo   Python installe mais chemin introuvable.
        echo   Fermez cette fenetre, redemarrez et relancez LevelUp.bat.
        goto :end
    )
)

echo   Python : !PY!

:: -- Creer le venv --------------------------------------------------------------
echo   Creation de l'environnement virtuel...
"!PY!" -m venv .venv >> "!INSTALL_LOG!" 2>&1
if !ERRORLEVEL! neq 0 (
    echo   Erreur creation du venv. Details : !INSTALL_LOG!
    goto :end
)

:: -- Mettre a jour pip -----------------------------------------------------------
echo   Mise a jour de pip...
.venv\Scripts\python.exe -m pip install --upgrade pip --disable-pip-version-check -q >> "!INSTALL_LOG!" 2>&1
if !ERRORLEVEL! neq 0 echo   pip non mis a jour ^(reseau/proxy ?^). Poursuite avec la version installee.

:: -- Installer les dependances ---------------------------------------------------
set "INSTALL_EXTRA=.[spnkr]"
if !OPT_NO_SPNKR! equ 1 set "INSTALL_EXTRA=."
set "PIP_OPTS=--disable-pip-version-check"
if !OPT_OFFLINE! equ 1 set "PIP_OPTS=!PIP_OPTS! --no-index"

echo   Installation des dependances ^(quelques minutes a la premiere execution^)...
.venv\Scripts\python.exe -m pip install -e "!INSTALL_EXTRA!" !PIP_OPTS! >> "!INSTALL_LOG!" 2>&1
if !ERRORLEVEL! neq 0 (
    echo.
    echo   Echec de l'installation des dependances.
    echo   Causes possibles :
    echo   - Pas de connexion internet
    echo   - Dossier en lecture seule ^(deplace LevelUp dans Documents^)
    echo   - Espace disque insuffisant
    echo   Details : !INSTALL_LOG!
    goto :end
)

:: Enregistrer le fingerprint de pyproject.toml
set "NEW_TS="
for /f "skip=1 tokens=*" %%h in ('certutil -hashfile "pyproject.toml" MD5 2^>nul') do (
    if not defined NEW_TS set "NEW_TS=%%h"
)
if defined NEW_TS echo !NEW_TS!>"!PYPROJECT_HASH!"

echo   OK - Environnement pret.
echo.

:: Lancer l'app
.venv\Scripts\python.exe launcher.py !PASS_ARGS!
if !ERRORLEVEL! neq 0 (
    echo.
    echo   [ERREUR] LevelUp s'est arrete avec le code !ERRORLEVEL!.
)

:: -- Fin -- toujours atteint ----------------------------------------------------
:end
echo.
echo   Appuie sur une touche pour fermer cette fenetre...
pause >nul
