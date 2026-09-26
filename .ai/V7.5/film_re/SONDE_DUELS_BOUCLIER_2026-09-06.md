# SONDE n°2 — le bouclier DU TUEUR décide-t-il des duels ?

> Date : 2026-09-06. Branche `feat/duels-lot1` (worktree `LevelUp-wt-duels-lot1`).
> Lot 1 de `.ai/PLAN_DUELS_PORTEE_2026-09-06.md`. Suite directe de
> `.ai/V7.5/film_re/SONDE_DUELS_2026-09-06.md` (sonde n°1, hors ligne).
> Instrument : `apps/go-api/internal/sync/killcollector/duels_bouclier_research_test.go`
> (+ `duels_bouclier_mesures_test.go`), composé des décodeurs et du pont de PRODUCTION
> (`filmdec.ScanBipedPositions` avec `CaptureDirs`, `replay.ScanClockOrigin`,
> `replay.ScanDeaths`, `replay.ScanPlayerIndices`, `replay.ResolveSlotXUID`,
> `replay.BuildKillPositions`). Base ouverte en LECTURE par `duckdb.OpenReadForQuery`.
> Aucune écriture, aucun réseau, aucune cuisson d'artefact.

## La question

La sonde n°1 a rejeté la réciprocité du dégât : le film n'émet que 0,8 à 4,8
`damage_aftermath` par mort, et le compte des duels serait un sous-comptage de 4 à 8x. Mais
elle s'interdisait la base, donc elle ignorait QUI était le tueur pour 50 à 90 % des morts.
Or le tueur est connu hors film, par le kill-feed (`match_kill_events_latest`).

**La question décisive devient donc** : pour les morts dont le kill-feed nomme le tueur, le
BOUCLIER DU TUEUR a-t-il chuté pendant la fenêtre d'engagement ? Si oui, la victime lui a
rendu des coups, et c'est un duel. Le bouclier est répliqué DANS le record de position à
chaque changement — deux à dix fois plus dense que le flux de dégâts.

## Seuils écrits AVANT la mesure (décision D2 du plan, non renégociés)

| Mesure | Ce qu'elle est | Seuil |
|---|---|---|
| **A** pont du tueur | part des kills du feed dont le TUEUR est localisé à T par le pont de production | `>= 80 %` |
| **O** oracle victime | part des kills où le bouclier de la VICTIME chute dans [T-2 s, T] — **plafond de capture** du canal, pas un résultat | (dénominateur de B) |
| **B** bouclier du tueur | part des kills où le bouclier du TUEUR chute dans [T-2 s, T] | `B/O` dans **[0,35 ; 0,90]** |
| **T** témoin | B avec la fenêtre reculée de 37 s | `B/T >= 3` |
| **D** discrimination | parmi les ADVERSAIRES du tueur en chute dans la fenêtre, la victime est-elle la seule ? | `>= 60 %` |

Fenêtre **2 s** et non 5 : la sonde n°1 a mesuré la réciprocité IDENTIQUE à 2, 3 et 5 s — la
riposte vit dans les deux premières secondes ou n'existe pas.

## Résultats — par match

Numérateur et dénominateur partout : les pourcentages ne s'additionnent pas, les comptes si.

| film | carte | kills du feed | A pont du tueur | O oracle victime | B bouclier du tueur | B/O |
|---|---|---:|---:|---:|---:|---:|
| 000d5950 | Cliffhanger | 93 | 86/93 = 92,5 % | 38/93 = 40,9 % | 14/93 = 15,1 % | 0,37 |
| 01e1f945 | Catalyst | 104 | 86/104 = 82,7 % | 59/104 = 56,7 % | 35/104 = 33,7 % | 0,59 |
| 00502e52 | Bazaar | 95 | 86/95 = 90,5 % | 55/95 = 57,9 % | 31/95 = 32,6 % | 0,56 |
| 7344d24f | Vagabond | 117 | 112/117 = 95,7 % | 85/117 = 72,6 % | 55/117 = 47,0 % | 0,65 |
| **cumul** | 4 cartes | **409** | **370/409 = 90,5 %** | **237/409 = 57,9 %** | **135/409 = 33,0 %** | **0,57** |

| film | témoin (fenêtre -37 s) | B sur la même population | B/témoin | D discrimination |
|---|---:|---:|---:|---:|
| 000d5950 | 1/83 = 1,2 % | 12/83 = 14,5 % | 12,00 | 8/14 = 57,1 % |
| 01e1f945 | 6/96 = 6,2 % | 32/96 = 33,3 % | 5,33 | 14/35 = 40,0 % |
| 00502e52 | 3/85 = 3,5 % | 25/85 = 29,4 % | 8,33 | 20/31 = 64,5 % |
| 7344d24f | 12/112 = 10,7 % | 54/112 = 48,2 % | 4,50 | 29/55 = 52,7 % |
| **cumul** | **22/376 = 5,9 %** | **123/376 = 32,7 %** | **5,59** | **71/135 = 52,6 %** |

Densité du canal, pour mémoire : 3,3 / 6,7 / 4,7 / 9,2 chutes de bouclier **rattachées à un
joueur** par kill du feed — deux à dix fois le flux de dégâts, comme la sonde n°1 l'annonçait.

Qualité du pont, lue et non supposée : 0 désaccord d'index et 0 collision de slot sur les
quatre films ; 26 à 31 chunks concordants pour la table identité -> index ; 90 à 117 morts
appariées au calage du fil.

## Verdict — **NO-GO pour le lot 7**

| Gate | Cumul | Verdict |
|---|---|---|
| `A >= 80 %` | 90,5 % (370/409) | **PASSE** |
| `B/O` dans [0,35 ; 0,90] | 0,57 (135/237) | **PASSE** |
| `B/témoin >= 3` | 5,59 (123/22) | **PASSE** |
| `D >= 60 %` | **52,6 % (71/135)** | **ÉCHOUE** |

Trois gates sur quatre passent, et ils passent nettement. **Le gate qui échoue est celui de la
DISCRIMINATION, et c'est le seul dont l'échec n'est pas réparable par un réglage.**

**Ce que la mesure établit, et qui est solide** :

1. **Le pont d'identité tient.** A = 90,5 % au cumul, et 100 % des tueurs du feed ont au moins
   un slot au pont : les 9,5 % perdus sont des morts où le tueur n'a pas d'échantillon de
   position dans la tolérance à l'instant T, pas des tueurs inconnus. C'était le préalable de
   tout le reste et il est acquis.
2. **Le signal est RÉEL.** B/témoin = 5,59 au cumul, et de 4,50 à 12,00 sur chacun des quatre
   films. En reculant la fenêtre de 37 s, la chute du bouclier du tueur tombe de 32,7 % à
   5,9 %. Ce que B détecte n'est pas une densité d'événements : c'est un fait de l'engagement.
3. **Le taux normalisé est dans la bande attendue.** B/O = 0,57, très au centre de
   [0,35 ; 0,90] : rapporté au plafond de capture du canal, un tueur sur deux perd du bouclier
   pendant la fenêtre. C'est exactement la proportion qu'on attend d'un mélange duels /
   exécutions dans un jeu d'arène.

**Ce qui condamne la feature** : une chute de bouclier ne DÉSIGNE PERSONNE. Dans 47,4 % des
fenêtres où le tueur a perdu du bouclier, au moins un AUTRE adversaire du tueur a perdu du
bouclier dans les mêmes deux secondes. Le signal dit « le tueur a pris des coups », il ne dit
pas « il les a pris DE SA VICTIME ». Or c'est précisément l'affirmation que la feature
publierait : « ce kill était un duel, contre ce joueur-là ».

Et la dégradation n'est pas uniforme : D vaut 64,5 % sur Bazaar (le seul film qui passe le
gate) contre 40,0 % sur Catalyst. Le même écart d'un facteur qui invalidait déjà le comptage
par réciprocité — un biais qui varie d'un match à l'autre est pire qu'un biais constant, parce
qu'il rend deux matchs incomparables. Le compte de duels d'un joueur dépendrait de la carte.

C'est le même mur que la sonde n°1 avait rencontré par l'autre bout, et il vaut d'être nommé
précisément, parce qu'il dit ce qu'il faudrait pour le franchir :

- la **sonde n°1** avait le lien (le dégât nomme son auteur) mais pas le rappel (0,8 à 4,8
  événements par mort) ;
- la **sonde n°2** a le rappel (3,3 à 9,2 chutes par mort) mais pas le lien (une chute n'a pas
  d'auteur).

Les deux canaux du film sont exactement complémentaires dans ce qu'ils manquent. Aucune
combinaison des deux ne referme l'écart : les rares dégâts nommés sont déjà comptés par la
sonde n°1, et les chutes de bouclier restent anonymes. **Il n'y a pas de troisième réglage à
essayer sur ces deux canaux.**

## Réserves — ce que cette mesure ne dit pas

- **L'éligibilité du témoin est bornée au domaine observable.** Un kill survenu dans les 39
  premières secondes du match verrait sa fenêtre reculée tomber avant la première lecture de
  bouclier, là où personne ne peut chuter : ces kills sont EXCLUS du dénominateur du témoin
  (409 kills, 376 éligibles). Sans cette borne, le témoin tomberait vers zéro par construction
  et `B/témoin` serait un artefact de bord. C'est un durcissement par rapport à la sonde n°1,
  qui ne l'appliquait pas.
- **La discrimination est mesurée AVEC les équipes**, ce que la sonde n°1 ne pouvait pas faire.
  Un joueur dont `match_participants.team_id` est NULL ne peut être ni candidat ni adversaire :
  sur ces quatre films les 8 participants de chaque match ont une équipe, donc la réserve est
  théorique ici.
- **Le lien slot -> joueur est celui du pont de production**, `fire = nil` (sans fermeture par
  tir) — la version conservatrice : moins de slots nommés, jamais un slot nommé à tort. Une
  chute sur un slot hors pont est ignorée, et cette perte est exactement ce que A quantifie.
- **Quatre films d'arène 4v4.** Rien n'est dit des modes à grand effectif (BTB), où la
  discrimination ne peut que se dégrader — plus d'adversaires, plus de chutes simultanées.
- **`publishable` est lu et compté, mais ne filtre pas** : 0 ligne non publiable sur les quatre
  matchs, la question ne se pose pas ici.
- **B est une borne basse**, plafonnée par le taux de capture du canal (O = 57,9 %). C'est
  pourquoi le gate se lit normalisé — et ce gate-là passe. Ce n'est pas le rappel qui ferme la
  feature, c'est la spécificité de l'attribution.

## Ce qui rouvrirait le dossier (condition de reprise)

Un canal qui porte l'AUTEUR du dégât avec une densité comparable à celle du bouclier :

- un flux de dégâts dense (le film n'en porte pas — mesuré, sonde n°1) ;
- **ou un compteur d'état ECS répliqué** qui daterait le dégât par couple (auteur, blessé) ;
- ou une source hors film (API de télémétrie par engagement) — aucune connue à ce jour.

Reporté au `.ai/V7.5/REGISTRE_REPORTS.md` avec cette condition.

## Reproduire

```bash
cd apps/go-api
GOCACHE=<worktree>/.gocache-lot1 CGO_ENABLED=1 \
  DUELS_DATA_ROOT=C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration \
  DUELS_MATCH=000d5950 DUELS_MAP=Cliffhanger \
  go test ./internal/sync/killcollector -run TestSondeDuelsBouclier -v -timeout 900s -count=1
```

Puis `01e1f945`/`Catalyst`, `00502e52`/`Bazaar`, `7344d24f`/`Vagabond`. **Un match par
process** (verrou `filmdec.LockProcessDecode`). CGO exigé (DuckDB). Coût mesuré : 1,6 à 2,3 s
par match, RAM négligeable — aucune cuisson d'artefact n'est déclenchée. Le test `t.Skip` sans
les trois variables d'environnement, donc il ne coûte rien à la CI.

La ligne `CUMUL kills=… pont=… A=… O=… B=… eligibles=… Belig=… T=… D=…` de chaque exécution
porte les entiers bruts : le cumul du tableau ci-dessus est leur somme, jamais une moyenne de
pourcentages.
