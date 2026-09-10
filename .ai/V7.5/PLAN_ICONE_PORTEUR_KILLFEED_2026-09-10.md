# PLAN — Genre `PORTEUR` : rendre au kill feed les icones de chassis que le film nomme deja

> Ecrit le 2026-09-10. Contrat d execution : skill `plan-execution` (ordre strict, statut par
> item, zero fix hors perimetre). Ce fichier est la seule source d avancement.

## 0. Le defaut, en une phrase

`labels.tsv` porte DEUX identifiants par ligne de classe VEHICULE — la racine de banque sonore
(souvent ambigue) et le tag `vehi` du porteur (unique) — et `rules.tsv` ne s indexe que sur le
premier. Consequence mesurable aujourd hui : un frag a la mitrailleuse du Warthog affiche
`killfeed-05` (tourelle UNSC generique) au lieu du Warthog, et un frag au canon gauss n affiche
rien. Les vignettes existent sur disque depuis l extraction.

Ce n est pas une limite de la donnee : le jeu, en mode Theater, distingue parfaitement. Et ce
n est plus une limite de notre connaissance depuis le 2026-09-02.

### Les pieces

| fait | source |
|---|---|
| `vehi 0xdd7f9102` = Warthog MITRAILLEUSE / LAAG (weap `0x0c6fd911`) | `film_re/WARTHOG_FINAL_2026-09-02.md` §1 |
| `vehi 0x64b925eb` = Warthog CANON GAUSS (weap `0x8647925a`) | idem |
| `vehi 0xbcfb852f` = Warthog LANCE-ROQUETTES | idem (deja servi via `veh_un_rockethog`) |
| `vehi 0x4ccc20e6` = `warthog_b_g` / `warthog_g`, confiance **moyenne** | `film_re/REWORK_WARTHOG_GUNGOOSE_2026-09-01.md` L47 |
| jpt `382cafaf` -> `vehi dd7f9102`, racine unique `tur_un_machinegun` | `damagetag/data/labels.tsv` L166 |
| jpt `4844f822`, `638085d4` -> `vehi 4ccc20e6`, racine unique | idem L193, L237 |
| jpt `2dc9eeae`, `4c64a26a` -> `vehi 64b925eb`, racine unique `tur_un_gausscannon` | idem L150, L199 |
| vignettes `killfeed-26` (warthog) presentes sur disque, nommees par la passe humaine | `icones/NOMMAGE_GATE_2026-08-09.tsv` L65 |

Cause de la peremption : `labels.tsv` est date du 2026-07-26, `rules.tsv` du 2026-08-27, et le
chantier vehicules qui a nomme les `vehi` a conclu les 1er et 2 septembre. Personne n est
repasse. La regle de surete d aout (« une racine ambigue n obtient rien ») etait juste avec ce
qu on savait alors ; elle est devenue trop prudente.

## 1. Perimetre — FERME

DANS le perimetre :
- un genre de regle `PORTEUR` dans `film/killicon`, cle = tag `vehi`, priorite au-dessus de
  `BANQUE` ;
- les regles `PORTEUR` pour les `vehi` Warthog PROUVES (liste a l etape 3, close) ;
- l entree de registre d arme que ces regles exigent, et son libelle FR/EN ;
- les garde-rails du nouveau genre ;
- la mise a jour des en-tetes des deux tables (dates, compte de regles).

HORS perimetre, explicitement — a NE PAS traiter meme si l occasion se presente :
- le Gungoose (aucun `vehi` d arme n existe : `film_re/CONTACT_ARMES_GUNGOOSE_2026-09-02.md` §2 —
  c est un `scen 0x004164ed` accroche au Mongoose ; chantier distinct) ;
- le Mongoose, le Chopper, le Phantom, le Pelican, les tourelles fixes ;
- les 10 lignes portees par `vehi b857fb95` et `003f00c7` (chassis de tourelle generique
  `turret_g`) : elles citent trois racines ET leur `vehi` n est identifie par aucun rapport.
  Elles restent sans icone, et le garde-rail existant
  `TestUnEffetPartageParPlusieursChassisNObtientAucuneIcone` doit rester VERT sans modification ;
- les citations « Artilleur de » (travail en cours non commite sur la branche courante) ;
- toute reecriture de `labels.tsv` a partir des fichiers du jeu.

## 2. Decisions tranchees AVANT execution

| # | question | decision | pourquoi |
|---|---|---|---|
| D1 | Le Hog mitrailleuse doit-il montrer le chassis ou la tourelle ? | **le chassis** (`killfeed-26`) | c est ce que le jeu affiche en Theater (constat utilisateur, 2026-09-10) |
| D2 | Le `vehi` doit-il devenir une vraie colonne de `labels.tsv` ? | **non** — extraction par regex sur `detail`, comme `GenreBanque` et `GenreGGGL` le font deja | ajouter une colonne imposerait de regenerer la table depuis le jeu (tag `gamefiles`, hors perimetre). La forme regex est le precedent etabli du paquet |
| D3 | weapon_key des regles `PORTEUR` Warthog | **`hinf_warthog`**, a creer | sans elle, la source perdrait sa ligne de statistiques : elle resout aujourd hui `hinf_turret_machinegun` via `BANQUE`. Une regle sans weapon_key REGRESSERAIT le sunburst « arme par kill » |
| D4 | libelle de `hinf_warthog` | `en = "Warthog"`, `fr = "Warthog"` | nom propre non traduit dans la VF du jeu ; meme convention que `hinf_rockethog` = « Warthog lance-roquettes » |
| D5 | `vehi 0x4ccc20e6` (confiance moyenne) | **inclus SI et seulement SI** l etape 3 le corrobore par une seconde source ; sinon `[!]` justifie | regle du depot : deux sources independantes avant d ecrire une regle d icone |
| D6 | canon gauss (`vehi 64b925eb`) | **regle ecrite SI et seulement SI** l atlas porte une vignette de gauss (etape 2) ; sinon `[!]` | `rules.tsv` affirme aujourd hui « aucune vignette de canon gauss » ; `killfeed-25.png` existe sur disque mais n est PAS nomme par la passe humaine. A verifier a l oeil, pas au raisonnement |
| D7 | affaiblir un garde-rail pour faire passer | **interdit** | les regles `PORTEUR` ne portent que sur des lignes a racine UNIQUE : aucun garde-rail existant n a besoin de bouger. Si l un rougit, c est le plan qui a tort |

## 1bis. AMENDEMENT DU PERIMETRE — le Gungoose rentre (2026-09-10, decision utilisateur)

Le perimetre initial excluait le Gungoose sur une premisse ECRITE ICI ET FAUSSE : « aucun
`jpt!` de labels.tsv ne pointe dessus ». La mesure demandee par l utilisateur l a refutee.

**Ce qui est vrai** : `labels.tsv:88` porte `00426796 VEHICULE VALIDE (nom vide) ARME DE
VEHICULE (vehi 000025aa, classe fixed) [effet proj 00426793 #0/3]`, et **62 morts** au corpus
(vue `_latest` ; 387 en lecture brute — voir la correction a l etape 0.3).
Le tag est nomme depuis longtemps par `RE_LOG_KILLWEAPON.md` : « 00426796 = projectile de
GUNGOOSE (vehicule Mongoose arme d un lanceur) », resolu par `vcdd -> sofd -> sofa -> uwfa ->
weap 0042678e`, et le journal l appelle « l ancre Gungoose ».

**D ou venait l erreur** : j ai confondu « pas de racine de banque » avec « pas de donnee ». Le
Gungoose est en fait le cas le PLUS PUR du defaut repare ici — il n a aucune banque du tout
(l un des deux seuls tags du catalogue dans ce cas), donc la voie BANQUE ne pouvait rien pour
lui, jamais. Le meme RE_LOG le disait deja, ligne 18226 : « c est la voie de nommage a etendre
aux 9 tags sans banque du catalogue, et elle n a pas ete instruite ici ». Le genre PORTEUR EST
cette voie.

**Le compromis, assume et garde** : la cle est le chassis MONGOOSE (`000025aa`), partage avec
le Mongoose nu — le Gungoose n a pas de `vehi` propre, c est un Mongoose avec une arme
accrochee. La regle n est juste que tant que tout ce que porte ce chassis vient de cette arme :
vrai aujourd hui (deux lignes, deux projectiles ; le Mongoose nu ne porte ni `uwfa` ni `scen`).
`TestChassisMongooseNePorteQueDesTagsGungoose` rougit au troisieme tag.

**Restent hors perimetre** : les 8 autres tags sans banque du catalogue, le gauss (0 kill), les
lignes `turret_g`, `d2ffec3f` (deux porteurs, rejete par la garde `+N` — elle sert deja).

## 2bis. Journal d execution (2026-09-10)

| etape | statut | gate |
|---|---|---|
| 0 mise en place | CLOSE | worktree `wt/killfeed-porteur` ; volume mesure (1002 vs 70) |
| 1 genre PORTEUR | CLOSE | `go test ./internal/games/halo_infinite/film/killicon/...` vert a table inchangee |
| 2 atlas a l oeil | CLOSE | vignette 26 = Hog quatre roues + tourelle arriere, distincte du 27 (rockethog) et des quads 28/29, lue sur `planche_killfeed.png`. 2.3 (gauss) `[!]` : sans objet |
| 3 corroboration | CLOSE | `dd7f9102` : QUATRE sources concordantes, toutes mode `0xc0803caa`. Ligne `labels.tsv:166` rouverte : porteur unique, pas de `+N`, racine unique. 3.2 et 3.3 `[!]` : 0 kill |
| 4 registre | CLOSE | `go test ./internal/games/weapons/...` vert apres mise a jour des cardinalites |
| 5 regles + garde-rails | CLOSE | DEUX regles (Warthog, puis Gungoose apres amendement) ; `./internal/games/...` vert ; `go vet` propre ; `gofmt` propre |
| 6 a l ecran | OUVERTE | — |
| 7 livraison | OUVERTE | — |

**Les TROIS garde-rails ont ete verifies PAR TEST NEGATIF**, pas seulement constates verts.
Chaque fois, le code a ete restaure immediatement apres :
- garde `+N` retiree -> `TestPorteurRefuseUnePluraliteDeclaree` rouge, message nomme ;
- priorite PORTEUR retiree -> `TestPorteurPrimeBanque` rouge en affichant litteralement
  l ancien defaut : `Warthog 382cafaf : "killfeed-05" par "BANQUE"` ;
- une entree retiree de `tagsGungooseConnus` -> `TestChassisMongooseNePorteQueDesTagsGungoose`
  rouge en nommant le tag intrus et en disant quoi faire.

**CE QUE LE COMPTEUR GLOBAL A CONFIRME, et pourquoi il ne suffisait pas.**
`TestCouvertureParClasse` fige les icones de classe VEHICULE : il est passe de 46 a 48. Le +2
vient ENTIEREMENT du Gungoose. Le Warthog n y apparait pas : son tag avait deja une icone — la
mauvaise. Un compte global ne voit pas un echange. C est exactement l argument de la section
5bis, verifie par l outil lui-meme.

## 3. Etapes

Regle d ordre : l etape N est close avant que N+1 commence. « Close » = tous ses items portent un
statut (`[x]` fait / `[~]` couvert ailleurs + reference / `[!]` non traite + justification ecrite)
ET son gate est passe avec la commande exacte indiquee.

### Etape 0 — Mise en place (gate : `git status` propre, branche correcte)

- [!] 0.1 Statuer sur le travail en cours non commite. **BLOQUE le 2026-09-10 16h28** : une
      AUTRE session travaille dans ce worktree EN CE MOMENT. HEAD a avance pendant la
      conversation (`24ded422b` -> `80c8e0af6`), `seed_citation_data.go` a ete touche a 16h15,
      et l index porte des suppressions preparees par quelqu un d autre (deux SVG de citation
      supprimes, un PNG ajoute). Un commit « WIP » ramasserait leur travail en vol.
      Decision reportee a l utilisateur. **Jamais `git stash`** (pile partagee).
- [x] 0.1bis RESOLU AUTREMENT (decision utilisateur, 2026-09-10) : worktree DEDIE plutot que
      commit dans l arbre partage. Le WIP citations reste intact la ou il est, sous la garde de
      l autre session ; il est deja documente au thought_log (lot citations, « aucun commit
      demande a ce stade »), donc rien n est en risque de perte. 0.1 est classe `[!]` a dessein :
      il n a pas ete fait, il a ete rendu inutile.
- [x] 0.2 Worktree `LevelUp-wt-killfeed-porteur`, branche `wt/killfeed-porteur` depuis
      `feat/v75` (24ded422b). Convention du depot (`LevelUp-wt-<slug>` / `wt/<slug>`) respectee.
      Le nom `fix/killfeed-icone-porteur-warthog` du plan initial est abandonne au profit de la
      convention worktree.
- [x] 0.3 Volume mesure le 2026-09-10 (`go run ./cmd/diag_q` en lecture seule sur
      `shared_matches_v2.duckdb`, 1 213 763 kills decodes, 168 tags distincts) :

      CHIFFRES CORRIGES LE 2026-09-10 : la premiere mesure lisait la table BRUTE
      `match_kill_events` (1 213 763 lignes, cinq revisions de decodeur qui coexistent, la meme
      mort comptee plusieurs fois). La lecture juste est la vue `match_kill_events_latest` —
      138 807 morts, 1384 matchs. Effectifs divises par ~6, RAPPORTS INCHANGES, donc aucune
      decision du lot n est touchee. Les valeurs brutes d origine sont conservees entre
      parentheses pour que la correction soit lisible.

      | jpt | porteur | ce qu il est | morts (`_latest`) |
      |---|---|---|---|
      | `382cafaf` | `vehi dd7f9102` | Warthog MITRAILLEUSE | **173** (brut : 1002) |
      | `00015cd1` | `vehi 0000d500` | tourelle FIXE de carte | 11 (brut : 70) |
      | `4844f822` | `vehi 4ccc20e6` | Warthog (confiance moyenne) | **0** |
      | `638085d4` | `vehi 4ccc20e6` | idem | **0** |
      | `2dc9eeae` | `vehi 64b925eb` | Warthog GAUSS | **0** |
      | `4c64a26a` | `vehi 64b925eb` | idem | **0** |

      LECTURE : les deux premiers rendent AUJOURD HUI la meme vignette `killfeed-05`
      (« Tourelle mitrailleuse »). 184 morts partagent donc une icone, et **94 % d entre elles
      sont des frags au Warthog etiquetes en tourelle fixe**. C est le defaut, chiffre.

      CONSEQUENCE SUR LE PERIMETRE — D5 et D6 sont tranches par la mesure, pas par l opinion :
      le gauss et `4ccc20e6` ne sont JAMAIS apparus dans le corpus. Une regle pour eux serait
      invérifiable a l ecran (etape 6 sans temoin) et non couverte par la mesure. Ils passent
      `[!]`. **Le lot se reduit a UNE regle.** Les etapes 2 et 3 retrecissent en consequence.

Gate : PASSE — `git branch --show-current` = `wt/killfeed-porteur` ; `git status` ne porte que
le plan (non suivi) ; aucune modification heritee de l arbre partage.

### Etape 1 — Le genre `PORTEUR` (gate : tests du paquet verts, table inchangee)

- [ ] 1.1 `killicon.go` : ajouter `GenrePorteur Genre = "PORTEUR"` avec son commentaire de
      doctrine (ce qu il resout, pourquoi il prime `BANQUE`).
- [ ] 1.2 `porteurRe` : `regexp.MustCompile(\`vehi ([0-9a-f]{8})\`)` + `uniquePorteur` bati sur
      `racineUnique` (helper existant — ne pas en ecrire un second).
- [ ] 1.3 GARDE D UNICITE : une ligne dont le detail porte `vehi XXXXXXXX +N` declare PLUSIEURS
      porteurs et ne doit rien obtenir. Verifier que `racineUnique` seul ne suffit PAS a
      l attraper (le `+N` ne cree pas de seconde capture) et ajouter le rejet explicite.
      **C est le point de rupture du plan** : sans lui, une regle `PORTEUR` deborderait sur un
      tag partage.
- [ ] 1.4 `resolve()` : brancher `PORTEUR` dans l ordre NOM > GGGL > **PORTEUR** > BANQUE > CLASSE.
- [ ] 1.5 `validate()` : accepter le nouveau genre ; exiger que la cle soit 8 chiffres hexa
      minuscules (une cle mal formee est un silence, pas une erreur — donc elle doit rougir).
- [ ] 1.6 Mettre a jour l en-tete de doctrine de `killicon.go` et le bloc « QUATRE GENRES » de
      `rules.tsv` (qui en annoncera cinq).

Gate : `cd apps/go-api && go test ./internal/games/halo_infinite/film/killicon/...` vert, AVANT
d avoir ajoute la moindre ligne de donnee. Aucune icone ne doit avoir bouge a cette etape :
le compte `provided.IconedTags` est identique.

### Etape 2 — Verifier l atlas a l oeil (gate : verdict ecrit sur chaque vignette)

- [ ] 2.1 Teinter et regarder `killfeed-25.png` et `killfeed-26.png` (masques blancs sur alpha :
      les lire tels quels ne montre rien). Planche de reference : `icones/planche_killfeed.png`.
- [ ] 2.2 Verdict ecrit : `killfeed-26` est-elle bien un Warthog a mitrailleuse ?
- [!] 2.3 `killfeed-25` / canon gauss — SANS OBJET depuis 0.3 : le gauss n a 0 kill au corpus.
      D6 est tranche par la mesure (pas de regle). Rouvrir le jour ou un frag au gauss apparait.

Gate : les deux verdicts sont ecrits dans ce fichier, avec ce qui a ete vu.

### Etape 3 — Corroborer chaque `vehi` avant de l ecrire (gate : tableau de preuves)

Pour CHACUN des trois candidats, exiger deux sources concordantes :

- [ ] 3.1 `0xdd7f9102` — WARTHOG_FINAL §1 (mitrailleuse/LAAG) + `ASSEMBLAGE_ENFANTS_2026-09-01`
      L88 (`warthog_b_g`). Attendu : CORROBORE.
- [!] 3.2 `0x64b925eb` (gauss) — SANS OBJET depuis 0.3 : 0 kill au corpus.
- [!] 3.3 `0x4ccc20e6` — SANS OBJET depuis 0.3 : 0 kill au corpus, et confiance « moyenne » au
      rapport source. Deux raisons de ne pas ecrire la regle ; une seule aurait suffi.
- [ ] 3.4 Verifier sur piece que chacun de ces `vehi` apparait dans `labels.tsv` avec une racine
      UNIQUE et sans `+N` (rouvrir les lignes, elles ont pu bouger).

Gate : un tableau `vehi | source 1 | source 2 | verdict` ecrit ici, une ligne par candidat.

### Etape 4 — Le registre d arme (gate : suite `games/weapons` verte)

- [ ] 4.1 `weapons/registry.go` : entree `hinf_warthog`, titre HINF, nom EN « Warthog »,
      classe `vehicle`, faction humaine — a poser a cote de `hinf_rockethog`.
- [ ] 4.2 `config/titles/halo_infinite/mappings/weapon_names.toml` : `hinf_warthog = { en, fr }`
      selon D4.
- [ ] 4.3 `weapons/off_arsenal_guard_test.go` : ajouter `hinf_warthog` a `horsArsenalHINF` ET a
      la table de classes attendues. Justification a ecrire dans le diff : le chassis n emet
      aucun record de degat `0xd2`, donc il reste invisible de la voie `weapon_kills` — meme
      propriete que `hinf_rockethog`.

Gate : `cd apps/go-api && go test ./internal/games/weapons/...` vert.

### Etape 5 — Les regles de donnees (gate : suite killicon + compte d en-tete)

- [ ] 5.1 Ecrire les lignes `PORTEUR` retenues aux etapes 2 et 3, chacune avec sa justification
      citant ses deux sources (`validate()` refuse une justification vide).
- [ ] 5.2 Mettre a jour l en-tete de `rules.tsv` : `date=` et `regles=` (garde-rail
      `TestEnteteAnnonceLeBonNombreDeRegles`).
- [ ] 5.3 Mettre a jour la section « RACINES DE BANQUE PRESENTES MAIS SANS REGLE » : la note
      `tur_un_gausscannon aucune vignette` devient fausse ou reste vraie selon D6 — dans les
      deux cas elle se met a jour dans CE commit (anti-pattern « doc inversee »).
- [ ] 5.4 Nouveau garde-rail `TestChaqueReglePorteurTrouveSonVehi` : chaque cle `PORTEUR` doit
      apparaitre dans `labels.tsv` sur une ligne a porteur unique, et son `weapon_key` doit
      exister au registre. Sans lui, la peremption de ce genre serait silencieuse — c est
      exactement ce que le present plan repare.
- [ ] 5.5 Nouveau garde-rail `TestPorteurPrimeBanque` : prouver sur le cas `dd7f9102` que la
      resolution rend bien la vignette du chassis et non celle de la tourelle. Un test qui
      constate la priorite, pas qui la re-implemente.

Gate : `cd apps/go-api && go test ./internal/games/halo_infinite/... && go vet ./...` vert, et
`provided.IconedTags` a augmente du nombre exact de tags attendus (l ecrire ici).

### Etape 6 — Voir a l ecran (gate : capture)

- [ ] 6.1 `make dev`, ouvrir un rejeu contenant un des tags mesures a l etape 0.3, verifier que
      la ligne du fil porte l icone du Warthog.
- [ ] 6.2 Verifier la teinte d equipe (les PNG sont des masques blancs portes par l alpha : une
      vignette qui sort blanche est un bug de teinture, pas de table).
- [ ] 6.3 Verifier qu aucune ligne qui portait deja une icone n en a change autrement que prevu.

Gate : capture jointe, ou justification `[!]` si aucun rejeu du corpus local ne contient le cas.

### Etape 7 — Livraison

- [ ] 7.1 Skill `delivery-checklist` deroule.
- [ ] 7.2 Entree `.ai/thought_log.md` (date, titre, statut, decision technique, resultats, suite).
- [ ] 7.3 Revue adversariale du diff (skill `adversarial-review`) — table de donnees versionnee
      + garde-rails : lot a risque.
- [ ] 7.4 `make gate-push`, puis push. **Ne pas merger vers `main` sans l utilisateur** (push
      main = deploiement prod automatique).

## 4. Ce que ce plan NE repare pas, et qui doit le savoir

- Le Gungoose reste sans icone. Sa chaine est identifiee (`weap 0x0042678e`, banque
  `veh_un_wargoose`) mais aucun `jpt!` de `labels.tsv` ne pointe dessus : la table est anterieure
  a la decouverte. Reparable, par un autre bout, et il faudrait d abord une mort mesuree au
  Gungoose pour valider quoi que ce soit.
- Les 10 lignes `turret_g` restent sans icone.
- L existence de `hinf_warthog` au registre rendra techniquement possible une citation
  « Artilleur de Warthog » — l image est en stock. **Ce n est pas une invitation** : la decision
  du 2026-09-10 de l ecarter portait sur le libelle, et elle n est pas revue ici.

## 5. Decouvertes (a remplir en cours d execution, a NE PAS traiter)

- `TestWeaponRegistry_SeedCardinalities` (`weapons/registry_test.go`) fige le NOMBRE d entrees
  du registre. Ajouter `hinf_warthog` l a fait rougir : 105/50 au lieu de 104/49. Compteurs mis
  a jour avec justification datee — c est le comportement voulu du garde-rail (il force l acte
  delibere), pas un affaiblissement. Aucun autre garde-rail n a bouge.
- **`TestChaqueRegleTrouveSaSource` ne prouve pas ce qu il semble prouver.** Il verifie qu une
  racine de regle `BANQUE` est CITEE quelque part dans `labels.tsv` — pas qu elle y est citee
  SEULE. Or la resolution, elle, exige l unicite. Une regle peut donc etre VERTE et resoudre
  ZERO tag. Cas avere : `BANQUE veh_un_scorpion -> killfeed-31` est inerte, ses six lignes
  citant toutes plusieurs racines. Meme famille de defaut silencieux que celui repare ici.
  NON TRAITE (hors perimetre, decouvert en rendant service a la session citations).
- L index 25 de l atlas kill feed n est nomme par PERSONNE (seul trou de la serie 0..87 dans
  `NOMMAGE_GATE_2026-08-09.tsv`). Sur la planche il ressemble a un chassis compact vu de dessus,
  pas a un Hog a canon long. Non traite : hors perimetre, et sans consequence tant que le gauss
  a 0 kill.

## 5bis. Ecart assume au plan

- Le gate de l etape 1 demandait de constater que `provided.IconedTags` etait INCHANGE. Il n a
  pas ete releve en chiffre : avec zero regle PORTEUR en table, la branche ajoutee dans
  `matchRule` est inatteignable, donc la table est identique par construction. Le controle a ete
  remplace par quelque chose de STRICTEMENT PLUS FORT a l etape 5 : `TestPorteurPrimeBanque`
  epingle tag par tag ce qui change (`382cafaf` bascule) ET ce qui ne doit PAS changer
  (`00015cd1` reste sur la vignette generique). Un compte global n aurait pas vu un echange.

## 6. Reprise de session

Lire ce fichier de haut en bas : le premier item sans statut est le point de reprise. Verifier
`git log --oneline -10` sur `fix/killfeed-icone-porteur-warthog` pour savoir ce qui est deja
commite. Ne jamais recommencer une etape close.
