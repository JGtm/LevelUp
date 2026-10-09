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
- [x] Gates complets (Go concernés, lint, check-types, vitest), journal, commit(s), push, CI.


### E7 — Reprise (2026-10-09) : la 1re colline apparaît au coup d'envoi
- [x] Mesure de l'instant d'activation sur 14 films (`zone_colline_activation_mesure_test.go`) :
  (a) images-clés, (b) émissions delta, (c) coup d'envoi.
- [x] Règle `hillFirstActivation` (`zone_states_hill_activation.go`) : coup d'envoi ramené dans la
  fenêtre des images-clés (repli nommé `repli_colline_premiere_au_coup_d_envoi`), sinon 1re
  image-clé qui porte le bloc, sinon premier contact ; jamais après le premier contact.
- [x] Tests (`zone_states_hill_activation_test.go`, 3 cas, mutation : 3 échecs sans la règle).
- [x] Collines suivantes vérifiées (début = image du point, écart 0 à 1 image).
- [!] Délai de prise de 5,1 s après chaque déplacement : mesuré, NON appliqué — question de jeu
  posée au user (cf. Journal).
- [x] Chronique v92 amendée (schéma inchangé : 92 n'est publié nulle part), recuisson des 11 films,
  gates, push, CI.

### E8 — Reprise (2026-10-09, demande du user) : le seuil de garde se lit dans le film, match par match
Le seuil (`holdTicksPerPoint`) ne dépend plus d'une table par variante : au point, la garde
accumulée par le camp qui marque depuis le point précédent EST le seuil. L'artefact s'assemble
une fois le film lu : dès qu'un point est marqué, le seuil vaut pour tout le match.
- [x] Règle `resolveHoldThreshold` (`hill_hold_threshold.go`, producteur `film/replay`) : au point, garde
  prise par le camp qui marque depuis le point précédent (intervalle `]point précédent, point]`) ;
  point écarté si un autre camp a gardé autant ou plus (point absent de la courbe) ; statistique =
  valeur la plus fréquente, à égalité la plus haute (un tic se perd sur une garde interrompue,
  jamais un de trop ; le maximum prendrait les intervalles fusionnés, 73 sur `26602661`). Source
  « mesuré dans le film » ; mode à colline SEUL (`ScoreInput.HillScoring`, posé par l'appelant sur
  la variante, même prédicat que la méthode des collines) — Bases et Total Control inchangés (test).
- [x] Match sans point : repli NOMMÉ `repli_seuil_garde_table_de_variante` (registre, compté dans
  `coverage.fallbacks[]`) ; sans entrée, pas de barre. Désaccord film / table : le film prime,
  contradiction comptée `repli_seuil_garde_table_contredite` (CondContradiction, même canal).
- [x] Mode à score en secondes : seuil mesuré 1 → ni série ni seuil publiés (`e449a696` : 199 points
  à 1 tic, 60 écartés, mesuré sans cuisson par la règle de production).
- [x] Appelant : `poserLeReglementDuScore` (`replaybuild/reglement_du_score.go`) seul site (cuisson +
  deux outils de recherche), garde-rail `TestReglementDuScoreUnSeulSite`.
- [x] Commentaires inversés corrigés : `regulation.toml` (dont la ligne fausse sur `7de0b91d`),
  `loader_regulation.go`, `hill_hold_ticks.go`, `document_score.go`, `score_timeline.go`,
  `hillHoldLogic.ts`.
- [x] Contrat : AUCUN champ neuf. Un bloc `coverage.score.holdThreshold` a été écrit puis retiré :
  `TestDocumentShapeMatchesGolden` exige une montée de `SchemaVersion` pour tout changement de forme,
  champ optionnel compris (le 93 est à levelup-57). Repli et contradiction passent par
  `coverage.fallbacks[]` (format existant) ; le détail (points, écartés, table) va au journal de
  cuisson (`logHoldThreshold`). Schéma 92 inchangé, aucune révision de couche touchée.
- [x] Tests Go (`hill_hold_threshold_test.go`, 8 cas ; `reglement_du_score_test.go`, 2 cas) ; mutations :
  règle du film retirée → 6 échecs ; mode remplacé par le maximum → `PointFusionneNeGagnePas` rouge.
- [x] Vérification film par film (cache isolé, binaire du lot) : 13 films cuits (les 11 + `7f172b20`
  et `84c2221e`, Vacancy cuit désormais), `e449a696` mesuré sans cuisson (Harvest hors bornes),
  `8c12fd58` (Gruntpocalypse KOTH) mesuré sans cuisson : aucun tic de garde, pas de barre.
- Gate : `go test` replay / fallback / replaybuild / replayview / mappings / archlint,
  typecheck + vitest + eslint du fichier web touché (commentaires), golangci-lint des paquets
  touchés, push et CI. (`make openapi-gen` / `generate-types` : sans objet, contrat inchangé.)

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

- **E5 close (2026-10-09)**. Recuisson un par un (cache isole du worktree, copie de la base partagee, pic 92 a 279 Mio) des 11 films KOTH a carte bornee. DEUX DEFAUTS TROUVES ET CORRIGES pendant le controle : (1) l allegement de Bastion sautait le retour a zero d une vidange (0,017 -> 0, sous le pas de 0,02) et l escalier tenait la colline a 2 % jusqu au point suivant -> `appendHillGauge` publie tout retour a zero ; (2) une lecture NON chainee isolee publiait une jauge pleine d une seconde sur un film classe (`5acb0e0a`, 1 lecture, 0 chainee ; les vraies lectures de jauge de colline chainent a 99,6-99,9 %, 1 682/1 688 et 3 469/3 474) -> `zoneSeries.gaugeHill` = lectures chainees + retours au repos (le zero est souvent le dernier record du paquet, donc non chaine : le filtrer seul figeait la jauge sur 4 films, 28 a 529 frames). Tests de mutation : les deux regles rougissent leur cas. Controle final (`controle_koth.js`) : 11/11 schema 92, au plus 1 colline active par frame, `unpaired` 0, 0 frame de jauge tenue hors segment ; classe 0 point de jauge (prise instantanee). Oracle (`oracle_prise.js`) : 165 prises de colline sur 7 films arene/Doubles, 165 precedees d une montee du MEME camp finissant a moins de 3 frames, montee mediane 10 frames (1 s), p90 10 a 23 ; classe 67 prises, 0 montee. Barre de garde : temoin 34/35 la frame avant chaque point du camp qui marque (le 35e tic tombe sur la frame du point, ou la barre se vide), classe 39/40.

- **E6 close (2026-10-09)**. `go test ./...` local (CGO) : seul echec `internal/mapdecoupe` `TestOraclePositionsJouees`, qui lit les artefacts du cache LOCAL du worktree (les 11 KOTH recuits ici, 3 cartes reconnues pour 5 exigees) — vert une fois ces artefacts mis de cote, et sans objet en CI (aucun artefact). golangci-lint des paquets touches : 3 constats pre-existants hors fichiers du lot (`mappings` : `loader_outcomes.go`, `loader_endpoints.go`, `registry.go`). Web : typecheck a froid (`node_modules/.tmp` purge), `npm run lint` 0 erreur (26 avertissements pre-existants, aucun dans un fichier du lot), vitest complet 890 fichiers / 9 371 tests verts. Push `feat/koth-colline` : CI `b5d20f3cb` verte au niveau job (Go Linux et Windows, couverture + baseline, lint, contrat OpenAPI, Frontend), gitleaks et Deploy Pre-Check verts. Registre des reports : seuil Squad, courbes de score de Lattice, cartes hors bornes.

**A FAIRE PAR LE SUPERVISEUR (accord du user requis)** : republication du parc au schema 92 (`backfill-replay --only-existing`, verdict `republier`, assemblage seul) — elle concerne TOUS les artefacts (un v91 se lit « a republier ») ; les collines ne changent que sur les 10 artefacts KOTH du parc local (`0d9a9af9`, `26602661`, `f75e7053`, `21ece4d8`, `7f1bbf06`, `a36c8bed`, `5ed17fe3`, `606d9844`, `01e1f945`, `8076f97f`). Fusion dans `feat/v75` non faite (consigne).

- **E7 close (2026-10-09, reprise)**. Activation de la 1re colline, 14 films (instants rapportés au coup d'envoi `t0FilmMs`, frames de 100 ms ; cuisson pour les 11 à carte bornée) :

  | film | variante | image-clé sans le bloc | image-clé avec le bloc (désignation 1re colline) | 1er contact |
  |---|---|---|---|---|
  | 0d9a9af9 | Doubles | -11,2 s | +8,8 s | +17,0 s |
  | 01e1f945 | Arène | -14,1 s | +5,9 s | +15,3 s |
  | 21ece4d8 | Arène | -15,3 s | +4,7 s | +16,8 s |
  | 606d9844 | Arène | -9,7 s | +10,3 s | +15,1 s |
  | 7f1bbf06 | Arène | -6,6 s | +13,4 s | +21,5 s |
  | 8076f97f | Arène | -17,8 s | +2,3 s | +17,3 s |
  | a36c8bed | Arène | -17,6 s | +2,4 s | +15,7 s |
  | 26602661 | Classé | -13,4 s | +6,6 s | +27,6 s |
  | 5acb0e0a | Classé | -11,1 s | +8,9 s | +15,6 s |
  | 7de0b91d | Classé | -13,2 s | +6,8 s | +19,7 s |
  | f75e7053 | Classé | -14,0 s | +6,0 s | +37,2 s |
  | e449a696, 7f172b20, 84c2221e | Squad, Classé x2 | même motif (absent à +20 s, présent à +40 s du 1er paquet ti=13) ; pas de coup d'envoi sans cuisson | | |

  (a) les images-clés (toutes les 20 s) BORNENT la création de l'objet de mode, avec la désignation de la 1re colline (`0xc4c98230`) dès la 1re image-clé qui le porte ; l'intersection des 11 fenêtres est ]-6,6 s ; +2,3 s] autour du coup d'envoi — elles le contiennent toutes. (b) aucune lecture CHAÎNÉE du bloc avant le premier contact (les lectures antérieures sont de la contamination d'ancrage : tags et modes par joueur incohérents) ; la création est un record que l'ancrage ne reconnaît pas. (c) retenu : coup d'envoi ramené dans la fenêtre, repli nommé et compté. Délai d'activation au départ : nul à la précision des images-clés (compatible avec 0 sur les 11 films, au plus +2,3 s), non mesurable plus finement. Recuisson : 11/11 films, 1re période au coup d'envoi (écart 0), repli déclenché 1 fois par film, collines suivantes au point (écart 0 à 1 image) ; placement, jauge et oracle inchangés (165/165). Deux écarts « ? » et « 1932 » sur `26602661` et `7de0b91d` viennent des courbes de score anormales déjà reportées (D1), pas des périodes.

  **Découverte — délai de prise après chaque déplacement** : sur 50 déplacements (14 films), la première émission du bloc (prise, ou montée de jauge) n'arrive JAMAIS avant 5,09 s après la bascule du désignateur, et 13 fois entre 5,09 et 5,19 s (5 fois à 5,09 s exactement en Classé : un joueur posté sur la colline suivante la prend à cet instant). Le film ne porte aucune émission propre à cet instant (balayage de toutes les lectures chaînées de +3 à +5,6 s : rien de récurrent). Ce délai n'est PAS appliqué : il ne se lit que comme un plancher, et ce que le jeu affiche pendant ces 5 s (colline suivante visible avec compte à rebours, ou absente) appartient au user. Question posée.

- **E8 close (2026-10-09, reprise)**. Seuil de garde mesuré dans le film, match par match (règle `resolveHoldThreshold`). Mesure sur pièces avant la règle (gains au point sur les 11 artefacts E7) : le dernier tic du camp qui marque tombe TOUJOURS sur l'image du point, y compris quand le gain vaut 35 ou 40 ; les 34 / 33 / 39 sont des intervalles à garde interrompue (écarts de 2 à 39 s entre deux tics), donc un tic PERDU, jamais un de trop. Au-dessus : intervalles fusionnés par un point absent de la courbe (73 sur `26602661`, 66 contre 77 sur `7de0b91d`). D'où : point écarté si l'autre camp a gardé autant ou plus, puis valeur la plus fréquente, à égalité la plus haute. Résultat par film (binaire du lot, cache isolé, 13 cuissons de 0 à 11 s) :

  | film | variante | seuil du film | table | points retenus / écartés | accord |
  |---|---|---|---|---|---|
  | 0d9a9af9 | Doubles | 35 | 35 | 4 / 0 | oui |
  | 01e1f945 | Arène | 35 | 35 | 5 / 0 | oui |
  | 21ece4d8 | Arène | 35 | 35 | 4 / 0 | oui |
  | 606d9844 | Arène | 35 | 35 | 3 / 0 | oui |
  | 7f1bbf06 | Arène | 35 | 35 | 3 / 0 | oui |
  | 8076f97f | Arène | 35 | 35 | 3 / 0 | oui |
  | a36c8bed | Arène | 35 | 35 | 4 / 0 | oui |
  | 26602661 | Classé | 40 | 40 | 3 / 0 | oui |
  | 5acb0e0a | Classé | 40 | 40 | 5 / 0 | oui |
  | f75e7053 | Classé | 40 | 40 | 5 / 0 | oui |
  | 7f172b20 | Classé (Vacancy) | 40 | 40 | 7 / 0 | oui |
  | 84c2221e | Classé (Vacancy) | 40 | 40 | 4 / 0 | oui |
  | 7de0b91d | Classé | 39 | 40 | 1 / 1 | NON : contradiction comptée, le film prime |
  | e449a696 | Squad (Harvest, sans cuisson) | 1 | absente | 199 / 60 | pas de barre (un tic = un point) |
  | 8c12fd58 | Gruntpocalypse KOTH (sans cuisson) | aucun tic de garde | absente | 0 | pas de barre |

  Barre du camp qui marque au point (garde de l'intervalle / seuil) : 1,00 partout, sauf les intervalles à tic perdu (0,97 sur `0d9a9af9`, `f75e7053`, `7f172b20` ; 0,94 sur `a36c8bed`) ; les intervalles fusionnés de `26602661` et `7de0b91d` dépassent 1 et le client plafonne à 1. Artefacts recuits comparés à ceux d'E7 : identiques sur 10 films ; sur `7de0b91d` seuls `holdTicksPerPoint` (40 -> 39) et l'entrée `repli_seuil_garde_table_contredite` de `coverage.fallbacks` changent. Format : un bloc `coverage.score.holdThreshold` a d'abord été écrit (replaydoc, conversion, OpenAPI, types web), puis retiré : la garde de forme (`TestDocumentShapeMatchesGolden`) exige une montée de schéma pour tout champ, optionnel compris ; repli et contradiction passent par `coverage.fallbacks[]`, le détail par le journal de cuisson. Schéma 92 inchangé, aucune révision de couche touchée. Gates : `go test` replay (35,5 s), fallback, replayview, replaybuild, mappings, archlint (53,8 s) verts ; `go vet -tags research` replay + replaybuild ; golangci-lint des 4 paquets : 3 constats préexistants de `mappings` seulement ; web : `tsc -b` vert, eslint et vitest de `hillHoldLogic` (12 tests) verts.

  **Republication** : rien ne marque les artefacts déjà cuits comme périmés (ni schéma ni révision). Si ce lot est fusionné AVANT la recuisson commune du parc (au 93), celle-ci le couvre sans coût. Sinon : `levelup backfill-replay --one <match_id>` par match KOTH (la CLI n'a pas de filtre par variante ; `--one` cuit sans sauter, et rejoue depuis les faits s'ils sont frais, 0 à 1 s par match) — 23 artefacts KOTH dans le parc local (15 Classé, 7 Arène, 1 Doubles). Les 23 ont été cuits ici par le binaire du lot (les 13 du tableau + `5ed17fe3` Argyle et neuf classés de Vacancy) : 21 rendent la valeur de leur table (35 ou 40), `cef67c66` (film tronqué, 10 pistes) n a aucun tic de garde, comme avant, et `7de0b91d` est le SEUL artefact du parc local dont l affichage change (40 -> 39 et une contradiction comptée).
## Découvertes (hors périmètre, non traitées)

- D1 — Courbe de score de deux films classés de Lattice : `26602661` publie un point « camp 1 -> 0 »
  à la frame 2268 (descente du score), `7de0b91d` saute un point (camp 1 de 1 à 2 à 5346 avec 66 tics
  de garde dans l'intervalle, puis « camp 0 -> 0 » à 5414). Le seuil de garde (40) se lit sur les
  intervalles sains ; l'anomalie de la courbe n'est pas traitée ici.
- D2 — Cartes absentes des bornes de quantification : `Harvest` (Squad) et `Vacancy - Ranked`
  (Classé, `7f172b20`, `84c2221e`) — cuisson « carte hors catalogue ».
- D3 — Le seul film Squad en cache (`e449a696`, 18/09) est un match au score à la SECONDE (180
  points, le seul des 57 Squad du registre) : il ne mesure pas le seuil de garde de la variante.
- D4 (E8) — Au client, la barre du camp qui marque ne s'affiche jamais PLEINE : le dernier tic
  tombe sur l'image du point, et `readHillHold` remet les deux barres à zéro à cette image même
  (mécanique énoncée par le user et tenue par `hillHoldLogic.test.ts`). L'image d'avant montre
  34/35 (39/40), ou 33/35 sur un intervalle à tic perdu. Afficher la barre pleine à l'image du
  point puis la vider à l'image suivante est une décision d'affichage du user, non prise ici.
- D5 (E8) — La garde de mode suit `isHillVariant`, qui classe aussi les variantes Firefight et
  Gruntpocalypse « King of the Hill / KOTH » (15 matchs au registre). Le seul film en cache
  (`8c12fd58`, Gruntpocalypse, Vallaheim) ne porte aucun tic de garde : pas de barre. Les
  Firefight à colline restent sans film mesuré.
