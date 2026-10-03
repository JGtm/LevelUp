# RAPPORT — Armes lâchées au sol, lecture tactique (lot 6.3, 2026-09-10)

> **Diagnostic seul.** Aucun code de production modifié, aucune base ouverte, aucune recuisson.
> Branche `wt/armes-au-sol`, worktree `LevelUp-wt-armes-au-sol`.
> Demande utilisateur : « visualiser d'un point de vue tactique qui lâche des armes spéciales
> que quelqu'un peut récupérer » ; les munitions restantes ne sont pas un chiffre à afficher —
> « si les avoir ne coûte pas cher, on les capte mais on n'affiche pas, pas encore ».

---

## 0. LE RÉSULTAT QUI COMMANDE TOUS LES AUTRES

**Le calque « Armes au sol » ne dessine RIEN à l'écran aujourd'hui, sur aucun match, et ce
n'est pas une question de couverture : c'est une clé de jointure qui ne joint pas.**

- `groundWeapons[].w` est écrit `"30484ea6"` — minuscules, sans préfixe
  (`document_ground_weapon_items.go`, `fmt.Sprintf("%08x", o.FamilyID)`).
- `weaponLabels` est indexé `"0x30484EA6"` — MAJUSCULES, préfixé. Vérifié sur les 64 artefacts :
  aucune clé nue minuscule (`grep '"weaponLabels":{"[0-9a-f]\{8\}"'` → 0 fichier).
- `useReplayGroundWeapons.ts` appelle `padIconRefFor(item.w, labels, titleSlug)` ;
  `padIconRefFor` fait `labels?.[weapon]` — une lecture EXACTE, sans normalisation
  (`useReplayWeaponPads.ts:131`). Elle rend `undefined`, donc `null`.
- `groundWeaponsLayer.ts` : `const icon = style.iconOf(item.w); if (!icon) continue`.
  Sans vignette, RIEN n'est dessiné — c'est la règle écrite en tête du calque, et elle
  s'applique ici à 100 % des objets.

Le calque voisin des socles ne souffre pas du défaut : `weaponPads[].weapon` vaut
`"0x80977BA5"`, la même écriture que `weaponLabels`. Le test unitaire du calque
(`groundWeaponsLayer.test.ts:38`, `w: '0a1992bc'`) passe parce qu'il branche un `iconOf` de
substitution : il mesure le tracé, jamais la résolution.

> **À confirmer d'un coup d'œil à l'écran par le superviseur** (le serveur local n'a pas été
> sollicité depuis ce worktree). Si c'est confirmé, tout ce qui suit se lit dans cet ordre :
> aucune infobulle, aucun filtre et aucun bloc de match ne vaut quoi que ce soit tant que
> l'objet n'est pas visible.

Le même écart d'écriture touche `weaponChanges[].w` / `.from` (`%08x` également) : tout futur
lecteur web de ce canal qui voudrait nommer l'arme heurtera le même mur.

---

## 1. MÉTHODE

- **Corpus** : les 64 artefacts de rejeu cuits du parc local,
  `data/cache/replays/halo_infinite/<8 hex>.json`, schéma 51 (les `.derived.json` ignorés).
  Séparation demandée : **61 films d'arène** (roster ≤ 16) et **3 films de Grand combat**
  (roster > 16), comptés à part et jamais mêlés aux pourcentages.
- **Instrument** : trois `*_research_test.go` jetables déposés dans
  `apps/go-api/internal/games/weapons/` (le paquet qui porte le registre d'armes STATIQUE, donc
  aucune base à ouvrir), supprimés après mesure — précédent E0. Ils lisent le JSON des artefacts
  avec des structures locales et n'appellent aucun code de production. Sortie brute intégrale au
  §7.
- **Lecture de code** : `build_ground_weapons.go`, `document_ground_weapons.go`,
  `document_ground_weapon_items.go`, `ground_weapon_rules.go`, `ground_weapon_objects.go`,
  `ground_weapon_pads.go`, `document_weapon_changes.go`, `inventory.go`, `options.go` ;
  côté web `groundWeaponsLayer.ts`, `useReplayGroundWeapons.ts`, `useReplayWeaponPads.ts`,
  `hoverLayers.ts`, `MatchPadControlSection.tsx` ; page Tactique
  (`features/tactical/`, `internal/service/tactical_service*.go`, `internal/analysis/tactical/`,
  `domain/tactical_raster.go`).
- **« Arme spéciale »** : la définition retenue est **`role ∈ {sniper, power, special}`** du
  registre canonique (`internal/games/weapons/registry.go`). C'est la dimension FONCTION de
  combat, pas la manipulation : elle isole exactement ce qu'un socle distribue et ce qu'un
  adversaire a intérêt à ramasser (S7 Sniper, Shock Rifle, SPNKr, Skewer, Cindershot, Hydra,
  Épée, Marteau, Needler, Sentinel Beam) et laisse dehors le CQS48 Bulldog (`shotgun`) et tout
  l'arsenal de départ. La classe `heavy` a été écartée comme critère : elle décrit la façon de
  porter l'arme et retiendrait le Bulldog tout en laissant le Needler dehors.
  Jointure `w` → rôle : high-32 des `weaponRegistryInfiniteFilmshell` → `weapon_key` →
  `weaponRegistryWeapons.role`. **Couverture : 50 objets sur 12 466 (0,4 %) hors registre**,
  tous sur une seule famille, `d7915565`.

---

## 2. QUESTION 1 — CE QUE LE CANAL COUVRE AUJOURD'HUI

### 2.1 Les volumes, arène (61 films)

| Grandeur | Parc | Part |
|---|---|---|
| Objets de la chaîne | 14 444 | — |
| **Objets publiés dans `groundWeapons`** | **12 466** | moyenne **204,4 / film** |
| Laissés au calque des socles (`atRest`) | 1 978 | 13,7 % de la chaîne |
| `origin = dropped` (l'arme d'un mort) | **9 794** | **78,6 %** |
| `origin = spawned` (le reste) | 2 672 | 21,4 % |
| **`dropper` renseigné** | **9 794** | **78,6 %** |
| dont résolus en joueur (`identity.bipedSlots`) | 7 746 | 79,1 % des `dropper` |
| **`picker` renseigné** | **337** | **2,7 %** |
| dont résolus en joueur | 336 | 99,7 % des `picker` |
| `end = pickup` / `seen` / `open` | 337 / 11 770 / 359 | 2,7 % / 94,4 % / 2,9 % |

Grand combat (3 films) : 1 867 objets (622,3 / film), `dropped` 56,5 %, `picker` 1,8 %,
`dropper` résolu en joueur 58,0 %. **Deux fois plus dense et deux fois moins nommé** — c'est
cohérent avec la décision D13 (Grand combat non traité) et rien n'en est tiré ici.

### 2.2 Le lâcheur n'est PAS ce que le contrat annonce — et c'est un piège documentaire

Le commentaire de `GroundWeapon.Dropper` dit : « le slot de la VIE qui l'a lâchée, **quand un
lâcher du flux delta coïncide** (même paquet à 500 ms près, moins de 1,5 m) ». **Le code ne fait
pas cela.** `Dropper` vaut `o.DropperSlot`, posé par `gwPadsClass` — le slot de la vie de bipède
qui **S'ACHÈVE** à moins de 200 ms et 1,5 m de la naissance de l'objet, c'est-à-dire la règle qui
classe l'apparition `dropped`. Le seul autre site qui écrit ce champ n'existe pas
(`grep DropperSlot` : une seule affectation).

**Mesure décisive : `dropper` renseigné == `origin = dropped`, exactement, 9 794 = 9 794.**
Il n'existe pas un seul objet `spawned` avec un lâcheur nommé, sur 2 672.

Conséquence, et c'est la réponse directe à la question posée :

> **Le canal distingue les deux gestes, mais il ne nomme QUE le lâcher à la mort.**
> `origin = dropped` **est** le lâcher à la mort — la distinction est déjà mesurée, elle n'a
> pas besoin d'une jointure avec le fil des morts. Le lâcher **volontaire** tombe dans
> `spawned`, mélangé aux armes éjectées d'un râtelier et aux armes de départ abandonnées, et il
> y est **anonyme**.

Ce que serait la jointure manquante, et ce qu'elle rapporterait : voir §5, option E — mesurée,
et le rendement est faible.

### 2.3 Les armes spéciales

| Grandeur | Parc (arène) | Part |
|---|---|---|
| Objets classés `sniper` / `power` / `special` | **2 168** | **17,4 %** des publiés |
| dont `origin = dropped` (donc lâcheur nommé) | **1 665** | 76,8 % des spéciales |
| dont `end = pickup` (ramassage observé) | **220** | 10,1 % des spéciales |
| Chaîne COMPLÈTE (lâcheur ET ramasseur nommés) | **187** | 3,1 / film |

Ventilation des rôles sur les 12 416 objets classés : `automatic` 4 641, `sidearm` 2 738,
`precision` 2 513, **`power` 1 173**, **`sniper` 578**, **`special` 417**, `shotgun` 356.

**Le chiffre qui porte l'intérêt tactique** : sur les **337** ramassages observés du parc,
**220 (65,3 %)** portent sur une arme spéciale, alors que les spéciales ne font que **17,4 %**
des objets au sol. Quand une arme au sol est reprise, c'est presque deux fois sur trois une arme
qui compte. La rareté du signal (2,7 %) est donc compensée par sa qualité.

Chaîne complète toutes armes : **280** objets sur le parc (4,6 / film), dont 187 spéciales.

### 2.4 Pourquoi le ramasseur est si rare, et ce que ce n'est PAS

Ce n'est pas un défaut d'appariement : c'est le SOURCE qui est mince, et l'appariement, lui, est
sévère par construction.

- `weaponChanges` publie **2 089** changements sur le parc (2 378 décodés, 289 ré-annonces
  écartées) : **taken 1 366, dropped 609, swapped 114**.
- Le lieur reçoit `taken + swapped` = **1 480 prises**, et n'en lie que **337 (22,8 %)**.
  Les 1 143 autres sont des prises de drapeau, des prises AU SOCLE (l'objet n'a jamais bougé,
  il appartient à `weaponPads`) ou des prises d'objet sans piste — la couverture du document
  le dit elle-même (`groundWeaponItems.takesTotal` / `pickupLinked`).
- En regard, le parc porte **416 socles** d'arme, **597 grappes** et **1 601 occupations
  datées** : le ramassage AU SOCLE est déjà servi ailleurs (`padPickups`, et l'écran
  `MatchPadControlSection`). Ce qui manque ici est le ramassage d'une arme **par terre**.

### 2.5 Deux réserves mesurées, à porter au registre

1. **Deux films sur 61 (3,3 %) ont TOUTE l'attribution de proximité à zéro** : `30724141`
   (222 objets, `dropperNamed = 0`, `pickupLinked = 0`) et `0797ce72` (217 objets, idem). Sur
   ces deux films, `coverage.placements.withOwner` vaut **0** également, alors que
   `placements.lives` vaut 342 et 317. Le défaut n'est donc **pas** propre aux armes au sol :
   c'est la jointure « qui était là » qui s'effondre en bloc. **Cause non instruite — hors
   périmètre de ce lot** (règle « zéro fix opportuniste »). À rapprocher du lot 6.1 (pont aplati)
   et du point « `94a28b8b`, 5 vies sans identité » du lot 6.4.
2. **20,9 % des `dropper` ne se résolvent pas en joueur** (2 048 sur 9 794) via
   `identity.bipedSlots`, contre 0,3 % des `picker`. Une infobulle doit donc savoir dire
   « lâchée par un joueur non nommé » une fois sur cinq, et ne jamais inventer.

---

## 3. QUESTION 2 — CE QUE L'ÉCRAN MONTRE DÉJÀ

### 3.1 Le calque « Armes au sol » du rejeu

**Ce qu'il est censé montrer** (`groundWeaponsLayer.ts`, `useReplayGroundWeapons.ts`) :

- la **vignette de l'arme**, posée à plat sur sa position de repos, hauteur 6,5 px, cernée d'un
  liseré de 0,9 px — délibérément plus petite et plus discrète que la vignette d'un socle
  (8 px), « une arme abandonnée est un fait secondaire du terrain » ;
- **ni losange, ni compte à rebours** : le losange dit un LIEU qui réapprovisionne, une arme au
  sol n'est pas un lieu ;
- une **opacité qui EST la mesure** (`groundWeaponPresenceAt`) : pleine tant qu'une preuve de
  présence tient, dégradée dans l'intervalle `[t1, t1max]` où le film ne dit plus rien, rien
  au-delà de la première preuve d'absence ;
- une **bascule** dans le tiroir des réglages (`ReplaySettingsLayers.tsx`, i18n
  `layerGroundWeapons` / `layerGroundWeaponsHint`, FR et EN), allumée par défaut
  (`replayCompose.ts:187`), masquée si le film ne porte aucun objet.

**Ce qu'il ne montre pas, et le code le dit lui-même.** L'en-tête de `useReplayGroundWeapons.ts`
porte la phrase : « PAS DE SURVOL, ET C'EST DÉLIBÉRÉ (périmètre du lot du 2026-08-30) : le calque
affiche, il n'interroge pas. […] **Ce qu'on perd est nommé : le NOM de l'arme au sol ne se lit
nulle part, seule sa silhouette la dit.** »

Donc : **ni le lâcheur, ni le ramasseur, ni le nom de l'arme ne sont lisibles.** Les champs
`dropper` et `picker` sont dans le document, servis à chaque rejeu, et **aucun lecteur web ne les
ouvre** — vérifié par grep sur `apps/web/src` : les seuls consommateurs de `groundWeapons` sont
`useReplayGroundWeapons.ts` (position + `w`), `groundWeaponTime.ts` (`t0`/`t1`/`t1max`),
`replayNormalize.ts` (recopie) et les tests de contrat.

Le survol EXISTE pour trois calques voisins (poses, socles d'arme, drapeaux) et sa distribution
est déjà factorisée : `hoverLayers.ts` promet en toutes lettres qu'« un quatrième calque
survolable s'ajoutera en UNE ligne, à un seul endroit ». L'infobulle a son modèle
(`ReplayWeaponPadTip.tsx`, 71 lignes) et son hook (`useReplayWeaponPads.ts`, partie survol).

**Et par-dessus tout cela : la vignette ne se résout pas (§0), donc rien n'est peint.**

### 3.2 Les autres écrans

- **`MatchPadControlSection`** (onglet du match) répond DÉJÀ à « qui a tenu le fusil de
  précision, qui a raflé l'épée » — mais **uniquement pour les SOCLES** (`padPickups[].xuid`,
  schéma 30). Une arme reprise **par terre** n'y entre pas. C'est le voisin naturel d'un futur
  bloc « lâchée / reprise ».
- **`MatchEquipmentUsageSection`** compte les socles vidés, sans ramasseur.
- **Page Tactique** : cinq lectures et cinq seulement — `morts`, `kills`, `gagne`, `temps`,
  `routes` (`features/tactical/i18n.ts:60-64`). **Rien sur les armes.** Et la page ne lit PAS
  l'artefact : elle lit un **sidecar** par match (`domain.TacticalRasterSidecar`) qui ne porte
  que des cellules d'occupation, des spawns, des premières entrées et des routes. Une lecture
  d'arme y exigerait un champ de plus dans le sidecar, donc une **recuisson des sidecars**
  (pas des artefacts, qui portent déjà tout).

---

## 4. QUESTION 4 — LES MUNITIONS AU LÂCHER

### 4.1 Ce que porte l'inventaire, et à quelle cadence

`Inventory` (`inventory.go`) est lu **AUX IMAGES-CLÉS**, et seulement là. `Am []AmmoSlot` donne,
« dans l'ordre de `Loadout.W` », un `mag` (chargeur), un `res` (réserve) et une `gauge`
(fraction consommée, pas restante — piège documenté). Le canal DELTA d'inventaire ne transporte
que les **grenades** : le canal munitions y est refusé en bloc, et **18 artefacts sur 64 portent
`ammoRefused: true`** — pour ceux-là, aucune munition ne viendra jamais des paquets delta.

Volumes, arène : **11 974 lectures d'inventaire**, dont **9 678 (80,8 %) portent un bloc `am`** ;
2 286 lectures vides. Champs présents : `mag` 17 346, `res` 19 356, `gauge` 1 510.
L'inventaire et les loadouts partagent leurs instants : **1 552 instants communs sur 1 566** —
la jointure « la ligne `am[i]` est l'arme `loadouts.w[i]` » est donc praticable.

### 4.2 Ce qu'on peut joindre, et à quel prix de fraîcheur

Pour chacun des **9 794** objets à lâcheur connu, on cherche la dernière lecture d'inventaire de
ce slot AVANT `t0`, et l'on vérifie que le loadout du même instant porte la famille lâchée :

| Issue | Objets | Part |
|---|---|---|
| **Lecture exploitable** (l'arme est au loadout, l'emplacement est indexable) | **7 677** | **78,4 %** |
| L'arme n'est pas au loadout de cette lecture | 390 | 4,0 % |
| Lecture marquée vide (`empty`, mort ou inconnue) | 112 | 1,1 % |
| Aucune lecture avant `t0` | 1 615 | 16,5 % |

- **Fraîcheur** : écart `t0` − lecture retenue, **p50 = 9,0 s, p90 = 18,0 s, p99 = 31,1 s**.
  En prenant la lecture la plus proche (avant OU après, ce qui n'a pas de sens pour un mort) :
  p50 = 5,0 s, p90 = 10,2 s.
- **`mag` absent dans 809 des 7 677 cas** (10,5 %) : ce sont les armes à jauge, qui n'émettent
  pas de chargeur.
- **Armes spéciales** : **1 241 des 2 168** (57,2 %) ont une lecture exploitable.

### 4.3 Verdict

> **Techniquement joignable, à coût quasi nul — mais ce n'est PAS « les munitions à l'instant
> `t0` ».** C'est « les munitions à la dernière image-clé avant le lâcher », en retard de 9 s en
> médiane et de 18 s au neuvième décile. Or ces 9 secondes sont exactement celles pendant
> lesquelles le porteur a tiré : c'est la fenêtre où la valeur change le plus.
> Un champ nommé `ammo` mentirait ; un champ nommé `ammoAtLastKeyframe` dirait la vérité et
> n'aurait aucun lecteur.

**Recuisson : NON — et même : rien à cuire du tout.** `inventory`, `loadouts` et `groundWeapons`
sont **tous les trois déjà dans le document servi**. N'importe quel lecteur (web ou Go) peut
faire la jointure à la lecture, sans champ nouveau, sans bump de `SchemaVersion`, sans passe de
backfill. La cuire dans l'artefact ne ferait qu'y figer un chiffre périssable et imposerait une
recuisson des 64 artefacts locaux **et** de la production.

**Recommandation** : ne rien cuire, ne rien afficher, et consigner ici la recette de jointure
(§4.2) pour le jour où la question se reposera. La capture « à bas coût » demandée est donc
acquise **sans écrire une ligne** : elle est déjà dans le document.

**Le seul canal qui daterait vraiment les munitions au lâcher** serait le canal munitions des
paquets delta — refusé en bloc par le scanner sur 18 films sur 64, et non exploité pour les
munitions. Le rouvrir est un chantier de décodage, pas un branchement : hors sujet ici.

---

## 5. QUESTION 3 — LES OPTIONS DE LECTURE TACTIQUE

| # | Option | Données présentes ? | Effort | Recuisson | Fichiers | Impact |
|---|---|---|---|---|---|---|
| **0** | **Réparer la jointure de vignette** (§0) : normaliser la clé avant `labels?.[weapon]`, + garde-rail sur la forme des clés | Oui — c'est un défaut, pas un manque | **XS** | **NON** | `useReplayWeaponPads.ts` (`padIconRefFor`), un test de forme de clé ; variante Go : aligner `%08x` sur `0x%08X` — mais alors recuisson OUI | **Rend le calque visible.** Préalable à tout le reste |
| **A** | **Infobulle du calque** : nom de l'arme, « lâchée par X », « reprise par Y », origine, fenêtre `[t1, t1max]` | Oui, intégralement | **S** | **NON** | `useReplayGroundWeapons.ts` (+ survol), `hoverLayers.ts` (1 ligne), `ReplayCanvas.tsx` (2 lignes — attention au seuil `max-lines`, 751 L), `ReplayCanvasTips.tsx`, nouveau `ReplayGroundWeaponTip.tsx`, i18n FR+EN | Lâcheur lisible sur **78,6 %** des objets (résolu en joueur 79,1 % de ceux-là), ramasseur sur **2,7 %**. Lève la perte que le code se reproche lui-même |
| **B** | **Filtre « armes spéciales »** sur le calque (et sur l'infobulle) | **Non** : le rôle n'est nulle part dans l'artefact. `weaponLabels[].key` donne le `weapon_key`, pas le rôle — mais il est posé **à la requête** (`replay_weapon_labels.go`), donc un `role` s'y ajoute **sans recuisson** | **S** | **NON** | `document_labels.go` (+1 champ), `service/replay_weapon_labels.go`, `api/openapi.yaml` + `make generate-types`, interrupteur web + i18n | Isole les **17,4 %** qui portent l'intérêt tactique et **65,3 %** des reprises observées. Interdit la table d'armes en dur côté web |
| **C** | **Bloc de match « lâchées et reprises »**, voisin de `MatchPadControlSection` : une ligne par arme spéciale, lâcheur → ramasseur | Oui | **M** | **NON** | nouveau `MatchGroundWeaponSection.tsx` + `groundWeaponChainLogic.ts`, i18n, mêmes portes que ses voisines (`useMatchReplay`) | **187 chaînes complètes** sur le parc, **3,1 / film**. Faible volume : la note de pied DOIT dire que 97,3 % des reprises ne sont pas observées, sinon le lecteur croit tenir le total |
| **D** | **Page Tactique, lecture « cellules de lâcher »** (où les armes spéciales tombent, sur N matchs) | **Non** au grain sidecar : `TacticalRasterSidecar` ne porte que cellules / spawns / premières entrées / routes. L'artefact, lui, porte tout | **L** | **OUI** — recuisson des **sidecars** (pas des artefacts) + rattrapage du parc | `domain/tactical_raster.go` (+ `schema_version`), la cuisson des sidecars, `tactical_service_lectures.go`, `tactical_service_cablage.go`, `features/tactical/i18n.ts` (6ᵉ question), `TacticalAnalysisView` | **27 spéciales lâchées / film** en moyenne (1 665 / 61) : sur 38 matchs retenus, ≈ 1 000 points — assez pour une carte. **Le « qui ramasse » n'y tient pas** (2,7 %) : cette lecture ne peut dire que le LÂCHER |
| **E** | **Nommer le lâcheur VOLONTAIRE** en liant `weaponChanges` (`dropped` / `swapped`) à l'objet `spawned` de même famille | Oui, mais le rendement est mesuré et faible | S | **OUI** (champ cuit) | `document_ground_weapon_items.go`, `SchemaVersion` | **12,7 % seulement** : 92 événements appariés sur 723, dont 89 sur un objet `spawned` — soit **+0,7 %** d'objets nommés. **Non retenu** |
| **F** | **Munitions au lâcher** | Oui, déjà servies (§4) | **XS côté lecteur, rien côté serveur** | **NON** | aucun | Capture acquise sans écrire une ligne. Rien à afficher (décision utilisateur) |

### Détail de l'option E — pourquoi elle est écartée, sur pièces

Le contrat de `Dropper` PROMET ce lien (§2.2). Mesuré : sur **723** événements
`dropped` (609) + `swapped` (114), dont 45 sans famille lisible, **92 (12,7 %)** trouvent un
objet de la même famille né à ±1 s. **La fenêtre n'est pas la contrainte** : ±0,5 s, ±1 s et
±2 s donnent le même compte (91, 92, 92). La distance lâcheur → objet au lien à 1 s vaut
p50 = 1,11 m, p90 = 3,90 m, max = 31,72 m (sur les pistes PUBLIÉES, donc décimées : un seuil de
production à 1,5 m en garderait environ la moitié).

La cause probable est mesurée à côté : **1 978 objets (13,7 %) sont classés `atRest`** et laissés
au calque des socles parce qu'ils n'ont émis aucune position delta. Une arme lâchée volontairement
tombe aux pieds de son porteur et ne bouge pas — elle sort donc de `groundWeapons` par
construction. Le lien ne peut pas trouver ce qui n'est pas publié.

**Ce qui reste à faire quoi qu'il arrive : corriger le commentaire de `GroundWeapon.Dropper`**,
qui décrit un mécanisme absent du code (anti-pattern « doc inversée » du CLAUDE.md). Coût XS,
aucune recuisson, à faire dans le lot qui touchera ce fichier.

---

## 6. RECOMMANDATION D'ORDRE

1. **Option 0 — réparer la vignette (XS, aucune recuisson).** Sans elle, tout le reste est
   invisible. À confirmer d'abord d'un coup d'œil à l'écran. Garde-rail obligatoire : un test
   qui fige la forme des clés (`groundWeapons[].w` et `weaponChanges[].w` en `%08x`,
   `weaponLabels` en `0x%08X`) — le dépôt a déjà payé ce genre de divergence.
   Corriger le commentaire de `Dropper` dans le même passage.
2. **Option A — infobulle « lâcheur → ramasseur » (S, aucune recuisson).** C'est la demande
   utilisateur servie au plus court : « qui lâche, où, qui ramasse » se lit alors au rejeu,
   sur l'objet lui-même, sans nouvelle page. Elle doit dire « joueur non nommé » une fois sur
   cinq côté lâcheur, et ne jamais deviner.
3. **Option B — rôle publié + filtre « armes spéciales » (S, aucune recuisson).** Elle rend
   l'option A utilisable sur les matchs denses (204 objets par film, dont 82 % sans intérêt
   tactique) et prépare C et D.
4. **Option C — bloc de match (M, aucune recuisson)** : à ouvrir seulement si A + B confirment
   l'usage à l'écran. 3,1 chaînes complètes par film, c'est peu ; le bloc vaut par la qualité du
   signal, pas par son volume.
5. **Option F — munitions** : rien à faire. La jointure est consignée au §4.2 ; on ne cuit rien,
   on n'affiche rien.
6. **Option D — page Tactique (L, recuisson des sidecars)** : plus tard, et **sur le lâcher
   seulement**. La demande « qui ramasse » ne peut pas y être honorée avec 2,7 % de ramasseurs
   nommés — l'y mettre produirait une carte qui ment par omission.
7. **Option E — non retenue** (12,7 % de rendement, recuisson exigée).

### Ce que ce rapport ne tranche pas

- **La cause du double zéro d'attribution sur `30724141` et `0797ce72`** (§2.5, réserve 1). Il
  faudrait comparer le nuage de positions et la carte des vies de ces deux films à ceux d'un
  film sain — c'est le même geste que le lot 6.1 et il n'a pas été fait ici.
- **La confirmation à l'écran du §0.** Le raisonnement est complet et vérifié sur les 64
  artefacts, mais aucun rendu n'a été observé depuis ce worktree.

---

## 7. SORTIE BRUTE DE L'INSTRUMENT

Instruments : `internal/games/weapons/armes_au_sol{,2,3}_research_test.go`, supprimés après
mesure. 64 artefacts, schéma 51, `data/cache/replays/halo_infinite/`.

```
=== ARENE (roster <= 16) : 61 films ===
objets publies         12466 (moyenne/film 204.4)
  origin=dropped       9794 (78.6%)
  origin=spawned       2672 (21.4%)
  dropper connu        9794 (78.6%)
  dropper -> xuid      7746
  picker connu         337 (2.7%)
  picker -> xuid       336
  end pickup/seen/open 337 / 11770 / 359
  SPECIALES            2168 (17.4%) dont dropped 1665, ramassees 220, lacheur nomme 1665
  famille inconnue     50   (une seule famille : d7915565)

=== GRAND COMBAT (roster > 16) : 3 films ===
objets publies         1867 (moyenne/film 622.3)
  origin=dropped       1055 (56.5%)   origin=spawned 812 (43.5%)
  dropper connu        1055 (56.5%)   dropper -> xuid 612
  picker connu         34 (1.8%)      picker -> xuid 33
  end pickup/seen/open 34 / 1762 / 71
  SPECIALES            139 (7.4%) dont dropped 109, ramassees 15

=== COUVERTURE PUBLIEE PAR LES ARTEFACTS (arene) ===
objets de la chaine 14444, publies 12466, laisses aux socles (atRest) 1978
lacheur nomme 9794 (78.6%)
prises recues (takesTotal) 1480, liees a un objet 337 (22.8%)
socles d'arme (pads) 416, grappes 597, occupations datees 1601
films non balayes 0, films a 0 lacher a la mort 2   (30724141, 0797ce72)
weaponChanges : decodes 2378, re-annonces 289, publies 2089
                (taken 1366 / dropped 609 / swapped 114)

=== ROLES des armes au sol (arene) ===
  automatic    4641      power        1173
  sidearm      2738      sniper        578
  precision    2513      special       417
  shotgun       356
chaine complete (dropper ET picker connus, end=pickup) : 280 ; dont speciales 187 / 220 prises

=== D. LACHERS VOLONTAIRES ===
evenements dropped+swapped 723 (sans famille 45) ; objets origin=spawned 2672
  apparies a +/-500 ms : 91 (12.6%)
  apparies a +/-1000 ms : 92 (12.7%)  — dont objet dropped 3, spawned 89
  apparies a +/-2000 ms : 92 (12.7%)
  distance lacheur-objet au lien 1 s : p50=1.11 m p90=3.90 m max=31.72 m (n=92)

=== E. MUNITIONS AU LACHER ===
lectures d'inventaire 11974 dont avec bloc am 9678 (80.8%), vides 2286
champs presents dans les blocs am : mag 17346, res 19356, gauge 1510
instants distincts : inventaire 1566, loadouts 1552, communs 1552
objets a lacheur connu 9794 :
  lecture avant portant l'arme 7677 (78.4%) / ne la portant pas 390 / morte 112 / aucune 1615
  chargeur (mag) absent dans 809 des 7677
  ecart t0 - lecture retenue, ms : p50=9000 p90=18000 p99=31100
  ecart t0 - lecture la PLUS PROCHE (avant ou apres), ms : p50=5000 p90=10200
  armes SPECIALES : 2168 au total, 1241 (57.2%) avec une lecture exploitable
films avec coverage.grenadeReads.ammoRefused = true : 18 / 64
```

Témoin de format (les trois écritures qui ne joignent pas) :

```
groundWeapons[].w   "30484ea6"            (%08x, minuscules, sans prefixe)
weaponChanges[].w   "30484ea6"            (idem)
weaponPads[].weapon "0x80977BA5"          (0x%08X)
weaponLabels{}      "0x0A1992BC"          (0x%08X)  -- aucune cle minuscule sur 64 artefacts
loadouts[].w        ["0x2B1824D5", ...]   (0x%08X)
```

## 8. RÉFÉRENCES

- `apps/go-api/internal/analysis/replay/document_ground_weapon_items.go` — le calque publié
- `apps/go-api/internal/analysis/replay/ground_weapon_rules.go` — `gwPadsClass`, la règle du lâcher
- `apps/go-api/internal/analysis/replay/document_weapon_changes.go` — prises et lâchers datés
- `apps/go-api/internal/analysis/replay/inventory.go` — `AmmoSlot`, cadence image-clé
- `apps/web/src/features/match-replay/layers/useReplayGroundWeapons.ts` — le câblage, et la
  phrase « PAS DE SURVOL, ET C'EST DÉLIBÉRÉ »
- `apps/web/src/features/match-replay/layers/useReplayWeaponPads.ts:124` — `padIconRefFor`
- `apps/web/src/features/match-replay/MatchPadControlSection.tsx` — le voisin qui répond déjà
  pour les socles
- `apps/go-api/internal/domain/tactical_raster.go` — ce que le sidecar tactique porte
- `.ai/REFERENCE_CANAUX_EQUIPEMENT_2026-09-09.md` §2 bis — « les armes spéciales suivent la
  même grammaire »
