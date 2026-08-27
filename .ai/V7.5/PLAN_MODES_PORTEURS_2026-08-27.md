# PLAN — Modes a PORTEUR et modes restants (Oddball porteur, Assaut, VIP, Extraction, Land Grab)

> Ecrit le 2026-08-27 par la session superviseur, sur commande utilisateur (verbatim resume :
> « drapeau et bases c'est nickel mais Oddball on n'arrive pas a suivre le joueur-porteur du
> crane [...] VIP pas finalise [...] Assaut on ne l'a pas [...] Land Grab une ou deux parties
> [...] Extraction je ne connais pas, si t'as rien laisse tomber [...] puis les effets visuels
> (couronne VIP, porteur de bombe) et les effets sonores »).
>
> Contrat d'execution : skill `plan-execution` — il fait foi, ce plan ne le paraphrase pas.
> Branche du lot O : `wt/oddball-porteur` (worktree `../LevelUp-wt-oddball-porteur`), base
> `5ab8a448d`. Regles NON NEGOCIABLES heritees du chantier obj-etat (elles s'appliquent a
> CHAQUE lot de ce plan) : tout decodage de film passe par l'executeur borne
> `internal/filmproc` (le garde-rail archlint `no_unbounded_film_loop_test` le force) ;
> protocole COMMITE avant toute mesure ; seuils jamais abaisses apres coup ; temoins negatifs
> obligatoires ; arret propre au seuil rate. Document d'entree : le handoff
> `.ai/V7.5/HANDOFF_ODDBALL_PORTAGE_2026-08-27.md` (acquis / refutations / pistes P1-P4).

---

## 0. ETAT DES LIEUX — VERIFIE SUR PIECES LE 2026-08-27 (sondes API du jour comprises)

| mode | matchs registre | films en cache | oracle API par joueur | verdict d'entree |
|---|---|---|---|---|
| Oddball | 26 | **7** (`60ae07c4`, `92f18088`, `24dbb67d`, `43716616`, `51ebbc0f` pont casse, `c88ec007`, `d9781168`) | OUI — `match_objective_stats_latest` : `time_as_skull_carrier_seconds`, `skull_grabs`, `skull_scoring_ticks`, `skull_carriers_killed`, `longest_time_as_skull_carrier_seconds` | LOT O — la demande n° 1 |
| Assaut (Neutral/One Bomb, Husky Raid) | 9 | **9** (tous) | **NON — verifie ce jour sur les 9 payloads bruts : aucun bloc `AssaultStats`, seulement CoreStats+PvpStats** | LOT A — sans oracle API, gates internes au film seulement |
| VIP | 3 | 3 | OUI — `VipStats` riche (verifie ce jour) : `TimeAsVip` au 1/10 s, `TimesSelectedAsVip`, `LongestTimeAsVip`, `VipKills`, `KillsAsVip`, `VipAssists` | LOT V — voie NEUVE par contraintes (le film ne porte pas le bit VIP, §2.10 obj-etat : ne PAS re-mesurer le film) |
| Extraction | 2 | 2 | OUI — `ExtractionStats` (initiations/conversions/extractions, compteurs) | LOT E — sonde bornee (« si t'as rien laisse tomber ») |
| Land Grab | 1 (`19f1028b`, 2024-02) | **0 — film EXPIRE, sonde API du jour : 404/410** | OUI — payload verifie ce jour : bloc `ZonesStats` (colonnes Stronghold*), deja ingere par l'extracteur | LOT L — cablage statique seul (sert les matchs FUTURS) |

**Corrections de faits importantes (etablies ce jour, sur pieces) :**

1. La note du journal du 27/08 « 19 films Oddball recuperables par `cmd/fetch_film_chunks` »
   est FAUSSE : cet outil ne telecharge que les chunks des manifests DEJA en cache, et les 19
   vieux matchs (2021-2023) n'ont AUCUN manifest. Sonde API du jour sur `72f735b0` (2023-09) :
   film EXPIRE (404/410). **Le corpus Oddball est 7 films, point.** Le seul levier
   d'agrandissement est que l'utilisateur REJOUE a ces modes (le sync capture les films
   recents).
2. L'API n'a PAS de bloc Assaut : le porteur de bombe n'aura jamais de juge de paix API.
3. Land Grab utilise le bloc `ZonesStats` (comme Strongholds) : l'oracle des prises existe
   deja en base pour le match connu.
4. L'outil de balayage du statborg `cmd/tmp_statnames` n'existe plus (constat du 27/08) : le
   lot O le reecrit en CLI durable.

## 1. DECISIONS PRODUIT — TRANCHEES ICI (reco superviseur, l'utilisateur contre-valide au CR)

1. **Ordre des lots : O (Oddball porteur) → A (Assaut) → V (VIP) → E (Extraction) → L (Land
   Grab) → R (rendu + sons des modes livres).** Un seul executeur a la fois (pas de builds Go
   concurrents sur ce poste).
2. **Assaut sans porteur reste livrable** : vies libres de la bombe (meme patron que le crane
   libre publie) + etat des sites d'amorcage s'il se mesure. Le PORTEUR de bombe ne se publie
   que si la mecanique gagnee au lot O tient sur Assaut avec ses temoins internes (pas
   d'oracle API : les temoins font le gate).
3. **VIP par CONTRAINTES** (API + kill feed), jamais par le film : la voie film est `[!]`
   definitif (§2.10 obj-etat) et ne se rejoue pas.
4. **Extraction = sonde bornee** : 2 films ; si ni canal ni oracle exploitable au terme de la
   sonde, `[!]` avec condition de reprise et ON S'ARRETE (mot de l'utilisateur).
5. **Land Grab = cablage de DONNEE seulement** (role `landgrab_zone` au catalogue + table du
   titre), degradation par absence — aucun etat vivant tant qu'aucun film n'existe.
6. **Rendu/sons en dernier**, par mode LIVRE, aux patrons existants (drapeau porte, jauge sur
   forme de Bastion, sons de zones/socles poses par la session sons). Couronne VIP et icone
   porteur de bombe : d'abord chercher l'icone du jeu (chaine d'extraction d'icones existante),
   sinon glyphe dessine — decision a la phase R, pas avant.

## 2. LOT O — ODDBALL, campagne D10 « fragmentation des longs portages »

> Executeur : agent dedie dans `../LevelUp-wt-oddball-porteur` (branche `wt/oddball-porteur`).
> Donnees et config lues dans le DEPOT PRINCIPAL via `LEVELUP_REPO_ROOT`
> (`c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration`) — aucune ecriture de base,
> lectures DuckDB par `OpenReadForQuery` uniquement. Commits prefixe `oddball-d10(...)`,
> jamais `git add -A`, jamais de push. `.ai/thought_log.md` et `.ai/V7.5/REGISTRE_REPORTS.md`
> ne sont PAS touches par cette branche (textes fournis au CR, le superviseur les consigne).
> Ce lot couvre les phases **O0 a O4 (diagnostic)**. O2 (gate) et O5 (publication) sont un
> second lot, ouvert par arbitrage superviseur sur les chiffres de O1/O3/O4.

**Objectif.** Nommer par les chiffres POURQUOI un long portage se fragmente (condition de
reprise ecrite au registre), puis — second lot, si l'arbitrage l'autorise — tenir le gate
historique INCHANGE : recouvrement `time_as_skull_carrier_seconds` >= 80 % par joueur ET
porteur principal identifie sur >= ceil(0,75 x n) des n films admis. Succes final = porteur
publie dans `objectiveObjects` ; echec = `[!]` documente, arret propre.

**Ce qui est ACQUIS et ne se re-prouve pas** (handoff §1) : oracle API officiel ; primitive de
proximite bimodale (seuil constate ~1,5 m) ; « mourir = lacher » 91,7 % ; sommeil refute ;
signal SPATIAL (temoin 12 m : 0-3,3 %) ; precondition de pont >= 50 % de slots nommes.
**Ce qui est REFUTE et ne se rejoue pas** (handoff §2) — en particulier la fenetre de la
queue (80,7 % = bruit demontre) et l'oracle du score personnel comme gate.

### O0 — Qualification du corpus + protocole COMMITE (aucune mesure avant le commit)

- [x] O0.1 Lire, dans l'ordre : le handoff Oddball ; les entrees 2026-08-26/27 de fin de
      `REGISTRE_REPORTS.md` ; les en-tetes des instruments `oddball_*_test.go`
      (`internal/analysis/replay/`) ; l'en-tete de la garde de mode qui protege le pont
      d'identite (19-22 Go hors CTF — la trouver via les references du handoff §4-P4).
      — Fait 2026-08-27. La garde de mode est `attachFlagCarries` (`build_objectives_live.go`,
      « HORS CTF, LE CALQUE S'ARRETE ICI ») ; le vrai correctif est le plafond
      `objectiveevents.maxDeathsPerSlot` (slotidentity_deaths.go), qui borne le pont
      d'identite SUR TOUT FILM — les instruments D4-D9 l'appellent deja sur Oddball.
- [x] O0.2 Qualifier le PONT des 7 films du §0 : taux de slots de bipede nommes par film
      (instrument existant de la campagne D6/D9, reutilise tel quel). Admis si >= 50 %.
      `51ebbc0f` exclu d'office (9/84, lecon commitee). Publier le tableau film -> taux ->
      admis/exclu DANS le protocole. Les 3 films jamais mesures (`60ae07c4` 2024-10,
      `92f18088`, `c88ec007`) peuvent echouer au DECODAGE (version de film plus ancienne) :
      un film indecodable est EXCLU avec sa raison, pas repare.
      — Fait 2026-08-27 (`TestOddballSommeilD8` un film/processus, log `D10_P0_pont.log`) :
      ADMIS `24dbb67d` 89,7 %, `43716616` 86,1 %, `92f18088` 90,6 %, `d9781168` 87,5 % ;
      EXCLUS `60ae07c4` et `c88ec007` (bornes de quantification absentes — Live Fire hors
      catalogue, pas une version de film), `51ebbc0f` d'office. Corpus admis = 4 films.
- [x] O0.3 Ecrire et COMMITTER `.ai/V7.5/replay2d/registre_film/D10_PROTOCOLE.md` : corpus
      admis fige, definitions (vie libre interieure, micro-lacher, causes a/b/c/d de O1),
      seuil d'ouverture de O2 (repris du §O1 ci-dessous, sans le modifier), temoins, et
      l'interdit de la fenetre de queue. Le hash du commit est cite au CR.
      — Fait 2026-08-27 : protocole + log de qualification + 3 oracles figes (releves de
      `match_objective_stats_latest`, 58 lignes) commites ensemble (hash au CR).

**Gate O0** : protocole commite AVANT toute mesure de O1 ; tableau de qualification publie ;
aucun seuil different de ceux ecrits ici.

### O1 — P1 : instrumenter les vies libres INTERIEURES aux longs portages API (diagnostic)

- [x] O1.1 Nouvel instrument `oddball_fragmentation_d10_test.go` (meme package que les D4-D9,
      meme garde d'environnement, jamais en CI), qui pour chaque film admis : (a) prend le
      plus gros porteur API du film (oracle `_latest`) ; (b) rejoue la reconstruction D9
      (premiere traversee, parametres FIGES de D9 — rien ne se regle) ; (c) pour chaque vie
      libre du crane, publie : duree, position de naissance, ramasseur reconstruit, porteur
      reconstruit precedent, distance du porteur precedent a la position de re-prise a
      l'instant de la traversee, meme-joueur (oui/non).
      — Fait 2026-08-27 : auto-controle contre `d9Reconstruit` inchangee OK sur 4/4 films.
- [x] O1.2 Ventiler les SECONDES MANQUANTES du plus gros porteur API de chaque film en 4
      causes nommees d'avance : (a) vie libre interieure re-attribuee a un TIERS alors que le
      porteur precedent est a <= 1,5 m de la re-prise ; (b) vie libre interieure re-ramassee
      par le MEME joueur mais comptee comme nouveau portage (fragmentation sans vol) ;
      (c) trou sans traversee (aucune attribution) ; (d) autre / hors intervalle. Log fige
      `D10_P1_ventilation.log` + tableau par film dans le CR.
      — Fait 2026-08-27 : (a)+(b) = 0,0 / 0,0 / 3,5 / 0,0 % — la cause dominante est (d)
      (84,3-100 %) : le manquant vit HORS des trous dont le porteur precedent est P.
- [x] O1.3 Distribution des durees des vies libres interieures (q50/q75/q90/max), publiee —
      c'est elle qui fixera le N de chainage de O2 (N = q90 arrondi a la seconde superieure,
      REGLE ECRITE ICI, avant mesure).
      — Fait 2026-08-27 : corpus n=50, q50 2,85 / q75 3,88 / q90 5,05 / max 6,50 s ;
      N = 6 s PUBLIE, non utilise : le seuil d'ouverture de O2 est tenu sur 0/4 films
      (2/4 exiges) — **P1 INFIRMEE**, le lot s'arrete en O3/O4 comme le plan l'ecrit.

**Seuil d'ouverture de O2 (ECRIT AVANT MESURE, ne se rebaisse pas)** : causes (a)+(b)
couvrent >= 50 % des secondes manquantes du plus gros porteur sur >= la moitie des films
admis. Sinon : P1 INFIRMEE, le lot s'arrete en O3/O4 et le CR le dit.

### O3 — P3 : elucider ce que DATENT les evenements `th=10` du crane (diagnostic borne)

- [x] O3.1 Sur les films admis : confronter chaque `th=10` de crane aux instants de
      transition de la chaine (naissances/morts de vies libres, debuts/fins de trous) ;
      publier l'histogramme des ecarts et le rapport compte-`th=10` / tics API par joueur.
      — Fait 2026-08-27 (`oddball_th10_d10_test.go`, log `D10_P3_th10.log`) : histogramme
      BIMODAL — pic a <= 100 ms (11-41 evenements par film) puis masse au-dela de 5 s ;
      rapports tics/th10 par joueur 1,43-4,75 (l'ordre du 3,1-3,7 du handoff).
- [x] O3.2 Verdict nomme : « les `th=10` datent X » avec accord chiffre, ou « non etabli ».
      S'ils datent ramassages et/ou lachers avec accord >= 80 %, ils deviennent l'ANCRAGE
      candidat de O2 (une seule remesure autorisee, protocole amende et commite avant).
      — Fait 2026-08-27 : **NON ETABLI sur 4/4 films** — accord naissances 13,8-26,6 %,
      silences 8,8-21,8 %, union 22,5-41,1 % (seuil 80). Le profil (pic exact minoritaire
      + masse loin des transitions, compte proportionnel au temps de portage) est celui
      d'un HEARTBEAT de possession, pas d'un marqueur de transition. Pas d'ancrage O2.

### O4 — P4 : inventaire du statborg Oddball (l'outil de balayage est a REECRIRE)

- [x] O4.1 Reecrire l'outil de balayage des emplacements du statborg en CLI durable
      `cmd/statnames-sweep` (remplace `cmd/tmp_statnames` disparu ; cite par
      `objectiveevents/named.go`) — lecture de film via `filmproc` OBLIGATOIRE, un film par
      processus, aucune ouverture de base en ecriture.
      — Fait 2026-08-27 : 3 fichiers (main/sweep/confront), parent filmproc + enfant sous
      sentinelle 2 Gio + mode -confront pur ; la reference de `named.go` est mise a jour.
- [x] O4.2 Balayer les films admis ; confronter les compteurs par joueur aux colonnes oracle
      des 7 matchs (58 lignes `match_objective_stats`) sur MOITIES DISJOINTES (moitie pour
      chercher, moitie pour verifier — repartition ecrite au protocole O0.3).
      — Fait 2026-08-27 : 4/4 films balayes (pics 0,01 Gio), 1792 valeurs finales, pont
      statborg 8/7/2/3 slots nommes ; confrontation sur les moities du protocole.
- [x] O4.3 Verdict nomme : « le statborg replique / ne replique pas un compteur de crane »,
      avec, si oui, l'emplacement et son accord chiffre sur la moitie de verification.
      (Le negatif du 18/08 portait sur 26 images-cles d'UN film — ce balayage-ci est la
      mesure complete que le handoff §4-P4 demande.)
      — Fait 2026-08-27 : **NE REPLIQUE PAS** — 0 candidat sur 56 emplacements x 5 colonnes
      x encodages (meilleurs accords 13,3-80,0 % pour 90 exige, 15 paires de recherche) ;
      controle positif interne : le pont nomme les slots par `comp 2 B` (morts), la lecture
      est saine. La piste P4 du handoff est SOLDEE (log `D10_P4_statborg.log`).

**Gate du lot (O0-O4)** : les 3 logs figes existent (`D10_P1_ventilation.log`, `D10_P3_th10.log`,
`D10_P4_statborg.log`), chaque phase a son verdict ecrit, le protocole n'a pas bouge apres
coup (l'historique git en temoigne), `go vet ./...` et la compilation des packages touches
passent dans le worktree. CR au superviseur avec les textes journal/registre PRETS.

### O2 + O5 (second lot, sur arbitrage superviseur — NE PAS COMMENCER dans ce lot)

O2 : reconstruction avec chainage meme-joueur (N fixe par O1.3) et/ou ancrage `th=10` (si O3
le donne), protocole commite avant mesure, gate historique inchange, temoins spatial et
joueur-aleatoire, une seule mesure. O5 si gate tenu : cle porteur dans `objectiveObjects`
(schema +1 — numero fixe A LA FUSION, triplet Go/contrat/web, chronique), remplacement PROPRE
des deux REFUS gardes par tests (rien pendant portage, pas de prolongation apres t1), rendu
crane-sur-porteur au patron du drapeau porte (`flagCarriesLayer`), re-cuisson des TEMOINS
seulement avec verification du CONTENU (recette au registre), i18n FR+EN des libelles neufs,
gates : `go test` packages touches + contracttest, `tsc -b`, vitest `match-replay`, lint.

## 3. LOTS SUIVANTS (chacun ouvre son plan detaille a son lancement ; perimetres fixes ici)

### LOT A — Assaut (9 films, AUCUN oracle API)
1. Identite de l'OBJET bombe par la recette `ti=42` du drapeau/crane (mot MPP, naissance aux
   `assault_bomb` du catalogue statique, temoin de selectivite = 0 autre candidat).
2. Vies libres de la bombe publiees au patron du crane libre (aucun porteur promis).
3. Etat des SITES d'amorcage : chercher le canal `ti=13` (rampe de jauge comme la capture de
   Bastion ; les roles statiques existent, 4/4 en `team_index = -1`). Oracle : le score de
   mode / les manches (armement -> explosion = increments dates), kill feed en corroboration.
4. Porteur de bombe : SEULEMENT si le lot O a livre une mecanique qui tient, temoins internes
   (spatial 12 m, joueur-aleatoire) aux memes seuils — sans oracle API, les temoins font foi.

### LOT V — VIP (3 films, VipStats riche, le film ne porte PAS le bit)
Reconstruction par CONTRAINTES, voie jamais mesuree : les morts du VIP sont dans le kill feed
deja decode ; `TimesSelectedAsVip` (2 par joueur sur le match sonde) et `TimeAsVip` au 1/10 s
par joueur contraignent l'affectation des periodes VIP par equipe. Protocole a ecrire (gate :
somme des durees reconstituees par joueur vs `TimeAsVip` API, accord >= 90 % ; temoin :
affectation aleatoire des periodes <= 20 %). D'abord ETABLIR la regle de rotation (mort du
VIP ? minuterie ?) sur les 3 films — si la regle ne se nomme pas, `[!]` avec condition.

### LOT E — Extraction (2 films) — SONDE BORNEE
Chercher : zones d'extraction ACTIVES (canal `ti=13` designateur, comme la colline KOTH) et
evenements initiation/conversion (statborg nomme ? `th=10` ?). Oracle : `ExtractionStats`
(compteurs par joueur) + score de mode. 2 films = pas de moities disjointes : tout verdict
positif se publie avec la reserve de corpus ecrite, sinon `[!]` condition >= 2 films de plus.

### LOT L — Land Grab (0 film, oracle ZonesStats deja en base)
Cablage de donnee SEUL : role `landgrab_zone` au decodeur `mapvar` + entree
`objective_roles.toml` (l'incoherence est deja nommee au §2.8 du plan obj-etat). Aucun etat
vivant. Sert le premier match FUTUR (l'utilisateur doit rejouer pour creer un film).

### LOT R — Rendu + sons des modes livres (dernier)
Crane-sur-porteur (si O5), bombe portee + jauge d'amorcage (si A), couronne VIP (si V),
marqueurs Extraction (si E). Icones : d'abord la chaine d'icones du jeu, sinon glyphe.
Sons : patrons `zoneSound`/`padSound` de la session sons (prise/amorcage/desamorcage,
couronne). Chaque effet suit la DONNEE livree de son mode — rien en avance.

## 4. DECOUVERTES (a consigner ici, ne pas traiter)

- (lot O, 2026-08-27) **Live Fire est hors du catalogue de bornes de quantification**
  (`map_quant_bounds.json`) : les 2 films Oddball de cette carte (`60ae07c4`, `c88ec007`)
  sont indecodables en coordonnees monde — c'est ce qui les a exclus du corpus D10. Et
  **Lattice est hors du catalogue d'objectifs** (`map_objectives.json`) : `92f18088` n'a
  aucun socle `oddball_spawn`, la classe « retour » y est indisponible. Completer ces deux
  catalogues agrandirait le corpus Oddball mesurable de 4 a 6 films. NON TRAITE ici.
- (lot O, 2026-08-27) **Asymetrie des deux ponts d'identite sur 2 films** : le pont
  STATBORG par instants de mort (`SlotIdentityByDeaths`) ne nomme que 2 slots sur
  `43716616` et 3 sur `d9781168`, alors que le pont BIPEDE des memes films est sain
  (86,1 / 87,5 %). Cause non instruite (prudence du pont ? morts communes < 3 ? marge x2 ?).
  NON TRAITE ici — mais toute mesure statborg par joueur sur ces films en herite.
- (lot O, 2026-08-27) Des joueurs a 0 tic API portent des evenements `th=10` de crane
  (`43716616` : 2 joueurs) — coherent avec une possession plus courte qu'un tic, et avec la
  nature heartbeat etablie en O3. Simple note de denominateur.

## 5. REPRISE DE SESSION

Lire ce plan, puis le protocole `D10_PROTOCOLE.md` s'il existe, puis les logs `D10_*.log`
figes. L'avancement fait foi par les statuts `[x]`/`[~]`/`[!]` de ce fichier, mis a jour et
commites avec chaque phase. Le superviseur tient journal et registre a la fusion — pas
l'executeur.
