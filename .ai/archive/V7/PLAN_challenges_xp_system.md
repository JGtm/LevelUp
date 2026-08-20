# PLAN — Système de Défis, Arcs Narratifs & XP LevelUp

> Statut : Brouillon — en cours d'affinage
> Inspiré par : fonctionnalité "Objectifs personnalisés" de csstat
> Scope : LevelUp (Halo Infinite) — nouveau système de progression sociale

## Glossaire des termes propres à ce système

| Terme | Abrév. | Définition |
|-------|--------|------------|
| **Points de Prestige** | PP | Monnaie de progression LevelUp — à ne pas confondre avec l'XP Halo Infinite (Season XP, Career XP) ni les Credits (CR) |
| **Prestige** | — | Niveau de progression déduit du total PP. Affiché sur le profil et dans le leaderboard |
| **Arc** | — | Séquence ordonnée de défis incarnant un rôle de joueur (ex. "Le Slayer") |
| **Défi** | — | Objectif mesurable sur une fenêtre temporelle déclarée, avec palier de difficulté automatique |
| **Palier** | — | Difficulté calculée du défi : Normal / Heroic / Legendary / Mythic |
| **Baseline** | — | Moyenne personnelle sur les 20 derniers matchs du même mode — sert à l'anti-smurf et au calcul de palier |
| **Baseline collective** | — | Somme des baselines individuelles des membres d'un défi collectif |
| **Moment card** | — | Image générée à la validation d'un défi, stockée dans la galerie Profil |

> À enrichir au fil des décisions. Tout nouveau terme introduit dans ce plan doit être défini ici.

---

## Contexte & inspiration

csstat implémente un système d'objectifs personnalisés (table `goals`) qui permet à un joueur de se fixer des cibles sur 5 métriques CS2, avec suivi de progression et badge visuel à la validation. La feature est propre mais plate : pas de gamification, pas de récompense symbolique, pas de dimension sociale, pas de narrative.

LevelUp peut implémenter un équivalent Halo — et l'étendre sur trois axes : arcs narratifs, rituel de validation, et un système XP autonome pour la comparaison entre amis.

---

## Axe 1 — Objectifs personnels avec arcs narratifs

### Concept

Un objectif ne flotte plus dans le vide : il appartient à un **arc**, un rôle de joueur que l'utilisateur cherche à incarner. L'arc donne du sens à la progression et transforme la liste de tâches en campagne personnelle.

### Arcs prédéfinis (exemples Halo)

| Arc | Identité | Objectifs enchaînés |
|-----|----------|---------------------|
| Le Slayer | Joueur offensif pur | kills/match > 12, puis > 16, puis > 20 |
| Le Support | Impact non-létal | KAST > 65%, medals "Bodyguard" > 5/mois |
| Le Consistant | Régularité avant tout | win_rate > 55% sur 3 périodes consécutives |
| L'Explorateur | Polyvalence map/mode | jouer 8 maps différentes, puis top 3 sur 4 maps |

### Arcs libres

L'utilisateur peut créer un arc avec un titre narratif personnel, et y attacher ses objectifs dans l'ordre qu'il veut. Quand un objectif tombe, le suivant dans l'arc se déverrouille automatiquement.

### Structure de données envisagée

```
arc
  id, user_id, title, description, is_preset
  -> ordered list of challenges (FK)

challenge
  id, arc_id, user_id
  metric, target, period, label
  position (ordre dans l'arc)
  unlocked_at, completed_at
  is_active (seul le premier non-complété est actif)
```

---

## Axe 2 — Fenêtres temporelles

### Taxonomie recommandée

Un défi porte toujours sur une **fenêtre déclarée à la création**. Trois types exposés au joueur, un quatrième interne uniquement.

| Type | Valeurs disponibles | Cas d'usage | Contrainte |
|------|---------------------|-------------|------------|
| **Session** *(défaut)* | 1, 2 ou 3 sessions | Défis tactiques, engagement immédiat | min. 5 matchs par session pour valider — en dessous, progression affichée mais pas évaluée |
| **Rolling jours** | 7, 14, 30 jours | Joueurs à rythme régulier, objectifs mensuels | dépend de la fréquence de jeu |
| **Deadline libre** | date ISO choisie | Défis d'escouade, objectifs d'arc à moyen terme | min. 2 jours dans le futur |
| *N matchs* *(interne)* | 10, 20 matchs | baseline et anti-smurf uniquement | jamais exposé dans l'UI |

### Pourquoi la session comme défaut

La session est l'unité naturelle de jeu — le joueur pense déjà "ma prochaine session", pas "mes 14 prochains jours". Elle crée une urgence sans être arbitraire, et s'appuie sur l'infrastructure existante (`sessions` table, `session_id` dans `player_match_enrichment`).

"Session unique" = défi court, résultat connu ce soir. "3 sessions" = horizon d'une semaine pour un joueur normal. C'est assez pour la plupart des métriques.

### Cas win_rate

Win_rate est sensible à la variance sur petits échantillons. Règles spécifiques :

- Fenêtre "1 session" : autorisée uniquement si la session fait ≥ 5 matchs, sinon l'évaluation est différée à la session suivante
- Fenêtre "7 jours" : minimum 8 matchs joués sur la période pour que le défi soit évaluable
- Fenêtre "deadline" : pas de minimum strict, mais un avertissement à la création si la baseline du joueur est calculée sur moins de 10 matchs

### Baseline et anti-smurf — fenêtre interne

Indépendamment du type de fenêtre choisi par le joueur, la **baseline personnelle** (pour calculer la difficulté et valider l'anti-smurf) est toujours calculée sur les **20 derniers matchs du même type de mode** en fenêtre glissante. Ce chiffre n'est pas exposé dans l'UI.

### Données insuffisantes — gap fonctionnel accepté

Quand un joueur crée un défi sur un mode où il a peu de matchs, le système dégrade gracieusement plutôt que de bloquer :

| Matchs disponibles sur le mode | Comportement |
|-------------------------------|--------------|
| < 5 matchs | Défi autorisé, palier non calculé, **aucun XP bonus** — tracking pur. Message : *"Joue encore X matchs en [mode] pour débloquer les bonus."* |
| 5 – 9 matchs | Palier calculé, tagué *"estimé"*, bonus XP réduit de moitié |
| ≥ 10 matchs | Comportement normal — palier plein, XP plein |

En escouade, chaque membre est évalué individuellement sur ses propres données. Un membre sans données est traité comme "tracking pur" — il participe et contribue au défi collectif, mais sans bonus XP personnel.

---

## Axe 3 — Rituel de validation

### Ce qui se passe à la validation

À chaque sync, le système compare l'ancienne valeur calculée à la nouvelle. Si un seuil est franchi :

1. **Point de notification** — un indicateur discret apparaît sur l'icône nav de la page Profil (pas de popup, pas d'interruption).
2. **Moment card** — générée et révélée avec une animation d'entrée quand l'utilisateur navigue vers le Profil (galerie "Moments") : stat atteinte, palier de difficulté, date, nombre de matchs joués pour y arriver, nom de l'arc si applicable. Exportable en image.
3. **Journal des accomplissements** — entrée ajoutée dans l'onglet "Historique défis" du Profil avec contexte (fenêtre temporelle, performances sur la période).
4. **Prochain défi** — voir section dédiée ci-dessous.
5. **Contribution PP** — voir Axe 6.

Pas de toast — trop intrusif, interrompt le flux de jeu ou de navigation.

### Placement dans la navigation

Principe : **la home ne reçoit que des surfaces** (widget, teaser). L'expérience complète vit sur des pages dédiées.

| Élément | Emplacement principal | Sur la home ? |
|---------|-----------------------|---------------|
| Point de notification | Icône nav (Profil) | Oui — indicateur nav uniquement |
| Moment card | Page Profil — galerie "Moments" | Non |
| Journal des accomplissements | Page Profil — onglet "Historique défis" | Non |
| Défis actifs & arcs | Page dédiée "Défis & Arcs" | Oui — carousel (voir ci-dessous) |
| Leaderboard amis | Page dédiée "Social / Amis" | Widget top-3 si home non surchargée — à trancher |

### Carousel de défis sur la home

Placé **au-dessus de la section "Faits marquants"**, un carousel de tuiles de défis — même forme visuelle que les tuiles de matchs existantes.

- Switch **Actifs / Terminés** pour basculer entre les défis en cours et les défis validés récents
- Chaque tuile affiche : nom du défi, palier (couleur), progression (barre ou valeur actuelle / cible), fenêtre temporelle restante
- Tuiles "Terminés" affichent le palier atteint et la date de validation
- Scroll horizontal, nombre de tuiles visibles selon la largeur écran (même comportement que les matchs)
- Clic sur une tuile → page "Défis & Arcs" avec focus sur ce défi

### "Prochain défi" — définition

Trois cas selon le contexte du défi validé :

1. **Dans un arc** : le défi suivant de la séquence s'active automatiquement — aucune action requise.
2. **Défi libre complété** : le système propose le palier supérieur de la même métrique (+15 % stretch), que le joueur accepte ou ignore.
3. **Aucun défi actif après validation** : le système pioche dans le **catalogue de templates** (voir ci-dessous) et propose 3 options à des difficultés différentes.

Le catalogue de templates est la pièce centrale : sans lui, la suggestion est vide. Il contient des défis prédéfinis par métrique, mode et palier, filtrés sur ce que le joueur n'a jamais tenté.

---

## Axe 4 — Objectifs d'escouade

### Modèle pool collectif

Un seul membre qui impose un défi crée des frictions. Le bon modèle : un **pool de suggestions** dont chacun choisit ce qui l'intéresse.

**Fonctionnement** :
1. Le système génère un pool de 6 à 9 défis thématiques (renouvelé chaque semaine ou sur demande)
2. N'importe quel membre peut proposer un défi depuis le pool, ou en créer un freeform
3. Les autres membres **rejoignent librement** les défis qui les intéressent
4. Chaque membre qui rejoint **choisit son palier** (Normal / Heroic / Legendary / Mythic) sur le même thème

**Exemple concret** :
```
Défi escouade — "Slayer Session"
  Pierre  → Heroic   : 55 % win rate, 3 sessions
  Karim   → Legendary: 65 % win rate, 3 sessions
  Sofia   → Normal   : 45 % win rate, 2 sessions
```

Le leaderboard interne compare la **progression relative à son propre palier**, pas les valeurs brutes — sinon l'Onyx écrase toujours le Gold.

### Deux modes de participation

**Mode collectif** : les contributions s'additionnent vers un total partagé.
- Exemple : "L'escouade accumule 30 victoires cette semaine" — chaque victoire de n'importe quel membre compte.
- Barre de progression commune, chaque membre voit sa part.

**Mode compétitif** : même thème, chacun court pour son compte.
- Exemple : "Qui atteint en premier son palier sur le défi Slayer ?"
- Classement interne figé à la première validation.

### Baseline en mode compétitif

La baseline anti-smurf reste **individuelle** par design : chaque membre a sa propre baseline, donc son propre palier calculé indépendamment. C'est ce qui justifie le modèle "palier par personne" — Pierre en Heroic et Karim en Legendary utilisent leurs stats respectives, pas une baseline d'équipe.

Pour les membres sans données suffisantes, les règles de dégradation de l'Axe 2 s'appliquent individuellement. Un membre "tracking pur" participe et contribue, sans PP bonus personnel.

### Baseline en mode collectif

Pour un défi collectif cumulatif ("l'escouade accumule 30 victoires cette semaine"), la logique est différente — il n'y a pas de cible individuelle mais un total partagé.

**Baseline collective** = somme des baselines individuelles de tous les participants sur la même fenêtre.

Exemple : 3 membres à 4 victoires/semaine chacun → baseline collective = 12 victoires/semaine. Anti-smurf : cible ≥ baseline_collective × 1.08, sinon rejeté à la création.

**Problème de taille variable** : "30 victoires pour 3 membres" n'a pas le même poids si un 4e rejoint en cours de route. Solution : la cible est stockée en interne comme valeur **par membre** (`target_per_member`), recalculée au total à l'affichage.

```
target_total = target_per_member × nb_participants_actifs
```

Si un membre part, le total cible baisse — la barre est recalculée. Si un membre rejoint, elle monte. Un défi ne peut pas être validé rétroactivement si la cible baisse sous la progression déjà atteinte.

### Prérequis

Le concept d'escouade/liste d'amis n'existe pas encore dans LevelUp — à construire en parallèle.

### Structure de données envisagée

Stockage dans `shared_social.duckdb` — voir Annexe C.

```
challenge_template          ← dans metadata.duckdb (référentiel statique)
  id, metric, window_type, window_value
  label_en, label_fr
  tiers: { normal_target, heroic_target, legendary_target, mythic_target }

squad_challenge             ← dans shared_social.duckdb
  id, template_id (nullable si freeform), created_by
  mode (collective | competitive)
  window_type, window_value, expires_at

squad_challenge_participant ← dans shared_social.duckdb
  squad_challenge_id, user_id
  chosen_tier, data_tier (Normal/estimated/tracking)
  current_value, completed_at
```

---

## Axe 5 — Multiplicateur de performance

### Principe

Le score de performance brut (calculé sur les stats de match) reste **objectif et immuable** — comparable entre tous les joueurs.

Les défis alimentent une couche séparée : un **bonus de défi**, visible distinctement dans l'UI, qui ne pollue pas le score brut.

Affichage décomposé — plus lisible pour la comparaison entre amis :
```
Score brut     1.18
+ Bonus défis  +0.04   [Heroic — Slayer Session validé]
─────────────────────
Score total    1.22
```

Le leaderboard entre amis affiche les trois lignes — chacun voit d'un coup d'œil le niveau brut et l'effort de défi de l'autre.

### Sécurité "défi trop facile"

Un défi n'est éligible au bonus que s'il représente un **effort réel** :

- La cible doit être **supérieure à la baseline personnelle** de la période (moyenne des N dernières périodes pour la même métrique)
- Si le joueur a déjà atteint la cible dans plus de 70% de ses périodes récentes → défi invalide pour le bonus, signalé à la création : *"Tu atteins déjà cette valeur régulièrement — ce défi ne génèrera pas de bonus."*
- Écart minimal suggéré : +10% au-dessus de la baseline pour un bonus plein, entre 0 et +10% → bonus partiel (ex. 50%)

### Règle fondamentale

Le multiplicateur ne joue **qu'à la hausse** : bonus si atteint, zéro sinon. Aucun malus pour un défi manqué — sinon les joueurs n'oseraient pas se fixer des objectifs ambitieux.

---

## Axe 6 — Système Prestige (PP)

### Concept

Un système de **Points de Prestige (PP)** propre à LevelUp, indépendant de l'XP Halo Infinite, qui agrège les actions valorisables au fil du temps. Il sert de base au leaderboard entre amis.

PP ≠ XP Halo — le terme "Prestige" est intentionnellement distinct des systèmes in-game (Season XP, Career XP, Credits).

### Sources de PP

PP des défis selon palier — voir Annexe B.

| Action | PP |
|--------|----|
| Partie jouée | +10 |
| Victoire | +15 |
| Défi personnel — Normal | +50 |
| Défi personnel — Heroic | +75 |
| Défi personnel — Legendary | +125 |
| Défi personnel — Mythic | +200 |
| Défi d'escouade (bonus +20 % sur palier perso) | ×1.2 |
| Arc complété (tous les défis) | +200 |
| Streak : 3 sessions consécutives avec activité | +30 |
| Médaille Heroic+ obtenue | +5 à +20 selon rareté |

### Niveaux Prestige

Système de paliers avec labels narratifs (exemples) :

```
0       → Recrue
500     → Soldat
1 500   → Vétéran
3 000   → Spécialiste
6 000   → Élite
12 000  → Légendaire
```

Les niveaux sont visibles sur le profil et dans le leaderboard.

### Leaderboard entre amis

- Liste des amis classés par PP total (ou PP de la période : semaine / mois)
- Filtre par arc actif : "Qui progresse le plus sur l'arc Slayer ce mois ?"
- Pas de leaderboard global public dans un premier temps — rester dans la sphère privée pour éviter le gaming du système

### Prérequis

- Système de "liste d'amis" ou d'escouade (à construire)
- Table `prestige_events` pour logguer chaque gain avec source et timestamp (auditabilité, évite les bugs de double-comptage)
- Table `user_prestige` ou vue matérialisée sur le total PP par joueur

---

## Questions ouvertes

1. **Baseline : 20 matchs ou N sessions ?** — 20 matchs glissants est propre techniquement, mais "les 5 dernières sessions" est plus naturel pour le joueur. Trancher avant l'implémentation.
2. **Rétroactivité PP** — accorder des PP pour les parties déjà syncées, ou partir de zéro ? Option médiane : rétroactivité limitée aux 90 derniers jours pour donner un démarrage non-vide.
3. **Catalogue de templates** — contenu statique versionné en TOML (comme `fields.toml`) ou table dans `metadata.duckdb` ? TOML est plus simple à maintenir, DB permet des templates communautaires plus tard.
4. **Naming** — risque de confusion avec les Défis hebdomadaires officiels Halo Infinite. Appeler "Missions LevelUp" ou "Objectifs" pour éviter l'ambiguïté avec "Défis Halo".
5. **Escouade** — groupe persistant avec invitation, ou simple liste d'amis ad hoc ? Vérifier si une notion d'amis/contacts existe déjà dans le modèle de données.
6. **Home page** — un seul widget ou deux (défi en cours + top-3 leaderboard) ? À trancher selon l'état réel de la home.

---

## Prochaines étapes (à affiner)

- [ ] Valider les axes prioritaires (tout ? arcs + XP en premier ?)
- [x] Inventorier les métriques Halo disponibles pour les défis — voir Annexe A
- [ ] Définir la structure de données complète (migration DuckDB ou schéma Go API)
- [ ] Maquetter l'UI : page "Défis & Arcs", page "Leaderboard amis", section profil
- [ ] Décider si le système XP est dans le Go API ou dans le frontend

---

## Annexe A — Métriques disponibles pour les défis

> Source : `canonical/fields.go` (40 FieldKey constants), `match_participants` (35 colonnes),
> tables secondaires `medals_earned`, `pve_match_stats`, `match_skill_rank`.
> Toutes les métriques sont référencées par leur **FieldKey canonique** — multi-titre par design.

### Règle générale d'agrégation

Les défis portent toujours sur une **fenêtre glissante**, jamais sur un total carrière (trop facilement sandbaggable). Fenêtre de référence : les **20 derniers matchs du même type de mode** (ou 30 jours calendaires si supérieur).

Les métriques sur des compteurs bruts (kills, damage) sont **normalisées à la minute** en interne — le joueur voit "X kills par match en moyenne" mais le calcul utilise les KPM pour neutraliser les matchs de durées différentes.

---

### Groupe 1 — Universel (tous modes confondus)

Ces métriques sont valides quelle que soit la playlist ou le titre.

| FieldKey | Affichage | Agrégation | Anti-smurf | Plafond |
|----------|-----------|------------|------------|---------|
| `FieldKDA` | KDA moyen | moyenne glissante | ratio : cible > baseline × 1.08 | ~10 (pratique) |
| `FieldKDR` | K/D moyen | moyenne glissante | ratio | ~5 |
| `FieldAccuracy` | Précision (%) | moyenne glissante | delta absolu : cible > baseline + 3 pp | 100 % |
| `FieldDamageDealt` | Dégâts/min | normalisé/min | ratio | aucun |
| `FieldHeadshotKills` | Headshots/match | normalisé/min | ratio | aucun |
| `FieldMeleeKills` | Kills mêlée/match | normalisé/min | ratio | aucun |
| `FieldGrenadeKills` | Kills grenade/match | normalisé/min | ratio | aucun |
| `FieldPowerWeaponKills` | Kills armes lourdes/match | normalisé/min | ratio | aucun |
| `FieldMaxKillingSpree` | Meilleure série | max glissant | ratio | aucun |
| `FieldPersonalScore` | Score perso/min | normalisé/min | ratio | aucun |
| `performance_score` | Score perf. LevelUp | moyenne glissante | delta absolu : cible > baseline + 5 pts | 100 pts |

**Médailles (via `medals_earned`)** : nombre de fois qu'une médaille spécifique est obtenue sur la fenêtre. Seules les médailles "Heroic tier" et au-dessus sont pertinentes (les médailles communes sont trop fréquentes). Exemple : "Obtenir 3× Overkill ce mois-ci".

---

### Groupe 2 — PvP uniquement

| FieldKey | Affichage | Anti-smurf | Note |
|----------|-----------|------------|------|
| `FieldWinRate` | Taux de victoire (%) | delta absolu : +5 pp minimum | 0–100 % |
| `kills_vs_expected` | Kills / kills attendus | **anti-smurf natif** — déjà relatif au MMR | ratio idéal |
| `deaths_vs_expected` (inversé) | Survie relative | anti-smurf natif | ratio inversé |
| `FieldKills` (normalisé) | Kills/min | ratio | sensible au niveau de jeu |
| `FieldAssists` (normalisé) | Assists/min | ratio | |
| `offensive_conversion` | Conversion offensive | ratio | calculé dans `performance_score.go` |
| `defensive_resistance` | Résistance défensive | ratio | calculé dans `performance_score.go` |

**Note sur `kills_vs_expected`** : c'est la métrique la plus honnête du système — elle est déjà normalisée par le MMR prédit. Un KDA de 2.0 n'a pas la même valeur pour un Onyx que pour un Platinum. `kills_vs_expected > 1.5` signifie "50 % de plus que ce que ton niveau devrait produire" — la cible se recalibre automatiquement quand le joueur progresse.

---

### Groupe 3 — Classé uniquement

| FieldKey | Affichage | Anti-smurf | Note |
|----------|-----------|------------|------|
| `FieldCSRValue` | CSR (rang classé) | delta absolu : +50 pts minimum | 0–2000 |
| `FieldLUSRValue` | LUSR (rang LevelUp) | delta absolu : +0.05 minimum | 0–2.0 |
| `rating_delta` positif | Progression CSR/match | ratio | objectif de type "ne pas perdre de SR" |

---

### Groupe 4 — Firefight / PvE uniquement

| FieldKey | Affichage | Anti-smurf | Note |
|----------|-----------|------------|------|
| `FieldWavesCompleted` | Vagues complétées | ratio | très volatile, fenêtre 10 matchs |
| `FieldBossesKilled` | Boss éliminés/session | ratio | |
| `FieldEliteKills` | Elites/match | ratio | |
| `FieldHunterKills` | Hunters/match | ratio | résistants = challenge |
| `total_kills` PvE | Total kills/match | normalisé/min | |

---

### Métriques exclues (et pourquoi)

| Métrique | Raison d'exclusion |
|----------|--------------------|
| `kills` total carrière | Récompense le volume, pas la qualité — sandbaggable |
| `shots_fired` / `shots_hit` brut | Sans contexte, non interprétable |
| `team_mmr` / `enemy_mmr` | Hors contrôle du joueur |
| `kills_expected` etc. (prédictions) | Résultats statistiques, pas actions joueur |
| `time_played_seconds` brut | Récompense la présence, pas la performance |
| `is_with_friends`, `had_bot_teammate` | Contexte social, non performatif |
| `rank_in_match` | Trop volatil, dépend du lobby |

---

## Annexe B — Système de difficulté automatique (4 paliers)

### Principe

Quand un joueur crée un défi, le système calcule automatiquement sa difficulté en combinant :
1. **L'écart personnel** (`stretch_ratio`) : à quel point la cible dépasse sa propre baseline récente
2. **Le plancher population** : où se situe la cible parmi tous les joueurs LevelUp actifs

La difficulté est affichée immédiatement dans le formulaire de création, avant validation.

---

### Les 4 paliers

| Palier | Couleur | Code hex | Signification |
|--------|---------|----------|---------------|
| **Normal** | Gris | `#9CA3AF` | Effort accessible, légèrement au-dessus du niveau actuel |
| **Heroic** | Bleu | `#3B82F6` | Progression significative requise |
| **Legendary** | Violet | `#8B5CF6` | Territoire élite, peu de joueurs y atteignent |
| **Mythic** | Or | `#F59E0B` | Performances exceptionnelles, quasi-plafond personnel |

Références visuelles Halo Infinite : couleurs de rareté cosmétiques (Common/Rare/Epic/Legendary), proches des grades de difficulté campagne.

---

### Formule de calcul

**Étape 1 — stretch_ratio**

Pour les métriques **ratio/score** (KDA, précision, performance_score, win_rate) :
```
stretch = (target - baseline) / max(baseline_headroom, 0.01)
```
où `baseline_headroom = max_value - baseline` (espace restant jusqu'au plafond)

Pour les métriques **count normalisées** (kills/min, damage/min, etc.) :
```
stretch = target / baseline
```

**Étape 2 — palier personnel**

| stretch | Palier personnel |
|---------|-----------------|
| < 1.08 (ou stretch < 0.08 pour les ratios) | Rejeté — trop facile |
| 1.08 – 1.25 | Normal |
| 1.25 – 1.50 | Heroic |
| 1.50 – 1.85 | Legendary |
| > 1.85 | Mythic |

**Étape 3 — plancher population (cap vers le bas)**

La population ne peut que **réduire** la difficulté, jamais l'augmenter :

| Position de la cible dans la population | Cap maximum |
|-----------------------------------------|-------------|
| En dessous de la médiane (p50) | Normal |
| Entre p50 et p75 | Heroic |
| Entre p75 et p90 | Legendary |
| Au-dessus de p90 | Mythic éligible |

**Résultat final** : `min(palier_personnel, cap_population)`

**Exemple** : un débutant qui vise un KDA de 1.5 alors que sa baseline est 0.8 → stretch = 1.87 → palier personnel : Mythic. Mais si KDA 1.5 est sous la médiane des joueurs LevelUp → cap à Normal. Résultat affiché : **Normal**. Le système lui dit : *"Cet objectif représente un grand effort pour toi, mais reste en dessous du niveau médian — difficulté Normal."*

---

### Message de rejet (défi trop facile)

Si stretch < 1.08, le formulaire affiche :
> "Ta moyenne récente est déjà de X sur cette métrique. Relève la cible d'au moins 8 % pour activer un défi."

---

### PP par palier

| Palier | Multiplicateur | PP défi personnel | PP défi escouade |
|--------|:--------------:|:-----------------:|:----------------:|
| Normal | ×1.0 | 50 PP | 60 PP |
| Heroic | ×1.5 | 75 PP | 90 PP |
| Legendary | ×2.5 | 125 PP | 150 PP |
| Mythic | ×4.0 | 200 PP | 240 PP |

Le palier est figé **à la création du défi** — il ne change pas si le joueur progresse entre-temps.

---

## Annexe C — Architecture de stockage

### Principe de répartition

Chaque base a une responsabilité unique. Le système de défis se répartit sur trois bases existantes + une nouvelle.

| Donnée | Base | Justification |
|--------|------|---------------|
| Catalogue de templates | `metadata.duckdb` | Référentiel statique, même pattern que `weapon_labels`, `mode_name_tr` |
| Défis personnels, arcs, XP personnel | `stats.duckdb` (par joueur) | Données privées, isolation par joueur, cohérent avec l'architecture actuelle |
| Défis d'escouade, participants, leaderboard, XP croisé | `shared_social.duckdb` (nouveau) | Cross-joueurs par nature — même pattern que `shared_matches_v2` |
| Moment cards (image générée) | Blob storage | Fichier binaire — chemin référencé dans `stats.duckdb` |

### `shared_social.duckdb` — tables envisagées

```sql
-- Groupes sociaux (escouade, liste d'amis)
squad (id, name, created_by, created_at)
squad_member (squad_id, user_id, joined_at)

-- Défis d'escouade
squad_challenge (id, squad_id, template_id, created_by, mode, window_type,
                 window_value, expires_at, created_at)
squad_challenge_participant (challenge_id, user_id, chosen_tier,
                              data_tier, current_value, completed_at)

-- Prestige et leaderboard cross-joueurs
prestige_events (id, user_id, source_type, source_id, pp_amount, tier,
                 created_at)  -- source_type: 'match'|'challenge'|'arc'|'streak'
user_prestige (user_id, total_pp, current_level,
               updated_at)  -- vue matérialisée ou table mise à jour à chaque event
```

### `stats.duckdb` (par joueur) — tables à ajouter

```sql
arc (id, user_id, title, description, is_preset, created_at)
challenge (id, arc_id, user_id, metric, target, window_type, window_value,
           label, tier, data_tier, position, unlocked_at, completed_at)
moment_card (id, challenge_id, blob_path, created_at)
```

### `metadata.duckdb` — table à ajouter

```sql
challenge_template (id, metric, window_type, window_value,
                    label_en, label_fr,
                    normal_target, heroic_target, legendary_target, mythic_target,
                    mode_filter,  -- 'universal'|'pvp'|'ranked'|'pve'
                    schema_version)
```
