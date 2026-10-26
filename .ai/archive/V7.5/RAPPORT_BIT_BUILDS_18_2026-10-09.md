# Le bit de trop des films <= HI_1_8_0 : quelle donnée déclarée le décide

Date : 2026-10-09. Worktree `LevelUp-wt-ecart108`, branche `feat/ri-ecart-108`. Recherche seulement :
aucun code de production n'est modifié. Ghidra (HaloInfinite.exe, HTTP `127.0.0.1:8089`) a été lu
uniquement en `GET`. Les sorties sont sous `$B = scratchpad/ri/ecart108/b2/`.

Ce rapport fait suite à `.ai/RAPPORT_I53_I57_ECART_33_2026-10-09.md` (mission B), qui situait le bit
« entre la porte d'i48 et le R(5) d'i53 » sans en connaître la position.

## 0. Verdict

1. **La position du bit est mesurée, à trois bits près.** Le flux <= 1.8 n'a pas le **drapeau final**
   du bloc de propriété malléable d'i53 (`FUN_1407efc5c`, lecture à `1407efeba`, octet `+0x20` de
   l'état). L'autre lecture possible, indiscernable sur toutes les données disponibles, est que **le
   R(5) de n (`FUN_1424e2f20`) y fait 4 bits**. Tout le reste de la fenêtre est aligné : i48 R(6),
   i49, i50, i51, i52, la porte R(8), les 11 portes R(12) et les 7 premiers drapeaux d'i53.
2. **Lire le flux sans ce bit ferme les records.** Sur `60ae07c4`, les écarts +33 (163 records), +18
   (46) et +28 (9) passent tous à 0. Les fermetures ti=35 passent de 1 à 223. Sur les 13 films
   HI_1_8_0 du cache, on gagne 4 051 fermetures sans en perdre une seule. Sur le corpus de 28 films,
   ti=35 passe de 5 401 à 5 689 fermés.
3. **La donnée déclarée qui sépare les deux familles existe, et elle est unique.** Entre un film 1.8
   et un film 1.9, le registre complet de `chunk_00` est identique à l'octet (0xC7108 octets : noms,
   niveaux, bourrage), **sauf l'octet 0**. C'est la **version majeure du film** (`chunk_00+0`), qui
   vaut `0x25` pour HI_1_8_0 et `0x26` pour HI_1_9_0. Ailleurs dans `chunk_00`, deux entrées de la
   table par type changent aussi : le genre 35 passe de 3 à 4, le genre 36 de 2 à 3. Sur le parc
   (2 279 films), la frontière tombe exactement sur la version majeure :
   - la variante ne gagne que sur les films de majeure <= `0x25` ;
   - elle casse tous les films de majeure >= `0x26`.
4. **Aucune donnée déclarée qui diffère entre 1.8 et 1.9 n'est consultée par l'exécutable courant
   sur le chemin de lecture de la fenêtre.** Le recensement est complet (§1).
   - Les seules entrées des lecteurs qui ne viennent pas du flux sont :
     - le niveau du composant (`param_4`, lu dans le registre, **identique** en 1.8 et 1.9) ;
     - `DAT_145121140` pour i49 (tiré du type de moteur de la variante, **2 sur tout le parc**).
   - Le bloc d'i53 ne porte qu'une condition, `CMP EBX,0x2` à `1407efea7`, et elle porte sur le
     drapeau `+0x1f`, pas sur le drapeau final.
   - La version majeure n'est consultée qu'à deux endroits :
     - `FUN_1428e219c` (`CMP dword ptr [RDX],0x29` à `1428e21a6`) : le jeu ne démarre la lecture du
       film que si sa majeure vaut 0x29, sinon il appelle `FUN_142988e98` ;
     - `FUN_14299bd78` (`14299bd83`), sur le même test.
   - Les genres 35 et 36 ne sont consultés que par des messages de la vue A (§1.4).
5. **Conséquence, à soumettre à l'utilisateur.** Le principe posé est que le film déclare de quoi le
   jeu actuel le relit. L'exécutable lu ne le réalise pas pour ce bit :
   - il ne relit pas ces films du tout, puisque sa garde de version majeure s'arrête avant toute
     lecture pour une majeure différente de 0x29. C'était déjà relevé dans le dépôt
     (`vue_a_versions.go`, garde de version majeure, 2026-10-06) ;
   - aucune de ses branches ne consulte la donnée qui sépare 1.8 de 1.9.

   Je n'en conclus pas que « le jeu lirait ce flux de travers » : **l'exécutable courant ne lit pas
   ce flux**, donc il ne peut pas servir d'oracle de sa grammaire. Le fait mesuré est que le film
   1.8 déclare une version majeure différente, et que son flux d'i53 a un bit de moins.
   - Une règle de grammaire qui se brancherait sur cette version majeure serait fidèle à une donnée
     que le film déclare et que le jeu lit (§3).
   - En revanche, elle ne serait pas lue dans une branche du jeu. C'est une **décision de
     l'utilisateur** ; elle n'est pas écrite.

## 1. Lu dans le jeu : les conditions de lecture de la fenêtre

### 1.1 Les lecteurs, depuis les descripteurs

Les descripteurs ont été lus avec `read_memory` sur `0x143d0c000` (8 Ko), sortie `$B/qw2.txt`. Un
descripteur fait 0x50 octets : `+0x00` donne le nom (`LEA RAX,[chaîne]; RET`), `+0x10` l'écrivain,
`+0x28` le lecteur. Les noms ont été décodés depuis les octets de chaque fonction de nom.

| Composant | Descripteur | Lecteur `+0x28` | Ce qui est lu | Entrées hors flux |
|---|---|---|---|---|
| i48 `biped-desired-ability-set-component` | `143d0cad0` | `1410f8fcc` : thunk `MOV RCX,[R8+0x10]; ADD RCX,0xa34; JMP 1406d0ff0` | `FUN_1406d0f20` R(3) ; `FUN_1406d1024` R(1), puis R(6) si le bit vaut 0 | aucune (R9, le niveau, est ignoré) |
| i49 `biped-control-context-component` | `143d0cb20` | `14107166c` | R(2 ou 4), R(1) | `CMP byte ptr [0x145121140],0x1` à `14107167b` |
| i50 `biped-map-editor-flag-component` | `143d0cc10` | `142f02854` | R(8) à plat | aucune |
| i51 `biped-emp-timer-component` | `143d0cbc0` | `142f02830` → `1432065d8` → `FUN_1406d84b4` | largeur 8 (`MOV dword ptr [RSP+0x20],0x8` à `1432065f1`) ; les globales `143cd84b0`/`143cd8910` sont des flottants de déquantification | aucune |
| i52 `biped-low-frequency-data-component` | `143d0ca30` | `140fc91c8` → `140fc91e0` | 3 × R(1), porte R(1), puis si elle vaut 1 : `FUN_14080d6f0` R(32) et `FUN_14080dec4("voice-designator")` R(32). La résolution de poignée (`FUN_140821f44`, `FUN_14080d61c`, sur `DAT_144eae7b8`/`DAT_144b404f0`) ne reçoit pas le lecteur de bits | aucune |
| i53 `biped-malleable-property-component` | `143d0cd60` | `140ff6764` → `FUN_1407efc5c`, `FUN_1424e2f20` | voir §1.2 | niveau (`param_3`/`R8D`, recopié dans `EBX` à `1407efc80`) |

**Primitives** (désassemblage, sortie `$B/d_*.asm`) : `1406cf008`, `1406d6c7c`, `1407f08f8`,
`1424e2f20`, `140e82b84`, `1406d0f20`, `1406d1024`, `142f02854`, `14080d6f0`, `14080dec4`. Aucune ne
lit de globale ni n'appelle autre chose qu'une primitive de lecture de bits. Aucune n'appelle
`FUN_141102ed0` (version par type), `FUN_1428e1c0c` (version de format) ou `FUN_14076f91c` (garde
de pleine précision).

### 1.2 Le bloc d'i53 (`FUN_1407efc5c`, 174 lignes, `$B/d_1407efc5c.asm`)

1. `CALL 1407f08bc` (R(1) puis R(8)).
2. 11 × `CALL 140e82b84` (R(1) puis R(12)).
3. 7 × `CALL 1406cf008`, rangés en `+0x18` à `+0x1e`.
4. `CMP EBX,0x2 ; JC` à `1407efea7`, puis un drapeau en `+0x1f` (à `1407efeaf`).
5. Un drapeau final en `+0x20`, sans condition (à `1407efeba`).

Le niveau est la seule condition. `FUN_1407efc5c` n'a qu'un appelant, `FUN_140ff6764` à `140ff6784`
(`get_xrefs_to`). Le bloc n'est donc partagé avec aucun message qui lui passerait une version de
genre.

### 1.3 La boucle et l'en-tête d'état complet

- **`FUN_142e2c690`** (relu, `scratchpad/ri/ecart108/i53_g_142e2c690.c`) : pour chaque entrée
  nommée, elle prend le niveau à `entrée+0x100` et appelle `vtable+0x28`. Si le contrôle de
  corruption est levé, elle lit un R(1) puis un R(32). Elle n'a aucune autre clé.
- **`FUN_142e2bfd0`** consulte seulement :
  - `FUN_1428e1c0c` (version de format, `chunk_00+4`) : 24 en 1.8 comme en 1.9 ;
  - `FUN_14076cea8` (contrôle de corruption) : faux sur tout le parc.
- **Lecteur de `chunk_00`, `FUN_14299ab50`.**
  - Les largeurs du registre et de la table par type viennent de deux tables indexées par la version
    de format : `FUN_141cfff30` (défaut `0x659000`) et `FUN_141cffe20` (défaut `0xf60`).
  - `FUN_14299bcb0` complète ensuite les genres que le film ne déclare pas avec les versions natives.
  - C'est la compatibilité de lecture que porte le jeu. Elle ne touche pas au registre, et elle ne
    distingue pas 1.8 de 1.9 (même format 24).

### 1.4 Les données qui diffèrent entre 1.8 et 1.9, et leurs consommateurs

- **Version majeure (`chunk_00+0`).** Recherche `search_instructions CMP … 0x29` sur tout le
  programme. Deux comparaisons à la base du film :
  - `FUN_1428e219c` à `1428e21a6`, appelée par `FUN_140ba23e4` à `1423b9bd7`. Sous `*film == 0x29`,
    elle reporte le bit de contrôle, puis boucle `FUN_1428e27c0` (la lecture). Sinon elle appelle
    `FUN_142988e98`, qui ne pose qu'un état (`FUN_141cfebac` sur `+0x1488`, code 0). **Le film n'est
    pas lu.**
  - `FUN_14299bd78` à `14299bd83`, appelée par `FUN_142999158` (chemin d'écriture ou de copie).
  - Aucun accesseur du film ne rend la majeure. Les accesseurs `FUN_1428e1*` qui lisent la structure
    du film ont été balayés (`$B/acc/`) : ils lisent `+0xcb208` (table par type), `+0xcd670`, le
    registre par nom (`FUN_1428e1b50`, `FUN_1428e1dac`, appelés par la boucle delta seulement), etc.
- **Genres 35 et 36 de la table par type.**
  - Les seuls appelants de `FUN_141102ed0` passent des genres de message de vue A. C'est le cas de
    `0x23` dans `FUN_142f17500`, un lecteur de message à `variant_name`.
  - `get_xrefs_to(141102ed0)` donne 22 sites. Aucun n'est dans un lecteur de composant de la
    fenêtre.
  - `FUN_1428e1c64` n'a qu'un appelant, `FUN_141102ed0`.
- **Type de moteur de la variante** (`DAT_145121140 = FUN_14051a4b8(type)`) : `type=2` sur les 2 279
  films lus (§2.1), donc aucune différence.

## 2. Mesures

Environnement : `GOCACHE=go-build-ecart108`, CGO, tests en avant-plan.

### 2.1 Les données déclarées de `chunk_00`, sur le parc

`TestBit18Parc` (`BIT18_RACINE=data/cache/film_chunks`) → `$B/parc.tsv` (2 279 films).

| Build | Films | Majeure | Format | Genres | Blocs | Empreinte du registre | Type de moteur |
|---|---|---|---|---|---|---|---|
| (sans section) | 3 + 2 | 0x1f / 0x21 | 20 | - | 49 | … | - |
| HI_1_4_1 | 1 | 0x21 | 21 | 116 | 49 | `0x40531a0d86ce90ce` | 2 |
| HI_1_5_1 | 1 | 0x22 | 23 | 119 | 49 | `0x8d6dec5f4fc182c1` | 2 |
| **HI_1_8_0** | 13 | **0x25** | 24 | 121 | 49 | **`0x33c7e724716d8cc5`** | 2 |
| **HI_1_9_0** | 3 | **0x26** | 24 | 121 | 49 | **`0x33c7e724716d8cc5`** | 2 |
| HI_1_10_0 | 34 | 0x27 | 24 | 121 | 49 | `0x9b6397b3ad58e258` | 2 |
| HI_1_11_0 | 57 | 0x28 | 25 | 122 | 49 | `0x8879e2b6746ba047` | 2 |
| HI_1_12_0 | 147 | 0x28 | 27 | 123 | 50 | `0x36ca8c3d2a2f9b88` | 2 |
| HI_1_13_0 | 2 017+ | 0x29 | 27 | 123 | 50 | `0x36ca8c3d2a2f9b88` | 2 |

- Le contrôle de corruption vaut `false` partout.
- La table par type est uniforme à l'intérieur de chaque build.
- Entre HI_1_8_0 et HI_1_9_0, elle ne change qu'en deux genres : **35 (3 → 4)** et **36 (2 → 3)**.

### 2.2 Le registre complet, octet à octet

`TestBit18Registre` (`BIT18_REF=11de8353`, 1.9) → `$B/diff_<film>.txt`.

- **`60ae07c4` (1.8) contre `11de8353` (1.9)** : sur les 0xC7108 octets du registre, **1 octet
  diffère, l'octet 0** (`26` contre `25`). Aucune entrée ne change de nom ni de niveau, et aucun
  octet de bourrage ne diffère.
  - Après le registre, seuls diffèrent les genres 35 et 36, puis les champs propres à
    l'enregistrement :
    - chaîne de version, chaîne de build ;
    - identifiant de build (`+0x60`), changelist (`+0x64`) ;
    - horodatage (`+0x268`) ;
    - deux blocs de 16 octets (`+0x3278`), qui varient d'un film à l'autre.
- **`084a804d` (1.10)** : quatre entrées changent (`object-forward-and-up[-dynamic-precision]` dans
  les blocs 38, 39, 40 et 43, niveau 1 → 2). Ce n'est pas le bipède.
- **`a521164d` (1.4.1)** : le bloc 6 (statborg) est réordonné. i57 et i59 du bipède sont au niveau 1
  au lieu de 2, et deux entrées sont absentes. 116 genres.

### 2.3 Les bits bruts de la fenêtre

La sonde 2.7.d1 a été relancée sur `60ae07c4`, `11de8353`, `084a804d` et `111fa685`, avec
`BIT18_FENETRE=1`. Pour chaque record bipède, elle écrit une ligne W : les bits du flux depuis le
début d'i48 jusqu'à 64 bits après le début d'i54. Sorties : `$B/sonde/` ; découpage `fen.awk` →
`fen0.tsv` ; écarts de fin `w_ecart.tsv`.

- **Références** (1 342 records fermés des trois films) :
  - i49 = `1/1`, i50 = 0, i51 = 0, i52 = `000/1` (porte ouverte, puis les deux R(32)) ;
  - 12 portes d'i53 fermées dans 1 339 cas, drapeaux `0111111|0` ;
  - **n = 5 et v = 0x1f dans 1 342 cas sur 1 342.**
- **`60ae07c4`, écart +33** (162 records) : préfixe identique jusqu'aux 7 drapeaux `0111111|0`, puis
  **n = 11, v = 0x780**. Même chose pour +18 et −75.
- Le flux suit les drapeaux comme ceci (Z = suite de zéros, O = suite de uns) :
  - références : `Z3 O1 Z1 O5…` (drapeau final 0, n = `00101`, v = `11111`) ;
  - 1.8 : `Z2 O1 Z1 O5…`.
  - **Il manque un zéro dans la suite de trois qui forme le drapeau final et les deux bits de tête
    de n.**

**Balayage de la position** (`balayage2.awk`). On insère un bit factice à chaque position k depuis
le début d'i49, puis on relit avec la grammaire du jeu. On exige la forme modale complète des
références (i49 à i52, portes nulles, drapeaux `0111111|0`, n = 5, v = 0x1f).

- **Seuls k = 106 à 108 passent**, dans toutes les populations : +33 (141/162), +18 (36/46), −75,
  −183, −291, etc.
- k = 106 est exactement 87 (début d'i53) + 12 (portes) + 7 (drapeaux).
- Une insertion plus tôt (i49 à i52, ou portes d'i53) décale les drapeaux en `1111110` ou `0011111`.
  Ces formes ne se voient jamais.
- Les records qui échouent portent des valeurs d'i49 ou d'i53 différentes de la modale. Ce n'est pas
  une autre position.
- **Population C (+28)** : sans drapeau final, les 9 records lisent n = 5 et v = 0x1e, avec trois
  portes ouvertes. C'est la même forme que les 3 références à portes ouvertes (n = 5).

**Drapeau final ou largeur de n.** Les deux lectures donnent la même chose quand le drapeau vaut 0 et
que n < 16 : c'est le cas de 100 % des références. **Les données disponibles ne les départagent
pas.** Un indice structurel penche vers le drapeau final, sans valeur de preuve :
- l'écrivain courant (`FUN_1431f6444`) émet `+0x18` à `+0x20`, et le champ `+0x1f` a été ajouté
  sous niveau 2 ;
- `FUN_1424e2f20` est un lecteur R(5) à plat, partagé.

### 2.4 La lecture sans ce bit, rejouée de bout en bout

`bit18Variante` (ligne V). On rejoue la marche de production (`WalkKeyframeFullState`) sur une copie
du payload où un bit nul est inséré à la place du drapeau final d'i53. L'écart est mesuré contre le
début du record suivant + 1. Le contrôle (même marche, sans insertion) reproduit l'écart d'origine
sur **100 % des records** de tous les films mesurés.

**Corpus de 28 films** (`$B/var28/par_film.txt`, 50 s) :

| Films | Majeure | Fermés avant | Avec la variante | Gagnés | Perdus | +33/+18/+28 ramenés à 0 |
|---|---|---|---|---|---|---|
| `60ae07c4` HI_1_8_0 | 0x25 | 1 | **223** | 222 | 0 | **218 / 218** |
| `a349fea8` (f20) | 0x21 | 9 | 59 | 56 | 6 | 50 / 57 |
| `a521164d` HI_1_4_1 | 0x21 | 9 | 29 | 26 | 6 | 24 / 26 |
| `50247b26` (f20) | 0x1f | 6 | 2 | 0 | 4 | 0 / 3 |
| 24 films de majeure >= 0x26 | 0x26-0x29 | 5 376 | quasi nul | 0 | presque tout | - |

- Sur `60ae07c4`, les écarts −75, −183, −291… deviennent exactement −108, −216, −324… Ce sont les
  entrées de slot libéré du rapport R-VEH-4, et la règle Rslot les fermerait en plus.
- **Avec la variante limitée aux films de majeure <= 0x25** (simulation de mesure) :
  - ti=35 passe de **5 401 à 5 689** fermés sur 10 710 ;
  - les films de majeure >= 0x26 restent identiques (variante non appliquée) ;
  - `60ae07c4` remplit les trois critères de la mission.
- Les 16 pertes sur les formats 20 et 21 ne sont pas instruites : ces formats portent déjà d'autres
  défauts après i22 (D-4 à D-6).

**Hors corpus, sur le parc** (`$B/varparc/par_film.txt`). Ces films n'ont pas de faits, la carte a
été posée à « Live Fire - Ranked » **sans vérification**. Sous la portée, la position du bipède est
brute.

- Les **12 autres films HI_1_8_0** gagnent tous :
  - 3 829 fermetures au total, de 99 à 562 par film, **0 perte** ;
  - +33/+18/+28 ramenés à 0 : 3 604 sur 3 605.
- Les **2 autres films HI_1_9_0** perdent tout (392/395 et 284/284).
- **HI_1_5_1** (0x22) : 10 gagnés, 2 perdus.

**Non mesuré** : le témoin de hasard (en-tête à 109 bits) sous la variante ; les fichiers de
référence (goldens) et `keyframe_closure.golden`, puisqu'aucun code de production n'a changé.

## 3. Conclusion pour le superviseur

- **La donnée déclarée qui décide du bit, au sens strict « lue par le jeu courant pour choisir la
  grammaire » : aucune.** C'est établi par le recensement du §1.
  - Les seules entrées hors flux des lecteurs de la fenêtre sont le niveau du registre (identique)
    et le type de moteur (identique).
  - La version majeure, seule donnée du registre qui diffère, n'est consultée que par la garde
    `1428e21a6`, et cette garde **refuse de lire** les films de majeure ≠ 0x29.
- **La donnée déclarée qui sépare les familles, au sens « le film la porte, et elle coïncide
  exactement avec la grammaire mesurée » : la version majeure `chunk_00+0`.**
  - Sur le parc entier, <= `0x25` et >= `0x26` correspondent aux deux grammaires.
  - Les genres 35 et 36 de la table par type changent au même endroit, mais aucun lecteur de la
    fenêtre ne les consulte.
- **Règle de grammaire candidate, non écrite, à soumettre à l'utilisateur :**
  - « au niveau 1, le bloc de propriété malléable du bipède (`FUN_1407efc5c`) ne porte pas son
    drapeau final (`+0x20`, lu à `1407efeba`) quand la version majeure déclarée en `chunk_00+0` est
    <= `0x25` » ;
  - variante équivalente sur les données : « n se lit sur 4 bits ».
  - Elle se branche sur une donnée que le film déclare, et non sur la chaîne de build.
  - Mais le seuil `0x25` est mesuré : il n'est lu dans aucune branche de l'exécutable. Le principe
    « une règle par build posée à la main est exclue » s'y applique peut-être ; c'est à
    l'utilisateur d'en juger.
- **Ce qui reste à regarder pour la lire au lieu de la mesurer :**
  1. un exécutable HI_1_8_0 (ou HI_1_9_0), pour relire `FUN_1407efc5c` et `FUN_1424e2f20` à cette
     version. C'est le seul moyen de trancher entre drapeau final et R(4), et de lire la règle au
     lieu de la mesurer ;
  2. un film 1.8 dont un record aurait n >= 16 ou un drapeau final à 1 dans sa forme 1.9. Aucun dans
     les 4 films dont les bits ont été extraits ;
  3. l'événement posé par `FUN_142988e98` (code 0 sur `+0x1488`) et son traitement : est-ce un refus
     définitif, ou une conversion du film par un autre chemin ? Non lu. C'est la seule piste qui
     pourrait faire de l'exécutable courant un lecteur des films de majeure ≠ 0x29.

## 4. Découvertes non traitées (règle 5)

- **i49 dans le Go.** Le Go décide R(2) ou R(4) par `Mouvement.FullPrecision` du profil
  (`components_biped_ability.go`). Le jeu, lui, teste `DAT_145121140`, que `identite.go` décrit
  comme `FUN_14051a4b8(TypeDeMoteur)`, et non `fullPrecisionGate`. Sans effet mesuré : type 2 sur
  tout le parc.
- **Le bloc de 16 octets à `+0x3278` de la section 2** varie d'un film à l'autre et n'est pas
  nommé.
- **Les 16 fermetures perdues sous la variante** sur les formats 20 et 21 (`a349fea8`, `a521164d`,
  `50247b26`) ne sont pas instruites.

## 5. Instruments et commandes

Instruments `research` :

- `apps/go-api/internal/games/halo_infinite/film/internal/grammar/bit18_registre_research_test.go` :
  - `TestBit18Parc` et `TestBit18Registre` ;
  - `bit18Fenetre` (ligne W) et `bit18Variante` (ligne V).
- Deux crochets d'une ligne dans `ri27d1_instrument_research_test.go` (`dumpRecord`) et
  `ri27d0_images_cles_research_test.go` (`ImageCle`). Ils sont inactifs sans `BIT18_FENETRE` ou
  `BIT18_VARIANTE`.

Commandes :

```
cd apps/go-api ; R=C:/…/LevelUp-wt-ecart108/data/cache/film_chunks
BIT18_RACINE=$R BIT18_OUT=$B go test -tags=research -count=1 -run '^TestBit18Parc$' ./internal/games/halo_infinite/film/internal/grammar/
BIT18_REF=11de8353 BIT18_FILMS=60ae07c4,084a804d,a521164d,a349fea8,50247b26 BIT18_RACINE=$R BIT18_OUT=$B \
  go test -tags=research -count=1 -run '^TestBit18Registre$' ./internal/games/halo_infinite/film/internal/grammar/
BIT18_FENETRE=1 RI27D1_RECORDS=60ae07c4,11de8353,084a804d,111fa685 RI27C_FILMS=<idem> RI27C_RACINE=$R RI27C_OUT=$B/sonde \
  go test -tags=research -count=1 -run '^TestRI27d0ImagesCles$' ./internal/games/halo_infinite/film/internal/grammar/
BIT18_VARIANTE=1 RI27C_FILMS=<28 films de d1/run.sh> RI27C_CARTES=<idem> RI27C_RACINE=$R RI27C_OUT=$B/var28 \
  go test -tags=research -count=1 -run '^TestRI27d0ImagesCles$' ./internal/games/halo_infinite/film/internal/grammar/
```

- Pour `$B/varparc` : mêmes variables, films HI_1_8_0, HI_1_9_0 et HI_1_5_1 du cache hors corpus,
  `RI27C_CARTES=<film>=Live Fire - Ranked`.
- Découpage et balayage : `$B/sonde/fen.awk`, `balayage2.awk`, `apres_drapeaux.awk`.
- Ghidra : décompilations `$B/g_*.c`, désassemblages `$B/d_*.asm`, descripteurs `$B/qw2.txt`.
