# Plan : évolutions de la page de rejeu et KPI Mainmise (2026-11-10)

Statut : **go du user le 2026-11-10, exécution en cours**. Exécution sous le skill `plan-execution`.

Référence visuelle : https://claude.ai/artifact/Hpw99Ki4YEXT9ZQj1M3sZR. Sources des maquettes copiées dans `.ai/annexes_maquettes_rejeu/` (`ReplayPage.dc.html`, `ChronoPage.dc.html`, `ChronoReel.dc.html`, `MainmisePage.dc.html`, `Main.dc.html`). L'exécutant lit la source de la maquette du lot avant de coder : elle fait foi pour la disposition, les libellés et les durées d'animation ; le code de l'application fait foi pour les composants, les tokens et les règles.

## 0. Objectif et critère de succès

Six évolutions validées par le user le 2026-11-09, livrées dans cet ordre :

1. **Fil d'événements filtrable** : le titre du fil devient un menu (Tout, Éliminations, Objectifs, Armes et bonus, Véhicules, Médailles, Assistances), les objectifs entrent dans le fil en bandeaux.
2. **Clic sur une élimination** : la lecture saute à l'instant, le tueur et la victime sont mis en valeur sur la carte (voile de 10 %, équerres qui se verrouillent, noms en étiquettes avec l'arme). Le bouton Lecture retire tout.
3. **Bouton « Vue »** : préréglages de calques (Complet, Combat, Déplacements, Objectif, Personnalisé), en remplacement du tiroir de réglages.
4. **Fiche de joueur en panneau** : un clic sur une fiche ouvre un panneau à la place du fil, avec Suivre, Seul et Masquer.
5. **Graphiques** : « Écart de score » et « Supériorité numérique » s'ajoutent à « Score dans le temps », dans l'onglet Chronologie de la vue du match.
6. **Mainmise** : part du match passée en tête du score, plus le nombre de retournements, sur la vue du match, Sessions, Timeseries et Escouade.

Le plan est réussi quand chaque lot a passé son gate technique et sa vérification à l'écran par le user, et que la revue adversariale finale est traitée.

## 1. Constats sur pièces (inventaires du 2026-11-10, `release/v8.1.2`)

Chemins web relatifs à `apps/web/src/features/match-replay/` (« MR »), Go relatifs à `apps/go-api/`.

- **Fil.**
  - `ui/ReplayKillFeed.tsx` (496 lignes, au plafond de 500). Entrées : `model.feed` (`model/replayModel.ts:166-179`, `model/killFeedLogic.ts:488`).
  - Une entrée est un frag (avec ses médailles rattachées à ± 500 ms et son assistance), une médaille seule, une mort neutre ou une entrée/sortie de partie.
  - Aucun filtre, aucune ligne d'objectif.
  - Le même `model.feed` nourrit la frise et les effets de mort : un filtre ne doit porter que sur l'affichage.
- **Données déjà servies à la page.**
  - Médailles : datées à la milliseconde, dans le contexte du rejeu.
  - Assistances : portées par chaque frag (`assist_state`, nom, part de dégâts), datées de l'instant du frag.
  - Objectifs : `doc.objectives[]` (`{t, xuid, stat, timeMs}`), familles `flag|zone|hill|skull|vip|bomb` (`model/objectiveFamilies.ts`).
  - Aucun libellé par action d'objectif n'existe côté rejeu.
  - Toute datation venue du document exige `filmClockTrusted(doc)` (`lib/replay/scoreTimeline.ts:195`).
- **Clic.** Rien n'est cliquable dans le fil. `seekTo` vit dans le canvas (`hooks/useReplayPlayback.ts:409`) ; `model/playbackStore.ts` est en lecture seule pour la page. Aucun mécanisme de mise en valeur d'un joueur sur la carte (`layers/replayMarkers.ts`, `layers/replayLabels.ts`). `ui/ReplayCanvas.tsx` fait 781 lignes.
- **Réglages.** `settings/ReplaySettingsDrawer.tsx`, état dans `settings/useReplaySettings.ts`, persisté en localStorage par `settings/replayPreferences.ts`. Aucun préréglage. Le tiroir porte aussi Effets, Chaleur, Couleur des points et Son.
- **Fiches.** `ui/ReplayTeams.tsx`, `ui/ReplayPlayerCard.tsx`. Rien n'est cliquable. Suivre, Seul et Masquer n'existent pas. Tirs, précision et détail du score sont sur la ligne de feuille du contexte (`MatchScoreboardRow`), en totaux de fin de match, pas dans le document.
- **Chronologie.** `features/match-view/MatchViewTabChronology.tsx`. « Score dans le temps » : `MatchScoreCurveChart` + `_scoreCurve.ts` (expose déjà `leads`). `lib/replay/scoreTimeline.ts` porte `leaderStates` (`:395`) et `leadChanges` (`:351`). Aucun calcul de joueurs en vie par équipe ; `isAliveAt(track, frame)` existe (`lib/replay/replayLogic.ts:458`).
- **Score.** `ScoreTimeline.Teams[].Total` (cumulé, en images du document) est la seule forme utilisable pour un temps en tête. `TeamID` absent = identité non résolue : ne rien deviner. La table `[score_timeline]` de `regulation.toml` classe les modes en `curve`, `events` (peu de points : CTF, KOTH, bombes) et `hidden` (Assassin), servie dans `header.score_timeline_kind`.
- **Dérivé par match.** Recette de `match_vehicle_takes` : projection pure, capability fine `film.<x>`, migration (table append-only + vue `_latest`), persister INSERT-only, étape dans `Deriver` (`internal/sync/replayartifacts/derivations.go:112`), commande `backfill-*`, garde-rails anti-ART, lecture par port et repo sur `_latest`.
- **Pages.** Vue du match : `MatchViewHeader`. Sessions : `SessionDetailMatchRow`, `SessionCompareEntry`, table `ExplorerMatchesTable`. Timeseries : `TimeseriesMatchRow`, `MapBreakdownRow`, `OutcomesOverTime`. Escouade : `SquadDynamiquePage.tsx`, `PerformanceSeries`.
- **Noms pris.** « Dominance » désigne déjà un badge narratif et un graphique. « Mainmise » est libre.

## 2. Décisions

| # | Décision | Raison |
|---|---|---|
| D-1 | **Le filtre du fil ne touche que l'affichage** ; `model.feed` reste entier | La frise et les effets de mort lisent le même fil. |
| D-2 | **Filtres** : Assistances = frags dont l'assistance est nommée ; Médailles = médailles seules et frags qui en portent ; Objectifs = actions de `doc.objectives[]` hors frags et assistances. **Révisé le 2026-11-10 (user)** : Tout, Éliminations, Médailles, Assistances, Armes et bonus et Véhicules sont TOUJOURS dans le menu ; seul Objectifs est conditionnel (absent d'un match sans objectif ni horloge sûre, test sur un Assassin). Un filtre sans aucune ligne dans tout le match affiche « Aucune entrée pour ce match », texte distinct de « Rien à cet instant » (le match a des lignes, plus tard). Le filtre mémorisé reste appliqué tel quel ; le repli sur Tout ne concerne que Objectifs sur un match sans objectif, sans écraser la préférence. | Sémantique de la maquette (`matchFilter`). Le user préfère un menu stable et un message explicite à un menu qui change d'un match à l'autre. |
| D-3 | **Armes et bonus, Véhicules** : VALIDÉS par le user (2026-11-10), toujours dans le menu. Leurs lignes sont construites depuis ce que le document porte déjà de daté (aucun nouveau champ, aucune recuisson) ; si l'une des deux n'avait RIEN de daté exploitable, arrêt sur ce point et rapport précis de ce qui manque, jamais une entrée vide en permanence ni écartée. Vérifié en 1.1 : les deux ont des événements datés (voir 1.1). | Aucun changement de document ni recuisson dans ce plan. |
| D-4 | **Le choix du filtre est mémorisé** dans le navigateur (`replayPreferences`) ; la mise en valeur d'un kill et le joueur suivi ne le sont pas | Même règle que le point de vue, non persisté. |
| D-5 | **Saut depuis la page** : un canal de demande dans `playbackStore`, que le canvas exécute. Le canvas reste le seul à écrire le temps. | Respecte « le canvas écrit, tout le monde lit ». |
| D-6 | **Le bouton Vue remplace les Calques du tiroir** ; Effets, Chaleur, Couleur des points et Son restent accessibles dans le même panneau, sous les préréglages | Validation du user : « réglages déplacés là ». Rien ne disparaît. |
| D-7 | **Suivre** zoome et centre la carte sur le joueur ; **Seul** n'affiche que lui ; **Masquer** le retire. Seul coupe Masquer ; Suivre coupe Masquer ; Masquer coupe Seul et Suivre. | Validé le 2026-11-09. |
| D-8 | **Précision et détail du score du panneau = totaux de fin de match**, libellés comme tels. Pas de précision « à l'instant ». | Le user attend l'article de Dend pour la précision en temps réel. |
| D-9 | **Écart de score et Mainmise lisent le score du rejeu** (`Teams[].Total`), y compris en Assassin. En mode `events`, l'écart s'affiche en marches et Mainmise affiche un tiret. | Mesure sur 174 rejeux : fiable sur les modes à score continu, pas sur les modes à peu de points. |
| D-10 | **Supériorité numérique = joueurs en vie de l'équipe moins ceux d'en face, bots compris**, échantillonnée sur l'axe du match | Règle « les bots comptent partout ». |
| D-11 | **Le calcul de Mainmise vit dans `internal/analysis/`**, pas dans le décodeur, avec un jeu de cas partagé avec `leaderStates` côté web | Algorithme pur ; le décodeur n'est pas modifié ; une seule définition du meneur (égalité = personne en tête, la première prise de tête n'est pas un retournement). |
| D-12 | **Mainmise est un dérivé par match persisté** (table append-only + `_latest`), unité 0..1, agrégée par moyenne des matchs mesurés | ADR 0006, 0026, 0039. Un match sans mesure ne compte ni comme 0 ni dans le dénominateur. |
| D-13 | **Emplacements de Mainmise** : vue du match (en-tête entre Performance et Rang, et carte Écart de score) ; Sessions (match par match, après Vies nettes, avant Intensité ; comparaison de sessions ; pas de bloc « Soirée ») ; Timeseries (par mois après Performance solo ; par carte après la paire Taux de victoire / Performance par carte) ; Escouade (Dynamique, après Vies nettes / Écart cumulé, avant Engagement) | Validé le 2026-11-09. |
| D-14 | **Courses de drapeau : hors plan** | Non validé sous cette forme. |

## 3. Dépendances, branche, gestes du user

- **Branche** `feat/rejeu-maquettes`, depuis la branche d'intégration en cours au lancement, dans un worktree dédié. Lots 1 à 5 : web seul. Lot 6 : `apps/go-api` et web.
- **Après** la fusion du lot « seuils en millisecondes » du plan 30 images/s (même dossier `features/match-replay`) et le renommage des imports de la v8.1.2 (lot 6).
- **Gestes du user** : le go ; la vérification à l'écran de chaque lot (inscrite dans la page des vérifications, collection `verifs`) ; le rétro-remplissage de Mainmise sur le parc (6.6) ; la fusion.

## 3 bis. Contrat d'exécution

- Skill `plan-execution`. Ordre strict des lots. Statuts `[x]`, `[~]` (référence), `[!]` (justification). Aucune case vide à la clôture. Un commit au moins par lot.
- Zéro correctif hors périmètre : section 8.
- Reprise : Journal (section 9), puis `git log --oneline -10`.
- Effort : lots 1 et 3 moyens ; lots 2, 4 et 5 moyens à lourds (dessin sur la carte) ; lot 6 lourd (base, quatre pages).
- Exécutants Sonnet, un worktree, un lot à la fois. Une revue adversariale à la fin (une ronde), deux relecteurs pour le lot 6 (écritures en base).
- Règles web : logique dans `model/` ou `*_logic.ts`, composants dans `ui/`, dessin dans `layers/` ; textes dans `i18n/fr.ts` et `i18n/en.ts` (contrat typé) ; tokens sémantiques seulement ; fichier ≤ 500 lignes, fonction ≤ 80. `ReplayKillFeed.tsx` et `ReplayCanvas.tsx` sont au plafond : extraire avant d'ajouter.
- Gate web de chaque lot : `make check-types` ; `make test-web` ; `npx eslint` sur les dossiers touchés. Gate Go du lot 6 : `make go-api-test` ; `cd apps/go-api && go test ./...` ; `go test -tags=integration -p 1 ./...` ; `make go-api-lint` ; `make openapi-gen` et `make generate-types` sans diff résiduel.
- TDD : la logique pure de chaque lot est écrite test d'abord.
- Jamais de navigateur ouvert par un agent. Le rendu se vérifie par le user ; l'agent fournit la liste exacte de ce qu'il doit voir.

## 4. Lots

### Lot 1 — Fil d'événements filtrable
- [x] 1.1 **Vérifier sur pièces** ce que le document porte de daté pour « Armes et bonus » (`pickups`, changements d'arme) et « Véhicules » (prises de véhicule), et statuer D-3 pour chacune. **Armes et bonus : OUI.** `doc.padPickups[]` porte, pour les prises sur socle (armes ET bonus : surbouclier, camouflage), `t` (image, instant exact par le ramassage natif `biped_pickup`) et `xuid` ; le nom vient de `weaponPads[pad].weapon` via `padNameFor` (table locale des bonus, sinon `weaponLabels`). Lignes retenues : `t` ET `xuid` présents seulement. Non repris : `doc.pickups[]` brut (toutes les prises, ~100-230 par match, ramasseur parfois absent, familles d'équipement sans table de noms côté web) et les prises sans date (intervalle des images-clés, environ 20 s). **Véhicules : OUI, avec réserve.** `doc.vehicles[].rides[]` porte l'occupant (`xuid`), le début `t0` (`src` = `event` ou `mixed` : à la milliseconde ; `gap` écarté, non daté) et le siège. Lignes = montées (« est monté à bord : Warthog ») ; un épisode qui chevauche le précédent du même joueur dans le même véhicule n'est pas une seconde montée ; décor, éléments de carte et vies de décor écartés. Réserve à remonter au user : la primitive d'occupation ne nomme que 15,6 à 21,1 % des vies de véhicule (commentaire de `VehicleRide`, `libs/film/replay/document_vehicles.go`) : la liste est fidèle, pas complète ; et le document ne porte AUCUNE destruction (`end` vaut toujours `unknown`) : la ligne « a détruit le Ghost de ... » de la maquette n'est pas réalisable sans changement du décodeur.
- [x] 1.2 **Extraire** de `ui/ReplayKillFeed.tsx` l'en-tête et les lignes en fichiers voisins, sans changement de rendu (tests existants verts tels quels).
- [x] 1.3 **Logique pure** `model/feedFilterLogic.ts`, tests d'abord : familles d'une entrée, prédicat par filtre (D-2), filtres disponibles pour un match, clé de préférence.
- [x] 1.4 **Lignes d'objectif** : `model/objectiveFeedLogic.ts` construit les entrées depuis `doc.objectives[]` (hors frags et assistances), sous `filmClockTrusted`. Une ligne en bandeau (`ui/ReplayObjectiveLine.tsx`) : horloge, pictogramme de famille, joueur à la couleur de son équipe, libellé de l'action. Libellés FR et EN pour chaque action des familles `flag`, `zone`, `hill`, `skull`, `vip`, `bomb` (liste fermée tirée de `libs/film/internal/facts/objectives/named.go`, écrite dans le commit). Elles s'insèrent dans la liste affichée, pas dans `model.feed` (D-1).
- [x] 1.5 **Menu** : le titre du fil devient un bouton-menu (composant de menu existant du projet), avec l'icône et le libellé du filtre actif. Navigation clavier, `aria`.
- [x] 1.6 **État vide** : un filtre sans ligne à l'instant courant affiche le bloc vide en pointillé du projet, jamais une liste absente.
- [x] 1.7 Tests de composant : le filtre ne change ni la frise ni les effets de mort ; les règles du fil (une ligne, pas de retour, défilement interne) tiennent pour la ligne d'objectif.

Gate : gate web. Vérification du user : sur un CTF, un Bases et un Assassin, ouvrir le menu, passer chaque filtre, vérifier les bandeaux d'objectifs et que la frise ne bouge pas.

### Lot 2 — Clic sur une élimination
- [x] 2.1 **Canal de saut** (D-5) : demande de saut dans `model/playbackStore.ts` (`requestSeek` / `onSeekRequest`, une demande faite avant le montage du canvas attend, la dernière seule), exécutée par `useReplayPlayback` (option `onSeekRequest` : pause, puis `seekTo` borné à la fenêtre). Une demande de la page n'est pas un geste manuel. Tests du store (25) et du lecteur (`useReplayPlayback.killFocus.test.tsx`, 12). L'image visée est la première dont l'instant n'est pas antérieur à celui du frag (`frameAtOrAfter`) : arrondie vers le bas, la ligne cliquée disparaîtrait du fil (`feedAt`).
- [x] 2.2 **État de mise en valeur** `model/killFocusLogic.ts`, tests d'abord (26) : cible d'une ligne, état (`reduceKillFocus`, jeton par clic : un clic de plus rejoue, même sur la même ligne), résolution sans position inventée (tueur et victime séparément ; fiche occupée, couleur, position requises), slots dont le nom cède la place. Effacé par Lecture, Recommencer, glissé de frise, sauts, pas d'image, saut vers un repère (`onUserMove` du lecteur, donc leurs raccourcis clavier), remplacé par un autre clic, effacé au changement de document et au démontage ; jamais mémorisé (tenu par le magasin de la page). Les gestes sont dans `model/killFocusMotion.ts` (courbe `cubic-bezier(.2,.8,.2,1)` appliquée avant les images-clés, comme la maquette ; 11 tests).
- [x] 2.3 **Ligne cliquable** : la ligne de frag devient un bouton (`aria-pressed`, infobulle FR et EN `killFeedFocusHint`) SEULEMENT quand la page fournit le magasin de lecture et le document ; les autres formes de ligne ne le sont pas. « Sélectionnée » = `bg-muted` + deux filets aux encres de camp du tueur (gauche) et de la victime (droite). Clavier : Entrée et Espace activent ; Espace ne remonte pas au raccourci global de lecture ; la souris ne prend pas le focus (`mousedown` annulé) pour que Espace reste « Lecture » après un clic. Le filtre du lot 1 est intact (14 tests de composant, `ReplayKillFeed.focus.test.tsx`).
- [x] 2.4 **Calque de mise en valeur** `layers/killFocusLayer.ts` + `layers/useReplayKillFocus.ts` : voile de 10 % (encre `--scrim`, ajoutée à `canvasInk`) percé d'un rond de 34 px par joueur, équerres ouvertes aux quatre coins (demi-côté 22, branche 8, trait 3) à la couleur de camp, verrouillage 340 ms (échelle 2,2 et rotation 45 degrés vers 1 et 0, dépassement 0,88 à 72 %), fondu du voile 240 ms, jouées à chaque clic par le temps RÉEL, boucle d'images d'écran qui s'arrête à 340 ms ; `prefers-reduced-motion` : poses d'arrivée, aucune boucle. Deux ids dans `LAYER_ORDER`, en dernier : `mise-en-valeur` (voile et équerres, au-dessus des pions et des morts) puis `etiquettes-mise-en-valeur`.
- [x] 2.5 **Noms en étiquettes** : étiquette pleine du tueur (couleur de camp, texte sombre, pictogramme de l'arme teinté, ou rien quand l'arme n'est pas identifiée) et étiquette en creux de la victime (liseré et texte de camp, croix), coin bas droit coupé, 13 px en graisse 900 à 27 px sous le pion. Le nom normal sous le pion est retiré pour ces deux slots (`MarkerStyle.nameHiddenAtSlot`, qui ne touche pas la flèche d'un joueur hors cadre). Un acteur sans position, hors cadre, sans nom ou sans fiche n'a ni équerre ni étiquette.
- [x] 2.6 `ui/ReplayCanvas.tsx` : 485 lignes de code avant, 484 après (plafond 500, mesuré par `max-lines` ESLint). Extrait : la préparation de la toile de `draw` (`layers/canvasStage.ts`, `prepareStage`, 4 tests), sans changement de rendu ; les tests existants du canvas passent sans modification (seuls `replayCompose.test.ts` et `sceneBinding.guard.test.ts` suivent les deux nouveaux ids de calque).

Gate : gate web. Vérification du user, **en lecture et à l'arrêt** : cliquer plusieurs éliminations de suite, vérifier l'animation à chaque clic, l'étiquette avec l'arme, puis Lecture qui retire tout.

### Lot 3 — Bouton « Vue »
- [ ] 3.1 **Logique pure** `settings/viewPresetLogic.ts`, tests d'abord : quatre préréglages (Complet, Combat, Déplacements, Objectif) comme ensembles de calques ; « Personnalisé » dès qu'un calque diffère d'un préréglage ; un préréglage n'active jamais un calque indisponible pour le match (`available.*`).
- [ ] 3.2 **Bouton et panneau** en haut à droite de la carte : préréglages en premier, calques ensuite, puis Effets, Chaleur, Couleur des points et Son (D-6), repris des composants `SettingsToggle`, `SettingsChoice`, `SettingsSegments`.
- [ ] 3.3 **Retrait du tiroir** `ReplaySettingsDrawer` et de son bouton, avec leurs tests et imports (0 code mort). Les préférences déjà mémorisées restent lues telles quelles.
- [ ] 3.4 Le préréglage actif est mémorisé (`replayPreferences`).

Gate : gate web. Vérification du user : chaque préréglage, un calque basculé (passage en Personnalisé), fermeture et réouverture de la page, et la place gagnée sur la carte.

### Lot 4 — Fiche de joueur en panneau
- [ ] 4.1 **Logique pure** `model/playerPanelLogic.ts`, tests d'abord : joueur choisi ; règles d'exclusion de D-7 ; ce que le panneau affiche pour un joueur non publié (rien d'inventé, jamais de zéro).
- [ ] 4.2 **Fiche cliquable** (bouton accessible) ; le panneau prend la place du fil, « Retour au fil » le referme. Composants de liste du projet, pas de mini-cartes.
- [ ] 4.3 **Contenu** : compteurs au format de la fiche ; précision et détail du score en listes, lus sur la ligne de feuille du contexte, libellés « sur le match » (D-8).
- [ ] 4.4 **Suivre** : zoom et centrage de la carte sur le joueur, image par image, sans changer le temps ; s'arrête si le joueur n'a plus de position (la carte reste où elle est).
- [ ] 4.5 **Seul et Masquer** : filtrage du dessin des pistes, des noms et des effets du ou des joueurs concernés. Un joueur présent reste dans les fiches, même masqué sur la carte.
- [ ] 4.6 Le point de vue (`useReplayViewpoint`) n'est pas modifié par ces actions.

Gate : gate web. Vérification du user : ouvrir une fiche, Suivre en lecture, Seul puis Masquer (exclusions), Retour au fil, sur une partie à 4 contre 4 et une grande partie.

### Lot 5 — Graphiques
- [ ] 5.1 **Écart de score** : logique pure `features/match-view/_scoreDelta.ts`, tests d'abord, à partir des séries de `_scoreCurve.ts` : écart signé de l'équipe du joueur dans le temps, marches en mode `events`. Composant `MatchScoreDeltaChart.tsx` par `ChartCard` : aire au-dessus et en dessous de zéro, tokens divergents, infobulle (instant, score, écart), titre neutre, bouton d'information, légende en bas.
- [ ] 5.2 **Supériorité numérique** : logique partagée `lib/replay/aliveBalanceLogic.ts`, tests d'abord (D-10), sur les pistes du document et l'horloge `matchClock` ; composant `MatchNumbersAdvantageChart.tsx`, mêmes conventions. Masqué si le rejeu n'est pas disponible.
- [ ] 5.3 **Ordre** dans `MatchViewTabChronology.tsx` : Frags cumulés, Points marqués, Score dans le temps, Écart de score, Supériorité numérique, Dominance, Cadence. Libellés FR et EN dans `features/match-view/i18n/`.
- [ ] 5.4 Garde-rails des graphiques (`components/charts/`) verts ; frontière entre rejeu et vue du match respectée (logique partagée dans `lib/`).

Gate : gate web. Vérification du user : trois matchs (serré, retournement, à sens unique), les trois graphiques alignés sur le même temps.

### Lot 6 — Mainmise
- [ ] 6.1 **Calcul pur** `internal/analysis/leadshare/`, tests d'abord : depuis `Teams[].Total`, le pas du document et l'origine, rendre par camp le temps en tête, le temps à égalité, la durée mesurée et le nombre de retournements. Jeu de cas JSON partagé avec `leaderStates` côté web (test de parité des deux côtés). Refus mesuré : identité de camp non résolue, horloge sans origine, mode `events`, score non couvert.
- [ ] 6.2 **Dérivé persisté** selon la recette de `match_vehicle_takes` : capability `film.lead_share` (Halo Infinite seulement), migration (table append-only `match_lead_share` + vue `match_lead_share_latest`, ligne de match toujours écrite avec sa couverture), persister INSERT-only, étape dans `Deriver`, compteurs au journal, garde-rails anti-ART et de lecture `_latest` mis à jour, skill `db-schema`.
- [ ] 6.3 **Commande** `levelup backfill-lead-share [--match] [--dry-run] [--force]`, lecture par `replayartifacts` seulement ; `docs/COMMANDS.md` FR et EN.
- [ ] 6.4 **Lecture** : port et repo sur `_latest`. Champs servis (0..1, absent = non mesuré) : `lead_share` et `lead_changes` sur l'en-tête de la vue du match, `SessionDetailMatchRow`, `SessionCompareEntry` (moyenne des matchs mesurés et leur nombre), `TimeseriesMatchRow`, agrégats par mois et par carte, série d'Escouade. `make openapi-gen`, `make generate-types`.
- [ ] 6.5 **Web** (D-13) : en-tête de la vue du match et carte Écart de score ; colonne de Sessions et ligne de comparaison normalisée ; Timeseries par mois et par carte ; Escouade, section Dynamique. Tiret quand non mesuré, jamais « N non mesurés ». Entrée d'aide et libellés FR et EN (« Mainmise »).
- [ ] 6.6 **Rétro-remplissage du parc** : geste du user (`backfill-lead-share`, lecture des rejeux seulement, aucune recuisson). L'agent fournit la commande et la durée estimée sur 20 matchs.
- [ ] 6.7 Démo : graines mises à jour si la démo sert ces pages.

Gate : gate Go et gate web. Vérification du user : Mainmise sur la vue d'un match à score continu, tiret sur un CTF, puis Sessions, comparaison de sessions, Timeseries et Escouade.

### Lot 7 — Clôture
- [ ] 7.1 Revue adversariale du diff complet, une ronde (deux relecteurs pour le lot 6). Chaque constat corrigé ou statué.
- [ ] 7.2 `docs/` (guides bilingues touchés), `.ai/thought_log.md`, registre des reports.

## 5. Risques

| Risque | Parade |
|---|---|
| Deux fichiers au plafond de taille | Extraction d'abord, sans changement de rendu (1.2, 2.6) |
| Mise en valeur correcte à l'arrêt, fausse en lecture | Vérification du user en lecture (lot 2) |
| Mainmise et graphique de l'écart qui divergent | Une seule définition du meneur, parité testée (D-11) |
| Rejeux anciens sans score ou sans origine | Refus mesuré, tiret à l'écran (6.1) |
| Entrées « Armes et bonus » et « Véhicules » sans données datées | Statuées en 1.1 (D-3) |

## 6. Hors périmètre

Courses de drapeau. Précision en temps réel (attend l'article de Dend). Dessin des véhicules et planche des pions. Chatter. Tout changement du document de rejeu ou du décodeur.

## 8. Découvertes

- **Lot 1, familles colline et crâne.** `named.go` ne nomme AUCUNE action pour `hill` ni `skull` (KOTH et Oddball rendent nil) : la liste fermée des libellés compte onze actions (flag x7, zone x2, vip, bomb), pas les six familles entières. Le garde-rail `objectiveFeedStats.guard.test.ts` rougit le jour où la table en nomme une. Pictogrammes colline et crâne déjà prêts (`model/feedGlyphs.ts`).
- **Lot 1, garde-rail web sur un fichier Go.** Le garde de parité lit `libs/film/internal/facts/objectives/named.go` directement : ce fichier n'est pas dans l'instantané `film-source` (produit par `cmd/film-web-snapshot`, côté Go) et le filtre `croisee` de `ci.yml` ne couvre pas `libs/film/internal/facts/**`. Un changement de `named.go` seul ne lancerait donc pas ce test en CI. Correctif hors périmètre (Go + CI) : ajouter le fichier à l'instantané et au filtre.
- **Lot 1, composant de menu.** `components/ui/` n'a aucun menu déroulant ; le menu du fil reprend le patron du menu de vitesse du rejeu (`ReplaySpeedMenu`) avec `role=menu`, entrées `menuitemradio` et navigation clavier du sélecteur de joueur (`PlayerSwitcher`).
- **Lot 1, destruction de véhicule.** Le document ne porte aucune destruction (`end` toujours `unknown`) : la maquette montre « a détruit le Ghost de ... », irréalisable sans changement du décodeur (hors plan, D-3).
- **Lot 1, Armes et bonus.** Seules les prises sur socle datées et nommées entrent (`padPickups`) ; les armes ramassées au sol (`pickups` brut, `groundWeapons`) n'y sont pas : à décider avec le user si elles doivent y figurer.

- **Lot 2, destruction et décalage de la croix.** La maquette (variante « projecteur ») dessine aussi une croix blanche sur la victime ; non reprise : la croix de mort rouge existante marque déjà l'endroit, et l'étiquette de la victime porte sa propre croix (le plan ne cite que voile, équerres et étiquettes).
- **Lot 2, tueur embarqué.** La position d'un joueur embarqué est celle de son véhicule (`carrierPosition`) : l'équerre et l'étiquette se posent sur le véhicule, dont le calque écrit déjà le nom du conducteur : second affichage possible du nom dans ce cas rare, non traité (hors périmètre, à voir à l'écran sur un frag au volant).
- **Lot 2, densité d'écran.** Les calques multiplient leurs tailles d'écran par `k` (= `canvasPixelRatio`) alors que `draw` pose déjà `setTransform(dpr)` : sur un écran à densité supérieure à 1, les repères pourraient sortir plus grands que prévu. Le lot suit la convention commune (même `k` que les noms et les pions) ; constat non vérifié à l'écran, non traité.
- **Lot 2, `frameToMs`.** Sa signature passe de `ReplayDocumentReady` à `Pick<ReplayDocumentReady, 'frameIntervalMs'>` (élargissement de type seul, comme `msToFrames`), pour que la logique pure de la mise en valeur se teste sans document complet.

## 9. Journal

- **2026-11-10, lancement.** Go du user. Branche `feat/rejeu-maquettes` depuis `origin/release/v8.1.2` (après la fusion de la PR 190), worktree `LevelUp-wt-maquettes`. Précision du user sur D-2 : le filtre Objectifs n'apparaît que sur un match qui a des objectifs ; c'est la règle générale de D-2 (une entrée sans aucune ligne dans le match n'est pas proposée), à couvrir par un test nommé sur un match Assassin.
- **2026-11-10, changement de décision du user (D-2, D-3).** Médailles, Assistances, Armes et bonus et Véhicules sont toujours dans le menu ; seul Objectifs est conditionnel. Un filtre sans aucune ligne dans tout le match affiche « Aucune entrée pour ce match » (texte distinct de « Rien à cet instant »). Armes et bonus et Véhicules validés : lignes construites depuis le document (1.1). D-2, D-3 et 1.1 mis à jour.
- **2026-11-11, lot 1 livré.** 1.1 à 1.7 faits : menu de filtre, lignes d'objectif en bandeaux, lignes de prises (socles) et de montées en véhicule, deux états vides, préférence mémorisée, extraction de `ReplayKillFeed.tsx` (lignes dans `ReplayFeedLines.tsx`, rangée dans `model/feedRow.ts`, horloge dans `ReplayFeedClock.tsx`). Gate web passé (chiffres au thought_log). Reste la vérification à l'écran par le user.
- **2026-11-11, lot 2 livré.** 2.1 à 2.6 faits : clic sur une ligne de frag (bouton accessible), saut en pause sur l'image du frag, mise en valeur du tueur et de la victime (voile de 10 %, équerres qui se verrouillent en 340 ms, étiquettes de nom avec l'arme), effacée par Lecture et par tout saut manuel. Choix tranché : recliquer la ligne choisie REJOUE l'animation (le user : « rejouée à chaque clic »), alors que la maquette dit qu'un second clic désélectionne ; seule la Lecture ou un saut manuel retire la mise en valeur. Gate web passé : `make check-types` ok ; `make test-web` 1 094 fichiers / 11 367 tests passés (5 et 23 sautés, comme avant) ; ESLint 0 erreur et 8 avertissements, comme avant le lot ; `knip-ratchet` 0/0/0 ; lints couleurs, champs, contrats, imports croisés propres. Reste la vérification à l'écran par le user, à l'arrêt et en lecture.
