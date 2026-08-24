# PLAN — RE courte : ti=47 i2 « personal-ai-data » (annonces de zone)

Date : 2026-08-24. Origine : REGISTRE_REPORTS lignes « ti=47 i2 personal-ai-data-component
(non porté) : le canal le plus concentré du corpus » et « Sondes F2/F4/F5 » (lot C phase 0,
17-18/08). Backlog Notion « Rejeu 2D — bilan du 18/08/2026 », § Prêt à exploiter. Go
utilisateur : 2026-08-24, avec la direction produit suivante : si les annonces se
concrétisent, la restitution sera un EFFET UI à la capture, JAMAIS du texte à l'écran.

Branche : `wt/ti47-annonces` (worktree dédié, base feat/v75). Exécution sous le contrat du
skill `plan-execution` (ordre strict, gates, statuts, zéro fix hors périmètre).

## Objectif et critère de succès

Porter la LECTURE de `ti=47 i2` sous instruments de recherche gatés, établir sa sémantique
par corrélation aux oracles connus, et rendre un VERDICT : ce canal porte-t-il des annonces
identifiables (capture / perte / contestation de zone, et lesquelles) ? Le lot S'ARRÊTE au
verdict — aucune publication d'artefact, aucun bump de schéma, aucun rendu web.

Succès = verdict tranché (positif avec table de classes datées, ou négatif écrit — les deux
sont des succès), reproductible par les commandes du rapport.

## Faits établis (ne pas re-mesurer)

- ti=47 i1 (« splash » DYNAMIQUE, déjà porté) : scalaire quantifié sur un treillis de pas
  4 584 ; anti-corrélé aux captures de zone (0,01x), suit les événements de drapeau en CTF
  (1,65-3x). Les annonces de zone ne passent PAS par i1.
- ti=47 i2 : NON porté ; 1 214x le plancher de densité, 77-81 % des records ti=47 en modes à
  zones, quasi absent en CTF — le profil attendu d'une « voix de l'IA personnelle ».
- Le hook `SetProbeHook` du lot 0 existe déjà pour i0/i1 (point d'entrée du portage).
- Leçon i27 (registre, 18/08) : tout classement fondé sur `MaskCensus` SEUL est à relire au
  filtre réel/fantôme (le bruit d'ancrage a déjà fait mettre un canal en réserve à tort).
- Oracles disponibles hors film : captures/possessions de zone (`zoneStates`, schéma 15+),
  événements nommés `flag_*`, périodes de colline KOTH (schéma 18), scoreTimeline.
- La lettre A/B/C affichée en jeu n'existe dans AUCUNE donnée décodée à ce jour (règle
  « aucun texte » des calques, testée). Si i2 portait un identifiant de zone nommée, ce
  serait une PREMIÈRE — le noter comme tel, pas le présupposer.

## Hors périmètre (fermé)

- MCP / jeu vivant (recette R7-d dynamique) : ce lot est OFFLINE PUR sur le corpus local.
  Si la sémantique de i2 est indéchiffrable sans le jeu, c'est un verdict (négatif
  conditionnel : « nécessite une session MCP », à consigner).
- Tout changement de schéma d'artefact, de contrat OpenAPI, de rendu web.
- Les autres composants de ti=47 ; ti=13 i1 (RE complète = autre lot, décision utilisateur).

## Phase 0 — Portage de lecture sous garde

- [x] 0.1 Porter la déser de `ti=47 i2` (patron des composants déjà portés, table de déser
      existante ; largeurs depuis le descriptor +0x28, jamais devinées).
      **FAIT 2026-08-24, PAR UNE AUTRE VOIE, ÉCRITE ICI.** Le descriptor `+0x28` est
      INACCESSIBLE : `mcp__ghidra__list_instances` rend `{"instances": []}` (aucune instance
      Ghidra ouverte), et le lot est offline pur. La largeur n'a donc pas été LUE — elle a été
      **MESURÉE sur les octets**, par le protocole du lot C-bis (LOTCBIS_PHASE0 §3.2), étendu
      de « confirmer » à « découvrir » : histogramme de chaînage sur tous les décalages, cible
      restreinte à la bande, distance entre débuts de records consécutifs, longueur de chaîne.
      **`ti=47 i2` = R(45)**, record singleton de 72 bits. Témoin positif dans le même
      archétype (`i1` = R(24), porté depuis le lot 0) retrouvé à 100 % par les quatre mesures
      sur les 11 films. La déser vit dans l'instrument (`PeekBits(pay, at, 45)`), PAS dans
      `traverse.go` : porter en production sans le binaire dépasse le périmètre « verdict ».
- [x] 0.2 Instruments de recherche gatés par env var `TI47_FILM` (patron exact de
      `filmdec/i22_delta_research_test.go` : LECTURE SEULE, sauté partout ailleurs, CI
      comprise, `CGO_ENABLED=0`), pointés sur `data/cache/film_chunks/<match>` du dépôt
      principal. — FAIT : `filmdec/ti47_annonces_{scan,largeur,}_test.go`, gate `TI47_FILM`
      (`SKIP` vérifié sans la variable), sorties TSV sous `registre_film/lotF2/`.
- [x] 0.3 Recensement masque -> atteint sur >= 6 films couvrant >= 3 modes (2 Strongholds,
      2 KOTH, 1 CTF témoin négatif, 1 Slayer témoin négatif), AVEC filtre réel/fantôme
      (rattachement à une vie/entité confirmée, leçon i27). — FAIT sur **11 films / 5 modes**
      (2 Strongholds, 4 KOTH, 1 Oddball, 3 CTF, 1 Slayer), bande observée en image-clé contre
      fantôme de même cardinalité, taux hors grammaire et vies (slot,gen) publiés.

Gate 0 : taux d'atteinte du masque publié par film ; densités par mode cohérentes avec le
recensement du lot C (77-81 % en zones, quasi 0 en CTF) — sinon, le curseur est mal placé :
STOP et consigner. Clore avant phase 1.

**GATE 0 — PASSÉ (2026-08-24).** Part de records ti=47 annonçant i2, mesure de ce lot contre
le lot C : 81,15 / 76,29 (Strongholds, lot C 80,53 / 77,23) · 74,93 / 78,89 / 81,12 / 77,70
(KOTH, lot C 74,00 / 80,08 / 80,97 / 79,90) · 10,93 (Oddball, 11,00) · 0,54 / 1,71 / 0,50
(CTF, 0,55 / 1,72 / 0,42) · 1,16 (Slayer, 1,16). Écart maximal 1,2 point : le curseur est au
même endroit qu'au lot C. Détail et mesures de largeur : `registre_film/LOTF2_TI47.md`.

## Phase 1 — Sémantique par corrélation

- [ ] 1.1 Cardinalité : les valeurs de i2 forment-elles une ÉNUMÉRATION (peu de valeurs
      distinctes, répétées) ou un continuum/treillis (comme i1) ? Publier l'histogramme.
- [ ] 1.2 Alignement temporel : distance de chaque émission i2 à l'oracle le plus proche
      (capture de zone, bascule de possession, début/fin de contestation si mesurable,
      période de colline KOTH). Fenêtres à publier (médiane, p90) ; témoin = instants tirés
      de la même loi hors événements.
- [ ] 1.3 Séparation des classes : si énumération, chaque valeur corrèle-t-elle à UN type
      d'événement (capture vs perte vs contestation vs annonce de manche) ? Matrice
      valeur x type d'oracle.
- [ ] 1.4 Le petit plus produit : y a-t-il des classes SANS oracle connu (candidates
      « contestation », que `zoneStates` ne sait pas mesurer) ? Si oui, les dater et
      proposer une vérification visuelle (matchs + timestamps pour le Theater du user).
- [ ] 1.5 Verdict écrit : positif (table de classes, couverture, fenêtres) / négatif
      (mesures à l'appui) / conditionnel (ce qui manque et pourquoi le corpus ne suffit pas).

Gate 1 : chaque affirmation du verdict est adossée à un chiffre reproductible par une
commande du rapport. STOP : rendre le CR. Toute exploitation produit (effet UI à la capture,
publication d'un champ) = lot séparé après arbitrage superviseur et décision utilisateur.

## Garde-rails d'exécution

- Commandes `go` : UNE à la fois, GOCACHE PRIVÉ au lot ($env:GOCACHE vers un dossier dédié).
- Plafonner toute accumulation par composant (la bombe RAM `NamedEventsFrom`/`incrementTimes`
  — OOM ~26 Go — est au registre : ne pas la reproduire dans un instrument).
- Données : chunks du dépôt principal, lecture seule ; aucun réseau.
- Aucun fichier du dossier `.ai/` du principal modifié par ce lot (journal et registre =
  superviseur) ; le rapport vit dans le worktree.

## Découvertes

(consigner ici — rien corriger)

## CR attendu

Rapport `.ai/V7.5/replay2d/registre_film/LOTF2_TI47.md` (dans le worktree) : mesures des
gates 0 et 1, verdict 1.5, et si positif une proposition d'exploitation MINIMALE (champ
d'artefact envisagé, coût, schéma requis — SANS l'implémenter). Statut de chaque item.
Commits atomiques `ti47(pN): ...`, JAMAIS `git add -A`, instruments gatés seulement.
