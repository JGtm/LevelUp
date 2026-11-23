# E.3 - Nom des options de variante : instruction (sans code)

Dépôt : décodeur de film, tête de `feat/v1.6.0` (commit 785c38e). Rien n'a été modifié dans le dépôt.
Chemins relatifs à ce dépôt. Films du corpus désignés par un libellé neutre : film A (partie classique,
10 options) et film B (minijeu asymétrique, 19 options). Marques : « vérifié » = relu ou rejoué ;
« déduit » = raisonné sans preuve, avec de quoi le vérifier.

## Réponse courte

- Le film ne donne PAS les noms des options. Vérifié sur deux films, par recherche à tous les décalages de bits.
- Les fichiers du jeu lisibles par les briques ne les donnent pas non plus. Vérifié sur les archives `.module`
  de `deploy/any` (groupes `mat `, `snd!`, `prt3`, `lsnd`, `coll`, `phmo`, `mode`, `sbsp` exclus).
- Le jeu les porte très probablement comme chaînes de son exécutable (déduit), illisible sur disque ici
  (accès refusé, vérifié). Le nom doit donc venir d'une table bâtie une fois par le mainteneur, à la
  manière du catalogue des dégâts (`damagetag`), chaque ligne se contrôlant par son hachage.
- Piège de cadrage : le document de rejeu ne publie AUJOURD'HUI aucune option (vérifié). Publier le nom
  « à côté du hachage » demande d'abord de publier les options. Trois petits items de code.

## 1. Ce que le décodeur lit et publie aujourd'hui

Vérifié sur pièces.

- Lecture : `internal/grammar/film_infos_de_match.go`. Les options sont au chemin Bond `3.9.0.i` de la
  variante : `.0.0` le hachage du nom, `.1.0` huit octets de valeur (modèles `optionNom` et `optionValeur`,
  lignes 85-86 ; rangement par `ajouterUneOption`, lignes 189-215). Le chemin est « mesuré contre un décodeur
  tiers », pas relu dans l'exécutable (en-tête du fichier, lignes 5-8).
- Structure : `internal/profile/infos_de_match.go`, `OptionDeVariante{Hachage uint32, Genre uint8, Valeur float64}`
  (lignes 25-33) dans `InfosDeMatch.Options` (ligne 81). Genre 2 = booléenne, genre 3 = nombre (flottant 32
  bits) ; un autre genre ou un nombre non fini n'est pas rendu.
- Où ça vit : `profile.FilmIdentity.InfosDeMatch` (`internal/profile/identite.go:104`), donc dans la section 2
  du fichier de faits (JSON de l'identité, `replay/filmfacts_fichier.go`, schéma 20). Façade : `decfilm.ReadMatchInfo`
  (`decfilm/infos_de_match.go:31`).
- Document de rejeu : la liste des options n'y est PAS. Seul `rules` (`replay/document.go:429`,
  `replay/document_regles.go`) en tire une valeur : `flagReturnS`, par le hachage `0x2b16b76f`
  (constante `hachageDelaiDeRetourDuDrapeau`, ligne 16). Aucun autre lecteur de `Options` hors tests.
- Hachage : 32 bits. La fonction est un murmur3 x86 32 bits, graine 0, sur le nom normalisé (majuscules en
  minuscules, `-` et espace en `_`, saut de ligne en `#`). Source de la normalisation : commentaire de
  `internal/grammar/objectif_bombstate_test.go:36-45` (relevé Ghidra de `FUN_140748a74`) ; le commentaire de
  `OptionDeVariante` dit « murmur3 du nom en minuscules ».
  Vérifié : `murmur3("ctfoptions_flagreturntime") = 0x2b16b76f`, rejoué par un programme hors dépôt.
  Vérifié : AUCUNE implémentation de cette fonction n'existe dans le code non test du dépôt (les constantes
  `0xcc9e2d51` et `0x85ebca6b` n'apparaissent que dans des commentaires). Il faut l'écrire (une vingtaine de lignes).
- Mesure, deux films (lecture de `chunk_00` décompressé par `decfilm.ReadFilmChunk`, puis `ReadMatchInfo`) :
  film A, variante lue, 10 options, toutes des nombres ; film B, variante lue, 19 options (5 booléennes, 14
  nombres). Aucune des 29 n'est `0x2b16b76f` (ces parties ne sont pas des CTF). Vérifié.
- Le nom `CTFOptions_FlagReturnTime`, écrit dans `replay/document_regles.go:15`, `GLOSSARY.md:125` et
  `FORMAT.md:604`, n'a pas de preuve de jeu dans le dépôt : la note d'inventaire de la phase C1 l'attribue au
  parseur tiers (déduit). Il passera par la même règle que les autres : un nom sans preuve du jeu ou du film
  reste un hachage.

## 2. Le film porte-t-il le nom en clair ?

Non. Vérifié sur film A et film B.

- Le `chunk_00` décompressé fait 1 973 120 octets sur les deux films. C'est un flux de bits (lecture MSB
  d'abord, `internal/source/bits.go`) : une recherche d'octets ne voit que les chaînes alignées par hasard.
  Contrôle : le nom de la variante n'est trouvé qu'au décalage de bits 3.
- Méthode : pour chacun des 8 décalages, extraction de toutes les suites imprimables (4 caractères ou plus ;
  environ 3 300 par film à l'alignement 0), hachage de chaque suite (brute, en minuscules, et chaque mot de
  3 caractères ou plus) avec le murmur3 ci-dessus, comparaison aux hachages d'options du film.
  Résultat : 0 hachage retrouvé sur 10 (film A) et sur 19 (film B).
- Aucune chaîne contenant « option » dans le flux, hors trois noms de scripts de mode (un script d'options
  par mode, `SlayerOptions`, `MinigameOptions`, `Option.lua`, `OptionCasual.lua`, `OptionAsymmetrical.lua`).
  Ce sont des noms de scripts, pas des noms d'options.
- Conclusion : le film ne porte que le hachage (cohérent avec le commentaire de `OptionDeVariante`). La voie
  « chaîne lue dans le film » de la décision est vide.
- Limite : deux films, un seul build vu. Non établi pour un autre build.

## 3. Où le jeu donne les noms

### Vérifié négatif

- Clone local `halo-infinite-files-reader` (étiquette v1.2.0) : lit les archives `.module` (`himodule`), les
  variantes de carte Forge (`mapvar`) et les maillages de navigation (`hinavmesh`). Il n'interprète ni variante
  de partie ni tag d'options (son README : « not a generic tag decoder »).
- Installation du jeu (44 archives sous `deploy/any`) : parcours de tous les tags (groupes lourds exclus, jusqu'à
  32 Mo) de `common`, `multiplayer`, `multiplayer_r3`, `forge`, `levels` et `globals`, avec le même hachage sur
  chaque suite de 3 caractères ou plus. Résultat : 0 sur les 30 hachages testés (les 29 des deux films plus
  `0x2b16b76f`), y compris dans les 151 tags `hsc*` et les 293 tags `luas` (Lua compilé de l'interface). Le tag
  `hsc*` des scripts de drapeau contient `FlagReturnTimerMonitorThread` (un nom de fonction, pas l'option).
- L'exécutable n'est pas lisible sur disque (accès refusé). Le relevé Ghidra des notes de recherche a été fait
  sur la mémoire ; le serveur Ghidra n'a pas répondu dans cette session.

### Déduit

- Les noms vivent dans l'exécutable : le jeu enregistre ses noms par `FUN_140748a74` (le murmur3) sur des
  chaînes littérales (`CtfStats_FlagCaptures` y est enregistré ainsi, notes de recherche sur le mode score).
  Un enregistrement par nom d'option est probable. Vérification : relever, dans le relevé Ghidra, les appelants
  de `FUN_140748a74` qui passent une chaîne contenant `Options_`, puis rejouer le hachage.
- Les noms pourraient aussi être dans des `.module` non parcourus ou sous une autre forme. Vérification : parcourir
  les groupes exclus et les sous-jeux.

### Comment les tables de noms existantes sont entrées (vérifié, `NOTICE.md`)

| Table | Origine | Crédit |
|---|---|---|
| `damagetag/data/jpt_ids.txt`, `labels.tsv` | liste établie le 2026-07-26 par l'outil du mainteneur depuis les en-têtes des archives de son installation ; noms d'affichage du jeu ; classes et statuts du mainteneur | aucun tiers |
| `medalname`, `damageeffects`, `reponsesdejeu` | prises dans le parseur tiers (MIT), en-tête de crédit gardé, texte de licence dans le dossier de données | crédit au parseur, licence copiée |
| `killicon/data/rules.tsv` | écrite par le mainteneur, une justification par règle | aucun tiers |

Pour les options, la décision déjà prise exclut la table du parseur tiers. La voie qui reste est celle de
`damagetag` : une table de noms de jeu, bâtie par le mainteneur depuis son installation (ou son relevé de
l'exécutable), qui ne contient que des noms (identifiants et libellés, comme les 468 identifiants de `jpt!`).
Le crédit de la méthode au tiers va dans l'en-tête du fichier, comme le plan le dit. Le décodeur ne lit pas le
jeu : la table est une donnée embarquée (`go:embed`).
Variante écartée : une table fournie par l'hôte à chaque cuisson. Le document est rejoué depuis les faits et doit
rester autoportant ; une table embarquée suit la version du décodeur, une table de l'hôte devrait entrer dans
l'empreinte de cuisson. Déduit.

## 4. Plus petite forme de publication

- Où : un champ `options` dans la section `rules` du document (`MatchRules`, `replay/document_regles.go`), une
  entrée par option lue, dans l'ordre du film : `{hash, kind, value, name?}`. `name` est omis quand le nom n'est
  pas prouvé. C'est un ajout compatible : un lecteur qui ignore le champ ne voit aucun changement.
  Pourquoi dans `rules` : la couche existe, avec sa révision (`replay/layers.go:180`, `grammar.Rev`), et `rules`
  est déjà la porte des réglages de variante. `MatchRules.vide()` doit tenir compte de `options`.
- Compteur : `coverage.rules.options` avec `read`, `named`, `unnamed` (déduit : la clé exacte se choisit en suivant
  `coverage.flagCarries`). Une option sans nom reste publiée avec son hachage seul et se compte.
- La table : `variantoption/data/noms.tsv` (hachage hexadécimal, nom), dans un petit paquet qui porte aussi le
  murmur3 et la normalisation. Même gabarit que `medalname` et `damagetag` : en-tête de provenance, ligne de
  crédit, ligne dans `NOTICE.md`.
- Qui la bâtit : le mainteneur, hors du module (outil de recherche derrière l'étiquette `gamefiles` s'il lit les
  archives ; relevé de l'exécutable sinon). Ce n'est pas du code du décodeur.
- Le test de contrôle : pour chaque ligne, `murmur3(normalise(nom)) == hachage` ; pas de hachage en double ;
  nom non imprimable ou trop long refusé.

## 5. Découpage en items de code

### E.3a - Hachage et table (aucun changement de document)
- Fichiers : `variantoption/murmur.go`, `variantoption/table.go`, `variantoption/data/noms.tsv`, `NOTICE.md`.
- Preuve : test du hachage (`ctfoptions_flagreturntime` donne `0x2b16b76f`) ; test « chaque ligne se rejoue par
  son hachage » ; test de normalisation (majuscule, `-`, espace, saut de ligne).
  La table ne contient que des noms dont une source du jeu est établie. Si aucune ne l'est à l'exécution, le
  fichier est vide et l'item le dit : le remplissage passe en `[!]`, pas le code.
- Effet sur le document : aucun.
- Prérequis hors code : la collecte des noms (point 6).

### E.3b - Publier les options dans `rules`
- Fichiers : `replay/document_regles.go` (type d'option, champ, `vide()`, construction depuis `InfosDeMatch.Options`
  avec la table), `replay/document_chronicle.go` (une entrée), `FORMAT.md` et `GLOSSARY.md`.
- Preuve : test de `rulesOf` (nom présent, nom absent, option booléenne, option nombre) ; sur film A et film B, le
  document porte 10 et 19 options, `name` absent partout tant que la table est vide.
- Effet sur le document : `rules.options` apparaît sur tout film dont la variante est lue. Changent : les
  références d'équivalence qui hachent le document et `replay/testdata/document_shape.golden` (ligne 1083,
  `Rules`). La révision de la couche `rules` monte, une seule fois pour la v1.6.0 (règle 19 de LevelUp).
  Déduit ; à confirmer par `TestCalquesNePortentQueLesRevisionsConnues` (`replay/layers_test.go:111`).

### E.3c - Compteur des options sans nom
- Fichiers : `replay/` (la section `coverage` du document), `FORMAT.md`.
- Preuve : sur les deux films, `read` = 10 et 19, `named` = lignes de la table retrouvées, `unnamed` = le reste ;
  test de cohérence `read = named + unnamed`.
- Effet sur le document : une clé de `coverage` apparaît ; mêmes références que E.3b. À passer avec E.3b en un
  seul passage d'équivalence.

Ordre : E.3a, puis E.3b et E.3c ensemble. E.3a se livre même table vide ; le remplissage est le seul point
qui dépend d'une source du jeu.

## 6. Ce que je n'ai pas pu établir

- La source d'un nom dans le jeu : aucune trouvée. Ni le film, ni les `.module` testés ne les portent. Reste
  l'exécutable, illisible sur disque ici et non relu (Ghidra indisponible). C'est ce qui décide si E.3 publie
  des noms en v1.6.0 ou seulement le code et la politique.
- La provenance du seul nom déjà écrit (`CTFOptions_FlagReturnTime`) : sans preuve de jeu dans le dépôt.
- Les groupes de tags exclus du parcours (matériaux, sons, collisions, modèles, rendu) et les sous-jeux de
  campagne : non parcourus.
- Un autre build de film : un seul build vu.
- La clé exacte du compteur dans `coverage` et l'ampleur du changement des références d'équivalence : à voir à
  l'exécution.
- Le nombre d'options que la table couvrirait : le backlog parle de 98 noms chez le parseur tiers (rapporté, non
  revérifié) ; deux films donnent 29 hachages distincts, aucun nom prouvé.
