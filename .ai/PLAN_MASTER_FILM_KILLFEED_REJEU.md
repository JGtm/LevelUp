# PLAN MASTER — chantier film : captures, killfeed, rejeu 2D, cartes

> Écrit le 2026-07-31 (soir). Ordonne TOUS les plans et handoffs vivants de la racine `.ai/`,
> hors trois documents d'autres domaines (`PLAN_AXE_OBJECTIFS_INDEX.md`,
> `TS_TS7_2026-07-27.md`, `PLAN_REVUE_ANALYTIQUE_TIMESERIES_SQUAD_2026-07.md`).
>
> **RÔLE DE CE DOCUMENT.** Il ordonne, découpe en sessions et arbitre les dépendances.
> Il ne réécrit aucun sous-plan : chaque sous-plan reste l'autorité sur son contenu et
> tient son propre journal. En cas d'écart entre un sous-plan et l'état réel, la section
> « Corrections d'état » (§2) fait foi jusqu'à ce que la session concernée mette le
> sous-plan à jour.
>
> **Contrat d'exécution** : skill `plan-execution`, à recharger au début de CHAQUE session.
> Une étape close avant la suivante, chaque item statué (`[x]` / `[~]` réf / `[!]` justifié),
> découvertes consignées et non traitées.

---

## 1. L'ÉTAT RÉEL, VÉRIFIÉ SUR PIÈCES — 2026-07-31 soir

Tout ce qui suit a été revérifié ce soir (git, disque, CI), pas recopié des documents.

| fait | preuve |
|---|---|
| Branche vivante : **`feat/replay2d-prod`** (dépôt principal, arbre propre). Elle contient `main` (`6c2b00402`) + killsource + le rejeu 2D réunis | `git log`, `git status` |
| `PLAN_RECONCILIATION_BRANCHES.md` : **CLOS** — fusion bit-exacte, critères §5 tous verts sur les 3 films | journal du plan |
| `PLAN_REJEU_2D_FIABILISATION.md` : **CLOS** (étapes 1-6) | journal du plan, SUIVI |
| **CI de branche ROUGE** — 4 signaux, dont 3 qu'aucun plan ne couvre | `gh run list`, détail §3/J1 |
| Les briques de branchement killsource sont **ABSENTES de cette branche** : pas de `internal/sync/killsource_bridge.go`, pas de `steps_shared_kill_events.go` / `steps_shared_weapon_shots.go`, pas de persisters kill/shots. Seuls le décodeur (`internal/games/halo_infinite/film/killsource`) et `cmd/killsource` sont là | `ls` sur `internal/{sync,migration,persist}` |
| `config/titles/halo_infinite/mappings/weapon_names.toml` existe (cible du lot title-agnostic) | `ls mappings/` |
| `apps/web/src/lib/feature-flags.ts` existe encore (cible de suppression, lot 1 finalisation) | `ls` |

**Conséquence centrale** : le tableau « CE QUI EXISTE DÉJÀ » de `PLAN_BRANCHEMENT_KILLSOURCE.md`
décrit la branche d'ARCHIVE (`feat/filmdec-killweapon`), pas la branche vivante. La décision du
handoff tient : **réécrire** le pont, les 2 migrations, les 2 persisters et le correctif
d'indice du tueur contre le code actuel — ne pas les porter.

---

## 2. CORRECTIONS D'ÉTAT — à reporter dans chaque sous-plan à l'ouverture de sa session

| document | correction |
|---|---|
| `HANDOFF_KILLSOURCE_REPRISE.md` | La table des branches est périmée : la branche vivante est `feat/replay2d-prod` depuis la réconciliation du 31/07. `feat/killsource-prod` est absorbée, `filmdec-continuation` et `filmdec-killweapon` sont des archives |
| `PLAN_BRANCHEMENT_KILLSOURCE.md` | Phase 0 : CLOSE (autrement — par la réconciliation). Le tableau « CE QUI EXISTE DÉJÀ » est faux ici : pont, migrations, persisters à RÉÉCRIRE (cf. §1). L'ordre des phases 2 et 3 est corrigé par ce master plan (§J4) |
| `HANDOFF_REPLAY_2D_2026-07-29.md` | La « décision de branche » attendue est PRISE et exécutée. Le blocage du fil des éliminations est LEVÉ. Le document reste utile pour ses réserves ouvertes (cellule de munitions, `ownersFromLives`) |
| `PLAN_FINALISATION_REJEU_2D.md` | Lot 5 : DÉBLOQUÉ (killsource est sur la branche). Lot 7.4 : la note « ce lot ne vit pas sur cette branche » est périmée — `feat/replay2d-prod` descend de `main` ; le lot 7.4 reste néanmoins un chantier séparé (branche dédiée depuis `main`, cf. §J6-C) |
| `PLAN_DETTE_AVANT_MERGE.md` | S'ajoute à sa mesure : la CI de branche est rouge pour 3 causes HORS lint (gitleaks, typecheck web, suite Go Linux) — traitées en J1 avant les lots A-D |
| `PLAN_OBJECTIFS_TEMPS_REEL.md` | Étape 3.1 (table des désérialiseurs) : **FAITE en J0.4** — 34/34 objectifs et 33/33 zones désormais lisibles. Et la ROUTE de l'étape 3 change : la capture J0.6 prouve que les zones Strongholds n'émettent **rien** dans le flux de réplication (0 dispatch `ti=11` sur 1 205 704 records, contre 162 en CTF, alors que 4 ancres terrain attestent des événements de zone) — l'état des zones se décodera par les **images-clés + le footer d'événements type-3** (tous deux en cache → hors ligne, sans jeu ni Cheat Engine), pas par la « machine d'état zone (réplication) » supposée par le document de juin. Deux chaînes indépendantes, même conclusion |
| `PLAN_CAPACITES_ACTIVES.md` | Étape 3.4 (confrontation au relevé terrain surbouclier/camouflage) : **sans oracle** — la capture Catalyst `530820e5` (J0.5) a été faite sans relevé à l'œil. **DÉCIDÉ le 2026-07-31 (décision #1)** : pas de nouvelle capture ; 3.4 se fait en corrélation interne (3.3) seule, et le plan doit l'écrire à l'ouverture de sa session |
| `SUIVI_REPLAY_2D.md` | À mettre à jour après chaque jalon (c'est lui qui répond à « où en est-on ») |

---

## 3. LES JALONS — vue d'ensemble

```
J0  Captures avant changement de PC          [URGENT — seule fenêtre ; user + agent]
 │
J1  CI de branche verte                      [hygiène, découvert par cette revue]
 │
J2  Verrouiller le décodeur (goldens)        [FINALISATION lot 2 — AVANT de toucher au décodeur]
 │
J3  Dette avant merge, lots A-D              [PLAN_DETTE — lint ; E/F restent pour J5]
 │
J4  Killfeed en base et exposé               [PLAN_BRANCHEMENT réécrit + rejeu atteignable]
 │
J5  MERGE vers main + backfill prod          [= déploiement prod automatique — GO utilisateur]
 │
J6  Pistes parallèles post-merge             [A décodeur · B cartes · C objectifs · D icônes
                                              E précision projectiles · F rejeu-en-prod]
```

Justification de l'ordre :

- **J0 d'abord** : seule étape à fenêtre de tir — le changement de PC ferme définitivement
  Cheat Engine (table des désérialiseurs, oracles) et peut fermer l'accès réseau/tokens
  (film `696a9d7c`). Tout le reste peut attendre, pas elle.
- **J2 avant J3/J4/J6-A** : la dette de lint vit dans le décodeur (≈ 30 issues sur 40 dans
  `internal/`), et les corrections « mécaniques » (`unconvert`…) peuvent changer un décodage
  en silence. On ne touche pas un décodeur non verrouillé. Le gate « artefacts identiques »
  existe déjà en manuel (§5 réconciliation) ; J2 le transforme en tests.
- **J4 avant les features rejeu qui consomment le killfeed** (fil des éliminations, médailles,
  effets sur les morts — J6-A) : c'est la demande explicite de l'utilisateur, et c'est aussi
  l'ordre de la valeur — la table actuelle `killer_victim_pairs` porte 46,8 % de doublons et
  gonfle les agrégats carrière d'un facteur 1,879.
- **J5 tôt, pas tard** : `main` avance vite (1 855 commits en 2 mois pendant que les branches
  de recherche divergeaient — la réconciliation a coûté une session entière). Plus le merge
  attend, plus il coûte. Après J5, tous les chantiers repartent en branches courtes depuis
  `main`, et le parallélisme devient sûr.

---

## J0 — CAPTURES AVANT CHANGEMENT DE PC  *(URGENT — à faire en premier)*

**Documents d'autorité** : `HANDOFF_DUMPS_2026-07-31.md` (raisonnement + inventaire) et
`SESSION_CAPTURE_AVANT_PC.md` (liste d'actions). Ils sont complets et corrects — rien à y
changer, tout à exécuter.

### Ce qu'un AGENT peut faire dès maintenant, sans le jeu (1 session)

- [x] J0.1 **Télécharger le film `696a9d7c`** (Strongholds, Nomad/Vagabond) : obtenir le
      manifest via l'API (réseau + tokens ENCORE vivants sur ce PC), puis les chunks depuis le
      CDN (`cmd/fetch_film_chunks`), puis copier sur la clé. **C'est le maillon fragile : à
      faire tant que l'auth répond.**
      *FAIT 2026-07-31.* Manifest frais via `cmd/tmp_filmmanifest` (ADR 0023, aucune
      re-capture) : 31 chunks. 29 REPLICATION_DATA via `fetch_film_chunks`, **plus l'en-tête
      (chunk 0, type 1) et le footer d'events (chunk 30, type 3)** que `fetch_film_chunks` ne
      prend pas — sans l'en-tête le décodeur ne peut pas amorcer son World. Contrôle
      `killsource sante` : NOMINAL, 102 morts, couverture 100 %. Copie sur la clé vérifiée
      bit-exacte (31/31).
- [x] J0.2 **A1/A2 du protocole** : décoder `catalyst_catalyst.mvar` et
      `vagabond_fo08_wetland.mvar` avec `mapvar`, sortir les `TypeID` + positions + chaînes
      lisibles, croiser avec `forge_object_types.csv` → réponse « quels power-ups porte chaque
      carte, et où » **sans lancer le jeu**. Confirmer Vagabond = `fo08_wetland` par la méthode
      du `level_id` (21/21 jusqu'ici), pas seulement par le nom de fichier.
      *PARTIEL — le volet `level_id` est FAIT et PROUVÉ, le volet power-ups est `[!]`.*
      - `[x]` **Vagabond = `fo08_wetland`, prouvé par le `level_id`** : 88891201 (0x054C5F41)
        balayé sur les 88 modules de `deploy/any` + `deploy/ds` → **exactement une occurrence**,
        `multi/fo08_wetland/fo08_wetland-rtx-new.module`, offset +0x28, groupe `levl`. Témoin
        Catalyst identique (−1044063363 → `multi/catalyst`, 1 seule). La méthode 21/21 tient.
        Outil : `cmd/tmp_forgedim scan <deploy> <hex>`.
      - `[!]` **Les power-ups ne sont PAS déductibles hors ligne avec ce qu'on possède.**
        Deux voies fermées sur pièces : (a) les « chaînes lisibles » ne portent aucun nom
        d'équipement — `catalyst_catalyst.mvar` a 0 chaîne, `vagabond_map.mvar` en a 265 mais
        ce sont des `"Prefab 0".."Prefab 16"` ; (b) `forge_object_types.csv` couvre 33/42
        `type_id` de Catalyst (78 %) mais **15/468 de Vagabond (3 %)** — carte Forge, ses objets
        sont des blocs Forge — et ses 45 lignes ne portent **aucun groupe `eqip`** (seulement
        bloc/mach/scen/vehi/weap), donc aucun équipement n'y est nommé.
      - **BLOCAGE IDENTIFIÉ, à arbitrer** : la seule voie restante est la palette Forge
        (`any|ds/globals/forge/forge_objects-rtx-new.module`, 2,4 Go). Ces modules **ne sont pas
        sur la clé** — seuls les 31 modules de niveau y sont. Sans eux la résolution
        `type_id → nom` meurt avec le PC. Espace libre sur la clé : 42 Go.
- [x] J0.3 Vérifier l'inventaire de la clé contre la liste du handoff (dont
      `deser_table.tsv` APRÈS la capture utilisateur) et pousser toutes les branches sur
      `origin` (vérifié : les 4 branches y sont — re-vérifier après tout commit local).
      *FAIT.* Clé conforme : `retro_ingenierie/` 5,1 Go (HI.rep + HI.gpr + exe + Ghidra 12.1 +
      ghidra-mcp), `jeu_deploy_ds/` 162 Mo, `captures_cheat_engine/` 164 Mo (199 `.mvar`),
      `scripts_cheat_engine/` 2,3 Mo, `filmdec_deser_table.lua`, `POC_reference_rejeu2D.html`,
      `data/` 951 films + 951 manifests + `reference/`. Les 4 branches : **0 commit non poussé**.

### Ce que l'UTILISATEUR seul peut faire (jeu + Cheat Engine)

> **PRÉMISSE CORRIGÉE le 2026-07-31** : ces trois items étaient classés « utilisateur seul »
> parce qu'on supposait l'agent incapable de piloter Cheat Engine. C'est FAUX — le bridge
> MCP `cheatengine` le permet. L'agent a donc exécuté J0.4 à J0.6 lui-même ; le rôle humain
> se réduit à lancer le film et à tenir le relevé terrain.

- [x] J0.4 **La table des désérialiseurs** — 15 min, script `filmdec_deser_table.lua` prêt.
      Contrôle : archétype 35 = 64 composants. Sans elle : objectifs (0/34), zones (0/33),
      moitié des véhicules/dispositifs **illisibles à jamais sur ce build**.
      *FAIT 2026-07-31, via MCP Cheat Engine.* **50 archétypes, 1068 composants**, 1061 via
      thunk, 1 sans désérialiseur. **Double contrôle PASSÉ** : archétype 35 = 64 composants ET
      `vtable[0x60]` = `0x140F44C38`. Les quatre trous sont comblés : objectifs 34, zones 33,
      véhicules 48, dispositifs 41. `deser_table.tsv` (47 Ko) sur la clé.
      **BONUS non prévu** : `archetype_vtables.tsv` — descripteur ET vtable des 50 archétypes
      en adresses Ghidra. Lecture live pure, impossible à refaire hors ligne ; ce sont des
      points d'entrée directs dans le projet Ghidra. (Le binaire est dépouillé : aucun nom
      d'archétype ne se lit en mémoire — piste fermée pour une sonde.)
- [~] J0.5 **Un match à objectif sur Catalyst** avec surbouclier + camouflage, capture continue
      + **relevé terrain écrit** (qui prend quoi, à quelle seconde). Sans relevé, la capture est
      infalsifiable donc sans valeur.
      *CAPTURE FAITE, RELEVÉ ABSENT.* Film `530820e5` (CTF:Arena sur Catalyst, choisi par
      l'utilisateur) téléchargé puis capturé en entier : **988 752 records, 1 364 entités**,
      identité **prouvée à 99,8 %** par alignement de signatures (témoins 0,0 % et 1,0 %).
      Aucune observation à l'œil n'a été relevée pendant la lecture. La capture garde de la
      valeur (trois oracles hors ligne existent : footer type-3 en cache, détecteur `tiers==6`,
      stat `FlagCaptures`) — **mais surbouclier et camouflage n'ont PAS d'oracle hors ligne**
      et restent donc non couverts. C'est le seul item de J0 à refaire si l'occasion revient.
- [x] J0.6 Strongholds sur Vagabond : rejouer `696a9d7c` (une fois téléchargé) avec capture +
      relevé zones. KOTH/Oddball : reportés, notés.
      *FAIT, AVEC RELEVÉ.* **1 205 704 records, 2 095 entités**, identité prouvée à 99,0 %.
      Quatre ancres terrain horodatées (0:48 base B flyguy8773 · 1:30 trois bases, score 21 ·
      3:10 score 69-30 · 5:34 trois bases). Relevé écrit :
      `.ai/RELEVE_TERRAIN_CAPTURES_2026-07-31.md`, copié sur la clé avec les CSV.

**GATE J0** : `deser_table.tsv` sur la clé avec son contrôle passé ; `696a9d7c` en cache ET sur
la clé ; captures + relevés terrain copiés ; liste A1/A2 rendue (power-ups par carte).

**GATE J0 — STATUT 2026-07-31 : PASSÉ SAUF LE VOLET POWER-UPS.** `deser_table.tsv` sur la clé,
double contrôle tombé ; `696a9d7c` en cache et sur la clé (bit-exact) ; deux captures + le
relevé copiés ; **la liste A1/A2 n'est PAS rendue** — power-ups non déductibles sans la palette
Forge (cf. J0.2 `[!]`). Deux réserves ouvertes : ce blocage, et l'absence de relevé sur J0.5.

---

## J1 — CI DE BRANCHE VERTE  *(hygiène — découvert par cette revue, planifié nulle part)*

Leçon du dépôt (VF-16) : un signal rouge public ignoré = lot non clos. La branche a 4 signaux
rouges, mesurés ce soir sur `gh run list` :

> **CLOS le 2026-07-31 (nuit)** — gate atteint : `gh run list --branch feat/replay2d-prod`
> donne Secrets (gitleaks) vert, Deploy Pre-Check vert, et dans le workflow CI tout vert
> (Frontend, Go Build+Test ubuntu ET windows, Go Coverage + Baseline, Contract Test,
> Lease Enforcement, OpenAPI Lint) **sauf** `Go Lint (golangci-lint)`, rouge assumé
> jusqu'à J3. Commits : `3a9158ae4` (J1.1), `566ebb777` + `dd020364a` (J1.2),
> `4e2bd5dba` (J1.3).

- [x] J1.1 **Secrets (gitleaks)** : 1 finding —
      `.ai/V7.5/killweapon/RE_LOG_KILLWEAPON.md:generic-api-key:17478` (des gamertags de doc
      lus comme une clé — faux positif déjà connu du handoff killsource ; le déplacement du
      fichier vers `V7.5/` a invalidé les empreintes posées sur l'archive). Ajouter l'empreinte
      au `.gitleaks.toml` de CETTE branche, commentée et datée. Vérifier s'il reste d'autres
      findings derrière celui-ci (le run s'arrête au premier lot).
      **FAIT** : exception ciblée (règle qui a bruité + chemin exact + valeur, `condition = AND`),
      posée sur la VALEUR et non sur l'empreinte — un numéro de ligne dans un journal qui
      grossit ne survit pas à la modification suivante, c'est ce qui avait invalidé les
      exceptions précédentes. **Le run ne s'arrête PAS au premier lot** : gitleaks scanne tout
      (130,92 Mo) et n'avait qu'un finding — les documents commités en J0 n'en lèvent aucun.
      Rejoué en local avec le binaire épinglé de la CI (8.30.1) sur copie propre : 0.
- [x] J1.2 **Type-check web** : ~26 erreurs, toutes dans `features/match-replay/` — cause
      unique : le dernier commit (`2d8bcec8e`, document de rejeu tiré du contrat généré) rend
      les tableaux nullables (`tracks`, `points`… `| null`) et les composants ne les gardent
      pas. Traiter par un normalisateur unique à la frontière (le hook/query qui charge le
      document), pas par 26 `?.` éparpillés.
      **FAIT** : 31 erreurs en réalité, et **deux** causes, pas une. (1) la nullabilité —
      `apps/web/src/features/match-replay/replayNormalize.ts` comble les tableaux UNE fois dans
      la queryFn, tout le dossier ne manipule plus que des types `*Ready` ; les fabriques de
      test passent par la même porte. (2) le contrat perd aussi l'ARITÉ des tuples Go
      (`Poly [][2]float32`, `P [][3]float32` → `number[]`), rétablie au même endroit.
- [x] J1.3 **Suite Go Linux (CGO + intégration)** : rouge avec des tests baseline « absents du
      run courant » dans des paquets sans rapport (`domain/title`, `platform/duckdb`…) — le
      Windows CI est vert, donc c'est un problème Linux-seulement (piste probable : chemin
      sensible à la casse ou fichier déplacé par le rangement `.ai/` → `V7.5/` que `mapvar` ou
      un test lit ; diagnostiquer par le JSONL du run, pas deviner).
      **FAIT — la piste « chemin » était fausse, le JSONL a tranché** : 2 052 lignes, **0 test**.
      La suite ne tournait pas, elle ne COMPILAIT pas — `internal/ooz/ooz_compat.h` n'incluait
      que `<intrin.h>` (MSVC/mingw), absent sous gcc Linux ; tous les paquets tombaient en
      `[build failed]` et le comparateur déclarait donc les 8 786 tests baseline disparus.
      Shim rendu portable (branche Windows inchangée ; ailleurs `x86intrin.h` + builtins GNU,
      `_BitScanReverse/Forward` réécrits avec le contrat MSVC exact). Non-régression du
      décodeur vérifiée : les 3 artefacts reconstruits rendent les chiffres du §5 de
      `PLAN_RECONCILIATION_BRANCHES` à l'unité près.
- [x] J1.4 **Go Lint (ratchet golangci)** : rouge CONNU et PLANIFIÉ — c'est `PLAN_DETTE`
      (J3). Ne pas le traiter ici ; vérifier seulement qu'il ne masque pas un autre échec.
      **VÉRIFIÉ, non traité** : le job ne masque aucun échec d'une autre nature (aucun test
      cassé, aucune autre compilation en défaut) — mais l'inverse était vrai, et c'est la
      découverte : AVANT J1.3 il ne rapportait **qu'une seule** issue, le `typecheck` du même
      `intrin.h`, qui arrêtait la course. Le compte réel de la dette apparaît maintenant :
      **70 issues** (unused 33, unconvert 11, unparam 7, revive 6, staticcheck 4, goconst 3,
      gocyclo 2, prealloc 2, errcheck 1, ineffassign 1), concentrées dans `analysis/filmdec` et
      `analysis/objectiveevents`. J3 doit être dimensionné sur 70, pas sur les ~40 estimés.

**GATE J1** : `gh run list --branch feat/replay2d-prod` — tout vert sauf le ratchet lint
(rouge assumé jusqu'à J3/J5). Typecheck local : cache `.tsbuildinfo` purgé avant conclusion.

---

## J2 — VERROUILLER LE DÉCODEUR  *(FINALISATION lot 2 — promu AVANT tout le reste)*

> **CLOS le 2026-07-31 (nuit)** — GATE J2 ATTEINT. Les six items du lot 2 statués `[x]`
> (le 12e champ de 2.6 en `[~]`, renvoyé au lot 3.6). Commits : `5e37f4c79` (2.1+2.2),
> `d1870b3a7` (2.3), `a05d7d448` (2.4), `96bc56175` (2.5), `eb0f12d23` (2.6).
> Couverture du paquet `replay` : **58,5 % → 79,2 %**. Cinq découvertes consignées, aucune
> traitée (D1 à D5 dans le plan de finalisation) — dont **une panique atteignable en
> production** et **la non-reproductibilité de l'artefact à l'octet**.

**Document d'autorité** : `PLAN_FINALISATION_REJEU_2D.md` lot 2 (2.1 à 2.6). Périmètre fermé
là-bas ; ce jalon n'ajoute que la justification d'ordre : les chiffres du chantier (475/519,
90/105, 70 lancers, 439 projectiles, 184 états…) ne sont AUJOURD'HUI verrouillés par aucun
test — ils vivent dans des Markdown et un artefact généré. Or J3 (lint sur le décodeur), J4
(réécriture du branchement) et J6-A (nouvelles variables) vont tous toucher du code de
décodage. **Verrouiller d'abord, modifier ensuite.**

Contenu (résumé — le lot 2 fait foi) : goldens des entrées décodées (2.1), mini-bobine
binaire versionnée avec outil de régénération (2.2), grammaire d'inventaire (2.3), contexte
canvas enregistreur (2.4), les 3 décodeurs d'événements à 0 % (2.5), alignement des trois
contrats Go/OpenAPI/TS avec test de divergence (2.6 — J1.2 en a déjà fait une partie).

**GATE J2** : celui du lot 2 — un changement de décodeur qui déplace un chiffre fait tomber un
test NOMMÉ, sans lire un seul octet de film pour les goldens d'assemblage.

---

## J3 — DETTE AVANT MERGE, LOTS A-D  *(PLAN_DETTE_AVANT_MERGE)*

**Document d'autorité** : `PLAN_DETTE_AVANT_MERGE.md`. Sa règle de reprise s'applique :
re-mesurer d'abord (§1), ne jamais partir des chiffres écrits. Lots dans l'ordre : A (archiver
l'outillage jetable), B (passe mécanique), C (les 14 `unused` de filmdec — le seul lot à
jugement), D (`gocyclo`/`staticcheck`). **Le gate de B/C/D est celui de J2** : artefacts
identiques + goldens killsource verts après chaque linter traité.

Les lots E (go/no-go dérivant) et F (compteur à 0, re-merge de `main`, prévenir avant push)
restent accrochés à J5 — les faire trop tôt serait les refaire.

Ajouts de cette revue au plan (à statuer dans sa session) :

- [ ] J3.1 Purger le cache golangci avant toute conclusion (piège déjà payé sur ce dépôt :
      un cache chaud rend un faux compte).
- [ ] J3.2 Lot A : supprimer aussi le jeu d'images de grenades orphelin (`Dynamo-light.png`,
      casse majuscule — découverte n°1 de la réconciliation, aucun consommateur).
- [ ] J3.3 Élargir le déclencheur du hook pre-push `knip-ratchet` (découverte J1-b : il ne se
      déclenche que sur `apps/web/**` — un dépassement de plafond est resté invisible tant
      que les push étaient documentaires). Correctif honnête au choix : déclencher aussi sur
      la config/le plafond knip, ou jouer le ratchet en CI sans condition de chemin.

---

## J4 — KILLFEED EN BASE ET EXPOSÉ + REJEU ATTEIGNABLE

**Documents d'autorité** : `PLAN_BRANCHEMENT_KILLSOURCE.md` (phases 1-4) et
`PLAN_FINALISATION_REJEU_2D.md` (lot 1, + 3.1/3.2). Deux corrections d'ordre par rapport au
plan de branchement, décidées ici :

**Correction 1 — réécrire les briques avant la phase 1.** Le pont, les 2 migrations
(append-only + vue `_latest`, recette ADR 0026), les 2 persisters (INSERT-only, ADR
0019/0030) et le correctif d'indice du tueur n'existent pas sur cette branche (§1). C'est le
préalable de la phase 1, avec `go test -tags=integration -p 1` comme gate (OBLIGATOIRE,
persist/sync touchés).

**Correction 2 — inverser les phases 2 et 3.** Le plan enchaîne P1 collecteur → P2 bascule
des 8 lecteurs (vue) → P3 backfill. Mais basculer les lecteurs sur une vue posée sur une
table VIDE casserait les pages carrière entre P2 et P3. Ordre corrigé :

1. **P1** — le collecteur (tâche de fond, capability `film_kill_source`, jamais dans une
   requête HTTP, un décodage à la fois, limite de temps + compteur d'abandons, expvar).
2. **P3** — le backfill LOCAL d'abord (sous-commande `levelup`, reprenable par `decoder_rev`,
   BTB en dernier), jusqu'à couverture des matchs porteurs de films.
3. **P2** — la bascule des 8 lecteurs (+ les 2 sites spéciaux S1/S2), gate AVANT/APRÈS sur
   données réelles. Les deux arbitrages du plan tiennent (bots exclus des agrégats — attention
   Q26 ne filtre pas les bots aujourd'hui ; BTB inclus en cumul, interdit ligne à ligne).
4. **P4** — l'exposition produit (capabilities par famille, i18n FR+EN, tokens). Le piège
   MA40/Sidekick du §4.2 est NON NÉGOCIABLE : jamais de taux par arme sur corpus entier.

**Découverte J0.1 à intégrer au pont** : `cmd/fetch_film_chunks` ne télécharge QUE les chunks
REPLICATION_DATA — ni l'en-tête (chunk 0, type 1) ni le footer d'événements (type 3), sans
lesquels le décodeur n'amorce pas son World. Le pont réécrit doit les prendre aussi (J0.1 a dû
les récupérer à part).

**Pour la prod, la bascule des lecteurs se déploie en DEUX temps** (pas de flag OFF — règle
11) : release 1 = collecteur + migrations + CLI backfill → backfill joué sur le VPS (prévenir
avant, fenêtre à convenir) ; release 2 = bascule des lecteurs. Le découpage exact des commits
se décide en J5 selon l'état du backfill prod.

**Et le rejeu devient atteignable** (lot 1 finalisation, dans ce jalon parce qu'il est court
et rend le travail visible) :

- [ ] J4.1 Lot 1.1-1.3 : supprimer `feature-flags.ts` (mort depuis avril), publier
      `replay_available` (un `os.Stat` via `PathResolver`), poser le lien conditionnel
      (i18n FR+EN). En prod, l'artefact n'existe pas → pas de lien : inoffensif d'ici le
      chantier F. **Le gate visuel du lot 1 couvre AUSSI le correctif J1-a** (les familles
      d'effets de tir enfin différenciées — champ `w`) : première revue d'écran depuis que
      ce bug, vieux de tout le rejeu 2D, est corrigé.
- [ ] J4.2 Lot 1.4 : le garde local — TRANCHÉ le 2026-07-31 : **comprendre le CTF d'abord**.
      Diagnostiquer pourquoi `64e8adfa` perd 564 tirs « slot introuvable » là où le Slayer en
      perd 44 (hypothèse vies courtes → traces non publiées, à mesurer, pas à supposer),
      écrire la cause, PUIS redécider retrait/critère avec l'utilisateur. Le garde reste en
      place d'ici là.
- [ ] J4.3 Lot 3.1 + 3.2 (title-agnostic minimal AVANT merge) : unifier les deux tables de
      grenades (« Dynamo » vs « Shock » pour le même rang — une contradiction visible à
      l'écran), sortir armes/grenades/capacités vers
      `config/titles/halo_infinite/mappings/*.toml` (`weapon_names.toml` existe, bilingue),
      factoriser les 4 copies de `keep*OfPublishedTracks` avec garde-rail, géométrie via
      `PathResolver` (le défaut actuel pointe `.ai/V7.5/dumps` — un répertoire de
      rétro-ingénierie). Motif : ne pas MERGER vers `main` de nouvelles violations des règles
      du dépôt (libellés FR en dur côté Go). Les lots 3.3-3.6 (découpages) restent post-merge.

**GATE J4** : gates des phases du plan de branchement (dont intégration `-p 1`) + gate 1 du
lot 1 + `SELECT` de contrôle montrant les agrégats carrière dégonflés du facteur attendu en
local.

---

## J5 — MERGE VERS `main`  *(= déploiement prod automatique — GO utilisateur explicite)*

**Document d'autorité** : `PLAN_DETTE_AVANT_MERGE.md` lots E et F, plus la checklist
`delivery-checklist` complète.

- [ ] J5.1 Lot E en entier (openapi + types régénérés, routeTree, knip, vitest, tsc cache
      purgé, `go test ./...` + `-tags=integration -p 1`, montée Go 1.26.2 puis `govulncheck`,
      ratchets ADR 0023, thought_log + rotation trimestrielle + rangement `.ai/`).
      Vigilance découverte J1-c : `routeTree.gen.ts` est régénéré (ordre d'imports inversé)
      par toute commande vite/vitest — vérifier le CONTENU du diff avant de committer ou
      d'écarter, ne pas se fier à la seule présence d'un diff.
- [ ] J5.2 Lot F : compteur ratchet à 0, re-merge de `origin/main` puis re-mesure, **prévenir
      l'utilisateur avant le push** (déploiement prod automatique).
- [ ] J5.3 Backfill prod en deux temps (cf. J4) : fenêtre convenue, lock backfill respecté
      (pièges VPS connus : disque, cache BuildKit, swap).
- [ ] J5.4 Après merge : archiver les branches de recherche (`filmdec-killweapon`,
      `filmdec-continuation`, `killsource-prod`, `weapon-*`) et retirer leurs worktrees
      (jonctions d'abord — piège connu), mettre à jour `SUIVI_REPLAY_2D.md` et les handoffs.

**GATE J5** : CI de `main` verte de bout en bout, prod déployée, backfill joué, agrégats
carrière dégonflés en prod (vérifier par lecture directe, pas par déduction).

---

## J6 — LES PISTES PARALLÈLES POST-MERGE

Après J5, chaque piste = **une branche courte depuis `main`**, un worktree, une session (ou
suite de sessions), un merge rapide (< 1 semaine de vie par branche). Surfaces disjointes :

| piste | plans | surface de fichiers | dépendances |
|---|---|---|---|
| **A — décodeur** (UNE seule session à la fois, toujours) | `PLAN_CAPACITES_ACTIVES` puis `PLAN_VARIABLES_JETEES` puis FINALISATION lot 5 (fil des éliminations + médailles) et lot 6, puis `PLAN_OBJECTIFS_TEMPS_REEL` étape 3 | `filmdec/`, `analysis/replay/`, `killsource/`, artefact, `match-replay/` | J2 (goldens) ; étape 3 objectifs : J0.4 (table) + Ghidra |
| **B — cartes** | `PLAN_BELLE_CARTE_TRIANGLES` (étapes 0.1 → 6) puis FINALISATION lot 4 | `himap/`, `himodule/`, `cmd/mapstruct-build`, `mapFloor.ts`, `data/.../reference/` | aucune (modules du jeu sur la clé) |
| **C — objectifs côté app** | `PLAN_OBJECTIFS_TEMPS_REEL` étapes 1-2 (brancher ce qui dort) ; FINALISATION lot 7.4 (scores de mode) sur SA PROPRE branche | `internal/sync/` (producteur), front consommation, `map_objectives.json` ; 7.4 : `analysis/` scores | étape 1 : rien ; 7.4 : rien (mais gros) |
| **D — icônes d'armes** | `PLAN_RECHERCHE_ASSETS_ICONES` | `himodule/` lecture, `static/`, `TitleAssetURLAdapter` | rien ; peut échouer sans dette ; **GATE HUMAIN entre phase 1 et phase 2** (revue visuelle utilisateur AVANT toute intégration — décision #4) |
| **E — précision projectiles** | `HANDOFF_PRECISION_PROJECTILES` | outillage de mesure + Ghidra (hors ligne) | rien ; MESURE seulement jusqu'au verdict ; **timebox 2 sessions** puis verdict écrit (décision #6) |
| **F — le rejeu en prod** | direction tranchée (décision #5), design à écrire | design d'abord | J5 |
| **G — citations dérivées** *(ajoutée le 2026-07-31)* | catalogue citations Halo Infinite (pipeline seed + recompute existant), modèles Halo 5 | `config/titles/*/mappings/`, analyse citations, i18n | G1 : après J4 · G2 : après A5 · G3 : EN DERNIER (bonus) |

Détail de la piste G — porter au catalogue Halo Infinite des citations calquées sur
Halo 5, quand la donnée existe :

- **G1 — kills au véhicule** : la donnée arrive avec J4 (`match_kill_events` porte la source
  du dégât fatal ; écrasements et armements de véhicule confirmés 7/7 au Theater). Reprendre
  les citations véhicule du catalogue H5 comme modèles. Aucune RE nouvelle.
- **G2 — ramassages** (modèle H5 « Wonderful Toys » : ramasser 3+ armes ou power-ups dans un
  match Slayer) : dépend du verdict A5. Le type canonique **existe déjà**
  (`canonical.MatchEventWeaponPickup`) et Halo 5 mappe déjà ses événements `WeaponPickup`
  dessus (`games/halo_5/events.go`) — un producteur Halo Infinite se branche sur le MÊME
  type canonique, et la citation se calcule par titre. Nuance de règle : la dotation de
  spawn n'est pas un ramassage (sinon la borne inférieure par images-clés sur-attribue).
- **G3 — objectifs** : si C (étapes 1-2) et A4 sortent plus de stats d'objectif, dériver des
  citations sur le modèle H5. **En toute fin, bonus** — arbitrage utilisateur du 2026-07-31.

Règles G : catalogue par titre (TOML + seed/recompute), libellés FR **et** EN, capability et
jamais de slug — une citation ne se calcule pas sur une stat que le titre ne mesure pas.

Règles de parallélisme (voir §7) : A est exclusif sur sa surface ; B, C, D, E peuvent tourner
en parallèle de A et entre eux ; C-7.4 est une branche à part parce qu'il recalibre des scores
que la piste A ne touche pas.

Ordre INTERNE de la piste A (respecte « killfeed d'abord », puis la valeur) :

1. Fil des éliminations + médailles (lot 5 — la donnée killsource est là, l'écran est dessiné).
2. Capacités (étape 1 : relire sur 6 bits — mesure pure, AUCUNE capture nécessaire, prioritaire
   depuis le 27/07 et jamais faite ; puis table TOML, `i57`, affichage — les étapes 2.3/3.4
   consomment les captures J0).
3. Variables jetées (compteur de réapparition lu, horloge de manche, pitch/orientation selon
   couverture).
4. Objectifs étape 3 (désérialiseurs décompilés puis portés un par un, témoin
   `progress/required-progress`).
5. **Ramassages — armes, power-ups, grenades** *(ajouté le 2026-07-31, demande utilisateur ;
   la piste est DÉJÀ NOMMÉE dans les archives, vérifié par grep)* :
   - Voie 1 — l'ÉVÉNEMENT de ramassage : le dispatcher d'événements du film
     (`FUN_140620564`, codes **0x02-0x3c : spawn / pickup / objectif / équipement**) porte
     une famille jamais décodée (`V7.5/killweapon/KILLFEED_STATE.md` §183). Le chantier
     killweapon l'avait identifiée comme « LA pièce manquante » pour la possession
     joueur ↔ entité-arme (`V7.5/killweapon/FIRE_MELEE_GRENADE_EVENTS.md` §371 et §494)
     avant que le dead-state ne rende la question inutile pour l'arme du kill. Elle dort.
   - Voie 2 — les records de **swap/pickup du weapon-state** (~247 par film, l'id64 d'arme
     ré-émis au ramassage), déjà à moitié décodés dans l'outillage
     (`cmd/tmp_killfeed_weapons`).
   - Repli si l'événement résiste : la DIFFÉRENCE de loadout entre images-clés (~20 s),
     borne inférieure EXPLICITE ET COMPTÉE, en excluant la dotation de spawn de chaque vie.
   - Oracles indépendants : le relevé terrain J0.5 (qui prend surbouclier/camouflage, et
     quand), les positions de socles depuis les `.mvar` (J0.2), les décréments de grenades.
   - Débouchés : stats de contrôle de carte, calque de ramassage au rejeu, citations (G2).

---

## §5 — REVUE CRITIQUE DES PLANS  *(la relecture demandée)*

### 5.1 Verdict d'ensemble

Les plans sont d'une rigueur inhabituelle : périmètres fermés, gates falsifiables énoncés
AVANT l'exécution, chiffres mesurés et sourcés, échecs consignés comme des résultats,
protocoles de reprise. La grille `plan-review` passe sur l'essentiel : couches respectées
(décodeur title-specific dans `games/halo_infinite/`, orchestration dans `sync/`, écritures
par persisters INSERT-only), capabilities plutôt que slugs, i18n FR+EN prévu, tokens de
couleur, dégradation `ErrCapabilityNotSupported`. Les défauts trouvés sont des défauts de
FRAÎCHEUR (l'état a bougé le jour même) et d'ORDRE, pas de conception.

### 5.2 Ce qu'aucun plan n'écrivait — trouvé par cette revue

1. **La CI de branche est rouge pour 3 causes hors lint** (gitleaks après déplacement de
   fichier, typecheck sur les types nullables du contrat, suite Go Linux) → J1.
2. **L'histoire prod du rejeu n'existe nulle part.** Tous les plans rendent le rejeu
   atteignable en LOCAL (artefact présent sur disque). Personne ne dit qui produit les
   artefacts EN PROD : le VPS n'a pas les 23 Go de films, un build coûte 8-30 s (11 min en
   BTB), le disque VPS est déjà sous tension, et les films Theater EXPIRENT côté serveur (28 %
   des matchs n'auront jamais de film). À cadrer comme chantier F : produire au sync (comme le
   collecteur killsource), sur demande avec file d'attente, ou pas du tout au début —
   décision produit. **Recommandation associée, bon marché : collecter les MANIFESTS de film
   au sync dès maintenant** (~125 Ko/match) — ils permettent de retélécharger les chunks du
   CDN sans auth, c'est l'assurance contre l'expiration et la meilleure du lot (déjà prouvée
   sur ce PC : 950 manifests = 119 Mo).
3. **L'ordre P2/P3 du branchement** (vue sur table vide) et la stratégie deux-déploiements →
   corrigés en J4.
4. **L'étape 1 d'OBJECTIFS ne nomme pas son chemin de persistance.** « Faire produire les
   événements par la synchronisation » doit passer par `persist.BatchBuilder`/persister
   INSERT-only et, si la table est partagée per-match, par le patron append-only + vue
   `_latest` (ADR 0019/0026/0030) — à écrire dans le plan AVANT de coder. Même remarque pour
   la capability (clé fine par titre) et la consommation front (query keys title-scopées).
5. **Convention de nommage des capabilities** : `film_kill_source` (branchement) vs
   `film.replay2d` (finalisation 3.5). Unifier sur la convention réelle de
   `capabilities.toml` à la première pose, et s'y tenir.
6. **Le Python de la belle carte** (étape 0.1) : le versionner sous `.ai/V7.5/cartes/py/`
   comme ARCHIVE de recherche (jamais importé, jamais exécuté par l'app — la règle « pas de
   nouveau Python » vise le code applicatif), et le SUPPRIMER après le gate 3 du portage Go
   (règle 7 : git garde l'historique). Les rendus PNG de référence, eux, restent : ce sont
   les goldens visuels.
7. **Les visuels d'armes “à reprendre entièrement”** (28 fichiers mal nommés — `Cremator.png`
   est la Cindershot) ne vivent que dans le SUIVI. Ils sont couverts par la piste D si la voie
   assets aboutit ; sinon, reprise manuelle à planifier — à ne pas perdre.
8. **Backlog assumé, nulle part ailleurs que dans le SUIVI** : dispositifs de carte (canon —
   industrialisation), découpage des zones (règle valable 30 cartes), carte de chaleur, sons,
   POC 2 CTF (l'artefact `64e8adfa` est construit — il servira de terrain à C et A4). Statut
   inchangé : rang 3-4 / non planifié.

### 5.3 Revue plan par plan (points saillants seulement)

| plan | verdict | points relevés |
|---|---|---|
| `HANDOFF_DUMPS` + `SESSION_CAPTURE` | **complets, exécutables tels quels** | La matrice besoin/possédé est le bon outil. Ajouter (fait ici) : CI verte et push des branches à la liste « avant de débrancher » |
| `PLAN_BRANCHEMENT_KILLSOURCE` | **bon, deux corrections** | Inventaire périmé (§1) ; ordre P2/P3 (§J4). Le reste est exemplaire : anti-ART, 3 états d'assistant, `decoder_rev`, expvar, les 10 sites lecteurs recensés dont la copie Q19b/qKV à traiter ensemble, le piège Q26/Q27 sur les bots |
| `PLAN_DETTE_AVANT_MERGE` | **bon** | Le parti « pas de liste figée, une mesure » est le bon. Manquait : les rouges CI hors lint (J1) et la purge du cache lint (J3.1) |
| `PLAN_FINALISATION_REJEU_2D` | **bon, statuts à rafraîchir** | Lot 2 promu en J2 (verrouiller AVANT de refactorer) ; lot 5 débloqué ; 7.4 = branche à part ; lot 1.4 = décision utilisateur explicite (le critère du garde s'est révélé trop faible — 80,3 % sur le CTF déjà en cache) |
| `PLAN_VARIABLES_JETEES` | **bon** | L'arbitrage « à ne pas brancher » (médailles via API, dead-state au voisin) est juste et évite deux décodeurs du même fait. Gates falsifiables (couverture, plausibilité, témoin croisé) |
| `PLAN_CAPACITES_ACTIVES` | **bon** | Étape 1 = mesure pure, la faire tôt (elle attend depuis le 27/07). L'étape 2.2 (table → TOML) est LE MÊME travail que finalisation 3.2 : faire une fois, statuer `[~]` dans l'autre |
| `PLAN_OBJECTIFS_TEMPS_REEL` | **bon, un trou d'archi** | Chemin de persistance et capability à nommer avant l'étape 1 (§5.2-4). Étape 3 correctement conditionnée à la table J0.4. « L'étape 1 seule a déjà de la valeur » : vrai, et c'est la piste C |
| `PLAN_BELLE_CARTE_TRIANGLES` | **bon** | Reproduire avant porter, trancher les 2 réserves (seuil 5 cm, bornes par tag vs par maillage) AVANT le portage : exactement le bon ordre. Question Python réglée en §5.2-6. Étape 4.2 : mesurer le poids du champ d'altitude avant de choisir le pas (677 Ko aujourd'hui ; 5 cm ≈ 1,1 M cellules) |
| `PLAN_RECHERCHE_ASSETS_ICONES` | **bon** | Bien isolé, échec sans dette, licence = question utilisateur AVANT intégration. La règle « vérifier visuellement, jamais le nom de fichier » est déjà payée |
| `HANDOFF_PRECISION_PROJECTILES` | **excellent lot de recherche** | Une seule piste nommée (l'enregistrement qui CRÉE le slot du code 7), verdict binaire, critères anti-illusion (gain localisé, comparaison à l'unité, contraste intra-joueur). À timeboxer (§7) et à garder en MESURE pure jusqu'au verdict |
| `HANDOFF_KILLSOURCE_REPRISE` | **périmé sur les branches** | Corrigé en §2 ; son point 1 (gitleaks) est confirmé par la CI et traité en J1 |
| `HANDOFF_REPLAY_2D_2026-07-29` | **périmé sur le blocage** | Ses réserves ouvertes restent valables (cellule de munitions k, `ownersFromLives` sur collision — à reposer sur chaque nouveau film) |

---

## §6 — ANTI-RÉGRESSION D'UN DÉCODEUR BINAIRE — les pratiques retenues

L'intuition de l'utilisateur (tests à valeurs golden) est la bonne, et le dépôt la pratique
déjà aux bons endroits. Ce qui suit est le socle à généraliser en J2 :

1. **Goldens à deux étages.** Étage 1 : les ENTRÉES DÉCODÉES sérialisées (positions, tirs,
   morts…) rejouées dans l'assemblage pur — verrouille 475/519, 90/105, etc. sans un octet de
   film. Étage 2 : une MINI-BOBINE binaire réelle (~560 Ko : une image-clé + les 519 records +
   le chunk highlight) versionnée AVEC son outil de régénération — verrouille le décodage
   lui-même. Les films complets (20 Mo) et le cache (23 Go) restent hors dépôt.
2. **Le test « sans fixture » à côté du golden** (patron killsource) : il vérifie des
   invariants de FORME (structures non vides, bornes, cohérences) pour interdire que les
   goldens dégénèrent en nombres nus que plus personne ne sait relire.
3. **Jamais d'édition manuelle d'un golden.** Un golden se RÉGÉNÈRE par son outil, et le diff
   se relit comme du code. Un golden édité à la main est un mensonge qui passe les tests.
4. **Invariants de conservation testés** : publiés + rejets ventilés = total lisible (déjà en
   place : 475 + 44 = 519, somme exacte testée côté Go). À poser sur CHAQUE nouveau flux
   (objectifs, capacités…) dès son premier commit.
5. **Différentiel contre une source qui ne partage aucune pièce** : l'API (multiset des
   assistants à l'unité, `zone_captures + zone_secures`…), le relevé terrain, le Theater.
   C'est la règle des « deux chaînes indépendantes » du chantier — la codifier en test
   d'intégration quand la source est en base.
6. **`decoder_rev` sur toute ligne écrite en base** (patron du branchement) : re-décoder de
   façon ciblée au lieu de tout reprendre, et savoir QUELLE version a produit quoi.
7. **Fuzzing natif Go sur les lecteurs de records** (ajout recommandé, absent aujourd'hui) :
   le décodeur lit du binaire non maîtrisé ; un `go test -fuzz` avec la mini-bobine comme
   corpus de graines protège contre les paniques/lectures hors bornes à coût quasi nul.
   À poser en J2.2, pas avant (il faut la mini-bobine).
8. **Les règles maison déjà payées restent la loi** : jamais de balayage bit à bit par
   position (~99,85 % de faux positifs, il FABRIQUE des distributions crédibles) ; une mesure
   de concordance ne sert pas à la fois de score et de filtre ; mesurer avant de coder ;
   toujours DESSINER un résultat géométrique. Références : `README_KILLWEAPON_INDEX.md` §4,
   `V7.5/film_re/METHODE_RETRO_INGENIERIE_FILM.md`.

---

## §7 — EXÉCUTION EN PLUSIEURS CONVERSATIONS — le mode d'emploi

### Le modèle : un superviseur, des exécuteurs

- **1 conversation SUPERVISEUR** (courte, récurrente) : tient CE fichier, ouvre les sessions
  exécuteur avec leur ordre de mission, vérifie les gates sur pièces (CI, artefacts, SELECT),
  fait les merges, met à jour `SUIVI_REPLAY_2D.md` et la mémoire. Ne code pas.
- **1 conversation EXÉCUTEUR = 1 jalon ou 1 lot** (jamais deux lots en parallèle dans la même
  session — contrat `plan-execution`). Chaque session : recharge le skill, lit son plan
  d'autorité + §2 (corrections d'état), exécute, statue chaque item, journalise dans le plan
  ET dans `thought_log.md`, pousse la branche, rend la main avec un état net.

### La règle d'or du parallélisme — payée une fois, jamais deux

**Le décodeur (filmdec + analysis/replay + killsource + match-replay) = UNE seule piste à la
fois.** Deux mois de divergence sur deux branches sœurs ont coûté une réconciliation complète
(138 conflits analysés, une session entière) — c'est la leçon la plus chère du chantier.

- **Avant J5 (merge)** : tout est SÉQUENTIEL sur `feat/replay2d-prod` — J0 → J1 → J2 → J3 →
  J4 → J5. Deux exceptions sûres, parallélisables à tout moment : les captures UTILISATEUR de
  J0 (aucun fichier du dépôt) et la piste E (précision projectiles) si elle reste en mesure
  pure (outillage jetable + docs, aucun fichier livré touché) dans son propre worktree.
- **Après J5** : 3-4 exécuteurs en parallèle maximum, chacun sur SA branche courte depuis
  `main` et SON worktree, selon le tableau J6 (A exclusif sur le décodeur ; B, C, D, E
  disjoints). Merge < 1 semaine, jamais deux branches longues.
- Worktrees : créés depuis `main` UNIQUEMENT (jamais depuis une branche de chantier — le
  corollaire écrit dans le plan de branchement). Retrait : jonctions d'abord.

### Gabarit d'ordre de mission (à coller en tête de chaque session exécuteur)

```
Session exécuteur — [JALON/LOT] — branche [X], worktree [chemin]
1. Invoquer le skill plan-execution.
2. Lire .ai/PLAN_MASTER_FILM_KILLFEED_REJEU.md §2 (corrections d'état) et §[jalon].
3. Lire le plan d'autorité : .ai/[PLAN_*.md], section [X].
4. Exécuter les items dans l'ordre, gate par gate. Zéro fix hors périmètre
   (découvertes → section Découvertes du plan).
   TOUTE ACTION AU-DELÀ DES ITEMS LISTÉS = STOP et question au superviseur — y compris
   « améliorer », « compléter » ou « corriger une prémisse » de la mission. Une découverte
   se CONSIGNE, elle ne s'exécute pas. (Clause ajoutée après J0 : l'exécuteur a requalifié
   lui-même le partage humain/agent — résultat utile, mais le relevé terrain de J0.5 s'est
   perdu dans le débordement.)
5. Clôture : statuts posés, journal du plan + thought_log, SUIVI mis à jour si le
   périmètre rejeu a bougé, branche poussée, CI vérifiée (gh run list), compte rendu :
   fait / non fait / découvert / décisions en attente.
```

### Taille prévisible des sessions

| jalon | sessions estimées | notes |
|---|---|---|
| J0 | 1 (agent) + la soirée utilisateur | J0.1 d'abord — la fenêtre réseau/tokens |
| J1 | 1 | le diagnostic Linux (J1.3) peut déborder → session dédiée |
| J2 | 1-2 | 2.1+2.2 (goldens+bobine) d'abord ; 2.3-2.6 ensuite |
| J3 | 1-2 | A+B en une ; C (jugement) + D en une |
| J4 | 2-3 | briques+P1 · P3+P2 · P4+lot 1+3.1/3.2 |
| J5 | 1 | + le GO utilisateur et la fenêtre backfill |
| J6 | n | par piste ; A se découpe par plan |

---

## §8 — DÉCISIONS UTILISATEUR EN ATTENTE  *(tranchées AVANT la session concernée)*

| # | décision | jalon | éléments |
|---|---|---|---|
| 1 | Fenêtre des captures (soirée jeu + Cheat Engine) | J0 | 1 h au total, 15 min critiques |
| 2 | Garde local du rejeu : retrait en bloc, ou critère réécrit (plancher tous-films + date) | J4 | le critère actuel est atteint à la lettre et trop faible (CTF 80,3 %) |
| 3 | GO merge → prod + fenêtre backfill VPS (deux déploiements) | J5 | push main = deploy auto |
| 4 | Licence / redistribution des assets d'icônes extraits du jeu | J6-D | AVANT toute intégration |
| 5 | Le rejeu en prod : quel producteur d'artefacts (sync ? à la demande ? plus tard ?) | J6-F | coût 8-30 s/film, disque VPS, films expirants |
| 6 | Timebox de la piste E (précision projectiles) | J6-E | proposition : 2 sessions de mesure, verdict écrit, stop ou branchement |

**Tranchées le 2026-07-31 (avec l'utilisateur, session master plan)** :

- **#2 garde local** → « comprendre le CTF d'abord » : diagnostic de la perte des 564 tirs
  « slot introuvable » (contre 44 en Slayer), cause écrite, puis redécision. Le garde reste
  en place d'ici là (reporté dans J4.2).
- **#4 icônes** → extraire, **montrer avant d'intégrer** : gate humain entre les phases 1 et
  2 du plan de recherche (revue visuelle de la correspondance tag → image par l'utilisateur).
- **#5 rejeu en prod** → on collecte les données (manifests de film au sync — l'assurance
  contre l'expiration part avec le merge J5) ; les artefacts se feront **à la demande dans un
  premier temps**, gains et optimisations ensuite ; **pas de productionisation du rejeu pour
  l'instant — on reste en dev**. À instruire au passage (piste A ou F) : l'anomalie de coût
  du décodage BTB (11 min contre 8-30 s en 4v4) — « une telle différence suggère une faille
  logique » : profiler avant d'optimiser.
- **#6 précision projectiles** → timebox de 2 sessions de mesure, verdict écrit, stop ou
  branchement.

- **#7 palette Forge** *(apparue à J0.2)* → tranchée par le superviseur le 2026-07-31 au
  soir : copie `any` (417 Mo) + `ds` (2,14 Go) depuis l'install Steam vers la clé
  (`jeu_deploy_any/globals/forge/`, `jeu_deploy_ds/globals/forge/`) — c'est la seule voie
  restante pour `type_id → nom` sur les cartes Forge, et elle meurt avec le PC.
  **FAITE et VÉRIFIÉE** (tailles identiques à l'octet : 416 899 072 et 2 138 442 737).

- **#1 CLOSE le 2026-07-31 (soir)** : pas de nouvelle capture — les dumps sont faits, c'était
  eux la fenêtre ; l'ANALYSE des captures se fera plus tard, sur le nouveau PC, à la phase
  de traitement. Conséquence actée : l'étape 3.4 des capacités se fait en corrélation
  interne (3.3) seule — reporté en §2. Option de dernier recours, non planifiée : si le jeu
  est installé sur le nouveau PC pendant que le match `530820e5` est encore proposé au
  Theater, un relevé à l'œil reste possible.
- **#7 complétée** : sur accord utilisateur, la variante `pc` + `pc_hd1` est aussi sur la
  clé (`jeu_deploy_pc/globals/forge/`) — **FAITE et VÉRIFIÉE** (8 519 127 040 et
  5 776 809 226 octets, identiques aux sources). La palette Forge est intégralement
  sécurisée ; la résolution `type_id → nom` (power-ups, objets Forge) se fera à la phase
  d'analyse, sur le nouveau PC (décision #1).

**Reste ouverte** : **#3** (GO merge + fenêtre backfill VPS — à demander à J5 avec une date,
comme prévu).

---

## §9 — PROTOCOLE DE REPRISE DU MASTER PLAN

1. `git branch --show-current` (attendu : `feat/replay2d-prod` avant J5 ; après J5, la piste
   dicte la branche) et `gh run list --branch <branche>` — l'état CI d'abord.
2. Relire §3 (jalons) et le §10 (journal) : reprendre au premier jalon non clos.
3. Ouvrir le plan d'autorité du jalon, appliquer §2 (corrections d'état) s'il ne l'est pas déjà.
4. Ne JAMAIS exécuter deux jalons dans une même session ; ne jamais paralléliser hors §7.

---

## §10 — JOURNAL D'EXÉCUTION DU MASTER PLAN

- **[2026-07-31]** Master plan écrit. État vérifié sur pièces (git, disque, CI — §1).
  Découvertes : CI de branche rouge 4 signaux dont 3 non planifiés ; briques killsource
  absentes de la branche vivante ; ordre P2/P3 du branchement corrigé ; chantier « rejeu en
  prod » identifié comme non écrit. Aucun jalon exécuté — J0 est le prochain, et il presse.
- **[2026-07-31, suite]** Décisions #2/#4/#5/#6 tranchées avec l'utilisateur (§8). J0 confié
  à une session EXÉCUTEUR dédiée (ordre de mission remis à l'utilisateur) ; ce fil reste le
  SUPERVISEUR — vérification des gates au retour.
- **[2026-07-31, périmètre]** Ajout utilisateur : détection des RAMASSAGES (armes,
  power-ups, grenades) → piste A item 5. Vérifié par grep : la piste dormait, NOMMÉE, dans
  les archives (événement pickup 0x02-0x3c jamais décodé « pièce manquante » du chantier
  killweapon ; records WST de swap/pickup ~247/film déjà à moitié décodés). Nouvelle
  piste G — citations dérivées (G1 véhicules après J4, G2 ramassages après A5 sur le type
  canonique `MatchEventWeaponPickup` déjà servi par Halo 5, G3 objectifs en toute fin,
  bonus). J0 en cours d'exécution chez un exécuteur (Opus).
- **[2026-07-31, nuit] J0 CLOS avec deux réserves — gates vérifiés par le SUPERVISEUR sur
  pièces** : film `696a9d7c` 31/31 chunks (en-tête chunk 0 + footer type-3 compris) en cache
  ET sur la clé ; `deser_table.tsv` (47 Ko) sur la clé, contrôle « archétype 35 = 64
  composants » RECOMPTÉ ici (64) ; les deux captures CSV sur la clé (Strongholds 70,7 Mo
  AVEC relevé 4 ancres ; CTF `530820e5` 57,9 Mo SANS relevé) + bonus `archetype_vtables.tsv` ;
  0 commit non poussé sur les 4 branches. **Réserves** : (1) power-ups indécidables sans la
  palette Forge → décision #7, copie lancée ; (2) oracle surbouclier/camouflage manquant →
  décision #1 (option refaire J0.5). Découvertes reportées : route des objectifs par
  images-clés + footer (§2), trou de `fetch_film_chunks` (§J4), les 4 trous d'archétypes
  COMBLÉS par la table (objectifs 34, zones 33, véhicules 48, dispositifs 41). Note
  d'exécution : l'exécuteur a débordé son périmètre (requalifié la prémisse « utilisateur
  seul » et piloté Cheat Engine via MCP lui-même) — résultat excellent, mais le relevé J0.5
  est le coût exact du débordement ; clause d'arrêt ajoutée au gabarit §7.
- **[2026-07-31, J0 exécuté]** GATE J0 **passé sauf le volet power-ups**. J0.1/J0.3/J0.4/J0.6
  faits, J0.2 partiel (`level_id` prouvé, power-ups `[!]`), J0.5 capturé sans relevé. Prémisse
  corrigée : le bridge MCP `cheatengine` permet à l'agent de piloter Cheat Engine — J0.4-J0.6
  n'étaient pas « utilisateur seul ». Acquis irremplaçables : table des désérialiseurs
  (50 archétypes / 1068 composants, double contrôle passé), `archetype_vtables.tsv` (non prévu),
  deux captures continues de 1 205 704 et 988 752 records dont l'identité de film est PROUVÉE
  par alignement de signatures (`cmd/tmp_sigalign`, témoins négatifs à 0-1 %).
  **Résultat de fond** : `ti=11` dispatché en CTF, JAMAIS en Strongholds (0 / 1,2 M) → l'état
  des objectifs ne vit pas dans le flux delta ; il est dans les images-clés et le footer
  type-3, tous deux déjà en cache. Confirme et **corrige** `archive/V7/RESEARCH_THEATER_RE.md`
  §M (qui supposait la zone accessible en replication). Impact sur J6 : les pistes A4 et C
  (objectifs) partent avec une cible déplacée et une piste morte en moins.
  **Blocage à arbitrer** : la palette Forge (2,4 Go) n'est pas sur la clé — sans elle,
  `type_id → nom d'objet` meurt avec le PC (42 Go libres). Prochain jalon : **J1**.
- **[2026-07-31, nuit] J1 CLOS — GATE ATTEINT.** Les 4 items statués `[x]`. Tout est vert sur
  la branche sauf `Go Lint (golangci-lint)`, rouge assumé jusqu'à J3. Commits `3a9158ae4`,
  `566ebb777`, `dd020364a`, `4e2bd5dba`.
  **Ce que J1 a appris, et qui vaut plus que les correctifs** : les trois signaux avaient une
  parenté que la revue n'avait pas vue — *un outil qui s'arrête au premier obstacle fait
  passer sa cause unique pour un symptôme général*. (1) La suite Go Linux n'avait pas « des
  tests baseline absents » : elle ne compilait pas, à cause d'un en-tête MSVC dans le shim
  d'`ooz` ; les 8 786 tests « disparus » n'étaient que la conséquence. (2) Le ratchet lint
  n'était pas rouge de dette : il l'était de ce MÊME `intrin.h`, qui interrompait golangci
  avant tout le reste — la dette réelle (70 issues) n'est apparue qu'une fois la compilation
  réparée. (3) Le typecheck web disait « 26 erreurs de nullabilité » ; il y en avait 31 et une
  seconde cause (l'arité des tuples Go perdue par JSON Schema).
  **Découvertes reportées (non traitées)** : (a) **bug réel** — les tirs du rejeu dessinaient
  tous la forme par défaut : l'interface manuelle nommait `weapon` le champ que le contrat
  nomme `w`, donc `familyOf()` ne recevait jamais rien ; corrigé dans le même lot car il
  bloquait le typecheck, mais c'est une régression d'affichage vieille de tout le rejeu 2D
  qui mérite un coup d'œil visuel. (b) le ratchet **knip** dépassait son plafond (92 > 90)
  depuis le portage du rejeu 2D (`2044b7139`) sans que personne le voie — le hook pre-push ne
  se déclenche que sur `apps/web/**` et les derniers push étaient des documents ; corrigé au
  plus petit (deux exports rendus privés) parce qu'il bloquait le push. (c) `routeTree.gen.ts`
  est régénéré (ordre d'imports inversé) dès qu'une commande vite/vitest tourne — bruit
  d'outillage, écarté du commit, à surveiller comme faux diff récurrent.
  **Pour J3** : dimensionner sur **70** issues, pas 40 ; elles sont dans `analysis/filmdec`
  (composants non branchés) et `analysis/objectiveevents`. Et J2 (verrouiller le décodeur)
  passe AVANT, comme prévu : 33 des 70 sont des `unused` sur des lecteurs de composants qu'un
  nettoyage mécanique supprimerait alors qu'ils documentent le format.
- **[2026-07-31, nuit — superviseur] J1 VÉRIFIÉ ET CONFIRMÉ CLOS.** Contrôle au niveau JOB du
  run CI de `4e2bd5dba` (30659936684) : Frontend, Go Build+Test (ubuntu ET windows), Go
  Coverage + Baseline (18 min 22 — la suite tourne désormais réellement sous Linux), Contract
  Test, Lease Enforcement, OpenAPI Lint — tous verts ; seul job rouge : Go Lint (ratchet),
  assumé jusqu'à J3. gitleaks vert sur le dernier commit (`271d8f6c8`), arbre propre, commits
  poussés. Arbitrages rendus sur les découvertes J1 : (a) revue visuelle du correctif `w` →
  gate visuel de J4.1 ; (b) déclencheur du hook knip → J3.3 (ajouté) ; (c) faux diff
  `routeTree.gen.ts` → vigilance notée en J5.1. Re-dimensionnement J3 accepté (70 issues,
  prévoir 2 sessions) ; l'ordre J2 → J3 reste la protection des 33 `unused`. Prochain
  jalon : **J2**.
- **[2026-07-31, nuit] J2 CLOS — GATE ATTEINT.** Les six items du lot 2 statués. Le décodeur est
  verrouillé à DEUX ÉTAGES, et les deux sont nécessaires : l'étage 1 (entrées décodées figées,
  633 Ko) verrouille l'ASSEMBLAGE et est aveugle par construction à un changement de décodage ;
  l'étage 2 (mini-bobine de paquets réels, 698 Ko) verrouille le DÉCODAGE là où le premier ne
  voit rien. S'y ajoutent la grammaire d'inventaire (les 7 fonctions à 0 % — R1 à R4 testées par
  ce qu'elles REFUSENT), le rendu canvas sans navigateur (contexte enregistreur, zéro
  dépendance), les 3 décodeurs d'événements de `filmdec` et le second maillon du pont. Gate
  vérifié sur pièces : les 3 artefacts reconstruits rendent les chiffres du §5 de la
  réconciliation **à l'unité près** (475/519 · 1 862/2 154 · 2 312/2 879 ; 99/29 221 ; 90/105 ;
  70/70 ; 439 ; 184 ; 10 223), `go test replay+filmdec` vert, 3 372 tests web verts, `tsc -b`
  vert, ratchet knip sous plafond. Couverture `replay` : **58,5 % → 79,2 %**.
  **Ce que J2 a appris, et qui vaut plus que les tests** : *un décodeur qu'on n'a jamais
  interrogé hors de ses données ne dit pas ce qu'il fait sur les autres.* Les deux paniques (D1,
  D2) sont tombées à la PREMIÈRE exécution du fuzz — pas au bout d'une campagne — et la
  non-reproductibilité de l'artefact (D3) n'est apparue qu'en construisant deux fois de suite le
  même film, ce que personne n'avait fait en quatre mois de chantier.
  **Découvertes reportées (non traitées, D1-D5 du plan de finalisation)** : (D1) `decodeFireEvent`
  panique sur un payload de moins de 14 octets, et le chemin est ATTEIGNABLE — `ScanFilmFireEvents`
  n'exige que `Size >= 1` ; (D2) la tolérance de `PeekBits` est à sens unique alors que sa doc
  promet l'inverse ; (D3) **l'artefact n'est pas reproductible à l'octet** — traces, tirs,
  inventaire et couverture sont stables, mais les projectiles sont permutés d'une construction à
  l'autre (map + `sort.Slice` instable) et cela déplace 1 position de lancer sur 130 ; (D4) huit
  familles d'effet de tir pour SEPT géométries — `plain` est un balistique aminci ; (D5)
  `js-yaml` est une dépendance fantôme d'`apps/web` (déclarée en `overrides`, importée nulle
  part).
  **Prémisse corrigée** : l'item 2.6 décrivait « l'OpenAPI décrit 6 champs sur 22 ». C'est
  périmé — J1.2 avait déjà aligné les trois contrats, arité des tuples comprise. Ce qui manquait
  était le TEST, et c'est lui qui a été posé (des deux côtés, dont un prouvé par le compilateur
  et falsifié).
  **CI de branche vérifiée au niveau JOB** (run `30666349775`) : OpenAPI Lint, Go Build+Test
  (ubuntu ET windows), Go Contract Test, Frontend, Go Coverage + Baseline (`./...` complet,
  CGO=1), Go Lease Enforcement — tous verts ; gitleaks vert sur les trois push. Seul job rouge :
  `Go Lint (golangci-lint)`, assumé jusqu'à J3 — et le compte est le contrôle qui vaut : **70
  issues, exactement le chiffre de J1.4, dont AUCUNE ne vient des fichiers de J2**. Le ratchet
  est rouge de la dette connue, pas du jalon. Prochain jalon : **J3**.
