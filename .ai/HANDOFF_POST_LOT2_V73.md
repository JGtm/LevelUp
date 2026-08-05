# HANDOFF — après le lot 2 v7.3 (« petites choses »), deps et passe découvertes

> Écrit le 2026-08-03 en clôture de la campagne. Point d'entrée pour toute session
> future : lire ce fichier, puis `.ai/V7.3/PLAN_V73_LOT2_PETITES_CHOSES.md`
> (INTÉGRALEMENT statué — les statuts font foi) et les entrées thought_log du
> 2026-08-02/03. Doctrine de pilotage multi-agents : mémoire agent
> `orchestration-opus-lots` (hors repo).

## État au moment de l'écriture

**Sur `main`, déployé ou en cours de déploiement (ordre chronologique du 03/08) :**

| Merge | Contenu | CI/Deploy |
|---|---|---|
| `f8913a473` | Lot 2 complet (16 items, détail au plan) | CI + deploy VERTS, conteneurs healthy |
| `ef165518b` | echarts 6.1.0 (CVE-2026-45249 éteinte) + harnais visuel 105 captures | CI + deploy VERTS |
| `fad59e9b1` | docs : plan D3 statué | — |
| `69785cecc` | Passe découvertes : 9 quick wins, 3 garde-rails | CI lancée, verdict non rendu à l'écriture |
| squash #74 | npm-minor-patch (7 paquets dev dont Playwright 1.62.1) | CI + deploy EN COURS à l'écriture |

Si un rouge est apparu sur les deux derniers runs après cette écriture : la cause
probable est une interaction lockfile/baseline triviale — corriger en avant, ne pas
revert (précédent : tout rouge de la campagne a été une cause annexe, jamais le code
du lot).

## Ce qui attend une action UTILISATEUR

1. **Coup d'œil prod 2 min** (seule vérification visuelle restante du lot) : poser un
   like sur la galerie médias (doit survivre à un rechargement ET à un redéploiement),
   supprimer un média de test (modale de confirmation, disparition définitive).
   Les items 1.5/3.1 sont couverts par tests discriminants, il ne manque que l'œil.
2. **Tag `v7.3.0`** quand la v7.3 est jugée close (déclenche notification de release
   + « Quoi de neuf »). Rien du lot 2 ne le conditionne ; le Replay 2D (chantier
   utilisateur) est le seul sujet ouvert de la version.

## Chantiers ouverts, par priorité recommandée

- **Replay 2D** — chantier utilisateur en cours, hors périmètre agent.
- **Killsource** — re-différé le 02/08 ; branche `feat/killsource-prod` VIVANTE,
  reprise par `.ai/HANDOFF_KILLSOURCE_REPRISE.md` (inclut la validation « précision
  Infinite exploitable ? » et, à sa livraison, les véhicules Infinite dans le
  sunburst — le pendant H5 est déjà en prod).
- **TS7 (PR #70)** — bloqué AMONT : `typescript-eslint` peer `<6.1.0` (re-vérifié le
  03/08, commentaire daté sur la PR). Revérifier `npm view typescript-eslint@latest
  peerDependencies` avant toute tentative ; lot C du plan deps NON exécutable d'ici là.
- **Palette joueurs escouade** — LE sujet design consigné : ΔE 6,7 entre
  `narrative-dominant` et `divergent-pos`, `divergent-pos` double emploi (3e joueur +
  remplissage positif), token gris neutre manquant pour les échelles divergentes,
  collision `outcome-dnf` = `narrative-humiliation` (#8B5CF6). Mérite un artefact de
  propositions avant tout code.
- **Reliquat structurel** (liste exhaustive : plan lot 2, encadré de la section
  Découvertes) — notamment : `media_files.liked` global vs likes par viewer (à
  traiter avec une v2 de la suppression/likes), audit `RequireAuth` du groupe
  `/players/{slug}` (les co-membres passent par ownership), fixture démo inexploitable
  pour les pages joueur (bloque e2e et harnais sur données synthétiques), baselines
  du harnais visuel côté CI Linux (projet `visual` volontairement hors CI, suffixe
  -win32), migration seed FR `citation_mappings`, déplacement du registre d'armes
  hors de `halo_infinite/migrations/`, plugin TanStack Router qui régénère
  `routeTree.gen.ts` à chaque tsc/vitest (bruit de diff permanent), doublons i18n
  `compare/kpi/highlights`, ~37 vouvoiements résiduels dans les manifests, helper
  `errDBBusy()` à factoriser, 3e copie du seuil 1,59 (`coach_advisor/signals.go`).

## Pièges opérationnels du poste (vécus cette campagne)

1. **gate-push local** : l'env git-bash casse le lien des binaires de test embarquant
   `libduckdb_static` (emutls) — PowerShell natif lie correctement. Contournement
   validé : produire le JSONL depuis PowerShell (`go test -tags=integration -count=1
   -p 1 -json ./... > f.jsonl`) puis `bash scripts/check_test_baseline.sh tests
   --from-jsonl f.jsonl`. Un « ok » de go test peut être un hit de cache : seul
   `-count=1` + lien à froid fait foi.
2. **Encodage** : jamais de roundtrip `Get-Content | Set-Content` PowerShell 5.1 sur
   un fichier UTF-8 accentué (2 mojibakes durant la campagne, dont le plan entier —
   réparé). Édition de texte = outils d'édition, pas le shell.
3. **Baseline de tests** : toute suppression/renommage de test exige la réconciliation
   de `.ai/baselines/tests_pre_migration.jsonl` (retirer TOUTES les lignes d'événements
   du test) — sinon CI Coverage+Baseline rouge. Précédents : 11 tests du flag (3.3),
   1 test likes (1.5).
4. **Flakes qualifiés** (verts 10/10 isolés, documentés au plan) :
   `TestCareerLive_NilAPIResponse_NotCached` (timeout 2 s sous charge),
   `TestStartImport_HappyPathReturns202WithJobID` (TempDir Windows),
   `TestWorker_Run_PersistsAndACKs` (course WAL). Ne pas « corriger » sans analyse.
5. **Artefacts délégués** : une URL d'artefact rendue par un sous-agent peut être un
   fantôme — vérifier par `Artifact list` avant de la relayer ; préférer : l'agent
   écrit le fichier, l'orchestrateur relit INTÉGRALEMENT et publie.

## Références rapides

- Plan statué : `.ai/V7.3/PLAN_V73_LOT2_PETITES_CHOSES.md`
- Plan deps (lots A/B/C) : `.ai/PLAN_DEPS_ECHARTS_TS7_2026-07-27.md` (A et B faits,
  C bloqué amont)
- Harnais visuel : `apps/web/e2e/visual/` (README dedans ; baselines Lab versionnées,
  pages joueur locales non versionnées)
- Artefacts de décision (privés) : Rendement/Résistance `9775c1ab…`, dominance
  `b0cee06c…`, sign-off echarts `658ae5e8…`
- Notion « Backlog LevelUp » : section « Pour la v7.3 » à jour (callout Suivi du
  lot 2, 16 puces barrées avec commits)
