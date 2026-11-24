# Rapport bl-score : A.5, A.6, A.4, A.3 (2026-11-23)

Worktree `dec-v160-bl-score`, branche `piste/bl-score`, rien de commite. « vérifié » = commande rejouée ou pièce relue ;
« déduit » = raisonné sans preuve. Aucun identifiant de joueur ni pseudo ; films par identifiant court.

Suppositions : « accord exact » = le dernier total de `scoreTimeline.players[].score` égale `personal_score` de la copie
de l'API (22/11), joueur par joueur ; un joueur absent du tableau compte comme inexact. Binaire « avant » = source
figée avant toute modification (`src_avant`) ; « après » = worktree.

## A.5 : pénalité de suicide ou de trahison publiée (FAIT, vérifié)

**Cause prouvée** (sonde sur les enregistrements bruts, 1 595 films de l'archive) : le film écrit la pénalité comme une
baisse de exactement 100 par unité, dans une émission où le compteur de suicides (composant 4, valeur A) ou de
trahisons (valeur B) monte de la même unité : 1 282 pénalités dans 635 films, 2 seulement où le compteur monte sans que
la baisse vaille ses unités. Les autres baisses (1 350) n'ont aucun compteur : vidage de ligne d'un joueur qui part
(score à zéro), changement d'occupant, bruit. La règle distingue donc les trois par un fait, comme demandé.
Deux défauts annexes prouvés dans la même émission : un score qui tombe sous zéro (436 émissions négatives, 340 avec
le compteur) était jeté comme ancrage parasite ; un score qui tombe à zéro par pénalité était pris pour un vidage de ligne.

**Code** : `internal/facts/objectives/score_penalties.go` (neuf) + branchements dans `score.go`, `scoreboard_stats.go`,
`named_series.go`, `named.go`, `seat_cuts.go`. La suite est choisie sur la valeur brute (valeur lue + 100 par pénalité
subie) puis chaque point reprend sa valeur lue. Pas de goroutine, pas de seuil (100 = prix de la réponse dans la table).
Ce qui n'est pas confirmé est inchangé et reste écarté par la plus longue suite, comme avant.

**Tests** : `score_penalties_test.go`, 11 tests sur données construites (suicide qui recule, trahison dernière du joueur,
sous zéro, zéro par pénalité, vidage sans compteur écarté, baisse de 50 avec suicide non confirmée, deux unités, première
émission, frontière de manche, occupant partant/arrivant, équivalence sans pénalité). Mutation jouée : lecture désactivée,
6 tests rouges.

**Mesure** (vérifié, `cmp.py`, `cmp2.py`) :

| Échantillon | Exacts avant | Exacts après | Exacts devenus inexacts |
|---|---|---|---|
| 20 films du corpus d'équivalence (modes variés) | 228 / 324 | 228 / 324 | 0 |
| 67 films dont un total change (parmi les 1 595 scannés ; 71 en changent, 67 sont dans la copie de l'API) | 864 / 1 030 | 928 / 1 030 | **0** |

Sur les 20 films aucun total final ne change (les joueurs pénalisés marquent ensuite), donc rien à gagner : 11 documents sur 20
changent quand même, par la seule courbe. Sur les 67 films, les 64 totaux modifiés deviennent tous exacts (64 sur 64), dont
11 totaux négatifs tous justes. `coverage.scoreActions.complete` ne change sur aucun des 67 films ; `scoreboardBehind` passe de 112 à 144.
Documents identiques à un fil et à quatre fils (sha-256 du JSON) sur 5 films (64e8adfa, e5adf7b2, bcb6d393, 93fcc545,
1c4c63c2). `go vet ./...` avec et sans `-tags=research` : propre ; `gofmt -l` : vide.

Rouge à la suite du changement : voir `fragments.md` (porte de revision de `objectives`, second oracle). Fragment du pilote : montée
de `objectives.Rev`, chronique, golden, écarts d'équivalence attribués (11 films sur 20), fixtures de contrat (6 films).

## A.6 : la table des réponses de jeu entre dans une révision (FAIT, vérifié)

Cause : `reponsesdejeu` n'est importé que par `replay`, hors de la fermeture des imports de toute couche révisée. Le calque
`scoreActions` est attribué à `grammar.Rev` (`replay/layers.go`).
Code : `revision.Couche.Associes` (paquets hors couche dont les octets, fichiers embarqués compris, entrent dans le périmètre) ;
`CouchesRevisees()` déclare `reponsesdejeu` pour `grammar` ; le second oracle (`equivalence_test.go`) redéclare la même chose.
Tests : `revision/perimetre_associes_test.go` (4 tests : une ligne de plus change l'empreinte, le CRLF non, une déclaration fausse
échoue, dans le module réel seule `grammar` contient la table). Rouge attendu, vérifié : `revision` (golden grammar, périmètre
grammar), `internal/grammar` `TestGrammarRevSuitLaGrammaire`. Montée de révision et goldens : au pilote (`fragments.md`).

## A.4 : Assaut, points de bombe (VERDICT, vérifié sur 8 films, aucun code)

Films : 3 One Bomb (9f57c612, c75f33b8, df8fcbef) et 5 Neutral Bomb (69b16f5d, 3d58eb37, ce083875, 35b75a31, 34bb3bc8). Les
écarts entre le score de l'API et la somme des actions publiées existent sur les 8 (le tableau du film, lui, égale l'API pour
tous sauf quelques joueurs absents). Le film ne déclare aucun script propre à l'Assaut : ses scripts sont ceux du Slayer
(`slayerevents.lua`) et des scripts de graphe Forge (déduit : leurs noms) ; la table n'a pas de groupe d'Assaut et 3 079 messages sur 4 010 (9f57c612) sont « hors arbre ».
Ce que le film nomme, point par point (écart tableau moins actions, rapproché des messages du joueur à ±2,5 images) :

| Points | Compteur | Message | Verdict |
|---|---|---|---|
| +200 (détonation) | `bomb_detonations` (composant 0 A) monte dans la même émission : **22 sur 22**, 8 films | rangs 506 + 516 (7 fois), 335 (5), absents de la table | **nommé par le compteur** ; pas de nom d'action |
| +10 (par image, séries de 3 à 7) | aucun composant ne change avec lui | rang 349, sans texte, **80 sur 85** ; absent de la table | message stable mais **non nommé** |
| +25, +50 en plus d'une aide ou d'une élimination | aucun | rangs 50, 51, 75 (9 sur 14) ; autres sans motif net | non établi |
| +250, +150, +60, +75 | aucun | aucun motif | non établi |

Décision : pas de publication. Le montant de la détonation se lit dans le film (compteur et saut du score personnel dans la
même émission), mais aucune source ne nomme l'action ; en publier une exigerait d'inventer un nom, et le film resterait « incomplet »
(les +10 et les bonus restent sans nom). Le film `34bb3bc8` (Neutral Bomb Squad) a en outre des points publiés en trop
(-2 400 à +450 d'écart) : son groupe de scripts est mal résolu, autre sujet. Piste suivante, si le pilote veut publier : décider le nom
de l'action de détonation, ou lire l'arbre du graphe Forge dans les fichiers du jeu.

## A.3 : arbre décalé sur HI_1_10_0, Grandes batailles à drapeau (VERDICT, vérifié, aucun code)

**Mesure refaite à la tête** (9 Grandes batailles Slayer de HI_1_10_0 ; `replay-build` sans puis avec les décalages de
HI_1_11_0 appliqués à HI_1_10_0, expérience sur copie, non retenue) :
- les trames arrêtées ne disparaissent pas : 154 à 646 par film, 2 990 au total, sur une soixantaine de genres différents (et non
  plus sur les quatre genres lus par A.2a). L'égalité n'est donc **toujours pas testable** : la somme des actions égale l'API pour
  33 joueurs sur 232 sans décalage, 38 avec ;
- avec le décalage, les rangs non confirmés passent de 216 à 0 (4 films concernés) : les ancres de texte sont identiques à celles de HI_1_11_0 ;
- lectures directes du tableau aux rangs 179 à 263, sur les 5 films dont le calage message-action tient (écart sous 33 ms) : 14 lectures
  non nulles concordantes avec la table décalée, 0 contraire (31 et 0 au rapport B.3 sur les 9 films).
Le décalage n'est donc pas **prouvé par l'égalité** ; il reste « lu, non prouvé », avec plus de pièces qu'avant. Non codé.

**Écarté** : l'hypothèse que les quatre genres de A.2a suffisaient à rendre l'égalité testable sur HI_1_10_0.
**Piste suivante** : la vue A de HI_1_10_0 (table des genres de cette build ; le décalage d'un cran de l'arbre des événements et les
genres illisibles ont peut-être la même cause), à lever avant de rouvrir le décalage.

**34 Grandes batailles à drapeau** (`btbflagevents.lua`, 55 films retrouvés dans les relevés de B.3) : aucun groupe de la table ne
porte leur arbre. Meilleur accord des couples (rang, mot texte) avec les 11 groupes : 13,5 % (BTB Slayer, sans décalage), 10,2 %
(Firefight, décalage -4), le reste sous 1 % ; décalages de -8 à +8 essayés. Aucun décalage ni groupe voisin ne se prouve : verdict,
la table n'a pas ce groupe et le film ne le nomme pas. Il faudrait la source du groupe (fichiers du jeu).

## Hors périmètre vu en passant (non traité)

- 4 `TestContractFixtures*` de `replay` sont rouges à la base (fixtures de schéma 127 absentes de `LevelUp-wt-dec160`), identiques
  avec la source d'avant.
- 9f57c612 : des trois événements de détonation publiés (`objectives[]`), celui de l'image 10 234 porte le xuid d'un autre
  joueur que celui dont le score personnel monte de 200 à cet instant (les deux autres concordent). Vérifié sur le document ;
  cause non cherchée (pont d'identité des sièges par manche, déduit) : à reprendre avec A.4 si la détonation se publie.

## Pièces (hors dépôt)

`pistes/bl-score/` : `fragments.md`, `avancement.md`, `cmp.py` (accord par film), `cmp2.py` (joueurs dont le total change),
`diffdocs.py`, `a3lect.py`, `a4corr.py`, `sondes/` (sondes jetables retirées du worktree), `docs_*` (documents), `scan_all.txt`
(1 595 films), `final_cand*.txt`, `fils.out`.
