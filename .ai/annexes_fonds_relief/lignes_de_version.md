# Lignes de notes de version et de journal des modifications du fond Relief (à poser par le pilote à l'intégration, N20)

Proposées par l'agent de l'item E.2 le 2026-11-18, relues par le pilote. À poser dans `docs/RELEASE_NOTES.md`,
`docs/FR/RELEASE_NOTES.md` (section v8.3) et `docs/CHANGELOG.md`, `docs/FR/CHANGELOG.md` (Added / Ajouté) au moment
de verser `feat/fonds-relief` dans `release/v8.3.0`, après la cuisson du parc. Ajouter aussi une ligne de crédit à
`docs/ACKNOWLEDGMENTS.md` (OpenSpartan Cartographer, Den, licence MIT).

## Notes de version
- EN : **A Relief background for the maps** — a third background, beside "Plan" and "Rendu", shows the terrain from above: lighter when higher, shaded, cut to the playable area, with optional relief lines (height breaks, contour lines). Inspired by OpenSpartan Cartographer, by Den (MIT licence).
- EN : **One floor at a time** — in the replay, Tactics and Terrain occupation, pick a floor (or "All", or a height) and the others fade; in the replay, "Follow the open player" moves the cut with them.
- EN : **Markers readable on every background** — each marker gets a dark band edged with a light rim, so it stands out on Plan, Rendu and Relief, in the four colour palettes.
- EN : **Live Fire, complete** — its background is redone from the game's navigation mesh: it gains its wall plan, its Relief and the walkable areas that were missing.
- FR : **Un fond Relief pour les cartes** — un troisième fond, à côté de « Plan » et de « Rendu », montre le terrain vu de dessus : plus clair quand il est plus haut, ombré, découpé à la zone jouable, avec des traits du relief en option (ruptures de hauteur, courbes de niveau). D'après OpenSpartan Cartographer, de Den (licence MIT).
- FR : **Un étage à la fois** — dans le rejeu, la Tactique et l'Occupation du terrain, on choisit un étage (ou « Tout », ou une hauteur) et les autres s'estompent ; dans le rejeu, « Suivre le joueur ouvert » déplace la coupe avec lui.
- FR : **Des marqueurs lisibles sur tous les fonds** — chaque marqueur reçoit une bande sombre bordée d'un filet clair, pour se détacher du Plan, du Rendu et du Relief, dans les quatre palettes de couleurs.
- FR : **Live Fire, complète** — son fond est refait d'après le maillage de navigation du jeu : elle gagne son plan des murs, son Relief et les zones praticables qui lui manquaient.

## Journal des modifications
- EN : **Maps: Relief background and relief lines** — `{key}.relief.webp` and `{key}.relief.json.gz` next to each background, `reliefImage` and `reliefData` in the sidecar, `?variante=relief` and `background.relief.json` on the background routes, `mapfond-build --relief-seul`; brick `halo-infinite-map-images` v1.4.0, after OpenSpartan Cartographer (Den, MIT).
- EN : **Replay, Tactics, Terrain occupation: floor cut** — "All", one pill per floor, a height slider; in the replay, "Follow the open player".
- EN : **Replay: two-tone marker outline** — tokens `marker-outline-band` and `marker-outline-rim`, contrast of at least 3:1 guarded under the four visions (4.4:1 measured).
- EN : **Maps: Live Fire rebuilt** — mesh read from the shared module `any/globals/common-rtx-new.module`, wall plan and relief published.
- FR : **Cartes : fond Relief et traits du relief**, **Rejeu, Tactique, Occupation du terrain : coupe par étage**, **Rejeu : cerne deux tons des marqueurs**, **Cartes : Live Fire refaite** (mêmes contenus, mêmes noms de fichiers et de jetons).

## À reprendre dans l'ADR 0049 après la cuisson du parc
« Current state » (nombre de cartes avec relief, poids ajouté, cartes sans relief et pourquoi).

## Constats hors périmètre (E.2)
`docs/FOUNDATIONS_GUIDE.md` §3.6 (EN et FR) dit encore « un vecteur, jamais une seconde image » pour le plan des murs et « 1.5 m » de marge, amendés par l'ADR 0048 les 16 et 17/11.
