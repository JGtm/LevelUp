# GUIDE — la ventilation des tirs par arme (`shared.match_weapon_shots`)

> Etat au 2026-07-28, apres 7ter.89. Lot `tw`. Journal : `.ai/RE_LOG_KILLWEAPON.md` section
> **7ter.78**, plus **7ter.85 / 7ter.87** pour la precision par arme (§3bis) et **7ter.88 /
> 7ter.89** pour ce qui attribue un tir a un joueur et pour le piege de methode qui va avec
> (§3ter). Les deux sections ont ete ajoutees le 2026-07-28.
> Ce guide fait foi sur le SCHEMA, l API du persister et la PORTEE de la donnee.
> Il ne remplace pas `.ai/GUIDE_KILLSOURCE.md`, qui porte sur une AUTRE question
> (l arme du kill) et un AUTRE espace d identifiants.

---

## 0. LA PHRASE A RETENIR AVANT TOUT

**LE TOTAL PAR JOUEUR N APPORTE RIEN DE NEUF. LA SEULE INFORMATION NOUVELLE EST LA VENTILATION.**

`match_participants.shots_fired` donne deja le total, et il le donne mieux : c est la reference.
Cette table existe pour dire **avec QUELLE arme** ces tirs ont ete faits. Tout usage qui se
contente de `SUM(shots_decoded)` par joueur reconstruit, moins bien, une colonne qui existe.

**ET LA SECONDE PHRASE, AJOUTEE LE 2026-07-28 : SI VOUS ALLEZ CALCULER UNE PRECISION PAR ARME,
LISEZ §3bis AVANT D ECRIRE LA REQUETE.** Le calcul naif — agreger tirs et touches par arme sur
tout le corpus — **INVERSE l ordre du MA40 et du Sidekick** par rapport a la reference de l API.
Ce n est pas une imprecision, c est une reponse fausse, et seules **quatre armes** sont
publiables.

**ET LA TROISIEME, AJOUTEE LE 2026-07-28 AU SOIR : SI VOUS VOULEZ LA PRECISION D UN JOUEUR — PAS
CELLE D UNE ARME — N OUVREZ PAS CETTE TABLE. LISEZ §3quater.** Le film porte un estimateur de
precision **par joueur** qui ne demande **NI l API NI un denominateur choisi** : le **taux de
remplissage** des enregistrements de degat. Erreur absolue mediane **0.0266**, `r = +0.8204`.
C est la forme a lire pour publier une precision de joueur. **Elle ne remplace pas §3bis et ne le
contredit pas** : ce sont deux flux differents et deux questions differentes — la ventilation par
ARME reste ce que §3bis en dit, sans un mot de change (§3quater.4 tranche le point).

---

## 1. CE QUE LA TABLE PORTE

| | |
|---|---|
| Grain | `match x joueur x arme` |
| Mesure | `shots_decoded` — **une seule colonne**, le nombre de fire-events decodes |
| Portee | `publishable` + `gate_reason` (verdict de la porte), `decoder_rev`, `decode_pass` |
| Ecriture | INSERT purs, append-only (ADR 0026/0030) |
| Lecture | **`match_weapon_shots_latest` UNIQUEMENT** |

### 1.1 Un fire-event est un tir, exactement — SUR ARSENAL A TRACE INSTANTANEE

> ⚠ **PORTEE AJOUTEE LE 2026-07-28 (RE_LOG 7ter.101, index §22) — ELLE EST DANS LE TITRE PARCE
> QU ELLE CHANGE LE SENS DE LA MESURE.** La loi `k = 1` vaut sur les arsenaux a **trace
> instantanee**. Sur un arsenal a **projectiles**, le meme evenement ne compte plus les tirs :
> il suit les **TOUCHES**. Mesure par famille de mode, egalite a `|d| <= 5` contre un fond
> permute intra-film de 200 tirages : en **Tactical Slayer** (BR75 seul), `events == shots_fired`
> rend **45** egalites reelles contre **14.3** permutees (max 24) — **0/200** ; en **FIESTA**,
> c est `events == shots_hit` qui bat sa nulle, **174 contre 134.7** (max 163) — **0/200** —
> pendant que `events == shots_fired` y tombe **SOUS** la sienne (8 contre 25.7). Le BTB se
> comporte comme le Fiesta (43 contre 29.1, 0/200). **C est la cause mesuree du refus de la porte
> en Fiesta (§3 reserve 1) : ce n est pas un deficit de decodage, c est un CHANGEMENT D UNITE.**

**Loi k = 1, sans coefficient.** Mediane 0.994 sur 3 129 observations de mode standard, 84.0 %
dans une tolerance de 10 %. Corpus : 946 films, 10 779 observations joueur x match,
2 214 414 fire-events. Controle negatif par permutation intra-film : erreur reelle 0.0735 contre
0.4377 au hasard, **0 sur 500 tirages**.

**N APPLIQUER AUCUN COEFFICIENT PAR ARME.** Appris sur des films disjoints, un tel tableau fait
**deux fois PIRE** que k = 1 sur l erreur du joueur typique (0.0450 contre 0.0216). C est de
l absorption de residu, pas un modele.

### 1.2 Ce que la table REFUSE de porter

| Colonne absente | Pourquoi |
|---|---|
| **touches** | **`HitLikely` de `weapon_scanner.go` EST MORT — ne jamais le publier, ne jamais le lire.** Il annonce **75-79 %** de touches ; la precision reelle a pour **mediane 0.446** (q10 0.321 / q90 0.547 sur 3 129 joueurs). Sur un joueur : **314 « touches » pour 130 reelles**, avec MOINS de tirs decodes que la verite. **C EST LE 75-79 % QUI EST FAUX** — la bande 27-45 % souvent citee a cote est, elle, l ordre de grandeur **NORMAL** de la precision reelle, pas une anomalie. ⚠ **MAIS LA QUESTION DES TOUCHES N EST PLUS FERMEE** : un **AUTRE** bit, `eventStart+106` polarite 0, porte bien une information de touche — facteur 1.8 hors echantillon sur **1 556** joueurs, controle intra-arme, **reproduit par un second instrument ecrit de zero** (7ter.82). **PORTEE STRICTE** : mode STANDARD, armes a **trace instantanee** ; il vaut **ZERO sur les armes a projectile lent** (Needler 0.0075) et il est **BATTU en Fiesta**. La contradiction avec le balayage anterieur (656 positions, 8 joueurs, << aucune ne bat un taux constant >>) est **RECONCILIEE, pas ouverte** : refait a cette taille, un balayage aveugle ne DESIGNE `+106` que sur **42.3 %** des films alors que le drapeau y bat deja le taux sur **172/196** — deficit d effectif, pas desaccord (7ter.82 (7)). Cf. 7ter.80 (7)(8)(9) + 7ter.82. **ET POUR LES ARMES A PROJECTILE, LA QUESTION EST FERMEE PAR UNE AUTRE VOIE** : les impacts existent (codes 6 et 7) mais aucun evenement ne nomme leur TIREUR, et il n existe AUCUN evenement de creation de projectile — §3ter.2. |
| tete / corps | Hors de portee. Jamais mesure. |
| coefficient par arme | Cf. 1.1 — sur-ajustement mesure. |
| nom de l arme | Se resout a la lecture (`metadata.weapon_labels`). Figer un nom obligerait a redecoder 949 films pour re-etiqueter un referentiel. |
| total du joueur | C est un `SUM`, et il n a jamais ete l objet (cf. §0). |

### 1.3 L identifiant d arme — DEUX UNITES QUI NE SE MELANGENT JAMAIS

```
weapon_id    (ici)                 identifiant FILMSHELL 64 bits, lu a eventStart+40 sur
                                   64 bits. C est celui de metadata.weapon_labels.weapon_id
                                   (UBIGINT, 39 armes). Colonne UBIGINT : le Fuel Rod SPNKr
                                   (0x9d6aaed242c9679f) deborde un BIGINT signe.

source_tag   (match_kill_events)   tag jpt! 32 bits du dead-state de la VICTIME.
                                   AUTRE ESPACE, AUTRE LARGEUR, autre question.
```

Les joindre l un a l autre rend zero ligne, silencieusement.

---

## 2. LA PORTE DE PUBLICATION, ET CE QU ELLE COUTE

**Une ventilation n est publiable que si le TOTAL decode du joueur tombe a +-10 % de son
`shots_fired`.**

**CETTE PORTE EXIGE LA REFERENCE API.** Aucun detecteur offline ne la remplace, et ce n est pas
faute d avoir cherche :

- taux de pas du compteur de tirs : correlation **r = 0.15** avec l erreur ;
- porte sur le NOMBRE D ARMES du joueur : **+3.8 points** en jetant **46.6 %** des observations.

C est une dependance assumee, ecrite ici plutot que decouverte plus tard.

### 2.1 Le verdict est CALCULE par le persister, jamais fourni

`EvaluateShotsGate(decodedTotal, shotsFired)` est **l unique copie** de la regle. Le collecteur
fournit la reference (`WeaponShotsPlayer.ShotsFired`), pas le verdict. Consequence : il est
IMPOSSIBLE de publier `publishable = true` sans avoir eu de reference.

| `gate_reason` | Sens |
|---|---|
| `total-dans-tolerance-10pct` | seul cas publiable |
| `total-hors-tolerance-10pct` | ecart superieur a 10 % |
| `reference-shots-fired-absente` | aucune reference : la porte REFUSE, elle ne suppose pas |
| `reference-nulle-decodage-non-nul` | l API dit 0 tir, le decodeur en attribue. **Erreur d attribution**, pas un ecart |

`gate_reason` est renseigne **meme quand la porte passe** : une portee qui ne s ecrit que sur les
echecs laisse croire que le succes n en a pas.

### 2.1bis CE QUE LA PORTE LAISSE PASSER — mesure sur les 949 films du cache

```
                          joueurs (shots_fired > 0)   dans la porte +-10 %   ratio mediane
roster <= 16                        6 811                3 748  (55.0 %)         0.968
roster > 16                         2 911                  584  (20.1 %)         0.557
```

**PORTEE DE CE CHIFFRE, et elle change tout** : ces 55.0 % couvrent **TOUS LES MODES melanges**,
Fiesta compris — pas le seul mode standard. La loi k = 1 est etablie a **84.0 % dans la meme
tolerance sur le mode standard SEUL** ; les deux chiffres ne se contredisent pas, ils ne portent
pas sur la meme population. La mediane du ratio, elle, est **0.968** en dessous de 17 joueurs :
c est la loi k = 1 qui se lit directement, sans coefficient.

Au-dela de 16 joueurs, la mediane tombe a **0.557** : le grand format **SOUS-decode de moitie**
en son milieu, tout en portant une queue de SUR-attribution (moyenne 1.033 pour une mediane
0.557). C est la reserve n°1 du §3, mesuree.

L indice du film reste non resolu pour **272 joueurs sur 7 489** (roster <= 16) et **64 sur
3 298** (roster > 16) : ces joueurs n ont aucune ligne du tout (cf. §3.2).

### 2.2 `publishable = FALSE` NE VEUT PAS DIRE LA MEME CHOSE QUE DANS `match_kill_events`

| Table | `publishable = FALSE` signifie |
|---|---|
| `match_kill_events` | juste EN AGREGAT, faux ligne par ligne (marge de bijection nulle, BTB) |
| **`match_weapon_shots`** | **NE PAS UTILISER.** Une ventilation hors tolerance est fausse EN AGREGAT AUSSI — c est le total qui est faux |

Un lecteur de cumul ne peut pas rattraper par la somme ce que la porte vient de refuser.
**Tout lecteur de cette table filtre `WHERE publishable`.**

### 2.3 La vue ne filtre PAS

`match_weapon_shots_latest` rend aussi les lignes refusees. Filtrer dans la vue rendrait le refus
invisible : un lecteur ne saurait plus distinguer « aucun tir decode » de « decodage refuse ».
Le filtre appartient au lecteur, et il est obligatoire.

---

## 3. TROIS RESERVES QUI VOYAGENT AVEC LA DONNEE

1. **FIESTA ET GRAND FORMAT NE SONT PAS LIVRABLES.** 6.9 % et 38.8 % des joueurs seulement
   tombent dans la tolerance, et **5.1 % du grand format SUR-attribue de plus du double**. Pire :
   **168 observations ou l API dit `shots_fired = 0` EXACTEMENT et ou le decodeur attribue
   32 090 tirs** (zero cas en mode standard). **Ce n est pas une perte, c est une erreur
   d attribution.** La porte les refuse une par une — il n y a donc PAS de colonne « mode » :
   c est la mesure qui tranche, pas une liste tenue a la main.

   > **LA CAUSE DU REFUS EN FIESTA EST MESUREE DEPUIS LE 2026-07-28 (7ter.101, index §22), ET CE
   > N EST PAS UN DEFICIT DE DECODAGE — C EST UN CHANGEMENT D UNITE.** En Fiesta, l evenement
   > compte les **TOUCHES**, pas les tirs : agregat de famille `events/shots_fired` **0.3147**
   > contre `events/shots_hit` **1.1046** (Tactical : **0.9457** et **3.4007**), et l egalite
   > `events == shots_hit` a `|d| <= 5` bat son fond permute intra-film **174 contre 134.7**
   > (max 163, **0/200**) pendant que `events == shots_fired` tombe **sous** le sien (8 contre
   > 25.7). **Consequence pratique : ne jamais << reparer >> le Fiesta par un coefficient.** Une
   > ligne Fiesta rendue publiable par calibrage publierait un compte de TOUCHES sous le nom
   > `shots_decoded`. La porte a raison de la refuser.
2. **LES ARMES RARES ET LOURDES SONT SOUS-ESTIMEES** (Skewer : 5 204 events sur 2 214 414).
   Aucune population « arme dominante » n existe pour les calibrer, donc **aucun correctif n est
   applique** — un correctif non calibrable serait une invention.
3. **LE DEFICIT DES ARMES AUTOMATIQUES EST INEXPLIQUE.** **Ce n est PAS la deduplication** par
   proximite d octet : mesure a **174 161 events identiques avec et sans**, et re-mesure par
   comptage exhaustif en 7ter.80 (3) — **ZERO** event supprime sur 7 343 955 marqueurs.

   > ⚠ **AVERTISSEMENT (2026-07-27) — LES CHIFFRES « MA40 0.928, Sidekick 0.925 contre BR75
   > 0.981 » (n = 275/80/313/55) SONT IRREPRODUCTIBLES ET NE DOIVENT PLUS ETRE CITES.** Ils
   > n existaient que comme assertion **ici meme** : aucune section du RE_LOG, aucun outil,
   > aucune methode ecrite. Deux origines ont ete testees en 7ter.80 (6) — indice joueur sur
   > 4 bits (**REFUTE** : ratios 1.94 a 2.15) et numerateur restreint a l arme dominante avec
   > denominateur total (**PARTIEL** : reproduit MA40 a 0.924, aucune des trois autres).
   > **Mesure courante du meme pipeline** (mode standard, ratio par joueur) : **MA40 0.971 ·
   > Mk51 Sidekick 1.004 · BR75 1.007 · Bandit Evo 1.007**. Le Sidekick n est meme pas une arme
   > automatique (183.8 ms entre coups, sauts de compteur a rapport d horloge 0.82).
   > Un chiffre ecrit comme un fait sans l etre a coute a ce chantier une chasse entiere au
   > mauvais endroit — **statut : `[NON VERIFIE]`**.
   >
   > **CE QUI EST MESURE**, lui : les armes qui portent le phenomene sont le **MA40 AR** et le
   > **Needler**, les deux seules a cycler a 83.4 ms, et **les tirs manquants sont RECUPERABLES**
   > par le compteur de tir du fire-event (7 bits a `eventStart+22`) — 7ter.80 (1)(4).
   > ⚠ Une mesure concurrente sur une AUTRE population (949 films tous modes, arme dominante
   > >= 85 %) rend `k_fire = 0.935` au Sidekick : **le cas Sidekick est CONTESTE, pas tranche**
   > (7ter.81 (9)).

### 3.1 L absence de ligne : ce qu elle dit, et ce qu elle ne dit pas

Pas de ligne `(match, joueur, arme)` = aucun fire-event decode pour cette arme. **Ce n est une
mesure de zero QUE si la ligne du joueur est `publishable`.** Sur un joueur dont la porte a
echoue, l absence ne dit rien. *Une absence de mesure n est jamais une mesure d absence.*

### 3.2 LA LIMITE STRUCTURELLE — un echec total est INVISIBLE

Le verdict de la porte vit sur les LIGNES du joueur. Un joueur dont **aucune** arme n est
decodee n a **aucune ligne**, donc aucun verdict : son echec est indistinguable d un joueur qui
n a pas tire. **Le cas « l API annonce des tirs, le decodeur n en trouve aucun » est SILENCIEUX
dans cette table.** Le compter exige de croiser avec `match_participants` :

```sql
-- joueurs d un match dont l API annonce des tirs et dont la ventilation est absente
SELECT p.match_id, p.xuid, p.shots_fired
FROM match_participants p
LEFT JOIN (SELECT DISTINCT match_id, player_xuid FROM match_weapon_shots_latest) s
  ON s.match_id = p.match_id AND s.player_xuid = p.xuid
WHERE p.shots_fired > 0 AND s.player_xuid IS NULL
```

Ce n est pas reparable a ce grain : une ligne « zero tir » serait une mesure fabriquee.

---

## 3bis. LA PRECISION PAR ARME — CE QUI EST PUBLIABLE, ET L INTERDIT QUI VA AVEC

> **CETTE SECTION MANQUAIT AU GUIDE ALORS QUE LE RESULTAT DATAIT DE DEUX LOTS.** Le journal et
> l index portaient 7ter.85 et 7ter.87 ; ce guide, lu par qui UTILISE la donnee, ne les portait
> pas. Sources : `.ai/RE_LOG_KILLWEAPON.md` 7ter.85 (lot `pv`) et 7ter.87 (verification
> adversariale `pv.ref`), resumes dans `.ai/ETAT_DE_L_ART_KILLWEAPON.md` §17 et §17.3.
> **Portee** : 949 films caches, **571 retenus** (roster <= 16, sans participant fantome,
> ancrage unique), **4 422 observations joueur x match**, TOUS MODES.
>
> ⚠ **CETTE SECTION PORTE SUR LA PRECISION PAR ARME. POUR LA PRECISION D UN JOUEUR, VOIR
> §3quater** — autre flux, autre grain, aucune reference API, et **elle ne change RIEN a ce qui
> suit** : la liste des quatre armes, l interdit du calcul sur le corpus entier et le piege de
> l inversion MA40 / Sidekick restent tous en vigueur (§3quater.4 le tranche explicitement).

### 3bis.0 L INTERDIT — NE JAMAIS CALCULER LA PRECISION D UNE ARME SUR LE CORPUS ENTIER

**Ce n est pas une imprecision, c est une REPONSE FAUSSE : le chiffre naif INVERSE l ordre de
deux armes.**

```
                      chiffre NAIF (corpus entier)   reference API      verdict
  MA40 AR                      0.4485                    0.4196       le film le met DEVANT
  Mk50 Sidekick                0.3701                    0.4491       ... alors qu il est DERRIERE
  ecart publie                 MA40 +8 points            Sidekick +3 points
```

**Le signe est faux et l ecart est exagere de 11 points.** La cause n est pas un defaut de
decodage : le taux par arme depend de la **POPULATION DE TIREURS**. Le Sidekick est une arme de
SECOURS ; son taux mesure sur le film monte de **0.370 -> 0.408 -> 0.444 -> 0.484** selon la
seule purete de la population. Un chiffre par arme sans population est un chiffre sans sens.

```
  ecart absolu moyen a la reference API
     corpus entier                    0.0361      <- INTERDIT
     joueurs a arme dominante >= 50 %  0.0209
     joueurs a arme dominante >= 80 %  0.0144      <- le seul chiffre a publier
```

Le chiffre naif est **2.5 x plus faux**. **Il n existe aucune requete SQL legitime de la forme
`AVG(hits)/AVG(shots) GROUP BY weapon` sur `match_weapon_shots`.** Toute ventilation par arme
passe par la restriction a la population a arme dominante, et cette population se publie AVEC le
chiffre.

### 3bis.1 LA LISTE PUBLIABLE — QUATRE ARMES, ET RIEN D AUTRE

Population : les joueurs dont **une seule arme porte >= 80 % des tirs decodes** du match.
Intervalle : **+-0.03 hors echantillon** (partage des films par SHA-256 du `match_id`).

```
  arme            effectif                biais mesure du film        publiable
  MA40 AR         443 joueurs, 264 films  SURESTIME de +0.0249        OUI
  BR75            282 joueurs, 149 films  SOUS-ESTIME de -0.0108      OUI
  Mk50 Sidekick    49 joueurs (12 a >= 95 %)   -0.0050                OUI, AVEC son effectif
  Bandit Evo       54 joueurs,  13 films       +0.0132                OUI, AVEC son effectif
  S7 Sniper        11 observations, 2 films                           NON
  tout le reste                                                       NON
```

**TROIS RESERVES QUI VOYAGENT AVEC CE TABLEAU, ET QUI VIENNENT DE LA VERIFICATION 7ter.87 :**

1. **NE JAMAIS PUBLIER UN ORDRE entre MA40, BR75 et Sidekick.** Leur etendue de reference vaut
   **0.0295** et l erreur du film **0.0136** : la moitie de ce qu il y aurait a distinguer. Les
   quatre valeurs se publient comme quatre valeurs, jamais comme un classement.
2. **LE BIAIS DU BR75 N EST PROBABLEMENT PAS UNE PROPRIETE DU FILM.** Il **contient zero** au
   cran >= 95 % (-0.0107 [-0.0210..+0.0002]) et sa regression sur (1 - purete) rend une ordonnee
   a l origine de -0.0023 [-0.0103..+0.0073] : c est une **contamination de la reference** par
   les autres armes du joueur, pas un biais du decodeur. Celui du MA40, lui, n en est pas une —
   mais il **decroit** avec la purete jusqu a +0.0082 a purete 100 %, hors de l intervalle
   publie.
3. **LA RESTRICTION A L ARME DEGRADE L ESTIMATION.** Le taux des memes joueurs calcule sur
   TOUTES leurs armes est **plus proche de la reference** que le taux restreint a l arme, et
   cela **a tous les crans de purete** (7ter.87 (3)). La ventilation par arme est donc un
   affichage, pas une amelioration de mesure.

Les armes NON publiables ne se trompent pas un peu : **elles se trompent d un facteur 30 a 60**.
Sur les hitscan l ecart a la reference vaut 0.011 a 0.077 ; sur Needler, Pulse Carbine et Plasma
Pistol il vaut **0.19 a 0.25**. Et la population a arme dominante ne contient presque aucune arme
a projectile — le Needler compte **122 observations a >= 50 %, 2 a >= 80 %, 0 a >= 90 %** : il
n existe aucune base pour les calibrer, et un correctif non calibrable serait une invention.

> **CETTE LISTE NE GRANDIRA PAS PAR LA VOIE DES IMPACTS — C EST MESURE, PLUS SUPPOSE
> (2026-07-28).** La demande << la precision pour TOUTES les armes >> passait par la ventilation
> des codes 6 / 7 par arme. Le corps de ces evenements est maintenant decode champ par champ, et
> **il ne porte aucun identifiant d arme** : la porte du tag est fermee sur **168 380** impacts a
> position certaine, **zero exception**, avec un controle positif du meme instrument a
> **1.0000** et une reproduction independante a l unite (7ter.91, verifiee par 7ter.94).
> **Reponse a la question de l utilisateur : NON — la precision reste publiable pour ces QUATRE
> armes et pour aucune autre, et aucune arme a projectile n en fera partie par cette voie.**
> Ce qui reste ouvert, et qui est une AUTRE question : les touches **RECUES** (§3ter.2) — sans
> reference API.

### 3bis.2 L ACCORD PAR ARME CONTRE L API N EST **PAS** UNE VALIDATION NEUVE

C est le point le plus contre-intuitif du dossier, et il doit etre lu avant toute utilisation du
tableau. **L accord par JOUEUR entre le film et l API etait deja etabli** (7ter.81 (5)). Or tout
regroupement de ces memes joueurs reproduit l accord — **y compris un regroupement au hasard** :

```
  nulle P2 (memes joueurs, etiquettes d arme redistribuees AU HASARD)
     le reel bat la nulle    8 / 200  a purete >= 80 %
     le reel bat la nulle  197 / 200  a purete >= 50 %      <- le hasard fait aussi bien
  correlation de rang (rho de Spearman) film / API par arme
     la nulle la reproduit ou la bat  200 / 200             <- REFUTE comme preuve
```

**Consequence pratique : ne jamais presenter << le tableau par arme colle a l API >> comme une
preuve que la ventilation par arme est juste.** Elle est juste PAR HERITAGE de la mesure par
joueur, et son merite propre n est pas demontre par cet accord.

Le seul test qui SORT du regroupement et qui passe est le **contraste intra-joueur** (deux armes
chez le MEME joueur dans le MEME match) : etendue **0.3428** contre une nulle a
**0.0392 [0.0119..0.1032]**, **0/200**. Mais il ne mesure pas une precision : le contraste du
Needler vaut **-0.2085** pour une precision reelle de l ordre de 0.24 a 0.39 — **il mesure la
VISIBILITE de l arme par le decodeur autant que sa precision**, et rien ne dit laquelle des deux
on lit.

---

## 3ter. CE QUI ATTRIBUE UN TIR A UN JOUEUR, ET POURQUOI LES TOUCHES DE PROJECTILE NE SUIVRONT PAS

> Ajoutee le 2026-07-28. **Ce guide ne portait RIEN de 7ter.88 ni de 7ter.89 alors que les deux
> lots portent sur son objet meme** : ce qui relie un fire-event a un joueur, et jusqu ou cette
> relation permet d aller. Sources : `.ai/RE_LOG_KILLWEAPON.md` **7ter.88** (lot `pj.own`) et
> **7ter.89** (verification adversariale `tv.ref`), index §2.1.

### 3ter.1 LE LIEN OBJET -> JOUEUR DU FIRE-EVENT EST UNE FONCTION — `[ETABLI]`

Le fire-event (code 36) porte, a son emplacement de presence 0, un **handle d entite** (30 bits
d indice + 2 bits de generation, `FUN_1406d3140`). Ce handle **ne designe pas le joueur** : il
designe un objet. Mais l application handle -> indice de tireur est une **FONCTION**, pas une
correspondance floue :

```
  handles rendant UN SEUL indice de tireur   14 896 / 15 192 = 0.9805   (150 films, 319 883 occurrences)
  la meme mesure apres permutation INTRA-film          0.0652
  handles par nombre d indices distincts      1:14896  2:258  3:31  4:6  5:1
```

**Reproduit A L UNITE par un second binaire ecrit pour verifier** (lot `tv.ref`, 7ter.89 (1)) :
memes 14 896 / 15 192, meme table de distincts, memes volumes de paquets et d evenements. C est
ce qui autorise le statut `[ETABLI]`, et le reproducteur est nomme.

**CE QUE CET OBJET N EST PAS** : le joueur. Regroupe par joueur, un handle ne rend une seule
famille d arme qu a **0.6632** (contre 0.0502 au hasard) ; il y a **10 handles par joueur et par
match**, generation 1 dans 99.94 % des cas. << C est l arme tenue >> reste `[PLAUSIBLE]`.

**LA POSITION DU CHAMP DE TIREUR** est `+34` bits apres le debut de l evenement — et elle tient
**par deux instruments independants du depot** (7ter.26 (5) et `weaponv3.FirePi5SpanBefore`),
**pas** par le test du roster. Lire §3ter.3 avant de la deplacer d un bit pour quelque raison
que ce soit.

### 3ter.2 LES TOUCHES DE PROJECTILE NE SERONT PAS VENTILEES PAR JOUEUR PAR CETTE VOIE

Le film porte bien les impacts de projectile — ce sont les **codes 6 et 7** (7ter.86, detaille
dans `.ai/GUIDE_KILLSOURCE.md` §6bis). Mais **l evenement d impact ne nomme jamais le TIREUR** :
il nomme la cible et l objet projectile.

Et le tireur n est pas ailleurs dans le flux, parce qu **IL N EXISTE AUCUN EVENEMENT DE CREATION
DE PROJECTILE**. Le critere est net : un objet est cree exactement UNE fois, donc un evenement de
creation les COUVRE TOUS. Aucun code ne le fait.

```
  couverture de l ensemble des projectiles, 150 films
    meilleur candidat (code 5)     0.2318    nulle 0.1601
    fire-event (code 36)           0.0931    nulle 0.0711
  test sur une reference INDEPENDANTE (le candidat n en fait pas partie), 7ter.89 (3)
    meilleur candidat du corpus    0.5046    nulle de MEME POOL 0.3600    rapport 1.40
    ce qu un nommage REEL exige    1.0000                                 rapport 11.37
```

**Le projectile n existe dans le flux d evenements qu a l INSTANT DE SA MORT.**

> ⚠ **NE PLUS CITER << la somme 0.5408 + 0.4606 = 1.0014 est une partition mesuree >>.** C est
> une IDENTITE ALGEBRIQUE : l ensemble de reference EST l union des deux ensembles compares,
> donc la somme vaut `1 + recouvrement` quoi qu il arrive — reproduite a **neuf decimales** sur
> deux corpus (7ter.89 (2)). Le resultat negatif tient, mais par le rapport reel/nulle ci-dessus.

**CONSEQUENCE POUR CETTE TABLE** : `shots_hit` par arme et par joueur **n est pas atteignable par
la voie des evenements** pour les armes a projectile. C est cela, et non un defaut de decodage,
qui explique que la precision par arme soit fausse d un facteur 30 a 60 sur le Needler, le
Bulldog, la Cindershot, l Hydra et le SPNKr (§3bis).

**CE QUI RESTE A PORTEE, ET C EST UNE AUTRE QUESTION** : les touches **RECUES** par joueur. Le
code 7 nomme sa CIBLE dans le **meme pool de handles** que le code 36 nomme son objet, et §3ter.1
attribue ce pool a un joueur. Ce sont les touches **DONNEES** — la question de `shots_hit` — qui
sont fermees par ce qui precede.

> ⚠⚠ **CE PARAGRAPHE DISAIT << LES TOUCHES PAR ARME SONT A UN DECODAGE DE DISTANCE >>. C EST
> MESURE COMME FAUX — LE DECODAGE A ETE FAIT, ET IL NE DONNE RIEN** (7ter.91, verifie
> independamment par 7ter.94). Corrige le 2026-07-28 ; l ancienne phrase est retiree, pas
> nuancee.

**LE CORPS DU CODE 7 EST DECODE, ET LE TAG N Y EST JAMAIS.** Le tag est bien lu par
`FUN_14080d69c` — la MEME fonction que le tag `jpt!` du dead-state — mais **derriere une porte
qui n est jamais ouverte dans le flux** : le bit qui la commande vaut **1** sur **168 380**
impacts a position certaine (97 988 codes 7 + 70 392 codes 6, 949 films), **sans une seule
exception**, et cette population est celle du PREMIER evenement de chaque paquet, dont la
position ne depend d aucune longueur de corps.

```
  controle positif du meme instrument (reecriture d un tag jpt! connu, puis relecture)
     lot c7b     129 381 / 129 384 = 1.0000
     lot tp.ref   70 950 /  70 950 = 1.0000     (binaire independant, profondeur 0)
  test decisif : taux de touche par arme, avant / apres    AUCUNE arme ne bouge d un centieme
```

**AUCUN AUTRE CHAMP DU CORPS N EST UN IDENTIFIANT D ARME** — `w16`, le seul assez large,
porte 2 199 petits entiers (0, 19, 1, 61). **LES TROIS VOIES SONT FERMEES** : l arme dans
l evenement (7ter.91), le tireur dans l evenement (7ter.86 (5)(a)), un pont projectile -> tir
(7ter.88 (4), corrige 7ter.89 (3)).

**ET LA DEDUPLICATION PAR OBJET-SOURCE EST SANS OBJET ICI** : sans tag, aucune ligne n est nommee
par une arme, donc il n y a rien a dedupliquer par arme. L hypothese qui la motivait — << une
roquette qui blesse trois joueurs emet 3 codes 7 >> — est de surcroit **REFUTEE** : la dedup
retire 2.2-3.1 % au code 7 mais **4.9-5.9 % au code 6**, qui frappe la geometrie et ne peut pas
se diffuser (7ter.91 (6), reproduit 7ter.94 (3)).

### 3ter.3 LE PIEGE DE METHODE — LA << PURETE >> NE DISCRIMINE PAS UN ALIGNEMENT DE BITS

C est le piege qui a failli deplacer la position du champ de tireur, et **il a ete releve deux
fois le meme jour** (7ter.87 sur la ventilation par arme, 7ter.89 (4) ici).

```
  purete du lien handle -> indice de tireur      egalites exactes contre le roster de l API
    +32   0.8422   <- seule chute            +32    0 / 150  (2 / 150 sous filtre)
    +33   0.9808                             +33    0 / 150  (2 / 150 sous filtre)
    +34   0.9805   <- position retenue       +34   50 / 150  (60 / 150 sous filtre)
    +35   0.9803                             +35   50 / 150  (69 / 150 sous filtre)  <- il BAT
    +36   0.9801                             +36   46 / 150  (60 / 150 sous filtre)
```

**LA REGLE, EN TROIS LIGNES.** (1) Une statistique de qualite interne — purete, taux d accord,
part de terminaisons propres — **exclut un cadrage grossier et rien de plus** : elle est plate
sur les positions voisines. (2) Seule l **EGALITE EXACTE contre une reference EXTERNE** discrimine
un alignement. (3) Et quand meme l egalite exacte ne separe pas, **on ne tranche pas avec un
ecart moyen** : c est la statistique agregee que la methode du chantier interdit comme
discriminant (PATRON E). Ici, `+34` tient par deux instruments du depot, pas par le roster.

**COROLLAIRE, ET C EST UN INTERDIT** : ne jamais chercher un champ en **balayant les bits hors du
parcours nominal**. Ce balayage rend des resultats, et il **FABRIQUE des distributions
credibles** — un en-tete plausible sort 1 fois tous les 18 bits, 1 fois tous les 8 366 bits avec
toutes les contraintes ; le scan cible d un film rend 292 candidats pour 87 kills dont
**193/292 sont la MEME paire** (7ter.27 (6)). Cf. `.ai/GUIDE_KILLSOURCE.md` §6bis.5.

---

## 3quater. LA PRECISION D UN JOUEUR SANS AUCUNE REFERENCE EXTERNE — LE TAUX DE REMPLISSAGE

> **AJOUTEE LE 2026-07-28. C EST LE LIVRABLE PRODUIT DE LA JOURNEE, ET IL N ETAIT DANS AUCUN
> GUIDE.** Sources : `.ai/RE_LOG_KILLWEAPON.md` **7ter.98** (lot `rc.unite`) et **7ter.101**
> (verification adversariale `rc.ref`), resumees dans `.ai/ETAT_DE_L_ART_KILLWEAPON.md` **§21** et
> **§22**. Tout ce qui suit est reproduit **a l unite par un quatrieme binaire**.
> ⚠ **CE N EST PAS CETTE TABLE.** La quantite ci-dessous se lit dans un AUTRE flux du film — les
> enregistrements de degat — et elle n a **aucun collecteur** aujourd hui, pas plus que
> `match_weapon_shots`. Elle est ecrite ici parce que c est ici qu on vient chercher << comment
> publier une precision >>.

### 3quater.1 LES DEUX OBJETS, ET IL FAUT LES NOMMER AVANT DE LIRE UN CHIFFRE

```
  RECORD    un enregistrement de degat du flux d evenements : code 36
            (`bits(pl,1,1)==1 && bits(pl,2,7)==36`, premier octet `0xD2`).
            Le chantier voisin le designe par `pay[0] >> 1 == 105` : MEME OBJET, mesure
            `1 799 630 = 1 799 630` sur 927 films.

  PORTEUR   un record dont le TABLEAU A (la table d applications de degat) est NON VIDE.
            C est un sous-ensemble strict des records.

  TAUX DE REMPLISSAGE   porteurs / records, par joueur et par match.
```

**L UNITE DU RECORD DEPEND DE L ARSENAL** (7ter.101) : c est un **TIR** sur arme a trace
instantanee (Tactical : `records == shots_fired` **45** egalites a `|d| <= 5` contre **14.3** au
fond permute intra-film, **0/200**) et une **TOUCHE** sur arsenal a projectiles (Fiesta : 174
contre 134.7, max 163, 0/200 ; BTB : 43 contre 29.1, 0/200). **LE PORTEUR, LUI, EST LA TOUCHE DANS
LES DEUX CAS.** C est ce qui fait du remplissage une precision — et **c est aussi ce qui borne sa
portee** (§3quater.3).

### 3quater.2 LE TAUX DE REMPLISSAGE **EST** LA PRECISION — `[ETABLI]`

Population **hitscan, `records >= 20`, n = 2 185** :

```
  taux de remplissage porteurs/records   mediane   0.4267
  precision API shots_hit/shots_fired    mediane   0.4462
  agregats                                         0.4250  contre  0.4366
  erreur absolue MEDIANE par joueur                0.0266
  correlation                                      r = +0.8204
```

**ET LE TEST QUI FAIT LA DIFFERENCE N EST PAS LA CORRELATION, C EST LA LOCALISATION** : quand la
precision DOUBLE d un quartile a l autre, le rapport `records/tirs` reste **PLAT** et le
remplissage la **SUIT**. Aucun facteur d echelle ne produit les deux colonnes a la fois.

```
  Tactical (n=138)   precision  0.1819  0.2465  0.2906  0.3558   <- DOUBLE
                     rec/tirs   0.9305  0.9306  0.9239  0.9457   <- PLAT
                     port/rec   0.1724  0.2222  0.2727  0.3220   <- SUIT
  Arena    (n=2087)  precision  0.3495  0.4248  0.4771  0.5407
                     port/rec   0.3396  0.4092  0.4630  0.5227
```

**CONTROLE NEGATIF, ET IL SE LIT EN CROIX** — fond permute intra-film, 200 tirages, `|d| <= 5`,
population hitscan n = 2 225. Aucun facteur d echelle ne produit ce tableau : deux cases battent
leur fond a **0/200**, les deux autres tombent **SOUS** le leur.

```
  records  == shots_fired    reel 109   permute  78.5  (max 98)    -> 0/200
  records  == shots_hit      reel  12   permute  52.8              -> SOUS le fond
  porteurs == shots_hit      reel 251   permute 182.0  (max 210)   -> 0/200
  porteurs == shots_fired    reel   3   permute  25.7              -> SOUS le fond
```

**HORS ECHANTILLON** (partage par SHA-256 du prefixe, puis inversion) : `med(porteurs/touches)`
**0.8128** et **0.8153**. Sur la quantite publiee (`porteurs/records` contre
`shots_hit/shots_fired`, 579 films, n = 4 607) : moitie A **r = 0.7773** (n = 2 380), moitie B
**r = 0.7701** (n = 2 227).

**L APPARIEMENT INDICE -> JOUEUR N EST PAS EN CAUSE**, et c est mesure sur la vraie quantite
publiee : ancrage film-seul `r = 0.7740` / MAE 0.0802 ; **ordre de la base** `r = 0.5731` /
MAE 0.1141 ; **nulle permutee 200 tirages** `r = 0.5730 [0.5586..0.5878]` — **l ordre de la base
EST la nulle permutee au quatrieme chiffre**, et le reel la bat **0/200**.

### 3quater.3 LES QUATRE RESERVES QUI VOYAGENT AVEC CE TAUX

1. **LE CONTROLE ZERO NE PASSE QU A 19/34 (0.5588).** Sur les joueurs dont l API annonce
   `shots_hit = 0`, seuls 19 sur 34 rendent zero porteur — films sans participant fantome inclus.
   La cause identifiee est un **participant fantome qui vole l indice d un vrai joueur**. **C est
   la reserve la plus dure du dossier** : elle dit qu une fraction des porteurs attribues a un
   joueur ne sont pas les siens.
2. **LE DECODAGE PORTE UN DEFICIT D ENVIRON 7 % SUR LES RECORDS ET DE 15 % SUR LES PORTEURS**, et
   il n est **pas explique**. Le taux survit parce que le deficit se compense en partie au
   quotient — ce n est pas une raison de le corriger par un coefficient (PATRON E).
3. **SEUL LE TAUX EST PUBLIABLE, JAMAIS LE COMPTE.** `porteurs / shots_hit` vaut **0.8509** en
   mediane : un compte de porteurs n est pas un compte de touches, et le publier comme tel
   annoncerait 15 % de touches en moins.
4. **PORTEE MESUREE : ARSENAL A TRACE INSTANTANEE.** Elle est etablie sur la population hitscan
   (n = 2 185) et corroboree par quartile sur l Arena (n = 2 087). **Elle n est PAS etablie en
   Fiesta ni en BTB**, ou le record lui-meme change d unite (il y suit les touches) — le quotient
   `porteurs/records` n y mesure donc plus une precision, et rien ne dit ce qu il mesure.

**ET UNE REGLE D ECRITURE, PARCE QUE DEUX DEFINITIONS COEXISTENT** (7ter.101, index §22.6) :
ecrire **BRUTS** ou **INDEXES** a cote de tout compte. Records du corpus : **1 799 630 BRUTS**
contre **1 774 183 INDEXES** (1.41 %). Porteurs sur les 150 films de reference : **98 685 TOUS**
contre **97 556 INDEXES** (1.14 %). Sur `000d5950` les deux coincident — c est le film ou
l ambiguite ne se voit pas.

### 3quater.4 CE QUE CELA CHANGE POUR §3bis, ET C EST TRANCHE : **RIEN**

Deux formulations contradictoires dans le meme guide seraient pires que pas de guide. Voici la
decision, et elle est mesuree, pas arbitree (index §22.7) :

```
  LA LISTE DES QUATRE ARMES PUBLIABLES A +-0.03 (§3bis.1)          RESTE VALABLE, INCHANGEE
  L INTERDIT DU CALCUL PAR ARME SUR LE CORPUS ENTIER (§3bis.0)     RESTE EN VIGUEUR
  LE PIEGE DE L INVERSION MA40 / SIDEKICK                          RESTE EN VIGUEUR
  << NE JAMAIS PUBLIER UN ORDRE entre MA40, BR75 et Sidekick >>    RESTE EN VIGUEUR
```

**POURQUOI CES DEUX SECTIONS NE SE CONTREDISENT PAS** : elles ne portent ni sur le meme flux, ni
sur le meme grain, ni sur la meme question.

```
                        §3bis (precision PAR ARME)        §3quater (precision DU JOUEUR)
  flux du film          fire-events (code 36 de tir)      enregistrements de degat + tableau A
  grain                 match x joueur x ARME             match x joueur
  reference externe     API obligatoire (la porte §2)     AUCUNE
  ce qui est publiable  4 armes, a +-0.03                 un taux par joueur, MAE 0.0266
```

**CE QUE §3quater APPORTE VRAIMENT** : la precision **d un joueur** se publie desormais **sans
appeler l API et sans passer par cette table**. C est le seul enonce neuf.

**CE QU IL N APPORTE PAS, ET IL FAUT LE DIRE AUSSI FORT** : *la precision par arme ne devient pas
disponible.* Le remplissage est un **TAUX par joueur**, pas un compte ventilable, et §19 a ferme
la voie des impacts par mesure (le corps des codes 6 et 7 ne porte **aucun** identifiant d arme —
`b0 == 1` sur 168 380 observations, zero exception). **Aucune arme a projectile n entrera dans la
liste des quatre par cette voie.**

**ET IL NE REMPLACE PAS LA PORTE DE PUBLICATION DE §2.** Celle-ci compare un **TOTAL DE TIRS
DECODE** a `shots_fired` ; le remplissage estime une **PRECISION**. Ce sont deux grandeurs
differentes : `EvaluateShotsGate` continue d exiger la reference API, et §2 reste vrai mot pour
mot.

**ENFIN, §3quater CONFIRME LA RESERVE 3 DE §3bis.1** — *la restriction a l arme degrade
l estimation* : le meilleur estimateur de precision disponible aujourd hui est **par joueur**, pas
par arme, et il l est par deux routes independantes (7ter.87 (3) et le tableau ci-dessus).

---

## 4. LE COUT ET LE VOLUME — MESURES, PAS SUPPOSES

```
PORTEE 1 — passe de tirs seule (lecture + decompression + scan) :
   1.65 s CPU / film   mediane 1.42 s   p90 2.49 s   dont 66 % de lecture et decompression

   chunks DEJA decompresses par le decodeur de source :  ~0.5 s / film
   passe autonome                                     :  ~1.45 s / film
   rapporte au decodage de la source (8 a 30 s/film)  :  +2 % a +18 %

PORTEE 2 — harnais de mesure du lot tw (947 films, scan + DEUX correlations complementes
   + resolution xuid->indice bit-level + requetes DuckDB) :
   mediane 1.90 s   moyenne 2.20 s   p90 3.41 s   total 2 085 s

   Ce second chiffre N EST PAS le cout de production : il mesure l instrument, pas la passe.
```

**Consequence de branchement** : greffer la passe de tirs sur le decodeur de source (chunks deja
en memoire) coute **+2 % a +6 %**. La lancer separement coute **jusqu a +18 %** pour le meme
resultat.

**VOLUME MESURE sur les 949 films du cache** (lignes `match x indice x arme`) :

```
roster <= 16    823 films    33 300 lignes    40.5 lignes / film   4.0 armes/joueur (mediane)
roster > 16     116 films    16 032 lignes   138.2 lignes / film   5.0 armes/joueur (mediane)
```

Soit **~52 lignes par film toutes tailles confondues**, et de l ordre de **3 Mo pour 1 000
matchs** — negligeable devant les 300 Mo du shared.

---

## 5. L API — COMMENT REMPLIR LA TABLE

**AUCUN APPELANT AUJOURD HUI.** La table, sa vue, son persister et son cablage `BatchBuilder`
existent et compilent ; **personne ne les remplit**. Ce qui suit est le contrat pour le
collecteur qui viendra.

```go
pass := persist.WeaponShotsBatch{
    MatchID:    matchID,
    DecoderRev: "filmshots@2026-07-27",   // requis : sert a savoir QUELS matchs redecoder
    Players: []persist.WeaponShotsPlayer{{
        PlayerIndex: 3,                    // indice de replication BRUT, 5 bits (0..31)
        XUID:        "2533274...",         // vide = bot, ou indice non rattache au roster
        ShotsFired:  &shotsFiredAPI,       // LA REFERENCE. nil = la porte refusera
        Weapons: []persist.WeaponShotCount{
            {WeaponID: 0x2b1824d542c9679f, Shots: 118},  // BR75
            {WeaponID: 0xf408190f42c9679f, Shots: 42},   // Sidekick
        },
    }},
}

// chemin BatchBuilder (sync primaire) :
builder.SetWeaponShots(&pass)
// chemin direct (completion tardive d un film, backfill) :
err := persist.NewWeaponShotsPersister(sharedDB).PersistPass(ctx, pass)
```

`Set…` et non `Add…` : **l unite de production est le FILM ENTIER**. Concatener deux passes
produirait une ventilation qui n a jamais existe — et un doublon `(indice, arme)` que le
persister refuse.

### 5.1 Ce que le persister REFUSE, et pourquoi

| Refus | Raison |
|---|---|
| `PlayerIndex` hors `0..31` | ce ne serait pas une lecture du champ 5 bits |
| `WeaponID <= 2` | 0/1/2 sont les sentinelles grenade/melee/vehicule d `analysis` — pas des identifiants filmshell. Les ecrire fabriquerait une jointure fausse |
| `Shots <= 0` | une ligne a zero n est pas une mesure ; c est l ABSENCE de ligne qui porte le zero |
| doublon `(indice, arme)` | un `SUM` le compterait deux fois, sans rien signaler |
| `ShotsFired` negatif | ce n est pas une reference, c est un defaut amont — la porte le prendrait pour un ecart |
| `MatchID` / `DecoderRev` vide | une passe doit dire de quel match et de quel decodeur elle vient |

Le meme identifiant d arme chez DEUX joueurs differents est **normal** et accepte.

---

## 6. LE CORRECTIF D INDICE — CE QUI A CHANGE DANS `weapon_scanner.go`

`FireEvent.PlayerIndex` se lisait sur **4 bits** (`b5 >> 4`). Il en fait **5**, a
`eventStart+31`. Consequence de l ancienne lecture : **tout participant d indice >= 16 decodait
ZERO** — 323 observations, 90 722 tirs API, 106 films.

**LA PREUVE EST OFFLINE** — 949 films du cache, population EXACTE de `ScanFireEventsB5`
(marqueur 11 bits strict + filtre d arme + dedup). Elle ne doit rien a l accord avec une
reference :

```
                       roster <= 16 (823 films)          roster > 16 (116 films)
events                    1 672 653                          542 992
bit emprunte a 1          1  (0.00006 %)                   172 038  (31.68 %)
hors [0, roster-1]        1  (0.00006 %)                         0  (0 %)
valeur max                (31,5)=20  (32,4)=7              (31,5)=26  (32,4)=15
valeurs distinctes        (31,5)=9   (32,4)=8              (31,5)=25  (32,4)=16
couverture du roster      identique (0.899 moyenne)        0.917 contre 0.615 (mediane)
```

Lecture : sous 17 joueurs le bit emprunte vaut 1 **une seule fois sur 1 672 653 events** — les
deux lectures rendent donc la meme valeur, sauf sur cet unique event. Au-dela, la lecture 4 bits
**SATURE a 15** (16 valeurs distinctes, couverture 0.615) tandis que la 5 bits atteint 26 et
couvre 0.917. Le champ 4 bits est **TRONQUE**, pas ambigu.

Corroboration independante, sur une AUTRE structure du film : `weaponv3/pi_resolver.go` declare
5 bits pour le meme espace d indices.

### 6.1 CE QUE LE CORRECTIF FAIT, ET CE QU IL NE FAIT PAS — mesure sur les consommateurs

Le seul chemin de production est `sync/backfill_weapons.go` -> `analysis.CorrelateKillsGlobal`.
Rejoue sur les memes films avec les deux largeurs, kill par kill :

```
(a) AVEC L INDICE DE LA PRODUCTION (`getXuidToPI` = ORDRE DB team_id, rank)
    roster <= 16   823 films, 72 077 kills   arme identique 100.00 %   0 gagnee 0 perdue 0 changee
                                             0 / 823 films avec la moindre difference
    roster > 16    124 films, 27 579 kills   86.57 % identique
                                             1 757 gagnees / 1 082 perdues / 865 changees
                                             chemin fire_event 6 126 -> 6 121  (EN BAISSE)

(b) AVEC L INDICE JUSTE (resolution bit-level du motif xuid, weaponv3.ResolveBest)
    roster <= 16   identique a (a) : 100.00 %, aucune difference
    roster > 16    81.77 % identique
                                             4 065 gagnees / 376 perdues / 588 changees
                                             chemin fire_event 10 880 -> 14 272  (+31.2 %)
```

**CE QUI SE LIT LA-DEDANS, ET RIEN DE PLUS :**

1. **En dessous de 17 joueurs, l effet est STRICTEMENT NUL** — 72 077 kills, zero difference,
   zero film touche. Ce n est pas « negligeable », c est zero.
2. **Au-dela, la largeur juste vaut quelque chose UNIQUEMENT si l indice du tueur est juste
   aussi** : gains/pertes **10.8 pour 1** avec la resolution bit-level, contre **1.6 pour 1**
   avec l ordre DB — et sous l ordre DB, le nombre de kills attribues par fire-event BAISSE.
3. **Donc le correctif ne repare pas le grand lobby en production.** `getXuidToPI` derive
   l indice de l ORDRE DB — declare faux depuis la v3. Corriger la largeur d un champ ne repare
   pas un appariement qui ne repose pas sur ce champ.

**A NE PAS ECRIRE** : « le correctif ameliore l attribution en grand lobby ». Il rend au champ sa
largeur reelle ; le benefice mesure exige de remplacer AUSSI `getXuidToPI`. Ce remplacement
n a pas ete fait ici (hors perimetre) et il n est PAS acquis : le gain (b) est mesure sur des
kills, pas confronte a une verite terrain.

> **MISE A JOUR 2026-07-28 (RE_LOG 7ter.92, lot `co.pi` ; VERIFIE PAR 7ter.95, lot `co.ref`) — LE
> REMPLACEMENT EST FAIT, ET SON SIGNE EST `[ETABLI]` PAR DEUX INSTRUMENTS.** `getXuidToPI` est
> SUPPRIME ; le pipeline lit desormais l indice DANS LE FILM (`resolveFilmPlayerIndices`, meme
> regle que `ResolveBest`). Accord d **egalite exacte du nom d arme** avec l oracle `killsource`.
> **Chiffres de reference a citer** — ceux de la verification, mesures avec la FONCTION LIVREE sur
> **239 films 4v4 / 16 411 kills**, films a resolution incomplete COMPRIS : **ordre de la base
> 22.820 %**, **permutation AU HASARD 21.304 %**, **indice du film 77.040 %**, McNemar
> `z = 92.788`, le film gagne **237 films sur 239**. (7ter.92 publiait 22.268 / 22.658 / 76.384
> sur 116 films restreints aux films a resolution complete ; `co.ref` y retrouve **76.304 %**.)
> **LE GAIN EST LOCALISE, et le test qui le montre est par le TUEUR** : sur les kills dont le
> tueur avait DEJA le bon indice par l ordre de la base, les deux bras rendent **exactement le
> meme entier** (1 736 / 2 284, 76.007 % des deux cotes) ; sur les autres, 14.221 % -> 77.207 %.
> ⚠ **NE PAS ECRIRE << la permutation au hasard fait MIEUX que l ordre de la base >>** : le sens
> de cet ecart n est pas reproductible d un instrument a l autre. L enonce juste est *l ordre de
> la base est INDISCERNABLE d une permutation tiree au sort* — corrobore sans aucun oracle :
> il designe le bon joueur **11.769 %** du temps contre **11.137 %** attendus au pur hasard
> (949 films, 7ter.95 (3)).
> **LE +31.2 % CI-DESSUS RESTE CE QU IL ETAIT : PAS UNE PREUVE. Ne plus le citer comme telle.**
> ⚠ **La mesure ci-dessus porte sur le 4v4** : au-dela de 16 joueurs l oracle n est pas publiable
> ligne par ligne (7ter.53), donc le signe **n y est pas mesure**.
> Portee de la nouvelle lecture : en 4v4, **56.87 %** des films nomment TOUS leurs participants et
> **7.531 %** des participants n ont aucun indice dans le flux — ceux-la sortent **sans arme**
> (sentinelle `-1`), jamais avec celle d un autre.

### 6.2 LA CONTRADICTION APPARENTE AVEC `weaponv3`, RECONCILIEE PAR MESURE

`weaponv3/correlate.go` porte depuis longtemps une mesure ecrite : *« applique globalement,
SpanBefore regresse aussi les petits matchs car bit-31 n y est pas du pi »*. La mesure ci-dessus
dit l inverse. **Les deux sont vraies : elles ne portent pas sur la meme POPULATION**, et c est
mesure sur les MEMES films (60 films, mode `relax`) :

```
                                         events    bit emprunte a 1   hors roster   max
roster <= 16  STRICTE  (11 bits)         96 333     0     (0.000 %)    0 (0.000 %)    7
roster <= 16  RELACHEE (3 bits + canon)  97 882   246     (0.251 %)  612 (0.625 %)   31
roster > 16   STRICTE  (11 bits)         37 940  12 188  (32.124 %)    0 (0.000 %)   23
roster > 16   RELACHEE (3 bits + canon)  39 801  12 794  (32.145 %)  182 (0.457 %)   30
```

Le bit emprunte n est du bruit **que dans la population relachee** — et c est exactement celle
que `weaponv3` scanne par defaut (`FireRelax3 = true`). Son gating par taille de roster reste
donc justifie CHEZ LUI, et il ne dit rien de la largeur reelle du champ.

**LECON DE METHODE** : deux mesures qui se contredisent en apparence sur le meme objet se
reconcilient d abord en comparant leurs POPULATIONS. Ici, l ecart n etait pas dans le champ, il
etait dans le marqueur qui selectionne les evenements.

**`weaponv3/correlate.go` : aucun effet, et par construction** — avec son defaut
(`FireRelax3 = true`) il n appelle JAMAIS `analysis.ScanFireEventsB5`. Fige par
`TestScanFiresUS_NEmprunteJamaisLaV2ParDefaut`. Il a par ailleurs son propre layout 5 bits
(`FirePi5SpanBefore`), identique au correctif, deja actif au-dela de 16 joueurs. `weaponv3` n a
aucun appelant de production.

---

## 7. LE JOUR OU UN COLLECTEUR EXISTERA — LA LISTE A NE PAS OUBLIER

1. **`internal/ops/seed_demo.go`** (`sharedTablesWhere`) : la table n y est PAS. Tant qu elle est
   vide, c est correct. Le jour ou elle porte des donnees affichees, l oubli livrerait une demo
   dont la ventilation est vide — panne silencieuse, visible seulement a l ecran. La VUE ne se
   copie pas : la demo doit rejouer la migration (elle les joue deja).
   *(`match_kill_events` est dans le meme etat et pour la meme raison.)*
2. **`cmd/rebuild_mp/main.go`** (`dependentViews`) : si un jour un reconstructeur CTAS touche une
   table portant `match_weapon_shots_latest`, la vue doit entrer dans cette liste — sinon le swap
   la laisse pointer dans le vide. Outil `//go:build ignore`, donc AUCUN test ne le protege.
3. **L ordonnancement avec le decodeur de source** : les deux passes lisent les MEMES chunks.
   Les enchainer dans le meme process economise 66 % du cout (§4) — mais attention, les
   parametres de replication de `filmdec` sont des GLOBAUX de package.
4. **La resolution indice -> xuid** : `weaponv3.ResolveXuidToPIAllStrings` (meme regle que
   `ResolveBest` — motif xuid 8 octets LE relu en BE, cherche au bit pres, 5 bits AVANT le motif —
   mais en UNE passe par chunk : mediane 5 ms, p90 85 ms, max 160 ms sur 60 films). `getXuidToPI`
   **n existe plus** : le backfill utilise desormais `resolveFilmPlayerIndices`, qui appelle ce
   meme resolveur (7ter.92).

---

## 8. CE QUE CE GUIDE NE COUVRE PAS

- **Le collecteur** : il n existe pas. Ecrire la passe est un travail a part entiere (resolution
  indice -> xuid, lecture de `shots_fired`, ordonnancement par rapport au decodeur de source).
- **Le nommage des armes** : `metadata.weapon_labels`, hors de ce guide.
- **L arme du KILL** : autre question, autre table, autre espace d identifiants —
  `.ai/GUIDE_KILLSOURCE.md`.
- **Les lecteurs** : aucun. Le premier qui viendra devra filtrer `publishable` (§2.2).
