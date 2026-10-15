# Plan — ajustements du 2026-10-14 (liste du user `Ajustements.txt`)

Origine : liste du user du 2026-10-14 (26 points : Tactique, Vue match, Séries temporelles,
Escouade, Général). Diagnostic en lecture seule le 2026-10-14 (six enquêteurs, copies des bases du
14/10 dans le scratchpad de session, aucune écriture). Contrat d'exécution : skill
`plan-execution`. Superviseur : session principale. Exécutants : Sonnet par défaut, Opus pour les
lots film (rétro-ingénierie) et pour la relecture du lot CSR (sync/persist).

**Statut (15/10) : lots A, B, C, D, E, F intégrés dans feat/v75 (feat/aj14-integ b49e6d29e, CI verte) ; lot équipement porté en correction de relecture ; restent G (étiquettes), A5/A6 (rattrapage CSR), H1/H2 (cuisson VIP, recuisson schéma 95).**

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
- **D-3 ter** (user, 14/10, second message) : le choix `152544be` est SUSPENDU. Le user veut les
  sons NATIFS du jeu (pose et téléportation), pas un rendu isolé d'une banque : un agent Sonnet
  explore (lot S) — fichier natif, ou recombinaison du geste tel que le moteur le joue (blend
  piloté par paramètre, cf. HANDOFF_SONS_RECONSTITUTION H4) ; Ghidra ouvert par le user.
  B3 et B4 passent au lot F (ils dépendent de S et de la pose publiée).
- **D-3 quater** (user, 14/10, validé à l oreille) : POSE = `pose_1_enregistrement_fidele` (cinq couches natives de la banque, remplace le contenu de `translocator_deploy.wav`) ; TÉLÉPORTATION = `teleport_5_video_depart_puis_effet` (départ `10f62776` puis effet `efe84c70` à +0,20 s, relevé sur une vidéo de jeu du user aux deux téléportations ; le coup `0431c88f` appartient à la pose seule). Branchement : lot F (F4 bis).
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
- [x] A1 (824fa6e72) (D-10, validé) **Un match est classé ssi `RankRecap.PostMatchCsr.InitialMeasurementMatches > 0`**
  (règle d'OpenSpartan Workshop ; nos tests le vérifient sur une charge classée ET une charge
  sociale réelles) — même esprit que
  règle que l'import OpenSpartan (`openspartan_import_service.go:313-319`, déjà « sa présence ⟹
  match classé »). Le nom de playlist (`isRankedPlaylist`) cesse de décider et est supprimé
  (code mort) ; la référence des playlists classées (`rankedplaylists.IsRanked`) ne décide pas
  non plus, elle sert de CONTRÔLE (alerte A3 si playlist de référence sans CSR, ou CSR sur une
  playlist hors référence).
- [x] A2 (824fa6e72, sync V1 et V2 + import) L'extraction CSR (joueur et partagée) ne dépend plus de `reg.IsRanked` ; `is_ranked` du
  registre est posé depuis la présence du CSR (un seul point de décision, réutilisé par le sync,
  l'import et les rattrapages) ; garde « payload tronqué » conservé ; écritures par
  `BatchBuilder.Submit()` (règle ART n°1), lectures `_latest` inchangées.
- [x] A3 (6526d6f98, fixture réelle Nuzzles, 5 garde-rails, alerte `reference_control`) Tests : charge de match classé réelle (fixture) → CSR écrit ; garde-rail grep interdisant
  `!reg.IsRanked` comme seul garde de l'extraction CSR ; alerte de qualité (CSR manquants sur
  matchs classés) branchée sur `csr_coverage_repo.go`.
- [x] A4 (6526d6f98, `levelup backfill-is-ranked`) Réparer `match_registry.is_ranked` depuis la présence du CSR (même point de décision
  qu'A2 ; commande de backfill idempotente, persister selon ADR 0019/0026).
- [x] A5 (15/10, serveur arrêté, sauvegarde `data/backups/avant_rattrapage_csr_2026-10-15/` : shared-csr +1 697 lignes, csr joueur +203 (Nuzzles ; défaut de candidature par joueur corrigé en route, fix/csr-candidats-joueur), is_ranked 4 121 corrigés, LUSR 7 732 lignes, engagement 7 010 ; ranked juillet-septembre ≈ 90 % classés) (séquence : backfill --all --shared-csr, backfill --all --csr, backfill-is-ranked, recalcul LUSR v2 canonique, engagement --force ; relecture adversariale du 14/10 corrigée (bb8ee5267, CI verte) ; séquence documentée FR/EN : backfill --all --shared-csr, backfill --all --csr, backfill-is-ranked, backfill --all --lusr, backfill --all --engagement-scores --force) (écriture accordée au go du plan, serveur arrêté, sauvegarde préalable) `backfill-csr`
  joueur puis partagé pour tous les profils, local ; mesure avant/après (matchs classés sans CSR
  depuis le 16/09 → 0 attendu hors matchs dont l'API ne rend rien).
- [ ] A6 Prod : lecture seule de l'état (prévenir le user avant toute commande VPS) ; backfill prod
  dans la séquence de déploiement, sur accord.
- Gate : `go test` des paquets touchés + `-tags=integration -p 1 ./internal/sync/...`, archlint,
  golangci nouveau = 0 ; tuile d'un match classé de Nuzzles du 07/10 avec CSR au navigateur.

### Lot B — Rejeu, web seul (Sonnet)
- [x] B0 (constat 1× : éclat, lien et faille rendus ; faille discrète ; défauts réels = glissade du pion et trait de traînée) CONSTAT À L ÉCRAN avant tout code : rejeu d'un match à translocations cuit au schéma courant,
  lu à 1× autour de chaque téléportation (éclat de fiche, lien, faille, son) ; captures jointes au
  rapport. Ce qui se voit et ce qui ne se voit pas décide de B1-B2 (rien n'est « corrigé » à
  l'aveugle).
- [x] B1 (a21fc751d) Translocation = coupure : modèle des images coupées par slot depuis `translocations[]` ;
  `positionAt`/`trailAt` (ou l'appelant `replayMarkers.ts`) ne relient jamais départ et arrivée.
- [x] B2 (a21fc751d, calque `failles` après `trajectoires`) Calque dédié aux failles APRÈS les trajectoires dans `LAYER_ORDER` ; éclat daté aux deux
  bouts (~1,2 s, `TRANSLOCATION_FLASH_S`), faille lisible (taille, animation) ; tokens.
- [~] B3 (déplacé en F4 bis, D-3 ter) Son de téléportation déclenché par `translocations[]` (geste choisi par le user, D-3) ;
  le déclencheur `translocator_beacon → translocator_deploy` (mort depuis début septembre) est
  rebranché sur la pose dès qu'elle est publiée (F2), son test fabriqué est remplacé par un test
  sur données réelles.
- [~] B4 (déplacé en F4 bis) Garde-rail : aucun son de pose sur une famille `kind = "carried"` du manifeste ; chaque son
  de pose a au moins un déclencheur réel dans les fixtures.
- [x] B5 (9fbc0e511) Bandeau à manches : retrait du texte `a - b` et de son code mort (`roundsTally`, clés
  i18n, tests) ; « Manche X » en gras, couleur premier plan, 11 px.
- [x] B6 (1e41c66cb ; reprise faite 1e2f69eb0 : pas de piste Score tant que l en-tête est inconnu ; feat/v75 fusionné 82f830cae) Frises : `score_timeline_kind` du header passé à `scoreTrack` (`hidden` → pas de frise
  Score) ; la comparaison de rendu `sameLeadSegments` est SUPPRIMÉE avec ses tests (règle déclarée
  seule) ; contrôle de qualité : liste des variantes jouées sans règle déclarée (test sur le
  registre local, rouge si une variante marquant au frag n'est pas déclarée).
- [x] B7 (93e2f4cce, garde-rail registre ⇔ teinte) Forerunner : teinte `forerunner` sur les armes de D-4 au TOML, commentaire corrigé ;
  garde-rail Go : toute arme de faction `forerunner` du registre a la teinte `forerunner`.
- Tests : pion à mi-image d'une translocation jamais entre départ et arrivée ; aucune traînée
  n'enjambe une translocation ; ordre des calques ; bandeau ; frise `hidden` et deux frags même image.
- Constat Chrome du superviseur (14/10, Vite aj14-b + serveur réel, lecture 1×, 1b2d9e08 au
  schéma 93, 3 téléportations) : saut net sans traînée, faille violette aux deux bouts, onde à
  l'arrivée, lien pointillé, éclat sur la fiche ; son de téléportation non branché (F4 bis).
  Captures : `~/Downloads/verif_translocateur_rejeu.png`.
- Gate : vitest + tsc + eslint + Playwright ; lecture à 1× d'un match à translocations, d'un
  Oddball à manches, d'un Slayer ; gate visuel du user.

### Lot C — Vue match (Sonnet)
- [x] C1 (943669b60) Bloc Médias toujours affiché (sous la capacité `media`), état vide restauré FR/EN ;
  `hasMediaItems` supprimé ; test « liste vide → titre + message ».
- [x] C2 (f4a0a4ad6 ; rendu à voir au gate visuel) Occupation du terrain : chaleur lissée de la Tactique (`lisserLaGrille` + flou de
  `peindreLaChaleur` déplacés dans `lib/replay/`, pas d'import entre features).
- [x] C3 (4daad5eba) Retrait : « film décodé · N joueurs présents à la fin », « N socles vidés »,
  « (N types d'armes, repliés) » (chevron conservé), avec clés et champs morts.
- [x] C4 (0e31550de ; ramassages compris, pastilles vides par prise conservées) « Prises par joueur » : ligne « Bonus perdus » retirée (composant partagé avec
  l'Escouade) + modèle `losses` mort ; selon D-6, aussi la ligne de « Ramassages ».
- [x] C5 (5a4f1d5c6 ; infobulle explicative de la carte conservée, à confirmer) « N vies terminées par une mort » retiré selon D-5.
- [x] C6 (7811cb59d ; sans infobulle, aucun texte d'aide n'existait) « Objectifs » en deux blocs : « Actions d'objectif par joueur », « Total d'objectif par
  équipe » (garde « exactement deux camps » conservé).
- Gate : vitest/tsc/eslint ; Vue match d'un CTF, d'un Oddball, d'un Slayer au navigateur.

### Lot D — Tactique (Sonnet)
- [x] D1 (10f2f332e) Barre de chargement : drapeau `meta` « sans barre » (clé en constante `lib/query/`) sur la
  requête de cellule et son préchargement ; prédicat de `TopProgressBar` ; test avec vrai `QueryClient`.
- [x] D2 (372b33088) Champ coéquipier (composant partagé Tactique/Escouade/Tendances) : `text-xs` en compact ;
  libellé « Ajouter un coéquipier… » / « Add a teammate… » (le champ ajoute à la composition,
  3 max, cherche au-delà des coéquipiers fréquents).
- [x] D3 (c374637a0) Pilule « Réapparition » masquée sans amas ni filtre actif ; gardée si un filtre est posé.
- [x] D4 (3cf328e0f) Colonne de droite : mode normalisé côté Go (`analysis.NormalizeModeLabel` avec la carte) ;
  icône d'arme (`arme_image_url`/`arme_image_tinted` publiés via l'adapter d'assets,
  `WeaponIcon` du fil du rejeu, repli texte) ; score coloré par l'issue, mot d'issue retiré
  (gardé en infobulle). `openapi.yaml` + `make generate-types`.
- [x] D5 (0257e0b42 ; cause : seul fetch à jeton sans `retryOnAuth` ; échec mémorisé par lecteur ; WARN au changement d'état) = H3 (`seasons_catalog`, Go court).
- Gate : Go des paquets touchés + vitest/tsc/eslint ; clic de zone sans barre au navigateur.

### Lot E — Escouade et Séries temporelles (Sonnet)
- [x] E1 (cumuls d escouade + FDA ADR 0006 côté Go, `attachSquadTotals`) Historique des matchs de l'escouade (Synergies) : colonnes dominance, soirée,
  frags/morts/assistances cumulés de l'escouade, FDA d'escouade (formule ADR 0006, jamais le
  quotient), calculées côté Go.
- [~] E2 (D-8 : la section reste en Contributions, rien à faire)
- [x] E3 (prop `showTime`) « Contrôle des ressources, par match » : heure retirée du titre de ligne (prop, la carte
  est partagée avec Sessions).
- [x] E4 (section « Soirées précédentes », barres groupées) « Contrôle des ressources, par soirée » en bâtons groupés (médiane et 50 % conservés,
  soirée non comparable = barre grisée), placée à gauche de « Rapport de force, soirée après
  soirée » dans une même rangée (hauteurs alignées ; seule carte → pleine largeur).
- [x] E5 (helper `components/charts/axisFoot.ts` + test sur options réelles + garde-rail) Frises sous l'axe X : helper partagé qui CALCULE `grid.bottom` et la position de la bande
  depuis la police, la hauteur de ligne et la marge des étiquettes (plus aucun nombre ajusté à la
  main) ; appliqué à `eveningsChart.ts`, `objectifCharts.ts`, `empriseCharts.ts` ; test
  « bande + étiquettes ≤ bas de grille ».
- [x] E6 (barres par match et par rôle) « Rapport de force au fil de la session » en bâtons par match et par rôle, cumul en
  infobulle.
- [x] E7 (« 37 % (51/137) » sur le bâton) « Appui reçu » : valeur absolue lisible (« 37 % (51/137) ») en étiquette et infobulle,
  écart à côté ; titre clarifié FR/EN (« Frags appuyés par un coéquipier ») ; test « soirée à
  l'habituel → valeur lisible ».
- Gate : vitest/tsc/eslint (+ Go si E1) ; Escouade et Séries temporelles au navigateur.

### Lot S — Sons natifs du translocateur (Sonnet, lecture seule du dépôt, en parallèle)
- [x] S1 (pose : 5 couches natives prouvées, 96,1 % de l énergie de l enregistrement ; téléportation : seul `10f62776` = play_007_abl_quantum_teleport_player_start nommé, geste entier non prouvé ; `152544be` = version monde du coup de POSE) Retrouver la pose et la téléportation telles que le jeu les joue : événements de la
  banque de l'appareil et de la balise, chaîne équipement → effet → son, blend et paramètre
  qui le pilote ; Ghidra (HTTP 127.0.0.1:8089, lecture seule) pour nommer les événements.
- [x] S2 (8 rendus, planche v2 publiée le 14/10 ; trou : quel effet est la téléportation — enregistrement en jeu du user ou clés d événement via Ghidra) Rendus candidats (pose, téléportation) dans le scratchpad + rapport ; le superviseur
  les ajoute à la planche d'écoute, le user choisit.

### Lot F — Film : poussée du répulseur et pose de faille (Opus ; après fusion images-clés)
- [x] F1 (canal ÉTABLI le 14/10 : 104 ref0 = slot poussé, impulsion portée depuis l exe, 105 ref1 = auteur ; 0d9a9af9 : JGtm à 0:54,8 et 1:04,4 par WolfzyLIVE, Sterio23 à 3:59,0 par Clearshields ; ancre 215e7022 à la ms du kill ; contrôle trajectoire 14/14 contre 1/14 au témoin ; rappel 7/9 dont 2 faux positifs du détecteur (fentes de corps-à-corps), 7/7 sur les vraies poussées ; 104 jamais en tête de liste) Instruction répulseur, seuil écrit d'avance : lire les 104/105/119 par la grammaire de
  PRODUCTION (vue A), pas le harnais R12 (palette non classée sur ce build), sur 215e7022 et
  a6ae19fb (ancres Theater) puis 0d9a9af9 ; critère de validité : la poussée tombe sur le slot
  qui accélère dans 0-300 ms (contrôle trajectoire), ≥ 80 % des poussées détectées près d'un
  porteur retrouvées, les deux poussées de JGtm (0:55, 1:04) retrouvées. Le critère ne passe
  pas → on cherche ailleurs (autres événements, composants de l'objet `0x7ca85adc`, Ghidra),
  jamais de repli déduit (D-1) ; blocage remonté au user avec les lectures faites.
- [x] F1 bis (consigne du user, 14/10 ; charges : négatif DÉCRIT — i56 jamais armé par le répulseur, delta des objets portés NON LU par la marche, images-clés brutes non interprétables ; Ghidra : `Equipment_GetRemainingCharges` lit objet+0x50b, composant i27 `equipment-charges-remaining` écrit +0x5a7, lien non établi) Validation par l équipement et les charges : auteur porteur du répulseur 130/130 paires ; publication des seules paires 105 → 104 (normes 6,00 / 5,50 m/s), 998 104 isolées = lectures fausses (genre 36), non publiées, comptées ; charges : aucun canal décroissant aux images-clés, delta des objets portés non lu → instruction bornée (Ghidra : source des utilisations restantes du HUD).
- [~] F2 (RÉSOLU par le lot équipement porté, levelup-80, 14/10 : pose = énergie i24 → 0 puis i28 +1 entrée, ~1,63 s après l activation i21, 123/123 au parc ; ancres 1b2d9e08 164,11 s, 6acce632, a0c36016 ; cause commune = i28 lisait n+1 entrées) (ex-BLOQUÉ 14/10, rien de déduit publié : aucun 103 ni événement de vue A aux poses ; la pose est très probablement dans le DELTA de l objet porté, que la marche de vue B ne lit pas — elle s arrête sur un composant non porté avant ces records ; charge du genre 117 non portée non plus) Instruction pose de faille : liste COMPLÈTE des 103, composants i18/i20/i21 des objets
  ti=37 de la balise (`0x730dc70f`), lecture complète des images-clés (lot images-clés fusionné),
  Ghidra pour l'effet de pose ; sur 1b2d9e08 et 2 autres films. Pas de déduction (D-2).
- [x] F3 (a99688494, poussé ; grammar-2026-10-14, schéma 94, calque `knockbacks[]`, paire 105 → 104 seule) Faits : nouvelle section + révision du codec ; document :
  `knockbacks[]` et/ou pose de faille + couverture (lus, publiés, sans identité) ; schéma 93 → 94 ;
  goldens.
- [x] F4 (e67fa8236 ; onde + sillage, son du répulseur par souffle dédoublonné avec le kill ; lecture 1× au navigateur à faire après recuisson) Web : schéma, normalisation, son de poussée (source du jeu identifiée, choix à l'oreille
  du user), onde visuelle (modèle `thrusterDashFx.ts`), pose de faille datée et sonore
  (`translocator_deploy` rebranché sur la vraie pose).
- [~] F4 bis (e67fa8236 : son de téléportation branché, contenu de translocator_deploy.wav remplacé ; déclencheur de pose + test réel + garde-rail → lot équipement porté) (ex-B3/B4) Son de téléportation (choix S2) sur `translocations[]` ; son de pose
  rebranché sur la pose publiée ; test fabriqué remplacé par un test sur données réelles ;
  garde-rail : chaque son de pose a un déclencheur réel dans les fixtures.
- [x] F6 (b1009e8e9 ; killsource json identique à l octet sur 20 films ; golden de forme du schéma 94 régénéré sans relever le schéma — ACCEPTÉ par le superviseur : le schéma 94 n a jamais été publié ni cuit au parc) Relecture adversariale (14/10) : P1 compléments datés killsource/objectives à révision constante + preuve ; P2 test du dédoublonnage son/kill ; P3 auteur dans le dédoublonnage, test de génération, `WithoutImpulse` publié, chiffres alignés — correction en cours.
- [x] F5 (e67fa8236) `REFERENCE_CANAUX_EQUIPEMENT` corrigé (geste vs effet).
- Gate : tests grammaire/replay, `TestCodecCouvreFilmInputs`, archlint (surface `decfilm`),
  rejeu de 0d9a9af9 et 1b2d9e08 cuit À L'UNITÉ et lu à 1× ; recuisson du parc en H2.

### Lot EP — Équipement porté (session levelup-80, prompt du superviseur)
- [x] G1-G5 livrés sur feat/equip-porte @ 17a14ea69 (CI verte) : correctif i28 (n et non n+1, FUN_140f72dec) ; objet porté lu au delta (i24/i21/i22/i28) ; pose de faille = i24 → 0 puis i28 +1, 1,63 s après i21 (123/123) ; catalogue `equipment_energy.json` lu dans les tags (voie A) ; schéma 95 (`carriedEquipment[]`, `riftPoses[]`) ; faille à la pose, son de pose calé, usages restants sur la fiche ; rendu vérifié en lecture (harnais du lot B).
- [x] EP6 (783fbc6f0 ; capteur, traqueur, champ, écran sonnent à chaque utilisation lue ; mur garde sa pose ; témoins 5676a9ba et 1f8e8849, 30/30) (décision user 15/10) Rebrancher les sons de pose du capteur, du traqueur, du champ de réparation et de l écran occultant sur `carriedEquipment[].uses` ; liste datée de `poseSoundTriggers.guard.test.ts` vidée.
- [x] Relecture adversariale (15/10) : 0 P1, 3 P2 (calques absents des tests figés, mur rang 19 jamais compté, journal), 5 P3 ; i28 et règle du HUD confirmés contre l exe → corrections transmises à levelup-80.
- [x] Fusion : avance rapide de feat/v75 sur feat/equip-porte @ 027a5bcaf (CI verte, relecture traitée : P2-1 fait, P2-2 réfuté par mesure — énergie absente = énergie de l objet porté, les deux fiches du mur portent 1 —, P3 faits).
- [x] EP7 (4cd7886f0, dc03981a8 ; SCHÉMA 96, grammar-2026-10-15, faits 14 ; 18/18 fins de champ lues au parc ; killsource/objectives constantes prouvées par replay-equiv) (user 15/10, session levelup-80) Fin du champ de réparation lue dans le film (entité ti=41 nommée par la pile i28, DEL = fin) → son `repair_field_end` ; schéma 95 amendé (non cuit).
- [x] Recuisson du parc au schéma 96 (15/10, 11:04-12:55, rc 0 partout) : 238 artefacts ; au parc riftPoses 123, knockbacks 132, fins suivies 745/758 ; killsource 0 erreur de décodage, 1 erreur d écriture (34db6b33@40500, tueur divergent crédit/film) ; VIP 00761d27 cuit (72 pistes) ; rasters 238 ; air relancé. Séquence : replay --only-existing, usage-summary, pad-tiers --force, vehicle-takes --force, bomb-stats, flag-grabs-net, killsource, VIP 00761d27 (--one), tactical-rasters ; + backfill killsource (killsource-2026-10-14 : chemin de lecture de 23 morts sur deux films) — accord user.

### Lot G — Étiquettes de barres (Sonnet ; après C, D, E)
- [x] G1 (cause : ancre inline-flex du Tooltip alignée sur la ligne de base, +2 à +2,5 px ; après 0 px) Constat au navigateur du décalage vertical (cause non prouvée par lecture) avant correction.
- [x] G2 (879cadad5 : segmentInk, SegmentLabel, barLabel) Helper unique : encre noire ou blanche choisie au meilleur contraste WCAG avec la couleur
  du segment (`lib/accessibility/wcagContrast.ts`, couleurs résolues par `resolveToken`), composant
  `SegmentLabel` (centrage vertical unique) et variante ECharts.
- [x] G3 (879cadad5) Migration des 8 pistes HTML et 8 graphes ECharts recensés (isolement, ressources,
  objectif, équipements, ramassages, répartition des frags, échanges d'assistances, synthèses).
- [x] G4 (segmentInk.guard.test.ts) Garde-rail grep (modèle `teamTint.guard.test.ts`) : pas de `text-white`/`text-black` sur
  un segment hors helper, pas d'étiquette `inside` à couleur fixe.
- Gate : vitest/tsc/eslint ; captures claire et sombre de chaque page concernée ; gate visuel user.

- [x] G5 (1fb5fd253 : carte « Appui reçu » de Sessions, nom d axe de la frise)
- [x] Gate visuel du user (15/10, « Parfait ! ») : valeurs à côté de la barre (27bb98209) ; texte blanc partout avec contour sombre (9572c5133). Fusion après la recuisson (gel de feat/v75).

### Lot H — Opérations (superviseur)
- [x] H1 (00761d27 cuit au schéma 96 le 15/10 ; vérification couronne/score/porteur à faire par le user) Cuire le match VIP recommandé à l'unité (`backfill-replay --one <id complet>`), après la
  recuisson commune du lot images-clés ; relever couronne, score et porteur ; consigner les
  défauts VIP (pont d'identité de la couronne, registre l.40) sans les traiter.
- [x] H2 (fait le 15/10, voir lot EP) (accord explicite requis) Recuisson du parc au schéma du lot F + passes aval.
- [x] H3 (exécuté par le lot D, item D5, 0257e0b42) `seasons_catalog` (l'appel en direct est fiable, D-9) : confirmer la cause des 401/403
  intermittents (jeton de la requête expiré au moment de l'appel, cf. G7) ; sur 401/403, un seul
  nouvel essai après rafraîchissement du jeton (helper existant, jamais de recapture) ; l'échec
  d'un lecteur ne bloque plus le titre entier 30 min ; niveau WARN au changement d'état ; test.
  (Lot Go court, Sonnet.)

## 4. Découvertes (hors périmètre, non traitées)
- (recuisson 96, PRÉEXISTANTS, hors correctif i28 selon levelup-80 et le journal) killsource : 34db6b33@40500 = D-39 déjà consigné le 14/10 ; « tueur divergent entre crédit et film » (la clef (match_id, time_ms) apparie deux morts) ; 73 films « équipe de bot(s) NON LUE dans BOT_METADATA » (HI_1_12_0, hors_grammaire=1 ; lecture BOT_METADATA suspecte sur HI_1_12_0 déjà consignée, journal l.116685) ; 4 « vies publiées sans identité » (69b16f5d, 4db2574e, c75f33b8, 9f57c612).
- (15/10) Commit étranger non poussé dans le checkout principal pendant le gel : 17a896063 « feat(citations): nouvelle image pour la citation Is that my ball? » (auteur inconnu de cette session).
- (équipement porté) `repair_field_end` reste attaché à une pose `deployed` du champ : ne part jamais (rebranchement à décider).
- (équipement porté) 5676a9ba : une utilisation de mur sans pose `deployed` (slot 557, image 1406) reste muette ; 8 murs pris à 0,99 sans fiche → comptés ambigus.
- (équipement porté) `TestBatchQueue_Drain_PartialFailure_OnlySomeACKed` (persist) rouge une fois sous charge, vert 3/3 seul.
- (intégration) 22 tests E2E (legacy-redirect, slice-0b, 2b, 3b, 4, 4b, 4c, 7, 8, 9) appellent l API en dur sur `localhost:8000` au lieu de l URL de base configurée : inexécutables hors du dispositif CI ; échecs identiques sur feat/v75. Le reste : 57 passées, 28 ignorées.
- (intégration) golangci signale des directives `//nolint` mal formées ; `npm audit` signale des vulnérabilités.
- (relecture EP) `TestGoldenInputsFidelite` (gate local) rouge à l identique sur b1009e8e9 : dette antérieure.
- (équipement porté) Sons de pose MORTS depuis le 15/09 pour capteur, traqueur, champ de réparation et écran occultant (seul le mur porte une pose `deployed`) ; listés dans `poseSoundTriggers.guard.test.ts` ; rebranchement sur `carriedEquipment[].uses` = décision produit.
- (équipement porté) Mur : deux fiches de coût (0,33 et 1) → `ambiguousCost` ; identifiant atypique 0x0000d5ff recopié tel quel.
- (équipement porté) Live Fire - Ranked : ~16 % des trames ne ferment pas (rejet hors datum + vue C), autre défaut que i28.
- (équipement porté) Charges : HUD = floor((énergie + 1e-4) / coût par usage) ; coût et énergie initiale = constantes du tag eqip (non transmises par le film) ; baisses mesurées par famille : répulseur 0,2 (énergie 0,6 au ramassage = 3 charges), capteur et écran 0,25, mur 0,33, champ 0,5, translocateur 1. DÉCISION USER 14/10 : VOIE A (constantes lues dans les tags eqip du jeu, référence générée et versionnée par GlobalID, formule du HUD), sur la base B (énergie + consommations datées).
- (lot F) `ScanKnockbacks` relit toutes les vues A dans une passe à part au lieu du pont d identité (coût seulement).
- (lot F) a6ae19fb slot 526 : fausse alerte de régression i48 — rang 6 confirmé par le lot images-clés ; le 10 venait d une lecture sans contexte de carte (`NewFilmContext` nu) ; contrôle porteur F remis au critère strict via `NewFilmContextForMap`.
- (lot A) Après réparation d is_ranked, ~3 900 matchs sortent du LUSR (qui exclut les classés) : recalcul LUSR ajouté à A5.
- (lot A) Migration historique `shared_backfill_is_ranked_and_season`, `classifyPeakType`, `countRankedMatchesInRegistry` ne passent pas par la règle.
- (lot A) `needs_backfill` ne tient pas compte du contrôle de référence.
- (lot F) CAUSE COMMUNE de la pose de faille et des charges illisibles : la vue B s arrête sur un composant non porté avant les records des objets d équipement portés (delta jamais lu pendant le port, seul le NEW l est) ; la charge du genre 117 n est pas portée. Proposition : lot de grammaire dédié (lire le delta des objets portés, i20/i10/i27), puis recuisson.
- (lot E) La carte « Appui reçu » de Sessions garde l ancien titre et l écart seul (même défaut de lecture que E7).
- (lot E) Étiquette d axe « Écart (points) » rognée à gauche (grille `left: 8` + `containLabel`).
- (lot E) Plafond d imports croisés entre features à 7/7.
- (lot B) Harnais Playwright jetable du rejeu (API simulée, document réel) dans le scratchpad : à versionner si utile (aucun harnais de rendu du rejeu au dépôt).
- (lot B) Le contrôle de qualité B6 porte une liste d exceptions en dur (VIP, Total Control : objectifs sans entrée de catalogue).
- (lot B) Armes Halo 5 classées `forerunner` au registre sans `[shot_tints]` H5 : non contrôlées.
- (lot F) RETIRÉ : la « sous-lecture du genre 36 » (998 faux 104, 9 % de listes arrêtées) était un artefact des instruments sans contexte de carte ; dans le contexte de la cuisson : 135 104 = 135 paires, 0,42 % de listes arrêtées. RÈGLE pour tout instrument de recherche : lire sous `NewFilmContextForMap` + `PoserLaCarteEtLeDecoupage` (comme `replay/build_from_film.go:97`).
- (lot D) `FetchCSRSeasonCalendar` n'est pas enrobé de `retryOnAuth` (même défaut que D5).
- (lot D) Sans xuid dans le contexte, `RetryWithFreshTokens` ne relance pas.
- (lot D) Code mort `errEmpty` en pied de `seasons_catalog.go` ; `registry_pages.go:46` cite l'ancien message.
- (lot D) `TestAddOverlayEntryConcurrentNePerdPasDEntree` (mapcatalog) rouge une fois sur deux passages : instable.
- (lot D) L'exécutant a utilisé des scripts Python pour éditer (CRLF corrigés) : règle « pas de Python » rappelée aux exécutants suivants.
- (lot C) Le champ `outcomes` (kept/dropped) du bloc Emprise, servi par Go et déclaré dans
  `openapi.yaml`, n'a plus de lecteur web depuis D-6.
- (lot C) `lib/replay/crossFeatureBoundary.guard.test.ts` ne couvre que match-view ↔ match-replay ;
  aucun garde n'interdit un import features/tactical → features/match-view.
- (lot C) Playwright de la Vue match exige le serveur :8000 : à rejouer avant la fusion web.
- `MinePickupsCard` porte une ligne « Bonus perdus » (voir D-6).
- Démo (`feat/recos-d`, non fusionné) choisit déjà le VIP recommandé.
- Harnais R12 : palette « non classée » sur 0d9a9af9 (build non reconnu par la recherche).

## 5. Reprise de session
Lire ce fichier (cases), `git log --oneline -10` de chaque `feat/aj14-*`, `ListAgents`.
