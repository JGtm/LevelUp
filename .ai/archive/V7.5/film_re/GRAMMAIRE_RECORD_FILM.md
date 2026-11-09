# Grammaire d'un enregistrement de film Halo Infinite — référence

> Établi le 2026-07-26. **Ce document fait foi** sur l'en-tête d'un record et sur les pièges
> qui ont coûté du temps. Toute lecture divergente doit être arbitrée contre lui, ou le corriger.
> Deux méthodes indépendantes le fondent : le désassemblage, et une vérité terrain de 807 855
> curseurs de bits réels (`.ai/V7.5/dumps/ce_capture_delta.csv`).

---

## 1. L'en-tête — LA RÈGLE

```
RECORD DELTA = [1 préfixe] [idLow] [2 tag] [1 gate] [3 maskCount] [6 × maskCount index]
                                                   └── FUN_1406d7610 ──────────────────┘
             = 7 + idLow + 6·n bits

branche DENSE (> 7 composants) :
             = [1 préfixe] [idLow] [2 tag] [1 gate=1] [64 masque]
             = 68 + idLow bits
```

**IL N'Y A PAS DE BIT `maskSel`.** `FUN_1406d7610` **retourne elle-même sa longueur en bits** —
`4`, `6·count+4`, ou `0x41` selon la branche. C'est de l'arithmétique écrite dans le code, pas une
inférence : il n'y a aucune place pour un cinquième bit de préambule. `consumeMask`
(`traverse.go`) est un portage fidèle.

## 1 bis. L'AMORCE DE PAQUET — la faute qui a coûté le plus cher (2026-07-26)

**Avant le premier record d'un paquet type-0, le flux porte une amorce de 2 bits.** Le décodeur
séquentiel ne la consommait pas. Conséquence : tout le walk était décalé de 2 bits **dès le
premier record de chaque frame**, sur 100 % des frames.

```
FUN_14298816c   memcpy(buffer, flux, *(hdr+4))          <- payload exact (confirme WalkPackets)
                FUN_1406d5cc0(reader, 3)                <- reader positionné au bit 0 du payload
FUN_142987460   DAT_144706104 = FUN_1406cf008(reader)   <- R(1) AVANT tout record
                puis vtable[0x40] = FUN_1406cd128       <- la boucle de records
FUN_1406cf008   « *(p+0x2c) += 1 »                      <- un seul bit
```

**Témoin, invariant par carte et par film.** Le compteur de composants tient sur 3 bits : un
record épars porte donc **au plus 7 composants**. Vérité terrain (138 390 records bipèdes) :
**99,86 %** dans cette plage, mode à 4.

| configuration | part de masques 1..7 sur DELTA de slots bindés |
|---|---|
| amorce 0, idLow 11 (l'ancien code) | **14,65 %** — *sous le niveau du hasard* |
| **amorce 2, idLow 11** | **84,81 %** — retenu |
| amorce 1, idLow 12 | 51,67 % |
| amorce 2, idLow 10 | 5,24 % |
| niveau du hasard mesuré | 10,67 % |
| vérité terrain | 99,86 % |

**Ce que ça explique.** Nous « atteignions » `i22`/`i47`/`i48`/`i56` à 100 % parce que notre
masque était un **masque dense tiré du hasard**. Excès de présence par rapport à la vérité :
`i22` ×209, `i47` ×331, `i48` ×356, `i56` ×642. Après correction : ×63, ×66, ×86, ×69.

**Le second bit n'est PAS localisé au désassemblage** — un seul `R(1)` y est établi. Le second
est retenu par la mesure, et cette distinction doit rester visible : c'est une question ouverte,
pas un fait acquis.

**NÉCESSAIRE, PAS SUFFISANT.** Après correction, `i22` lit encore **92,46 %** de comptes
impossibles. Il reste au moins une faute dans le **corps** des records.

> **Leçon de méthode, la plus chère de ce chantier.** Le §4 de ce document mesurait déjà
> « amorce de paquet 2 bits, en-tête 21 » — depuis sa rédaction. **Le constat n'a jamais été
> porté dans le décodeur.** Une mesure juste, écrite dans un document qui « fait foi », n'a
> servi à rien pendant des semaines. Vérifier qu'une conclusion documentée est *effectivement
> câblée* fait partie de la conclusion.

## 2. `idLow` EST UNE VALEUR DE RUNTIME — le piège central

`idLow` sort de `FUN_1406d310c(DAT_1451f98d4[7*2])`, une table peuplée **au chargement de la
carte**. Elle **diffère donc d'un film à l'autre** :

| film | idLow | en-tête épars | en-tête dense |
|---|---|---|---|
| `000d5950` (Cliffhanger) | **11** | 18 bits | 79 bits |
| film de la capture live | **14** | 21 bits | 82 bits |

**Conséquence pour tout décodeur** : une largeur d'en-tête codée en dur ne peut pas être
universelle. Le `21` historique se décompose `1 + idLow(14) + 2 + 1 + 3` — et le
« bit en trop » par rapport à une lecture à 13 bits est le **quatorzième bit du champ
d'identifiant**, pas un drapeau de masque.

Le fenêtrage à 13 bits d'`offline_biped.go` est un **motif de reconnaissance** qui fonctionne
parce que les slots joueur vivent dans une plage étroite — ce n'est **pas** le champ `id`.

## 3. Chaîne d'appel, pour exclure « le bit est en amont »

```
FUN_1406cd128   boucle de records : [R(32) si mode film] + code préfixe R(1) ; si 0 → R(2)
FUN_1406d3140   codec d'ID : R(idLowBits) + R(2) tag ;  id = tag<<30 | (base + low)
FUN_141f86b58   DELTA : appelle le masque SANS lire un bit intercalé
FUN_14076cb60   lit le masque en toute première instruction
```

Le garde `R(8)` du mode film n'existe **que** pour NEW (type 1) et DEL (type 2), **jamais** pour
DELTA (type 3).

## 4. Vérification empirique (reproductible)

Vérité terrain : `.ai/V7.5/dumps/ce_capture_delta.csv` — `eid, typeIndex, compIndex, param4,
bitCursor`, soit le curseur exact du désérialiseur de **chaque** composant.

- En-tête entre records : **21 bits sur 93,89 %** des 177 660 paires.
- Confirmé **séparément pour chaque `count`** de 1 à 7 (82,8 % à 98,2 %) — sept confirmations.
- Premier record d'un paquet : `curseur(1er comp) − 6·count = 23` ⇒ amorce de paquet 2 bits,
  en-tête 21.
- Branche dense, chemin **disjoint** du compteur et des index : `82 = 68 + 14`. Même réponse.

## 5. Ce qui a été RÉFUTÉ, pour ne pas y revenir

**L'ajout d'un bit `maskSel`** régresse tous les archétypes, sans exception :

| archétype | clean avant → après |
|---|---|
| `ti=35` bipède | 23 485 → 20 351 (−13,3 %) |
| `ti=11` objectif | 17 → 5 (−70,6 %) |
| `ti=5` joueur | 103 → 92 |
| `ti=41` | 195 → 160 |

Désynchronisations sur les slots joueur : 876 → 4 027 (**×4,6**). Le conditionner au seul bipède
serait tout aussi faux, puisque le bipède régresse aussi.

## 6. Deux distinctions à garder en tête

**Atteint ≠ juste.** `i22`, `i47`, `i48`, `i56` sont atteints à **100 %** par le walk séquentiel,
et pourtant leur contenu est **faux** : **91,19 %** des comptes de grenades lus sont
*physiquement impossibles* — on lit des valeurs jusqu'à **255**. Un critère **absolu** — une borne
du jeu — réfute plus vite que n'importe quelle statistique.

> **Ne pas écrire « c'est du bruit »** (correction utilisateur, 2026-07-26). Le film ne
> contient pas de bruit : il contient des champs que nous ne savons pas encore situer. Dire
> « bruit » attribue le défaut au flux et **ferme la question** ; dire « notre curseur dérive
> en amont » l'attribue au décodeur et la garde ouverte. La différence n'est pas rhétorique :
> c'est elle qui a fait chercher — et trouver — la voie du marqueur (`grenade_events.go`,
> 70 lancers, 8/8 joueurs) alors que la voie des composants était déclarée sans espoir.

### Bornes de jeu utilisables comme critères absolus (source : utilisateur, 2026-07-26)

| règle | conséquence pour le décodage |
|---|---|
| **2 types de grenade au plus**, **2 unités de chaque** | `i22` : `count ≤ 2` et chaque valeur `≤ 2`. Le critère précédemment écrit ici (« 4 types, 4 unités ») était **trop permissif** — la réfutation en est renforcée, pas affaiblie. **Seule borne vraiment intemporelle de ce tableau** : elle tient à tout instant du match, pas seulement au spawn. |
| Slayer : spawn en grenades à fragmentation, **aucune capacité Spartan** | Vaut **AU SPAWN UNIQUEMENT** — voir l'avertissement ci-dessous. |
| **Super Fiesta : aléatoire**, mais **pool réduit** | Variété attendue, mais bornée. `000d5950` est un Super Fiesta : la variété y est normale, les valeurs > 2 non. |
| Super Fiesta : capacités possiblement **boostées** | Les charges d'`i56` peuvent dépasser le nominal sur ce mode — ne pas en faire un critère de rejet. |

### AVERTISSEMENT — le ramassage au sol détruit les témoins « de bout en bout »

Une version antérieure de ce tableau affirmait que sur un Slayer, `i48` devait être « **vide
de bout en bout** » et `i47` « quasi constant », et présentait cela comme le **témoin
catégorique** le plus fort disponible. **C'est FAUX** (correction utilisateur, 2026-07-26) :

> Les créateurs de carte posent au sol des grenades de tous types **et** des capacités
> Spartan, que n'importe quel joueur ramasse en cours de partie.

Donc sur un Slayer, voir du Plasma dans `i47` ou une capacité dans `i48` **n'est pas une
anomalie** : c'est un ramassage. Ces deux règles ne contraignent que l'**instant du spawn**,
et un décodeur qui ne sait pas encore situer les spawns ne peut pas s'en servir. Le seul
critère qui reste utilisable sans rien savoir du temps est la **borne de quantité** (≤ 2).

**Leçon de méthode** : un critère « catégorique » qui repose sur une règle de mode doit être
vérifié contre les mécaniques qui l'assouplissent (ramassage, variantes de mode, boost)
AVANT d'être promu témoin décisif. Ici, la promotion était prématurée.

**Défaut d'écriture ≠ valeur lue.** Pour `i56`, 72,77 % de charges « pleines » sont un artefact :
le désérialiseur écrit `0x7F` **sans consommer un bit** quand le bit de masque est à 0. Toujours
séparer « ce que le déser a écrit » de « ce que le flux contenait ».

## 7. Le résidu de désynchronisation n'est pas le masque

Sur `000d5950`, il reste 876 désynchronisations de slots joueur. **792** tombent sur
`i00 game-engine-team-mapping-component` — des records dont le slot 512..519 est lié à un
archétype **qui n'est pas le bipède** : de l'**aliasing de slot**. L'archétype `ti=35` lui-même
est à **23 485 clean / 17 desync = 99,93 %**.

*(Réserve : cette explication de remplacement a été jugée insuffisamment étayée par l'audit
adversarial du 2026-07-26. À confirmer avant de s'en servir comme prémisse.)*
