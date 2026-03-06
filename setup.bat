@echo off
chcp 65001 >nul 2>&1
setlocal EnableDelayedExpansion

:: ============================================================================
:: LevelUp - Script d'installation automatique (Windows)
::
:: Double-cliquez sur ce fichier pour :
::   1. Vérifier / installer Python 3.12+
::   2. Créer l'environnement virtuel (.venv)
::   3. Installer les dépendances
::   4. Lancer le dashboard (le wizard de configuration s'affiche au premier lancement)
::
:: Pré-requis : Windows 10/11, connexion internet
:: ============================================================================

title LevelUp - Installation

echo.
echo  ╔══════════════════════════════════════════╗
echo  ║       LevelUp - Installation Halo        ║
echo  ╚══════════════════════════════════════════╝
echo.

:: ── 1. Vérifier Python ──────────────────────────────────────────────────────

echo [1/4] Verification de Python...

:: Chercher Python dans le PATH
where python >nul 2>&1
if %ERRORLEVEL% equ 0 (
    for /f "tokens=*" %%i in ('python --version 2^>^&1') do set PYVER=%%i
    echo   Trouve : !PYVER!
    goto :check_python_version
)

:: Chercher dans les emplacements standards Windows
set "PYTHON_FOUND="
for %%P in (
    "%LOCALAPPDATA%\Programs\Python\Python312\python.exe"
    "%LOCALAPPDATA%\Programs\Python\Python313\python.exe"
    "%LOCALAPPDATA%\Programs\Python\Python311\python.exe"
    "%ProgramFiles%\Python312\python.exe"
    "%ProgramFiles%\Python313\python.exe"
) do (
    if exist %%P (
        set "PYTHON_FOUND=%%~P"
        goto :found_python
    )
)

:: Python non trouvé, proposer installation via winget
echo   Python non trouve.
echo.
echo   LevelUp necessite Python 3.10 ou superieur.
echo   Voulez-vous installer Python 3.12 via winget ?
echo.
choice /C ON /N /M "   [O]ui / [N]on : "
if %ERRORLEVEL% equ 2 (
    echo.
    echo   Installation annulee. Installez Python manuellement :
    echo   https://www.python.org/downloads/
    pause
    exit /b 1
)

echo.
echo   Installation de Python 3.12...
winget install --id Python.Python.3.12 --scope user --accept-source-agreements --accept-package-agreements
if %ERRORLEVEL% neq 0 (
    echo.
    echo   [ERREUR] L'installation de Python a echoue.
    echo   Installez Python manuellement : https://www.python.org/downloads/
    pause
    exit /b 1
)

:: Rafraîchir le PATH après installation
set "PATH=%LOCALAPPDATA%\Programs\Python\Python312\;%LOCALAPPDATA%\Programs\Python\Python312\Scripts\;%PATH%"
echo   Python 3.12 installe avec succes.

:found_python
if defined PYTHON_FOUND (
    echo   Trouve : !PYTHON_FOUND!
    set "PATH=%~dp0;!PYTHON_FOUND!\..\;!PYTHON_FOUND!\..\Scripts\;%PATH%"
)

:check_python_version
:: Vérifier que la version est 3.10+
for /f "tokens=2 delims= " %%v in ('python --version 2^>^&1') do set PYVER_NUM=%%v
for /f "tokens=1,2 delims=." %%a in ("!PYVER_NUM!") do (
    set PYMAJOR=%%a
    set PYMINOR=%%b
)
if !PYMAJOR! lss 3 (
    echo   [ERREUR] Python 3.10+ requis, version trouvee : !PYVER_NUM!
    pause
    exit /b 1
)
if !PYMAJOR! equ 3 if !PYMINOR! lss 10 (
    echo   [ERREUR] Python 3.10+ requis, version trouvee : !PYVER_NUM!
    pause
    exit /b 1
)
echo   OK - Python !PYVER_NUM!

:: ── 2. Créer le venv ────────────────────────────────────────────────────────

echo.
echo [2/4] Environnement virtuel...

cd /d "%~dp0"

if exist ".venv\Scripts\python.exe" (
    echo   .venv existe deja, verification...
    .venv\Scripts\python.exe --version >nul 2>&1
    if %ERRORLEVEL% equ 0 (
        echo   OK - .venv fonctionnel
        goto :install_deps
    )
    echo   .venv corrompu, recreation...
    rmdir /s /q .venv
)

echo   Creation de .venv...
python -m venv .venv
if %ERRORLEVEL% neq 0 (
    echo   [ERREUR] Impossible de creer le venv.
    pause
    exit /b 1
)
echo   OK - .venv cree

:: ── 3. Installer les dépendances ────────────────────────────────────────────

:install_deps
echo.
echo [3/4] Installation des dependances...

.venv\Scripts\python.exe -m pip install --upgrade pip --quiet
.venv\Scripts\python.exe -m pip install -r requirements.txt --quiet
if %ERRORLEVEL% neq 0 (
    echo   [ERREUR] L'installation des dependances a echoue.
    echo   Verifiez votre connexion internet et reessayez.
    pause
    exit /b 1
)

:: Vérifier les packages critiques
.venv\Scripts\python.exe -c "import streamlit; import duckdb; import polars; print('OK')" >nul 2>&1
if %ERRORLEVEL% neq 0 (
    echo   [ERREUR] Packages critiques manquants apres installation.
    pause
    exit /b 1
)
echo   OK - Dependances installees

:: ── 4. Lancer le dashboard ──────────────────────────────────────────────────

echo.
echo [4/4] Lancement de LevelUp...
echo.
echo   Le dashboard va s'ouvrir dans votre navigateur.
echo   Si c'est votre premiere utilisation, un assistant de
echo   configuration vous guidera.
echo.
echo   Pour arreter : Ctrl+C dans cette fenetre ou fermez-la.
echo.

.venv\Scripts\python.exe -m streamlit run streamlit_app.py --server.headless false

echo.
echo   LevelUp arrete.
pause
