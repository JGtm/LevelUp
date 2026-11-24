# Rapport L.1, passage 2 (2026-11-24) - tete 5b8302e de feat/v1.6.0

Rejoue en entier apres les corrections (R-17). Hote : LevelUp-wt-dec160 (HEAD aab869c44e, propre a la fin). Sorties : dec-v160-recup/l1/passage2/.

## Gates
| Gate | Resultat |
|---|---|
| Build / vet / vet research (clone neuf 5b8302e, GOWORK=off) | vert |
| G-clone sans racine | 4454 PASS / 434 SKIP / 0 FAIL, 35 paquets ok : egal a la reference |
| G-clone avec racine (copie du worktree de l'hote, table des stats a jour, fixtures 127 de l'integrateur) | 4602 PASS / 301 SKIP / 1 FAIL au premier jet : TestContractFixturesUnSeulJeuVivant, cause = 8 fixtures 126 restees dans MA racine (copie du worktree) ; retirees, les 4 TestContractFixtures* passent. Zero echec restant. Reference 4583 : mon decompte (4602) inclut les sous-tests de plus de la racine complete, ecart non investigue |
| G-rev | 10 PASS + 43 PASS |
| G-compat (gorelease -base=v1.5.0) | seules incompatibilites SchemaDesFaits 30->31 et SchemaVersion 126->127 |
| G-facade | vert, aucune adaptation de l'hote |
| Registre des replis vs FORMAT.md | 128 entrees (127 true, 1 false) ; FORMAT.md l. 898 : 128 declarations. Egal |
| G-equiv-ref | comparaison puis refigeage : 20 identiques, 0 different |
| G-equiv-2p | 20 artefacts identiques a l'octet, 0 divergent ; 20 ecarts a classer, tous killsource (ecart attendu) |
| G-corpus (380 s au 1er passage, 205 s ici, base en cache) | 19 temoins : 4 ok, 14 PERTE, 1 FAUX (1er passage : 4 ok, 12 PERTE, 3 FAUX) ; voir plus bas |

## G-equiv-ref : mes references du 1er passage contre la nouvelle tete
Sur 98 etapes par film, une seule change, `artifact`, sur 20/20 films (aucune etape ne change dans les 97 autres, donc aucune des corrections ne touche une etape observee du balayage : elles agissent a la publication). Diff champ par champ du document sur les 20 films (eb0ae98 contre 5b8302e, jdiff_artifact.txt) :

| Films | Ecart du document | Attribution |
|---|---|---|
| 20/20 | 7 compteurs neufs : coverage.controlPlaces.unitContradicted et six coverage.flagCarries.gaugeSlots*/gaugeReads* (environ +164 octets) | compteurs neufs |
| 084a804d | un tir reattribue (shots[].slot, x, y, h, p changent, v ajoute) ; coverage.vehicles.shots et shotsVehicleWeapon, shotStats, controlPlaces.* changent | tir de vehicule reattribue (liaison isolee refusee) |
| 084a804d | gaugePaired, gaugeSpans, gaugePoints, gaugeWalkSpans changent ; flagCarries[].spans[].returnProgress +16 | jauge du drapeau |
| 084a804d, 111fa685, 11de8353, 51101d1d, 9f57c612, bcb6d393, e5adf7b2 | hits[].w ajoute (22 a 208 par film) et damage[].w ajoute (25 a 221), jamais retire | touches armees |
| 084a804d, 1c4c63c2, 50247b26 | une entree de coverage.fallbacks en moins (43->42, 41->40, 46->45) et noms/compteurs decales | compteur du repli resserre |

Tous les ecarts sont attribues ; 20 films refiges. Fichiers : 20 replay/testdata/equivalence/<short8>.tsv, 3 colonnes (etape, compte, sha), sans identifiant ni pseudo ni chemin (grep fait). Le refigeage donne les memes 346 insertions / 146 suppressions que le 1er passage par rapport a HEAD (les references de HEAD sont toujours celles de la v1.5.0).

## G-corpus
Statut : ok x4 (0797ce72, 111fa685, e5adf7b2, 11de8353), PERTE x14, FAUX x1 (4f77afc1). Telemetrie : trois revisions changees sur 19/19, attendu. 21 lignes "decision sans effet" (deja connues, non traitees).

Ecarts du 1er passage DISPARUS : 084a804d jauge du drapeau (gaugePaired 1->0, gaugePoints 183->0, gaugeSpans 1->0), tir (shots.v/presents 475->474, vehicles.shots, vehicles.shotsVehicleWeapon) ; 4f77afc1 hits.w/presents 2019->2005 ; 50247b26 fallbacks/n 45->46 et son FAUX (0->1256) ; FAUX de 084a804d (0->2). Le gate n'affiche plus aucun gain ni perte sur ces compteurs (ils sont donc egaux a la base ; je n'ai pas vu de ligne de gain pour hits.w). Ecarts NOUVEAUX par rapport au 1er passage : AUCUN (37 lignes de perte, toutes deja presentes au 1er passage).

FAUX restant : coverage repli `repli_tireur_par_le_siege_de_la_place` 0 -> 6 sur 4f77afc1. Cause etablie par l'enquete : six tirs du siege 2 (frames 8985-9495) vraiment attribues au siege (source siege), allant au joueur qui tient aussi la place 2 sur 329 autres tirs ; sortie voulue.

PERTE restantes, par mesure (film, avant -> apres) et cause :
1. Cause "socles d'equipement neufs dans la liste" :
   - coverage.padDating.uncovered : bcb6d393 4->5, fb1a1a72 14->15, d9781168 24->26, c75f33b8 33->38, bf15f7ab 4->5, 51ebbc0f 16->18, 084a804d 89->143, 60ae07c4 11->18, a349fea8 97->133, a521164d 17->32, 50247b26 25->39, bfecd02b 17->18, 4f77afc1 121->159, 396cfc92 4->5, 0a247154 7->10
   - coverage.padDating.ambiguous : fb1a1a72 13->14, d9781168 4->6, 084a804d 71->84, 60ae07c4 7->9, a349fea8 36->40, 50247b26 8->12, 4f77afc1 60->65
   - coverage.groundWeapons.unknown : d9781168 13->14, c75f33b8 27->32, 51ebbc0f 9->10, 084a804d 127->134, 60ae07c4 5->8, a349fea8 67->77, a521164d 10->12, bfecd02b 11->12, 4f77afc1 91->97
2. Cause "portages et objectifs lus en plus" : bcb6d393 coverage.flagCarries.ambiguousReturns 0->1, coverage.flagCarries.noBridge 0->1, coverage.objectives.noSlot 0->2.
3. Cause "portages en plus, denominateur qui grandit" (23 sur 23 puis 41 sur 41, 100 % des portages avant et apres) : 084a804d coverage.flagCarries.carrierTeamUnknown 23->41.
4. Cause "compteur du repli du tireur compte une fois de plus" : 4f77afc1 coverage.fallbacks/n 40->41 (le meme repli que le FAUX).
5. Cause "penalite de trahison publiee" : 4f77afc1, un joueur, courbe de score 100->0 (mesure joueur/<xuid>/score).
Aucun ecart "non explique".

## Decisions a presenter (format de config/replay_corpus.toml, NON inscrites ; la date et la phrase d'acceptation sont a donner par l'utilisateur)
```
[[decision]]
mesure   = "coverage.padDating.uncovered"
temoins  = ["bcb6d393", "fb1a1a72", "d9781168", "c75f33b8", "bf15f7ab", "51ebbc0f", "084a804d", "60ae07c4", "a349fea8", "a521164d", "50247b26", "bfecd02b", "4f77afc1", "396cfc92", "0a247154"]
decodeur = "v1.6.0"
date     = "<a donner>"
raison   = "socles d equipement neufs dans la liste des socles (decodeur v1.6.0) : le perimetre compte grandit, aucun socle d arme ne regresse ; la definition du compteur ne change pas."

[[decision]]
mesure   = "coverage.padDating.ambiguous"
temoins  = ["fb1a1a72", "d9781168", "084a804d", "60ae07c4", "a349fea8", "50247b26", "4f77afc1"]
decodeur = "v1.6.0"
date     = "<a donner>"
raison   = "socles d equipement neufs dans la liste des socles (decodeur v1.6.0), meme cause que coverage.padDating.uncovered."

[[decision]]
mesure   = "coverage.groundWeapons.unknown"
temoins  = ["d9781168", "c75f33b8", "51ebbc0f", "084a804d", "60ae07c4", "a349fea8", "a521164d", "bfecd02b", "4f77afc1"]
decodeur = "v1.6.0"
date     = "<a donner>"
raison   = "socles d equipement neufs dans la liste des socles (decodeur v1.6.0), meme cause que coverage.padDating.uncovered."

[[decision]]
mesure   = "coverage.flagCarries.ambiguousReturns"
temoins  = ["bcb6d393"]
decodeur = "v1.6.0"
date     = "<a donner>"
raison   = "trois portages du drapeau et treize lignes d objectif lus en plus (decodeur v1.6.0) : rien n est perdu, les objets neufs sont comptes."

[[decision]]
mesure   = "coverage.flagCarries.noBridge"
temoins  = ["bcb6d393"]
decodeur = "v1.6.0"
date     = "<a donner>"
raison   = "portages du drapeau lus en plus (decodeur v1.6.0), meme cause que coverage.flagCarries.ambiguousReturns."

[[decision]]
mesure   = "coverage.objectives.noSlot"
temoins  = ["bcb6d393"]
decodeur = "v1.6.0"
date     = "<a donner>"
raison   = "lignes d objectif lues en plus (decodeur v1.6.0), meme cause que coverage.flagCarries.ambiguousReturns."

[[decision]]
mesure   = "coverage.flagCarries.carrierTeamUnknown"
temoins  = ["084a804d"]
decodeur = "v1.6.0"
date     = "<a donner>"
raison   = "portages du drapeau lus en plus (decodeur v1.6.0) : 23 sur 23 puis 41 sur 41, soit 100 % des portages avant et apres ; le denominateur grandit, rien ne regresse."

[[decision]]
mesure   = "coverage.fallbacks/n"
temoins  = ["4f77afc1"]
decodeur = "v1.6.0"
date     = "<a donner>"
raison   = "le repli repli_tireur_par_le_siege_de_la_place compte une fois de plus (decodeur v1.6.0), meme cause que la decision suivante."

[[decision]]
mesure   = "repli repli_tireur_par_le_siege_de_la_place"
temoins  = ["4f77afc1"]
decodeur = "v1.6.0"
date     = "<a donner>"
raison   = "six tirs du siege 2 (frames 8985 a 9495) attribues au siege, au joueur qui tient aussi cette place sur 329 autres tirs : repli neuf, sortie voulue (decodeur v1.6.0)."

[[decision]]
mesure   = "joueur/<xuid>/score"
temoins  = ["4f77afc1"]
decodeur = "v1.6.0"
date     = "<a donner>"
raison   = "un joueur passe de 100 a 0 : la penalite de trahison de 100 points est publiee dans la courbe du score (decodeur v1.6.0) ; valeur de l API non verifiee."
```
(Le format exact du FAUX d'un repli au manifeste est a confirmer avec les decisions existantes de repli, que je n'ai pas relues ; la ligne `mesure` du repli est une proposition. La ligne du score porte le xuid reel dans le manifeste, comme les decisions existantes du depot hote ; ici il est masque.)

## Fichiers modifies non commites
- Decodeur dec-v160 (feat/v1.6.0, 5b8302e) : 20 fichiers replay/testdata/equivalence/<short8>.tsv (references refigees sur 5b8302e). Rien d'autre.
- Hote LevelUp-wt-dec160 : aucun (go.mod restaure depuis passage2/go.mod.orig, cmd/zz_equiv_dump supprime, git status vide).
- Hors depot : passage2/ contient des dumps avec de vrais identifiants (art_old/, art_new/, dump_v160b/, gate_work_corpus/gate.log) : ne rien en copier dans un depot public. passage2/gate_work_racine (copie 1,3 Go du worktree hote) et gate_work_corpus sont a supprimer par l'utilisateur ou le pilote.
