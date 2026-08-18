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
| `internal/analysis/replay/document_zones.go` (NEUF) | 165 | la FORME (`ZoneState`, `ZoneSpan`, `ZonesCoverage`) et la chronique du schema 15 |
| `internal/analysis/replay/zone_states.go` (NEUF) | 338 | la REGLE : series par tag, fenetres, catalogue renumerote, echelle de la jauge |
| `internal/analysis/replay/zone_states_owner.go` (NEUF) | 362 | le volet PROPRIETAIRE : appariement des slots, intervalles, controle |
| `internal/analysis/replay/zone_states_hill.go` (NEUF) | 166 | le volet COLLINE : periodes de garde par la grappe des positions |
| `internal/analysis/replay/build_zones.go` (NEUF) | 97 | le CABLAGE (`decodeFilmZoneReads`, `attachZoneStates`) et son journal |
| `internal/replaybuild/zones.go` (NEUF) | 165 | le CATALOGUE du match, dans l'ordre ou le service le sert |
| `internal/analysis/replay/{build,document,coverage}.go` | +12/+11/+10 | `Options.Zone`, `ReplayDocument.ZoneStates`, `Coverage.Zones`, schema 14 -> **15** |
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

- `SchemaVersion` **14 -> 15**, chronique dans `document_zones.go` et raison ecrite dans
  `structure_test.go` (le garde exige une justification, pas un numero).
- `wantReplayDocumentFields` **35 -> 36** + 3 schemas (`ZoneState`, `ZoneSpan`, `ZonesCoverage`).
- `api/openapi.yaml` REGENERE (`make openapi-gen`), `generated.ts` regenere.
- Garde web `_ListeExhaustive` : elle a **ROUGI puis PASSE** — 4 erreurs `tsc` exigeant
  `zoneStates` et `zoneStates[].spans` dans les deux listes, ce qui est le comportement attendu.
- `coverage.zones.roles` est une CHAINE et non un tableau : ce temoin de jointure que rien ne
  parcourt aurait sinon fait entrer un tableau nullable de plus dans la frontiere du client.
- Golden `assembly_000d5950.golden` : **une ligne** change (`schema 14` -> `schema 15`). Le film
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
- [x] **CB.2b.2** — contrat : schema 15, 36 champs, OpenAPI et types regeneres, garde web rougie
  puis verte, golden justifie, **3 temoins re-cuits** avec faits et chiffres publies.
- [x] **CB.2b.3** — rendu : teinte par proprietaire, colline active, arc de progression, helper
  pur `zoneStateAt` teste ; pulses conserves (constat ecrit) ; gates web verts.

## 9. Decouvertes (hors perimetre — notees, NON traitees)

1. **LE CALQUE DU DRAPEAU PART EN VRILLE SUR UN FILM DE BASTION.** `cmd/replay-build --facts` sur
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
