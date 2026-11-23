# Plan : fenêtres de console du bureau et historique de synchronisation (2026-11-22)

Demande du user (2026-11-22) : (1) issue GitHub #213 « Manually sync » (7 000 matchs Infinite + Halo 5,
bouton introuvable) ; (2) LevelUp bureau ouvre de brèves fenêtres de terminal qui passent au premier plan
et mettent les films et les jeux en pause (rapport d'un collègue). Go donné le 2026-11-22 (« Ok avec tes
correctifs ; regarde s'il n'y a pas d'autres coupables ; tu me feras valider le texte de la réponse »).

Contrat : skills `plan-execution` + `levelup-plan`. Deux pistes PARALLELES et indépendantes, une branche et
un worktree chacune, depuis `origin/release/v8.3.0` :

| Piste | Branche | Worktree |
|---|---|---|
| A, fenêtres | `fix/bureau-sans-fenetre-console` | `Downloads/Scripts/LevelUp-wt-fenetres` |
| B, historique | `fix/sync-historique-complet` | `Downloads/Scripts/LevelUp-wt-sync` |

Ce fichier ne vit que sur la branche A ; le pilote seul le tient.

## Piste A : aucune fenêtre de console

Constat (pilote, sur pièces) : la tâche planifiée du réglage `scheduled_refresh` lance
`cmd.exe /d /s /c "set ... && levelup-server.exe -mode refresh-only"` (`internal/platform/schedtask/xml.go`),
toutes les 6 h et au réveil (`StartWhenAvailable`), sous la session de l'utilisateur (`InteractiveToken`) :
Windows crée une console visible (Windows Terminal sous Windows 11), même quand l'application tourne (le
serveur sort aussitôt, la console a déjà été créée). ADR 0045 L21.e l'annonçait (« may flash »).

Correctif décidé : la tâche lance la coquille `LevelUp.exe` (sous-système graphique, aucune console), qui
relaie au serveur, lancé caché comme au démarrage normal, puis sort avec son code.

- [x] A0 Audit des autres coupables (pilote), consigné ici :
  - dépendances Go du serveur (`go list -deps ./cmd/server`, 283 paquets) : seuls `platform/childproc`
    (CREATE_NO_WINDOW + HideWindow) et `pkg/duckdbbackup` (restic, `exec` nu) lancent un processus ;
  - code Rust de la coquille : seul `sidecar.rs` lance (CREATE_NO_WINDOW) ;
  - dépendances Rust (541 crates du `Cargo.lock`) : `open`, `tauri-plugin-opener`, `tauri` (relance) passent
    CREATE_NO_WINDOW ou ShellExecute ; l'updater lance l'installeur par ShellExecute (fenêtre attendue,
    seulement à une mise à jour) ;
  - crochet NSIS : `nsExec` (caché), désinstallation seulement.
  - restic : aucune planification dans l'application (timers systemd du VPS) ; le déclenchement manuel
    (`POST /settings/backup/run`) est masqué hors `can_use_restic`, faux sur le bureau, et restic n'y est
    pas livré : pas un coupable.
  - Conclusion : la tâche planifiée est le seul lancement de console du code livré. Une fenêtre vue sans
    le réglage actif viendrait d'ailleurs que LevelUp.
- [x] A1 Serveur : la variable `LEVELUP_SHELL_EXE` (chemin de la coquille), lue une fois dans
  `internal/config` ; la définition XML de la tâche lance `<coquille> --refresh-only --data-dir "<d>"
  --resources-dir "<r>" --logs-dir "<l>"` ; plus de `cmd.exe` ; chemins refusés réduits à ce que la ligne
  de commande Windows ne protège pas (guillemet, retour à la ligne, barre oblique inverse finale) ; coquille
  inconnue = état `error` avec un code fixe, sans tâche posée.
- [x] A2 Coquille : `LevelUp.exe --refresh-only ...` relaie au serveur voisin (`-mode refresh-only`, les
  trois racines dans l'environnement, CREATE_NO_WINDOW, flux nuls), attend, sort avec son code, AVANT le
  constructeur Tauri (ni instance unique, ni fenêtre) ; la coquille pose `LEVELUP_SHELL_EXE` au serveur
  qu'elle lance.
- [x] A3 ADR 0045 (L21.e et contrat D-1) amendés ; « may flash » retiré.
- [x] A4 Gate : `go test ./internal/platform/schedtask/... ./internal/config/... ./cmd/server/...
  ./internal/archlint/...`, `make desktop-test` (cargo test, clippy, fmt), golangci-lint du diff.
- [!] A5 Vérification à l'écran du user (installeur de la release) : `schtasks /Run` sur la tâche, aucune
  fenêtre. Report admis (seul le user a la machine installée).

## Piste B : tout l'historique se synchronise

Constat (agent de recherche, recoupé par le pilote) : Infinite rattrape l'historique tout seul tant que
`sync_meta.history_complete_at` est absente (`internal/sync/engine.go:480`, 200 matchs par passe) ; Halo 5
s'arrête au premier match connu (`internal/games/halo_5/capture.go:164`) et ne rattrape jamais ; le
bouton de synchronisation n'existe que dans l'admin et sur l'Accueil vide ; l'aide renvoie à un bouton
des Paramètres qui n'existe plus.

- [x] B1 Halo 5 rattrape l'historique comme Infinite : tant que la fin de l'historique n'a pas été vue
  une fois, le sync saute les matchs connus et continue à paginer (plafond par passe inchangé) ; fin vue
  sans erreur = clé posée. Écriture `sync_meta` sous le verrou d'écrivain du joueur.
- [x] B2 `GET /players/{slug}/sync/status` porte `history_complete` (les deux titres) ; openapi et
  `generated.ts` régénérés.
- [x] B3 Web : carte « Synchronisation » dans Paramètres › Données : « Synchroniser maintenant » pour le
  joueur affiché (droit d'écriture, délai restant dit), état de l'historique (complet, ou rattrapage en
  cours) ; textes fr et en.
- [x] B4 Aide fr et en corrigée (bouton, rattrapage automatique, les deux titres).
- [x] B5 Gate : tests ciblés Go (`games/halo_5/...`, `sync/...`, handlers), `-tags=integration` sur les
  paquets sync touchés, typecheck, vitest ciblé puis complet une fois par le pilote, lint cross-feature.
- [x] B6 Réponse sur l'issue #213 : texte validé par le user le 2026-11-22, posté (commentaire
  7077190663). Le joueur n'existe pas sur l'instance en ligne : le rapport vient de l'application de bureau
  (l'agent WebView2 se présente comme Edge), où la synchronisation automatique est active par défaut.

## Clôture

- [x] C1 Revue adverse du diff des deux pistes (Sonnet, une ronde par lentille : A, B Go, B web ; journal).
- [x] C2 `delivery-checklist` + `levelup-livraison` ; thought_log ; fusion des deux branches dans
  `release/v8.3.0` (squash), CI lue après poussée.

## Journal

- 2026-11-22 : plan écrit, A0 en grande partie fait par le pilote, worktrees créés.
- 2026-11-22 : piste A écrite (5767071bee, cce9d80ef2, 5c6ea16ad8) ; diff relu par le pilote (XML de la
  tâche, relais Rust, appel en tête de `main`). Limite acceptée : une ancienne tâche `cmd.exe` jamais
  réécrite n'est pas reconnue par le désinstalleur ; elle est réécrite au premier démarrage réglage actif.
  A5 reste au user.
- 2026-11-22 : revue adverse de la piste A (Sonnet, une ronde) : 0 P0, 0 P1, 2 P2 retenus et corrigés dans
  le lot car ils touchent l'objectif (aucun test ne tenait l'absence de fenêtre ; erreurs du relais perdues
  en release) : `afa0619c54`, tests rejoués par le pilote (2 + 11 verts). 9 conditions vérifiées, dont le
  découpage réel par CommandLineToArgvW et une tâche d'essai au chemin à espaces.
- 2026-11-22 : B1 et B2 écrits (`072778c920`, `052891145f`) ; diff du rattrapage relu par le pilote.
- 2026-11-22 : B3 et B4 écrits (`026c202d63`, `138be40f90`, pause `8ecd35633d`, libellé `6d01202319`).
- 2026-11-22 : revue adverse web de B (Sonnet) : 14 conditions tiennent ; retenus et envoyés en correction :
  texte faux en pause avec historique incomplet (P1, résultat faux affiché), bouton actif pendant un sync
  lancé ailleurs (action refusée à l'écran), tests manquants (réponse d'un autre joueur, statuts terminaux).
- 2026-11-22 : revue adverse Go de B (Sonnet) : 9 conditions tiennent, 2 P2. (1) échecs carnage/events
  non comptés dans les erreurs : jugé hors lot, le rattrapage saute les matchs connus avec ou sans la clé,
  la non-reprise d'un match dégradé préexiste (Découvertes). (2) lecture RO du statut toutes les 3 s,
  course possible avec une ouverture RW du sync dans le même process : correction retenue, mémoriser un
  `true` (définitif) et espacer la relecture d'un `false`, à faire après la fin de l'agent web (même
  worktree).
- 2026-11-22 : corrections de relecture faites : web `b446b15894` (pause avec historique incomplet, sondage
  au repos de 15 s, tests du joueur et des statuts terminaux), Go `e2aa51dbfa` (mémoire de `history_complete`
  dans le dépôt, oubli branché sur l éviction de purge) ; gates de l agent verts, archlint complet compris.
- 2026-11-22 : lecture de la prod : synchronisation automatique active (cycle de 15 min, 11 joueurs) ;
  `joueur_1194` absent de la prod. Découverte ci-dessous (alerte « zero-insert »).

## Découvertes

- `archlint/childproc_single_spawn_test.go` balaie encore `libs/...`, absent depuis la bascule des briques :
  le code des briques n'est plus gardé (balayage des dépendances du 2026-11-22 : aucune n'y lance de
  processus).
- `pkg/duckdbbackup` lance restic par `exec` nu, hors du garde-rail (`pkg/` exclu).
- `AddFriendFlow` (web) appelle `POST /sync/initial`, réservé aux admins : un non-admin qui ajoute un ami
  pourrait recevoir un refus. Non vérifié.
- Prod : l'alerte `auto_sync: zero-insert prolongé` sonne à chaque cycle pour 4 joueurs (plus de 11 000
  fois chacun depuis juin) alors que leurs matchs arrivent par le déclencheur de fin de partie : le compteur
  ne voit pas les matchs rangés par une autre voie. Bruit de journal, à porter au backlog.
- Halo 5 : un match rangé sans participants ni chronologie (échec carnage ou events) n'est jamais repris,
  ni par le delta ni par le rattrapage (matchs connus sautés) ; `EventsFailed` / `CarnageFailed` ne comptent
  pas comme erreurs du cycle (`games/halo_5/capture.go`, `livesync/runner.go`). Préexistant.
- La tâche `LevelUp-disk-hygiene` du poste de développement lance PowerShell chaque nuit (fenêtre possible
  sur le poste du user seulement, hors application).

## Clôture (2026-11-22)

- A5 `[!]` : vérification à l'écran réservée au user (installeur de la release : `schtasks /Run` sur la
  tâche `LevelUp\Refresh-*`, aucune fenêtre), et de même la carte Synchronisation à l'écran.
- Branche intégrée `integ/fenetres-sync` (un commit par piste + correctif du cliquet de taille des fonctions
  web : `SyncCard` passait à 87 lignes, extrait en `SyncLaunchControls` et `useSyncCardLaunch`).
- Gates sur la branche intégrée : golangci-lint 0 issue ; `check_test_baseline.sh` (LevelUp, décodeur, API
  Halo) sans échec ni test perdu ; typecheck, eslint, lint des frontières, vitest complet ; `make
  desktop-test`. `replay-equiv` `[!]` : non jouable dans un worktree sans archive des films (« aucun chunk
  de données »), diff sans rejeu ni décodeur ; ne pas monter de jonction vers `data/` (incidents du 16/09 et
  du 13/11).
- Découvertes portées au backlog (zero-insert, matchs Halo 5 dégradés, garde childproc) ; `AddFriendFlow`
  et la tâche disk-hygiene restent ici, non vérifiés ou hors application.
