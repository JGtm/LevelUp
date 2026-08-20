# Inventaire complet des `weapon_id` Halo 5 hors-arsenal (non mappés à un rôle)

> Artefact de travail pour classification produit. Genere le 2026-07-17.
> Source : `v_weapon_kills` (H5) + `weapon_labels` (metadata H5) + registre
> `weapon_ids` (metadata H5). Lecture SEULE.

## Methode de lecture

- CLI `duckdb` absent de l'environnement ; aucun serveur MCP `duckdb` disponible
  (seul Notion l'est). Modele mono-process : le serveur dev peut tenir les DB en RW.
- Les deux DB ont donc ete **copiees** vers le scratchpad puis interrogees
  **read-only** (`?access_mode=read_only`) par un petit programme Go **jetable**
  (driver `github.com/duckdb/duckdb-go/v2`, supprime apres coup). Aucune ecriture,
  aucune sonde laissee dans le repo.
- Requete de comptage des frags :
  `SELECT effective_weapon_id, COUNT(*) FROM v_weapon_kills WHERE effective_weapon_id NOT IN (0,1,2) GROUP BY 1 ORDER BY 2 DESC`
  (`effective_weapon_id = COALESCE(reconciled_as, weapon_id)`, cf. vue
  `steps_shared_core.go`). Sentinelles 0/1/2 exclues comme le fait le runner de
  couverture (`registry_weapon_coverage.go`) — leur volume observe = **0**.
- Libelles : `weapon_labels.name_en` (metadata H5). IDs sans ligne = UGC/inconnu.
- Couverture : `weapon_ids WHERE title_slug='halo_5'`.

## Totaux

| Metrique | IDs distincts | Frags |
|---|---:|---:|
| Total `v_weapon_kills` (hors sentinelles 0/1/2) | 66 | 268 327 |
| **Couverts** (mappes a une arme d'arsenal, un role) | 40 | 251 678 (93,8 %) |
| **NON couverts** (hors-arsenal : vehicules, tourelles, buckets, UGC) | 26 | 16 649 (6,2 %) |

Nota couverture : le registre **code** (`weapon_registry.go`,
`weaponRegistryH5Stock`) mappe **40** stock_ids = 35 armes historiques + 5 long-tail
ajoutes au commit `9e0a8217d` (2026-07-17 : Frag/Plasma/Splinter Grenade, Golf Club,
Oddball/Ball). La copie de `metadata.duckdb` interrogee n'a encore que **35** de ces
40 ids seedes (`weapon_ids`) : les 5 long-tail attendent un reseed au boot
(`INSERT OR IGNORE`). Ils sont donc comptes ici comme **couverts** (le code fait foi)
et **exclus** du tableau ci-dessous. Detail de ces 5 (deja mappes cote code) :

| weapon_id | libelle | frags | role code |
|---|---|---:|---|
| 4106030681 | FRAG GRENADE | 10 494 | grenade |
| 2460880172 | PLASMA GRENADE | 5 292 | grenade |
| 3190813201 | SPLINTER GRENADE | 2 079 | grenade |
| 409331533 | Golf Club | 360 | melee |
| 393532233 | Ball (Oddball) | 179 | melee |

## Tableau exhaustif des 26 `weapon_id` NON couverts (tri frags decroissant)

| weapon_id | libelle (weapon_labels) | frags | categorie brute |
|---|---|---:|---|
| 3168248199 | Spartan | 8 812 | bucket d'attribution |
| 2457457776 | *(aucun label)* | 2 332 | UGC / inconnu |
| 3010146366 | Ghost | 1 204 | vehicule |
| 1063919886 | Mongoose | 928 | vehicule |
| 4028516791 | Warthog | 766 | vehicule |
| 47178948 | Environmental Explosives | 589 | bucket d'attribution (environnement) |
| 419783896 | Banshee | 371 | vehicule |
| 2988661926 | Chaingun Turret | 360 | tourelle |
| 3227919741 | Mantis | 247 | vehicule |
| 1730553442 | Scorpion | 237 | vehicule |
| 1749823285 | Splinter Turret | 237 | tourelle |
| 2907783784 | ROCKET POD TURRET | 211 | tourelle |
| 1206711506 | Wraith | 107 | vehicule |
| 3207900961 | Wasp | 100 | vehicule |
| 4233134183 | Gauss Turret | 68 | tourelle |
| 698769165 | SHADE PLASMA TURRET | 25 | tourelle |
| 390856427 | *(aucun label)* | 11 | UGC / inconnu |
| 244872079 | SCORPION ANTI INFANTRY TURRET | 8 | tourelle |
| 2023669721 | Plasma Turret | 7 | tourelle |
| 3541732101 | *(aucun label)* | 6 | UGC / inconnu |
| 1351500565 | Hunter Arm Turret | 6 | tourelle |
| 3394982816 | Phaeton | 5 | vehicule |
| 642449794 | *(aucun label)* | 4 | UGC / inconnu |
| 2497647768 | *(aucun label)* | 4 | UGC / inconnu |
| 2631958027 | *(aucun label)* | 2 | UGC / inconnu |
| 2957796559 | *(aucun label)* | 2 | UGC / inconnu |

## Repartition des 26 non couverts par categorie

| Categorie | IDs | Frags |
|---|---:|---:|
| Vehicule (Ghost, Mongoose, Warthog, Banshee, Mantis, Scorpion, Wraith, Wasp, Phaeton) | 9 | 3 965 |
| Tourelle (Chaingun, Splinter, Rocket Pod, Gauss, Shade Plasma, Scorpion AI, Plasma, Hunter Arm) | 8 | 922 |
| Bucket d'attribution (Spartan, Environmental Explosives) | 2 | 9 401 |
| UGC / inconnu (absents de `weapon_labels`) | 7 | 2 361 |
| **Total non couvert** | **26** | **16 649** |

## Ambiguites / notes de classification

- **`3168248199` « Spartan » (8 812 frags)** — de loin le plus gros bucket
  hors-arsenal, a lui seul ~53 % des frags non couverts. Nature ambigue : bucket
  generique d'attribution renvoye par l'API H5 quand le frag n'est pas rattache a un
  stock_id d'arme — typiquement melee/beatdown, assassinats, splatters non attribues
  a un vehicule precis, et divers. Ce n'est PAS une arme tenue. A classifier au
  produit : soit « autre / non attribue », soit ventile si une source secondaire
  (medals, killer/victim melee) permet de le desambiguiser.
- **`47178948` « Environmental Explosives » (589 frags)** — hasard de map
  (explosifs/tonneaux, chutes), pas une arme. Bucket d'attribution environnemental.
- **UGC / inconnus (7 ids, 2 361 frags)** — absents de `weapon_labels`. Le plus gros
  (`2457457776`, 2 332 frags) est probablement une variante REQ/UGC ou un contenu
  Forge non catalogue par le fetcher `cmd/h5-metadata-fetch` ; les 6 autres sont
  marginaux (<= 11 frags). Non identifiables sans un nouveau fetch du catalogue H5.
- **Vehicules & tourelles** — non ambigus (libelles clairs). Intentionnellement non
  mappes a un role de combat : leur donner un role fausserait le donut « Frags par
  type d'arme » et l'insight coach `blind_spot_power` (cf. commentaire
  `weapon_registry.go` l.366+). Une categorie produit `vehicule` / `tourelle`
  dediee serait le bon receptacle.
- Coherence : cet inventaire reproduit exactement l'enumeration de tracabilite du
  commentaire `weaponRegistryH5Stock` (`weapon_registry.go` l.366-387) — meme 26 ids,
  memes ordres de grandeur. Aucun ID surprise hors de cette liste.
