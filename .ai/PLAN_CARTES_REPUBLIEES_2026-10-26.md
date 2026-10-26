# Plan : cartes republiées regroupées quand elles désignent le même terrain

> Rédigé le 2026-10-26, validé par le user le même jour (« Ok avec ton plan »).
> Contrat d'exécution : skill `plan-execution`. Branche `feat/cartes-republiees`, depuis
> `release/v8.1.0` (03c5d89fa), worktree `C:\Users\Guillaume\Downloads\Scripts\LevelUp-wt-cartes-republiees`.
> Suite de `.ai/PLAN_CARTES_VARIANTES_STATS_2026-10-26.md` (même résolveur de groupes).

## Constat

Une même carte peut porter plusieurs identifiants (asset republié). Hors variante, chaque identifiant
reste une vignette et une ligne à part : Shogun (9 + 1 parties chez JGtm, donc deux vignettes sous le
plancher de 10), Bazaar, Houseki. Le regroupement livré par le plan des variantes ne fusionne des
identifiants de même nom que si la carte a une variante (`analysis.GroupMapsByBase`).

## Règle (décision user 2026-10-26)

Deux identifiants de même nom forment un groupe SEULEMENT s'ils désignent le même terrain, établi
par les données et non par le nom : ils se résolvent vers la même clé de MODULE (carte intégrée au
jeu) par la cascade existante (`mapcatalog/callouts_resolution.go` essai 1, ou la cascade des fonds
`service/replay_map_background.go`). Une carte Forge republiée a sa propre clé (son map_id) : ses
versions restent SÉPARÉES partout (règle écrite dans `callouts_resolution.go` : deux assets de même
nom sont souvent deux versions Forge dont les zones ne se superposent pas). Une clé obtenue par le
repli PAR NOM de l'index des fonds ne prouve rien pour ce besoin : seule une clé de module compte.
Un identifiant sans clé reste seul. Libellé du groupe : celui de l'identifiant le plus joué.

## Étapes

### E0 — Mesure sur pièces (sans arrêter le serveur de :8000, sans copier ses bases verrouillées)
- [x] Lister les noms portés par plusieurs map_id au registre (sauvegarde lisible ou API locale, même
      méthode que le E0 du plan des variantes), avec leurs parties -> M1, M3 (56 noms, 135 map_id)
- [x] Pour chacun : clé de module de chaque map_id (même / différente / absente), Forge ou intégrée
      -> M2, M3 (20 noms intégrés, une clé unique chacun ; 36 Forge sans clé)
- [x] Pour chaque paire de même clé : recouvrement des positions de morts (p1-p99, seuil 90 %) ;
      sous le seuil = ARRÊT et décision user -> M4 (2 paires mesurées : 100,0 % et 98,2 % ; 31 non
      mesurables faute de film, écart signalé)
- Gate : tableau écrit dans la section Mesures -> fait

### E1 — Règle dans le résolveur
- [ ] La table des groupes reçoit, en plus des noms, la clé de module de chaque map_id (lue au même
      endroit que les noms, injectée : l'analyse reste pure)
- [ ] `GroupMapsByBase` (ou une fonction sœur) regroupe les map_id de même nom ET de même clé de
      module, sans variante ; les groupes de variantes existants absorbent ces identifiants
- [ ] Tests unitaires : intégrée republiée = un groupe ; deux versions Forge = deux groupes ; clé
      absente = seul ; variante + republiée = un seul groupe
- [ ] Commentaire de `GroupMapsByBase` et de `map_variants.toml` mis à jour (contrat, pas historique)
- Gate : `go test ./internal/analysis/... ./internal/platform/duckdb/ ./internal/service/... ./internal/archlint/` + golangci ratchet

### E2 — Clôture
- [ ] `go test ./internal/...`, lint ratchet, thought_log, ce plan statué
- [ ] Liste « À vérifier à l'écran » (JGtm : Shogun, Bazaar, Houseki)

## Mesures (E0, 2026-10-26)

Sources, sans rien ouvrir en écriture (le serveur local tient les bases) : instantané parquet `warehouse/snapshots/
v...263/shared/match_registry.parquet` + `match_participants.parquet` (9 594 matchs, 304 map_id) pour les noms retenus
(même règle que `nomDeCarteRetenuSQL`) et les comptes ; sauvegarde sans verrou `data/backups/avant_recuisson_98_2026-10-16/
shared_matches_v2.duckdb`, ouverte en lecture seule, pour les positions (`kill_positions_latest`, 268 194 lignes, 2 600
matchs) ; catalogues versionnés du worktree pour les clés (`map_quant_bounds.json`, `map_callouts.json`,
`map_backgrounds/`). Outils jetables non versionnés (`cmd/_e0sql`, `cmd/_e0republiees`), supprimés en clôture. Colonnes
« parties » = tous joueurs / JGtm (xuid 2533274823110022).

**M1 - Noms portés par plusieurs map_id.** 56 noms (135 map_id). 20 sont des cartes intégrées (une clé de module unique
pour tous leurs identifiants) ; les 36 autres sont des cartes Forge (le catalogue des bornes les envoie vers un CANEVAS
`fo*`, qui n'a ni fond ni zones propres, ou ne les connaît pas).

**M2 - Clé de module.** Définie comme l'essai 1 de `callouts_resolution.go` : le catalogue des bornes rend un module
pour le map_id, puis pour le nom retenu, ET ce module a une entrée `maps` au catalogue versionné des zones (ce qui
écarte tous les canevas). Résultat : 20 clés (btb_drydock, btb_engine, btb_exiled, btb_fragmentation, btb_highpower,
catalyst, chasm, ctf_aquarius, ctf_bazaar, ctf_breaker, ctf_forbidden, ctf_illusion, forest, ridgeline, sgh_blueprint,
sgh_crystalcaves, sgh_interlock, sgh_streets, va_behemoth, va_launchsite), soit exactement les 20 fonds figés keyés par
module. Aucun identifiant d'une carte intégrée n'a de fond propre keyé par son map_id, ni d'entrée par map_id au catalogue
des bornes ou des zones (`maps_by_id`) : rien n'indique une carte Forge qui porterait le nom d'une carte intégrée.

**M3 - Tableau (une ligne par nom ; identifiants abrégés à 8 caractères, parties tous joueurs / JGtm).**

| Nom | Identifiants (parties) | Clé de module | Type | Recouvrement des positions de morts (p1-p99) |
|---|---|---|---|---|
| Aquarius | 33c0766c 324/2 ; c395f3ac 52/44 | ctf_aquarius | intégrée | 1 paire(s) non mesurable(s) (un côté sans film) |
| Aqueduct | f8e1938f 1/0 ; 0b22acc2 1/0 | aucune | Forge (hors catalogue des bornes) | sans objet |
| Argyle | dd600260 124/12 ; 2e35d8c0 4/0 | aucune | Forge (canevas fo11_blank) | sans objet |
| Arrival | 5e3e78c5 4/0 ; e0bd2b53 1/0 ; beaf8ac9 1/0 | aucune | Forge (canevas fo03_space) | sans objet |
| Bazaar | 298d5036 173/1 ; 3e1e4cec 59/52 | ctf_bazaar | intégrée | 1 paire(s) non mesurable(s) (un côté sans film) |
| Behemoth | 53136ad9 109/0 ; e9a5a982 54/44 ; c3eabce0 18/0 | va_behemoth | intégrée | mesuré e9a5a982/c3eabce0 100.0% ; 2 paire(s) non mesurable(s) (un côté sans film) |
| Breaker | e6cbfe01 104/0 ; 6a147baf 21/0 | ctf_breaker | intégrée | 1 paire(s) non mesurable(s) (un côté sans film) |
| Catalyst | e859cf75 139/0 ; f7e8cde9 55/44 ; 11ca6a12 7/0 ; 4b6a9a35 3/0 | catalyst | intégrée | 6 paire(s) non mesurable(s) (un côté sans film) |
| Chasm | fc1ced39 97/0 ; a455572d 47/40 | chasm | intégrée | 1 paire(s) non mesurable(s) (un côté sans film) |
| Cliffhanger | 81274d6f 78/0 ; 5324364b 60/49 | ridgeline | intégrée | 1 paire(s) non mesurable(s) (un côté sans film) |
| Cliffside | 4bffd021 21/17 ; c9d416e2 7/0 | aucune | Forge (canevas fo05_desert) | sans objet |
| Critical Dewpoint | bae4df14 16/10 ; f718abc4 4/0 | aucune | Forge (canevas fo11_blank) | sans objet |
| Curfew | 63d634be 26/16 ; 35dcc4fc 3/0 ; 54fdd41b 3/0 | aucune | Forge (canevas fo11_blank) | sans objet |
| Daimyo | 4161cf3c 5/0 ; 26874260 1/0 ; 0590f221 1/0 ; f23f1c8d 1/0 | aucune | Forge (hors catalogue des bornes) | sans objet |
| Dead Water | 1ada8f92 4/0 ; 73966558 1/0 | aucune | Forge (hors catalogue des bornes) | sans objet |
| Deadlock | 08607bf4 112/0 ; 6f82df05 24/2 | btb_drydock | intégrée | mesuré 08607bf4/6f82df05 98.2% |
| Diminished | 50a1a3b5 13/0 ; ad05da60 1/0 | aucune | Forge (hors catalogue des bornes) | sans objet |
| Disciple | e7df2b12 8/0 ; 525451ca 4/1 | aucune | Forge (canevas fo11_blank) | sans objet |
| Domicile | 921aebb1 25/18 ; f631b010 3/0 | aucune | Forge (canevas fo05_desert) | sans objet |
| Dynasty | cfd90b63 23/19 ; 90cd321d 1/0 | aucune | Forge (canevas fo08_wetland) | sans objet |
| Exiled | 31051f43 19/0 ; 9bb8d9df 16/0 ; 78905252 5/0 | aucune | Forge (canevas fo05_desert) | sans objet |
| Forbidden | 87c03bfd 57/43 ; 7e8b3a2c 18/0 ; db2b5f9c 5/0 | ctf_forbidden | intégrée | 3 paire(s) non mesurable(s) (un côté sans film) |
| Forest | e8d56863 69/43 ; 619bea21 65/0 | forest | intégrée | 1 paire(s) non mesurable(s) (un côté sans film) |
| Fragmentation | 4f196016 103/0 ; 068f41f4 18/0 | btb_fragmentation | intégrée | 1 paire(s) non mesurable(s) (un côté sans film) |
| High Ground | bb7b78ae 17/11 ; 3d52899a 9/0 | aucune | Forge (canevas fo08_wetland) | sans objet |
| Highpower | c494ef7c 139/1 ; 33c6505d 25/0 | btb_highpower | intégrée | 1 paire(s) non mesurable(s) (un côté sans film) |
| Houseki | cf034ec8 17/14 ; 6439625e 1/1 | aucune | Forge (canevas fo09_academy) | sans objet |
| Illusion | 9e821f5e 65/56 ; 86ef3b1c 7/0 | ctf_illusion | intégrée | 1 paire(s) non mesurable(s) (un côté sans film) |
| Immolate | 8e801fa1 2/0 ; 47823612 1/0 ; b3518c8b 1/0 ; 014f85f6 1/0 | aucune | Forge (canevas fo11_blank) | sans objet |
| Isolation | 01af558d 27/18 ; 5dbe246f 7/0 | aucune | Forge (canevas fo08_wetland) | sans objet |
| Kaiketsu | 98a83f87 15/10 ; 24c7bfb5 1/0 | aucune | Forge (canevas fo05_desert) | sans objet |
| Kiken'na | df7dbf08 15/12 ; 6ae862f6 1/0 | aucune | Forge (canevas fo08_wetland) | sans objet |
| Kusini Bay | 6a995886 26/0 ; 89f3b8ad 25/0 ; 7e1cb2dd 19/0 | aucune | Forge (canevas fo11_blank) | sans objet |
| Last Broadcast | 16b21e15 4/0 ; 74ef5a59 3/0 | aucune | Forge (hors catalogue des bornes) | sans objet |
| Launch Site | 56a11b8c 48/0 ; 5646ce03 24/24 | va_launchsite | intégrée | 1 paire(s) non mesurable(s) (un côté sans film) |
| Live Fire | b6aca0c7 432/2 ; 6c01f693 58/48 ; 7de09dad 17/0 | sgh_interlock | intégrée | 3 paire(s) non mesurable(s) (un côté sans film) |
| Megapolis | 0c299a3a 7/0 ; e67f1328 3/0 | aucune | Forge (hors catalogue des bornes) | sans objet |
| Oasis | 6aa0a116 84/1 ; 7aa6fed5 29/0 | btb_exiled | intégrée | 1 paire(s) non mesurable(s) (un côté sans film) |
| Perilous | c5ac9f12 14/11 ; 2cdf7097 3/0 ; 84254700 1/0 | aucune | Forge (canevas fo08_wetland) | sans objet |
| Prism | 2fdb8370 60/47 ; 92d23264 13/0 | sgh_crystalcaves | intégrée | 1 paire(s) non mesurable(s) (un côté sans film) |
| Rat's Nest | 133c0185 18/0 ; 0140cf1b 17/0 ; f2908a90 11/0 ; 97696a83 1/0 | aucune | Forge (canevas fo08_wetland) | sans objet |
| Recharge | 8420410b 406/1 ; 2b6d2baf 52/44 ; 3adbac00 28/0 | sgh_blueprint | intégrée | 3 paire(s) non mesurable(s) (un côté sans film) |
| Ronin | aab54656 9/0 ; f459867d 8/1 | aucune | Forge (canevas fo08_wetland) | sans objet |
| Salvation | cd08bc7a 26/11 ; f633db01 5/0 | aucune | Forge (canevas fo11_blank) | sans objet |
| Scarlett's Landing | 2413e279 8/0 ; 79042fc0 2/0 ; 1612a412 2/0 | aucune | Forge (canevas fo08_wetland) | sans objet |
| Scarr | 247637f8 72/0 ; ccb81e1f 17/0 | btb_engine | intégrée | 1 paire(s) non mesurable(s) (un côté sans film) |
| Shiro | 2890782c 24/16 ; 2962c4e0 1/0 | aucune | Forge (canevas fo05_desert) | sans objet |
| Shogun | 33075df7 15/9 ; 8f51ccb9 2/1 | aucune | Forge (canevas fo11_blank) | sans objet |
| Smallhalla | 98783453 17/5 ; 2bfd7643 4/0 ; c48750e8 1/0 | aucune | Forge (canevas fo08_wetland) | sans objet |
| Solution | ee43d273 11/9 ; 70a5226e 2/0 ; 35c4680c 1/0 | aucune | Forge (canevas fo05_desert) | sans objet |
| Starboard | ae7b2540 37/0 ; 7a9265af 30/18 ; 1ce029c4 7/0 ; 50771a22 2/0 | aucune | Forge (canevas fo03_space) | sans objet |
| Streets | f0a1760f 415/2 ; 9c7b0b0f 54/47 | sgh_streets | intégrée | 1 paire(s) non mesurable(s) (un côté sans film) |
| The Pit | 648ae7aa 26/20 ; ec1d4f25 10/0 | aucune | Forge (canevas fo09_academy) | sans objet |
| Vagabond | 105f5d84 24/14 ; 5612030c 3/0 | aucune | Forge (canevas fo08_wetland) | sans objet |
| Vallaheim | 688c2033 28/0 ; 54c26d80 16/0 ; 74ced3e5 8/0 | aucune | Forge (hors catalogue des bornes) | sans objet |
| Waterworks | 9d1f3a8d 16/0 ; 0661af5e 1/0 | aucune | Forge (hors catalogue des bornes) | sans objet |

**M4 - Recouvrement.** Les positions de morts n'existent que pour les matchs dont le film est décodé, c'est-à-dire
presque seulement ceux de JGtm. Les 20 cartes intégrées donnent 33 paires d'identifiants de même clé ; seules 2 ont des
positions (au moins 100) des deux côtés : Behemoth e9a5a982 / c3eabce0 = 100,0 % et Deadlock 08607bf4 / 6f82df05 =
98,2 % (boîtes de 241 et 1 764 positions). Les 31 autres ont un côté sans film : non mesurables, leur seul fondement est
la clé de module (M2). Aucune paire mesurée sous 90 % : pas d'arrêt. Écart assumé au plan, signalé au compte rendu : il
prévoyait l'arrêt si la mesure était impossible ; elle l'est pour 31 paires sur 33, faute de film, ce que le plan des
variantes avait déjà constaté pour les doublons de carte (M4 de ce plan). L'exécution a poursuivi sur la clé de module
seule, qui est le critère de la règle.

**M5 - Forge, séparées par la règle.** Shogun (33075df7 : 15 parties dont 9 JGtm ; 8f51ccb9 : 2 dont 1), Houseki
(cf034ec8 : 17 dont 14 ; 6439625e : 1 dont 1) et les 34 autres noms Forge à plusieurs identifiants (Argyle, Starboard,
Salvation, Dynasty, Shiro, Vagabond, Isolation, The Pit, Perilous...) n'ont PAS de clé de module : leurs identifiants
restent chacun une carte. Le repli par nom de l'index des fonds les ferait converger vers le fond de l'identifiant
cuit (33075df7 pour Shogun), mais c'est précisément le repli que la règle écarte.


## Découvertes (hors périmètre, non traitées)
