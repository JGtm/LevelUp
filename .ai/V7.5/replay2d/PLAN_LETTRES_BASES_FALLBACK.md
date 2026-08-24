# PLAN — Lettres A/B/C des bases (fallback par ordre canonique mesuré)

Date : 2026-08-24. Origine : backlog Notion REPLAY 2D item 2 (libellés A/B/C du HUD),
réfuté dans les données décodées ; arbitrage utilisateur du 24/08 : « en fallback, faute de
mieux, un ordre stable nous suffit à dire A, B ou C » — route « les deux en parallèle »
(ce lot + une RE Ghidra séparée qui cherche la règle vraie ; si elle contredit ce fallback
AVANT fusion, on corrige ici avant l'écran).

Branche : `wt/lettres-bases` (worktree dédié, base feat/v75 `b16ba17e5`). Exécution sous le
contrat du skill `plan-execution` (ordre strict, gates, statuts, zéro fix hors périmètre).
Ce plan est commité PAR le lot (premier commit).

## Objectif et critère de succès

Étiqueter A/B/C les zones des modes à BASES SIMULTANÉES (Strongholds, Total Control) dans
le rejeu 2D, par l'ordre canonique le plus plausible du moteur (les slots ti=13, stables
entre matchs d'une même carte), publié par match, rendu au canvas. Succès = les témoins
re-cuits portent l'ordre, la lettre s'affiche sur les zones (et NULLE PART ailleurs :
jamais KOTH, jamais CTF), les artefacts anciens dégradent sans lettre, gates verts.
Le verdict final (« nos lettres = celles du jeu ? ») appartient au gate Theater de
l'utilisateur — ce lot fournit les témoins et l'item de planche, il ne tranche pas.

## Décisions TRANCHÉES (ne pas rouvrir)

1. Source de l'ordre : l'ordre des SLOTS ti=13 tel que le chantier zones l'a mesuré
   (3 slots / 3 ids identiques sur les 2 matchs Bastion). Vérifier sur pièces d'abord
   (phase 0) : si l'ordre n'est PAS stable inter-match sur une même carte, STOP et CR —
   le fallback tombe, pas de bricolage de remplacement.
2. Publication : champ OPTIONNEL sur les données de zone servies (pas d'incrément de
   `SchemaVersion` — règle du dépôt : un champ optionnel n'incrémente pas, précédent daté
   du 24/08 dans le même dépôt). Nom proposé : `letterRank` (0/1/2), au niveau zone.
3. Rendu : lettre majuscule (A=0, B=1, C=2) à l'ancre de la zone, style des libellés de
   callouts (blanc cerné de noir, insensible au thème), UNIQUEMENT quand le document dit
   des zones simultanées. La règle « aucun texte » des calques zones/objectifs est LEVÉE
   pour ce seul glyphe : le garde-test qui interdit fillText/strokeText est amendé avec
   justification datée (décision utilisateur 2026-08-24, calibrage Theater à suivre) —
   il continue d'interdire tout AUTRE texte.
4. Pas de lettre dans les fiches ni le fil des morts dans ce lot (rendu canvas seulement).
5. Artefact ancien sans champ => aucune lettre, aucun avertissement (dégradation muette).

## Hors périmètre (fermé)

- KOTH (la colline n'a pas de lettre en jeu), CTF, Oddball.
- La preuve que nos lettres = celles du jeu (lot RE Ghidra séparé + gate Theater user).
- Toute re-cuisson de masse (bombe RAM consignée) ; le backfill des artefacts anciens.
- ReplayCanvas.tsx (cliquet 797) : le rendu vit dans le calque zones existant.

## Phase 0 — L'ordre, vérifié sur pièces

- [x] 0.1 Retrouver dans `internal/analysis/replay/zone_states*.go` (et la construction
      slot->zone du lot C-bis) où vit l'index de slot ti=13 par zone, et vérifier qu'il
      est disponible au moment de la publication.
- [x] 0.2 Mesurer la stabilité : sur les 2 matchs Bastion du corpus (et tout autre
      Strongholds/TC disponible dans `data/cache/film_chunks` du principal), l'ordre des
      slots donne-t-il la MÊME permutation zone->rang sur une même carte ? Instrument
      jetable gaté par env var si besoin (patron TI47_FILM).
- [x] 0.3 Verdict de phase : ordre stable => on publie ; instable => STOP + CR.

Gate 0 : permutation identique sur les matchs d'une même carte, chiffres au CR.

### Journal de la phase 0 (2026-08-24)

**0.1 — l'index est là où il faut.** La carte zone -> slot de jauge est `gaugeSlot`, rendue
par `pairGaugeSlots` dans `zone_states_owner.go:40` (`zoneOwnerStates`), c'est-à-dire
EXACTEMENT au point où les `ZoneState` sont construits, et le catalogue du match
(`in.Zones`) y est disponible pour juger la complétude. Rien à déplacer, rien à recalculer.

**0.2 — GATE 0 TENU : 8 cartes, 17 films, 8/8 permutations identiques.** Instrument
`lettres_ordre_research_test.go` (mesure seulement) : il appelle `pairGaugeSlots` et
`pairOwnerSlots` DE PRODUCTION sur les captures nommées du statborg, lignes de match gelées
relues des exports versionnés du registre — aucune base ouverte.

| carte | films | permutation (zone -> lettre) | slots de jauge (propriétaire) |
|---|---|---|---|
| vagabond | 3 | z0=C, z1=A, z2=B | 1532>z1 (p1530), 1537>z2 (p1535), 1542>z0 (p1540) |
| forest | 2 | z0=A, z1=C, z2=B | 1413>z0 (p1411), 1418>z2 (p1416), 1423>z1 (p1421) |
| fortress | 2 | z0=C, z1=B, z2=A | 1445>z2 (p1443), 1450>z1 (p1448), 1455>z0 (p1453) |
| houseki | 2 | z0=B, z1=C, z2=A | 1518>z2 (p1516), 1523>z0 (p1521), 1528>z1 (p1526) |
| illusion | 2 | z0=B, z1=C, z2=A | 1603>z2 (p1601), 1608>z0 (p1606), 1613>z1 (p1611) |
| kaiketsu | 2 | z0=A, z1=B, z2=C | 1558>z0 (p1556), 1563>z1 (p1561), 1568>z2 (p1566) |
| origin | 2 | z0=A, z1=B, z2=C | 1525>z0 (p1523), 1530>z1 (p1528), 1535>z2 (p1533) |
| prism | 2 | z0=C, z1=A, z2=B | 1415>z1 (p1413), 1420>z2 (p1418), 1425>z0 (p1423) |

Trois faits que la table porte, et qui décident de la règle de publication :

1. **La permutation n'est PAS l'ordre spatial**, et elle diffère d'une carte à l'autre
   (A,B,C sur Kaiketsu et Origin ; C,A,B sur Vagabond et Prism ; B,C,A sur Houseki et
   Illusion). C'est précisément ce qui rend le fallback utile : le rang spatial du
   catalogue ne dit rien de la lettre.
2. **Les slots `ti=13` d'une carte de Bastion forment des BLOCS RÉGULIERS de pas 5**, un
   par zone, `[propriétaire, canal neutre, jauge]` aux offsets 0, 1, 2 — sur les 8 cartes,
   sans exception. L'ordre des jauges est donc l'ordre des blocs, c'est-à-dire l'ordre
   d'allocation du moteur au chargement de la carte : la reproductibilité inter-match n'est
   pas une coïncidence de vote, elle est structurelle.
3. **Sur Vagabond, la table reproduit exactement la phase 2a** (1532 -> zone 1, 1537 ->
   zone 2, 1542 -> zone 0) avec ses chiffres de captures (59/71 et 66/77) : l'instrument
   lit bien ce que la production publie.

**0.3 — verdict : ORDRE STABLE, on publie.** La règle retenue en conséquence :
la lettre n'est publiée que si les zones appariées couvrent TOUT le catalogue de la carte
(bijection). Sans cette garde, une zone muette décalerait les lettres des suivantes — le
cas existe (`aaaf6c76` sur Kaiketsu : 0 capture attribuée, donc 0 zone appariée).

**Négatif gardé — un oracle refusé.** Une première écriture de l'instrument remplaçait les
captures nommées par la GRAPPE des positions (méthode du volet colline), pour se passer du
roster. Elle retrouve les trois slots canoniques de Vagabond, mais donne AUSSI une zone au
quatrième bloc (1545-1547, l'objet de MODE, dont les rampes suivent toutes les captures de
la carte) et le fait gagner l'élection sur un film sur trois. La grappe ne sépare pas une
prise d'un passage ; les captures nommées, si.

**Incident mémoire, et ce qu'il a changé (2026-08-24).** La première version de l'instrument
bouclait sur le corpus dans UN processus. Laissée orpheline par un redémarrage de l'hôte,
elle est montée à 18,4 Gio résidents / 95 Gio de commit et a mis la machine à genoux. Cause
racine : `objectiveevents.NamedEvents` -> `incrementTimes` émet un événement PAR UNITÉ de
compteur (la bombe déjà au registre des reports, OOM ~26 Gio sur `51101d1d`) — du code de
PRODUCTION, hors périmètre de ce lot. L'instrument a été réécrit : un film = un processus,
sentinelle mémoire (arrêt net au-delà de 3 Gio de tas), plafonds francs sur chaque
collection, refus d'emblée des variantes hors zones simultanées, et le pic de tas publié
AVEC la mesure. **Régime constaté sur les 17 films : 32 à 123 Mio de tas, 40 à 105 s.**

## Phase 1 — Publication (champ optionnel, pas de bump)

- [ ] 1.1 `letterRank` optionnel au niveau zone dans la charge servie (là où les zones
      partent déjà au client — suivre l'existant de zoneStates/mapObjectives, pas de
      nouveau canal), omis quand l'ordre n'est pas établi.
- [ ] 1.2 Contrat OpenAPI + `make generate-types` (generated.ts) ; si tableau nullable,
      passer par la liste NULLABLE_ARRAYS existante.
- [ ] 1.3 Re-cuire les témoins nécessaires SEULEMENT (2-3 : les 2 Bastion + 1 Total
      Control s'il y en a un au cache), UN à la fois via `cmd/replay-build --facts`,
      anciens sauvegardés sous le motif `_backup_*` existant. La bombe RAM consignée a
      frappé un comp de mode-score sur un film CTF : si un re-bake dérape en mémoire,
      tuer, consigner, continuer avec les témoins qui passent.
- [ ] 1.4 Tests Go du package touché (table : zone avec rang, sans rang, mode non
      simultané => absent).

Gate 1 : témoin re-cuit porte `letterRank` cohérent avec la mesure 0.2 ; artefact schéma
antérieur servi tel quel => champ absent, aucun crash contrat.

## Phase 2 — Rendu web

- [ ] 2.1 Lettre dessinée à l'ancre de zone dans le calque zones (zoneStatesLayer /
      objectivesLayer selon qui possède l'ancre — suivre l'existant), style libellé
      callout, seulement si `letterRank` présent.
- [ ] 2.2 Amender le garde-test « ni fillText ni strokeText » : il autorise LE glyphe de
      lettre (une seule chaîne d'un caractère, A-C), avec le commentaire daté prescrit en
      décision 3 ; il continue d'échouer sur tout autre texte.
- [ ] 2.3 Tests vitest : lettre présente avec rang, absente sans rang, jamais en KOTH ;
      garde amendé testé dans les deux sens.
- [ ] 2.4 Item de planche pour le calibrage : préparer le texte de l'item (matchs témoins,
      ce que l'utilisateur doit comparer au Theater) et le livrer au CR — la republication
      de la planche appartient au superviseur.

Gate 2 (depuis apps/web du worktree, node_modules/.tmp purgé) : `npx tsc -b` 0 erreur ;
`npx vitest run src/features/match-replay` 0 échec ; `npx eslint src/features/match-replay`
0 erreur nouvelle.

## Garde-rails d'exécution

- Commandes `go` : UNE à la fois, GOCACHE privé (`<worktree>/.gocache`, jamais commité) ;
  `npm ci` dans apps/web du worktree ; vitest hors sandbox si besoin.
- Données du principal (`c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/`) :
  lecture seule, SAUF l'écriture des artefacts témoins re-cuits (JSON du cache replays,
  anciens sauvegardés) — aucune écriture DuckDB nulle part.
- Coordination : une autre session a des modifications non commitées sur `openapi.yaml` /
  `generated.ts` dans le principal — ne pas s'en préoccuper (fusion gérée par le
  superviseur, régénération au besoin) ; ce lot ne touche PAS `SchemaVersion`.
- Ne pas toucher `.ai/thought_log.md` ni `REGISTRE_REPORTS.md` du principal.

## Découvertes

(consigner ici — rien corriger)

## CR attendu

Statut par item, mesures 0.2, sorties des gates, liste des témoins re-cuits (avant/après),
texte de l'item de planche, commits `lettres(pN): ...` (jamais `git add -A`), aucun push.
