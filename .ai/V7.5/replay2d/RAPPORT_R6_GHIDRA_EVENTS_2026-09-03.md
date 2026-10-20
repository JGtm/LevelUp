# RAPPORT R6 — Ghidra sur les événements d'équipement : la charge du 117 lue et validée, le verdict des types muets

Date : 2026-09-03. Lot R6 du plan `PLAN_LECTURE_FIABLE_EQUIPEMENT_2026-09-03.md`. Rétro-ingénierie
statique (décompilation, xrefs, lecture mémoire) sur le projet Ghidra existant `HI`
(`C:\Users\Guillaume\Downloads\HI.gpr`, programme `HaloInfinite.exe`), + validation sur pièces
côté film. LECTURE SEULE des deux côtés : aucun débogueur, aucune écriture binaire, aucun DuckDB.
Instruments (ce worktree, package `apps/go-api/internal/analysis/filmdec/`) :

- `r6_layout117_research_test.go` — `TestR6Layout117` (layout 117 rejoué sur les 18 événements)
- `r6_sondage_liste_research_test.go` — `TestR6SondageListe` (types derrière les têtes fermées)

Gardés par `R6_ROOT`/`R6_ARTS`/`R6_CAT`/`R6_IDS`(/`R6_MAPS`), skip par défaut, `CGO_ENABLED=0`,
lecture O(1) par paquet, `go vet ./internal/analysis/filmdec/` vert. Commandes en annexe C.

## Verdict en cinq phrases

**La charge utile de l'événement 117 est entièrement lue de l'exe et validée 18/18 : un
identifiant d'effet 32 bits gardé, puis DEUX positions quantifiées — position A = le DÉPART du
saut, position B = l'ARRIVÉE, exactes à 0,00-0,26 m sur les 18 événements des 5 films.** **Le
« mot constant 0x42689F84 aligné octet » de R1 était un artefact de fenêtre : le vrai champ du
protocole est un R(32) gardé, constant `0xA1344FC2`.** **La quantification est celle du
catalogue de production (`map_quant_bounds.json`) : la formule génératrice des largeurs, lue
dans l'exe, reproduit exactement les `axisWidths` mesurés du repo.** **Pour les types muets
(30, 42, 43, 48, 93, 104…), l'exe ne fournit PAS de site d'émission lisible (négatif B2a.4
reconfirmé) mais prouve que leur réception est fonctionnelle et leur diffusion non filtrée par
type : l'absence vient de l'émission ; le sondage film n'en trouve aucune occurrence non plus
derrière 597 têtes marchées — verdict « aucune trace, cause exacte indéterminée côté exe ».**
**Découverte latérale : `unit_zoom` (21) EST dans la bobine — 5 occurrences en 2e position de
liste — le verdict E8 « absent » ne valait que pour les têtes.**

## 0. Outillage — comment l'instance Ghidra a été créée (et le piège machine rencontré)

Aucune instance Ghidra ne tournait (`list_instances` : 0 ; TCP 127.0.0.1:8089 refusé). Le
blocage a été LEVÉ sans GUI ni ré-import : le plugin GhidraMCP 5.12.0 installé
(`%APPDATA%\ghidra\ghidra_12.1_PUBLIC\Extensions\GhidraMCP`) contient un serveur headless
(`com.xebyte.headless.GhidraMCPHeadlessServer`, GhidraLaunchable) qui sait OUVRIR un projet
existant (`--project`). Deux pièges, résolus :

1. Le jar du plugin ne doit PAS être sur `-cp` (il serait chargé par le class loader parent,
   qui ne voit pas les jars Ghidra) — le layout le découvre seul depuis le dossier Extensions.
2. **Piège machine** : sous ce JDK (Adoptium 25.0.3.9, celui de `java_home.save`), TOUT
   `Selector.open()` échoue — le pipe NIO interne tente une socket Unix et le `connect0`
   AF_UNIX rend `Invalid argument` (filtrage local, indépendant du sandbox, reproduit sur un
   programme minimal). Contournement prouvé : `-Djdk.net.unixdomain.tmpdir=Q:\nexistepas` fait
   échouer le bind UDS, que le JDK rattrape en repli TCP (`createListener`,
   PipeImpl.java:203-218). Ce piège toucherait AUSSI le serveur HTTP du plugin dans le GUI.

Commande de lancement complète : annexe C.1. Le pont MCP s'y connecte (fallback TCP), mais ses
outils dynamiques ne sont pas exposés à une session déjà ouverte : toutes les requêtes de ce
lot sont passées par l'API HTTP du greffon (`/decompile_function`, `/get_xrefs_to`,
`/read_memory`, `/search_byte_patterns`, `/get_function_by_address`), comme les lots B/E
d'août. Le serveur a été arrêté en fin de lot (verrou projet libéré).

## 1. Question A — la charge utile du 117, champ par champ (source exe, validée film)

### 1.1 Chaîne de preuve côté exe

Descripteur type 117 : objet `0x144724c48` → vtable `0x143d06658`. Slots utiles : lecteur
`+0x68 = 0x140F04FB8`, écrivain `+0x60 = 0x142EEC354`, applicateur `+0x78 = 0x140998434`,
domaines `+0x58 = 0x141166E50` (décodé sur octets : d(0)=2, i≠0 → chemin d'erreur : les refs
1-2 n'existent pas pour ce type).

Lecteur `FUN_140f04fb8` (décompilé) :

```c
FUN_14080d69c(this, flux, charge, 0xffffffff);        // [R(1) g0 ; si g0 : R(32)] -> charge+0x00
si !FUN_14076f91c() : FUN_14076e524(&v, flux, out_region, 0x10)   // vecteur quantifié k=16
sinon                : FUN_1411b259c(&v, flux)                    // 96 bits bruts (mode debug/local,
                                                                  // jamais observé dans les films)
charge+0x04..0x0F = position 1 ; même lecture -> charge+0x10..0x1B = position 2   // 28 octets ✓
```

Écrivain `FUN_142eec354` symétrique : `FUN_1407edb6c` (mot gardé) puis 2 ×
`FUN_141f860b0` → `FUN_1407eb61c(flux, pos, -1, 0x10, 0)`.

Lecteur de vecteur `FUN_14076e524` (décompilé, PIÈGE DE PORTE) :

- `R(1)` — **porte INVERSÉE** : bit **0** → lire `R(w_r)` un INDEX DE RÉGION BSP puis
  quantifier aux bornes de LA région ; bit **1** → bornes par défaut du moteur (±20000,
  `DAT_143b8c6b8`), largeurs 22/22/22. (Le bloc « lire l'index » s'exécute sous
  `cVar6 == '\0'` — vérifié deux fois, et c'est le sens que la validation film confirme.)
- `w_r = DAT_144632be0`, RUNTIME : `FUN_140be9a14` le pose à `log2ceil(nb_régions)` et à **1
  si une seule région** — mesuré 1 bit (valeur 0) sur les 18 événements.
- 3 composants `R(bx) R(by) R(bz)` (`FUN_140cc5128`, MSB-first), largeurs par (région, k)
  dans `DAT_1445ccbe0`, calculées par `FUN_140be9b88` :

```
b[i] = min(26, ceil(log2( min(2^22, ceil(extent[i] / (2*g))) )))
g(k) = (1/120 m) * 2^(16-k)   (FUN_140be9c78 + DAT_143cd9758 = 1/120) ; k=16 -> g = 1/120
```

- Déquantification : `pos[i] = min[i] + (q[i] + 0.5) * (max[i]-min[i]) / 2^b[i]`
  (demi-pas `DAT_143cd84b0` = 0.5).

**Recoupement fort avec la production** : cette formule reproduit EXACTEMENT les `axisWidths`
de `data/titles/halo_infinite/reference/map_quant_bounds.json` (vérifié : aquarius
77,8/46,2/18,1 m → 13/12/11 ✓). Les « régions » du lecteur sont les régions BSP dont le
catalogue dit lui-même tenir ses bornes (« world bounds … de la RÉGION 0 »).

### 1.2 Layout final du record 117 (du 1er bit du paquet)

| Champ | Largeur | Valeur mesurée (18/18) |
|---|---|---|
| bit config + bit continuation | 2 | `11` |
| type `R(7)` | 7 | 117 |
| ref0 : porte, index, génération | 1+8+2 | 1, slot−512, gen — l'unité qui saute |
| portes ref1, ref2 | 1+1 | `00` (jamais présentes) |
| g0 (porte du mot) | 1 | 1 |
| mot `R(32)` | 32 | **`0xA1344FC2` constant** (chaîne-id d'effet) |
| position A : porte région | 1 | 0 (= région) |
| index de région `R(1)` | 1 | 0 (région 0 ; largeur runtime =1 car 1 seule région) |
| `R(bx) R(by) R(bz)` | axisWidths carte | qA → **le DÉPART du saut** |
| position B : porte + index + q | idem | qB → **l'ARRIVÉE du saut** |
| bit de continuation de liste | 1 | **1 : un événement SUIT toujours** (type 15 `Script`, 18/18) |

**Correction de R1 §4.2** : le « mot 32 bits constant `0x42689F84`, aligné octet » n'existe
pas dans le protocole. Le champ réel est le `R(32)` gardé qui commence au bit 23 et vaut
`0xA1344FC2` ; comme le bit 23 vaut 1 et le bit 55 vaut 0, la FENÊTRE d'octets 3-6 affiche
`42 68 9F 84` — un décalage d'un bit pris pour un alignement. La sonde position de R1 (3×16
bits, bornes d'affichage de l'artefact) ne pouvait pas trouver les positions : mauvaises
largeurs ET mauvaises bornes (piège R3 §0.3).

### 1.3 Validation sur pièces : 18/18

`TestR6Layout117` rejoue le layout sur chaque tête 117 des 5 films, identifie la carte par
calibration (l'entrée du catalogue × w_r qui valide le plus d'événements — méthode R3 §0.4),
et confronte A/B à la discontinuité de piste du slot désigné (paire de frames consécutives à
déplacement 2D max dans [f−2, f+6]). Critère : A et B à ≤ 1,5 m (2D) du couple from/to.

| Film | Carte calibrée (bits) | Événements | Verdict | Écarts dA / dB (m) |
|---|---|---|---|---|
| `1b2d9e08` | canevas fo13_frost [15 15 17] | 3 | 3/3 | 0,00-0,11 |
| `a0c36016` | forest [13 13 13] | 4 | 4/4 | 0,02-0,22 |
| `4577fcc4` | canevas fo13_frost [15 15 17] | 6 | 6/6 | 0,08-0,26 |
| `f2966f08` | behemoth [17 17 15] | 2 | 2/2 | 0,01-0,22 |
| `faff9935` | canevas fo13_frost [15 15 17] | 3 | 3/3 | 0,04-0,22 |

**18/18, ordre A=départ / B=arrivée sur les 18** (jamais l'inverse), résidus au niveau du
bruit de la piste publiée (arrondie au cm, échantillonnée à 100 ms). Le mode « 96 bits bruts »
(`FUN_14076f91c`) n'apparaît dans aucun film.

### 1.4 Conséquences pour le chantier (D4 / P1.3)

1. **Le va-et-vient complet est dans l'événement** : départ ET arrivée exacts, datés, avec le
   slot — plus besoin de dériver from/to de la discontinuité de piste (P2.2 peut consommer
   directement l'événement), et le filtre vitesse (P1.4) garde son rôle de simple exemption.
2. **La POSE reste illisible** : la charge ne porte ni identité de balise ni position de pose
   — rien de plus que {effet, départ, arrivée}. Entre la pose et le premier échange, la
   position de la faille n'est toujours connue d'aucun canal (confirmé : R1 §1-3 + ce layout).
   Après le PREMIER échange, la faille est à la position de DÉPART du saut (va-et-vient) —
   exactement ce que D4 prévoit de dessiner.
3. L'applicateur `FUN_140998434` (réception) résout l'unité, ouvre le bloc de tag 'eqip'
   (sous-bloc 0x1a) et joue les effets aux deux positions (`FUN_1409984bc`) : effets
   sonores/visuels de départ et d'arrivée — cohérent avec « TeleportEffects », l'événement est
   bien un effet cosmétique répliqué à tous (d'où sa présence fiable dans le film).

## 2. Question B — les types muets : ce que l'exe dit, ce que le film confirme

### 2.1 Côté exe : réception fonctionnelle, diffusion non filtrée par type, émission introuvable

Pour 30/42/43/48/93/104 (objets `0x144724d80`/`e70`/`d18`/`eb8`/`c78`/`c38`) :

1. **Domaines des références** (`vtable+0x58`, décodés sur octets) : 30 et 48 = {4,8,7} ;
   42 = {2,8,7} ; 43 et 93 = {2,0,7} ; 104 = {0,…} ; 103 = {7,0,7} ; 100 et 32 = {1,8,7}.
   (Domaine→largeur : table E2 de la grammaire. Ces valeurs ferment l'hypothèse « refs 13
   bits » de R5 §3.1 pour le 103, désormais sourcée.)
2. **Prédicat de diffusion** (`vtable+0x50`, lu par la pertinence `FUN_1408f0074`) : thunks
   `AND/OR 0x81` (diffusable, vérifier 1 réf) pour 30/42/48/104/117/100/32/39, `0x82` (2 réfs)
   pour 43/103 ; **93 seul est conditionnel** (`FUN_142efa3c4` → `FUN_142f24ee4` : refusé aux
   clients non pertinents quand un champ de charge vaut 2). AUCUN type n'est « jamais
   diffusable » : la réplication ne filtre pas ces types par nature.
3. **Applicateurs non vides** : 48 `FUN_142f12820` EXÉCUTE la pose de câble sur l'arme de
   l'unité reçue (comportement de requête appliquée — C2S) ; 104 `FUN_140bc8170` APPLIQUE une
   poussée (vecteur 12 octets de la charge) à l'unité ; 42 `FUN_142f10c6c` déclenche l'esquive ;
   43 `FUN_142ef7794` lance une action de mobilité. Le moteur sait les recevoir : ce ne sont
   pas des types morts.
4. **Sites d'émission : INTROUVABLES statiquement** — le négatif B2a.4 est reconfirmé et
   étendu : aucun `MOV ECX/EDX/R8D/R9D, <type>` d'émission (les seuls immédiats 0x75 du binaire
   sont le thunk de version du descripteur et du code sans rapport) ; les objets descripteurs ne
   sont référencés que par l'initialiseur de table `FUN_140e453b4` ; la création des slots
   sortants (masque de clients cibles `+0x30` compris) reste anonyme. `vtable+0x30` n'est pas
   un émetteur : c'est `MOV ECX,type ; JMP FUN_141102ed0` — la VERSION par type
   (table statique `DAT_14474cd90`, stride 8 : version au +0, taille de réception au +4).
5. Lecteurs décompilés (grammaires PARTIELLES — largeurs de sous-lecteurs non tracées) :
   30 `FUN_142f16ad8` : [R(1);si 1:R(32)] + R(32) "variant-name" (l'équipement !) + sous-champs
   + R(4) + … + [R(1) ; si 1 : vecteur quantifié k=16] (une position optionnelle) ;
   42 `FUN_142f169d0` : 2 sous-lecteurs + R(8) + 32 bits ; 43 `FUN_142ef8f04` : R(1) +
   `FUN_1408f02c8` (gros record, 164 o) ; 48/93/119 non décompilés champ à champ (sans objet
   pour le verdict). **Les tailles de réception (structures C) ne sont PAS des longueurs en
   bits : aucun « pas de liste complète » générique n'en sort** — la marche de liste n'est
   possible qu'à travers les types à grammaire fermée.

### 2.2 Côté film : le sondage derrière les têtes fermées — zéro occurrence

`TestR6SondageListe` marche la liste d'événements des paquets dont la TÊTE a une grammaire
fermée (103 charge 0 bit ; 100 ; 117 avec la carte calibrée ; les 12 types 0-bit acceptés si
leurs 3 portes de refs valent 0 — domaines non sourcés). 12 films (les 9 de R5 + les 3 autres
témoins R1). Dénominateurs :

- **597 paquets à tête fermée marchés** : 286 finissent la liste proprement (bit 0), 206
  s'arrêtent sur une réf non sourcée d'un type 0-bit, 105 lisent le TYPE de l'événement n°2.
- Types vus en position 2 (parc) : `{0:17 1:3 5:3 6:14 7:3 9:2 11:1 15:22 21:5 23:1 36:20
  38:3 75:3 76:1 82:8}` — distribution plausible (dégâts, projectiles, tirs, Script…), la
  marche est saine.
- **AUCUNE occurrence de 28/30/31/42/43/48/51/93/98/104/115/116/119 en position 2.**

Limites dites : profondeur 1 seulement (l'événement n°2 est presque toujours opaque), et 597
paquets ≠ les 325 160 du parc. Ce sondage ne PROUVE pas « jamais écrit », il constate
« aucune trace là où la grammaire permet de lire ».

### 2.3 Verdict par type

| Type | Verdict R6 | Éléments |
|---|---|---|
| 30 `biped_equipment_activation` | **Aucune trace ; indéterminé côté exe** | 0 tête/325 160 + 0 en pos. 2/597 ; réception et diffusion OK ; version 3 (`DAT_14474cd90`) ; porte une position ET un "variant-name" : ce serait le canal d'usage idéal S'IL était émis — il ne l'est pas dans nos films |
| 42 `biped_dodge` | **Aucune trace ; probablement jamais émis en multi** | idem ; l'applicateur déclenche une esquive de bipède — geste IA/campagne plausible (aucun « dodge » joueur en MP Infinite) |
| 43 `initiate_mobility_action` | **Aucune trace ; probablement jamais émis en multi** | idem (mantle/actions de mobilité) |
| 48 `weapon_tether_request` | **Aucune trace ; requête C2S probable** | applicateur = EXÉCUTE la requête à réception ; une requête client→serveur n'a pas à figurer dans le flux répliqué que le film enregistre |
| 93 `activate_spartan_ability` | **Aucune trace ; seul type à diffusion conditionnelle** | `+0x50` dynamique (contenu-dépendant) ; « spartan ability » = vocabulaire campagne/H5, pas l'équipement MP |
| 104 `EquipmentKnockbackPlayer` | **Aucune trace ; notification ciblée probable** | applicateur applique la poussée à l'unité ; la paire 119 (Request, C2S) / 104 (effet) suggère un envoi au seul client du joueur poussé — jamais au flux enregistré |
| 31/51/98/115/116/119/28 | **Aucune trace** (mêmes dénominateurs) | non instruits individuellement au-delà de §2.1 |

**Formellement : « jamais écrit dans le film » reste non prouvé** (il faudrait le site de
création et son masque de cibles, introuvables statiquement — ou la liste complète décodée sur
tout le parc). Mais le faisceau est convergent : rien en tête (325 160), rien en position 2
(597 lisibles), réception/diffusion fonctionnelles, émission jamais observée.

## 3. Découvertes latérales (hors périmètre, notées — rien traité)

1. **`unit_zoom` (21) EST dans la bobine** : 5 occurrences en position 2 (06dfe6d9,
   084a804d, 4f77afc1, 8a485699, d1dfbc02). Le verdict E8 (« aucun événement zoom dans les
   1 369 films ») ne valait que pour les TÊTES de liste. Pour le chantier lunette : la liste
   complète est la voie.
2. **Un type 15 `Script` suit CHAQUE événement 117** (18/18, et 4 fois derrière d'autres
   têtes) : le translocateur déclenche un événement script accolé dans le même paquet.
3. `action_weapon_fire` (36) : 20 occurrences en position 2 — le canal killsource gagnerait
   aussi à la liste complète.
4. La table `DAT_14474cd90` (version + taille de réception par type, statique) et les thunks
   `+0x30 = version(type)` : utilisable pour vérifier des hypothèses de build.
5. Le piège JDK/AF_UNIX de §0 vaut pour TOUT usage futur de Ghidra sur cette machine (GUI
   compris si le plugin HTTP refuse de démarrer : même cause probable).

## 4. Statut des items du lot

- Question A (layout 117) : **fait — validé 18/18**, layout champ par champ §1.2, correction
  du mot de R1, sémantique A=départ/B=arrivée établie par la mesure.
- Question B côté exe (émetteurs/conditions) : **fait dans la limite du statiquement possible**
  — émetteurs introuvables (négatif argumenté §2.1.4, B2a.4 reconfirmé), conditions de
  diffusion et applicateurs sourcés ; « tailles de charge en bits » : NON disponibles (dit).
- Question B côté film (sondage liste) : **fait — borné et dit** : 0 cible derrière 597 têtes.

## Annexe A — adresses sourcées ce lot (exe retail, projet HI)

| Objet | Adresse |
|---|---|
| Lecteur 117 / écrivain / applicateur / effets | `0x140F04FB8` / `0x142EEC354` / `0x140998434` / `0x1409984BC` |
| Mot gardé lecteur/écrivain | `FUN_14080d69c` / `FUN_1407edb6c` |
| Vecteur quantifié : lecteur / 3 composants / écrivain | `FUN_14076e524` / `FUN_140cc5128` / `FUN_1407eb61c` |
| Bits par axe / granularité / log2ceil | `FUN_140be9b88` / `FUN_140be9c78` + `DAT_143cd9758`=1/120 / `FUN_1406d310c` |
| Tables : bornes défaut / bits défaut / bits par région / largeur index | `DAT_143b8c6b8`(±20000) / `DAT_1445cc9e0` / `DAT_1445ccbe0` / `DAT_144632be0` |
| Init des tables (défaut / par carte) | `FUN_140de9a1c` / `FUN_140be9a14` |
| Demi-pas 0.5 / seuil granularité 1e-4 / 2^22 | `DAT_143cd84b0` / `DAT_143cd837c` / `DAT_143cd975c` |
| Dispatcher réception / boucle de liste | `FUN_14080a9d4` / `FUN_14076a1c4` |
| Enveloppe émission / charge+sentinelle / sélection / pertinence | `FUN_140bbd474` / `FUN_1424d80bc` / `FUN_140bbd2b4` / `FUN_1408f0074` |
| Version par type + table | `FUN_141102ed0` + `DAT_14474cd90` |
| Vtables domaines +0x58 : 117/103/100·32/30·48/42/43·93/104 | `0x141166E50`/`0x14116CF48`/`0x140EBFEE8`/`0x140EBFF50`/`0x140EBFF6C`/`0x140EBFF20`·`0x140EBFF38`/`0x141173CB0` |
| +0x50 diffusion : commun 1 réf / 2 réfs / 93 conditionnel | `0x141191F20`(0x81) / `0x14119DE10`(0x82) / `0x142EFA3C4`→`FUN_142ef7fc8`→`FUN_142f24ee4` |
| Applicateurs 48/104/42/43 | `0x142F12820`/`0x140BC8170`/`0x142F10C6C`/`0x142EF7794` |
| Lecteurs 30/42/43 | `0x142F16AD8`/`0x142F169D0`/`0x142EF8F04` |

## Annexe B — cartes calibrées des 5 films témoins (entrée map_quant_bounds.json)

`1b2d9e08` et `4577fcc4` et `faff9935` : `944396dd-5661-4a16-b1d8-a6053f762c55` (canevas
fo13_frost, bits [15 15 17] — nom Forge indécidable, R3 §0.4) ; `a0c36016` : `forest`
[13 13 13] ; `f2966f08` : `behemoth` [17 17 15].

## Annexe C — commandes exactes rejouables

### C.1 Instance Ghidra headless (PowerShell ; arrêtée en fin de lot)

```
& "C:\Users\Guillaume\AppData\Local\Programs\Eclipse Adoptium\jdk-25.0.3.9-hotspot\bin\java.exe" `
  -Xmx8g -Djava.system.class.loader=ghidra.GhidraClassLoader `
  -Djava.io.tmpdir=C:\Users\Guillaume\AppData\Local\Temp `
  -Djdk.net.unixdomain.tmpdir=Q:\nexistepas `
  -Dfile.encoding=UTF8 -Xshare:off --enable-native-access=ALL-UNNAMED -Dlog4j.skipJansi=true `
  -cp "C:\Users\Guillaume\Downloads\ghidra_12.1_PUBLIC\Ghidra\Framework\Utility\lib\Utility.jar" `
  ghidra.Ghidra com.xebyte.headless.GhidraMCPHeadlessServer `
  --bind 127.0.0.1 --port 8089 --project C:\Users\Guillaume\Downloads\HI.gpr
# puis : POST /load_program_from_project  body {"path":"/HaloInfinite.exe"}
# requetes : GET /decompile_function?address=0x140f04fb8 · /get_xrefs_to?address=... ·
#            /read_memory?address=...&size=16 (plafond 16 o) · /search_byte_patterns?pattern=...
```

### C.2 Layout 117 (depuis apps/go-api de CE worktree ; <mig> = LevelUp-go-migration)

```
CGO_ENABLED=0 \
  R6_ROOT=<mig>/data/cache/film_chunks \
  R6_ARTS=<mig>/data/cache/replays/halo_infinite \
  R6_CAT=<mig>/data/titles/halo_infinite/reference/map_quant_bounds.json \
  R6_IDS=1b2d9e08,a0c36016,4577fcc4,f2966f08,faff9935 \
  go test ./internal/analysis/filmdec/ -run '^TestR6Layout117$' -count=1 -timeout 20m -v
```

### C.3 Sondage de liste (12 films ; R6_MAPS = les cartes calibrées de C.2)

```
CGO_ENABLED=0 R6_ROOT=... R6_CAT=... (idem C.2) \
  R6_MAPS="1b2d9e08=944396dd-5661-4a16-b1d8-a6053f762c55,a0c36016=forest,4577fcc4=944396dd-5661-4a16-b1d8-a6053f762c55,f2966f08=behemoth,faff9935=944396dd-5661-4a16-b1d8-a6053f762c55" \
  R6_IDS=000d5950,06dfe6d9,084a804d,4f77afc1,8a485699,bf2a9f05,d1dfbc02,1b2d9e08,a0c36016,4577fcc4,f2966f08,faff9935 \
  go test ./internal/analysis/filmdec/ -run '^TestR6SondageListe$' -count=1 -timeout 20m -v
```

Durées mesurées : 0,24 s (C.2) et 0,98 s (C.3), artefacts compris. Aucune source d'aléa.
