# Remerciements

> Version anglaise : [../ACKNOWLEDGMENTS.md](../ACKNOWLEDGMENTS.md)

LevelUp parle aux services Halo via **son propre client Go**, écrit pour ce projet
(`apps/go-api/`). Il ne dépend d'aucune bibliothèque d'API Halo tierce — aucune n'apparaît dans
`go.mod`, et la synchronisation, le décodage du film et l'accès aux données sont tous maison.

Ce client existe parce que d'autres ont défriché le terrain avant. Les travaux ci-dessous sont
des références dont nous avons appris, pas du code que nous embarquons, sauf les fichiers de données nommés plus bas :

- **Andy Curtis** ([acurtis166](https://github.com/acurtis166)) pour [SPNKr](https://github.com/acurtis166/SPNKr) —
  la référence qui a cartographié les points d'entrée de Halo Infinite et le format du film
  Theater. Notre lecteur de temps forts a commencé comme un portage Go de
  `spnkr/film/highlight_events.py`, avant que le décodeur ne se dote de sa propre grammaire.
- **Den Delimarsky** ([dend](https://github.com/dend)) pour [Grunt](https://github.com/dend/grunt)
  et [OpenSpartan](https://github.com/OpenSpartan) — la documentation communautaire des services
  Halo, et la voie d'import des données OpenSpartan.
- **Den Delimarsky** ([dend](https://github.com/dend)) pour [OpenSpartan Cartographer](https://github.com/OpenSpartan/cartographer) et [films-ui](https://github.com/OpenSpartan/films-ui) —
  les principes du fond Relief (terrain teinté selon la hauteur, traits vectoriels, coupe par étage) ; licence MIT, aucun code repris.
- **Den Delimarsky** ([dend](https://github.com/dend)) pour libfilmshell (OpenSpartan, licence MIT) — les tables des
  réponses de score, des médailles et des effets de dégât, et les largeurs de charge de certains messages du film, lues
  par le décodeur de film (`halo-infinite-film-decoder`, v1.6.1) ; les en-têtes de crédit et les textes de licence se
  trouvent à côté des tables.
- **llampwall** ([llampwall](https://github.com/llampwall)) pour le lecteur de rejeu 3D bâti sur le décodeur de film
  (`llampwall/installation-08`) et les lectures qu'il a apportées : gestes, commandes d'accroupi et de saut, tireur par
  place de contrôle, visées sans position, arme en main, munitions portées, positions à la cadence pleine du film,
  projectiles collés, socles d'équipement. Elles ont été réécrites sous les règles du décodeur (v1.6.1) ; aucune ligne du
  dépôt de départ n'est copiée.
- **nuzzles** (l'analyseur Rust `halo_api`) — la méthode de lecture des morceaux d'un film sur plusieurs fils, derrière
  l'option de fils du décodeur (v1.6.1, génération des rejeux deux à trois fois plus rapide) ; aucun code repris.
- **Gravemind2401** ([Gravemind2401](https://github.com/Gravemind2401)) pour [Reclaimer](https://github.com/Gravemind2401/Reclaimer) —
  sa lecture des formats de fichiers de cartes Halo est la référence derrière notre décodeur de
  géométrie (`halo-infinite-map-images/himap/` : sections SBSP, filtres de maillage, groupes SDDT).
- **Gamergotten** ([Gamergotten](https://github.com/Gamergotten)) et les contributeurs de
  [Infinite Runtime Tag Viewer](https://github.com/Gamergotten/Infinite-runtime-tagviewer), avec
  **Lord Zedd** et **Exhibit** qui ont extrait les définitions de tags qu'il porte : les fichiers de
  disposition `halo-infinite-map-images/himap/sbsp.xml` et les deux `weap.xml` de `apps/go-api/cmd/` sont des copies
  non modifiées de ses fichiers `Plugins/` (voir `NOTICE`).

Si vous utilisez ce projet, pensez à soutenir aussi les projets amont en les « star » sur GitHub.
