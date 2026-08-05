# Investigation — Noms de zones / callouts de map (récupérables, délimités, plaçables 2D ?)

> Petite investigation (2026-06-26) déclenchée par la question : peut-on sortir les NOMS des zones
> d'une map (callouts comms type "Bunker/Rampe", zones d'objectif Strongholds/CTF/KOTH), les
> DÉLIMITER, et les PLACER sur la carte 2D qu'on extrait déjà des modules ?
> Complément de `.ai/V7.5/cartes/HANDOFF_MAP_GEOMETRY_FROM_MODULES.md` (master géométrie). PAS DE CODE écrit ici,
> juste le repérage de la source + faisabilité.

## TL;DR

- Les noms de zones ne sont **PAS dans le film** (le replay ne réplique que le dynamique ; l'identité
  de zone vit dans la state-machine hors-film — déjà confirmé par le travail objective-events :
  `objective_id` toujours NULL).
- Ils ne sont **PAS non plus dans le module `ds/`** (celui qu'on utilise pour la géométrie : réduit à
  géométrie + rendu + son).
- Ils sont dans la **variante `any/`** du module de map (tag scénario / placements gameplay).
- **Récupérables + plaçables sur la 2D : oui**, avec l'infra existante (himodule + ooz + parser de
  struct-tree de tag), même espace monde que la géométrie. Mini-chantier neuf mais **faible risque**.

## Investigation empirique (outil `cmd/tmp_moduleread`, CGO-free, container-list)

Recensement des groupes de tags par variante, map **catalyst** (`deploy/<variante>/levels/multi/catalyst/catalyst-rtx-new.module`) :

| Variante | Groupes de tags | Verdict zones/callouts |
|---|---|---|
| **`ds/`** (géométrie) | 7 : bitm, rtgo, mode, sbnk, **sbsp**, levl (+18 resources) | NON — géométrie/rendu/son uniquement |
| **`any/`** | **41** : `mat` (matériaux), `scgt`, un tag **scénario/script (`hscn`/`hsc*`)**, **`pfnd` (pathfinding/nav)**, **`scen` (scenery)**, **`sddt` (structure-design/placement)**, effets, sons… | OUI — c'est là que vivent les placements gameplay nommés |
| **`pc/`** | (parse container échoue — layout d'entrée différent, cf handoff : lock PoC ~23%) | à ignorer, `any/` suffit |

Conclusion : les callouts/zones sont dans le **scénario du module `any/`**, pas le film, pas la `ds/`.

## Réponses aux 3 questions

### 1. Récupérables ?  → Oui, très probablement (offline, universel)
- Le tag scénario (`any/`) porte les "named locations" / placements d'objets.
- Réserves : (a) les **noms** sont des **string-IDs** (hash) → il faut la table de strings pour le
  texte lisible ; (b) les **zones d'objectif** (Strongholds/KOTH/CTF) peuvent être dans le scénario
  OU dans un module **game-mode** séparé (à confirmer) ; les **callouts comms** sont intrinsèques à
  la map → dans le scénario.

### 2. Délimités ?  → Oui, mais deux natures différentes
- **Callouts comms** = typiquement **point + rayon d'influence** (nearest-callout) → rendu en
  marqueurs labellisés ou régions Voronoï, PAS des polygones nets.
- **Zones d'objectif** (Strongholds…) = **volumes** (boîte/cylindre : position + extents) → contours
  exploitables directement.

### 3. Plaçables sur la carte 2D ?  → Oui, directement
- Placements en coordonnées monde = **même espace que la géométrie déjà extraite** → la transformation
  monde→2D existante s'applique telle quelle (cf overlay positions joueurs déjà à 84%,
  `cmd/tmp_mapoverlay`).

## Faisabilité / effort

Pas de mur conceptuel. Réutilise TOUT le pipeline géométrie (`internal/himodule` Open/Files/Extract +
`internal/ooz` décompression + le field-walker version-aware de struct-tree de tag). Frictions RE
réelles :
1. Layout **version-spécifique** du tag scénario (même type de field-walker que la géométrie ; plugin
   IRTV `scnr`/scenario + `Halo/TagObjects/TagLayouts.cs` pour les tailles-par-type).
2. Résolution **string-ID → texte** (table de strings du build / liste communautaire).
3. Confirmer où sont les **zones d'objectif** (scénario de map vs module de game-mode).

## Étape qui confirmerait à 100% (non faite)

Décompresser le tag scénario du module `any/` (via `himodule.Extract`, CGO/ooz) et lister son bloc
"named locations" → sortir la liste réelle des callouts d'une map + leurs positions. C'est le premier
pas d'implémentation quand on reprendra.

## Pointeurs

- Master géométrie : `.ai/V7.5/cartes/HANDOFF_MAP_GEOMETRY_FROM_MODULES.md` (pipeline module→carte 2D, §9/§10).
- Outils : `cmd/tmp_moduleread` (liste container, CGO-free) ; `internal/himodule` (Open/Files/Extract) ;
  `internal/ooz` (Kraken/Oodle) ; `cmd/tmp_geores` (rendu carte) ; `cmd/tmp_mapoverlay` (overlay positions).
- Modules sur disque : `D:\SteamLibrary\steamapps\common\Halo Infinite\deploy\{any,ds,pc}\levels\multi\<map>\<map>-rtx-new.module`.
