# Lot `tirs_vehicules` — le Ghost ne tire jamais dans le rejeu (enquête du 2026-09-23)

Commit de référence `43a01721e` (le checkout principal est passé à `fe2106f4b` « Archivages docs »
pendant l'enquête ; `ETAT_DE_L_ART_KILLWEAPON.md` vit désormais sous `.ai/V7.5/` ; les numéros de
ligne du `thought_log.md` sont ceux de `fe2106f4b`). Lecture seule du dépôt principal. Une seule sonde Go (test `research`
sur 3 fichiers de faits persistés, aucun film), sous verrou `voie.sh`, dans le worktree détaché
`C:\Users\Guillaume\Downloads\Scripts\LevelUp-wt-inv-tirs_vehicules` (à retirer par le superviseur).
Scripts Node et journaux : `SP/sondes/tirs_vehicules/` (copie de la sonde Go incluse).

Légende : **[M]** mesuré (fichier:ligne, sortie chiffrée) · **[D]** déduit · **[H]** hypothèse.

---

## 1. Constat reproduit sur pièces

### 1.1 Le match 81c02726 (Quick Play, Strongholds, Isolation, 8 joueurs)

Horloge : `t = affiché_s × 10 + 227` ; frag `time_ms` → `t = (time_ms − originMs 10 559) / 100`.

| Élément | Valeur | Source |
|---|---|---|
| Épisodes de G MONEY 2123 (filmIndex 2) dans un Ghost `5b80c406` | slot 514, t 311→1088 (affiché 0:08.4→1:26.1) ; slot 541, t 2064→2869 (3:03.7→4:24.2) ; `src: proximity`, sans siège | document + sonde **[M]** |
| Pont slot → index | index 2 → slots [514 532 537 541], liens directs par création de bipède | sonde **[M]** |
| Frags de G MONEY au Ghost (`source_tag = f712c64a`, `veh_cv_ghost`) | t 395 (0:16.8), 623 (0:39.6), 650 (0:42.3), 2159 (3:13.2), 2228 (3:20.1), 2402 (3:37.5) — **tous dans un des deux épisodes** | `match_kill_events_latest` (copie) **[M]** |
| Événements de tir décodés (`FilmInputs.Fire`) | 1 063, **tous** d'arme personnelle (moitié basse `0x42C9679F`) ; **zéro** d'arme de véhicule, zéro identifiant décalé | sonde **[M]** |
| Tirs de l'index 2 pendant ses épisodes (±1 s) | **0** (ses 116 tirs sont tous rattachés à pied, hors épisodes) | sonde **[M]** |
| Tirs de l'index 2 dans les 2 s avant chacun des 6 frags au Ghost | **0 / 6** | sonde **[M]** |
| Orphelins de la porte bipède | 104, tous d'arme personnelle, index 0/1/4/5/6/7, aucun dans un épisode → `shotsNoRide 104` | sonde **[M]** |

**Conclusion sur 81c02726 [M]** : le tir du Ghost **n'entre jamais** dans le flux de tirs décodé.
Les trois hypothèses du superviseur sur la chaîne aval sont réfutées : (a) aucun événement d'arme de
véhicule ne porte un autre `FilmIndex` — il n'en existe aucun ; (b) horloge et bornes sont justes —
les 6 frags tombent dans les épisodes ; le pont `IndexParSlot` est juste. **(c) est la cause :
l'arme du Ghost n'émet rien que le décodeur lise.** La seconde porte (`vehicle_shots.go`) n'a rien
à rattacher.

### 1.2 Deux autres fichiers de faits (sonde, même instrument)

- `8a485699` (arène 8 joueurs, 9 familles) **[M]** : armes de véhicule décodées et publiées —
  Gungoose `0042678E` (8), Wasp `11725DC4` (4), Scorpion `49E40D17` (4), Wraith `121B4009` (3).
  Épisode de Ghost (42,4 s), deux de Banshee (12,8 s + 11,5 s), un de Chopper (10,6 s) : **0
  événement** de l'occupant, d'aucune classe.
- `4f77afc1` (BTB, index jusqu'à 31) **[M]** : décodées `C7D50912` (187), `121B4009` (180),
  `0BB6976B` (107), `0042678E` (95). **Aucun** événement pour les 14 frags au Ghost, le frag à la
  LAAG, le frag à la LMG du Falcon, les 2 frags à la tourelle plasma du Wraith (fenêtres de 2 s).

### 1.3 Le parc (111 documents, kill-feed de la copie de base)

`SP/sondes/tirs_vehicules/kills_vs_tirs.mjs` — frags de classe VÉHICULE, et part précédée d'un
tir d'arme de véhicule publié du tueur dans les 2 s **[M]** :

| Arme (tag de dégât) | frags | docs | couverts par un épisode du tueur | précédés d'un tir publié |
|---|---:|---:|---:|---:|
| Ghost `f712c64a` | 63 | 12 | 38 | **1** |
| Wraith `deffdc6b`/`7c4f8a28` | 47 | 3 | 32 | 13 (mortier : vol > 2 s) |
| Falcon tourelle latérale `674a7d69`/`a859230a` | 32 | 3 | 11 | 9 |
| Warthog LAAG `382cafaf` | 31 | 7 | 13 | **0** |
| Banshee `fa4fad21`/`53b1e6c5`/`a21bf18a` | 26 | 4 | 22 | 3 |
| Rockethog `5a4450e4` | 7 | 2 | 3 | 3 |
| Gungoose `00426796` | 5 | 2 | 1 | 0 |
| Scorpion `19bd6810`/`0bece71e` | 4 | 1 | 4 | 4 |
| Falcon LMG `77a61ef5`, tourelle plasma Wraith `3c7560e0` | 2 + 2 | — | — | 1 + 0 |

Le Ghost est piloté **1 522 s** (40 épisodes, 15 documents) pour **0 tir** publié.

### 1.4 La partition qui explique tout [M]

`V3F_TIRS_COVENANT_2026-09-02.md` §2/§4 donne, pour chaque arme de véhicule, la nature de son son de
tir (`snd!` = son d'un coup ; `lsnd` = son en BOUCLE, tir continu). Confronté au parc :

| Son de tir (V3F) | armes | tag vu dans un film ? |
|---|---|---|
| `lsnd` (boucle) | Ghost `00015435`, Banshee M1 `0000aa68`, Chopper `b40e9618`, LMG Falcon `00015cd3`, LAAG `0c6fd911`, Wasp `d3c407ed` | **0 sur 6, jamais** |
| `snd!` (coup) | Wraith `121b4009`, Rockethog `c7d50912`, Wasp `11725dc4`, Gungoose `0042678e` | **4 sur 4** |

Témoin indépendant, armes de joueur : le **Rayon de Sentinelle** (tir continu) est pris **540 fois**
dans le parc et tiré **0 fois** sur 190 615 tirs publiés (`perso_continu.mjs`) **[M]**.

---

## 2. Cause racine

### 2.1 Prouvée (sur faits persistés)

Le décodeur de production (`film/internal/grammar/fire_events.go:152-181`) ne lit que l'**événement
de tête** des paquets delta dont l'octet 0 vaut `0xD2`, à **offsets fixes** (`:163-169`), et jette
en silence tout record de moins de 113 bits (`:229-232`). **Les armes de véhicule à tir continu
(Ghost, canons de la Banshee, Chopper, LAAG, LMG du Falcon, tourelle plasma du Wraith) n'y figurent
pas** : zéro événement dans 10 épisodes de Ghost sur 3 films, zéro dans les 2 s de 20 frags au
Ghost, zéro publié sur tout le parc. Aucun lot aval (seconde porte, tables client, teintes, sons)
ne peut rendre un tir qui n'est pas dans les entrées de l'assemblage.

### 2.2 Où le film porte-t-il ces tirs ? — candidates classées [H], et la sonde qui tranche

1. **Un ÉTAT de tir, pas un événement par coup** [H, la plus probable] : son de tir en boucle
   (`lsnd`), 6/6 armes continues absentes, Rayon de Sentinelle identique. Candidats d'état :
   `ti=40 i18 unit-control` (commandes de l'unité), composants d'état de l'objet arme porté par le
   véhicule (`VEHICULES_ARCHETYPE_40.md`, jamais décodés).
2. **`action_weapon_fire` hors tête de liste** : la liste d'un paquet peut porter plusieurs
   événements (61 % des événements ne sont pas en tête, `.ai/V7.5/replay2d/
   RAPPORT_R7_TRAME_COMPLETE_2026-09-03.md` §4.3) ; la production n'en lit que la tête.
3. **`action_weapon_fire` en tête mais < 113 bits** (chemin court de la grammaire, `estCourt`,
   `r7_charges_lot6_research_test.go` `r7Charge36`) — écarté par la garde de taille sans compteur.

**Sonde proposée (NON lancée — interdit du brief ; décision utilisateur)** : 2 films ARÈNE,
`81c02726` puis `8a485699`, un à la fois, sous verrou.
- S1 — marcheur de liste complète existant (`r7_marche_liste_research_test.go`) sur les fenêtres de
  Ghost : tout événement (tête ou non) dont une référence de domaine 1 désigne le Ghost (769/771)
  ou son pilote (514/541) ; tout type 36 hors tête ; tout type 36 de tête écarté par la garde de
  113 bits, avec sa longueur.
- S2 — deltas des composants de l'entité Ghost (`ti=40`) et de son objet arme pendant l'épisode :
  quel composant bascule au début d'une rafale et à la cadence du tag (225 coups/min × 2 canons,
  V3F §3.3) ; témoin décalé de ±60 s.
- S3 si S1 et S2 sont négatifs : Ghidra (lecture seule) — chemin d'émission de
  `action_weapon_fire` pour un canon `lsnd` contre un canon `snd!`.
- Gate de la sonde : un signal dont une occurrence précède chacun des 6 frags de G MONEY dans les
  2 s, absent au témoin décalé, et nul hors de ses épisodes.
- Coût : deux décodages de film arène (minutes, RAM d'un film arène), aucune écriture d'artefact.

### 2.3 Défauts aval qui empêcheront le rendu même quand le signal sera lu [M]

1. **Tables client clées par des tags jamais vus** : sur les 11 tags de `vehicleShotFx.ts:74-88`,
   `vehicleShotSound.ts:78-115`, `vehicleWeaponMounts.ts:193-206`, **7 n'apparaissent dans aucun
   document** (Ghost, Banshee ×2, Chopper, Scorpion `00015cfa`, Falcon `00015cd3`, Wasp
   `d3c407ed`). Trois tags réellement publiés en sont **absents** : `0BB6976B` (51 tirs, tourelle
   latérale du Falcon), `49E40D17` (13, canon du Scorpion), `850902EF` (25, non identifié).
2. **Rockethog MUET** : `C7D50912` est le lance-roquettes du Rockethog (`vehi bcfb852f → weap
   c7d50912`, `WARTHOG_FINAL_2026-09-02.md` §1, `labels.tsv` `5a4450e4`). Le lot 5.8.4 l'a déclaré
   « LAAG / Gauss / roquettes non départagés » (`vehicleShotSound.ts:42`, `vehicleShotFx.ts:49`)
   et ne le fait sonner que pour la famille `rockethog`, qu'aucun châssis ne porte. Ses 127 tirs sont
   sur le châssis ENFANT `bcfb852f` (famille vide) ou sur `fe32c0f4` (`warthog`) : silence
   (`replaySound.ts:523-534`).
3. **Tirs et tireurs posés à la NAISSANCE de la tourelle** : les tourelles sont des vies `ti=40`
   ENFANTS sans aucun échantillon ; `vehiclePosAt` (`vehicle_shots.go:197-218`) et
   `vehiclePositionAt` (web) retombent sur la naissance. Écart au châssis porteur (`slot+1/+2`,
   même fenêtre) : tirs de roquettes **médiane 44,7 m, max 119 m** ; lance-grenades du Falcon
   **20,9 m / 82,6 m** ; épisodes d'artilleur (LAAG, Falcon, Wraith) : 12,6 à 39,8 m de médiane sur
   ~1 500 s (`tirs_enfants.mjs`, `episodes_enfants.mjs`).
4. **Teinte du Ghost** : `PLASMA = plasma_cool` (bleu) ; l'utilisateur veut du plasma ROUGE
   Banished — la teinte `plasma_hot` existe (`fxInk.ts:34`, Ravageur, Stalker, Canicule).

### 2.4 D'où viennent les 11 tags, et pourquoi ils ne correspondent pas [M/D]

Ils viennent du mode `tir-vehi` du lot V3F (2026-09-02) sur le module `any/globals/common` :
`vehi` → `weap` référencé en ligne → champ « Weapon Fire Sound ». L'espace est le bon (GlobalID
32 bits de tag `weap` = moitié haute de `Shot.w`, vérifié sur `0d76e8f1` pour `c7d50912` et
`11725dc4`). Ils ne correspondent pas pour deux raisons : **(1)** les armes à tir continu ne sont
pas dans le flux décodé, donc aucun tag ne peut y correspondre [M] ; **(2)** pour certaines armes à
coup, le film écrit un autre GlobalID (Scorpion : `49E40D17` observé contre `00015cfa`) — même
phénomène de module que les châssis (`vehicle_families.go:36-54`) [D]. Aucun lot n'a vérifié un tag
de Ghost, de Banshee ou de Chopper contre un document.

### 2.5 La famille « ? » [M]

Elle se décide dans `vehicleFamilyOf` (`vehicle_families.go:275`), table statique ; une valeur
absente rend la famille vide (repli nommé `repli_chassis_vehicule_marqueur_neutre`). Les châssis
« ? » fréquents sont les **tourelles enfants** que la table exclut délibérément
(`vehicle_families.go:57-61`, qui affirme « aucun n a ete observe au parc » — faux aujourd'hui) :
`dd7f9102` LAAG (45 vies, voisin `slot+1` = Warthog 44/45), `bcfb852f` roquettes (16, 15/16),
`1a043c29` et `f4c45d71` tourelles latérales du Falcon (25 et 24, chaîne `[1a043c29][f4c45d71]
[Falcon]`), `001b33fc` tourelle plasma et `233c877d` pièce du mortier du Wraith (14 et 17, chaîne
`[233c877d][001b33fc][Wraith]`), `0000d4ff`/`0000d500` enfants du Scorpion. Aucun n'a jamais un
échantillon de position.

---

## 3. Historique

| Date | Lot | Ce qui a été fait | Pourquoi le Ghost n'en a pas profité |
|---|---|---|---|
| 2026-09-02 | V3F | Tags `weap` et sons de tir Covenant tirés du module `common` | aucun tag confronté à un film |
| 2026-09-02 | V4 | Seconde porte (`vehicle_shots.go`), mesurée sur `0d76e8f1` (Warthog, Wasp : 23 tirs) | pas de Ghost dans la mesure |
| 2026-09-03 | montages | `vehicleWeaponMounts.ts` ; tags vérifiés sur 2 seulement (`c7d50912`, `11725dc4`) | Ghost ajouté sans observation |
| 2026-09-05 | sons de tir | 9 puis 10 armes câblées par tag V3F ; « constat annexe : les tirs véhicules ne sont PAS câblés » | câblé sur des tags jamais vus |
| 2026-09-08 | retours utilisateur | « `attachVehicleShots` ne publie que sur 2 artefacts sur 129 […] d'où l'absence totale d'effet UI/son » (`thought_log.md:6961-6963`) puis **« Point 8 — CLOS »** : sur Isolation (= `81c02726`) « deux Ghost courts », « le vrai facteur limitant est le nombre de chevauchées ARMÉES » (`thought_log.md:6821-6834`) | **fermeture fausse** : ces deux épisodes durent 78 et 80 s et portent 6 frags au Ghost |
| 2026-09-19/20 | 5.2a.5 | direction et style des tirs `v` DÉJÀ publiés, mesurés sur 4 documents ; `shotsNoRide` écarté comme « tirs perdus à la cuisson » (`PLAN_DECODEUR_FILM` l. 5943-5980) | n'a regardé que les tirs publiés |
| 2026-09-21 | 5.8.2 / 5.8.3 / 5.8.4 | style par tag, montages mesurés sur sprite, Warthog départagé par famille | 5.8.4 a rendu le Rockethog muet (§2.3.2) |

Aucun lot n'a jamais vérifié qu'un tir de Ghost existait dans un document réel.

---

## 4. Solution proposée

### Option A — recommandée : lire le tir dans le film, le nommer par un registre mesuré, le poser sur le véhicule qui le porte

**A0 — Sonde de localisation** (§2.2, recherche, taille M) : autorisation utilisateur requise
(décodage de 2 films arène). Tout le reste de A dépend de son verdict.

**A1 — Le tir décodé par la GRAMMAIRE** (Go, taille L) :
- `film/internal/grammar/fire_events.go` : le record 36 lu par le modèle M, plus d'offsets fixes.
  Le type 36 seul (l'octet `0xD2` couvre aussi le 37 `weapon_overheat`), les trois références (réf 0
  = unité tireuse, bipède OU véhicule, sonde lue), les gardes D/E/F, le tireur = champ R(5) sur
  **5 bits** quand il est présent, sinon l'unité de la réf 0 ; parcours de la liste complète si S1
  le demande ; nouveau canal du tir continu selon S2/S3.
- `film/replay/film_inputs.go`, `filmfacts_encode.go` / `_decode.go` : les faits portent l'indice
  5 bits, le slot de la réf 0 et le canal continu. **`SchemaDesFaits` et `grammar.Rev` montent :
  RE-DÉCODAGE de tout le parc (décision utilisateur, un film à la fois), puis republication.**

**A2 — La publication** (Go, `film/replay`, taille M) :
- `vehicle_shots.go` : un tir dont la réf 0 est un VÉHICULE se pose directement sur lui ;
  l'épisode ne sert plus qu'à nommer l'occupant.
- Tourelles enfants : parent lu dans le film si S2 le trouve (`object-parent-state` de l'entité
  `ti=40`), sinon repli NOMMÉ et compté « voisin `slot+1/+2` de même fenêtre » (mesure V8, 44/45
  sur le LAAG). Le tir, le tireur et son cône se posent sur le châssis porteur ; la tourelle n'est
  plus dessinée seule à sa naissance. Bonus : la tourelle nomme la VARIANTE du châssis
  (`bcfb852f` → `rockethog.png`, `64b925eb` → `warthog_gauss.png`).
- `Shot` gagne la clé d'arme de véhicule. **Montée de schéma : oui. Republication : oui.** Cette
  partie ne demande PAS de re-décodage : elle peut sortir avant A1 (republication depuis les faits).

**A3 — Le registre et le rendu** (config + Go + web, taille M) :
- `config/titles/halo_infinite/mappings/vehicle_weapons.toml`, clé = **tag observé dans un film** ;
  par entrée : véhicule, arme, tir continu ou coup, forme, teinte, son, montage, PREUVE (documents,
  nombre de tirs, tag de dégât co-occurrent). Publié dans le document ; les trois tables client
  indexées par `vehicleWeapTag` sont supprimées (0 code mort). Libellés FR + EN.
- Tir continu (si S2 rend un état) : rafale rendue à la cadence du tag pendant l'intervalle de tir,
  son en boucle (les `tir_RAFALE_*` de V3F existent) — rendu à valider par l'utilisateur.

**Tests et garde-rails**
- Rouge avant / vert après : sur un mini-film extrait de `81c02726` (fenêtre d'un frag au Ghost),
  un tir de Ghost décodé et posé sur le Ghost.
- Garde-rail « pas d'entrée jamais observée » : test Go qui exige que toute clé du registre figure
  dans `testdata/vehicle_weapon_tags_observed.json` (sortie datée de l'instrument de corpus, avec
  documents et comptes) ; et réciproquement, tout tag de classe véhicule observé ≥ 5 fois a une
  entrée ou une ligne `inconnu` motivée.
- Ratchet web : aucun littéral de tag d'arme ni appel `vehicleWeapTag(` dans `features/match-replay`
  hors du lecteur de registre. Ratchet Go : aucune lecture à offset fixe du record 36 hors grammaire.

**Gate sur DOCUMENTS RÉELS** (instruments : `kills_vs_tirs.mjs`, `tirs_enfants.mjs`,
`sweep_w.mjs`, à porter en test `research` du paquet)
- G1 `81c02726` : au moins un tir de Ghost publié dans les 2 s avant chacun des 6 frags de G MONEY
  (0/6 aujourd'hui), zéro hors de ses épisodes, posé à ≤ 3 m du sprite.
- G2 parc : frags de classe véhicule précédés d'un tir de l'arme du tueur ≥ 90 % par famille dont le
  signal est lu — Ghost 1/63, LAAG 0/31, Banshee 3/26, lance-grenades Falcon 9/32 aujourd'hui
  (fenêtre 4 s pour le mortier).
- G3 : tirs de tourelle à ≤ 2 m du châssis porteur en médiane (44,7 m aujourd'hui).
- G4 : 100 % des tirs d'arme de véhicule ont un style et un son (ou un silence DÉCIDÉ) venus du
  registre.
- G5 (avec A1) : joueurs d'index ≥ 16 avec des tirs (0/39 aujourd'hui) ; identifiants décalés = 0
  (1 205 aujourd'hui).

**Risques** : S1-S3 négatifs (le film ne porterait pas le tir continu) → seule l'option B resterait ;
la montée de `grammar.Rev` périme tous les faits (re-décodage de plusieurs heures, bombe RAM à
séquencer) ; A1 change l'attribution de nombreux tirs en BTB et avec bots (goldens à re-figer,
changement déclaré).

**Taille** : L au total (A0 M, A1 L, A2 M, A3 M).

### Option B — repli NOMMÉ, non recommandé seul : « un frag au véhicule = un tir »

Pour chaque frag dont la source de dégât est une arme de véhicule (faits `killsource`, déjà
persistés), publier un tir synthétique à l'instant du frag, posé sur le véhicule du tueur, avec le
style et le son de l'arme. Repli nommé et compté (`repli_tir_vehicule_par_frag`). Taille S, pas de
re-décodage, montée de schéma et republication depuis les faits. Limite : 6 éclairs sur tout le
match de G MONEY au lieu d'une rafale continue, jamais un tir raté. Utile en attendant A0/A1.

**Recommandation** : option A. Lancer A0 dès l'accord de l'utilisateur. Sortir tout de suite la
partie de A2/A3 qui ne demande pas de re-décodage (pose des tourelles, registre clé par tags
observés, Rockethog audible, Ghost en `plasma_hot`), puis A1 selon le verdict de A0.

### Table de vérité à faire valider

| Véhicule | Arme | Tag film | Observé (tirs / docs) | Porteur mesuré | Tir (V3F) | Effet proposé | Teinte proposée | Son proposé | Source |
|---|---|---|---|---|---|---|---|---|---|
| Ghost | canons plasma jumelés | `00015435` (V3F) | **0** / 0 (63 frags) | châssis `5b80c406` | continu (`lsnd`) | plasma, nez | **plasma_hot (rouge)** | `vehicle_shot_ghost` (validé 04/09) | V3F + choix utilisateur |
| Banshee | canons plasma | `0000aa68` (V3F) | 0 | `c6e79dcc` | continu | plasma, nez | plasma_hot ? | `vehicle_shot_banshee_m1` (validé 05/09) | V3F |
| Banshee | bombe à combustible | `0000aa69` (V3F) ; `850902EF` ? | 0 (`850902EF` : 25, dispersés) | `c6e79dcc` | coup | explosion | blast ? | `vehicle_shot_banshee_m2` | V3F ; `850902EF` à identifier |
| Wraith | mortier plasma (conducteur) | `121B4009` | 142 / 4 | châssis `ae845375` (140/142) | coup | orbe plasma | plasma_cool (actuel) ou plasma_hot ? | `vehicle_shot_wraith` ✓ | V3F + mesure |
| Wraith | tourelle plasma (artilleur) | inconnu | 0 (2 frags) | enfant `001b33fc` | ? | plasma | plasma_hot ? | aucun | `labels.tsv` |
| Chopper | canons avant | `b40e9618` (V3F) | 0 | `3d4a8a5a` | continu | ? | ? | `vehicle_shot_chopper` (validé) | V3F |
| Warthog | LAAG (artilleur) | `0c6fd911` (V3F le nomme « FalconLMG ») | 0 (31 frags) | enfant `dd7f9102` | continu | balistique | kinetic | aucune reconstruction | `WARTHOG_FINAL` §1 |
| Rockethog | lance-roquettes (artilleur) | `C7D50912` | 127 / 2 | enfant `bcfb852f` (125/127) | coup | roquette | blast ? (kinetic aujourd'hui) | `vehicle_shot_warthog_rocket` (validé 31/08, **muet depuis 21/09**) | `WARTHOG_FINAL` + mesure |
| Warthog Gauss | canon Gauss | `8647925a` | 0 (tourelle jamais vue) | enfant `64b925eb` | ? | ? | electric ? | aucun | `WARTHOG_FINAL` |
| Scorpion | canon principal | **`49E40D17`** (tables : `00015cfa`, jamais vu) | 13 / 2 (4/4 frags précédés) | châssis `f6f54e56` | coup | obus | blast ? | `vehicle_shot_scorpion` (existe, mal clé) | mesure |
| Scorpion | mitrailleuse (artilleur) | inconnu | 0 | enfant `0000d500` | ? | balistique | kinetic | ? | `labels.tsv` |
| Wasp | arme `11725DC4` | `11725DC4` | 4 / 1 | châssis `b65b3b4a` | coup (450 coups/min) | ? | kinetic | `vehicle_shot_wasp` ✓ | V3F |
| Wasp | arme `d3c407ed` | `d3c407ed` (V3F) | 0 | `b65b3b4a` | continu (600) | ? | ? | aucun | V3F |
| Gungoose | mitrailleuses avant | `0042678E` | 15 / 3 | châssis `af31ab1a` (dessiné Mongoose) | coup | balistique | kinetic | `vehicle_shot_gungoose` ✓ | `CONTACT_ARMES_GUNGOOSE` + mesure |
| Falcon | LMG latérale | `00015cd3` (V3F) | 0 (2 frags) | enfant `f4c45d71` | continu | balistique | kinetic | `vehicle_shot_falcon_lmg` (validé) | V3F + `labels.tsv` |
| Falcon | lance-grenades OU Gauss latéral | **`0BB6976B`** (absent des tables) | 51 / 3 (32 frags) | enfant `1a043c29` | coup | grenade ou Gauss ? | blast ou electric ? | aucun | mesure |
| Shade | tourelle plasma | inconnu | 0 | `000df0c4` | ? | plasma | plasma_hot ? | aucun | `labels.tsv` |

Les colonnes « effet », « teinte » et « son » sont des PROPOSITIONS, à trancher par l'utilisateur.
Seules les colonnes « tag », « observé » et « porteur » sont mesurées.

---

## 5. Questions pour l'utilisateur (réponse en une ligne chacune)

1. Le Ghost, les canons de la Banshee, le Chopper, la LAAG et la LMG du Falcon tirent-ils EN CONTINU tant que la gâchette est tenue (Ghost ≈ 7,5 coups/s, deux canons alternés) ?
2. Ghost : éclair plasma ROUGE (`plasma_hot`, la teinte du Ravageur) — OK ? Même teinte pour les canons de la Banshee, la tourelle du Wraith et le Shade ?
3. Mortier du Wraith : bleu (actuel) ou rouge ?
4. Chopper : ses canons sont-ils cinétiques (balistique) ou plasma ?
5. Rockethog : le son « roquettes » validé le 31/08 doit-il être rejoué (muet depuis le 21/09) ?
6. Warthog LAAG : prêter en attendant le son de la LMG du Falcon, ou silence jusqu'à une reconstruction ?
7. La tourelle latérale du Falcon qui tire `0BB6976B` : lance-grenades ou canon Gauss ? (un Falcon porte deux tourelles : la LMG `f4c45d71` et celle-ci)
8. Wasp : `11725DC4` (coup, 450 coups/min) est-il l'autocanon ou les missiles ?
9. Un artilleur (Warthog, Falcon, Wraith) doit-il être dessiné SUR le véhicule porteur, sans marqueur de tourelle séparé ?
10. Autorisez-vous la sonde A0 (décodage de `81c02726` puis `8a485699`, sans écriture d'artefact), puis, selon son verdict, le re-décodage du parc et sa republication ?

---

## 6. Hors périmètre découvert (noté, non traité)

1. **Repliement du tireur sur 4 bits (BTB)** [M] : `FireEvent.FilmIndex` (4 bits) sert au rattachement et c'est la seule valeur persistée
   (`filmfacts_encode.go:88-99`) ; `ShooterIndex5` n'est ni lu par le rejeu ni persisté. Dans les 6 documents dont un index
   dépasse 15, les **39/39** joueurs d'index ≥ 16 n'ont aucun tir à pied publié, et leurs tirs retombent sur l'index − 16. Les
   **434** tirs d'arme personnelle posés sur un véhicule viennent tous de ces documents (41 sur des Ghost, 50 sur des Banshee :
   artefacts de repliement).
2. **Records de tir hors cadrage canonique** [M] : **1 205** tirs publiés (24 documents) ont un identifiant d'arme lu décalé de 5 bits
   (garde R(5) absente) ou de 3 bits. Ils sont attribués à un index artificiel (2, 3, 6 ou 7), sans style ni son. 21 des 24
   documents contiennent des bots [D : tirs de bots].
3. `ScanFireEvents` accepte les octets de tête `0xD2` = types 36 ET 37 (`weapon_overheat`). Le 37 ne tombe aujourd'hui que par la
   garde de taille, pas par choix.
4. **Couverture des épisodes** [M] : 25 des 63 frags au Ghost ne sont couverts par aucun épisode du tueur (ex. Tupacamaru9556,
   `4f77afc1`, 10 frags au Ghost t 8706-9393, aucun épisode publié).
5. Deux commentaires faux : `vehicle_families.go:57-61` (« aucune tourelle enfant observée au parc ») et
   `vehicleShotSound.ts:42` / `vehicleShotFx.ts:49` (« c7d50912 ne départage pas LAAG / Gauss / roquettes »).
6. `replaySound.ts:532` prend la PREMIÈRE vie du slot porteur, pas celle qui couvre l'instant du tir.
7. Le Gungoose est dessiné en Mongoose (`af31ab1a` → `mongoose`) alors que son arme `0042678E` le désigne.
