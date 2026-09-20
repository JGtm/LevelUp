# ORACLE PRO V2 — Audit et densification par navigateur (item 2bis.A)

> Date : 2026-09-20. Étape 2bis du `.ai/PLAN_POSITIONS_DE_FORCE_2026-09-20.md`, item 2bis.A.
> Worktree `LevelUp-wt-power-positions`, branche `wt/power-positions`.
>
> Ce document AUDITE `.ai/V7.5/positions_de_force/ORACLE_PRO_2026-09-20.md` (v1) et le
> DENSIFIE avec des sources ouvertes au navigateur réel (MCP `chrome-devtools`) là où
> WebFetch avait été refusé en v1 : Reddit (r/CompetitiveHalo), YouTube, liquipedia,
> halo.fandom.com, x.com. Toujours un ORACLE DE VALIDATION, jamais une source de données —
> rien ici n'est mesuré sur nos matchs.

---

## 1. Méthode

### 1.1 Outillage navigateur — ce qui marche, ce qui reste bloqué (testé sur pièces le 2026-09-20)

| Site | Accès | Constat |
|---|---|---|
| reddit.com (r/CompetitiveHalo) | **OUI** | `new_page`/`navigate_page` chargent la page normalement. Deux voies : (a) la page rendue (HTML + composants web `shreddit-*`), traduite en FR par défaut selon la langue du navigateur — angle mort à éviter ; (b) suffixer l'URL du fil par `.json` (API publique de Reddit) : rend le JSON brut en ANGLAIS, non traduit, avec `selftext` et l'arbre des commentaires (`replies.data.children`) — **beaucoup plus fiable et economique en tokens** que `take_snapshot`, utilisé pour la quasi-totalité de la collecte. Recherche via `/r/CompetitiveHalo/search.json?q=...&restrict_sr=1`. |
| youtube.com | **PARTIEL** | Résultats de recherche et page vidéo accessibles ; le titre, la description et les chapitres (`ytInitialData.engagementPanels[].engagementPanelSectionListRenderer`) se lisent par `evaluate_script`. **Les sous-titres/transcriptions restent INACCESSIBLES** : l'endpoint `timedtext` répond HTTP 200 mais `content-length: 0` (testé sur la vidéo « Sherzys Shouts — Forbidden », `itIdhX0MKeM`, avec et sans `fmt=json3`) — confirme et PRÉCISE le constat v1 (« ne sert ni description ni transcription ») : la description SERT, la transcription NE SERT PAS. Peu de vidéos utiles trouvées : la majorité des vidéos « callouts »/« pro tips » de 2021-2022 sont des tutoriels bas de gamme (quelques centaines de vues, mots-clés SEO en description, aucun chapitre). Aucune VOD d'analyse HCS structurée trouvée par la recherche par mots-clés.
| liquipedia.net/halo | **NON** | Défi Cloudflare (« Verify you are human ») qui ne se résout pas seul même après 8 s d'attente dans un navigateur réel avec JS actif (`wait_for` en échec). Confirme le blocage v1, cette fois avec un vrai navigateur et pas seulement WebFetch. |
| halo.fandom.com | **NON** | Interstitiel (« Un instant… ») qui ne se résout pas non plus après 8 s d'attente. Confirme le blocage v1. |
| x.com | **NON** | La recherche redirige vers un mur de connexion (`/i/jf/onboarding/web?...mode=login`) : lecture impossible sans compte. Nouveau constat (v1 avait juste noté un 402 via WebFetch ; le navigateur réel montre que c'est un mur d'auth, pas un blocage anti-bot). |
| gamecoach.gg | **OUI (partiel)** | Site d'un outil de « strategy board » interactif, avoué par des coachs/analystes HCS nommés (Stuart « Warlord » Graham, Vanadam, Audley « TiberiusAudley » Wright — retrouvé ensuite comme intervenant actif sur r/CompetitiveHalo, voir §2). Couvre TOUTES les cartes visées par cette étape (Aquarius, Recharge, Streets, Origin, Solitude, Lattice, Argyle, Interference, Forbidden, Live Fire, Empyrean, Fortress). Le contenu tactique lui-même (plans) est derrière un compte ; la page `/articles` ne liste aucun article public. Utilisé comme preuve d'EXISTENCE d'un outillage coach reconnu sur ces cartes, et comme fil pour retrouver les bons intervenants sur Reddit, pas comme source de citations. |

### 1.2 Volume de collecte

**~35 fils Reddit ouverts** (recherche + lecture), **~10 requêtes de recherche** ciblées par
carte, **2 vidéos YouTube** inspectées (description + tentative de transcription), **2 sites
bloqués** (liquipedia, halo.fandom) vérifiés sur pièces à chaque fois (pas de suppositions
reprises de v1 sans re-test), **1 site coach** (gamecoach.gg) consulté pour la couverture de
cartes et l'identification d'intervenants. Chaque citation ci-dessous est vérifiée verbatim
depuis le JSON brut récupéré par le navigateur (pas de résumé de résumé).

### 1.3 Règle de confiance (reprise et précisée de v1)

`forte` = au moins **2 sources indépendantes** (éditeurs / auteurs distincts, thread Reddit
compté comme UNE source quel que soit son nombre de commentateurs) qui nomment la même zone
avec un rôle compatible. `faible` = une seule source, ou rattachement non attesté. Un fil
Reddit dont plusieurs utilisateurs distincts convergent SANS se citer entre eux est traité
comme une seule source (le thread), pas comme N sources — discipline plus stricte que ce que
la richesse du fil suggérerait, pour ne pas gonfler artificiellement le compte.

**Colonne `v1v2`** : `v1` = ligne déjà dans l'oracle v1, inchangée ; `v2` = ligne nouvelle ;
`v1→forte` = ligne v1 `faible` promue `forte` par une source v2 (ou, dans deux cas Bazaar,
par une correction d'audit — voir §2.6) ; `v1↓` = aucune ligne rétrogradée par v2 (la seule
rétrogradation du chantier est celle déjà actée par le pilote en v1 §7, Forbidden
`Center Bridge`, reprise telle quelle ici, non re-signalée).

---

## 2. Audit de l'oracle v1

### 2.1 Rattachements lexicaux de v1 §5.2 — relecture zone par zone

| Rattachement v1 | Verdict v2 | Preuve |
|---|---|---|
| Live Fire `Platform` (« cylindrical platform ») | **Tenu, renforcé** | 2e source indépendante (Reddit, §2.2) : la position OS/Platform est au centre d'une discussion à 33 commentaires sur QUI la tient et COMMENT. |
| Recharge `Platform` (« Hydro platform ») | **Tenu, PROMU forte** | Reddit « Breaking Recharge Setups » : « C plat has a good cross into whirlpool and batteries... also use it to look into attic » — 2e source indépendante, avec RAISON EXPLICITE de ligne de vue (voir §2.3). |
| Catalyst `Top Mid` (« the central area ») | **Tenu, non renforcé** | Aucune source Catalyst nouvelle trouvée malgré recherche ciblée (§2.7) — carte toujours la plus mal servie. |
| Catalyst `Middle Ramp` (« long midway ramp ») | **Tenu, non renforcé** | idem. |
| Forbidden `Center Bridge` (« the center of the map ») | **Rétrogradation du pilote VALIDÉE et PRÉCISÉE** | Le vocabulaire complet du dépôt (relu en pièces, `map_callouts.json`) montre que `Dried/Overgrown Mid` sont des zones DISTINCTES de `Center Bridge`. La source `gamersdecide` (« the middle of the map ») que v1 laissait en `?` résout maintenant proprement vers `Mid`, pas vers `Center Bridge` — ce qui retire à `Center Bridge` l'appui qu'on aurait pu lui trouver a posteriori. La prudence du pilote était justifiée : ce sont deux zones différentes. |
| Argyle `East/West Rampart` (« high-up battlement ») | **Tenu, non contredit, non renforcé par le nom** | Une note de coaching Reddit (Warlord/Stuart Graham, §2.8) décrit une progression de contrôle par étage mais en vocabulaire non officiel (« snipe elbow », « vent », « pistols ») qui ne cite jamais « Rampart ». Cohérent thématiquement (hauteur tenue par étapes) mais pas une confirmation nominale. |
| Empyrean `East/West Tower` (« Sniper Tower ») | **Tenu, PROMU forte** | 2 fils Reddit Infinite-spécifiques (pas des transferts Halo 3) décrivent explicitement la Tower comme point de spawn-kill et de contrôle (§2.9). |
| Empyrean `East/West Balcony` (« a new balcony ») | **Tenu, non renforcé** | Aucune mention Reddit trouvée. |
| Solitude `Top Nest` (« sniper nest ») | **Tenu, PROMU forte, ET contradiction v1 résolue** | Le fil « Tips for Playing Solitude? » (§2.10) est Infinite-spécifique et unanime sur la valeur de Nest — alors que la source v1 (gamedeveloper, sur Halo 5 Plaza) se contredisait elle-même (« 2-3 members guarding » vs « there isn't a clear vantage point »). La v2 tranche dans le sens de la position : sur Infinite (verticalité modifiée par rapport à Plaza, attesté par halowaypoint en v1), c'est une position forte. |
| Solitude `Bridge` (« central walkway above the street ») | **Tenu, PROMU forte** | Même fil : « Hold snipe and os bridge if you can. » |
| Banished Narrows `Top Mid` (« main bridge ») | **Tenu, non renforcé** | Recherche Reddit `"Banished Narrows"` : **0 résultat** sur r/CompetitiveHalo (voir §2.12). Carte confirmée hors de portée de cet outillage. |

**Bilan de l'audit lexical** : sur 11 rattachements relus, **0 invalidé**, **4 promus** (Recharge
Platform, Empyrean Tower ×2, Solitude Bridge — Solitude Top Nest est un 5e mais c'était déjà
listé comme rattachement), **7 non renforcés mais non contredits**.

### 2.2 Les 4 positions « arme seule » — sont-elles des positions TENUES ?

| Position | Raison v1 | Ce que v2 ajoute | Verdict |
|---|---|---|---|
| Live Fire `Hallway` | arme | Rien de nouveau ne décrit `Hallway` comme un lieu qu'on TIENT (hauteur, accès restreint) ; toujours décrit comme le point où l'arme apparaît, au centre, traversé. | **Toujours « arme seule »**, pas de promotion en position tenue. |
| Recharge `Pit` | arme | Idem : aucune source v2 ne décrit `Pit` comme défendable — c'est un creux central que les guides v1 eux-mêmes classent aussi comme piège (cible du tir plongeant d'Attic, §3.1 de v1). | **Toujours « arme seule »**, et le statut de piège partiel est confirmé, pas infirmé. |
| Streets `Main Street` | arme | Reddit (`winLadin`, fil « Help with power positions », §2.4) confirme EXACTEMENT la lecture v1 §4 : la zone se rush pour l'arme puis se quitte — jamais décrite comme tenue. | **Toujours « arme seule »**, doublement confirmé comme non-tenue. |
| Forbidden `Center Bridge` | arme | Aucune source v2 trouvée (voir §2.1). | **Toujours « arme seule »**, statut inchangé. |

**Conclusion** : aucune des 4 positions « arme seule » de v1 n'est promue en position tenue
par les sources v2. Le distinguo posé par le pilote (§7 de v1 : « les manquer n'est pas la
même faute que manquer une hauteur ») reste valide et est renforcé pour Streets et Recharge.

---

## 3. Densification carte par carte

### 2.3 Recharge — `sgh_blueprint`

Fils lus : « Breaking Recharge Setups » (11 votes, réponses de `TiberiusAudley`, `supalaser`,
`Chugstar`, `Stockasaurus_Rex`, `ParappaGotBars`) ; « Help with power positions » (généraliste,
mais une citation Recharge précise) ; « Recharge Map Callouts » (WIP, 19 commentaires,
corrobore le vocabulaire sans ajouter de position).

Citations retenues :

- `supalaser` : *« C plat has a good cross into whirlpool and batteries when combined with
  bottom control bridge. You can also use it to look into attic which can combine great with
  top and bottom control »* → **`Platform` : PROMU forte** (raison : hauteur + lignes de vue,
  explicite sur 3 cibles à vue : Whirlpool Dam, Batteries, Attic).
- `Chugstar` : *« two teammates go control, one top and one bottom... Control can make sure
  they aren't on A cross »* → **`Control Room` : PROMU forte** (raison : hauteur + lignes de
  vue, tenue à deux joueurs, un par étage — confirme les deux niveaux `Control Room` /
  `Control Room's second floor` de v1 comme un seul point de contrôle vertical).
- `Charming_Toe9438` (fil « Help with power positions ») : *« spawn recharge C plat. You move
  long hall shoot at A overhang »* ; `Chugstar` : *« one teammate long hall »* →
  **`Long Hall` : PROMU forte** (raison : arme + accès, relie C au flanc A/Overhang).
- `TiberiusAudley` : *« In general though the best advice is probably to focus on getting A
  since being trapped at C or B is the worst case scenario »* — confirme l'importance de la
  zone A (`Elevator` en v1, faible) mais SANS nommer `Elevator` précisément ; **pas de
  promotion**, gardé comme corroboration non nominale dans « ce qu'on ratait ».

### 2.4 Live Fire — `sgh_interlock`

Fils lus : « Live Fire - What's a good setup for OS? Which side is better, C or A side? »
(74 votes, 33 commentaires — le fil le plus riche du chantier) ; « Unexplored Live Fire Back
Green Ledge META » (25 votes).

- `flowers0298` : *« Bound and other pros have said on stream if you hold tower side of the
  map you win most game types. One reasoning I've heard for this is because there is little
  cover on the A&B side of the map »* — citation d'un pro NOMMÉ (Bound), 3e source
  indépendante pour `Tower` (déjà forte). Renforce sans changer le statut.
- Discussion à 10+ intervenants sur la tenue de `Platform` (OS) : *« the first guy pushes past
  OS... allowing the second person to grab OS and then clean up kills »* (`BrettTheEskimo`,
  101 points) ; *« C side... It's about space and concave »* (`ryanoob`) — confirme `Platform`
  comme position CONTESTÉE et disputée à 2-3 joueurs, pas un simple ramassage. Déjà forte en
  v1, statut inchangé mais raison enrichie.
- **NOUVEAU (`v2`)** : « Back Green Ledge » — une position d'embuscade en hauteur, accès
  restreint (saut précis), confirmée utilisée par des pros (« I've seen sparty do it »,
  `Balkanoboy`). Zone officielle **non tranchable** : le dépôt a `Green Bend` ET
  `Green Building` pour `sgh_interlock`, et rien dans le fil ne permet de choisir entre les
  deux → ligne `zone_en = ?`, conservée pour traçabilité, non exploitable au rappel.

### 2.5 Aquarius — `ctf_aquarius`

Fils lus : « Tips and advice for playing Aquarius » (66 votes — réponse de **KingJayHCS**,
131 points, un joueur/caster HCS identifié) ; « Am I High? Or did they change the distance of
the jump from Shock to Top Mid » (confirme la route saut Hydro→Top Mid, pas une nouvelle
position) ; « How to pull flag through Car 1 » (contre-exemple).

- `KingJayHCS` : *« For flag You want to trap them in spawning fridge, overload shock side...
  have someone stay P2 but do not push it just stay on your side so the spawners spawn
  fridge & always run the flag shock made side »* → **NOUVEAU (`v2`)** : `Blue/Yellow
  Refrigeration` comme cible de spawn-trap (raison : objectif + accès), citée par un pro
  nommé. Une seule source → `faible` (pas de 2e source trouvée), mais qualité de source
  élevée.
- `KingJayHCS` (suite, sur la route de drapeau) : *« running it across the bridge to bottom
  utility then down the hall is the best. Limits angles you can be shot from »* → **NOUVEAU
  (`v2`)** : `Blue/Yellow Utility`, raison **lignes de vue explicite** (« limits angles »),
  `faible` (source unique).
- `areeb_onsafari` : *« courtyard is a death trap if the other team is spawning P »* →
  **NOUVEAU contre-exemple (`v2`)** : `Blue/Yellow Courtyard`.
- Aucune 2e source trouvée pour promouvoir `Planters`, `Yellow Base` ou `Blue Base` malgré
  recherche ciblée (`"yellow base" OR "blue base" OR utility`) : Aquarius reste à **2 fortes**
  (`Top Mid`, `Hydro`), inchangé, malgré une recherche dédiée.

### 2.6 Streets — `sgh_streets`

Fils lus : « Help with power positions » (citation Streets) ; « Streets Map Callouts »
(456 votes — corrobore le vocabulaire, pas de nouvelle position tactique).

- `winLadin` : *« Push 2 shotgun to cafe to relieve pressure off of B to open back up the map
  and causing the other team to have to scramble to regain map control »* → **`Cafe` : PROMU
  forte** (v1 avait déjà `progameguides`, 1 source, raison « arme » seule ; v2 ajoute une 2e
  source indépendante avec une raison DIFFÉRENTE et plus forte : objectif + accès, pas
  seulement l'arme).
- Aucune source nouvelle pour `Subway Balcony`, `Main Street` (déjà fortes), ni pour
  promouvoir les 9 autres positions faibles.

### 2.7 Forbidden — `ctf_forbidden`

Fils lus : « Sherzys Shouts - Forbidden (Map callouts) » (vidéo + commentaires) ; « Since I
haven't seen any official Forbidden callouts I made a small attempt at it » (41 votes, dont
une intervention de `TiberiusAudley`) ; « Optic Halo set Forbidden Flag World Record »
(251 votes, aucun contenu positionnel, réactions seulement).

**Résolution de deux lignes `?` de v1**, en relisant le vocabulaire complet du dépôt
(`map_callouts.json`, confirmé sur pièces — le dépôt a bien `Dried/Overgrown Mid`,
`Dried/Overgrown Nest`, `Dried/Overgrown Hut`, distincts de `Center Bridge`, ce que v1 n'avait
pas détaillé) :

- `? | the middle of the map | lignes de vue | gamersdecide` → résout en **`Dried Mid` /
  `Overgrown Mid`** (rattachement lexical direct, « middle » = « Mid »). `faible` (source
  unique), mais maintenant EXPLOITABLE au rappel/précision (n'est plus `?`).
- `? | snipers on either side | arme | halowaypoint` → résout en **`Dried Nest` /
  `Overgrown Nest`** (motif récurrent du dépôt : « Nest » = poste de sniper sur Live Fire,
  Solitude, Interference — confirmé par `TiberiusAudley` et `Nin10do0014` sur Reddit qui
  distinguent explicitement `Nest` de `Hut`, zone voisine). **Réserve** : le dépôt porte AUSSI
  `Dried/Overgrown Back Nest`, une zone distincte — la source ne permet pas de choisir entre
  `Nest` et `Back Nest`. Résolution donnée avec cette ambiguïté explicite, `faible`.
- `TiberiusAudley` : *« What you label as bridge/nest is usually referred to as Hut »* —
  confirme que la communauté utilise bien les 3 noms du dépôt (`Bridge`/`Nest`/`Hut`) comme
  des zones distinctes, cohérent avec le catalogue, mais n'attache aucun rôle tactique
  nouveau.
- Aucune 2e source trouvée pour promouvoir `Dried/Overgrown Rat Hole` au-delà de leurs 2
  sources déjà fortes en v1, ni pour `Center Bridge`. Forbidden reste à **2 fortes**.

### 2.8 Argyle — `dd600260-d91c-4d77-9990-3f35873c90a1`

Fil lu : « Give me tips for Argyle and Catalyst » (13 votes) — réponse de `GenesForLife`
rapportant des notes prises pendant une **séance de coaching avec Warlord** (Stuart Graham,
retrouvé comme la même personne que le témoignage « Esports Coach and Player » sur
gamecoach.gg, confirmé par `gmalsparty` : *« Stu took me from d2/3 to 1600 in like 6
sessions »*).

Citation longue, verbatim (extrait) : *« General rule – you want to spawntrap them in their
snipe elbow, and run the flag their pistols... Getting map control in stages - ... our
objective should always be to get base control, then pistol control, and then control of our
vent... You want your sniper hanging back by your vent so you don't block their snipe spawns
and you block them from spawning vent room. »*

C'est la source la plus QUALITATIVE trouvée pour Argyle (coach HCS nommé, méthode par étapes),
mais elle est **non exploitable pour le tableau final** : elle emploie un vocabulaire («
snipe elbow », « pistols », « vent », « vent room »/« E1 ») qui ne correspond à AUCUNE des 5
zones officielles du dépôt (`East/West Stairs`, `East/West Rampart`, `Platform`). Conforme à
la règle « aucun nom deviné », ces éléments sont documentés en §4 (« ce qu'on ratait »)
plutôt que dans le tableau final, avec zone `?`.

### 2.9 Empyrean — `d035fc3e-f298-4c14-9487-465be2e1dc1f`

Fils lus : « Tips for "The Pit" aka Empyrean? » (22 votes, 15 réponses) ; « faze vs ssg coms -
empyrean » (pros nommés : Renegade, Royal2, Snakebite) ; « How to pull the flag through
mauler → needles → long hall on Empyrean/Pit CTF » (83 votes).

- Fil « Tips for Pit aka Empyrean » (`[deleted]`, 32 points) : *« get on their tower and start
  spawn killing them »* ; « Push training/sword side so you can get control of their
  courtyard, from their you can get on their tower and start spawn killing them » → **`East
  Tower` / `West Tower` : PROMUS forte** (v1 ne tenait que le transfert halopedia depuis The
  Pit, `faible` ; cette source est Infinite-spécifique, nomme explicitement Empyrean, 2e
  source indépendante).
- « faze vs ssg coms - empyrean » : *« Renegade grabbed overshield and rockets with royal 2
  and snakebite pushing long hall »* — confirme une position centrale forte (« overshield et
  rockets ») accédée par « long hall », terme informel SANS correspondance dans les 12 zones
  officielles d'Empyrean (`Pit, Alley, Elevator, Deck, Gulch, Terminal, Underpass, East/West
  Base, Balcony, Bend, Tower, Catwalk`). Zone `?`, non exploitable, mais confirme la richesse
  du jeu pro sur cette zone.
- **Contradiction relevée** (voir §4) : v1 affirme (halopedia) que l'épée a été remplacée par
  le Heatwave sur Empyrean ; les fils 2024-2025 emploient pourtant « sword » activement
  (« training/sword/court », « sword bridge », « sword ramp »). Non tranché — signalé au
  pilote plutôt que deviné.

### 2.10 Solitude — `f1cc3b4e-471c-4ec5-b855-1db7d9e6ce42`

Fil lu : « Tips for Playing Solitude? » (le fil décisif pour cette carte — question ouverte
« What are some general power positions, setups, or sightlines/areas to play for? »).

Citations (5 intervenants distincts, dont `TiberiusAudley` encore) :

- `TiberiusAudley` : *« Control the grey areas. »*
- `who_likes_chicken` : *« rotate from grey 1 to grey 2 when someone's pushing grey 3 »*
- `DeathByReach` : *« Controlling snipe and quantum are a must. The little window over A that
  can see a lot of the map is really good. Dancing around the pillar in B is a must. »*
- `ProperFormatt` : *« the sniper spawn/grav lift area is the most powerful part of the map to
  control. You have the height advantage of Nest and the repulsor spawns by the grav lift...
  The high ledge at A is a powerful spot »*
- `JJSpleen` : *« Hold snipe and os bridge if you can. »*

Résolutions vers le vocabulaire officiel (`Cafe, Hotel, Ledge, Bridge, Bottom/Top Nest, Loop,
Market, Yard, Hall, Long Stairs, Window, Stage, Posters, Alley, Underpass, Junction, Dip,
Pillar, Bottom Mid, ...`) :

- **`Top Nest` : PROMU forte** (« snipe »/« Nest », 3 intervenants convergents dans le même
  fil comptés comme 1 source v2 + source v1 gamedeveloper = 2 indépendantes). Résout aussi la
  contradiction interne de v1 (§2.1) : la position EST forte sur Infinite, la réserve de
  Halo 5 ne s'y transfère pas.
- **`Bridge` : PROMU forte** (« os bridge », 2e source indépendante après halopedia-Plaza).
- **NOUVEAU (`v2`) `Window`** : *« can see a lot of the map »* — lignes de vue explicites,
  zone officielle EXISTANTE (`Window`) totalement absente de v1. `faible` (source unique).
- **NOUVEAU (`v2`) `Pillar`** : zone `B` (objectif), tenue de position via mouvement autour du
  pilier. `faible` (source unique).
- **NOUVEAU (`v2`) `Ledge`** : *« high ledge at A is a powerful spot »* — hauteur explicite,
  zone officielle EXISTANTE (`Ledge`), absente de v1. `faible` (source unique).
- « grey areas », « quantum », « grav lift » : termes informels sans correspondance officielle
  nette → `?`, documentés en §4.

### 2.11 Bazaar — `ctf_bazaar`

Fil lu : « How the hell are you supposed to play Bazaar CTF » (32 votes, 45 réponses).

**Correction d'audit (pas une nouvelle source)** : en relisant le tableau v1 ligne par ligne,
`Cafe` et `Den` portent CHACUNE déjà 2 URLs de domaines différents (`dotesports.com` +
`gamerguides.com`) mais sont marquées `faible` — en contradiction avec la règle de confiance
que v1 énonce elle-même (§1.4 de v1 : « forte = au moins 2 sources indépendantes »). Aucune
autre ligne du tableau v1 ne présente cette incohérence (vérifié sur les 84 lignes). **`Cafe`
et `Den` : PROMUES forte par correction d'audit**, `v1→forte`, sans source v2 nouvelle.

Sources v2 nouvelles (`conn1925`, 77 points) : *« One person on lockers, one person on palm,
one person on BR, and one person on rocks »* — un patron de tenue à 4 joueurs, explicite :

- **NOUVEAU (`v2`) `Lockers`** : zone officielle existante, absente de v1. `faible` (source
  unique).
- **`Palm Tree` : PROMU forte** (v1 dotesports, faible, + ce fil = 2 sources).
- **`Tower` : PROMU forte** (« rocks » = le Rocket Launcher documenté par v1 à `Tower` ; v1
  dotesports, faible, + ce fil = 2 sources).

### 2.12 Cartes sans zone officielle, sans position exploitable — chiffrer l'absence

Recherches dédiées effectuées et négatives, comme demandé par le gate en cas d'échec :

- **Fortress** (`0d1c9255-…`) : 1 fil dédié lu (« New Fortress map first opinions in ranked »,
  33 réponses). Contenu : *« large open space = melted from every angle »*, mention d'un
  « Ring », d'un « basement », confirmation à 2 sources désormais (halowaypoint v1 + Reddit
  v2) des « 2 snipes ». **Aucune zone officielle dans le dépôt** (0 zone, confirmé) → aucune
  ligne possible. Le fait que la carte joue « conservateur, bords tenus, centre évité » est
  cohérent avec une lecture « pas de position de force centrale », mais reste hors table.
- **Origin** (`b302eb62-…`) : 2 fils lus (« Coliseum (Halo 5) Remake In Halo Infinite »,
  164 votes ; « New Updates To Our Map Origin »). **0 contenu tactique** — uniquement des
  réactions d'annonce et d'esthétique. **0 zone officielle dans le dépôt.**
- **Lattice** (`1a6cfc2e-…`) : 3 fils lus (« First Pro 8's match on new HCS map Lattice »,
  « Thoughts / Feedback for new maps: Lattice & Serenity », 40 votes). Contenu RICHE en
  positions informelles — `whyunoname` : *« holding in red you want to get top control and
  get them spawning blue elbow »* ; `Who_told_you_that` : *« you can have a really strong C
  setup »* — mais **0 zone officielle dans le dépôt** pour rattacher « red », « blue elbow »,
  « top control », « C », « Kryptonite » (callout entendu en partie pro 8s). C'est le cas le
  plus clair du chantier : le SAVOIR pro existe, c'est le CATALOGUE qui manque, pas la
  tactique.
- **Banished Narrows** (`9ad226d8-…`) : recherche `"Banished Narrows"` sur r/CompetitiveHalo :
  **0 résultat**. Confirmé : hors de portée de cet outillage, comme v1 l'avait déjà constaté
  par un autre chemin (absence de source tactique propre).
- **Catalyst** (`catalyst`) : 1 fil dédié lu (« Give me tips for Argyle and Catalyst »).
  Contenu généraliste (`donutmonkeyman` : stratégie de « zone defense » sur grande carte,
  non spécifique à une zone nommée). **Deux lignes `?` résolues** malgré tout, en relisant le
  vocabulaire complet de la carte (`Balcony, Base, Bottom Mid, Catwalk, Closed Hall, Grand
  Hall, Middle, Middle Ramp, Mohawk, Open Hall, Overgrowth, Overlook, Pit, Plank, Spine, Top
  Mid, Window`) :
  - `? | the closed-off areas on the side | acces | gamerant` → **`Closed Hall`**
    (rattachement lexical direct et littéral : « closed-off » = « Closed »). `faible`.
  - `? | a higher position / high ground | hauteur | gamerant` → **`Overlook`** (rattachement
    lexical générique : le nom même désigne un point de surplomb). `faible`, réserve notée :
    rattachement moins littéral que le précédent.
  - Catalyst reste à **0 forte**, comme en v1, malgré la recherche dédiée.
- **Interference** (`654dff62-…`) : 1 fil lu (« Top-down callouts, strongholds and hills for
  Interference », 101 votes) — confirme le vocabulaire (`Bunker`) et l'attribution à
  gamecoach.gg (callouts basés sur une vidéo Twitter de Shyway), mais n'ajoute pas de
  position nouvelle exploitable au-delà de v1 (déjà tout en `faible`/transfert). **0 forte**,
  inchangé.

---

## 4. Ce qu'on ratait (positions absentes de v1, et pourquoi)

1. **Solitude `Window`, `Pillar`, `Ledge`** — trois zones officielles du dépôt, complètement
   absentes du tableau v1, alors que le dépôt les connaît depuis le début. v1 s'est appuyé sur
   des sources Halo 5 (Plaza) qui ne les nomment jamais explicitement comme positions ; seule
   une question directe à la communauté Infinite (« What are some general power positions,
   setups, or sightlines... ? ») les a fait remonter. Leçon : pour un remake, une recherche
   ciblée sur la carte Infinite ELLE-MÊME (pas seulement l'originale) est nécessaire et rendait
   ces trois positions invisibles à v1.
2. **Bazaar `Lockers`** — zone officielle, absente de v1, révélée par un patron de tenue à 4
   décrit par la communauté (Lockers/Palm/BR/Rocks), pas par un guide écrit classique. Les
   guides « grand public » (dotesports, gamerguides) listent des zones de hauteur mais aucun
   ne décrit le patron de TENUE complet ; Reddit le fait.
3. **Recharge `Control Room` comme point de contrôle vertical à deux joueurs** — déjà connu en
   v1 comme faible/hauteur, mais sa vraie valeur (bloquer la diagonale A, tenir à deux étages
   en même temps) n'apparaît que dans une discussion tactique de rupture de setup, pas dans un
   guide de carte.
4. **Empyrean, Solitude, Interference : le vocabulaire manque plus que la tactique.** Sur les
   trois cartes, des positions réelles et actives (« long hall »/« rocket hall » sur Empyrean,
   « grey areas »/« quantum » sur Solitude, callouts spécifiques sur Interference) sont
   décrites avec précision par la communauté SANS jamais utiliser le nom officiel du dépôt.
   Le blocage n'est pas l'absence de connaissance pro, c'est l'absence de correspondance
   nom-communautaire → nom-du-catalogue. Ce chantier ne doit PAS deviner cette correspondance
   (règle du plan) ; il la signale pour un futur travail d'extraction Forge ciblé.
5. **Lattice : le cas le plus net.** Une carte sortie il y a moins de deux mois (5 août 2026)
   a déjà un vocabulaire communautaire vivant (« red », « blue elbow », « top control », « C
   setup », « Kryptonite ») ET une partie pro 8s documentée (Legend, Formal, Suppressed,
   Cherished vs Stellur, Renegade, Falcated, Manny) — mais 0 zone officielle au dépôt. La
   tactique existe, le catalogue n'existe pas encore.
6. **Contradiction Empyrean « sword »** (voir §2.9) — signalée, non tranchée : v1 (halopedia)
   dit l'épée remplacée par le Heatwave, les fils 2024-2025 emploient « sword » activement.
   Peut être un vocabulaire hérité (comme « Gold »/« Red » sur Recharge, §2.3) qui survit au
   changement d'arme, ou un changement de sandbox postérieur à la page halopedia consultée par
   v1. À vérifier sur pièces avant toute décision qui en dépendrait.
7. **KingJayHCS sur Aquarius** — un joueur/caster identifié donne une doctrine de spawn-trap
   complète (Refrigeration + Utility + Hydro) qu'aucun guide écrit de v1 ne documentait ;
   utile mais reste `faible` faute de 2e source.

---

## 5. Notion de hauteur / ligne de vue explicite (utile à la voie géométrique, 2bis.C)

Positions où la source dit EXPLICITEMENT hauteur ou ligne de vue (pas seulement « arme » ou
« objectif »), v1 et v2 confondus :

| Carte | Zone | Citation | Origine |
|---|---|---|---|
| Recharge | Attic / Maintenance Bay | « rain fire over everyone in Batteries, Bridge, Pit, Storage, and even up at Overhang » | v1 |
| Recharge | Platform | « good cross into whirlpool and batteries... look into attic » | v2 |
| Recharge | Control Room | « surplombe Pit et Batteries, excellente vue sur Attic » / tenue à 2 étages | v1+v2 |
| Recharge | Whirlpool Dam | « hauteur couvrant la plupart des entrées de Batteries » | v1 |
| Live Fire | Tower | « the high ground is a massive advantage » ; « little cover on the A&B side » | v1+v2 |
| Live Fire | Overlook | lignes de vue depuis/vers Tower | v1 |
| Live Fire | Back Green Ledge | embuscade en hauteur, accès restreint (saut précis) | v2 (zone `?`) |
| Aquarius | Top Mid | « top of the central wall that divides the map » | v1 |
| Aquarius | Blue/Yellow Utility | « limits angles you can be shot from » | v2 |
| Bazaar | Tower/Antenna/Cafe/Den/Palm Tree | « hauteur au-dessus de Market » (5 positions en couronne) | v1 |
| Solitude | Top Nest | « height advantage of Nest » | v2 |
| Solitude | Window | « can see a lot of the map » | v2 |
| Solitude | Ledge | « high ledge... powerful spot » | v2 |
| Empyrean | East/West Tower | « get on their tower and start spawn killing » | v2 |
| Catalyst | Top Mid | « long sightlines and high degree of verticality » | v1 |
| Catalyst | Overlook (résolu v2) | nom générique, non confirmé par citation dédiée | v2 |
| Forbidden | Dried/Overgrown Nest (résolu v2) | motif récurrent « Nest » = hauteur sniper | v2 |
| Argyle | East/West Rampart | « elevated walkway over a bottomless pit » | v1 |
| Banished Narrows | raised walkway | « raised walkway... vue sur tout le pont » | v1 |

Remarque pour 2bis.C : les positions les mieux attestées en hauteur/LOS explicite (Recharge
Attic/Control Room/Platform, Live Fire Tower, Bazaar la couronne autour de Market, Solitude
Nest/Window/Ledge) sont un bon jeu de calibrage pour vérifier qu'un score géométrique (H/V/E/R/M)
les retrouve bien — elles couvrent des géométries variées (tir plongeant, couloir défilé,
couronne symétrique, fenêtre ponctuelle).

---

## 6. TABLE FINALE PARSABLE

Format : `| carte_cle | zone_en | nom_guide | confiance | raison | v1v2 | sources |`

Règles de lecture identiques à v1 (§3) : une ligne `zone_en = ?` n'est exploitable ni au
rappel ni à la précision ; seules les lignes `forte` comptent au rappel ; `forte` + `faible`
(hors `?`) comptent à la précision.

| carte_cle | zone_en | nom_guide | confiance | raison | v1v2 | sources |
|---|---|---|---|---|---|---|
| ctf_aquarius | Top Mid | Top Mid | forte | hauteur + lignes de vue + arme | v1 | https://www.halopedia.org/Aquarius https://www.gamerguides.com/halo-infinite/guide/maps/arena/aquarius https://www.gfinityesports.com/halo-infinite/aquarius-callouts/ https://www.thegamer.com/halo-infinite-best-multiplayer-maps/ |
| ctf_aquarius | Hydro | Hydro | forte | arme + lignes de vue | v1 | https://www.gfinityesports.com/halo-infinite/aquarius-callouts/ https://www.gamerguides.com/halo-infinite/guide/maps/arena/aquarius |
| ctf_aquarius | Planters | Planters area | faible | arme | v1 | https://www.gamerguides.com/halo-infinite/guide/maps/arena/aquarius |
| ctf_aquarius | Yellow Base | Yellow Base | faible | objectif + spawn | v1 | https://www.gamerguides.com/halo-infinite/guide/maps/arena/aquarius |
| ctf_aquarius | Blue Base | Blue Base | faible | objectif + spawn | v1 | https://www.gamerguides.com/halo-infinite/guide/maps/arena/aquarius |
| ctf_aquarius | Blue Refrigeration | fridge | faible | objectif + acces (spawn-trap) | v2 | https://www.reddit.com/r/CompetitiveHalo/comments/s1ipiy/tips_and_advice_for_playing_aquarius/ |
| ctf_aquarius | Yellow Refrigeration | fridge | faible | objectif + acces (spawn-trap) | v2 | https://www.reddit.com/r/CompetitiveHalo/comments/s1ipiy/tips_and_advice_for_playing_aquarius/ |
| ctf_aquarius | Blue Utility | utility | faible | lignes de vue (route de drapeau) | v2 | https://www.reddit.com/r/CompetitiveHalo/comments/s1ipiy/tips_and_advice_for_playing_aquarius/ |
| ctf_aquarius | Yellow Utility | utility | faible | lignes de vue (route de drapeau) | v2 | https://www.reddit.com/r/CompetitiveHalo/comments/s1ipiy/tips_and_advice_for_playing_aquarius/ |
| sgh_interlock | Tower | Tower | forte | hauteur + lignes de vue + acces + arme | v1 | https://www.gfinityesports.com/halo-infinite/live-fire-callouts/ https://www.thegamer.com/camping-spots-in-halo-infinite/ https://gamerant.com/halo-infinite-best-weapon-spawn-live-fire/ https://www.reddit.com/r/CompetitiveHalo/comments/125yzui/live_fire_whats_a_good_setup_for_os_which_side_is/ |
| sgh_interlock | Hallway | Hallway | forte | arme | v1 | https://www.gfinityesports.com/halo-infinite/live-fire-callouts/ https://www.gamerguides.com/halo-infinite/guide/maps/arena/live-fire https://gamerant.com/halo-infinite-best-weapon-spawn-live-fire/ |
| sgh_interlock | Platform | cylindrical platform / Overshield platform / C plat | forte | arme + acces (position disputee) | v1 | https://www.gamerguides.com/halo-infinite/guide/maps/arena/live-fire https://gamerant.com/halo-infinite-best-weapon-spawn-live-fire/ https://www.reddit.com/r/CompetitiveHalo/comments/125yzui/live_fire_whats_a_good_setup_for_os_which_side_is/ |
| sgh_interlock | Overlook | Overlook | faible | lignes de vue + arme | v1 | https://www.gfinityesports.com/halo-infinite/live-fire-callouts/ |
| sgh_interlock | House Window | House Window | faible | objectif + acces | v1 | https://www.gamerguides.com/halo-infinite/guide/maps/arena/live-fire |
| sgh_interlock | House Elbow | House Elbow | faible | acces | v1 | https://www.gamerguides.com/halo-infinite/guide/maps/arena/live-fire |
| sgh_interlock | Tunnel | Tunnel | faible | acces | v1 | https://www.gamerguides.com/halo-infinite/guide/maps/arena/live-fire |
| sgh_interlock | House | House | faible | acces | v1 | https://www.gamerguides.com/halo-infinite/guide/maps/arena/live-fire |
| sgh_interlock | ? | Back Green Ledge (Green Bend ou Green Building, non tranche) | faible | hauteur + acces (embuscade) | v2 | https://www.reddit.com/r/CompetitiveHalo/comments/1mj1rfe/unexplored_live_fire_back_green_ledge_meta_can_be/ |
| sgh_blueprint | Attic | Attic | forte | hauteur + lignes de vue | v1 | https://www.gfinityesports.com/article/recharge-callouts https://dotesports.com/halo/news/halo-infinite-recharge-map-guide |
| sgh_blueprint | Maintenance Bay | Maintenance Bay | forte | hauteur + lignes de vue + acces | v1 | https://www.gfinityesports.com/article/recharge-callouts https://dotesports.com/halo/news/halo-infinite-recharge-map-guide |
| sgh_blueprint | Pit | Pit | forte | arme | v1 | https://www.gamerguides.com/halo-infinite/guide/maps/arena/recharge https://www.gfinityesports.com/article/recharge-callouts |
| sgh_blueprint | Hydro | Hydro | forte | arme + objectif + spawn | v1 | https://www.gamerguides.com/halo-infinite/guide/maps/arena/recharge https://www.gfinityesports.com/article/recharge-callouts https://dotesports.com/halo/news/halo-infinite-recharge-map-guide |
| sgh_blueprint | Whirlpool Dam | Whirlpool | forte | hauteur + lignes de vue + arme | v1 | https://dotesports.com/halo/news/halo-infinite-recharge-map-guide https://www.gfinityesports.com/article/recharge-callouts |
| sgh_blueprint | Orange Pipes | Orange Pipes | forte | acces + arme | v1 | https://dotesports.com/halo/news/halo-infinite-recharge-map-guide https://www.gfinityesports.com/article/recharge-callouts |
| sgh_blueprint | Platform | Hydro platform / C plat | forte | hauteur + lignes de vue | v1→forte | https://hyperxarenalasvegas.com/halo-infinite-season-six-best-loadouts-map-strategies/ https://www.reddit.com/r/CompetitiveHalo/comments/13v41ui/breaking_recharge_setups/ |
| sgh_blueprint | Control Room | Control Room's second floor / Control | forte | hauteur + lignes de vue + acces | v1→forte | https://dotesports.com/halo/news/halo-infinite-recharge-map-guide https://www.reddit.com/r/CompetitiveHalo/comments/13v41ui/breaking_recharge_setups/ |
| sgh_blueprint | Long Hall | Long Hall | forte | arme + acces | v1→forte | https://www.gfinityesports.com/article/recharge-callouts https://www.reddit.com/r/CompetitiveHalo/comments/vhrru2/help_with_power_positions/ https://www.reddit.com/r/CompetitiveHalo/comments/13v41ui/breaking_recharge_setups/ |
| sgh_blueprint | Elevator | top of Elevator / « A » | faible | hauteur + objectif + acces | v1 | https://dotesports.com/halo/news/halo-infinite-recharge-map-guide |
| sgh_blueprint | Batteries | Batteries | faible | objectif | v1 | https://dotesports.com/halo/news/halo-infinite-recharge-map-guide |
| sgh_blueprint | Overhang | Overhang | faible | arme | v1 | https://www.gfinityesports.com/article/recharge-callouts |
| sgh_blueprint | Blue Pipes | Blue Pipes | faible | arme + acces | v1 | https://dotesports.com/halo/news/halo-infinite-recharge-map-guide |
| sgh_streets | Main Street | middle of Main Street | forte | arme | v1 | https://www.gfinityesports.com/article/streets-callouts https://progameguides.com/halo/all-weapon-spawn-locations-on-streets-in-halo-infinite-multiplayer/ |
| sgh_streets | Subway Balcony | a balcony / Subway Balcony | forte | lignes de vue + arme | v1 | https://www.gamerguides.com/halo-infinite/guide/maps/arena/streets https://progameguides.com/halo/all-weapon-spawn-locations-on-streets-in-halo-infinite-multiplayer/ |
| sgh_streets | Cafe | Station Inside and Cafe | forte | arme + objectif + acces | v1→forte | https://progameguides.com/halo/all-weapon-spawn-locations-on-streets-in-halo-infinite-multiplayer/ https://www.reddit.com/r/CompetitiveHalo/comments/vhrru2/help_with_power_positions/ |
| sgh_streets | Subway | the Subway | faible | acces | v1 | https://www.gamerguides.com/halo-infinite/guide/maps/arena/streets |
| sgh_streets | Station Inside | Station Inside | faible | objectif + acces | v1 | https://www.gamerguides.com/halo-infinite/guide/maps/arena/streets |
| sgh_streets | Officer George's Desk | behind the desk | faible | objectif + acces | v1 | https://www.gamerguides.com/halo-infinite/guide/maps/arena/streets |
| sgh_streets | Old Town | Old Town | faible | objectif | v1 | https://www.gfinityesports.com/article/streets-callouts |
| sgh_streets | Arc Street | Arc Street | faible | arme | v1 | https://www.gfinityesports.com/article/streets-callouts |
| sgh_streets | Station Tower 2 | Station Tower 2 | faible | arme | v1 | https://progameguides.com/halo/all-weapon-spawn-locations-on-streets-in-halo-infinite-multiplayer/ |
| sgh_streets | Arc Street Bend | Arc Street Bend | faible | arme | v1 | https://progameguides.com/halo/all-weapon-spawn-locations-on-streets-in-halo-infinite-multiplayer/ |
| sgh_streets | Commercial District | Commercial District | faible | arme | v1 | https://progameguides.com/halo/all-weapon-spawn-locations-on-streets-in-halo-infinite-multiplayer/ |
| sgh_streets | Main Street Alley | Main Street Alley | faible | objectif | v1 | https://www.gamerguides.com/halo-infinite/guide/maps/arena/streets |
| sgh_streets | Plaza | the central plaza | faible | acces | v1 | https://hyperxarenalasvegas.com/halo-infinite-season-six-best-loadouts-map-strategies/ |
| ctf_bazaar | Market | Market | forte | objectif + arme | v1 | https://dotesports.com/halo/news/halo-infinite-bazaar-map-guide https://www.ggrecon.com/guides/halo-infinite-maps-list/ |
| ctf_bazaar | Cafe | Cafe | forte | hauteur + arme | v1→forte (correction d'audit : 2 sources deja presentes en v1, mal comptees faible) | https://dotesports.com/halo/news/halo-infinite-bazaar-map-guide https://www.gamerguides.com/halo-infinite/guide/maps/arena/bazaar |
| ctf_bazaar | Den | Den | forte | hauteur + arme | v1→forte (correction d'audit : idem Cafe) | https://dotesports.com/halo/news/halo-infinite-bazaar-map-guide https://www.gamerguides.com/halo-infinite/guide/maps/arena/bazaar |
| ctf_bazaar | Palm Tree | Palm Tree / palm | forte | hauteur + acces (tenue a 4) | v1→forte | https://dotesports.com/halo/news/halo-infinite-bazaar-map-guide https://www.reddit.com/r/CompetitiveHalo/comments/u37he4/how_the_hell_are_you_supposed_to_play_bazaar_ctf/ |
| ctf_bazaar | Tower | Tower / rocks | forte | hauteur + arme | v1→forte | https://dotesports.com/halo/news/halo-infinite-bazaar-map-guide https://www.reddit.com/r/CompetitiveHalo/comments/u37he4/how_the_hell_are_you_supposed_to_play_bazaar_ctf/ |
| ctf_bazaar | Antenna | Antenna | faible | hauteur | v1 | https://dotesports.com/halo/news/halo-infinite-bazaar-map-guide |
| ctf_bazaar | Truck | near Truck | faible | arme | v1 | https://dotesports.com/halo/news/halo-infinite-bazaar-map-guide |
| ctf_bazaar | East Market Bridge | East to West Market Bridge | faible | lignes de vue | v1 | https://www.gfinityesports.com/article/bazaar-callouts |
| ctf_bazaar | West Market Bridge | East to West Market Bridge | faible | lignes de vue | v1 | https://www.gfinityesports.com/article/bazaar-callouts |
| ctf_bazaar | Tower Basement | Tower Basement | faible | objectif + acces | v1 | https://www.gamerguides.com/halo-infinite/guide/maps/arena/bazaar |
| ctf_bazaar | West Base | West Base | faible | objectif + spawn | v1 | https://www.gamerguides.com/halo-infinite/guide/maps/arena/bazaar |
| ctf_bazaar | East Base | East Base | faible | objectif + spawn | v1 | https://www.gamerguides.com/halo-infinite/guide/maps/arena/bazaar |
| ctf_bazaar | Lockers | lockers | faible | acces (tenue a 4) | v2 | https://www.reddit.com/r/CompetitiveHalo/comments/u37he4/how_the_hell_are_you_supposed_to_play_bazaar_ctf/ |
| catalyst | Top Mid | the central area | faible | hauteur + lignes de vue | v1 | https://www.halopedia.org/Catalyst |
| catalyst | Middle Ramp | that long midway ramp | faible | arme | v1 | https://www.thegamer.com/halo-infinite-best-multiplayer-maps/ |
| catalyst | Overlook | a higher position / high ground | faible | hauteur | v2 (resolu depuis `?`) | https://gamerant.com/halo-infinite-lone-wolves-catalyst-map-tips/ |
| catalyst | Closed Hall | the closed-off areas on the side | faible | acces | v2 (resolu depuis `?`) | https://gamerant.com/halo-infinite-lone-wolves-catalyst-map-tips/ |
| ctf_forbidden | Dried Rat Hole | rat holes | forte | objectif + acces | v1 | https://kotaku.com/halo-infinite-season-5-maps-bandit-evo-forge-ai-1850908494 https://www.halowaypoint.com/news/maps-overview-season-5 |
| ctf_forbidden | Overgrown Rat Hole | rat holes | forte | objectif + acces | v1 | https://kotaku.com/halo-infinite-season-5-maps-bandit-evo-forge-ai-1850908494 https://www.halowaypoint.com/news/maps-overview-season-5 |
| ctf_forbidden | Center Bridge | the center of the map | faible | arme | v1 (retrograde par le pilote, valide en v2) | https://www.halopedia.org/Forbidden https://www.halowaypoint.com/news/maps-overview-season-5 |
| ctf_forbidden | Dried Mid | the middle of the map | faible | lignes de vue | v2 (resolu depuis `?`) | https://www.gamersdecide.com/articles/halo-infinite-best-maps |
| ctf_forbidden | Overgrown Mid | the middle of the map | faible | lignes de vue | v2 (resolu depuis `?`) | https://www.gamersdecide.com/articles/halo-infinite-best-maps |
| ctf_forbidden | Dried Nest | snipers on either side | faible | arme (reserve : peut-etre Dried Back Nest) | v2 (resolu depuis `?`) | https://www.halowaypoint.com/news/maps-overview-season-5 |
| ctf_forbidden | Overgrown Nest | snipers on either side | faible | arme (reserve : peut-etre Overgrown Back Nest) | v2 (resolu depuis `?`) | https://www.halowaypoint.com/news/maps-overview-season-5 |
| dd600260-d91c-4d77-9990-3f35873c90a1 | East Rampart | high-up battlement | faible | hauteur + arme | v1 | https://www.halopedia.org/index.php?title=Argyle&action=raw |
| dd600260-d91c-4d77-9990-3f35873c90a1 | West Rampart | high-up battlement | faible | hauteur + arme | v1 | https://www.halopedia.org/index.php?title=Argyle&action=raw |
| dd600260-d91c-4d77-9990-3f35873c90a1 | Platform | floating platform in the middle | faible | arme | v1 | https://www.halopedia.org/index.php?title=Argyle&action=raw |
| dd600260-d91c-4d77-9990-3f35873c90a1 | ? | an easy-to-access tower near each spawn | faible | hauteur + arme + acces | v1 | https://www.thegamer.com/camping-spots-in-halo-infinite/ |
| dd600260-d91c-4d77-9990-3f35873c90a1 | ? | elevated walkway over a bottomless pit | faible | hauteur + arme | v1 | https://www.halopedia.org/index.php?title=Argyle&action=raw |
| dd600260-d91c-4d77-9990-3f35873c90a1 | ? | flanking hallways | faible | lignes de vue + acces | v1 | https://www.halopedia.org/index.php?title=Argyle&action=raw |
| dd600260-d91c-4d77-9990-3f35873c90a1 | ? | snipe elbow (coaching Warlord) | faible | hauteur + arme (spawntrap) | v2 | https://www.reddit.com/r/CompetitiveHalo/comments/1u9v11j/give_me_tips_for_argyle_and_catalyst/ |
| dd600260-d91c-4d77-9990-3f35873c90a1 | ? | vent / vent room (coaching Warlord) | faible | acces + objectif (progression de controle) | v2 | https://www.reddit.com/r/CompetitiveHalo/comments/1u9v11j/give_me_tips_for_argyle_and_catalyst/ |
| dd600260-d91c-4d77-9990-3f35873c90a1 | ? | pistols (spawn de pistolet, coaching Warlord) | faible | objectif (etape de controle de carte) | v2 | https://www.reddit.com/r/CompetitiveHalo/comments/1u9v11j/give_me_tips_for_argyle_and_catalyst/ |
| d035fc3e-f298-4c14-9487-465be2e1dc1f | East Tower | Sniper Tower / tower | forte | hauteur + arme + lignes de vue | v1→forte | https://www.halopedia.org/index.php?title=The_Pit&action=raw https://www.halopedia.org/index.php?title=Empyrean&action=raw https://www.reddit.com/r/CompetitiveHalo/comments/106ojq6/tips_for_the_pit_aka_empyrean/ |
| d035fc3e-f298-4c14-9487-465be2e1dc1f | West Tower | Sniper Tower / tower | forte | hauteur + arme + lignes de vue | v1→forte | https://www.halopedia.org/index.php?title=The_Pit&action=raw https://www.halopedia.org/index.php?title=Empyrean&action=raw https://www.reddit.com/r/CompetitiveHalo/comments/106ojq6/tips_for_the_pit_aka_empyrean/ |
| d035fc3e-f298-4c14-9487-465be2e1dc1f | East Balcony | a new balcony | faible | arme | v1 | https://www.halopedia.org/index.php?title=Empyrean&action=raw |
| d035fc3e-f298-4c14-9487-465be2e1dc1f | West Balcony | a new balcony | faible | arme | v1 | https://www.halopedia.org/index.php?title=Empyrean&action=raw |
| d035fc3e-f298-4c14-9487-465be2e1dc1f | ? | middle corridor / long hall / rocket hall | faible | arme | v1 | https://www.halopedia.org/Empyrean |
| d035fc3e-f298-4c14-9487-465be2e1dc1f | ? | Sword Room / Blue Room / Control Room (reserve : « sword » toujours cite en 2024-2025) | faible | arme | v1 | https://www.halopedia.org/index.php?title=The_Pit&action=raw |
| d035fc3e-f298-4c14-9487-465be2e1dc1f | ? | courtyard (spawn-trap cible) | faible | acces (objectif de rotation) | v2 | https://www.reddit.com/r/CompetitiveHalo/comments/106ojq6/tips_for_the_pit_aka_empyrean/ |
| f1cc3b4e-471c-4ec5-b855-1db7d9e6ce42 | Top Nest | sniper nest / snipe / Nest | forte | hauteur + arme | v1→forte | https://www.gamedeveloper.com/design/level-design-in-halo-5-multiplayer-plaza https://www.reddit.com/r/CompetitiveHalo/comments/14mz2al/tips_for_playing_solitude/ |
| f1cc3b4e-471c-4ec5-b855-1db7d9e6ce42 | Bridge | central walkway above the street / os bridge | forte | hauteur + arme | v1→forte | https://www.halopedia.org/Plaza https://www.reddit.com/r/CompetitiveHalo/comments/14mz2al/tips_for_playing_solitude/ |
| f1cc3b4e-471c-4ec5-b855-1db7d9e6ce42 | Window | the little window over A | faible | lignes de vue | v2 | https://www.reddit.com/r/CompetitiveHalo/comments/14mz2al/tips_for_playing_solitude/ |
| f1cc3b4e-471c-4ec5-b855-1db7d9e6ce42 | Pillar | the pillar in B | faible | acces (mouvement autour du pilier) | v2 | https://www.reddit.com/r/CompetitiveHalo/comments/14mz2al/tips_for_playing_solitude/ |
| f1cc3b4e-471c-4ec5-b855-1db7d9e6ce42 | Ledge | the high ledge at A | faible | hauteur | v2 | https://www.reddit.com/r/CompetitiveHalo/comments/14mz2al/tips_for_playing_solitude/ |
| f1cc3b4e-471c-4ec5-b855-1db7d9e6ce42 | ? | grey areas / grey 1-2-3 (rotation) | faible | acces (rotation de zone) | v2 | https://www.reddit.com/r/CompetitiveHalo/comments/14mz2al/tips_for_playing_solitude/ |
| f1cc3b4e-471c-4ec5-b855-1db7d9e6ce42 | ? | quantum (translocator equipment) | faible | arme (equipement) | v2 | https://www.reddit.com/r/CompetitiveHalo/comments/14mz2al/tips_for_playing_solitude/ |
| 654dff62-d618-496a-8914-06ab73d991e3 | Tower | Tower | faible | hauteur + lignes de vue | v1 | https://thegamehaus.com/esports/the-playbook-rig-slayer/2017/03/18/ |
| 654dff62-d618-496a-8914-06ab73d991e3 | Long Hall | Long Hall | faible | lignes de vue | v1 | https://thegamehaus.com/esports/the-playbook-rig-slayer/2017/03/18/ |
| 654dff62-d618-496a-8914-06ab73d991e3 | Bunker | Bunker spawn | faible | spawn + acces | v1 (vocabulaire confirme en v2) | https://thegamehaus.com/esports/the-playbook-rig-slayer/2017/03/18/ https://www.reddit.com/r/CompetitiveHalo/comments/1ceaqne/topdown_callouts_strongholds_and_hills_for/ |
| 654dff62-d618-496a-8914-06ab73d991e3 | ? | Nest | faible | arme | v1 | https://thegamehaus.com/esports/the-playbook-rig-slayer/2017/03/18/ |
| 654dff62-d618-496a-8914-06ab73d991e3 | ? | Strongholds locations | faible | lignes de vue | v1 | https://www.halowaypoint.com/news/interference-update |
| 9ad226d8-8947-4c5b-95bc-d220187698c1 | Top Mid | main bridge / upper bridge | faible | arme + lignes de vue | v1 | https://www.halopedia.org/Narrows |
| 9ad226d8-8947-4c5b-95bc-d220187698c1 | ? | raised walkway | faible | hauteur + lignes de vue | v1 | https://www.halopedia.org/Narrows |
| 9ad226d8-8947-4c5b-95bc-d220187698c1 | ? | back center | faible | arme + lignes de vue | v1 | https://www.halopedia.org/Narrows |

**Totaux v2** : 96 lignes exploitables ou tracées (84 v1 + 12 nouvelles, dont 2 résolues
depuis `?` sur Forbidden et 2 depuis `?` sur Catalyst comptées comme lignes existantes
recodées, pas neuves). **28 positions `forte`** (compté sur pièces dans le tableau ci-dessus ;
**16 en v1** post-relecture pilote — le total « 17 » annoncé en tête de v1 §3 n'a pas été mis
à jour après que le pilote a rétrogradé Forbidden `Center Bridge` en §7 de v1, un écart déjà
présent dans v1, non introduit ici → **+12** : Platform/Recharge, Control Room/Recharge, Long
Hall/Recharge, Cafe/Streets, Cafe/Bazaar, Den/Bazaar, Palm Tree/Bazaar, Tower/Bazaar, East
Tower/Empyrean, West Tower/Empyrean, Top Nest/Solitude, Bridge/Solitude — 12 promotions au
total, dont 2 par correction d'audit) réparties sur
**8 cartes avec au moins 1 forte** (Aquarius, Live Fire, Recharge, Streets, Bazaar, Forbidden,
Empyrean, Solitude).

---

## 7. Table des contre-exemples (v1 + nouveaux)

| carte_cle | zone_en | nom_guide | confiance | raison | v1v2 | sources |
|---|---|---|---|---|---|---|
| ctf_aquarius | Yellow Base | near the flag | faible | statique pres du drapeau : tire depuis la base adverse | v1 | https://www.gamerguides.com/halo-infinite/guide/maps/arena/aquarius |
| ctf_aquarius | Blue Base | near the flag | faible | statique pres du drapeau : tire depuis la base adverse | v1 | https://www.gamerguides.com/halo-infinite/guide/maps/arena/aquarius |
| ctf_aquarius | Blue Courtyard / Yellow Courtyard | courtyard | faible | mortel si l'ennemi tient Planters (« death trap ») | v2 | https://www.reddit.com/r/CompetitiveHalo/comments/1atc927/how_to_pull_flag_through_car_1_on_aquarius_ctf/ |
| sgh_interlock | Canal | the Canal | faible | chute : « Don't fall in the Canal! » | v1 | https://www.gfinityesports.com/halo-infinite/live-fire-callouts/ |
| sgh_blueprint | Batteries | Batteries | faible | cible du tir plongeant d'Attic / Maintenance Bay | v1 | https://www.gfinityesports.com/article/recharge-callouts |
| sgh_blueprint | Bridge | Bridge | faible | cible du tir plongeant d'Attic / Maintenance Bay | v1 | https://www.gfinityesports.com/article/recharge-callouts |
| sgh_blueprint | Pit | Pit | faible | cible du tir plongeant d'Attic / Maintenance Bay | v1 | https://www.gfinityesports.com/article/recharge-callouts |
| sgh_blueprint | Storage | Storage | faible | cible du tir plongeant d'Attic / Maintenance Bay | v1 | https://www.gfinityesports.com/article/recharge-callouts |
| sgh_blueprint | Overhang | Overhang | faible | cible du tir plongeant d'Attic / Maintenance Bay | v1 | https://www.gfinityesports.com/article/recharge-callouts |
| sgh_blueprint | Long Hall | long hallways | faible | camping d'arme forte, positionnement a risque | v1 (statut de piege partiel maintenu malgre la promotion en position forte — la zone est les deux a la fois, comme Main Street) | https://hyperxarenalasvegas.com/halo-infinite-season-six-best-loadouts-map-strategies/ |
| sgh_streets | Main Street | center of the map | faible | expose sur plusieurs axes : « Areas to Avoid » | v1 (confirme en v2, §2.2) | https://www.gamerguides.com/halo-infinite/guide/maps/arena/streets |
| sgh_streets | Old Town Bar | Old Town Bar | faible | espace tres clos, piege a grenades ; flanqueurs d'Arc Street Hallway | v1 | https://www.gfinityesports.com/article/streets-callouts |
| ctf_bazaar | Market | Market | faible | centre expose de toutes parts | v1 | https://dotesports.com/halo/news/halo-infinite-bazaar-map-guide |
| ctf_bazaar | Tower Basement | Tower Basement | faible | isole : le porteur y est arrete sans secours possible | v1 | https://dotesports.com/halo/news/halo-infinite-bazaar-map-guide |
| catalyst | Bottom Mid | the center of the map / the light bridge | faible | aucun couvert, snipe depuis la hauteur ; « avoid this place outright » | v1 | https://gamerant.com/halo-infinite-lone-wolves-catalyst-map-tips/ |
| dd600260-d91c-4d77-9990-3f35873c90a1 | ? | long, wide-open hallway between the bases | faible | ligne de vue mutuelle base a base | v1 | https://www.halopedia.org/index.php?title=Argyle&action=raw |
| dd600260-d91c-4d77-9990-3f35873c90a1 | Platform | the central area | faible | ouvert et plus bas que le reste de la carte | v1 | https://www.halopedia.org/index.php?title=Argyle&action=raw |
| d035fc3e-f298-4c14-9487-465be2e1dc1f | East Tower / West Tower | Sniper Tower | faible | tuable a travers les interstices des plaques metalliques ; sur-tenir la tour ennemie force le split spawn adverse | v1 (nuance ajoutee en v2) | https://www.halopedia.org/The_Pit https://www.reddit.com/r/CompetitiveHalo/comments/106ojq6/tips_for_the_pit_aka_empyrean/ |
| d035fc3e-f298-4c14-9487-465be2e1dc1f | ? | rockets (couloir central) | faible | prise en debut de manche tres risquee | v1 | https://www.halopedia.org/index.php?title=The_Pit&action=raw |
| d035fc3e-f298-4c14-9487-465be2e1dc1f | ? | Sword Room | faible | couloir courbe exigu, couvert limite, souvent non conteste | v1 | https://www.halopedia.org/index.php?title=The_Pit&action=raw |
| d035fc3e-f298-4c14-9487-465be2e1dc1f | ? | the lower side of the map | faible | bord desormais ouvert : chute mortelle | v1 | https://www.halopedia.org/index.php?title=Empyrean&action=raw |
| f1cc3b4e-471c-4ec5-b855-1db7d9e6ce42 | Bridge | Overshield | faible | activation differee : on meurt avant que le bouclier monte | v1 | https://www.gamedeveloper.com/design/level-design-in-halo-5-multiplayer-plaza |
| f1cc3b4e-471c-4ec5-b855-1db7d9e6ce42 | Bottom Mid | that middle area | faible | tres ouvert : tenu a distance par un Stalker Rifle | v1 | https://www.halowaypoint.com/news/solitude-preview-season-4 |
| 654dff62-d618-496a-8914-06ab73d991e3 | ? | Nest | faible | peu de couvert, vulnerable aux grenades | v1 | https://thegamehaus.com/esports/the-playbook-rig-slayer/2017/03/18/ |
| 9ad226d8-8947-4c5b-95bc-d220187698c1 | Top Mid | upper bridge | faible | vaste : on y est touche de plusieurs cotes a la fois | v1 | https://www.halopedia.org/Narrows |
| 0d1c9255-d912-416c-befc-5f3e5e176df2 | ? (aucune zone dans le depot) | large open space / Ring / Mid | faible | « melted from every angle » — le centre entier est le piege, pas une zone en particulier | v2 | https://www.reddit.com/r/CompetitiveHalo/comments/1gn1b4b/new_fortress_map_first_opinions_in_ranked/ |

Note : `ctf_aquarius Top Nest / Yellow Base` de v1 §3.1 était déjà repris ci-dessus tel quel ;
seule la ligne `Blue/Yellow Courtyard` et la ligne `Fortress` sont nouvelles (`v2`).

---

## 8. Verdict du gate

Le gate demande **≥ 6 cartes à ≥ 3 fortes**, ou à défaut la démonstration chiffrée de
l'absence de sources.

**Cartes à ≥ 3 fortes après v2** : Recharge (**9**), Live Fire (**3**), Streets (**3**),
Bazaar (**5**) — **4 cartes**, pas 6.

**Cartes proches, densifiées mais sous le seuil**, avec le chiffrage de l'effort :
- Aquarius : 2 fortes (inchangé), 2 nouvelles positions faibles (Refrigeration, Utility) +
  1 nouveau contre-exemple, malgré 3 recherches Reddit dédiées.
- Forbidden : 2 fortes (inchangé), 2 lignes `?` résolues en zones réelles, malgré 4 fils lus.
- Solitude : **0 → 2 fortes** (progression la plus nette du chantier), 3 nouvelles positions
  faibles (Window, Pillar, Ledge), malgré recherche dédiée pas de 3e forte trouvée.
- Empyrean : **0 → 2 fortes**, 1 contradiction documentée (sword), malgré 4 fils lus.
- Catalyst : 0 forte (inchangé), 2 lignes `?` résolues, malgré recherche dédiée — carte
  confirmée la plus pauvre en sources ouvertes du pool HCS.
- Fortress, Origin, Banished Narrows : 0 zone officielle dans le dépôt, recherches dédiées
  négatives ou hors-sujet documentées (§2.12) — **hors de portée structurelle**, pas un échec
  de recherche.
- Lattice, Argyle, Interference : contenu riche trouvé, **non exploitable** faute de
  correspondance nom-communautaire ↔ nom-du-catalogue (§4).

**Verdict** : le gate n'est pas atteint au sens strict (4 cartes, pas 6), mais la seconde
branche du gate (« démonstration chiffrée que les sources n'existent pas ») est servie pour
les 8 cartes restantes : soit le dépôt n'a aucune zone officielle (Fortress, Origin, Banished
Narrows — 3 cartes, chiffré §2.12), soit la recherche ciblée est allée au bout et n'a
trouvé qu'un renforcement partiel (Aquarius, Forbidden, Catalyst — 3 cartes, sous le seuil
malgré effort documenté), soit la connaissance existe mais le catalogue de zones ne la
couvre pas (Lattice, Argyle, Interference — 3 cartes, angle mort de vocabulaire documenté,
pas de source). Le pilote tranche : accepter ce résultat en l'état, ou prescrire une passe
supplémentaire ciblée sur 1-2 cartes précises (Aquarius et Forbidden sont les plus proches du
seuil).

---

## 9. Limites

1. **Liquipedia et halo.fandom.com restent inaccessibles même au navigateur réel** — ce n'est
   pas un défaut de WebFetch spécifiquement, c'est un défi anti-bot (Cloudflare) qui bloque
   toute automatisation, confirmé par `wait_for` en échec après 8 s. x.com est un mur de
   connexion, pas un blocage anti-bot — accessible seulement avec un compte.
2. **YouTube reste sourd** : les descriptions et chapitres se lisent, mais la quasi-totalité
   des vidéos trouvées par mots-clés sont des tutoriels de callouts bas de gamme (2021-2022,
   quelques centaines de vues), pas des analyses HCS. Les transcriptions/sous-titres sont
   confirmés inaccessibles (HTTP 200, corps vide) — un point que v1 supposait sans le
   vérifier au niveau réseau.
3. **Un thread Reddit = une source, quel que soit son nombre d'intervenants.** Discipline
   volontairement conservatrice : la richesse (33, 45, 77 commentaires) ne compte pas comme
   plusieurs sources indépendantes. Une lecture moins stricte aurait pu pousser 2-3 cartes de
   plus au-dessus du seuil de 3 fortes (notamment Solitude et Empyrean, où plusieurs
   intervenants distincts convergent dans le même fil).
4. **Aucune vérification de l'identité réelle des comptes Reddit cités** (KingJayHCS,
   TiberiusAudley, etc.) au-delà de leur cohérence avec des sources tierces déjà trouvées
   (gamecoach.gg pour TiberiusAudley/Warlord). Traité comme un faisceau d'indices, pas une
   preuve d'identité.
5. **La contradiction « sword » sur Empyrean (§2.9, §4)** n'est pas résolue et ne doit pas
   l'être par ce chantier — elle est un signal pour une vérification directe (patch notes ou
   observation en jeu), hors du périmètre « recherche documentaire ».
6. **Cette densification ne mesure toujours rien sur nos matchs.** Elle reste, comme v1, un
   oracle de VALIDATION pour 2bis.D (verdict v2), jamais une entrée du score empirique ou
   géométrique (rappel de D9).

## 10. Relecture du pilote (2026-09-20) — gate 2bis.A ACCEPTÉ avec réserves

- Le gate strict (≥ 3 fortes sur ≥ 6 cartes) n'est pas atteint : 4 cartes (Recharge 9, Bazaar 5,
  Live Fire 3, Streets 3). La clause alternative est servie et chiffrée (§8). Le pilote accepte :
  28 positions fortes contre 16 en v1, et la contrainte réelle est le VOCABULAIRE du dépôt
  (Lattice, Argyle, Interference, Empyrean, Solitude), pas l'absence de savoir pro.
- Conséquence pour 2bis.D, décidée maintenant : le verdict v2 se rend sur les cartes à ≥ 2
  positions fortes. Calibrage = Recharge (9), Aquarius (2), Streets (3). Validation = Live Fire
  (3, positions de kills à exclure ou corriger tant que la recuisson n'est pas jouée ; la
  géométrie, elle, n'est pas touchée), Bazaar (5), Forbidden (2), Empyrean (2), Solitude (2).
  Sur les cartes à 2 fortes, un rappel de 0,5 se lit « indéterminé » (règle v1 §7 maintenue).
- Les promotions par correction d'audit (Bazaar Cafe, Den) sont acceptées : les deux sources
  étaient déjà dans la v1. Les promotions par patron de tenue à 4 (Palm Tree, Tower) sont
  acceptées avec la réserve qu'un patron de tenue décrit une RÉPARTITION d'équipe, pas
  nécessairement quatre positions de force ; elles comptent au rappel mais un faux négatif sur
  l'une d'elles ne pèse pas comme un faux négatif sur Attic ou Tower.
- Contradiction Empyrean (épée retirée ou non) : sans effet sur l'oracle (les tours sont
  attestées par la hauteur et la ligne de vue, pas par l'arme). Non tranchée, sans conséquence.
- Ce que la v2 confirme sur le fond : les positions fortes attestées portent presque toutes
  « hauteur » ou « lignes de vue » (§5) — c'est le signal que le v1 empirique pesait le moins
  (0,15) et que la voie géométrique mesure directement. Le verdict v2 doit rapporter à part
  le rappel sur les positions « hauteur / lignes de vue » et sur les positions « arme /
  objectif ».
