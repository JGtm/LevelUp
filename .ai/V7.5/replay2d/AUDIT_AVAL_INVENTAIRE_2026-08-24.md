# Audit aval — pertes d'inventaire entre décodeur et fiche joueur (2026-08-24)

> Périmètre : la chaîne AVAL du décodage — tout ce qui peut jeter une lecture d'inventaire
> DÉJÀ décodée avant qu'elle n'atteigne l'écran. Audit sur pièces, aucune correction
> apportée. Chemin suivi : `ScanFilmKeyframeInventory` (Go) → `buildInventory` →
> `keepInventoryOfPublishedTracks` → `ReplayDocument.Inventory` → transport JSON →
> `normalizeReplayDocument` → `rosterLogic.inventoryAt` → `ReplayInventoryRow` (web).

## Sommaire des points, du plus grave au moins grave

| # | Gravité | Fichier:ligne | Résumé |
|---|---|---|---|
| 1 | Fiche partielle silencieuse, potentiellement massive | `apps/web/src/features/match-replay/replayNormalize.ts` (tout le fichier, absence totale de garde) + `apps/go-api/internal/replaybuild/replaybuild.go:307` | Le client ne lit JAMAIS `schemaVersion` : un artefact non re-cuit depuis un bump de schéma (ex. v6, capacité d'armure) sert une fiche amputée SANS AUCUN signal visuel — indiscernable d'un « rien à afficher ». |
| 2 | Fiche partielle par vie | `apps/go-api/internal/analysis/replay/inventory.go:92-100` (`buildInventory`) | Toute lecture d'inventaire dont l'horodatage précède l'origine du rejeu (`sorted[0].TimestampUS`, calculée sur TOUTES les positions, y compris celles hors carte) est jetée sans compteur ni log — invisible dans `Coverage`. |
| 3 | Fiche partielle, dégradation non observable | `apps/go-api/internal/analysis/replay/inventory_decode.go:195-200` (`ScanFilmKeyframeInventory`) | Les erreurs de lecture PAR CHUNK sont avalées par un `continue` nu, sans compteur ni log — contrairement aux scanners frères (capacité i48, camouflage, grappin) qui remontent tous une `Stats` loguée en détail dans `build.go`. Un film partiellement corrompu peut perdre la majorité de son inventaire tout en étant traité comme « lisible ». |
| 4 | Fiche vide totale (bloc entier), improbable en pratique | `apps/go-api/internal/analysis/replay/build.go:210-214` | Si `ScanFilmKeyframeInventory` erre, TOUT l'inventaire du match est abandonné d'un bloc (`slog.Warn` puis `inventory = nil`). Le seul chemin d'erreur observé dans le décodeur est `read == 0` (aucun chunk lisible), ce qui aurait déjà fait échouer `ScanFilmBipedPositions` en amont (retour d'erreur non avalé, ligne 189-192) — donc ce cas précis semble inatteignable en pratique SAUF asymétrie de lecture entre les deux scans (comptage de chunks différent, cache partiellement purgé entre les deux appels). À vérifier côté décodeur (hors périmètre de cet audit). |
| 5 | Fiche vide totale par joueur (voulu, mais sans garde-rail visible) | `apps/go-api/internal/analysis/replay/published_tracks.go:39-54` (`keepOfPublishedTracks`) via `inventory.go:154-157` | Un slot dont la track n'atteint pas `DefaultMinPoints=2` positions publiées (`build.go:20`, `build.go:516-582`) perd la totalité de son inventaire ET ses tirs/lancers/armes — c'est la règle documentée et partagée par 4 calques, donc cohérente. Un joueur qui rejoint en fin de match / meurt quasi instantanément (biped visible sur 0-1 frame décimée) tombe dans ce cas. Aucun compteur ne publie combien de slots sont ainsi écartés spécifiquement pour l'inventaire (la couverture existante — `doc.Coverage` — ne porte pas d'entrée `Inventory`, contrairement à `Shots`, `Grenades`, `Equipment`). |
| 6 | Fiche vide par vie (comportement voulu et documenté) | `apps/web/src/features/match-replay/rosterLogic.ts:438-445` (`inventoryAt`) + `ReplayInventoryRow.tsx:75` (`if (!read && !ability) return null`) | Une vie qui ne recoupe AUCUNE image-clé (ni passée ni future dans sa fenêtre — vie plus courte que l'intervalle inter-keyframe ~20s, en fin de match sans keyframe suivante) rend `inventoryAt` → `null`, et la ligne entière disparaît sans état vide explicite (juste l'espace réservé `min-h-4` / `min-h-[18px]` resté blanc). C'est un CHOIX assumé par le code (âge négatif toléré, mais pas d'invention hors fenêtre de vie) — noté ici parce que rien à l'écran ne dit « inventaire non lu pour cette vie » vs « pas de donnée disponible sur ce match ». |

## Détail des points

### 1. Absence de garde SchemaVersion côté client (le plus grave)

- **Fichier** : `apps/web/src/features/match-replay/replayNormalize.ts` (fonction
  `normalizeReplayDocument`, lignes 153-233) ; `apps/go-api/internal/analysis/replay/document.go:149`
  (`SchemaVersion = 18`) ; `apps/go-api/internal/replaybuild/replaybuild.go:293-307`
  (`ArtifactUpToDate` : `head.SchemaVersion == replay.SchemaVersion`).
- **Condition de déclenchement** : un match dont l'artefact JSON sur disque a été construit
  AVANT un bump de schéma (ex. avant le 2026-08-14, `SchemaVersion` < 6) et n'a pas encore
  été repassé par `levelup backfill-replay` (commande MANUELLE d'opérateur, cf.
  `cmd_backfill_replay.go` — « une montée de schéma périme d'un coup TOUS les artefacts »,
  ligne 54). L'API sert cet artefact tel quel (aucune vérification de fraîcheur côté
  service HTTP trouvée dans ce périmètre).
- **Effet** : `raw.abilities` est `undefined` → normalisé en `[]` (ligne 160) ; la capacité
  d'armure ne s'affiche JAMAIS pour ce match, silencieusement, et rien ne distingue ce cas
  de « aucune capacité lue sur ce match ». Le même mécanisme vaut pour TOUT champ ajouté
  par une montée de version ultérieure (grappin v8, poses v9, zones v16, jauge v18…) : un
  vieux match reste éternellement dégradé tant qu'un opérateur ne relance pas la commande
  de re-cuisson, et l'utilisateur final n'a aucun moyen de le savoir depuis l'écran.
- **Gravité** : fiche partielle silencieuse — potentiellement sur TOUS les matchs plus
  anciens que le dernier bump de schéma tant que le backfill de masse n'a pas tourné à leur
  sujet. C'est un cas plausible d'explication aux « fiches parfois vides » signalées par
  l'utilisateur : pas un bug de décodage, un simple défaut de re-cuisson non signalé.

### 2. Inventaire antérieur à l'origine — écarté sans mesure

- **Fichier** : `apps/go-api/internal/analysis/replay/inventory.go:88-100`.
- **Condition** : `r.TimestampUS < origin`, où `origin` (`build.go:347`) est
  `sorted[0].TimestampUS` — le plus ancien horodatage de TOUTE position bipède du film,
  y compris les positions sans coordonnée monde (`HasWorld=false` n'est filtré QUE dans
  `decimateTracks`, pas dans le calcul de `origin`, cf. `build.go:344-349` vs `build.go:523-526`).
  Si la toute PREMIÈRE image-clé du film (la plus riche : chargeurs au plein, grenades de
  spawn, capacités par défaut — cf. le commentaire du témoin terrain à `inventory_decode.go:53-57`)
  porte un horodatage strictement antérieur au premier point de position retenu comme
  origine, elle est purement et simplement perdue.
- **Ce que le code affirme** (commentaire ligne 90-91) : « il n'a pas de place sur l'axe » —
  posture cohérente (ne pas inventer une frame négative), mais AUCUN compteur ne publie
  combien de lectures sont ainsi écartées ; ce n'est ni dans `Coverage`, ni dans un log.
  Impossible de savoir, à la lecture du document, si cette perte est arrivée sur un match
  donné et représente 0 ou 8 lectures.
- **Gravité** : fiche partielle — touche potentiellement la lecture LA PLUS RICHE du match
  (l'inventaire de spawn, avant tout dégainage), c'est-à-dire précisément celle qui sert de
  référence aux 20 premières secondes de chaque vie (via le repli « à venir » de
  `nearestReading`). Mérite une mesure chiffrée (nombre de matchs / nombre de lectures
  perdues) avant de juger si c'est anecdotique ou systématique.

### 3. Erreurs de chunk avalées dans le décodeur d'inventaire, sans télémétrie

- **Fichier** : `apps/go-api/internal/analysis/replay/inventory_decode.go:195-200`
  (boucle `for c := 1; c <= n; c++ { chunk, err := filmdec.ReadFilmChunk(dir, c); if err != nil { continue } ... }`).
- **Comparaison avec les scanners frères** : `ScanFilmAbilityRanks`, `ScanFilmCamoStates`,
  `ScanFilmGrappleReads` (appelés juste après dans `build.go:219-254`) rendent chacun une
  structure `Stats` (records vus, masque présent, lus, illisibles, sans canal…) que
  `build.go` logue en détail (`slog.Info`, lignes 224-227, 236-239, 249-253).
  `ScanFilmKeyframeInventory` ne rend AUCUNE statistique — seulement la liste des
  inventaires décodés, ou une erreur globale si `read == 0`.
- **Condition de déclenchement** : tout chunk illisible AUTRE que la totalité du film
  (cache partiellement corrompu, film tronqué en cours de téléchargement, i/o
  intermittente) est silencieusement ignoré, chunk par chunk, sans qu'aucun compteur ne le
  révèle. Le film peut sortir avec une fraction seulement de ses images-clés d'inventaire
  décodées, et rien ne le distingue d'un film sain avec moins de keyframes.
- **Gravité** : fiche partielle non observable — pas un bug qui casse l'affichage, mais un
  angle mort de diagnostic : si un opérateur constate une fiche clairsemée sur un match
  donné, ce point de code ne lui donnera aucun indice (pas de log, pas de compteur) pour
  savoir si la cause est « peu de keyframes dans le film » ou « chunks corrompus ».

### 4. Abandon en bloc si `ScanFilmKeyframeInventory` erre (probablement inatteignable)

- **Fichier** : `apps/go-api/internal/analysis/replay/build.go:210-214`.
- **Condition** : `ScanFilmKeyframeInventory` ne retourne une erreur QUE si `read == 0`
  (aucun chunk lisible du tout, `inventory_decode.go:211-213`). Or `ScanFilmBipedPositions`
  (`build.go:189-192`) est appelée AVANT sur le MÊME `filmDir` et retourne une erreur NON
  avalée (`return ReplayDocument{}, err`) en cas d'échec total de lecture — ce qui
  empêcherait d'atteindre la ligne 210 dans le même scénario. Le seul moyen d'atteindre ce
  `slog.Warn` semble être une ASYMÉTRIE entre les deux scans : `CountFilmChunks(dir)`
  (appelé indépendamment dans chaque scanner) pourrait renvoyer un compte différent d'un
  appel à l'autre si le cache disque change entre les deux (purge concurrente, écriture en
  cours) — hors périmètre de cet audit (relève du décodage, pas de l'aval), mais à signaler
  aux agents qui instruisent le décodeur.
- **Gravité** : fiche vide totale POUR L'INVENTAIRE SEUL (les autres calques restent
  intacts) — mais chemin probablement jamais emprunté en pratique.

### 5. `keepInventoryOfPublishedTracks` — aucune télémétrie de couverture dédiée

- **Fichier** : `apps/go-api/internal/analysis/replay/published_tracks.go:39-54` et
  `inventory.go:152-157`.
- **Condition** : un slot dont la track ne compte pas au moins `DefaultMinPoints=2` points
  décimés publiés (typiquement une vie extrêmement courte — spawn puis mort quasi
  immédiate, ou un joueur qui rejoint après le dernier paquet exploitable) est absent de
  `publishedSlots`, et TOUT son inventaire décodé est jeté silencieusement par ce filtre
  générique (partagé par tirs, lancers, armes, inventaire — donc cohérent entre calques,
  cf. le commentaire de conception du fichier).
- **Ce qui manque** : contrairement à `Shots` (`shotCov.Unpublished`,
  `build.go:375-378`), `Grenades` (`grenCov.Unpublished`, `build.go:388-391`) ou
  `Equipment` (`equipmentCoverage`, `build.go:412`), **l'inventaire n'a AUCUNE entrée dans
  `doc.Coverage`**. Impossible de savoir, à la lecture du document ou de ses logs, combien
  d'inventaires décodés ont été jetés par ce filtre pour un match donné — alors que les
  trois autres calques qui partagent exactement le même filtre publient ce nombre.
- **Gravité** : fiche vide pour les joueurs à vie(s) trop courte(s) — comportement voulu
  (le client n'aurait nulle part où poser cet inventaire), mais son AMPLEUR n'est
  observable nulle part, ce qui complique le diagnostic « pourquoi telle fiche est vide »
  rapporté par l'utilisateur (une vie courte de fin de partie, un rejoin tardif, sont deux
  causes possibles et se confondent aujourd'hui).

### 6. Ligne d'inventaire invisible sans lecture dans la fenêtre de vie (voulu, mais muet)

- **Fichier** : `apps/web/src/features/match-replay/rosterLogic.ts:373-391`
  (`nearestReading`, le foyer canonique partagé par loadout/inventaire/capacité) et
  `apps/web/src/features/match-replay/ReplayInventoryRow.tsx:75`
  (`if (!read && !ability) return null`).
- **Condition** : `nearestReading` cherche par SLOT (jamais par joueur — correct, une
  dotation ne franchit pas une mort) la lecture la plus proche, PASSÉE ou FUTURE, sans
  AUCUN seuil d'âge (pas de coupure au-delà d'un certain écart — contrairement à
  `heldReading`, qui lui a un paramètre `maxAge`, cf. `replayLogic.ts:266-282`, utilisé
  pour le bouclier/santé). Concrètement : tant qu'il existe UNE lecture pour ce slot,
  aussi vieille ou aussi lointaine soit-elle, la ligne l'affiche (pâlie par `freshness`).
  Le VRAI cas vide est : aucune lecture n'existe DU TOUT pour ce slot précis (vie plus
  courte que l'intervalle inter-keyframe et qui ne recoupe la fenêtre d'aucune image-clé,
  ni avant ni après). Dans ce cas la fonction rend `null`, la ligne entière disparaît
  (`return null` dans le composant), et l'espace réservé (`min-h-4` / `min-h-[18px]`,
  `ReplayTeams.tsx:339,354-359,384-397`) reste vide SANS aucun texte ni pictogramme
  d'état — à la différence des autres lacunes du même composant qui portent toutes un
  pictogramme dédié (`AmmoFullMark`, `HolsterMark`, `AbilityUnknownMark`).
- **Gravité** : fiche partielle par vie, comportement documenté et assumé — mais l'absence
  totale de placeholder visuel rend ce cas indiscernable, à l'œil, d'un bug d'affichage.
  Un utilisateur qui voit une zone d'inventaire vide ne peut pas savoir si c'est « rien n'a
  été lu pour cette vie » (état légitime) ou une régression.

## Ce qui a été vérifié et écarté (pas un point de perte)

- **Mismatch slot / track (numérotation vs ordre d'apparition)** : aucune trouvaille. Le
  client indexe TOUJOURS par le champ `slot` porté par chaque enregistrement (jamais par
  position dans le tableau `tracks`), aussi bien côté `rosterLogic.ts` que
  `equippedLogic.ts` et `ReplayInventoryRow.tsx`. `keepOfPublishedTracks`
  (`published_tracks.go`) construit son ensemble de slots publiés de la même façon. Pas de
  confusion trouvée entre un rang d'affichage et l'identité de slot.
- **Seuil d'âge au-delà duquel la fiche masquerait une lecture ancienne** : il n'existe
  PAS pour l'inventaire/loadout/capacité (`nearestReading` n'a pas de `maxAge`) —
  uniquement pour le bouclier/santé (`heldReading`, `maxAge = Number.POSITIVE_INFINITY`
  passé explicitement dans `ReplayInventoryRow`/`playerStateAt`, donc en pratique
  également sans coupure). Ce n'est donc pas une cause de fiche vide pour l'inventaire.
- **État vide explicite pour "Inventory vide au niveau du document"** : le document Go
  distingue déjà « champ absent » vs « tableau vide » vs `null` à travers
  `normalizeReplayDocument`, sans confusion relevée.

## Recommandations (hors périmètre de correction, à noter pour un futur chantier)

1. Publier `doc.Coverage.Inventory` (comptes construits / publiés / écartés par
   `keepInventoryOfPublishedTracks` et par le filtre d'origine), symétriquement à
   `Shots`/`Grenades`/`Equipment`.
2. Faire remonter des `Stats` depuis `ScanFilmKeyframeInventory` (chunks lus/illisibles),
   loguées comme les scanners frères.
3. Envisager un signal visible côté client quand `doc.schemaVersion < <version attendue>`
   (bandeau « rejeu à re-cuire », ou au minimum un log console) plutôt qu'une dégradation
   par silence.
