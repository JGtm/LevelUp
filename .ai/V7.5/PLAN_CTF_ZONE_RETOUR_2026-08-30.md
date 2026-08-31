# PLAN — CTF : la ZONE DE RETOUR du drapeau (rejeu 2D)

> Commande utilisateur du 2026-08-30 : « quand le drapeau de notre équipe est en dehors de son
> emplacement, il a une zone autour de lui. Se mettre dedans permet de le retourner plus vite (je
> crois que plus on est plus ça retourne vite, mais c'est du feeling). Du coup la jauge se vide
> plus vite. De base si personne est dedans la jauge se vide et le drapeau finit lui-même par
> revenir. C'est un angle que j'ai oublié de prendre en compte dans le replay. »
>
> Branche : `wt/ctf-zone-retour` (worktree dédié `LevelUp-wt-ctf-retour`), depuis `feat/v75`.
>
> **ÉLARGISSEMENT du 2026-08-31**, sur demande de l'utilisateur après lecture du premier lot :
> « la contestation je la veux bien stp. Pour le drapeau neutre aussi faut le gérer. » Les deux
> étaient au registre des questions ouvertes ; ils entrent au périmètre (§4 ter et §4 quater).
> Le drapeau neutre était explicitement HORS périmètre dans la première rédaction — ce paragraphe
> est ce qui reste de cette version, gardé pour que la trace du changement se lise.

---

## 1. Ce que le rejeu fait aujourd'hui, et le trou exact

`flag_carries_lives.go` tient quatre états par drapeau : `home`, `carried`, `carried_open`,
`dropped`. Trois transitions seulement : prise, fin de portage, et `flag_returns` (le retour
**crédité** à un joueur). Son en-tête dit le trou en toutes lettres :

> LE RETOUR AUTOMATIQUE N'EST PAS SIMULÉ (…) le délai a été cherché sur les trois films CTF par
> l'écart entre une fin de portage sans reprise et la prise suivante au socle : la distribution
> est trop dispersée pour qu'un seuil s'en déduise.

Conséquence mesurée sur les artefacts en cache (17 films reconnus CTF,
`data/cache/replays/halo_infinite`) : **36 spans `dropped` se terminent par un `home`** (retour
crédité), mais des spans de **100 à 162 secondes** subsistent — un drapeau posé au sol pendant deux
minutes et demie n'a jamais existé à l'écran.

> **CORRECTION du 2026-08-31, à lire avec ce chiffre.** Ces 17 artefacts ne sont pas tous du CTF
> ordinaire : la confrontation à `match_registry` (§4 quater) montre qu'au moins NEUF d'entre eux
> sont des parties `CTF:Arena Neutral Flag`, où le calque publiait alors deux drapeaux au lieu
> d'un. Le constat de départ tient (les lâchers interminables sont réels et se voient aussi sur le
> corpus de contrôle, tout en CTF ordinaire), mais le dénominateur « 17 films CTF » mélangeait deux
> variantes.

Il manque donc trois choses :

1. le **retour automatique** (la minuterie qui ramène le drapeau) ;
2. la **zone** autour du drapeau tombé (rayon) ;
3. la **jauge** de retour, et son accélération avec le nombre de défenseurs présents.

## 2. ACQUIS — le jeu NOMME sa mécanique (relevé du 2026-08-30, fichiers du jeu)

Les tags `hsc*` de Halo Infinite sont du Lua compilé qui garde ses noms en clair. Le script
`scripts/ParcelLibrary/parcel_deliver_object.lua` (tag `hsc*` dans
`any/globals/common-rtx-new.module`) porte **toute** la mécanique du drapeau, nommée :

| Ce que le user décrit | Le nom que le jeu lui donne |
|---|---|
| les états du drapeau | `FlagStates = { DoesNotExist, Incoming, OnStandDefend, OnStandDeliver, InHand, Resetting, Returning, Contested, ContestedRefilling }` |
| la jauge | `flagReturnTimer`, `flagReturnTimerRate`, `onReturnProgress` |
| la zone | `innerAreaMonitorRadius`, `outerAreaMonitorRadius`, `cylinderHeight`, `cylinderPositiveHeightScalar`, `cylinderNegativeHeightScalar` |
| entrer / sortir de la zone | `OnBipedEnterInnerRadius` / `OnBipedExitInnerRadius` / `…OuterRadius`, et les raisons `Owner|NonOwnerPlayerEntered|ExitedReturnRadius`, `…ContestRadius` |
| « plus on est, plus ça va vite » | **`CalculateReturnRateHarmonic`** + `UpdateReturnRate`, avec `innerBipedCountPerTeam` / `outerBipedCountPerTeam` |
| la minuterie sans personne | `flagResetSeconds`, `FlagReturnTimerMonitorThread`, état `Resetting` |
| le retour au contact | `flagTouchReturnEnabled` |
| l'ennemi qui bloque | `flagContestedStateEnabled`, `flagContestRefillRate`, états `Contested` / `ContestedRefilling` |
| le cercle affiché à l'écran | `drawBoundaryOnDrop` |
| le son | `flagReturnLoopTeam` / `flagReturnLoopEnemy` / `flagResetLoop` (+ variantes `flagBTB…`), RTPC `ctf_return_status` |

**Trois enseignements qui changent le modèle** :

- **`CalculateReturnRateHarmonic`** : l'intuition du user est CONFIRMÉE par le nom même de la
  fonction — plus de défenseurs = plus vite, mais en série **harmonique** (rendement décroissant :
  1, 1 + 1/2, 1 + 1/2 + 1/3 …), le même patron que l'accélération de capture des zones.
- **DEUX rayons** : un `inner` (le retour) et un `outer` (la contestation). L'ennemi présent dans
  le rayon extérieur CONTESTE et fait REFAIRE la jauge (`ContestedRefilling`) — un angle que le
  user n'a pas mentionné et qu'il faudra lui soumettre.
- **`flagResetSeconds`** existe : la minuterie nue est une constante de configuration, pas une
  émergence.

**Ce que ce relevé ne donne PAS** : les VALEURS. Le pool de constantes Lua ne rend ses nombres
qu'au prix d'un décodage du chunk ; les noms, eux, sont en clair. Les valeurs se cherchent donc
(a) par la sonde `cmd/tmp_ctflua` (rapprochement nom ↔ double voisin dans le pool), (b) à défaut
par la MESURE sur le corpus, qui reste l'arbitre.

## 3. La chaîne d'observation neuve : la NAISSANCE DE L'OBJET AU SOCLE

La tentative antérieure mesurait un PROXY (l'écart entre le lâcher et la prise suivante au socle).
Le présent plan lit l'**objet** : `flag_objects.go` produit déjà les **vies libres** du drapeau
(`ti=42`, mot MPP `0x2A392328`), et une vie qui **naît à moins de 1,5 m d'un socle** est le
drapeau qui rentre — datée à la frame. C'est une observation, pas une inférence.

**Le contrôle qui l'autorise** : sur les retours que le statborg CRÉDITE (`flag_returns`), les deux
chaînes doivent tomber à la même frame. Deux lectures disjointes (compteurs de statistique d'un
côté, records de création du monde de l'autre) : leur accord est une preuve, leur désaccord un
refus. Seuil écrit AVANT la mesure : **≥ 80 % des retours crédités datés à ≤ 1 s par la chaîne
objet**. En dessous, la chaîne objet ne sert pas à dater les expirations.

## 3 bis. RÉSULTATS DU LOT 0 (mesures du 2026-08-30/31)

### Les VALEURS que le jeu déclare — pool de constantes Lua DÉCODÉ

Le chunk `hsc*` range ses constantes en entrées typées **gros-boutistes** :
`0x00` nil · `0x01`+1 o booléen · `0x03`+4 o flottant simple · `0x04`+8 o longueur + chaîne.
Une table littérale compile en `SETFIELD` successifs : le pool alterne donc clé, valeur — sauf
quand la valeur **doublonne** une constante déjà émise, auquel cas elle est référencée et
n'apparaît pas. Déroulé de la table `CONFIG` de `parcel_deliver_object.lua` :

| Champ | Valeur lue |
|---|---|
| `updateDeltaSeconds` | **0,1** (tick de 100 ms — exactement le pas du rejeu) |
| `innerAreaMonitorRadius` | **1,3** |
| `outerAreaMonitorRadius` | doublon (la seule valeur déjà émise qui soit un rayon est 1,3) |
| `cylinderPositiveHeightScalar` / `…Negative…` | **0,5** / **0,4** |
| `cylinderHeight` | **2** |
| `flagCarrierMovespeedScalar` | **0,715** |
| `flagCarrierSprintSpeedScalar` | **1,0595** |
| `flagCarrierSlideSpeedScalar` | **1,25** |
| `flagCarrierTraitsLingerTimeSec` | **0,3** |
| `flagResetSeconds` | **15** (défaut de la BIBLIOTHÈQUE — `FlagInitArgs.returnTimer` l'écrase par instance) |

`flagCarrierMovespeedScalar = 0,715` est le **contrôle du décodage** : c'est la pénalité de
vitesse du porteur de drapeau, connue et vérifiable en jeu. Un décodage faux ne la rendrait pas.

**L'unité est celle du rejeu.** Les positions du film et les socles du catalogue vivent dans le
même espace que les tags ; le dépôt l'appelle « mètres » (calage `0,0920 m/px` de
`carte_gate_gamefiles_test.go`, cartes de ~100 unités de côté). **Le rayon de retour vaut donc
1,3 dans les coordonnées du rejeu** — un cylindre à peine plus large qu'un Spartan : « se mettre
dedans », c'est marcher sur le drapeau.

### Ce que la MESURE dit (corpus : les 2 films CTF de Catalyst — Behemoth n'a pas de socles au catalogue)

- **ACCORD DES DEUX CHAÎNES** — et **deux corrections de méthode ont été nécessaires avant que le
  chiffre veuille dire quelque chose**, l'une et l'autre écrites ici parce qu'elles se
  reproduiraient :

  1. **CIRCULARITÉ.** La première version lisait le retour crédité sur les spans `home` du
     document. Or la production ramène désormais AUSSI le drapeau sur la naissance de l'objet :
     un `home` ne dit plus laquelle des deux chaînes l'a produit, et l'instrument confrontait la
     chaîne objet à elle-même (il annonçait 81,8 % pour cette raison). Corrigé : les retours
     crédités se relisent sur les **événements bruts du statborg**.
  2. **DÉNOMINATEUR.** `flag_returns` **NE NOMME PAS SON DRAPEAU** — c'est la raison même pour
     laquelle la production s'abstient quand deux drapeaux sont au sol. Rapporté aux ÉPISODES, un
     même événement se retrouve donc attribué aux DEUX drapeaux tombés, et la moitié de ces
     attributions est fausse par construction (mesuré : les instants 2109 et 2670 de `64e8adfa`
     sont crédités aux deux drapeaux à la fois). La question licite est par **ÉVÉNEMENT
     distinct** : « ce retour crédité a-t-il une naissance d'objet à moins d'une seconde ? ».
     Et les films dont la carte est **absente du catalogue d'objectifs** (`53ce4390`, Behemoth)
     sortent du dénominateur : sans socle, la chaîne objet se tait par construction — les compter
     mesurerait la couverture du catalogue de cartes, pas l'accord des deux lectures.

  **RÉSULTAT SOUS CETTE MÉTHODE : 15 / 15 = 100 %.** Les 15 retours crédités distincts des deux
  films de Catalyst ont TOUS une naissance d'objet au socle à moins d'une seconde — écarts
  min = 0, **médiane = 1 frame**, max = 1 frame. (`53ce4390` sort du dénominateur avec ses 10
  crédits, et le journal de l'instrument le dit en clair.) Seuil écrit avant la mesure : 80 %.
  **TENU, et largement** — la chaîne objet est licite pour dater les retours automatiques.
- **RAYON, corroboration indépendante** : sur les 36 retours crédités des artefacts en cache, le
  défenseur le plus proche à l'instant du retour est à **moins de 1,2 m dans 23 cas sur 36** ; sur
  les 25 épisodes retournés de l'instrument, 19 ont un défenseur à moins de 1 m. La valeur du jeu
  (1,3) est cohérente avec la distribution ; l'ajustement par dispersion, lui, **ne tranche pas**
  (cv ≥ 0,92 partout) — l'échantillon est trop petit pour départager 1 m de 3 m tout seul.
- **MINUTERIE NUE — LA MESURE LA PLUS FAIBLE DU LOT, et il faut le dire.** Sur le corpus corrigé,
  six épisodes se terminent par un retour sans qu'aucun défenseur n'ait été vu à moins de 1,3 m :
  **2,2 / 2,2 / 3,4 / 9,0 / 11,2 / 29,1 s**. Les courts ne sont PAS des expirations — ce sont des
  retours joueur dont la position n'a pas été échantillonnée dans le rayon (la réplication est
  irrégulière), et le diagnostic se lit dans le même tableau : à 3 m le plus long tombe à 11,2 s,
  à 15 m il ne reste AUCUN épisode « désert ». Il reste donc **une seule observation propre :
  29,1 s** — cohérente avec les 30,0 / 31,4 / 32,2 s d'un premier passage moins strict, et avec le
  fait qu'aucun épisode inoccupé ne dépasse ce seuil. Le défaut de bibliothèque est 15 s, écrasé
  par instance (`FlagInitArgs.returnTimer`) : une seule observation à 29,1 s le CONTREDIT
  franchement (un drapeau seul serait rentré à 15 s). **Retenu : 30 s, comme la meilleure valeur
  soutenue, pas comme une mesure serrée** — inscrit au registre des reports pour re-mesure sur un
  corpus élargi.

## 4. Lots

### Lot 0 — RECHERCHE (instrument, aucune production touchée) — EN COURS

`ctf_retour_zone_research_test.go`, sous garde `OBJ_FILM` / `OBJ_REPO`, sur les trois films CTF du
corpus de la phase 0 (`64e8adfa` Catalyst, `530820e5` Catalyst, `53ce4390` Behemoth — leurs
équipes sont gelées dans `objCorpus`).

- [ ] **0.1** accord des deux chaînes (seuil ≥ 80 %, ci-dessus) ;
- [ ] **0.2** rayon : pour chaque candidat de 1 à 15 m, le séjour ininterrompu dans la zone avant
      le retour. Le bon rayon est celui qui MINIMISE la dispersion à occupation égale ;
- [ ] **0.3** loi : séjour à 1 défenseur contre séjour à plusieurs — confrontation au patron
      harmonique nommé par le jeu ;
- [ ] **0.4** expiration : durée des épisodes retournés SANS qu'aucun défenseur n'entre jamais ;
- [ ] **0.5** valeurs du jeu : sonde `cmd/tmp_ctflua` sur le pool de constantes du script Lua.
      Ce qu'elle rend est un CANDIDAT, que la mesure valide ou écarte.

### Lot 1 — Le retour AUTOMATIQUE, daté (Go)

- [ ] `flag_carries_lives.go` : une naissance de l'objet au socle du drapeau ferme son `dropped`
      et ouvre un `home`, exactement comme un `flag_returns` le fait déjà ;
- [ ] la couverture publie le compte des retours datés par chaque chaîne (crédit / objet / aucune),
      pour que le silence reste lisible ;
- [ ] la doc de `flag_carries_lives.go` est corrigée dans le MÊME commit (la section « LE RETOUR
      AUTOMATIQUE N'EST PAS SIMULÉ » devient fausse — anti-pattern « doc inversée »).

### Lot 2 — La ZONE et la JAUGE, publiées (Go, schéma +1)

- [ ] `FlagSpan` (état `dropped`) porte le rayon de la zone et la progression de la jauge, ou un
      calque frère `flagReturnZones` — arbitrage au moment de l'écriture, selon ce que le lot 0
      autorise à affirmer ;
- [ ] la jauge se calcule sur la loi mesurée et **atterrit sur l'instant de retour OBSERVÉ** : le
      modèle donne la forme, l'observation donne les bornes ;
- [ ] `openapi.yaml` + `generated.ts` régénérés dans le même commit.

### Lot 3 — Le rendu (web)

- [ ] cercle de la zone autour du drapeau tombé du camp propriétaire, jauge de retour, état
      accéléré quand des défenseurs sont dedans ;
- [ ] tokens sémantiques uniquement, strings FR **et** EN dans `i18n.ts` ;
- [ ] pas de logique dans le composant (hook / `*_logic.ts`).

### Lot 4 — Clôture

- [ ] `go test ./...` + `go vet` + typecheck/lint/vitest ; entrée `.ai/thought_log.md` ;
      `cmd/tmp_ctflua` SUPPRIMÉ ; registre des reports mis à jour.

## 4 bis. LE DRAPEAU NEUTRE — ce que disait la première rédaction

> Le user exclut explicitement le mode drapeau neutre (…) Ce qu'il faudrait pour bien faire, si le
> user le demande un jour : reconnaître la variante. Le constructeur du rejeu est hors ligne et ne
> connaît pas `game_variant_name` ; le signal disponible côté film serait la présence d'un objet
> d'objectif unique naissant au socle NEUTRE. Non mesuré, donc non écrit.

Le user l'a demandé le lendemain, et c'est exactement ce signal qui a été mesuré et écrit :
**§4 quater**. Le paragraphe reste ici parce que la prédiction était juste — elle vaut mieux
consignée que remplacée.

## 4 ter. LOT 5 — LA CONTESTATION (demandée par le user le 2026-08-31)

**Ce que le jeu fait**, lu dans la table de constantes de `UpdateFlagState` :
`GetAnyEnemyTeamInOuterArea` → état `Contested`, puis `ContestedRefilling` au taux
`flagContestRefillRate` — la jauge repart en arrière. Deux rayons sont surveillés
(`innerAreaMonitorRadius` / `outerAreaMonitorRadius`) ; les états et les raisons distinguent
`ReturnRadius` de `ContestRadius`.

**Ce qu'on a pu établir, et ce qu'on n'a pas pu.**

- Le rayon de contestation est **1,3**, comme celui du retour. Ce n'est pas une supposition de
  confort : sa valeur est **DÉDUPLIQUÉE** dans le pool de constantes (le chunk ne réémet pas une
  valeur déjà présente), donc elle vaut l'une des constantes déjà émises à ce point — `0,1`,
  `true` ou `1,3` — et **1,3 est la seule qui soit un rayon**. La cohérence vient d'ailleurs : les
  deux cylindres diffèrent par leur HAUTEUR (`cylinderInnerHeight` / `cylinderOuterHeight`, deux
  champs distincts vus dans le constructeur d'args), pas par leur rayon.
- **Le TAUX DE RECUL, lui, reste inconnu** : `flagContestRefillRate` est dédupliqué parmi une
  dizaine de nombres, et rien ne dit lequel. Or `false` est émis très tôt dans la même table
  (`complete = false`), donc la déduction « booléen dédupliqué ⇒ vrai » ne tient pas non plus :
  `flagContestedStateEnabled` n'est pas lisible non plus.

**Décision de rendu, et pourquoi elle est la plus faible des deux** : le rejeu **TIENT** la jauge
pendant la contestation au lieu de la faire reculer. Un recul inventé serait une vitesse fausse à
l'écran ; un arrêt est seulement une progression qu'on ne réclame pas. Et l'arrêt ne fait pas
mentir la fin : la jauge est remise à l'échelle pour atteindre 1 à l'image du retour OBSERVÉ — la
contestation redistribue la forme, jamais les bornes. Le taux entre au registre des reports, avec
sa condition de reprise (décoder le BYTECODE, pas seulement le pool).

## 4 quater. LOT 6 — LE DRAPEAU NEUTRE (demandé par le user le 2026-08-31)

**Le trou** : `replaybuild/flagspawns.go` écartait le socle neutre en amont, pour une raison
valable (sur une partie ordinaire il ferait un troisième drapeau immobile). Conséquence : une
partie **à drapeau neutre** publiait DEUX drapeaux qui n'existaient pas et répartissait entre eux
les portages d'un objet unique. Le parc est loin d'être marginal : **25 matchs
`CTF:Arena Neutral Flag`** dans `match_registry`, et **23 de leurs films sont en cache**.

**Le mode n'est pas dans le film** — le constructeur du rejeu est hors ligne et ne connaît pas
`game_variant_name`. Mais l'OBJET tranche : il ne renaît que CHEZ LUI.

| | où l'objet renaît |
|---|---|
| variante ordinaire | aux socles D'ÉQUIPE (41 / 16 / 18 naissances à 0,0 m, mesure du 18/08) |
| drapeau neutre | au socle NEUTRE, seul point de retour |

**Seuil écrit avant la mesure** : au moins **3** naissances au socle neutre ET strictement plus
qu'aux socles d'équipe. Le défaut est la variante ordinaire — une naissance égarée au centre ne
bascule rien, un film muet garde le comportement d'avant. On se trompe du côté qui ne casse rien.

**L'ORACLE est disjoint du discriminant** : le discriminant ne lit que le film, l'oracle est la
variante déclarée par l'API du titre (`match_registry.game_variant_name`). Corpus de contrôle :
`a1995edc` (Forest), `323ec1cf` et `e94163af` (Bazaar) — trois parties à drapeau neutre dont la
carte est au catalogue d'objectifs — confrontées aux trois films ordinaires du corpus.

**Ce que le mode neutre change en aval** : rien d'autre que le jeu de socles. Un seul socle donne
un seul drapeau, d'équipe -1 ; sa rentrée se date comme les autres ; et **la zone de retour
disparaît d'elle-même** — elle appartient au camp propriétaire, et un drapeau neutre n'en a pas.
Le client ne trouve ni défenseur ni contestataire, la jauge se réduit à la minuterie. C'est
exactement la règle du jeu, obtenue sans qu'aucune branche ne la dise.

## 5. Questions ouvertes

1. ~~La contestation~~ — **DEMANDÉE ET LIVRÉE le 2026-08-31** (§4 ter), avec sa limite écrite :
   la jauge est TENUE, pas reculée, faute d'avoir pu lire `flagContestRefillRate`.
2. ~~Le drapeau neutre~~ — **DEMANDÉ ET LIVRÉ le 2026-08-31** (§4 quater).
3. ~~Le retour au contact~~ — **TRANCHÉ PAR LA MESURE le 2026-08-31, faute de pouvoir le lire.**
   `flagTouchReturnEnabled` reste illisible (booléen dédupliqué, et `false` est émis plus tôt dans
   la même table). Mais la mesure répond à sa place : un renvoi au CONTACT donnerait un séjour
   NUL dans la zone, or le séjour vaut **3,1 s à un défenseur** et n'est jamais nul — le contact
   seul ne renvoie donc pas en Arena. La phrase « le toucher le RENVOIE » de `flag_carries.go`
   est corrigée dans le même commit ; **la règle d'attribution qui s'appuyait dessus, elle, ne
   bouge pas** : instantané ou non, un joueur ne PORTE pas son propre drapeau.
4. **Le socle neutre est bien au MILIEU** (confirmé par l'utilisateur le 2026-08-31, et par le
   catalogue) : Aquarius bases (−13,0) / (13,0) socle neutre (0,0) ; Bazaar (−23,0) / (23,0) →
   (0,3) ; Catalyst (0,21) / (0,−21) → (0,0) ; Behemoth 75 / 32 → 54. C'est exactement le point
   que le mode neutre retient. **LIMITE** : quelques cartes du catalogue ne déclarent AUCUN socle
   neutre (Absolution, Banished Narrows) — sur celles-là le discriminant ne peut pas basculer et
   garde la variante ordinaire, ce qui est le bon défaut mais serait faux si elles hébergeaient
   la variante. Non mesuré.
