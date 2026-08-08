# RECHERCHE — OÙ VONT LES TIRS PERDUS DU CTF

> Session de recherche du 2026-08-08, branche `research/v75-ctf`. Répond à la **décision #2**
> du master plan (`../PLAN_MASTER_FILM_KILLFEED_REJEU.md` §8 et §J4.2), qui conditionne le lot
> « rejeu 2D public » (piste F). Aucun code de production n'a été modifié : la mesure passe par
> deux fichiers de test sous garde d'environnement,
> `apps/go-api/internal/analysis/replay/ctf_research_test.go` et `ctf_bridge_research_test.go`.

## 0. VERDICT EN UNE PAGE

**Le facteur ~13x n'existe pas.** Il compare deux comptes ABSOLUS sur des films de tailles
différentes : 564 rejets sur 2 879 événements (CTF) contre 44 sur 519 (Slayer). En TAUX c'est
19,6 % contre 8,5 %, soit **2,3x** — et cet écart ne survit pas au contrôle apparié par carte.

**La cause est identifiée, chiffrée, reproductible : 63 à 90 % des tirs perdus tombent pendant
une VIE NON NOMMÉE.** Le pont slot→joueur nomme chaque vie par **la mort qui la termine**
(`lives.go`). Une vie que nulle mort ne termine — celle qui court de la dernière mort d'un joueur
jusqu'au coup de sifflet final — n'a pas de nom, et tout ce qu'elle porte est rejeté « slot
introuvable ». Le rejet est un **angle mort de la méthode de nommage**, pas une propriété du CTF.

**Ce n'est pas le mode qui décide.** Le meilleur des trois CTF mesurés (88,5 %) fait **mieux que
le KOTH** (86,4 %) et se tient à 0,5 point d'un Team Slayer joué sur la même carte (89,0 %). Sur
861 des 951 films du cache (ceux qui portent un fil de morts exploitable), la part de temps-joueur
non couverte vaut 9,3 % en médiane en CTF contre 7,0 % en Slayer — **1,3x**, avec des
distributions qui se recouvrent (p10 identique à 3,5 %).

**Quatre hypothèses sont RÉFUTÉES par la mesure** : la réplication clairsemée, la dérive
d'horloge, les familles d'armes non mappées, les flux propres au mode. Détail au §5.

**LE CORRECTIF N'EST PAS UNE HYPOTHÈSE : IL EST ÉCRIT ET MESURÉ.** Deux **fermetures**
déterministes — pas des votes : elles n'attribuent que lorsqu'un seul candidat reste possible —
portent **les sept films de 79,7-93,4 % à 88,7-96,4 %**, avec **+12,3 points sur le pire film**.
Leurs deux garde-fous ont refusé un tiers des déductions, ce qui est la raison de leur croire.
Détail et échec de réglage compris : §7.5.

**ATTENTION AU DÉNOMINATEUR (§3bis)** : tous ces taux portent sur les tirs **que le film contient**,
pas sur les tirs du match. Le film n'en porte que 69 à 87 % en arène (23 % en Fiesta), si bien que
la part des tirs RÉELS posés sur la carte vaut **61 à 83 %**, pas 90 %. Cette autre moitié du
sujet est le chantier de la piste E, pas celui-ci.

**Recommandation (§7)** : le rejeu public n'est pas livrable **avec le code d'aujourd'hui**, mais
le blocage n'est plus une inconnue — c'est **un lot d'implémentation borné**, sans aucune
rétro-ingénierie nouvelle. Exécuter ce lot dans v7.5 ou plus tard est une décision utilisateur.

---

## 1. CE QUE DIT LE CONSTAT D'ORIGINE, ET CE QU'IL NE DIT PAS

Le master plan pose : « `64e8adfa` perd 564 tirs *slot introuvable* là où le Slayer en perd 44 ».
Trois corrections de lecture, avant toute hypothèse :

1. **Les dénominateurs diffèrent d'un facteur 5,5** : 2 879 événements contre 519. Le rapport des
   taux est de 2,3x, pas de 13x.
2. **Le témoin Slayer est un cas particulier de rareté.** `000d5950` porte 519 records de tir pour
   2 228 tirs et 595 touches en base, soit **0,23 record par tir** ; `64e8adfa` en porte 2 879
   pour 3 585 tirs, soit **0,80**. Le témoin est quatre fois moins dense que le film qu'on lui
   oppose : 44 rejets n'ont pas le pouvoir statistique qu'on leur prête.
3. **`01e1f945` n'est pas un Slayer.** Le plan de fiabilisation le classe « Catalyst, Slayer » ;
   `match_registry` dit `Arena:King of the Hill on Catalyst`. Le tableau des trois films opposait
   en réalité un Fiesta Slayer, un KOTH et un CTF — pas deux Slayer et un CTF.
   **À corriger dans `../PLAN_FINALISATION_REJEU_2D.md` §1.4.**

## 2. LA MÉTHODE

Deux instruments **sous garde d'environnement** (ils se sautent partout ailleurs, CI comprise —
vérifié : `--- SKIP` sur les deux, paquet vert en 42,9 s). Ils rejouent l'enchaînement de
`BuildFromFilm` et comptent à côté de lui, sans rien changer au décodeur ni à l'assemblage.

| instrument | ce qu'il produit |
|---|---|
| `ctf_research_test.go` (`CTFLostShots`) | par film : taux, ventilation des rejets en sous-causes, histogramme de l'écart au plus proche échantillon du tireur, densité de réplication, répartition par décile / par joueur / par arme |
| `ctf_bridge_research_test.go` (`CTFBridge`) | anatomie du pont : chaque vie non nommée avec ses bornes, sensibilité à la fenêtre d'appariement, sonde de dérive d'horloge, résidu d'appariement par décile |

La sous-cause du rejet est tranchée par préséance :

```
joueur hors pont    aucun slot du film n'est rattaché à cet index de joueur
vie non nommee      une vie SANS identité couvre l'instant du tir
trou de position    le joueur est au pont, mais aucun échantillon n'est à moins de 120 ms
```

**Corpus** : trois CTF, trois Slayer, un KOTH, tous sur des cartes du catalogue de bornes, tous en
arène 8 joueurs (les films BTB sont écartés : 11 min de décodage, anomalie déjà consignée au
master plan). Le contrôle est **apparié par carte** — Catalyst CTF contre Catalyst Team Slayer,
Aquarius CTF contre Aquarius Team Slayer — sans quoi on mesurerait la taille de la carte en
croyant mesurer le mode. Lectures du corpus film en **lecture seule** depuis le dépôt principal.

## 3. LE TABLEAU CENTRAL

Trié par taux de rattachement décroissant. Le seuil du garde local est **85 %**.

| film | mode | carte | durée | tirs dispo. | rattachés | **taux** | sans slot | dont vie non nommée | dont trou de position | ambigus |
|---|---|---|---|---|---|---|---|---|---|---|
| `0edb8512` | Team Slayer | Aquarius | 538 s | 2 808 | 2 623 | **93,4 %** | 164 | 139 (85 %) | 25 | 21 |
| `000d5950` | Fiesta Slayer | Cliffhanger | 498 s | 519 | 475 | **91,5 %** | 44 | 30 (68 %) | 14 | 0 |
| `9aeca4b3` | Team Slayer | Catalyst | 750 s | 2 760 | 2 457 | **89,0 %** | 303 | 262 (86 %) | 41 | 0 |
| `db7b8c3c` | **CTF** | Aquarius | 671 s | 3 547 | 3 140 | **88,5 %** | 342 | 241 (70 %) | 101 | 65 |
| `01e1f945` | KOTH | Catalyst | 534 s | 2 154 | 1 862 | **86,4 %** | 281 | 178 (63 %) | 103 | 11 |
| `64e8adfa` | **CTF** | Catalyst | 834 s | 2 879 | 2 312 | **80,3 %** | 564 | 509 (90 %) | 55 | 3 |
| `829abef9` | **CTF** | Behemoth | 618 s | 2 614 | 2 084 | **79,7 %** | 431 | 396 (92 %) | 35 | 99 |

Ce que ce tableau dit, et qui n'était pas dit :

- **Les modes se recouvrent.** Un CTF (88,5 %) passe devant un KOTH (86,4 %) et talonne un Team
  Slayer (89,0 %). L'écart CTF↔Slayer n'est pas une frontière, c'est un décalage de distribution.
- **Apparié par carte**, le CTF reste en dessous : Catalyst 80,3 % contre 89,0 % (−8,7 points),
  Aquarius 88,5 % contre 93,4 % (−4,9 points). Il y a donc **un effet de mode réel, de l'ordre de
  5 à 9 points** — mais c'est un décalage, pas un facteur 13.
- **Deux films sur sept passent sous 85 %**, et ce sont deux CTF. Le garde a donc raison de
  refuser, mais pas pour la raison écrite : il refuse une classe de MATCHS (ceux qui finissent
  sans que les joueurs meurent), pas une classe de MODES.

## 3bis. CE QUE LE TAUX MESURE — ET CE QU'IL NE MESURE PAS

**Question posée par l'utilisateur le 2026-08-08, et elle est la bonne** : « 90 %, ça veut dire
qu'on a 90 % des tirs du match ? » **Non.** Le dénominateur de tout ce document est le nombre
d'événements de tir **que le film porte**, pas le nombre de tirs du match. `ETAT_DU_POC.md` le
disait déjà — « confondre les deux ferait passer un plafond de format pour un échec de
décodage » — et ce verdict laissait la même ambiguïté. Corrigé ici.

Deux facteurs se multiplient :

| film | mode | ① part des tirs du match présente dans le film | ② taux de placement (§7.5) | **① × ② = tirs réels sur la carte** |
|---|---|---|---|---|
| `0edb8512` | Team Slayer | 86,6 % | 96,4 % | **83,4 %** |
| `db7b8c3c` | **CTF** | 86,0 % | 94,5 % | **81,2 %** |
| `64e8adfa` | **CTF** | 80,3 % | 92,6 % | **74,3 %** |
| `9aeca4b3` | Team Slayer | 71,9 % | 95,0 % | **68,3 %** |
| `01e1f945` | KOTH | 74,4 % | 89,7 % | **66,7 %** |
| `829abef9` | **CTF** | 69,3 % | 88,7 % | **61,5 %** |
| `000d5950` | Fiesta Slayer | **23,3 %** | 93,3 % | **21,7 %** |

(① = records de tir décodés / `match_participants.shots_fired` sommé sur les 8 joueurs.)

**Sur un match d'arène, 6 à 8 tirs sur 10 réellement tirés apparaissent sur la carte. Pas 9
sur 10.** Tout ce document, garde local compris, porte sur la colonne ② — c'est la seule que le
pont peut changer.

**La colonne ① est un AUTRE chantier, et il est déjà ouvert** : la piste E (worktree
`research/v75-precision`, `VERDICT_PRECISION_PROJECTILES.md`) mesure le même phénomène par un
instrument indépendant — `records / tirs API` à **0,92 en Tactical contre 0,31 en Fiesta**. Deux
sessions, deux méthodes de sélection différentes, même constat : la complétude du flux de tirs
dépend fortement de la playlist. Savoir si c'est le format qui n'écrit pas tout ou notre balayage
qui ne trouve pas tout **n'est pas tranché**, et ce n'est pas l'objet de cette session.

**Conséquence pour le garde local** : un plancher exprimé en ② (proposition §7.3 : 88 %) dit « on
place presque tout ce que le film porte ». Il ne dit RIEN sur ①. Si l'écran doit un jour annoncer
une exhaustivité à l'utilisateur, c'est ① × ② qu'il faut publier — et le bandeau du POC, qui
affiche déjà « tirs placés / lisibles », est **honnête à condition de ne pas lire « lisibles »
comme « tirés »**.

## 4. LA CAUSE, ÉTABLIE

### 4.0 DEUX NUMÉROS À NE PAS CONFONDRE — la confusion a déjà coûté une conclusion

Relevé par l'utilisateur le 2026-08-08, après une première rédaction qui les écrasait en un seul.
Le film porte **deux espaces de numérotation distincts**, et tout raisonnement qui les mélange
est faux :

| numéro | ce qu'il désigne | stable ? | statut |
|---|---|---|---|
| **player-slot / index de joueur** | LE JOUEUR | **OUI, sur tout le match** | **PROUVÉ et déjà utilisé** : chunk_27 `b36` (duo, 2 bits) + `b37` (équipe, 1 bit) = 8 combinaisons, bijection per-match (`film_re/RECAP_STATS_EXPLOITABLES.md`) ; et les 5 bits devant le xuid, mesurés sur **116 films sur 116**, lus par `player_index.go` (43 chunks concordants sur `64e8adfa`, 0 désaccord) |
| **identifiant d'entité biped** | UNE VIE | **NON** | **MESURÉ** : journal du 2026-07-03, « 99 ENTITÉS biped ≠ 99 joueurs (respawns + ragdolls sur le match) ». Cette session : **138 identifiants distincts pour 8 joueurs** sur `64e8adfa`, alloués en ordre croissant, 141 segments de vie |
| **le lien entre les deux** | quel bipède appartient à quel joueur | — | **NON RÉSOLU**, et documenté comme tel : journal du 2026-07-28, la piste i19 réfutée par son propre contrôle (handle constant `0x8000004F`, critère « huit entités distinctes par image-clé » échoué **0/26**) |

**Ce que ce tableau change** : la difficulté n'est PAS d'identifier un joueur — c'est acquis et
robuste. Elle est de savoir **quel corps il occupe à un instant donné**. Le fil des morts sert
aujourd'hui de substitut à ce lien manquant, et c'est ce substitut qui a l'angle mort décrit
ci-dessous.

### 4.1 La ventilation

Sur les sept films, la cause « vie non nommée » pèse **63 à 92 %** des rejets. « Trou de
position » plafonne à 4,8 % des tirs disponibles (KOTH) et vaut 0,9 à 2,9 % ailleurs. Aucun film
n'a un seul rejet « joueur hors pont » : les huit index de joueur sont toujours au pont.

### 4.2 La signature temporelle ne laisse pas de place au doute

Taux de rejet par décile de film :

| film | D1 | D2 | D3 | D4 | D5 | D6 | D7 | D8 | D9 | **D10** |
|---|---|---|---|---|---|---|---|---|---|---|
| `0edb8512` | 0 % | 1 % | 0 % | 4 % | 0 % | 0 % | 1 % | 1 % | 6 % | **42 %** |
| `000d5950` | 0 % | 0 % | 6 % | 6 % | 0 % | 3 % | 3 % | 6 % | 2 % | **40 %** |
| `9aeca4b3` | 0 % | 2 % | 8 % | 15 % | 4 % | 7 % | 0 % | 8 % | 13 % | **46 %** |
| `db7b8c3c` | 20 % | 0 % | 0 % | 10 % | 2 % | 1 % | 0 % | 0 % | 2 % | **52 %** |
| `01e1f945` | 28 % | 43 % | 19 % | 1 % | 0 % | 3 % | 0 % | 11 % | 1 % | **48 %** |
| `64e8adfa` | 1 % | 0 % | 0 % | 0 % | 2 % | 12 % | 18 % | 37 % | 57 % | **62 %** |
| `829abef9` | 4 % | 27 % | 20 % | 3 % | 16 % | 6 % | 0 % | 1 % | 23 % | **74 %** |

**Le dernier décile porte le plus gros rejet dans les SEPT films, tous modes confondus** (40 % à
74 %). C'est la signature des dernières vies : elles ne se terminent par aucune mort.

Le KOTH ajoute un **U** — 28 %, 43 %, 19 % en tête de film. Ce n'est pas la même cause, et le §4.4
la nomme.

### 4.3 L'anatomie du pont confirme le mécanisme, vie par vie

`64e8adfa` (CTF, 80,3 %) porte **19 vies non nommées**, dont deux gigantesques :

```
vie  slot 602   530,9 s -> 748,8 s   duree 218,0 s
vie  slot 607   554,2 s -> 749,7 s   duree 195,5 s
```

Deux joueurs cessent de mourir à 531 s et 554 s d'un film de 834 s. Ils redeviennent
**anonymes pour le dernier quart du match** — et ce sont exactement les deux index de joueur qui
portent les plus forts taux de perte du film (42,6 % et 45,3 %, contre 0 % pour l'index qui est
mort 12 fois).

`0edb8512` (Team Slayer, 93,4 % — le meilleur film) porte **2 vies non nommées**, et rien d'autre.

La relation dose-effet se lit aussi joueur par joueur : le taux de perte suit **inversement le
nombre de morts**. Sur le KOTH, l'index à 7 slots perd 23,8 % et l'index à 17 slots perd 0 %.

### 4.4 La fenêtre d'appariement de 150 ms est une cause SECONDAIRE, réelle et mesurée

`deathMatchWindowMS` vaut 150 ms. En rejouant l'appariement avec des fenêtres plus larges (copie
de recherche, la constante du code reste intouchée) :

| film | 150 ms | 500 ms | 2 000 ms | 10 000 ms | vies |
|---|---|---|---|---|---|
| `64e8adfa` | 122 | 123 | 123 | 123 | 141 |
| `0edb8512` | 93 | 93 | 93 | 93 | 95 |
| `db7b8c3c` | 139 | 139 | 139 | 139 | 150 |
| **`01e1f945`** | **97** | **104** | **105** | 105 | 113 |

Sur le KOTH, élargir à 500 ms nomme **sept vies de plus** — et c'est précisément le film dont la
signature temporelle porte un U inexpliqué par les dernières vies. Ailleurs le gain est nul ou
d'une vie. La fenêtre n'est donc pas la contrainte générale, mais elle mord sur certains films.

### 4.5 Le mécanisme, en une phrase

**Le pont nomme une vie par la mort qui la termine ; un joueur qui cesse de mourir cesse d'être
localisable.** Les octets sont là, les positions sont là (16,7 ms de pas médian), l'identité du
tireur est écrite dans l'événement de tir. Il ne manque que **le nom du slot**.

Le CTF n'est pas une cause : c'est un **amplificateur**, et pas par la durée. Sur les 861 films
exploitables du cache, la durée médiane d'un CTF (510 s) est même **inférieure** à celle d'un
Slayer (525 s) : ce qui change est la part du temps-joueur postérieure à la dernière mort — 9,3 %
contre 7,0 % en médiane. À durée égale, le CTF s'achève plus souvent sur une phase où les joueurs
survivent.

Le master plan supposait l'inverse — « un mode où l'on meurt davantage produit plus de vies
courtes, donc plus de traces non publiées ». **C'est le contraire qui est vrai** : c'est le joueur
qui meurt PEU qui coûte des tirs, et le film incriminé (`64e8adfa`, 834 s) est un long match serré
(2-3) où deux joueurs ont cessé de mourir au dernier quart.

### 4.6 Le plafond, chiffré

Si toute vie était nommée, il ne resterait que « trou de position » et « ambigu » :

| film | taux actuel | **plafond** |
|---|---|---|
| `9aeca4b3` | 89,0 % | **98,5 %** |
| `0edb8512` | 93,4 % | **98,4 %** |
| `64e8adfa` | 80,3 % | **98,0 %** |
| `000d5950` | 91,5 % | **97,3 %** |
| `db7b8c3c` | 88,5 % | **95,3 %** |
| `829abef9` | 79,7 % | **94,9 %** |
| `01e1f945` | 86,4 % | **94,7 %** |

**Les sept films passeraient le plancher de 85 % avec une marge de 10 à 13 points.** C'est la
taille du gain, et elle justifie qu'on cherche le nom du slot ailleurs que dans le fil des morts.

## 5. CE QUI EST RÉFUTÉ

| hypothèse du plan | verdict | preuve chiffrée |
|---|---|---|
| **Réplication clairsemée** (les positions manquent aux instants de tir) | **RÉFUTÉE** | pas médian entre échantillons du même slot : **16,7 ms** dans les 7 films, p90 entre 17,1 et 17,5 ms ; part des pas au-delà de la tolérance de 120 ms : **0,07 % à 0,29 %**. Sur les rejets où le joueur EST au pont, l'écart au plus proche échantillon vaut plus de 15 s dans 89 % des cas (`64e8adfa` : 502/561) — ce n'est pas un trou de réplication, c'est une autre vie |
| **Familles d'armes non mappées en CTF** | **RÉFUTÉE** | le rejet suit l'exposition : sur `64e8adfa` l'arme la plus tirée (1 897 records) perd 22,5 %, la deuxième (825) 11,3 % ; sur `db7b8c3c` les deux premières perdent 8,6 % et 8,3 %. Aucune famille ne concentre la perte. *Réserve, et elle est solide* : `0xC7D5091200000000` sur `829abef9` perd **29 records sur 29** dans un film qui en rejette 16,5 % (p ≈ 10⁻²³), famille hors catalogue et variante nulle. La réfutation vaut pour les familles CONNUES ; cet identifiant-là est un objet à part, classé H2 (§6) |
| **Chunks / flux propres au mode** | **RÉFUTÉE** | la ventilation des causes est la MÊME dans les trois modes ; seule l'amplitude change. Le dernier décile porte le plus gros rejet dans les 7 films. Aucun film n'a de rejet « joueur hors pont » |
| **Fenêtres temporelles (retours de drapeau)** | **RÉFUTÉE comme cause dominante** | le rejet n'est pas épisodique mais **monotone croissant** sur `64e8adfa` (0 %, 0 %, 0 %, 0 %, 2 %, 12 %, 18 %, 37 %, 57 %, 62 %). Une fenêtre d'événement produirait des pics, pas une rampe |
| **Objets portables absorbant des événements** | **RÉFUTÉE** | le porteur de drapeau reste un biped répliqué : ses positions sont dans le flux. Aucun index de joueur n'est absent du pont, dans aucun film |
| **Dérive d'horloge fil des morts ↔ film** *(hypothèse ajoutée en séance)* | **RÉFUTÉE** | le résidu moyen d'appariement est **plat par décile** : 34,3 / 34,2 / 34,3 / 34,5 / 34,2 / 34,7 / 28,7 / 34,0 / 34,6 / 33,9 ms sur `64e8adfa`, 35,7→35,0 ms sur `0edb8512`. La sonde « offset de la seconde moitié » diverge, mais sur 10 appariements seulement : c'est un ajustement dégénéré, pas une dérive |

## 6. HYPOTHÈSES RESTANTES, CLASSÉES

| # | hypothèse | statut | ce qu'il faudrait pour trancher | poids estimé |
|---|---|---|---|---|
| H1 | **Un événement de match renouvelle les bipeds sans mort.** Sur `64e8adfa`, sept vies non nommées se terminent entre 744,3 s et 749,7 s, et cinq nouvelles commencent entre 751,5 s et 753,9 s. La base ne porte que 2 morts entre 720 et 750 s : ces fins de vie ne sont expliquées par aucune mort. Candidat naturel en CTF : la réinitialisation qui suit une capture | **OUVERT, un seul film** | corréler les fins de vie non appariées avec les événements `mode` du film sur les 129 CTF du cache ; un décompte de vagues égal au nombre de captures confirmerait | fort sur ce film (7 vies dont les 2 plus longues) |
| H2 | **Une source de dégât inconnue du catalogue.** `0xC7D5091200000000` sur `829abef9` : famille `0xC7D50912` absente des **42** entrées de `weapon_labels`, et variante (32 bits bas) **nulle** là où 36 des 42 armes portent `0x42C9679F`. Recherché dans tout le corpus de rétro-ingénierie : **aucune occurrence** | **OUVERT, et ce n'est pas le hasard** | **29 rejets sur 29** dans un film qui rejette 16,5 % : p ≈ 10⁻²³. La concentration est réelle, sa cause non établie. À instruire : ces 29 records sont-ils groupés dans le temps (un joueur, une vie non nommée) ou leur index de tireur est-il systématiquement inexploitable ? | 6,7 % des pertes d'un film |
| H3 | **Le rejet « ambigu » est mal borné.** 99 sur `829abef9` et 65 sur `db7b8c3c`, contre 0 sur deux films. Deux vies du même joueur se recouvrent dans la fenêtre de 120 ms | **OUVERT** | mesurer le recouvrement réel des vies concernées ; c'est un chantier de découpage des vies, distinct du nommage | 2 à 4 % sur deux films |
| H4 | **La fenêtre de 150 ms** (§4.4) | **MESURÉ, non tranché** | vérifier qu'élargir à 500 ms ne crée aucun appariement faux (le tri glouton par écart croissant le rend improbable, ce n'est pas une preuve) | +7 vies sur un film sur quatre |

**Ce qui N'EST PAS une hypothèse restante** : la cause dominante. Elle est établie, et elle
n'appelle pas d'autre mesure — elle appelle une décision.

## 7. RECOMMANDATION

### 7.1 EN L'ÉTAT DU CODE : NON. APRÈS UN LOT BORNÉ ET DÉJÀ MESURÉ : OUI.

**La réponse a changé en cours de session**, et il faut le dire dans cet ordre :

- **Le code d'aujourd'hui ne peut pas être ouvert au public.** Deux films sur sept passent sous
  le plancher de 85 %, et le mécanisme qui les y fait passer frappe **tous les modes** dans la
  phase la plus regardée : le dernier décile perd 40 à 74 % de ses tirs dans les sept films.
  Livrer ça, ce serait un écran qui tait la moitié d'une fin de match sans le dire.
- **Mais le correctif n'est plus une hypothèse : il est écrit et mesuré** (§7.5). Les deux
  fermetures portent **les sept films à 88,7 % ou mieux**, avec +12,3 points sur le pire. Le
  blocage n'est plus « on ne sait pas pourquoi » ; c'est **un lot d'implémentation borné**.

**Ce qui reste à la décision de l'utilisateur** : exécuter ce lot dans v7.5 et ouvrir le rejeu, ou
le garder en local et livrer le lot plus tard. Les deux sont défendables ; ce document ne tranche
pas ce point, il en supprime l'inconnue.

### 7.2 LE LOT, DANS L'ORDRE — aucune rétro-ingénierie nouvelle

1. **Les deux fermetures** (§7.5), portées des instruments de recherche vers `owners.go`, avec
   leurs deux garde-fous (contestation, recouvrement) et leurs compteurs **publiés dans
   `coverage`** — la règle du chantier veut qu'on dise ce qu'on perd, elle vaut aussi pour ce
   qu'on déduit. Gain mesuré : +1,7 à +12,3 points. La fenêtre de réapparition se calibre **sur
   le film traité**, jamais sur une constante importée.
2. **Élargir la fenêtre d'appariement des morts à 500 ms** (§4.4) : +7 vies sur le KOTH, 0 ou +1
   ailleurs. Le moins cher des trois, et il n'a pas encore été composé avec les fermetures — le
   gain cumulé est donc probablement **supérieur** aux chiffres du §7.5.
3. **Instruire H1** (§6) : la vague de renouvellement des corps à ~747 s sur `64e8adfa`. Si la
   réinitialisation d'après-capture est confirmée, elle se lit dans les événements `mode` **déjà
   en base**.

Ce que ce lot ne fait PAS : résoudre `joueur → biped` (§4.0). Ce lien reste non résolu, et les
fermetures sont précisément ce qui permet de s'en passer pour l'essentiel du gain.

### 7.5 LES DEUX FERMETURES — mesurées, pas proposées

Le §7.2 ne se contente pas de recommander : les deux fermetures ont été **implémentées en
instrument de recherche et mesurées sur les sept films** (`ctf_closure_research_test.go`).

**Ce qui les sépare du repli voté retiré le 2026-07-28**, et c'est la seule chose qui compte :

```
le vote        plusieurs candidats, on garde le mieux placé        -> un CHOIX
la fermeture   un seul candidat POSSIBLE, les autres sont exclus   -> une DÉDUCTION
```

Dès que deux candidats subsistent, **rien n'est attribué**.

| | fermeture A — par le corps disponible | fermeture B — par la réapparition |
|---|---|---|
| le raisonnement | un joueur tire alors qu'aucune de ses vies nommées ne le couvre : son corps est l'une des vies anonymes vivantes à cet instant. S'il n'y en a **qu'une**, c'est elle | une vie commence **une réapparition après la mort qui l'a causée**, et le fil des morts nomme cette victime. Si **une seule** mort tombe dans la fenêtre, la vie est la sienne |
| ce qu'elle lit | les tirs (qui portent déjà leur auteur) + les vies | le fil des morts + les vies |
| réglage | aucun | fenêtre de réapparition **calibrée sur le film lui-même** (centiles de l'écart début-de-vie ↔ mort précédente, mesurés sur les vies DÉJÀ nommées) — un réglage importé d'un autre film serait une supposition |

**Deux garde-fous, posés AVANT la mesure, et qui peuvent la réfuter** :

1. **contestée** — deux joueurs revendiquent le même corps : aucune attribution.
2. **recouvrement** — un joueur n'a qu'un corps. Si le corps déduit chevauche dans le temps une
   vie déjà nommée du même joueur, l'attribution est **impossible** et elle est rejetée. C'est le
   pendant du critère « huit entités distinctes » qui a réfuté la piste i19 le 2026-07-28.

Un taux de rejet élevé dirait que la fermeture attrape autre chose que des bipèdes de joueur.
Les compteurs sont publiés ci-dessous **précisément pour qu'on puisse en juger**.

#### Le réglage de B, et l'échec qui l'a corrigé

La calibration livre un fait qui n'était pas cherché : **la réapparition est DÉTERMINISTE, et
c'est une constante DU MATCH, pas du jeu.**

| films | délai mesuré (centile 5 → médiane) |
|---|---|
| `000d5950`, `0edb8512`, `9aeca4b3` | **8 090 → 8 092 ms** · 8 092 → 8 294 · 8 149 → 8 166 |
| `01e1f945`, `64e8adfa`, `829abef9`, `db7b8c3c` | **10 176 → 10 179 ms** · 10 089 → 10 188 · 10 112 → 10 118 · 10 103 → 10 306 |

Deux millisecondes d'écart entre le centile 5 et la médiane : ce n'est pas une distribution,
c'est une constante. Mesurée sur les vies **déjà nommées**, donc sans rien supposer.

**Mon premier réglage était faux, et il a échoué exactement là où il fallait réussir.** J'avais
pris `[p05, p95]` par prudence. Or le centile 95 monte à **51 717 ms** et **67 688 ms** sur les
deux films CTF — ce sont les vies dont la mort précédente du même joueur n'est PAS celle qui les
a fait réapparaître (premières vies). Fenêtre trop large ⇒ plusieurs morts dans la fenêtre ⇒
**13 vies sur 13 contestées** sur `64e8adfa`, gain **nul**. Corrigé par une fenêtre serrée à
**médiane ± 750 ms** (vingt fois plus étroite que l'intervalle entre deux morts, et vingt fois
plus large que la dispersion réelle). Les deux réglages sont publiés ci-dessous : l'échec fait
partie du résultat.

#### Résultats — les sept films

| film | mode | avant | après A | **après A+B** | **gain** |
|---|---|---|---|---|---|
| `0edb8512` | Team Slayer | 93,4 % | 95,2 % | **96,4 %** | +2,96 |
| `9aeca4b3` | Team Slayer | 89,0 % | 89,7 % | **95,0 %** | +5,98 |
| `db7b8c3c` | **CTF** | 88,5 % | 90,4 % | **94,5 %** | +5,92 |
| `000d5950` | Fiesta Slayer | 91,5 % | 91,5 % | **93,3 %** | +1,73 |
| `64e8adfa` | **CTF** | 80,3 % | 88,8 % | **92,6 %** | **+12,26** |
| `01e1f945` | KOTH | 86,4 % | 88,5 % | **89,7 %** | +3,25 |
| `829abef9` | **CTF** | 79,7 % | 83,2 % | **88,7 %** | +8,95 |

**LES SEPT FILMS PASSENT AU-DESSUS DE 88,7 %**, contre deux sous 85 % avant. Le gain est le plus
fort exactement là où le garde refusait : +12,3 points sur le pire film, +9,0 sur le second.

Comparaison des deux fenêtres de B : la serrée gagne sur trois films (+3,5 à +5,5 points), fait
jeu égal sur trois, et **perd 0,75 point sur le KOTH**. Elle est donc recommandée, sans être
uniformément meilleure — et c'est dit.

#### Les garde-fous ont mordu, et c'est ce qui rend le chiffre crédible

Sur les sept films, fermeture serrée : **33 vies attribuées, 17 refusées** — 10 contestées (deux
joueurs revendiquent le même corps) et 7 rejetées par le contrôle de recouvrement (un joueur n'a
qu'un corps). **Un tiers des déductions sont refusées par leurs propres contrôles.**

C'est le résultat le plus important de cette section : la méthode se censure elle-même. Un
contrôle qui ne rejette jamais rien ne prouve rien ; celui-ci rejette, donc quand il laisse
passer, il a été mis à l'épreuve.

#### Ce qui reste après les deux fermetures

De 3,6 % (`0edb8512`) à 11,3 % (`829abef9`) des tirs restent non placés. Le résidu est fait
des vies encore anonymes que ni A ni B ne peuvent trancher, et — sur `829abef9` — des **99 rejets
« ambigu » (3,8 %)** qui deviennent la première cause de ce film et relèvent d'un autre chantier
(H3, découpage des vies). Le plafond du §4.6 (94,7-98,5 %) n'est donc pas atteint, mais l'essentiel
de l'écart l'est.

### 7.3 Et le garde local, alors ?

**Le garde reste en place**, et son critère doit être **réécrit** — ce que le plan de
fiabilisation demandait déjà. Ce que la mesure permet maintenant d'écrire, et qui manquait :

- un **plancher sur TOUS les films mesurés**, pas sur « au moins deux » (un critère qu'on
  satisfait en choisissant ses films ne protège de rien — leçon déjà tirée le 2026-07-31) ;
- un **corpus nommé** : les sept films de ce document, modes et cartes explicites, à rejouer à
  chaque changement du pont ;
- une **date de réexamen**, sans quoi le garde devient le « compatibility guard forever » que le
  CLAUDE.md interdit.

Proposition à soumettre, **chiffrée sur le corpus réel et non sur un plafond théorique** :

| plancher | code d'aujourd'hui | après le lot §7.2 (mesuré) |
|---|---|---|
| 85 % | 5 films sur 7 | **7 sur 7** |
| **88 %** | 4 sur 7 | **7 sur 7** |
| 90 % | 2 sur 7 | 5 sur 7 |
| 95 % | 0 sur 7 | 2 sur 7 |

**Le plancher à retenir est donc 88 %** : c'est le plus exigeant que le lot §7.2 franchit sur
TOUS les films mesurés, sans exception à négocier. Avec, dans le même critère : verdict du pont
nominal sur tous, **corpus nommé** (les sept films de ce document, modes et cartes explicites,
rejoués à chaque changement du pont), et **date de réexamen**.

Poser 90 % obligerait à excepter deux films, donc à retomber dans le défaut de 2026-07-31 — un
critère qu'on satisfait en choisissant ses films. Poser 85 % laisserait passer le code actuel sur
5 films, ce qu'on vient précisément de juger insuffisant.

### 7.4 Un prédicteur sans décodage, à connaître

La part de temps-joueur postérieure à la dernière mort se calcule **en base, sans toucher au
film** (`highlight_events` + `match_registry`). Elle vaut 3,3 % sur le meilleur film et 8,6 % sur
le pire des CTF, et à l'échelle du cache : médiane 9,3 % en CTF, 7,0 % en Slayer, 9,9 % en
Strongholds, 4,1 % en Oddball.

**Honnêteté sur ce prédicteur** : il ordonne mal. J'avais enregistré, avant de voir les
résultats, la prédiction que `9aeca4b3` (13,9 % de temps non couvert) perdrait plus que
`64e8adfa` (8,6 %). **Il perd moins** (11,0 % contre 19,6 %). Le temps non couvert ne suffit donc
pas : ce qui compte est le nombre de TIRS qui y tombent, et une pondération par les tirs de la
base ne corrige pas l'ordre. À utiliser comme signal de tri grossier, jamais comme critère.

---

## §8 — LES TIRS FATALS  *(priorité utilisateur, posée le 2026-08-08)*

> « Moi en priorité je veux les tirs fatals — quelle couverture on a sur les tirs qui ont causé
> des morts ? » Mesuré par `ctf_fatal_research_test.go` sur les sept films, **717 morts**, croisé
> avec la CLASSE de la source de dégât (`match_kill_events_latest.source_tag` -> table de nommage
> `jpt!` de `damagetag/data/labels.tsv`).

### 8.1 Ce qui est DÉJÀ acquis, en base, sans rien faire

| | matchs | morts | arme du kill | tueur nommé |
|---|---|---|---|---|
| **film en cache** | **951** | **74 909** | **99,55 %** | 98,87 % |
| film absent du cache | 392 | 49 785 | 0 % (aucun film à décoder) | 100 % (fil de l'API) |

**Qui a tué qui, quand, avec quelle arme : résolu et persisté.** Ce qui manque est le **OÙ** — et
le dépôt porte déjà une table `kill_positions` (`killer_x/y/z`, `victim_x/y/z`, `time_ms`)
**VIDE : 0 ligne, 0 match.** Elle a été créée pour cela et jamais alimentée.

### 8.2 Ce que la mesure ajoute — et une mort sur cinq n'est pas un tir

Un tir fatal posé sur la carte exige quatre maillons ; le plus faible commande. Sur 717 morts :

| classe de la source | morts | part | tir placé | aucun tir | non placé | **taux** |
|---|---|---|---|---|---|---|
| **ARME** | 562 | 78,4 % | 397 | 138 | 27 | **70,6 %** |
| MÊLÉE | 82 | 11,4 % | 76 | 2 | 4 | 92,7 % |
| (tag inconnu) | 32 | 4,5 % | 6 | 25 | 1 | 18,8 % |
| GRENADE | 18 | 2,5 % | 11 | 5 | 2 | 61,1 % |
| OBJET EXPLOSIF | 10 | 1,4 % | 2 | 7 | 1 | 20,0 % |
| VÉHICULE | 8 | 1,1 % | 0 | 8 | 0 | 0 % |
| DÉGÂT GLOBAL | 5 | 0,7 % | 0 | 5 | 0 | 0 % |

**21,6 % des morts ne sont pas causées par un tir** et ne peuvent porter aucun record de tir.
Les juger sur ce critère serait une faute de mesure.

Sur les morts **par arme**, film par film : `0edb8512` 87,2 % · `db7b8c3c` 78,2 % · `9aeca4b3`
78,1 % · `829abef9` 76,4 % · `64e8adfa` 72,0 % · `01e1f945` 64,9 % · **`000d5950` 41,4 %** (le
film Fiesta, qui ne porte que 23,3 % des tirs de son match — cf. §3bis).

### 8.3 CE QUI COMMANDE N'EST PLUS LE PONT

Sur les 562 morts par arme, la perte se répartit ainsi :

```
24,6 %   AUCUN record de tir du tueur dans les 1,5 s qui précèdent   -> colonne ① (piste E)
 4,8 %   le tir existe mais le pont ne sait pas le placer            -> colonne ② (ce document)
```

**Après les fermetures du §7.5, le pont ne coûte plus que 4,8 % sur ce qui compte le plus.** Le
reste appartient à la complétude du flux de tirs, c'est-à-dire à un autre chantier.

### 8.4 LA LIMITE DE CETTE MESURE, ET ELLE EST IMPORTANTE

Cet instrument mesure « sait-on placer **UN** tir du tueur dans la fenêtre qui précède la mort »,
**PAS** « sait-on placer **LE COUP QUI A TUÉ** ». La preuve est dans le tableau : les morts à la
**mêlée** affichent 92,7 % — absurde pour un tir fatal. Ce qui est trouvé là, c'est le dernier tir
du tueur avant qu'il ne frappe au corps à corps.

L'imputation du coup mortel n'est pas l'objet de ce document : elle est faite par le chantier
arme-par-kill (`killsource`), et elle est à **99,55 %** (§8.1). Les 70,6 % ci-dessus sont donc une
mesure de **LOCALISATION**, pas d'imputation — et une **borne haute** de ce qu'un fil des
éliminations pourrait poser sur la carte.

### 8.5 CE QUE ÇA CHANGE DANS L'ORDRE DES PRIORITÉS

`kill_positions` devient le meilleur rapport valeur/coût du chantier, et **il ne dépend pas du
rejeu 2D** : le tueur, la victime, l'instant et l'arme existent déjà pour 74 909 morts ; seules
les coordonnées manquent, et le pont sait les produire. Ordre proposé au §7.2, amendé :
`kill_positions` d'abord, les fermetures comme préalable technique, le garde du rejeu ensuite.

---

## ANNEXE — REPRODUIRE

```bash
cd apps/go-api
CGO_ENABLED=0 FILM_CACHE_ROOT=<mainrepo>/data/cache CTF_RESEARCH_OUT=<dir> \
  CTF_RESEARCH_FILMS="64e8adfa:Catalyst,000d5950:Cliffhanger,9aeca4b3:Catalyst,01e1f945:Catalyst,db7b8c3c:Aquarius,0edb8512:Aquarius,829abef9:Behemoth" \
  go test ./internal/analysis/replay/ -run CTFLostShots -timeout 60m

CGO_ENABLED=0 FILM_CACHE_ROOT=<mainrepo>/data/cache CTF_RESEARCH_OUT=<dir> \
  CTF_BRIDGE_FILMS="64e8adfa:Catalyst,0edb8512:Aquarius,01e1f945:Catalyst,db7b8c3c:Aquarius" \
  go test ./internal/analysis/replay/ -run CTFBridge -timeout 60m

CGO_ENABLED=0 FILM_CACHE_ROOT=<mainrepo>/data/cache CTF_RESEARCH_OUT=<dir> \
  CTF_CLOSURE_FILMS="64e8adfa:Catalyst,829abef9:Behemoth,db7b8c3c:Aquarius,9aeca4b3:Catalyst,01e1f945:Catalyst,0edb8512:Aquarius,000d5950:Cliffhanger" \
  go test ./internal/analysis/replay/ -run CTFExclusionClosure -timeout 90m
```

Sans ces variables, les deux tests se sautent (`--- SKIP`, message nommant la variable). Le
corpus film est lu en LECTURE SEULE ; aucune sortie n'est écrite dans `data/`.

**Coût mesuré** : environ 4 à 6 min par film sur ce poste (trois sessions en parallèle), soit
~40 min pour les sept films et ~25 min pour les quatre anatomies.

**Les sorties brutes de cette session sont versionnées** sous
`replay2d/mesures_ctf_2026-08-08/` (26 fichiers, 168 Ko) : tout chiffre de ce document s'y relit
sans re-décoder un film.
