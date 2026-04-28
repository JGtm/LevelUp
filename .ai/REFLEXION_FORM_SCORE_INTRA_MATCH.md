# Reflexion theorique — Indicateur de "forme du joueur" base sur les donnees in-match

> **Statut** : reflexion en cours — pas d'implementation. Cadre conceptuel a valider avant tout code.
> **Date** : 2026-04-28 (revisions iteratives — refonte majeure : passage du composite a 5 sous-signaux vers un **residu unique** sur 3 courbes)
> **Contexte** : 6e iteration sur le sujet "detection de forme". Les 5 precedentes (LOWESS, EWMA KD,
> Form Score 14 vs 90, Performance Score relatif, Win Streak) reposent toutes sur des **agregats
> post-match** ET capturent en realite du **skill** plus que de la **forme**. Cette reflexion
> repart d'une distinction stricte entre les deux axes, exploite les timestamps intra-match des
> `highlight_events`, normalise par l'activite de l'equipe alliee, et **mesure la forme comme
> l'ecart entre l'engagement observe et l'engagement attendu** (en fonction du contexte du match
> et du style historique du joueur).

---

## 1. Distinction fondamentale : skill vs forme

C'est le pivot du concept. Si on confond les deux, on construit un nieme proxy du Performance Score et le signal n'apporte rien.

| Axe | Question repondue | Mesure | Existe deja ? |
|---|---|---|---|
| **Skill** | "A quel point sait-il jouer ?" | Qualite des actions (precision, kill/death efficiency, expected delta) | Oui — `analysis/performance_score.go` |
| **Forme** | "Avec quelle energie joue-t-il ce match-ci ?" | Ecart entre engagement observe et engagement attendu | **A construire** |

**Test de bonne separation** : un joueur peut etre dans 4 etats independants (skill x forme). Si la correlation entre FormScore et PerfScore depasse ~0.5 sur un large echantillon, la separation a echoue.

**Implication critique** : la forme mesure la **quantite et la regularite de l'engagement**, pas la qualite des actions. Un joueur qui meurt 15 fois mais reste actif est plus "en forme" qu'un joueur qui meurt 3 fois en restant cache.

---

## 2. Concept central : la forme comme **residu** de l'engagement attendu

C'est le coeur du design refactor.

### 2.1 Idee directrice

A chaque instant `t` du match, on peut calculer ce que le joueur **aurait du faire** vu :
1. Le **contexte du match** : activite globale, position de son equipe vs ennemie
2. Son **style historique** : sa part habituelle dans le travail collectif

L'**engagement attendu** est cette prediction. La **forme** est l'ecart entre l'observe et l'attendu, integre sur le match.

```
forme(t)        = pace_joueur(t) - pace_attendu(t)
FormScore_match = percentile(mean_t forme(t), distribution_historique_du_joueur)
```

Sortie : un seul nombre 0-100 par match, **directement lisible** :
- 50 = ecart moyen comme d'habitude (forme normale pour ce joueur)
- > 50 = ecart moyen plus positif que d'habitude (en forme)
- < 50 = ecart moyen plus negatif que d'habitude (sous-forme)

### 2.2 Calcul de l'engagement attendu (version MVP)

```
pace_attendu_joueur(t) = coef_team_share * pace_team_per_player(t)
```

Ou `coef_team_share` est la part historique du joueur dans le travail de ses equipes (mediane glissante 200 matchs sur la meme categorie de mode).

Exemple :
- Joueur historiquement a `coef_team_share = 1.12` (prend 28 % du travail dans un 4v4 ou la fair share serait 25 %)
- A `t = 5min`, son equipe alliee fait 12 events/min/joueur
- Pace attendu : 1.12 × 12 = **13.4 events/min**
- S'il fait 16 : au-dessus de l'attendu, en forme ce moment-la
- S'il fait 9 : en-dessous, sous-forme ce moment-la

La courbe attendue **suit dynamiquement la courbe equipe**, mais a son ratio personnel. Quand l'equipe est en feu, son attendu monte. Quand l'equipe subit, son attendu baisse. Le joueur n'est juge que sur son **ecart a ce qui etait raisonnable de lui dans ce contexte**.

### 2.3 Pourquoi cette approche est meilleure que le composite a 5 sous-signaux

| Composite a 5 sous-signaux (abandonne) | Residu unique (retenu) |
|---|---|
| Poids arbitraires (pourquoi 0.30 / 0.25 / 0.20 / 0.15 / 0.10 ?) | Aucun poids — un seul percentile |
| Normalisations heterogenes (certaines vs equipe, d'autres personnelles) | Une seule normalisation : ecart vs son ecart historique |
| Diagnostic decompose que l'utilisateur ne consulte jamais en pratique | Diagnostic visuel direct (3 courbes) |
| Bricole : addition d'idees disparates | Concept unifie : "ecart a l'attendu" |

Les 5 dimensions du composite restent **visibles dans la courbe** sans etre numerisees (cf §6).

---

## 3. Architecture : un signal, quatre echelles d'affichage

Le residu (joueur − attendu) est le **signal source unique**. Il s'affiche differemment selon l'echelle :

```
ECHELLE              FORMAT                                 LECTURE
=================    ==================================     ============================
Intra-match          3 courbes superposees                  "A cet instant, j'etais
(Match View)         (equipe / attendu / joueur)             au-dessus / en-dessous
                                                             de mon attendu"

Match (carte)        1 nombre = percentile du residu mean   "Sur ce match, FormScore 62"

Session              Batons signes par match                Pattern warmup / plateau /
(5-15 matchs)        hauteur = delta brut                   burnout immediatement
                                                             lisible

Timeseries long      Batons agreges par session             Trajectoire long terme
(50-500 matchs)      ou courbe lissee                       
```

Semantique partout identique : **ecart a l'attendu**. Aucune rupture conceptuelle entre les echelles.

---

## 4. Donnees disponibles, perimetre et limitations

### 4.1 Sources retenues

`shared.highlight_events` (granularite **milliseconde**, ~50-100 evts/match cote joueur, plus pour l'equipe) :

| event_type | Acteur stocke | Usage |
|---|---|---|
| `kill` | KillerXUID | Compte engagement |
| `death` | VictimXUID | Compte engagement + ancre annotation post-mort |
| `assist` | PlayerXUID | Compte engagement |
| `medal` | PlayerXUID | Compte engagement |

Le calcul forme utilise :
- Events du joueur cible
- Events de ses coequipiers humains (pour `pace_team_per_player`)
- Events du lobby entier (pour `coef_lobby_share` exposable independamment, cf §7.2)

### 4.2 Score d'objectif comme events virtuels (modes asymetriques)

Pour ne pas sous-estimer l'engagement dans les modes objectif (CTF, Strongholds, Oddball — ou un porteur de drapeau s'engage sans tuer), on enrichit le compte d'events avec un proxy derive du `personal_score` :

```
events_objectif_estimes = (personal_score - 100*kills - 50*assists) / poids_unitaire_objectif
```

Avec `poids_unitaire_objectif` calibre par mode (~ 25 points par capture). Ces events virtuels sont **ajoutes au total brut** pour le calcul de `pace_joueur` et `pace_team` (sans timestamp, donc repartis uniformement sur la duree du match).

Limites :
- C'est un agregat post-match (pas exploitable pour la courbe instantanee — repartition uniforme)
- Mode-dependant (Slayer pur = ~0 points objectif, donc pas d'apport)

### 4.3 Donnees a ne **pas** utiliser

**Stats par minute renvoyees par l'API** (kills_per_minute, etc.) :
- Centieme par seconde -> rarement varient de plusieurs dixiemes
- Precision insuffisante
- **Decision** : tous les ratios temporels recalcules depuis timestamps bruts, jamais depuis agregats API

### 4.4 Boundary du match (point en cours de calibration)

Le module FormScore est **agnostique sur la methode de detection du match start/end**. Il prend `matchStartMS` et `matchEndMS` comme inputs. Toute amelioration future de la detection se propage sans modification du module.

### 4.5 Perimetre et exclusions

**Modes exclus du calcul** :
- **PvE / Firefight** : bots adversaires ininterpretables. Non couvert v1.
- **Coequipiers bots dans PvP** : filtres du compte equipe (denominateur et events).

**Modes inclus tels quels** :
- **Modes asymetriques** : pas de traitement special. Score objectif (cf 4.2) compense les portions sans combat. Tous les joueurs passent par tous les roles dans l'historique.

**Modes avec fallback** :
- **FFA / 1v1** : pas d'equipe -> fallback sur lobby_share comme reference. Le module detecte le mode et applique la regle.

**Limites connues acceptees v1** :
- **Lobby vide post-quitters** : cas rare, traite v2.
- **Composition d'equipe variable au fil du match** (3v4 apres quitter) : recalcul dynamique de `N_team`. Marginal en pratique.
- **Match court (< 3 min)** : trop peu d'events -> FormScore = null.

---

## 5. Niveau intra-match — la courbe a 3 traces

### 5.1 Les 3 traces

Pour chaque instant `t` du match (echantillonne tous les 10s), on calcule en fenetre glissante de `W = 90s` :

```
pace_team_per_player(t) = events_equipe_alliee_dans_window / (W/60s) / N_team
pace_joueur(t)          = events_joueur_dans_window / (W/60s)
pace_attendu_joueur(t)  = coef_team_share * pace_team_per_player(t)
```

- **Trace gris clair** : `pace_team_per_player(t)` — pouls collectif de l'equipe alliee
- **Trace gris fonce pointille** : `pace_attendu_joueur(t)` — ce que le joueur aurait du faire vu son style et son equipe
- **Trace coloree pleine** : `pace_joueur(t)` — engagement effectif

### 5.2 Lecture visuelle

| Configuration | Interpretation |
|---|---|
| Joueur **au-dessus de l'attendu** en continu | En forme — porte plus que d'habitude |
| Joueur **a niveau de l'attendu** | Forme normale pour ce joueur dans ce contexte |
| Joueur **sous l'attendu** en continu | Sous-forme — fait moins que ce que ses habitudes laissaient prevoir |
| Joueur qui plonge sous l'attendu apres une mort | Mort qui le sort du jeu (signal de tilt visible) |
| Joueur qui spike au-dessus en fin | Push final ou come-back |

Annotations superposees :
- **Triangles rouges** sur les deaths post-creux > 30s (morts passives — caught off-guard)
- **Disques neutres** sur autres deaths
- **Bandes rouges translucides** sur les periodes post-mort jusqu'au prochain event d'engagement

### 5.3 Ce que la courbe **ne capture pas**

- Qualite des actions (skill — c'est l'objet du PerfScore)
- Succes des engagements (gagne le duel ou pas)
- Analyse strategique (positionnement, control de zone)
- Dynamique lobby vs equipe ennemie (deja couvert par tug of war + cadence ailleurs sur Match View)

### 5.4 Pourquoi pas de trace lobby ni trace ennemie

La dynamique lobby (tug of war, cadence globale, pression ennemie) est **deja couverte** ailleurs sur Match View. La courbe forme se concentre sur ce qui lui est unique : le rapport **joueur vs son attendu**. Pas de duplication.

---

## 6. Le residu unique — calcul du FormScore

### 6.1 Formule

```
FormScore_brut = mean_t (pace_joueur(t) - pace_attendu_joueur(t))     // residu moyen sur le match
FormScore_0_100 = percentile(FormScore_brut, distribution_historique_du_joueur)
```

`distribution_historique_du_joueur` = les FormScore_brut de ses 200 derniers matchs sur la meme categorie de mode.

50 = ecart moyen comme d'habitude. > 50 = au-dessus de son habituel. < 50 = en-dessous.

### 6.2 Horizon de baseline

**Recommandation** : fenetre adaptative `min(200, all_matches_same_category)` avec seuil minimum **30 matchs**.

- < 30 matchs sur la categorie : FormScore non calcule, retourne `null` (cold start)
- 30-200 matchs : tous utilises pour la baseline
- > 200 matchs : on plafonne aux 200 plus recents (le metagame derive)

**Pourquoi en matchs et non en jours** : compteur invariant par activite (un joueur 1 match/sem et un joueur 50/jour traites pareil).

### 6.3 Les 5 dimensions du composite abandonne restent **visibles** sur la courbe

On ne perd pas le diagnostic, il devient visuel :

| Ancienne dimension (abandonnee) | Lecture visuelle equivalente |
|---|---|
| E1 Densite | Ecart vertical moyen joueur vs attendu |
| E2 Resilience post-mort | Largeur des bandes rouges post-death |
| E3 Persistance / regularite | Lissage / variance visuelle de la courbe joueur |
| E4 Tenue de tempo | Pente du gap (joueur − attendu) entre 1ere et 2e mi-temps |
| E5 Promptitude | Position temporelle du premier point non-nul de la courbe joueur |

Pas besoin de pastilles decomposees. La courbe raconte tout. Pour un narratif texte automatique (v2), on pourra extraire ces dimensions a la lecture (ex. "Tu as bien commence puis decroche en 2e mi-temps").

---

## 7. Les statistiques personnelles connexes (exposees independamment)

Le calcul du FormScore produit en sous-produit deux statistiques personnelles utiles ailleurs.

### 7.1 EngagementCoefficient — `coef_team_share`

```
coef_team_share = mediane historique de (pace_joueur / pace_team_per_player)
                 sur les 200 matchs recents de la categorie de mode
```

Lecture : "ce joueur prend habituellement X % du travail dans ses equipes". Caracterise le style intra-equipe (leader / equilibre / passager).

C'est aussi le **multiplicateur** utilise dans le calcul du pace_attendu (cf §2.2).

### 7.2 EngagementCoefficient — `coef_lobby_share`

```
coef_lobby_share = mediane historique de (pace_joueur / pace_lobby_per_player)
                  sur les 200 matchs recents de la categorie de mode
```

Lecture : "ce joueur prend habituellement X % de l'action totale du lobby". Caracterise le style absolu (mixe style + skill + qualite des equipes habituelles).

Comparaison des deux — la lecture combinee est plus riche que chacune isolement :

| Profil | `coef_team` | `coef_lobby` | Lecture |
|---|---|---|---|
| Leader d'equipes moyennes | 1.3 | 1.0 | **Porte son equipe**, tombe souvent dans des equipes difficiles |
| Passager d'equipes fortes | 1.0 | 1.4 | **Fait sa part** mais beneficie d'equipes dominantes |
| Sous-engage en equipe forte | 0.7 | 1.1 | **Passager** — l'equipe domine sans lui |
| Equilibre | 1.0 | 1.0 | Profil tout-terrain |

Exposee dans `/players/{slug}/engagement_profile` ou Synthesis.

### 7.3 MatchIntensity — caracteristique du match

```
match_intensity = pace_lobby_total_per_player_per_min sur la duree complete du match
```

Caracteristique objective du match, independante du joueur. Permet :
- Affichage Match View : badge "Intensite du match : 14.2 events/min — chaotique (P88 vs ton historique)"
- Filtre Timeseries : "afficher seulement ma forme sur les matchs > P75 d'intensite"
- Coloration / dimensionnement des batons Timeseries selon intensite (raffinement viz)
- Profil joueur (v2) : "ce joueur est plus engage dans les matchs intenses que calmes"

Stockee au niveau match, ne change jamais apres ingestion.

---

## 8. Visualisation par echelle

### 8.1 Match View — courbe a 3 traces (VALIDE)

**Mock 10 retenu** pour l'onglet equipe de Match View (cf §5 + exigences visuelles §8.6). Mock 10 = Mock 1 (3 traces equipe / attendu / joueur) **avec auto-zoom Y et hierarchie visuelle marquee**.

Format :
- 3 courbes superposees, echantillonnees toutes les 10s sur la duree du match
- **Auto-zoom Y** : range calcule dynamiquement depuis les donnees du match avec padding ±1, label de l'axe affiche le range
- **Hierarchie visuelle** : equipe trace fine effacee (1.5 px, gris fonce), attendu trace fine pointillee (1.5 px, gris medium), joueur trace epaisse saturee (4 px, couleur d'accent)

Variantes considerees mais non retenues :
- `mock_1` : version originale a Y fixe et hierarchie homogene — risque de chart "muet" en match equilibre, remplace par Mock 10
- `mock_2` : residu signe seul (joueur − attendu en une trace) — minimaliste mais perd le contexte des paces absolus
- `mock_3` : hybride avec gap shading + annotations deaths — gap shading rejete (pas l'esthetique recherchee)

### 8.2 Carte synthese (vignette match)

Affichage compact d'un FormScore 0-100 + un mini-sparkline du residu :
- Pastille couleur (vert / orange / rouge selon zone P25-P75 personnelle)
- Texte "FormScore 62 — au-dessus de votre habitude"
- Optionnel : sparkline 30 px de hauteur du residu

Voir mockup `mock_4`.

### 8.3 Session / Periode — courbe a 3 traces (VALIDE)

**Mock 11 retenu** pour la vue session/periode. Mock 11 = Mock 8 (3 traces, 1 point/match) **avec auto-zoom Y et hierarchie visuelle marquee**, exactement comme Mock 10 a l'echelle intra-match.

Format :
- 3 traces, granularite "1 point = 1 match" au lieu de "1 point = 10s"
- **Auto-zoom Y** : range calcule dynamiquement avec padding ±1
- **Hierarchie visuelle** identique a Mock 10 (equipe effacee, attendu pointille, joueur saturee epaisse)
- Marqueurs ronds visibles sur chaque match pour rappeler la discretisation
- Segments lineaires entre points (pas de smoothing : matchs discrets)

L'oeil ne change pas de langage entre les deux echelles : meme grammaire visuelle Match View ↔ Session, juste un zoom temporel different.

Variantes considerees mais non retenues :
- `mock_8` : version originale a Y fixe — remplace par Mock 11 pour les memes raisons que Mock 1 → Mock 10
- `mock_5` : batons positifs colores (option B) — initialement leading, abandonnee pour preserver la consistance visuelle avec Match View
- `mock_6` : batons signes centres sur zero (option A) — alternative diagnostic detaille
- `mock_9` : 2-lignes (lignes continues + batons FormScore alignes) — variante densitee si besoin futur

### 8.4 Timeseries long — batons agreges par session

Pour 50-500 matchs. Voir mockup `mock_7`.

- 1 baton = 1 session, hauteur = mean FormScore de la session
- Largeur du baton proportionnelle au nombre de matchs de la session
- Coloration : vert > P75, orange P25-P75, rouge < P25
- Bande P25-P75 personnelle fixe en arriere-plan

Optionnellement transposable au format Mock 8 (3 traces agregees par session) si l'horizon reste raisonnable et qu'on prefere preserver la consistance visuelle integrale.

### 8.5 Recap des choix retenus

| Echelle | Mock retenu | Format |
|---|---|---|
| **Match View** (intra-match, single-player) | **Mock 10** | 3 traces continues, 1 point = 10s, auto-zoom Y + hierarchie visuelle |
| **Session / Periode** (single-player) | **Mock 11** | 3 traces discretes, 1 point = 1 match, auto-zoom Y + hierarchie visuelle |
| **Session / Periode squad** (2-4 joueurs) | **Mock 12** | N traces de residus signes autour de 0, auto-zoom Y, comparaison parallele |
| Carte synthese | Mock 4 | Vignette compacte avec FormScore + sparkline |
| Timeseries long | Mock 7 | Batons agreges par session (largeur variable) |

Cle de coherence : Mock 10 et Mock 11 partagent la meme grammaire visuelle (3 traces equipe / attendu / joueur, hierarchie visuelle identique, auto-zoom). L'utilisateur garde le meme reflexe de lecture entre les echelles intra-match et session, juste un zoom temporel different.

### 8.6 Exigences visuelles obligatoires (Mock 10 + Mock 11)

Au moment de l'implementation, ces deux exigences sont **non-negociables** pour eviter le risque de "chart muet" en match equilibre / joueur dans son style habituel (cas commun ~70 % des matchs) :

**1. Auto-zoom Y axis**
- Calculer dynamiquement `Yrange = [floor(min(data)) - 1, ceil(max(data)) + 1]` a partir de toutes les valeurs des 3 traces du match (ou de la session)
- Pas de `min: 0, max: 20` fixe
- Afficher le range courant dans le label de l'axe Y (ex. `events / min (auto-zoom 8-15)`) pour rappeler l'echelle a l'utilisateur
- Trade-off accepte : on ne peut plus comparer visuellement deux matchs entre eux dans Match View. Ce n'est pas l'usage principal de cette vue (l'utilisateur est dans le contexte d'un match a la fois). Pour la comparaison inter-match, c'est la vue Session/Timeseries qui sert.

**2. Hierarchie visuelle marquee**
- Equipe : trace fine 1.5 px, couleur tres effacee (`#444c56` ou equivalent gris fonce qui se fond dans le fond)
- Attendu : trace fine 1.5 px, pointillee, gris medium (`#8b949e`)
- Joueur : trace epaisse 3.5-4 px, couleur d'accent saturee (`#58a6ff` ou couleur retenue par le design system)
- Effet : le joueur "pop" toujours, meme si son tracage est colle aux deux autres. L'oeil capture instantanement "voici ma trace" sans confusion avec l'equipe ou l'attendu.

Ces deux exigences combinees permettent de rendre les variations subtiles de 0.5-1 events/min lisibles et de preserver l'expressivite du chart meme dans les matchs peu spectaculaires.

**Rejetes explicitement** (decisions actees) :
- Pas de gap shading (remplissage colore entre joueur et attendu) — esthetique non retenue
- Pas de pastille FormScore en coin de chart — non retenue

La pastille FormScore sera disponible dans la **carte synthese du match** (Mock 4) qui est un autre composant, pas dans le chart lui-meme.

### 8.7 Extension squad — vue session/periode pour 2 a 4 joueurs (VALIDE)

**Mock 12 retenu** pour la vue **session/periode squad**. Rupture grammaticale assumee vs Mock 11 single-player parce que le besoin cognitif est different : ici on veut **comparer en parallele** les coequipiers, pas lire individuellement.

Format :
- 1 chart unique, axe Y centre sur 0 avec auto-zoom
- N traces de **residus signes** : `joueur_i − attendu_i` par match, pour chaque membre du squad
- Ligne 0 = "exactement a l'attendu personnel" (chaque joueur compare a son propre attendu, pas un attendu commun)
- Joueur cible en bleu epais 4px (continuite avec Mock 10/11), coequipiers en couleurs distinctes 2.5px
- Auto-zoom Y avec affichage du range dans le label

Pourquoi cette rupture est legitime :
- Sur Mock 10/11, le besoin = "diagnostic de mon match/ma session" (pace absolu + attendu utiles)
- Sur Mock 12, le besoin = "qui dans le squad porte/decroche" (comparaison parallele indispensable)
- Les paces absolus mentent en comparaison inter-joueurs : un coef 1.3 a un attendu plus haut qu'un coef 0.8. Comparer leurs paces brut melange style et forme.
- Seuls les **residus** sont comparables entre joueurs (chacun mesure contre sa propre baseline)

Lecture cle :
- **Synchronisation** : les N traces bougent ensemble vers le haut/bas → squad-flow ou burnout collectif
- **Divergence** : 1 trace en haut pendant que les autres baissent → solo-carry
- **Incoherence** : traces dispersees autour de 0 → squad pas aligne ce match-la

Variantes considerees mais non retenues :
- **Small multiples** (4 mini Mock 11s empiles) : preserve la grammaire single-player mais lecture sequentielle au lieu de parallele → rate l'objectif de comparaison entre coequipiers
- **Multi-trace single chart avec equipe + attendus + joueurs** : 9 lignes pour 4 joueurs → illisible
- **Heatmap session × joueur** : tres compact mais lecture categorielle (pas de magnitude) → en complement v2 si squad > 4 joueurs

Reutilisation existante :
- `internal/service/squad_service_v2.go` charge deja les events par `MatchIDs` partages — meme dataset
- Chaque joueur a son `coef_team_share` deja calcule individuellement (load batch des N coefs)
- Le `pace_attendu_i` se calcule pour chaque joueur i avec son coef propre

Pour la vue **Match View squad** (intra-match) : a designer ulterieurement, mais probablement small multiples (Mock 10 mini × N) parce qu'a l'echelle intra-match l'usage est plutot diagnostic individuel par joueur, pas comparatif. Donc grammaire Mock 10 preservee pertinente.

---

## 9. Architecture cible (haut niveau)

Module a creer : `apps/go-api/internal/analysis/temporal/form_score.go`

### 9.1 Inputs

```
type FormScoreInput struct {
    PlayerEvents    []canonical.HighlightEvent  // events filtres pour le xuid cible
    TeamEvents      []canonical.HighlightEvent  // events autres coequipiers humains (bots filtres)
    LobbyEvents     []canonical.HighlightEvent  // events lobby entier (humains)
    NTeam           int                         // taille equipe alliee humains (joueur cible inclus)
    NHumansLobby    int                         // taille lobby humains
    XUID            string
    MatchStartMS    int64                       // boundary externe (agnostique a la methode)
    MatchEndMS      int64                       // boundary externe
    History         []HistoricalFormBrut        // residus bruts des 200 matchs precedents pour percentile
    CoefTeamShare   float64                     // coefficient personnel (categorie de mode)
    CoefLobbyShare  float64                     // expose ailleurs, pas utilise dans calcul forme
    PersonalScore   int                         // pour calcul events_objectif_estimes (cf 4.2)
    Mode            ModeCategory                // PvP_team / PvP_ffa / PvP_1v1
}

type HistoricalFormBrut struct {
    MatchID  string
    Brut     float64  // FormScore_brut de ce match
}
```

### 9.2 Outputs

```
type FormScoreOutput struct {
    FormScore        float64                  // 0-100 (percentile)
    ResidualBrut     float64                  // valeur brute (mean joueur - attendu)
    EngagementCurve  []EngagementPoint        // les 3 traces pour Match View
    MatchIntensity   float64                  // events/min/joueur du lobby (caracteristique match)
    Confidence       string                   // "full" / "partial" / "insufficient_history"
}

type EngagementPoint struct {
    TimeMS         int64
    PaceJoueur     float64
    PaceTeam       float64    // pace_team_per_player
    PaceAttendu    float64    // coef_team_share * pace_team_per_player
    PostDeathFlag  bool       // dans une zone post-mort
    IsPassiveDeath bool       // marquee si death precedee par creux > 30s
}
```

### 9.3 Reutilisations existantes

- `temporal/lowess.go::LowessSmooth` pour lissage optionnel
- `analysis/performance_score.go::PercentileRank` pour normalisation
- `analysis/match_history_avg.go` pour categorisation modes
- `analysis/sessions.go::ComputeSessions` pour pattern session
- `analysis/temporal/rolling.go::RollingMean` pour fenetre glissante

### 9.4 Stockage hybride

| Donnee | Strategie | Justif |
|---|---|---|
| FormScore 0-100 + ResidualBrut par match | **Stocke** dans `player_match_enrichment` | Calcul lourd (percentile sur 200 matchs), relu frequemment |
| Courbe intra-match (`EngagementCurve`) | **Live** depuis `highlight_events` | Leger (50-100 events), non re-utilise hors Match View |
| `coef_team_share`, `coef_lobby_share` | **Stocke** par categorie | Mediane glissante 200 matchs, recalcule periodiquement |
| `match_intensity` | **Stocke** au niveau match | Caracteristique permanente du match |

---

## 10. Hypotheses a valider avant implementation

H1. **Le FormScore est-il decorrele du PerfScore ?** Sur 500 matchs, `corr(FormScore, PerfScore)` < 0.5. Si > 0.5, separation echouee.

H2. **Le modele "attendu = coef × team" predit-il bien dans la pratique ?** Sur 100 matchs, calculer R² entre `pace_attendu` et `pace_joueur` reel. Si R² < 0.3, le modele MVP est trop simple — passer aux baselines conditionnelles par intensite (cf §13).

H3. **Le coefficient `coef_team_share` est-il stable dans le temps ?** Calculer sur 2 fenetres glissantes (matchs 1-100 vs 101-200) et verifier ratio max/min < 1.3.

H4. **Cold start** : combien de joueurs ont < 30 matchs sur leur categorie principale ? Si majorite, abaisser seuil a 20.

H5. **Test critique de la separation skill / forme** : sur des matchs ou le joueur a haute perf et bas engagement (ou inverse), le FormScore et PerfScore divergent-ils comme attendu ?

H6. **Match intensity comme contexte** : est-ce que la repartition `coef_team_share` varie significativement selon match_intensity (P33 / P66 buckets) ? Si oui, justifie l'evolution v2 vers baselines conditionnelles.

H7. **Score d'objectif (cf 4.2)** : la formule de proxy events_objectif est-elle stable par mode ? Calibrer le poids unitaire sur sample multi-modes.

---

## 11. Risques et limites assumees

- **PvE / Firefight exclus** : bots adversaires ininterpretables. Non couvert v1.
- **Coequipiers bots PvP** : filtres de l'equipe et lobby.
- **Modes asymetriques** : pas de traitement special. Score objectif compense partiellement.
- **FFA / 1v1** : fallback automatique sur lobby_share.
- **Lobby vide post-quitters** : cas rare, traite v2.
- **Composition d'equipe variable** : recalcul dynamique `N_team`, marginal en pratique.
- **Match court (< 3 min)** : FormScore = null.
- **Modele MVP `attendu = coef × team` est lineaire et statique** : si H2 montre R² insuffisant, evolution vers regression contextuelle (intensity, score gap, etc.) ou conditional baselines.
- **Precision matchStart en cours de calibration** : la courbe est decalee si imprecise, mais le residu mean est robuste car `pace_joueur` et `pace_attendu` sont decales pareil.
- **Stats per-minute API insuffisamment precises** : tous ratios recalcules depuis timestamps bruts.
- **Morts passives non comptees comme sous-score numerique** : annotation visuelle uniquement.
- **Score d'objectif sans timestamp** : reparti uniformement, ne contribue pas a la forme intra-match (seulement au compte total).

---

## 12. Verification (theorique, sans implementation)

Avant de committer du code, valider **conceptuellement** sur 7 cas :

1. **Joueur en bonne forme connue** : 5 matchs ou ca tournait -> FormScore > 60 attendu.
2. **Joueur en creux** : 5 matchs ou il n'y arrivait pas -> FormScore < 40 attendu.
3. **Joueur en session bruyante** : warmup puis burnout -> les batons doivent tracer le pattern.
4. **Joueur skill mais atone** : PerfScore eleve, peu d'engagements -> FormScore < 50, PerfScore > 70.
5. **Joueur moyen mais investi** : PerfScore moyen, beaucoup d'engagements -> FormScore > 60, PerfScore ~ 50.
6. **Joueur sur equipe steamrollee** mais qui fait sa part interne : FormScore ~ 50 (normal pour son contexte).
7. **Joueur sur equipe dominante mais passager** : FormScore < 50 (sous-engage par rapport a son attendu eleve).

Tests unitaires obligatoires (au moment de coder) :
- 0 events -> FormScore null
- Match symetrique (events uniformes joueur ET equipe) avec joueur = attendu -> FormScore ~ 50
- Joueur qui surperforme constamment l'attendu -> FormScore > 80
- matchStartMS imprecise (decalage 5s) -> FormScore reste robuste (decalage absorbe par les deux paces)
- Mode FFA -> fallback sur lobby_share automatique
- Equipe entiere a 0 events sauf le joueur -> ratio division par zero protege
- `coef_team_share = 0` (joueur structurellement passif) -> attendu = 0 partout, residu = pace_joueur

---

## 13. Evolution v2 envisagee

Si H2 ou H6 montrent que le modele MVP est insuffisant, evolution naturelle vers **baselines conditionnelles par intensite** :

```
historique_low_intensity  = matchs avec intensity < P33 du joueur
historique_med_intensity  = matchs entre P33 et P66
historique_high_intensity = matchs > P66
```

Chaque match juge contre **son bucket d'intensite**. Le `coef_team_share` lui-meme peut etre conditionnel (3 valeurs au lieu de 1).

Avantages :
- Capture le fait que le joueur peut avoir des reactions differentes selon l'intensite
- Toujours principielle (pas de poids arbitraire)

Cout :
- Triple le besoin en historique : 30 × 3 = 90 matchs minimum pour activation complete
- Cold start plus long
- Implementation plus complexe (3 distributions a maintenir)

A activer **uniquement si valide par H2/H6**.

---

## 14. Fichiers critiques (references existantes)

- `apps/go-api/internal/games/canonical/match.go:85-122` — type `HighlightEvent`
- `apps/go-api/internal/analysis/performance_score.go` — `PercentileRank`, `PercentileRankInverse`
- `apps/go-api/internal/analysis/temporal/lowess.go` — `LowessSmooth`
- `apps/go-api/internal/analysis/temporal/rolling.go` — `RollingMean`, `RollingMeanAdaptive`
- `apps/go-api/internal/analysis/match_history_avg.go` — categorisation modes
- `apps/go-api/internal/analysis/sessions.go` — `ComputeSessions`
- `apps/go-api/internal/port/highlight_events.go` — interface acces events
- `apps/go-api/internal/migration/steps_shared.go:118-126` — schema `highlight_events`
- `apps/web/src/components/charts/TimeseriesLineChart.tsx` — composant graphe inter-match
- `.ai/SPEC_ECHARTS_TIMESERIES.md` — blueprint buildOption ECharts
- `.ai/mockups/forme/forme_visualizations.html` — 6+ mockups visuels

---

## 15. Hors scope volontaire

- Implementation Go (theorie tant que H1-H7 non validees sur donnees reelles)
- Comparaison inter-joueurs sur le FormScore (intra-personnel par construction)
- Coupe par playlist a l'interieur d'une categorie de mode
- Prise en compte du teammate quality (ponderation par MMR coequipiers)
- Detection automatique de tilt / fatigue (label categoriel)
- Generation narratif texte automatique (v2 a partir des dimensions visuelles)
- PvE / Firefight non couverts v1
- Lobby massivement quitte non gere v1
- Trace lobby/ennemie sur la courbe forme : deja couvert par tug of war + cadence existants

---

## 16. Questions ouvertes restantes

- **Largeur fenetre glissante (W)** : 60s, 90s, 120s ? 90s point de depart raisonnable.
- **Echantillonnage de la courbe** : 5s, 10s, 15s ? 10s par defaut.
- **Faut-il inclure les `assist` dans le compte d'events** ? Vote : inclure MVP, retirer si H1 montre bruit.
- **Seuil "creux" pour mort passive** : 30s par defaut, a calibrer par categorie de mode.
- **Poids unitaire des events_objectif_estimes** : ~25 points par capture, a calibrer par mode.

---

## 17. Livraison — TODO documentation et i18n

Au moment de l'implementation, il faudra livrer en parallele les artefacts documentaires suivants :

- **[ ] Glossaire applicatif** : ajouter une entree dediee au concept "FormScore" qui explique :
  - La distinction skill (PerfScore) vs forme (FormScore) — pour eviter la confusion utilisateur
  - La formule du residu : "ecart entre engagement observe et engagement attendu, vu ton style historique"
  - La lecture des 3 courbes (equipe / attendu / joueur) sur Match View et Session
  - L'interpretation des 3 zones (en forme > P75, normale P25-P75, sous-forme < P25)
- **[ ] Glossaire EngagementCoefficients** :
  - `coef_team_share` : "part historique du joueur dans le travail de ses equipes"
  - `coef_lobby_share` : "part historique du joueur dans l'action totale du lobby"
  - Lecture combinee des deux (cf §7.2 du present doc)
- **[ ] Glossaire MatchIntensity** : caracteristique objective du match (events/min/joueur lobby), comparable inter-matchs
- **[ ] Manifest i18n** : creer `apps/web/src/lib/i18n/manifests/forme.toml` avec :
  - Labels FR/EN des 3 traces (Equipe alliee / Attendu / Toi)
  - Phrases narratives types (au-dessus de votre habitude, sous-forme, etc.)
  - Tooltips de chaque mockup
- **[ ] Documentation hypotheses validation** : seuils de declenchement de FormScore (cold start a 30 matchs), conditions d'invalidation (match < 3 min, lobby majoritairement quitte), modes exclus (PvE, FFA fallback)
- **[ ] Mockup file synchronise** : `.ai/mockups/forme/forme_visualizations.html` doit refleter **Mock 10** (Match View) et **Mock 11** (Session) comme choix valides finaux, avec exigences §8.6 (auto-zoom + hierarchie visuelle)

---

## Annexe A — Diagnostic des 5 tentatives precedentes

| # | Approche | Statut | Pourquoi insatisfaisant |
|---|---|---|---|
| 1 | LOWESS sur perf_score (Squad V2) | Actif | Lisse du **skill**, pas de la forme |
| 2 | EWMA KD (Timeseries) | Partiel | Signal de skill base sur KD post-match |
| 3 | Form Score 14 vs 90 (Python) | Non porte Go | Bonne idee de delta court vs long, mais sur perf agrege = **skill** |
| 4 | Performance Score relatif | Production | Definition du skill, pas un signal de forme |
| 5 | Win Streak | Anecdotique | Plus long streak passe — ni skill ni forme actuels |

**Le point aveugle commun** : aucun ne distingue conceptuellement skill et forme, aucun n'exploite le contexte d'equipe + style historique pour modeliser un attendu. Le FormScore actuel introduit explicitement ces deux ingredients.

---

## Annexe B — Pourquoi le residu unique > composite a 5 signaux

Le composite a 5 sous-signaux (E1-E5) etait sophistique mais bricole :
- 5 normalisations heterogenes (relative equipe pour certaines, personnelle pour d'autres)
- Poids fixes arbitraires (0.30, 0.25, 0.20, 0.15, 0.10 — aucune justification rigoureuse)
- Le diagnostic decompose n'est **jamais consulte en pratique** par l'utilisateur final
- Risque de double-comptage entre signaux (E1 densite et E3 persistance partagent de l'info)

Le residu unique resout ces problemes :
- **Une seule normalisation** (percentile vs son propre historique)
- **Aucun poids** a justifier
- Diagnostic visuel dans la courbe (les 5 dimensions y sont lisibles directement)
- **Transparence du calcul** : "ecart entre observe et attendu, vu ton style historique"

Trade-off : un narratif texte automatique necessite de re-extraire les dimensions de la courbe (E2 visible comme largeur de bande post-mort, etc.). Cout v2 acceptable.

---

## Annexe C — Pourquoi la normalisation par equipe (et non lobby) est centrale

Sans normalisation, le pace absolu d'un match pollue le signal forme :
- Match Slayer 12 min equilibre : ~80 events lobby
- Match Slayer 4 min steamroll : ~50 events concentres
- Match Strongholds tactique : longues phases calmes
- Match BTB chaotique : events partout

Avec normalisation **lobby** : on annule le pace absolu, mais on **ne distingue pas** :
- Joueur bien engage dans une equipe ecrasee (qui apparait artificiellement faible vs lobby)
- Joueur passager dans une equipe dominante (qui apparait artificiellement fort vs lobby)

Avec normalisation **equipe** : on isole la **contribution individuelle au sein du collectif**.

Demonstration chiffree (4v4, 110 events lobby) :

| Scenario | Equipe alliee | Equipe ennemie | Joueur cible | lobby_share | team_share | Lecture forme correcte |
|---|---|---|---|---|---|---|
| Equipe ecrasee | 30 | 80 | 8 | 0.58 (bas) | **1.07** (normal) | Le joueur fait sa part malgre le steamroll |
| Equilibre | 55 | 55 | 14 | 1.02 | 1.02 | Identique |
| Equipe dominante, joueur passager | 80 | 30 | 25 | 1.82 (haut) | **1.25** (modere) | Le joueur beneficie du contexte mais ne porte pas |

`team_share` est structurellement le bon denominateur pour la **forme individuelle**. `lobby_share` est plus utile comme metric de **profil/style** (dans EngagementCoefficient) car il capture l'effet equipe ET l'effet individuel ensemble.
