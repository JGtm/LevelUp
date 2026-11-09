# REVUE ADVERSARIALE — intégration des véhicules dans le rejeu 2D (2026-09-02)

> Relecteur à contexte frais (skill `adversarial-review`). Aucune correction apportée : ce
> fichier est un REGISTRE DE CONSTATS, rien d'autre. Aucun `git add`, aucun commit, aucun
> fichier du diff modifié.
>
> **État relu** : worktree `C:\Users\Guillaume\Projects\LevelUp-wt-vehicules`, branche
> `wt/vehicules-tourelles`, HEAD `230b615b1`, arbre de travail au **2026-09-02 19:06**.
> ATTENTION : l'arbre a BOUGÉ pendant la revue — l'instantané fourni en entrée annonçait
> HEAD `67584a13c` avec `cmd/vs-measure/{armes,plateau}.go` modifiés et `goose.go` non suivi ;
> ces trois-là ont été absorbés par le commit `230b615b1` en cours de relecture. Les constats
> ci-dessous portent sur l'arbre daté ci-dessus.
>
> **Écart connu NON re-signalé** (consigne) : `internal/service` et `internal/assets` ne
> compilent pas sous `CGO_ENABLED=0` (duckdb) ; idem `cmd/{vs-measure,weapon-sounds}` sous
> `CGO_ENABLED=0` (`internal/ooz`, contrainte de build C++). Préexistant.

---

## 0. Gates rejoués (octets bruts, code de sortie vérifié)

| Commande | Résultat |
|---|---|
| `gofmt -l internal/analysis/{replay,filmdec}/ internal/service/ internal/assets/ cmd/vs-measure/ cmd/weapon-sounds/` | sortie VIDE |
| `CGO_ENABLED=0 go vet ./internal/analysis/replay/... ./internal/analysis/filmdec/...` | exit 0 |
| `CGO_ENABLED=0 go test ./internal/analysis/replay/ ./internal/analysis/filmdec/` | `ok replay 29,382s` · `ok filmdec 1,171s` |
| `CGO_ENABLED=1 go test ./internal/service/... ./internal/assets/...` | 6 paquets `ok`, 0 `FAIL` |
| `CGO_ENABLED=1 go vet ./cmd/vs-measure/... ./cmd/weapon-sounds/...` + `go test ./cmd/weapon-sounds/...` | exit 0 · `ok 0,499s` |
| `npm run typecheck` | exit 0, aucune sortie |
| `npx vitest run src/features/match-replay/` | **127 fichiers / 1976 tests passés**, 0 échec |
| `npm run lint` | **0 erreur**, 24 avertissements — tous préexistants ; le seul sur un fichier touché est `ReplayCanvas.tsx:522 exhaustive-deps 'objectiveObjects'`, déjà là avant ce lot |

---

## 1. Conditions vérifiées QUI TIENNENT (24)

Elles sont listées parce qu'un relecteur qui ne valide jamais rien n'a plus de crédit quand
il alarme. Chacune a été ouverte dans le code réel, pas déduite du diff.

1. **Parité de schéma** : `document.go:328 SchemaVersion = 29` ↔ `replaySchemaLogic.ts:32
   EXPECTED_REPLAY_SCHEMA_VERSION = 29`. Le garde-rail de parité est le SEUL consommateur de
   la constante web (`replaySchemaLogic.ts:20-25` le dit, vérifié par grep : 0 autre lecteur).
2. **Compat des artefacts v28** : le chemin de LECTURE (`replay_service.go GetReplay`) ne
   compare AUCUNE version — seul `internal/replaybuild/artifact_store.go:112` refuse un
   artefact soumis dont le schéma diffère, ce qui est exactement la clé de reprise du
   backfill voulue. Un v28 déjà cuit continue donc de se servir, sans véhicules.
3. **Normalisation** : `replayNormalize.ts:332` comble `vehicles`, `samples`, `rides` ;
   `spawn` reste optionnel. Le contrat `replayContract.test.ts:243,248-250` recense les trois
   chemins nullables et le test de frontière (`:383-391`) vérifie le comblement.
4. **Golden** : `assembly_000d5950.golden` ne change QUE la ligne de schéma — aucune dérive
   d'un autre calque.
5. **`vehicleScreenAngle`** : dérivation refaite. Après `ctx.rotate(a)`, le vecteur local
   `(0,-1)` (nez du sprite) devient `(sin a, -cos a)` ; la direction écran d'un cap monde `h`
   sous projection Y inversée est `(cos h, -sin h)` ; d'où `a = 90° − h`. Le code
   (`vehiclesLayer.ts:95`) est correct, et `VEHICLE_DEFAULT_HEADING_DEG = 90 → 0 rad` tient.
6. **Sprites nez en haut** : `static/vehicles-assets/halo_infinite/replay/index.json` porte
   « pivoté 180 deg (nez en haut) » pour chopper/falcon/ghost/mongoose/gungoose… — cohérent
   avec l'hypothèse du calque.
7. **Échelle** : `mongoose.png` mesure réellement **78×128 px** (IHDR lu) →
   `MONGOOSE_REFERENCE_LENGTH_MM = 1280` est juste. Le Mongoose rend
   `1280 × 0,009297 = 11,9 px` CSS = `1,75 × 6,8` (noyau du pion) — dans la fourchette 1,5-2
   demandée. Scorpion `260×388` → 3,03× ; Pelican `841×1122` → brut 104,3 px, écrêté doux à
   `47,6 + √56,7 = 55,1 px`. La règle se comporte comme annoncée sur les vrais fichiers.
8. **Teinte multiply** : après `fillRect`, la transformée est remise à l'identité PUIS le
   miroir est ré-appliqué avant le `destination-in` (`replayDraw.ts:387-409`) — les deux
   passes sont alignées au pixel. L'alpha est bien restauré (`multiply` sur fond `Da=0` rend
   `Cs` opaque, `destination-in` remet `Da·Sa`). Le défaut `source-in` est **strictement**
   inchangé, et `replayDraw.test.ts:69-88` le prouve par la trace des opérations.
9. **`drawRotatedSprite`** : `save`/`restore` équilibrés, un seul niveau, aucune fuite d'état
   (`replayDraw.ts:490-500`). Pas de fuite DPR : l'appelant multiplie l'échelle par `time.k`
   (`vehiclesLayer.ts:382`) et `drawVehiclesLayer` encadre sa boucle d'un `save`/`restore`
   avec remise de `globalAlpha` (`:360, 393-394`).
10. **Chaîne d'assets complète** : les 13 familles de `vehicleFamilyByChassis` ont toutes
    leur PNG sous `static/vehicles-assets/halo_infinite/replay/` (vérifié fichier par
    fichier) ; `static.URL(KindVehicle, slug, "replay/"+f, ".png")` compose
    `/static/vehicles-assets/{slug}/replay/{f}.png` (`urls.go:17-22`, `path.Join`) et
    `server_apiv1.go:1357` sert `/static/*` par un `http.FileServer` nu — les URLs résolvent.
11. **`assignVehicleWindows`** (`vehicle_tracks.go:128-143`) : les deux clamps sont posés dans
    le bon ordre (le voisin `i-1` a déjà été écrêté à `firstUS` de `i` quand `i` le lit), donc
    les fenêtres de deux vies d'un même slot ne se recouvrent JAMAIS et le nuage de positions
    n'est pas attribué deux fois. Vérifié aussi par `TestVehiculeDeuxViesDunMemeSlot`.
12. **Cap absent ≠ cap nul** : `headingForJSON` (`build_aim.go:21-27`) republie 0 en 360, donc
    `omitempty` n'efface jamais un cap réel, et côté web `h !== undefined`
    (`vehiclesLayer.ts:80,84`) est le bon test.
13. **`vehicleActiveRides`** : `.filter().sort()` ne mute pas `track.rides` ; le comparateur
    rend `NaN` quand deux sièges sont inconnus (`Infinity - Infinity`), ce que la spec ECMA
    traite comme `+0` — le tri reste stable et l'ordre du document est préservé, conforme au
    commentaire.
14. **Le port board ne casse pas la sortie** : `decodeExitRefs` (`event_list.go:214-228`) lit
    les MÊMES domaines 1/1/7 aux MÊMES offsets qu'avant l'extraction. `TestBoardEventGrammar`
    (garde-rail sans film, `event_list_board_grammar_test.go:41`) + toute la suite filmdec
    sont verts.
15. **Pas de débordement de lecture aggravé** : la grammaire board passe de `dom1(13/17 bits)
    + 2×dom7(16 bits)` à `dom2(11) + dom3(10) + dom7(16)` — elle lit MOINS loin qu'avant. Le
    risque d'index hors bornes de `readBitsAt` (`offline_biped.go:460-467`, sans garde) n'est
    pas accru par ce lot.
16. **Occupants hors bande écartés** : `vehicleEventsByOccupant` (`vehicle_rides.go:279-281`)
    exige `OccupantPresent && OccupantInBand`, ce qui empêche un slot mal lu de fabriquer un
    épisode.
17. **Pas de formule `dist3` recopiée** : le nouveau code n'utilise que `math.Hypot` en PLAN
    (2D). Le garde-rail `TestUneSeuleFormuleDeDistance3D` est vert.
18. **Aucune comparaison de slug**, aucune clé `capabilities.toml` touchée (`git status` :
    zéro fichier sous `config/titles/`), aucune query key nouvelle, `routeTree.gen.ts`
    intact.
19. **Zéro couleur en dur côté web** : grep hex + classes Tailwind couleur sur
    `vehiclesLayer.ts`, `useReplayVehicles.ts`, `ReplaySettingsLayers.tsx`, `replayDraw.ts` →
    aucune occurrence. Les encres viennent de l'appelant (`neutralInk`, `labelStroke`).
20. **i18n FR ET EN** : `layerVehicles` / `layerVehiclesHint` présents dans les deux locales
    (`i18n.ts:159-161` et `:505-507`) et typés au contrat (`i18nContract.ts:365-374`).
21. **Seuils de taille** : aucun NOUVEAU fichier au-dessus de 500 L
    (`vehicle_rides.go` 321, `vehicle_tracks.go` 319, `document_vehicles.go` 311,
    `build_vehicles.go` 154, `vehicle_families.go` 120, `replay_vehicle_labels.go` 100,
    `vehiclesLayer.ts` 395, `useReplayVehicles.ts` 184). `build.go` +13 L et `document.go`
    +35 L de commentaire seulement — l'accroissement de la dette gelée est minimal. Aucune
    fonction nouvelle > 80 L. Le plafond de `ReplayCanvas.tsx` ne bouge pas (678 → 678).
22. **Journal en contexte côté service** : `replay_vehicle_labels.go` utilise bien
    `slog.InfoContext/WarnContext` (`:56, 71, 77`). Le `slog.Info` sans ctx de
    `build_vehicles.go` / `document_vehicles.go` suit le patron du paquet `replay` (build.go
    fait pareil partout) — pas une régression de ce lot.
23. **Dégradation title-agnostic** : `resolveVehicleLabels` compose pour `s.titleSlug` sans
    comparaison ; un titre sans dossier d'assets sert des URLs qui 404 et le client garde son
    marqueur neutre — comportement identique à une famille inconnue.
24. **L'hypothèse « le trou de position est en fait une MORT »** (le faux positif évident :
    un joueur tué à moins de 1,5 m d'un véhicule ouvre un trou ≥ 3 s, puisque `lives.go:41`
    fixe la coupure de vie à 5 s et le respawn médian à 8 s). ELLE EST RÉFUTÉE PAR LA MESURE
    DU DÉPÔT, sur la MÊME primitive (`vehicules_v1_conducteur_test.go:68-72` sélectionne les
    trous par le même oracle géométrique que la production) : l'occupant meurt DANS son trou
    dans 0 % (30 trous, 4 films) à 3,8 % (80 trous) des cas, contre 20-21,3 % pour le témoin
    à occupant décalé. Le résiduel est publié (`Coverage.RidesFromGap`). **Constat non
    retenu.**

---

## 2. Registre des constats

### BLOQUANT

#### B1 — `.gocache-agentA/` (167 Mo, 3107 fichiers) n'est pas ignoré par git

- **Fichier** : racine du dépôt, `.gitignore` (absence d'entrée) — `git check-ignore -v
  .gocache-agentA` rend **exit 1** (non ignoré) ; `git status --porcelain` le liste `??`.
- **Déclenchement** : le commit groupé demandé. `git add -A` ou `git add .` depuis la racine.
- **Conséquence observable** : 3107 objets et 167 Mo de cache de compilation Go entrent dans
  l'historique, de façon irréversible sans réécriture. `du -sh .gocache-agentA` → `167M`.
- **Gravité proposée** : P0 (à traiter AVANT le `git add`, pas après).

### MAJEUR

#### M1 — Un épisode d'occupation peut être publié HORS de la fenêtre d'affichage de son véhicule : le joueur n'est alors dessiné NULLE PART

- **Fichiers** : `internal/analysis/replay/vehicle_rides.go:157-161` (bornes de l'épisode) ·
  `vehicle_rides.go:261` (rattachement à la vie) · `vehicle_tracks.go:131` (`loUS`) ·
  `vehicle_tracks.go:209-233` (`vehicleBounds`) · `apps/web/src/features/match-replay/
  vehiclesLayer.ts:106-108` et `:362` (fenêtre de tracé) · `vehiclesLayer.ts:252-268`
  (prédicat) · `useReplayVehicles.ts:90` · `replayMarkers.ts:244` (retour anticipé).
- **Ce qui devrait être vrai** : tout épisode `[T0,T1]` publié pour une vie est contenu dans
  la fenêtre `[T0, T1Max]` de cette vie ; sinon la suppression du pion couvre des images où
  le véhicule n'est pas dessiné.
- **Condition de déclenchement, vérifiée sur pièces** : une vie SANS record de création lu
  (cas réel — `Coverage.WithSpawn` est publié précisément parce qu'il est `< Published`,
  cf. `document_vehicles.go:192-195`, et `vehicleTrackOf:186` publie une vie sur ses seuls
  échantillons). Alors `bornUS = l.firstUS` et `t0 = frame(firstUS)`
  (`vehicle_tracks.go:212, 220`), tandis que la fenêtre de rattachement commence à
  `loUS = firstUS − 20 s` (`vehicle_tracks.go:131`, `vehicleCensusTolUS = 20_000_000`).
  `vehicleLifeAt` (`vehicle_rides.go:261`) accepte donc un trou dont le début tombe jusqu'à
  **20 s avant `t0`**, et `vehicleRideOf` publie `T0 = clock.frame(startUS)` **sans aucun
  clamp** sur `[t0, t1max]`. Cas symétrique en fin : un trou qui se ferme après `goneByUS`
  donne `T1 > t1max`.
- **Conséquence observable** : sur ces images, `vehicleVisibleAt` rend faux → aucun sprite,
  aucun nom sur le véhicule (`vehiclesLayer.ts:362 continue`) ; et `isEmbarkedAt` rend vrai →
  `drawTracksLayer` supprime le pion, le nom, la traînée ET la croix de mort du joueur
  (`replayMarkers.ts:244`). Le joueur DISPARAÎT de la carte pendant jusqu'à 20 s, sans qu'un
  seul pixel ne le remplace.
- **Gravité proposée** : P1 (résultat faux servi à l'UI ; c'est le constat que je corrigerais
  en premier). Correctif de forme : borner `r.T0`/`r.T1` à `[t0, t1max]` de la vie côté Go,
  OU conditionner le prédicat à `vehicleVisibleAt` côté web.

#### M2 — La réserve du calque (FR et EN) affirme le contraire de ce que le calque fait

- **Fichiers** : `apps/web/src/features/match-replay/i18n.ts:160` (FR) et `:506` (EN) ·
  `vehiclesLayer.ts:106-108`.
- **Ce que la réserve dit** : « le calque cesse de dessiner un véhicule **à la dernière preuve
  mesurée de sa présence**, rien de plus » / « the layer stops drawing a vehicle **at the last
  measured proof of its presence**, nothing more ».
- **Ce que le code fait** : `vehicleVisibleAt` rend vrai jusqu'à **`t1max` inclus**, et
  `t1max` est par construction la PREMIÈRE PREUVE D'ABSENCE (`document_vehicles.go:62-64`,
  `vehicle_tracks.go:225-228`), pas la dernière preuve de présence (`t1`).
- **Déclenchement, chiffré par un test du lot lui-même** : `build_vehicles_test.go:118-121`
  (`TestVehiculeVieSansOccupant`) publie `T1 = 210`, `T1Max = 410` — le sprite est dessiné
  **à pleine opacité pendant 200 frames = 20 s** après la dernière preuve. Le calque frère
  des armes au sol ESTOMPE entre `t1` et `t1max` (`document_ground_weapon_items.go:59-68`)
  précisément pour ne pas affirmer ça, et sa propre réserve i18n le dit.
- **Conséquence observable** : la phrase qui porte l'honnêteté du calque est fausse en FR ET
  en EN ; l'utilisateur voit un véhicule là où le film ne dit rien.
- **Gravité proposée** : P1 (soit la réserve s'aligne sur `t1max`, soit le tracé s'arrête à
  `t1` — mais pas les deux versions en même temps).

#### M3 — `VehicleTrack.End = "inconnue"` : une valeur d'énumération FRANÇAISE dans le contrat publié

- **Fichier** : `internal/analysis/replay/document_vehicles.go:38`.
- **Ce qui devrait être vrai** : les valeurs d'énumération du document sont des identifiants
  stables, en anglais, comme partout ailleurs dans ce paquet.
- **Vérifié sur pièces, dans le MÊME paquet** : `document_ground_weapon_items.go:42,44,46` →
  `"pickup"`, `"seen"`, `"open"` ; `equipment_placements.go:211,213,215` → `"deployed"`,
  `"dropped"`, **`OriginUnknown = "unknown"`** ; et dans le **même fichier neuf**,
  `vehicle_rides.go:68,71,73` → `"event"`, `"mixed"`, `"gap"`. Une seule valeur du lot est en
  français, et c'est exactement celle dont l'équivalent anglais existe déjà à côté.
- **Conséquence observable** : la valeur part dans `openapi.yaml` (schéma `VehicleTrack.end`),
  dans `generated.ts`, et dans tous les artefacts que le backfill v29 va cuire. La changer
  après le backfill est un changement de contrat + une re-cuisson complète. Accessoirement,
  c'est une chaîne FR émise par Go dans une charge utile (règle L2 du dépôt) : si un client
  l'affiche un jour, elle est intraduisible.
- **Gravité proposée** : P1 (coût quasi nul maintenant, élevé après le premier backfill).

#### M4 — Le prédicat « embarqué » reste actif quand le calque des véhicules est ÉTEINT

- **Fichiers** : `useReplayVehicles.ts:66-72` (contrat), `:90`, `:171` (`paint` sort si
  `!enabled`), `:183` (le prédicat est rendu inconditionnellement) · `ReplayCanvas.tsx:425`.
- **Déclenchement** : l'utilisateur décoche « Véhicules » dans le tiroir des calques
  (`ReplaySettingsLayers.tsx:213-219`). Le réglage est persisté
  (`useReplaySettings.ts:391`), donc l'état survit au rechargement.
- **Conséquence observable** : `paint` ne dessine plus rien, mais `isEmbarkedAt` continue de
  supprimer pion + nom + traînée + croix de mort de CHAQUE occupant pendant CHAQUE épisode
  (`replayMarkers.ts:244`). Les joueurs clignotent hors de la carte sans qu'aucun objet ne
  porte l'information, et **aucun réglage ne permet de les récupérer**. La réserve du calque
  ne mentionne pas cet effet de bord.
- **Note** : le comportement est revendiqué en commentaire (`useReplayVehicles.ts:69-70`),
  mais l'argument (« un pion figé à son point d'embarquement resterait faux ») ne vaut que
  tant qu'un véhicule le remplace ; quand le calque est éteint, il ne reste rien.
- **Gravité proposée** : P1 — décision d'ergonomie à ARBITRER par l'utilisateur (la décision
  de cadrage « PION EMBARQUÉ » ne tranche pas le cas « calque éteint »), pas par l'auteur ni
  par le relecteur.

#### M5 — Le gate C6 est affirmé PASSÉ dans le code, sans aucune pièce

- **Fichier** : `apps/web/src/features/match-replay/vehiclesLayer.ts:23-24` — « Vérifié par
  `vehicleScreenAngle.test.ts` sur les quatre points cardinaux **ET par le gate C6 (film
  Behemoth SF réel, cf. le rapport du lot)** ».
- **Vérifié sur pièces** : (a) `vehicleScreenAngle.test.ts` **n'existe pas** (`ls
  src/features/match-replay/ | grep -i vehicle` → `useReplayVehicles.ts`,
  `vehiclesLayer.test.ts`, `vehiclesLayer.ts`) ; (b) aucun rapport de lot C n'existe dans
  `.ai/V7.5/film_re/` (les cinq rapports datés du 2026-09-02 portent sur les échelles, les
  sons de destruction, la reconstruction V3E, la destruction datée et l'embarquement) ;
  (c) l'item C6 du plan (`PLAN_INTEGRATION_REJEU_VEHICULES.md:145`) est `[ ]`.
- **Conséquence observable** : la seule constante d'écran du calque est documentée comme
  validée sur un film réel alors que rien ne l'atteste ; un lecteur qui suit la référence ne
  trouve rien. La dérivation mathématique, elle, TIENT (cf. §1.5) — c'est la PREUVE
  EMPIRIQUE annoncée qui manque, et le plan en faisait un gate.
- **Gravité proposée** : P1 (soit le gate est joué et la pièce consignée, soit la phrase
  perd sa seconde moitié).

#### M6 — Contrat `plan-execution` non honoré : zéro item statué, zéro entrée de journal pour les lots A/B/C

- **Fichiers** : `.ai/V7.5/PLAN_INTEGRATION_REJEU_VEHICULES.md` — **23 items `[ ]`, 0 `[x]`**
  (comptés par grep) ; ligne `:184` « Statuts » : les quatre lots non cochés ; section
  « Découvertes » (`:178-180`) VIDE. `.ai/thought_log.md` : les 4 entrées ajoutées portent sur
  V3/V3D/V3E (sons, embarquement, destruction) — **aucune** sur les lots A, B, C du plan
  d'intégration.
- **Conséquence observable** : CLAUDE.md (« L'absence d'entrée = tâche non terminée » ;
  plan-execution règle 3 « Aucun item sans statut à la clôture ») rend la livraison non
  close ; une reprise de session ne peut pas lire l'avancement là où le plan dit qu'il se lit.
- **Aggravant** : `weaponPadsLayer.ts:123` renvoie explicitement le lecteur à « cf.
  Découvertes » pour une dette qu'il déclare hors périmètre — la section pointée est vide.
- **Gravité proposée** : P1 (bloque `delivery-checklist`, pas la correction du code).

### MINEUR

#### m1 — Références de documentation vers des fichiers inexistants

- `internal/analysis/replay/build_vehicles.go:14` : « c'est lui, et lui seul, qui BORNE la fin
  de vie (**cf. build_vehicles_end.go**) » — ce fichier n'existe pas (`ls
  internal/analysis/replay/ | grep -i vehicle` : `build_vehicles.go`, `build_vehicles_test.go`,
  `document_vehicles.go`, `vehicle_families.go`, `vehicle_rides.go`, `vehicle_tracks.go`). Le
  bornage vit en réalité dans `vehicle_tracks.go:209-233`.
- `apps/web/src/features/match-replay/vehiclesLayer.ts:23` : `vehicleScreenAngle.test.ts`
  (cf. M5).
- `apps/web/src/features/match-replay/weaponPadsLayer.ts:123` : « cf. Découvertes » (cf. M6).
- **Conséquence** : trois renvois qui ne mènent nulle part dans des en-têtes dont c'est
  précisément la fonction dans ce dépôt.

#### m2 — Le manifeste d'échelle et les sprites ne se rechargent JAMAIS à un changement de titre

- **Fichier** : `useReplayVehicles.ts:117` (`if (!enabled || manifestRef.current) return`) et
  `:96-99` (`if (map.has(family)) continue`), alors que `titleSlug` figure bien dans les
  dépendances (`:144`).
- **Déclenchement** : naviguer d'un rejeu `halo_infinite` vers un rejeu d'un autre titre sans
  démontage du composant.
- **Conséquence observable** : le `Map` de `manifestRef` et celui de `rawImagesRef` gardent le
  contenu du titre PRÉCÉDENT ; les véhicules du nouveau titre sont dimensionnés et dessinés
  avec les sprites de l'ancien. La garde par ref annule l'effet de la dépendance.

#### m3 — Chargement de sprite sans `onerror` : 404 silencieux

- **Fichier** : `useReplayVehicles.ts:102-108` — `im.onload` seulement ; pas de `im.onerror`.
- **Déclenchement** : une famille dont le PNG manque sous `/static/vehicles-assets/{slug}/
  replay/` (cas nominal pour tout titre autre que `halo_infinite`, et pour toute famille
  ajoutée à `vehicle_families.go` avant son sprite).
- **Conséquence observable** : la famille reste indéfiniment sans vignette, `redraw()` n'est
  jamais rappelé, et RIEN n'est journalisé — alors que le chemin du manifeste, lui, fait un
  `console.warn` (`:137`). Équivalent front de l'anti-patron « swallowed error ».

#### m4 — `Ambiguous` sous-compte les chevauchements à 3 occupants et plus

- **Fichier** : `internal/analysis/replay/document_vehicles.go:277` — `if i > 0 && r.T0 <=
  rides[i-1].T1`.
- **Déclenchement** : trois épisodes triés par `T0` sur une même vie, où l'épisode 3 chevauche
  l'épisode 1 mais commence APRÈS `T1` de l'épisode 2 (Razorback 4 places, nommé
  explicitement par la décision C7 du plan ; Warthog conducteur + tourelleur + passager).
- **Conséquence observable** : `coverage.vehicles.ambiguous` annonce moins d'ambiguïtés qu'il
  n'y en a — un compteur d'honnêteté qui ment vers le bas.

#### m5 — Sentinelle `TimestampUS > 0` pour « record de création lu »

- **Fichier** : `internal/analysis/replay/vehicle_tracks.go:185` (`hasSpawn`) et `:213`
  (`vehicleBounds`).
- **Ce qui devrait être vrai** : 0 n'est pas un instant légal de l'horloge du film.
  `filmdec.EquipmentCreation` (`equipment_creation.go:95+`) NE PORTE AUCUN drapeau de
  présence — le zéro de la structure est indiscernable d'un record réellement daté à 0.
- **Conséquence observable** : un véhicule pré-placé dont le record de création tomberait au
  tout premier paquet du film perdrait d'un coup sa naissance, son châssis ET sa famille
  (donc son sprite), sans que `Coverage.WithChassis` ne le distingue d'un record illisible.
  Probabilité faible, coût de lisibilité nul si on passe une clé de présence explicite.

#### m6 — `WithHeading` dépend d'un invariant lointain et non cité

- **Fichier** : `document_vehicles.go:246` (`if s.H != 0`) — juste seulement parce que
  `headingForJSON` (`build_aim.go:21-27`) republie 0 en 360. Le commentaire du compteur
  (`:206-209`) ne mentionne pas cette dépendance ; celui de `VehicleSample.H` (`:137-138`) la
  mentionne pour la sérialisation, pas pour le comptage. Un futur changement d'arrondi casse
  le compteur en silence.

### NOTE

- **N1 — Poids du commit groupé** : `.ai/V7.5/film_re/sons_v3_reconstruits/{Banshee, Chopper,
  Falcon_LMG, Ghost, Gungoose, Mouvement_generique_partage, Scorpion, Warthog_roquettes,
  Wasp, Wraith}` = **280 Mo / 201 fichiers** non suivis (Banshee 45 Mo, Wraith 38 Mo à eux
  deux). Le dépôt ne suit aujourd'hui que 109 `.wav` au total. À arbitrer avant `git add` —
  hors périmètre technique de cette revue, mais pas hors périmètre du commit demandé.
- **N2 — Fichier parasite servi publiquement** :
  `static/vehicles-assets/halo_infinite/replay/files_list.txt` (18 lignes, simple listing du
  dossier) est un résidu d'outillage exposé sous `/static/`. `index.json` est le manifeste ;
  ce fichier ne sert à rien et n'est pas mentionné par le plan (item A1).
- **N3 — Troisième copie sans garde-rail** (CLAUDE.md n°6) : la boucle « trou ≥ 3 s dans le
  flux d'un slot » existe désormais en trois exemplaires —
  `attachement_phase0_bord_test.go:333`, `vehicules_v1_conducteur_test.go:178`,
  `vehicle_rides.go:198-206` ; et le couple « véhicule le plus proche / échantillon le plus
  proche » en deux exemplaires quasi identiques (`attachement_phase0_bord_test.go:198,217` vs
  `vehicle_rides.go:213,237`). Aucun garde-rail n'interdit la divergence. La production doit
  rester alignée sur l'instrument, sinon les chiffres publiés cessent d'être comparables aux
  rapports qu'ils citent.
- **N4 — Indirection à retour mort** : `boardRefs` (`event_list.go:250-255`) a un unique
  appelant (`decodeBoardRefs:260`) et son retour du milieu (`r1`) n'est lu nulle part dans le
  dépôt.
- **N5 — Sémantique d'absence inatteignable** : `coverage.go:194-200` et
  `document_vehicles.go:181-183` documentent que l'ABSENCE de `coverage.vehicles` dit
  « l'appelant n'a rien fourni à lire » ; `attachVehicles` (`build_vehicles.go:152`) pose
  toujours le pointeur dans `BuildFromPositions` — le cas décrit ne peut pas se produire.
  (Même patron que `GroundWeaponItems` : dette de commentaire, pas de code.)
- **N6 — Couverture de test** : les fonctions PURES du calque web sont bien couvertes
  (`vehiclesLayer.test.ts`, 24 cas ; `replayDraw.test.ts`, 11 cas dont la trace exacte de la
  passe `multiply`). Ne sont couverts par AUCUN test : `drawVehiclesLayer`,
  `drawVehicleOccupantNames`, `drawUnknownVehicleMarker` (tracé canvas) ; et le golden
  `assembly_000d5950` n'exerce pas le calque (`opt.Vehicles` reste `Scanned:false`), donc
  aucune régression de FORME du document véhicule ne serait détectée par le golden.
- **N7 — `fmt.Print*` dans les outils CLI** : les nouveaux fichiers `cmd/weapon-sounds/*`
  (hirc_texte 23, remonter_banque 5, mesure_wav 3, pck_banques 3, pck_dump 3, rendu_event 2,
  hirc_event 1) et `cmd/vs-measure/{sprite,rot180}.go` (5, 2) en font usage. C'est le patron
  DÉJÀ en place dans ces binaires hors ligne (`tir.go` : 7) — pas une régression, mais la
  règle 3 de CLAUDE.md ne fait pas d'exception écrite pour `cmd/`.

---

## 3. Verdict

**GO AVEC RÉSERVES** — le cœur technique du lot tient : 24 conditions vérifiées sur pièces,
tous les gates rejoués verts (Go et web), la géométrie de rotation et la composition
`multiply` sont mathématiquement correctes et testées, la chaîne d'assets résout de bout en
bout, la compatibilité des artefacts v28 n'est pas cassée, et le faux positif le plus évident
du prédicat d'occupation (la mort près d'un véhicule) est réfuté par la mesure du dépôt sur la
même primitive.

Ce qui doit être traité **avant le `git add`** :

1. **B1** — sortir `.gocache-agentA/` (167 Mo) du périmètre du commit / l'ajouter au
   `.gitignore`. Non négociable : c'est irréversible.
2. **M3** — `"inconnue"` → identifiant anglais, tant que rien n'est cuit en v29.
3. **M1** — borner les épisodes à la fenêtre de leur véhicule (ou conditionner le prédicat) :
   c'est le seul constat qui produit un rendu FAUX (joueur invisible).
4. **M2** — aligner la réserve i18n FR/EN sur le comportement réel (`t1max`), ou l'inverse.
5. **M5 / M6** — jouer et consigner le gate C6, statuer les 23 items du plan, écrire les
   entrées de journal des lots A/B/C. Sans ça, `delivery-checklist` ne peut pas passer.
6. **M4** — à ESCALADER à l'utilisateur, pas à corriger d'autorité : la décision de cadrage
   « PION EMBARQUÉ » ne dit rien du cas « calque éteint », et le comportement actuel fait
   disparaître des joueurs sans recours.
7. **N1** — arbitrer les 280 Mo de WAV avant de les inclure au même commit.

Les MINEUR m1-m6 et les NOTE N2-N7 sont consignables (Découvertes / thought_log) sans bloquer,
à l'exception de m2 si un second titre est prévu à court terme.

Bornes de boucle (skill §8) : ceci est la RONDE 1. La ronde 2 ne doit relire QUE les
corrections apportées, par un contexte frais, et le total P0+P1 doit décroître strictement.
