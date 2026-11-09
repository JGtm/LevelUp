# HANDOFF — Escouade, onglet Usages → « Tactique » (2026-09-26)

> Chantier de conception, AUCUN code écrit, rien commité (hors ce fichier, la maquette et le
> journal, eux aussi non commités). À lire en entier avant de reprendre. Source de vérité : la
> maquette publiée et les décisions ci-dessous ; la conversation d'origine est résumée ici.

## 1. Où en est le chantier

- **Maquette** : https://claude.ai/artifact/6ZKxrug6cLzkxpYs2VWYaP (version 18), copie locale
  `.ai/V7.5/MAQUETTE_ONGLET_TACTIQUE_ESCOUADE_2026-09-26.html`. Chiffres RÉELS de la soirée du
  22/09 (JGtm + Chocoboflor + Madina97294, 7 matchs), relevés en lecture seule dans
  `shared_matches_v2.duckdb` et `players/JGtm/stats.duckdb` (serveur arrêté, CLI `duckdb -readonly`).
- **Journal** : `.ai/thought_log.md`, entrée « [2026-09-26] Escouade, onglet Usages… » et ses
  compléments 1 à 16 (l'historique complet des itérations et des chiffres).
- **Validé par l'utilisateur** : tout l'onglet jusqu'à « Match par match » inclus, et tous les
  DÉPLACEMENTS de cartes hors de l'onglet (le principe ; la forme des cartes déplacées reste à revoir,
  §4).
- **Non encore validé** : le bas de l'onglet (« Ce que chaque camp en tire », « Rendement face à
  l'adversaire », le nuage « Groupés ou isolés », « Par rapport à d'habitude » en deux graphes) — la
  v18 intègre ses derniers retours mais il ne l'a pas encore revue.
- **Prochaine étape demandée par l'utilisateur** : revoir la FORME et les DONNÉES de tout ce qui a
  quitté l'onglet, avec les mêmes angles que ci-dessous (§3). Pas d'implémentation avant son « ok »
  explicite ; ensuite plan sous `plan-execution`, exécuteur Opus dans un worktree dédié.

## 2. L'onglet proposé (état v18)

Onglet renommé « Tactique » (ex-Usages). Sept questions à servir, sur UNE soirée (la session
sélectionnée) : groupés ou non ; riposte en groupe ; maîtrise des bonus / armes spéciales /
véhicules face à l'adversaire ; ramassé puis utilisé ou perdu ; détail par carte ; bilan vu de
l'extérieur ; par rapport à d'habitude.

| Bloc | Carte | Forme retenue |
|---|---|---|
| Bilan de la soirée | Qui a tenu la carte | Une barre par ressource (bonus, armes spéciales, véhicules), deux segments : notre camp (`team-ally`) / adversaire (`team-enemy`), compte et part ÉCRITS DANS les segments ; trait orange à 50 % |
| Bilan de la soirée | Au fil de la session | UN graphe : une courbe cumulée par ressource (couleurs de ressource), petits points pâles par match (taille = volume), bande de résultats sous l'axe (case victoire/défaite, encoche de dominance) |
| Rôles dans l'escouade | Qui prend quoi chez nous | Une fiche par joueur (liseré à sa couleur), MÊMES lignes dans le même ordre sur chaque fiche (zéro = ligne atténuée), une pastille par prise ; bonus : pastille vide = perdu (gardé ou lâché) ; ligne « Bonus perdus : nous x sur y, eux… » |
| Carte par carte | Match par match | Colonnes = matchs (heure, carte, mode, « Victoire 3–0 », badge de dominance si drapeau) ; lignes = ressource puis chaque objet de la carte ; case = « 5–2 » colorée plus / moins que l'adversaire ; armes de râtelier repliées |
| Prendre, et s'en servir | Ce que chaque camp en tire | Barre épaisse = part des frags (valeurs dedans), barre fine dessous = part de l'exposition (temps d'effet / prises) |
| Prendre, et s'en servir | Rendement face à l'adversaire | Écart relatif de rendement (+14 %), axe commun −50/+50 %, valeur au BOUT de la barre, rendements bruts de l'autre côté du zéro |
| Groupés ou isolés | Nos morts… | Le nuage de Synergies (déplacé) : un point par mort, x = distance / portée radar, y = délai de riposte (log), zones annotées (part des morts, taux de riposte en 5 s) |
| Par rapport à d'habitude | Les prises / Le jeu groupé | Deux graphes soirée après soirée, ce soir à droite, médiane des soirées précédentes en pointillé fin |

Définitions tranchées : « armes spéciales » = socles de PUISSANCE (les râteliers à part) ;
« d'habitude » = les 10 soirées précédentes de la même composition, sur les familles de mode jouées
ce soir-là, chaque soirée comptée à part ; tout l'onglet sur les matchs de la COMPOSITION (plus le
périmètre `filteredMatches` du bloc d'usage).

## 3. Les goûts et objectifs de l'utilisateur (à appliquer à la suite)

Ce qui a été dit ou tranché pendant la séance, à reprendre tel quel :

1. **Un angle de lecture par carte, une question par carte.** Une carte qui répète une autre sous un
   autre dénominateur est retirée. Deux lectures qui se recoupent → les séparer nettement (prises /
   production / rôles).
2. **Le graphique avant le texte.** Pas de phrase de synthèse ni de statistiques en texte au bout
   des lignes (« ça rend la lecture moins fluide ») ; seule une valeur courte au bout d'une courbe
   est acceptée.
3. **Les valeurs DANS les barres** quand elles y tiennent, au bout sinon ; jamais seulement en
   infobulle ; ligne au-dessus uniquement en repli.
4. **Couleurs d'équipe plutôt que mots** : pas de « Nous » / « Eux », `team-ally` / `team-enemy`
   (réglables par l'utilisateur). La hachure n'est plus l'adversaire (elle reste pour « sans film »).
5. **Pas de jargon** : « parité » est remplacé par « 50 % : autant que l'adversaire ».
6. **Un seul graphe quand les séries partagent une mesure et une échelle** (plutôt que trois petits
   graphes et trois lignes de référence).
7. **Tendances en courbes** : cumul au fil de la soirée ; soirée après soirée pour l'habitude.
8. **Alignement strict** : mêmes lignes, même ordre, même hauteur d'une fiche / d'un joueur à l'autre ;
   un zéro reste une ligne.
9. **Rôles sans jugement** (pas de classement ni de vert/rouge sur la répartition interne) ; un fait
   (pastille vide = bonus perdu) est accepté.
10. **Vu de l'extérieur = camp contre camp** ; l'adversaire est utile seulement quand il éclaire NOTRE
    jeu (son propre isolement n'intéresse pas).
11. **Code couleur des ressources pour toute l'app** : bonus sarcelle, armes spéciales violet,
    véhicules orange, armes de râtelier bleu (valeurs validées au validateur de palette, clair et
    sombre, dans la maquette). Pastille de couleur devant chaque nom de ressource.
12. **Comparer à l'habitude sur des modes similaires**, pas tous modes confondus ; montrer la
    dispersion plutôt qu'un delta seul.
13. **Résultat toujours visible**, la dominance en plus quand elle existe (82 % des matchs n'ont pas
    de drapeau).
14. **Méthode attendue** : chiffres réels, le challenger (« tu peux me challenger »), décider plutôt
    que proposer dix variantes, une recommandation par point, réponses courtes.

## 4. À revoir maintenant : ce qui a quitté l'onglet

Les destinations sont VALIDÉES. Reste à revoir, pour chaque carte, sa forme et ses données avec les
angles du §3, dans sa page d'arrivée (et sans défaire ce qui y existe déjà).

| Carte | Arrive sur | Points à instruire |
|---|---|---|
| Écart cumulé au FDA attendu | Escouade › Dynamique | Cohérence avec les autres cumuls de l'onglet (balance des dégâts, écart d'engagement) ; lignes par joueur aux couleurs `squad-player-*` ; valeur de fin au bout de courbe seulement |
| Répartition des frags | Escouade › Contributions | Barres empilées par classe : valeurs dans les segments ? couleurs de classe à rapprocher du code ressource (armes lourdes = violet des armes spéciales ; véhicule indigo `frag-vehicle` vs orange ressource : écart à trancher) |
| Outils de destruction (et Précision par rôle, Halo 5) | Escouade › Contributions | Doublon partiel avec Répartition des frags ? une seule carte « avec quoi on frague » ? |
| Rapport de force par famille de mode · Ce que mon camp prend de l'objectif | Escouade › Contributions | Deuxième rendu d'objectifs de la page (les 9 tuiles d'API au-dessus des onglets) : réconcilier les deux dénominateurs (constat du 2026-09-21) ; forme « fiches » ou « camp contre camp » selon la question |
| Frags non ripostés (nuage) | Escouade › Tactique (depuis Synergies) | Fait (§2) ; vérifier ce qui reste de la section Coordination de Synergies (Appui, Riposte, rôles de portée / hauteur) et sa lisibilité sans le nuage |
| Grappin, mur, capteur et autres équipements | Synthèse et Séries temporelles (solo) | Les usages solo gardent aujourd'hui les « formes retenues » et le bloc « servi ou gâché » (dix-huit cartes côté Escouade avant refonte : même surcharge probable côté solo) ; appliquer les mêmes angles |
| Camouflage / surbouclier en violet (`frag-heavy`) | Sessions, Synthèse | Passer au sarcelle des bonus (code couleur, §3.11) |

Cartes RETIRÉES (absorbées, pas déplacées) : les deux donuts « Notre part… », « Contrôle des armes
spéciales », « Écart à la parité par famille d'arme », « Détail des prises », « Emprise match par
match », « Régularité match par match », « Part de mon camp », « Cadence de chacun », « Qui porte quel
usage », « Taux de rafle », « Contrôle des armes par niveau » — voir la matrice « Aujourd'hui » de la
maquette pour la correspondance carte → nouvelle carte.

## 5. Données : ce qui manque (du plus léger au plus lourd)

Tableau complet et références dans la maquette (section « Ce que les données ne disent pas
encore ») et dans le rapport de l'enquêteur résumé au journal. L'essentiel :

- **Go minime** : un seul périmètre (composition) pour les blocs d'usage ; résultat, carte et
  drapeau de dominance sur chaque match du bloc ; bonus vidés par match (`powerup_pickups_json`, lu
  mais non transmis) ; frags et temps d'effet des bonus (`camo_kills`, `overshield_kills`,
  `camo_ms`, `overshield_ms` : stockés pour tout le lobby, jamais lus).
- **Agrégat Go** : comptes pris / utilisé / gardé / lâché par camp (calculés, seuls des taux sont
  publiés) ; habitude (10 soirées, mêmes familles de mode) ; camp entier et adversaire pour
  morts groupées et riposte ; frags depuis un véhicule côté adversaire.
- **Correctif Go + rattrapage** : la projection au fil de l'eau des niveaux d'armes n'écrit rien
  (WARN « identités de match illisibles », `sync/replayartifacts/padtiers.go:327`) — sans elle,
  pas de séparation armes spéciales / râteliers pour les matchs récents.
- **Rattrapage seul** : `backfill-usage-summary` (152 matchs résumés ; 13 soirées sur 46 pour
  l'habitude venue du film).
- **Nouvelle mesure du film** : véhicules (montées, pertes, temps à bord) ; arme spéciale prise
  puis lâchée sans un tir.

## 6. Brief pour l'agent qui reprend les cartes déplacées

**Livrable** : UN artefact de maquette (même grammaire que la maquette de l'onglet Tactique :
jetons de l'app, clair et sombre, chiffres réels) qui, pour chaque page d'arrivée du §4, montre
« Aujourd'hui » (les cartes telles qu'à l'écran, y compris celles qui arrivent) puis
« Proposition », et un TRI par carte : garder telle quelle, garder en changeant la forme, fusionner
avec une autre, retirer — chaque verdict justifié en une phrase par la question de la page.

**Le critère du tri** : une carte reste si elle répond à la question de sa page mieux qu'une autre
carte de la même page. Axes de lecture PROPOSÉS (à faire confirmer par l'utilisateur en tête
d'artefact, en une ligne chacun, pas en questionnaire) :

| Page / onglet | Question proposée |
|---|---|
| Escouade › Synergies | Comment l'escouade s'en sort-elle ensemble (résultats, cartes, historique, coordination) ? |
| Escouade › Contributions | Qui apporte quoi à l'escouade (frags, armes, objectif, impact, médailles) ? |
| Escouade › Dynamique | Comment la soirée évolue-t-elle au fil des matchs (cumuls, intensité, engagement) ? |
| Synthèse / Séries temporelles (solo) | Comment je joue, moi (usages d'équipement compris) ? |

**Méthode** : relever les pages dans le code (ordre réel à l'écran, sources de données), mesurer en
lecture seule (§6 bis), appliquer les règles du §3, recommander une option par carte, ne rien coder.

## 6 bis. Pièges et règles de reprise

- Le principal `LevelUp` est PARTAGÉ : une autre session y avait des modifications non commitées le
  26/09 (`CLAUDE.md`, `docs/CONTRIBUTING.md`, `docs/FR/CONTRIBUTING.md`, une entrée du journal).
  Ne rien committer d'autrui ; stager explicitement.
- Tout lot de code : worktree dédié, exécuteur Opus, plan écrit avant, « ok » explicite de
  l'utilisateur (une remarque n'est pas une validation).
- Lire `.ai/V7.5/REFERENCE_CANAUX_EQUIPEMENT_2026-09-09.md` avant toute affirmation sur
  l'équipement ; son §4 (« Escouade : aucun bloc d'usage ») est périmé, à corriger au prochain
  commit qui touche ce document.
- Requêtes de mesure : serveur arrêté, `duckdb -readonly`, jamais d'ouverture RW ; les rejeux du
  22/09 ne sont pas dans le cache local (`data/cache/replays/halo_infinite/`), ceux du 07/09 oui.
- Pas de navigateur piloté : la vérification visuelle est faite par l'utilisateur sur l'artefact.

## 7. Cartes déplacées : tranché et validé le 2026-09-26 (suite)

Maquette : https://claude.ai/artifact/C3EWYk6cNEhNzhqnWqPo45 (v8), copie
`.ai/V7.5/MAQUETTE_TRI_CARTES_DEPLACEES_2026-09-26.html`. Périmètre = les CINQ cartes « déplacées » de la
matrice « Aujourd'hui » de 6ZKxrug6 (pas le reste des pages d'arrivée).

- **Écart cumulé au FDA attendu → Dynamique** : valeur de fin au bout des courbes, pastilles « écart moyen » retirées, abscisse de la balance des dégâts.
- **Répartition des frags → Contributions** : comptes dans les segments s'ils tiennent (6 px de marge), sinon ligne de repli au-dessus ; total au bout ; légende en bas centrée. Couleurs : échanger Véhicule et Tourelle.
- **Outils de destruction → Contributions** : carte séparée (pas de fusion) ; légende joueurs, compte au bout, pastille de classe ; toutes les armes nommées par le film (plus de « Autres armes » ni de repli « Grenade » sur la feuille de match ; `fragdist.go:366`).
- **Objectif → Contributions, quatre cartes** : Rapport de force par famille de mode (camp contre camp, familles encadrées) · L'objectif au fil de la session (parts Prendre / Défendre / Tenir cumulées, chaque match pèse pareil, masquée sous 3 matchs à objectif) · Qui prend quoi à l'objectif chez nous (style des fiches de médailles, contenu en lignes par action) · L'objectif, soirée après soirée (même mesure, 10 soirées précédentes à ≥ 3 matchs à objectif tous modes, résultat sous chaque soirée ; pas de ratio composite). « Ce que mon camp prend de l'objectif » disparaît ; la conversion au Drapeau est abandonnée.
- Les 9 tuiles d'objectifs de l'en-tête ne s'affichent jamais (code mort, champ servi par la seule route v2).
