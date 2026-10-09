# Plan — Le drapeau OBJET (mot MPP `0x2A392328`) : sa piste libre a cote des portages

> Ecrit le 2026-08-18 par la session de pilotage, a la suite de la phase 0 du plan attachement
> (registre : « le drapeau de CTF est un objet `ti=42` identifie : mot MPP `0x2A392328` ») et de la
> publication des portages (`flagCarries`, schema 14). Contrat `plan-execution`. Worktree frere.

## Acquis (ne pas re-mesurer)

- `0x2A392328` revient sur les 3 films CTF et 2 cartes : 110/41/46 creations `ti=42`, dont 41/16/18 a
  0,0 m d'un `flag_spawn`, une a 1 ms d'un evenement de l'oracle ; aucun autre mot ecarte du catalogue
  d'armes ne fait les deux (`attachement_phase0_drapeau_test.go`).
- La chaine des socles (`replay/ground_weapon_*`, `filmdec/ground_weapon_creation.go`) ECARTE
  aujourd'hui ce mot (identite hors catalogue) : le drapeau libre n'est ni socle ni piste.
- `flagCarries` (schema 14) publie `carried` / `carried_open` / `dropped` / `home` ; `dropped` = derniere
  position du porteur, faute de piste propre ; le lacher n'est pas date (`carried_open`, registre).
- Un objet du monde replique sa position quand il est LIBRE (chemin delta, `ScanFilmWorldObjects`),
  cesse quand il est porte : la piste du drapeau libre est donc lisible a l'image, ses fins de vie
  (nouvelle creation au socle) aussi.

## Decisions tranchees

1. `0x2A392328` entre au manifeste (`config/titles/halo_infinite/mappings/replay_labels.toml`) comme
   famille `flag` (libelle FR/EN via le TOML, jamais en dur) ; la chaine des socles le RECONNAIT et
   l'EXCLUT des `weaponPads` (un drapeau n'est pas un socle d'arme) — garde-rail.
2. Publication : `flagObjects` [{ team (socle `flag_spawn` le plus proche de la premiere creation),
   spans [{ t0, t1, points [{t, x, y}] }] }] = les vies LIBRES du drapeau (creation -> fin de piste),
   dans le meme fichier `document_objectives_live.go` ; `flagCarries.dropped` prend la position de la
   piste libre quand elle existe (sinon inchangee) et le lacher devient DATE quand une vie libre
   commence pendant un `carried_open` (=> `carried` ferme a t0 de la vie libre : la reprise du registre
   « un canal qui date le lacher »). Schema 15, contrat +1, chronique, OpenAPI, `generated.ts`,
   `NULLABLE_ARRAYS`, goldens, temoins CTF re-cuits ; films non-CTF : vide.
3. Controle (seuils AVANT mesure) : chaque vie libre commence a < 1,5 m d'un `flag_spawn` OU a < 1,5 m
   de la derniere position du porteur qui vient de finir (>= 90 % des vies) ; temoin : creations `ti=42`
   d'armes ordinaires <= 20 % ; sinon negatif ecrit, `flagObjects` non publie.
4. Rendu : phase 3 du plan objectifs vivants (a partir d'`objectivesLayer.ts`), pas ici.
5. **ARBITRAGE DU 2026-08-18 (superviseur), qui SCINDE la decision 2.** Le controle 3 refuse la
   PISTE (75,6 % contre 90 %) mais il ne refuse pas ce que la piste CORRIGE : sa branche
   « porteur » est precisement la sous-population sur laquelle les deux corrections s'appliquent.
   La decision 2 se lit donc ainsi : `flagObjects` n'est PAS publie ; la datation du lacher et le
   repositionnement du `dropped` LE SONT, restreints aux vies libres nees a moins de 1,5 m de la
   derniere position du porteur qui vient de finir — jamais a celles nees a un socle
   (`flagFreeNearSpawn` les ecarte explicitement) ni ailleurs. Le schema monte quand meme a 15 :
   le CONTENU de `flagCarries` change (portages fermes, `dropped` deplaces) sans qu'aucune cle ne
   bouge, et un artefact 14 doit se lire « a re-cuire ». Contrat inchange en nombre de champs ;
   `coverage.flagCarries` gagne trois compteurs (sous-schema).

## Phases

- [x] 1 Manifeste + exclusion des socles + garde-rail ; instrument de mesure du controle 3 (garde `OBJ_FILM`).
- 2 SCINDEE PAR L'ARBITRAGE DU 2026-08-18 (decision 5 ci-dessous), parce que le controle 3 refuse
      la PISTE sans refuser ce qu'elle CORRIGE :
  - [!] 2a `flagObjects` (la piste publiee) : NON PUBLIE. Le CONTROLE 3, ecrit avant la mesure,
        la REFUSE — 149/197 = 75,6 % (seuil 90 %). Decision 3 appliquee telle qu'ecrite.
  - [x] 2b Les deux CORRECTIONS de `flagCarries` : lacher volontaire DATE (`carried_open` ->
        `carried`) et `dropped` repositionne sur la piste libre. Elles ne touchent QUE les vies
        nees aux pieds d'un porteur — la sous-population que le controle 3 VALIDE. Schema 15,
        3 compteurs de couverture, OpenAPI + `generated.ts` + golden + 4 temoins re-cuits.
- [x] 3 Registre/journal (textes au CR), plan statue.

- [!] 4 LE LANCER (hypothese de l'utilisateur, 2026-08-18) : REFUTEE PAR LA MESURE, et refutee
      DEUX FOIS — par le balayage et par la distribution. La regle ELARGIE (branche porteur a
      R m dans les 2 s ; branche socle, seuils 90 %/20 % et reference du porteur INCHANGES ;
      R balaye 1,5 / 3 / 5 / 8 / 10 m) n'atteint 90 % A AUCUN RAYON, et le temoin creve son
      plafond de 20 % des R = 8 m. Instrument `drapeau_lancer_controle_test.go`, ecrit et
      COMMITE AVANT la mesure (`7f74a43fa`). `flagObjects` reste non publie (2a inchangee),
      et les corrections de `flagCarries` ne sont PAS etendues : aucun R n'a ete retenu.

**Gate** : controle 3 tenu ; contrat/OpenAPI/goldens/web verts ; portages inchanges ; sinon `[!]`.

## Journal du plan
- 2026-08-18 — plan ecrit, agent lance (worktree frere `../LevelUp-wt-drapeau`, base `9b2aff1c2`).
- 2026-08-18 — PHASE 1 CLOSE. `0x2a392328` entre au manifeste sous une table NEUVE
  (`[[objective_objects]]`, famille `flag`, libelles FR/EN) : ce n est pas `equipment_objects`
  (archetype 37, chaine `sofd -> sofa -> eqip`) mais l archetype 42, celui des armes au sol.
  La chaine des socles le RECONNAIT et l ecarte AVANT la question « est-ce une arme ? » — hier
  il etait ecarte par accident (hors catalogue), ce qui ne se maintient pas. La couverture gagne
  `groundWeapons.objectives` (sous-ensemble NOMME de `rejected`) : sans lui, un drapeau reconnu
  et un octet de bruit sortent par la meme porte. Garde-rail a temoin
  (`ground_weapon_flag_exclusion_test.go`) : une famille DU CATALOGUE D ARMES declaree drapeau
  ne fait plus de socle, et le meme scan sans la table en fait un — sans ce temoin, le test
  vert ne prouverait rien. Instrument du controle 3 ecrit
  (`drapeau_objet_controle_test.go`, garde `OBJ_FILM` + `OBJ_REPO`), seuils 90 %/20 % et
  fenetre de lacher 1 s ECRITS AVANT la mesure ; rayon = `originDropMaxDist`, jamais redeclare.
  Gates verts : go build/vet/test (replay, games, contracttest, replaybuild, archlint),
  golangci 0 issue, tsc, eslint, vitest 1715/1715.
- 2026-08-18 — PHASE 2 : LE CONTROLE 3 REFUSE LA PISTE. `[!]`, et la decision 3 s'applique telle
  qu'elle a ete ecrite — negatif ecrit, `flagObjects` NON PUBLIE.
  LA MESURE, sur les trois films CTF du corpus (`drapeau_objet_controle_test.go`, garde
  `OBJ_FILM` + `OBJ_REPO`) : **149/197 = 75,6 %** des vies libres naissent a moins de 1,5 m d'un
  `flag_spawn` ou du porteur qui vient de finir, contre **>= 90 %** exige. Par film :
  `64e8adfa` 79/110 = 71,8 % (29 au socle, 55 au porteur) · `530820e5` 33/41 = 80,5 % (15/22) ·
  `53ce4390` 37/46 = 80,4 % (16/22).
  LE TEMOIN TIENT, ET C'EST CE QUI REND LE NEGATIF INTERPRETABLE : les creations `ti=42` d'ARMES
  ORDINAIRES, passees a la MEME regle, ne font que 122/950 = **12,8 %** (seuil <= 20 %). La piste
  discrimine donc d'un facteur six — elle n'est pas du bruit — mais un quart des vies reste
  inexplique, et publier ces 48 vies-la ferait dessiner comme drapeau des objets dont rien ne dit
  qu'ils le sont.
  LE DIAGNOSTIC ECARTE LA PISTE FACILE : sur les 48 non expliquees, **3** seulement naissent la ou
  l'objet reposait deja (re-creation d'un drapeau au sol). Le residu n'est donc pas un artefact de
  re-creation sur place ; sa cause reste ouverte.
  DEUX DEFAUTS D'INSTRUMENT ONT ETE CORRIGES AVANT DE CONCLURE, aucun ne touche un seuil :
  (a) la reference « porteur » ne retenait que la DERNIERE frame d'un portage, ce qui excluait par
  construction le LACHER VOLONTAIRE — le phenomene meme que le lot existe pour dater (48,2 % ->
  71,8 % sur `64e8adfa`) ; (b) le jeu de socles reutilisait le filtre de PRODUCTION, qui ecarte le
  socle neutre a bon droit pour publier mais pas pour mesurer, quand le plan et son acquis parlent
  du ROLE `flag_spawn` (+1 vie). Le temoin a suivi les deux corrections (12,5 % -> 15,1 % sur
  `64e8adfa`, 12,8 % au total) : elles ne flattent pas le drapeau.
  CE QUI A ETE MESURE MALGRE TOUT, sur la chaine de publication ecrite puis RETIREE — les chiffres
  sont au CR et valent pour la reprise : 108/39/44 vies libres publiables, **2 portages
  `carried_open` fermes par une vie libre** (film `530820e5`, les 2 qu'il portait), 31/17/4
  `dropped` repositionnes sur la piste libre.
  ANCRAGE TENU : 78/30/29 portages publies, identiques a l'item 1.3 et aux artefacts en cache ;
  socles d'arme et couverture `groundWeapons` inchanges (retenues 352/239/359).
  TEMOINS NON RE-CUITS `[~]` : plus rien de nouveau a publier (schema 14 inchange), et les deux
  ancrages qu'ils devaient servir sont verifies contre les artefacts EXISTANTS par la mesure
  ci-dessus.
- 2026-08-18 — ARBITRAGE APPLIQUE : phase 2 scindee, `2a` `[!]` (piste non publiee), `2b` `[x]`
  (les deux corrections livrees). LA RAISON EN UNE PHRASE : le controle 3 refuse la population
  ENTIERE des vies libres, mais sa branche « porteur » est exactement celle que les corrections
  consomment — elles ne touchent aucune des 48 vies inexpliquees.
  LA RESTRICTION EST DANS LE CODE, PAS DANS UN COMMENTAIRE : `flagFreeNearSpawn` ecarte d'abord
  toute naissance a moins de 1,5 m d'un socle, puis la distance au porteur fait le reste. Le
  garde-rail porte SES DEUX TEMOINS NEGATIFS (`flag_objects_test.go`) : une vie nee loin du
  porteur ne ferme rien, et une vie nee AU SOCLE ne ferme rien MEME a 0,5 m du porteur — ce
  second temoin est la condition de l'arbitrage, pas un ornement.
  SCHEMA 15 : aucun champ neuf, mais le CONTENU de `flagCarries` change (portages fermes,
  `dropped` deplaces) sans qu'aucune cle ne bouge — un artefact 14 se lit « a re-cuire ».
  Couverture : `objectLives` (le denominateur), `closedByObject`, `dropsRepositioned`.
  Contrat : 35 champs de document, INCHANGE ; trois proprietes de plus au sous-schema
  `FlagCarriesCoverage`. OpenAPI et `generated.ts` regeneres, golden `schema 15`, quatre temoins
  re-cuits sous le cache du depot principal.
- 2026-08-18 — PHASE 4 : LE LANCER N'EXISTE PAS DANS LA MESURE. L'utilisateur a propose une cause
  aux 48 residuelles du controle 3 : dans le jeu on ne LACHE pas seulement le drapeau a ses pieds,
  on peut aussi le LANCER quelques metres devant soi. La regle ELARGIE a ete ecrite et COMMITEE
  AVANT tout chiffre (`7f74a43fa`) — vie libre expliquee si elle nait a < 1,5 m d'un `flag_spawn`
  (branche socle, rayon INCHANGE) OU a <= R m de la position du porteur dans les 2 s de la fin de
  son portage, R balaye 1,5 / 3 / 5 / 8 / 10 m ; seuils du plan inchanges (>= 90 %, temoin <= 20 %),
  temoin passe a la MEME regle et au MEME R.
  LE BALAYAGE, cumul des trois films CTF (197 vies libres, 950 creations d'armes) :
  R = 1,5 m -> 162/197 = 82,2 %, temoin 136/950 = 14,3 % · R = 3 m -> 162/197 = 82,2 %,
  temoin 151/950 = 15,9 % · R = 5 m -> 162/197 = 82,2 %, temoin 167/950 = 17,6 % ·
  R = 8 m -> 163/197 = 82,7 %, temoin 211/950 = 22,2 % · R = 10 m -> 164/197 = 83,2 %,
  temoin 227/950 = 23,9 %. AUCUN R NE TIENT LES 90 %, et le temoin CREVE son plafond des 8 m.
  LE RAPPORT EST INVERSE, ET C'EST LA REFUTATION : de 1,5 a 10 m le drapeau gagne DEUX vies
  (+1,0 point), le temoin en gagne QUATRE-VINGT-ONZE (+9,6 points). Elargir le rayon profite dix
  fois plus aux armes ordinaires qu'au drapeau — R ne mesure donc pas une portee de lancer, il
  mesure la densite des joueurs sur la carte. C'est exactement ce que le temoin existe pour dire.
  LA DISTRIBUTION REFUTE UNE SECONDE FOIS, SANS PASSER PAR LES SEUILS. Sur les 35 residuelles a
  1,5 m, **26 (74 %) n'ont AUCUNE reference porteur** dans les 2 s : aucun portage ne couvre leur
  naissance ni ne vient de s'achever, il n'y a meme pas de lanceur candidat. Les 9 mesurables sont
  a mediane 20,6 m, p90 et max 43,1 m ; par tranche : `]1,5-3 m] : 0` · `]3-5 m] : 0` ·
  `]5-8 m] : 1` · `]8-10 m] : 1` · `]10-20 m] : 2` · `> 20 m : 5`. UN LANCER DE DRAPEAU PORTE
  QUELQUES METRES, ET LA TRANCHE DES QUELQUES METRES EST VIDE. Le residu n'est pas un lancer trop
  long : c'est, aux trois quarts, une naissance dont le film ne dit rien.
  CE QUE LA MESURE APPREND MALGRE SON NEGATIF, ET QUI N'EST PAS TRAITE ICI (hors perimetre, note
  pour le registre) : a rayon INCHANGE (1,5 m), passer la fenetre de 1 s a 2 s fait monter le
  controle 3 de 149/197 = 75,6 % a 162/197 = 82,2 % — treize vies — pour +1,5 point de temoin
  seulement (12,8 % -> 14,3 %). C'est un gain PROPRE, a l'oppose du rayon : ce qui manquait au
  lacher, c'est le DELAI, pas la distance. Cela ne se transporte PAS tel quel en production :
  `flagFreeDropWindowMS` sert a FERMER des portages et a REPOSITIONNER des lachers, exigences plus
  fortes qu'expliquer une naissance, et sa doc borne la seconde precisement pour ne jamais
  rattraper le portage PRECEDENT. Piste a mesurer, pas correctif a appliquer.
  CONSEQUENCES : `flagObjects` reste `[!]` (2a inchangee, aucun R retenu) ; les corrections de
  `flagCarries` livrees en 2b ne sont PAS etendues ; aucun schema, contrat, OpenAPI, golden ni
  temoin ne bouge. Le schema du document est a 16 depuis le lot des zones (`ti=13`) — la
  publication qu'aurait demandee un controle tenu aurait donc ete 17 / contrat 37, non 15 / 36.
