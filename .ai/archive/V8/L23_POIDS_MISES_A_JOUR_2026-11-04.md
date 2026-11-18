# L23 - Poids des mises à jour : mesure et options (2026-11-04)

Statut : MESURE faite (23.1), décision prise le 2026-11-04 (23.2), mise en oeuvre faite (23.3, voir « Résultat » en fin de rapport) ; gate visuel de (e) à l'utilisateur.
Plan : `.ai/V8/PLAN_DESKTOP_2026-11-01.md` §9.19 (ligne L23) et §9.26. ADR 0045 D-17 (ffmpeg livré).

Unité : « Mo » = Mio (1 048 576 octets), comme les « 193 Mo » du plan. L'installeur fait 202 663 134 octets = 193,3 Mo.

## 1. Pièces mesurées

| Installeur | Run Release | Commit |
|---|---|---|
| A | 41586437921 | L22 (2026-11-03 05:52) |
| B | 41722766811 | L22b (2026-11-03 21:32) |
| C | 41816350092 | L22c (2026-11-04 10:46) ; identique à l'octet près (SHA-256 `ca0ff53c...`) à `Downloads\LevelUp-installeur\LevelUp_8.1.0_x64-setup.exe` |
| D | 41825436589 | commit de documentation seul (2026-11-04 11:59) |

Les quatre sont des installeurs `8.1.0` successifs de `release/v8.1.0` (aucun tag `v8.1.0` posé : la première release publiée du
bureau n'existe pas encore ; les releases `v8.0.x` publiées sont celles du site hébergé, sans installeur). Chacun a été ouvert avec 7-Zip 26.00 (`7z x`
sur l'installeur NSIS : méthode `LZMA:23`, archive SOLIDE, un seul flux de 202 610 394 octets), puis chaque fichier a été haché (SHA-256).

## 2. Composition de l'installeur (C, 193,3 Mo)

Méthode de la part compressée : l'archive NSIS étant solide, 7-Zip ne donne pas de taille compressée par fichier. Chaque famille a été
recompressée séparément avec `7z a -t7z -m0=lzma:d=8m:fb=64 -mx=9 -ms=on` (même algorithme et même dictionnaire de 8 Mo que NSIS). La somme des
familles (194,97 Mo en 7z) est 3,8 % sous la taille réelle du flux NSIS (202,61 Mo) : la colonne « part dans l'installeur » est la compression
mesurée de chaque famille MISE À L'ÉCHELLE du flux réel (facteur 1,0392). Précision attendue : quelques pour cent par famille, pas davantage.

| Famille | Non compressé (Mo) | Part dans l'installeur (Mo) | % |
|---|---:|---:|---:|
| ffmpeg.exe + ffprobe.exe + 7 DLL + LICENSE.LGPL (build LGPL « shared ») | 158,2 | 47,1 | 24,4 |
| `web/titles/halo_5` (1 265 PNG : plaques nominatives, emblèmes) | 40,0 | 41,4 | 21,4 |
| `levelup-server.exe` (Go + DuckDB CGO, DWARF et symboles inclus) | 154,8 | 32,9 | 17,0 |
| `data/.../reference/map_backgrounds` (228 WebP) | 29,1 | 29,4 | 15,2 |
| `static/medals` | 7,3 | 7,5 | 3,9 |
| `web/titles/halo_infinite` (PNG) | 6,6 | 6,9 | 3,6 |
| `static/maps` | 9,3 | 6,8 | 3,5 |
| `data/.../warehouse/metadata-prebuilt.zip` | 5,0 | 5,0 | 2,6 |
| `static/sounds` (183 Opus) | 4,3 | 4,5 | 2,3 |
| `static/commendations` | 2,2 | 1,9 | 1,0 |
| Bootstrapper WebView2 embarqué (`MicrosoftEdgeWebview2Setup.exe`) | 1,8 | 1,8 | 0,9 |
| `levelup-desktop.exe` (coquille Tauri, déjà `strip` + LTO) | 5,0 | 1,8 | 0,9 |
| `web/assets` (JS et CSS compilés) | 5,6 | 1,3 | 0,7 |
| `web` (reste : logos, icônes, polices, `index.html`) | 1,2 | 1,2 | 0,6 |
| `static/weapons-assets` | 1,1 | 1,1 | 0,6 |
| `data/.../reference` hors fonds de carte (JSON, CSV, structures) | 9,4 | 0,9 | 0,5 |
| `static/ranks`, racine de `static/`, `vehicles-assets`, `abilities-assets`, `prestige-assets` | 1,9 | 1,3 | 0,8 |
| `config/`, `data/titles/halo_5`, plugins NSIS, notices et licences | 0,5 | 0,2 | 0,1 |
| **Total** | **443,4** | **193,3** | **100** |

Regroupé par grande famille demandée :

| Grande famille | Part dans l'installeur (Mo) | % |
|---|---:|---:|
| ffmpeg + ffprobe | 47,1 | 24,4 |
| front `web/` (dont 41,4 d'images Halo 5 et 6,9 d'images Halo Infinite) | 50,8 | 26,3 |
| `static/` (dont 4,5 de sons) | 23,5 | 12,2 |
| référentiel `data/` | 35,4 | 18,3 |
| `levelup-server.exe` | 32,9 | 17,0 |
| bootstrapper WebView2 | 1,8 | 0,9 |
| `levelup-desktop.exe` (coquille) | 1,8 | 0,9 |
| `config/`, NSIS, notices | 0,2 | 0,1 |

Constats qui corrigent le plan :
- **ffmpeg pèse 47 Mo dans l'installeur, pas 80** (158 Mo décompressés : le build « shared » livre `avcodec-62.dll` à 87 Mo, `avfilter` 30 Mo, `avformat` 19 Mo,
  `swscale` 12 Mo ; les deux `.exe` ne font que 0,8 Mo). Les « ~80 Mo » de D-17 sont l'ordre de grandeur de l'archive téléchargée, pas de sa part compressée.
- **Les images Halo 5 du front (41 Mo) pèsent presque autant que ffmpeg** : 308 plaques de 1296x272 (32 Mo) et 320+ emblèmes, en PNG déjà peu compressibles. Ils n'étaient pas dans la liste des suspects du plan.
- **Le serveur Go pèse 155 Mo décompressés dont 39 Mo de sections `.debug_*` (DWARF)** et une table de symboles COFF : il est construit sans `-ldflags "-s -w"`.
- Les images sont déjà compressées (PNG, WebP) : 40 % de l'installeur (fonds de cartes, Halo 5, Halo Infinite) ne se compresse pas davantage par NSIS.

## 3. Ce qui change entre deux versions

Comparaison fichier par fichier (SHA-256) des quatre installeurs, deux à deux consécutifs (A vers B, B vers C, C vers D) :

| Famille | Part (Mo) | A vers B | B vers C | C vers D (doc seule) |
|---|---:|---|---|---|
| ffmpeg + DLL | 47,1 | identique | identique | identique |
| `data/` (référentiel) | 35,4 | identique | identique | identique |
| `static/` (images, sons) | 23,5 | identique | identique | identique |
| `web/titles/*` (images) | 48,3 | identique | identique | identique |
| `config/`, notices, plugins NSIS | 0,2 | identique | identique | identique |
| `levelup-server.exe` | 32,9 | DIFFÈRE | DIFFÈRE | DIFFÈRE |
| `levelup-desktop.exe` | 1,8 | DIFFÈRE | DIFFÈRE | DIFFÈRE |
| `web/assets/*` + `index.html` (noms hachés) | 1,3 (dont 0,70 de fichiers qui changent) | 358 fichiers, 3,3 Mo non compressés | 122 fichiers, 3,2 Mo non compressés | identique |
| Bootstrapper WebView2 | 1,8 | DIFFÈRE (Microsoft l'a mis à jour) | identique | identique |

Le test C vers D est instructif : un commit de documentation seul change quand même le serveur et la coquille (le binaire Go embarque `vcs.revision` et
`vcs.time`, le build Rust des horodatages). Aucune reproductibilité n'est à attendre de ces deux fichiers.

**Conclusion.** À une mise à jour typique, ce qui change réellement est : serveur 32,9 + coquille 1,8 + front compilé 0,7 = **35,4 Mo, soit 18 % de l'installeur**
(37,2 Mo si le bootstrapper change aussi). **Au moins 157 Mo (82 %) du téléchargement sont identiques octet pour octet à l'installation précédente.**

Réserve : ces quatre builds sont de la même journée. Sur plus d'historique, `git log v8.0.0..HEAD` (7 versions) montre que les familles lourdes bougent rarement : `static/` touché par 6
commits (la plupart sans effet sur le poids), `data/titles/*/reference` par 1, `metadata-prebuilt.zip` et `apps/web/public/titles` par 0, le choix de ffmpeg par 1 (son introduction),
`config/` par 13 (0,1 Mo). Une mise à jour entre deux versions réelles pourra donc aussi renouveler un peu de `static/` ou de référentiel ; le gros reste inchangé.

## 4. Ce que Tauri 2 et l'ADR 0045 permettent

- **Pas de delta.** Le plugin `tauri-plugin-updater` 2.12.0 (lu dans `~/.cargo/registry`) télécharge l'installeur ENTIER d'une URL (`latest.json`,
  `platforms.windows-x86_64.url`), vérifie la signature minisign du fichier entier, puis lance l'installeur (`install_inner(&self, bytes: &[u8])` : l'installeur
  est même gardé en mémoire). Aucune notion de différentiel, de fichier partiel ou de seconde ressource. Un « installeur différentiel » n'existe pas dans cette pile (NSIS non plus).
- **Le désinstalleur de l'ancienne version retire les fichiers qu'il a posés.** Ce que l'installeur dépose sous le dossier du programme est reposé à chaque version ; ce que la
  mise à jour ne reposerait pas serait supprimé. Donc « ne pas réembarquer une ressource inchangée » exige qu'elle vive HORS du dossier d'installation (par exemple sous la racine de données) et
  que quelque chose la fournisse.
- **ffmpeg : déjà prévu.** `media.ResolveFFmpeg` cherche `LEVELUP_FFMPEG_DIR`, puis le dossier de l'exécutable, puis `PATH`. Un ffmpeg posé sous `<DataRoot>/tools/...` se branche par une ligne de
  recherche de plus côté serveur (ou la variable d'environnement posée par la coquille, qui pose déjà les trois racines). Aucun changement du contrat coquille-serveur
  (événements stdout, arrêt, port, cookie) ; seule D-17 (« ffmpeg livré ») est amendée.
- **Ressources lourdes (`static/`, référentiel, images) : plus délicat.** Le serveur lit `web/`, `static/` et `data/titles/.../reference` sous `ResourcesRoot` (= dossier d'installation, posé par la
  coquille dans `LEVELUP_RESOURCES_DIR`). Un paquet sous `DataRoot` demande un second emplacement de lecture : cela ne change pas le contrat de la coquille mais précise la notion de
  « trois racines » de l'ADR 0045 (amende de l'amendement de l'ADR 0008). À écrire dans l'ADR avant le code.
- **Téléchargement hors installeur : qui le fait ?** Le serveur Go (déjà sous `netguard`/`httpx`), pas la coquille Rust : D-18 limite Rust à quelques centaines de lignes sans logique métier. La
  signature d'un paquet réutiliserait la clé de l'updater (minisign : le crate `minisign-verify` est déjà dans l'arbre de la coquille ; côté Go, une vérification équivalente est à écrire).

## 5. Composants ffmpeg réellement requis (pour l'option b)

Relevé dans `internal/media/*.go`, `internal/ops/media_tooling.go`, `internal/ops/media_thumbnails.go` :

| Rôle | Composants |
|---|---|
| Encodeurs | `libwebp` (miniatures WebP animées, `-quality 75`, `fps=12`, 480 px), `aac` (renditions audio HLS), un encodeur H.264 parmi `libx264`, `h264_mf` (Windows, celui du bureau) ou `libopenh264` ; `libopus` seulement pour les outils de développement (`cmd/sounds-encode`), pas au runtime |
| Muxers | `hls`, `mp4` (segments fMP4), `webm` (remux MKV/AVI), `webp`, `null` (sonde d'encodeur) |
| Filtres | `amix`, `alimiter`, `volume`, `aevalsrc`, `asplit`, `fps`, `scale`, `format`, `thumbnail`, `color` (source `lavfi` : demande `libavdevice`), plus ceux que ffmpeg insère seul (`aresample`, `aformat`, `null`) |
| Démuxeurs / décodeurs d'entrée | conteneurs `.mp4 .webm .mov .avi .mkv` ; images `.png .jpg .jpeg .bmp .gif` ; codecs vidéo et audio des captures de joueurs (H.264, HEVC, AV1, VP9 ; AAC, Opus, PCM, MP3, FLAC) : **liste ouverte, imposée par ce que les captures du monde réel contiennent** |
| Autres | `ffprobe` (`-show_streams`, sortie JSON) ; protocoles `file` et `pipe` |

Le risque d'un ffmpeg réduit est dans la ligne « décodeurs d'entrée » : un codec absent rend le clip illisible sans erreur évidente.

## 6. Options

Gains en Mo par mise à jour (le téléchargement complet actuel est de 193,3 Mo). « Mesuré » = obtenu sur pièces dans cette session ; « estimé » = raisonnement, non vérifié.

| Option | Gain par mise à jour | Coût | Risques | Ce que voit l'utilisateur |
|---|---|---|---|---|
| **(f) Serveur sans symboles : `go build -ldflags "-s -w -X main.version=..."`** (non demandé dans le plan : découvert par la mesure) | **-11,2 Mo** (mesuré : vrai build `-s -w` du worktree, 154,8 vers 94,0 Mo décompressés, 31,7 vers 20,9 Mo recompressés ; `-trimpath` n'apporte que 0,01 Mo). Gain aussi sur le disque : -61 Mo | S : une option de ligne de commande dans `release.yml`, `build-installer.ps1` (et `Makefile` si le même binaire l'emploie) ; `check-version.sh` lit `main.version=8.1.0"`, vérifié compatible avec le nouveau `-ldflags` | Plus de débogueur symbolique (delve) sur ce binaire ; les traces de panique restent complètes (la table des fonctions de Go est conservée). Le binaire n'a PAS été lancé ici (interdit) : le job `e2e` d'installation de la CI le valide | Rien. Installeur plus léger de 6 %. |
| **(e) Images Halo 5 (et Halo Infinite) recompressées en WebP** | **-27 Mo** pour les 308 plaques en WebP qualité 85 (mesuré sur 21 plaques : 2,19 Mo vers 0,31 Mo, -86 %) ; en WebP sans perte : -29 % seulement, soit -9 Mo. Emblèmes (5 Mo, transparence) et PNG Halo Infinite (6,6 Mo) non mesurés | S à M : script de conversion, références du catalogue (`spartan-catalog.json`) et du front, tests | Perte visuelle (qualité 85 : à juger sur pièces par l'utilisateur) ; la transparence des emblèmes impose de tester | Images Halo 5 plus nettes ou non selon le choix ; chargement des pages plus rapide aussi sur le site hébergé |
| **(a) ffmpeg posé à côté, téléchargé par le serveur** (premier lancement, ou à la demande à la première fonction média) | **-47 Mo** (-24 %) | M : téléchargement (serveur Go) avec somme de contrôle épinglée, hébergement du zip sur NOTRE release (les « autobuilds » de BtbN sont purgés), recherche sous `<DataRoot>/tools/`, état « ffmpeg en cours d'installation » dans les pages Médias (le bloc « encodeur H.264 absent » de L20 existe), nettoyage à la désinstallation, tests | **Contredit D-9/D-17 (autonomie hors ligne, écartée par l'utilisateur)** : pas de miniatures, de HLS ni de remux tant que le réseau manque au premier besoin ; antivirus qui regarde un exécutable téléchargé ; version de ffmpeg à faire évoluer indépendamment de l'application | Installation plus légère de 24 % ; première utilisation des médias avec une attente |
| **(b) ffmpeg réduit** (build `--disable-everything` + composants du §5) | **-30 à -40 Mo** (estimé : un ffmpeg ciblé fait de l'ordre de 20 Mo non compressé, 7 Mo dans l'installeur ; NON mesuré, il faut le construire) | L : construire un ffmpeg LGPL sous Windows en CI (MSYS2, 30 à 60 min de build, mise en cache), épingler ses sources, redistribuer les sources correspondantes (obligation LGPL, aujourd'hui assurée par le lien vers l'archive de BtbN) | Codec d'entrée oublié (AV1, HEVC des captures) : clips illisibles ; correctifs de sécurité de ffmpeg à reconstruire soi-même ; coût de maintenance permanent | Rien si la liste est juste, clips refusés sinon |
| **(c) Paquet « ressources » versionné à part** (`web/titles`, `static`, référentiel, éventuellement ffmpeg ; retéléchargé seulement si son empreinte change) | **-106 Mo** sans ffmpeg (images Halo 5 41,4 + fonds de carte 29,4 + `static` 23,5 + `metadata-prebuilt` 5,0 + Halo Infinite 6,9), **-153 Mo** avec : le téléchargement tombe à **~29 Mo** (serveur 21,7 avec (f), coquille 1,8, bootstrapper 1,8, front et divers 3,6 ; la part qui change d'une version à l'autre n'est que de ~24 Mo). Mesuré sur pièces : ces familles sont identiques sur les quatre builds | L : un second emplacement de lecture des ressources côté serveur (§4), ADR 0045 amendé, publication et signature d'un zip par release, téléchargeur avec reprise, état d'avancement dans l'application, premier lancement hors ligne, nettoyage à la désinstallation, adaptation du job `e2e` et du banc de gate de l'installeur | Le plus gros chantier ; contredit D-9 si ffmpeg y entre ; la logique « ressource absente » doit être gracieuse partout (images, sons) ; le premier lancement dépend du réseau | Première installation plus légère ; premier lancement plus long (téléchargement) ; mises à jour de 29 Mo |
| **(d) Ne rien faire** | 0 | 0 | Aucun technique | 193 Mo à chaque mise à jour : 32 s à 50 Mbit/s, 2 min 40 à 10 Mbit/s |
| (g) Bootstrapper WebView2 en `downloadBootstrapper` (au lieu d'`embedBootstrapper`) | -1,8 Mo | S (une ligne de `tauri.conf.json`) | Installation sur un poste sans WebView2 (Windows 10 nu) exige le réseau ; WebView2 est présent d'office sous Windows 11 | Rien sur Windows 11 |

Fréquence : 11 releases publiées entre le 2026-08-04 et le 2026-11-04 (7.3.0 à 8.0.7), soit environ 3,7 par mois, groupées en rafales (8.0.0 à 8.0.7 en 6 jours). Ce sont les releases du
site hébergé ; aucune release de bureau n'est encore publiée, donc la fréquence des mises à jour du bureau et leur nombre d'utilisateurs sont inconnus. Hypothèse prudente de 2 à 4 mises à jour
par mois et par utilisateur : 0,4 à 0,8 Go par mois et par utilisateur avec l'état actuel. L'hébergement des releases (GitHub) ne coûte rien ; le coût est celui du temps de l'utilisateur et de sa bande passante.

Combinaisons (téléchargement complet d'une mise à jour, en Mo) :

| Combinaison | Téléchargement | Gain |
|---|---:|---:|
| Aujourd'hui | 193,3 | |
| (f) | 182,1 | -6 % |
| (f) + (e) | 155,1 | -20 % |
| (f) + (e) + (a) | 108,0 | -44 % |
| (f) + (c) sans ffmpeg | 75,9 | -61 % |
| (f) + (c) avec ffmpeg | 28,8 | -85 % |

## 7. Recommandation

1. **Faire (f) tout de suite.** C'est le meilleur rapport gain/risque du rapport (-11,2 Mo, une option de compilation, rien de visible, déjà mesuré par un vrai build) et il n'est
   pas dans le plan : il vient de la mesure. À livrer avant le tag pour que le premier installeur publié soit déjà léger, car chaque installeur publié reste téléchargé.
2. **Faire (e) si l'utilisateur valide le rendu visuel** : -27 Mo pour un coût faible, sans rien changer à l'architecture ni aux décisions D-9/D-17, et utile aussi au site hébergé. La qualité (85, sans perte ou autre) se
   juge sur des images, donc à l'utilisateur.
3. **Ne pas faire (b)** : le gain (30 à 40 Mo) est inférieur à celui de (c)/(a) et le coût de maintenance (build ffmpeg maison, correctifs de sécurité, risque de codec d'entrée manquant) est permanent.
4. **(a) et (c) sont une décision de principe, pas d'ingénierie** : elles rendent le premier lancement dépendant du réseau, ce que D-9 a écarté il y a trois jours. Je proposerais d'**attendre un retour
   réel** après la première release publiée (nombre d'utilisateurs, fréquence des mises à jour, connexion) avant de rouvrir D-9. Si l'utilisateur veut agir maintenant sur la
   deuxième plus grosse part, (c) en un seul paquet (ffmpeg compris) est préférable à (a) seul : il traite 153 Mo avec un seul mécanisme, alors que (a) en traite 47 pour presque le même coût d'infrastructure.
5. **(d) reste défendable** pour un public réduit sur de bonnes connexions : après (f) et (e), 155 Mo.

Avec (f) + (e), le poids passe de 193 à 155 Mo (-20 %) sans toucher à une seule décision déjà prise.

## 8. Décisions attendues de l'utilisateur (23.2)

1. (f) serveur sans symboles : oui ou non, avant le tag `v8.1.0` ?
2. (e) images Halo 5 : faut-il recompresser, avec quelle qualité (WebP 85 mesuré -86 % ; sans perte -29 %), et les emblèmes avec transparence sont-ils inclus ? Un échantillon converti (21 plaques, deux qualités) est dans le dossier de mesure du §10 pour en juger.
3. (a), (b) ou (c) : maintenir D-9 (ffmpeg et ressources livrés, mises à jour complètes) ou le rouvrir ? Si rouvert, choix (a) ffmpeg seul ou (c) paquet complet, et moment (avant ou après le tag).
4. (g) bootstrapper WebView2 téléchargé : oui ou non (gain de 1,8 Mo, faible).

Note : la ligne L23 du plan citait aussi « `cargo audit` en CI » : c'est fait (22c.2, job `desktop-audit` de `ci.yml`), rien à reprendre ici.

## 9. Ce qui n'a pas pu être mesuré

- La taille réelle d'un ffmpeg réduit (option b) : il faudrait le construire.
- La compression NSIS exacte par famille : l'archive est solide, les parts sont estimées par recompression 7z à dictionnaire équivalent, mises à l'échelle (±4 % sur le total).
- Le comportement du serveur compilé en `-s -w` (le lancer était interdit) ; la forme du binaire, de ses `-ldflags` et de `check-version.sh` a été relue, l'exécution reste à valider par le job d'installation de la CI.
- La qualité visuelle des WebP recompressés (aucune image regardée), et le gain sur les emblèmes et sur les PNG Halo Infinite.
- Une mise à jour réelle (aucun poste n'a été touché ; la durée de 32 s à 50 Mbit/s est un calcul, non une mesure).
- La fréquence des mises à jour du bureau et le nombre d'utilisateurs : aucune release de bureau n'est publiée.
- Les versions de l'historique d'avant le 2026-11-03 : seuls les artefacts de CI des quatre runs ci-dessus étaient disponibles.

## 10. Pièces de mesure

Extractions, empreintes et archives de test dans le répertoire temporaire de session (`...\scratchpad\l23\`, non versionné) : `hashes.json` (SHA-256 des fichiers des quatre installeurs),
`comp/*.7z` (compressions par famille), `srv_sw.exe` (build `-s -w` du serveur, non lancé). Commande type de la compression :
`7z a -t7z -m0=lzma:d=8m:fb=64 -mx=9 -ms=on <famille>.7z <fichiers>`.

## 11. Résultat (23.3, 2026-11-04)

Décisions de l'utilisateur du 2026-11-04 : (f) OUI ; (e) WebP qualité 85 OUI sous gate visuel ; (a), (b), (c) NON (D-9 maintenue) ; (g) NON.

| Pièce | Avant | Après | Témoin |
|---|---:|---:|---|
| `levelup-server.exe` (`-s -w`) | 163 138 322 octets (155,6 Mo) | 98 608 640 octets (94,0 Mo) | `go build` local, mêmes `-X main.version=8.1.0` ; `check-version.sh` vert sur le binaire sans symboles et rouge sur une autre version |
| Images Halo 5 du front (1 257 fichiers) | 39,67 Mo | 8,97 Mo | `node scripts/convert-halo5-images.mjs` |
| **Installeur** | **202 663 134 octets (193,3 Mo)** | **159 348 542 octets (152,0 Mo)** | `build-installer.ps1 -SkipWeb` après `npm run build` ; `e2e-install.ps1` code 0 (installation, deux lancements avec arrêt propre, désinstallation) |

Gain total -41,3 Mo (-21 %) : un peu plus que les 155 Mo prévus au §6 pour (f) + (e), parce que les emblèmes et les vignettes ont aussi été comptés (-2,9 Mo) et que les plaques sont tombées à 4,3 Mo (-87 %).

Traces de panique : un binaire d'essai compilé en `-s -w` panique avec noms de fonctions, fichiers et lignes complets (la table `pclntab` de `runtime` est conservée) ; le contrat est écrit dans le commentaire de `release.yml`. `TestE2E` (`-tags=e2e_windows`) compile désormais le serveur en `-s -w` et reste vert.

Écart à la décision (e), justifié par mesure : les masques du personnalisateur Spartan (emblèmes, vignettes) sont codés par canaux (R = primaire, V = secondaire, B = tertiaire) ; un WebP avec perte les salit (4:2:0). Mesure sur la recoloration, part des pixels écartés de plus de 16 niveaux sur 255 : plaques 2,0 % en moyenne (13,5 % la pire, n° 101), emblèmes 3,5 %, vignettes d'emblème 16 % (échantillons de 6), sans perte 0 %. Les plaques (grandes, 87 % de gain) restent en qualité 85 ; emblèmes et vignettes sont sans perte (-37 % et -23 %).

Gate visuel : `C:\Users\Guillaume\Downloads\webp-halo5-avant-apres\comparer.html` (douze paires : quatre emblèmes à transparence, quatre plaques dont la pire des 308, le fond et trois vignettes ; les PNG et WebP bruts, plus leur recoloration avec une palette bleu, gris, or, et le poids de chaque fichier). Si l'utilisateur refuse, le commit des images est retiré.

Non fait : lancer l'application avec la nouvelle interface (aucun navigateur ni coquille lancés, par consigne) ; le rendu réel des WebP dans le canvas de recoloration n'a donc été vérifié que par décodage libwebp (Pillow) et par les tests vitest, pas dans le moteur du navigateur.
