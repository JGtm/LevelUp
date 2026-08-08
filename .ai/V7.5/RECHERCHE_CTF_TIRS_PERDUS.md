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

**La taille du gain est connue** : si toutes les vies étaient nommées, les sept films
monteraient à **94,7–98,5 %** de tirs rattachés, tous très au-dessus du plancher de 85 %.

**Recommandation (§7)** : le rejeu public **n'est pas livrable en v7.5**, mais pour une raison
différente de celle qu'on croyait, et le correctif est identifié, borné, et ne demande **aucune
rétro-ingénierie nouvelle**.

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

## 4. LA CAUSE, ÉTABLIE

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
| **Familles d'armes non mappées en CTF** | **RÉFUTÉE** | le rejet suit l'exposition : sur `64e8adfa` l'arme la plus tirée (1 897 records) perd 22,5 %, la deuxième (825) 11,3 % ; sur `db7b8c3c` les deux premières perdent 8,6 % et 8,3 %. Aucune famille ne concentre la perte. *Réserve* : `0xC7D5091200000000` sur `829abef9` perd **29 records sur 29** et n'est dans aucune des 42 entrées de `weapon_labels` — sa forme diffère (32 bits bas nuls). 6,7 % des pertes de ce film ; classé en hypothèse restante (§6) |
| **Chunks / flux propres au mode** | **RÉFUTÉE** | la ventilation des causes est la MÊME dans les trois modes ; seule l'amplitude change. Le dernier décile porte le plus gros rejet dans les 7 films. Aucun film n'a de rejet « joueur hors pont » |
| **Fenêtres temporelles (retours de drapeau)** | **RÉFUTÉE comme cause dominante** | le rejet n'est pas épisodique mais **monotone croissant** sur `64e8adfa` (0 %, 0 %, 0 %, 0 %, 2 %, 12 %, 18 %, 37 %, 57 %, 62 %). Une fenêtre d'événement produirait des pics, pas une rampe |
| **Objets portables absorbant des événements** | **RÉFUTÉE** | le porteur de drapeau reste un biped répliqué : ses positions sont dans le flux. Aucun index de joueur n'est absent du pont, dans aucun film |
| **Dérive d'horloge fil des morts ↔ film** *(hypothèse ajoutée en séance)* | **RÉFUTÉE** | le résidu moyen d'appariement est **plat par décile** : 34,3 / 34,2 / 34,3 / 34,5 / 34,2 / 34,7 / 28,7 / 34,0 / 34,6 / 33,9 ms sur `64e8adfa`, 35,7→35,0 ms sur `0edb8512`. La sonde « offset de la seconde moitié » diverge, mais sur 10 appariements seulement : c'est un ajustement dégénéré, pas une dérive |

## 6. HYPOTHÈSES RESTANTES, CLASSÉES

| # | hypothèse | statut | ce qu'il faudrait pour trancher | poids estimé |
|---|---|---|---|---|
| H1 | **Un événement de match renouvelle les bipeds sans mort.** Sur `64e8adfa`, sept vies non nommées se terminent entre 744,3 s et 749,7 s, et cinq nouvelles commencent entre 751,5 s et 753,9 s. La base ne porte que 2 morts entre 720 et 750 s : ces fins de vie ne sont expliquées par aucune mort. Candidat naturel en CTF : la réinitialisation qui suit une capture | **OUVERT, un seul film** | corréler les fins de vie non appariées avec les événements `mode` du film sur les 129 CTF du cache ; un décompte de vagues égal au nombre de captures confirmerait | fort sur ce film (7 vies dont les 2 plus longues) |
| H2 | **Une source de dégât qui n'est pas une arme portée** (`0xC7D5091200000000`, 29/29 perdus, hors catalogue) | **OUVERT** | greper l'index arme-par-kill (`../README_KILLWEAPON_INDEX.md`) pour cet identifiant ; vérifier s'il apparaît dans d'autres films | 6,7 % des pertes d'un film, marginal ailleurs |
| H3 | **Le rejet « ambigu » est mal borné.** 99 sur `829abef9` et 65 sur `db7b8c3c`, contre 0 sur deux films. Deux vies du même joueur se recouvrent dans la fenêtre de 120 ms | **OUVERT** | mesurer le recouvrement réel des vies concernées ; c'est un chantier de découpage des vies, distinct du nommage | 2 à 4 % sur deux films |
| H4 | **La fenêtre de 150 ms** (§4.4) | **MESURÉ, non tranché** | vérifier qu'élargir à 500 ms ne crée aucun appariement faux (le tri glouton par écart croissant le rend improbable, ce n'est pas une preuve) | +7 vies sur un film sur quatre |

**Ce qui N'EST PAS une hypothèse restante** : la cause dominante. Elle est établie, et elle
n'appelle pas d'autre mesure — elle appelle une décision.

## 7. RECOMMANDATION

### 7.1 Le rejeu public n'est pas livrable en v7.5

**Non**, et la raison a changé. Ce n'est pas « le CTF est cassé » : c'est que **deux films sur
sept, soit 29 % du corpus mesuré, passent sous le plancher de 85 %**, et que le mécanisme qui les
y fait passer — un joueur qui survit à la fin du match devient anonyme — frappe **tous les
modes**. Le dernier décile perd 40 à 74 % de ses tirs dans les sept films.

Livrer en l'état exposerait un écran qui, dans la phase la plus regardée d'un match (la fin), tait
en silence la moitié de ce que le film porte. C'est exactement ce que le garde a été posé pour
empêcher, et la règle du chantier tient : « je préfère rien afficher que quelque chose de
complètement faux ».

### 7.2 Ce qu'il faut faire à la place, et c'est borné

**Le pont ne doit plus dépendre du seul fil des morts.** Le plafond du §4.6 dit que le reste du
pipeline est sain : 94,7 à 98,5 % une fois les vies nommées. Trois chantiers, par rapport
gain/coût décroissant, **aucun ne demande de rétro-ingénierie nouvelle** :

1. **Nommer la dernière vie de chaque joueur par continuité de slot.** Une vie non nommée qui
   commence là où une vie nommée du même joueur s'arrête n'a pas besoin d'être devinée : la
   chaîne des vies d'un slot est déjà lue. À vérifier sur pièces avant de coder — c'est
   l'hypothèse à instruire en premier, et elle porte l'essentiel des 15 points de gain.
2. **Élargir la fenêtre d'appariement à 500 ms** (§4.4) : +7 vies sur le KOTH, 0 ou +1 ailleurs,
   zéro régression observée sur quatre films. Le moins cher des trois.
3. **Instruire H1** (§6) : si la réinitialisation d'après-capture est confirmée, elle se lit dans
   les événements `mode` déjà en base et rend nommables les vies qu'elle interrompt.

### 7.3 Et le garde local, alors ?

**Le garde reste en place**, et son critère doit être **réécrit** — ce que le plan de
fiabilisation demandait déjà. Ce que la mesure permet maintenant d'écrire, et qui manquait :

- un **plancher sur TOUS les films mesurés**, pas sur « au moins deux » (un critère qu'on
  satisfait en choisissant ses films ne protège de rien — leçon déjà tirée le 2026-07-31) ;
- un **corpus nommé** : les sept films de ce document, modes et cartes explicites, à rejouer à
  chaque changement du pont ;
- une **date de réexamen**, sans quoi le garde devient le « compatibility guard forever » que le
  CLAUDE.md interdit.

Proposition à soumettre : **plancher 90 % sur les sept films, verdict du pont nominal sur tous,
date de réexamen à la clôture du chantier « nommage des vies »**. À 90 %, le corpus actuel donne
2 films sur 7 conformes ; après le chantier §7.2, le plafond calculé en donne 7 sur 7.

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

## ANNEXE — REPRODUIRE

```bash
cd apps/go-api
CGO_ENABLED=0 FILM_CACHE_ROOT=<mainrepo>/data/cache CTF_RESEARCH_OUT=<dir> \
  CTF_RESEARCH_FILMS="64e8adfa:Catalyst,000d5950:Cliffhanger,9aeca4b3:Catalyst,01e1f945:Catalyst,db7b8c3c:Aquarius,0edb8512:Aquarius,829abef9:Behemoth" \
  go test ./internal/analysis/replay/ -run CTFLostShots -timeout 60m

CGO_ENABLED=0 FILM_CACHE_ROOT=<mainrepo>/data/cache CTF_RESEARCH_OUT=<dir> \
  CTF_BRIDGE_FILMS="64e8adfa:Catalyst,0edb8512:Aquarius,01e1f945:Catalyst,db7b8c3c:Aquarius" \
  go test ./internal/analysis/replay/ -run CTFBridge -timeout 60m
```

Sans ces variables, les deux tests se sautent (`--- SKIP`, message nommant la variable). Le
corpus film est lu en LECTURE SEULE ; aucune sortie n'est écrite dans `data/`.

**Coût mesuré** : environ 4 à 6 min par film sur ce poste (trois sessions en parallèle), soit
~40 min pour les sept films et ~25 min pour les quatre anatomies.

**Les sorties brutes de cette session sont versionnées** sous
`replay2d/mesures_ctf_2026-08-08/` (11 fichiers, 48 Ko) : tout chiffre de ce document s'y relit
sans re-décoder un film.
