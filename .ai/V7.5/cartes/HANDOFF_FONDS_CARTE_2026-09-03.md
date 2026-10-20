# HANDOFF — Les fonds de carte du rejeu 2D

> Rédigé le 2026-09-03, à la fin d'une session qui a livré cinq lots sur `feat/v75`.
> Destinataire : l'agent qui reprendra le sujet. **Lis la section « Ce qui est établi » avant de
> creuser quoi que ce soit** : plusieurs pistes évidentes ont déjà été suivies, dont deux
> réfutées par la mesure et deux conclusions FAUSSES que j'ai moi-même tirées.

## 1. De quoi on parle

Le rejeu 2D place des joueurs à des positions du monde sur une carte vue du dessus. Il lui faut
donc, par carte, **un plan à plat + un calage** — la règle qui dit quel pixel vaut quel mètre.

Ces couples vivent dans `data/titles/halo_infinite/reference/map_backgrounds/` : **106 paires**
`{clé}.png` + `{clé}.json`. Le sidecar porte `calibration` avec `metersPerPixel`, `originX`,
`originY` (coin HAUT-GAUCHE), `widthPx`, `heightPx`, et la convention écrite en clair :

    xMonde = originX + (px + 0,5) * mpp
    yMonde = originY - (py + 0,5) * mpp

Structure Go : `replay.MapBackgroundCalibration`,
`apps/go-api/internal/analysis/replay/map_background.go:64`.

## 2. Ce qui est ÉTABLI — ne le refais pas

### 2.1 `static/maps/halo_infinite/` NE CONTIENT PAS de plans

129 fichiers, dimensions **560x320 (97), 560x316 (31), 530x316 (1)** — un format FIXE. Ce sont
les **vignettes d'aperçu d'asset** (confirmé par l'utilisateur le 2026-09-03), servies à la vue
match par `staticAssetURL('map', ...)`.

Contrôle visuel fait : `Argyle.jpg` est une **capture de jeu en perspective, au niveau du sol**.
À comparer avec `map_backgrounds/catalyst.png`, qui est un plan à plat en niveaux de gris. Aucun
calage ne peut faire tomber un joueur au bon endroit sur une image en perspective.

**Il n'y a RIEN à récupérer de ce dossier.** J'ai perdu une demi-session à croire le contraire.

### 2.2 Le fond se résout par TROIS chemins, pas un

`service/replay_map_background.go:resolveBackgroundKey` essaie, dans l'ordre :

1. la clé **map_id** (cartes Forge) — la présence du sidecar décide ;
2. l'**index par NOM** (`replay.MapBackgroundIndex`), qui rattrape les cartes republiées sous un
   nouvel asset et les cartes natives (keyées par module installé) ;
3. depuis le 2026-09-03, l'**héritage variante vers base** (`_-_ranked`, `_heavies`, `_firefight`).

**Compter les cartes « sans fond » sur la seule clé map_id donne un résultat FAUX.** C'est mon
erreur n°2 : j'ai annoncé 34 manquantes là où il y en avait 23, et 11 des cartes que je citais
étaient déjà servies (The Pit, Argyle, High Ground, Isolation, Perilous, Shiro, Rat's Nest,
Cliffside, Goliath, Dynasty, Domicile, Critical Dewpoint).

### 2.3 Le calage automatique depuis les zones est RÉFUTÉ, pas inexploré

Hypothèse testée : « emprise de l'image = boîte des polygones de zones + marge constante ».
Jeu de contrôle : les **37 cartes** qui ont zones ET calage publié. Résultat : **15 reproduites,
22 échouées**, écarts jusqu'à **286 m** (The Pit +220/-286, Kaiketsu -215, High Ground -201).

Cause lisible dans le code : le cadre publié est la boîte de la **matière dessinée + 6 m**
(`himap.CadreUtile`, `MargeCadreUtile`, `apps/go-api/internal/himap/cadre_utile.go:31`) — une
grandeur qui **exige la géométrie du jeu**. Les échecs sont soit des cartes dont les zones
nomment le canevas entier (aspect 1,000 : Smallhalla, Sylvanus, Dynasty, Insolence), soit des
décors qui débordent les lieux nommés.

La piste « enveloppe des positions jouées » est **intestable** : `map_positions_jouees.json` ne
couvre qu'**UNE** carte (Dredge).

### 2.4 Ce qui MARCHE : la variante hérite de sa base — mesuré

- 19 paires variante/base dont les deux fonds sont publiés : 12 partagent déjà le même sidecar,
  et les **7 cuites séparément s'accordent à 0,012 m en originX et 0,021 m en originY** ;
- 10 paires présentes au catalogue de zones : emprise monde identique à **0,000 m**.

Livré le 2026-09-03 (`map_background_index.go`). Sens UNIQUE variante vers base ; une variante
qui a son propre fond garde le sien ; une base ambiguë ne se transmet pas. **Gain : +8 cartes.**

### 2.5 Trois faux manques déjà identifiés

- **Narrows** — son fond EXISTE sous `944396dd`, dont le sidecar ne déclare aucun `mapNames`.
  Une ligne suffirait, mais le renommage appartient au chantier registre.
- **Dévissage** — nom FR de Cliffhanger, qui a son fond.
- **`allaheim Firefight.jpg`** — nom de fichier TRONQUÉ, doublon de `Vallaheim Firefight.jpg`.

## 3. LA QUESTION OUVERTE, et c'est la seule

**23 cartes n'ont pas de fond, dont 16 au catalogue de zones. Lesquelles sont du BTB ou du
Firefight ?** Ces deux modes ne sont pas supportés par le produit (utilisateur, 2026-09-03) : y
cuire un fond serait du travail perdu.

**Ce qui ne permet PAS de le déterminer** — testé, tout est négatif :

- l'inventaire UGC (`.ai/V7.5/cartes/inventaire_rotation_ugc_2026-08-27.json`) : son champ
  `famille` ne distingue que `forge` / `native` ;
- le champ `canevas` : ce sont des toiles de Forge (`fo05_desert`, `fo11_blank`, `fo03_space`),
  pas des tailles de partie ;
- les vignettes de `static/maps` : elles existent pour TOUTES les cartes, quel que soit le mode.

**Ce qui le permettrait** : la base des matchs. `match_registry` porte la carte ET la playlist.
Il n'existe pas d'outil de requête ad hoc utilisable (pas de client DuckDB sur le PATH), donc
**la première tâche est d'en écrire un petit** : pour chaque carte sans fond, les playlists où
elle a réellement été jouée. Go, CGO (gcc msys64), sous `apps/go-api/cmd/`.

Liste à trancher — au catalogue de zones (16) :

> Apostle, Aqueduct, Argyle, Argyle - Ranked, Arrival, Daimyo, Dead Water, Detachment, Exiled,
> Exiled Firefight, Kusini Bay, Kusini Bay Firefight, Last Broadcast, Suban, Vallaheim,
> Waterworks

Hors catalogue de zones (8) :

> **Live Fire** et Live Fire - Ranked, Cole Protocol, Munera Platform H6, Munera Platform W4,
> Out With A Bang, House of Reckoning, TFF Night Of The Undead

Les deux `Firefight` s'éliminent par leur nom. Pour les autres, la requête tranche.

## 4. Par où commencer

1. **L'outil de requête** (section 3). Il resservira à chaque inventaire.
2. **Live Fire d'abord.** Ses réglages de cuisson sont écrits dans `map_fond_reglages.json` sous
   `sgh_interlock`, mais **la cuisson n'a jamais été gatée**. C'est de l'arène, et probablement
   la plus jouée des cartes sans fond : le meilleur rapport effort/gain du lot.
3. Les autres cartes en mode supporté, une fois la liste triée.
4. **Cole Protocol, Munera H6/W4, Out With A Bang** sont bloquées faute d'ancre d'objectif
   (cf. commit `bb7f8ae4c`) — c'est un obstacle distinct, à instruire séparément.
5. **Vallaheim** est un cas laissé ouvert VOLONTAIREMENT : sa variante Firefight a un fond, la
   base non, et leurs zones sont identiques à 1 mm. L'héritage inverse (base qui hérite de sa
   variante) n'a pas été ouvert, parce que la boîte du fond est celle du DÉCOR et que rien ne
   prouve que la variante Firefight ne pose pas d'objets hors de l'emprise de la base.

## 5. Pièges — chacun a déjà mordu quelqu'un

- **NE FABRIQUE AUCUN CALAGE APPROXIMATIF.** Un fond mal calé est PIRE que pas de fond : les
  joueurs apparaissent à côté du décor, et rien ne dit lequel des deux ment. `coversPlayedArea`
  (`mapBackground.ts`) écarte un fond qui ne recouvre pas la zone jouée — c'est un filet, pas une
  autorisation à approximer.
- **IL N'Y A PLUS AUCUN REPLI.** Le sol reconstruit a été SUPPRIMÉ le 2026-09-03 (demande
  utilisateur : il ne devait plus pouvoir cuire un calque de ~45 000 cellules par inadvertance).
  Une carte sans fond calé n'a donc plus rien sous les joueurs, hors props Forge (3,4 % du
  terrain). Ton travail est le seul chemin.
- **LA PASSE NATIVE N'EST PAS REPRODUCTIBLE d'une machine à l'autre.** Rejouée depuis un autre
  chemin d'installation Steam, elle rend un fichier **plus petit de 645 Ko** : le champ `source`
  écrit le chemin d'installation au dépôt, et la section native perd des sommets. **Restaure
  toujours la section native depuis HEAD** et ne rejoue que ce que tu produis. À instruire avant
  toute reconstruction native complète.
- **NE RECUIS PAS LES ARTEFACTS EN LOT.** Quatre sinistres mémoire documentés. Toute cuisson de
  masse se DEMANDE d'abord.
- L'inventaire UGC date du **2026-08-27** : une carte jouée depuis n'y est pas.

## 6. Ce qui a été livré sur `feat/v75` le 2026-09-03

| lot | contenu |
|---|---|
| `callouts(forge)` | catalogue de zones ouvert aux cartes UGC — 61 cartes, 2392 zones |
| `callouts(lexique)` | les 266 identifiants de libellé résolus — **100 % des zones nommées**, 64 cartes |
| `fonds(variantes)` | héritage variante vers base — +8 cartes |
| `rejeu(sol)` | **suppression** du sol reconstruit — 586 lignes, plus aucun calque qui cuise par inadvertance |
| `rejeu(doc)` | le commentaire du fond de carte disait « seules 21 en ont » : il y en a 106 |

## 7. Leçon de méthode, et elle m'a coûté deux fois

J'ai conclu **deux fois** depuis un seul dossier, et deux fois l'utilisateur a redressé :

1. j'ai compté les fonds manquants sur la seule clé `map_id`, en ignorant la résolution par nom ;
2. j'ai pris les vignettes de `static/maps` pour des plans dessinés, sur la foi de leur nom.

Dans les deux cas, **une minute de vérification aurait suffi** — lire `resolveBackgroundKey`,
ouvrir une image. Sur ce sujet, un dossier ne prouve rien : il faut lire le code qui le consomme,
et regarder ce qu'il y a dedans.
