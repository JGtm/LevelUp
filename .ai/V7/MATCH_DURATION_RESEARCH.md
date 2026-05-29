# Recherche : Estimation de la durée réelle d'un match

> Statut : **Bloqué** sur T₀ — voir § Limitations  
> Dernière mise à jour : 2026-04-06

---

## Contexte

Objectif : reconstruire la durée réelle d'un match Halo Infinite (et les timings par vie) à partir
des données disponibles en DB (`match_participants`, `killer_victim_pairs`, `match_registry`).

Match de référence pour toutes les validations :
- `41b61fb9-3d71-40b7-bde7-45682fba6d57` — Fortress, 2026-03-31, T=447s
- Joueurs full-match : TheTatice, JGtm, Chocoboflor, Madina97294
- XUID JGtm : `2533274823110022`

---

## Structure temporelle d'un match (modèle validé)

```
Début film
    │
    ├── [pré-match variable] ← T₀ inconnu (30s à 60s selon matchs)
    │
T₀  ├── · spawn initial R=8s ·
    │
    ├── █ L₀ █ (vie 0)
    │
    ├── · R · spawn 1 ·
    ├── █ L₁ █ (vie 1)
    │   ...
    ├── · R · spawn D ·
    ├── █ L_final █ (vie D, jusqu'au buzzer)
    │
buzzer (= T₀ + T_réel)
    │
    ├── [post-game ~12s] ← FilmLength - PlayableDuration = 12s
    │
Fin film
```

**Il y a (D+1) spawns** (incluant le spawn initial) et **(D+1) vies** (incluant L_final).

---

## Formules validées

### 1. Estimation de T (durée du match)

$$\boxed{T = (\text{avg\_life} + R) \times (D+1)}$$

- `avg_life` = `AverageLifeDuration.total_seconds()` depuis l'API (`CoreStats`)
- `R = 8.0s` (respawn fixe, inviolable — min(Δ_brut) > 8s vérifié sur tous les joueurs)
- `D` = nombre de morts du joueur

**Précision validée sur Fortress** (avg_api brut) :

| Joueur | D | avg_api | T_est | T_réel | err |
|---|---|---|---|---|---|
| TheTatice | 14 | 21.9s | 448.5s | 447s | +1.5s |
| JGtm | 13 | 23.9s | 446.6s | 447s | −0.4s |
| Chocoboflor | 13 | 24.0s | 448.0s | 447s | +1.0s |
| Madina97294 | 10 | 33.2s | 453.2s | 447s | +6.2s |

**Médiane** sur joueurs full-match : err = **+1.2s** (±2s). Acceptable.

> ⚠️ La colonne `avg_life_seconds` en DB est tronquée au degré 1 (`23.9 → 23.0`).
> Utiliser l'API directement donne +10s de précision par joueur.
> Avec DB tronquée, médiane err = **−5.5s** (biais systématique vers le bas).

### 2. Décomposition des vies (avec T₀ connu)

$$L_0 = \frac{t_0^{\text{film}} - T_0}{1000} - R$$

$$L_i = \frac{t_i^{\text{film}} - t_{i-1}^{\text{film}}}{1000} - R \quad \text{pour } i = 1..D-1$$

$$L_{\text{final}} = T - \frac{t_{D-1}^{\text{film}} - T_0}{1000} - R$$

$$\text{check :} \quad \frac{L_0 + \sum_{i=1}^{D-1} L_i + L_{\text{final}}}{D+1} = \text{avg\_api} \quad \checkmark$$

### 3. Résidu sans T₀ (seule inconnue calculable)

$$L_0 + L_{\text{final}} = \text{avg\_api} \times (D+1) - \sum_{i=1}^{D-1}(\Delta_i^{\text{ms}}/1000 - R)$$

T₀ s'annule algébriquement — ce résidu est calculable sans accès au film mais non décomposable.

---

## Champ `avg_life_seconds` — définition exacte

Source API : `MatchStats.Players[].PlayerTeamStats[].Stats.CoreStats.AverageLifeDuration`  
Format : ISO duration `"PT23.9S"` → `timedelta.total_seconds() = 23.9`

**Ce que l'API calcule :**
$$\text{avg\_life} = \frac{\text{total alive time}}{D+1} = \frac{T - (D+1) \times R}{D+1} = \frac{T}{D+1} - R$$

- Le dénominateur est **D+1** (pas D) — le spawn initial de début de match compte
- La **dernière vie L_final est incluse** dans la moyenne (ce n'est pas une vie censurée)
- Les lives internes (L₀..L_{D-1}) ET L_final entrent dans la moyenne

---

## Impact sur `highlight_events` (countdown non fiable)

`time_ms` dans `highlight_events` est relatif au **début du fichier film** (qui inclut le countdown).
Le graphe "Premier frag / Première mort" (page Teammates) soustrait le countdown estimé :

```
countdown = duration_seconds - playable_duration_seconds
real_start_time = start_time + countdown
```

**Problème** : `PlayableDuration` est non fiable pour les matchs **Ranked Arena** — pour de nombreux
matchs, `PlayableDuration = Duration` → countdown calculé = 0, même lorsqu'un temps de préparation
réel existe. On ne peut pas distinguer "pas de countdown" de "valeur incorrecte de l'API".

Conséquence : les `time_ms` des events restent décalés vers les premières secondes pour ces matchs.

Exploration du dépôt [`dend/grunt`](https://github.com/dend/grunt) (TypeScript, `match-info.ts`) —
confirme que les seuls champs temporels exposés sont `StartTime`, `EndTime`, `Duration` et
`PlayableDuration`. **Aucun `GameplayStartTime`** ni champ similaire. L'API officielle n'expose
rien de plus.

**Correction partielle appliquée** (session 2026-04-02) dans `_teammates_first_events_queries.py`
et `_events_repo.py` — mais la source `PlayableDuration` reste non fiable pour Ranked Arena.

---

## Pistes explorées pour T₀

### ❌ `FilmLength - PlayableDuration`
- Résultat observé : 12s constant sur 6 matchs récents
- **Erreur** : cette différence est le **post-game** (outro), pas le pré-match
- FilmLength API est mesuré depuis T₀, pas depuis le début du film
- `FilmLength - PlayableDuration ≠ T₀`

### ❌ `film_match_start_ms` (champ DB expérimental)
- Précision : ~55% seulement
- **Banni** de toute utilisation en production

### ❌ Tick binaire filmshell (bytes 3-4)
- Tick = compteur 11 bits (~480 Hz), wrappe toutes les 4.27s
- Séquencement local des frames uniquement — pas un temps absolu
- Aucun signal T₀ dans le binaire film

### ❌ Objets MVAR (`objects.json`)
- `Map Intro Camera`, `Team Intro Arrow`, `Spawn Point [Initial]`, `Winning Team Outro`
- Données purement géométriques (X/Y/Z + heading)
- Aucune information temporelle

### ❌ `real_start_time` (champ `match_registry`)
- Égal à `start_time` pour tous les 1527 matchs non-null
- Offset = 0ms pour tous — champ inutile

### ❌ Triangulation multi-joueurs
- T₀ se simplifie algébriquement dans toutes les équations
- Mathématiquement infaisable

### ❌ Téléchargement films binaires
- Films vivent longtemps (pas d'expiration en 2 semaines vérifiée empiriquement)
- Mais ne contient pas de signal T₀ identifiable (voir tick/MVAR ci-dessus)
- 9 chunks × 1532 matchs = ~14k fichiers à dl pour les historiques

### ✅ Seule source fiable : observation externe
- Sur Fortress 2026-03-31 : JGtm L₀≈4s (observation utilisateur)
- → T₀ = first_death_ms − R×1000 − L₀×1000 = 25861ms depuis début film
- Non scalable pour 1532 matchs

### ❓ `FilmLength` vs `Duration * 1000` — non testé
- Si `FilmLength < Duration * 1000` systématiquement pour les matchs Ranked, cela prouverait
  que le film ne couvre pas le countdown → `countdown = Duration - FilmLength/1000`
- Distinct de la piste `FilmLength - PlayableDuration` (= post-game de 12s, déjà invalidée)
- À valider par requête SQL sur `media_files` vs `match_registry.duration_seconds`

### ❓ Discontinuités de position filmshell — piste la plus prometteuse
- Filmshell filtre les grands sauts de coordonnées (`DISCONTINUITY_THRESHOLD = 4000`)
  correspondant aux morts/respawn et au **spawn initial** (transition lobby → carte)
- La **première discontinuité majeure** dans le film = spawn initial du joueur
- Son index de frame ÷ framerate (~60Hz calculé depuis `FilmLength`) = `gameplay_start_ms`
- Implémentation : détecter le marqueur `A0 7B 42`, extraire les deltas coord1/coord2
  (offset +10/+11), identifier le premier grand saut dans les chunks `filmChunkN_dec`
- **Non validé** — à prototyper sur 3-4 matchs de référence avec countdown observé

### ❓ `highlight_events` — borne supérieure via `MIN(time_ms)`
- `MIN(time_ms)` sur tous les événements d'un match (tous types, tous joueurs) est une
  borne supérieure du début réel
- Sanity check : si `countdown_calculé > MIN(time_ms)`, le countdown est surestimé
- Utile pour détecter les faux positifs avant implémentation complète

### ❓ Table de valeurs typiques par mode (fallback)
- Ranked Arena → countdown ~0–5s, BTB/Custom → 10–20s
- Peut servir de fallback quand `PlayableDuration` semble incohérent
- Non calibré — des mesures sur un batch suffisant manquent pour valider

---

## Limitations actuelles

1. **T₀ fondamentalement inaccessible** pour les matchs historiques sans signal identifiable dans les données disponibles.

2. **Troncature `avg_life_seconds`** en DB : le transformer stocke `floor(23.9) = 23.0`, introduisant jusqu'à −13s par joueur. Les appels API bruts donnent la valeur correcte.

3. **Variabilité du pré-match** : 30s à 60s selon le type de match (ranked vs. privé, lobby lent, etc.) — impossible à prédire sans source externe.

4. **KVP incomplet** : seuls 950/1532 matchs ont des killer_victim_pairs. Les vies internes sont inaccessibles pour les autres.

5. **`PlayableDuration` non fiable (Ranked Arena)** : `PlayableDuration = Duration` pour de nombreux matchs Ranked → countdown = 0 à tort. Les `time_ms` dans `highlight_events` sont décalés d'autant pour ces matchs, sans moyen de le corriger sans T₀.

---

## Ce qui fonctionne aujourd'hui

| Besoin | Solution | Précision |
|---|---|---|
| Estimer T (durée match) | `(avg_api_brut + R) × (D+1)` médiane full-match | **±2s** |
| Estimer T (avec DB tronquée) | `(avg_db + R) × (D+1)` médiane full-match | **±8s** (biais −5s) |
| Résidu L₀+L_final | `avg_api×(D+1) − Σ(Δi−R)` | Exact (si avg_api brut) |
| Vies internes L₁..L_{D-1} | `(Δi_ms / 1000) − R` depuis KVP | Exact |

---

## Prochains déblocages possibles

### Priorité 1 — `FilmLength` vs `Duration` (facile, ~1h)
Requête SQL sur les matchs connus : comparer `FilmLength` (si stocké dans `media_files`)
vs `duration_seconds` de `match_registry`. Si absent : télécharger 5-10 films via filmshell
sur des matchs avec countdown connu. Attendu : `FilmLength ≈ Duration - countdown + 12s`.

### Priorité 2 — Premier spawn via discontinuité filmshell (~4h)
POC Python sur les chunks `filmChunkN_dec` déjà téléchargés :
1. Détecter le marqueur `A0 7B 42`
2. Extraire les deltas coord1/coord2 (offset +10/+11)
3. Identifier le premier grand saut (`> DISCONTINUITY_THRESHOLD`) → `countdown_ms`
4. Valider sur 3-4 matchs de référence avec countdown observé manuellement

Si validé : ajouter colonne `gameplay_start_ms` dans `match_registry`
(migration `target_db="shared"`), backfillable pour les films déjà téléchargés.

### Seule vraie solution API
Un champ `match_start_offset_ms` dans l'API Stats Halo — ce champ n'existe pas aujourd'hui.
