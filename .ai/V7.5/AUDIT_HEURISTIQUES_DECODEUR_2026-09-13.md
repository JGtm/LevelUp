# AUDIT — Les heuristiques qui decident a la place de la grammaire (lot 0.E, D13 / D14)

> Date : 2026-09-13. Lot 0.E de `.ai/PLAN_DECODEUR_FILM_2026-09-13.md`. Skill `adversarial-audit` :
> perimetre x axe, registre date, **l'audit ne corrige rien**. Worktree `LevelUp-wt-decfilm-0E`,
> branche `feat/decfilm-0E`. Aucun fichier de production modifie, aucun decodage de film, aucune
> base ouverte. Chemins relatifs a `apps/go-api/`.

## 1. Cadrage

**Perimetre** (code de production, `*_test.go` exclus) :
`internal/games/halo_infinite/film/replay/` (138 fichiers, 34 569 lignes) ·
`internal/games/halo_infinite/film/killsource/` (20 / 4 718) ·
`internal/analysis/objectiveevents/` (18 / 4 556) ·
`internal/replaybuild/` (15 / 2 746) · `internal/sync/killcollector/` (17 / 3 966).
Plus, par exception nommee au brief, les **fonctions d'INFERENCE de `filmdec` qui decident en
production** (`DetectI0Layout`, `frame_chain_infer`, calibrations `axisW` / `indexW`,
`bipedSlotBand`, `CalibrateMPPWidths`, `warnUnknownRegistry`). La GRAMMAIRE de `filmdec`
(largeurs de composants, deserialiseurs, etats par defaut) est HORS AXE.

**Axe unique** : toute decision de PRODUCTION prise par HEURISTIQUE la ou le film ECRIT le fait.

- Heuristique = fenetre temporelle, seuil de distance, majorite, calibration statistique,
  inference par le fil des morts, « le plus proche », « le premier vu », profil plat par defaut.
- Le film ECRIT = evenement nomme du canal d'evenements, record de creation (NEW), composant
  d'etat d'image-cle ou de delta, table de `chunk_00` (identite, table des joueurs), pied de film
  (recompenses, kill feed, evenements de mode).
- Une DECISION DE FAIT compte (origine d'une pose, equipe d'un portage, index de joueur d'un slot,
  manche d'un slot statborg, attribution d'un kill). Un parametre de RENDU ou de PERF ne compte
  pas (lissage, pas d'echantillonnage, budget, taille de tampon).

**Amendement D14** (pilote, 2026-09-13) : chaque ligne porte en outre le repli associe — NOMME ou
ANONYME, sa condition de declenchement telle qu'elle DEVRAIT etre, le risque de declenchement
alors que la lecture existe, et son critere de retrait mesurable. La table (E) recense les replis
ANONYMES (decision de secours ni nommee ni comptee), qui alimentent le lot 1.9.0.

## 2. Methode

Fan-out de six auditeurs en lecture seule, un par famille de fichiers, chacun ignorant les autres :
equipement / armes au sol / capacites · objectifs · vies-identites-vehicules ·
killsource + killcollector · objectiveevents + replaybuild · inferences `filmdec`. Chacun a recu
l'axe, les faits de reference etablis (rapport F.0, rapport du lot H, handoff, ADR 0034), les regles
de recevabilite et l'amendement D14. **Le grep ne prouve rien : chaque site retenu a ete OUVERT et
LU** avec sa fonction entiere et son appelant. Les constats les plus porteurs ont ensuite ete
**re-verifies sur pieces par l'auditeur principal** — section 9.

**Sites de decision LUS : 528** (equipement 130 · objectifs 91 · vies-identites 41 ·
killsource+killcollector 134 · objectiveevents+replaybuild 61 · inferences `filmdec` 77 corps de
fonction, dont 24 remontees d'appelants resolvant 68 sites). **Fichiers de production ouverts : 201**
(les sous-ensembles de `replay/` se recoupent ; 36 + 36 + 60 + 36 + 33 avec recouvrement).

### Commandes de recensement, rejouables

```bash
cd apps/go-api
P="internal/games/halo_infinite/film/replay internal/games/halo_infinite/film/killsource \
   internal/analysis/objectiveevents internal/replaybuild internal/sync/killcollector"

# 1. le balayage large des motifs (509 hits avec filmdec, 321 sans)
grep -rnE --include='*.go' \
  '(Window|Fenetre|Tolerance|Seuil|Threshold|Nearest|Closest|Majorit|Infer|Guess|Devin|Calibr|Heuristi|Repli|Fallback|MaxDist|MinDist|Epsilon|Eps[A-Z]|Detect|Probe|Auto)' \
  $P internal/games/halo_infinite/film/filmdec | grep -v '_test\.go:'

# 2. les constantes de seuil : la liste courte qui cadre la lecture (27 constantes)
grep -rnE --include='*.go' \
  '(const|var)[[:space:]]+[a-zA-Z_]*([Ww]indow|[Tt]olerance|[Ss]euil|[Tt]hreshold|[Mm]axDist|[Mm]inDist|Gap|Eps)[a-zA-Z_]*[[:space:]]*=' \
  $P | grep -v '_test\.go:'
```

Le script de controle « chaque `fichier:ligne` du registre existe » est en section 10.

## 3. Comment lire le registre

Le registre EST le classement : chaque ligne vit dans la table (A), (B), (C) ou « non etabli » qui
la caracterise, et porte les dix champs demandes. La dupliquer une fois a plat et une fois classee
n'aurait rien ajoute. Champs d'une ligne :

`fichier:ligne` · **fait decide** · heuristique et parametres exacts · ce que le film ecrit a la
place + etat du lecteur (PORTE / A PORTER / RIEN) · preuve, negatif mesure, ou « non etabli » ·
cout S / M / L · gain chiffre quand une source le dit · **[D14-1] repli NOMME ou ANONYME** ·
**[D14-2] condition de declenchement souhaitable + risque de declenchement alors que la lecture
existe (oui / non + `fichier:ligne`)** · **[D14-3] critere de retrait mesurable + compteur existant**.

Les lignes regroupent par FAIT decide, pas par fichier : sept sites qui decident le meme fait sont
UNE ligne a sept sites, parce qu'ils se convertissent ensemble.

---

## 4. TABLE (A) — le film l'ecrit ET un lecteur de production existe : conversion courte (famille 1.9)

### A1 — Le decoupage d'i0 auto-detecte alors que le catalogue de carte le donne

- **Sites** : `internal/sync/killcollector/positions.go:253` (`DefaultScanFilmOptions()`, `Layout`
  laisse nil), `internal/sync/killcollector/hits.go:151` et `:157`. Detection :
  `internal/games/halo_infinite/film/filmdec/i0_layout.go:157`.
- **Fait decide** : `GateBits`, les trois largeurs d'axe et l'index de REGION — donc quels
  enregistrements de position sont acceptes, et dans quelle AABB ils sont dequantifies.
- **Heuristique** : profil de taux de bascule sur fenetre de 72 bits, 6 chunks au plus, frontiere a
  saturation >= 0,10 et effondrement >= x8 ; **`GateBits` force a 5** et `Region` jamais renseignee
  (`filmdec/i0_layout.go:191`).
- **Ce que le jeu ecrit** : `MapQuantEntry.Layout()` (`filmdec/map_bounds.go:69`), derive des bornes
  du `.module` par la loi moteur. **PORTE** — et deja impose sur l'autre chemin :
  `replay/build_from_film.go:87` (`NewFilmContextForMap`). Catalogue commis : 79 entrees, 0 sans
  `axisWidths`.
- **Preuve** : `filmdec/film_context.go:33-42` — sur Live Fire l'auto-detection rend
  `gate=5 region=0 13/12/11` contre `gate=6 region=1 12/12/11` au catalogue ; meme longueur totale
  d'i0, mais la porte de region ne teste qu'UN bit : **27 enregistrements sur 267 400 appartenant a
  une AUTRE region passent** (mesure du 2026-09-03 sur `60ae07c4`). Sur les 12 autres films du
  corpus d'equivalence, catalogue et detection concordent au bit pres.
- **Cout** S. **Gain** : 27 / 267 400 faux enregistrements elimines sur Live Fire, et deux passes de
  detection supprimees par film (6 chunks bit a bit, plus la bande propre de la detection).
- **[D14-1]** Repli **NOMME** deux fois : `bipedI0Layout` (`filmdec/offline_biped_band.go:120`,
  « celui que l'appelant force, sinon celui lu dans le film ») et `resolveI0Layout`
  (`filmdec/film_context.go:138`).
- **[D14-2]** Devrait etre : « la carte de ce match n'est pas au catalogue, ou son entree est
  anterieure au champ `axisWidths` ». **Risque OUI** : `filmdec/offline_biped_band.go:121`
  (`if opt.Layout != nil`) — l'appelant tient `entry` en parametre et en lit `Range()` a la ligne
  suivante (`positions.go:254`) ; il ne passe simplement pas `Layout`.
- **[D14-3]** 0 appel a `DetectI0LayoutOf` depuis un chemin de production sur le corpus gate.
  **Pas de compteur** : le `I0LayoutReport` est jete (`_`) aux trois sites d'appel.

### A2 — La carte du film devinee par signature de largeurs, alors que le collecteur tient son nom

- **Site** : `internal/sync/killcollector/hits.go:151` (`DetectFilmWorldRange(dir, path, "")`) ;
  detection `internal/games/halo_infinite/film/filmdec/weapon_hit_distance_resolver.go:144`.
- **Fait decide** : les bornes de dequantification employees pour la distance de chaque touche
  (`match_weapon_hit_distance`).
- **Heuristique** : `DetectI0Layout` puis balayage lineaire du catalogue, accepte **ssi exactement
  un** candidat partage les largeurs d'axe.
- **Ce que le jeu ecrit** : le nom de carte du match, `c.mapNames.MapKeysForMatch` — **PORTE**,
  employe par le MEME collecteur a `internal/sync/killcollector/positions.go:210` ; le parametre
  `mapNameOverride` existe deja et est passe vide.
- **Preuve** : `RAPPORT_F0_DEPLOIEMENT_103_2026-09-13` §0.3 et §6 reserve 2 — jumelles d'echelle
  mesurees Behemoth/Fragmentation, Catalyst/Deadlock, Prism/Scarr ; `fccc61cd` rend **3 candidats a
  egalite**. Sur ces cartes `len(hits) != 1` et les distances sont desactivees **en silence**.
- **Cout** S. **Gain** : 3 paires de jumelles (6 cartes) recuperent leurs distances ; le nombre de
  matchs concernes n'est chiffre par aucune source.
- **[D14-1]** MIXTE : `mapNameOverride` est nomme et documente ; **la decision de le passer vide est
  ANONYME** (`hits.go:151`). La degradation, elle, est nommee (`hits.go:152-155`).
- **[D14-2]** Devrait etre : « ce match n'a aucune identite de carte connue ». **Risque OUI** :
  `internal/sync/killcollector/hits.go:151`.
- **[D14-3]** 0 match a distances desactivees sur le corpus gate. **Pas de compteur** — l'echec est
  un `slog.DebugContext`, invisible au niveau de log de prod ; `metricHitsScanFail` ne couvre pas
  cette branche.

### A3 — Le couple (tueur, victime) recolle sur le voisin, alors que le kill-event porte les deux

- **Site** : `internal/games/halo_infinite/film/killsource/feed.go:162` (`reconstructPairs`, jumeau
  `:203`).
- **Fait decide** : a qui est attribue un frag quand le kill-feed ne porte pas les deux champs au
  meme instant.
- **Heuristique** : recollage sur un voisin immediat, fenetre de **2 instants**.
- **Ce que le film ecrit** : le KILL-EVENT (code 85) porte victime ET tueur dans le meme
  enregistrement — grammaire `victime(E5) tueur(E5) [% TUEUR] R1 assistant(E5) [% ASSISTANT]`,
  `killsource/eventchain.go:242-256`. **PORTE** — mais lu uniquement pour l'ASSISTANT
  (`killsource/assist.go:230`), jamais pour le couple.
- **Preuve** : `killsource/feed.go:149-151` — les couples recolles echouent MOINS que les autres
  (5/64 = 7,8 % contre 42/372 = 11,3 %, p = 0,886), « mais il FABRIQUE un couple quand la vraie
  victime est un bot ».
- **Cout** M. **Gain** : 64 couples recolles sur 372 dans la serie de reference cessent d'etre une
  reconstruction.
- **[D14-1]** Repli **NOMME** : `kf.fab`, « couples RECOLLES sur le voisin » (`feed.go:49`).
- **[D14-2]** Devrait etre : « le film ne porte aucun kill-event pour cet instant ». **Risque OUI** :
  `killsource/feed.go:162` recolle des que le champ manque, sans consulter le kill-event du meme
  paquet.
- **[D14-3]** `len(kf.fab) == 0` sur le corpus gate. **Compteur derivable** :
  `Coverage.ReconstructedPairs` moins `Coverage.SameInstantPairs` (`killsource/decode.go:182`, `:184`).

### A4 — Le porteur du crane infere des tics de score, alors qu'il voyage dans le canal des armes tenues

- **Site** : `internal/games/halo_infinite/film/replay/skull_carries.go:390`
  (`skullCarryIntervals`) ; fermeture de periode a `skullTickGapMS = 3000` (`skull_carries.go:47`).
- **Fait decide** : QUI porte le crane et de quand a quand (`skullCarries`,
  `time_as_skull_carrier_seconds`).
- **Heuristique** : le porteur est celui dont le score de mode monte ; un trou > 3 s ferme la
  periode.
- **Ce que le film ecrit** : le crane voyage dans le canal des ARMES TENUES, famille `0x0017592c`
  (`replay/held_object_carry.go:15`, deja au manifeste `replay_labels.toml`). Lecteur
  `filmdec.ScanFilmHeldWeaponChanges` + `replay.BuildHeldObjectCarry` : **PORTE** — et
  **cable pour la BOMBE seulement** : `BuildHeldObjectCarry` n'a qu'UN appelant de production,
  `replay/bomb_carries.go:142` (verifie sur pieces, section 9).
- **Preuve** : le negatif D4 souvent cite porte sur les TROUS DES VIES LIBRES
  (`replay/document_objective_objects.go:14-19`), **pas** sur le canal des armes tenues. Pour ce
  canal applique au crane : **non etabli** — et c'est pourquoi la conversion est un lot court sous
  gate, pas une certitude.
- **Cout** S (le meme cablage que `bomb_carries.go`). **Gain** : non chiffre.
- **[D14-1]** Pas de repli aujourd'hui : la voie des tics est la voie unique.
- **[D14-2]** Sans objet en l'etat. Apres conversion, le repli devrait etre « le canal des armes
  tenues n'emet rien pour ce film », et les tics deviennent ce repli, compte.
- **[D14-3]** Accord 100 % entre les periodes des tics et celles du canal des armes tenues sur le
  corpus gate. Compteurs existants : `Trains`, `NoBridge`, `CarrierAbsent`.

### A5 — Le drapeau qui rentre choisi par « le seul au sol », alors qu'il est deja nomme en amont

- **Site** : `internal/games/halo_infinite/film/replay/flag_carries_lives.go:267` et `:278`
  (`applyFlagReturn`).
- **Fait decide** : QUEL drapeau rentre a sa base sur un `flag_returns` — donc la fin d'un portage
  et l'etat publie du drapeau.
- **Heuristique** : l'UNIQUE drapeau au sol ; zero ou deux candidats -> abstention comptee.
- **Deja disponible dans le code** : `ev.flag`, pose en amont par l'equipe du rendeur
  (`replay/flag_carries_home.go:82-98`), voyage jusqu'ici et ne sert qu'a un court-circuit
  (`flag_carries_lives.go:264`) — jamais a poser le drapeau. **PORTE, non utilise.**
- **Preuve** : le negatif ecrit (« l'evenement ne porte ni l'objet ni l'equipe ») est vrai du
  STATBORG seul ; il ne couvre pas `ev.flag`.
- **Cout** S. **Gain** : les `AmbiguousReturns` que `ev.flag` tranche ; valeur sur le parc non citee.
- **[D14-1]** Repli **ANONYME** — une boucle de recherche du seul au sol, sans nom ni constante.
- **[D14-2]** Devrait etre : « `ev.flag` est indetermine ». **Risque OUI** :
  `replay/flag_carries_lives.go:267` s'execute meme quand `ev.flag >= 0`.
- **[D14-3]** `ambiguousReturns == 0` sur le corpus gate. **Compteur existant.**

### A6 — Le dead-state apparie au kill-feed par une fenetre de 2,5 s, alors que les deux portent le meme paquet

- **Sites** : `internal/games/halo_infinite/film/killsource/options.go:114` (`tolMS = 2500`),
  employe a `match.go:30`, `:45`, `:103`, `:143`, `bijection.go:44`, `hybrid.go:278`, `assist.go:362`.
- **Fait decide** : quel dead-state correspond a quelle mort du feed — donc l'arme de chaque kill,
  et l'assistant.
- **Heuristique** : demi-fenetre temporelle de 2 500 ms plus egalite de NOMS (donc dependance a la
  bijection, cf. B2).
- **Ce que le film ecrit** : le kill-event et le dead-state d'une meme mort vivent dans les paquets
  A EVENTS et portent tous deux l'identite de paquet `(chunk, pidx)` (`killsource/scan.go:49`,
  `killsource/assist.go:169`). **PORTE, non utilise** — `Kill` ne transporte pas `(chunk, pidx)`
  (`killsource/match.go:186-193`), donc le lien est structurellement inaccessible en aval.
- **Preuve** : la valeur 2500 est **non etablie** — `options.go:112-113` la justifie par la
  comparabilite (« la changer invaliderait les mesures publiees »), pas par une mesure. Ce qui EST
  mesure, c'est le controle de hasard (horloge decalee de +-20 s et +-60 s, `match.go:19-21`).
- **Cout** M. **Gain** : non chiffre.
- **[D14-1]** **NOMME** (constante `tolMS`) mais presente comme le mecanisme, pas comme un substitut.
- **[D14-2]** Devrait etre : « les deux structures ne partagent pas d'ancrage commun ».
  **Risque OUI, systematique** : `killsource/match.go:30`.
- **[D14-3]** Appariement par identite de paquet donnant le meme resultat sur 100 % du corpus.
  **Compteurs existants** : `PathStats.Population/Matched` par voie, `Stats.MultiCandidate`
  (`killsource/hybrid.go:350`) = 0 sur la serie de reference.

### A7 — Le chunk du pied de film choisi par argmax de kills, alors que le manifeste declare son type

- **Site** : `internal/games/halo_infinite/film/killsource/feed.go:82` (`if nk > bestN`).
- **Fait decide** : quel chunk est le kill-feed — donc toute la verite credit, la bijection, et tout
  ce qui en depend.
- **Heuristique** : on analyse TOUS les chunks et on garde celui qui produit le plus d'evenements
  `kill`.
- **Ce qui l'ecrit** : le MANIFESTE du film declare le type de chunk (type 3 = pied,
  `killcollector/bridge.go:13-16`), transporte jusqu'a `filmsource.Film.Meta()`
  (`internal/analysis/filmsource/film.go:41`) et deja lu par ce patron dans le depot :
  `internal/analysis/objectiveevents/extract.go:144-155` (`footerData`). **PORTE, non consomme** :
  `killsource.loadFilm` ne copie que `src.Chunk(ch)` et perd `Meta()` (`killsource/chunks.go:78-99`).
- **RESERVE, dite** : le manifeste n'est PAS les octets du film — c'est un descripteur externe
  (`data/cache/film_manifests`), et son absence est un cas reel (plan §2.2). La conversion doit donc
  garder l'argmax en **repli compte**, pas le supprimer. Que `chunk_00` porte lui-meme le type de
  chaque chunk est **non etabli** (question NE1).
- **Preuve** : aucun defaut mesure de l'argmax a ce jour. C'est une conversion de ROBUSTESSE.
- **Cout** S. **Gain** : non chiffre.
- **[D14-1]** **ANONYME** : rien ne presente l'argmax comme un substitut au type de manifeste ;
  `killsource/feed.go:58-59` le presente comme la methode.
- **[D14-2]** Devrait etre : « le manifeste ne declare aucun chunk de type 3 ». **Risque OUI,
  systematique** : `killsource/feed.go:71` boucle sans jamais consulter un type.
- **[D14-3]** Le chunk retenu par argmax est celui du type 3 sur 100 % du corpus. **Pas de compteur**
  — seul l'echec total est visible (`ErrNoKillFeed`).

### A8 — La troncature des trajectoires de projectile : un garde-fou qui compense une faute de decodage

- **Site** : `internal/games/halo_infinite/film/replay/projectiles.go:106`
  (`projectileMaxStepM = 10`, declaree `projectiles.go:58`).
- **Fait decide** : le vol publie s'arrete ici — la trajectoire dessinee d'une grenade ou d'une
  roquette.
- **Heuristique** : tout pas > 10 m entre deux points de grille coupe le vol, sans recoudre.
- **Ce que le film ecrit** : les positions. **Le defaut est en AMONT**, dans la dequantification de
  `filmdec` — caracterise dans `.ai/RAPPORT_LOT_B_DECODEUR_FORK_2026-09-11.md`.
- **Preuve** : 76 artefacts du parc (schema 51) — **947 trajectoires sur 15 735 = 6,0 %** portent au
  moins un pas impossible, soit 4 901 pas ; signature d'UN BIT qui bascule (sur Live Fire, saut =
  exactement la moitie de l'etendue Y, 31,89 m pour 63,775 m, couvrant 3 907 des 4 901 pas).
- **Cout** M. **Gain** : **6,0 % des trajectoires du parc** retrouvent leur fin reelle.
- **CE N'EST PAS UNE CONVERSION 1.9** : rien a lire a la place, une CAUSE a corriger dans `filmdec`.
  Consigne ici parce que c'est le cas le plus net de la doctrine D13 « corriger la cause, pas le
  symptome », et parce que le garde-fou devient supprimable apres.
- **[D14-1]** **NOMME ET COMPTE** — troisieme valeur de retour de `buildProjectiles` (`tronquees`),
  remontee a la couverture du document. Modele de conformite.
- **[D14-2]** Risque de declenchement a tort : **NON** (10 m par pas = 100 m/s, au-dessus de tout
  projectile du jeu : grenade < 20 m/s, roquette < 30 m/s).
- **[D14-3]** Compteur de trajectoires coupees a 0 sur le corpus gate apres correction de `filmdec`
  -> suppression du garde-fou. **Compteur deja en place.**

---

## 5. TABLE (B) — le film l'ecrit, le lecteur est A PORTER : bloquant nomme

Aucune ligne de cette table n'appartient au lot 3.6 par defaut : la majorite a deja son lot au plan
(1.4, 1.5, 1.6, 1.7, 1.8, 3.2, 3.4, 3.5). Le lot 3.6 (ports de composants d'image-cle) ne recoit que
B6 et B7.

### B1 — L'EQUIPE d'un joueur : 4 bits lus et JETES, et cinq commentaires qui disent l'inverse

- **Bloquant nomme** : `internal/games/halo_infinite/film/filmdec/traverse.go:515-517` —
  `case "managed-player-team-designator-component": br.ReadBits(4); return variant, nil, true`.
  La grammaire est PORTEE (largeur juste, ti=9 i0, valeur = designateur + 1) ; **seule la
  publication manque**. Verifie sur pieces (section 9).
- **Fait decide, en NEUF endroits** : `Track.Team` publie en dur a -1
  (`replay/build.go:580`) · qui tient une zone (`replay/zone_states_owner.go:293` et `:364`) · qui
  tient la colline (`replay/zone_states_hill.go:150`) · l'identite des deux courbes de score
  (`replay/score_team_identity.go:42`) · a quel drapeau appartient un portage
  (`replay/flag_assign.go:279`) · le camp d'un `bomb_carriers_killed`
  (`replay/bomb_stats.go:445`) · les coequipiers du calque d'isolement
  (`replay/death_context.go:171`) · la table d'equipe de la cuisson
  (`replaybuild/matchfacts.go:264` et `:492`) · l'equipe d'un evenement d'objectif
  (`objectiveevents/extract.go:184` et `:250`).
- **Heuristiques employees a la place** : valeur plate `-1` ; table `xuid -> equipe` prise dans
  `match_participants` ; et, pour la propriete d'une zone, une **CALIBRATION STATISTIQUE SUR UN
  ORACLE EXTERIEUR AU FILM** — `replay/zone_states_owner.go:353` (`ownerScores`) elit le canal de
  propriete en maximisant l'accord avec l'equipe du capteur **lue dans la base**, seuil
  `zoneOwnerMinAgreements = 2`. Le code l'ecrit lui-meme : « L'ORACLE EST EXTERIEUR AU FILM [...]
  `coverage.zones.ownerAgreed` se lit comme la qualite du MEILLEUR candidat, pas comme une preuve
  independante » (`zone_states_owner.go:346-349`, verifie sur pieces).
- **Preuve** : `docs/adr/0034-film-decoder-profile-and-layers.md` D-9 (le film est la SEULE source,
  la base n'est qu'un compteur de controle) ; handoff §2 point 4 : 16/18 puis 9/10 films en accord
  avec la base, six Grande bataille a 24/24, FFA a 0, jamais plus de deux designateurs sur
  595 matchs.
- **DOC INVERSEE, a corriger dans le meme lot** (anti-patron n° 9 de CLAUDE.md) :
  `replay/document.go:590` (« Team vaut -1 : L'EQUIPE N'EST PAS DANS LE FILM ») ·
  `replay/death_context.go:107` (« JAMAIS depuis le film, qui ne porte aucun camp ») ·
  `replay/flag_assign.go:12` · `replay/score_team_identity.go:9` · `replay/document_vehicles.go:168`.
  **Et, dans le meme fichier, une seconde doc inversee** : `replay/document.go:594` affirme « le
  film ne porte aucun gamertag » alors que `replay/lives.go:54` documente le gamertag « porte PAR LE
  FILM lui-meme, dans le meme enregistrement que le xuid (32 octets UTF-16LE) ».
- **Cout** M. **Gain** : neuf decisions de fait cessent de dependre de la base ; le rejeu hors ligne
  publie une equipe reelle (critere S5 du plan). Non chiffre en volume.
- **[D14-1]** `Team: -1` est un repli **ANONYME** : un litteral dans un `struct`, sans nom, sans
  commentaire au site, sans compteur (`replay/build.go:580`). Le repli sans roster de
  `zone_states_owner.go:364` est **a moitie nomme** (commente, sans fonction ni constante).
- **[D14-2]** Devrait etre : « le film ne porte pas de designateur d'equipe pour ce slot ».
  **Risque OUI, inconditionnel** : `replay/build.go:580` s'applique a toutes les pistes ; et
  `replay/zone_states_owner.go:364` se declenche sur `len(teams) == 0` — l'absence de la TABLE,
  jamais un diagnostic sur le film.
- **[D14-3]** « 0 piste publiee a `team == -1` » et « `ownerChecked > 0` sur 100 % des artefacts »
  sur le corpus gate. **Pas de compteur** pour `Team == -1` ; compteur indirect pour la zone
  (`OwnerChecked` / `OwnerAgreed`).
- **Lot du plan** : **1.7** (l'equipe reelle sans base), amont `traverse.go` + 1.4.

### B2 — L'INDEX DE JOUEUR d'un slot : onze voies d'inference pour une table que le film ecrit

- **Bloquant nomme** : aucun lecteur de production de la **table des joueurs de `chunk_00`**
  (32 enregistrements de `0x1450` octets, XUID en clair a 85 bits, gamertag a `sub+0xc14`, l'ORDRE
  des enregistrements EST le `player_index`). Le lecteur n'existe que dans des instruments de
  recherche (`filmdec/chunk00_section3_research_test.go`, `section3_roster_research_test.go`).
  Verifie : `grep -rn "0xc14" --include='*.go' internal/` ne rend que des `*_research_test.go`.
- **Fait decide** : l'index de joueur d'un slot — donc l'auteur de chaque action d'objectif, le
  porteur du drapeau et du crane, la couronne VIP, le tueur / la victime / l'assistant de chaque
  frag, l'attribution des tirs et des touches.
- **Les onze voies heuristiques qui le decident aujourd'hui** :
  1. `objectiveevents/slotidentity_deaths.go:240` — instants de mort, `deathInstantToleranceMS = 150`,
     `deathInstantMargin = 2`, `deathInstantMin = 3` ; appariement glouton `:264`.
  2. `objectiveevents/slotidentity.go:97` — triplet K/D/A confronte a la feuille.
  3. `objectiveevents/slotidentity_elimination.go:124` — elimination sur le roster.
  4. `objectiveevents/slotidentity_residue.go:87` — residu de manche (la plus fragile).
  5. `replaybuild/matchfacts.go:406` — l'ORDRE des quatre voies ci-dessus.
  6. `killsource/bijection.go:147` — matrice de votes + algorithme hongrois + 40 redemarrages.
  7. `killsource/roster.go:66` — `nPlay` derive du seul kill-feed.
  8. `killsource/walk.go:195` — rejet hors bande bipede (cf. B5).
  9. `sync/killcollector/shots.go:122` — motif du xuid au bit pres, **premiere occurrence gagnante**.
  10. `replaybuild/kills.go:169` — gamertag -> xuid depuis le fil des morts, **le premier gagne**.
  11. `replay/player_index.go:110` — roster derive du fil des morts.
- **Preuve** : handoff §2 point 2 — 32 slots sur **1 351 films sur 1 351**, sept builds ;
  `filmdec/equipe_film_oracle_research_test.go` : `filmIndex − rang` constant sur **76/76 films**.
- **Negatifs mesures du repli actuel, chiffres** : un joueur qui meurt moins de 3 fois echappe par
  construction — `c0a82e88` : le slot 22, LE voleur ET LE captureur, perdu, calque `objectives`
  ampute de ses DEUX seules actions ; `43716616` : 62,3 s du plus gros porteur d'Oddball a 0 ;
  `51ebbc0f` : une manche entiere absente, ecart K/D/A cumule de 9 ; `3372e7eb` : **6 joueurs
  publies pour 8 a la feuille**, les deux manquants a 0 mort ; `4f77afc1` : 18 vies
  `index_hors_table` ; BTB — **62 assistants nommes pour 122 a l'API (51 %)** ; et
  `sync/killcollector/shots.go:117-118` : la voie du motif rend **77,0 % d'accord** contre 22,8 %
  pour l'ordre de la base (239 films, 16 411 kills, McNemar z = 92,8) — soit **23 % de desaccord
  silencieux** sur l'attribution des tirs.
- **ASYMETRIE relevee, verifiee sur pieces** : deux lecteurs du meme fait, deux regimes.
  `replay/player_index.go:30` EXIGE la concordance entre tous les chunks et refuse de publier sinon
  (« une majorite serait un vote, et c'est exactement ce que ce chantier a retire ») ;
  `sync/killcollector/shots.go:209` retient la PREMIERE occurrence sans exiger aucune concordance.
- **Cout** M. **Gain** : les onze voies disparaissent ensemble, avec leurs sept constantes calibrees
  et leurs trois corpus de calibration a maintenir.
- **[D14-1]** Chaque voie est **NOMMEE** (`OriginDeathInstants`, `OriginSheetTriplet`,
  `OriginElimination`, `OriginRoundResidue`, `PathWalk`/`PathScan`) — sauf les deux « le premier
  gagne » (`replaybuild/kills.go:169`, `killcollector/shots.go:209`), **ANONYMES**.
- **[D14-2]** Devrait etre : « la table d'identite de `chunk_00` est absente ou illisible sur ce
  film ». **Risque OUI, a 100 % des cuissons** : `objectiveevents/slotidentity_rounds.go:137`,
  `replaybuild/matchfacts.go:408`, `killsource/decode.go:140`, `replaybuild/kills.go:89`
  appellent ces voies **sans condition**.
- **[D14-3]** **Le compteur de retrait existe deja et voyage jusqu'au document** :
  `identity.statborgSlots[].method` (via `RoundIdentity.Origin` ->
  `replay/identity_registry_section.go:279`). Critere D14 applicable voie par voie, sans
  instrumentation neuve. **TROU CONSTATE PAR L'AUDITEUR PRINCIPAL** :
  `replay/identity_registry_section.go:288` (`methodeStatborg`) ne traduit PAS
  `OriginRoundResidue` — un slot nomme par la voie du residu est publie avec `MethodNone`
  (« aucune voie — le lien n'a jamais eu de candidat »). La voie la plus fragile est donc la seule
  que le compteur ne voit pas, et le document affirme l'absence d'un lien qui existe.
- **Lots du plan** : **1.5** (table des joueurs), **1.6** (registre d'identite), **1.8** (kill feed).

### B3 — L'origine d'une pose de PANNEAU DE MUR (item 1.9.1, fixe par l'utilisateur)

- **Site** : `internal/games/halo_infinite/film/replay/equipment_placements.go:313`
  (`equipmentOrigin`), seuil `originDropWindowUS = 200_000` (`:227`).
- **Fait decide** : `deployed` contre `dropped` — la seule origine que le rendu dessine.
- **Heuristique** : purement temporelle depuis l'item F.1 du 2026-09-13 (la clause de distance a ete
  RETIREE) : `deployed` si l'ecart entre la creation et la fin de vie du poseur depasse 200 ms.
- **Ce que le film ecrit, pour le MUR seulement** : l'evenement de type 103
  `EquipmentSpawnedObject`, dont `ref1` (index 13 bits, base 512, + 2 bits de generation) designe
  l'objet engendre — **216 des 216 poses de panneau publiees du parc, dans les trois origines**.
  Lecteur : **A PORTER**. La lecture de TETE de la liste d'evenements est PORTEE
  (`filmdec/event_list.go:95`, `PacketHeadEventType`) et **927 des 931 occurrences du 103 sont en
  position 1** ; les decodeurs de reference gardee (`readDom1Ref`, `readPlainRef`,
  `event_list.go:187` et `:209`) sont PORTES pour les evenements vehicule. Le decodage du 103 et de
  ses trois references reste a ecrire : c'est le bloquant, et il est petit.
- **Preuve** : `RAPPORT_F0_DEPLOIEMENT_103_2026-09-13` §1.2 et §2.3 — resolution 93,6 % contre 2,2 %
  au temoin de hasard (facteur 42), dt median +49 ms ; objet designe nomme `0x528fce46` 227 fois et
  `0x686b40c9` 3 fois, les deux seuls objets `kind = "deployed"` du manifeste.
- **Gain chiffre** : 216 poses de panneau cessent d'etre classees par une fenetre ; et les
  **7 defauts d'origine sur 216 panneaux** que le parc porte aujourd'hui (3 classes `dropped`,
  4 `unknown`, alors qu'un panneau ne peut pas etre lache a la mort) se ferment par lecture —
  §7 « decouvertes » du rapport F.0.
- **Cout** S a M.
- **CE QUE L'AUDIT CORRIGE DANS L'ENONCE DU PLAN** : l'item 1.9.1 propose, pour les appareils
  PORTES, « l'enregistrement de creation de l'objet (`consumeDefaultStateTI37` : reference de
  createur, `ability-enabled-id`) s'il porte la cause ». **Cette piste est deja un negatif mesure**
  et le code le dit : `filmdec/equipment_creation.go:29-30` — « leur porte est FERMEE sur
  503 records de creation sur 503 (2026-08-17) », pour les DEUX champs. Le lot 1.9.1 ne doit donc
  pas la re-instruire : les appareils portes restent en heuristique temporelle comptee (ligne C1).
- **[D14-1]** Repli **NOMME** (`OriginUnknown`, contrat du champ `Origin`), compte
  (`coverage.placements.deployed/dropped/unknown`, invariant teste).
- **[D14-2]** Apres conversion, la condition devrait etre « aucun 103 ne designe cette vie ».
  **Risque OUI aujourd'hui sur un sous-repli anonyme** : `equipment_placements.go:330`
  (`if gap < bestGap`) classe l'origine sur une vie qui NE CONTIENT PAS l'instant de la pose, sans
  que ce cas soit distingue du cas couvrant.
- **[D14-3]** « 100 % des poses de panneau classees par le 103 » et « 0 panneau classe `dropped` ou
  `unknown` » sur le corpus gate. Compteur a etendre : `ByFamilyOrigin` existe
  (`equipment_placements.go:199`), il ne ventile pas par PROVENANCE de la decision.

### B4 — L'activation de la PREMIERE colline : le code dit lui-meme que la donnee est dans l'image-cle

- **Site** : `internal/games/halo_infinite/film/replay/zone_states_hill.go:110`
  (`hillFirstContact`), bornes posees `:156-172`.
- **Fait decide** : l'instant d'activation de la premiere colline — la borne gauche de la premiere
  periode publiee.
- **Heuristique** : « premier contact » — premiere emission du designateur, du proprietaire ou de la
  jauge dans une portee de 3 slots. Le code l'appelle lui-meme « une borne HAUTE de l'activation ».
- **Ce que le film ecrit** : `zone_states_hill.go:21-25` — « la premiere designation vit dans
  l'IMAGE-CLE, que le delta ne re-emet pas » ; mesure : l'objet de mode est absent des images-cles a
  0 et 20 s, present a 40 s sur les 4 films. Lecteur : **A PORTER** — le cadre juste
  (`filmdec/keyframe_fullstate_loop.go`, `WalkKeyframeFullState`) existe depuis R7-d et **n'est
  branche nulle part** ; la production ne ferme **aucun** record d'image-cle (0 sur 62 686,
  6 films, 3 builds).
- **Cout** L (depend du lot 1.4). **Gain** : non chiffre (les secondes de colline mal datees ne sont
  pas mesurees).
- **[D14-1]** Repli **ANONYME** : `hillFirstContact` n'est pas nomme comme un repli et n'est pas
  compte ; il s'applique a 100 % des films, toujours.
- **[D14-2]** Devrait etre : « aucune image-cle de ce film ne porte la designation ».
  **Risque OUI, inconditionnel**.
- **[D14-3]** Accord de `hillFirstContact` avec la designation d'image-cle a +-1 frame sur le corpus
  gate. **Pas de compteur.**
- **Lot du plan** : **1.4** (cadre d'image-cle d'etat complet en production).

### B5 — La bande de slots bipede, deduite puis COMBLEE

- **Sites** : derivation `filmdec/offline_biped_band.go:161` (`bipedSlotBand`), comblement `:186`
  (`fillSlotBand`) et `:190` (`filledSlotMap`) ; portes de production
  `killsource/walk.go:195`, `replay/build_from_film.go:126`,
  `sync/killcollector/positions.go:294`.
- **Fait decide** : QUELS slots sont des corps de joueur — la porte en amont de tout le perimetre
  (positions, vies, identites, dead-states credibles).
- **Heuristique** : union des slots `ti=35` vus a la PREMIERE image-cle de chaque chunk balaye
  (`break` a `:178`) plus le chunk suivant, puis **comblement de tous les trous entre min et max**,
  sans aucune borne de largeur.
- **Ce que le film ecrit** : la table des joueurs de `chunk_00` (cf. B2). **A PORTER.**
- **Negatif / preuve** : `RAPPORT_LOT_H_VERSIONS_2026-09-13` §D5 — bandes mesurees
  `[512, 643..767]`, `[128, 757..767]`, `[512, 7808]`, `[512, 8064]` selon le build ; a 7 808 ou
  8 064 la bande comblee couvre **le domaine entier de 13 bits** (`slotBandDomain = 8192`) et
  **la porte ne filtre plus rien** (`hors_plage_bipede` tombe a 0 ou 1, le refus bascule sur
  `victime_hors_roster`). Le rapport le qualifie « point a instruire, pas encore une cause etablie ».
- **Cout** M. **Gain** : non chiffre.
- **[D14-1]** Le comblement est **NOMME** (`fillSlotBand` / `filledSlotMap`) mais presente comme une
  PROPRIETE (« les slots biped sont alloues dans une bande contigue »), pas comme un repli ; le
  rejet cote killsource est **ANONYME** (trois `continue` nus, `killsource/walk.go:195`, `:198`,
  `:201`, `:204`).
- **[D14-2]** Devrait etre : « un slot bipede existe qu'aucune image-cle ne porte ». Le comblement
  n'est **pas conditionnel du tout** (`offline_biped_band.go:186`) : il s'execute toujours, y
  compris quand le film a tout dit. **Risque OUI**, et dans les deux sens (bande trop large : la
  porte ne filtre plus ; bande trop etroite : `killsource/walk.go:195` rejette un record vrai).
- **[D14-3]** « largeur de bande == effectif du match » verifiable des que `chunk_00` est lu.
  **Pas de compteur** publie (ni cardinal de bande, ni bornes, ni `hors_plage_bipede` — ce dernier
  n'existe que dans un test de recherche, `killsource/btb2025_abstention_research_test.go:95`).
- **Lot du plan** : **3.5** (bande de slots bipede par build).

### B6 — Le registre ECS suppose du build `HI_1_13_0` sur tous les films

- **Site** : `internal/games/halo_infinite/film/filmdec/registry_fingerprint.go:111`
  (`warnUnknownRegistry`), appele `filmdec/registry.go:214` dans `parseRegistry`, atteint par la
  cuisson (`filmdec/film_context.go:254`) et par `sync/killcollector/hits.go:114`.
- **Fait decide** : **aucune, et c'est le defaut** — la decision reellement prise est « decoder quand
  meme avec la grammaire `HI_1_13_0` », au `return` muet de `registry_fingerprint.go:113`. Or
  l'ordre des composants EST l'index de bit du masque de presence : c'est tout le dispatch.
- **Heuristique** : FNV-1a 64 sur `kind|flags|nom` des slots non vides, compare a **UNE SEULE**
  constante `KnownRegistryFingerprint = 0x61e492dd4de7fd4e`.
- **Ce que le film ecrit** : l'identification de build, en clair, dans la section suivante du MEME
  chunk (`filmdec/registry.go:186`) ; `FilmMajorVersionFromHeader`
  (`filmdec/film_major_version.go:65`) lit deja la version dans les 4 premiers octets.
  **A PORTER** : indexer la table (archetype, composant) par build.
- **Preuve** : `RAPPORT_LOT_H_VERSIONS_2026-09-13` §D3 — 8 empreintes distinctes relevees,
  **9 cuissons sur 9 en WARN** pour les versions 33/37/39/40, 0 sur 7 pour la version 41 ; les films
  anciens portent 49 blocs / 1 029-1 033 slots contre 50 / 1 067 pour la reference.
- **Cout** L. **Gain** : « la table sur laquelle repose TOUT le dispatch du decodeur n'est valable
  que pour le build `HI_1_13_0` » — 228 films du cache se decodent sous une grammaire dont le film
  declare lui-meme qu'elle n'est pas la sienne.
- **[D14-1]** **NOMME** : `registry_fingerprint.go:105`, « Le film reste decode : l'empreinte est un
  signal, pas une porte ».
- **[D14-2]** Devrait etre : « le build est inconnu -> erreur typee, ou grammaire indexee par
  build ». **Risque OUI** : `registry_fingerprint.go:111` ignore la version du film, pourtant lue
  dans la meme cuisson (`replay/build_from_film.go:103`).
- **[D14-3]** Le WARN existe (`registry_fingerprint.go:118`) mais est **deduplique par empreinte ET
  par PROCESS** (`sync.Map registryWarned`, `:115`) : le denominateur est detruit, le critere
  « 0 WARN sur N cuissons » est **inutilisable en l'etat**. Il faut d'abord un compteur par film.
- **Lot du plan** : **3.2** (le registre par build).

### B7 — Les composants d'objectif ti=11 / ti=12 : 100 % de desynchronisation

- **Sites** : `replay/bomb_armings.go:246` (existence d'un armement) ·
  `replay/zone_states_owner.go:179` (quel slot `ti=13` EST quelle zone) ·
  `replay/zone_states_hill.go:304` (ou est la colline gardee).
- **Fait decide** : l'existence et l'instant de chaque armement de bombe ; l'appariement
  jauge <-> zone ; la zone de la colline.
- **Heuristiques** : sommet de rampe + quantum plein 254 ; sommet le plus proche a +-2 000 ms puis
  vote MODAL puis argmax ; vote modal des positions.
- **Ce que le film ecrit** : ti=11 porte `progress` (i12), `required-progress` (i13), `state` (i14)
  et `managed-objective-object-reference-component` (i3, R(32)) qui reference l'objet physique.
  Lecteur : **A PORTER** — `filmdec/objective_scan.go` est un INSTRUMENT, pas une source (voie
  delta : legalite 45,7 % / 40,3 %, sous le hasard ; voie image-cle : ancre juste a UN BIT pres).
  `filmdec.KeyframeClosure` (0.A.3) donne le bloquant par archetype.
- **Preuve** : ti=11 et ti=12 sont a **100 % de desynchronisation** en image-cle aujourd'hui
  (handoff §4 bis, note `NOTE_IMAGECLE_ETAT_COMPLET_2026-09-13` point 4).
- **Cout** L. **Gain** : non chiffre.
- **[D14-1]** Pas de repli : degradation vers le silence, comptee (`BelowFull`, `Paired`/`Unpaired`).
- **[D14-2]** Sans objet en l'etat.
- **[D14-3]** Fermeture de ti=11 et ti=12 a 100 % au ratchet `KeyframeClosure`.
- **Lots du plan** : **1.4** puis **3.6.b**.

### B8 — Le timer de reapparition et l'etat du moteur de jeu : decodes, typés, ZERO consommateur

- **Bloquants nommes** : `player-respawn-timer-component` (ti=5 i1, R(1)+R(10)+R(10)) — decode et
  typé a `filmdec/traverse.go:469` et `filmdec/capture.go:23`, `:37-38`, **aucun consommateur de
  production** ; `game-engine-current-state-component` (ti=0 i2, R(3)) — publie par un hook
  (`filmdec/components_game_engine.go:101-103`), **aucun consommateur** ;
  `game-engine-team-mapping-component` (ti=0 i0) — masque + une valeur R(4) par equipe presente,
  **jetes** (`filmdec/components_team_mapping.go:33-45`).
- **Faits decides a la place, par heuristique** :
  - `replay/closures_respawn.go:37` — fenetre de reapparition CALIBREE sur le film traite
    (mediane mesuree +- `respawnHalfWidthUS = 750_000`), pour nommer une vie anonyme.
  - `replay/death_context.go:213` — l'etat d'un coequipier (visible / en attente / hors de vue),
    `FenetreVisibiliteMs = 1000`.
  - `replay/inventory_dead_readings.go:55` — pourquoi une lecture d'inventaire est vide,
    `invDeadWindowMS = 8_000`.
  - `replay/t0_film.go:292` — le COUP D'ENVOI publie, detecte par le premier mouvement
    (`t0FilmJumpM = 5.0`, `t0FilmCumulM = 0.5`, `t0FilmWindowMS = 1000`, `t0FilmMinBurst = 2`,
    `t0FilmMaxDelayMS = 120000`).
  - `replay/score_team_identity.go:42` — quel camp est derriere chaque slot d'equipe (repli par la
    somme des frags).
- **Preuve des heuristiques** : elles sont bien mesurees — reglage p05/p95 ECHOUE (p95 a 51,7 s et
  67,7 s, 13 vies sur 13 contestees) ; inventaire 88,3 % contre 1,1 % (facteur 82) ; t0 marge
  mediane 22 700 ms, ecart-type 299 ms, CV 0,013 sur 83 matchs. **Que le film n'ecrive PAS ces faits
  est, lui, non etabli** : les composants sont la, ils ne sont pas lus.
- **Cout** M. **Gain** : non chiffre.
- **[D14-1]** Replis **NOMMES** (« FERMETURE B », « SANS FIL DES MORTS, RIEN NE BOUGE »,
  `identityByFrags`) sauf `EtatHorsDeVue` (`replay/death_context.go:222`), **ANONYME** dans sa forme
  (un `return` de fin de fonction) bien que compte.
- **[D14-2]** Devrait etre : « le composant d'etat n'est pas present dans ce film ».
  **Risque OUI** pour `score_team_identity.go:56` (la condition est `scores == nil ||
  scores[0] == scores[1]` — l'absence ou l'egalite des scores DE LA BASE, jamais un diagnostic sur
  le film) et pour `death_context.go:213` (ne consulte aucun canal d'etat).
- **[D14-3]** Une fois ces composants publies : `ClosedByRespawn == 0`, « 0 coequipier classe par
  defaut », `unknown` residuel d'inventaire == 0, `t0FilmMs` du detecteur == `t0` de l'etat moteur a
  +-1 frame. Compteurs : `ClosedByRespawn` et `T0FilmCoverage` **publies** ; « hors de vue mesure »
  contre « hors de vue faute de canal » **non distingues**.
- **Lot du plan** : aucun a ce jour — **a rattacher a 3.6** (ports de composants) ou a un lot 1.9.

---

## 6. TABLE (C) — le film NE l'ecrit PAS : heuristique legitime, elle reste avec sa couverture

**Regle de la table** : chaque ligne cite le NEGATIF MESURE qui la fonde — un chiffre et sa source.
Une ligne sans negatif mesure n'entre pas ici : elle va en table (D), « non etabli ».

| # | `fichier:ligne` | Fait decide | Heuristique (parametres) | NEGATIF MESURE qui la fonde | [D14] repli / risque / retrait |
|---|---|---|---|---|---|
| C1 | `replay/equipment_placements.go:313` | origine d'une pose d'un appareil PORTE (capteur, traqueur, ecran, champ) | fenetre temporelle 200 ms (`:227`) ; la clause de distance a ete retiree le 2026-09-13 | **F.0 §2.3** : le type 103 designe 216/216 poses de panneau et **0 sur 31 `deployed` + 0 sur 145 `dropped`** d'appareils portes ; voie indirecte par piece voisine a +-5 s : **14,7 % sur `deployed` contre 21,8 % sur `dropped`** (elle ne separe pas) ; le record de creation ti=37 a sa **porte FERMEE sur 503/503** (`filmdec/equipment_creation.go:29`). Le fait temporel, lui, est mesure des deux cotes : lachers 20-40 ms, deploiements 14-42 s | NOMME (`OriginUnknown`, compte `deployed/dropped/unknown`) · risque OUI sur un sous-repli anonyme (`:330`, vie la plus proche au lieu de la vie couvrante) · retrait : sans objet, la piste est fermee |
| C2 | `replay/equipment_placements.go:561` | le POSEUR d'une pose et son cap de visee | bipede le plus proche dans +-250 ms (`:42`) a <= 3,0 m (`:49`) ; cap a +-200 ms (`:54`) | porte de reference d'entite du default-state **FERMEE sur 503 records sur 503** (2026-08-17, `filmdec/equipment_creation.go:29`) ; mediane poseur 0,52-0,60 m sur 11 films contre 11 a 36 m pour le temoin | NOMME (`Owner: -1` + `OriginUnknown`) · risque OUI (`:592`, critere purement geometrique) · retrait : `withOwner == placements` — compteur existant |
| C3 | `killsource/calibrate.go:73` et `:93` | `axisW` / `indexW` de la marche des morts, puis `recordStateParam` (`:139`) | balayage de 21x3 = 63 configurations sur 400 paquets ; garde-fou `flatRatio = 2.0` -> **defauts 14/1** (`:95`) ; RSP balaye 0..5 **sans garde-fou** | « ne se lisent nulle part dans le film : installes au chargement de la map » (`calibrate.go:5-7`) ; les tables de precision dumpees du jeu **effondrent 3 films sur 4** quand on les injecte (`:11-14`) ; 4 films de reference = 4 couples DISTINCTS (14/1, 17/2, 16/2, 17/1) | NOMME (`calibration.Flat`) · risque OUI (`:93` ne distingue pas « hors espace » de « echantillon non discriminant ») · retrait : `Flat == false` sur 100 % du corpus — **pas de compteur effectif** : `Result.Calibration` n'a **aucun consommateur** dans `killcollector`/`replaybuild` |
| | | | | **AMENDEMENT** : le rapport du lot H §D4 tranche que ces largeurs « sont une DONNEE de la carte et du build ». Le remplacement n'est donc PAS une lecture du film mais une **entree de profil** (lot 3.4) ; `MapQuantEntry.AxisWidths` existe et le MEME collecteur la resout deja (`sync/killcollector/positions.go:209`). Degradation chiffree : **6,6 / 4,2 / 4,7 % de morts credibles** sur les pires films contre 28-64 % en version 41. Reserve : la calibration ne rend qu'une valeur UNIFORME, « qui n'est la largeur d'aucune carte » (`filmdec/traverse.go:1181-1184`) | |
| C4 | `killsource/assist.go:247` | `gate15` — longueur du corps du code 15, donc la chaine d'evenements | calibration par film : les deux valeurs essayees, `score[1] > score[0]` (`:258`) | etat RUNTIME du jeu, **absent du flux** (`killsource/eventbody.go:73-75`) ; le fichier l'ecrit : « il ne se lit pas, il se TRANCHE par film » (`:243-246`) | NOMME (`pickGate15`, valeur publiee `AssistStats.Gate15`) · risque : egalite tranchee a `false` sans marge (`:258`) · retrait : `gate15` derive du build — l'ecart `score[0]`/`score[1]` n'est pas publie |
| C5 | `replay/bomb_arms.go:295` et `:312` | QUI a arme la bombe | periode de portage fermee par lacher la plus proche, +-2 500 ms (`:149`) ; a defaut l'unique periode couvrante | **negatif Ghidra du 2026-09-04** (`bomb_arms.go:9-12`) : le Lua `primitive_carriable_arming_base` (tag `25af9c45`) porte 6 evenements d'armement et **le seul porteur d'identite est `activatingTeam`** — le moteur n'ecrit pas l'acteur. Ecart lacher-armement +247 a +259 ms, 10 appariements / 4 films | NOMME (`BombActorSourceActiveCarry`, compteur `ArmingsByActiveCarry` publie) · risque OUI (`bomb_arms.go:204` : la passe 2 se declenche aussi quand la periode a ete CONSOMMEE, pas seulement quand le film est muet) · retrait : `armingsByActiveCarry == 0` — compteur existant |
| C6 | `replay/bomb_armings.go:314` | la MECHE publiee, et la retenue du calque entier | fenetre de sens 120 000 ms (`:84`), dispersion CV <= 0,20 (`:91`), mediane des delais corriges | la mèche n'est ecrite nulle part ; CV mesure **0,016 / 0,016 / 0,017** contre **0,725** pour la lecture simple que ce seuil a refutee | NOMME (`BombFuseMS = 4930`, « valeur de REFERENCE, DEDUITE ») · risque NON (branche exige `len(delays) == 0` ET couverture complete) · retrait : compteur `Detonations` + `verdict.Measured` existants |
| C7 | `objectiveevents/round_bounds.go:139` et `objectiveevents/statborg.go:464` | quelles MANCHES sont reelles, et leurs bornes | medianes et majorites : `2*len(debuts) > slotsMax` (`round_bounds.go:289`), mediane BASSE (`:326`) ; admission `statMinRoundRun = 3` OU (`statMinRoundRecords = 25` ET `statMinRoundRecordShare = 10 %`), tolerance `statMaxEmptyRoundRun = 1` | le film ecrit le NUMERO de manche par enregistrement, **jamais la liste des manches jouees ni leurs bornes**. `24dbb67d` : 2 slots declarent la manche 1 des 85 193 ms, les 8 autres a 298 909 ms — le minimum jetterait 3 612 enregistrements legitimes, la mediane en jette 16. 65 films, 227 manches brutes : ancrage fortuit <= 5,84 %, manche reelle >= 21 % | NOMME a trois niveaux (`materialRounds`, tolerance de manche courte, **repli terminal `out[0] = true`** `statborg.go:637`) · risque OUI sur le repli terminal (il DECRETE la manche 0 reelle quand aucune ne l'est, y compris sur un film tronque) · retrait : **pas de compteur** du repli terminal |
| C8 | `objectiveevents/named_bounds.go:84` | combien d'ACTIONS un pas de compteur produit | `maxUnrollPerStep = 16` ; un pas au-dela est rejete en bloc (`:207`) | le film ecrit la VALEUR du compteur, **pas les instants unitaires**. 68 artefacts confrontes a l'oracle API et a la feuille : population saine, pire pas **3** ; population aberrante, plus petit pas **64**, puis 84, 9 482, 15 608, 15 610 — **facteur 21 de vide**. Validation a l'unite : `16ea3668`/`f8efc5ca`/`8bc6074f` tombent a 6, 6 et 8 = exactement la feuille | NOMME ET COMPTE (`eventBudget.rejetes`, `named_bounds.go:112`, publie en `slog.Warn`, jamais tronque) · risque OUI, structurel et assume (marge 5,3x le pire pas sain) · retrait : `deroulages_rejetes = 0` — **modele D14, reserve : c'est un log, pas un expvar** |
| C9 | `replay/zone_states_owner.go:105` | la LETTRE A/B/C d'une zone | ordre croissant des slots `ti=13` de jauge, borne a 3, bijection exigee | **negatif Ghidra du 2026-08-24** : la lettre n'existe dans AUCUNE donnee decodee — ni catalogue de formes, ni variante, ni binaire. Positif : 8/8 films par carte rendent la meme permutation, 8 cartes / 17 films | NOMME (trois portes fermees, silence plutot que decalage) · risque NON · retrait : sans objet |
| C10 | `replay/vehicle_rides.go:83` | qu'un joueur est a bord, de quand a quand, et de QUEL vehicule | trou de replication >= 3 000 ms + vehicule le plus proche sous 1,5 m en plan, echantillon < 1 s | le bloc supplementaire du record d'image-cle `ti=40` **NE NOMME PAS son occupant** : le meilleur canal designe le bon occupant **1 fois sur 18 (5,6 %)**, exactement le score du temoin par permutation (`filmdec/vehicle_occupancy.go:20-27`). Elargir le rayon n'ajoute aucun trou confirme | NOMME (« LE TROU RESTE, EN REPLI », `vehicle_rides_events.go:39`, compteur `st.repli`) · risque OUI (`vehicle_rides.go:226` : un episode d'evenement REEL mais non rattache ne ferme pas la porte) · retrait : `st.repli == 0` — compteur en **journal seulement** |
| C11 | `replay/vehicle_tracks.go:84` | le CAP publie d'un vehicule | direction de la velocite `i1` au-dela de 5 m/s ; en deca, aucun cap | `i2` est **REFUTE** et `i21` est **ABSENT** de `ti=40` (`vehicle_tracks.go:81-83`, rapport V1_CONDUCTEUR_VISEE_2026-09-01) ; oracle V1a.3 : ecart median 1,7-2,1°, R 0,992-0,997 contre 51-88° pour le temoin | report du dernier cap, **ANONYME** (`:380`) · risque NON · retrait : **pas de compteur** |
| C12 | `replay/vehicle_tracks.go:33` | les bornes d'une vie de vehicule | tolerance `vehicleCensusTolUS = 20 s` autour du recensement d'image-cle | « le lot V3 visait la DESTRUCTION datee par la mort du conducteur : **REFUTEE** sur 460 vies et 12 films » (`vehicle_rides.go:5-7`). Intervalle d'image-cle median 20,00 s, p90 20,00-20,02 s sur 8 films | borne haute par defaut `lastUS + tol`, **ANONYME** (`:166-167`) · risque OUI (indistinguable d'une vie qui court jusqu'a la fin) · retrait : **pas de compteur** |
| C13 | `replay/document_ground_weapon_items.go:322` | QUEL objet au sol une prise d'arme consomme | meme famille + fenetre de vie + ramasseur a <= 1,5 m ; l'objet le plus proche gagne | **trois hypotheses de lien natif REFUTEES** (`document_weapon_changes.go:19-24`) : suppression d'entite 1/71 sous son propre temoin ; attachement au porteur 1/21 ; appariement par les armes 5-12 % de gagnants nets contre 70 % exiges. Positif : a l'instant d'une prise, l'objet le plus proche est a **0,61 m de mediane** (74,8 % sous 1 m) contre 4-7 m pour un autre bipede ; sans le critere de famille, **27 mauvaises familles sur 33 liens** | NOMME (`Picker: -1`, fins `seen`/`open`) · risque OUI (`:343`) · retrait : `PickupLinked == TakesTotal` moins les prises non liables — **l'ecart n'est pas ventile par cause** |
| C14 | `replay/ground_weapon_bounds.go:146` | QUI fait disparaitre un objet au sol et QUAND | premier passage d'un bipede a < 1,5 m dans la fenetre de bornage | « le film ne porte AUCUN evenement type pick-up » (mesure du 2026-08-12) ; le record DEL n'est pas isolable (**78 090 candidats pour 477 vies**) ; l'hypothese « la reference du `biped_pickup` designe l'objet » est REFUTEE (`512 + index` = slot du RAMASSEUR, 32/32 paires de verite terrain) | `gwPickupHit{Found:false}` puis `Status = unknown` — **ANONYME au site** mais compte (`GroundWeaponCoverage.Unknown`) · risque OUI (`:160`) · retrait : recalcul du cycle sur `padPickups[].t` (instant natif) |
| C15 | `replay/killpos.go:40` | les coordonnees publiees d'un kill | echantillon le plus proche a +-120 ms (`replay/shots.go:28`) ; abstention si deux corps | le record de tir (type 105) **ne porte AUCUNE position monde** (`vehicle_shots.go:12-16`) ; le fil des morts ne porte que victime + instant (`lives.go:47-63`) | pas un repli (abstention comptee : `Both`/`KillerOnly`/`VictimOnly`/`Dropped`/`NoBridge`) · risque NON · retrait : sans objet |
| C16 | `killsource/botmeta.go:90` | slot, identifiant et nom des bots | balayage bit a bit, nom UTF-16BE de 4 a 48, slot/botID a deux offsets negatifs constants | le stride declare de 2 076 octets est **VRAI pour nbBots = 1 et FAUX des nbBots >= 2** (une entree y est decalee d'un DEMI-OCTET) — `botmeta.go:18-21` | NOMME (« DEUX LECTEURS, ET LA RAISON EST MESUREE ») · risque OUI (`:74` balaie inconditionnellement, y compris a nbBots = 1 ou le stride est exact) · retrait : sans objet ; compteur `Roster.UnpinnedBots` existant |
| C17 | `replay/abilities.go:311` | la PALETTE de capacites du film, donc le NOM de chaque capacite | majorite / purete : 0,90 des 10 lectures, unanimite en deca | le registre de `chunk_00` est bit-a-bit identique d'un film a l'autre pour les noms et flags ; sa longueur ne suit pas les familles (1 973 120 octets sur un film famille A comme sur trois films famille B) ; **aucun marqueur de groupe de tags (`sofd`, `eqip`, `vcdd`, `uwfa`, `glpa`) dans aucun chunk** | NOMME (`return nil` = aucun nom publie, journalise « non classee ») · risque NON · retrait : sans objet ; **pas de compteur** du taux de classement au document |
| C18 | `replay/abilities.go:149` | une lecture de capacite tenue pour du BRUIT | rang > `abilityRankDomainMax = 27` | les blocs `sofd` du jeu comptent au plus ~27 entrees (mesure independante) ; sur 64 films du parc, **2 lectures sur 4 952 depassent**, zero collateral. Le filtre par fenetre de vie a ete essaye puis ECARTE (14 lectures legitimes hors fenetre) | NOMME ET COMPTE (`AbilityCoverage.ScanNoise`, invariant `Reads == ScanNoise + Unpublished + Published`) · risque marginal (`:156` teste un plafond global, pas l'appartenance a la palette) · retrait : `scanNoise == 0` — **compteur deja en place** |
| C19 | `replay/document_ability_impulses.go:282` | de QUEL equipement parle une impulsion ou une charge | dernier rang i48 de la MEME vie, anterieur, tolerance `lifeGapUS` (5 s) | le `sub` du composant a ete essaye comme discriminant et **REFUTE par le corpus** (R8 §8.5) ; sans decoupage par vie, signature d'heritage sur `11de8353` ; sans anteriorite, **4 des 8 lectures de `00ba2e1c`** passent du propulseur au grappin | NOMME ET COMPTE (`NoIdentity` / `OtherFamily` separes, invariant a six cases) · risque OUI (`:285` : la tolerance de 5 s aux DEUX bornes peut rattacher un geste de bord de vie a la vie voisine) · retrait : `noIdentity == 0` — compteurs existants |
| C20 | `replay/inventory_ammo_rules.go:53` et `inventory_grenades_rules.go:91`/`:112` | le chargeur, la reserve, la jauge, l'emplacement degaine, et les compteurs de grenades | « le plus long » parmi les parses admissibles ; « le PREMIER » motif i22 apres l'ancre, puis repli positionnel dans `[ammoStart-216, ammoStart-127]` bits | les VALEURS sont ecrites, **leur OFFSET ne l'est pas** — il n'y a aucune table de position dans le record ; **51 records sur 150 admettent plusieurs parses**. Loi de position etablie AVANT implementation sur **24 films, 1 167 records** (loi mesuree [-204, -139], elargie de 12 bits, parasite le plus proche a 105 bits) | NOMMES ET COMPTES : `AmmoCandidates` publie sous `cand` ; R2b declare « LE REPLI POSITIONNEL », marque par lecture (`GrenadesByPosition`) et compte par film. **Les deux meilleurs modeles D14 du perimetre** · risque NON · retrait : `cand == 1` et `GrenadesByPosition == 0` — compteurs deja publies |
| C21 | `replay/inventory_dead_readings.go:55` | pourquoi une lecture d'inventaire est vide (`dead` contre `unknown`) | fenetre de 8 000 ms apres une mort du porteur | « LE DECODEUR NE SAIT PAS POURQUOI » (`inventory.go:170-173`) ; 8 films, 1 419 records, 247 lectures vides : **88,3 % des vides dans la fenetre contre 1,1 % des pleines (facteur 82)**, 93,8 % contre 0,7 % sur la verite terrain (137x) ; balayage 2-20 s, 8 s = separation maximale | NOMME (« SANS FIL DES MORTS, RIEN NE BOUGE ») · risque OUI mais mesure et assume (1,1 % de faux positifs) · retrait : sans objet tant que le respawn timer n'est pas porte (cf. B8) — compteur en **journal seulement** |
| C22 | `replay/equipment_episodes.go:411` | un episode de surbouclier actif | quantum brut `> OvershieldFullQ = 64`, machine a deux etats | « 0 faux positif sur ~150 000 mesures hors porteurs » ; les deployables ne se datent pas par les canaux mesures, la mobilite n'a pas d'instant d'usage par i54 | NOMME ET PUBLIE (`EndRead = false` a la fermeture de fin de vie, « pour qu'un client qui sonne la desactivation ne la sonne QUE sur une fin mesuree ») · risque NON · retrait : sans objet |
| C23 | `replay/grenades.go:221` | quel projectile appartient a un lancer de grenade | naissances a +-200 ms, la plus proche de la main, refusee au-dela de 4 m | le lancer porte son auteur et son rang, les pistes de projectile sont decodees, **le LIEN ne l'est pas**. 65 des 70 lancers apparient a +-200 ms contre 11-13 pour les memes lancers decales en bloc ; distance auteur-naissance mediane 0,77 u contre 6,4 pour un instant permute | NOMME (`GrenadeSrcBiped` publie dans `Src` : « la meme grandeur lue ailleurs ») · risque OUI (`:239`) · retrait : sans objet ; frequence mesurable par `Src`, **aucun compteur agrege** |
| C24 | `replay/pickup_origin.go:149` | l'ORIGINE d'un ramassage (`spawner` / `ground`) | < 1,0 m d'un point d'apparition catalogue, sinon d'une pose `dropped` ; position a +-100 ms | **quatre voies geometriques refutees** (`NOTE_ORIGINE_LEVITATION_2026-09-01`) : distance seule mediane 1,33 m et 46 % sous le metre, sans pouvoir de separation ; fin de vie d'objet 25,6 % d'injectivite ; levitation separation 0,00 m ; recurrence = trafic, pas points. La reference de l'evenement ne designe pas l'objet (32/32) | NOMME (« ABSTENTION EXPLICITE, jamais un repli ») · risque OUI **indirect** (`:231` fait dependre tout le seau `ground` du verdict C1) · retrait : `OriginUnknown == 0` avec `SpawnPointsState == established` — compteurs existants |
| C25 | `replay/flag_objects.go:481` | « cette naissance EST une rentree au socle », et quel socle | distance <= `flagHomeExactDist = 0,10 m` | 626 vies libres / 12 films CTF : rentrees **<= 0,008 m**, lachers **>= 0,324 m**, **rien entre les deux** ; temoin du defaut `b8a44fe8` (2 cm separaient une mesure juste d'une mesure fausse, 58,1 s de portage fantome) | pas de repli · risque NON · retrait : sans objet |
| C26 | `replay/geometry.go:220` | qu'un echantillon de position est un ARTEFACT DE DECODAGE | centiles p1/p99, marge de 12 etendues centrales, plancher 0,5 m, desactive sous 200 points | balayage des 64 artefacts du parc : **trou franc entre 17,7** (plus petit artefact) **et 9,5** (plus grand ecart legitime) ; `81c02726` : 1 point sur 16 064 a 440 m sous le terrain fixait `MinX`, `MaxY` et `MinZ` | garde de dernier ressort NOMMEE (`:243-246`) · risque non etabli · retrait : `ecartes == 0` — compteur en **journal seulement** |
| C27 | `replay/vehicle_tracks.go:66` | deux vies de vehicule sont LE MEME vehicule | meme chassis + meme point sous 0,5 m + debut dans l'intervalle non observe | 13 paires, ecart **0,00 m dans 12 cas et 0,01 m dans le dernier**, **temoin NUL** (0 paire a chassis differents) | pas un repli · risque NON · retrait : sans objet ; `cov.Merged` **publie** |
| C28 | `replay/map_weapon_pads.go:99` | un emplacement de socle de la carte est-il ALLUME | <= 1,0 m d'un socle publie du match, le plus proche gagne | **32 positions d'oracle sur 3 cartes, 32 appariees, mediane 0,01 m** ; negatif decisif : Cliffhanger porte 17 emplacements au fichier, 10 servis en CTF, **ZERO en Super Fiesta** | NOMME (`return nil` = le client retombe sur les socles du film ; `CatalogN` publie) · risque NON · retrait : sans objet |
| C29 | `replay/flag_neutral.go:80` | la variante est « drapeau neutre » | majorite des naissances : `>= 3` ET `> TeamBirths` | 41 / 16 / 18 naissances a 0,0 m d'un socle d'equipe sur 3 films ordinaires ; seuil ecrit avant mesure | NOMME (mode ordinaire par defaut, compte `NeutralFlag`/`NeutralBirths`/`TeamBirths`) · **risque OUI** : `:80` bascule sur un comptage alors que **l'appelant connait le nom de variante** et le descend deja pour tous les autres calques (`ZoneInput.Hill`, `BombInput.Scanned`, `VipInput.Scanned`, `SkullInput.Scanned`) · retrait : concordance `neutralFlag` <-> `game_variant_name` sur 100 % du corpus |
| C30 | `replay/skull_carries.go:209` | la DUREE publiee de chaque portage de crane | demi-fenetre `(w-1)/2`, `w` = mediane des ecarts intra-train mesuree sur le film | 1 113,0 s publiees pour **1 249,0 s a l'oracle API**, ratio **0,891**, manque proportionnel au nombre de periodes (**1,36 s par periode**, 100 periodes, 4 films) ; `w/2` faisait DEPASSER 2 joueurs sur 28 | pas de repli (film sans ecart mesurable -> aucune fenetre) · risque NON · retrait : sans objet |
| C31 | `killsource/scan.go:59` | l'existence et le contenu d'un dead-state (voie de rattrapage) | balayage de toutes les positions de bit, 4 tests simultanes | controle negatif : **1 seul candidat sur 208 913 712 bits** de paquets sans event ; concordance **346/346 au meme bit** avec la marche, desaccord 0 ; leurre 468 identifiants au hasard -> 0 candidat | NOMME (« L'HYBRIDE : LA MARCHE DECIDE, LE SCAN RATTRAPE », `PathScan`) · risque OUI (`killsource/decode.go:133` lance le scan sur TOUT le film inconditionnellement) · retrait : `Stats.Scan.Published == 0` — **compteurs publies** (`Stats.Agree`/`Disagree`) |
| C32 | `killsource/world.go:250` | quelles ancres d'image-cle sont valides | balayage exhaustif + filtres structurels, seuil `bits32(buf, q+32) < 50` | recupere 8 a 25 ancres selon le film que le walker officiel saute, **ZERO contradiction d'archetype** ; sans la restriction d'archetype : 3 480 slots, 1 705 contradictions | NOMME (`sweepKeyframe`, « BALAYAGE COMPLEMENTAIRE ») · risque OUI (`world.go:230` ajoute sans verifier que le walker s'est arrete) · retrait : 0 ancre recuperee — **pas de compteur** |
| C33 | `objectiveevents/statborg.go:151` et `:128` | un enregistrement statborg est-il reel ; une emission est-elle un vrai point de mode | `statMaxCounter = 2^20` ; `statMaxModeScore = 250` | 11 films tous modes : pire valeur saine **102 934**, plus petite aberrante **2 415 919 104**, **vide total entre les deux** ; `53ce4390` : un point isole a 2 104 faisait passer une manche fantome pour reelle et portait le score d'equipe de 1 a 2 104 ; canal frere nul dans 98,3 % de 3 986 enregistrements | NOMMES (`statCountersInDomain`, `modeScoreInDomain`) · risque theorique OUI pour 250 (`objectiveevents/score.go:239`) · retrait : ne pas retirer tant que l'ancrage reste relache — **pas de compteur** (rejets en `continue` muets) |
| C34 | `objectiveevents/rosterfit.go:72` | publier ou TAIRE l'integralite des actions d'objectif d'un match | `n <= StatPlayerSlots = 8` sieges | audit du 2026-09-10 §6, oracle `match_objective_stats_latest` : `4f77afc1` (BTB:CTF, 18 sieges) **65 prises publiees pour 4 reelles** ; `879a4dba` (23) desaccord dans les deux sens ; `5676a9ba` (21) 23 captures pour 51. **129 actions publiees sur 3 films, dont 65 qui n'ont pas eu lieu.** Le plafond de 8 slots est une contrainte de FORMAT du statborg | NOMME (`RosterFitsStatborg`, refus journalise) · **risque OUI, structurel** : la garde refuse sur un critere d'EFFECTIF alors que le fichier reconnait ne pas prouver la causalite (`rosterfit.go:36-38`) ; et `n <= 0` rend VRAI (`:72`), donc une cuisson sans faits de match publie sans garde · retrait : **compteur publie dans l'artefact** (`coverage.objectives.refusedByRoster`) — et la garde tombe des que `chunk_00` (B2) porte les 32 slots |
| C35 | `replay/flag_objects.go:356` | l'instant et la position du LACHER volontaire d'un drapeau | naissance d'une vie libre dans `]t0,t1[` a <= 1,5 m, hors socle ; position la plus proche a +-1 000 ms | **+1 129,3 s de portage faux** sur 197 periodes / 11 films CTF avant cette chaine, 69 joueurs sur 95 au-dessus de leur oracle, `closedByObject` valait 0 | compteurs `ClosedByObject`, `DropsRepositioned`, `ObjectLives` publies · risque NON · retrait : sans objet |
| C36 | `replay/vip_crown.go:89` | la fin de la periode de port de la couronne | premiere mort apres la selection, sinon selection suivante, sinon fin | oracle API `TimeAsVip` : **24/24 joueurs a +0,2-0,3 s**, recouvrement 100 % 3/3, temoin aleatoire 8/8 contre 0-1/8. Que « le film ne porte pas le bit VIP (script-side) » est en revanche **non etabli** -> voir NE6 | NOMME (compteurs `ClosedByDeath`/`ClosedBySelection` publies) · risque NON · retrait : sans objet |
| C37 | `replay/closures.go:148` et `closures_respawn.go:98` | l'identite d'une vie de bipede anonyme (fermetures A et B) | deduction par unicite : un seul candidat POSSIBLE, sinon rien. Fenetre de reapparition calibree sur le film +- 750 ms | 31 attributions / **99 refus** sur 7 films (87 contestees, 12 rejetees) ; le reglage p05/p95 a ECHOUE (p95 a 51,7 s et 67,7 s, 13 vies sur 13 contestees). « Des que deux candidats subsistent, RIEN n'est attribue » | NOMMES (« FERMETURE A », « FERMETURE B ») · **risque OUI** : `replay/identity_registry_pont.go:82` appelle `closeBridge` sans condition, et `replay/closures.go:341` (`freeLives`) ne retient que « vie sans xuid », y compris les vies REFUSEES par la lecture directe (`index_hors_table`, `lectures_divergentes`) · retrait : `ClosedByShot == 0` et `ClosedByRespawn == 0` — **compteurs publies** |
| C38 | `replay/identity_registry_bridge.go:98` | l'identite de TOUTES les vies quand aucun record de creation n'a ete lu | appariement glouton des morts promu en producteur de nom | les 7 paires EXACTEMENT ECHANGEES entre deux vies finissant a la meme image (`d9781168`, `64e8adfa`) | **NOMME, et c'est le SEUL repli du perimetre qui porte deja un kill-switch documente** : bascule 2026-09-08, cible « quand `testdata/inputs_000d5950.bin.gz` portera les creations », critere `bridgeNamedLives == 0` (`identity_registry_bridge.go:36-40`). Reserve : la cible est une condition, pas une date · **risque NON** (`identity_registry_pont.go:54`, `crea.Slots == 0`) · retrait : deja ecrit, compteur publie |

**Note de methode sur (C)** : C3 est la seule ligne de cette table dont le remplacement est acte par
un autre document — le rapport du lot H tranche que `axisW`/`indexW` sont une donnee de carte et de
build. Elle reste en (C) parce que **le FILM ne l'ecrit pas** ; sa conversion est le lot 3.4, pas un
lot 1.9. Le registre le dit plutot que de la ranger de force en (A).

---

## 7. TABLE (D) — NON ETABLI : ni lecture identifiee, ni negatif mesure

Une ligne entre ici quand l'heuristique decide un fait mais qu'aucune source du depot ne dit si le
film l'ecrit. Chacune porte **la question a instruire**.

| # | `fichier:ligne` | Fait decide | Ce qui manque | Question a instruire |
|---|---|---|---|---|
| NE1 | `killsource/feed.go:82` (amont de A7) | quel chunk est le pied de film | le manifeste le dit, mais c'est un descripteur EXTERNE | **`chunk_00` porte-t-il le type de chaque chunk ?** Sa section d'en-tete porte « une table de 123 u32 par type » (handoff §2) dont la semantique des valeurs 1..6 est explicitement hors perimetre du plan (§1.2) |
| NE2 | `killsource/options.go:89` (`Views = 8`) | combien de vues de replication sont marchees par paquet | aucune mesure citee ; la borne peut tronquer sans rien signaler (`killsource/walk.go:64`) | le paquet declare-t-il son nombre de vues ? Combien de paquets atteignent la borne ? |
| NE3 | `sync/killcollector/hits.go:131` (`WeaponHitPairWindowUS = 1 s`) | quelle touche appartient a quel tir | aucune mesure de la fenetre au site d'appel | le film lie-t-il le tir et le degat autrement que par le temps ? (les deux portent `(chunk, pidx)`, cf. A6) |
| NE4 | `sync/killcollector/credit.go:83` (`creditToleranceMS = 5`) | le couple (tueur, victime) des 29,3 % de matchs SANS film | justification de **comparabilite**, pas de mesure (`credit.go:79-82`) | sans objet a terme (les films expirent) — a documenter comme tel |
| NE5 | `replay/flag_carries.go:273` (`flagGrabMergeMS = 250`) | deux prises de drapeau sont LA MEME | aucune mesure de l'ecart reel entre `flag_grabs` et `flag_steals` | quel est l'ecart mesure entre les deux emissions du statborg pour un meme vol ? |
| NE6 | `replay/vip_crown.go` (en-tete) | « le film ne porte PAS le bit VIP (script-side) » | assertion sans mesure citee | le composant de mode VIP existe-t-il dans le registre ? (le resultat, lui, est valide 24/24) |
| NE7 | `replay/held_object_carry.go:20-22` | « la mort ferme SANS emission » du canal des armes tenues | assertion sans chiffre | combien de morts de porteur sont suivies d'une emission du canal ? (mesure directe, corpus d'Assaut) |
| NE8 | `replay/zone_states.go:309` (`zoneRampMinSamples = 3`, `zoneRampMinAmplitude = 4096`) | « il y a une capture en cours ici » | **deux constantes posees sans mesure citee**, dans un bloc dont l'en-tete annonce « repris TELS QUELS de la mesure » | instrumenter : combien de rampes fausses a chaque seuil ? |
| NE9 | `replay/zone_states.go:144` (`zoneCaptureDistanceM = 5,0`) | l'attribution geometrique d'une capture a une zone | seuil ecrit AVANT la mesure ; la mesure qui suit dit mediane joueur-zone **6,6 m**, seuls ~10 % DEDANS, taux strict 73,2 % | le seuil a-t-il ete re-arbitre apres cette mesure ? |
| NE10 | `replay/zone_attribution.go:149` (`DefaultMaxGapFrames = 2`) | la position d'un joueur a l'instant d'une capture | raisonnement ecrit, aucun chiffre de derive | mesurer la derive reelle a 1, 2, 3 frames |
| NE11 | `replay/bomb_stats.go:461` | « la victime de ce kill portait la bombe » | la constante appliquee vaut **150 ms** (`replay/lives.go:41`) alors que le commentaire du site annonce « mediane 34 ms, maximum 36 » — **4x le maximum cite** | la borne est-elle intentionnelle ? Combien de kills basculent entre 36 et 150 ms ? |
| NE12 | `replay/grapple_lines.go:276` | la FIN d'une traction de grappin | le negatif « le film ne porte pas de fin de traction » **n'est pas enonce comme mesure** | existe-t-il un troisieme corps de tag sur i59 ? |
| NE13 | `objectiveevents/extract.go:114` (`classifyObjectiveMode`) et `replaybuild/zones.go:141` (`isVipVariant`) | la famille d'objectif ; publier ou non la couronne VIP | classement par `strings.Contains` sur le nom de variante | **hors axe strict** : le marqueur canonique est `GameVariantCategory = 23`, verifie sur les payloads bruts, **non porte par `port.MatchFacts`** (`zones.go:134`). Ce n'est pas le film, c'est un champ a cabler — **la plus petite correction du perimetre** |
| NE14 | `objectiveevents/extract.go:210` (`captureScorer`, +-2 000 ms) | qui a marque une capture CTF | argmax temporel | **HORS PRODUCTION** : `objectiveevents.Extract` n'a qu'UN appelant non-test, `cmd/diag_weapons_v3/process.go:37` (verifie sur pieces). A traiter comme du diagnostic, ou a supprimer (regle 7) |
| NE15 | `replay/killpos_opening.go:62` (`OpeningLeadMS = 1 500`) | la position d'ouverture d'un duel | **proxy DECLARE comme tel**, gate ecrit avant mesure (ecart median <= 2 m) | rien a instruire : la ligne est ici pour memoire, c'est une heuristique assumee et bornee |
| NE16 | `replay/successions.go:92` (`candidateIn`) | quelle piste anonyme appartient au bot remplacant | `candidateIn` **ne restreint pas** ses candidates au slot du remplacant (limite ecrite `replay/identity.go:271-275`) | combien de relais attribuent une piste d'un autre slot ? |
| NE17 | `objectiveevents/score.go:194` (`longestRun`) | quelles emissions d'un compteur sont reelles | une valeur mal lue mais PLUS GRANDE prolonge la suite au lieu de la rompre (`round_bounds.go:14-15`) | combien de points `longestRun` ecarte-t-il par film ? **pas de compteur** |

---

## 8. TABLE (E) — REPLIS ANONYMES (amendement D14) : le materiau du lot 1.9.0

Sites ou une decision de SECOURS est prise **sans etre ni nommee ni comptee**. Les replis nommes ET
comptes ne figurent pas ici (ils sont dans les lignes A/B/C). Ce tableau est le registre d'entree du
lot 1.9.0 (« registre des replis + ratchet »).

| `fichier:ligne` | Decision de secours | Declencheur | Comptee ? |
|---|---|---|---|
| `replay/build.go:580` | `Team: -1` — « equipe inconnue » sur TOUTE piste | inconditionnel | non |
| `replay/equipment_placements.go:330` | classer l'origine sur une vie qui ne CONTIENT PAS l'instant | aucune vie du poseur ne couvre `T0US` | non (les 3 origines ne distinguent pas) |
| `replay/usage_summary.go:249` / `:284` / `:286` | crediter un geste au dernier occupant du slot, puis a la premiere vie, puis au **dernier occupant du MATCH** | aucune vie ne couvre la frame | non — et **32 a 95 % des poses d'un film tombent hors de toute fenetre publiee** (153/351, 443/466, 34/105 sur 3 films, constat N-3 de REG-R2) |
| `replay/usage_summary_outcomes.go:230` | ecraser a 0 un « garde » negatif | `taken - utilise - lache < 0` | non — l'incoherence disparait sans trace |
| `replay/grapple_lines.go:190` / `:208` | rattacher la traction a la vie qui couvre le TIR, puis a la vie la plus proche | `lifeCovering` nil | non |
| `replay/ground_weapon_rules.go:402` | nommer la famille d'arme par son identifiant brut `0x%08x` | `weaponv3.WeaponName` rend "" | non |
| `replay/document_ground_weapon_items.go:337` / `:339` | abandonner le lien d'une prise | famille absente / position d'acteur absente a +-250 ms | non par cause |
| `replay/document_ability_impulses.go:238` / `:285` | fusionner une lecture dans le geste precedent ; elargir la fenetre de vie de 5 s aux deux bornes | ecart <= 1 s ; instant hors bornes strictes | non |
| `replay/held_object_carry.go:164` / `:172` | affirmer `FinParMort = false` pour un porteur anonyme ; fermer la periode a la prise suivante d'un AUTRE slot | `p.XUID == 0` ; aucune mort dans l'intervalle | non |
| `replay/inventory_decode.go:192` | plafond de grenade par defaut `DefaultGrenadeMax = 2` | `grenMax == 0` | non |
| `replay/world_object_precision.go:47` | conserver les largeurs d'axe par defaut — **qui sont celles de Cliffhanger** (`filmdec/traverse.go:164-166`) | entree de catalogue incomplete | non (`slog.Warn` seul) |
| `replay/identity_registry_pont.go:179` | servir le PREMIER occupant nomme du siege | aucune vie nommee ne couvre l'instant | non |
| `replay/published_tracks.go:63` | nommer une piste anonyme par le pont aplati | `t.XUID == ""` | non |
| `replay/identity.go:52` | apparier une piste a la vie qui la recouvre LE MIEUX | plusieurs vies nommees sur le slot | non (aucun seuil minimal de recouvrement) |
| `replay/death_context.go:171` | ECARTER entierement une mort dont la victime n'est dans aucune equipe de la base | `Equipes` ne porte pas le xuid | non — aucune ligne, aucun compteur, aucun log |
| `replay/death_context.go:222` | classer « vivant hors de vue » tout ce qui n'est ni visible ni en attente | defaut de fin de fonction | oui (`HorsDeVue`) mais indistinct de « canal non lu » |
| `replay/vehicle_tracks.go:166` / `:380` | fin de vie de vehicule a +20 s ; report du dernier cap | aucune image-cle ne cesse de recenser ; vitesse < 5 m/s | non |
| `replay/player_index.go:78` / `:83` | sauter un chunk de replication illisible ou muet | `FilmChunkAt` echoue ; resolution vide | non |
| `replay/flag_assign.go:232` / `:239` / `:250` | attribuer au socle le plus proche ; taire l'invariant « jamais son propre drapeau » ; donner `flagIndex = 0` a TOUS les portages | 3 regles muettes ; equipe inconnue ; `len(spawns) == 0` | non |
| `replay/flag_carries.go:391` | la position de LACHER prend celle de la PRISE | aucun point publie a la frame de fin | non |
| `replay/flag_carrier_tracks.go:80` | ecarter la piste en silence | le pont ne nomme pas le slot | non (distinct du refus nomme) |
| `replay/zone_states_owner.go:364` / `:431` | sans roster, toute valeur non neutre (ou <= 1) vaut un camp | `len(teams) == 0` | non |
| `replay/zone_states_hill.go:137` / `:454` | remplacer les votes des rampes vides par ceux de TOUTE la periode ; faire courir le dernier intervalle de propriete jusqu'a l'infini | aucune rampe ; dernier groupe | non |
| `replay/skull_carries.go:308` | le gate laisse passer sans rien verifier | xuid sans aucune vie nommee | non |
| `replay/bomb_armings.go:225` | `startT = 0` quand le hold commence avant la frame 0 | `frameOf(StartMS)` faux | non |
| `killsource/walk.go:195` / `:198` / `:201` / `:204` | rejeter un dead-state (hors bande, indice hors `nPlay`, categorie hors enum) ; jeter un record desynchronise (`:167`) | quatre `continue` nus | non — `hors_plage_bipede` n'existe que dans un test de recherche |
| `killsource/roster.go:87` | inventer des noms `?N` pour rendre le probleme hongrois carre | `len(names) < nPlay` | non |
| `killsource/roster.go:115` | rendre `"?"` pour un indice hors bijection | indice invalide | partiellement (assistant seulement) ; pour victime et tueur, `"?"` part en base tel quel |
| `killsource/feed.go:123` | remplacer un gamertag manquant par `xuid:<N>` | `gt[e.XUID] == ""` | non |
| `killsource/calibrate.go:109` / `:150` | exclure un paquet de l'echantillon ; ignorer un paquet non localise | taille/type ; `locateRecords < 0` | non |
| `killsource/eventbody.go:52` / `eventchain.go:158` | arreter la chaine sur un code non modelise (95 codes sur 123) ou un `cfgIdx` non resolu | `default: return false` | non |
| `killsource/chunks.go:78` | **perdre le type de chunk du manifeste** | `loadFilm` ne copie pas `src.Meta()` | non — cause racine de A7 |
| `killsource/hybrid.go:284` | « le plus proche en temps » pour une mort que personne ne revendique | plusieurs candidats dans la fenetre | non (l'arbitrage est silencieux) |
| `killsource/match.go:108` / `:148` | premier candidat gagnant pour une mort de bot / par un bot | `break` | non (l'unicite n'est pas verifiee) |
| `killsource/decode.go:165` | ne pas lancer la sonde a porte relachee | `cov.Covered >= cov.RealPairs` | non (`Probe` nil, indiscernable d'une sonde a zero) |
| `killsource/label.go:46` | publier `"Autres"` au lieu d'un nom | nom vide ou non publiable | non — 206 tags sur 468 concernes |
| `sync/killcollector/identities.go:74` | rendre un xuid VIDE pour un gamertag inconnu | `m.ParNom[nom]` absent | non — la mort est ecrite sans xuid de victime |
| `sync/killcollector/roster.go:73` | n'attribuer AUCUN xuid a deux participants homonymes | doublon de gamertag | non (`slog.Warn` seul) |
| `sync/killcollector/shots.go:138` / `:209` | jeter les DEUX xuids qui se disputent un indice ; retenir la PREMIERE occurrence sans exiger la concordance | collision ; `!deja` | non / non — **contraste avec `replay/player_index.go:44` qui refuse de publier sur desaccord** |
| `sync/killcollector/hits.go:77` / `:154` / `:160` | sauter toute la passe de precision ; desactiver les distances | `filmDir == nil` ; erreurs de detection | non (« best-effort silencieux », `slog.Debug`) |
| `sync/killcollector/positions.go:218` | « le premier vu » parmi les noms de carte qui resolvent | boucle nue | non |
| `sync/killcollector/positions.go:296` | **degradation sur le pont par morts** | `ScanBipedCreations` en erreur, **toute** erreur y compris transitoire | non — deux `slog.Warn`, aucun expvar |
| `sync/killcollector/isolation_facts.go:265` | ecrire `teammates_left = 0` sur toutes les lignes | colonne conservee apres retrait du producteur | non — constante indiscernable d'une mesure |
| `objectiveevents/statborg.go:261` / `:363` / `:637` | abandonner un enregistrement ; s'arreter au milieu de ses composants ; **DECRETER la manche 0 reelle** | `continue` muets ; `len(out) == 0` | non |
| `objectiveevents/named_series.go:111` / `:123` / `:153` | jeter une emission negative ; hors domaine ; sauter la manche entiere du slot | trois `continue` | non |
| `objectiveevents/slotidentity_deaths.go:143` / `:159` / `:203` | table d'identite VIDE ; **ignorer une mort sans xuid** ; jeter une emission du compteur de morts | `len(deaths) == 0` ; `d.XUID == ""` ; hors `[0, 1000]` | non — et `d.XUID == ""` contredit « VIES ANONYMES N'EXISTENT PAS » |
| `objectiveevents/slotidentity_rounds.go:183` / `:287` / `:341` | debut de manche = MINIMUM au lieu du consensus ; abandonner le slot au premier arrive ; faire retomber l'instant sur la PREMIERE manche | manche hors consensus ; `deja` ; tous les `startMS > timeMS` | non — **le premier a un defaut mesure : 213 s d'attribution fausse sur `24dbb67d`** |
| `objectiveevents/extract.go:136` | famille d'objectif VIDE -> aucune action nommee du match | aucun mot-cle reconnu | non |
| `replaybuild/kills.go:126` / `:169` | abandonner l'assistant ; **« en cas de divergence, le premier gagne »** | resolution echouee ; `!seen` | non / non |
| `replaybuild/replaybuild.go:463` / `:467` / `:517` | abandonner la mort neutre ; garder le repere generique ; abandonner le relais du bot | `VictimXUID == 0` ; icone absente ; `Sscanf` en erreur | non |
| `replaybuild/matchfacts.go:269` / `:456` / `:478` / `:496` | retirer le joueur de la table d'equipe, du roster publie, du tableau de participants | `TeamID < 0` ; `ParseUint` echoue (bot) ; `XUID == ""` | non |
| `replaybuild/zones.go:222` | catalogue de zones VIDE -> aucun etat de zone | `len(zones) == 0` | non |
| `replaybuild/derivations_index.go:175` | juger la fraicheur des derivations sur la TAILLE de l'artefact | `ArtifactBytes == st.Size()` | non (compromis assume) |
| `filmdec/i0_layout.go:191` | `GateBits` force a 5 et `Region` a 0, quoi que dise le film | inconditionnel | non — `I0LayoutReport.IndexBitOnes` mesure le cas et **le rapport est jete aux 3 sites** |
| `filmdec/offline_biped_band.go:186` | bande comblee sur tout `[min, max]`, jusqu'au domaine 13 bits | inconditionnel | non |
| `filmdec/traverse.go:121` / `:196` / `:207` | `param_4 = 1` pour un composant hors table ; garder les largeurs Cliffhanger ; `IndexW` a 1 | nom hors table ; `AxisW[i] == 0` ; `GateBits` trop court | non |
| `filmdec/position_capture.go:240` | largeur **uniforme 14** pour les 3 axes de tout chemin absolu i0 | `absoluteAxisW > 0`, **toujours vrai en production** | non |
| `filmdec/default_state.go:372` / `:384` | `mppLeadBits = 9`, `mppIndexBits = 5` par defaut | `replay/build_ground_weapons.go:130` rend un restaurateur vide sur `!w.Valid()` | non pour la voie SOCLES (la voie equipement, elle, WARN) |
| `filmdec/film_chunks.go:90` | `break` sur un trou de numerotation : les chunks 8..N d'un film dont le 7 manque sont ABANDONNES | `m.Index != want` | non |
| `filmdec/equipment_creation_width.go:312` | `continue` sur une vie que les paquets delta n'ont pas vue | `len(spans) == 0` | non — `cal.Anchors++` est APRES le `continue`, les ancres ecartees sont invisibles du denominateur |

**Compte** : **62 replis anonymes** recenses, dont **9 portent un defaut deja mesure**
(`usage_summary.go:249/284/286` avec ses 32-95 % ; `slotidentity_rounds.go:183` avec ses 213 s ;
`statborg.go:637` ; `chunks.go:78` ; `shots.go:209` ; `positions.go:296` ; `build.go:580` ;
`world_object_precision.go:47` ; `i0_layout.go:191`).

---

## 9. Ordre propose des conversions (A) — a recopier en tete de la famille 1.9

Ordre par gain DECROISSANT, gain mesure d'abord, gain de robustesse ensuite. `1.9.1` est fixe par
l'utilisateur et garde sa place.

| Item | Site | Fait converti | Gain | Cout |
|---|---|---|---|---|
| **1.9.1** | `replay/equipment_placements.go:313` | origine d'une pose : le MUR par l'evenement 103 (216/216 panneaux), les appareils portes restent en repli temporel COMPTE | 216 poses lues au lieu d'etre classees ; **7 defauts d'origine sur 216 panneaux** fermes | S-M |
| **1.9.2** | `sync/killcollector/positions.go:253` (+ `hits.go:157`) | le decoupage d'i0 vient du catalogue de carte, plus de l'auto-detection | **27 faux enregistrements sur 267 400** elimines sur Live Fire ; 2 passes de detection supprimees par film | S |
| **1.9.3** | `killsource/feed.go:162` | le couple (tueur, victime) lu au kill-event 85 au lieu d'etre recolle sur le voisin | **64 couples sur 372** cessent d'etre une reconstruction ; supprime la fabrication d'un couple quand la victime est un bot | M |
| **1.9.4** | `sync/killcollector/hits.go:151` | la carte du film vient du nom de match, plus d'une signature de largeurs | **6 cartes jumelles** recuperent leurs distances (3 paires mesurees) | S |
| **1.9.5** | `replay/skull_carries.go:390` | le porteur du crane lu au canal des armes tenues (famille `0x0017592c`), les tics deviennent le repli compte | non chiffre ; supprime une inference la ou un canal PORTE existe et n'est cable que pour la bombe | S |
| **1.9.6** | `replay/flag_carries_lives.go:267` | le drapeau qui rentre pris dans `ev.flag`, deja nomme en amont | les `ambiguousReturns` (compteur publie) | S |
| **1.9.7** | `killsource/options.go:114` (sites `match.go:30`, `:45`, `:103`, `:143`) | dead-state et kill-feed apparies par l'identite de paquet `(chunk, pidx)`, la fenetre de 2,5 s devient le repli compte | non chiffre ; retire la seule constante du paquet justifiee par la comparabilite et non par une mesure | M |
| **1.9.8** | `killsource/feed.go:82` | le chunk du pied pris au type du manifeste, l'argmax devient le repli compte | robustesse ; aucun defaut mesure a ce jour | S |

**1.9.0 — prealable, propose** : le registre des replis. Nommer et COMPTER les 62 replis anonymes de
la table (E), poser le ratchet (« aucun repli nouveau sans nom ni compteur »), et publier la
ventilation `coverage.<fait>.{grammaire, repli, contradiction}` que la famille 1.9 exige de chaque
lot. Sans lui, le critere de retrait D14 (« compte de repli a 0 sur le corpus gate ») n'est pas
mesurable pour la majorite des lignes de ce registre.

**Hors famille 1.9, a router** :

- `replay/projectiles.go:106` (A8) — **6,0 % des trajectoires du parc** (947 sur 15 735). Ce n'est
  pas une conversion mais une CAUSE a corriger dans la dequantification de `filmdec` (lot B-bis,
  `.ai/RAPPORT_LOT_B_DECODEUR_FORK_2026-09-11.md`) ; le garde-fou devient supprimable apres.
- `replaybuild/zones.go:141` (NE13) — **la plus petite correction du perimetre** : porter
  `GameVariantCategory` dans `port.MatchFacts`. Aucun decodage.
- `replay/identity_registry_section.go:288` — `methodeStatborg` ne traduit pas `OriginRoundResidue` :
  la voie la plus fragile de B2 est publiee comme `MethodNone`. **Decouverte, non traitee** (regle 7).

## 10. Verification adverse — ce que l'auditeur principal a re-ouvert lui-meme

Quinze constats porteurs ont ete re-verifies ligne par ligne, hors du contexte de l'auditeur qui les
a produits. **Aucun n'est tombe** ; deux ont ete AMENDES et un TROU a ete trouve en plus.

| Constat | Verdict | Ce que la re-lecture a ajoute |
|---|---|---|
| `filmdec/traverse.go:515-517` lit 4 bits d'equipe et les jette | **tient** | la grammaire est juste, seule la publication manque ; le `case` est dans la traversee DELTA, et i0 precede le bloquant i4 de l'image-cle — donc atteignable aussi en image-cle |
| `replay/document.go:590` « L'EQUIPE N'EST PAS DANS LE FILM » | **tient** (doc inversee) | **et une SECONDE dans le meme fichier** : `document.go:594` dit « le film ne porte aucun gamertag » quand `replay/lives.go:54` documente le gamertag « porte PAR LE FILM lui-meme » |
| `killsource/calibrate.go:93-95` repli 14/1 | **tient** | le repli est nomme (`Flat`) mais `Result.Calibration` n'a aucun consommateur : il est **silencieux en production** |
| `killsource/decode.go:128` appelle `calibrate` sans entree de carte | **tient** | `killsource.Options` ne porte aucune borne de carte (lu en entier) alors que `killcollector` tient le catalogue (`capture.go:36`) |
| `sync/killcollector/positions.go:253` n'passe pas `Layout` | **tient** | `entry` est le PARAMETRE de la fonction et `entry.Range()` est lu a la ligne suivante (`:254`) ; `MapQuantEntry.Layout()` existe (`map_bounds.go:69`) |
| `replay/build_from_film.go:87` impose deja le catalogue | **tient** | et le commentaire `filmdec/film_context.go:33-42` chiffre le defaut (27/267 400 sur Live Fire) |
| `killsource/feed.go:82` argmax contre type de manifeste | **tient, AMENDE** | `filmsource.Film.Meta()` porte bien `ChunkType` (`film.go:41`) et `objectiveevents/extract.go:144` l'utilise deja — **mais le manifeste est un fichier EXTERNE** ; la conversion garde l'argmax en repli compte |
| `killsource/eventchain.go:242` le kill-event porte victime ET tueur | **tient** | grammaire lue en entier ; `killEventPlausible` exige `killer != victim` |
| `replay/BuildHeldObjectCarry` n'a qu'un appelant | **tient** | `bomb_carries.go:142`, et la famille du crane `0x0017592c` est documentee dans le meme fichier |
| `replay/usage_summary.go:286` rend le dernier occupant du MATCH | **tient** | le commentaire du site porte la mesure verbatim : « 32 a 95 % des poses d'un film tombent hors de toute fenetre publiee » |
| `objectiveevents/statborg.go:637` `out[0] = true` | **tient** | repli terminal inconditionnel, non compte |
| `objectiveevents/slotidentity_rounds.go:183` minimum au lieu du consensus | **tient** | le commentaire du site chiffre le defaut que le repli reintroduit : 213 s d'attribution fausse sur `24dbb67d` |
| `objectiveevents/rosterfit.go:72` garde d'effectif | **tient, AMENDE** | `n <= 0` rend **VRAI** : une cuisson sans faits de match publie le calque SANS garde |
| `replay/zone_states_owner.go:346-349` oracle exterieur au film | **tient** | l'aveu est verbatim dans le code |
| `replay/identity_registry_section.go:288` `methodeStatborg` | **TROU TROUVE** | `OriginRoundResidue` n'est traduit par aucune methode canonique : le lien part en `MethodNone` (« le lien n'a jamais eu de candidat »). Le compteur de retrait de B2 ne couvre donc **pas** la voie la plus fragile, et le document servi affirme l'absence d'un lien qui existe |

Un constat a ete **ECARTE** en re-lecture : « le pont APLATI (`PontParSlot`) credite les frags au
premier occupant d'un siege » (entree ouverte de `.ai/V7.5/REGISTRE_REPORTS.md`). Sur pieces, le
pont aplati **n'a plus d'accesseur** depuis le lot 6.1 du 2026-09-10
(`replay/identity_registry.go:185`) et `sync/killcollector/positions.go:46` documente son retrait.
L'entree du registre des reports est perimee.

## 11. Ce que l'audit n'a PAS couvert

1. **Le web.** Aucun fichier de `apps/web/` n'a ete ouvert. Une heuristique cote client (par exemple
   l'appariement du fil des morts par `killFeedLogic.ts` quand l'origine n'est pas publiee) sort du
   perimetre.
2. **La grammaire de `filmdec`** — largeurs de composants, deserialiseurs, etats par defaut. Hors axe
   par le brief. Seules ses fonctions d'INFERENCE sont entrees.
3. **Les autres paquets d'`analysis/`** (`sessionusage`, `narrative`, `temporal`, `breakdown`) : une
   heuristique qui agrege un fait deja cuit n'est pas dans l'axe, mais elle peut en propager un.
4. **`internal/himap` et les catalogues** : la production des bornes de carte et des geometries n'a
   pas ete auditee ; A1, A2 et C3 la supposent juste.
5. **Aucune mesure n'a ete refaite.** Tous les chiffres de ce registre sont recopies de leur source
   (commentaire de code date, rapport, note de RE) et cites avec elle. L'audit n'a decode aucun film,
   ouvert aucune base, execute aucun test.
6. **Les lignes « non etabli » ne sont pas des constats negatifs** : elles disent qu'aucune source du
   depot ne tranche, pas que le film se tait.
7. **Le cout et le gain** ne sont chiffres que la ou une source les chiffre. « Non chiffre » est
   ecrit tel quel plutot que devine — 31 lignes sur 79 sont dans ce cas.

## 12. Gate de l'audit — controle rejouable

Chaque `fichier:ligne` du registre existe : script de controle, joint au rapport.

Le registre ecrit les chemins en forme courte (`replay/...`, `killsource/...`, `filmdec/...`,
`objectiveevents/...`) ; le script les re-prefixe avant de verifier.

```bash
cd apps/go-api
R=../../.ai/V7.5/AUDIT_HEURISTIQUES_DECODEUR_2026-09-13.md
grep -ohE '(internal/[A-Za-z0-9_/.-]+\.go|(replay|killsource|filmdec|objectiveevents|replaybuild|sync/killcollector|analysis/filmsource|games/canonical)/[A-Za-z0-9_/.-]+\.go):[0-9]+' \
  "$R" | sort -u | while IFS=: read -r f l; do
    case "$f" in
      internal/*)                        p="$f" ;;
      replay/*|killsource/*|filmdec/*)   p="internal/games/halo_infinite/film/$f" ;;
      objectiveevents/*)                 p="internal/analysis/$f" ;;
      *)                                 p="internal/$f" ;;
    esac
    if [ ! -f "$p" ] || [ "$(wc -l < "$p")" -lt "$l" ]; then echo "MANQUE $f:$l -> $p"; fi
  done
echo "controle termine"
```

**Resultat au 2026-09-13** : **241 references distinctes, 241 resolues, 0 manquante**. Les
53 references de tete ont en outre ete verifiees une a une avec le CONTENU de la ligne
(section 10).

**`git diff --stat` attendu** : ce fichier et `.ai/PLAN_DECODEUR_FILM_2026-09-13.md`. **Aucun fichier
de production.**

## 13. Comptes de cloture

| Grandeur | Valeur |
|---|---|
| Sites de decision LUS (fonction entiere + appelant) | **528** |
| Fichiers de production ouverts | **201** (recouvrements compris) |
| Lignes de registre, table (A) — conversion courte | **8** |
| Lignes de registre, table (B) — lecteur a porter | **8** (couvrant 40 sites de decision) |
| Lignes de registre, table (C) — negatif mesure | **38** |
| Lignes de registre, table (D) — non etabli | **17** |
| Lignes de registre, table (E) — replis ANONYMES | **62** |
| **Total de lignes de registre** | **133** |
| Replis dont le critere de retrait est deja mesurable (compteur publie) | **19** |
| Replis sans aucun compteur | **62** (table E) plus 11 replis nommes mais non comptes |
| Constats re-verifies sur pieces par l'auditeur principal | **15** (0 tombe, 2 amendes, 1 trou trouve en plus) |
| Constats ecartes en re-lecture | **1** (entree perimee du registre des reports) |



