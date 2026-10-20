# HANDOFF — LES DUMPS : ce qu'on a, ce qu'il faut, ce qu'on perd

> Écrit le 2026-07-31, avant changement de PC. Sujet à traiter **ensemble**, ensuite.
> Complète `SESSION_CAPTURE_AVANT_PC.md`, qui donne la liste d'actions ; celui-ci donne le
> RAISONNEMENT et l'inventaire large.

---

## LA RÈGLE QUI COMMANDE TOUT

Une donnée se range dans l'une de trois cases, et c'est la seule chose à retenir :

| case | exemples | conséquence |
|---|---|---|
| **reproductible sans rien** | artefacts de rejeu, structures de carte figées | on ne transporte que par confort |
| **reproductible avec le jeu installé** | bornes de carte, fonds de carte de nouvelles cartes | on transporte le jeu (fait : 31 modules) |
| **IRREPRODUCTIBLE sans Cheat Engine** | table des désérialiseurs, captures mémoire | **on transporte, ou on perd définitivement** |

La troisième case est la seule qui presse. Tout le reste peut attendre.

---

## CE QUI EST DÉJÀ SUR LA CLÉ — 3,1 Go dans `E:/LevelUp_rejeu2D/`

| quoi | taille | pourquoi |
|---|---|---|
| `retro_ingenierie/HI.rep` + `HI.gpr` | 2,7 Go | le projet Ghidra : des mois d'analyse, 369 désérialiseurs nommés |
| `retro_ingenierie/HaloInfinite.exe` | 84 Mo | sans lui le projet Ghidra s'ouvre sur rien |
| `jeu_deploy_ds/multi/` | 162 Mo | 31 modules de niveau — suffisent à `mapquant-build`, **sans le jeu installé** |
| `captures_cheat_engine/` | 164 Mo | les 40 captures existantes, dont les 199 `.mvar` et les fichiers non versionnés |
| `scripts_cheat_engine/` | — | les 11 scripts existants |
| `filmdec_deser_table.lua` | — | **le script neuf** : la table complète, tous archétypes, en une passe |
| `POC_reference_rejeu2D.html` | 4,1 Mo | la référence d'écran, qui vivait dans un répertoire temporaire |

Et dans `E:/data/` : les 949 films (23 Go), les bases DuckDB rafraîchies, les données de
référence de carte, les deux artefacts de rejeu.

**54 Go restent libres.** L'espace n'est pas une contrainte — c'est important pour la suite.

---

## LA MATRICE — CE QUE CHAQUE PLAN EXIGE, ET CE QU'ON A

C'est la réponse à « est-ce que j'ai tout ? ». Une ligne par étape qui a besoin de données.

| plan · étape | données nécessaires | statut |
|---|---|---|
| **Variables jetées** — *tout le plan* | les films déjà en cache | ✅ **rien à faire** |
| **Capacités** — relire l'index sur 6 bits | les films déjà en cache | ✅ **rien à faire** |
| **Capacités** — compléter la table | un match avec **surbouclier et camouflage** + son **relevé terrain** | ⚠ à capturer |
| **Capacités** — état actif `i57` | le même match, le même relevé | ⚠ à capturer |
| **Objectifs** — brancher ce qui dort | les bases DuckDB + `map_objectives.json` | ✅ **sur la clé** |
| **Objectifs** — sémantique des zones | les bases DuckDB | ✅ **sur la clé** |
| **Objectifs** — décoder l'entité | **table des désérialiseurs** + Ghidra + les deux films | ⚠ table à capturer |
| **Objectifs** — affichage CTF | film `64e8adfa` | ✅ en cache, **artefact construit** |
| **Objectifs** — affichage Strongholds | film `696a9d7c` | ❌ **absent, à télécharger** |
| **Finalisation** — lots 1 à 4 | ce qui est déjà là | ✅ |

**Conclusion : il manque exactement trois choses.** Le film `696a9d7c`, la table des
désérialiseurs, et une capture de match à objectif avec son relevé. Tout le reste est acquis.

---

## LES DEUX FILMS QUE VOUS VOULEZ — ÉTAT PRÉCIS

### `64e8adfa` — Catalyst, CTF ✅ COMPLET

| | |
|---|---|
| chunks | **en cache**, 34 Mo, 45 fichiers |
| manifest | présent |
| carte cataloguée | oui — Catalyst a ses bornes |
| artefact de rejeu | **construit le 2026-07-31** : 138 traces, 50 144 points, 2 312 tirs, 153 lancers, 237 projectiles |
| événements d'objectif | **68** mesurés (contre 0 sur un Slayer) |
| entités d'objectif | **5 par image-clé** |

**Rien à faire dessus**, sauf éventuellement une capture Cheat Engine si vous voulez l'oracle —
mais **vérifiez d'abord s'il contient surbouclier et camouflage**. S'il ne les a pas, il faudra
un autre match Catalyst pour compléter la table des capacités.

### `696a9d7c` — Nomad / Vagabond, Strongholds ❌ INCOMPLET

| | |
|---|---|
| chunks | **absents du cache** |
| manifest | **absent** |
| carte cataloguée | **non** — Vagabond n'a pas de bornes |
| module du jeu | **non identifié** parmi les 31 copiés |
| artefact de rejeu | impossible en l'état |

**Trois actions, dans cet ordre :**

1. **Obtenir le manifest** — il vient de l'API Halo, donc **réseau + tokens**. C'est le maillon
   fragile : sur le nouveau PC, si l'authentification ne suit pas, ce film est perdu.
2. **Télécharger les chunks** — depuis le CDN Azure pré-signé, **sans authentification** :
   `cd apps/go-api && go run ./cmd/fetch_film_chunks/ -cache ../../data/cache`
3. ~~**Résoudre le module de Vagabond**~~ — **RÉSOLU le 2026-07-31, et gratuitement.**

   Le nom des 199 fichiers `.mvar` porte le lien : `<carte>_<module>.mvar`. Donc
   **`vagabond_fo08_wetland.mvar` → Vagabond = `fo08_wetland`**, et ce module est **présent
   dans le jeu ET déjà sur la clé**. Ses bornes sont donc productibles.

   **Mais le préfixe `fo` dit autre chose, et c'est important** : `fo03_space`, `fo05_desert`,
   `fo08_wetland`, `fo09_academy`, `fo13_frost` sont des **toiles Forge**. On retrouve d'ailleurs
   `banished_narrows_fo05_desert.mvar` et `absolution_fo09_academy.mvar` sur le même modèle.

   **Vagabond est donc une carte Forge**, et cela change ce qu'il faut en attendre :

   | | |
   |---|---|
   | les **bornes** de déquantification | viennent de la **toile** (`fo08_wetland`) — disponibles |
   | le **sol et les structures** | ne sont **pas** dans le BSP de la toile : ils sont dans les **objets placés du `.mvar`** |

   Le fond de carte d'une carte Forge se construit donc autrement que celui de Cliffhanger.
   C'est un chantier à part, mais **la donnée est là** — le `.mvar` est sur la clé et le
   décodeur `mapvar` existe et est testé.

   *(La méthode rigoureuse reste le `level_id` du `.mvar` cherché dans les `.module`, qui a
   validé 21 niveaux sur 21. Le nom de fichier est un indice, pas une preuve : le confirmer
   avant de figer quoi que ce soit.)*

---

## CE QUI MANQUE, PAR ORDRE D'URGENCE

### 1. ⚠ LE FILM `696a9d7c` — À TÉLÉCHARGER, IL N'EST PAS EN CACHE

Match `696a9d7c-009f-4750-8202-f1819947aa4e`, Strongholds sur **Nomad / Vagabond**.

**Vérifié : ni le film ni son manifest ne sont sur le disque.** Or les chunks se téléchargent
depuis le CDN à partir d'un **manifest**, lui-même obtenu de l'API Halo. Sans réseau ni
authentification sur le nouveau PC, ce film devient inatteignable.

```bash
# 1. Obtenir le manifest (passe par l'API Halo : a besoin du reseau et des tokens)
#    -> synchroniser ce match, ou passer par le chemin film du sync
# 2. Puis, les blobs viennent du CDN Azure pre-signe, SANS auth :
cd apps/go-api && go run ./cmd/fetch_film_chunks/ -cache ../../data/cache
# 3. Copier data/cache/film_chunks/696a9d7c/ sur la cle
```

**À faire pendant que la machine actuelle est encore en place.**

### 2. ⚠ LA TABLE DES DÉSÉRIALISEURS — 15 minutes, script prêt

Détail complet dans `SESSION_CAPTURE_AVANT_PC.md`. Rappel de l'enjeu chiffré :

| archétype | composants lisibles |
|---|---|
| biped · objets au sol · projectiles · armes au sol | **complets** |
| véhicules | 32/48 |
| dispositifs | 18/41 |
| **zones** | **0/33** |
| **objectifs** | **0/34** |

### 3. LES CAPTURES DE MATCH

| match | ce qu'il apporte | statut |
|---|---|---|
| **Strongholds sur Nomad/Vagabond** (`696a9d7c`) | l'entité d'objectif en **mode à zones**, plus probablement le **camouflage actif** | demandé, film à télécharger d'abord |
| **Objectif sur Catalyst** | l'entité d'objectif **sur une carte déjà cataloguée** (donc rejeu constructible), plus **surbouclier et camouflage** | recommandé |
| KOTH · Oddball | second et troisième modes à objectif — le témoin qui distingue « propre au CTF » de « vrai pour tous » | **reporté, noté** |

---

## LES OBJETS DE LA CARTE — ARMES, RÂTELIERS, ÉQUIPEMENTS, GRENADES

Sujet ouvert le 2026-07-31. **Bonne nouvelle : le gros est déjà décodable, sans aucune capture.**

### Ce qui est acquis — le placement STATIQUE

Le décodeur `mapvar` lit les variantes de carte (`.mvar`) et rend, **par objet placé** :

| champ | ce que c'est |
|---|---|
| `TypeID` | l'identifiant de type d'objet Forge |
| `Pos` | la position **dans le même repère monde que les joueurs** |
| `Up` / `Forward` | l'orientation |
| `TeamIndex` | l'équipe, quand l'objet en a une |
| `Labels` | des hachages `murmur3` de noms `snake_case` — **le hachage est craqué** |
| `Names` | les chaînes lisibles du fichier |

**Les 199 `.mvar` sont sur la clé**, le décodeur existe et il est testé. Un râtelier d'armes,
un point d'apparition d'arme lourde, un socle d'équipement : ce sont des objets placés, donc
ils sont là.

**Ce qui manque** : la table `TypeID → ce que c'est`. On a `forge_object_types.csv` (45 types
avec leur emprise mesurée) et `objectives.go` (65 labels d'objectif) — ni l'un ni l'autre ne
couvre les armes et les équipements. **C'est du travail de table, pas de décodage.**

### Ce qui est bloqué — l'état VIVANT

Savoir si le lance-roquettes est **encore sur son râtelier** ou déjà ramassé, c'est une autre
question : elle passe par les entités `ti=42` (armes au sol) et `ti=37` (objets au sol).

Leurs désérialiseurs sont **complets** (21/21 et 31/31). Mais **leur position n'est pas
décodable** : trois routes ont été essayées et réfutées sur pièce — sur la voie delta,
5 échantillons contre **1 006 sur un jeu de slots fantôme** de même cardinalité. Le signal est
sous le bruit.

### Ce que ça donne comme chemin

1. **Sans rien capturer** : afficher les emplacements d'armes, d'équipements et de grenades
   **là où la carte les pose**. C'est déjà beaucoup, et c'est immédiat.
2. **Avec la table des désérialiseurs** : rouvrir la question de la position vivante, avec des
   grammaires complètes plutôt que devinées.
3. **Ce qui n'est pas promis** : le suivi « qui a ramassé quoi ». Rien ne le donne aujourd'hui.

---

## COMMENT SAVOIR QUELS POWER-UPS SONT SUR UNE CARTE — SANS LANCER LE JEU

Vous demandiez à confirmer quels power-ups sont présents. **Le `.mvar` doit pouvoir le dire** :
surbouclier et camouflage sont des objets placés, comme les armes.

**La marche à suivre**, dans cet ordre :

1. Décoder `catalyst_catalyst.mvar` et `vagabond_fo08_wetland.mvar` avec `mapvar`, et sortir la
   liste des `TypeID` distincts avec leur nombre d'occurrences et leurs positions.
2. Croiser avec `forge_object_types.csv` (45 types nommés) : ce qui est déjà nommé l'est.
3. Pour les `TypeID` inconnus, deux voies :
   - les **chaînes lisibles** du fichier (`Names`) portent des noms d'instance et de préfab —
     un « overshield » ou un « camo » s'y lit souvent en clair ;
   - **le relevé à l'œil en jeu**, qui reste le juge de paix.

**Si cela marche, on n'a plus besoin de lancer le jeu pour répondre à la question** — ni pour
Catalyst, ni pour aucune des 199 cartes dont on a la variante.

---

## RATISSER LARGE — CE QU'ON POURRAIT AUSSI EMPORTER

54 Go libres : la question n'est pas la place, c'est de ne rien oublier d'irremplaçable.

### Fait le 2026-07-31

| quoi | taille | pourquoi |
|---|---|---|
| **Ghidra 12.1** + `ghidraRun.bat` | 880 Mo | le projet `HI.rep` ne s'ouvre qu'avec la **même version majeure** |
| **`ghidra-mcp`** | 240 Mo | l'outillage qui a permis d'interroger Ghidra depuis l'agent |
| **Les 950 manifests de film** | 119 Mo | ils permettent de **re-télécharger n'importe quel film depuis le CDN sans repasser par l'API** — c'est l'assurance contre une perte d'authentification, et c'est la meilleure du lot |

### La contrainte du nouveau PC, et ce qu'elle change

Le nouveau PC ne peut pas faire tourner **Ghidra, le jeu et Cheat Engine en même temps**.

**Ce n'est pas bloquant**, parce que ces trois-là ne sont nécessaires simultanément que dans une
boucle « capturer puis décompiler dans la foulée » — et cette boucle peut se faire en différé :

```
ICI, maintenant  :  jeu + Cheat Engine     ->  les captures
LA-BAS, plus tard:  Ghidra SEUL            ->  la decompilation
LA-BAS, ensuite  :  ni l'un ni l'autre     ->  le portage Go et l'affichage
```

Ce qui rend cette séparation possible, c'est que la table des désérialiseurs est une **constante
du build** : une fois capturée, elle vaut pour toujours sur cette version du jeu. On n'a plus
besoin du jeu pour s'en servir.

### Reste à considérer

- **Cheat Engine** : à emporter seulement si vous comptez relire des tables sauvées. Aucune
  capture nouvelle ne sera possible sans le jeu de toute façon.
- **Le dossier `deploy/pc/levels/multi`** en plus de `ds` : `mapstruct-build` lit `pc` par
  défaut. On a copié `ds` (qui suffit à `mapquant-build`) — vérifier lequel est requis pour le
  **fond de carte** avant de s'en passer.
- **La configuration MCP** (`.mcp.json` ou équivalent) : quels serveurs, quels chemins.

### Inutile de transporter

Le dépôt Git (il est sur GitHub), les artefacts de rejeu (reconstructibles en une minute), les
structures de carte figées (reconstructibles depuis les modules).

---

## L'IMAGE DE CARTE — LA BELLE EXISTE, ET ELLE N'EST PAS DANS L'APP

> **CORRECTION.** J'avais d'abord répondu que « le pipeline ne sait pas produire d'image de
> carte ». **C'est faux, et l'utilisateur avait raison.** La belle image existe, elle a été
> produite, et elle est validée par une mesure. Ce qui suit la remet à sa place.

### CE QUI A ÉTÉ PRODUIT — et c'est excellent

`C_carte_triangles.png` : **« Sol de Ridgeline reconstruit depuis les TRIANGLES — 10 357
instances / 28,9 M triangles / 0 non résolue »**, en altitude colorée de −6 à +6 m.

**Le témoin qui la valide, sur le même rendu** : **82,0 % des positions joueur à moins de 25 cm
du sol reconstruit, contre 22,8 % attendus par hasard.** Et les deux critères d'acceptation de
l'utilisateur sont tenus : le fer à cheval ressort **en anneau**, la zone sud est reliée par
**deux ponts**.

### DEUX CHAÎNES, À NE PLUS CONFONDRE

| | la chaîne **AABB** | la chaîne **TRIANGLES** |
|---|---|---|
| ce qu'elle lit | la **boîte englobante** de chaque instance | le **maillage réel** : sommets, indices, LOD |
| ce qu'elle rend | des rectangles | la géométrie |
| où elle vit | **`cmd/mapstruct-build`**, en Go, **en production** | **Python jetable**, hors dépôt |
| couverture | ridgeline + sgh_streets | **ridgeline seule** |
| dans l'app | **oui — c'est ce que le rejeu affiche** | **non** |

**Ce que le rejeu montre aujourd'hui, c'est la chaîne AABB.** La belle image vient de l'autre.

### CE QUE RECLAIMER A VRAIMENT ÉTÉ

`Gravemind2401/Reclaimer` est une implémentation **C# tierce**. Elle a servi à **confirmer les
offsets de tag** de la chaîne des triangles — `RuntimeGeoTag`, `ScenarioStructureBspTag`,
`ModuleItem` — et de **juge visuel**. C'est donc bien « le process décrit dans le projet
Reclaimer » qui a produit l'image : nous l'avons porté, en Python, et il marche.

### CE QUI RESTE À FAIRE, ET C'EST ÉCRIT EN UNE LIGNE

> *« Porter la recette en Go pour cuire l'image des 29 autres cartes. Aujourd'hui elle n'existe
> qu'en Python jetable, sur ridgeline seule. »* — `V7.5/cartes/HANDOFF_GEOMETRIE_TRIANGLES.md`

**Trois corrections à ne pas perdre au portage**, toutes dans ce document :

1. les sommets sont en **`u16` brut**, pas `i16 + 32768` — écart aux bornes de **5,8 mm** contre
   84 mm avec la mauvaise lecture. **Tout résultat géométrique antérieur au 2026-07-26 est
   entaché d'une erreur médiane de 8,4 cm** et doit être régénéré ;
2. le chaînage ne passe **pas** par `@0x88`/`@0x8c` — 0,0 % de résolution, réfuté ;
3. le « critère en or » par l'AABB est **tautologique** — à retirer.

### ET POUR VOS DEUX CARTES ?

**Non testé** — ni Catalyst ni Vagabond n'ont été passés dans la chaîne des triangles. Ce qu'on
sait :

- **Catalyst** : la chaîne AABB plafonne à 40-49 %. La chaîne des triangles lit le maillage réel,
  donc elle **pourrait** faire mieux — mais elle passe par le même bloc d'instances, donc si la
  géométrie de Catalyst n'est vraiment pas instanciée, elle butera pareil. **À mesurer, pas à
  supposer.**
- **Vagabond** : carte **Forge**. Son sol vit dans les objets placés du `.mvar`. C'est un
  troisième chemin, encore inexploré.

### CE QUI RESTE VRAI

Le rejeu **fonctionne sans fond de carte** — l'artefact Catalyst le prouve (`structure=0`).
**Ne conditionnez pas les captures à l'image de carte** : capturez, décodez, affichez. Le fond
est un chantier à part, et c'est **le meilleur candidat** dès qu'on veut de belles cartes,
puisque la recette est déjà prouvée.

---

## PROTOCOLE DE CAPTURE — à suivre tel quel

Cette section est écrite pour être exécutée par quelqu'un qui découvre le chantier.

### AVANT de lancer le jeu — 20 minutes, sans rien installer

- [ ] **A1.** Décoder les deux `.mvar` (`catalyst_catalyst`, `vagabond_fo08_wetland`) et sortir
      la liste des `TypeID`, leurs positions et les chaînes lisibles.
- [ ] **A2.** En déduire **quels power-ups chaque carte porte**, et à quel endroit. C'est la
      question de l'utilisateur, et elle se répond peut-être **sans lancer le jeu**.
- [ ] **A3.** Si la réponse est claire, la capture en jeu n'aura plus qu'à la **confirmer** —
      ce qui est un bien meilleur usage du temps de jeu que de la chercher.

### Capture 1 — LA TABLE DES DÉSÉRIALISEURS ⏱ 15 min ⚠ CRITIQUE

- [ ] **B1.** Jeu lancé, **dans un film** en Théâtre (au menu, le registre vaut 0).
- [ ] **B2.** Cheat Engine → `HaloInfinite.exe` → *Show Cheat Table Lua Script* →
      `filmdec_deser_table.lua` → Execute.
- [ ] **B3.** **Vérifier le contrôle imprimé : archétype 35 = 64 composants.** S'il ne tombe
      pas, la structure a bougé — ne pas se fier à la sortie, le signaler.
- [ ] **B4.** Copier `deser_table.tsv` sur la clé.

### Capture 2 — CATALYST, MODE À OBJECTIF ⏱ 20 min

- [ ] **C1.** Un match **CTF, Strongholds ou Total Control sur Catalyst**.
      Le film `64e8adfa` (CTF) est déjà en cache avec son artefact : si vous voulez capturer
      **celui-là précisément**, c'est le plus économique — il est déjà mesuré (68 événements
      d'objectif, 5 entités par image-clé).
- [ ] **C2.** Rejouer en Théâtre avec `filmdec_full_capture.lua` (ou `filmdec_delta_capture.lua`).
- [ ] **C3.** **LE RELEVÉ TERRAIN — sans lui la capture ne vaut rien.** Noter, à la seconde
      près : qui prend le **surbouclier** et quand · qui prend le **camouflage** et quand · les
      instants de **capture de zone** ou de **prise de drapeau** · qui **porte** le drapeau et
      pendant combien de temps.
- [ ] **C4.** Noter l'identifiant du film et le copier avec sa capture.

### Capture 3 — VAGABOND, STRONGHOLDS ⏱ 20 min

- [ ] **D1.** **D'ABORD télécharger le film `696a9d7c`** (cf. plus haut) — tant que le réseau et
      les tokens répondent. Sans lui, la capture n'aura pas de film auquel se rapporter.
- [ ] **D2.** Rejouer en Théâtre avec le même script de capture continue.
- [ ] **D3.** Même relevé terrain qu'en C3, en insistant sur les **zones** : laquelle est prise,
      par quelle équipe, à quelle seconde, et quand elle est reprise.
- [ ] **D4.** Copier capture + film sur la clé.

### Reporté, et noté

**KOTH** et **Oddball** — un second et un troisième mode à objectif. Ils donneraient le témoin
qui distingue « propre au CTF » de « vrai pour tous les objectifs ». Non bloquants.

---

## PIÈGES CONNUS, POUR NE PAS LES REDÉCOUVRIR

1. **Le `.git` de `LevelUp-go-migration` n'est pas un dépôt** : c'est un fichier pointeur vers
   `LevelUp/.git`, lui-même **shallow** (greffe `2eb6811b`, 831 Mo d'objets). Un `cp` du dossier
   ne donne rien d'utilisable — sur le nouveau PC : cloner depuis GitHub, puis recréer les
   worktrees.
2. **Les fichiers non versionnés qu'un clone ne rendra pas** : les 199 `.mvar` (77 Mo,
   `.gitignore:258`) et les quatre `world_dump*` cachés sous `data/cache/film_chunks/`
   (`.gitignore:105`). Ils sont sur la clé — ne pas les écraser en restaurant.
3. **`data/titles/halo_infinite/reference/` n'existe que sur cette branche**, pas sur le tronc.
   Un clone de `main` ne l'aura pas.
4. **Copier une base DuckDB pendant que l'API tourne** peut capturer un état déchiré. Éteindre
   avant.
5. **Une capture ne vaut que par son relevé terrain.** Une capture sans notes « qui a pris quoi,
   à quel moment » est une masse de bits qu'on ne peut confronter à rien — donc infalsifiable,
   donc sans valeur pour ce chantier.

---

## CE QU'ON PERD, CONCRÈTEMENT, SI RIEN N'EST FAIT

| non fait | conséquence |
|---|---|
| table des désérialiseurs | objectifs vivants, zones, moitié des véhicules et dispositifs : **illisibles à jamais sur ce build** |
| film `696a9d7c` | pas de Strongholds sur Vagabond ; le mode à zones reste sans donnée propre |
| capture Catalyst | pas d'oracle pour valider le décodage des objectifs — on décoderait sans pouvoir se contredire |
| Ghidra + exe | **toute résolution statique meurt** ; les 369 désérialiseurs nommés sont à refaire |
| modules de niveau | le lot « toutes les cartes » exige alors le jeu réinstallé |

---

## PROTOCOLE DE REPRISE

1. Faire d'abord ce qui exige le jeu **et** Cheat Engine : la table, puis les captures.
2. Télécharger `696a9d7c` tant que le réseau et les tokens répondent.
3. Copier tout sur la clé, y compris les **relevés terrain écrits**.
4. Le reste — décompilation Ghidra, portage Go, affichage — se fait ensuite, hors ligne, à deux.

---

## JOURNAL D'EXÉCUTION — session J0 du 2026-07-31 (soir)

Statuts détaillés dans `PLAN_MASTER_FILM_KILLFEED_REJEU.md` §J0. Résumé et écarts :

**Une prémisse de ce document est tombée.** J0.4-J0.6 y étaient classés « utilisateur seul ».
Faux : le bridge MCP `cheatengine` permet à l'agent de piloter Cheat Engine (lecture mémoire,
chargement de script Lua, hook AOB, dump). L'agent a exécuté les trois. Le rôle humain se
réduit à lancer le film et à tenir le relevé terrain.

**Fait :**

| item | résultat |
|---|---|
| J0.1 film `696a9d7c` | 31 chunks, décodage NOMINAL, identité prouvée à 99,0 %, clé bit-exacte |
| J0.2 `level_id` | **Vagabond = `fo08_wetland` PROUVÉ** — 1 seule occurrence sur 88 modules, groupe `levl` |
| J0.2 power-ups | **`[!]` bloqué** — voir Découvertes n°2 |
| J0.3 inventaire + branches | clé conforme ; 4 branches à 0 commit non poussé |
| J0.4 table des désérialiseurs | 50 archétypes, 1068 composants, double contrôle passé |
| J0.5 Catalyst | capture faite (`530820e5`, CTF:Arena) **sans relevé terrain** |
| J0.6 Vagabond | capture faite **avec** 4 ancres terrain |

**Le résultat de fond, établi par deux chaînes indépendantes.** `ti=11` (objectifs) est
dispatché 162 fois en CTF et **zéro fois sur 1 205 704 records de Strongholds**, alors que
quatre événements de zone y sont relevés à l'œil. L'absence est propre au MODE, pas à la
capture. Et elle recoupe `archive/V7/RESEARCH_THEATER_RE.md` §M/M-ter (juin) : les événements
`type_hint=10` vivent dans le **chunk type-3**, le score continu dans **TYPE_2**. Ce document
supposait la zone accessible « via la machine d'état zone (replication) » — **notre capture
réfute cette piste**. Conséquence : pour les objectifs, chercher dans les images-clés et le
footer d'events, jamais dans le flux delta. Les deux sont déjà en cache, donc exploitables
hors ligne, sans jeu ni Cheat Engine.

### DÉCOUVERTES — consignées, NON traitées (plan-execution règle 7)

1. **`fetch_film_chunks` ne prend que les chunks type-2.** L'en-tête (type 1) et le footer
   d'events (type 3) sont ignorés — or sans l'en-tête le décodeur ne peut pas amorcer son
   World, et sans le footer l'équipe d'une capture de drapeau est perdue (c'est exactement ce
   que `RESEARCH_THEATER_RE.md` §M-ter demande de corriger en production). Pris à la main ce
   soir via `cmd/tmp_fetchchunk0` pour les deux films. **Audit du cache : 949 footers présents
   sur 951** — le trou est donc quasi résorbé, mais l'outil, lui, n'est toujours pas corrigé.
   À traiter avec le chantier F (rejeu en prod) ou au branchement du collecteur.
2. **La palette Forge n'est pas sur la clé.** `any|ds/globals/forge/forge_objects-rtx-new.module`
   (417 Mo + 2,0 Go) restent sur `D:`. Sans eux, `type_id → nom d'objet` est indécidable : c'est
   ce qui bloque le volet power-ups de J0.2. Seuls les 31 modules de niveau ont été copiés.
   Espace libre sur la clé : 42 Go. **Décision utilisateur.**
3. **Deux films ont expiré côté serveur** : `33b9fbe9` et `f8c067d7` (BTB:Stockpile) rendent
   `HTTP 404 — No film found for that match`. Ce n'est pas une perte liée au changement de PC,
   c'est l'expiration Theater que le plan maître nomme (§5.2-2). Ils sont définitivement
   inatteignables ; leurs manifests en cache ne servent plus à rien.
4. **Le binaire est dépouillé** : aucun nom d'archétype ne se lit en mémoire (ni chaîne inline,
   ni pointeur vers de l'ASCII, ni symbole). Le nommage de `ti=11`/`ti=23` passera par Ghidra,
   à partir des vtables relevées dans `archetype_vtables.tsv`.
5. **L'index de record est une horloge approchée** du temps de film (~2 200-2 470 records/s,
   7 % d'écart sur 4 ancres). Utile pour situer, insuffisant pour dater : la datation fine
   passe par l'alignement de signature.

### Outil créé ce soir

`cmd/tmp_sigalign` (jetable, consigné) — **prouve de quel film provient une capture** par
recherche des signatures de 16 octets dans les chunks, avec témoin négatif obligatoire.
Mesuré : 99,0 % et 99,8 % sur les films sources, 0,0 à 1,0 % sur les témoins.
