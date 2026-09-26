# Audit — couverture des calques d'objectif du rejeu 2D contre l'oracle API (2026-09-10)

> Worktree `LevelUp-wt-couverture-objectifs`, branche `wt/couverture-objectifs` (depuis
> `feat/v75` a `4f272ee34`). **Aucune base DuckDB ouverte** (un serveur de developpement les
> tient), **aucune recuisson**, **aucune correction de code** : conformement au skill
> `adversarial-audit` §0, cet audit produit un registre, pas un diff.

## 0. Verdict en cinq lignes

1. **Le drapeau SUR-mesure massivement le portage** : 3 246,4 s publiees pour 2 121,2 s a
   l'oracle sur les 11 films CTF qui portent le calque, soit **153,1 %** — et **69 joueurs sur
   95 sont au-dessus de leur propre oracle**. C'est l'inverse exact du crane (78,9 -> 82,2 %
   au rapport 6.2). La cause est ECRITE dans le code : le lacher volontaire n'est borne par
   aucune chaine (`flag_carries.go:26`).
2. **Les ACTIONS du drapeau, elles, sont exactes** sur les CTF a 8 joueurs : ratio 1,000 sur
   `flag_grabs`, `flag_captures`, `flag_steals`, `flag_returns`, `flag_carriers_killed`,
   film par film — sauf trois compteurs qui explosent sur un joueur, et `flag_secures`
   (223 actions a l'oracle, **0 publiee, jamais nommee** : `named.go:90-102`).
3. **Tout ce qui depasse 8 joueurs est faux ou muet.** Le statborg ne lit que 8 slots joueur
   (`statborg.go:104-106`). Les 3 films BTB du parc (26 a 36 participants) publient
   0 s de portage pour 266,2 s a l'oracle, et 129 actions d'objectif dont 65 « prises de
   drapeau » sur un film qui n'en compte **4** en tout.
4. **Le defaut « `CompletedByLines` mono-manche » NE frappe PAS le drapeau** : les deux CTF
   multi-manche du parc (`7fce3219`, `cde26226`) rendent `noSlot = 0` et des actions exactes.
   Ce qui frappe, c'est une **manche FANTOME** : `e60aaf06` (Strongholds:Arena, score 200-82)
   declare 3 manches alors que **toutes** ses series de score n'en portent qu'une — et perd
   **84,4 % de ses actions**.
5. **Aucun temps d'occupation de zone par joueur n'est publie** : 3 507,2 s a l'oracle
   (`time_in_zones_seconds`, 6 films), 0 s publiee. `zone_attribution.go` attribue des
   ACTIONS a des zones, jamais un sejour a un joueur.

---

## 1. Cadrage

**Perimetre.** Les calques d'objectif du rejeu 2D et leur rendu :
`apps/go-api/internal/analysis/replay/` (`flag_carries*.go`, `skull_carries*.go`,
`bomb_carries.go`, `bomb_armings.go`, `zone_states*.go`, `hill_hold_ticks.go`,
`zone_attribution.go`, `objectives.go`, `held_object_carry.go`, `coverage.go`),
`apps/go-api/internal/analysis/objectiveevents/` (identite par manche, table des
emplacements), `apps/go-api/internal/replaybuild/`, `apps/web/src/features/match-replay/`.

**Axes** (six, imposes par l'instruction) : 1 temps de portage / temps en zone ·
2 actions · 3 identite par manche · 4 bornage des periodes · 5 objet porte sans position ·
6 modes sans oracle et modes sans film.

**Contrat de reference.** L'ORACLE OFFICIEL, pas une opinion : vue
`match_objective_stats_latest` exportee le 2026-09-10 serveur arrete
(`.ai/V7.5/replay2d/registre_film/oracle_vague6_*.tsv`, committee au premier commit de cette
branche). Plus la doctrine : CLAUDE.md regles 3, 7, 14 ; anti-patrons 6, 9, 10.

**Ce qui N'EST PAS un constat** (dette assumee) : la baseline lint ; les exemptions de seuil
commentees ; la reserve ecrite de `carried_open` (l'incertitude est publiee, pas cachee) ;
les 4 portages « debordants » du rapport 6.2 §2.5, dont l'oracle prouve qu'ils sont REELS et
deja sous-mesures.

---

## 2. Methode et calibrage

### 2.1 L'instrument

`.ai/V7.5/outillage/couverture_objectifs/` (Go, 5 fichiers, 1 039 L, `go vet` propre).
Il lit **les 64 artefacts deja cuits** de `data/cache/replays/halo_infinite/` et **les trois
TSV d'oracle**, et n'ouvre aucune base. Ses huit sorties brutes sont committees sous
`.ai/V7.5/replay2d/registre_film/vague6_*.tsv|log` : chaque chiffre de ce document s'y relit.

Conventions, ecrites pour etre contestables :

- un **span** de `flagCarries` couvre les images `[t0, t1]` **bornes incluses** — duree
  `(t1 - t0 + 1) x frameIntervalMs`, soit 100 ms par image ;
- une **periode de portage** est une suite maximale de spans `carried`/`carried_open`
  **contigus** du **meme xuid** (`spansOfTransitions`, `flag_carries_lives.go:351-375`,
  decoupe un meme portage en plusieurs spans a chaque changement de position) ;
- la **position d'un porteur a une image** reproduit `posOfPlayerAt`
  (`apps/web/src/features/match-replay/model/livesPosition.ts:54-77`) : une vie qui couvre
  l'image, sinon la derniere position d'une vie close depuis moins de 15 images
  (`KILLPOS_WINDOW_MS = 1 500` ms, `livesPosition.ts:42`), sinon rien.

### 2.2 Calibrage — deux moities, et celle qui n'est pas reproductible

**Le calibrage demande par l'instruction — « 82,2 % de l'oracle sur les films Oddball cuits » —
n'est PAS reproductible ici, et la raison est mesurable** : `grep -l '"skullCarries"'` sur les
64 artefacts rend **zero fichier**, et le recensement (`vague6_couverture_parc.tsv`, colonne
`skullCarries`) le confirme film par film. **Aucun film Oddball n'a d'artefact au parc** — les
82,2 % viennent d'une CUISSON hors ligne (`replay-build`) faite par le lot 6.2, pas d'une
lecture du parc. Le rapport 6.2 le dit lui-meme en §1.1. Le reproduire exigerait de recuire
quatre films, ce que le cadrage de cet audit interdit.

Les deux moities de l'instrument ont donc ete calibrees separement
(`vague6_calibrage.log`) :

| # | temoin | attendu | mesure | verdict |
|---|---|---|---|---|
| 1 | spans `carried` recalcules vs `coverage.flagCarries.carries` | egalite | **11 films / 11** | l'instrument lit ce que le producteur ecrit |
| 2 | somme `time_as_skull_carrier_seconds` des 4 films Oddball | 1 249,0 s (rapport 6.2 §3.3) | **1 249,0 s** | la lecture de l'oracle est juste |
| 3 | les trois porteurs nommes au rapport 6.2 §3.3 | 51,1 / 62,3 / 40,8 s | **51,1 / 62,3 / 40,8 s** | a la decimale |

### 2.3 Le parc mesurable, et ce qu'il ne couvre pas

64 artefacts, tous presents a l'oracle. 11 matchs de l'oracle **n'ont pas** d'artefact :
`d9781168 51ebbc0f c88ec007 43716616 24dbb67d 60ae07c4 92f18088` (les 7 Oddball),
`64e8adfa fb1a1a72` (CTF), `1c4c63c2` (One Flag), `a349fea8` (BTB Heavies:Total Control).

| famille | oracle | avec artefact | mesurable ici |
|---|---|---|---|
| CTF (toutes variantes) | 16 | **13** | 11 avec calque de portage + 2 BTB sans |
| Strongholds | 5 | **5** | 5 |
| Total Control | 2 | **1** | 1 (sans calque de zones) |
| Oddball | 7 | **0** | rien — cf. §2.2, chiffres au rapport 6.2 |
| KOTH | 0 | 0 | rien — cf. §9 |
| Assaut | 0 | 0 | rien — **et l'API n'a aucune colonne bombe** (§9) |

---

## 3. CTF — le mode le mieux couvert, et celui qui se trompe le plus en duree

### 3.1 Axe 1 — temps de portage publie contre l'oracle

Source : `vague6_couverture_portage.tsv` (95 lignes joueur x film).

| film | carte | manches | periodes | publie | oracle | ratio | joueurs > oracle |
|---|---|---|---|---|---|---|---|
| `16ea3668` | Aquarius | 1 | 16 | 212,8 s | 131,0 s | **1,624** | 6 / 6 |
| `58864b3c` | Domicile | 1 | 10 | 159,5 s | 110,1 s | **1,449** | 6 / 6 |
| `7fce3219` | Takamanohara | 2 | 22 | 261,7 s | 173,9 s | **1,505** | 8 / 8 |
| `8bc6074f` | Origin | 1 | 18 | 348,4 s | 209,3 s | **1,665** | 8 / 8 |
| `a0c36016` | Forest | 1 | 24 | 263,3 s | 156,5 s | **1,682** | 7 / 8 |
| `b8a44fe8` | Forest | 1 | 25 | 696,2 s | 401,1 s | **1,736** | 6 / 7 |
| `bc60b4d9` | Illusion | 1 | 20 | 163,1 s | 120,2 s | **1,357** | 7 / 7 |
| `bf5ced1b` | Illusion | 1 | 5 | 49,6 s | 33,1 s | **1,498** | 3 / 3 |
| `cde26226` | Critical Dewpoint | 2 | 25 | 672,9 s | 480,4 s | **1,401** | 8 / 8 |
| `f8efc5ca` | Absolution | 1 | 23 | 288,8 s | 196,0 s | **1,473** | 7 / 7 |
| `4ecdf3e7` | High Ground (drapeau neutre) | 1 | 9 | 130,1 s | 109,6 s | **1,187** | 3 / 5 |
| **11 films** | | | **197** | **3 246,4 s** | **2 121,2 s** | **1,531** | **69 / 95** |
| `4f77afc1` | Flood Gulch (BTB) | 1 | 0 | **0,0 s** | 73,2 s | **0,000** | 0 / 9 |
| `879a4dba` | Fortitude (BTB) | 1 | 0 | **0,0 s** | 193,0 s | **0,000** | 0 / 13 |

Distribution du ratio par joueur, sur les 73 joueurs des 11 films a calque et dont l'oracle
est non nul : **min 0,000 · p25 1,106 · mediane 1,380 · p75 1,969 · max 63,000**.

**Le controle exige par l'instruction ECHOUE, et c'est le constat principal de l'audit :
69 joueurs sur 95 (72,6 %) depassent leur propre oracle.**

Contre-exemples nommes (film, xuid, publie -> oracle) :

| film | xuid | periodes | publie | oracle | ecart |
|---|---|---|---|---|---|
| `16ea3668` | 2535417044536883 | 1 | 56,7 s | **0,9 s** | **+55,8 s (63x)** |
| `b8a44fe8` | 2535442462807197 | 5 | 223,9 s | 59,0 s | +164,9 s |
| `b8a44fe8` | 2533274858283686 | 7 | 342,9 s | 271,8 s | +71,1 s |
| `cde26226` | 2535468122262494 | 6 | 115,2 s | 42,3 s | +72,9 s |
| `8bc6074f` | 2535449383340628 | 4 | 80,8 s | 42,4 s | +38,4 s |

Le cas `16ea3668` / 2535417044536883 se lit dans l'artefact a l'oeil nu : un unique span
`carried [2317..2883]` (56,7 s) pour un joueur dont l'oracle donne **2 vols, 0 prise et
0,9 s de portage**. Il a vole le drapeau, l'a lache 0,9 s plus tard, et le calque l'a laisse
dans sa main pendant 55,8 s.

**Cause, ecrite dans le code et confirmee par la mesure.**
`apps/go-api/internal/analysis/replay/flag_carries.go:26-28` :

> « Le LACHER VOLONTAIRE n'est observable par aucune de ces chaines et n'est donc PAS borne :
> un portage qui en contient un est trop long. »

Les quatre faits qui ferment un portage (capture, mort, nouvelle prise, fin de match) ne
comprennent pas le lacher. Le fichier annonce que « le biais joue CONTRE ce qui est
affirme » ; **l'oracle chiffre ce biais pour la premiere fois : +1 129,3 s sur 197 periodes,
soit +5,73 s par periode en moyenne.**

**Levier mesurable, non exploite.** La couverture publie deja un canal capable de borner :
`coverage.flagCarries.objectLives = 37` sur `16ea3668` (les vies de l'objet drapeau), et
`closedByObject = 0` — **le canal existe et ne ferme aucun portage**. Sur le meme film,
`dropsRepositioned = 12` et `dropsWithheld = 14` : douze repositionnements de drapeau lache
sont deja lus, quatorze retenus.

### 3.2 Axe 2 — actions publiees contre l'oracle

Source : `vague6_couverture_actions.tsv`.

| action | publie | oracle | ratio | films en accord exact |
|---|---|---|---|---|
| `flag_captures` | 41 | 44 | 0,932 | 11 / 13 |
| `flag_grabs` | 443 | 397 | 1,116 | 10 / 13 |
| `flag_steals` | 193 | 141 | 1,369 | 11 / 13 |
| `flag_returns` | 97 | 101 | 0,960 | 9 / 11 |
| `flag_carriers_killed` | 88 | 98 | 0,898 | 10 / 13 |
| `flag_capture_assists` | 115 | 39 | 2,949 | 9 / 12 |
| **`flag_secures`** | **0** | **223** | **0,000** | **0 / 13** |

**Lecture, et elle est en deux temps.**

**(a) Sur les 11 films CTF a 8 joueurs, les actions sont EXACTES.** Ratio 1,000 joueur par
joueur sur `flag_grabs`, `flag_captures`, `flag_steals`, `flag_returns`,
`flag_carriers_killed` — huit films sur onze n'ont aucun ecart d'aucune sorte. Le calque
`objectives` fait ce qu'il annonce.

**(b) Trois compteurs explosent sur un joueur unique, et l'oracle les refute :**

| film | action | joueur | publie | oracle | comp |
|---|---|---|---|---|---|
| `a0c36016` | `flag_capture_assists` | 2535427927026623 | **84** | **0** | 20 B |
| `cde26226` | `flag_steals` | 2535469190789936 | **65** | **1** | 24 A |
| `bf5ced1b` | `flag_carriers_killed` | — | 0 | 1 | 21 B |
| `4ecdf3e7` | `flag_grabs` | — | 21 | 22 | 22 A |

Le cas `a0c36016` est important au-dela de son film : le comp qui explose est **20 B**, et
c'est exactement celui que l'en-tete de `maxUnrollPerStep`
(`objectiveevents/named_series.go:41-48`) cite comme « le pire deroulage d'un film SAIN »
(17 306 sur `d9781168`, comp 20 B, slot 12). **L'oracle etablit aujourd'hui que ce comp
produit des actions qui n'ont pas eu lieu** : la justification de la borne repose donc sur
une population qu'elle qualifie de saine a tort (anti-patron n° 9, doc inversee).

**(c) `flag_secures` n'est jamais publie, et ce n'est pas un bug : c'est une absence de table.**
`objectiveevents/named.go:90-102` — la table `ObjectiveTypeFlag` mappe 21 A, 20 B, 21 B, 22 A,
23 A, 24 A, 2 A, 3 A. **Aucun emplacement ne porte `flag_secures`**, et la constante n'existe
pas dans la liste de `named.go:156-168`. 223 actions de l'oracle sont hors de portee du calque
par construction. Meme constat, non chiffre faute de colonne dans l'export exploitable, pour
`zone_offensive_kills` / `zone_defensive_kills` (table `ObjectiveTypeZone`, `named.go:103-110`).

### 3.3 Axe 3 — identite par manche

Source : `vague6_identite.tsv`.

Nombre de manches : deux sources independantes sont lues dans le MEME artefact —
`coverage.score.rounds` et le nombre de manches distinctes du fil de score
(`scoreTimeline.teams[].rounds[].round`). Le registre, lui, ne porte pas de compte de manches.

| film | manches (couverture) | manches (fil de score) | `obj_available` | `noSlot` | % non nommees | `flag_noBridge` | spans portes sans xuid |
|---|---|---|---|---|---|---|---|
| `7fce3219` (CTF, 2 manches) | 2 | **2** | 348 | **0** | 0,0 % | **0** | **0** |
| `cde26226` (CTF, 2 manches) | 2 | **2** | 415 | **0** | 0,0 % | **0** | **0** |
| les 9 autres CTF | 1 | 1 | 54 a 15 808 | **0** | 0,0 % | **0** | **0** |
| `e60aaf06` (Strongholds) | **3** | **1** | 154 | **130** | **84,4 %** | 0 | 0 |
| les 4 autres Strongholds | 1 | 1 | 92 a 152 | 0 | 0,0 % | 0 | 0 |

**Reponse a la question posee (« le defaut `CompletedByLines` mono-manche frappe-t-il aussi
le drapeau, la bombe et les zones ? ») :**

- **Drapeau : NON, et c'est mesure.** Les deux seuls CTF multi-manche du parc rendent
  `noBridge = 0`, `noSlot = 0`, aucun span porte anonyme, et des actions **exactes**
  (`7fce3219` : 70 prises publiees pour 70 a l'oracle ; `cde26226` : 71 pour 71). La garde
  `if len(lines) == 0 || len(ri.byRound) != 1 { return ri }`
  (`objectiveevents/slotidentity_rounds.go:257-259`) coute **zero action et zero seconde**
  sur le parc mesurable, parce que `CompletedByElimination`
  (`slotidentity_elimination.go:83-100`, elle **par manche**) suffit sur ces deux films.
- **Zones : OUI, mais par une MANCHE FANTOME, pas par le multi-manche.** `e60aaf06` est un
  `Strongholds:Arena` de 415 s au score **200-82** — un mode a points, une manche.
  L'artefact se contredit lui-meme : `coverage.score.rounds = 3`, alors que les deux
  series d'equipe ET les huit series de joueur du fil de score ne portent **que la manche 0**.
  Consequence : 130 actions sur 154 partent en `noSlot`, et les zones tombent a
  **10 captures publiees pour 38** et **2 securisations pour 11**. Une voie de completion par
  manche ne rendrait rien ici : il n'y a **qu'une** manche a completer. La cause est en amont,
  dans la detection des manches (`RealRounds`, `objectiveevents/statborg.go:414-430`).
- **Bombe : non prouvable ici.** Aucun artefact d'Assaut au parc (§9).

**Ce qu'une voie de completion PAR MANCHE rendrait sur le parc mesurable : rien de mesurable.**
`noSlot` vaut 0 sur 18 films sur 19 ; le seul film qui perd des actions les perd pour une
autre cause. Le report du registre (« CTF MULTI-MANCHE », ligne 592) garde sa valeur sur
`fb1a1a72` et `64e8adfa` — **qui n'ont pas d'artefact au parc**, donc hors de portee de cet
audit.

### 3.4 Axe 4 — bornage des periodes

Source : `vague6_bornage.tsv`.

La decouverte 4 du rapport 6.2 (« un train est borne par son premier et son dernier tic,
1,5 a 2 s perdues par periode ») **ne s'applique pas au drapeau**, et la raison est
structurelle : les deux calques ne lisent pas le meme canal.

| calque | source de la periode | biais attendu |
|---|---|---|
| crane (`skull_carries.go:13-17`) | un TRAIN de tics `comp 0 A` | **sous**-estime : la seconde d'amorce et celle de chute manquent |
| drapeau (`flag_carries.go:17-32`) | des EVENEMENTS de statborg (prise, capture, mort, reprise) | **sur**-estime : le lacher volontaire n'est borne par rien |
| bombe / crane porte (`held_object_carry.go:20-24`) | des TRANSITIONS du canal des armes tenues | borne exacte des deux cotes |

Mesure sur les 197 periodes de drapeau : **publie 3 246,4 s, oracle 2 117,1 s, ecart
+1 129,3 s, soit +5,73 s PAR PERIODE** — et l'ecart est positif sur **10 films sur 11**
(de +2,15 a +11,80 s par periode).

**Simuler une demi-fenetre de tic aux deux bornes sur le drapeau est donc contre-indique** :
la colonne `publie_demi_fenetre_s` de la sortie ajoute 1,0 s par periode et fait passer
**71 des 72 joueurs qui ont un portage publie** au-dessus de leur oracle, contre 69 aujourd'hui.
La correction irait dans le mauvais sens.

Controle qui ecarte l'explication facile : l'etat `carried_open` (borne haute assumee) ne
represente **qu'UN span sur tout le parc** (`b8a44fe8`, 18,7 s). Il ne peut pas porter
1 129,3 s.

**Ce qui reste a trancher par l'utilisateur, avec ses deux chiffres.** Pour le CRANE
(mesure du rapport 6.2, non re-mesurable ici) la demi-fenetre reste la proposition ouverte :
82,2 % aujourd'hui, et le rapport estime le manque a 1,5-2 s par periode. C'est un
changement de DEFINITION du portage. **Pour le DRAPEAU, la decision produit est l'inverse** :
faut-il fermer un portage sur un lacher deduit (et de quelle facon) ? Les deux chiffres a
poser sur la table : **1,531** aujourd'hui, **1,000** vise.

### 3.5 Axe 5 — drapeau porte sans position du porteur

Source : `vague6_objet_sans_position.tsv`.

| film | images portees | images muettes | % | periodes | periodes avec trou | expliquees par un vehicule |
|---|---|---|---|---|---|---|
| `b8a44fe8` | 6 962 | 258 | 3,7 % | 25 | 1 | 0 |
| `f8efc5ca` | 2 888 | 75 | 2,6 % | 23 | 1 | 0 |
| `cde26226` | 6 729 | 134 | 2,0 % | 25 | 1 | 0 |
| `8bc6074f` | 3 484 | 52 | 1,5 % | 18 | 1 | 0 |
| `58864b3c` | 1 595 | 12 | 0,8 % | 10 | 1 | 0 |
| `4ecdf3e7` | 1 301 | 2 | 0,2 % | 9 | 1 | 0 |
| 5 autres films | 10 505 | **0** | 0,0 % | 87 | 0 | 0 |
| **total CTF** | **32 464** | **533** | **1,64 %** | **197** | **6** | **0** |

A comparer aux **694 images / 7,0 %** du parc Oddball (rapport 6.2 §2.1). Le drapeau est
**quatre fois moins touche**. Aucune image muette n'est expliquee par une occupation de
vehicule : les 11 films a calque sont des CTF:Arena, sans vehicule.

**Ce que le rendu fait aujourd'hui, et pourquoi le stopgap ne se transpose PAS tel quel.**

| calque | ligne | comportement sans position |
|---|---|---|
| drapeau | `apps/web/src/features/match-replay/layers/flagCarriesLayer.ts:237-241` | `const p = now.xuid ? posOf(now.xuid, frame) : null; if (p) return p` puis **`return { x: now.x, y: now.y }`** — le drapeau se dessine a l'ANCRE DU SPAN, une position PERIMEE |
| crane | `layers/skullCarrierLayer.ts:100-101` | `const w = layer.posOf(c.xuid, frame); if (!w) continue` — **le crane disparait** |
| bombe | `layers/bombCarrierLayer.ts:129-130` | idem — **la bombe disparait** |

Et `model/skullPresence.ts:65` donne la precedence a `carried` : le calque de l'objet LIBRE
se tait pendant qu'aucun glyphe n'est dessine.

**Verdict par calque sur le stopgap consigne (« rendre l'objet LIBRE quand le porteur n'a pas
de position ») :**

- **crane** : s'applique tel quel — c'est le cas d'origine, l'objet est invisible ;
- **bombe** : s'applique tel quel, meme code, meme symptome (non mesurable faute d'artefact) ;
- **drapeau** : **NE s'applique PAS tel quel**. Le drapeau n'est jamais invisible ; il est
  dessine a une position perimee, ce qui est un defaut DIFFERENT (et discutablement pire :
  un objet immobile faux se lit comme un fait). « Libre » est par ailleurs un etat publie qui
  a un sens precis (le drapeau au sol, `dropped`), et le forcer pendant un portage reel
  contredirait `flagCarries`. La bonne forme pour le drapeau est a decider a part.

---

## 4. Strongholds

### 4.1 Axe 2 — actions

| film | manches | `zone_captures` publie/oracle | `zone_secures` publie/oracle |
|---|---|---|---|
| `32d9a94f` (Perilous) | 1 | 39 / 39 — **1,000** | 16 / 16 — **1,000** |
| `396cfc92` (Illusion) | 1 | 48 / 48 — **1,000** | 10 / 10 — **1,000** |
| `572e236b` (Fortress) | 1 | 36 / 36 — **1,000** | 9 / 9 — **1,000** |
| `81c02726` (Isolation) | 1 | 28 / 28 — **1,000** | 7 / 7 — **1,000** |
| `e60aaf06` (Banished Narrows) | **3 (fantomes)** | **10 / 38 — 0,263** | **2 / 11 — 0,182** |
| **total** | | 161 / 189 — 0,852 | 44 / 53 — 0,830 |

**Quatre films sur cinq sont PARFAITS, joueur par joueur.** Toute la perte du mode tient sur
`e60aaf06` et sa manche fantome (§3.3). Detail joueur par joueur dans
`vague6_couverture_actions.tsv` : les 8 joueurs de `e60aaf06` perdent chacun de 1 a 6 captures.

### 4.2 Axe 1 — temps en zone : rien n'est publie par joueur

Source : `vague6_couverture_zones.tsv`.

| film | zones publiees | possession eq. 0 | possession eq. 1 | oracle `time_in_zones_seconds` | publie par joueur |
|---|---|---|---|---|---|
| `32d9a94f` | 3 | 630,6 s | 526,2 s | 524,6 s (8 joueurs) | **0 s** |
| `396cfc92` | 3 | 698,2 s | 658,2 s | 619,8 s (8) | **0 s** |
| `572e236b` | 3 | 531,9 s | 366,4 s | 457,8 s (9) | **0 s** |
| `81c02726` | 3 | 235,4 s | 479,1 s | 316,4 s (8) | **0 s** |
| `e60aaf06` | 3 | 664,0 s | 468,9 s | 599,9 s (8) | **0 s** |
| `5676a9ba` (TC) | **0** | — | — | 988,7 s (26) | **0 s** |
| **total** | | | | **3 507,2 s** | **0 s** |

Le calque `zoneStates` publie l'etat d'une ZONE (proprietaire, jauge, activite), pas
l'occupation d'un JOUEUR. `zone_attribution.go:1-31` le dit sans ambiguite : il croise
formes x instants x positions pour attribuer **une ACTION a une zone**, jamais un sejour a un
joueur. **La couverture du temps en zone par joueur est donc de 0 % par construction, sur
3 507,2 s d'oracle.** Ce n'est pas une regression : c'est un calque que personne n'a ecrit.

**`zone_scoring_ticks` : non prouvable ici.** La colonne vaut **0 sur les 328 lignes de
l'oracle** (elle n'est peuplee que pour les modes a colline). La confrontation demandee
« `zone_scoring_ticks` vs les tics de colline publies » exigerait un match KOTH a l'oracle —
il n'y en a aucun dans l'export (§9).

---

## 5. Total Control — un seul film, et aucun calque de zones

`5676a9ba` (BTB:Total Control, Insolence, 682 s, 31 lignes d'oracle) :

- `zoneStates` : **absent** (0 zone), `coverage.zones` : **absent** — alors que les cinq
  Strongholds publient chacun 3 zones et leur couverture ;
- actions : **23 captures publiees pour 51** (0,451), **4 securisations pour 9** (0,444) ;
- portage : sans objet ;
- temps en zone : 988,7 s a l'oracle, 0 s publiee.

Deux causes se superposent et l'audit ne les separe pas : le mode (Total Control classe
`ObjectiveTypeZone` par `extract.go:119-121`, donc la table d'emplacements est la meme que
Strongholds) et l'effectif (31 participants — cf. §6). Le second film du mode, `a349fea8`
(BTB Heavies), n'a pas d'artefact.

---

## 6. BTB — le plafond de 8 joueurs, et ce qu'il publie a la place

Les trois films BTB du parc portent 26 a 36 lignes d'oracle. Le statborg, lui, ne connait
que **huit** slots de joueur — `objectiveevents/statborg.go:104-107` :

```go
// statSlotMin / statSlotMax / statTeamSlotMax delimitent les slots d'entite :
// 6 et 8 pour les deux equipes, 10 a 24 (pairs) pour les huit joueurs.
statSlotMin     = 6
statSlotMax     = 24
```

et `statborg.go:252` rejette tout slot hors de `[6, 24]` ou impair.

| film | participants | portage publie / oracle | `flag_grabs` | `flag_steals` | `flag_returns` | `flag_carriers_killed` | `flag_captures` |
|---|---|---|---|---|---|---|---|
| `4f77afc1` (BTB:CTF) | 36 | **0,0 / 73,2 s** | **65 / 4** | 2 / 13 | 2 / 6 | 4 / 11 | 1 / 2 |
| `879a4dba` (BTB:CTF) | 26 | **0,0 / 193,0 s** | 9 / 23 | 4 / 5 | — | 8 / 10 | 3 / 5 |
| `5676a9ba` (BTB:TC) | 31 | — | — | — | — | — | — |

Deux comportements, et **un seul des deux est defendable** :

1. **Le calque de PORTAGE se tait, et il a raison.** `coverage.flagCarries.flagFilm = false`
   avec `bursts = 0` : la regle `IsFlagFilm` (`objectiveevents/flagfilm.go:78-80`,
   `Bursts > 0 && Captures > 0 && Captures <= Bursts && Steals > 0`) refuse le film. Un
   silence documente vaut mieux qu'un calque faux — mais il coute **266,2 s de portage reel
   sur 22 joueurs**, dont 10 joueurs a plus de 9 s (jusqu'a 47,5 s pour 2535460487570599 sur
   `879a4dba`).
2. **Le calque des ACTIONS n'a AUCUNE garde equivalente, et il publie du faux.** Sur
   `4f77afc1`, il annonce **65 prises de drapeau** la ou l'oracle en compte **4 pour les
   36 participants reunis** (16,3x). Ce ne sont pas des prises mal attribuees : elles n'ont
   pas eu lieu. Ces 65 actions sont rendues a l'ecran comme n'importe quelle autre
   (`layers/objectivesLayer.ts:305-330`, `buildObjectivePulses` itere sur tout
   `doc.objectives`).

**Hypothese, NON prouvee ici** : le plafond de 8 slots est la cause. Ce qui est prouve, c'est
(a) la constante, (b) l'effectif reel, (c) le desaccord des compteurs. Le lien causal
demanderait de decoder un film BTB slot par slot — hors perimetre de cet audit. **A ranger
comme hypothese forte, pas comme cause.**

---

## 7. Oddball — non re-mesurable ici, chiffres du rapport 6.2

Aucun des 7 matchs Oddball de l'oracle n'a d'artefact (§2.2). Ce que l'audit a pu faire :
**verifier la moitie ORACLE des chiffres du rapport 6.2** — 1 249,0 s au total sur les quatre
films cuits hors ligne, et les trois porteurs cites a la decimale (§2.2, calibrage 2 et 3).
Les chiffres publies (985,9 -> 1 026,9 s, 78,9 -> **82,2 %**) restent ceux du rapport 6.2 §3.3
et ne sont **pas** reproduits ici.

Ce qui reste vrai et transposable :

- `43716616` : 62,3 s toujours perdues, 2 trains `noBridge` (rapport 6.2 §6, decouverte 1) ;
- 694 images de portage sans position, 7,0 % du parc Oddball (decouverte 2) ;
- sous-estimation de 1,5 a 2 s par periode (decouverte 4) — **et l'audit etablit que ce
  biais est propre aux calques a TRAIN DE TICS, cf. §3.4**.

---

## 8. KOTH — le calque existe, le parc n'a pas de film cuit

- **Aucun match KOTH dans l'export d'oracle** de la vague 6 (recensement des 75 lignes de
  `oracle_vague6_registry.tsv` : 0 KOTH). Rien a confronter.
- **Aucun artefact KOTH au parc** (recensement des 64 : 0).
- **Des films KOTH SONT au cache** : `21ece4d8` (26 chunks), `7f1bbf06` (19), `a36c8bed` (23),
  et le registre versionne `oracle_lotA_bis.tsv` en compte **47** (46 `KOTH:Arena` +
  1 `Ranked:King of the Hill`).
- **La cuisson a echoue pour une cause NOMMEE, et ce n'est pas le calque** :
  `.ai/V7.5/replay2d/registre_film/E4_cuisson_koth.log` — les 7 films tentes rendent
  `carte hors catalogue ([])`, code de sortie 10.
- **Le calque, lui, est mesure** : `hill_hold_ticks.go:8-12` — `comp 23 A` reproduit
  `ZonesStats.StrongholdScoringTicks` de l'API **exactement, joueur par joueur, sur 31 joueurs
  de 4 films** (lot E1-bis du 2026-08-30).

**Conclusion KOTH : ce qui manque est le CATALOGUE DE CARTES, pas le calque.** Chiffrer la
couverture KOTH demande (a) une cuisson reussie de 3 films au cache, (b) un export d'oracle
qui contienne ces matchs. Les deux sont hors perimetre d'un audit.

---

## 9. Assaut — pas d'oracle officiel, et il faut le dire

**L'API n'a AUCUNE colonne d'Assaut.** L'export de la vague 6 porte 42 colonnes
(drapeau, zones, crane, VIP, power seed, extraction) : aucune ne nomme la bombe. Le code le
constate deja pour son compte (`objectiveevents/named.go:128-133` : « ces statistiques ne sont
repliquees nulle part dans le film, et l'API n'en publie aucune non plus — payload
GetMatchStats des 3 variantes, 2026-08-31 »).

**Ce qui est publie** (`replay/bomb_stats.go:12-40`) : `bomb_detonations` (compteur statborg
`comp 0 A`), `bomb_grabs`, `time_as_bomb_carrier_seconds`, `bomb_arms`, plus les calques
`bombCarries` / `bombArmings`.

**Etat du parc** : **0 artefact d'Assaut** (aucun des 64 ne porte `bombCarries`), **0 match
d'Assaut a l'oracle**. Des films SONT au cache : `c75f33b8` (26 chunks), `1c01e34f` (23),
`69b16f5d` (19) ; `35b75a31` a ete purge.

**L'oracle INDIRECT possible, et ce qu'il vaut — a ne PAS confondre avec une preuve :**

| substitut | ce qu'il borne | force |
|---|---|---|
| `team_0_score` / `team_1_score` du registre | le nombre de manches gagnees, donc un MINORANT des detonations | faible : un score n'est pas un compte de detonations |
| explosions publiees du film (`bomb_detonations`) confrontees a l'anneau `ti=12` | le compte de detonations | **deja joue** : gate A4, `A4_statborg_assaut.log` — sommes exactes 4/4 films sur moities disjointes |
| morts du releve confrontees a `comp 2 B` | la sante de la lecture | deja joue : 37/37 = 100 % (meme log) |

**Ce qui n'est pas prouvable, et le restera sans decodeur externe** : `bomb_grabs` et
`time_as_bomb_carrier_seconds` par joueur. Aucune chaine independante ne les publie. Un
releve Theater manuel serait le seul temoin.

---

## 10. Registre des causes PROUVEES, classees par gain recuperable x effort

| # | cause | `fichier:ligne` | declenchement | effectif mesure | effort |
|---|---|---|---|---|---|
| **C1** | Le lacher volontaire ne ferme pas un portage de drapeau | `replay/flag_carries.go:17-32` | tout portage suivi d'un lacher non date | **+1 129,3 s de portage FAUX** sur 197 periodes / 11 films ; 69 joueurs / 95 au-dessus de leur oracle | fort |
| **C2** | Le calque `objectives` n'a aucune garde d'effectif | `objectiveevents/statborg.go:104-107, 252` (constat) ; absence de garde symetrique a `flagfilm.go:78-80` | film a plus de 8 joueurs | **129 actions publiees** sur 3 films BTB, dont **65 prises pour 4 reelles** | faible (silence) / fort (couverture) |
| **C3** | Le portage de drapeau se tait sur les films BTB | `objectiveevents/flagfilm.go:78-80` (`Bursts > 0`) | `bursts = 0` sur BTB | **266,2 s** de portage reel, 22 joueurs, 2 films | fort |
| **C4** | Manche fantome : `coverage.score.rounds` contredit le fil de score du meme artefact | `objectiveevents/statborg.go:414-430` (`RealRounds`) | film mono-manche dont le statborg tire 3 manches | **130 actions / 154** perdues sur `e60aaf06` (28 captures + 9 securisations) | moyen |
| **C5** | `flag_secures` n'est dans aucune table d'emplacement | `objectiveevents/named.go:90-102` et `156-168` | tous les CTF | **223 actions** jamais publiees, 13 films | fort (retro-ingenierie du comp) |
| **C6** | Deroulage aberrant d'un compteur sur un joueur | `objectiveevents/named_series.go:36-52` (bornes a 100 000 / 1 000 000, jamais atteintes) | comp 20 B, comp 24 A | **84 assistances de capture pour 0** (`a0c36016`), **65 vols pour 1** (`cde26226`) | moyen |
| **C7** | Aucun calque de zones sur Total Control | `replay/zone_states.go` (constat : `zoneStates` absent de `5676a9ba`) | mode Total Control | 1 film sur 1, 988,7 s d'oracle, 23/51 captures | moyen |
| **C8** | Aucun temps d'occupation de zone par joueur n'existe | `replay/zone_attribution.go:1-31` | tous les modes a zone | **3 507,2 s** d'oracle, 0 s publiee, 6 films | fort |
| **C9** | Le glyphe du drapeau se fige a une position perimee | `apps/web/.../layers/flagCarriesLayer.ts:237-241` | porteur sans position a l'image | **533 images / 32 464** (1,64 %), 6 periodes / 197 | faible |
| **C10** | Le glyphe du crane et celui de la bombe disparaissent | `layers/skullCarrierLayer.ts:100-101`, `layers/bombCarrierLayer.ts:129-130`, precedence `model/skullPresence.ts:65` | idem | **694 images** Oddball (rapport 6.2) ; bombe non mesurable | faible |

### Hypotheses NON prouvees — a ne pas traiter comme des causes

| hypothese | ce qui est prouve | ce qui manque |
|---|---|---|
| Le plafond de 8 slots (`statSlotMax = 24`) **cause** les compteurs faux du BTB | la constante, l'effectif reel (26-36), le desaccord des compteurs | le lien causal : decoder un film BTB slot par slot |
| Le deroulage aberrant de `comp 20 B` (C6) est de la meme famille que les 9 513 a 15 645 `assists`/film de `16ea3668`, `8bc6074f`, `f8efc5ca` (22 a 38 par seconde) | les comptes, et le fait que la doctrine de `maxUnrollPerStep` qualifie deja de « SAIN » un deroulage de 17 306 sur le meme comp 20 B | **l'export d'oracle ne porte PAS de colonne `assists`** : le taux est physiquement impossible, mais il n'est refute par aucun oracle ici |
| La manche fantome de `e60aaf06` vient du critere de `RealRounds` | la contradiction interne (3 vs 1), l'effectif perdu (130/154) | quel ancrage produit les manches 1 et 2 : demande de rejouer `RealRounds` sur le film |

---

## 11. Proposition de lot correctif

Ordre = gain recuperable / effort. Chaque item porte sa preuve d'entree, son gate mesurable
et sa dependance a une recuisson. **Aucun item n'est engage : c'est une proposition.**

- [ ] **L1 — Taire le calque `objectives` quand le film depasse 8 slots joueur.**
  *Preuve d'entree* : `4f77afc1` publie 65 `flag_grabs` pour 4 a l'oracle (`vague6_couverture_actions.tsv`).
  *Contenu* : une garde d'effectif symetrique a `IsFlagFilm`, avec son compteur de couverture
  nomme (`refusedByRoster`) et son `slog` — jamais un silence muet.
  *Gate* : sur `4f77afc1`, `879a4dba`, `5676a9ba`, le calque publie **0 action** ou **100 %
  d'accord avec l'oracle** ; aucun changement sur les 61 autres films (`replay-diff` : 0 perte).
  *Recuisson* : **oui**, 3 films.
  *Gain* : 129 actions fausses retirees. *Effort* : faible.

- [ ] **L2 — Rendre le glyphe porte quand le porteur n'a pas de position.**
  *Preuve d'entree* : drapeau 533 images / 32 464 (`vague6_objet_sans_position.tsv`) ; crane
  694 images (rapport 6.2 §2.1).
  *Contenu* : crane et bombe — appliquer le stopgap consigne (objet LIBRE au dernier repos) ;
  **drapeau — decision separee** : aujourd'hui il se fige a l'ancre du span
  (`flagCarriesLayer.ts:241`), ce qui affiche un fait faux plutot que rien.
  *Gate* : `skullCarrierLayer.test.ts` et `bombCarrierLayer.test.ts` prouvent le rendu sur une
  image sans position ; aucune image ou deux glyphes du meme objet coexistent.
  *Recuisson* : **non** (lot web pur).
  *Gain* : 1 227 images visibles. *Effort* : faible.

- [ ] **L3 — Fermer la manche fantome de `e60aaf06`.**
  *Preuve d'entree* : `coverage.score.rounds = 3` contre 1 seule manche dans **toutes** les
  series de `scoreTimeline` du meme artefact ; registre `Strongholds:Arena` 200-82 ;
  130 actions / 154 en `noSlot`.
  *Contenu* : instruire `RealRounds` (`statborg.go:414-430`) sur ce film ; le critere de
  contiguite + coherence laisse passer 3 manches la ou le fil de score n'en voit qu'une.
  *Gate* : `e60aaf06` passe de **10 a >= 36 `zone_captures` publiees** (oracle 38) et de
  **130 a <= 8 `noSlot`** ; les 4 autres Strongholds restent a 1,000 (aucune regression) ;
  `coverage.score.rounds` concorde avec le fil de score sur les 64 films du parc.
  *Recuisson* : **oui**, 1 film (verifier ensuite les 4 autres Strongholds : 0 perte).
  *Gain* : 130 actions rendues, dont 28 captures et 9 securisations. *Effort* : moyen.

- [ ] **L4 — Borner le deroulage par joueur et par comp.**
  *Preuve d'entree* : `a0c36016` 84 `flag_capture_assists` pour **0** ; `cde26226` 65
  `flag_steals` pour **1**.
  *Contenu* : la borne `maxUnrollPerStep = 100 000` (`named_series.go:48`) est trois ordres de
  grandeur au-dessus du phenomene. Poser une borne par (slot, comp) calee sur la mesure, et
  **reecrire l'en-tete** qui qualifie aujourd'hui de « film SAIN » un deroulage de 17 306 sur
  le comp 20 B — l'oracle le refute (anti-patron n° 9).
  *Gate* : sur les 64 films, aucun joueur au-dessus de son oracle sur une action de drapeau ou
  de zone (tolerance 0) ; `a0c36016` passe de 84 a 0 et `cde26226` de 65 a 1 ; les 11 films
  deja exacts ne bougent d'aucune action.
  *Recuisson* : **oui**, au moins `a0c36016` et `cde26226`.
  *Gain* : 148 actions fausses retirees. *Effort* : moyen.
  *Prealable* : instruire d'abord l'hypothese `assists` (§10) — la meme borne pourrait retirer
  40 806 pulses sur 3 films, ou n'en retirer aucun ; **on ne borne pas sans oracle**.

- [ ] **L5 — Fermer un portage de drapeau sur le lacher.**
  *Preuve d'entree* : +1 129,3 s (+5,73 s/periode), 69 joueurs / 95 au-dessus de leur oracle,
  pire cas `16ea3668` / 2535417044536883 : 56,7 s publiees pour 0,9 s.
  *Contenu* : exploiter le canal DEJA lu et DEJA compte —
  `coverage.flagCarries.objectLives = 37` et **`closedByObject = 0`** sur `16ea3668`. Une vie
  d'objet drapeau qui reapparait au sol DATE le lacher.
  **Changement de definition : a trancher par l'utilisateur** (le calque cesserait d'etre une
  borne haute assumee).
  *Gate* : ratio publie/oracle des 11 films passe de **1,531 a <= 1,05** ; **0 joueur**
  au-dessus de son oracle a 0,5 s pres ; le nombre de periodes ne DIMINUE pas (>= 197 :
  fermer un portage n'en supprime aucun) ; aucun joueur ne perd du temps qu'il avait a
  l'oracle.
  *Recuisson* : **oui**, les 11 films CTF a calque.
  *Gain* : 1 129,3 s de faux retirees — le plus gros du parc. *Effort* : fort.

- [ ] **L6 — Publier le portage de drapeau sur BTB.**
  *Preuve d'entree* : 266,2 s d'oracle, 0 s publiee, 22 joueurs, 2 films.
  *Dependance* : L1, et l'instruction du plafond de 8 slots (hypothese du §10 a fermer
  d'abord).
  *Gate* : `4f77afc1` et `879a4dba` publient un portage dont le ratio a l'oracle est dans
  [0,8 ; 1,05], aucun joueur au-dessus du sien.
  *Recuisson* : **oui**, 2 films. *Effort* : fort.

- [ ] **L7 — Calque de zones sur Total Control.**
  *Preuve d'entree* : `5676a9ba` publie 23 captures de zone mais `zoneStates` vide et
  `coverage.zones` absent, la ou les 5 Strongholds publient 3 zones chacun.
  *Gate* : `5676a9ba` publie >= 3 zones et `coverage.zones` ; captures >= 46/51.
  *Recuisson* : **oui**, 1 film (2 quand `a349fea8` sera cuit). *Effort* : moyen.

- [ ] **L8 — `flag_secures` : instruire l'emplacement.**
  *Preuve d'entree* : 223 actions a l'oracle, 0 publiee, 13 films.
  *Contenu* : balayage des comps non mappes de la table drapeau contre l'oracle par joueur —
  meme protocole que celui qui a nomme `flag_grabs` (`named.go:139-155`).
  *Gate* : le comp candidat reproduit `flag_secures` **joueur par joueur** sur >= 10 films,
  et **aucun** desaccord ; sinon on ne publie rien.
  *Recuisson* : **oui**, 13 films. *Effort* : fort.

- [ ] **L9 — Temps d'occupation de zone par joueur. ESCALADE UTILISATEUR, pas un correctif.**
  *Preuve d'entree* : 3 507,2 s d'oracle, 0 s publiee, 6 films.
  *Pourquoi une escalade* : c'est un calque qui n'existe pas, pas un calque casse. Il demande
  un croisement continu positions x formes de zone sur toute la duree du film (aujourd'hui
  `zone_attribution.go` ne le fait qu'aux instants d'action), et donc une decision de cout.
  *Gate si engage* : ratio publie/oracle dans [0,85 ; 1,00] par joueur sur les 5 Strongholds,
  aucun joueur au-dessus du sien. *Effort* : fort.

- [ ] **L10 — KOTH : catalogue de cartes, pas calque.**
  *Preuve d'entree* : `E4_cuisson_koth.log`, 7 films, `carte hors catalogue ([])`, sortie 10.
  Le calque, lui, reproduit l'API **exactement** sur 31 joueurs / 4 films
  (`hill_hold_ticks.go:8-12`).
  *Gate* : 3 films KOTH du cache cuisent sans `carte hors catalogue` ; un export d'oracle qui
  les contient permet alors la mesure — **elle n'est pas faite ici**. *Effort* : moyen.

---

## 12. Decouvertes hors perimetre (consignees, NON traitees)

1. **`buildObjectivePulses` itere sur TOUTES les actions, `kills` et `assists` compris** —
   `apps/web/src/features/match-replay/layers/objectivesLayer.ts:305-330`. Sur `8bc6074f` et
   `f8efc5ca` cela represente **15 648 et 15 645 pulses** construits par image de scene. Cout
   de rendu non mesure ici, mais le denominateur de couverture, lui, est deja fausse :
   `coverage.objectives.available = 15 808` sur `8bc6074f` fait lire « 100 % de couverture »
   sur un calque dont 99 % du contenu est un seul compteur.
2. **`coverage.flagCarries.closedByObject = 0` sur les 11 films a calque** alors que
   `objectLives` vaut 37 sur `16ea3668` : un canal lu, compte, publie — et jamais consomme.
   C'est le levier de L5, et c'est aussi un compteur de couverture qui n'informe personne.
3. **La justification de `maxUnrollPerStep` est fondee sur une population mal qualifiee** —
   `objectiveevents/named_series.go:41-48` appelle « pire deroulage d'un film SAIN » les
   17 306 evenements du comp 20 B de `d9781168`. L'oracle prouve sur `a0c36016` que ce meme
   comp produit 84 actions pour 0 reelle. Le commentaire est a reecrire dans le lot qui
   touchera la borne (anti-patron n° 9).
4. **`4ecdf3e7` (drapeau neutre) perd un joueur entier** : 2533274877168586, 4,1 s a l'oracle,
   0 publiee, 0 periode — le seul joueur a 0 publie hors BTB. Non instruit.
5. **`zone_offensive_kills` / `zone_defensive_kills`** ne figurent dans aucune table
   d'emplacement (`named.go:103-110`), comme `flag_secures`. Non chiffre : ces colonnes n'ont
   pas ete confrontees dans cet audit.
6. **`config/replay_corpus.toml`, famille `oddball`** : la raison du temoin decrit encore un
   residu ferme depuis le 2026-09-08 (deja consigne au rapport 6.2 §6.6, toujours ouvert).

---

## 13. Ce que cet audit N'A PAS pu prouver, et ce qu'il faudrait

| question | pourquoi non prouvable ici | ce qu'il faudrait |
|---|---|---|
| Reproduire les 82,2 % de couverture Oddball | aucun artefact Oddball au parc | recuire 4 films (`replay-build`, hors perimetre d'audit) |
| Couverture du portage de la BOMBE | 0 artefact d'Assaut, **0 colonne bombe a l'API** | un releve Theater manuel : c'est le seul temoin possible |
| Couverture KOTH (tics de colline vs `zone_scoring_ticks`) | 0 match KOTH a l'oracle, 0 artefact ; la colonne vaut 0 sur les 328 lignes | un export d'oracle contenant des KOTH + le catalogue de cartes (L10) |
| Le plafond de 8 slots cause-t-il les compteurs faux du BTB ? | seuls la constante, l'effectif et le desaccord sont etablis | decoder un film BTB slot par slot |
| Les 40 806 `assists` des 3 films sont-elles fausses ? | l'export n'a pas de colonne `assists` | ajouter `assists` / `kills` a l'export d'oracle — une requete, pas un chantier |
| Le defaut mono-manche coute-t-il quelque chose sur `fb1a1a72` / `64e8adfa` ? | ces deux films n'ont pas d'artefact | les cuire, puis rejouer l'instrument |
| Le multi-manche du drapeau et de la bombe | 2 CTF multi-manche mesures (0 perte), 0 Assaut | plus de films multi-manche au parc |

---

## 14. Reproduction

```bash
# les huit sorties de ce document, d'un seul coup, sans ouvrir aucune base
cd .ai/V7.5/outillage/couverture_objectifs
CGO_ENABLED=0 go run . \
  -parc   <racine>/data/cache/replays/halo_infinite \
  -oracle <racine>/.ai/V7.5/replay2d/registre_film \
  -out    <racine>/.ai/V7.5/replay2d/registre_film
```

Les sorties committees (`vague6_*.tsv|log`) sont exactement celles de cette commande sur le
parc du 2026-09-10 (64 artefacts, schema 51).
