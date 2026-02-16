# Guide Rapide - Navigation Documentation LevelUp

> **Mis à jour** : 2026-02-16
> **Objectif** : Savoir quel document utiliser selon votre besoin

---

## 🎯 Question : Quel document lire ?

### Je veux...

#### ...commencer un nouveau sprint/phase

**→ Lire** : `.ai/PLAN_CONSOLIDE_FINAL.md`
- Vue d'ensemble complète
- Ordre chronologique optimal
- Dépendances critiques
- Évite duplications

#### ...implémenter Sprint 5, 6, 7, ou 8 (Features UI)

**→ Lire** : `.ai/PLAN_UNIFIE.md`
- Détails complets par sprint
- Tâches, tests, gates de livraison
- Consulter aussi `.ai/SPRINT_EXPLORATION.md` si sprint ≥6

#### ...implémenter Phase 5 (Shared DB partie 1)

**→ Lire** : `.ai/PHASES_5_10_ANALYSES.md`
- Analyse approfondie teammates_service.py
- Analyse approfondie _match_queries.py
- Analyse approfondie _roster_loader.py
- Code AVANT/APRÈS complet
- ~35 pages détaillées

#### ...implémenter Phases 6, 7, ou 8 (Shared DB partie 2 + Filtres + Modules)

**→ Lire** : `.ai/PHASES_6_10_COMPLETE.md`
- Phase 6 : 8 méthodes duckdb_repo + UI critiques
- Phase 7 : 3 bugs filtres identifiés avec solutions
- Phase 8 : 11 modules, 3 critiques avec fallbacks
- Code AVANT/APRÈS complet
- ~35 pages détaillées

#### ...implémenter Sprint 9 (Cleanup Pandas + src/db/)

**→ Lire dans l'ordre** :
1. `.ai/PLAN_UNIFIE.md` Sprint 9 (structure générale)
2. `.ai/PHASES_6_10_COMPLETE.md` Phases 7-8 (bugs identifiés à intégrer)
3. `.ai/SPRINT_EXPLORATION.md` (audit 35 fichiers Pandas)

#### ...implémenter Phase 9 (Cleanup Brutal Tables)

**→ Lire** : `.ai/PHASES_6_10_COMPLETE.md` Phase 9
- 8 tables à supprimer
- 9 tables à conserver
- Script validation Python
- Procédure backup→cleanup→validation

#### ...implémenter Sprint 11 + Phase 10 (Documentation)

**→ Lire** :
1. `.ai/PLAN_UNIFIE.md` Sprint 11 (tests + docs Sprint)
2. `.ai/PHASES_6_10_COMPLETE.md` Phase 10 (13 docs architecture)
3. Liste consolidée dans PLAN_CONSOLIDE_FINAL.md

#### ...chercher une fonction/fichier spécifique

**→ Utiliser** : `.ai/PHASES_5_10_INDEX.md`
- Index rapide par phase
- Index par fonction (teammates_service, etc.)
- Index par problème (bugs Phase 7, modules critiques Phase 8)
- Liens directs vers analyses

#### ...suivre ma progression

**→ Utiliser** : `.ai/MIGRATION_V5_FINAL_CHECKLIST.md`
- 163 items à cocher
- Métriques progression
- Notes de session
- Roadmap visuelle

#### ...comprendre l'architecture V5 finale

**→ Lire** : `.ai/MIGRATION_V5_FINAL_GUIDE.md`
- Vue d'ensemble migration complète
- Architecture AVANT/APRÈS
- Phases 0-10 résumées
- Points critiques

#### ...référence technique détaillée (bugs, inventaire)

**→ Lire** : `BUGFIX_V5_2026-02-15.md`
- Document source (164KB)
- Bugs critiques corrigés (§1-5)
- Inventaire exhaustif 31 fichiers
- Tests de complétude (§16)
- **Note** : Gros document, consulter ponctuellement

---

## 📚 Structure Complète de Documentation

```
📁 Documentation LevelUp
│
├─ 🎯 PLAN_CONSOLIDE_FINAL.md          [CE FICHIER - COMMENCER ICI]
│   └─ Plan unifié évitant duplications
│
├─ 📘 PLAN_UNIFIE.md                   [Features + Cleanup]
│   ├─ 12 sprints (S0-S12)
│   ├─ S0-S4 ✅ livrés
│   └─ S5-S12 📋 à faire
│
├─ 🏗️ Migration V5 Finale              [Architecture Shared DB]
│   ├─ MIGRATION_V5_FINAL_GUIDE.md     [Vue d'ensemble]
│   ├─ PHASES_5_10_ANALYSES.md         [Phase 5 détaillée]
│   ├─ PHASES_6_10_COMPLETE.md         [Phases 6-10 détaillées]
│   ├─ PHASES_5_10_INDEX.md            [Index navigation]
│   └─ MIGRATION_V5_FINAL_CHECKLIST.md [Suivi progression]
│
├─ 🔍 Exploration & Référence
│   ├─ SPRINT_EXPLORATION.md           [Audit codebase]
│   ├─ BUGFIX_V5_2026-02-15.md         [Référence technique]
│   └─ project_map.md                  [Cartographie]
│
└─ 📖 Guides Annexes
    ├─ README_MIGRATION_V5.md          [Index général migration]
    ├─ START_HERE.md                   [Démarrage rapide]
    └─ thought_log.md                  [Journal décisions]
```

---

## 🗓️ Workflow Recommandé

### Démarrage Nouvelle Étape

```
1. Lire PLAN_CONSOLIDE_FINAL.md
   ↓ [Identifier étape courante]
   
2. Suivre document(s) indiqué(s)
   ↓ [Lire détails techniques]
   
3. Cocher items dans CHECKLIST
   ↓ [Suivre progression]
   
4. Référence ponctuelle BUGFIX si besoin
   └─ [Détails très spécifiques]
```

### Pendant Implémentation

```
Code éditeur (VS Code)
   ↓
Document analyse (PHASES_*.md) en parallèle
   ↓
Copier code AVANT/APRÈS
   ↓
Tests
```

---

## 🎓 Par Rôle

### Développeur Features (S5-S8)

**Documents principaux** :
1. PLAN_CONSOLIDE_FINAL.md (ordre global)
2. PLAN_UNIFIE.md (détails sprint)
3. SPRINT_EXPLORATION.md (données disponibles)

### Développeur Architecture (Phase 5-10)

**Documents principaux** :
1. PLAN_CONSOLIDE_FINAL.md (ordre global)
2. MIGRATION_V5_FINAL_GUIDE.md (vue d'ensemble)
3. PHASES_5_10_ANALYSES.md (Phase 5)
4. PHASES_6_10_COMPLETE.md (Phases 6-10)
5. PHASES_5_10_INDEX.md (navigation rapide)

### Reviewer/QA

**Documents principaux** :
1. PLAN_CONSOLIDE_FINAL.md (comprendre ordre)
2. MIGRATION_V5_FINAL_CHECKLIST.md (vérifier items)
3. PLAN_UNIFIE.md Sprint 11 (tests finaux)

### Chef de Projet

**Documents principaux** :
1. PLAN_CONSOLIDE_FINAL.md (vue 30 jours)
2. MIGRATION_V5_FINAL_CHECKLIST.md (progression)
3. Métriques de succès (dans PLAN_CONSOLIDE_FINAL.md)

---

## 📊 Comparaison Documents

| Document | Pages | Focus | Utilisation |
|----------|-------|-------|-------------|
| **PLAN_CONSOLIDE_FINAL** | 15 | Unification | Quotidienne |
| PLAN_UNIFIE | 77 | Features | Par sprint |
| MIGRATION_V5_FINAL_GUIDE | 25 | Architecture | Vue ensemble |
| PHASES_5_10_ANALYSES | 35 | Phase 5 code | Implémentation Ph5 |
| PHASES_6_10_COMPLETE | 35 | Phases 6-10 code | Implémentation Ph6-10 |
| PHASES_5_10_INDEX | 6 | Navigation | Recherche rapide |
| CHECKLIST | 12 | Suivi | Quotidienne |
| BUGFIX | 164 | Référence | Ponctuelle |

---

## ⚠️ Pièges à Éviter

### ❌ NE PAS :

1. **Lire BUGFIX en entier avant de commencer**
   → Trop dense, utiliser comme référence

2. **Ignorer PLAN_CONSOLIDE_FINAL**
   → Risque de duplication ou mauvais ordre

3. **Faire Phase 5-8 avant S5-S8**
   → Les features cassent si architecture change avant

4. **Faire Phase 9 (cleanup brutal) avant S9 (Pandas migration)**
   → Code Pandas reste, tests échouent

5. **Lancer sans backup (Phase 9)**
   → Perte données possible

### ✅ À FAIRE :

1. **Toujours commencer par PLAN_CONSOLIDE_FINAL**
   → Comprendre où vous êtes dans le plan global

2. **Utiliser PHASES_5_10_INDEX pour recherche rapide**
   → Trouver infos en <30s

3. **Cocher CHECKLIST quotidiennement**
   → Progression visible

4. **Consulter BUGFIX ponctuellement**
   → Détails techniques spécifiques

5. **Backup AVANT Phase 9**
   → Sécurité données

---

## 🚀 Actions Immédiates

### Si vous démarrez maintenant :

```bash
# 1. Lire plan consolidé
cat .ai/PLAN_CONSOLIDE_FINAL.md

# 2. Vérifier état actuel
git log --oneline | grep -E "Sprint|Phase" | head -10

# 3. Identifier prochaine étape
# → Si S5-S8 non fait : PLAN_UNIFIE.md Sprint X
# → Si S5-S8 fait : PHASES_5_10_ANALYSES.md Phase 5

# 4. Cocher checklist
code .ai/MIGRATION_V5_FINAL_CHECKLIST.md
```

---

## 📞 Aide Rapide

**Question** : "Je ne sais pas par où commencer"
→ **Réponse** : Ouvrir `.ai/PLAN_CONSOLIDE_FINAL.md` section "Prochaine Action Immédiate"

**Question** : "Je cherche une fonction spécifique"
→ **Réponse** : Ctrl+F dans `.ai/PHASES_5_10_INDEX.md`

**Question** : "Je veux voir les bugs identifiés"
→ **Réponse** : `.ai/PHASES_6_10_COMPLETE.md` Phase 7 (3 bugs) et Phase 8 (3 modules)

**Question** : "Quelle est ma progression ?"
→ **Réponse** : `.ai/MIGRATION_V5_FINAL_CHECKLIST.md` section "Métriques"

**Question** : "Dois-je lire BUGFIX ?"
→ **Réponse** : Non en entier. Consulter ponctuellement si besoin détail technique.

---

**Bon courage avec la migration ! 🎯**
