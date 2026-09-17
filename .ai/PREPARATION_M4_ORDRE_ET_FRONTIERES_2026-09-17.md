> SECONDE NOTE DE PREPARATION DE M4 — ecrite a la cloture de M2 (tete 492cb0923, lot 3.3 fusionne, trois lots M3 en vol),
> EN COMPLEMENT de `.ai/PREPARATION_M4_2026-09-17.md` (analyse sur pieces de 4.1 a 4.4, base 24b67e339, AVANT la cloture de M2).
> Celle-ci porte l ORDRE des lots, les FRONTIERES avec les lots M3 en vol, le cout reel de la recuisson (87 artefacts),
> la montee de schema unique 61 -> 62 et le decoupage en commits. Produite par un workflow de 8 agents (3 mesures, 3 sceptiques,
> synthese, critique de completude a 24 constats) puis corrigee constat par constat (§6). Les briefs d executant 4.1 et 4.2+4.4
> restent hors depot jusqu au lancement des lots (leur `__BASE__` = l integration apres la fusion de 3.6.a).

# NOTE DE PREPARATION DE M4 — la publication (lots 4.1, 4.2, 4.4 ; 4.3 reporte V16)

> Mesures reprises le **2026-09-17** (`date +%F`) en LECTURE SEULE dans le worktree
> `C:/Users/Guillaume/Downloads/Scripts/LevelUp-wt-recherche-film`, branche
> `feat/recherche-decodeur-film`, tete **`492cb0923`** (fusion du lot 3.3), plus les trois
> branches M3 en vol lues par `git show <branche>:<chemin>` et
> `git log --name-only 492cb0923..<branche>`. Aucun fichier du depot n'a ete modifie, **aucun
> decodage de film, aucun `replay-equiv`, aucun corpus gate**. Chemins Go relatifs a
> `apps/go-api/`, chemins web a `apps/web/`.
>
> **Revision 2 (2026-09-17)** : cette note a ete corrigee sur 24 constats de critique, chacun
> re-verifie sur pieces. Les corrections de fond sont au §5 ; les six constats **refutes par la
> mesure** y portent leur preuve. Trois decisions de pilote non negociables la gouvernent :
> **(a)** V20 (1) tient — la recuisson a lieu a la cloture de M3, sur signal utilisateur ;
> **(b)** aucun lot M4 ne demarre avant la fusion des trois lots M3 en vol ; **(c)** le lot
> « 4.0 facade reduite » est **non retenu**.

---

## 0. Resume, et l'ordre des lots

### 0.1 En une phrase

M4 tel qu'il est ecrit **ajoute** a la frontiere du decodeur sans rien lui retirer (§1.2) ; la
valeur du jalon n'est pas la reduction de la facade mais **une seule montee de schema qui achete
la selectivite** (§3) et **un fichier de faits qui transforme la prochaine recuisson du parc en
republication** (§2). L'ordre ci-dessous obeit a une seule regle : **aucun lot M4 ne demarre
avant que les trois lots M3 en vol soient fusionnes**, parce que les trois mutent des fichiers
que M4 doit rouvrir (§4) et que tout ratchet gele sur une base perimee rougit a la fusion.

### 0.2 Les vagues

`__BASE__`, pour tout brief de M4 et pour tout ratchet gele, vaut **l'integration APRES la fusion
de 3.6.a** — c'est-a-dire apres les trois fusions. Il n'y a **plus de vague « maintenant »**.

| Vague | Contenu | Depend de | Peut demarrer |
|---|---|---|---|
| **P — le prealable** | fusion des TROIS lots M3 en vol : **3.4.1** (9 commits, `492cb0923..feat/decfilm-341`), **3.1.1** (1 commit, `f6b37a662`), **3.6.a** (1 commit, `2dd16f357`) ; corpus re-fige | — | en cours, hors M4 |
| **R — LA recuisson, une seule (V20 (1))** | recuisson du parc a la cloture de M3, **sur signal utilisateur** : `levelup backfill-replay --only-existing`, **87 artefacts** (§0.3). Elle applique le contenu de M3 au schema 61 ; elle n'ecrit **aucun** fait (4.1.2 n'existe pas encore) | P | sur signal |
| **V1** | 4.1.1-a (le codec des entrees passe en production, **dans `package replay`**), 4.1.1-c (`PathResolver.FilmFactsPath` + ratchet du litteral + mesure admin), **et en parallele** 4.2.1-a (la table `layers`, SANS montee de schema) | `__BASE__` | apres P |
| **V2** | executeur 4.1 : 4.1.1-b (les cinq sections), 4.1.2 (bascule `BuildBytes`), 4.1.3 (S8) — **et en parallele** executeur 4.2 : **C2, la montee unique 61 -> 62** | V1 (la table `layers` alimente C2) | apres V1 |
| **V3** | 4.2.2 (le web lit `layers`), 4.4.1 (`Digest` + verdict a trois sorties), 4.4.2 (le badge par couche) | C2 fusionne **et** 4.1.2 fusionne (§3.5 : sans faits sur le disque, `republier` n'existe pas) | apres V2 |
| **Cloture M4** | ADR 0034 amende, §5 complet, S1-S8 re-verifies, **M4-P4** : republication en une passe (§3.9) | tout ce qui precede | sur signal |

**La machine, pas les commits, est la ressource serialisee.** V1 et V2 se parallelisent par
frontiere de FICHIERS (§4.3) ; les deux gates AVEC decodage de V2 — l'equivalence 20 films de C2
et le S8 de 4.1.3 — ne peuvent pas tourner en meme temps (un seul decodage a la fois). Ils
s'ordonnent sur signal du pilote, jamais par les executants.

### 0.3 QUAND recuire : V20 (1) tient, et le cout reel est **87 artefacts**

`Digest.UpToDate()` est une **egalite de `SchemaVersion` et rien d'autre**
(`internal/replaybuild/artifact_digest.go:54`). Toute montee marque donc « a recuire » tout ce qui
est cuit.

**Le chiffre exact, mesure ce jour** : la recuisson est la passe
`levelup backfill-replay --only-existing` — le drapeau dit lui-meme « ne traiter que les matchs
qui ont deja un artefact sur disque (passe d apres un bump de schema) »
(`cmd/levelup/cmd_backfill_replay.go:200-201`). Son domaine est donc le parc d'artefacts, pas le
parc de films :

| Grandeur | Mesure du jour | Lieu |
|---|---:|---|
| Artefacts cuits (hors derives) | **87** | `data/cache/replays/halo_infinite/*.json` (depot principal) |
| Sidecars derives | 75 | `*.derived.json`, meme dossier |
| Films au cache | 1 589 | `data/cache/film_chunks/` (jonction, identique dans les deux arbres) |

**Les 1 589 films ne sont PAS le cout d'une recuisson** : sans `--only-existing`, la passe joint
le cache film au registre (`replaysACuire`, `cmd_backfill_replay.go:243`) et cuirait tout le
parc — ce que V20 (1) n'a jamais demande. V20 (1) parlait de **76 artefacts** (plan ligne 108) ;
il y en a **87** aujourd'hui. L'ordre de grandeur est le meme, et il est dix-huit fois plus petit
que celui qu'annoncait la revision 1 de cette note.

**Decision : V20 (1) s'applique tel quel — une recuisson a la cloture de M3, sur signal
utilisateur.** Elle se joue avant tout lot M4.

> **Option, non recommandee, presentee pour le seul cout** : attendre 4.1.2 ferait ecrire les
> faits « en passant » par cette recuisson, et la republication de M4-P4 deviendrait une passe de
> secondes. Le cout d'y renoncer — c'est-a-dire de tenir V20 (1) — est **une seconde passe
> complete de 87 artefacts** a la cloture de M4 : 87 x ~15 s ~ **22 minutes** de machine, un
> decodage a la fois (les neuf durees consignees a `.ai/V7.5/MESURES_CUISSON_PERF.md:181-190`
> vont de 14,0 s a 27,5 s, plus un BTB a 1 min 40 : compter ~25 min).
> Ce paragraphe ne re-arbitre pas V20 (1) ; il en chiffre le prix, qui est petit.

---

## 1. La facade `film/decfilm` — le lot « 4.0 » est **NON RETENU**

### 1.1 L'etat mesure, sur les DEUX bases

| Grandeur | `492cb0923` | `feat/decfilm-341` (la base reelle apres fusion) |
|---|---:|---:|
| Lignes de `film/decfilm/decfilm.go` | 434 | **446** |
| Declarations de premier niveau | 163 | **166** |
| Marge avant le plafond de 500 lignes | 66 | **54** |

3.4.1 ajoute trois symboles — `type Options = killsource.Options`, `func DefaultOptions()`,
`func LargeurIndexDePlage()` — soit `killsource` +2 et `profile` +1 par famille d'origine. **Tout
compteur gele sur `492cb0923` rougit a la fusion** : c'est la raison n°1 de la regle d'ordre du
§0.2.

Le chiffre 163 est ecrit trois fois et **aucun test ne le compte** : `decfilm.go:22`,
`docs/adr/0034-film-decoder-profile-and-layers.md:340` et `:511`. La seule borne effective est le
plafond de 500 lignes par fichier (`internal/archlint/film_file_size_test.go:48`), soit ~20
symboles de marge au rythme mesure (446/166 = 2,69 L par symbole).

Table d'usage (mesuree sur `492cb0923`, a re-mesurer a l'entree du lot) : 29 paquets
consommateurs, 163 symboles declares et 163 cites (difference symetrique vide dans les deux
sens), 49 exclusifs a des outils `cmd/` (30 %), 115 cites par un seul paquet (71 %), 83 cites en
production par un paquet `internal/`, 21 cites uniquement en test, 10 alias vers `film/types`.

### 1.2 Ce que M4, tel qu'il est ecrit, fait a la facade

Il l'**augmente**. Aucun item de `### M4` (plan `:4981-5019`) ne retire un symbole, et deux en
ajoutent :

- **4.1** a besoin d'une porte d'ecriture et d'une porte de lecture des faits. Le type a
  serialiser est `replay.FilmInputs` (`film/replay/film_inputs.go:42`, 40 champs) et ses champs
  exportes **nomment des types de `grammar`** (`:53`, `:55`, `:57`) : la porte ne peut pas etre
  anonyme.
- **4.4.1** exige les **quatre** revisions de couche (ADR 0034 `:446-449`) la ou la facade n'en
  exporte qu'une (`decfilm.go:85`, `const Rev = facts.Rev`).

Le nombre exact ajoute n'est pas mesurable avant d'ecrire le lot — c'est une projection, pas une
mesure, et elle n'est pas chiffree ici.

### 1.3 Pourquoi le lot « 4.0 » est non retenu

1. **Il n'est necessaire a aucun item de M4.** 4.1 s'ecrit sans lui (§2.7) ; 4.2 et 4.4 ne
   touchent pas `decfilm`. Rien dans la chaine mesuree n'en depend.
2. **Aucun gain pour l'utilisateur.** Zero rendu change, zero seconde gagnee, zero fait de plus
   lu dans un film. C'est de l'hygiene d'architecture ; la doctrine du depot (seul developpeur,
   pragmatisme) dit de ne pas en faire un lot quand un jalon a des livrables mesurables devant
   lui.
3. **La reduction n'a pas d'item numerote**, et V16 vient de retirer de M4 son lot voisin (4.3,
   « un seul type publie », statue `[!]`, plan `:5006`). Ouvrir 4.0 pendant qu'on ferme 4.3 serait
   incoherent.
4. **Le vrai poids est ailleurs** : `film/replay` est cite par **242 identifiants `replay.X`**
   hors de `film/` (§1.4). Reduire 166 sans toucher 242, c'est deplacer la dette, pas la
   frontiere.

**Ce qui tombe avec le lot** : le re-pointage des 10 alias `film/types`
(`decfilm.go:214, 215, 231, 279, 290, 327, 339, 359, 366, 395`) sur `film/types`. C'etait le
commit (6) du brief 4.1 ; il est **retire du brief** et consigne ici, non retenu — meme raison
que ci-dessus (aucun item du plan, aucun gain mesurable), et il touchait `killcollector` et
`replaybuild` pour rien.

### 1.4 Ce qui reste : **un compteur, et sa methode de comptage ecrite**

Un seul fichier de test, dans le commit d'ouverture de 4.1 — pas un lot. Sa raison d'etre est au
§1.2 : M4 **ajoute** a la facade, et rien aujourd'hui ne le dirait.

| # | Geste | Ou |
|---|---|---|
| a | `internal/archlint/film_facade_surface_test.go` : plafond **global** date sur les declarations exportees de premier niveau de `decfilm.go`, compte **par AST** (`go/parser`), plus une table de plafonds **par famille d'origine**, plus un plancher anti-muet (`t.Fatalf` si le fichier manque ou si le compte tombe a zero) | commit d'ouverture de 4.1 |
| b | **Le plafond compagnon** sur les identifiants `replay.X` cites hors de `film/`, dans le meme fichier | idem |

**La methode de comptage, ecrite et reproductible en une commande** (decision de pilote — un
plafond dont la valeur ne se reproduit pas n'est pas un ratchet) :

```sh
# (a) surface de la facade — cible du ratchet AST ; ce grep en est le controle croise
grep -cE '^(const|type|var|func) [A-Z]' \
  apps/go-api/internal/games/halo_infinite/film/decfilm/decfilm.go
#   -> 163 sur 492cb0923 ; 166 sur feat/decfilm-341

# (b) plafond compagnon — IDENTIFIANTS DISTINCTS, pas occurrences
find apps/go-api -name '*.go' -not -path '*/film/*' -print0 \
  | xargs -0 grep -hoE '\breplay\.[A-Z][A-Za-z0-9_]*' | sort -u | wc -l
#   -> 242 sur 492cb0923
```

Definitions, sans lesquelles la valeur n'est pas reproductible :

- **perimetre (a)** : le seul fichier `film/decfilm/decfilm.go`, declarations de premier niveau
  (colonne 0) dont le nom est exporte. Le ratchet les compte par AST ; le grep ci-dessus est le
  controle croise, et les deux rendent le meme nombre sur les deux bases.
- **perimetre (b)** : tout fichier `.go` du module **dont le chemin ne contient pas `/film/`**,
  tests compris. **Identifiants distincts** (`sort -u`), jamais occurrences — les occurrences
  valent 1 192 sur la meme base et ne mesurent que la verbosite.
- **controle croise de (b)** : restreindre l'ensemble aux seuls fichiers qui importent
  `halo_infinite/film/replay` rend **le meme 242** (30 paquets, 160 fichiers). Les deux variantes
  se valent ; la commande ci-dessus est la forme retenue parce qu'elle tient en une ligne.
- **date et base** : la valeur gelee porte en commentaire `// <date> — base <sha de __BASE__>`,
  et elle est **re-mesuree a l'entree du lot** sur `__BASE__`, jamais recopiee d'ici.

Le ratchet doit rougir **dans les deux sens** : un symbole de plus, et un symbole de moins sans
baisse de la constante dans le meme commit (sinon la marge se reconstitue en silence). La preuve
se fait par mutation a la main avant le commit.

### 1.5 Ce qui n'est **pas** retenu

- **Le lot « 4.0 facade reduite » en entier** (§1.3), y compris le re-pointage des 10 alias
  `film/types`.
- La descente des six outils de recherche (`diag_film`, `diag_weapons_v3`, `rdata_weapon_scan`,
  `statnames-sweep`, `oddball-terrain`, `zone-attribution`) sous `film/research` avec tag
  `research` — le motif existe (`film/research/{grenadeids,cmd_grenadeids,largeursaxe}`), le gain
  serait de 30 symboles, mais ces outils servent au diagnostic courant et les sortir du build par
  defaut est une decision utilisateur. **Consigne, non traite.**
- Le cas `cmd/killsource` (31 symboles cites, 19 exclusifs) : meme raison, en plus lourd.
- La sortie de la famille mapquant vers un paquet exporte `film/mapquant` : `LoadMapQuantCatalog`
  est le symbole le plus partage du depot (7 paquets).

---

## 2. Lot 4.1 — les faits persistes par film

### 2.1 Le serialiseur existe deja, il est en test — et il a **onze lecteurs**

Huit fixtures `film/replay/testdata/inputs_<short8>.bin.gz` (un par build de `goldenBuilds()`),
**11 049 200 octets au total, 1 381 150 o en moyenne**, ecrites par `encodeGoldenInputs`
(`golden_inputs_encode_test.go:24`). Le codec est delta-code, quantifie, prouve point fixe build
par build (`TestGoldenBuildsInputsRoundTrip`), couvert **par reflexion**
(`TestCodecCouvreFilmInputs`, `golden_inputs_canaux_test.go:409`), borne par un budget
(`goldenInputsBudget = 12 << 20`, `golden_inputs_budget_test.go:40`) et double d'un oracle de
fidelite (`TestGoldenInputsFidelite`, `golden_inputs_fidelite_test.go:43`, qui saute sans le cache
de films).

**Deux mesures corrigent la revision 1 de cette note, et elles commandent le decoupage (§2.7) :**

| Mesure | Valeur | Consequence |
|---|---:|---|
| Les **huit** `golden_inputs_*.go` | **2 532 L** | dont `golden_inputs_test.go` 433, `_budget_` 72, `_fidelite_` 118 — que le lot ne deplace pas |
| Les **cinq** fichiers « du codec » | **1 909 L** | `_codec_` 475, `_encode_` 373, `_decode_` 371, `_canaux_` 491, `_film_` 199 |
| Fichiers de test qui citent le type `goldenInputs` | **11**, pas 5 | `document_pickups_test.go`, `e191_composants_research_test.go`, `e191_origine_mesure_research_test.go`, `golden_builds_test.go`, `golden_inputs_budget_test.go`, `vies_un_echantillon_test.go` **restent** en place et le citent |

Le type lui-meme est declare **hors** des cinq : `type goldenInputs struct` a
`golden_inputs_test.go:270`, avec `func (g *goldenInputs) options()` a `:292` et les trois champs
de cle de cuisson a `:274` (`MapModule`), `:279` (`AxisW`), `:284` (`LayoutDetected`). Un
« deplacement pur des cinq fichiers » **ne compile pas**.

**4.1 reste un lot de PROMOTION et de fermeture de trous, pas un lot de conception.** Le format
retenu est le codec maison : `encoding/gob` n'existe nulle part dans le depot (zero occurrence) et
un JSON des entrees pese plus que l'artefact lui-meme (un artefact mesure a 2 524 016 o pour
`000d5950` contre 982 629 o de faits compresses).

### 2.2 Les trous mesures — c'est la ou est le risque

| # | Trou | Preuve |
|---|---|---|
| 1 | **Quatre champs de `FilmInputs` ne sont pas transportes** : `FlagMarks`, `ZoneReads`, `ZoneScanned`, `BombReads` | `golden_inputs_canaux_test.go:387-392` ; la justification (« le fixture ne fournit AUCUNE garde ») est vraie pour un fixture et **fausse en production** : tout CTF, tout KOTH/Strongholds, tout Assaut les remplit. Le test le dit lui-meme (`:384-386`) |
| 2 | **`opt.FilmIdentity` n'est pas un champ de `FilmInputs`** | pose a `build_from_film.go:81`, consomme a `build_pistes.go:201` -> `coverage_decoder.go:167` ; sans lui l'artefact rejoue publierait un `build` vide et un bloc `registry` absent, c'est-a-dire l'ambiguite que D-7 interdit |
| 3 | **Le compteur de replis du BALAYAGE** | ne a `build_from_film.go:62-63`, **avant** le premier balayage (commentaire `:59`), declenche des la phase film (`build_from_film.go:115`, `film_scan.go:172` -> `inventory_decode.go:195`). La ventilation balayage/assemblage n'est ecrite nulle part : **c'est une mesure obligatoire du lot** |
| 4 | **Deux familles de faits hors `FilmInputs`** : `decfilm.Result` (killsource) et `[]types.StatRecord` + `CaptureBurstTimes` + le temoin `truncated` (statborg) | `replaybuild.go:264` (`readFilmStats`), `:356` (`decodeKillSource`), `:284` (`replay.BuildFromFilm`) — **trois familles, et `replaybuild.go:436-438` ecrit « DEUX DECODAGES COMPLETS DU MEME FILM, ET C'EST VOULU »** |
| 5 | **`Kill.paquet` est non exporte** (`killsource/kill.go:72`, raison `:69`) et **`digest.Of` hache les champs exportes ou non** (`internal/analysis/digest/digest.go:20`) | consequence directe : **l'etape `killsource` du TSV d'equivalence ne peut pas servir d'oracle** a un rejeu depuis les faits. S8 se juge sur la ligne `artifact`, et sur elle seule |

### 2.3 Le chemin, et un risque de nom

Le modele a recopier est `internal/domain/title/registry_tactical.go` : fichier a part
(`registry.go` porte sa dette gelee par la baseline), un `const` unique (`:24
SousDossierRasters`), deux methodes (`:41` et `:52`). **Attention** : ce sidecar-la vit **sous**
le dossier des artefacts (`:42`, `ReplayArtifactsDir(titleSlug)`) et ne survit que parce que les
deux parcours du dossier sautent les repertoires (mise en garde ecrite `:35-40`). Les faits, eux,
doivent naitre **frere** de `data/cache/replays/`, sous `CacheRootDir()` (`registry.go:782`) — ils
echappent alors aux deux parcours par construction. La cle du nom de fichier est
`FilmShortMatchID` (`internal/domain/title/film_id.go:28`), non negociable.

**Risque de nom, a trancher au premier commit** : `<short8>.facts.json` existe deja et veut dire
**l'inverse** — ce sont les faits que la **base** sait (`internal/replaybuild/facts_file.go`, ecrit
par `levelup replay-facts-export`, lu par `cmd/replay-equiv`). **Recommandation :
`<short8>.filmfacts.bin`**, et la distinction ecrite en tete du fichier Go.

Un ratchet a etendre **dans le meme commit** : `internal/archlint/no_hardcoded_film_cache_dirs_test.go`
(`:40`, expression `"film_manifests"|"film_chunks"`) doit apprendre `"film_facts"` — et, comme la
definition canonique du nouveau litteral **ne vit pas dans `filmcache.go`**, l'allowlist (`:35-38`,
deux entrees aujourd'hui) doit gagner `internal/domain/title/registry_film_facts.go` dans le meme
commit, sinon le ratchet mord sa propre source. En revanche
`no_runtime_versioned_catalog_write_test.go` **ne mord pas** : sa table (`:58-64`) ne contient que
`MapWeaponPadsPath` et `FilmProfilesPath` (chemins suivis par git), et `data/cache/*` est
gitignore (`.gitignore:114`).

### 2.4 L'en-tete : `DecoderCoverage` verbatim, et la regle de fraicheur

`replay.DecoderCoverage` (`coverage_decoder.go:33`) porte **deja** `{sourceRev, profileRev,
grammarRev, factsRev, build, registry}`, avec la regle « chaine vide et bloc present sur un build
inconnu » (`:44-53`). Ecrire un second bloc de revisions serait la **troisieme copie**. Le fichier
de faits porte donc ce type tel quel, ce qui offre un invariant gratuit et testable : **l'en-tete
relu == le `coverage.decoder` de l'artefact produit, champ pour champ.**

Regle de fraicheur au lot 4.1, **sans finesse** : faits utilisables si et seulement si version du
codec, schema de faits, **les quatre** revisions et la cle de cuisson (module de carte, largeurs
d'axe, `LayoutDetected`) sont egaux a ce que le binaire courant resout. La finesse par couche est
**l'objet du lot 4.4** — `coverage_decoder.go:24-27` le dit deja. Ne pas l'anticiper.

### 2.5 Le verrou solo : six sites, et rien a changer

`filmproc.AcquireSolo` / `AcquireSoloWait` sont pris par les **points d'entree**, jamais par
`replaybuild` (aucune occurrence dans `internal/replaybuild/`). Les six sites de production :

```
cmd/levelup/cmd_backfill_replay_child.go:80    AcquireSoloWait
cmd/replay-build/main.go:151                   AcquireSolo
cmd/replay-corpus-gate/bake.go:100             AcquireSoloWait
cmd/replay-equiv/child.go:77                   AcquireSoloWait
cmd/replay-worker/job.go:296                   AcquireSoloWait
internal/replaychild/replaychild.go:196        AcquireSolo   <-- le chemin POST-SYNC, en prod
```

Le sixieme est le chemin qui tourne sur le serveur (`sync/replayartifacts` -> `replaychild.Spawn`)
et c'est **exactement celui sous lequel un rejeu depuis les faits s'executera**. Un rejeu depuis
les faits ne decompresse aucun chunk et ne merite pas le verrou, mais **le sortir du verrou
deplacerait une garantie memoire dans une boucle** — la porte par laquelle quatre sinistres RAM
sont passes. **Recommandation : ne rien changer au lot 4.1** ; rendre la decision au point
d'entree seulement si 4.4 la mesure necessaire.

### 2.6 Le cout : ce qui est mesure, ce qui ne l'est pas

`.ai/V7.5/MESURES_CUISSON_PERF.md:181-190` donne des durees de cuisson complete pour **neuf films
seulement** (14,0 s pour `000d5950` ... 1 min 40 pour `084a804d`, BTB 26 joueurs), soit ~4,3 min ;
les onze autres films du corpus n'y ont **aucune duree**. Toute phrase du type « 7 a 9 minutes sur
les 20 films » est une extrapolation, pas une mesure — elle n'entre pas dans le plan.

Ce qui manque est mesurable **pour le prix d'une cuisson** : les cinq phases sont deja
instrumentees (`logPhase` a `replaybuild.go:262, 265, 285, 294, 357`, declaree `timing.go:30`).

**Gate de 4.1.3, formule en mesure et pas en promesse** : consigner le tableau des cinq phases sur
deux temoins (`000d5950`, `084a804d`) **avant** d'ecrire le chiffre attendu de la passe depuis les
faits. Le « secondes contre 15 s » du plan se prouve, il ne s'annonce pas.

### 2.7 Decoupage en commits — **il compile a chaque commit, et aucun paquet n'est sans appelant**

Le decoupage de la revision 1 livrait, aux commits (1) a (3), **un paquet de production sans
appelant de production** (le seul appelant possible, `internal/replaybuild/`, n'ouvrait qu'au
commit (4)) — CLAUDE.md regle 7. Deux sorties existaient ; la mesure tranche pour la premiere.

**Option retenue : le codec reste dans `package replay` et s'exporte. Aucun paquet neuf.**
Les raisons, toutes mesurees :

1. **Onze fichiers de test citent `goldenInputs`** (§2.1) et six d'entre eux restent en place. Un
   paquet neuf obligerait a re-pointer six fichiers hors perimetre, dont trois tests de recherche.
2. **La regle des 500 lignes ne l'exige pas** : le plus gros fichier du codec fait **491 L**
   (`golden_inputs_canaux_test.go`). L'argument « c'est pourquoi c'est un paquet » de la revision 1
   etait faux.
3. **La porte d'ecriture/lecture reste interne** : `FilmInputs` est declare `package replay`
   (`film_inputs.go:1`, `:42`) et `applyTo` est non exporte (`:170`). Un paquet frere devrait
   exporter les deux sens ; ici, il n'y a rien a ouvrir.
4. **Zero ligne de baseline JSONL a retirer** : la baseline est indexee par **(Package, Test)**
   (`.ai/baselines/tests_pre_migration.jsonl`, `scripts/check_test_baseline.sh`, controle 1
   « PRESENCE »). Rien ne bouge tant que ni le paquet ni le nom d'un test ne changent. C'est le
   mode de panne qui a produit deux CI rouges le 16/09.
5. **Aucune entree a ajouter a `couchesDuDecodeur`** (`archlint/film_layers_deps_test.go`) : pas de
   paquet neuf, donc pas de classement, donc pas de R2/R3 a satisfaire.

**Un fichier ne monte PAS en production** : `golden_inputs_film_test.go` (199 L) appelle
`source.LoadDir(dir, nil)` (`:69`). R4 (`TestCoucheDePublicationNeChargePasLeFilm`,
`film_layers_deps_test.go:473`) balaie **exactement les `.go` non-test poses a plat dans
`film/replay/`** (`film_layers_deps_helpers_test.go:161-177`) et refuse tout appel a
`source.{LoadDir,Load,LoadIntoDir,DirSource,MemoryChunks}` hors des quatre enveloppes tolerees. Le
promouvoir rougirait R4. C'est de toute facon un **generateur de fixtures**, pas du codec : il lit
le film ET le `<short8>.facts.json` du corpus. **Il reste un `_test.go`.**

| # | Commit | Contenu | Gate |
|---|---|---|---|
| 1 | **4.1.1-a** — le codec passe en production | Les quatre fichiers de codec (`_codec_` 475, `_encode_` 373, `_decode_` 371, `_canaux_` 491) deviennent des fichiers de **production** de `package replay` (`filmfacts_codec.go`, `_encode.go`, `_decode.go`, `_canaux.go`), avec le **type** et les constantes d'en-tete extraits de `golden_inputs_test.go:219-290` vers `filmfacts.go`. Les tests restent des tests, dans leurs fichiers, sous leurs noms actuels. `golden_inputs_film_test.go` NE BOUGE PAS. + le compteur de surface (§1.4) | `git diff --stat -- film/replay/testdata/` **VIDE** ; `go build ./... && go vet ./...` ; les cinq tests du codec verts ; `archlint` complet vert ; **aucune ligne de baseline touchee** (le prouver : `git diff --stat -- .ai/baselines/` vide) ; **aucun decodage** |
| 2 | **4.1.1-c** — le chemin, l'ecriture, la mesure | `internal/domain/title/registry_film_facts.go` + ecriture atomique (`platform/atomicfile`) + le litteral et son ratchet (§2.3) + la ligne « ce cron ne purge pas les faits » dans `scheduler/replay_purge_cron.go` + la mesure de taille pour M4-D1 (§3.8) | tests de `domain/title` verts ; le nouveau ratchet **rougit sur une copie du litteral** (mutation verifiee a la main) ; `make check-types` si la mesure touche le web ; **aucun decodage** |
| 3 | **4.1.1-b** — les cinq sections | `FilmInputs` **complet**, `FilmIdentity`, replis du balayage, statborg, killsource ; `champsNonTransportes` **videe et son mecanisme supprime** avec sa derniere entree | `TestCodecCouvreFilmInputs` etendu aux cinq sections **par reflexion** ; golden d'en-tete ; round-trip point fixe sur les 8 fixtures ; **taille consignee** et le commentaire de budget corrige (§5, decouverte 2) ; **aucun decodage** |
| 4 | **4.1.2** — la bascule | `replay.BuildFromFacts` exporte + bascule dans `BuildBytes` entre `:244` et `:257` (l'entree de catalogue est resolue avant, elle valide l'en-tete) ; ecriture des faits sur le chemin decode | invariant teste : en-tete relu == `coverage.decoder` produit ; la branche « relire » traverse **le meme `writeArtifactBytes`** (`artifact_store.go:168`) ; `go test -tags=integration ./... -p 1` ; **aucun decodage** |
| 5 | **4.1.3** — S8, regime court | `replay-equiv` sait cuire depuis les faits ; le parent compare les **deux passes** du meme commit | 10 films au plus (V2) : `artifact` identique a l'octet ; toute etape intermediaire divergente **classee** (forme / contenu, protocole V14) et consignee, jamais regeneree |
| 6 | **4.1.3 (suite)** — S8, regime complet | les 20 films | 20/20 sur `artifact` ; **tableau des cinq phases consigne** ; corpus gate `--reference=base` : 0 perte |

**L'appelant de production arrive au commit (4), dans le meme lot.** Entre (1) et (3), le code
promu n'est pas un paquet orphelin : ce sont des symboles exportes de `package replay`, appeles
des le commit (1) par les onze fichiers de test qui les citaient deja.

> **Sortie de secours, si le pilote preferait le paquet neuf** : le paquet
> `film/replay/filmfacts/` et son appelant `internal/replaybuild` doivent alors entrer dans **le
> meme commit**, ce qui fond les commits (1) a (4) en un seul — plus gros, plus dur a relire, et
> il faut re-pointer six fichiers de test plus la baseline. La mesure ne le recommande pas.

---

## 3. Lots 4.2 + 4.4, et la montee de schema unique **61 -> 62**

### 3.1 Pourquoi une seule montee

`SchemaVersion` vaut **61** (`film/replay/document.go:48`) ; le document porte **58 balises
`json:` racine**, gelees des deux cotes par `apps/go-api/contracttest/replay_contract_test.go:702`
(`wantReplayDocumentFields = 58`). Une montee marque tout le parc a recuire (§0.3). Deux montees
= deux recuissons. **Tout ce qui attend un champ doit entrer dans la meme.**

### 3.2 Ce que la montee porte

1. **Les compteurs de couverture du balayage de M3**, aujourd'hui journalises et non publies —
   le godoc de `grammar/grenade_events.go:172-174` l'ecrit : « Elle est JOURNALISEE et ne voyage
   pas dans l'artefact : ajouter un compteur au document de rejeu changerait sa forme, donc son
   schema, ce qui est une decision de pilote et pas un effet de bord de ce decodeur. » Cinq
   compteurs emis a `:371-376`. S'y ajoutent ceux de 3.4.1 (positions gardees / jetees par index
   de plage) et la voie des morts (`killsource/kill.go:136` `PathWalk`, `:139` `PathScan`) : **rien
   n'existe dans `coverage.go` pour les accueillir**. **A re-mesurer sur `__BASE__`** : publier ce
   que les lots M3 ont reellement ajoute, pas la liste ecrite ici.
2. **`layers`** (4.2.1) — c'est ce qui **supprime la necessite de la recuisson suivante**. Le poser
   dans la meme montee que les compteurs, c'est acheter la selectivite au prix d'une recuisson
   qu'on paie de toute facon.

**Ordre impose dans le commit** : les compteurs d'abord (ils changent `coverage`), `layers`
ensuite (il change la racine), le golden de forme regenere **une seule fois**, a la fin.

### 3.3 La checklist du commit de schema — 8 points herites + **4 neufs**

Les huit points sont ecrits a `.ai/PREPARATION_M2_PAS_4_A_6_2026-09-17.md:1150-1163` et leur
execution integrale est consignee au §5 du plan (lot 2.6.3). Adaptes a 62 :

1. `film/replay/coverage.go` : les champs neufs ;
2. `domain/replaydoc/*` : les jumeaux (dont `Layers`) ;
3. `service/replayview/convert_coverage.go` + `convert_document.go` : la conversion ;
4. `film/replay/document.go:48` : `SchemaVersion = 62` ;
5. `film/replay/document_chronicle.go` : **l'entree v62 ecrite DANS CE COMMIT** (ADR 0034,
   correction 2 ; derniere entree v61 a `:1571`) ;
6. `film/replay/testdata/document_shape.golden` regenere par sa **porte unique**
   (`document_shape_test.go:124`, `-update` **et** `REPLAY_CONTRACT_UPDATE=1`, et elle ne rend
   jamais `ok` : `t.Fatalf` meme en succes, `:162`) ;
7. `api/openapi.yaml` regenere **EN DERNIER** (`make openapi-gen` puis `make generate-types`),
   gate `go test ./internal/api/ -run TestOpenAPIYAMLIsUpToDate -count=1` (CGO) ;
8. `service/replayview/parity_test.go` vert (reflexif, pas de liste a maintenir).

**Quatre points neufs, qui n'existaient pas au lot 2.6.3 :**

9. `apps/go-api/contracttest/replay_contract_test.go:702` : `wantReplayDocumentFields`
   **58 -> 59** ;
10. `apps/web/src/lib/replay/replayDocumentSchema.ts:102` : la cle `layers` dans le
    `z.strictObject` — **sans quoi `tsc -b` rougit** par `_MemesCles`
    (`replayDocumentSchema.test.ts:34`) et le badge dirait `invalid` en production. **D11 du plan
    est a amender : 4.2.2 touche DEUX fichiers web, pas un** ;
11. **les 8 fixtures de contrat CHANGENT DE NOM** : leur producteur les nomme
    `fmt.Sprintf("replay_schema_%d_%s.json.gz", SchemaVersion, f.film)`
    (`film/replay/contract_fixtures_test.go:150`). Au schema 62, les huit
    `apps/web/src/features/match-replay/test/fixtures/go/replay_schema_61_*.json.gz` deviennent
    `replay_schema_62_*.json.gz` : **8 suppressions + 8 creations**, pas huit modifications, plus
    le `manifest.json`. **Aucun garde-rail ne detecte un fichier orphelin** (mesure : rien ne
    balaie ce dossier cote web) — les huit anciens se suppriment **a la main, dans le meme
    commit** ;
12. **`film/replay/structure_test.go:1164`** epingle la version en dur :
    `if SchemaVersion != 61 { t.Fatalf("... incrementer exige une raison ecrite ci-dessus ...") }`.
    C'est **le seul epinglage dur** du depot (mesure : aucun autre). Le commit de montee doit y
    ecrire **la raison de v62** dans la chronique de commentaires (derniere entree `// - v61` a
    `:1147`) **et** passer l'epingle a 62.

**Et le point que la table des tailles impose (decision de pilote, droit explicite accorde)** :
`internal/archlint/film_file_size_test.go` gele `document_chronicle.go` a **1626** (`:107`) et
`structure_test.go` a **1168** (`:135`) ; les deux fichiers pesent **exactement** 1 626 et
1 168 L ce jour. Les points 5 et 12 les font donc **tous les deux** depasser. L'en-tete du ratchet
(`:76-90`) nomme deja ces deux fichiers comme **la seule exception ecrite** : « Leur plafond monte
du volume de l entree ajoutee, DANS LE COMMIT QUI MONTE `SchemaVersion`, et JAMAIS autrement. »
Le commit de montee **releve ces deux entrees**, avec justification datee dans la table.

### 3.4 La forme de `layers`

`Layers map[string]string` a la racine, `omitempty`, **regime identique a celui de `coverage`** :
l'OBJET absent = artefact anterieur au schema 62 ; l'objet present avec une **entree absente** =
ce calque n'a pas ete produit (c'est une reponse, pas un trou) ; entree presente = produit, sous
la revision nommee. La valeur est la revision de la **couche** qui a produit le calque, telle
qu'elle se nomme deja (`source-...`, `profile-...`, `grammar-...`, `killsource-...`, plus
`publication-62` pour ce que produit `film/replay`, qui n'a pas de revision de sources : le seul
`...Rev` du paquet est `UsageSummaryRev` et il concerne les usages, pas le document).

Le nom d'un calque = **la cle JSON du document**, pas le nom de l'etape de balayage ni celui du
bloc de couverture : c'est le lecteur qui decide, et le lecteur est `normalizeReplayDocument`.

**Trois champs racine n'y entrent JAMAIS** : `mapObjectives`, `mapWeaponPads`, `weaponTiers` —
la cuisson ne les ecrit pas (`document_shape_test.go:372-376`, garde
`TestDocumentShapeCalquesALaRequeteRestentHorsCuisson`).

Un test de fermeture, sur le modele de cette garde : tout champ racine cuit a une entree dans
`layers` ou une justification datee dans une table ; toute entree nomme un champ existant ; la
valeur est l'une des cinq revisions connues, jamais une chaine libre.

### 3.5 `Digest` : le verdict a trois sorties

Aujourd'hui `Digest` porte cinq champs (`artifact_digest.go:44-51`), `digestFromBytes` ne
deserialise que quatre cles, et **« a recuire » se decide par une egalite** (`:54`). Cinq sites de
decision en dependent (`api/wire/registry_build_queue.go:395`,
`sync/replayartifacts/artifacts.go:280`, `cmd/levelup/cmd_backfill_replay.go:232`,
`cmd_backfill_replay_repair.go:78` et `:85`), plus un sixieme site de lecture a **l'ecriture** :
`wouldDowngrade` (`artifact_store.go:88-89`).

| Verdict | Condition | Conduite |
|---|---|---|
| `aJour` | schema egal **et** les quatre revisions egales | rien |
| `republier` | les quatre revisions egales, seule la publication a bouge | rejouer depuis les faits |
| `redecoder` | une revision de couche a bouge **ou** les faits manquent sur le disque | verrou solo, decodage complet |

**Piege a ecrire dans le lot** : `republier` retombe sur `redecoder` quand le fichier de faits
manque — jamais un quatrieme etat « je republierais si j'avais les faits ». Et `wouldDowngrade`
doit apprendre les revisions : il ne refuse aujourd'hui qu'a **schema egal** et se tait a schema
different (`artifact_store.go:85-87`, `:94`).

**Dependance dure : 4.4 sans 4.1 est vide** — sans faits sur le disque, `republier` n'existe pas
et le verdict se reduit a l'egalite d'aujourd'hui. Si le jalon devait s'arreter apres 4.2, 4.4 se
statue `[!]` avec cette justification plutot que de se livrer ampute.

### 3.6 Le badge : il nomme les **couches**, pas les calques

`computeReplaySchemaStatus` (`replaySchemaStatusLogic.ts:70`) rend un statut a quatre etats ; V17
(M4-P1) tranche « une cle de plus par langue, aucun cinquieme etat ». La cle porte donc une
**liste de couches** (au plus cinq noms), pas une liste de calques : les quatre revisions sont
des **constantes de compilation** posees ensemble (`coverage_decoder.go:169-172`), une montee de
`grammar.Rev` perime d'un coup tous les calques qu'elle produit, et une empreinte par calque est
**interdite hors de `film/revision`** (`archlint/no_ad_hoc_source_fingerprint_test.go:57`, table
d'exception inexistante). Mesure a l'appui : neuf champs racine au moins sortent de la famille
`facts` (`bomb_armings.go:210`, `bomb_carries.go:147`, `bomb_stats_document.go:92` et `:93`,
`build_objective_objects.go:34`, `build_objectives_live.go:227` et `:234`, `objectives.go:189`,
`skull_carries.go:460`), tout le reste des familles pistes / inventaire / calques / vehicules /
zones, qui lisent `grammar`.

Le manquement reste une **donnee**, mise en mots par le badge (`replaySchemaStatusLogic.ts:19-21`,
constat R2-2), FR **et** EN, parite par typage (`i18nContract.ts:10`). En FR, sans anglicisme :
« lecture des octets », « profil », « grammaire », « faits », « publication ».

### 3.7 Le support de la mesure M4-D1, **decide**

V17 (M4-D1, plan `:105`) demande que 4.1 « publie la taille occupee **dans la page admin** ». La
revision 1 de cette note rendait la decision a l'executant. Elle est prise ici, sur mesure.

**Support 1 — le badge admin de la route de rejeu** (`ReplaySchemaBadge.tsx:83`, `if (!isAdmin)
return null`, monte a `.../matches/$matchId/replay.tsx:217-222`), livre au lot 4.4.2. Il porte la
taille des faits **du film affiche**.

**Support 2 — une ligne dans la page admin de monitoring.** La mesure demandee par le pilote —
`grep -rl "cache" apps/web/src/routes/admin/` — **ne rend rien**, et c'est un faux negatif : les
12 routes de `src/routes/admin/` sont des coquilles d'une ligne qui montent une page de
`features/admin/`. La mesure etendue aux pages montrees tranche :

| Element | Fichier:ligne |
|---|---|
| Section qui **publie deja des tailles sur disque** (« runtime Go, restarts, disque, tailles des bases DuckDB + WAL ») | `src/features/admin/system/ResourcesSection.tsx:1-6`, montee par `/admin/system` (`src/routes/admin/system.tsx:7`) |
| Formatage en octets | `src/features/admin/format.ts` (`formatBytes`) |
| Source | `GET /admin/monitoring/resources` — `internal/api/handlers/admin_monitoring.go:415`, runner `internal/api/wire/registry_monitoring_resources.go:25` |
| Type porteur | `domain.AdminResourcesResponse` (`internal/domain/admin_monitoring.go:204`), qui porte deja `Disk`, `Databases []ResourceDBFile` et `DBTotalBytes` |

**Decision : les deux supports.** La ligne de `/admin/system` est le lieu naturel — c'est deja la
page ou l'on va voir ce que le disque porte. Cout : un champ de plus sur
`domain.AdminResourcesResponse`, un `os.ReadDir` + `Stat` sur un dossier **plat par titre**
(`FilmFactsDir(slug)`), une ligne de rendu. **A ne pas oublier** : ce champ change le contrat
OpenAPI, donc `make openapi-gen` + `make generate-types` + `TestOpenAPIYAMLIsUpToDate`.
`internal/api/openapi_schema_semantics_test.go:128` ne liste que `db_inventory_status` pour ce
type : un champ de plus n'y touche pas.

### 3.8 4.4.1, seconde moitie : ce qui est livrable, et ce qui ne l'est pas

La case 4.4.1 du plan (`:5013-5015`) promet deux choses. **Une seule est atteignable a M4**, et il
faut le dire avant d'ecrire le lot plutot que de cocher la case pour la moitie de son texte.

| Promesse | Etat | Preuve |
|---|---|---|
| « **un changement de publication ne redecode pas** » | **LIVRABLE** — c'est exactement le verdict `republier` (§3.5), adosse aux faits de 4.1 | `artifact_digest.go:54` remplace par un verdict a trois sorties ; les six sites y passent |
| « **un changement de grammaire ne recuit que ce qui en depend** » | **NON ATTEIGNABLE en l'etat** | les quatre revisions sont des **constantes de compilation posees ensemble** (`coverage_decoder.go:167-172`) : des que `grammar.Rev` bouge, tout artefact porte une `grammarRev` differente, et « ce qui en depend » vaut **tout le parc** — le comportement d'aujourd'hui. Une empreinte par calque, qui seule le rendrait vrai, est **interdite hors de `film/revision`** (`archlint/no_ad_hoc_source_fingerprint_test.go:57`, aucune table d'exception) : c'est un lot a part entiere, pas un item de 4.4 |

Ce que `layers` apporte reellement, et qui n'est pas rien : la **granularite de couche** rend le
verdict lisible et le badge honnete (`grammar` a bouge, `publication` seule a bouge), et fait
tomber la recuisson a une republication chaque fois que **seule** la publication bouge — le cas le
plus frequent, puisque c'est ce que fait toute montee de schema.

**Reformulation proposee de la case 4.4.1, pour l'utilisateur — non decidee ici** :

> « 4.4.1 `replaybuild.Digest` porte les revisions de couche ; « a recuire » se decide par
> couche ; un changement de publication **republie depuis les faits au lieu de redecoder** ; un
> changement de couche redecode ce qui en depend — et, les quatre revisions etant posees ensemble
> a la compilation, « ce qui en depend » reste aujourd'hui tout le parc. »

### 3.9 M4-P4, la republication de cloture : **l'outil existe**

V17 (M4-P4, plan `:105`) : « a la cloture de M4, une seule passe depuis les faits partout ou c'est
possible, redecodage seulement la ou les faits manquent. » La revision 1 de cette note ne lui
donnait ni commit ni gate. Mesure : **il n'y a pas d'outil a ecrire**, seulement un rapport a
completer.

| Element | Mesure |
|---|---|
| La commande | `go run ./cmd/levelup backfill-replay --title halo_infinite --only-existing` — `cmd/levelup/main.go:133`, drapeaux `cmd_backfill_replay.go:193-208` |
| Ce que `--only-existing` fait | « ne traiter que les matchs qui ont deja un artefact sur disque (passe d apres un bump de schema) » (`:200-201`) ; filtre a `filtrerEtTrierReplay`, `:225-237` |
| Ce qui decide, film par film | `replaybuild.ArtifactUpToDate(path)` (`:232`) — **l'un des cinq sites qui passent au verdict a 4.4.1** : la passe devient donc « republier / redecoder » sans une ligne de plus |
| Un decodage a la fois | chaque film est un enfant (`executerPasseReplay`, `cmd_backfill_replay_passe.go:44`), qui prend `AcquireSoloWait` (`cmd_backfill_replay_child.go:80`) |
| Ce qui manque, et c'est tout | le rapport de passe (`replayBackfillReport`, `cmd_backfill_replay_passe.go:26-41` ; affichage `:198-215`) ne sait pas dire **republie** contre **redecode** |

**Le commit de M4-P4 appartient au brief 4.2+4.4** (il vit dans les memes fichiers que le verdict)
et tient en deux gestes : deux compteurs `republies` / `redecodes` dans `replayBackfillReport`,
alimentes par le verdict, et leurs deux lignes dans `afficherRapportReplay`. Son **gate** est la
passe elle-meme, jouee sur signal apres la fusion : **87 artefacts traites**, la ventilation
republies / redecodes / deja a jour consignee, l'empreinte de chaque artefact republie verifiee
(`digest.Of` sur les octets, deja l'etape `artifact` du TSV), et la duree totale consignee au §5
du plan.

### 3.10 Decoupage en commits, et les gates

| # | Commit | Gate |
|---|---|---|
| C1 | **4.2.1-a** `film/replay/layers.go` (table nom -> couche + fermeture), **aucun champ ajoute**, la fonction n'est appelee par personne | tests de `film/replay` ; `archlint` ; **empreinte de forme INCHANGEE** (controle : `document_shape.golden` absent du diff) ; aucun decodage |
| C2 | **la montee 61 -> 62** : les 12 points du §3.3 + les deux plafonds de taille releves | regime COMPLET (V2) : `go test ./...`, `-tags=integration -p 1`, `TestOpenAPIYAMLIsUpToDate`, `make check-types`, `make test-web` ; **equivalence 20 films : une seule etape divergente attendue, la publication**, difference limitee aux champs neufs, prouvee champ par champ (precedent : cloture M2, ADR `:468-472`) ; corpus gate 17 temoins : 0 perte, 0 changement hors blocs neufs ; consignation au §5 |
| C3 | **4.2.2** le web lit les revisions de calque (lecture + helper pur + les 17 commentaires `coverage.X` re-pointes) | `make check-types`, `make test-web`, fixtures v62, **aucun changement de rendu** |
| C4 | **4.4.1** `Digest` + verdict + les cinq appelants + `wouldDowngrade` | tests Go cibles + integration `-p 1` ; **une passe de verification SANS decodage** : mutation du champ `layers` d'un artefact de test, le verdict bascule dans les deux sens |
| C5 | **4.4.2** le badge par couche | `make check-types`, `make test-web`, test du badge en FR **et** en EN ; controle visuel, sans rituel de mesure impose |
| C6 | **M4-P4** les deux compteurs de la passe de republication (§3.9) | tests de `cmd/levelup` verts ; la passe elle-meme est un geste de pilote, sur signal, apres la fusion |

---

## 4. Les lots M3 en vol : ce qu'ils mutent, et pourquoi la regle d'ordre les rend sans objet

### 4.1 Mesure du jour — `git log --name-only 492cb0923..<branche>`

| Lot | Commits | Fichiers que M4 voulait rouvrir, et qu'il mute |
|---|---:|---|
| **3.4.1** (marche des morts, largeurs d'axe) | 9 | `film/decfilm/decfilm.go` (**+3 symboles**, 434 -> 446 L) · `film/replay/route_profil_calibre_test.go` · `internal/archlint/film_layers_deps_test.go` (−1 L) · `internal/replaybuild/replaybuild.go` (−1/+1) · `internal/replaybuild/kills.go` · `internal/sync/killcollector/{collector_run,map_identity}.go` · **les 8 fixtures web `replay_schema_61_*.json.gz` + `manifest.json`** |
| **3.1.1** (build inconnu, statut `presumee`) | **1** (`f6b37a662`) | **`film/replay/cle_du_film.go` (NEUF, 165 L, production)** · `film/replay/cle_du_film_recensement_research_test.go` (neuf) · `internal/replaybuild/replaybuild.go` (575 -> **566 L**) · `internal/replaybuild/refus_de_cuisson.go` (neuf) · `internal/replaychild/replaychild.go` · `sync/killcollector/{collector,collector_metrics,collector_run,registry_flags,roster}.go` · `sync/replayartifacts/{artifacts,cuisson}.go` · `cmd/levelup/cmd_backfill_replay_child.go` |
| **3.6.a** (composants ti=9) | 1 (`2dd16f357`) | `film/internal/grammar/*` (composants, `grammar.Rev`, `ecs_table.tsv`) · `film/types/testdata/shapes.golden` · **les 8 memes fixtures web + `manifest.json`** |

### 4.2 Les quatre collisions que la revision 1 avait manquees

1. **`decfilm.go` est mute par 3.4.1** : la revision 1 concluait « aucune collision » pour 4.1.1-a
   et faisait geler un plafond a 163 sur `492cb0923`. Gele la, il rougit a la fusion (166).
2. **`film/replay/` est mute par DEUX lots M3** : 3.4.1 (`route_profil_calibre_test.go`) et 3.1.1
   (`cle_du_film.go`, production, plus son test de recensement). La phrase « aucun lot M3 ne mute
   `film/replay/` » de la revision 1 etait fausse, et la FRONTIERE « tu es SEUL muteur de
   `film/replay/` » du brief 4.1 l'aurait ete aussi.
3. **`internal/archlint/film_layers_deps_test.go` est mute par 3.4.1**, le fichier meme que le
   commit (1) de 4.1 devait modifier.
4. **`internal/replaybuild/replaybuild.go` est mute par les deux** — 3.4.1 (+1/−1) et 3.1.1
   (575 -> 566 L). Au passage, 3.1.1 **rend 9 lignes de marge** sous le plafond de 577
   (`film_file_size_test.go:123`), ce qui desserre la bascule du commit (4) de 4.1.

**Toutes les quatre sont sans objet des lors que la regle d'ordre du §0.2 s'applique** : les lots
M4 partent d'une base qui contient deja les trois fusions, et tout ratchet gele s'y mesure a
l'entree du lot. Elles sont listees ici parce qu'un plan qui ne dit pas *pourquoi* une collision a
disparu la laisse revenir.

### 4.3 La frontiere entre les deux executants M4, par fichier

- **Executant 4.1** : seul muteur des fichiers **neufs** de `film/replay/` (le codec promu), de
  `internal/domain/title/`, de `internal/scheduler/replay_purge_cron.go`, et de
  `internal/replaybuild/replaybuild.go` a son commit (4).
- **Executant 4.2+4.4** : seul muteur de `film/replay/{document.go, document_chronicle.go,
  coverage.go, layers.go, structure_test.go, testdata/document_shape.golden}`, de
  `internal/domain/replaydoc/`, de `internal/service/replayview/`, de `apps/go-api/contracttest/`,
  de `api/openapi.yaml`, de tout `apps/web/`, et de
  `internal/replaybuild/{artifact_digest.go, artifact_store.go}` a son commit C4.
- **Partages, en AJOUT SEULEMENT** : `internal/archlint/` (le compteur de surface a 4.1, les deux
  plafonds de taille a 4.2), le plan (chacun ses cases et ses lignes §4/§5).
- **La machine** : un seul decodage a la fois, et les deux gates de decodage (C2 et 4.1.3)
  s'ordonnent sur signal du pilote.

---

## 5. Decouvertes, avec leur verdict

> Regle appliquee : seul developpeur, pragmatisme — une decouverte hors plan ne devient pas un
> lot. « Retenue » = elle entre comme **point d'un commit deja prevu**. « Non retenue » = elle est
> consignee et rien de plus.

| # | Decouverte (mesuree) | Verdict |
|---|---|---|
| 1 | **`no_second_artifact_sink_test` ne garde pas ce que l'item 4.1.2 du plan croit** : il compte les appels a `SetArtifactStoredSink(` (`:25`, notification Discord groupee), avec deux appelants autorises (`:29-32`). Le garde-rail « un seul puits d'octets d'artefact » **reste a ecrire** ; ce que 4.1.2 doit preserver est le passage par `writeArtifactBytes` (`artifact_store.go:168`) | **RETENUE** — correction du texte du plan dans le commit 4.1.2, gate reformule (§2.7) |
| 2 | **Le commentaire de budget des fixtures a derive** : `golden_inputs_budget_test.go:26` annonce 11 044 446 o « UNE SEULE MESURE FAIT FOI », le disque en porte **11 049 200** (+4 754). Le test reste vert (il n'assertit que le plafond de 12 Mio) | **RETENUE** — corrigee au commit 3 de 4.1 |
| 3 | **17 calques commentes dans `replayNormalize.ts`**, pas 16 (oubli d'`equipmentPlacements`, `:109`) | **RETENUE** — corrigee dans le commit 4.2.2 |
| 4 | **D11 du plan est faux d'un fichier** : 4.2.2 touche `replayNormalize.ts` **et** `replayDocumentSchema.ts:102` | **RETENUE** — point 10 de la checklist §3.3 |
| 5 | **Rien ne compte la surface de la facade** : 163 (166 apres 3.4.1) ne vit que dans deux commentaires et l'ADR | **RETENUE bornee** — un compteur date et son compagnon (§1.4), **pas** de lot de reduction |
| 6 | **`film/replay` pese 242 identifiants distincts cites hors de `film/`** (mesure reproductible au §1.4), 1,5 fois la facade ; l'ADR `:335` en annoncait 239 a la cloture M2 | **RETENUE** comme mesure d'entree du plafond compagnon ; aucune action de reduction |
| 7 | **Le meme film est decode deux fois par killsource sur deux chemins** : `sync/killcollector/collector_run.go:128` (pour la base) et `replaybuild/kills.go` (pour l'artefact) | **NON RETENUE** — hors perimetre de M4, consignee au §4 du plan |
| 8 | **La population de `[]types.StatRecord` par film n'est mesuree nulle part** — aucune etape du TSV ne l'expose | **RETENUE** — mesure obligatoire du commit 3 de 4.1 |
| 9 | **Le godoc de la facade compte par section, l'arbre par paquet de destination** : 10 alias de `film/types` sont ranges sous `source` / `killsource` / `objectives`. Le total est juste, la ventilation ne l'est pas | **NON RETENUE** — elle devait etre corrigee par le commit qui retirait ces alias, et ce commit est abandonne (§1.3) |
| 10 | **Les six enveloppes `dir string` de la facade** (`decfilm.go:113, 137, 171, 188, 191, 405`) ont plus d'un appelant de production ; leur critere de retrait est deja ecrit (`archlint/no_film_reread_test.go:11-15`) | **NON RETENUE** — le critere existe, il se declenchera seul |
| 11 | **`internal/replaychild/replaychild.go:196` est un sixieme site de verrou solo**, le chemin post-sync de production | **RETENUE** — nommee dans le brief 4.1 |
| 12 | **`profile.Profile` porte une interface** (champs non exportes) : s'il devait entrer au fichier de faits — il n'y entre pas, `FilmIdentity` suffit — il faudrait une forme plate | **NON RETENUE** — note pour 4.4 |
| 13 | **Le codec a onze lecteurs de test, pas cinq**, et le type `goldenInputs` vit hors des cinq fichiers a promouvoir (`golden_inputs_test.go:270`) | **RETENUE** — elle commande le decoupage du §2.7 |
| 14 | **`golden_inputs_film_test.go` ne peut pas monter en production** : il appelle `source.LoadDir` (`:69`) et R4 balaie exactement les `.go` non-test de `film/replay/` (`film_layers_deps_helpers_test.go:175`) | **RETENUE** — ecrite dans le brief 4.1 |
| 15 | **`structure_test.go:1164` epingle `SchemaVersion != 61` en dur** — le seul epinglage dur du depot | **RETENUE** — point 12 de la checklist §3.3 |
| 16 | **Les 8 fixtures web changent de NOM au schema 62** (`contract_fixtures_test.go:150`), et **aucun garde-rail ne detecte un orphelin** | **RETENUE** — point 11 de la checklist §3.3, suppression a la main dans le meme commit |
| 17 | **Le message d'erreur de `film_file_size_test.go` promet une exception que le code n'implemente pas** : `:182-187` annonce « Seule exception ecrite : document_chronicle.go ... » puis fait `t.Errorf` + `continue`, sans branche d'exemption — l'exception est **procedurale** (le droit de relever l'entree dans la table), pas automatique, et le message laisse croire l'inverse | **NON RETENUE** — une ligne consignee ; le droit de relever les deux entrees est accorde explicitement au §3.3 |
| 18 | **`filmcache` range les chunks a plat (`data/cache/film_chunks/{short8}`) et les artefacts par titre** ; le plan range les faits par titre | **NON RETENUE** — tranche au commit 2 de 4.1 : **par titre**, la cle de cuisson porte deja `titleSlug`, la purge et la mesure admin sont par titre |

---

## 6. Les 24 constats de la critique, et leur sort

| # | Constat | Sort |
|---|---|---|
| 1 | 4.4.1 seconde moitie non couverte | **applique** — §3.8, avec la reformulation proposee a l'utilisateur |
| 2 | les autres cases ont un commit | verifie, sans effet |
| 3 | « 2 532 lignes » designe huit fichiers, pas cinq (1 909) | **applique** — §2.1 |
| 4 | le deplacement des cinq ne compile pas (`goldenInputs` a `:270`) | **applique** — §2.1, §2.7 |
| 5 | 87 artefacts, pas 1 589 films | **applique** — §0.3 reecrit |
| 6 | 166 declarations / 446 L sur 3.4.1 | **applique** — §1.1 |
| 7 | `apps/go-api/contracttest/`, pas `internal/contracttest/` | **applique** — §3.1, §3.3, et les deux briefs |
| 8 | le plafond compagnon n'est pas reproductible | **applique** (et le chiffre **refute**) — §1.4 |
| 9 | 3.4.1 mute `decfilm.go` | **applique** — §4.1, rendu sans objet par §0.2 |
| 10 | 3.4.1 mute `route_profil_calibre_test.go` | **applique** — §4.2, et **etendu** : 3.1.1 mute `film/replay/` aussi |
| 11 | 3.4.1 mute `film_layers_deps_test.go` et `replaybuild.go` | **applique** — §4.1 |
| 12 | la table des tailles bloque deux commits | **applique et etendu** — §3.3 (deux fichiers, pas un : `structure_test.go` aussi) |
| 13 | les 8 fixtures changent de nom | **applique** — §3.3 point 11 |
| 14 | 3.1.1 n'a aucun commit | **REFUTE** — §6 ci-dessous |
| 15 | V20 (1) re-arbitre par le pilote | **applique** — §0.3, V20 (1) tient |
| 16 | M4-D1 sans support nomme | **applique** — §3.7, deux supports decides |
| 17 | M4-P4 sans commit ni gate | **applique** — §3.9 |
| 18 | quels fichiers deplacer au commit (1) | **applique** — §2.7 |
| 19 | ou vit la porte des faits | **applique** — §2.7, option retenue : pas de paquet neuf |
| 20 | methode du plafond a 242 | **applique** — §1.4 |
| 21 | droit de relever `film_file_size_test.go` | **applique** — §3.3, accorde explicitement |
| 22 | support de la mesure M4-D1 | **applique** — §3.7 |
| 23 | base `__BASE__` non resolue pour le ratchet | **applique** — §0.2 ; et le commit (6) qui posait le probleme est supprime (§1.3) |
| 24 | ligne de gate `./internal/contracttest/` | **applique** — les deux briefs corriges |

**Les six constats refutes ou corriges sur pieces** sont au compte rendu de cette revision ; le
plus important est le **14**, qui gouvernait toute la serialisation : `feat/decfilm-311` porte
**un commit** (`f6b37a662`, 18 fichiers, +1 069/−29), et ce commit **cree un fichier de
production dans `film/replay/`**. C'est l'inverse du « lot dont aucun octet n'est encore
committe », et c'est la collision la plus serieuse des quatre.
