# CONCEPTION — INVERSION DE PRESEANCE CREDIT <-> FILM

> Ecrit le 2026-08-02. **IMPLEMENTE LE 2026-08-03 — §6 CLOS, tous les items statues (voir §9).**
> Les mesures du corps du document sont celles de la CONCEPTION, prises en lecture seule sur des
> COPIES des bases de developpement. Les mesures de VERIFICATION apres implementation sont au §9 :
> elles confirment le document au chiffre pres (134 866 lignes, 73 589 enrichissements,
> 980 orphelins dont 13 humain-contre-humain, 0 instant ambigu, 0 identite divergente).
>
> **Ce qui RESTE : la bascule des 20 sites de lecture** (phase 2 du plan de branchement) — c est
> la session 2/2, et elle n a PAS ete faite ici.
>
> Documents d autorite : `PLAN_MASTER_FILM_KILLFEED_REJEU.md` (J4, STRATEGIE DE MERGE) et
> `PLAN_BRANCHEMENT_KILLSOURCE.md` (phase 2 — LA BASCULE, qui SUIT cette conception).

---

## 0. LE PROBLEME, RAPPELE EN TROIS LIGNES

La preseance actuelle est **film > credit** : un match porteur d un film voit sa passe de film
remplacer ENTIEREMENT la generation servie (la vue `_latest` retient une passe par match). Or la
passe de film ne publie que **74,4 %** des morts que le credit porte a **98,4 %**. La bascule des
lecteurs a donc ete arretee en session 3 : elle aurait efface 25 697 morts sans erreur ni compteur.

Cette conception inverse la relation : **le CREDIT devient la base** (la liste des morts) et **le
FILM devient un enrichissement** (l arme via `source_tag`, l assistant nomme, les parts de degats,
la divergence).

---

## 1. LE FAIT QUI COMMANDE TOUTE LA CONCEPTION

**Les deux sources sont le MEME flux, lu a deux endroits.**

Le kill-feed du film est extrait du chunk HIGHLIGHT et decode par
`analysis.ParseHighlightEvents` (`games/halo_infinite/film/killsource/feed.go:59`) — **exactement
le meme parseur** que celui qui alimente la table `highlight_events` depuis l API. Ce ne sont pas
deux mesures independantes d un meme evenement : c est une seule mesure, livree par deux canaux.

Consequence directe, et elle est mesuree : **il n y a aucun ecart d horloge a absorber.** Les
instants sont des millisecondes entieres identiques au bit pres. C est ce qui rend l appariement
trivial — et c est ce qu il fallait etablir avant de choisir une tolerance.

Deuxieme fait, verifie separement : `ComputeKillerVictimPairs` (`analysis/killer_victim.go:124`)
horodate le couple sur l instant du **KILL** (`TimeMS: k.t`), pas sur celui de la mort. Le film
ancre ses couples sur le meme instant. Les deux axes de temps coincident donc par construction, et
non par chance.

---

## 2. L APPARIEMENT — MESURE, PUIS TRANCHE

Perimetre : base Halo Infinite de developpement, **1 343 matchs** porteurs de couples credit, dont
**949 porteurs d une passe de film**. Cote credit : `killer_victim_pairs` deduplique (c est
l objet meme que le producteur credit reproduit — meme source, meme algorithme, meme tolerance de
5 ms). Cote film : `match_kill_events_latest` restreinte a `read_path IN ('marche','scan')`.

### 2.1 Les deux clefs candidates, aux quatre tolerances demandees

Sur le perimetre film (949 matchs) : **74 569 lignes de film**, **98 662 morts de credit**.

| tolerance | clef | film apparie | film orphelin | credit apparie | credit seul |
|---|---|---|---|---|---|
| 0 ms | match + victime + tueur + dt | 72 296 (96,9 %) | 2 273 (3,1 %) | 72 296 (73,3 %) | 26 366 (26,7 %) |
| 0 ms | **match + dt** (RETENUE) | **73 589 (98,7 %)** | **980 (1,3 %)** | **73 589 (74,6 %)** | **25 073 (25,4 %)** |
| +-50 ms | match + victime + tueur + dt | 72 304 | 2 265 | 72 304 | 26 358 |
| +-50 ms | match + dt | 73 618 | 951 | 73 848 | 24 814 |
| +-200 ms | match + victime + tueur + dt | 72 304 | 2 265 | 72 304 | 26 358 |
| +-200 ms | match + dt | 73 645 | 924 | 74 234 | 24 428 |
| +-1 s | match + victime + tueur + dt | 72 304 | 2 265 | 72 304 | 26 358 |
| +-1 s | match + dt | 73 792 | 777 | 76 073 | 22 589 |

### 2.2 CE QUE CE TABLEAU DIT, ET QUI TRANCHE

**(a) La tolerance ne sert a rien.** Sur la clef a quatre colonnes, passer de 0 ms a 1 SECONDE
gagne **8 lignes sur 74 569** (0,01 %), et le plateau est atteint des 50 ms. Il n existe pas de
population de morts « decalees de quelques millisecondes » : c est la consequence directe du §1.

**(b) La tolerance DETRUIT la bijection.** A tolerance 0 sur la clef `match + dt`, le nombre de
lignes de film appariees et le nombre de morts de credit appariees sont **egaux au chiffre pres**
(73 589 = 73 589) : l appariement est une bijection stricte. Des 50 ms ils divergent (73 618 cote
film pour 73 848 cote credit), et l ecart se creuse a 1 s (73 792 pour 76 073). Cette divergence
est la signature d une correspondance devenue multiple : une ligne de film qui capture plusieurs
morts de credit. **Toute tolerance non nulle achete quelques appariements de plus au prix de
l unicite** — et l unicite est ce qui rend l enrichissement sur.

**(c) La clef doit etre `(match_id, time_ms)`, pas la clef a quatre colonnes.** Mesure
d unicite sur le perimetre film : `(match_id, time_ms)` est **strictement unique des deux
cotes** — 98 662 clefs pour 98 662 morts de credit, 74 569 clefs pour 74 569 lignes de film,
**zero collision**. Les identites n ajoutent donc aucun pouvoir discriminant ; elles ne font que
RETIRER des appariements valides quand le film n a pas su resoudre un xuid.

**(d) L identite ne contredit JAMAIS l identite.** Sur les 73 589 lignes appariees par la clef
courte : **0 divergence de xuid de victime, 0 divergence de xuid de tueur.** Le seul ecart est
une absence — 631 victimes et 754 tueurs pour lesquels le film n a pas resolu de xuid. C est
precisement la population que la clef a quatre colonnes perdait (1 293 lignes, soit la totalite de
son deficit contre la clef courte). L identite reste donc un CONTROLE (elle doit concorder quand
les deux cotes la portent), elle n est pas un critere de jointure.

### 2.3 L APPARIEMENT RETENU

> **Clef : `(match_id, time_ms)`. Tolerance : ZERO.**
> **Controle obligatoire : quand les deux cotes portent un xuid, il doit concorder** — une
> divergence est une erreur qui doit echouer bruyamment, pas une ligne a ecarter en silence.
> Justification : les deux cotes lisent le meme flux avec le meme parseur (§1) ; la clef est
> mesuree unique des deux cotes ; c est la seule tolerance ou l appariement est une bijection.

---

## 3. LE SORT DES FILM-ORPHELINS — LES GARDER, ET VOICI POURQUOI

980 lignes de film (1,3 %) n ont aucune mort de credit en face. **Aucune n est une anomalie du
decodeur.** Anatomie complete :

| population | n | ce que c est |
|---|---|---|
| `read_origin = 'bot'` (victime sans xuid) | 819 | un humain tue un BOT |
| `read_origin = 'tueur-bot'` | 149 | un BOT tue un humain |
| `credit-concordant` / `source-victime`, deux xuids | 13 | humain contre humain, non appariee par le credit |

**Le controle qui ferme la question** : sur les 980 orphelins, **0 ne tombe sur un instant sans
aucun evenement de l API**. 831 coincident avec un `kill` de l API, 158 avec un `death`. Autrement
dit chaque orphelin correspond a un evenement REEL que l API porte — simplement pas sous une forme
que l appariement kill<->death du credit sait transformer en couple.

La raison est structurelle et deja documentee : **le kill-feed de l API est HUMAIN SEUL** (un bot
n a pas de xuid, sa mort ne produit aucun evenement). Quand un humain tue un bot, l API porte le
KILL et pas la MORT ; quand un bot tue un humain, elle porte la MORT et pas le KILL. Dans les deux
cas, aucun couple — donc aucune ligne de credit. `killer_victim_pairs` ne contient d ailleurs
**aucune ligne de bot** en production, cote tueur comme cote victime.

### DECISION : LES ORPHELINS SONT CONSERVES

**Le credit fait foi sur l existence des morts D HUMAINS. Il ne fait pas foi sur les morts que sa
source ne peut structurellement pas representer.** Rejeter les orphelins jetterait 968 morts de
bot mesurees, au motif qu une source aveugle aux bots ne les confirme pas — c est-a-dire traiter
une absence de mesure comme une mesure d absence, exactement la faute que le DDL de
`match_kill_events` interdit partout ailleurs.

Cette decision ne rouvre rien : l arbitrage (A) du DDL et de la phase 2.3 est deja
**« les bots sont DANS la table, ce sont LES LECTEURS CARRIERE qui les filtrent »**. La conserver
ici est la seule option coherente avec lui.

Les 13 orphelins humain-contre-humain sont conserves au meme titre : ils portent deux xuids, un
evenement API existe a leur instant, et ils representent 0,017 % des lignes de film. Ils sont le
symetrique attendu de la bijection greedy du credit (une mort deja consommee par un autre kill).
**Ils sont a surveiller par un compteur, pas a rejeter** : c est la seule population des trois
dont le mecanisme n est pas demontre.

---

## 4. LES TROIS ETATS DE L ASSISTANT — VERIFICATION

Rappel du DDL : `assist_known = FALSE` = ON NE SAIT PAS ; `TRUE` + gamertag NULL = mesure « pas
d assistant » ; `TRUE` + gamertag renseigne = l assistant nomme. Confondre les deux premiers
fabriquerait des faits jamais observes.

Etat actuel de la table, par producteur :

| producteur | etat | n |
|---|---|---|
| CREDIT (`highlight-events`) | 1. on ne sait pas | 50 125 |
| FILM | 1. on ne sait pas | 2 341 |
| FILM | 2. mesure : PAS d assistant | 49 147 |
| FILM | 3. assistant NOMME | 23 081 |

Ce que le film apporte reellement aux **73 589** lignes appariees :

| apport | n | part |
|---|---|---|
| `source_tag` (l arme) | **73 589** | **100 %** |
| assistant NOMME | 23 018 | 31,3 % |
| mesure « pas d assistant » | 49 050 | 66,7 % |
| film muet sur l assistant (etat 1) | 1 521 | 2,1 % |
| `diverges = TRUE` | 1 294 | 1,8 % |

**La regle de fusion qui preserve les trois etats — et la combinaison a interdire :**

1. Mort de credit **non enrichie** -> `assist_known = FALSE`, tous les champs d assistant NULL.
   C est l etat 1, inchange, et c est deja ce que le producteur credit ecrit aujourd hui.
2. Mort de credit **enrichie** -> les champs d assistant sont **recopies VERBATIM** de la ligne de
   film, y compris quand celle-ci est en etat 1 (les 1 521 lignes ou le film est muet). Le film
   reste l unique autorite sur ce qu il a ou n a pas observe.
3. **INTERDIT : deriver `assist_known` d autre chose que de la ligne de film.** Un defaut a `TRUE`
   sur une mort non enrichie fabriquerait 60 297 « mesures : pas d assistant » jamais observees.
   C est la seule combinaison que la fusion peut fabriquer, et elle doit etre rendue impossible a
   l ecriture, pas documentee.

**Verification que rien n est fabrique** : la fusion ne CREE aucun champ d assistant. Chaque ligne
enrichie herite d une ligne de film existante (bijection stricte, §2.2b), chaque ligne non enrichie
garde l etat 1 du producteur credit. Le nombre de lignes en etat 2 apres inversion est donc
exactement le nombre de lignes de film en etat 2 aujourd hui, ni plus ni moins. **Aucune
combinaison nouvelle n apparait.**

Meme raisonnement, meme conclusion, pour `source_tag`/`source_category`/`diverges`/parts de
degats : NULL sur une mort non enrichie (« non mesure »), valeur du film sinon. `diverges` en
particulier reste NULL hors enrichissement — ecrire `FALSE` se lirait « mesure : pas de
divergence ».

---

## 5. LE CRITERE DE BASCULE — ATTEINT

Le critere ecrit en session 3 : `lignes_passe / morts_api >= 98,4 %` sur le meme perimetre.

**Un prealable de methode, decouvert en le verifiant** : l oracle brut (`COUNT(*)` des evenements
`death`) **sur-compte**. Sur les 394 matchs sans film, `highlight_events` porte **15 120 groupes
(match, instant, joueur) en double exact** — 50 857 lignes pour 35 737 morts distinctes. Compare a
cet oracle gonfle, le credit tombait a 69,3 % sur ce perimetre et paraissait tres inegal. **Sur
l oracle DEDUPLIQUE, il est uniforme** :

| perimetre | oracle dedup. | base credit | couverture |
|---|---|---|---|
| A — 949 matchs a film | 100 172 | 98 662 | **98,5 %** |
| B — 394 matchs sans film | 35 737 | 35 224 | **98,6 %** |
| **TOTAL — 1 343 matchs** | **135 909** | **133 886** | **98,5 %** |

Le 98,4 % de la session 3 etait donc juste (il portait sur A, ou les doublons ne pesent que
0,1 %), mais il n etait pas la propriete d un perimetre : **le credit tient 98,5 % PARTOUT**.

**Etat cible de la table apres inversion :**

| grandeur | valeur |
|---|---|
| base credit (couples dedupliques, 1 343 matchs) | 133 886 |
| + film-orphelins conserves (§3) | + 980 |
| **= `match_kill_events` cible** | **134 866** |
| table actuelle (`match_kill_events_latest`) | 124 694 |
| **gain net** | **+10 172 morts** |
| dont enrichies par le film (arme + assistant) | **73 589** (54,6 % de la table) |

Sur le perimetre film, la passe publie (98 662 + 158) / 100 172 = **98,7 %** de l oracle
deduplique en ne comptant que les morts qu il contient, et 99 642 lignes en comptant les 819 morts
de bot qu il ne peut pas porter. **Le critere est franchi dans les deux lectures.**

Aujourd hui, a titre de comparaison : sur 949 matchs a film, **389 perdent des morts** a la
bascule actuelle (24 919 morts perdues) et 129 seulement en gagnent. Apres inversion, aucun match
ne peut en perdre — c est un invariant, et le §6.5 le pose en test.

**Halo 5 : l inversion est un no-op, verifie.** 268 337 lignes, **0 ligne de film** (le format des
films H5 est autre), `read_path = 'kill-feed'` sur la totalite. Le titre est deja integralement
credit-base. La conception ne le touche pas — et c est le comportement title-agnostic attendu : un
titre sans film n a rien a enrichir.

---

## 6. PLAN D IMPLEMENTATION PROPOSE

> Ordre indicatif ; a passer par `plan-review` avant execution. Chaque lot se clot sous
> `plan-execution` (gate + statut de chaque item + entree thought_log).

### 6.1 Ce qui NE change pas — et c est l essentiel

- **Aucune migration de schema.** `match_kill_events` porte deja toutes les colonnes necessaires ;
  la vue `_latest` (une passe par match) reste inchangee et reste le seul chemin de lecture.
- **Aucun changement au decodeur** (`filmdec`, `killsource/decode.go`, calibration, goldens). La
  fusion se place APRES le decodage, sur son resultat.
- **Aucun re-decodage de film.** L enrichissement des 949 matchs est **deja en base**, indexe par
  `(match_id, time_ms)`. La recomposition est SQL -> SQL. C est le point qui rend cette bascule bon
  marche : ni les 23 Go de films, ni les 3 a 11 heures de decodage, ni les tokens.

### 6.2 Le fusionneur — une fonction PURE, une seule copie

Nouveau `internal/sync/killcollector/merge.go`, a cote de `BuildCreditBatch` et
`BuildKillSourceBatch` qui y vivent deja :

```
MergeCreditAndFilm(base persist.KillSourceBatch, film persist.KillSourceBatch) persist.KillSourceBatch
```

- indexe `film.Deaths` par `TimeMS` (clef unique, §2.2c) ;
- pour chaque mort de `base` : si une ligne de film porte le meme `TimeMS`, **controler la
  concordance des xuids presents des deux cotes** (erreur si divergence), puis recopier les champs
  d enrichissement (`SourceTag`, `SourceCategory`, `Assist*`, `*DamagePct`, `Diverges`,
  `ReadPath`, `ReadOrigin`) ; sinon laisser l etat credit (§4) ;
- **ajouter** les lignes de film non consommees (les 980 orphelins), telles quelles ;
- `Publishable` de la passe : celui de la passe de film quand il y en a une, sinon celui du credit.

Pure, sans DB, sans contexte : testable seule. `read_path`/`read_origin` restent **par ligne** —
c est ce qui garde `persist.FilmReadPaths` operant et permet a un lecteur de savoir, mort par
mort, si l arme a ete mesuree.

### 6.3 Les deux producteurs — la preseance s inverse

| producteur | aujourd hui | apres |
|---|---|---|
| `persist/kill_events_credit.go` (LIVE) | `matchCoveredByFilmPass` -> **refuse d ecrire** | lit l enrichissement de la passe film courante depuis `match_kill_events_latest`, le fusionne a sa base credit, ecrit une passe RECOMPOSEE |
| `sync/killcollector/credit.go` (backfill) | `covertParUnFilm` -> **refuse d ecrire** | idem |
| `sync/killcollector` (film) | ecrit sa propre passe, qui remplace tout | ne publie plus seul : son resultat est fusionne sur la base credit du match avant ecriture |

Les deux fonctions de refus (`matchCoveredByFilmPass`, `covertParUnFilm`) ne sont PAS supprimees :
elles deviennent le detecteur « ce match porte-t-il un enrichissement a preserver ». Meme requete,
sens inverse. `FilmReadPaths` reste la liste d autorite, testee en positif — l acquis de J4R-4 est
conserve.

**Point d attention** : la recomposition live coute une lecture supplementaire par match
synchronise (une requete indexee sur `(match_id, written_at)`, l index existe deja). A mesurer au
gate, pas a supposer.

### 6.4 La reprise — une migration, pas un backfill

Sur le modele de `shared_kill_events_from_pairs_v1` (`migration/steps_shared_kill_events_from_pairs.go`) :
une migration `shared_kill_events_credit_base_v1`, INSERT-only, idempotente, qui pour chaque match
ecrit une passe = base credit (depuis `killer_victim_pairs`) + enrichissement de la passe courante
(depuis `match_kill_events_latest`) + orphelins.

Elle ne demande **aucun film** : elle peut donc etre jouee en production telle quelle, ce qui est
l argument operationnel decisif du §6.1. Le re-backfill par la CLI
(`levelup backfill-killsource`) reste utile pour les matchs dont le film n a jamais ete decode —
il n est PAS un prealable a la bascule.

Piege connu, a ne pas re-decouvrir : le decoupeur SQL de `internal/sync/schema.go` n est pas
conscient des commentaires (il coupe sur chaque `;`, meme dans un `--`) ; celui de `migration`
l est. Ecrire la migration dans `migration`.

### 6.5 Tests — dont LE garde-rail qui manquait

1. **Unitaires du fusionneur** : les trois etats de l assistant preserves ; un orphelin conserve ;
   un xuid divergent rejete ; la clef exacte (une ligne de film a `t+1` n enrichit PAS).
2. **LE GARDE-RAIL D INVARIANT — celui dont l absence a coute la session 3** :
   *une passe ecrite pour un match ne peut jamais porter moins de morts que la base credit de ce
   match.* C est exactement la propriete qui, testee, aurait fait rougir la bascule au lieu de la
   laisser effacer 25 697 morts en silence. A poser cote persister (refus a l ecriture) ET en test.
3. **Mutations qui doivent rougir** : retirer la recopie de `source_tag` ; defauter
   `assist_known` a TRUE ; retirer l ajout des orphelins ; elargir la clef a une tolerance.
4. **Integration `go test -tags=integration -p 1 ./...`** — OBLIGATOIRE, persist et sync sont
   touches (anti-ART, ADR 0019/0030).
5. **Gate avant/apres sur donnees reelles**, sur une COPIE : les 11 mesures de GATE 2 (session 3),
   plus la couverture par match, plus le compte des trois etats de l assistant.

### 6.6 Ordre, et ce qui suit

`6.2 fusionneur` -> `6.3 producteurs` -> `6.4 migration de reprise` -> `6.5 gate sur copie` ->
**ALORS SEULEMENT** la bascule des 20 sites de lecture (phase 2 du plan de branchement), puis le
retrait de `killer_victim_pairs` (chemin en 7 etapes du DDL, inchange). Le seed de demonstration
(`ops/seed_demo.go`) et `cmd/rebuild_mp` restent les deux pieges de la bascule, deja recenses.

---

## 7. DECOUVERTES CONSIGNEES — NON TRAITEES (regle « zero fix opportuniste »)

1. **Le garde-fou `assist_extra_count` A BOUGE.** Le DDL ecrit « 0 partout aujourd hui. S il
   bouge, c est le declencheur de migration vers une table fille ». Mesure du 2026-08-02 :
   **5 lignes a 1**, sur 5 matchs distincts. Le declencheur est franchi au sens litteral. Volume
   ridicule (5 sur 124 694), donc pas une urgence — mais la doctrine dit de statuer, pas
   d ignorer. **A arbitrer separement.**
2. **`highlight_events` porte des doublons exacts sur le perimetre B** : 15 120 groupes
   `(match_id, time_ms, xuid)` en DOUBLE exact sur les 394 matchs sans film (20 617 groupes sains).
   Sans consequence sur le credit (l appariement deduplique de fait), mais **tout comptage
   d evenements par `COUNT(*)` sur cette table sur-compte de ~42 % sur ce perimetre** — et c est
   ce qui a fait paraitre le credit a 69,3 % au lieu de 98,6 %. A instruire : defaut d insertion
   ou double chargement.
3. **`(match_id, time_ms)` n est PAS unique sur Halo 5** : 268 330 clefs pour 268 337 lignes
   (7 collisions). Sans effet ici (aucun film H5, donc aucune jointure d enrichissement), mais
   **l unicite de la clef est une propriete MESUREE du corpus Infinite, pas une garantie de
   schema** — le fusionneur doit se comporter proprement si elle est violee (rejeter, pas choisir
   au hasard).
4. **1 294 lignes `diverges = TRUE`** parmi les appariees (1,8 %) : la source du degat designe un
   autre responsable que le credit. C est l information attendue, pas un defaut — mais aucun
   lecteur ne l expose aujourd hui.

---

## 8. CE QUI RESTE A DECIDER AVANT D IMPLEMENTER

1. **Faut-il aller au-dela des couples, jusqu aux MORTS ?** La base credit proposee est la liste
   des COUPLES (133 886 = 98,5 % de l oracle). Les 2 023 morts restantes n ont aucun kill
   appariable (tueur bot, suicide, chute, environnement). Le schema sait les porter
   (`feed_killer_gamertag` est nullable) et cela porterait la couverture a 100 %. **Recommandation :
   NON pour cette bascule** — ce serait changer la definition de ce que la table contient EN MEME
   TEMPS qu on change sa preseance, donc rendre le gate avant/apres illisible. A traiter comme un
   lot distinct, apres.
2. **Le sort des 13 orphelins humain-contre-humain** : conserves (§3), mais faut-il un compteur
   dedie (ADR 0009) ou suffit-il du `read_origin` existant ? Recommandation : un compteur — c est
   la seule des trois populations dont le mecanisme n est pas demontre.
3. **Le cout de la recomposition live** (§6.3) : a mesurer au gate. S il s avere non negligeable
   sur un cycle de sync complet, la lecture d enrichissement peut etre restreinte aux matchs
   effectivement porteurs d une passe de film.
4. **Le decoupage en releases pour la production.** Le plan J4 prevoit deux temps (release 1 =
   producteurs + migration ; release 2 = bascule des lecteurs). La migration du §6.4 ne demandant
   aucun film, elle tient entierement dans la release 1 — a confirmer au moment du decoupage.

---

## ANNEXE — REPRODUIRE LES MESURES

Base : COPIE de `data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb` (etat du
2026-08-02 13:18). Outil : `go run ./apps/go-api/cmd/diag_q <db> "<sql>"` (ouvre en
`access_mode=read_only`). Aucun serveur ne tenait la base pendant les mesures.

Definitions employees partout :

```sql
-- perimetre film
WITH perim AS (SELECT DISTINCT match_id FROM match_kill_events_latest
               WHERE read_path IN ('marche','scan'))
-- cote film
F AS (SELECT * FROM match_kill_events_latest WHERE read_path IN ('marche','scan'))
-- cote credit (deduplique)
C AS (SELECT match_id, time_ms FROM killer_victim_pairs GROUP BY 1, 2)
-- oracle DEDUPLIQUE (le brut sur-compte, cf. decouverte n2)
D AS (SELECT DISTINCT match_id, time_ms, xuid FROM highlight_events
      WHERE LOWER(event_type) = 'death')
```

Piege de mesure a ne pas reproduire : definir le cote credit par
`SELECT DISTINCT match_id, killer_xuid, victim_xuid, time_ms, killer_gamertag, victim_gamertag`
gonfle le jeu (plusieurs orthographes de gamertag pour une meme clef) et fait sortir un nombre
d apparies SUPERIEUR au cardinal de la source — c est ce qui a produit un « 85 150 apparies pour
74 569 lignes de film » lors d une premiere passe. Le `DISTINCT` doit porter sur la clef, pas sur
les attributs.

---

## 9. EXECUTION — 2026-08-03, §6 CLOS

> Session EXECUTEUR 1/2 (implementation). La bascule des lecteurs est la session 2/2 et n a PAS
> ete faite. Perimetre ferme au §6 ; toute decouverte hors perimetre est au §10.

### 9.1 Statut de chaque item du §6

| item | statut | ou |
|---|---|---|
| 6.1 aucune migration de schema, aucun changement au decodeur, aucun re-decodage | `[x]` | tenu : 0 ligne touchee dans `filmdec`/`killsource` ; la reprise est SQL -> SQL en **14,5 s** |
| 6.2 fusionneur PUR, une seule copie | `[x]` | `persist/kill_events_merge.go` — **PAS** `killcollector` (voir 9.3) |
| 6.3 les trois producteurs inversent la preseance | `[x]` | `persist/kill_events_credit.go` (live) · `killcollector/credit.go` (backfill) · `killcollector/collector.go` (film) |
| 6.4 migration de reprise, INSERT-only, idempotente, sans film | `[x]` | `migration/steps_shared_kill_events_credit_base.go` |
| 6.5-1 unitaires du fusionneur | `[x]` | 8 tests, **9 mutations verifiees rouges** |
| 6.5-2 **LE garde-rail d invariant** | `[x]` | DEUX garde-rails, pas un (voir 9.4) — **2 mutations rouges** |
| 6.5-3 mutations qui doivent rougir | `[x]` | 17 au total : 9 fusionneur + 2 persister + 6 migration |
| 6.5-4 integration `-tags=integration -p 1 ./...` | `[x]` | 0 echec |
| 6.5-5 gate avant/apres sur donnees reelles, sur COPIE | `[x]` | tableau 9.2 |
| 6.6 ordre d execution | `[x]` | 6.2 -> 6.3 -> 6.4 -> gate. La bascule des lecteurs N EST PAS faite. |
| §8-2 compteur dedie pour les 13 orphelins humain-contre-humain | `[x]` | `killsource_orphelins_film_humain_contre_humain` — mesure **13**, au chiffre pres |
| §8-3 cout de la recomposition live | `[x]` | la sonde `matchCoveredByFilmPass` reste en tete : la lecture complete ne suit que sur les matchs a film. Passe complete de 1 926 matchs, hors ligne : **5 min 1 s** |
| §7-1 `assist_extra_count` reconcilie | `[x]` | voir 9.5 |
| §8-1 aller jusqu aux MORTS (2 023 restantes) | `[!]` | NON — recommandation du document suivie : lot distinct, APRES la bascule. Changer la definition du contenu en meme temps que la preseance rendrait le gate illisible. |
| §8-4 decoupage en releases | `[~]` | la migration ne demandant aucun film, elle tient dans la release 1 : **confirme**. Le decoupage exact se decide en J5. |

### 9.2 GATE AVANT/APRES — base Halo Infinite de developpement (COPIE)

| mesure | avant | apres |
|---|---|---|
| **lignes servies** (`match_kill_events_latest`) | 124 694 | **134 866** |
| matchs servis | 1 343 | 1 343 |
| lignes de film (arme mesuree) | 74 569 | 74 569 |
| assistant etat 1 — ON NE SAIT PAS | 52 466 | 62 638 |
| assistant etat 2 — mesure : PAS d assistant | 49 147 | **49 147** |
| assistant etat 3 — assistant NOMME | 23 081 | **23 081** |
| `diverges = TRUE` | 1 296 | 1 296 |
| victime sans xuid | 1 450 | **819** |
| `assist_extra_count >= 1` | 5 | 5 |
| `killer_victim_pairs` (doit rester INTACTE) | 250 139 | 250 139 |
| **matchs perdant des morts** | **389** | **0** |

**Ce que ce tableau prouve, ligne par ligne.** Les etats 2 et 3 de l assistant sont IDENTIQUES au
chiffre pres : la fusion ne fabrique aucun fait — c est exactement la verification annoncee au §4.
L etat 1 augmente de +10 172, c est-a-dire des morts de credit ajoutees, et de rien d autre. Les
victimes sans xuid tombent de 1 450 a 819 : les 631 lignes de film dont le decodeur n avait pas
resolu la victime portent desormais le xuid du credit — **631 morts qui etaient injoignables par
tout agregat carriere le sont devenues**, et c est exactement la population que le §2.2d designait.
Les 819 qui restent sont les morts de BOT, que le credit ne peut structurellement pas nommer.

**Couverture atteinte** : base credit 133 886 / oracle deduplique 135 909 = **98,5 %**, et
0 match ne publie moins que sa base. Sur le seul perimetre film : 98 662 / 100 172 = **98,5 %**.

**Halo 5 — no-op VERIFIE sur la donnee, pas seulement sur l absence de film** : 268 337 -> 268 337,
**0 ligne perdue, 0 ligne ajoutee**, 0 ligne de film.

### 9.3 CE QUI S ECARTE DU DOCUMENT, ET POURQUOI

1. **Le fusionneur vit dans `persist`, pas dans `sync/killcollector`** (§6.2). Le §6.2 le placait
   a cote de `BuildCreditBatch` ; c est IMPOSSIBLE : le producteur LIVE vit dans `persist`, et
   `persist` ne peut pas importer `killcollector` (qui l importe deja). `persist` est par ailleurs
   le proprietaire de `KillSourceBatch`, type d entree ET de sortie de la fusion.
2. **LA BASE CREDIT SE DEDUPLIQUE SUR L IDENTITE `(time_ms, victime, tueur)`, PAS SUR LA CLEF.**
   Mesure du 2026-08-03, et c est le seul decoupage qui rende les deux cardinaux justes :
   - dedup sur `(match_id, time_ms)` seul : 268 330 sur Halo 5 — **PERD 7 morts REELLES** (sept
     instants portent deux victimes distinctes) ;
   - dedup sur toutes les colonnes (ce que fait la reprise `from_pairs` existante) : **145 974**
     sur Halo Infinite — **FABRIQUE 12 088 doublons** (une meme mort sous deux orthographes de
     gamertag ; verifie : ces 12 088 clefs portent UN SEUL xuid de victime et UN SEUL de tueur) ;
   - dedup sur l identite : **133 886** et **268 337**. Les deux cibles.

   La clef `(match_id, time_ms)` reste la clef d APPARIEMENT ; l identite est la clef de
   DEDUPLICATION. Ce sont deux choses differentes, et le document ne les distinguait pas.
3. **Le producteur credit-seul deduplique lui aussi.** Il tirait **50 125** morts de
   `highlight_events` la ou l oracle deduplique en compte 35 224 — 140 % — a cause des
   15 120 doublons exacts de la decouverte n2. Sans cette deduplication, la cible 134 866 etait
   inatteignable et l invariant « jamais moins que la base » devenait ininterpretable.
4. **`written_at` de la migration est ecrit en UTC**, pas via `now()`. `CAST(now() AS TIMESTAMP)`
   rend l heure LOCALE (+2 h) alors que tous les producteurs Go ecrivent `time.Now().UTC()` : une
   passe de migration aurait battu toute passe fraiche pendant deux heures. Defaut PREEXISTANT
   dans `steps_shared_kill_events_from_pairs.go` — consigne au §10, non traite.
5. **`killscope` recoit les deux voies de film.** La migration en aurait fait une TROISIEME copie
   (apres `killsource.Path` et `persist.FilmReadPaths`) ; la regle du depot impose de centraliser
   a la 3e ET de poser un garde-rail. Le decodeur n a PAS ete touche : il reste le proprietaire du
   type, et l egalite est verrouillee par `killcollector.TestFilmReadPathsEgalesAuDecodeur` (plus
   un test d exhaustivite qui rougit si le decodeur ajoutait une 3e voie). Le ratchet
   `archlint/no_raw_kill_scope_literal_test.go` police desormais `marche`/`scan`.
   **Dette H3 refermee.**

### 9.4 LE GARDE-RAIL D INVARIANT — DEUX, PAS UN

Le §6.5-2 en demandait un ; il en fallait deux, parce qu ils attrapent des fautes differentes :

| garde | ou | ce qu il attrape |
|---|---|---|
| `KillSourceBatch.CreditBaseCount` | `validateKillSourceBatch` | une FUSION qui perd des lignes. Pose par le fusionneur lui-meme, jamais a la main. |
| sonde `killer_victim_pairs` | `PersistPass` | un producteur qui **ne fusionne pas du tout** — celui-la laisserait `CreditBaseCount` a zero et passerait le premier. C est LE cas de la session 3. |

La sonde ne tourne pas sur le chemin live (les couples du match n y sont pas encore ecrits, la
mesure serait celle du cycle precedent) : la base y est EN MAIN, donc couverte par le premier.
Test `TestSondeRefuseUnePasseDeFilmQuiRemplaceLaBaseCredit` : le scenario de la session 3, rejoue.
La migration porte le meme invariant, RELU APRES ECRITURE (`verifierPlancherReprise`).

### 9.5 `assist_extra_count` — RECONCILIE

Mesure du 2026-08-03 : **5 lignes a 1 sur 124 694 (0,004 %), aucune au-dela de 1**, sur 5 matchs
distincts. Le declencheur litteral (« 0 partout ; s il bouge, migrer ») etait donc franchi — et
pourtant l hypothese de schema TIENT, parce que ce compteur ne mesure pas « une mort porte deux
assistants » mais « deux kill-events ATTACHES a la meme mort nomment des assistants differents »
(portee que `killsource.Assist.Extra` delimite lui-meme). Un seuil a zero sur une quantite
d APPARIEMENT ne pouvait pas rester vrai indefiniment.

**Declencheur qui le remplace, ecrit dans le DDL** : migrer vers une table fille des que (a) UNE
ligne porte `assist_extra_count >= 2`, OU (b) la part des lignes a `>= 1` depasse 0,1 % de
`match_kill_events_latest` (125 lignes aujourd hui, 25x la mesure). Le log du persister et la
restitution de la CLI ont ete recrits : ils annoncaient « l hypothese est en defaut, migrer », ce
qui est FAUX a l echelle mesuree — une alerte qui crie a chaque passe cesse d etre lue.

### 9.6 CROISEMENT DES DEUX IMPLEMENTATIONS — elles concordent au chiffre pres

La migration SQL et le fusionneur Go sont deux implementations INDEPENDANTES de la meme regle.
Jouees separement sur deux copies neuves de la meme base, puis comparees ligne a ligne sur
`(match_id, time_ms, victime, tueur, source_tag, assist_known, assistant, part_tueur, diverges)` :

```
sql_lignes  go_lignes  dans_sql_pas_go  dans_go_pas_sql
   134866     134866          0                0
```

**Zero ecart sur 134 866 lignes.** Ce n est pas une verification de plus : c est la seule qui
teste la REGLE plutot que son ecriture.

---

## 10. DECOUVERTES DE L EXECUTION — CONSIGNEES, NON TRAITEES

1. **`CAST(now() AS TIMESTAMP)` rend l heure LOCALE en DuckDB** (+2 h ici) alors que tous les
   producteurs Go ecrivent `time.Now().UTC()`. `steps_shared_kill_events_from_pairs.go` (la
   reprise de J4) ecrit donc un `written_at` DEUX HEURES DANS LE FUTUR : sa passe bat toute passe
   fraiche pendant deux heures. Sans consequence aujourd hui (elle n ecrit que des matchs sans
   aucune passe), mais c est un piege arme. **A traiter avec H1-H6.**
2. **La reprise `from_pairs` ne deduplique PAS a l identite** : son `SELECT DISTINCT` porte sur
   toutes les colonnes et rendrait **145 974** lignes la ou il y a 133 886 morts (+9 %). Elle n a
   jamais ecrit sur la base de dev (tous les matchs y etaient deja couverts) — le defaut n a donc
   jamais eu de consequence mesuree, mais il attend une base neuve. La reprise credit-base la
   precede desormais dans l ordre canonique, donc `from_pairs` ne trouve plus rien a reprendre :
   **la question de son retrait reste ouverte** et n a pas ete tranchee ici.
3. **`publishable` devient un attribut de la PASSE FUSIONNEE, celui du film** (regle du §6.2,
   suivie). Consequence mesuree : sur les **366 matchs** dont la passe de film est non publiable
   (BTB, marge de bijection nulle), les lignes de credit ajoutees heritent de
   `publishable = FALSE` alors qu elles seraient publiables sans film. Aucun impact aujourd hui —
   **aucun lecteur ne lit cette colonne** (phase 2.3 du plan de branchement). **A trancher a la
   bascule des lecteurs**, qui est le lot qui donnera un sens a cette colonne : soit on l accepte,
   soit `publishable` devient per-ligne comme `read_path`.
4. **La sonde de plancher coute une lecture de `killer_victim_pairs` par passe hors ligne.** La
   table n a aucun index (surface ART), donc c est un scan complet. Passe de 1 926 matchs :
   5 min 1 s, dominee par cette sonde et par les requetes du producteur credit. Acceptable hors
   ligne ; **a re-mesurer le jour ou cette passe tournerait dans un cycle de sync**.
5. **3 matchs portent des `highlight_events` kill/death SANS aucun couple** dans
   `killer_victim_pairs` (1 346 contre 1 343). C est ce qui justifie que le producteur credit-seul
   garde `highlight_events` comme source plutot que de basculer sur `killer_victim_pairs` : la
   bascule perdrait ces 3 matchs. Volume negligeable, mecanisme non instruit.
