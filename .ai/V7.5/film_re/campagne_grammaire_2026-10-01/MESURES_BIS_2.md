# Mesures bis 2 — campagne de grammaire, réponse à la critique de complétude (2026-10-01)

> Répond aux points **24, 25, 26** et à **T6-C2** de `CRITIQUE_COMPLETUDE_1.md`. Worktree
> `LevelUp-wt-campagne-grammaire`, branche `feat/campagne-grammaire`, rien de commité. Aucun fichier
> de production modifié sur disque, `grammar.Rev` inchangé. Aucune base, aucune cuisson, aucun
> backfill. Films en lecture seule : les 20 films du corpus (19 témoins de `config/replay_corpus.toml`
> + `1c4c63c2`) et le témoin utilisateur `81c02726`. Ghidra en lecture seule (`decompile_function`,
> `disassemble_function`, `read_memory`, `get_xrefs_to`). Modules du jeu installé en lecture seule.
>
> Convention : **mesuré** = compté par une sonde sur les films ; **estimé** = dérivé d'une mesure
> par une hypothèse écrite ; **hypothèse** = non mesurée. « Sain » = paquet fermé qui ne contredit
> aucun des trois invariants de l'écrivain (juge de `MESURES_BIS_1.md` §0) ; « contredit » = il en
> contredit au moins un. Sain n'est pas juste.
>
> **Corrections du 2026-10-02** (critique de complétude n° 2, points N3, N18, N21), faites en place et
> marquées « corrigé le 2026-10-02 » : statut de la surcouche (§1.2), « hors cadre » de `81c02726`
> (§5.1), hausse du « hors cadre » de HI_1_12_0 sous T7 (§5.4).

## 0. Résumé chiffré

| Point | Question | Verdict | Chiffre clé (mesuré) |
|---|---|---|---|
| 24 | Fourche `+0x74` du bloc 0xbc | **Non tranchable par la fermeture** : aucune forme ne ferme plus que les témoins décalés d'un bit. Le bloc 0xbc n'apparaît que dans des lectures désalignées. | 1 570 paquets arrêtés par le bloc : fermés `R(96)` 6, `e420` 9, témoins `R(95)` 11, `R(97)` 10 |
| 25 | A/B de position par site (T4) | 2 sites en gain net de paquets (`flock-position`, `tacmap-displayasset`) ; `world-object-i0` +12 338 records d'image-clé mais +31 paquets net seulement ; les autres au niveau du bruit ou en perte (`ti38-i18`, `unit-actor-state`) ; **T4-C3 établi sur Live Fire** | `flock-position` +416 / −10 paquets ; lecture par index de plage sur Live Fire : `0797ce72` +3 269 / −3 paquets (+17,1 %), `60ae07c4` +1 806 / −49 (+12,6 %) |
| T6-C2 | Porte `+0x818` en image-clé `ti=40` | Table « châssis → type de physique » lue dans les modules : Falcon et Wasp = type 6 (vtol), toutes les familles connues cohérentes. La porte levée ferme 63 records non-VTOL, la porte posée 1 ; mais 98 % des records `ti=40` restent non fermés pour une autre cause | 7 059 records bornés : référence 14, porte posée 26, porte levée 136, porte par châssis 137 ; 3 châssis inconnus non identifiables (absents des modules installés) |
| 26 | Témoin `81c02726` | Paquets perdus localisés ; composant bloquant = **`ti=43 i19`** (premier non porté du masque), pas `i20`/`i21`/`i22` ; la grammaire T7 les ferme | Fenêtre de la 3e montée (ticks 2984-3021) : 181 → 218 / 222 paquets fermés, rejets 37 → 0 ; film : 15 191 → 18 771 / 19 933 (+3 580, 1 contredit) |
| (corpus) | Effet de la grammaire T7 sur les 20 films | **T7-5 et T7-6 réfutés** : le port `ti=43` réduit la cause n°1 sur HI_1_13_0 | +18 105 / −12 paquets (462 contredits, dont 382 sur HI_1_10_0) ; HI_1_13_0 « hors cadre » 86 921 → 81 049 |

## 1. Protocole et contrôles

### 1.1 Sondes (tag `research`, aucun fichier de production)

| Fichier | Rôle |
|---|---|
| `film/internal/grammar/campagne_bis2_bc_research_test.go` | point 24 : relecture de la vue C des paquets arrêtés par le bloc 0xbc, bloc porté sous quatre formes ; vecteurs V4/V5 de T5 et un vecteur `flags = 2` construit d'après l'écrivain |
| `film/internal/grammar/campagne_bis2_positions_research_test.go` (tags `research,campagne_overlay`) | point 25 : A/B par site, contextes « instruments » et « production », relevé des index de plage |
| `film/internal/grammar/campagne_bis2_vehicules_research_test.go` (tags `research,campagne_overlay`) | T6-C2 (images-clés `ti=40`) et point 26 (grammaire `ti=43`, fenêtre du témoin) ; juge des invariants sur chaque marche |
| `film/internal/grammar/campagne_bis2_chassis_research_test.go` | T6-C2 : noms MPP (`variant-name`, nom de queue) des châssis, par film |
| `internal/himodule/campagne_bis2_vehi_research_test.go` | T6-C2 : type de physique (`FUN_1408b44fc`) de chaque `vehi` lu dans les modules installés |
| `film/research/cmd_fermeture/bis2_regions_research_test.go` | T4-C3 : bornes des 4 plages de Live Fire (`himap.RegionsBSPExternes`) et tags sbsp candidats des cartes du corpus (`himap.BSPQuantification`) |

### 1.2 La surcouche de recherche (`go test -overlay`)

Les treize sites d'exception, la queue de `waypointstate`, la lecture par index de plage et les
composants non portés `ti=40` / `ti=43` sont appelés par le dispatch des composants, qu'aucune copie
de la marche n'atteint. Trois fichiers de production ont donc une **copie de recherche** dans
`mesures_bis2_overlay/` (`capture.go`, `lecteur_position.go`, `lecteur_position_exceptions.go`),
que `go test -overlay=<json>` substitue au fichier du dépôt **pour la seule compilation du test**.
Le fichier de production sur disque n'est pas modifié. Ajouts de chaque copie :

- `capture.go` : un crochet d'interception par nom de composant (`bis2Intercepteur`) et le nom du
  composant en cours ;
- `lecteur_position_exceptions.go` : une bascule par site vers la **lecture du jeu** (celle que
  J6.3 portait et que J6-bis, R3 et R3-bis ont retirée : diffs `0fc3277b5`, `562e060ba`,
  `26ef60122` ; `world-object-i0`, `flock-position` et `ti38-i18` d'après T4 §1.2-1.3 et
  `FUN_14076e29c` / `FUN_14076e3e4` relus dans Ghidra ce jour) ; la queue `R(1)` de
  `waypointstate` en bascule séparée ;
- `lecteur_position.go` : le relevé de chaque index de plage lu (composant, archétype, index, région
  du profil) et, sur demande, les largeurs de la ligne de l'index lu quand les bornes de sa plage
  sont fournies.

Sans bascule, chaque copie lit comme la production. Fichier JSON : chemins absolus des trois
fichiers vers leurs copies.

Statut (corrigé le 2026-10-02, PLAN §6.3 D10) : outillage de MESURE seulement, jamais preuve de
gate. Les fichiers taggés `research && campagne_overlay` sont HORS CI : le
`go vet -tags=research ./...` de la CI ne les compile pas. Les copies sont des fichiers ENTIERS,
pris sur la tête `69564ef7d` : J12 modifie `lecteur_position.go` (+1/−1) et
`lecteur_position_exceptions.go` (+4/−4). Elles sont à re-synchroniser depuis les fichiers post-J12
à la fusion de J12, faute de quoi une mesure annulerait ces changements.

### 1.3 Contrôles (mesurés, tous tenus)

- Marche de référence des sondes = carte v2 : 629 142 paquets, 284 704 fermés, 2 588 167 / 5 961 028
  records utiles fermés (instruments, 20 films). Fermés contredits de la référence : 8 388
  (= MESURES_CIBLEES §T2-4).
- `81c02726` : la référence de la sonde (15 191 / 19 933, 123 544 / 154 600 utiles) est identique à
  `cmd_fermeture -mode v2` lancé sur ce film (`carte_v2_81c02726_*.tsv`).
- Contexte de production : référence identique au contexte des instruments sur 17 films ; écart
  sur Live Fire seul (`0797ce72` −81 paquets, `60ae07c4` −4 : la région jouée vaut 1 au catalogue,
  0 dans le contexte des instruments).
- Un film à la fois, sentinelle `filmproc` 4 Gio ; pics 49 à 273 Mio. Durées : point 24 99 s ; A/B
  positions (17 marches par film + fermeture d'image-clé) 1 471 s ; production 208 s ; `ti=43`
  191 s ; images-clés `ti=40` 14 s.

Sorties : `mesures_bis2_tsv/` (27 fichiers, une ligne par film et par clé).

---

## 2. Point 24 — la fourche `+0x74` du bloc 0xbc (T5-3)

### 2.1 Grammaire relue (Ghidra, ce jour)

`FUN_141fdae44(lecteur, bloc, param_3)` décompilé et désassemblé :

- appelé par la vue C avec `param_3 = 1` (`FUN_1406d0388`, octets `41 b0 01` = `MOV R8B,1` avant le
  `CALL` en `1422f5111`) ;
- `R(5)` flags ; `R(2)` (forme courte) ; `R(17)` lacet (`[RSP+0x20] = 0x11` en `141fdaf9c`) ; `R(16)`
  tangage (`0x10` en `141fdafca`) ; `R(1)` puis varint cat. 1 (`MOV R8D,0x1` en `141fdaff6`) ;
- **tout le reste est sous `flags & 2`** (`TEST byte [RSI],0x2` en `141fdb01c`) :
  `FUN_142f2e63c(cat 1)` = `R(1)` [varint cat. 1] ; `R(2)` ; `R(3)` ; `c = R(3)` puis `c` fois
  { `R(1)` [varint cat. 0] ; `R(4)` ; `R(4)` } (`FUN_142f26754`, `FUN_142ed0674`, `FUN_14101d200`) ;
  `R(5)` ; `FUN_1406d01fc` ; `R(6) R(3)` ; `FUN_1406d0ff0` ; **la fourche** ; `FUN_140c1e79c` ;
  `FUN_14076d528(0x13, 0xa)` et `(0x13, 0x8)` (`[RSP+0x30] = 0x13`, `[RSP+0x28] = 0xa / 0x8`) ;
  `FUN_142f0ec48` = `FUN_1408f0ac4(cat 0)` puis `+0xb8 = !présent` et `R(11)` si présent ; `R(7)` ;
- la fourche (`141fdb0c8`) : `FUN_1404f293c() == 0 && param_3` → `FUN_14076e420(…, 0x10)` ; sinon
  `FUN_1406d676c(…, 0x60)` (`MOV R9D,0x60` en `141fdb102`). `FUN_1404f293c` rend `état != 2` d'un
  objet de session lu par TLS : non résoluble statiquement (comme T5 l'avait dit).

### 2.2 Méthode

Pour chaque paquet dont la vue C s'arrête sur le bit `b` d'une entrée (`ArretVueCBlocBC`), la vue C
est relue depuis la fin de la vue B avec le bloc porté, sous quatre formes : `r96`, `e420`, et deux
**témoins de hasard** `r95`, `r97` (la forme `r96` à un bit près). La vue C ne lie aucune entité :
le monde n'en dépend pas, la relecture suffit (pas de nouvelle marche).

### 2.3 Résultat (mesuré, `mb2_bc.tsv`, `mb2_bc_paquets.tsv`)

| Build | Paquets arrêtés par le bloc | dont après un rejet | Fermés `r96` | Fermés `e420` | Témoin `r95` | Témoin `r97` |
|---|---|---|---|---|---|---|
| HI_1_13_0 | 388 | 240 | 3 | 6 | 4 | 3 |
| HI_1_12_0 | 8 | 7 | 0 | 0 | 0 | 0 |
| HI_1_11_0 | 88 | 85 | 1 | 1 | 1 | 1 |
| HI_1_10_0 | 80 | 28 | 0 | 0 | 0 | 0 |
| HI_1_9_0 | 18 | 11 | 0 | 0 | 0 | 0 |
| HI_1_8_0 | 145 | 116 | 2 | 2 | 6 | 6 |
| HI_1_4_1 | 139 | 50 | 0 | 0 | 0 | 0 |
| version-31 | 552 | 4 | 0 | 0 | 0 | 0 |
| version-33 | 152 | 76 | 0 | 0 | 0 | 0 |
| **corpus** | **1 570** | 617 | **6** (27 utiles) | **9** (53 utiles) | **11** | **10** |

- Fermés par une forme et pas par l'autre : `r96` 3, `e420` 6 (tous HI_1_13_0 et HI_1_8_0).
- Paquets « fermés » dont un bloc est invalide chez l'écrivain (`flags & 0x1f == 0` sans
  `flags & 2`, ou tangage hors `[16384 ; 49152]` sous `flags & 1`) : `r96` 3 sur 6, `e420` 5 sur 9.
- Deux paquets ferment sous les quatre formes : leur bloc n'a pas `flags & 2`, la fourche n'y est
  pas lue (`60ae07c4` 1:680, `e5adf7b2` 1:354, 0 utile chacun).

### 2.4 Verdict

- **Mesuré** : aucune des deux formes ne ferme plus de paquets que les témoins décalés d'un bit
  (6 et 9 contre 11 et 10 sur 1 570). Le taux de fermeture des quatre formes est celui du hasard
  (≤ 0,7 %), et la moitié environ des paquets « fermés » portent un bloc invalide chez l'écrivain.
- **La fourche n'est pas tranchable par la fermeture sur ce corpus** : le corpus ne contient
  aucun bloc 0xbc lu au bon bit. Les 1 570 arrêts « bloc 0xbc » sont des marqueurs de
  désalignement, de la même nature que les kinds 1/2/3 (T5-1) : la vue C est lue au mauvais endroit
  et le bit `b` y vaut 1 par hasard.
- La borne de T5-3 (« ≤ 953 paquets, ≤ 19 378 utiles ») est réfutée : gain mesuré du portage C-b
  ≤ 9 paquets et 53 utiles, au niveau des témoins.
- Conséquence pour L5 (décision au superviseur, non prise ici) : porter le bloc ne ferme rien ;
  requalifier `ArretVueCBlocBC` comme désalignement (comme C-a pour les kinds) coûte zéro paquet.
  La fourche reste une question ouverte du jeu (rôle de `FUN_1404f293c` en relecture Theater).

---

## 3. Point 25 — A/B de position (T4)

### 3.1 A/B par site, contexte des instruments (mesuré, `mb2_positions.tsv`, `mb2_agg_positions_site_build.tsv`)

Chaque site seul passe à la lecture du jeu ; marche complète par film (le monde dépend des
paquets précédents) ; « image-clé » = records d'image-clé fermés (`KeyframeClosure`, tous
archétypes). Δ utiles = variation exacte des records utiles fermés.

| Site (lecture du jeu) | Paquets gagnés / perdus (net) | Δ utiles fermés | Δ image-clé | Builds en baisse nette | Builds en hausse nette |
|---|---|---|---|---|---|
| `flock-position` | +416 / −10 (+406) | +3 318 | 0 | HI_1_9_0 −1 | HI_1_11_0 +293, HI_1_4_1 +64, HI_1_12_0 +24, HI_1_8_0 +10, version-33 +10, HI_1_13_0 +4, HI_1_10_0 +2 |
| `tacmap-displayasset` | +77 / −16 (+61) | +1 873 | 0 | HI_1_10_0 −1 | HI_1_13_0 +60 (`4f77afc1` +66, `51ebbc0f` −10), HI_1_8_0 +1, HI_1_9_0 +1 |
| `world-object-i0` | +654 / −623 (+31) | +3 017 | **+12 338** | HI_1_10_0 −64, HI_1_8_0 −34, HI_1_13_0 −19, HI_1_11_0 −10 | HI_1_9_0 +154, version-33 +3, HI_1_4_1 +1 |
| `tacmap-areaofinterest` | +12 / −2 (+10) | +5 | 0 | — | 5 builds, +1 à +4 |
| `respawn-location` | +10 / −9 (+1) | +6 | 0 | — | HI_1_13_0 +1 |
| `i0-bipede-prechigh` | +2 / −3 (−1) | +13 | −1 | HI_1_10_0 −2 | HI_1_13_0 +1 |
| `tacmap-cooptetherarea` | 0 / −1 (−1) | 0 | 0 | HI_1_13_0 −1 (`c75f33b8`) | — |
| `crew-order` | +3 / −5 (−2) | +5 | 0 | HI_1_13_0 −3 | HI_1_11_0 +1 |
| `tacmap-waypointstate` (position + queue) | 0 / −2 (−2) | −3 | 0 | HI_1_10_0 −1, HI_1_13_0 −1 (`d9781168` 34:336) | — |
| `tacmap-poiicon` | +3 / −6 (−3) | 0 | 0 | HI_1_10_0 −3, HI_1_9_0 −1 | HI_1_13_0 +1 |
| `flock-destination` | +12 / −16 (−4) | −1 | 0 | HI_1_10_0 −3, HI_1_9_0 −2, version-33 −1 | HI_1_13_0 +2 |
| `ti38-i18` | +156 / −170 (−14) | −24 | **−441** | HI_1_10_0 −15, HI_1_11_0 −4, HI_1_8_0 −3, HI_1_9_0 −2, version-31 −1 | HI_1_13_0 +9, HI_1_12_0 +1, version-33 +1 |
| `unit-actor-state` | +18 / −33 (−15) | −261 | −3 | HI_1_10_0 −11, HI_1_13_0 −6 (`4f77afc1` −8) | HI_1_11_0 +2 |
| **les treize** | +1 315 / −802 (+513) | +7 971 | +12 012 | HI_1_10_0 −113, version-31 −1 | HI_1_11_0 +283, HI_1_9_0 +155, HI_1_4_1 +69, HI_1_13_0 +63, HI_1_12_0 +32, version-33 +17, HI_1_8_0 +8 |

Détail image-clé de `world-object-i0` (mesuré) : HI_1_13_0 +10 537, HI_1_10_0 +823, HI_1_12_0
+644, HI_1_9_0 +274, HI_1_11_0 +205, HI_1_8_0 +143, version-33 +42, HI_1_4_1 +34, **version-31
−364**. `ti38-i18` : HI_1_10_0 −345, HI_1_13_0 −96 (image-clé `ti=38`).

Lecture (mesuré) :

- Les sites ne sont pas indépendants : les treize ensemble (+513 net) ne font pas la somme des
  sites seuls (+467 net).
- `flock-position` est le site au plus fort gain net (+406), ses 10 pertes sur trois films
  (`60ae07c4` −8, `11de8353` −1, `1c4c63c2` −1) ; `tacmap-displayasset` gagne +60 sur HI_1_13_0 malgré les −10 de
  `51ebbc0f` déjà connus (R3).
- `world-object-i0` : la lecture du jeu ferme +12 338 records d'image-clé (+9,1 % des 135 110 records
  d'image-clé fermés en référence, tous archétypes) mais rend 623
  paquets delta ; son solde delta est nul sur HI_1_13_0 (−19) et négatif sur HI_1_10_0 / HI_1_8_0.
- Le contexte des instruments ne départage pas « compensation » et « grammaire » : un paquet perdu
  peut être une fermeture factice de la référence (le juge n'a pas été joué sur ces A/B, cf. §6).

### 3.2 T4-C2 : queues écrites sans condition (mesuré)

- `tacmap-waypointstate` : position du jeu seule 0 / −1 net ; queue `R(1)` seule (ancienne
  position) −1 net ; les deux −2 net. La liste `d9781168` 34:336 est perdue dans les trois cas
  (−3 utiles) : sur ce corpus, aucun des deux écarts seul ne la conserve, contrairement à la
  prédiction de T4 §4 C2 (« il faut les deux écarts »).
- `flock-destination` : lire la queue `R(2)` sans condition est **identique par construction** à
  la lecture actuelle (`level > 1`) : le niveau du registre vaut 2 pour `ti=21 i2..i11` sur les 20
  films (MESURES_CIBLEES T4-C2). Aucune marche supplémentaire n'était nécessaire.

### 3.3 T4-C3 : largeurs par index de plage, sous le contexte de production

**Contexte de production** : `NewFilmContextForMap(film, entrée du catalogue)` puis la pose des
largeurs de `replay.installWorldObjectPrecision` ; carte de chaque témoin d'après
`config/replay_corpus.toml` (`1c4c63c2` exclu : carte inconnue sans base). Le profil que
`killsource` calibre n'est pas posé (limite).

**Bornes des plages (lues dans les modules installés, HI_1_13_0, `mb2_regions_live_fire.tsv`,
`mb2_sbsp_candidats.tsv`)** :

- Live Fire (`sgh_interlock`, 4 plages, ordre du `levl`) : plage 0 `[12 12 13]`, **plage 1 = la
  plage jouée du catalogue** `[12 12 11]` (bornes identiques au catalogue), plage 2 `[13 13 13]`,
  plage 3 `[19 19 16]` (−3 500 à 4 390 m : la plage lointaine).
- Fragmentation (`btb_fragmentation`) et Illusion (`ctf_illusion`) : **un seul tag sbsp**.
- Toutes les autres cartes du corpus : **deux tags sbsp** (la plage jouée et une plage lointaine
  `[18 18 18]` pour les toiles Forge, `[19 20 16]` Cliffhanger, `[18 18 16]` Oasis). L'ordre des
  plages de ces cartes n'a pas été lu ; « plage 1 = le second sbsp » est une **hypothèse**.

**Index de plage lus, référence, contexte de production (mesuré, `mb2_index_de_plage.tsv`)** :

| Film | Build | idx −1 (porte) | idx 0 | idx 1 | idx 2 | idx 3 | part ≠ plage jouée (hors porte) |
|---|---|---|---|---|---|---|---|
| `0797ce72` Live Fire | HI_1_13_0 | 254 | 321 | **150 243** | 78 | 4 203 | 3,0 % |
| `60ae07c4` Live Fire | HI_1_8_0 | 1 203 | 694 | **279 964** | 311 | 4 281 | 1,9 % |
| `396cfc92` Illusion (1 sbsp) | HI_1_13_0 | 387 | 180 464 | 800 | — | — | 0,44 % |
| `e5adf7b2` Fragmentation (1 sbsp) | HI_1_11_0 | 7 300 | 319 944 | 3 552 | — | — | 1,1 % |
| `a349fea8` / `a521164d` Fragmentation Heavies (1 sbsp) | version-33 / HI_1_4_1 | 16 063 / 5 772 | 523 268 / 191 347 | 9 885 / 3 544 | — | — | 1,9 % / 1,8 % |
| 2 sbsp, HI_1_13_0 (8 films) | | | | 0,07 % à 1,1 % des index | | | |

Sur Live Fire, l'index 3 est lu à 99 % par `ti=40 i0` (`0797ce72` : 4 155 des 4 203 lectures) : des
véhicules hors de l'arène, lus aujourd'hui aux largeurs de la plage 1.

**Lecture par index de plage (A/B, mesuré, `mb2_positions_production.tsv`)** :

| Film | Variante | Paquets fermés | Gagnés / perdus | Δ utiles | Fermés contredits | Gagnés contredits |
|---|---|---|---|---|---|---|
| `0797ce72` HI_1_13_0 | référence | 19 124 / 26 740 | — | — | 36 | — |
| | + lecture par index | **22 390** | **+3 269 / −3** | **+26 664** | 22 | 3 |
| | 13 sites jeu + lecture par index | 23 061 | +3 942 / −5 | +32 590 | 20 | 3 |
| `60ae07c4` HI_1_8_0 | référence | 13 965 / 49 696 | — | — | 137 | — |
| | + lecture par index | **15 722** | **+1 806 / −49** | **+11 052** | 104 | 4 |
| | 13 sites jeu + lecture par index | 15 941 | +2 045 / −69 | +11 105 | 100 | 21 |

Taux de fermeture des paquets qui lisent l'index 3 sur `0797ce72` : 3,3 % → **82,0 %** (même taux
que l'index 1) ; sur `60ae07c4` : 5,2 % → 46,5 %.

Cartes à deux sbsp, plage 1 = second sbsp (hypothèse) : référence + lecture par index ±0 à ±4
paquets sur tous les films (rien) ; avec les treize sites au jeu, `4f77afc1` +385 / −99 contre
+155 / −90 sans, `bcb6d393` +118 contre +32 — mais seulement par l'effet des sites d'exception, qui
passent alors par `FUN_14076e524`. Non concluant pour l'ordre des plages de ces cartes.

Cartes à un seul sbsp (index ≠ 0 impossible si « un sbsp = une plage ») : paquets **fermés** qui
lisent un index 1, référence : Illusion 101, Fragmentation 136 / 25 / 2 (mb2_index_impossibles.tsv).
Soit la prémisse est fausse (la carte déclare une plage de plus, portée ailleurs, comme Live Fire),
soit ces fermetures sont factices. Non tranché.

**Verdict T4-C3** : **établi sur Live Fire** (mesuré, deux builds) : lire la ligne de l'index lu
ferme +3 269 paquets (net +3 266, +17,1 %) sur `0797ce72` et +1 806 (net +1 757, +12,6 %) sur `60ae07c4`, avec moins de
fermetures contredites qu'avant. Il exige les bornes de chaque plage déclarée (extension du
catalogue, donnée de carte). Ailleurs, l'effet est nul ou non mesurable sans l'ordre des plages.

### 3.4 Verdicts par site (ce que la mesure permet de dire)

Sur HI_1_13_0 (gagnés / perdus, mesuré) :

- Aucun paquet perdu : `flock-position` (+4 / 0), `tacmap-poiicon` (+1 / 0).
- Gain net avec des pertes : `tacmap-displayasset` (+72 / −12), `ti38-i18` (+34 / −25, mais −96
  records d'image-clé `ti=38`), `flock-destination` (+7 / −5), `tacmap-areaofinterest` (+4 / −1),
  `respawn-location` (+2 / −1), `i0-bipede-prechigh` (+2 / −1).
- Perte nette : `world-object-i0` (+107 / −126, mais +10 537 records d'image-clé),
  `unit-actor-state` (+2 / −8, −174 utiles), `crew-order` (+1 / −4), `tacmap-waypointstate`
  (0 / −1), `tacmap-cooptetherarea` (0 / −1).

Le critère de retrait écrit (« monter sans aucune baisse sur les bobines ») n'est rempli par aucun
site sur l'ensemble des builds. Appliqué par build, il ne retirerait sur HI_1_13_0 que
`flock-position` et `tacmap-poiicon`. Aucune de ces mesures ne dit si un paquet perdu était une
fermeture juste ou une compensation : le juge des invariants n'a pas été joué sur ces A/B (§6).

---

## 4. T6-C2 — images-clés `ti=40` et porte `+0x818`

### 4.1 Le type de physique de chaque châssis, lu dans les modules (mesuré, `mb2_vehi_type_physique.tsv`)

`FUN_1408b44fc` relu (désassemblage) : douze blocs `vehi + 0xde0 + 0x14·k`, premier compte
(`+0x10`) non nul. Lu dans la MainStruct de chaque tag `vehi` (4 400 octets) des modules globals
installés :

| Type | Châssis (famille de `vehicle_families.go` / `vehicle_turrets.go`) |
|---|---|
| 0 `human_tank` | `0000d3db`, `f6f54e56` (Scorpion) |
| 1 `human_jeep` | `00002705`, `fe32c0f4`, `cb96ca07`, `5159c8ef`, `75312e51`, `7617ff6e` (Warthog) ; `000025aa`, `af31ab1a`, `de26e3d7` (Mongoose) |
| 2 `human_plane` | `000026f0` (Pelican), `000026f2` (Phantom) |
| 3 `alien_scout` | `0000d3dc`, `5b80c406`, `9af9e693` (Ghost) ; `00002706`, `ae845375` (Wraith) ; `86799cb6` (Skiff) |
| 4 `alien_fighter` | `000026ed`, `c6e79dcc`, `0001530a` (Banshee) |
| 5 `turret` | tourelles et pièces montées (`038df01a`, `3a8060e2`, `000df0c4`, `dd7f9102`, `0000d4ff`, `0000d500`, `64b925eb`, `bcfb852f`, `1a043c29`, `233c877d`, `001b33fc`, `f4c45d71`) |
| **6 `vtol`** | **`0000254b` (Falcon), `b65b3b4a` (Wasp)** |
| 7 `chopper` | `002ba902`, `3d4a8a5a` (Chopper) |

Toutes les familles connues tombent dans le type attendu : la table des noms de T6 §2.3 et la
disposition du bloc sont confirmées par les données. **« Type 6 = vtol » et « Falcon = vtol » sont
établis** (mesuré dans les tags, plus seulement probables).

### 4.2 Effet sur la fermeture des records d'image-clé (mesuré, `mb2_ti40_*.tsv`)

Copie de recherche : les 16 composants `ti=40` non portés lus par le crochet (grammaires de T6 §4),
`i33`/`i34` sous une porte posée ou levée, le `MPPWord32` du même record relevé par
l'observateur. 21 films (corpus + `81c02726`), 7 059 records `ti=40` bornés.

| Variante | Records fermés |
|---|---|
| Référence (arrêt sur `i30`, non porté) | 14 |
| Composants portés, porte posée partout (le repli actuel, étendu à l'image-clé) | 26 |
| Composants portés, porte levée partout | 136 |
| Porte par châssis (table §4.1 ; châssis inconnu → levée) | 137 |
| Porte par châssis (châssis inconnu → posée) | 88 |
| L'une ou l'autre (oracle) | 162 |

Par classe de châssis :

| Classe | Records | Ferme porte posée seule | Ferme porte levée seule | Les deux | Aucune |
|---|---|---|---|---|---|
| connu, non VTOL | 1 384 | 1 | **63** | 0 | 1 320 |
| connu, VTOL (Falcon, `4f77afc1`) | 73 | 2 | 1 | 0 | 70 |
| inconnu (builds anciens, lectures fausses) | 5 602 | 9 | 58 | 14 | 5 521 |

Lecture :

- **Mesuré** : sur les non-VTOL, la porte levée ferme 63 records et la porte posée 1 : le repli
  « porte supposée posée » serait faux en image-clé, comme T6 §3.1 le prédisait.
- **Mesuré** : la porte n'est pas le bloquant principal. Avec les 16 composants portés et la bonne
  porte, 98 % des records restent non fermés : 4 479 s'arrêtent avant leur frontière (« sous ») et
  2 444 la dépassent (« sur »), sans désynchronisation nommée. Les pièces montées (type 5, porte
  levée par la table) dépassent toutes leur frontière (`4f77afc1` : 326 sur 326). Une autre largeur
  de l'archétype `ti=40` est fausse en image-clé ; elle n'est pas identifiée ici (candidats
  écrits par T6 : `i43 simulation-state` « partiel », `i2` « partiel », l'état par défaut).
- Sur le Falcon seul (73 records), la mesure ne départage pas (2 contre 1).

### 4.3 Identification des châssis `77ef810a`, `4118381d`, `d0b40d0a`

Méthodes essayées, dans l'ordre :

1. **Modules installés** (`campagne_bis2_vehi_research_test.go`, globals `any`/`pc`/`ds` et
   niveaux `multi`) : les trois GlobalID sont **absents** (comme `10754375`, un Wraith de la table).
   Ce sont des identifiants de modules d'anciens builds, qu'aucun fichier installé ne porte.
2. **Registre du dépôt** (`damagetag/data/labels.tsv`, `vehicle_families.go`, `vehicle_turrets.go`,
   documents `.ai`) : aucune occurrence.
3. **Ghidra** : sans objet ; un GlobalID de tag est une donnée de module, pas du code.
4. **Données de création des films** (`campagne_bis2_chassis_research_test.go`,
   `mb2_chassis_noms.tsv`) : `4118381d` n'apparaît qu'une fois, dans un record NEW de `084a804d`
   (z = −10,9 m, `variant-name` `e8e2ffd3`) ; `d0b40d0a` une fois, NEW de `50247b26` (z = 12,4 m,
   `variant-name` 0) ; `77ef810a` aucune création lue sur le corpus. Le `variant-name` ne discrimine
   pas les familles (`42c9679f` est partagé par Falcon, Mongoose et Warthog sur `4f77afc1`).

**Non identifiés.** Ce que la mesure permet de dire : ce sont des châssis à une seule création
lue, jamais présents en image-clé ; sur les 21 films, 459 valeurs de « châssis » distinctes sont
lues (435 dans des records NEW, la plupart une seule fois), bien plus que de véhicules réels : la
plupart sortent de NEW que T2-4 qualifie de lectures fausses. Une lecture fausse du NEW est
l'explication la plus économique (**hypothèse**). Par la loi de l'écrivain (`FUN_142f09c74`),
les DELTA qui annoncent `i33`/`i34` sur ces slots (T6-C1 : 457, 373, 2 523) prouvent la porte du
véhicule du slot, quel que soit le châssis lu.

---

## 5. Point 26 — le témoin utilisateur `81c02726`

### 5.1 La carte v2 sur ce film (mesuré, `carte_v2_81c02726_*.tsv`)

`cmd_fermeture -mode v2` : 15 191 / 19 933 paquets fermés, 123 544 / 154 600 utiles ; vue B
sortie par rejet hors datum sur 4 152 paquets, dont 4 149 « vue C : terminateur hors cadre » (le
« hors cadre » du film vaut **4 150** en tout : ces 4 149, plus 1 après une sortie par terminateur,
`carte_v2_81c02726_fermeture_sorties_vueB.tsv` ; c'est le 4 150 du §5.3 ; corrigé le 2026-10-02) ;
ventilation des rejets : « naissance non lue » 3 560 paquets (28 537 utiles), « non mesurable »
586. Causes nommées `ti=43` : 14 paquets (`i39` principal).

### 5.2 Où sont les paquets perdus, et pourquoi (mesuré, `mb2_ti43_fenetre_81c02726_725-775s.tsv`)

- Horloge : `t_tick = (ts − 440,57 s) × 10 − 105,59` (premier paquet du film à 440,57 s,
  `originMs` 10 559, 100 ms par tick, `81c02726.json`). Tick 2941 = ts 745,2 s.
- Les 71 records NEW `ti=43` du film portent **tous** le masque {`i19`, `i21`, `i22`, `i23`, `i34`,
  `i35`} ; aucun ne porte `i20`. Le premier composant non porté rencontré est donc **`i19`
  `device-position-animation-name`**. En référence, aucun des 71 n'apparaît dans les records rendus par la marche (la traversée
  désynchronise, le slot n'est pas lié).
- Au tick 2941 (ts 745,31 s, paquet 16:568) naît le dispositif de slot 2308 ; à 742,21 s (16:196) le
  2306, à 748,61 s (16:964) le 2319. Leurs DELTA suivants sont **rejetés hors datum** (le slot n'est
  pas lié) : entre 740 et 766 s, eid `40000904` 74 paquets, `40000902` 68, `4000090f` 56 ; chaque
  rejet clôt la vue B et fait lire la vue C au mauvais endroit (« hors cadre »). C'est exactement le
  mécanisme de T7 §4.

| Fenêtre (ticks) | Paquets | Fermés, référence | Fermés, grammaire T7 | Rejets, référence → T7 |
|---|---|---|---|---|
| 2930-2983 (avant la montée) | 318 | 222 (69,8 %) | 316 (99,4 %) | 94 → 0 |
| **2984-3021 (3e montée)** | 222 | 181 (81,5 %) | **218 (98,2 %)** | 37 → 0 |
| 3022-3103 | 485 | 458 (94,4 %) | 474 (97,7 %) | 15 → 0 |
| 3104-3119 (3e montée, fin) | 90 | 89 | 89 | 0 → 0 |
| 3120-3169 (fin du film) | 294 | 68 (23,1 %) | 68 (23,1 %) | 225 → 225 |

- Résidu de fin de film, non lié à `ti=43` : à partir de 764,26 s (après un DEL en 17:440), l'eid
  `40000934` (slot 2356) est rejeté 362 fois ; aucun NEW de ce slot n'est lu (naissance non lue, T1).

### 5.3 La grammaire T7 les ferme-t-elle ? (mesuré)

Oui. Copie de recherche : les 22 composants `device-*` de T7 §5 lus par le crochet (dont `i31` qui
déclare une désynchronisation au-delà de 8 moniteurs). Sur `81c02726` : les 71 NEW `ti=43` traversent
proprement (71 sur 71) ; **15 191 → 18 771 paquets fermés (+3 580, 0 perdu, 1 contredit)**, utiles
fermés 123 544 → 157 384, « hors cadre » 4 150 → 597, plus aucune cause `ti=43`.

Ce qui n'est **pas** mesuré : que la montée soit publiée dans le document (`noTrack` 2 → 0) ; cela
exige une cuisson (interdite ici). Le relevé de la porte « attaché » d'`object-parent-state` ne
montre aucune montée vers les slots de Ghost (769, 771) ni en référence ni avec T7, y compris pour
les deux premières montées que la production publie : ce canal n'est pas celui de la montée, il ne
sert pas de témoin.

**Correction des documents antérieurs** : `HANDOFF_RETOURS_REJEU_2026-09-25.md` §3.2 (« i20, i21,
i22 ») et `PLAN_RETOURS_REJEU_2026-09-23.md` l. 919 (« désynchronise sur i21 ») : sur la tête du
worktree, le NEW désynchronise sur `i19` (mesuré).

### 5.4 Effet de la grammaire T7 sur le corpus (mesuré, `mb2_ti43.tsv`)

| Build | Paquets fermés réf. → T7 | Gagnés / perdus | Gagnés contredits | Δ utiles fermés | « Hors cadre » réf. → T7 |
|---|---|---|---|---|---|
| HI_1_13_0 | 224 818 → 231 182 | +6 364 / 0 | 35 | +62 716 | 86 921 → 81 049 |
| HI_1_12_0 | 5 835 → 17 132 | +11 297 / 0 | 4 | +80 071 | 3 249 → 4 029 |
| HI_1_10_0 | 28 741 → 29 130 | +401 / −12 | **382** | +451 | 79 964 → 79 967 |
| HI_1_11_0, HI_1_9_0, HI_1_8_0, version-31 | | +11, +30, +1, +1 | 11, 29, 1, 0 | | |
| HI_1_4_1, version-33 | | 0 | 0 | | |
| **corpus (20 films)** | 284 704 → 302 797 | **+18 105 / −12** | 462 | **+143 314** | 264 757 → 259 883 |

- HI_1_13_0 : `396cfc92` +4 087 (« hors cadre » 8 362 → 4 320), `f75e7053` +1 623, `4f77afc1` +501,
  `fb1a1a72` +118 ; aucun perdu ; 35 gagnés contredits sur 6 364 (0,5 %).
- HI_1_10_0 : 382 des 401 gagnés contredisent un invariant de l'écrivain : sur ce build, le gain
  est factice (même constat que pour les fermetures factices de MESURES_CIBLEES §4.1).
- **HI_1_12_0 : le « hors cadre » MONTE (3 249 → 4 029) alors que les fermés triplent** (ajouté le
  2026-10-02, mesuré, `mb2_ti43_causes.tsv`, `bcb6d393`, seul film du build). C'est un déplacement
  de cause bloquante : la carte range un paquet sous son PREMIER bloquant. En référence, 12 127
  paquets sont bloqués par `ti=43` (dont `i35` 12 112), et 283 listes sont non localisées. Sous T7,
  plus aucun n'est bloqué par `ti=43` et 262 listes restent non localisées. Les 12 148 paquets ainsi
  libérés se répartissent en : 11 297 fermés ; **780 qui avancent jusqu'à la vue C et y sont arrêtés
  hors cadre** ; 71 qui butent sur `ti=12 i21` (+40), `ti=12 i22` (+26), `ti=10 i2` (+2),
  `ti=11 i4`, `ti=12 i16` et `ti=12 i18` (+1 chacun). Aucun paquet n'est perdu (+11 297 / 0).
- **T7-5 et T7-6 sont réfutés sur HI_1_13_0** : le lien « NEW `ti=43` désynchronisé → rejet hors
  datum → hors cadre » ne vaut pas « 3 eid » ; le port retire 5 872 paquets « hors cadre » de
  HI_1_13_0 (6,8 % de la cause n°1 de ce build). La mesure de T7-5 cherchait un NEW désynchronisé
  « lu » ; ces NEW ne figurent pas dans les records rendus par la marche (le paquet se ferme sans
  eux), d'où son « 3 eid ».
- La borne de T7-1 (« ≤ 12 939 paquets ») ne bornait que les causes nommées ; l'effet mesuré la
  dépasse de 5 166 paquets par la voie indirecte.

---

## 6. Limites

- Surcouche : la lecture du jeu de `world-object-i0` sous `precHigh = 1` (queue de poignée
  `FUN_14076e3e4` = `FUN_1408f0ac4(cat 0)` + `R(1)` [+ `R(11)`]) est relue au décompilé, pas
  vérifiée au bit sur un vecteur.
- Le juge des invariants n'a été joué que sur les marches `ti=43` et sur le contexte de production
  (§3.3, §5.4), pas sur les 17 A/B par site du contexte des instruments.
- Ordre des plages des cartes à deux sbsp : non lu (hypothèse « plage 1 = second sbsp »).
- Table des types de physique : lue dans l'installation HI_1_13_0 ; appliquée aux seuls châssis
  qu'elle contient (tous ceux des films HI_1_13_0) ; aucun châssis des builds anciens n'y figure.
- `1c4c63c2` : exclu du contexte de production (carte non connue sans base).
- Rien n'est publié : aucune mesure ne dit l'effet sur le document de rejeu.

## 7. Découvertes (consignées, non traitées)

1. **Le port `ti=43` (L2) est un levier de la cause n°1 sur HI_1_13_0** (−5 872 « hors cadre »,
   +6 364 paquets, 0 perdu) : à reclasser dans le plan de phase 2 (§6.1) et dans le RAPPORT (T7-5,
   T7-6). Son gate doit écarter HI_1_10_0 ou y juger les fermetures (382 / 401 factices).
2. **La carte v2 range les naissances lues mais désynchronisées en « naissance non lue »** (3 560
   paquets sur `81c02726`) : la ventilation des rejets (CARTE §4, M1) ne distingue pas un NEW
   traversé en échec d'un NEW absent. Les chiffres de la région (iii) et de l'oracle-NEW en
   dépendent.
3. **Live Fire, lecture par index de plage** : +3 269 et +1 806 paquets, le plus gros gain par film
   mesuré dans cette campagne hors `bcb6d393` ; exige d'étendre le catalogue aux bornes de chaque
   plage déclarée.
4. Cartes à un sbsp (Illusion, Fragmentation) : 101 et 136 paquets fermés lisent un index 1.
5. Images-clés `ti=40` : 98 % des records restent non fermés avec la bonne porte ; les pièces
   montées dépassent toutes leur frontière.
6. Bloc 0xbc : jamais lu au bon bit sur le corpus ; les 1 570 arrêts sont des désalignements.
7. `world-object-i0` au jeu : +12 338 records d'image-clé, sauf version-31 (−364).

## 8. Gate de la sonde

Joué le 2026-10-01 depuis `apps/go-api` (GOCACHE dédié, CGO) :

- `go vet -tags=research` sur `film/internal/grammar`, `internal/himodule`,
  `film/research/cmd_fermeture` : aucun message.
- `go vet -tags=research,campagne_overlay -overlay=mesures_bis2_overlay/overlay.json` sur
  `film/internal/grammar` : aucun message.
- `go test -tags=research -run 'TestCampagneBis2|TestBis2'` (trois paquets) : `ok` ;
  `TestCampagneBis2BlocBCVecteurs` PASS (V4 fermé, 1 entrée, 1 bloc valide ; V5 bloc invalide ;
  bloc `flags = 2` construit d'après l'écrivain lu sur 188 bits) ; les tests sur films sont sautés
  sans leurs variables d'environnement.
- Même commande sous `-tags=research,campagne_overlay -overlay` : `ok`.
- `git status` : aucun fichier suivi modifié de plus qu'au début de la mission ; `grammar.Rev`
  = `grammar-2026-09-27.3`.
