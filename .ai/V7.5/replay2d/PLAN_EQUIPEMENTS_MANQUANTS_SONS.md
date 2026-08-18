# Plan — Equipements MANQUANTS : identifiants, noms, sons

> Ecrit le 2026-08-18. Demande utilisateur : les equipements que le rejeu ne nomme pas, ceux
> que le jeu a AJOUTES (« vers debut 2025 »), et les sons qui manquent a ceux qu'on nomme
> deja. Les deux sujets ne se separent pas — la chaine qui NOMME un `eqip` (`sofd -> sofa ->
> {string_id, eqip}`) et celle qui lui trouve un SON (`eqip -> snd! -> sbnk -> .wem`) partent
> du meme tag, et la seconde peut nommer ce que la premiere ne casse pas (une banque
> s'appelle `sb_007_abl_repairfield` en clair). Un seul lot.
>
> Worktree FRERE `C:\Users\Guillaume\Projects\LevelUp-wt-equip-sons`, branche `wt/equip-sons`,
> base `feat/v75` HEAD (`3203bb7b8`). Films et jeu par chemins ABSOLUS ; aucune base DuckDB.
> Contrat `plan-execution` + `CLAUDE.md`. Un commit par phase.

## Acquis — lus sur pieces avant d'ecrire ce plan

1. **Le manifeste** (`config/titles/halo_infinite/mappings/replay_labels.toml`) porte
   **21 lignes `[[equipment_objects]]`** (id, `family`, `name_id`, `provenance`, `kind`),
   dont **20 nommees** et une seule `other` : `0x4396db42`, provenance `sofa_anonyme`. Il
   porte aussi **1 ligne `[[objective_objects]]`** (le drapeau, `0x2a392328`, `ti=42`, une
   autre chaine — hors de ce lot).
2. **La chaine de nommage est etablie et instrumentee** (PLAN_NOMMAGE_EQIP_TRANSLOCATEUR,
   gates 0/1/2 PASSES le 2026-08-18) :
   `sofd` (palette du match) --rang--> `sofa` --+--> `string_id` (murmur3 x86_32 seed 0)
   --+--> `eqip` (l'objet du monde). Instruments versionnes :
   `himap/sonde_sofd_gamefiles_test.go`, `himap/sonde_eqip_gamefiles_test.go`,
   `himap/sonde_stringid_dico_test.go`, `himap/sonde_ti37_gamefiles_test.go`.
   Les modules n'ont **plus de table de chaines** (`stringsSize` = 0) : un `string_id` se
   CASSE par dictionnaire, il ne se lit pas.
3. **Le negatif de `0x4396db42` est deja chiffre** : `string_id` 0xb328c9fa resistant au
   dictionnaire a DEUX jetons (340 000 candidats) **et a TROIS** (51 millions, esperance de
   collision 0,07) ; modele `hlmt:eea39cf4` partage avec aucun `eqip` nomme ; dependances
   `gldf` + `lens` (objet LUMINEUX). Diagonale `i48` : rang 10 de la famille A a 86,4 % —
   son RATTACHEMENT est verrouille, son NOM ne l'est pas. L'utilisateur pense a l'« ecran de
   dissimulation » : **ce plan ne l'ecrit nulle part tant qu'une chaine ne le rend pas**.
4. **Le corpus mesure aujourd'hui** = les artefacts de rejeu sur disque
   (`C:\Users\Guillaume\Projects\LevelUp\data\cache\replays\halo_infinite\`, 33 JSON dont
   **12 portent des poses**). Releve du jour, schema 10, sans re-decodage :
   **21 identifiants, 3 206 poses**, dont `0x4396db42` = **104 poses / 9 films /
   26 deployees** (la planche du 18/08 disait 94 / 7 / 24 — l'ecart est la croissance du
   jeu d'artefacts, pas une divergence de methode).
   Le cache de films brut compte **951 dossiers**
   (`C:\Users\Guillaume\Projects\LevelUp\data\cache\film_chunks\<id>`), et une pose ne s'en
   tire que par un DECODAGE COMPLET (`filmdec.ScanFilmEquipmentPlacements` : bande de slots,
   `ScanFilmWorldObjects`, calibration MPP, oracle de position). Cf. decision 1.
5. **Les sons** : `RECETTE_SONS_ARMES.md` part du champ NOMME « Weapon Fire Sounds » du tag
   `weap` et ne connait PAS le groupe `eqip` (0 occurrence, aucun mode de
   `cmd/weapon-sounds`). Le negatif du lot R3-V (item R3.8, commit `fecbe67e1`) est ecrit
   dans `replaySound.ts` avec ses quatre chemins fouilles. Les tables de branchement :
   `EQUIPMENT_SOUND_STEMS` (camo, overshield — activate/deactivate) et
   `EQUIPMENT_PLACEMENT_SOUND_STEMS` (wall, sensor, threat_seeker — un stem de POSE), gardees
   par `replaySoundAssets.guard.test.ts` (manifeste = dossier d'assets, 0 asset mort, duree
   par categorie).
6. **Les sources audio sont sur le disque et NOMMEES par fichier** :
   `C:\Program Files (x86)\Steam\steamapps\common\Halo Infinite\Sound\win\SFX\*.pck`,
   **841 packs**, dont **un seul `abl`** : `sb_007_abl_repairfield.pck`. C'est deja la piece
   qui a confirme le rang 23 « champ de reparation » par une seconde chaine. Les autres
   equipements n'ont pas de pack : leurs `.wem` sont EMBARQUES dans les `sbnk`
   (`pc/globals/globals-rtx-new.module`, 7,24 Go, **jamais** en parallele d'un autre gros
   chargement). Outils presents : `_outils/vgmstream/vgmstream-cli.exe` (le seul a decoder le
   Vorbis Wwise) et `ffmpeg` (winget) pour l'egalisation.
7. **L'egalisation de reference** (lot R2-S, 18/08) : **-16 LUFS / plafond -1 dBTP par gain
   LINEAIRE strict** sur les 41 fichiers de `static/sounds/halo_infinite/`. Duree :
   armes/lancers/melee <= 1,2 s ; explosions et EQUIPEMENTS gardent la duree de leur source
   (plafond `SOUND_CUT_MAX_S`). Tout son ajoute par ce lot est un son d'EQUIPEMENT.

## Decisions tranchees AVANT toute mesure (seuils ecrits d'avance)

1. **LE CORPUS DE REFERENCE EST LE JEU D'ARTEFACTS, ET LE BUDGET DE DECODAGE EST DE
   45 MINUTES.** Les 951 films ne se croisent pas a la main : chaque film demande un decodage
   complet. Procedure : mesurer le cout d'UN film representatif, extrapoler a 951, et
   **ecrire le chiffre**. Si l'extrapolation depasse 45 min, le croisement plein-corpus est
   REFUSE et dit comme tel — jamais remplace par un echantillon choisi (biais de selection).
   Un croisement partiel n'est admis que si l'echantillon est defini par un critere
   INDEPENDANT de l'equipement cherche (p. ex. « les N premiers dossiers par ordre
   alphabetique »), et il est alors publie comme partiel avec son denominateur.
2. **UN NOM VIENT D'UNE CHAINE, JAMAIS D'UNE RESSEMBLANCE.** Trois chaines sont admises, et
   chacune a son seuil, ecrit ici :
   - **(a) dictionnaire murmur3** : un `string_id` casse NOMME. L'esperance de collision
     fortuite (`candidats x cibles / 2^32`) doit rester **< 0,10** ; au-dela, le resultat est
     du bruit et n'est pas ecrit. Sur une cible UNIQUE, la profondeur 3 reste legitime
     (51e6 / 2^32 = 0,012) ; sur les ~200 cibles d'un balayage global, la profondeur 2 est le
     plafond.
   - **(b) modele partage** (`hlmt` identique a un `eqip` nomme) : NOMME la famille, pas la
     variante — c'est deja la provenance `sofa_modele` du manifeste.
   - **(c) banque sonore nommee** : un `eqip` dont la chaine de sons atteint les `.wem` d'un
     `.pck` au nom explicite (`sb_007_abl_<nom>`) est nomme par ce nom **a deux conditions
     cumulatives** : le chemin est une suite de DEPENDANCES DE TAGS (aucun rapprochement par
     distance ou par frequence), et **aucun autre `eqip` du jeu n'atteint le meme pack**
     (temoin de selectivite). Sans le temoin, la coincidence n'est pas un nom.
3. **`0x4396db42` : verdict binaire.** Il est nomme par (a), (b) ou (c), ou il RESTE `other`
   avec son negatif enrichi. « Ecran de dissimulation » n'entre au manifeste que si une
   chaine le rend ; l'intuition de l'utilisateur est une PISTE a tester, pas une source.
4. **DATER LES TAGS : hypothese ecrite avant la mesure, et son controle.** Les modules ne
   portent pas de date de tag. Le seul signal structurel disponible est le MODULE D'ACCUEIL :
   `multiplayer-rtx-new`, `multiplayer_r1`, `multiplayer_r2`, `multiplayer_r3` ressemblent a
   des increments de livraison. **Hypothese H4** : un equipement recent vit dans un module
   d'indice plus eleve qu'un equipement de lancement. **CONTROLE, ecrit d'avance** :
   grappin / repulseur / detecteur de menaces / mur (lancement, 2021) doivent tomber dans un
   module d'indice INFERIEUR ou EGAL a celui du translocateur quantique / traqueur / champ de
   reparation (saisons 4-5). **Si le controle echoue, H4 est REFUSEE** et la datation se dit
   « non etablie » — la difference avec le manifeste reste alors le seul fait publiable
   (« present dans le jeu, absent du manifeste »), sans date.
5. **PERIMETRE DU MANIFESTE.** N'entrent en phase 2 que des identifiants NOMMES par une
   chaine. Un `eqip` du jeu qui n'apparait dans AUCUN artefact n'entre PAS dans
   `[[equipment_objects]]` : le garde-rail de parite est BILATERAL (toute ligne doit avoir un
   identifiant du corpus, `TestPariteObjetsEquipementDuCorpus`). L'inventaire des equipements
   du jeu hors corpus est une SORTIE DE MESURE (au CR et au plan), pas une ligne de manifeste.
6. **AUCUN SON INVENTE, AUCUN SON EMPRUNTE SANS PARENTE.** La regle du fichier
   `replaySound.ts` tient : une famille sans source reste MUETTE. Un emprunt n'est admis que
   s'il est adosse a une parente STRUCTURELLE ecrite (le precedent : traqueur = capteur, meme
   `hlmt` de famille, meme geste de pose). « Il reste un fichier disponible » n'est pas une
   raison.
7. **MEMOIRE ET PROCESSUS.** Un seul `go` a la fois ; un seul gros module en RAM
   (`pc/globals` = 7,24 Go **jamais** avec un autre) ; `GOCACHE` dans le scratchpad ; aucune
   base DuckDB ouverte ; jamais `git add -A`, jamais `git stash`, jamais de push.

## Phases

### Phase 1 — INVENTAIRE : tous les `eqip` du jeu, croises manifeste et corpus — CLOSE le 2026-08-18

Jeu mesure : build `269225.26.04.08.1618-1.hi_1_13_0` (Steam,
`C:\Program Files (x86)\Steam\steamapps\common\Halo Infinite`).

- [x] 1.1 **Instrument** : `himap/sonde_eqip_inventaire_gamefiles_test.go` — trois tests,
      lecture seule, un module a la fois, 2 a 4 s au total.
      `TestSondeEqipInventaire` (inventaire + croisements), `TestSondeEqipModulesParFamille`
      (hypothese de datation H4), `TestSondeEqipSignatureGlobale` (modeles partages et
      recensement des groupes de dependances). Il REUTILISE `lisEqip` et `balayeSofd` : aucun
      troisieme balayage ecrit.
- [x] 1.2 **Nommage global** : **116 tags `eqip`**, **337 `sofa`**, **185 `sofd`**,
      **259 identifiants de chaine dont 14 casses** (335 264 candidats, esperance de collision
      **0,0202** — sous le seuil de 0,10 de la decision 2a). Les 14 noms du jeu :
      `mobility_sprint`, `melee_default`, `ability_location_sensor`, `ability_deployable_wall`,
      `ability_grapple_hook`, `ability_evade`, `ability_knockback`, `active_camo`,
      `powerup_overshield`, `quantum_translocator`, `threat_seeker`, `repair_field`,
      **`regen_field`**, `unsc_thruster`.
      **90 `eqip` sont rattaches a un `sofa`** — ce sont les equipements de JOUEUR ; les
      26 autres sont des objets qu'aucune palette n'expose.
- [x] 1.3 **Croisement MANIFESTE** : 21 lignes / 21 identifiants du corpus, **0 sans tag dans
      le jeu installe** (la parite tient dans les deux sens). **73 `eqip` sont AU JEU SEUL** :
      rattaches a un `sofa`, absents du corpus.
      **CE QUE SONT CES 73, ET POURQUOI CE NE SONT PAS 73 EQUIPEMENTS MANQUANTS.** Un `sofa`
      porte DEUX `eqip` : l'APPAREIL (celui qui vole, qui porte la banque sonore propre) et
      l'OBJET DU MONDE (celui que le film cree, donc celui du corpus). Sur les 73, la moitie
      sont les appareils des equipements DEJA au manifeste — `5c8e2316` va avec le champ de
      reparation `32d97758`, `a1344fc2` avec la balise `730dc70f`, `1e79ebda` avec le
      repulseur `7ca85adc`, `5f5f6fef` avec le capteur `72b63d69`, `2f3f467b` avec le
      traqueur `4744d742`, `c12e5469` avec le mur `2974c233`, `aceaf8f2` avec le camouflage,
      `fd8a47af` avec le surbouclier. **Un seul NOM du jeu manque vraiment au manifeste :
      `regen_field`** (`sofa 457923a0`, palette `13c097ed` rang 10, `eqip` `0dc0d5b4` +
      `03bad9e7`) — jamais vu au corpus, donc PAS de ligne de manifeste (decision 5).
- [x] 1.4 **Croisement CORPUS** : mesure du jour sur les artefacts de disque
      (`data/cache/replays/halo_infinite/`, 33 JSON, **12 portent des poses**) —
      **21 identifiants, 3 206 poses**. `0x4396db42` : **104 poses / 9 films / 26 deployees**
      (la planche du 18/08 disait 94 / 7 / 24 : c'est la croissance du jeu d'artefacts).
      **LE CROISEMENT PLEIN-CORPUS EST REFUSE, ET VOICI SON CHIFFRE** (decision 1, budget
      45 min). Un decodage mesure sur `000d5950` (27 chunks, film d'arene, le plus petit des
      trois calibres) coute **53,9 s** pour la seule calibration — `ScanFilmEquipmentPlacements`
      ajoute par-dessus le balayage des records de creation. Les 951 films du cache pesent
      **22,9 Go** (`06dfe6d9` en fait 46, `00ba2e1c` 28). Extrapolation PLANCHER :
      951 x 53,9 s = **14,3 heures**, et le plancher est optimiste (les films BTB ont 45 chunks
      contre 27). Le budget est depasse d'un facteur 19 : le croisement ne se fait pas, et il
      n'est pas remplace par un echantillon (biais de selection, decision 1).
- [x] 1.5 **VERDICT `0x4396db42` : IL RESTE `other`.** Les trois chaines de la decision 2 ont
      ete parcourues, et les trois rendent un NEGATIF — chacune avec son denominateur :

      **(a) dictionnaire — NEGATIF, temoins a l'appui.** Un dictionnaire ELARGI a ete construit
      pour ce lot (`himap/sonde_eqip_anonyme_test.go`, 31 prefixes x 176 jetons a deux jetons =
      **1 898 966 candidats**, esperance de collision **0,0022** sur 5 cibles). Il ajoute les
      champs lexicaux qui manquaient au dictionnaire global : occultation (`veil`, `fog`,
      `mist`, `curtain`, `obscure`, `conceal`, `chaff`, `haze`...), projection (`emitter`,
      `projector`, `dispenser`, `canister`, `dome`), brouillage (`jammer`, `scrambler`,
      `interference`), lumiere dure (`hardlight`, `photon`, `prism`). **Le controle passe
      10 sur 10** : `repair_field`, `quantum_translocator`, `threat_seeker`, `active_camo`,
      `powerup_overshield`, `ability_location_sensor`, `ability_deployable_wall`,
      `ability_grapple_hook`, `ability_evade`, `ability_knockback` sortent tous de CE
      dictionnaire — son negatif vaut donc quelque chose. **`0xb328c9fa` ne sort pas**, ni les
      quatre autres anonymes du corpus (rangs 19 a 22). Le dictionnaire global l'avait deja
      refuse a profondeur 3 (49 954 499 candidats, esperance 0,070) : deux negatifs
      independants.
      **CE QUE CELA REFUTE, ET IL FAUT LE DIRE** : `shroud_screen`, `smoke_screen`,
      `stealth_screen`, `cloak_screen`, `camo_screen` — avec les 31 prefixes, dont `ability_`,
      `equipment_`, `abl_`, `deployable_` — SONT tous dans l'enumeration. Aucun ne hache vers
      `0xb328c9fa`. L'« ecran de dissimulation » n'est donc pas ecarte comme objet, mais
      **aucun de ses noms plausibles n'est le nom de ce tag**.

      **(b) modele partage — NEGATIF sur le DENOMINATEUR COMPLET.** La sonde precedente ne
      comparait qu'aux 21 identifiants du corpus ; celle-ci compare aux **116 tags `eqip` du
      jeu**. `0x4396db42` porte `hlmt:eea39cf4` et il est **le seul** — la ou 16 modeles sont
      partages par deux `eqip` ou plus (`c62d74c3` capteur x2, `75070361` grappin x2,
      `99953db8` propulseur x2, `2e76f0a9` mur x4, `d8ca727d` panneaux x5, `c4b1aebd`
      surbouclier+camouflage x6). Son frere de `sofa`, `0x4eebcb18`, n'a **aucun** `hlmt` : il
      porte un `proj` (projectile). Le rang 10 est donc un objet LANCE qui deploie un objet
      lumineux (`gldf` + `lens`) — une description, pas un nom.

      **(c) banque sonore nommee — NEGATIF, avec un INSTRUMENT CALIBRE.** La chaine
      `eqip -> effe -> snd! -> sbnk -> .wem -> pack nomme` a ete ouverte pour ce lot
      (`cmd/weapon-sounds`, modes `eqip-sons` et `eqip-banks`) et elle est **calibree par trois
      controles independants** : bank `5724312f` (SELECTIVE, 1 `eqip` = `5c8e2316`) tombe dans
      `sb_007_abl_repairfield` — le champ de reparation se nomme lui-meme ; bank `1db55179`
      tombe dans `sb_010_grn_cv_plasmagrenade` (`eqip caaadcb0` = grenade plasma) ; bank
      `2f019657` dans `sb_010_grn_un_lightninggrenade` (`eqip aada07f3` = grenade Dynamo,
      « lightning » etant exactement ce que `damagetag/data/labels.tsv` en dit).
      **Le rang 10 a bien sa propre banque** — `92c830f5`, atteinte par le SEUL `4eebcb18`,
      **38 `.wem` embarques** — mais **aucun de ces 38 ne vit dans un `.pck` nomme**. Elle ne
      nomme donc rien. Le negatif est celui d'un instrument qui marche, pas d'un instrument
      absent.

      **CE QUE LE LOT VERROUILLE QUAND MEME** : `0x4396db42` et `0x4eebcb18` sont les deux
      `eqip` du `sofa` `0xeb500815`, rang 10 des palettes `d91958af` ET `03137359` — et le
      rang 10 de la TROISIEME palette de famille A (`13c097ed`) est occupe par un AUTRE `sofa`
      (`457923a0` = `regen_field`). Le rang 10 n'est donc pas un equipement unique selon la
      palette : c'est une position, pas une identite. `regen_field` n'est PAS le rang 10
      anonyme.
- [x] 1.6 **EQUIPEMENTS « DEBUT 2025 » : H4 EST REFUTEE, LA DATATION N'EST PAS ETABLIE.**
      Le controle ecrit d'avance (decision 4) ne peut meme pas etre applique : **il n'y a que
      DEUX combinaisons de module d'accueil** sur les 90 `eqip` de `sofa` — `globals` (87) et
      `common` (3). Les quatre modules `multiplayer`, `multiplayer_r1`, `_r2`, `_r3` ne
      portent **aucun** tag `eqip` (0 sur 116). Equipements de lancement et equipements de
      saison 5 vivent dans le MEME fichier : le module d'accueil ne separe rien. Les
      horodatages de fichier ne separent rien non plus (tous a l'instant de l'installation,
      2026-07-11 17:23). **Aucune date de tag n'est lisible hors ligne** — ce lot ne date donc
      aucun equipement, et le dit.
      **CE QUE LE LOT ETABLIT A LA PLACE** : la difference JEU \ MANIFESTE, sans date.
      Un seul equipement NOMME du jeu est inconnu du manifeste (`regen_field`), et il l'est
      parce que le corpus ne l'a jamais montre. Les autres `sofa` inconnus du manifeste ont
      des identifiants de chaine non casses (rangs 13 a 18 et 24 a 26 de la famille A, plus
      les palettes `4519617c`, `68cb90bc`, `81d9aeed`, `a761a216`, `fa8a7615`, `51e60c5a`,
      `758be0bc`) : ils existent, ils ne se nomment pas, et aucun n'a d'objet au corpus.

**Gate 1 : PASSE.** `go vet ./internal/himap/ ./cmd/weapon-sounds/` verts, les trois sondes
tournent (2 a 4 s), le dictionnaire elargi passe ses 10 temoins, chaque identifiant cite a son
denominateur, aucun nom ecrit sans sa chaine. Le seul `other` du manifeste le reste, et son
negatif est desormais triple.

### Phase 2 — MANIFESTE : les identifiants nommes entrent, avec leurs garde-rails

- [ ] 2.1 Pour chaque identifiant que la phase 1 NOMME et qui est au corpus : ligne
      `[[equipment_objects]]` complete (`id`, `family`, `name_id` si `sofa_string_id`,
      `provenance`, `kind`). Familles : reutiliser la liste fermee existante ; n'en creer une
      que si la chaine impose un objet nouveau, et l'ecrire au commentaire du fichier.
- [ ] 2.2 Libelles FR/EN : le manifeste ne porte aujourd'hui de `en`/`fr` que sur
      `[[objective_objects]]` ; les familles d'equipement sont libellees cote web. **Verifier
      sur pieces** ou vit le libelle d'une famille de pose avant d'en ajouter un — ne pas
      creer une seconde source (regle CLAUDE.md n° 6).
- [ ] 2.3 Icone : n'ajouter `icon = "hud/..."` que si l'asset existe reellement sous
      `static/weapons-assets/halo_infinite/hud/` (verification sur pieces, pas de chemin
      devine).
- [ ] 2.4 Loader : `mappings/loader_replay_labels_equipment.go` +
      `verifieProvenanceEquipement` — les invariants (famille nommee => provenance de
      structure ; `sofa_anonyme`/`aucune` => famille `other` ; `sofa_string_id` => `name_id`)
      doivent rester FATAUX et couvrir toute famille nouvelle.
- [ ] 2.5 Tests : `replaylabels/catalog_test.go` (dont `TestPariteObjetsEquipementDuCorpus`)
      et les tests du loader passent. Si la phase 1 ne nomme RIEN de nouveau, cette phase est
      statuee `[!]` avec sa justification — pas de ligne inventee pour la remplir.

**Gate 2** : `go build ./...`, `go vet ./...`,
`go test ./internal/games/... ./internal/himap/...` (hors `_gamefiles`), golangci 0.
Commit `feat(v7.5-rejeu-eqip): ...`.

### Phase 3 — SONS : adapter la recette des armes au groupe `eqip`

- [ ] 3.1 **Cartographier avant de coder** (lecon n° 1 de la recette) : sur quoi un `eqip`
      accroche-t-il un son ? Recensement des dependances par GROUPE sur tous les `eqip`
      (`snd!`, `lsnd`, `sbnk`, `effe`, autres) avec leurs effectifs. Le resultat DECIDE de la
      suite ; s'il n'y a aucun lien de tag vers le son, la phase s'arrete la, en negatif ecrit.
- [ ] 3.2 **Chaine `eqip` -> evenements -> banks -> `.wem`** : etendre `cmd/weapon-sounds`
      d'un mode qui parte du groupe `eqip` au lieu du champ « Weapon Fire Sounds » du `weap`.
      Reutiliser tout l'aval existant (`lien.go`, `bank.go`, `arbre.go`, `conteneurs*.go`,
      `pck.go`) — ne rien reimplementer. Deux passes, deux processus, un JSON entre les deux
      (contrainte memoire).
- [ ] 3.3 **Cibles** : balise du translocateur (`translocator_beacon`), champ de reparation
      (`repair_field`), l'ecran de dissimulation SI la phase 1 le nomme, et **toute famille du
      manifeste encore muette** (aujourd'hui : grapple, thruster, repulsor, repair_field,
      translocator_beacon, powerup_* ne sonnent pas a la pose ; l'inventaire du manifeste
      tranchera la liste exacte).
- [ ] 3.4 **Extraction** : `.wem` -> `.wav` par `vgmstream-cli.exe` (ffmpeg NE decode PAS le
      Vorbis Wwise). Fichiers de travail dans le scratchpad, jamais dans le depot.
- [ ] 3.5 **Egalisation** : -16 LUFS / plafond -1 dBTP par gain LINEAIRE strict (lot R2-S).
      Duree : categorie EQUIPEMENT, donc pas de troncature a 1,2 s, plafond `SOUND_CUT_MAX_S`.
- [ ] 3.6 **Branchement** : `EQUIPMENT_PLACEMENT_SOUND_STEMS` (geste de pose) et/ou
      `EQUIPMENT_SOUND_STEMS` (episode d'etat) selon ce que le manifeste dit de la famille —
      `kind = deployed` / `carried` et l'existence d'un debut ET d'une fin mesures. Le
      garde-rail `replaySoundAssets.guard.test.ts` doit rester vert (0 stem sans fichier,
      0 fichier sans stem, durees par categorie).
- [ ] 3.7 **Ce qui n'est pas trouve est un NEGATIF ECRIT** : chemins fouilles, denominateurs,
      et la condition de reprise. Mettre a jour le commentaire de `replaySound.ts` si le
      negatif du 18/08 est desormais faux ou incomplet (une doc inversee est un anti-pattern
      liste dans `CLAUDE.md`).

**Gate 3** : gates Go (build, vet, tests `internal/games` + `internal/himap` hors
`_gamefiles`, golangci 0) ; gate web si `apps/web/` ou `static/sounds/` sont touches
(`npm ci` dans le frere, purge de `.tmp`, `tsc`, lint, `vitest` match-replay).
Commit `feat(v7.5-rejeu-eqip): ...` ou `mesure(...)` selon l'issue.

## Regles dures de ce lot

Un nom vient d'une chaine ou n'existe pas. Un son vient du jeu ou n'existe pas. Les seuils
sont ecrits avant la mesure et ne bougent pas apres. Zero fix hors perimetre : toute
decouverte va dans la section ci-dessous, pas dans le diff. Ni journal ni registre — les
textes partent au compte rendu.

## Journal d'execution

(rempli phase par phase)

## Decouvertes — notees, NON traitees
