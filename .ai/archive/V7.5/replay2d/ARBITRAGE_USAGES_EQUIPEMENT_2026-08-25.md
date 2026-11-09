# Chantier A1 — arbitrage superviseur après la phase 0 (2026-08-25) : gate PASSÉ, verdict NÉGATIF pour les deux capacités, phases 1 et 2 non exécutées

> Lu sur pièces : `PLAN_CAPACITES_ACTIVES.md` (le contrat, phases 0/1/2 et décisions D1-D6),
> `traverse.go:810` et `ability_energy.go:69` (état réel d'`i56` avant de coder),
> `offline_biped.go:334` (largeur du champ `maskCount`), `i56_drops_test.go` et
> `i57_reach_test.go` (instruments existants, réutilisés sans les dupliquer). Mesures neuves :
> **19 décodages de film, un processus par film**, corpus de six films (`084a804d`,
> `06dfe6d9`, `00ba2e1c` famille A ; `000d5950`, `00502e52`, `07aa428d` famille B),
> **1 427 763 records delta biped** parcourus pour `i51`, six tableaux vies × rang pour `i48`.
> Commits `f91985c9d` (instrument 0.4) et `4f7662c8f` (publication et mesure d'`i51`), sur
> `wt/usages-equipement`, base `a995cf45d`, **non poussée**.
>
> **Pour le superviseur : rien ne bloque, et rien n'attend une donnée que seul l'utilisateur
> possède.** Ce document existe parce que la mesure a rendu un NON, et qu'un NON se publie
> avec ses dénominateurs. Les options de suite sont en §5, chiffrées — l'une d'elles est neuve
> et bon marché.

---

---

## 0-BIS. VERDICT FINAL DU 2026-08-25 — le propulseur est CLASSÉ à son tour : le seuil de reproductibilité est tombé

> Le superviseur a validé la Phase 0-bis **sous les quatre seuils écrits d'avance à l'étape
> 0.7, sans en renégocier aucun**, avec le contrôle de datation déclaré ÉLIMINATOIRE.
> Six films mesurés, un processus par film. **Aucun seuil n'a été renégocié, et le verdict
> est négatif.**

| seuil | exigé | mesuré | verdict |
|---|---|---|---|
| (1) volume | ≥ 150 transitions `tag==1` cumulées | **147** (dont 107 sur vies de propulseur) | non atteint, non bloquant |
| (2) reproductibilité | ≥ 6 films sur 8 tenant les 3 critères | **3 succès / 3 échecs sur 6** | **TOMBÉ** |
| (3) datation *(éliminatoire)* | transitions en cours de vie, pas au spawn | **0,0 % au spawn sur 5 films / 6** (8,3 % sur le 6ᵉ) | **TENU** |
| (4) charge utile | établir ce que `tag==1` transporte | **non mesurable par le flux** (voir plus bas) | indéterminé |

**Le détail du seuil (2), qui tranche :**

| film | fam. | masse prop. | total | % prop. (≥ 75 %) | grappin (≤ 0,10) | sans identité (≤ 0,15) | |
|---|---|---|---|---|---|---|---|
| `000d5950` | B | 38 | 43 | 88,4 % | 0,00 | 0,13 | PASSE |
| `00502e52` | B | 15 | 25 | **60,0 %** | 0,06 | **0,22** | ÉCHOUE |
| `07aa428d` | B | 36 | 52 | **69,2 %** | **0,12** | **0,28** | ÉCHOUE |
| `00ba2e1c` | A | 8 | 10 | 80,0 % | 0,00 | 0,03 | PASSE |
| `06dfe6d9` | A | 6 | 13 | **46,2 %** | 0,00 | 0,05 | ÉCHOUE |
| `084a804d` | A | 4 | 4 | 100,0 % | 0,00 | 0,00 | PASSE |

**3 échecs sur 6 films, quand le seuil n'en autorise que 2 sur 8 : hors d'atteinte
arithmétiquement.** Même si les deux films manquants passaient tous les deux, le compte
serait de 5 sur 8. **Ils n'ont donc pas été décodés** — ils ne peuvent pas changer le verdict,
et dépenser du temps machine sur un résultat déjà déterminé n'aurait rien prouvé.

**Ce que ce négatif ne dit PAS.** Le signal n'est pas une illusion : trois films tiennent le
contraste (dont un à 88,4 % avec **zéro** transition sur les vies de grappin), et le seuil
éliminatoire de datation est tenu partout — **le tag 1 date bien un événement en cours de
vie, pas une dotation au spawn.** Ce qui manque, c'est la RÉGULARITÉ.

**La limite de méthode, notée sans qu'elle serve d'excuse.** Le témoin « vies sans identité
`i48` » est structurellement gonflé : `i48` n'est transmis qu'environ une fois par vie, si
bien que **4,8 % à 60 % des vies n'ont aucune identité selon le film** (150 vies sur 250 pour
`084a804d`, 53 sur 89 pour `07aa428d`). Une vie de propulseur non identifiée tombe dans le
témoin et pénalise le contraste deux fois — et les deux films où le témoin dépasse le seuil
sont précisément les plus riches en vies anonymes. **Le seuil était posé d'avance en
connaissance de cette méthode ; le verdict tient.** Mais toute reprise devrait commencer par
améliorer le taux d'identification des vies, sans quoi elle rejouerait le même biais pour le
même résultat.

**Sur le seuil (4), une précision qui vaut pour tout le chantier.** Le test prévu — si `R(2)`
était une largeur fausse, la marche AVAL casserait — s'est révélé inapplicable : **sur les six
films, aucun record n'a de composant après `i59` dans son masque.** `i59` est toujours le
dernier composant annoncé, il n'y a rien en aval dont la casse trahirait un décalage. La
largeur du tag 1 **n'est pas falsifiable par le flux** sur ce corpus ; il reste la preuve
documentaire (`FUN_142f2679c` = `R(2)` plat, corps sur `tag==3` seulement). Dit comme tel
plutôt que déclaré « tenu ».

### Ce qui est classé, et ce que ça coûte

**Répulseur ET propulseur : classés sur mesures.** Cinq voies fermées par écrit, chacune avec
ses dénominateurs : `i27` (objets du monde), `i54` (mobilité), `i56` (énergie), `i51` (EMP),
`i59` (tags 0/2 génériques, tag 1 non reproductible).

**Conséquence produit, assumée : pas de son ni d'effet d'équipement actif pour ces deux
capacités dans le rejeu 2D — faute de CANAL, et non faute de fichier son.** L'archive sonore
de l'utilisateur n'a jamais été sollicitée et ne doit pas l'être tant qu'aucun canal ne date
les usages : un son déclenché sur un signal à 46-69 % de justesse se remarquerait
immédiatement à l'oreille, et la règle du chantier est qu'on n'affiche jamais une donnée
qu'on n'a pas mesurée.

**Ce qui reste vivant** : le camouflage, le surbouclier et le grappin, livrés, chacun sur un
canal dédié validé — la méthode fonctionne, elle a simplement épuisé ce que ces deux
capacités-là laissent voir dans le film.

---

## 0. ADDENDUM DU 2026-08-25 (soir) — la sonde `i59` a été autorisée, exécutée, et elle change le verdict pour LE PROPULSEUR

> Le superviseur a autorisé l'option A de ce document (§5) en périmètre **strictement
> instrumental** : sonde seule, aucun code de production, aucune publication, et les Phases
> 1-2 restent fermées **même en cas de positif**. Étape **0.7** ajoutée au plan, barre de
> décision écrite dans l'instrument AVANT la mesure. Trois films, un processus par film.
> Ce qui suit remplace la recommandation du §5 — le reste du document reste vrai.

**Le contrôle de validité est passé, et c'est lui qui rend le reste opposable.** Le tag 3
(grappin, canal déjà livré) a été mesuré avec les autres bien qu'on connaisse sa réponse : il
ressort à **1,08 / 1,63 / 1,14** transition par vie-lue sur les vies de grappin, contre
**0,00 / 0,05 / 0,00** sur les autres rangs. La méthode discrimine donc quand il y a quelque
chose à discriminer ; un négatif obtenu avec elle est une propriété du signal, pas un défaut
de mesure.

**Tags 0 et 2 — ce que le superviseur avait mandaté : NÉGATIF, état générique.** Portés par
**99 à 100 % des vies lues**, transitions partout à volume comparable (écart maximal entre une
cible et le témoin sans identité : **1,5×**, quand le tag 3 fait 20× et plus). C'est le défaut
exact d'`i57`. La réserve écrite dans ce document avant la mesure est confirmée : on classe,
sans renégocier le seuil.

**Tag 1 — le résultat inattendu : il discrimine LE PROPULSEUR.**

| film | RÉPULSEUR (6) | **PROPULSEUR (5/21)** | GRAPPIN (4/20) | autres rangs | sans identité | transitions |
|---|---|---|---|---|---|---|
| `00ba2e1c` | 0,03 | **0,50** | **0,00** | 0,00 | 0,03 | 10 |
| `06dfe6d9` | 0,03 | **0,32** | **0,00** | 0,02 | 0,05 | 13 |
| `000d5950` | — | **1,52** | **0,00** | 0,07 | 0,13 | 43 |

**Zéro transition sur les 76 vies de grappin cumulées** — le confondeur qui avait avalé
`i56`. **52 des 66 transitions (78,8 %) tombent sur des vies de propulseur**, alors que
celles-ci ne pèsent que 7,6 % à 25,8 % des vies lues : enrichissement **×3,4 à ×9**.

**Verdict par capacité, révisé :**

| capacité | verdict | suite |
|---|---|---|
| **Répulseur** (6) | **CLASSÉ SUR MESURES** — rien sur quatre voies (`i56`, `i51`, `i59` tags 0/2, `i59` tag 1 : 2 transitions pour 72 vies) | pas de son ni d'effet, **faute de canal** — pas faute de fichier son |
| **Propulseur** (5/21) | **PISTE POSITIVE À CONFIRMER** — `i59 tag==1` | quatre seuils ci-dessous, à valider avant toute implémentation |

**Les quatre seuils de reprise proposés** (aucun n'est tenu aujourd'hui — c'est pourquoi rien
n'est affiché) :

1. **Volume** — ≥ **150 transitions `tag==1`** cumulées (66 aujourd'hui), corpus élargi à
   **8-10 films** portant du propulseur dans les DEUX familles (rangs 5 et 21).
2. **Reproductibilité** — sur **6 films sur 8** au moins : ≥ 75 % de la masse sur les vies de
   propulseur, **≤ 0,10 par vie-lue sur le grappin**, **≤ 0,15 sur les vies sans identité**.
3. **Datation, jamais contrôlée pour ce tag** — les instants `tag==1` doivent tomber EN COURS
   DE VIE, pas à l'apparition (le contrôle qui avait qualifié `i54` : « 3 épisodes seulement à
   moins de 2 s d'un spawn »). Un tag qui ne se lèverait qu'au spawn daterait une DOTATION,
   pas un usage — et produirait un effet à chaque réapparition.
4. **Sémantique** — le corps d'`i59` n'est porté que pour `tag==3` ; on ignore ce que
   `tag==1` transporte. Établir s'il a une charge utile avant d'en faire un événement produit.

**Ce que je ne fais pas, et pourquoi** : ni Phase 1 ni Phase 2, conformément à la consigne —
un signal à 66 transitions dont on ne connaît ni la sémantique ni le comportement au spawn
n'est pas un canal livrable, c'est une piste qui mérite un lot de confirmation. **Décision
au superviseur.**

---

## 1. Le verdict, en cinq lignes *(état avant l'addendum §0 — conservé pour la traçabilité)*

| capacité | rang(s) | canal cherché | verdict | pourquoi |
|---|---|---|---|---|
| **Répulseur** | 6 (famille A) | `i56` énergie, puis `i51` EMP | **AUCUN CANAL** | ses vies portent 3,5 à 4,3 fois MOINS de chutes d'énergie que les vies de grappin du même film |
| **Propulseur** | 5 (A) / 21 (B) | idem | **AUCUN CANAL** | famille A : dominé par le grappin ; famille B : en tête, mais égalé ou dépassé par des vies SANS identité |

**Conséquence appliquée (Décision D5 du plan) : les Phases 1 et 2 ne s'exécutent pas. Zéro
ligne de production, aucun rendu, aucun son, aucun fichier téléchargé depuis l'archive.**
Le gate de la Phase 0 est néanmoins **PASSÉ** : ce qu'il demandait était « un verdict publié
par capacité, même négatif », et il l'est.

**Ce que je recommande, en une ligne** : ne pas classer tout de suite. Un canal à **3 234
lectures** (`i59`, tags 0 et 2) n'a jamais été croisé aux rangs, il a la forme de ceux qui ont
livré camo, surbouclier et grappin, et son coût d'interrogation est faible — §5, option A.

---

## 2. Ce que le plan supposait, et qui était faux

C'est le résultat le plus réutilisable de ce lot, parce qu'il ferme une question qui revenait
depuis le 2026-08-14.

L'actualisation du 2026-08-24 rouvrait le chantier sur un pari précis : le plafond d'`i56`
(176 lectures sur le film témoin) serait un **plancher** dû au décodeur, et la marche à haute
couverture — celle qui atteint 100 % des annonces sur 14 composants bipède — le ferait
tomber. **Trois mesures indépendantes disent que ce plancher n'existe pas.**

1. **`i56` était déjà branché et déjà publié.** `case "biped-spartan-ability-energy…"` →
   `consumeBipedSpartanAbilityEnergy`, `traverse.go:810-811` ; hook `SetAbilityEnergyHook`,
   `ability_energy.go:69`, posé le 2026-08-15. Un instrument utilisant déjà la marche de
   production existait (`i56_drops_test.go`). **Il n'y avait rien à brancher.**
2. **Tout ce qui est annoncé est lu.** Sur `000d5950` : `masque∋i56 176 · i56 LU 176 ·
   illisible 0` → **couverture 176/176 = 100,0 %**, contre un seuil de 80 % posé avant la
   mesure. Le seuil est atteint — et il ne mesurait pas le bon obstacle.
3. **La cause supposée pèse 0,1 %.** Les records à « masque dense » sont **201 sur 171 979**
   (`TestI57Reach`, même film), et l'instrument conclut de lui-même : « AUCUN composant ne
   casse la marche : la traversée n'est PAS le facteur limitant ».

**`i56` n'est pas mal décodé : il est rarement transmis** (0,089 % à 0,125 % des records
delta selon le film). Le journal de l'étape 3 l'écrivait déjà le 2026-08-14 (« transmis trop
rarement dans les deltas pour dater un usage ») ; ce lot le confirme avec le bon dénominateur
et referme la reprise n°1 du registre.

**Correction de vocabulaire à propager** (elle a nourri l'espoir de couverture) : « masque
dense » ne signifie PAS « plus de 7 composants ». Le champ `maskCount` de l'en-tête fait
**3 bits** (`offline_biped.go:334`) — 7 est le maximum représentable, un masque creux de 8
composants n'existe pas. Le masque dense est l'AUTRE encodage (porte à 1 → `R(64)`,
`consumeMask`), et il est marginal.

---

## 3. Les mesures, avec leurs dénominateurs

### 3.1 Identification des vies porteuses (item 0.3) — gate PASSÉ largement

`TestI48PaletteRank`, un processus par film. `i48` est lu à **100 % de ses annonces sur les
six films (759/759, 0 illisible)**.

| film | famille | records delta | i48 annoncé/lu | rang 5 PROP. | rang 6 RÉP. | rang 21 PROP. |
|---|---|---|---|---|---|---|
| `084a804d` | A | 330 981 | 143/143 | 9 vies | 14 vies | — |
| `06dfe6d9` | A | 336 212 | 230/230 | 19 vies | 28 vies | — |
| `00ba2e1c` | A | 240 645 | 206/206 | 16 vies | 31 vies | — |
| `000d5950` | B | 171 851 | 92/92 | — | — | 23 vies |
| `00502e52` | B | 182 876 | 82/82 | — | — | 8 vies |
| `07aa428d` | B | 165 198 | 56/56 | — | — | 12 vies |

Seuil demandé : ≥ 5 vies de chaque capacité sur au moins un film. **Atteint sur trois films
pour chacune.** L'élargissement au lot `d4be4ab95` prévu en repli n'a pas été nécessaire.

> Au passage, une hypothèse du plan tombe : `00ba2e1c` était pressenti comme témoin NÉGATIF
> (« n'a ni rang 6 ni rang 9 confirmés : à vérifier en 0.3 »). Il porte **31 vies de rang 6** —
> c'est le film le plus riche en répulseurs du corpus. Le témoin a donc été pris ailleurs :
> les autres rangs du même film, et le groupe **sans identité `i48`**, plus sévère.

### 3.2 Le croisement décisif (item 0.4) — chutes de charge par vie-LUE

`TestI56CrossI48Rank`, jointure PAR VIE (slot), jamais par fenêtre temporelle. Une chute =
quartet fort décroissant entre deux lectures du même emplacement de la même vie (définition
identique à celle du 2026-08-15, pour que les mesures se comparent).

| film | famille | GRAPPIN (4/20) | **PROPULSEUR (5/21)** | **RÉPULSEUR (6)** | autres rangs | TÉMOIN sans identité |
|---|---|---|---|---|---|---|
| `00ba2e1c` | A | **0,59** | 0,30 | 0,17 | 0,09 | 0,60 |
| `06dfe6d9` | A | **0,83** | 0,14 | 0,16 | 0,27 | 0,91 |
| `084a804d` | A | **1,15** | 0,27 | 0,27 | 0,69 | 0,02 |
| `000d5950` | B | 0,52 | **0,69** | — | 0,15 | 0,60 |
| `00502e52` | B | 0,53 | **1,25** | — | 0,17 | 1,38 |
| `07aa428d` | B | 1,70 | **2,42** | — | 0,30 | 0,93 |

**Répulseur** : la cible est minoritaire (0,16-0,27) là où le grappin du même film fait
0,59-1,15, et les autres rangs ne sont pas à zéro (0,69 contre 0,27 sur `084a804d`). Le
critère échoue dans les deux sens.

**Propulseur** : en famille A, dominé par le grappin sur les trois films. En famille B, il
mène son film à chaque fois (0,69 / 1,25 / 2,42) — c'est le seul résultat qui ait pu faire
hésiter — mais le grappin le suit de près (ratio 1,3 / 2,4 / 1,4) et **le témoin « vies sans
identité » l'égale ou le dépasse sur deux films sur trois** (1,38 contre 1,25 ; 0,60 contre
0,69). Un canal qui ne bat pas des vies dont on ne sait rien ne distingue rien.

**Le bar, rappelé** : le camouflage (Phase A, livré) donnait **39 transitions sur les vies du
rang cible et 0 sur 574 autres vies**. Voilà à quoi ressemble une exclusivité. Rien ici n'en
approche, et c'est pourquoi le verdict est un NON franc plutôt qu'un « à creuser ».

### 3.3 Le candidat secondaire (item 0.5) — `i51` ne voyage pas

`i51 biped-emp-timer` (R(8), minuteur quantifié 0..10 s) jetait ses bits ; il les publie
désormais (`emp_timer.go`, largeur inchangée). Mesure sur les six films :

    records delta biped cumulés   1 427 763
    masque ∋ i51                          0     sur les six films
    lectures                              0

Le composant est bien nommé dans l'archétype (`"biped-emp-timer-component"`) mais **n'est
annoncé dans aucun paquet delta**. Aucun tableau d'exclusivité n'a pu être produit : il n'y a
rien à croiser. Négatif sans zone grise.

---

## 4. Ce que la mesure établit de POSITIF

`i56` n'est pas inerte, et ses chutes ne sont pas du bruit : **elles suivent le grappin**, sur
les deux familles de palette et sur les six films. C'est cohérent avec sa sémantique
(compteur de charges, confirmée depuis le 2026-08-14 : quartet fort = charges entières) et
avec le jeu — le grappin est la capacité dont on consomme le plus de charges.

Autrement dit : **`i56` est un compteur de charges de l'équipement porté, pas un canal par
capacité.** Et la seule capacité qu'il « voit » bien a déjà son canal dédié livré
(`i59 tag==3`, grappin). Il n'y a donc rien à en tirer, même indirectement.

---

## 5. Options pour la suite — chiffrées, à trancher par le superviseur

### Option A — `i59`, les TAGS 0 et 2, JAMAIS CROISÉS *(recommandation)*

C'est la seule piste neuve, et **la vérification de sa matière a été faite avant de
l'écrire** — pour ne pas proposer une idée là où le plan exige des chiffres.

`i59 biped-spartan-ability-non-predicted-state` porte un **tag `R(2)`** — quatre valeurs.
**`tag==3` est le canal du GRAPPIN, livré et validé** (115/117 sur les vies de rang 20).
L'instrument existant compte bien tous les tags (`i59_tag3_test.go:71`) mais **ne croise aux
rangs `i48` que le tag 3** (ligne 72). Les autres n'ont jamais été interrogés.

Mesure du 2026-08-25 sur `00ba2e1c` (`TestI59Tag3Count`), à comparer aux canaux qui viennent
d'échouer :

| canal | annoncé au masque | lu | couverture | matière exploitable |
|---|---|---|---|---|
| `i56` (item 0.4) | 254 | 254 | 100 % | 254 lectures |
| `i51` (item 0.5) | **0** | 0 | — | **rien** |
| **`i59` (cette option)** | **3 274** | **3 234** | **98,8 %** | **tag 0 : 1 572 · tag 2 : 1 565** |

**`i59` porte 12,7 fois plus de lectures qu'`i56` sur le même film**, et deux de ses tags
pèsent chacun ~1 570 lectures sans avoir jamais été confrontés à un rang. C'est aussi le
patron qui a fonctionné **trois fois** : un composant d'ÉTAT dédié (`i28` camo, `i5`
surbouclier, `i59 tag==3` grappin), jamais un compteur générique.

**Coût : FAIBLE.** Tout l'outillage existe : le hook (`SetAbilityNonPredictedHook`), les
instruments (`i59_tag3_test.go`, `i59_anchor_test.go`), et le tableau d'exclusivité écrit
dans ce lot (`rank_cross_shared_test.go`) se réutilise tel quel — il suffit de compter des
tags par vie au lieu des chutes. Estimation : un instrument, six films, une demi-session.

**La réserve, dite avant la mesure comme il se doit** : les tags 0 et 2 représentent à eux
deux ~97 % des lectures. Une valeur aussi permanente peut n'être qu'un « au repos / en
cours » générique — exactement le défaut qui a tué `i57` (bit 0 à 1 sur 386/386). La
fréquence est une condition nécessaire, pas suffisante. C'est précisément ce que le
croisement par rang trancherait, pour un coût faible et avec le même bar que ce lot-ci.

*Non exécuté ici : hors périmètre de la Phase 0, qui nomme `i56` et `i51` et rien d'autre
(règle 7 du contrat d'exécution). Seule la mesure de DISPONIBILITÉ ci-dessus a été faite, et
elle relève de l'arbitrage — pas du plan. Ouvrir ce lot est une décision du superviseur.*

### Option B — CLASSER le périmètre A1 « usages répulseur/propulseur »

Quatre voies indépendantes sont désormais fermées PAR ÉCRIT : `i27` sur les objets du monde
(0/501, 2026-08-15), `i54` mobilité (par vie, 12 films, 2026-08-16), `i56` énergie (ce lot),
`i51` EMP (ce lot). Le registre notait déjà le 2026-08-18 que « la dernière voie de datation
de l'usage d'équipement par le film est fermée ».
**Coût : 0.** Ce que ça coûte vraiment : renoncer à l'objectif produit **alors qu'un canal à
3 234 lectures n'a jamais été interrogé** (option A) — c'est pourquoi ce classement n'est plus
la recommandation par défaut, seulement le repli si l'option A échoue à son tour.

### Option C — livrer la capacité PORTÉE, sans prétendre dater son usage

L'identité est solide (`i48`, 100 % de ses annonces lues, six films) et déjà affichée. On
pourrait la rendre plus lisible sur la fiche sans inventer d'instant d'usage.
**Coût : FAIBLE.** Ce n'est PAS l'objectif demandé (« montrer les usages »), c'est un lot
différent — **décision utilisateur**, pas une décision technique.

### Option D — rejouer `i54` × `i56` par fenêtre temporelle *(déconseillé)*

C'est ce que `PLAN_CLOTURE_V75.md` §A1 proposait. La Décision D1 l'écartait déjà sur la foi
du test par vie du 2026-08-16 ; ce lot ajoute une raison indépendante : **`i56` est trop rare
(0,1 %) pour peupler une fenêtre**, quelle que soit la qualité du croisement. Rejouer, ce
serait épuiser une troisième fois une piste tranchée deux fois.

### Option E — capture runtime (Cheat Engine)

Sortirait de la doctrine offline-pure du chantier. **À ne considérer que si l'utilisateur
change cette doctrine explicitement** — ce n'est pas un arbitrage technique.

---

## 6. Découvertes reçues, à porter par le superviseur

Notées, **non traitées** (règle 7 du contrat d'exécution). Elles sont aussi consignées dans la
section « Découvertes » du plan.

1. **« Masque dense » ne veut pas dire « plus de 7 composants ».** Le champ `maskCount` fait
   3 bits ; 7 est le maximum représentable, pas une limite du détecteur. Le masque dense est
   l'autre encodage (`R(64)`), et il pèse 0,1 %. Cette formulation traîne dans plusieurs
   documents où elle sert d'espoir de couverture — à corriger là où elle apparaît.
2. **`084a804d` mélange des rangs des deux familles de palette** : {1, 4, 5, 6, 8, 9, 10,
   **19**, 23, **44**}, alors que les cinq autres films du corpus sont homogènes. Le 19 est
   dans la plage famille B, le 44 hors de toute palette connue (1 lecture). Lecture fantôme ou
   vrai mélange ? Non diagnostiqué — à savoir avant toute conclusion fondée sur la palette de
   CE film.
3. **`i51` a désormais un hook et zéro donnée.** Si la durée d'immobilisation par EMP redevient
   un objectif, la donnée devra être cherchée hors du delta bipède. Le hook ne coûte rien à
   laisser en place et ne rapporte rien en l'état.
4. **`i56` est un compteur de charges qui suit le grappin** — voir §4. Utile à qui voudrait un
   jour compter des charges consommées ; inutile pour identifier QUELLE capacité.

## 7. Coût machine, et re-dérivation

**Contraintes D17 tenues, mesurées et non supposées** : 19 décodages de film, **un processus
par film**, jamais deux vivants à la fois, jamais pendant un `go build`. **Pic RSS observé :
23 MiB** — plafond de 3 072 MiB surveillé en continu (`Start-Process -PassThru` +
`WorkingSet64`, arrêt automatique au dépassement), jamais approché. Durées : 5 s
(`000d5950`, 21 Mo) à 127 s (`084a804d`, 60 Mo). **Le film-bombe `51101d1d` n'a jamais été
ouvert.**

Toutes les mesures se re-dérivent en une commande, un film à la fois (depuis `apps/go-api`) :

```bash
# couverture i56 et forme du signal (item 0.2)
CGO_ENABLED=0 I56_DROPS_FILM=<repo>/data/cache/film_chunks/000d5950 \
  go test ./internal/analysis/filmdec/ -run '^TestI56DropsAreEvents$' -v

# populations creux/dense — la cause supposée du plancher (item 0.2)
CGO_ENABLED=0 I57_FILM=<repo>/data/cache/film_chunks/000d5950 \
  go test ./internal/analysis/filmdec/ -run '^TestI57Reach$' -v

# vies par rang (item 0.3)
CGO_ENABLED=0 I48_FILM=<repo>/data/cache/film_chunks/00ba2e1c \
  go test ./internal/analysis/filmdec/ -run '^TestI48PaletteRank$' -v

# le croisement décisif (item 0.4)
CGO_ENABLED=0 I56X_FILM=<repo>/data/cache/film_chunks/00ba2e1c \
  go test ./internal/analysis/filmdec/ -run '^TestI56CrossI48Rank$' -v

# le candidat secondaire (item 0.5)
CGO_ENABLED=0 I51X_FILM=<repo>/data/cache/film_chunks/00ba2e1c \
  go test ./internal/analysis/filmdec/ -run '^TestI51CrossI48Rank$' -v

# la matiere de l'option A — distribution des tags i59 (arbitrage, pas item de plan)
CGO_ENABLED=0 I59_FILM=<repo>/data/cache/film_chunks/00ba2e1c \
  go test ./internal/analysis/filmdec/ -run '^TestI59Tag3Count$' -v
```

## 8. État de la branche

`wt/usages-equipement`, base `a995cf45d`, **non poussée**. Trois commits, **542 lignes
ajoutées dont une seule en production** : `traverse.go:495`, `br.ReadBits(8)` devient
`publishEmpTimer(br.ReadBits(8))` — même largeur lue, valeur publiée en plus, parcours de
bits identique. `SchemaVersion` reste à **18**, inchangé et identique à `origin/feat/v75`
(vérifié le 2026-08-25) : rien n'est publié au document, donc rien ne le fait bouger.

Gates verts à la clôture : `go vet ./internal/analysis/filmdec/... ./internal/analysis/replay/...`
exit 0 ; `go test ./internal/analysis/filmdec/ ./internal/analysis/replay/` → `ok` / `ok`.
