# RAPPORT R5 — Les événements nommés des AUTRES équipements : inventaire, recensement, croisement

Date : 2026-09-03. Lot R5 du plan `PLAN_LECTURE_FIABLE_EQUIPEMENT_2026-09-03.md`. Recherche
pure, lecture seule : aucun fichier de production touché, aucun DuckDB ouvert. Instruments
(worktree agent, package `apps/go-api/internal/analysis/filmdec/`) :

- `equipements_events_research_test.go` (recensement, vérité terrain, croisement par slot)
- `equipements_events_sondes_research_test.go` (croisement temps seul, sonde de liste, bilans)

Gardés par `EQUIP_EVENTS_ROOT` / `EQUIP_EVENTS_ARTS` / `EQUIP_EVENTS_IDS`, skip par défaut,
`CGO_ENABLED=0`, lecture O(1) par paquet (têtes de liste uniquement — aucune trame décodée),
`go vet ./internal/analysis/filmdec/` vert. Commandes rejouables en annexe.

## Verdict en quatre phrases

**Un seul équipement a un événement d'usage fiable en tête de paquet : le translocateur
(type 117, établi par R1).** **Les poses de déployables (mur, capteur, écran, chercheur) ont un
événement réel — type 103 `EquipmentSpawnedObject` — qui les date à ~100 ms, mais la TÊTE seule
n'en rappelle que 60 % (77/129 murs) avec des films à 0 %, et il ne nomme PAS le poseur (ses
références désignent des objets, pas le bipède).** **Camo/surbouclier (type 100
`PowerUpApplied`) et réparation (type 118) existent mais sont inexploitables en tête (rappel
7/31 et 0/11).** **Répulseur, grappin, propulseur : AUCUNE tête sur 325 160 paquets — le
grappin restant couvert par son canal de production existant (grappleLines).**

## 0. Méthode (héritée de R1, rapport §0 et §4)

- Canal : liste d'événements en tête des paquets delta, `[1 bit config][(1 R(7) type …)* 0]`.
  Tête lisible en O(1) : `pay[0]&0xC0==0xC0`, `type = (pay[0]&0x3F)<<1 | pay[1]>>7`.
- Piège d'horloge : paquets en horloge MOTEUR, artefacts en horloge FILM. Conversion :
  `ms_film = (ts_paquet − ts_premier_paquet_chunk1)/1000` ; vérité terrain :
  `ms_film = originMs + frame × frameIntervalMs`.
- Vérité terrain (artefacts `data/cache/replays/halo_infinite/<id8>.json`) : `grappleLines[]`
  (tractions datées, slot), `equipmentPlacements[]` origin=deployed des familles déployables
  (mur, capteur, écran, champ de réparation, threat_seeker ; les « deployed » de familles non
  déployables — grapple/thruster/repulsor — sont des attributions douteuses du canal placements,
  comptées à part), `equipmentEpisodes[]` camo/surbouclier (épisodes + « débuts » regroupés à
  écart > 5 s, approximation d'activation dite comme telle). Les kills répulseur ne sont PAS
  datés dans les artefacts (`killEffects` est une table de libellés) : pas de vérité répulseur.
- Fenêtre d'appariement ±1 200 ms (le 117 tombe 5-80 ms avant le geste ; la vérité est sur une
  grille de 100 ms et les poses sont vues avec retard). Les médianes de dt sont rapportées.

## 1. R5.1 — Inventaire fermé des types suspects (annexe A de la grammaire)

Critère : nom exe parlant d'équipement ou d'usage de capacité. Taille = tampon de réception.

| Type | Nom exe | Taille | Pourquoi suspect |
|---|---|---|---|
| 28 | biped_debug_teleport | 12 | téléport (adjacent) |
| 30 | biped_equipment_activation | 72 | LE candidat générique d'usage |
| 31 | equipment_teleport_request | 8 | requête translocateur |
| 39 | biped_throw_initiate | 20 | lancer (grenades probables) |
| 42 | biped_dodge | 24 | esquive (propulseur ?) |
| 43 | initiate_mobility_action | 164 | mobilité (propulseur ?) |
| 48 | weapon_tether_request | 20 | câble (grappin ?) |
| 51 | biped_throw_release | 68 | lancer (grenades probables) |
| 93 | activate_spartan_ability | 64 | activation de capacité |
| 98 | Equipment | 8 | générique (9 bits fixes) |
| 100 | PowerUpApplied | 12 | camo / surbouclier |
| 103 | EquipmentSpawnedObject | 0 | objet créé par un équipement (pose) |
| 104 | EquipmentKnockbackPlayer | 12 | répulseur (joueur poussé) |
| 105 | EquipmentObjectKnockedBack | 4 | répulseur (objet poussé) |
| 115 | synchronized_teleport | 44 | téléport (adjacent) |
| 116 | teleport_effects | 56 | téléport (adjacent) |
| 117 | EquipmentTranslocatorTeleportEffects | 28 | ÉTABLI par R1 (18/18) |
| 118 | repair_complete | 4 | réparation |
| 119 | EquipmentKnockbackRequest | 276 | répulseur (requête) |

Aucun nom de l'annexe A ne contient camo, overshield, shroud, grapple, thruster ou quantum :
les 123 types sont couverts, la liste est close.

## 2. R5.2 — Recensement des têtes sur 9 films (325 160 paquets delta)

Films : les 3 imposés (`1b2d9e08` Dynasty, `a0c36016`, `000d5950` Fiesta) + 6 choisis pour leur
richesse en vérité terrain (`06dfe6d9`, `084a804d`, `4f77afc1`, `8a485699`, `bf2a9f05`,
`d1dfbc02` — grappins 360, murs 129, capteurs 27, camo 258 épisodes, surbouclier 15 épisodes).

Têtes des types suspects PAR FILM (dénominateur : paquets delta du film) :

| Film | Paquets delta | t39 | t100 | t103 | t105 | t117 | t118 |
|---|---|---|---|---|---|---|---|
| 000d5950 | 30 371 | 61 | 0 | 46 | 0 | 0 | 0 |
| 06dfe6d9 | 24 418 | 94 | 1 | 52 | 4 | 1 | 7 |
| 084a804d | 32 297 | 187 | 10 | 29 | 2 | 0 | 4 |
| 4f77afc1 | 34 514 | 151 | 6 | 30 | 1 | 0 | 0 |
| 8a485699 | 42 705 | 62 | 0 | 40 | 0 | 0 | 0 |
| bf2a9f05 | 43 880 | 96 | 0 | 74 | 0 | 0 | 0 |
| d1dfbc02 | 36 870 | 74 | 0 | 70 | 0 | 0 | 0 |
| 1b2d9e08 | 33 571 | 99 | 0 | 0 | 0 | 3 | 0 |
| a0c36016 | 46 534 | 188 | 0 | 6 | 1 | 4 | 0 |
| **Parc** | **325 160** | **1 012** | **17** | **347** | **8** | **8** | **11** |

**Treize types suspects ont ZÉRO tête sur les 325 160 paquets : 28, 30, 31, 42, 43, 48, 51,
93, 98, 104, 115, 116, 119.** Ce zéro est significatif : des types NON suspects très rares
apparaissent bien en tête sur le même parc (type 3 : 3 têtes ; type 85 `PlayerKilledEvent` :
2 ; type 106 : 3 ; type 41 : 7). Un type émis au volume des usages mesurés (~900 usages
d'équipement sur le parc) aurait laissé des têtes. L'absence TOTALE de 30
(`biped_equipment_activation`), 93, 104, 119 et 48 suggère qu'ils ne sont pas émis dans le
film multijoueur — sans que la tête seule puisse le prouver (un type toujours émis derrière un
autre dans le même paquet serait invisible ici ; improbable 13 fois de suite, dit pour mémoire).

## 3. R5.2 — Croisement avec la vérité terrain

### 3.1 Identification du porteur par ref0 : NÉGATIF pour 100/103/118

L'hypothèse R1 (ref0 = bipède, porte 1 + index 8 bits base 512 + gen 2) qui ferme 18/18 sur le
type 117 rend, sur les autres types, des valeurs qui DÉRIVENT avec le temps de film (103 :
542→709 sur `000d5950` ; 100 : 640→695 sur `4f77afc1`) — la signature d'index d'OBJETS du monde
(l'allocation croît), pas de slots bipèdes. Précision par slot w8 : 103 = 3/347, 100 = 0/17,
118 = 0/11, soit le hasard. L'hypothèse w=9 n'apparie rien de plus. Décodage manuel de deux
têtes 103 (bit à bit) : ref0 ≈ objet (13 bits), ref1 ≈ objet (13 bits), ref2 absente —
**le poseur n'est référencé nulle part dans l'en-tête du 103**. Pour le 100, la charge porte un
mot 32 bits constant `D6 75 7C 79` (le `variant-name` de la grammaire prouvée, un chaîne-id).
Largeurs non sourcées de l'exe (vtable+0x58 des descripteurs) : hypothèses, cohérentes mais non
prouvées — voir la sonde 3.5 qui les corrobore.

### 3.2 Type 103 `EquipmentSpawnedObject` vs poses déployées — le signal des DÉPLOYABLES

Croisement TEMPS SEUL (slot inapte, cf. 3.1), fenêtre ±1 200 ms. Rappel par film (murs) :

| Film | Rappel murs | Rappel capteurs | Autres déployables |
|---|---|---|---|
| 000d5950 | 15/17 | 3/4 | — |
| 06dfe6d9 | 6/35 | 1/12 | champ 2/9, chercheur 1/1 |
| 084a804d | 2/3 | 1/4 | champ 0/2 |
| 4f77afc1 | **0/14** | — | écran 1/1 |
| 8a485699 | 14/17 | 1/3 | — |
| bf2a9f05 | 17/20 | 1/2 | — |
| d1dfbc02 | 19/19 | 1/2 | — |
| a0c36016 | 4/4 | — | — |
| **Parc** | **77/129 (60 %)** | **8/27 (30 %)** | champ 2/11, écran 1/1, chercheur 1/1 |

Précision parc : 189/347 têtes datent une pose publiée (99 vers un deployed, 90 vers un
dropped/unknown — les objets lâchés à la mort SONT aussi des apparitions d'objets d'équipement) ;
158/347 sans correspondance — attendu : l'artefact ne publie que les poses CONFIRMÉES
(`coverage.placements` : 6 651 ancres → 295 publiées sur `000d5950`), le film voit plus
d'apparitions que l'artefact n'en publie. Datation des 189 appariés : dt médian +94 ms,
107/189 à ±300 ms, bornes −1 165/+1 179 ms.

Lecture : l'événement est RÉEL et date la pose (~100 ms), mais (a) en tête seule le rappel
va de 0/14 à 19/19 selon le film — inutilisable seul ; (b) plusieurs 103 par pose existent
(d1dfbc02 : 28 appariements pour 19 murs) ; (c) pas d'attribution au poseur. Le canal de
production `equipmentPlacements` (entités) donne déjà datation + position + poseur (289/295
avec owner sur `000d5950`) : **le 103 n'apporterait, même en liste complète, qu'une datation
d'appoint, pas une lecture nouvelle.** Les deux films faibles (06dfe6d9 6/35, 4f77afc1 0/14)
ne sont PAS un décalage d'horloge : leurs dt bruts sont dispersés des deux signes (−31 s à
+45 s), et des appariements fins existent tard dans le film (+713 ms à 569 s) — c'est bien la
tête qui manque (événement enfoui ou non émis pour ces poses), pas la conversion.

### 3.3 Type 100 `PowerUpApplied` vs camo / surbouclier — FAIBLE en tête

17 têtes parc, présentes uniquement sur les 3 films à épisodes camo/surbouclier ET powerups
(06dfe6d9 : 1, 084a804d : 10, 4f77afc1 : 6). Croisement temps seul : précision 7/17 (6 camo +
1 surbouclier), dt médian +43 ms, 6/7 à ±300 ms — quand il apparie, il date bien. Mais rappel :
camo 7/31 épisodes (débuts regroupés : 6/19), surbouclier 1/15 (débuts : 1/15) ; et 0 partout
sur `4f77afc1` (6 têtes, 15 épisodes, aucun appariement — les 6 têtes y portent toutes le même
`variant-name`, probablement un seul genre de powerup, et les épisodes viennent d'autre chose).
Réserves : les épisodes mesurent l'INVISIBILITÉ OBSERVÉE (fragmentée par les tirs, retardée par
le fondu), pas l'activation — le regroupement > 5 s est une approximation dite comme telle.
Verdict : l'événement existe (la liste complète pourrait le rendre exploitable), la tête seule
ne suffit pas, et ref0 désigne l'objet powerup, pas le joueur.

### 3.4 Type 118 `repair_complete` — signal d'EFFET, pas d'usage

11 têtes, exclusivement sur les 2 films à champ de réparation (7 + 4) — corrélation d'existence
au niveau film. Mais 0/11 appariées aux poses (±1,2 s) ; les émissions vont par rafales
(6 têtes en 5 s espacées de ~500 ms, références CONSTANTES dans la rafale = le même objet
réparé), à −24,5 s…−2,6 s de la pose publiée la plus proche. Sémantique probable : « une
réparation s'est accomplie » (par tick/section), pas « le champ est posé ». Ne date PAS l'usage.

### 3.5 Répulseur (104/105/119) — RIEN d'exploitable, et rien pour le mesurer

104 et 119 : 0 tête sur 325 160 paquets. 105 : 8 têtes (les objets poussés par un répulseur
sont rares), ref0 STABLE en bande bipède (516-563, ne dérive pas — plausiblement une unité),
mais AUCUNE vérité terrain datée n'existe côté artefacts (ni kills répulseur datés, ni poussées)
— les 4 « appariements » grapple à ~+1 s sont des coïncidences de densité. **Non croisable
aujourd'hui : à re-mesurer si une vérité datée apparaît.**

### 3.6 Grappin et propulseur — AUCUN événement, mais le grappin est déjà couvert

- Grappin : 360 tractions mesurées sur le parc, et AUCUNE tête de type 48
  (`weapon_tether_request`) ni d'aucun autre suspect corrélé. La production date déjà chaque
  traction par son propre canal (grappleLines : lightReads/heavyReads/pulls) — le canal des
  événements n'ajoute rien, et rien ne manque.
- Propulseur : types 42/43 à 0 tête ; aucune vérité datée (les « thruster deployed » du canal
  placements sont des attributions douteuses, comptées à part). Rien à lire, rien pour mesurer.
- Type 39 `biped_throw_initiate` (1 012 têtes) : c'est le canal des lancers de GRENADES — sur
  1 012 têtes, 74 coïncidences d'équipement seulement (essentiellement des place:dropped, la
  mort qui suit la grenade). Hors périmètre équipement, dit pour fermer la piste.

## 4. R5.3 — Table finale et estimation « liste complète »

| Équipement | Signal d'usage dans le canal des événements (tête) | Fiabilité mesurée |
|---|---|---|
| Translocateur | **OUI** — type 117 (R1) | précision 18/18, rappel 8/8 (R1) |
| Mur / capteur / écran / champ / chercheur (poses) | **PARTIEL** — type 103, temps seul | date à ~100 ms quand présent ; rappel murs 60 % (0-100 % selon film) ; ne nomme pas le poseur ; redondant avec placements |
| Camo / surbouclier | **PARTIEL/FAIBLE** — type 100 | dt +43 ms quand apparié ; rappel épisodes 7/31 et 1/15 ; 0 sur un film entier |
| Réparation (accomplie) | **NON pour l'usage** — type 118 = effet | 0/11 vs poses ; rafales par objet réparé |
| Répulseur | **NON en tête** (104/119 : 0 ; 105 : 8 têtes) | pas de vérité datée pour mesurer |
| Grappin | **NON** (48 : 0 tête / 325 160) | déjà daté par grappleLines en production |
| Propulseur | **NON** (42/43 : 0 tête) | pas de vérité datée non plus |

**Ce que la LISTE COMPLÈTE (vs tête seule) rapporterait — mesuré :**

1. Sonde directe sur les têtes 103 (charge PROUVÉE à 0 bit : l'événement se termine dans la
   tête) : sous l'hypothèse refs 13+2 bits, 347 têtes se décodent en 267 « seul événement du
   paquet » + 80 « suivi d'un autre » (23 %), et la distribution des types suivants (36×19,
   6×14, 0×13, 82×8, 21×4, 15×4, …) REPRODUIT la distribution du census — auto-validation
   forte de l'hypothèse de largeur (un désalignement rendrait du bruit uniforme, pas le
   census). **La profondeur de liste est réelle : un quart des paquets à tête 103 portent au
   moins un second événement.**
2. Bornes hautes du gain par les déficits de rappel en tête : murs 52/129 poses sans tête 103,
   capteurs 19/27, camo 24/31 épisodes sans tête 100, surbouclier 14/15 — SI l'événement est
   émis pour chaque usage (indémontrable en tête seule), la liste complète récupérerait au plus
   ces déficits. S'y ajoute la seule réserve de R1 (3 spent sans tête 117).
3. MAIS le gain FONCTIONNEL resterait faible : pour les poses, la production lit déjà mieux
   (placements avec poseur et position) ; pour camo/surbouclier, l'attribution au porteur
   exigerait en plus de résoudre les refs objets ; seuls le translocateur (fait) et
   d'éventuels types aujourd'hui invisibles en tête (30, 93, 104…) changeraient la donne — et
   leur absence totale en tête sur 325 160 paquets rend improbable qu'ils existent dans la
   bobine. **Le décodage de la liste complète reste le chantier `PLAN_PERCER_TRAME_FILM`, pas
   un prérequis d'un équipement précis.**

## 5. Statut des items du plan

- R5.1 (inventaire) : **fait** — liste fermée de 19 types (section 1), critère dit, tailles.
- R5.2 (recensement + croisement ≥ 5 films) : **fait** — 9 films dont les 3 imposés, précision
  et rappel par type avec dénominateurs (sections 2-3), négatifs dits avec dénominateurs.
- R5.3 (rapport + estimation liste complète) : **fait** — section 4 (sonde mesurée 80/347 +
  bornes par déficits de rappel).

## 6. Réserves et limites (à lire avant tout usage aval)

1. TÊTE SEULE : tout rappel mesuré ici est un rappel « en tête de liste » — borne BASSE du
   rappel réel de l'événement dans le film.
2. Vérités terrain imparfaites : placements = poses CONFIRMÉES (filtrées), épisodes camo/OS =
   invisibilité observée (fragmentée, retardée), grappleLines = tractions (les tirs sans
   traction ne sont pas dedans, `unpairedFires` non croisés). Les dénominateurs sont ceux des
   artefacts, pas ceux du jeu.
3. Largeurs de références des types 100/103/105/118 : hypothèses (corroborées par la sonde et
   la dérive), à sourcer de l'exe (`vtable+0x58` des descripteurs) avant tout décodeur.
4. Les dénominateurs de rappel par type n'incluent que les films où le type a au moins une
   tête quelque part (ex. grapple×103 : 341 = 360 − 19, `1b2d9e08` sans tête 103).
5. Modes : le parc couvre Fiesta/arène/BTB implicitement (via la variété des artefacts) mais
   aucune identification de mode n'a été faite film par film ; l'hétérogénéité du rappel 103
   (0/14 vs 19/19) reste NON EXPLIQUÉE — mesurée, pas interprétée.

## Annexe — commandes exactes (depuis `apps/go-api` du worktree)

```
CGO_ENABLED=0 \
  EQUIP_EVENTS_ROOT=C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks \
  EQUIP_EVENTS_ARTS=C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/replays/halo_infinite \
  EQUIP_EVENTS_IDS=000d5950,06dfe6d9,084a804d,4f77afc1,8a485699,bf2a9f05,d1dfbc02,1b2d9e08,a0c36016 \
  go test ./internal/analysis/filmdec/ -run '^TestEquipementsEvenements$' -count=1 -timeout 30m -v
```

Durée mesurée : < 1 s de recensement par film (0,28 s pour les 9 films, artefacts compris).
Le log complet du run final est reproductible à l'identique (aucune source d'aléa).
Statistiques dt (§3.2) recalculables depuis le log : lignes `[type 103] … dt … ms` sans
`HORS fenetre`.
