# PLAN — Temps forts d'un match : détection, puis clips vidéo en 3D (2026-11-15)

Statut : **rédigé le 2026-11-15 ; s'exécute APRÈS la clôture de `.ai/PLAN_REJEU_3D_2026-11-15.md`
(demande du user) ; D-1 à D-6 tranchés le 2026-11-15 (« ok bien ») ; D-7 (rangement des temps forts détectés) à confirmer au go.
Exécution en pistes parallèles (section 1 bis) : Sonnet par défaut, Haiku, Opus si nécessaire, Fable pour l'étape 0 et la revue finale.**
Contrat d'exécution : skill `plan-execution` (ordre strict dans chaque piste, un item clos et vérifié avant le suivant,
aucune case vide à la clôture, zéro fix hors périmètre, découvertes en section 8). Grille relue : skill
`plan-review`.

**Amendé le 2026-11-17** par le plan maître `.ai/PLAN_MAITRE_8_3_0_2026-11-17.md` (ordre entre plans, fichiers
partagés, borne de 4 agents d'écriture commune aux quatre plans, worktrees `C:/t/tf-<piste>` sans jonction).
Ce plan garde son propre go. Points portés ici :
- les maquettes B.1 et D.1 se dessinent DANS les cadres validés de la refonte du rejeu (Standard, Cinéma,
  Plein cadre) ; les repères de la barre de lecture y figurent dans les DEUX barres (celle du Standard et la
  barre d'une ligne) ;
- B.3 écrit `ui/ReplayTransport.tsx` après l'item E.2 de la refonte ;
- la table des médailles du film est montée : `release/v8.3.0` porte le décodeur v1.4.1 ;
- un besoin de changer le décodeur ou le document de rejeu revient au user avant tout code.

Constats relus le 2026-11-15 sur `origin/release/v8.2.0` (`0ccbed4e2c`) et sur le projet csstat
(`<poste>\Downloads\Scripts\csstat`, `docs/highlights-pipeline.md`). L'exécutant les RE-VÉRIFIE
sur sa base, `release/v8.3.0` (le user a sorti la 3D et les temps forts de la 8.2.0 pour la 8.3.0 le
2026-11-15) : le plan 3D aura changé le rejeu entre-temps.

---

## 0. Objectif et critère de succès

Idée venue du projet csstat d'un ami du user (CS2, `madina972/csstat`). Il trouve les meilleurs moments d'un
match, les note, calcule les bornes du clip, puis le filme en pilotant le jeu et ffmpeg. Le user
(2026-11-15) :
- la détection ne doit pas reposer sur les seules médailles, même si elles couvrent l'essentiel ;
- sans la 3D, des temps forts en 2D ne valent pas la peine d'être regardés ;
- l'aboutissement est le clip vidéo rendu par notre lecteur 3D ;
- le plan s'exécute après la 3D.

Rien n'est repris du code de csstat : seule l'idée et la démarche (noter, regrouper, borner, fusionner)
sont reprises, avec crédit dans l'ADR et la note de version.

| Lot | Ce que le user verra | Critère de succès |
|---|---|---|
| 0 | Une page de vote sur des moments candidats | Vérité terrain écrite (votes du user) ; mesure « médailles seules » contre « médailles et contexte » sur le même jeu |
| A | Rien à l'écran | Détecteur pur testé, qui atteint le seuil de D-2 sur la vérité terrain ; temps forts rangés par match |
| B | Temps forts listés sur la page match et repères dans la barre de lecture du rejeu 3D | Maquette validée AVANT code ; sauts exacts au début du moment ; rien hors bureau sans 3D (D-4) |
| C | « Créer le clip » sur un temps fort, clip rangé dans Médias | Rendu par le lecteur 3D, encodage par le ffmpeg livré, fichier lu sans erreur, durée = bornes ± 1 image ; file bornée, jamais de rendu en lot sans accord |
| D | Page « Temps forts » sur plusieurs matchs (joueur, session, escouade) | Lecture par périmètre (ADR 0036) ; état vide conforme |

## 1. Décisions à trancher AVANT exécution

- **D-1 — signaux de détection.** **Tranché (user, 2026-11-15, « ok bien ») : proposition retenue.** Proposition : les médailles plus le contexte que le document de rejeu
  porte déjà.
  - **Médailles** : avec leur instant, comme le font `highlight_events` et, une fois le décodeur monté, la
    table des 157 médailles lues dans le film.
  - **État du joueur à l'instant de chaque élimination** : vie et bouclier (`Point.sh`, `Point.hp`). Une
    élimination à presque rien de vie vaut plus.
  - **Rapport de force** : nombre d'adversaires vivants autour (pistes et vies). Les situations à 1 contre
    plusieurs se lisent sans médaille.
  - **Séries proches du seuil** : des éliminations rapprochées que l'intervalle des médailles ne couvre pas.
  - **Distance et arme** : distance du tir (positions) et arme ou cause de la mort (`killsource`).
  - **Objectifs** : capture de drapeau après un long portage (`FlagCarries`), zone prise ou tenue sous
    pression (`ZoneStates`), bombe (`BombArmings`), crâne (`SkullCarries`).
  - **Moment du match** : score serré, dernière minute, prolongation, élimination ou point décisif
    (`ScoreTimeline`, `comeback.go`).
  - **Équipement et véhicules** : élimination au grappin, au répulseur, en véhicule ou contre un véhicule
    détruit.
  - **Densité d'action** : pic d'intensité (`analysis/narrative/intensity.go`, à réutiliser, pas à recopier).

  Chaque signal entre au modèle seulement si le lot 0 montre qu'il améliore le résultat sur la vérité
  terrain.
- **D-2 — seuil de qualité.** **Tranché (user, 2026-11-15, « ok bien ») : proposition retenue.** Proposition : parmi les moments que le détecteur classe dans ses 3 premiers
  par match, au moins 70 % sont votés « bon » par le user. Sur les moments votés « bon », au moins 80 % sont
  retrouvés.
- **D-3 — vérité terrain.** **Tranché (user, 2026-11-15, « ok bien ») : proposition retenue.** Proposition : une page de vote (artefact `verifs`). Pour 12 matchs choisis par le
  user, elle montre environ 10 candidats par match, issus d'un détecteur large, que le user vote « bon »,
  « moyen » ou « nul ». Il peut aussi ajouter un moment oublié par son instant de chrono. Les votes sont
  versionnés en annexe.
- **D-4 — où les temps forts apparaissent.** **Tranché (user, 2026-11-15, « ok bien ») : proposition retenue.** Proposition, suivant le user (« sans 3D pas fou à regarder ») :
  au bureau avec la 3D active seulement. Il n'y a ni liste ni repère en 2D, et le site ne change pas. La
  détection tourne quand même partout où le rejeu existe, pour que les clips et la page D se remplissent dès
  que la 3D est activée.
- **D-5 — rendu du clip.** **Tranché (user, 2026-11-15, « ok bien ») : proposition retenue.** Proposition :
  - à la demande seulement (« Créer le clip »), une file d'un rendu à la fois, sans rendu automatique de tous
    les temps forts (règle des cuissons en lot) ;
  - caméra au choix : 1re personne du joueur, 3e personne, ou caméra de suivi cinématique ;
  - HUD selon la décision D-10 du plan 3D ; sons du rejeu inclus ;
  - encodage par le ffmpeg livré avec l'app de bureau (ADR 0045), H.264 et AAC en MP4, lu partout.

  La façon de rendre (fenêtre cachée, pas à pas image par image) se reprend de l'export du lecteur du
  contributeur si son source la porte. Sinon, elle est mesurée au lot C, item C.1.
- **D-6 — rangement des clips.** **Tranché (user, 2026-11-15, « ok bien ») : proposition retenue.** Proposition : un clip rendu devient un média de la bibliothèque existante
  (ADR 0046 : identité = empreinte du contenu), rattaché au match. Il apparaît donc dans Médias avec les
  captures du user, marqué « clip généré ». Les clips ne comptent pas dans le plafond du cache 3D (D-11 du
  plan 3D) : ce sont des fichiers voulus par le user, jamais supprimés automatiquement.
- **D-7 — rangement des temps forts détectés.** Proposition : un fichier par match à côté de l'artefact de
  rejeu (`PathResolver.ReplayArtifactsDir`), recalculé depuis le rejeu sans relire le film, avec une révision
  du détecteur (changement de révision = recalcul au démarrage, comme l'ADR 0047). Il n'y a aucune écriture
  par match dans une base partagée, donc aucune exposition au bug ART. La page D lit par périmètre, avec un
  cache invalidé au sync (ADR 0036).

## 1 bis. Orchestration : pistes parallèles et modèles

Demande du user (2026-11-15) : un plan pensé pour la parallélisation, exécuté par **Sonnet** principalement,
**Haiku** dès que possible, **Opus** si nécessaire, **Fable** pour l'étape 0 et la revue finale.

Les règles d'exécution sont celles de la section 1 bis du plan 3D : pistes, ordre strict dans une piste,
dépendances écrites seulement, un agent par worktree (`LevelUp-wt-tf-<piste>`, sans jonction vers `data/`),
agents sans commit ni push, au plus 4 agents Sonnet ou Opus d'écriture à la fois, un seul rendu de clip
à la fois sur la machine, `generated.ts` régénéré par le pilote, `model` explicite.

| Piste | Item | Modèle | Attend | Note |
|---|---|---|---|---|
| T0 Étape 0 | 0.0 conception du détecteur | **Fable** | go | lecture seule : signaux lisibles dans le document de rejeu, analyses existantes à réutiliser (`go-features`), démarche csstat, coût du calcul par match et du rattrapage. Rendu : `annexes_temps_forts/conception.md` |
| T0 | 0.1 détecteur large jetable | Sonnet | 0.0 | |
| T0 | 0.2 page de vote | Sonnet | 0.1 | le user choisit les 12 matchs et vote |
| T0 | 0.2 votes recopiés en annexe | Haiku | votes du user | |
| T0 | 0.3 mesure M-1, poids et seuils proposés | **Fable** | 0.2 | puis choix du user, Gate 0 |
| T1 Maquettes | B.1 + D.1 une seule page de maquettes | Sonnet | go | aucun code ; choix du user |
| T2 Rendu | C.1 mesure du rendu d'un clip | Sonnet | go | l'agent prépare, le user lance |
| T3 Détecteur | A.4 capacité `highlights` | Haiku | go | modèle existant |
| T3 | A.1 + A.2 détecteur et tests | Sonnet ; **Opus** si le seuil D-2 n'est pas tenu après une passe | Gate 0 | |
| T3 | A.3 fichier par match, rattrapage | Sonnet | A.1 | puis Gate A |
| T4 Rejeu et page match | B.2 route | Sonnet | A.3 | |
| T4 | B.3 web | Sonnet | B.1 validé, B.2 | puis Gate B |
| T2 | C.2 rendu image par image et encodage | Sonnet ; **Opus** si C.1 montre un défaut de cadence ou de synchronisation du son | C.1, Gate A | |
| T2 | C.3 + C.4 média et tests | Sonnet | C.2 | puis Gate C |
| T5 Plusieurs matchs | D.2 lecture par périmètre | Sonnet | A.3 | |
| T5 | D.3 web | Sonnet | D.1 validé, D.2, C.3 | puis Gate D |
| T6 Clôture | R.1 revue finale | **Fable** | Gates A à D | voir section 7 bis |
| T6 | R.2 corrections retenues | Sonnet ; Opus si l'écart est un algorithme | R.1 | une seule ronde |

Vagues : dès le go, T0 (0.0), T1, T2 (C.1) et T3 (A.4) ; après Gate 0, T3 ; après Gate A, T4, T5 et C.2 en
parallèle ; à la fin, R.1, R.2 et la clôture.

## 2. Ce qui existe (relu sur pièces)

- **Événements datés.** `port/highlight_events.go` (`HighlightEventFilters` : kills, morts, médailles avec
  `time_ms`, garde-fou contre le balayage complet).
- **Document de rejeu** (`domain/replaydoc/document.go`) : `Tracks` (positions, vie et bouclier, visée),
  `Shots`, `Projectiles`, `Vehicles`, `EquipmentEpisodes`, `GrappleLines`, `Objectives`, `ScoreTimeline`,
  `FlagCarries`, `ZoneStates`, `BombArmings`, `SkullCarries`, `VipCrown`, `NeutralDeaths`, `Roster`.
- **Analyses réutilisables.** `analysis/narrative/intensity.go`, `analysis/comeback.go`,
  `analysis/comeback_objective.go`, `analysis/first_blood.go`, `analysis/max_killing_spree.go`,
  `analysis/match_impact.go`. Le skill `go-features` est à relire avant de coder.
- **Médailles du film.** Table des 157 médailles dans le décodeur (non tagué au 2026-11-15) ; montée dans
  LevelUp à vérifier avant le lot A.
- **Après le plan 3D** : lecteur 3D, caméras, HUD, sons ; bibliothèque Médias (ADR 0046) ; ffmpeg livré
  (ADR 0045).
- **csstat, pour la démarche** : note par élimination, regroupement par fenêtre variable, coupures nettes
  (mort du joueur, changement de manche), bornes avant et après adaptées au type de moment, fusion des
  segments proches de même type et de note suffisante, chevauchements tranchés au profit de la meilleure
  note.

## 3. Lot 0 — vérité terrain et mesure

- [ ] 0.0 Conception du détecteur par un agent **Fable** (`claude-fable-5-2`), en lecture seule. Il dit
      quels signaux de D-1 le document de rejeu permet de lire, et comment. Il dit aussi quelles analyses
      existantes réutiliser, quel calcul coûte quoi par match, et comment le rattrapage se parallélise sans
      dépasser la mémoire. Rendu : `.ai/annexes_temps_forts/conception.md`.
- [ ] 0.1 Détecteur large jetable (tous les candidats plausibles, sans seuil), rangé en annexe.
- [ ] 0.2 Page de vote (D-3) publiée ; 12 matchs choisis par le user ; votes recopiés dans
      `.ai/annexes_temps_forts/verite_terrain.json`.
- [ ] 0.3 Mesure M-1, par un agent **Fable** : médailles seules contre médailles et chaque signal de D-1, ajouté un par un, sur la
      vérité terrain. Les signaux retenus sont ceux qui améliorent le résultat ; les autres sont écartés et
      notés en annexe.
- **Gate 0 :** votes versionnés, table M-1 écrite, liste des signaux retenus validée par le user.

## 4. Lot A — détecteur

- [ ] A.1 Paquet pur `internal/analysis/highlights/` : note par moment (signaux retenus en 0.3, poids
      nommés), regroupement, coupures, bornes, fusion, tri. Aucune constante magique : poids et seuils dans
      une table nommée, testée.
- [ ] A.2 Tests sur la vérité terrain (seuil D-2), plus des cas construits : mort au milieu d'une série,
      fin de manche, prolongation, joueur parti.
- [ ] A.3 Fichier par match et révision du détecteur (D-7) ; calcul après chaque cuisson de rejeu et rattrapage
      au démarrage, joueur par joueur, journalisé (`slog.InfoContext`, compte et durée).
- [ ] A.4 Capacité de titre `highlights` (`capabilities.toml` ; Halo 5 : absente), avec
      `ErrCapabilityNotSupported` en dégradation.
- **Gate A :** `go test ./internal/analysis/highlights/... ./internal/service/...`, seuil D-2 tenu.

## 5. Lot B — temps forts dans le rejeu 3D et la page match

- [ ] B.1 Maquette : liste de la page match, repères dans la barre de lecture et panneau du lecteur 3D.
      Choix du user AVANT code.
- [ ] B.2 Route `GET /players/{slug}/matches/{id}/highlights` (libellés résolus dans la langue de la
      requête, catalogue `i18n.Msg`), handler mince.
- [ ] B.3 Web selon la maquette ; saut au début du moment, caméra sur le joueur du moment ; textes fr et en ;
      clés de requête dans `lib/query/keys.ts` ; visible seulement selon D-4.
- **Gate B :** `make generate-types`, `make check-types`, `make test-web`, vérification à l'écran par le
  user dans `verifs`, en lecture.

## 6. Lot C — clips vidéo

- [ ] C.1 Mesure du rendu sur le poste du user : un clip de 15 s en 1080p à 60 i/s, temps de rendu, mémoire,
      taille du fichier.
- [ ] C.2 Rendu image par image depuis le lecteur 3D (horloge pilotée, pas le temps réel), sons du rejeu
      mixés, encodage par ffmpeg (D-5). File d'un rendu à la fois, annulable, reprise propre après un arrêt.
- [ ] C.3 Clip rangé comme média (D-6), rattaché au match, marqué « clip généré » ; ouvrir le dossier,
      supprimer avec la Corbeille (règles de l'ADR 0046).
- [ ] C.4 Tests : bornes du clip, file, annulation, échec d'encodage journalisé, média créé une seule fois.
- **Gate C :** tests verts, 3 clips rendus et lus par le user, durées conformes.

## 7. Lot D — page « Temps forts » sur plusieurs matchs

- [ ] D.1 Maquette (joueur, session, escouade ; règle des nouveaux indicateurs multi-matchs du user :
      sessions, séries temporelles, escouade). Choix du user AVANT code.
- [ ] D.2 Lecture par périmètre et cache invalidé au sync (ADR 0036) ; état vide à bloc affiché.
- [ ] D.3 Web selon la maquette, avec « Créer le clip » depuis la liste.
- **Gate D :** `make gate-push`, vérification à l'écran par le user.

## 7 bis. Revue finale (piste T6)

- [ ] R.1 Revue finale par un agent **Fable**, en lecture seule, suivant le skill `adversarial-review`, sur
      tout le diff du plan. Axes :
      - justesse du détecteur sur la vérité terrain ;
      - aucune écriture par match dans une base partagée ;
      - révision du détecteur et rattrapage ;
      - file de rendu, annulation, reprise ;
      - médias créés une seule fois, jamais supprimés automatiquement ;
      - rien hors bureau sans 3D ;
      - textes fr et en.
      Rendu : `annexes_temps_forts/revue_finale.md`, chaque constat avec un fichier, une ligne et un
      scénario d'échec.
- [ ] R.2 Constats vérifiés sur pièces par le pilote. Ceux retenus sont corrigés par Sonnet (Opus pour un
      algorithme), avec un test qui échoue avant. Une seule ronde.
- **Gate R :** chaque constat retenu et corrigé, ou écarté avec motif ; `make gate-push` vert.

Clôture : entrée `.ai/thought_log.md`, report des `[!]` dans `.ai/REGISTRE_REPORTS.md`, ADR « temps forts :
détection et clips » (EN seul) avec crédit à csstat pour la démarche, note de version.

## 8. Découvertes

(à remplir pendant l'exécution ; rien n'y est traité dans ce plan)

- 2026-11-18 — signal candidat pour le détecteur, à mesurer en 0.1 et 0.3 comme les autres (remarque du user,
  pas une décision : « pas mal pour les highlights de s'appuyer sur le score ! Les petites actions ne vont pas
  rapporter beaucoup mais les points notables si »). Le document de rejeu publie chaque action de score d'un
  joueur avec son nom et ses points (`scoreActions[]` : capture et vol du drapeau, zone capturée, contrôle de
  la colline, destruction de véhicule, aide par IEM, élimination du porteur...). Les temps forts « mode » du
  film en sont l'écho (mesure : `.ai/V8.3/annexes_decodeur_1118/temps_forts_mode.md`). Les points de l'action
  donnent un poids lu dans le film, sans table à écrire : une action à gros points est notable, une petite
  non. Limite connue : aucune action de score publiée en Total Control.

## 9. Reprise de session

Section 1, puis la table de la section 1 bis : pour chaque piste, première case vide des sections 3 à 7 bis
dont la colonne « Attend » est close ; `git worktree list` et `ListAgents` ; `.ai/thought_log.md` (entrées
« temps forts »).
