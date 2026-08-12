# Plan — Fiches joueur enrichies du rejeu 2D (arme en main, vitals, capacite, grenades, drops)

> Branche cible : `feat/v75` (mode branche unique v7.5 ; JAMAIS de merge vers `main` avant le
> tag). Le rejeu reste OFF EN PROD : le garde `handlers/replay_local_gate.go` n'est PAS touche.
> Contrat d'execution : skill `plan-execution` (ordre strict, une etape a la fois, aucun item
> sans statut a la cloture, zero fix hors perimetre). Reprise : lire la section « Suivi » en bas.

## Objectif et critere de succes

Enrichir les fiches joueur de la page replay (`features/match-replay/ReplayTeams.tsx`) avec, par
joueur et a l'instant lu : la **sante**, l'**arme en main** (a gauche) / **secondaire** (a droite),
la **capacite d'armure equipee**, les **grenades portees** — et un calque **armes au sol (drops)**.

**Critere de succes** : sur le match `000d5950` (Cliffhanger, seul artefact sur disque), gate
visuel utilisateur PASSE ; chaque donnee affichee est soit une mesure du film soit une LACUNE
explicite (jamais un zero ni une valeur par defaut) ; parite FR+EN ; zero couleur en dur ; CI de
branche verte au niveau job.

**Contrainte non negociable** : OFFLINE-PUR. Aucune donnee ne doit dependre d'une capture Cheat
Engine ni d'un acces online. Tout ce qui n'est decodable que par le walk delta calibre CE est
HORS PERIMETRE (voir Phase 3, renvoyee post-v7.5).

**Multi-titre** : le decodage film est **Halo Infinite uniquement**. La feature se degrade par
ABSENCE DE DONNEE (les champs `hp`/`d`/`a`/`g` sont simplement absents pour un autre titre ou un
match sans film) : le front n'affiche alors rien pour ces lignes, exactement comme le bouclier
absent aujourd'hui. Aucune comparaison de slug ; aucune panic ; aucune donnee d'un autre titre.

---

## Etat de l'art VERIFIE sur pieces (investigation 7 agents, 2026-08-12)

> Le doc `.ai/ETAT_DU_POC.md` (2026-07-27) est PERIME (i43/i42 y sont dits « non decodes/non
> cables » — FAUX au regard du code actuel). Ce tableau fait foi ; il est mesure sur l'artefact
> `data/cache/replays/halo_infinite/000d5950.json` et cite le code de `feat/v75`.

| Donnee | Decodee offline | Extraite/publiee | Cadence | Couverture 000d5950 | Statut |
|---|---|---|---|---|---|
| Bouclier (i5) | oui | oui (`Point.Sh`) | par-record, AU CHANGEMENT | 15,81 % pts / 73,7 % vies | **DEJA AFFICHE** (fiche+canvas) |
| Sante (i4) | oui | oui (`Point.Hp`) | par-record, AU CHANGEMENT | 0,56 % pts / 32,3 % vies | publie, **0 consommateur web** |
| Arme portee (paire) | oui | oui (`Loadout.W`) | keyframe ~20 s | 150 loadouts / 24 kf | affiche (liste) |
| Arme EN MAIN (drawn slot i42) | oui | oui (`Inventory.D`) | keyframe ~20 s | 150/184 ; {0:70,1:70,2:10} | publie, **pas mis en valeur** |
| Capacite equipee (index i48) | oui | oui (`Inventory.A` + `abilityLabels`) | keyframe ~20 s | 132/184 (71,7 %) ; table 4/11 | publie, **pas dans la fiche** |
| Grenades comptes (i22) | oui | oui (`Inventory.G` + `grenadeLabels`) | keyframe ~20 s | 120/184 (65,2 %) | publie (`InventoryRow`) |
| Munitions (i30..i42) | oui | oui (`Inventory.Am`) | keyframe ~20 s | 150/184 | **DEJA AFFICHE** |
| Lancers de grenade (events) | oui | oui (`doc.grenades`) | evenement | 70/70 (100 %) | publie, pas dans la fiche |
| Swap d'arme (diff keyframe) | oui | derivable | ~20 s (grossier) | 12-14/70 transitions | non calque |
| Armes au sol / DROPS (ti=42) | identite oui, position portee | NON (filtree) | keyframe ~20 s | 397 occ. familles | **needs_wiring Go** |
| Capacite ACTIVE (i57) | consumed_only | non | delta | — | not_offline dense |
| Grenade SELECTIONNEE / en main (i47) | consumed_only | non | delta | — | not_offline (voir Decisions) |
| Swap fin / arme equipee dense (i43) | not offline (calib CE) | non | delta | — | **Phase 3 (RE, hors v7.5)** |
| Event typé pickup/drop | NON (le film n'en porte pas) | non | — | — | non decodable |
| Compteur d'utilisations capacite | NON localise | non | — | — | non decodable |
| Surbouclier / regen bouclier | consumed_only, semantique non etablie | non | — | 0/4620 >1 | non exploitable |

Points d'ancrage code (verifies) :
- Walk vitals per-record (offline-pur, DEJA cable) : `filmdec/offline_biped.go:159,264` +
  `filmdec/offline_aim.go:125-206` -> peuple `Point.Hp/Sh/H` ; `replay/build.go:345-350`.
- Scan keyframe inventaire (offline-pur, DEJA cable) : `replay/inventory_decode.go:147` ->
  `Inventory{D,A,G,Am}` ; `replay/build.go:143-148` ; contrat `replay/document.go:99-134`.
- Scan keyframe loadout : `filmdec/keyframe_loadout.go:64` ; `replay/loadouts.go:59-97`.
- Front fiches : `web ReplayTeams.tsx` (PlayerCard, ShieldBar, InventoryRow, WeaponsRow) ;
  helper report+fondu `web replayLogic.ts:254-270` (`heldReading`, `freshness`).
- DROPS : identite d'arme au sol deja decodable mais FILTREE `filmdec/keyframe_loadout.go:36-39`
  (`keyframeBipedTI=35` ecarte `ti=42`) ; position via object-position-component (i0 world-object,
  45 bits) `filmdec/traverse.go:229-258`.

---

## Decisions produit TRANCHEES (ne pas les rouvrir en cours d'execution)

1. **Sante** : afficher via le MEME patron que le bouclier (`heldReading` + maintien court +
   fondu de fraicheur + LACUNE si non lue). JAMAIS une jauge pleine par defaut. Raison mesuree :
   mediane 0 echantillon/vie ; une barre permanente serait vide/fausse la plupart du temps, et
   reporter EN ARRIERE (unique mesure = 30 % juste avant la mort) peindrait faux tout le debut de
   vie. La sante est une valeur ABSOLUE repliquee-au-changement, donc le report AVANT est honnete
   (inchangee), le report ARRIERE ne l'est pas. `healthHold` initial = 2000 ms (identique au
   bouclier) ; ajustable au gate visuel. Couleur = token semantique distinct du bouclier.
2. **Arme en main / secondaire** : l'arme EN MAIN (`W[D]` quand `D` ∈ {0,1}) s'affiche A GAUCHE,
   la secondaire (`W[1-D]`) A DROITE, comme dans le jeu. `D=2` (rien degaine, typiquement 1er
   keyframe) = pas de mise en valeur, ordre par emplacement. `D` absent/-1 = report de la derniere
   lecture avec son age, ou « non lu ». L'icone d'arme (deja disponible cote kill feed) peut etre
   reutilisee ; a defaut, le libelle. Reutiliser les tokens de couleur d'equipe existants.
3. **Capacite equipee** : afficher le nom (`abilityLabels[a]`) ; index hors table = garder le
   NUMERO marque non interpretable (patron existant `InventoryRow`/`abilityText`). Le COMPTEUR
   d'utilisations n'est PAS affiche (non localise — 36006 positions testees, aucune ne reproduit
   le releve). La capacite ACTIVE (i57) n'est PAS affichee (consumed_only, non publiee).
4. **Grenades** : afficher les COMPTES portes par rang (`Inventory.G`, deja dans `InventoryRow`) +
   optionnellement un marqueur des LANCERS depuis `doc.grenades`. La « grenade EN MAIN /
   selectionnee » n'est PAS affichee : elle n'est PAS decodable offline-pur (i47 consumed_only,
   pas d'archetype porte ; au mieux une inference du dernier lancer, ecartee pour ne pas afficher
   une inference comme une mesure).
5. **Swaps** : deriver du DIFF des loadouts d'un meme slot entre keyframes (granularite ~20 s),
   avec age affiche. PAS de swap fin intra-keyframe (Phase 3). Marquer clairement « etat de
   reference, ~20 s », jamais un suivi continu.
6. **Drops (armes au sol)** : calque OPTIONNEL Phase 2 (Go). Recommandation : le faire APRES le
   gate de la Phase 1, decision d'inclusion v7.5 par l'utilisateur sur le chiffre de couverture.
7. **Cadence honnete partout** : vitals = au changement ; loadout/arme-en-main/capacite/grenades
   = keyframe ~20 s avec AGE affiche et fondu. Toute valeur non lue = LACUNE, jamais un defaut.
8. **HORS PERIMETRE v7.5, renvoye Phase 3 / registre** : suivi dense per-record (arme equipee au
   swap fin, capacite active continue, grenade selectionnee) — bloque par le walk delta non
   offline-pur (calibration CE i0/i21 + faute de corps i22). C'est un chantier de RE, pas un
   cablage. Aucun executeur ne doit s'y engager dans ce lot.

---

## Phase 1 — Cablage web des fiches enrichies (offline-pur, DONNEE DEJA PUBLIEE)

> Effort : MOYEN, front uniquement (aucune ligne Go de decodage — tout est deja dans l'artefact
> et le contrat OpenAPI genere : `Point.hp/sh`, `Inventory.d/a/g/am`, `doc.grenades`). Livrable
> independamment. Perimetre FERME ci-dessous.

- [ ] **1.1 Barre de sante** — ajouter `health: {value, age} | null` a `PlayerState`
  (`rosterLogic.ts`), peuple par `heldReading(live.points, frame, p => p.hp, HEALTH_HOLD)` ;
  composant `HealthBar` calque sur `ShieldBar` (fondu `freshness`, LACUNE si null, token couleur
  sante). JAMAIS 100 % par defaut. `HEALTH_HOLD = 2000` ms initial.
  - Gate : `grep -n "p.hp" apps/web/src/features/match-replay/*.ts*` retourne >=1 lecteur ;
    `make check-types` ; `make test-web` (nouveau test `rosterLogic` sur report+lacune sante).
- [ ] **1.2 Arme en main a gauche / secondaire a droite** — dans `WeaponsRow` (ou nouveau
  `EquippedWeaponsRow`), ordonner par `Inventory.D` : `W[D]` a gauche marquee « en main »,
  `W[1-D]` a droite ; `D=2`/absent gere selon Decision 2. Reutiliser l'icone d'arme si dispo.
  - Gate : test `killFeedLogic`/nouveau `equippedLogic` sur les 3 cas (`D=0`, `D=1`, `D=2`) ;
    `make test-web`.
- [ ] **1.3 Capacite equipee dans la fiche** — afficher `abilityLabels[Inventory.A]` (nom
  bilingue), index hors table -> numero marque non interpretable. Reutiliser `abilityText`.
  - Gate : test rendu capacite connue + inconnue ; `make test-web`.
- [ ] **1.4 Grenades portees** — verifier que `InventoryRow` rend deja `Inventory.G` par rang ;
  si non, l'ajouter (labels `grenadeLabels`). Optionnel : pastille de LANCER depuis `doc.grenades`.
  - Gate : test rendu comptes + lacune (`GrenadesRead=false`) ; `make test-web`.
- [ ] **1.5 Swaps (diff keyframe)** — deriver cote client le changement de `Loadout.W` d'un meme
  slot entre deux keyframes ; indicateur discret avec AGE. PAS de calque serveur.
  - Gate : test diff sur 2 loadouts same-slot ; `make test-web`.
- [ ] **1.6 i18n FR+EN** — toutes les strings nouvelles dans `match-replay/i18n.ts` (parite par
  typage `Record<Locale, T>`). Libelles « en main », « secondaire », « Sante », « Capacite ».
  - Gate : `make check-types` ; lint i18n vert.
- [ ] **1.7 Couleurs** — tokens semantiques uniquement (skill `color-tokens`), zero hex/Tailwind
  couleur.
  - Gate : `grep -rnE "#[0-9a-fA-F]{3,6}|bg-(red|blue|green)" apps/web/src/features/match-replay/` = vide.
- [ ] **1.8 Degradation multi-titre** — verifier qu'un artefact sans `hp`/`d`/`a` (ou H5) rend la
  fiche SANS ces lignes, sans erreur (les champs sont deja `?:` dans le contrat genere).
  - Gate : test rendu fiche avec `Point` sans `hp` et `Inventory` sans `d/a`.

**Gate de Phase 1** : `make check-types` + `make test-web` verts ; gate visuel utilisateur sur
`000d5950` (temoins choisis par l'utilisateur) ; entree `thought_log.md`.

---

## Phase 2 — Calque DROPS (armes au sol) — offline-pur, cablage Go borne

> Effort : MOYEN (Go : filmdec + replay + contrat + web). A faire APRES le gate Phase 1.
> Decision d'inclusion v7.5 par l'utilisateur sur le chiffre de couverture mesure en 2.1.

- [ ] **2.1 Recolter l'identite + la position des armes au sol (ti=42)** au keyframe : lever le
  filtre `keyframeBipedTI=35` (`filmdec/keyframe_loadout.go:36-39`) pour COLLECTER aussi les
  familles `ti=42` (397 occ. mesurees) et leur position (object-position-component i0 world-object,
  45 bits, deja porte `traverse.go:229-258`). Nouveau scan `ScanFilmKeyframeGroundWeapons`
  (analysis/filmdec), pur, sans capture. MESURER la couverture sur `000d5950` et la consigner.
  - Gate : `cd apps/go-api && go test ./internal/analysis/filmdec/ -run GroundWeapon` vert ;
    couverture consignee dans le CR.
- [ ] **2.2 Calque document** — nouveau champ optionnel `GroundWeapons []GroundWeapon` dans
  `ReplayDocument` (`replay/document.go`, `omitempty`, symetrique de `Loadouts`) ; `build*` +
  `keepOfPublishedTracks` non requis (objets, pas tracks) ; cablage `replay/build.go:143` (scan) +
  `:265` (publish). Regen contrat : `go run ./cmd/openapi-gen` + `make generate-types`.
  - Gate : `go test ./internal/analysis/replay/` ; `make generate-types` sans diff non commite.
- [ ] **2.3 Rendu web** — calque discret « armes au sol » sur le canvas (`ReplayCanvas`/
  `replayMarkers.ts`), style neutre, sous les joueurs. i18n + tokens.
  - Gate : `make test-web` ; gate visuel.

**Gate de Phase 2** : tests Go + web verts ; `make gate-push` (filet local) ; gate visuel ; entree
`thought_log.md`. Si l'utilisateur juge la couverture trop faible, Phase 2 est renvoyee post-v7.5
(consigner au registre).

---

## Phase 3 — HORS v7.5 (registre) : suivi dense per-record

> NE PAS EXECUTER dans ce lot. Consigne ici pour fermer le perimetre.

Le suivi dense per-record (arme equipee au swap fin, capacite active continue, grenade
selectionnee i47/i48, pickup/swap par frame) exige de rendre le walk delta bit-exact `i0..i43`
OFFLINE : resoudre les largeurs de precision `i0/i21` (aujourd'hui issues d'une capture Cheat
Engine, `traverse.go:1199-1209` « NOT a general decode path. Empty by default ») ET corriger la
faute de corps `i22` (92,46 % de comptes impossibles, `frame_records.go:81`). C'est un chantier de
RETRO-INGENIERIE, pas un cablage. -> Registre `.ai/V7.5/REGISTRE_REPORTS.md`, condition de reprise :
« walk delta offline bit-exact jusqu'a i43 ».

**Non decodable (a ne pas chercher)** : evenement typé pickup/drop (le film n'en porte aucun) ;
compteur d'utilisations de capacite (non localise) ; surbouclier/regen bouclier (semantique non
etablie, temoin de recharge echoue).

---

## Decouvertes (a consigner ici pendant l'execution, NE PAS traiter hors perimetre)

- (vide au demarrage)

## Contraintes transverses (rappel grille plan-review)

- Couches Go : decodage pur en `internal/analysis/filmdec` ; assemblage en
  `internal/analysis/replay` ; DTO d'artefact en `replay/document.go` ; aucun acces DB/HTTP.
- Tests par couche : filmdec (unitaire), replay (assemblage sur fixture), web (hooks +
  rendu jsdom). Logging : `slog.*Context` cote Go si nouvelle op significative ; jamais
  `fmt.Println`. Front : routes file-based, query keys dans `lib/query/keys.ts`, i18n FR+EN,
  tokens couleur.
- Offline-pur : aucun chemin ne doit appeler le walk delta calibre CE. Le decodage lourd reste
  hors ligne (I/O disque sur les chunks du film).

## Suivi / reprise de session

- Avancement = les cases de ce fichier + `thought_log.md`. Reprise : lire ce plan, puis
  `git -C <worktree> log --oneline -8 feat/v75`, puis rouvrir la premiere case non cochee.
- Contrat d'execution : skill `plan-execution` (fait foi). Cloture de lot = gate de phase +
  CI de branche verte au niveau job (mode branche unique v7.5, pas de merge).
