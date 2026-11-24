# Revue r1 : non-déterminisme et état partagé (v1.5.0...feat/v1.6.0)

Lentille : tout ce qui lance ou sert des fils. Lecture seule. Chemins relatifs à la racine du dépôt du décodeur.

Mention « vérifié » : relu dans le code ou rejoué par un test. Mention « déduit » : raisonné sur le code, sans exécution.

## Constats recevables

### C1. Les tests d'identité et de course des voies sont sautés dans la CI du dépôt

- **Où** : `.github/workflows/ci.yml:79` (tâche « Film race (grammar, replay) ») ; `replay/film_scan_fond_test.go:59` (`optionsDuFond` appelle `goldenEntryPourTest`) ; `internal/depottest/depottest.go:57` (le saut).
- **Condition de déclenchement** : lancer les tests sans la variable `DECODEUR_FILM_RACINE_DONNEES` et sans `go.work` au-dessus du dépôt. C'est le cas de la CI : aucun fichier sous `.github/` ne pose cette variable, et le dépôt est pris seul.
- **Conséquence observable** : 10 des 13 tests `^TestVoie` de `replay` se déclarent absents (`SKIP`), dont tous ceux qui tiennent les invariants du lot : `TestVoieFondIdentiqueATousLesFils`, `TestVoieOrdreDesEtapesIdentique`, `TestVoieMondeIdentique`, `TestVoieMondeCompteLesReplisUneSeuleFois`, `TestVoieSiegesIdentique`, `TestVoieAUnFilAucuneGoroutine`, `TestVoieRepliAUnFilSiMarcheEnErreur`, `TestVoieFondLitCeQueLeFilmPorte`, `TestVoieNeLitQueDeVraiesValeurs`, `TestVoieMondeLitLesSoclesAuxLargeursDesPoses`. Seuls passent les trois qui ne décodent aucun film (`TestVoiePaniqueRelayee`, `TestVoiePaniqueDUneVoieRelayee`, `TestVoieArretEntreDeuxLectures`). La tâche de CI reste verte sans avoir comparé une sortie à 1, 2, 4 et 8 fils ni cherché une course dans la tâche de fond et les voies. Le commentaire de la tâche (« prouvés sous -race : mini-bobines du dépôt, aucune base ») est faux pour `replay` : les bobines sont dans le dépôt, mais l'entrée de catalogue de la carte ne l'est pas.
- **Preuve** : vérifié. Rejoué sans la variable : 10 `SKIP`, 3 `PASS`. Rejoué avec la variable pointée sur un dépôt hôte, sous `-race` : 17 tests `PASS`, aucune course signalée.
- **Gravité proposée** : P1. Le code tient l'invariant aujourd'hui ; c'est sa garde qui ne tourne pas là où elle est annoncée.

### C2. Le nombre de fils demandé n'est pas le nombre de fils employés

- **Où** : `replay/film_scan_voies.go:150-159` (`demarrerLesVoies` lance toujours deux voies dès que `Fils() > 1`) ; `internal/grammar/keyframe_etats_predecodage.go:85` et `internal/grammar/keyframe_predecodage.go:89` (`executerEnParallele(Fils(), …)` appelé depuis le fil principal pendant que les voies tournent) ; `replay/film_scan_fond.go:103`.
- **Condition de déclenchement** : `Options.Vitesse.Fils = 2` (ou 3) sur un film dont la marche des trames réussit.
- **Conséquence observable** : après la marche, trois goroutines calculent en même temps (chaîne principale, voie du monde, voie des sièges). Pendant le pré-calcul des mots des images-clés, lancé par la chaîne principale, s'y ajoutent `Fils` ouvriers : quatre fils actifs pour une demande de 2, cinq pour une demande de 3. Pendant la marche, le pré-passage des images-clés (`Fils` ouvriers) tourne en même temps que la tâche de fond. La borne écrite dans `replay/options_vitesse.go:24` (« nombre de fils que le décodeur a le droit de lancer ») n'est donc pas tenue ; un hôte qui dimensionne ses cuissons simultanées sur cette valeur emploie plus de cœurs que prévu. La sortie n'est pas touchée.
- **Preuve** : vérifié à la lecture (aucune des trois sources ne retranche les fils déjà lancés par les autres). Non mesuré à l'exécution.
- **Gravité proposée** : P2.

### C3. Un contexte partagé qui porte déjà des créations du monde fait compter un repli deux fois à plusieurs fils

- **Où** : `internal/grammar/film_context_voie.go:136-149` (`Deriver` ne transmet ni `recup.pistes` ni `recup.creations`) et `:179` (`Fusionner` additionne tous les replis de la voie) ; `replay/film_scan_voies.go:150-159` (aucune garde sur des mémos du monde déjà remplis chez le parent) ; `internal/grammar/creations_du_monde.go:134` (le repli se compte au premier rendu d'une clé, par contexte).
- **Condition de déclenchement** : un hôte appelle `replay.LireLesVehicules(fc, wr)` (fonction publique, que le collecteur de l'hôte appelle déjà sur un contexte) puis passe ce même contexte à la cuisson par `Options.Contexte`, avec `Vitesse.Fils >= 2`, sous le même profil de balayage et les mêmes bornes, sur un film dont la passe rend au moins une création de véhicule derrière la marche.
- **Conséquence observable** : à un fil, la cuisson retrouve la clé des véhicules déjà rendue et ne recompte rien. À plusieurs fils, la voie du monde part sans ce mémo, relève de nouveau, recompte, et la fusion ajoute ce compte à celui du parent : `repli_creations_du_monde_apres_la_marche` vaut plus à N fils qu'à un fil. Le rapport des replis des faits persistés et `coverage.fallbacks` du document ne sont plus identiques à l'octet d'un nombre de fils à l'autre. Les valeurs lues, elles, restent les mêmes.
- **Preuve** : déduit. L'en-tête de `film_context_voie.go` écrit lui-même la précondition (« le parent ne doit pas les avoir demandés avant de dériver ») et le test `TestVoieOrdreDesEtapesIdentique` dit avoir vu rouge la mutation voisine (le fil principal lit les poses avant de dériver). Rien ne vérifie cette précondition à l'exécution. Le chemin de cuisson de l'hôte connu n'appelle pas `LireLesVehicules` avant la cuisson : le défaut est latent. Pour le vérifier : un test qui appelle `LireLesVehicules` sur un contexte, puis compare `ComptesDesReplis()` après un balayage à 1 et à 2 fils.
- **Gravité proposée** : P2.

## Écart au contrat annoncé, sans conséquence observable (non recevable comme défaut)

- Le contrat du lot dit « une seule source de goroutines (`executerEnParallele`) ». Le décodeur en porte trois : `internal/grammar/fils.go:46`, `replay/film_scan_fond.go:103`, `replay/film_scan_voies.go:107`. Le garde `TestUneSeuleSourceDeGoroutines` les autorise nommément et les deux fichiers de `replay` disent pourquoi. Chacune a son propre relais de panique. À trancher par le pilote : soit le contrat est à réécrire, soit c'est un écart.
- Sortie anticipée : si les positions échouent et qu'une voie ou la tâche de fond a paniqué sur une lecture que la chaîne n'aurait pas faite à un fil, la jointure rejoue la panique là où un fil rend l'erreur des positions (`replay/film_scan_voies.go:197-201`, `replay/film_scan_fond.go:176-181`). Aucune entrée connue ne fait paniquer une lecture : pas de déclencheur concret, donc pas un constat.

## Conditions vérifiées qui tiennent : 21

1. `executerEnParallele` : à `n <= 1` ou une seule tâche, aucun `go` ; zéro tâche rend la main ; plus de fils que de tâches est borné ; chaque indice tiré une fois (compteur atomique) ; la panique d'indice le plus petit est rejouée après la jointure.
2. Pré-passage des images-clés : chaque tâche est pure (payload et preuve en lecture) ; les rangements s'exécutent sur le fil appelant, dans l'ordre des indices ; le résultat ne dépend pas du nombre de fils.
3. La preuve des ancres est figée avant le pré-passage et n'est que lue par les ouvriers ; le registre du film n'a aucun champ paresseux.
4. Le mémo des marches d'ancres, seul mémo écrit par plusieurs fils, est sous verrou et rend des copies.
5. Pré-calcul des mots : les records du pré-calcul et ceux de la fenêtre viennent de la même marche mémorisée, triés par bit ; la fenêtre ne lit que les records non admis, par leur début ; garde sur le nombre de records.
6. Aucune variable de paquet de `grammar`, `profile` ou `source` n'est écrite en production (les cartes de paquet sont lues seulement ; `inferResyncTargets` n'est jamais affectée hors test).
7. `source.Film` n'a aucun accesseur qui écrit : la tâche de fond ne partage que lui et une copie du roster.
8. Tâche de fond : chaque valeur est écrite avant la fermeture de son canal et lue après ; le pont verse les deux replis au même endroit qu'à un fil.
9. `Deriver` : le profil de balayage, la grammaire du film et les vies sont copiés par valeur ; le mémo des tables de datums est cloné ; observation, replis, diagnostics, pistes et créations du monde sont neufs.
10. Les objets partagés par pointeur entre le parent et les voies (lectures bipèdes, objets du monde, équipement porté, ancrage, générations vivantes, en-têtes exacts, bande de slots, registre) ne sont que lus par leurs lecteurs ; `GenerationsVivantes.A` rend une copie.
11. Un mémo que le parent ou une voie refait (ancrage entier, table absente) est rangé dans son propre champ, jamais dans l'objet partagé.
12. Tous les poseurs de largeurs MPP et de profil appelés pendant le balayage restituent la valeur d'avant : une voie dérivée juste après la marche lit sous le profil que la chaîne aurait à un fil.
13. La chaîne principale ne demande jamais les pistes ni les créations du monde sur le contexte du film pendant une cuisson : seule la voie du monde les relève, dans l'ordre d'un fil (poses, socles, véhicules, projectiles).
14. Les replis « une fois par film » (découpage d'i0 par défaut, registre inconnu, contrôle de corruption) ne s'ajoutent pas deux fois à la fusion.
15. La fusion se fait sur le fil principal, après la fin de chaque voie, dans un ordre fixe ; aucun parcours de `map` n'y atteint la sortie.
16. Les voies ne publient, ne journalisent et n'observent rien : l'ordre des étapes observées et des déclenchements du compteur de replis est celui de la chaîne.
17. Les largeurs MPP que la voie donne aux socles et aux véhicules sont celles que la chaîne publie (`journaliserPoses` rend les statistiques telles quelles, erreur comprise).
18. Aucune goroutine ne survit au balayage : les deux jointures sont différées et attendent toujours ; une voie ne bloque sur aucun canal.
19. Une panique de voie ou de tâche de fond ferme toutes les valeurs non rendues et n'est rejouée qu'une fois, à l'attente ou à la jointure.
20. À un fil : ni tâche de fond, ni voie, ni pré-passage, ni `Chauffer`.
21. Sous `-race`, avec les données de test présentes : les 17 tests des voies et de la tâche de fond de `replay` passent sans course ; les tests `TestFils`, `TestChauffer`, `TestDeriver`, `TestFusionner`, `TestExecuterEnParallele` de `grammar` passent sans course.

## Limites de cette revue

- Les tests du dépôt comparent 1, 2, 4 et 8 fils sur deux bobines que le balayage des positions refuse. Aucun test du dépôt ne compare le document et les faits complets d'un film entier d'un nombre de fils à l'autre : la chaîne après les positions (tirs, images-clés, naissances, portage, capacités) pendant que les voies tournent n'est couverte ici que par la lecture du code.
- La détection de courses du paquet `replay` entier n'a pas été lancée (dette assumée).
