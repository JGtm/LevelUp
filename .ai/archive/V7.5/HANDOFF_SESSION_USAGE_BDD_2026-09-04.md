# HANDOFF — Usages d'equipement, socles et objectifs a l'echelle d'une SESSION

> Redige le 2026-09-04. Chantier ouvert, non demarre sur la voie retenue.
> Branche de depart : `feat/v75`. Worktree existant : `LevelUp-wt-session-usage`
> (branche `wt/session-usage`, un commit de WIP a relire, voir §6).

---

## 1. Ce que l'utilisateur veut, et ce qui est deja tranche

La page **Sessions** (`apps/web/src/features/session-detail/`, route
`stats/sessions`) doit porter, en contexte **Solo** et en contexte **Escouade**
(selecteur `useSessionContextStore.matchContext`), les memes trois familles de
grandeurs que la vue match :

1. **Usages d'equipement** — camouflage, surbouclier, mur de protection, grappin,
   objets laches au sol. **Les grenades en sont EXCLUES** : decision utilisateur du
   2026-09-04, « ce ne sont pas des equipements ». Le champ peut etre produit quand
   meme (il coute trois lignes), il ne s'affiche pas ici.
2. **Controle des armes speciales** — prises de socle.
3. **Objectifs** — par role (prendre / defendre / tenir) et par famille de mode.

### Les formes sont VALIDEES, elles ne sont pas a redessiner

Maquette de reference, seize formes, sur donnees reelles :
**https://claude.ai/code/artifact/2ec1b8eb-5b4d-4484-b632-c8ee91569825**
(planches precedentes, conservees comme catalogue :
`7f1bdb9f-ab26-4c14-a422-3eb4f90bf21f` cadences par 10 min ·
`caa2cd6a-0264-42b5-a0cd-dca8599d3af4` part du lobby).

Doctrine d'affichage arretee avec l'utilisateur :

- **TOUT est NORMALISE, aucune valeur brute.** La page sert a comparer DES SESSIONS
  entre elles : cadences **par dix minutes de jeu**, ou **parts en pourcentage**.
  Une grille « valeurs brutes » a ete explicitement ecartee pour ce motif. Les
  comptes bruts ne subsistent qu'en TEXTE a cote d'un taux, comme denominateur
  d'honnetete — jamais comme axe de comparaison.
- **La reference est l'equipe d'en face, et elle n'est JAMAIS affichee** : ni ligne,
  ni nom, ni couleur. Elle est le denominateur, et le trait de **parite** (colore,
  jeton distinct — jamais une teinte de donnee).
- **DEUX denominateurs partout** : part de **mon equipe** (parite = 100/effectif
  d'equipe) et part du **lobby** (parite = 100/effectif du lobby). Les deux ensemble
  repondent a « ai-je ete un poids mort » — etre a la parite de son equipe pendant
  que l'equipe est sous celle du lobby n'est pas un defaut du joueur.
- **Axe des valeurs gradue sous chaque forme**, avec son intitule, comme partout
  ailleurs dans l'app.
- Trois formes composent la grammaire : **ecart a la parite** · **jauge double avec
  etendue** · **piste du lobby** (colore = nous, decoupe par joueur ; hachure = eux).
  Plus la **grille alignee** (une echelle et un axe par colonne) et la **bande de
  regularite** (une case par match, teintee par l'ecart).

### La decision d'architecture, changee le 2026-09-04

Une premiere voie — un **resume sidecar** `{short8}.usage.json` a cote de l'artefact
— a ete lancee puis **ABANDONNEE sur decision de l'utilisateur** : « il faut les
sauvegarder en BDD lors du sync ».

**Voie retenue : persistance en base, ecrite AU SYNC.**

---

## 2. Pourquoi il faut persister — la mesure qui l'impose

Les grandeurs d'equipement et de socle n'existent aujourd'hui **que dans les
artefacts de rejeu** (`data/cache/replays/{slug}/{short8}.json`).

| Mesure | Valeur (2026-09-04) |
|---|---|
| Artefacts en cache | 114 |
| Poids total | 208 Mo |
| Poids moyen d'un artefact | **1,8 Mo** |
| Matchs d'une session typique | 9 |

Ouvrir neuf artefacts a chaque chargement de la page Sessions represente ~16 Mo de
JSON a analyser par requete. Exclu. D'ou la persistance d'un **resume par match**,
quelques centaines d'octets par joueur.

Les **objectifs**, eux, sont deja en base (`match_objective_stats_latest`, les deux
camps compris) : **rien a produire pour le bloc 3**, seulement a agreger.

---

## 3. Ce que le resume doit porter

Par **(match_id, xuid)** :

| Champ | Source dans l'artefact |
|---|---|
| `grapple_pulls` | `grappleLines[].slot` -> proprietaire par `tracks[].slot` |
| `camo_episodes`, `camo_ms`, `camo_kills` | `equipmentEpisodes[fam="camo"]` |
| `overshield_episodes`, `overshield_ms`, `overshield_kills` | idem `fam="overshield"` |
| `deployed_<famille>` | `equipmentPlacements` avec `origin != "dropped"`, familles deployables |
| `dropped_objects` | `equipmentPlacements` avec `origin == "dropped"`, **hors familles de grenade** |
| `grenades_thrown` | `grenades[].i` (index joueur du film) x `grenades[].rank` — produit, non affiche |
| `pad_pickups` | `padPickups[].xuid`, **socles de bonus EXCLUS** (cf. §4) |
| `pad_pickups_by_weapon` | idem, ventile par identifiant d'arme |

Au niveau du **match** :
`frame_interval_ms`, `frame_count`, `duration_ms`, `pad_occupancies`, `pad_named`,
`pad_unnamed`, `powerup_pad_pickups` (par famille de bonus), et la liste des socles
d'arme presents sur la carte avec leurs occupations.

**L'effectif de chaque camp doit etre derivable** : la part d'equipe et la parite en
dependent. Il vient de `match_participants` (deja en base) — ne pas le dupliquer.

---

## 4. LE PIEGE, verifie sur pieces — socle d'ARME contre socle de BONUS

**`weaponPads[].weapon` melange armes et bonus.** Releve sur 40 artefacts :

- **26 armes reelles**
- **`powerup_camo`** : 17 occurrences
- **`powerup_overshield`** : 15 occurrences
- un identifiant non catalogue (`0xD7915565`)

Aucun equipement DEPLOYABLE (grappin, mur, capteur) n'apparait en socle dans ce
corpus.

Le bloc « Controle des armes speciales » de la vue match **exclut deja** les socles
de bonus : ils tombent dans sa note de pied (`REPLAY_TEXT.padControl.gapFmt.powerup`
— « jamais rattachable : un bonus s'identifie par un nom, pas par une famille
d'arme »), tandis que le bloc equipement les compte a part sur sa ligne « Socles de
bonus de puissance vides », **anonyme**.

**Reproduire exactement cette frontiere.** `pad_pickups` ne compte que les socles
d'ARME ; les bonus vont dans `powerup_pad_pickups`. La regle vit cote web dans
`features/match-replay/padControlLogic.ts` et `weaponPadFamilies.ts` ; cote Go,
chercher l'equivalent (`internal/analysis/replay/document_pickups.go`) **avant** d'en
ecrire un second.

**Corollaire a dire a l'ecran** : camouflage et surbouclier apparaissent sous deux
formes distinctes — **episode actif** (le film mesure que l'effet court) et **socle
de bonus vide** (le socle se vide). Meme objet vu par deux bouts, jamais additionnes.

---

## 5. Le decoupage propose — trois lots

### S1 — Le socle de donnee (bloquant, a faire en premier)

1. **Table append-only** dans `shared_matches_v2.duckdb`, recette **ADR 0026** :
   `id` PK + `written_at`, **vue `<table>_latest` obligatoire** (les lecteurs ne
   lisent QUE la vue — une lecture brute sert des lignes perimees). Recette d'ajout :
   ADR 0026 + `internal/migration/append_only_rebuild.go`.
2. **Ecriture INSERT-only via `persist`** — ADR 0019 / 0030 : passer par
   `internal/persist/BatchBuilder.Submit()` -> `persist.*Persister.Persist()`.
   **Aucun UPSERT, aucun `ON CONFLICT DO UPDATE`.** Le garde-rail
   `internal/sync/no_art_patterns_test.go` refusera tout le reste, et il ne faut
   **jamais** l'allowlister ici.
3. **Branchement AU SYNC** : l'etape post-sync qui cuit les artefacts est dans
   `internal/sync/engine_postsync.go` (voir ~ligne 286, construction `replaybuild`
   des matchs inseres) et `internal/sync/engine.go` (~ligne 113, l'etape est nil quand
   le wiring ne l'installe pas — elle n'est posee qu'en LOCAL). Le resume se derive de
   l'artefact **qui vient d'etre ecrit** : meme etape, pas de second decodage de film.
4. **CLI de backfill** pour les 114 artefacts deja cuits : lit **un artefact a la
   fois**, jamais de map globale vivante (lecon du chantier `backfill-replay`, qui a
   sature la machine le 2026-08-20), reprenable, `--force`, `--match`.
5. **Title-agnostic** : branchement sur une **capability**, jamais sur `slug == "..."`
   (ratchet `no_slug_comparison_test.go`). Un titre sans film ne produit rien,
   proprement.
6. **Tests** : la projection pure sur une fixture d'artefact (il en existe deja —
   chercher `match-replay/test/` cote web et les fixtures Go **avant** d'en fabriquer,
   cf. [[reference-test-ddl-copies-derivent]] : jamais de DDL recopiee, les fixtures
   passent par les migrations reelles), le piege socle-bonus couvert par une assertion
   nominative, et le test d'integration anti-ART (`go test -tags=integration ./...`,
   OBLIGATOIRE avant livraison sur tout ce qui touche persist/sync).

**Controles croises a exiger dans le rapport**, sur les artefacts reels :

| Match | Attendu |
|---|---|
| `696a9d7c` | **26 prises de socle nommees, 8 anonymes** ; les 10 occupations `powerup_camo` en `powerup_pad_pickups`, **zero** dans `pad_pickups` |
| `b8a44fe8` | **51 prises nommees, 11 anonymes** |
| Session 2026-07-31 (9 matchs, 8 avec film) | **193 prises nommees, 102 anonymes** au total |

### S2 — L'agregat de session et le contrat

Le service agrege sur les matchs de la session (le decoupage existe :
`analysis.ComputeSessions`, seuil `DefaultSessionGapMinutes = 120`) et publie, par
grandeur :

- la somme **du joueur**, de **son camp**, du **lobby** ;
- les **deux effectifs moyens** (equipe, lobby) — d'ou les deux parites ;
- l'**etendue** match par match et le **compte de matchs au-dessus de la parite** ;
- la **cadence par dix minutes** (denominateur : la duree jouee des matchs MESURES,
  pas de la session entiere).

**Ne publier que du normalise** cote contrat, conformement a la doctrine du §1.
Prevoir le champ « matchs mesures / matchs de la session » : la couverture des films
n'est jamais totale (8 sur 9 sur le temoin) et l'ecran doit le dire.

### S3 — Le front

Les seize formes de la maquette, dans `features/session-detail/`, contextes Solo et
Escouade. La grille alignee existe deja : **`components/charts/ValueGrid`**
(+ `valueGridModel`), livree le 2026-09-03 pour la vue match — la reutiliser, ne pas
en poser une seconde. Couleurs d'equipe : **jetons `team-ally` / `team-enemy`**
surchargeables par les reglages d'accessibilite, jamais `teamColorResolver` (cf.
[[reference-team-color-identite-vs-graphe]]). Couleurs d'escouade : `squad-player-1..3`.

---

## 6. Ce qui existe deja

- **`wt/session-usage`, commit `b8dc38107`** : un WIP de la voie sidecar, interrompu.
  Seule la sortie de la regle « socle d'arme contre socle de bonus » hors du decodeur
  y est reutilisable. **Le reste est a jeter** — il ecrivait un fichier a cote de
  l'artefact, ce que la decision du 2026-09-04 annule.
- **Vue match livree et mergee** (`feat/v75`, merge `2577e57a5`) : les trois blocs y
  sont deja des graphes, avec `ValueGrid`, `SectionCard`, et les jetons d'equipe
  surchargeables. C'est le modele a suivre pour S3.
- **Temoin de travail** : session du **2026-07-31, 19:22 -> 21:06**, joueur `JGtm`,
  escouade `JGtm + Madina97294 + Chocoboflor` (les trois sur les neuf matchs),
  9 matchs dont 8 avec film, trois familles de mode (6 Assassin, 2 Drapeau,
  1 Bastion), objectifs sur 3 matchs avec **deux jeux de colonnes differents**.
  C'est le meilleur cas de test du depot : il porte les quatre difficultes d'un coup.

---

## 7. Chiffres de reference du temoin, pour verifier l'agregat

Session 2026-07-31, 8 matchs mesures, 4 644 s de jeu, lobbies de 8,4 joueurs en
moyenne (parite joueur **11,9 %**, parite d'equipe **24,2 %**).

| Grandeur | Mon camp / lobby | JGtm / son equipe | JGtm / lobby |
|---|---|---|---|
| Prises de socle | 45,6 % | 20,5 % | 9,3 % |
| Camouflages | 50,0 % | 14,8 % | 7,4 % |
| Murs de protection | 33,3 % | 42,9 % | 14,3 % |
| Surboucliers | 45,5 % | 20,0 % | 9,1 % |
| Tractions de grappin | 44,4 % | 50,0 % | 22,2 % |
| Objets laches au sol | 49,1 % | 24,6 % | 12,1 % |
| Objectif — prendre | 56,5 % | 11,5 % | 6,5 % |
| Objectif — defendre | 56,8 % | 38,1 % | 21,6 % |
| Objectif — tenir | 59,7 % | 15,5 % | 9,2 % |

Ventilation des socles par famille d'arme (prises nommees) : lourdes 39 dont 18 au
camp et 8 a JGtm ; precision 44 dont 18 et 5 ; autres 110 dont 52 et 5.

---

## 8. Interdits

- Aucun UPSERT sur la table nouvelle, aucune allowlist du garde-rail anti-ART.
- Aucun re-decodage de film : le resume se derive de l'artefact deja cuit.
- Aucune modification du format de l'artefact lui-meme.
- Aucune valeur brute publiee comme axe de comparaison sur la page Sessions.
- Aucun `slug == "..."`, aucun libelle FR/EN en dur cote Go.
- Toute string UI nouvelle est **bilingue FR et EN**, parite par typage.
- Cuisson d'artefacts en lot : **DEMANDER avant** (bombe RAM x4, verrou
  `filmproc.AcquireSolo`). Ce chantier n'en a pas besoin — il lit des artefacts
  existants.
