# HANDOFF — reprise du rôle SUPERVISEUR du chantier film

> Écrit le 2026-08-03 à la passation (contexte plein). **Point d'entrée unique** du rôle
> superviseur. Le document d'autorité reste `PLAN_MASTER_FILM_KILLFEED_REJEU.md` — celui-ci
> dit où on en est, ce qui vient, et comment on travaille.

---

## 0. MISE À JOUR — 2026-08-03 fin de journée (les §2, §3 et §5 ci-dessous sont PÉRIMÉS sur l'état)

**LES DEUX SESSIONS « KILLFEED VISIBLE » SONT FAITES, VÉRIFIÉES ET POUSSÉES.** Le §5 (prompt du
lot 1/2) et la session 2/2 sont derrière nous. État réel à la reprise :

| | |
|---|---|
| Branche | `feat/replay2d-prod`, HEAD **`62baf42df`**, arbre propre, **poussée**, 0 retard sur `main` |
| Lot 1/2 — inversion de préséance | **CLOS** (`4ab023d56`) — CI verte VÉRIFIÉE AU NIVEAU JOB (4 workflows, 0 job rouge) |
| Lot 2/2 — bascule des 17 lecteurs | **CLOS** (`39da43fbf` + merge main `f36b75273` + 2 correctifs superviseur `10068dad2`, `62baf42df`) |
| CI du push final | run **30810162011** (+ 3 workflows frères) **EN COURS à la passation** — **LE VÉRIFIER AU NIVEAU JOB** (`gh run view 30810162011`) : c'est le seul gate encore ouvert du lot 2/2 |
| Donnée en base (dev) | celle de l'inversion : la table sert 134 866 lignes, 98,5 % partout ; `killer_victim_pairs` reste une TABLE (base crédit), plus aucun lecteur ne la lit |

**Ce que le superviseur a vérifié sur pièces aujourd'hui** (méthode : ne pas refaire, s'y fier) :
suites `-p 1` + intégration + goldens (fixtures ABSOLUES) rejouées localement sur les deux lots ;
mesure DB refaite DE ZÉRO sur copie neuve (faux repo root `LEVELUP_REPO_ROOT`) — tous les chiffres
des comptes rendus reproduits au chiffre près ; 2 mutations de garde-rails injectées/vues
rouges/retirées (lot 1) ; artefacts de rejeu 3/3 bit-identiques à la baseline par empreintes.

**Arbitrages rendus (ne pas rouvrir sans motif neuf)** :
1. **Bascule DIRECTE des lecteurs** (la vue de compat du §2.1 du plan de branchement était
   infaisable : circulaire depuis l'inversion).
2. **`publishable` ne filtre aucun lecteur** (le filtre coûterait 47 037 morts ; la colonne
   conditionne l'affichage futur de l'ARME, pas l'existence des morts).
3. **Artefacts de rejeu : baseline re-posée** — les versions du 02/08 de `01e1f945`/`64e8adfa`
   sont perdues (provenance des ~650 Ko inexpliquée), la forme courante fait référence :
   `data/cache/replays/halo_infinite/baseline_2026-08-03/` + `SHA256SUMS.txt`. Le gate se joue
   PAR COMPARAISON d'empreintes, jamais en reconstruisant par-dessus.

**Pièges NOUVEAUX (en plus du §7)** :
- Le test du plancher du persister (`TestRefusDuPersister`) vit derrière `//go:build integration` :
  une mutation « vue verte » sous un run nu est un FAUX VERT — muter sous le tag.
- **Hook pre-push `shared-social-gate` : `bash` peut résoudre vers le stub WSL** (non installé sur
  ce PC) → échec en 0,1 s sans message utile. Pousser depuis Git Bash (ou corriger le PATH). Le
  hook ne s'était JAMAIS déclenché depuis le changement de PC (glob `platform/duckdb/**`).
- La régénération d'`openapi.yaml`/`generated.ts` à un merge peut créer une collision au ratchet
  contrat (`lint-contract-ratchet`) : `TeammatesQueryRequest` est passé en baseline datée —
  **re-shim candidat au lot hygiène**.
- 19 fixtures de tests semaient l'ancienne table (passées au journal en phase 2) : les tests qui
  « couvraient » la bascule ne la couvraient pas.

**Visible à l'écran après cette bascule** (pour la release note / à montrer à l'utilisateur) :
duel de contrôle 101 → 29 frags ; agrégats carrière dégonflés partout (facteur moyen 1,879) ;
côté Halo 5, un « joueur fantôme » (bots en chaîne vide, 161 frags / 127 morts) SORT du top-10
némésis — c'est une correction, mais elle se voit.

**LA SUITE, dans l'ordre (stratégie §3 inchangée sur le fond)** :
1. **Verdict CI au niveau job** du run 30810162011 — si rouge, traiter avant tout.
2. **Intégration `feat/re-mode-score`** (worktree `.claude/worktrees/re-mode-score`) : rebaser sur
   `feat/replay2d-prod`, rejouer les goldens, intégrer (revue déjà faite : 0 P0, 1 P1).
3. **Hygiène** : H1-H6 (§ dédiée du master plan) + lot E `delivery-checklist` + re-shim
   `TeammatesQueryRequest` + le défaut préexistant `CAST(now() AS TIMESTAMP)` consigné dans
   `steps_shared_kill_events_from_pairs.go`.
4. **MERGE vers `main`** — GO utilisateur EXPLICITE + fenêtre backfill prod convenue (la reprise
   SQL→SQL joue en prod sans les films, ~15 s + passe crédit ~5 min par titre). Push `main` =
   déploiement prod automatique : prévenir AVANT.

**Consignes MASTER PLAN pour le repreneur** : le document d'autorité reste
`PLAN_MASTER_FILM_KILLFEED_REJEU.md` — il est À JOUR au 2026-08-03 (lots 1/2 et 2/2 marqués FAIT
dans le tableau des jalons ; le plan du prochain lot est sa **piste C-bis**, ligne ~1757, avec le
registre P1/P2 de la revue). À chaque clôture de lot : statuer les items du plan concerné
(`[x]`/`[~]`/`[!]`), reporter le statut dans le tableau des jalons du master plan, entrée
thought_log, et vérifier les comptes rendus SUR PIÈCES avant tout commit (§1 — les trois règles
payées cher restent la méthode).

### Prompt du prochain lot (intégration re-mode-score) — à remettre tel quel

```
Modèle recommandé : Opus 5 — effort élevé.

Session EXÉCUTEUR — intégration de feat/re-mode-score dans feat/replay2d-prod (piste C-bis).
Dépôt principal, branche feat/replay2d-prod.
COMMENCER PAR : merger origin/main dans la branche (discipline anti-divergence), et VÉRIFIER
que la CI du dernier push (run 30810162011 et suivants) est verte AU NIVEAU JOB — si rouge,
STOP et remonter au superviseur avant tout.

1. Invoquer plan-execution, arch-rules, db-schema.
2. Lire : master plan PISTE C-BIS EN ENTIER (≈ ligne 1757 — le plan d'intégration, le registre
   P1/P2 de la revue, et la raison d'être : intégrer pour NE PAS diverger, le débouché rejeu
   reste dev) ; le §0 du présent handoff (pièges nouveaux).
3. Périmètre FERMÉ = la piste C-bis : rebaser feat/re-mode-score (worktree
   .claude/worktrees/re-mode-score, base b9f163d80 = J3-1, AVANT l'inversion et la bascule) sur
   feat/replay2d-prod ; résoudre ; traiter le P1 de la revue et les P2 rentables ; intégrer.
   ⚠ pièges : conflits thought_log = résolution en BASH octets bruts ; retirer une jonction
   AVANT tout `git worktree remove` ; si le code objectifs lit le kill-feed, il lit
   match_kill_events_latest — killer_victim_pairs est INTERDITE aux lecteurs (le garde
   TestAucunLecteurNeLitLAncienneTable doit rester vert, ne pas l'allowlister).
4. GATE : go test ./... -p 1 ; go test -tags=integration -p 1 ./... (-p 1 NON NÉGOCIABLE) ;
   TestGoldenFilms avec KILLSOURCE_FIXTURES=<chemin ABSOLU> (la branche date d'avant J4 : les
   goldens sont LE filet du rebase) ; artefacts de rejeu comparés PAR EMPREINTES à
   data/cache/replays/halo_infinite/baseline_2026-08-03/SHA256SUMS.txt — jamais en
   reconstruisant par-dessus ; ratchet lint 0 ; CI au niveau job après push (pousser depuis
   Git Bash — le hook shared-social-gate échoue sous un bash WSL).
   Les nouveaux tests doivent ÉCHOUER quand on casse ce qu'ils gardent (mutation).
5. INTERDITS : toucher au décodeur, aux producteurs killsource ou aux lecteurs basculés ;
   écrire sur le VPS de prod ; productioniser le rejeu (décision #5 : on reste en dev).
   TOUTE ACTION AU-DELÀ = STOP et question au superviseur.
6. Clôture : statuts dans la piste C-bis + tableau des jalons du master plan, entrée
   thought_log, compte rendu : ce qui a été intégré, le sort du P1 et de chaque P2, les
   conflits du rebase et leur résolution, découvertes, décisions en attente.
```

Le reste du document (rôle §1, décisions §4, dette §6, pièges §7) reste valable.

---

## 1. TON RÔLE — superviseur, pas exécuteur

Tu tiens le master plan, tu ouvres les sessions exécuteur avec leur ordre de mission, tu
**vérifies les gates sur pièces** (CI au niveau JOB, tests, mesures), tu fais les commits et
les merges, tu tranches les arbitrages. **Tu ne codes pas.** L'utilisateur lance les sessions
exécuteur dans d'autres conversations et te rapporte leurs comptes rendus.

Trois règles que ce chantier a payées cher :

1. **Vérifier sur pièces, jamais sur parole.** Un compte rendu d'exécuteur se contrôle
   (`gh run view <id>` au niveau JOB — pas seulement le workflow ; `git log`, `git status`,
   un grep du fait affirmé). Plusieurs fois, la vérification a changé la réponse.
2. **Un gate qui ne peut pas échouer ne garde rien.** Motif rencontré **6 fois** :
   `t.Skip`/`continue` sans assertion, garde CI au chemin doublement préfixé, ratchet filtré
   par chemin, golden qui skippe sans fixture. Le chercher systématiquement.
3. **Merger `origin/main` DANS la branche à chaque début de session.** La divergence a déjà
   coûté une session entière de réconciliation. Au 2026-08-03 : **0 commit de retard**.

Modèles/effort par type de session : §7 du master plan (« Modèle et niveau d'effort »).
La règle courte : **décider → monter le modèle ; appliquer sous gate → le filet protège**.

---

## 2. ÉTAT AU 2026-08-03 — vérifié sur pièces

| | |
|---|---|
| Branche vivante | **`feat/replay2d-prod`**, HEAD `d8f420e81`, **arbre propre**, poussée |
| À jour de `main` | **oui, 0 commit de retard** (merge `1a0c1eb5c`) |
| CI | **verte** (dernier run de code vérifié au niveau job : 8/8) |
| Jalons clos | **J0** captures · **J1** CI · **J2** goldens décodeur · **J3** dette lint 0 · **J4** killfeed en base **+ revue adversariale complète (2 relecteurs, 2 rondes, 3 P1 → 0)** |
| Dernier livrable | `CONCEPTION_INVERSION_PRESEANCE.md` — la conception mesurée du prochain lot |

### Ce qui est acquis, concrètement

- Le **killfeed enrichi** (arme du kill via `source_tag`, assistant nommé, parts de dégâts)
  est **en base** pour les deux titres, sans doublon, sans surface ART, gardé par des tests
  qui échouent quand on casse ce qu'ils protègent.
- Le **rejeu 2D** est atteignable (lien conditionnel) mais **gardé local** — décision #5 :
  « on reste en dev », pas de productionisation.
- Le **décodeur** est verrouillé par goldens, dont un qui **tourne en CI** sans fixture.
- **Aucun lecteur n'est basculé** : `killer_victim_pairs` reste la source des pages. **Donc
  la valeur n'est pas encore visible à l'écran.**

---

## 3. LA STRATÉGIE ARRÊTÉE — finaliser le VISIBLE avant de merger

Décision utilisateur du 2026-08-02 : **on ne merge pas J4 seul.** `main` = déploiement prod ;
il veut une release propre et complète. Périmètre avant merge :

1. **Killfeed VISIBLE** ← *le prochain lot, prompt au §5* — inversion de préséance, bascule
   des lecteurs. C'est lui qui fait apparaître les armes et les assists à l'écran et qui
   dégonfle les agrégats carrière (« 101 frags au lieu de 29 »).
2. **Intégration `feat/re-mode-score`** — rebaser sur `feat/replay2d-prod`, intégrer le code
   objectifs (revue faite : 0 P0, 1 P1, quelques P2 ; docs déjà restitués).
3. **Hygiène** — dette H1-H6 (§ dédiée du master plan) + lot E de `delivery-checklist`.
4. **MERGE** — GO utilisateur explicite, fenêtre backfill prod convenue.

**Post-merge, branches courtes depuis `main`** : rejeu public (piste F, conçue de bout en
bout), cartes Catalyst/Vagabond (piste B), **Ghidra KOTH/Oddball** (recherche — **jamais
bloquante pour un merge**), containment lettre-de-zone, précision projectiles (piste E).

---

## 4. CE QUI A ÉTÉ TRANCHÉ — ne pas rouvrir sans motif neuf

| # | décision |
|---|---|
| #1 | Pas de nouvelle capture Cheat Engine — les dumps sont faits, l'analyse se fera plus tard |
| #2 | Garde local du rejeu : **comprendre le CTF d'abord** (564 tirs perdus vs 44 en Slayer), puis redécider |
| #4 | Icônes d'armes : extraire, **montrer avant d'intégrer** (gate humain) |
| #5 | **Rejeu en prod : on reste en DEV.** Collecte des données oui, artefacts à la demande plus tard |
| #6 | Précision projectiles : timebox 2 sessions, verdict écrit |
| #7 | Palette Forge intégralement sauvée sur la clé (4 variantes, vérifiées à l'octet) |
| — | **Base = les couples** (133 886, 98,5 %), pas « toutes les morts » — ne pas changer la définition du contenu en même temps que la préséance |
| — | **Compteur dédié** pour les 13 orphelins humain-contre-humain |
| — | Gate visuel des cartes : **artefact de revue + validation EXPRESSE** de l'utilisateur ; témoins propres à chaque carte (le fer à cheval et les ponts sont **Cliffhanger seulement**) |

**Reste à trancher par l'utilisateur** : le découpage en releases prod (à poser au moment de
la bascule, avec les options chiffrées).

---

## 5. LE PROCHAIN LOT — implémenter l'inversion de préséance

**Document d'autorité : `.ai/CONCEPTION_INVERSION_PRESEANCE.md`** (§6 porte le plan
d'implémentation). Ce qu'il a établi, et qui rend le lot bon marché :

- Les deux sources ne sont **pas** deux mesures indépendantes : `killsource/feed.go:59`
  appelle **le même parseur** (`analysis.ParseHighlightEvents`) que celui qui alimente
  `highlight_events`. **Vérifié.** Donc aucun écart d'horloge.
- Appariement : clé **`(match_id, time_ms)`, tolérance ZÉRO** — bijection stricte
  (73 589 = 73 589). Dès ±50 ms, une ligne de film capture plusieurs morts : la tolérance
  achète 8 appariements et **coûte l'unicité**.
- **980 orphelins de film CONSERVÉS** (819 victimes bot, 149 tueurs bot) : le kill-feed de
  l'API est humain-seul — les rejeter traiterait une absence de mesure comme une mesure
  d'absence.
- Cible **134 866 lignes** (contre 124 694), +10 172 morts, 73 589 enrichies.
- **Chiffre du chantier corrigé** : l'oracle brut sur-comptait (15 120 groupes en double
  dans `highlight_events`) → le crédit tient **98,5 % PARTOUT**.
- **Aucune migration de schéma, aucun re-décodage** : reprise SQL→SQL, jouable en prod sans
  les 23 Go de films.

### Prompt de la session (à remettre tel quel)

```
Modèle recommandé : Opus 5 — effort élevé.

Session EXÉCUTEUR — implémenter l'inversion de préséance crédit↔film (killfeed visible, 1/2).
Dépôt principal, branche feat/replay2d-prod.
COMMENCER PAR : merger origin/main dans la branche (discipline anti-divergence).

1. Invoquer plan-execution, arch-rules, db-schema.
2. Lire : .ai/CONCEPTION_INVERSION_PRESEANCE.md EN ENTIER (c'est le document d'autorité, tout
   y est mesuré — ne PAS re-mesurer l'appariement) ; master plan §J4 + « STRATÉGIE DE MERGE » ;
   PLAN_BRANCHEMENT_KILLSOURCE phase 2.
3. Périmètre FERMÉ = le §6 du document de conception :
   - Inverser la logique : le CRÉDIT devient la base (toutes les morts, 98,5 %), le FILM
     ENRICHIT les morts qu'il couvre (source_tag, assistant nommé, parts) sans jamais en
     retirer. Appariement (match_id, time_ms), tolérance ZÉRO — la bijection est prouvée, ne
     pas introduire de tolérance.
   - Les 980 orphelins de film sont CONSERVÉS (bots). Poser le COMPTEUR DÉDIÉ pour les 13
     orphelins humain-contre-humain (seule population dont le mécanisme n'est pas démontré).
   - Les TROIS états de l'assistant survivent : une mort crédit-seule reste assist_known=FALSE
     (« on ne sait pas ») ; une mort enrichie porte l'assistant nommé. Aucune combinaison ne
     doit fabriquer un « pas d'assistant » jamais observé.
   - Le garde-rail que la conception a identifié comme manquant : une passe ne peut JAMAIS
     porter moins de morts que la base crédit du match. Le poser et le tester.
   - Re-backfill local (SQL→SQL, pas de re-décodage) puis MESURER : la table atteint-elle
     134 866 lignes et 98,5 % de couverture ?
   - RÉCONCILIER assist_extra_count : le DDL déclare 0 comme déclencheur de migration vers
     une table fille, la mesure dit 1 (était 5). Corriger le seuil déclaré OU documenter
     l'écart — pas les deux versions en même temps.
4. GATE : go test ./... ; go test -tags=integration -p 1 ./... (-p 1 NON NÉGOCIABLE) ;
   TestGoldenFilms EN LOCAL avec KILLSOURCE_FIXTURES=<chemin ABSOLU> ; les 3 artefacts de
   rejeu bit-identiques ; ratchet lint à 0 ; CI vérifiée AU NIVEAU JOB après push.
   Les nouveaux tests doivent ÉCHOUER quand on casse ce qu'ils gardent (le vérifier par
   mutation, comme la ronde J4R l'a fait).
5. INTERDITS : basculer les lecteurs (c'est la session 2/2, APRÈS que la couverture soit
   mesurée) ; toucher au décodeur ; écrire sur le VPS de prod ; changer la définition du
   contenu (base = les couples, tranché). TOUTE ACTION AU-DELÀ = STOP et question au
   superviseur.
6. Clôture : statuts dans CONCEPTION_INVERSION_PRESEANCE.md + master plan (§J4 + journal),
   entrée thought_log, compte rendu : couverture atteinte, le SELECT avant/après, les
   compteurs, découvertes, décisions en attente.
```

**Session 2/2 (après)** : la bascule des 8 lecteurs + `seed_demo` (S1) + `rebuild_mp` (S2),
avec le gate AVANT/APRÈS par lecteur. **Ré-inventorier les sites par grep** — l'inventaire du
plan était incomplet (un 11e site, `v_gamertag_lookup`, a été trouvé en cours de route).

---

## 6. DETTE ET DÉCOUVERTES À NE PAS PERDRE

**Dette d'hygiène H1-H6** : section dédiée du master plan (commentaires de test périmés, 3e
erreur avalée, `FilmReadPaths` sans verrou, `rebuild_mp` inutilisable dès qu'une vue dépend
de la table, 4 tests migration en skip permanent, rangement `.ai/` → V7.5).

**Travail parallèle terminé, à intégrer** : `feat/re-mode-score` (worktree
`.claude/worktrees/re-mode-score`) — objectifs/score/formes de zones. Revue faite, docs
restitués. Elle est partie de `b9f163d80` (J3-1) : **rebaser** puis **rejouer les goldens**.

**Trous de couverture par mode** (consignés par la revue) : Strongholds/CTF bien couverts ;
**KOTH et Oddball presque pas** ; la « lettre de zone » (quel joueur dans quelle zone) est
désormais **faisable sans RE** en croisant formes × événements datés.

---

## 7. PIÈGES DE CE CHANTIER — les plus coûteux

1. **Le faux vert** (6 occurrences) : `t.Skip`/`continue` sans assertion, chemin de garde
   faux, ratchet filtré. Un test qui passe avec ET sans le correctif ne teste rien.
2. **`TestGoldenFilms` skippe sans `KILLSOURCE_FIXTURES`** — et les tests Go s'exécutent
   depuis le répertoire du PAQUET : un chemin relatif fait skipper en silence. **Chemin
   ABSOLU obligatoire.** (Erreur commise par le superviseur précédent.)
3. **`-p 1` non négociable** sur `-tags=integration` : le driver DuckDB est mono-process,
   le parallèle donne des faux verts avec des durées fantômes.
4. **Ne jamais publier un taux par arme sur corpus entier** : il INVERSE l'ordre
   MA40/Sidekick — réponse fausse, pas imprécision.
5. **Conflits `thought_log.md`** : résoudre en BASH sur octets bruts (PowerShell 5.1
   double-encode l'UTF-8 sans BOM).
6. **Un journal daté ne se réécrit pas** — l'entrée neuve en tête dit l'état réel.
