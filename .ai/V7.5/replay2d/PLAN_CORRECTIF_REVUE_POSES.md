# PLAN CORRECTIF — revue adversariale du lot « poses d'equipement ti=37 »

> Perimetre FERME : les findings F1-F5 + mineurs de la revue du 2026-08-17.
> Branche `wt/poses-revue-fix`, base `feat/v75` = `0ae609b2c` (inclut le lot poses ET
> le lot « capteur officiel » `c0ff4ae39`). Zero fix hors perimetre.

## F1 — BLOQUANT — `t1` mesure la MISE AU REPOS, pas la disparition

Chaine constatee : `filmdec/projectiles.go splitLives` clot la vie au premier record
portant le composant i18 (`AtRest`) — regle du PROJECTILE (ti=41) appliquee telle quelle
a l'EQUIPEMENT (ti=37) par `ScanFilmWorldObjects`, qui sert les deux archetypes.
Consequence : `EquipmentLifeSpan.T1US` = fin du VOL de l'objet lance, publie comme `t1`
puis affiche « jusqu'a leur disparition ».

Mesure (instrument versionne, 3 films, `EQUIP_CREATION_FILM`) :
1. Le composant i18 de ti=37 dans le REGISTRE du film — est-ce `projectile-at-rest-state` ?
2. Distribution des durees de vie des poses CONFIRMEES, par GlobalID, sous deux regles :
   (a) regle actuelle (coupure au premier i18) ; (b) coupure au seul trou de 250 ms.
3. Confrontation aux durees OFFICIELLES connues : capteur de menaces
   « Sensor Duration: 15 secondes » (source deja citee dans `threatSensor.ts`),
   mur de protection ~10 s.
4. Recherche d'une fin EXPLICITE : le record DEL (`recDel`, `frame_records.go`) est-il
   isolable par balayage bit a bit pour une vie donnee ? Si le denominateur de bruit
   l'interdit, l'ecrire (negatif mesure) plutot que de l'inventer.

- [x] F1.a mesure faite — VERDICT : **NEGATIF sur la fin explicite, POSITIF sur le defaut.**
      Instrument versionne : `filmdec/equipment_life_end_test.go`.
      1. `ti=37 i18` = `item-at-rest-component` (et `ti=41 i18` = `projectile-at-rest-state`) :
         la coupure `AtRest` a un sens pour les DEUX archetypes. Elle n'est pourtant PAS la
         cause : distributions de durees IDENTIQUES avec et sans elle sur les 2 films qui
         publient. Ce qui borne la vie, c'est le trou de 250 ms du flux de POSITION — le
         decodeur ne suit que les records porteurs d'i0, et un objet pose cesse d'en emettre.
      2. `t1` n'est PAS la disparition, et c'est PROUVE : le recensement des keyframes
         (walker durci, 249/250) montre **101 poses sur 295** (`000d5950`) et **228 sur 537**
         (`00ba2e1c`) encore recensees plus d'une seconde apres la fin de leur flux de position.
      3. Aucune fin explicite n'est isolable : record DEL balaye bit a bit = **78 090** et
         **158 098** candidats (pour 477 et 993 vies) ; queue de records de la meme cle sans
         contrainte de position = **98,0 %** et **99,4 %** des cles SANS aucune vie en portent.
         Le recensement keyframe, lui, est vrai mais espace de **20,0 s** : il prouve la survie,
         il ne date pas la mort.
      Durees mesurees (a = production, b = sans la coupure i18 — identiques) :
      mur `0x8e2dc574` med 0,68 s (n=13) · capteur `0x72199cba` med 2,09 s (n=19) sur
      `000d5950` ; `00ba2e1c` : 12 identifiants, med 0,47 a 4,20 s. `0014603f` ne calibre pas.
- [x] F1.b branche NEGATIVE appliquee : `t1` garde sa valeur (aucune re-cuisson) et est
      redocumente PARTOUT comme la MISE AU REPOS, borne INFERIEURE de la duree de vie
- [x] F1.c doc corrigee : `filmdec/{projectiles,equipment_creation_width,equipment_placements}.go`,
      `replay/{equipment_placements,document}.go`, i18n FR+EN (`layerPlacementsHint`).
      Contrat INCHANGE (openapi ne porte pas les commentaires Go, aucune forme de type modifiee)
- [x] F1.d calque web : `placementEndFrame` — le capteur se tient a sa duree OFFICIELLE (15 s,
      meme citation Waypoint que la portee et la cadence deja en tete de `threatSensor.ts`), les
      autres poses vont jusqu'a la derniere image du rejeu ; jamais avant `t1` (borne mesuree)
- [~] F1.e aucune re-cuisson : la branche negative ne change AUCUNE valeur publiee

Gate : `go test ./internal/analysis/filmdec/ ./internal/analysis/replay/ ./contracttest/...`,
instrument rejoue sur les 3 films, `git status` sans divergence openapi/generated.

## F4/F5 — code mort et commentaire perime (`filmdec/default_state.go`)

- [x] F4 `SetMPPLeadBits` / `MPPLeadBits()` supprimes — grep 0 occurrence dans tout le depot
- [x] F5 commentaire de `mppLeadBits` reecrit vers `CalibrateMPPWidths`

## F2 — familles `wall/sensor/other` enumerees en 3 endroits sans garde-rail

Sources : `internal/games/mappings/loader_replay_labels.go` (`equipmentFamilies`),
`apps/web/.../equipmentPlacementsLayer.ts` (`PLACEMENT_RENDER`),
`apps/web/.../i18n.ts` (`placementFamily`).

- [x] F2 `placementFamily.guard.test.ts` (4 assertions, lit `loader_replay_labels.go` et le TOML) :
      les trois enumerations coincident. Gate : `npx vitest run src/features/match-replay`

## F3 — `ReplayCanvas.tsx` 942 L > 861 L d'avant le lot

- [x] F3 les QUATRE cuissons hors ecran (sol, zones, chaleur, objectifs) sortent dans
      `useReplayStaticLayers.ts` (142 L) : une seule amorce `cookLayer`, le cadrage partage
      `canvasView` au lieu de 4 recopies. **952 -> 860 L**, sous le niveau d'avant le lot.
      Le redessin passe par `drawRef` (les calques se declarent avant `draw`).
      Cliquet pose : `placementFamily.guard.test.ts` refuse au-dela de 861 L.
      Gates : `npx tsc -b --force` vert, `npx eslint src/features/match-replay` 0 probleme,
      `npx vitest run src/features/match-replay` 44 fichiers / 615 tests verts

## Mineurs

- [x] F7 la distinction est PUBLIEE : `EquipmentPlacementStats.Scanned` (filmdec) ->
      `coverage.placements.scanned` (replay). `calibrated: false` couvrait deux pannes ;
      `scanned: false` = film illisible (ou assemblage sans film), `scanned: true` +
      `calibrated: false` = calibration qui refuse de trancher. Aucune bosse de SchemaVersion
      (champ additif). `make openapi-gen` + `make generate-types` joues dans le meme commit
- [x] F10 ligne de registre ecrite (re-cuisson de MASSE du schema 9, 3 temoins seulement).
      AUCUN run de masse lance
- [!] F6/F8/F9/F11 : **contenu non transmis**. Le brief les designe par leur seul numero, et
      le rapport de revue du 2026-08-17 n'existe nulle part dans le depot (grep sur `.ai/` du
      worktree, du principal et de `LevelUp-wt-review`). Ecrire une ligne de registre
      « avec fichier:ligne » exigerait d'inventer le constat. A REPRENDRE : le superviseur
      transmet les quatre enonces, les lignes s'ecrivent alors en une passe
- [x] Note TOML « diagonale FRAGILE + les 4 plats = les grenades `gggl` » : NOTE DE REVUE
      ajoutee a `replay_labels.toml` + ligne de registre. Seuil et critere INCHANGES

## Journal d'execution

| # | commit | ce qui est fait |
|---|---|---|
| F1 | `7b5e36d18` | mesure (3 films) : la disparition n'est PAS dans le film. `t1` redocumente partout, le calque cesse d'effacer a `t1` |
| F4/F5 | `e95235886` | `SetMPPLeadBits`/`MPPLeadBits()` supprimes, commentaire vers `CalibrateMPPWidths` |
| F2 | `883a9edd7` | `placementFamily.guard.test.ts` — les 4 enumerations de familles tenues ensemble |
| F3 | `370fb0ffb` | `useReplayStaticLayers.ts` : 952 -> 860 L, cliquet a 861 |
| mineurs | (ce commit) | F7 publie (`coverage.placements.scanned`), F10 + note TOML au registre, F6/F8/F9/F11 `[!]` faute d'enonce |
