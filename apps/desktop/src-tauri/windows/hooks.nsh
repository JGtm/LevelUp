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
