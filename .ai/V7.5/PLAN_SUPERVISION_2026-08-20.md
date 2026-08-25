# Plan de supervision — reprise v7.5 + registre-film — 2026-08-20

> Session de pilotage (contrat : skill `plan-execution`). Un agent par worktree frère
> `C:\Users\Guillaume\Downloads\Scripts\LevelUp-wt-<nom>` (l'ancien principal
> `C:\Users\Guillaume\Projects\LevelUp` N'EXISTE PLUS — le dépôt de travail est
> `Downloads\Scripts\LevelUp-go-migration`, branche `feat/v75`). Fusions `--no-ff` par le
> superviseur après CR vérifié sur pièces, push superviseur, CI verte au niveau JOB.
> Base de chaque lot : `feat/v75` (`ce9933ea5`). Schéma artefact : 18 (prochain libre 19).

## Décisions tranchées (ne pas re-décider)

- **D7 (arrondi de `p`) : CLOS — on GARDE 0,1°.** Un arrondi 0,5°/entier crée une classe
  « zéro exact » qui entre en collision avec la convention d'absence (`omitempty` = « le
  film ne porte pas p ») que le demi-pas de la formule garantissait vide à 0,1°. Le gain
  (~3-4 % de taille) ne paie pas la redéfinition de la convention + goldens + risque.
- **R3-1 (durée de la croix de mort) : 2,5 s** (actuelle 1,5 s trop brève — l'utilisateur
  ne la voit pas ; 4 s surchargerait les mêlées denses). À poser dans le lot UI.
- **R3-2 (rendu écran de dissimulation) : BLOQUÉ** tant que `0x4396db42` n'est pas
  identifié (artefact utilisateur en cours — vérif Theater).
- **variant-probe : SUPPRESSION** (session « sonde variants » close, voie API fermée,
  0 appelant, précédent `tmp_filmmanifest` ; git garde l'historique) + retrait des 2
  allowlists datées.
- **Garde Fiesta des lâchers : devient un réglage** (le réglage `showDroppedPlacements`
  existant suffit, visible dans tous les modes, défaut ON). La garde dure masquait 26
  lâchers réels sur `000d5950` — c'est le symptôme rapporté par l'utilisateur.
- **Murs portatifs : durée nominale à l'écran** (précédent `SENSOR_DURATION_MS=15000`
  officiel) — le film ne date AUCUN despawn (`t1` = fin de suivi, médiane 0,6 s), donc
  fenêtre = `t0 + durée officielle` (« une dizaine de secondes », citation Waypoint déjà
  dans `equipment_life_end_test.go` — l'exécutant vérifie la valeur exacte citée).
- **Pions gris en fin de partie : SUPPRIMÉS** (demande utilisateur).
- **Killfeed sous le graphe Dominance (match view) : SUPPRIMÉ** (demande utilisateur ;
  le fil qui défile DANS le rejeu reste).
- **Score : bandeau au-dessus du canvas, tous modes** — spec utilisateur :
  `[barre progression couleur équipe, score écrit dedans] [temps courant] [barre équipe 2]`.
- **Dettes laissées en l'état (justifié)** : slots tag 3 `1545/1547` (impact produit nul),
  silences intra-rampe Bastion (seule lecture honnête des données ; à surveiller au gate).
- **Re-cuisson** : témoins d'abord (gate visuel), masse (954 manifests, ~8 h) EN DERNIER,
  serveur arrêté, en arrière-plan.

## Lots (ordre, worktree, gates)

| # | Lot | Worktree / branche | Statut |
|---|---|---|---|
| L1 | Hygiène CI + dettes (gitleaks FP, variant-probe, ZoneStateNow, repo-root ×2, registre 177/182/201/374/375) | `LevelUp-wt-hygiene` / `wt/hygiene-ci` | [ ] lancé 20/08 |
| L2 | Re-tirage UGC 5 cartes KOTH sans colline (Forbidden, Illusion, Oasis, Oasis FF, Empyrean) + catalogue rôle `hill` | `LevelUp-wt-koth-ugc` / `wt/koth-ugc` | [ ] lancé 20/08 |
| L3 | Nettoyage UI rejeu/match-view (killfeed Dominance OUT, pions gris OUT, fiches compactes/heatmap revert+réactivité+état mort, calques traînées/morts orientés, croix 2,5 s, murs durée nominale, garde Fiesta→réglage) | `LevelUp-wt-ui` / `wt/ui-nettoyage` | [ ] attend rapport E1 |
| L4 | Bandeau score au-dessus du canvas (tous modes, scoreTimeline) | `LevelUp-wt-score` / `wt/score-bandeau` | [ ] attend E1, après L3 |
| L5 | Callouts : suivre la langue de l'app | selon E1 (web et/ou Go) | [ ] attend E1 |
| L6 | Sons balise : balayage structurel 1305 sbnk (`-banks all` à ajouter à weapon-sounds) + page v3 — **décodeur vgmstream absent : demander à l'utilisateur avant tout téléchargement** | `LevelUp-wt-sons` / `wt/sons-balise` | [ ] partiel possible sans décodeur |
| L7 | Artefact utilisateur « équipements inconnus » (`0x4396db42` ×104 / 9 matchs, dates+cartes+timings+joueurs) | superviseur (Artifact) | [ ] données prêtes (E2) |
| L8 | Oddball crâne : investigation i3 + oracle score-par-seconde (réponse : le film ne réplique PAS le crâne comme le drapeau — 0 porteur/26 images, aucun compteur statborg) | `LevelUp-wt-oddball` / `wt/oddball-crane` | [ ] après L3/L4 |
| L9 | Portabilité VPS : correctifs selon rapport E3 | selon E3 | [ ] attend E3 |
| L10 | Re-cuisson témoins + gate visuel en app (Z-zones, équipements, socles, fil des morts, drapeau `64e8adfa`, + vérif des fixes L3/L4) | superviseur (browser) | [ ] après fusions |
| L11 | Backfill masse : `backfill-replay --only-existing` + `backfill-killsource` (serveur arrêté, arrière-plan) | superviseur | [ ] dernier |
| L12 | Docs de clôture : thought_log, registre (D7 clos, décisions), handoffs, ce plan | superviseur | [ ] fin |

Gates communs à tout lot code : Go = `go build ./...`, `go vet ./...`, tests des paquets
touchés, `golangci-lint` (CGO) ; web = `npx tsc -b --force` NU (jamais pipé), `vitest run`
ciblé, eslint = baseline. Machine : une commande `go` à la fois par worktree, `GOCACHE`
propre au worktree, jamais de boucle sur le corpus de films.

## File d'attente (non oublié, non lancé cette session sauf capacité)

- Armes lâchées au sol (aucun canal « telle arme au sol en (x,y) ») — file données #1.
- 26 naissances de drapeau sans porteur → `flagObjects` (instrument en place).
- Nommage himap : 3 type_id socles, `0x4396db42` (si Theater ne tranche pas), `parcel_mp_weapon_manager.lua`.
- Translocateur état actif ; heatmap « morts subies » (question A8 sans réponse).
- A0-bis (zones Bastion pas toutes neutres — PLAN_CLOTURE_V75), fonds Forge refusés (33), hygiène P2 registre-film (RealRounds, OriginResolved *bool, components_hooks_test 600 L).
- Migration repo-root des AUTRES tests si l'allowlist en garde (hors les 2 traités en L1).

## Découvertes (notées, non traitées)

- (E5) `vgmstream-cli` et les outils de mix/page (`_outils/*.py`, machine précédente) ont
  disparu avec `C:\Users\Guillaume\Projects\` — la recette d'écoute doit être reconstruite.
- (E2) 224 lâchers (grenades/grapple/thruster) ne sont jamais dessinés (règle générale du
  calque, indépendante de Fiesta) — comportement assumé, pas un bug.
- (E4) La commande de reproduction du registre :201 était un faux vert (`\|` littéral RE2).

## Journal

- 2026-08-20 — Plan écrit après 5 explorations (UI/équipements/portabilité/dettes/sons —
  3 rapports reçus, E1 UI et E3 portabilité en vol). Décisions ci-dessus tranchées.
  L1+L2 lancés en worktrees frères.

- 2026-08-25 — Journee 2 : ouvrier distant FUSIONNE (2 rondes, P0 ronde 2 corrige au point d'ecriture) ; A1 CLOS sur mesures (repulseur+propulseur, 5 voies) ; grappin sonore livre ; catalogue REGENERE (133->273 collines, +302 TC, +73 Firefight, garde anti-rack repare) ; blindage cuissons valide sur crash reel ; passe only-existing complete (34/35 schema 18). Sons balise : voies jeu epuisees -> decoupes video publiees (designation utilisateur attendue). Worktrees session tous nettoyes. Restent (utilisateur) : designations decoupes, Ghidra compagnon, Theater 0x4396db42, gate visuel, arbitrage garde local ; (declencheurs) : masse, clockOffsets, profiling bombe, remediation cache.
