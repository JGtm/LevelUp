# Inventaire des calques du rejeu sous « Seul » et « Masquer » (lot 4, item 4.0)

Source : `apps/web/src/features/match-replay/layers/replayCompose.ts` (`LAYER_ORDER`, 31 calques),
relu sur la tete de `release/v8.2.0` le 2026-11-15.

Regle (user, 2026-11-15, D-4) : « Seul » n'affiche, sur tous les calques, que ce qui appartient au
joueur de la fiche ; « Masquer » retire ce qui lui appartient. Ce qui n'appartient a aucun joueur
reste affiche. Un effet de mort appartient au tueur ET a la victime. Un element dont l'attribution
ne se lit pas dans le document reste affiche (rien n'est devine).

Colonnes : `Avant` = le calque suivait deja `shownAtSlot` / `shownProjectile` (oui / non / sans objet).
`Decision` = case du plan qui le traite, ou `reste` (decor ou sans attribution lisible, compte plus bas).

| Calque | Avant | Element attribuable | Champ du document qui le prouve | Decision |
|---|---|---|---|---|
| `fond-carte` | sans objet | aucun (image de la carte) | aucun | reste |
| `sol-forge` | sans objet | aucun (props de la carte) | aucun | reste |
| `plan-murs` | sans objet | aucun (murs de la carte) | aucun | reste |
| `chaleur` | non | la presence (positions) et les lieux de mort : une chaleur par vie / par mort | `tracks[].slot/startFrame/endFrame` ; lieux de mort des effets de mort | 4.3 bis (chaleur) |
| `zones-nommees` | sans objet | aucun (vocabulaire du terrain) | aucun | reste |
| `objectifs-cuits` | sans objet | aucun (socles, zones, collines : decor du mode) | aucun | reste |
| `projectiles` | oui | grenade / tir lie | `grenades[].slot`, `projectiles[].shot` -> `shots[].slot` | deja fait |
| `socles-armes` | sans objet | aucun (un socle est un lieu) ; `padPickups[].xuid` ne dit que l'etat plein/vide | aucun | reste |
| `armes-au-sol` | non | arme lachee : lacheur, ramasseur | `groundWeapons[].dropper`, `.picker` (slots), `.t0`, `.t1` | 4.3 bis (armes au sol) |
| `poses-equipement` | non | pose : poseur ; marque de revelation du capteur : poseur et vie revelee | `equipmentPlacements[].owner` (slot, -1 = aucun), `.t0` | 4.3 bis (poses) |
| `vehicules` | non | vehicule occupe : ses occupants ; explosion de destruction : occupant et destructeur | `vehicles[].rides[].slot/xuid/t0/t1` ; `vehicles[].destruction.by.xuid`, `.occupant` | 4.3 bis (vehicules) |
| `vies-apparitions` | oui | vie | `LivesActor.slot/slotFrame` | deja fait |
| `vies-morts` | oui | vie | idem | deja fait |
| `vies-tueurs` | oui | vie du tueur | idem | deja fait |
| `trajectoires` | pion, nom, cone, trainee : oui ; CROIX DE MORT : non | la croix est a la vie qu'elle termine (sa victime) | `tracks[].slot`, derniere image de la vie | 4.3 bis (croix de mort) |
| `failles` | non | faille / aller-retour du translocateur : son porteur | `translocations[].slot`, `.t` ; stations derivees (`slot`, `t0`) | 4.3 bis (failles) |
| `marques-de-tir` | oui | tireur | `shots[].slot` | deja fait |
| `gestes-capacite` | sillage du propulseur : oui ; CABLE DU GRAPPIN et POUSSEE DU REPULSEUR : non | grappin : son porteur ; poussee : la personne poussee ET son auteur | `grappleLines[].slot`, `.t0` ; `knockbacks[].slot`, `.by`, `.t` | 4.3 bis (grappin, repulseur) |
| `tirs` | oui | tireur | `shots[].slot` | deja fait |
| `fin-de-vol` | oui | lanceur | `projectileOwners` | deja fait |
| `etat-zones` | sans objet | aucun : proprietaire = une equipe, jamais un joueur | `zoneStates[].spans[]` (equipe, jauge) | reste |
| `drapeaux` | non | drapeau PORTE : son porteur ; onde de capture : son auteur. Drapeau a la base, tombe, zone de retour : personne | `flagCarries[].spans[].state` + `.xuid` (porteur seulement pour `carried` / `carried_open`) ; `objectives[].xuid` (stat `flag_captures`) | 4.3 bis (drapeaux) |
| `objets-objectif` | sans objet | crane LIBRE : personne (le porte est le calque `crane-porte`) | `objectiveObjects[]` ne porte aucun champ de propriete (`en`, `fr`, `family`, `pts`) | reste |
| `couronne-vip` | non | le VIP | `vipCrown[].xuid` | 4.3 bis (couronne) |
| `crane-porte` | non | le porteur | `skullCarries[].xuid` | 4.3 bis (crane) |
| `bombe-portee` | non | le porteur ; bombe LIBRE (au sol) : personne | `bombCarries[].xuid` | 4.3 bis (bombe portee) |
| `deflagration` | non | l'auteur credite de l'explosion | `objectives[].xuid` (stat `bomb_detonations`) | 4.3 bis (deflagration) |
| `pulses-objectif` | non | l'auteur de l'action (capture, retour, prise de zone) | `objectives[].xuid` | 4.3 bis (pulses) |
| `morts` | non | tueur ET victime | feed : `killer xuid`, `victimXuid` ; vies de `tracks[]` | 4.1 a 4.3 |
| `mise-en-valeur` | oui | tueur et victime mis en valeur | `KillFocusResolvers.shownAtSlot` | deja fait |
| `etiquettes-mise-en-valeur` | oui | idem | idem | deja fait |

## Comptes

- 31 calques : 9 suivaient deja le filtre sur tout leur contenu (projectiles, vies x3, marques de tir, tirs,
  fin de vol, mise en valeur x2) ; 14 portent un element attribuable que le filtre ne lisait
  pas (morts, chaleur, armes au sol, poses, vehicules, failles, drapeaux, couronne, crane, bombe portee,
  deflagration, pulses, la croix de mort du calque `trajectoires`, dont le pion suivait deja, et le cable du
  grappin et la poussee du repulseur du calque `gestes-capacite`, dont le sillage du propulseur suivait deja) ; 8 sans objet
  ou sans attribution lisible (fond, sol, plan des murs, zones nommees, objectifs cuits, socles d'armes, etat
  des zones, objets d'objectif).
- Rectification a la relecture du code : la croix de mort de `trajectoires` (`drawDeathMark`) ne suivait PAS
  `shownAtSlot` (un test le figeait : « une mort est un evenement du match »). Elle suit desormais sa victime.
  Le tueur n'est pas dans la piste : un joueur isole ne garde pas la croix de sa victime ; l'effet tueur ->
  victime est le calque `morts`.
- Meme rectification pour `gestes-capacite` : seul le sillage du propulseur suivait `shownAtSlot`. Le cable du
  grappin est a son porteur ; la poussee du repulseur est a la personne poussee ET a son auteur (`by`, lu a
  l'image de la poussee ; sans auteur publie, le pousse decide).
- Elements sans attribution lisible, laisses affiches (jamais devines) :
  - `objets-objectif` : le crane libre ne porte aucun champ de propriete ; libre = a personne.
  - `etat-zones` et `objectifs-cuits` : la propriete d'une zone est une EQUIPE (`zoneStates[].spans[].team`),
    pas un joueur.
  - `socles-armes` : `padPickups[].xuid` dit qui a pris un socle, pas a qui il est ; un socle est un lieu.
  - drapeau a la base ou tombe, zone de retour, bombe libre, arme au sol sans lacheur ni ramasseur
    lisibles (`dropper` et `picker` < 0), pose sans poseur mesure (`owner` < 0), vehicule vide : personne.
  - emplacements de naissance des vehicules (`vehicleCycles`) : un lieu.
- Elements dont une PARTIE seulement se lit : un effet de mort dont le tueur ou la victime n'a pas de vie
  retrouvee ne prouve rien pour cette partie ; l'autre partie decide ; sans aucune partie lue, l'effet reste.
- Aucun champ manquant au document : toutes les cases ci-dessus se traitent cote web, sans changement Go
  ni regeneration.
