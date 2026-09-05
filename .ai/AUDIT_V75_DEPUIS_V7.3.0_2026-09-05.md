# Audit feat/v75 depuis v7.3.0 — 2026-09-05 — vers une v2 du rejeu et du film

> Registre d'audit (skill `adversarial-audit`). L'audit ne corrige rien : il produit ce registre,
> qui devient un plan. Annexes (15 rapports d'auditeurs + 14 verdicts de vérification adverse) :
> `.ai/AUDIT_V75_DEPUIS_V7.3.0_2026-09-05_annexes/`. Chaque constat ci-dessous cite son rapport
> d'origine (G1..G10, W1..W5) et son verdict (V-GO-*, V-WEB-*).

## Cadrage

- **Périmètre** : tout ce qui a été fait sur `feat/v75` depuis le tag `v7.3.0` (commit
  `a2719a68c`, 2026-08-04) jusqu'à `736ccf3c3` (dernier commit de code). HEAD réel au moment de
  l'audit : `081871f09`, qui n'ajoute qu'un plan sous `.ai/` ; aucun fichier cité n'a bougé entre
  les deux. 2 058 commits, 3 870 fichiers, +975 k / −17 k lignes (dont ~500 k de dumps, assets et
  archives `.ai/`).
- **Questions de l'utilisateur** (1) architecture, ordonnancement, couches, allers-retours
  superflus ; (2) rejeu 2D et fil des éliminations qui « font la même chose chacun de son côté »,
  code fluide et maintenable ; (3) tests pertinents, non auto-validants ; (4) facilité de porter le
  rejeu à un futur titre ; (5) chaque feature ou graphe neuf derrière une capability active pour
  Halo Infinite seulement. L'UI et l'UX sont hors périmètre.
- **Méthode** : campagne « périmètre × axe » — 15 auditeurs en contexte frais, lecture seule
  (G1 décodeur, G2 flux du rejeu, G3 couches API, G4 chaîne kill, G5 multi-titre Go, G6 tests
  film, G7 tests rejeu, G8 tests API et garde-rails, G9 outils `cmd/`, G10 chaîne des cartes,
  W1 architecture web, W2 duplication web, W3 tests web, W4 capabilities web, W5 Match View),
  puis 14 vérificateurs adverses (consigne « réfuter ; en cas de doute, réfuter ») sur les 86
  constats P0 et P1. Le superviseur a rejoué sur pièces les faits pivots de chaque rapport.
  Aucun `go test`, `go build`, lint ni `vitest` n'a été exécuté : tout est établi par lecture,
  `git` et `grep` (les faits de CI sont lus dans les workflows).
- **Doctrine de référence** : CLAUDE.md (règles 1-16, règles ART, anti-patterns 1-10, section
  multi-titre, tag `gamefiles`), skills `arch-rules`, `frontend-patterns`, `color-tokens`,
  ADR 0008 / 0011 / 0012 / 0019 / 0023 / 0025 / 0026 / 0030 / 0031,
  `.ai/PLAN_MASTER_FILM_KILLFEED_REJEU.md` §6.
- **Dette assumée, non remontée** : baseline lint golangci ; protections mémoire `filmproc` et
  verrou process `filmdec` (D10) ; « le VPS ne décode jamais » ; audit perf du 2026-09-02 (C1-C7
  traités) ; les items de `.ai/V7.5/REGISTRE_REPORTS.md` (cités « au registre ») ; les constats de
  `.ai/AUDIT_V7.2.0_MAIN_2026-08-06.md` (cités « persistant » quand ils le sont encore).
- **Limite de méthode, relevée par la vérification** : la base `a2719a68c` (v7.3.0) tombe la
  veille du merge `feat/replay2d-prod` (2026-08-05). Une partie du chantier « neuf » était déjà
  dans le périmètre de l'audit du 2026-08-06 ; deux réfutations (C4, C6) tiennent à cette
  confusion, et les constats qui persistent depuis sont marqués « persistant 06/08 ».

## Chiffres de cadrage

| Grandeur | Valeur |
|---|---|
| Commits depuis v7.3.0 / dont rejeu web / analysis-replay / filmdec / cmd | 2 058 / 373 / 366 / 254 / 256 |
| `internal/analysis/filmdec` | 108 fichiers, 25 637 L source ; 318 fichiers, 86 327 L de tests (58 % des tests inertes en CI) |
| `internal/analysis/replay` | 114 fichiers, 26 834 L ; 296 fichiers, 80 665 L de tests (187 fichiers sans test exécuté en CI) |
| Code Halo-only sous `internal/analysis/` | 248 fichiers, 57 672 L (0 à v7.3.0) |
| `apps/web/src/features/match-replay` | 203 fichiers, 43 361 L dans UN répertoire plat ; 156 tests, 33 617 L ; 44 % de commentaires |
| `cmd/` | 150 → 176 répertoires, 94 413 L dont 46 459 écrites depuis v7.3.0 ; 11 commandes sans référence hors `.ai/` (7 599 L, 8 %) ; 8 commandes citées dans `docs/COMMANDS.md` |
| Bumps de `ReplayDocument.SchemaVersion` | 43 en cinq semaines (2 → 39) ; parc local 106 artefacts sur 9 schémas, 0 à la version courante |
| Churn (commits par fichier) | `match-replay/i18n.ts` 111, `ReplayCanvas.tsx` 111, `replay/document.go` 76, `replay/build.go` 76 |
| Capabilities | 25 clés fines dont 5 `film.*` ; 0 pour l'artefact de rejeu ; les 5 clés `film.*` sont servies par `GET /titles/{slug}/capabilities` mais aucun client web ne les lit |
| Fonctions > 80 L dans le Go touché (hors cmd) | 102 ; `consumeByName` 805 L (+48 depuis le 06/08), `BuildFromFilm` 361, `BuildFromPositions` 362 |
| Garde-rails `archlint` | 35 (13 neufs) ; 0 des 13 neufs avec « test du test » |

## Bilan de la vérification adverse

| Soumis | Tiennent | Requalifiés | Réfutés |
|---|---|---|---|
| 7 P0 | 3 (P0-1, P0-5, P0-7) | 1 → P1 (P0-4) | 3 (P0-2, P0-3, P0-6) |
| 79 P1 | 32 | 20 → P2 | 27 |
| **86** | **35** | **21** | **30** |

Trente constats sur 86 sont tombés : les « doublons » de la chaîne kill côté Go sont pour la
plupart des décisions écrites (film contre crédit API, lignes per-kill contre cumuls, trois clés
non fusionnables), et plusieurs « manques » ont un mécanisme existant que l'auditeur n'avait pas
vu (endpoint des capabilities, sentinelle auth sur tout le module, golden de registre réel en CI,
persister d'Assaut sur `match_objective_events`). Le registre et la proposition de v2 s'appuient
sur ce qui reste, pas sur l'inventaire brut. Registre final : **3 P0, 33 P1, 21 P2 requalifiés**
plus les P2 des auditeurs (non soumis à vérification, listés à part).

## Synthèse en douze lignes

1. Le cœur est bien fait là où il a été conçu : un seul lecteur de bits, une grammaire ECS unique,
   une source de film unique, des persisters INSERT-only relus par `_latest` (y compris
   `match_objective_events` pour l'Assaut), le registre réel d'un film décodé et figé en CI par la
   mini-bobine, un régime web neuf exemplaire (`_scoreCurve`, `usageLogic`, `valueGridModel`,
   `lib/replay/scoreTimeline`).
2. Trois P0 tiennent. La révision du décodeur de kills n'a pas bougé en 14 commits de décodeur :
   les lignes en base sont exclues à vie de toute reprise (P0-1). Côté web, trois politiques
   distinctes traitent « l'artefact n'a pas d'origine » (P0-5) et l'onglet Chronologie superpose
   deux horloges alors que le dépôt possède le recalage et ne l'applique qu'au rejeu (P0-7).
3. L'artefact de rejeu cumule trois rôles : format de stockage, contrat OpenAPI (100 de ses 165
   types sont des schémas) et modèle de calcul relu pour dériver des tables. D'où 43 bumps de
   schéma en cinq semaines, un parc jamais recuit (arbitrage utilisateur daté) et des projections
   câblées sur la seule branche locale, inactives en production tant que l'ouvrier n'est pas
   activé (P2 latent, qui devient actif au merge de v7.5.0).
4. Le rejeu n'a aucune clé de capability propre : il s'allume par présence de fichier (P2, aucun
   chemin produit ne mène à la page sur Halo 5). Ce qui tient en P1 : deux endpoints promettent un
   503 et servent 200 `[]` (D2) ; trois surfaces (distance de kill, usages de session, filtre de
   l'Explorer) s'affichent sur Halo 5 sans porte (L3-L5). Le canal des clés fines existe déjà
   côté API ; il manque le client web.
5. Le décodeur Halo Infinite (248 fichiers, 57 672 L) vit sous `internal/analysis/` sans ADR ni
   exemption (D5, persistant 06/08 et multiplié depuis) ; deux sites servent le paquet Halo à tout
   titre qui déclarerait `film.kill_source` (P2 : le piège suppose un décodeur, pas une ligne de
   TOML).
6. Côté web, la duplication est dans les micro-blocs des calques, prouvée à l'octet : index des
   vies ×4, projection monde → canvas ×8 (34 appels), `allyTeamFromScoreboard` ×3, pulsation ×3.
   Le prédicat d'intervalle ×10 est uniforme (P2). Le « roster reconstruit six fois » est réfuté
   (une implémentation, mémoïsée).
7. Côté Go, le décodeur porte ses propres copies : préambule d'événement ×6 avec un canonique sans
   appelant de production et une divergence `dom3` (7 mesuré contre 8 recopié), triple boucle
   bipède ×9 identique caractère pour caractère, deux décodeurs d'entité morts (586 L) dont la
   cible RE est déclarée fausse, 22 setters sans appelant et deux bascules sans date.
8. Les tests sont pertinents et honnêtes dans leur masse, mais la CI ne prouve pas le décodage
   par familles : aucune assertion de valeur sur un document obtenu d'un film (G1), 26 familles de
   balayage sur ~34 sans octets réels en CI (F3), les deux specs de rendu jamais jouées (M1), la
   baseline de présence aveugle sur tout le chantier (G3), le cron de purge sans test (G7).
   L'oracle Cheat Engine est épuisé et son test n'assertionne rien : ce n'est pas un filet manquant.
9. Les garde-rails (35 `archlint` + les gardes web) tiennent sur des listes manuelles : deux tables
   append-only hors des gardes ART (G4), le lint couleur canonique hors CI remplacé par 9 copies
   divergentes (M6), un garde de porteur sur liste (M4), une allowlist croisée démentie par quatre
   imports (N4). Aucun ratchet neuf n'a de test du test (P2).
10. `cmd/` a doublé (94 k lignes) : onze commandes sans référence (7 599 L, plusieurs au registre
    des reports), une sentinelle mémoire en triple exemplaire alors que les copies importent déjà
    le paquet canonique (H5), l'exemption lint « scripts ponctuels » qui couvre `cmd/server` et
    `cmd/replay-worker` (H7), un catalogue versionné réécrit par le runtime (P1, avant merge), et le
    maillon `livraison.py` de la chaîne des sons hors dépôt (H2).
11. Le fil des éliminations et le rejeu partagent bien un modèle côté Go (`match_kill_events`,
    `SlotIdentityFrom` qui confronte film et API) ; c'est côté web que la jointure est refaite
    dans 12 `useMemo` d'une route sans test et que l'horloge n'a pas de foyer unique.
12. Le plan du projet avait prévu ces gestes (`PLAN_FINALISATION_REJEU_2D.md`, lot 3 : découper le
    paquet Go, découper le canvas, poser `film.replay2d`, statuer les champs non lus) : ils sont
    restés ouverts pendant ~700 commits sur le même code. La v2 proposée ci-dessous les reprend et
    les complète.

## Constats retenus — P0

### P0-1 — `KillSourceDecoderRev` figé alors que le décodeur a changé (G2, V-GO-A1)
- Où : `apps/go-api/internal/sync/killcollector/collector.go:65` ; prédicat de reprise
  `killcollector/postsync.go:369-377`, `cmd/levelup/cmd_backfill_killsource.go:403`.
- Règle : contrat écrit sur la constante elle-même (« LA FAIRE ÉVOLUER à chaque changement de
  décodage ») ; anti-pattern 9 (doc inversée).
- Conséquence : 14 commits sur `games/halo_infinite/film/killsource/` depuis v7.3.0, 0 bump (le
  seul commit touchant la ligne est un déplacement de paquet). Le merge `328d83232` déclare une
  correction du kill-feed des matchs à véhicules ; les lignes déjà en base portent la révision
  courante et sont exclues à vie de la reprise : source du dégât, catégorie et assistant servis
  faux sur la fiche, la carrière et la distance de kill, sans compteur.
- Reproduction : `git log --oneline v7.3.0..HEAD -- apps/go-api/internal/games/halo_infinite/film/killsource/ | wc -l ; git log -G'KillSourceDecoderRev = ' --oneline v7.3.0..HEAD -- apps/go-api/internal/sync/killcollector/collector.go`
- Vérification adverse : **tient, P0 maintenu**. La mesure inverse même l'argument de l'auditeur
  G2 sur le double décodage : 43 bumps de schéma d'artefact contre 0 bump de révision.
- Traitement : bumper la révision ; garde-rail d'empreinte (hash du paquet décodeur → la révision
  doit changer) ; une révision par famille de lignes (kills, tirs, positions).

### P0-5 — Trois politiques différentes pour « l'artefact n'a pas d'origine » (W1, V-WEB-1)
- Où : `features/match-replay/replayWindow.ts:112` (null), `killFeedLogic.ts:237` (appariement
  mesuré), `replayMediaLogic.ts:56` et `seatLogic.ts:179` (0). Le `0` de `presenceFeed.ts:69` est
  inatteignable.
- Règle : R6 (cinq copies de la soustraction, trois décisions) ; anti-pattern 9
  (`replayMediaLogic.ts:43` affirme « même choix que le reste du lecteur »).
- Conséquence : sur un artefact sans `originMs` (5 des 106 artefacts locaux), la frise se cadre sur
  le film entier, la piste médias à décalage 0 et les pistes kills au décalage apparié : une capture
  et le frag qu'elle montre jusqu'à ~40 s l'un de l'autre (chiffres de l'en-tête de
  `killFeedLogic.ts:20-21`).
- Reproduction : `grep -n "originMs == null\|Number.isFinite(originMs)\|Number.isFinite(doc.originMs)\|typeof origin === 'number'" apps/web/src/features/match-replay/{replayWindow,replayMediaLogic,presenceFeed,seatLogic,killFeedLogic}.ts`
- Vérification adverse : **tient** (correction : trois politiques distinctes, pas quatre).
- Traitement : une fonction `replayClock(doc, header)` rend une horloge établie ou null, un seul
  verdict pour la page ; garde-rail interdisant `originMs` hors de ce module.

### P0-7 — Deux horloges m:ss côte à côte dans l'onglet Chronologie (W5, persistant 06/08 étendu, V-WEB-4a)
- Où : `features/match-view/MatchViewTabChronology.tsx:84-121` empile `MatchKDCumulChart` (axe
  `event_time_ms`, recalé gameplay côté Go) et `MatchScoreCurveChart` (axe `frame × frameIntervalMs`,
  grille de l'artefact) ; aucun `t0Ms/originMs` sous `features/match-view/`.
- Règle : contrat écrit `domain/match_view.go:154-163` (« un consommateur qui pose les deux sur la
  même timeline décale les kills de ~18 à 28 s »).
- Conséquence : « le score bascule à 2:10 » et « le frag décisif à 2:10 » peuvent désigner des
  instants distants de ±20 s ; `_scoreCurve.ts:104-110` affirme que son 0 est le coup d'envoi.
- Reproduction : `grep -rn "t0Ms\|t0_ms" apps/web/src/features/match-view --include=*.ts --include=*.tsx | grep -v test` (vide).
- Vérification adverse : **tient, renforcé** — le dépôt possède le recalage et ne l'applique qu'au
  rejeu ; un test verrouille la prémisse fausse « borné au coup d'envoi » de la courbe de score.
- Traitement : `lib/replay/matchClock.ts` (les conversions entre `event_time_ms`, `t0_ms`,
  `originMs`, `t0FilmMs`) consommé par le fil ET par les graphes de la Match View.

## Constats retenus — P1 (par thème)

Chaque ligne : constat, localisation, règle, conséquence en une phrase, verdict. Le détail, la
commande de reproduction et le traitement proposé sont dans le rapport d'axe et le verdict cités.

### A. Flux de production et catalogues (G2, G10)
- **A0** (ex-P0-4) Le runtime écrit un catalogue versionné que chaque déploiement écrase :
  `sync/replayartifacts/mvar_rattrapage.go:175-236` → `mapcatalog.AddEntry` →
  `data/titles/halo_infinite/reference/map_weapon_pads.json` (suivi par git) ;
  `scripts/deploy.sh:33` fait `git reset --hard origin/main`. En local, le commit `5426e256b`
  (« journal(wip)… sans relecture ») a fait entrer +332 lignes de données de référence.
  **Vérification (V-GO-D2) : tient, gravité P0 → P1** — la production ne l'exécute pas encore
  (code absent de `main` ; en placement `worker` sans jeton, la passe dégrade en `off` avant
  l'appel) ; le volet local est confirmé et renforcé (l'écriture vient bien du runtime).
  À traiter **avant le merge de v7.5.0**, sinon le P0 devient réel au premier déploiement.

### D. Multi-titre et capabilities côté Go (G5, V-GO-B2)
- **D2** Doc inversée : `/positions` et `/objective-events` promettent un 503
  (`api/wire/registry_pages.go:113-117`) qui n'arrive jamais, les tables du film étant créées pour
  tous les titres par le registre de migrations commun → 200 `[]` sur Halo 5. **Tient** — chaîne
  vérifiée maillon par maillon ; doc inversée à trois endroits.
- **D5** Le décodeur du film et ses tables Halo Infinite vivent sous `internal/analysis/` : 248
  fichiers / 57 672 L (persistant 06/08, multiplié) ; `vehicle_families.go`,
  `usage_summary_families.go` portent des identifiants `eqip`/`vehi` en dur. ADR 0012. **Tient** —
  chiffres reproduits à l'unité ; aucune ADR, exemption, garde-rail ni entrée de registre ne couvre
  l'emplacement.

### E. Architecture du décodeur (G1, V-GO-C1)
- **E1** `filmdec/entity.go` + `entity_quant.go` : deux décodeurs de record ~92 % identiques,
  0 appelant, ~565 lignes mortes sur 586 (trois helpers vivants : `bitLen`, `readQuantStat`,
  `quantStatDefaultWidth`). R7. **Tient, aggravé** — `components_batch7.go:6-8` documente la
  supersession (« CORRECTED 2026-06-14 : l'ancien `DecodeEntityRecordQ` visait une autre
  fonction ») ; la doctrine maison a supprimé `ParseStatborgRecord` pour le même motif.
- **E2** 22 `Set*` exportés sur 66 sans aucun appelant ; deux bascules A/B sans date ni critère
  (`useLegacyAngularVel` `traverse.go:1134-1140`, `useBipedDefaultStateDeser` `:1049-1064`) là où
  `simStateComplete` dans le même fichier porte les trois ; deux crochets prouvablement toujours
  nil promenés par 8 blocs de sauvegarde/restauration. R7, R11. **Tient** — deux chiffres
  corrigés : 8 blocs et non 12 ; et `consumeObjectAngularVelocity` est du code VIVANT (déser
  correct d'i3 pour `ti=40`), seule la branche `if useLegacyAngularVel` est morte — le traitement
  ne doit supprimer que la branche et le drapeau, pas le désérialiseur.
- **E3** Grammaire de la liste d'événements (préambule 9 bits) écrite six fois sous deux conventions
  (`Skip(1)+ReadBit` vs `Skip(2)`) ; le canonique `event_list.go` n'a qu'un appelant de production,
  lui-même ; divergence active `dom3RefWidth = 7` (mesuré par oracle) contre `3: 8` dans deux
  tables recopiées de l'exécutable ; la sixième copie (`transloc_events.go`) est postérieure de
  deux jours au canonique ; la justification écrite du refus de factoriser
  (`biped_pickups.go:207-209`) est périmée. R6, anti-pattern 8. **Tient** — rien de faux servi
  aujourd'hui (la production ne lit pas le domaine 3).
- **E4** Triple boucle du marcheur de records delta bipède copiée neuf fois
  (`grep "bipedHeaderBits + bipedIndexBits\*bipedMinMaskCnt"`). R6. **Tient** — squelette vérifié
  identique caractère pour caractère par `diff` entre deux copies ; aucune grammaire propre à un
  événement ne justifie la recopie.
- **E6** (persistants 06/08, non soumis) `consumeByName` 757 → 805 L ; 5 fichiers > 500 L
  (+347 L sur les quatre déjà relevés) ; `probe_export.go` 9 exports sans appelant ;
  `frame_debug.go` 0 appelant.

### F. Tests du décodeur (G6, V-GO-C1)
- **F3** 26 familles de balayage sur ~34 utilisées en production n'ont aucun test sur octets réels
  en CI (4 seulement via `equivalence_minifilm_test.go`, redoublées par `zero_disque_test.go`).
  **Tient, plus fort qu'annoncé** — les points d'entrée de famille (`ScanCamoStates`,
  `ScanZoomEvents`, `ScanBipedPickups`, `ScanInventoryDeltas`, …) ne sont appelés par AUCUN test
  de `filmdec`, gardé ou non ; seules les enveloppes `ScanFilm*` le sont, depuis des instruments
  gardés par des variables d'environnement.

### G. Tests du rejeu, de l'API, garde-rails (G7, G8, V-GO-C2)
- **G1** Aucune assertion de VALEUR en CI sur un `ReplayDocument` obtenu d'un film : les deux
  appels à `BuildFromFilm` exécutés attendent une erreur ; la seule cuisson réelle
  (`wire/build_queue_worker_binary_integration_test.go:197-205`) n'assert que la forme ; 7 étapes
  d'équivalence en CI contre 49 en local. **Tient** (titre réécrit : un document EST obtenu par
  l'e2e `wire`, rien n'est vérifié sur son contenu).
- **G3** Le gate « présence des tests » (`scripts/check_test_baseline.sh`) ignore tout le chantier :
  0 entrée pour `analysis/replay`, `replaybuild`, `replayartifacts`, `killcollector`,
  `objectiveevents`. **Tient** — le contrôle « par paquet » qu'annonce le script est en réalité
  global, et la CI ne l'invoque pas (doc inversée).
- **G4** `kill_positions` et `match_weapon_hit_distance` ne sont enrôlées dans aucune des deux
  listes anti-ART (`sync/no_art_patterns_test.go:68`, `append_only_state_guard_test.go:26`).
  Règle ART 3. **Tient** — enrôlement gratuit : aucune violation existante.
- **G7** `ReplayPurgeCron.RunOnce`, seul cron qui supprime des fichiers, n'a aucun test ; inverser
  la garde `months <= 0` purge tout le parc au premier tick. **Tient, aggravé** — la godoc dit
  « exporté pour les tests ».

### H. Outils `cmd/` (G9, V-GO-D1)
- **H2** La chaîne des sons a un maillon de livraison en Python hors dépôt (`_outils/livraison.py`,
  générateur de `weaponSoundVariations.ts` et des 177 assets de `static/sounds/`). R2. **Tient,
  une pièce à charge sur deux réfutée** — `akpk_unpack.py` est déjà porté en Go
  (`cmd/weapon-sounds/pck_dump.go`, 2026-09-02) et la recette est versionnée
  (`.ai/V7.5/RECETTE_SONS_ARMES.md`, qui nomme six scripts) ; il reste cinq scripts hors dépôt,
  dont `livraison.py`, et aucun mode `livrer` parmi les 39 modes de la commande.
- **H5** Troisième implémentation vivante de la sentinelle mémoire
  (`cmd/levelup/backfill_memlimit.go`, `cmd/replay-worker/memlimit.go`) alors que
  `filmproc/memguard.go` est canonique. Anti-pattern 8. **Tient, gravité à relever** — la
  justification écrite des copies (« factoriser exigerait d'ouvrir un paquet interne partagé »)
  est démentie par le fichier lui-même : `backfill_memlimit.go:48` importe déjà `internal/filmproc`
  et l'appelle (`:133-134`), idem `replay-worker/job.go:30` ; `memguard.go` sert déjà les deux
  doctrines divergentes par callback. Aggravant : le garde-rail `archlint`
  (`no_unbounded_film_loop_test.go:168-172`) entérine cette justification périmée et valide
  l'erreur au lieu de l'attraper.
- **H7** `.golangci.yml:152-168` exempte `^cmd/` de 12 linters au motif « scripts ponctuels » mais
  couvre `cmd/server` (déployé par `release.yml:62`) et `cmd/replay-worker` (`deploy-worker.sh:100`).
  **Tient** — l'exemption fine (`:211`, `:246`) est morte, déjà couverte par `^cmd/` ; le gate est
  un ratchet `--new-from-merge-base` (`Makefile:307`), donc l'exemption mord sur le code NEUF des
  deux binaires de production.

### I. Chaîne des cartes (G10, V-GO-D2)
- **I1** Deux tests ouvrent l'installation du jeu hors du tag `gamefiles`
  (`cmd/mapstruct-build/equivalence_gamefiles_test.go` sans `//go:build`,
  `cmd/mapfond-build/reglages_test.go`) ; le ratchet `himap/corpus_tag_test.go` ne regarde qu'un
  paquet. **Tient** (coût à ramener à 2 modules).
- **I2** `internal/himap/heightfield.go` : 175 lignes mortes maintenues par leur seul test. R7.
  **Tient, renforcé** — aucune réservation au registre ; l'en-tête ne porte pas l'avertissement que
  le handoff lui prête.
- **I3** Cinq fichiers > 500 L créés pendant la campagne sans exemption (`himap/cuisson_forge.go`
  1 048, `cmd/mapfond-build/reglages.go` 879, `himap/cartes_forge.go` 681 = table de 89 cartes
  compilée, `himap/cuisson.go` 615, `cmd/mapfond-build/cuisson.go` 537). R5. **Tient** — chiffres
  exacts, périmètre même sous-estimé ; nuance écrite sur `cartes_forge.go`.

### J. Architecture de la page de rejeu (W1, V-WEB-1)
- **J3** `replaySchemaLogic.ts` : 32 lignes, 28 commits, zéro lecteur à l'exécution ; seul
  consommateur, son garde-rail. R7. **Tient** — une décision datée existe (registre des reports
  l.449) mais sa condition a été franchie sans exécution.
- **J5** `roundAtFrame` (`lib/replay/scoreTimeline.ts:315`) mort dans le commit qui l'a remplacé,
  gardé vivant par 6 assertions, avec la sémantique que son remplaçant documente comme fausse. R7.
  **Tient.**

### K. Duplication dans la page de rejeu (W2, V-WEB-2)
- **K1** Index « vies par slot » + « la vie qui couvre l'image » réécrit 4 fois byte-identique
  (`fireMark.ts:51-60`, `grappleLayer.ts:63-72`, `shotFx.ts:90-101`, `thrusterDashFx.ts:143-152`).
  **Tient** — 4 blocs md5-identiques, aucune canonique, garde-rail qui ne grep que `livesByXuid`.
- **K2** `allyTeamFromScoreboard` extraite « pour ne pas être dupliquée » et dupliquée 3 fois
  (`useReplayBombBlast.ts:79-82`, `useReplayFlagCarries.ts:119-122`, `useZoneStates.ts:78-81`) ;
  `teamOfXuid` byte-identique entre deux hooks. **Tient** — canonique jamais importée par les 3
  hooks, aucun garde-rail.
- **K3** Passage monde → canvas ré-emballé 8 fois (4 corps + 4 closures byte-identiques, 34 appels)
  ; le cadrage est un objet que rien ne sait projeter ; 9 déclarations du même type `CanvasView`.
  **Tient.**
- **K4** `alphaOf` + 4 constantes de pulsation copiés dans 3 calques portés. **Tient** — md5
  identiques ; conséquence réécrite : les trois glyphes relèvent de trois modes distincts et ne
  co-occurrent pas (le risque est la dérive, pas la désynchronisation visible).

### L. Capabilities côté web (W4, V-WEB-3a)
- **L3** `MatchKillDistanceSection` rend sa carte inconditionnellement (0 `return null`) avec, sur
  halo_5, un message promettant un décodage de film qui n'aura jamais lieu ; doc inversée
  `MatchViewPage.tsx:352`. **Tient.**
- **L4** `SessionUsageSection` affiche sur halo_5 une carte « Ce titre ne publie pas de résumé
  d'usage des films » (`unsupported` traité comme `empty`, pas `hidden`, `usageLogic.ts:476-482`).
  **Tient.**
- **L5** Filtre « Avec rejeu / Sans rejeu » de l'Explorer et colonnes « rejeu » de deux tableaux
  rendus sans porte, là où la colonne Waypoint voisine est gatée. **Tient.**

### M. Tests de la page de rejeu et couleurs (W3, V-WEB-4b, V-WEB-3b)
- **M1** Les deux seules preuves de rendu (`e2e/replay-explosion-raster.spec.ts`,
  `replay-muzzle-raster.spec.ts`) n'ont jamais tourné en CI : job gaté `pull_request`
  (`ci.yml:524`), 0 merge de PR sur la fenêtre. **Tient** (fenêtre corrigée : depuis le
  2026-08-15, 331 commits).
- **M3** Le seul test qui parle de `ReplayCanvas.tsx` compte ses lignes (cliquet ≤ 665 dans
  `placementFamily.guard.test.ts:300-304`) ; la composition de la scène n'a aucun test. **Tient.**
- **M4** `carrierPosition.guard.test.ts` : 3 cas sur 4 sur liste manuelle ; un 6e calque de porteur
  échapperait au garde. **Tient.**
- **M6** Le garde-rail couleur canonique (`tools/lint-no-hardcoded-colors.mjs`, seuil 0, périmètre
  `features/` + `components/`) n'est joué qu'en `pre-push` (`lefthook.yml:75-77`, contournable) :
  aucun step CI, aucun script npm, aucune règle ESLint, pas dans `make gate-push`. **Tient** —
  nuance : 9 copies partielles tournent en CI, sur une quinzaine de fichiers nommés à la main,
  avec trois regex inégales (`fxInk.guard` sans aucune classe Tailwind).

### N. Match View (W5, V-WEB-4a)
- **N1** `MatchKDCumulChart.buildOption` : 260 L de logique dans un `.tsx` exécutées sans oracle ;
  `MatchCadenceChart.buildOption` 162 L sans aucun test. R5, anti-pattern 7. **Tient.**
- **N4** L'allowlist `match-view=>match-replay` (`tools/lint-cross-feature-imports.mjs`) se dit
  « strictement bornée au chargement de l'artefact » et quatre imports la démentent
  (`equipmentKillBadges.ts`, `MatchImpactBadgesBar.tsx`, `MatchViewTabChronology.tsx`).
  Anti-pattern 9. **Tient.**

## Constats requalifiés P2 par la vérification adverse (21)

Faits exacts, gravité abaissée ; classés au backlog avec leur motif.

- **A1** (G2) Les trois dérivations post-cuisson (usage, statistiques d'Assaut, T0 film) ne sont
  câblées que sur la branche « construction locale » (`replayartifacts/artifacts.go:321-352`) ; en
  placement `worker` elles ne s'exécutent jamais. Motif : le tiers T0-film est déjà au registre
  (l.18) et le défaut est latent tant que l'ouvrier n'est pas activé en prod. **Note du
  superviseur : devient actif au merge de v7.5.0 — placé dans le lot pré-release.**
- **A2** (G2) Les dérivés n'ont aucun rattrapage (work-list = artefacts cuits ce cycle, rattrapage
  = absence de fichier). Motif : prédicat `os.Stat` justifié et mesuré ; corpus périmé = arbitrage
  utilisateur daté (registre l.17).
- **B5** (G4) Tolérance 5 ms recopiée quatre fois (`games/halo_infinite/events.go:31`,
  `sync/collect.go:146`, `sync/engine_highlight_events.go:383`, `killcollector/credit.go:83`) :
  littéral magique portant un invariant écrit, sans garde-rail. La moitié « conversion
  `HighlightEvent → RawEvent` ×3 » est réfutée (2 copies).
- **C1** (G3) SQL brut, appel réseau et runner dans la racine `api/`
  (`api/wire/registry_replay_build.go:92-129`, `registry_build_queue.go:46-348`). Motif :
  entorse réelle sur des fichiers neufs, mais les conséquences de testabilité annoncées sont
  fausses et le motif a dix précédents.
- **C2** (G3) Quatre services (et non trois) injectent `port.ReplayService` alors que
  `port.ReplayAvailability` existe. Motif : couplage réel, préjudices annoncés faux.
- **C3** (G3) `service/replay_weapon_labels.go:34` importe `games/halo_infinite/replaylabels`
  (paquet qui se déclare Infinite). Motif : effet nul (Halo 5 n'a pas `replay_labels.toml`) ; une
  occurrence, pas trois.
- **D1** (G5) Toute la chaîne du rejeu (artefact, 4 routes `/replay*`, étape post-sync 1.58) n'a
  aucune clé de capability ; le plan prévoyait `film.replay2d` (lot 3.5). Motif : aucun lien ne
  mène à la page sur Halo 5, garde loopback.
- **D3** (G5) L'étape 1.58 met en file (`enqueueAll`) avant sa sonde de titre. Le volet UGC est
  réfuté (une seconde sonde garde `rattraperCartesAbsentes`) ; inatteignable aujourd'hui.
- **D4** (G5) `killcollector/classifier.go:39-44` et `replaybuild/replaybuild.go:431` servent le
  paquet Halo à tout titre qui déclarerait `film.kill_source`. Motif : le piège suppose un
  décodeur, pas une ligne de TOML.
- **E5** (G1) `killcollector/positions.go:166-211` enchaîne quatre balayages sans
  `LockProcessDecode`, contre `decode_gate.go:16-18`. Motif : aucune concurrence possible
  (0 goroutine, appelants séquentiels) ; un garde-rail du verrou existe déjà
  (`replay/world_object_precision_guard_test.go:107-112`), borné à `BuildFromFilm` : à généraliser.
- **F4** (G6) Le garde-rail de `consumeByName` est différentiel entre deux copies ; `ecsRow` ne
  capture ni `grammar` ni `bits_typ` (263 composants co-renseignés, 179 largeurs entières).
  Motif : `consumeByName` est exercé sur octets réels par le golden inconditionnel, dont la
  population ne couvre toutefois pas l'archétype bipède.
- **G5** (G8) La liste « fermée » de `no_film_reread` omet 4 enveloppes (pas 2). Motif : dette
  inventoriée au `REGISTRE_REPORTS.md:15`, passe inerte (`not_exposed`).
- **H3** (G9) Dix chaînes de fabrication d'assets versionnés hors de toute documentation ;
  `docs/COMMANDS.md` cite 8 commandes sur 176 (parité FR tenue). Motif : la « règle enfreinte »
  citée n'existe nulle part ; `COMMANDS.md` se déclare cheat-sheet ; six oracles détectent un asset
  périmé.
- **J2** (W1) Le recalage d'horloge (`alignFeed`) est refait dans `killFx.ts:119` et
  `replaySound.ts:630` alors que la route passe déjà `feedEntries`. Motif : coût, pas divergence.
- **K5** (W2) Prédicat « l'intervalle couvre l'image » : 10 écritures, 2 orthographes. Motif : la
  divergence de borne alléguée n'existe pas (même convention fermée à `t1`, 10 sites uniformes).
- **L2** (W4) La route de rejeu n'est gatée que par `matchmaking` : trois requêtes 404 et un état
  vide sur halo_5. Motif : aucun chemin produit n'y mène ; item de plan ouvert
  (`PLAN_FINALISATION_REJEU_2D.md:414`).
- **L8** (W4) Sons servis depuis `/static/sounds/halo_infinite/` quel que soit le titre
  (`replayAudioMix.ts:61`, sans slug ; `replaySound.ts:16` contredit `:206`). Motif : sans effet
  possible avant qu'un second titre ait un pack de sons ; même classe que `teamLabel.ts`.
- **M5** (W3) `canvasInk.guard.test.ts` promet « chaque `InkVar` » et n'en vérifie qu'une sur six.
  Motif : le garde couvre 1/1 de son périmètre déclaré ; reste une liste non dérivée du type et un
  docstring trop large.
- **N2** (W5) `MatchCombatCtfOverlay.test.tsx` : quatre tests dont le mock tronque l'option à
  80 caractères, seul test de l'overlay CTF. Fait exact, gravité abaissée (V-WEB-4a).
- **N3** (W5) Quatre écritures du format `MmSSs`, trois dans `features/match-view/`, alors que
  `formatDurationMShort` existe. Fait exact ; le volet « doc inversée » est réfuté.
- **Résidus de constats réfutés** : deux formateurs d'horloge coexistent (`replayLogic.formatClock`
  `Math.floor` visible, `formatClockMMSS` `Math.round` sur un `title` inerte) — ex-P0-6 ;
  `match_player_positions` (heatmap) n'a pour producteur que `diag_weapons_v3 -write`, serveur
  arrêté (G9, non contredit : la vérification a porté sur `match_objective_events`) — ex-P0-3,
  voir escalade 1 ; trois fonctions portent `paquetUS/1000 ∓ deathOffsetMS` sans helper ni
  garde-rail (ex-B6) ; l'en-tête de `killsource_weapon_scope.go:7-8` nomme le moteur de citations
  qu'il ne sert pas (ex-B1) ; SQL brut sur la clé `sync_meta` (ex-G6).

## Constats P2 des auditeurs (non soumis à vérification adverse)

- G1 : 37 enveloppes `ScanFilm*(dir)` (dette datée, lot 6) ; formes d'entrée/retour hétérogènes
  des 24 étapes de `BuildFromFilm` (deux balayages sans erreur ni compteur).
- G2 : `os.ReadFile` + `Unmarshal(ReplayDocument)` en 5 exemplaires ; `objectiveevents/awards.go`
  282 L sans appelant ; 4 renvois de doc vers des fichiers inexistants (`engine_postsync.go`) ;
  `BuildFromFilm` 361 L / 26 balayages / ≥ 7 dépendances d'ordre implicites ; `ReplayDocument`
  triple rôle (stockage, contrat, calcul) — C7 : 100 des 165 types exportés de `analysis/replay`
  sont des schémas du contrat OpenAPI public (`handlers/replay.go:68,91,116`).
- G3 : ports publiant des types `analysis/` (`port/session_usage.go`, `services.go`) ;
  `MatchViewService` 19 dépendances / 15 `With*` ; trois fragments du découpage anti-god-file
  repassés > 500 L ; handler couplé au service concret (`handlers/presence.go`).
- G4 : 4e définition de la base crédit (mesurée fausse) en code mort dans une migration ;
  `TestNoRawKillScopeLiteral` refuse `"scan"`/`"marche"` (3 faux positifs documentés) et laisse
  passer le test en négatif ; `TestUneSeuleFormuleDeDistance3D` ne voit qu'un répertoire ; même
  champ documenté sur deux horloges.
- G5 : `collectHits` gaté par une clé de publication que halo_5 déclare ; le kill feed enrichi
  ignore `match.killfeed.per_kill` (supported pour halo_5) ; cache de film non partitionné par titre.
- G6 : 85 fichiers de test (21 213 L) sans aucune assertion (dont `default_state_ti42_oracle_test.go`)
  ; 6 écrivains de bits dans les tests ; 8 recettes pointant `Projects/LevelUp` (chemin inexistant).
- G7 : doc inversée sur le golden d'entrées ; échelle de la jauge de zone tautologique ; 7 copies
  « marshal + write d'un artefact » dans `replayartifacts` ; 187 fichiers / 163 gates d'env sans
  inventaire ; témoins `.mvar` versionnés sous une règle `.gitignore` qui les exclut ; différentiel
  `named_onepass` partageant `incrementTimes`.
- G8 : ratchets sur nombre nu `0.83`/`1.59` ; 0 « test du test » sur 13 ratchets neufs ;
  `start_time_utc` en `TIMESTAMP` nu dans 8 fixtures ; 3 fixtures DDL hors garde de parité ;
  kill-switch daté 2026-10-01 non branché sur `TestNoExpiredTODO` ; `no_second_artifact_sink` sans
  saut de commentaires ; `filmdecVarsGeles` relevé 2 fois en 2 jours, baisse en `t.Logf`.
- G9 : `cmd/server/main.go` `main()` 1 437 → 1 538 L ; en-têtes périmés (`cmd/killsource`,
  `replay-worker/memlimit.go`) ; 4 `sql.Open` à la main ; `cmd_backfill.go` 1 026 L ; huit sondes
  auto-déclarées jetables encore présentes (hors celles au registre).
- G10 : 7 loaders de catalogues + cache dans `analysis/replay` (le préjudice de concurrence est
  réfuté, C4 ; reste l'emplacement) ; 62 des 118 exports de `himap` sans appelant externe ;
  `mapcatalog.WriteAtomic` réimplémente `platform/atomicfile` ; ~16 700 L Halo-only hors de
  `games/halo_infinite/` ; persistants `DescribeRootBlocks`, `MeshKey`.
- W1 : la route `replay.tsx` assemble deux sources dans 12 `useMemo` sans test ; 17 extractions du
  canvas sur un cliquet qui compte les commentaires ; 59 fichiers / 10 996 L citent un seuil de
  taille comme raison d'exister ; deux conventions de calque (10 fonctions vs 11 `.paint()`) =
  58 dépendances de `draw` ; 11 faux hooks (996 L) dont 9 sans test ; position de lecture en trois
  représentations pilotées par ref à travers 23 modules.
- W2 : moule Fx dupliqué (`flagCaptureFx` ≡ `bombBlastFx` à 4 lignes près) ; `buildSoundTimeline`
  136 L à mi-migration ; 3 infobulles répétant la géométrie ; helpers contournés (`botKey`,
  `frameIntervalMs || 100`, `ZONE_FILL_ALPHA` 0.09 vs 0.1) ; `alignFeed` exécuté 4×, `buildPlayers`
  3× par chargement (coût, pas divergence).
- W3 : 11 fichiers de test > 500 L sans gate ; fixture de poses figeant `showDropped: false` contre
  la prod ; aucun seuil de couverture web ; 25 modules / 3 261 L sans aucun test (dont
  `ReplayCanvas.tsx`).
- W4 : deux requêtes film par match visité sur halo_5 (503) ; « power-ups », « SPRITE » ;
  `teamLabel.ts` retombe sur les noms d'équipe Halo Infinite.
- W5 : `ValueGrid` absent du catalogue et du bac à sable ; 7 chaînes FR en dur dans les options
  ECharts ; 4 `rgba()` non marqués ; `ExplorerMatchesTable.tsx` +28 L sur dette gelée ; ~40 chaînes
  i18n dupliquées match-view/squad.

## Constats écartés

### Réfutés en vérification adverse (30)

| Constat | Axe | Motif d'écart |
|---|---|---|
| P0-2 Quatre lectures attribuent l'arme sans la porte `publishable` | G4 | Les quatre sont des cumuls, catégorie explicitement exemptée par l'arbitrage (B) du DDL et par une décision datée du thought_log ; le fil per-kill est le seul à porter la porte, par décision |
| P0-3 `diag_weapons_v3 -write` unique écrivain de `match_objective_events` | G9 | `persist/bomb_stats_persister.go:243` écrit la table en INSERT pur sur le chemin post-sync vivant (`artifacts.go:352`) ; la commande de reproduction (`rg "WriteMatch\("`) ne pouvait pas trouver un autre écrivain ; `tablesProtegees` = tables append-only par contrat (celle-ci a une PK naturelle) ; `-write` = dry-run par défaut, « serveur arrêté » ; déjà classé au 06/08 |
| P0-6 Deux formateurs d'horloge sur le même écran | W2 | La sortie `Math.round` n'atteint aucun pixel (`title` d'un élément `pointer-events-none`, décision documentée `ReplayTimelineTracks.tsx:21`) ; toutes les horloges visibles passent par `formatClock` ; la doc de `_scoreCurve.ts:33-35` est exacte |
| A3 `match_weapon_shots` décodé et écrit pour zéro lecteur | G2 | Registre des reports l.24 : décision utilisateur ferme du 2026-09-01 nommant `film.weapon_shots` ; « passe supplémentaire » factuellement faux |
| A4 Même film décodé deux fois sous deux clés de fraîcheur | G2 | Doublon de P0-1 ; dette déjà écartée par G2 lui-même ; conséquence inversée par la mesure |
| B1 Agrégation `source_tag` écrite cinq fois, populations divergentes | G4 | Les cinq lectures rendent le même corpus |
| B2 Refus d'ambiguïté écrit six fois sur deux clés | G4 | Trois décisions sur trois clés que le schéma rend non fusionnables |
| B3 La Match View lit trois fois la même ligne | G4 | Deux requêtes spécifiques au fil s'ajoutent, en parallèle, à une requête générale |
| B4 `BuildKillPositions` reçoit la mauvaise grandeur d'horloge | G4 | La conversion est exacte et la doc décrit la bonne grandeur |
| B6 Conversion film ↔ match réécrite neuf fois | G4 | Trois fonctions à corriger si le calage change, pas neuf |
| C4 `analysis/replay` lit le disque, cache mutable, `service` fait de l'I/O | G3 | 2 des 7 fichiers antérieurs à la base (déjà vus le 06/08) ; I/O et état de process antérieurs ; préjudice de concurrence contredit par le mutex et la clé par répertoire ; volet `service/` écarté par le document lui-même |
| C5 File de jobs câblée `handlers → ops` sans port | G3 | Le handler est monté sur des stubs `httptest` par un test existant ; le seam-fonction est la norme du dépôt ; précédent antérieur |
| C6 Fusionneur crédit/film mal placé, seconde implémentation SQL | G3 | Thèse causale et remède faux (le cycle vient des 28 tests in-package de `persist`) ; le « miroir SQL » a été jugé P2 de contenu au 06/08 |
| F1 Le seul test sur le registre réel est éteint (chemin absolu) | G6 | `TestGoldenMiniBobine` décode le registre réel du même film `000d5950` (`chunk_00.bin` versionné, 435 Ko) inconditionnellement, dans le job `go-coverage` qui fait `exit $rc` ; un `archetypeBlockSlots` faux ne passe pas la CI. Résidu : aucune assertion nommée sur les indices d'archétype (le traitement « repointer `chunk00Dir` sur la mini-bobine, `t.Fatal` » reste bon) |
| F2 L'oracle Cheat Engine versionné n'est jamais consommé | G6 | `TestTI42WidthOracle` n'a aucune assertion de mesure (que des `t.Logf`) : dégardé, il ne peut pas échouer ; l'oracle est déclaré « ÉPUISÉ » au registre l.227 ; contradiction interne avec le P2 « 85 fichiers sans assertion » du même rapport |
| G2 L'oracle API de la fixture e2e n'est jamais confronté au décodage | G7 | `SlotIdentityFrom` compare les triplets film ↔ API et la CI asserte (`t.Fatal`) sur le résultat |
| G6 Le garde auth ADR 0023 ne balaie que la racine de `sync/` | G8 | `platform/auth/sentinel_test.go` fait `filepath.Walk` sur tout `apps/go-api` |
| H1 `cmd/vs-measure` à supprimer, zéro référence | G9 | Report assumé entré le jour de HEAD au `REGISTRE_REPORTS.md:12` avec condition de reprise et propriétaire ; G9 écarte `replay-equiv` pour le même motif à la ligne 13 |
| H4 `cmd/vehicle-sprite` mal classé au registre | G9 | Citation tronquée : la ligne dit « outil maintenu / à supprimer après extraction des assets », décision de l'auteur, « garder » en premier |
| H6 Douze commandes (20 308 L) sans référence | G9 | `cmd/weapon-sounds` a deux références web, que le tableau de G9 mentionne lui-même : onze commandes, 7 599 L (8 % de `cmd/`, pas 22 %) |
| J1 Le roster film × scoreboard est reconstruit six fois | W1 | Une implémentation unique ; les trois sites de la page sont mémoïsés sur des entrées stables (coût au chargement, pas divergence ni « par rendu ») |
| J4 `lib/replay/` factorisation abandonnée | W1 | Le dossier a reçu toute la logique de score pour laquelle il fut créé ; la frontière restante est une décision documentée (`lint-cross-feature-imports.mjs` l.82-99) tenue par un cliquet |
| L1 Aucune clé `film.*` servie au web | W4 | `GET /api/v1/titles/{slug}/capabilities` sert les cinq clés, sans auth, dans le contrat, déjà typé dans `generated.ts` ; il manque un client, pas un canal |
| L6 Colonnes d'Assaut gatées par `objective_stats` (API) | W4 | Porte fine `film.bomb_stats` appliquée côté Go, chaîne fermée, test dédié |
| L7 Cinq tables Halo Infinite en dur dans le TS du rejeu | W4 | Ce sont des règles de rendu (fichier `.wav` à jouer, paramètres Wwise générés, taille d'icône, hiérarchie votée, géométrie de montage), pas des libellés ; les manifestes ne portent aucun champ équivalent ; jointure par `weapon_key` canonique avec trois garde-rails contre les manifestes ; repli title-safe documenté |
| L9 « Heatmap des positions », dictionnaire inline hors garde | W4 | « heatmap » n'est pas dans `FORBIDDEN_PATTERNS` ; deux manifestes FR sous garde-rail (`explorer.toml:54`, `timeseries.toml:194`) le servent et passent ; la décision « carte de chaleur » est de portée rejeu (`i18nContract.ts:499`) |
| L10 « lobby » × 7 dans `usageI18n.ts` | W4 | Six occurrences (`${lobby}` est un jeton) ; « lobby » n'est pas interdit ; `engagement.toml` sert lui-même « lobby » en prose FR (`:211`, `:257`) ; décision de vocabulaire (escalade 6) |
| L11 Familles de mode en dur au lieu de `useAssetLabel` | W4 | Le kind `mode_family` n'existe dans aucun `assets.toml` ; les sept clés sont une énumération canonique inter-titres (`narrative/objective_participation.go:35`) traduite comme ses deux voisines |
| M2 `ZONE_ALPHA_ORDER` : test auto-validant | W3 | Le tableau agrège quatre constantes de production ; un réglage d'opacité remonté seul (accident daté 2026-08-25) rend le test rouge |
| N5 `xuidMeta`, `teamColor`, `teamSeriesColor` sans test de comportement | W5 | Les contre-épreuves existent dans d'autres dossiers |

### Écartés par le superviseur avant vérification (sélection ; la liste complète est dans chaque annexe)

| Constat | Motif |
|---|---|
| Existence de deux sources d'un kill (API crédit + film) et de deux tables | Décision d'architecture datée et justifiée (le film n'existe pas quand le match arrive) |
| Double décodage par cycle, relectures de chunks, orchestration ouvrier | Perf traitée par l'audit du 2026-09-02 |
| `replay_local_gate.go` sans date | Faux : bascule 2026-07-28, retrait cible 2026-11-08, critère mesurable non satisfait (R11 conforme) |
| Tables du film créées pour tous les titres | Position documentée (`migration/registry.go:40-47`) ; retenu seulement comme cause de D2 |
| Quatre producteurs d'artefacts = duplication | Faux : tous passent par `replaybuild.NewBuilder` |
| `backfill-team-scores`/`rounds` dupliquent ADR 0032 | Faux : ils appellent `RoundsDecideVariants()` |
| Goldens = snapshots du code | Dispositif prescrit par le plan master §6.1, régénération outillée, garde de forme |
| DDL de test recopiée dans `replayartifacts`/`killcollector`/persist | Non constaté : migrations réelles via `migration.RunForDB` |
| Tautologie i18n dans les tests web | Faux sur pièces (9 fichiers sur 156, jamais comme attendu creux) |
| `isAliveAt`/`trackWindow`/`positionAt` dupliqués | Faux : définis une fois, consommés par 14 modules |
| Mélange FR/EN des identifiants Go | Mesuré : 3 / 1 076 identifiants ; aucune règle écrite sur la langue des identifiants |
| État global de `filmdec` (118 vars) | Décision D10 gelée par ratchet daté ; seule la part morte est retenue (E2) |

## Points contrôlés sans constat

- Slug en dur dans le code neuf : aucun (0 occurrence Go et web) ; allowlists `archlint` inchangées.
- Chemins hors `PathResolver` : aucun nouveau ; 27 usages conformes.
- Écritures per-match hors `persist` sur les tables critiques : aucune ; les persisters neufs sont
  INSERT-only et relus par `_latest`.
- Fallback auth legacy dans le code neuf : aucun (sentinelle ADR 0023 sur tout le module).

## Ce qui est sain (à préserver dans la v2)

- Go : `bits_word.go` (un cœur, trois enveloppes documentées), `FilmContext` avec ratchet AST à
  double sens, `filmsource` source unique, `killsource` réutilisant la grammaire,
  `port/kill_source.go` (injection title-agnostic exemplaire), `domain/killscope`,
  `notify/replay.go`, `presence_service.go`, persisters neufs relus par `_latest` (dont
  `bomb_stats_persister` sur `match_objective_events`), `SlotIdentityFrom` (film confronté à
  l'API, asserté en CI), `TestGoldenMiniBobine` (registre réel + 10 lignes figées + contrôle
  négatif, inconditionnel), `steps_shared_kill_events_from_pairs_test.go` (mutation par mutation),
  `session_usage_aggregate_integration_test.go` (migrations réelles + vrai persister),
  `arbitration_clocks_utc_guard_mutation_test.go` (le seul vrai test du test),
  `seed_schema_enrollment_ratchet_test.go` (cliquet dérivé de la source), `sync_root_freeze_test.go`
  (égalité stricte), `platform/auth/sentinel_test.go` (balayage du module entier), les cinq clés
  `film.*` (gating propre des écritures, `film.bomb_stats` jusqu'aux colonnes), l'endpoint
  `GET /titles/{slug}/capabilities`, zéro slug en dur, `PathResolver` respecté partout, l'arbitrage
  écrit per-kill / cumuls sur `publishable`.
- Web : `replayNormalize`/`replayContract.test.ts` (contrat prouvé par `tsc`),
  `lib/replay/scoreTimeline.ts` (31 tests à oracle), `_scoreCurve`/`_scoreEvents`/
  `_killDistanceChart`/`valueGridModel`+`usageGrids`+`usageLogic` (logique pure hors `.tsx`,
  oracles à la main), `replayVideoEncoder.test.ts` (niveaux H.264 tirés de la norme),
  `killFeedLogic.test.ts` (valeurs écrites), `ZONE_ALPHA_ORDER` (test qui attrape un réglage
  remonté seul), 11 garde-rails dérivés d'une source d'autorité, `catalogLabel.ts` (libellés servis
  par l'artefact : le modèle), les cinq tables de rendu `hinf_*` (jointes par clé canonique,
  trois garde-rails contre les manifestes, repli documenté), `buildPlayers` unique et mémoïsé.

## Réponses aux cinq questions

1. **Architecture, ordonnancement, couches, allers-retours.** Les couches sont respectées dans le
   code de lecture (handlers, repos, services) ; les entorses vérifiées sont réelles mais de rang
   P2 (SQL et runner dans `api/wire` sur des fichiers neufs, `port.ReplayService` injecté dans
   quatre services, un import `service → games/halo_infinite`). Les allers-retours confirmés :
   un artefact à triple rôle qui a changé 43 fois de schéma sans jamais recuire le parc, des
   projections câblées sur la seule branche locale, un décodeur de kills dont la révision ne bouge
   pas (P0-1), un catalogue versionné réécrit par le runtime. Réfutés : « le film est décodé deux
   fois » (doublon de P0-1), « la Match View lit trois fois la même ligne », « le fusionneur est
   mal placé ».
2. **Rejeu 2D et fil « chacun de son côté ».** L'intuition tient côté horloge et côté calques :
   trois politiques « pas d'origine » (P0-5), deux horloges dans l'onglet Chronologie (P0-7), le
   recalage refait dans deux modules (P2), la micro-duplication prouvée à l'octet dans les calques
   (K1-K4). Elle ne tient pas côté roster (une implémentation, mémoïsée) ni côté chaîne kill Go
   (les représentations multiples sont des décisions écrites : film contre crédit, per-kill contre
   cumuls, trois clés). La page n'a pas de modèle joint : la route assemble deux sources dans
   12 `useMemo` sans test, et l'horloge n'a pas de foyer unique.
3. **Tests.** Pertinents et honnêtes dans leur masse (oracles à la main, migrations réelles, pas de
   mock auto-validant côté API). L'auto-validation avérée se réduit à deux cas P2 (échelle de la
   jauge de zone, `MatchCombatCtfOverlay`) ; les deux cas « graves » annoncés sont réfutés ou
   requalifiés. Les trous vérifiés : aucune assertion de valeur sur un document obtenu d'un film
   (G1), 26 familles de balayage sur ~34 sans octets réels en CI (F3), specs de rendu jamais jouées
   (M1), baseline de présence aveugle (G3), cron de purge sans test (G7), 260 L de logique de graphe
   sans oracle (N1), lint couleur hors CI (M6). Les filets réels que les auditeurs n'avaient pas
   vus : golden de la mini-bobine sur registre réel, `SlotIdentityFrom` film ↔ API.
4. **Portabilité à un futur titre.** Les écritures sont gatées par `film.*` (sain). Le décodeur
   vit sous la couche agnostique (D5, P1) ; deux sites servent le paquet Halo à tout titre déclarant
   `film.kill_source` (P2) ; un import `service → games/halo_infinite` (P2) ; sons sans slug (P2) ;
   les cinq tables `hinf_*` du TS sont des règles de rendu jointes par clé canonique (sain). Un
   second titre à film écrirait ses propres règles de rendu et devrait déplacer ou dupliquer le
   décodeur ; un second titre sans film reçoit aujourd'hui trois surfaces vides (L3-L5).
5. **Capabilities.** Tenu pour les cinq familles d'écriture et pour les colonnes d'Assaut
   (`film.bomb_stats`). Non tenu pour l'artefact (aucune clé, P2 tant qu'aucun lien ne mène à la
   page) et pour trois surfaces web rendues sur Halo 5 (P1) ; deux endpoints répondent 200 vide au
   lieu du 503 promis (P1). Le canal des clés fines existe côté API ; il manque
   `useDataCapability` côté web.

## Proposition de v2

Principe directeur : **un fait, un producteur, un stockage canonique, une horloge, un modèle joint,
une porte.** Chaque item renvoie au constat qui le fonde ; les items marqués « plan lot 3 »
reprennent des gestes déjà décidés par le projet (`PLAN_FINALISATION_REJEU_2D.md`). Les items
fondés sur un constat réfuté ont été retirés.

### V2-1 — Le pipeline film → faits (Go)
1. **Révision du décodeur** (P0-1) : bump immédiat ; garde-rail d'empreinte (hash du paquet
   `killsource` → la révision doit changer) ; une révision par famille de lignes (kills, tirs,
   positions).
2. **Les projections se déclenchent sur « un artefact vient d'être rangé »**, quel que soit le
   rangeur (`StoreBuildArtifact` de l'ouvrier ou `buildAll` local) (A1) : c'est le seul geste qui
   rend les statistiques d'Assaut et le bloc usages vivants en production au merge.
3. **Un composant de rattrapage** unique pour les quatre passes, avec un seul prédicat de fraîcheur
   (le digest : « artefact absent OU périmé OU sans dérivés ») (A2).
4. **`match_player_positions`** : projection de l'artefact par `persist` (comme
   `match_objective_events` l'est déjà pour l'Assaut) ou retrait de l'endpoint et du calque
   (escalade 1) ; enrôler `kill_positions` et `match_weapon_hit_distance` dans les gardes ART (G4).
5. **Séparer le document stocké du document servi** (C7, P2) : un DTO `domain/replaydoc` composé
   par le handler ; `analysis/replay.ReplayDocument` cesse d'être un schéma OpenAPI ; la version
   servie ne bouge que si le contrat change (escalade 2).
6. **Calques déclaratifs** : un fichier `layer_<nom>.go` déclare `{nom, requiert, produit, scan,
   document, coverage}` ; un ordonnanceur trivial fait le tri topologique ; `BuildFromFilm`,
   `document.go` et `coverage.go` ne bougent plus quand on ajoute un calque (plan lot 3.3).
7. **Catalogue de cartes** (A0) : `map_weapon_pads.json` n'est plus écrit par le runtime ; overlay
   non versionné (`reference/generated/…`) fusionné en lecture ; la promotion au catalogue reste un
   acte d'outil et de revue.

### V2-2 — Le décodeur (`filmdec`)
1. Un marcheur unique `walkDeltaBipedRecords(fc, visit)` pour les neuf boucles (E4) ; une table de
   domaines unique et un `readGuardedRef(dom)` ; `PacketHeadEventType` appelé par les six
   préambules, avec la valeur mesurée `dom3 = 7` comme seule vérité et un test qui confronte les
   deux tables recopiées à la table canonique (E3).
2. Supprimer le mort : `entity.go`/`entity_quant.go` (en gardant `bitLen`, `readQuantStat`,
   `quantStatDefaultWidth`), les 22 setters, la branche `if useLegacyAngularVel` et le drapeau
   `useBipedDefaultStateDeser` (le désérialiseur `consumeObjectAngularVelocity` reste : il est
   vivant pour `ti=40`), les deux crochets toujours nil et leurs 8 blocs de sauvegarde,
   `probe_export.go`, `frame_debug.go` ; redescendre le ratchet de variables en même temps
   (E1, E2, E6).
3. Généraliser le garde-rail du verrou (`world_object_precision_guard_test.go`) en ratchet AST :
   tout appelant de `Scan*` détient `LockProcessDecode` (E5) ; puis les réglages descendent dans
   `FilmContext` et le verrou devient inutile (D10 daté).
4. `FilmContext` entrée unique : `Scan<Quoi>(fc, opt) ([]<Quoi>, <Quoi>Stats, error)` ; les formes
   `film`, `dir` (lot 6, G5) et `payload` disparaissent (G1 P2).
5. Frontière : `filmdec` et `analysis/replay` descendent sous `games/halo_infinite/film/`
   (ADR 0012, D5 ; escalade 5) ; ce qui est réellement cross-titre (types du document servi,
   assemblage pur) reste sous `analysis/` ou `domain/`. Poser d'abord le ratchet
   « `analysis/replay` n'importe pas `games/{slug}` » (déjà vrai, à figer), migrer
   `vehicle_families` et `usageWallPanelIDs` vers `replay_labels.toml`.

### V2-3 — Les couches (items P2, à prendre lot par lot)
1. `port.BuildQueue` + `service/build_queue_service.go` : vide `api/wire` de son SQL et de son
   runner (C1).
2. `port.ReplayAvailability` remplace `port.ReplayService` dans les quatre services (C2).
3. `analysis.KillPairToleranceMS` exporté et migré aux quatre sites (B5) ; un helper pour les
   trois fonctions `paquetUS/1000 ∓ deathOffsetMS` (résidu B6).
4. Trois ratchets `archlint` : `service/ → games/{slug}/` (C3), champ `port.*Service` dans une
   struct de `service/` (C2), type `analysis/` en corps de route Huma (C7).
5. Retiré (constats réfutés) : la « représentation canonique du kill » en une requête, le
   déplacement du fusionneur crédit/film, le port de catalogue de cartes pour `analysis/replay`.

### V2-4 — Multi-titre et capabilities
1. Deux clés manquantes : `film.replay_artifact` (produire ET servir l'artefact) et une
   `Capability` title-level `replay` (plan lot 3.5, D1, L2). Gater `NewReplayHandler`,
   `replayartifacts.Run` (en tête, avant `enqueueAll`, D3), les deux `With*Repo` de positions et
   objective-events (D2 : un 503 réel ou le retrait), et la route web (escalade 3).
2. Côté web, consommer le canal qui existe : un client de `GET /titles/{slug}/capabilities`
   (déjà typé dans `generated.ts`), `useDataCapability('film.*')` jumeau de `useCapability`, un
   seul `FeatureGate` (L1 réfuté : il manque le client, pas le canal).
3. Règle à écrire : une surface alimentée par l'artefact ou une table `film.*` porte DEUX portes,
   capability (ce titre a-t-il un film ?) puis présence de donnée (ce match-là en a-t-il un ?). Un
   état vide sur un titre sans film est un bloc mort. Application immédiate à `MatchKillDistanceSection`
   (L3), `SessionUsageSection` (`unsupported` → `hidden`, L4), filtre et colonnes rejeu de
   l'Explorer (L5).
4. Les deux sites qui servent le paquet Halo à tout titre passent par `games.Resolver` (D4) ;
   `synthetic_title_b` déclare les clés `film.*` (`not_exposed` + un cas `supported`) pour prouver
   la dégradation ; `PathResolver.FilmCacheDir(titleSlug)` (G5 P2).
5. `soundUrlOf(stem, titleSlug)` avec garde-rail (L8). Les cinq tables de rendu `hinf_*` restent où
   elles sont : elles sont jointes par clé canonique et garde-raillées (L7 réfuté) ; un futur titre
   à rejeu écrira les siennes.

### V2-5 — La page de rejeu (web)
1. Un pipeline en quatre étages : `queries.ts` (inchangé) → `model/useReplayModel(doc, matchView,
   settings)` (le seul lieu de jointure : `clock` unique avec un seul repli (P0-5), `feed` recalé
   une fois (J2), `score`, `media`, `identity` ; remplace les 12 `useMemo` de la route ; testable
   sans React) → `layers/` sous un contrat unique `paint(ctx, frame, dpr)` (les 10 fonctions libres
   migrent, `draw` devient une boucle testable, M3) → `playbackStore` hors de l'arbre (le canvas
   écrit, tout le monde lit ; `ReplayTransport` et `ReplaySettingsDrawer` remontent frères)
   (plan lot 3.4).
2. `lib/replay/matchClock.ts` (P0-7) consommé par le fil ET par les graphes de la Match View ; les
   quatre modules que l'allowlist croisée dément (N4) descendent dans `lib/replay/` et l'allowlist
   porte sur des modules, pas sur une paire de features.
3. Cinq canoniques avant tout nouveau calque : `replayView.ts` (`CanvasView` + `projectTo`, K3),
   `buildLivesBySlot`/`lifeOfSlotAt` (K1), `useMatchSides(scoreboard)` (K2), `carriedGlyphPulse.ts`
   (K4), `covers`/`spansAt` (K5) ; un seul formateur d'horloge visible (résidu P0-6) ; chaque
   canonique sort avec son garde-rail le même jour (R6).
4. Suppressions : `replaySchemaLogic.ts` (J3), `roundAtFrame` (J5) ; les 11 faux hooks redeviennent
   des fonctions pures testées (W1 P2) ; le seuil R5 se mesure en lignes de code (`max-lines` eslint
   avec `skipComments`, escalade 4) et l'arborescence suit les responsabilités : `layers/`, `ui/`,
   `model/`, `hooks/`, `sound/`, `export/`, `settings/`, `i18n/`.
5. Match View : `_kdCumul.ts` et `_cadence.ts` extraits sur le patron de `_scoreCurve.ts` avec
   oracles (N1) ; `MatchCombatCtfOverlay.test.tsx` réécrit avec l'option entière et des abscisses
   attendues (N2) ; `formatDurationMShort` aux quatre sites (N3).

### V2-6 — Les tests
1. **Décodeur** : étendre le golden inconditionnel de la mini-bobine aux ~26 familles `Scan*` non
   couvertes (F3), en appelant les points d'entrée de famille et non les enveloppes ; y ajouter les
   assertions nommées d'archétype (`i0`/`i4`/`i11`/`i43-46`) et l'empreinte du registre (résidu
   F1) ; repointer `chunk00Dir` sur la mini-bobine versionnée avec `t.Fatal` ; un contrôle
   code ↔ `ecs_table.tsv` sur les 179 largeurs entières (F4).
2. **Rejeu** : une assertion de valeur sur le document obtenu par l'e2e `wire` (G1 : kills, score,
   `originMs` attendus écrits à la main) ; promouvoir `film_e2e/c0a82e88` en fixture partagée avec
   ses 49 digests ; poser une baseline de présence sur les paquets du chantier et brancher le
   script en CI (G3) ; tests de `RunOnce` du cron de purge (G7).
3. **Instruments** : les 85 fichiers sans assertion et les `*_research_test.go` sortent de
   `_test.go` (build tag `filminstruments` + garde-rail jumeau de `corpus_tag_test.go`, ou
   `cmd/filmprobe/`) ; corriger les 8 recettes de chemin (G6 P2).
4. **Garde-rails** : pour chaque liste manuelle, un cliquet qui dérive l'ensemble attendu de la
   source (vues `_latest` (G4), enveloppes `dir` (G5), hooks de porteur (M4), membres d'`InkVar`
   (M5)) ; un test du test par ratchet neuf (vecteur fautif + vecteur légitime, patron
   `arbitration_clocks_utc_guard_mutation_test.go`) ; égalité stricte pour les gels ; motifs
   ancrés sur un contexte plutôt que sur un mot nu (G8 P2).
5. **Web** : les deux specs de rasterisation dans le job `frontend` (M1) ; `draw()` extrait en
   `replayCompose.ts` et testé au contexte enregistreur (M3) ; `lint-no-hardcoded-colors` en CI et
   suppression des 9 copies (M6) ; ratchet de couverture sur `features/match-replay/**` (W3 P2).

### V2-7 — Les outils et les catalogues
1. `cmd/` en trois familles : le produit (`server`, `levelup`, `replay-worker`, `replay-build`,
   `replay-equiv`) repasse au régime normal du lint (H7) ; la fabrication (11 chaînes) sous
   `cmd/tools/` avec une section de `docs/COMMANDS.md` (sortie versionnée, prérequis, quand
   rejouer) (H3) ; les sondes sans référence et hors registre se suppriment (8 aujourd'hui, les
   autres à la clôture de leur question, selon le registre des reports) ou deviennent
   `levelup diag <x>`.
2. Le maillon `livraison.py` est porté en Go (`weapon-sounds livrer`) pour fermer la recette
   entièrement dans le dépôt (H2), comme `pck_dump` l'a fait pour `akpk_unpack.py`.
3. Les catalogues : une seule identité de carte (`mapref.Identity{MapID, Module, Names}`, un
   résolveur, une liste de suffixes de variante) ; `himap.CartesForge` devient une donnée JSON
   (I3) ; `heightfield.go` supprimé (I2) ; les cinq paquets Halo-only de la chaîne des cartes sous
   `games/halo_infinite/maps/` avec ratchet d'import (G10 P2) ; `corpus_tag_test.go` promu en
   `archlint` sur tout le module et les deux tests fautifs sous le tag (I1).
4. Sentinelle mémoire : les deux copies de `cmd/` passent sur `filmproc.Arm` (elles importent déjà
   le paquet) ; `sentinelleTokens` réduit au seul helper (H5).

## Ordre de marche proposé

1. **Lot 0 — avant le tag v7.5.0** (deux à trois jours) : les trois P0 (bump + garde-rail
   d'empreinte de `KillSourceDecoderRev` ; `replayClock` ; `matchClock` côté Match View) ; les deux
   items qui deviennent actifs au merge (overlay non versionné pour le catalogue, A0 ; projections
   sur « artefact rangé », A1) ; enrôlement gratuit des deux tables dans les gardes ART (G4).
2. **Lot 1 — capabilities** (V2-4) : deux clés, client web des clés fines, règle des deux portes
   sur les trois surfaces, 503 réel ou retrait pour les deux endpoints, `synthetic_title_b`.
3. **Lot 2 — flux et projections** (V2-1 items 3-5) : rattrapage unique, `match_player_positions`
   (après escalade 1), document servi séparé (après escalade 2).
4. **Lot 3 — modèle web** (V2-5) : `useReplayModel`, `matchClock`, contrat de calque,
   `playbackStore`, canoniques et suppressions.
5. **Lot 4 — tests et garde-rails** (V2-6), qui peut démarrer en parallèle des lots 2-3 pour
   sécuriser leurs refontes.
6. **Lot 5 — décodeur** (V2-2 items 1-4) : marcheur unique, préambule unique, code mort, verrou.
7. **Lot 6 — frontière de titre, couches et outils** (V2-2 item 5, V2-3, V2-7) : déplacements de
   paquets (après escalade 5), ports, `cmd/`.

Chaque lot se cadre sous `plan-review`, s'exécute sous `plan-execution` dans un worktree dédié, se
relit sous `adversarial-review`. Les constats P2 des auditeurs entrent au backlog du lot qui touche
leur fichier ; ils ne justifient pas de lot propre.

## Escalades utilisateur (décisions à trancher, aucune action entreprise)

1. `match_player_positions` (heatmap) n'a pour producteur que `diag_weapons_v3 -write` : projection
   de l'artefact par `persist` (comme `match_objective_events` pour l'Assaut) ou retrait de
   l'endpoint `/positions` et du calque ?
2. Séparer le document servi du document stocké implique un bump de contrat : maintenant (avant le
   tag v7.5.0) ou après ?
3. `film.replay_artifact` : refuser la cuisson pour un titre sans clé, ou seulement ne pas la
   servir ?
4. Le seuil R5 côté web : lignes brutes ou lignes de code ? (44 % de commentaires dans le rejeu.)
5. `analysis/filmdec` et `analysis/replay` sous `games/halo_infinite/film/` : déplacement pur
   (grosse diff, aucun changement de comportement) à planifier avant ou après la release.
6. Vocabulaire FR : « lobby » (six occurrences dans les usages de session, dénominateur « les
   8 ou 12 joueurs du match ») et « Heatmap » (servi par deux manifestes de l'Explorer et des
   séries temporelles) ne sont interdits nulle part ; les garder comme mots assimilés (les inscrire
   dans la doctrine du garde anti-anglicismes) ou les bannir (et corriger les manifestes existants) ?

## Suite

Ce registre devient le plan `.ai/PLAN_V2_REJEU_FILM_<date>.md` (à cadrer sous `plan-review`, lot
par lot, en commençant par le lot 0). Aucune modification de code dans cette session. Entrée
`.ai/thought_log.md` ajoutée. Rien n'est commité : registre, annexes et journal sont en attente de
la décision de l'utilisateur.

## Décisions de l'utilisateur (2026-09-05, après lecture du registre)

1. `match_player_positions` : projection de l'artefact par `persist` (patron
   `bomb_stats_persister`) ; le mode `-write` du diagnostic disparaît pour cette table.
2. Document servi séparé du document stocké : maintenant, avant le tag v7.5.0 (le grand nombre
   de versions pendant le développement est assumé, pas un défaut).
3. `film.replay_artifact` gouverne la production : pas de cuisson pour un titre sans clé ;
   l'affichage suit.
4. Seuil R5 côté web compté en lignes de code (`max-lines` avec `skipComments`), seuil 500
   inchangé.
5. Déplacement de `analysis/filmdec` et `analysis/replay` sous `games/halo_infinite/film/` après
   le merge de v7.5.0 ; inscrit dans Notion sous « Séquence à dérouler à la release, dans
   l'ordre : ».
6. « heatmap » banni partout (« carte de chaleur »), ajouté au garde anti-anglicismes ; « lobby »
   gardé comme mot assimilé et documenté dans la doctrine du garde.

Exécution : `.ai/PLAN_V2_REJEU_FILM_2026-09-05.md` (sept lots parallèles en worktrees dédiés,
revue adversariale en fin de tâche, intégration par le superviseur).
