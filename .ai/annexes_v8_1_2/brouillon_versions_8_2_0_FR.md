# Notes de version 8.2.0, BROUILLON FR (2026-11-12, mis à jour le 2026-11-13)

Base : `origin/release/v8.2.0` à `31f91a79e8` (PR #181 à #206 fusionnées ; la PR #207 est ouverte, voir plus bas).
Rien n'est posé dans `docs/` : les deux tests
`TestReleaseNotesDocs_*` exigent que le bloc le plus récent des notes porte le numéro de la première entrée
versionnée du changelog, donc le bloc `v8.2` et l'entrée `[8.2.0]` se posent ENSEMBLE à la clôture, avec la
montée de version (`Cargo.toml`, `Cargo.lock`, `tauri.conf.json`). Le pendant anglais est `brouillon_versions_8_2_0_EN.md`.

Les lignes marquées **[À CONFIRMER]** dépendent d'une PR encore ouverte, la **#207** (cuisson à 30 images par
seconde, pictogramme et tués sur les destructions de véhicule, décodeur v1.3.6, schéma 115) : la PR est
ouverte, sa fusion est suspendue au verdict de cinq lignes « non établi » du gate du corpus. Les retirer ou les
finaliser avant la clôture, aucune ne doit partir telle quelle. La #203 (destructions de véhicule, décodeur
v1.3.4, schéma 113), la #205 (fonds de carte par le maillage de navigation), la #206 (alertes de sécurité des
dépendances) et les étapes 4 à 8 du plan d'architecture sont fusionnées : leurs lignes sont posées, sans
marque. À la lecture du 2026-11-13, sans la #207 la cuisson reste à 10 images par seconde (pas de 100 ms).

## 1. Bloc pour `docs/FR/RELEASE_NOTES.md`

À coller en tête du fichier, sous `## Dernières nouveautés`, avant le bloc `**v8.1.2 — ...**`. La première
section est reprise telle quelle comme notes de l'updater et de la page de la version.

````markdown
**v8.2 — Un rejeu plus complet, des fonds de carte plus justes, la Mainmise, de nouveaux graphiques de match et des rejeux compressés**

**Le rejeu**
- **Un fil d'événements qui se filtre** — le titre du fil ouvre un menu : Tout, Éliminations, Objectifs, Armes et bonus, Véhicules, Médailles, Assistances. Les actions d'objectif apparaissent en bandeaux, les armes ramassées au sol entrent dans le fil, et un filtre sans aucune ligne dans le match le dit au lieu de rester vide. Ton choix est mémorisé
- **Un clic sur une élimination** — la lecture se met en pause à cet instant, et le tueur et la victime sont mis en valeur sur la carte, avec leur nom et l'arme
- **La zone de l'élimination** — chaque élimination du fil indique la zone de la carte où se tenait le tueur, quand la carte en nomme une
- **Un bouton « Vue »** — dans le coin de la carte, quatre préréglages de calques (Complet, Combat, Déplacements, Objectif) et un panneau qui regroupe les effets, la chaleur, la couleur des points et le son. Le bouton Réglages de la barre de lecture disparaît
- **La fiche d'un joueur dans un panneau** — un clic sur une fiche remplace le fil par ses totaux de fin de match (précision, dégâts, détail du score, objectif, armes), avec Suivre, Seul et Masquer. Échap referme le panneau
- **Le film mieux lu** — les positions sont lues aux bonnes largeurs sur les cartes à plusieurs régions, le type des objets et les capacités du Spartan sont reconnus avec plus de rigueur : un rejeu régénéré gagne en précision
- **Tourelles fixes, bots et lieux de mort** — une tourelle fixe est reconnue comme telle, des pistes parasites ne s'affichent plus, les bots d'un film sans fil d'éliminations gardent leur nom, et des éliminations qui n'avaient aucun lieu en retrouvent un
- **Des icônes du jeu pour les grenades et les capacités** — les vignettes du rejeu viennent des planches d'interface du jeu, au trait pour les grenades
- **Les véhicules détruits** — le fil et son filtre « Véhicules » listent chaque destruction de véhicule (« Ghost détruit », ou « a détruit : Ghost » quand le destructeur est connu), avec entre parenthèses le joueur à bord, ou leur nombre quand ils sont plusieurs. Les joueurs morts avec leur véhicule sont lus dans le film ; un joueur éjecté puis tué, ou tué juste avant le véhicule, n'est pas compté. Les destructions apparaissent dans un rejeu régénéré
- **Des postures et des armes au sol lues plus justement** — un sprint ou une escalade ne reste plus ouvert après la mort du joueur, et une arme au sol est plus souvent reliée à celui qui la ramasse
- **Des fonds de carte tenus par le maillage de navigation** — les vingt cartes natives sont recadrées sur la zone réellement jouable, le sol de référence vient du maillage du jeu, et un nouveau calque « Plan des murs » trace les murs en vecteurs entre le fond et les zones (interrupteur dans le panneau « Vue », allumé par défaut, présent pour 111 fonds sur 114). Live Fire n'a pas de maillage : son fond reste celui d'avant
- **Une carte qu'on tourne** — un réglage « Orientation » du panneau « Vue » fait tourner la carte par quarts de tour, dans le sens horaire, et le retient pour chaque carte. La Tactique et « Occupation du terrain » lisent la même orientation et le même plan des murs
- **[À CONFIRMER, #207 ouverte] Rejeux à 30 images par seconde** — les mouvements sont plus fluides, position, visée et bouclier compris ; un rejeu cuit à 10 images par seconde est reconnu comme « à refaire » (« cadence périmée »), sans rien recuire seul
- **[À CONFIRMER, #207 ouverte] Les destructions de véhicule, complétées** — la ligne « a détruit » du fil porte le pictogramme de l'arme qui a détruit le véhicule, et la parenthèse dit « (N tués avec le véhicule) » quand des joueurs sont morts avec lui

**La vue du match et la Mainmise**
- **Deux graphiques dans l'onglet Chronologie** — « Écart de score » et « Supériorité numérique » (joueurs en vie, bots compris), en aires autour de zéro et aux couleurs des équipes. Avec plus de deux équipes, la comparaison se fait contre la meilleure équipe adverse ; sans rejeu, la carte le dit
- **Mainmise** — la part de la durée du match passée seul en tête du score, avec le nombre de retournements. Sur la vue du match, dans Sessions, Séries temporelles (par match, par mois, par carte) et Escouade, pour Halo Infinite. Un tiret remplace le chiffre quand il ne peut pas être calculé, par exemple dans un mode où le score avance trop rarement (drapeau, colline, bombe)

**Des rejeux compressés**
- **Cinq fois moins de place** — les rejeux sont rangés compressés, contenu identique : sur un parc local, 3,6 Go deviennent 0,75 Go. Les anciens rejeux restent lisibles sans rien faire
- **Compression depuis l'application Windows** — la mise à jour des rejeux lancée depuis l'application les écrit compressés, et propose aussi de compresser ceux qui ne peuvent plus être refaits

**Notes et statistiques**
- **Les notes LUSR sont recalculées au premier démarrage** — un match que le calcul ne parvenait pas à résoudre reçoit enfin une note. Comme chaque note dépend des précédentes, la plupart de celles de ton historique bougent, le plus souvent de moins de cinq points
- **FDA nette dans Solo contre Escouade et dans les alertes de comportement** — la FDA se lit en frags plus un tiers des assistances moins les morts, par match, et non plus en quotient ; la formule de l'Aide est corrigée dans chaque langue, et l'alerte de série de défaites se juge sur une chute de FDA en points
- **Les alertes de comportement dans ta langue** — onze phrases composées en français par le serveur passent par le catalogue de langues
- **Probabilité de victoire retirée** — statistique abandonnée, retirée des tableaux Synergies et Progression et du détail d'un match

**L'interface et l'application Windows**
- **Une barre latérale plus compacte** — le pied devient une rangée d'icônes, replié il garde un menu « Plus », et une infobulle ne s'ouvre au focus que si le focus vient du clavier
- **Un volet des référentiels qui glisse** — cartes, armes et médailles s'ouvrent par le côté, avec un mode épinglé
- **Un bouton « Parcourir »** dans les réglages de l'application Windows pour choisir un dossier
- **Des mises à jour plus propres** — les anciens fichiers de l'interface sont retirés à la mise à jour et à la désinstallation
- **Des aperçus de liens dans ta langue** — un lien copié depuis l'application porte ta langue, un lien nu s'affiche en anglais
- **Un même tri dans les tableaux** — les tableaux d'administration, les joueurs croisés de la Carrière et le Classement trient par la même règle : premier clic dans le sens propre à la colonne, bascule ensuite entre croissant et décroissant, valeurs absentes toujours en dernier
- **Deux graphes qui suivent la palette d'accessibilité** — l'écart cumulé de frags (Palmarès, Explorateur) et la bande des séquences changent de couleur dès que la palette change, sans recharger la page
````

## 2. Bloc pour `docs/FR/CHANGELOG.md`

Remplace `## [Unreleased]` par un `## [Unreleased]` vide, puis pose ceci en dessous, la date étant celle de la
publication. L'entrée « Icônes du jeu » reprend celle déjà présente sous `[Unreleased]`. Le fichier est lu à
l'exécution par la page des modifications : ne rien y laisser de **[À CONFIRMER]** à la fusion.

````markdown
## [8.2.0] - AAAA-MM-JJ

> Le rejeu, les fonds de carte et la Mainmise côté joueur ; côté code, LevelUp consomme les quatre briques publiées depuis leurs dépôts, `libs/` disparaît du dépôt et les couches HTTP, stockage, multi-titre et web sont séparées par des garde-rails à plafond 0.

### Ajouté
- **Mainmise (`film.lead_share`)** — dérivé par match persisté : part de la durée jouée passée seul en tête du score et nombre de retournements. Migration `shared_match_lead_share_v1` (table `match_lead_share` en insertion seule, vue `match_lead_share_latest`), `persist.LeadSharePersister`, calcul pur `internal/analysis/leadshare` avec un jeu de cas partagé par les tests Go et web, commande `levelup backfill-lead-share` (`--dry-run` en lecture seule, sans migration). Champs servis sur la vue du match, Sessions, Séries temporelles et Escouade ; absents, jamais à zéro, quand le match n'est pas mesuré (mode à peu de points, score non couvert, coup d'envoi non mesuré).
- **Page de rejeu** — fil filtrable (Tout, Éliminations, Objectifs, Armes et bonus, Véhicules, Médailles, Assistances ; filtre mémorisé dans `replayPreferences`), clic sur une élimination (pause et mise en valeur du tueur et de la victime), armes ramassées au sol dans le fil, bouton « Vue » et quatre préréglages de calques (`viewPresetLogic`), fiche de joueur en panneau avec Suivre, Seul et Masquer, zone du tueur dans le fil (portage de `NommerZone`, sans champ nouveau ni recuisson).
- **Graphiques de la vue du match** — « Écart de score » (`_scoreDelta.ts`) et « Supériorité numérique » (`lib/replay/aliveBalanceLogic.ts`) dans l'onglet Chronologie, option ECharts commune `_divergentStepOption.ts`.
- **Rejeux compressés** — les artefacts se rangent en `{short8}.json.gz` (gzip niveau 6) derrière une porte unique, `internal/replayartifact` (forme détectée par les octets magiques, parc mixte lu, le compressé gagne) ; `levelup replay compress [--dry-run]` (conversion sur place, atomique, relecture identique avant suppression, jamais automatique) ; sur le bureau, la file de régénération compresse à sa fin les bruts qu'elle n'a pas pu refaire et l'offre de mise à jour propose la compression seule (`raw`). Repli sans lien dur pour exFAT et FAT32. Contenu décompressé inchangé.
- **Seuils du rejeu en millisecondes** — tout seuil exprimé en images devient une durée, convertie avec le pas du document (`frameIntervalMs` côté web, `framesOfMS` côté décodeur, `HistoricalFrameIntervalMS` = 100) ; à 100 ms, aucune valeur ne change.
- **Lecture du film (décodeur)** — positions lues aux largeurs de la région que le flux désigne (table par index de région dans `map_quant_bounds.json`, `profile.MapQuantEntry.Regions`), type d'entité lié à son identifiant complet, capacité du Spartan (étiquette 3). Décodeur v1.3.4 à la fin de la release : `SchemaVersion` 113, révisions `grammar-2026-11-12.4`, `killsource-2026-11-10`, `profile-2026-11-12.2`, `UsageSummaryRev` `us7` inchangée.
- **Destructions de véhicule** — le document de rejeu porte `vehicles[].destruction` (destructeur humain par son xuid ou bot par son entrée de roster, joueurs à bord, occupants tués avec le véhicule lus dans le film : dead-state du bipède écrit dans le paquet de la destruction ou après, et lecture de parenté qui le rattache au véhicule ; éjecté puis tué, tué juste avant, et morts de nettoyage de fin de manche non comptés) et le bloc de couverture `coverage.vehicles.destructions` (`killedNoIdentity`, `killedCleanup`) ; la page lit ces champs (`vehicleDestructionsOf`, ligne « détruit » du fil et de son filtre « Véhicules », parenthèse « à bord » au pluriel de la langue). `destruction.causeKey` est au contrat mais reste vide : son remplissage à la requête est un lot à part. Sur 266 films à véhicules (1 687 occupants nommés) : 1 554 tués lus, accord de 1 529 avec l'ancienne proximité (1 589).
- **Fonds de carte tenus par le maillage de navigation** (ADR 0048, amendement de l'ADR 0043) — les cartes natives sont cuites avec leur maillage (lu dans le module `any` par `halo-infinite-files-reader` et `halo-infinite-map-images` v1.1.0) : le fond riche reste la base, le maillage sert de référence de sol et de masque de rognage (1,5 m par défaut, réglages par carte dans `map_fond_reglages.json`, leviers `rogneAuNavmesh` et `sansNavmesh`) ; 20 fonds natifs recuits, Live Fire sans maillage (dégradation nommée). Plan des murs (`MapBackground.plan`, arêtes de bord en mètres monde) dans le sidecar pour 111 fonds sur 114, natifs et Forge (`mapfond-build -plan-seul` l'ajoute sans recuire) ; paquet `internal/mapfondsidecar`, seul lecteur et écrivain du sidecar côté LevelUp.
- **Web : plan des murs et orientation** — calque statique « Plan des murs » (interrupteur `replay-show-wall-plan`, défaut actif) et orientation par quarts de tour dans le sens horaire (`replay-orientation.<map_id>`, par carte) dans la projection unique du rejeu ; la Tactique et « Occupation du terrain » lisent les mêmes préférences (`useMapViewPreferences`).
- **Rejeu : tourelle fixe, bots, lieu des morts** — châssis de tourelle fixe reconnu, pistes parasites écartées, bots nommés sur un film sans fil d'éliminations (roster lu dans `BOT_METADATA`), lieu des morts lu par corps (`IsolationDecoderRev` montée).
- **Bureau** — commande IPC `pick_folder` (bouton « Parcourir », une seule permission, limitée à `127.0.0.1`) et capacité `can_pick_folder` vraie pour le seul profil `desktop` ; ADR 0045 amendée. Les anciens fichiers de l'interface sont retirés à la mise à jour (`hooks.nsh`) et à la désinstallation (`uninstall_cleanup_engine.go`).
- **Web** — pied de barre latérale compact et menu « Plus », volet des référentiels glissant avec mode épinglé, infobulles ouvertes au focus clavier seulement (`lib/dom/focusVisible.ts`), jeton `--scrim` à la place des noirs et blancs absolus (garde-rail étendu à `bg-black` et `text-white`).
- **Icônes du jeu pour les grenades et les capacités** — vignettes extraites des atlas d'interface du jeu, aux mêmes chemins ; le paquet de ressources change (un téléchargement unique pour l'application de bureau) ; le dossier inutilisé `static/abilities-assets` est supprimé.

### Modifié
- **Briques publiées** — LevelUp consomme `halo-infinite-files-reader`, `halo-infinite-map-images`, `halo-api` et `halo-infinite-film-decoder` depuis leurs dépôts dédiés (en fin de release : `halo-api` v1.0.1, `halo-infinite-files-reader` v1.1.0, `halo-infinite-map-images` v1.1.0, `halo-infinite-film-decoder` v1.3.4 ; la #207 monterait ce dernier à v1.3.6) ; `libs/`, `go.work` et `go.work.sum` sont supprimés, les chemins de module renommés (`github.com/JGtm/...`). `halo-api` porte `retract [v0.1.0, v1.0.0]` (la capture de classement de son `v1.0.0` n'était pas pseudonymisée). Résolveur unique `internal/briques` (`go list -m -f {{.Dir}}`), script `scripts/briques-workspace.sh` pour développer une brique dans un clone (`levelup-briques.work`, jamais un `go.work`), `replay-equiv -update` accepté sur un clone hors du dépôt et refusé sur le cache de modules. Docker, CI, Makefile, lefthook, Dependabot et baselines réécrits ; les tests des briques se jouent par chemin de module depuis `apps/go-api`, leur lint vit dans leur dépôt. Ratchets de taille de fichier et de longueur de fonction ramenés à l'arbre de LevelUp.
- **Interrupteurs d'environnement retirés** — `LEVELUP_USE_SHARED_PROVIDER`, `LEVELUP_POSTSYNC_BURST`, `LEVELUP_SYNC_V2_DRYRUN`, `LEVELUP_PERSIST_NO_FETCH_CACHE`, `LEVELUP_LUSR_V2_MODE_COUPLING`, `LEVELUP_LUSR_V2_SQUAD_OFFSET`, `LEVELUP_WATCHER_BROADCAST`, `PRESTIGE_ENABLED` et `LEVELUP_LOG_JSON` ; l'image pose `LEVELUP_LOG_FORMAT=json`. Six variables sont lues une seule fois par `internal/config` ; `os.Getenv` et `os.LookupEnv` sont interdits hors `internal/config` et `cmd` (garde-rail `archlint`). Deux branches de compatibilité restent, datées du 2026-12-31. La réparation automatique des trous de note est supprimée.
- **FDA nette** — Solo contre Escouade lit la FDA native du match (repli sur le calcul canonique), l'alerte de série de défaites se juge sur une chute en points (`TiltFDADropPoints` 2,5, sévérité haute 4,5), la formule de l'Aide et `Assistances/2` des deux `fields.toml` sont corrigées.
- **LUSR** — repli du solveur sans les comptes de frags et de morts quand il ne converge pas (`LUSRRev` = `lusr-2026-11-09`) : rejeu complet au premier démarrage (ADR 0047), les matchs non convergents reçoivent une note.
- **Alertes de comportement localisées** — onze phrases passent au catalogue `internal/i18n/messages/patterns.toml`.
- **Écritures** — awards et marqueur `psa_checked_at` écrits après l'alias, intensités de match non écrites journalisées et comptées, lignes illisibles journalisées, surface de `duckdb.OpenReadWrite` figée par une liste datée de 43 fichiers, listes d'exception datées par git.
- **Architecture, étapes 4 à 8 (plan d'architecture, closes)** — HTTP : les handlers tiennent des interfaces de port et n'importent plus le paquet `service` ni de moteur de synchronisation, `api/wire` ne porte plus ni SQL ni `database/sql`, les fichiers statiques et la page web vivent dans `api/handlers`, le classement d'une panne de lecture passe par `platform/dbfault`, la purge des sessions est une tâche de fond déclarée dans les quatre profils. Stockage : tout SQL et tout `database/sql` quittent `service/`, `analysis/`, `progression/`, `scheduler/`, `prestige/` et `port/` pour des dépôts derrière des ports (`importstore`, `cronstore`, `profilerepo`, `worldrepo`, `sqlfrag`, `playerresolve`) ; l'erreur « introuvable » est une erreur de domaine ; quatre garde-rails à plafond 0. Multi-titre : `service/` n'importe plus aucun paquet de titre (16 imports ramenés à 0), un titre sans rejeu reçoit `ErrCapabilityNotSupported`. Web : frontières de features contrôlées sans exception de fichier, `echarts` importé d'un seul dossier, appels réseau par le client commun (`api.getText`, `api.getStatic`), onze tableaux triés à la main passés sur TanStack Table (`useColumnSort`), statistiques de graphes dans `lib/charts/stats`, lecture unique du bootstrap, la page du Rejeu 2D devient une page de feature, types d'API manuels remplacés par le contrat généré (le contrat de `GET /settings` est complété : 56 champs au lieu de 30). Clôture : témoins identiques avant et après sur 90 routes (dont 12 pour Halo 5). Restes datés, non tenus à zéro : `os.Getenv` hors `internal/config` (13 en liste datée), `fetch` hors client (4), types d'API homonymes d'un schéma du contrat (11).
- **Aperçus de liens** — langue du lien (`?lang=`), puis du robot, puis langue de repli ; le bouton « Copier le lien » porte la langue du partageur.
- **Licences** — origine des données des briques écrite, `coupe.png` retiré, `sbsp.xml` crédité à Infinite Runtime Tag Viewer, capture de classement de `halo-api` pseudonymisée, identités réelles remplacées dans les fixtures de `apps/` (registre `IDENTITES_APPS.tsv`, garde-rail étendu).
- **Cuisson des rejeux gardée à 100 ms** — le défaut de construction du décodeur est passé à 33 ms en v1.3.0 ; `replaybuild.pasDuRejeu` pose le pas que la page lit, donc les rejeux restent cuits à 10 images par seconde dans cette release. `replay-equiv` joue au pas des références du décodeur (33 ms). Le passage à 30 images par seconde est la PR #207, ouverte.
- **[À CONFIRMER, #207 ouverte]** Cuisson à 30 images par seconde (`internal/replaybuild/cadence.go`, pas de 33 ms), le pas entre dans le critère de fraîcheur (« cadence périmée » côté serveur et badge de l'admin, rien n'est recuit seul), décodeur v1.3.6 (`SchemaVersion` 115 : faits du match indépendants du pas de la grille en v1.3.5, passes bit à bit derrière la marche des trames et positions de véhicule et visée des occupants changées en v1.3.6), route `GET /assets/{title_id}/kill-sources` pour les pictogrammes, pictogramme et tués sur la ligne « a détruit ». La PR attend le verdict de cinq lignes « non établi » du gate du corpus et l'acceptation de deux FAUX attribués à la grille.

### Sécurité
- **Dépendances** — `undici` 8.12.4 (via `jsdom`), `vitest` et `@vitest/mocker` 4.1.12 (`npm update`, plages inchangées) et `minimatch` 11 (trois surcharges ciblées : `eslint`, `@eslint/config-array`, `@typescript-eslint/typescript-estree`) ; `npm audit` à 0 vulnérabilité. Ce sont des dépendances de développement : rien n'en part dans l'application. L'alerte `glib` (Rust, 0.18.5, tirée par `gtk` pour les cibles Linux de Tauri) n'est pas compilée sur Windows (ADR 0045) et ne se corrige que par une montée de Tauri vers gtk-rs 0.20 : à fermer comme non applicable ou à suivre avec la prochaine montée de Tauri.

### Supprimé
- **Probabilité de victoire** — capability, champs de cinq schémas OpenAPI et de `SkillSnapshot`, lecteurs de `platform/duckdb`, colonnes de Synergies et de Progression ; le calcul et la colonne stockée ne changent pas.
- **Commandes sans utilisateur** et le code sans appelant qu'elles tenaient : 96 commandes retirées, 17 rétablies après comparaison du code inatteignable (seules appelantes de 63 fonctions), quatre autres retirées avec leur code (classement mondial, essai de lecture Halo 5) ; `docs/COMMANDS.md` documente les commandes de maintenance gardées.
- Alertes de motifs jamais alimentées du coach (`PatternReport`).

### Corrigé
- **Gate de corpus des rejeux** — `replay-build` se recompile sans cgo, le niveau du journal se lit dans `logging`.
- **Décodeur (v1.3.1)** — création des records des builds HI_1_11_0 à HI_1_13_0 (le jeu saute le masque sous une porte à 0 et n'écrit aucune porte pour dix archétypes à masque de création vide : 35 paquets perdus depuis la v1.0.1 sont relus) et charges de la vue A (genres 12, 14, 106) limitées aux builds où elles sont prouvées. Sur HI_1_9_0 et HI_1_10_0 leur vraie disposition reste non établie.
- **Gate du corpus** — deux requalifications bornées (compteurs déplacés vers la marche des trames, bloc neuf des destructions) ; le faux positif du banc de vérité sur le repli de la bulle d'écran occultant d'un film est accepté comme instruit.
- **Comparaison** — la bannière de confidentialité et l'avertissement « données partielles », que le serveur n'émettait jamais, sont retirés de la page.
- **Décodeur** — validation d'une carte à région invalide, bits d'index d'un module local à plus de deux régions, un index de région sans ligne lit zéro bit comme le jeu.
````
