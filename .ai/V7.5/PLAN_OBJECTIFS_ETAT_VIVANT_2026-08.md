# Plan — L'ETAT VIVANT DES OBJECTIFS DE MODE (rejeu 2D), inventaire par mode et suite

> Ecrit le 2026-08-25 par la phase D0 du LOT D de `PLAN_REPLAY2D_NOTION_2026-08-25.md` (point 11
> de l'encadre Notion « REPLAY 2D » : « Objectifs de mode : PLACEMENT statique acquis, ETAT vivant
> a faire »). Branche `wt/obj-etat`, base `c42624dd5`. Contrat d'execution : skill
> `plan-execution` — il fait foi, ce plan ne le paraphrase pas.
>
> **D0 EST UNE PHASE DE PLAN. Aucune implementation n'a ete faite, aucune mesure lancee.** Tout
> ce qui suit est etabli SUR PIECES par lecture du code et des archives de chantier ; chaque fait
> porte son `fichier:ligne` ou son document. Les seuils, eux, sont ECRITS AVANT MESURE et ne se
> rebaissent pas.

---

## 1. Ce qui est DEJA vivant (verifie sur pieces — ne pas re-mesurer, ne pas reimplementer)

| ce qui vit | ou, dans le code | schema |
|---|---|---|
| **Zones de Bastion (Strongholds)** : proprietaire par intervalle, `null` = personne, jauge de capture en direct | `replay/zone_states.go`, `zone_states_owner.go`, `zone_states_gauge.go` ; contrat `document_zones.go` | 16 puis 18 |
| **Colline de KOTH** : la colline DESIGNEE par le film (tag 5 de l'objet de mode), periodes fermees a la bascule, forme appariee par la grappe des positions, `Active = true` | `replay/zone_states_hill.go:41-135` ; role `hill` au catalogue et a la table du titre | 18 |
| **Drapeau de CTF** : la vie de chaque drapeau (`carried` / `carried_open` / `dropped` / `home`), porteur en xuid, lacher volontaire date par l'OBJET `ti=42` | `replay/flag_carries.go`, `flag_objects.go`, `document_objectives_live.go` | 14 puis 15 |
| **Score dans le temps** (l'oracle transverse de tout ce plan) : score de MODE par camp et score PERSONNEL par joueur, a la milliseconde, par manche | `objectiveevents/score.go:26-38`, `replay/document_score.go` | 12 |
| **Objectifs STATIQUES** de tous les modes servis (formes + marqueurs), croises A LA REQUETE | `service/replay_map_objectives.go`, `replay/map_objectives.go` | — |
| **Calques web** : geometrie statique / etat vivant separes | `objectivesLayer.ts` (393 l.), `zoneStatesLayer.ts` (377 l.), `flagCarriesLayer.ts` (279 l.) | — |

Etat du depot a la base `c42624dd5` : `replay.SchemaVersion = 19` (`replay/document.go:158`),
`wantReplayDocumentFields = 37` (`contracttest/replay_contract_test.go:331`),
`EXPECTED_REPLAY_SCHEMA_VERSION = 19` (`apps/web/src/features/match-replay/replaySchemaLogic.ts:33`).

**CORRECTION AU BRIEF DU LOT D** : le brief annonce « schema courant 18, prochain libre 19 ». Le
19 est PRIS depuis le 2026-08-25 par le lot 1 « lecture vide » (`inventory[].empty`, chronique en
tete de `document.go`). Le prochain numero libre est **20** — et la regle du depot est « un numero
par montee, DANS L'ORDRE DE FUSION » : le numero exact se fixe a la phase de publication, pas
maintenant. Arbitrage n°1 pour le superviseur (§6).

---

## 2. INVENTAIRE PAR MODE de l'etat vivant manquant

Colonnes : le CANAL mesurable (offline-pur obligatoire — aucune dependance Cheat Engine ni Ghidra
a l'extraction), l'ORACLE et son TEMOIN NEGATIF, le SEUIL, et le verdict de faisabilite.

### 2.1 Strongholds / Bastion — `[~]` COUVERT, deux residus ecrits

Le proprietaire vivant EST publie : canal `ti=13` **tag 4** (`0xFFFFFFFF` = personne, sinon index
d'equipe), appariement slot -> zone par les prises NOMMEES du statborg attribuees geometriquement
(`ZoneMethodCaptures`, mesure 93,1 % et 98,4 %, temoins 41-48 % et 51-57 %). L'etat CONTINU et
l'etat NEUTRE sortent tous deux du meme canal — **les evenements de prise ne suffisent pas et ne
sont pas ce qui porte l'etat** : ils ne servent qu'a APPARIER un slot a une zone, une fois par
match. C'est la reponse a la question du brief.

Deux residus, ECRITS et non traites par ce plan :

- **`contested` est REFUTE** (pas reporte) : les slots de rampe ne portent PAS de tag 4, les deux
  familles sont disjointes — la question est VIDE sur ce corpus (`document_zones.go`, § « ce que
  la mesure a refuse de publier »).
- **Les zones a `ownerUnpaired`** (jauge appariee, aucun canal de propriete elu) ne sont pas
  publiees, et c'est deliberement : `coverage.zones.ownerUnpaired` les compte pour que le silence
  ne soit pas muet.

La DONNEE de Bastion est donc complete ; ce qui reste est du RENDU, et c'est l'item D-R
(phase D6) : la progression de capture doit se lire SUR la forme de la base au lieu d'un arc
externe, et l'appartenance y est jugee trop discrete.

### 2.2 KOTH — `[ ]` LE PROPRIETAIRE DE LA COLLINE MANQUE (le trou le moins cher du lot)

Ce qui manque, exactement : `hillStatesOf` (`zone_states_hill.go:341-371`) construit ses
intervalles avec `Active: true` et **`Owner` jamais renseigne** — le rejeu sait QUELLE colline est
active, jamais QUI la tient. C'est ecrit noir sur blanc en tete du fichier : « CE QUE CE VOLET NE
PUBLIE PAS : le PROPRIETAIRE. Le tag 4 du slot voisin est un canal de propriete au sens de la
phase 2a, mais il n'a pas ete confronte au roster sur les KOTH — on ne publie pas ce qu'on n'a pas
mesure. »

| | |
|---|---|
| **canal** | `ti=13` tag 4 du slot voisin du designateur — DEJA lu par le code (`ser.owner[d.slot+1]`, condition d'election du designateur, `zone_states_hill.go:70`). Offline-pur, aucun balayage neuf. |
| **oracle** | Le SCORE DE MODE par camp (`scoreTimeline.teams[]`, schema 12) : en KOTH le score de mode EST le temps de colline (« l'API compte des secondes de colline », chronique 33->34 du contrat). Le camp dont le score MONTE pendant un intervalle est celui qui tient la colline. Oracle a la milliseconde, deja en production, aucune base ouverte. |
| **corroboration** | Les evenements `th=10` de prise de colline, acteur nomme par le pont bipede (`SlotIdentityResolved`) puis equipe par le roster. Approximatif (5-20 s) : publie comme controle, jamais comme gate. |
| **temoins negatifs** | (a) PERMUTATION : la serie de tag 4 d'un slot appliquee a une autre colline ; (b) DECALAGE : le meme accord mesure avec les intervalles decales de +20 s. Ce sont les deux temoins de la phase 2a, reutilises tels quels. |
| **seuil (ECRIT AVANT MESURE)** | accord >= **90 %** des confrontations possibles (intervalle portant >= 2 emissions non neutres de tag 4 ET >= 1 increment de score de mode), sur >= **3 des 4** films KOTH du corpus ; temoin (a) <= **60 %** ET temoin (b) <= **60 %**. Denominateurs publies, jamais un taux nu. |
| **verdict** | **FAISABLE** — canal, oracle et temoins existent tous ; rien a decoder de neuf. |

### 2.3 Total Control — `[ ]` AUCUN ETAT VIVANT (le plus gros trou du lot)

Etat actuel : le role `totalcontrol_zone` est SERVI statiquement depuis le 2026-08-25
(`objective_roles.toml`, entree `match = ["Total Control"]`) — le rejeu dessine donc aujourd'hui
**13 a 18 formes neutres** sur une partie de Total Control, alors que le mode n'en ACTIVE QUE 3
par manche. Le role est explicitement HORS de `heldZoneRoles`
(`replaybuild/zones.go:75-78`) : aucun etat n'est publie, et le commentaire de la table dit
pourquoi et a quelle condition il pourra y entrer (« sans designateur, un etat de zone publierait
14 a 18 zones tenues la ou il y en a 3 »).

| | |
|---|---|
| **canal (etat)** | `ti=13` tag 4, le MEME que Bastion. `ObjectiveTypeOf` classe deja Total Control en `zone` (`objectiveevents/extract.go:136`) : la table d'emplacements nommes `zone_captures` / `zone_secures` s'applique donc, et avec elle la methode `ZoneMethodCaptures`. |
| **canal (les 3 ACTIVES)** | deux hypotheses a departager, dans cet ordre : (H1) un DESIGNATEUR tag 5 comme en KOTH, mais qui designe trois zones ou change par manche ; (H2) pas de designateur — les 3 actives se DEDUISENT de l'appariement lui-meme (seules les zones qui recoivent des prises nommees existent dans le match). |
| **oracle** | (a) les prises NOMMEES `zone_captures` du statborg, a la milliseconde, attribuees geometriquement a une forme du catalogue ; (b) les MANCHES, deja publiees (`scoreTimeline.teams[].rounds`) — c'est ce qui borne « les 3 actives DE CETTE manche ». |
| **temoin negatif** | formes DECALEES de 12 m en x et y (la convention de `cmd/zone-attribution`, `defaultWitnessOffsetM`) : le taux d'attribution doit s'effondrer. |
| **seuils (ECRITS AVANT MESURE)** | (1) attribution : >= **80 %** des prises nommees tombent dans UNE forme du catalogue, temoin decale <= **20 %** ; (2) cardinalite : l'ensemble des zones appariees par manche vaut **exactement 3** sur >= **2** films ; (3) proprietaire : accord tag 4 / equipe du capteur >= **90 %** (le chiffre tenu par Bastion : 48/48 et 51/56). Les trois doivent passer pour que `totalcontrol_zone` rejoigne `heldZoneRoles`. |
| **verdict** | **FAISABLE SOUS RESERVE DE CORPUS** — tout l'outillage existe ; il faut des films de Total Control (recensement, phase D1). |

### 2.4 Oddball — `[ ]` LE CRANE : la voie du MARQUEUR est morte, une AUTRE voie s'est ouverte

Trois negatifs independants sont acquis et ne se rejouent pas (registre des reports, ligne « Le
CRANE d'Oddball n'est ni lu ni publie ») : le marqueur de portage du drapeau est TOTALEMENT absent
du film Oddball (0 porteur sur 26 images-cles) ; le statborg ne replique AUCUN compteur de crane ;
la signature structurelle seule laisse 195 motifs candidats. **Ce plan ne repasse par aucun des
trois.**

Ce qui a CHANGE depuis ce report (2026-08-18), et qui le rouvre :

1. **La recette d'identite d'un objet d'objectif existe et a fait ses preuves.** Le drapeau a ete
   identifie comme le mot de 32 bits `0x2A392328` du bloc `object-multiplayer-properties` d'un
   record de CREATION `ti=42`, par trois lectures et un temoin de selectivite
   (`replay_labels.toml`, § OBJETS D'OBJECTIF ; instrument `attachement_phase0_drapeau_test.go`).
   La table `[[objective_objects]]` est une liste FERMEE dont le commentaire dit que le crane n'y
   est pas « faute de canal ET d'oracle » — c'est cette phrase que ce plan attaque.
2. **L'oracle qui manquait est en production.** La condition de reprise ecrite au registre nommait
   « (a) le SCORE PAR SECONDE DE PORTAGE ». Le score PERSONNEL par joueur, a la milliseconde, est
   publie depuis le schema 12 (`scoreTimeline.players[]`).

| | |
|---|---|
| **canal (identite)** | le mot MPP de 32 bits des creations `ti=42` ECARTEES du catalogue d'armes — deja balaye par la chaine des socles, zero lecture de film ajoutee. |
| **canal (portage)** | l'ABSENCE de replication : un objet porte cesse d'emettre sa position (etabli par `flag_objects.go`, § « Le principe »). Un trou entre deux vies libres du crane = un portage. |
| **oracle** | le SCORE PERSONNEL : en Oddball il monte a ~1 Hz pendant qu'on tient le crane. Le joueur dont le score personnel s'incremente pendant un trou EST le porteur. |
| **corroboration** | les 87 evenements `th=10` de crane du film `24dbb67d`, 87 acteurs, **87/87 nommes par le pont bipede** (mesure du 2026-08-18, saine). |
| **temoins negatifs** | (a) SELECTIVITE : aucun AUTRE mot ecarte ne reunit « naitre a <= 3 m d'un `oddball_spawn` » ET « coincider a <= 1 s d'un evenement `th=10` de crane » — c'est le temoin exact qui a etabli le drapeau ; (b) PORTEUR : un joueur tire au hasard hors trou <= 5 %. |
| **seuils (ECRITS AVANT MESURE)** | (1) identite : **UN SEUL** mot candidat, LE MEME sur >= **2** films Oddball, temoin (a) = **0** autre candidat ; (2) portage : >= **90 %** des trous ont EXACTEMENT UN joueur dont le score personnel s'incremente sur toute leur duree, temoin (b) <= **5 %**. |
| **verdict** | **FAISABLE SOUS RESERVE DE CORPUS** (>= 2 films Oddball exiges ; un seul est cite au corpus historique, `24dbb67d`). Si l'identite ne sort pas : `[!]` MESURE, et la ligne du registre est mise a jour, pas contournee. |

### 2.5 Extraction — `[!]` NI CANAL NI ORACLE, ET PROBABLEMENT NI CORPUS

- Le role `extraction_zone` est servi STATIQUEMENT (`objective_roles.toml`, `neutral = true`) et
  explicitement EXCLU de `heldZoneRoles` (`replaybuild/zones.go`, § « Seules les zones TENUES »).
- **`ObjectiveTypeOf` ne connait pas Extraction** (`objectiveevents/extract.go:120-145` : ni
  `extraction` ni aucun synonyme). Aucun evenement nomme, aucune table d'emplacements, donc
  **aucun oracle** — et donc aucun appariement slot -> zone possible par la methode de Bastion.
- Le mode est marginal en base (releve `.ai/BACKLOG.md` : 2 matchs Extraction).

**Condition de reprise** : (1) >= 2 films d'Extraction en cache ; ALORS (2) chercher un canal
`ti=13` (slots, tags emis) sur ces films, et un oracle dans le score de mode. Sans (1), rien.

### 2.6 Stockpile — `[!]` CORPUS (report deja acte, maintenu)

Roles `stockpile_socket` / `stockpile_navpoint` servis statiquement. `ObjectiveTypeOf` ne connait
pas Stockpile non plus : aucun evenement nomme. Le report « Stockpile : aucun film exploitable
(404 Theater) » date du 2026-08-17 (`PLAN_OBJECTIFS_VIVANTS_2E_LECTURE.md`, corpus). Le noyau est
un objet PORTE : si un film apparait, la recette d'identite `ti=42` du §2.4 s'y applique telle
quelle, avec `stockpile_socket` pour socle. **Condition de reprise : >= 1 film de Stockpile.**

### 2.7 Assaut (bombe) — `[!]` NI ORACLE NI CORPUS ETABLI

Role `assault_bomb` servi statiquement (4/4 objets en `team_index = -1`). `ObjectiveTypeOf` ne le
connait pas : aucun evenement nomme. La bombe est un objet porte : meme recette que §2.4.
**Condition de reprise : >= 2 films d'Assaut au recensement D1**, sinon le mode ne s'ouvre pas.

### 2.8 Land Grab — `[!]` UNE INCOHERENCE A SIGNALER, PAS A CORRIGER ICI

`ObjectiveTypeOf` classe Land Grab en `zone` (`extract.go:135`) — il a donc des evenements nommes
— **mais aucune entree de `objective_roles.toml` ne le sert**, et ses hashs `landgrab_zone` sont
presents dans les fichiers de carte sans role associe (dit en toutes lettres par
`service/replay_map_objectives.go`, commentaire du cas nominal). Ni formes servies, ni etat.
**Condition de reprise : decision produit** (le mode vaut-il un calque ?), puis ajout du role au
decodeur `mapvar` + entree de table. Hors perimetre de ce lot.

### 2.9 Firefight (PvE) — `[!]` ROLE AU CATALOGUE, SERVI PAR AUCUN MODE, ET C'EST VOULU

`firefight_objective` (5 volumes avec forme par carte, mesure du 2026-08-20) entre au catalogue
mais aucune entree ne le sert : « la decision de le servir demande de savoir quel libelle de
manche Firefight le porte, ce qu'aucune mesure ne dit encore — le servir au cas ou afficherait
cinq zones sur une carte PvE sans savoir laquelle est active » (`objective_roles.toml`, en-tete).
**Condition de reprise : etablir le libelle de manche qui active un objectif Firefight.**

### 2.10 VIP — `[!]` DEFINITIF SUR CE QUE LE FILM PORTE

**Ce que dit le depot du substitut `managed-objective-object-reference`** — c'est le composant
**i3 de l'archetype `ti=11`** (`filmdec/testdata/ecs_table.tsv:271`), decrit comme « LA REFERENCE
vers l'objet physique (le drapeau, le crane, le noyau) », priorite HAUTE, statut **`non_porte`,
deser inconnu, 0 appelant**. La couverture de dispatch de `ti=11` est **0 / 34** composants.

Et il est INATTEIGNABLE par les deux voies du film, chacune refutee PAR SON TEMOIN (lot R4,
`PLAN_R4_OBJECTIFS_VIVANTS_TI11.md`, ligne 205 du registre des reports) :

- voie DELTA : `matchWorldObjectRecord` ne reconnait pas ces records — 4 680 contre 6 421 pour un
  fantome de meme taille et de meme voisinage (0,73x ; 0,37x sur le second film), 45,9 % et 36,4 %
  d'index hors grammaire ;
- voie IMAGE-CLE : bloquee par la grammaire du CORPS d'un record d'image-cle, resolue nulle part —
  et la lignee R7 (a..e) a mesure que la bit-exactitude plafonne a 0,51-0,85 % sur le bipede, avant
  que **l'utilisateur n'ARRETE la RE de l'image-cle** (borne d'arret R7-e, respectee depuis).

Cote binaire, le releve de l'utilisateur (Notion) et le lot Ghidra du 24/08 concordent : le bit VIP
vit dans un octet d'attributs de SCRIPT (poseur, effaceur, `Player:IsVIP()`), **sans aucun
serialiseur**, et `ApplyVIPPlayerFX` n'existe pas dans l'executable
(`PLAN_RE_LETTRE_HUD_GHIDRA.md:29-31`) — meme cause que la lettre A/B/C, dont le lot Ghidra a
etabli qu'elle sort du SCRIPT (`Navpoint_SetDisplayText`) et non du moteur. Enfin, aucun role VIP
n'existe au catalogue statique (597 objectifs, 7 roles a l'origine).

**Statut : `[!]` — le plan ne promet PAS le VIP.** Condition de reprise, une seule et elle est
chere : la grammaire du CORPS d'un record d'image-cle (le meme deblocage qui servirait `ti=42`),
aujourd'hui hors v7.5 par decision utilisateur. **Ce qu'il faudrait MESURER si elle tombait**, dans
l'ordre : `i5 type`, `i12`/`i13 progress`, `i14 state`, puis SEULEMENT `i3 object-reference` —
c'est l'ordre ecrit par R4, et il n'a pas bouge.

---

## 3. CONTRAINTES D'ARCHITECTURE (elles gouvernent le decoupage du §4)

1. **UN SEUL bump de schema pour tout le lot.** Consequence directe : **aucune phase de mesure ne
   touche la production**, et TOUTES les publications se font dans UNE phase terminale (D5). Le
   numero se fixe a ce moment-la (>= 20), jamais avant : la regle du depot est « un numero par
   montee, dans l'ordre de FUSION », et deux sessions concurrentes en ont deja echange
   (chroniques v15/v16 et v17/v18).
2. **La montee de schema est un TRIPLET, pas un entier.** `replay.SchemaVersion`,
   `contracttest.wantReplayDocumentFields` (+ sa ligne de chronique), et
   **`EXPECTED_REPLAY_SCHEMA_VERSION` cote web** (`replaySchemaLogic.ts:33`, ajoutee le
   2026-08-25). Oublier le troisieme ferait lire « stale » a tous les artefacts neufs — mais un
   garde-rail de PARITE le rattrape : `replaySchemaLogic.guard.test.ts` LIT le fichier Go et
   echoue si les deux divergent. Le gate web de D5 le joue ; ne pas s'en etonner, le corriger.
3. **Re-cuisson des TEMOINS seulement** — jamais de cuisson de masse. Un film par PROCESSUS
   (lecon « balayage corpus = bombe RAM »), et **un seul decodage `filmdec` par process** (le
   balayage `ti=13` installe un hook global).
4. **Title-agnostic strict.** Aucune comparaison `slug == "..."` (ratchet
   `no_slug_comparison_test.go`). Ce qui est propre au titre va en DONNEE :
   `config/titles/{slug}/mappings/objective_roles.toml` (quels roles pour quel mode) et
   `replay_labels.toml` (`[[objective_objects]]` : identite et nom d'un objet d'objectif, EN+FR).
   La degradation se fait **par ABSENCE de donnee** — un titre sans table n'a pas de calque, et
   `coverage` dit lequel des deux silences. C'est deja le patron de `flagCarries` et `zoneStates`.
5. **Aucune base en ecriture, aucun serveur de dev.** Les lectures de `match_registry` se font en
   LECTURE SEULE (`duckdb.OpenReadForQuery`, correct meme si le serveur tient la base — ADR
   0013/0016). Un `server.exe` tourne sur ce poste : ne jamais ouvrir la base en RW.
6. **Le RENDU ne demarre qu'APRES la fusion du lot A.** Les deux lots ecrivent dans
   `apps/web/src/features/match-replay/`. **D6 (item D-R) et D7** sont donc apres A, et c'est
   l'ordre du §4. D6 ne depend QUE de A (il consomme de la donnee deja publiee au schema 18) ;
   D7 depend en plus de D5.
7. **Zero code mort, zero flag OFF.** Ce qu'un calque vivant remplace (par ex. les pulses
   substituts d'une famille) se RETIRE quand la donnee arrive, jamais « au cas ou ».

---

## 4. DECOUPAGE EN PHASES

> Regle d'ordre : **une phase a la fois**. Une phase est CLOSE quand tous ses items sont statues
> (`[x]` fait / `[~]` couvert ailleurs avec reference / `[!]` non traite avec justification
> ECRITE), son gate joue DANS la session avec ses codes de retour au journal du plan, et le plan
> mis a jour et commite avec le lot. Aucune case vide a la cloture. Aucun fix hors perimetre : les
> decouvertes vont au §7, elles ne se traitent pas.
>
> Commits : prefixe `obj-etat(D<n>):`, sur `wt/obj-etat` uniquement, jamais `git add -A`, jamais de
> push. `.ai/thought_log.md` et `.ai/V7.5/REGISTRE_REPORTS.md` ne sont PAS touches par cette
> branche : leurs TEXTES sont fournis au CR de lot, le superviseur les consigne a la fusion.

### D1 — RECENSEMENT DU CORPUS, et verdict de faisabilite par mode (STOP au verdict)

Sans lui, chaque phase suivante decouvrirait son corpus en cours de route. 951 films sont en cache
(`data/cache/film_chunks`), mais leur MODE n'est pas dans le manifeste : il se lit dans
`match_registry.game_variant_name`.

> **D1 EST FAIT (2026-08-26, second passage).** Le blocage d'outillage ci-dessous est LEVE :
> `cmd/zone-attribution` porte desormais un drapeau `-census` qui recense le corpus PAR MODE, en
> lecture seule (`OpenReadForQuery`), sans MCP ni binaire neuf. Le mode vient du `pair_name` du
> registre, normalise par la MEME fonction que le service (`analysis.NormalizeModeLabel`) —
> aucune liste de modes n'est ecrite dans le code, le recensement rend ce que le registre
> contient.
>
>	LEVELUP_REPO_ROOT=<repo> go run ./cmd/zone-attribution -census -cache <repo>/data/cache
>
> **1 940 matchs, 45 modes distincts, 87 sans `pair_name` exploitable.** Sortie brute figee :
> `.ai/V7.5/replay2d/registre_film/D1_recensement_modes.log`. Ce qui decide des phases D3 et D4 :
>
> | mode | matchs | **films en cache** | artefacts (schemas) | verdict |
> |---|---|---|---|---|
> | **Total Control** | 110 | **4** (+3 en « Fiesta Total Control ») | 0 | **D3 OUVRABLE** — le seuil du §2.3 est >= 2 films |
> | **Oddball** | 26 | **7** | 1 (schema 20) | **D4 OUVRABLE** — le seuil du §2.4 est >= 2 films |
> | **Stockpile** | 39 | **2** | 0 | condition de reprise du §2.6 (>= 1 film) ATTEINTE |
> | **Extraction** | 2 | **2** | 0 | condition (1) du §2.5 ATTEINTE ; la condition (2) — trouver un canal ET un oracle — reste entiere |
> | Assaut (`Neutral Bomb` / `One Bomb` / `Neutral Bomb Squad`) | 8 | **8** | 0 | condition du §2.7 (>= 2 films) ATTEINTE |
> | KOTH | 52 | 47 | 4 (schema 20) | corpus large, mais le canal proprietaire est bloque sur la DECISION PRODUIT (D2/D2-bis/D2-ter) |
> | Strongholds | 75 | 50 | 3 (schema 20) | deja couvert |
>
> **CE QUE CE RECENSEMENT CORRIGE, ET C'EST L'ESSENTIEL.** Le plan tenait Oddball pour limite a
> UN film (`24dbb67d`) et Stockpile pour `[!]` CORPUS DEFINITIF (« aucun film exploitable, 404
> Theater », report du 2026-08-17). Les deux sont FAUX aujourd'hui : 7 films d'Oddball et 2 de
> Stockpile sont en cache. Le report Stockpile est a AMENDER au registre — il decrivait l'etat du
> cache a sa date, pas une impossibilite.
>
> **VIP : 3 matchs, 3 FILMS EN CACHE — et cela ne change RIEN.** Le §2.10 ne bute pas sur le
> corpus mais sur le film lui-meme (le bit vit dans un octet d'attributs de script, sans aucun
> serialiseur). Avoir les films ne cree pas la donnee : VIP reste `[!]` definitif.
>
> Le detail du premier passage, et pourquoi il ne suffisait pas, est conserve ci-dessous.
>
> **D1 EST BLOQUE SUR SON OUTILLAGE (2026-08-26, PREMIER PASSAGE — LEVE DEPUIS).** L'item D1.1
> prescrit un `COPY (...) TO ...` en lecture seule, sur le modele de `oracle_export.sql`. Cette
> recette suppose un client SQL : le serveur MCP `duckdb` documente par `CLAUDE.md`, ou le binaire
> `duckdb`. **Aucun des deux n'est disponible dans cette session** (`duckdb` absent du PATH, MCP
> non expose), et un `server.exe` tient la base — toute ouverture doit passer par
> `OpenReadForQuery`, donc par du code Go.
>
> CE QUI A ETE OBTENU MALGRE TOUT, avec le CLI EXISTANT et teste (`cmd/zone-attribution
> -select-only`, `OpenReadForQuery`, aucune ecriture) :
>
>	1 940 matchs au registre · 208 en mode a ZONES · 151 sans film en cache ·
>	3 sans bornes de carte · 6 sans formes au catalogue · 48 MESURABLES
>
> **ET CE CHIFFRE NE REPOND PAS A LA QUESTION DE D1**, il faut le dire : ce CLI compte les zones
> du role `strongholds_zone` SEUL. Les 48 films eligibles rendent tous « 3 zone(s) » — la
> signature exacte de Bastion (3 zones par carte, sans exception mesuree le 2026-08-20). Une carte
> de Total Control en declare 13 a 18 : aucun de ces 48 n'en est un, mais l'outil ne saurait pas
> le DIRE — un match de Total Control tomberait dans les « 6 sans formes au catalogue » sans etre
> nomme. Et il ne voit ni Oddball, ni Extraction, ni Stockpile, qui ne sont pas de la famille
> `zone`.
>
> **ARBITRAGE DEMANDE (§6, n°7)** : le recensement par mode exige un chemin de lecture qui
> n'existe pas. Trois options, par cout croissant — (a) exposer le MCP `duckdb` a la session ;
> (b) ajouter un drapeau `-mode` a `cmd/zone-attribution` (mais c'est un outil de MESURE, pas de
> recensement) ; (c) un petit `cmd` de recensement dedie. Creer un binaire de production pour un
> comptage ponctuel est une decision, pas une initiative d'executeur : elle revient au
> superviseur.

- [x] D1.1 Export LECTURE SEULE d'un recensement : `match_id`, `game_variant_name`, `map_id`,
      `map_name`, `start_time` pour tout `match_registry`. Meme recette que l'oracle du lot A
      (`registre_film/oracle_export.sql`) : un `COPY (...) TO '<repo>/.ai/V7.5/replay2d/registre_film/census_modes_2026-08.tsv' (HEADER, DELIMITER E'\t')`.
      **La base ne s'ouvre JAMAIS en RW** (§3.5).
- [x] D1.2 Croiser le TSV avec la presence d'un repertoire `data/cache/film_chunks/<8 premiers
      caracteres>` : un film present ou absent, par match.
- [x] D1.3 Table par mode, versee au journal du plan : Total Control, Oddball, Extraction,
      Stockpile, Assaut, Land Grab, KOTH, Strongholds, CTF — pour chacun : matchs au registre,
      matchs AVEC film, cartes distinctes, et la liste des identifiants courts retenus.
- [x] D1.4 Statuer CHAQUE mode du §2 : ouvert (phase dediee) ou `[!]` corpus avec son chiffre.
      Un mode a moins de films que son seuil exige (§2) est `[!]` — le seuil ne se rebaisse pas.

**Gate D1** : le TSV est commite, la table est au journal, chaque mode du §2 porte un statut chiffre.
Commandes NUES (depuis `apps/go-api`) :

    go build ./...
    go vet ./...

**STOP — validation superviseur.** D1 peut retirer D3 et/ou D4 du lot ; il ne peut pas en ajouter.

### D2 — KOTH : MESURER le proprietaire de la colline (aucun changement de production)

- [x] D2.1 Instrument sous garde `ETAT_FILM` (racine du cache film), LECTURE SEULE, dans
      `internal/analysis/replay` : pour chaque film KOTH du corpus, rendre le designateur elu, ses
      periodes, et la serie de tag 4 du slot voisin — en APPELANT le code de production
      (`hillDesignatorOf`), jamais en recopiant sa grammaire.
      **FAIT** — `colline_proprietaire_d2_test.go`. **ECART ASSUME SUR LA GARDE** : l'instrument
      reutilise `ctCharge` (harnais du lot C-ter volet 1) et donc la garde **`ZONE_FILM`**, un
      film par processus, au lieu d'inventer un `ETAT_FILM` et un troisieme chargeur de film. La
      regle des deux copies l'emporte sur le nom ecrit au plan. Appelle bien la production :
      `zoneSeriesOf`, `hillDesignatorOf`, `mergeZoneRuns`, `zoneNeutralOwner` — zero grammaire
      recopiee. Saute proprement sans film (verifie : SKIP, exit 0).
- [x] D2.2 Confronter, periode par periode, la valeur de tag 4 au camp dont le SCORE DE MODE monte
      (`objectiveevents.SeriesTotal(recs, objectiveevents.ModeScoreComponent, true)` — les slots
      d'equipe 6 et 8). Publier numerateur ET denominateur, par film.
      **FAIT, ET C'EST LA QUE LA MESURE S'ARRETE** : l'ORACLE ne tient pas (chiffres au journal
      ci-dessous). La confrontation se fait par INTERVALLE DE PROPRIETE CONSTANTE plutot que par
      periode designee — plus fin, plus de confrontations possibles, et c'est exactement la
      granularite ou le score tique. Ce raffinement n'a pas suffi.
- [x] D2.3 Les DEUX temoins du §2.2 : permutation des collines, decalage +20 s. Publier les deux.
      **FAIT** — mesures au journal. Ils ne tranchent rien, faute de signal a departager.
- [!] D2.4 Corroboration `th=10` : acteur de chaque prise de colline -> xuid (`SlotIdentityResolved`)
      -> equipe (roster fige au corpus, aucune base ouverte). Publie comme CONTROLE.
      **NON ATTEINT** : le gate se juge sur D2.2, et D2.2 a rendu un negatif. Enchainer sur un
      SECOND oracle apres l'echec du premier serait changer d'oracle en cours de phase — le
      contournement exact que le contrat interdit. C'est au superviseur d'arbitrer une reprise
      sur cet oracle-la (cf. condition de reprise ci-dessous), pas a l'executeur de la decider.
- [x] D2.5 Verdict au journal : seuil du §2.2 tenu ou NON TENU, avec les chiffres. Non tenu =
      NEGATIF ecrit, item `[!]`, et D5 ne publie rien pour KOTH. **NEGATIF ECRIT.**

**Gate D2 : NON ATTEINT (2026-08-26). NEGATIF ECRIT — ET IL PORTE SUR L'ORACLE, PAS SUR LE
CANAL.**

| film | canal tag 4 (slot voisin) | oracle : slots d'equipe | confrontables | accord | temoin decalage |
|---|---|---|---|---|---|
| `01e1f945` | slot 1472, **99** emissions | 2 (series de 3 et 2 points) | **2** | 2/2 | 2/3 = 66,7 % |
| `0a247154` | slot 1623, **213** emissions | 2 (series de 4 et 3 points) | **1** (bijection DEGENEREE) | 1/1 | 2/3 = 66,7 % |
| `606d9844` | slot 1499, **13** emissions | **1 SEUL** (strict ET brut) | 0 | — | — |
| `8076f97f` | slot 1541, **35** emissions | **1 SEUL** (strict ET brut) | 0 | — | — |

**FILMS AVEC UN DENOMINATEUR EXPLOITABLE : 0 SUR 4.** Le seuil demandait >= 90 % sur >= 3 films ;
deux films rendent UNE ou DEUX confrontations (dont une a bijection degeneree — une seule valeur
observee, qui « explique » tout par construction), et les deux autres n'ont aucun oracle. Le seuil
n'est pas rebaisse, et un accord de 2/2 ne se presente pas comme un resultat : sur deux
confrontations, il est indistinguable du hasard.

**LA CAUSE EST NOMMEE, ET ELLE INVALIDE L'HYPOTHESE D'ORACLE DU §2.2.** Le score de mode KOTH
tel que le FILM le replique n'est PAS un compteur de secondes de colline : c'est un compteur de
**COLLINES GAGNEES**. Ses series valent 3 et 2 points sur `01e1f945`, 4 et 3 sur `0a247154` —
exactement les scores que l'API publie pour ces matchs (3-2 et 4-2). Il ne s'incremente donc
qu'a la FIN d'une periode, quelques fois par match, et non pendant la garde. La remarque de la
chronique du contrat (« l'API compte des secondes de colline en KOTH ») decrit certains matchs,
pas ce composant : elle ne pouvait pas servir d'oracle continu, et c'est ce que la mesure etablit.

**ET SUR DEUX FILMS LE FILM NE REPLIQUE QU'UN SEUL CAMP.** `606d9844` ne porte que le slot 8,
`8076f97f` que le slot 6 — et le DIAGNOSTIC ecarte notre propre filtre : la lecture NON STRICTE
(sans `longestRun`) rend exactement les memes slots. Ce n'est pas nous qui jetons la serie du
second camp, c'est le film qui ne la porte pas. Ces deux matchs sont precisement ceux dont l'API
publie de GROS scores (105-8 et 78-105), ce qui rend la divergence de nature encore plus nette.

**CE QUI N'EST PAS EN CAUSE, ET QU'IL FAUT DIRE** : le canal. Le slot voisin du designateur
emet 13 a 213 valeurs par film, le designateur est elu sur les quatre films (2 a 5 bascules), et
la structure de l'objet de mode se confirme a chaque fois. **Le proprietaire de la colline reste
donc NON MESURE, pas REFUTE.**

**CONDITION DE REPRISE (a arbitrer par le superviseur, cf. §6)** : un oracle CONTINU du camp qui
tient la colline. Deux pistes, dans l'ordre de cout :
1. **Les evenements `th=10` de prise de colline** (item D2.4, non atteint) : acteur -> xuid par
   `SlotIdentityResolved` -> camp par le roster fige. Approximatifs (5-20 s) et peu nombreux,
   mais ils datent des PRISES, ce qui suffirait a orienter la bijection sans exiger une serie
   continue. C'est la reprise la moins chere.
2. **Le score PERSONNEL** (`PersonalScoreComponent`) : en KOTH il monte pendant la garde pour
   chaque joueur present sur la colline. Il donnerait l'oracle continu qui manque, au prix du
   pont slot -> xuid -> camp. C'est la piste qui rendrait le gate du §2.2 jugeable tel qu'il est
   ecrit.

**Commandes NUES du gate (inchangees, pour la reprise)** :

    go build ./...
    go vet ./...
    go test ./internal/analysis/replay/ -run TestEtatVivantKoth -v -timeout 60m
    golangci-lint run --new-from-merge-base=origin/main

(le meme `go test` sans `ETAT_FILM` doit SKIP proprement — c'est ce qui tourne en CI.)

### D2-bis — KOTH : le meme canal, un ORACLE DE BASCULES (arbitrage superviseur du 2026-08-26)

> **CE BLOC EST ECRIT AVANT LA MESURE, ET COMMITE AVANT ELLE.** C'est la condition meme de sa
> validite : une tolerance choisie apres avoir vu les resultats ne serait plus un seuil, ce serait
> un reglage. Le commit qui porte ce texte ne contient AUCUN chiffre de mesure.

**POURQUOI UN SECOND ORACLE, ET PAS UNE COURSE D'ORACLES.** D2 a rendu un negatif dont la cause
est nommee : le score de MODE en KOTH compte des collines GAGNEES, pas des secondes de garde, et
deux films sur quatre n'en portent qu'un camp. Les evenements `th=10` de prise, eux, existent
INDEPENDAMMENT de ce compteur — ils sont lus au footer, un par prise, avec l'acteur en xuid. Les
quatre films redeviennent donc exploitables. Le canal teste ne change pas : c'est toujours le
tag 4 du slot voisin du designateur.

**CE QUI EST TESTE, EN UNE PHRASE** : a chaque prise de colline, le canal de propriete BASCULE
vers le camp du preneur.

- [x] D2b.1 Oracle : les evenements `th=10` de type prise de colline du film (acteur -> xuid ->
      camp par le ROSTER FIGE du corpus, aucune base ouverte), poses sur l'axe de frames du rejeu
      par `p2aFrameOf` (l'origine est retranchee — sans quoi tout serait decale de `originMs`).
      **FAIT** — 43 a 92 prises par film, camp connu, zero prise hors axe.
- [x] D2b.2 Canal : les BASCULES du tag 4 du slot voisin — les changements de valeur vers une
      valeur NOMMEE (0 ou 1), le neutre n'etant pas une prise. Segmentation par `mergeZoneRuns`,
      comme la production. **FAIT** — 13 a 213 emissions selon le film.
- [x] D2b.3 Appariement et accord : pour chaque prise, la bascule la PLUS PROCHE dans la
      tolerance ; accord sous la MEILLEURE bijection valeur <-> camp, denominateurs publies.
      **FAIT** — instrument `colline_proprietaire_d2bis_test.go`.
- [x] D2b.4 Temoins, et le verdict. **FAIT — GATE NON ATTEINT**, chiffres ci-dessous.

**LA TOLERANCE : +/- 20 s, ET ELLE VIENT DE LA DONNEE, PAS D'UN REGLAGE.** Le decodeur qualifie
lui-meme ces evenements d'`approx (~5-20s)` (`objectiveevents/extract.go`, `extractFromTh10`) :
l'instant lu est celui du bloc de temps fort, pas celui de l'action. Vingt secondes est la BORNE
HAUTE de cette imprecision documentee ; la retenir, c'est refuser de trancher plus finement que
l'oracle ne le permet.

**ET C'EST POURQUOI LE TEMOIN DE DECALAGE PASSE A +60 s — correction assumee a la consigne.**
L'arbitrage demandait « temoins permutation + decalage 20 s inchanges ». Un decalage de 20 s
tombe EXACTEMENT DANS une fenetre de tolerance de +/- 20 s : il ne deplacerait pas l'appariement,
et rendrait donc le meme taux que le signal — un temoin qui ne peut pas echouer n'est pas un
temoin. Le decalage temoin est porte a **+60 s** (trois fois la tolerance). Le +20 s est mesure
et publie QUAND MEME, pour la continuite avec D2, mais il est lu comme ce qu'il est : un controle
de stabilite interne a la fenetre, jamais un temoin negatif.

**LES SEUILS SONT INCHANGES** : accord >= **90 %** sur >= **3 des 4** films exploitables ;
temoin de permutation <= **60 %** ; temoin de decalage +60 s <= **60 %**.

**REGLE D'ESCALADE, ECRITE D'AVANCE** : un film offrant moins de **6** prises confrontables ne
compte NI POUR NI CONTRE (denominateur insuffisant, il sort du calcul et le dit). S'il reste
**moins de 3** films exploitables, la mesure S'ARRETE et bascule sur l'oracle du score PERSONNEL
— jamais les deux a la fois, et jamais apres avoir vu le resultat du premier sur le meme film.

**Gate D2-bis** : les trois seuils ci-dessus. Commandes NUES, un film par processus :

    go vet ./internal/analysis/replay/
    ZONE_FILM=<chunks d'un film> go test ./internal/analysis/replay/ -run CollineProprietaireD2Bis -v -timeout 60m
    go test ./internal/analysis/replay/...

**Gate D2-bis : NON ATTEINT (2026-08-26), SUR SES DEUX CLAUSES. Mais le negatif a change de
nature — le canal n'est plus muet, il est INSUFFISANT.**

| film | duree | emissions du canal | SIGNAL | temoin a (permutation) | temoin b (+60 s) | controle (+20 s) |
|---|---|---|---|---|---|---|
| `01e1f945` | 540 s | 99 | **89,1 %** (57/64) | 100 % mais DEGENERE (7/7 sur 66) | **55,6 %** | 50,0 % |
| `0a247154` | 787 s | 213 | **88,0 %** (81/92) | 100 % mais DEGENERE (5/5 sur 92) | **56,6 %** | 57,3 % |
| `606d9844` | 235 s | 13 | **87,1 %** (27/31) | **82,8 %** | **93,8 %** (degenere) | 71,9 % |
| `8076f97f` | 349 s | 35 | **71,7 %** (38/53) | **83,3 %** (10/12) | **76,7 %** | 70,8 % |

- **Accord >= 90 % : 0 film sur 4.** Les quatre films sont EXPLOITABLES (31 a 92 prises
  confrontables, tous au-dessus du minimum de 6) — la regle d'escalade ne se declenche donc PAS,
  et le gate echoue sur son seuil, pas sur son corpus.
- **Temoin de decalage <= 60 % : 2 films sur 4 seulement.**

**LE SEUIL N'EST PAS REBAISSE, ET IL NE DOIT PAS L'ETRE ICI** — mais ce qu'on voit merite d'etre
dit avec precision, parce que ce n'est pas le meme negatif qu'en D2 :

*Sur les DEUX films longs*, le canal se detache nettement de ses temoins : 89,1 % contre 55,6 %,
88,0 % contre 56,6 %. Un ecart de plus de trente points, dans le bon sens, avec des denominateurs
de 64 et 92. **Ce canal porte quelque chose.** Il manque 1 a 2 points au seuil.

*Sur les DEUX films courts*, tout se confond : le signal tombe a 87,1 % et 71,7 %, les temoins
montent a 82,8 % et 93,8 %. Ces deux films sont precisement ceux dont le canal ne parle presque
pas (13 et 35 emissions, 2 bascules de designateur) — le meme trait qui les avait deja rendus
inexploitables en D2, pour une autre raison.

**LA TOLERANCE N'A PAS ETE ABUSIVE** : l'ecart median prise <-> bascule vaut 31 a 124 frames
(3,1 a 12,4 s) pour une fenetre de 20 s, et le controle a +20 s tombe a 50 % sur le meilleur film
— l'appariement est donc temporellement NET, bien plus que la fenetre ne l'autorisait.

**UNE RESERVE METHODOLOGIQUE, ECRITE ET NON CORRIGEE.** Sur `606d9844`, le « pire autre canal »
elu par le temoin de permutation est le slot **1500**, c'est-a-dire `d.slot+2` — le CAPTEUR du
MEME objet de mode ([tag 5 designateur][tag 4 proprietaire][tag 4 capteur][tag 3 jauge],
`zone_states_hill.go`). Un canal frere du meme objet n'est pas une permutation : il porte
plausiblement une information voisine, et l'opposer au signal revient a comparer l'objet a
lui-meme. **Ce defaut n'est PAS corrige apres coup** — redefinir un temoin en ayant vu son
resultat serait exactement le reglage que le protocole interdit. Il est signale au superviseur,
qui peut decider d'un temoin excluant les slots `d.slot..d.slot+3` et d'une REMESURE, protocole
re-ecrit d'abord.

**CE QUI RESTE VRAI APRES DEUX ORACLES** : le canal existe, il est elu sur 4 films sur 4, et il
n'a jamais ete refute — il n'a pas atteint le niveau de preuve exige. **NON MESURE AU SEUIL, pas
refute.** Rien n'est publie, `hillStatesOf` n'est pas touche.

**CONDITION DE REPRISE — UNE SEULE PISTE RESTE, et elle etait deja ecrite** : l'oracle du score
PERSONNEL (`PersonalScoreComponent` + pont slot -> xuid), le seul oracle CONTINU disponible. La
regle d'escalade ne l'a pas declenche (les 4 films sont exploitables) : y aller demande un
arbitrage explicite du superviseur, pas une initiative d'executeur. Deuxieme option, non
technique : le superviseur ou l'utilisateur peut juger que 88-89 % avec un temoin a 56 % suffit
pour un calque de rejeu — c'est une decision PRODUIT sur le niveau de preuve, et elle ne
m'appartient pas.

### D2-ter — KOTH : le meme canal, l'oracle CONTINU du score PERSONNEL

> **ECRIT ET COMMITE AVANT LA MESURE**, comme D2-bis. Le commit qui porte ce texte ne contient
> aucun chiffre de resultat.

**LA JUSTIFICATION D'ARBITRAGE, ET ELLE EST FINE.** D2-bis a mesure 87-89 % avec une tolerance
de +/- 20 s. Or cette tolerance porte l'imprecision de l'ORACLE, pas celle du canal : les
evenements `th=10` sont dates au bloc de temps fort, pas a l'action. Les 11 a 13 % d'ecart
pourraient donc etre le PLANCHER DE BRUIT de l'oracle et non l'erreur du canal. Un oracle
CONTINU — qui ne demande aucune tolerance d'appariement — mesure le canal lui-meme. C'est le
dernier oracle disponible ; il n'y en aura pas de quatrieme.

**CE QUI EST TESTE** : pendant qu'un camp TIENT la colline selon le canal, ce sont les joueurs
DE CE CAMP dont le score personnel monte.

- [x] D2t.1 Pont slot -> xuid par les INSTANTS DE MORT (`objectiveevents.SlotIdentityByDeaths`,
      celui de la production) puis xuid -> camp par le ROSTER FIGE du corpus. Aucune base.
- [x] D2t.2 Oracle : `SeriesTotal(recs, PersonalScoreComponent, false)`, serie cumulee par slot
      de JOUEUR. Delta par CAMP sur chaque intervalle = somme des deltas de ses joueurs.
- [x] D2t.3 Intervalles : les memes qu'en D2 — valeur du tag 4 CONSTANTE et NOMMEE, segmentation
      `mergeZoneRuns`. Duree minimale **5 s** : le score personnel tique a la seconde, et un
      intervalle plus court n'accumule rien de lisible.
- [x] D2t.4 Temoins et verdict.

**LA CONFRONTATION EXIGE UNE DOMINANCE, ET LE SEUIL EST ECRIT ICI.** En KOTH le score personnel
monte AUSSI par les frags, des deux cotes : « un seul camp marque » ne se produirait presque
jamais. Un intervalle est donc CONFRONTABLE quand un camp domine nettement —
`max >= 2 x min` **ET** `max >= 5` points. Sinon il s'abstient, et l'abstention est COMPTEE.
Ces deux nombres sont poses avant la mesure : 2x ecarte le coude a coude ou le bruit des frags
decide, 5 points ecarte les intervalles ou personne n'a rien accumule.

**TEMOIN (a) — PERMUTATION, AVEC LA LECON DE `606d9844`.** En D2-bis, le « pire autre canal »
elu etait `d.slot+2`, le capteur du MEME objet de mode : un frere, pas une permutation.
L'exclusion est desormais **STRUCTURELLE** et non une liste : l'objet de mode occupe quatre
slots consecutifs a partir du designateur ([tag 5][tag 4 proprietaire][tag 4 capteur][tag 3
jauge], `zone_states_hill.go`), donc **tout slot de `d.slot` a `d.slot+3` est exclu du temoin**.
Un canal frere ne peut plus etre oppose a l'objet dont il fait partie.

**TEMOIN (b) — DECALAGE +60 s.** Sans fenetre de tolerance ici, le decalage est un temoin plus
faible qu'en D2-bis (le score s'accumule continument) : il est mesure et publie, mais c'est le
temoin (a) qui porte la charge de la preuve. Le dire d'avance evite de le surinterpreter apres.

**SEUILS INCHANGES** : accord >= **90 %** sur >= **3 des 4** films exploitables ; temoins
<= **60 %**. **ESCALADE** : moins de **6** intervalles confrontables = film non exploitable.
**IL N'Y A PAS DE D2-quater** : si le gate echoue encore avec un contraste propre, la decision
passe a l'utilisateur, elle ne se remesure pas.

**Gate D2-ter** — commandes NUES, un film par processus :

    go vet ./internal/analysis/replay/
    ZONE_FILM=<chunks d'un film> go test ./internal/analysis/replay/ -run CollineProprietaireD2Ter -v -timeout 60m
    go test ./internal/analysis/replay/...

**Gate D2-ter : NON ATTEINT (2026-08-26). ET L'ORACLE EST EN CAUSE UNE TROISIEME FOIS —
CETTE FOIS LA DEMONSTRATION EST DIRECTE.**

| film | pont | SIGNAL | exploitable | delta dominant / domine (median) | temoin b (+60 s) |
|---|---|---|---|---|---|
| `01e1f945` | 8/8 | **53,3 %** (8/15) | oui (15) | **150 / 25** | 55,6 % |
| `0a247154` | 8/8 | **69,2 %** (18/26) | oui (26) | **150 / 0** | 60,9 % |
| `606d9844` | 5/5 | 80,0 % (4/5) | **NON** (5 < 6) | — | degenere |
| `8076f97f` | 8/8 | **100 %** (6/6) | oui (6, tout juste) | — | 66,7 % |

Trois films exploitables ; accord >= 90 % sur **1 sur 3** (et sur six confrontations). **GATE
NON ATTEINT.** Le temoin de permutation est SANS OBJET sur les deux films les plus fournis : une
fois les slots freres exclus structurellement, aucun autre canal ne porte assez d'intervalles
confrontables. L'exclusion demandee a donc bien ete appliquee — et elle a vide le temoin.

**LA CAUSE EST MESUREE, PAS SUPPOSEE.** Le diagnostic d'ampleur (pose apres le verdict, sans le
changer) donne un delta dominant MEDIAN de **150 points** contre **0 a 25** pour le camp domine.
Un frag vaut environ cent points, un tic de colline quelques-uns : **ce que « domine » mesure,
ce sont les FRAGS, pas la garde.** L'oracle continu ne regarde donc pas la colline — il regarde
qui a tue pendant l'intervalle, ce qui n'est pas la meme chose et peut meme s'y opposer (l'equipe
qui pousse pour reprendre la colline tue en entrant). Sur `01e1f945` la bijection retenue est
d'ailleurs INVERSEE par rapport a celle de D2-bis, ce qui est la signature d'un signal absent.

**CE QUE CELA CHANGE POUR LA LECTURE DE D2-bis, ET C'EST L'INVERSE DE L'HYPOTHESE D'ARBITRAGE.**
L'arbitrage supposait que les 87-89 % de D2-bis pouvaient etre le plancher de bruit de son
oracle. L'oracle continu ne fait pas mieux : il fait NETTEMENT MOINS BIEN, pour une raison
nommee et chiffree. **Le meilleur etat des connaissances sur ce canal reste donc D2-bis :
88-89 % d'accord contre un temoin a 56 %, sur les deux films longs.**

**IL N'Y A PAS DE D2-quater** — le protocole le disait d'avance, et les trois oracles disponibles
sont epuises : score de MODE (compte des collines gagnees, un camp manquant sur deux films),
prises `th=10` (imprecision de +/- 20 s), score PERSONNEL (domine par les frags). **La decision
passe a l'utilisateur.**

### D3 — TOTAL CONTROL : MESURER l'etat des zones (ouverte seulement si D1 le permet)

- [ ] D3.1 Denombrer, par film de Total Control : slots `ti=13` emetteurs, tags observes, taux de
      chainage (le temoin de largeur du balayage, `ManagedPropertyScan.Chained` vs `Walked`).
- [ ] D3.2 Chercher un DESIGNATEUR (H1) par le predicat de production `hillDesignatorOf` : existe-t-il
      un slot de tag 5 chaine dont le voisin porte un proprietaire qui parle ? Combien de bascules,
      et tombent-elles sur les bornes de MANCHE (`objectiveevents.RealRounds` /
      `SeriesByRound`) ?
- [ ] D3.3 Appariement (H2) : prises nommees `zone_captures` -> forme du catalogue par la position
      de leur auteur ; taux d'attribution et temoin decale de 12 m. Cardinalite de l'ensemble
      apparie PAR MANCHE.
- [ ] D3.4 Proprietaire : accord tag 4 / equipe du capteur, avec ses denominateurs.
- [ ] D3.5 Verdict : les trois seuils du §2.3 sont-ils tenus ? Quelle hypothese (H1 ou H2) porte
      les 3 actives ? Non tenu = NEGATIF ecrit, `[!]`, et `totalcontrol_zone` NE rejoint PAS
      `heldZoneRoles`.

**Gate D3** : (1) attribution >= 80 % / temoin <= 20 % ; (2) exactement 3 zones appariees par
manche sur >= 2 films ; (3) accord proprietaire >= 90 %.

---

#### PROTOCOLE OPERATIONNEL — ECRIT ET COMMITE AVANT LA MESURE (2026-08-26)

> Meme regime que D2-bis et D2-ter : le commit qui porte ce texte ne contient AUCUN chiffre de
> resultat. Les seuils ci-dessus ne bougent pas ; ce qui suit dit COMMENT ils se mesurent.

**LE CORPUS.** Les films de Total Control du recensement D1 (4 en « Total Control », 3 en
« Fiesta Total Control »). Leurs identifiants sont listes par `-census`, qui imprime desormais
les identifiants courts des modes a **12 films ou moins** — les modes rares sont precisement ceux
dont on a besoin nommement, les modes massifs restent agreges.

**L'OUTILLAGE, ET POURQUOI IL N'Y A PAS D'INSTRUMENT NEUF POUR LE SEUIL (1).**
`cmd/zone-attribution` EST l'outil de cette mesure — son en-tete le dit : « MESURE le croisement
quel joueur est DANS quelle zone a l'instant d'une prise », avec son temoin negatif a 12 m
obligatoire. Il ne lui manque qu'une chose : son role est ECRIT EN DUR
(`mapvar.RoleStrongholdZone`). Un drapeau `-role` l'ouvre a `totalcontrol_zone` sans toucher a
sa logique. C'est la meme extension que `-census` : le tri des outils existants avant l'ecriture
d'un neuf.

**LA DIVISION DU TRAVAIL ENTRE LES TROIS SEUILS**, parce qu'ils n'ont pas les memes besoins :

	seuil (1) attribution   `zone-attribution -role totalcontrol_zone` — il a la base, donc le
	                        ROSTER, que les instruments du paquet `replay` n'ont pas le droit
	                        d'ouvrir.
	seuil (2) cardinalite   les manches viennent du FILM (`objectiveevents.RealRounds`), les
	                        zones appariees du meme croisement que (1).
	seuil (3) proprietaire  tag 4 confronte a l'equipe du CAPTEUR — donc au roster, donc dans le
	                        meme outil que (1).

**REGLE D'ESCALADE, ECRITE D'AVANCE** : un film offrant moins de **6** prises attribuables ne
compte NI POUR NI CONTRE (denominateur insuffisant). S'il reste **moins de 2** films
exploitables, le seuil (2) est inatteignable par construction et la mesure S'ARRETE — `[!]`
corpus, sans chercher d'oracle de remplacement.

**L'ORDRE EST CELUI DU COUT, ET IL S'ARRETE AU PREMIER ECHEC.** (1) d'abord : sans attribution,
ni la cardinalite ni le proprietaire n'ont de sens, puisque les deux se lisent sur des zones
appariees. Puis (2). Puis (3). Un seuil rate = NEGATIF ecrit, `totalcontrol_zone` NE rejoint PAS
`heldZoneRoles`, et l'entree du titre reste `neutral = true` en formes seules.

**LE NIVEAU DE PREUVE DU SEUIL (3) EST DEJA TRANCHE.** Si l'accord du proprietaire retombe sur le
plafond d'environ 88 % du MEME canal tag 4, avec le meme type d'oracle qu'en D2-bis, la
**decision utilisateur du 2026-08-26 (option a)** s'etend par coherence : c'est le meme canal,
mesure de la meme facon, et il serait incoherent de l'accepter sur la colline et de le refuser
sur les zones. Elle se CITE, elle ne se redemande pas. Elle ne couvre en revanche PAS les seuils
(1) et (2), qui portent sur l'appariement et non sur le canal.

**Gate D3 — commandes NUES** :

    go vet ./cmd/zone-attribution/ ./internal/analysis/replay/
    LEVELUP_REPO_ROOT=<repo> go run ./cmd/zone-attribution -census -cache <repo>/data/cache
    LEVELUP_REPO_ROOT=<repo> go run ./cmd/zone-attribution -role totalcontrol_zone -cache <repo>/data/cache
    go test ./internal/analysis/replay/... ./cmd/zone-attribution/...

### D4 — ODDBALL : MESURER l'identite du crane puis son portage (ouverte seulement si D1 le permet)

- [ ] D4.1 Identite : sur chaque film Oddball, les creations `ti=42` ECARTEES du catalogue d'armes ;
      pour chaque mot de 32 bits distinct, la distance de naissance au plus proche `oddball_spawn`
      du catalogue de carte et l'ecart au plus proche evenement `th=10` de crane. Meme instrument
      de forme que `attachement_phase0_drapeau_test.go`.
- [ ] D4.2 Temoin de SELECTIVITE : compter les AUTRES mots qui reunissent les deux conditions.
      Le seuil exige **zero**.
- [ ] D4.3 Portage : decouper la vie du crane en TROUS (l'objet cesse d'emettre) ; pour chaque
      trou, chercher le joueur dont le score PERSONNEL s'incremente sur toute sa duree
      (`objectiveevents.SeriesTotal(recs, objectiveevents.PersonalScoreComponent, false)`, slot ->
      xuid par `SlotIdentityResolved`). Temoin : un joueur tire au hasard hors trou.
- [ ] D4.4 Verdict : les deux seuils du §2.4. Non tenu = NEGATIF ecrit, `[!]`, la ligne du registre
      des reports est mise a JOUR (texte fourni au CR), et D5 ne publie rien pour Oddball.

**Gate D4** : (1) UN seul mot candidat, le meme sur >= 2 films, 0 autre candidat ; (2) >= 90 % des
trous a porteur unique, temoin <= 5 %. Commandes NUES :

    go build ./...
    go vet ./...
    go test ./internal/analysis/replay/ -run TestEtatVivantOddball -v -timeout 60m
    golangci-lint run --new-from-merge-base=origin/main

### D5 — PUBLICATION : un SEUL bump de schema pour tout ce que D2-D4 ont tenu

Cette phase ne mesure rien : elle publie ce que les gates precedents ont valide, et RIEN d'autre.

- [ ] D5.1 Producteurs, un par verdict tenu (KOTH `Owner` sur les intervalles de colline ; Total
      Control : `totalcontrol_zone` dans `heldZoneRoles` + la voie des 3 actives ; Oddball : entree
      `[[objective_objects]]` `family = "ball"` EN+FR + calque de portage du crane). Un verdict non
      tenu n'a AUCUN code.
- [ ] D5.2 `Coverage` : chaque calque publie ses DENOMINATEURS et ses rejets par cause. Regle du
      depot : un calque sans couverture se lit comme une exhaustivite ; l'ABSENCE du bloc doit
      rester distincte du zero.
- [ ] D5.3 Le TRIPLET de version (§3.2) : `replay.SchemaVersion` (numero libre au moment du lot,
      >= 20) avec sa chronique en tete de `document.go` ET dans le fichier de contrat du calque ;
      `wantReplayDocumentFields` + sa ligne de chronique ; `EXPECTED_REPLAY_SCHEMA_VERSION`.
- [ ] D5.4 Contrat client : `go run ./cmd/openapi-gen` (jamais d'edition a la main de
      `api/openapi.yaml`), `make generate-types`, frontiere de nullabilite web
      (`NULLABLE_ARRAYS` / `NULLABLE_ARRAY_PATHS` et `normalizeReplayDocument` — tableaux
      IMBRIQUES compris).
- [ ] D5.5 Golden d'assemblage re-congele (`testdata/assembly_000d5950.golden`) et TEMOINS
      re-cuits : les films des gates D2-D4 UNIQUEMENT, **un film par processus**, via
      `cmd/replay-build --map <carte> --facts <faits.json> <matchId>` (aucune base ouverte).
- [ ] D5.6 Tests : producteurs testes PURS (sans film) ; un test de non-regression par calque.

**Gate D5** : commandes NUES (depuis `apps/go-api` puis `apps/web`) :

    go build ./...
    go vet ./...
    go test ./internal/analysis/... ./internal/replaybuild/... ./contracttest/... ./internal/archlint/...
    go run ./cmd/openapi-gen -check
    golangci-lint run --new-from-merge-base=origin/main
    npx tsc -b --force
    npx eslint .
    npx vitest run src/features/match-replay src/lib

Mise en garde mesuree deux fois (journaux des 18/08) : **ne pas cuire de film pendant le gate
web** — les garde-rails qui balaient `src/` expirent au delai de 5 000 ms sur machine chargee, et
ces echecs ne sont pas des regressions.

### D6 — **ITEM D-R** : la progression de capture et l'appartenance SUR LA FORME de la base

> Ajout de perimetre valide par l'utilisateur le 2026-08-25 (item D-R du plan parent
> `PLAN_REPLAY2D_NOTION_2026-08-25.md`, lot D). **Cette phase ne demarre qu'APRES la fusion du lot
> A** (memes fichiers de calque). Elle est INDEPENDANTE de D2-D5 : elle ne consomme que de la
> donnee DEJA publiee (schema 18) — le superviseur peut donc la resequencer avant D2 sans rien
> casser, tant que A est fusionne.

**CE QUI EXISTE DEJA, et qu'il ne faut ni re-prouver ni reecrire** (releve du superviseur,
recoupe sur pieces) :

- l'appartenance TEINTE DEJA la forme reelle : `paintZoneState`
  (`apps/web/src/features/match-replay/zoneStatesLayer.ts:335-350`) remplit le trace de
  `traceZonePath`, la MEME fonction que le calque statique — jamais une seconde geometrie ;
- le « cercle » vu a la capture est l'arc de jauge EXTERNE `drawGaugeArc`
  (`zoneStatesLayer.ts:366-377`), plus le pulse ponctuel de capture d'`objectivesLayer.ts` ;
- une hierarchie d'encres existe deja en embryon : l'arc est trace avec `colorOfCapturer(owner)`
  (`useZoneStates.ts:80-81`), c'est-a-dire l'encre du camp OPPOSE au proprietaire courant.

- [x] D6.1 **Remplissage progressif de la FORME**, proportionnel a la jauge lue en escalier
      (`zoneGaugeAt`, `zoneStatesLayer.ts:100`), par decoupe canvas (`ctx.clip` sur
      `traceZonePath`) puis balayage — **agnostique de la forme** : une boite orientee comme un
      cylindre passent par le meme trace. Il REMPLACE l'arc externe (`drawGaugeArc` est retire
      avec son alpha et ses tests : regle 0 code mort, pas de bascule qui garde les deux).
      **FAIT** — `paintCaptureFill` (`zoneStatesPaint.ts:181-203`), balayage VERTICAL bas ->
      haut. Le sens du balayage est un ARBITRAGE que le plan laissait ouvert : une fraction
      d'ANGLE n'est pas une fraction d'AIRE sur une boite orientee (un demi-tour d'arc y couvre
      une part qui depend de l'orientation, pas de la capture), alors qu'une bande horizontale
      clippee est proportionnelle sur toute forme — et elle ne balaie pas la lettre centrale.
      `drawGaugeArc`, `ZONE_GAUGE_ALPHA`, `ZONE_GAUGE_WIDTH`, `ZONE_GAUGE_MIN_RADIUS` et
      `ZoneGaugeArc` sont SUPPRIMES.
- [x] D6.2 **Hierarchie des encres** quand une equipe capture une base tenue par l'adversaire :
      progression de l'attaquant FRANCHE par-dessus la teinte du proprietaire AFFAIBLIE. Le
      detail (valeurs d'alpha, sens du balayage, ordre de peinture) se TRANCHE DANS CE PLAN au
      moment de l'ouverture de la phase, jamais en cours d'implementation.
      **LIMITE DE DONNEE A INSCRIRE, ET ELLE EST STRUCTURELLE** : le film ne dit PAS qui capture.
      Le tag 4 dit qui TIENT ; l'attaquant n'est deduit que par opposition, ce qui ne vaut que
      dans un match a deux camps ET sur une zone DEJA TENUE. Sur une zone NEUTRE, `colorOfCapturer`
      rend `null` aujourd'hui et l'arc se peint en encre neutre : la progression y restera
      NEUTRE — ne pas inventer un attaquant.
      **FAIT** — `ZONE_UNDER_CAPTURE_FILL_ALPHA = 0,16` (le proprietaire recule) contre
      `ZONE_CAPTURE_FILL_ALPHA = 0,55` (l'attaquant passe franc), `zoneStatesPaint.ts:88-89`.
      La limite de donnee est tenue et testee : une zone neutre en capture rend une progression
      NEUTRE, une zone tenue par un camp non situable aussi.
- [x] D6.3 **Alphas d'appartenance** : evaluer le renforcement de `ZONE_HELD_FILL_ALPHA` (0,22
      aujourd'hui, `zoneStatesLayer.ts:198`) et de `ZONE_ACTIVE_FILL_ALPHA` (0,3) — la teinte
      actuelle est jugee trop discrete par l'utilisateur. Valeurs proposees au plan, verdict au
      gate VISUEL.
      **FAIT** — tenue 0,22 -> **0,30**, active 0,30 -> **0,42**. L'ECART entre les deux se
      creuse en meme temps qu'ils montent : renforcer la seule zone tenue aurait efface ce qui
      distingue la colline active, le repere le plus utile de la carte. L'echelle COMPLETE
      compose avec le retrait des zones libres du lot A et vit dans `ZONE_ALPHA_ORDER` —
      libre 0 < en perte 0,16 < tenue 0,30 < active 0,42 < progression 0,55 —, testee par son
      ORDRE et non par ses valeurs, precisement pour que le gate visuel puisse toutes les bouger.
      **Verdict VISUEL a l'utilisateur.**
- [x] D6.4 **POINT DE CONTROLE DONNEES (gate, pas un item de confort)** : verifier sur les
      artefacts TEMOINS que `coverage.zones.gaugePoints > 0` la ou la progression doit se voir.
      **CORRECTION AU BRIEF DE L'AJOUT DE PERIMETRE** : la serie `gauge` n'a PAS ete validee sur
      KOTH — c'est l'inverse. Elle est publiee **sur les modes a zones SIMULTANEES SEULEMENT
      (Bastion)**, la ou le tag 3 est la vraie rampe de capture (97 % des captures precedees
      d'une rampe, lot C-bis), et elle est **DELIBEREMENT ABSENTE sur une colline de KOTH**, ou
      le meme tag est un compteur de transfert d'environ une seconde
      (`replay/zone_states_gauge.go:34-37` ; `document_zones.go`, § `ZoneState.Gauge` :
      « `coverage.zones.gaugePoints` y vaut 0 »). Le controle se lit donc ainsi :
      Strongholds = points ATTENDUS (si zero sur un temoin Bastion : `[!]`, condition de reprise) ;
      KOTH = zero ATTENDU, ce n'est pas un defaut, et la colline garde son seul etat
      d'appartenance ; Total Control = INCONNU tant que D3 n'a pas mesure, donc `[!]` par defaut.
      **`[x]` — FAIT LE 2026-08-26, PAR LECTURE PURE DU CACHE. Aucune cuisson n'a ete
      necessaire** : sept artefacts du cache local portent deja `coverage.zones`, tous au
      schema 18, et ils separent les deux methodes sans ambiguite.

      | artefact | methode | `gaugePoints` | zones | spans |
      |---|---|---|---|---|
      | `7344d24f` | `captures+geometry` | **1 701** | 3 | 39 |
      | `696a9d7c` | `captures+geometry` | **1 794** | 3 | 37 |
      | `af13e2b2` | `captures+geometry` | **467** | 3 | 8 |
      | `0a247154` | `designator+geometry` | **0** | 5 | 6 |
      | `01e1f945` | `designator+geometry` | **0** | 4 | 5 |
      | `606d9844` | `designator+geometry` | **0** | 3 | 3 |
      | `8076f97f` | `designator+geometry` | **0** | 3 | 3 |

      **VERDICT : le contrat du producteur est tenu sur pieces.** Les trois artefacts a zones
      SIMULTANEES portent tous une serie de jauge fournie (467 a 1 794 points) ; les quatre
      films a COLLINE en portent zero, exactement comme `zone_states_gauge.go` l'annonce
      (« EN KOTH, RIEN »). La correction de fait du §6.2 est donc confirmee une seconde fois,
      et par la donnee cette fois : la jauge est une affaire de Bastion, jamais de colline.
      Le rendu de D6 est cadre par cette mesure — sur une colline, la progression ne peut pas
      s'afficher faute de serie, et le calque ne le devine pas : il recoit un tableau vide.

      DENOMINATEUR DE LA LECTURE : 40 artefacts au cache, dont 7 portent `coverage.zones` —
      les 33 autres sont des modes sans zone tenue (le calque ne publie alors rien, et son
      ABSENCE est distincte du zero, cf. `ZonesCoverage`). Total Control reste INCONNU : aucun
      artefact du cache n'en porte, ce que D1 confirmera au recensement.
- [x] D6.5 **Le pulse ponctuel de capture est CONSERVE** : il marque un INSTANT, la progression
      decrit une DUREE — les deux se lisent ensemble (c'est deja l'arbitrage ecrit en tete
      d'`objectivesLayer.ts`). Rien n'est retire de ce cote.
      **FAIT — par NON-ACTION verifiee** : `buildObjectivePulses` et `drawObjectivePulses` ne
      sont pas touches, la famille ZONE garde son role entier.
- [x] D6.6 Tokens semantiques uniquement (aucun hex, aucune classe Tailwind couleur) ; aucun texte
      neuf sur le canvas ; tests : forme du remplissage a 0 / 0,5 / 1, agnosticite boite/cylindre,
      hierarchie sur zone tenue ET sur zone neutre, garde « aucune couleur en dur ».
      **FAIT** — aucune valeur de couleur n'est ecrite dans le lot (les encres arrivent resolues
      de `useZoneStates`, inchange) ; aucun texte neuf (la garde « rien hors le glyphe A-C »
      passe telle quelle). 35 cas sur le calque, dont 6 neufs. **Mordant prouve par DOUBLE
      MUTATION** : remplissage par le HAUT (`box.top` au lieu de `box.bottom - h`) = 1 echec ;
      hierarchie des encres INVERSEE (0,16 / 0,55 permutes) = 2 echecs. Les deux mutations ont
      ete revertees.

**Gate D6** : volet TECHNIQUE = les trois commandes web NUES de D5, plus le point de controle
D6.4 statue sur les temoins. Volet VISUEL = **l'utilisateur** (planche + en app).

**Gate D6 (volet TECHNIQUE) : TENU (2026-08-26).** Depuis `apps/web` du worktree, apres
`npm ci` et purge de `node_modules/.tmp` : `npm run typecheck` = **0** ; `npx vitest run
src/features/match-replay` = **0** (75 fichiers, 1 088 tests) ; `npx eslint` sur les quatre
fichiers touches = **0** (zero erreur, zero avertissement). Cliquet `ReplayCanvas.tsx` :
**742 lignes, INCHANGE** — le lot ne touche pas le canvas. Aucune commande Go n'a ete lancee.
**Le volet VISUEL reste ouvert** : il appartient a l'utilisateur, et il porte deux questions
precises — les nouveaux aplats (tenue 0,30 / active 0,42) et le sens de balayage bas -> haut.

**UNE EXTRACTION QUE LE PLAN NE PREVOYAIT PAS, ET CE QUI L'A IMPOSEE.** Le passage de l'arc
exterieur au remplissage de la forme a porte `zoneStatesLayer.ts` de 417 a 568 lignes, au-dessus
du seuil du depot. La regle etant d'extraire et jamais d'exempter, la peinture est partie dans
`zoneStatesPaint.ts` (259 l.) : le calque decide CE QU'IL FAUT peindre (etat couvrant la frame,
valeur de jauge, lettre), la peinture decide COMMENT (opacites, hierarchie, ordre des passes,
decoupage). Trois fichiers source sous le seuil apres coup : calque 345, peinture 259,
`objectivesLayer` 414. La coupure suit une couture reelle — un reglage d'encre se change sans
rouvrir la lecture d'etat — et la GEOMETRIE n'a pas ete dupliquee : `traceZonePath` reste la
seule source de la forme, et l'emprise ecran passe par `zoneCornersWorld` / `zoneCanvasRadius`,
deux helpers EXTRAITS de `traceZonePath` lui-meme plutot que recopies.

### D7 — RENDU des calques NEUFS de D5 (web) — **apres D5 ET apres la fusion du lot A**

- [ ] D7.1 KOTH : la colline active se TEINTE de l'encre du camp qui la tient (le patron
      `paintZoneState` existe deja pour Bastion — la meme fonction, pas une seconde copie) ;
      neutre = pas de teinte, trait faible du calque statique.
- [ ] D7.2 Total Control : seules les zones APPARIEES de la manche courante se peignent ; les
      formes du vivier restent au trait faible. Arbitrage n°3 (§6) s'applique ici.
- [ ] D7.3 Oddball : le crane colle au marqueur de son porteur, pose a sa position quand il est
      libre, socle `oddball_spawn` present/absent. Meme frontiere que `flagCarriesLayer.ts` :
      geometrie hors ecran, etat dans la boucle.
- [ ] D7.4 Les pulses SUBSTITUTS de la famille concernee sont RETIRES des que la donnee vivante
      arrive (`buildObjectivePulses`, meme regle que le drapeau). Zero code mort. La famille ZONE
      n'est PAS concernee : son pulse reste (cf. D6.5).
- [ ] D7.5 Tokens semantiques uniquement, strings FR **et** EN, aucun texte dessine sur le canvas
      hors le glyphe de lettre deja autorise.
- [ ] D7.6 Tests : formes, etats, garde « aucune couleur en dur », parite i18n.

Un verdict `[!]` en D2-D4 retire simplement son item ici : **on ne dessine pas ce qui n'est pas
publie.**

**Gate D7** : volet TECHNIQUE = les trois commandes web NUES de D5. Volet VISUEL = **l'utilisateur**
(planche + en app) — il n'appartient pas a l'agent, et aucune verification navigateur n'est faite
par le lot.

### D8 — CLOTURE DU LOT

- [ ] D8.1 Tous les items du plan statues, aucune case vide.
- [ ] D8.2 CR de lot : mesures, seuils tenus ou non, chiffres, et les TEXTES prets a coller pour
      `.ai/thought_log.md` et `.ai/V7.5/REGISTRE_REPORTS.md` (le superviseur les consigne).
- [ ] D8.3 Les lignes de registre a AMENDER sont nommees : crane d'Oddball, `ti=11` objectifs
      vivants, VIP, et toute condition de reprise creee par une phase `[!]`.

---

## 5. Regles dures de ce plan

Mesure AVANT production ; un seuil ecrit ne se rebaisse pas ; un canal refute s'ECRIT ; aucun
rendu avant la donnee ; aucun mode publie sans sa couverture ; **le plan ne promet pas ce que le
film ne porte pas** (VIP, Extraction, Stockpile, Assaut, Land Grab, Firefight sont `[!]` par
construction) ; un seul decodage `filmdec` par process ; un film par processus ; aucune base en
ecriture ; jamais `git add -A` ; aucun push ; aucune attente passive.

---

## 6. ARBITRAGES DEMANDES AU SUPERVISEUR (avant de lancer D1)

1. **Le numero de schema.** Le brief dit 19 ; 19 est PRIS (lot 1 « lecture vide », 2026-08-25). Le
   lot prendra le premier numero libre au moment de sa fusion (>= 20). Confirmer que « UN SEUL
   bump » signifie bien « toutes les publications dans la phase D5 », ce qui est le decoupage
   propose.
2. **La priorite si le corpus est maigre.** Ordre propose, par cout croissant et valeur
   decroissante : **D2 (KOTH proprietaire)** — le moins cher, canal et oracle deja en main ;
   **D3 (Total Control)** — le plus gros trou produit ; **D4 (Oddball)** — un pari de recherche,
   avec un negatif honorable pour issue. Confirmer ou reordonner.
3. **Total Control aujourd'hui, en attendant D3.** Le rejeu d'une partie de Total Control dessine
   deja 13 a 18 formes neutres la ou 3 seulement sont actives (entree servie depuis le 2026-08-25).
   Trois options : (a) laisser en l'etat jusqu'a D3 ; (b) retirer l'entree de la table du titre
   jusqu'a ce que l'etat vivant existe ; (c) la garder et le dire dans l'UI. **Decision produit,
   pas technique.**
4. **Les modes fermes.** Confirmer que Extraction, Stockpile, Assaut, Land Grab, Firefight et VIP
   restent `[!]` pour v7.5, avec leurs conditions de reprise au registre — le plan ne les ouvre pas.
5. **La table `[[objective_objects]]` est FERMEE par son commentaire** (« le crane n'y est PAS, et
   ce n'est pas un oubli »). D4 est precisement la mesure qui autoriserait a l'ouvrir : confirmer
   que le lot a mandat pour l'amender SI le gate D4 passe.
6. **La sequence de l'item D-R (D6).** Il ne depend QUE de la fusion du lot A, pas de D2-D5 :
   c'est un rendu de donnee deja publiee. Le superviseur peut donc le placer AVANT D2 (livrer un
   gain visible tot) ou le laisser ou il est. **Et une correction de fait a acter au passage** :
   le brief de l'ajout de perimetre dit que la serie `gauge` « n'a ete validee que sur KOTH » —
   c'est l'INVERSE (§ D6.4). Elle est publiee sur les modes a zones simultanees (Bastion) et
   volontairement absente sur une colline. La progression sur forme concerne donc Bastion
   d'abord, Total Control apres D3, et JAMAIS la colline de KOTH — qui garde sa seule
   appartenance. Confirmer que ce perimetre reduit est bien celui attendu.
7. **Le chemin de recensement de D1 n'existe pas** (detail au bloc D1). Le plan prescrit un
   `COPY` SQL en lecture seule ; ni le MCP `duckdb` ni le binaire `duckdb` ne sont disponibles, et
   la base est tenue par un serveur. Trois options : exposer le MCP a la session ; ajouter un
   drapeau `-mode` a `cmd/zone-attribution` (mais c'est un outil de MESURE) ; ou un `cmd` de
   recensement dedie. Creer un binaire de production pour un comptage ponctuel est une decision
   de superviseur, pas une initiative d'executeur.
8. **D2-bis : 88-89 % avec un temoin a 56 % — suffisant ou non ?** Le seuil de 90 % n'est pas
   atteint et n'a pas ete rebaisse. Trois suites possibles : basculer sur l'oracle du score
   PERSONNEL (le dernier disponible) ; remesurer avec un temoin de permutation excluant les slots
   FRERES de l'objet de mode (defaut signale, deliberement NON corrige apres coup) ; ou trancher
   que ce niveau de preuve suffit pour un calque de rejeu — cette derniere est une decision
   PRODUIT, elle ne m'appartient pas.
   **TRANCHE EN PARTIE LE 2026-08-26 : LES DEUX PREMIERES OPTIONS SONT EPUISEES.** D2-ter a
   mesure l'oracle du score PERSONNEL — 53-69 % sur les deux films les plus fournis, domine par
   les frags (delta dominant median 150 points contre 0-25 pour le camp domine) — et le temoin
   de permutation FRERES-EXCLUS s'y est revele SANS OBJET. Les trois oracles disponibles sont
   consommes. **Il ne reste que la decision PRODUIT** : 88-89 % d'accord contre un temoin a 56 %
   (D2-bis, films longs) suffit-il pour teinter une colline dans un rejeu ?

---

## 7. Decouvertes (hors perimetre — consignees, NON traitees)

- (2026-08-26, D1) **87 matchs du registre n'ont pas de `pair_name` exploitable, et 8 modes
  sortent du recensement sous un UUID BRUT** (`100e12e4-...`, `f526fb9e-...`). Le libelle de mode
  du registre n'est donc pas toujours un libelle : sur ces lignes, la table de roles du titre ne
  peut rien servir — degradation propre, mais silencieuse. NON TRAITE : c'est la qualite du
  registre, pas le chantier des objectifs.
- (2026-08-26, D1) **`NormalizeModeLabel` rend « urvive The Undead 3.0 »** — le « S » de
  « Survive » est mange par le retrait de prefixe. Un mode s'affiche donc ampute quelque part
  dans le produit. Defaut REEL de la normalisation partagee, hors perimetre de ce plan.
- (2026-08-26, D2) **Le composant de score de MODE ne compte pas la meme grandeur d'un film KOTH
  a l'autre, et deux films n'en portent qu'un camp.** `01e1f945` et `0a247154` rendent des series
  de 2 a 4 points (des collines gagnees) ; `606d9844` et `8076f97f` ne rendent qu'UN slot
  d'equipe, lecture stricte comme brute. La courbe de score PUBLIEE au document (schema 12) est
  donc, sur ces films, partielle ou d'une autre nature que ce que son nom laisse croire — le
  calque `scoreTimeline` publie bien ce que le film porte, mais un lecteur qui attendrait « le
  temps de colline » y trouverait autre chose. NON TRAITE ici : c'est le chantier du score, pas
  celui des objectifs. A verser au registre.
- (2026-08-26, D6) **`colorOfCapturer` est appele avec le PROPRIETAIRE, et rend le camp d'en
  face** (`useZoneStates.ts:90-91`). C'est une DEDUCTION a deux camps, correcte sur une zone
  tenue et muette ailleurs — sur une zone NEUTRE en cours de capture, la page ne peut nommer
  personne et la progression reste neutre. Ce n'est pas un defaut du client : le film ne
  replique aucun proprietaire sur les slots de rampe (mesure du lot C-bis). Si D3 ouvrait un
  mode a plus de deux camps, cette deduction cesserait d'etre valide — la nommer ici pour que
  personne ne la prenne pour une lecture.
- (2026-08-26, D6) **Le fixture de test du calque publie `progress` sur ses intervalles** alors
  que plus rien ne le lit depuis le schema 18 (le client ne dessine plus le sommet). Le champ
  est CONSERVE au fixture a dessein — c'est lui qui rend le cas « sans `gauge`, le sommet ne
  remplace pas la progression » non tautologique — mais il n'a plus aucun lecteur de production.
- **Land Grab est classe `zone` par `ObjectiveTypeOf` mais n'a ni role ni entree de table.** Il a
  donc des evenements nommes que rien ne consomme, et ses hashs `landgrab_zone` sont dans les
  fichiers de carte sans role. Incoherence reelle, non traitee ici (§2.8).
- **`ObjectiveTypeOf` ne connait ni Extraction, ni Stockpile, ni Assaut** : trois modes servis
  statiquement dont AUCUN evenement nomme ne peut sortir. La liste des modes du classifieur et
  celle de `objective_roles.toml` ont diverge sans que rien ne le signale.
- **Une exception au catalogue Total Control** : sur Sylvanus, un objet porte `totalcontrol_zone`
  SANS `totalcontrol_include`, boite de 2,74 x 2,20 x 0,16 m — un decoy probable, publie quand meme
  par contrat (le role vient d'un label explicite, jamais d'une heuristique de taille). Deja au
  registre.
- **`firefight_objective`** : role au catalogue (5 volumes par carte de Firefight), servi par aucun
  mode, faute de savoir quel libelle de manche l'active.

---

## 8. Protocole de reprise de session

1. Lire ce plan de haut en bas : les **statuts d'items** du §4 disent ou en est le lot, le
   **journal du plan** (§9) porte les mesures et les verdicts.
2. Lire la tete de `.ai/thought_log.md` (le superviseur y consigne les fusions).
3. Verifier SUR PIECES avant de coder et avant de cocher : rouvrir le fichier et la ligne cites —
   le code a bouge deux fois pendant la redaction de ce plan (schema 19, entree Total Control).
4. Une phase a la fois, dans l'ordre. Le contrat complet est le skill `plan-execution`.

## 9. Journal du plan (avancement — source de verite pour la reprise)

- 2026-08-25 — **D0 CLOSE.** Plan ecrit sur pieces (code, catalogues, archives de chantier).
  Aucune mesure lancee, aucun code de production touche. Verdicts de faisabilite : KOTH
  proprietaire FAISABLE ; Total Control et Oddball FAISABLES SOUS RESERVE DE CORPUS ; Extraction,
  Stockpile, Assaut, Land Grab, Firefight et VIP `[!]` avec conditions de reprise. Six arbitrages
  au §6, dont DEUX corrections de fait au brief : le numero de schema (19 est PRIS depuis le
  2026-08-25, prochain libre 20) et le perimetre de la serie `gauge` (validee sur Bastion, PAS sur
  KOTH — l'inverse de ce que dit l'ajout de perimetre). **Ajout de perimetre du 2026-08-25 integre**
  en phase D6 (item D-R : progression et appartenance sur la FORME), independante de D2-D5 et
  conditionnee a la seule fusion du lot A ; les anciennes phases D6/D7 deviennent D7/D8.
  **STOP — validation superviseur avant D1.**

- 2026-08-26 — **D6 (item D-R) CLOSE, volet technique.** Base : fusion d'`origin/feat/v75`
  (`ae12cb1a0`, qui apporte le lot A — retrait des zones libres a 0,5 / 1,6 px, seuil
  `owner === null`, encre neutre `divergent-neutral`) puis relecture integrale de
  `zoneStatesLayer.ts` AVANT d'ecrire une ligne. La progression de capture se lit desormais SUR
  la forme (balayage vertical bas -> haut, `ctx.clip` sur `traceZonePath`), l'arc exterieur et
  ses quatre reglages sont supprimes, la hierarchie d'encres compose avec le retrait du lot A
  (`ZONE_ALPHA_ORDER`). Items : D6.1, D6.2, D6.3, D6.5, D6.6 `[x]` ; **D6.4 `[~]`** — le contrat
  « aucune serie = aucune progression » est verrouille par test, mais la lecture de
  `coverage.zones.gaugePoints` sur les temoins exige de cuire des films, donc du Go, interdit sur
  ce creneau. Gates web verts (0 / 0 / 0), `ReplayCanvas.tsx` inchange a 742. Extraction non
  prevue de `zoneStatesPaint.ts` (seuil des 500 lignes franchi par le lot). Mordant prouve par
  double mutation. **STOP — D2 attend le signal du superviseur.**

- 2026-08-26 — **D6.4 CLOSE par lecture pure** (7 artefacts du cache, tableau a l'item) : la
  jauge est publiee sur Bastion (467 a 1 794 points) et vaut ZERO sur les quatre films KOTH. La
  phase D6 est integralement close cote technique.

- 2026-08-26 — **D2 : GATE NON ATTEINT, NEGATIF ECRIT.** Instrument
  `colline_proprietaire_d2_test.go` (garde `ZONE_FILM`, un film par processus, lecture seule,
  aucune base). Les quatre films KOTH mesures. Le CANAL est vivant — designateur elu sur 4/4,
  slot voisin de propriete a 13-213 emissions — mais **l'ORACLE du plan est faux** : le score de
  mode KOTH du film compte des COLLINES GAGNEES (3-2, 4-2 : les scores de l'API), pas des
  secondes de garde, et deux films ne repliquent qu'UN SEUL camp (confirme lecture stricte ET
  brute). Zero film sur quatre a un denominateur exploitable. Le proprietaire de la colline est
  **NON MESURE, pas refute** ; deux pistes de reprise sont ecrites au gate. **STOP — arret
  propre au seuil, aucun contournement, aucun changement de production.**

- 2026-08-26 — **D2-bis : GATE NON ATTEINT sur ses deux clauses, mais le negatif change de
  nature.** Protocole ecrit et commite AVANT la mesure (`6470e7ca1`), instrument
  `colline_proprietaire_d2bis_test.go`, oracle = prises `th=10`, tolerance +/- 20 s derivee de
  l'imprecision documentee du decodeur, temoin de decalage porte a +60 s (20 s tombait DANS la
  fenetre). Les 4 films sont exploitables (31 a 92 prises confrontables) : la regle d'escalade ne
  se declenche pas. Accord >= 90 % sur **0 film sur 4** (89,1 / 88,0 / 87,1 / 71,7 %). MAIS sur
  les deux films LONGS le canal se detache nettement de ses temoins (89,1 % contre 55,6 % ;
  88,0 % contre 56,6 %) — il porte quelque chose, il manque 1 a 2 points. Sur les deux films
  COURTS (13 et 35 emissions) tout se confond. Reserve methodologique ECRITE ET NON CORRIGEE : le
  temoin de permutation elit parfois un slot FRERE du meme objet de mode (`d.slot+2`, le
  capteur), ce qui n'est pas une permutation — signale, pas repare apres coup. **NON MESURE AU
  SEUIL, toujours pas refute.** Reprise : l'oracle du score PERSONNEL (arbitrage superviseur
  requis), ou une decision PRODUIT sur le niveau de preuve exige. **STOP.**

- 2026-08-26 — **D2-ter : GATE NON ATTEINT, et l'oracle est en cause une TROISIEME fois.**
  Protocole commite avant la mesure (`d92c436a2`). Score personnel : 53,3 % et 69,2 % sur les deux
  films les plus fournis, 100 % sur six confrontations du troisieme, un quatrieme non exploitable.
  **La cause est mesuree** : delta dominant MEDIAN de 150 points contre 0-25 pour le camp domine —
  un frag vaut ~100, un tic de colline quelques-uns. L'oracle regarde qui a TUE, pas qui tient la
  colline. L'hypothese d'arbitrage (« 87-89 % = plancher de bruit de l'oracle th=10 ») est donc
  CONTREDITE : l'oracle continu fait nettement moins bien. Le temoin de permutation, une fois les
  slots freres exclus structurellement, s'est revele SANS OBJET. **Trois oracles epuises ; pas de
  D2-quater ; decision produit a l'utilisateur.**

- 2026-08-26 — **D1 FAIT, et il ROUVRE D3 ET D4.** `cmd/zone-attribution -census` (drapeau neuf,
  lecture seule, meme normalisation de mode que le service) recense 1 940 matchs et 45 modes.
  **Total Control : 4 films** (+3 en variante Fiesta) — le seuil de D3 est >= 2, la phase est
  OUVRABLE. **Oddball : 7 films** — le plan n'en connaissait qu'UN, D4 est OUVRABLE. **Stockpile :
  2 films** alors que le report du 2026-08-17 le disait sans corpus exploitable : ce report est a
  AMENDER, il decrivait l'etat du cache a sa date. **Extraction : 2 films** (condition 1 du §2.5
  atteinte ; canal et oracle restent a trouver). **VIP : 3 films en cache, et cela ne change
  rien** — le §2.10 bute sur le film, pas sur le corpus. Sortie figee dans
  `registre_film/D1_recensement_modes.log`.

- 2026-08-26 — **DECISION PRODUIT RENDUE (utilisateur) : option (a).** Le niveau de preuve de
  D2-bis — 88-89 % d'accord contre un temoin a 56 %, canal jamais refute, elu 4 films sur 4 — est
  **ACCEPTE** pour le calque du proprietaire de colline (precedent : la garde de l'ouvrier a
  88 %). L'erreur residuelle est concentree aux BASCULES, dans la fenetre de +/- 20 s de l'oracle.
  D2 passe donc en IMPLEMENTATION, sans bump de schema (la publication attend le bump unique de D5).

- 2026-08-26 — **D2 IMPLEMENTE : le proprietaire de la colline est publie.** Le canal (tag 4 du
  slot voisin du designateur) entre dans `hillStatesOf` : une periode se SUBDIVISE aux
  changements de main, chaque morceau porte son camp, le neutre sort en `Owner` nil. **Aucun bump
  de schema** — `ZoneSpan.Owner` existe deja, seul son CONTENU change sur la voie colline ; la
  publication attend le bump unique de D5. Le niveau de preuve accepte (88-89 % contre temoin
  56 %, decision utilisateur du 26/08, erreur concentree aux bascules) est ecrit EN TETE du
  producteur, avec les trois campagnes — le prochain lecteur le trouve sur place. Deux defauts
  attrapes en cours d'implementation : un fixture de test qui confondait « un seul camp » et
  « une seule emission » (l'election du designateur exige 2 echantillons), et surtout **une
  periode partiellement couverte par le canal perdait sa partie non couverte** — corrige, les
  trous sortent en intervalles ACTIFS sans camp. Mordant par double mutation (camp retire : 3
  echecs ; comblement des trous retire : 1 echec).
