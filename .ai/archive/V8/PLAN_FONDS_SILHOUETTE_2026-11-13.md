# Plan : fonds au maillage, lot 2 : silhouette par carte, plans Forge décalés, style plan (2026-11-13)

Statut : **CLOS le 2026-11-14** (PR #208 fusionnée dans `release/v8.2.0`, e83b4f4c2c ; briques files-reader v1.2.0, map-images v1.3.0). Ouvert le 2026-11-13 (levée de l'incident confirmée par levelup-dd le 2026-11-13 : archive restaurée, 1 614 films ; règle : aucune suppression récursive pouvant traverser les jonctions du worktree, caches par `disk-hygiene.ps1 -Lot`). Exécution sous le skill `plan-execution`. Suite du plan clos

`.ai/archive/V8/PLAN_FONDS_NAVMESH_NATIF_2026-11-13.md` (PR #205).

## 1. Objectif et critère de succès

Le user a validé sur la planche des 111 cartes (artefact `NWhq6zEKK4XZiREqCNLViJ`, magasin `avis`,
versionné dans `data/titles/halo_infinite/reference/map_fond_avis_2026-11-13.json` sur
`feat/fonds-silhouette`, c61baa7a33) : règle de rognage par défaut = **silhouette extérieure** du
maillage (composante principale, trous intérieurs conservés), vue **plan seul au bord exact** du
maillage (« parfait »). Il a repéré qu'un masque dilaté d'un mètre débordait des arêtes : toute marge
est nommée, le plan n'en a aucune.

Réussi quand :
- les 111 fonds sont recuits selon le fichier d'avis (silhouette par défaut, Yuletide et Recharge
  gardent le publié, les cartes « aucune » et commentées traitées par le diagnostic du § 3) ;
- le plan des murs du sidecar ne porte plus que la composante principale du maillage (plus de bordure
  de canevas Forge) ;
- le rejeu propose le style « plan » (silhouette exacte + murs, sans fond) comme option de fond ;
- planche avant/après remise au user, validation carte par carte dans l'artefact.

Hors périmètre : rotation, Tactique (déjà livrés) ; recuisson des rejeux.

## 2. Décisions tranchées

| # | Décision | Raison |
|---|---|---|
| D-1 | `OptionsNavmesh.Mode` : `faces` (actuel) ou `silhouette` (remplissage par flood depuis le bord de la grille sur le masque des faces : les trous intérieurs restent) ; `Marge` nommée, défaut natif 0,5 m, Forge 0,5 m aussi (était 3 m) | Validation du user ; marge visible |
| D-2 | Composante principale : `hinavmesh.ComposantePrincipale(m) []bool` (union des sommets, plus grande aire) ; `PlanDesMurs` et le masque de rognage l'utilisent ; les autres composantes comptées dans le bilan | Dalles du canevas Forge hors arène |
| D-3 | Le fichier d'avis est lu par `mapfond-build` (`-avis <chemin>`) : `etat` = `silhouette` / `faces` / `publie` (pas de recuisson, plan seul) / `plan` ; `aucune` ou vide = carte laissée telle quelle ET listée dans le rapport | Une seule source des choix du user |
| D-4 | Style `StylePlan` = silhouette exacte (marge 0) en aplat + arêtes ; publié comme fichier `<cle>.plan.webp` à côté du fond, référencé par le sidecar (`planImage`) ; le web propose « Fond : rendu / plan » dans le panneau Vue, préférence globale | Le user a validé cette vue comme chez Den |
| D-5 | Blobs Forge décalés : diagnostic AVANT toute recuisson de ces cartes (repère du blob vs calage du fond ; hypothèse : variante `.mvar` publiée avec un autre canevas ou une translation) ; sans cause trouvée, la carte garde son publié et le plan n'est PAS écrit | Jamais un patch par carte |

## 3. Étapes

Statuts `[x]` / `[~]` / `[!]` ; gate par étape ; ordre strict ; découvertes en § 5.

### Étape 0 — Pré-requis
- [x] 0.1 Confirmation écrite de levelup-dd que les écritures lourdes sont permises (consignée ici).
- [x] 0.2 Branches : brique `halo-infinite-files-reader` `feat/composante-principale`, brique
  `halo-infinite-map-images` `feat/silhouette`, LevelUp `feat/fonds-silhouette` (existe, depuis
  `release/v8.2.0`, porte le fichier d'avis). Worktree `../LevelUp-wt-navmesh` basculé sur cette branche.
- Gate : branches créées, `go build` vert.

### Étape 1 — Brique lecteur : composante principale (rapide)
- [x] 1.1 `hinavmesh.ComposantePrincipale(m *Maillage) (garde []bool, autres int, aire float64)`.
- [x] 1.2 Tests : maillage synthétique à deux composantes ; `gamefiles` : Isolation (blob) et Chasm
  (natif, 5 secondaires) : la composante principale couvre l'emprise des ancres.
- Gate : `go vet ./... && go test ./...` ; `LEVELUP_HALO_DEPLOY=... go test -tags gamefiles ./hinavmesh/`.
  Tag `v1.2.0` posé (délégation du 13/11).

### Étape 2 — Brique images : mode silhouette, composante, style plan (moyen)
- [x] 2.1 `OptionsNavmesh.Mode` (D-1) et `Marge` documentée ; `appliqueNavmesh` rasterise les faces de
  la composante principale, dilate de `Marge`, et en mode silhouette remplit les trous (flood depuis
  le bord) ; `BilanCuisson` : mode, marge, cellules retirées, composantes ignorées.
- [x] 2.2 `PlanDesMurs(m, garde)` : arêtes de bord des seules faces gardées.
- [x] 2.3 `StylePlan` : aplat de la silhouette EXACTE (marge 0) + arêtes ; `FondPNG` l'accepte avec un
  `Rendu` ou directement avec un maillage et un calage (`PlanPNG(m, calage)`).
- [x] 2.4 Tests unitaires (silhouette d'un anneau : le trou reste ; composante) ; `gamefiles` Bazaar
  (silhouette 0,5 m : part rognée épinglée, 17/17 ancres) ; README.
- Gate : vet, `-race`, `gamefiles` ; revue adversariale Sonnet ; `go.mod` sur files-reader v1.2.0 ;
  tag `v1.2.0`.

### Étape 3 — Diagnostic des Forge décalées (moyen, lecture seule)
- [x] 3.1 Pour les 14 cartes (Rat's Nest, Diminished, Shiro, Shogun, Houseki, Detachment, Alpha Site,
  Ardent Prayer, Smallhalla, Showdown Arena, Flood Gulch, Dredge, Urban Raid, Vallaheim) : emprise du
  blob vs calage du fond, ancres d'objectif dans le maillage (oracle existant
  `TestOracleAncresDansLeMaillage`), variante `.mvar` et canevas ; cause par carte écrite au Journal.
- [x] 3.2 Recharge : pourquoi le carré nord-ouest (socle d'arme) n'est pas dans le maillage natif
  (secondaire ? hors composante ?) ; décision : garder le publié ou corriger.
- Gate : tableau des causes ; règle générale si elle existe (jamais un patch par carte), sinon ces
  cartes restent `publie` avec raison datée dans `map_fond_reglages.json`.

### Étape 4 — LevelUp : constructeur, avis, recuisson, planche (moyen)
- [x] 4.1 `go.mod` sur les tags v1.2.0 ; `mapfond-build -avis` (D-3) ; mode et marge dans
  `map_fond_reglages.json` (`navmeshMode`, `margeNavmesh`) ; `-plan-seul` écrit aussi `<cle>.plan.webp`.
- [x] 4.2 Sidecar : `planImage` (nom de fichier) ; OpenAPI, types web, `mapfondsidecar`, parité.
- [x] 4.3 Recuisson des 111 selon le fichier d'avis ; Cliffhanger : règle silhouette (validée) mais
  l'oracle `mapdecoupe` doit rester ≥ 0,85 : mesurer ; s'il refuse, Cliffhanger reste `publie` et
  c'est écrit.
- [x] 4.4 Planche avant/après (artefact, cases par carte) ; gate visuel du user.
- Gate : tests ciblés, archlint, lint, check-types ; CI de la PR.

### Étape 5 — Web : style plan (moyen)
- [x] 5.1 Panneau Vue : « Fond : rendu / plan » (fr et en), préférence globale ; le calque fond charge
  `planImage` quand il existe ; calque « Plan des murs » masqué en mode plan (les murs y sont déjà).
- [x] 5.2 Tests, garde-rails, knip.
- Gate : vitest, tsc, eslint ; vérification à l'écran (artefact `verifs`).

### Étape 6 — Clôture
- [x] 6.1 PR vers `release/v8.2.0`, CI verte, fusion ; ADR 0048 amendé (mode silhouette, composante,
  style plan) ; guide EN/FR ; journal, registre, backlog, archive du plan ; `disk-hygiene -Lot navmesh`.

## 4. Journal

| Date | Étape | Fait | Gate | Sha |
|---|---|---|---|---|
| | | | | |

## 5. Découvertes

## 6. Amendement du 2026-11-13 (après l'étape 3)

| # | Décision | Raison |
|---|---|---|
| D-6 | **Composantes retenues** (remplace « composante principale » de D-2 pour le rognage ET le plan) : une composante du maillage est gardée si elle porte au moins une ancre d'objectif avec appui vertical (abs(dz) ≤ 2 m), OU si elle fait au moins 4 m² et qu'au moins 90 % de son aire tombe sur la matière du fond (la matière rendue dans le cadre, avant le rognage au maillage). Repli sans ancre ni matière : plus grande aire. Le plan des murs est calculé sur ces composantes et rogné au cadre du fond. | Diagnostic de l'étape 3 : sur 15 Forge (dont 12 des 14 signalées, plus Domicile, Yuletide, Security Zone) la plus grande composante est le SOL DU CANEVAS, pas l'arène ; aucune carte n'est décalée (ancres dans une face, dz médian ≤ 0,4 m). Le critère « 90 % dans la matière » reprend les îlots jouables sans objectif (Recharge : le carré du socle d'arme est la composante #7, 6,3 m², 100 % dans le fond ; Ardent Prayer, Showdown). L'appui vertical écarte le sol (Alpha Site : ancres à 5 à 6 m au-dessus ; Vallaheim : 26 m). |
| D-7 | `hinavmesh` expose TOUTES les composantes (`Composantes(m) []Composante`, aire, emprise, faces), la sélection vit dans `himap` (elle a besoin des ancres et de la matière du rendu) | Couches : le lecteur ne connaît ni ancres ni rendu |
| D-8 | Flood Gulch et Urban Raid ne relèvent pas des composantes (fond plus large que le maillage ; arène qui dépasse le cadre) : traités par le rognage silhouette ordinaire, le plan rogné au cadre ; résultat à juger sur la planche | Mesuré |

Journal :

| 2026-11-13 | 1 | files-reader : `ComposantePrincipale`, `TrianglesDe` ; Isolation 2 240/2 348 faces, 16 écartées (772,6 m²), emprise principale X [-49,8 ; -0,6] Y [-37,9 ; -5,6] ; Bazaar 611/670, 16 îlots (5 % de l'aire) | vet, tests, `gamefiles` verts (rejoués) | 6ba4743, PR #3 (à étendre : D-7) |
| 2026-11-13 | 3 | Diagnostic lecture seule des 14 Forge + Recharge : aucun décalage de repère (ancres dans une face, dz médian 0 à 0,38 m) ; cause = composante principale = sol du canevas (12 cartes + Domicile, Yuletide, Security Zone) ; Dredge et Urban Raid : plan hors cadre (879/2 527 et 366/949 arêtes) ; Recharge : carré = composante #7 (6,3 m², socle `power`), et deux ressources contiguës illisibles (1111, 1113) ; règle D-6 mesurée : IoU ≥ 0,88 sur les cartes validées, Vallaheim 94 → 73 % couvert (sol retiré, voulu) | tableau par carte au rapport de l'agent ; images `scratchpad/diag_forge/img/` | — |
| 2026-11-13 | 2 | map-images `feat/silhouette` (bae6950, f02ee23, 21d605e ; PR #2) : `ComposantesRetenues` (D-6), `Mode` faces/silhouette, marge 0,5 m par défaut (les DEUX chaînes, était 3 m), `PlanDesMurs(m, garde)`, `PlanRogneAuCadre`, `PlanPNG` ; Bazaar silhouette 0,5 m : 17/17 ancres, 62,18 % rognés, 5 composantes gardées sur 17 (4 ancre, 1 matière). Revue adversariale : rien de bloquant ; corrections en cours (doc de la marge, `CombleAuMaillage` sur la marge résolue, couverture finale même sans rognage, 4 tests de câblage, commentaire périmé) | vet, `-race`, `gamefiles` verts (rejoués) | en cours |

Découvertes (étape 2) :
- Défaut de marge 3 m → 0,5 m : dans LevelUp, 72 cartes ont `rogneAuNavmesh: true` dont 70 sans `margeNavmesh` ; elles changeront à la recuisson (voulu : règle validée par le user), et le commentaire « 0 = 3 m » de `cmd/mapfond-build/reglages.go:89` est à corriger à l'étape 4.
- Une composante coupée par le cadre à plus de 10 % de son aire et sans ancre est écartée (critère matière évalué hors grille = non) ; effet à observer sur la planche.
| 2026-11-13 | 2 (clos) | map-images : corrections de revue d1e1c25 (marge des deux chaînes documentée, `CombleAuMaillage` sur la marge résolue, couverture finale posée même sans rognage, 4 tests de câblage vérifiés par mutation) ; PR #2 fusionnée, tag `v1.2.0` (b7602e3) ; files-reader PR #3 fusionnée, tag `v1.2.0` (99b8060) | vet, `-race`, `gamefiles` verts (rejoués) | v1.2.0 × 2 |
| 2026-11-13 | 4 (lancée) | Décision du pilote (amende D-3) : les cartes « aucune » ou commentées sont RECUITES en silhouette et marquées « à revoir » (leur défaut venait de la sélection de composante, corrigée en v1.2.0) ; « publie » sautées ; image plan `<cle>.plan.webp` + `planImage` au sidecar | — | — |
| 2026-11-14 | 4 | LevelUp `feat/fonds-silhouette` : briques v1.2.0, `mapfond-build -avis`, `navmeshMode`, `planImage` + route `?variante=plan`, `-plan-seul -rogne-publie` ; 66 recuites, 42 Forge sans variante rognées sur le fond publié, Cliffhanger fond d'août (oracle 0,675 < 0,85), Yuletide et Recharge fond publié + plan ; 111 images plan ; pertes d'ancres : Megapolis 12→9, Pharaoh 9→8, Absolution 21→19 ; plan rejoué ≠ cuisson sur 5 cartes (Shogun surtout) ; planche de validation lot 2 : artefact 4Kv1566De6ViWN5Jed4VFa (collection `avis`, vide et lisible au 2026-11-14) | gates ciblés, mapdecoupe, archlint, lint, check-types, openapi-check verts | 6a64bee5e9 … 0f5b46f53b |
| 2026-11-14 | 4.4 | Gate visuel du user sur la planche lot 2 : 111 nouveaux fonds validés ; à reprendre : Ardent Prayer, Urban Raid ; Vallaheim Firefight à ignorer (mode non supporté) ; règle générale (commentaire Deadlock) : couper le rendu à la silhouette du plan pour TOUT ce qui dépasse, eau comprise ; Catalyst : un bout manque au sud du fond par rapport au plan ; avis versionnés `map_fond_avis_lot2_2026-11-14.json` | gate visuel | — |

### Étape 4 bis — Règle du user : le plan est le masque (ajout du 2026-11-14)
- [~] 4b.1 (voir 4t.2) Rognage FINAL à la silhouette du plan (marge 0,5 m), appliqué en dernier, après l'eau et tout autre calque peint (Deadlock : étendue d'eau hors plan).
- [~] 4b.2 (voir 4t.3) Matière manquante dans la silhouette (Catalyst sud) : mesurer d'où vient le trou (géométrie absente du rendu, écrêtage, tranche) avant toute correction ; option : aplat du plan là où la silhouette n'a pas de matière.
- [x] 4b.3 Ardent Prayer et Urban Raid : comprendre (îlots multiples ; arène qui dépasse le cadre) ; Vallaheim : exclu (Firefight).

## 7. Amendement du 2026-11-14 : plan à hauteurs (validé par le user)

| # | Décision | Raison |
|---|---|---|
| D-9 | Le style « plan » devient le **plan à hauteurs** : teinte continue par altitude de face (rampe en six pas, du bleu ardoise `#60708C` au jaune paille `#F6ECB0` : `#60708C #7E8C96 #9EA6A0 #BEBEA0 #DCD6A4 #F6ECB0`, échelle du 2e au 98e centile des altitudes de la carte), ombre portée douce des niveaux plus hauts (décalage 0,7 m vers le sud-est, seuil 1,2 m, facteur 0,72), arêtes de bord en encre, arêtes recouvertes par un niveau plus haut (plus de 1,5 m) en pointillé gris. | Validé par le user le 2026-11-14 (« ultra propre ») avec la contrainte daltoniens : rampe vérifiée par simulation Machado 2009 (protan, deutan, tritan) : clarté OKLab monotone 54 → 94, écart entre pas voisins 6,7 à 9,6 (×100) dans les trois cas ; la rampe précédente montait à 3,8 en haut de rampe, refusée. |
| D-10 | Étages superposés : une seule image (pointillés + ombre) ; le sélecteur d'étage reste une option future, non planifiée | Choix du user (option 1) |

### Étape 4 ter — Brique images v1.3.0 : plan à hauteurs, rognage final au plan
- [x] 4t.1 `PlanPNG` produit le plan à hauteurs (D-9), même calage ; tests (rampe monotone, ombre, pointillé sur deux faces superposées synthétiques).
- [x] 4t.2 Rognage FINAL du fond à la silhouette du plan (marge 0,5 m) après l'eau et tout calque peint (4b.1).
- [x] 4t.3 Catalyst sud (4b.2) : mesure de la cause, correction générale ou report écrit.
- Gate : vet, `-race`, `gamefiles`, revue adversariale ; tag v1.3.0.
| 2026-11-14 | 4t | map-images `feat/plan-hauteurs` (95dba5a, 80ed7c1 ; PR #3) : plan à hauteurs (rampe vérifiée en constantes, ombre, pointillés ; trous intérieurs papier), rognage final après l'eau dans les deux chaînes (Deadlock : 407 176 cellules d'eau retirées sur 425 172, 0 de matière) ; diagnostic Catalyst : l'amas sud (30,8 m², 21 300 cellules) est effacé par le rognage aux ZONES nommées (aucun callout ne le couvre), pas par la géométrie ; règle (décision du pilote, conforme à la règle du user « le plan est le masque ») : sous rognage au maillage, la silhouette prime sur les zones (union) ; en cours | vet, `-race`, `gamefiles` verts | en cours |
| 2026-11-14 | 4t (clos) | map-images v1.3.0 (e36ba32, tag e86b2ff) : corrections de revue (altitudes non finies, arêtes pentues, câblage des deux chaînes testé sans jeu, journal de garde, README) ; silhouette prime sur les zones (Catalyst 22 → 25/29 ancres, 24 123 cellules sauvées) | vet, `-race`, `gamefiles` ; mutations vérifiées | v1.3.0 |
| 2026-11-14 | 4 (recuisson finale) + 5 | LevelUp : 66 recuites, 41 Forge rognées depuis le fond d'origine (masque issu du maillage, pas de l'image plan), plans à hauteurs pour toutes les cartes sauf Vallaheim (ignorée), Deadlock 407 178 cellules d'eau retirées, Cliffhanger fond d'août (oracle 0,675) ; web : préférence « Fond : Rendu / Plan » dans les trois vues, murs selon le fond réellement affiché | gates ciblés, archlint, lint, check-types, openapi-check, vitest des vues 2D | a160e5df56, fc2572d32c … 5cf49b1a0f |
| 2026-11-14 | 6 | PR #208 : CI verte (32) ; trois remises à jour sur la release qui avançait (décodeur v1.3.7, correctif de lint staticcheck sur `replayverite` d'abord appliqué ici puis par la release, test de cadence corrigé par la release) ; fusion squash | CI 32 verts sur 12acacfbea, seule la ligne de journal diffère ensuite | e83b4f4c2c |
