# CHECKLIST POST-CAMPAGNE — 2026-07-12 (SUPERSÉDÉE le 2026-07-13 par ETAT_CONSOLIDE_2026-07-13.md)

> Campagne d'exécution des 9 plans + outillage CI : mergée (PR #54), déployée en prod le
> 2026-07-12, salve post-deploy exécutée (backfill engagement, lot V H5, vérifs H6 LUSR).
> Ce fichier consolide : (A) les vérifications UTILISATEUR, (B) les restes PARTIELs à
> compléter ou valider en l'état, (C) le backlog découvert en campagne.
> Statuer chaque case ; quand A et B sont vides, archiver ce fichier en V7.

---

## A — Vérifications utilisateur (passe visuelle + gates)

La prod est à jour : tout est constatable sur https://lvelup.info (ou en local `make dev`).

### A1. Admin — refonte monitoring (plan 3)
- [ ] Les 6 onglets : État `/admin` · Détections `/admin/detections` · Données
      `/admin/data` · Sync `/admin/sync` · Système `/admin/system` · Gestion
      `/admin/management` (nav séparée à droite)
- [ ] Anciennes URLs redirigent : `/admin/logs` (→ system, search préservé),
      `/admin/convergence`, `/admin/data-quality`, `/admin/access`, `/admin/titles`,
      `/admin/lab`
- [ ] Panneau Détections : tri (Occ./Dernière), filtre statut, actions
      Reconnaître / Sourdine / Résoudre / Rouvrir (note optionnelle) ; badge nav compte
      les `open` seulement
- [ ] Persistance : statuer 2-3 détections, redémarrer le serveur → statuts conservés
      (critère de succès du plan — `data/global/monitoring.duckdb`)
- [ ] État : fraîcheur pour halo_infinite ET halo_5 ; crons & heartbeats (`never` en
      rouge tant que la feature n'est pas passée) ; KPIs Fraîcheur / Disque / 5xx
- [ ] Système : ressources (tailles DB/WAL par titre, disque, restarts ≥ 1), viewer de
      logs avec filtres URL
- [ ] Esthétique harmonisée A8 (KPIs, headers de sections, tableaux)

### A2. Notifications (plan 4)
- [ ] Badge cloche = non-lues significatives uniquement (un `match_synced` frais,
      désormais `info`, ne le fait plus monter ; la page Notifications garde le compteur
      complet)
- [ ] Auto-read : ouvrir la cloche, fermer (clic extérieur / Esc / navigation) → à la
      réouverture les notifs consultées sont « Anciennes », le badge a décru ;
      « tout marquer lu » fonctionne toujours
- [ ] Toast : un `match_synced` produit toujours son toast
- [ ] Coalescence : 2 uploads média rapprochés du même joueur → une seule notif dont le
      compteur s'incrémente et qui remonte en tête
- [ ] Prod : badge JGtm bien en dessous de 57 ; plus de paires
      `objective_assigned`+`completed` ; plus de rafales `skill_tier`

### A3. Engagement Halo Infinite (plan 5)
- [ ] Match témoin `bc918a5a` : 3 courbes, « Joueur attendu » indépendante de « Équipe
      réelle », ligne « Lobby » dans le tooltip, sous-titre « forme — attendu : ta
      réponse habituelle... »
- [ ] Timeseries et Squad : sections engagement rendent correctement
- [ ] Cohérence des valeurs : Chocoboflor coefs ~0.8, Madina ~1.3-1.4 (unranked) —
      conforme à leurs profils ?

### A4. GATE HUMAIN E6b — scores engagement Halo 5 (plan 6) — DÉCISION
- [ ] La courbe engagement s'affiche sur un match H5 + badge « calibration provisoire »
- [ ] Protocole du plan (section E6) : un match intense dominé, un match intense subi,
      un match calme, la forme du jour — les scores « ont du sens » ?
- [ ] Point d'attention : taux de rejet PvP_unranked à 8,5 % (> seuil indicatif 5 %)
- **Verdict à rendre** : OK → passage H5 en `supported` (retrait auto du badge) ;
  KO → ajustement des poids `config/titles/halo_5/constants.toml [engagement]`

### A5. Match View H5 — vérification visuelle prod (plan 7, item V3)
- [ ] `7e3fa711-e2bd…` : map Tidal, mode Assassin, Dominance remplie, rang « Placement »
- [ ] `ccf64951-4d5c…` : Tidal, Capture du drapeau, playlist « Super Fiesta Fête » entière
- [ ] Un match H5 sans `damage_taken` → card Résistance ABSENTE ; un match classé →
      card Résultat attendu ABSENTE (plus de placeholders)
- [ ] Galerie média H5 : libellés maps/modes résolus, filtres peuplés ; clic média →
      match `f88f6d8b` (association prod-only) ; « Sans match » réduit

### A6. Actions data-quality (plan 2, B4/B5.5) — admin onglet Données, DRY-RUN d'abord
- [ ] B4.1 — Registry-names backfill (dry-run puis réel)
- [ ] B4.2 — Convergence aliases / xuids orphelins (local mesuré à 131 ; prod à mesurer)
- [ ] B4.3 — Lying-bits reset
- [ ] B4.4 — Mode translations
- [ ] B5.5 — migrate-media-paths (dry-run puis réel)

### A7. ADRs — engagements d'architecture long terme (plan 9)
- [ ] Relire ADR 0030 (`docs/adr/0030-persist-write-aggregates.md`) : batchs opaques,
      allowlist datée `OpenReadWrite`, garde-rail lecture `_latest`
- [ ] Relire ADR 0031 (`docs/adr/0031-title-data-source-boundary.md`) et TRANCHER
      l'amendement proposé de l'ADR 0027 (*Proposé* → *Accepté (amendé)*)
- [ ] Confirmer l'axe MT-27 ajouté au registre multi-titre

---

## B — Restes PARTIELs : à compléter ou valider en l'état

### B1. PLAN_MONITORING_TRIAGE_DETECTIONS (seul plan de campagne encore à la racine)
- [ ] **B1.6** — backfill LUSR si retard ≥ 4 j : SUSPENDU au départage « 0 candidat »
      (voir B2) ; si backlog confirmé vide → statuer « sans objet »
- [ ] **B2.4** — soak 24-48 h post-deploy (T0 = 2026-07-12) : re-mesurer le bruit prod
      (attendu : −95 %, la cascade LUSR étant éteinte)
- [ ] **B7.4** — soak 30 j ré-armé (T0 = 2026-07-12, échéance ~2026-08-11)
- [ ] B4.x / B5.5 → couvert par A6 ci-dessus

### B2. Départage LUSR « 0 candidat » (plan hotfix, observation résiduelle H6)
Le shadow LUSR est silencieux car 0 candidat — soit backlog vide (sain), soit watermark
désynchronisé qui masque un retard (piège connu).
- [ ] **Prochaine session de jeu Halo Infinite** : les matchs frais affichent leur LUSR →
      CLOS. Sinon → `lusr_v2_canonical_backfill` dry-run pour mesurer, `--commit` si retard.

### B3. PLAN_ENGAGEMENT_AGNOSTIC (racine, PARTIEL) — suspendu au gate A4
- [ ] Après verdict E6b : passage `supported` (ou recalibration) + archivage du plan en V7
- [ ] Re-backfill prod : DÉJÀ FAIT le 2026-07-12 (×2 passes par titre) — rien à refaire

### B4. PLAN_MIGRATION_SQUASH (racine, PARTIEL par conception) — décision, zéro urgence
- [ ] Option 1 : VALIDER EN L'ÉTAT (capacité + preuve zéro-perte livrées, aucun squash) —
      le plan reste dormant jusqu'au besoin
- [ ] Option 2 : GO M3→M6 — confirmer le périmètre v1 (player DBs, bloc contigu
      title-owned), baseline + invariant réel + répétition sur `../LevelUp-prod-copy`
      (rafraîchir via restic si périmée) + merge

### B5. Ménage de branches
- [ ] Replier `docs/cloture-salve-2026-07-12` (clôture + cette checklist) au prochain
      train de merge vers main
- [x] Branche `wip/gh3-i18n-seasons-labelen` : supprimée (contenu prouvé intégralement
      dans main via `42141636f`)

---

## C — Backlog découvert en campagne (à planifier, aucun bloquant)

### Outillage / CI
1. **`.dockerignore` n'exclut pas `data/titles/`** : ~17 Go scannés dans le contexte à
   chaque build Docker sur le VPS (cause plausible du piège disque BuildKit). Fix
   2 lignes + purge du cache builder.
2. **Fixture démo E2E** : 60 specs data-dépendantes skippées en CI (aucun générateur
   déterministe auto-suffisant n'existe — `seed-demo` exige les données réelles prod).
   Décider : générateur synthétique committé / fixture versionnée / statu quo skips.
3. **Dette lint gelée** (~479 issues funlen/errcheck) : la CI est verte au ratchet ;
   chantier de résorption optionnel.
4. **Restes hérités d'avant-campagne** (journal deploy 2026-07-10) : V10c (lecture
   budgets sous charge → statuer J4/J6), `populate-assets` absent de l'image prod
   (fallback UI couvre), backup off-site restic à statuer.

### Code / données
5. **Explorer H5 (Q19c)** : « matchs récents cible » lit les colonnes brutes du registre
   (map/mode/playlist NULL pour H5) → porter la cascade de résolution complète.
6. **`legacy_source_used source=duckdb_oauth`** émis pour les 4 joueurs au post-sync du
   2026-07-12 → la Phase 5 ADR 0023 (D2, retrait des fallbacks legacy) n'est PAS armable.
   Migrer la source (RT du store uniquement) avant le 2026-07-17 si on veut tenir la date.
7. **H5 known-set indisponible** en prod : « Can't open a connection to same database
   file with a different configuration » → collecte sans delta (WARN récurrent) — à
   investiguer (config d'ouverture divergente).
8. **H5 achievements** : `RunAchievementsOnly` échoue ×4 joueurs (auto_sync « erreurs
   partielles ») — gap connu du registre parité H5, à prioriser ou accepter.
9. **spartan_cron** : WARN répétés sur les joueurs démo sans player DB (Trimbutton,
   GeleJugefi, …) — bruit à éteindre (skip des profils démo ou provisioning).
10. **Cron leaderboard** : `découverte saison active échouée` (404 saison/playlist)
    toujours présent — « comportement connu » du chantier leaderboard au 2026-07-10,
    à requalifier (attendu ou à corriger).
11. **Détecteur data-quality H5 en LOCAL** : `data_quality_error / internal error`
    (probable schéma H5 shared absent en local) — découverte plan 2.
12. **`GET /admin/monitoring/errors`** UI-orphelin (panneau remplacé) : garder comme
    diagnostic curl-able ou supprimer — micro-décision produit.
13. Mineurs consignés dans les sections Découvertes des plans V7 : glossaire help EN
    sans entrées engagement ; `cmd/engagement-validate` H3 inerte (à rebrancher sur les
    bins) ; clés manifest `engagement.coef.team_share_*` dormantes ; colonne
    `MatchIntensity`/`SaveMatchIntensity` à réévaluer ; `HTTPError` déclaré 3× ;
    flake `TestStartImport_HappyPathReturns202WithJobID` ; pattern `test.skip` inline
    de `p7-dto-rename` fragile.
