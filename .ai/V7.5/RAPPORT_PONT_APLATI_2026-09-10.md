# RAPPORT — LES SIX LECTEURS DU PONT APLATI (`PontParSlot`), MESURE SUR LE PARC

> Lot 6.1 du `PLAN_MASTER_2026-09-09.md` (vague 6), phase A. Worktree `wt/pont-aplati`.
> Mesure du 2026-09-10. Aucune base DuckDB ouverte, aucun backfill, aucune recuisson du parc.

## 0. Ce que ce rapport tranche

Le report 1.6 du plan maître (découverte 2 du lot E2-bis) dit : six consommateurs lisent le pont
APLATI `PontParSlot()` — qui garde le PREMIER occupant d'un siège recyclé — au lieu du registre à
l'instant (`XUIDAt`). Sur un film long à sièges recyclés (`084a804d` : 123 sièges à deux corps),
ces lectures « peuvent créditer le mauvais joueur ». La question posée était : **combien, où, et
qu'est-ce qui est réellement servi faux à l'écran.**

Réponse mesurée : **l'écart existe, il est prouvé, et il est minuscule** — un siège sur 74 films,
deux ramassages nommés faux à l'écran, une lecture d'inventaire, zéro épisode d'équipement, zéro
portage de bombe. Le site du sync (`kill_positions`, servi à Tactique) a une fenêtre d'exposition
d'**une frame** sur tout le parc mesuré. Le détail est ci-dessous ; la phase B (migration) est
justifiée par la NATURE de l'écart, pas par son volume.

## 1. Les six sites, vérifiés sur pièces au HEAD de `wt/pont-aplati`

Le code a bougé depuis la rédaction du report (lots 4.3 et 5.1). Grep de production
(`grep -rn "PontParSlot()" --include=*.go internal/ | grep -v _test.go`) — **six sites, pas un de
plus** :

| # | Site (HEAD) | Report 1.6 | Nature de la lecture |
|---|---|---|---|
| 1 | `analysis/replay/bomb_carries.go:137` | `:142` | GARDE `len(...) == 0` — aucune attribution |
| 2 | `analysis/replay/bomb_carries.go:142` | `:142` | ATTRIBUTION — `BuildHeldObjectCarry(events, pont, deaths)` |
| 3 | `analysis/replay/bomb_stats_document.go:77` | `:77` | TÉMOIN booléen `len(...) > 0` — aucune attribution |
| 4 | `analysis/replay/build.go:161` | `:152` | ATTRIBUTION — frags/assistances sous équipement actif |
| 5 | `analysis/replay/build.go:228` | `:219` | ATTRIBUTION — ramassages natifs (`pickupInputs.slotXUID`) |
| 6 | `analysis/replay/inventory_dead_readings.go:33` + `:42` | `:42` | GARDE + ATTRIBUTION — requalification `unknown` → `dead` |
| 7 | `sync/killcollector/positions.go:309` | `:301` | ATTRIBUTION — `kill_positions` / `kill_openings` persistés AU SYNC |

Soit **quatre attributions réelles** (2, 4, 5, 6) plus le site de sync (7), et **deux témoins
booléens** (1, 3) qui ne nomment personne.

## 2. Le critère d'écart, démontré avant de compter

`PontParSlot()` rend `OwnerReport.SlotXUID`. `ownersFromLives` (lives.go:522) y garde la PREMIÈRE
vie nommée d'un siège et marque le siège AMBIGU dès qu'une vie suivante nomme un autre joueur ;
`poserIdentiteDeVie` (identity_registry_mutations.go:54) fait exactement pareil pour les
déductions. `xuidAt` (identity_registry_pont.go:166) rend la vie qui COUVRE l'instant, s'abstient
sur un siège ambigu, et retombe sinon sur le pont aplati.

D'où l'équivalence, qui est ce qui rend la mesure exhaustive et bon marché :

> **pont aplati ≠ registre à l'instant ⟺ le siège est AMBIGU** (`SlotCollisions`).

- Siège à une seule identité (même joueur sur deux vies) : les deux accesseurs rendent la même
  chose, à tout instant. Aucun écart.
- Siège hors vie nommée : `xuidAt` retombe sur le pont aplati. Aucun écart.
- Siège ambigu : `xuidAt` nomme le VRAI occupant dans une vie qui couvre l'instant, et
  S'ABSTIENT hors vie ; le pont aplati sert le premier occupant dans les deux cas. **Écart.**

Ce compte est publié par la production elle-même : `coverage.bridge.slotCollisions` dans chaque
artefact. La mesure ci-dessous le reconstruit indépendamment depuis `identity.bipedSlots` (une
ligne PAR VIE, avec ses bornes de frames) et **les deux comptes concordent sur les 74 films** —
c'est le contrôle qui valide l'instrument.

## 3. L'instrument, et pourquoi celui-là

Deux voies étaient possibles : rejouer le registre sur les films en cache par un
`_research_test.go`, ou lire les artefacts déjà cuits. **La seconde a été retenue** : les
artefacts publient déjà la table de vies du registre (`identity.bipedSlots`, une ligne par vie
avec `link.from`/`link.to`) ET les lectures publiées de chaque site (`pickups` avec leur frame et
leur xuid, `inventory` avec sa frame, `equipmentEpisodes` avec sa fenêtre, `bombCarries`). Rejouer
un registre pour retrouver ce que l'artefact publie déjà aurait coûté des heures de décodage pour
la même réponse.

Deux populations, disjointes :

1. **Le parc local** — 64 artefacts au schéma 51, `data/cache/replays/halo_infinite/`, recuits le
   2026-09-10 (62 sur 64 ; 2 datent du 09-07). **Lus en lecture seule, jamais réécrits.**
2. **Le corpus témoin d'équivalence** — 10 des 13 films de
   `internal/analysis/replay/testdata/equivalence/CORPUS.txt` présents au cache (`1c4c63c2`,
   `60ae07c4`, `a349fea8` absents), dont **`084a804d`**, le film aux 123 sièges recyclés que le
   report nomme, et `d9781168`. Cuits pour cette mesure par
   `cmd/replay-build --facts <short8>.facts.json` avec
   `LEVELUP_REPO_ROOT=<worktree wt/pont-aplati>` : la cuisson lit les chunks du parc principal
   par chemin explicite et **écrit dans le `data/cache/` du worktree**, jamais dans le parc.
   Aucune base ouverte (les faits du match viennent des fichiers `.facts.json` figés), aucun accès
   à l'installation du jeu (les catalogues de bornes sont versionnés).

Total : **74 films, 1 205 sièges de bipède, ~13 000 ramassages, ~16 900 lectures d'inventaire,
1 146 épisodes d'équipement, 11 portages de bombe.**

## 4. Résultat — sièges ambigus

| Population | Films | Films à siège ambigu | Sièges ambigus | `slotCollisions` déclaré |
|---|---|---|---|---|
| Parc (64) | 64 | **0** | **0** | 0 sur les 64 |
| Corpus témoin (10) | 10 | **1** (`084a804d`) | **1** (siège 603) | 1 |
| **Total** | **74** | **1** | **1** | concordant |

`084a804d` au HEAD : `collisionsSlot=1`, `slotsRecycles=123`, `viesTotal=353`, `viesNommees=199`,
`parCreation=257`, `parCreationPropagee=95`, `liens non résolus=1`. **123 sièges recyclés, UN
SEUL ambigu** : les correctifs E2-bis (« le corps est `(slot, génération)` ») ont supprimé la
quasi-totalité du défaut à la racine. Le report 1.6 est resté écrit dans les termes d'AVANT ces
correctifs.

## 5. Résultat — écarts par site

Un « écart » = une lecture publiée pour laquelle `PontParSlot[slot] != XUIDAt(slot, instant)`.
Deux natures : **FAUX** (le registre nomme un AUTRE joueur — le pont a crédité le mauvais) et
**ARBITRAIRE** (le registre s'abstient — le pont a publié un nom que rien n'établit).

| Site | Lectures mesurées | Écarts | dont FAUX | dont ARBITRAIRE |
|---|---|---|---|---|
| 5 — ramassages natifs (`build.go:228`) | 13 232 | **2** | 0 | **2** |
| 6 — inventaire mort (`inventory_dead_readings.go:42`) | 16 884 | **1** | 0 | 1 |
| 4 — frags sous équipement (`build.go:161`) | 1 146 épisodes | **0** | 0 | 0 |
| 2 — portage de bombe (`bomb_carries.go:142`) | 11 portages (1 film) | **0** | 0 | 0 |
| 1 et 3 — témoins booléens | — | sans objet | — | — |
| 7 — `kill_positions` (sync) | cf. §6 | fenêtre d'**1 frame** | — | — |

Tous les écarts sont sur `084a804d`, siège 603.

### 5.1 Ce qui est réellement servi faux à l'écran

Le siège 603 de `084a804d` porte trois vies, toutes nommées par LECTURE DIRECTE du record de
création du bipède :

```
vie [2314..2779]  2533274817603732   creation_bipede
vie [3101..3118]  2533274817603732   creation_bipede_propagee
vie [9721..9721]  2535419608696209   creation_bipede      <- deuxieme corps du siege
```

Le pont aplati sert `2533274817603732` pour tout le film. Conséquences publiées :

- **Deux ramassages nommés au mauvais joueur, visibles sur la page de rejeu.** Frames 9802 et
  9968 (une arme, puis un grappin), publiés avec `xuid = 2533274817603732` alors que le seul
  corps que le FILM établit sur ce siège à cet instant est `2535419608696209`. Le registre à
  l'instant, lui, s'abstiendrait : aucune vie NOMMÉE ne couvre ces deux frames (elles tombent
  après la dernière frame de position du film), et sur un siège ambigu `XUIDAt` refuse de servir
  le premier occupant. **La migration retire un nom faux ; elle ne rend pas le vrai** — c'est la
  doctrine du dépôt (le silence plutôt qu'un nom arbitraire), et il faut le dire tel quel.
- **Une lecture d'inventaire** (frame 3180, hors des trois vies) dont la requalification
  `unknown` → `dead` s'appuie sur un nom que rien n'établit. **Aucun effet visible** : le nom ne
  sert qu'à choisir la liste de morts consultée ; il ne s'affiche pas.
- **Aucun frag ni assistance sous camouflage/surbouclier** n'est mal crédité : aucun des 1 146
  épisodes du parc ne tombe sur un siège ambigu. Le cas historique du report (`fb1a1a72` slot 625,
  une assistance) a été **résolu à la racine par E2-bis** — ce film est au parc et sort à 0.
- **Aucun portage de bombe** mal attribué : le seul film à bombe du corpus (`9f57c612`, 11
  portages) n'a aucun siège ambigu.

## 6. Le site du sync (`kill_positions`), et sa fenêtre exacte

`positions.go:309` passe le pont aplati à `replay.BuildKillPositions` /
`BuildKillOpenings`, qui l'inversent (`slotsByXUID`, killpos.go:211) pour chercher, à l'instant du
coup fatal, la trajectoire du tueur et celle de la victime. Sur un siège ambigu, l'inversion a
deux effets : le SECOND occupant perd ce siège (position introuvable → `NoBridge`/`Dropped`), et
le PREMIER se le voit attribuer sur toute la durée du film (une position d'un AUTRE corps peut
lui être servie, si `positionOf` ne retient que ce siège-là — la garde `n != 1` l'écarte dès que
le joueur a un échantillon ailleurs).

La fenêtre d'exposition se calcule exactement : ce sont les frames d'un siège ambigu qui
n'appartiennent pas au premier occupant. Sur le parc mesuré :

```
084a804d, siege 603 : fenetre d exposition = 1 frame (100 ms), la DERNIERE du film
                      (frameCount = 9722, donc derniere frame = 9721)
                      pendant [9721..9721], 2533274817603732 n occupe AUCUN autre siege
tous les 73 autres films : 0 siege ambigu -> 0 frame d exposition
```

**Une frame de 100 ms sur 74 films.** C'est la totalité de ce que `kill_positions` peut servir de
faux à Tactique aujourd'hui, et il faudrait qu'une mort tombe dans les 33 ms (`killPosTolerance`)
autour de la dernière frame d'un match de 16 minutes. Le report 1.6 disait « surtout
`killcollector/positions.go` » : la mesure **inverse la hiérarchie** — c'est le site le moins
exposé des six, et c'est le seul dont le correctif coûte un redécodage du parc.

## 7. Verdict de la phase A

1. **L'écart est réel et prouvé** : deux ramassages nommés faux sur la page de rejeu, sur un film
   du corpus. La phase A ne rend donc PAS « 0 écart partout », et le lot ne s'arrête pas ici.
2. **Le volume ne justifie rien à lui seul** : 3 lectures sur ~31 000, 1 siège sur 1 205, 1 film
   sur 74. Ce qui justifie la migration, c'est la NATURE du défaut — le pont aplati **publie un
   nom là où le registre sait qu'il faut se taire** —, et le fait qu'aucun garde-rail n'empêche
   aujourd'hui un septième lecteur de rouvrir la même porte.
3. **Le coût n'est pas uniforme** : cinq sites sur six vivent dans `analysis/replay` (cuisson
   pure, aucun écrit en base, gate corpus pour preuve). Le sixième (`positions.go`) persiste au
   sync et exige, pour que la correction atteigne les lignes déjà écrites, un bump de
   `IsolationDecoderRev` et un `backfill-killsource` sur tout le parc — pour une frame de 100 ms.
   Ce coût est signalé au superviseur, la décision de REJOUER le backfill lui appartient ; le code
   corrigé, lui, est livré (le plan maître le prescrit).

## 8. Découvertes (consignées, NON traitées — règle 7)

1. **Un siège partagé entre un bot et un humain trompe les DEUX accesseurs.** `poserBidDeVie`
   (identity_registry_mutations.go:85) pose `bid` sur la vie mais ne touche ni `SlotXUID` ni
   `SlotAmbiguous` — délibérément (un bot n'a pas de xuid à y mettre). Conséquence : sur un siège
   qu'un humain a occupé puis qu'un bot occupe, `XUIDAt` ne trouve aucune vie à xuid couvrant
   l'instant, ne voit pas d'ambiguïté, et **retombe sur l'humain** — exactement comme le pont
   aplati. Migrer vers `XUIDAt` ne corrige donc pas cette figure-là. *Condition de reprise : le
   lot qui traitera les sièges d'index partagés bot/humain (verdict I0, découverte 4 du lot E2).*
2. **`buildPickups` publie des ramassages au-delà de l'axe de frames.** Les deux ramassages du
   §5.1 sont à `T` 9802 et 9968 alors que `frameCount` vaut 9722 : `document_pickups.go` calcule
   `T` sans borner à `clk.frames`, contrairement aux autres calques. Le client reçoit des
   ramassages qu'aucune frame ne porte. P2, sans rapport avec le pont.
3. **Trois films du corpus d'équivalence sont absents du cache** (`1c4c63c2`, `60ae07c4`,
   `a349fea8`), dont `1c4c63c2` qui est « le plus gros film du cache » selon `CORPUS.txt`. Le
   corpus d'équivalence n'est donc rejouable qu'à 10/13 sur ce poste.
