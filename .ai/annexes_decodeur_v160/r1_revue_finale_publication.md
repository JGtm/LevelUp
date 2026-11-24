# Relecture r1 : contrat publié et tests (v1.5.0...feat/v1.6.0)

Lentille : contrat publié et tests. Périmètre : `replay/`, `decfilm/`, `cmd/`, documentation, tests de tout le dépôt.
Lecture seule : aucun fichier du dépôt modifié. Les mesures ont été prises par des tests posés hors du dépôt
(`go test -overlay`), avec `GOWORK=off`.

Marquage : « vérifié » = relu dans le fichier ou commande rejouée ; « déduit » = raisonné sur le code, sans preuve mesurée.

## Constats recevables

### C1 (P1). La vérification de style de l'intégration continue échoue sur la branche

- Où : `.github/workflows/ci.yml:92-95` (golangci-lint v2.12.2, `--timeout 5m`, aucune base de référence).
- Condition : lancer la commande de l'intégration continue sur la tête de `feat/v1.6.0`.
- Conséquence : 18 défauts, contre 0 sur `v1.5.0` avec le même outil (vérifié, les deux arbres rejoués avec la version 2.12.2).
  Le travail `golangci-lint` sort en échec, donc la branche ne passe pas son gate.
- Détail des 18 :
  - `internal/grammar/film_context_voie.go:75` et `:76` : erreur de `c.I0Layout()` et de `c.Registry()` non lue (errcheck) ;
  - `replay/equipment_placements_decode.go:40` : `decodeFilmSpawnEvents` sans appelant (voir C6) ;
  - `replay/eliminations_bots_test.go:14` (`actionDeBot`) et `replay/pistes_partagees_test.go:110` (`pisteTrieeDeTest`) : aides de test sans appelant ;
  - `replay/document_score_actions.go:307` et `revision/perimetre.go:133` : complexité 16, plafond 15 ;
  - `internal/facts/objectives/named_series.go:113` : paramètre `teams` toujours faux ;
  - `replay/tracks_publication.go:436`, `replay/document_repliques.go:111`, et 8 littéraux répétés
    (`chattername/table.go:42`, `internal/grammar/dispatch_object.go:171`, `internal/grammar/etats_mouvement_hooks.go:139`,
    `internal/grammar/frame_closure_detail.go:84`, `replay/build_score.go:146`, `replay/layers.go:178`, `replay/observe.go:56`,
    `replay/projectiles_colles_calque.go:60`).

### C2 (P1). Aucun test du dépôt ne voit disparaître six calques neufs

- Où : `replay/film_inputs.go:334-338` (le passage des répliques, des gestes, de l'arme tirée, des projectiles collés et des vitalités
  vers l'assemblage) ; `replay/golden_assembly_test.go:353` (`renderAssembly`) ; `replay/filmfacts_producteur_test.go:92`.
- Condition : supprimer ces cinq lignes. Tout document publié perd alors `chatter[]`, `announcements[]`, `gestures[]`,
  `drawnWeapons[]`, `vitals[]`, `attachments[]`, `projectiles[].attach` et leurs couvertures.
- Conséquence : les tests restent verts (vérifié par mutation posée hors dépôt, racine de données de l'hôte fournie ; sélection
  jouée : tests des calques neufs, goldens, faits, équivalence, mini-films, contrat ; seuls échouent les 4 `TestContractFixtures*`
  de la dette déclarée, qui échouent aussi sans la mutation). Limite : sélection large par `-run`, pas la suite entière.
- Pourquoi : les 8 jeux d'entrées figés portent 0 réplique, 0 geste, 0 arme tirée, 0 vitalité, 0 collage, 0 apparence et 0 commande
  de contrôle (vérifié par mesure), et `renderAssembly` n'imprime aucun calque neuf, y compris ceux que ces films portent
  (`ammoReads` : 146 à 4 442 lignes ; `fullRate` sur toutes les pistes ; `padCandidates` jusqu'à 37 ; `deto` ; `shots[].o` ;
  `coverage.controlPlaces`). Les diffs des 8 goldens d'assemblage ne montrent donc que les socles et le numéro de schéma.
  `TestBuildFromFactsEgaleLAssemblageDirect` compare par `renderAssembly` : l'égalité « depuis les faits = depuis le film » n'est
  pas vérifiée dans le dépôt pour ces calques. Ce qui tient : le codec feuille à feuille (`TestCodecCouvreFilmInputs`) et les tests
  unitaires de chaque constructeur.

### C3 (P2). `coverage.controlPlaces.bindings` ne compte pas ce que la documentation dit

- Où : `replay/places_de_controle.go:156` (`slices.Compact`) puis `:115` ; texte à `replay/places_de_controle.go:55` et `FORMAT.md:467`
  (« shots whose record binds a place »).
- Condition : deux tirs du même joueur, à la même place, dans la même frame.
- Conséquence : le compteur publié est le nombre de liaisons distinctes (place, frame, joueur), pas un nombre de tirs.
  Mesuré sur les jeux figés : 605 tirs dont le record nomme place et unité, `bindings` = 599 (`000d5950`) ; 3 807 contre 3 803
  (`fb1a1a72`). Un lecteur qui rapproche `bindings` des tirs lit un manque qui n'existe pas. Le test
  (`places_de_controle_test.go:50`) pose ses tirs à des frames distinctes et ne distingue pas les deux sens.

### C4 (P2). `layers.attachments` peut déclarer le calque produit alors que sa passe n'a rien posé

- Où : garde `replay/layers.go:310-312` (canal lu et composant présent) ; passe `replay/projectiles_colles.go:151`
  (sort aussi quand `a.projCov == nil`, c'est-à-dire quand aucune piste de projectile n'est fournie, `replay/build_pistes.go:247`).
- Condition : la lecture des trajectoires échoue (`replay/film_scan_pont.go:277-279`, « projectiles illisibles ») ou ne rend aucune
  piste, alors que le canal des collages a lu des collages.
- Conséquence (déduit du code) : `layers.attachments` présent, `attachments[]` et `coverage.projectiles.attach` absents ; les
  collages lus ne sont ni publiés ni comptés. Le test `projectiles_colles_test.go:194` (« sans piste ») fige ce silence.
  Ce qui le vérifierait : un assemblage avec `Options.Projectiles` vide et des lectures de collage, puis la lecture de `doc.Layers`.

### C5 (P2). Un projectile lu collé deux fois est publié deux fois

- Où : `replay/projectiles_colles.go:122-131` et `:180` (une seule lecture par vol), `:186` (toutes les autres vont dans
  `attachments[]`) ; `internal/grammar/canal_des_projectiles_colles.go:135-139` (une seconde lecture de la même vie est gardée et
  comptée dans `Repeats`).
- Condition : deux records du même projectile (même slot, même génération) ouvrent la porte « attache ».
- Conséquence (déduit du code) : le même projectile sort sur son vol et dans `attachments[]`, ou deux fois dans `attachments[]`,
  alors que `FORMAT.md:435` dit « published once ». Le compteur `Repeats` du canal n'est pas publié
  (`ProjectileAttachCoverage` ne le porte pas) : le document ne permet pas de voir le cas. Fréquence non mesurée : les jeux figés
  ne portent aucun collage. Ce qui la donnerait : `ProjectileAttachStats.Repeats` sur le corpus.

### C6 (P2). Code mort neuf

- Où : `replay/equipment_placements_decode.go:40`, `decodeFilmSpawnEvents`.
- Condition : la refonte a déplacé la lecture vers `decodeFilmSpawnEventsLecture` (`replay/lecture_monde.go:105`), appelée par
  `film_scan_monde.go:88` et `film_scan_voie_monde.go:75`.
- Conséquence : la fonction n'a plus aucun appelant, ni en production ni en test (vérifié par recherche et par l'outil de style).

### C7 (P2). Compteurs publiés sans description dans `FORMAT.md`

- Où : `FORMAT.md:435-442` (projectiles collés) et `FORMAT.md:452-457` (santé et bouclier).
- Condition : lire `FORMAT.md` pour interpréter la couverture.
- Conséquence (vérifié par recherche) : `coverage.projectiles.attach.records`, `.created` et `.wideLocations` n'y figurent pas ;
  `coverage.vitals.afterDeath`, `.ramp`, `.births` et `coverage.ammoReads.empty` n'y sont pas nommés par leur clé (la santé est
  décrite en prose, sans les clés JSON). Le registre de la version 127 les liste, sans définition pour `wideLocations` et `empty`.

### C8 (P2, hors du diff, déjà présent en v1.5.0). Pseudos et chemins de poste à la tête du dépôt

- Pseudo d'un joueur tiers : `internal/facts/killsource/rev_chronique.go:232` (un pseudo, plus celui du mainteneur).
- Pseudo du mainteneur (le même que le nom du compte qui héberge le module) : 67 lignes hors chemin d'import, dont
  `internal/facts/objectives/awards.go:51` et `:135`, `internal/grammar/ability_charges.go:13`,
  `internal/grammar/highlight_events.go:89`, le fichier `internal/grammar/highlight_events_jgtm_test.go` (27 lignes),
  `replay/ability_impulses_film_test.go`, `replay/ability_charges_film_test.go`, `replay/document_chronicle.go` (1 ligne).
- Chemins de poste : `research/divergence/divergence_research_test.go:19` et `:20`,
  `research/divergence/fermeture_carte_research_test.go:10`, `research/duree/duree_research_test.go:14`.
- Condition : lecture du dépôt public. Conséquence : la règle « aucun pseudo, aucun chemin de poste » n'est pas tenue.
  Rien de cela n'est ajouté par ce lot (vérifié : mêmes comptes à `v1.5.0`).

## Note, sans constat

- `replay/contract_fixtures_budget_test.go:46` : le relèvement du plafond à 5 Mio porte « à confirmer par le mainteneur », puis
  le plafond passe à 11 Mio. La raison est écrite ; la confirmation reste à donner.
- Le rang des socles de power-up dans `weaponPads[]` change sur un film inchangé (golden `assembly_a521164d` : rangs 31 à 33
  devenus 36 à 38), donc aussi `padPickups[].pad`. Le registre dit que le tableau gagne les socles d'équipement et que les socles
  d'arme ne bougent pas ; il ne dit pas ce déplacement en toutes lettres.

## Conditions vérifiées qui tiennent : 16

1. Interface publique : seules `SchemaVersion` (126 vers 127) et `SchemaDesFaits` (30 vers 31) sont incompatibles (`apidiff` sur
   tous les paquets publics) ; aucun symbole retiré, aucune signature changée, aucun type rendu non comparable.
2. Chaque calque et champ neuf a sa ligne au registre 127 et son paragraphe dans `FORMAT.md` (environ 75 clés cherchées ;
   exceptions en C7).
3. `omitempty` : les slots publiés (`damage[].slot`, `by`, `attachments[].by`) valent au moins 512, donc jamais effacés ;
   les valeurs où zéro est mesuré sont des pointeurs (`d`, `d2`, `mag`, `res`, `gauge`, `line`, `place`, `ms0`, `ms1`).
4. `len(damage[])` = `damages` + `damagesZero`, sur les 8 jeux figés.
5. `ammoReads` : `read` = somme des six cas, et le nombre de lignes = `fromKeyframe` + `fromDelta`, sur les 8 jeux.
6. `len(weaponPads)` = `pads` + `powerupPads` ; `len(padCandidates)` = `candidates` + `powerupCandidates` ; aucun candidat avec cycle.
7. Pleine cadence : chaque point de grille est le premier échantillon de sa frame (x, y, z au centième), sur toutes les pistes
   des 8 jeux ; `fullRateSamples` et `fullRateAimSamples` égalent les colonnes ; `projectiles[].z` a la longueur de `p`.
8. Sur ces 8 films, un slot de piste ne porte qu'une vie : la clé (slot, emplacement) de `ammoReads` ne mêle pas deux vies.
9. Le codec des faits porte chaque feuille des sections neuves (`TestCodecCouvreFilmInputs`, joué avec la racine de l'hôte).
10. Les goldens passent à la tête (`TestGoldenAssembly`, `TestGoldenBuildsAssembly`, forme du document, point fixe du fichier de faits).
11. Tris neufs à clé complète ou stables sur un ordre d'entrée fixe (éliminations, actions de score, gestes, munitions,
    liaisons de place) ; les parcours de `map` neufs ne font que des sommes.
12. Identifiants : tout nombre de 16 chiffres de la plage des xuid, dans les fichiers texte de la tête, est au registre
    des identifiants fictifs (`testdata/IDENTITES.tsv`) ; de même les formes hexadécimales.
13. Fichiers texte neufs (`bench/films.tsv`, `bench/releves/v1.5.0.tsv`, `chattername/data/names.tsv`) : ni identifiant, ni
    pseudo, ni chemin ; les apports de tiers ont leur crédit dans `NOTICE.md`.
14. Sauts de test ajoutés : tous des instruments de recherche ou un film absent, avec leur raison.
15. Chaque compteur de couverture neuf a un écrivain ; aucun n'est figé à zéro hors `seatsBeyondBand`, déprécié et dit.
16. Seuils : aucun fichier de production touché ne passe 500 lignes, hors `replay/document_chronicle.go` (le registre, fait de
    commentaires, déjà au-delà en v1.5.0) ; aucune fonction ne passe 80 lignes ; `go vet` (avec et sans l'étiquette de
    recherche) passe.
