# Plan : Tactique v2, vue cockpit — 2026-10-06

> Sources, à lire avant tout lot, qui FONT FOI pour le rendu :
> - maquette validée par l'utilisateur le 2026-10-06, position « Après », largeur « pleine » :
>   `.ai/V7.5/MAQUETTE_TACTIQUE_2026-10-06.html` (script lisible : bibliothèque `RL` l. 484-668 dont
>   `zoneOf` l. 646-666 ; `planSvg` l. 765-792 ; `vLegend` / `rampCss` / `legendBounds` l. 794-812 ;
>   `mapItem` l. 866-881 ; `planInfo` l. 882-890 ; `renderApres` l. 891-984 ; `zoneLabel` l. 992-997 ;
>   `matchTile` l. 1001-1015 ; feuille de style de la vue cockpit l. 194-284, bascule à trois colonnes
>   l. 222-228) ; les lignes « remplace : … », les encarts « Maquette. », la ligne « N autres cartes
>   ouvrables non reproduites » et le pied « budget de hauteur » ne se portent pas ;
> - relevés : `.ai/V7.5/MESURES_TACTIQUE_2026-10-06.md` (Q1-Q13, zones nommées, calculs, longueurs
>   des mini-tuiles) ;
> - `.ai/thought_log.md`, entrées Tactique de septembre (phases 4 à 8, lots M1 et M1b, lot 3.2 du
>   pas adaptatif, retours rejeu L2 « le fond ne bouge plus ») ;
> - plan modèle de la même famille : `.ai/PLAN_TIMESERIES_USAGES_EMPRISE_2026-10-05.md`.
>
> Contrat d'exécution : skill `plan-execution` (ordre strict, un lot à la fois, gate passé avant le
> suivant, aucun item sans statut, zéro fix hors périmètre, découvertes consignées §8). Statuts :
> `[x]` fait et vérifié, `[~]` couvert ailleurs (référence), `[!]` non fait (justification écrite).
> Aucune case vide à la clôture d'un lot. « Clos » = les 5 actions de la règle 6 du skill.
>
> Statut du plan : **PHASE 1 (plan seulement) — rédigé et relu à la grille `plan-review` le
> 2026-10-06 ; aucun lot exécuté.** La phase 2 (exécution, un lot à la fois avec compte rendu et
> attente du « continue ») ne démarre que sur « go » du superviseur. Branche `feat/tactique-v2`
> (créée sur `origin/feat/v75` = `b033d30f0`), worktree `C:\Users\Guillaume\Downloads\Scripts\LevelUp-wt-tactique`.

## 0. Objectif, critère de succès, hors périmètre

**Objectif.** L'onglet Tactique (5e onglet d'Ascension) devient UN écran à trois colonnes sous la
barre de filtres inchangée : « Cartes jouées » à gauche, la carte du plan au centre (titre = nom de
la carte, trois réglages en pilules, fond + calque, rampe verticale), « Zone sélectionnée » à
droite (nom en jeu de la zone, valeur, mini-tuiles « Rejeu »). Une lecture neuve « Solde frags −
morts » ; le nom de zone et les champs des mini-tuiles sont résolus côté Go. Tout ce qui perd son
dernier lecteur est supprimé (règle n° 7) : riposte, coordination d'équipe, tuiles KPI, barre
d'outils, bascule d'écran, H2, pied de carte, avec leurs chaînes Go, tests, chaînes UI et entrées
openapi.

**Critère de succès.** (1) Le rendu suit la maquette « Après », pleine largeur (S1-S16) ; (2) les
sept lectures sont servies, `solde` comprise, chacune avec son unité ; (3) le nom de zone suit la
règle V6 et tient les deux témoins (D3, D26) ; (4) les mini-tuiles portent les champs V7, le
bouton de rejeu n'apparaît que si l'artefact existe ; (5) l'inventaire §4 est supprimé, chaque
preuve grep à 0 ; (6) tous les gates des lots verts, dernière exécution dans la session ;
(7) Halo 5 et toute capability absente dégradent proprement (jamais un 500, jamais une donnée
inventée) ; (8) docs du lot de clôture à jour.

**Hors périmètre** (consigné, non traité) :
- Tout ce qui précède la rangée d'onglets d'Ascension (en-tête, sous-titre, bandeau de conseils,
  onglets) et la barre de filtres (`TacticalFilterBar` sur `useLocalFilterBar`,
  `features/_shared/useLocalFilterBar.tsx:174`) : inchangés. Les autres onglets d'Ascension
  n'héritent que de la largeur (V2).
- Page Escouade : `coordination.Echanges` / `Mesurer` / `Ripostes` et `TacticalRepository.KillEvents`
  (lus par `service/teammates/teammates_squad_echange.go:133-411`) restent.
- Pages Sessions et Séries temporelles (`CoordinationBlock`, `analysis/coordination/bloc.go`) : rien.
- Rejeu 2D : son nommage de zone (centre 3D le plus proche) n'est PAS aligné ici (découverte §8).
- Sidecars, cuisson, recuisson, backfill : rien.
- Page Explorateur : un lien vers elle seulement (L10), aucun changement de sa page.

## 1. Décisions

### 1.1 Validées par l'utilisateur (2026-10-06, brief §2) — fermes

- **V1** La page reste le 5e onglet d'Ascension ; tout ce qui précède les onglets ne change pas.
- **V2** Le `<main>` d'Ascension passe à la largeur des autres pages (`p-6`, sans `container`
  ni `max-w-6xl`) ; tous les onglets en héritent ; gate visuel = celui de l'utilisateur.
- **V3** Barre de filtres inchangée.
- **V4** Un seul écran, trois colonnes ; plus de bascule « Grille / Analyse », de H2, de barre
  d'outils séparée, de bandeau de tuiles KPI, de carte « Coordination d'équipe ». Gauche
  « Cartes jouées » (≈ 208 px, recherche sans accents, liste verticale à défilement interne de
  hauteur FIXE, vignettes ≈ 100 px 16:9, triées par matchs décroissants, carte active surlignée,
  repli « N cartes sous le plancher », la plus jouée choisie d'office sans `?carte=`). Centre :
  titre = nom + ⓘ (dénominateurs, pas réel), pilules « Lecture », « Joueurs » (Escouade désactivé
  sans composition), « Réapparition » ; fond + calque, boîte de 800 px de haut, rapport du fond
  respecté, étiquette discrète du nom de zone, rampe verticale de 220 px au bord droit, bandeau
  d'état des lectures d'artefact ; plus de pied. Droite « Zone sélectionnée » (≈ 360 px, hauteur de
  la carte du plan). La page défile.
- **V5** Lectures et libellés : `morts` « Morts », `kills` « Frags », `solde` « Solde frags −
  morts » (NEUVE), `gagne` « Victoires − défaites », `temps` « Temps de présence », `routes`
  « Trajets après réapparition », `isole` « Morts seul » ; EN « Deaths », « Kills », « Kills −
  deaths », « Wins − losses », « Time on map », « Routes after respawn », « Deaths alone » ; unités
  FR « morts par match », « frags par match », « frags − morts par match », « engagements par
  match », « secondes par match », « passages par match », « morts seul par match ».
- **V6** Nom en jeu de la zone résolu CÔTÉ GO (réponse de `/tactical/{map}/cellule`) via
  `TacticalCalloutsStore.ZonesDeLaCarte` ; règle (a) polygone contenant le centre ET tranche
  [z_bas − 0,25 ; z_haut + 0,25] contenant le z médian ; (b) à plusieurs, la tranche la plus
  étroite qui contient la majorité des événements ; (c) sinon le polygone le plus proche à moins de
  2 m (distance au bord), tranches compatibles d'abord ; (d) sinon pas de nom, « Zone sans nom ».
- **V7** Mini-tuile d'une contribution (contrat `cellule` enrichi, additif) : bande d'issue
  3 px, mode en libellé de l'app, score mon camp d'abord, issue en mot, date · heure au fuseau du
  joueur ; instant, « Tué par … · arme » / « A tué … · arme », badge « seul · N m » / « près · N m »
  (± 1,5 s, portée du radar du match, « seul » sans distance sans coéquipier visible, pas de badge
  sur un frag) ; bouton de rejeu seulement si l'artefact existe, lien `?t=&clock=` ; tri du plus
  récent au plus ancien ; ellipse sur l'arme seule, texte complet au survol.
- **V8** `HEAT_ALPHA_MIN` 0,12 → 0,45, `HEAT_ALPHA_MAX` 0,75 → 0,85 (noyau partagé, voulu) ; la
  rampe CSS de légende suit.
- **V9** Sémantique factuelle et neutre (liste du brief §2.9) ; FR ET EN dans
  `lib/i18n/manifests/tactical.toml` ; clés mortes purgées.
- **V10** Retirés avec chaîne Go, tests, chaînes et openapi s'ils n'ont plus de lecteur : riposte
  (`Echange`), « Coordination d'équipe » (bloc `Coordination`), tuiles KPI (`KPIStrip` ici,
  `Isolement` du raster sans lecteur), `TacticalToolbar`, `TacticalScreenSwitch`, H2, `PiedDuPlan`,
  `tactical.maps.intro`, pied de la grille. La portée du radar reste nécessaire (badge, lecture
  `isole`) : relogée, pas recopiée.
- **V11** Liens croisés au dernier lot (Vue match « Occupation du terrain » → tactique ; vignette →
  Explorateur seulement si l'Explorateur sait filtrer par carte).
- **V12** Découvertes de la maquette consignées §8, non traitées.

### 1.2 Tranchées par le planificateur — à confirmer au « go »

- **D1 — Ordre : Go additif (L1-L3), web (L4-L8), PUIS suppressions Go et contrat (L9), liens
  (L10), clôture (L11).** *Contredit le brief §3* (« Go … → suppressions puis web ») : le web LIT
  aujourd'hui `echange`, `coordination`, `isolement`, `matchs_sans_rayon`
  (`TacticalAnalysisView.tsx:255-263, 382-429`, `TacticalCoordinationCard.tsx:38-51`) ; retirer ces
  champs du contrat avant que le web cesse de les lire ferait rougir `tsc` au gate du lot Go
  (`npm run generate-types` en fait partie). Même ordre que le plan modèle (L5 web puis L6 Go).
- **D2 — Lecture `solde` dans `analysis/tactical`, même machinerie que `CellulesSignees`.**
  `solde.go` (NEUF) : `RasteriseSolde(g, matchs, frags, morts []domain.PositionSample)` rend un
  `*Raster` qui garde les passages par match de l'UNION des deux faces (le plancher en matchs
  distincts, `PlancherMatchsParCellule`, porte sur l'union : maquette `raster` l. 547-550) et le
  compte par face ; `(*Raster).CellulesSolde()` : `Valeur = (frags − morts) / N` (N = matchs
  MESURÉS de l'univers, chaque face normalisée par le même N), `Brut = frags − morts`, `Matchs`,
  `Frags`, `Morts` ; `Somme` propage les comptes par face. `domain.CelluleTactique` gagne `Frags`,
  `Morts` (`omitempty`, lus par la sous-ligne de zone « N frags, M morts · K matchs distincts »,
  maquette l. 966). Service : `validerLecture` (`tactical_service_perimetre.go:37-43`),
  `facesDeLaQuestion` (deux faces, `tactical_service.go:314-334`), `cellulesLisibles` /
  `rasteriser` (pas adaptatif sur les cellules lisibles de la lecture signée,
  `tactical_service_grille.go:36-73`), `remplirRaster` (`EchelleSymetrique`, l. 97-112).
- **D3 — Nom de zone : algorithme PUR `analysis/tactical/zones.go` (+ `geometrie.go`).** Forme d'une
  zone = contour + parties, trous exclus, règle pair-impair (c'est la forme que le rendu dessine,
  `games/halo_infinite/film/replay/callouts_catalog.go:143-152`). Constantes nommées
  `MargeTrancheZM = 0.25`, `RayonZoneM = 2` (strict : « à moins de »). (b) « majorité » = plus de
  la moitié des événements à z connu ; à défaut de majorité pour tous les candidats, celui qui en
  contient le plus, puis la tranche la plus étroite ; toute égalité finale par `VolumeIndex`
  (déterminisme). **z inconnu** (lectures d'artefact `temps` / `routes`, dont les sidecars ne
  portent aucun z ; ou z tous NULL) : (a) sans test de tranche si UN SEUL polygone contient le
  centre ; (c) sans test de tranche si aucun ne le contient ; plusieurs polygones empilés → pas de
  nom (l'étage ne se devine pas). Le port n'est PAS remplacé : `domain.ZoneNommee`
  (`domain/tactical_raster.go:276-283`) gagne `Polygone`, `Parties`, `Trous`, `ZBas`, `ZHaut`,
  `VolumeIndex` (additif) et `zonesNommees` (`service/tactical_callouts.go:75-85`) les projette ;
  le jumeau pur `tactical.ZoneNommee` (`analysis/tactical/spawn.go:62`) idem. Le centre est celui
  de la cellule au pas demandé ((col + 0,5) × pas, (lig + 0,5) × pas).
- **D4 — Les z des événements viennent des DEUX lectures existantes**, pas d'une troisième :
  `QTacticalPositions` (`platform/duckdb/tactical_repo.go:204-223`) gagne `killer_z`, `victim_z`
  (NULL conservé → `*float64`) ; `QTacticalIsolement` (`tactical_repo_isolement.go:59-80`) gagne
  `victim_z`. Face mort = z de la victime, face frag = z du tueur (maquette `points` l. 516-528).
  Mêmes lignes, mêmes fenêtres : coût marginal, une seule définition des positions.
- **D5 — Mode, score, issue, date d'une contribution : le CANONIQUE déjà en cache.**
  `port.PlayerMatchesRepository.LoadPlayerMatches(ctx, slug, gamertag, {MapIDs: [carte]})`
  (`port/player_matches.go:109-127` ; cache invalidé au sync, ADR 0036 I3) — une lecture par
  requête de détail. Mode = `labelPourLocale(Summary.PairMode, locale)`
  (`service/timeseries_service_sections.go:296`, même paquet, aucune copie). Score =
  `analysis.ReadTeamScore` + `FormatTeamScoreLabel` + `ScoreKind`, mon camp d'abord, table
  `rounds_decide` injectée (patron `match_history_service_enrich.go:203-225`,
  `teammates_service_assets.go:297-330`, ADR 0032) — format UNIQUE « X - Y »
  (`analysis/team_score_display.go:140-152`), *écart assumé à la maquette « 3 – 1 »*. Issue =
  `Resultat` canonique existant (`domain/tactical_cellule.go:193-202`), mot et jeton côté web
  (`useOutcomeMapping`, `outcomeTokenFromCanonical`, patron actuel `TacticalCellCard.tsx:189-199`).
  Date = `MatchStartedAt` existant, formatée au fuseau `useAppShellStore(s => s.userTimezone)`
  (patron `components/ui/match-card.tsx:47-60`). Dépendances injectées par `With*` (D15).
- **D6 — Arme = le registre de fragdist.** `source_tag` → `port.KillSourceClassifier`
  (`port/kill_source.go:20-23`, déjà câblé par `r.killSourceClassifierFor(pdb)`) → `weapon_key` →
  `port.WeaponLabelResolver.ResolveWeaponLabels` (`port/weapon_range.go:96-101`, FR + EN depuis
  `weapon_name_labels` / `weapon_names.toml`, mise en œuvre `WeaponRangeRepo`), UN appel par requête
  pour toutes les clés ; publié `arme_label` + `arme_label_en`. À défaut : `categorie_source` brute
  (enum gelée du film) ; le web traduit les quatre catégories de la maquette (`Headshot` « tir à la
  tête », `AttachedDamage` « dégât collé », `SilentMelee` « assassinat », `ChainedProjectile »
  « projectile en chaîne ») et n'écrit rien pour les autres. Titre sans classificateur (Halo 5) :
  ni arme ni catégorie.
- **D7 — Badge de placement.** Nouvelle lecture BORNÉE `TacticalRepository.ContextesDeMort(ctx,
  q)` sur `match_death_context_latest` des seuls matchs des contributions (liste liée en constante
  sur `match_id`, ADR 0036 I2) ; appariement PUR au contexte le plus proche à ± 1 500 ms de
  l'instant pour la même victime (`analysis/tactical/placement.go`, `TolerancePlacementMs`) ; seuil
  = portée du match (`s.rayonsParMatch`, donc `mappings.PorteesDuRadarParMatch`) ; comparaison =
  `coordination.APortee` exportée (borne INCLUSIVE, d = portée → « près » : règle de l'app,
  `analysis/coordination/isolation.go:66-74` ; *la maquette disait « seul » à ≥ 18 m*). Aucun
  coéquipier visible (`PlusProcheM` nil) → « seul » sans distance ; portée inconnue avec une
  distance → pas de badge ; pas de badge sur un frag ni sur une entrée / une réapparition. Publié
  `placement {seul, distance_m}` ; distance tronquée au mètre et « < 1 » côté web. La comparaison
  existe déjà en deux copies (`aPortee`, `vies_pres_ou_seul.go:141-145` ; inline,
  `tactical_service_cellule.go:225`) : le badge serait la troisième → helper exporté + garde-rail
  (CLAUDE.md n° 6, L1.1).
- **D8 — Rejeu.** `ReplayService.AvailableSet` UNE fois par requête (`port/services.go:176-182` :
  forme imposée pour une liste ; même présence d'artefact que `IsAvailable`,
  `service/replay_service.go:62-77`), publié `replay_available` par contribution. Bouton =
  `lib/match-nav/MatchReplayLink.tsx` étendu (prop `search` portant `t` et `clock`, variante
  `large` 36 px = forme de `features/match-view/MatchHeader.replayLink.tsx:37-46`) : une seule
  copie du lien (CLAUDE.md n° 6), ses deux portes (capability `replay` + `available`) gardées ;
  aucun import de `features/match-view` (lot voisin).
- **D9 — Portée du radar relogée.** `TacticalRaster.RayonsRadarM` (`rayons_radar_m`, `omitempty`),
  posé par `rasterIsole` avec `rayonsDistincts` (`tactical_service_isolement.go:191-203`, gardé) ;
  le bloc `Coordination` part en L9. L'ⓘ de « Morts seul » cite la ou les portées, les matchs sans
  portée (`matchs_sans_rayon`) et les morts écartées faute de coéquipier en mesure d'accompagner
  (`morts_equipe_a_terre`, champ publié sans lecteur aujourd'hui : il en gagne un, c'est un
  dénominateur de la lecture).
- **D10 — Zone sélectionnée par défaut = la plus chaude** de la lecture affichée (|valeur| max, à
  égalité le plus de matchs distincts : maquette `selectedCell` l. 751-755, `RL.hottest`
  l. 559-566), tant que l'utilisateur n'a rien cliqué ; remise à zéro au changement de lecture, de
  joueurs, de réapparition ou de carte (mécanisme existant, `TacticalAnalysisView.tsx:118-123`).
  « Aucune zone sélectionnée » quand la lecture n'a aucune cellule.
- **D11 — Carte par défaut.** `carteEffective(scope.carte, cartes)` (pur) : la carte de l'URL si
  elle est ouvrable dans le filtre, sinon la plus jouée ouvrable ; l'URL n'est pas réécrite (pas
  d'entrée d'historique fantôme). Carte de l'URL absente du filtre ou sous le plancher : le plan
  affiche son nom et « Aucun match sur cette carte dans le filtre », sans requête de lecture.
- **D12 — Lectures d'artefact dans la colonne de zone.** Même panneau pour toutes les lectures
  (valeur, matchs distincts, mini-tuiles) ; fait d'une tuile `temps` « Entrée dans la zone »,
  `routes` « Réapparition » (le serveur sert déjà ces contributions,
  `tactical_service_cellule.go:340-377`) ; `questionSansCellule` (`tacticalView.logic.ts:164-166`)
  et ses deux chaînes impératives disparaissent. Champ `face` du contrat : `mort` | `frag` |
  `entree` | `reapparition`.
- **D13 — Grille et hauteurs**, constantes nommées dans `features/tactical/cockpit.logic.ts` :
  colonne des cartes 208 px de large et 551 px de haut (maquette l. 222-228), colonne de zone
  360 px, boîte du plan 800 px de haut au plus, rampe 220 px, marge de légende 90 px ; trois
  colonnes dès 1 400 px de large, en deçà les cartes en rangée défilante au-dessus (maquette
  `.mlist` l. 203). `PLAN_HAUTEUR_MAX_PX` (720, `tacticalView.logic.ts:189`) est remplacé.
- **D14 — Rampe de légende construite depuis la rampe PEINTE** (`heatRamp` / `heatRampDivergent`
  déjà résolues dans `usePeinture`, échantillonnées en arrêts) : elle suit l'opacité V8 sans
  seconde source ; verticale (`0deg`, borne basse en bas, positif en haut).
- **D15 — Dépendances du détail injectées par `With*`** (`tactical_service_cablage.go`) :
  `WithPlayerMatches(repo, slug, gamertag)`, `WithRoundsDecide`, `WithKillSourceClassifier`,
  `WithWeaponLabels`, `WithReplay`. Chaque source est best-effort : nil ou
  `ErrCapabilityNotSupported` → Debug, champ absent ; autre erreur → Warn / Error nominatif, champ
  absent ; la liste des contributions est toujours servie. Une section de durée par source (ADR
  0036 I6). La fabrique `Tactical` quitte `api/wire/registry_pages.go` (619 L, au-delà du seuil)
  pour `api/wire/registry_pages_tactical.go` (NEUF).
- **D16 — Lien vers l'Explorateur.** L'Explorateur filtre par LIBELLÉ de carte, FR d'abord
  (`?maps=`, `explorerScope.ts:13,56,110` ; `filterByExplorerMapNames`,
  `service/match_history_service_filters.go:197-210`), pas par `map_id`. Lien posé avec
  `maps=<map_name_fr || map_name>` SI L10 établit sur pièces que la tactique
  (`mapNameFRFromAssetTranslations`, `tactical_repo.go:155-163`) et l'historique résolvent le même
  libellé FR ; sinon `[!]`, rien d'inventé. La vignette devient un groupe (bouton de sélection +
  icône-lien), jamais un lien dans un bouton.
- **D17 — Lien Vue match → Tactique** : `/ascension/tactique` avec `search={{ carte }}`, posé dans
  le bandeau d'« Occupation du terrain » (`features/match-view/MatchPositionsHeatmap.tsx`), après
  vérification que `MatchViewHeader.MapID` (`meta.MapAssetID`,
  `match_view_builders_header.go:114-116`) est bien `match_registry.map_id`. Après fusion de
  `feat/matchview-emprise` ; sinon insertion minimale dans ce seul fichier.
- **D18 — Opacité** : les deux constantes (`lib/replay/heatPaint.ts:35-36`) et les deux assertions
  (`heatPaint.test.ts:282-283`), rien d'autre. Elles changent aussi la carte de chaleur du rejeu 2D,
  « Occupation du terrain » de la Vue match (`MatchPositionsHeatmap.tsx:49,134`) et les vignettes :
  voulu (noyau partagé), dit au compte rendu.
- **D19 — Grappes de réapparition** : comportement existant (servies par les lectures d'artefact et
  par le filtre de grappe, `tactical_service_lectures.go:64-157`) ; la pilule « Réapparition » offre
  « Toutes » puis les grappes nommées de la réponse affichée.
- **D20 — Plus de pied de carte** ; les cellules hors du cadre du fond (aujourd'hui
  `footer_off_frame`) passent dans l'ⓘ quand il y en a (« dit, jamais avalé »).
- **D21 — `KPIStrip` supprimé** : son SEUL lecteur de production est
  `TacticalAnalysisView.tsx:29` (grep §4.B) ; règle n° 7. *Le brief dit « KPIStrip sur cette
  page ».* Les commentaires qui le citent (`components/ui/metric-trend.tsx:10,19`) sont corrigés.
- **D22 — `EvenementsJournal` / `EvenementsLocalises` retirés avec l'échange** : leur seul
  producteur est `lireLeJournal` (`tactical_service.go:248, 348-360`), qui ne lit le journal que
  pour eux et pour l'échange ; zéro lecteur web (grep §4.E). `PointsIgnores` (sans lecteur web
  non plus, mais produit par le rasterisage et journalisé) n'est pas touché (§8).
- **D23 — Instant de la tuile = `formatClock`** (`lib/replay/replayLogic.ts:438`, « m:ss »,
  l'horloge du rejeu qu'ouvre le bouton, déjà employée par la liste actuelle) — *la maquette écrit
  « 05:07 »* ; aucun troisième format d'horloge.
- **D24 — États vides : un titre, aucun conseil** (« Aucun match sur cette carte dans le filtre »,
  « Pas assez de matchs mesurés sur cette carte », « Densité insuffisante pour dessiner un plan ») ;
  les nombres qu'ils portaient sont dans l'ⓘ.
- **D25 — Fichiers web** : `TacticalPage.tsx` recomposé ; NEUFS `TacticalMapsColumn.tsx`,
  `TacticalZoneCard.tsx`, `TacticalRejeuTile.tsx`, `cockpit.logic.ts`, `plan.logic.ts`,
  `zone.logic.ts` (+ tests) ; `TacticalPlanCard.tsx` réécrit ; `tacticalView.logic.ts` (476 L) ne
  grossit pas.
- **D26 — Témoins de zone sur le catalogue RÉEL** (patron
  `games/halo_infinite/film/replay/callouts_catalog_test.go:108`) : Illusion (−13, 5), 11 morts à z
  médian 2,90 m → « Nid blindé » par (c) à 0,81 m ; Bazaar (−7, −1), 9 morts à z médian 3,18 m →
  trois polygones contiennent le centre, la règle (b) tranche : le nom rendu par la règle DU BRIEF
  est fixé au lot et consigné au journal (*la maquette, règle différente — la plus fréquente par
  événement, sans marge —, disait « Grande cour ouest »*). Les z des événements viennent des
  données de la maquette (`VM.witness`), copiés dans le test avec leur provenance.
- **D27 — Géométrie** : point dans polygone et distance au bord écrits UNE fois en production
  (`analysis/tactical/geometrie.go`) ; les deux copies de test (`hinavmesh/oracle_ancres_test.go:198`,
  `mapdecoupe/oracle_corpus_test.go:338`) sont des oracles indépendants, laissés tels quels.
- **D28 — Attribution des commits** : ligne système de la session
  (`Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`, celle de l'historique) ; le brief en
  citait une autre (signalé au compte rendu).

## 2. Spécification de rendu (non négociable)

La maquette fait foi, puis ce §2.

- **S1** Barre de filtres, en-tête, conseils et onglets intacts ; sous la barre, la grille cockpit
  (écart 12 px) ; la page défile.
- **S2** « Cartes jouées » : titre, champ de recherche en tête (placeholder « Carte », nom
  accessible « Rechercher une carte »), filtre au fil de la frappe, insensible à la casse et aux
  accents, sur le nom affiché ET le nom canonique ; liste verticale à défilement interne dans une
  colonne de hauteur fixe ; « Aucune carte ouvrable ne correspond » quand la recherche vide la liste.
- **S3** Vignette : 100 px 16:9 (fond + mini-plan « Morts » existant), nom (ellipse), « 54 · 30 V /
  24 D », barre fine victoires / défaites (`outcome-win` / `outcome-loss`) ; active = bordure 2 px
  `primary`, `aria-pressed` ; nom accessible « Sélectionner <carte> ».
- **S4** Repli `<details>` « N cartes sous le plancher » (avec recherche : « x sur N cartes sous le
  plancher », ouvert si seules elles correspondent) ; lignes texte « nom · 7 sur 10 ».
- **S5** Carte du plan : bandeau = titre (nom de la carte) + ⓘ, à droite trois réglages en pilules
  (étiquette atténuée + valeur) : « Lecture » (liste, ordre V5), « Joueurs » (Moi / Escouade /
  Adversaires, `aria-pressed`, Escouade désactivé avec infobulle sans composition), « Réapparition »
  (« Toutes » + grappes).
- **S6** ⓘ du plan : « {retenus} matchs mesurés sur {filtres} · {source} · grille {pas} m · 3 matchs
  distincts par zone » ; `gagne` + « {v} victoires et {d} défaites mesurées, 3 matchs distincts de
  chaque côté » ; `isole` + la règle « seul : aucun coéquipier visible, ou le plus proche au-delà de
  {portée} (portée du radar) », les matchs sans portée et les morts écartées ; + les cellules hors
  cadre s'il y en a (D20) ; source « journal des morts » ou « artefacts de rejeu ».
- **S7** Bandeau d'état au-dessus du fond, lectures d'artefact seulement : « N matchs en attente de
  traitement », « N matchs sans film » (`warning`).
- **S8** Corps : fond + calque dans une boîte de 800 px de haut au plus, rapport du fond respecté,
  centrée, marge droite réservée à la légende ; cadre de la cellule choisie ; étiquette du nom de
  zone posée à côté de la cellule, du côté où elle tient (maquette `zoneLabel` l. 992-997),
  « Zone sans nom » à défaut ; états vides (D24) et « Mise à jour… » posés SUR le fond (le fond ne
  se démonte jamais, acquis du lot L2 des retours rejeu).
- **S9** Rampe verticale au bord droit, 220 px, centrée verticalement, indépendante du fond :
  borne haute + unité en haut, borne basse en bas ; divergente (positif en haut) pour `gagne` et
  `solde`.
- **S10** Zone sélectionnée : titre = nom en jeu (ellipse), coordonnées en mention atténuée
  « x −14…−12 m · y 4…6 m » ; valeur en grand (signe « + » / « − » sur une lecture signée) +
  unité ; sous-ligne « N matchs distincts » (`gagne` : « v victoires, d défaites · N matchs
  distincts » ; `solde` : « f frags, m morts · N matchs distincts ») ; intertitre « Rejeu » ;
  liste à défilement interne ; sans sélection : titre « Zone sélectionnée », une ligne « Aucune zone
  sélectionnée ».
- **S11** Mini-tuile (maquette l. 1001-1015 et CSS l. 265-282) : bande d'issue 3 px ; ligne 1 mode
  (ellipse de secours), score, issue en mot (couleur d'issue), date · heure à droite ; ligne 2
  pastille mono de l'instant, fait, arme (seule à se tronquer), badge à droite ; bouton de rejeu
  36 px à droite, nom accessible « Ouvrir le rejeu à m:ss » ; infobulle = texte complet.
- **S12** Couleurs : jetons seulement (`outcome-*`, `warning`, `primary`, rampes
  `heatmapRampTokens`), aucune classe Tailwind de couleur ni hex (skill `color-tokens`).
- **S13** FR + EN pour toute chaîne (manifeste `tactical.toml`), aucun anglicisme (garde
  `lib/i18n/no-anglicisms.guard.test.ts`), aucun emoji, aucun impératif adressé au joueur.
- **S14** Halo 5 / capability absente : l'onglet n'apparaît pas (`FeatureGate` / `RouteCapabilityGate`
  `replay`, inchangés) ; sur Halo Infinite sans source de dégât, sans catalogue ou sans artefact,
  les champs manquants sont absents, jamais remplacés par un texte de repli.
- **S15** Lien de la tuile : `<Link>` du routeur (jamais `<a href>`), `?t=<instant_ms>&clock=<clock>`.
- **S16** Accessibilité : rampe `role="img"` nommée « Échelle de la lecture, de {lo} à {hi} » ;
  liste des cartes et liste « Rejeu » navigables au clavier ; focus visible.

## 3. Cibles

### 3.1 Contrat (Go, `internal/domain/`) — forme ; noms définitifs fixés au lot, tags snake_case

```go
const TacticalQuestionSolde = "solde"                      // tactical.go

type CelluleTactique struct { /* existant */
    Frags int `json:"frags,omitempty"` // solde seulement
    Morts int `json:"morts,omitempty"`
}
type TacticalRaster struct { /* existant */
    RayonsRadarM []float64 `json:"rayons_radar_m,omitempty"` // lecture isole (D9)
    // L9 retire : Echange, Coordination, Isolement, EvenementsJournal, EvenementsLocalises
}
type TacticalContribution struct { /* existant : MatchID, InstantMs, Clock, XUID, Resultat, MatchStartedAt */
    Face            string             `json:"face,omitempty"`            // mort|frag|entree|reapparition
    AutreGamertag   string             `json:"autre_gamertag,omitempty"`  // tueur (mort) / victime (frag)
    ArmeLabel       string             `json:"arme_label,omitempty"`
    ArmeLabelEN     string             `json:"arme_label_en,omitempty"`
    CategorieSource string             `json:"categorie_source,omitempty"`
    Placement       *TacticalPlacement `json:"placement,omitempty"`       // mort seulement
    ModeLabel       string             `json:"mode_label,omitempty"`      // langue de la requête
    ScoreLabel      string             `json:"score_label,omitempty"`     // « X - Y », mon camp d'abord
    ScoreKind       string             `json:"score_kind,omitempty"`      // points | rounds
    ReplayAvailable bool               `json:"replay_available"`
}
type TacticalPlacement struct {
    Seul      bool     `json:"seul"`
    DistanceM *float64 `json:"distance_m,omitempty"` // nil = aucun coéquipier visible
}
type TacticalZoneNom struct{ NomFR, NomEN string }      // tactical_zone.go (NEUF)
type TacticalCelluleReponse struct { /* existant */
    Zone *TacticalZoneNom `json:"zone,omitempty"` // nil = zone sans nom
}
type ZoneNommee struct { /* existant : NomFR, NomEN, X, Y */
    Polygone, Parties, Trous …; ZBas, ZHaut float64; VolumeIndex int // non sérialisés
}
type ContexteDeMort struct { MatchID, VictimXUID string; TimeMs int64; PlusProcheM *float64; Visibles, HorsDeVue int }
// TacticalKillPosition : + KillerZ, VictimZ *float64, KillerGamertag, VictimGamertag, SourceTag *uint32, SourceCategory string
// MortContexte         : + Z *float64, KillerGamertag, SourceTag *uint32, SourceCategory string
```

`domain/tactical.go` est à 468 L : les types neufs vont dans `domain/tactical_zone.go`.

### 3.2 Composants web (ordre à l'écran)

| Colonne | Bloc | Source (contrat) | Composant |
|---|---|---|---|
| — | barre de filtres | inchangée | `TacticalFilterBar` |
| gauche | « Cartes jouées » | `tactical/maps` | `TacticalMapsColumn` + `TacticalMapTile` (compacte) |
| centre | carte du plan | `tactical/{map}/raster` | `TacticalPlanCard` (réécrit) + `TacticalPlanFond` |
| droite | « Zone sélectionnée » | `raster.cellules` + `tactical/{map}/cellule` | `TacticalZoneCard` + `TacticalRejeuTile` |

Aucune route neuve ; aucune clé de requête neuve (`lib/query/keys.ts` non touché : les quatre
lectures existantes suffisent) ; `?carte=` reste dans `tacticalScope.ts`.

## 4. Inventaire des suppressions — preuves par grep (relevées le 2026-10-06, à REJOUER avant de supprimer)

Chaque preuve se rejoue par `Grep` sur `apps/web/src` (hors `lib/api/generated.ts`) ou
`apps/go-api`. Attendu APRÈS suppression : 0 occurrence hors fichiers supprimés et commentaires
historiques au passé.

- **A. Bascule et grille (web, L4).** `TacticalScreenSwitch` (`TacticalPage.tsx:145-151, 320-356`),
  `ContenuGrille` et la carte « Cartes jouées » en grille (l. 186-308), pied `tactical-couverture`
  (l. 200-209), `couvertureGrille` (`tacticalLogic.ts:78-86`, seul lecteur `TacticalPage.tsx:109`).
- **B. Barre d'outils, H2, KPI, coordination, pied (web, L5).** `TacticalToolbar.tsx` (seul lecteur
  `TacticalAnalysisView.tsx:42`) ; H2 et `pageTitle` (`TacticalAnalysisView.tsx:136-141`,
  `tacticalView.logic.ts:35-38`) ; `buildKpiCards` (l. 350-431) ; `components/layout/KPIStrip.tsx`
  + `KPIStrip.test.tsx` (seul lecteur `TacticalAnalysisView.tsx:29`) ; `TacticalCoordinationCard.tsx`
  + `TacticalCoordinationCard.logic.test.ts` (seul lecteur `TacticalAnalysisView.tsx:39`) ;
  `libelleRayons`, `DISTANCE_DECIMALES`, `formatDistanceM`, `positionCategorie`
  (`tacticalView.logic.ts:214-271`) si plus aucun lecteur ; `ratioSafe` (l. 130-133) si plus lu ;
  `PiedDuPlan`, `LegendeDuPlan` horizontale, `rampeCss` horizontale (`TacticalPlanCard.tsx:192-216,
  352-403`) ; `PLAN_HAUTEUR_MAX_PX` (D13) ; `HistogramChart` et `withLowSampleNote` RESTENT (autres
  lecteurs : `SquadRiposteCard.tsx`, `ChartsShowcasePage.tsx`, `coordinationModel.ts`).
- **C. Carte « Cellule sélectionnée » (web, L6).** `TacticalCellCard.tsx` + `TacticalCellCard.test.tsx`
  (seul lecteur `TacticalAnalysisView.tsx:38`), `questionSansCellule` (`tacticalView.logic.ts:164-166`).
- **D. Chaînes (web, L7).** Clés de `tactical.toml` et accesseurs d'`i18n.ts` sans lecteur après
  L4-L6, relevé par grep de chaque accesseur `t.<nom>` et de chaque clé : au minimum
  `tactical.maps.intro`, `tactical.maps.label`, `tactical.screen.*`, `tactical.kpi.*`,
  `tactical.coordination.*` (dont `radius_value` / `radius_join` si l'ⓘ ne les relit pas),
  `tactical.plan.title`, `tactical.plan.scale_*`, `tactical.plan.footer_*`, `tactical.cell.*`
  remplacées, `tactical.toolbar.question_label`, `tactical.analysis.page_title`,
  `tactical.analysis.questions.*` (remplacées par les lectures V5) ; manifeste régénéré.
- **E. Échange (Go, L9).** `TacticalRaster.Echange` (`domain/tactical_page.go:213-217`),
  `mesurerEchange` (`tactical_service.go:380-410`), `lireLeJournal` / `compterJournal`
  (l. 336-378) et leurs deux appels (`tactical_service.go:256`, `tactical_service_isolement.go:93`),
  `EvenementsJournal` / `EvenementsLocalises` (`domain/tactical_page.go:130-145`,
  `tactical_service.go:248, 264-266`) ; cas de test dans `tactical_service_echange_test.go`
  (les cas sans échange — portes de capability, `MapsPlayed`, sans lecteur, l. 113-272 — sont
  DÉPLACÉS tels quels, noms inchangés, vers `tactical_service_portes_test.go`),
  `api/handlers/tactical_test.go:193-207`. `coordination.Echanges` / `Mesurer` RESTENT (Escouade,
  Vue match `match_view_builders_riposte.go:66`).
- **F. Coordination (Go, L9).** `domain/tactical_coordination.go` entier (`TacticalBinDistance`,
  `TacticalCoordination`, `TacticalBornesDistanceM`, `TaCoordDistances`), `TacticalRaster.Coordination`
  (`tactical_page.go:208-211`), `construireCoordination` et `mesurerCoordination`
  (`tactical_service_isolement.go:169-188, 205-244`) et leurs appels (`tactical_service.go:179, 195`,
  `tactical_service_isolement.go:68`), `analysis/coordination/distances.go` + `distances_test.go`
  (seul lecteur de production `construireCoordination`), entrée `domain.TaCoordDistances` de la liste
  blanche `analysis/coordination/no_naked_rate_test.go:58-63, 97` (une liste blanche qui BAISSE),
  `service/tactical_service_coordination_test.go`. `coordination.FenetreEchangeMs` RESTE (Escouade,
  Vue match, bloc de coordination).
- **G. Isolement du raster (Go, L9).** `TacticalRaster.Isolement` (`tactical_page.go:199-206`) et ses
  deux écritures (`tactical_service_isolement.go:76-77, 239-240`) ; le bilan de `coordination.Isolement`
  reste calculé par `rasterIsole` (cellules isolées, `MatchsSansRayon`, `MortsEquipeATerre`).
- **H. Contrat (L9).** Schémas `TacticalCoordination`, `TacticalBinDistance` ; champs `echange`,
  `coordination`, `isolement`, `evenements_journal`, `evenements_localises` de `TacticalRaster` ;
  alias web `TacticalCouverture`, `TacticalCoordination`, `TacticalBinDistance`
  (`lib/api/types.ts:3417, 3430-3431`) ; snapshot `lib/api/contract-surface.snapshot.json`
  régénéré par la procédure (`UPDATE_CONTRACT_SURFACE=1`), disparitions listées au journal.
- **Baseline de tests** : avant toute suppression de test Go, `Grep` du nom dans
  `.ai/baselines/tests_pre_migration.jsonl` (relevé de juin, antérieur à l'onglet : attendu 0).

## 5. Organisation et gates communs

- Exécuteur seul, dans le worktree, lots SÉQUENTIELS. Aucun sous-agent, aucun push, aucun merge,
  aucun `git stash`, aucun `git add -A` (stager fichier par fichier), aucun `--no-verify`, aucun
  Python, aucun navigateur, aucune base de `data/` ouverte (tests sur `:memory:` ou fixtures ; le
  catalogue `data/titles/halo_infinite/reference/map_callouts.json`, fichier de référence versionné,
  se lit en test comme le fait déjà `callouts_catalog_test.go`), aucun serveur arrêté ou relancé,
  une commande `go` à la fois, aucune cuisson ni backfill.
- Environnement Go, à chaque appel PowerShell :
  `$env:Path = "C:\msys64\ucrt64\bin;$env:Path"; $env:CGO_ENABLED = "1"; $env:CC = "C:\msys64\ucrt64\bin\gcc.exe"`.
  Web : `npm ci` dans `apps/web` du worktree (node_modules RÉEL) avant le premier gate web.
- **Gate Go** (depuis `apps/go-api`) : `go build ./...` ; `go vet` des paquets touchés ;
  `gofmt -l internal cmd` muet ; `go test -count=1` des paquets touchés puis du module en lots
  couvrant tout `go list ./...` ; `go test -tags=integration -p 1 ./internal/platform/duckdb/...` dès
  que `platform/duckdb` bouge (L2) ; `go test ./internal/archlint/...` ; `make go-api-lint` (Git
  Bash ; en cas de verrou d'un autre worktree : `golangci-lint run --new-from-merge-base=origin/main`
  avec `GOLANGCI_LINT_CACHE` isolé et `--allow-parallel-runners`, règles inchangées) ; contrat :
  `go run ./cmd/openapi-gen`, `go run ./cmd/openapi-gen -check`, `npm run generate-types` (dans
  `apps/web`), `node tools/check-generated-types-fresh.mjs` (racine). Codes de sortie vérifiés,
  filtre d'échec ancré (`^--- FAIL:`).
- **Gate web** (depuis `apps/web`, vitest hors sandbox) : purge `node_modules\.tmp` ;
  `npx tsc -b --force` ; `npm run lint` (0 erreur) ; `npx vitest run --pool=forks` ;
  `node scripts/build_i18n_manifests.mjs` puis `git diff --exit-code` sur `src/lib/i18n/generated/`
  (manifeste à jour, parité FR / EN vérifiée par le build) ; depuis la racine :
  `node tools/knip-ratchet.mjs` (0 / 0 / 0 — knip est aveugle sur ce poste, §8 du plan modèle : les
  preuves grep §4 restent les juges), `node tools/lint-no-hardcoded-colors.mjs` (0),
  `node tools/lint-no-hardcoded-fields.mjs` (0), `node tools/lint-cross-feature-imports.mjs`
  (≤ 7, aucune dérogation morte), `npx lefthook run pre-push` (PATH avec `C:\msys64\ucrt64\bin` et
  `C:\Program Files (x86)\GnuWin32\bin`). Les tests e2e Playwright (`e2e/visual/readme-shots`
  ouvre `ascension/tactique`) ne tournent qu'en PR vers `main` : non lancés ici.
- **Seuils** (CLAUDE.md n° 5) : fichier ≤ 500 L, fonction ≤ 80 L, ≤ 5 paramètres, complexité ≤ 12
  pour tout fichier créé ou modifié ; mesure jointe au journal du lot. Attention :
  `tactical_service.go` 465 L, `domain/tactical.go` 468 L, `tacticalView.logic.ts` 476 L,
  `TacticalAnalysisView.tsx` 432 L, `api/wire/registry_pages.go` 619 L (ne pas l'agrandir : D15),
  `lib/api/types.ts` (dette existante).
- **TDD** : chaque règle neuve a son test ROUGE écrit et vu rouge AVANT le code ; puis vert ; puis au
  moins UNE MUTATION par règle (modification volontaire qui doit faire rougir, annulée ensuite,
  restauration vérifiée par `git diff`), consignée au journal du lot (« mutation : … → rouge »).
- **Clôture de lot** = gate vert + items statués + section du lot mise à jour ici + entrée en FIN
  de `.ai/thought_log.md` + commit local `feat(tactique-v2/<lot>): …` (fichiers stagés un par un,
  message en français, ligne D28) + compte rendu au superviseur, puis attente du « continue ».
- **Lots voisins** (ne pas toucher leurs fichiers ; dépendances) :
  - `feat/matchview-emprise` (`features/match-view/`, `service/match_view*`) : L10 touche
    `MatchPositionsHeatmap.tsx` (après sa fusion, sinon insertion minimale) ; L1 renomme
    `aPortee` dans `analysis/coordination/vies_pres_ou_seul.go`, que son plan ne modifie pas (« aucun
    type ni fonction neuve dans `coordination` ») mais appelle (`ViesPresOuSeul`) ; D18 change le
    rendu de sa carte « Occupation du terrain », qu'il déclare inchangée.
  - `feat/sessions-emprise` (`features/session-detail/`, `service/session_page*`, `domain/session_*`) :
    aucun fichier commun ; `CoordinationBlock` et `coordination.Bloc` intacts.
  - `feat/rejeu-equipes-web` (`features/match-replay/`, modèle web du rejeu) : seul
    `lib/replay/heatPaint.ts` est commun — deux constantes et leur test, rien d'autre (D18).
  - Docs communes à la clôture (`CHANGELOG`, `RELEASE_NOTES`, `.ai/thought_log.md`) : conflits
    attendus à la fusion, résolus en gardant les deux côtés.

## 6. Lots

### L1 — Go : briques pures et contrat additif · moyen

Périmètre : `analysis/coordination/{vies_pres_ou_seul.go, isolation.go}`, une ligne de
`service/tactical_service_cellule.go`, `archlint/` (garde-rail neuf), `analysis/tactical/{solde.go,
zones.go, geometrie.go, placement.go, merge.go, spawn.go}` (+ tests), `domain/{tactical.go,
tactical_cellule.go, tactical_page.go, tactical_raster.go, tactical_zone.go}`, contrat.

- [x] L1.1 `coordination.APortee(d *float64, rayon float64) bool` (export de `aPortee`,
  `vies_pres_ou_seul.go:141-145`) ; appelants `isolation.go:72-74`, `vies_pres_ou_seul.go:83` ;
  `celluleIsole` (`tactical_service_cellule.go:225`) l'appelle. Garde-rail
  `archlint/no_local_portee_comparison_test.go` : empreinte « distance déréférencée comparée par
  `<=` à un rayon / une portée » et « `PlusProcheM != nil &&` », refusée dans tout fichier de
  production hors de `analysis/coordination/vies_pres_ou_seul.go` ; auto-test qui prouve que
  l'empreinte reconnaît l'ancienne copie de `celluleIsole` (littéral) et ignore l'appel au helper ;
  mutation : réintroduire la comparaison inline dans `tactical_service_cellule.go` → rouge.
  `TestAucunTauxNu` vert (retour `bool`, liste blanche).
- [x] L1.2 `analysis/tactical/solde.go` (D2) ; `domain.CelluleTactique.Frags`, `.Morts`. Tests
  ROUGES d'abord (`solde_test.go`) : plancher sur l'union (deux matchs de frags + un match de morts =
  3 matchs → cellule lue ; deux matchs → retirée), match muet compté au dénominateur, valeur
  (f − d) / N, signe, comptes par face, `Somme` de deux rasters de solde. Mutations : plancher par
  face au lieu de l'union, dénominateur = matchs de la cellule, `Somme` qui perd les faces → rouges.
- [x] L1.3 `analysis/tactical/zones.go` + `geometrie.go` (D3) ; jumeau `tactical.ZoneNommee` étendu.
  Tests ROUGES d'abord (`zones_test.go`, géométrie synthétique) : (a) un candidat, marge 0,25
  (z = z_haut + 0,24 compatible, + 0,26 non), (b) majorité puis plus étroite, (b) sans majorité,
  égalité par `VolumeIndex`, (c) tranche compatible d'abord puis toutes, distance au bord 1,99 m
  retenue / 2,00 m non, polygone contenant à distance 0 au second passage, (d) sans nom, z inconnu
  (un / zéro / plusieurs polygones), trou exclu, partie incluse. Mutations : marge 0, `<=` 2 m,
  majorité « ≥ moitié », plus large au lieu de plus étroite, trous ignorés → rouges.
- [x] L1.4 `analysis/tactical/placement.go` (D7) : `ContexteLePlusProche(contextes, victime, t)`
  (± `TolerancePlacementMs` = 1 500, bornes comprises, le plus proche, même victime) et
  `PlacementDeLaMort(ctx, rayon, aUnRayon) *domain.TacticalPlacement`. Tests ROUGES d'abord : borne
  1 500 / 1 501, autre victime ignorée, deux candidats, nil sans contexte, « seul » sans distance,
  d = portée → près, d > portée → seul, portée inconnue + distance → nil. Mutations : tolérance
  stricte, `<` au lieu de `APortee` → rouges.
- [x] L1.5 Contrat additif (§3.1) : `TacticalQuestionSolde`, `TacticalContribution` enrichie,
  `TacticalPlacement`, `TacticalZoneNom`, `TacticalCelluleReponse.Zone`,
  `TacticalRaster.RayonsRadarM`, `ZoneNommee` étendue, `ContexteDeMort`, champs de
  `TacticalKillPosition` / `MortContexte` ; commentaires de contrat au présent (règle 17).
- [x] L1.6 Contrat régénéré (openapi + `generated.ts`), diff additif ; garde
  `contract-surface.guard.test.ts` verte sans régénérer le snapshot.
- Gate : gate Go + contrat ; `TestTacticalEtCoordinationSontPurs`, `TestAucunTauxNu`,
  `TestNoLocalRadarRangeLookup` rejoués nommément ; `npx vitest run src/lib/api` vert.

Journal L1 (2026-10-06, exécuteur, `feat/tactique-v2`) — TDD : chaque test vu rouge avant le code :
- **L1.1** `coordination.APortee` exportée (`vies_pres_ou_seul.go`, doc au présent, garde-rail nommé) ;
  `isolation.go` et `celluleIsole` (`tactical_service_cellule.go`, commentaire corrigé : le PARCOURS
  y est réécrit, la COMPARAISON est le helper) l'appellent. Garde-rail
  `archlint/no_local_portee_comparison_test.go` (deux empreintes : distance déréférencée comparée par
  `<=` à une portée, `PlusProcheM != nil &&`) écrit d'abord et vu ROUGE sur la copie de `celluleIsole`
  (deux violations), auto-test (trois copies reconnues, quatre faux positifs écartés : `himap`,
  `replayverite`, appel du helper) vert d'emblée.
- **L1.2** `analysis/tactical/solde.go` : `RasteriseSolde` (union des faces dans `cellules`, comptes par
  face dans le champ `faces` du `Raster`), `CellulesSolde`, `sommerFaces` appelé par `Somme` ;
  `domain.CelluleTactique.Frags` / `.Morts`. `solde_test.go` (5 tests) vu rouge (symboles absents).
- **L1.3** `analysis/tactical/zones.go` (`NommerZone`, `MargeTrancheZM`, `RayonZoneM`, règles
  `polygone` / `empilee` / `proche`, z inconnu) + `geometrie.go` (forme pair-impair, distance au
  bord) ; jumeau `tactical.ZoneNommee` étendu (`Polygone`, `Parties`, `Trous`, `ZBas`, `ZHaut`,
  `VolumeIndex`). `zones_test.go` (9 tests) vu rouge ; cas « exactement la moitié n'est pas la
  majorité » ajouté pour que la mutation `>=` rougisse.
- **L1.4** `analysis/tactical/placement.go` : `TolerancePlacementMs` = 1 500, `ContexteLePlusProche`
  (même match, même victime, le plus proche, à égalité le plus ancien), `PlacementDeLaMort` (par
  `coordination.APortee`). `placement_test.go` (3 tests) vu rouge.
- **L1.5** Contrat additif : `TacticalQuestionSolde` ; `TacticalContribution` (`Face`,
  `AutreGamertag`, `ArmeLabel` / `ArmeLabelEN`, `CategorieSource`, `Placement`, `ModeLabel`,
  `ScoreLabel` / `ScoreKind`, `ReplayAvailable`) + constantes `TacticalFace*` ;
  `TacticalCelluleReponse.Zone` ; `TacticalRaster.RayonsRadarM` ; `domain.ZoneNommee` étendue ;
  `domain/tactical_zone.go` (NEUF : `TacticalZoneNom`, `TacticalPlacement`, `ContexteDeMort`) ;
  `TacticalKillPosition` (`KillerZ`, `VictimZ`, gamertags, `SourceTag`, `SourceCategory`) et
  `MortContexte` (`Z`, `KillerGamertag`, `SourceTag`, `SourceCategory`).
- **L1.6** Contrat : `openapi.yaml` +58 / −0, `generated.ts` +25 / −0 (premier `npm ci` du
  worktree fait ici : `generate-types` en dépend) ; `check-generated-types-fresh` OK ;
  `vitest src/lib/api` 5 fichiers / 36 tests verts, snapshot de surface intact.
- **Mutations** (script `mutation.ps1` du scratchpad, restauration vérifiée par empreinte SHA-256,
  toutes ROUGES) : plancher par face au lieu de l'union ; dénominateur = matchs de la cellule ;
  `Somme` qui perd les faces ; comparaison inline réintroduite dans `celluleIsole` (garde-rail) ;
  borne stricte dans `APortee` (rougit l'isolement ET les vies) ; marge de tranche 0 ; rayon de 2 m
  inclusif ; majorité « ≥ moitié » ; plus large au lieu de plus étroite ; trous ignorés ; nom posé
  sur des zones empilées sans z ; second passage seul en (c) (tranches compatibles ignorées) ;
  tolérance de placement stricte ; borne de portée stricte au badge ; badge sans portée connue.
- **Gate** : `go build ./...` 0 ; `go vet` des cinq paquets touchés 0 ; `gofmt -l internal cmd`
  muet ; `go test -count=1 ./...` (349 paquets : 195 ok, 153 sans test, 1 FAIL) — le seul échec,
  `TestLUSRV2Shadow_RafalesBornees_300Candidats` (`internal/sync/skill`, test de durée de rafale :
  2,018 s pour un plafond de 2 s), est hors périmètre et la machine portait les `go test` d'une autre
  session ; rejoué seul : ok ; `golangci-lint run --new-from-merge-base=origin/main`
  (cache isolé, `--allow-parallel-runners`) 0 issues ; `openapi-gen -check` à jour ; garde-rails
  rejoués nommément en `-v` : `TestNoLocalPorteeComparison` (+ auto-test),
  `TestNoLocalRadarRangeLookup` (+ auto-test), `TestTacticalEtCoordinationSontPurs`,
  `TestAucunTauxNu` PASS. `-tags=integration` non requis (aucun paquet `platform/duckdb`, `sync`,
  `persist`, `migration` modifié).
- Seuils : `domain/tactical.go` 468 → 492 L (sous 500 ; L9 en retirera) ; fichiers neufs ≤ 185 L ;
  plus longue fonction neuve `NommerZone` (~30 L) ; aucun paramètre au-delà de 4.
- Écarts : le lint a été lancé d'emblée avec le cache isolé (règles et commande de `make go-api-lint`
  inchangées) ; les types neufs de L1.5 utilisés par L1.4 ont été posés avec L1.4.

### L2 — Go : lectures enrichies, contextes de mort, zones polygonales, lecture « solde » · lourd

Périmètre : `platform/duckdb/{tactical_repo.go, tactical_repo_isolement.go,
tactical_repo_contextes.go (NEUF)}` (+ tests), `port/tactical.go`, `service/{tactical_callouts.go,
tactical_service.go, tactical_service_grille.go, tactical_service_perimetre.go,
tactical_service_isolement.go}` (+ tests), `api/handlers/tactical.go` (docs), contrat,
`docs/adr/0036-page-reads-are-scoped.md` (liste I2).

- [x] L2.1 `QTacticalPositions` + scan (D4, D6 : `killer_z`, `victim_z`, `feed_killer_gamertag`,
  `victim_gamertag`, `source_tag`, `source_category`) ; `QTacticalIsolement` + `scanMortContexte`
  (`victim_z`, `feed_killer_gamertag`, `source_tag`, `source_category`). Tests `:memory:` écrits
  d'abord (patron `tactical_repo_test.go`) : colonnes lues, z NULL conservé (pointeur nil, jamais 0),
  tag / catégorie NULL → absents ; `TestTacticalRepo_PerimetreRestreint_FenetresBornees` vert sans
  modification. Mutation : z NULL lu comme 0 → rouge.
- [x] L2.2 `TacticalRepository.ContextesDeMort(ctx, q)` (`tactical_repo_contextes.go`, D7) : liste
  blanche liée en constante sur le `match_id` de `match_death_context_latest`, aucune
  sous-requête ; table absente → `games.ErrCapabilityNotSupported` (patron
  `squad_life_placement_repo.go:73-78`) ; liste vide → aucune requête. Test `:memory:` écrit
  d'abord : dernière passe entière par match, matchs demandés seulement, NULL conservés,
  `exigerFenetresBornees` ; ADR 0036 : test ajouté à la liste I2 et au tableau ; double de test
  du port étendu (`service/tactical_mock_test.go`). Mutation : liste liée par sous-requête → rouge.
- [x] L2.3 `zonesNommees` projette contour, parties, trous, tranche, index de volume (D3) ;
  `service/tactical_callouts_test.go` (NEUF ou étendu) : projection sur fixture, puis témoins D26 sur
  le catalogue réel par `tactical.NommerZone` (Illusion → « Nid blindé », règle (c), 0,81 m ;
  Bazaar → règle (b), trois polygones, nom consigné). Si un témoin change de nom à cause des
  parties / trous : arrêt et compte rendu (la forme D3 se discute avant de forcer). Mutation :
  projection sans le contour → rouge.
- [x] L2.4 Lecture `solde` (D2) : `validerLecture`, `facesDeLaQuestion`, `cellulesLisibles`,
  `rasteriser` / `rasteriserSurGrille`, `remplirRaster` ; doc des corps Huma (`handlers/tactical.go:115,
  212` : « … | solde | … ») ; tests service (mocks de port) écrits d'abord : dispatch, deux faces,
  échelle symétrique, plancher sur l'union, `MatchsRetenus` = mesurés, `Frags` / `Morts` des
  cellules, `kills` / `gagne` inchangés ; test handler : `solde` accepté, question inconnue toujours
  400. Mutations : une seule face, échelle non symétrique → rouges.
- [x] L2.5 `rasterIsole` pose `RayonsRadarM` (D9) ; test : deux formats → deux portées triées,
  aucune moyenne. Mutation : portées non dédupliquées → rouge.
- [x] L2.6 Contrat régénéré (additif).
- Gate : gate Go + `go test -tags=integration -p 1 ./internal/platform/duckdb/...` + contrat ;
  `TestNoRawAppendOnlyReads`, `TestLecturesDeLaVueDesNoms_Ratchet`, `TestCampaignExclusionGuard`
  rejoués nommément.

Journal L2 (2026-10-06, exécuteur, `feat/tactique-v2`) — tests `:memory:` seulement, aucune base de `data/` ouverte :
- **L2.1** `QTacticalPositions` lit `killer_z`, `victim_z`, `feed_killer_gamertag`, `victim_gamertag`,
  `source_tag`, `source_category` ; `QTacticalIsolement` / `scanMortContexte` lisent `victim_z`,
  `feed_killer_gamertag`, `source_tag`, `source_category`. z NULL → pointeur nil par `nullFloatPtr`
  (existant, réutilisé) ; tag NULL → nil par `tagOuNil` (neuf, `tactical_repo.go`).
  `tactical_repo_zone_test.go` (2 tests) vu ROUGE avant le code ;
  `TestTacticalRepo_PerimetreRestreint_FenetresBornees` vert sans modification.
- **L2.2** `ContextesDeMort` (`tactical_repo_contextes.go`, 70 L) : liste blanche exigée (refus sinon),
  liste vide → aucune requête, liée en constantes sur `match_death_context_latest.match_id`, table
  absente → `ErrCapabilityNotSupported`. Tests `BorneEtNull` (fenêtres bornées), `ListeExigee`,
  `TableAbsente` vus rouges (méthode absente) ; `DernierePasseEntiere` ajouté APRÈS le code à la
  relecture de l'item (deux passes, morts différentes : seule la neuve sort), vert d'emblée, mutation
  « table brute » ROUGE. ADR 0036 : test `BorneEtNull` dans la liste I2 et au tableau. Doubles du port
  étendus : `service/tactical_mock_test.go` et `service/teammates/teammates_squad_echange_test.go`.
- **L2.3** `zonesNommees` (et `zonesPures`) projettent contour, parties, trous, tranche, index de
  volume. Témoins D26 sur le catalogue réel : Illusion → « Nid blindé » par (c) à 0,81 m (inchangé) ;
  Bazaar → trois polygones contiennent le centre, règle (b), « Pont du marché ouest » / « West Market
  Bridge » (la maquette disait « Grande cour ouest » sur une autre règle ; une sonde sans parties ni
  trous rend les mêmes noms : AUCUN nom ne change à cause des parties / trous, pas d'arrêt).
- **L2.4** `solde` servie : `validerLecture` l'accepte ; `rasteriserLaCible` (`tactical_service_grille.go`)
  garde les deux faces séparées (`projeterFaces` → `RasteriseSolde`, pas choisi sur `CellulesSolde`) ;
  `cellulesLisibles` et `remplirRaster` (échelle symétrique, aucun côté victoire / défaite) ;
  `facesDeLaQuestion` rend déjà les deux faces par sa branche par défaut (contrat écrit au présent,
  test `Solde_DeuxFacesAuJournal` vert d'emblée) ; `idsDeLUnivers` partagé. Docs Huma
  (`handlers/tactical.go:115, 212`). `tactical_service_solde_test.go` (3 tests) vu rouge (« question
  inconnue (solde) ») ; test handler `TestTacticalHandler_QuestionSolde` vert d'emblée (le handler
  transmet, `frags` / `morts` posés en L1) ; `QuestionInconnue400` inchangé et vert.
- **L2.5** `rasterIsole` pose `RayonsRadarM = rayonsDistincts(rayons)` ;
  `TestIsole_RayonsRadarDistinctsEtTries` (deux Arène + un BTB → [18 24], vide hors « isole ») vu rouge.
- **L2.6** Contrat : `openapi.yaml` 2 descriptions (« … | solde | … »), `generated.ts` idem ;
  `-check` à jour, `check-generated-types-fresh` OK, `vitest src/lib/api` 5 fichiers / 36 tests verts.
- **Mutations** (toutes ROUGES, restauration vérifiée par empreinte) : z NULL lu comme 0 (positions ;
  isolement) ; liste liée par sous-requête ; table brute au lieu de `_latest` ; projection sans le
  contour ; une seule face au solde ; échelle non symétrique au solde ; portées non dédupliquées.
- **Gate** (avant-plan) : `go build ./...` 0 ; `go vet` des cinq paquets touchés 0 ; `gofmt -l internal
  cmd` muet ; paquets touchés 0 ; module en six lots couvrant les 349 paquets de `go list ./...`
  (196 ok, 153 sans test, 0 FAIL) ; `go test -count=1 -tags=integration -p 1
  ./internal/platform/duckdb/...` 4 ok ; `go test ./internal/archlint/...` ok ; garde-rails nommés en
  `-v` : `TestNoRawAppendOnlyReads`, `TestLecturesDeLaVueDesNoms_Ratchet`, `TestCampaignExclusionGuard`
  PASS ; `golangci-lint run --new-from-merge-base=origin/main` (cache isolé) 0 issues ; contrat ci-dessus.
  Après l'ajout de `DernierePasseEntiere` : vet, tests et lint du paquet `platform/duckdb` rejoués, verts.
- Seuils : `service/teammates/teammates_squad_echange_test.go` 587 → 592 L (déjà au-delà de 500 avant
  le lot ; +5 = le double du port, imposé par l'extension de l'interface ; non découpé, hors
  périmètre) ; autres fichiers touchés ≤ 498 L ; fonctions neuves ≤ 20 L ; ≤ 4 paramètres.
- Écarts : `teammates_squad_echange_test.go` et `tactical_service_lectures.go` (`zonesPures`) hors de
  la liste du périmètre, touchés par nécessité (double du port ; même projection que `zonesNommees`) ;
  `tactical_service_isolement.go` touché pour L2.5 (au périmètre).
- Écart assumé à la maquette (décision du superviseur, 2026-10-06) : le nom retenu pour le témoin
  Bazaar (−7, −1) est « Pont du marché ouest » (règle (b) du brief), et non « Grande cour ouest » de la
  maquette (autre règle).

### L3 — Go : le détail de zone enrichi et son câblage · lourd

Périmètre : `service/{tactical_service_cellule.go, tactical_service_cellule_enrichir.go (NEUF),
tactical_service_cablage.go, tactical_service.go}` (+ tests, `tactical_mock_test.go`),
`api/wire/{registry_pages.go, registry_pages_tactical.go (NEUF), registry_pages_tactical_wiring_test.go
(NEUF)}`, `api/handlers/tactical_cellule_test.go`, contrat.

- [x] L3.1 Les trois sources du détail posent `Face`, `AutreGamertag`, la source brute (tag,
  catégorie) et le z interne de chaque contribution (`celluleDeKills`, `celluleIsole`,
  `contributionsDuSidecar`) ; `temps` → `entree`, `routes` → `reapparition` (D12).
- [x] L3.2 Nom de zone (V6, D3) : `s.zonesDeLaCarte` + centre de la cellule au pas demandé + z des
  événements de la cellule → `tactical.NommerZone` → `out.Zone` ; règle retenue au journal (Debug).
- [x] L3.3 `tactical_service_cellule_enrichir.go` (D5, D6, D7, D8) : une lecture canonique par
  requête (`LoadPlayerMatches`, filtre `MapIDs`), un `ResolveWeaponLabels` pour toutes les clés, un
  `ContextesDeMort` borné aux matchs des contributions, un `AvailableSet` ; mode (langue de la
  requête, `ctxkeys.Locale`), score (mon camp d'abord, camp lu sur `Self.TeamID`), arme /
  catégorie, placement (faces `mort`), `ReplayAvailable` ; sections de durée
  `tactical_cellule_canonique`, `tactical_cellule_armes`, `tactical_cellule_contextes`,
  `tactical_cellule_rejeu` (ADR 0036 I6) ; dégradations D15.
- [x] L3.4 Injecteurs `With*` (D15) et champs du service ; `tactical_service.go` ne grossit pas (les
  champs neufs vivent dans une struct déclarée dans `tactical_service_cellule_enrichir.go` et
  embarquée par une ligne).
- [x] L3.5 Câblage : fabrique `Tactical` déplacée dans `api/wire/registry_pages_tactical.go`
  (taille de `registry_pages.go` avant / après au journal), injecteurs inconditionnels sauf ce que
  les capabilities gouvernent déjà (classificateur nil sur Halo 5) ; garde-rail de câblage
  `registry_pages_tactical_wiring_test.go` (patron `registry_pages_timeseries_wiring_test.go`,
  `appelsDansFactory`) : chaque `With*` présent, inconditionnel. Mutation : un `With*` retiré →
  rouge.
- [x] L3.6 Tests service écrits d'abord (`tactical_service_cellule_enrichir_test.go`) : chaque champ ;
  score en manches sur une variante `rounds_decide` ; mon camp d'abord en camp 1 ; arme par la clé,
  catégorie à défaut, rien à défaut ; badge (seul sans distance, près, seul à distance, aucun sur un
  frag) ; `replay_available` vrai / faux ; « Zone sans nom » ; chaque source absente
  (`ErrCapabilityNotSupported`) ou en échec → champ absent, contributions servies, journal ;
  compteurs : une lecture par source et par requête ; Halo 5 (sans classificateur ni catalogue) ;
  ownership inchangé (`matchs_non_ouvrables`). Mutations : camp inversé, lecture canonique par
  contribution, badge sur un frag, `replay_available` toujours vrai → rouges.
- [x] L3.7 Test handler du détail : clés snake_case servies (`face`, `mode_label`, `score_label`,
  `placement`, `replay_available`, `zone`) ; contrat régénéré (additif).
- Gate : gate Go + contrat ; `TestNoNewSlugComparison`, `TestAucunTypeAnalysisEnCorpsHuma`
  rejoués nommément.

Journal L3 (2026-10-06, exécuteur, `feat/tactique-v2`) :
- **L3.1** Les trois sources rendent des `contributionLue` (la contribution publiée, plus la hauteur
  de l'événement et la source de dégât brute, non publiées) et l'univers de leur lecture : kills /
  gagne / solde → `faceDeKill` (mort : autre = tueur, z = victime ; frag : autre = victime, z =
  tueur) ; isole → face `mort`, autre = tueur ; temps → `entree`, routes → `reapparition`
  (`faceDArtefact`, sans z ni source). Le filtre d'ouvrabilité et le tri sortent de `Cellule` dans
  `garderLesOuvrables` (comportement inchangé). Tests : `tactical_service_cellule_faces_test.go`
  (faces et autre joueur) et trois assertions `Face` ajoutées aux tests isole / temps / routes, vus
  rouges.
- **L3.2** `nommerLaCellule` : centre de la cellule au pas demandé (`Grille.Centre`), hauteurs des
  contributions retenues, `tactical.NommerZone` sur `zonesPures(s.zonesDeLaCarte(...))` ; règle et
  distance au journal (Debug). Tests : la même cellule se nomme « Étage » pour une mort (z de la
  victime) et « Rez » pour un frag (z du tueur), vu rouge ; « Zone sans nom » → `zone` absente.
- **L3.3** `tactical_service_cellule_enrichir.go` (297 L) : `enrichir` ne lit rien sans contribution ;
  sinon `poserModeEtScore` (une `LoadPlayerMatches` filtrée sur la carte, `Validate` appelé ; mode
  par `labelPourLocale` et `ctxkeys.Locale` ; score par `scoreDuMatch` : mon camp d'abord sur
  `Self.TeamID`, `analysis.ReadTeamScore` + `FormatTeamScoreLabel`, table `rounds_decide`),
  `poserArmes` (classificateur par contribution, UN `ResolveWeaponLabels` pour les clés distinctes ;
  arme FR / EN, sinon catégorie brute ; sans classificateur rien), `poserPlacements` (UN
  `ContextesDeMort` borné aux matchs des morts retenues, `ContexteLePlusProche` +
  `PlacementDeLaMort`, portée par `rayonsParMatch` ; faces `mort` seulement), `poserRejeu` (UN
  `AvailableSet`). Sections `tactical_cellule_canonique`, `tactical_cellule_armes`,
  `tactical_cellule_contextes`, `tactical_cellule_rejeu`, feuilles (aucune des lectures n'en déclare).
  Dégradations : source nil ou `ErrCapabilityNotSupported` → DEBUG « tactique: detail de zone, source
  <nom> absente » ; autre erreur → WARN « … en echec » ; champ absent, liste servie.
- **L3.4** `sourcesDuDetail` (déclarée dans le fichier d'enrichissement) portée par UNE ligne de
  `TacticalService` (`detail sourcesDuDetail`) ; `tactical_service.go` 468 → 469 L (cette ligne) ;
  `WithPlayerMatches`, `WithRoundsDecide`, `WithKillSourceClassifier`, `WithWeaponLabels`,
  `WithReplay` dans `tactical_service_cablage.go`.
- **L3.5** Fabrique `Tactical` déplacée telle quelle dans `api/wire/registry_pages_tactical.go` (61 L) ;
  `registry_pages.go` 619 → 579 L. Cinq injecteurs ajoutés, tous inconditionnels (classificateur nil
  sur un titre sans `film.kill_source`, décidé par `killSourceClassifierFor`). Garde-rail
  `registry_pages_tactical_wiring_test.go` (neuf `With*`, argument exact, aucune porte ; factory
  absente de `registry_pages.go`) vu rouge avant le déplacement.
- **L3.6** `tactical_service_cellule_enrichir_test.go` (6 tests, 9 sous-cas de sources) vu rouge
  (compilation) avant le code : chaque champ sur quatre contributions (points et manches, camp 1
  d'abord, arme / catégorie / rien, quatre badges dont aucun sur un frag, rejeu vrai / faux), mode en
  anglais, une lecture par source, ownership (contextes lus pour m1 seul), titre sans classificateur
  ni catalogue, chaque source absente ou en échec (champ absent, liste servie, ligne de journal au bon
  niveau).
- **L3.7** `TestTacticalHandler_CelluleMiniTuile` : dix clés snake_case servies (dont `zone`) ; vert
  d'emblée (les champs datent de L1.5). Contrat régénéré : aucun écart (`openapi-gen -check` à jour,
  `generated.ts` inchangé).
- **Mutations** (toutes ROUGES, restauration vérifiée) : hauteur du tueur pour une mort ; autre joueur
  d'une mort = la victime ; camp inversé ; lecture canonique par contribution ; badge sur un frag ;
  `replay_available` toujours vrai (première écriture invalide — compilation cassée —, refaite pour
  compiler, rouge) ; `WithReplay` retiré du câblage.
- **Gate** (avant-plan) : `go build ./...` 0 ; `go vet` service / wire / handlers 0 ; `gofmt -l internal
  cmd` muet ; paquets touchés verts ; module en six lots couvrant les 349 paquets (195 ok, 153 sans
  test, 1 FAIL : `TestLUSRV2Shadow_RafalesBornees_300Candidats`, le test de durée de §8, pendant
  qu'un `api.test` et des `go` d'une autre session tournaient ; rejoué seul puis paquet seul : verts) ;
  `go test ./internal/archlint/...` ok ; `TestNoNewSlugComparison`, `TestAucunTypeAnalysisEnCorpsHuma`
  PASS en `-v` ; `golangci-lint` (cache isolé) 0 issues ; contrat à jour.
- Seuils : `registry_pages.go` 579 L (au-delà de 500, en baisse de 40) ; autres fichiers touchés
  ≤ 469 L ; fonctions neuves ≤ 35 L ; `garderLesOuvrables` 5 paramètres (au seuil).
- Écarts : aucun au périmètre ; `tactical_service_cellule_faces_test.go` (tests de L3.1 / L3.2) est
  un fichier de test neuf non nommé par le plan.

### L4 — Web : largeur d'Ascension, cockpit à trois colonnes, « Cartes jouées » · moyen

Périmètre : `features/ascension/AscensionLayout.tsx` (+ test), `features/tactical/{TacticalPage.tsx,
TacticalMapsColumn.tsx (NEUF), TacticalMapTile.tsx, cockpit.logic.ts (NEUF), tacticalLogic.ts,
i18n.ts}` (+ tests), `lib/i18n/manifests/tactical.toml` (+ généré).

- [ ] L4.0 `npm ci` (node_modules réel) ; premier gate web à blanc consigné (état de départ).
- [ ] L4.1 `AscensionLayout.tsx:78` : `<main className="space-y-6 p-6">` (V2) ; test d'abord
  (`AscensionLayout.test.tsx` : ni `container` ni `max-w-6xl`, `p-6`) ; tests des autres onglets
  rejoués nommément (`AscensionProfilTab`, `AscensionObjectivesTab`, `AscensionCoachingTab`,
  `features/tendances/*`). *Six onglets en héritent (Tendances compris), pas cinq.*
- [ ] L4.2 `cockpit.logic.ts` : constantes D13, `carteEffective` (D11), `normaliserRecherche`
  (minuscules, sans diacritiques), `filtrerCartes` (nom affiché + canonique), partition ouvrables /
  sous plancher (verdict du serveur `sous_plancher`, jamais recalculé), libellés de repli. Tests
  ROUGES d'abord ; mutations : accents gardés, carte sous plancher prise par défaut, URL ignorée.
- [ ] L4.3 `TacticalMapsColumn.tsx` + `TacticalMapTile` compacte (100 px, 16:9, ligne
  « 54 · 30 V / 24 D », barre fine) : S2-S4 ; tests (recherche, active, repli, ouverture du repli
  quand seules les cartes sous plancher correspondent, aucune carte ouvrable).
- [ ] L4.4 `TacticalPage.tsx` recomposé : grille cockpit (D13) ; colonne gauche ; le groupe centre +
  droite monte, TRANSITOIREMENT, la `TacticalAnalysisView` existante (remplacée en L5 et L6) ;
  bascule, grille et pied retirés (§4.A) ; états dans l'ordre existant (composition impossible,
  échec, attente, vide) ; `TacticalPage.test.tsx` et `TacticalPage.relecture.test.tsx` adaptés
  (sélection d'office, URL non réécrite, le fond et les vignettes restent montés en relecture).
- [ ] L4.5 Chaînes neuves de la colonne (FR + EN, mots de la maquette) ; manifeste régénéré.
- [ ] L4.6 §4.A rejoué → 0.
- Gate : gate web.

### L5 — Web : la carte du plan · lourd

Périmètre : `lib/replay/heatPaint.ts` (+ test, deux constantes), `features/tactical/{TacticalPlanCard.tsx,
TacticalPlanFond.tsx, TacticalAnalysisView.tsx, plan.logic.ts (NEUF), tacticalView.logic.ts,
tacticalLecture.logic.ts, i18n.ts}` (+ tests), `components/layout/KPIStrip.tsx` (+ test, supprimés),
`components/ui/metric-trend.tsx` (commentaires), manifeste.

- [ ] L5.1 V8 / D18 : constantes et assertions (`heatPaint.test.ts:282-283`) ; tests du noyau verts ;
  mutation : ancienne valeur → rouge.
- [ ] L5.2 `plan.logic.ts` : texte de l'ⓘ par lecture (S6, D9, D20), bornes de légende (réutilise
  `planLegend`), arrêts de la rampe depuis la rampe peinte (D14), dimensions de la boîte (D13), type
  `TacticalQuestion` étendu à `solde` (+ `QUESTIONS` de `tacticalLecture.logic.ts:287-294`, unité).
  Tests ROUGES d'abord ; mutations : ⓘ sans le pas, rampe horizontale, positif en bas.
- [ ] L5.3 `TacticalPlanCard.tsx` réécrit (S5-S9) : `titleWithInfo(ⓘ, { trailing: pilules })`
  (`components/ui/title-with-info.tsx:32`), pilules « Lecture » / « Joueurs » / « Réapparition »
  (état local comme aujourd'hui), bandeau d'état, corps fond + calque + cadre de la cellule, rampe
  verticale, états vides en titre seul (D24) ; `TacticalPlanFond` à 800 px.
- [ ] L5.4 `TacticalAnalysisView.tsx` : plus de H2, de barre d'outils, de KPI ni de carte
  Coordination ; la colonne de droite garde, TRANSITOIREMENT, `TacticalCellCard` (remplacée en L6).
- [ ] L5.5 Suppressions §4.B (fichiers, tests, exports), commentaires devenus faux corrigés
  (`metric-trend.tsx:10,19`) ; §4.B rejoué → 0.
- [ ] L5.6 Tests : `TacticalAnalysisView.test.tsx` et `.fond.test.tsx` adaptés (le fond reste le même
  `<img>` en relecture ; ⓘ ; pilules ; « Escouade » désactivé ; bandeau d'état ; états vides) ; un
  test par lecture (unité, rampe divergente pour `gagne` / `solde`).
- [ ] L5.7 Chaînes neuves du plan (FR + EN) ; manifeste régénéré.
- Gate : gate web ; `lib/replay/heatPaint.test.ts`, `features/match-view/MatchPositionsHeatmap*`
  et `features/match-replay/**/useReplayHeatmap*` rejoués nommément.

### L6 — Web : la zone sélectionnée · lourd

Périmètre : `lib/match-nav/MatchReplayLink.tsx` (+ test), `features/tactical/{TacticalZoneCard.tsx
(NEUF), TacticalRejeuTile.tsx (NEUF), zone.logic.ts (NEUF), TacticalAnalysisView.tsx,
TacticalPlanCard.tsx, TacticalCellCard.tsx (supprimé), tacticalView.logic.ts, i18n.ts}` (+ tests),
manifeste.

- [ ] L6.1 `zone.logic.ts` : `zoneLaPlusChaude` (D10), coordonnées (S10), valeur signée, sous-ligne
  par lecture, modèle de tuile (fait par face, arme ou catégorie traduite ou rien, badge « seul »,
  « seul · N m », « près · N m », « < 1 », date au fuseau, texte complet), position de l'étiquette
  du plan. Tests ROUGES d'abord ; mutations : |valeur| ignorée, badge arrondi au lieu de tronqué,
  badge sur un frag, fuseau ignoré.
- [ ] L6.2 `MatchReplayLink` : prop `search` et variante `large` (D8) ; tests d'abord (lien avec
  `?t=&clock=`, rien sans `available`, rien sans capability) ; tests de l'Explorateur, de l'Escouade
  et de `match-card` rejoués nommément.
- [ ] L6.3 `TacticalRejeuTile.tsx` + `TacticalZoneCard.tsx` (S10, S11, S15) ; sélection par défaut
  dans `TacticalAnalysisView` ; étiquette du nom de zone sur le plan (S8) ; colonne à la hauteur de
  la carte du plan, liste à défilement interne.
- [ ] L6.4 Suppressions §4.C ; §4.C rejoué → 0.
- [ ] L6.5 Tests : carte de zone (sans sélection, sélection d'office, clic, lecture signée, `solde`,
  lecture d'artefact, zone sans nom), tuile (deux lignes, ellipse sur l'arme seule, bouton absent sans
  artefact, ordre du plus récent au plus ancien), étiquette du plan.
- [ ] L6.6 Chaînes neuves de la zone (FR + EN) ; manifeste régénéré.
- Gate : gate web.

### L7 — Web : sémantique et chaînes · moyen

- [ ] L7.1 V5 et V9 sur toutes les chaînes survivantes de `tactical.toml` : libellés et unités des
  lectures, « Joueurs », « Lecture », « Réapparition » / « Toutes », états vides (D24), bandeau
  d'état (« en attente de traitement », « sans film »), échecs sans conseil, « Coéquipier
  introuvable » sans impératif, plus de « La question », « Clique une zone chaude du plan »,
  « ▼ moins c'est mieux », « échantillon faible », « cuisson », « Spawn de départ », « Cellule
  sélectionnée », « Voir dans le rejeu ».
- [ ] L7.2 Purge §4.D (clés et accesseurs sans lecteur, preuve par grep de chacun) ; manifeste
  régénéré ; garde anti-anglicismes verte.
- [ ] L7.3 Test `tacticalStrings.test.ts` (NEUF) : titres et mots de la maquette FR copiés (S2-S11),
  libellés et unités V5 FR et EN, balayage de toutes les chaînes FR sans impératif listé ci-dessus ni
  anglicisme. Mutation : réintroduire « Spawn de départ » → rouge.
- Gate : gate web.

### L8 — Web : suppressions résiduelles et preuves · rapide

- [ ] L8.1 Rejouer §4.A-D côté web → 0 (hors commentaires historiques) ; tout export, type, fichier
  ou commentaire devenu faux, retiré ou corrigé.
- [ ] L8.2 `TacticalFond.mesure.test.ts` (instrument ignoré par défaut, porte `TACTIQUE_MESURE`) :
  sélecteurs adaptés au cockpit, relevés KPI et pied retirés ; non exécuté (il lit des documents de
  rejeu cuits).
- [ ] L8.3 Ratchets : knip, imports croisés (≤ 7, aucune dérogation morte), couleurs, champs en dur ;
  si un plafond baisse, l'abaisser.
- Gate : gate web.

### L9 — Go : suppressions et contrat · moyen

Le web ne lit plus `echange`, `coordination`, `isolement`, `evenements_*` depuis L5 (L9.1 le prouve).

- [ ] L9.1 Rejouer §4.E-H (producteurs et lecteurs Go, lecteurs web → 0) et la baseline de tests.
- [ ] L9.2 §4.E échange (déplacement des cas survivants, noms inchangés).
- [ ] L9.3 §4.F coordination (liste blanche `no_naked_rate_test.go` réduite).
- [ ] L9.4 §4.G `Isolement` du raster.
- [ ] L9.5 §4.H contrat : `openapi.yaml` et `generated.ts` en baisse seulement, snapshot régénéré
  par la procédure (disparitions listées au journal), alias de `lib/api/types.ts` retirés ; garde
  rejouée sans la variable.
- Gate : gate Go + contrat + gate web ; preuves §4 rejouées → 0.

### L10 — Liens croisés · rapide

- [ ] L10.1 D17 : vérification sur pièces de `MatchViewHeader.MapID` ; état de `feat/matchview-emprise`
  au moment du lot (fusionnée ou non, dit au journal) ; lien posé ; test (route, `search`, absent
  sans `map_id`).
- [ ] L10.2 D16 : vérification sur pièces du libellé FR (tactique vs historique) ; lien posé depuis la
  vignette et test, OU `[!]` avec la justification.
- Gate : gate web (+ gate Go si un fichier Go change).

### L11 — Clôture · rapide

- [ ] L11.1 `docs/CHANGELOG.md` + `docs/FR/CHANGELOG.md` (bloc `[7.5.0]`, entrée « Tactics tab »
  l. 34 : quatre tuiles KPI et carte de coordination retirées, vue cockpit, lecture « Solde »,
  noms de zone, mini-tuiles) ; `docs/RELEASE_NOTES.md` + `docs/FR/RELEASE_NOTES.md` (« A Tactics
  tab » / « Un onglet Tactique », l. 42-46, dont « La coordination d'équipe ») ; `README.md` + `docs/FR/README.md` (« Tactics » /
  « Tactique », l. 34 et 114) ; lignes re-vérifiées au moment d'écrire.
- [ ] L11.2 ADR 0036 relue (test I2 de L2.2 présent dans la liste et le tableau) ; aucune autre ADR
  concernée.
- [ ] L11.3 Statut de chaque item ; §8 relue ; entrée finale du journal.
- [ ] L11.4 Revue adversariale du diff cumulé (lots à risque : L2 lectures bornées, L3 enrichissement
  et câblage, L9 contrat) : à demander au SUPERVISEUR (l'exécuteur n'a pas de sous-agent) ; `[!]`
  tant qu'il ne l'a pas lancée.
- Gate : gate Go complet + gate web complet + contrat, rejoués après les docs.

## 7. Reprise de session

Relire le skill `plan-execution`, puis ce fichier (cases, journaux de lot), puis les dernières
entrées de `.ai/thought_log.md` du worktree et `git -C <worktree> log --oneline -10`. Reprendre à la
première case non statuée du lot courant. Les décisions du §1 sont fermes une fois le « go » donné.

## 7 bis. Relecture plan-review (2026-10-06, phase 1)

Grille `.claude/skills/plan-review/SKILL.md`, passée sur ce fichier :
- §1 structure : objectif et critère (§0) ; lots ordonnés du pur (L1) aux lectures (L2), à
  l'enrichissement (L3), au web, puis aux suppressions contractuelles (L9) ; effort par lot ;
  branche nommée ; bloqueurs documentés (D1 ordre, témoin qui change en L2.3 = arrêt, L10.2
  conditionnel, L11.4 dépendant du superviseur) — OK.
- §2 couches Go : algos purs dans `analysis/tactical` et `analysis/coordination` (solde, zones,
  géométrie, placement, `APortee`), types dans `domain/`, orchestration dans `service/`, SQL dans
  `platform/duckdb` seulement (lecture neuve bornée), port étendu (`ContextesDeMort`), handler sans
  logique (docs seulement) — OK.
- §3 multi-titre : capabilities `film.kill_positions` / `match.events.spatial` (positions),
  `film.replay_artifact` (artefacts), `replay` (onglet) ; classificateur et catalogue absents sur
  Halo 5 = champs absents ; `ErrCapabilityNotSupported` testé (L2.2, L3.6) ; aucun `slug ==` ; aucun
  champ de stats ni asset neuf ; aucun chemin hors `PathResolver` (catalogue par
  `MapCalloutsPath`) — OK.
- §4 adapters : mode, score et issue par le canonique (`PlayerMatchesRepository`), libellés d'arme
  par le résolveur du registre (TOML) — pas de SQL de libellé dans le service — OK ; écart assumé :
  lectures tactiques par repo de port, comme tout l'onglet.
- §5 tests : purs (L1), `:memory:` avec fenêtres bornées (L2), mocks de port (L2.4, L3.6), câblage
  (L3.5), handler (L2.4, L3.7), web logique + composants (L4-L7) ; une mutation par règle — OK.
- §6 logs : sections de durée et `slog.*Context` à chaque dégradation (L3.3) — OK.
- §7 front : aucune route ni clé de requête neuve ; FR + EN dans le manifeste ; issue par
  `useOutcomeMapping` ; libellés de carte, de mode et d'arme venus du Go ; jetons seulement — OK.
- §8 livraison : critère par lot (gate), journal par lot, dépendances externes nommées (lots
  voisins, revue du superviseur) — OK.
- §9 exécutabilité : périmètres fermés (listes, preuves grep), gates à commandes exactes, statuts et
  « aucune case vide », ordre strict, Découvertes, reprise, renvoi au skill — OK.

Défauts trouvés et CORRIGÉS à la relecture : (1) la première version retirait l'échange et la
coordination du contrat avant le web (ordre du brief) — `tsc` aurait rougi au gate du lot Go : D1 ;
(2) elle posait le badge par une comparaison neuve dans le service — troisième copie de la règle de
portée : L1.1 ; (3) elle laissait `KPIStrip` « hors de cette page » alors qu'il n'a pas d'autre
lecteur : D21 ; (4) elle supposait que le port de callouts portait déjà polygones et tranches : D3,
L2.3.

## 8. Découvertes (à consigner ici, pas à traiter)

- (maquette) 3 matchs du 27/04/2026 sans nom de carte dans `match_registry` (MESURES Q2).
- (maquette) 103 rangées pour 77 cartes : 23 cartes réparties sur plusieurs `map_id`, 20 rangées sans
  nom (MESURES Q12) ; la grille groupe par (`map_id`, `map_name`) (`QTacticalMaps`,
  `tactical_repo.go:91`) — même défaut dans la colonne « Cartes jouées ».
- (maquette) `pair_name_fr` / `playlist_name_fr` vides en base, playlist « Quick Play » partout
  (MESURES Q13).
- (maquette) 17 morts par carte sans arme ni catégorie (Illusion et Bazaar, MESURES « Mini-tuiles »).
- (maquette) Le rejeu 2D nomme les zones par centre 3D le plus proche (`calloutsLayer.zoneAt`), règle
  différente de V6 (Bazaar (−7, −1) : « Porte ouest » à 2,36 m, MESURES « Zones nommées ») ; proposer
  la règle V6 au rejeu.
- (phase 1) `TacticalRaster.PointsIgnores` n'a aucun lecteur web (pré-existant) ; non retiré.
- (phase 1) `TacticalFilterBar` n'est importé que par `TacticalPage` ; c'est `useLocalFilterBar` qui
  est partagé avec Synthèse / Citations / Relations.
- (phase 1) Deux implémentations de test « point dans polygone » (oracles `hinavmesh`, `mapdecoupe`) ;
  laissées (D27).
- (phase 1) `celluleIsole` recopiait la comparaison de portée (deuxième copie, `tactical_service_cellule.go:225`) :
  traitée en L1.1 parce que le badge en ferait la troisième.
- (L1) `TestLUSRV2Shadow_RafalesBornees_300Candidats` (`internal/sync/skill`) mesure une durée
  murale (rafale < 2 s) : rouge à 2,018 s dans la suite complète pendant que d'autres sessions
  faisaient tourner leurs `go test`, vert rejoué seul. Test sensible à la charge ; non traité.
- (L2) `golangci-lint` avertit « unknown linters in //nolint directives » : des directives `//nolint` mal
  formées (texte libre lu comme noms de linters, ex. `match_view_builders_team.go:48`
  `//nolint:PLR0913 — clé canonique…`) ; pré-existant, hors périmètre ; non traité.
- (L2) `service/teammates/teammates_squad_echange_test.go` (587 → 592 L) : 5 lignes de double du port ajoutées
  à un fichier déjà au-delà de 500 L ; ACCEPTÉ par le superviseur le 2026-10-06 (dette gelée, aucun
  découpage dans ce lot).
- (L3) Deux copies de la lecture « mon camp / l'autre » sur `Summary.Teams` + `Self.TeamID` d'une ligne
  canonique : `analysis.buildScoreLabelCanonical` (`home_canonical_recent.go`, libellé seul) et
  `service.scoreDuMatch` (`tactical_service_cellule_enrichir.go`, libellé et nature). Une troisième
  imposera le helper exporté et son garde-rail (CLAUDE.md n° 6) ; non traité.
- (L3) `TestLUSRV2Shadow_RafalesBornees_300Candidats` a de nouveau rougi dans la suite complète (38,9 s,
  un `api.test` d'une autre session actif), vert rejoué seul : même constat qu'en L1.
- (phase 1) Le catalogue de callouts couvre AUSSI des cartes Forge (`maps_by_id`, 2 536 zones selon
  `callouts_catalog.go`) : « carte sans catalogue » = carte absente du catalogue, pas « carte Forge ».

## 9. Points où le code contredit le brief (phase 1)

1. Ordre des lots (brief §3) : les suppressions Go ne peuvent précéder le web (D1).
2. `AscensionLayout.tsx` : le `<main>` est l. 78 (brief : l. 75), et l'onglet a SIX onglets
   (Tendances, 2026-10-05) — tous héritent de la largeur.
3. `TacticalFilterBar` n'est pas partagé (§8) ; sans effet (barre inchangée).
4. `TacticalCalloutsStore.ZonesDeLaCarte` ne rend que nom + point de référence : polygones et
   tranches sont jetés par `zonesNommees` (`tactical_callouts.go:75-85`) ; extension additive (D3).
5. Règle (b) : la maquette (`zoneOf` l. 646-666) choisit « la plus fréquente par événement, puis la
   plus étroite », sans marge de 0,25 m ; le brief (TRANCHÉ) fait foi ; le nom du témoin Bazaar peut
   différer de la maquette (D26).
6. `KPIStrip` n'a pas d'autre lecteur que cette page : supprimé (D21).
7. `heatPaint.ts` a un TROISIÈME consommateur : « Occupation du terrain » de la Vue match
   (`MatchPositionsHeatmap.tsx:49,134`), que le lot `feat/matchview-emprise` déclare inchangée (D18).
8. Badge : la maquette met « seul » à ≥ 18 m ; la règle de l'app est inclusive (d = portée → près)
   (D7) ; l'ⓘ dit « au-delà de ».
9. Score : format unique de l'app « X - Y » (`TeamScoreLabel`) au lieu de « 3 – 1 » (D5).
10. Instant : `formatClock` « 5:07 » au lieu de « 05:07 » (D23).
11. « Même IsAvailable que la Vue match » : pour une liste, le port impose `AvailableSet` (même
    présence d'artefact) (D8).
12. Libellé d'arme : la Vue match écrit, au kill feed, le nom PROPRE de `damagetag`
    (`KillSourceIcon.Label`, non localisé) ; le registre de fragdist (`weapon_names.toml`, FR / EN)
    est celui que le brief nomme et que les MESURES montrent (« Marteau antigravité ») (D6).
13. « Cartes sans catalogue (Forge) » : le catalogue couvre des cartes Forge (§8).
14. Attribution des commits : ligne système de la session (D28).
