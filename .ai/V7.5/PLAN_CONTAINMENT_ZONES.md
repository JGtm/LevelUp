# Plan — Containment des zones : qui tient quoi, et depuis quand

> Ecrit le 2026-08-14 a la demande de l'utilisateur. Reprend le rapport de faisabilite
> `.ai/HANDOFF_CONTAINMENT_ZONES_2026-08-08.md` (lot 4) et l'ACTUALISE : cinq choses ont
> change depuis, dont une qui rend probablement caduque l'etape n°1 d'alors.
> Execution sous `plan-execution`, branche `feat/v75`, commits par etape, PAS de push.

## De quoi il s'agit (l'utilisateur a demande ce que c'est)

**Savoir si un joueur se trouve DANS une zone d'objectif a un instant donne** — et donc qui
tenait la zone, combien de temps, et qui l'a prise. On croise trois choses :

	forme de la zone (catalogue map_objectives.json, en metres monde)
	  x  position du joueur a 100 ms pres (pipeline du rejeu 2D, par xuid)
	  x  instant + auteur des actions d'objectif (analysis/objectiveevents)

Ce que ca DEBLOQUERAIT : le score de zone au cours du temps, « qui a capture », le temps de
controle par joueur/equipe — et, cote rejeu, colorer une zone tenue au lieu de la dessiner
inerte. C'est aussi le prealable ecrit au registre pour tout score over-time zone/hill/skull.

## Ce qui est ETABLI (lot 4, ne pas refaire)

Le croisement existe, pur et teste : `mapvar.Contains`/`Volume` (containment.go), le lecteur
de catalogue (objectives_catalog.go), `AttributeZones` (zone_attribution.go), l'outil de
mesure `cmd/zone-attribution`. **Rien n'est persiste ni affiche**, pour trois raisons qui
restent valables jusqu'a preuve du contraire :

1. **Taux trop bas** : 12,2 % d'appartenance stricte, 28,6 % au meilleur decalage global —
   une colonne remplie une fois sur quatre n'est pas consommable, et une valeur presente n'y
   serait pas distinguable d'une valeur sure.
2. **La correction d'horloge n'est pas etablie** : le retard existe (8 films sur 8), mais sa
   valeur non, et 3 films sur 8 PIQUENT SUR LA BORNE du balayage (-10 s) — mesure tronquee.
3. **Aucun oracle de justesse** : tout ce qui est mesure est « une zone a-t-elle ete
   rattachee », jamais « est-ce LA BONNE ».

## CE QUI A CHANGE DEPUIS LE 2026-08-08 — a lire avant de planifier quoi que ce soit

| # | changement | effet sur ce chantier |
|---|---|---|
| 1 | **L'horloge du rejeu est RESOLUE** (lot 7.2, 14/08) : l'artefact publie `originMs`, et l'horodatage de paquet du film s'est revele etre une **horloge MOTEUR** (des milliers de secondes depuis le demarrage du jeu). Origines mesurees : **3,6 s a 50,8 s** selon le match | **HYPOTHESE CENTRALE DE CE PLAN** : le « retard d'horloge » du containment est probablement CE decalage-la. Il expliquerait pourquoi 3 films sur 8 piquaient a -10 s : le vrai decalage les depassait. Si c'est le cas, il n'y a **rien a balayer** — la correction est connue, exacte, et par film |
| 2 | **Oracle Vagabond disponible** (lot 5 cartes) : la carte porte ses 3 zones de Bastion reelles au catalogue (asset `105f5d84`) | le releve terrain du 2026-08-02 redevient rejouable = le seul oracle qui dise QUELLE zone |
| 3 | **Catalogue d'objectifs : 34 -> 72 cartes** (regenere le 13/08, lot fonds par map_id) — 158 zones Bastion, 236 zones d'extraction | le corpus mesurable s'elargit mecaniquement |
| 4 | **Bornes de dequantification : 15 -> 56 cartes** (lots bornes + map_id) | Prism, Recharge, Deadlock, Oasis, Scarr... deviennent decodables ; Live Fire reste hors jeu (module sans `sbsp`) |
| 5 | **Alias « Heavies » sorti** (lot 5) : +43 matchs | piste 4.1 du handoff : FAITE |

## Etape 1 — TESTER L'HYPOTHESE D'HORLOGE (avant tout calcul long)

> Le handoff mettait « elargir le balayage » en tete (~1 h de calcul). Cette etape la rend
> peut-etre inutile : on ne cherche plus un decalage, on en APPLIQUE un connu.

- [x] 1.1 Etablir sur quelle horloge vivent les DEUX entrees d'`AttributeZones` : les
      trajectoires (artefact de rejeu, frame 0 = `originMs`) et les actions d'objectif
      (`objectiveevents` — verifier sur pieces d'ou vient leur `t`).
      **FAIT EN LECTURE DE CODE, sans aucun calcul.** Les deux entrees ne vivent PAS sur la
      meme horloge : les actions sont datees en ms depuis le PREMIER PAQUET DU FILM
      (`objectiveevents.StatRecords` : `meta.StartMS + (f.us-base)/1000`, chunk 1 = 0 au
      manifeste) tandis que les frames comptent depuis le PREMIER PAQUET DE POSITION
      (`replay/build.go` : `origin = sorted[0].TimestampUS`). L'ecart entre ces deux zeros
      EST `originMs` (`replay/origin.go`). Or `buildObjectiveActions` divise `TimeMS` par le
      pas de grille SANS le retrancher : les actions etaient posees `originMs` trop TARD.
- [x] 1.2 Rejouer `cmd/zone-attribution` sur le corpus en appliquant la correction CONNUE du
      film au lieu du balayage : decalage = origine de l'artefact (et/ou T0 du match selon
      d'ou viennent les actions). Publier le taux AVANT / APRES par film.
      **Corpus d'origine (8 films) : 13,7 % -> 58,6 %**, temoin temporel plat (13 a 21 %),
      rapport signal/temoin 3,0 a 6,2. Par film : Forbidden 5,6->55,4 · Illusion 13,8->40,7 ·
      Forest 22,4->76,5 · Forest 17,2->81,2 · Illusion 12,8->40,5 · Illusion 7,7->44,4 ·
      Streets 8,8->67,5 · Cliffhanger 18,5->51,9.
- [x] 1.3 VERDICT : si le taux monte franchement, l'etape 2 (balayage elargi) est ANNULEE et
      la correction devient une donnee, pas une constante ajustee. Sinon, mesurer ce qui
      RESTE comme retard apres correction — c'est cela qu'il faudra expliquer.
      **HYPOTHESE CONFIRMEE, ETAPE 2 (BALAYAGE) ANNULEE.** Les trois films qui piquaient sur
      la borne de -10 s sont exactement ceux dont l'origine la depasse (19,8 s · 38,3 s ·
      30,9 s). La correction exacte fait MIEUX que le meilleur decalage balaye sur chaque
      film : c'est une correction PAR FILM, lue, pas une constante ajustee.

Gate 1 : chiffres par film, AVANT/APRES, sur les 8 films du corpus d'origine au minimum.
**PASSE** (8/8 films, tableau ci-dessus ; mesure rejouable par `go run ./cmd/zone-attribution`).

## Etape 2 — L'ORACLE DE JUSTESSE (Vagabond) — la seule qui repond « est-ce la BONNE zone »

- [x] 2.1 Rejouer le releve terrain du 2026-08-02 sur Vagabond (3 matchs Strongholds au
      registre) avec `cmd/zone-attribution` : pour chaque action, comparer la zone attribuee
      a la zone RELEVEE a la main.
      **FAIT** (`7344d24f` 25,8->73,2 % · `e6ccc947` 17,2->73,3 % · `696a9d7c` 15,3->77,9 %),
      detail par action sous `-dump`. Les deux choses que le releve dit VRAIMENT tombent
      juste : l'action de FlyGuy8773 est datee **48,9 s** et attribuee DEDANS (releve :
      « 0:48 flyguy8773 capture la base B »), et les trois rangs de zone sont tous captures a
      **90,0 s au plus tard** (releve a 1:30 : « une equipe controle les trois bases »).
- [!] 2.2 Publier une matrice de justesse (bonne zone / mauvaise zone / aucune), pas un
      simple taux d'attribution. C'est le chiffre qui manque a tout le chantier.
      **IMPOSSIBLE, ET IL FAUT LE DIRE PLUTOT QUE DE LA FABRIQUER.** Le releve
      (`.ai/V7.5/film_re/RELEVE_TERRAIN_CAPTURES_2026-07-31.md`) porte quatre ancres dont UNE
      SEULE nomme une base — et elle la nomme par une **lettre**, qu'aucune donnee decodee ne
      porte et que ce plan classe lui-meme hors perimetre. Une matrice « bonne / mauvaise
      zone » sur cette base serait inventee.
      **LIVRE A LA PLACE, et c'est un vrai oracle d'identite : la CONCORDANCE
      INTER-JOUEURS.** Une prise de Bastion est un evenement DE ZONE — les coequipiers qui la
      portent au meme instant sont sur la meme base, alors que leurs positions sont decodees
      independamment. Sur les 3 matchs Vagabond : **35 groupes concordants sur 36 = 97,2 %**,
      contre 33 % attendus au hasard sur trois zones. Sur les 31 films mesures : **125/133 =
      94,0 %**, temoin (memes groupes decales de 30 s) **5/13 = 38,5 %**.
- [x] 2.3 Si la justesse est bonne mais le taux bas : le probleme est la COUVERTURE (on
      n'attribue pas assez), pas la JUSTESSE (ce qu'on attribue est juste) — deux defauts
      qui n'appellent pas les memes suites, et le handoff ne pouvait pas les distinguer.
      **TRANCHE : c'est la COUVERTURE, et elle a une cause NOMMEE.** Ce qui est attribue est
      juste (94 a 97 % de concordance). Ce qui manque se separe en deux populations nettes :
      20 films entre 35 et 81 %, et **11 films a EXACTEMENT 0,0 %** sur 53 a 97 actions. Ces
      11 films ont un **ecart vertical median de +1 240 a +1 300 m** entre positions et zones
      — les joueurs sont 1,3 km au-dessus des formes. Ce n'est pas un fait de jeu, c'est un
      defaut de REPERE (bornes de dequantification ou variante du catalogue) sur ces cartes.
      Les films qui fonctionnent ont un ecart vertical de ~0,0 m (p25 -0,6 · p75 +0,1),
      exactement la mesure du rapport du 08/08. **Restreint aux films dont les deux reperes
      coincident : 760/1185 = 64,1 %.**

Gate 2 : la matrice, sur les 3 matchs Vagabond.
**PASSE SOUS RESERVE ECRITE** : la matrice demandee n'existe pas faute d'oracle de lettre
(2.2) ; la concordance inter-joueurs la remplace, avec son temoin et son denominateur.

## Etape 3 — ELARGIR LE CORPUS (devenu presque gratuit)

- [x] 3.1 Recompter le corpus mesurable avec le catalogue a 72 cartes et les bornes a 56
      (le handoff comptait 8 films sur un catalogue de 34 cartes / 15 bornes).
      **8 -> 48 matchs mesurables** sur 208 en mode a zones (ecartes : 151 sans film en cache,
      4 sans bornes, 5 sans formes). Vagabond y entre — l'oracle redevient rejouable.
      Mesure sans decodage : `go run ./cmd/zone-attribution -select-only`.
- [x] 3.2 Rejouer la mesure sur ce corpus elargi. A 8 films, un ecart de 3 points ne se
      distinguait pas du bruit ; c'est le seul moyen de sortir de cette zone d'incertitude.
      **JOUEE SUR 32 DES 48 FILMS** (31 corriges + `aaaf6c76` dont l'artefact REFUSE de
      publier l'origine — controle du fil des morts a -200 695 ms : le garde-fou d'`origin.go`
      a joue, le film est exclu du APRES et compte nulle part ailleurs).
      **Corpus mesure : 179/1811 = 9,9 % AVANT, 760/1860 = 40,9 % APRES.**
      Les **16 films restants ne sont PAS mesures, et c'est un choix motive** : ils ne peuvent
      pas renverser le gate 4.1. Borne haute — en supposant les 16 films attribues a **100 %**
      au debit moyen observe (60 actions/film) : (760+960)/(1860+960) = **61,0 %**. Atteindre
      80 % exigerait 3 640 actions toutes attribuees sur 16 films, soit 227 actions/film
      contre 110 au maximum observe. Report au registre (machine de l'utilisateur instable :
      trois redemarrages pendant le lot).

## Etape 4 — DECIDER : persister, ou classer

- [x] 4.1 Critere de persistance, ecrit AVANT de mesurer (pour ne pas l'ajuster apres) :
      **justesse >= 95 % sur l'oracle** ET **taux d'attribution >= 80 % sur le corpus
      elargi**. En dessous : on ne persiste pas, on ecrit pourquoi, et on classe.
      **CRITERE APPLIQUE TEL QU'ECRIT — NON FRANCHI.**
      · *Taux* : **40,9 %** sur le corpus elargi (64,1 % meme en excluant les 11 films dont
        les reperes ne coincident pas). Le seuil est 80 %. **ECHEC, et il n'est pas
        marginal.**
      · *Justesse* : **aucun oracle de zone n'existe** (2.2). Le substitut le plus fort
        disponible — la concordance inter-joueurs — donne 97,2 % sur Vagabond et 94,0 % sur
        le corpus ; il ne mesure pas la meme chose et ne peut pas tenir lieu du critere.
      **DECISION : ON NE PERSISTE PAS, ET ON CLASSE.** Le lot du 08/08 avait raison de
      refuser ; ce lot leve la 2e de ses trois objections (la correction d'horloge est
      desormais ETABLIE et LUE, plus estimee) et laisse les deux autres debout.
- [!] 4.2 Si le critere passe : table append-only (recette ADR 0026 : INSERT pur +
      `written_at`, lecture par vue `_latest` UNIQUEMENT, garde-rail anti-ART) alimentee par
      le pipeline, puis exposition. NE PAS improviser le schema : suivre la recette.
      **NON TRAITE — le gate 4.1 n'est pas franchi.** Aucune table, aucune migration, aucun
      champ ajoute a un document publie. La recette ADR 0026 ne s'applique pas : il n'y a pas
      de nouvelle table.
- [!] 4.3 Debouche visuel immediat s'il passe : le calque d'objectifs du rejeu (livre au lot 4
      de la parite) dessine deja les zones — les colorer par equipe TENANTE au fil du temps
      est un branchement, pas une feature nouvelle.
      **NON TRAITE — meme raison.** Aucun affichage n'a ete branche.

## Ce que ce lot a DECOUVERT et n'a PAS traite (perimetre tenu)

1. **Le calque d'objectifs du rejeu EN PRODUCTION est decale de `originMs`.** C'est le meme
   defaut que 1.1, du cote servi : `buildObjectiveActions` (`replay/objectives.go`) pose
   `t = TimeMS / interval` sans retrancher l'origine, et le client consomme `a.t` tel quel
   (`apps/web/src/features/match-replay/objectivesLayer.ts`, `buildObjectivePulses`) — seul
   le kill feed applique `originMs` (`killFeedLogic.ts`). Les pulses d'action d'objectif sont
   donc dessines **3,6 s a 50,8 s trop tard**, et sur l'element le plus proche du joueur au
   MAUVAIS instant. NON CORRIGE ICI : hors perimetre du plan, et la correction touche un
   document servi (donc `go test ./contracttest/` + regeneration de types web).
2. **Onze films sur 31 ont un ecart vertical de ~1 270 m** entre positions et zones (cf. 2.3).
   Defaut de repere par carte, pas du croisement. NON INSTRUIT.
3. `a521164d` (Fragmentation Heavies) rend **0 action posee** : a regarder si ce corpus est
   repris.

## Hors perimetre (ecrit noir sur blanc)

- **KOTH** (colline mobile) : hors de portee par construction, et deja acte « hors v7.5 ».
- **La lettre A/B/C des zones** : ni la variante de carte ni le film ne la portent — elle
  demande un releve Theater. `Zone.SpatialRank` reste un rang stable, JAMAIS affiche comme
  une lettre du jeu (garde deja documentee).
- **Live Fire** : module sans `sbsp`, pas de bornes, donc pas de trajectoires — hors corpus.
- Toute persistance avant le gate 4.1.

## Ce qui peut faire echouer ce plan (a dire tot)

1. L'hypothese d'horloge est fausse : les actions d'objectif ne vivent pas sur l'horloge que
   `originMs` corrige. On le saura a l'etape 1.1, en lecture de code, avant tout calcul.
2. L'oracle Vagabond est trop petit (3 matchs) pour trancher une justesse a 95 %. Dans ce
   cas, le dire — et ne pas maquiller un intervalle de confiance large en verdict.
3. Le taux reste bas apres correction : alors le defaut n'est pas l'horloge mais la
   PRESENCE des positions (un joueur hors du film, une action sans auteur decodable) — un
   autre chantier, a ouvrir avec ses propres chiffres.
