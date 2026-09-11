# Lot 6.7 phase B2 — couverture des calques d'objectif : rendu, KOTH et `flag_secures` (2026-09-11)

> Worktree `LevelUp-wt-couverture-objectifs-b`, branche `wt/couverture-objectifs-b`, depuis
> `feat/v75` a `c3860bc46` (audit + phase B1 inclus).
> Preuve d'entree de chaque item : `.ai/V7.5/AUDIT_COUVERTURE_OBJECTIFS_2026-09-10.md` (§10
> causes, §11 gates) et `.ai/V7.5/RAPPORT_ODDBALL_FANTOMES_2026-09-10.md` §6.2.
> **Aucune base DuckDB n'a ete ouverte**, aucun artefact du parc partage n'a ete recuit, le
> serveur de developpement n'a pas ete touche : toutes les cuissons sont HORS LIGNE, dans un
> dossier de scratchpad, a partir des chunks lus en LECTURE SEULE.

## 0. Verdict en cinq lignes

1. **Le glyphe porte ne ment plus et ne disparait plus** : crane, bombe et drapeau suivent
   desormais UNE SEULE regle quand le porteur n'a pas de position — l'objet est rendu LIBRE a sa
   derniere position CONNUE, avec l'habillage d'un lieu. 1 227 images concernees au parc.
2. **Les zones de Total Control restent absentes, et c'est `[!]` instruit** : l'audit supposait
   un defaut ; c'est une decision utilisateur datee du 2026-08-27, et une mesure NEUVE sur le
   film de l'audit lui-meme la confirme (cardinal 3 pendant **0,0 %** du temps exploitable, seuil
   80 %).
3. **Les films KOTH cuisent** : 6 sur 7, code de sortie 0. La cause de l'echec n'etait PAS le
   catalogue de cartes (les 6 cartes portent 5 ou 6 collines a forme) mais l'identite de carte non
   resolue — le match n'etait pas au registre local.
4. **Les tics de colline sont desormais tenus par un test qui echoue** : corpus 4 films / 31
   joueurs -> **6 films / 47 joueurs, 0 desaccord, 0 joueur au-dessus de son oracle**.
5. **`flag_secures` a un emplacement** : `comp 23 B`, seul exact des 260 balayes, sur **11 films,
   79 slots, 0 desaccord**. 233 securisations publiees pour 264 a l'oracle, **ratio 1,000 film par
   film sur 11 films**, aucun joueur au-dessus.

---

## 1. Etat des items

| # | item | statut | gate |
|---|---|---|---|
| 1 | glyphe porte sans position du porteur (web) | `[x]` | tenu |
| 2 | calque de zones sur Total Control | `[!]` | **NON atteignable sans contredire une decision utilisateur ou publier 15 zones pour 3** ; §3 |
| 3 | KOTH : catalogue de cartes | `[x]` pour 6 films / `[!]` pour `0a247154` | tenu ; §4 |
| 4 | `flag_secures` : instruire l'emplacement | `[x]` | tenu, gate STRICT passe |

Commits, dans l'ordre :

| item | SHA | message |
|---|---|---|
| 1 | `98ec81da0` | le glyphe PORTE sans position du porteur est rendu LIBRE, une seule regle |
| 2 | `dc0b0ef30` | item 2 (zones Total Control) statue `[!]` — le designateur est refute sur `5676a9ba` |
| 3 | `33301324e` | les tics de colline tenus par un GATE, corpus 4 -> 6 films, 31 -> 47 joueurs |
| 4 | `3e1e1dc69` | `flag_secures` a un emplacement — `comp 23 B`, exact sur 11 films, 0 desaccord |

## 1.1 Le protocole de mesure, et ce qui le rend rejouable

La phase B1 avait ses faits de match exportes depuis la base. Ici la base n'a pas ete ouverte du
tout : les `<short8>.facts.json` des cuissons hors ligne sont **derives des exports commites** de
la vague 6 (`oracle_vague6_registry.tsv` + `oracle_vague6_participants.tsv`), c'est-a-dire de la
meme chaine que `levelup replay-facts-export`.

**La derivation est validee A L'OCTET** : appliquee a `01e1f945`, elle reproduit exactement le
fichier de reference commite `apps/go-api/internal/analysis/replay/testdata/equivalence/01e1f945.facts.json`
(memes joueurs, meme triplet, memes scores, meme `mapId`, memes `mapNames`).

**SA LIMITE, ET ELLE SE VOIT DANS LES CHIFFRES** : les exports ne portent pas
`joined_in_progress` / `left_in_progress`. Un film d'arene a 9 ou 10 lignes de feuille compte
donc 9 ou 10 SIEGES au lieu de 8, et la garde d'effectif livree en B1 le tait entierement. Deux
films CTF sur treize sont dans ce cas (§5.4) ; en production la garde lit la colonne et ces films
publient. Le balayage par joueur, lui, ne passe pas par la garde : il les mesure, et les trouve
exacts.

---

## 2. Item 1 — le glyphe porte sans position du porteur (`98ec81da0`)

### 2.1 Cause verifiee sur pieces

Les trois calques repondaient chacun a leur facon a la meme image muette. Relu ligne par ligne
avant de coder :

| calque | ligne | comportement AVANT |
|---|---|---|
| crane | `layers/skullCarrierLayer.ts:100-101` | `if (!w) continue` — le crane DISPARAIT |
| bombe | `layers/bombCarrierLayer.ts:129-130` | idem — la bombe DISPARAIT |
| drapeau | `layers/flagCarriesLayer.ts:237-241` | repli sur `{x: now.x, y: now.y}`, l'ANCRE DU SPAN : une position PERIMEE, dessinee avec l'habillage du PORTE |

Et `model/skullPresence.ts:65` donnait la precedence a `carried` sans condition : le calque de
l'objet LIBRE se taisait pendant que le calque du porte ne dessinait rien. Effectif mesure a
l'audit : **533 images / 32 464 (1,64 %)** pour le drapeau, **694 images (7,0 % du parc Oddball)**
pour le crane.

### 2.2 Correctif — une seule ecriture de la regle

`apps/web/src/features/match-replay/model/carriedGlyphPlace.ts` (neuf, 104 lignes) porte
`carriedGlyphPlaceAt`. Sans position du porteur a l'image, l'objet est rendu **LIBRE a sa
derniere position connue** : la derniere position du porteur DEPUIS LE DEBUT DU PORTAGE (jamais
avant `t0` — au-dela ce serait la position d'avant la prise), a defaut le repli servi par
l'appelant. Les trois etats (`carried` / `free` / `absent`) sont EXCLUSIFS : c'est l'invariant qui
interdit deux glyphes du meme objet a la meme image.

Les trois consommateurs :

- **crane** — la precedence du portage de `skullPresenceAt` est CONDITIONNEE a une position du
  porteur (5e parametre `posOf`, `null` = comportement historique, degradation intacte) ; le repli
  est le dernier repos, puis le socle. `useReplayObjectiveObjects` prend desormais le document et
  relit les positions par le resolveur commun `useCarrierPosAt`, **le meme que
  `skullCarrierLayer`** — les deux calques doivent s'accorder a l'image pres, sinon le crane
  clignoterait entre porte et libre.
- **bombe** — rendue au sol (`ALPHA_GROUND`, sans le decalage vertical du porte) a sa derniere
  position connue. Repli `null` : la bombe n'a AUCUN canal de position propre.
- **drapeau** — `flagPointAt` devient `flagPlaceAt` et rend aussi `loose`. L'ancre du span reste
  le dernier repli (c'est une position MESUREE), mais elle perd le decalage `FLAG_OFFSET_X` et le
  bornage hors cadre : l'habillage « porte » designe un JOUEUR, et il n'y en a pas a l'ecran. **Le
  SURVOL suit la meme geometrie** (`flagAt`), sans quoi la cible serait a 6 px du glyphe.

L'etat publie (`now.state`) n'est JAMAIS falsifie : l'infobulle continue de lire ce que l'artefact
ecrit. Seul le RENDU change.

**Garde-rail** (regle n° 6 du depot, 3 consommateurs) :
`model/carriedGlyphPlace.guard.test.ts` interdit que l'ancre perimee ou le balayage arriere se
reecrivent ailleurs, et derive de la source la liste des trois consommateurs.

### 2.3 Gate, AVANT -> APRES

| commande | avant | apres |
|---|---|---|
| `npx vitest run src/features/match-replay/{layers,model}` (4 fichiers cibles) | **12 tests ROUGES** | **12 verts** |
| `npx vitest run src/features/match-replay` | — | **2 705 verts**, 3 skip |
| `npx vitest run` (suite web complete) | — | **7 354 verts**, 17 skip, 695 fichiers |
| `make check-types` (apres purge de `node_modules/.tmp`) | — | **propre** |
| eslint sur les 12 fichiers touches | — | **0 erreur, 1 avertissement PREEXISTANT** (`ReplayCanvas.tsx:547`, `zoneInk.outline`), verifie tel quel sur le fichier de `HEAD` |

**Tests de mutation** (un par calque, plus deux sur le helper) :

- `carriedGlyphPlace.test.ts` — le balayage arriere rend la position la plus RECENTE (une
  implementation qui partirait de `t0` servirait la plus ancienne) ; les trois etats sont exclusifs.
- `bombCarrierLayer.test.ts` — le repli ne remonte JAMAIS avant `t0`.
- `flagCarriesLayer.test.ts` — le SURVOL suit le glyphe libere (ecart de decalage exactement
  `FLAG_OFFSET_X` entre porte et libere, des deux cotes).
- `skullPresence.test.ts` — sans lecteur de position, la precedence du portage est INCHANGEE (si
  le defaut etait `() => null`, ce test rougirait).
- `skullCarrierLayer.test.ts` — « exactement un glyphe a chaque image » : le calque porte se tait
  ET la presence bascule sur `free` a la meme image.

**Recuisson : AUCUNE.** Lot web pur, le document ne change pas.

---

## 3. Item 2 — les zones de Total Control : `[!]`, et la cause n'est pas celle de l'audit

### 3.1 Ce que l'audit supposait, et ce que le code dit

L'audit (§5) constate que `5676a9ba` publie 23 captures mais `zoneStates` vide et `coverage.zones`
absent, et ecrit : « Deux causes se superposent et l'audit ne les separe pas : le mode et
l'effectif ». Instruction sur pieces : **ce n'est ni l'un ni l'autre, et ce n'est pas un defaut de
code.**

1. `totalcontrol_zone` n'est pas dans `heldZoneRoles` (`internal/replaybuild/zones.go:66`) ;
2. l'entree Total Control de `config/titles/halo_infinite/mappings/objective_roles.toml` a ete
   **RETIREE le 2026-08-27 sur decision utilisateur (option (a))**, motivee par trois mesures
   independantes, chacune avec son protocole ecrit d'avance (phases D3, D3-bis, D3-ter) :
   attribution **38,2 %** (seuil 80 %), largeurs MPP **refutees a l'unite**, cardinal **0,0 a
   27,8 %** (seuil 80 %) ;
3. la condition de reprise est ECRITE dans le fichier : un ancrage plus fort du balayage `ti=13` —
   « un CHANTIER DE DECODAGE, pas un ajustement de configuration : ne pas remettre cette entree
   sans lui ».

### 3.2 La mesure neuve, sur la cible exacte du gate

`5676a9ba` n'etait PAS au corpus D3-ter du 2026-08-27 (`66aa5f0b`, `bf831a6b`, `a521164d`,
`d2c64f8c`). L'instrument a donc ete rejoue sur lui — sortie
`replay2d/registre_film/vague6_b2_item2_total_control.log` :

| grandeur | valeur | seuil |
|---|---|---|
| couverture de la serie du designateur | 51,3 % du match | 50 % |
| temps exploitable | 339 799 ms sur 669 887 | — |
| **cardinal 1** | **100,0 % du temps exploitable** | — |
| **cardinal 3** (les trois zones du mode) | **0,0 %** | **80 %** |

Le canal qui donne la colline de KOTH ne donne pas les trois zones de Total Control sur ce film.

### 3.3 Pourquoi le gate serait atteint par une publication FAUSSE

Le catalogue d'Insolence (`d5c5eb4f-0dcb-4677-a866-eae0dcbfde9b`) porte **15** `totalcontrol_zone`,
toutes avec forme, pour **3** zones actives par manche. Remettre l'entree TOML publierait donc le
VIVIER — cinq fois trop — ce qui satisferait la lettre du gate (« >= 3 zones ») en trahissant son
esprit, et refarait exactement le dommage que le retrait de 2026-08-27 nomme.

### 3.4 La seconde moitie du gate est hors d'atteinte par construction

« captures publiees >= 46/51 » : depuis l'item 5 de B1, la garde d'effectif refuse le film ENTIER
(27 lignes de feuille pour 8 slots de statborg, `coverage.objectives.refusedByRoster`). Les 23
captures de l'audit sont donc devenues 0, et les rendre suppose de fermer l'hypothese du plafond
de 8 slots — que l'audit §10 range explicitement parmi les hypotheses NON prouvees, et qui est le
prealable de L6.

**Aucun code n'a ete modifie pour cet item** : corriger ici demanderait soit de contredire une
decision utilisateur datee, soit de publier 15 zones pour 3.

---

## 4. Item 3 — KOTH : la cause reelle, et le gate des tics de colline (`33301324e`)

### 4.1 L'hypothese de l'audit est REFUTEE

L'audit posait la question en deux branches : « catalogue d'objectifs sans formes de colline pour
ces cartes ? identite de carte non resolue ? ». Recensement du catalogue versionne pour les 7
cartes des films KOTH du cache :

| film | carte | volumes `hill` | dont a forme |
|---|---|---|---|
| `01e1f945` | Catalyst | 6 | 6 |
| `0a247154` | Solitude - Ranked | 5 | 5 |
| `21ece4d8` | Live Fire | 5 | 5 |
| `606d9844` | Chasm | 5 | 5 |
| `7f1bbf06` | Streets | 5 | 5 |
| `8076f97f` | Shogun | 5 | 5 |
| `a36c8bed` | Isolation | 5 | 5 |

**Rien ne manque au catalogue, rien n'est a extraire.** La cause etait deja ecrite dans le journal
d'echec lui-meme (`E4_cuisson_koth.log`) : « match absent du registre — identite de carte non
resolue (passer `--map-name`) », `err="sql: no rows in result set"`
(`cmd/levelup/cmd_backfill_replay.go:349-363`). Les 7 matchs n'etaient pas au registre local le
2026-08-30 ; ils y sont depuis (`oracle_vague6_registry.tsv` les porte tous).

### 4.2 Cuisson, AVANT -> APRES

| | avant (2026-08-30) | apres (2026-09-11) |
|---|---|---|
| films cuits | **0 / 7**, code de sortie 10 | **6 / 7, code de sortie 0** |
| `zoneStates` publies | — | **3 ou 4 par film** |
| `coverage.zones` | absent | **present** (methode `designator+geometry`, role `hill`) |

`0a247154` (Solitude - Ranked) est statue **`[!]`** : il n'a **aucun chunk au cache** (purge) — il
n'y a rien a cuire, et ce n'est pas un defaut de code. Sortie :
`replay2d/registre_film/vague6_b2_koth_parc.tsv` et `vague6_b2_koth_zones.tsv`.

### 4.3 Gate des tics de colline, AVANT -> APRES

`hill_hold_ticks.go:8-12` affirmait que `comp 23 A` reproduit `ZonesStats.StrongholdScoringTicks`
« exactement, joueur par joueur, sur 31 joueurs de 4 films ». **Cette affirmation n'etait tenue par
aucun test qui echoue** : `TestCollineStatborgE1Bis` est un RELEVE qui journalise ses desaccords et
passe quand meme (anti-patron n° 9).

| | avant | apres |
|---|---|---|
| corpus | 4 films / 31 joueurs | **6 films / 47 slots apparies** |
| desaccords | (non re-mesures depuis le 2026-08-30) | **0** |
| joueurs au-dessus de leur oracle | — | **0** (tolerance 0) |
| tenue par un test qui echoue | **non** | **oui** (`koth_hold_ticks_gate_test.go`) |

Detail par film (sortie `vague6_b2_item3_koth.log`) : `01e1f945` 8/8, `21ece4d8` 8/8, `606d9844`
8/8, `7f1bbf06` 8/8, `8076f97f` 8/8, `a36c8bed` 7/7 — 0 desaccord partout.

**Test de mutation** : le gate rejoue son PROPRE comparateur sur un oracle mute d'UN tic et exige
qu'il trouve le desaccord. Sans lui, un comparateur toujours vert passerait sur n'importe quel film.

L'oracle de Chasm et Shogun (les deux films neufs) est **derive des exports commites**, jamais
tape : la meme derivation reproduit LIGNE POUR LIGNE les quatre films deja geles dans
`e1bOracle`, lignes de bot comprises.

---

## 5. Item 4 — `flag_secures` : `comp 23 B` (`3e1e1dc69`)

### 5.1 Le protocole, et le gate ecrit d'avance

Celui qui a nomme `flag_grabs` (`named.go:139-155`) : balayage des emplacements, valeur par
valeur apres pont slot -> xuid, l'exactitude PAR JOUEUR et jamais la coincidence de somme. Le gate
etait STRICT et pose avant la mesure : **>= 10 films ET aucun desaccord**, sinon on ne publie rien.

Corpus : les **13 CTF d'arene** du parc qui ont des chunks. Les trois CTF de BTB (`4f77afc1` 36
lignes, `879a4dba` 26, `1c4c63c2` 24) sont exclus — au-dela de huit sieges le statborg ne dit de
qui il parle (`rosterfit.go`), et les y inclure fabriquerait des desaccords qui ne diraient rien du
composant.

**TEMOIN POSITIF** : le balayage doit d'abord retrouver `flag_grabs` a `comp 22 A` sans desaccord
sur le film. Un balayage qui ne sait plus reconnaitre une reponse juste ne prouve rien de son
silence.

### 5.2 Le resultat

**`comp 23 B` est le SEUL emplacement exact des 260 balayes** (comps 0 a 64 x cote x strict), et il
l'est sur **11 films sur 11 exploitables, 79 slots apparies, 0 desaccord**.

Controle de NON-VACUITE — « zero desaccord » ne vaut rien si la cible est nulle partout :

| film | joueurs apparies a securisation NON NULLE | somme |
|---|---|---|
| `16ea3668` | 5 | 7 |
| `4ecdf3e7` | 4 | 10 |
| `58864b3c` | 3 | 5 |
| `64e8adfa` | 4 | 14 |
| `8bc6074f` | 7 | 20 |
| `a0c36016` | 7 | 16 |
| `b8a44fe8` | 7 | 43 |
| `bc60b4d9` | 4 | 12 |
| `bf5ced1b` | 3 | 5 |
| `cde26226` | 5 | 20 |
| `f8efc5ca` | 6 | 21 |
| **total** | **55** | **169** |

Deux films ne temoignent **ni pour ni contre**, et il faut le dire : `7fce3219` (CTF multi-manche)
ne rend AUCUN pont d'identite par le triplet, et `fb1a1a72` fait tomber le TEMOIN POSITIF (§6, D2).

### 5.3 Pourquoi `23 B` etait « tombe », et pourquoi ce n'est pas une contradiction

`named.go` le listait parmi « ce qui est TOMBE au controle ». Il y etait tombe comme candidat a la
**RECOMPENSE** `runner_stopped`, co-nommee avec `21 B` (etat de l'art §17.5/§17.6). La recompense
couvre indistinctement « tuer le porteur adverse » (`flag_carriers_killed`, `21 B`) et
« securiser » (`flag_secures`) — et l'oracle de l'epoque ne portait aucune colonne `flag_secures` :
**la bonne cible n'etait pas dans le jeu d'essai**. Le commentaire est corrige a sa source, pas
contourne (anti-patron n° 9).

Le registre fige `.ai/refs/TABLE_STATS_STATBORG.tsv` porte la nouvelle ligne ; sa concordance avec
`namedStatSlots` est elle-meme testee (`TestTableStatborgConcordeAvecNamedStatSlots`), et ce test
a d'ailleurs rougi tant que la ligne manquait.

### 5.4 Publication mesuree, AVANT -> APRES

Cuisson HORS LIGNE des 13 CTF apres le correctif — sortie `vague6_b2_ctf_secures.tsv` :

| | avant | apres |
|---|---|---|
| securisations publiees | **0** sur 264 | **233** sur 264 (**0,883**) |
| films au ratio 1,000 | 0 | **11 sur 13** |
| joueurs au-dessus de leur oracle | — | **0** |

Les deux films a 0 (`4ecdf3e7`, `8bc6074f`) sont **tus par la garde d'effectif de B1** parce que
mes faits HORS LIGNE ne portent pas `joinedInProgress` (§1.1) : leurs 9 lignes comptent 9 sieges au
lieu de 8, `refusedByRoster` vaut 96 et 218. En production la garde lit la colonne et ces deux
films publient — l'audit les mesurait d'ailleurs a 1,000 sur les autres actions de drapeau. Le
balayage par joueur, lui, les trouve exacts (8 slots chacun, 0 desaccord). **Les 13 films sont donc
couverts, chacun par au moins une des deux mesures.**

---

## 6. Films a recuire — la liste exacte pour le superviseur

**EN PLUS des 26 films de B1** (§8 du rapport B1), et par item :

| films | raison | item | nature |
|---|---|---|---|
| `01e1f945` `21ece4d8` `606d9844` `7f1bbf06` `8076f97f` `a36c8bed` | zones et tics de colline KOTH desormais cuisibles | 3 | **PREMIERE cuisson** (aucun artefact au parc) |
| `64e8adfa` `fb1a1a72` | `flag_secures` publiee | 4 | **PREMIERE cuisson** (aucun artefact au parc) |
| `16ea3668` `4ecdf3e7` `58864b3c` `7fce3219` `8bc6074f` `a0c36016` `b8a44fe8` `bc60b4d9` `bf5ced1b` `cde26226` `f8efc5ca` | `flag_secures` publiee | 4 | recuisson — **DEJA dans les 26 de B1**, ces films y gagnent seulement une raison de plus |
| `4f77afc1` `879a4dba` | aucune : ils restent tus par la garde d'effectif | 4 | **rien a faire** (deja dans les 26 de B1) |

Soit **8 films neufs** (6 KOTH + 2 CTF), tous en PREMIERE cuisson, et 11 des 26 films de B1 qui
changent pour une raison supplementaire.

`0a247154` (KOTH, Solitude - Ranked) n'a **aucun chunk au cache** : rien a cuire, `[!]`.

**Aucune montee de `SchemaVersion`**, aucune modification d'`openapi.yaml` ni de `generated.ts` :
`objectives[].stat` est une chaine libre au schema, et le web traite la famille `flag_*` par
prefixe (`objectivesLayer.ts:250`) — `flag_secures` se dessine comme `flag_carriers_killed`, sans
son propre son ni sa propre chaine d'interface. **Aucune string UI nouvelle**, donc aucun ajout
i18n.

---

## 7. Gates de lot, joues reellement

| commande | resultat |
|---|---|
| `go test ./internal/analysis/replay/ ./internal/analysis/objectiveevents/ ./internal/replaybuild/` | **ok**, 3 paquets |
| `CGO_ENABLED=1 go test ./internal/service/...` | **ok**, 4 paquets |
| `golangci-lint run ./internal/analysis/replay/... ./internal/analysis/objectiveevents/...` | **0 issues** |
| `go vet` (paquets touches + hook de pre-commit, 4 fois) | propre |
| `gofmt -l internal/` | propre |
| `make check-types` (apres purge de `node_modules/.tmp`) | propre |
| `npx vitest run` (suite web complete) | **7 354 verts**, 17 skip |
| `npx eslint` sur les 12 fichiers web touches | 0 erreur, **1 avertissement PREEXISTANT** verifie tel quel sur `HEAD` |
| `make openapi-gen` / `generate-types` | **sans objet** : aucun changement de contrat |

**Goldens.** Aucun golden de CI ne change du fait des items 1 et 3. L'item 4 fait bouger l'etape
`objectives` des films CTF du corpus d'equivalence (`64e8adfa`, `53ce4390`, `51101d1d`, `084a804d`,
`1c4c63c2`) : c'est le diff ATTENDU d'une statistique nouvellement nommee. **Les references n'ont
PAS ete regenerees**, et la raison est au §8 D1 — le corpus porte un ecart PREEXISTANT qu'un
`-update` baquerait en silence. Le corpus d'equivalence appartient au superviseur (instruction du
lot) ; il doit trancher D1 avant de re-figer.

Les seules references de TEST qui bougent sont celles de l'item 4 : la ligne `flag 23 B` de
`.ai/refs/TABLE_STATS_STATBORG.tsv`, exigee par le test de concordance.

---

## 8. Decouvertes, consignees et NON traitees

- **D1 — Le corpus d'equivalence porte un ecart PREEXISTANT sur l'etape `score`, sur TOUS les films
  testes, y compris le golden Slayer.** `000d5950`, `7344d24f`, `d9781168`, `64e8adfa`, `53ce4390`,
  `51101d1d`, `084a804d` : `attendu compte=1`, `obtenu compte=0` (sha de la valeur vide).
  `b.observe("score", stats.score)` (`internal/replaybuild/replaybuild.go:223`) observe donc un
  `stats.score` nil la ou la reference en avait un.
  **VERIFIE AU BINAIRE DE `c3860bc46`** (worktree temporaire, meme racine de donnees) : l'ecart est
  IDENTIQUE. Il est donc anterieur a B2, et il est dans la branche que B1 a livree. Les artefacts,
  eux, publient bien leur `scoreTimeline` — l'ecart porte sur l'OBSERVATEUR, pas forcement sur la
  sortie. **A instruire avant tout `-update` du corpus** : re-figer maintenant effacerait la trace.
  Reproduction : `LEVELUP_REPO_ROOT=<racine> go run ./cmd/replay-equiv -films 000d5950`.
- **D2 — `fb1a1a72` publie 10 prises de drapeau pour 0 a l'oracle** sur le joueur
  2535450323793545 (slot 24), alors que les 7 autres slots sont exacts. `comp 22 A` est un
  emplacement de PRODUCTION : c'est donc une publication FAUSSE, et elle passe SOUS la borne de
  deroulage recalibree par B1 (16). Le film n'avait pas d'artefact au parc, l'audit ne l'avait donc
  jamais mesure. Famille C6, non traitee ici.
- **D3 — `7fce3219` (CTF multi-manche) ne rend AUCUN pont d'identite par le triplet.** Le film est
  parfaitement cuisible et publie 15 securisations pour 15 (le chemin de production utilise le pont
  PAR MANCHE via le fil des morts), mais le pont par triplet — celui des instruments — y est vide.
  Deux ponts, deux couvertures : a instruire si un instrument futur veut couvrir le multi-manche.
- **D4 — Deux films KOTH laissent un joueur reel non apparie** : `a36c8bed` 7 slots sur 8, `8076f97f`
  8 sur 9. Le compteur n'est pas en cause (0 desaccord sur les apparies) ; c'est le pont d'identite.
- **D5 — Le temps en zone par joueur reste a 0 sur les six films KOTH** desormais cuits
  (`vague6_b2_koth_zones.tsv`, colonne `publie_par_joueur`), pour 226 a 603 s d'oracle par film.
  C'est la cause C8 / l'item L9 de l'audit — une ESCALADE utilisateur, pas un correctif, et
  explicitement renvoyee au lot 6.9 par le plan maitre.
- **D6 — Le catalogue d'Insolence porte 15 `totalcontrol_zone`** pour 3 zones actives : le vivier
  que le retrait du 2026-08-27 a justement refuse de servir. Consigne pour le jour de la reprise.
- **D7 — `0a247154` (KOTH, Solitude - Ranked) n'a plus de chunks au cache.** Meme famille que les
  cinq purges deja consignees au rapport 6.2 §6.5.

---

## 9. Reproduction

```bash
# 1. faits du match, DERIVES des exports commites (aucune base ouverte) — cf. §1.1
#    la derivation est validee a l'octet contre testdata/equivalence/01e1f945.facts.json

# 2. cuisson HORS LIGNE d'un film (chunks lus en lecture seule, racine de travail en scratchpad)
LEVELUP_REPO_ROOT=<racine de travail> \
  go run ./apps/go-api/cmd/replay-build --map <carte> --facts <short8>.facts.json <matchId> \
  <racine partagee>/data/cache/film_chunks/<short8>

# 3. instruments par film (un film par processus, aucune base)
ZONE_FILM=<chunks>/<short8> go test ./internal/analysis/replay/ -run TotalControlInstant -count=1 -v
ZONE_FILM=<chunks>/<short8> go test ./internal/analysis/replay/ -run KothHoldTicks       -count=1 -v
ZONE_FILM=<chunks>/<short8> go test ./internal/analysis/replay/ -run FlagSecuresSweep    -count=1 -v

# 4. confrontation du parc a l'oracle API
cd .ai/V7.5/outillage/couverture_objectifs
CGO_ENABLED=0 go run . -parc <parc cuit> -oracle <racine>/.ai/V7.5/replay2d/registre_film -out <sortie>
```

Sorties committees : `.ai/V7.5/replay2d/registre_film/vague6_b2_*`.
