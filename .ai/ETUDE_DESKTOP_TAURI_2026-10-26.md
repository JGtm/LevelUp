# Étude — version Desktop de LevelUp (Tauri) — 2026-10-26

Statut : **étude**, rien d'implémenté. Sert d'entrée au plan de la phase P0 (à cadrer sous
`plan-review`). Les décisions ouvertes (§11) sont à trancher par le user avant tout plan.

Méthode : quatre explorations en lecture seule (runtime serveur, auth, frontend, données et
release), puis vérification sur pièces par le pilote des points qualifiés de défaut (§3.1, §8).
Les références `fichier:ligne` datent du 2026-10-26 (branche `release/v8.0.7`) : RE-VÉRIFIER
avant de coder.

---

## 1. Verdict

Faisable, et la base s'y prête :

- le serveur écoute déjà sur `127.0.0.1` par défaut (`LEVELUP_API_HOST`), en `AuthMode=none` ;
- bases créées au premier lancement, métadonnées pré-construites extraites d'un zip
  (`cmd/server/prebuilt_metadata.go`), assistant de premier lancement (`features/setup`) ;
- connexion Xbox par code d'appareil (SISU) sans application Azure ni redirection ;
- la cuisson des films tourne déjà dans un processus enfant (`-film-child`, `filmproc/runner.go`),
  basse priorité, un film à la fois sur la machine (`filmproc/solo.go`), plafond mémoire ;
- le port s'ouvre avant les migrations et sert le SPA avec un 503 `server_starting`
  (`cmd/server/boot_gate.go`) : l'écran « Démarrage… » est presque gratuit.

Mais huit chantiers serveur (§3) précèdent tout emballage, dont un défaut qui touche déjà
l'usage local (§3.1).

## 2. Architecture recommandée

**Tauri 2 en coquille + le serveur Go en sidecar.** La webview charge
`http://127.0.0.1:<port>` servi par le Go, pas un dist embarqué.

Pourquoi :

- **Même origine** : cookie de session `SameSite=Lax`, contrôle CSRF par `Origin`, `/static`
  racine-relatif, routeur en historique navigateur avec repli SPA côté serveur : tout marche
  tel quel. Un dist sous `tauri://localhost` rendrait le cookie tiers (non envoyé) et
  obligerait à réécrire base d'API, `/static` et repli SPA.
- **Un seul binaire serveur** pour l'hébergé et le Desktop : aucun contrat qui diverge, les
  tests existants couvrent les deux.
- **Rust réduit à la coquille** (quelques centaines de lignes) : lancement du sidecar,
  démarrage, barre des tâches, mise à jour, instance unique, fermeture propre.

| Option | Pour | Contre |
|---|---|---|
| **Tauri 2** (retenu) | Updater signé et installeurs NSIS/MSI mûrs (updater 2.10.x, 2026) ; plugins single-instance, deep-link, autostart, notification, window-state, tray, opener | Toolchain Rust ; deux processus ; fermeture propre à construire sous Windows (§3.4) |
| Wails | Natif Go, un seul processus (le routeur chi peut être monté en `AssetServer.Handler`), fermeture triviale | v2 sans updater ni vrai tray ; v3 (qui les a) encore en bêta (v3.0.0-beta.12 au moment de l'étude) ; un panic serveur tue l'UI |
| Electron | Chromium identique partout (WebCodecs pour l'export vidéo du rejeu) | ~150 Mo de plus, aucun gain face à WebView2 sous Windows |
| Client léger vers lvelup.info | Trivial | Aucune valeur ; à garder comme option « se connecter à un serveur » dans la même coquille |

Cible : **Windows d'abord** (joueurs PC de Halo Infinite ; WebView2 = Chromium, WebCodecs
disponible). macOS ensuite (`release.yml` construit déjà darwin-arm64, jamais testé en CI).

Modèle : **Desktop autonome**, mono-utilisateur, bases et sync locales. L'hébergé reste
l'instance communautaire et la vitrine (avec la démo).

## 3. Prérequis serveur (par gravité)

Les points 1 à 6 bloquent un Desktop propre.

### 3.1 Le refresh token n'est pas persisté en mode `none` — défaut vérifié en lecture

En `AuthMode=none` (défaut), `server_apiv1.go:356-394` branche `PasswordLinkStrategy` ; elle
retourne `ErrSessionNotAuthenticated` sans compte local, et `completeLogin`
(`api/handlers/auth_login_complete.go`) ne pose alors que le jeton Spartan dans la SESSION.
Seul `XboxSSOLinkStrategy.persistRTATokens` (`service/xbox_auth_service.go:287`), câblé en
mode `xbox` seulement, écrit `data/auth/watcher_tokens/{xuid}.json`. Conséquence : la sync
tombe à l'expiration du jeton de session ; seuls `token-capture` / `token-import` sèment le
store en mode `none`. `docs/INSTALL.md` affirme le contraire (doc probablement inversée).
**Non vérifié par exécution** : à confirmer par un premier lancement réel avant correction.

### 3.2 Données et ressources mélangées sous une même racine

`PathResolver` (`internal/domain/title/registry.go:558-1146`) impose `<repoRoot>/data/...` ;
la racine est détectée en cherchant `db_profiles.example.json` (`config/config.go:602-628`).
Sous Windows, `Program Files` est en lecture seule. Séparer :

- racine **ressources** (RO, installée) : `config/` (445 Ko), `static/` (108 Mo),
  `data/titles/*/reference/` (42 Mo, dont fonds de carte 30 Mo), `metadata-prebuilt.zip`,
  le dist web, `docs/RELEASE_NOTES.md` ;
- racine **données** (RW, `%LOCALAPPDATA%\LevelUp`) : bases, caches, sessions, auth,
  `app_settings.json`, `db_profiles.json`, `.env.local`.

Le plus gros chantier : le référentiel du rejeu doit sortir de `data/`.

### 3.3 Port dynamique et signal « prêt »

Écouter `127.0.0.1:0` et annoncer le port réel sur stdout en JSON
(`{"event":"ready","port":N}`). Aujourd'hui la bannière affiche `srv.Addr`, pas le port lié
(`boot_gate.go:246`).

### 3.4 Fermeture propre sous Windows

Pas de SIGTERM sous Windows ; le gestionnaire de signal de crash y est un no-op
(`main.go:1888`). Un sidecar tué = arrêt brutal = risque de rejeu WAL DuckDB (#7659,
`cmd/duckdb_7659_repro`). À faire :

- le serveur surveille la fermeture de son stdin (coquille morte = arrêt propre) ;
- point d'arrêt réservé à la boucle locale, protégé par un secret propre au lancement ;
- côté Tauri : intercepter `ExitRequested`, écran « Fermeture en cours… », attendre le budget
  existant (40 s, `shutdown_budget.go`) ;
- en mode Desktop, CHECKPOINT plus fréquents (extinction du PC, fermeture de session).

### 3.5 Sécurité locale

En `none`, tout processus local (ou une page web par DNS rebinding) peut lire l'API et
appeler les routes admin (`RequireAdmin` transparent, `authz.go:34`). Secret de session par
défaut public `CHANGE_ME_IN_PRODUCTION` (`config.go:236`), exigé seulement en
`LEVELUP_ENV=production`. Proposition : mode d'auth `desktop` :

- secret propre au lancement, transmis par la coquille, échangé contre le cookie au premier
  chargement ; toute autre requête refusée ;
- contrôle strict de `Host` (anti-rebinding) ; `LEVELUP_CORS_ORIGINS` = origine exacte du sidecar ;
- secret de session généré et conservé à l'installation ;
- consentement de mise en commun (ADR 0038) accordé d'office au seul utilisateur local,
  bannière de consentement masquée.

### 3.6 Migrations sûres pour un poste sans opérateur

Échec de migration seulement journalisé, le boot continue sur un schéma à moitié migré
(`main.go:467-471`) ; `title_schema_version` écrit mais jamais relu
(`migration/registry.go:168-195`) : aucune garde de retour en arrière. À faire : copie des
bases avant toute montée de schéma, arrêt net avec écran explicite si échec, refus d'ouvrir
une base plus récente que le binaire (indispensable avec une mise à jour automatique).

### 3.7 Instance unique

`dblease` est un mutex intra-process ; aucun verrou inter-process hors film
(`film_decode.lock`). Plugin single-instance de Tauri + fichier de verrou dans le dossier de
données (modèle `filmproc/solo.go`).

### 3.8 Dépendances externes

- Police Roboto via Google Fonts (`apps/web/index.html:23-25`) : l'auto-héberger (hors ligne,
  vie privée ; profite aussi à l'hébergé).
- ffmpeg/ffprobe sur le PATH (`media/ffmpeg_slot.go`, `media/ffprobe.go`) : le livrer (build
  LGPL, ~80 Mo) ou téléchargement optionnel ; sans lui les médias se désactivent (boot OK).
- restic (sauvegarde) : sur Desktop, remplacer par un export Parquet périodique vers un
  dossier choisi.

### 3.9 Règle d'architecture transverse

Pas de `if desktop` éparpillés : un **profil de déploiement** (`hosted` / `desktop` / `demo`)
exposé par `/bootstrap` en capacités fines (comptes, invitations, groupes, mise en commun,
admin, restic, Discord), le front branche sur les capacités (modèle `HasCapability` des
titres). Aujourd'hui il n'existe pas de mode unique : `LEVELUP_ENV`, `LEVELUP_DEMO_MODE` et
`AuthMode` jouent ce rôle par morceaux.

## 4. Jetons au réveil (lancement après une longue absence, sortie de veille)

### Existant

- rafraîchissement à l'usage, RT rotaté réécrit aussitôt + cache Halo purgé
  (`main.go:1499-1516`) ;
- classification des erreurs (révoqué / config / transitoire), marqueur `reauth_required`,
  bannière (`bootstrap_service.go:263`), porte des jetons morts (`sync/deadtoken/gate.go`) ;
- boucle 5 min avec vérification immédiate au boot (`platform/auth/refresh_loop.go`) — mais
  pour le seul compte suivi par le watcher.

### Manques constatés

1. **Pas de rattrapage au lancement** : le scheduler attend un intervalle complet (6 h par
   défaut) avant le premier tick (`scheduler/auto_sync_run.go:27`).
2. **Pas de vérification des jetons de sync au boot** : un jeton mort se découvre au milieu
   de la première sync.
3. **Veille** : le repli `time.Now().Add(50 min)` du cache Spartan
   (`platform/halo/player_token_cache.go:82`) se compare sur l'horloge monotone de Go, qui ne
   compte pas la veille sous macOS/Linux → jeton expiré servi au réveil → 401. Les expirations
   réelles (heure murale) ne sont pas touchées. Touche aussi le serveur actuel sur un poste
   qui se met en veille.
4. **Expiration par inactivité** : un RT jamais utilisé meurt (de l'ordre de 90 jours selon la
   politique Microsoft, non mesuré ici). Invisible sur le VPS (refresh toutes les 6 h), réel
   sur Desktop. Non réparable automatiquement : la cause est connue (inactivité), la seule
   issue est une reconnexion par l'utilisateur (code d'appareil) — ce n'est pas une
   re-capture pour masquer un bug (ADR 0023).

### Séquence « réveil » proposée (au lancement et en sortie de veille)

Sortie de veille détectée côté Go, sans dépendre de Tauri (écart heure murale / horloge
monotone d'une minute à l'autre).

1. Attendre le réseau ; une erreur réseau ne marque jamais un jeton mort.
2. Rafraîchir le RT de chaque profil suivi, écrire le RT rotaté avant toute autre action →
   état « OK » ou « à reconnecter ».
3. OK et dernière sync plus vieille que l'intervalle : rattrapage, stats d'abord (progression
   affichée), films ensuite en arrière-plan basse priorité.
4. Mort : bannière « Reconnecter Xbox » (code d'appareil), aucune sync.
5. Garder le RT en vie app fermée : démarrage avec Windows réduit (opt-in), ou tâche planifiée
   hebdomadaire lançant le binaire en mode « rafraîchir seulement » (quelques secondes, sans
   fenêtre).

## 5. Coquille Tauri

- écran de démarrage puis fenêtre principale, taille et position mémorisées ;
- liens `target="_blank"` (~10 sites, dont xbox.com/activate) vers le navigateur système ;
- téléchargements (export vidéo du rejeu par blob, CSV par navigation) : boîte « Enregistrer
  sous » native ;
- barre des tâches : état de la sync, « Synchroniser maintenant », « Ouvrir », « Quitter » ;
- liens profonds `levelup://t/halo_infinite/match/...` ;
- démarrage avec Windows réduit (opt-in) ;
- notifications natives : le front interroge déjà toutes les 30 s
  (`notifications/queries.ts:52`) ; dans Tauri il relaie « Match analysé », « Nouveau
  record », « Citation obtenue ».

## 6. Ce que le Desktop fait mieux que l'hébergé

| Fonction | Base existante | Valeur |
|---|---|---|
| Analyse automatique en fin de match | watcher : présence Xbox toutes les 10 s, sync en fin de match (`watcher/player_watcher.go`) | Fonction phare ; en local, détecter le processus du jeu au lieu de sonder Xbox |
| Mode jeu | cuisson en priorité basse, un film à la fois | Ne pas cuire pendant que Halo tourne (énumération de processus seulement, aucune injection : anti-triche) |
| Rejeu 2D en local | placement `local` (défaut dev), ~50 s et jusqu'à 3 Go par film en enfant | Le VPS ne cuit pas ; chacun cuit ses rejeux |
| Import des captures | `internal/media` (ffmpeg, vignettes, HLS) | Surveiller le dossier de captures |
| Hors ligne | lecture des bases locales | Consultation sans réseau |
| Import depuis l'hébergé | export Parquet par joueur (`levelup backup`) | « Récupérer mes données depuis lvelup.info » au premier lancement |
| Rapport de diagnostic | `server.crash.log`, journaux par catégorie | Zip journaux + versions, jetons expurgés, joint au formulaire de retour |
| Contribution aux rejeux (plus tard) | `cmd/replay-worker`, routes `/api/v1/internal/*` | Les Desktop volontaires cuisent pour l'hébergé |

Disparaissent ou se simplifient sur Desktop : admin multi-comptes, invitations, verrou
d'instance, consentement de mise en commun, Discord, restic. Amis, escouades, classements
restent possibles en suivant des profils avec son propre jeton (pool à un jeton : 5 req/s,
`main.go:1575`).

## 7. Disque, mémoire, taille

- Films bruts : **104 Go mesurés** sur le poste de dev (`data/cache/film_chunks`, 4 046
  films, ~27 Mo par film) sur 144 Go pour tout `data/` ; faits ~4,5 Mo et rejeu ~2 Mo par
  film. Sans rétention, un joueur assidu remplit son disque. Décision VPS actuelle : purge
  manuelle, pas d'éviction auto — à redécider pour le Desktop (§11 q4).
- Installation neuve : ~150 Mo de ressources ; la croissance vient des films.
- Mémoire : DuckDB plafonné à 512 Mo et 2 threads par connexion (`platform/duckdb/db.go`),
  aucun plafond global au processus serveur ; en ajouter un en mode Desktop.
- Installeur estimé : 150 à 250 Mo (binaire Go + DuckDB statique, `static/`, référentiel,
  ffmpeg éventuel) ; `static/` mérite un tri.

## 8. Releases hébergées et Desktop au même rythme

- **Une étiquette = deux livraisons** : le déploiement actuel (`deploy.yml`,
  `scripts/deploy.sh`) et `release.yml` étendu : build Windows CGO (existe déjà), build Tauri,
  NSIS, `latest.json` signé par la clé Tauri, publié sur GitHub Releases. Version unique
  depuis l'étiquette, injectée dans `tauri.conf.json` et `-ldflags -X main.version`.
  Aujourd'hui `release.yml` construit le dist web sans le joindre, et les binaires attendent
  `config/`, `static/`, le dist sur disque.
- **Signature de code** : sans elle, SmartScreen avertit à chaque installation. Azure Trusted
  Signing (~10 $/mois) ou certificat OV. macOS plus tard : notarisation (99 $/an).
- **Mise à jour** : la notification « nouvelle version » existante
  (`api/wire/notifications_boot.go`) compare seulement version précédente et courante, sans
  source externe. Sur Desktop : updater Tauri adossé au changelog in-app, proposé et non
  imposé, toujours après la copie de §3.6.
- **Côté hébergé** : police auto-hébergée, profil de déploiement, migrations à arrêt net,
  page « Télécharger pour Windows ». Aucune régression : même binaire.
- **Licence** : l'AGPL autorise la distribution binaire avec lien vers les sources. Le
  décodeur `internal/ooz` (GPLv3) reste exclu du serveur (`cmd/server/licence_gplv3_test.go`)
  et le Desktop n'en a pas besoin : les assets du jeu sont cuits hors ligne et versionnés, le
  serveur ne lit jamais l'installation du jeu. Assets Halo redistribués identiques à
  l'hébergé. Lien de dons à vérifier face aux règles d'usage du contenu de jeu de Microsoft.

## 9. Phasage

| Phase | Contenu | Taille |
|---|---|---|
| **P0 — serveur** (profite aussi à l'hébergé) | §3.1 à §3.9 ; séquence réveil et correction de l'horloge (§4) | L |
| **P1 — coquille minimale** | sidecar, démarrage, fermeture propre, liens externes, instance unique, NSIS, updater signé, CI | M |
| **P2 — expérience Desktop** | fin de match + mode jeu, notifications, tray, autostart, tâche planifiée de refresh, quota disque, ffmpeg, diagnostic, import depuis l'hébergé | M-L |
| **P3** | macOS, contribution aux rejeux, liens profonds | M |

P0 demande des ADR : profil de déploiement, emplacement des données, mode d'auth `desktop`,
amendement de l'ADR 0023 si chiffrement des jetons.

## 10. Découvertes hors périmètre (notées, non traitées)

- `LEVELUP_ROOT` et `LEVELUP_DATA` posés dans le Dockerfile et le compose, lus par aucun code
  Go (configuration morte).
- Commentaire « API FastAPI exposée en local » dans `docker-compose.yml` (vestige Python).
- `docs/INSTALL.md` affirme que l'assistant persiste le RT dans le store : faux en mode `none`
  (§3.1), à corriger avec le correctif.
- La migration en échec qui laisse booter (§3.6) concerne aussi l'hébergé.

## 11. Décisions ouvertes (user)

1. Tauri ou Wails ? Recommandation : Tauri.
2. Windows seul en v1 ? Recommandation : oui.
3. Desktop autonome (+ option « se connecter à un serveur ») ou client de l'hébergé ?
   Recommandation : autonome.
4. Films bruts sur Desktop : purge automatique une fois les faits extraits, quota réglable,
   ou manuelle comme sur le VPS ?
5. Chiffrer les jetons au repos (aujourd'hui JSON en clair, 0600) par DPAPI / gestionnaire
   d'identifiants Windows, derrière un port côté Go ? Amende l'ADR 0023.
6. Budget de signature de code ; ffmpeg livré ou téléchargement optionnel ?

## Sources externes

- Wails releases : https://github.com/wailsapp/wails/releases
- Tauri updater changelog : https://tauri.app/release/updater/all-versions/
- Tauri shell (sidecar) changelog : https://tauri.app/release/shell/all-versions/
