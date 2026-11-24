# Plan : fonds de carte natifs tenus par le maillage de navigation, calque « plan » et rotation (2026-11-13)

Statut : **CLOS le 2026-11-13** : PR #205 fusionnée dans `release/v8.2.0` (5417b2ba3e), briques `v1.1.0` taguées. Le nettoyage des caches de compilation (`disk-hygiene -Lot navmesh`) n'est PAS fait : incident sur l'archive des films vers 18 h 30 (interdiction de toute suppression et écriture lourde sur C:), à faire quand levelup-dd lève la consigne. Le worktree `../LevelUp-wt-navmesh` reste en place : il porte trois jonctions vers `data/` du dépôt principal, à retirer par `rmdir` sans `/s` avant toute suppression.

## 1. Objectif et critère de succès

Les cartes natives (studio) sont aujourd'hui cuites depuis leur géométrie de rendu seule, avec des
réglages à la main par carte (boîtes, plafonds, masques de zones, bornes aux volumes de mort :
`data/titles/halo_infinite/reference/map_fond_reglages.json`, 1 257 lignes). Les cartes Forge, elles,
sont tenues par leur maillage de navigation (`navmesh.blob`) : référence de sol et rognage. La mesure du
2026-11-13 (thought_log) établit que **29 des 31 modules natifs portent le même `hkaiNavMesh`**, dans
le module `any`, lisible par `hinavmesh` sans changer le décodeur.

Le plan est réussi quand :

- une carte native se cuit avec son maillage comme référence de sol et comme masque de rognage, par la
  chaîne de cuisson existante, sans réglage manuel neuf ;
- chaque fond publié (natif ou Forge) porte, dans son sidecar, le **plan des murs** (arêtes de bord du
  maillage, en mètres monde) et le rejeu le dessine en calque, activable ;
- le rejeu se tourne par quarts de tour, réglage mémorisé par carte, sans recuisson ni changement de
  données ;
- les 20 fonds natifs sont recuits, mis en planche avant/après, et validés visuellement par le user carte
  par carte (le gate visuel est le sien : `feedback_visual_gate_user_names_witnesses`) ;
- Live Fire (`sgh_interlock`) et le tutoriel, sans maillage, gardent la chaîne actuelle et le disent
  dans leur sidecar (`degradations`), sans calque plan.

Hors périmètre : tout état en direct du jeu (cycles de socles, poignées d'objets : lecture mémoire chez
Den, pas une source pour nous) ; la carte de chaleur de la vue match (`MatchPositionsHeatmap`, même
fond, autre page : rotation non portée, noté en Découvertes) ; la purge des réglages manuels devenus
inutiles (listés en Découvertes, décision du user après la planche) ; les positions de force (le
maillage natif lève la limite notée dans `.ai/AUDIT_POSITIONS_DE_FORCE_2026-10-27.md` § 4, chantier
séparé).

## 2. État des lieux (mesuré le 2026-11-13, sonde hors dépôt)

- Maillage natif : module `deploy/any/levels/multi/<carte>/<carte>-rtx-new.module`, ressource sans groupe
  voisine du tag `pfnd`, en-tête propriétaire `ucsh` (0x1b octets d'en-tête, puis table), puis quatre
  fichiers-tag Havok 2022.1 : `hkaiClusterGraph`, `hkaiNavMesh`, `hkaiTraversalAnnotationLibrary`,
  `hkcdStaticAabbTree`. Ni conteneur Bond ni zlib (différence avec le blob Forge). Bazaar : 2 312
  sommets, 670 faces, 51,4 × 27,9 m, `up` déclaré (0,0,1), érosion 0.
- Couverture : 29 modules sur 31. Absents : `sgh_interlock` (Live Fire), `academy_tutorial`. Maillages
  secondaires de 11 à 21 Ko à côté du principal : Chasm (5), Launch Site (5), Cliffhanger (1).
- Calage : le fond « encre » de Bazaar rogné au maillage se superpose au pixel près (même repère monde).
- Chaîne existante : `himap.CuitCarteForge` porte déjà `NavmeshReference`, `RogneAuNavmesh`,
  `MargeNavmeshCarte`, `ToleranceNavmesh`, `MaillageNiveauHaut` (`cuisson_forge.go:114-134`,
  `reference_navmesh.go`). `CuitCarteNative` (`OptionsCuisson`) ne les a pas.
- Depuis `release/v8.2.0`, LevelUp consomme les briques publiées (`go.mod` : `halo-infinite-files-reader
  v1.0.1`, `halo-infinite-map-images v1.0.1`), plus de `libs/` ni de `go.work` sur cette branche. Le code
  des briques se modifie dans leurs dépôts dédiés (plan v8.1.2, D-14) ; pseudo-version `@<sha>` permise
  sur une branche de travail, tag obligatoire avant `main`, **tags des briques posés par le user**.
- Web : fond affine sans rotation (`layers/mapBackground.ts:14,74`, `ReplayCanvas.tsx:488`) ; projection
  unique `canvasView` partagée par le dessin et le survol (`hooks/useReplayView.ts`) ; calques à
  interrupteurs dans `settings/ReplaySettingsLayers.tsx`, état `settings/useReplaySettings.ts`, persistance
  `settings/replayPreferences.ts` ; zones de callout dessinées en vecteurs monde (`lib/replay/calloutsPaint.ts`,
  modèle pour le calque plan) ; `ReplayCanvas.tsx` au plafond de taille (781 lignes : extraire avant d'ajouter).
- Sidecar : `replaydoc.MapBackground` (`domain/replaydoc/map.go:97`), servi par
  `service/replay_map_background.go` ; l'image n'est servie que si le sidecar est lisible.

## 3. Décisions tranchées

| # | Décision | Raison |
|---|---|---|
| D-1 | Le maillage natif se lit par `hinavmesh.DecodeNatif(ressource []byte)` : régions trouvées par signature `TAG0` + taille de section, maillage cherché par son TYPE (règle déjà en place pour la génération TSTR/FSTR), refus si plusieurs `hkaiNavMesh` dans une même ressource | Le préambule à cinq régions du blob Forge n'existe pas ici ; le décodeur de fichier-tag ne change pas |
| D-2 | La ressource se trouve par `himodule` : parmi les ressources sans groupe du module `any` de plus de 10 Ko, celles qui portent la chaîne `hkaiNavMesh` ; la plus grosse est le maillage principal, les autres sont comptées dans le bilan (« maillages secondaires », non dessinés) | Lien `pfnd` -> ressource non décodé ; le module `any` fait ~14 Mo, le balayage coûte moins d'une seconde ; qualifier les secondaires = Découverte |
| D-3 | Les cartes natives gardent le rendu riche (`CuitCarteNative`) ; on lui ajoute les options navmesh de `CuitCarteForge` par un helper partagé (règle des 2 copies : `appliqueNavmesh(r, bilan, opts)` dans `himap`, appelé des deux chaînes). Marge de rognage par défaut natif : 1,5 m, réglable par carte dans `map_fond_reglages.json` | Le user veut le fond « satellite » rempli, cadré par le maillage ; la mécanique existe |
| D-4 | Le plan des murs est un **vecteur**, pas une image : arêtes de bord du maillage (arête vue une seule fois) en mètres monde, dans le sidecar (`plan: { edges: [[x1,y1,x2,y2],...], vertices, faces, source }`, `omitempty`). Dessiné par un calque canvas en tokens d'encre, trait fin, par-dessus le fond, sous les pions | Net à tout zoom, suit la rotation sans recuisson, 50 Ko pour Bazaar ; une image de plus doublerait le poids servi |
| D-5 | Calque « Plan des murs » (fr) / « Wall plan » (en) : interrupteur dans les calques du tiroir, **actif par défaut** quand le sidecar porte un plan, absent sinon | Pas de feature livrée éteinte (règle 11) ; le user juge à l'écran |
| D-6 | Rotation : quatre positions (0, 90, 180, 270), appliquée dans la projection `canvasView` (permutation des axes monde avant la projection ; l'image de fond se dessine sous `ctx.rotate`), donc survol, export vidéo et tous les calques suivent sans une ligne. Réglage **par carte**, persisté dans `replayPreferences` | Une seule définition de la projection ; chaque carte a son orientation naturelle |
| D-7 | Les cartes Forge reçoivent aussi le plan (même exportateur, maillage déjà chargé) ; leur cuisson ne change pas | Un seul sidecar, un seul calque |
| D-8 | Live Fire et tutoriel : chaîne actuelle, `degradations` : « aucun maillage de navigation dans le module any » ; pas de plan | Mesuré ; pas de repli inventé |
| D-9 | Recuisson des 20 natifs avec les réglages manuels **conservés** ; planche avant/après par `cmd/mapfond-planche` ; le user désigne carte par carte ce qu'il valide ; les réglages rendus inutiles sont listés, pas retirés | Témoins avant/après (`feedback_temoins_avant_apres_refactor`) ; remarque n'est pas validation |
| D-10 | Briques : branches de travail dans les dépôts dédiés (`halo-infinite-files-reader`, `halo-infinite-map-images`), pseudo-version dans LevelUp pendant le développement, **tags v1.1.0 posés par le user** avant la fusion vers `main` | Plan v8.1.2 D-14 ; `feedback_tags_depot_decodeur_delegues` ne délègue que le décodeur |
| D-11 | Si le plan « maquettes rejeu » (bouton Vue, D-6 de ce plan) est fusionné avant l'étape 4, l'interrupteur et la rotation vont dans le panneau Vue ; sinon dans le tiroir, et le plan maquettes les déplace | Les deux touchent `settings/` ; le second à passer rebase |

## 4. Prérequis et branches

- Branche LevelUp : `feat/fonds-navmesh-natif` depuis `release/v8.2.0`, worktree dédié
  `../LevelUp-wt-navmesh`. Pas de PR par étape (dev solo) ; une PR vers la release à la fin.
- Clones des briques (absents du poste) : `../halo-infinite-files-reader` et `../halo-infinite-map-images`,
  branches `feat/navmesh-natif` et `feat/cuisson-navmesh-natif` depuis `main`.
- Jeu installé (`C:/XboxGames/Halo Infinite/Content/deploy`, détecté par `himap.DeployRoot`) pour les tests
  `gamefiles` et la recuisson.
- Caches Go dédiés par lot (skill `plan-execution` § 5) ; clôture par `scripts/disk-hygiene.ps1 -Lot`.
- Agents Sonnet, une ronde `adversarial-review` par étape à risque (1, 2, 4). Coût annoncé au user avant
  tout workflow.

## 5. Étapes

Statuts : `[x]` fait, `[~]` couvert ailleurs (référence), `[!]` non traité (justification écrite). Une
étape est close quand tous ses items portent un statut, son gate est passé et consigné au Journal (§ 7).
Étape N close avant N+1. Zéro fix hors périmètre : § 8.

### Étape 0 — Mise en place (rapide)

- [x] 0.1 Cloner les deux dépôts dédiés, créer les branches ; créer le worktree LevelUp.
- [x] 0.2 Témoins AVANT : copier les 20 fonds natifs et leurs sidecars actuels dans
  `.ai/V8.3/annexes_fonds_navmesh/avant/` (hors dépôt git : dossier listé dans `.gitignore` ou archive hors
  arbre) ; planche `cmd/mapfond-planche` « avant ».
- Gate : les deux clones compilent (`go vet ./... && go test ./...`), planche « avant » écrite.

### Étape 1 — Brique `halo-infinite-files-reader` : maillage natif (rapide)

- [x] 1.1 `hinavmesh/natif.go` : `DecodeNatif(ressource []byte) (*Maillage, error)` — en-tête `ucsh`
  vérifié (signature, longueur), régions par signature `TAG0` et taille de section (bits de drapeau
  masqués), maillage par type, erreurs typées (sans `TAG0`, aucun maillage, plusieurs maillages).
  `Decode` (Forge) inchangé ; le parcours des régions commun est extrait si les deux chemins partagent
  plus de quelques lignes.
- [x] 1.2 `himodule/navmesh.go` : `RessourcesNavmesh(m *Module) ([]File, error)` — D-2, triées par taille
  décroissante ; journal `slog.DebugContext` par ressource trouvée.
- [x] 1.3 Tests : `natif_test.go` (buffer synthétique : en-tête tronqué, aucun `TAG0`, deux maillages) ;
  `natif_gamefiles_test.go` (tag `gamefiles`, nommage imposé par `archlint/gamefiles_tag_test.go`) : Bazaar
  = 2 312 sommets, 670 faces, emprise à 0,1 m près, `HautSuppose=false` ; balayage des 31 modules : 29
  maillages, les deux absents nommés, secondaires comptés (Chasm 5, Launch Site 5, Cliffhanger 1).
- [x] 1.4 README de la brique : paragraphe « native maps » ; `doc.go` de `hinavmesh` : la phrase
  « n'existe QUE pour les cartes Forge » (`conteneur.go:24-29`) est fausse depuis la mesure, à corriger
  (règle 17 : affirmation fausse rencontrée = corrigée).
- Gate : `go vet ./... && go test ./...` puis `go test -tags gamefiles ./hinavmesh/ ./himodule/` depuis le
  clone ; revue adversariale Sonnet (une ronde) ; commit poussé, sha noté au Journal.

### Étape 2 — Brique `halo-infinite-map-images` : cuisson native tenue par le maillage, plan des murs (moyen)

- [x] 2.1 `himap/navmesh_commun.go` : `OptionsNavmesh { Maillage *hinavmesh.Maillage; Rogne bool;
  Marge float64; Tolerance float64; NiveauHaut bool }` et `appliqueNavmesh(ctx, r, bilan, OptionsNavmesh)` ;
  `CuitCarteForge` migré dessus (ses cinq champs deviennent ce bloc : changement d'API, v1.1.0),
  `OptionsCuisson` (natif) reçoit `Navmesh OptionsNavmesh` et l'applique après `PeupleDepuisModule`,
  avant les rognages existants. Garde-rail : test grep interdisant un second appel direct à
  `ArmeReferenceDepuisNavmesh` hors du helper.
- [x] 2.2 `himap/maillage_natif.go` : `ChargeMaillageNatif(ctx, racineDeploy, module) (*hinavmesh.Maillage,
  BilanMaillage, error)` — ouvre le module `any` (`LevelsDir("any")`), D-2, `DecodeNatif` ; bilan =
  secondaires ignorés, absence = erreur typée `ErrSansMaillage` que la cuisson consigne en dégradation.
- [x] 2.3 `himap/plan_murs.go` : `PlanDesMurs(m *hinavmesh.Maillage) PlanMurs` — arêtes de bord (vue une
  fois, clé ordonnée), sommets en mètres monde, compte des faces et sommets ; déterministe (ordre stable).
  Test unitaire sur un maillage de trois faces (deux arêtes partagées, six de bord).
- [x] 2.4 `himap/fond_png.go` : style `StylePlan` (remplissage plat, bord des faces) pour la planche et le
  diagnostic seulement ; la production garde `StyleJeu`.
- [x] 2.5 Tests `gamefiles` : Bazaar cuit avec maillage — `bilan.NiveauDeJeu` inchangé à 0,2 m près, part
  rognée mesurée et figée, 17 ancres sur 17 au sol ; Live Fire — dégradation présente, rendu égal au
  témoin (empreinte).
- [x] 2.6 README + `example_test.go` : exemple natif avec maillage.
- Gate : `go vet ./... && go test ./...`, `go test -tags gamefiles ./himap/ -run 'Navmesh|Plan|Natif'` ;
  revue adversariale Sonnet ; commit poussé, sha noté.

### Étape 3 — LevelUp : constructeur, sidecar, recuisson (moyen)

- [x] 3.1 `go.mod` de `apps/go-api` : pseudo-versions des deux briques (sha de 1 et 2) ; `make go-api-lint`.
- [x] 3.2 `cmd/mapfond-build/cuisson.go` : cartes natives — `ChargeMaillageNatif`, options navmesh par défaut
  (rogne, marge 1,5 m, tolérance 0, niveau bas), surcharges lues dans `map_fond_reglages.json` (clés
  `navmesh_marge_m`, `navmesh_tolerance_m`, `navmesh_niveau_haut`, `navmesh: false` pour désactiver,
  chaque entrée avec raison et date comme les autres : `TestReglagesFondJustifies`). Fonction de la
  chaîne native déjà longue : extraire `optionsNavmeshNatif(e, cle)`.
- [x] 3.3 `domain/replaydoc/map.go` : `MapBackgroundPlan { Source string; Vertices, Faces int; Edges
  [][4]float64 }`, champ `Plan *MapBackgroundPlan json:"plan,omitempty"` ; écrit par `mapfond-build` pour
  natifs (maillage chargé) et Forge (blob du dépôt) ; `openapi.yaml` + `make generate-types`.
- [x] 3.4 `service/replay_map_background.go` : rien à changer si le sidecar est relu tel quel ; test
  `replay_map_background_traversee_test.go` étendu : sidecar avec plan servi, sidecar sans plan servi.
- [x] 3.5 Recuisson des 20 natifs + des Forge validées (plan seul ajouté) : `go run ./cmd/mapfond-build
  -natives -forge`, puis `cmd/mapfond-webp -convertir` ; planche « après » ; témoins comparés : les Forge
  ont une image identique à l'octet (seul le sidecar change) ; les natifs changent (attendu), part rognée
  par carte écrite au Journal.
- [x] 3.6 `docs/COMMANDS.md` (EN + FR) : options navmesh de `mapfond-build`.
- Gate : `go test ./internal/domain/replaydoc/ ./internal/service/ -run 'MapBackground'` ; `go test
  ./internal/archlint -count=1 -timeout 40m` ; `make check-types` ; planche « après » remise au user avec
  la liste des 20 cartes ; **gate visuel du user** carte par carte (il nomme les témoins). Les cartes
  refusées repassent par un réglage dans `map_fond_reglages.json` (raison datée), pas par un patch.

### Étape 4 — Web : calque « Plan des murs » et rotation (moyen)

- [x] 4.1 Extraire de `ReplayCanvas.tsx` le dessin du fond dans `layers/mapBackgroundDraw.ts` (il est au
  plafond ; aucun comportement ne change ; test de non-régression par capture canvas existante).
- [x] 4.2 `layers/wallPlanLayer.ts` : dessin des arêtes du sidecar en tokens d'encre (`canvasInk`), trait
  1 px CSS, calque statique recuit au changement de cadrage comme les quatre existants
  (`useReplayView.ts:174-199`). Test : arêtes projetées aux bons pixels pour un calage connu ; calque
  absent sans plan.
- [x] 4.3 `settings/useReplaySettings.ts` + `ReplaySettingsLayers.tsx` : interrupteur `wallPlan`, défaut
  vrai, masqué quand le fond n'a pas de plan ; `i18n/fr.ts` et `en.ts` ; `replayPreferences.ts` persiste.
- [x] 4.4 Rotation : `model/orientation.ts` (`quarterTurns: 0|1|2|3`, `orientWorld(x,y,turns,bounds)`,
  inverse pour le survol) ; `useReplayView.ts` applique l'orientation aux bornes et à `canvasView` ;
  `mapBackgroundDraw.ts` dessine l'image sous `ctx.rotate` ; commentaires « affine sans rotation » de
  `mapBackground.ts:14,74` et `ReplayCanvas.tsx:488` corrigés (règle 17). Réglage par carte dans
  `replayPreferences` (clé `rotation.<mapId>`). Commande dans le tiroir (ou le panneau Vue, D-11) :
  « Orientation » (fr) / « Orientation » (en), quatre positions. Tests : survol inversé exact aux quatre
  positions ; export vidéo au cadre 16:9 tourné ; heatmap du rejeu suit (même projection).
- [x] 4.5 Garde-rails existants verts : `canvasInk.guard.test.ts`, `deltaLayersContract.guard.test.ts`,
  `escapeLayers.guard.test.ts` ; fichiers ≤ 500 lignes.
- Gate : `npx vitest run src/features/match-replay src/lib/replay` ; `make check-types` ; `npx knip` ;
  revue adversariale Sonnet ; vérification à l'écran par le user sur Bazaar, Isolation (Forge), Live Fire
  (sans plan), aux quatre orientations — artefact `verifs` (`reference_artefact_verifs_user`).

### Étape 5 — Clôture (rapide)

- [x] 5.1 Tags `v1.1.0` des deux briques **posés par le user** ; `go.mod` passe des pseudo-versions aux
  tags ; `go mod tidy` ; CI verte.
- [x] 5.2 ADR : amendement de 0041/0043 (EN) inutile (pas de nouvelle brique) ; **nouvel ADR 0048**
  « Fonds de carte tenus par le maillage de navigation » (EN) : sources natif et Forge, vecteur dans le
  sidecar, rotation dans la projection, Live Fire en dégradation.
- [x] 5.3 `docs/FOUNDATIONS_GUIDE.md` (EN + FR) : section fonds de carte mise à jour.
- [x] 5.4 `.ai/thought_log.md` (entrée de clôture), `.ai/REGISTRE_REPORTS.md` (reports de § 8),
  `.ai/BACKLOG.md` (ligne du 2026-11-13 close), ce plan archivé sous `.ai/archive/V8/`.
- [x] 5.5 `make gate-push` ; PR vers `release/v8.2.0` (titre Conventional Commits) ; `main` reste un geste
  du user ; `disk-hygiene.ps1 -Lot navmesh -Apply`.
- Gate : CI verte, PR fusionnée dans la release.

## 6. Grille `plan-review` (points pertinents)

- Couches : décodage et cuisson dans les briques (sans import LevelUp, ADR 0041) ; orchestration de
  cuisson dans `cmd/mapfond-build` (outil hors ligne, pas un service) ; type de sidecar dans
  `domain/replaydoc` ; le service et le handler ne changent pas ; aucun accès DuckDB.
- Multi-titre : chemins par `PathResolver` (`MapBackgroundDir`) ; la cuisson est un outil Halo Infinite
  déclaré tel (`himap`), pas une capability nouvelle ; le web ne branche rien sur un slug.
- Tests : unitaires (synthétiques) dans les briques, `gamefiles` sur le jeu installé, traversée du service,
  vitest par calque, garde-rails existants.
- Logging : `slog.*Context` dans les briques et l'outil ; dégradations comptées dans le bilan, jamais
  avalées.
- Web : i18n fr + en dans les dictionnaires typés, tokens d'encre seulement, pas de route nouvelle,
  query key inchangée (le sidecar est déjà chargé).
- Exécutabilité : périmètre fermé par item, gates en commandes, statuts, ordre, Découvertes, reprise § 7.

## 7. Journal et reprise de session

Reprendre ici : lire ce journal, `git log --oneline -5` sur les trois branches, puis l'étape ouverte.

| Date | Étape | Fait | Gate | Sha |
|---|---|---|---|---|
| 2026-11-13 | 0 | Clones `../halo-infinite-files-reader` (`feat/navmesh-natif`, 01ec0fd) et `../halo-infinite-map-images` (`feat/cuisson-navmesh-natif`, 95547a3) ; worktree `../LevelUp-wt-navmesh` (`feat/fonds-navmesh-natif`, 0ecb75a827) ; 20 webp + 20 sidecars copiés dans `.ai/V8.3/annexes_fonds_navmesh/avant/` (dossier ignoré par git) ; planche avant depuis les PNG du backup du 2026-09-10 (même cuisson du 26/08) | vet + tests verts dans les deux clones | — |
| 2026-11-13 | 1 | `halo-infinite-files-reader` : `DecodeNatif` (erreurs typées, régions par signature TAG0), `RessourcesNavmesh(ctx) (ressources, illisibles, err)` (préfixe `ucsh` + marque, tri stable), factorisation `maillagesDeRegion` partagée avec `Decode` ; 8 tests synthétiques (masque de drapeaux testé avec 0x80000000 : 0x40000000 est le bit feuille) ; `gamefiles` : Bazaar 2 312 / 670, 29 cartes, 2 sans, secondaires et illisibles épinglés par carte (17 sur 12 cartes) ; README, commentaire de couverture corrigé ; revue adversariale : 5 corrections appliquées | vet + tests + `gamefiles` verts, rejoués par le pilote | b8e5446, 0a8f7b8 (poussé) |
| 2026-11-13 | 2 | `halo-infinite-map-images` : `OptionsNavmesh` + `appliqueNavmesh` (deux étapes : référence avant projection, rognage après la frontière de mort côté natif ; ordre Forge historique conservé), `ChargeMaillageNatif` + `BilanMaillage`, `PlanDesMurs(m) (PlanMurs, error)`, `StylePlan` (silhouette seulement : `Rendu` n'a pas de contour), `BilanCuisson.CellulesHorsNavmesh`, variable globale `NiveauHautNavmesh` supprimée (paramètre), garde-rail grep ; Bazaar : 17/17 ancres au sol, part rognée 56,47 %, 1 455 255 cellules hors maillage, 1 174 arêtes de plan ; revue adversariale : 6 corrections | vet + `-race` + `gamefiles` verts, rejoués par le pilote | c51b771, 5de7c19, 7feaf08, 17bbd5b (poussé) |
| 2026-11-13 | 3 | LevelUp `feat/fonds-navmesh-natif` : briques en pseudo-versions (files-reader a2397d6, map-images 196f8de), `MapBackgroundPlan` + `plan` au sidecar (OpenAPI, types web), paquet `internal/mapfondsidecar` (le type du décodeur de film ignore `plan` : lecture/écriture du sidecar complet par un seul paquet), `cuisson_native.go` / `cuisson_forge.go` extraits, `rogneAuNavmesh` en `*bool` (défaut vrai natif, faux Forge), `sansNavmesh`, drapeau `-plan-seul`, archlint corrigé (entrée périmée retirée, cible Makefile) ; dépôt hors ligne reconstitué dans le worktree ; 20 natifs recuits (Live Fire en dégradation, sans plan), 51 Forge recuites image identique à l'octet, 43 Forge plan seul : 111 sidecars sur 114 avec plan (manquent Live Fire et les deux Thunderhead sans blob) ; planches avant/après et expériences de marge ; **gate visuel du user : 20 cartes validées le 2026-11-13** | tests ciblés, lint 0, archlint ok (259 s), check-types, openapi-check verts ; rejoués par le pilote | 5f96239b30 … 3ae8af5750 (9 commits, non poussés) |
| 2026-11-13 | 4 + 4 bis | Web : extraction `mapBackgroundDraw.ts` + drapeaux de terrain (sans changement) ; `lib/replay` : `orientation` (sens horaire, D-13), `wallPlan`, `strokeWallPlan`, `useZoomResetOnTurns` partagés par trois vues ; rejeu : calque `plan-murs` entre fond et zones, interrupteur « Plan des murs » (défaut vrai, visible si plan) et « Orientation » (0/90/180/270, par carte, clé `replay-orientation.<mapId>`) dans le panneau Vue ; `CanvasView.orient` lu par `projectTo` (survol, export, chaleur, Suivre, caps des pions/véhicules/armes) ; Tactique (`TacticalPlanCard`) et Occupation du terrain (`MatchPositionsHeatmap`, `_positionsPaint.ts`) lisent les mêmes préférences ; revue adversariale : 6 corrections (code mort, plan cuit éteint, logger, tests de câblage, Suivre, sens) ; allowlist inter-features : 2 entrées datées. Pas de contrôle à l'écran (agents sans navigateur) : artefact `verifs` | vitest 5 791 verts, tsc, eslint, knip, lint:colors verts, rejoués par le pilote | d54cf531f2 … 0a1bbd3b5f (9 commits) |
| 2026-11-13 | 5 | Tags `v1.1.0` posés (files-reader a7b8c01, map-images 436cffe ; PR #2 et #1 fusionnées, délégation du user), LevelUp sur les tags ; ADR 0048, amendement 0043, guide EN/FR ; filet local : lint Go, web, `replay-equiv` 20/20 verts, baseline coupée par les tests d'autres sessions (300 s) ; CI : trois correctifs (harnais raster, parité du sidecar, Cliffhanger `sansNavmesh` + plan conservé, fond d'août restauré à l'octet) ; PR #205 fusionnée dans `release/v8.2.0` | CI 32 verts | 5417b2ba3e |

## 8. Découvertes (hors périmètre, à ne pas traiter ici)

- Maillages secondaires (Chasm 5, Launch Site 5, Cliffhanger 1) : nature à qualifier (volumes de
  véhicules ? étages détachés ?).
- Réglages manuels de `map_fond_reglages.json` rendus inutiles par le maillage : liste produite à 3.5,
  purge sur décision du user.
- `MatchPositionsHeatmap` (vue match) : rotation non portée. REPRIS en 4.7 le 2026-11-13 (demande du user).
- Positions de force : le sol natif exact est disponible ; relancer l'audit du 2026-10-27 § 4.
- Lien `pfnd` -> ressource de maillage : non décodé, D-2 balaye par signature.
- Ressources candidates illisibles (étape 1, `RessourcesNavmesh`) : sur 14 des 29 modules natifs, une à quatre
  ressources sans groupe de plus de 10 Ko ne s'extraient pas (`bloc N hors borne`, données au-delà de la fin
  du `.module`, drapeaux 0x6402 que `UseHd1` ne couvre pas ; ex. academy_weapon_drills 2, ctf_breaker 4,
  sgh_streets 2). Journalisées (`slog.Warn`) et rendues par `ErrRessourcesIllisibles`, pas avalées. Nature
  inconnue : lecteur de flux compagnon à décoder, piste séparée.
- ADR 0043 dit que les dépôts dédiés sont « produits depuis `libs/`, jamais édités sur place » ; depuis la
  bascule de la v8.1.2 (plan D-14), `libs/` n'existe plus sur la release et les briques se modifient dans
  leurs dépôts. Phrase à amender dans l'ADR à l'étape 5.2 (relevé par la revue de l'étape 1).
- Cause des ressources illisibles : branche « blocs » de `himodule.Extract` (`module.go:352`), drapeaux
  0x6402 ; non traitée ici.
- `himap/cuisson_forge.go` (`combleTrousForge`) utilise `MargeNavmesh` en dur et ignore la marge réglée par
  carte, alors que son commentaire dit « la marge reprend celle du rognage » (relevé par la revue de l'étape
  2, existant avant ce chantier). Non traité.
- Sur Bazaar, la substitution par la référence du maillage n'est pas exercée (`couverte=false`,
  `substituees=0`) : le gain natif démontré est le rognage. À observer sur les cartes à toits (étape 3.5).
- Quatre fonds Forge (Lattice, Thunderhead, Warehouse, Threshold) : le code d'avant ce chantier produit déjà une
  image différente de celle publiée (dérive antérieure, données ou outil). Image publiée conservée, plan ajouté.
- Catalyst : quatre ancres superposées au centre (0 ; 0 ; z 27,3) sont sur l'étage haut du maillage ; le réglage
  `maillageNiveauHaut: true` les rendrait (26/29 au lieu de 22/29). Fond validé tel quel par le user le
  2026-11-13 ; réglage possible plus tard.
- Aquarius : une ancre centrale (0 ; 7,1 ; z 1,1) perd la surface à z 3,34 dessinée au-dessus ; `combleTrous`
  non essayé. Validé tel quel.
- Dépôt hors ligne `.ai/re_dump` : absent du poste, reconstitué dans le worktree (92 blobs par `mapnav-fetch`,
  51 variantes depuis `data/cache/mvar`, 20 fichiers de lien vides reconstitués depuis les `mapNames`) ; 43
  Forge sans variante au cache. Dossier ignoré par git : à sauvegarder sur la clé.
- Cliffhanger : 46,9 % de matière rognée, rochers non praticables en trous ; validé tel quel par le user.

### Étape 4 bis — Page Tactique et carte de chaleur de la vue du match (ajout du user, 2026-11-13)

Demande du user pendant l'étape 4 : « pour la vue 2D faut pas oublier la page Tactique ». Décision D-12 :
le plan des murs et l'orientation par carte s'appliquent à TOUTES les vues 2D qui posent le fond de carte,
avec la MÊME préférence d'orientation par carte (`replay-orientation.<mapId>`) : une carte tournée dans le
rejeu est tournée partout.

- [x] 4.6 Page Tactique : plan des murs dessiné sur le plan (`features/tactical/TacticalPlanFond.tsx` +
  `TacticalPlanCalque.tsx`, cadrage `useCadrageDuPlan.ts` / `plan.logic.ts`), sous les calques de morts et
  de coéquipiers ; interrupteur partagé avec le rejeu (`replay-show-wall-plan`) ; orientation par carte lue
  de la même clé, appliquée au fond (img tournée) et à la projection des points.
- [x] 4.7 Carte de chaleur de la vue du match (`features/match-view/MatchPositionsHeatmap.tsx`, `_positionsHeat.ts`
  `mapFrame`) : même orientation par carte, plan des murs par-dessus le fond, sous la chaleur ; pas
  d'interrupteur propre (suit celui du rejeu).
- [x] 4.8 Tests : projection tournée sur un point connu dans les deux vues ; plan absent = rien ; garde-rails
  verts.
- Gate : `npx vitest run src/features/tactical src/features/match-view src/features/match-replay` ;
  `make check-types` ; `npx knip` ; vérification à l'écran par le user (artefact `verifs`).
| D-13 | Sens de l'orientation : HORAIRE (à 90°, le nord passe à droite), une seule convention pour le fond, les arêtes, la chaleur et les caps | Lecture naturelle d'une rotation « 90° » ; tranché le 2026-11-13 après la revue de l'étape 4 (livrée en antihoraire) |
- Vignettes de la colonne des cartes de la Tactique (`TacticalMapTile`) : ne tournent pas avec l'orientation
  (relevé le 2026-11-13, non demandé).
- Filet local du 2026-11-13 (étape 5.5) : lint Go, typecheck et lint web, harnais `replay-equiv` (20 identiques)
  verts ; la baseline des tests Go bute deux fois de suite sur le délai de 300 s par paquet (`internal/sync`,
  `platform/duckdb`, `ops` puis `archlint`), la machine étant chargée par les tests d'autres sessions
  (`sync.test.exe`, `service.test.exe`, `scheduler.test.exe`… visibles pendant le run). Aucun test en échec,
  seulement des paquets coupés. La CI reste le gate d'autorité pour la baseline.
- CI de la PR #205 (2026-11-13) : (a) harnais de rasterisation du rejeu : portée `orientation` injectée
  (2 imports de valeur sur `replayView.ts`) ; (b) parité stocké/servi : le sidecar complet
  (`mapfondsidecar.MapBackground`) devient la seule source stockée du fond, projection avec `Plan`,
  garde-rail qui aplatit les champs embarqués ; (c) Cliffhanger : l'oracle de découpage des zones nommées
  (`mapdecoupe`, IoU médian 0,775 < 0,85, marges 3/5/8 m à 0,746/0,701/0,788) refuse le fond rogné au
  maillage : retour à la cuisson d'août par `sansNavmesh`, plan des murs conservé. Le user avait validé
  le fond rogné sur planche ; l'oracle prime (seuil fixé avant mesure).
- Cliffhanger sans maillage : la recuisson actuelle (himap v1.1.0) ne reproduit pas le fond d'août (2 299 × 2 097 en
  échelle automatique, 1 250 × 1 140 à 0,092 m/px, contre 1 633 × 1 627 en août) ; le fond d'août a été restauré
  à l'octet depuis `0ecb75a827` et son plan rattaché. Une recuisson ultérieure de Cliffhanger ne reproduira pas ce
  fond : réglage d'échelle et cadrage à requalifier avant toute recuisson de cette carte.
