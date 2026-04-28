# Reflexion theorique — Indicateur de "forme du joueur" base sur les donnees in-match

> **Statut** : reflexion en cours — pas d'implementation. Cadre conceptuel a valider avant tout code.
> **Date** : 2026-04-28 (revisions iteratives)
> **Contexte** : 6e iteration sur le sujet "detection de forme". Les 5 precedentes (LOWESS, EWMA KD,
> Form Score 14 vs 90, Performance Score relatif, Win Streak) reposent toutes sur des **agregats
> post-match** ET capturent en realite du **skill** plus que de la **forme**. Cette reflexion
> repart d'une distinction conceptuelle stricte entre les deux axes, exploite les timestamps
> intra-match des `highlight_events` pour capter la dimension energie / engagement / etat d'esprit,
> et **normalise par l'activite de l'equipe alliee** (et non du lobby entier) pour annuler l'effet
> du contexte d'equipe (steamroll subi ou inflige).

---

## 1. Distinction fondamentale : skill vs forme

C'est le pivot du concept. Si on confond les deux, on construit un nieme proxy du Performance Score et le signal n'apporte rien.

| Axe | Question repondue | Mesure | Existe deja ? |
|---|---|---|---|
| **Skill** | "A quel point sait-il jouer ?" | Qualite des actions (precision, kill/death efficiency, expected delta) | Oui — `analysis/performance_score.go` |
| **Forme** | "Avec quelle energie joue-t-il ce match-ci ?" | Quantite, rythme, persistance d'engagement | **A construire** |

**Test de bonne separation** : un joueur peut etre dans 4 etats independants :

| Skill | Forme | Lecture |
|---|---|---|
| Eleve | Elevee | Performance brillante (cas attendu) |
| Eleve | Basse | Joueur **competent mais atone** — joue prudent, attend, peu d'actions (fatigue, distraction) |
| Bas | Elevee | Joueur **moins fort mais investi** — multiplie les actions, prend des risques |
| Bas | Basse | Mauvais match complet (mais on sait pourquoi : ni skill ni energie) |

Ces 4 etats doivent etre **separables** par les deux scores. Si la correlation entre FormScore et PerfScore depasse ~0.5 sur un large echantillon, la separation a echoue.

**Implication critique** : les sous-signaux de forme doivent mesurer **la quantite et la regularite de l'action**, pas la qualite du resultat. Un joueur qui meurt 15 fois mais reste actif est plus "en forme" qu'un joueur qui meurt 3 fois en restant cache.

---

## 2. Cadrage utilisateur (consolide)

- Dimensions retenues : **flow intra-match** ET **pattern de session (warmup/burnout)**
- Usage final : viz dual — **courbe** sur Match View (intra-match) + **batons** sur Timeseries/Session (inter-match)
- Composition : **score composite unique 0-100** avec poids fixes
- Reference visuelle : **bande P25-P75 personnelle** (pas une simple ligne mediane), calculee sur historique fixe (200 matchs recents)
- **Normalisation par activite de l'equipe alliee** : retenue pour les signaux pertinents (E1, E4). Le lobby entier n'est pas utilise dans la viz forme car la dynamique lobby/equipes-vs-equipes est deja couverte ailleurs sur Match View (tug of war, cadence).
- **Engagement Coefficients** (deux niveaux) : statistiques personnelles absolues exposees independamment du FormScore — `team_share` et `lobby_share` racontent deux choses differentes (style intra-equipe vs style absolu).

---

## 3. Donnees disponibles, perimetre et limitations

### 3.1 Sources retenues

`shared.highlight_events` (granularite **milliseconde**, ~50-100 evts/match cote joueur, plus pour le lobby entier) :

| event_type | Acteur stocke | Usage forme |
|---|---|---|
| `kill` | KillerXUID | Composante engagement |
| `death` | VictimXUID | Composante engagement + ancre de recovery |
| `assist` | PlayerXUID | Composante engagement |
| `medal` | PlayerXUID | Composante engagement (qualite hors KD) |

Les events sont disponibles **pour tous les joueurs du lobby** dans cette table (filtrage par xuid optionnel). Le calcul forme utilise :
- Events du joueur cible
- Events des **autres joueurs de son equipe alliee** (pour normalisation E1, E4 et la trace de reference de la courbe)
- Events du lobby entier (humains uniquement, bots filtres) **uniquement** pour le calcul du second EngagementCoefficient (`coef_lobby_share`)

### 3.2 Donnees a ne **pas** utiliser

**Stats par minute renvoyees par l'API** (kills_per_minute, deaths_per_minute, etc.) :
- L'API renvoie une valeur au centieme par seconde -> les ratios par minute varient rarement de plusieurs dixiemes entre matchs
- Precision insuffisante pour detecter une variance subtile d'engagement
- **Decision** : tous les ratios temporels sont **recalcules depuis les timestamps bruts** de `highlight_events`, jamais depuis les agregats API

### 3.3 Boundary du match (point en cours de calibration)

L'utilisateur travaille a determiner quand commence **reellement** un match (vs ce que l'API renvoie comme `start_time`, qui peut inclure le countdown ou diverger du moment de jeu reel).

**Implication design** : le module FormScore est **agnostique sur la methode de detection du match start/end**. Il prend en entree :
- `matchStartMS int64` — timestamp de reference du debut de match
- `matchEndMS int64` — timestamp de reference de fin de match
- Une liste d'events pour le joueur cible
- Une liste d'events pour ses coequipiers humains (bots filtres) — pour la trace equipe et E1/E4
- Une liste d'events pour le lobby entier (humains uniquement) — pour `coef_lobby_share` uniquement

Tous les calculs intra-match utilisent un **temps relatif** par rapport a `matchStartMS`. Toute amelioration future de la detection se propage sans modification du module.

**Sous-signal sensible** : E5 Promptitude (time-to-first-event) depend directement de la precision du match start. Tant que la detection est en cours de calibration, son poids dans le composite reste volontairement faible (10 %).

### 3.4 Perimetre et exclusions

**Modes exclus du calcul** :
- **PvE / Firefight** : les bots adversaires ont un comportement regulier qui rend la normalisation ininterpretable. PvE traite a part (ou non couvert v1).
- **Coequipiers bots dans PvP** : les bots sont **filtres du compte equipe et lobby** (on les retire des denominateurs de normalisation et on n'inclut pas leurs events).

**Modes inclus tels quels** :
- **Modes asymetriques** (CTF, Strongholds, Oddball, etc.) : pas de traitement special. Justification utilisateur : tous les joueurs passent par les memes roles a un moment ou un autre, ca s'equilibre statistiquement sur l'historique.

**Modes avec fallback** :
- **FFA / Slayer libre / 1v1** : pas d'equipe -> `pace_team_per_player = pace_joueur` par construction donc le ratio n'a pas de sens. **Fallback** : utiliser `pace_lobby_per_player` (normalisation lobby) pour E1/E4 dans ces modes. Le module marque le mode et applique la regle adequate.

**Limites connues acceptees v1** :
- **Lobby vide post-quitters** : si une grosse fraction du lobby DC, les pace explosent mecaniquement. Cas rare, traite v2.
- **Composition d'equipe variable au fil du match** (3v4 apres quitter) : recalcul dynamique de `N_team` au moment du calcul. Cas marginal en pratique, accepte.
- **Match court (< 3 min)** : trop peu d'events pour stats stables -> FormScore = null.

---

## 4. Architecture du concept : deux niveaux hierarchiques

C'est le coeur du design. Plutot que deux indicateurs separes, on a **un seul signal a deux granularites** :

```
NIVEAU 1 (intra-match)              NIVEAU 2 (inter-match)
================================    ================================
Courbe d'engagement(t)         -->  Score composite par match
- pace equipe alliee per player     (5 sous-scores derives de la courbe)
- pace joueur (superpose)
"Pouls du joueur vs equipe"         "Resume statistique du pouls"
                                    
Affiche sur Match View              Affiche sur Timeseries / Session
```

**Relation formelle** : la courbe est le **signal source**. Les 5 sous-signaux sont des **statistiques** de cette courbe. Le score composite est une moyenne ponderee de ces statistiques apres normalisation percentile vs historique du joueur.

**Avantages de cette hierarchie** :
1. Pas de redondance conceptuelle entre les deux niveaux
2. Drill-down naturel : Timeseries -> baton bas -> click -> Match View -> on voit la courbe et on comprend
3. Validation croisee gratuite : si la courbe raconte un effondrement et le score est haut, c'est un bug

---

## 5. Niveau 1 — La courbe d'engagement intra-match (joueur vs equipe alliee)

### 5.1 Definition

Pour chaque instant `t` du match (echantillonne tous les 5 a 10s), on calcule **deux densites** dans une fenetre glissante de `W = 90s` :

```
pace_team_per_player(t) = events_equipe_alliee_dans_[t - W/2, t + W/2] / (W / 60s) / N_team
pace_joueur(t)          = events_joueur_dans_[t - W/2, t + W/2]        / (W / 60s)
```

`pace_team_per_player` est la **norme dynamique** : ce que chaque coequipier (joueur cible inclus) fait en moyenne dans la fenetre. Le joueur est compare a cette norme, pas au lobby entier.

### 5.2 Pourquoi pas de trace lobby ni trace ennemie

La dynamique lobby (tug of war, cadence globale, pression ennemie) est **deja couverte** ailleurs sur Match View. La courbe forme se concentre sur ce qui lui est unique : le rapport **joueur vs son equipe**. Pas de duplication d'information.

### 5.3 Lecture visuelle

| Configuration visuelle | Interpretation |
|---|---|
| Joueur **au-dessus** de la norme equipe en continu | Il **porte son equipe** — engagement supra-normal pour ce match |
| Joueur **a niveau** de la norme equipe | Il **fait sa part** — engagement normal (50e percentile attendu) |
| Joueur **sous** la norme equipe en continu | Il **decroche** — sous-engagement par rapport a ce que ses coequipiers font dans les memes conditions |
| Joueur qui plonge sous norme a un moment precis | Effondrement ponctuel (tilt, mort qui le sort du jeu) |
| Joueur qui spike au-dessus apres avoir suivi | Reveil / push final |

Avantage cle de la normalisation team : si l'equipe entiere se fait ecraser, **toute la courbe baisse ensemble** (joueur ET equipe). Le joueur n'est pas injustement penalise. S'il s'effondre **plus que son equipe**, ca devient un vrai signal forme.

### 5.4 Annotations a superposer

- **Triangles rouges** sur les deaths precedees d'un creux > 30s (morts passives — caught off-guard, signal forme basse) — **option B retenue : annotation visuelle, pas de sous-score E6 numerique**
- **Disques neutres** sur les autres deaths (morts actives — survenues en plein engagement, signal neutre)

Pas de bande horizontale de reference necessaire : la trace equipe **est** la reference dynamique. Le delta visuel entre les deux courbes raconte la forme.

### 5.5 Ce que la courbe **ne capture pas**

- La qualite des actions (skill — c'est l'objet du PerfScore)
- Le succes des engagements (gagne le duel ou pas)
- L'analyse strategique (positionnement, control de zone)
- Le contexte vs equipe ennemie (-> couvert par tug of war / cadence ailleurs)

---

## 6. Niveau 2 — Score composite par match (5 sous-signaux)

Chaque sous-signal est une **statistique extraite du signal source** (courbe ou events bruts), normalisee en percentile vs historique personnel du joueur sur la meme categorie de mode.

| # | Signal | Formule | Normalise par equipe ? | Direction |
|---|---|---|---|---|
| **E1** | **Densite d'engagement relative** | mediane sur le match de `pace_joueur(t) / pace_team_per_player(t)` | **Oui** | plus haut = mieux |
| **E2** | **Resilience post-mort** | mediane des durees death -> prochain event d'engagement (kill, assist, medaille) du joueur | Non (intrinsequement personnel) | plus bas = mieux |
| **E3** | **Persistance** | coefficient de variation des intervalles inter-events du joueur | Non (regularite personnelle) | plus bas = mieux |
| **E4** | **Tenue de tempo relative** | pente regression de `pace_joueur(t) / pace_team_per_player(t)` entre 1ere et 2e moitie du match | **Oui** | plus haut = mieux (≥ 0 ideal) |
| **E5** | **Promptitude** | time-to-first-event du joueur depuis matchStartMS | Non (signal individuel anchored sur start) | plus bas = mieux |

### 6.1 Pourquoi ces signaux capturent l'energie et pas le skill

- **E1** mesure le **share intra-equipe** du joueur. Si l'equipe entiere est suppressee par les ennemis, tout le monde baisse — le joueur n'est puni que s'il baisse **plus que ses coequipiers**. Independamment du pace absolu (slow Strongholds vs nervous Slayer) ET du contexte de match (steamroll subi ou inflige).
- **E2** est du pur mindset : tilt = retrait long apres mort, en forme = retour rapide. Personnel par construction.
- **E3** capture la regularite. Faible = engagement continu (flow), elevee = bursts entrecoupes de creux longs (decrochage).
- **E4** detecte l'usure intra-equipe : tu maintiens ta part dans l'equipe en 2e mi-temps ou tu ralentis pendant que tes coequipiers compensent ?
- **E5** est la mise en action, pas la mise en reussite. Premier event quel qu'il soit (un assist a 2s suffit).

### 6.2 Normalisation : percentile vs historique personnel

Chaque sous-signal `Ek` est converti en percentile `pk ∈ [0, 100]` calcule sur la distribution de `Ek` dans l'historique du joueur sur la meme categorie de mode (PvP_ranked, PvP_unranked).

- Helper existant : `apps/go-api/internal/analysis/performance_score.go::PercentileRank` et `PercentileRankInverse`
- Categorisation existante : `apps/go-api/internal/analysis/match_history_avg.go`

**50 = exactement comme d'habitude. > 50 = plus engage que d'habitude. < 50 = moins engage.**

### 6.3 Composite final

```
FormScore = w1*p1 + w2*p2 + w3*p3 + w4*p4 + w5*p5
```

| Signal | Poids | Justification |
|---|---|---|
| E1 Densite (relative equipe) | **0.30** | Mesure la plus directe et pure de "fais-tu ta part dans ton equipe" |
| E2 Resilience | **0.25** | Pur mindset, le plus independant du skill |
| E3 Persistance | **0.20** | Marqueur de flow continu |
| E4 Tenue de tempo (relative equipe) | **0.15** | Endurance, signal plus bruite que les autres |
| E5 Promptitude | **0.10** | Signal court, depend de la precision du matchStartMS |

Somme = 1.0 -> FormScore ∈ [0, 100].

### 6.4 Horizon de baseline

**Recommandation** : fenetre adaptative `min(200, all_matches_same_category)` avec seuil minimum **30 matchs**.

- < 30 matchs sur la categorie : FormScore non calcule, retourne `null` (cold start)
- 30-200 matchs : tous utilises pour la baseline percentile
- > 200 matchs : on plafonne aux 200 plus recents (le metagame derive, le joueur change)

**Pourquoi pas en jours** : compteur en matchs invariant par activite (un joueur 1 match/sem et un joueur 50/jour traites pareil).

**Pourquoi pas "match courant exclu"** : la baseline est plus robuste avec plus de donnees ; le match courant ne pollue pas significativement une distribution de 200 points.

### 6.5 EngagementCoefficient — deux statistiques personnelles exposees independamment

La normalisation produit **deux coefficients distincts** a exposer independamment du FormScore. Ils racontent deux choses differentes et leur lecture comparee est riche.

```
coef_team_share(joueur, categorie)  = mediane historique de (pace_joueur / pace_team_per_player)
coef_lobby_share(joueur, categorie) = mediane historique de (pace_joueur / pace_lobby_per_player)
```

| Coefficient | Mesure | Comparable inter-joueurs ? |
|---|---|---|
| `coef_team_share` | **Style intra-equipe** : "quelle proportion du travail d'equipe je prends habituellement" | Oui, c'est un trait personnel pur |
| `coef_lobby_share` | **Style absolu** : "quelle part de l'action totale je prends habituellement" (mixe style + skill + qualite des equipes habituelles) | Oui, mais influence par le contexte |

Lecture comparee — la combinaison des deux est plus riche que chacune isolement :

| Profil | `coef_team` | `coef_lobby` | Interpretation |
|---|---|---|---|
| Leader d'equipes moyennes | 1.3 | 1.0 | **Porte son equipe**, tombe souvent dans des equipes difficiles |
| Passager d'equipes fortes | 1.0 | 1.4 | **Fait sa part** mais beneficie d'equipes dominantes |
| Sous-engage en equipe forte | 0.7 | 1.1 | **Passager** — l'equipe domine sans lui |
| Equilibre | 1.0 | 1.0 | Profil tout-terrain |

A exposer dans un endpoint dedie (ex. `/players/{slug}/engagement_profile`) ou dans le payload Synthesis. Reutilisable hors contexte forme.

---

## 7. Visualisation par vue

### 7.1 Match View — vue intra-match (courbe joueur vs equipe alliee)

**Format** : deux courbes superposees + annotations.

- Axe X : temps relatif au matchStartMS (s ou min selon zoom)
- Axe Y : events/min instantanes (valeurs absolues, lisibles directement)
- **Trace gris/translucide** : `pace_team_per_player(t)` — la norme dynamique de l'equipe alliee
- **Trace coloree au premier plan** : `pace_joueur(t)` — engagement individuel
- **Triangles rouges** sur deaths post-creux > 30s (morts passives)
- **Disques neutres** sur autres deaths
- Pastille en coin : score composite final + breakdown des 5 sous-scores

**Pourquoi cette superposition** : on lit instantanement "tu portes / tu suis / tu decroches" par rapport a tes coequipiers, sans confusion avec la dynamique lobby (deja visible sur tug of war / cadence).

### 7.2 Session — vue intra-session (5 a 15 matchs consecutifs)

**Format** : batons hard-edge.

- 1 baton par match consecutif d'une session
- Hauteur = FormScore par match
- Couleur selon zone : sous P25 personnel = creux, dans bande = normal, au-dessus P75 = en forme
- Bande horizontale P25-P75 du joueur (sur 200 matchs recents) en arriere-plan

**Lecture** :
- 5 batons ↗ qui sortent par le haut = warmup reussi
- 5 batons ↘ qui rentrent dans la zone basse = burnout
- Plateau dans la bande = regime de croisiere

**Pourquoi des batons** : matchs discrets, pas continus. La courbe mentirait sur l'interpolation entre matchs.

### 7.3 Timeseries long — vue tres inter-match (50-500 matchs)

**Format** : option A = batons agreges par session (1 baton = 1 session, hauteur = FormScore moyen). Option B = courbe lissee LOWESS α=0.3 + bande P25-P75 fixe.

**Decision** : option A par defaut (preserve la granularite session, qui est le bon niveau de lecture pour la forme), option B en zoom out extreme.

**Reference** : bande P25-P75 personnelle calculee sur 200 matchs et **fixe**. Une bande glissante derive avec le joueur et masque les progressions reelles.

---

## 8. Architecture cible (haut niveau)

Module a creer : `apps/go-api/internal/analysis/temporal/form_score.go`

### 8.1 Inputs (parametres explicites)

```
type FormScoreInput struct {
    PlayerEvents    []canonical.HighlightEvent  // events filtres pour le xuid cible
    TeamEvents      []canonical.HighlightEvent  // events autres coequipiers humains (bots filtres)
    LobbyEvents     []canonical.HighlightEvent  // events lobby entier (pour coef_lobby_share uniquement)
    NTeam           int                         // nombre de joueurs humains de l'equipe alliee (joueur cible inclus)
    NHumansLobby    int                         // nombre de joueurs humains du lobby entier
    XUID            string
    MatchStartMS    int64                       // boundary externe, agnostique a la methode de detection
    MatchEndMS      int64                       // boundary externe
    History         []HistoricalForm            // pre-calcule : sous-signaux des N matchs precedents pour percentile
    CoefTeamShare   float64                     // coefficient personnel (sur la categorie de mode)
    CoefLobbyShare  float64                     // coefficient personnel (sur la categorie de mode)
    Mode            ModeCategory                // PvP_team / PvP_ffa / PvP_1v1 — pour decider du fallback
}

type HistoricalForm struct {
    MatchID   string
    Subscores [5]float64  // E1..E5 deja calcules
}
```

L'agnostisme sur `matchStartMS` / `matchEndMS` est essentiel : le module forme **n'a pas a savoir** comment le start est detecte. Toute amelioration future se propage sans modification.

Pour les modes FFA/1v1, le module utilise automatiquement `LobbyEvents`/`NHumansLobby` au lieu de `TeamEvents`/`NTeam` pour calculer E1 et E4.

### 8.2 Outputs

```
type FormScoreOutput struct {
    FormScore       float64                  // 0-100
    Subscores       [5]float64               // p1..p5 (percentiles)
    RawSignals      [5]float64               // E1..E5 valeurs brutes (pour debug/narratif)
    EngagementCurve []EngagementPoint        // niveau 1 : pace joueur + pace equipe (pour Match View)
    Confidence      string                   // "full" / "partial" / "insufficient_history"
}

type EngagementPoint struct {
    TimeMS         int64
    PaceJoueur     float64    // events/min instantane joueur (fenetre glissante)
    PaceTeam       float64    // events/min instantane equipe alliee / N_team (norme dynamique)
    PostDeathFlag  bool       // dans une periode post-mort active
    IsPassiveDeath bool       // marqueur sur l'event death : precedee par un creux > 30s
}
```

### 8.3 Reutilisations existantes

- `temporal/lowess.go::LowessSmooth` pour lissage de la courbe (optionnel)
- `analysis/performance_score.go::PercentileRank/Inverse` pour normalisation
- `analysis/match_history_avg.go` pour categorisation modes
- `analysis/sessions.go::ComputeSessions` pour pattern session
- `analysis/temporal/rolling.go::RollingMean` pour fenetre glissante

### 8.4 Pipeline d'ingestion

- Nouveau champ `form_score` dans `player_match_enrichment` (ajout additif, ADR 0005 compatible)
- Nouveau champ ou table dediee pour `coef_team_share` et `coef_lobby_share` par categorie (mediane glissante de 200 matchs, recalculee periodiquement)
- Calcul au sync apres ingestion des `highlight_events` complets (joueur + equipe + lobby)
- La courbe intra-match recalculee a la volee depuis events au moment de la requete Match View (peu couteux, evite cache)

---

## 9. Hypotheses a valider avant implementation

H1. **Le FormScore est-il decorrele du PerfScore ?** Sur 500 matchs, `corr(FormScore, PerfScore)` < 0.5. Si > 0.5, separation echouee — revoir signaux.

H2. **E2 (resilience) a-t-elle assez de signal ?** Necessite ≥ 2-3 deaths/match. Verifier nombre median de deaths par categorie. Si < 3, marquer non-applicable et redistribuer poids.

H3. **E3 (persistance) varie-t-elle assez entre matchs ?** Ratio max/min du E3 sur 100 matchs > 2. Si non, signal inutile.

H4. **E5 (promptitude) reste-t-il fiable malgre l'incertitude sur matchStart ?** A reverifier une fois la detection matchStart stabilisee. Si toujours bruyant, sortir du composite.

H5. **Cold start** : combien de joueurs ont < 30 matchs sur leur categorie principale ? Si majorite, abaisser seuil a 20.

H6. **Normalisation equipe change-t-elle vraiment l'ordre des matchs ?** Comparer FormScore avec normalisation team vs normalisation lobby vs sans normalisation sur 100 matchs. Si correlation entre versions > 0.9 sur tout, simplifier. Si < 0.9, la normalisation team apporte un signal reel — particulierement attendu sur les matchs "steamroll subi" (l'utilisateur ne devrait plus etre injustement penalise).

H7. **EngagementCoefficient (team_share) est-il stable dans le temps ?** Calculer sur 2 fenetres glissantes (matchs 1-100 vs 101-200) et verifier ratio max/min < 1.3. Si stable, le coefficient est une vraie caracteristique de style. Si instable (composition d'equipe trop variable), c'est plutot un signal evolutif.

H8. **Les deux coefficients (team_share et lobby_share) sont-ils suffisamment differents pour justifier d'exposer les deux ?** Si `corr(team_share, lobby_share)` > 0.85 sur l'ensemble des joueurs, garder seulement `team_share`. Si decorrele, garder les deux et exposer leur lecture comparee.

---

## 10. Risques et limites assumees

- **PvE / Firefight exclus** : bots adversaires ininterpretables pour la normalisation. Decision actee.
- **Coequipiers bots PvP** : filtres de l'equipe et du lobby (denominateurs et events).
- **Modes asymetriques** : non traites a part. Justification : tous les joueurs passent par tous les roles dans l'historique, ca s'equilibre. La normalisation team attenue les ecarts mode-specifiques.
- **FFA / 1v1** : fallback automatique sur `lobby_share` pour E1/E4 (pas d'equipe a normaliser).
- **Lobby vide post-quitters** : la part du joueur explose mecaniquement. Cas rare, traite v2.
- **Composition d'equipe variable** (3v4 apres quitter, etc.) : recalcul dynamique de `N_team`, mais cas marginal en pratique. Accepte sans traitement specifique.
- **Match court (< 3 min)** : trop peu d'events -> FormScore = null.
- **Score composite poids fixes** : par construction perd la finesse narrative. Le breakdown des 5 sous-scores attenue (visible Match View).
- **Precision matchStart en cours de calibration** : E5 a poids volontairement faible (10 %).
- **Stats per-minute API insuffisamment precises** : tous ratios temporels recalcules depuis timestamps bruts.
- **Morts passives non comptees comme sous-score numerique** : option B retenue (annotation visuelle uniquement). Si validation ulterieure montre une variance utile et decorrelee de E1+E3, possibilite de promotion en E6.

---

## 11. Verification (theorique, sans implementation)

Avant de committer du code, valider **conceptuellement** sur 5+ cas reels :

1. **Joueur en bonne forme connue** : 5 matchs ou ca tournait -> FormScore > 60 attendu.
2. **Joueur en creux** : 5 matchs ou il n'y arrivait pas -> FormScore < 40 attendu.
3. **Joueur en session bruyante** : 6 matchs warmup puis burnout -> les batons doivent tracer le pattern.
4. **Joueur skill mais atone** : PerfScore eleve, peu d'engagements (passif, defensif) -> FormScore < 50, PerfScore > 70 -> separation OK.
5. **Joueur moyen mais investi** : PerfScore moyen, beaucoup d'engagements (multiplie sans toujours reussir) -> FormScore > 60, PerfScore ~ 50.
6. **Joueur sur equipe steamrollee** (cas critique de la normalisation team) : equipe ennemie ecrase, joueur fait sa part au sein de son equipe affaiblie. FormScore attendu ~ 50 (normal) plutot que < 30 (mauvais). C'est le test cle de la valeur ajoutee de la normalisation team vs lobby.
7. **Joueur sur equipe dominante mais passager** : equipe alliee ecrase, joueur cible profite sans porter. FormScore attendu < 50 (sous-engage par rapport a son equipe en feu) plutot que > 70 (illusion donnee par lobby_share).

Si ces cas (et particulierement 6 et 7) ne fonctionnent pas, l'hypothese centrale est invalidee.

Tests unitaires obligatoires (au moment de coder) :
- 0 events -> FormScore null
- 1 kill 0 death -> E2 null, score sur E1/E3/E4/E5 avec poids redistribues
- Match symetrique (events uniformes joueur ET equipe) -> FormScore ~ 50
- Match "ideal" (engagement dense regulier, recovery rapide, tempo soutenu, part > coef habituel) -> FormScore > 80
- matchStartMS imprecise (decalage 5s) -> E5 affecte mais composite ne bouge que de quelques points
- Mode FFA -> fallback sur lobby_share automatique, FormScore valide
- Equipe entiere a 0 events sauf le joueur -> ratio division par zero protege, fallback valide

---

## 12. Fichiers critiques (references existantes a reutiliser)

- `apps/go-api/internal/games/canonical/match.go:85-122` — type `HighlightEvent`
- `apps/go-api/internal/analysis/performance_score.go` — `PercentileRank`, `PercentileRankInverse`
- `apps/go-api/internal/analysis/temporal/lowess.go` — `LowessSmooth`
- `apps/go-api/internal/analysis/temporal/rolling.go` — `RollingMean`, `RollingMeanAdaptive`
- `apps/go-api/internal/analysis/match_history_avg.go` — categorisation modes
- `apps/go-api/internal/analysis/sessions.go` — `ComputeSessions` pour pattern session
- `apps/go-api/internal/port/highlight_events.go` — interface acces events (filtres ou non par xuid, ou par teamID)
- `apps/go-api/internal/migration/steps_shared.go:118-126` — schema `highlight_events`
- `apps/web/src/components/charts/TimeseriesLineChart.tsx` — composant graphe inter-match (a etendre pour les batons FormScore)
- `.ai/SPEC_ECHARTS_TIMESERIES.md` — blueprint buildOption ECharts (pour la courbe joueur vs equipe Match View)

---

## 13. Ce que cette reflexion **ne couvre pas** volontairement

- Pas d'implementation Go (theorie tant que H1-H8 non validees sur donnees reelles)
- Pas de comparaison inter-joueurs sur le FormScore (intra-personnel par construction). Les EngagementCoefficients sont eux comparables inter-joueurs.
- Pas de coupe par playlist (a l'interieur d'une categorie de mode) — raffinement v2
- Pas de prise en compte du teammate quality (pondération par MMR coequipiers)
- Pas de detection automatique de tilt / fatigue (label categoriel)
- Pas de generation de narratif texte automatique — possible v2
- PvE / Firefight non couverts v1
- Lobby massivement quitte non gere v1
- Trace de la dynamique lobby/ennemie sur la courbe forme : **deja couvert** par tug of war + cadence existants. Pas de duplication.

---

## 14. Questions ouvertes restantes

- **Largeur de la fenetre glissante de la courbe (W)** : 60s, 90s, 120s ? 90s point de depart raisonnable.
- **Echantillonnage de la courbe** : 5s, 10s, 15s ? 10s par defaut.
- **Faut-il inclure les `assist` dans le compte d'events** ? Vote : inclure MVP, retirer si H1 montre bruit.
- **Seuil "creux" pour mort passive** : 30s par defaut, a calibrer par categorie de mode si H8 montre une variance enorme.

---

## Annexe A — Diagnostic des 5 tentatives precedentes

| # | Approche | Statut | Source | Pourquoi insatisfaisant |
|---|---|---|---|---|
| 1 | LOWESS sur perf_score (Squad V2) | Actif | `temporal/lowess.go` | Lisse du **skill**, pas de la forme |
| 2 | EWMA KD (Timeseries) | Partiel | `service/timeseries_service.go` | Signal de skill base sur KD post-match |
| 3 | Form Score 14 vs 90 (Python) | Non porte Go | Legacy | Bonne idee de delta court vs long, mais sur perf agrege = **skill** |
| 4 | Performance Score relatif | Production | `analysis/performance_score.go` | Definition du skill, pas un signal de forme |
| 5 | Win Streak | Anecdotique | `analysis/home.go` | Plus long streak passe — ni skill ni forme actuels |

**Le point aveugle commun** : aucun ne distingue conceptuellement skill et forme. Aucun n'exploite le contexte d'equipe pour normaliser. Le concept FormScore introduit **explicitement** la quantite/regularite/persistance comme axe orthogonal au skill, normalise par l'activite de l'equipe alliee pour annuler l'effet du contexte de match.

---

## Annexe B — Pourquoi les sous-signaux initiaux (S1-S5) ont ete remplaces

Premiere version (avant reorientation skill vs forme + normalisation equipe) :

| Initial | Probleme |
|---|---|
| S1 Cadence (IKI) | Cadence des kills = correle a la qualite = skill, pas forme |
| S2 Vivacite (TTFK) | Vitesse du premier kill = depend du skill |
| S3 Resilience (Death recovery) | Conserve sous le nom E2 (vraiment indicatif d'etat d'esprit) |
| S4 Densite d'exploits (medailles elite) | Marqueurs de performance = skill |
| S5 Symetrie K/D temporelle | Melange skill et engagement |

Replacement par E1-E5 oriente quantite/regularite/persistance, decorrele du skill par construction, **et normalise par l'equipe pour les signaux concernes** (E1, E4).

---

## Annexe C — Pourquoi la normalisation par equipe (et non lobby ni rien) est centrale

Sans normalisation, le pace absolu d'un match pollue le signal forme :
- Match Slayer 12 min equilibre : ~80 events lobby
- Match Slayer 4 min steamroll : ~50 events concentres
- Match Strongholds tactique : longues phases calmes
- Match BTB chaotique : events partout

Avec normalisation **lobby** : on annule le pace absolu, mais on **ne distingue pas** :
- Joueur bien engage dans une equipe ecrasee (qui apparait artificiellement faible vs lobby)
- Joueur passager dans une equipe dominante (qui apparait artificiellement fort vs lobby)

Avec normalisation **equipe** : on isole la **contribution individuelle au sein du collectif**. C'est la mesure pure de "fait-il sa part dans l'equipe qu'il a", independamment du contexte du match.

Demonstration chiffree (4v4, 110 events lobby) :

| Scenario | Equipe alliee | Equipe ennemie | Joueur cible | lobby_share | team_share | Lecture forme correcte |
|---|---|---|---|---|---|---|
| Equipe ecrasee | 30 | 80 | 8 | 0.58 (bas) | **1.07** (normal) | Le joueur fait sa part malgre le steamroll |
| Equilibre | 55 | 55 | 14 | 1.02 | 1.02 | Identique (les deux convergent) |
| Equipe dominante, joueur passager | 80 | 30 | 25 | 1.82 (haut) | **1.25** (modere) | Le joueur beneficie du contexte mais ne porte pas |

`team_share` est structurellement le bon dénominateur pour la **forme individuelle**. `lobby_share` est plus utile comme metric de **profil/style** (dans EngagementCoefficient) car il capture l'effet equipe ET l'effet individuel ensemble — c'est une caracteristique mixte.
