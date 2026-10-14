# Plan — ajustements du 2026-10-14 (liste du user `Ajustements.txt`)

Origine : liste du user du 2026-10-14 (26 points : Tactique, Vue match, Séries temporelles,
Escouade, Général). Diagnostic en lecture seule le 2026-10-14 (six enquêteurs, copies des bases du
14/10 dans le scratchpad de session, aucune écriture). Contrat d'exécution : skill
`plan-execution`. Superviseur : session principale. Exécutants : Sonnet par défaut, Opus pour les
lots film (rétro-ingénierie) et pour la relecture du lot CSR (sync/persist).

**Statut : PROPOSÉ — aucune exécution avant le go du user et les décisions D-1 à D-9.**

## 0. Constats sur pièces (résumé ; le détail est dans les rapports d'enquête du 14/10)

| # | Point | Cause prouvée | Recuisson |
|---|---|---|---|
| G1 | Translocateur invisible et muet | **Historique relu le 14/10 (réponse au user)** : rien n'a été supprimé, mais deux pièces sont mortes sans bruit. (a) SON DE POSE : le son extrait par le user (« la dépose de la faille », `translocator_deploy.wav`, commit `34d26cfb8` du 27/08) est branché sur une pose `deployed` de la balise ; au 03/09 le parc en comptait déjà 0 sur 16 (journal l.104872), et depuis la règle du 15/09 (`b7f33b54c` : « déployé = ce qu'un 103 désigne ») la balise n'est jamais `deployed` → son mort depuis début septembre ; le test `replaySound.test.ts:333` fabrique une pose qui n'existe pas et l'a masqué. (b) SON DE TÉLÉPORTATION : jamais branché (commit du 27/08 : « livré sous translocator_teleport mais pas branché » ; le fichier n'a jamais été commité) ; les 23 gestes de la banque de l'appareil (`dcfaa487`) sont rendus dans `Downloads/Jeux/Halo Infinite/Halo Infinite - Sons v75/rendus/`. (c) VISUEL : éclat de fiche + lien du va-et-vient + faille livrés le 03/09 (`a8f9e3993`), TOUJOURS dans le code, données identiques entre l'artefact du 17/09 et celui d'aujourd'hui ; le gate visuel du user prévu ce jour-là (cuisson de 1b2d9e08) n'a pas de trace d'exécution. Par lecture de code : le saut est interpolé (`replayLogic.ts:59-70`), la traînée trace un trait plein (`:89-97`), la faille est peinte sous les trajectoires (ordre inchangé depuis avant le 06/09) — **non vérifié à l'écran**, à constater au navigateur avant toute correction (B0). | non pour le rendu ; oui si la pose est publiée |
| G2 | Répulseur sans son | **Historique relu le 14/10** : ce qui a été trouvé et livré est toujours là — rang de l'équipement nommé (`abilityLabels`), ramassage et CONSOMMATION (`spent`) datés par le film, son du kill au répulseur. Le `spent` est volontairement muet (`equipmentChangeSound.ts`, en-tête) et ne marque que la FIN de l'équipement (WolfzyLIVE : un seul `spent` à 1:16,3 pour deux poussées à 0:55 et 1:04). Le son d'activation trouvé dans le jeu (`7bd0883c`, 11 rendus dans le même dossier) n'a jamais été branché (journal l.25032 : « pas tant que le calque ne le dessine pas »). Les événements de poussée 104/105 (R12) sont lus par la grammaire de vue A puis jetés (`vue_a_charges_armes.go:190-207`). Trajectoires : JGtm poussé par WolfzyLIVE à 0:55,0 et 1:04,6 (arc balistique, éloignement 0,98-1,00, porteur à 1,8-2,5 m). **Contre-preuve du 14/10** : le harnais R12 lit sur ce film trois 104 (1:35,5, 3:47,8, 4:30,1, horloge du harnais) et quatre 105, palette « non classée » ; aucun aux instants des trajectoires (aucun décalage constant ne les aligne) → canal à établir sur ce build avant publication (F1). | oui (révision des faits) |
| G3 | Deux compteurs de manches (Oddball, CTF à manches) | `ReplayScoreBanner.tsx` : pastilles au-dessus (l.106, `RoundDots`), texte `a - b` sous l'horloge (l.138-145) ; « Manche X » en 9 px gris (l.125-132). 16 artefacts concernés (13 Oddball, 3 CTF). | non |
| G4 | Pas de tir violet Forerunner | Chaîne de teinte complète et thème présent ; seul `hinf_sentinel_beam` est déclaré `forerunner` (`replay_labels.toml:352-356`, commentaire faux « seule arme Forerunner ») ; le registre `games/weapons/registry.go` classe aussi Cindershot et Heatwave `forerunner`, teintés plasma au TOML. Teinte posée à la requête : pas de recuisson. | non |
| G5 | CSR absent (Nuzzles, et en fait TOUS) | **Aucun CSR par match écrit depuis le 16/09 14:52 UTC, pour personne** (vérifié : `max(written_at)` de `match_csrs`). `match_registry.is_ranked` faux sur 876 des 877 matchs « Ranked » de 2026 : `isRankedPlaylist` (`sync/transforms_helpers.go:100-117`) cherche « ranked » dans un nom que le payload ne porte pas au sync ; `ExtractCSRRowIfRanked` et `ExtractAllSharedCSRRows` abandonnent sur `!reg.IsRanked`. Seuls les rattrapages manuels (heuristique de nom, `csr_backfill.go:156-168`) ont écrit. Touche tuiles, Explorer, détail de match, Relations, pic de compétence, Carrière, Comparer. CSR courant par playlist : sain. | non ; backfill CSR |
| G6 | Dominance = Score en Slayer | `regulation.toml [score_timeline]` classe déjà Slayer et Team Snipers `hidden` (servi par `MatchViewHeader.score_timeline_kind`, lu par la Vue match) ; le rejeu ne le lit pas et s'appuie sur une comparaison de rendu (`sameLeadSegments`) qui échoue dès que deux frags tombent dans la même image (estimé 60 Slayer sur 150). Le user juge cette garde inadaptée (14/10) : elle est SUPPRIMÉE, la règle déclarée décide seule. | non |
| G7 | ERROR `seasons_catalog` | **Correction du diagnostic (14/10, sur remarque du user)** : l'appel en direct RÉUSSIT (`season_catalog` partagé : 11 saisons, `last_fetched_at` = 13/10 21:10). Les échecs sont des 401 (2 196) et 403 (340) intermittents dans `logs/service.log`. Le 11/10 à 21:29:33 l'échec tombe dans la même seconde qu'un rafraîchissement de jeton avec rotation et un `RpsTicket refusé (401)` : l'appel part avec le jeton de la requête (`enrichWithHaloTokens`), sans relance après rafraîchissement, et un seul échec d'un lecteur coupe l'appel 30 min pour tout le titre (`seasons_catalog.go:312-314`). Hypothèse la plus forte, à confirmer en H3. | non |
| G8 | Match VIP | 33 matchs VIP, 3 avec film complet, aucun cuit ; recommandé : Bazaar du 03/02 (`00761d27`). Couronne VIP jamais vérifiée sur un vrai film ; pont d'identité de la couronne manquant (registre l.40). | cuisson unitaire |
| S1 | « Appui reçu » vide le 22/09 | Données complètes (13 matchs, 137/137 frags lus, 51 appuyés = 37,2 %) ; le graphe trace l'ÉCART à l'habituel (≈ 37,8 %) → bâton quasi nul, lu comme « aucun appui ». Défaut de lisibilité, pas de calcul. | non |

Vue match, Tactique, Escouade : constats fichier:ligne dans les rapports (repris par item
ci-dessous). Régression « Médias » : commit `54d06bb7b` du 22/09 (« un titre de section ne
s'affiche jamais au-dessus de rien ») a supprimé l'état vide.

## 1. Décisions

Fermes (déjà tranchées par le user ou par les règles du projet) :
- **D-0a** La grammaire prime : aucune poussée ni pose publiée sur une heuristique de trajectoire
  si le film porte le signal ; les trajectoires servent de CONTRÔLE à seuil écrit d'avance.
- **D-0b** Une seule recuisson du parc, groupée, sur accord explicite ; jamais en lot sans
  demander. Les lots film partent de `feat/v75` APRÈS la fusion du lot images-clés (schéma 93).
- **D-0c** Fusion dans `feat/v75` : geste du user. Push `main` : jamais dans ce plan.

Réponses du user du 14/10 : D-4 ok (trois armes), D-5 partout, D-6 oui, D-7 colonnes ajoutées
(dominance, soirée, frags/morts/assistances de l'escouade, FDA d'escouade ADR 0006), D-8 la section
reste en Contributions, D-9 NON (l'appel en direct est fiable : diagnostic G7 corrigé, lot H3 réécrit).

Tranchées le 14/10 (second message du user) :
- **D-1 / D-2** Répulseur et pose de faille : le Theater du jeu les montre, donc le film les porte
  (le jeu actuel relit tous les films). AUCUNE déduction par trajectoire ni repli : le lot F
  trouve et publie le signal du film (grammaire, Ghidra si nécessaire). Un échec n'est pas un
  « négatif » acceptable : il se remonte au user avec ce qui a été lu, sans rien publier de déduit.
- **D-3** Sons : planche d'écoute publiée (https://claude.ai/artifact/AL8m5bUoPSkaiPZUo7MqSf),
  choix du user relus dans sa base (collection `choix`) avant B3 et F4. Le dépôt ne contient,
  pour ces deux équipements, que la pose de faille et les trois variantes du kill au répulseur
  (vérifié sur `origin/feat/v75` le 14/10).
- **D-3 bis** (user, 14/10, planche d'écoute) : téléportation du translocateur = `dcfaa487_152544be`
  (3,26 s ; rendu dans « Halo Infinite - Sons v75/rendus/ », à ré-échantillonner 48 kHz et plafonner
  -1 dBTP comme les autres, stem `translocator_teleport`). Répulseur : « déjà bon » — le son
  existant du répulseur est retenu, aucun nouveau son ; il sera rejoué sur la poussée lue dans le
  film (F4) en plus du kill.
- **D-10** CSR : classé ⟺ `RankRecap.PostMatchCsr.InitialMeasurementMatches > 0`, la règle
  d'OpenSpartan Workshop (`MatchTableEntity.IsRanked`, champ extrait par
  `Queries/Select/PlayerMatches.sql:52`). Validé par le user.

## 2. Règles d'environnement (tous les lots)

- Worktree dédié `../LevelUp-wt-aj14-<lot>`, branche `feat/aj14-<lot>` depuis `feat/v75`. Jamais
  le worktree principal, jamais `git stash`, jamais `git add -A`.
- `GOCACHE=%LOCALAPPDATA%\go-build-aj14-<lot>`, `GOLANGCI_LINT_CACHE=%LOCALAPPDATA%\golangci-aj14-<lot>`.
  Jamais `go build ./...` ; tests des paquets touchés, `go test ./...` une fois en fin de lot ;
  `-tags=integration -p 1` si sync/persist touchés.
- Web : `npm ci` dans le worktree ; `tsc -b`, eslint, vitest des fichiers touchés, puis
  `npx playwright test` avant fusion web (E2E non lancée par la CI hors PR vers `main`).
- Données : lecture via `LEVELUP_REPO_ROOT` sur le dépôt principal ; aucune écriture de base ni de
  cache hors des étapes marquées « écriture accordée ».
- L'exécutant ne modifie ni ce plan, ni le journal, ni le registre : il rend statuts, texte de
  journal et découvertes ; le superviseur reporte.
- Rendu du rejeu : vérifié EN LECTURE (vitesse 1×, image fractionnaire), jamais seulement à
  l'arrêt. UI : FR+EN, tokens de couleur, aucun possessif, aucune mention technique à l'écran.
- Fin de lot : gate vert, commits préfixés `aj14-<lot>`, push de la branche (CI), rapport,
  `scripts/disk-hygiene.ps1 -Lot aj14-<lot> -Apply`.

## 3. Lots

Ordre : A (P0 données) et B, C, D, E en parallèle tout de suite ; F après la fusion du lot
images-clés ; G (étiquettes de barres) après C, D et E (mêmes fichiers) ; H (opérations) en dernier.

### Lot A — CSR par match (P0, Go sync + backfill ; exécutant Sonnet, relecteur Opus)
- [ ] A1 (D-10, validé) **Un match est classé ssi `RankRecap.PostMatchCsr.InitialMeasurementMatches > 0`**
  (règle d'OpenSpartan Workshop ; nos tests le vérifient sur une charge classée ET une charge
  sociale réelles) — même esprit que
  règle que l'import OpenSpartan (`openspartan_import_service.go:313-319`, déjà « sa présence ⟹
  match classé »). Le nom de playlist (`isRankedPlaylist`) cesse de décider et est supprimé
  (code mort) ; la référence des playlists classées (`rankedplaylists.IsRanked`) ne décide pas
  non plus, elle sert de CONTRÔLE (alerte A3 si playlist de référence sans CSR, ou CSR sur une
  playlist hors référence).
- [ ] A2 L'extraction CSR (joueur et partagée) ne dépend plus de `reg.IsRanked` ; `is_ranked` du
  registre est posé depuis la présence du CSR (un seul point de décision, réutilisé par le sync,
  l'import et les rattrapages) ; garde « payload tronqué » conservé ; écritures par
  `BatchBuilder.Submit()` (règle ART n°1), lectures `_latest` inchangées.
- [ ] A3 Tests : charge de match classé réelle (fixture) → CSR écrit ; garde-rail grep interdisant
  `!reg.IsRanked` comme seul garde de l'extraction CSR ; alerte de qualité (CSR manquants sur
  matchs classés) branchée sur `csr_coverage_repo.go`.
- [ ] A4 Réparer `match_registry.is_ranked` depuis la présence du CSR (même point de décision
  qu'A2 ; commande de backfill idempotente, persister selon ADR 0019/0026).
- [ ] A5 (écriture accordée au go du plan, serveur arrêté, sauvegarde préalable) `backfill-csr`
  joueur puis partagé pour tous les profils, local ; mesure avant/après (matchs classés sans CSR
  depuis le 16/09 → 0 attendu hors matchs dont l'API ne rend rien).
- [ ] A6 Prod : lecture seule de l'état (prévenir le user avant toute commande VPS) ; backfill prod
  dans la séquence de déploiement, sur accord.
- Gate : `go test` des paquets touchés + `-tags=integration -p 1 ./internal/sync/...`, archlint,
  golangci nouveau = 0 ; tuile d'un match classé de Nuzzles du 07/10 avec CSR au navigateur.

### Lot B — Rejeu, web seul (Sonnet)
- [ ] B0 CONSTAT À L ÉCRAN avant tout code : rejeu d'un match à translocations cuit au schéma courant,
  lu à 1× autour de chaque téléportation (éclat de fiche, lien, faille, son) ; captures jointes au
  rapport. Ce qui se voit et ce qui ne se voit pas décide de B1-B2 (rien n'est « corrigé » à
  l'aveugle).
- [ ] B1 Translocation = coupure : modèle des images coupées par slot depuis `translocations[]` ;
  `positionAt`/`trailAt` (ou l'appelant `replayMarkers.ts`) ne relient jamais départ et arrivée.
- [ ] B2 Calque dédié aux failles APRÈS les trajectoires dans `LAYER_ORDER` ; éclat daté aux deux
  bouts (~1,2 s, `TRANSLOCATION_FLASH_S`), faille lisible (taille, animation) ; tokens.
- [ ] B3 Son de téléportation déclenché par `translocations[]` (geste choisi par le user, D-3) ;
  le déclencheur `translocator_beacon → translocator_deploy` (mort depuis début septembre) est
  rebranché sur la pose dès qu'elle est publiée (F2), son test fabriqué est remplacé par un test
  sur données réelles.
- [ ] B4 Garde-rail : aucun son de pose sur une famille `kind = "carried"` du manifeste ; chaque son
  de pose a au moins un déclencheur réel dans les fixtures.
- [ ] B5 Bandeau à manches : retrait du texte `a - b` et de son code mort (`roundsTally`, clés
  i18n, tests) ; « Manche X » en gras, couleur premier plan, 11 px.
- [ ] B6 Frises : `score_timeline_kind` du header passé à `scoreTrack` (`hidden` → pas de frise
  Score) ; la comparaison de rendu `sameLeadSegments` est SUPPRIMÉE avec ses tests (règle déclarée
  seule) ; contrôle de qualité : liste des variantes jouées sans règle déclarée (test sur le
  registre local, rouge si une variante marquant au frag n'est pas déclarée).
- [ ] B7 Forerunner : teinte `forerunner` sur les armes de D-4 au TOML, commentaire corrigé ;
  garde-rail Go : toute arme de faction `forerunner` du registre a la teinte `forerunner`.
- Tests : pion à mi-image d'une translocation jamais entre départ et arrivée ; aucune traînée
  n'enjambe une translocation ; ordre des calques ; bandeau ; frise `hidden` et deux frags même image.
- Gate : vitest + tsc + eslint + Playwright ; lecture à 1× d'un match à translocations, d'un
  Oddball à manches, d'un Slayer ; gate visuel du user.

### Lot C — Vue match (Sonnet)
- [ ] C1 Bloc Médias toujours affiché (sous la capacité `media`), état vide restauré FR/EN ;
  `hasMediaItems` supprimé ; test « liste vide → titre + message ».
- [ ] C2 Occupation du terrain : chaleur lissée de la Tactique (`lisserLaGrille` + flou de
  `peindreLaChaleur` déplacés dans `lib/replay/`, pas d'import entre features).
- [ ] C3 Retrait : « film décodé · N joueurs présents à la fin », « N socles vidés »,
  « (N types d'armes, repliés) » (chevron conservé), avec clés et champs morts.
- [ ] C4 « Prises par joueur » : ligne « Bonus perdus » retirée (composant partagé avec
  l'Escouade) + modèle `losses` mort ; selon D-6, aussi la ligne de « Ramassages ».
- [ ] C5 « N vies terminées par une mort » retiré selon D-5.
- [ ] C6 « Objectifs » en deux blocs : « Actions d'objectif par joueur », « Total d'objectif par
  équipe » (garde « exactement deux camps » conservé).
- Gate : vitest/tsc/eslint ; Vue match d'un CTF, d'un Oddball, d'un Slayer au navigateur.

### Lot D — Tactique (Sonnet)
- [ ] D1 Barre de chargement : drapeau `meta` « sans barre » (clé en constante `lib/query/`) sur la
  requête de cellule et son préchargement ; prédicat de `TopProgressBar` ; test avec vrai `QueryClient`.
- [ ] D2 Champ coéquipier (composant partagé Tactique/Escouade/Tendances) : `text-xs` en compact ;
  libellé « Ajouter un coéquipier… » / « Add a teammate… » (le champ ajoute à la composition,
  3 max, cherche au-delà des coéquipiers fréquents).
- [ ] D3 Pilule « Réapparition » masquée sans amas ni filtre actif ; gardée si un filtre est posé.
- [ ] D4 Colonne de droite : mode normalisé côté Go (`analysis.NormalizeModeLabel` avec la carte) ;
  icône d'arme (`arme_image_url`/`arme_image_tinted` publiés via l'adapter d'assets,
  `WeaponIcon` du fil du rejeu, repli texte) ; score coloré par l'issue, mot d'issue retiré
  (gardé en infobulle). `openapi.yaml` + `make generate-types`.
- Gate : Go des paquets touchés + vitest/tsc/eslint ; clic de zone sans barre au navigateur.

### Lot E — Escouade et Séries temporelles (Sonnet)
- [ ] E1 Historique des matchs de l'escouade (Synergies) : colonnes dominance, soirée,
  frags/morts/assistances cumulés de l'escouade, FDA d'escouade (formule ADR 0006, jamais le
  quotient), calculées côté Go.
- [~] E2 (D-8 : la section reste en Contributions, rien à faire)
- [ ] E3 « Contrôle des ressources, par match » : heure retirée du titre de ligne (prop, la carte
  est partagée avec Sessions).
- [ ] E4 « Contrôle des ressources, par soirée » en bâtons groupés (médiane et 50 % conservés,
  soirée non comparable = barre grisée), placée à gauche de « Rapport de force, soirée après
  soirée » dans une même rangée (hauteurs alignées ; seule carte → pleine largeur).
- [ ] E5 Frises sous l'axe X : helper partagé qui CALCULE `grid.bottom` et la position de la bande
  depuis la police, la hauteur de ligne et la marge des étiquettes (plus aucun nombre ajusté à la
  main) ; appliqué à `eveningsChart.ts`, `objectifCharts.ts`, `empriseCharts.ts` ; test
  « bande + étiquettes ≤ bas de grille ».
- [ ] E6 « Rapport de force au fil de la session » en bâtons par match et par rôle, cumul en
  infobulle.
- [ ] E7 « Appui reçu » : valeur absolue lisible (« 37 % (51/137) ») en étiquette et infobulle,
  écart à côté ; titre clarifié FR/EN (« Frags appuyés par un coéquipier ») ; test « soirée à
  l'habituel → valeur lisible ».
- Gate : vitest/tsc/eslint (+ Go si E1) ; Escouade et Séries temporelles au navigateur.

### Lot F — Film : poussée du répulseur et pose de faille (Opus ; après fusion images-clés)
- [ ] F1 Instruction répulseur, seuil écrit d'avance : lire les 104/105/119 par la grammaire de
  PRODUCTION (vue A), pas le harnais R12 (palette non classée sur ce build), sur 215e7022 et
  a6ae19fb (ancres Theater) puis 0d9a9af9 ; critère de validité : la poussée tombe sur le slot
  qui accélère dans 0-300 ms (contrôle trajectoire), ≥ 80 % des poussées détectées près d'un
  porteur retrouvées, les deux poussées de JGtm (0:55, 1:04) retrouvées. Le critère ne passe
  pas → on cherche ailleurs (autres événements, composants de l'objet `0x7ca85adc`, Ghidra),
  jamais de repli déduit (D-1) ; blocage remonté au user avec les lectures faites.
- [ ] F2 Instruction pose de faille : liste COMPLÈTE des 103, composants i18/i20/i21 des objets
  ti=37 de la balise (`0x730dc70f`), lecture complète des images-clés (lot images-clés fusionné),
  Ghidra pour l'effet de pose ; sur 1b2d9e08 et 2 autres films. Pas de déduction (D-2).
- [ ] F3 Faits : nouvelle section + révision du codec ; document :
  `knockbacks[]` et/ou pose de faille + couverture (lus, publiés, sans identité) ; schéma 93 → 94 ;
  goldens.
- [ ] F4 Web : schéma, normalisation, son de poussée (source du jeu identifiée, choix à l'oreille
  du user), onde visuelle (modèle `thrusterDashFx.ts`), pose de faille datée et sonore
  (`translocator_deploy` rebranché sur la vraie pose).
- [ ] F5 `REFERENCE_CANAUX_EQUIPEMENT` corrigé (geste vs effet).
- Gate : tests grammaire/replay, `TestCodecCouvreFilmInputs`, archlint (surface `decfilm`),
  rejeu de 0d9a9af9 et 1b2d9e08 cuit À L'UNITÉ et lu à 1× ; recuisson du parc en H2.

### Lot G — Étiquettes de barres (Sonnet ; après C, D, E)
- [ ] G1 Constat au navigateur du décalage vertical (cause non prouvée par lecture) avant correction.
- [ ] G2 Helper unique : encre noire ou blanche choisie au meilleur contraste WCAG avec la couleur
  du segment (`lib/accessibility/wcagContrast.ts`, couleurs résolues par `resolveToken`), composant
  `SegmentLabel` (centrage vertical unique) et variante ECharts.
- [ ] G3 Migration des 8 pistes HTML et 8 graphes ECharts recensés (isolement, ressources,
  objectif, équipements, ramassages, répartition des frags, échanges d'assistances, synthèses).
- [ ] G4 Garde-rail grep (modèle `teamTint.guard.test.ts`) : pas de `text-white`/`text-black` sur
  un segment hors helper, pas d'étiquette `inside` à couleur fixe.
- Gate : vitest/tsc/eslint ; captures claire et sombre de chaque page concernée ; gate visuel user.

### Lot H — Opérations (superviseur)
- [ ] H1 Cuire le match VIP recommandé à l'unité (`backfill-replay --one <id complet>`), après la
  recuisson commune du lot images-clés ; relever couronne, score et porteur ; consigner les
  défauts VIP (pont d'identité de la couronne, registre l.40) sans les traiter.
- [ ] H2 (accord explicite requis) Recuisson du parc au schéma du lot F + passes aval.
- [ ] H3 `seasons_catalog` (l'appel en direct est fiable, D-9) : confirmer la cause des 401/403
  intermittents (jeton de la requête expiré au moment de l'appel, cf. G7) ; sur 401/403, un seul
  nouvel essai après rafraîchissement du jeton (helper existant, jamais de recapture) ; l'échec
  d'un lecteur ne bloque plus le titre entier 30 min ; niveau WARN au changement d'état ; test.
  (Lot Go court, Sonnet.)

## 4. Découvertes (hors périmètre, non traitées)
- `MinePickupsCard` porte une ligne « Bonus perdus » (voir D-6).
- Démo (`feat/recos-d`, non fusionné) choisit déjà le VIP recommandé.
- Harnais R12 : palette « non classée » sur 0d9a9af9 (build non reconnu par la recherche).

## 5. Reprise de session
Lire ce fichier (cases), `git log --oneline -10` de chaque `feat/aj14-*`, `ListAgents`.
