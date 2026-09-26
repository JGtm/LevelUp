# Plan — Nommer TOUS les objets d'equipement par la structure du jeu (`eqip` -> `sofd` -> nom), et le translocateur quantique

> Ecrit le 2026-08-18. Question utilisateur : « pour les `other` on n'a pas des pistes pour
> resoudre ce qu'ils sont exactement ? Pour le translocateur quantique tu peux investiguer ? ».
> Suite de PLAN_POSES_EQUIPEMENT_PUBLICATION (`caca2289b`, schema 9). Branche `feat/v75`,
> worktree principal (films + installation du jeu), contrat `plan-execution`. Lot de MESURE +
> MANIFESTE ; le rendu est le lot UI parallele (`PLAN_UI_POSES_EQUIPEMENT.md`).
>
> **CORRECTION DU 2026-08-18, a lire avant le reste** : la chaine annoncee ci-dessous
> (`eqip` GlobalID -> entree `sofd` -> string_id) N'EST PAS celle du jeu. Le bloc `sofd` ne
> reference AUCUN `eqip` : il reference un `sofa` par rang, et c'est le `sofa` qui porte a la
> fois l'identifiant de chaine ET les references `eqip`. La chaine reelle est
> `sofd -> sofa -> {string_id, eqip}` — exactement le maillon que le chantier voisin traverse
> pour l'armement de vehicule (`vcdd -> sofd -> sofa -> uwfa -> weap`). Le plan est conserve
> tel qu'ecrit, avec sa correction : l'erreur etait d'avoir lu le §9 comme si `entree+0x18`
> pointait la definition finale, quand il pointe le maillon suivant.

## La piste, et pourquoi elle est plus forte que la diagonale

- Le lot precedent a nomme par DIAGONALE `id x rang i48 du poseur` (>= 85 %) : deux
  identifiants passent, quatre echouent entre 75 et 82 % faute de denominateur `i48` sur BTB,
  et **21 identifiants sur 21 se resolvent pourtant en tags `eqip` du jeu**. Le nom est donc
  DANS LE JEU, pas dans la statistique.
- RECETTE_LOADOUT §9 : le rang de capacite est resolu a l'execution par `FUN_1407E7648`, qui
  parcourt le bloc du groupe `sofd` (donnees a `tag+0x28`, nombre a `tag+0x38`) et compare
  **`entree+0x18` au HANDLE DE DEFINITION**. §13 : les entrees portent des identifiants de
  chaine dont le murmur3 rend `ability_deployable_wall`, `ability_location_sensor`,
  `ability_grapple_hook`, `ability_evade`, `ability_knockback`, `powerup_overshield`,
  camouflage, translocateur (`rang 11`), traqueur, champ de reparation.
- Les 4 identifiants « plats » (sans poseur) sont eux aussi des `eqip` : vraisemblablement les
  POWER-UPS poses par la carte (surbouclier, camouflage) et autres objets d'equipement du
  decor — le meme chemin les nomme, et cela ouvre l'item « objets poses par la carte » du
  cahier des charges.

## Decisions tranchees avant execution

1. **La chaine structurelle devient la SOURCE du nom** ; la diagonale `i48` devient un
   CONTROLE (elle doit CORROBORER : un `eqip` nomme « mur » doit avoir sa diagonale sur les
   rangs mur quand le denominateur existe ; un desaccord franc = on n'ecrit PAS le nom et on
   le dit). Le manifeste `[[equipment_objects]]` porte la provenance : `structure` (chaine
   sofd) + `diagonale` (corroboree / non mesurable / contredite).
2. **Familles fermees, etendues** : `wall`, `sensor`, `translocator_beacon`, `repulsor`,
   `grapple`, `thruster`, `powerup_overshield`, `powerup_camo`, `powerup_other`, `other`. Un
   nom murmur3 non casse reste `other` avec son GlobalID.
3. **Le translocateur** : deux signaux a mesurer — (a) sa BALISE est un objet `ti=37`
   (identifiant `eqip` nomme translocateur par la chaine) ; (b) le RETOUR est un saut de
   position du poseur (discontinuite > X m en une frame, vie continue, arrivee a moins de Y m
   de la balise). Les seuils sont ecrits AVANT la mesure : X = 4 m en 100 ms (aucun deplacement
   a pied ni sprint ne le fait), Y = 2 m. Temoin : les sauts des vies SANS balise (spawn exclu).
4. Un seul gros module en memoire ; un seul decodage filmdec par process ; aucune base en
   ecriture ; JAMAIS `git add -A` ; jamais d'attente passive.

## Phases

### Phase 0 — LA CHAINE : `eqip` -> `sofd` -> nom — CLOSE le 2026-08-18

- [x] 0.1 **LA CHAINE PASSE PAR `sofa`, PAS DIRECTEMENT PAR `eqip`** — et c'est la correction
      qui debloque tout. Le bloc de palette d'un `sofd` est un data-block de pas **32 octets**
      (608/19, 544/17, 672/21, 704/22, 864/27 sur les palettes de `globals` : le pas `0x20` du
      decompile), chaque entree portant une reference de tag serialisee de 28 octets
      (`+0x08` GlobalID, `+0x14` fourCC du groupe) suivie de deux octets. Le fourCC vaut
      **`sofa`** sur 100 % des entrees des neuf palettes lues — jamais `eqip`.

      **L'identifiant de chaine est dans le `sofa`, a `+0x10` de son bloc racine.** L'offset
      n'est pas devine : cinq egalites exactes avec les noms que la RECETTE §13 avait obtenus
      par un autre chemin le fixent.

          rang 0  0xf08fa6e6 = mobility_sprint          rang 4  0x87b1d7a4 = ability_grapple_hook
          rang 1  0x566bb170 = ability_location_sensor  rang 5  0xed76a664 = ability_evade
          rang 2  0xedebd7b7 = ability_deployable_wall

      **CONTROLE GRATUIT, et il passe** : l'octet `+0x1d` de chaque entree de palette vaut 1
      pour les capacites et **0 pour les trois rangs exacts** que le §13 appelle « categorie
      nulle » (0 `mobility_sprint`, 3, 7 `melee_default`). Trois rangs designes d'avance, trois
      zeros.

      **LA PALETTE DE LA FAMILLE A EST DESORMAIS LUE, PAS DEDUITE** (`sofd 0xd91958af`,
      27 rangs). Elle reproduit les DIX rangs que le §13 nommait, sans en contredire un :

          rang  0  mobility_sprint          rang 11  quantum_translocator   <== LA CIBLE
          rang  1  ability_location_sensor  rang 12  threat_seeker
          rang  2  ability_deployable_wall  rang 23  repair_field
          rang  4  ability_grapple_hook     rang  8  active_camo
          rang  5  ability_evade            rang  9  powerup_overshield
          rang  6  ability_knockback        rang  7  melee_default

      Les modules n'ont plus de table de chaines (`stringsSize` = 0 sur les neuf modules
      indexes) : un `string_id` se CASSE par dictionnaire murmur3, jamais ne se lit. Le
      dictionnaire s'arrete a DEUX jetons apres le prefixe, et c'est une decision chiffree :
      a trois jetons il compte 25 millions de candidats contre ~200 cibles, soit 1,2 collision
      fortuite attendue — il en a rendu exactement une, visible a l'oeil nu
      (`ability_punch_detector_thruster`). A deux jetons, 340 000 candidats, esperance 0,02, et
      **tous** les noms etablis par ailleurs sortent quand meme.

      Instruments versionnes : `himap/sonde_sofd_gamefiles_test.go`,
      `himap/sonde_eqip_gamefiles_test.go`, `himap/sonde_stringid_dico_test.go` (dont
      `TestDicoStringIDConnus`, garde-rail du dictionnaire qui tourne PARTOUT — il ne touche pas
      aux fichiers du jeu).

- [x] 0.2 **20 DES 21 IDENTIFIANTS SE NOMMENT PAR LA STRUCTURE**, contre 2 par la diagonale.
      Quatre chemins, et chacun etablit autre chose :

          provenance       n   ce qu'elle etablit
          sofa_string_id  12   le `string_id` du `sofa` est casse : le nom est celui du jeu
          sofa_modele      4   `string_id` non casse, mais MEME `hlmt` qu'un `eqip` nomme
          sofa_parent      2   engendre par l'`eqip` d'un `sofa` nomme (`eqip -> eqip`)
          gggl_entree      4   entree de la liste des grenades du jeu
          sofa_anonyme     1   rattache, non nomme -> reste `other`

      **LES QUATRE « PLATS » SONT LES QUATRE GRENADES.** Le `gggl` (0x00000057) porte un bloc de
      832 octets = 4 x 0xd0, et les quatre identifiants sans diagonale sont exactement les
      quatre PREMIERES references `eqip` de ce bloc, aux offsets 0x4c, 0x11c, 0x1ec, 0x2bc — la
      periode 0xd0 est celle que le chantier kill feed avait mesuree de son cote
      (ETAT_DE_L_ART_KILLWEAPON 7ter.48, « 2 couples par entree, ecarts 0x38/0x98 »). L'ordre
      des entrees EST le rang de type du depot (`filmdec.GrenadeTypeIDsByRank`) et les banques
      sonores de `damagetag/data/labels.tsv` le confirment entree par entree :

          entree 0  0xbcabbe43  kineticunsc        -> grenade_frag     933 poses · 9 films
          entree 1  0xcaaadcb0  plasma             -> grenade_plasma   293 · 9
          entree 2  0xaada07f3  shock / lightning  -> grenade_dynamo   177 · 6
          entree 3  0x0f5716ff  kineticbanished    -> grenade_spike    306 · 8

      **UNE AFFIRMATION DU LOT PRECEDENT EST REFUTEE PAR LES ARTEFACTS** : « les 4 identifiants
      plats = objets du monde SANS POSEUR » est faux. Sur les trois artefacts sur disque, ces
      poses ont un poseur a moins de 3 m dans 96 a 100 % des cas (57/59, 49/49, 46/48 sur
      `000d5950`) et un cap de visee dans 83 a 94 %. « Plat » ne voulait pas dire « sans
      poseur » : cela voulait dire que le RANG DE CAPACITE du poseur ne dit rien — ce qui est
      exactement attendu d'une grenade, dont le type est independant de la capacite d'armure de
      celui qui la lance. La distance mediane a leur poseur (0,001 a 0,004 de l'AABB, et ces
      poses font 52 a 63 % du total) dit qu'elles naissent COLLEES a un bipede.

      Les deux `sofa_parent` sont les PANNEAUX DEPLOYES du mur : `0x686b40c9` est engendre par
      `0xc12e5469` (le premier `eqip` du `sofa` `ability_deployable_wall`) et `0x528fce46` par
      `0x37c87a13` (celui du `sofa` du rang 19). Tous deux portent un `bloc` (objet de decor) la
      ou l'appareil lance porte un `hlmt` d'equipement.

      Les quatre `sofa_modele` sont les rangs **19 a 22** de la famille A, et le partage de
      modele les nomme sans dictionnaire — en tombant EXACTEMENT sur ce que la verite terrain
      Theater du 2026-07-27 disait, sans que la structure la consulte :

          rang 19  0x8e2dc574  hlmt 2e76f0a9  = celui de ability_deployable_wall   (Theater : mur)
          rang 20  0x8c77ffe7  hlmt 75070361  = celui de ability_grapple_hook      (Theater : grappin)
          rang 21  0xeef5d48d  hlmt 99953db8  = celui de ability_evade             (Theater : propulseur)
          rang 22  0x72199cba  hlmt c62d74c3  = celui de ability_location_sensor   (Theater : capteur)

      **LE SEUL `other` RESTANT** est `0x4396db42` (51 poses, 5 films), rattache au `sofa`
      `0xeb500815` — le rang 10 de la famille A, le trou le plus frequent du corpus (67 lectures
      `i48`). Son `string_id` 0xb328c9fa resiste au dictionnaire y compris a TROIS jetons
      (51 millions de candidats sur 6 cibles, esperance de collision 0,07 : le negatif est
      solide, pas un manque d'effort), et son `hlmt:eea39cf4` n'est partage avec aucun `eqip`
      nomme. Ses dependances `gldf` + `lens` le rangent parmi les objets LUMINEUX, comme le
      surbouclier, le camouflage et le translocateur — piste, pas nom.

- [x] 0.3 Manifeste `[[equipment_objects]]` reecrit : **21 lignes** (id, family, name_id,
      provenance), 15 familles fermees au lieu de 3, et DEUX invariants nouveaux, tous deux
      fataux (`mappings/loader_replay_labels_equipment.go`, `verifieProvenanceEquipement` —
      section EXTRAITE dans son propre fichier, l'hote franchissait les 500 lignes) :
      une famille nommee EXIGE une provenance de structure, et `sofa_anonyme` / `aucune`
      EXIGENT la famille `other` — dans les deux sens, sans quoi `provenance` serait un
      commentaire. `provenance = sofa_string_id` exige en outre son `name_id`.
      Garde-rail de PARITE BILATERAL : `replaylabels/catalog_test.go`,
      `TestPariteObjetsEquipementDuCorpus` — tout identifiant du corpus a une ligne, toute ligne
      a un identifiant du corpus.

**Gate 0 : PASSE.** 20 identifiants sur 21 nommes par la structure du jeu (contre 2 par la
diagonale), 0 contradiction, 1 `other` documente avec son negatif chiffre.

### Phase 1 — CONTROLE par la diagonale, et distribution nommee — CLOSE le 2026-08-18

- [x] 1.1 **LA DIAGONALE PASSE 9 SUR 9.** Rejouee sur DEUX films qui couvrent les deux jeux de
      rangs du corpus — `000d5950` (rangs 19-22) et `06dfe6d9` (famille A, 892 poses) —
      l'instrument `equipment_creation_owner_test.go` rend, pour chaque identifiant, la
      distribution des rangs `i48` de son poseur. Sur les rangs CONNUS (les lectures `-1` sont
      comptees a part, jamais confondues), le rang dominant est CELUI DE LA STRUCTURE, sans
      exception :

          film      identifiant  structure                    diagonale (rangs connus)
          000d5950  0x8c77ffe7   rang 20 grapple              20:20  19:1        95,2 %
          000d5950  0x72199cba   rang 22 sensor               22:12  21:1        92,3 %
          000d5950  0x8e2dc574   rang 19 wall                 19:9   20:1        90,0 %
          000d5950  0xeef5d48d   rang 21 thruster             21:19  19/20/22:3  86,4 %
          000d5950  0x528fce46   rang 19 wall (parent)        19:8   21/22:2     80,0 %
          06dfe6d9  0x4744d742   rang 12 threat_seeker        12:2               100,0 %
          06dfe6d9  0x7ca85adc   rang  6 repulsor             6:22   10:1        95,7 %
          06dfe6d9  0x32d97758   rang 23 repair_field         23:20  2:1 5:1     90,9 %
          06dfe6d9  0x4396db42   rang 10 (anonyme)            10:19  3 autres    86,4 %
          06dfe6d9  0x273fe0eb   rang  4 grapple              4:20   5 autres    80,0 %
          06dfe6d9  0x72b63d69   rang  1 sensor               1:10   4 autres    71,4 %
          06dfe6d9  0x430dda48   rang  5 thruster             5:10   6 autres    62,5 %
          06dfe6d9  0x2974c233   rang  2 wall                 2:13   8 autres    61,9 %
          06dfe6d9  0x686b40c9   rang  2 wall (parent)        2:5    4 autres    55,6 %

      Les TEMOINS (un autre bipede vivant au meme instant) sont plats ou dominés par `-1` sur
      les 14 lignes : `0x8c77ffe7` passe de `20:20` a `19:11 20:2`, `0x72199cba` de `22:12` a
      `-1:9 20:7`. **Statut par identifiant** : 14 CORROBORES, 6 non mesurables (les quatre
      grenades — distribution indistinguable de leur temoin, ce qui est le signe attendu d'un
      objet dont le type ne depend pas de la capacite du lanceur — plus `0x730dc70f` a une seule
      pose et `0xb781197a` / `0xe7be9f5c` absents de ces deux films), **0 CONTREDIT**.

      **CE QUE LE CONTROLE APPREND EN PLUS.** Les quatre identifiants qui echouaient le seuil de
      85 % du lot precedent avaient RAISON : leur nature n'etait pas en cause, c'etait le
      denominateur `i48`. Et `0x4396db42` — le seul `other` restant — retombe a 86,4 % sur le
      rang 10, ce qui CONFIRME independamment que son `sofa` anonyme est bien le rang 10 de la
      famille A. La diagonale ne le nomme pas (le rang 10 n'a pas de nom), mais elle verrouille
      son rattachement.

- [x] 1.2 **Distribution des FAMILLES par film**, calculee sur les artefacts sur disque
      (schema 9, identifiant publie par pose — donc sans re-decodage) :

          film      famille               poses  avec poseur  avec cap
          000d5950  grenade_spike            59           57        49
          000d5950  grenade_plasma           49           49        43
          000d5950  grenade_frag             48           46        45
          000d5950  thruster                 33           31        25
          000d5950  grapple                  30           30        28
          000d5950  grenade_dynamo           29           29        24
          000d5950  wall                     28           28        26
          000d5950  sensor                   19           19        14
          00ba2e1c  grenade_spike           113          105        95
          00ba2e1c  grenade_dynamo           86           74        71
          00ba2e1c  grenade_frag             77           75        66
          00ba2e1c  grenade_plasma           41           41        36
          00ba2e1c  repulsor                 39           36        25
          00ba2e1c  grapple                  37           35        29
          00ba2e1c  wall                     36           34        29
          00ba2e1c  repair_field             36           33        28
          00ba2e1c  sensor                   32           32        24
          00ba2e1c  other                    21           21        17
          00ba2e1c  thruster                 19           19        17
          06dfe6d9  grenade_frag            463          397       359
          06dfe6d9  wall                     62           59        45
          06dfe6d9  repulsor                 59           52        45
          06dfe6d9  other                    53           44        33
          06dfe6d9  grapple                  48           42        37
          06dfe6d9  repair_field             41           39        33
          06dfe6d9  thruster                 37           34        28
          06dfe6d9  sensor                   35           31        23
          06dfe6d9  grenade_plasma           33           31        31
          06dfe6d9  grenade_dynamo           33           29        25
          06dfe6d9  grenade_spike            23           21        17
          06dfe6d9  threat_seeker             4            4         3
          06dfe6d9  translocator_beacon       1            1         0

      **LE CHIFFRE QUE LE LOT UI DOIT VOIR AVANT DE DESSINER** : les grenades font **63 %** des
      poses de `000d5950`, **59 %** de `00ba2e1c` et **62 %** de `06dfe6d9`. Dessiner « les
      equipements poses » sans les separer noierait le mur et le capteur sous les grenades.
      Aucun identifiant hors table (parite verifiee par la requete comme par le garde-rail), et
      `other` tombe de 100 % des poses BTB a 21/537 et 53/892.

- [x] 1.3 **CONTROLE PAR LA DUREE DE VIE** (ajout au lot en cours d'execution, source
      OFFICIELLE fournie par l'utilisateur : Halo Waypoint, « Sandbox Overview Season 4 »).
      Le jeu chiffre le detecteur de menaces : rayon 3,1 -> 4,25 wu, frequence de ping
      2,8 -> 1,8 s, **duree du capteur 6,5 -> 15 s**, duree de revelation 2,5 -> 0,75 s. Le
      traqueur, lui, est un projectile a UNE impulsion qui rebondit.

      Mesure de `t1 - t0` sur les trois artefacts (pas d'image 100 ms), par identifiant :

          famille              id          n    mediane   p10    p90    max
          wall                 0x2974c233   67     0,9 s   0,60   2,08   20,7
          wall                 0x8e2dc574   13     0,7 s   0,60   1,08    2,5
          wall (panneaux)      0x528fce46   15     0,5 s   0,30   0,76   18,9
          wall (panneaux)      0x686b40c9   31     0,5 s   0,30   0,90   13,5
          sensor               0x72b63d69   67     2,3 s   1,56   4,36   18,4
          sensor               0x72199cba   19     2,1 s   1,54   3,00   12,0
          threat_seeker        0x4744d742    4     2,3 s   1,88   4,61    5,6
          grapple              0x273fe0eb   85     1,0 s   0,64   1,88   31,8
          grapple              0x8c77ffe7   30     0,85 s  0,69   1,61    3,1
          thruster             0x430dda48   56     1,2 s   0,85   2,40    3,7
          thruster             0xeef5d48d   33     1,1 s   0,72   1,86    2,5
          repulsor             0x7ca85adc   98     1,2 s   0,70   2,23   21,0
          repair_field         0x32d97758   77     1,5 s   1,06   2,14    4,1
          other (rang 10)      0x4396db42   74     2,2 s   1,53   4,29    8,6
          translocator_beacon  0x730dc70f    1     0,7 s      —      —     0,7
          grenade_spike        0x0f5716ff  195     1,2 s   0,60   2,00    4,4
          grenade_dynamo       0xaada07f3  148     1,9 s   1,00   3,30    5,7
          grenade_plasma       0xcaaadcb0  123     3,5 s   2,32   6,74   20,7
          grenade_frag         0xbcabbe43  588     4,1 s   2,40   8,86   39,4

      **LE CONTROLE NE PEUT PAS TOURNER, ET LA RAISON EST DANS LA MESURE, PAS DANS
      L'IDENTIFICATION.** `t1` vaut `tr.Pts[len-1].TimestampUS`
      (`filmdec/equipment_creation_width.go:70`) : le DERNIER POINT DE POSITION de la vie
      decodee des paquets delta. Or un encodage delta ne transmet que ce qui CHANGE — un objet
      pose qui s'immobilise cesse d'etre transmis. `t1 - t0` mesure donc la duree du MOUVEMENT
      REPLIQUE, pas la duree de vie de l'objet. Le commentaire du champ (« T1US est le dernier
      point de la vie : la disparition ») est FAUX sur ce point.

      **LA PREUVE EST DANS LES GRENADES, et c'est une corroboration de plus de leur
      identification** : les quatre types s'ordonnent EXACTEMENT selon leur physique de rebond,
      ce qu'aucune autre lecture de ce champ n'expliquerait —

          spike   1,2 s   elle COLLE a l'impact : le mouvement s'arrete tout de suite
          dynamo  1,9 s   elle rebondit peu puis s'ancre
          plasma  3,5 s   elle rebondit avant d'adherer
          frag    4,1 s   elle rebondit ET roule

      Meme lecture ailleurs : l'appareil du mur (0,7-0,9 s = son vol) contre ses PANNEAUX
      (0,5 s : ils se deploient sur place et ne bougent plus), et le grappin (0,85-1,0 s = aller
      et retour du crochet).

      **CONSEQUENCE POUR LE CAPTEUR : ni confirme, ni contredit.** 2,1-2,3 s de mouvement ne
      s'opposent pas a 15 s de duree officielle, parce que ce ne sont pas la meme grandeur. La
      contradiction serait une durée de mouvement SUPERIEURE a 15 s ; le p90 vaut 3,0 et 4,36 s.

      **CONSEQUENCE POUR LE RENDU, et elle appartient au lot UI** : dessiner une pose sur le
      seul intervalle `[t0, t1]` affiche un detecteur pendant ~2 s la ou le jeu le laisse 15 s,
      et un mur pendant ~0,8 s. La duree de vie REELLE n'est pas dans ce champ ; la publier
      demanderait de trouver la fin de l'entite (record de suppression), qui n'est pas dans ce
      lot.

      **LE TRAQUEUR EST BIEN DISTINCT DU DETECTEUR** dans la chaine : `sofa` different
      (0x3b0ba8b6 contre 0x7f6671b2), `string_id` different (`threat_seeker` contre
      `ability_location_sensor`), `hlmt` different (a0675b2e contre c62d74c3). Deux familles
      distinctes au manifeste. Ses 4 poses ont 2,3 s de mouvement median, compatible avec un
      projectile qui rebondit — mais n = 4, on ne conclut pas.

      **CONVENTION D'UNITE, ecrite et NON tranchee.** Les positions du rejeu sont en METRES
      (bornes AABB du BSP de la carte). Le « wu » de Waypoint est l'unite monde de Slipspace :
      4,25 wu de rayon valent ~8,5 m de diametre si 1 wu = 1 m, ~26 m si 1 wu = 3,048 m (Blam
      legacy). **Le film ne porte AUCUNE portee de detection** — ni dans le record de creation,
      ni dans la vie de l'objet — donc ce lot ne tranche pas : convention SUPPOSEE 1 wu = 1 m,
      a confirmer visuellement par l'utilisateur. A noter : le lot UI declare deja un rayon de
      capteur de 8 m, choisi comme ordre de grandeur d'ecran — il coincide avec la lecture
      1 wu = 1 m des 4,25 wu, sans qu'aucun des deux ait informe l'autre.

- [x] 1.4 **TROIS CONTROLES DE PLUS, demandes en cours d'execution — et les trois font tomber
      une explication.** Contexte fourni : `000d5950` est un match SUPER FIESTA, mode ou les
      equipements sont des variantes AMELIOREES (duree, portee, charges) et ou une nouvelle pose
      REMPLACE la precedente ; et un equipement pose SURVIT A LA MORT DE SON POSEUR.

      **(a) LE REMPLACEMENT N'EXPLIQUE RIEN — il n'y a pas de successeur.** Pour chaque poseur,
      les poses de capteur ordonnees par instant, et l'ecart entre la fin de l'une et le debut de
      la suivante :

          film      poses de capteur (avec poseur)  avec un SUCCESSEUR du meme poseur
          000d5950                              19                                  0
          00ba2e1c                              32                                  1
          06dfe6d9                              31                                  1

      **Zero sur 19 sur le film Fiesta.** Un poseur pose UN capteur, jamais deux : le mecanisme
      de remplacement ne peut pas etre la cause des vies courtes, puisqu'il n'y a rien qui
      remplace. (Rappel : `owner` est un SLOT, donc une VIE — « le meme poseur » se lit par vie,
      et une vie ne pose qu'un capteur.) Le predicat est REFUTE, et il l'est proprement.

      **(b) LA DUREE MESUREE NE DEPEND PAS DU MODE**, ce qui est le troisieme argument pour
      « `t1` = fin du mouvement » :

          film      identifiant  n   mediane  p90    max     mode documente au depot
          000d5950  0x72199cba  19    2,1 s   3,00   12,0    Fiesta (Super Fiesta)
          00ba2e1c  0x72b63d69  32    2,2 s   8,19   18,4    BTB:Fiesta Slayer
          06dfe6d9  0x72b63d69  35    2,3 s   2,96   11,1    Big Team Battle, mode non documente

      2,1 / 2,2 / 2,3 s : les trois films rendent la MEME valeur a 0,2 s pres, alors que leurs
      modes different. Une duree d'equipement amelioree se verrait ; un temps de VOL, non.
      **LIMITE ASSUMEE** : le predicat (b) demandait un film NON Fiesta, et je n'ai pas pu
      l'etablir — le registre des matchs est tenu en ECRITURE par le serveur local, et l'ouvrir
      en lecture depuis un second processus viole les ADR 0013/0016. Les modes ci-dessus sont
      ceux que le DEPOT documente (journal du 16/08 pour `000d5950`, registre des reports
      ligne « index de capacite 7 » pour `00ba2e1c`) ; celui de `06dfe6d9` reste inconnu. La
      valeur officielle de 15 s n'est donc toujours pas testee — et elle ne le sera pas par ce
      champ, cf. 1.3.

      **(c) `t1` N'EST PAS BORNE PAR LA MORT DU POSEUR — verification sur pieces, et c'est un
      NEGATIF (pas de bug).** L'objet a bien sa propre vie :

          film      poses avec poseur  t1 APRES la fin de la vie du poseur  t1 a <= 2 frames
          000d5950                289                       253  (87,5 %)         2  (0,7 %)
          00ba2e1c                505                       442  (87,5 %)         4  (0,8 %)
          06dfe6d9                784                       678  (86,5 %)         5  (0,6 %)

      Dans 86 a 88 % des cas l'objet cesse de bouger APRES la fin de la vie de son poseur
      (mediane +1,4 a +2,55 s), et la coincidence exacte est au niveau du hasard. Aucun clamp
      dans le code non plus : `T1US` vient de la trajectoire de l'objet
      (`equipment_creation_width.go:70`), et la couche qui attache le poseur
      (`replay/equipment_placements.go`) ne touche ni `T0` ni `T1`. Cote web, le calque lit
      `[t0, t1]` tels quels — c'est le comportement attendu.

      **(d) LES VARIANTES ONT BIEN LEUR PROPRE `eqip`, mais leur nom ne derive PAS du nom de
      base.** La structure le montre deja : le rang 19 a son `sofa` (0xfb80ca6f), ses `eqip`
      (37c87a13 + 8e2dc574) et le MEME `hlmt` que le mur du rang 2 — c'est exactement le patron
      « variante avec son propre tag ». Mais son identifiant de chaine ne s'ecrit pas
      « base + marque de variante » : 2 970 candidats (18 bases connues x 55 affixes, en
      suffixe, en prefixe et en remplacement de prefixe) contre les 6 identifiants non casses,
      esperance de collision 4e-6, **zero resultat** (`TestDicoVariantesSuffixes`, versionne,
      tourne partout). **ET L'HYPOTHESE « rangs 19-22 = variantes Fiesta » A UN CONTRE-EXEMPLE
      DOCUMENTE** : `00ba2e1c` est un BTB:Fiesta Slayer d'apres le registre du depot, et il
      montre les rangs BAS de la famille A, pas les rangs 19-22. Le lien palette <-> mode n'est
      donc pas etabli, et il ne le sera pas sans le registre.

      **PORTEE DU CAPTEUR PAR MODE** : la valeur officielle de 4,25 wu est celle du mode
      STANDARD (saison 4 et apres). Un mode qui ameliore l'equipement peut servir une portee
      differente. Le film n'en porte aucune : rien de tout cela n'est mesurable ici, et le rayon
      d'ecran reste une valeur declaree.

**Gate 1 : PASSE** — aucune contradiction, les `other` restants sont listes (un seul
identifiant, `0x4396db42`), et le controle par la duree de vie est CONCLUANT SUR CE QU'IL
MESURE (l'ordre des grenades, l'independance au mode, l'absence de remplacement, l'absence de
clamp sur la mort du poseur) et EXPLICITEMENT MUET sur la duree officielle du capteur.

### Phase 2 — LE TRANSLOCATEUR — CLOSE le 2026-08-18 (balise OUI, retour NEGATIF MESURE)

- [x] 2.1 **LA BALISE EXISTE, ELLE EST NOMMEE, ET LE CORPUS PASSE DE 1 A 4 POSES.**
      L'identifiant `0x730dc70f` est l'`eqip` du `sofa` `0x8f1be870`, dont l'identifiant de
      chaine `0x1f7c6a15` donne `quantum_translocator` — le rang 11 de la famille A, celui que
      la RECETTE §13 nommait deja translocateur par un autre chemin. Il se publie donc dans
      `equipmentPlacements` avec la famille `translocator_beacon`.

      La recherche de porteurs n'a coute AUCUN decodage : les artefacts sur disque publient les
      lectures `i48`, et une requete sur les 29 artefacts rend les seuls films ou un rang 11 est
      lu — `83ee3f9f` (5 lectures), `64e8adfa` (4), `06dfe6d9` (3), `82f29378` (1). Les quatre
      ont ete mesures :

          film      poses totales  BALISES  note
          83ee3f9f             64        3  HORS du corpus calibre de 11 films
          06dfe6d9            892        1  seul porteur connu avant ce lot
          64e8adfa            229        0  4 lectures i48 rang 11, aucune balise posee
          82f29378            176        0  1 lecture

      **UNE LECTURE `i48` RANG 11 NE VAUT PAS UNE POSE** : porter le translocateur n'est pas
      l'employer. `64e8adfa` en est la demonstration — 4 porteurs, zero balise.

- [x] 2.2 **LE RETOUR N'EST PAS UNE DISCONTINUITE DE POSITION — NEGATIF MESURE, seuils
      inchanges.** L'instrument `filmdec/translocateur_test.go` (garde `TRANSLOC_FILM`) cherche
      les sauts > 4 m en <= 150 ms dans une vie de bipede, filtre de vitesse DESACTIVE :

          film      fenetre        positions  vies  transitions  SAUTS > 4 m  balises
          06dfe6d9  chunks 19-21      31 429    41       31 290            0        1
          83ee3f9f  chunks 1-5        24 908    19       24 860            0        3
          64e8adfa  film complet     293 837   138      293 126            4        0
          82f29378  film complet      65 505    63       65 261            0        0

      Les fenetres de `06dfe6d9` et `83ee3f9f` CONTIENNENT les quatre balises du corpus
      (la balise de `06dfe6d9` est posee a 329,3 s, les chunks 19-21 couvrent ~317-367 s).
      **Zero saut, donc zero retour a confronter : le taux n'est pas bas, il est vide.** Les
      quatre seuls sauts du corpus sont sur un film SANS balise, tous sur un meme slot (607),
      de 4,7 a 12,1 m en 16-17 ms, et aucun n'arrive a moins de 2 m d'une pose — le profil d'une
      aberration de balayage ou d'un vehicule, pas d'une teleportation.

      **DEUX EXPLICATIONS RESTENT OUVERTES, et la mesure ne les separe pas.** (a) Les porteurs de
      ces films n'ont pas utilise le retour — poser la balise et ne pas y revenir est un usage
      courant. (b) Le retour n'est PAS transmis comme un deplacement : la replication detruirait
      l'entite et en creerait une autre, donc une NOUVELLE vie de bipede — et un saut « dans une
      vie » ne peut par construction pas le voir. L'hypothese (b) est celle qu'il faudrait tester
      en premier, et le plan l'excluait explicitement (« vie continue, pas un spawn ») : le
      critere ecrit avant la mesure interdisait de la voir. C'est une limite du plan, pas un
      echec de l'instrument.

- [x] 2.3 **RIEN N'EST PUBLIE DE PLUS QUE LA BALISE**, et c'est la regle du chantier : pas de
      `translocations` au schema 10, pas de bordure animee bleu -> feu (spec Notion 21.1). La
      balise seule voyage, par sa famille `translocator_beacon` dans `equipmentPlacements`
      (schema 9, inchange). Le negatif du retour entre au registre avec sa condition de reprise.

**Gate 2 : PASSE en NEGATIF.** Balise : 4 poses, 2 films, nommee par la structure. Retour : 0
saut sur 4 films et 415 000 transitions contemporaines — la bordure animee reste impossible, et
elle le reste pour une raison mesuree.

### Phase 3 — Journal, registre, planche — CLOSE le 2026-08-18

- [x] 3.1 Registre : cinq lignes mises a jour (« mur et capteur famille A non nommes » SORTIE,
      « rangs 19 et 22 famille B » SORTIE, « GlobalID `eqip` pas lisibles dans les `sofd` »
      SORTIE avec sa cause, « rang 10 » enrichie de son negatif chiffre, « 4 identifiants plats
      sans poseur » CORRIGEE) et cinq lignes nouvelles (retour du translocateur, filtre
      `MaxSpeedMPS`, `t1` n'est pas la duree de vie, table `PLACEMENT_RENDER` a etendre,
      convention `wu`). Journal : `.ai/thought_log.md` + `.ai/V7.5/replay2d/thought_log_replay.md`.
      La planche (item F1) sera corrigee a sa prochaine mise a jour — elle n'est pas touchee par
      ce lot.

## Regles dures

Un nom vient de la structure ou n'existe pas ; la diagonale corrobore ; seuils fixes avant
mesure ; zero fix hors perimetre ; commits sur `feat/v75`, pas de push.

## LA DECOUVERTE LA PLUS LOURDE DU LOT — `equipmentPlacements` n'est pas ce que son nom dit

Elle n'etait pas cherchee : elle est sortie du GOLDEN re-genere, quand toutes les poses ont pris
un nom d'un coup. Le golden de `000d5950` montre, repetee, une configuration qui n'a rien d'une
pose sur la carte :

    grenade_spike 0x0f5716ff t=[317, 331] pos=(25.52, 10.40, -1.08) poseur=515 cap=195.7 deg
    grenade_spike 0x0f5716ff t=[317, 322] pos=(25.54, 10.41, -1.08) poseur=515 cap=195.7 deg
    sensor        0x72199cba t=[317, 333] pos=(24.31, 10.35, -1.76) poseur=515 cap=195.7 deg

Deux grenades IDENTIQUES a 2 cm l'une de l'autre, au MEME instant, plus une capacite, pour le
MEME poseur. Personne ne lance deux grenades dans la meme image : ce sont les DEUX GRENADES DE
LA DOTATION et la CAPACITE du joueur, creees ensemble quand il les recoit. Le motif se repete
sur les poseurs 515, 518, 514, 513, 519, 512.

**LA MESURE SEPARE TOTALEMENT DEUX CLASSES D'OBJET.** Part des poses qui partagent leur poseur
ET leur image avec une grenade, sur les trois artefacts :

    0x686b40c9  panneaux du mur (famille A)     31 poses     0 / 31 =   0,0 %
    0x528fce46  panneaux du mur (rang 19)       15 poses     0 / 15 =   0,0 %
    ------------------------------------------------------------------------
    0x72b63d69  capteur                         67 poses    35 / 67 =  52,2 %
    0x2974c233  appareil du mur                 67 poses    37 / 67 =  55,2 %
    0x4396db42  rang 10                         74 poses    45 / 74 =  60,8 %
    0x7ca85adc  repulseur                       98 poses    64 / 98 =  65,3 %
    0x430dda48  propulseur                      56 poses    37 / 56 =  66,1 %
    0x32d97758  champ de reparation             77 poses    52 / 77 =  67,5 %
    0xeef5d48d  propulseur (rang 21)            33 poses    24 / 33 =  72,7 %
    0x8e2dc574  appareil du mur (rang 19)       13 poses    10 / 13 =  76,9 %
    0x273fe0eb  grappin                         85 poses    67 / 85 =  78,8 %
    0x72199cba  capteur (rang 22)               19 poses    15 / 19 =  78,9 %
    0x8c77ffe7  grappin (rang 20)               30 poses    24 / 30 =  80,0 %
    les 4 grenades                             1054 poses            88-98 %
    0x730dc70f  balise du translocateur          1 pose      1 /  1 = 100,0 %

**ZERO sur 46 d'un cote, jamais moins de 52 % de l'autre.** Les DEUX seuls identifiants que la
chaine rattachait par `sofa_parent` — les panneaux deployes du mur — sont exactement les deux
qui ne naissent JAMAIS avec une dotation. Deux lectures independantes (la structure des tags du
jeu, et la co-occurrence dans le film) designent le meme couple.

**CE QUE CELA CHANGE POUR LE RENDU, et ce n'est pas mince** : la majorite des entrees de
`equipmentPlacements` ne sont pas des objets POSES sur la carte, ce sont les objets d'equipement
CREES A LA POSITION DU JOUEUR quand il recoit sa dotation. Dessiner un arc de mur a ces
positions dessine un mur ou personne n'en a deploye.

**CE QUI N'EST PAS ETABLI, et il faut le dire** : que les poses ISOLEES (celles qui ne
co-occurrent pas avec une grenade) soient les deploiements reels. L'ordre de grandeur y invite —
appareil du mur isole contre panneaux : 10 contre 13 sur `00ba2e1c`, 20 contre 18 sur
`06dfe6d9`, 3 contre 15 sur `000d5950` — mais un facteur qui vaut 1,3, 0,9 puis 5 n'est pas une
regle. Le critere de dotation employe ici (meme poseur, MEME image) est en outre le plus strict
possible : il sous-estime la dotation des qu'une grenade manque au balayage, ce qui gonfle les
« isolees ». Aucune conclusion n'est tiree, aucun filtre n'est pose : le lot NOMME, il ne
retranche pas.

## Decouvertes non traitees (portees au registre, PAS corrigees ici)

1. **Le decodeur de production JETTE les teleportations.** `DefaultScanFilmOptions` porte
   `MaxSpeedMPS = 100` : toute position dont la vitesse depuis la precedente depasse 100 m/s
   est rejetee comme faux positif du balayage bit a bit. Un retour de translocateur fait 20 a
   40 m en une image, soit 200 a 400 m/s — il est donc INVISIBLE dans les trajectoires
   publiees, et le restera tant que ce filtre ne distingue pas une aberration d'une
   teleportation legitime. Consequence a peser hors de ce lot : le grappin long et les
   ascenseurs gravitationnels peuvent etre coupes par le meme filtre.
2. **Les rangs 13 a 18 et 24 a 26 de la famille A ne sont pas casses** (ni a deux ni a trois
   jetons pour ceux qui portent un objet observe). Leurs `eqip` ne sont pas au corpus, donc ils
   ne bloquent rien aujourd'hui.
3. **Neuf palettes `sofd` de 10 rangs ou plus** vivent dans `globals`, et le `glpa` en
   reference douze. Trois d'entre elles partagent leurs rangs 0 a 9 (la « famille A ») ;
   quatre autres partagent un prefixe different et portent des `sofa` de categorie 2 — une
   valeur de categorie que ce lot n'a pas cherche a nommer.
4. **Le `sofa` porte deux autres identifiants de chaine** a `+0x14` et `+0x18` de son bloc
   racine, partages entre rangs (0x67b8b3df sur tous les deployables de la famille A,
   0xd2fc28a6 puis 0xb37dcbe4). Ils sentent la categorie et la sous-categorie ; non casses,
   non necessaires au lot.
5. **LE MANIFESTE ELARGI EXIGE UNE ACTION DU LOT UI, ET ELLE N'EST PAS FAITE ICI.** Le rendu
   entre par `PLACEMENT_RENDER` (`apps/web/src/features/match-replay/equipmentPlacementsLayer.ts`),
   une table de trois cles — `wall`, `sensor`, `other` — et `placementKind` rend `null` pour
   toute famille hors table : « une famille hors table = aucun dessin, jamais celui d'une
   voisine ». Consequence MESUREE du changement de manifeste : les poses qui tombaient dans
   `other` (donc dessinees en point neutre quand la bascule est allumee) portent desormais
   `grapple`, `thruster`, `repulsor`, `repair_field`, `threat_seeker`, `powerup_*` ou
   `grenade_*` et **ne se dessinent plus du tout**. Sur `06dfe6d9`, cela retire 839 des
   892 poses de l'affichage — dont les 552 grenades, ce qui est vraisemblablement souhaitable,
   et 287 poses d'equipement qui ne le sont pas. Le mur passe au contraire de 1 a 4
   identifiants (13 -> 62 poses sur ce film) et le capteur de 1 a 2. Ce lot ne touche pas a
   `apps/web/` (lot parallele) : la decision et le code appartiennent au lot UI.
6. **Le nuage de bipedes NON FILTRE fait tomber le processus sur les gros films.** Avec
   `MaxSpeedMPS = 0` (obligatoire pour voir une teleportation, cf. decouverte n°1),
   `ScanFilmBipedPositions` sur les 45 chunks de `06dfe6d9` et les 14 de `83ee3f9f` meurt sans
   sortie ni panique — le processus est tue. Contournement employe : borner le balayage aux
   chunks utiles (`TRANSLOC_CHUNKS`). Une mesure plein-film du saut demanderait un balayage en
   flot (traiter chunk par chunk sans conserver le nuage), qui n'est pas dans ce lot.
