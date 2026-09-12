# Architecture cible du décodeur de film Theater — et comment l'atteindre sans régression

> Document de conception, écrit le 2026-09-12 à la demande de l'utilisateur, après le lot G
> (version de film lue dans l'en-tête) et le relevé structurel du même jour. Ce n'est PAS un plan :
> l'utilisateur en fera un plan détaillé plus tard. Il dit l'état, la cible, les principes, les
> contrôles anti-régression et une trajectoire d'étapes chacune prouvable. Chemins : ceux d'APRÈS
> le lot E (`internal/games/halo_infinite/film/{filmdec,replay}`) ; tant que le lot E n'est pas
> fusionné, lire `internal/analysis/{filmdec,replay}`.

## 1. Pourquoi maintenant — ce que les trois derniers défauts ont en commun

| Défaut (septembre 2026) | Cause de fond |
|---|---|
| Projectiles lus un bit trop tôt sur Live Fire (lot B-bis) | la largeur d'index de région était écrite deux fois : le lecteur du bipède suivait le catalogue, celui des objets du monde un littéral |
| Kill feed vide sur 211 films de 2025 (lot G) | la version du film (4 premiers octets de `chunk_00`) n'était lue par personne ; trois appelants passaient « version inconnue » |
| Gate corpus vert sur un correctif qu'il ne voyait pas (lots B-bis et G) | aucun témoin sur la carte ni sur la version concernées ; l'empreinte du décodeur ne hache que `killsource/` |

Le point commun : la connaissance du format vit en plusieurs copies, dans des globales de paquet
et des littéraux, sans qu'un objet unique dise « voici ce film et voici comment il se lit ».
Chaque lecteur re-décide. Quand deux lecteurs décident différemment, seul celui qui a un témoin
au corpus se fait prendre.

## 2. État constaté (relevé du 2026-09-12, sur pièces)

- Volume : 60 000 lignes de production — filmdec 25 800 (110 fichiers), constructeur de rejeu
  34 500 (138), killsource 4 700 (20), objectiveevents 4 500 (17), filmsource 500 (3). Six fichiers
  de filmdec dépassent 500 lignes (`traverse.go` 1 380, `unit_weaponstate.go` 970,
  `frame_records.go` 793). filmdec exporte 803 fonctions.
- Lecteurs de bits : `filmdec.BitReader` (le canonique) et `killsource.evReader` (un second,
  pour les chaînes d'événements). Six paquets hors filmsource et filmdec lisent des octets bruts
  de chunk : `killsource/{chunks,feed,walk,world}.go`, `weaponv3/pi_resolver.go`,
  `filmcache/filmcache.go`.
- Paramètres de déchiffrage : une douzaine de variables globales de paquet dans filmdec
  (`TraversalPrecision`, `WorldObjectPrecision`, `DeltaQuantum`, `DeltaAxisWidth`,
  `PositionFullPrecision`, `PositionDeltaHasHandleTail`, `PositionCalibratedSkip`,
  `WorldPositionRange`, `absoluteAxisW`, `MobilityActionExtraBits`, `KeyframeBodyVariants`),
  installées PAR FILM depuis le catalogue de carte par `replay.installWorldObjectPrecision` et
  restaurées en `defer`. C'est de l'état mutable partagé : non parallélisable, et une seconde
  écriture du même paramètre passe inaperçue (lot B-bis).
- Le film chargé ne sait pas ce qu'il est : `filmsource.Film` porte chunks décompressés,
  paquets, bornes et métadonnées de chunk, ni version, ni carte, ni profil. Depuis le lot G,
  `filmdec.FilmMajorVersion(f)` la lit à la demande ; elle n'est pas portée par le film.
- Grammaires versionnées à l'intérieur des données : un enregistrement de création de bipède
  porte sa propre version (R(8), `bipedCreationVersion = 13`) ; le kill feed branche sur la version
  du film (jusqu'à 38, 39-40, à partir de 41). Rien ne relie ces versions à un profil global.
- Consommateurs de filmdec au-delà du rejeu et de killsource : `sync/killcollector` (6
  fichiers), `replaybuild`, `service`, `ops`, `haloclient`, `ingest`, et 7 outils `cmd/`
  (cartes, bornes, structure, fonds, zones, armes). La frontière « qui a le droit de décoder » n'est
  pas tracée.
- Oracles et gates existants, à réutiliser : goldens d'assemblage figés sur entrées
  (`assembly_000d5950.golden` + `inputs_000d5950.bin.gz`, 9 tests Golden), `replay-corpus-gate`
  (8 témoins, 53 vers 53 au lot E), `replay-equiv` (équivalence octet pour octet, 106 artefacts au
  lot B de la campagne v2), `replay-diff`, `TestKillSourceDecoderRevSuitLeDecodeur` (empreinte),
  ratchets `archlint` (tag gamefiles, analysis vers games, comparaison de slug), oracle `swap.sh`
  sur les 4 témoins d'identité, catalogue `map_quant_bounds.json`, `PathResolver`, capabilities.

## 3. Principes de la cible

1. Un film est un objet qui sait ce qu'il est. Version, carte et profil de déchiffrage sont
   résolus une fois au chargement, immuables ensuite.
2. La connaissance du format vit en un seul endroit : le profil. Largeurs de champs, porte
   d'index de région, quantums, implantation du gamertag, variantes de corps d'image-clé : une
   table par version et par carte, pas des globales ni des littéraux dans les lecteurs.
3. Les lecteurs sont des fonctions pures du couple (profil, octets). Pas d'état de paquet, pas
   d'install ni de restore ; deux films se décodent en parallèle sans se voir.
4. Une seule porte d'entrée aux octets bruts. Tout ce qui lit des bits passe par le lecteur
   canonique et par le film chargé ; c'est interdit ailleurs, et un ratchet le dit.
5. Séparer lire, savoir, comprendre, assembler, publier. Cinq couches, une responsabilité
   chacune, des types de sortie stables entre elles.
6. Chaque connaissance a un témoin. Une carte à index de région sur 2 bits, une version 39, une
   version 40, un film sans temps forts : chacun au corpus gate, sinon le gate ment.
7. Tout changement de décodeur fait sonner un gate : empreinte sur tout le décodeur, révision de
   sortie par couche, montée de schéma quand le contenu cuit change.
8. Title-agnostic par construction : le décodeur est propre à Halo Infinite
   (`games/halo_infinite/film`), ce qui en sort est canonique ou passe par les adapters ;
   `analysis/` n'importe jamais un titre (ratchet posé au lot E).

## 4. Couches cibles et responsabilités

```
games/halo_infinite/film/
  source/    LIRE       charger un film (cache ou API), décompresser, découper en chunks et
                        paquets, lire l'en-tête (version), exposer le lecteur de bits canonique.
                        Aucune connaissance de ce que CONTIENT un record.
                        Sortie : *Film (immuable) = octets + métadonnées + Profile.
  profile/   SAVOIR     la table des profils : par version majeure (31 à 41 et suivantes) et par
                        carte (catalogue de bornes) : largeurs d'axes, IndexW et région, quantums,
                        implantation du gamertag, variantes de corps, grammaires connues. Données
                        plus tests de cohérence (toute valeur a une preuve datée : Ghidra, mesure,
                        film témoin). Aucune logique de lecture.
  grammar/   COMPRENDRE les décodeurs de records et de composants (aujourd'hui filmdec) :
                        fonctions pures (profile, bits) vers valeurs typées (positions, états,
                        créations, événements). Un fichier par famille de composant, sous 500 L.
                        Pas d'écriture d'artefact, pas de DuckDB, pas de journal métier.
  facts/     ASSEMBLER  de la chronologie brute aux FAITS du match : vies et identité (registre),
                        tirs, morts et sources de dégât (killsource), objectifs, équipement,
                        véhicules, projectiles, grenades. Chaque fait porte sa couverture
                        (compteurs de contrôle : accord, contradiction, silence) et sa révision.
  replay/    PUBLIER    le document de rejeu 2D (schéma versionné) et ses dérivés ; ne décode
                        rien lui-même ; consomme facts/. Recuisson pilotée par SchemaVersion.
```

Ce qui reste hors du décodeur, inchangé : `sync/killcollector` (orchestration des passes et
persistance via `persist/`), `replaybuild` (parent et enfant, verrous, artefacts), `service/` et
`api/` (exposition), `analysis/` (algorithmes inter-titres sur des types canoniques).

Règles de dépendance, à poser en ratchets `archlint` :

- `source` n'importe rien du décodeur ; `profile` n'importe que `source` (pour l'en-tête) et le
  catalogue de carte ; `grammar` importe `source` et `profile` ; `facts` importe `grammar` ;
  `replay` importe `facts`. Jamais l'inverse.
- Personne hors `source` ne lit `chunk[i][j]` ni ne crée un lecteur de bits.
- `analysis/` n'importe pas `games/{slug}` (déjà posé, allowlist datée à vider).
- Les outils `cmd/` de fabrication (cartes, bornes, structure, fonds) consomment `source` et
  `grammar` via une façade unique, pas les fonctions internes.

## 5. Le profil de déchiffrage, l'objet central

```go
// profile.Profile : IMMUABLE, résolu une fois par film.
type Profile struct {
    MajorVersion int             // u32 LE à l'offset 0 de chunk_00 (lot G) ; l'API le confirme
    Map          MapQuantEntry   // bornes, largeurs d'axes, IndexW, région jouée (catalogue)
    Highlight    HighlightLayout // gamertag en tête ou décalé (jusqu'à 38, 39-40, dès 41)
    Keyframe     KeyframeLayout  // largeur d'en-tête PAR TYPE d'entité (mesure ti=9 : 47 bits)
    Movement     MovementLayout  // quantums de delta, largeur d'axe absolue, drapeaux de queue
    // une entrée par famille de globale actuelle, jamais une globale de plus
}

func Resolve(f *source.Film, catalog *MapQuantCatalog) (Profile, error)
```

Ce qu'il remplace : les douze globales de filmdec et leurs install et restore ; les littéraux de
porte (le « 3 bits » du lot B-bis) ; les trois appels `ParseHighlightEvents(data, 0)` du lot G.

Ce qu'il rend possible : décoder deux films en parallèle ; tester une grammaire sur un profil
synthétique sans film ; dire dans l'artefact sous quel profil il a été cuit (le champ
`coverage.filmMajorVersion` du lot G est le premier pas).

Ce qu'il exige : que chaque valeur ait une preuve (test de cohérence du profil : origine datée,
film témoin), pas une valeur « historique » comme l'en-tête de 64 bits l'a été.

## 6. Réutiliser, ne pas réinventer

| Besoin | Existant à réutiliser tel quel | Ce qui manque |
|---|---|---|
| Lecteur de bits | `filmdec.BitReader` | absorber `killsource.evReader` |
| Chargement du film | `filmsource.Film`, `LoadDir`, `LocalFilmCache` | porter version et profil |
| Version | `filmdec.FilmMajorVersion(f)` (lot G) | la résoudre au chargement, pas à la demande |
| Bornes de carte | `MapQuantCatalog` et `map_quant_bounds.json` | rien |
| Installation de précision | `replay.installWorldObjectPrecision` | devient `profile.Resolve`, sans état |
| Empreinte du décodeur | `TestKillSourceDecoderRevSuitLeDecodeur` | étendre au décodeur entier, une révision par couche |
| Non-régression sur films réels | `replay-corpus-gate` | témoins v39, v40, v38 et moins, film sans temps forts, région 2 bits (fait) |
| Équivalence structurelle | `replay-equiv` (octet pour octet) | le rejouer à chaque étape |
| Goldens figés sur entrées | `assembly_*.golden` et `inputs_*.bin.gz` | un golden par version de film |
| Identité | `swap.sh` sur 4 témoins | inchangé |
| Frontières | ratchets `archlint` | trois ratchets neufs (section 4) |
| Chemins | `PathResolver` | rien |
| Multi-titre | capabilities `film.*` | rien : le décodeur reste sous `games/halo_infinite` |

## 7. Contrôles anti-régression, la règle du chantier

Un pas de la trajectoire n'est clos que si, sur la même entrée, la sortie est identique à celle
d'avant le pas. Pas « les tests passent », mais « les octets sont les mêmes ».

1. Équivalence octet pour octet (`replay-equiv`) sur un corpus figé d'une vingtaine d'artefacts
   couvrant toutes les versions présentes (31, 33, 37, 38, 39, 40, 41) et les familles de mode :
   zéro différence attendue à chaque pas structurel. Toute différence arrête le pas ; elle ne se
   « justifie » pas.
2. Corpus gate (`replay-corpus-gate --reference=base`) : témoins élargis avant le chantier (le
   lot H les choisit) ; zéro perte, zéro gain à chaque pas.
3. Goldens figés sur entrées : un par version de film ; un pas ne les régénère jamais, sauf
   montée de schéma explicite.
4. Empreinte du décodeur entier : le test refuse un changement de source sans montée de la
   révision de la couche concernée (`grammar`, `facts`) ; une montée de révision de `facts`
   entraîne le backlog killsource ; un contenu cuit qui change entraîne la montée de schéma.
5. Ratchets de frontière posés avant de déplacer (comme au lot E) : lecture brute hors `source`,
   dépendances entre couches, taille de fichier (plafond gelé, jamais accru).
6. Mutation obligatoire pour chaque valeur de profil migrée : la fausser doit rougir un test
   nommé (leçon des revues de septembre : « zéro désynchronisation avec zéro composant » ne
   prouve rien).
7. Oracles externes conservés : table des scores, API des statistiques, `swap.sh`.

## 8. Trajectoire : des pas petits, chacun prouvable

Ordre fondé sur le risque : d'abord ce qui ne change aucun comportement, ensuite l'état, enfin
les frontières.

| Pas | Contenu | Preuve de clôture |
|---|---|---|
| 0 | Lot H : mesure calque par version ; choix des témoins ; corpus figé pour `replay-equiv` | tableau et témoins au corpus gate |
| 1 | Version et carte portées par `Film` au chargement ; `Profile` créé avec les valeurs actuelles, encore recopiées dans les globales (double écriture temporaire, test d'égalité) | équivalence à zéro différence ; test « profil égale globales » |
| 2 | Les lecteurs reçoivent `Profile` en paramètre et cessent de lire les globales, famille par famille (positions, objets du monde, images-clés, temps forts, équipement) | équivalence à zéro différence à chaque famille ; mutation par valeur |
| 3 | Suppression des globales et des install et restore ; deux films décodés en parallèle dans un test | équivalence ; test de concurrence |
| 4 | Un seul lecteur de bits ; `killsource` et `weaponv3` passent par la façade ; ratchet « lecture brute » | équivalence ; ratchet vert à allowlist vide |
| 5 | Découpage en couches (`source`, `profile`, `grammar`, `facts`, `replay`) par déplacements purs, ratchets de dépendance posés avant (méthode du lot E) | équivalence ; `git mv` sans ajout ni suppression ; gate corpus |
| 6 | Empreinte étendue et révisions par couche ; golden par version | une mutation de source rougit l'empreinte |
| 7 | Scission des fichiers de plus de 500 lignes (`traverse.go` en premier) sans changement de comportement | équivalence ; baseline lint non accrue |
| 8 | Exploiter le profil : là où le lot H a montré une divergence par version, brancher (nouveau comportement, donc montée de révision, de schéma et recuisson) | corpus gate : gains attendus, zéro perte ; oracles |

Les pas 1 à 7 ne changent pas un octet des artefacts : c'est ce qui rend le chantier sûr et
interruptible. Le pas 8 est le seul qui produit de la valeur visible, et il n'est possible que
parce que les sept précédents ont rendu la version et la carte explicites.

## 9. Ce que ce chantier n'est pas

- Pas une réécriture : chaque grammaire acquise par rétro-ingénierie (Ghidra, p-code, mesures)
  reste telle quelle ; seuls ses paramètres et son emplacement bougent.
- Pas un changement de format d'artefact ni de tables DuckDB, sauf au pas 8, et seulement par
  montée de schéma.
- Pas avant la release v7.5 : lots G et H d'abord, puis les tâches Notion 9 et 10, puis ceci.
- Pas de « au cas où » : une couche ou un ratchet sans consommateur au pas où il est introduit
  n'est pas introduit.

## 10. Risques et parades

| Risque | Parade |
|---|---|
| Une globale a un lecteur caché (test, outil `cmd/`) | pas 1 : double écriture, puis grep ratcheté avant suppression |
| Le profil devient un fourre-tout | une entrée par famille, chaque champ avec sa preuve datée, revue adversariale par famille |
| `replay-equiv` trop lent pour tourner à chaque pas | corpus figé de taille bornée (une vingtaine d'artefacts, toutes versions) ; gate complet en fin de pas majeur |
| Chantier long, pression de livrer partiellement | chaque pas est clos et fusionnable seul ; l'ordre est fait pour s'arrêter proprement après n'importe lequel |
| Conflits avec les branches vivantes du décodeur | fenêtre dédiée, aucune autre branche filmdec ou replay en vol (méthode du lot E) |
