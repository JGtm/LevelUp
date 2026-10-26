# Lot C-bis — phase 2b : la PUBLICATION. `zoneStates` entre au document, et le rendu teinte les zones

> Perimetre : CB.2b.1, CB.2b.2, CB.2b.3 de `LOTCBIS_ARBITRAGE_PHASE1.md` §« Phase 2b ».
> Acquis repris sans les refaire : `LOTCBIS_PHASE0/1.md` (grammaire ti=13), `LOTCBIS_PHASE2A.md`
> (appariement, proprietaire, colline). Branche `wt/zones-ti13-p2b`, base `32ee72107`.
> Gates : `LOTCBIS_p2b_gates.log`. Temoins : `lotCbis/temoin_<short8>.log`.

## 0. Le resultat, en une phrase

**Le document porte l'etat des zones, et il se verifie sur la forme publiee** : sur les deux
Bastion du corpus, **57/59 = 96,6 %** et **64/66 = 97,0 %** des captures nommees attribuees sont
suivies d'un intervalle appartenant a l'equipe du capteur (seuil 90 %) ; sur le KOTH, **20
periodes actives couvrent 4 975 des 5 343 frames = 93,1 %** (seuil 80 %). Le calque pese **+0,22 %**
de l'artefact.

## 1. Ce qui est ecrit — fichiers et lignes

| fichier | L | ce qu'il porte |
|---|---|---|
| `internal/analysis/filmdec/zone_state_scan.go` (NEUF) | 289 | LE BALAYAGE de `ti=13` en production : `ScanFilmManagedProperties`, desers de production par hook, **zero copie de grammaire** |
| `internal/analysis/replay/document_zones.go` (NEUF) | 165 | la FORME (`ZoneState`, `ZoneSpan`, `ZonesCoverage`) et la chronique du schema **16** (ecrite en 15, renumerotee a la fusion — §10) |
| `internal/analysis/replay/zone_states.go` (NEUF) | 338 | la REGLE : series par tag, fenetres, catalogue renumerote, echelle de la jauge |
| `internal/analysis/replay/zone_states_owner.go` (NEUF) | 362 | le volet PROPRIETAIRE : appariement des slots, intervalles, controle |
| `internal/analysis/replay/zone_states_hill.go` (NEUF) | 166 | le volet COLLINE : periodes de garde par la grappe des positions |
| `internal/analysis/replay/build_zones.go` (NEUF) | 97 | le CABLAGE (`decodeFilmZoneReads`, `attachZoneStates`) et son journal |
| `internal/replaybuild/zones.go` (NEUF) | 165 | le CATALOGUE du match, dans l'ordre ou le service le sert |
| `internal/analysis/replay/{build,document,coverage}.go` | +12/+11/+10 | `Options.Zone`, `ReplayDocument.ZoneStates`, `Coverage.Zones`, schema 15 -> **16** (14 -> 15 avant la fusion de l item 4, cf. §10) |
| `contracttest/replay_contract_test.go` | +30 | `wantReplayDocumentFields` 35 -> **36**, 3 schemas de plus, chronique |
| `apps/web/.../zoneStatesLayer.ts` (NEUF) | 181 | le calque VIVANT : `zoneStateAt`, teinte, surbrillance, arc de jauge |
| `apps/web/.../useZoneStates.ts` (NEUF) | 63 | le pont camp -> encre, et les zones dans l'ordre servi |
| `apps/web/.../objectivesLayer.ts` | 360 | le calque STATIQUE, allege ; `traceZonePath` EXPORTE (une seule geometrie) |

Tests : `zone_states_test.go` (12 cas sans film, dont le VOLUME d'un vrai match),
`replaybuild/zones_test.go` (5 cas, dont la garde de jointure), `zone_state_p2b_temoin_test.go`
(le temoin sur film reel), `objectivesLayer.test.ts` (+8 cas).

## 2. La forme publiee (extrait REEL, `7344d24f`)

```json
{"zoneRef":2,"key":664094208,"spans":[
  {"t0":194,"t1":424,"owner":0,"progress":0.9998104,"active":false},
  {"t0":425,"t1":515,"owner":1,"progress":0.99971914,"active":false},
  {"t0":516,"t1":778,"owner":0,"progress":0.99999046,"active":false}]}
```

et sa couverture : `{"method":"captures+geometry","roles":"strongholds_zone","catalog":3,
"slots":19,"paired":3,"unpaired":2,"captures":71,"attributed":59,"ownerChecked":46,
"ownerAgreed":46,"spans":39,"hillPeriods":0,"unknownOwner":0}`.

En KOTH (`01e1f945`) : `{"zoneRef":0,"spans":[{"t0":561,"t1":1118,"owner":null,
"progress":0.9998424,"active":true}, ...]}`, methode `positions+geometry`, roles
`strongholds_zone,extraction_zone`.

## 3. Les chiffres de controle

| film | mode | captures | attribuees | **captures -> intervalle du capteur** | proprietaire (couverture) | intervalles | taille |
|---|---|---|---|---|---|---|---|
| `7344d24f` | Strongholds | 71 | 59 | **57/59 = 96,6 %** (seuil 90 %) | **46/46 = 100 %** | 39 | +2 944 o (**+0,221 %**) |
| `696a9d7c` | Strongholds | 77 | 66 | **64/66 = 97,0 %** | **51/51 = 100 %** | 37 | +2 792 o (**+0,215 %**) |
| `01e1f945` | KOTH | 0 | — | sans objet | — | 20 (periodes) | — |

KOTH, clause de couverture : **4 975 frames actives / 5 343 = 93,1 %** pour 80 % exiges, sur 6
zones du catalogue appariees par la grappe.

**Le controle se lit SUR LA FORME PUBLIEE**, pas sur les compteurs internes : pour chaque capture
nommee attribuee geometriquement, le temoin relit `zoneStates[].spans` a la frame de la capture
et compare `owner` a l'equipe du capteur (roster). C'est exactement ce que le client dessinera.

**Le balayage lit les MEMES BITS que la mesure de la phase 2a**, et c'est verifie a l'unite pres :
`7344d24f` rend **36 082 records ancres, 10 841 marches abouties, 3 702 chainees** — la phase 2a
(recopie de grammaire dans des fichiers de test) mesurait 36 082 / 10 841 et 3 662 + 40 = 3 702
chainees. La promotion en production n'a rien change a la lecture, elle a supprime la copie.

## 4. Trois corrections que la mesure a imposees (et que le code porte en commentaire)

1. **La cle de jointure n'etait pas joignable.** `AttributeZones` rend `SpatialRank` et
   `InstanceID` ; or `instance_id` vaut **0 sur TOUTES les entrees** du catalogue versionne, et le
   rang est attribue PAR ROLE. Le catalogue est donc RENUMEROTE sur l'ordre servi
   (`zoneCatalogOf`) : le rang devient l'index de `mapObjectives.zones`. Avant : 0 capture
   appariee sur 59 attribuees, et le film de Bastion basculait sur la methode « colline ».
2. **L'election du canal de proprietaire par simple voisinage elit le mauvais canal.** Une carte
   de Bastion porte, par zone, le proprietaire canonique (11 a 16 emissions, {0, 1} seulement) ET
   un canal bavard ou `0xFFFFFFFF` domine (32 a 39 emissions). Compter les changements VOISINS
   d'une capture elit le second — il en a plus (20-21 contre 14) : controle a **45,8 %**, zones
   qui basculent sans arret vers « personne ». Noter l'**ACCORD AVEC LE ROSTER** (la valeur qui
   SUIT la capture est-elle l'index d'equipe du capteur ?) elit le canal canonique : **96,6 %**.
   L'inventaire des canaux est publie dans le temoin, film par film.
3. **La jauge ne parcourt pas la plage declaree par le deser.** Ramenee lineairement sur
   [-100, +100], toute valeur reelle vaut **0,50 a trois decimales pres** — un arc a moitie plein
   en permanence. `progress` est donc la part de l'EXCURSION MESUREE de la jauge de cette zone sur
   ce match (1 = le sommet atteint, une capture menee a son terme). L'echelle est relative, et le
   champ le dit.

## 5. Le contrat

- `SchemaVersion` **15 -> 16** (ecrit 14 -> 15, renumerote a la fusion — §10), chronique dans `document_zones.go` et raison ecrite dans
  `structure_test.go` (le garde exige une justification, pas un numero).
- `wantReplayDocumentFields` **35 -> 36** + 3 schemas (`ZoneState`, `ZoneSpan`, `ZonesCoverage`).
- `api/openapi.yaml` REGENERE (`make openapi-gen`), `generated.ts` regenere.
- Garde web `_ListeExhaustive` : elle a **ROUGI puis PASSE** — 4 erreurs `tsc` exigeant
  `zoneStates` et `zoneStates[].spans` dans les deux listes, ce qui est le comportement attendu.
- `coverage.zones.roles` est une CHAINE et non un tableau : ce temoin de jointure que rien ne
  parcourt aurait sinon fait entrer un tableau nullable de plus dans la frontiere du client.
- Golden `assembly_000d5950.golden` : **une ligne** change (`schema 15` -> `schema 16` apres fusion ; `14 -> 15` avant). Le film
  temoin est un Slayer, sans zone : le calque n'y publie rien, et c'est ce que le golden montre.

## 6. Le rendu (CB.2b.3)

Le calque VIVANT vit dans `zoneStatesLayer.ts`, a cote du statique : ici la geometrie (cuite une
fois hors ecran), la l'etat (repeint a chaque image, comme les socles d'arme). Les deux tracent la
MEME forme — `traceZonePath` est exporte, jamais recopie.

- zone TENUE : remplissage + lisere a l'encre du camp (`team-ally` / `team-enemy` par le pont
  `team_side` « t{N} » de la ligne « moi ») ; **camp inconnu = encre neutre**, jamais devinee ;
- zone que PERSONNE ne tient : **lisere seul** ;
- colline ACTIVE : remplissage et trait renforces ; les zones sans etat a cette frame ne sont pas
  repeintes et gardent le trait faible du statique — elles paraissent estompees par contraste ;
- `progress` : arc trace HORS de la forme, qui se referme a mesure de la capture.

**Les pulses d'action RESTENT**, et ce n'est pas un doublon : ils marquent l'INSTANT (un anneau
qui s'ouvre et s'eteint), l'etat decrit une DUREE. Sur les temoins, la bascule de teinte tombe
exactement sur le pulse — l'un annonce, l'autre installe.

`ReplayCanvas.tsx` reste **a 858 lignes**, son plafond : les encres d'objectif sont parties dans
`useZoneStates` AVANT l'ajout, comme le cliquet l'exige.

## 7. Note pour la planche (ce que l'utilisateur doit voir)

- **`7344d24f` et `696a9d7c` (Bastion, Vagabond)** : les trois zones sont TEINTEES, et leur teinte
  BASCULE a l'instant des captures — c'est le pulse qui annonce, la teinte qui suit. La zone de
  rang 2 est prise a la frame 194 (19,4 s) par l'equipe 0, reprise a 425 par l'equipe 1. Entre
  deux prises, une zone que personne ne tient garde son lisere seul.
- **`01e1f945` (KOTH, Catalyst)** : une seule colline est en SURBRILLANCE a la fois, et elle
  change de place au fil du match (zones 1, 0, 3, 6, 4, 7 dans cet ordre). **RESERVE** : le
  service ne sert AUCUNE zone en KOTH aujourd'hui (la table de roles du titre n'en declare pas,
  parce que le catalogue de formes n'a aucun role de colline) — l'artefact porte donc la mesure,
  et l'ecran ne la montrera qu'une fois ce role existant. C'est ecrit dans `zones.go` et publie
  dans `coverage.zones.roles`.

## 8. Statut des items

- [x] **CB.2b.1** — assemblage Go pur : scanner promu en production (zero copie), regle,
  forme, cablage, catalogue par match. Tests sans film : 12 cas, dont le volume d'un vrai match.
- [x] **CB.2b.2** — contrat : schema **16** (15 avant la fusion de l item 4), 36 champs, OpenAPI et types regeneres, garde web rougie
  puis verte, golden justifie, **3 temoins re-cuits** avec faits et chiffres publies.
- [x] **CB.2b.3** — rendu : teinte par proprietaire, colline active, arc de progression, helper
  pur `zoneStateAt` teste ; pulses conserves (constat ecrit) ; gates web verts.

## 9. Decouvertes (hors perimetre — notees, NON traitees)

1. **CLOSE (2026-08-18, item 4 : `deathProgressions` plafonne — §11).** LE CALQUE DU DRAPEAU PARTAIT EN VRILLE SUR UN FILM DE BASTION. `cmd/replay-build --facts` sur
   `7344d24f` n'a JAMAIS rendu d'artefact : le processus atteint **19 a 22 Go** de memoire
   residente et reste bloque > 15 min, systematiquement, apres le journal des socles d'arme —
   c'est-a-dire dans `attachFlagCarries`. **Le calque des zones est hors de cause** : un temoin
   cuit avec les MEMES faits mais une variante `Slayer:Arena` (donc aucune zone, aucun balayage
   `ti=13`) se bloque a l'identique. La cause probable est le discriminant de mode : un film de
   Bastion porte des bursts de capture et des evenements nommes que la table du drapeau lit comme
   des prises. C'est du code de l'item 4 (perimetre en LECTURE SEULE ici), et cela bloque toute
   cuisson d'artefact de Bastion en production.
2. **`instance_id` vaut 0 sur toutes les entrees de `map_objectives.json`.** Le champ existe,
   il ne designe rien. Toute jointure future par cet identifiant echouera en silence.
3. **`game_variant_name` porte le mode AVANT le contexte** (`Strongholds:Arena`, `KOTH:Arena`), a
   l'inverse des `pair_name` du registre (`Arena:Slayer`). `NormalizeModeLabel`, ecrit pour les
   seconds, garde « Arena » sur les premiers et PERD le mode. Tout appelant qui normalise un
   `game_variant_name` avant de chercher un jeton de mode se tait sans le dire.
4. **La recopie de grammaire de la phase 2a est SUPERSEDEE** (`zone_state_ti13_scan_test.go`) :
   son balayage vit desormais en production. Elle reste en place parce que les mesures de la
   phase 2a en dependent ; son retrait est un arbitrage de superviseur, avec la cloture du lot.
5. **La progression pourrait etre resolue dans le temps.** Les intervalles publies sont ceux du
   PROPRIETAIRE ; la jauge n'y entre que par son sommet. Couper aussi aux sommets de rampe
   donnerait une jauge vivante pour ~60 intervalles de plus par zone.

## 10. Revue adversariale, ronde 1 (2026-08-18) — neuf corrections, une fusion, un numero

Gates par point : `LOTCBIS_p2b_R1_gates.log`. Un commit par point.

| # | ce qui etait faux | ce qui a change | verrou |
|---|---|---|---|
| R1-1 | le repli « colline » s ouvrait sur tout mode sans capture | il ne s ouvre que sur un mode a colline | `TestZoneStates*` |
| R1-2 | le web retranchait l origine du film une seconde fois | plus de soustraction cote client | `objectivesLayer.test.ts`, grep 0 residu |
| R1-3 | le canal proprietaire s elisait par voisinage et pouvait tenir deux zones | election sur un SEUIL d accord roster, une zone par canal ; `ownerUnpaired` publie | `TestZoneStates*`, contrat regenere |
| R1-4 | les rampes que la grappe ne localise pas etaient perdues | elles se comptent (`unpaired` en methode positions) | `TestZoneStates*` |
| R1-5 | deux collines pouvaient etre actives a la fois | une seule, par construction de la fusion des periodes | contre-epreuve : l ancienne fusion ECHOUE |
| R1-6 | `useZoneStates` rendait un objet neuf a chaque rendu (toute la scene recuite au survol) | retour sous `useMemo` | `useZoneStates.test.ts` (stabilite de reference) |
| R1-7 | rien ne verifiait que `zoneRef` (index cuit) joint la liste servie | `zoneCatalogMatches` + `joinable` ; le CALQUE refuse de peindre (`drawZoneStates` prend un bloc, 5 parametres) | `objectivesLayer.test.ts` « jointure REFUSEE : ne peint rien » ; contre-epreuve : garde retiree = 1 failed |
| R1-8 | le scanner ti=13 emportait `Names`, `dominantName` et un second hook global que rien ne lisait | retires ; i0 reste marche, pas recolte | filmdec vert, grep 0 consommateur |
| R1-9 | le DDL du test des faits n avait pas `map_id` (lu depuis `b0fb3e10f`) : 4 cas rouges en CI | DDL + INSERT portent `map_id` ; blanchiment et NULL verifies | contre-epreuve sur le DDL de HEAD : 4/4 « Binder Error » |

**Fusion de `feat/v75` (`5e40de47f`) et RENUMEROTATION.** L item 4 (drapeau objet) a publie ses
corrections de `flagCarries` sous le **schema 15** et a ete fusionne le premier ; ce lot devient
donc le **schema 16**. Conflits resolus dans `document.go` (chronique v15 = drapeau, v16 = zones,
`SchemaVersion = 16`) et `structure_test.go` (les deux justifications, dans l ordre, `attendu 16`) ;
`document_zones.go` porte la raison du 16 ; `wantReplayDocumentFields` reste **36** (leur compte
35 : `flagObjects` n est pas publie) ; `openapi.yaml` et `generated.ts` regeneres (identiques a la
fusion automatique) ; golden `assembly_000d5950.golden` regenere (**une ligne** : `schema 15` ->
`schema 16`) ; tous les commentaires « schema 15 » de zoneStates (web + `coverage.go`) passes a 16.
`ReplayCanvas.tsx` : le cliquet est descendu a **812** cote feat/v75 (extraction `useReplayTiming`),
la fusion donnait 813 — un commentaire condense, **812** tenu.

## 11. Temoins re-cuits PAR LE CLI, schema 16 (2026-08-18, apres la fusion et la ronde 1)

Le CLI `cmd/replay-build --facts` est DEBLOQUE par le fix de l item 4 (`deathProgressions`
plafonne) : la decouverte n° 1 du §9 est close. Trois cuissons, un processus chacune, sous
watchdog (plafond 3 Go de RAM, 10 min) — aucune n a approche le plafond. Journaux :
`lotCbis/cuisson_cli_<short8>.log` ; artefacts ecrits dans le cache du worktree
(`data/cache/replays/halo_infinite/<short8>.json`, `LEVELUP_REPO_ROOT` = worktree, film lu dans
le main tree). Le controle « captures -> intervalle du capteur » ne se lit pas sur l artefact seul
(l attribution geometrique n y est pas publiee) : il vient de l instrument du journal, rejoue sur
le MEME code (`temoin_<short8>.log`, mis a jour) ; la couverture KOTH, elle, est relue SUR
L ARTEFACT CLI (DuckDB, `read_json`).

| film | mode | CLI : duree / pic RAM | artefact (o) | schema | `coverage.zones` (artefact) | captures -> intervalle du capteur (instrument) | KOTH (artefact) |
|---|---|---|---|---|---|---|---|
| `7344d24f` | Strongholds | 239 s / 187 Mo | 2 202 930 | 16 | catalogue 3, apparies 3, non apparies 2, captures 71, attribuees 59, proprietaire 46/46, 39 intervalles | **57/59 = 96,6 %** | — |
| `696a9d7c` | Strongholds | 230 s / 145 Mo | 2 071 392 | 16 | catalogue 3, apparies 3, non apparies 2, captures 77, attribuees 66, proprietaire 51/51, 37 intervalles, 1 valeur inconnue | **64/66 = 97,0 %** | — |
| `01e1f945` | KOTH | 208 s / 97 Mo | 1 816 953 | 16 | catalogue 8, apparies 6, non apparies 8 (R1-4 : les rampes non localisees se comptent), 20 periodes | sans objet | **20 periodes, 4 975 / 5 343 frames = 93,1 %**, 6 zones actives, **0 chevauchement** (R1-5 verifie sur la forme publiee) |

Les chiffres de controle sont ceux du §3, a l identique, apres les corrections R1-1..R1-9 :
aucune des neuf n a deplace une capture ni une periode sur ce corpus — c est attendu (seuil
d accord a 2 et unicite du canal n ont pas change les elections, la fusion des gardes n avait pas
de recouvrement ici) et c est verifie. La taille du calque sur l artefact complet reste dans les
+0,15 a +0,22 % mesures.

## 12. Correctif CI apres fusion : `PlayerIndex` -> `FilmIndex` (2026-08-18)

**Le garde-rail.** `internal/archlint/no_player_index_identity_test.go` (`TestNoPlayerIndexInFilmScope`)
interdit tout identifiant `PlayerIndex` hors commentaire dans `analysis/filmdec` et `analysis/replay` :
un index de film est un ORDRE, pas une identite — le champ s appelle `FilmIndex` (valable seulement a
l interieur d un film), et une jointure entre joueurs passe par le XUID (precedent : `FireEvent.FilmIndex`,
`fire_events.go`). Le job CI « Go Coverage + Baseline » de `feat/v75` (run 32173015919) l a vu rougir
apres la fusion `bbbc0f92d` : le scanner ti=13 de cette phase avait introduit `ManagedPropertyRead.PlayerIndex`.

**Renomme, sans changement de comportement** (branche `wt/fix-filmindex-zones`) :
`ManagedPropertyRead.PlayerIndex` -> `FilmIndex` (`zone_state_scan.go` : champ + commentaire de statut, et
son ecriture dans le hook ; litteral de `replay/zone_states_test.go`) ; le convertisseur exporte
`ManagedPropertyPlayerIndex` -> `ManagedPropertyFilmIndex` (`components_managed_property.go`, ses trois
appelants `ti13_etat_report_test.go` / `ti13_vecteurs_test.go` / `zone_state_scan.go`, et les deux
commentaires qui le nomment). Les `PlayerIndex` de `weaponv3`, `skill_v2`, `persist`, `cmd/diag_*` sont
HORS perimetre et hors regle (le garde-rail ne couvre que le film et le rejeu).

**Gates** (`LOTCBIS_fix_filmindex_gates.log`) : `gofmt -l` vide, `go vet` filmdec+replay, `go test
./internal/archlint/`, `go test` filmdec+replay (no-CGO), `go build ./...` (CGO), `go test ./internal/replaybuild/`
(CGO) — **6/6 EXIT 0**.

**Lecon.** Les gates locaux du lot (§ gates de la phase 2b, deux rondes de revue comprises) ne lancaient
pas `./internal/archlint/` : la CI l a attrape apres la fusion. Ce paquet (~10 s) entre dans la liste des
gates de TOUT lot qui touche `filmdec` ou `replay`, au meme titre que `gofmt` et `vet`.
