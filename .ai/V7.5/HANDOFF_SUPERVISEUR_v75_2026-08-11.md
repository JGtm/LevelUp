# HANDOFF SUPERVISEUR — pipeline v7.5 (2026-08-11)

> Point d'entree unique du role SUPERVISEUR du chantier v7.5. Ecrit avant un auto-compact du
> contexte. Le document d'autorite reste le REGISTRE (`.ai/V7.5/REGISTRE_REPORTS.md`) pour les
> reports, et la memoire agent pour la methode. Ce fichier dit l'etat, ce qui reste, et comment
> on travaille.

## 0. ETAT EN UNE PHRASE

Release v7.5 en cours d'assemblage sur la branche unique **`feat/v75`** (HEAD `758abcf17`,
CI de branche 9/9 verte). Prod figee a `781daf0c6` (lots objectifs 1-3) jusqu'a UN SEUL merge
final = release + tag v7.5.0. Le lot **piste F (rejeu 2D)** est LIVRE (2026-08-12) : reste son
gate visuel utilisateur et son push.

## 1. MODE DE TRAVAIL (decisions utilisateur, non negociables)

- **BRANCHE UNIQUE `feat/v75`** (user 08/08) : tous les lots >=4 sont des SERIES DE COMMITS sur
  `feat/v75`. Cloture d'un lot = **CI DE BRANCHE verte au niveau JOB**, PAS de merge. Le
  superviseur re-merge `origin/main` aux frontieres. UN SEUL merge final vers main = deploiement
  prod (prevenir avant) + tag v7.5.0. Hotfix prod eventuel = `fix/*` vers main, absorbe ensuite.
- **Role superviseur** : piloter les sessions executeur (l'utilisateur les lance ailleurs et
  rapporte les CR), VERIFIER SUR PIECES, faire les commits/merges/git, tenir le registre et la
  memoire. NE PAS coder les features (exception : resolution de conflits d'integration au repli,
  regen de contrat, petits realignements — precedent etabli).
- **LECON PAYEE CHER (11/08)** : tout CR d'executeur qui dit « report / impossible / a refaire »
  sur un sujet deja travaille se VERIFIE SUR PIECES et se CHALLENGE avant d'etre transmis a
  l'utilisateur. Ne jamais relayer un pessimisme non verifie (cf.
  `memory/feedback_challenge_executor_cr_before_relaying.md`). Traduire, ne pas relayer le
  jargon d'executeur — l'utilisateur pense PRODUIT.
- **Prompt N+1 proactif** a chaque cloture ; le superviseur ecrit les prompts, PAS les executeurs.
- **Registre des reports** (`.ai/V7.5/REGISTRE_REPORTS.md`) : tout report y entre avec sa
  condition de reprise, a chaque cloture.

## 2. CE QUI EST LIVRE DANS `feat/v75` (verifie sur pieces)

| Lot | Etat | Note |
|---|---|---|
| 1-3 objectifs | MERGES + DEPLOYES PROD (781daf0c6) | dominance, tests, suppression objectivescore |
| 5 cartes Catalyst/Vagabond | sur feat/v75 | 21 fonds de carte + calage ; ORACLE containment dispo (Vagabond 3 zones) |
| Voie B (CTF) | repliee | correctif attribution des tirs du rejeu ; **rejeu OFF prod** (garde local) |
| Voie C (icones) | repliee + integration fiches livree | 168 PNG, cle = tag weap |
| Voie D (precision) | repliee | verdict NEGATIF definitif |
| **Icone arme du kill dans le kill feed** | LIVRE + VERIFIE (758abcf17) | pont offline `killicon`, `MatchKillFeed.tsx` DOM couleur team ; 0 trou d'arme, degradation OK ; cf. `memory/project_v75_icone_arme_killfeed.md` |

2 conflits semantiques d'integration attrapes par la CI de branche et resolus par le superviseur
(himap/extract.go ; helper de test `at`->`pointAt`). Contrat BridgeHealth regenere.

## 3. RESTE AVANT LE TAG v7.5.0

1. ~~**PISTE F — rejeu 2D**~~ **LIVREE le 2026-08-12** (plan et verdict :
   `.ai/V7.5/PLAN_PISTE_F_REJEU2D.md`). Ce qui a ete fait, et ce qui reste :
   - **F1 fond de carte** : servi par l'API (`GET .../replay/background` = calage,
     `.../replay/background.png` = image), dessine SOUS le rejeu A LA PLACE du sol
     reconstruit, repli propre quand la carte n'a pas de PNG. Resolution
     `match -> nom de carte -> module` par la chaine EXISTANTE (catalogue de bornes) ; aucune
     seconde regle de nommage. Oracle : 15/15 cartes du catalogue.
   - **F2 kill feed** : `ReplayKillFeed.tsx` sous la carte, synchronise a l'instant du rejeu.
     Il REUTILISE les briques du kill feed de la Match View (collecte des kills, cascade de
     couleur d'equipe, `WeaponIcon`) mais n'EST PAS `MatchKillFeed` : celui-la aligne tous
     les kills du match sur l'axe de categories d'un graphe ECharts (position calculee depuis
     `binIdx` + encart de grille), et il n'y a pas de graphe dans la page de rejeu. Le monter
     tel quel aurait affiche une frise figee sous un rejeu qui avance.
   - **DECOUVERTE QUI A CHANGE LE LOT** : les deux horloges different. Les events de la Match
     View sont recales sur le debut du GAMEPLAY, le film part du debut du MATCH. Mesure sur
     000d5950 (T0 = 18 465 ms) : ecart median **-0,6 s a offset nul** contre 3,1 s en ajoutant
     T0. `t0_ms` a donc ete expose au contrat (il ne l'etait pas) — sans quoi le feed aurait
     eu 18 s de retard.
   - Garde local **INCHANGE** : le rejeu reste OFF en prod.
   - **RESTE** : accord de commit/push utilisateur, CI de branche verte, **gate visuel sur
     000d5950 avec temoins nommes par l'utilisateur**. PIEGE : le worktree n'a pas les bases
     (12 Kio) et le kill feed depend de la Match View — lancer le serveur sur la racine de
     donnees du depot principal.

2. **COMPLETION DES CARTES** (requis user 11/08 AVANT le tag) : il manque des cartes au-dela des
   21 (Halo en a bien plus). Extraire leurs fonds depuis le jeu, comme le lot 5. A grouper avec
   « cartes v2 ».
3. **CONTAINMENT re-statue** : avec l'ORACLE Vagabond (lot 5) + `clockOffsets` elargi au-dela de
   -10 s. Le lot 4 avait rendu un rapport de faisabilite ; le re-mesurer maintenant que l'oracle
   existe. Registre : « oracle disponible ».
4. **CARTES v2** (dernier lot avant tag, decision user) : regle universelle de zone jouable
   (validee sur les 26 cartes par l'oracle des ancres, pas par carte), toits Chasm/Forbidden,
   zone jouable (149 % d'exces / 4 % de trous). + la completion des cartes (point 2).
5. **HYGIENE DE CLOTURE + TAG** : items du registre (loadGameVariant, steaktacular, litteral
   `film_chunks` en dur ~7 CLI, labels de mode non resolus, alias ` sentry defense`, doublon
   journal `teammates.02`), lot E `delivery-checklist`, re-shim TeammatesQueryRequest. Puis
   **merge unique vers main + tag v7.5.0** (= deploiement prod, PREVENIR avant + fenetre
   backfill : reprise SQL->SQL ~15 s + passe credit ~5 min par titre).

## 4. POST-v7.5 (registre, NE PAS faire dans la release)

- Backfill-killsource (avril-juillet 2026 a 0-5 %) + departage des 5 etiquettes ambigues du kill
  feed par loadout film (migration DB + re-decodage, disproportionne pour 68 kills).
- Recherche KOTH (Ghidra zones/possession), peuplement live objectifs, trajectoire armes lourdes.
- Rejeu 2D : activation prod (retrait du garde) = arbitrage utilisateur a l'hypercare (seuil
  d'activation a re-statuer, 87,39 % juste vs 88 % gonfle), peuplement d'artefacts (`film_chunks`
  VIDE -> re-cacher les films avant tout rebuild), calque objectifs dans le canvas.
- Cas-limite d'attribution des tirs du rejeu (limite d'information, OFF prod — NE PLUS creuser).

## 5. PIEGES MACHINE / CI (verifies)

- `himap` (tests cartes) depasse 60 min en local -> **cibler `go test` par `-run`**, la CI Linux
  tranche. Ne jamais tuber `go test` dans un `head` (faux vert).
- `research/**` NE declenche PAS la CI (globs) ; commit vide n'instancie pas ci.yml (paths-ignore)
  -> pour forcer la CI : PR vers main OU commit touchant un fichier hors `.ai/**.md`.
- Conflits `thought_log.md` = resolution en UNION par BASH octets bruts (PowerShell double-encode).
- Chaine CGO = winlibs (`PATH=/c/msys64/ucrt64/bin`, CGO_ENABLED=1) ; ne pas toucher a CC.
- Push depuis GIT BASH (hook shared-social-gate casse sous stub WSL).
- Contrat : regen = `go run ./cmd/openapi-gen` (openapi.yaml) + `make generate-types` (generated.ts).
- Flake CI connu : `Go Coverage + Baseline` peut rougir sur `sharedprovider` (B-swap) -> re-run.
- Nettoyage worktree : verifier containment (`git merge-base --is-ancestor`) AVANT suppression ;
  jonctions node_modules a retirer avant `worktree remove --force`.

## 6. ARTEFACT DE REJEU EXISTANT

`data/cache/replays/halo_infinite/000d5950.json` (2,19 Mo) — le SEUL artefact sur disque. Le
rejeu 2D est visible en local pour ce match tout de suite. Gate visuel de la piste F a faire
dessus.
