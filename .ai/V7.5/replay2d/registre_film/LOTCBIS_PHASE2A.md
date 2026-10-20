# Lot C-bis — phase 2a : la MESURE dans `replay`. Le slot est la zone, et le tag 4 est le proprietaire

> Perimetre : CB.2a.1, CB.2a.2, CB.2a.3 + Gate 2a de `LOTCBIS_ARBITRAGE_PHASE1.md`.
> Acquis repris sans les refaire : `LOTCBIS_PHASE0.md` (grammaire), `LOTCBIS_PHASE1.md` (port,
> tags, slots, chiffres). Mesures du 2026-08-18, branche `wt/zones-ti13-p2`, base `30b8ed311`.
> Gates : `LOTCBIS_p2a_gates.log`. Sorties : `lotCbis/<short8>_p2a.tsv`, journaux bruts
> `lotCbis_p2a_<short8>.log`. **MESURE SEULEMENT** : aucun fichier de production touche, aucun
> champ de document, aucun schema. La forme de `zoneStates` est PROPOSEE en section 7, pas ecrite.

## 0. Le resultat, en une phrase

**Un slot `ti=13` n'est pas une zone : c'est UNE PROPRIETE RESEAU NOMMEE.** Trois familles de
proprietes coexistent sur une carte de Strongholds, et la phase 2a les a appariees aux zones du
catalogue : la JAUGE (tag 3, une par zone, carte slot -> zone coherente a 93-98 % et IDENTIQUE sur
les deux matchs), le PROPRIETAIRE (tag 4, une par zone, valeur = index d'equipe du capteur a
100 % / 91,1 % hors emissions neutres), et un troisieme canal constamment neutre. **Gate 2a :
TENU**, par les deux voies a la fois.

## 1. Ce que l'instrument fait, et ce que sa recopie coute

`filmdec` ne peut pas importer `replay` (cycle) et l'ancrage (`matchWorldObjectRecord`) comme le
rejeu (`consumeByName`) n'y sont pas exportes. La phase 2a recopie donc, dans des fichiers de
TEST du paquet `replay`, le strict necessaire : l'en-tete de record d'objet du monde, la table de
largeurs de `ti=13` (i0 = R(32), i1 = variant mode A, i2..i33 = mode B), et la bande de slots vue
en image-cle. **Aucun fichier de production n'a ete ajoute, deplace ou exporte.**

**La recopie est controlee par le flux, pas par la confiance.** Deux gardes tournent a chaque
passe :

| garde | ce qu'elle interdit |
|---|---|
| REGISTRE | l'archetype 13 doit declarer les noms attendus aux index attendus, sinon `t.Fatal` (le lot 0 a mesure DEUX decoupages de registre selon le build) |
| CHAINAGE | la position de fin calculee doit porter un en-tete de record valide — definition de la phase 0 (`ti13HeaderAt`), structurelle et sans contrainte de bande |

| film | mode | records ancres | rejoues | **chainage SCALAIRE** | chainage PAR JOUEUR | temoin +3 bits |
|---|---|---|---|---|---|---|
| `7344d24f` | Strongholds | 36 082 | 10 841 | **3 662/3 880 = 94,4 %** | 40/6 961 = 0,6 % | 38/10 841 = 0,4 % |
| `696a9d7c` | Strongholds | 36 677 | 8 558 | **3 708/3 823 = 97,0 %** | 69/4 735 = 1,5 % | 46/8 558 = 0,5 % |
| `01e1f945` | KOTH | 6 845 | 5 226 | **3 681/3 798 = 96,9 %** | 474/1 428 = 33,2 % | 35/5 226 = 0,7 % |
| `8076f97f` | KOTH | 4 573 | 2 898 | **1 278/1 370 = 93,3 %** | 396/1 528 = 25,9 % | 44/2 898 = 1,5 % |
| `606d9844` | KOTH | 1 335 | 969 | **349/371 = 94,1 %** | 240/598 = 40,1 % | 6/969 = 0,6 % |
| `0a247154` | KOTH | 2 928 | 1 962 | 223/304 = 73,4 % | 833/1 658 = 50,2 % | 23/1 962 = 1,2 % |
| `000d5950` | Slayer (temoin) | 661 | 267 | **2/14 = 14,3 %** | 7/253 = 2,8 % | 11/267 = 4,1 % |

**93,3 a 97,0 % de chainage scalaire contre 0,4 a 1,5 % de temoin** : la recopie lit les memes
bits que la production (fourchette de la phase 0 : 87,0 a 99,3 %). Et le chainage decompose dit
quelque chose que le taux global cachait : **en Strongholds le trafic APPARENT des composants par
joueur est de la contamination d'ancrage** (0,6-1,5 %, au niveau du temoin), alors qu'en KOTH il
chaine reellement (25,9-40,1 %) — les deux lectures independantes de la phase 1 se confirment ici
par une troisieme voie. **Le temoin Slayer se tait** : 20 valeurs scalaires, 14,3 % de chainage,
aucune rampe.

## 2. La STRUCTURE de `ti=13`, publiee pour la premiere fois (i0)

La phase 1 consommait i0 sans le lire. La phase 2a le publie, et l'archetype se lit alors sans
mystere : **un slot = une propriete** (nom + valeur scalaire + 32 valeurs par joueur). Sur
`7344d24f`, 24 slots emettent une valeur scalaire ; leur tag dominant les range en familles :

| slots | tag dominant | volume | ce qu'ils portent |
|---|---|---|---|
| 1532, 1537, 1542, 1545, 1547 | **3** (quantifie) | 296 a 1 146 emissions | **la JAUGE de capture** |
| 1530, 1535, 1540, 1546 | **4** (R32) | 6 a 16 emissions | **le PROPRIETAIRE** (0, 1) |
| 1531, 1536, 1541 | **4** (R32) | 32 a 39 emissions | un canal constamment NEUTRE (`0xFFFFFFFF` dominant) |
| 1520-1528, 1533, 1534, 1538, 1539 | 8, 0, 1, 6, 7 | 1 a 7 emissions | proprietes muettes ou hors sujet |

**C'est la reponse au « non mesurable » de la premiere passe** : les slots bavards en tag 4 ne
sont PAS ceux que la jauge designe (10 slots portent le tag 4, 2 seulement portent aussi une
zone du tag 3 — et ces deux-la n'ont qu'une valeur, donc zero changement). Les deux canaux
parlent d'objets differents, et il fallait le mesurer avant de conclure quoi que ce soit.

## 3. CB.2a.1 — le slot de la jauge EST la zone

Oracle : les captures/securisations nommees du statborg (`NamedEvents` + `SlotIdentity` sur le
roster gele), posees sur l'axe du rejeu apres retrait de l'origine publiee (`OriginMs` : 33,3 s et
33,6 s). Position du capteur : `AttributeZones` sur les trajectoires decodees. Cote film : le slot
dont une rampe du tag 3 culmine dans [t-2 s ; t+2 s].

### La courbe de tolerance, publiee avec son temoin (contrat de `AttributeOptions.MaxDistanceM`)

| tolerance | `7344d24f` attribuees | temoin zones a 12 m | `696a9d7c` attribuees | temoin |
|---|---|---|---|---|
| 0 m (strict) | 52/71 = 73,2 % | 11,3 % | — | — |
| 2 m | 57/71 = 80,3 % | 16,9 % | — | — |
| **5 m (verdict)** | **59/71 = 83,1 %** | **26,8 %** | **66/77 = 85,7 %** | **26,0 %** |
| 10 m | 62/71 = 87,3 % | 40,8 % | — | — |

Le seuil du verdict (5 m) etait ecrit AVANT la mesure, avec sa justification. Note utile : le
taux STRICT vaut ici 73,2 %, tres au-dessus des ~10 % que le commentaire d'`AttributeZones`
annonce pour le corpus general — les captures de zone se font bien DANS la zone.

### La table slot -> zone, et sa STABILITE

| slot | `7344d24f` | `696a9d7c` | verdict |
|---|---|---|---|
| 1532 | zone rang **1** (19/22 votes) | zone rang **1** (21/21) | identique |
| 1537 | zone rang **2** (18/18) | zone rang **2** (23/23) | identique |
| 1542 | zone rang **0** (14/15) | zone rang **0** (18/19) | identique |
| 1545 | zone rang 0 (2/2) | absent | — |
| 1547 | zone rang 1 (1/1) | absent | — |

| mesure | `7344d24f` | `696a9d7c` | seuil | verdict |
|---|---|---|---|---|
| coherence de la carte | **54/58 = 93,1 %** | **62/63 = 98,4 %** | 90 % | **TENU** |
| temoin PERMUTATION (slots reapparies) | 41,4 % | 47,6 % | — | le hasard de la carte |
| temoin DECALE (+20 s) | 57,1 % | 51,4 % | — | non informatif seul |
| **stabilite inter-films** | **3/3 slots communs = 100,0 %** | 90 % | **TENUE** |

La stabilite est verifiee par un test qui relit les TSV (`TestZoneEtatPhase2aStabilite`), donc
rejouable sans re-decoder un film.

**Reserve ecrite** : la cle `tag 5` n'est PAS la cle de nommage des slots de jauge. Elle est
absente de 1532/1542 et DIFFERE entre les deux films sur 1537 (`0x27954600` vs `0xE2D00983`). Les
identifiants stables mesures en phase 1 (1525-1527) vivent sur d'AUTRES slots, muets ici. La
clause de stabilite est donc tenue par le NUMERO DE SLOT, pas par le tag 5.

## 4. CB.2a.2 — le tag 4 est le PROPRIETAIRE, et sa clause temporelle ne tient pas

### La valeur (la question produit) : ETABLIE

Les valeurs du tag 4 ne sont pas des handles : ce sont **`0xFFFFFFFF`, `0x00000000`,
`0x00000001`** — aucune equipe, equipe 0, equipe 1. Mesure PAR SLOT (agreger tous les slots
melangeait deux familles de proprietes et faisait conclure a tort a la premiere passe) :

| slot | zone | captures equipe 0 | captures equipe 1 | lecture |
|---|---|---|---|---|
| 1530 | 1 | `0x00000000` 9/9 puis 8/8 | `0x00000001` 10/10 puis 9/9 | **PROPRIETAIRE** |
| 1535 | 2 | `0x00000000` 10/10 puis 10/10 | `0x00000001` 5/5 puis 11/11 | **PROPRIETAIRE** |
| 1540 | 0 | `0x00000000` 6/6 puis 7/7 | `0x00000001` 6/6 puis 6/6 | **PROPRIETAIRE** |
| 1531, 1536, 1541 | 1, 2, 0 | `0xFFFFFFFF` (tout) | `0xFFFFFFFF` (tout) | canal NEUTRE, autre semantique |
| 1546 | 0 | `0x00000001` 5/5 (`696a9d7c`) | `0x00000001` 2/2 (`7344d24f`) | discordant — le seul |

| film | concordance valeur == index d'equipe | hors emissions neutres | seuil | verdict |
|---|---|---|---|---|
| `7344d24f` | 48/105 = 45,7 % | **48/48 = 100,0 %** | 90 % | **PROPRIETAIRE** |
| `696a9d7c` | 51/120 = 42,5 % | **51/56 = 91,1 %** | 90 % | **PROPRIETAIRE** |

Les cinq discordances de `696a9d7c` viennent toutes du slot 1546, le plus faible du lot (5 a 6
emissions). Les trois slots canoniques — un par zone — sont a **100 % sur les deux films**.

### La clause temporelle : NON TENUE, et le rappel est la cause

| film | precision | hasard | facteur (exige 2x) | rappel | hasard | seuil | verdict |
|---|---|---|---|---|---|---|---|
| `7344d24f` | 85/144 = 59,0 % | 10,5 % | **5,60x** | 101/135 = **74,8 %** | 13,2 % | 80 % | **NON TENU** |
| `696a9d7c` | 90/146 = 61,6 % | 11,1 % | **5,53x** | 117/151 = **77,5 %** | 14,0 % | 80 % | **NON TENU** |

La precision passe largement (5,5x le hasard, pour un facteur 2 exige) ; c'est le RAPPEL qui
manque de 2,5 a 5,2 points. Interpretation, sans deplacer le seuil : une capture ne change pas
toujours le proprietaire — une zone deja tenue par l'equipe qui la securise ne fait pas changer
le tag 4, et `zone_secures` compte dans le denominateur.

**Mesure SANS CIRCULARITE, publiee a cote** (precision contre TOUTES les captures du film, aucun
vote de zone) : 96/144 = 66,7 % pour un hasard de 27,9 % (2,39x) sur `7344d24f`, 99/146 = 67,8 %
pour 29,0 % (2,34x) sur `696a9d7c`. La zone d'un slot de tag 4 est etablie par VOTE (les captures
proches de ses changements) faute de slot commun avec la jauge : ce vote rend la precision PAR
ZONE partiellement circulaire, d'ou cette seconde colonne qui ne lui doit rien.

**« Conteste » : NON MESURABLE.** La piste demandait la valeur du tag 4 pendant les rampes non
abouties ; or les slots de rampe (tag 3) ne portent pas de tag 4 — la question est vide sur ce
corpus, et c'est ecrit plutot que contourne.

## 5. CB.2a.3 — KOTH : la colline se lit dans la grappe

**L'hypothese de depart a ete REFUTEE par la mesure, et la trace en est gardee dans le code.** La
premiere definition segmentait par SLOT ACTIF (une colline = un slot, comme une zone = un slot en
Strongholds) : sur `01e1f945`, **UN SEUL slot (1474) porte la jauge de tout le match**, donc cette
lecture rend UNE periode et n'apparie rien. La definition retenue segmente par RAMPE : chaque
montee est une session de garde, la zone se lit dans la grappe des positions PENDANT la montee, et
les rampes voisines qui designent la meme zone fusionnent. **Le seuil (80 %) et la methode
d'attribution (grappe jugee contre des zones translatees de 12 m) sont inchanges.**

| film | carte | zones au catalogue | rampes -> periodes | couverture | seuil | verdict | NETTETE | grappe vs temoin |
|---|---|---|---|---|---|---|---|---|
| `01e1f945` | Catalyst | 8 | 60 -> 28 | **503 398 / 552 836 ms = 91,1 %** | 80 % | **TENU** | 7/28 periodes, 54,0 % du temps | 36,1 % vs 24,1 % |
| `8076f97f` | Shogun | 3 | 22 -> 3 | **302 607 / 360 859 ms = 83,9 %** | 80 % | **TENU** | 2/3 periodes, 100 % du temps | 31,5 % vs **2,2 %** |
| `606d9844` | Chasm | 8 | 7 -> 5 | **202 320 / 247 114 ms = 81,9 %** | 80 % | **TENU** | **0/5 periodes, 0 %** | 28,9 % vs 19,9 % |
| `0a247154` | Solitude | **carte ABSENTE du catalogue** | — | — | — | NON MESURABLE | — | — |
| `000d5950` | Cliffhanger (temoin) | 8 | **0 rampe** | — | — | SANS OBJET | — | — |

**La clause de couverture est TENUE sur trois films (deux exiges), et elle est FAIBLE — il faut le
dire.** Des que des rampes sont reparties sur le match, les periodes etendues couvrent presque
tout : le 80 % est quasi mecanique. Ce qui porte reellement le resultat, c'est la NETTETE
(la zone retenue devance la 2e d'un facteur 2 ET bat le temoin translate), et elle est tres
inegale : excellente sur Shogun (100 % du temps attribue, temoin a 2,2 % contre 31,5 %), moyenne
sur Catalyst (54 %), **nulle sur Chasm** (0/5). Le verdict au seuil ecrit est TENU ; la qualite
reelle de l'appariement KOTH est « etablie sur 2 films sur 3 ».

Cause probable du cas Chasm, notee sans etre traitee : 42 trajectoires seulement, 7 rampes, un
match tres desequilibre (105-8) — et surtout, **le catalogue ne contient aucun role de colline**,
donc l'appariement se fait sur les zones de Strongholds et d'Extraction de la carte, qui ne
coincident pas forcement avec les emplacements de colline.

## 6. GATE 2a — verdict

> Enonce : CB.2a.1 >= 90 % ET (CB.2a.2 proprietaire etabli OU CB.2a.3 tenu).

**GATE 2a : TENU, et par les deux branches a la fois.**

- CB.2a.1 : **93,1 % et 98,4 %** (seuil 90 %), temoins a 41-48 % (permutation) et 51-57 %
  (decalage), stabilite inter-films **100 %**.
- CB.2a.2 : **PROPRIETAIRE ETABLI** — 100,0 % et 91,1 % hors emissions neutres (seuil 90 %) ;
  clause temporelle non tenue (rappel 74,8 % et 77,5 % pour 80 %), negatif ecrit.
- CB.2a.3 : **TENU** au seuil de couverture sur 3 films ; nettete inegale, ecrite ci-dessus.

## 7. Forme PROPOSEE de `zoneStates` (proposition, AUCUNE publication)

Ce que la mesure autorise a publier, et rien de plus :

```
zoneStates: [{
  zoneRef,            // index dans mapObjectives.zones — la carte slot -> zone est mesuree
  spans: [{
    t0, t1,           // frames du rejeu (axe des positions, origine deja retranchee)
    owner,            // teamId | null — valeur du tag 4 : 0, 1, ou null pour 0xFFFFFFFF
    progress?         // 0..1 — jauge du tag 3, dequantifiee depuis [-100, +100]
  }]
}]
coverage.zones: { zonesAppariees, zonesTotales, capturesAppariees, capturesTotales }
```

Trois reserves a porter dans la phase 2b, chacune mesuree ci-dessus :

1. **La carte slot -> zone se construit PAR MATCH**, a partir des captures nommees : elle n'est
   pas une constante de carte. Un match sans oracle nomme (KOTH, Oddball) n'a pas de carte.
2. **`owner` n'est publiable qu'en Strongholds** : il vient du tag 4, dont les slots ne parlent
   que la ou il y a des captures nommees. En KOTH, la mesure ne rend que la zone ACTIVE.
3. **`progress` est une jauge sans identite propre** : c'est le slot qui porte la zone, pas la
   valeur. Publier la valeur sans la carte du meme match n'aurait pas de sens.

## 8. Statut des items

- [x] **CB.2a.1** — appariement mesure sur les deux Strongholds, courbe de tolerance publiee,
  table slot -> zone ecrite, stabilite verifiee par test. 93,1 % / 98,4 % pour un seuil de 90 %.
- [x] **CB.2a.2** — clause temporelle NON TENUE (rappel), volet VALEUR **TENU** :
  proprietaire etabli a 100,0 % / 91,1 %. « Conteste » non mesurable (cause ecrite).
- [x] **CB.2a.3** — couverture >= 80 % sur 3 films KOTH ; `0a247154` non mesurable (carte absente
  du catalogue), temoin Slayer muet. Nettete publiee, inegale.
- [x] **Gate 2a** — TENU par les deux branches.

## 9. Cout machine (D17)

Un film par processus, avant-plan, plafond 3 Go jamais approche (le processus de test culmine a
59 Mo mesures). Duree par film, compilation comprise : **32 s** (`0a247154`, saute au catalogue) a
**463 s** (`7344d24f` premiere passe, machine partagee avec un autre agent) ; **116 a 175 s** en
regime de cache chaud. Gates : `go test` des deux paquets 40 s, lint 25 s.

**Incident machine, note pour la memoire du depot** : un `go test` tue par un pipe PowerShell
(`Select-Object -First N`) laisse le binaire `*.test.exe` ORPHELIN, qui continue de tourner et
vole la machine (1 095 s de CPU constates avant de le tuer). Rediriger vers un fichier de log,
jamais vers un pipe tronque.

## 10. Decouvertes (hors perimetre — notees, NON traitees)

1. **Un slot `ti=13` est une PROPRIETE, pas un objet de jeu.** Trois familles coexistent par zone
   (jauge, proprietaire, canal neutre). Toute lecture future de `ti=13` doit partir de la, et le
   nom porte par i0 est la cle qui le montre — il etait consomme sans etre lu depuis la phase 1.
2. **Les slots 1531, 1536, 1541 portent `0xFFFFFFFF` en permanence** apres chaque capture, des
   deux equipes, sur les deux films. Un canal enumerable qui ne prend jamais qu'une valeur sur ce
   corpus : candidat « equipe qui conteste » ou « dernier proprietaire », a mesurer sur un match
   ou une zone est reprise sans etre securisee.
3. **Le catalogue d'objectifs n'a AUCUN role de colline (KOTH)** — mesure sur 6 cartes : seuls
   `strongholds_zone`, `extraction_zone`, `flag_*` et `oddball_spawn` existent. Tant que ce role
   manque, l'appariement KOTH se fait sur des zones d'un autre mode, ce qui plafonne sa nettete.
4. **Solitude est absente du catalogue de formes** (`4a5e5612-...`), donc tout appariement
   geometrique y est impossible. Le catalogue couvre 72 cartes sur la centaine jouee.
5. **Le taux d'attribution STRICT (0 m) des captures de zone vaut 73,2 %**, tres au-dessus des
   ~10 % que le commentaire d'`AttributeZones` annonce pour le corpus general. Ce commentaire
   merite d'etre re-mesure : il decrit peut-etre un corpus plus large (toutes actions confondues).
6. **`8076f97f` et `0a247154` declenchent un WARN d'empreinte de registre ECS inconnue** ; la
   garde de registre de la phase 2a passe quand meme (les quatre noms attendus sont aux bons
   index). Deux builds de jeu differents dans le corpus.
