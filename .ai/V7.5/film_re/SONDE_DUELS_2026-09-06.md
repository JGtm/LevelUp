# SONDE — duels et portée des engagements depuis le film Infinite

> Date : 2026-09-06. Branche `feat/duels` (worktree `LevelUp-wt-duels`).
> Instrument : `internal/analysis/replay/duels_sonde_research_test.go` (+ `_mesures_test.go`),
> composé UNIQUEMENT de décodeurs de production (`ScanFilmWeaponDamages`,
> `ScanBipedPositions`, `ScanDeaths`, `buildLifeSpans`, `bestDeathOffset`,
> `nameLivesByDeaths`). Hors ligne : aucune base ouverte, aucun roster.
> Films : 4 cartes d'arène du cache local, tous chunks balayés.

## La question

Peut-on compter les duels d'un joueur et ceux qu'il a gagnés ? Critère retenu avec
l'utilisateur : **la réciprocité du dégât**. Un duel est un échange où le dégât circule dans
les deux sens ; un tir dans le dos, où la victime n'a jamais rien placé, est une élimination.
Le critère ne suppose aucune intention et ne modélise aucune ligne de vue : il se mesure.

## Seuils écrits AVANT la mesure

| Mesure | Seuil | Ce qu'il protège |
|---|---|---|
| M0 pont d'index | >= 80 % | sans lui, rien d'autre n'a de sens |
| M1 réciprocité | >= 40 % | un joueur d'arène meurt en combat, pas en exécution |
| M2 rapport au témoin | >= 3 | la réciprocité doit être un fait, pas une densité |
| M3 distance résolue | >= 70 % | en dessous, la portée est un appoint, pas un axe |

## Résultats

| film | carte | morts | 0xC0 t0 | 0xC0 t1 | M0 | M1.0 fatal | écart médian | M1 duels (5 s) | M2 témoin | M3 distance |
|---|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| 000d5950 | Cliffhanger | 90 | 428 | 124 | 64,7 % | 40,0 % | 433 ms | 4/36 = 11,1 % | 0/36 | 97,2 % |
| 01e1f945 | Catalyst | 97 | 217 | 40 | 43,3 % | 10,3 % | 1601 ms | 0/10 = 0 % | 0/10 | 100 % |
| 00502e52 | Bazaar | 94 | 322 | 112 | 80,7 % | 50,0 % | 231 ms | 6/47 = 12,8 % | 0/47 | 95,7 % |
| 7344d24f | Vagabond | 117 | 91 | 65 | 54,9 % | 14,5 % | 884 ms | 1/17 = 5,9 % | 0/17 | 100 % |

M5, voie bouclier (fraction `i5`, répliquée dans le record de position quand elle change) :

| film | lectures | chutes | chutes/mort | oracle victime | 2 tiers ou plus en chute |
|---|---:|---:|---:|---:|---:|
| 000d5950 | 27 404 | 315 | 3,5 | 44,4 % | 10,0 % |
| 01e1f945 | 47 951 | 735 | 7,6 | 69,1 % | 48,4 % |
| 00502e52 | 30 035 | 444 | 4,7 | 60,6 % | 18,1 % |
| 7344d24f | 67 518 | 1 105 | 9,4 | 76,1 % | 46,9 % |

## Verdict

**M1 ÉCHOUE, et l'échec est quantifié.** La réciprocité mesure 0 à 12,8 % contre un seuil de
40 %. La cause n'est PAS le critère : c'est la MAIGREUR du flux de dégâts.

Le film n'émet que **91 à 428 enregistrements `damage_aftermath`** pour 90 à 117 morts, soit
0,8 à 4,8 par mort — quand une seule élimination au fusil de combat en demande quatre.
Le sous-type 1 (`damage_section_response`), que le décodeur de production écarte, n'ouvre
aucun réservoir : 40 à 124 paquets de plus, même ordre de grandeur. **Il n'y a pas de flux de
dégâts complet à décoder dans le film : il n'y est pas.**

L'arithmétique de la perte se referme. Voir un duel exige que DEUX dégâts survivent à
l'échantillonnage, un dans chaque sens. Avec P(le dégât fatal est capturé) mesurée à
0,40 (Cliffhanger) et 0,50 (Bazaar), la réciprocité attendue sous indépendance vaut 0,16 et
0,25 ; on mesure 0,11 et 0,13. Le compte des duels serait donc un **sous-comptage de 4 à 8x**,
et pas un sous-comptage uniforme — il varierait d'un facteur 5 d'un match à l'autre, ce qui
est pire qu'un biais constant.

**Ce que la sonde valide en revanche, et qui reste acquis :**

1. **La base d'atterrissage vaut 512, et elle se DÉTACHE.** Le critère de calibration est
   « le slot était-il VIVANT à l'instant du dégât », pas « le slot existe » : l'existence est
   satisfaite par presque n'importe quelle base et ne note que la densité des slots. Sous le
   critère de vivacité, 512 sort premier sur les 4 films avec un rapport de 2,1x à 2,6x au
   deuxième candidat. C'est la même logique que le plateau de `bestDeathOffset`.
2. **La jointure dégât fatal -> fin de vie est RÉELLE là où elle existe** : écart médian
   231 ms et 433 ms sur les deux films riches. Les deux flux parlent bien du même fait.
3. **M2 = 0 sur les quatre films.** En déplaçant la fenêtre de riposte de 37 s vers le passé,
   la réciprocité tombe à zéro partout. La spécificité est parfaite : ce qu'on détecte est
   vrai, on n'en détecte simplement presque rien. C'est un problème de RAPPEL, pas de bruit.
4. **M3 = 95,7 % à 100 %.** La distance de l'engagement se résout presque toujours. C'est la
   seule mesure verte sans réserve.
5. **La médiane de portée des éliminations est stable sur quatre cartes différentes** :
   5,8 / 6,3 / 6,8 / 6,9 m. Une constante de jeu, pas un artefact de carte.
6. **L'axe vertical est RÉEL — et la première mesure était aveugle.** L'écart entre distance
   3D et distance plane (0,0-0,2 m) ne mesure pas le dénivelé : à 6 m, un mètre de hauteur ne
   déplace la distance 3D que de 8 cm. Remesuré sur |dz| le même jour : **médiane 0,4 / 0,9 /
   0,4 / 0,7 m, et 25,7 / 40,0 / 15,6 / 23,5 % des engagements à plus d'un mètre de dénivelé.**
   Un engagement sur quatre se joue avec un étage d'écart. Le signe (qui était au-dessus) reste
   à mesurer côté tueur — `kill_positions` le permet en SQL pur.

## La voie bouclier : plus dense, pas encore suffisante

La fraction de bouclier est deux à dix fois plus dense que le flux de dégâts (3,5 à 9,4
chutes par mort). Son oracle de validité — « la victime a-t-elle perdu du bouclier dans les
3 s avant de mourir ? », à quoi la réponse devrait être « toujours » — plafonne à **44-76 %**.
Et sa discrimination se dégrade là où sa densité est la meilleure : sur Catalyst et Vagabond,
deux autres joueurs ou plus perdent du bouclier dans la même fenêtre dans 47-48 % des cas.
Une chute de bouclier ne désigne donc pas un adversaire.

**MAIS la sonde n'a pas mesuré la bonne chose, et c'est délibéré.** Elle s'interdisait la base,
donc elle ignorait QUI était le tueur pour 50 à 90 % des morts. Or le tueur est connu par
ailleurs, hors film : `match_kill_events` le porte par le kill-feed, à 97,6 % de couverture.

La question décisive n'est donc pas celle que M5 a posée. Elle est :
**pour les morts dont le kill-feed nomme le tueur, le BOUCLIER DU TUEUR a-t-il chuté pendant
la fenêtre d'engagement ?** Si oui, la victime lui a rendu des coups, et c'est un duel. Cette
mesure exige le pont slot -> xuid (`ResolveSlotXUID`) et donc le roster, donc la base. Elle
n'a pas été faite ici. C'est le gate du lot 1 du plan.

## Reproduire

```bash
cd apps/go-api
CGO_ENABLED=0 \
  DUELS_FILM=<repo>/data/cache/film_chunks/000d5950 DUELS_MAP=Cliffhanger \
  go test ./internal/analysis/replay -run TestSondeDuels -v -timeout 900s
```

Un film par process (verrou `filmdec.LockProcessDecode` pris par la sonde). Coût mesuré :
1,6 à 3,3 s par film, RAM négligeable — aucune cuisson d'artefact n'est déclenchée.
