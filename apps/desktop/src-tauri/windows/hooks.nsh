; Crochets NSIS de l'installeur LevelUp (ADR 0045 D-15).
;
; Installation par utilisateur, sans elevation : le programme vit sous
; %LOCALAPPDATA%\Programs\LevelUp, les donnees sous %LOCALAPPDATA%\LevelUp. Tauri 2 propose par
; defaut %LOCALAPPDATA%\<nom du produit>, c'est-a-dire la racine de DONNEES : le programme et
; les donnees partageraient un dossier. Le crochet deplace ce SEUL emplacement par defaut ; un
; dossier choisi par l'utilisateur (page de dossier, ou /D=) ou celui d'une installation
; precedente (cle de registre relue par le modele) n'est jamais change.
;
; Le desinstalleur de Tauri ne supprime que les fichiers listes a l'installation et ne retire un
; dossier que s'il est vide (RMDir sans /r) : la racine de donnees n'est jamais touchee. Seule la
; case « supprimer les donnees de l'application » efface %LOCALAPPDATA%\<identifiant Tauri>
; (profil WebView2), et elle n'est ni cochee ni proposee en mode silencieux.

!macro NSIS_HOOK_PREINSTALL
  ${If} $INSTDIR == "$LOCALAPPDATA\${PRODUCTNAME}"
    StrCpy $INSTDIR "$LOCALAPPDATA\Programs\${PRODUCTNAME}"
    SetOutPath $INSTDIR
    ; Le modele a cree le dossier par defaut avant ce crochet : vide, il disparait ; plein
    ; (donnees de l'utilisateur), RMDir le laisse.
    RMDir "$LOCALAPPDATA\${PRODUCTNAME}"
  ${EndIf}
!macroend

; Le modele de Tauri pose MUI_FINISHPAGE_NOAUTOCLOSE (la page d'extraction reste affichee
; jusqu'a un clic, pour lire le detail en cas d'incident) et ne ferme seul que le mode passif.
; L'installeur interactif de LevelUp passe de lui-meme a la page finale quand l'extraction a
; REUSSI : SetAutoClose true est evalue a la fin de la section d'installation, apres ce crochet.
; Un echec (Abort, erreur de copie) garde la page d'extraction, et son journal, affichee.
!macro NSIS_HOOK_POSTINSTALL
  SetAutoClose true
!macroend

; Desinstallation (ADR 0045 D-28) : le desinstalleur ne retire que les fichiers listes a l'installation. L'entree de
; demarrage avec Windows (cle Run de l'utilisateur) et la tache planifiee `LevelUp\Refresh-<empreinte>` vivent HORS
; du dossier du programme : le serveur livre les retire (`-mode uninstall-cleanup`), AVANT l'effacement de ses
; fichiers. Le mode n'ouvre aucune base et ne touche JAMAIS a la racine de donnees ; son code de sortie est ignore
; (la desinstallation ne depend pas de ce nettoyage), et le delai le borne.
;
; Pas dans le flux de mise a jour : le modele de Tauri rejoue le desinstalleur de l'ancienne version avec /UPDATE,
; et l'utilisateur qui met a jour garde son entree et sa tache.
;
; La racine de donnees (qui nomme la tache) est celle de l'environnement du desinstalleur, a defaut la racine par
; defaut %LOCALAPPDATA%\LevelUp, comme la coquille.
!macro NSIS_HOOK_PREUNINSTALL
  ${If} $UpdateMode <> 1
    Push $0
    ReadEnvStr $0 LEVELUP_DATA_DIR
    ${If} $0 == ""
      System::Call 'Kernel32::SetEnvironmentVariable(t, t)i("LEVELUP_DATA_DIR", "$LOCALAPPDATA\${PRODUCTNAME}").r0'
    ${EndIf}
    ${If} ${FileExists} "$INSTDIR\levelup-server.exe"
      nsExec::ExecToLog /TIMEOUT=60000 '"$INSTDIR\levelup-server.exe" -mode uninstall-cleanup'
      Pop $0
    ${EndIf}
    Pop $0
  ${EndIf}
!macroend
