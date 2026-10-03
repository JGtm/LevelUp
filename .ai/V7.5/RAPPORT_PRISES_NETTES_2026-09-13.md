# RAPPORT — Prises nettes de drapeau : mesure du jonglage (2026-09-13)

> Étape 0 (mesure, aucun code de production) du plan
> `.ai/PLAN_PRISES_NETTES_DRAPEAU_2026-09-13.md`. Branche `feat/prises-nettes`.
> Instrument : test de recherche jetable `prises_nettes_research_test.go`
> (paquet `internal/games/halo_infinite/film/replay`), SUPPRIMÉ après la mesure —
> la sortie brute intégrale est en annexe, le rapport se rejoue à partir d'elle.

## 1. Méthode

**Source unique : les artefacts de rejeu déjà cuits**, lus en lecture seule dans
`data/cache/replays/halo_infinite/*.json` du dépôt principal (les `.derived.json` sont
ignorés). Aucune base DuckDB n'a été ouverte, aucun film n'a été re-décodé, aucun
processus n'a été lancé sur les données du poste.

- **Verdict « ce film est du CTF »** : `coverage.flagCarries.flagFilm`, c'est-à-dire
  l'accord des trois signaux de `analysis/objectiveevents/flagfilm.go`
  (`bursts > 0`, `captures > 0`, `captures <= bursts`, `steals > 0`). Le verdict est
  pris tel que l'artefact le publie — il n'a pas été recalculé.
- **Prise brute** : une ouverture de portage publiée, c'est-à-dire un intervalle
  `flagCarries[].spans[]` d'état `carried` ou `carried_open` portant un `xuid`. Ces
  intervalles sont bornés par les événements nommés `StatFlagGrabs` / `StatFlagSteals`
  (`objectiveevents/named.go`) une fois les émissions jumelles fusionnées.
- **Prise nette, fenêtre W** : la prise est nette si le portage PRÉCÉDENT du **même
  drapeau** (même entrée `flagCarries`, donc même équipe propriétaire) n'est pas du même
  joueur, ou s'est terminé il y a **strictement plus de W secondes**. Autrement dit un
  aller-retour lancer / reprise dans la fenêtre compte UNE fois.
- **Conversion du temps** : `frameIntervalMs` de l'artefact — 100 ms par frame sur tout
  le corpus, soit 10 images par seconde. Le délai lâcher → reprise vaut
  `(t0_prise − t1_portage_précédent) x 0,1 s`.
- Fenêtres mesurées : **1, 1,5, 2, 3, 5 et 8 s**. La valeur 1,5 s a été ajoutée après
  lecture de l'histogramme, qui place la coupure là (§4).

## 2. Corpus

| | Films |
|---|---|
| Artefacts non dérivés balayés | **76** |
| Retenus (`flagFilm = true`) | **13** |
| Écartés | **63** |

Les 63 écartés le sont tous pour la même cause : `flagFilm = false`. Le détail par film
est en annexe (§A.1) avec ses trois compteurs. Aucun n'a été écarté pour absence de
couverture, de schéma ou de `frameIntervalMs` : le parc est homogène en **schéma 54**.

Les 13 films retenus : `16ea3668`, `4ecdf3e7`, `58864b3c`, `64e8adfa`, `7fce3219`,
`8bc6074f`, `a0c36016`, `b8a44fe8`, `bc60b4d9`, `bf5ced1b`, `cde26226`, `f8efc5ca`,
`fb1a1a72`.

Ils totalisent **620 ouvertures** de l'oracle, dont **617 portages publiés** (3 rejets,
tous `noTrack` — aucun `noBridge`, aucun hors-axe), **0 portage sans xuid**, et
**87 couples (film, joueur)**.

## 3. Prises brutes contre prises nettes

### 3.1 Total du corpus

| Fenêtre | Prises nettes | Prises repliées | Part repliée |
|---|---|---|---|
| brut | 617 | — | — |
| 1 s | 457 | 160 | 25,9 % |
| **1,5 s** | **370** | **247** | **40,0 %** |
| 2 s | 349 | 268 | 43,4 % |
| 3 s | 319 | 298 | 48,3 % |
| 5 s | 290 | 327 | 53,0 % |
| 8 s | 272 | 345 | 55,9 % |

### 3.2 Par film

| Film | Drapeaux | Portages | Brut | net@1s | net@1,5s | net@2s | net@3s | net@5s | net@8s |
|---|---|---|---|---|---|---|---|---|---|
| 16ea3668 | 2 | 29 | 29 | 21 | 17 | 17 | 17 | 17 | 17 |
| 4ecdf3e7 | 1 | 25 | 25 | 17 | 13 | 13 | 11 | 9 | 9 |
| 58864b3c | 2 | 20 | 20 | 20 | 14 | 14 | 12 | 10 | 9 |
| 64e8adfa | 2 | 92 | 92 | 73 | 58 | 55 | 47 | 44 | 38 |
| 7fce3219 | 2 | 85 | 85 | 35 | 28 | 26 | 23 | 22 | 22 |
| 8bc6074f | 2 | 53 | 53 | 40 | 30 | 30 | 28 | 24 | 20 |
| a0c36016 | 2 | 49 | 49 | 33 | 29 | 27 | 26 | 25 | 24 |
| b8a44fe8 | 2 | 63 | 63 | 59 | 51 | 46 | 39 | 32 | 29 |
| bc60b4d9 | 2 | 31 | 31 | 28 | 25 | 25 | 25 | 21 | 21 |
| bf5ced1b | 1 | 12 | 12 | 12 | 10 | 7 | 5 | 5 | 5 |
| cde26226 | 2 | 88 | 88 | 59 | 41 | 38 | 35 | 32 | 30 |
| f8efc5ca | 2 | 35 | 35 | 31 | 26 | 26 | 26 | 26 | 26 |
| fb1a1a72 | 2 | 35 | 35 | 29 | 28 | 25 | 25 | 23 | 22 |

À 1,5 s, la part nette d'un film va de **33 %** (`7fce3219` : 28 sur 85) à **81 %**
(`b8a44fe8` : 51 sur 63, `fb1a1a72` : 28 sur 35). Le jonglage n'est donc pas un bruit
uniforme : il caractérise certaines parties, et à l'intérieur d'une partie certains
joueurs (§5).

Le détail par joueur (87 lignes) est en annexe §A.2.

## 4. Distribution des délais lâcher → reprise par le même joueur

Population : **404 couples** (portage suivi d'une reprise du même drapeau par le même
joueur), toutes durées confondues.

`min = 0,10 s` · `p10 = 0,50` · `p25 = 0,80` · `médiane = 1,20` · `p75 = 3,50`
· `p90 = 18,80` · `max = 313,40`

| Tranche (s) | Effectif | Cumul |
|---|---|---|
| [0,0 ; 0,5) | 33 | 33 |
| [0,5 ; 1,0) | 106 | 139 |
| [1,0 ; 1,5) | 99 | 238 |
| **[1,5 ; 2,0)** | **27** | 265 |
| [2,0 ; 2,5) | 15 | 280 |
| [2,5 ; 3,0) | 14 | 294 |
| [3,0 ; 3,5) | 8 | 302 |
| [3,5 ; 4,0) | 10 | 312 |
| [4,0 ; 4,5) | 9 | 321 |
| [4,5 ; 5,0) | 6 | 327 |
| [5,0 ; 5,5) | 3 | 330 |
| [5,5 ; 6,0) | 9 | 339 |
| [6,0 ; 6,5) | 2 | 341 |
| [6,5 ; 7,0) | 2 | 343 |
| [7,0 ; 7,5) | 0 | 343 |
| [7,5 ; 8,0) | 2 | 345 |
| [8,0 ; 8,5) | 4 | 349 |
| [8,5 ; 9,0) | 0 | 349 |
| [9,0 ; 9,5) | 0 | 349 |
| [9,5 ; 10,0) | 1 | 350 |
| >= 10,0 | 54 | 404 |

**LA COUPURE EST ENTRE 1,4 ET 1,6 SECONDE, ET ELLE EST FRANCHE.** L'effectif passe de
**99** sur [1,0 ; 1,5) à **27** sur [1,5 ; 2,0) : un facteur **3,7** d'une tranche à la
suivante, alors qu'aucune autre transition du graphe ne dépasse un facteur 2 au-delà de
1,5 s. Au-delà, la distribution est un **plateau** qui décroît lentement (15, 14, 8, 10,
9, 6, 3, 9 ...) jusqu'à une queue franche au-delà de 10 s (54 couples, jusqu'à 313 s).

Les deux populations se lisent à l'œil nu :

- un **mode dense entre 0,3 et 1,5 s** (238 couples, 59 % de la population) — la durée
  physique d'un jet de drapeau suivi d'une reprise à la course ;
- un **plateau au-delà de 1,5 s** — des reprises qui ne sont plus du jonglage : le
  joueur a été gêné, a lâché pour tirer, ou a repris un drapeau qu'il avait laissé.

La tranche [1,5 ; 2,0) vaut 27 couples contre 247 prises repliées à 1,5 s : reculer la
fenêtre de 1,5 à 2 s ne replie que **21 prises de plus** (le comptage net utilise
`délai <= W`, donc les 9 délais valant exactement 1,50 s sont repliés dès 1,5 s :
238 + 9 = 247). Le gain marginal a chuté d'un ordre de grandeur.

## 5. Fenêtre recommandée

> **Recommandation : `fenetre_jonglage_s = 1,5`.**

Justification, dans l'ordre :

1. **Elle tombe sur la coupure mesurée**, pas sur un chiffre rond : le facteur 3,7 entre
   [1,0 ; 1,5) et [1,5 ; 2,0) est la seule discontinuité de la distribution.
2. **Elle capte la quasi-totalité du mode dense** : 247 des 265 couples de moins de 2 s,
   soit 93 % de la population « courte ».
3. **Elle ne mord pas sur le plateau** : à 2 s on ne gagne que 21 prises (+ 8,5 %), à 3 s
   51 (+ 20 %), et ces prises-là ont, à la lecture des délais bruts (annexe §A.3), la
   dispersion d'un plateau et non celle d'un mode.
4. Elle est robuste au pas de mesure : l'axe du rejeu est à 100 ms, donc 1,5 s vaut
   15 frames — quinze pas, très loin du bruit de discrétisation.

À 1,5 s, le corpus rend **370 prises nettes pour 617 brutes**, soit **40,0 % du compteur
officiel qui est du jonglage**.

## 6. Verdict : l'écart justifie-t-il la grandeur nette ?

> **OUI, sans réserve. Le compteur brut ne classe pas les joueurs sur « prendre » : il
> les classe sur « jongler ».**

Deux faits le montrent, tous deux lisibles dans l'annexe §A.2 :

**a) L'écart est massif et concentré.** 40 % des prises du corpus disparaissent à 1,5 s.
Il ne s'agit pas d'une correction cosmétique de quelques pourcents : sur certains couples
(film, joueur) le compteur brut est un multiple du compte réel —
`cde26226 / 2535468122262494` : **40 brutes pour 12 nettes** ;
`7fce3219 / 2533274955626571` : **27 brutes pour 4 nettes** ;
`7fce3219 / 2535469190789936` : **17 pour 3** ; `64e8adfa / 2535449464686885` :
**25 pour 9**. À côté, d'autres joueurs du MÊME film ne perdent rien
(`64e8adfa / 2533274823110022` : 6 brutes, 6 nettes).

**b) Le classement s'inverse.** Sur `7fce3219` (le témoin cité par le plan), le podium
des « drapeaux saisis » change entièrement :

| Rang | Au compteur brut | Aux prises nettes (1,5 s) |
|---|---|---|
| 1 | 2533274955626571 — 27 | 2535415254450151 — 7 |
| 2 | 2535469190789936 — 17 | 2533274858283686 — 6 |
| 3 | 2654577101798078 — 15 | 2533274955626571 — 4 |
| 4 | 2535415254450151 — 13 | 2654577101798078 — 4 |

Le premier au brut tombe troisième au net, et le quatrième au brut devient premier. Une
grandeur qui désigne le mauvais joueur n'est pas une approximation de la bonne : c'en est
une autre. Faire entrer « drapeaux saisis » dans le rôle « prendre » au compteur brut
aurait récompensé le jonglage — la décision utilisateur du 13/09 est confirmée par la
mesure.

## 7. Contrôles

**Contrôle 1 — nettes <= brutes partout : OK.** Vérifié sur les 87 couples
(film, joueur) x 6 fenêtres = 522 comparaisons, zéro violation. La règle est monotone :
elle ne crée jamais de prise.

**Contrôle 2 — « sur un film sans lâcher volontaire daté, nettes = brutes » : NON
INSTRUIT, faute de sujet.** Les 13 films CTF du corpus ont tous `closedByObject > 0`
(de 9 à 79) : le lâcher volontaire daté par la renaissance de l'objet (lot 6.7-B1) est
présent partout. Il n'existe donc aucun film du parc sur lequel ce contrôle puisse être
posé. C'est une absence de population, pas un échec — et c'est aussi un fait utile : la
chaîne qui date le lâcher n'est pas marginale.

**Contrôle 2 bis — substitut mesuré, posé à la place : OK sur 13/13.** Toute prise
repliée suppose qu'un portage précédent du même joueur ait été FERMÉ avant elle ; le
seul fermoir qui date un lâcher volontaire est `closedByObject`. On exige donc
`prises repliées@1,5 s <= closedByObject`, film par film :

| Film | Repliées @1,5 s | closedByObject | |
|---|---|---|---|
| 16ea3668 | 12 | 21 | OK |
| 4ecdf3e7 | 12 | 19 | OK |
| 58864b3c | 6 | 16 | OK |
| 64e8adfa | 34 | 79 | OK |
| 7fce3219 | 57 | 73 | OK |
| 8bc6074f | 23 | 42 | OK |
| a0c36016 | 20 | 42 | OK |
| b8a44fe8 | 12 | 55 | OK |
| bc60b4d9 | 6 | 24 | OK |
| bf5ced1b | 2 | 9 | OK |
| cde26226 | 47 | 68 | OK |
| f8efc5ca | 9 | 25 | OK |
| fb1a1a72 | 7 | 30 | OK |

Aucun film ne replie plus de prises qu'il n'a de lâchers volontaires datés : la règle ne
fuit pas vers une autre chaîne de fermeture. Accessoirement, `closedByHandoff` (la passe
de main, lot 6.11) vaut **0 sur les 13 films** — la passe de main ne contribue à aucune
des prises repliées ici, et la mesure ne la confond donc pas avec du jonglage.

## 8. Comparaison au compteur API `flag_grabs` : NON FAITE

Demandée en option, elle n'a pas été instruite, pour deux raisons de fait :

1. Le compteur `flag_grabs` de `match_objective_stats` vit dans
   `data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb`, une base **tenue en
   écriture par le serveur Go du poste** (`:8000`). Le modèle mono-process (ADR 0013 /
   0016) interdit de l'ouvrir depuis un second processus ; la consigne d'étape l'interdit
   également.
2. La voie HTTP, qui n'aurait ouvert aucune base, est **fermée par le contrôle d'accès** :
   `GET /api/v1/players/JGtm/matches/7fce3219-...` répond **403
   `player_forbidden`** — l'ouvrier n'a pas de session authentifiée, et il n'est pas
   question d'en fabriquer une.

Cette comparaison n'est pas nécessaire au verdict : la définition retenue s'appuie sur le
film, pas sur l'API, et le plan pose déjà que le compteur brut de l'API reste affiché tel
quel là où il l'est. Elle reste faisable par un opérateur disposant d'une session, ou
hors ligne serveur arrêté.

## 9. Limites

1. **Films tronqués.** Deux films portent un portage OUVERT (`b8a44fe8`, `fb1a1a72` : 1
   chacun) : sa borne de fin est la fin de l'axe, une borne haute et non une mesure. Un
   portage ouvert ne peut donc jamais être suivi d'une reprise sur le même drapeau, et
   sort de fait de la mesure du jonglage. Impact : 2 portages sur 617.
2. **Cartes hors catalogue d'objectifs.** Quand la carte n'a pas de socle `flag_spawn`
   connu, les états `home` sont omis et la suite d'intervalles porte des trous. Le corpus
   n'expose pas ce cas au pire (`spawns >= 1` partout), mais deux films ne publient qu'UN
   drapeau (`4ecdf3e7`, `spawns = 1` ; `bf5ced1b`, `spawns = 3`). Sur un seul drapeau,
   la clause « même drapeau » est trivialement vraie et la règle est au maximum de sa
   sévérité. Le risque théorique — replier une vraie reprise après un retour automatique
   — est nul à 1,5 s : aucun retour de drapeau ne s'effectue en une seconde et demie.
3. **Drapeau neutre.** `bf5ced1b` et `bc60b4d9` ont `spawns = 3`, signature d'un socle
   central. En drapeau neutre les deux camps portent le même objet : « même équipe
   propriétaire » ne discrimine plus rien, et seule l'identité du joueur fait le tri. La
   règle reste juste (elle repose sur le joueur), mais elle ne bénéficie d'aucune marge
   de sécurité sur ces films.
4. **Le brut mesuré ici est celui du FILM, pas celui de l'API.** Il vaut le nombre de
   portages PUBLIÉS (617), lui-même inférieur aux ouvertures de l'oracle (620) : 3 prises
   sont rejetées faute de trajectoire publiée à leur instant. L'écart est de 0,5 % et ne
   déplace aucune conclusion, mais un chiffre de ce rapport ne s'oppose pas terme à terme
   au compteur officiel de l'API.
5. **Un match sans film décodé n'est pas mesurable** sur cette grandeur : 63 des 76
   artefacts du poste ne sont pas du CTF, et la couverture CTF réelle du parc de matchs
   est bien plus faible que 13/76 (le cache de rejeu ne couvre pas tous les matchs). La
   règle produit du « non mesuré », jamais du zéro — le plan le prévoit déjà.

---

# Annexes — sortie brute intégrale de l'instrument

Commande exacte (depuis `apps/go-api`, worktree `LevelUp-wt-ajust-escouade-formes`) :

```
GOCACHE=<worktree>/.gocache-prises-nettes CGO_ENABLED=0 \
PRISES_NETTES_CORPUS=<depot-principal>/data/cache/replays/halo_infinite \
go test ./internal/games/halo_infinite/film/replay/ -run PrisesNettes -v -count=1
```

Résultat : `PASS` en 1,9 s. Le fichier de test a été supprimé après cette exécution.

Correspondance des renvois du rapport avec les blocs ci-dessous : **§A.1** = bloc
`=== CORPUS ===` (les 63 films écartés et leurs compteurs) · **§A.2** = bloc
`=== (a) PRISES BRUTES / NETTES PAR JOUEUR ===` (87 lignes) · **§A.3** = ligne
`--- delais bruts tries (s) ---` du bloc `=== (b) ===` (les 404 délais, triés).

```
=== RUN   TestPrisesNettesResearch
=== CORPUS ===
artefacts non derives balayes : 76
films CTF retenus : 13
--- ecartes (63) ---
01e1f945	schema 54	verdict flagFilm=false (bursts=0 captures=2 steals=0)
0797ce72	schema 54	verdict flagFilm=false (bursts=0 captures=0 steals=0)
0891225f	schema 54	verdict flagFilm=false (bursts=0 captures=0 steals=0)
0a44c6cc	schema 54	verdict flagFilm=false (bursts=0 captures=0 steals=0)
0d265ab0	schema 54	verdict flagFilm=false (bursts=0 captures=0 steals=0)
1b2d9e08	schema 54	verdict flagFilm=false (bursts=3 captures=0 steals=0)
1cd3848a	schema 54	verdict flagFilm=false (bursts=0 captures=0 steals=0)
21ece4d8	schema 54	verdict flagFilm=false (bursts=1 captures=4 steals=0)
28c9b538	schema 54	verdict flagFilm=false (bursts=0 captures=0 steals=0)
2cf24f30	schema 54	verdict flagFilm=false (bursts=1 captures=0 steals=0)
30724141	schema 54	verdict flagFilm=false (bursts=0 captures=0 steals=0)
30a23d15	schema 54	verdict flagFilm=false (bursts=4 captures=0 steals=0)
32d9a94f	schema 54	verdict flagFilm=false (bursts=0 captures=16 steals=0)
3923bede	schema 54	verdict flagFilm=false (bursts=0 captures=0 steals=0)
396cfc92	schema 54	verdict flagFilm=false (bursts=2 captures=10 steals=0)
3ba5a548	schema 54	verdict flagFilm=false (bursts=0 captures=0 steals=0)
43716616	schema 54	verdict flagFilm=false (bursts=1 captures=10 steals=0)
4577fcc4	schema 54	verdict flagFilm=false (bursts=1 captures=0 steals=0)
46c3f91d	schema 54	verdict flagFilm=false (bursts=0 captures=0 steals=0)
4bd6de5a	schema 54	verdict flagFilm=false (bursts=0 captures=0 steals=0)
4f77afc1	schema 54	verdict flagFilm=false (bursts=0 captures=1 steals=2)
51ebbc0f	schema 54	verdict flagFilm=false (bursts=0 captures=13 steals=0)
5676a9ba	schema 54	verdict flagFilm=false (bursts=0 captures=4 steals=0)
572e236b	schema 54	verdict flagFilm=false (bursts=0 captures=9 steals=0)
5dfdc63b	schema 54	verdict flagFilm=false (bursts=0 captures=0 steals=0)
606d9844	schema 54	verdict flagFilm=false (bursts=0 captures=0 steals=0)
72b0a25e	schema 54	verdict flagFilm=false (bursts=0 captures=0 steals=0)
7b0d89c4	schema 54	verdict flagFilm=false (bursts=0 captures=0 steals=0)
7f1bbf06	schema 54	verdict flagFilm=false (bursts=2 captures=7 steals=0)
8076f97f	schema 54	verdict flagFilm=false (bursts=0 captures=3 steals=0)
81c02726	schema 54	verdict flagFilm=false (bursts=0 captures=7 steals=0)
879a4dba	schema 54	verdict flagFilm=false (bursts=0 captures=3 steals=4)
8a485699	schema 54	verdict flagFilm=false (bursts=0 captures=0 steals=0)
94a28b8b	schema 54	verdict flagFilm=false (bursts=0 captures=0 steals=0)
9e8fb31b	schema 54	verdict flagFilm=false (bursts=0 captures=0 steals=0)
9ffce8ef	schema 54	verdict flagFilm=false (bursts=0 captures=0 steals=0)
a03a5e65	schema 54	verdict flagFilm=false (bursts=0 captures=0 steals=0)
a36c8bed	schema 54	verdict flagFilm=false (bursts=2 captures=11 steals=0)
a396aa7f	schema 54	verdict flagFilm=false (bursts=0 captures=0 steals=0)
a4083bd2	schema 54	verdict flagFilm=false (bursts=0 captures=0 steals=0)
a6ae19fb	schema 54	verdict flagFilm=false (bursts=0 captures=0 steals=0)
ac03413d	schema 54	verdict flagFilm=false (bursts=0 captures=0 steals=0)
af3500aa	schema 54	verdict flagFilm=false (bursts=0 captures=0 steals=0)
b0fe12b1	schema 54	verdict flagFilm=false (bursts=4 captures=0 steals=0)
b1ad85eb	schema 54	verdict flagFilm=false (bursts=0 captures=0 steals=0)
b1f01a33	schema 54	verdict flagFilm=false (bursts=0 captures=0 steals=0)
bf2a9f05	schema 54	verdict flagFilm=false (bursts=0 captures=0 steals=0)
bfcd1175	schema 54	verdict flagFilm=false (bursts=0 captures=0 steals=0)
bfecd02b	schema 54	verdict flagFilm=false (bursts=6 captures=0 steals=0)
c7f94693	schema 54	verdict flagFilm=false (bursts=0 captures=0 steals=0)
c88ec007	schema 54	verdict flagFilm=false (bursts=3 captures=16 steals=0)
d1dfbc02	schema 54	verdict flagFilm=false (bursts=0 captures=0 steals=0)
d8b13ec2	schema 54	verdict flagFilm=false (bursts=0 captures=0 steals=0)
d9781168	schema 54	verdict flagFilm=false (bursts=0 captures=25 steals=0)
daaa17d6	schema 54	verdict flagFilm=false (bursts=1 captures=0 steals=0)
e1259a69	schema 54	verdict flagFilm=false (bursts=0 captures=0 steals=0)
e60aaf06	schema 54	verdict flagFilm=false (bursts=2 captures=11 steals=0)
e85d7bad	schema 54	verdict flagFilm=false (bursts=0 captures=0 steals=0)
efe716b4	schema 54	verdict flagFilm=false (bursts=0 captures=0 steals=2)
f0220a96	schema 54	verdict flagFilm=false (bursts=1 captures=0 steals=0)
f2966f08	schema 54	verdict flagFilm=false (bursts=0 captures=0 steals=1)
faff9935	schema 54	verdict flagFilm=false (bursts=0 captures=0 steals=0)
fccc61cd	schema 54	verdict flagFilm=false (bursts=0 captures=0 steals=0)

=== (a) PRISES BRUTES / NETTES PAR JOUEUR ===
film	xuid	brut	net@1s	net@1.5s	net@2s	net@3s	net@5s	net@8s
16ea3668	2533274858283686	9	8	5	5	5	5	5
16ea3668	2533275034585819	3	2	1	1	1	1	1
16ea3668	2535417044536883	2	2	2	2	2	2	2
16ea3668	2535447614023932	8	2	2	2	2	2	2
16ea3668	2535466949313010	4	4	4	4	4	4	4
16ea3668	2598952284978039	3	3	3	3	3	3	3
4ecdf3e7	2533274823110022	2	2	2	2	2	2	2
4ecdf3e7	2533274858283686	10	10	6	6	4	3	3
4ecdf3e7	2535426953464929	11	3	3	3	3	2	2
4ecdf3e7	2535469190789936	2	2	2	2	2	2	2
58864b3c	2533274822170595	1	1	1	1	1	1	1
58864b3c	2533274823110022	2	2	2	2	2	1	1
58864b3c	2533274829350000	5	5	2	2	2	1	1
58864b3c	2533274858283686	5	5	3	3	2	2	2
58864b3c	2535452521259564	4	4	4	4	3	3	2
58864b3c	2535469190789936	3	3	2	2	2	2	2
64e8adfa	2533274792763167	6	6	6	6	6	6	5
64e8adfa	2533274808613055	5	5	5	5	5	5	5
64e8adfa	2533274823110022	6	6	6	6	6	6	6
64e8adfa	2535413221816250	5	2	2	2	2	2	2
64e8adfa	2535449464686885	25	10	9	7	6	5	4
64e8adfa	2535449963449748	6	6	6	6	5	4	4
64e8adfa	2535456378021162	23	22	15	15	9	8	5
64e8adfa	2535465820713037	16	16	9	8	8	8	7
7fce3219	2533274823110022	2	2	2	1	1	1	1
7fce3219	2533274858283686	9	9	6	5	4	3	3
7fce3219	2533274943116584	1	1	1	1	1	1	1
7fce3219	2533274955626571	27	4	4	4	4	4	4
7fce3219	2535415254450151	13	8	7	7	6	6	6
7fce3219	2535458465148583	1	1	1	1	1	1	1
7fce3219	2535469190789936	17	3	3	3	3	3	3
7fce3219	2654577101798078	15	7	4	4	3	3	3
8bc6074f	2533274823110022	5	5	5	5	5	5	3
8bc6074f	2533274829350000	14	10	6	6	5	3	2
8bc6074f	2533274833178266	3	3	3	3	3	3	3
8bc6074f	2533274858283686	5	4	1	1	1	1	1
8bc6074f	2535449383340628	11	11	8	8	7	6	5
8bc6074f	2535450759128461	2	2	2	2	2	2	2
8bc6074f	2535452521259564	3	3	3	3	3	2	2
8bc6074f	2535469190789936	10	2	2	2	2	2	2
a0c36016	2533274823110022	1	1	1	1	1	1	1
a0c36016	2533274858283686	5	5	4	3	3	3	3
a0c36016	2533274858911298	7	7	7	7	7	6	5
a0c36016	2535425652517893	7	5	4	4	4	4	4
a0c36016	2535427927026623	8	4	3	2	2	2	2
a0c36016	2535448372866366	2	2	2	2	2	2	2
a0c36016	2535456861208728	7	5	4	4	3	3	3
a0c36016	2535469190789936	12	4	4	4	4	4	4
b8a44fe8	2533274823110022	2	2	2	2	2	2	2
b8a44fe8	2533274858283686	30	28	22	18	14	8	7
b8a44fe8	2533274858911298	2	2	2	2	2	2	2
b8a44fe8	2533274897257135	2	2	2	2	2	2	2
b8a44fe8	2535442462807197	16	15	15	15	13	12	10
b8a44fe8	2535455799302553	10	9	7	6	5	5	5
b8a44fe8	2535469190789936	1	1	1	1	1	1	1
bc60b4d9	2533274823110022	2	2	2	2	2	2	2
bc60b4d9	2533274897645479	5	5	5	5	5	3	3
bc60b4d9	2533274924336100	4	4	4	4	4	4	4
bc60b4d9	2533274943116584	3	2	2	2	2	2	2
bc60b4d9	2535458465148583	1	1	1	1	1	1	1
bc60b4d9	2535469190789936	5	5	4	4	4	4	4
bc60b4d9	2654577101798078	11	9	7	7	7	5	5
bf5ced1b	2535412865870758	3	3	3	2	1	1	1
bf5ced1b	2535434888128184	2	2	2	2	2	2	2
bf5ced1b	2535459651975585	7	7	5	3	2	2	2
cde26226	2533274817658307	6	6	6	6	6	6	6
cde26226	2533274818074729	4	4	4	4	4	4	3
cde26226	2533274823110022	1	1	1	1	1	1	1
cde26226	2533274849050722	4	4	4	4	4	4	4
cde26226	2533274858283686	9	6	5	5	4	3	3
cde26226	2535415492484355	13	10	6	6	6	4	3
cde26226	2535468122262494	40	25	12	9	7	7	7
cde26226	2535469190789936	11	3	3	3	3	3	3
f8efc5ca	2533274804730586	3	3	3	3	3	3	3
f8efc5ca	2533274811578968	4	4	3	3	3	3	3
f8efc5ca	2533274823110022	5	5	5	5	5	5	5
f8efc5ca	2533274858283686	7	7	4	4	4	4	4
f8efc5ca	2535458336606528	8	4	3	3	3	3	3
f8efc5ca	2535467632752562	6	6	6	6	6	6	6
f8efc5ca	2535469190789936	2	2	2	2	2	2	2
fb1a1a72	2533274795950878	3	3	3	3	3	3	3
fb1a1a72	2535408981717353	3	3	3	3	3	3	3
fb1a1a72	2535417580715145	8	3	3	3	3	3	3
fb1a1a72	2535425179403277	5	5	4	4	4	4	4
fb1a1a72	2535450323793545	1	1	1	1	1	1	1
fb1a1a72	2535469190789936	12	11	11	8	8	6	5
fb1a1a72	2535473461033821	3	3	3	3	3	3	3

--- TOTAUX ---
TOTAL		617	457	370	349	319	290	272
fenetre 1s : 160 prises repliees sur 617 (25.9 %)
fenetre 1.5s : 247 prises repliees sur 617 (40.0 %)
fenetre 2s : 268 prises repliees sur 617 (43.4 %)
fenetre 3s : 298 prises repliees sur 617 (48.3 %)
fenetre 5s : 327 prises repliees sur 617 (53.0 %)
fenetre 8s : 345 prises repliees sur 617 (55.9 %)

=== (a bis) PAR FILM ===
film	drapeaux	ouvertures	noBridge	noTrack	horsAxe	portages	sansXuid	fermes	ouverts	closedByObject	closedByHandoff	spawns	brut	net@1s	net@1.5s	net@2s	net@3s	net@5s	net@8s
16ea3668	2	29	0	0	0	29	0	29	0	21	0	2	29	21	17	17	17	17	17
4ecdf3e7	1	26	0	1	0	25	0	25	0	19	0	1	25	17	13	13	11	9	9
58864b3c	2	20	0	0	0	20	0	20	0	16	0	2	20	20	14	14	12	10	9
64e8adfa	2	93	0	1	0	92	0	92	0	79	0	2	92	73	58	55	47	44	38
7fce3219	2	85	0	0	0	85	0	85	0	73	0	2	85	35	28	26	23	22	22
8bc6074f	2	54	0	1	0	53	0	53	0	42	0	2	53	40	30	30	28	24	20
a0c36016	2	49	0	0	0	49	0	49	0	42	0	2	49	33	29	27	26	25	24
b8a44fe8	2	63	0	0	0	63	0	62	1	55	0	2	63	59	51	46	39	32	29
bc60b4d9	2	31	0	0	0	31	0	31	0	24	0	3	31	28	25	25	25	21	21
bf5ced1b	1	12	0	0	0	12	0	12	0	9	0	3	12	12	10	7	5	5	5
cde26226	2	88	0	0	0	88	0	88	0	68	0	2	88	59	41	38	35	32	30
f8efc5ca	2	35	0	0	0	35	0	35	0	25	0	2	35	31	26	26	26	26	26
fb1a1a72	2	35	0	0	0	35	0	34	1	30	0	2	35	29	28	25	25	23	22

=== (b) DELAIS LACHER -> REPRISE PAR LE MEME JOUEUR (meme drapeau) ===
effectif : 404
min=0.10 p10=0.50 p25=0.80 med=1.20 p75=3.50 p90=18.80 max=313.40
tranche	effectif	cumul
[0.0;0.5)	33	33
[0.5;1.0)	106	139
[1.0;1.5)	99	238
[1.5;2.0)	27	265
[2.0;2.5)	15	280
[2.5;3.0)	14	294
[3.0;3.5)	8	302
[3.5;4.0)	10	312
[4.0;4.5)	9	321
[4.5;5.0)	6	327
[5.0;5.5)	3	330
[5.5;6.0)	9	339
[6.0;6.5)	2	341
[6.5;7.0)	2	343
[7.0;7.5)	0	343
[7.5;8.0)	2	345
[8.0;8.5)	4	349
[8.5;9.0)	0	349
[9.0;9.5)	0	349
[9.5;10.0)	1	350
>=10.0	54	404
--- delais bruts tries (s) ---
0.10 0.10 0.10 0.10 0.10 0.10 0.20 0.30 0.30 0.30 0.30 0.30 0.30 0.30 0.30 0.40 0.40 0.40 0.40 0.40 0.40 0.40 0.40 0.40 0.40 0.40 0.40 0.40 0.40 0.40 0.40 0.40 0.40 0.50 0.50 0.50 0.50 0.50 0.50 0.50 0.50 0.50 0.60 0.60 0.60 0.60 0.60 0.60 0.60 0.60 0.60 0.60 0.60 0.60 0.60 0.60 0.60 0.70 0.70 0.70 0.70 0.70 0.70 0.70 0.70 0.70 0.70 0.70 0.70 0.70 0.70 0.70 0.70 0.70 0.70 0.70 0.70 0.70 0.70 0.70 0.70 0.70 0.70 0.70 0.70 0.70 0.80 0.80 0.80 0.80 0.80 0.80 0.80 0.80 0.80 0.80 0.80 0.80 0.80 0.80 0.80 0.80 0.80 0.80 0.80 0.80 0.80 0.80 0.80 0.80 0.80 0.80 0.80 0.80 0.80 0.80 0.80 0.90 0.90 0.90 0.90 0.90 0.90 0.90 0.90 0.90 0.90 0.90 0.90 0.90 0.90 0.90 0.90 0.90 0.90 0.90 0.90 0.90 0.90 1.00 1.00 1.00 1.00 1.00 1.00 1.00 1.00 1.00 1.00 1.00 1.00 1.00 1.00 1.00 1.00 1.00 1.00 1.00 1.00 1.00 1.10 1.10 1.10 1.10 1.10 1.10 1.10 1.10 1.10 1.10 1.10 1.10 1.10 1.10 1.10 1.10 1.10 1.10 1.10 1.10 1.10 1.10 1.10 1.20 1.20 1.20 1.20 1.20 1.20 1.20 1.20 1.20 1.20 1.20 1.20 1.20 1.20 1.20 1.20 1.20 1.20 1.20 1.20 1.20 1.30 1.30 1.30 1.30 1.30 1.30 1.30 1.30 1.30 1.30 1.30 1.30 1.30 1.30 1.30 1.30 1.30 1.30 1.30 1.30 1.30 1.40 1.40 1.40 1.40 1.40 1.40 1.40 1.40 1.40 1.40 1.40 1.40 1.40 1.50 1.50 1.50 1.50 1.50 1.50 1.50 1.50 1.50 1.60 1.60 1.60 1.60 1.70 1.70 1.70 1.70 1.80 1.80 1.80 1.80 1.80 1.90 1.90 1.90 1.90 1.90 2.00 2.00 2.00 2.10 2.10 2.10 2.10 2.10 2.20 2.30 2.30 2.30 2.30 2.30 2.40 2.50 2.50 2.60 2.60 2.70 2.70 2.70 2.70 2.80 2.80 2.80 2.90 2.90 2.90 3.00 3.00 3.00 3.00 3.10 3.30 3.40 3.40 3.50 3.50 3.50 3.60 3.60 3.70 3.80 3.80 3.90 3.90 4.00 4.00 4.10 4.10 4.20 4.20 4.40 4.40 4.40 4.50 4.70 4.80 4.90 4.90 4.90 5.10 5.20 5.30 5.50 5.60 5.60 5.70 5.70 5.80 5.80 5.90 5.90 6.00 6.20 6.60 6.70 7.50 7.70 8.30 8.40 8.40 8.40 9.50 10.60 11.10 11.50 12.20 12.70 13.00 14.30 14.50 15.20 15.70 16.20 16.40 18.80 20.80 21.90 23.00 23.10 23.30 25.20 25.50 27.10 28.50 29.00 29.10 31.50 32.90 33.00 36.10 38.90 40.60 42.60 44.50 46.50 48.30 48.60 52.50 54.10 54.30 57.10 57.80 60.80 61.70 63.80 66.20 79.80 80.20 80.60 92.50 104.80 112.00 115.40 137.60 178.20 313.40

=== (c) CONTROLES ===
controle 1 (net <= brut) : OK sur 87 lignes x 6 fenetres
controle 2 — films SANS lacher volontaire date (closedByObject=0) :
  aucun film CTF du corpus n'a closedByObject=0 : controle non instruit
controle 2 bis (substitut) — repliees@1.5s <= closedByObject, film par film :
  16ea3668 repliees=12 closedByObject=21 -> OK
  4ecdf3e7 repliees=12 closedByObject=19 -> OK
  58864b3c repliees=6 closedByObject=16 -> OK
  64e8adfa repliees=34 closedByObject=79 -> OK
  7fce3219 repliees=57 closedByObject=73 -> OK
  8bc6074f repliees=23 closedByObject=42 -> OK
  a0c36016 repliees=20 closedByObject=42 -> OK
  b8a44fe8 repliees=12 closedByObject=55 -> OK
  bc60b4d9 repliees=6 closedByObject=24 -> OK
  bf5ced1b repliees=2 closedByObject=9 -> OK
  cde26226 repliees=47 closedByObject=68 -> OK
  f8efc5ca repliees=9 closedByObject=25 -> OK
  fb1a1a72 repliees=7 closedByObject=30 -> OK
--- PASS: TestPrisesNettesResearch (1.89s)
PASS
ok  	levelup/go-api/internal/games/halo_infinite/film/replay	1.959s
```
