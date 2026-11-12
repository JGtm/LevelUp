# Notes de version 8.2.0, BROUILLON FR (2026-11-12)

Base : `origin/release/v8.2.0` à `20b8f386a6`. Rien n'est posé dans `docs/` : les deux tests
`TestReleaseNotesDocs_*` exigent que le bloc le plus récent des notes porte le numéro de la première entrée
versionnée du changelog, donc le bloc `v8.2` et l'entrée `[8.2.0]` se posent ENSEMBLE à la clôture, avec la
montée de version (`Cargo.toml`, `Cargo.lock`, `tauri.conf.json`). Le pendant anglais est `brouillon_versions_8_2_0_EN.md`.

Les lignes marquées **[À CONFIRMER]** dépendent d'une PR encore ouverte (#203, cadence à 30 images/s) : les
retirer ou les finaliser avant la clôture, aucune ne doit partir telle quelle.

## 1. Bloc pour `docs/FR/RELEASE_NOTES.md`

À coller en tête du fichier, sous `## Dernières nouveautés`, avant le bloc `**v8.1.2 — ...**`. La première
section est reprise telle quelle comme notes de l'updater et de la page de la version.

````markdown
**v8.2 — Un rejeu plus complet, la Mainmise, de nouveaux graphiques de match et des rejeux compressés**

**Le rejeu**
- **Un fil d'événements qui se filtre** — le titre du fil ouvre un menu : Tout, Éliminations, Objectifs, Armes et bonus, Véhicules, Médailles, Assistances. Les actions d'objectif apparaissent en bandeaux, les armes ramassées au sol entrent dans le fil, et un filtre sans aucune ligne dans le match le dit au lieu de rester vide. Ton choix est mémorisé
- **Un clic sur une élimination** — la lecture se met en pause à cet instant, et le tueur et la victime sont mis en valeur sur la carte, avec leur nom et l'arme
- **La zone de l'élimination** — chaque élimination du fil indique la zone de la carte où se tenait le tueur, quand la carte en nomme une
- **Un bouton « Vue »** — dans le coin de la carte, quatre préréglages de calques (Complet, Combat, Déplacements, Objectif) et un panneau qui regroupe les effets, la chaleur, la couleur des points et le son. Le bouton Réglages de la barre de lecture disparaît
- **La fiche d'un joueur dans un panneau** — un clic sur une fiche remplace le fil par ses totaux de fin de match (précision, dégâts, détail du score, objectif, armes), avec Suivre, Seul et Masquer. Échap referme le panneau
- **Le film mieux lu** — les positions sont lues aux bonnes largeurs sur les cartes à plusieurs régions, le type des objets et les capacités du Spartan sont reconnus avec plus de rigueur : un rejeu régénéré gagne en précision
- **Tourelles fixes, bots et lieux de mort** — une tourelle fixe est reconnue comme telle, des pistes parasites ne s'affichent plus, les bots d'un film sans fil d'éliminations gardent leur nom, et des éliminations qui n'avaient aucun lieu en retrouvent un
- **Des icônes du jeu pour les grenades et les capacités** — les vignettes du rejeu viennent des planches d'interface du jeu, au trait pour les grenades
- **[À CONFIRMER, #203] Les véhicules détruits** — le fil et son filtre « Véhicules » listent les destructions de véhicule
- **[À CONFIRMER, cadence] Rejeux à 30 images par seconde** — les mouvements sont plus fluides, position, visée et bouclier compris

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
````

## 2. Bloc pour `docs/FR/CHANGELOG.md`

Remplace `## [Unreleased]` par un `## [Unreleased]` vide, puis pose ceci en dessous, la date étant celle de la
publication. L'entrée « Icônes du jeu » reprend celle déjà présente sous `[Unreleased]`. Le fichier est lu à
l'exécution par la page des modifications : ne rien y laisser de **[À CONFIRMER]** à la fusion.

````markdown
## [8.2.0] - AAAA-MM-JJ

> Le rejeu et la Mainmise côté joueur ; côté code, LevelUp consomme les quatre briques publiées en v1.0.1 et `libs/` disparaît du dépôt.

### Ajouté
- **Mainmise (`film.lead_share`)** — dérivé par match persisté : part de la durée jouée passée seul en tête du score et nombre de retournements. Migration `shared_match_lead_share_v1` (table `match_lead_share` en insertion seule, vue `match_lead_share_latest`), `persist.LeadSharePersister`, calcul pur `internal/analysis/leadshare` avec un jeu de cas partagé par les tests Go et web, commande `levelup backfill-lead-share` (`--dry-run` en lecture seule, sans migration). Champs servis sur la vue du match, Sessions, Séries temporelles et Escouade ; absents, jamais à zéro, quand le match n'est pas mesuré (mode à peu de points, score non couvert, coup d'envoi non mesuré).
- **Page de rejeu** — fil filtrable (Tout, Éliminations, Objectifs, Armes et bonus, Véhicules, Médailles, Assistances ; filtre mémorisé dans `replayPreferences`), clic sur une élimination (pause et mise en valeur du tueur et de la victime), armes ramassées au sol dans le fil, bouton « Vue » et quatre préréglages de calques (`viewPresetLogic`), fiche de joueur en panneau avec Suivre, Seul et Masquer, zone du tueur dans le fil (portage de `NommerZone`, sans champ nouveau ni recuisson).
- **Graphiques de la vue du match** — « Écart de score » (`_scoreDelta.ts`) et « Supériorité numérique » (`lib/replay/aliveBalanceLogic.ts`) dans l'onglet Chronologie, option ECharts commune `_divergentStepOption.ts`.
- **Rejeux compressés** — les artefacts se rangent en `{short8}.json.gz` (gzip niveau 6) derrière une porte unique, `internal/replayartifact` (forme détectée par les octets magiques, parc mixte lu, le compressé gagne) ; `levelup replay compress [--dry-run]` (conversion sur place, atomique, relecture identique avant suppression, jamais automatique) ; sur le bureau, la file de régénération compresse à sa fin les bruts qu'elle n'a pas pu refaire et l'offre de mise à jour propose la compression seule (`raw`). Repli sans lien dur pour exFAT et FAT32. Contenu décompressé inchangé.
- **Seuils du rejeu en millisecondes** — tout seuil exprimé en images devient une durée, convertie avec le pas du document (`frameIntervalMs` côté web, `framesOfMS` côté décodeur, `HistoricalFrameIntervalMS` = 100) ; à 100 ms, aucune valeur ne change.
- **Lecture du film (décodeur)** — positions lues aux largeurs de la région que le flux désigne (table par index de région dans `map_quant_bounds.json`, `profile.MapQuantEntry.Regions`), type d'entité lié à son identifiant complet, capacité du Spartan (étiquette 3). Révisions : `grammar-2026-11-10.3`, `killsource-2026-11-10`, `profile-2026-11-10`, `SchemaVersion` 108.
- **Rejeu : tourelle fixe, bots, lieu des morts** — châssis de tourelle fixe reconnu, pistes parasites écartées, bots nommés sur un film sans fil d'éliminations (roster lu dans `BOT_METADATA`), lieu des morts lu par corps (`IsolationDecoderRev` montée).
- **Bureau** — commande IPC `pick_folder` (bouton « Parcourir », une seule permission, limitée à `127.0.0.1`) et capacité `can_pick_folder` vraie pour le seul profil `desktop` ; ADR 0045 amendée. Les anciens fichiers de l'interface sont retirés à la mise à jour (`hooks.nsh`) et à la désinstallation (`uninstall_cleanup_engine.go`).
- **Web** — pied de barre latérale compact et menu « Plus », volet des référentiels glissant avec mode épinglé, infobulles ouvertes au focus clavier seulement (`lib/dom/focusVisible.ts`), jeton `--scrim` à la place des noirs et blancs absolus (garde-rail étendu à `bg-black` et `text-white`).
- **Icônes du jeu pour les grenades et les capacités** — vignettes extraites des atlas d'interface du jeu, aux mêmes chemins ; le paquet de ressources change (un téléchargement unique pour l'application de bureau) ; le dossier inutilisé `static/abilities-assets` est supprimé.

### Modifié
- **Briques publiées** — LevelUp consomme `halo-infinite-files-reader`, `halo-infinite-map-images`, `halo-api` et `halo-infinite-film-decoder` en v1.0.1 depuis leurs dépôts dédiés ; `libs/`, `go.work` et `go.work.sum` sont supprimés, les chemins de module renommés (`github.com/JGtm/...`). `halo-api` porte `retract [v0.1.0, v1.0.0]` (la capture de classement de son `v1.0.0` n'était pas pseudonymisée). Résolveur unique `internal/briques` (`go list -m -f {{.Dir}}`), script `scripts/briques-workspace.sh` pour développer une brique dans un clone (`levelup-briques.work`, jamais un `go.work`), `replay-equiv -update` accepté sur un clone hors du dépôt et refusé sur le cache de modules. Docker, CI, Makefile, lefthook, Dependabot et baselines réécrits ; les tests des briques se jouent par chemin de module depuis `apps/go-api`, leur lint vit dans leur dépôt. Ratchets de taille de fichier et de longueur de fonction ramenés à l'arbre de LevelUp.
- **Interrupteurs d'environnement retirés** — `LEVELUP_USE_SHARED_PROVIDER`, `LEVELUP_POSTSYNC_BURST`, `LEVELUP_SYNC_V2_DRYRUN`, `LEVELUP_PERSIST_NO_FETCH_CACHE`, `LEVELUP_LUSR_V2_MODE_COUPLING`, `LEVELUP_LUSR_V2_SQUAD_OFFSET`, `LEVELUP_WATCHER_BROADCAST`, `PRESTIGE_ENABLED` et `LEVELUP_LOG_JSON` ; l'image pose `LEVELUP_LOG_FORMAT=json`. Six variables sont lues une seule fois par `internal/config` ; `os.Getenv` et `os.LookupEnv` sont interdits hors `internal/config` et `cmd` (garde-rail `archlint`). Deux branches de compatibilité restent, datées du 2026-12-31. La réparation automatique des trous de note est supprimée.
- **FDA nette** — Solo contre Escouade lit la FDA native du match (repli sur le calcul canonique), l'alerte de série de défaites se juge sur une chute en points (`TiltFDADropPoints` 2,5, sévérité haute 4,5), la formule de l'Aide et `Assistances/2` des deux `fields.toml` sont corrigées.
- **LUSR** — repli du solveur sans les comptes de frags et de morts quand il ne converge pas (`LUSRRev` = `lusr-2026-11-09`) : rejeu complet au premier démarrage (ADR 0047), les matchs non convergents reçoivent une note.
- **Alertes de comportement localisées** — onze phrases passent au catalogue `internal/i18n/messages/patterns.toml`.
- **Écritures** — awards et marqueur `psa_checked_at` écrits après l'alias, intensités de match non écrites journalisées et comptées, lignes illisibles journalisées, surface de `duckdb.OpenReadWrite` figée par une liste datée de 43 fichiers, listes d'exception datées par git.
- **Architecture, étape 4 (4.1 à 4.3)** — `worldenrich` n'importe plus `service`, les jobs de synchronisation passent par un port et un service, progression, campagne et profil passent par des ports, la limite mémoire des bases passe par un port ; garde-rail sur le stockage dans les handlers. **[À CONFIRMER]** : 4.4 à 4.8 (SQL de `api/wire` vers `platform/duckdb`, handlers déplacés, interfaces de `port/`, garde-rails d'import) restent ouverts.
- **Aperçus de liens** — langue du lien (`?lang=`), puis du robot, puis langue de repli ; le bouton « Copier le lien » porte la langue du partageur.
- **Licences** — origine des données des briques écrite, `coupe.png` retiré, `sbsp.xml` crédité à Infinite Runtime Tag Viewer, capture de classement de `halo-api` pseudonymisée, identités réelles remplacées dans les fixtures de `apps/` (registre `IDENTITES_APPS.tsv`, garde-rail étendu).
- **[À CONFIRMER, #203]** Décodeur v1.2.3 : destructions de véhicule (`vehicles[].destruction`), lecture de la vue A par la table des genres, `SchemaVersion` 109.
- **[À CONFIRMER, cadence]** Rejeux à 30 images par seconde, `SchemaVersion` 110, révision des résumés d'usage `us8`.

### Supprimé
- **Probabilité de victoire** — capability, champs de cinq schémas OpenAPI et de `SkillSnapshot`, lecteurs de `platform/duckdb`, colonnes de Synergies et de Progression ; le calcul et la colonne stockée ne changent pas.
- **Commandes sans utilisateur** et le code sans appelant qu'elles tenaient : 96 commandes retirées, 17 rétablies après comparaison du code inatteignable (seules appelantes de 63 fonctions), quatre autres retirées avec leur code (classement mondial, essai de lecture Halo 5) ; `docs/COMMANDS.md` documente les commandes de maintenance gardées.
- Alertes de motifs jamais alimentées du coach (`PatternReport`).

### Corrigé
- **Gate de corpus des rejeux** — `replay-build` se recompile sans cgo, le niveau du journal se lit dans `logging`.
- **Décodeur** — validation d'une carte à région invalide, bits d'index d'un module local à plus de deux régions, un index de région sans ligne lit zéro bit comme le jeu.
````
