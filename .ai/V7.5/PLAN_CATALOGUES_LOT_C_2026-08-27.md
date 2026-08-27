# PLAN — LOT C : Catalogues de cartes (bornes + objectifs), deblocages Oddball/Assaut/Land Grab

> Ecrit le 2026-08-27 par la session superviseur. Ce lot leve le verrou COMMUN nomme par les
> lots O et A : des catalogues de cartes incomplets excluent des films (Oddball : Live Fire
> x2 sans bornes de quantification ; Lattice sans catalogue d'objectifs) et vident des gates
> par construction (Assaut : 0 site `assault_bomb` sur les 5 cartes du corpus). Il absorbe le
> lot L (cablage `landgrab_zone`), meme famille de travail. Branche : `wt/catalogues`
> (worktree `../LevelUp-wt-catalogues`), base `b14140106`.
> Contrat : skill `plan-execution`. Regles du chantier : filmproc pour tout decodage, gates
> ecrits ici et jamais abaisses, decouvertes au §5 non traitees, aucun fix hors perimetre.
> IMPORTANT : ce lot repare des CATALOGUES (donnees derivees du jeu installe, pratique
> etablie des chantiers cartes) — l'extracteur de rejeu livre reste offline-pur : il ne lit
> que le film et les catalogues.

## 0. CE QUI EST SU (verifie les 27/08, ne pas re-etablir)

- `map_quant_bounds.json` : Live Fire ABSENTE — les 2 films Oddball Live Fire (`60ae07c4`,
  `c88ec007`) sont INDECODABLES (lot O, O0.2). Lattice y est (le film `92f18088` decode).
- `map_objectives.json` : Lattice ABSENTE (0 socle `oddball_spawn` — reserve consignee au
  protocole D10) ; AUCUN site `assault_bomb` pour Origin, Curfew, Absolution, Rat's Nest,
  Urban Raid (les 4 seuls du catalogue : Isolation, Snowbound, The Pit, High Ground).
  Deux hypotheses NON departagees (CR lot A) : (H1) la variante `.mvar` extraite le 25/08 ne
  portait pas les objets d'Assaut ; (H2) hash de role NON RESOLU sur les cartes recentes
  (patron KOTH, `mapvar/objectives.go`).
- Land Grab : `ObjectiveTypeOf` le classe `zone` (`extract.go:135`) mais AUCUNE entree de
  `objective_roles.toml` ne le sert, hashs `landgrab_zone` presents dans les fichiers de
  carte sans role associe (§2.8 du plan obj-etat, incoherence nommee). Aucun film Land Grab
  (expire) : le cablage sert les matchs FUTURS. Decision produit : OUI, cabler (plan
  chantier §1.5).
- Outillage d'extraction existant (chantiers cartes) : `cmd/mapquant-build`,
  `cmd/mapobj-build`, `cmd/mapstruct-build`, decodeur `mapvar` ; point d'entree documente :
  `.ai/V7.5/cartes/HANDOFF_MAP_GEOMETRY_FROM_MODULES.md` et les en-tetes des CLIs. Le jeu
  est installe sur ce poste (chemins : voir les defauts/docs des CLIs — les LIRE, ne pas
  deviner).
- Gate A1 (identite bombe), a rejouer TEL QUEL apres reparation :
  `assaut_a1_identite_test.go` — UN SEUL mot, LE MEME sur >= 2 films, temoin = 0 autre
  candidat. Mots recurrents deja publies (premiers a confronter, sans en faire le gate) :
  `0x3FEE4FCF` (7/7 films), `0xE9E7FF79` (4). Publication A2 (si gate tenu) : patron du
  crane libre — famille `bomb` au manifeste `replay_labels.toml` (EN+FR), garde d'exclusion
  des socles etendue, calque `objectiveObjects` web, re-cuisson des TEMOINS seulement avec
  verification du CONTENU (recette du registre : racine temporaire a jonctions — config du
  worktree, data du principal).
- Schema 21 / contrat 39 intacts. L'ajout d'une famille au manifeste est de la DONNEE
  (verifier quand meme ; si un bump s'impose : STOP et arbitrage au CR).

## 1. PHASES

> Ordre strict C1 -> C2 -> C3 -> C4 -> C5. Items [x]/[~]/[!], commits `catalogues-c(<n>):`,
> jamais `git add -A`, pas de push, thought_log/REGISTRE jamais touches (textes au CR).
> Donnees via `LEVELUP_REPO_ROOT` (depot principal), DuckDB en lecture seule. Un seul
> build/test Go a la fois. AUCUNE re-mesure des campagnes D4-D10 (interdit — le lot O est
> clos) : ici on repare des catalogues et on rejoue les seuls gates A1 (+A2 si tenu) et la
> QUALIFICATION des films Live Fire.

### C1 — Bornes de quantification : Live Fire

- [x] C1.1 Etablir POURQUOI Live Fire manque a `map_quant_bounds.json` (lire l'outil qui
      l'a produit et sa source ; Live Fire est une carte du jeu de base — l'absence a une
      cause, la nommer).
      — Fait 2026-08-27. Cause a DEUX etages : (1) le module prouve `sgh_interlock`
      (level_id 1253388187, unicite 1/1) ne porte AUCUN tag sbsp — ses 4 regions de
      compression vivent dans `ds/globals/common`, referencees par GlobalID depuis le levl
      (deux blocs du levl donnent le meme ordre : 7047b96f, d88e1d88, a59f5052, 91c336c1) ;
      (2) la region JOUEE est la 1 (d88e1d88), pas la 0, et l'index de region d'i0 fait
      2 bits (4 regions) la ou toute la chaine supposait 1 bit — sans cette extension,
      une simple entree de bornes n'aurait PAS fait decoder les films (le decodeur exige
      l'en-tete d'i0 nul, donc region 0). Preuves : ancres d'objectifs toutes dans d88e1d88
      (plus petite englobante), index 01 sur 59 376/59 377 records i0 des 2 films,
      decoupage lu [13 12 11] au gate 5 = [12 12 11] a l'index pres.
- [x] C1.2 Produire les bornes de Live Fire avec l'outil existant, MEME METHODE que les
      cartes presentes (aucune borne bricolee a la main).
      — Fait 2026-08-27 : `himap.RegionsBSPExternes` (le critere moteur de sbsp_region.go
      resolu a travers ds/globals), declaration explicite de la region dans
      `cmd/mapquant-build` (patron des preuves de mapModule), preuve statique rejouee en
      continu par `himap.TestPreuveRegionsLiveFire` (ordre des regions + 48 ancres +
      plus petite englobante). Catalogue etendu : champs `region`/`regionIndexBits`
      (omitempty — les 78 entrees existantes ne changent pas d'un octet, diff verifie),
      chaine de decodage etendue (layout bipede depuis le catalogue dans BuildFromFilm et
      les instruments, filtre de region dans matchBipedHeader, IndexW world-object pose par
      SetWorldObjectPrecisionFromLayout), controle TestControleBornesFilms amende (attendu
      X += bits-1 — le cas que son en-tete predisait). Regeneration complete par le CLI :
      seule l'entree `live fire` s'ajoute (79 cartes).
- [x] C1.3 GATE C1 : les 2 films Oddball Live Fire DECODENT (un film par processus) et leur
      pont bipede se mesure (instrument de qualification du lot O rejoue TEL QUEL) ; publier
      film -> taux -> admis/exclu au critere 50 % inchange. Log fige
      `C1_livefire_qualification.log`. (Qualification SEULEMENT — le corpus Oddball passe de
      4 a N films consigne pour une reprise future ; aucune remesure D9/D10.)
      — Fait 2026-08-27, log fige + controle standard du catalogue ACCORD 2/2. Tableau :
      `c88ec007` -> pont 124/147 = 84,4 % -> ADMIS (34 vies libres du crane, 1 socle) ;
      `60ae07c4` (2024-10) -> DECODE en monde (13 430 ancres, 146 creations, 502 pistes
      delta — le catalogue n'est plus la cause) mais EXCLU : empreinte ECS du film
      INCONNUE (version de film anterieure), 0 creation au mot elu du crane, l'instrument
      s'arrete proprement AVANT le pont (le cas prevu par O0.2 : « un film indecodable est
      EXCLU avec sa raison, pas repare » — decouverte au §5). Corpus Oddball mesurable :
      4 -> 5 films. Les sorties diagnostiques D8.x du log sont la sortie de l'instrument
      rejoue : elles n'amendent AUCUN verdict de campagne (interdit respecte).

### C2 — Catalogue d'objectifs : sites d'Assaut + oddball_spawn de Lattice

- [x] C2.1 Departager H1/H2 sur pieces : ce que la variante `.mvar` des cartes du corpus
      contient reellement (Origin, Curfew, Absolution en priorite ; Rat's Nest et Urban
      Raid si le meme chemin les sert), et ce que le decodeur `mapvar` resout ou laisse en
      hash inconnu. Verdict ecrit avec fichiers/hashs a l'appui.
      — Fait 2026-08-27. Les `map.mvar` des 5 cartes ont ete re-telecharges par la chaine
      du CLI (dry-run + save-mvar, versions du jour) et dumpes objet par objet. VERDICT :
      H1 et H2 sont TOUTES LES DEUX vraies, chacune a moitie — (H1) AUCUN objet ne porte
      le role historique `assault_bomb` (-534119345) NI meme `assault_include` sur les 5
      cartes (0/26 330 objets) ; (H2) le motif d'Assaut existe sous DEUX hashs de label
      NON RESOLUS, stables sur les 5 cartes : -1537427652 = position CENTRALE neutre
      (1/carte), -1843278509 = positions de BASE par equipe (2/carte) — topologie exacte
      du mode (bombe neutre au centre, sites aux bases), objets co-portant oddball_spawn/
      minigame_include (marqueurs generiques de spawn). Chasse murmur3 (TestHuntLabels,
      66 radicaux x 32 suffixes = 2173 candidats) : AUCUN nom — patron KOTH (hash sans
      nom). Au passage : minigame_exclude (-1047411729) et oddball_exclude (1191941951)
      resolus sur Absolution (candidats a labelNames, decouverte §5). Lattice : son
      map.mvar PORTE le socle oddball_spawn (label resolu) — la carte n'avait simplement
      jamais ete extraite.
- [~] C2.2 Ajouter les sites `assault_bomb` des cartes du corpus au catalogue par la MEME
      chaine que les autres roles (donnee, pas de code special-case) ; idem
      `oddball_spawn` de Lattice.
      — Lattice : FAIT (mapobj-build --from-file, entree complete : 9 objectifs dont
      1 oddball_spawn, level_id -992358985 = fo13_frost, version/nom completes depuis la
      resolution reseau de la meme session). Sites d'Assaut : NON AJOUTES — gouverne par
      le gate C2.3, non tenu (voir ci-dessous) ; les candidats restent figes au registre
      (`C2_sites_candidats.json`) pour la reprise.
- [!] C2.3 GATE C2 (ecrit ici) : pour chaque carte reparee, les sites sont DANS les bornes
      de la carte, leur COMPTE est publie (attendu : 1-2 sites d'armement par carte
      d'Assaut, 1-2 socles de crane pour Lattice), et un controle de coherence spatiale
      passe : en One Bomb/Neutral Bomb, chaque EXPLOSION datee par le score de mode (releves
      A0.3, commits du lot A) doit avoir eu de l'activite de joueurs a proximite du site
      dans les secondes qui precedent — accord >= 75 % des explosions a <= 10 m d'un site
      ajoute, temoin sites decales de 12 m <= 25 %. Log fige `C2_sites_controle.log`.
      Si le controle spatial rate : les entrees ne rentrent PAS au catalogue, `[!]` chiffre.
      — ASSAUT : GATE NON TENU 2026-08-27, chiffre : signal MAXIMAL (25/25 explosions des
      8 films avec activite au site — V1 dmin mediane 0,91-3,54 m, V2 presence soutenue
      d'un meme joueur au meme site 25/25) mais TEMOIN a 75-100 % contre <= 25 exige, sur
      la V1 ET la V2 (amendement UNIQUE declare de la definition d'activite, seuils du
      plan intacts). CAUSE NOMMEE par les dmin temoin de V1 (0,22-5,97 m sur les arenes) :
      le temoin est INSATURABLE a cette echelle — decalage 12 m < 2 x rayon 10 m (disques
      recouvrants) ET l'activite de 8-16 joueurs couvre le voisinage de tout point des
      zones de combat dans une fenetre de 5-15 s. Ce zero refute l'INSTRUMENT « activite
      de joueurs », pas les sites (motif structurel intact). RIEN n'entre au catalogue
      pour l'Assaut. Condition de reprise (arbitrage superviseur au CR) : temoin
      d'echelle coherente (decalage >= 4 x rayon d'activite), OU validation par la
      naissance de l'OBJET (gate A1 : jambe temporelle independante des sites + temoin de
      selectivite interne — non circulaire).
      — LATTICE : GATE TENU — 1 socle (attendu 1-2), team -1, @ (-59.70, -34.42, 53.91),
      DANS les bornes (X[-231;231.6] Y[-227;226.3] Z[-946;242.3]) ; le controle spatial
      d'explosions ne s'applique qu'aux modes bombe (lettre du gate). L'entree Oddball
      d'objective_roles.toml sert deja `oddball_spawn` : le film 92f18088 a desormais son
      socle.

### C3 — Rejouer le gate A1 TEL QUEL ; publier A2 si et seulement s'il tient

- [!] C3.1 `assaut_a1_identite_test.go` rejoue sans AUCUNE modification de seuil ni de
      logique (seul le catalogue a change). Log fige `C3_identite_bombe_rejeu.log`.
      — SANS OBJET 2026-08-27 : la precondition (« seul le catalogue a change ») n'est
      pas remplie — le gate C2.3 a interdit l'entree des sites d'Assaut au catalogue.
      L'instrument lit les sites du catalogue, inchange pour ces cartes : le resultat est
      determine par construction. UN film echantillon rejoue en confirmation (35b75a31,
      log fige) : « 0 site assault_bomb », GATE A1.3 = 0 candidat — bit-identique au
      verdict du lot A. La bombe reste [!] par ANCRAGE ABSENT.
- [!] C3.2 Si gate tenu (UN mot, >= 2 films, temoin 0) : publication des vies libres de la
      bombe au patron du crane (les 4 items A2.1-A2.4 du plan lot A, repris tels quels :
      manifeste famille `bomb` EN+FR, garde d'exclusion, calque web, re-cuisson temoins
      avec verification de CONTENU). Si un bump de schema s'avere requis : STOP, arbitrage.
      — Tombe avec C3.1 (pas de mot MPP, pas d'id a ecrire au manifeste).
- [x] C3.3 Si gate rate : `[!]` avec chiffres, rien ne se publie (pas de 2e essai).
      — Fait 2026-08-27 : [!] consigne avec chiffres au log C3, rien ne se publie. Le lot
      AJOUTE a la condition de reprise du lot A : les sites candidats sont desormais
      FIGES et localises (C2_sites_candidats.json), 25/25 explosions avec activite au
      site — il ne manque que l'arbitrage sur le temoin d'entree (ou la validation par le
      gate A1 lui-meme avec les candidats en entree d'instrument, non circulaire).

### C4 — Land Grab : cablage du role (ex-lot L)

- [x] C4.1 Role `landgrab_zone` au decodeur `mapvar` + entree `objective_roles.toml`
      (match Land Grab), degradation par absence de donnee — patron des roles existants,
      zero special-case dans le code.
      — Fait 2026-08-27. Les NOMS sont RESOLUS (pas des hashs muets comme KOTH) : chasse
      murmur3 rejouable (TestHuntLabels sur cliffhanger_map.mvar versionne) —
      landgrab_include = -886053664 (18 objets), landgrab_zone = 996801386 (9 volumes a
      forme) + 9 marqueurs [-941529218] non resolus (n'attribuent aucun role, regle de
      Bastion/KOTH) ; motif volume+marqueur identique a Bastion ; census : 29 entrees du
      catalogue porteuses du marqueur. labelNames + RoleLandGrabZone + roleByLabel
      (auto-verifies par TestLabelTableIsSelfConsistent), roles admis + surfaciques du
      loader, entree [[modes]] Land Grab neutral=true avec l'avertissement VIVIER (9
      declarees, 3 actives par vague — la lecon Total Control ecrite dans le TOML).
- [x] C4.2 GATE C4 : tests unitaires du decodeur/roles verts ; sur une carte porteuse de
      hashs `landgrab_zone`, les formes sortent du catalogue (verification statique par
      test, pas de film requis) ; le commentaire d'incoherence du §2.8 (dans
      `service/replay_map_objectives.go`) est mis a jour dans le MEME commit.
      — Fait 2026-08-27 : `TestLandGrabZonesCliffhanger` (9 zones, toutes avec forme, sur
      la fixture versionnee) + tests mapvar/mappings/service verts (le verrou
      FichierDuDepot passe a 8 modes avec assertion positive landgrab servi+neutre) ;
      Cliffhanger RE-EXTRAITE par la chaine (map.mvar version 5a78537e du jour) : l'entree
      du catalogue porte 9 landgrab_zone/9 avec forme, hashs sortis du census unresolved ;
      commentaire §2.8 SOLDE dans le meme commit.

### C5 — Cloture

- [ ] C5.1 Gates du lot : `go build ./...` exit 0, `go vet` propre sur packages touches,
      `go test` des packages touches ; si C3.2 a publie : contracttest + `tsc -b` (cache
      purge) + vitest `match-replay` + lint web 0 erreur.
- [ ] C5.2 Plan statue (0 case vide), logs figes commites, CR avec textes journal/registre.

## 2. COMPTE-RENDU ATTENDU

Cause de l'absence Live Fire ; verdict H1/H2 ; comptes de sites par carte + controle spatial
chiffre ; qualification des 2 films Live Fire ; verdict du rejeu A1 (LE mot ou echec
chiffre) ; si publication : familles, comptes de vies libres par temoin re-cuit, gates web ;
etat Land Grab ; commits ; textes thought_log + registre ; decouvertes.

## 5. DECOUVERTES (a consigner, ne pas traiter)

- (C1, 2026-08-27) **`60ae07c4` (Oddball Live Fire, 2024-10) porte une empreinte de
  registre ECS INCONNUE** (5686524277687893529, connue 7053924395561516366) : le film
  decode en monde (ancres, creations ti=42, pistes delta) mais AUCUNE creation ne se
  resout au mot MPP du crane — grammaire des composants d'une version de jeu anterieure.
  Qualifier les vieux films exige un travail de grammaire PAR VERSION (largeurs MPP,
  tables de composants) — chantier de fond, PAS ce lot. Le film reste exclu avec sa raison.
- (C1, 2026-08-27) Le lecteur world-object dequantifie tous les records aux largeurs de LA
  region cataloguee ; un record d'une AUTRE region (ordre du 1/59 377 observe) consomme en
  realite d'autres largeurs et desaligne SON record. La table par region
  (`SetAbsPerIndexAxisW`) existe pour le chemin sim-state ; l'etendre au lecteur
  world-object attendra une carte ou le cas pese. Limite ecrite dans
  SetWorldObjectPrecisionFromLayout.
- (C2, 2026-08-27) **Deux labels resolus par la chasse murmur3, candidats a `labelNames`**
  (garde-fou : coherence semantique OK, temoin d'execution a poser avant l'entree) :
  `minigame_exclude` = -1047411729 et `oddball_exclude` = 1191941951 (observes sur
  Absolution). NON AJOUTES ici (hors perimetre — aucun role, purs filtres).
- (C2, 2026-08-27) **Rat's Nest et Urban Raid restent HORS du catalogue d'objectifs** :
  leurs map.mvar sont telecharges (scratch de session) et parses (23 et 9 objectifs de
  roles RESOLUS — hill, oddball_spawn, strongholds_zone...), l'ingestion est a une
  commande pres (`mapobj-build --from-file`). Non faite ici : le plan ne demandait que
  les sites d'Assaut (non tenus) et Lattice. A la prochaine passe catalogue.
- (C2, 2026-08-27) Les map.mvar re-telecharges ne sont PAS versionnes (0,77-0,99 Mo
  chacun ; les 3 .mvar du dump versionne font 17-94 Ko). Le catalogue porte map_id +
  version_id + mvar_file : re-telechargeables par la chaine documentee du CLI.
