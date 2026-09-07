# `features/match-replay/` — la page de rejeu 2D

Ce dossier portait 370 fichiers à plat. Depuis le 2026-09-06 (lot v2, item D.11) il est rangé
par RESPONSABILITÉ. Ce fichier dit **où va un fichier neuf**, et pourquoi — sans cette règle
écrite, l'arborescence se remélange en trois lots.

## La règle de classement

Un fichier a **une** place, décidée par le PREMIER critère qui s'applique, dans cet ordre :

| # | Dossier | Ce qui y entre | Le critère, en une phrase |
|---|---|---|---|
| 1 | `i18n/` | le dictionnaire FR/EN, son contrat de champs, la lecture d'un libellé du document | il porte du TEXTE affiché, ou décide de la langue |
| 2 | `sound/` | manifeste des sons, tirage de variante, mixage, moteurs de véhicule, contrôles de son | il décide, choisit ou joue un SON |
| 3 | `export/` | dialogue d'export, plan d'images, conteneur vidéo, encodeur, capture PNG, panneaux repeints pour la vidéo | il sert à produire un FICHIER (image ou vidéo) |
| 4 | `settings/` | tiroir de réglages, bascules de calques, préférences persistées, ancrage du panneau | il appartient au TIROIR ou aux préférences |
| 5 | `layers/` | calques, effets, glyphes, marques, encres du canvas, réglages du canvas, et les hooks qui ne câblent QUE leur calque | **une de ses fonctions exportées reçoit un `CanvasRenderingContext2D`** — il PEINT |
| 6 | `ui/` | composants React de la page : toile, fil, fiches, frise, barre de lecture, surimpressions | c'est un composant React (`.tsx`) que la page monte |
| 7 | `hooks/` | lecture, cadrage, zoom, glisser, molette, clavier, frise, durées, cascades de camp | c'est un hook React qui pilote le LECTEUR, pas un calque |
| 8 | `model/` | normalisation, contrat de types, horloge, roster, fenêtres, sièges, ce qu'un calque doit SAVOIR avant de peindre | tout le reste : de la donnée, pas du trait |

Deux dossiers de service complètent la liste :

- `test/` — les doubles partagés (`fakeAudio`, `testDoc`, `recordingContext`, fixtures de pose)
  et `featureFiles.ts`, qui rend à un garde-rail la liste RÉCURSIVE des fichiers du rejeu ;
- la RACINE ne garde que ce que la Match View consomme (`queries.ts`,
  `MatchEquipmentUsageSection`, `MatchPadControlSection`) : leur foyer définitif est l'objet de
  l'item D.13 (constat N4 de l'audit), pas un choix par défaut.

## Les deux pièges que cette règle évite

**« Fx » ne veut pas dire « calque ».** `shotFx.ts`, `killFx.ts`, `grenadeFx.ts`,
`playerCardFx.ts` calculent ce qu'un effet doit montrer et ne touchent jamais la toile : ils
vivent dans `model/`. Le critère est le contexte de dessin, pas le nom du fichier — c'est ce
qui rend le classement vérifiable par un test plutôt que discutable en revue.

**Un test suit son module, un garde-rail suit ce qu'il garde.** `x.test.ts` vit à côté de
`x.ts`. Un garde qui balaie la feature (« ce motif ne s'écrit qu'ici ») prend sa liste à
`test/featureFiles.ts` : `readdirSync(__dirname)` ne voit plus qu'un huitième du rejeu et
resterait VERT en ne gardant plus rien.

## Ce que la taille dit, et ne dit pas

Le seuil de R5 (500 lignes) se mesure en lignes de CODE — règle `max-lines` d'ESLint avec
`skipComments` et `skipBlankLines` (décision utilisateur du 2026-09-05). Un fichier long parce
qu'il s'explique n'est pas un god file ; un fichier qui dépasse s'extrait par RESPONSABILITÉ,
ou porte une exemption datée en tête. Le cliquet de lignes brutes qui pesait sur
`ReplayCanvas.tsx` (≤ 665) a été supprimé le 2026-09-06 : il avait payé 17 extractions, dont 14
alors que le code du fichier était déjà conforme.
