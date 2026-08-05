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
      - ~~**BLOCAGE IDENTIFIÉ, à arbitrer**~~ — **LEVÉ le 2026-08-01**, par la palette Forge
        enfin lue. État de l'art : `.ai/ETAT_DE_L_ART_FORGE_PALETTE_ZONES.md`.
        - `[x]` **La palette est résolue à 99,0 %** (2758/2785 `type_id` des 199 cartes,
          contre 45 auparavant) ; Catalyst **36/36**, Vagabond **479/479**. Contrôle positif
          **45/45 identiques** (groupe ET dimensions) sur `forge_object_types.csv`.
        - `[x]` **Le volet power-ups est CLOS — par la négative, et c'est un résultat.**
          `eqip` = **3 types sur 2785**, portés par **5 cartes sur 199**, aucun sur Catalyst
          ni Vagabond. La variante de carte **ne place pas d'équipement** : surbouclier et
          camouflage viennent du mode de jeu ou du scénario de base, pas du `.mvar`. La
          question A2 n'a donc pas de réponse dans ce fichier — elle n'y est pas écrite.
        - `[!]` **Le NOM d'objet reste indécidable hors ligne** : table des chaînes vide sur
          **88/88 modules**, aucun nom dans le tag `forg` (753 Ko), `GlobalID ≠ murmur3` du
          chemin de tag (testé sur 2 identifiants de niveau connus × 6 formes). Un `type_id`
          inconnu reste SANS NOM. Seule la CLASSIFICATION (groupe de tag) est décidable.
        - **Correction à porter** : l'emprise de la palette est en **unités monde**
          (× 3,048), pas en mètres — établi sur les véhicules (Pelican 32,7 m, Warthog
          6,8 × 3,1 × 2,5 m). Les positions et formes du `.mvar` sont en mètres.
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

- [x] J3.1 Purger le cache golangci avant toute conclusion (piège déjà payé sur ce dépôt :
      un cache chaud rend un faux compte).
      *FAIT 2026-08-01 — purge avant la mesure d'entrée ET avant la mesure de sortie.*
- [x] J3.2 Lot A : supprimer aussi le jeu d'images de grenades orphelin (`Dynamo-light.png`,
      casse majuscule — découverte n°1 de la réconciliation, aucun consommateur).
      *FAIT — 8 PNG (4 types × light/dark). `index.json` ne désignait que le jeu minuscule.*
- [x] J3.3 Élargir le déclencheur du hook pre-push `knip-ratchet` (découverte J1-b : il ne se
      déclenche que sur `apps/web/**` — un dépassement de plafond est resté invisible tant
      que les push étaient documentaires). Correctif honnête au choix : déclencher aussi sur
      la config/le plafond knip, ou jouer le ratchet en CI sans condition de chemin.
      *FAIT 2026-08-01, par la SECONDE voie — et la première n'aurait rien réglé : le mode
      d'échec constaté n'est pas « le push ne touchait pas de source web », c'est « la branche
      n'a jamais été poussée », qu'aucun glob ne rattrape. Le ratchet est un step du job
      `frontend` de `ci.yml`, sans filtre de chemin ; le hook local reste en filet rapide et
      son commentaire dit qu'il n'est pas l'autorité. Vérifié depuis `apps/web` : sort en 0.*
- [x] J3.4 **Gardes du décodeur** (découvertes J2-D1/D2, arbitrées le 2026-07-31) :
      (a) `decodeFireEvent` panique sur un payload < 14 octets et le chemin est atteignable
      (`ScanFilmFireEvents` n'exige que `Size >= 1`) — poser la garde de longueur, auditer
      les décodeurs frères (lancers, projectiles) pour le même motif, et conserver l'entrée
      de crash du fuzz comme corpus de régression ; (b) aligner `PeekBits` sur son contrat
      documenté (tolérance à sens unique aujourd'hui). **AVANT le merge** : J4 met le
      décodeur dans un collecteur en tâche de fond du process sync — une panique sur film
      malformé y coûterait le process entier. Sous protection des goldens J2.
      *FAIT 2026-08-01. (a) Garde de longueur posée (`decodeFireEvent` rend `ok bool`), le
      harnais de fuzz appelle désormais les lecteurs avec EXACTEMENT le contrat de production
      (ses deux contournements retirés), la graine `seed_04` — troncature à 3 octets d'un
      payload de tir, produite par `collectFuzzSeeds` donc régénérable — est l'entrée de crash
      gardée en régression, et deux tests dédiés figent les deux seuils. **Audit des frères :
      ils sont SAINS** — les scanners de lancers et de projectiles bornent leur balayage ET
      lisent par `PeekBits` ; `offline_aim` teste `at+n > total` avant chaque composant ;
      `offline_biped` et `i0_layout` bornent leur boucle sur `total`. `decodeFireEvent` était
      le seul à lire à offsets FIXES derrière une garde de taille qui ne les couvrait pas.
      (b) `PeekBits` aligné : sa tolérance valait à la fin, pas au début.*
- [x] J3.5 **Tri total des projectiles** (découverte J2-D3) : l'artefact n'est pas
      reproductible à l'octet (map + `sort.Slice` instable, 1 position de lancer sur 130
      bouge). Un tri total rend l'artefact ET le fixture déterministes — au service des
      goldens, du diff, et du cache prod futur.
      *FAIT 2026-08-01 — **et la cause annoncée n'était que la moitié de l'histoire**. Le diff
      champ par champ de deux constructions montre exactement DEUX champs mobiles :
      `projectiles` a le même multi-ensemble à l'ordre près (instabilité de tri, comme prévu),
      mais **`grenades` change de VALEUR** — le lancer `t=1580` de `01e1f945` sort à
      (11,41 ; 17,99) ou à (12,72 ; −187,11). Chaîne unique : l'itération de la map de
      `ScanFilmWorldObjects` fixe l'ordre d'arrivée, un tri sur le seul instant laisse cet aléa
      départager les naissances de projectile EX ÆQUO, et `birthNear` prend la naissance d'un
      INDICE donné — l'aléa choisissait la position publiée. Quatre tris rendus totaux.
      **Preuve, les deux volets** : deux constructions successives de chacun des trois films
      rendent des octets identiques ; et le fixture d'entrées des goldens, régénéré deux fois,
      rend la même empreinte — alors qu'il DIFFÉRAIT de la version committée (celle-ci n'était
      qu'un tirage). Fixture stabilisé committé ; sortie figée d'assemblage INCHANGÉE.*
- [~] J3.6 Retirer la dépendance fantôme `js-yaml` d'`apps/web` (découverte J2-D5).
      Découverte J2-D4 (huit familles d'effet pour sept géométries, `plain` = balistique
      aminci) : **AUCUNE action** — c'est le repli sobre documenté (`ETAT_DU_POC.md`).
      **REFUSÉ le 2026-08-01, prémisse fausse** : ce n'est pas une dépendance fantôme mais un
      correctif de sécurité (commit `54a6eb3df`, **CVE-2026-53550 / GHSA-h67p-54hq-rp68**).
      `@redocly/openapi-core` épingle `js-yaml` en exact `4.1.1` (vulnérable) ; l'`overrides`
      est le seul levier qui hisse le lockfile à `4.3.0`. Le retirer réintroduirait la faille
      et rouvrirait l'alerte Dependabot #3. **Décision superviseur RENDUE le 2026-08-01 :
      ne rien faire, statut `[~]` déjà couvert** — vérifié sur pièces (commit `54a6eb3df` +
      `package.json`) ; l'intention réelle de J2-D5 (un test indépendant des paquets tiers)
      est déjà appliquée (`replayContract.test.ts` passe par les types générés). Condition
      de retrait futur de l'override, à re-vérifier aux montées Dependabot : les
      consommateurs requièrent nativement js-yaml >= 4.2.

**Découpage arrêté** : **J3-1 CLOSE le 2026-07-31 (70 → 43)** = lots A + B + J3.1/J3.2
(+ J3.6 clos sans retrait, cf. ci-dessus) ; **J3-2** = lots C + D + J3.3/J3.4/J3.5 + les
3 `revive argument-limit` renvoyés par J3-1 (regrouper les paramètres de décodage en
structure : `kfBetterCand`, `fire_scanner_v3.go:120`, `killsource/bijection.go:259`).

**À savoir pour J3-2** (legs de J3-1) : le lot C compte **34** verdicts, pas 33 (une
position portait revive + unused superposés — `consumeForgePlayerDataEditedObjectsIDs`) ;
et le gate des 7 grandeurs du §5 doit vérifier l'inventaire PAR LECTURE DE L'ARTEFACT
(`jq '.inventory|length'`) — `replay-build` ne journalise pas cette grandeur, sans ce
relevé elle n'est pas vérifiée du tout.

> **J3-1 CLOS le 2026-08-01.** Lots A et B faits, J3.1/J3.2 faits, J3.6 **refusé sur prémisse
> fausse** (sécurité — voir l'item). **Mesure 70 → 43.** Les 7 linters mécaniques sont à zéro ;
> restent 34 `unused` (et non 33 — un `unused` était masqué par un `revive` à la même
> position), 4 `staticcheck`, 3 `revive argument-limit`, 2 `gocyclo`.
> **J3-2 hérite d'un item de plus** : les 3 `revive argument-limit`, écartés de la passe
> mécanique parce que rentrer sous le plafond demande de regrouper des paramètres.

> **J3-2 CLOS le 2026-08-01 — ET J3 AVEC LUI. 43 → 0.** Lot C (les 34 `unused` statués un par
> un : **0 cas (a)**, 32 retirés, 2 gardés sous `//nolint` daté à condition de retrait), lot D
> (`gocyclo` découpés ; les 4 `staticcheck` lus d'abord — **aucun défaut de logique**), les 3
> `argument-limit` regroupés en structure, J3.3/J3.4/J3.5 faits.
> `golangci-lint --new-from-merge-base=origin/main`, cache purgé : **`0 issues.`** — c'est la
> cible **F1** atteinte. `go test ./...` vert. Les 7 grandeurs des 3 films identiques à
> l'unité près à chaque gate, et l'artefact est désormais **reproductible à l'octet**.
> **Restent pour J5** : F2 (re-merger `origin/main` puis RE-MESURER — la base du ratchet bouge
> avec `main`) et tout le lot E, plus F3 (prévenir avant le push sur `main`).
> **Découverte à traiter en priorité, hors périmètre J3** : un garde de CI ne se déclenche
> jamais (« Guard — feedback-drawer », chemin doublement préfixé sous
> `working-directory: apps/web`) — détail au journal du plan de dette, découverte n°3.

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
*FAIT — J4 session 1 (2026-08-01). Les 2 migrations, les 2 persisters, le pont ET le collecteur
sont écrits contre le code actuel ; GATE 1 atteint sur film réel. La réécriture a démenti trois
énoncés recopiés de l'archive (cache sans kill-feed, `splitSQL`, couverture du DELETE nu) —
détail au journal du plan de branchement.*

**Correction 2 — inverser les phases 2 et 3.** Le plan enchaîne P1 collecteur → P2 bascule
des 8 lecteurs (vue) → P3 backfill. Mais basculer les lecteurs sur une vue posée sur une
table VIDE casserait les pages carrière entre P2 et P3. Ordre corrigé :

1. **P1** — le collecteur (tâche de fond, capability `film_kill_source`, jamais dans une
   requête HTTP, un décodage à la fois, limite de temps + compteur d'abandons, expvar).
2. **P3** — le backfill LOCAL d'abord (sous-commande `levelup`, reprenable par `decoder_rev`,
   BTB en dernier), jusqu'à couverture des matchs porteurs de films.
   *FAIT — J4 session 2 (2026-08-01). `levelup backfill-killsource`, hors ligne par
   construction (aucun client HTTP derrière la source de chunks). Les DEUX producteurs sont
   câblés avant la passe, comme l'exigeait l'arbitrage : les tirs par arme sortent de la MÊME
   passe de décodage que les morts, et le producteur credit-seul (SQL → SQL) couvre les matchs
   dont le film a expiré, avec préséance film > credit testée dans les deux ordres d'arrivée.
   Le mur de coût a été PROFILÉ puis CORRIGÉ sous double gate (artefacts bit-identiques) :
   1 145 s → 46,7 s sur le plus gros film.*
3. **P2** — la bascule des 8 lecteurs (+ les 2 sites spéciaux S1/S2), gate AVANT/APRÈS sur
   données réelles. Les deux arbitrages du plan tiennent (bots exclus des agrégats — attention
   Q26 ne filtre pas les bots aujourd'hui ; BTB inclus en cumul, interdit ligne à ligne).
   *PARTIELLE — J4 session 3 (2026-08-02). L'inventaire était incomplet de moitié (**20** sites
   de lecture, pas 8). Trois préalables ont été livrés : le producteur LIVE des kill-events pour
   les deux titres (les deux producteurs existants étaient hors ligne — zéro appel depuis
   `api/`, `sync/v2/`, `cmd/server/`, `service/`), la reprise dédupliquée de l'ancienne table, et
   la suppression de `v_killer_victim_full`. **La bascule elle-même est ARRÊTÉE par une mesure** :
   la passe de film ne publie que **74,4 %** des morts de l'oracle API là où l'ancienne table en
   porte **98,4 %** — la remplacer effacerait 25 697 morts sur 949 matchs, sans erreur ni
   compteur. Le corollaire était écrit dans le DDL depuis la session 1 ; ce qui manquait était sa
   magnitude. GATE 2 passé au sens strict : les 11 mesures sont IDENTIQUES avant/après sur les
   deux titres.*
4. **P4** — l'exposition produit (capabilities par famille, i18n FR+EN, tokens). Le piège
   MA40/Sidekick du §4.2 est NON NÉGOCIABLE : jamais de taux par arme sur corpus entier.
   *FAIT — J4 session 4 (2026-08-02), pour ce qui était armable sans rouvrir P2.* Trois familles,
   trois clés : `film.kill_source` (déjà là), **`film.weapon_shots` (nouvelle)** et
   `match.weapon.accuracy`. La nouvelle clé a un CONSOMMATEUR testé sur film réel dans les deux
   sens (kill_source seule → 90 morts et 0 tir ; les deux → 90 morts et 30 tirs). La raison du
   `not_exposed` de la précision était PÉRIMÉE (« table non peuplée ») : réécrite sur le vrai
   motif — le taux sur corpus entier inverse l'ordre MA40/Sidekick — avec son critère de bascule.
   **Aucune surface produit ouverte**, conformément à l'arbitrage : elle se viderait.*

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

- [x] J4.0 **Réparer le garde de CI mort** (découverte J3-2 n°3, arbitrage superviseur du
      2026-08-01) : « Guard — feedback-drawer ne doit pas importer le wrapper api » grepe
      `apps/web/src/features/...` **sous `working-directory: apps/web`** — le chemin est
      doublement préfixé, `grep` sort en erreur sur un fichier absent, et un `if` sur une
      erreur est faux : le garde passe toujours, sans rien vérifier. **Vérifié le 2026-08-01 :
      l'invariant TIENT** (le fichier n'importe pas `api` et pose `credentials: 'omit'`) —
      c'est donc un trou latent, pas une fuite active, mais il partirait sur `main` en J5.
      Trois gestes, pas un : corriger le chemin ; **rendre le garde auto-vérifiant** (échouer
      bruyamment si le fichier cible n'existe pas — patron du dépôt,
      `TestHalowaypointAllowlistEntriesPointToExistingFiles`) ; **auditer les autres gardes
      CI par grep** pour le même motif. Leçon généralisable : *un garde qui ne peut pas
      échouer ne garde rien*.
      *FAIT 2026-08-01 (J4 session 1).* Chemin corrigé (`src/features/...`, relatif au
      `working-directory`), contrôle d'existence AVANT le grep, et les **trois**
      comportements PROUVÉS en rejouant le script du step tel que l'extrait le parseur
      YAML : arbre réel → rc=0 ; cible absente → rc=1 avec `::error::` ; cible présente
      important `api` → rc=1. La regex elle-même n'avait jamais été exécutée : vérifiée
      sur trois formes d'import positives. **Audit des autres gardes : mécanique, pas au
      jugement** — un détecteur (tous les `run:` de tous les workflows × leur
      `working-directory` effectif × les chemins du dépôt qu'ils citent) donne **1 avant
      correction, 0 après**, et le témoin de détection a été joué sur la version d'avant
      pour prouver que le détecteur voit ce défaut-là. Découverte annexe consignée au
      journal du plan de branchement (deux `--if-present` : même motif, latent, non traité).
- [x] J4.1 Lot 1.1-1.3 : supprimer `feature-flags.ts` (mort depuis avril), publier
      `replay_available` (un `os.Stat` via `PathResolver`), poser le lien conditionnel
      (i18n FR+EN). En prod, l'artefact n'existe pas → pas de lien : inoffensif d'ici le
      chantier F. **Le gate visuel du lot 1 couvre AUSSI le correctif J1-a** (les familles
      d'effets de tir enfin différenciées — champ `w`) : première revue d'écran depuis que
      ce bug, vieux de tout le rejeu 2D, est corrigé.
      *PARTIEL — J4 session 3 (2026-08-02) : **1.1 FAIT** (fichier supprimé, plafond knip
      abaissé de 29/90/86 à **0/0/0** — le compte réel était déjà nul, le ratchet ne ratchetait
      plus). **1.2 et 1.3 NON TRAITÉS** : la phase 2 a absorbé la session (la mesure qui a
      arrêté la bascule ne pouvait pas se deviner sans l'implémenter). Donc **AUCUN gate visuel
      à demander pour l'instant** — le lien vers le rejeu n'existe pas encore, et la revue du
      correctif `w` reste due avec lui.*
      *FAIT — J4 session 4 (2026-08-02). **1.2** : `port.ReplayService.IsAvailable` (un `os.Stat`,
      le MÊME service que l'endpoint `/replay` — une seule résolution de chemin dans le dépôt) →
      header `replay_available`. **1.3** : `ReplayLink` en tête de la rangée d'actions, FR
      « Rejeu 2D » / EN « 2D replay », 4 tests dont l'absence sur un match sans artefact.
      **ET LE LIEN N'AURAIT JAMAIS PU APPARAÎTRE** : l'outil hors ligne écrit l'artefact sous la
      forme COURTE du `match_id`, l'app ne manipule que la forme COMPLÈTE — le service cherchait un
      fichier que personne n'écrit, et `GetReplay` rendait un 404 sur un artefact PRÉSENT (les 3
      artefacts du dépôt sont tous en forme courte). Correctif : `title.FilmShortMatchID`, une
      seule troncature dans le dépôt, garde-rail au 3e exemplaire. Le défaut a été trouvé en
      cherchant l'URL à donner à l'utilisateur pour le gate visuel — le gate a donc servi AVANT
      d'être joué. **GATE VISUEL DÛ : artefact de revue produit, NON déclaré passé.***
- [ ] J4.2 Lot 1.4 : le garde local — TRANCHÉ le 2026-07-31 : **comprendre le CTF d'abord**.
      Diagnostiquer pourquoi `64e8adfa` perd 564 tirs « slot introuvable » là où le Slayer en
      perd 44 (hypothèse vies courtes → traces non publiées, à mesurer, pas à supposer),
      écrire la cause, PUIS redécider retrait/critère avec l'utilisateur. Le garde reste en
      place d'ici là.
- [x] J4.3 Lot 3.1 + 3.2 (title-agnostic minimal AVANT merge) : unifier les deux tables de
      grenades (« Dynamo » vs « Shock » pour le même rang — une contradiction visible à
      l'écran), sortir armes/grenades/capacités vers
      `config/titles/halo_infinite/mappings/*.toml` (`weapon_names.toml` existe, bilingue),
      factoriser les 4 copies de `keep*OfPublishedTracks` avec garde-rail, géométrie via
      `PathResolver` (le défaut actuel pointe `.ai/V7.5/dumps` — un répertoire de
      rétro-ingénierie). Motif : ne pas MERGER vers `main` de nouvelles violations des règles
      du dépôt (libellés FR en dur côté Go). Les lots 3.3-3.6 (découpages) restent post-merge.
      *FAIT — J4 session 4 (2026-08-02). La contradiction des grenades était RÉELLE et mesurée sur
      `000d5950` : le lancer publiait « Shock » quand le compteur porté de la MÊME fiche disait
      « Dynamo ». Le décodeur ne nomme plus rien (liste ORDONNÉE de tags) ; le lancer publie son
      RANG, index dans la table du titre — un index ne peut pas diverger de sa table. Les
      catalogues (armes, grenades, capacités, effets de tir) sont dans
      `config/titles/halo_infinite/mappings/` et BILINGUES, `SchemaVersion 1 → 2` ; côté web
      `shotEffects.ts` ne connaît plus une seule arme Halo. Les 4 copies de `keep*` sont
      factorisées avec garde-rail (témoin de détection joué), la géométrie passe par
      `PathResolver.MapGeometryDir`. **3 libellés CHANGENT et ce sont des corrections** : l'énum du
      décodeur nommait `0x9D6AAED2` « M41 SPNKr » (SPNKr à combustible) et `0x3E070217` « Pulse
      Carbine » (carabine Vestige). `mapvar` `[!]` : hors périmètre fermé de la session.
      **Les 3 artefacts reconstruits et comparés champ par champ** : tout ce qui n'est pas un
      libellé est IDENTIQUE, y compris les 70/130/153 lancers hors leur type.*

**ÉCART DES GOLDENS KILLSOURCE — instruit le 2026-08-02 (session 2bis), verdict ci-dessous.**
`TestGoldenFilms` échoue sur les 4 films de référence. **L'hypothèse du superviseur est
réfutée** : le correctif de performance de J4-2 (`Skip` au lieu de la boucle plafonnée) n'y
est pour rien — la divergence est déjà présente, signature identique, à `96bc56175` (J2), et
`Skip(8N)` est strictement équivalent à N × `ReadBits(8)`, plafond compris. **La cause est
`47c9e72ac`** (« réunir les deux décodeurs filmdec — fusion à trois voies »), dont le message
annonce « killsource (golden) vert » : le test avait SKIPPÉ, faute de `KILLSOURCE_FIXTURES`.
Sous-cause prouvée : le `br.ReadBits(2)` ajouté dans `consumeAbsoluteWithGate` (champ « fini »,
FUN_14076e304, mesuré par capture CE sur 100 % de 154 158 dispatches, déjà lu par le chemin
world-object). La calibration l'absorbe en rétrécissant l'axe — `3·axisW + indexW` perd
exactement 2 sur les QUATRE films (43→41, 53→51, 50→48, 52→50) ; neutraliser cette seule ligne
restaure `axisW=17 indexW=2` sur 9b191a7f. **L'oracle ne départage pas** : comparaison ligne à
ligne des sorties JSON complètes (état figé `5b5f97c69` contre HEAD), **371 lignes publiées sur
371 identiques** — même victime, même tueur, même tag, même étiquette, même catégorie ; 380/380
morts de l'API et 30/30 ancres Theater inchangées des deux côtés. Seules bougent la chaîne de
calibration, les compteurs par voie, et le champ `voie` de **4 lignes sur 371** (1,1 %).
**TRANCHÉ par le superviseur le 2026-08-02 : re-congeler** — la consigne « restaurer » reposait
sur une prémisse tombée (il n'y a pas de départage arbitraire qui bouge, il y a un décodeur qui
lit un champ qu'il ne lisait pas). *FAIT : 13 lignes sur les 4 goldens de film, et
`cumul.golden` INCHANGÉ — la ventilation agrégée est identique (marche 340/340, scan 37/37,
appariés 334 et 29), seul le partage par film a bougé.* **Backfill : RIEN À REJOUER** — les
124 694 lignes ont été produites par le décodeur actuel, dont les lignes publiées sont
identiques à celles du décodeur figé.

- [x] J4.4 **Le golden en CI, sur une mini-bobine versionnée** (correctif structurel du défaut
      ci-dessus — 3e occurrence après le garde feedback-drawer J4.0 et le ratchet knip J3.3).
      *FAIT 2026-08-02.* `TestGoldenMiniBobine` tourne **sans fixture et sans variable
      d'environnement**, donc dans le `go test ./... -p 1` du job de couverture, en 0,63 s.
      La bobine (`testdata/minibobine_000d5950`, 3,8 Mo) est un **préfixe CONTIGU** des 6
      premiers chunks du film `000d5950`, en-tête compris, suivi de son chunk HIGHLIGHT —
      contigu et depuis le début **parce que le décodeur l'exige** : le patron de la mini-bobine
      de J2 (paquets cherry-pickés) y décode **zéro** mort, mesuré (644 Ko → 0 candidat ;
      préfixe 00-03 → 6 ; préfixe 00-05 → 10). Le golden fige **les 10 lignes publiées en
      entier** — tag, étiquette, catégorie, crédit, voie, origine, divergence — dont **trois
      ancres Theater** (00:35 Disruptor, 00:44 Skewer, 01:12 M41 SPNKr) et **la ligne de
      divergence 01:12**. Trois gardes, pas un : la bobine absente ou tronquée échoue
      bruyamment (comptage de chunks), un plancher de 10 lignes publiées interdit de figer une
      sortie vide, et la recette de fabrication est un test exécutable qui retrouve le chunk
      HIGHLIGHT **par son contenu** (n27 ici, n62 en BTB) — régénération vérifiée
      **byte-identique**. **Témoin de détection joué** : en neutralisant le `br.ReadBits(2)`,
      la sortie de la bobine change (`recordStateParam` 0→2, croissance 1.000→1.004) —
      **le canal de détection est la ligne de calibration, pas les lignes publiées**, et c'est
      écrit dans le fichier. Portée déclarée : la calibration tombe en PROFIL PLAT sur un
      préfixe, donc ce garde ne verrouille pas le balayage — seul J4/`TestGoldenFilms` le fait.

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
- [ ] J5.1bis **Fermer le ratchet knip** (découverte J3-2 n°4, arbitrage superviseur du
      2026-08-01) : les plafonds valent `29/90/86` pour un compte réel de `0/0/0`. Les
      abaisser à la valeur mesurée — **ici et pas avant** : la surface web n'est stable
      qu'au merge, et un plafond à 0 pendant J4 bloquerait un push intermédiaire légitime
      (un export posé dans un commit, son consommateur dans le suivant). Les sessions J4
      surveillent le compte pour qu'il ne dérive pas d'ici là.
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
| **B — cartes** *(c'est ici que Catalyst et Vagabond obtiennent leur fond de carte — étapes 5.2 et 5.3)* | `PLAN_BELLE_CARTE_TRIANGLES` (étapes 0.1 → 6) puis FINALISATION lot 4 | `himap/`, `himodule/`, `cmd/mapstruct-build`, `mapFloor.ts`, `data/.../reference/` | aucune (modules du jeu sur la clé) |

**État des fonds de carte au 2026-08-01, mesuré** (question utilisateur) : **2 cartes sur 14**
ont un sol reconstruit — `ridgeline.json` (Cliffhanger) et `sgh_streets.json` (Streets).
**Catalyst n'en a pas** (ses deux artefacts, `01e1f945` et `64e8adfa`, portent
`structure = 0` : le rejeu y tourne sans sol, c'est la dégradation prévue) ; **Vagabond n'a
ni bornes, ni sol, ni artefact** — seul son module est prouvé (`fo08_wetland`, J0.2). La
seule image de carte produite à ce jour est Cliffhanger
(`E:/LevelUp_rejeu2D/scratchpad_recherche/C_carte_triangles.png`, plus les PNG de
`.ai/V7.5/dumps/`). `catalyst_overlay.png` **n'est pas un rendu de carte** : c'est une figure
de recherche qui documente une RÉFUTATION (« le critère géométrique est réfuté sur la vérité
terrain »), ses rectangles gris sont des boîtes englobantes et son panneau Catalyst porte la
mention « origine arbitraire ». Ne pas la citer comme une carte.
**Sous-produit à regarder en OUVERTURE de B2** : `.ai/V7.5/dumps/map_forge_shapes.png` — le
nom promet des formes d'objets Forge, donc peut-être une partie de la réponse à la question
des zones. Non ouvert par le superviseur : à vérifier, pas à supposer.

**CRITÈRE VISUEL DE LA PISTE B — arbitré avec l'utilisateur le 2026-08-01.** Le résultat
attendu est **`.ai/V7.5/dumps/carte_validee_v1.png`**, l'image validée le 2026-07-26 : une
vue du dessus où l'**architecture réelle** se lit (fer à cheval en anneau, structure
circulaire centrale, plateformes hexagonales, deux ponts au sud), où la **roche se distingue
des plateformes construites**, et où **aucun rectangle** n'apparaît. La règle d'arbitrage est
écrite en tête de `PLAN_BELLE_CARTE_TRIANGLES.md` : **le visuel commande, le poids s'adapte**
(tuiles, compression, niveau de détail) — jamais l'inverse. Motif de l'ajout : l'étape 4.2 du
plan demandait d'« arbitrer le pas et le poids » sans dire à quoi le résultat devait
ressembler — un exécuteur optimisant le poids aurait choisi 25 cm de bonne foi et produit une
carte en gros blocs **en respectant le plan à la lettre**. Ajouté aussi : un item 4.2bis, un
champ d'altitude seul peut ne pas porter les arêtes ni la différence de matière de la
référence — à trancher sur l'image, pas sur une intuition de format.

**GATE HUMAIN DE LA PISTE B — exigé par l'utilisateur le 2026-08-01.** Toute étape qui
produit une image, et **chaque carte** nouvellement traitée, doit produire un **artefact de
revue** : une page publiée portant, par carte, le rendu et `carte_validee_v1.png` **côte à
côte à la même échelle**, la couverture, et la checklist en **deux parties** : (a) critères
GÉNÉRAUX valables partout — architecture lisible, terrain distinct des plateformes
construites, aucun rectangle, échelle et orientation cohérentes avec les positions de
joueurs, couverture publiée ; (b) **témoins PROPRES À LA CARTE**, nommés AVANT la revue.
L'anneau du fer à cheval et les deux ponts sont les témoins de **Cliffhanger uniquement** —
pour toute autre carte, l'utilisateur (ou une source externe : carte en jeu, rendu Reclaimer)
donne deux ou trois repères attendus, et la session ne choisit JAMAIS ses propres témoins
après avoir vu son rendu. La
session **rend la main et attend une validation EXPRESSE** — aucune revue visuelle ne se
déclare passée par la session elle-même, et un `[x]` posé sans validation écrite au journal
est invalide. Détail opératoire dans `PLAN_BELLE_CARTE_TRIANGLES.md`, section « GATE
HUMAIN ». **Ce gate est le modèle à reprendre pour tout livrable dont le juge est l'œil** —
notamment le lot 1 du rejeu (J4.1) et les retouches visuelles (étape 6 de la piste B).

**LES NOMS DE ZONES (CALLOUTS) ENTRENT DANS LA PISTE B — 2026-08-01, demande utilisateur.**
Ils n'y étaient PAS : ils dormaient au backlog rang 4 (« non planifié »). Ils y entrent comme
**étape 5bis** parce que c'est le même tag, les mêmes cartes, la même passe — et parce qu'ils
portent le critère (a) du gate visuel : une carte sans ses noms reste un dessin.
**Découverte matérielle URGENTE, traitée le jour même** : l'investigation du 2026-06-26
établit que les noms de zones ne sont **ni dans le film, ni dans la variante `ds/`** (celle
qu'on utilise pour la géométrie) mais dans la variante **`any/`** — or la clé ne portait que
`ds/levels/multi`. Les 31 modules de `any/levels/multi` (971 291 037 octets) y ont été copiés
et **vérifiés conformes** : sans ce geste, le lot callouts serait mort avec le PC, exactement
comme la palette Forge l'aurait été (décision #7). **Leçon à retenir pour les autres lots
« plus tard » : vérifier que leur MATIÈRE PREMIÈRE est sur la clé, pas seulement que le plan
existe.**
| **B2 — la palette Forge : NOMMER et MESURER** *(ajoutée le 2026-08-01)* | session de recherche dédiée (prompt superviseur) | `mapvar/`, `himodule/`, `cmd/mapobj-build`, `map_objectives.json`, plus tard le rendu | palette Forge (sur la clé ET sur `D:` depuis le 31/07) ; **prérequis de tout affichage d'objectif** |

**ÉTAT DE B2 AU 2026-08-01 — travail EN COURS sur une seule de ses questions.**

| question B2 | état |
|---|---|
| **Q2 — la FORME des zones** | **EN COURS** — travail parallèle démarré le 2026-08-01 (hors de ce fil superviseur) |
| Q1 — NOMMER (`type_id → nom` : power-ups, armes, équipements, socles) | **non couvert** par ce travail — reste à faire |
| Q2bis — les champs du record déjà parsés mais **jamais extraits** (échelle, **délai de réapparition**, ordre d'apparition, présent-au-départ, forme/taille) | **non couvert** — reste à faire |

**Deux conséquences opérationnelles :**

1. **Q2bis se fait dans la MÊME passe que Q2, ou elle coûtera deux fois.** Les deux lisent le
   même record d'objet dans `mapvar` ; la session qui y est déjà n'a qu'à énumérer les champs
   non extraits pendant qu'elle y est. **Transmettre Q2bis au travail en cours** est le geste
   le moins cher — sinon il faudra rouvrir le même fichier plus tard, avec un contexte à
   reconstruire.
2. **Ne PAS lancer Q1 ni Q2bis en parallèle** du travail en cours : même surface
   (`mapvar/`), c'est la règle d'or du §7 (une seule piste à la fois sur une surface
   partagée). Les enchaîner après.
| **H — hygiène post-merge** *(ajoutée le 2026-08-01)* | audit ciblé sous skill `adversarial-audit` | `analysis/weaponv3`, `analysis/objectiveevents` et leurs appelants | après J5 ; **ne jamais supprimer une migration déjà appliquée en prod** |

**Condition d'entrée de la piste B** (découverte J3-2, à ne pas perdre) : `internal/himodule`
**n'a aucun test**, et le découpage de `loadHd1` fait en J3-2 a été vérifié **par lecture, pas
par exécution**. Le gate des artefacts ne le couvre pas : `replay-build` lit les fichiers de
structure FIGÉS, il ne repasse pas par `himodule`. Premier geste de B : **régénérer une
structure de carte et la comparer à la version figée** — c'est le test d'équivalence qui
manque, et B a besoin d'`himodule` de toute façon.

**Piste H, périmètre** : la lignée `weaponv3` (shadow **non promue**, piste documentée comme
MORTE) et `objectiveevents` restent invisibles à `unused` parce qu'un `cmd` les référence —
c'est le plus gros gisement de code mort restant. **Audit avant suppression**, jamais
l'inverse : vérifier chaque appelant, et laisser intactes les migrations
(`steps_shared_weapon_kills_v3.go` et consorts) — une migration appliquée en production ne se
supprime pas, son historique fait partie du schéma.

**Pourquoi B2 existe — constat vérifié le 2026-08-01.** L'extraction d'objectifs ne porte
**aucune forme** : `mapvar.Objective` = `Pos` + `Forward` + `TeamIndex` + labels. Une zone y
est un POINT. Dessiner un disque serait donc une invention, pas une lecture — et
l'observation utilisateur va plus loin : bases, extractions et collines ont des **angles
droits**, et la présence de `Up` ET `Forward` dans `mapvar.Object` (deux axes = base
d'orientation complète) indique des **boîtes ORIENTÉES**, qu'un disque ne peut pas
représenter même en ajustant un rayon. Trois sources candidates à trier, dans cet ordre :
(1) `forge_object_types.csv` porte DÉJÀ une emprise par type (`min/max_{x,y,z}`, `dx/dy/dz`,
colonne `geom`) sur 45 types — c'est une boîte, pas un rayon ; (2) une éventuelle **échelle
par objet** dans le `.mvar`, non décodée aujourd'hui (Forge permet de redimensionner) ;
(3) l'entité de zone à l'exécution (`ti=23`, **33/33 désérialiseurs lisibles depuis J0.4**)
si le statique ne suffit pas — cas des collines mobiles (KOTH). B2 traite dans la même passe
le `[!]` power-ups de J0.2 (`type_id → nom`) : même module, même parcours.

**Q2bis — LES CHAMPS DÉJÀ PARSÉS QUE PERSONNE N'A LUS** *(ajouté le 2026-08-01, question
utilisateur sur les râteliers d'armes, points d'apparition, grenades et power-ups qui
réapparaissent).* Constat vérifié dans `mapvar` : le `.mvar` est décodé par un lecteur
**structuré** (`cb2.go`, format Bond — « tout tag/type inconnu remonte une erreur, on ne saute
JAMAIS »), donc **tous les champs du record d'objet sont déjà parsés**. Mais `parseObject`
n'en EXTRAIT que six (2, 3, 4, 5, 7, 10) et `readGameplayBag` trois (1, 8, 9). **Tout le
reste est sous la main, non lu.** Or Forge expose par objet placé des réglages qui sont
exactement ce qu'on cherche : **délai de réapparition**, ordre d'apparition, présent-au-départ,
échelle, forme et taille de zone. Enumérer ces champs et regarder ce qu'ils contiennent est
une passe de quelques heures, pas un chantier de rétro-ingénierie — et elle sert Q2 (forme
des zones) et Q2bis (réapparition) **en une fois**.

**Ce que Q2bis débloquerait, et par quel chemin** : la présence d'une arme ou d'un power-up à
l'instant T se calcule alors comme *placement (B2 Q1) + événement de ramassage (piste A5) +
délai de réapparition (Q2bis)* — **sans jamais décoder l'entité vivante**, c'est-à-dire en
contournant la route qui a été RÉFUTÉE trois fois (position des `ti=42` / `ti=37` : 5
échantillons contre 1 006 sur un jeu de slots fantôme, signal sous le bruit).
**Nuance à ne pas se tromper** : la table des désérialiseurs de J0.4 **ne débloque PAS** cette
route — `ti=42` (21/21) et `ti=37` (31/31) étaient déjà complets. Le verrou n'a jamais été la
grammaire, c'est le décodage positionnel. Ne pas rouvrir cette piste en croyant que J0.4 l'a
changée.
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
4. Objectifs — **OUVRIR PAR LA SESSION DE VERDICT « recette mode → score »** (hypothèse
   utilisateur du 2026-08-01 : une indirection dirait, selon le mode, OÙ lire le score et
   QUELS événements comptent ; prompt remis par le superviseur ; **Fable 5, effort max** ;
   parallélisable en worktree dès maintenant — mesure pure, zéro code livré). Elle
   cartographie le système d'événements (dispatcher `FUN_140620564` codes 0x02-0x3c,
   footer type-3, flux de score TYPE_2) et rend son verdict AVANT le portage des
   désérialiseurs. L'étape 3 du plan objectifs (décompilation + portage, témoin
   `progress/required-progress`) suit le verdict — la cartographie sert aussi A5 (mêmes
   codes) et G3.
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

### Modèle et niveau d'effort par session  *(ajouté le 2026-08-01, demande utilisateur)*

La règle en une phrase : **si la session doit DÉCIDER (un verdict de mesure, de RE, une
conception), monter le MODÈLE ; si elle doit APPLIQUER sous un gate objectif, c'est le FILET
qui protège — inutile de surpayer le modèle.** L'effort suit la même pente : élevé pour
diagnostiquer et interpréter, moyen pour exécuter du spécifié, bas pour du scripté sans
ambiguïté. Jamais « bas » sur une session qui touche le décodeur.

| session | modèle | effort | pourquoi |
|---|---|---|---|
| Superviseur (fil unique) | Fable 5 (contexte 1M utile) | élevé | arbitrages, revues, vérification des gates — là où une erreur coûte le plus |
| J3-1 — lots A+B (mécanique) | Sonnet 5 *(ou Opus en mode rapide)* | moyen | mission fermée, gates objectifs (goldens J2) ; le « jugement » du lot A est déjà écrit dans le critère du plan |
| J3-2 — lot C (33 verdicts) + D + gardes | Opus 5 | élevé | brancher/supprimer/documenter des grammaires + refactors sous goldens |
| J4 — branchement killsource (3 sessions) | Opus 5 | élevé | code de prod sous invariants anti-ART, concurrence, migrations — le jalon le plus risqué |
| J5 — merge + backfill prod | Opus 5 | moyen | piloté par checklist (lots E/F) ; la sûreté vient de la checklist et du superviseur |
| J6-A — features décodeur (fil, variables, table capacités/i57, branchement ramassages) | Opus 5 | élevé | mesure + code, protocoles falsifiables déjà écrits |
| J6-A — sessions de VERDICT RE (relecture 6 bits, événement pickup 0x02-0x3c, désérialiseurs objectifs) | **Fable 5** | **max** | le profil exact où l'auto-illusion coûte des semaines (patrons d'erreur documentés) ; un verdict faux invalide tout l'aval |
| J6-B — port triangles (étapes 1-4) | Opus 5 | élevé | portage à gates numériques bit-exacts |
| J6-B — génération 14 cartes + retouches (étapes 5-6) | Sonnet 5 | moyen | exécution outillée, revue visuelle humaine derrière |
| J6-C — producteur sync objectifs | Opus 5 | élevé | écritures persist (chemin cadré §5.2-4) |
| J6-C — consommation front · J6-G citations | Sonnet 5 | moyen | patterns du dépôt, gates typecheck/vitest |
| J6-D — icônes | Sonnet 5 | moyen | exploration bon marché, échec sans dette, gate humain |
| J6-E — précision projectiles (timebox 2 sessions) | **Fable 5** | **max** | timebox court sur problème dur : le meilleur modèle rentabilise les 2 sessions |
| J6-F — design rejeu prod | superviseur (Fable 5) | élevé | document de conception, court |
| Vérifications scriptées, inventaires | Haiku 4.5 | bas | exécuter une liste de commandes, rien à décider |

Précisions d'usage :
- **Mode rapide (Opus)** : quand la latence domine (longues passes mécaniques, backfills) —
  même modèle, sortie plus rapide ; n'y mettre aucun verdict.
- **Contexte 1M** : pour le superviseur et les sessions de verdict RE qui tiennent les états
  de l'art en tête ; une mission fermée n'en a pas besoin.
- Chaque ordre de mission remis par le superviseur porte désormais une ligne « Modèle
  recommandé / effort » en tête.

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
- **[2026-07-31, nuit — superviseur] J2 CONFIRMÉ CLOS, arbitrages D1-D5 RENDUS.**
  Contre-vérifié : run 30666349775 au niveau job (identique au constat ci-dessus), fixtures
  sur disque (`inputs` 620 Ko gz, `minifilm` 700 Ko, golden 8 Ko), arbre propre. Arbitrages :
  **D1 + D2 → J3.4, AVANT le merge** — motif : J4 met le décodeur dans un collecteur en
  tâche de fond du process sync, une panique sur film malformé y coûterait le process entier
  (et ses writers DuckDB) ; sous les goldens J2, la garde est sûre. **D3 → J3.5** (tri
  total — sert les goldens, le diff et le cache prod futur). **D5 → J3.6**. **D4 → aucune
  action** (`plain` est le repli sobre documenté). J3 découpé en 2 sessions : **J3-1** =
  lots A + B + J3.1/J3.2/J3.6 ; **J3-2** = lots C + D + J3.3/J3.4/J3.5. Prochain : **J3-1**.
- **[2026-08-01 — exécuteur] J3-1 CLOS : lots A + B, J3.1, J3.2 ; J3.6 REFUSÉ. 70 → 43.**
  Mesure d'entrée refaite cache purgé (70, dont 33 `unused`) — conforme à J1.4, aucune dérive
  depuis J2. **Lot A** : 405 répertoires jetables supprimés (462 fichiers) + 3 entrées
  d'allowlist retirées → 62. **J3.2** : 8 PNG de grenades orphelins. **Lot B** : les 7 linters
  mécaniques à zéro, traités un par un, **gate joué 5 fois** (3 artefacts reconstruits, les 7
  grandeurs du §5 identiques à l'unité près à chaque passage, goldens verts, `go test ./...`
  vert) → 43. Commits `d53c133a9`, `0ca499eaf`, `1cb67b790`.
  **J3.6 refusé sur prémisse fausse** : `js-yaml` en `overrides` n'est pas une dépendance
  fantôme mais le correctif de **CVE-2026-53550** (commit `54a6eb3df`) — `@redocly/openapi-core`
  épingle la version vulnérable en exact, l'`overrides` est le seul levier. Le retirer
  réintroduirait la faille. **Décision superviseur attendue** (par défaut : ne rien faire).
  **Deux choses à savoir pour J3-2** : (a) le lot C compte **34** verdicts et non 33 — un
  `unused` était masqué par un `revive` à la même position, golangci n'en rend qu'un ;
  (b) J3-2 hérite des **3 `revive argument-limit`**, écartés de la passe mécanique (rentrer
  sous le plafond = regrouper des paramètres de décodage, ce n'est pas du nommage).
  **Piège de méthode consigné** : `replay-build` ne journalise PAS les états d'inventaire —
  sans un relevé sur l'artefact (`jq '.inventory|length'`), une des 7 grandeurs du §5 n'est
  pas vérifiée du tout. Prochain jalon : **J3-2**.
- **[2026-08-01 — superviseur] J3-1 VÉRIFIÉ ET CONFIRMÉ CLOS (70 → 43).** Run CI
  30671925050 contrôlé au niveau job : tous verts (Coverage + Baseline 20 min 56), seul Go
  Lint rouge — à 43, égal à la mesure locale. Arbre propre. **J3.6 tranché : refus de
  l'exécuteur VALIDÉ** — contre-vérifié sur pièces (l'override est le correctif
  CVE-2026-53550, commit `54a6eb3df`) ; statut `[~]`, condition de retrait notée. Reste
  pour J3-2 : 34 `unused` + 4 `staticcheck` + 3 `argument-limit` + 2 `gocyclo` = 43, plus
  J3.3/J3.4/J3.5. Garde-fou ajouté au verdict (b) du lot C : croiser avec la table des
  désérialiseurs J0.4 — un lecteur d'un archétype PLANIFIÉ (ti=11, 23, 40, 43) est un cas
  (c) daté, pas une suppression. Prochain : **J3-2** (Opus 5, effort élevé).
- **[2026-08-01 — superviseur, périmètre]** Hypothèse utilisateur ajoutée : une « RECETTE
  mode → score » existerait dans le film — une indirection qui dit, selon le mode, où se
  lit le score et quels événements comptent (captures/défenses de base, drapeau pris/
  déposé/capturé/rendu, frag du porteur près du dépôt — tous récompensés en score
  personnel, donc traqués). Les symptômes du plan objectifs (sur-comptage crâne ×20,
  libellés invalidés) sont compatibles avec une lecture SANS recette ; J0 a prouvé que le
  mode change la représentation (ti=11 : 162 en CTF, 0 en Strongholds). Session de
  verdict dédiée créée (Fable 5, effort max, parallélisable en worktree — mesure pure),
  en ouverture de la piste A-objectifs : cartographie du système d'événements + témoins
  Slayer (score = frags killsource), ancres terrain J0.6, personal_score à l'unité.
  Sous-produit consigné pour A5 (mêmes codes 0x02-0x3c). Étapes 1-2 du plan objectifs
  (côté app) : inchangées.
- **[2026-08-01 — superviseur, périmètre] Piste B2 créée : la FORME des zones.** Observation
  utilisateur : les zones (bases, extraction, KOTH) ne sont pas toujours circulaires — angles
  droits fréquents. Vérifié sur pièces, et le problème est plus large que « gérer les
  formes » : **l'extraction ne porte aucune forme du tout** (`mapvar.Objective` = un point +
  une orientation). Rien n'est donc à corriger dans un rendu — il n'y a rien à rendre.
  Trois sources candidates identifiées (emprise de `forge_object_types.csv` / échelle
  par objet dans le `.mvar` / entité `ti=23` à l'exécution), voir §J6-B2. Groupée avec le
  `[!]` power-ups de J0.2 : même module Forge, même passe. **Placement** : avant tout
  affichage d'objectif (plan objectifs étape 4.1, lot 7.3c) ; sans dépendance sur le
  verdict mode→score — les deux sessions sont parallélisables (surfaces disjointes :
  `mapvar`/`himodule` contre film/Ghidra), ou séquentielles sur le même worktree.
- **[2026-08-01 — J3-2, exécuteur] J3 CLOS : 43 → 0.** Lot C, les 34 `unused` statués un par
  un — et le verdict s'est décidé sur un INDICE, pas sur une intuition : le traverseur
  s'arrête au PREMIER composant présent non porté, donc un lecteur du composant i<sub>k</sub>
  ne change rien s'il existe un trou à un indice antérieur. Croisement fait sur pièces entre
  le registre ECS (118 archétypes, **1 067** composants — le compte de J0.4) et les 188 noms
  du `switch consumeByName`. **Zéro cas (a)** : aucun n'était un bug de dispatch. 32 retirés
  (17 supplantés par le port INLINE de `traverse.go` — souvent le port CORRIGÉ ; 11 dans un
  archétype ni décodé ni planifié, dont un doublon et un dont la prémisse est réfutée par la
  vérité EXE ; 4 primitives dont le fait survit au retrait), 2 gardés sous `//nolint` daté —
  ce sont exactement les deux qui visent un archétype PLANIFIÉ, objectifs `ti=11` et zones
  `ti=23`. Lot D : les 4 `staticcheck` LUS d'abord, aucun défaut de logique ; les 2 `gocyclo`
  découpés ; les 3 `argument-limit` regroupés en structure. J3.3 (ratchet knip en CI sans
  filtre de chemin — élargir le glob du hook n'aurait rien réglé), J3.4 (garde de longueur +
  `PeekBits` aligné + corpus de régression), J3.5 (artefact reproductible à l'octet).
  **Mesure de sortie : `0 issues.`** (cache purgé) — la cible F1 est atteinte ; `go test ./...`
  vert ; les 7 grandeurs des 3 films identiques à l'unité près à chaque gate.
  **J3.5 a démenti sa propre prémisse** : le défaut n'était pas qu'un tri instable, `grenades`
  changeait de VALEUR d'une exécution à l'autre. Une seule chaîne causale expliquait les deux.
  **Découvertes remontées** (détail au journal du plan de dette) : un garde de CI qui ne se
  déclenche JAMAIS (n°3, à traiter en priorité) ; deux archétypes qu'un seul câble porterait
  de 0/N à N/N (n°1, pour J6-A) ; plafonds knip périmés à 0/0/0 (n°4) ; `himodule` sans
  aucun test (n°5) ; la lignée `weaponv3` + `objectiveevents` morte mais invisible à `unused`
  parce qu'un `cmd` la référence (n°6).
- **[2026-08-01 — superviseur] J3 VÉRIFIÉ ET CONFIRMÉ CLOS — LE RATCHET LINT EST VERT.**
  Run CI `30693673708` contrôlé au niveau job à sa fin : **les 8 jobs verts, `Go Lint
  (golangci-lint)` COMPRIS** — première fois du chantier (il était rouge de bout en bout
  depuis la réconciliation). Arbre propre, 4 commits de code + 1 de documents.
  **Arbitrages rendus sur les découvertes** : (n°3, garde de CI mort) → **J4.0**, contre-
  vérifié d'abord — *l'invariant TIENT* (`feedback-drawer/queries.ts` n'importe pas `api` et
  pose `credentials: 'omit'`), donc trou latent et non fuite active, mais il partirait sur
  `main` en J5 ; correctif en trois gestes (chemin, garde auto-vérifiant, audit des autres
  gardes par grep). (n°4, plafonds knip) → **J5.1bis** et pas avant : un plafond à 0 pendant
  J4 bloquerait un push intermédiaire légitime. (n°5, `himodule` sans test) → **condition
  d'entrée de la piste B** : le gate des artefacts ne couvre PAS ce paquet (`replay-build`
  lit les structures figées), l'équivalence se prouvera en régénérant une structure et en la
  comparant à la figée. (n°6, `weaponv3`/`objectiveevents`) → **piste H**, post-merge, sous
  `adversarial-audit`, avec l'interdit explicite de toucher aux migrations appliquées.
  (n°1, `ti=19`/`ti=46`) → levier J6-A ; **les lecteurs supprimés sont récupérables au
  commit `a5ff6b5ed`** — le noter là où J6-A le lira.
  **Ce que J3 a appris** : le verdict du lot C ne s'est pas décidé au jugement mais sur un
  INDICE mesurable — le traverseur s'arrêtant au premier composant non porté, un lecteur
  placé après un trou ne change rien. Zéro cas (a) sur 34 : aucun n'était un bug de
  dispatch. Prochain jalon : **J4** (Opus 5, effort élevé, 3 sessions).
- **[2026-08-01 — J4 session 1/3, exécuteur] BRIQUES + PHASE 1 CLOSES, GATE 1 ATTEINT.**
  J4.0 (garde de CI mort) + les 2 migrations + les 2 persisters + le pont + le collecteur.
  Commits `e1b46d9d8`, `32589dad4`, `d06df0fe6`, `36fc76835`. Le collecteur EXISTE et est testé,
  mais **aucun appelant de production ne l'invoque** : son déclencheur est la sous-commande de
  backfill (session 2), conformément à l'ordre corrigé P1 → P3 → P2 → P4.
  **GATE 1** : film réel `9b191a7f` décodé → écrit → relu **par la vue `_latest`** (90 morts,
  2 passes) ; `-tags=integration -p 1` sync+persist vert (une exception : le flake de timing WAL
  connu, repassé 3× isolé) ; les garde-fous anti-ART verts **sans une seule entrée d'allowlist** ;
  `SUM(assist_extra_count)` interrogeable ; `go test ./...` vert.
  **Ce que la session a appris, et qui vaut plus que le code** : *un plan qui recopie l'état
  d'une autre branche recopie aussi ses énoncés périmés.* Quatre prémisses du plan de branchement
  et du guide ont été démenties **en les exécutant**, pas en les relisant — (1) le cache disque
  porterait « seulement les chunks de réplication » : **faux**, les 949 films utilisables portent
  aussi en-tête ET kill-feed, donc **le backfill de la session 2 est intégralement HORS LIGNE**
  (ni réseau, ni tokens, ni CDN — un risque de moins, celui de l'expiration serveur en cours de
  route) ; (2) `splitSQL` couperait sur un `;` en commentaire : il est conscient des `--` ;
  (3) le scan anti-ART ne verrait pas un DELETE nu : `TestNoRawDeleteOnAppendOnlyTables` le voit ;
  (4) « le collecteur va dans `internal/sync/` » : oui, mais pas à la racine — le ratchet
  `TestSyncRootPackageFrozen` (80 fichiers gelés) impose un sous-paquet, d'où
  `internal/sync/killcollector`. **C'est un ratchet qui l'a dit, pas une relecture.**
  Et la seule question que le guide laissait explicitement ouverte (« le persister accepte-t-il
  un tueur nommé sans XUID ? », cas `tueur-bot`) est **tranchée : oui**, avec son test.
  **Mesures pour dimensionner la session 2.** *Périmètre* : **1 926** matchs au registre,
  **1 343** porteurs de `killer_victim_pairs`, **949** films en cache — et **949/949
  correspondent à un match du registre, zéro orphelin**. Donc **394 porteurs de paires (29,3 %)
  n'auront JAMAIS de film** : l'estimation « au moins 28 % » du plan était juste. La table porte
  **250 139 lignes pour 133 886 clés distinctes — 46,5 % de doublons** (contre 46,8 % mesurés en
  juillet sur une table plus petite : le défaut ne se résorbe pas).
  *Coût* : de **0,20 s/chunk** (8 chunks, 1,6 s) à **16,6 s/chunk** (69 chunks, **19 min 05**),
  en passant par 0,44 s/chunk à 33 chunks (14,4 s) et 9,1 s/chunk à 63 chunks (**9 min 35**).
  **Passer de 63 à 69 chunks (+9,5 %) DOUBLE le temps.** Ce n'est pas une pente, c'est un mur —
  facteur **83** sur le coût par chunk. Corpus : 120 films ≤ 20 chunks, 554 de 21-30, 216 de
  31-40, 35 de 41-50, **24 au-delà de 50**. Backfill complet : ~2 h 30-3 h pour les 925 premiers,
  **4 à 8 h pour les 24 derniers**, à passer en dernier. L'anomalie est **mesurée, pas corrigée**
  (consigne du 2026-07-31 respectée) — mais elle a une conséquence appliquée : la limite de temps
  par match passe de 22 à **45 min**, parce qu'avec une telle pente une limite trop juste
  transformerait un film lent mais VALIDE en perte de donnée.
  **Découvertes reportées (non traitées)** : (a) deux gardes CI en `--if-present` — même motif
  que J4.0, latent (les deux scripts existent aujourd'hui) ; (b) `match_weapon_shots` n'a **aucun
  producteur** (la ventilation vient du scanner de fire-events, hors périmètre de la session) ;
  (c) le **second producteur** de `match_kill_events` (chemin `highlight_events`, crédit seul)
  n'existe pas — sans lui, la table couvrira 949 matchs sur 1 325 et le retrait de
  `killer_victim_pairs` reste impossible ; (d) `seed_demo.go` copie `killer_victim_pairs` : à la
  bascule, oublier d'y ajouter la remplaçante donnerait une démo aux duels vides, panne
  silencieuse visible seulement à l'écran.
- **[2026-08-01 — superviseur] J4 session 1/3 VÉRIFIÉE ET CONFIRMÉE CLOSE — CI TOUTE VERTE.**
  Run `30697874627` contrôlé au niveau job à sa fin : **les 8 jobs verts**, Go Lint compris
  (le ratchet reste à 0 après l'ajout des migrations, persisters, pont et collecteur), plus
  ADR 0021 Gate et Deploy Pre-Check. Arbre propre, commits `e1b46d9d8` → `deba72742`.
  **Arbitrages rendus pour la session 2** :
  1. **Le producteur de `match_weapon_shots` MONTE EN SESSION 2, avant le backfill.** Motif
     décisif : le backfill est la passe chère (2 h 30 à 7 h 30, plus les BTB à 19 min pièce) ;
     décoder 949 films pour les seules morts obligerait à TOUT re-décoder ensuite pour les
     tirs. **Tout ce qui lit un film doit être câblé avant que la passe ne tourne.**
  2. **Le second producteur de `match_kill_events` MONTE AUSSI EN SESSION 2 — et il ne coûte
     rien.** Vérifié sur pièces : `killer_victim_pairs` est produit depuis la table
     `highlight_events` (`events_completion_persister.go`), déjà EN BASE — pas depuis un
     film. Le second producteur est donc une transformation **SQL → SQL**, sans réseau ni
     décodage, qui couvre les **394 matchs (29,3 %) dont le film a expiré**. Sans lui, la vue
     de compatibilité de la session 3 perdrait 29,3 % des matchs : ce ne serait plus une
     déduplication mais une régression silencieuse.
     **Piège à traiter dans le même geste** : un match porteur des DEUX sources doublerait
     ses lignes — la règle de préséance (film > highlight, via `source_tag`) doit être écrite
     ET testée, sinon on réintroduit exactement le doublon qu'on est en train d'éliminer. Et
     les lignes issues de `highlight_events` portent `assist_known = FALSE` (« on ne sait
     pas »), jamais « pas d'assistant ».
  3. **Le mur de coût se PROFILE avant le backfill** (levée ciblée de l'interdit « ne pas
     toucher au décodeur ») : 0,20 s par chunk à 8 chunks contre 16,6 s à 69 — **facteur 83**,
     c'est une signature de bug (complexité superlinéaire), pas une propriété d'échelle.
     Timeboxé ; tout correctif reste sous les goldens J2 et l'exigence d'artefacts
     **bit-identiques**. Si la cause n'est pas claire dans le temps imparti : documenter et
     lancer le backfill tel quel.
  4. **Les deux gardes CI en `--if-present`** (même motif que J4.0, latents) → **J5.1bis**,
     avec les plafonds knip : c'est le lot « hygiène des gates ». **`seed_demo.go`** est déjà
     recensé comme site S1 du plan de branchement — à traiter à la bascule (session 3), pas
     avant.
  **Quatre prémisses du plan démenties en l'exécutant**, dont deux qui changent le risque :
  le cache porte en-tête + réplication + kill-feed pour les 949 films, donc **le backfill est
  intégralement hors ligne** (ni réseau, ni tokens, ni CDN — le risque d'expiration en cours
  de route disparaît) ; et le collecteur ne pouvait pas vivre à la racine d'`internal/sync`
  (`TestSyncRootPackageFrozen` gèle le paquet à 80 fichiers) — c'est un ratchet qui a dicté
  `internal/sync/killcollector`, pas une relecture d'architecture.
- **[2026-08-01 — J4 session 2/3, exécuteur] LES DEUX PRODUCTEURS, LE BACKFILL, ET LE MUR DE
  COÛT QUI ÉTAIT UN BUG.** Périmètre (A)→(E) traité ; **phase 3 CLOSE**, phase 2 intacte
  (session 3). Détail complet au journal du `PLAN_BRANCHEMENT_KILLSOURCE.md`.
  **(A) Le mur de coût est tombé, et il n'était pas où on le cherchait.** Un profil CPU a suffi :
  78 % du temps dans `filmdec.ReadBits`, appelé depuis UNE fonction —
  `consumeObjectMultiplayerProperties` sautait le corps d'un TLV **un octet à la fois**, plafonné
  à 1 048 576 itérations. Le corps n'est jamais interprété : `Skip(8N)` a le même état final.
  **1 145 s → 46,7 s** sur le plus gros film (×24,5), 331 s → 33,7 s, 575 s → 51 s ; le facteur 83
  sur le coût par chunk tombe à ~5. **Double gate tenu** : 3 artefacts de rejeu bit-identiques
  (revérifiés après TOUTES les modifications de la session) et JSON `killsource` bit-identique
  sur 8 films. *La leçon vaut plus que le gain* : la non-linéarité n'était pas dans le décodage
  mais dans le **chemin d'erreur** — c'est la fréquence des traversées ratées qui croît avec la
  taille du film, et chaque ratée coûtait un million de lectures. Chercher un algorithme
  quadratique dans le chemin nominal aurait été chercher au mauvais endroit.
  **(B) et (C) : les deux producteurs sont câblés AVANT le backfill**, comme l'arbitrage
  l'exigeait. Les tirs sortent de la même passe de décodage que les morts (un film décodé une
  fois, deux tables écrites) ; le producteur credit-seul est une transformation SQL → SQL qui
  emploie `analysis.ComputeKillerVictimPairs` — LA fonction qui produit `killer_victim_pairs`
  aujourd'hui. Préséance film > credit **testée dans les deux ordres d'arrivée**, trois états de
  l'assistant vérifiés colonne par colonne en base.
  **LE DÉFAUT QUI A COÛTÉ UNE PASSE — et c'est une remarque de l'utilisateur qui l'a ouvert.**
  `match_participants.gamertag` est VIDE en production (4 xuids nommés sur 16 996) ; la source
  canonique est la vue `v_gamertag_lookup` et sa cascade, où **`killer_victim_pairs` est la
  source VIVANTE** des noms d'adversaires depuis que `xuid_aliases` a cessé d'être alimenté en
  avril 2026. Et le film ne donne pas toujours un pseudo : sans gamertag il écrit
  `xuid:<décimal>`, qui EST l'identité. Résultat du défaut : **16 908 morts écrites, 10 avec un
  xuid de victime** — des lignes qu'aucun agrégat carrière ne peut joindre. Après correctif :
  **77 morts, 75 xuids victime, 77 xuids tueur** sur les mêmes films. *Une passe qui « réussit »
  sans erreur peut n'avoir rien produit d'utilisable : un backfill se contrôle sur ce qu'il rend
  JOIGNABLE, pas sur son taux d'échec.*
  ⚠ **Dépendance qui gouverne la phase 2** : supprimer `killer_victim_pairs` sans avoir d'abord
  rebranché `v_gamertag_lookup` sur `match_kill_events_latest` ferait retomber tous les
  adversaires sur « Joueur #### ».
  **Découverte grave à arbitrer, PRÉEXISTANTE** : `TestGoldenFilms` échoue déjà sur les 4 films
  de référence — prouvé sur l'arbre restauré, diffs identiques au bit près avec et sans le
  correctif (A). Les goldens datent du 31/07 17:49 ; **J3 (lots B/C/D) a modifié le décodage sans
  que personne le voie**, parce que ce test se skippe sans `KILLSOURCE_FIXTURES` et que le gate
  J3 portait sur les artefacts de REJEU. Les ancres Theater tiennent. *Un gate qui se skippe sur
  une variable d'environnement n'est pas un gate, c'est une option.*
  **GATE 3 ATTEINT.** Backfill complet : 949 films en **2 h 52 min 57 s** (74 569 morts, 0 erreur,
  0 abandon) puis 394 matchs credit-seul en 33 s (50 125 morts, **949 refusés par la préséance**).
  **Couverture 1 343 matchs — la cible exacte, et 100 % de ce qui est récupérable** : les 583
  restants portent `events_loaded = TRUE` sans événement (404 déjà constaté par le sync, dont 527
  de 2023). Contrôle : **250 139 → 124 694 lignes, facteur 2,006** ; sur le duel de contrôle joint
  par xuid, **101 frags affichés → 29** — mot pour mot la phrase qui motivait le chantier.
  `-tags=integration -p 1` vert, `go test ./...` vert, ratchet lint 0 (cache purgé), 3 artefacts
  bit-identiques.
  ⚠ **`assist_extra_count` a bougé pour la PREMIÈRE fois : 5 sur 74 569 morts.** Le déclencheur de
  migration de l'ADR 0026 a fait ce pour quoi il a été posé. **Arbitrage attendu** : table fille,
  ou plafond documenté comme perte connue de 0,007 %.
- **[2026-08-01 — superviseur, périmètre] Râteliers, points d'apparition, grenades et
  power-ups qui réapparaissent** (question utilisateur). Réparti sur trois pistes déjà
  existantes — **où** ils sont : acquis (`mapvar` rend position, orientation, équipe et
  libellés de chaque objet placé), il manque le NOM → **B2 Q1** ; **qui** ramasse quoi →
  **piste A5** ; **forme** des zones → **B2 Q2**. Manquait le **cycle de réapparition** :
  nouvelle question **B2 Q2bis**, fondée sur un constat vérifié dans le code — le `.mvar` est
  lu par un décodeur STRUCTURÉ qui n'ignore rien, donc tous les champs du record sont **déjà
  parsés**, mais `parseObject` n'en extrait que 6 et `readGameplayBag` que 3. Les réglages
  Forge par objet (délai de réapparition, ordre, présent-au-départ, échelle, forme/taille)
  sont donc probablement **sous la main, non lus**. Si c'est le cas, la présence d'une arme à
  l'instant T = placement + ramassage + délai, **sans décoder l'entité vivante** — la route
  réfutée trois fois est contournée, pas rouverte. Piège inscrit au plan : J0.4 **ne débloque
  pas** la position des `ti=42`/`ti=37` (déjà 21/21 et 31/31 — le verrou est positionnel, pas
  grammatical).
  **État B2** : Q2 (forme des zones) EN COURS dans un travail parallèle depuis ce jour ; Q1 et
  Q2bis NON couverts. Recommandation rendue : transmettre Q2bis au travail en cours (même
  passe, même fichier) et ne pas lancer Q1/Q2bis en parallèle (surface `mapvar/` partagée).
- **[2026-08-02 — superviseur] J4 session 2 + 2bis VÉRIFIÉES ET CONFIRMÉES CLOSES.**
  Run CI `30742303580` au niveau job : **8 jobs verts sur 9** — Go Lint (0 issue), Frontend,
  Build+Test ubuntu ET windows, Contract Test, OpenAPI Lint, Lease Enforcement, plus gitleaks
  et Deploy Pre-Check. **Un seul job rouge**, `Go Coverage + Baseline`, et **une seule cause** :
  `TestNoExpiredTODO` — le `TODO(expiry:2026-08-01)` de `season_pass_repo_tracks.go:297`
  (repli de rétro-compatibilité `track-def` → `bp-item-def`) est arrivé à échéance hier.
  **Dette datée arrivée à terme, sans rapport avec J4** ; le garde fait exactement son travail.
  **HYPOTHÈSE DU SUPERVISEUR RÉFUTÉE, et c'est la bonne nouvelle.** J'avais accusé le
  correctif de performance de J4-2 (« le plafond de 1 048 576 itérations disparaît, donc
  l'état final diffère »). **Faux, deux fois** : par la mesure — la divergence est déjà
  présente, signature identique au bit près, dès `96bc56175` (J2), donc avant J4 ; et par la
  lecture — le plafond est CONSERVÉ (`tlvBodyMaxBytes = 1<<20`, exactement le `i < 1<<20`
  d'origine) et `ReadBits` n'a d'autre effet que d'avancer `pos`, donc `Skip(8N)` atterrit au
  même endroit. J3 est exonéré par la même mesure.
  **LA CAUSE est `47c9e72ac`** — le commit de RÉCONCILIATION des deux `filmdec`, dont le
  message annonçait « killsource (golden) vert » alors que le test avait **skippé** faute de
  `KILLSOURCE_FIXTURES`. Faux vert de bonne foi, et il a tenu deux jours.
  Sous-cause prouvée : le `br.ReadBits(2)` ajouté dans `consumeAbsoluteWithGate` ; la
  calibration l'absorbe en rétrécissant l'axe et `3*axisW + indexW` perd **exactement 2** sur
  les quatre films (43→41, 53→51, 50→48, 52→50).
  **L'ORACLE NE DÉPARTAGE PAS, et on sait de combien** : 371 lignes publiées sur 371
  **identiques** (victime, tueur, tag, étiquette, catégorie), 380/380 morts de l'API et 30/30
  ancres Theater inchangées des deux côtés ; `cumul.golden` INCHANGÉ. Ne bougent que la
  calibration, les compteurs par voie, et le champ `voie` de **4 lignes sur 371**.
  **Conséquences statuées** : goldens re-congelés avec justification chiffrée ; **backfill NON
  rejoué** (124 694 lignes) — réserve écrite : l'équivalence est mesurée sur les 4 films de
  référence, pas sur les 1 343 matchs.
  **Défaut structurel corrigé** : `TestGoldenMiniBobine` tourne **sans fixture ni variable
  d'environnement**, donc en CI — vérifié positivement sur le runner (run puis pass). C'est la
  première fois du chantier que la sortie du décodeur killsource est gardée par un test que
  personne ne peut skipper par inadvertance. **Une prémisse de MON prompt est tombée en le
  construisant** : le patron de mini-bobine de J2 (paquets choisis, concaténés hors
  continuité) décode **zéro mort** ici — le décodeur construit son monde par accumulation
  depuis l'en-tête. Il fallait un **PRÉFIXE contigu**. À reprendre pour toute future bobine.
  **Deux limites déclarées, acceptées telles quelles** : (1) une sous-cause B subsiste dans
  les 892 insertions de `47c9e72ac`, non isolée, **effet borné et mesuré** (la provenance de
  4 lignes sur 371) → consignée pour la piste A, pas bloquante ; (2) la mini-bobine **ne
  verrouille pas le balayage de calibration** (profil plat sur un préfixe) — seul
  `TestGoldenFilms`, sur films entiers, le fait, et il reste optionnel.
  **RÈGLE PERMANENTE POSÉE** : toute session qui touche au décodeur (`filmdec`, `killsource`)
  DOIT jouer `TestGoldenFilms` en local avec `KILLSOURCE_FIXTURES=<chemin ABSOLU>` — les
  tests Go s'exécutent depuis le répertoire du paquet, un chemin relatif fait skipper le test
  en silence (erreur commise par le superviseur lui-même le 2026-08-01). La CI ne peut pas
  couvrir ce cas : les films pèsent 107 Mo.
- **[2026-08-02 — exécuteur] J4 session 3 : la bascule est PRÉPARÉE, pas faite — et c'est une
  mesure qui l'a décidé.** Item zéro clos (le `TODO(expiry)` échu de `season_pass_repo_tracks.go`
  retiré sur mesure : **zéro** item subsiste en `track-def`, vérifié sur DEUX bases — la base de
  dev dont l'`asset_index` remonte au 2026-04-21, donc antérieure à la bascule de kind, et la
  graine de production `metadata-prebuilt.zip`. 30 lignes de part et d'autre, toutes des chemins
  `RewardTracks/`, aucun chemin d'item : le repli ne pouvait plus apparier quoi que ce soit).
  **LE RÉSULTAT QUI COMPTE** : la phase 2 telle qu'écrite — « `killer_victim_pairs` devient une
  vue sur `match_kill_events_latest` » — a été implémentée, appliquée aux bases réelles, MESURÉE,
  puis retirée. Oracle (les événements `death` de `highlight_events`, 949 matchs à film,
  Halo Infinite) : **API 100 266 · ancienne table 98 662 (98,4 %) · passe de film 74 569
  (74,4 %)**. La vue `_latest` ne retenant qu'UNE passe par match, la passe de film — plus riche
  par ligne, plus pauvre en couverture — serait devenue la génération servie et aurait effacé
  **25 697 morts**, sans erreur, sans compteur, sans qu'un nom ou un instant ne change. Le
  corollaire figurait dans le DDL depuis la session 1 ; **ce qui manquait n'était pas
  l'avertissement, c'était sa magnitude** — un risque documenté sans chiffre se lit comme un
  risque théorique.
  **Livré et vérifié** : (1) le **producteur LIVE** des kill-events pour les DEUX titres — les
  deux producteurs existants étaient hors ligne (grep : zéro appel depuis `internal/api/`,
  `internal/sync/v2/`, `cmd/server/`, `internal/service/`), donc tout match synchronisé après le
  dernier backfill n'avait AUCUNE ligne ; (2) title-agnostic par construction, et ce n'est pas
  cosmétique — `highlight_events` de Halo 5 ne porte aucun événement kill/death, donc le
  producteur crédit-seul de `killcollector` n'aurait jamais couvert ce titre, qui pèse **268 337
  couples sans un seul doublon** ; (3) la **reprise dédupliquée** (migration
  `shared_kill_events_from_pairs_v1`) : Infinite 0 ligne (déjà couvert), Halo 5 **268 337** —
  le titre passe de « la table n'existe pas » à couverture complète ; (4) la préséance testée EN
  POSITIF sur les voies de film, l'ancienne forme (`read_path <> 'highlight-events'`) faisant
  passer toute voie autre que la sienne pour un film ; (5) **`v_killer_victim_full` supprimée**
  (Q20 lit la table, mêmes six colonnes) et **`feature-flags.ts`** avec le plafond knip abaissé
  **de 29/90/86 à 0/0/0** ; (6) `v_gamertag_lookup` rebranché sur la table canonique **en gardant
  la jambe historique en UNION** — la canonique seule coûtait 4 gamertags sur 18 219, mesuré.
  **GATE 2 au sens strict** : 11 mesures avant/après sur les deux titres, **toutes identiques**.
  **Découvertes reportées, non traitées** : (a) le découpeur SQL de `internal/sync/schema.go`
  n'est PAS conscient des commentaires — il coupe sur chaque `;`, y compris dans un `--` ; un
  point-virgule ajouté dans un commentaire a fait échouer 13 tests avec « empty query », loin de
  la cause. ⚠ **Deux découpeurs différents dans le dépôt, un seul est sûr** (celui de
  `migration`, constat de la session 1) ; (b) `match_kill_events` n'entre PAS dans le contrat de
  snapshot : toute table de cette liste est REQUISE à la lecture, l'y ajouter rendrait illisible
  tout snapshot antérieur (le lecteur ne dégrade pas, il refuse le snapshot entier).
  **NON FAIT, et à re-planifier** : lot 1.2/1.3 (lien vers le rejeu), lots 3.1/3.2, phase 4.
  **Donc aucun gate visuel à demander pour l'instant** — le lien n'existe pas, et la revue du
  correctif `w` reste due avec lui.
  **CE QUE LE RETRAIT DE `killer_victim_pairs` ATTEND** — un critère, pas une date : que le
  collecteur parte de la liste officielle des morts et l'ENRICHISSE, au lieu de publier la seule
  liste qu'il sait décoder. Mesurable : `lignes_passe_film / morts_api` ≥ 98,4 % sur le même
  périmètre. *Le doublon se voit et se corrige ; la mort manquante, non.*
- **[2026-08-02] CONCEPTION DE L'INVERSION — `.ai/CONCEPTION_INVERSION_PRESEANCE.md`.** Le critère
  ci-dessus est ATTEIGNABLE sans re-décoder un seul film, et c'est le résultat qui compte : les
  deux sources ne sont pas deux mesures indépendantes mais **un seul flux lu par deux canaux** —
  le kill-feed du film passe par `analysis.ParseHighlightEvents`, le MÊME parseur que
  `highlight_events`. D'où un appariement `(match_id, time_ms)` à **tolérance ZÉRO**, seule valeur
  où il est une bijection stricte (73 589 = 73 589 ; dès 50 ms les deux côtés divergent), et une
  reprise **SQL→SQL** : l'enrichissement des 949 matchs est déjà en base. Table cible **134 866**
  contre 124 694 aujourd'hui (**+10 172 morts**, dont 73 589 enrichies). **Un chiffre du chantier
  est corrigé** : l'oracle brut sur-comptait (15 120 groupes en double exact dans
  `highlight_events` sur les 394 matchs sans film) — sur l'oracle dédupliqué le crédit tient
  **98,5 % PARTOUT**, pas seulement sur le périmètre film. Les 980 orphelins de film sont
  CONSERVÉS : aucun ne tombe sur un instant sans événement API, et 968 sont des morts de bot que
  le kill-feed humain-seul de l'API ne peut structurellement pas porter. **Rien n'est
  implémenté** ; 4 arbitrages restent ouverts (§8 du document).

---

## PISTE F — LE REJEU EN PRODUCTION : la conception  *(écrite le 2026-08-02, questions utilisateur)*

> Ce n'est PAS un bloquant du merge (§J5 le vérifie), mais c'est un préalable à l'ouverture
> de la feature en production. Les décisions ci-dessous sont des recommandations du
> superviseur, à valider avant de coder.

### Le fait qui commande toute l'architecture

**L'artefact est REPRODUCTIBLE À L'OCTET depuis J3.5.** Un artefact construit sur le poste de
développement est donc **identique** à celui qu'un VPS produirait. *Où* on le construit
devient une pure question de coût, jamais de justesse. C'est le levier qui rend la suite
simple.

### Les grandeurs mesurées

| grandeur | valeur |
|---|---|
| artefact de rejeu | **~2 Mo** par match (2,19 · 1,64 · 2,62 Mo mesurés) |
| film source | ~24 Mo par match, **23 Go** pour 949 films |
| coût de décodage (poste de dev, APRÈS le correctif de J4-2) | 33 morceaux → 7,3 s · 40 → 33,7 s · 64 → 51 s · 69 → 46,7 s |
| matchs sans film, définitivement | **29,3 %** (les films Theater expirent côté serveur) |
| contrainte VPS de prod | petit CPU, **disque sous tension** (plafond de cache 5 Go, zéro swap — incidents de gel documentés) |

### 1. OÙ GÉNÉRER — un OUVRIER EXTERNE, et c'est un RÔLE, pas une machine

> **Recommandation corrigée le 2026-08-02 sur objection de l'utilisateur, et il a raison.**
> Ma première version faisait produire les artefacts sur son poste, puis les poussait. Deux
> défauts rédhibitoires : (1) **le dépôt est PUBLIC** — une conception qui dépend d'une
> machine nommée n'est ni déployable par un tiers ni reproductible ; (2) le poste **n'est pas
> toujours allumé**. Ce n'était pas un inconfort, c'était un défaut de conception.

**Le principe** : l'ouvrier est un **rôle**, tenu par n'importe quelle machine — le second VPS
(choix de l'utilisateur : plus de CPU et de RAM), un poste de développement, ou rien du tout.
**Le VPS web ne décode JAMAIS.** Il met en file et il sert.

```
VPS web (petit)                          OUVRIER (second VPS, ou poste, ou rien)
  met en file un job  ──── HTTP ────►      demande le prochain job
  résout le MANIFESTE (il a les tokens)    télécharge les morceaux (CDN pré-signé)
  sert l'artefact quand il arrive          décode, construit l'artefact (~2 Mo)
  ◄──────────── HTTP ────────────────      renvoie l'artefact, puis SUPPRIME les morceaux
```

**Trois propriétés qui découlent de ce découpage, et qui valent d'être vues :**

1. **L'ouvrier n'a AUCUN secret Halo.** Le manifeste exige les tokens ; les morceaux, eux,
   viennent d'un **CDN Azure pré-signé, sans authentification**. Donc le VPS web résout le
   manifeste et met **les URL pré-signées dans le job** — l'ouvrier devient un simple nœud de
   calcul, sans identifiants, sans accès à la base. C'est ce qui rend sûr de le faire tourner
   n'importe où.
2. **Aucune infrastructure nouvelle.** L'ouvrier *tire* le travail en HTTPS (deux routes
   internes : prendre un job, rendre un résultat) — pas de port entrant, pas de système de
   fichiers partagé, pas de Redis. C'est le patron de csstat, porté sur la pile Go existante.
3. **La dégradation sans ouvrier est déjà écrite.** Aucun ouvrier déployé → aucun artefact →
   aucun lien (`os.Stat`, lot 1.2). Indispensable pour un dépôt public : la feature s'installe
   sans que personne ait à monter un ouvrier.

**Le rattrapage reste à part** : 951 matchs ≈ 8 h une seule fois, et les films sont déjà en
cache sur le poste — cela se fait par le CLI (`levelup backfill-*`), sans passer par la file.
Un ouvrier neuf ne doit pas retélécharger 23 Go pour rattraper l'historique.

### 1bis. RÉTENTION GLISSANTE + PURGE TRIMESTRIELLE — décision utilisateur 2026-08-02

**On ne génère que les artefacts des matchs des N derniers mois** (défaut proposé : 3), et une
purge trimestrielle retire les artefacts plus vieux. Deux conséquences de conception :

- **La fenêtre est un RÉGLAGE ADMIN**, pas une constante. Elle vit dans `app_settings` (le
  store de configuration existant), s'édite depuis la page admin, et **commande deux choses à
  la fois** : ce qu'on accepte de mettre en file (un match hors fenêtre n'est pas construit) et
  ce que la purge supprime (un artefact hors fenêtre est effacé). Une seule valeur, deux
  usages — sinon ils divergent.
- **La purge est elle-même un JOB** (récurrent, cf. le scheduler existant ADR 0027), donc
  visible dans le même monitoring que les constructions. Elle supprime des FICHIERS (les
  artefacts, `cache` jetable) — **jamais les films** (archive perpétuelle) ni les lignes de
  base (`match_kill_events`). Le film reste : si un match ressort de la fenêtre puis y rentre
  (fenêtre élargie par l'admin), l'artefact se reconstruit.
- **Idempotence** : reconstruire un artefact déjà présent et à jour (`decoder_rev` inchangé)
  est un no-op — la purge et la reconstruction ne se courent pas après.

### 2. QUEL SYSTÈME DE JOBS — L'OSSATURE EXISTE DÉJÀ, on l'ÉTEND

**Ne rien réinventer. Vérifié le 2026-08-02, le dépôt a déjà tout le squelette :**

| brique existante | où | ce qu'elle fait |
|---|---|---|
| `domain.AsyncJobStatus` + `JobStatus` | `internal/domain/job.go` | états `queued/running/succeeded/failed/cancelled/interrupted`, `progress_pct`, `current_step`, `started_at`, `finished_at`, `error` — **le contrat de job est déjà écrit** |
| `JobStore` (ring mémoire) | `handlers` (`h.jobs.SetStatus/Update/Get`) | suit les jobs asynchrones, déjà consommé par l'auto-sync |
| `GET /admin/monitoring/jobs` | `admin_monitoring.go` | **la route de monitoring des jobs existe** |
| page admin | `apps/web/src/routes/admin/{sync,logs,system}.tsx` | le dashboard admin est là, avec sync et logs |
| scheduler récurrent | ADR 0027 (cycle orchestrator) | fait déjà tourner du travail de fond périodique |

Ce que la piste F **ajoute**, et rien de plus : un **type de job** « construire un rejeu » (le
`JobStore` est agnostique du genre), une **file persistante** (le ring mémoire actuel perd les
jobs au redémarrage — acceptable pour l'auto-sync qui se relance, pas pour une file de
construction : une petite table d'état, écrite par le chemin d'écriture normal), et le
**protocole ouvrier** ci-dessous.

**Le patron reste celui de csstat** (`POST …/parse` → `202 {jobId}` → ouvrier → maj + invalide
le cache), **porté sur ces briques Go** — pas BullMQ, pas Redis, pas d'ouvrier Node.

csstat fait exactement le bon geste (`POST /matches/:id/parse` → `parseQueue.add()` → `202
{jobId}` → l'ouvrier construit, met à jour, invalide le cache). **Le patron se reprend tel
quel.** Son implémentation, non : c'est **BullMQ (Node + Redis)**, or LevelUp est **Go +
DuckDB sans Redis**. Ajouter Redis et un ouvrier Node à une pile Go pour une feature de
consultation serait un mauvais échange.

Version Go, à volume faible et **un seul ouvrier** : une table d'état de job + une goroutine
d'ouvrier, ou le cycle d'orchestration qui existe déjà (ADR 0027) — il fait déjà du travail de
fond. Sérialisation obligatoire de toute façon : `filmdec` porte des globaux de paquet, un
seul décodage à la fois par process (§J4).

### 2bis. LES JOBS ET LE MONITORING — de bout en bout, DANS l'app  *(décision utilisateur du 2026-08-02)*

**La demande**

**Un artefact est un FICHIER, pas une ligne.** 2 Mo de JSON immuable, servi tel quel : c'est
du stockage de fichiers, et c'est déjà ce que fait `data/cache/replays/{titre}/{matchId}.json`.
L'intuition sur DuckDB est juste, pour deux raisons cumulées : le modèle **mono-process
writer** (ADR 0013/0016) et le fait qu'un blob de 2 Mo dans une colonne est un anti-patron.
Ce qui mérite éventuellement une ligne, c'est **l'état d'un job** (en file / en cours / fait /
échoué) et l'index (`decoder_rev`, `built_at`) — petit, écrit par le chemin d'écriture normal.
Et à l'étage 0, même ça est inutile : `os.Stat` suffit.

### 4. RÉTENTION — la règle est INVERSE de l'intuition

| donnée | politique | pourquoi |
|---|---|---|
| **films** (morceaux) | **archive PERPÉTUELLE, en local** | **irremplaçables** : ils expirent côté serveur, 29,3 % sont déjà perdus. Jamais sur la prod |
| **artefacts** | **cache, avec expiration** (TTL ou LRU) | **reproductibles à l'octet** depuis le film — les jeter ne perd rien |
| manifests de film | conserver (119 Mo pour 950) | permettent de re-télécharger depuis le CDN **sans authentification** |

Autrement dit : **on garde le brut pour toujours, on jette le dérivé sans état d'âme.** Un
artefact expiré se reconstruit ; un film expiré est perdu à jamais.

### 3bis. UN SEUL DÉCODEUR, DEUX EXTRACTEURS, ET OÙ CHACUN TOURNE — tranché 2026-08-02

**La prémisse « le killfeed est moins gourmand » est vraie pour UN des deux killfeeds, fausse
pour l'autre. Il faut les distinguer** (vérifié sur le schéma `match_kill_events`, qui sépare
explicitement « Vérité 1 : le crédit » et « Vérité 2 : la source du dégât ») :

| killfeed | ce qu'il porte | source | coût | où il tourne |
|---|---|---|---|---|
| **basique** | tueur / victime / instant | `highlight_events` en base → SQL | négligeable | **VPS web — inchangé, il y est déjà** |
| **enrichi** | + **arme du kill** (`source_tag`), + **assistant nommé**, + parts de dégâts, + morts de bots, + catégorie | **décodage du film ENTIER** (`killsource` prend `MemoryChunks`, tous les chunks) | ~50 s/gros film — **même ordre que le rejeu** | **OUVRIER distant** |

À l'écran, l'enrichi est la différence entre « A a éliminé B » (croix générique) et « A a
éliminé B au BR75, assisté par C ». C'est ce que le fil des éliminations du rejeu (lot 5) et
les agrégats carrière dégonflés consomment.

**« Deux décodeurs » — NON, et la séparation est une RÈGLE, pas un accident.** Il y a **un**
décodeur bas niveau (`filmdec`) et **deux extracteurs** au-dessus : `killsource` (lit le
dead-state → morts, arme, assistant) et `replay` (lit les positions → trajectoires, tirs).
Leur séparation est **doctrinale** : le rejeu ne re-décode PAS les morts, il les CONSOMME de
`killsource` via `Options.Kills` — parce que « deux décodeurs du même fait divergeraient »
(règle écrite du chantier). Les fusionner serait une régression.

**Le vrai coût à surveiller n'est pas « deux décodeurs » mais DEUX TRAVERSÉES du même film** :
construire un rejeu complet (avec fil des éliminations) fait tourner `killsource` ET
`BuildFromFilm`, soit deux parcours. Trois conséquences arrêtées :

1. **Tout décodage de film → l'ouvrier.** Le VPS web ne décode jamais un film ; il garde le
   killfeed basique SQL (léger, permanent) qu'il a déjà. La règle « le léger reste sur le
   web » est donc tenue — le léger, c'est le SQL.
2. **Un passage, deux sorties, la LOURDE conditionnelle.** L'ouvrier décode un film et rend :
   les lignes killfeed enrichi → **toujours** écrites en base (permanent, léger à stocker) ;
   l'artefact rejeu → **seulement si le match est dans la fenêtre** de rétention. Un match
   récent (donc dans la fenêtre) produit les deux d'un coup ; un vieux match consulté à la
   demande re-décode pour l'artefact, le killfeed étant déjà en base (idempotent, skip).
3. **La FUSION des deux traversées en une seule** (une passe `filmdec` alimentant les deux
   extracteurs, gain ~2× sur le décodage) est une **optimisation à trancher par une mesure**,
   PAS un prérequis. Elle couple les deux extracteurs sur un état `filmdec` partagé — coût de
   conception réel. À mesurer avant de s'y engager ; ne pas la faire d'office.

**Dépendance déclarée** : déléguer le killfeed enrichi à l'ouvrier le couple à la disponibilité
de celui-ci. Acceptable, parce que (a) l'enrichi n'est de toute façon PAS produit en prod
aujourd'hui (CLI/backfill seulement, mesuré 2026-08-02), donc on ne perd rien ; (b) la
dégradation est propre — sans ouvrier, le killfeed basique reste, et l'écran montre la croix
générique au lieu de l'arme. Jamais un blanc, jamais un faux.

### 4bis. LE MONITORING DE BOUT EN BOUT — l'app voit tout, même le travail distant

**Exigence utilisateur : le monitoring dans l'app, bout en bout, même quand l'ouvrier tourne
sur l'autre VPS.** C'est faisable ET c'est déjà le bon modèle, parce que **l'ÉTAT vit côté web,
pas côté ouvrier.** L'ouvrier ne fait que calculer ; il ne détient rien. La file, les statuts,
l'historique sont dans la base du VPS web — donc le dashboard admin les voit sans jamais
interroger l'ouvrier.

Ce que l'admin doit voir (une page `admin/replays`, sœur de `admin/sync`) :

| vue | source | déjà là ? |
|---|---|---|
| file : en attente / en cours / faits / échoués | table d'état des jobs | contrat `AsyncJobStatus` oui, table à créer |
| par job : match, étape, %, durée, erreur | `AsyncJobStatus` (champs existants) | **oui** |
| les OUVRIERS : dernier battement, job courant, débit | table d'ouvriers (heartbeat) | à créer |
| couverture : matchs dans la fenêtre / construits / manquants | requête sur les artefacts + la fenêtre | à créer |
| actions admin : (re)mettre en file un match, forcer une purge, changer la fenêtre | routes admin (patron `admin_actions.go`) | patron oui |

**Le heartbeat est ce qui rend le distant OBSERVABLE** : l'ouvrier, à chaque prise de job et
périodiquement, POST un battement (id d'ouvrier, job courant, horodatage). Un ouvrier muet
depuis N minutes s'affiche « hors ligne » — et comme l'ouvrier ne détient aucun état, sa
disparition ne perd aucun job : le job repris s'affiche `interrupted` puis `queued` (l'état
`interrupted` EXISTE DÉJÀ dans `job.go`, prévu pour « running → interrupted au redémarrage »).

**Santé du décodage** : `killhealth.go`/expvar (ADR 0009) existe déjà côté décodeur — l'ouvrier
remonte ces compteurs dans le résultat du job, et le monitoring les agrège. Rien à inventer.

### 5. CE QUE LE MERGE DOIT VÉRIFIER — et c'est tout

Le merge est sûr **à condition que la prod ne génère rien**. Trois vérifications, à faire en
J5 (§J5.5) :
1. aucun chemin de code de prod ne télécharge ni ne décode un film — le producteur live est le
   chemin **crédit** (SQL, vérifié le 2026-08-02), le décodeur de film est **CLI seulement** ;
2. le garde local (`replay_local_gate.go`) est **toujours en place** — la prod ne sert donc
   même pas la route ;
3. aucun artefact n'est déployé par mégarde (ils ne sont pas versionnés).

**Conception de la piste F à écrire AVANT d'ouvrir la feature en prod, pas avant le merge.**

---

## PISTE C-bis — INTÉGRATION DE `feat/re-mode-score` (objectifs mode/score/zones)

> Écrit le 2026-08-02 après revue adversariale à 3 relecteurs Fable (contexte frais, aveugles).
> Branche `feat/re-mode-score` (worktree `.claude/worktrees/re-mode-score`), base `b9f163d80`
> (J3-1) — elle n'a PAS les commits J4. Surface de code disjointe de J4 (elle touche
> `objectiveevents/`, `objectivescore/`, `replay/objectives`, `mapvar/`, `cmd/mapobj-build`).

### Verdict de la revue

**Le CODE est solide** : 0 P0, 1 P1, quelques P2 ; méthode de recherche jugée « remarquable »
(contrôles négatifs réels, faux positifs publiés, relevé pré-enregistré, goldens intacts,
multi-titre propre, forme inconnue → nil jamais de rayon inventé). **Les DOCUMENTS retardent
sur les mesures** — c'est là que sont les P0.

### Constats à traiter, par gravité

| # | gravité | où | action | quand |
|---|---|---|---|---|
| D-P0a | P0 **doc** | `HANDOFF_EVENEMENTS_NOMMES §2` porte 2 noms DÉMONTRÉS FAUX (`flag_taken`, `runner_stopped`) alors que `.ai/refs/TABLE_STATS_STATBORG.tsv` est corrigée | aligner le handoff sur la TSV/le code | **session restitution** |
| D-P0b | P0 **doc** | `ETAT_DE_L_ART_FORGE_PALETTE_ZONES` résumé de tête conserve 2 verdicts que le corps réfute (« power-ups clos par la négative » sur prémisse `power-up=eqip` fausse ; « nommage indécidable » alors que résolu) | réécrire le résumé à la date du dernier commit | **session restitution** |
| D-P1a | P1 code | `cmd/mapobj-build/refresh.go:62-114` : refresh peut écrire un catalogue `schema_version=2` avec zones sans `shape` (rendues points), rien ne distingue « non migré » de « ponctuel » ; `refreshOffline` sans test. **N'affecte pas l'artefact actuel** (242/242 ont shape) | marqueur distinct + test, OU consigner avec condition « avant prochaine migration v2→v3 » | **à l'intégration** |
| D-P1b | P1 doc | résumés/tableaux périmés dans les 3 docs (mode-score §1 CTF, §8 KOTH vainqueur inversé, §3bis « oracle secondaire » qui inverse la leçon, handoffs « rien implémenté » faux) | mise à jour | **session restitution** |
| D-P2 | P2 code | `awards.go:130-137` no-op + commentaire inversé · `slotidentity.go:94-107` 2e passe non testée · `named_test.go`/`slotidentity_test.go` `continue` au lieu de `t.Skip` (faux vert — **5e occurrence du motif** dans le chantier) | corriger | **à l'intégration** |

### STATUTS À LA CLÔTURE DE LA PISTE C-bis — 2026-08-05

| # | statut | ce qui a été livré | ce qui le garde |
|---|---|---|---|
| D-P0a | `[~]` | couvert par la session de restitution du 2026-08-02 (`65f27a66b`) | — |
| D-P0b | `[~]` | idem | — |
| D-P1a | `[x]` | `carried_from_schema` marque toute carte reportée d'un schéma antérieur ; note du catalogue + log `cartes_non_migrees` séparé du compte `sans_forme` | `cmd/mapobj-build/refresh_test.go` — 5 tests (le fichier n'en avait AUCUN), **2 mutations vues rouges** : marquage retiré, marquage inconditionnel |
| D-P1b | `[~]` | couvert par la session de restitution du 2026-08-02 | — |
| D-P2 | `[x]` | 3 volets : garde `if first` no-op retiré d'`awards.go` (commentaire inversé remplacé par le comportement réel) · 2e passe de `slotidentity.go` enfin exercée · `continue` → comptage + `t.Skip` dans `named_test.go` et `slotidentity_test.go` | 2 tests sur la première lecture · `TestSlotIdentityRefuseUnXuidRevendiqueParDeuxSlots` · **mutations vues rouges** : sauter la 1re lecture fait tomber 3 tests dont la réconciliation JGtm ; renvoyer `claim` fait tomber le test de 2e passe |
| dette TSV `zone 2 B = deaths` | `[x]` | **tranchée : LÉGITIME.** La ligne a un lecteur — le pont d'identité (`slotidentity.go` lit le triplet) ; les morts ne sont pas des événements d'objectif et le kill-feed les porte déjà avec l'arme et l'assistant. Le constat R1 était faux dans sa formulation : la TSV recense TOUS les emplacements décodés, `namedStatSlots` n'en est qu'un consommateur. Colonne `lecteur` ajoutée, en-tête de `named.go` corrigé | `TestTableStatborgConcordeAvecNamedStatSlots` vérifie la concordance **dans les deux sens** sur les seules lignes de lecteur `named.go` — **3 mutations vues rouges**, dont celle qui reproduit la question posée |

**Ce que l'intégration a découvert et traité en plus** : le calque d'objectifs ajoutait 2 champs
publiés (`objectives` sur le document et sur `Coverage`) **sans que le contrat les décrive**.
`contracttest/replay_contract_test.go` l'a attrapé — c'est exactement le défaut qu'il existe pour
empêcher. Contrat régénéré (`make openapi-gen`, jamais à la main), `generated.ts` régénéré,
frontière de nullabilité web comblée (`replayNormalize.ts`), le chiffre du chantier passe de 22 à
23 champs. Le rendu, lui, n'est PAS branché — décision #5 tient.

**Écart de méthode assumé** : intégré par **merge**, pas par rebase. 26 des 42 commits touchent
`.ai/thought_log.md`, qui conflite en tête à chaque fois ; le recouvrement réel est de 4 fichiers.
Le merge donne le même arbre pour une seule résolution par fichier.

**Conflits sémantiques que git n'a pas signalés** (les deux corrigés, aucun autre trouvé) :
`FilmshellWeaponKeysByFamily` déplacée par `main` de `games/halo_infinite/migrations` vers
`games/weapons` ; `readBitsBE` supprimée comme code mort par J3-2 alors que `statborg.go`,
écrit sur une base antérieure, lui donne un appelant (rétablie, en-tête de `film.go` corrigé).

### GATE DES ARTEFACTS DE REJEU — `[!]` NON PASSÉ, ARBITRAGE SUPERVISEUR REQUIS

Deux raisons distinctes, et la seconde compte plus que la première.

1. **La baseline n'est pas sur ce PC.** `data/cache/replays/halo_infinite/baseline_2026-08-03/`
   et son `SHA256SUMS.txt` sont restés sur l'autre poste. Consigne E respectée à la lettre :
   **rien n'a été régénéré** sous `data/cache/replays/`. Les 3 artefacts en place sont intacts
   (mtime 2026-08-04 09:56) ; leurs empreintes actuelles, relevées en lecture seule pour que la
   comparaison soit faisable dès l'arrivée de la baseline :
   `000d5950.json` `d028dff5…871e1` · `01e1f945.json` `3a24ca13…12d683` · `64e8adfa.json`
   `bf3f2182…892275`.

2. **Le gate « bit-identique » ne peut PLUS passer après cette intégration, et c'est normal.**
   `Coverage.Objectives` est déclaré **sans `omitempty`** (`coverage.go:114`) : tout artefact
   régénéré porte désormais `"objectives"` dans son bloc `coverage`, y compris sur un match sans
   objectif. Le document a un champ de plus **par construction** — c'est précisément ce que
   re-mode-score apporte. Un artefact régénéré DIFFÉRERA donc de la baseline du 03/08, et ce
   n'est pas une régression.

   **À trancher par le superviseur** : re-poser la baseline après l'intégration (le gate
   redevient une comparaison d'empreintes sur la nouvelle référence), ou comparer
   structurellement en tolérant les deux clés neuves. La première est plus simple et plus sûre
   — mais elle demande de régénérer, donc la baseline du 03/08 doit d'abord être rapatriée pour
   qu'on puisse vérifier que RIEN D'AUTRE n'a bougé. Tant que ce contrôle n'est pas fait, on ne
   sait pas si l'intégration a laissé le reste du document intact.

### Décisions utilisateur (2026-08-02)

1. **Intégrer le prouvé, Ghidra APRÈS le merge.** On n'attend pas la RE pour merger.
2. **Session de restitution ciblée** pour réaligner les documents (D-P0a/b, D-P1b) — mécanique,
   pas de la recherche.

### Ordre d'intégration (dans J5 ou juste après)

1. Merger `feat/replay2d-prod` (J4) → `main` d'abord.
2. Rebaser `feat/re-mode-score` sur `main` (surface disjointe → rebase attendu propre ; le
   seul recouvrement est `mapvar/`, que J4 a laissé en `[!]` exprès).
3. **Rejouer les goldens du rejeu 2D** (les 7 grandeurs) + `go test -tags=integration -p 1`
   après rebase — la base a bougé de J3-1 à `main`.
4. Traiter D-P1a et D-P2 dans le lot d'intégration (petits, sous goldens).
5. Revue adversariale de l'intégration elle-même si le rebase produit des conflits non triviaux.

### Ce qui part en piste post-merge (J6)

- **Ghidra — nommage** (piste A-objectifs ou nouvelle) : nommer les 4 emplacements de
  zone/power-up + Oddball via le décompilateur / la lecture mémoire jeu lancé. **C'est LE
  contrôle que les 3 relecteurs ET le 1er agent pointent** — il fermerait KOTH/Oddball et
  aurait attrapé les noms faux. Fable, effort max (verdict RE).
- **Containment « lettre de zone »** (code, PAS de la RE) : croiser les formes de zone
  décodées × les événements datés → quel joueur est DANS quelle zone à l'instant d'une
  capture. Faisable en croisant les deux chantiers ; inverserait le « Quelle zone : NON »
  du chantier d'origine. Opus.
- **Le décodage d'objectifs n'a AUCUN producteur ni rendu** (mesuré) : le brancher (sync →
  `Options.Objectives`, puis web) est un lot produit à part entière, après l'intégration.

### Trous de couverture par mode, à ne pas oublier (consignés par la revue)

Slayer complet · Strongholds/zones : manque la lettre A/B/C et « combien de bases à t » ·
CTF : manque la machine d'état du drapeau (porteur/position) et 15 % de zones de livraison
sans forme · **KOTH et Oddball : presque rien** (pas d'événements nommés, sémantique de score
non établie, 1 match KOTH à vainqueur inversé non expliqué).

---

## J4 — CLOS le 2026-08-02, CI VERTE (vérifiée au niveau job)

Run CI `30758938137` (b0a76756f) : **8 jobs verts**, Go Lint (ratchet) compris, Coverage +
Baseline 21m31, Build+Test ubuntu ET windows, Frontend, Contract, OpenAPI, Lease. Commits
J4 : `f9d5a11e6` (producteur live killfeed) · `0afd83f7e` (rejeu atteignable + catalogues
title-agnostic) · `2fe1aef40` (identité + TODO échu) · `74e93d427` (docs + piste F) ·
`b0a76756f` (fix lint : `renderStringMap` morte, seule issue réintroduite par J4).

**État de J4** : sessions 1→4 + 2bis faites. **Reste hors J4, reporté à raison** : la bascule
des 8 lecteurs (phase 2) attend le critère `lignes_passe_film/morts_api ≥ 98,4 %` via
l'inversion de préséance — lot post-merge. Le killfeed enrichi (arme, assistant, tirs) est
EN BASE ; la table legacy `killer_victim_pairs` reste intacte jusqu'à la bascule.

**LE CRITÈRE EST ATTEINT — 2026-08-03, inversion de préséance implémentée (session 1/2).** Il l'est
par une voie que la formulation d'origine n'anticipait pas : ce n'est PAS la passe de film qui est
montée à 98,4 %, c'est la **préséance qui s'est inversée**. Le crédit devient la base (98,5 % de
l'oracle dédupliqué, PARTOUT et pas seulement sur le périmètre film) et le film ENRICHIT les morts
qu'il couvre — sans jamais en retirer. Mesure sur copie de la base de dev : **124 694 → 134 866
lignes servies**, 73 589 enrichissements, 980 orphelins de film conservés (968 morts de bot),
**389 matchs perdaient des morts → 0**. Les états 2 et 3 de l'assistant sont identiques au chiffre
près avant/après : la fusion ne fabrique aucun fait. Halo 5 : no-op vérifié sur la donnée
(268 337 → 268 337, 0 perdue, 0 ajoutée). Statut de chaque item du §6, écarts et découvertes :
**§9 et §10 de `.ai/CONCEPTION_INVERSION_PRESEANCE.md`**. **La bascule des 20 lecteurs reste à
faire** — c'est la session 2/2, et son gate AVANT/APRÈS par lecteur est inchangé.

**Prochaine étape = J5**, et son prérequis est la **revue adversariale du lot J4** (écritures
persist/sync/migration — 2 relecteurs, règle du dépôt) AVANT le merge. Base de diff du lot à
risque : `ea3cfc88b..b0a76756f` scopé sur `internal/{persist,migration,sync/killcollector}`.

### Restitution des docs re-mode-score — CLOSE le 2026-08-02 (commit `65f27a66b`, doc-only vérifié)

D-P0a/b et D-P1b traités, chaque correction adossée à sa source (TSV, code, corps du document).
Meilleur que le mandat : les 10 lignes de la table réalignées (récompenses → statistiques),
pas seulement les 2 signalées. Deux refus VALIDÉS : la ligne §1 mêlant décodage non livré et
compteurs datés (arbitrage, hors mandat) ; les entrées antérieures du thought_log (ne pas
falsifier un journal daté — l'entrée neuve en tête dit l'état réel).
**Dette consignée pour l'intégration** : la TSV porte `zone comp 2 B = deaths` (8/8, a servi
au pont d'identité §20.3) absent de `namedStatSlots`. À trancher à l'intégration : légitime
(deaths n'est pas un événement d'objectif à nommer) ou lecteur manquant. Croiser avec le
constat R1 « TSV concorde ligne à ligne avec namedStatSlots » — l'un des deux a une exception.

---

## REVUE ADVERSARIALE DU LOT J4 — 2 relecteurs Fable aveugles, 2026-08-02

**Verdict : 0 P0, 3 P1, ~6 P2. Le cœur anti-ART est SOLIDE** (les deux relecteurs, ~35
conditions vérifiées cumulées : INSERT purs, lectures `_latest`, préséance testée 2 sens,
3 états assistant tenus, `killer_victim_pairs` intacte, allowlist inchangée). Les P1 sont des
trous de COUVERTURE, pas des bugs — le producteur live est prouvé, ce qui l'entoure ne l'est
pas encore. **À corriger avant le merge J5.**

| # | grav | où | défaut | recoupé ? |
|---|---|---|---|---|
| J4R-1 | P1 | `steps_shared_kill_events_from_pairs.go:85-176` | migration de reprise SANS test comportemental (jouée sur base vide) — dédup 46,5 %, une passe/match, préséance film au backfill régressables sans test rouge | B seul |
| J4R-2 | P1 | `events_completion_persister.go:250-296` | second producteur (complétion) écrit `match_kill_events`, JAMAIS asserté | B seul |
| J4R-3 | P1 | `steps_shared_kill_events_from_pairs.go:177-183` + persist/killcollector/seed | littéraux `kill-feed`/`credit-seul` en multiples copies + commentaire renvoyant à `steps_shared_kill_events_from_pairs_test.go` **qui n'existe pas** (vérifié) | **A ET B** |
| J4R-4 | P2 | `killcollector/collector_test.go:261` | porte `CapFilmWeaponShots` non couverte sans fixture (skip = faux vert — 6e occurrence du motif) | B |
| J4R-5 | P2 | `kill_events_credit.go:116`, migration:163, `convergence_backfill_events.go:323` | erreurs avalées : `continue`/`Skipped++` sans log ni compteur (anti-pattern n°10) | B |
| J4R-6 | P2 | `ARCHITECTURE_V6.md` FR+EN, `copilot-instructions.md`, + 4 commentaires code | `v_killer_victim_full` droppée mais documentée « garantie v6 » ; « vue de compatibilité killer_victim_pairs » qui n'existe pas ; en-tête `events_completion_persister.go:20` faux | **A ET B** |
| J4R-7 | P2 | `cmd/rebuild_mp/main.go:30,105` | `DROP ... CASCADE` ne supprime pas les vues DuckDB → l'outil de réparation ART avorterait ; **défaut PRÉEXISTANT**, mais le lot a réécrit la liste + le commentaire sur cette prémisse fausse | B |

**Ronde de correction avant merge** (session dédiée, Opus, périmètre fermé) : J4R-1/2/3
(2 tests comportementaux + centralisation littéraux avec vrai garde-rail) ; J4R-4/5/6
(règles écrites : test capability, logs, doc bilingue) ; J4R-7 = corriger le commentaire
mensonger + CONSIGNER la dette (ne pas réécrire l'outil, hors périmètre). Puis **ronde 2** :
relecture des seules corrections par un contexte frais (skill §8, 2 rondes max).

### RONDE DE CORRECTION — CLOSE le 2026-08-02. 7/7 traités, aucun `[!]`.

| # | statut | correction livrée | ce qui la garde |
|---|---|---|---|
| J4R-1 | `[x]` | migration de reprise jouée sur base **peuplée** : dédup (4 lignes → 2 morts), une passe par match + passes distinctes, préséance (match à film non réimporté, source du dégât conservée), récence (film postérieur gagne) | `migration/steps_shared_kill_events_from_pairs_test.go` — 4 tests, **4 mutations vérifiées rouges** : `DISTINCT` retiré → dédup ; `NOT IN` neutralisé → préséance ; passe constante → une-passe-par-match ; ordre `_latest` inversé → récence |
| J4R-2 | `[x]` | chemin complétion asserté : lignes présentes, `assist_known = FALSE`, portée déclarée, source/parts NULL ; + préséance film et non-addition des passes | `TestEventsCompletionPersister_EcritLaTableCanonique` + `_PreseanceFilm` + assertion canonique ajoutée à `_KVIdempotent` — **supprimer l'appel `persistCreditKillEvents` fait rougir 2 tests** (vérifié) |
| J4R-3 | `[x]` | vocabulaire de portée centralisé dans **`internal/domain/killscope`** (feuille SANS import). 4 écrivains + CLI + 3 tests le lisent ; `persist.ReadPathLiveFeed`/`ReadOriginCreditOnly` et `killcollector.CreditReadPath`/`CreditReadOrigin` supprimées (pas d'alias). Commentaire mensonger corrigé — et le fichier qu'il citait EXISTE désormais | `archlint/no_raw_kill_scope_literal_test.go` (walk `internal/` + `cmd/`, allowlist **vide**, commentaires Go et SQL sautés) — **vérifié rouge sur littéral réintroduit** ; + `killscope_test.go` épingle les valeurs de fil et leur distinction. Referme J4R-4-bis : écrire `read_path <> 'kill-feed'` exige un littéral, que le ratchet refuse |
| J4R-4 | `[x]` | porte `CapFilmWeaponShots` testée **sans fixture** : `collectShots` appelée directement sur un chunk de réplication synthétique (instrument de `shots_test.go`), les deux sens vérifiés | `killcollector/shots_capability_test.go` — **inverser la condition fait rougir les DEUX assertions** (vérifié). Tourne partout où DuckDB tourne, donc en CI |
| J4R-5 | `[x]` | 3 sites : `CreditKillEventsFromPairs` (prend `ctx`, compte + `WarnContext`, compteur `killsource_live_couples_sans_victime_nommee`) ; reprise migration (`compterCouplesSansVictime` **avant** l'INSERT, même périmètre que le filtre, compteur `killsource_reprise_couples_sans_victime_nommee`) ; `convergence_backfill_events.go` (`ErrorContext` + `convergence_events_persist_failed_total`, **loggé avant** le `Skipped++`) | `TestCoupleSansNomDeVictimeEstEcarteEtCompte` (lit le compteur via `observability.LoadCounter`) + `TestRepriseCompteLesCouplesSansVictime` (nom vide ET nom NULL ; vérifie que le comptage ne compte PAS les matchs déjà couverts) |
| J4R-6 | `[x]` | `v_killer_victim_full` retirée des garanties v6 dans `docs/ARCHITECTURE_V6.md`, `docs/FR/ARCHITECTURE_V6.md` et `.github/copilot-instructions.md` (CHANGELOGs NON touchés — historiques datés). **5** commentaires code corrigés : `order.go:175`, `steps_shared.go:212`, `steps_shared_kill_events_from_pairs.go:116`, + 2 références stales trouvées en vérifiant (`match_view_repo_extras.go:61`, `ops/snapshot_read.go:88`). En-tête `events_completion_persister.go` : 3 tables écrites, pas 2 | `TestRunForDB_Shared_V6ViewsExist` **réparé** : il exigeait la vue supprimée (il aurait échoué le jour où son skip serait levé) ; il vérifie désormais les 3 vues restantes **et que la supprimée ne revient pas** |
| J4R-7 | `[x]` | commentaire de `cmd/rebuild_mp/main.go` corrigé **sur mesure**, pas sur parole : sonde DuckDB jouée — `DROP TABLE t CASCADE` réussit mais LAISSE la vue au catalogue, et le `CREATE VIEW` de recréation échoue en `Catalog Error: View with name "v" already exists!`. Le mode de panne annoncé était l'inverse du vrai (« vue manquante en silence » → en fait l'outil **avorte**, transaction annulée, base intacte). Outil **non réécrit** (hors périmètre) | aucun test (outil `//go:build ignore`) — dette consignée ci-dessous |

**Dette ouverte par J4R-7** : `cmd/rebuild_mp` est **inutilisable en l'état** dès que la table
reconstruite porte des vues dépendantes — c'est-à-dire toujours. La réparation ART par CTAS est
donc indisponible tant que l'outil ne DROPpe pas explicitement les vues capturées avant de les
recréer. Défaut **PRÉEXISTANT** (le lot J4 n'a fait qu'en réécrire la liste). **Lot séparé**, hors
ronde. Ne pas « améliorer » la liste `dependentViews` en croyant refermer le trou.

### Découvertes de la ronde — CONSIGNÉES, NON TRAITÉES (hors périmètre fermé)

1. **`persist.FilmReadPaths` recopie `killsource.PathWalk`/`PathScan`.** `persist` déclare
   `[]string{"marche", "scan"}` alors que les valeurs vivent, typées, dans
   `games/halo_infinite/film/killsource/kill.go:126,129`. La copie est probablement délibérée
   (`persist` ne doit pas importer un paquet title-specific) mais elle n'est ni datée ni
   verrouillée. `killscope` serait le domicile naturel. Le ratchet J4R-3 ne les police PAS :
   « marche » et « scan » sont des mots courants, un grep produirait du bruit.
2. **Deux erreurs avalées voisines de celle de J4R-5**, dans le même `switch` :
   `convergence_backfill_events.go` — `MarkEventsEmptyDefinitive` et `MarkNoFilmDefinitive`
   incrémentent `Skipped++` sans log ni compteur. Même anti-pattern n°10, à 20 lignes du site
   corrigé. Non traités : hors des 3 sites nommés par le constat.
3. **Quatre tests `migration` skippent en permanence** (`sharedBaseSchemaIsGlobal()` est faux
   depuis Phase 1.5 b23) : `TestRunForDB_Shared_V6ViewsExist` et ses 3 voisins. La couverture est
   assurée ailleurs (`TestTitleStepsRunEndToEnd_Shared`) et le skip est documenté — mais c'est
   le motif « skip = faux vert » qui a produit J4R-4, à l'échelle de 4 tests. À trancher :
   supprimer, ou rebrancher sur le chemin title-owned.

---

## STRATÉGIE DE MERGE — RÉVISÉE le 2026-08-02 (décision utilisateur)

**On ne merge PLUS J4 seul. On finalise le VISIBLE, puis on merge un tout cohérent.**
`main` = déploiement prod ; l'utilisateur préfère une release propre et complète.

### Périmètre AVANT le merge (tout sur `feat/replay2d-prod`)

| lot | état | ce qu'il rend | sessions |
|---|---|---|---|
| **J4-fix** — ronde de correction post-revue (7 constats) + ronde 2 | CLOS 2026-08-02 | — | 1-2 |
| **Killfeed VISIBLE — 1/2 : l'inversion de préséance** (crédit = base, film = enrichissement) + re-backfill | **FAIT 2026-08-03** — `.ai/CONCEPTION_INVERSION_PRESEANCE.md` §9. 124 694 → **134 866** lignes servies, **98,5 %** de couverture, **0 match ne perd de mort** (389 avant) | la donnée est en base et complète ; rien n'est encore à l'écran | 1 |
| **Killfeed VISIBLE — 2/2 : la bascule des lecteurs** + `seed_demo` (S1) + `rebuild_mp` (S2) | **FAIT 2026-08-03** — `PLAN_BRANCHEMENT_KILLSOURCE.md` §2.4/§2.5. Bascule DIRECTE sur `match_kill_events_latest` (la vue de compat était infaisable : la table est devenue la base crédit) ; aucun lecteur ne filtre `publishable` (un filtre coûterait 47 037 morts sur 366 matchs) ; S1 et S2 étaient déjà faits. Gates : unit + intégration verts, goldens sur films réels verts, 3 artefacts bit-identiques à la baseline, ratchet lint 0, 3 garde-rails vus rouges | **LE vrai visible, mesuré** : agrégats carrière dégonflés (« 101 → 29 » retrouvé au chiffre près, Q27 15 741 → 10 486 sur JGtm), journal de match sans doublons (1 832 → 458 lignes pour 458 instants), et un opposant fantôme retiré des agrégats **Halo 5** (161 frags / 127 morts sur le joueur le plus actif) | 1 |
| **Intégration re-mode-score** — intégrer le code objectifs + P1/P2 de sa revue (docs déjà restitués) | **FAIT 2026-08-05** — piste C-bis, statuts ci-dessus. Intégré par MERGE (26 des 42 commits conflitent sur le journal ; recouvrement réel = 4 fichiers). D-P1a, D-P2 et la dette TSV traités, 7 mutations vues rouges. 2 conflits sémantiques invisibles à git corrigés. Contrat régénéré : le calque publiait 2 champs non décrits, `contracttest` l'a attrapé. **RESTE OUVERT : le gate des artefacts de rejeu** (voir ci-dessous) | anti-divergence : le code a atterri, son débouché rejeu reste dev (décision #5, rendu non branché) | 1 |
| **Hygiène** — rangement `.ai/` → V7.5, lot E delivery-checklist | à faire | — | 1-2 |
| **MERGE** — revue adverse finale si besoin, GO utilisateur, backfill prod | à faire | la release | 1 |

### CLARIFICATION — ce qui est « visible » et ce qui ne l'est pas

- **Le killfeed enrichi EST le visible** : c'est lui qui apporte de la valeur à l'écran une fois
  les lecteurs basculés. C'est le cœur de la release.
- **re-mode-score s'intègre pour NE PAS diverger**, mais son débouché principal (le calque
  d'objectifs) est dans le **rejeu 2D, qui reste en DEV (décision #5)**. On fait atterrir le
  code proprement ; on ne branche pas son affichage rejeu avant le merge.

### RESTE POST-MERGE (branches courtes depuis `main`)

Rejeu 2D public (piste F) · cartes Catalyst/Vagabond + les 12 autres (piste B) · **Ghidra
KOTH/Oddball** (recherche, JAMAIS bloquante pour un merge) · containment lettre-de-zone ·
précision projectiles (piste E). Le rejeu et ses cartes restent **dev** tant que #5 tient.

### DISCIPLINE OBLIGATOIRE pendant la finalisation

**Merger `origin/main` DANS `feat/replay2d-prod` régulièrement** (à chaque début de session,
ou au moins entre chaque lot). Sinon on reproduit exactement la divergence que la
réconciliation a coûté une session entière à réparer — et un acteur pousse `main` en parallèle.

### Rangement `.ai/` — session dédiée, quand la branche est LIBRE

Pas en vrai simultané de la correction J4 (même worktree = collision d'index git). À jouer
entre deux lots. Tâche SOIGNEUSE : re-vérifier vivant/clos doc par doc (ne pas archiver un
plan encore actif : capacités, objectifs, cartes, variables jetées non faits), et ne pas
casser les références de code (`dumps/` est lu par du code : `cmd/replay-build`, `mapvar`).

### Ronde 2 (relecture des corrections, contexte frais) — CLOSE le 2026-08-02

**Verdict : 6 défauts FERMÉS (J4R-1/2/3/4/5/7), J4R-6 P1 fermé avec 2 commentaires de test
résiduels (P2), AUCUN défaut introduit.** La boucle a convergé : **3 P1 → 0 P1** (skill §8,
2 rondes, décroissance stricte respectée). Chaque fermeture vérifiée par la mutation-test qui
devrait rougir (retrait DISTINCT, suppression d'appel, inversion de condition, injection de
littéral → garde-rail rouge re-vert après retrait). **La revue adversariale du lot J4 est
close** — 0 P0/P1 résiduel. Le producteur live, la migration de reprise et le second
producteur sont désormais gardés par des tests qui échouent quand on casse ce qu'ils protègent.

### DETTE P2 À SOLDER DANS LE LOT D'HYGIÈNE (avant merge, pas dans le lot J4)

| # | où | quoi |
|---|---|---|
| H1 | `duckdb/extra_coverage_test.go:131,137` + `player_repos_test.go:189-193` | 2 commentaires de test périmés citant `v_killer_victim_full` (résidu J4R-6, doc inversée n°9, zéro impact fonctionnel) |
| H2 | `sync/convergence_backfill_events.go:317-318` | 3e erreur avalée du même switch que J4R-5 (`case f.err != nil: res.Skipped++` sans log ni compteur) — non consignée par le lot, la ronde 2 l'a rattrapée |
| H3 | `persist/kill_events_credit.go:84` | recopie les read_path film `"marche"/"scan"` de `killsource/kill.go:126,129` sans verrou d'égalité — J4R-3 n'a centralisé que read_origin ; centraliser aussi les read_path (frontière killsource title-specific ↔ domain/killscope à trancher) |
| H4 | `cmd/rebuild_mp/main.go` | l'outil de réparation ART avorte dès qu'une vue dépend de la table (DROP CASCADE DuckDB ne supprime pas les vues) — outil non réécrit, dette de J4R-7 |
| H5 | `migration_test.go` (+ voisins) | 4 tests migration en skip permanent (`sharedBaseSchemaIsGlobal()` faux depuis Phase 1.5) — assertions dormantes |

| H6 | `.ai/` racine | rangement des docs 100% clos → V7.5 (HANDOFF_DUMPS, SESSION_CAPTURE, PLAN_RECONCILIATION, PLAN_REJEU_2D_FIABILISATION, HANDOFF_KILLSOURCE_REPRISE, HANDOFF_REPLAY_2D + ceux clos d'ici le merge). **Reporté au lot hygiène** (2026-08-02) : chaque déplacement casse des liens croisés (master plan + inter-plans) à mettre à jour ; d'autres plans se clôturent d'ici le merge → une seule passe propre. Ne PAS déplacer les états de l'art (référence courante : KILLWEAPON, ADDENDUM, README_INDEX, GUIDE_WEAPON_SHOTS, CHANTIER_VOISIN, SUIVI, ETAT_DU_POC, CAHIER, CLE_USB) ni les plans encore actifs (FINALISATION, BELLE_CARTE, CAPACITES, OBJECTIFS, VARIABLES, ASSETS_ICONES, PRECISION). Vérifier qu'aucun chemin déplacé n'est lu par du code (`dumps/` l'est, les plans non — à confirmer). |
