# HANDOFF — reprise du rôle SUPERVISEUR du chantier film

> Écrit le 2026-08-03 à la passation (contexte plein). **Point d'entrée unique** du rôle
> superviseur. Le document d'autorité reste `PLAN_MASTER_FILM_KILLFEED_REJEU.md` — celui-ci
> dit où on en est, ce qui vient, et comment on travaille.

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
