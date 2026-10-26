# Plan — Poses d'equipement : separer la DOTATION du DEPLOIEMENT, puis dessiner les familles nommees

> Ecrit le 2026-08-18, suite du lot de nommage (`77d77d2cf`) : 20 identifiants sur 21 nommes par
> la structure du jeu (`sofd -> sofa -> {string_id, eqip}`), 15 familles au manifeste. Deux
> consequences que ce lot traite : (1) le rendu web ne connait que 3 familles sur 15 (839 des
> 892 poses de `06dfe6d9` ne seraient plus dessinees) ; (2) **`equipmentPlacements` mele la
> DOTATION AU SPAWN et les objets DEPLOYES** — deux grenades identiques a 2 cm au meme instant
> + une capacite, meme poseur ; separation totale : 0/46 de co-occurrence pour les panneaux de
> mur (`sofa_parent`), 52-100 % pour tous les autres. Branche `feat/v75`, contrat
> `plan-execution`. Deux voies : Go (principal) puis web (worktree frere).

## Ce qui est acquis (lot nommage, ne pas re-mesurer)

- Manifeste `[[equipment_objects]]` : 21 identifiants, familles `wall` (4 dont 2 panneaux
  `sofa_parent`), `sensor` (2), `grapple` (2), `thruster` (2), `translocator_beacon`,
  `threat_seeker`, `repulsor`, `repair_field`, `powerup_overshield`, `powerup_camo`,
  `grenade_{frag,plasma,dynamo,spike}`, `other` (1, rang 10 anonyme). Diagonale de controle
  9/9, 0 contradiction.
- Les grenades font 59-63 % des poses ; les 4 « plats » etaient les 4 grenades ; 96-100 % ont
  un poseur a < 3 m.
- `t1` = fin du MOUVEMENT replique, pas la duree de vie (grenades : spike 1,2 s < dynamo 1,9 <
  plasma 3,5 < frag 4,1, l'ordre de leur physique de rebond) ; le remplacement Fiesta, le mode
  et la mort du poseur sont REFUTES comme causes de la vie courte du capteur ; un objet pose
  survit au poseur (fait de jeu, verifie : cesse de bouger APRES la fin de vie du poseur dans
  86-88 % des cas, coincidence 0,6-0,8 %, aucun clamp).
- Translocateur : balise `0x730dc70f` nommee, 4 poses sur 2 films ; RETOUR negatif mesure
  (0 saut > 4 m en <= 150 ms sur 56 150 transitions) — la piste « vie continue » interdisait
  par construction de voir un retour qui reinitialise la vie ; a reprendre autrement.
- Le web : `PLACEMENT_RENDER` = table par famille (`wall`, `sensor`, `other`), une famille hors
  table ne dessine rien (c'est voulu, et c'est ce qui rend ce lot sur).

## Decisions tranchees avant execution

1. **L'origine d'une pose est MESUREE, pas presumee** : `origin = spawn` si l'objet est cree
   dans la meme fenetre que le DEBUT DE VIE du poseur (|t0_objet - t0_vie| <= 2 frames) ET a
   moins de 1,5 m de lui ; `deployed` sinon (cree en cours de vie). Les seuils sont ecrits
   avant la mesure ; la distribution des ecarts (t0_objet - t0_vie) par famille est PUBLIEE
   avant de trancher — s'il n'y a pas deux modes nets (un pic a 0 et une queue), le critere
   est faux et on l'ecrit.
2. **Publication** : champ `origin` sur chaque pose (`spawn` | `deployed` | `unknown` sans
   poseur), `SchemaVersion` 9 -> 10, couverture par famille x origine.
3. **Le rendu web ne dessine que les DEPLOYES** (un objet de dotation n'est pas « pose » — il
   accompagne le joueur) : mur = arc (les PANNEAUX `sofa_parent` sont l'objet reel du mur
   deploye — arbitrer sur pieces si l'appareil `0x8e2dc574` est une dotation, auquel cas
   l'arc se dessine sur les panneaux), capteur = ping (existant), `threat_seeker` = un seul
   ping (une impulsion — source Waypoint) puis rien, `translocator_beacon` = balise
   (marqueur discret, dure [t0, t1]), `repair_field` = zone (disque a l'encre de l'equipe,
   rayon DECLARE), `powerup_*` = **objets de la CARTE** (marqueur d'icone du power-up a la
   position, pendant sa presence — c'est l'item « power-ups poses par la carte » du cahier
   des charges, sans poseur), grenades / grappin / propulseur / repulseur = RIEN en tant que
   poses (les lancers de grenade ont deja leur calque ; le reste est de la dotation), `other`
   = point neutre sur bascule (existant).
4. Icones : `static/abilities-assets/halo_infinite/*` (dark/light) et `grenades-assets`
   existent ; un power-up sans icone versionnee -> marqueur neutre + infobulle, jamais une
   icone voisine.
5. Regles inchangees : tokens uniquement, FR/EN, un seul decodage filmdec par process, aucune
   base en ecriture, JAMAIS `git add -A`, jamais d'attente passive.

## Phases

### Phase G (Go, principal) — ORIGINE — CLOSE le 2026-08-18

> **LA DECISION N°1 EST REFUTEE, ET LA MESURE DONNE L'AUTRE BOUT DE LA VIE.** Le critere
> ecrit avant mesure cherchait une DOTATION AU SPAWN : creation dans les 2 frames du DEBUT de
> la vie du poseur. Elle compte **4 poses sur 3 661 (0,1 %)**. La meme mesure, prise a la FIN
> de la vie du poseur, en compte **3 242 sur 3 661 (88,6 %)**. Les poses de
> `equipmentPlacements` ne sont pas des objets recus au spawn : ce sont, en majorite, les
> objets que le joueur PORTAIT, relaches quand il MEURT. Le plan avait prevu ce cas
> (« si elle n'a pas deux modes nets, dis que le critere est faux plutot que de le forcer ») ;
> il y a bien deux modes, sur l'autre axe. Vocabulaire publie en consequence :
> `deployed` / `dropped` / `unknown` — **`spawn` n'est pas publie, il est vide.**

- [x] G.1 **DISTRIBUTION SUR LES 11 FILMS CALIBRES — 4 250 poses, 3 661 a poseur mesure,
      589 sans poseur, 21 identifiants distincts.** Instrument versionne
      `replay/origine_poses_research_test.go` (garde `ORIGINE_FILM` + `ORIGINE_MAP`), un film
      par PROCESSUS, 11 executions a exit 0. Le poseur n'est pas recalcule : l'instrument
      appelle `equipmentOwner`, LA fonction de production.

      Les noms de carte des 11 films — necessaires aux METRES — viennent du **snapshot parquet
      du registre** (`match_registry_20260711_090652.parquet`, `read_parquet` en memoire) : la
      voie que le cadre du lot prescrivait, et elle evite la seconde ouverture de la DB tenue
      RW (ADR 0013/0016).

          axe mesure                        <= 2 frames    <= 1 s   <= 5 s   <= 20 s   > 20 s
          |t0_objet - t0_vie|  (DEBUT)         4   0,1 %      1,0 %    2,2 %    32,3 %   64,4 %
          |t0_objet - t1_vie|  (FIN)        3242  88,6 %      0,7 %    1,4 %     3,4 %    5,9 %

      **DEUX MODES : OUI, ET SUR L'AXE DE LA FIN.** Un pic a 0 qui prend 88,6 % de la
      population, puis une queue etalee de 1 s a plus de 20 s. Sur l'axe du DEBUT il n'y a pas
      de pic du tout : 96,7 % de la population est au-dela de 5 s, la distribution est celle
      d'un instant quelconque dans la vie.

      **LES 4 POSES DU MODE « SPAWN » NE SONT PAS DES DOTATIONS** — verification sur pieces,
      et elle acheve l'hypothese : ce sont deux paires de grenades de fragmentation dont la
      vie du poseur dure **0,13 s et 1,49 s**. A cette duree, debut et fin de vie se
      confondent ; les quatre sont aussi a <= 2 frames de la FIN. Le mode « spawn » est vide.

      **TEMOIN INTERNE, sur les MEMES poses et la MEME methode** — la distance de la pose aux
      deux extremites de la vie de son poseur :

          distance a la position de DEBUT de vie   mediane 27,03 m
          distance a la position de FIN de vie     mediane  0,57 m     facteur 47,5

      L'objet nait la ou le poseur s'arrete, pas la ou il est apparu.

- [x] G.2 **ORIGINE CLASSEE ET TABLE FAMILLE x ORIGINE.** Regle publiee (`equipmentOrigin`) :
      `dropped` si |t0_objet - t1_vie| <= 2 frames ET distance < 1,5 m ; `deployed` sinon ;
      `unknown` sans poseur. **Les DEUX seuils sont ceux du plan, ecrits avant la mesure** —
      seul l'AXE a change, et il a change parce que la mesure l'a dit.

      Les seuils ne se reglent pas, ils se constatent : les lachers sont a **20-40 ms** et
      **0,63 m** de la fin de vie, les deploiements a **14-42 s** et **5,6-21,3 m**. Trois
      ordres de grandeur separent les deux populations — n'importe quel seuil entre 1 s et 10 s
      rendrait le meme classement.

          famille               poses   dropped  deployed  unknown   % deployed
          wall                    222        91       120       11        54,1
          threat_seeker             4         3         1        0        25,0
          repair_field            105        69        26       10        24,8
          sensor                  155       106        36       13        23,2
          other                   108        71        25       12        23,1
          repulsor                163       108        33       22        20,2
          thruster                163       124        23       16        14,1
          grapple                 306       241        26       39         8,5
          grenade_spike           412       342        21       49         5,1
          grenade_frag           1991      1519       100      372         5,0
          grenade_plasma          388       349        17       22         4,4
          grenade_dynamo          229       200         6       23         2,6
          translocator_beacon       2         2         0        0         0,0
          powerup_overshield        1         1         0        0         0,0
          powerup_camo              1         1         0        0         0,0

      Par film (11 films, `% deployed` de 4,5 a 15,2) :

          film      poses  dropped  deployed  unknown     film      poses  dropped  deployed  unknown
          000d5950    295      250        39        6     07aa428d    239      195        34       10
          00162144    181      165        16        0     084a804d    922      481        53      388
          00502e52    246      223        20        3     331ff98d    238      223        15        0
          00ba2e1c    537      437        68       32     64e8adfa    229      215        11        3
          06dfe6d9    892      661       123      108     9edfcaa9    316      231        48       37
          cfb85a58    155      146         7        2

      **LES TROIS PREDICTIONS, STATUEES UNE PAR UNE :**

      1. « grenades et capacites de dotation ~100 % `spawn` » — **REFUTEE, et par le
         mecanisme, pas par le taux.** Elles sont a 95 % `dropped` : elles ne sont pas RECUES
         a la position du joueur, elles y sont LACHEES. Le groupe simultane que le lot
         precedent avait vu (deux grenades identiques a 2 cm + une capacite) est reel, mais il
         se produit a la MORT, pas au spawn.
      2. « panneaux de mur, balise, capteur lance, seeker, champ de reparation ~100 %
         `deployed` » — **TENUE POUR LES PANNEAUX SEULS (97,7 et 97,9 %), REFUTEE POUR LES
         QUATRE AUTRES** (capteur 15-28 %, champ de reparation 24,8 %, seeker 25 % pour n = 4,
         balise 0 % pour n = 2). La prediction confondait la FAMILLE avec l'IDENTIFIANT : dans
         chaque famille sauf les panneaux, l'identifiant est l'APPAREIL que le joueur PORTE, et
         un appareil porte est lache bien plus souvent qu'il n'est deploye.
      3. « power-ups ~100 % `unknown` (pas de poseur), positions RECURRENTES » — **REFUTEE SUR
         LE PEU QU'ON A, ET LE PEU EST DIT.** Le corpus entier porte **une** pose de
         surbouclier et **une** de camouflage ; les deux ont un poseur et les deux sont
         `dropped` (23,0 et 38,5 ms, 0,53 et 0,67 m de la fin de vie). n = 1 ne prouve rien :
         la prediction n'est pas testable a la force voulue, et ce qui existe la contredit.
         Aucune position recurrente n'a pu etre cherchee. Les 589 poses `unknown` du corpus ne
         sont PAS des power-ups : elles sont eparpillees sur 7 identifiants, et 388 d'entre
         elles viennent d'un seul film (`084a804d`, 922 poses, 256 traces).

- [x] G.3 **L'APPAREIL DU MUR ET LE CAPTEUR SONT PORTES, PAS DEPLOYES — ET L'ARC DU MUR SE
      DESSINE DONC SUR LES PANNEAUX.** La diagonale de la nature, par identifiant :

          0x528fce46  panneaux du mur (rang 19)   47/48 = 97,9 %   0 lacher sur 48   DEPLOYED
          0x686b40c9  panneaux du mur (rang  2)   42/43 = 97,7 %   1 lacher sur 43   DEPLOYED
          ----------------------------------------------------------------------------------
          0x2974c233  APPAREIL du mur (rang  2)   25/85 = 29,4 %                     carried
          0x72b63d69  capteur (rang 1)            27/95 = 28,4 %                     carried
          0x32d97758  champ de reparation        26/105 = 24,8 %                     carried
          0x7ca85adc  repulseur                  33/163 = 20,2 %                     carried
          0x72199cba  CAPTEUR (rang 22)            9/60 = 15,0 %                     carried
          0x8e2dc574  APPAREIL du mur (rang 19)    6/46 = 13,0 %                     carried
          0x273fe0eb  grappin                    22/223 =  9,9 %                     carried
          les 4 grenades                                 2,6 a 5,1 %                 carried

      **97,7 % D'UN COTE, 29,4 % DE L'AUTRE, ET RIEN ENTRE LES DEUX.** Reponse aux deux
      questions du plan : `0x8e2dc574` est PORTE (35 lachers sur 46), `0x72199cba` est PORTE
      (50 lachers sur 60). Un appareil de mur a la position d'un joueur est, 87 fois sur 100,
      l'objet qu'il tenait en mourant.

      **CONTROLE INTERNE, ET C'EST LUI QUI VALIDE TOUTE LA MESURE** : si les 88,6 % de lachers
      etaient un artefact de la methode (fenetre trop large, poseur mal attribue), les
      panneaux les porteraient aussi. Ils en ont **zero sur 48** et **un sur 43**. La methode
      SAIT rendre 0 %.

      **TROISIEME LECTURE INDEPENDANTE DU MEME COUPLE.** La chaine des tags du jeu
      (`sofa_parent`) et la co-occurrence avec une grenade (0/46) designaient deja les deux
      panneaux ; la vie du poseur les designe une troisieme fois, par un signal qui ne partage
      aucune piece avec les deux premiers.

      Manifeste : `kind` (`carried` | `deployed`) sur les **21 lignes**, liste FERMEE, plus un
      invariant FATAL BILATERAL (`verifieNatureEquipement`) — `deployed` exige la provenance
      `sofa_parent` et reciproquement. `kind = dotation` du plan n'est pas ecrit : la mesure
      dit `carried` (« porte, donc lache »), et `map_object` n'a **aucun membre** au corpus, ce
      qui interdit de l'ouvrir (regle 7 : pas de vocabulaire mort). Test
      `TestLoadReplayLabels_NatureEquipement`, les deux sens.

      Trois identifiants ont un effectif trop faible pour que la mesure tranche seule (balise
      n = 2, les deux power-ups n = 1) : leur `kind` repose sur la structure (aucun n'est
      `sofa_parent`), et c'est ECRIT au manifeste pour que personne ne le lise comme une mesure.

- [x] G.4 **PUBLIE — SCHEMA 10.** `origin` sur chaque pose (`deployed` / `dropped` /
      `unknown`, jamais vide) ; `coverage.placements` gagne `deployed` / `dropped` / `unknown`
      et le croisement `byFamilyOrigin` (cle `"<famille>/<origine>"`). La version monte pour un
      champ de SOUS-OBJET, et la raison est ecrite aux trois endroits (`document.go`,
      `structure_test.go`, chronique) : c'est le SENS DU CALQUE qui change, et un client v9
      dessine aujourd'hui un mur la ou personne n'en a deploye.

      Invariant teste : `deployed + dropped + unknown == placements`, exactement
      (`TestPlacementCoverageEquilibreLesOrigines`) — meme regle que `LayerCoverage.Balanced`.
      Machine verrouillee sur donnees synthetiques (`equipment_origin_test.go`, 6 tests) :
      fenetre, distance, decoupage des vies, choix de la vie qui CONTIENT l'instant.

      Golden : le titre de section disait « dotation au spawn ET objets deployes » — corrige,
      avec la lecon gardee en commentaire (un groupe de creations simultanees ne dit pas OU
      dans la vie du poseur il tombe ; il fallait mesurer les deux bouts). Le golden publie
      desormais l'origine par pose ET le croisement famille x origine ; sa premiere ligne est
      `wall deployed 0x528fce46`, suivie du groupe `dropped` de deux grenades + capteur.

      Contrat regenere (`make openapi-gen` puis `make generate-types`) : `origin` requis sur
      `EquipmentPlacement`, 5 champs sur `EquipmentPlacementCoverage`. Temoins re-cuits :
      `000d5950`, `06dfe6d9`, `00ba2e1c`.

      **CORRECTIF PORTE AU PASSAGE, assigne a ce lot par le registre** (ligne « `T1US` n'est
      pas la duree de vie — et son commentaire dit le contraire », reprise « 1 ligne, prochain
      lot qui touche `equipment_placements.go` ») : le commentaire de `T0`/`T1` disait « la
      disparition ». Il dit maintenant ce que le champ est — la fin du MOUVEMENT REPLIQUE.

**Gate G : PASSE.** Distributions publiees avec leurs denominateurs, les trois predictions
statuees (1 refutee, 1 tenue pour un identifiant sur cinq, 1 refutee sur n = 2), gates Go et
lint verts, contrat regenere, temoins re-cuits.

### Phase W (web, worktree frere `wt/familles-poses`, apres G)

> **TROIS CORRECTIONS QUE LA MESURE DE LA PHASE G IMPOSE A CETTE PHASE**, a lire avant W.1 :
>
> 1. **Le filtre est `origin === 'deployed'`, et il retire 88,6 % des poses.** Il n'y a pas de
>    « dotation au spawn » a ecarter : ce qu'on ecarte, ce sont des objets LACHES A LA MORT.
> 2. **L'arc du mur va sur les PANNEAUX** (`0x528fce46` / `0x686b40c9`, `kind = deployed`,
>    97,7-97,9 % de deploiements), **jamais sur l'appareil** (`0x8e2dc574` 13,0 %,
>    `0x2974c233` 29,4 %). Un mur deploye produit DEUX poses `deployed` — l'appareil qui vole
>    ET ses panneaux : les dessiner toutes deux ferait deux arcs pour un seul mur.
> 3. **`origin === 'unknown'` pour les power-ups ne dessinerait RIEN** : le corpus entier porte
>    UNE pose de surbouclier et UNE de camouflage, et les DEUX ont un poseur et sont `dropped`.
>    L'hypothese « power-ups = objets de la carte sans poseur » n'a aucun appui mesure. Ne pas
>    coder un filtre qui n'a aucun membre (regle 7) — soit on trouve d'abord un film qui porte
>    de vrais power-ups de socle, soit la famille reste hors table.

- [x] W.1 **TABLE ETENDUE ET FILTRE POSE.** `PLACEMENT_RENDER` passe de 3 a 13 entrees et de
      `Record<string, PlacementKind>` a `Record<string, PlacementKind | null>` : cinq familles
      DEPLOYABLES (`wall`, `sensor`, `translocator_beacon` -> `beacon`, `threat_seeker` ->
      `seeker`, `repair_field` -> `field`), `other` -> `unnamed`, et SEPT familles portees a
      `null` explicite et commente (4 grenades, `grapple`, `thruster`, `repulsor`).

      **LES POWER-UPS RESTENT HORS TABLE, PAS MEME A `null`** (correction 3) : n = 1 par
      power-up sur les 11 films, les deux avec poseur et `dropped`. Un commentaire de la table
      porte la mesure ET la condition de reprise ; un test verrouille l'absence
      (`'powerup_overshield' in PLACEMENT_RENDER === false`).

      Filtre : `placementIsDeployedObject` = `origin === 'deployed'` ET — pour le mur seul —
      identifiant de PANNEAU. `placementOrigin` lit l'origine ABSENTE comme `unknown`, jamais
      `deployed` (le champ est optionnel au contrat : le parc anterieur au schema 10 est encore
      en production). L'objet non identifie ECHAPPE au filtre : sa bascule est un diagnostic
      (« voir ce qu'on ne sait pas nommer, d'ou qu'il vienne »), comportement inchange.

      **DECOUPE IMPOSEE PAR LE SEUIL DE TAILLE** (CLAUDE.md n°5) : le calque serait passe a
      593 L. Deux fichiers neufs, sur une frontiere qui se dit en une phrase —
      `placementShapes.ts` (346 L : le socle du cadrage + les formes qui tiennent dans un
      CENTRE PROJETE) et `placementWall.ts` (179 L : le mur, geometrie MONDE +
      `WALL_PANEL_IDS`). Le calque retombe a 427 L, decide, et n'aiguille plus que le capteur.
      Les tests suivent la meme decoupe (277 / 197 / 150 L) avec un decor partage
      `test/placementFixtures.ts` (121 L) — trois copies du meme cadrage auraient diverge au
      premier reglage d'echelle (CLAUDE.md n°6).

- [x] W.2 **RENDUS.** Balise = losange ferme + coeur, en pixels d'ecran, A DEMEURE sur [t0, t1]
      et SANS pulsation (rien de mesure ne bat) ; le losange est symetrique par ses deux axes,
      donc il ne pointe nulle part — la seule chose que la mesure autorise a dire. Traqueur =
      UNE onde, `SEEKER_IMPULSE_MS = SENSOR_SWEEP_MS` (meme rythme, ecrit et teste), rayon en
      PIXELS (`SEEKER_IMPULSE_RADIUS_PX = 14`) parce que la source officielle ne chiffre AUCUNE
      portee pour lui — puis plus rien, ni zone ni anneau ni point. Champ de reparation =
      disque a l'encre d'equipe, `REPAIR_FIELD_RADIUS_M = 3` DECLARE (les trois sources sont
      vides : film muet, source officielle muette, corpus non mesure) et sa borne est
      POINTILLEE — la meme grammaire que le mur sans cap : ce qui n'est pas affirme est en
      pointille, ce qui vient d'une source publiee est plein (l'anneau du capteur).

      Mouvement reduit : le traqueur garde un anneau IMMOBILE au rayon plein pendant la meme
      fenetre — supprimer l'onde sans rien mettre a sa place aurait rendu la famille invisible
      a qui demande moins de mouvement.

      Infobulles FR/EN par REGLE DE RENDU (`placementFamily` passe de 2 a 5 cles, parite par
      typage) ; noms FR pris au manifeste (`[ability_palettes.ranks]`) : « Traqueur de
      menaces », « Champ de reparation », « Balise du translocateur ». Bascules inchangees,
      mais `placementCounts` passe desormais par `placementKind` — un film dont toutes les
      poses sont des lachers n'allume plus une commande qui n'afficherait rien.

- [x] W.3 **SONS : AUCUN AJOUT, ET LE SILENCE EST UNE MESURE.** Releve de la bibliotheque de
      l'utilisateur : sept dossiers d'equipement (Active Camo, Drop Wall, Grappleshot,
      Overshield, Repulser, Threat Sensor, Thruster) et **zero fichier** pour le traqueur, la
      balise ou le champ (0 correspondance sur `*seeker*`, `*transloc*`, `*repair*`,
      `*quantum*` dans TOUTE la bibliotheque, EQUIPMENT/GRENADE/WEAPONS comprises).
      `EQUIPMENT_PLACEMENT_SOUND_STEMS` reste a deux entrees ; le releve et la condition de
      reprise (« il reste UNE ligne a ecrire ») sont ecrits au-dessus de la table.

      **CORRECTION DE COHERENCE PORTEE AU PASSAGE, ET ELLE SORT DE LA MESURE DE LA PHASE G** :
      `buildSoundTimeline` sonnait TOUTE pose de famille `wall` / `sensor`, lachers compris —
      91 poses de mur sur 222 et 106 de capteur sur 155 sonnaient un « deploiement » qui etait
      une mort. Et un mur reellement deploye sonnait DEUX fois (appareil + panneaux). Le son
      partage desormais `placementIsDeployedObject` avec le calque : un seul predicat, donc
      pas de derive entre ce qu'on voit et ce qu'on entend.

- [x] W.4 **TESTS ET GATES.** 102 tests sur les cinq fichiers du perimetre : table famille par
      famille (les 5 deployables, les 7 portees a `null`, les 2 power-ups hors table, une
      famille inconnue), filtre d'origine (`dropped` / `unknown` / ABSENTE), panneaux contre
      appareil (dont « un mur deploye ne rend QU UN arc »), balise sans pulsation et
      symetrique, traqueur a UNE impulsion qui ne se rejoue jamais (verifie a l'age d'un second
      ping de capteur), champ sans onde a aucun age, capteur lache qui ne revele personne,
      survol qui suit le dessin. Garde-rail neuf `placementPanels.guard.test.ts` : les deux
      identifiants du web rejoues contre les 21 lignes du TOML, dans les DEUX sens
      (`kind = deployed` <-> table web), plus la cardinalite du manifeste pour qu'une decoupe
      cassee echoue au lieu de rendre des tests vides et verts.

      Gates (cache `node_modules/.tmp` purge avant) : `npm run typecheck` exit 0,
      `npm run lint` exit 0 — **0 erreur**, 19 avertissements pre-existants dont aucun dans
      `match-replay` —, `npm run test` exit 0 : **446 fichiers, 4 083 tests verts, 14 skips**.
      Zero hex, zero classe Tailwind de couleur, parite FR/EN par typage.

**Gate W** : typecheck/lint/vitest exit 0 — PASSE. Reste le gate VISUEL utilisateur.
Ce que la mesure predit sur `000d5950` (golden en depot, 295 poses, 39 `deployed`) : **19
formes** — 15 arcs de mur (panneaux `0x528fce46`) et 4 capteurs ; les 2 appareils de mur
`deployed` et les 18 poses de familles portees ne dessinent plus rien, et 28 formes fantomes
disparaissent par rapport a aujourd'hui (11 murs laches, 15 capteurs laches, 2 appareils).
`06dfe6d9` : 123 `deployed` sur 892, repartition par identifiant non mesuree ici (artefact
absent du worktree).

**CE QUE LE GATE VISUEL NE POURRA PAS VOIR, ET IL FAUT LE DIRE AVANT** : la BALISE a **zero
pose `deployed` sur tout le corpus** (2 poses, 2 lachers) — son rendu n'a AUCUN temoin, nulle
part. Le TRAQUEUR en a **une seule** (sur 4), le CHAMP en a 26 (sur 105) mais aucun des trois
n'apparait dans `000d5950`, dont le golden ne porte que `wall`, `sensor`, `grapple`,
`thruster` et les grenades. Un gate sur `000d5950` et `06dfe6d9` valide donc le FILTRE, les
PANNEAUX et le capteur ; il ne peut rien dire des trois formes neuves. Le film a power-ups de
carte, lui, n'a plus d'objet : la famille est restee hors table (correction 3).

## Regles dures

Aucune origine presumee ; un objet de dotation ne se dessine pas ; les icones ne se pretent
pas ; le journal/registre : phase G les edite (principal), phase W fournit ses textes au CR.

## Decouvertes non traitees (portees au registre, PAS corrigees ici)

1. **LE MODE DE CHAQUE FILM EST ACCESSIBLE, et deux lignes du registre le supposaient
   impossible.** Le snapshot parquet du registre
   (`data/backups/staging/halo_infinite/shared_matches_v2/match_registry_*.parquet`) se lit par
   `read_parquet` en memoire, sans ouvrir la DB que le serveur tient RW. Les 11 films du corpus
   calibre sont donc classes :

       000d5950 Cliffhanger       Slayer:Arena Super Fiesta   00ba2e1c Obituary   BTB:Fiesta Slayer
       00162144 Smallhalla        Slayer:Arena                06dfe6d9 Threshold  BTB:Fiesta CTF
       00502e52 Bazaar            Slayer:Arena Super Fiesta   084a804d Fortitude  BTB Heavies:CTF
       07aa428d Illusion          Slayer:Arena Super Fiesta   9edfcaa9 Oasis      BTB:CTF
       331ff98d Domicile          CTF:Arena                   cfb85a58 Nemesis    CTF:Arena
       64e8adfa Catalyst          CTF:Arena

   **SIX DES ONZE FILMS NE SONT PAS FIESTA**, et le mode de `06dfe6d9` — « non documente »
   jusqu'ici — est BTB:Fiesta CTF. Cela leve la limite assumee de l'item 1.4(b) du lot de
   nommage (« le predicat demandait un film NON Fiesta, et je n'ai pas pu l'etablir ») et rend
   executable la reprise de la ligne « le lien palette <-> MODE n'est pas etabli » (« c'est une
   requete, pas une recherche, des que le registre est accessible »). **NON TRAITE ICI** : hors
   perimetre de ce lot, qui porte l'origine des poses. Les deux lignes du registre sont mises a
   jour avec la voie d'acces, pas avec le resultat.

2. **`084a804d` perd son poseur sur 42 % de ses poses (388 sur 922)**, quand les dix autres
   films sont entre 0 et 12 %. C'est le film le plus dense du corpus (256 traces, 4 623 slots
   d'archetype 37) : la fenetre de 250 ms x 3 m y trouve moins souvent un bipede unique. Aucune
   consequence sur les conclusions de ce lot (les taux `deployed` y sont les plus BAS, donc la
   perte ne fabrique pas de deploiements), mais un rendu qui filtrerait sur `origin` masquerait
   ce film plus que les autres. Non mesure plus loin.

3. **`go test ./...` EST ROUGE SUR LA BRANCHE, ET LES DEUX CAUSES SONT ANTERIEURES A CE LOT.**
   Verifie sur pieces, pas suppose :

   - **`internal/archlint` — `TestNoRawKillScopeLiteral`.** Le garde-rail J4R-3 interdit le
     litteral `"scan"` (une valeur de portee de kill feed). Le lot de nommage precedent
     (`77d77d2cf`) a introduit un DICTIONNAIRE murmur3 qui contient le mot anglais « scan »
     parmi ses jetons candidats :
     `internal/himap/sonde_stringid_dico_test.go:51`. **Faux positif du garde-rail**, et la
     preuve qu'il precede ce lot : `git show HEAD:...sonde_stringid_dico_test.go | grep '"scan"'`
     rend la ligne 51, et ni `internal/archlint/` ni ce fichier ne sont modifies par ce lot
     (`git status` vide sur les deux). **NON TRAITE** : elargir l'allowlist d'un garde-rail
     demande une justification datee (regle du depot), et ce n'est pas le sujet d'un lot sur
     l'origine des poses. Reprise : soit allowlister
     `internal/himap/sonde_stringid_dico_test.go` avec sa date et son motif (c'est un
     dictionnaire de mots, pas une seconde source de verite de portee), soit resserrer
     `killScopeRE` pour n'attraper le litteral que hors d'une liste de jetons.
   - **`internal/himap` — timeout de 602 s.** Deja au registre (« Suite complete
     `go test ./internal/himap/` au-dela du timeout Go de 10 min », 2026-08-09) :
     `TestBalayageDesCartes` rasterise 27 cartes en serie. Ce n'est pas un test rouge, c'est
     la borne de 10 min. Inchange par ce lot.

   Les paquets que ce lot touche sont VERTS (`internal/analysis/replay`,
   `internal/games/mappings`, `internal/games/halo_infinite/replaylabels`, `contracttest`).

4. **UN MUR DEPLOYE PRODUIT DEUX POSES, ET LA PHASE W DOIT LE SAVOIR** : l'appareil qui vole
   (`0x2974c233` / `0x8e2dc574`, 31 poses `deployed` au corpus) ET ses panneaux
   (`0x528fce46` / `0x686b40c9`, 89 poses `deployed`). Dessiner l'arc sur toute pose
   `wall` + `deployed` dessinerait donc DEUX arcs pour un seul mur. La regle que la mesure
   soutient : l'arc sur les PANNEAUX (`kind = deployed`), rien sur l'appareil. Decision de
   rendu, elle appartient a la phase W.
