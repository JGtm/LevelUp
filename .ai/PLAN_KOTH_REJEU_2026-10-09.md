# PLAN — Roi de la colline au rejeu : une colline, sa capture, sa garde, son étage (2026-10-09)

Branche `feat/koth-colline` (worktree `LevelUp-wt-koth`, depuis `feat/v75`). Exécution sous le
contrat `plan-execution`. Signalement de l'utilisateur du 2026-10-09 sur son 2v2 Roi de la colline
(`0d9a9af9`, `Doubles:King of the Hill`, Forest, 3-1).

## Objectif

1. Une seule colline visible : la colline active, à sa vraie place.
2. La capture d'une colline se voit (jauge qui monte, ou qui se vide quand le camp tenant la perd).
3. La barre de garde par camp apparaît au score sur Doubles et Classé (seuils MESURÉS).
4. La colline active dit son étage, avec le composant des autres zones.

## Constats sur pièces (avant tout code)

- **C1 — la jauge de capture de colline EXISTE dans le film** (tag 3 du bloc nommé de l'objet de
  mode, slot `jauge`). Forme mesurée sur `0d9a9af9`, `01e1f945`, `21ece4d8` : montée de +0,1 par
  100 ms (0 -> 0,97/0,98 en 0,9-1,0 s) menée par le camp que le canal POUSSEUR nomme, puis retour à
  0 et bascule du PROPRIÉTAIRE (oracle premier contact -> première prise : 0,92 / 0,92 / 0,94 s) ;
  sur une colline TENUE, la jauge saute à 0,983 et se VIDE en ~1 s (pas de 16 ms), le propriétaire
  passe au neutre à 0 ; montée interrompue = la jauge redescend (contestation). Comptes : 30 / 33 /
  60 rampes et 1 229 / 1 395 / 2 732 pas descendants non nuls. Classé (`26602661`, `7de0b91d`) :
  **0 émission de jauge**, prise instantanée (0,00 s) — pas de capture en Classé. Squad
  (`e449a696`) : jauge présente (28 rampes, 956 pas descendants). Le commentaire « EN KOTH, RIEN »
  confondait « compteur de transfert d'une seconde » et absence de capture : c'est la capture.
- **C2 — la période 3 du témoin est mal placée.** Les votes de production (positions pendant les
  « rampes ») prennent la rampe ouverte au retour à zéro de la capture précédente : la période 4 a
  voté sur [373,5 s ; 383,5 s], quand les joueurs sont encore à la colline 3. Témoin indépendant
  (présence du camp PROPRIÉTAIRE dans une zone aux frames où le canal le dit propriétaire, en
  frames) sur le témoin : P1 z3 632/762, P2 z5 618/764, P3 **z4 658/690** (production : z2),
  P4 z2 410/440. Sur 8 autres films KOTH (37 périodes) la présence du gagnant vaut 64 à 100 % des
  frames tenues ; deux périodes sans colline au catalogue tombent à 25 et 27 %.
- **C3 — seuils de garde mesurés (`comp 23 A`, union par camp depuis le point précédent)** :
  Doubles `0d9a9af9` 35/35/34/35 ; Classé `5acb0e0a` 40×5, `f75e7053` 40/40/39/40/40, `84c2221e`
  40×4, `7f172b20` 40/40/40/40/40/39/40 ; Arène `01e1f945` 35×5 (contrôle). Squad : le seul film
  en cache (`e449a696`) est un match à 180 points (score à la seconde), la mesure n'y a pas de
  sens ; non déclaré. Cibles (registre) : Doubles 17/22 matchs finissent à 3 (le reste < 3),
  Squad 56/57 à 3.
- **C4 — étage** : le calque statique pose déjà les contours d'étage (`drawZoneFloorContours`) ;
  sur une colline, la peinture vivante les recouvre et six collines grises se superposent.


## Contrat d'exécution (revue `plan-review` du 2026-10-09)

- Contrat par défaut : skill `plan-execution`. Ordre strict E1 -> E6 ; une étape est CLOSE quand
  son gate a tourné vert, chaque item est statué (`[x]` fait / `[~]` couvert, référence / `[!]`
  non traité, justification au journal) et le plan est à jour. Aucune case vide à la clôture.
- Reprise : relire ce fichier (cases + Journal), reprendre à la première case non statuée.
- Zéro correctif hors périmètre : section « Découvertes ».
- Décisions tranchées avant exécution : (a) avant le premier contact avec l'objet de mode, AUCUNE
  colline n'est peinte (le film ne date pas l'activation de la première, cf. `zone_states_hill.go`) ;
  (b) une période de garde sans colline nette au catalogue n'est PAS publiée (écartée et
  comptée) plutôt que posée sur une colline plausible ; (c) le seuil de garde d'une variante sans
  film mesurable reste non déclaré (pas de jauge) ; (d) aucune chaîne d'interface nouvelle n'est
  prévue (la barre et la jauge existent) — si une étiquette apparaît, FR et EN dans `i18n.ts`.
- Multi-titre : tout vit dans le producteur Halo Infinite (`film/replay`) et dans le rendu générique
  du rejeu, qui décide sur le DOCUMENT (`coverage.zones.hillPeriods`), jamais sur un slug.
- Effort : E1 moyen, E2 lourd (contrat de document), E3 rapide, E4 moyen, E5 moyen, E6 rapide.
- Cuisson : un film à la fois, cache isolé dans le worktree (copie de la base partagée, jamais la
  base tenue par le serveur). Une recuisson du parc KOTH n'est PAS faite ici : elle est signalée.

## Gates (commandes exactes, depuis `apps/go-api` sauf mention)

- E1/E2 : `go test ./internal/games/halo_infinite/film/replay/ ./internal/games/halo_infinite/film/internal/facts/fallback/ ./internal/domain/replaydoc/ ./internal/service/replayview/ ./internal/replaybuild/ ./internal/archlint/`
- E2 : `make openapi-gen` puis `make generate-types` (racine), `git diff --stat` sur `openapi.yaml` / `generated.ts`
- E3 : `go test ./internal/games/mappings/`
- E4 : `make check-types` ; `npx vitest run src/features/match-replay` (apps/web) ; `npx eslint` des fichiers touchés
- E5 : `node <scratchpad>/controle_koth.js <artefact>` sur chaque film recuit
- E6 : `golangci-lint run` des paquets touchés, puis push et CI suivie au niveau job

## Étapes

### E1 — Producteur : la colline se place par la garde (règle générale)
- [x] `hillGardeOf` + `hillLocator.place` (`zone_states_hill_garde.go`, `zone_states_hill.go`) : présence du camp propriétaire par frame tenue ; retenue si la zone en
  tête est NETTE (`clearModalZone`) et couvre au moins 50 % des frames tenues ; sous 50 % la
  période est ÉCARTÉE et comptée (`unpaired`) ; sans garde lisible (aucune frame tenue avec piste
  de camp), repli NOMMÉ vers les votes de jauge actuels.
- [x] Repli au registre (`repli_colline_votes_sans_garde`), ancre, critère de retrait.
- [x] (`zone_states_hill_garde_test.go`, 3 cas) Tests : placement par la garde, seuil 50 %, repli.
- Gate : `go test ./internal/games/halo_infinite/film/...` (paquets touchés), mesure par période
  rejouée sur les 9 films (0 période placée hors de la colline que la garde désigne).

### E2 — Producteur : la jauge de capture de la colline publiée (schéma 92)
- [x] `hillGaugeInputOf` + `attachHillGauges` + `hillGaugeSegments` (`zone_states_hill_gauge.go`, test `zone_states_hill_gauge_test.go`) : émissions de la jauge du bloc dans les périodes de chaque colline, allégées
  (même allègement que Bastion) ; segments `gaugeRamps` par pousseur LU (camp nommé = capteur ;
  neutre et jauge qui baisse = `draining`, le camp tenant perd la colline).
- [x] Champ `draining` (Go, replaydoc, conversion, OpenAPI, types web), `SchemaVersion` 91 -> 92,
  chronique, garde-fous de forme.
- [x] Commentaires faux corrigés côté Go (`zone_states_gauge.go`, `hill_hold_ticks.go`, `zone_states_hill.go`, `document_zones.go`) ; TOML [~] E3 ; web [~] E4.
- Gate : tests Go replay + replaydoc + replayview, `make openapi-gen`, `make generate-types`.

### E3 — Régulation : seuils et cibles mesurés
- [x] `[hold_ticks_per_point]` : `Doubles:King of the Hill = 35`, `Ranked:King of the Hill = 40`.
- [x] `[score_target]` : `Doubles:King of the Hill = 3`, `Squad:King of the Hill = 3`.
- [!] Squad sans seuil de garde : aucun film Squad mesurable en cache (le seul est au score à la seconde, D3) ; non déclaré, donc pas de barre de garde en Squad. À reprendre sur un film Squad à 3 points.
- Gate : tests `internal/games/mappings`.

### E4 — Web : une colline, sa capture, son étage
- [x] (`staticObjectivesOf`, `useZoneStates.staticElements`) Calque statique : les zones d'un document à collines ne s'y dessinent plus (le calque
  vivant les porte) ; Bastion / Total Control inchangés.
- [x] (`paintZoneAt`, `gaugeRampAt`, `isHillState`) Calque vivant : la colline active porte ses contours d'étage (réemploi de
  `drawZoneFloorContours`) ; la jauge qui se vide prend l'encre du camp qui tenait.
- [x] (`zoneSoundEvents`) Son : les rampes de jauge d'une colline ne déclenchent pas les sons de capture de Bastion
  (la colline garde sa grammaire sonore).
- [x] (`hillHoldLogic.test.ts`, 3 cas) Barre de garde : comportement vérifié par test sur la mécanique décrite (ne descend pas,
  reprend, vide au point).
- Gate : `make check-types`, vitest des fichiers touchés, lint web.

### E5 — Recuisson et contrôle sur artefacts (un film à la fois)
- [ ] Témoin `0d9a9af9`, Classé `26602661` `5acb0e0a` `f75e7053` `7de0b91d`, Arène `01e1f945`
  `21ece4d8` `606d9844`.
- [ ] Comptes : collines actives par frame (au plus 1), placement par période, points de jauge,
  barre de garde aux instants de point.
- Gate : script de contrôle sur artefacts (sortie consignée au journal).

### E6 — Livraison
- [ ] Gates complets (Go concernés, lint, check-types, vitest), journal, commit(s), push, CI.

## Critères de succès mesurables

| Point | Critère |
|---|---|
| 1 | Au plus une colline peinte par frame sur les artefacts recuits ; témoin P3 = z4, P4 = z2 |
| 2 | Témoin : `gaugePoints` > 0, rampes avec camp lu ; jauge 0 -> ~1 en ~1 s autour de chaque prise |
| 3 | Témoin : `holdTicksPerPoint` = 35, `targetScore` = 3 ; Classé : 40 et 4 |
| 4 | La colline active porte `floorInRange(z)` contours, test vitest dédié |

## Journal

- **E1 close (2026-10-09)**. Placement par la garde, repli `repli_colline_votes_sans_garde` au registre (ancre de `repli_colline_votes_periode_entiere` suivie). Gate : `go test` replay (34,9 s), fallback, archlint (67 s) verts. Recuisson de 11 films KOTH (cache isole) : 0 declenchement du repli sans garde, `unpaired` 0, au plus 1 colline active par frame, et plus AUCUNE periode fusionnee avec la suivante (base : 4 films sur 11 en avaient). Temoin : P3 z4 [2467-3590], P4 z2 [3591-4198] (base : z2 sur les deux). Controle independant : les trois films classes de Lattice (`26602661`, `5acb0e0a`, `7de0b91d`) rendent la MEME sequence de collines z3, z4, z0, z2, z1 ; la base posait la 4e periode de `5acb0e0a` sur z4.

- **E2 close (2026-10-09)**. Jauge des collines publiee (serie allegee par periode active, segments a pousseur constant, champ `draining`), schema 91 -> 92 (chronique v92, plafonds de `document_chronicle.go` 3181 -> 3208 et `structure_test.go` 1430 -> 1434 par l exception ecrite du ratchet de taille), goldens d assemblage et de forme, fixtures de contrat web regenerees (8, schema 92), `openapi.yaml` (+2 lignes) et `generated.ts` (+1). Gate : `go test` replay 37 s, archlint 70 s, replayview, replaybuild, `internal/api -run OpenAPI` verts. Recuisson du temoin : 531 points de jauge, 48 segments (24 prises au camp lu, 20 vidanges, 4 sans camp), `depuis_les_faits=true` (faits inchanges). Premiere prise du temoin : 0,067 -> 0,967 en 10 frames (1 s) puis 0.

- **E3 close (2026-10-09)**. `[hold_ticks_per_point]` Doubles 35, Classe 40 ; `[score_target]` Doubles 3, Squad 3 (registre + regle enoncee par le user pour le 2v2, deux sources citees). Le garde `TestRadarRangeM_TableLivree` exige que toute variante des autres tables ait sa portee radar : Doubles et Squad entrent a 18 m par la regle d affectation ecrite (hors BTB = 18 m, utilisateur 2026-09-05) — consequence : leurs matchs entrent dans la lecture « ou je meurs isole » de Tactique. Commentaires TOML et `loader_regulation.go` corriges (la prise de colline est une capture courte, instantanee en classe). Gate : `go test ./internal/games/mappings/` vert.

- **E4 close (2026-10-09)**. Web : le calque statique ne dessine plus les zones d un document a collines (Bastion et Total Control inchanges, test) ; le calque vivant peint la colline seulement pendant ses intervalles (jauge comprise), avec ses contours d etage (`drawZoneFloorContours` exporte, aucun ajout de ligne a `objectivesLayer.ts`, 519 L gele) et la vidange a l encre du camp tenant ; le son ignore la jauge des collines. Commentaires faux corriges (`zoneStatesLayer.ts`, `zoneSound.ts`, `hillHoldLogic.ts`, `types.ts`). Aucune chaine d interface neuve. Gate : `npm run typecheck` vert, eslint des 11 fichiers touches vert, vitest `src/features/match-replay src/lib/replay src/lib/api` 250 fichiers / 3 590 tests verts (dont `zoneStatesHill.test.ts` 8 cas, fixtures de contrat schema 92).

## Découvertes (hors périmètre, non traitées)

- D1 — Courbe de score de deux films classés de Lattice : `26602661` publie un point « camp 1 -> 0 »
  à la frame 2268 (descente du score), `7de0b91d` saute un point (camp 1 de 1 à 2 à 5346 avec 66 tics
  de garde dans l'intervalle, puis « camp 0 -> 0 » à 5414). Le seuil de garde (40) se lit sur les
  intervalles sains ; l'anomalie de la courbe n'est pas traitée ici.
- D2 — Cartes absentes des bornes de quantification : `Harvest` (Squad) et `Vacancy - Ranked`
  (Classé, `7f172b20`, `84c2221e`) — cuisson « carte hors catalogue ».
- D3 — Le seul film Squad en cache (`e449a696`, 18/09) est un match au score à la SECONDE (180
  points, le seul des 57 Squad du registre) : il ne mesure pas le seuil de garde de la variante.
