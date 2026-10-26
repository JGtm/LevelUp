# LOT 3 — LES GRENADES SANS L'ANCRE : R2 DECOUPLE DE R1 (2026-08-25)

> Branche `wt/grenades-sans-ancre` (base `feat/v75`, lot inventaire fusionne). Document
> fondateur : `MESURE_TROUS_INVENTAIRE_2026-08-24.md` (§3.1, piste 1). Corpus identique :
> 24 films de `data/cache/film_chunks/`, echantillonnes a pas constant sur les 951 prefixes
> tries, 698 images-cles, **6 721 records de bipede** — les memes totaux au record pres.
>
> Outils de mesure laisses au depot, gates par variable d'environnement (sautes en CI) :
> `apps/go-api/internal/analysis/replay/inventory_position_i22_test.go`
> (`TestPositionI22`, `TestInventaireApresR2b`, `TestOracleTypesPortesEtLances`).

---

## 0. Verdict en trois phrases

1. **La loi de position existe, et elle est exacte.** Le motif i22 commence a un offset compris
   dans **[-204, -139] bits du debut du bloc de munitions**, repere que R4 etablit au bit pres
   SANS aucune information de capacite. La regle « le PREMIER motif i22 de cette fenetre » rend
   la position vraie **1 167 fois sur 1 167** sur les records ou l'ancre existe et permet de la
   verifier.
2. **Elle transfere aux records sans ancre**, qui sont pourtant DEUX FOIS PLUS COURTS : **4 277
   des 4 278 records cibles** rendent une lecture, sur les MEMES modes d'offset que la
   reference, tandis que le controle negatif (meme regle, fenetre decalee de deux largeurs) ne
   rend que 5,4 % de lectures, **toutes de somme nulle et concentrees a 92 % sur un unique
   offset** — du remplissage, pas un champ.
3. **Le chiffre du 24/08 s'effondre** : les records ARMES SANS GRENADE passent de **4 278
   (63,7 %) a 1 (0,015 %)**, et les compteurs de grenade lus de **1 271 (18,9 %) a 5 551
   (82,6 %)**. Les 11 films qui ne rendaient AUCUNE grenade de tout le match en rendent
   desormais entre 106 et 230 chacun.

---

## 1. La demarche : d'abord la position, jamais la valeur

La mesure du 24/08 avait ferme les voies par MOTIF (§4 : fenetre elargie a 400 bits, variante
d'ancre au bit 26, prefixe 17 bits — cette derniere battue par son propre temoin, 443 contre
328). Ce lot ne rouvre aucune d'elles. Il part d'un fait deja acquis du fichier de production :
**R3 et R4 trouvent leur position SANS R1.** R4, en particulier, borne le bloc de munitions par
un critere de LARGEUR — le parse doit atterrir exactement sur le bit de porte d'i43, juste avant
la premiere famille d'arme. Le record porte donc des reperes qui marchent.

La question posee a donc ete une question de POSITION, pas de contenu : **a quelle distance d'un
repere independant le motif i22 se trouve-t-il ?**

### 1.1 Cinq reperes mis en concurrence sur le meme corpus

Corpus d'entrainement : les **1 167 records ou R1, R2 et R4 reussissent tous les trois** — la
position VRAIE d'i22 y est connue, puisque la voie par l'ancre la donne. C'est exactement le
`h_complet` du tableau du 24/08, au record pres.

| repere | valeurs d'offset distinctes | couverture du top-3 | verdict |
|---|---:|---:|---|
| debut du record | 76 | 39,3 % | ne borne rien |
| fin du record | 490 | 8,0 % | ne borne rien |
| derniere famille d'arme | 81 | 23,9 % | ne borne rien |
| premiere famille d'arme | 23 | 49,0 % | borne, mais moins bien |
| fin du bloc de munitions | 23 | 49,0 % | idem |
| **debut du bloc de munitions** | **22** | **62,1 %** | **retenu** |
| *(temoin : ancre R1, le repere actuel)* | *16* | *66,8 %* | *reference* |

Le debut du bloc de munitions est le seul repere INDEPENDANT dont la dispersion approche celle
de l'ancre elle-meme. L'offset y tient dans **66 bits** : `[-204, -139]`.

### 1.2 La strategie de departage, mesuree et non choisie

Trois strategies ont ete balayees sur toutes les fenetres possibles (pas de 5 bits, de -400 a 0),
en maximisant `2 x justes - rendues` — une regle qui rend beaucoup de mauvaises lectures est pire
qu'une regle qui se tait :

| strategie | meilleure fenetre | lectures rendues | dont JUSTES |
|---|---|---:|---:|
| **premier** motif de la fenetre | `[-250, -135]` | **1 167 / 1 167 (100 %)** | **1 167 (100,00 %)** |
| motif UNIQUE de la fenetre | `[-250, -155]` | 834 / 1 167 (71,5 %) | 834 (100,00 %) |
| dernier motif de la fenetre | `[-250, -155]` | 1 069 / 1 167 (91,6 %) | 834 (78,02 %) |

« Premier » est la seule des trois a etre a la fois complete et exacte. C'est la regle livree.

### 1.3 La marge, mesuree elle aussi

Le motif i22 est une grammaire FAIBLE : un record en porte **8 a 9 occurrences** en moyenne
(53,5 % a 8, 33,1 % a 9), et 3 a 4 dans les seuls 400 bits precedant le bloc de munitions. Ce
n'est donc jamais le motif qui identifie i22, c'est sa position — et la position n'est sure que
si la fenetre est etroite.

**Le candidat parasite le plus proche EN DESSOUS d'une position vraie est a 105 bits** (minimum
sur les 1 167 records ; pire cas releve : `000d5950`, chunk 21, slot 588). La fenetre livree est
`[-216, -127]` : la loi mesuree elargie de 12 bits de chaque cote, ce qui laisse encore plus de
90 bits de marge au parasite le plus proche.

---

## 2. LE TEST REFUTABLE : la loi tient-elle la ou l'ancre n'est pas ?

C'est le seul point qui pouvait tuer ce lot, et il n'etait pas gagne d'avance : la mesure du
24/08 avait releve que **les records sans ancre sont DEUX FOIS PLUS COURTS** (6 322 bits contre
12 697), donc qu'ils n'ecrivent pas le meme etat. Rien ne garantissait qu'i22 y occupe la meme
place relative.

### 2.1 Taux de lecture et forme de la distribution

Population cible : **4 278 records** ou R1 echoue et R4 passe — le meme dénombrement, au record
pres, que la ligne « capacite + grenades » du tableau du 24/08.

| | fenetre | lectures rendues | offsets | sommes |
|---|---|---:|---|---|
| **signal** | `[-216, -127]` | **4 277 / 4 278 (100,0 %)** | -159 (50,0 %), -184 (17,5 %), -175 (14,1 %), -200 (5,0 %), -139 (3,3 %)... 25 valeurs | 2 (66,0 %), 0 (13,0 %), 1 (8,4 %), 4 (7,7 %), 3 (4,9 %) |
| **temoin decale** | `[-396, -307]` | **229 / 4 278 (5,4 %)** | -314 (92,1 %) puis des unites | **0 (100,0 %)** |
| *reference (entrainement)* | | *1 167 / 1 167* | *-159 (26,7 %), -175 (26,1 %), -167 (9,3 %), -200 (8,6 %)... 22 valeurs* | *par construction > 0* |

Trois faits concordent, et chacun aurait pu refuter seul :

- **le taux** : 100,0 % contre 5,4 % pour le temoin, soit un rapport de 18,6x ;
- **la forme** : les offsets trouves sur la cible tombent sur les **memes modes** que la
  reference (-159, -184, -175, -200, -139, -151, -167, -155, -192, -171, -176), et non repartis
  au hasard sur les 90 bits de la fenetre. Une fenetre trop large qui ramasserait du bruit
  produirait un etalement, pas les memes pics ;
- **le contenu du temoin** : ses 229 lectures sont a **100 % de somme nulle** et a 92 % au meme
  offset unique. C'est un motif de remplissage nul, identifiable comme tel — exactement ce que la
  mesure du 24/08 redoutait de laisser passer, et ce que la fenetre de la loi n'attrape pas.

### 2.2 L'ORACLE INDEPENDANT : types portes contre types lances

Les compteurs i22 sont lus aux IMAGES-CLES ; les lancers de grenade sont decodes dans les
PAQUETS DELTA (`filmdec.ScanFilmGrenadeThrows`) et portent le TYPE lance. **Les deux canaux ne
partagent aucun bit.** Un joueur qui lance une Dynamo en portait une : si R2b lisait du bruit, la
repartition des types portes n'aurait aucune raison de suivre celle des types lances.

Table par couple (film, rang de grenade), 24 films x 4 rangs :

| | lance ET porte | **lance mais JAMAIS porte** | porte sans lancer | ni l'un ni l'autre |
|---|---:|---:|---:|---:|
| **signal** | **63** | **0** | 16 | 17 |
| **temoin** (rang porte decale de 1) | 52 | **11** | — | — |

**Zero contre-exemple sur 63 couples.** Mieux : sur les **20 films ou le canal des lancers rend
quelque chose, le vecteur des types portes est IDENTIQUE au vecteur des types lances, film par
film** — y compris sur les 11 films qui, avant ce lot, ne rendaient AUCUNE grenade. Les 16
couples « porte sans lancer » se lisent en un coup d'oeil : ce sont exactement les 4 films dont
le canal des lancers est vide (`1c5c10cc`, `97b34406`, `a26dbcdb`, `e67ad176` — 4 x 4 = 16), donc
des cases sans oracle, pas des desaccords. Le temoin, lui, produit 11 contradictions.

---

## 3. Ce qui a ete livre

### 3.1 R2b, la voie positionnelle — `inventory_grenades_rules.go` (nouveau)

Les DEUX voies de lecture d'i22 vivent desormais dans un fichier a elles (`inventory_decode.go`
etait a 500 lignes, seuil du depot) :

- `invGrenadeCountsAt` — **la grammaire i22, en une seule copie** pour les deux voies : elles ne
  doivent pas pouvoir diverger sur ce qu'est un motif i22 ;
- `invGrenadesAfter` (R2a) — inchangee : premier motif de somme non nulle apres l'ancre ;
- `invGrenadesNearAmmo` (R2b) — premier motif dont le debut tombe dans `[-216, -127]` bits du
  debut du bloc de munitions.

**R2a reste PRIORITAIRE** : R2b n'est appelee que si R2a n'a rien rendu. Aucune lecture existante
ne change donc de valeur, et le cran de rendement le verrouille (`wantInvGrenAnchor = 120`
inchange sur le film de reference).

### 3.2 La somme nulle est publiee, parce que la position la borne

R2a exigeait `somme > 0` faute de position sure : la mesure du 24/08 avait montre (§4.4) que
**104 records sur 104** portaient un motif i22 entierement nul apres l'ancre — un Spartan ayant
lance toutes ses grenades — et que la regle le rejetait, rendant « zero grenade » indistinguable
d'une non-lecture. Bornee par la POSITION, R2b n'a plus besoin de ce garde-fou de VALEUR : elle
publie le zero, conformement a la doc du champ `Inventory.G` (« un tableau present dont une case
vaut 0 dit "ce type, aucune en reserve", ce qui est une mesure »). R2a garde le sien : sans
position sure, il reste son seul rempart.

Mesure : **656 lectures sur 5 551 (11,8 %)** ont leurs quatre compteurs a zero.

### 3.3 Telemetrie par voie — `KeyframeInventoryStats`

`GrenadesByAnchor` / `GrenadesByPosition`, alimentes par `ScanFilmKeyframeInventory` depuis le
champ de decodage `KeyframeInventory.GrenadesByPosition`. **Aucun champ de contrat ajoute, aucun
bump de `SchemaVersion`** : la provenance est une telemetrie, pas une donnee que le client
consomme. Les deux voies n'ont pas le meme statut — R2a est exacte par construction, R2b repose
sur une loi mesuree sur 24 films — et fondues dans un total, une derive du repli sur un film
d'une autre version du jeu ne se verrait nulle part.

---

## 4. AVANT / APRES, chiffre sur les memes 24 films

Rejoue par le DECODEUR DE PRODUCTION (`keyframeInventories`), pas par une copie des regles.

| grandeur | avant (24/08) | apres (25/08) |
|---|---:|---:|
| records de bipede | 6 721 | 6 721 |
| **records ARMES SANS GRENADE** | **4 278 (63,7 %)** | **1 (0,015 %)** |
| compteurs de grenade lus | 1 271 (18,9 %) | **5 551 (82,6 %)** |
| dont par l'ancre (R2a) | 1 271 | 1 172 (*) |
| dont par la position (R2b) | 0 | **4 379** |
| lectures vides (aucune arme) | 1 169 (17,4 %) | 1 169 (17,4 %) — inchange, hors perimetre |
| films rendant ZERO grenade | **11 / 24** | **0 / 24** |

(*) 1 271 -> 1 172 : la difference tient au denombrement, pas a une perte. Le 24/08 comptait
`h_complet` (1 167) plus les 102 « grenades seules » et 2 autres ; ici seule la voie effectivement
empruntee est comptee, R2b n'etant jamais appelee quand R2a a repondu. Aucun record ne perd sa
lecture : le seul « arme sans grenade » restant est unique et isole (`97b34406`).

### Pourquoi le §2.1 dit 4 278 et cette table dit 4 379 (R2b)

Les deux chiffres mesurent des POPULATIONS DIFFERENTES, et l'ecart se reconcilie exactement.
Le §2.1 raisonne sur les **4 278 records cibles** : R1 (l'ancre) n'y rend RIEN — ce sont les
seuls records que la mesure du 24/08 pouvait voir, puisqu'elle n'avait alors aucune autre voie
pour les grenades. Mais il existe une TROISIEME population, orthogonale a celle-la : des records
ou l'ANCRE EXISTE (R1 reussit) mais ou R2a n'y trouve, apres l'ancre, AUCUN motif i22 de somme
non nulle — le cas « 104/104 » du 24/08 §4.4 (un Spartan ayant lance toutes ses grenades avant
cette image-cle). `invPosObserve` les classait `invPosHorsSujet` et les ecartait purement et
simplement ; ni le corpus d'entrainement ni les trois controles du §2 ne les voyaient. Cette
revue les instrumente desormais sous une categorie dediee, `invPosAncreSommeNulle`
(`inventory_position_i22_test.go`), et le rejeu du corpus de reference (24 films,
`INV_SAMPLE=24`, 2026-08-25) en denombre **102**, dont R2b lit une somme nulle sur les **102**
(zero desaccord avec l'oracle des types, zero lecture R2b muette) :

```
ANCRE SOMME NULLE : 102 records — R2b somme nulle 102, R2b non-nulle accorde a l'oracle 0,
R2b muet 0, oracle muet (film) 0, oracle muet (record) 0
```

La reconciliation, chiffre pour chiffre : sur les 4 278 records CIBLES (sans ancre), R2b en lit
**4 277** (le seul restant, `97b34406`, est l'« arme sans grenade » isole de la table ci-dessus).
En y ajoutant les **102** records ANCRES A SOMME NULLE, qui n'appartiennent PAS aux 4 278 (ils
ont une ancre) mais que R2b lit aussi puisque R2a y echoue (`!inv.GrenadesRead`) :

	4 277 (cibles lus) + 102 (ancres somme nulle, lus) = 4 379

— exactement le chiffre publie par cette table pour R2b. Le §2.1 et le §4 ne se contredisent
donc pas : le premier borne son propos a la population qu'il visait (sans ancre), le second
publie le rendement TOTAL de la voie R2b, qui deborde legerement ce perimetre.

Les 11 films muets, un par un : `0873c469` 0 -> 201, `2a4bc093` 0 -> 126, `4013dc34` 0 -> 162,
`4c8d2287` 0 -> 197, `53ce4390` 0 -> 230, `71ad4abd` 0 -> 106, `7da6e3f0` 0 -> 160,
`8778233d` 0 -> 187, `b7b37365` 0 -> 193, `cbd4f623` 0 -> 211, `d40afcfb` 0 -> 158.

### Le film de reference (`000d5950`, golden)

184 records, 150 armes. Grenades lues : **120 -> 150**, soit **exactement le nombre de records
dont les munitions sont lues** — R2b se borne au bloc de munitions, elle rend donc une lecture
partout ou ce bloc existe et nulle part ailleurs. Les 34 records restants sont ceux qui ne
portent AUCUNE arme, deja etiquetes `Empty` par le lot precedent. Selection de grenade (R5) :
92 -> 106, mecaniquement (elle est gatee par la lecture des compteurs).

**Le golden a bouge d'UNE SEULE LIGNE**, et uniquement par ajout :

```diff
-184 etat(s) publie(s) · 120 avec grenades lues · 150 avec munitions lues
+184 etat(s) publie(s) · 150 avec grenades lues · 150 avec munitions lues
```

Aucun autre chiffre du document assemble ne change (tirs, projectiles, vies, morts, poses
d'equipement, episodes de camouflage). Le fixture d'entrees (`inputs_000d5950.bin.gz`) et le
golden ont ete regeneres par la porte prevue (`REPLAY_FILM_DIR=... -update`), jamais a la main.

---

## 5. Ce que ce lot ne dit PAS

- **Que R2b lit le bon champ sur un film d'une AUTRE version du jeu.** La loi est mesuree sur
  24 films du cache courant. La telemetrie par voie existe pour que la question soit posable
  plus tard sur pieces.
- **Que les 656 lectures a compteurs nuls sont toutes des « plus aucune grenade ».** L'oracle des
  types est un oracle d'ENSEMBLE (par film), pas par record : il etablit que R2b lit le champ des
  grenades, pas que chaque zero individuel est exact. Le croisement par vie (compteur qui ne peut
  varier que par lancer, ramassage ou mort) reste a faire — il demande le pont
  `FilmIndex -> slot`, qui vit dans la couche d'assemblage et non dans le decodeur.
- **Pourquoi l'ancre de capacite n'est pas un invariant** (regime bimodal, piste 5 du 24/08).
  Ce lot contourne le probleme pour les grenades ; il ne le resout pas, et la CAPACITE reste
  dependante de l'ancre aux images-cles (elle est rattrapee par le canal i48 des deltas).
- **Pourquoi 8,3 % des images-cles ne portent aucun record de bipede**, et **le departage des
  blocs de munitions multi-parses** : non instruits, comme le 24/08.

## 6. Decouvertes hors perimetre (notees, NON traitees)

1. **Le fixture d'entrees fige n'est PAS sensible au decodage** — c'est ecrit dans sa doc
   (`golden_inputs_test.go` : « un changement du DECODAGE ne le fait pas bouger »), et de fait
   `TestGoldenAssembly` passait au vert alors que le golden portait encore `120`. Le cran de
   rendement `TestInventoryRulesOnRealBinary` a bien attrape le changement, mais le golden
   assemble aurait pu deriver silencieusement de la realite du decodeur. A instruire : le golden
   devrait-il etre regenere systematiquement avec le decodeur, ou son decouplage assume-t-il ce
   decalage ?
2. **`invAmmoSearchSpan = 300` et la fenetre de R2b se recouvrent.** R2b cherche jusqu'a 216 bits
   avant le debut du bloc, lui-meme cherche jusqu'a 300 bits avant la premiere famille d'arme.
   Aucune consequence mesuree, mais les deux profondeurs sont reglees independamment alors
   qu'elles decrivent la meme region du record.
3. **4 films sur 24 ne rendent AUCUN lancer de grenade** (`1c5c10cc`, `97b34406`, `a26dbcdb`,
   `e67ad176`) alors qu'ils portent des grenades a toutes les images-cles. Ce sont les films a
   grande equipe (les plus gros records). Le canal des lancers a donc, lui aussi, son propre trou
   — a instruire dans le chantier des lancers, pas ici.

---

## 7. Reproduire

```bash
# depuis apps/go-api — la loi de position, son temoin, et le transfert aux records sans ancre
CGO_ENABLED=0 INV_CACHE=<repo>/data/cache/film_chunks INV_SAMPLE=24 \
  go test ./internal/analysis/replay/ -run '^TestPositionI22$' -timeout 180m -v

# la taxonomie APRES, par le decodeur de production
CGO_ENABLED=0 INV_CACHE=<repo>/data/cache/film_chunks INV_SAMPLE=24 \
  go test ./internal/analysis/replay/ -run '^TestInventaireApresR2b$' -timeout 180m -v

# l'oracle independant : types portes contre types lances
CGO_ENABLED=0 INV_CACHE=<repo>/data/cache/film_chunks INV_SAMPLE=24 \
  go test ./internal/analysis/replay/ -run '^TestOracleTypesPortesEtLances$' -timeout 180m -v
```

**UN SEUL balayage a la fois par machine** (meme avertissement que le 24/08 : deux process
concurrents sur le meme cache ont deja produit un film a zero image-cle).

## 8. Gates

| gate | resultat |
|---|---|
| `go vet ./internal/analysis/replay/ ./internal/analysis/filmdec/` | propre |
| `go test ./internal/analysis/replay/ ./internal/analysis/filmdec/` | ok (19,7 s / 0,9 s) |
| `go test ./contracttest/...` | ok |
| `golangci-lint run ./internal/analysis/replay/...` | **0 issues** |
| goldens | regeneres par la porte prevue ; diff = 1 ligne, additive |

### Revue adversariale du 2026-08-25 : 4 constats, 4 corriges

1. **Telemetrie non observee** : le slog de `BuildFromFilm` (`build.go` ~l.214-218) ne
   journalisait pas `GrenadesByAnchor` / `GrenadesByPosition`. Ajoutees au meme log
   (`grenadesParAncre`, `grenadesParPosition`), memes conventions que les cles voisines.
2. **Seuil de taille** : `inventory_decode.go` etait a 502 lignes (seuil CLAUDE.md n°5). Le
   bloc MUNITIONS (R3+R4 : `SlotAmmo`, `readAmmo`, `invParseAmmoBlock`, `invSolveAmmoBlock`,
   `invAmmoSearchSpan`) en a ete extrait vers un nouveau fichier voisin,
   `inventory_ammo_rules.go` (156 L), meme precedent que l'extraction de
   `inventory_grenades_rules.go`. `inventory_decode.go` retombe a **360 lignes**. Comportement
   inchange (memes fonctions, memes signatures, meme package).
3. **Doc inversee** : l'en-tete de R5 (`inventory_grenade_selection.go` l.7-17) citait encore
   « 120 records, 69/92 » (mesure pre-R2b). Mise a jour : population passee a 150, cran
   `wantInvGrenadeSel` a 106 avec R2b (les 30 records supplementaires positionnels), la mesure
   d'origine 120/92 restant explicitement datee comme portant sur la seule voie R2a.
4. **Population non controlee + rapport non reconcilie** :
   (a) les ~102 lectures R2b issues de records ANCRES a somme nulle (motif i22 entierement nul
   apres l'ancre) etaient classees `invPosHorsSujet` par `invPosObserve` — ni corpus
   d'entrainement ni controles ne les instruisaient. Nouvelle categorie
   `invPosAncreSommeNulle` + verification dediee (`invPosVerifieAncreSommeNulle`, reutilise
   `invGrenadesNearAmmo` — la fonction de PRODUCTION — et l'oracle des types de
   `TestOracleTypesPortesEtLances`) : rejouee sur le corpus de reference, elle denombre
   **102** records, dont **102** lus par R2b a somme nulle et **0** desaccord avec l'oracle.
   (b) §4 du rapport reconcilie desormais 4 278 (§2.1, cibles) + 102 (ancres somme nulle) - 1
   (l'unique cible non lue, `97b34406`) = **4 379**, le chiffre publie pour R2b — voir la
   section dediee au §4.

`TestPositionI22` rejoue (INV_CACHE local, INV_SAMPLE=24, `-timeout 180m`) : **436,7 s**
(~7,3 min), PASS. `go test ./internal/analysis/replay/... ./contracttest/...` (sans
INV_SAMPLE, gate CI) : ok. `go vet ./...` : propre. `golangci-lint run
--allow-parallel-runners ./internal/analysis/replay/...` : 0 issues (une fin de ligne CRLF
introduite hors de ce lot dans `build.go` a ete normalisee en LF au passage — gofmt l'exigeait,
aucun changement de contenu).
