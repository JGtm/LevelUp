# Plan — L'equipement au rejeu 2D : le nommer, le dater, le poser

> Ecrit le 2026-08-15. Il REMPLACE `PLAN_EQUIPEMENT_ACTIF.md`, dont les etapes 3 et 4 sont
> bloquees sur un canal (`i57`) que la mesure a refuse. Le present plan part d'un canal que
> personne n'avait ouvert : l'archetype `ti=37` « equipment ».
> Branche `feat/v75`. Execution sous le contrat du skill `plan-execution` (ordre strict, un
> item = un statut, zero fix hors perimetre, zero report d'une etape executable).

## Objectif et critere de succes

Montrer dans le rejeu 2D, pour chaque joueur : **QUEL** equipement il porte, **QUAND** il
l'active, **OU** l'objet pose se trouve — et le faire sonner.

Critere de succes, et il est negatif autant que positif : aucune de ces trois quantites
n'est affichee sans avoir ete MESUREE. Une quantite non etablie s'affiche comme ABSENTE,
jamais devinee. C'est la regle qui a fait echouer le lot du 14/08 et elle avait raison.

## Vision globale — les cinq canaux, et ce que chacun peut porter

| canal | archetype | ce qu'il porte | etat au 2026-08-15 |
|---|---|---|---|
| `i48` biped-desired-ability-set | 35 biped | IDENTITE : rang de palette COMPLET | **PUBLIE** (`filmdec/ability_rank.go`) — 748 lectures, 0 illisible, verite terrain 8/8 |
| champ d'ancrage image-cle | 35 keyframe | identite, **fenetre 16..23 seulement** | publie (`replay/inventory_decode.go`), borgne par construction |
| `i56` biped-spartan-ability-energy | 35 biped | ENERGIE : masque R(3) + 7 bits par charge, `0x7F` = plein | porte, valeur JETEE ; mesure existante sur UN film (176 lectures), jugee trop maigre |
| `i57` biped-spartan-ability | 35 biped | 4 etats, dont UN SEUL paie 24 bits de charge utile | porte ; mesure : 1 414 lectures, association a `i54` de 2,1x les temoins — **ne tranche pas** |
| **`ti=37` `equipment-*`** | **37 equipment** | **active / deploye / createur / energie / POSITION** | **porte, valeur JETEE, JAMAIS MESURE** |

Le dernier est le seul qui nomme la chose en clair, et le seul qui porte une POSITION.

### Verdicts du 2026-08-16 — l'ETAT ACTIF par famille (PLAN_ETAT_ACTIF_EQUIPEMENT, 5 gates passes)

Le lot de mesure « etat actif/inactif » a interroge les canaux que ce tableau ne couvrait
pas. Ce qu'il change a la vision globale :

| famille | etat actif | verdict chiffre |
|---|---|---|
| **Camouflage (rang 8)** | **SE LIT — `i28` queue[1]**, binaire 0/4095 | transitions EXCLUSIVES aux vies rang 8 (39 contre 0 sur 574 autres vies, 2 films + temoin negatif a 0) ; courbes 0->4095->0 publiees (plateau 16,2 s) — activation ET desactivation datables |
| **Surbouclier (rang 9)** | **SE LIT — `i5` NON clampe**, regle `q > 64` | 86,7-92,3 % des mesures des porteurs au-dela du plein (q max 223 = 3,498), **0 faux positif** sur ~113 000 mesures hors porteurs (3 films) ; `Point.sh` est CLAMPE — publier exigera un champ non clampe |
| **Deployables (mur 19, capteur 22)** | **NE SE DATE PAS** par les canaux mesures | `activated` : 1 transition sur 3 films ; R(24) d'i57 : PAS un handle (valeurs quasi toutes uniques, vies vivantes ±2 s = 1-3 %) ; naissances ti=37 : densite 4,7-5,3/s, temoins au niveau du reel. Reste la voie `charges-remaining` (EN RESERVE, decision user 16/08) |
| **Mobilite (grappin 20, propulseur 21...)** | `i54` REFUTE ; **grappin : `i59` tag==3** | i54 : 0,55 episode/vie (mobilite) contre 0,45 (autres) sur 12 films — action generique, pas un usage. MAIS 115/117 lectures tag==3 identifiees = vies rang 20 : l'evenement du GRAPPIN existe, et le corps non porte (`FUN_142f25e90`, position+quaternions) est le candidat ANCRE — lot justifie au registre |

Consequence sur la phase 1 ci-dessous : les canaux gagnants sont `i28` queue[1] (camo),
`i5` non clampe (surbouclier) et `i59` tag==3 (usage grappin, apres portage du corps) —
PAS les quatre champs ti=37, qui restent sans identite ni datation exploitables.

## Ce qui est ACQUIS — ne pas re-chercher

| acquis | ou | consequence |
|---|---|---|
| La palette des capacites (rangs 0..23, 12 noms) | RECETTE_LOADOUT §13 | les 7 equipements sont identifies |
| La palette varie PAR FILM et se deduit de la signature des rangs | RECETTE_LOADOUT §14-15 | ne jamais presumer la famille A |
| L'identite est PUBLIEE et nommee (8/8) | `ability_rank.go`, lot du 14/08 | le « QUEL » est FAIT |
| Les 7 desers `ti=37` sont portes | `traverse.go:277-345`, `components_batch3.go:46` | il n'y a AUCUN decodage nouveau a ecrire pour la phase 0 |
| `EquipmentTypeIndex = 37` + `ScanFilmWorldObjects` | `filmdec/projectiles.go:85-120` | les positions sont un APPEL DE FONCTION |
| Le patron de publication d'une valeur jetee | `filmdec/ability_rank.go` | hook + marche + statistiques de denominateur |
| Les sons des 7 equipements (Activate / Deactivate / Recharge / Idle) | bibliotheque utilisateur, `Downloads/Audio Library.../EQUIPMENT` | la phase 4 n'attend aucune extraction |

## Ce qui est REFUTE — ne pas rejouer

- **Le bit 0 d'`i57` comme interrupteur** : 1 sur 386 lectures sur 386. Un interrupteur qui
  ne bascule jamais n'informe de rien.
- **La traversee d'`i56`/`i57` comme cause de rarete** : ZERO composant casse la marche sur
  1 535 records. La rarete est une FREQUENCE DE TRANSMISSION, pas un defaut de decodage.
- **Elargir le champ d'image-cle a 6 bits** (`large6`/`suite6`/`aval6`) : 0/8 contre la
  verite terrain la ou `prod3` rend 6/8.
- **Presumer la famille A de palette** : trois films n'exposent que les rangs 19..22.

## Phases

Chaque phase est livrable independamment et se clot sur un gate VERIFIABLE. Une phase
n'ouvre pas tant que la precedente n'est pas close.

### Phase 0 — MESURER — CLOSE le 2026-08-15

Corpus : **12 films**, un film par processus. Les 8 dont la palette `i48` etait deja mesuree
le 14/08 — `000d5950` (verite terrain Theater), `00162144` (palette hors famille A),
`00ba2e1c`, `06dfe6d9`, `084a804d`, `00502e52`, `07aa428d`, `0014603f` (temoin negatif :
`i48` jamais au masque) — plus **4 tires a pas regulier** des 951 du cache (`331ff98d`,
`64e8adfa`, `9edfcaa9`, `cfb85a58`), pour ne pas mesurer seulement les films qui marchaient
deja. Duree 11 a 256 s par film.

- [x] 0.1 Volume `ti=37`. **1 097 619 records delta, 19 260 slots, 6 904 vies d'objet.**
      L'archetype est verifie SUR PIECES (31 composants) : les index du plan sont exacts, et
      il en porte SIX de plus, jamais cites — i25 being-hacked, **i26 energy-delay-ticks-left**,
      **i27 charges-remaining**, i28 tracked-object-handles-stack, i29 command-tick,
      i30 has-infinite-uses.
- [x] 0.2 Les quatre champs publies par hook sur le deser de PRODUCTION
      (`filmdec/equipment_state.go`, `SetEquipmentStateHook`) : **13 187 records annoncent au
      moins un des quatre (1,20 %), marche aboutie 13 099, cassee 88.** Par champ
      (masque / lu / porte fermee) : deployed 3839/3820/0 · activated 2126/2121/894 ·
      creator 2209/2194/866 · energy 5268/5217/0. **Transitions : 81 sur 208 paires pour
      `activated`, 1 601 sur 3 960 tous champs confondus.**
- [x] 0.3 `ScanFilmWorldObjects(dir, wr, 37)` : **9 043 trajectoires, 628 368 echantillons.**
      Les bornes MONDE exigeraient une base ; le controle est donc joue en coordonnees
      NORMALISEES de l'AABB, contre le nuage des BIPEDES du meme film — plus severe et sans
      dependance : **610 693 sur 628 368 (97,2 %) dedans, 12 films sur 12 (92,3 a 99,7 %).**
- [x] 0.4 Controle non circulaire du `R(5)` de `creator`. **ECHOUE, et publie tel quel :
      0 valeur sur 1 328 ne tombe dans `bipedSlotBand`.** Le repli C2 (« au moins un index
      compact ») passe mais est VACUEUX — 5 bits plafonnent a 31 et tous les films ont plus
      de 31 slots ; dit tel quel. Le seul controle qui tranche vient de la verite terrain :
      sur `000d5950`, 8 joueurs, le champ prend 15 valeurs distinctes et monte a 28 ; sur le
      corpus il couvre ses 32 valeurs avec une queue plate. **Ni slot, ni index de joueur.**
- [x] 0.5 `i56` elargi (`filmdec/i56_drops_test.go`, garde `I56_DROPS_FILM`, lecture par le
      deser de production). **2 519 042 records delta biped, 2 088 lectures (0,083 %), 0
      illisible, 1 275 slots ; 519 paires -> 282 CHUTES, 26 REMONTEES.** Les deux encodages
      sont separes : **282 chutes sur 282 franchissent un multiple de 16, aucune ne reste
      dans un quartet.** Temoins +/-5 s : **la coincidence avec `i54` est REFUTEE par
      l'elargissement — reel 7/601 (1,2 %) contre 9 (1,5 %) et 5 (0,8 %)** ; le 4,5 % du
      14/08 etait 3 episodes sur 67 d'un seul film, et l'instrument le reproduit au chiffre
      pres sur ce film-la avant de s'effondrer sur les onze autres.
- [!] 0.6 Sept equipements ou deployables seuls ? **NON TRANCHE, et la mesure ne pouvait pas
      le trancher — c'est le resultat, pas un abandon.** Compte : 9 043 trajectoires sur
      12 matchs, bien plus que des equipements de joueur ; l'archetype s'appelle `item-*`
      autant qu'`equipment-*` (i18 `item-at-rest`, i19 `item-ignore-player`), donc bonus au
      sol et socles y sont vraisemblablement melanges. **Aucun des quatre champs ne porte
      d'identite** : rien ne distingue une entite d'une autre. Porte au
      `REGISTRE_REPORTS.md` avec sa condition de reprise (chercher la reference de definition
      dans le record de CREATION, comme la famille high-32 des armes au sol).

**Gate 0 : PASSE.** Les trois verdicts, ecrits et chiffres :

1. **DATER l'activation — NON.** `activated` est transmis 2 121 fois sur 1 097 619 records
   (0,19 %) ; 949 vies d'objet sur 6 904 en recoivent une valeur, dont **227 seulement apres
   le premier record de la vie** (une valeur au premier record date la REPLICATION, pas le
   geste). Il reste 208 paires consecutives et 81 transitions sur 12 matchs, et **aucune des
   huit valeurs de R(3) n'est nommee** — elles sont toutes peuplees, aucune ne domine.
   Verdict identique pour `i56` : 282 chutes pour 1 275 vies lues, une chute pour cinq vies.
2. **QUI active — NON.** Controle prescrit ECHOUE (0/1 328), et le champ n'est pas davantage
   un index de joueur (28 sur un film a 8 joueurs, 32 valeurs a queue plate).
3. **OU l'objet est pose — OUI.** 97,2 % de 628 368 echantillons dans l'emprise des joueurs
   du meme film, 12 films sur 12. C'est le seul acquis exploitable du lot.

Instruments versionnes et gardes (`EQUIP_FILM`, `I56_DROPS_FILM`), sautes en CI.
`go build`, `go vet`, `go test ./internal/analysis/filmdec/ ./internal/analysis/replay/`
verts, `golangci-lint --new-from-merge-base=origin/main` 0 issue. Entree
`[2026-08-15]` au `thought_log.md`. **Journal detaille : `PLAN_EQUIPEMENT_ACTIF.md`
etape 5** (ce lot prolonge ses etapes 3 et 4, d'ou le journal y reste).

**CE QUE LA PHASE 0 DESIGNE POUR LA SUITE, et qui n'etait dans aucun plan.** Le recensement
au masque des 31 composants de `ti=37`, cumule sur les 12 films, place les quatre champs
mesures LOIN derriere deux autres :

    equipment-charges-remaining-component        16 125   <- 3,1x l energie, 7,6x l active
    equipment-energy-delay-ticks-left-component  10 608   <- un compte a rebours en ticks
    equipment-energy-component                    5 268
    equipment-deployed-component                  3 839
    equipment-creator-component                   2 209
    equipment-activated-component                 2 126

Les deux sont **DEJA PORTES** par `consumeByName` : il ne manque qu'un hook, comme pour les
quatre d'aujourd'hui. Un compteur de charges qui decroit date une utilisation par
construction. Hors du perimetre FERME de la phase 0 — c'est le premier geste de la suite.

### Phase 1 — PUBLIER le canal gagnant (branche sur le verdict 0) — CLOSE le 2026-08-16

> EXECUTION RESTREINTE (decision utilisateur du 16/08 : « pour active camo et surbouclier
> oui ca me va ») aux DEUX familles gagnees par PLAN_ETAT_ACTIF_EQUIPEMENT (gates A et B).
> Les canaux publies sont donc `i28` queue[1] (camo) et `i5` non clampe (surbouclier) —
> PAS les champs ti=37 (sans identite ni datation exploitables, verdict 0).

- [x] 1.1 AMENDE par la restriction : le lecteur de production est `filmdec/camo_state.go`
      (`ScanFilmCamoStates`, patron d'`ability_rank.go` : hook `camoStateHook` du deser de
      production, marche partagee `walkRecordTo` — un seul exemplaire pour i48 et i28 —,
      lectures localisees slot/chunk/packet/`TimestampUS`, `CamoStateStats` avec
      denominateurs Records/WithI28/Read/Unread/NoChannel). Le surbouclier n'exige AUCUN
      balayage neuf : le quantum brut `Shield.Q` voyage deja dans `ScanFilmBipedPositions`
      (CaptureDirs) ; la regle mesuree est figee en production (`filmdec.OvershieldFullQ`,
      q > 64, jamais la valeur clampee).
- [x] 1.2 `replay.Options.CamoStates` (patron `AbilityRanks`), peuplee par `BuildFromFilm`,
      absence NON FATALE et loggee (slog.Warn, la convention du fichier — BuildFromFilm ne
      recoit pas de contexte). L'assemblage (`equipment_episodes.go`) date les episodes PAR
      VIE : ouverture au passage actif (4095 / q>64), fermeture a la transition MESUREE
      (`endRead=true`) ou a la fin de la vie (`endRead=false` — le fil des morts date la
      fin de piste), clamp a la fenetre de la trajectoire publiee.
- [x] 1.3 `ReplayDocument.EquipmentEpisodes` (schema 6 -> 7, chronique ecrite) +
      `Coverage.Equipment` (vies porteuses / vies publiees, par famille) + contrat OpenAPI
      regenere (`make openapi-gen`, contracttest 28 -> 29 champs) + `generated.ts` +
      normalisation web (`replayNormalize.ts`, `replayContract.test.ts`).
- [x] 1.4 Fixture d'entrees v4 (`REPLAYINPUTS4` : + `CamoStates`, + `Shield.Q`) regenere
      depuis le film de reference ; golden d'assemblage regenere. SON EVOLUTION, et elle
      enseigne : `000d5950` (Fiesta) publie 36 episodes camo sur 22 vies alors qu'AUCUNE
      vie n'y porte l'equipement rang 8 — controle a l'instrument (`I28_FILM`) : 698
      lectures queue[1] STRICTEMENT binaires (0:617 · 4095:81), transitions sur des vies
      rangs 19-22. C'est le POWER-UP de camouflage qui allume le canal : i28 est l'etat
      d'invisibilite de l'UNITE, l'exclusivite rang 8 de la phase A etait la VALIDATION du
      canal sur des films sans power-up. L'etat est l'etat — on publie. Surbouclier : 0
      episode sur ce film (temoin de forme [0, 64] du 2026-08-05, reproduit au zero pres).

**Gate 1 : PASSE.** Couverture publiee dans le document (`coverage.equipment`) et mesuree
sur 4 films du corpus local (artefacts schema 7 construits par `backfill-replay` sur cache
restreint) — chiffres au journal de cloture ci-dessous. Une couverture partielle est un
resultat : la plupart des vies ne portent ni camo ni surbouclier, zero est une valeur.

Interdits de cette phase : aucun rendu, aucun son, aucune valeur par defaut inventee —
tenus (le rendu et le son sont arrives aux phases 3 et 4).

### Phase 2 — NOMMER (manifeste, jamais de litteral en dur) — CLOSE le 2026-08-16

- [~] 2.1 SANS OBJET sous la restriction : l'effet est type par la FAMILLE (`camo` /
      `overshield`), identifiants STABLES du document (meme regle que `NeutralDeath.Kind`),
      pas par un libelle de rang. Aucun libelle FR/EN en dur cote Go ; les libelles UI
      vivent dans `i18n.ts` web (`equipmentActive`, FR **et** EN, parite par typage
      `Record<'camo' | 'overshield', string>`). `replay_labels.toml` inchange.
- [~] 2.2 Couvert par l'existant : le comportement des rangs non resolus (affiches comme
      rang) n'est pas touche par ce lot.
- [x] 2.3 AUCUN nom ajoute — rien a prouver.

**Gate 2 : PASSE** — `go test ./internal/games/...` vert (aucun fichier du perimetre n'y
vit, le gate confirme l'absence de regression) ; aucun nom ajoute sans provenance.

### Phase 3 — MONTRER — CLOSE le 2026-08-16 (gate visuel utilisateur RESTANT)

- [x] 3.1 Effet PLEINE FICHE, deux effets distincts et semantiquement evidents
      (`ReplayTeams.tsx`, chaine slot -> fiche du flash de mort) : le CAMOUFLAGE ESTOMPE
      la fiche entiere (opacite 0.4 — le joueur disparait a l'ecran de jeu, sa fiche fait
      pareil ; l'infobulle dit pourquoi), le SURBOUCLIER la SURLIGNE (anneau plein + halo
      + fond au token `info` — le MEME que la jauge de bouclier : un surbouclier est un
      sur-BOUCLIER). AUCUNE remanence inventee : l'effet est actif exactement sur
      [t0, t1] de l'episode mesure (`equipmentFx.ts`, bornes incluses, teste). Les deux
      effets se composent quand les episodes se recouvrent.
- [~] 3.2 SANS OBJET sous la restriction : ni le camouflage ni le surbouclier ne POSENT
      d'objet sur la carte, et les positions ti=37 restent sans identite (verdict 0.6) —
      rien a projeter sans nommer.
- [x] 3.3 Tokens semantiques uniquement (`tokenCssVar('info')`, zero hex, zero classe
      Tailwind couleur) ; effets STATIQUES, sans animation — `prefers-reduced-motion`
      respecte par construction ; strings FR **et** EN dans `i18n.ts`, parite par typage.

**Gate 3 : gates techniques PASSES** (typecheck apres purge de `node_modules/.tmp`, lint,
test — exit 0, chiffres au journal). **Gate visuel utilisateur : RESTANT** — la session ne
juge pas son propre rendu ; films temoins conseilles : `084a804d` (10 lectures rang 8,
8 rang 9) ou `06dfe6d9`.

### Phase 4 — FAIRE SONNER — CLOSE le 2026-08-16 (gate d'ecoute utilisateur RESTANT)

- [x] 4.1 RESTREINT aux deux familles gagnees : `Active Camo - Activate/Deactivate` et
      `Overshield - Activate/Deactivate` convertis par la recette VALIDEE du lot grenades
      (PCM s16le, 48 kHz, stereo, tronque a 1,200 s, `-map_metadata -1` + `-bitexact`) —
      temoin RECONSTRUIT : `M9 Frag Grenade - Explode.wav` re-converti rend un fichier
      IDENTIQUE A L'OCTET (SHA256) a `explosion_frag.wav` livre, et les 4 fichiers font
      exactement 230 444 o. Declenchement : debut d'episode = Activate ; fin MESUREE
      (`endRead`) = Deactivate. DEUX CHOIX DOCUMENTES pour le gate d'ecoute : (a) la fin
      mesuree du surbouclier est l'EPUISEMENT (retour sous 100 %) — y jouer Deactivate est
      une mise en scene ; (b) un episode ferme par la MORT ne sonne PAS de desactivation
      (rien ne l'a mesuree, et le kill sonne deja la).
- [!] 4.2 Les `Recharge` ne sont PAS joints : ni la phase 0 ni les phases A/B n'etablissent
      une remontee d'energie DATEE (la recharge n'a pas d'instant mesure — et un asset que
      rien ne joue casserait le garde-rail). Les WAV restent dans la bibliotheque
      utilisateur, non verses.
- [x] 4.3 Une famille hors table (`EQUIPMENT_SOUND_STEMS`) reste MUETTE, jamais le son
      d'une voisine (teste) ; garde-rail `replaySoundAssets.guard.test.ts` etendu aux 4
      stems (manifeste = dossier, 0 asset mort).

**Gate 4 : garde-rail et gates web VERTS** (journal). **Gate d'ECOUTE utilisateur :
RESTANT** — memes films temoins que le gate visuel. Le FILTRE PAR CATEGORIE de sons
(crainte de surcharge sonore) n'est PAS implemente : idee portee au REGISTRE_REPORTS avec
sa condition de reprise (decision user au gate d'ecoute).

## Journal de cloture des phases 1-4 (2026-08-16, execution restreinte camo + surbouclier)

**COUVERTURE PUBLIEE (`coverage.equipment`), mesuree sur 4 films du corpus local** — les
artefacts schema 7 sont sur disque (`data/cache/replays/halo_infinite/`) :

    film      contexte                        vies   camo (vies/episodes)  surbouclier (vies/episodes)
    000d5950  Fiesta arene (golden, fam. B)     99   22 / 36 (24 fins mesurees)   0 / 0
    00ba2e1c  BTB Fiesta Slayer (temoin neg.)  208    0 / 0                        0 / 0
    084a804d  BTB Heavies CTF (fam. A)         256    9 / 15 (10 fins mesurees)    5 / 6 (6 fins mesurees)
    06dfe6d9  NON CONSTRUCTIBLE : carte `Threshold` hors catalogue de bornes — echec VOULU
              du pipeline (une carte sans bornes ne produit pas d'artefact), pas un defaut du lot

**TROIS CONTROLES QUE CES CHIFFRES PORTENT.** (1) Le temoin negatif de la mesure
(`00ba2e1c` : 0 transition, 0 valeur 4095, 0 q>64 sur tout le film) est reproduit par la
production AU ZERO PRES. (2) Sur `084a804d`, l'episode de surbouclier le plus long publie
dure 61,6 s — LE chiffre du gate B (« episodes dates de 6,2 a 61,6 s ») retrouve par une
chaine independante de l'instrument. (3) Sur `000d5950` (jamais mesure en phase A), les
36 episodes camo sans porteur rang 8 ont ete CONTROLES a l'instrument (`I28_FILM`) avant
d'etre acceptes : 698 lectures queue[1] strictement binaires (0:617 · 4095:81) — c'est le
POWER-UP de camouflage de Fiesta ; i28 est l'etat d'invisibilite de l'UNITE, quelle qu'en
soit la source. L'etat est l'etat.

**GATES TECHNIQUES, resultats exacts du 2026-08-16 :**

    go build ./...                                          exit 0
    go vet ./...                                            exit 0
    go test ./internal/analysis/... ./internal/replaybuild/...   exit 0
    go test ./internal/games/...                            tous paquets ok (gate 2)
    golangci-lint run --new-from-merge-base=origin/main     0 issues
    web (purge node_modules/.tmp puis) : typecheck exit 0 · lint exit 0 (0 erreur ;
      19 warnings preexistants, aucun dans les fichiers du lot) · vitest complet
      exit 0 (423 fichiers, 3 807 tests ; passe 1 : 1 flaky CONNU hors perimetre,
      PalmaresRelationsPage, vert seul — 2e observation consignee au registre)

**GATES RESTANTS, et ils appartiennent a l'utilisateur** : le gate VISUEL (phase 3) et le
gate d'ECOUTE (phase 4). Film temoin conseille : `084a804d` (9 vies camo, 5 vies
surbouclier, artefact schema 7 sur disque). `06dfe6d9` n'est pas constructible (carte
hors bornes) ; `000d5950` montre le camo de power-up sans surbouclier.

## Regles dures (elles ont deja tranche ce chantier)

1. **Aucun effet, aucun son, aucun nom sans donnee mesuree.** Le lot du 14/08 a eu raison de
   ne rien afficher.
2. **Un rang non resolu s'affiche comme rang.**
3. **Offline-pur et universel** : pas de Cheat Engine, pas de capture runtime.
4. **Un seul decodage `filmdec` par process** (hooks globaux, `LockProcessDecode`).
5. **Aucune base DuckDB ouverte en ecriture** : le serveur de l'utilisateur tourne.
6. **Zero fix hors perimetre.** Toute decouverte va en section « Decouvertes », pas dans le
   diff.

## Statuts d'item et cloture

`[x]` fait · `[~]` couvert ailleurs (avec la reference) · `[!]` non traite (avec la
justification ecrite). **Aucune case vide a la cloture d'une phase.**

## Decouvertes — a consigner, NE PAS traiter dans ce chantier

- Type de grenade des 2 etiquettes AMBIGU (`31e8d17e`, `88f1034c`) : piste non prise —
  joindre un kill a la grenade au dernier LANCER du meme joueur (`doc.grenades` porte le
  type 0..3 et l'auteur). **Validable sans rien affirmer** : les 15 tags VALIDE forment un
  oracle ; mesurer le taux d'accord de la jointure sur eux AVANT de l'appliquer aux deux
  ambigus. A quantifier d'abord : quelle part des morts ces 2 tags representent (jamais
  compte).
- `i59 biped-spartan-ability-non-predicted-state`, branche `tag==3` : corps
  `FUN_142f25e90` = vecteur position + quaternions + dequantifications, NON PORTE. Candidat
  a la pose d'un mur ou a l'ancre d'un grappin si `ti=37` ne les couvre pas.
- `i51 biped-emp-timer` : un TIMER, jamais interroge.

## Protocole de reprise de session

1. Lire ce fichier de haut en bas : les cases cochees disent ou en est le chantier.
2. Lire l'entree la plus recente de `.ai/thought_log.md` portant « equipement ».
3. Lire `.ai/V7.5/REGISTRE_REPORTS.md` pour les reports en cours.
4. **Verifier sur pieces** avant de coder : rouvrir le fichier et la ligne cible, le code a
   pu bouger.
