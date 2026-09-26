# Gates du rejeu 2D — une seule session — 2026-08-17

> Tout ce qui attend ton jugement depuis le 16/08, dans l'ordre où le faire.
> Compte ~25 min sur la planche (A), ~20 min dans l'app (B), et deux réponses en fin de course (C).
>
> **Planche** : `effets_rejeu_2d.html` republié sur l'artefact existant. Tes verdicts du 16/08 y
> sont toujours (même clé de stockage). 27 items : 6 **nouveaux**, 11 **à rejuger**, 2 **gate en
> app**, 8 inchangés. Le bandeau de gauche filtre par ces trois catégories — commence par
> « Nouveau depuis le 16/08 » puis « À rejuger », les 8 autres sont là pour mémoire.

---

## (A) Sur la planche — ce qu'il faut regarder et écouter

Bascule « Fond des aperçus » (Sombre / Clair) en haut à gauche : chaque effet doit tenir dans
les deux. Bouton « Copier le bilan » en bas à gauche quand tu as fini.

### Sur la carte (8 items)

| item | ce qu'il faut regarder | valeur déclarée à l'écran |
|---|---|---|
| **A1** marqueur, traînée, cône, nom · *à rejuger* | Le cône « un peu plus prononcé » comme demandé. Le NOM sous le point (nouveau, lot habillage). Le losange = ami, le disque cerclé = toi. Est-ce lisible à trois vies ? | cône 52 px, demi-ouverture 0,42 rad, alpha 0,55 · nom 8,5 px semi-gras, contour 2,6 px · noyau 3,4 px (+0,7 px/étage) · croix 5 px fixe · traînée 7 s |
| A2 éclair de bouche | Rien n'a bougé au rendu — la bascule « Effets de tirs » (ON) est nouvelle, elle se juge en (B) | 6 frames ≈ 600 ms |
| A3 effet de mort | Rien n'a bougé au rendu, mais il est désormais **éteint par défaut** à ta demande. Toujours d'accord ? | fenêtre 1,5 s |
| **A4** explosions et nappe Dynamo · *à rejuger* | Tu disais « trop bref ». Elles durent maintenant **2,4 s**. Et **la nappe Dynamo, que tu n'avais pas pu voir, est là** (4e vignette) | explosion 2,4 s (flash 120 ms, onde 650 ms) · nappe Dynamo 2,5 s |
| A5 grappin | Inchangé ; l'aperçu est passé du schéma au vrai calque | ligne 1,25 px, alpha 0,85 · ancre = point monde fixe |
| **A6** zones de callout · *à rejuger* | La police était « trop grande » : elle est passée de 25 px à **9,5 px**. L'aperçu pose un nom de joueur à côté — c'est LUI la borne. Le débordement se juge en (B), sur une vraie carte | libellé 9,5 px, cerne 1,9 px · borne = taille d'un nom de joueur + 1 px |
| A7 objectifs et fond | Pour mémoire (bornes Forge reprises depuis) | — |
| **A8** carte de chaleur · *NOUVEAU* | Deux lectures : présence et éliminations. La rampe bleue est-elle assez lisible sur un fond de carte ? En manque-t-il une (morts subies) ? | cellule 0,5 m · σ 2 m · échelle p50 → p95 · opacité 0,12 → 0,55 |

### Les objets posés sur le terrain (4 items — TOUS nouveaux)

C'était ton refus n°1 du 16/08 (« j'aimerais les murs de protection et capteurs »).

| item | ce qu'il faut regarder | valeur déclarée à l'écran |
|---|---|---|
| **W1** mur de protection | **La forme est un ARC, pas un rectangle**, parce que tu as dit « ça laisse passer les dégâts dans un sens et pas dans l'autre » : la concavité regarde le poseur. À droite de l'aperçu, le même mur **sans cap de visée** : cercle pointillé, aucune orientation inventée | rayon **1,6 m DÉCLARÉ** (choix d'écran calibré : 110° → corde 2,6 m ; le film ne porte ni dimension ni orientation d'objet) · plancher de lisibilité 6 px · trait 2 px + halo |
| **W2** capteur de menaces | Le disque à demeure, l'**onde de ping**, et la marque « révélé » sur l'adversaire quand il est dans le rayon **au moment du ping** | rayon **4,25 m** · ping **1,8 s** · révélation **0,75 s** — **source : Halo Waypoint, « Sandbox Overview Season 4 »**, sous l'hypothèse écrite **1 wu = 1 m** (l'autre conversion citée, 3,048 m/wu, donnerait 26 m de diamètre) · course de l'onde 400 ms = choix de rythme, pas une mesure |
| **W3** balise, traqueur, champ · *sans témoin visuel* | **Rien à comparer** : aucune image du jeu, aucune portée, aucune orientation ne sont publiées pour ces trois-là. Les formes sont des choix d'écran — à approuver ou refuser, pas à vérifier | balise losange 5,5 px d'écran, **sans pulsation** (rien ne bat) · traqueur : **une seule** impulsion de 400 ms, rayon 14 px d'écran (pas en mètres : aucune portée publiée) · champ **3 m DÉCLARÉ**, borne **pointillée** pour dire qu'elle n'est pas affirmée |
| **W4** le filtre « déployé » | La table des familles, telle qu'elle est dans le code. **88,6 % des « poses » sont des objets LÂCHÉS À LA MORT**, pas déployés : le calque ne dessine que les déploiements (19 formes au lieu de 47 sur le témoin). Veux-tu une bascule pour voir aussi les lâchers ? | grenades et capacités portées = `null` explicite · power-ups = absents de la table (le corpus n'en porte qu'un de chaque, tous deux lâchés à la mort) |

### Les fiches joueur (6 items — maquettes HTML, le vrai gate est en B)

| item | ce qu'il faut regarder |
|---|---|
| B1 éclat de mort | Inchangé (0,62 s × 3) |
| **B2** réapparition · *à rejuger* | Tu voulais « plus lent » : 0,55 s → **1,2 s**, et le texte **« Réapparition dans X s »** |
| B3 vitalité · B4 armes | Inchangés |
| **B5** grenades, capacité, munitions · *à rejuger* | Tu voulais « les images pour les grenades, pas de texte sauf le compteur » : les 4 vignettes versionnées y sont, l'encre suit le fond. Et le **glyphe SVG** (pas un caractère) pour une capacité non identifiée |
| B7 verre et encadré doré · *gate en app* | Déjà validé le 16/08, LIVRÉ ; l'ancien rendu (opacité 0,4) a été supprimé du code. La maquette n'est qu'une imitation — voir (B) |

### Le fil des morts (1 item)

| item | ce qu'il faut regarder |
|---|---|
| **C1** marque d'assistance · *à rejuger* | Plus aucun mot « assisté par ». À la place : la **vignette d'assistance du jeu** (`killfeed-62`, celle que tu as désignée), le nom, et **« - N % »** de participation. Deuxième ligne de l'aperçu |

### Les sons (6 items — casque, ils sont ré-encodés en Opus mais la DURÉE est celle qui est livrée)

| item | ce qu'il faut écouter | durées livrées (mesurées) |
|---|---|---|
| **D0** règles · *à rejuger* | La durée n'est plus imposée par le lecteur : **elle est portée par le fichier, catégorie par catégorie** | plafond de sûreté 4,0 s · fondu 0,25 s · 8 voix |
| **D1** tirs par arme · *à rejuger* | **Écoute d'abord les 4 nouvelles** (marquées « nouveau ») : ton extraction les a fournies, il n'y a **plus aucune arme muette** | 26 armes · 1,200 s sauf MA40 AR 1,055 s et MA5K Avenger 1,051 s |
| D2 lancers | Inchangés | 4 × 1,200 s |
| **D3** explosions et mêlée · *à rejuger* | **Les quatre ont été re-coupées.** La Dynamo part de « Full », le début « throw » retiré (attaque mesurée à +33,63 dB à 3,350 s, coupe à 3,310 s). **Attention à la frag** : voir question (C1) | frag **3,335 s** · plasma 4,000 s · spike 3,951 s · dynamo 1,527 s · mêlée 1,200 s |
| **D4** équipements · *à rejuger* | Re-coupés eux aussi (« écourtés » disais-tu). En Fiesta, un dash = un Activate camo : 3,4 s de son à chaque dash, est-ce trop bavard ? | camo 3,416 / 1,903 s · surbouclier 3,996 / 2,155 s |
| **D5** sons de POSE · *NOUVEAU* | Deux gestes de pose sonnent. Les trois autres familles sont **muettes par mesure** : ta bibliothèque ne contient aucun fichier pour le traqueur, la balise ni le champ | mur 2,940 s · capteur 1,265 s |

### Réglages et refus (2 items)

| item | ce qu'il faut regarder |
|---|---|
| **E1** tiroir · *gate en app* | La maquette ne montre que les libellés et l'ordre. Le vrai gate est en (B) |
| **F1** refus mesurés · *à rejuger* | Deux lignes en sont SORTIES (mur, capteur). Le prochain gros verrou est l'**état vivant des objectifs** (crâne, drapeau, noyau) — ta demande n°2 |

---

## (B) Dans l'app seulement — ce que la planche ne peut pas montrer

**Prérequis** : `make dev` (go-api sur :8000 + vite). Cinq choses sont du React ou dépendent
d'une vraie carte, elles n'existent pas dans un fichier HTML autonome.

### Sur quels matchs — la liste vérifiée sur pièces, cache local du 17/08

| match | carte / mode | schéma | ce qu'il sert |
|---|---|---|---|
| `000d5950` | **Cliffhanger, Super Fiesta Slayer** — le film de référence | **10** (re-cuit le 17/08 à 12:59) | **mur (28 poses) et capteur (19 poses)**, grappin (25 tractions), camouflage (36 épisodes), tirs et grenades nominaux. C'est AUSSI le témoin de non-régression |
| `06dfe6d9` | **Threshold** (canevas Forge, bornes corrigées) | **10** (re-cuit le 17/08 à 13:31) | **le seul** qui porte le traqueur, le champ de réparation et des objets `other` — donc **le seul terrain de W3 et de la bascule « objets non identifiés »**. Après le filtre « déployé », il dessine 40 formes : 18 murs, 12 capteurs, **9 champs de réparation, 1 traqueur** — et **0 balise** (voir ci-dessous). Attention : son pont slot→joueur est déclaré « non publiable » et 88 % de ses tirs ne se rattachent pas — juge les POSES, pas les tirs |
| `084a804d` | Fortitude, BTB Heavies CTF | **7** (cuit le 16/08) | camouflage et surbouclier (les épisodes) — **mais AUCUNE pose ni ligne de grappin** : l'artefact est antérieur au schéma 9/10. Et ses bornes sont périmées (l'un des trois artefacts locaux à re-cuire) : la carte peut être mal cadrée |
| `64e8adfa` | Catalyst, CTF | ancien | callouts + objectifs |

**Correction par rapport au brief** : `000d5950` est Cliffhanger, pas Bazaar (Bazaar Super Fiesta,
c'est `00502e52`) ; et `084a804d` **ne peut pas** servir le gate des poses ni celui du grappin.

### Les sept gates

1. **Tiroir en overlay** (E1) — ouvrir « Réglages » sur `000d5950` : le panneau passe-t-il bien
   PAR-DESSUS la carte, sans que le canvas se retaille ? Vérifier les deux bascules d'effets
   (**Effets de tirs** ON, **Effets de mort** OFF) et le **(i)** qui dit « la couverture des tirs
   peut ne pas être totale ». Fermer, recharger : la vitesse et les calques doivent avoir survécu.
2. **Effets de fiche en direct** (B7) — `084a804d` ou `000d5950`, colonne de droite : le VERRE du
   camouflage (le contenu doit rester lisible à travers) et l'ENCADRÉ DORÉ du surbouclier. En
   Fiesta, la fiche réagit à chaque dash — c'est attendu, pas un bug.
3. **Habillage** — noms sous les points, formes (losange ami / disque cerclé pour toi), logo,
   rangée « fil | carte | fiches ». Seuls les noms et les formes sont dans la planche ; le reste
   est du DOM.
4. **Callouts sur une vraie carte** (A6) — `000d5950` puis `64e8adfa` : la police à 9,5 px, et
   surtout **le débordement des polygones hors décor**, qui ne peut se juger que là. 19 cartes sur
   22 sont découpées, 3 restent brutes.
5. **Poses et capteur en situation** (W1, W2, W4) — `000d5950` : **19 formes dessinées** (15 murs,
   4 capteurs) sur 295 poses publiées, les 276 autres étant des objets lâchés à la mort. Regarder
   l'échelle : un arc de 1,6 m et un disque de 4,25 m sur une carte réelle.
   Pour **W3**, `06dfe6d9` — mais **seulement le champ de réparation (9) et le traqueur (1)** :
   **la balise n'a AUCUN témoin dans tout le cache local** (une seule pose, et elle est lâchée à
   la mort, donc jamais dessinée). Son losange ne se juge que sur la planche.
   **Ajout post-fusion du 18/08** : le capteur reste désormais affiché sur toute sa **durée
   OFFICIELLE de 15 s** (`SENSOR_DURATION_MS`) même quand le film mesure une fenêtre plus courte,
   et les AUTRES poses (mur compris) ne s'effacent plus à `t1` — elles restent visibles **jusqu'à
   la fin du rejeu**. `t1` ne date que l'instant où l'objet cesse de bouger, jamais sa disparition
   (correctif de la revue des poses du 18/08, F1). Vérifiable sur `000d5950` : un mur ou un
   capteur posé tôt dans le match doit encore être visible en toute fin de partie, pas seulement
   pendant sa fenêtre mesurée d'origine.
6. **Deux murs quasi confondus** (F11, revue adversariale des poses du 18/08) — `000d5950` **vers
   3:55** (image 2352, intervalle 100 ms) : deux poses de mur `0x8e2dc574`, à 0,13 m et 100 ms
   l'une de l'autre, mais avec deux SLOTS de poseur distincts (553 et 556) et des caps quasi
   OPPOSÉS (354,4° / 182,2°, écart 172,2°) — `replay/testdata/assembly_000d5950.golden:109-110`.
   Deux coéquipiers posant dans la même embrasure à 100 ms près est plausible en arène ; une vie
   DÉDOUBLÉE l'est aussi, et rien dans le document ne tranche entre les deux. **GATE VISUEL** :
   regarder cet instant dans l'app — un seul mur visible ou deux arcs quasi superposés ? Si
   c'est un dédoublement, la clé de déduplication de `confirmPlacements`
   (`filmdec/equipment_placements.go` — triplet slot/génération/T0 de la vie) est insuffisante et
   devra être complétée par la position.
7. **Fil des morts — la marque d'assistance en DEUX pictogrammes** — la vignette du jeu
   `killfeed-62` (déjà sur la planche, item C1) est désormais suivie d'un second pictogramme, le
   glyphe ami/moi du tueur assistant (`PlayerMark`, le même losange/disque cerclé que sur la
   carte) — habillage déjà en place au moment de la construction de cette planche, mais que le
   mockup HTML de C1 ne reproduit pas (React uniquement). À vérifier dans l'app sur `000d5950` :
   la ligne d'assistance du fil doit porter les deux pictogrammes l'un derrière l'autre, avant le
   nom de l'assistant.

**Témoin de non-régression** : `000d5950` (Cliffhanger). Si quelque chose s'y est cassé — tirs,
grenades, pont slot→joueur — le journal de cuisson le dit : les trois verdicts doivent rester
`nominal`.

---

## (C) Les deux questions ouvertes — il me faut une réponse

### C1. L'explosion de fragmentation : 3,335 s ou 1,2 s ?

Le 16/08 tu as écrit « pour la grenade frag c'est ok » — elle durait alors **1,2 s**, et c'était
un VALIDÉ. Le lot des durées l'a re-coupée **avec les trois autres**, à **3,335 s**, pour que les
quatre soient cohérentes entre elles. Personne ne t'a demandé si tu voulais ce changement-là.

- **Garder 3,335 s** : les quatre explosions ont le même souffle ; une frag sonne comme une frag,
  pas comme un tir.
- **Revenir à 1,2 s** : ton verdict initial est respecté, mais la frag redevient plus sèche que la
  plasma (4,0 s) et la spike (3,95 s) qui explosent à côté d'elle.

Écoute D3 sur la planche, les quatre à la suite, et tranche.

### C2. Le mur en arc concave : est-ce le bon parti ?

Tu as dit « je validerai plus tard » quand la forme a été décidée. Elle vient de ta phrase :
« cet équipement laisse passer les dégâts dans un sens et pas dans l'autre ». D'où **un arc dont
la concavité regarde le poseur** — ses tirs sortent par l'intérieur, ceux d'en face butent sur la
face convexe — posé **devant** lui, à 1,6 m, dans la direction de son regard.

Ce que l'arc n'affirme PAS : ni les dimensions réelles de l'objet (le film n'en porte aucune), ni
son orientation propre (le cap `h` est celui du REGARD du poseur, seule quantité mesurée). Sans
cap — une pose sur sept — c'est un cercle pointillé.

Trois réponses possibles : **l'arc convient** · **il faut un rectangle** (mais alors sur quelle
orientation, puisque seule celle du regard est mesurée ?) · **il faut autre chose** — dis quoi.

---

## Ce que je fais de tes réponses

« Copier le bilan » sur la planche produit un texte prêt à coller : une ligne par item avec
`[VALIDÉ]` / `[À REVOIR]` / `[sans avis]`, ta remarque, et les deux questions ouvertes en fin de
document. Colle-le tel quel — je le convertis en lots comme le 16/08.
