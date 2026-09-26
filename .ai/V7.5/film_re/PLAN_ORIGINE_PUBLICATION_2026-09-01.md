# PLAN — publication de l'origine des ramassages (lot final)

Branche `wt/origine-equipement`. Contrat : skill `plan-execution` (ordre strict, une etape a la
fois, aucun report d'action faisable, statut sur chaque item).

Prerequis leve avant l'etape 1 : `origin/feat/v75` etait deja en **SchemaVersion 31** (chantier
`wt/pickup-nommage`, le NOM de l'objet ramasse). L'amont a ete integre par MERGE (21 commits,
aucun conflit) AVANT toute numerotation. Le schema de ce lot est donc **32**, pas 31.

Decouverte a l'integration : le lot parallele conclut, dans
`.ai/V7.5/film_re/NOTE_ORIGINE_LEVITATION_2026-09-01.md`, qu'il manque « un catalogue de points
d'apparition d'equipement extrait des `.mvar` » et que « son absence est ce qui bloque la branche
point d'apparition depuis deux lots ». C'est exactement ce que la recette de ce worktree produit.

## Etape 1 — typer les 16 points de la recette

- [x] 1.1 Types resolus — mais par la TROISIEME voie, les deux premieres etant refutees
- [x] 1.2 `fosp` sonde : NON ELUCIDE, consigne (ses references ne resolvent dans aucun module)
- [x] 1.3 GATE A passe : 66/66 ramassages non-arme nommes par le manifeste ; les points
      d'equipement/grenade recoupent bien les classes 2/3
- [x] 1.4 GATE B passe : 47,6 % des armes sur les socles d'armes prouves contre un temoin de
      densite a 20,0 % — 2,4x le hasard

## Etape 2 — regenerer `map_weapon_pads.json` par la recette

- [x] 2.1 Recette en production (`himap.EstPointDApparition`), table typee
      (`mapvar.spawnPointTypes`), garde-rail `TestRecetteRedonneLaTableDesPoints` VERT (16 = 13 + 3)
- [x] 2.2 60 `.mvar` manquants telecharges (72 cartes au catalogue, 12 deja la), auth JGtm,
      aucune re-capture, `--dry-run` : catalogue d'objectifs partage intact
- [x] 2.3 GATE NON-REGRESSION PASSE : 72/72 entrees identiques AU CARACTERE hors `spawn_points`,
      en-tete inchangee, 0 carte perdue, seule cle ajoutee `spawn_points`

## Etape 3 — publier l'origine dans le document de rejeu

- [ ] 3.1 Champ `origin` sur les ramassages non-arme : `spawner` / `ground` / absent
- [ ] 3.2 Schema 32 (courant verifie = 31), contrat openapi + generated + contracttest
- [ ] 3.3 Gate golden nomme
- [ ] 3.4 Compteurs de couverture, DONT le compteur de CARTE ABSENTE du catalogue
      (decision produit : le trou se VOIT, il ne se telecharge pas pendant la cuisson)

## Etape 4 — cuisson pilote, UN SEUL film

- [ ] 4.1 Catalyst `01e1f945`, une invocation, avant-plan, `filmproc.Arm` 3 GiB,
      zero process concurrent verifie AVANT, ecriture dans le worktree
- [ ] 4.2 Comptes `origin` de l'artefact confrontes aux mesures de recherche

## Etape 5 — cloture

- [x] 5.1 Gates Go : contracttest, replay, mapvar, replaybuild, mapopads-build VERTS.
      `himap` : mes tests verts un par un ; la suite complete depasse le delai par defaut et
      monte a 11 GiB — lenteur locale CONNUE du paquet, consignee en tete du garde-rail.
- [x] 5.2 Gates web : typecheck vert, 128 fichiers / 1971 tests vitest verts
- [x] 5.3 Note + thought_log
- [x] 5.4 Commits locaux, AUCUN merge, AUCUN push

## Doctrine rappelee par le coordinateur

La cuisson reste OFFLINE-PURE : jamais de telechargement pendant la generation d'artefact. Les
trous de catalogue se COMPTENT et se comblent par la CLI / le sync, pas par la cuisson.

## Journal

### Etape 1 — CLOSE. Deux voies refutees, la troisieme livre.

**Voie 1, la chaine de tags — REFUTEE.** type_id -> `food` -> `foki` -> `fosp` -> objet
engendre : elle s'arrete au `fosp`. Ses references ne resolvent dans AUCUN module indexe
(96 804 entrees), et le manifeste d'equipement du titre ne recoupe pas une seule reference des
16 types. Resultat brut : 13 INDETERMINE sur 16. `fosp` reste NON ELUCIDE.

**Voie 2, les naissances `ti=37` — REFUTEE, et le diagnostic est net.** Sur 283 naissances lues,
0 nommee : 244 sans identite transmise et 39 avec identite mais hors manifeste — **39
identifiants distincts, une occurrence chacun**. C'est la signature d'un identifiant d'INSTANCE,
pas d'un identifiant de catalogue. `AbilityID` n'est pas la cle du manifeste sur ce canal.

**Voie 3, le canal natif des ramassages — ELLE LIVRE.** `BipedPickup.CatalogID` est un
identifiant de catalogue (mesure du depot), et c'est la table du manifeste qui a etabli les
classes 2/3 au schema 31. Sur Catalyst : 199 ramassages, 66 non-arme, **66 nommes sur 66**,
41 apparies a moins d'un metre d'un point catalogue.

**Le temoin est le TAUX DE BASE du film** : 66 non-arme sur 199 = 33,2 %. Un point qui rend
33 % de non-arme ne porte aucune information.

| type_id | armes | gren. | equip. | non-arme | lecture |
|---|---|---|---|---|---|
| `0x5F379533` | 13 | 0 | 0 | 0,0 % | socle PROUVE `power` — coherent |
| `0x6253CFC0` | 7 | 1 | 3 | 36,4 % | socle PROUVE `rack` — au taux de base, bruit pur |
| `0x5E86D110` | 0 | 0 | 4 | 100,0 % | socle PROUVE `powerup` — 4 `powerup_overshield` |
| `0xADEEE6D8` | 3 | 15 | 1 | 84,2 % | point NON-ARME, contenu 15/16 grenade |
| `0xE42158DF` | 1 | 0 | 11 | 91,7 % | point NON-ARME, contenu 11/11 equipement |
| `0x0CD504B0` | 10 | 3 | 0 | 23,1 % | SOUS le taux de base — pas un point d'equipement |
| `0x5F3FA667` | 4 | 0 | 0 | 0,0 % | sous le taux de base |
| `0xAEDF9CF0` | 1 | 0 | 2 | 66,7 % | n = 3, pas de majorite |
| `0x2BEF1E2D` | 2 | 0 | 0 | — | n < 3 |
| `0xF8EC7E1E` | 1 | 0 | 1 | — | n < 3 |

**CE QUI VALIDE LA METHODE** : les trois socles PROUVES tombent exactement ou ils doivent, et
aucun n'a servi a calibrer quoi que ce soit. C'est le controle le plus fort du lot.

**SEUIL MANQUE, ET PUBLIE COMME TEL.** Le seuil ecrit avant la mesure etait « n >= 3 ET 80 %
d'une meme classe sur les trois ». `0xADEEE6D8` rend 15 grenades sur 19 = 78,9 % : il le manque
d'une observation. Le seuil n'a PAS ete deplace apres coup. La lecture retenue pour le catalogue
est en DEUX temps, et le second est une hypothese, pas une preuve :

1. *ce point est-il non-arme ?* — critere ECRIT AVANT (part non-arme > taux de base) :
   `0xADEEE6D8` (84,2 %) et `0xE42158DF` (91,7 %) le passent franchement ;
2. *grenade ou equipement ?* — juge DANS les non-armes, effectifs publies : 15/16 grenade pour
   `0xADEEE6D8`, 11/11 equipement pour `0xE42158DF`. Une carte, un film : a confirmer.

**Decouverte** : les deux `type_id` trouves a la main au lot precedent, par pure geometrie, sont
exactement les deux que le canal natif type — l'un grenade, l'autre equipement. Deux chaines
independantes, meme resultat.

### Etape 2 — CLOSE. La non-regression est devenue STRUCTURELLE.

**LA SOURCE A DERIVE, et une regeneration en bloc aurait ete une regression.** Neuf des 72
cartes rendent aujourd'hui un `.mvar` different de celui qui a bati le catalogue le 2026-08-19
(Deadlock : 462 objets au catalogue, 410 au telechargement d'aujourd'hui). Une regeneration
complete a effectivement reecrit leurs socles d'ARME — mesure : 63/72 identiques, 9 modifies.
Or ces socles alimentent des chemins livres.

**REPONSE : un mode qui ne PEUT PAS ecrire un socle.** `--only-add-spawn-points` charge le
catalogue existant, recalcule les socles pour VERIFIER qu'ils retombent a l'identique, et
n'ecrit que `spawn_points`. Une carte dont les socles ne retombent pas est SAUTEE et COMPTEE.
La garantie est structurelle, pas verifiee apres coup.

**BOGUE ATTRAPE EN CHEMIN, ET IL AURAIT PUBLIE UN FAUX CATALOGUE.** Le premier aplatissement
ecrivait chaque `.mvar` sous son nom nu — or **58 cartes partagent `mvar_file: "map.mvar"`** :
elles s'ecrasaient toutes dans un seul fichier, et 65 cartes sur 72 sortaient avec les socles
d'une carte etrangere (signature : des « 11 pads » uniformes). Seuls les noms desambiguises
sont ecrits desormais.

**Resultat** : 1934 points sur 63 cartes — 346 `grenade`, 348 `equipment`, 1240 `unknown`.
Neuf cartes restent SANS points : le trou est voulu, visible, et inscrit dans les `notes` du
fichier lui-meme.

### Etapes 3, 4 et 5 — CLOSES.

**Etape 3.** Schema 32. La collision a ete EVITEE en verifiant l amont d abord : `feat/v75` etait
deja en 31 (`wt/pickup-nommage`), integre par merge AVANT numerotation. La chronique v32 leve
explicitement la refutation ecrite au v31, dont l une des deux raisons etait « le depot ne
declare aucun point d apparition d equipement ».

**Etape 4 — LE RECOUPEMENT EST EXACT, et c est le meilleur controle du lot.**

L artefact rend `origineSocle=33`, `origineSol=14`, `origineInconnue=19` — somme 66, l invariant
boucle. La recherche, elle, avait trouve **41** ramassages non-arme sous le metre d un point,
mais sur les 65 points BRUTS de la recette, socles d armes compris. Or la production exclut les
trois types de socle d arme (ils sortent par `pads`). Les ramassages non-arme tombes sur ces
types etaient : `0x5E86D110` 4 et `0x6253CFC0` 4, soit 8.

> **41 - 8 = 33.** Le compte de production tombe exactement sur celui de la recherche, une fois
> retiree la part que la production exclut par construction.

`pointsCatalogue=35` contre 65 objets au dump : c est le REGROUPEMENT a 1 m (`PadSpotMergeM`),
le meme que celui des socles. 65 objets declares font 35 emplacements.

Premiere cuisson : `carteAuCatalogue=false`. La CLI cuit a partir d un NOM de carte et n a pas de
`map_id` sans fichier de faits — la chaine etait donc muette hors service. Corrige par un repli
par nom public, `map_id` restant prioritaire.

**Etape 5.** Gates verts. Aucun merge, aucun push : la revue adversariale passe avant.

### Ronde 1 de corrections (double revue) — CLOSE

**P1-A** trois etats (`map_absent` / `not_established` / `established`), portes jusque dans la
forme du JSON par un POINTEUR sur la tranche : cle absente != `[]`. Le booleen
`mapCatalogMissing` est RETIRE, pas complete.

**P1-B** borne haute du `ground` = `UntilMax` de la pose. Sur donnees reelles, `origineSol`
tombe de 14 a 10 : quatre attributions etaient fausses.

**P1-C** effacement des points perimes au saut pour derive, avec compteur ; contre-cas « sans
dump » qui NE doit PAS effacer, teste explicitement.

**Le verrou durci change le resultat** : 16 cartes sautees au lieu de 9. `memesSocles` seul ne
verifiait rien sur une carte sans socle ; en ajoutant `objects_n` et `level_id`, sept cartes de
plus se revelent derivees. Catalogue final 1662 points / 56 cartes, 72/72 socles identiques.

**Les inversions ont trouve un defaut que la revue n'avait pas vu** : les deux inversions P1-A
PASSAIENT, mes attendus etant les constantes testees. Corrige en litteraux + test d'unicite +
test `replaybuild` sur un chemin qui n'en avait aucun.

**P2** : 1 `resolve()` durci (nom ambigu refuse, ambiguite CALCULEE depuis le catalogue) +
`objects_n`/`level_id` au verrou · 2 `Mixed` sur les fusions de points, journalise (jamais nul
sur BTB) · 3 compteurs `sansDump` et « acceptee a zero point » · 4 commentaire faux rectifie
(`ground` se publie hors catalogue) · 5 `Kind` CONSOMME (`spawnerByPointKind`) plutot que
supprime — il donne le controle du typage en production · 6 parametres groupes
(`pickupInputs`), commentaire corrige · 7 chronique du contrat schema 32 · 8 journal avec les
chiffres de production.

## Decouvertes (notees, NON traitees)

- Neuf cartes du catalogue ont une source UGC qui a derive depuis le 2026-08-19. Re-extraire
  leurs socles changerait ce qui est servi sur ces cartes : c'est une DECISION PRODUIT, hors
  du perimetre de ce lot. Consigne, non traite.
- Le paquet `himap` monte a 11 GiB et depasse le delai `go test` par defaut : le balayage du
  catalogue Forge (4 235 tags) est intrinsequement lourd. Il saute sans le jeu installe, donc la
  CI ne le voit pas — mais il interdit de lancer la suite `himap` en parallele d une cuisson.
  Ecrit en tete du garde-rail. Non traite (ce serait un lot d optimisation a part).
- Quatre cartes de `map_objectives` absentes du catalogue des socles pourraient y entrer
  (le generateur en trouve 75 contre 72). Ajout de cartes = changement de perimetre servi,
  hors lot. Non traite.
