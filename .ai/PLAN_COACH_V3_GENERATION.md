# Plan — Coach V3 : extensions de génération

> **Créé le** : 2026-06-09
> **Statut** : Proposé — 3 phases **indépendamment livrables** (ordre suggéré, pas imposé)
> **Priorité** : 🟡 produit (chaque phase = arbitrage produit propre avant branche)
> **Origine backlog** : section `Coach proactif × Prestige` — items Squad coach / Coach négatif soft / Coach tone

## Cadre commun

Tout passe par `internal/progression/coach_advisor/` (génération de signaux/proposals) et son pont
vers `internal/prestige/` (challenges/arcs). État actuel vérifié :
- `coach.GenerateInput` est **strictement per-user** (`coach_advisor/service.go:67-78`).
- LOWESS calcule une pente (`profile/lowess.go`) mais n'émet **que** des signaux positifs.
- Les labels/réasons coach sont **génériques** (pas paramétrés par ton).
- Notifications coach **in-app uniquement** (`player_notifications`).

ADR de référence : 0014 (couches progression), 0020 (pont coach→Prestige), 0021 (synthèse dynamique).

---

## Phase A — Coach négatif soft (neutre / soft-négatif, pas pessimiste)

> Note produit : il s'agit de **neutre à soft-négatif**, jamais culpabilisant. ADR 0014 §6.1 cadre
> aujourd'hui le coach comme strictement positif → cette phase **amende l'ADR** avec garde-fous.

### Détection (backend)
1. Étendre `profile/lowess.go` : exposer la **pente négative soutenue** (déjà calculable —
   `Slope = last - first`), avec seuils de significativité (durée mini, amplitude mini) pour éviter le
   bruit match-à-match.
2. Nouveau `SignalKind` (ex. `LOWESS_SOFT_NEGATIVE`) dans `coach_advisor/signals.go`, **distinct** du
   positif, avec garde-fous : n'émettre que si tendance soutenue ET au-dessus d'un plancher de
   confiance.
3. Mapping vers une **opportunité** (pas un reproche) : signal → proposal de stabilisation
   (`prestige` arc/challenge orienté « consolider X »).

### Narration UI/UX (à concevoir — point explicite demandé)
La narration de ce signal est le **vrai sujet produit**. À cadrer avant implémentation front :
- **Registre** : « opportunité de stabiliser X » / « X mérite ton attention » — jamais « tu régresses ».
- **Placement** : où s'affiche un signal soft-négatif ? (même surface que les signaux positifs ?
  section dédiée « à surveiller » ? badge neutre ?) → maquette à produire.
- **Densité / fréquence** : plafonner (1 signal soft-négatif max par session ? cooldown ?) pour ne pas
  noyer le positif.
- **Couleur/ton visuel** : token **neutre** (pas rouge/alerte) — respecter la règle couleurs `apps/web`
  (token sémantique, pas de hex). Probable `neutral`/`info`, à valider.
- **i18n** : tone guidelines FR + EN qui maintiennent le cadre non-culpabilisant.
- **Opt-in ?** : décision ouverte — activer pour tous (reformulation positive systématique) vs setting
  joueur. À trancher avec la maquette.

**Livrable de cadrage** : une mini-spec UX (placement + wording + tokens + règle de fréquence) **avant**
le commit front. Le backend (détection + signal) peut être livré et testé indépendamment, signal
**non émis vers l'UI** tant que la narration n'est pas validée.

### Tests
- Détection : séries synthétiques (tendance négative soutenue → signal ; bruit → pas de signal).
- Garde-fou : un signal positif et un soft-négatif ne se contredisent pas sur la même métrique.

---

## Phase B — Coach tone (ton narratif **universel**)

> **Décision produit (2026-06-09, validée user)** : le ton n'est **PAS** un setting joueur. Un seul
> ton **par défaut et universel** : soft, non-culpabilisant, factuel-encourageant. Pas de
> `coach_tone` dans `user_preferences`, pas de sélecteur UI, pas de matrice `× tons`. L'ancienne
> proposition (setting `neutral|technical|motivating|playful`) est **abandonnée**.

### Backend
- **Aucun** champ setting. Le ton vit entièrement dans la banque i18n (contenu), pas dans le code.

### Contenu i18n
- Banque de templates i18n **× `SignalKind`** (FR + EN), rédigée selon une **tone guideline** unique
  et documentée (registre soft, jamais « tu régresses »). C'est le gros de l'effort (contenu).
- Cette banque est **la même** que celle de Phase A : Phase B se réduit à la rédaction de la guideline
  + des templates universels. → **Fusion de fait avec Phase A** ; ne reste plus une phase séparée à
  arbitrer.

### Tests
- Résolution template : (kind, lang) → bon template (plus de dimension `tone`).

---

## Phase C — Squad coach (signal niveau escouade)

> La plus structurelle. `SquadChallenge` et `RefreshSquadPool` **existent déjà**
> (`prestige/types.go:246-259`, `service_pilot_pool.go:145-195`) mais **sans filtre coach** (sélection
> aléatoire), et il n'existe **aucun** signal coach niveau escouade.

### Backend
1. **Profil agrégé d'escouade** : moyenne LUSR par axe sur les membres (réutiliser le profil per-user
   existant, agréger).
2. **Signal coach escouade** : variante de `coach.GenerateInput` acceptant un contexte squad (ajouter
   `SquadID`/membres au lieu d'un seul `UserID`). Détecter un pattern collectif (orientation
   combat/objectif/support).
3. **Filtre coach sur le pool** : étendre `RefreshSquadPool` pour accepter un filtre dérivé du signal
   escouade (au lieu du shuffle pur) → proposer un `SquadChallenge` calibré sur la composition.

### Tests
- Profil agrégé déterministe sur fixtures multi-membres.
- `RefreshSquadPool` avec filtre coach ≠ shuffle aléatoire (sélection orientée).

### UX
- Surface squad existante → à étendre (hors périmètre détaillé ici, dépend de la phase produit squad).

---

## Ordre suggéré & estimation
1. **Phase A** (négatif soft) — backend testable vite ; front gaté sur la mini-spec UX.
2. **Phase B** (tone) — mutualisable avec A côté i18n.
3. **Phase C** (squad) — la plus lourde (touche `coach`, `coach_advisor`, `prestige`, front squad).

Chaque phase = **arbitrage produit + branche dédiée**. Ne pas tout ouvrir d'un coup.

## Références
- `internal/progression/coach_advisor/{service.go, signals.go, synthesis_grammar.go}`
- `internal/progression/profile/lowess.go`
- `internal/prestige/{types.go, service_pilot_pool.go}`
- `internal/domain/settings.go` (settings joueur)
- ADR 0014 §6.1 (cadre positif — à amender pour Phase A), ADR 0020, ADR 0021
