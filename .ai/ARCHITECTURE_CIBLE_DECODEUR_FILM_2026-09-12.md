# Architecture cible du décodeur de film Theater — et comment l'atteindre sans régression

> Document de conception, écrit le 2026-09-12 à la demande de l'utilisateur, après le lot G
> (version de film lue dans l'en-tête) et le relevé structurel du même jour. Ce n'est PAS un plan :
> l'utilisateur en fera un plan détaillé plus tard. Il dit l'état, la cible, les principes, les
> contrôles anti-régression et une trajectoire d'étapes chacune prouvable. Chemins : ceux d'APRÈS
> le lot E (`internal/games/halo_infinite/film/{filmdec,replay}`) ; tant que le lot E n'est pas
> fusionné, lire `internal/analysis/{filmdec,replay}`.
>
> Amendé le 2026-09-12 (même jour) après relecture sur pièces : écarts entre le texte et le code
> (sections 2 et 5), principes ajoutés (section 3, à partir du 9), ce que le film déclare
> réellement de lui-même (section 5 bis), contrôles supplémentaires (section 7, à partir du 8),
> second chantier de la publication (section 11), tests des deux côtés (section 12), forme
> (section 13). Les ajouts sont marqués « (ajout) » dans les tables.

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
- (ajout) L'état de paquet ne se limite pas à cette douzaine : le ratchet
  `archlint/filmdec_package_vars_test.go` gèle **118 noms** de niveau paquet dans filmdec
  (mesure du 2026-09-05), dont une trentaine de crochets d'observation posés par les tests
  (`SetRecordMaskHook`, `SetProbeHook`, `unitRefHook`...), des compteurs, et des tables de
  grammaire déguisées en `var`. La table d'observation des largeurs de composants
  (`frame_chain_infer.go`) s'écrit sans verrou pendant un balayage. C'est pour tout cela que
  `filmdec.LockProcessDecode()` sérialise tout décodage du process, et que le ratchet
  `archlint/decode_lock_held_test.go` impose sa tenue à tout chemin de production. Tant que les
  crochets et cette table existent, « deux films en parallèle » est faux par construction, même
  après la migration des douze paramètres.
- Le film chargé ne sait pas ce qu'il est : `filmsource.Film` porte chunks décompressés,
  paquets, bornes et métadonnées de chunk, ni version, ni carte, ni profil. Depuis le lot G,
  `filmdec.FilmMajorVersion(f)` la lit à la demande ; elle n'est pas portée par le film.
- (ajout) Mais un objet « le film se connaît » existe déjà en partie : `filmdec.FilmContext`
  (`film_context.go`, lot 2 du chantier cuisson) porte le film, ses chunks, la bande de slots,
  le découpage i0 (imposé par le catalogue ou auto-détecté) et le registre, résolus
  paresseusement et mémorisés par film. Le `Profile` de la section 5 et `FilmContext` se
  recouvrent : sans décision explicite on aura deux objets pour la même question.
- (ajout) Une partie de la connaissance est INFÉRÉE des octets, pas tabulée : `DetectI0Layout`
  lit les frontières de champs sur le profil statistique de bascule des bits (dent de scie),
  `frame_chain_infer.go` résout les archétypes des transitoires par exploration. Le chantier
  cuisson a sorti la première du chemin de production quand le catalogue impose la valeur
  (`FilmContext.impose`). Le texte initial de ce document (« résolu une fois au chargement,
  immuable ») ne disait pas ce que deviennent ces inférences.
- (ajout) Publication : `replay.SchemaVersion = 53`, un numéro unique pour tout le document,
  avec la règle « un champ optionnel n'incrémente pas le schéma » ; tout artefact d'un numéro
  inférieur est « à recuire » en entier (`replaybuild.Digest.UpToDate`), et la recuisson totale
  a déjà été bloquée une fois par la bombe mémoire de `NamedEventsFrom`. Le document servi est un
  type jumeau sans numéro (`domain/replaydoc`), la version voyage dans un en-tête HTTP. Côté
  web : types générés d'OpenAPI (effacés à l'exécution), normalisation à la frontière
  (`normalizeReplayDocument`), une fixture partagée écrite à la main qui déclare
  `schemaVersion: 1`, un badge admin « à jour / à recuire ». Zod est une dépendance du web mais
  ne valide pas le document de rejeu.
- (ajout) Ce que la CI voit : `go test`, `go vet -tags=gamefiles` (compilation seule), vitest,
  typecheck. Ni le corpus gate, ni `replay-equiv`, ni `swap.sh` ne tournent en CI : toutes les
  preuves « octet pour octet » de ce document sont locales. Aucun `Benchmark` n'existe dans
  filmdec, replay ni killsource : le gain du chantier cuisson (3 min 30 vers 15 s par film) n'a
  pas de gardien.
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
9. (ajout) La grammaire vient du JEU, le film n'en est que la clé et le témoin. Largeurs,
   quantums et implantations se dérivent des fichiers du jeu (exécutable, `.module`) par un
   outil de fabrication qui écrit un catalogue versionné avec sa provenance, comme
   `cmd/mapquant-build` le fait déjà pour les bornes. Le film fournit la clé (build en clair,
   version majeure) et des déclarations qui servent de CONTRÔLE (ordre des composants du
   registre, table par type). Ce que le film permet d'INFÉRER par statistique reste un oracle de
   test, jamais le chemin de production. Détail en section 5 bis.
10. (ajout) Une version inconnue échoue fort. Un build ou une version majeure absents du profil
    donnent une erreur typée, un compteur expvar par version (ADR 0009) et un film mis de côté,
    jamais une lecture avec le profil précédent par défaut. Un profil peut hériter d'un autre,
    mais l'héritage est une entrée explicite marquée « présumée », et un test liste les
    présumés : le corpus doit les confirmer avant qu'ils deviennent « prouvés ».
11. (ajout) Frontières imposées par le compilateur avant les ratchets. Les couches internes du
    décodeur vivent sous un sous-dossier `internal/` de `games/halo_infinite/film/`, donc
    inaccessibles hors du décodeur à la compilation (esprit ADR 0030). Les ratchets ne gardent
    que ce que le compilateur ne sait pas exprimer (dépendances entre couches sœurs, lecture
    brute, taille de fichier).
12. (ajout) Sémantique d'erreur par couche. `grammar` est pure : elle ne journalise pas, elle
    rend valeur plus diagnostics typés. `facts` agrège les diagnostics en couverture (accord,
    contradiction, silence). Seul l'orchestrateur (`killcollector`, `replaybuild`) journalise en
    `slog` et publie en expvar. Aucune erreur n'est avalée : elle remonte ou elle se compte.
13. (ajout) Aucune panique ne sort du décodeur. Un décodeur binaire face à un build neuf peut
    sortir des bornes ; le lecteur de bits et les grammaires de records ont des cibles de fuzz
    (`go test -fuzz`) et une frontière de reprise unique dans `facts` qui compte l'incident au
    lieu de faire tomber le sync.
14. (ajout) La preuve vit dans le code. Chaque valeur de profil porte, dans le code, l'origine
    (fonction Ghidra, mesure, film témoin nommé) et la date ; les relevés de `.ai/` en sont la
    chronique, pas la source, parce que `.ai/` tourne.
15. (ajout) Données et code séparés. Ce qui se dérive du jeu est une DONNÉE (catalogue JSON
    ou TOML versionné, jamais écrit à l'exécution, ratchet
    `no_runtime_versioned_catalog_write_test` étendu au profil) ; ce qui se lit est du CODE
    typé. Une valeur ne vit pas dans les deux.

## 4. Couches cibles et responsabilités

```
games/halo_infinite/film/
  source/    LIRE       charger un film (cache ou API), décompresser, découper en chunks et
                        paquets, lire l'en-tête (version), exposer le lecteur de bits canonique.
                        Aucune connaissance de ce que CONTIENT un record.
                        Sortie : *Film (immuable) = octets + métadonnées + Profile.
  profile/   SAVOIR     la table des profils : par BUILD (clé exacte lue en clair dans chunk_00,
                        section 2 ; la version majeure 31 à 41 en repli) et par carte (catalogue
                        de bornes) : largeurs d'axes, IndexW et région, quantums, implantation du
                        gamertag, variantes de corps, grammaires connues. Données dérivées du jeu
                        par outil de fabrication, plus tests de cohérence (toute valeur a une
                        preuve datée : Ghidra, mesure, film témoin). Aucune logique de lecture.
                        Une clé inconnue = erreur typée (principe 10).
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
- (ajout) Les types de sortie « stables entre couches » ont une adresse : un paquet de types
  sans dépendance (à la manière de `games/canonical`) que `grammar`, `facts` et `replay`
  produisent ou consomment, avec un test de contrat par type. Un changement de forme dans les
  faits ne casse pas silencieusement le rejeu ni le collecteur : il rougit le contrat.
- (ajout) Les crochets d'observation (les `Set*Hook` de filmdec) ne sont pas des globales : ils
  deviennent un observateur passé en paramètre du balayage (`nil` en production), et la table
  d'observation des largeurs devient un champ de cet observateur. C'est la condition du retrait
  de `LockProcessDecode` au pas 3.

## 5. Le profil de déchiffrage, l'objet central

```go
// profile.Profile : IMMUABLE, résolu une fois par film.
type Profile struct {
    Build        string          // en clair dans chunk_00 section 2 ("HI_1_13_0", "6.10026.18411.0") ;
                                 // la clé exacte, aujourd'hui lue par personne en production (ajout)
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

(ajout) Deux décisions que le texte initial laissait ouvertes :

- **`Profile` et `FilmContext`.** `FilmContext` absorbe le profil : il reste l'objet par film
  (film, chunks, bande de slots, registre, mémos), et gagne un champ `Profile` immuable résolu à
  la construction. Il n'y a pas de second objet. Ce qui est « résolu une fois » est le profil ;
  ce qui est « mémorisé à la demande » (registre parsé, bande de slots) reste un mémo de
  `FilmContext`, sans effet sur la grammaire.
- **Les inférences.** Une inférence n'est pas une valeur de profil. Là où le profil SAIT
  (catalogue, build), l'inférence ne tourne pas en production ; elle tourne dans un test corpus
  qui compare ce qu'elle voit à ce que le profil dit (principe 9 : le film est le témoin). Là où
  le profil NE SAIT PAS (carte hors catalogue, build inconnu), on ne devine pas : erreur typée
  et film mis de côté (principe 10). L'inférence de chaînes (`frame_chain_infer.go`) est d'une
  autre nature, elle résout l'archétype d'un transitoire à partir du flux et non une constante
  du format : elle reste dans `grammar`, sans état de paquet.

## 5 bis. Ce que le film déclare de lui-même, et ce qu'il ne déclare pas (ajout)

Relevé dans `.ai/V7.5/film_re/NOTE_CARTE_CHUNK00_2026-08-30.md`,
`GITHUB_RE_FINDINGS_EN.md` (« The wall »), `i0_layout.go`, et les deux pages Notion du backlog
« Format du film Theater, le bit de continuation » (30/08/2026) et « Le pied de film déchiffré »
(01/09/2026), qui disent la même chose et restent valables. `chunk_00` a QUATRE sections, le
dépôt n'en lit qu'une. Le pied du film n'est pas une grammaire non plus : le chunk de type 3 est
le flux des récompenses de score personnel (blocs de 60 octets : gamertag de l'acteur en UTF-16LE
en clair, slot, valeur, instant), le chunk de temps forts porte le kill feed. Ce sont des SOURCES
DE FAITS (identité directe par gamertag, ticks d'objectif, kills), à consommer dans `facts` ; ils
ne déclarent rien sur la façon de lire la trame d'état.

| Section de `chunk_00` | Contenu | Utile au profil ? | Lu en production ? |
|---|---|---|---|
| 1. Registre (832 000 o) | 50 blocs, 1 067 slots (archétype, composant) : ORDRE des composants par archétype, `kind`, drapeaux. Identique bit à bit entre films d'un même build. Aucune largeur, aucun handle, aucune implantation de champ | OUI, comme contrôle : l'ordre des composants d'un archétype est une déclaration du film que le profil doit respecter | oui (`registry.go` ; piège : `parseRegistry` divisait le fichier entier, faux positif au bloc 71 sur `00162144`, découverte notée non traitée) |
| 2. En-tête (604 o) | table de 123 u32 (valeurs 1 à 6, une par type d'événement, constante par build, cardinal qui croît avec le jeu : 119, 121, 122, 123) ; puis version `6.10026.18411.0`, build `HI_1_13_0`, saveur `release`, deux u32 inconnus | OUI : le BUILD en clair est la clé exacte du profil ; la table par type est, lecture la plus économique, une version de sérialisation par type (non prouvée : à lire dans l'exe, travail Ghidra) | NON. Seul le u32 à l'offset 0 (version majeure) est lu depuis le lot G |
| 3. Corps (~538 000 o) | dense, propre au film, gamertags en UTF-16LE ; non décodé | inconnu | non |
| 4. Queue | zéros | non | non |

Ce qui n'est PAS dans le film, et ne le sera jamais : les largeurs de champs, les quantums, les
bornes de carte, l'implantation des composants. Le moteur les calcule au chargement de la carte à
partir du tag scenario (`W = min(26, ceilLog2(ceil(60 * extent)))`, `FUN_140be9a14`) et les
lecteurs de composants sont des descripteurs à vtable compilés dans l'exécutable ; la réplication
référence les composants par ordinal de registre, pas par nom haché (négatif vérifié). C'est
« le mur » des notes de RE, et c'est pourquoi le principe 9 dit « du jeu, pas du film ».

Conséquences pour la cible :

- `source` lit les quatre sections et porte build, version, saveur et table par type sur le
  `Film` ; le profil se résout sur le build, avec la version majeure en repli pour les 5 films du
  cache sans section d'identification.
- Le corpus de témoins se choisit par BUILD (10 groupes mesurés sur 1 367 films), pas seulement
  par version majeure : deux builds à cardinal égal ont des tables différentes (`HI_1_8_0` et
  `HI_1_9_0`).
- Un item de recherche, hors du chantier : lire dans l'exe la fonction qui consomme la table par
  type. Si c'est bien une version de sérialisation par type, le film dit lui-même quand la
  grammaire d'un type change, et le profil peut s'en servir comme clé secondaire.

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
| (ajout) Objet par film | `filmdec.FilmContext` | absorbe `Profile` (section 5) |
| (ajout) Compte des globales | ratchet `filmdec_package_vars_test.go` (118, descente annoncée) | descendre à chaque famille du pas 2, atteindre 0 au pas 3 |
| (ajout) Verrou de décodage | `LockProcessDecode` + ratchet `decode_lock_held_test.go` | retirer au pas 3 et INVERSER le ratchet (le verrou devient interdit) |
| (ajout) Fabrication depuis le jeu | `cmd/mapquant-build` (bornes depuis les `.module`) | étendre au reste du profil ; test gamefiles « catalogue commis = catalogue régénéré » |
| (ajout) Lecture de `chunk_00` | `registry.go` (section 1) | sections 2 à 4 dans `source` ; build en clair porté par le film |
| (ajout) Frontière web | `normalizeReplayDocument`, `testDoc.ts` et son garde, badge de schéma | fixtures produites par Go, schéma zod à la frontière (section 12) |
| (ajout) Contrat OpenAPI | `make openapi-check`, `generate-types` | empreinte de forme du document côté Go (section 12) |

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
8. (ajout) Budget de temps par étape mesurée. `replay-equiv` rend déjà un compte et une
   empreinte par étape ; il rend aussi une durée, et un pas ne se clôt pas au-delà d'un budget
   fixé au pas 0 sur le corpus figé (le chantier cuisson n'a pas de gardien). Des `Benchmark`
   sur le lecteur de bits et les balayages chauds complètent, comparés par `benchstat`.
9. (ajout) Course détectée, pas supposée : le test « deux films en parallèle » du pas 3 tourne
   sous `go test -race` (avec `-gcflags=all=-d=checkptr=0` si un paquet tire le driver
   DuckDB), sinon il ne prouve rien.
10. (ajout) La double écriture du pas 1 est un kill-switch au sens de la règle 11 : date de
    bascule, date cible de retrait (le pas 3) et critère (compte de globales à 0) écrits dans
    le code.
11. (ajout) Consigner chaque gate local. La CI ne voit ni corpus gate ni équivalence : chaque
    clôture de pas inscrit dans le registre du chantier le commit, l'empreinte du corpus figé,
    la commande et le résultat. Un gate non consigné n'a pas eu lieu.
12. (ajout) Les goldens par version se posent au pas 0, pas au pas 6 : ce sont les seules
    preuves par version que la CI exécute, elles doivent exister avant le premier déplacement.
13. (ajout) Contrat entre les deux côtés (section 12) : fixtures produites par Go et consommées
    par vitest, empreinte de forme du document, posées AVANT le chantier parce qu'elles n'en
    dépendent pas et protègent aussi les lots G et H.

## 8. Trajectoire : des pas petits, chacun prouvable

Ordre fondé sur le risque : d'abord ce qui ne change aucun comportement, ensuite l'état, enfin
les frontières.

| Pas | Contenu | Preuve de clôture |
|---|---|---|
| 0 | Lot H : mesure calque par version ; choix des témoins PAR BUILD ; corpus figé pour `replay-equiv` ; (ajout) goldens par version, budget de temps, fixtures de contrat et empreinte de forme du document (section 12) | tableau et témoins au corpus gate ; (ajout) goldens et fixtures verts en CI |
| 1 | Build, version et carte portés par `Film` au chargement (les quatre sections de `chunk_00` lues dans `source`) ; `Profile` créé dans `FilmContext` avec les valeurs actuelles, encore recopiées dans les globales (double écriture temporaire datée, test d'égalité) | équivalence à zéro différence ; test « profil égale globales » |
| 2 | Les lecteurs reçoivent `Profile` en paramètre et cessent de lire les globales, famille par famille (positions, objets du monde, images-clés, temps forts, équipement) ; (ajout) les crochets d'observation deviennent un observateur passé en paramètre | équivalence à zéro différence à chaque famille ; mutation par valeur ; (ajout) ratchet des globales descendu à chaque famille |
| 3 | Suppression des globales, des install et restore, (ajout) du verrou `LockProcessDecode` et de la table d'observation sans verrou ; ratchet du verrou inversé ; deux films décodés en parallèle dans un test | équivalence ; test de concurrence sous `-race` ; compte de globales à 0 |
| 4 | Un seul lecteur de bits ; `killsource` et `weaponv3` passent par la façade ; ratchet « lecture brute » | équivalence ; ratchet vert à allowlist vide |
| 5 | Découpage en couches (`source`, `profile`, `grammar`, `facts`, `replay`) par déplacements purs sous `film/internal/`, ratchets de dépendance posés avant (méthode du lot E) ; (ajout) critère d'entrée : zéro branche filmdec ou replay non fusionnée, pas réalisé d'un bloc | équivalence ; `git mv` sans ajout ni suppression ; gate corpus |
| 6 | Empreinte étendue et révisions par couche ; (ajout) types de sortie entre couches avec test de contrat | une mutation de source rougit l'empreinte |
| 7 | Scission des fichiers de plus de 500 lignes (`traverse.go` en premier) sans changement de comportement | équivalence ; baseline lint non accrue |
| 8 | Exploiter le profil : là où le lot H a montré une divergence par version, brancher (nouveau comportement, donc montée de révision, de schéma et recuisson) ; (ajout) profil dérivé du jeu par outil de fabrication, politique de version inconnue active | corpus gate : gains attendus, zéro perte ; oracles ; (ajout) retour arrière défini avant la recuisson (section 10) |

Les pas 1 à 7 ne changent pas un octet des artefacts : c'est ce qui rend le chantier sûr et
interruptible. Le pas 8 est le seul qui produit de la valeur visible, et il n'est possible que
parce que les sept précédents ont rendu la version et la carte explicites.

(ajout) Le second chantier, celui de la publication (section 11), commence après le pas 5 et
dans sa propre fenêtre : il a besoin de la frontière `facts` et ne doit pas partager l'oracle
d'équivalence avec les pas 1 à 7.

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
| Conflits avec les branches vivantes du décodeur | fenêtre dédiée, aucune autre branche filmdec ou replay en vol (méthode du lot E) ; (ajout) mesuré le 2026-09-12 : quatre branches non fusionnées dans `feat/v75` touchent filmdec ou replay (`wt/decodeur-sous-titre` 993 fichiers, `wt/btb-2025-abstention` 16, `wt/ti11-cadre` 6, `wt/mesure-ti9` 3) ; le critère d'entrée du pas 5 est mesurable (`git branch --no-merged feat/v75` filtré par `git diff --name-only` sur ces chemins = vide) |
| (ajout) Le profil passé partout ralentit les boucles chaudes | budget de temps par étape (contrôle 8) ; le profil se passe par pointeur ou se lit une fois en tête de balayage, jamais dans la boucle de bits |
| (ajout) Retour arrière du pas 8 | les pas 1 à 7 se défont par `git revert`. Le pas 8 recuit sous un nouveau schéma : avant la recuisson, tag git du binaire précédent, artefacts précédents conservés jusqu'à validation du corpus gate, et la release ne copie les bases qu'après (règle « release = copie des bases locales ») |
| (ajout) Un build neuf arrive en production | principe 10 : erreur typée, compteur expvar par build, film mis de côté ; procédure d'ajout d'un profil (outil de fabrication, témoin au corpus, entrée présumée puis prouvée) écrite au pas 8 |
| (ajout) Les fixtures de contrat prolifèrent | un mini-film par build supporté, taille bornée ; le garde de `testDoc.ts` interdit toute fixture hors du jeu |

## 11. Le second chantier : la publication (ajout)

La publication n'est pas repensée dans la même fenêtre que les pas 1 à 7 : l'équivalence octet
pour octet exige un éditeur figé, et changer les deux côtés à la fois rend l'oracle muet. Mais la
cible du décodeur se dessine avec les problèmes de l'artefact en tête, parce que la couche `facts`
est précisément ce qui les résout. Trois problèmes constatés, trois cibles :

| Constat (2026-09-12) | Cible |
|---|---|
| Un `SchemaVersion` unique (53) couplé à la recuisson totale : un correctif sur UNE carte (v53, Live Fire) oblige à recuire tout le parc, et la recuisson totale a déjà été bloquée par la bombe mémoire de `NamedEventsFrom`. Les entrées décodées sont sérialisables (`inputs_000d5950.bin.gz`), mais seulement dans un test | Les FAITS sont persistés par film avec la révision de la couche qui les a produits. Un changement de publication se rejoue depuis les faits en secondes, sans redécoder (15 s par film aujourd'hui) ; un changement de grammaire ne recuit que la couche touchée. Le numéro de schéma ne dit plus que la FORME du document |
| « Un champ optionnel n'incrémente pas le schéma » : deux artefacts au même numéro peuvent porter des jeux de champs différents selon leur date de cuisson ; le web les complète à la frontière de normalisation, où les écarts se cachent. C'est la source des régressions « entre deux versions de schéma » | Révision par calque portée par le document (les compteurs de couverture en portent déjà une partie) ; la présence d'un calque se lit dans sa révision, jamais dans l'absence d'un champ. Un champ optionnel ajouté sans révision de calque rougit l'empreinte de forme (section 12) |
| Deux types pour un même document : `analysis/replay.ReplayDocument` (avec numéro) et `domain/replaydoc` (sans numéro, la version voyage dans l'en-tête `X-Replay-Latest-Schema-Version`) | Un seul type publié, produit par `replay/` depuis `facts/`, qui porte son numéro et ses révisions de calque ; l'en-tête HTTP ne porte que la version du producteur |

Ordre : après le pas 5 (la frontière `facts` existe), dans sa propre fenêtre, avec son propre
corpus d'équivalence. Ce chantier est le seul qui touche le web, et il ne le touche qu'à la
frontière de normalisation.

## 12. Tests des deux côtés : la frontière Go / web (ajout)

Les preuves sont asymétriques : côté Go, goldens et gates locaux ; côté web, des types générés
d'OpenAPI qui n'existent plus à l'exécution et une fixture écrite à la main qui déclare
`schemaVersion: 1`, un document que le serveur n'envoie jamais (le commentaire de cette fixture
énonce lui-même le principe inverse). Aucune preuve ne traverse la frontière, et la CI ne voit ni
corpus ni équivalence. Six garde-rails, tous exécutables en CI, à poser AVANT le chantier :

1. **Fixtures de contrat produites par Go, consommées par vitest.** Un test Go cuit le mini-film
   déterministe (`testdata/minifilm_000d5950/`) et fige un document JSON par version de schéma
   supportée, sous un dossier partagé par les deux applications. Vitest fait passer chaque
   fixture par `normalizeReplayDocument` et par les logiques pures des calques (`*_logic.ts`).
   Un mini-film par build supporté, taille bornée.
2. **Matrice de compatibilité déclarée.** Le web nomme la version minimale qu'il sait afficher.
   Toute fixture au-dessus doit rendre ; toute fixture en dessous doit donner l'état « à
   recuire » du badge admin, jamais une exception. Le badge existe (lot A du 2026-09-11) ; la
   matrice et son test manquent.
3. **Contrat à l'exécution.** Un schéma zod dérivé d'OpenAPI (zod est déjà une dépendance)
   valide le document à la frontière de transport, actif en test et derrière le badge admin.
   Il attrape le champ renommé ou le type changé que TypeScript ne voit plus une fois compilé.
4. **Empreinte de forme côté Go.** Un test hache la forme du document (champs, balises JSON,
   caractère optionnel) et échoue si elle change sans montée de `SchemaVersion` ou de révision
   de calque, ou si une montée arrive sans entrée dans `document_chronicle.go`. Miroir de
   l'empreinte du décodeur : il attrape « j'ai changé un champ et oublié le numéro ».
5. **Les anciens goldens ne se régénèrent jamais.** À chaque montée, on ajoute le nouveau et on
   garde l'ancien, avec un test qui vérifie que `replaybuild.Digest` le classe périmé et que le
   point d'écriture unique (`writeArtifactBytes`) refuse toute rétrogradation. Le garde existe,
   la preuve par version manque.
6. **Ratchet sur les fixtures.** Le garde de `testDoc.ts` s'étend : aucun littéral de version de
   schéma dans les tests web hors du jeu de fixtures produites par Go.

Ces six points ne dépendent pas du chantier décodeur ; ils protègent aussi les lots G et H en
cours. C'est le pas 0.

## 13. Forme : où cette décision doit vivre (ajout)

Les couches, le profil, la porte unique aux octets, la politique de version inconnue et la
séparation faits / publication sont des invariants durables : ils ont la taille d'un ADR
(le prochain numéro est 0034), pas d'un document `.ai/`, dont CLAUDE.md dit lui-même qu'il tourne
plus vite qu'il n'est maintenu. Ce document reste la chronique et l'état des lieux datés ; l'ADR
porte les décisions et leurs ratchets. Les chemins de ce document sont déjà mixtes (killsource
et filmcache sous `games/halo_infinite/film/`, filmdec et replay encore sous `analysis/`) : une
raison de plus pour que l'ADR nomme les paquets cibles et non les paquets du jour.
