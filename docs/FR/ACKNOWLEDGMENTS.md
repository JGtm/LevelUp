# Remerciements

> Version anglaise : [../ACKNOWLEDGMENTS.md](../ACKNOWLEDGMENTS.md)

LevelUp parle aux services Halo via **son propre client Go**, écrit pour ce projet
(`apps/go-api/`). Il ne dépend d'aucune bibliothèque d'API Halo tierce — aucune n'apparaît dans
`go.mod`, et la synchronisation, le décodage du film et l'accès aux données sont tous maison.

Ce client existe parce que d'autres ont défriché le terrain avant. Les travaux ci-dessous sont
des références dont nous avons appris, pas du code que nous embarquons :

- **Andy Curtis** ([acurtis166](https://github.com/acurtis166)) pour [SPNKr](https://github.com/acurtis166/SPNKr) —
  la référence qui a cartographié les points d'entrée de Halo Infinite et le format du film
  Theater. Notre lecteur de temps forts a commencé comme un portage Go de
  `spnkr/film/highlight_events.py`, avant que le décodeur ne se dote de sa propre grammaire.
- **Den Delimarsky** ([dend](https://github.com/dend)) pour [Grunt](https://github.com/dend/grunt)
  et [OpenSpartan](https://github.com/OpenSpartan) — la documentation communautaire des services
  Halo, et la voie d'import des données OpenSpartan.
- **Gravemind2401** ([Gravemind2401](https://github.com/Gravemind2401)) pour [Reclaimer](https://github.com/Gravemind2401/Reclaimer) —
  sa lecture des formats de fichiers de cartes Halo est la référence derrière notre décodeur de
  géométrie (`apps/go-api/internal/himap/` : sections SBSP, filtres de maillage, groupes SDDT).

Si vous utilisez ce projet, pensez à soutenir aussi les projets amont en les « star » sur GitHub.
