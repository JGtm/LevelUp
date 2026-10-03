# Plan — Le rang de capacite par `i48` : nommer juste, et debloquer l'effet actif

> Ecrit le 2026-08-14 apres la mesure du lot « equipement actif » (RECETTE_LOADOUT §14).
> Decision utilisateur : « oui bonne idee, planifie bien ». Execution sous `plan-execution`,
> branche `feat/v75`, commits par etape, PAS de push. Le reverse est FAIT — rien ici ne
> demande Ghidra ni capture runtime.

## Le fait qui commande tout le plan

**Le rang de palette est dans le film depuis le debut, et le deserialiseur le jette.**
`consumeBipedDesiredAbilitySet` lit `R(3)` compteur, `R(1)` porte, **`R(6)` identite** — il
consomme les 6 bits pour rester aligne puis les abandonne ; le hook de production
(`abilitySetHook`) ne publie que le `R(3)` et la largeur. Mesure de l'instrument
(`filmdec/i48_rank_test.go`, garde `I48_FILM`) : **748 lectures sur 8 films, ZERO illisible**,
environ une transmission par vie — ce qui suffit largement a une fiche.

Et le canal utilise aujourd'hui (le champ de 3 bits des images-cles) est **tronque** : son
motif d'ancrage se termine par `010`, qui sont les bits de POIDS FORT du rang. Il ne peut donc
voir que les rangs **16 a 23**, et il rend `rang − 16`. Cela explique enfin trois anomalies :
les valeurs ne sortent jamais de 3-7 ; **21 films sur 40 ne rendent aucune lecture** ; et les
films « ou les huit joueurs portent le meme equipement » sont un artefact.

**Ce que le canal `i48` ouvre** : les rangs 8 (camouflage), 9 (surbouclier) et 11
(translocateur) — les TROIS capacites que l'utilisateur veut voir actives — sont OBSERVES dans
le corpus (`084a804d` 8:10 9:8 · `06dfe6d9` 8:2 9:2 11:3). Le canal actuel ne les verra
jamais ; celui-ci les voit, avec le rang complet.

## Etape 1 — Publier le rang (le decodeur cesse de le jeter)

- [x] 1.1 Elargir le hook de production pour qu'il porte le RANG COMPLET (`R(6)`) en plus de
      ce qu'il publie deja. Ne pas casser ses appelants : la valeur `R(3)` et la largeur
      restent. L'instrument `i48_rank_test.go` devient le temoin de non-regression du hook.
      → `abilitySetHook(counter, rank, width)`, `rank = AbilitySetNoRank` porte ouverte ; le
      `consumeGate0R` est reecrit a plat, MEME parcours de bits (test pur : 4 bits porte
      ouverte, 10 fermee). Balayage de production `filmdec.ScanFilmAbilityRanks`. L'instrument
      lit i48 DEUX fois par record (a la main, puis par `consumeByName`) et exige l'accord :
      **92/92 concordantes sur 000d5950**.
- [x] 1.2 Faire remonter le rang jusqu'a l'inventaire du rejeu (`Inventory.A` ou un champ
      voisin — DECIDER et ECRIRE lequel, en regardant ce que la fiche consomme aujourd'hui).
      → DECISION : **ni l'un ni l'autre**. `Inventory.a` est RETIRE et le document publie un
      calque neuf, `abilities` (`{t, slot, r, src}`). Motif : les lectures d'`i48` vivent dans
      les paquets DELTA, pas aux images-cles — les loger dans `Inventory` aurait fabrique des
      lignes d'inventaire vides qui auraient masque la derniere image-cle du slot. Le RETRAIT
      plutot que la reinterpretation est ce qui protege du piege de sens : un client non mis a
      jour ne lit plus rien, au lieu de lire un nombre qui a change de signification.
      `SchemaVersion` 5 -> 6, `openapi-gen` + `generate-types` + `go test ./contracttest/`
      (compte de champs 27 -> 28) dans le meme commit ; re-cuisson des artefacts a l'etape 3.
- [x] 1.3 Le canal d'image-cle reste-t-il ? → **IL RESTE, converti a la MEME grandeur.** Ni
      repli sous un autre nom, ni retrait : les 3 bits qu'il lit sont les bits de POIDS FAIBLE
      du rang, et le motif d'ancrage porte deja les bits de poids fort (`010`). Le decodeur
      reconstruit donc `rang = invAbilityRankHigh<<3 | bas` — la constante est DERIVEE du
      motif, pas ecrite a cote de lui. Les deux canaux publient desormais un RANG, et chaque
      lecture dit son canal (`src`), ce qui rend leur fenetre respective auditable dans
      l'artefact meme. Le canal d'image-cle reste borgne (16..23) et c'est ECRIT partout ou il
      apparait.

Gate 1 : PASSE. Sur 000d5950 : 171 851 records delta, 92 annonces d'i48, **92 lues, 0
illisible** ; rangs `19:18 20:22 21:26 22:16`, aucun < 13. Golden regenere : **214 lectures de
capacite, 82 par i48 + 132 par image-cle**. Aucun test existant ne rougit (paquets `filmdec`,
`replay`, `contracttest`, web `match-replay`). Les rangs 8/9/11 sur `084a804d` et `06dfe6d9`
sont VERIFIES a l'etape 2 (classement de palette), qui les mesure film par film.

## Etape 2 — La palette du film (le verrou de la traduction rang -> nom)

> Mesure §14 : la palette VARIE. Signature famille A (rangs 1,2,4,5,6,8,9,10,11,12,23) sur
> `00ba2e1c`, `06dfe6d9`, `00162144`, `084a804d` ; rangs **19-22 exclusivement**, aucun < 13,
> sur `000d5950`, `00502e52`, `07aa428d`. Presumer la famille A donnerait des noms FAUX sur
> ces trois films.

- [x] 2.1 Chercher d'abord une DESIGNATION dans le film. → **NEGATIF, et mesure.** Trois
      sondages, tous en lecture : (a) le registre du `chunk_00` est bit-a-bit identique d'un
      film a l'autre pour les noms et flags de composants (fait deja etabli, `registry.go`) ;
      (b) sa LONGUEUR ne suit pas les familles — 1 973 120 octets aussi bien sur `00162144`
      (famille A) que sur les trois films de la famille B, 1 944 9xx sur les trois autres
      films de famille A ; (c) aucun marqueur de groupe de tags (`sofd`, `sofa`, `eqip`,
      `vcdd`, `uwfa`, `glpa`) n'apparait dans `chunk_00` ni dans `chunk_01` (seul « weap »
      sort, 19 fois, et ce sont les etiquettes `weapon-state-type-info`). Le choix du `sofd`
      n'est pas serialise : on passe a 2.2, et le NEGATIF est ecrit dans le TOML.
- [x] 2.2 IDENTIFIER LA PALETTE PAR SA SIGNATURE. → Regle ecrite dans `replay/abilities.go`
      (`classifyAbilityPalette`) avec ses chiffres : une palette est retenue quand **au moins
      90 % des lectures portent ses marqueurs**, et jamais en dessous de **10 lectures**. Le
      minimum est DERIVE du seuil (plus petit n tel qu'une lecture parasite ne disqualifie pas
      un film pur : (n−1)/n >= 0,90), pas choisi a part. **Le seuil n'est pas un reglage
      sensible** : six films sur sept sont purs a 100 %, le septieme a 96,2 % — tout seuil de
      50 % a 96 % rend le meme verdict, et un test le verifie. Une signature melangee, trop
      maigre ou inconnue ne nomme RIEN.
- [x] 2.3 Table indexee par **(palette, rang)**, en DONNEE. → `replay_labels.toml` :
      `[abilities]` devient `[[ability_palettes]]`, chaque palette portant son `id`, ses
      `markers` et ses `ranks`. Le loader REFUSE une palette sans id, sans marqueur, dupliquee,
      ou partageant un marqueur avec une autre (le classement serait ambigu). Les entrees
      `4`/`5` sont RELUES : elles etaient indexees par l'index TRONQUE, elles deviennent les
      rangs **20** et **21** de la famille B — meme capacite, meme nombre d'appuis, index
      corrige. Aucun nom n'est ajoute ni retire par cette relecture.

Gate 2 : PASSE. Classement mesure film par film (7 films, 848 lectures, 775 identites) :

| film | rangs observes (i48) | identites | palette | purete |
|---|---|---|---|---|
| `00ba2e1c` | 1:31 2:25 4:34 5:20 6:36 10:21 23:35 | 202 | famille A | 100 % |
| `06dfe6d9` | 1:19 2:25 4:35 5:22 6:38 8:2 9:2 10:34 11:3 12:4 23:35 | 219 | famille A | 100 % |
| `00162144` | 2:14 4:9 9:2 10:10 | 35 | famille A | 100 % |
| `084a804d` | 1:13 4:48 5:11 6:18 8:10 9:8 10:2 19:4 23:15 44:1 | 130 | famille A | 96,2 % |
| `000d5950` | 19:18 20:22 21:26 22:16 | 82 | famille B | 100 % |
| `00502e52` | 19:22 20:17 21:8 22:18 | 65 | famille B | 100 % |
| `07aa428d` | 19:11 20:10 21:13 22:8 | 42 | famille B | 100 % |

Les sept signatures sont rejouees telles quelles par un test — si la regle changeait au point
d'en reclasser une, elle nommerait des capacites differentes sur des films deja servis. Fait
NEUF, absent du §14 : `084a804d` montre 4 lectures `19` et une `44` (hors de toute palette
connue — un `sofd` compte ~27 entrees). C'est le bruit attendu d'un balayage bit a bit ; il est
compte CONTRE la purete plutot qu'ignore, et le film passe quand meme a 96,2 %.

## Etape 3 — Nommer les rangs qui manquent (sans jamais deviner)

- [x] 3.1 Rangs 19 a 22. → **20 et 21 NOMMES, 19 et 22 NON.** 20 (grappin) et 21 (propulseur)
      ont trois appuis : le releve Theater, le CONTROLE DE GROUPE (trois lectures identiques
      par triplet, triplets mutuellement distincts), et — nouveau — le canal i48, totalement
      independant, qui rend 20 sur les slots 513/516 et 21 sur le slot 518. 19 (mur) et 22
      (capteur) reposent sur UNE observation isolee chacun : ils gardent leur numero. C'est
      exactement la regle qui leur avait deja retire leur nom le 2026-08-14 ; l'index change,
      le verdict non.
- [x] 3.2 Rang 23 = champ de reparation. → NOMME. Deux chaines (murmur3 + banque sonore
      `sb_007_abl_repairfield`), et c'est le SEUL rang de la famille A dans ce cas. Observe
      85 fois sur trois films (`00ba2e1c` 35, `06dfe6d9` 35, `084a804d` 15) et sur AUCUN film
      de famille B — ce qui en fait aussi un marqueur fiable de la famille A.
- [x] 3.3 Couverture APRES. → Publiee ci-dessous. **Attention a la comparaison** : les
      47,0 % de reference etaient mesures sur le canal d'IMAGE-CLE seul, sur les films qui
      rendaient quelque chose — 21 films sur 40 n'en rendaient AUCUN. Ce n'est donc pas la
      meme grandeur sur le meme denominateur ; le dire vaut mieux que produire un « avant /
      apres » flatteur et faux.

Gate 3 : PASSE.

**Rangs nommes, avec leurs appuis** (aucun nom a un seul appui de meme nature) :

| palette | rang | nom | appuis |
|---|---|---|---|
| famille A | 1, 2, 4, 5, 6, 8, 9, 11, 12 | detecteur, mur, grappin, propulseur, repulseur, camouflage, surbouclier, translocateur, traqueur | inversion murmur3 du `sofd` — la table DU JEU, lue |
| famille A | 23 | champ de reparation | **DEUX chaines** : murmur3 + banque sonore `sb_007_abl_repairfield` |
| famille B | 20 | grappin | **TROIS** : releve Theater + controle de groupe (3 joueurs) + i48 (canal independant) |
| famille B | 21 | propulseur | **TROIS**, idem |

**NON nommes, et pourquoi** : famille A 0/3/7 (categorie NULLE — course, melee, marquage : ce
ne sont pas des capacites) ; famille A 10 (identifiant non inverse, mais **observe 67 fois** —
c'est le premier trou a combler) ; famille B 19 et 22 (une observation isolee chacun).

**DECISION A CONNAITRE, elle est discutable et donc ecrite** : l'inversion murmur3 du `sofd`
est admise comme source de nommage pour la famille A, alors qu'elle n'est qu'UNE chaine. Motif :
c'est une source d'une autre NATURE qu'une observation d'ecran — elle lit la table du jeu — et
c'est exactement elle qui a fait RETIRER deux noms le 2026-08-14 parce qu'elle contredisait deux
observations isolees. Une source assez sure pour retirer un nom l'est pour en donner un ;
l'utiliser seulement en negatif serait incoherent. Si l'utilisateur veut la barre plus haute,
seul le rang 23 survit en famille A, et l'item 4.2 tombe avec.

**COUVERTURE DE NOMMAGE, canal i48, 7 films** — 775 identites, **610 nommees, 78,7 %** :

| film | palette | identites | nommees | % | ce qui manque |
|---|---|---|---|---|---|
| `00ba2e1c` | A | 202 | 181 | 89,6 % | rang 10 (21) |
| `06dfe6d9` | A | 219 | 185 | 84,5 % | rang 10 (34) |
| `00162144` | A | 35 | 25 | 71,4 % | rang 10 (10) |
| `084a804d` | A | 130 | 123 | 94,6 % | 10 (2), 19 (4), 44 (1) |
| `000d5950` | B | 82 | 48 | 58,5 % | 19 (18), 22 (16) |
| `00502e52` | B | 65 | 25 | 38,5 % | 19 (22), 22 (18) |
| `07aa428d` | B | 42 | 23 | 54,8 % | 19 (11), 22 (8) |

Sur l'ARTEFACT du film de reference (les deux canaux reunis, `000d5950`) : **136 lectures
nommees sur 214, 63,6 %** — contre **0 avant ce lot**, la table etant alors keyee par l'index
tronque. Canal d'image-cle seul sur ce film : 88/132, 66,7 %.

**LE CONTROLE QUI CHANGE TOUT, et il etait ecrit avant la mesure** : la verite terrain Theater
des huit joueurs de `000d5950` est desormais reproduite **8 sur 8** par la lecture de
production. Elle etait a **2 sur 4** au 2026-07-28, et c'est ce « 2 sur 4 » qui avait ouvert
trois branches d'explication et coute le chantier. Aucune n'etait la bonne : les deux lectures
etaient justes, elles etaient comparees a la mauvaise palette.

## Etape 4 — L'effet actif plein-fiche (ce que tout ceci debloque)

> Rappel de la demande utilisateur : l'effet porte sur TOUTE LA FICHE. Surbouclier = fiche
> doree, camouflage = effet de verre, translocateur = bordure animee.

- [!] 4.1 PRE-REQUIS re-verifie : l'identite est connue (etapes 1-3), l'ETAT ACTIF ne l'est
      pas. `i57` est lu sur **0,82 %** des records et son association aux episodes `i54` vaut
      **72,2 % contre 34 % de temoin** — une erreur sur quatre. **AUCUNE LIGNE DE RENDU
      PLEIN-FICHE N'A ETE ECRITE**, et c'est le `[!]` que le plan lui-meme annonce, pas un
      abandon : la fiche doree, l'effet de verre et la bordure animee attendent une source
      d'etat. Une fiche doree une fois sur quatre a tort est pire qu'une fiche sobre.
- [x] 4.2 Ce qui devient possible SANS l'etat actif. → **LIVRE.** Les trois capacites que
      l'utilisateur veut voir sont desormais NOMMEES quand un film les porte : camouflage
      actif (rang 8), surbouclier (9), translocateur quantique (11). Elles sont observees dans
      le corpus — `084a804d` 8:10 9:8, `06dfe6d9` 8:2 9:2 11:3, `00162144` 9:2 — et le canal
      d'image-cle ne les verra JAMAIS (hors 16..23). Leurs vignettes de HUD existaient deja au
      depot (`hud/ActiveCamouflage`, `hud/Overshield`, `hud/QuantumTranslocator`) : nom et
      vignette arrivent ensemble. C'est un gain reel et immediat, a ne pas confondre avec
      l'effet demande.
- [!] 4.3 Source d'etat fiable : aucune n'est apparue pendant ce lot, et en chercher une
      etait hors perimetre. Les voies restent au registre (rendre `i56` lisible sur les
      records denses ; verite Theater datee a la seconde). Ce qui a CHANGE en leur faveur :
      quand l'une aboutira, l'effet se codera sans rien inventer — la palette est resolue et
      l'identite est publiee.

Gate 4 : NON FRANCHI, comme annonce. Ce plan livre l'IDENTITE, pas l'ETAT.

## Re-cuisson des artefacts (obligation du bump de schema)

**FAITE** : `levelup backfill-replay --only-existing`, en QUATRE passes bornees (`--limit 8`,
`6`, `5`, `4`) plutot qu'une longue — machine instable, un decodage a la fois. Resultat :
**23 artefacts construits, 0 erreur de decodage, 0 carte hors catalogue**, 26 min cumulees.
Verification sur piece (`000d5950.json`) : `schemaVersion` 6, `abilities` 214 lectures dont 82
`i48` et 132 `kf`, `abilityLabels` keye par RANG (20 grappin, 21 propulseur, vignettes
comprises), et `inventory[0].a` a bien DISPARU.

Ce que la passe a montre en prime — **le refus fonctionne en production** : sur les 23 films,
**7 sortent en palette « non classee »** (1 a 8 lectures, sous le plancher de 10) et recoivent
**zero nom** au lieu d'un nom faux. Les 16 autres se classent (15 famille A, 1 famille B) et
nomment 2 a 9 rangs chacun.

## Decouvertes (notees, NON traitees — hors perimetre)

1. **Le rang 10 est le premier trou a combler** : 67 lectures sur quatre films, identifiant de
   chaine non inverse. A lui seul il fait l'essentiel des 21,3 % de lectures sans nom.
2. **`084a804d` porte 4 lectures `19` et une `44`** que le §14 n'avait pas relevees ; `44` est
   hors de toute palette connue. Bruit de balayage, compte contre la purete.
3. Le serveur de dev local tenait `metadata.duckdb` pendant la re-cuisson : la resolution des
   noms EN est tombee en mode degrade (`map_name` brut). Sans consequence ici — les 23 cartes
   se sont resolues — mais c'est le piege mono-process documente, et il se representera.

## Hors perimetre

- L'etat actif lui-meme (voies au registre) ; le lecteur `i56` sur records denses.
- Toute renumerotation « au jugé » de la table actuelle sans passer par l'etape 2.
- Le rendu plein-fiche tant que 4.1 n'est pas leve.

## Ce qui peut faire echouer ce plan

1. **`i48` est rare** (0,03 a 0,09 % des records delta, ~une fois par vie). Si une vie n'en
   porte aucun, la fiche n'a pas d'identite pour cette vie — prevoir le repli (derniere
   valeur connue du joueur ? rien ?) et l'ECRIRE, ne pas le laisser au hasard.
2. **La palette peut rester indeterminee** sur des films dont la signature est trop pauvre
   (peu de rangs observes). C'est prevu : on n'affiche pas de nom, on affiche le rang.
3. **Le champ d'inventaire change de sens** (rang complet au lieu d'index tronque) : tout
   consommateur non mis a jour lirait un nombre qui ne veut plus dire la meme chose. C'est
   l'item 1.2, et c'est le vrai risque de regression de ce plan.
