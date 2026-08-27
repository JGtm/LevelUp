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

### 2.4 Oddball — `[!]` LE CRANE : identite ETABLIE, canal ETABLI, ORACLE REFUTE (mesure D4, 2026-08-27)

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
| **verdict** | **MESURE LE 2026-08-27 — seuil (1) TENU, seuil (2) NON TENU.** L'identite est etablie (`0x0017592C`, 4 films sur 4) et le canal se lit (15 a 46 trous fermes par film), mais l'oracle du score personnel NE DISCRIMINE PAS le portage (40,6 a 66,7 % contre un seuil de 90 % ; temoin hors trou a 66,7 et 71,4 % contre 5 %). Oddball reste `[!]`, le crane n'entre PAS au manifeste, D5 ne publie rien. Detail et reserves : journal du plan, entrees D4 du 2026-08-27. |

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

- [x] D3.1 Denombrer, par film de Total Control : slots `ti=13` emetteurs, tags observes, taux de
      chainage (le temoin de largeur du balayage, `ManagedPropertyScan.Chained` vs `Walked`).
      **CORPUS ETABLI (2026-08-26)** — `-census` nomme les films, `-role totalcontrol_zone
      -select-only` les dimensionne : **7 films mesurables**, tous avec leur vivier de zones.

      | film | carte | zones au catalogue | chunks | taille |
      |---|---|---|---|---|
      | `d2c64f8c` | Fortitude | 18 | 59 | 61 Mo |
      | `0862dce4` | Highpower | 13 | 50 | 54 Mo |
      | `66aa5f0b` | Command | 18 | 16 | 15 Mo |
      | `2f05dc98` | Refuge | 15 | — | — |
      | `bf831a6b` | Command | 18 | — | — |
      | `a521164d` | Fragmentation Heavies | 15 | 21 | 27 Mo |
      | `a349fea8` | Fragmentation Heavies | 15 | 51 | 67 Mo |

      Le vivier de 13 a 18 zones par carte est CONFIRME film par film — c'est exactement ce que
      decrit la table du titre, et c'est ce qui rend le seuil (2) discriminant : si
      l'appariement en retenait 13 a 18 au lieu de 3, le calque publierait un mode qui n'existe
      pas. Le seuil de 2 films du plan est tenu trois fois.

      **UN FAIT DE COUT, MESURE ET A RETENIR** : ces films sont des BTB (24 joueurs, cartes
      larges) et pesent 15 a 67 Mo contre 11 a 33 Mo pour les KOTH d'arene. La mesure complete
      d'UN SEUL de ces films par `zone-attribution` — qui construit l'artefact ENTIER avant de
      croiser — depasse dix minutes. Le corpus entier ne se mesure donc pas en avant-plan : il
      faut une fenetre longue, et c'est une contrainte d'ordonnancement, pas un obstacle de
      methode.
- [x] D3.2 Chercher un DESIGNATEUR (H1) par le predicat de production `hillDesignatorOf` : existe-t-il
      un slot de tag 5 chaine dont le voisin porte un proprietaire qui parle ? Combien de bascules,
      et tombent-elles sur les bornes de MANCHE (`objectiveevents.RealRounds` /
      `SeriesByRound`) ?
- [x] D3.3 Appariement (H2) : prises nommees `zone_captures` -> forme du catalogue par la position
      de leur auteur ; taux d'attribution et temoin decale de 12 m. Cardinalite de l'ensemble
      apparie PAR MANCHE.
- [x] D3.4 Proprietaire : accord tag 4 / equipe du capteur, avec ses denominateurs.
- [x] D3.5 Verdict : les trois seuils du §2.3 sont-ils tenus ? Quelle hypothese (H1 ou H2) porte
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

- [x] D4.1 Identite : sur chaque film Oddball, les creations `ti=42` ECARTEES du catalogue d'armes ;
      pour chaque mot de 32 bits distinct, la distance de naissance au plus proche `oddball_spawn`
      du catalogue de carte et l'ecart au plus proche evenement `th=10` de crane. Meme instrument
      de forme que `attachement_phase0_drapeau_test.go`.
- [x] D4.2 Temoin de SELECTIVITE : compter les AUTRES mots qui reunissent les deux conditions.
      Le seuil exige **zero**.
- [x] D4.3 Portage : decouper la vie du crane en TROUS (l'objet cesse d'emettre) ; pour chaque
      trou, chercher le joueur dont le score PERSONNEL s'incremente sur toute sa duree
      (`objectiveevents.SeriesTotal(recs, objectiveevents.PersonalScoreComponent, false)`, slot ->
      xuid par `SlotIdentityResolved`). Temoin : un joueur tire au hasard hors trou.
- [x] D4.4 Verdict : les deux seuils du §2.4. Non tenu = NEGATIF ecrit, `[!]`, la ligne du registre
      des reports est mise a JOUR (texte fourni au CR), et D5 ne publie rien pour Oddball.

**Gate D4** : (1) UN seul mot candidat, le meme sur >= 2 films, 0 autre candidat ; (2) >= 90 % des
trous a porteur unique, temoin <= 5 %. Commandes NUES :

    go build ./...
    go vet ./...
    go test ./internal/analysis/replay/ -run TestEtatVivantOddball -v -timeout 60m
    golangci-lint run --new-from-merge-base=origin/main

### D5 — PUBLICATION : un SEUL bump de schema pour tout ce que D2-D4 ont tenu

Cette phase ne mesure rien : elle publie ce que les gates precedents ont valide, et RIEN d'autre.

- [x] D5.1 Producteurs, un par verdict tenu (KOTH `Owner` sur les intervalles de colline ; Total
      Control : `totalcontrol_zone` dans `heldZoneRoles` + la voie des 3 actives ; Oddball : entree
      `[[objective_objects]]` `family = "ball"` EN+FR + calque de portage du crane). Un verdict non
      tenu n'a AUCUN code.
- [x] D5.2 `Coverage` : chaque calque publie ses DENOMINATEURS et ses rejets par cause. Regle du
      depot : un calque sans couverture se lit comme une exhaustivite ; l'ABSENCE du bloc doit
      rester distincte du zero.
- [x] D5.3 Le TRIPLET de version (§3.2) : `replay.SchemaVersion` (numero libre au moment du lot,
      >= 20) avec sa chronique en tete de `document.go` ET dans le fichier de contrat du calque ;
      `wantReplayDocumentFields` + sa ligne de chronique ; `EXPECTED_REPLAY_SCHEMA_VERSION`.
- [x] D5.4 Contrat client : `go run ./cmd/openapi-gen` (jamais d'edition a la main de
      `api/openapi.yaml`), `make generate-types`, frontiere de nullabilite web
      (`NULLABLE_ARRAYS` / `NULLABLE_ARRAY_PATHS` et `normalizeReplayDocument` — tableaux
      IMBRIQUES compris).
- [x] D5.5 Golden d'assemblage re-congele (`testdata/assembly_000d5950.golden`) et TEMOINS
      re-cuits : les films des gates D2-D4 UNIQUEMENT, **un film par processus**, via
      `cmd/replay-build --map <carte> --facts <faits.json> <matchId>` (aucune base ouverte).
- [x] D5.6 Tests : producteurs testes PURS (sans film) ; un test de non-regression par calque.

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

- 2026-08-26 — **LOT RUNNER : l'executeur canonique « un film = un processus BORNE ».** La mesure
  D3 a sature la machine DE TRAVAIL de l'utilisateur DEUX fois — d'abord 7 films BTB dans un
  processus, puis, apres correction, un film par processus mais SANS plafond ni priorite basse.
  **« Un film = un processus » est necessaire et PAS suffisant.** Nouveau paquet
  `internal/filmproc` : sentinelle memoire a deux plafonds (souple `debug.SetMemoryLimit` + dur
  par echantillonnage), lanceur parent/enfant avec protocole par CODE DE SORTIE, et **priorite
  CPU basse** de l'enfant (`BELOW_NORMAL_PRIORITY_CLASS` sous Windows — le poste de travail est
  la ; sans effet ailleurs, et c'est ecrit comme une decision). Plafond de MESURE a **2 Gio**,
  distinct des 3 Gio des passes de production : une mesure tourne pendant que l'utilisateur
  travaille. `zone-attribution` est cable dessus (`-child`), et sa boucle passe TOUJOURS par
  l'executeur, meme pour un seul film — c'est UN film BTB qui a suffi a prendre la machine.
  **Garde-rail** `archlint/no_unbounded_film_loop_test.go` : tout site d'appel de `BuildMatch` /
  `BuildFromFilm` doit etre DECLARE avec sa justification datee et son REGIME de decodage ;
  l'allowlist se perime toute seule (une entree qui ne designe plus d'appel fait rougir).
  **DECOUVERTE MAJEURE DU GARDE-RAIL** : `internal/sync/replayartifacts.buildAll` enchaine
  jusqu'a 5 films a travers `BuildMatch` DANS LE PROCESSUS DU SERVEUR, sans sentinelle — la
  forme exacte du sinistre du 2026-08-20 (effondrement sur le cinquieme). Local uniquement et
  best-effort, mais non borne en pic. Declare comme DETTE, pas comme exemption : l'executeur ne
  s'y transpose pas tel quel (son arret de processus est interdit la ou des handles d'ecriture
  DuckDB sont tenus, ADR 0013/0019/0030). **Arbitrage superviseur requis.**

- 2026-08-27 — **D3 : GATE NON ATTEINT SUR LE SEUIL (1). Negatif ecrit, arret propre.** Mesure
  des 7 films sous l'executeur borne (protocole commite avant : `025aea2be`).

  | film | carte | posees | dedans | **attribution** | temoin temporel | rapport |
  |---|---|---|---|---|---|---|
  | `bf831a6b` | Command | 12 | 9 | **75,0 %** | 16,7 % | 4,5 |
  | `66aa5f0b` | Command | 11 | 7 | **63,6 %** | 18,2 % | 3,5 |
  | `2f05dc98` | Refuge | 27 | 13 | **48,1 %** | 18,5 % | 2,6 |
  | `0862dce4` | Highpower | 44 | 16 | **36,4 %** | 6,8 % | 5,3 |
  | `d2c64f8c` | Fortitude | 50 | 10 | **20,0 %** | 10,0 % | 2,0 |
  | `a521164d` | Frag. Heavies | 0 | 0 | — | — | NON EXPLOITABLE (aucune prise) |
  | `a349fea8` | Frag. Heavies | — | — | — | — | NON MESURE (plafond memoire) |

  **CORPUS : 55 prises rattachees sur 144 = 38,2 %.** Le seuil (1) exige >= 80 % : **0 film sur
  5 exploitables** l'atteint, le meilleur plafonne a 75,0 % sur douze prises. Le seuil n'est pas
  rebaisse. Par le protocole (ordre par cout, arret au premier echec), les seuils (2)
  cardinalite et (3) proprietaire **ne sont PAS evalues** — les deux se lisent sur des zones
  appariees, et l'appariement ne tient pas.

  **CE N'EST PAS UN CORPUS ABSENT, C'EST UN APPARIEMENT QUI NE TIENT PAS.** Cinq films
  exploitables (11 a 50 prises), la regle d'escalade ne se declenche pas. Et le canal n'est pas
  muet : les temoins temporels restent bas (6,8 a 18,5 %) et le rapport reel/temoin vaut 2,0 a
  5,3 — il y a du signal, il est simplement tres loin du niveau qu'une publication exige.

  **CAUSE PROBABLE, NON INSTRUITE (le protocole arrete au seuil)** : les prises `zone_captures`
  sont approximatives (`th=10`, 5-20 s) et Total Control est du BTB — 24 joueurs, cartes larges,
  13 a 18 zones. A l'instant enregistre, le capteur est souvent deja sorti de la zone. Les
  compteurs le suggerent : sur Fortitude, 16 prises sans position et 13 ambigues sur 66
  identifiees. Instruire cela demanderait un oracle plus fin que le `th=10`, donc une phase
  a part.

  **`totalcontrol_zone` NE REJOINT PAS `heldZoneRoles`.** L'entree du titre reste `neutral = true`
  en formes seules. Rien n'est publie.

- 2026-08-27 — **L'EXECUTEUR BORNE A ATTRAPE UNE VRAIE BOMBE, et c'est la validation du lot
  RUNNER sur pieces.** `a349fea8` (Fragmentation Heavies, 51 chunks, 67 Mo) alloue **3,17 Gio en
  3,6 secondes** et franchit le plafond dur : l'enfant meurt avec son motif, le parent le compte
  en `memoire`, **et la passe continue**. Les six autres films culminent entre **0,07 et
  0,22 Gio**. Deux enseignements : (a) la machine ne mourait pas d'un film moyen mais de la
  SOMME des pics dans un processus unique — d'ou l'accumulation ; (b) le corpus contient bel et
  bien au moins un film-bombe, celui-la meme qui a fait suffoquer la machine deux fois. Le
  recap le DIT (« 1 film n'a PAS ete mesure — un taux calcule dessus ne porte pas sur ce qu'il
  annonce »), ce qui est exactement pourquoi ce compteur existe.

### D3-bis — TOTAL CONTROL : le SEUIL (2) SEUL, mesure reduite avant fermeture

> **PROTOCOLE ECRIT ET COMMITE AVANT LA MESURE.** Ce commit ne contient aucun chiffre de
> resultat. Arbitrage superviseur du 2026-08-27 : la possession vivante TC est `[!]` (pas de
> 4e oracle — lecon KOTH), mais le seuil (2) reste DU parce qu'il ne depend PAS de l'attribution
> des prises qui a coule le seuil (1).

**POURQUOI CE SEUIL SURVIT AU NAUFRAGE DU PREMIER.** Le seuil (1) mesurait « dans quelle ZONE se
tenait le capteur » — de la geometrie, sur des instants approximatifs. Le seuil (2) ne demande
aucune position : il compte des IDENTITES designees par le film. Les deux echouent ou reussissent
pour des raisons independantes.

**CE QUI EST TESTE, EN UNE PHRASE** : par MANCHE, le film designe-t-il exactement TROIS zones ?

- [x] D3b.1 Les DESIGNATEURS : les slots `ti=13` portant une serie de tag 5 CHAINEE (meme
      predicat de chainage que le volet colline — le tag 5 non chaine est de la contamination
      d'ancrage). Denombrer les slots, leurs valeurs distinctes, leurs bascules.
- [x] D3b.2 Les MANCHES : `objectiveevents.RealRounds` sur les enregistrements d'entite. Un
      film sans manche lisible n'est pas exploitable, et le dit.
- [x] D3b.3 Par manche, l'ENSEMBLE DESIGNE : les valeurs de tag 5 DISTINCTES en vigueur pendant
      la manche. Le verdict porte sur son CARDINAL.
- [x] D3b.4 Verdict et fermeture.

**« EXACTEMENT 3 », DEFINI AVANT DE COMPTER.** Le cardinal de l'ensemble designe d'une manche
vaut **3**. Ni « au moins 3 » (14 zones designees contiendraient 3 et passeraient), ni « 3 en
moyenne » (une moyenne masque une manche a 1 et une a 5).

**TOLERANCE SUR LES ROTATIONS DE MANCHE : +/- 2 s AUTOUR DE CHAQUE BORNE, EXCLUES DES DEUX
MANCHES.** A la bascule, le jeu retire trois zones et en pose trois autres ; les emissions de
cette fenetre appartiennent a la rotation elle-meme, pas a l'une des deux manches. Les compter
ferait mecaniquement six zones sur toute manche — un faux negatif garanti. Deux secondes est
l'ordre de grandeur mesure des bascules de designateur en KOTH (13 a 21 ms apres la capture,
volet 1) elargi d'un facteur de securite : on exclut large plutot que de trancher fin sur une
grandeur qu'on n'a pas mesuree ici.

**SEUIL** : cardinal == 3 sur **>= 80 %** des manches exploitables, et sur **>= 2 films**.
Une manche est exploitable si elle porte **>= 1** emission de designateur hors fenetre de
rotation.

**ESCALADE, ECRITE D'AVANCE** : un film sans designateur elu, ou sans manche lisible, n'est PAS
exploitable et ne compte ni pour ni contre. **`a349fea8` est EXCLU D'OFFICE** — bombe memoire
connue (3,17 Gio en 3,6 s), on ne la relance pas pour un comptage. S'il reste **moins de 2**
films exploitables, la mesure s'arrete : **TC entierement `[!]`**.

**LES DEUX ISSUES, ET ELLES SONT TOUTES DEUX DES RESULTATS** :
- **OUI** -> publication **ACTIVES SEULEMENT** : le calque TC sert les 3 zones designees du
  match, NEUTRES et sans possession (le seuil (3) n'a jamais ete evalue — on ne publie pas un
  proprietaire). Cela repond a la decision produit en attente : un rejeu TC cesse de dessiner
  13 a 18 formes la ou 3 sont actives.
- **NON** -> TC entierement `[!]`, et l'affichage du vivier repart en DECISION PRODUIT.

**Gate D3-bis** — commandes NUES, un film par processus, sous l'executeur borne :

    go vet ./internal/analysis/replay/
    ZONE_FILM=<chunks d'un film> go test ./internal/analysis/replay/ -run TotalControlDesignateur -v -timeout 60m
    go test ./internal/analysis/replay/...

- 2026-08-27 — **D3-bis : GATE NON ATTEINT (0/6). TOTAL CONTROL PASSE `[!]` EN ENTIER — mais le
  designateur n'est PAS refute pour autant, et la difference compte.**

  | film | manches reelles | slots « designateur » | zones designees | chainage `ti=13` |
  |---|---|---|---|---|
  | `bf831a6b` | 1 | 3 | **1** | 9 771 / 78 529 = 12,4 % |
  | `66aa5f0b` | 1 | 4 | **1** | 6 486 / 44 849 = 14,5 % |
  | `2f05dc98` | 1 | 18 | **19** | 77 355 / 700 578 = 11,0 % |
  | `d2c64f8c` | 1 | 24 | **23** | 79 725 / 1 002 091 = 8,0 % |
  | `a521164d` | 1 | 40 | **54** | 10 043 / 189 128 = 5,3 % |
  | `0862dce4` | 1 | 79 | **119** | 79 741 / 895 626 = 8,9 % |

  **Aucun film ne rend 3.** Seuil : cardinal 3 sur >= 80 % des manches, >= 2 films — obtenu
  **0,0 % partout**. Par l'arbitrage du 2026-08-27, branche NON : **TC entierement `[!]`**,
  `totalcontrol_zone` ne rejoint pas `heldZoneRoles`, et **l'affichage du vivier de 13-18 formes
  repart en DECISION PRODUIT**.

  **DEUX RESERVES QUI INTERDISENT DE LIRE CECI COMME UNE REFUTATION DU DESIGNATEUR**, et il faut
  les ecrire plutot que d'encaisser un negatif trop propre :

  1. **IL N'Y A QU'UNE MANCHE PAR FILM — la premisse du protocole est FAUSSE sur ce corpus.**
     `RealRounds` rend 1 sur les six. L'ensemble designe est donc pris sur TOUT LE MATCH, et la
     marge de rotation de +/- 2 s ne protege RIEN (il n'y a pas de borne interieure). Un
     designateur PARFAIT qui ferait tourner son trio N fois dans le match rendrait 3N valeurs,
     pas 3. La mesure, telle qu'ecrite, ne peut donc pas distinguer « designateur casse » de
     « designateur sain qui tourne » — et son unite, la manche, n'existe pas ici.
  2. **L'ANCRAGE `ti=13` EST MAUVAIS SUR CES FILMS** : 5,3 a 14,5 % de chainage, contre 87 a
     99 % mesures sur le corpus KOTH. Et le nombre de « slots designateurs » CROIT AVEC LA
     TAILLE DU FILM (3 et 4 sur les deux plus legers, 79 sur le plus lourd) — signature d'une
     contamination d'ancrage, pas d'une structure fixe d'objet de mode.

  **CE QUE CELA VAUT QUAND MEME** : les deux films les plus legers, ceux dont l'ancrage est le
  moins mauvais, rendent 3 et 4 slots et **UNE seule** valeur designee sur tout le match. C'est
  compatible avec un objet de mode unique — mais 1 n'est pas 3, et je n'en tire rien de plus.

  **AUCUNE REMESURE N'EST ENGAGEE** : elle exigerait un protocole neuf (unite = la ROTATION et
  non la manche, plus un ancrage `ti=13` fiable sur BTB), c'est-a-dire la chasse au protocole
  suivant que l'arbitrage du 2026-08-27 ferme explicitement. Consigne, et laisse au superviseur.

### D4 — ODDBALL : le PROTOCOLE, ecrit et commite AVANT la mesure

> **PROTOCOLE ECRIT ET COMMITE AVANT LA MESURE.** Ce commit ne contient aucun chiffre de
> RESULTAT. Les chiffres qu'il porte sont des DENOMINATEURS de corpus (combien de films, quelles
> cartes, quels catalogues les couvrent) : ils disent sur quoi la mesure va porter, pas ce
> qu'elle va rendre. Les seuils sont ceux du §2.4 et ne bougent pas.

**LE CORPUS EST DIMENSIONNE D'ABORD, ET IL SE REDUIT DE SEPT A QUATRE.** Le recensement D1
nommait 7 films Oddball. La recette d'identite du §2.4 exige DEUX catalogues par film : les
bornes de quantification de sa carte (sans elles un objet du monde ne rend que des quanta, et
« a moins de 3 m d'un socle » n'a aucun sens) et les objectifs ponctuels de sa carte (sans eux
il n'y a pas de socle `oddball_spawn` a mesurer). Releve AVANT toute mesure :

| film | carte | bornes de quantification | `oddball_spawn` au catalogue |
|---|---|---|---|
| `24dbb67d` | Recharge - Ranked | OUI | 1 |
| `43716616` | Smallhalla | OUI | 1 |
| `51ebbc0f` | Banished Narrows | OUI | 1 |
| `d9781168` | Dredge | OUI | 1 |
| `60ae07c4` | Live Fire - Ranked | **NON** (carte absente) | 1 |
| `c88ec007` | Live Fire | **NON** (carte absente) | 1 |
| `92f18088` | Lattice - Ranked | OUI | **carte absente du catalogue d'objectifs** |

**QUATRE films mesurables**, et c'est >= 2 : le seuil d'identite reste atteignable. Les trois
autres sont ecartes pour DEFAUT DE CATALOGUE, pas pour defaut de film — ils ne comptent ni pour
ni contre, et la cause est nommee. La bombe memoire connue `a349fea8` n'est PAS au corpus
Oddball : rien a exclure de ce cote. Chaque carte mesurable porte **exactement UN**
`oddball_spawn` — le socle est unique, ce qui rend « naitre au socle » lisible sans ambiguite.

- [x] D4.0 **CONTROLE D'ALIGNEMENT DE LECTURE, avant tout le reste.** Le mot MPP de 32 bits se
      lit derriere deux champs de largeur VARIABLE par film (decouverte 8 des armes au sol :
      9/5 en Quick Play, 8/3 sur les films BTB). L'instrument du drapeau lit aux largeurs PAR
      DEFAUT. **Le temoin est le compte de creations RESOLUES au catalogue d'armes** : un film
      ou ce compte est nul est un film lu aux mauvaises largeurs, et il ne compte NI POUR NI
      CONTRE. Sans ce controle, un film mal decoupe rendrait « aucun mot candidat » et ferait
      passer une panne de lecture pour une refutation.
- [x] D4.1 **Identite.** Creations `ti=42` ECARTEES du catalogue d'armes ; par mot de 32 bits
      distinct, (a) le nombre de creations nees a <= 3 m du `oddball_spawn` de la carte et (b)
      l'ecart temporel minimal a un evenement `th=10` de crane. Meme instrument de forme que
      `attachement_phase0_drapeau_test.go` — la tolerance de 3 m est celle, deja employee, de la
      chaine des poses.
- [x] D4.2 **Temoin de SELECTIVITE.** Compter les mots ecartes qui reunissent les DEUX
      conditions (naissance a <= 3 m d'un `oddball_spawn` ET coincidence a <= 1 s d'un
      evenement `th=10` de crane). Le seuil exige que ce compte vaille **UN** : le candidat, et
      aucun autre.
- [x] D4.3 **Portage.** Vies libres du mot candidat (`flagFreeLives`, meme regle d'appariement
      creation -> piste que les armes au sol) ; un TROU est l'intervalle entre la fin d'une vie
      libre et le debut de la suivante. Pour chaque trou, chercher le joueur dont le score
      PERSONNEL s'incremente sur toute sa duree
      (`objectiveevents.SeriesTotal(recs, PersonalScoreComponent, false)`, slot -> xuid par
      `SlotIdentityResolved` / `SlotIdentityByDeaths`).
- [x] D4.4 **Verdict.** Les deux seuils du §2.4. Non tenu = NEGATIF ecrit, `[!]`, ligne du
      registre des reports mise a JOUR (texte fourni au CR), et D5 ne publie rien pour Oddball.

**L'ORACLE EST MESURE AVANT D'ETRE CRU — c'est la reserve du superviseur, et elle est fondee.**
En D2-ter le score personnel s'est revele DOMINE PAR LES FRAGS (delta dominant median ~150 la ou
un tic de colline en vaut quelques-uns), ce qui a coule l'oracle continu sur la colline. En
Oddball le score personnel EST cense etre du portage (~1 point/s tenu), donc le diagnostic ne se
transpose PAS tel quel — mais il ne s'ecarte pas non plus sur parole. **Le meme DIAGNOSTIC est
donc publie A COTE du verdict, et avant lui** : l'AMPLEUR des deltas de score personnel par trou.
Un delta median qui se compte en centaines dit que l'oracle mesure des frags ; un delta median
proportionnel a la DUREE du trou dit qu'il mesure du portage. Si le diagnostic dit « frags »,
le seuil (2) n'est pas evalue et Oddball passe `[!]` ORACLE — refuter avec un oracle dont on
vient de montrer qu'il mesure autre chose ne prouverait rien.

**SEUILS (ceux du §2.4, RECOPIES SANS MODIFICATION)** :

1. **Identite** : UN SEUL mot candidat, LE MEME sur **>= 2** films mesurables, temoin de
   selectivite = **0** autre candidat.
2. **Portage** : **>= 90 %** des trous ont EXACTEMENT UN joueur dont le score personnel
   s'incremente sur toute leur duree ; temoin « porteur tire au hasard hors trou » **<= 5 %**.

**LE TEMOIN DE PORTAGE, DEFINI AVANT DE COMPTER** : pour chaque trou, un joueur tire parmi ceux
que le pont nomme, a l'EXCLUSION du porteur retenu, teste par le MEME predicat sur un intervalle
de MEME DUREE place hors de tout trou. Meme code, meme duree, meme predicat — sans quoi le temoin
controlerait autre chose que la mesure.

**ESCALADE, ECRITE D'AVANCE** : un film dont le controle D4.0 echoue, ou dont le pont ne nomme
pas au moins deux joueurs, n'est pas exploitable et ne compte ni pour ni contre. S'il reste
**moins de 2** films exploitables, le seuil (1) est repute NON TENU et la mesure s'arrete :
Oddball entierement `[!]` CORPUS. Si le seuil (1) tombe, le seuil (2) n'est PAS evalue — sans
identite, il n'y a pas d'objet dont decouper les trous.

**LES DEUX ISSUES SONT DES RESULTATS.** OUI aux deux seuils => le crane entre dans
`[[objective_objects]]` de `replay_labels.toml` avec sa justification DATEE (mandat du
superviseur du 2026-08-27), et D5 publie son portage. NON => negatif ecrit, `[!]` MESURE, ligne
du registre des reports mise a jour et non contournee, D5 ne publie rien pour Oddball.

**EXECUTION** : un film par processus sous `filmproc` (plafond memoire, priorite basse), aucune
base ouverte en ecriture, aucune re-cuisson d'artefact. Le film `24dbb67d` est TRONQUE (29
chunks, `PLAN_REMEDIATION_CACHE.md`) : s'il rend un pont degrade, il sort par l'escalade
ci-dessus comme n'importe quel autre, sans traitement de faveur.

- 2026-08-27 — **D4, SEUIL (1) : L'IDENTITE DU CRANE EST TENUE — `0x0017592C`.** Protocole
  commite avant la mesure (`1a29fe756`). Quatre films mesurables, un par processus, sentinelle
  memoire armee DANS le processus qui decode (pic observe 0,01 Gio sur les quatre — ces films
  d'arene ne sont pas des bombes). Sortie brute figee dans
  `registre_film/D4_oddball_identite.log`.

  | film | resolues au catalogue d'armes (controle D4.0) | ecartees / mots distincts | evenements `th=10` | mot elu | creations | au socle (min) | ecart min | AUTRES candidats |
  |---|---|---|---|---|---|---|---|---|
  | `24dbb67d` | 136 | 133 / 69 | 87 | **`0x0017592C`** | 23 | 3 (0,0 m) | **6 ms** | 1 (`0xCBA072DC`) |
  | `43716616` | 195 | 137 / 107 | 60 | **`0x0017592C`** | 16 | 4 (0,0 m) | **3 ms** | **0** |
  | `51ebbc0f` | 206 | 144 / 82 | 75 | **`0x0017592C`** | 21 | 4 (0,0 m) | **3 ms** | **0** |
  | `d9781168` | 409 | 256 / 199 | 124 | **`0x0017592C`** | 47 | 13 (0,0 m) | **5 ms** | 1 (`0x042100A6`) |

  **LE CONTROLE D'ALIGNEMENT D4.0 PASSE SUR LES QUATRE** (136 a 409 creations resolues au
  catalogue d'armes) : le bloc MPP est lu aux bonnes largeurs, aucun film n'est ecarte pour
  panne de lecture. **LE MEME MOT EST ELU SUR 4 FILMS SUR 4**, toujours en tete par le nombre de
  creations, toujours ne A 0,0 m du socle unique `oddball_spawn`, toujours a 3-6 ms d'un
  evenement `th=10` de crane. Le seuil exigeait « le meme sur >= 2 films » : il est tenu avec le
  double.

  **LE TEMOIN DE SELECTIVITE EST TENU SUR DEUX FILMS, ET REFUTE SUR DEUX — ecrit tel quel.** Le
  seuil demande « 0 autre candidat » ; `43716616` et `51ebbc0f` rendent exactement cela, et ces
  deux films suffisent au « >= 2 films » du protocole. Les deux autres portent CHACUN un second
  mot, et **ce n'est pas le meme des deux cotes** (`0xCBA072DC` sur l'un, `0x042100A6` sur
  l'autre) : ce n'est donc pas une identite rivale, c'est du bruit de singleton. Les deux se
  separent du mot elu sur les DEUX criteres a la fois — 1 seule creation contre 16 a 47, ne a
  1,7 et 2,9 m contre 0,0 m, coincidant a 157 et 177 ms contre 3 a 6 ms. **Le seuil n'est pas
  abaisse pour les absorber** : il est tenu par les deux films qui le tiennent litteralement, et
  les deux parasites sont publies avec leurs chiffres plutot que gommes.

  **CE QUE LA MESURE NE PROUVE PAS ENCORE** : que `0x0017592C` soit le crane PLUTOT QU'UN autre
  objet d'objectif ne au meme socle. La falsification n'est pas a chercher ailleurs — c'est le
  seuil (2) : si ce mot n'est pas le crane, ses trous de replication ne coincideront pas avec le
  portage. La mesure suivante est donc aussi le controle de celle-ci.

#### D4.3 — LE SEUIL (2) : les definitions operatoires, ECRITES AVANT LA MESURE

> Aucun chiffre de resultat dans cette section. Le seuil (2) du §2.4 dit « **>= 90 %** des trous
> ont EXACTEMENT UN joueur dont le score personnel s'incremente sur toute leur duree, temoin
> **<= 5 %** ». « S'incremente sur toute sa duree » et « hors trou » sont des phrases, pas des
> predicats : les voici rendus operatoires avant d'etre appliques.

**LE TROU.** Les vies LIBRES du mot `0x0017592C` se construisent par `flagFreeLives` — la MEME
regle d'appariement creation -> piste que les armes au sol, deja en production. Un TROU est
l'intervalle entre la fin d'une vie libre et le debut de la suivante : l'objet a cesse de
repliquer sa position, donc quelqu'un le porte (principe etabli, `flag_objects.go` § « Le
principe »). L'intervalle AVANT la premiere vie et celui APRES la derniere ne sont pas des
trous : rien ne les ferme, et un intervalle ouvert n'a pas de duree.

**LES DEUX HORLOGES.** Les vies sont datees en microsecondes MOTEUR, les emissions de score en
millisecondes depuis le PREMIER PAQUET du film. La conversion est celle du seuil (1),
`matchMS = (moteurUS - origineUS) / 1000`, et elle n'est plus une hypothese : le seuil (1) vient
de faire coincider des creations converties par cette formule avec des evenements `th=10` a
3-6 ms. C'est une VALIDATION de la conversion, obtenue en chemin.

**« S'INCREMENTE SUR TOUTE SA DUREE », RENDU OPERATOIRE.** Le trou est decoupe en TRANCHES
consecutives de `d4TrancheMS = 5000` ms ; la derniere tranche incomplete est fusionnee a la
precedente, de sorte que toute tranche dure au moins 5 s. Un slot QUALIFIE si son score
personnel cumule croit STRICTEMENT dans CHAQUE tranche. Un slot qui gagne cent points au debut
puis plus rien ne qualifie pas — c'est exactement ce que « sur toute sa duree » doit exclure, et
c'est le piege que D2-ter a paye au prix fort.

**LA TRANCHE VAUT 5 s, ET LA VALEUR SE JUSTIFIE** : le score d'Oddball monte a ~1 Hz pendant le
portage, donc 5 s laissent attendre environ cinq increments — assez pour qu'une absence
d'increment signifie quelque chose, assez peu pour qu'un trou de 20 s porte quatre verdicts
independants. Un trou de moins de 5 s n'a pas une seule tranche pleine : il n'est PAS
exploitable et ne compte ni pour ni contre.

**EXACTEMENT UN.** Le trou est REUSSI si le nombre de slots qualifiants vaut exactement 1. Zero
(personne ne marque) et deux ou plus (plusieurs marquent en continu) sont l'un et l'autre des
echecs, et ils se comptent SEPAREMENT : ils ne disent pas la meme chose sur l'oracle.

**LE TEMOIN, DEFINI AVANT DE COMPTER.** Pour chaque trou exploitable : un intervalle de MEME
DUREE, place entierement HORS de tout trou (donc a l'interieur d'une vie libre, la ou l'objet
replique et n'est donc porte par personne), soumis au MEME predicat. Le meme code, la meme
duree, le meme decoupage en tranches. L'intervalle temoin est choisi de facon DETERMINISTE (le
premier intervalle libre assez long rencontre a partir d'un rang tire par un generateur de
graine FIXE, `d4GraineTemoin`), pour que deux executions rendent la meme sortie. Un trou dont
aucun intervalle libre de meme duree n'existe n'a pas de temoin, et cela se DIT — le
denominateur du temoin est publie a part de celui de la mesure.

**LE DIAGNOSTIC DE L'ORACLE EST PUBLIE AVANT LE VERDICT** (reserve du superviseur, §D4). Pour
chaque trou reussi, le delta de score personnel du porteur retenu, rapporte a la DUREE du trou :
des points par seconde. Un oracle qui mesure du PORTAGE rend une valeur proche de 1 pt/s et
proportionnelle a la duree ; un oracle domine par les FRAGS rend des sauts de l'ordre de la
centaine, sans rapport avec la duree. Publier la mediane des points par seconde ET la mediane du
delta brut separe les deux cas sans avoir a en prejuger. **Si le diagnostic dit « frags », le
seuil (2) n'est pas evalue et Oddball passe `[!]` ORACLE** — le protocole du §D4 le dit deja, et
il ne se renegocie pas a la lecture du chiffre.

- 2026-08-27 — **D4, SEUIL (2) : NON TENU SUR LES QUATRE FILMS. Oddball passe `[!]`, et le
  crane N'ENTRE PAS au manifeste.** Definitions operatoires commitees avant la mesure
  (`f51c7fd5d`). Sortie brute figee dans `registre_film/D4_oddball_portage.log`.

  | film | vies libres | trous fermes | exploitables (>= 5 s) | porteur UNIQUE | aucun | plusieurs | part | temoin INTERVALLE | temoin JOUEUR | diagnostic |
  |---|---|---|---|---|---|---|---|---|---|---|
  | `24dbb67d` | 23 | 22 | 15 | 10 | 3 | 2 | **66,7 %** | 2/3 = 66,7 % | 1/3 = 33,3 % | 9,95 pt/s, delta 360 |
  | `43716616` | 16 | 15 | 11 | 7 | 2 | 2 | **63,6 %** | pas de denominateur | pas de denominateur | 8,98 pt/s, delta 160 |
  | `51ebbc0f` | 21 | 20 | 13 | 6 | 4 | 3 | **46,2 %** | 0/2 = 0,0 % | 0/2 = 0,0 % | 10,58 pt/s, delta 260 |
  | `d9781168` | 47 | 46 | 32 | 13 | 8 | 11 | **40,6 %** | 5/7 = 71,4 % | 1/7 = 14,3 % | 9,27 pt/s, delta 160 |

  **Seuil : >= 90 %. Mesure : 40,6 a 66,7 %. NON TENU, sur les quatre films, sans exception.**
  Ce verdict ne depend d'AUCUN temoin : la part seule le rend.

  **LE TEMOIN, QUAND IL A UN DENOMINATEUR, EST ACCABLANT.** Sur `d9781168` et `24dbb67d`, un
  intervalle de MEME DUREE place a l'interieur d'une vie libre — c'est-a-dire quand l'objet
  REPLIQUE, donc quand personne ne le porte — rend un marqueur continu unique dans **71,4 %** et
  **66,7 %** des cas, contre un seuil de 5 %. Le predicat n'est pas specifique aux trous : il
  attrape a peu pres autant de monde hors portage que pendant. **Denominateurs minuscules (7, 3,
  2, et 0 sur un film), et c'est dit** — trop peu de vies libres durent aussi longtemps que les
  trous. Le temoin CORROBORE, il ne porte pas le verdict a lui seul.

  **LE DIAGNOSTIC DE L'ORACLE, PUBLIE AVANT LE VERDICT COMME PROMIS : 9 a 10,6 points par
  seconde de trou** (deltas bruts de 160 a 360 sur des trous de 17 a 36 s). Le protocole du §2.4
  posait « ~1 pt/s pendant le portage ». **La mesure est dix fois au-dessus.**

  **CE QUE JE NE CONCLURAI PAS.** Je n'ecris PAS « donc c'est du frag », comme en D2-ter. Les
  deux lectures restent ouvertes et ce corpus ne les separe pas : soit le score personnel
  d'Oddball vaut reellement ~10 pt/s de portage (et le « ~1 Hz » du §2.4 etait une supposition de
  plan, jamais mesuree — elle l'est desormais, et elle est fausse), soit il melange le portage a
  tout le reste. Le temoin penche pour la seconde, mais avec 7 essais on ne tranche pas. **CE QUI
  EST ETABLI, c'est que l'oracle ne DISCRIMINE pas** : il ne distingue pas « quelqu'un porte » de
  « personne ne porte ». C'est suffisant pour le refuser, et insuffisant pour le nommer.

  **UNE LIMITE DE LA MESURE QUI M'EST PROPRE, ET QUI COMPTE.** Mon « trou » est l'intervalle
  entre deux vies libres. Il confond DEUX choses que le protocole supposait identiques : (a)
  quelqu'un porte l'objet, et (b) l'objet a ete rendu / reinitialise et re-cree a son socle. Le
  §2.4 tenait (b) pour negligeable ; rien ne l'a verifie, et 23 a 47 vies libres par match
  suggerent des re-creations frequentes. Une partie des trous « rates » peut n'etre pas des
  portages du tout. **Cela n'annule pas le negatif** (le seuil est rate de 25 a 50 points, pas de
  deux), mais cela interdit d'imputer l'echec au seul oracle.

- [x] D4.3 Portage : mesure faite sur les 4 films exploitables, `[!]` — seuil rate.
- [x] D4.4 Verdict : **les deux seuils du §2.4 ne sont PAS tous deux tenus** — (1) TENU,
      (2) NON TENU. Oddball reste `[!]`. Le crane N'ENTRE PAS dans `[[objective_objects]]` :
      le mandat d'amender le manifeste etait conditionne au gate, et le gate est le COUPLE
      des deux seuils. **D5 ne publie rien pour Oddball.**

**CE QUE D4 LAISSE D'ACQUIS MALGRE LE NEGATIF** — et ce n'est pas rien, parce que le report du
2026-08-18 disait « ni canal ni oracle » :

1. **L'IDENTITE DU CRANE EST ETABLIE** : `0x0017592C`, elu sur 4 films sur 4, ne a 0,0 m du socle,
   a 3-6 ms d'un evenement `th=10`. Le report ne peut plus dire « aucun candidat unique ».
2. **LE CANAL EXISTE ET SE LIT** : 16 a 47 vies libres par film, 15 a 46 trous fermes. Ce qui
   manque n'est PAS le canal.
3. **L'ORACLE, LUI, EST REFUTE** : le score personnel ne discrimine pas le portage. La condition
   de reprise du registre nommait « (a) le SCORE PAR SECONDE DE PORTAGE » — c'est exactement ce
   qui vient d'etre essaye, et ce qui vient d'echouer. **Cette condition de reprise est donc
   CONSOMMEE et doit etre REECRITE**, pas laissee en l'etat.
4. **UNE DECISION PRODUIT DISTINCTE RESTE OUVERTE, ET ELLE N'EST PAS COUVERTE PAR CE GATE** :
   publier les vies LIBRES du crane seules — ou l'objet se trouve quand personne ne le porte —
   ne demande aucun oracle, seulement l'identite (acquise) et le canal (acquis). Ce serait un
   calque d'OBJET, pas un calque de PORTAGE. Le gate D4 ne l'autorise pas et je ne l'ai pas
   fait ; il revient au superviseur de decider s'il ouvre ce chantier a part.

- 2026-08-27 — **`pair_name` AMPUTE : LA DONNEE STOCKEE EST DEJA TRONQUEE. Le normaliseur est
  HORS DE CAUSE, FIX-NORMALIZE-S est SANS OBJET.** Lecture seule (`cmd/diag_q`, CLI existante,
  `access_mode=read_only`). Match `007d53a4-1469-4174-be6b-3303c8e7bf36` :

      pair_name          « urvive The Undead 3.0 on TFF | Night Of The Undead »   (50 car.)
      game_variant_name  « TFF | Survive The Undead »                             INTACT
      map_name           « TFF | Night Of The Undead »                            INTACT

  **La chaine de preuve, maillon par maillon :**

  1. Le `pair_name` du registre est ampute A LA SOURCE — le « S » de « Survive » manque DEJA en
     base. `NormalizeModeLabel` n'a rien mange : il recoit une chaine deja coupee. **Ma reserve
     du 2026-08-26 — « aucune branche de cette fonction ne retire un caractere de tete » — est
     confirmee sur pieces.**
  2. Ce n'est pas non plus `constructPairName` (`sync/enrich_registry.go:103`) : il fabrique
     `gv + " on " + mp`, ce qui rendrait « TFF | Survive The Undead on TFF | Night Of The
     Undead » — avec le prefixe et sans le « 3.0 ». Il ne s'est de toute facon pas declenche, sa
     garde exigeant un `pair_name` egal au GUID.
  3. La valeur vient de `asset_translations` (metadata.duckdb, asset `53a7f98d-...`, type
     `pair`), et elle y est **IDENTIQUEMENT amputee dans les SEPT langues** (en-US comprise),
     toutes cuites le 2026-03-30. Une amputation qui traverse toutes les locales a l'identique
     n'est pas un accident de rendu : c'est la valeur recue.
  4. **Le corpus donne le plafond : sur 503 paires en-US, `max(length(name))` vaut 50, ZERO
     au-dela, quatre exactement a 50.** Les autres types plafonnent bien en dessous (playlist 37,
     game_variant 36, map 25) — ils ne touchent jamais la borne. **Un plafond dur de 50
     caracteres existe donc sur le nom de paire, et il garde la QUEUE, pas la tete.**
  5. Les DEUX seuls noms visiblement amputes sont ceux qui depassaient 50 : « urvive The Undead
     3.0 on ... » (57 car. attendus, 7 coupes en tete) et « ght:Heroic King of the Hill on
     Vallaheim Firefight » (56 attendus, 6 coupes). Les deux autres noms a 50 (« BTB
     Heavies:Total Control on Fragmentation Heavies ») sont COMPLETS et coherents : la
     coincidence a 50 n'en est pas une.
  6. Cote depot, **rien ne tronque** : `cmd/levelup/cmd_populate_assets.go:373` stocke
     `asset.PublicName` verbatim, la colonne est un `VARCHAR` sans borne, et aucun `[:50]`
     n'existe dans `internal/`.

  **CE QUI EST ETABLI** : la troncature est en amont de tout ce que ce lot touche, et le
  normaliseur est innocent. **CE QUI NE L'EST PAS** : si le plafond de 50 est celui de l'API Halo
  ou celui de notre couche de fetch, je ne l'ai pas tranche — il faudrait lire une reponse
  vivante, ce qui sort du perimetre hors ligne.

  **CIBLE DU FIX, CONSIGNEE ET NON TRAITEE (chantier DONNEES, hors de ce lot)** : la piece de
  reparation existe deja et n'a qu'une garde trop etroite. `constructPairName` reconstruit un nom
  propre a partir des deux champs INTACTS ; il ne se declenche que si `pair_name` vaut `pair_id`.
  Elargir sa garde au cas « `pair_name` ampute » (par exemple : le nom ne se termine pas par
  `map_name`, ou il ne commence pas par `game_variant_name`) reparerait les deux lignes, sans
  toucher au normaliseur. **Non fait : decision et lot du superviseur.**

### D5 — INVENTAIRE DE PUBLICATION (etat au 2026-08-27, AVANT les deux decisions en vol)

> D5 ne publie rien tant que les deux decisions utilisateur ne sont pas rendues. Cet inventaire
> dit exactement ce qui entre selon chaque issue, pour que la decision se prenne sur la liste.

**LE TRIPLET DE VERSION, RE-VERIFIE SUR PIECES CE JOUR** (les trois fichiers rouverts, pas de
memoire) :

| piece | fichier | valeur actuelle |
|---|---|---|
| `replay.SchemaVersion` | `internal/analysis/replay/document.go:173` | **20** |
| `wantReplayDocumentFields` | `contracttest/replay_contract_test.go:331` | **38** |
| `EXPECTED_REPLAY_SCHEMA_VERSION` | `apps/web/src/features/match-replay/replaySchemaLogic.ts:32` | **20** |

**LE PROCHAIN NUMERO LIBRE EST DONC 21**, et c'est le SEUL bump du lot, quelles que soient les
deux decisions.

**A — ACQUIS, N'ATTEND AUCUNE DECISION.**

- **Proprietaire de la colline KOTH.** Deja IMPLEMENTE (`zone_states_hill.go`, `hillStatesOf`
  publie `Owner`, periodes subdivisees aux changements de main). **Aucune cle nouvelle** :
  `ZoneSpan.Owner` existe deja, seul son CONTENU change sur la voie colline — donc
  `wantReplayDocumentFields` reste a **38**. Le bump n'est PAS reclame par la forme, il l'est par
  la REPRISE DU BACKFILL : un artefact 20 porte un `Owner` de colline toujours nul et doit se
  lire « a re-cuire », pas « a jour ». Niveau de preuve accepte : decision utilisateur du
  2026-08-26, 88-89 % contre temoin 56 %, erreur concentree aux bascules.

**B — SUSPENDU A LA DECISION (1) : le vivier Total Control.**

- Verdict D3 / D3-bis : `[!]` entier. **Aucun producteur, aucun champ, aucun bump imputable a
  TC.**
- Si l'entree est RETIREE (recommandation superviseur) : la modification porte sur
  `config/titles/halo_infinite/mappings/objective_roles.toml` (role `totalcontrol_zone`), PAS sur
  le schema. Elle change ce que le calque STATIQUE dessine — donc elle exige la re-cuisson des
  temoins TC, ou rien du tout si l'on accepte que les artefacts TC existants gardent leur vivier.
- Si l'entree est GARDEE : zero ligne de code, zero re-cuisson.

**C — SUSPENDU A LA DECISION (2) : les vies libres du crane.**

- Ce qui entre : UNE entree `[[objective_objects]]` dans `replay_labels.toml` — identifiant
  `0x0017592C`, `family = "ball"`, libelles EN+FR. **Rien d'autre : pas de calque de portage**
  (le seuil (2) l'a refuse), seulement l'objet LIBRE.
- **DEUX effets, pas un**, et le second n'est pas evident :
  1. `build_objectives_live.go:143` alimente `scan.Free` par `flagFreeLives` — le crane libre
     devient une piste publiee (16 a 47 vies par film mesurees).
  2. `ground_weapon_pads.go:192` consomme la MEME table via `weaponPadRule(cat.FlagObjects)` :
     l'entree transforme une exclusion **ACCIDENTELLE** en exclusion **VOULUE ET GARDEE**.
     Aujourd'hui le crane echappe aux socles d'armes seulement parce que son identifiant n'est
     pas au catalogue d'armes — c'est exactement l'accident que
     `ground_weapon_flag_exclusion_test.go` a ete ecrit pour empecher sur le drapeau.
- Champs de document : `objectivesLive.free` existe deja pour le drapeau — **a verifier au
  moment de coder** si la cle est partagee ou si le crane en demande une propre ; si elle est
  partagee, `wantReplayDocumentFields` reste a 38.

**D — TEMOINS A RE-CUIRE, un film par processus, via `cmd/replay-build` (aucune base ouverte).**

| decision | films |
|---|---|
| A (KOTH, acquis) | `01e1f945`, `606d9844`, `8076f97f` — le 4e du corpus KOTH, `0a247154`, joue sur **Solitude, absente du catalogue de formes** : il n'a pas de zones a peindre et ne sert pas de temoin |
| B (si retrait TC) | les films TC du recensement D1, `a349fea8` **EXCLU D'OFFICE** (bombe memoire, 3,17 Gio en 3,6 s) |
| C (si oui crane) | `24dbb67d`, `43716616`, `51ebbc0f`, `d9781168` — les 4 films Oddball mesurables ; les 3 autres du recensement n'ont pas de catalogue et ne se cuisent pas |
| tous cas | le golden d'assemblage `testdata/assembly_000d5950.golden` re-congele |

**RAPPEL DE GATE, mesure deux fois (journaux du 18/08) : ne PAS cuire de film pendant le gate
web.** Les garde-rails qui balaient `src/` expirent a 5 000 ms sur machine chargee, et ces echecs
ne sont pas des regressions.

### D3-ter, VERROU 1 — LA CALIBRATION DES LARGEURS : protocole ECRIT ET COMMITE AVANT LA SONDE

> Aucun chiffre de resultat. Mandat du product owner (2026-08-27) : Total Control est ROUVERT,
> l'arbitrage « pas de protocole suivant » est leve. Deux verrous dans l'ordre, arret au premier
> qui tient. Critere du verrou 1 fixe par le superviseur et RECOPIE SANS MODIFICATION :
> **chainage >= 80 % = canal lisible.**

**D'ABORD, UNE CORRECTION DE MON PROPRE CR D3-BIS.** J'ai presente le chainage de 5,3 a 14,5 %
comme une reserve invalidant la mesure. C'est INEXACT, et il faut le dire avant d'aller plus
loin : ce taux est le taux GLOBAL du balayage (`Chained/Walked` sur toutes les lectures `ti=13`),
alors que **l'election du designateur de D3-bis ne lisait DEJA que des lectures chainees**
(`totalcontrol_designateur_d3bis_test.go:165`, `!r.Chained` -> `continue`), exactement comme la
production (`zone_states.go:182`). Le taux global mesure la proportion de bruit dans le balayage ;
il ne mesure pas la qualite de ce que l'instrument a consomme. Ma reserve etait donc surdimensionnee
sur ce point precis. La seconde reserve — une seule manche par film, donc une unite de protocole
inexistante — reste entiere, et c'est elle que le verrou 2 corrige.

**L'HYPOTHESE DU SUPERVISEUR, ET POURQUOI ELLE SE TESTE AU LIEU DE SE PLAIDER.** Le precedent M3
est reel : sur `ti=42`, l'identite lue aux largeurs MPP par defaut rendait ZERO socle en silence
sur BTB. La question est de savoir si le balayage `ti=13` a la MEME dependance. La lecture du code
dit non — le bloc MPP appartient aux default-states de `ti=35/36/37/38/39/42/43`, pas a `ti=13`,
dont la grammaire porte en propre la mention « AUCUNE DESYNCHRONISATION N'EST POSSIBLE : la
largeur est entierement determinee par 4 bits lus dans le flux »
(`components_managed_property.go`). **Mais un argument de lecture peut manquer un chemin
indirect**, et le dernier lot a montre ce que coute de supposer (« le score d'Oddball monte a
~1 Hz » : suppose, jamais mesure, faux). La sonde MESURE donc l'independance au lieu de la
deduire.

- [x] D3t.1 **TEST A — le bouton fait-il quelque chose ?** Balayer `ScanFilmManagedProperties`
      sur le MEME film sous TROIS decoupages MPP installes par `SetMPPWidths` : le defaut
      **9/5**, le decoupage **8/3** mesure sur les films BTB du lot armes-au-sol, et un
      decoupage volontairement ABSURDE **12/7**. Comparer `Records`, `Walked`, `Chained` et le
      nombre de lectures, a l'unite pres.
- [x] D3t.2 **CRITERE DU VERROU 1**, tel que fixe : chainage `Chained/Walked` **>= 80 %** sur les
      deux films sondes. En dessous : verrou 1 NON TENU, arret, CR.
- [x] D3t.3 **DIAGNOSTIC DU VERROU REEL** — c'est ce que le superviseur demande en cas d'echec
      (« CR avec le verrou reel nomme »), et il se definit AVANT de mesurer pour ne pas se
      choisir a la lecture du chiffre. L'ancrage de `scanPayload` est un balayage EXHAUSTIF de
      toutes les positions de bit, valide par une signature FAIBLE (1 bit de prefixe, 13 bits de
      slot dans la bande, 2 bits de porte, 3 bits de compte, index croissants). Un tel ancrage
      produit mecaniquement des FAUX ANCRAGES en proportion de la taille du payload — ce qui
      predit exactement ce que D3-bis a observe : chainage qui BAISSE quand le film grossit, et
      nombre de « slots designateurs » qui CROIT avec lui. **Discriminant, ecrit d'avance :** les
      vrais records se concentrent sur peu de slots avec beaucoup de lectures, les faux se
      dispersent sur beaucoup de slots avec une ou deux lectures. Publier donc, sur le
      SOUS-ENSEMBLE CHAINE : le nombre de slots porteurs, et la part des lectures chainees portee
      par les **5** slots les plus fournis. **Si cette part est >= 80 %**, le taux global bas est
      un artefact d'ancrage et non une lecture fausse — le canal est lisible APRES filtrage.

**CE QUE LE DIAGNOSTIC N'AUTORISE PAS.** Meme si D3t.3 montre un sous-ensemble chaine tres
concentre, **le verrou 1 reste NON TENU si le critere >= 80 % de D3t.2 n'est pas atteint**, et
la mission s'ARRETE la, au CR, comme mandate. Le diagnostic NOMME le verrou reel ; il ne se
substitue pas au critere et il n'ouvre pas le verrou 2 de lui-meme. Ce sera au superviseur de
decider si le verrou reel ainsi nomme change le plan.

**CORPUS** : les deux films TC les plus LEGERS, `66aa5f0b` et `bf831a6b` — designes par le
superviseur, et ce sont ceux dont l'ancrage etait le moins mauvais en D3-bis. `a349fea8` EXCLU
D'OFFICE. Un film par processus, sentinelle memoire armee dans le processus qui decode, aucune
base ouverte.

- 2026-08-27 — **D3-ter VERROU 1 : NON TENU, et l'hypothese des largeurs est REFUTEE PAR LA
  MESURE.** Protocole commite avant la sonde (`5381ada22`). Deux films, un par processus, pic
  memoire 0,04 et 0,06 Gio. Sortie brute figee dans `registre_film/D3TER_largeurs_mpp.log`.

  **TEST A — le bouton des largeurs MPP ne fait RIEN sur `ti=13`.** Le MEME film balaye sous
  TROIS decoupages (defaut 9/5, BTB 8/3, absurde 12/7) rend des releves **identiques a l'unite
  pres**, sur les deux films :

  | film | slots | records | walked | chained | lectures | chainage |
  |---|---|---|---|---|---|---|
  | `66aa5f0b` | 528 | 100 918 | 44 849 | 6 486 | 93 915 | **14,5 %** |
  | `bf831a6b` | 669 | 165 518 | 78 529 | 9 771 | 158 186 | **12,4 %** |

  Trois decoupages, trois fois la meme ligne. **Le precedent M3 ne se transpose pas** : le bloc
  MPP appartient aux default-states de `ti=35/36/37/38/39/42/43`, `ti=13` n'en porte pas, et sa
  grammaire est integralement determinee par 4 bits lus dans le flux. Ce n'etait pas une piste
  faible, c'etait une piste NULLE — et elle est close par la mesure, pas par un argument de
  lecture.

  **CRITERE DU VERROU 1 : 14,5 % et 12,4 %, seuil >= 80 %. NON TENU sur les deux films.** Arret,
  comme mandate.

  **DIAGNOSTIC — et il ne conclut PAS ce qu'il devait conclure.** Discriminant ecrit d'avance :
  sous-ensemble chaine CONCENTRE = artefact d'ancrage ; DISPERSE = le bruit survit au filtre.
  Mesure : 314 et 491 slots portent des lectures chainees, les 5 plus fournis n'en portent que
  **50,3 %** et **46,6 %** (seuil 80 %). Verdict de mon propre discriminant : **DISPERSE**.

  **LA LIMITE EST DANS MON INSTRUMENT, ET JE LA NOMME PLUTOT QUE DE LA CONTOURNER.** J'ai defini
  la concentration sur TOUTES les lectures chainees. Or la population `ti=13` est ecrasee par le
  canal PAR JOUEUR (`i2..i33`, 32 instances par record), dont on sait deja qu'il chaine a 33 %
  contre 97 % pour le canal SCALAIRE (mesure du lot C-bis). Une dispersion mesuree sur cette
  population melangee ne dit presque rien du sous-canal scalaire — celui du tag 5, le seul que le
  designateur lise. **Mon discriminant n'etait donc pas le bon instrument pour la question qu'il
  devait trancher.** Je ne le rejoue PAS avec une population restreinte : le protocole interdit
  de choisir l'instrument apres avoir vu le chiffre, et cette regle vaut aussi contre moi.

  **CE QUE LA SONDE MONTRE QUAND MEME, ET QUI N'EST PAS RIEN.** Le tag 5 CHAINE est minuscule et
  propre : **4 slots / 4 valeurs distinctes** sur `66aa5f0b`, **3 slots / 3 valeurs distinctes**
  sur `bf831a6b`. Le bruit de 300 a 500 slots ne le touche pas. Je m'arrete la : dire ce que
  « 3 valeurs sur un mode a 3 zones » pourrait signifier serait exactement la conclusion que le
  verrou 2 est fait pour mesurer, et le verrou 2 n'est pas ouvert.

  **VERROU REEL, NOMME** : ce n'est NI les largeurs (refute a l'unite pres), NI le filtre de
  chainage du canal scalaire (le tag 5 chaine sort propre). Le taux global de 12-14 % est produit
  par la population MELANGEE du balayage — dominee par le canal par joueur, structurellement peu
  chainant — et **le critere « chainage >= 80 % » porte sur cette population melangee, donc sur
  une grandeur qui ne mesure pas la lisibilite du canal du designateur.** Le verrou est un verrou
  de METRIQUE avant d'etre un verrou de donnee. C'est au superviseur de decider si cela change le
  plan ; je ne rouvre rien de moi-meme.

  **TC reste `[!]`** et la decision d'affichage du vivier repart a l'utilisateur, comme prevu.

### D3-ter, VERROU 2 — L'UNITE INSTANT T : protocole ECRIT ET COMMITE AVANT LA MESURE

> Aucun chiffre de resultat. Le seuil du verrou 2 est celui du superviseur et **ne bouge pas d'un
> millimetre** : cardinal 3 sur **>= 80 %** du temps exploitable, sur **>= 2 films**.

**POURQUOI LA METRIQUE DE LISIBILITE EST REDEFINIE, ET POURQUOI CE N'EST PAS UN REGLAGE APRES
COUP** (justification du superviseur, recopiee). Le critere « chainage global >= 80 % » a ete
demontre **cable sur la mauvaise grandeur** : il mesurait une population MELANGEE dominee par le
canal par-joueur (`i2..i33`, 32 instances par record, chainage connu **33 %**), alors que la
lisibilite a trancher est celle du **sous-canal SCALAIRE tag 5 chaine**, le seul que le
designateur lise en production (`zone_states.go:182`). Le 33 % du canal par-joueur est une mesure
du lot C-bis, **anterieure a cette campagne** : la redefinition s'appuie sur des mesures
anterieures au resultat, pas sur le resultat. C'est ce qui la distingue du reglage apres coup que
le protocole interdit — et c'est la raison pour laquelle je ne me l'etais PAS autorisee moi-meme
au CR precedent.

**LA PRECONDITION DE LISIBILITE, REFORMULEE.** Le sous-canal tag 5 chaine doit rendre une SERIE
EXPLOITABLE : au moins **N** emissions, couvrant **>= 50 %** du temps de match. La couverture est
`(derniere emission - premiere emission) / (duree du match)`, le match etant borne par les
enregistrements d'entite.

**N EST FIXE PAR RELEVE, PAS PAR CHOIX — ET LE RELEVE EST FAIT SUR KOTH, JAMAIS SUR LES FILMS
TC.** Regle ecrite ici, avant de mesurer quoi que ce soit :

> **N := le PLUS PETIT nombre d'emissions de tag 5 chainees observe sur les quatre films du
> corpus KOTH**, mesure par LE MEME instrument que celui qui mesurera les films TC.

Deux raisons de prendre le MINIMUM et non la mediane. D'abord, le corpus KOTH est celui sur
lequel la voie designateur est **elue 4 films sur 4 et SERVIE EN PRODUCTION** : son plancher est,
par construction, le niveau d'emission auquel le depot accepte deja de lire un designateur.
Ensuite, une precondition PERMISSIVE est le choix CONSERVATEUR pour le gate : elle laisse entrer
plus de films TC dans la mesure, donc elle expose davantage l'hypothese a l'echec. Un N severe
aurait ecarte les films maigres et flatte le taux.

**LE MEME INSTRUMENT SUR LES DEUX CORPUS** : sans cela, « le TC emet moins que le KOTH » pourrait
n'etre qu'une difference de definition. Le releve KOTH et la mesure TC passent par le meme code,
et le releve est publie avant le verdict TC.

- [x] D3t2.1 **RELEVE KOTH** : sur `01e1f945`, `606d9844`, `8076f97f`, `0a247154` — nombre
      d'emissions de tag 5 chainees, nombre de slots porteurs, couverture. **N en decoule par la
      regle ci-dessus.**
- [x] D3t2.2 **MESURE TC** : les six films du recensement, `a349fea8` EXCLU D'OFFICE.
      Precondition d'abord ; un film qui ne la passe pas ne compte NI POUR NI CONTRE, et la cause
      est nommee.
- [x] D3t2.3 **VERDICT** : seuil du superviseur, inchange.

**LA MESURE, RENDUE OPERATOIRE.** L'ensemble DESIGNE a l'instant t est l'ensemble des valeurs de
tag 5 chainees EN VIGUEUR a t — pour chaque slot porteur, sa derniere emission a `<= t`. Un slot
qui n'a pas encore emis ne contribue pas ; **la valeur zero n'est pas une designation** (meme
regle qu'en D3-bis).

- Les **POINTS DE CHANGEMENT** sont les instants ou cet ensemble change, c'est-a-dire ou un slot
  quelconque passe a une valeur differente de la sienne. **Ce sont les rotations** : plus besoin
  de manches, et c'est tout l'objet de cette reformulation — l'unite « manche » n'existait pas sur
  ce corpus (1 manche par film, mesure D3-bis).
- Autour de chaque point de changement, une fenetre de **+/- 2 s est EXCLUE**, valeur inchangee
  depuis D3-bis : a la bascule le jeu retire des zones et en pose d'autres, et compter cette
  fenetre ferait mecaniquement un cardinal double.
- Le **TEMPS EXPLOITABLE** court de la PREMIERE emission a la fin du match, moins les fenetres
  exclues. Avant la premiere emission il n'y a pas de serie : cet intervalle n'est pas
  exploitable, il n'est pas non plus compte contre.
- Sur chaque intervalle restant, le cardinal est CONSTANT. On somme la duree des intervalles de
  cardinal **exactement 3** et on la rapporte au temps exploitable.

**INTERDIT, ECRIT ICI POUR M'Y TENIR** : ne rien conclure des « 3 et 4 valeurs distinctes »
relevees par la sonde du verrou 1. Un ensemble de 3 valeurs sur TOUT le match et un ensemble de
cardinal 3 A CHAQUE INSTANT sont deux enonces differents, et le second n'est pas implique par le
premier — c'est exactement ce que cette mesure existe pour trancher.

**ISSUES.** Tenu : les 3 actives (NEUTRES, sans possession — le seuil de possession n'a jamais
ete evalue) entrent en D5, contenu final A+B'+C, bump 21. Rate : **TC `[!]` DEFINITIF pour
v7.5**, CR avec le verrou nomme, et la decision d'affichage du vivier repart a l'utilisateur.

**EXECUTION** : un film par processus, sentinelle memoire armee dans le processus qui decode,
aucune base ouverte, `a349fea8` exclu d'office.

- 2026-08-27 — **D3-ter VERROU 2 : NON TENU sur les quatre films exploitables. TC est `[!]`
  DEFINITIF pour v7.5.** Protocole commite avant la mesure (`1c866d01d`). Un film par processus.
  Sorties brutes figees dans `registre_film/D3TER_releve_koth.log` et `D3TER_instant_tc.log`.

  **1. LE RELEVE KOTH FIXE `N`, ET IL DONNE EN PRIME LE CONTROLE POSITIF QUI MANQUAIT A TOUTE LA
  CAMPAGNE.**

  | film KOTH | emissions tag 5 chainees | slots | couverture | cardinal dominant |
  |---|---|---|---|---|
  | `01e1f945` | 7 | 4 | 79,9 % | **1 pendant 100,0 %** |
  | `606d9844` | **5** | 4 | 55,5 % | **1 pendant 100,0 %** |
  | `8076f97f` | 8 | 5 | 48,9 % | **1 pendant 98,4 %** |
  | `0a247154` | 8 | 4 | 74,7 % | **1 pendant 100,0 %** |

  Par la regle ecrite d'avance, **`N` = 5** (le plus petit des quatre).

  **LE CONTROLE POSITIF EST LE RESULTAT LE PLUS IMPORTANT DE CE VOLET.** King of the Hill n'a
  qu'UNE colline a la fois. L'instrument, applique tel quel a ce corpus, rend **cardinal 1 pendant
  98,4 a 100 % du temps exploitable, sur 4 films sur 4**. Il mesure donc bien ce qu'il pretend
  mesurer, et le canal est lisible avec seulement 5 a 8 emissions. **D3-bis n'avait aucun controle
  de ce genre** : son negatif etait invalidable, celui-ci ne l'est pas de la meme facon.

  **2. LA MESURE TC.** Precondition `N >= 5` et couverture `>= 50 %` :

  | film | emissions | slots | couverture | precondition | cardinal 3 | cardinal max observe |
  |---|---|---|---|---|---|---|
  | `bf831a6b` | 3 | 3 | 58,4 % | **ECHOUE** (3 < 5) | (1 pendant 100 %) | 1 |
  | `66aa5f0b` | 4 | 4 | 60,2 % | **ECHOUE** (4 < 5) | (1 pendant 100 %) | 1 |
  | `a521164d` | 59 | 40 | 86,3 % | passe | **0,0 %** | **36** |
  | `2f05dc98` | 22 | 18 | 93,1 % | passe | **27,8 %** | 16 |
  | `0862dce4` | 168 | 79 | 94,6 % | passe | **0,2 %** | **77** |
  | `d2c64f8c` | 27 | 24 | 93,9 % | passe | **3,9 %** | 22 |

  **Seuil : 80 % du temps exploitable, sur >= 2 films. Mesure : 0,0 a 27,8 % sur les quatre films
  exploitables. NON TENU.**

  **LE VERDICT EST ROBUSTE A LA PRECONDITION** : en comptant AUSSI les deux films ecartes (0,0 %
  chacun), la serie devient 0,0 / 0,0 / 0,0 / 0,2 / 3,9 / 27,8 % — meme verdict. `N` n'a donc
  sauve aucun chiffre, dans aucun sens.

  **3. LE VERROU, NOMME.** Le sous-canal tag 5 chaine reste CONTAMINE sur les films BTB, et le
  chiffre qui le dit est le **cardinal maximal simultane : 16, 22, 36 et 77**. Un objet de mode a
  trois zones ne peut pas designer soixante-dix-sept choses a la fois. Et ce maximum **croit avec
  la taille du film** (77 sur le plus lourd, 16 sur le plus leger des exploitables) — la meme
  signature que le nombre de slots en D3-bis. **Le filtre de chainage est NECESSAIRE mais PAS
  SUFFISANT sur BTB** : il nettoie assez pour que l'arene (KOTH) rende un cardinal stable, pas
  assez pour que le BTB en rende un.

  Ce n'est donc ni les largeurs (refute a l'unite pres au verrou 1), ni l'unite de mesure
  (corrigee ici, et validee par le controle positif KOTH), ni la metrique de lisibilite
  (redefinie et satisfaite : couverture 86 a 95 % sur les quatre exploitables). **C'est
  l'ANCRAGE sur les payloads BTB**, et il faudrait un ancrage plus fort — une bande de slots
  restreinte a l'objet de mode plutot que tous les slots `ti=13` des images-cles — ce qui est un
  chantier de decodage, pas un ajustement de protocole.

  **4. CE QUE JE NE CONCLUS PAS.** Les deux films legers rendent cardinal 1 pendant 100 % du
  temps, exactement comme KOTH. Il serait tentant d'en tirer que Total Control designe UNE zone a
  la fois et non trois. **Je ne le fais pas** : ces deux films ECHOUENT la precondition (3 et 4
  emissions), ils ne comptent ni pour ni contre, et une serie de 3 points ne decrit pas un match
  de 5 minutes. C'est une observation, pas un resultat — et elle rejoint l'interdit que je m'etais
  ecrit avant de mesurer.

- [x] D3t2.1 Releve KOTH : fait, `N` = 5, controle positif obtenu (cardinal 1 sur 4/4).
- [x] D3t2.2 Mesure TC : faite sur 6 films, 4 exploitables, 2 ecartes par la precondition.
- [x] D3t2.3 Verdict : **NON TENU**. `totalcontrol_zone` ne rejoint PAS `heldZoneRoles`, les 3
      actives n'entrent PAS en D5, **TC `[!]` DEFINITIF pour v7.5**. La decision d'affichage du
      vivier repart a l'utilisateur.

**CONTENU FINAL DE D5** : **A (proprietaire de colline KOTH) + C (vies libres du crane)**, bump
unique au schema **21**. B reste hors publication.

- 2026-08-27 — **D5 LIVRE : schema 21, contenu A + B + C.** Bump unique, triptyque bumpe
  ensemble, temoins re-cuits et VERIFIES sur pieces.

  **A — LE PROPRIETAIRE DE LA COLLINE EST PUBLIE.** Verifie dans les artefacts re-cuits :

  | temoin KOTH | zones | intervalles | AVEC proprietaire | camps | methode |
  |---|---|---|---|---|---|
  | `01e1f945` Catalyst | 4 | 100 | **50** | 0 et 1 | `designator+geometry` |
  | `606d9844` Chasm | 3 | 14 | **7** | 0 et 1 | `designator+geometry` |
  | `8076f97f` Shogun | 3 | 36 | **18** | 0 et 1 | `designator+geometry` |

  Le 4e film du corpus KOTH, `0a247154`, joue sur **Solitude, absente du catalogue de formes** :
  il n'a aucune zone a peindre et ne sert pas de temoin.

  **B — L'ENTREE TOTAL CONTROL EST RETIREE** (decision utilisateur, option (a)). Le mode ne
  declare plus aucun role : le vivier de 13 a 18 formes par carte ne s'affiche plus. Le bloc de
  retrait est DATE dans `objective_roles.toml`, avec les trois mesures qui le motivent et la
  condition de reprise (ancrage `ti=13` fiable sur BTB = chantier de decodage). **L'ancienne
  entree est conservee en commentaire** : le jour de la reprise, c'est ce qu'il faudra relire.

  Trois gardes posees, parce qu'une ABSENCE ne se garde pas toute seule :
  1. `TestObjectiveRoles_FichierDuDepot` : **7 modes** au lieu de 8, et l'assertion
     « `totalcontrol_zone` doit etre servi » est **INVERSEE** — elle exige desormais qu'il ne le
     soit PLUS, avec un message qui renvoie a la condition de reprise.
  2. `TestMapObjectives_TotalControl_NeSertPlusRien` (neuf) : les trois libelles de mode
     (`Arena:`, `BTB:`, `BTB:Fiesta`) ne rendent AUCUNE spec, et **Command** — qui porte 18
     zones `totalcontrol_zone` au catalogue — ne rend AUCUN objectif.
  3. Le commentaire de `replay_map_objectives.go` qui se disait « deja vieilli deux fois » a
     vieilli une troisieme : il est mis a jour dans le commit qui le perime.

  **Temoin TC re-cuit et verifie** : `66aa5f0b` (Command, BTB:Total Control) — la cle
  `zoneStates` est **ABSENTE** du document et `coverage.zones` vaut `null`. Ni zones ni vivier.

  **C — LE CRANE : identite au manifeste et exclusion GARDEE. La publication des vies libres n'a
  PAS ete faite, et la raison n'est pas un renoncement.**

  Livre : l'entree `[[objective_objects]]` `0x0017592c` famille `ball` EN+FR, la famille `ball`
  ouverte dans la liste fermee, et la table renommee `FlagObjects` -> `ObjectiveObjects` partout
  (20 occurrences, 11 fichiers) — garder `FlagObjects` aurait fait dire au code que le crane est
  un drapeau. Verifie dans les artefacts re-cuits, et le controle est fort : le compteur
  `coverage.groundWeapons.objectives` rend **23 / 16 / 21 / 47** sur les quatre films Oddball,
  c'est-a-dire EXACTEMENT les comptes de creations que la mesure D4 avait attribues a
  `0x0017592C` (23 / 16 / 21 / 47). L'exclusion des socles d'armes est passee d'ACCIDENTELLE a
  VOULUE ET GARDEE, et `TestLeManifesteNommeSesObjetsDObjectifDansLesDeuxLangues` exige
  desormais CHAQUE famille separement — un comptage global aurait laisse disparaitre une famille
  entiere en silence.

  **NON LIVRE, ET C'EST UNE DECOUVERTE, PAS UN ARBITRAGE DE MA PART** : les vies libres ne sont
  publiees NULLE PART, ni pour le crane ni pour le drapeau. Verifie sur pieces : `scan.Free` n'a
  que deux consommateurs (`closeByFreeLives`, `repositionFlagDrops`, `flag_objects.go`), aucune
  cle `free` n'existe au document, et `objectivesLive` n'existe pas davantage. Le fichier le dit
  lui-meme depuis le 2026-08-18 : « Elles ne sont PAS publiees — elles CORRIGENT le calque ».
  **Mon inventaire D5 du 2026-08-27 affirmait le contraire** (« `objectivesLive.free` existe deja
  pour le drapeau »), avec la mention « a verifier au moment de coder » ; la verification a eu
  lieu, et elle infirme. « Comme pour le drapeau » signifie donc, litteralement, « pas publie du
  tout ».

  Publier les vies libres du crane demanderait une cle de document NEUVE, un champ de contrat en
  plus (39 au lieu de 38), un rendu web, et le passage de la garde de mode `IsFlagFilm` qui
  arrete aujourd'hui tout ce chemin hors CTF. C'est une SURFACE PRODUIT que personne n'a
  specifiee. Je ne l'ai pas inventee : elle revient au superviseur et a l'utilisateur.

  **LE TRIPTYQUE, BUMPE ENSEMBLE** : `replay.SchemaVersion` 20 -> **21** (avec sa chronique en
  tete de `document.go` ET l'entree v20->v21 du cliquet de `structure_test.go`),
  `EXPECTED_REPLAY_SCHEMA_VERSION` 20 -> **21** (garde de parite verte).
  `wantReplayDocumentFields` reste a **38** et `api/openapi.yaml` est INCHANGE (`openapi-gen
  -check` vert) : aucune cle ne bouge, `generated.ts` ne bouge pas. Golden d'assemblage re-congele
  — son diff est la SEULE ligne `schema 20` -> `schema 21`, ce qui confirme qu'aucun champ n'a
  bouge.

  **UN PIEGE D'EXECUTION, RENCONTRE ET CORRIGE** : la premiere re-cuisson lisait le manifeste du
  depot PRINCIPAL (`LEVELUP_REPO_ROOT` y pointait pour les donnees) et non celui du worktree —
  les artefacts sortaient donc avec `objectives=0`, c'est-a-dire SANS l'entree du crane, tout en
  portant `schema=21` puisque la version est compilee dans le binaire. **Un artefact peut porter
  la bonne version et l'ancienne configuration** ; seule la verification sur pieces l'a montre.
  Corrige par une racine temporaire (jonctions : `config` du worktree, `data` du principal), et
  les sept temoins ont ete re-cuits avec.

- 2026-08-27 — **D5-bis LIVRE : le crane libre est PUBLIE et DESSINE. Schema 21 inchange,
  contrat 38 -> 39.** Dernier lot de code du point 11.

  **CE QUI EST PUBLIE.** Une cle de document neuve, `objectiveObjects` : une entree par VIE
  LIBRE de l'objet — il apparait, replique sa position, puis se tait. Forme volontairement
  GENERIQUE (`family`, `en`, `fr`, `t0`, `t1`, `pts`) pour que le drapeau puisse la rejoindre
  sans qu'aucune cle ne bouge. Couverture `coverage.objectiveObjects` avec ses denominateurs.

  **VERIFIE SUR LES QUATRE TEMOINS RE-CUITS**, et le controle est fort : le nombre de vies vaut
  **23 / 16 / 21 / 47**, exactement les comptes de creations que la mesure D4 avait attribues a
  `0x0017592C`. Zero vie hors axe sur les quatre.

  | temoin | vies | dont MOBILES | dont immobiles | points | hors axe |
  |---|---|---|---|---|---|
  | `24dbb67d` Recharge | 23 | 21 | 2 | 730 | 0 |
  | `43716616` Smallhalla | 16 | 14 | 2 | 454 | 0 |
  | `51ebbc0f` Banished Narrows | 21 | 15 | 6 | 546 | 0 |
  | `d9781168` Dredge | 47 | 33 | 14 | 1 157 | 0 |

  Les vies MOBILES sont la preuve que le canal porte bien du mouvement et pas seulement des
  apparitions : sur `43716616`, une vie court des frames 343 a 371 avec un point par frame.

  **LA GARDE DE MODE EST LEVEE POUR CE CANAL SEULEMENT, ET SUR PIECES.** `attachFlagCarries`
  s'arrete hors CTF pour proteger le pont d'identite (`SlotIdentityByDeaths`, 19 a 22 Go sur un
  film d'un autre mode). Ce calque-ci ne lit NI le statborg, NI le fil des morts, NI l'identite
  des joueurs : il ne consomme que le balayage `ti=42` de la chaine des socles, deja paye sur
  TOUS les films. Le placer sous la garde du drapeau l'aurait eteint sur Oddball — la ou il sert.
  **Aucune lecture de film n'est ajoutee par ce lot.**

  **LE SCHEMA RESTE 21, ET LA JUSTIFICATION EST ECRITE AU CONTRAT** : le bump du lot a eu lieu
  dans le meme lot, 21 n'a quitte ni le poste ni les temoins locaux, aucun artefact 21 n'existe
  ailleurs. Le bump unique reste unique. Le triptyque est coherent : `SchemaVersion` 21,
  `EXPECTED_REPLAY_SCHEMA_VERSION` 21, garde de parite verte.

  **CE QUE LE CALQUE REFUSE DE FAIRE, et c'est sa propriete centrale** : dessiner le crane
  pendant les portages. Entre deux vies il y a un trou ; quelqu'un porte, et le document ne dit
  pas qui — l'oracle a ete mesure puis REFUTE en D4. Le calque se tait. Il n'interpole pas
  davantage : le crane est a la DERNIERE position qu'il a emise, ou nulle part. Un test dedie
  garde chacune de ces deux proprietes (`objectiveObjectAt` hors vie = `null`, y compris
  au-dela de `t1`).

  **LE DRAPEAU N'ENTRE PAS DANS CE CALQUE, ET CE N'EST PAS UN REPORT** : le CONTROLE 3 de son
  propre lot a ECHOUE sur ses vies libres (149/197 = 75,6 % pour un seuil de 90 % ecrit avant la
  mesure ; temoin a 12,8 %). Un quart de ses vies reste inexplique. Le jour ou ce negatif sera
  leve, une ligne de `objectiveObjectPublished` suffira.

  **RENDU WEB** : glyphe BOULE (disque cerne) — distinct de la hampe + fanion du drapeau, parce
  que deux objets de mode au meme glyphe seraient indiscernables et que le rejeu se lit aussi en
  niveaux de gris. Encre NEUTRE : le document ne publiant aucun porteur, une encre d'equipe
  afficherait une appartenance que la mesure refuse. Aucun libelle ni bascule ajoute — donc
  aucune chaine i18n neuve.

  **DIXIEME EXTRACTION IMPOSEE PAR LE CLIQUET DU CANVAS**, en deux morceaux : le cablage du
  calque part dans `useReplayObjectiveObjects.ts` (patron de `useReplayFlagCarries`), les trois
  reglages constants dans `replayCanvasConfig.ts`. Le cliquet passe de **706 a 695** — il
  descend, il ne remonte pas.

  **GARDE-RAIL DE FRONTIERE** : `objectiveObjects` et `objectiveObjects[].pts` entrent dans
  `NULLABLE_ARRAYS` / `NULLABLE_ARRAY_PATHS`. Le test de contrat les a EXIGES a la compilation
  avant que je n'y pense — c'est exactement ce pour quoi il existe.

### D6-PORTAGE — LA VOIE DE LA PROXIMITE : protocole ECRIT ET COMMITE AVANT LA MESURE

> Mandat utilisateur du 2026-08-27 : le portage Oddball rouvre par une voie NOUVELLE. Ce n'est
> PAS un quatrieme passage du meme oracle — l'oracle change de NATURE (il sort du film), et le
> canal change de PRINCIPE (la proximite geometrique, pas la correlation d'une serie).

**VOLET 2 D'ABORD, PARCE QU'IL DECIDE DU GATE. L'ORACLE EST INSTRUIT SUR PIECES, ET IL EXISTE.**
La table `match_objective_stats_latest` porte, PAR JOUEUR et pour Oddball :

    skull_grabs                              prises
    time_as_skull_carrier_seconds            TEMPS DE PORTAGE, en secondes
    longest_time_as_skull_carrier_seconds    plus long portage
    skull_scoring_ticks                      tics de score
    kills_as_skull_carrier / skull_carriers_killed

Releve sur les quatre films du corpus (lecture seule) :

| film | lignes | `skull_grabs` | `skull_scoring_ticks` | `time_as_skull_carrier_seconds` | porteurs tues |
|---|---|---|---|---|---|
| `24dbb67d` | 8 | 2 | 321 | **331** | 12 |
| `43716616` | 10 | 2 | 213 | **218** | 10 |
| `51ebbc0f` | 8 | 4 | 255 | **266** | 13 |
| `d9781168` | 8 | 10 | 387 | **404** | 25 |

**DEUX CHOSES SE LISENT DEJA, ET ELLES COMPTENT.** (1) `skull_scoring_ticks` suit
`time_as_skull_carrier_seconds` a 3-4 % pres sur les quatre films : **un tic vaut une seconde de
portage**, ce qui CONFIRME enfin le « ~1 Hz » que le §2.4 supposait — supposition que D4 avait
mesuree fausse SUR LE SCORE PERSONNEL, et qui se revele juste sur le canal du MODE. Les deux
constats ne se contredisent pas : le score personnel melange le portage a tout le reste, le tic
de crane ne compte que le portage. (2) `skull_grabs` est MINUSCULE (2 a 10) devant les 16 a 47
vies libres : ce compteur ne compte donc PAS chaque ramassage, et il ne sera PAS l'oracle.

**L'ORACLE RETENU EST `time_as_skull_carrier_seconds`, PAR JOUEUR.** Trois raisons, dans l'ordre
de force : il mesure EXACTEMENT la grandeur que la reconstruction produit (une duree de portage
par joueur) ; il est **INDEPENDANT DU FILM** — il vient de l'API, il n'a jamais vu nos chunks, et
aucune erreur de decodage ne peut le contaminer ; il est DENSE (6 a 8 porteurs par film) la ou
`skull_grabs` ne rendrait que 2 a 10 points.

**LES EVENEMENTS `th=10` NE SERONT PAS L'ORACLE, et ce que je sais d'eux est dit ici.** Leur
compte (87 / 60 / 75 / 124) ne vaut ni les prises, ni les tics, ni les porteurs tues ; il suit
les tics dans un rapport de 3,1 a 3,7. **Je n'ai pas etabli ce qu'ils datent**, et le protocole
ne le leur demande pas : leur repartition PAR JOUEUR sera publiee a cote du verdict, comme
diagnostic, pour que la question se referme ou se pose proprement plus tard. Choisir un oracle
qu'on ne comprend pas quand il en existe un qu'on comprend serait la faute exacte de D4.

- [x] D6.1 **RECONSTRUCTION** par proximite aux bornes des vies libres.
- [x] D6.2 **CE QUE FAIT LA MORT** : mesure, pas hypothese.
- [x] D6.3 **VERDICT** contre l'oracle API, avec temoin.

**LA RECONSTRUCTION, RENDUE OPERATOIRE.** Les vies libres du crane sont deja publiees (schema
21). Entre la fin d'une vie (`t1`, DERNIERE POSITION REPLIQUEE — precise a l'image, pas le
plafond de 20 s des images-cles qui a coule l'item 2.5 des socles) et le debut de la suivante
(`t0`), il y a un TROU. Pour chaque trou :

1. **QUI** : le bipede dont la position a `t1` est la plus proche de la derniere position de
   l'objet. La position du bipede est prise a l'echantillon le plus proche de `t1`, et l'ecart
   temporel doit valoir **<= 250 ms** (`d6EcartMaxMS`) — au-dela, on compare deux instants, pas
   deux lieux.
2. **SEUIL DE DISTANCE : 1,5 m** (`d6RayonRamassageM`). **CE SEUIL N'EST PAS INVENTE ICI** : c'est
   `originDropMaxDist`, deja au depot, deja valide DES DEUX COTES sur la chaine des poses — les
   lachers y sont a 0,63 m de mediane, les deploiements a 5,6-21,3 m. Un objet ramasse est aux
   pieds de qui le ramasse, exactement comme un objet lache. **LA DISTRIBUTION COMPLETE des
   distances au plus proche sera PUBLIEE** : si les deux populations ne se separent pas, le seuil
   ne vaut rien et il faudra le dire plutot que de s'en servir.
3. **AMBIGUITE** : si le DEUXIEME plus proche est lui aussi sous le seuil ET a moins de
   **1,0 m** (`d6AmbiguiteM`) du premier, le porteur est **null** — le trou est compte, mais
   attribue a personne. C'est la doctrine deja tenue par les occupations de socle, dont le `xuid`
   vaut TOUJOURS null parce que l'oracle plafonnait a 79,7 %.
4. **RETOUR, PAS PORTAGE** — c'est ma reserve d'hier, transformee en cas CLASSE : si AUCUN joueur
   n'est sous le seuil a `t1` ET que la vie suivante nait a **<= 3 m** du socle `oddball_spawn`,
   le trou est un RETOUR. Aucun portage n'est attribue, et il ne compte pas contre.
5. **INEXPLIQUE** : aucun joueur proche, et la vie suivante ne nait pas au socle. Compte a part.
6. **FIN DU PORTAGE** : `t0` de la vie suivante, ou la MORT du porteur si elle tombe avant.

**D6.2 — CE QUE FAIT LA MORT SE MESURE, ET LE PROTOCOLE DIT QUOI.** Hypothese a tester : un
porteur qui meurt LACHE le crane, donc une vie libre doit naitre pres du lieu de sa mort. On
publie la part des morts de porteur suivies, dans les **3 s**, d'une naissance de vie libre a
**<= 3 m** du lieu de la mort. Ce chiffre ne conditionne pas le gate : il DIT si la regle de
cloture 6 est fondee.

**LE TEMOIN, DEFINI AVANT DE COMPTER** : la MEME chaine, du debut a la fin, mais le porteur d'un
trou est tire AU HASARD parmi les joueurs nommes (graine fixe, `d6GraineTemoin`) au lieu d'etre
le plus proche. Meme code, meme cloture, meme oracle, meme metrique. Si la proximite ne porte
rien, les deux scores se rejoindront.

**LE GATE, CHIFFRE ET ECRIT D'AVANCE — DEUX CONDITIONS, LES DEUX EXIGEES.**

1. **RECOUVREMENT DU TEMPS DE PORTAGE** : `somme sur les joueurs de min(reconstruit, API)` divise
   par `somme des temps API`. Seuil **>= 0,80**, et **temoin <= 0,50**.

   **POURQUOI 0,80 ET NON 0,90, ECRIT AVANT DE VOIR LE CHIFFRE.** Les deux grandeurs ne sont pas
   la MEME grandeur physique : le temps API compte le portage, le temps de trou compte
   l'absence de replication — laquelle inclut aussi les retours et les re-creations. Exiger 0,90
   testerait l'egalite de deux choses dont on sait deja qu'elles different. 0,80 avec un temoin
   trente points plus bas teste ce qui est reellement en question : **la proximite designe-t-elle
   le bon joueur ?**
2. **LE PORTEUR PRINCIPAL, criterium SANS SEUIL REGLABLE** : le joueur au plus grand temps API
   est-il aussi celui au plus grand temps reconstruit ? Exige sur **>= 3 des 4 films**.

**ESCALADE** : un film dont le pont ne nomme pas au moins deux joueurs, ou dont l'oracle API est
absent, n'est pas exploitable et ne compte ni pour ni contre. Moins de 2 films exploitables =
arret. **Si le gate tombe, STOP au CR** — pas de cinquieme oracle.

**SI LE GATE TIENT** : le porteur entre dans `objectiveObjects` (le champ generique est pret),
SANS nouveau bump — le schema 21 n'a toujours pas quitte le poste — et le rendu pose le crane sur
son porteur, patron du drapeau porte.

**EXECUTION** : les 4 films D4, un par processus, sentinelle memoire armee dans le processus qui
decode, base en LECTURE SEULE pour le seul oracle.

- 2026-08-27 — **D6-PORTAGE : GATE NON TENU. Le portage n'est PAS publie.** Protocole commite
  avant la mesure (`d7dd3c19c`). Quatre films, un par processus, pic 0,06 a 0,13 Gio. Sorties
  figees : `registre_film/D6_portage_proximite.log` et `D6_oracle_api_portage.json`.

  | film | trous | PORTE | ambigu | retour | inexplique | reconstruit | API | **recouvrement** | temoin | principal |
  |---|---|---|---|---|---|---|---|---|---|---|
  | `24dbb67d` | 22 | 7 | 3 | 2 | 10 | 190 s | 331 s | **51,7 %** | 11,3 % | manque |
  | `43716616` | 15 | 7 | 1 | 1 | 6 | 56 s | 218 s | **25,9 %** | 17,1 % | manque |
  | `51ebbc0f` | 20 | 3 | 0 | 2 | 15 | 7 s | 266 s | **2,6 %** | 4,1 % | manque |
  | `d9781168` | 46 | 18 | 4 | 8 | 16 | 206 s | 404 s | **48,9 %** | 27,8 % | **identifie** |

  **Seuil : recouvrement >= 80 % ET porteur principal sur >= 3 films sur 4. Mesure : 2,6 a
  51,7 %, principal sur 1 film sur 4. NON TENU, deux fois.** Aucune publication, comme ecrit.

  **CE QUI EST POSITIF, ET QUI NE DOIT PAS DISPARAITRE DANS LE NEGATIF :**

  1. **LA PRIMITIVE DE PROXIMITE FONCTIONNE, et la distribution le montre au lieu de l'affirmer.**
     Les distances au plus proche a l'instant du trou se separent nettement en DEUX populations :
     q25 entre **0,20 et 0,43 m** (quelqu'un est exactement la) contre q75 entre **5,5 et 7,9 m**
     (personne). Mediane **0,77 m** sur `d9781168` — la valeur meme que la doctrine du 12/08
     citait. **Le seuil de 1,5 m ne se regle pas, il se constate** : n'importe quelle valeur entre
     1 et 3 m rendrait le meme classement. Le seuil n'est donc PAS la cause de l'echec.
  2. **CE QUE FAIT LA MORT EST ETABLI, ET C'EST UN OUI FRANC : 22 sur 24 = 91,7 %.** Un portage
     ferme par la mort de son porteur est suivi, dans les 3 s, d'une naissance de vie libre a
     moins de 3 m du lieu de la mort — 5/5, 7/7, 10/10, et 1/2 sur le film au pont casse. **La
     regle « mourir, c'est lacher » n'est plus une hypothese de protocole : elle est mesuree.**
     C'est acquis quel que soit le sort du reste.
  3. **LE TEMOIN SE COMPORTE COMME IL DOIT sur les trois films au pont sain** : 51,7 contre 11,3
     (x4,6), 25,9 contre 17,1 (x1,5), 48,9 contre 27,8 (x1,8). La proximite PORTE de
     l'information — elle n'en porte simplement pas assez.

  **POURQUOI LE GATE TOMBE, NOMME SUR PIECES.** Deux causes distinctes, et il faut les separer.

  **(a) UN FILM EST HORS D'ETAT, ET CE N'EST PAS LA PROXIMITE.** `51ebbc0f` : le pont ne nomme
  que **9 slots de bipede sur 84**, contre 87/97, 62/72 et 140/160 ailleurs. Sans joueurs nommes,
  aucune proximite ne peut designer qui que ce soit — d'ou 2,6 %, et le seul film ou le temoin
  BAT le signal (4,1 contre 2,6). Ce film aurait du sortir par une precondition de pont que mon
  protocole n'a pas ecrite : je l'ai bornee a « au moins deux slots nommes », ce qui est
  beaucoup trop laxiste. **Defaut de mon protocole, pas de la donnee.**

  **(b) LA RECONSTRUCTION SOUS-ATTRIBUE, MASSIVEMENT ET SYSTEMATIQUEMENT.** Sur les trois films
  sains elle rend 190/331, 56/218 et 206/404 secondes — environ la MOITIE. La classe
  `inexplique` (personne a moins de 1,5 m, et la vie suivante ne nait pas au socle) pese 10/22,
  6/15 et 16/46 des trous. Et le detail par joueur est sans appel : sur `43716616`, deux gros
  porteurs de l'oracle (94 s et 62 s) recoivent 3 s et 0 s. **Ce ne sont pas des erreurs
  d'attribution, ce sont des portages entiers que la chaine ne voit pas commencer.**

  **CE QUE JE NE CONCLUS PAS.** Je n'ecris pas « la voie de la proximite est refutee ». Les deux
  positifs ci-dessus disent le contraire : la primitive discrimine, et la mort est comprise. Ce
  qui est refute, c'est **cette reconstruction-ci**, dont le point d'entree — « la derniere
  position repliquee est le lieu du ramassage » — rate la moitie des ramassages. Pourquoi, je ne
  l'ai pas mesure : il faudrait instrumenter ce que fait l'objet dans les quelques images qui
  precedent son silence, et ce serait un protocole neuf.

  **STOP AU VERDICT, comme mandate.** Pas de cinquieme oracle, pas de reglage du seuil apres
  coup — et surtout pas de publication d'un portage dont la moitie manque.

- [x] D6.1 Reconstruction par proximite : faite, sous-attribution massive, `[!]`.
- [x] D6.2 Ce que fait la mort : **MESURE ET ACQUIS — 22/24 = 91,7 %**.
- [x] D6.3 Verdict contre l'oracle API : **NON TENU** (2,6 a 51,7 % contre 80 % ; principal 1/4
      contre 3/4). Le portage n'entre PAS dans `objectiveObjects`, aucun rendu de porteur, aucun
      bump. Le calque du crane LIBRE, lui, reste tel qu'il a ete livre en D5-bis.

### D7 — SONDE DIAGNOSTIQUE : pourquoi la moitie des ramassages est invisible

> **PROTOCOLE ECRIT ET COMMITE AVANT LA MESURE.** Aucun chiffre de resultat. Arbitrage
> superviseur du 2026-08-27 : UNE sonde bornee, pas une cinquieme campagne. Elle NE PROPOSE PAS
> de reconstruction — elle NOMME une cause. L'oracle API et le seuil de 80 % restent le juge, et
> ils ne sont pas rejoues ici.
>
> **CORPUS : les DEUX films au pont sain, `24dbb67d` et `d9781168`.** `51ebbc0f` est exclu
> d'office (9 slots nommes sur 84 — c'est le defaut de precondition que D6 a nomme), et
> `43716616` n'entre pas : deux films suffisent a departager trois signatures, et la sonde est
> bornee par mandat.

**CE QUI EST DEJA ETABLI ET QU'ON NE REMESURE PAS** : la primitive de proximite discrimine (q25 a
0,20-0,43 m contre q75 a 5,5-7,9 m), la mort fait lacher (22/24 = 91,7 %), et la reconstruction
D6 rend environ la moitie du temps de portage de l'API. **La question est UNIQUEMENT : ou passe
l'autre moitie ?**

**TROIS PISTES, TROIS SIGNATURES ECRITES D'AVANCE. Chacune dit ce qui la CONFIRMERAIT et ce qui
l'ECARTERAIT — sans quoi la sonde trouverait ce qu'elle cherche.**

- [ ] **S1 — LA DERNIERE POSITION EST PERIMEE (ramassage en mouvement).** Si l'objet roule ou
      vole quand on le ramasse, sa derniere position REPLIQUEE est en retard de quelques images
      sur le lieu reel du ramassage, et chercher le joueur le plus proche a ce point-la vise a
      cote.
      **Mesure** : pour chaque trou, la distance minimale a un joueur nomme non plus au seul
      instant `t1`, mais sur une FENETRE `t1 ± W` avec **W = 0, 1, 2, 5, 10 images**. Publiee
      aussi : la VITESSE de l'objet sur ses trois dernieres images.
      **CONFIRME SI** la part de trous ayant un joueur sous 1,5 m monte NETTEMENT avec `W` —
      convention ecrite d'avance : **+15 points ou plus entre W = 0 et W = 5**. **ECARTEE SI**
      elle bouge de moins de 5 points : le joueur n'est alors pas « un peu plus loin dans le
      temps », il n'est nulle part.
- [ ] **S2 — DEUX PORTAGES FUSIONNES EN UN.** Si l'objet est relache puis repris trop vite pour
      emettre une vie libre lisible, deux portages n'en font qu'un, et la moitie des ramassages
      n'existe simplement pas dans la chaine des vies.
      **Mesure** : la duree des portages reconstruits confrontee a
      `longest_time_as_skull_carrier_seconds` de l'API pour CE film — un portage reconstruit plus
      long que le plus long portage que l'API connaisse est necessairement une fusion. Publiees
      aussi : la distribution des durees de vie libre et le compte de vies a UN seul point.
      **CONFIRME SI** au moins un portage reconstruit depasse le maximum de l'API, ou si la part
      des vies libres de moins d'une image est notable (**>= 20 %**). **ECARTEE SI** aucun
      portage ne depasse le maximum API et que les vies courtes sont marginales.
- [ ] **S3 — DES VIES NAISSENT SANS ETRE APPARIEES.** Si des vies libres apparaissent loin du
      socle, loin d'un joueur et loin du silence precedent, la chaine « une vie, un trou, une
      vie » est rompue et des trous entiers sont mal bornes.
      **Mesure** : classer la NAISSANCE de chaque vie libre — au socle (<= 3 m), aux pieds d'un
      joueur (<= 1,5 m), au lieu du silence precedent (<= 3 m), ou INEXPLIQUEE.
      **CONFIRME SI** la part de naissances inexpliquees est **>= 30 %**. **ECARTEE SI** elle
      est **<= 10 %**.

**LES TROIS NE S'EXCLUENT PAS**, et le protocole ne les force pas a le faire : la sonde publie
les trois signatures, et le CR dit laquelle domine — ou dit qu'aucune ne domine, ce qui serait un
resultat en soi.

**CE QUE LA SONDE NE FERA PAS.** Elle ne modifie AUCUN seuil de D6, ne rejoue AUCUN verdict, ne
propose AUCUNE reconstruction et ne publie RIEN dans l'artefact. Le mandat est de rendre la cause,
pas de la corriger dans la foulee — et la decision finale (nouvelle reconstruction, ou `[!]` avec
les acquis consignes) revient au superviseur, l'oracle API et le seuil de 80 % restant le juge.

**EXECUTION** : un film par processus, sentinelle memoire armee dans le processus qui decode,
oracle API FIGE en entree (aucune base ouverte par la sonde).
