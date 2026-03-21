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

:: -- Detection langue systeme ---------------------------------------------------
set "SCRIPT_LANG=en"
for /f "tokens=3" %%L in ('reg query "HKCU\Control Panel\International" /v LocaleName 2^>nul') do (
    set "_WIN_LOCALE=%%L"
)
if defined _WIN_LOCALE (
    set "_TMP=!_WIN_LOCALE:~0,2!"
    if /i "!_TMP!"=="fr" set "SCRIPT_LANG=fr"
)

:: -- Messages localises ---------------------------------------------------------
if /i "!SCRIPT_LANG!"=="fr" (
    set "MSG_REINSTALL=  Suppression du venv (--reinstall)..."
    set "MSG_VENV_DEAD=  Interpreteur du venv inaccessible (Python desinstalle ?), recreation..."
    set "MSG_VENV_INCOMPLETE=  Environnement incomplet detecte, reinstallation..."
    set "MSG_DEPS_CHANGED=  pyproject.toml modifie - mise a jour des dependances..."
    set "MSG_DEPS_OK=  OK - Dependances a jour."
    set "MSG_DEPS_PARTIAL=  Mise a jour partielle"
    set "MSG_FIRST_LAUNCH_TITLE=      LevelUp - Premier lancement"
    set "MSG_PYTHON_NOT_FOUND=  Python 3.10+ non trouve sur ce systeme."
    set "MSG_CREATING_VENV=  Creation de l'environnement virtuel..."
    set "MSG_PIP_UPDATE=  Mise a jour de pip..."
    set "MSG_PIP_WARN=  pip non mis a jour (reseau/proxy ?). Poursuite avec la version installee."
    set "MSG_INSTALLING=  Installation des dependances (quelques minutes a la premiere execution)..."
    set "MSG_INSTALL_FAIL=  Echec de l'installation des dependances."
    set "MSG_INSTALL_FAIL_CAUSES=  Causes possibles :"
    set "MSG_INSTALL_FAIL_NETWORK=  - Pas de connexion internet"
    set "MSG_INSTALL_FAIL_READONLY=  - Dossier en lecture seule (deplace LevelUp dans Documents)"
    set "MSG_INSTALL_FAIL_DISK=  - Espace disque insuffisant"
    set "MSG_READY=  OK - Environnement pret."
    set "MSG_WINGET_UNAVAIL=  winget non disponible sur ce systeme."
    set "MSG_INSTALL_MANUAL=  Installez Python 3.12 manuellement :"
    set "MSG_RELAUNCH=  Puis relancez LevelUp.bat."
    set "MSG_WINGET_PROMPT=  Voulez-vous installer Python 3.12 automatiquement via winget ?"
    set "MSG_WINGET_CHOICE=   [O]ui / [N]on : "
    set "MSG_WINGET_DECLINE=  Installez Python depuis https://www.python.org/downloads/"
    set "MSG_WINGET_FAIL=  Echec. Installez Python depuis https://www.python.org/downloads/"
    set "MSG_PYTHON_PATH_FAIL=  Python installe mais chemin introuvable."
    set "MSG_PYTHON_RESTART=  Fermez cette fenetre, redemarrez et relancez LevelUp.bat."
    set "MSG_ERROR=  [ERREUR] LevelUp s'est arrete avec le code"
    set "MSG_PRESS_KEY=  Appuie sur une touche pour fermer cette fenetre..."
    set "MSG_VENV_FAIL=  Erreur creation du venv."
    set "MSG_PYTHON_TOO_OLD=  Python 3.%%v detecte (trop ancien), recherche d'une version 3.10+..."
    set "MSG_CHOICE_KEYS=ON"
) else (
    set "MSG_REINSTALL=  Removing venv (--reinstall)..."
    set "MSG_VENV_DEAD=  Venv interpreter inaccessible (Python uninstalled?), recreating..."
    set "MSG_VENV_INCOMPLETE=  Incomplete environment detected, reinstalling..."
    set "MSG_DEPS_CHANGED=  pyproject.toml changed - updating dependencies..."
    set "MSG_DEPS_OK=  OK - Dependencies up to date."
    set "MSG_DEPS_PARTIAL=  Partial update"
    set "MSG_FIRST_LAUNCH_TITLE=      LevelUp - First launch"
    set "MSG_PYTHON_NOT_FOUND=  Python 3.10+ not found on this system."
    set "MSG_CREATING_VENV=  Creating virtual environment..."
    set "MSG_PIP_UPDATE=  Updating pip..."
    set "MSG_PIP_WARN=  pip not updated (network/proxy?). Continuing with installed version."
    set "MSG_INSTALLING=  Installing dependencies (this may take a few minutes on first run)..."
    set "MSG_INSTALL_FAIL=  Dependency installation failed."
    set "MSG_INSTALL_FAIL_CAUSES=  Possible causes:"
    set "MSG_INSTALL_FAIL_NETWORK=  - No internet connection"
    set "MSG_INSTALL_FAIL_READONLY=  - Read-only folder (move LevelUp to Documents)"
    set "MSG_INSTALL_FAIL_DISK=  - Insufficient disk space"
    set "MSG_READY=  OK - Environment ready."
    set "MSG_WINGET_UNAVAIL=  winget not available on this system."
    set "MSG_INSTALL_MANUAL=  Install Python 3.12 manually:"
    set "MSG_RELAUNCH=  Then relaunch LevelUp.bat."
    set "MSG_WINGET_PROMPT=  Would you like to install Python 3.12 automatically via winget?"
    set "MSG_WINGET_CHOICE=   [Y]es / [N]o : "
    set "MSG_WINGET_DECLINE=  Install Python from https://www.python.org/downloads/"
    set "MSG_WINGET_FAIL=  Failed. Install Python from https://www.python.org/downloads/"
    set "MSG_PYTHON_PATH_FAIL=  Python installed but path not found."
    set "MSG_PYTHON_RESTART=  Close this window, restart and relaunch LevelUp.bat."
    set "MSG_ERROR=  [ERROR] LevelUp exited with code"
    set "MSG_PRESS_KEY=  Press any key to close this window..."
    set "MSG_VENV_FAIL=  Error creating venv."
    set "MSG_PYTHON_TOO_OLD=  Python 3.%%v detected (too old), looking for 3.10+..."
    set "MSG_CHOICE_KEYS=YN"
)

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
        echo !MSG_REINSTALL!
        rmdir /s /q .venv >nul 2>&1
    )
)

:: -- Venv deja present -> validation et lancement --------------------------------
if not exist ".venv\Scripts\python.exe" goto :first_launch

:: 1. Interpreteur vivant ? (peut pointer vers un Python desinstalle)
.venv\Scripts\python.exe -c "import sys; sys.exit(0)" >nul 2>&1
if !ERRORLEVEL! neq 0 (
    echo !MSG_VENV_DEAD!
    rmdir /s /q .venv >nul 2>&1
    goto :first_launch
)

:: 2. Imports critiques presents ?
.venv\Scripts\python.exe -c "import streamlit, duckdb, polars" >nul 2>&1
if !ERRORLEVEL! neq 0 (
    echo !MSG_VENV_INCOMPLETE!
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
        echo !MSG_DEPS_CHANGED!
        set "INSTALL_EXTRA=.[spnkr]"
        if !OPT_NO_SPNKR! equ 1 set "INSTALL_EXTRA=."
        set "PIP_OPTS=--disable-pip-version-check"
        if !OPT_OFFLINE! equ 1 set "PIP_OPTS=!PIP_OPTS! --no-index"
        .venv\Scripts\python.exe -m pip install -e "!INSTALL_EXTRA!" !PIP_OPTS! >> "!INSTALL_LOG!" 2>&1
        if !ERRORLEVEL! equ 0 (
            echo !CURRENT_TS!>"!PYPROJECT_HASH!"
            echo !MSG_DEPS_OK!
        ) else (
            echo !MSG_DEPS_PARTIAL! ^(details : !INSTALL_LOG!^)
        )
    )
)

:: Venv OK -> lancer l'app
.venv\Scripts\python.exe launcher.py !PASS_ARGS!
if !ERRORLEVEL! neq 0 (
    echo.
    echo !MSG_ERROR! !ERRORLEVEL!.
)
goto :end

:: -- Premier lancement ----------------------------------------------------------
:first_launch
echo.
echo  ========================================
echo !MSG_FIRST_LAUNCH_TITLE!
echo  ========================================
echo.

set "PY="

:: Essayer py (Windows Python Launcher) -- versions par ordre de preference
where py >nul 2>&1
if !ERRORLEVEL! equ 0 (
    for %%v in (3.13 3.12 3.11 3.10) do (
        if not defined PY (
            for /f "tokens=*" %%i in ('py -%%v -c "import sys; print(sys.executable)" 2^>nul') do (
                if exist "%%i" set "PY=%%i"
            )
        )
    )
)

:: Verifier que la version trouvee est bien >= 3.10
if defined PY (
    for /f "tokens=*" %%v in ('"!PY!" -c "import sys; print(sys.version_info.minor)" 2^>nul') do (
        if %%v LSS 10 (
            echo !MSG_PYTHON_TOO_OLD!
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
    echo !MSG_PYTHON_NOT_FOUND!
    echo.
    where winget >nul 2>&1
    if !ERRORLEVEL! neq 0 (
        echo !MSG_WINGET_UNAVAIL!
        echo.
        echo !MSG_INSTALL_MANUAL!
        echo     https://www.python.org/downloads/
        echo.
        echo !MSG_RELAUNCH!
        start "" "https://www.python.org/downloads/"
        goto :end
    )
    echo !MSG_WINGET_PROMPT!
    echo.
    choice /C !MSG_CHOICE_KEYS! /N /M "!MSG_WINGET_CHOICE!"
    if !ERRORLEVEL! equ 2 (
        echo !MSG_WINGET_DECLINE!
        goto :end
    )
    winget install --id Python.Python.3.12 --source winget --scope user --accept-source-agreements --accept-package-agreements
    if !ERRORLEVEL! neq 0 (
        echo !MSG_WINGET_FAIL!
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
        echo !MSG_PYTHON_PATH_FAIL!
        echo !MSG_PYTHON_RESTART!
        goto :end
    )
)

echo   Python : !PY!

:: -- Creer le venv --------------------------------------------------------------
echo !MSG_CREATING_VENV!
"!PY!" -m venv .venv >> "!INSTALL_LOG!" 2>&1
if !ERRORLEVEL! neq 0 (
    echo !MSG_VENV_FAIL! Details : !INSTALL_LOG!
    goto :end
)

:: -- Mettre a jour pip -----------------------------------------------------------
echo !MSG_PIP_UPDATE!
.venv\Scripts\python.exe -m pip install --upgrade pip --disable-pip-version-check -q >> "!INSTALL_LOG!" 2>&1
if !ERRORLEVEL! neq 0 echo !MSG_PIP_WARN!

:: -- Installer les dependances ---------------------------------------------------
set "INSTALL_EXTRA=.[spnkr]"
if !OPT_NO_SPNKR! equ 1 set "INSTALL_EXTRA=."
set "PIP_OPTS=--disable-pip-version-check"
if !OPT_OFFLINE! equ 1 set "PIP_OPTS=!PIP_OPTS! --no-index"

echo !MSG_INSTALLING!
.venv\Scripts\python.exe -m pip install -e "!INSTALL_EXTRA!" !PIP_OPTS! >> "!INSTALL_LOG!" 2>&1
if !ERRORLEVEL! neq 0 (
    echo.
    echo !MSG_INSTALL_FAIL!
    echo !MSG_INSTALL_FAIL_CAUSES!
    echo !MSG_INSTALL_FAIL_NETWORK!
    echo !MSG_INSTALL_FAIL_READONLY!
    echo !MSG_INSTALL_FAIL_DISK!
    echo   Details : !INSTALL_LOG!
    goto :end
)

:: Enregistrer le fingerprint de pyproject.toml
set "NEW_TS="
for /f "skip=1 tokens=*" %%h in ('certutil -hashfile "pyproject.toml" MD5 2^>nul') do (
    if not defined NEW_TS set "NEW_TS=%%h"
)
if defined NEW_TS echo !NEW_TS!>"!PYPROJECT_HASH!"

echo !MSG_READY!
echo.

:: Lancer l'app
.venv\Scripts\python.exe launcher.py !PASS_ARGS!
if !ERRORLEVEL! neq 0 (
    echo.
    echo !MSG_ERROR! !ERRORLEVEL!.
)

:: -- Fin -- toujours atteint ----------------------------------------------------
:end
echo.
echo !MSG_PRESS_KEY!
pause >nul
