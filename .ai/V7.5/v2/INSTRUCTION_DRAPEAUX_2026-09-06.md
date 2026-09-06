# Instruction du fait « un joueur porte son propre drapeau » — 2026-09-06

Branche `feat/v2-drapeaux`, worktree `LevelUp-wt-v2-drapeaux`, base `7e5c454bc` (= `feat/v2-durees`,
SchemaVersion 45, revue DUREES-R1 close). Fait ouvert par la **decouverte n° 1** de
`.ai/V7.5/v2/INSTRUCTION_DUREES_2026-09-06.md`, laissee non traitee par ce lot-la.

---

## Verdict en une page

**DEFAUT DU CALQUE, corrige a la source.** L'etiquette d'equipe du drapeau est JUSTE ; c'est
l'attribution du portage qui etait fausse, pour deux raisons cumulees dans `assignFlags` :

| | Cause | Nature |
|---|---|---|
| **(1)** | L'etat « ou git chaque drapeau » se tenait a jour **en parcourant les prises**, si bien que la position de LACHER d'un portage y etait inscrite des son attribution — donc avant d'avoir eu lieu. Deux portages qui se recouvrent suffisent. | erreur d'**ORDRE** |
| **(2)** | Le repli sur le socle le plus proche, juste pour un VOL (il se fait a un socle), est **faux pour une PRISE** : elle se fait la ou l'objet est tombe, souvent pres du socle ADVERSE — c'est-a-dire du socle du camp qui marque, donc du porteur. | **REGLE manquante** |

Sur `bcb6d393` : **15 portages sur le drapeau de l'equipe 1 et 1 sur celui de l'equipe 0**
deviennent **16 sur 16 pour l'equipe 1, zero pour l'equipe 0**, et les **trois** captures se
publient sur le drapeau adverse (une l'etait sur le drapeau du camp qui marquait). Schema **45 ->
46**.

Deux hypotheses de depart sont **REFUTEES** : le 16e portage n'est ni un retour de drapeau ni un
artefact de joueur entre/sorti en cours de partie (§4), et l'etiquette d'equipe du catalogue de
carte n'est pas decalee par rapport a la feuille de match (§2 — preuve par la geometrie des trois
livraisons).

La premisse « une seule des trois captures est liee a un portage » est, elle, **a corriger** : les
trois portages captures existaient deja (`captured = true` sur les trois), et les trois `home`
etaient publies. Ce qui manquait, c'est que l'un des trois l'etait **sur le mauvais drapeau** (§3).

---

## Methode

Cuisson de production (`cmd/replay-build`), **un film par processus**, sous verrou d'exclusion
`filmproc.AcquireSolo` + verrou inter-agents (`mkdir`/`rmdir` sur le scratchpad), plafond memoire
3 Gio, priorite basse. Racine de travail dediee : `data/cache/{film_chunks,film_manifests,mvar}` et
`data/titles` en JONCTIONS vers le parc (lecture), `config/` en jonction vers le checkout teste,
`data/cache/replays` en VRAI dossier — **le parc n'a recu aucune ecriture** (verifie : `git status`
de `data/` vide des deux cotes).

Faits de match : `scratchpad/balayage/facts/*.facts.json`, exportes en lecture seule par
`levelup replay-facts-export` lors du balayage du parc.

Comparaison axe par axe : `cmd/replay-diff`. Diagnostic interne : **une sonde temporaire** posee
dans `assignFlags` (un `slog.Warn` par portage), cuisson, lecture du journal, puis
`git checkout --` du fichier — verifie identique a l'octet pres avant toute modification reelle.

---

## 1. A quel objet le film rattache un portage, et d'ou vient l'etiquette

**Le film ne rattache un portage a AUCUN objet.** Un portage est un compteur de statistique du
statborg (`flag_grabs` / `flag_steals`) sur un SLOT, plus la trajectoire de son porteur. L'objet
drapeau ne se lit que **libre** (`flag_objects.go`, vies libres portant `ID` = mot MPP et
`Key` = (slot, generation)) — et meme alors il ne porte ni equipe ni camp. `Track.Team` n'est pas
dans le film non plus. Il n'existe donc, dans le film, **aucune chaine qui nomme le drapeau d'un
portage**.

L'etiquette vient du **catalogue versionne de carte**
(`data/titles/halo_infinite/reference/map_objectives.json`), lu par `replaybuild.flagSpawns` : role
`flag_spawn`, champ `team_index`. Pour Cliffhanger (`mapId 5324364b-39a8-4f93-96a6-b80a1f18ce8a`) :

| socle | `team_index` | position |
|---|---|---|
| `flag_spawn` | **1** | (-7,30 · -1,79) |
| `flag_spawn` | **-1** (neutre, centre) | (18,44 · -20,81) |
| `flag_spawn` | **0** | (41,03 · 9,72) |

Le socle neutre est ecarte par `flag_neutral.go` (naissances aux socles d'equipe = 8, au neutre =
0 : la partie n'est pas a drapeau neutre). Restent deux drapeaux, dans l'ordre du catalogue :
**index 0 = equipe 1**, **index 1 = equipe 0**. Le rattachement portage -> index est ensuite
**purement geometrique** (`assignFlags`).

## 2. L'etiquette d'equipe est JUSTE — preuve independante par la livraison

La feuille de match `facts/bcb6d393.facts.json` donne `teamScores = [3, 0]` et place les
**quatre** porteurs dans l'equipe **0** (`2533274823110022`, `2533274858283686`,
`2535429985869093`, `2535469190789936`).

En CTF on livre le drapeau adverse **a sa propre base**. Les trois portages captures se terminent
en :

| capture | instant | porteur | equipe | position de fin | distance au socle `team_index` 0 (41,03 · 9,72) |
|---|---|---|---|---|---|
| 1 | 188 073 ms (frame 1751) | `2533274858283686` | 0 | (40,59 · 9,95) | **0,50 m** |
| 2 | 259 946 ms (frame 2470) | `2533274823110022` | 0 | (41,46 · 9,27) | **0,62 m** |
| 3 | 351 838 ms (frame 3389) | `2535429985869093` | 0 | (40,37 · 9,85) | **0,67 m** |

Les trois livraisons de l'equipe 0 se font au socle que le catalogue etiquette `team_index = 0`.
Le catalogue et la feuille de match **coincident donc**, et le `flag_delivery` du meme catalogue le
confirme (equipe 1 en (-7,30 · -1,79), equipe 0 en (41,02 · 9,73) — chaque zone de livraison est
au socle de son camp).

**Consequence** : le drapeau porte 15 fois est bien celui de l'equipe 1, et le 16e portage, pose
sur le drapeau de l'equipe 0 par un joueur de l'equipe 0, est **impossible**. L'etiquette n'est pas
en cause : l'attribution l'est.

## 3. Les trois captures, et pourquoi une seule paraissait liee

Le calque des ACTIONS (`objectives`, cuisson au schema 45) publie **trois** `flag_captures`, tous
a des joueurs de l'equipe 0 — le score `3-0` est donc entierement explique :

```
frame 1751 (188 073 ms)  flag_captures  2533274858283686   + 2 flag_capture_assists
frame 2470 (259 946 ms)  flag_captures  2533274823110022
frame 3389 (351 838 ms)  flag_captures  2535429985869093
```

La sonde montre que **les trois portages correspondants portaient deja `captured = true`**
(indices 4, 6 et 15 ci-dessous) et que **les trois `home` etaient publies**. La premisse « une
seule capture est liee » vient d'une lecture du document : seul le portage du drapeau « equipe 0 »
presentait la sequence lisible `carried` -> `home`, les deux autres etant separes de leur `home`
par un `dropped` d'une frame (cf. decouverte n° 2). Le releve exact, AVANT correctif :

| capture | ou le `home` etait publie |
|---|---|
| frame 1751 | **drapeau 1 (equipe 0)** — le camp qui marque, donc FAUX |
| frame 2470 | drapeau 0 (equipe 1) — juste |
| frame 3389 | drapeau 0 (equipe 1) — juste |

Le drapeau de l'equipe 1, prive de sa capture a 1751, rentrait chez lui par une **rentree
d'objet** (`homeByObject`) : une inference tenait la place d'un fait date. Apres correctif,
`homeByObject` passe de **2 a 1** — la capture reprend sa place et la rentree devient inutile.

## 4. Le 16e portage : un VRAI portage du drapeau adverse, mal etiquete

Sonde dans `assignFlags` (extrait, cuisson au HEAD 45 ; `drop0` = position du drapeau 0 « au sol »
telle que la boucle la croyait, `viaDrop` = index rendu par la regle du drapeau au sol) :

```
i=2  2533274858283686  t0=143777  t1=180531  steal=false capt=false  x0=(-0.05,-5.19) x1=(33.61,2.48)  viaDrop=0  fi=0  drop0=&[1.97 -7.35]
i=3  2535429985869093  t0=171941  t1=325913  steal=false capt=false  x0=(29.86,-3.75) x1=(-6.53,-2.19) viaDrop=0  fi=0  drop0=&[33.61 2.48]
i=4  2533274858283686  t0=180531  t1=188073  steal=false capt=TRUE   x0=(33.61,2.48)  x1=(40.59,9.95) viaDrop=-1 fi=1  drop0=&[-6.53 -2.19]
```

Trois faits se lisent sur ces trois lignes :

1. **La prise n'est pas un vol** : `steal = false` — c'est un `flag_grabs`, donc le ramassage d'un
   drapeau DEJA au sol. Le film ne publie aucun `flag_steals` a cet instant (4 vols au total :
   frames 471, 1084, 2042, 3130).
2. **Elle ramasse le drapeau que le portage precedent vient de poser** : `x0 = (33,61 · 2,48)` est
   **exactement** `x1` du portage `i=2`, au meme instant `180 531 ms` — **distance 0 m**, quand le
   rayon de ramassage est de 8 m.
3. **Le sol mentait** : `drop0 = (-6,53 · -2,19)`, c'est-a-dire la position de lacher du portage
   `i=3`, **qui ne se ferme qu'a 325 913 ms — 145 secondes plus tard**. `i=3` s'ouvre avant `i=4`,
   donc l'ancienne boucle, qui parcourait les PRISES, avait deja inscrit son lacher futur. La
   regle du drapeau au sol echoue (`viaDrop = -1`), et le repli geometrique tranche :

   | | distance depuis (33,61 · 2,48) |
   |---|---|
   | socle equipe 1 (-7,30 · -1,79) | **41,13 m** |
   | socle equipe 0 (41,03 · 9,72) | **10,37 m** |

   -> `fi = 1`, le drapeau de l'equipe **0**, celle du porteur.

**Ce n'est donc pas un artefact.** Les trois hypotheses concurrentes tombent :

- **retour de drapeau** : un `flag_returns` n'ouvre aucun portage (le calque ne lit que
  `flag_grabs` / `flag_steals`) ; les deux `flag_returns` du film sont a 1556 et 2174 ;
- **joueur entre en cours de partie** : les trois `joinedInProgress` (`2533274811363842` a
  282 262 ms, `2535418713587213`, `2535468064146356`) et les trois `leftInProgress`
  (`2533274876732804`, `2535460750735339`, `2791963697577117`) sont **tous de l'equipe 1**, et
  **aucun** ne porte le drapeau — les 16 portages sont a quatre joueurs de l'equipe 0 ;
- **slot re-attribue** : la couverture publie `noBridge = 0` et `noTrack = 0`, le film est
  MONO-MANCHE, et le portage se ferme sur la capture du MEME slot 8 s plus tard.

Le 16e portage est le dernier relais d'une course a trois : `i=2` porte le drapeau de l'equipe 1
jusqu'a 10 m de la base adverse, meurt ; `i=4` le ramasse a 0 m de la ou il est tombe et le livre.

## 5. Le correctif

Nouveau fichier `internal/analysis/replay/flag_assign.go` — deplacement PUR de `assignFlags`,
`nearestDroppedFlag` et `nearestSpawn` (`flag_carries.go` passe de 451 a 396 lignes), plus deux
changements :

1. **Le parcours se fait par EVENEMENTS DATES** (`flagGroundTimeline`) : une prise enleve le
   drapeau du sol, une fin l'y repose — ou le renvoie a sa base quand elle est une capture. A
   instant egal, **la fin passe avant la prise**, le meme ordre qu'`assembleFlagLives`
   ([flagLifeClose] avant [flagLifeOpen]) et pour la meme raison : un drapeau libere dans la meme
   milliseconde se reprend tout de suite — c'est le cas NOMINAL ici (180 531 ms).
2. **Une troisieme regle** : une PRISE que le sol ne rattache a rien va au drapeau **DEJA EN JEU**
   quand il est le SEUL. `enJeu[f]` = le drapeau a quitte son socle et n'y est pas rentre ; une
   capture l'y ramene. A deux drapeaux dehors, la regle **se tait** et le socle le plus proche
   reprend la main — on RETRECIT le repli, on ne le supprime pas.

L'ordre des trois regles est fige : **vol -> socle** ; **prise -> drapeau au sol a moins de 8 m** ;
**prise -> seul drapeau en jeu** ; sinon socle.

Limite ecrite dans le code : les retours credites (`flag_returns`) et les rentrees d'objet ne sont
pas connus a cet endroit (ils vivent dans `assembleFlagLives`, en aval). Ils ne peuvent donc que
laisser un drapeau « en jeu » **de trop**, ce qui fait TAIRE la troisieme regle au lieu de la faire
mentir.

## 6. Tests, prouves par mutation

`internal/analysis/replay/flag_assign_test.go` — quatre tests sur la geometrie de `bcb6d393`
reduite au banc (socle du camp qui subit en (0,0), socle du camp qui marque en (100,100)) :

| test | ce qu'il fige |
|---|---|
| `TestFlagAssignLeSolSuitLeTempsEtNonLOrdreDesPrises` | la cause (1). Les DEUX drapeaux sont dehors, donc la regle du « seul drapeau en jeu » est muette : seul l'ordre du temps peut repondre |
| `TestFlagAssignPriseVaAuSeulDrapeauEnJeu` | la cause (2). Le sol ne rattache rien, la prise est a 17 m du socle du porteur et a 124 m de l'autre |
| `TestFlagAssignADeuxDrapeauxEnJeuLaRegleSeTait` | **la contre-epreuve** : a deux drapeaux dehors, le socle reprend la main |
| `TestFlagAssignLeVolResteAuSocle` | l'ORDRE des regles : un vol prend le drapeau de son socle meme si l'autre git a ses pieds |

**Quatre mutations jouees, chacune ROUGE puis VERTE** :

```
M1  parcours par les PRISES (lacher inscrit d avance)
    --- FAIL: TestFlagAssignLeSolSuitLeTempsEtNonLOrdreDesPrises
        drapeau d'equipe 1 porte par [1 2], attendu [1 2 3]
M2  regle du « seul drapeau en jeu » retiree
    --- FAIL: TestFlagAssignPriseVaAuSeulDrapeauEnJeu
        drapeau d'equipe 1 porte par [1], attendu [1 2]
M3  seulEnJeu devient un fourre-tout (rend le PREMIER en jeu)
    --- FAIL: TestFlagAssignADeuxDrapeauxEnJeuLaRegleSeTait
        drapeau d'equipe 1 porte par [1 3], attendu [1]
M4  le vol ne court-circuite plus le sol
    --- FAIL: TestFlagAssignLeVolResteAuSocle    (+ 2 autres)
        drapeau d'equipe 1 porte par [1 2], attendu [1]
```

## 7. Mesure — cinq films re-cuits des deux cotes

Invariants du controle : **(A)** aucun portage sur le drapeau de sa propre equipe ;
**(B)** nombre de captures = score.

| film | variante | score | fautes (A) avant | fautes (A) apres | captures / score |
|---|---|---|---|---|---|
| **`bcb6d393`** | CTF:Arena | 3-0 | **1** | **0** | 3 = 3 — conforme |
| **`e94163af`** | CTF:Arena Neutral Flag | 1-5 | 0 | 0 | 6 = 1+5 — conforme (drapeau neutre : (A) sans objet) |
| **`c0a82e88`** | Husky Raid:CTF | 3-0 | 0 | 0 | 1 publiee (2 prises `noBridge`) |
| `cde26226` | CTF:Arena | 2-3 | 0 | 0 | 4 — conforme |
| `64e8adfa` | CTF:Arena, **2 manches** | 2-3 | **13** | **7** | 3 — conforme |

Par drapeau sur `bcb6d393` :

| | avant (schema 45) | apres (schema 46) |
|---|---|---|
| drapeau 0 — equipe **1** | 15 portages, 33 spans | **16 portages**, 35 spans |
| drapeau 1 — equipe **0** | **1 portage**, 3 spans | **0 portage**, 1 span (`home` tout le match) |
| `coverage.flagCarries.homeByObject` | 2 | **1** (la capture remplace une rentree inferee) |

**`cmd/replay-diff`, HEAD 45 contre HEAD 46** — l'effet est confine a `flagCarries` :

```
bcb6d393   12 ecarts / 679 mesures   ports 10 · couverture 1 (homeByObject) · entete 1 (schema)
e94163af    1 ecart  / 645 mesures   entete 1 (schema)  -- identique par ailleurs
c0a82e88    1 ecart  / 569 mesures   entete 1 (schema)  -- identique par ailleurs
cde26226    1 ecart  / 660 mesures   entete 1 (schema)  -- identique par ailleurs
64e8adfa   13 ecarts / 699 mesures   ports 8 gains / 3 pertes · homeByObject 10 -> 12 · schema
```

`changements = 0` partout : **aucune mesure ne change de valeur**, seules des mesures apparaissent
ou disparaissent.

## 8. Gates joues

```
cd apps/go-api
go test -count=1 ./internal/analysis/... ./internal/replaybuild/... ./internal/replaydiff/... \
        ./internal/archlint/... ./contracttest/...            # ok, 23 paquets
go test -count=1 -tags=integration -p 1 ./internal/api/wire/...  # ok 18,6 s (code de cuisson touche)
go build ./...                                                    # ok (CGO_ENABLED=1)
golangci-lint run --new-from-merge-base=origin/main ./...         # 0 issues
```

**Golden d'assemblage regenere : unique ecart = la ligne de version** (`schema 45` -> `schema 46`,
1 ligne sur 606 des deux cotes) — le film de reference `000d5950` n'est pas un CTF et n'est touche
par rien. Le contrat OpenAPI declare `schemaVersion` sans `enum`/`const`/`default` : un bump ne le
deplace pas ; aucun champ n'est ajoute au document.

## 8 bis. Un rouge de CI REPARE, apporte par le merge des corrections DUREES-R1

Le job « Go Coverage + Baseline non-regression » (`./...` complet, seul endroit ou la CI
execute `internal/service/replayview`) est sorti ROUGE apres le merge de `feat/v2-durees` :
**25 tests en echec, tous dans ce paquet, une seule cause**.

```
--- FAIL: TestChaqueChampStockeAUneDecision
    FlagCarriesCoverage.AmbiguousSlot est publie par l artefact et ABSENT du document servi,
    sans entree dans champsNonServis — soit le contrat le porte, soit la decision s ecrit
```

Le correctif C1 de DUREES-R1 ajoute le compteur `AmbiguousSlot` a
`replay.FlagCarriesCoverage` (les slots dont la vie anonyme est refusee au repli) sans le
miroiter dans le document SERVI. Le garde-rail de parite fait exactement son travail : un
compteur de couverture qui n atteint pas le client sortirait du contrat par omission.

**Repare ici** (directive « tout rouge se repare, meme prealable ») et par le cote QUI SERT,
pas par une exemption : les vingt-huit autres compteurs de ce bloc sont servis, celui-ci n a
aucune raison de ne pas l etre. Trois pas, sans decision de produit :
`replaydoc.FlagCarriesCoverage` gagne le champ a la MEME place, `toFlagCarriesCoverage` le
recopie, et `openapi.yaml` puis `generated.ts` sont REGENERES (jamais edites a la main) —
diff : 4 lignes au contrat, 2 aux types web. Paquet vert, contrat vert.

## 9. Decouvertes, notees et NON traitees

1. **`64e8adfa` garde 7 portages sur leur propre drapeau (13 avant).** Le film a **2 manches** et
   `closedOverlaps = 10` — une contradiction entre faits dates, deja publiee par la couverture. Aux
   sept instants restants, ou bien les DEUX drapeaux sont dehors (la troisieme regle se tait a
   dessein), ou bien `enJeu` est PERIME parce que le drapeau est rentre par un `flag_returns` ou
   une rentree d'objet, que `assignFlags` ne voit pas. **Une regle tentante est REFUTEE** : « une
   PRISE au socle S ne porte pas sur le drapeau de S » vaut pour six des sept fautes mais
   **contredit la septieme** (frame 6569 : un joueur de l'equipe 1 prend a 2,6 m du socle de
   l'equipe 0, et la bonne reponse est justement le drapeau de l'equipe 0). *Reprise* : faire
   partager a l'attribution la machine a etats de `assembleFlagLives` (retours credites + rentrees
   d'objet) plutot que d'en ecrire une seconde copie — la regle du depot interdit la troisieme
   copie d'un meme motif.
2. **Un portage ferme par la prise SUIVANTE DU MEME JOUEUR se publie `dropped` alors qu'il est
   porte.** `applyFlagLifeEvent` date la fin a `frameOfMatchMS(t1) + 1` et l'ouverture a
   `frameOfMatchMS(t0)` : quand `t1 == t0` (le meme joueur relache et reprend — ce que fait tout
   porteur qui tire), la fin tombe UNE FRAME APRES l'ouverture et ecrase l'etat `carried`. Mesure :
   **10 portages sur 16 sur `bcb6d393`** (avant comme apres ce lot), **17 sur 33 sur `e94163af`**.
   *Consequence mesuree de ce lot* : le 16e portage, desormais sur le bon drapeau, tombe sous ce
   defaut-la, et la duree PUBLIEE de `2533274858283686` passe de 358 a 282 frames — les 77 frames
   etaient auparavant visibles parce qu'elles etaient seules sur un drapeau que personne d'autre ne
   touchait. Le portage n'est pas perdu (il est publie, a la bonne place) ; c'est son span qui est
   masque. *Reprise* : une fin ne doit pas ecraser un `carried` ouvert au meme instant sur le meme
   drapeau — corriger la datation de la transition de fin, avec un test de mutation sur un relais
   du meme slot.
3. **Aucun compteur de couverture ne publie combien de portages la troisieme regle a attribues.**
   `FlagCarriesCoverage` est servi par le contrat (`internal/domain/replaydoc/coverage.go` +
   `openapi.yaml` + `generated.ts`) : ajouter un champ deborde le perimetre de ce fait. *Reprise* :
   au prochain lot qui touche la couverture du drapeau, publier `assignedByPlay` (et son
   pendant ambigu) pour que la regle se verifie au lieu de se croire.
