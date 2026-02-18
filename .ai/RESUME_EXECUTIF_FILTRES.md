# Résumé Exécutif : Bugs des filtres sidebar

> **1 page - Pour décision rapide**

---

## 🎯 Problème identifié

Tu as raison : **Le listing dynamique est la cause racine**.

Les playlists/modes/cartes affichés **changent** selon la période/session sélectionnée.

**Conséquence** :
- Options qui apparaissent/disparaissent → Désorientation
- Sélections perdues lors de changements de contexte → Frustration
- Corruption entre joueurs (clés globales non nettoyées) → Bugs

---

## 💡 Solution recommandée

**Options STATIQUES** (toutes les playlists du jeu) avec options **GRISÉES** si pas de matchs.

**Exemple** :

```
☑ Partie rapide (145 matchs)
☑ Arène classée (87 matchs)
☐ Assassin classé (23 matchs)
☐ BTB (i) ← GRISÉ
    "Aucun match dans cette période"
```

**Avantages** :
- ✅ Options stables → Pas de désorientation
- ✅ Préférences conservées → Pas de perte
- ✅ Feedback visuel clair → Meilleure UX

---

## 📊 Effort vs Impact

| Option | Effort | Bugs résolus | UX |
|--------|--------|--------------|-----|
| **A - Fix minimal** | 1-2j | 🟡 50% (corruption) | ❌ Désorientation reste |
| **B - Options statiques** ⭐ | 5-7j | ✅ 100% | ✅ Excellente |
| **C - Refactoring complet** | 2-3 sem | ✅ 100% | ✅ Excellente + maintenabilité |

---

## 🎯 Recommandation

**Option B** (options statiques) - **Meilleur ROI**

Phases :
1. Fix changement de joueur (1-2j) → Résout corruption
2. Options statiques (2-3j) → Résout désorientation
3. Optionnel : Persistance UUID (2j) → Robustesse long terme

**Total** : 5-7 jours pour résoudre complètement le problème.

---

## ❓ Question pour vous

**Quelle option préférez-vous ?**

- [ ] **A** - Fix rapide (1-2j) mais problème de fond reste
- [ ] **B** - Vraie solution (5-7j) 🌟 **RECOMMANDÉ**
- [ ] **C** - Industrialisation (2-3 sem) si projet à long terme

**Besoin de plus d'infos ?** Lire :
- `.ai/ANALYSE_FILTRES_SIDEBAR_2026-02-18.md` (analyse complète)
- `.ai/RECOMMANDATIONS_REDESIGN_FILTRES.md` (guide détaillé)
- `.ai/SYNTHESE_PROBLEMES_FILTRES.md` (diagrammes visuels)

---

**⏸️ En attente de votre décision avant de toucher au code.**
