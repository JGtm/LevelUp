# MESURE — témoin de bascule de l'arme du kill (étape A0)

> Date : 2026-09-01. Volet A du plan `.ai/V7.5/PLAN_SOURCE_UNIQUE_ARME_2026-09-01.md`.
> Branche `wt/arme-source-unique`, worktree `LevelUp-wt-arme-source`.
>
> Ce fichier est le témoin de la bascule : il dit, chiffres en main, ce que le remplacement
> de la chaîne `weapon_kills` par la source de dégât change. Il n'argumente pas, il mesure.

## 1. Le seuil d'acceptation — écrit AVANT la lecture des résultats

Cette section a été écrite et enregistrée **avant** le premier lancement de la sonde
(commit du seuil antérieur au commit des résultats, cf. `git log` de la branche). C'est la
condition qui rend la mesure honnête : un seuil rédigé après coup se plie toujours au
résultat obtenu.

> **La bascule est ACCEPTÉE si, et seulement si :**
>
> 1. le résidu « Non attribué » agrégé **DIMINUE** — en valeur absolue et en part du total
>    des frags API du même lot de matchs ;
> 2. **aucune classe d'arme à feu** (`shoulder`, `sidearm`, `heavy`) ne perd plus de **2 %**
>    de ses frags **sans explication nommée** — une explication nommée étant un mécanisme
>    identifié et écrit dans ce fichier (par exemple : « ces frags migrent vers la classe X
>    parce que l'ancienne chaîne recollait sur l'arme tenue les morts d'une arme sans tir »).
>
> Si l'une des deux conditions tombe, la bascule n'est pas acceptée en l'état et l'étape A1
> ne démarre pas avant que l'écart ait reçu une explication écrite.

Corollaires qui ne sont **pas** des critères d'acceptation, mais que la mesure doit tout de
même publier (décision D13 du plan) :

- la concordance entre ce que le kill feed du rejeu sait nommer et ce que le graphe sait
  classer (A0.4) ;
- l'écart de nom entre les deux dictionnaires (A0.5).

## 2. La commande de reproduction, telle quelle

La sonde vit dans `apps/go-api/internal/platform/duckdb/temoin_bascule_arme_probe_test.go`
(+ `temoin_bascule_concordance_test.go`). Build normal, **sautée** sans variable
d'environnement — motif des sondes du dépôt.

Les bases sont ouvertes en **lecture seule**, et sur une **COPIE** : le modèle mono-process
de DuckDB interdit d'ouvrir un fichier qu'un autre process tient en écriture.

```bash
# 1. copier les bases de production locales (aucun serveur ne doit tourner)
cp data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb /tmp/hinf_shared.duckdb
cp data/titles/halo_infinite/warehouse/metadata.duckdb          /tmp/hinf_meta.duckdb

# 2. lancer la sonde
export PATH="/c/msys64/ucrt64/bin:$PATH" CGO_ENABLED=1 CC=/c/msys64/ucrt64/bin/gcc.exe
cd apps/go-api
TEMOIN_ARME_SHARED=/tmp/hinf_shared.duckdb \
TEMOIN_ARME_META=/tmp/hinf_meta.duckdb \
TEMOIN_ARME_MATCHS=200 \
TEMOIN_ARME_SORTIE=../../.ai/V7.5/temoin_arme_sortie.md \
go test ./internal/platform/duckdb/ -run TemoinBasculeArme -v -count=1
```

## 3. Résultats

Lancement du 2026-09-01, sur les **200 matchs les plus récents** portant au moins une
ligne dans l'une des deux chaînes. Population : 198/200 ont des lignes `weapon_kills`,
200/200 ont une source de dégât mesurée — les deux chaînes voient donc bien le **même**
lot, la comparaison n'avantage personne.

## Perimetre mesure

| grandeur | valeur |
|---|---:|
| matchs | 200 |
| joueurs credites | 1052 |
| frags API (total) | 18752 |
| melee API | 1717 |
| grenade API | 986 |

## Ventilation par classe — ancienne chaine contre nouvelle

| classe | weapon_kills (servi aujourd'hui) | source de degat | ecart |
|---|---:|---:|---:|
| environmental | 171 | 171 | +0 |
| grenade | 986 | 986 | +0 |
| heavy | 304 | 4118 | +3814 |
| melee | 1717 | 1717 | +0 |
| shoulder | 703 | 5244 | +4541 |
| sidearm | 418 | 2532 | +2114 |
| unattributed | 14453 | 3984 | -10469 |
| **total frags API** | 18752 | 18752 | |

Residu « Non attribue » : ancienne 14453 (77.1 %), nouvelle 3984 (21.2 %).

## Couverture de l'ancienne chaine sur le lot

| grandeur | valeur |
|---|---:|
| lignes `weapon_kills` | 2795 |
| matchs portant au moins une ligne | 198 / 200 |
| lignes hors sentinelles (0/1/2) | 1736 |
| lignes sans identifiant d'arme | 1059 |
| lignes par match | 14.0 |
| lignes rapportees au total de frags API | 14.9 % |

## A0.4 — concordance graphe / kill feed (D13)

| grandeur | tags | morts |
|---|---:|---:|
| tags observes en base | 135 | 17433 |
| (a) obtiennent une IMAGE | 110 | |
| (b) obtiennent une CLE | 84 | |
| (c) image SANS cle | 28 | 1786 |
| (c) cle SANS image | 2 | 87 |
| ni image ni cle | 23 | 215 |

### Tags rendant une IMAGE sans CLE — le rejeu les nomme, le graphe non

| tag | morts | classe damagetag | nom kill feed | image | cle |
|---|---:|---|---|---|---|
| 0xdaa03c35 | 859 | MELEE |  | killfeed-65 |  |
| 0x2ff92041 | 522 | MELEE |  | killfeed-65 |  |
| 0x15dcdfe3 | 73 | ARME | Mutilator | killfeed-81 |  |
| 0xdeffdc6b | 48 | VEHICULE |  | killfeed-33 |  |
| 0xf712c64a | 43 | VEHICULE |  | killfeed-36 |  |
| 0xb258262f | 34 | ARME | Mutilator | killfeed-81 |  |
| 0xd21ac495 | 30 | MELEE |  | killfeed-65 |  |
| 0x119861b4 | 29 | GRENADE |  | killfeed-49 |  |
| 0x64ab85c4 | 25 | MELEE |  | killfeed-65 |  |
| 0xee2d686d | 22 | GRENADE |  | killfeed-49 |  |
| 0xc8681ecf | 19 | GRENADE |  | killfeed-49 |  |
| 0x382cafaf | 15 | VEHICULE |  | killfeed-05 |  |
| 0x7c4f8a28 | 11 | VEHICULE |  | killfeed-33 |  |
| 0x53b1e6c5 | 9 | VEHICULE |  | killfeed-35 |  |
| 0x01bc8b0b | 9 | ARME | Mutilator | killfeed-81 |  |
| 0xab06203c | 7 | VEHICULE |  | killfeed-32 |  |
| 0xfa4fad21 | 5 | VEHICULE |  | killfeed-35 |  |
| 0x28907150 | 5 | VEHICULE |  | killfeed-30 |  |
| 0x3c7560e0 | 4 | VEHICULE |  | killfeed-21 |  |
| 0x000027cc | 3 | MELEE |  | killfeed-65 |  |
| 0x72230737 | 3 | VEHICULE |  | killfeed-37 |  |
| 0x99d98aef | 3 | VEHICULE |  | killfeed-32 |  |
| 0x5a4450e4 | 3 | VEHICULE |  | killfeed-27 |  |
| 0xa21bf18a | 1 | VEHICULE |  | killfeed-35 |  |
| 0x1538c906 | 1 | GRENADE |  | killfeed-49 |  |
| 0x00015cd1 | 1 | VEHICULE |  | killfeed-05 |  |
| 0x77a61ef5 | 1 | VEHICULE |  | killfeed-86 |  |
| 0xd8cea1aa | 1 | MELEE |  | killfeed-65 |  |

### Tags rendant une CLE sans IMAGE

| tag | morts | classe damagetag | nom kill feed | image | cle |
|---|---:|---|---|---|---|
| 0x00403594 | 83 | DEGAT_GLOBAL |  |  | hinf_environment |
| 0x557a2b6c | 4 | DEGAT_GLOBAL |  |  | hinf_environment |

### Synthese par classe des tags « image sans cle »

| classe damagetag | tags | morts | lecture |
|---|---:|---:|---|
| MELEE | 6 | 1440 | CHOIX — servie par le compteur API `melee_kills` (D4) |
| VEHICULE | 15 | 159 | TROU — le rejeu affiche l'engin, le graphe dirait « Non attribue » (D13) |
| ARME | 3 | 116 | TROU — arme nommee par le film, absente du registre (D13) |
| GRENADE | 4 | 71 | CHOIX — servie par le compteur API `grenade_kills` (D4) |

## A0.5 — ecart de NOM entre le kill feed et le registre (D13)

| cle de registre | nom kill feed (damagetag) | nom registre (weapons.name) | morts |
|---|---|---|---:|
| hinf_sidekick | Mk51 Sidekick | Mk50 Sidekick | 1562 |
| hinf_bandit | Bandit Evo / M392 Bandit | M392 Bandit | 231 |
| hinf_shock_rifle | Shock Rifle / Shock Rifle (Ranked) | Shock Rifle | 12 |
| hinf_energy_sword | Energy Sword / Infected Energy Sword | Energy Sword | 6 |

## Volume ecarte par la decision D4 (melee et grenade servies par l'API)

| cle de registre (classe) | morts vues par la source de degat |
|---|---:|
| hinf_gravity_hammer (melee) | 1832 |
| hinf_energy_sword (melee) | 682 |
| hinf_frag_grenade (grenade) | 357 |
| hinf_dynamo_grenade (grenade) | 274 |
| hinf_plasma_grenade (grenade) | 222 |

## Arbitrage de la melee — la decision D4 tient-elle pour l'epee et le marteau ?

| population | morts |
|---|---:|
| compteur API `melee_kills` (autoritatif) | 1717 |
| melee NUE vue par la source (classe MELEE, aucune cle) | 1440 |
| melee D'ARME vue par la source (cle de classe registre `melee`) | 2514 |
| dont hinf_gravity_hammer | 1832 |
| dont hinf_energy_sword | 682 |

Ecart `melee_kills` moins melee NUE : +277. Ecart `melee_kills` moins (nue + arme) : -2237.


## 4. Verdict au regard du seuil de la section 1

**Condition 1 — le résidu « Non attribué » diminue.** TENUE, et largement :
**14 453 (77,1 %) → 3 984 (21,2 %)**, soit **10 469 frags** qui quittent le résidu. La
bascule ne déplace pas des frags d'une classe à l'autre : elle en nomme dix mille que
l'ancienne chaîne laissait anonymes.

**Condition 2 — aucune classe d'arme à feu ne perd plus de 2 % sans explication nommée.**
TENUE : **aucune classe d'arme à feu ne perd quoi que ce soit.** Les trois gagnent —
`shoulder` +4 541, `heavy` +3 814, `sidearm` +2 114. Les classes servies par les compteurs
API (`melee`, `grenade`) et la classe `environmental` sont rigoureusement identiques des
deux côtés, ce qui est le contrôle attendu : elles ne dépendent pas de la chaîne qu'on
remplace.

**La bascule est ACCEPTÉE.** L'étape A1 peut démarrer.

### Pourquoi l'écart est si grand ici, alors que le plan annonçait 18,7 %

Les deux nombres ne mesurent pas la même chose et il ne faut pas les confondre. Le 18,7 %
de la section 1.2 du plan est la part des **lignes `weapon_kills`** qui ne résolvent pas
vers une arme. Le 77,1 % ci-dessus est la part des **frags de l'API** que le sunburst
laisse dans « Non attribué » — dénominateur bien plus grand.

La couverture mesurée l'explique : sur ce lot, `weapon_kills` ne porte que **2 795 lignes
pour 18 752 frags API (14,9 %)**, soit 14 lignes par match, quand la moyenne du corpus est
de 82. Dont **1 059 lignes sans identifiant d'arme**. Sur les matchs récents, l'ancienne
chaîne n'échoue pas à corréler : elle n'a presque rien à corréler. Le constat est cohérent
avec celui qui fonde le volet B (l'étape post-sync ne rattrape pas les films publiés en
retard).

### Ce que la mesure a fait apparaître, et qui n'était pas prévu

**La décision D4 ne peut pas s'appliquer telle quelle à l'épée et au marteau.** Le compteur
API `melee_kills` vaut 1 717 sur le lot. La mêlée NUE vue par le film (les 6 tags de classe
`MELEE`, aucun d'eux ne portant de clé de registre) vaut 1 440 — écart de +277, cohérent
avec la couverture du film (17 433 morts mesurées pour 18 752 frags API, soit 93 %). En
revanche la mêlée **d'ARME** (clés de classe registre `melee` : marteau 1 832, épée 682)
vaut **2 514** : l'ajouter porterait le total à 3 954, soit **2 237 de plus que le compteur
API**.

Conclusion mesurée : **`melee_kills` de l'API n'inclut ni le marteau à gravité ni l'épée à
énergie.** Ce sont, pour l'API, des frags d'arme. Or le registre les classe `melee` — la
classe conflate « arme de corps à corps » et « mécanique de corps à corps ».

Conséquence sous D4 + D4bis appliqués littéralement : le total de la classe Mêlée reste le
compteur API (1 717), la classe reste une feuille sur Infinite, et les **2 514 frags du
marteau et de l'épée restent dans « Non attribué »**. Ce n'est **pas une régression** — ils
y sont déjà aujourd'hui (l'ancienne chaîne n'en voit que 131 + 191 sur tout le corpus, et
`buildRegistryFragClasses` écarte de toute façon la classe `melee`) — mais c'est un gain
laissé sur la table, et la décision qui le laisse repose sur une mesure (D4bis) qui porte
sur les **6 tags génériques** de mêlée, pas sur ces deux armes nommées. Inscrit en
section 6 du plan ; **non traité** dans ce lot.

### Concordance graphe / kill feed (D13)

Sur les 135 tags observés : 110 obtiennent une image, 84 une clé, **28 une image sans clé
(1 786 morts)** et 2 une clé sans image (87 morts, chute/environnement — attendu, une icône
fausse serait pire qu'aucune). Ventilé par classe, l'écart se lit en deux moitiés :

- **un CHOIX** — `MELEE` (6 tags, 1 440 morts) et `GRENADE` (4 tags, 71 morts) : servies par
  les compteurs API, elles n'ont pas besoin de clé pour être comptées ;
- **un TROU** — `VEHICULE` (15 tags, 159 morts sur ce lot ; 1 441 sur le corpus) et `ARME`
  (3 tags, 116 morts : le « Mutilator », nommé par le film, absent du registre).

Le trou véhicules/tourelles est traité par l'**étape A6** (décision D14). Le « Mutilator »
reste une exception assumée de l'allowlist A1.9.

### Écart de nom (A0.5)

Quatre clés portent un nom différent selon qu'on interroge le film ou le registre. Une seule
pèse : `hinf_sidekick`, **1 562 morts**, « Mk51 Sidekick » côté film contre « Mk50 Sidekick »
côté registre. Les trois autres sont des libellés du film qui énumèrent des variantes
(`Bandit Evo / M392 Bandit`, `Shock Rifle / Shock Rifle (Ranked)`,
`Energy Sword / Infected Energy Sword`) là où le registre porte le nom canonique — ce n'est
pas une divergence de fond, c'est la forme d'une clé de règle. Aucune correction dans ce
lot : le lecteur voit le libellé du registre partout où le graphe s'affiche.
