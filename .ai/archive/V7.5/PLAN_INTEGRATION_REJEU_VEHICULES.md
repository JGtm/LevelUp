# PLAN — Intégration des véhicules dans le rejeu 2D (point 2 du chantier)

> Écrit le 2026-09-02, revu à la grille `plan-review` le même jour (worktree
> `wt/vehicules-tourelles`). Base : cartographie lecture-seule 2026-09-02 (chemins:lignes
> vérifiés sur pièces). Statut : PROPOSÉ.
> Deux items dépendent du rendu de l'agent « point 3 », marqués [ATTENTE-P3] avec repli.

## Objectif et critère de succès

Rejouer un match Behemoth Super Fiesta et un Launch Site Super Fiesta dans le rejeu 2D
avec : véhicules visibles (sprites v4 validés), teintés à la couleur d'équipe du conducteur
courant (neutre sans conducteur), orientés dans le sens du déplacement (dernier cap à
l'arrêt), apparaissant à la naissance et disparaissant à la fin de vie, toggle de calque
FR/EN. Effort : A rapide, B lourd, C moyen, D rapide. Branche : `wt/vehicules-tourelles`.

## Contrat d'exécution (chaque agent exécutant le lit AVANT d'agir)

- Contrat `plan-execution` : ordre strict (un lot clos avant le suivant, sauf A∥B prévu),
  aucun report d'item exécutable, AUCUN fix hors périmètre (le consigner en « Découvertes »).
- Statuts d'item à la clôture d'un lot : `[x]` fait / `[~]` couvert ailleurs (référence) /
  `[!]` non traité (justification écrite). Aucune case vide.
- « Clos » = tous les items statués ET le gate du lot vert (commandes exactes ci-dessous,
  code de sortie vérifié, échecs cherchés avec `grep '^--- FAIL:'`, jamais un filtre nu).
- Reprise de session : l'avancement se lit ICI (statuts) + `.ai/thought_log.md` (une entrée
  par lot, préfixée en tête, octets bruts Bash). Aucun commit sans accord de l'orchestrateur.
- Découvertes hors périmètre : les noter en fin de ce fichier, section « Découvertes ».

## Décisions de cadrage (TRANCHÉES — ne pas rediscuter en cours d'exécution)

- UN document : champ `Vehicles` dans `ReplayDocument`
  (`internal/analysis/replay/document.go:310`) + bump `SchemaVersion`. AUCUNE route nouvelle,
  AUCUNE query key nouvelle, AUCUNE interface `port` nouvelle (route unique existante
  `GET /players/{player_slug}/matches/{match_id}/replay`).
- Title-agnostic : patron « titleSlug traversant + dégradation silencieuse » (modèle
  `mapWeaponPadsForKeys`, `internal/service/replay_map_weapon_pads.go:35-77`). Il n'existe
  AUCUNE clé capability `replay.*` : exemption assumée à la règle `HasCapability`, documentée
  ici — ne pas en créer une.
- Le package `replay` reste un DTO bespoke autoporté (pas de type canonical) : pattern
  établi du dépôt (document.go:3-4), exemption assumée à la grille.
- Teinte : option `'multiply'` ajoutée à `tintedIconCanvas` ; `'source-in'` reste le défaut
  (masques HUD). Sprites véhicules = traits noirs → toujours `'multiply'`.
- Orientation : cap = atan2 sur la vitesse (i1) en mouvement ; à l'arrêt dernier cap connu ;
  à la naissance cap du spawn. Sprites nez en haut. La constante d'écart écran (cap − 90° ou
  autre) n'est PAS une décision : c'est un GATE du lot C (vérifiée sur un film connu, valeur
  et preuve consignées dans le rapport du lot).
- Fin de vie : disparition nette du sprite. AUCUN effet de destruction en V1 (pas de killFx
  véhicule) — c'est tranché, ne pas en ajouter.
- TAILLE (décision utilisateur 2026-09-02) : proportionnelle au monde ENTRE véhicules
  (longueur sprite px × mm/px du manifeste), ancrée sur le pion : constante nommée unique
  telle que le Mongoose ≈ 1,5-2 pions de long (pion = pixels fixes, CORE 3,4 + RING 6,5 px).
  Véhicules en pixels fixes vis-à-vis du zoom, comme les pions. Plancher de lisibilité +
  plafond doux (Pelican). PRÉREQUIS : re-rendre les 12 sprites hors familles Warthog/Mongoose
  à -cellmm=10 fixe (ils étaient en cadrage auto -cote=256, échelle non garantie) + MAJ
  index.json — petit lot module AVANT le gate C.
- TOURELLES (question utilisateur 2026-09-02, supposition confirmée) : Shade et tourelle
  montée sont des VÉHICULES SANS TRANSLATION — couverts PAR CONSTRUCTION par le pipeline
  (même archétype ti=40 : census, vies, épisodes d'occupation, teinte). Shade déjà résolu
  dans la table des châssis + sprite servi ; tourelle montée : MPPWord32 jamais observé en
  film → marqueur neutre jusqu'à observation (dégradation normale). Orientation V1 =
  arbitraire (nez en haut) : un objet qui ne translate jamais n'a ni cap de mouvement ni cap
  de spawn décodable — assumé, ne pas inventer. Tourelle ARRACHÉE = arme portée, hors calque.
- PION EMBARQUÉ (décision utilisateur 2026-09-02) : le pion du joueur DISPARAÎT pendant son
  épisode d'occupation (le bipède ne réplique plus — le pion serait inventé) ; le véhicule
  teinté porte l'information. Le NOM du joueur s'aligne sur le véhicule (même convention de
  placement que sur les pions), conducteur en premier ; passagers empilés dessous (les
  épisodes portent le siège). À la sortie (ou mort à bord, rare), le pion et son nom
  reprennent au point de sortie ; la traînée est suspendue pendant l'épisode et reprend
  ensuite. Occupant inconnu → véhicule neutre, comportement pion inchangé.
- Identité châssis : table Go STATIQUE MPPWord32 → famille, alimentée par les valeurs
  observées (rapports V0/V1b + corpus), documentée valeur par valeur (film source). Inconnu
  → famille vide, libellé générique, compteur + `slog.InfoContext`. PAS de lecture des
  `.module` côté serveur.
- Libellés : pas de FR/EN en dur côté Go. Famille → nom affiché via le manifeste
  `index.json` du lot A (noms propres du jeu, identiques FR/EN) ; les libellés d'UI
  (calque, tooltips) via `i18n.ts` FR ET EN (parité typée).

## LOT A — Assets véhicules servis + teinte multiply (agent : Sonnet)

Items :
- [ ] A1. `static/vehicles-assets/halo_infinite/replay/` : copie des 18 PNG de
      `.ai/V7.5/film_re/sprites_v4/` + `index.json` (fichier → famille, mode hash source,
      mm/px, date, statut de validation). `.ai/` reste la source de travail.
- [ ] A2. Go : `KindVehicle` dans `internal/assets/static/kinds.go` + `layout.go`
      (`"vehicles-assets"`) + cas dans le test d'URL existant.
- [ ] A3. Web : entrée `vehicle` dans `FOLDER` de `apps/web/src/lib/staticAssets.ts`.
- [ ] A4. Web : option de composition `'multiply'` dans `TintedIconOptions`
      (`replayDraw.ts:355-386`), défaut `'source-in'` inchangé ; test vitest de l'option si
      canvas testable, sinon `[!]` justifié + vérification visuelle reportée au gate C.
Gate A (commandes exactes) :
`cd apps/go-api && gofmt -l internal/assets/ && go vet ./internal/assets/... && go test ./internal/assets/...`
`cd apps/web && Remove-Item -Recurse -Force node_modules\.tmp; npm run typecheck && npm run lint && npm run test`
Zéro régression d'icônes HUD (défaut inchangé). Dépendances : aucune (part en T0, ∥ B,
GOCACHE dédié).

## LOT B — Chemin de production Go : filmdec → ReplayDocument (agent : Opus)

Items :
- [ ] B1. `internal/analysis/replay/build_vehicles.go` (modèle `build_ground_weapons.go`),
      appelé depuis `build.go` : census ti=40 (`ScanFilmWorldObjectKeyframes(dir, 40)`),
      créations (`ScanFilmVehicleCreationsForBand` : MPPWord32, spawn pos+cap),
      trajectoires (`ScanFilmBipedPositionsForBand` bande ti=40, sous-échantillonnées).
- [ ] B2. Conducteur : productioniser la primitive « début de trou de position »
      (`vehicules_v1_conducteur_test.go`) → segments {t0,t1,xuid} par vie véhicule.
- [ ] B3. Monte/descend : brancher `ScanFilmVehicleEvents` — TRANCHÉ P3 (2026-09-02) :
      la grammaire board est décodée (domaines 2/3/7, occupant en bande 68/68, siège
      recoupé ; rapport `V3_EMBARQUEMENT_2026-09-02.md`). Épisodes d'occupation =
      board→exit appariés (6/6 trajets complets, écart de numérotation 0), complétés par
      début/fin de trou de position (sorties : 100 % ferment le trou).
- [ ] B4. Fin de vie : borne census SEULE, `cause=inconnue` — TRANCHÉ P3 : le gate
      « destruction datée par la mort du conducteur » a ÉCHOUÉ (rapport
      `V3_DESTRUCTION_DATEE_2026-09-02.md` : 0 occupant à bord à la fin serrée sur 64 vies,
      mort à bord ANTI-corrélée 3,8 % vs 21,3 % témoin — le conducteur sort vivant, le
      véhicule réplique encore 13-36 s après). INTERDIT de publier `cause=destruction`.
- [ ] B5. Table `MPPWord32 → famille` (fichier Go dédié + test) selon la décision de
      cadrage ; compteur de dégradation + `slog.InfoContext` par famille inconnue.
- [ ] B6. Types : `VehicleTrack` + `doc.Vehicles` + bump `SchemaVersion` + compteurs de
      couverture (modèle `GroundWeaponCoverage`).
- [ ] B7. `VehicleLabel` rempli À LA REQUÊTE côté service (modèle `WeaponLabel` /
      `replaylabels`) : `Img = static.URL(static.KindVehicle, slug, famille, ".png")`,
      `Tinted=true` ; test service avec mock (dégradation famille vide → pas d'Img, nil-safe).
- [ ] B8. openapi : `make openapi-gen`, schéma `VehicleTrack` présent, aucun champ orphelin.
Gate B :
`cd apps/go-api && gofmt -l internal/analysis/replay/ && go vet ./internal/analysis/... ./internal/service/... && CGO_ENABLED=0 go test ./internal/analysis/filmdec/ ./internal/analysis/replay/ ./internal/service/...`
(vert SANS variables d'env — tous les instruments skippent), `grep '^--- FAIL:'` vide, code
de sortie 0. `dist3` via adaptateur (garde-rail `TestUneSeuleFormuleDeDistance3D` vert).
Fichiers ≤ 500 L, fonctions ≤ 80 L. Dépendances : P3 pour B3/B4 (replis définis — le lot
n'attend PAS P3 pour être clos si les replis sont consignés).

## LOT C — Calque web véhicules (agent : Sonnet)

Items (après clôture B + `npm run generate-types`) :
- [ ] C1. `replayNormalize.ts` : `vehicles: raw.vehicles ?? []` + type `Ready` (patron
      `Filled`).
- [ ] C2. Helper `drawRotatedSprite(ctx, img, x, y, angle, scale)` dans `replayDraw.ts`
      (patron `save/translate/rotate/drawImage/restore` de `muzzleFlash.ts:159-165`) +
      test vitest de la transformation si testable, sinon `[!]` + preuve visuelle au gate.
- [ ] C3. `useReplayVehicles.ts` + `vehiclesLayer.ts` (patron `useReplayWeaponPads.ts`) :
      vignettes cuites par famille × équipe (`tintedIconCanvas(..., 'multiply')`, neutre
      sans conducteur), interpolation entre échantillons, cap selon la décision de cadrage,
      disparition à la fin de vie.
- [ ] C4. `ReplayCanvas.tsx` : insertion entre les meubles (étapes 6/7) et les trajectoires
      joueurs (étape 9).
- [ ] C5. Toggle `showVehicles` (`useReplaySettings.ts`) + entrée `ReplaySettingsLayers.tsx`
      conditionnée à `available` + i18n `layerVehicles` FR ET EN.
- [ ] C6. GATE D'ANGLE : sur un film Behemoth SF connu, vérifier qu'un Warthog roulant vers
      un repère identifiable est rendu nez dans le sens du mouvement ; consigner la
      constante d'écran retenue et la preuve (capture) dans le rapport du lot.
- [ ] C7. PION EMBARQUÉ, MULTI-PASSAGERS OBLIGATOIRE (rappel utilisateur 2026-09-02) : un
      véhicule porte PLUSIEURS épisodes d'occupation SIMULTANÉS (Warthog 3 places, Razorback
      4, Falcon tourelles...). CHAQUE occupant embarqué a son pion + nom supprimés ; les noms
      sont reportés sur le véhicule, empilés : conducteur (siège 0) en premier, puis
      passagers par siège croissant. Teinte du véhicule = équipe du conducteur ; à défaut
      (siège 0 inconnu) celle de n'importe quel occupant connu (même équipe en pratique).
      Reprise pion+nom de CHACUN à SA sortie (les sorties sont indépendantes) ; traînée
      suspendue pendant l'épisode. Taille selon la décision de cadrage (constante nommée,
      Mongoose ≈ 1,5-2 pions).
Gate C :
`cd apps/web && Remove-Item -Recurse -Force node_modules\.tmp; npm run typecheck && npm run lint && npm run test`
Couleurs par tokens sémantiques uniquement. Aucune édition de `routeTree.gen.ts`, aucune
query key nouvelle.

## LOT D — Vérification de bout en bout + clôture (orchestrateur)

- [ ] D1. Serveur dev + browser : rejouer 1 match Behemoth SF et 1 Launch Site SF ; chaque
      critère de l'objectif vérifié à l'œil (captures dans le rapport).
- [ ] D2. `adversarial-review` du diff complet par agent frais Opus (risques : document
      versionné, première rotation bitmap, teinte multiply).
- [ ] D3. `delivery-checklist` + `make gate-push` ; statuts de CE fichier tous remplis ;
      MAJ `PLAN_VEHICULES_TOURELLES.md` + thought_log ; commits sur la branche (accord
      utilisateur requis) ; pas de push sans accord explicite.

## Hors périmètre (assumé)

Sons dans le rejeu (moteurs = capture en jeu), calque « socles véhicules » par carte
(catalogue .mvar inexistant), animation de tourelle indépendante, Halo 5 (dégradation
silencieuse suffit).

## Découvertes (à remplir par les exécutants — ne pas corriger hors périmètre)

(vide)

## Statuts (clôture 2026-09-02)

- [x] LOT A — 4/4 items, gates verts (rapport de l'agent, 2026-09-02).
- [x] LOT B — 8/8 items, gates verts ; écart documenté : service compile en CGO=1 (duckdb,
      préexistant). fe32c0f4/cb96ca07 résolus ensuite par l'orchestrateur (V3D, hlmt Warthog).
- [x] LOT C — C1-C5, C7 faits ; C6 `[!]` : dérivation + 4 points cardinaux + recoupement réel
      (0,5°) OK, preuve VISUELLE reportée au D1 (pas d'outil navigateur en session).
- [~] LOT D — D2 fait (REVUE_ADVERSARIALE_REJEU : GO avec réserves, toutes corrigées le jour
      même : gocache ignoré, end=unknown, clamp épisodes, i18n honnête, citation C6, prédicat
      gaté sur le toggle). D1 = preuve visuelle UTILISATEUR en attente (merge local feat/v75).
      D3 = commit local fait ; gate-push/CI à jouer avant tout merge distant.

## Découvertes (registre)

- Orientations des sprites re-rendus : heuristique géométrique faillible — l'utilisateur
  signale des sens faux sur PLANCHE_ECHELLES ; correction à pointer (suivi post-commit).
- Sons : piste banques FERMÉE pour les moteurs (verdict utilisateur, V3E) → capture en jeu.
