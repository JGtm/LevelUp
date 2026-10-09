# Registre — effondrement de `assist_known` dans `match_kill_events` (Halo Infinite)

> Date de mesure : **2026-08-29**. Base : `data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb`.
> Outils : `cmd/diag_q` (SQL lecture seule), `cmd/diag_film_avail` (sonde jetable).
> Origine : §9 de `.ai/PLAN_SCORE_PAR_MANCHES.md` — question utilisateur « pourquoi je ne
> vois pas le tableau des assistances ? ».
>
> **VERSION 2, du 2026-08-29 — la version 1 a ete REFUTEE sur 12 affirmations par une revue
> adversariale (5 relecteurs en contexte frais).** Ce qui a change est liste en §0. Les
> chiffres ci-dessous ont tous ete re-mesures. Les chiffres NON mesures ici sont attribues a
> leur source.

---

## 0. CE QUE LA REVUE A REFUTE DANS LA VERSION 1

| affirmation v1 | verdict | valeur re-mesuree |
|---|---|---|
| « 416 sans film sur toute la base, 1 365 matchs au total » | **FAUX** | la commande selectionne **999** candidats sur un registre de **1 948** matchs. 1 365 et 416 etaient les chiffres de `match_kill_events_latest`, pas du registre. |
| « le rattrapage porte sur 374 matchs » | **FAUX** | 374 est le trou DEPUIS AVRIL ; la commande en traite **999**, dont 529 en 2023. |
| « une seule passe avec les tokens de JGtm rattrape les quatre joueurs / c'est heureux, seul token vivant » | **HORS SUJET** | le film s'obtient par identifiant de MATCH, sans xuid. N'importe quel token vivant couvre tout (§5bis). |
| « `xuid_aliases` a cesse d'etre alimente en avril 2026 » | **FAUX** | la table recoit 2 683 lignes apres avril. Seul l'alimentateur `source='highlight_events'` s'arrete (§7). |
| « Aucun code Go ne cree de nouveau manifeste de film » | **FAUX depuis le 2026-08-14** | `filmcache.Write` en cree (§3.4). |
| « 954 manifestes », « 14 manifestes du 07-18 au 07-31 » | **FAUX** | **951** manifestes `.json` (+ 3 `.json.raw`) ; **9** manifestes, du **07-25** au **07-31**. |
| 2026-04 film `pub` = 736 · 2026-02 film `pub` = 9 884 · matchs 2026-03 = 183 | **FAUX** | **751** · **9 875** · **184**. |
| « ~29 % du corpus deja perdu » | **NON ATTRIBUE** | recopie de `.ai/PLAN_MASTER_FILM_KILLFEED_REJEU.md:1677` (29,3 %) et de `replayartifacts/artifacts.go:19`. Non mesure ici (§4). |
| « la fenetre de retention couvre encore tout le trou » | **NON SOUTENUE** | l'echantillon ne tire que dans 2026, alors que 58 % du perimetre reel est anterieur (§4). |

**Le coeur du diagnostic (§1 a §3) a ete re-mesure a l'identique et tient.** Ce qui ne tenait
pas portait sur le PERIMETRE et sur ce qu'un lecteur ferait ensuite.

---

## 1. VERDICT

`assist_known` n'est produit QUE par le decodage du kill-feed du film. Le decodeur n'etait
atteignable que par une sous-commande manuelle hors ligne, qui ne lit que le cache disque. Ce
cache a ete alimente en continu jusqu'au **2026-04-07**, puis plus qu'un lot ponctuel de 9
manifestes fin juillet, et plus rien depuis le **2026-07-31**. Depuis avril, un match
synchronise n'a plus qu'une passe credit-seul, qui ecrit `assist_known = FALSE` par
construction — et correctement, le flux credit ne sait rien de l'assistant.

**Ce n'est pas une regression de l'attribution. C'est une regression de COUVERTURE.**

---

## 2. LE PRODUCTEUR, MESURE

Lignes de `match_kill_events_latest` par voie et par mois (`ak` = `assist_known`,
`pub` = `publishable`) :

| mois | voie | lignes | ak | pub |
|---|---|---|---|---|
| 2026-08 | credit (`highlight-events`) | 800 | **0** | 800 |
| 2026-07 | credit | 12 739 | **0** | 12 657 |
| 2026-07 | film (`marche`+`scan`) | 899 | **896** | 611 |
| 2026-06 | credit | 6 014 | **0** | 6 014 |
| 2026-05 | credit | 7 629 | **0** | 7 629 |
| 2026-05 | film | 92 | **92** | 0 |
| 2026-04 | credit | 6 355 | **0** | 6 341 |
| 2026-04 | film | 829 | **806** | 751 |
| 2026-03 | credit | 2 996 | **0** | 2 218 |
| 2026-03 | film | 12 922 | **12 730** | 9 171 |
| 2026-02 | credit | 2 073 | **0** | 1 175 |
| 2026-02 | film | 13 467 | **13 323** | 9 875 |

**`assist_known = TRUE` n'apparait JAMAIS sur la voie credit.** La hausse simultanee de
`publishable` a la MEME cause : le producteur credit publie tout, la passe film ne publie que
sous condition. Les deux courbes sont deux faces du basculement de producteur — co-consequences,
pas cause et effet.

### Couverture en matchs (comptee sur `match_registry`, pas sur les matchs a kill-events)

| mois | matchs au registre | avec passe film |
|---|---|---|
| 2026-08 | 8 | **0** |
| 2026-07 | 149 | 8 |
| 2026-06 | 65 | **0** |
| 2026-05 | 86 | 1 |
| 2026-04 | 86 | 11 |
| 2026-03 | 184 | **157** |
| 2026-02 | 173 | **160** |
| 2026-01 | 179 | 178 |
| 2025-12 | 153 | 153 |

---

## 3. LA CHAINE DE CAUSALITE

1. `assist_known` / `assist_gamertag` viennent du decodage du kill-feed
   (`internal/sync/killcollector/`), fusionne sur la base credit par
   `persist.MergeCreditAndFilm` (`kill_events_merge.go:301` : `credit.AssistKnown =
   film.AssistKnown` — la ligne de film fait foi, aucune derivation).
2. Avant ce lot, le decodeur n'etait atteignable que par `levelup backfill-killsource`
   (`cmd/levelup/cmd_backfill_killsource.go`). Verifie par grep a la date de la mesure :
   `killcollector.` n'avait aucun autre appelant de production.
3. Cette commande est **hors ligne par conception** : sa source est
   `killcollector.LocalCacheFilms`, sans aucun client HTTP. Elle decode ce qui est deja dans
   `data/cache/film_manifests/` + `film_chunks/`.
4. Le cache est herite du projet Python supprime a la migration
   (`internal/sync/haloclient/local_film_cache.go`, en-tete). **Jusqu'au 2026-08-14, aucun
   code Go n'en creait de nouveau manifeste** ; depuis cette date `filmcache.Write` en cree,
   mais seulement via l'etape 1.58 (artefacts de rejeu), bornee a la fenetre
   `replay_retention_months`, a 5 matchs par cycle, et **installee uniquement hors
   production**. Cela n'a pas suffi : 0 nouveau manifeste depuis le 2026-07-31.
5. Dates d'ecriture des **951 manifestes `.json`** (le repertoire contient 954 fichiers, dont
   3 `.json.raw` qui ne sont pas des manifestes) :

   | date | manifestes |
   |---|---|
   | 2026-03-14 | **850** |
   | 03-16 -> 03-27 | 81 |
   | **04-06 / 04-07** | **11** |
   | **07-25 / 07-26 / 07-31** | **9** |

   Les 9 manifestes de juillet expliquent exactement le sursaut : 8 d'entre eux sont les 8
   matchs de juillet a passe de film, le 9e est l'unique match de mai a passe de film.
6. Une seule revision de decodeur de FILM en base : `killsource-2026-07-31`, 949 matchs, du
   2023-04-15 au 2026-07-24 — exactement les films du cache. (Une seconde revision existe,
   `highlight-credit-2026-08-01`, 416 matchs : c'est celle du producteur credit-seul.)

**Le cache examine est bien le seul en vigueur.** `PathResolver.CacheRootDir()`
(`internal/domain/title/registry.go:704`) rend `<repoRoot>/data/cache` ; la cascade est
`--cache` -> `LEVELUP_LEGACY_FILM_CACHE_DIR` -> defaut ; la variable est vide dans
l'environnement courant et absente des fichiers de config ; une recherche disque
(`C:\Users\Guillaume` en profondeur 6 + autres volumes) ne trouve aucun autre
`film_manifests/` ou `film_chunks/`.

---

## 4. LE RATTRAPAGE — CE QUE LA MESURE PERMET DE DIRE, ET CE QU'ELLE NE PERMET PAS

Les films expirent cote 343. Le chiffre de **~29 % du corpus historique deja perdu N'EST PAS
MESURE ICI** : il est repris de `.ai/PLAN_MASTER_FILM_KILLFEED_REJEU.md:1677` (29,3 %) et de
`internal/sync/replayartifacts/artifacts.go:19` (~29 %), sans date ni methode publiees.

Sonde `cmd/diag_film_avail` (manifeste seul, aucun chunk telecharge) : **43 tirages, 43
disponibles, 0 expire**.

> **CE QUE CE 43/43 NE DIT PAS.** C'est un ECHANTILLON de 6 matchs par mois, et la sonde tire
> `ORDER BY start_time DESC` **uniquement dans 2026**. Elle ne dit donc rien de la
> disponibilite avant 2026 — alors que **584 des 999 candidats reels sont anterieurs a 2026**
> (529 rien qu'en 2023). Sur 43 tirages sans echec, la borne inferieure a 95 % est ~93 % :
> jusqu'a une trentaine d'expires resteraient compatibles avec ce resultat sur le seul
> perimetre 2026. La sonde a par ailleurs un biais connu : sa CTE part de
> `match_kill_events_latest`, donc un match sans aucun kill-event lui est invisible.

### Le perimetre reel de la commande

`matchsSansPasseDeFilm` selectionne `match_registry` moins les matchs deja a jour. Mesure :

| | valeur |
|---|---|
| `match_registry` | **1 948** matchs (2021-11-19 -> 2026-08-27) |
| candidats de la commande | **999** |
| dont sans passe de film DEPUIS le 2026-04-01 | **374** |

Ventilation des 999 candidats :

| annee | candidats |
|---|---|
| 2021 | 11 |
| 2022 | 2 |
| **2023** | **529** |
| 2024 | 39 |
| 2025 | 3 |
| **2026** | **415** |

> **CONSEQUENCE OPERATIONNELLE, ET ELLE INVALIDE LA RECETTE DE LA VERSION 1.** La passe trie
> du plus vieux au plus recent. Un `--limit 20` ne prend donc PAS 20 matchs du trou d'avril
> 2026 : il prend les 20 plus vieux du registre, c'est-a-dire des matchs de **2021**, jamais
> sondes, et que le « ~29 % perdu » designe comme les plus susceptibles d'etre expires. Un
> operateur verrait son lot d'essai revenir en 404 et conclurait a tort que le rattrapage est
> mort.

---

## 5. IMPACT PRODUIT — DEUX BLOCS QUI DISPARAISSENT SANS TRACE

| surface | filtre | comportement quand la mesure manque |
|---|---|---|
| Escouade > Synergies, « qui assiste qui » | `publishable AND assist_known` | `buildSquadAssistPairs` rend `nil` sur `measured == 0`. Avant ce lot : aucun log, aucun compteur. Le front ne monte le tableau que si `assist_pairs != nil` (`apps/web/src/features/squad/SquadSynergiesPage.tsx:166`). |
| Vue match, bloc assistances | `publishable AND assist_known` (`Q21dAssistPairs`, `internal/platform/duckdb/match_view_repo_assist_pairs.go:69,83`) | `measured_deaths = 0` ; l'ecran affiche « non disponibles ». |

Paires mesurables par mois :

| mois | morts mesurees | dont assistant nomme |
|---|---|---|
| 2026-08 | **0** | 0 |
| 2026-07 | 608 | 168 |
| 2026-06 | **0** | 0 |
| 2026-05 | **0** | 0 |
| 2026-04 | 728 | 301 |
| 2026-03 | 9 015 | 2 961 |
| 2026-02 | 9 764 | 2 797 |

**C'est le silence qui a coute les cinq mois.** Le defaut n'a ete vu que parce qu'un
utilisateur a demande pourquoi un tableau manquait.

---

## 5bis. LES TOKENS NE SONT PAS UNE CONTRAINTE — CORRECTION DE LA VERSION 1

`fetchFilmManifest` appelle `<hote>/<jeu>/films/matches/<matchID>/spectate`
(`internal/sync/haloclient/halo_client_film.go:113`) : **l'endpoint est indexe par MATCH,
aucun xuid**. `GetFilmChunks` passe par le meme manifeste. La selection de la passe en ligne
n'a aucun filtre de participant. Donc :

- **n'importe quel token Halo vivant recupere n'importe quel film** ; `--gamertag` ne sert
  qu'a choisir quel jeu d'identifiants du magasin utiliser ;
- une seule passe couvre les quatre joueurs **parce que le registre est partage**, pas parce
  qu'un joueur figure dans tous les matchs.

La version 1 concluait le contraire a partir du tableau ci-dessous. Le tableau est exact ; la
conclusion qu'il portait ne l'etait pas.

| joueur (xuid) | matchs depuis avril | dont sans passe de film |
|---|---|---|
| JGtm (2533274823110022) | 394 | 374 |
| Madina97294 (2533274858283686) | 201 | 182 |
| Chocoboflor (2535469190789936) | 189 | 173 |
| XxDaemonGamerxX (2533274833178266) | 10 | 10 |

---

## 6. CE QUI A ETE CONSTRUIT, ET CE QUE LA REVUE A CORRIGE

Revue adversariale du 2026-08-29 : **5 relecteurs en contexte frais, aveugles entre eux,
lecture seule**. Deux P0, chacun trouve par DEUX relecteurs independamment, tous deux
verifies sur pieces avant d etre retenus. **Les deux sont corriges.**

### P0-1 — l etape ne s executait sur AUCUN chemin de production

`runKillSource` obtient `GetFilmChunks` par assertion de type. Seul `HaloAPIClient` portait
cette methode : `PooledHaloClient` (chemin serveur) ne l avait pas, et `cachedHaloClient` —
pose SYSTEMATIQUEMENT sur le chemin V1 — non plus. L assertion echouait partout et l etape
sortait **sans log ni compteur**. Les deux garde-rails poses restaient verts : ils ne
verifiaient que « le hook est non nil » et « la ligne d appel existe dans le fichier ».

**Corrige** : `GetFilmChunks` en passe-plat sur les deux enveloppes (sans toucher a
l interface `HaloClient`, laissee optionnelle pour ne pas obliger les mocks a la porter) ;
l interface etroite est desormais EXPORTEE (`killcollector.FilmChunkFetcher`) pour que le
cablage et son garde-rail assertent la meme ; trois assertions **de compilation** sur les
types concrets ; un test qui refait l assertion sur les clients tels que le moteur les
construit ; un test qui verifie que l enveloppe DELEGUE (une methode qui rendrait toujours
`found = false` passerait les assertions tout en desarmant l etape) ; et un WARN + compteur
`killsource_postsync_client_sans_film` la ou il y avait un `return` nu.

### P0-2 — meme corrigee, la selection n aurait jamais servi 2026

Backlog trie du plus vieux au plus recent, 8 par cycle, **sans marqueur terminal** : un film
expire ne produit aucune ligne, donc le match restait a sa place. Les memes matchs de **2021**
auraient ete retentes indefiniment.

**Corrige, et LE DEPOT PORTAIT DEJA LA REPONSE** — c est le vrai enseignement. L etape 1.55
(weapon kills), qui tourne AVANT celle-ci sur le meme film, pose deja
`MBitWeaponKillsNoFilm` quand 343 ne sert plus le film. Mesure :

| annee | candidats | deja marques « film absent » |
|---|---|---|
| 2021 | 11 | **11** |
| 2022 | 2 | **2** |
| 2023 | 529 | **529** |
| 2024 | 39 | **39** |
| 2025 | 3 | 0 |
| 2026 | 415 | 1 |

**581 des 999 candidats sortent par ce seul filtre**, et ce sont exactement les
irrecuperables. Aucun bit nouveau, aucune migration, aucune ecriture supplementaire. L ordre
est passe du plus RECENT au plus vieux (les recuperables d abord), avec un horizon de lecture
borne et une jauge de backlog comptee SANS borne — une jauge tronquee decrirait l horizon, pas
le retard.

### P1/P2 corriges dans la meme ronde

| constat | correction |
|---|---|
| Racine de cache **lue** != racine **ecrite** (3 relecteurs) | UNE racine, celle du moteur si elle existe. `LocalFilmCache.RootDir()` expose. Les chunks telecharges rejoignent donc bien les existants. |
| Preparation des dossiers du cache recopiee 3 fois | `filmcache.EnsureDirs` — la disposition n est declaree que dans ce paquet, sa CREATION y vit aussi. |
| Budget de cycle expire compte en `killsource_erreurs_decodage` | Le budget est devenu une notion du COLLECTEUR (`WithBudget`), verifiee entre deux matchs. Plus d annulation de contexte, donc plus d arret nominal compte comme erreur. Compteur dedie `killsource_budgets_epuises`. |
| `clock.lap("kill_source", 0)` code en dur | `RunPostSync` rend le nombre de matchs ecrits ; la trace de cycle le publie. |
| Compteurs d assistances non titre-aware alors que `halo_5` est actif | `IncCounterT` + `ctxkeys.TitleSlug`. Test qui verifie que la cle NUE ne bouge pas sur un contexte halo_5. |
| `TestRunPostSync_CapabilityAbsente_EtapeVide` testait une propriete plus faible que celle annoncee | Vise desormais halo_5 (titre reel qui declare ses mappings sans `film.kill_source`). **Verifie par mutation** : retirer la porte fait echouer le test — ce qui n etait pas le cas avant. |
| La requete de backlog n etait verifiee que par `strings.Contains` | Fichier d integration dedie : la requete tourne sur une base migree. **Quatre mutations verifiees** (marqueur retire, ordre inverse, filtre credit retire, horizon applique a la jauge) : les quatre font echouer les tests. |
| `afficherPlanEnLigne` n imprimait jamais sa ligne d elision (>= 9 entrees) et affichait un compte negatif (6-7) | Elision calculee avant la boucle. |
| Aucun test sur la validation des drapeaux de la CLI | `cmd_backfill_killsource_test.go`, cas negatifs ET cas positif. **Mutation verifiee.** |
| Quatre erreurs d auth avalees dans la sonde | Chacune journalisee ; le message final porte la cause. |
| La sonde partait de `match_kill_events_latest` | Elle part du REGISTRE : c est ce biais qui avait sous-estime le perimetre (374 annonces contre 999 reels). |

### Etat des gates

`go build ./...` propre · `go test` vert sur `internal/sync/...`, `internal/service/...`,
`cmd/levelup` · `go test -tags=integration ./internal/persist ./internal/sync
./internal/sync/killcollector` **vert** (anti-ART) · `golangci-lint` **0 issue** sur les
paquets touches. Seul echec d archlint restant :
`internal/platform/duckdb/killsource_class_repo_test.go`, fichier NON SUIVI d une autre
session — hors perimetre (regle 7).

### Un besoin distinct, identifie sur precision utilisateur (2026-08-30)

Le rattrapage que l utilisateur voulait n etait PAS le decodage des assistances : c etait
**telecharger et CONSERVER les films et manifestes manquants**, periodiquement, avant qu ils
n expirent. Les deux besoins se recouvrent en partie (la passe de decodage archive ce qu elle
telecharge) mais leurs perimetres different :

| | `levelup archive-films` | `backfill-killsource --online` |
|---|---|---|
| but | sauver les octets | produire `assist_known` |
| perimetre | tout match sans film en local — **917 sur 1 948** | les matchs a redecoder — **417** |
| cout par match | un manifeste + N chunks | idem PLUS ~19 s de decodage |
| base partagee | lecture seule | ecriture |
| ordre | du plus VIEUX au plus recent | du plus RECENT au plus vieux |

**Les deux ordres sont opposes, et c est le meme raisonnement applique a deux buts.** Le
decodage prend les recents parce que les vieux sont deja perdus et que l ecran montre les
recents. L archivage prend les vieux parce qu il court apres l expiration : parmi les films
ENCORE servis, ce sont eux a qui il reste le moins de temps.

Ecart d archivage mesure le 2026-08-30 (manifeste absent du cache) :

| annee | matchs | sans film en local |
|---|---|---|
| 2021 | 11 | 11 |
| 2022 | 2 | 2 |
| 2023 | 536 | 527 |
| 2024 | 54 | 39 |
| 2025 | 415 | 3 |
| 2026 | 930 | 335 (en cours de resorption) |
| **total** | **1 948** | **917** |

**La commande ne saute PAS par defaut les matchs marques « film absent ».** Un manifeste coute
une requete, et c est la seule facon de transformer un marqueur pose il y a des mois en fait
mesure : **579 matchs anterieurs a 2025 n ont JAMAIS ete sondes** (la sonde du 2026-08-29 ne
tirait que dans 2026, cf. §4). `--sauter-marques` les exclut quand on veut economiser.

**Une affirmation ecrite puis dementie par son propre test.** L en-tete de la commande disait
« elle tourne serveur allume » parce qu elle ne fait que lire. Verifie pendant qu une passe de
backfill tenait la base : **c est faux** — DuckDB n autorise qu un processus par fichier
(regle 4), l ouverture echoue avec « File is already open in ... (PID …) ». Corrige. Ce qui
reste vrai : la passe ne peut rien corrompre, et l interrompre ne laisse jamais un film a
moitie lisible (chunks d abord, manifeste EN DERNIER — c est le marqueur de commit).

**Cout disque mesure** : 24,1 Mo par film en moyenne (sur 1 093 archives). Projection pour
tout ce qui manque : **~18,3 Go a ajouter**, pour 86,7 Go libres.

---

### RESULTATS DES DEUX PASSES (2026-08-30) — jouees, serveur arrete

**Passe de decodage** (`backfill-killsource --online --gamertag JGtm --films-only`) :
417 matchs, **391 ecrits**, 35 242 morts, 395 films telecharges et **395 archives, 0 erreur
d archivage**, 4 films sans kill-feed, 2 erreurs de decodage — 1 h 48.

**Les 2 erreurs de decodage etaient TRANSITOIRES** : rejouees le lendemain, elles passent
(2 ecrits, 182 morts). C est la demonstration qu il aurait ete FAUX de poser un marqueur
terminal sur une erreur de decodage — on aurait perdu ces matchs pour toujours. Seul le film
SANS KILL-FEED est terminal par nature.

**Backlog residuel : 4 matchs**, tous « film present, aucun chunk kill-feed ». Ils sont
retentes a chaque cycle : cout mesure **~4 s au total et AUCUN reseau**, leurs films etant
desormais archives en local. Boucle bornee et bon marche — consignee, non traitee (un
marqueur dedie demanderait un bit sur `backfill_completed` pour 4 matchs).

Couverture finale **avril -> aout 2026 : 86/86, 86/86, 65/65, 149/149, 8/8**.

| mois | `assist_known` avant -> apres | paires assistant->tueur | couverture film |
|---|---|---|---|
| 2026-08 | 0 -> **786** | 0 -> **339** | 0/8 -> **8/8** |
| 2026-07 | 896 -> **12 824** | 168 -> **2 734** | 8/149 -> **149/149** |
| 2026-06 | 0 -> **5 545** | 0 -> **1 066** | 0/65 -> **65/65** |
| 2026-05 | 92 -> **7 231** | 0 -> **1 623** | 1/86 -> **85/86** |
| 2026-04 | 806 -> **6 800** | 301 -> **2 146** | 11/86 -> **85/86** |

**Passe d archivage** (`archive-films --gamertag JGtm`) : 582 films manquants, **3 sauves,
579 EXPIRES cote 343, 0 erreur** — 2 min 55.

**Le fait que cette passe etablit, et que personne n avait mesure** : les 579 films anterieurs
a 2025 sont **definitivement perdus**. Le marqueur `MBitWeaponKillsNoFilm` du pipeline etait
donc EXACT — verifie par 579 reponses 404 independantes, et non plus suppose. Preuve
operationnelle : `archive-films --dry-run --sauter-marques` rend **0** manquant, ce qui
signifie que l ensemble « expire » et l ensemble « marque » coincident exactement.

**Etat final du cache** : **1 369 films** (951 au debut de la journee, **+418 sauves**),
32,32 Go, 85,2 Go libres.

**Recette du rattrapage periodique** — `--sauter-marques` evite de re-sonder les 579 morts
(2 min 55 a chaque passe) tout en prenant tout match nouveau dont le film n est pas encore en
cache :

```
levelup archive-films --gamertag JGtm --sauter-marques
```

Le fil de l eau (etape 1.57) archive deja les films des matchs qu il decode : cette passe est
le filet, pas le mecanisme principal.

---

### Ce qui reste ouvert

- **Le rattrapage n est pas joue.** Perimetre reel MESURE : 999 candidats bruts, dont
  **417 apres exclusion du marqueur terminal** — 414 en 2026, 3 en 2025, et **plus aucun match
  anterieur a 2025**. Recette, serveur ARRETE :

  ```
  levelup backfill-killsource --online --gamertag JGtm --dry-run
  levelup backfill-killsource --online --gamertag JGtm --limit 20   # lot d essai
  levelup backfill-killsource --online --gamertag JGtm              # tout
  ```

  Le lot d essai porte desormais sur les 20 matchs les plus RECENTS sans passe de film — pas
  sur des matchs de 2021 comme dans la version 1 de cette recette.
- **A surveiller au premier cycle** : `killsource_postsync_backlog_restant` et
  `killsource_postsync_client_sans_film` sur `/debug/vars`. Le second doit rester a zero ; s il
  monte, un client injecte a l execution ne porte pas la capacite film.
- **Non traite (regle 7)** : le chemin nominal de `RunPostSync` (capability presente + backlog
  non vide) n a toujours pas de test de bout en bout — il demande une base, un roster et un
  film. Les briques qu il enchaine sont couvertes une par une.

---

## 7. DECOUVERTE MESUREE — `xuid_aliases` (correction d'une affirmation fausse en v1)

La v1 affirmait que `xuid_aliases` avait cesse d'etre alimentee en avril 2026 et en faisait
une piste a instruire. **C'est faux** : la table recoit 2 683 lignes apres avril 2026. Mesure
par alimentateur :

| `source` | lignes | `last_seen` max |
|---|---|---|
| `highlight_events` | 6 714 | **2026-04-07** |
| `sync` | 1 388 | 2026-08-27 |
| `world_leaderboard` | 1 223 | 2026-06-14 |
| `repair_killer_victim_pairs` | 72 | 2026-05-08 |

Un seul alimentateur sur quatre s'arrete — et sa date coincide **au jour pres** avec le
dernier manifeste de film. L'en-tete de `internal/sync/killcollector/roster.go:47-50`
documente deja cet etat comme connu, delibere et compense (les gamertags du kill-feed portes
par `killer_victim_pairs` prennent le relais ; couverture mesuree 18 219 xuids, 36 masques).
**Il n'y a donc rien a instruire ici** : la seule information nouvelle est la coincidence de
date, consignee.

---

## 8. OUTILS ECRITS POUR CE REGISTRE

- `cmd/diag_film_avail` — sonde de disponibilite des films (manifeste seul). **Jetable.**
  Deux reserves documentees en §4 : elle ne tire que dans 2026, et sa CTE part de
  `match_kill_events_latest` (un match sans kill-event lui est invisible).
