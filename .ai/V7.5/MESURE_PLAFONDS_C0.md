# Mesure des plafonds — lot C, phase C0 (2026-08-25)

> Phase C0 du lot C « plafonds » de `.ai/V7.5/PLAN_REPLAY2D_NOTION_2026-08-25.md`, point 10 de
> l'encadre Notion REPLAY 2D. **MESURE SEULE** : aucune coupe n'est appliquee, aucun asset
> n'est regenere. Le verdict est au §6, les risques au §7.

## 1. La question posee

> « Pour les maps non validees, voir si on peut dataminer les hauteurs ou le joueur ne se rend
> jamais et virer les toits ou plafonds qui sont au-dessus de cette hauteur. »

Traduite dans la chaine existante, elle est precise et bornee : la cuisson d'un fond de carte
applique deja un plafond, `sol joue + 28 m` (`himap.TrancheDeJeuMax`,
`apps/go-api/internal/himap/rendu.go:44-47`, pose par `Rendu.Plafond`, `rendu.go:268`). Le point
10 revient a remplacer ce **+28 universel** par une valeur **deduite des hauteurs frequentees**,
carte par carte. La question de C0 : cette substitution coupe-t-elle les toits, et seulement eux ?

## 2. La source des hauteurs frequentees

**L'artefact de rejeu cuit porte le z des positions** — pas besoin de redecoder les films.

| quoi | ou |
|---|---|
| champ | `tracks[].points[].z`, `replay.Point.Z` (`apps/go-api/internal/analysis/replay/document_aim.go:50`) |
| type | `float32`, `json:"z,omitempty"`, publie a deux decimales |
| repere | le MEME que la geometrie monde ; c'est ce champ que l'oracle des 29 221 positions confronte au volume reconstruit (`apps/go-api/internal/himap/carte_oracle_gamefiles_test.go:17-32` et `:126`) |
| bornes | `bounds.minZ` / `bounds.maxZ` du document (`replay.Bounds`, `apps/go-api/internal/analysis/replay/document.go:456-457`) |
| corpus | `data/cache/replays/halo_infinite/*.json` — 35 documents lisibles au 2026-08-25 |

Le decodage offline par `filmdec` n'est donc PAS necessaire : il a deja eu lieu, son resultat est
dans l'artefact. C'est aussi ce que fait l'oracle de decoupage des callouts
(`apps/go-api/internal/mapdecoupe/oracle_corpus_test.go:348-387`).

**Le maillon manquant, et il n'est pas gratuit : l'artefact ne nomme pas sa carte.** Le film ne
porte ni carte ni mode (`replay.Document`, champ `MapObjectives`,
`apps/go-api/internal/analysis/replay/document.go:316-322`) ; en production c'est le registre des
matchs qui la nomme (`apps/go-api/internal/service/replay_map_background.go:78-115`). Un
instrument de mesure n'ouvre pas une base pour ca : il reprend la reconnaissance **hors ligne**
etalonnee le 2026-08-16 (paves du designer du tag `levl` + sol joue publie par le fond,
`apps/go-api/internal/mapdecoupe/oracle_corpus_test.go:174-241`).

## 3. « Carte validee » : ce qui existe, et ce qui n'existe pas

**Aucune liste formelle n'existe dans le code ni dans les donnees.** Le verdict est celui de
l'oeil de l'utilisateur, et il est consigne a UN seul endroit — le gate C4 du 2026-08-10,
`.ai/V7.5/cartes/PLAN_PORT_TRIANGLES_GO.md:190-205` :

| verdict utilisateur | cartes |
|---|---|
| nickel | Launch Site, Breaker, Behemoth, Fragmentation, Catalyst, Cliffhanger, Forest |
| « un peu rudimentaire » | Streets, Bazaar |
| **« on ne voit que les toits »**, forme globale correcte | **Illusion, Prism, Aquarius** |
| idem mais partiellement legitime (partie a ciel ouvert) | Forbidden |
| rien d'exploitable -> corrige depuis | Chasm, Highpower |
| non reconnues (jamais jouees) | Recharge, Vagabond |

Le seul signal MACHINE proche est `stats.covered` / `coveredShare` du sidecar de chaque fond
(`replay.MapBackgroundStats`, `apps/go-api/internal/analysis/replay/map_background.go:99-105`),
calcule par la voie de reference contre les toits
(`himap.SeuilCarteCouverte = 1/3`, `apps/go-api/internal/himap/rendu_reference.go:52`). **Il ne
coincide pas avec le verdict de l'oeil** : Launch Site est « nickel » ET `COUVERTE` a 35,9 %,
Breaker est « nickel » a 30,8 %, tandis qu'Aquarius, defectueuse, est a 66,7 %.

**POINT D'ARBITRAGE SUPERVISEUR.** Definition proposee, a valider avant tout C1 :

> Une carte est VALIDEE si l'utilisateur l'a jugee « nickel » au gate C4 du 2026-08-10, ou l'a
> jugee telle a un gate visuel ulterieur. La liste vit dans UN fichier de donnees versionne
> (par exemple un champ `visualVerdict` du sidecar de fond, ecrit a la main apres chaque gate),
> jamais dans un seuil calcule : `coveredShare` mesure une geometrie, pas un jugement.

Sans ce fichier, C1 ne peut pas « exclure les cartes validees du rognage » comme le plan
l'exige : il n'aurait aucun moyen de savoir lesquelles le sont.

## 4. L'instrument

`apps/go-api/cmd/mapplafond-mesure` (Go, hors ligne, GPLv3 comme `cmd/mapfond-build` — il passe
par `internal/himap` -> `internal/ooz`). Il ne coupe rien : il mesure.

```
CGO_ENABLED=1 go run ./cmd/mapplafond-mesure [--maps M1,M2] [--rejeux DIR]
    [--marge 5] [--rapport FICHIER] [--planches DIR] [--sans-jeu]
```

Ce qu'il fait, carte par carte, **une seule carte ouverte a la fois** :

1. **Corpus** (`frequentation.go`) — chaque document de rejeu est lu, resume en histogramme au
   centimetre, puis RELACHE ; les positions brutes ne survivent jamais a leur fichier.
2. **Reconnaissance** (`identite.go`) — la carte d'un film est celle dont les paves du designer
   contiennent >= 80 % de ses positions (altitude comprise) avec 15 points d'avance sur la
   suivante. Sinon le film est ECARTE : mieux vaut une carte sans corpus qu'une hauteur maximale
   venue d'un match joue ailleurs.
3. **Geometrie** (`geometrie.go`) — la carte est cuite par la chaine de PRODUCTION
   (`himap.CuitCarteNative`, `apps/go-api/internal/himap/cuisson.go:134`). Deux lectures : le
   z-buffer (l'altitude de la surface AFFICHEE, par pixel) et les boites monde des instances
   (`Instance.AABBMin/AABBMax`, `apps/go-api/internal/himap/instances.go:58-59`).
4. **Coupe** — pour un seuil donne : part de l'image qui changerait, instances entierement
   au-dessus (supprimees) et a cheval (decapitees).
5. **Faux positifs** — deux detecteurs INDEPENDANTS l'un de l'autre :
   - les **zones NOMMEES** du jeu (tag `levl`, catalogue `map_callouts.json`) dont le plancher
     est au-dessus du seuil. Une zone nommee est un espace de jeu dessine par le designer : elle
     existe que le corpus l'ait visitee ou non. Ce detecteur ne depend pas du corpus.
   - la **stabilite** : de combien `h max` descendrait si UN film manquait au corpus.
6. **Planche** (`planche.go`, `--planches`) — la carte cuite avec, en rouge, les pixels que la
   coupe emporterait. Elle dit OU la coupe mord ; elle ne montre PAS ce qui apparaitrait dessous
   (cela exige de rejouer la cuisson sous un autre plafond — c'est le changement de C1).

**Piege mesure et desamorce** : le balayage des 19 cartes dans un seul processus atteint
**15,5 Go de resident** ; la memoire est rendue au systeme entre chaque carte
(`libereCarte`, `main.go`). Pic par carte inchange (une grande carte BTB coute plusieurs Go a
cuire, c'est le cout de `himap`).

**Reproduction exacte du balayage de ce document** (~10 min, `LEVELUP_REPO_ROOT` parce que ce
worktree n'a pas de `db_profiles.json` local ; `--rejeux` parce que le cache de rejeux vit dans
le worktree principal) :

```
cd apps/go-api
LEVELUP_REPO_ROOT=<worktree> go run ./cmd/mapplafond-mesure \
  --rejeux <depot-principal>/data/cache/replays/halo_infinite \
  --rapport <sortie>.md --planches <sortie>/planches
```

## 5. Les mesures

Balayage du 2026-08-25, marge 5 m, corpus `data/cache/replays/halo_infinite` (depot principal),
19 cartes natives cuites une a une. Sortie brute et planches : `cmd/mapplafond-mesure --rapport
... --planches ...`. Trois cartes du catalogue de callouts n'ont pas de fond publie et sont hors
mesure (`academy_tutorial`, `pve_house`, `sgh_interlock`).

### 5.1 Ce que le corpus couvre — et ce qu'il ne couvre pas

| | |
|---|---:|
| documents de rejeu lus | 35 |
| rattaches a une carte | 11 |
| cartes natives avec au moins un film | **8 sur 19** |
| cartes natives avec au moins DEUX films | **3** (catalyst, ridgeline, sgh_blueprint) |

Les 24 films non rattaches ne sont pas un echec de la regle : **21 d'entre eux ont une altitude
mediane entre 52,6 et 176,6 m**, la plage exacte des sols joues des canevas FORGE (52,1 a
176,2 m au sidecar). Une carte Forge n'a AUCUNE zone nommee — elle ne peut pas etre reconnue par
cette voie, et elle n'a de toute facon pas de geometrie mesurable (§7 R1). Les 3 restants sont
des films natifs ecartes faute de marge : `64e8adfa` (catalyst 90 % contre ctf_breaker 77 %),
`82f29378` (ctf_breaker 92 % contre btb_exiled 86 %), `846044ba` (va_behemoth 60 %).

**Aucune des trois cartes que l'utilisateur a jugees « on ne voit que les toits » (Illusion,
Prism/`sgh_crystalcaves`, Aquarius) n'a un seul film dans le corpus local.** La hauteur
frequentee ne peut pas y etre calculee.

### 5.2 Ce que couperait un plafond a « hauteur max frequentee + 5 m »

| carte | films | positions | sol joue | h med | h p99 | h max | plafond actuel | seuil | image changee | volumes coupes | zones nommees au-dessus | perte si -1 film |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---|---|---:|
| `btb_highpower` | 1 | 23 285 | 44,1 | 42,9 | 50,0 | 58,9 | 72,1 | 63,9 | 4,57 % | 175 supprimes (2,06 %) · 343 decapites (4,05 %) / 8 479 | 0 | — |
| `catalyst` | 2 | 55 209 | 24,4 | 25,2 | 27,8 | 30,7 | 52,4 | 35,7 | **37,54 %** | 139 supprimes (1,72 %) · 571 decapites (7,05 %) / 8 099 | 0 | 1,2 |
| `chasm` | 1 | 10 915 | -136,7 | -138,0 | -134,0 | -132,8 | -108,7 | -127,8 | 15,39 % | 333 supprimes (14,89 %) · 160 decapites (7,15 %) / 2 237 | 0 | — |
| `ctf_bazaar` | 1 | 10 457 | 1,2 | 1,2 | 3,9 | 4,3 | 29,2 | 9,3 | 2,67 % | 141 supprimes (1,47 %) · 183 decapites (1,91 %) / 9 585 | 0 | — |
| `ctf_breaker` | 1 | 30 224 | 16,1 | 16,4 | 21,2 | 25,2 | 44,1 | 30,2 | 6,69 % | 18 supprimes (0,22 %) · 62 decapites (0,75 %) / 8 281 | 0 | — |
| `ridgeline` | 2 | 48 156 | -2,5 | -1,8 | 2,0 | 7,1 | 25,5 | 12,1 | **23,21 %** | 85 supprimes (1,36 %) · 362 decapites (5,81 %) / 6 231 | 0 | 0,0 |
| `sgh_blueprint` | 2 | 37 952 | 1,9 | 2,2 | 5,2 | 5,5 | 29,9 | 10,5 | 13,00 % | 27 supprimes (1,02 %) · 90 decapites (3,38 %) / 2 660 | 0 | 0,2 |
| `va_behemoth` | 1 | 39 959 | 8,7 | 8,4 | 13,2 | 14,7 | 36,7 | 19,7 | 16,06 % | 2 supprimes (0,03 %) · 47 decapites (0,66 %) / 7 101 | 0 | — |

Les onze autres cartes natives cuites n'ont aucun film : elles n'ont ni `h max`, ni seuil, ni
coupe. Elles apparaissent quand meme au §5.3, qui se lit sans corpus.

**La coupe est chiffree en VOLUMES et en PIXELS, pas en faces.** Le plan demandait « % de faces
rognees » ; compter les triangles exigerait de decoder le maillage de chaque instance, alors que
la boite monde de l'instance suffit a dire ce qu'un plafond supprime et ce qu'il decapite, et que
le z-buffer dit exactement ce que l'image perd. Les deux chiffres publies repondent a la question
posee ; un compte de triangles ne l'aurait pas mieux tranchee.

**Deux faits sortent de ce tableau.**

1. **Le seuil est TRES au-dessous du plafond actuel** — de 8,2 m (highpower) a 19,9 m (bazaar).
   Ce n'est pas un reglage fin, c'est un autre regime de coupe.
2. **Aucune ZONE NOMMEE ne passe au-dessus du seuil, sur aucune carte.** Le detecteur
   independant du corpus ne signale AUCUN etage praticable perdu. La coupe ne detruit donc pas
   d'espace de jeu declare — ce qu'elle detruit est du DECOR et des TOITURES, et c'est la que
   se joue le verdict.

### 5.3 Ou vit la matiere — le tableau qui tranche

Centiles de l'altitude des pixels porteurs de matiere, **en metres au-dessus du sol joue**. Il
se lit SANS corpus, donc il couvre aussi les cartes visees par le point 10.

| carte | sol joue | matiere p50 | p90 | p99 | max | h max - sol | seuil - sol |
|---|---:|---:|---:|---:|---:|---:|---:|
| `btb_drydock` | 75,8 | 1,6 | 11,6 | 18,3 | 28,0 | — | — |
| `btb_engine` | -0,7 | 0,7 | 5,2 | 15,4 | 27,4 | — | — |
| `btb_exiled` | 24,3 | 1,2 | 10,3 | 26,1 | 28,0 | — | — |
| `btb_fragmentation` | 5,8 | 1,3 | 15,6 | 22,4 | 28,0 | — | — |
| `btb_highpower` | 44,1 | 1,0 | 14,5 | 25,7 | 28,0 | 14,9 | 19,9 |
| `catalyst` | 24,4 | 5,7 | 17,5 | 19,3 | 28,0 | 6,3 | 11,3 |
| `chasm` | -136,7 | 0,1 | 12,8 | 22,8 | 27,3 | 3,9 | 8,9 |
| **`ctf_aquarius`** | 4,1 | -0,9 | **1,3** | **7,3** | **9,9** | — | — |
| `ctf_bazaar` | 1,2 | -0,4 | 6,4 | 10,2 | 13,9 | 3,1 | 8,1 |
| `ctf_breaker` | 16,1 | 0,3 | 10,2 | 17,8 | 21,5 | 9,1 | 14,1 |
| `ctf_forbidden` | 1,9 | 1,1 | 8,7 | 9,0 | 27,4 | — | — |
| **`ctf_illusion`** | 0,5 | 2,5 | 8,3 | **11,7** | 20,8 | — | — |
| `forest` | 1,8 | 1,3 | 10,9 | 21,0 | 28,0 | — | — |
| `ridgeline` | -2,5 | 2,6 | 21,0 | 27,2 | 28,0 | 9,5 | 14,5 |
| `sgh_blueprint` | 1,9 | 4,5 | 9,6 | 13,4 | 13,6 | 3,6 | 8,6 |
| **`sgh_crystalcaves`** | 17,7 | 0,6 | **5,9** | 24,5 | 28,0 | — | — |
| `sgh_streets` | 0,7 | 3,6 | 8,2 | 10,9 | 13,5 | — | — |
| `va_behemoth` | 8,7 | 0,6 | 12,0 | 25,5 | 28,0 | 5,9 | 10,9 |
| `va_launchsite` | -1,3 | 1,5 | 8,9 | 21,2 | 28,0 | — | — |

**Le seuil qu'un corpus donne a une arene est CONNU : `h max - sol` vaut de 3,1 a 9,5 m sur les
huit cartes mesurees, donc le seuil tombe entre sol + 8,1 et sol + 14,5 m.**

Confrontons-le aux trois cartes « toits » :

- **Aquarius** (66,7 % de matiere couvrant un sol praticable, la plus couverte des 19 natives) : sa matiere
  s'arrete a **sol + 9,9 m**, et **99 % de cette matiere est sous sol + 7,3 m**. Un seuil pose
  entre +8,1 et +14,5 passe AU-DESSUS de la carte entiere : il ne coupe rien.
- **Illusion** : p99 a sol + 11,7 m — un seuil a +8,1 mordrait quelques pour cent, un seuil a
  +14,5 rien du tout.
- **Prism** (`sgh_crystalcaves`) : p90 a sol + 5,9 m ; la coupe mordrait, mais entre p90 et p99,
  donc au plus quelques pour cent.

La raison est structurelle et le tableau la rend visible : **le toit d'une arene couverte n'est
pas haut. Il est juste au-dessus du sol qu'il cache** — dans la meme bande de 0 a 10 m que le
jeu lui-meme, c'est-a-dire exactement la ou tombe « hauteur frequentee + marge ».

### 5.4 Les deux cartes temoins

Planches versionnees dans `.ai/V7.5/dumps/plafonds_c0/` : la carte cuite par la chaine de
production, avec en ROUGE les pixels que la coupe emporterait.

**Temoin 1 — `catalyst`, etages praticables hauts, carte jugee « nickel » au gate C4.** Seuil
sol + 11,3 m, **37,54 % de l'image change**, 571 volumes decapites. La planche
(`catalyst_coupe.png`) montre ce qui part : la TOITURE de toute la structure centrale et le
plateau ouest. Ce ne sont pas des plafonds parasites — ce sont les toits qui donnent a la carte
sa lecture en vue de dessus. La coupe la transformerait en radiographie.

**Temoin 2 — `chasm`, carte COUVERTE (37,2 %), re-cuite au lot toits et validee le 2026-08-13.**
Seuil sol + 8,9 m, 15,39 % de l'image, et **14,89 % des volumes entierement supprimes** — la
part la plus forte des huit. La planche (`chasm_coupe.png`) montre ce qui part : les TETES DES
PILIERS du viaduc, une par une. Le viaduc resterait, troue.

**Reference — `ridgeline` (Cliffhanger), la carte de reference du chantier, jugee « nickel ».**
Seuil sol + 14,5 m, 23,21 % de l'image. La planche (`ridgeline_coupe.png`) montre ce qui part :
les MASSES ROCHEUSES qui ceinturent l'arene — celles dont l'investigation des toits ecrivait
deja, le 2026-08-13, qu'elles sont « ses rochers valides » et que nul seuil par pixel ne doit
epargner.

### 5.5 Variante au 99e centile — elle aggrave, elle ne sauve pas

Ignorer le dernier pour cent des positions (grappin, canon, chute) abaisse le seuil, donc coupe
davantage : catalyst passe de 37,54 a **44,19 %**, ridgeline de 23,21 a **33,31 %**, behemoth de
16,06 a 22,43 %. Et le seul faux positif du balayage apparait la : sur `ctf_breaker`, le seuil
p99 passe sous le plancher d'une zone NOMMEE, **« Promontoire »**. La variante p99 est donc
ecartee par la mesure meme.

## 6. Verdict

**NON CONCLUANT pour le besoin exprime, et REFUTE comme regle generale.** Trois constats
independants, chacun suffisant.

**1. Sur les cartes ou le defaut existe, la coupe ne peut pas mordre.** Le toit d'une arene
couverte vit dans la meme bande d'altitude que le sol qu'il cache : Aquarius, la plus couverte
des 19 cartes natives (66,7 % de matiere couvrante), a **toute** sa matiere sous sol + 9,9 m et 99 % sous
sol + 7,3 m. Le seuil qu'un corpus donne a une arene tombe entre sol + 8,1 et sol + 14,5 m — au-
dessus. Ce n'est pas un probleme d'etalonnage : un plafond est une regle GLOBALE d'altitude, et
« du toit au-dessus d'un sol » est une relation LOCALE, par pixel. C'est exactement la relation
que `himap/rendu_reference.go` mesure et corrige depuis le 2026-08-13.

**2. Sur les cartes ou la coupe mord, elle mord de la geometrie legitime, sur des cartes
validees.** Catalyst 37,5 % de l'image (ses toitures), Cliffhanger 23,2 % (ses rochers
d'identite), Chasm 14,9 % de ses volumes (les tetes de piliers du viaduc). Les trois sont
validees par l'utilisateur. Le detecteur de zones nommees ne les protege pas : il ne signale
aucun etage praticable perdu (0 partout au seuil `h max + 5`), parce que ce qui part n'est pas
un etage — c'est le decor et la toiture.

**3. Sur les cartes reellement NON VALIDEES, la mesure est impossible aujourd'hui.** Les cartes
refusees au dernier gate sont les 37 fonds `map_id` (Forge, masse refusee le 2026-08-13 :
« on voit encore trop les plafonds et toits »). Aucune n'a de zone nommee (donc aucun film
rattachable : 21 des 24 films ecartes sont a l'altitude des canevas Forge) et aucune n'est
re-cuisinable (les `.mvar` du depot de variantes sont gitignores et absents). Ni la hauteur
frequentee ni la geometrie ne sont accessibles pour elles.

**Ce que la mesure ne refute PAS**, et qui reste ouvert si le superviseur veut poursuivre :

- une coupe **par carte non validee seulement**, decidee a la main plutot que par le corpus,
  reste techniquement possible sur les cartes NATIVES — mais elle n'a plus rien de « datamine »,
  et le §5.3 dit qu'elle ne rendrait presque rien sur les trois cartes visees ;
- le vrai levier mesure existe deja et il est LOCAL : la voie de reference ne substitue que
  **dans la portee des ancres** (`PorteeAncre` = 25 m, `himap/reference.go:34`). Le residu de
  toits, s'il en reste, est donc a la PERIPHERIE. Elargir ou densifier les ancres est une piste
  bien plus proche de la cause que n'importe quel plafond ;
- pour les cartes FORGE, l'hypothese deja au registre (« coque de blocs fermee dans le vide »)
  n'est ni confirmee ni infirmee ici : elle exige les `.mvar`.

**Recommandation : NE PAS ouvrir C1 en l'etat.** Trois prealables, dans cet ordre :

1. **Confirmer le perimetre avec l'utilisateur** : quelles cartes juge-t-il encore mauvaises
   AUJOURD'HUI ? Le grief natif du 2026-08-10 a ete solde par le gate du 2026-08-13 ; le grief
   vivant porte sur les fonds Forge.
2. **Restaurer `.ai/re_dump/mapvar`** depuis la sauvegarde externe — sans quoi aucune carte
   Forge ne peut etre ni mesuree ni re-cuite.
3. **Trancher la definition de « carte validee »** (§3) et l'ecrire en DONNEE, sinon C1 ne peut
   pas exclure les cartes validees du rognage comme le plan l'exige.

## 7. Risques identifies

**R1 — Le besoin porte sur des cartes que RIEN ne permet de re-cuire aujourd'hui.** Les cartes
non validees au dernier gate sont les 37 fonds `map_id` (Forge). Les re-cuire exige les `.mvar`
du depot de variantes (`himap.DepotVariantesCarte = ".ai/re_dump/mapvar"`,
`apps/go-api/internal/himap/cartes_forge.go:299`). Ce dossier est **gitignore**
(`.gitignore:254`, « restaurees depuis sauvegarde externe, jamais versionnees ») et **vide**,
dans ce worktree comme dans le depot principal ; la cle de sauvegarde `E:` n'est pas montee.
Sans lui, ni C0 ni C1 ne peuvent toucher une seule carte Forge.

**R2 — Le corpus ne couvre pas les cartes visees, et ne le pourra pas par cette voie.** Une
carte Forge n'a AUCUNE zone nommee (fait de construction, `replay.ErrCalloutsUnknownMap`) : la
reconnaissance hors ligne n'a aucun appui sur elles. Les films Forge du corpus restent donc non
rattaches, et leur hauteur frequentee n'est pas calculable sans une autre regle.

**R3 — Sur les cartes mesurables, la coupe mord la geometrie LEGITIME.** Les planches le
montrent sans ambiguite (§5.4) : rochers d'identite sur Cliffhanger, toitures de la structure
centrale sur Catalyst, tetes de piliers du viaduc sur Chasm. Aucune de ces cartes n'est
« polluee par un toit » — les trois sont validees par l'utilisateur.

**R4 — L'estimateur `h max` n'est pas verifiable sur la majorite des cartes mesurees** : cinq
des huit cartes a corpus n'ont qu'UN film. Le controle « retirer un film » y est impossible, et
un plafond pose sur une seule partie est un plafond pose sur un echantillon de un.

**R5 — Le garde-fou des zones nommees disparait exactement la ou il faudrait.** Il ne connaît
que les 22 cartes natives du catalogue de callouts. Sur une carte Forge, aucun faux positif ne
serait detecte : la coupe s'y appliquerait a l'aveugle.

**R6 — La cuisson mesuree n'est pas celle qui a produit les PNG publies.** Le catalogue
d'objectifs a ete re-tire du reseau le 2026-08-25 (`d50f3b728`) et les ancres ont change :
Cliffhanger 24 au lieu de 14, Catalyst 29 au lieu de 24, Highpower 81 au lieu de 51. Le cadre,
le sol joue et la frontiere qui en decoulent bougent donc legerement. Les chiffres de ce
document decrivent la cuisson d'AUJOURD'HUI ; les PNG publies datent du 2026-08-12.

**R7 — Le grief d'origine sur les cartes NATIVES a peut-etre deja ete solde.** Les trois cartes
« on ne voit que les toits » du gate du 2026-08-10 (Illusion, Prism, Aquarius) ont ete re-cuites
par la voie de reference et le gate du lot toits a ete **VALIDE par l'utilisateur le
2026-08-13** (« excellent travail », `.ai/V7.5/REGISTRE_REPORTS.md` ligne 112). Leurs sidecars
publies portent la substitution (`cellsSubstituted` 116 018 / 152 000 / 310 884). Avant tout
C1, confirmer avec l'utilisateur quelles cartes il juge encore mauvaises AUJOURD'HUI.

## 8. Decouvertes (hors perimetre — NON traitees)

**D1 — `map_objectives.json` a perdu le lien entree -> module installe.** Le re-tirage du
2026-08-25 (`d50f3b728`) porte `module: "map"` et `mvar_file: "map.mvar"` sur **63 des 73
entrees**. `cmd/mapfond-build` resout ses cibles par ce champ
(`ciblesNatives` -> `himap.ChercheModuleInstalle(entree.Module)`, puis le repli par le depot de
variantes, `cmd/mapfond-build/resolution.go`) : aucune des deux voies ne resout `"map"`, et le
depot de variantes est vide. **Une re-cuisson des fonds aujourd'hui declarerait « non
installees » toutes ces cartes.** L'instrument contourne en passant par le catalogue de bornes
(nom affiche -> module) ; `mapfond-build`, lui, n'a pas ete touche.

**D2 — Le nombre d'ancres a change avec ce meme re-tirage** (cf. R6). Toute re-cuisson repartira
d'un cadre et d'un sol joue differents de ceux des PNG publies : a verifier avant de republier
quoi que ce soit.

**D3 — Aucune carte Forge n'est re-cuisinable** tant que `.ai/re_dump/mapvar` n'est pas restaure
depuis la sauvegarde externe. Le lot « bouillie Forge » du registre (ligne 114, gate map_id
refuse en masse) est bloque par la meme cause.

**D4 — Le balayage des 19 cartes dans un processus atteint 15,5 Go de resident.** Desamorce dans
l'instrument (memoire rendue entre chaque carte) ; `cmd/mapfond-build` cuit la meme sequence sans
cette precaution.
