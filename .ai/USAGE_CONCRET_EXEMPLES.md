# Usage concret : Mode exclusion en pratique

> **Exemples visuels** pour comprendre comment ça marche au quotidien

---

## 📱 Scénario 1 : Utilisation normale (90% des cas)

### Aujourd'hui (PROBLÈME)

```
┌─────────────────────────────────────────────────────────────────┐
│ JOUR 1 : Tu ouvres l'app, dernière session (3 matchs)          │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Sidebar - Filtres                                              │
│  ┌───────────────────────────────────────────┐                 │
│  │ Période : ⦿ Sessions  ○ Période          │                 │
│  │ Session : Dernière session                │                 │
│  └───────────────────────────────────────────┘                 │
│                                                                 │
│  ┌─ Playlists (3/4) ──────────────────────┐                   │
│  │ ☑ Partie rapide                         │                   │
│  │ ☑ Arène classée                         │                   │
│  │ ☑ Assassin classé                       │                   │
│  │ ☐ Firefight: Gruntpocalypse             │ ← Décoché        │
│  └─────────────────────────────────────────┘                   │
│                                                                 │
│  Tu regardes tes stats : 3 matchs PvP ✅                       │
└─────────────────────────────────────────────────────────────────┘

                         ⬇️ Tu changes de période

┌─────────────────────────────────────────────────────────────────┐
│ JOUR 1 : Tu passes en mode "Période - Ce mois-ci"              │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Sidebar - Filtres                                              │
│  ┌───────────────────────────────────────────┐                 │
│  │ Période : ○ Sessions  ⦿ Période          │                 │
│  │ Du : 01/02/2026  Au : 18/02/2026         │                 │
│  └───────────────────────────────────────────┘                 │
│                                                                 │
│  ┌─ Playlists (3/7) ──────────────────────┐                   │
│  │ ☑ Partie rapide                         │                   │
│  │ ☑ Arène classée                         │                   │
│  │ ☑ Assassin classé                       │                   │
│  │ ☐ BTB                                   │ ← NOUVEAU décoché❌│
│  │ ☐ Action Sack                           │ ← NOUVEAU décoché❌│
│  │ ☐ Firefight: Gruntpocalypse             │ ← Toujours décoché│
│  │ ☐ Firefight: Heroic                     │ ← NOUVEAU décoché❌│
│  └─────────────────────────────────────────┘                   │
│                                                                 │
│  ❌ Problème : BTB, Action Sack décochés alors que tu voulais  │
│     juste exclure Firefight !                                   │
│                                                                 │
│  Tu dois MANUELLEMENT tout recocher... ☹️                       │
└─────────────────────────────────────────────────────────────────┘
```

### Avec la solution (RÉSOLU)

```
┌─────────────────────────────────────────────────────────────────┐
│ JOUR 1 : Tu ouvres l'app, dernière session (3 matchs)          │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Sidebar - Filtres                                              │
│  ┌───────────────────────────────────────────┐                 │
│  │ Période : ⦿ Sessions  ○ Période          │                 │
│  │ Session : Dernière session                │                 │
│  └───────────────────────────────────────────┘                 │
│                                                                 │
│  ┌─ Playlists (3/4) ──────────────────────┐                   │
│  │ ☑ Partie rapide                         │                   │
│  │ ☑ Arène classée                         │                   │
│  │ ☑ Assassin classé                       │                   │
│  │ ☐ Firefight: Gruntpocalypse             │ ← Décoché        │
│  │                                         │                   │
│  │ 💡 Exclusion : Firefight                │ ← Nouveau feedback│
│  └─────────────────────────────────────────┘                   │
│                                                                 │
│  JSON sauvegardé en coulisse :                                  │
│  {                                                              │
│    "playlists_mode": "exclude",                                 │
│    "playlists_selected": ["Firefight: Gruntpocalypse"]          │
│  }                                                              │
│                                                                 │
│  Tu regardes tes stats : 3 matchs PvP ✅                       │
└─────────────────────────────────────────────────────────────────┘

                         ⬇️ Tu changes de période

┌─────────────────────────────────────────────────────────────────┐
│ JOUR 1 : Tu passes en mode "Période - Ce mois-ci"              │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Sidebar - Filtres                                              │
│  ┌───────────────────────────────────────────┐                 │
│  │ Période : ○ Sessions  ⦿ Période          │                 │
│  │ Du : 01/02/2026  Au : 18/02/2026         │                 │
│  └───────────────────────────────────────────┘                 │
│                                                                 │
│  ┌─ Playlists (5/7) ──────────────────────┐                   │
│  │ ☑ Partie rapide                         │                   │
│  │ ☑ Arène classée                         │                   │
│  │ ☑ Assassin classé                       │                   │
│  │ ☑ BTB                                   │ ← AUTO-COCHÉ ✅    │
│  │ ☑ Action Sack                           │ ← AUTO-COCHÉ ✅    │
│  │ ☐ Firefight: Gruntpocalypse             │ ← Toujours exclu  │
│  │ ☐ Firefight: Heroic                     │ ← AUTO-EXCLU ✅    │
│  │                                         │                   │
│  │ 💡 Exclusion : Firefight (2 playlists)  │ ← Feedback mis à jour│
│  └─────────────────────────────────────────┘                   │
│                                                                 │
│  ✅ Solution : Toutes les nouvelles playlists sont auto-cochées│
│     SAUF celles qui contiennent "Firefight"                     │
│                                                                 │
│  Rien à faire, ça marche direct ! 🎉                           │
└─────────────────────────────────────────────────────────────────┘
```

---

## 📱 Scénario 2 : Exclusion ponctuelle (9% des cas)

### Tu veux exclure BTB temporairement

```
┌─────────────────────────────────────────────────────────────────┐
│ Tu veux regarder tes stats SANS BTB (cartes trop grandes)      │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─ Playlists (4/7) ──────────────────────┐                   │
│  │ ☑ Partie rapide                         │                   │
│  │ ☑ Arène classée                         │                   │
│  │ ☑ Assassin classé                       │                   │
│  │ ☐ BTB                                   │ ← Tu décoches     │
│  │ ☑ Action Sack                           │                   │
│  │ ☐ Firefight: Gruntpocalypse             │                   │
│  │ ☐ Firefight: Heroic                     │                   │
│  │                                         │                   │
│  │ 💡 Exclusion : Firefight, BTB           │ ← Mis à jour      │
│  └─────────────────────────────────────────┘                   │
│                                                                 │
│  JSON sauvegardé :                                              │
│  {                                                              │
│    "playlists_mode": "exclude",                                 │
│    "playlists_selected": [                                      │
│      "Firefight: Gruntpocalypse",                               │
│      "Firefight: Heroic",                                       │
│      "BTB"                        ← Ajouté                      │
│    ]                                                            │
│  }                                                              │
│                                                                 │
│  Tu regardes tes stats : sans BTB ✅                           │
└─────────────────────────────────────────────────────────────────┘

                    ⬇️ Lendemain, tu rouvres l'app

┌─────────────────────────────────────────────────────────────────┐
│ JOUR 2 : Tu rouvres l'app                                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─ Playlists (4/7) ──────────────────────┐                   │
│  │ ☑ Partie rapide                         │                   │
│  │ ☑ Arène classée                         │                   │
│  │ ☑ Assassin classé                       │                   │
│  │ ☐ BTB                                   │ ← Toujours exclu  │
│  │ ☑ Action Sack                           │                   │
│  │ ☐ Firefight: Gruntpocalypse             │                   │
│  │ ☐ Firefight: Heroic                     │                   │
│  │                                         │                   │
│  │ 💡 Exclusion : Firefight, BTB           │ ← Préservé        │
│  └─────────────────────────────────────────┘                   │
│                                                                 │
│  ✅ Tes préférences sont conservées !                           │
│                                                                 │
│  Si tu recoches BTB → Il est retiré des exclusions             │
└─────────────────────────────────────────────────────────────────┘
```

---

## 📱 Scénario 3 : Mode PvE uniquement (1% des cas)

### Tu veux regarder UNIQUEMENT tes stats Firefight

```
┌─────────────────────────────────────────────────────────────────┐
│ Tu veux analyser tes performances PvE                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─ Playlists (2/7) ──────────────────────┐                   │
│  │ ☐ Partie rapide                         │ ← Tu décoches tout│
│  │ ☐ Arène classée                         │                   │
│  │ ☐ Assassin classé                       │                   │
│  │ ☐ BTB                                   │                   │
│  │ ☐ Action Sack                           │                   │
│  │ ☑ Firefight: Gruntpocalypse             │ ← Tu coches       │
│  │ ☑ Firefight: Heroic                     │ ← Tu coches       │
│  │                                         │                   │
│  │ 💡 Inclusion : Firefight uniquement     │ ← Mode change auto│
│  └─────────────────────────────────────────┘                   │
│                                                                 │
│  JSON sauvegardé (détection automatique) :                      │
│  {                                                              │
│    "playlists_mode": "include",      ← Mode changé !           │
│    "playlists_selected": [                                      │
│      "Firefight: Gruntpocalypse",                               │
│      "Firefight: Heroic"                                        │
│    ]                                                            │
│  }                                                              │
│                                                                 │
│  Détection auto : < 30% coché = mode "include"                 │
│                   > 70% coché = mode "exclude"                 │
│                                                                 │
│  Tu regardes tes stats : uniquement PvE ✅                      │
└─────────────────────────────────────────────────────────────────┘

                    ⬇️ Lendemain, tu rouvres l'app

┌─────────────────────────────────────────────────────────────────┐
│ JOUR 2 : Tu rouvres l'app (nouvelle playlist BTB Heavies)       │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─ Playlists (2/8) ──────────────────────┐                   │
│  │ ☐ Partie rapide                         │                   │
│  │ ☐ Arène classée                         │                   │
│  │ ☐ Assassin classé                       │                   │
│  │ ☐ BTB                                   │                   │
│  │ ☐ BTB Heavies                           │ ← NOUVEAU décoc ✅ │
│  │ ☐ Action Sack                           │                   │
│  │ ☑ Firefight: Gruntpocalypse             │                   │
│  │ ☑ Firefight: Heroic                     │                   │
│  │                                         │                   │
│  │ 💡 Inclusion : Firefight uniquement     │ ← Préservé        │
│  └─────────────────────────────────────────┘                   │
│                                                                 │
│  ✅ Mode "include" : Nouvelles playlists = AUTO-DÉCOCHÉES      │
│  ✅ Tu vois toujours UNIQUEMENT Firefight                       │
└─────────────────────────────────────────────────────────────────┘
```

---

## 🔄 Comparaison avant/après

### Cas d'usage : Tu changes de période (session → mois complet)

| Aspect | Avant (Problème) | Après (Solution) |
|--------|------------------|------------------|
| **Nouvelles playlists** | Décochées ❌ | Cochées ✅ (sauf Firefight) |
| **Tes exclusions** | Perdues ❌ | Préservées ✅ |
| **Action manuelle** | Recocher tout ☹️ | Rien à faire 🎉 |
| **Sauvegarde** | Valeurs spécifiques | Règle d'exclusion |
| **JSON** | `["A", "B", "C"]` | `exclude: ["Firefight"]` |

---

## 💡 Points clés à retenir

### 1. Détection automatique du mode

Tu n'as **rien à configurer**. Le système détecte automatiquement :

```
> 70% des playlists cochées → Mode "exclude"
  Tu veux "tout sauf quelques-unes"
  
< 30% des playlists cochées → Mode "include"
  Tu veux "seulement quelques-unes"
```

### 2. Feedback visuel discret

Un petit indicateur te montre ce qui est actif :

```
💡 Exclusion : Firefight          ← Mode normal
💡 Exclusion : Firefight, BTB     ← Exclusion multiple
💡 Inclusion : Firefight uniquement ← Mode rare
```

### 3. Pas de changement d'interface

L'interface reste **exactement la même** :
- ✅ Même checkboxes
- ✅ Même boutons "Tout" / "Aucun"
- ✅ Même expanders

La seule différence : **ça marche comme tu l'attends** ! 🎉

---

## 🎬 Animation du workflow complet

### Workflow typique sur 3 jours

```
JOUR 1 - Matin
┌────────────────────────────────────┐
│ Tu ouvres l'app                    │
│ → Dernière session                 │
│ → Firefight décoché (normal)       │
│ → Stats PvP : 5 matchs             │
└────────────────────────────────────┘

JOUR 1 - Après-midi
┌────────────────────────────────────┐
│ Tu veux voir ce mois-ci            │
│ → Change pour "Période"            │
│ → Tout est coché sauf Firefight ✅ │
│ → Stats PvP : 87 matchs            │
└────────────────────────────────────┘

JOUR 2 - Matin
┌────────────────────────────────────┐
│ Tu rouvres l'app                   │
│ → Dernière session (auto)          │
│ → Firefight toujours décoché ✅    │
│ → Stats PvP : 3 matchs             │
└────────────────────────────────────┘

JOUR 2 - Après-midi
┌────────────────────────────────────┐
│ Nouvelle playlist "BTB Heavies"    │
│ disponible                         │
│ → Tu changes pour "Ce mois"        │
│ → BTB Heavies AUTO-COCHÉ ✅        │
│ → Firefight toujours exclu ✅      │
│ → Stats PvP : 93 matchs            │
└────────────────────────────────────┘

JOUR 3 - Matin
┌────────────────────────────────────┐
│ Tu veux analyser PvE               │
│ → Décoches tout sauf Firefight     │
│ → Mode "include" activé auto       │
│ → Stats PvE : 15 matchs            │
└────────────────────────────────────┘
```

**Constat** : Tu n'as **jamais eu à recocher manuellement** des playlists ! 🎉

---

## 🚀 Pour toi concrètement

### Ce qui change pour toi

**Avant** (frustrant) :
1. Tu changes de période
2. 😤 "Mince, BTB et Action Sack sont décochés !"
3. Tu recoches manuellement
4. Lendemain, rebelote...

**Après** (magique) :
1. Tu changes de période
2. 😊 "Parfait, tout est coché sauf Firefight !"
3. Tu regardes tes stats direct
4. Lendemain, toujours bon !

### Ce qui ne change PAS

- ✅ Interface identique
- ✅ Même manipulations
- ✅ Même boutons
- ✅ Pas de nouvelle complexité

**Tu utilises l'app exactement pareil, mais ça marche mieux !**

---

## ❓ Questions pratiques

### "Et si je veux changer d'avis ?"

Tu recoches/décoches comme d'habitude. Le système s'adapte.

### "Comment je sais quel mode est actif ?"

Regarde le petit indicateur `💡 Exclusion/Inclusion : ...`

### "Ça marche entre les joueurs ?"

Oui ! Chaque joueur a ses propres préférences sauvegardées.

### "Et mes anciens filtres ?"

Ils sont automatiquement convertis en mode "include" (comportement actuel).

### "Je peux désactiver ça ?"

Oui, en décochant tout puis recochant ce que tu veux → Mode "include" manuel.

---

## 🎯 En résumé

**Un seul mot** : **Intelligent**

Le système comprend maintenant ton **intention** :
- "Je veux tout sauf Firefight" 
- Pas "Je veux ces 5 playlists spécifiques"

**Résultat** : Plus de frustration, plus de manipulation manuelle ! 🎉
