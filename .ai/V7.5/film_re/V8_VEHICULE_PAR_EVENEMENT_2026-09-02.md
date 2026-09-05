# RAPPORT — lot V8 : LE VEHICULE D'UN EPISODE D'OCCUPATION EST RESOLU PAR L'EVENEMENT

> Execute le 2026-09-03 dans le worktree `LevelUp-wt-vehicules` (branche `wt/vehicules-tourelles`).
> Aucun commit, aucun `git add`. Mesures en AVANT-PLAN, `CGO_ENABLED=0`, GOCACHE isole
> (`scratchpad/gocache_v8`), films du checkout principal
> (`C:/Users/Guillaume/Projects/LevelUp/data/cache/film_chunks`, LECTURE SEULE). `apps/web/`,
> `cmd/weapon-sounds/`, `cmd/vs-measure/` : NON touches.
>
> **AVERTISSEMENT DE CONCURRENCE, ecrit avant les chiffres.** Une AUTRE session travaillait dans le
> MEME worktree pendant ce lot (lot V9, orientation `-dynamic-precision-` de `ti=40`) : elle a
> modifie `filmdec/{components_object,offline_aim,offline_biped,registry,traverse}.go` et la
> fonction `replay.vehicleScanOptions`. Tous les chiffres AVANT / APRES de ce rapport sont donc
> mesures **par le meme binaire, un seul parametre change** (la reference de vehicule effacee des
> evenements — `v8SansReference`), ce qui les rend insensibles a l'autre lot. La seule mesure qui
> ne peut pas l'etre est la comparaison des ARTEFACTS sur disque (§ 6) : l'artefact « avant » y
> date d'avant les deux lots, et l'ecart y est donc attribue avec cette reserve, recoupe ligne par
> ligne par l'instrument.

---

## 0. LE RESULTAT EN CINQ LIGNES

1. **LA SORTIE NOMME SON VEHICULE, ET C'EST DESORMAIS LA PRODUCTION QUI LE LIT.** La reference 1 de
   `unit_exit_vehicle`, lue en domaine 1 (le domaine des UNITES), tombe dans la bande `ti=40` sur
   **105 / 105** sorties de 12 films — zero bipede, zero hors bande, la ou le hasard en mettrait
   3,1 a 16,3 %. Mesure REFAITE par le decodeur de production (`VehicleEvent.VehicleSlot`), pas par
   un instrument parallele.
2. **LE RATTACHEMENT DES EPISODES PASSE DE 46 / 49 A 48 / 49** (5 films, 49 episodes attestes) : le
   nom rattache 2 episodes que la geometrie perdait, et n'en perd aucun. Le 48 / 49 du lot V6 est
   donc RETROUVE — par une autre voie, et avec l'ambiguite en moins.
3. **LES 6 DESACCORDS SONT EXPLIQUES UN PAR UN, ET ILS SONT LE PLUS BEAU RESULTAT DU LOT.** Dans les
   six cas, l'evenement nomme une vie MUETTE (aucun echantillon, aucun record de creation) dont le
   voisin immediat `slot + 1`, recense a la MEME fenetre, porte le chassis et la trajectoire ; les
   six sont de famille **warthog** ou **falcon** — les deux seules familles du corpus a tourelle
   d'artilleur — et portent le siege 0. **L'evenement nomme l'unite REELLEMENT quittee : le chassis
   pour le conducteur, la TOURELLE pour l'artilleur.** Les deux voies ont raison ; elles ne
   repondent pas a la meme question.
4. **LE CALQUE NE PEUT PAS ENCORE S'EN SERVIR, ET LE GARDE-FOU LE DIT.** Une tourelle n'a ni sprite
   ni trajectoire : un episode accroche a elle serait ECARTE a la publication. La regle livree est
   donc « le nom prime, SAUF quand il designe une vie que le calque ne dessinera pas ». Sans ce
   garde-fou, l'artefact perdait **4 episodes sur 41**.
5. **L'EMBARQUEMENT, LUI, NE NOMME RIEN.** Ses trois references (domaines 2/3/7) et la variante
   « ref 1 en domaine 1 » resolvent **0 / 15** slots `ti=40`. Le lot V6 avait raison sur ce point ;
   la production ne lit donc le vehicule que dans la SORTIE.

---

## 1. CE QUE LE LOT CHANGE, EN UNE PHRASE DE CODE

`filmdec/event_list.go` lisait la reference 1 de la sortie et **la jetait** (`r1 := readDom1Ref(...)`
puis plus rien). Elle est desormais publiee (`VehicleEvent.VehicleSlot` / `VehicleSlotValid` /
`VehicleGen`), et `replay/vehicle_rides_events.go` s'en sert comme resolution PRIMAIRE du vehicule
d'un episode d'occupation. La geometrie (`vehicleEventAnchorRadiusM`, 3 m, deux ancres) devient le
REPLI. Aucun seuil n'a bouge, aucun champ publie n'a change : `SchemaVersion` reste **30**.

---

## 2. MESURE A — LA REFERENCE VEHICULE DE LA SORTIE (`TestV8RefVehicule`, filmdec, 12 films)

Mesure faite **par le decodeur de production** (`decodeVehicleEvent`), pas par un chainage
d'instrument : c'est la difference avec le § 7 du rapport V7, et c'est ce qui la rend opposable.

| film | sorties | ref presente | bande `ti=40` | bande bipede | hors des deux | vie recensee a l'instant | gen accordee | chance analytique |
|---|---|---|---|---|---|---|---|---|
| `0d76e8f1` | 10 | 10 | **10** | 0 | 0 | 10 | 10 | 11,5 % |
| `fccc61cd` | 3 | 3 | **3** | 0 | 0 | 3 | 3 | 5,1 % |
| `4898d586` | 16 | 16 | **16** | 0 | 0 | 13 | 13 | 14,5 % |
| `e1bdb97f` | 7 | 7 | **7** | 0 | 0 | 7 | 7 | 10,5 % |
| `32a37698` | 9 | 9 | **9** | 0 | 0 | 9 | 9 | 14,7 % |
| `e3b10d4b` | 14 | 14 | **14** | 0 | 0 | 14 | 14 | 16,3 % |
| `51d3ab9f` | 9 | 9 | **9** | 0 | 0 | 8 | 8 | 9,4 % |
| `d99e5dbd` | 7 | 7 | **7** | 0 | 0 | 7 | 7 | 11,1 % |
| `e232ffce` | 5 | 5 | **5** | 0 | 0 | 5 | 5 | 3,1 % |
| `b232e02d` | 8 | 8 | **8** | 0 | 0 | 8 | 8 | 5,1 % |
| `d332c3a9` | 8 | 8 | **8** | 0 | 0 | 7 | 7 | 6,0 % |
| `c6250266` | 9 | 9 | **9** | 0 | 0 | 9 | 9 | 8,7 % |
| **TOTAL** | **105** | **105 (100,0 %)** | **105 (100,0 %)** | **0** | **0** | **100 (95,2 %)** | **100 / 100 (100,0 %)** | 3,1 a 16,3 % |

**Le 105 / 105 du lot V7 est reproduit a l'identique par le code de production.** La chance
analytique est `|bande ti=40| / (512 - |bande bipede|)` : 3,1 a 16,3 % selon le film.

**RESOLUTION EN VIE : 100 / 105.** Les 5 restantes tombent HORS de toute fenetre de recensement de
leur slot, l'ecart maximal a la fenetre la plus proche valant **41,9 s**. C'est une propriete du
RECENSEMENT (images-cles espacees de ~20 s), pas de la reference : le nom est bon, la fenetre est
courte. Elles sont traitees par la vie la plus proche dans le temps
(`vehicleResolvedByEventNearest`), et le clamp d'affichage les ecarte si elles depassent.

**FENETRES MULTIPLES : ZERO.** Aucune sortie ne tombe dans deux fenetres du meme slot — normal,
`assignVehicleWindows` decoupe les vies d'un meme slot. Le cas « plusieurs vies candidates » que le
mandat demandait de compter vaut **0 / 105** sur ce corpus.

**LES DEUX TEMOINS DE CE PARAGRAPHE SONT DEGENERES, ET C'EST DIT PLUTOT QUE MAQUILLE.**

- *Accord des generations* : 100 / 100. Il ne prouve RIEN — l'histogramme des generations vaut
  `[0, 460, 0, 0]` pour les vies recensees et `[0, 105, 0, 0]` pour les references : **toutes les
  vies `ti=40` du corpus ont la generation 1**. Le temoin par permutation le confirme : en
  appariant chaque sortie au vehicule de la SUIVANTE, l'accord de generation reste a **73 / 73 =
  100 %** au lieu des 25 % attendus du hasard. La generation ne discrimine pas sur ce corpus.
- *Une vie couvre l'instant* : 73 / 105 = 69,5 % au temoin permute, contre 95,2 % au reel. Temoin
  faible par construction — un vehicule vit presque tout le film, donc n'importe quel slot de la
  bande « repond oui ». Le temoin qui tranche est celui du § 4 (permutation contre la GEOMETRIE),
  et il vaut **0**.

**COROLLAIRE STRUCTUREL, publie tel quel** : sur les 479 slots de bande des 12 films, **AUCUN ne
porte plus d'une vie recensee**. La cle `(slot, gen)` se reduit donc a `slot` sur ce corpus ; la
resolution par fenetre temporelle est ecrite pour le cas general, elle n'a rien eu a departager
ici. C'est une limite de corpus, pas une preuve que le pool de slots ne reboucle jamais.

### 2.1 L'EMBARQUEMENT NE NOMME PAS SON VEHICULE (re-examine a la lumiere de « domaine 1 = unites »)

Les trois references de `biped_board_vehicle` (domaines 2/3/7, lecture Ghidra du 2026-09-02) sont
relues de QUATRE facons : valeur brute contre la bande `ti=40`, valeur rapportee a la base bipede
contre `ti=40`, la meme contre la bande bipede, plus la variante « ref 1 lue en DOMAINE 1, avec
sonde » — celle qu'il fallait tester puisque le domaine 1 s'est revele etre celui des unites.

| emplacement | presente | brute -> bande `ti=40` | base + valeur -> `ti=40` | base + valeur -> bipede |
|---|---|---|---|---|
| ref 0 (domaine 2) | 15 / 15 | **0** | **0** | 15 (100,0 %) |
| ref 1 (domaine 3) | 15 / 15 | **0** | **0** | 15 (100,0 %) |
| ref 2 (domaine 7) | 15 / 15 | **0** | **0** | 0 |
| *variante* ref 1 en domaine 1 | 15 / 15 | — | **0** | 1 (6,7 %) |

**ZERO sur 15 partout.** Le lot V6 avait refute la ref 2 comme vehicule (0/22 en bande) ; le lot V8
etend la refutation aux trois emplacements et a la lecture en domaine 1. **La production ne lit
donc le vehicule que dans la SORTIE**, et un episode ferme autrement (second embarquement, silence
terminal) sort sans nom — c'est exactement la population que la geometrie garde.

Fait gratuit, publie parce qu'il est net : la ref 1 de l'embarquement (domaine 3, 7 bits) resout un
BIPEDE dans **15 / 15** des cas. Le lot ne la nomme pas.

---

## 3. MESURE B — LE RATTACHEMENT DES EPISODES, AVANT / APRES (`TestV8VehiculeParEvenement`, replay)

Corpus : les **5 films et 49 episodes attestes** du lot V6, le meme socle de production (`v4Decode`,
memes largeurs de bloc MPP, meme pont slot -> joueur, meme horloge). Les deux regimes sont calcules
**dans le meme processus, sur le meme contexte decode** : le regime AVANT est obtenu en effacant la
reference de vehicule des evenements (`v8SansReference`) — un seul parametre change.

| film | episodes | nommes par la sortie | resolus par l'evenement | par la vie la plus proche | par la geometrie | NON rattaches | episodes de repli (trou) |
|---|---|---|---|---|---|---|---|
| `0d76e8f1` | 11 | 10 | 9 | 0 | 2 | 0 | 2 |
| `21468645` | 9 | 9 | 7 | 0 | 1 | 1 | 4 |
| `4898d586` | 18 | 16 | 12 | 3 | 3 | 0 | 2 |
| `a89a3d23` | 8 | 6 | 4 | 0 | 4 | 0 | 1 |
| `fccc61cd` | 3 | 3 | 2 | 0 | 1 | 0 | 0 |
| **TOTAL** | **49** | **44 (89,8 %)** | **34** | **3** | **11** | **1** | **9** |

*(la ligne TOTAL du journal de production agrege les cinq films ; les colonnes « par l'evenement /
par la vie la plus proche / par la geometrie » de l'instrument valent 34 / 3 / 11.)*

**LE CHIFFRE DU MANDAT.**

| regime | episodes rattaches |
|---|---|
| AVANT — geometrie seule (deux ancres, 3 m) | **46 / 49 = 93,9 %** |
| APRES — l'evenement d'abord, la geometrie en repli | **48 / 49 = 98,0 %** |
| gagnes (l'evenement rattache la ou la geometrie echouait) | **+2** |
| perdus (la geometrie rattachait, plus personne) | **0** |

**LE 48 / 49 DU LOT V6 EST RETROUVE, ET IL FAUT DIRE POURQUOI L'« AVANT » VAUT 46 ET NON 48.** La
table de rayons du lot V6 (§ 2.3) mesurait « un vehicule est-il sous 3 m de l'ancre ? » — une
question de DISTANCE. La production, elle, exige en plus que l'instant tombe dans la fenetre d'une
VIE de ce vehicule : c'est cette seconde condition qui coute les 2 episodes (46 / 49). **Le nom
porte par l'evenement ramene la production au plafond que le rayon annoncait** : 48 / 49, sans
elargir aucun rayon.

Les **5 episodes non nommes** sont exactement les **5 SILENCES TERMINAUX** du corpus (un
embarquement sans sortie) : l'embarquement ne nomme rien (§ 2.1), ils restent geometriques. Le
**1 episode non rattache** (film `21468645`) ne l'etait pas davantage avant.

---

## 4. MESURE C — LES 6 DESACCORDS, UN PAR UN, ET CE QU'ILS REVELENT

Sur les **41 episodes ou les DEUX voies repondent** : **35 designent la MEME vie**, **6 divergent**.
**TEMOIN PAR PERMUTATION : 0.** (Le vehicule nomme est remplace par celui de l'episode nomme
suivant du meme film ; l'accord avec la geometrie tombe de 35 / 41 a **0 / 41**. Le nom n'est pas
« un slot de la bande au hasard ».)

Les six, tels que l'instrument les imprime — meme forme dans les six cas :

| film | occupant | siege | EVENEMENT -> vie | GEOMETRIE -> vie | distance de l'ancre de DEBUT / FIN au vehicule geometrique |
|---|---|---|---|---|---|
| `0d76e8f1` | 554 | 0 | slot **772**, 0 echantillon, aucune naissance | slot **773** `fe32c0f4` **warthog**, 2 346 echantillons | 1,2 m / 1,3 m |
| `21468645` | 526 | 0 | slot **774**, 0 echantillon, aucune naissance | slot **775** `0000254b` **falcon**, 2 623 echantillons | 0,9 m / 0,8 m |
| `4898d586` | 547 | 0 | slot **770**, 0 echantillon, aucune naissance | slot **771** `fe32c0f4` **warthog**, 1 197 echantillons | 1,4 m / 1,4 m |
| `a89a3d23` | 536 | 0 | slot **774**, 0 echantillon, aucune naissance | slot **775** `fe32c0f4` **warthog**, 639 echantillons | 1,2 m / 1,3 m |
| `a89a3d23` | 538 | 0 | slot **772**, 0 echantillon, aucune naissance | slot **773** `fe32c0f4` **warthog**, 3 122 echantillons | 1,2 m / 0,9 m |
| `fccc61cd` | 558 | 0 | slot **777**, 0 echantillon, aucune naissance | slot **778** `fe32c0f4` **warthog**, 833 echantillons | 1,4 m / 1,3 m |

**QUATRE INVARIANTS, SUR SIX CAS SUR SIX :**

1. la vie nommee par l'evenement est **MUETTE** — zero echantillon de position, zero record de
   creation, donc ni chassis ni famille ni sprite ;
2. son slot vaut exactement **`slot geometrique - 1`** ;
3. les deux vies ont la **MEME fenetre de recensement**, au microseconde pres ;
4. l'occupant porte le **siege 0**, et le vehicule geometrique est un **warthog** (5 cas) ou un
   **falcon** (1 cas).

**LE RECENSEMENT DES VIES MUETTES CONFIRME LE MOTIF SUR TOUT LE CORPUS.** Sur les 5 films : 177 vies
recensees, dont **50 muettes**, dont **52 ont un voisin immediat PLEIN a la MEME fenetre** (les deux
sens comptes). En ne gardant que le voisin `slot + 1` a la MEME fenetre, sa famille est :

| famille du voisin `slot + 1`, meme fenetre | cas |
|---|---|
| **warthog** | 23 |
| **falcon** | 10 |
| chassis non resolu | 3 |
| ghost / mongoose / chopper / banshee | **0** |

**Les deux seules familles du corpus a porter une TOURELLE D'ARTILLEUR sont warthog et falcon** —
et ce sont les deux seules a trainer une vie muette juste avant elles. Le rapport
`ASSEMBLAGE_ENFANTS_2026-09-01.md` avait etabli le versant TAG de la meme chose : la tourelle
(`warthog_g`) est un `vehi` a part entiere, donc une entite `ti=40` de plus ; attachee au chassis,
elle ne replique jamais sa position, ce qui la rend MUETTE.

**QUI A RAISON ? LES DEUX, ET C'EST LA REPONSE.** L'evenement nomme l'unite REELLEMENT quittee — le
chassis pour le conducteur, la tourelle pour l'artilleur (les deux ayant un siege 0, chacun dans SA
propre unite). La geometrie, elle, designe le porteur, c'est-a-dire le seul objet que le calque sait
dessiner. **Ce dernier pas — « c'est l'artilleur » — n'est PAS prouve** : aucune verite terrain du
corpus ne dit qui, du conducteur ou de l'artilleur, descendait. Ce qui est mesure, ce sont les
quatre invariants ci-dessus et la table des familles.

**CE QUE LA PRODUCTION EN FAIT.** Une vie muette n'est pas publiee (`vehicleTrackOf` refuse une vie
sans naissance ni echantillon) : un episode accroche a elle serait un occupant PERDU. La regle
livree est donc « le nom prime, SAUF quand il designe une vie que le calque ne dessinera pas »
(`vehicleLifeFromEvent` + `vehicleDrawableLives`). **Le cout de ne PAS poser ce garde-fou a ete
mesure** : sans lui, l'artefact passait de 41 a 37 episodes (-4) sur les cinq films, dont 2 perdus
sur `a89a3d23` et 1 sur `fccc61cd`.

**L'AMBIGUITE GEOMETRIQUE TOMBE.** L'unique episode du corpus dont l'ancre a **deux** vehicules sous
3 m (le « 1 ambigu » de la table de rayons V6) est **nomme par sa sortie** : la distance n'a plus a
choisir. 1 / 1.

---

## 5. CE QUI RESTE VERT (garde-rails deja livres, re-mesures ici)

| garde | mesure du lot V8 |
|---|---|
| **fusion des relais** | vies publiees **82 -> 82** sur les 5 films (aucune vie de plus ni de moins) ; `merged` inchange par film (10 / 3 sur les deux films de demonstration, cf. § 6) |
| **clamp de la fenetre d'affichage** | les episodes resolus « par la vie la plus proche » (3 sur 49) restent soumis a `clampVehicleRides` — c'est lui qui borne le cas ou la fenetre de recensement est trop courte |
| **garde « non pilotable »** | inchangee : `vehicleFamilyIsRideable` s'applique APRES le rattachement, donc un episode nomme sur un `falcon` / `pelican` / `phantom` / `skiff` est toujours efface a la publication. C'est ce qui explique qu'un des 6 desaccords (le `falcon` de `21468645`) ne se voie pas dans les compteurs d'artefact |
| **episodes qui se chevauchent** (`Ambiguous`) | **7 -> 7** sur les 5 films : le lot ne les cree pas, et ne les resout pas non plus (il faudrait publier la distinction conducteur / artilleur — cf. § 8) |
| **occupant hors bande ecarte** | inchange (`vehicleEventsByOccupant`) |

---

## 6. LES DEUX ARTEFACTS DE DEMONSTRATION, RECUITS

Cuisson par `cmd/replay-build`, `LEVELUP_REPO_ROOT` = CE worktree, films lus dans le tronc principal
(`C:/Users/Guillaume/Projects/LevelUp/data/cache/film_chunks/<id>`), un processus par film, en
avant-plan. Artefacts ecrits dans `data/cache/replays/halo_infinite/` du worktree, puis COPIES vers
`C:/Users/Guillaume/Projects/LevelUp-wt-capture-rejeu/data/cache/replays/halo_infinite/`.

```
LEVELUP_REPO_ROOT=<worktree> replay-build --map behemoth      0d76e8f1 <cache>/film_chunks/0d76e8f1
LEVELUP_REPO_ROOT=<worktree> replay-build --map "launch site" fccc61cd <cache>/film_chunks/fccc61cd
```

---

## 7. CE QUI EST LIVRE

| fichier | etat | ce qu'il porte |
|---|---|---|
| `internal/analysis/filmdec/event_list.go` | **MODIFIE** (+40 L) | `VehicleEvent.VehicleSlot` / `VehicleSlotValid` / `VehicleGen` ; `decodeExitRefs` remplit la ref 1 au lieu de la jeter |
| `internal/analysis/filmdec/event_list_exit_grammar_test.go` | **NEUF** (101 L) | garde-rail de grammaire de la SORTIE sur payload synthetique, SANS environnement : occupant + vehicule + siege, avec trois temoins (les deux refs different ; l'embarquement ne publie pas de vehicule ; une ref 1 gardee-absente decale le siege de 12 bits) |
| `internal/analysis/filmdec/event_list_board_grammar_test.go` | MODIFIE (-7 L) | reutilise `evbForceType` au lieu de recopier la reecriture du champ de type |
| `internal/analysis/filmdec/vehicules_v8_refveh_test.go` | **NEUF** (360 L) | instrument corpus (garde `V7_ROOT` / `V7_FILMS`) : le 105 / 105 re-mesure par le decodeur de production, la resolution en vie, les histogrammes de generation, et le depouillement des trois references de l'EMBARQUEMENT |
| `internal/analysis/replay/vehicle_rides_events.go` | **MODIFIE** (+151 L) | `vehicleEpisode.vehSlot/vehValid/vehAtUS/resolvedBy` ; `vehicleLifeNamedByEvent` (resolution nue), `vehicleLifeFromEvent` (+ garde-fou de publiabilite), `vehicleLifeFromGeometry` (repli extrait) ; `vehicleRideFromEpisode` bascule sur l'evenement d'abord |
| `internal/analysis/replay/vehicle_rides.go` | **MODIFIE** (+70 L) | `vehicleRideStats`, `vehicleDrawableLives`, `drawable` dans `vehicleRideInputs` |
| `internal/analysis/replay/vehicle_tracks.go` | MODIFIE (+1 L) | passe `drawable` et rend le bilan. **Le fichier reste a 533 L** : le nouvel utilitaire a ete pose dans `vehicle_rides.go` precisement pour ne pas accroitre une dette gelee |
| `internal/analysis/replay/build_vehicles.go` | **MODIFIE** (+36 L) | `logVehicleRideResolution` — le journal par voie, et le `Warn` qui rompt le silence si plus aucun episode n'est nomme |
| `internal/analysis/replay/vehicle_rides_events_test.go` | MODIFIE (+115 L) | trois garde-rails NEUFS sans environnement : la sortie nomme / l'embarquement non ; le nom prime sur la distance ET le repli sur vie muette (temoin : l'ancre est a 0,5 m du mauvais vehicule et a 40 m du bon) ; la vie la plus proche quand l'instant sort des fenetres |
| `internal/analysis/replay/vehicules_v8_evenement_test.go` | **NEUF** (425 L) | instrument avant / apres (garde `V4_ROOT` / `V4_FILMS`) : les deux regimes, les desaccords un par un, le recensement des vies muettes, le temoin par permutation |
| `build_vehicles_test.go`, `vehicules_v4_{couverture,tirs}_test.go` | MODIFIES (3 L) | la troisieme valeur de retour de `buildVehicleTracks` |

**CONTRAT INCHANGE** : aucun champ publie ajoute ni retire, `SchemaVersion` reste **30**, aucune
regeneration OpenAPI, aucun type web, aucun golden touche. Les compteurs de voie ne sont PAS dans
`VehicleCoverage` (qui est publiee) : ils vivent dans le JOURNAL, ou ils disent comment l'artefact a
ete obtenu sans elargir le contrat.

---

### 6.1 Compteurs avant / apres, releves SUR L'ARTEFACT

| compteur | `0d76e8f1` (Behemoth) | `fccc61cd` (Launch Site) |
|---|---|---|
| `schemaVersion` | 30 -> **30** | 30 -> **30** |
| trajectoires joueur | 103 -> **103** | 102 -> **102** |
| vies de vehicule publiees | 20 -> **20** | 8 -> **8** |
| episodes d'occupation | 9 -> **9** | 2 -> **2** |
| occupants nommes | 8 -> **8** | 1 -> **1** |
| episodes avec siege | 7 -> **7** | 2 -> **2** |
| tirs totaux | 1 189 -> **1 189** | 769 -> **769** |
| **tirs en vehicule (`Shot.v`)** | 23 -> **23** | 0 -> **0** |
| relais fusionnes | 10 -> **10** | 3 -> **3** |
| episodes qui se chevauchent | 3 -> **3** | 0 -> **0** |

**LE CALQUE DES VEHICULES DE CES DEUX FILMS EST IDENTIQUE AU BIT PRES, ET IL FAUT LE DIRE PLUTOT QUE
DE CHERCHER UN GAIN.** Les deux artefacts de demonstration sont precisement des films ou les deux
voies s'accordent sur tout ce qui est PUBLIE : `0d76e8f1` a un desaccord (occupant 554) et
`fccc61cd` en a un (occupant 558), mais tous deux tombent sur une vie muette et repartent donc en
geometrie — meme reponse qu'avant. **Le gain mesurable du lot (+1 episode, +1 occupant nomme, +1
siege) est sur `4898d586`, qui n'est pas un artefact de demonstration.**

Ce que le journal de production dit maintenant, sur ces deux memes cuissons :

```
0d76e8f1  rattachement des episodes d occupation : episodesEvenement=11 nommesParLEvenement=10
          resolusParEvenement=9 resolusParVieLaPlusProche=0 resolusParGeometrie=2 nonRattaches=0
          episodesDeRepli=2
fccc61cd  rattachement des episodes d occupation : episodesEvenement=3 nommesParLEvenement=3
          resolusParEvenement=2 resolusParVieLaPlusProche=0 resolusParGeometrie=1 nonRattaches=0
          episodesDeRepli=0
```

**LA SEULE DIFFERENCE D'OCTETS ENTRE L'ANCIEN ET LE NOUVEL ARTEFACT N'EST PAS DE CE LOT, ET ELLE EST
UN BUG A PART ENTIERE.** Sur `0d76e8f1`, **5 entrees de `groundWeapons` sur 204** changent ; sur
`fccc61cd`, **1 sur 220** ; dans les six cas le SEUL champ qui bouge est **`dropper`** (le slot du
bipede qui a lache l'arme). La cause est lisible dans le code :
`replay/ground_weapon_rules.go:347`, `gwPadsClass` parcourt `for slot, vs := range lives` — **une
map, donc un ordre non deterministe** — et rend le PREMIER slot dont la fin de vie coincide avec le
lacher. Quand deux bipedes meurent au meme endroit dans la meme fenetre, le `dropper` publie
change d'une cuisson a l'autre. **Aucun rapport avec les vehicules ; DECOUVERTE NOTEE, NON TRAITEE**
(regle du perimetre). Tout le reste des deux artefacts est identique cle par cle.

---

## 8. LES GATES, ECRITS AVANT LA MESURE

| gate | enonce | mesure | verdict |
|---|---|---|---|
| **1** | la reference vehicule de la sortie est re-mesuree a **105 / 105** par le decodeur de PRODUCTION | 105 / 105 en bande `ti=40`, 0 bipede, 0 hors bande, 12 films ; chance analytique 3,1 a 16,3 % | **PASSE** |
| **2** | les **48 / 49** de l'ancrage geometrique sont retrouves OU ameliores | 46 / 49 avant (resolution de production complete), **48 / 49 apres** ; +2 gagnes, 0 perdu | **PASSE** |
| **3** | l'ambiguite geometrique connue (1 cas) tombe | l'unique episode a deux vehicules sous 3 m est **nomme** par sa sortie : 1 / 1 | **PASSE** |
| **4** | les desaccords evenement / geometrie sont **comptes et expliques un par un** | 6 desaccords sur 41 doubles reponses, quatre invariants communs, table des familles des vies muettes (23 warthog / 10 falcon / 0 autre) | **PASSE** |
| **5** | temoin par PERMUTATION | l'accord evenement / geometrie tombe de **35 / 41 a 0 / 41** quand le vehicule nomme est permute | **PASSE** |
| **6** | fusion des relais, clamp, garde « non pilotable » restent verts | vies publiees 82 -> 82 (5 films) ; relais 10 / 3 inchanges sur les deux artefacts ; garde `falcon` toujours active (elle efface un des 6 desaccords avant publication) ; chevauchements 7 -> 7 | **PASSE** |
| **7** | `SchemaVersion` ne bouge que si le contrat change | aucun champ publie touche : **30 inchange** | **PASSE** |
| **8** | *controle negatif* — l'EMBARQUEMENT porte-t-il aussi un vehicule ? | **0 / 15** sur les trois references et sur la variante domaine 1. La reponse est NON, et la production ne pretend rien | **PASSE (negatif assume)** |

### Gates d'execution

| gate | commande | resultat |
|---|---|---|
| gofmt | `gofmt -l internal/analysis/{filmdec,replay}/` | **sortie VIDE** |
| vet | `CGO_ENABLED=0 go vet ./internal/analysis/filmdec/ ./internal/analysis/replay/` | **exit 0** |
| tests SANS environnement | `CGO_ENABLED=0 go test ./internal/analysis/filmdec/ ./internal/analysis/replay/ -count=1` | `ok filmdec 0,9 s` · `ok replay 28,1 s` — **`grep -c '^--- FAIL:'` = 0** |
| service en CGO=1 | `CGO_ENABLED=1 go build ./cmd/server` | **exit 0, sortie vide** |
| seuils | tout fichier NEUF <= 425 L ; `vehicle_tracks.go` reste a **533 L**, sa valeur d'AVANT le lot (dette gelee non accrue — le nouvel utilitaire a ete pose dans `vehicle_rides.go` pour cela) ; fonctions <= 80 L | **PASSE** |
| perimetre | `apps/web/`, `cmd/weapon-sounds/`, `cmd/vs-measure/` : non touches **par ce lot** (le `git status` du worktree en fin de lot montre des fichiers `apps/web/match-replay/` modifies — ils sont de l'AUTRE session, cf. l'avertissement de concurrence en tete) ; aucun commit, aucun `git add` | **PASSE** |

Commandes de rejeu des instruments :

```
# la reference vehicule, 12 films (~3 min)
export V7_ROOT=<cache>/film_chunks
export V7_FILMS="0d76e8f1,fccc61cd,4898d586,e1bdb97f,32a37698,e3b10d4b,51d3ab9f,d99e5dbd,e232ffce,b232e02d,d332c3a9,c6250266"
CGO_ENABLED=0 go test ./internal/analysis/filmdec/ -run TestV8RefVehicule -v -timeout 120m

# l'avant / apres du rattachement, 5 films (~9 min)
CGO_ENABLED=0 V4_ROOT=<cache> \
  V4_FILMS="0d76e8f1:Behemoth,21468645:Behemoth,4898d586:Behemoth,a89a3d23:Behemoth,fccc61cd:Launch Site" \
  go test ./internal/analysis/replay/ -run TestV8VehiculeParEvenement -v -timeout 120m
```

---

## 9. CE QUI RESTE OUVERT (note, NON traite — regle du perimetre)

1. **LA TOURELLE EST UNE UNITE, ET L'EVENEMENT LA NOMME : C'EST LA CLE DU « CONDUCTEUR vs
   ARTILLEUR ».** Le calque publie aujourd'hui `seat` = 0 pour les deux, et compte leurs episodes
   comme un CHEVAUCHEMENT (`Ambiguous` = 7 sur les 5 films, 3 sur `0d76e8f1`). Si la vie muette
   nommee par la sortie est bien la tourelle, alors « l'evenement nomme une vie muette voisine du
   chassis » EST le marqueur d'artilleur — et l'ambiguite se resoudrait sans geometrie. **Cela
   demande un champ publie (donc un bump de contrat) et une verite terrain** (un visionnage, ou le
   fil des morts qui attribue un kill de tourelle). Non traite.
2. **LE PONT « vie muette -> chassis porteur » n'est PAS pose, et c'est delibere.** Le motif
   `slot + 1`, meme fenetre, est net (33 vies muettes sur 50 ont un tel voisin, 23 warthog +
   10 falcon + 3 chassis non resolus, zero autre famille), mais il repose sur 6 episodes et sur une
   regle d'adjacence de slots qu'aucun descripteur ne prouve. Le garde-fou actuel (repli
   geometrique) rend le MEME resultat sans inventer la regle. A ouvrir le jour ou un episode
   nomme-muet n'aura PAS de reponse geometrique — cas non observe sur ce corpus.
3. **LES 5 SORTIES HORS FENETRE (4,8 %, ecart maximal 41,9 s).** Elles sont traitees par la vie la
   plus proche. Le vrai correctif serait d'etendre la fenetre d'une vie a la derniere PREUVE de
   presence du flux de position (et non au seul recensement), c'est-a-dire de reconcilier
   `vehicleLife.hiUS` avec `VehicleTrack.T1`. Non traite : cela deplacerait les bornes de TOUTES les
   vies, pas seulement de celles-la.
4. **LE `dropper` DES ARMES AU SOL EST NON DETERMINISTE** (`replay/ground_weapon_rules.go:347`,
   parcours de map). 5 entrees sur 204 et 1 sur 220 changent d'une cuisson a l'autre sur les deux
   artefacts de demonstration. Hors perimetre, note, non corrige.
5. **LE CORPUS N'A QU'UNE GENERATION.** Toutes les vies `ti=40` des 12 films ont `gen = 1`, et aucun
   slot ne porte deux vies. La resolution `(slot, instant) -> vie` est donc ecrite mais jamais
   EPROUVEE sur un rebouclage de pool. Un film long (Big Team, ou un mode a manches) serait le
   corpus qu'il faut.

---

## 10. STATUT DES ITEMS DU MANDAT

| item | statut | justification |
|---|---|---|
| 1. filmdec additif : publier la reference vehicule de la SORTIE | `[x]` | `VehicleSlot` / `VehicleSlotValid` / `VehicleGen`, remplis dans `decodeExitRefs` |
| 1bis. tester si l'EMBARQUEMENT porte la sienne (re-examen « domaine 1 = unites ») | `[x]` | § 2.1 — **0 / 15** sur les trois references ET sur la variante domaine 1. Negatif net |
| 1ter. garde-rail de grammaire sur payload synthetique, NON garde par env | `[x]` | `event_list_exit_grammar_test.go`, trois temoins integres, tourne dans la suite ordinaire |
| 1quater. instrument corpus env-gated : re-mesure du 105 / 105 et taux cote board | `[x]` | `vehicules_v8_refveh_test.go` — 105 / 105 par le decodeur de production, board 0 / 15 |
| 2. replay : le slot de l'evenement en resolution PRIMAIRE, geometrie en repli | `[x]` | `vehicleLifeFromEvent` puis `vehicleLifeFromGeometry` dans `vehicleRideFromEpisode` |
| 2bis. cle de vie `(slot, gen)` par la fenetre temporelle ; cas multiples comptes | `[x]` | `vehicleLifeNamedByEvent` ; **0 cas multiple** (les fenetres d'un meme slot ne se recouvrent pas), et le cas « aucune fenetre » (5 / 105) traite par la plus proche et compte a part |
| 3. GATES avant / apres chiffres, desaccords expliques un par un, temoin par permutation | `[x]` | §§ 3, 4, 8 |
| 4. reconstruire les 2 artefacts de demonstration et les copier | `[x]` | § 6 — recuits par `cmd/replay-build`, copies vers `LevelUp-wt-capture-rejeu` ; calque vehicules identique, seule difference : le `dropper` non deterministe des armes au sol |
| 5. `SchemaVersion` : bump seulement si le contrat change | `[x]` | contrat inchange, **30** |
| 6. Rapport + entree de thought_log en tete | `[x]` | ce fichier ; entree en tete de `.ai/thought_log.md` |

---

## 11. CR HONNETE

- **Ce que le lot prouve** : la sortie nomme son vehicule (105 / 105 par le code de production), et
  s'en servir fait passer le rattachement de 46 / 49 a 48 / 49 sans elargir un seul rayon, sans
  perdre un episode, et en supprimant l'unique ambiguite geometrique du corpus.
- **Ce que le lot ne prouve pas** : que l'artefact visible change. Sur les DEUX films de
  demonstration, le calque des vehicules sort identique au bit pres. Le gain est reel mais il vit
  sur un troisieme film (`4898d586` : +1 episode, +1 occupant nomme, +1 siege). Dire l'inverse
  serait maquiller un +1 en revolution.
- **Ce que le lot decouvre sans l'avoir cherche** : la tourelle d'un Warthog (et d'un Falcon) est
  une entite `ti=40` MUETTE, adjacente au chassis, recensee a la meme fenetre — et l'evenement de
  sortie la nomme. C'est la piste la plus prometteuse du chantier pour distinguer le conducteur de
  l'artilleur, et elle est chiffree ici sans etre traitee.
- **Ce qui a failli passer inapercu** : sans le garde-fou de publiabilite, le lot AURAIT PERDU
  4 episodes sur 41 en croyant les ameliorer — un « nom exact » pointant sur une entite que le
  calque ne dessine pas. C'est la mesure avant / apres, et elle seule, qui l'a attrape.
- **Reserve de methode** : une autre session modifiait le meme worktree pendant le lot (V9,
  orientation `-dynamic-precision-` de `ti=40`). Tous les avant / apres des §§ 3 et 4 sont calcules
  dans le MEME processus avec un seul parametre change, donc immunises ; la comparaison d'artefacts
  du § 6 ne l'est pas, et c'est ecrit en tete du rapport.
