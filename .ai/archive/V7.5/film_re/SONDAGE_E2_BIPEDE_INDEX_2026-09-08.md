# Sondage E2 — le film nomme-t-il le proprietaire d'un bipede ? (2026-09-08)

> Vague 3 du plan `.ai/PLAN_ORCHESTRATION_2026-09-07.md` §4 (decisions D12 et D13). Lot de
> RETRO-INGENIERIE DIAGNOSTIQUE, borne : une question, aucune ligne de production modifiee
> (`filmdec`, `himap`, `analysis/replay` intacts). Contrat d'execution : skill
> `plan-execution`. Branche `feat/v2-sondage-e2`, worktree `LevelUp-wt-sondage-e2`, base
> `feat/v2-p2-registre-joueurs` (`fd07570bc`).
>
> Mesures brutes versionnees : `mesures_e2_2026-09-08/` (5 TSV + la source de la sonde).

---

## 1. La reponse, en cinq lignes

1. **OUI.** Le record de CREATION (type NEW) d'une entite bipede `ti=35` porte l'index de
   participant de son proprietaire, dans son default-state (`FUN_140F44C38`), sur 5 bits.
2. Le champ est `ECS_ReadEntityRefIndex5` (`FUN_1407f2058` : `R(1)` porte INVERSEE puis
   `R(5)`) — la MEME primitive que `EnumA`/`EnumB` du dead-state, que la production lit deja
   comme des index de participant absolus (`killsource/walk.go:224`).
3. Sur 529 records lus (5 films), la porte est OUVERTE **529 fois sur 529**, et la valeur
   tombe dans le roster du film **527 fois sur 529** (les 2 exceptions sont un index que
   l'artefact ne publie pas, sur le seul film a bot).
4. Couverture mesuree des **vies de bipede nommees DIRECTEMENT** : **90,3 % / 86,8 / 95,0 %**
   sur `d9781168` / `bf15f7ab` / `64e8adfa` — **372 vies sur 408**, temoin fantome a **0**.
5. Contre le pont par morts : **329 concordants, 18 discordants, 39 vies neuves**. Les
   discordants sont **7 PAIRES EXACTEMENT ECHANGEES** entre deux vies qui finissent a la
   MEME image (fin de manche, fin de film) : c'est le pont qui se trompe, la lecture qui
   tranche.

---

## 2. Hypothese H1 — l'en-tete de creation de l'entite (RETENUE, critere d'arret atteint)

L'hypothese de l'utilisateur (« l'entite bipede avait l'index ; trajectoires et inventaire
etaient ses enfants ») est **verifiee** : le lien est a la CREATION, pas dans les positions.

### 2.1 Ce qui a ete lu, avant de coder

| Piece | Ce qu'elle apporte |
|---|---|
| `apps/go-api/internal/analysis/filmdec/default_state.go:135-200` (`consumeBipedDefaultState`) | La grammaire bit-exacte du default-state de `ti=35`, portee de `FUN_140F44C38`. Ses TROIS premieres feuilles etaient consommees et JETEES : version, `player-representation-name`, `ECS_ReadEntityRefIndex5`. |
| `filmdec/components_object.go:196-260` (`consumeDeadStateAnimBlock`) | `FUN_1407f2058` = `R(1)` porte ; si le bit vaut ZERO, `R(5)`. Le verdict R3 du 2026-06-07 la nomme : index de monde de 5 bits resolu par `FUN_1407f2058 -> FUN_14049746c -> FUN_140e958c4`. |
| `internal/games/halo_infinite/film/killsource/walk.go:198-224` | La PRODUCTION lit deja ces 5 bits comme un index de participant : `victim: int(d.dead.EnumA)`, `killer: int(d.dead.EnumB)`, valides `< nPlay`. |
| `filmdec/equipment_creation.go:83-96` | Le decoupage exact d'un en-tete de record NEW : `R(1)=0 + R(2)=1`, `R(13)` slot, `R(2)` generation, `R(6)` typeIndex. |
| `filmdec/frame_records.go:725-790`, `traverse.go:1075-1100` | Le meme decoupage vu du lecteur sequentiel, et le fait que `TraverseEntity` lit le `R(6)` typeIndex AVANT le default-state. |
| `.ai/V7.5/film_re/NOTE_PROJECTILE_OWNER_2026-09-01.md` | La piste de depart : les projectiles portent un index dans le meme espace de handles. Sa reserve n°2 disait `ti=37` CLOS (503/503 porte FERMEE) — sur `ti=35` la porte est OUVERTE 529/529. Le negatif du projectile ne s'etendait donc pas au bipede. |
| `filmdec/testdata/ecs_table.tsv` (`ti=35`, 64 composants) | Aucun composant DELTA de `ti=35` ne porte d'identite ; ce qui confirme que le lien, s'il existe, est dans le default-state. |

### 2.2 La specification EXACTE de l'enregistrement

Positions en bits, relatives au PREMIER bit de l'en-tete du record (`FrameRecord.HeaderBit`),
dans le payload d'un paquet DELTA (type 0). `IDLowBits = 13` sur les cinq films mesures.

| Offset | Largeur | Champ | Valeur observee |
|---|---|---|---|
| +0 | 1 | prefixe de type : 0 = « pas un delta » | 0 |
| +1 | 2 | type de record | 1 (`recNew`) |
| +3 | 13 | slot de l'entite (`IDLowBits`) | bande bipede du film |
| +16 | 2 | generation du handle | 0..3 |
| +18 | 6 | `R(6)` typeIndex | **35** |
| +24 | 1 | `g0` : porte de version du default-state | **1** (529/529) |
| +25 | 8 | version | **13** (529/529) |
| +33 | 1 | `gRep` : porte de `player-representation-name` | **1** (529/529) |
| +34 | 32 | `player-representation-name` (`FUN_14080dec4`) | **0x1876BDA0** — CONSTANTE sur les 5 films |
| +66 | 1 | porte `ECS_ReadEntityRefIndex5` — **INVERSEE** : 0 = valeur presente | **0** (529/529) |
| +67 | 5 | **index de participant absolu du proprietaire** | 0..10 selon le roster |

Verification bit a bit sur le premier record de `d9781168` (chunk 1, paquet 2052, octets
`88 80 d8 e1 b1 87 6b da`) : amorce de paquet 2 bits, en-tete a b2, slot = 515, generation 1,
`ti` = 35, `g0` = 1, version = 13, `gRep` = 1, representation = `0x1876BDA...`, puis la porte
et les 5 bits d'index = 3.

**Quand l'enregistrement apparait** : une fois par VIE de bipede — le pool de slots est
recycle et la paire `(slot, generation)` identifie la vie, exactement comme
`EquipmentLifeKey` le fait pour les objets du monde. Sur `64e8adfa`, 146 lectures pour
141 vies (6 vies portent deux lectures ; **aucune incoherence** : les deux lectures d'une
meme vie donnent toujours le meme index — 14 vies concernees sur les 3 films de mesure,
0 divergence).

**Cas des bots** : a VERIFIER avant integration. Sur `c75f33b8` (le seul film a bot du
materiau : 10 humains + `343 Flippant`), l'artefact publie 11 index (0..7, 9, 10, 11) et
l'index **8 lui manque**. La lecture directe rend `ref5 = 8` sur 2 vies, et **jamais 11**.
L'espace d'index du film n'est donc pas exactement celui que `PlayerIndexTable` + `bid`
publient. Report inscrit au registre.

**Cas des rejoints en cours** : non observable sur ce materiau (les `joinMatchMs` des
feuilles sont groupes). Ce que la mesure montre en revanche, c'est que la lecture ne depend
PAS des morts : sur `3372e7eb` — le film ou 6 joueurs sur 8 seulement etaient publies, faute
de mort — les 8 index sont lus et 38 vies sur 42 sont nommees directement.

**Frontieres de manche** : la generation du handle change avec la vie ; aucun traitement
particulier n'est requis.

### 2.3 Comment la mesure a ete faite — et la voie qui a ete REFUTEE en chemin

Deux passes, la premiere autorisant la seconde. Source archivee :
`mesures_e2_2026-09-08/sonde_e2c.go.txt`.

**Passe B — la chaine sequentielle (certitude, faible rendement).** World amorce par les
images-cles du chunk puis `DecodeFrameRecords` en sequence : c'est la recette de
`filmdec/game_entities_chain_test.go` (« la chaine ne devine rien »). Un record atteint par
la chaine est CADRE par la chaine des largeurs qui le precede. Rendement : 7 / 2 / 8 records
par film — trop peu pour une couverture, assez pour ETABLIR LA SIGNATURE (version 13,
porte de representation ouverte, mot `0x1876BDA0`).

**Passe A — l'ancrage bit a bit, avec la signature pour gate.** Le meme balayage d'en-tete
NEW que `equipment_creation.go`, mais dont le gate de selectivite n'est PAS le composant i0 :
c'est le mot de 32 bits. **Temoin fantome** (meme code, bande de meme cardinalite faite de
slots qu'aucun bipede n'occupe, decalage +4096) : **0 lecture sur les 5 films**. Le plancher
de faux positifs du gate est nul.

**VOIE REFUTEE, et elle merite d'etre ecrite** : la premiere sonde a repris le gate i0 de
`decodeBipedI0Pos` (`filmdec/vehicle_creation.go:43`), celui qui valide les creations
d'equipement, d'arme au sol et de vehicule. **Il ne peut RIEN valider sur le bipede** : le
default-state de `ti=35` n'est porte qu'a ~120 bits sur ~380 (`filmdec/default_state.go`,
en-tete de fichier), donc l'ancre i0 calculee APRES lui est fausse par construction. Mesure :
1 847 ancres, 181 acceptees, dont **170 a plus de 10 000 quanta** de la trajectoire reelle de
leur propre slot — 90 % de bruit. Un negatif obtenu par cette voie n'aurait rien dit du film.

### 2.4 La couverture mesuree

Trois films de mesure (borne du lot). « Vies » = `identity.bipedSlots` de l'artefact cuit
(schema 50) ; « nommees directement » = vies distinctes qu'une lecture d'index designe.

| Film | Mode | Vies | Lectures | Porte ouverte | Index hors roster | Vies nommees | Couverture | Concordants | Discordants | Nouveaux | Hors vie | Fantome |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| `d9781168` | Oddball (le temoin) | 176 | 167 | 167 | 0 | **159** | **90,3 %** | 131 | 13 | 22 | 1 | 0 |
| `bf15f7ab` | Slayer | 91 | 85 | 85 | 0 | **79** | **86,8 %** | 72 | 0 | 8 | 5 | 0 |
| `64e8adfa` | CTF | 141 | 146 | 146 | 0 | **134** | **95,0 %** | 126 | 5 | 9 | 6 | 0 |
| **total** | | **408** | **398** | **398** | **0** | **372** | **91,2 %** | **329** | **18** | **39** | **12** | **0** |

Deux films supplementaires ont ete lus pour instruire la SPECIFICATION (bots, joueurs qui ne
meurent pas), pas pour la couverture :

| Film | Vies | Lectures | Index hors roster | Vies nommees | Couverture | Concordants | Discordants | Nouveaux |
|---|---|---|---|---|---|---|---|---|
| `3372e7eb` | 42 | 42 | 0 | 38 | 90,5 % | 32 | 0 | 10 |
| `c75f33b8` (1 bot) | 88 | 89 | **2** | 75 | 85,2 % | 51 | 6 | 26 |

**Le denominateur honnete du champ.** Le champ fait 5 bits : il peut porter 0..31. Sur les
398 lectures des trois films de mesure, il vaut **toujours** un index du roster publie
(0..7). Un champ de bruit aurait rempli les 32 valeurs — c'est exactement ce que faisait la
voie refutee (§2.3), dont les `ref5` s'etalaient de 0 a 31 avec un mode a 22.

**Repartition par joueur** (`d9781168`) : 17 / 23 / 18 / 15 / 25 / 21 / 24 / 24 pour les
index 0 a 7 — huit joueurs, huit paquets de vies du meme ordre. Le pont par morts, lui, en
laissait 24 sans nom.

### 2.5 Les 18 discordants : SEPT PAIRES ECHANGEES, et le pont a tort

18 lignes discordantes portent sur 17 vies. **Quatorze d'entre elles vont par deux**, entre
deux vies qui **se terminent a la meme image** (ou a une image d'ecart), et l'echange est
EXACT — le pont attribue a l'une le joueur que le film ecrit sur l'autre, et reciproquement :

| Film | Vies (intervalles d'images) | Fin commune | Verdict |
|---|---|---|---|
| `d9781168` | 512 [0..984] / 528 [868..984] | 984 | echange exact |
| `d9781168` | 579 [3005..3212] / 582 [3138..3212] | 3212 | echange exact |
| `d9781168` | 645 [6113..6415] / 649 [6269..6415] | 6415 | echange exact |
| `d9781168` | 646 [6114..6320] / 648 [6212..6319] | 6320 / 6319 | echange exact |
| `d9781168` | 668 [6852..6962] / 669 [6877..6962] | 6962 | echange exact |
| `64e8adfa` | 588 [4561..5030] / 594 [4945..5030] | 5030 | echange exact |
| `64e8adfa` | 631 [7530..7648] / 637 [7519..7648] | 7648 | echange exact |

Les **trois restantes** (`d9781168` slots 521, 642, 644) ne s'apparient pas DANS l'ensemble
discordant : leur partenaire d'echange est vraisemblablement une vie que la lecture directe
n'a pas couverte (elles font partie des 17 vies sans record de creation lu), donc invisible
ici. Elles sont a instruire une par une, pas a supposer.

C'est la signature exacte du defaut connu de `nameLivesByDeaths`
(`replay/lives.go:437-470`) : l'appariement est glouton par ecart croissant, et quand deux
fins de vie tombent au meme instant — fin de manche, fin de film, double kill — le
departage est arbitraire (`ps[i].li < ps[j].li`, l'ordre des slots). Le registre des reports
porte deja la famille « grappes de frontiere de manche » (10 pistes sur `d9781168`, lot
P2-bis). **La lecture directe la ferme.**

Les 39 « nouveaux » sont des vies que le pont laissait `non_resolu` : sur `d9781168`, les
quatre premieres sont les vies d'ouverture (slots 514-517, images 0-39, index 2-3-4-5) —
celles d'avant la premiere mort, qu'un pont par morts ne peut structurellement pas nommer.

### 2.6 Commande de la sonde

Sonde JETABLE, NON COMMITTEE (voir §5) ; sa source est archivee telle qu'executee dans
`mesures_e2_2026-09-08/sonde_e2c.go.txt`. Pour la rejouer, la recopier sous
`apps/go-api/cmd/sonde_e2c/main.go` puis :

```bash
export CGO_ENABLED=0
cd apps/go-api
go run ./cmd/sonde_e2c \
  -film     <cache>/film_chunks/d9781168 \
  -artefact <cache>/replays/halo_infinite/d9781168.json \
  -out      <scratchpad>/d9781168_c.tsv
```

Elle ne prend aucun verrou de decodage de paquet (elle n'installe aucun hook `filmdec`) et
ne lit que le film et l'artefact ; elle n'ecrit rien sous `data/`.

---

## 3. H2 et H3 — NON EXPLOREES, et pourquoi c'est le contrat

Le lot s'arrete « a la premiere hypothese qui donne un lien avec une couverture >= 90 % des
vies de bipede d'un film ». H1 rend 90,3 % sur `d9781168` et 95,0 % sur `64e8adfa`, avec un
temoin fantome nul et zero index hors roster. **H2 (le flux `ti=5` complet) et H3
(`ManagedPropertyFilmIndex` / proprietes gerees) ne sont donc pas ouvertes** — statut `[!]`
assume, justification : critere d'arret atteint.

Ce que la lecture de H1 a quand meme etabli sur H2, et qui vaut d'etre note : le
default-state de `ti=5` (`consumeDefaultStateTI5`, `filmdec/default_state_arch.go:80-83`)
porte lui aussi un index de joueur, en `R(6)` cette fois, valide `< 0x20` par la valeur de
retour du deserialiseur. Le lien `entite joueur -> index` est donc lisible par la MEME voie
si on en a besoin un jour — mais il ne serait pas necessaire : H1 relie directement le CORPS
au joueur, sans passer par l'entite joueur.

---

## 4. Plan d'integration ADDITIF (5 items) — aucune reecriture

Aucun de ces items ne modifie une largeur de bit lue ni ne retire une voie existante.

| # | Item | Taille estimee |
|---|---|---|
| I1 | **Lecteur `ScanBipedCreations` dans `filmdec`** : publier les trois feuilles du prologue du default-state de `ti=35` par un hook `SetBipedCreationHook`, exactement comme `SetEquipmentCreationHook` publie celles de `ti=37` (`equipment_creation.go:70-86`) — les largeurs consommees ne changent PAS d'un bit, `consumeBipedDefaultState` deroule les memes `br.ReadBit()`/`ReadBits`. Le balayage ne peut PAS reutiliser `runCreationWalk` (son gate est i0, inatteignable : §2.3) : gate = mot de representation + bande de slots `ti=35`, avec le temoin fantome en test. | ~200 L Go + ~130 L de test |
| I2 | **Publication dans le registre d'identite** : `BipedCreation{Slot, Gen, TimestampUS, FilmIndex}` entre dans `IdentityInput` (`replay/build.go:69`), et `identity.bipedSlots[].link` devient `{source:"direct", method:"bipedCreation", readings:n}` pour les vies couvertes. Le pont par morts N'EST PAS supprime : il reste le repli `deduit` des vies sans record de creation. | ~120 L (`identity_registry.go`, `lives.go`, `lives_export.go`) |
| I3 | **Le pont par morts devient VERIFICATION** : compteurs `bridgeAgreements` / `bridgeSwaps` publies dans `coverage`, un desaccord n'ecrase plus le nom (le direct gagne) mais s'inscrit. Test de non-regression sur les 7 paires echangees relevees ici. | ~60 L + 1 test |
| I4 | **Gate corpus** : `identity.coverage.bipedSlot.direct` entre au gate (`make replay-corpus-gate`, `Makefile:231-232`) avec un plancher par film ; le gate echoue si `direct` baisse ou si `deduit` monte a `direct` constant — la regle deja ecrite en (g) de l'inventaire P1. | ~40 L de config + recuisson des 7 temoins |
| I5 | **Goldens** : les 18 noms echanges et les 39 vies neuves FONT BOUGER les goldens de `replay`. Le diff attendu se documente AVANT la recuisson (film par film, vie par vie), sans quoi une regression se cacherait dans le bruit. | 0 L de code, 1 revue de diff |

**Prealable a I1, non negociable** : trancher le cas des bots (§2.2 — `ref5 = 8` sur
`c75f33b8`, index absent de la table publiee, index 11 du bot jamais lu). Tant qu'il n'est
pas tranche, l'integration doit COMPTER les index hors table plutot que les publier.

**Deux limites a porter dans le code, pas dans un commentaire d'humeur :**

1. Le gate par signature manquerait un bipede dont la representation differe (autre corps
   que le Spartan multijoueur). Le lecteur doit donc COMPTER les records NEW `ti=35` dont le
   mot de 32 bits n'est pas la constante, et alarmer si ce compte devient non nul.
2. L'origine de la frise et la tolerance de 10 s n'ont servi qu'a la CONFRONTATION avec la
   verite terrain ; la lecture, elle, n'a besoin d'aucune horloge — la creation designe sa
   vie par `(slot, generation)`.

---

## 5. Ce que ce lot n'a PAS fait, et pourquoi

- **Aucune ligne de `filmdec`, `himap` ou `analysis/replay` modifiee** : c'est le contrat du
  sondage. Les trois sondes ont vecu sous `apps/go-api/cmd/sonde_e2*/` et ont ete
  **retirees du worktree**. Elles ne sont pas committees, et ce n'est pas de la pudeur : une
  sonde qui relit le chunk 0 par `ReadFilmChunk`, reconstruit sa bande de slots et boucle sur
  tous les chunks declencherait `no_film_reread_test.go`, `no_rewritten_slot_band_test.go` et
  `no_unbounded_film_loop_test.go` — les garde-rails qui existent justement pour interdire ce
  qu'une sonde jetable se permet. La source de la sonde de mesure est ARCHIVEE
  (`mesures_e2_2026-09-08/sonde_e2c.go.txt`), et l'item I1 est sa version propre.
- **Aucun test `gamefiles`, aucun `go test ./...` global, aucun Ghidra** : hors budget du
  lot. Les adresses `FUN_*` citees viennent des fichiers du depot, relus dans cette session.
- **Aucune ecriture sous `data/`** : le parc reste en lecture seule ; tout le materiau vient
  de la copie de travail du scratchpad.
- **H2 et H3 non ouvertes** : critere d'arret atteint (§3).

---

## 6. Decouvertes annexes (consignees, NON traitees)

1. **`nameLivesByDeaths` echange les noms de deux vies qui finissent a la meme image** — 9
   paires sur 2 films (§2.5). La cause est le departage arbitraire `ps[i].li < ps[j].li`
   (`replay/lives.go:462`) quand deux ecarts sont egaux. Ferme par I2/I3.
2. **Le default-state de `ti=5` porte un index de joueur en `R(6)`**
   (`default_state_arch.go:80-83`), jamais publie. Voie de secours pour H2 si le besoin
   revient.
3. **L'espace d'index de participant du film est plus large que la table publiee** sur les
   films a bot (`c75f33b8` : index 8 lu, absent de `identity.players`). A instruire avant
   I1.
