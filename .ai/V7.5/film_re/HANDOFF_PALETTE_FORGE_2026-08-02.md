# HANDOFF — palette Forge, formes de zone, nommage : où en est le chantier

> Écrit le 2026-08-02 en fin de session (contexte saturé). Branche `feat/re-mode-score`,
> worktree `.claude/worktrees/re-mode-score`, **12 commits poussés** (le dernier étant celui qui porte ce document), arbre propre.
>
> Ce document dit ce qui est ACQUIS, ce qui est RÉFUTÉ (pour ne pas le rejouer), et
> l'unique question ouverte avec son plan d'attaque.
>
> État de l'art complet : `.ai/ETAT_DE_L_ART_FORGE_PALETTE_ZONES.md`.
> Journal : `.ai/thought_log.md`, quatre entrées des 2026-08-01 et 02.

---

## CE QUI EST ACQUIS ET VÉRIFIÉ

| acquis | témoin |
|---|---|
| **La forme d'une zone est dans le `.mvar`** — sac à `#8 → #0[0] → #0[0]` : famille (2 cylindre, 3 boîte) + 4 entiers en virgule fixe 16.16 | pas établi : 393216 = 6 × 65536 exactement ; 39,2 % des 62 299 valeurs brutes sont des multiples exacts |
| **16 434 formes sur 197 cartes.** 100 % des objectifs SURFACIQUES en portent, 0 % des PONCTUELS | Extraction 434/434, Ravitaillement 186/186, Bastion 430/431 ; drapeau 0/669, socles 0/855 |
| **La forme est ORIENTÉE** (`Forward` + `Up`) | boîte alignée = +31 % de faux positifs sur une zone tournée de 20°, identique au millième sur une zone droite, témoin négatif 0,000 % |
| **`s5`/`s6` sont des TAILLES PLEINES** (boîte) ; `s5` est un RAYON (cylindre) ; `s7`/`s8` sont des distances au centre | **deux mesures indépendantes** : 11 coïncidences exactes cylindre/boîte contre 1 ; et l'oracle d'exécution ci-dessous |
| **L'oracle d'exécution** : aux 4 instants du relevé terrain de Vagabond, rapport signal/témoin **6,0** en lecture pleine contre **1,4** en lecture demi | sous la lecture demi, le témoin négatif attrape autant voire PLUS que les vraies zones (3 contre 2, 4 contre 3) |
| **La palette est résolue à 99,0 %** (2 758/2 785 `type_id` → groupe de tag) | contrôle positif **45/45 identiques** sur `forge_object_types.csv` |
| **Le nom d'un objet est un murmur3 DANS le tag** : `food` → bloc « Object Representations » → mot 0 = `Representation Name` | validé de l'extérieur : le dictionnaire de `Z-15/Halo-Infinite-Tag-Editor` donne les MÊMES valeurs en gros-boutiste (`gravity_hammer` 3C2CDAAC ↔ ACDA2C3C) |
| **67 entrées de palette nommées** | `skull_weapon` = 21 cartes / **21 instances** (un crâne d'Oddball par carte) ; `kill_volume`, `cubemap_volume`, `unsc_turret`, les véhicules, les armes |
| **L'emprise de la palette est en UNITÉS MONDE** (× 3,048), pas en mètres | Pelican 32,7 m, Warthog 6,8 × 3,1 × 2,5 m |
| **Les types d'emplacement, nommés par le jeu** | `PowerUpPad` · `PowerWeaponPad` · **`WeaponRack`** · `WeaponTrunk` · `EquipmentPad` · `GrenadePad` · `OrdnancePod` · `VehiclePad` |

**Corollaire important** : le groupe de tag `eqip` de la palette Forge, ce sont **les
grenades** (frag, plasma, spike — les 3 seules entrées `eqip` sur 4 235). Tout raisonnement
du type « aucun `eqip` placé donc aucun power-up » est un non-sequitur.

---

## LA STRUCTURE, TELLE QU'ELLE EST MAINTENANT COMPRISE

Le plugin `food.xml` (dépôt `Gamergotten/Infinite-runtime-tagviewer`, copié sous
`.ai/V7.5/dumps/forge_zones/reference_food_tag_definition.xml`) nomme le bloc que cette
session avait mesuré à l'aveugle :

```
_40  Object Representations            (le « bloc 1 » de 92 octets)
       _2   Representation Name        <- mot 0 : murmur3 du nom
       _2   Crate Variant              <- mot 1 : 0 chez 4 209 entrees sur 4 213
       _41  Configuration              <- reference vide dans nos cas
       _41  Object Definition (Crate)  <- LA reference weap/eqip/bloc
       _41  Menu Item Definitions      <- reference vide dans nos cas
```

**Et le lien avec la carte est établi** (mesure du 2026-08-02, balayage de l'arbre Bond
entier des 199 variantes) :

| valeur | où elle apparaît dans le `.mvar` | lecture |
|---|---|---|
| `Representation Name` | `/#3[]/#8/#24[]/#1/#0` | présent sur **99,95 %** des objets ; les comptes correspondent EXACTEMENT aux instances par type |
| `Crate Variant` | `/#3[]/#8/#1[]/#0[]/#0` | dans le même sac que le **délai de réapparition** ; c'est une surcharge PAR OBJET |

Autrement dit : **`mapvar` lit déjà ces deux champs et les jette**. Les brancher est une
lecture, pas une découverte.

---

## LA QUESTION OUVERTE, ET ELLE EST UNIQUE

**Nommer les quatre entrées « emplacement »** — les seules de la palette dont le
`Crate Variant` est non nul, toutes bâties sur le même modèle `-370671751` (0,40 × 0,40 ×
0,80 m).

| entrée | Representation Name | Crate Variant | usage |
|---|---|---|---|
| `1486653438` | `-1412311642` | `-88833402` | 27 cartes / 98 inst., jusqu'à 9 par carte |
| `-1062552774` | `-245254093` | `-224029073` | 21 cartes / 46 inst., jusqu'à 4 |
| `1882451900` | `-1351408675` | `-1319554708` | 20 cartes / 117 inst., jusqu'à 16 |
| `801517767` | `-219174009` | `-257275188` | 11 cartes / 24 inst., jusqu'à 4 |

> **MISE À JOUR 2026-08-02 (reprise)** — les quatre `Crate Variant` de ce tableau sont
> désormais **nommés** : `-88833402` = `banished_kinetic`, `-224029073` = `banished_shock`,
> `-1319554708` = `banished_plasma`, `-257275188` = `banished_hardlight`. Et la déduction
> du paragraphe ci-dessous est **affaiblie** : les deux emplacements de Vagabond portent la
> **même** variante par objet, donc seule la paire de `Representation Name` les sépare —
> l'attribution lance-roquettes/camouflage repose sur la seule altitude relevée. Voir
> « PISTE 1 — JOUÉE » plus bas.

**DEUX D'ENTRE ELLES SONT ÉPINGLÉES PAR LE TERRAIN.** Sur Vagabond il y a exactement deux
emplacements, diamétralement opposés ; l'utilisateur atteste qu'ils portent un
**lance-roquettes** et un **camouflage**, et que **le lance-roquettes est plus bas** (un
étage). Mesure : `1486653438` est à z = 51,4 et `-1062552774` à z = 54,8 — 3,4 m d'écart.
Les quatre objets de Vagabond partagent le **même** `Crate Variant` par objet
(`-88833402`) et ne diffèrent que par leur `Representation Name`. Donc :

- **`-1412311642` = la représentation de l'emplacement du LANCE-ROQUETTES** (bas) ;
- **`-245254093` = celle du CAMOUFLAGE** (haut).

Il manque le mot, pas le sens.

### Ce qui a été tenté et RÉFUTÉ — ne pas rejouer

| voie | mesure |
|---|---|
| chaînes du `.module` | table de chaînes **vide sur 88/88 modules** |
| chaînes des tags `food` / `fpal` | aucun nom |
| tag `forg` (753 Ko) | 3 chaînes, toutes des noms de nœud de script |
| `GlobalID = murmur3(chemin de tag)` | testé sur 2 identifiants de niveau connus × 6 formes : 0 |
| dictionnaire des chaînes du binaire | 105 291 mots (2 tokenisations) : **0** sur les 8 |
| vocabulaire armes/power-ups ciblé | combinaisons jusqu'à profondeur 3, 666 159 essais : **0** |
| dictionnaire Halo communautaire | 616 697 noms `hachage:nom` : **0** sur les 8 |
| `tagnames.txt` communautaires (2 dumps) | autre build : intersection **exactement 0** sur 4 235 `food` |
| chaînes littérales des `fosp`/`foki` | 575 chaînes récoltées, mais les 8 StringID sont **absents des 52 tags `fosp`** (623 768 octets fouillés) |
| appariement StringID ↔ littéral dans `fosp` | **réfuté** : les murmur3 des 5 écritures d'« Active Camouflage » sont absents des 24 512 octets de son propre tag, petit ET gros-boutiste |
| `Forge Kit` (`kit!`) | **un seul tag** dans le jeu installé, 10 suites imprimables, toutes des marqueurs |
| Ghidra, constante murmur3 `0xCC9E2D51` | **plus de 700 occurrences** : trop diffus pour cibler |

### PISTE 1 — JOUÉE LE 2026-08-02. Positive sur son test, négative sur son espoir.

> Détail complet : état de l'art §Q1.0-septies. Sorties : `noms_craques_stringid.txt`,
> `crate_variants.txt`, `representation_names.csv`.

**Ce qu'elle rend :**

- les divergences **existent** (51 couples carte/type ; la plus nette : Catalyst,
  type `493070541`, deux variantes sur la même carte) ;
- **la variante de caisse est un murmur3 de nom simple**, et les quatre entrées
  « emplacement » forment un ensemble **complet** : `banished_kinetic` (`1486653438`),
  `banished_plasma` (`1882451900`), `banished_shock` (`-1062552774`),
  `banished_hardlight` (`801517767`) — les 4 classes de dégât Banished, sans trou ;
- **le champ `/#8/#24[]/#1/#0` nomme l'OBJET** : `warthog`, `wasp`, `ghost`, `banshee`,
  `scorpion`, `mongoose`, `phantom`, `sniper_rifle`, `assault_rifle`, `bandit`,
  `commando_rifle`, `frag_grenade`, `plasma_grenade`, `spike_grenade`, `shade_turret`,
  `plasma_turret`, `unsc_turret` — 26 noms pour 0,027 collision fortuite attendue, dont
  `skull_weapon` **déjà présent** dans `objectives.go` : validation croisée gratuite ;
- **un 5ᵉ type d'emplacement** existe : `493070541` (8 cartes, 102 instances) ;
- **le râtelier mural est épinglé** : Cliffhanger, type `1882451900`, objet d'index 160,
  `up.z = 0,046`.

**Ce qu'elle réfute :** la variante **ne choisit pas** l'objet. Sur Vagabond, les deux
emplacements — celui du lance-roquettes et celui du camouflage — portent **la même**
variante `banished_kinetic` et le même délai de 120 s. La variante est une **famille**
(classe de dégât, style, matériau), pas une identité.

**Ce qui reste donc :** nommer les quatre `Representation Name`
(`-1412311642`, `-245254093`, `-1351408675`, `-219174009`). Ils résistent désormais à
**cinq** vocabulaires indépendants, dont un vivier neuf (les noms d'objets écrits par les
auteurs de cartes). `rocket_launcher`, `active_camo` et `powerup_active_camo` ont été
essayés à profondeur 3 et **réfutés** — ne pas les rejouer.

### PISTE 2 — ÉCARTÉE SUR MESURE. Il manque un DICTIONNAIRE, pas un SCHÉMA.

> Détail : état de l'art §Q1.0-nonies.

Son but d'origine — énumérer les variantes de caisse — est atteint autrement. Son but
résiduel a été testé : une définition de tag décrit une **disposition de champs**, or on
sait déjà où sont les champs (diff différentiel des quatre `food`, à l'octet près :
`Representation Name` en 1296, `Crate Variant` en 708 et 1300). Le blocage est « quelle
chaîne hache vers `-1412311642` » — **un schéma ne répond jamais à ça**.

Et le mur est identique un cran plus bas : le tag `weap` référencé par les quatre
(`-370671751`, 14 272 o) ne porte **aucun nom** — 49 suites imprimables, toutes des
marqueurs fourCC en petit-boutiste.

**Trouvaille au passage, publiée plutôt que comblée** : une **troisième** valeur d'identité,
aux offsets 688 et 732, une par entrée — `1067333871` / `-2060575876` / `1128236838`. Ni un
murmur3 de nom (0 correspondance, 3 vocabulaires), ni un identifiant de tag (`introuvable`
sur les 88 modules). Un troisième espace de nommage est en jeu.

**Et une correction de nature** : `493070541` n'est pas une entrée de palette `food` mais un
tag **`weap` de 14 088 octets** posé directement sur Catalyst.

### Les pistes qui restent, par valeur décroissante

1. **L'attribution par observation** — et c'est la seule qui ne demande aucune percée. Le
   `Representation Name` est un identifiant STABLE : on s'en sert sans connaître son libellé
   humain, et une seule confrontation au Théâtre nomme une entrée pour toujours. Le relevé
   terrain de Vagabond en épingle déjà deux (le bas / le haut).
2. **Un dictionnaire communautaire couvrant le build Forge** — les trois essayés ne le
   couvrent pas (intersection **exactement 0** sur 4 235 `food`).
3. **Ghidra, correctement ciblé** : non pas la constante de hachage mais le **parseur du tag
   `food`** (fourCC 0x666F6F64, ou la vtable du chargeur). Session de rétro-ingénierie à part
   entière, et sans garantie que la table de résolution soit encore dans le binaire.

---

## LE CONTRAT DE DONNÉES — **LIVRÉ en `schema_version` 2**, et adossé à deux témoins

Détail complet dans l'état de l'art. Les cinq règles :

1. **demi-extents en sortie, en mètres** — la conversion (`half_x = s5/2` pour la boîte,
   `radius = s5` pour le cylindre, division par 65536) se fait UNE fois à la lecture ;
2. **orientation obligatoire** dès qu'il y a une forme — ignorer `forward` se trompe de 31 % ;
3. **pas de forme = pas de champ `shape`**, et le rendu affiche un POINT — **jamais de
   disque par défaut** (65,8 % des objectifs sont des points) ;
4. **conserver le brut à côté du dérivé** (`raw: {family, s5..s8}`) ;
5. **pas de nom de zone** : les lettres A/B/C ne sont pas dans le fichier (trois zones d'une
   même carte ne diffèrent que par position, dimensions et `team_index`).

**CE QUI EST IMPLÉMENTÉ** *(2026-08-02 — détail dans l'état de l'art, section « LE CONTRAT
DE DONNÉES »)* :

| élément | où |
|---|---|
| lecture du sac de forme | `internal/analysis/replay/mapvar/shape.go` (`readShape`, `Object.Shape()`) |
| brut conservé à côté du dérivé | `mapvar.Object.ShapeRaw` |
| champ de sortie | `mapvar.Objective.Shape` (`omitempty`) |
| bump de schéma | `cmd/mapobj-build/catalog.go` — `catalogSchemaVersion = 2` |
| régénération HORS LIGNE | `cmd/mapobj-build --refresh-from <dossier de .mvar>` |
| catalogue régénéré | 34 cartes, 597 objectifs, **264 avec forme**, migration v1→v2 34/34 |
| tests d'ancrage | 4 dans `mapvar_test.go`, dont le témoin demi-extent / taille pleine |

**CE QUI NE L'EST PAS** : le **rendu**. Et le catalogue n'a encore **aucun lecteur**, ni Go
ni web (grep `MapObjectivesPath` / `map_objectives` : zéro consommateur) — le brancher est un
chantier à part entière, tenu par `.ai/PLAN_OBJECTIFS_TEMPS_REEL.md` étapes 1-2.

---

## OÙ EST TOUT

| quoi | où |
|---|---|
| état de l'art | `.ai/ETAT_DE_L_ART_FORGE_PALETTE_ZONES.md` |
| sorties de mesure (14 fichiers) | `.ai/V7.5/dumps/forge_zones/` |
| définitions de tag de référence | `.ai/V7.5/dumps/forge_zones/reference_{food,foki,fosp,kit}_tag_definition.xml` |
| chaînes de l'interface Forge | `.ai/V7.5/dumps/forge_zones/forge_ui_chaines_litterales.txt` |
| images | `zones_cliffhanger.png`, `catalyst_candidats.png`, `catalyst_powerup_candidat.png`, `vagabond_candidats.png` |
| **outillage archivé + restauration** | `.ai/V7.5/outillage/forge_palette/README.md` |
| les 199 `.mvar` (non versionnés) | `.claude/worktrees/filmdec-continuation/.ai/re_dump/mapvar/` — copie sur `E:/LevelUp_rejeu2D/captures_cheat_engine/mapvar/` |
| palette Forge | `D:/SteamLibrary/steamapps/common/Halo Infinite/deploy/{any,ds}/globals/forge/` |

**Outillage** : `cmd/tmp_forgename`, `tmp_forgeshape`, `tmp_forgedraw`, `tmp_zonetest` —
gitignorés par convention (`.gitignore:294`), archivés sous `.ai/V7.5/outillage/forge_palette/`
avec leur commande de restauration. CGO requis (`PATH` doit contenir `C:\msys64\ucrt64\bin`).

---

## RESTES SECONDAIRES, PUBLIÉS PLUTÔT QUE COMBLÉS

- le label de carte `-831896525` (65 objets de zone) — non craqué ;
- 27 `type_id` irrésolus (« aucune ref objet dans le food ») sur 2 785 ;
- « présent au départ » — trois drapeaux booléens candidats, aucun ne passe un test de
  corrélation (en Bond, un drapeau dont le défaut est vrai ne s'écrit jamais : ambigu par
  construction) ;
- 5 labels craqués cette session à porter dans `objectives.go` : `extraction_include`
  (−903313158), `firefight_include` (2140598169), `minigame_include` (248451123),
  `forge_include` (−1875636905), `assault_bomb` (−534119345).

## MODIFICATIONS HORS PÉRIMÈTRE DE RECHERCHE — à connaître

D'abord le contrat de données lui-même : `mapvar/shape.go`, `mapvar.Objective.Shape` et
`cmd/mapobj-build` (`catalogSchemaVersion = 2`, `--refresh-from`) sont du code livré, avec
leurs quatre tests — cf. la section précédente.

Ensuite une modification de DONNÉES, autorisée par le superviseur le 2026-08-02 :
`cmd/mapquant-build` gagne
`"Vagabond": "fo08_wetland"` **avec sa raison écrite** (le `level_id` 88891201 rend une
seule occurrence sur 88 modules, groupe `levl`), et le catalogue
`data/titles/halo_infinite/reference/map_quant_bounds.json` passe de 14 à **15 cartes**.
Sans quoi l'oracle d'exécution était impossible.

**Piège rencontré, à ne pas refaire** : `mapquant-build` et `replay-build` résolvent la
racine du dépôt en remontant l'arborescence — depuis un worktree ils écrivent dans l'ARBRE
PRINCIPAL. Exporter `LEVELUP_REPO_ROOT` vers le worktree avant de les lancer.
