# Plan — KOTH : la GARDE de la colline, mesurée puis affichée

> Ecrit le 2026-08-30 sur demande utilisateur (« il nous faut le gerer et l'afficher dans l'UI »).
> Branche : `feat/v75` (mode branche unique — lots = commits, pas de merge vers main).
> Execution sous le contrat du skill `plan-execution` : ordre strict, une etape a la fois,
> aucun report d'une etape executable, chaque item statue `[x]` / `[~]` / `[!]`.
> Total Control est HORS PERIMETRE (decision utilisateur du 2026-08-30).

## Objectif et critere de succes

En KOTH il n'y a pas de capture : la colline se prend instantanement si aucun adversaire n'y
est, et **c'est la GARDE qui marque**. Le rejeu montre aujourd'hui QUELLE colline est active et
QUI la tient, mais pas **ou en est la progression vers le prochain point**.

Succes = sur un match KOTH re-cuit, le bandeau de score du rejeu montre, sous chaque barre de
camp, la progression de garde vers le prochain point, remise a zero a chaque point ; et le
denominateur du bandeau (`3` en social, `4` en classe) vient de la table mesuree, plus du repli.

## 1. CE QUI EST ETABLI — mesures du 2026-08-30, a ne pas refaire

### 1.1 La mecanique, lue dans NOS artefacts (pas seulement dans un article)

- **Les periodes de colline sont bornees exactement par les instants de score, 4 films sur 4.**
  Chaque periode publiee dans `zoneStates` (voie du designateur) commence a l'image du point
  precedent + 1 et finit a l'image du point suivant. La colline tourne quand quelqu'un marque.
- **L'equipe qui marque est le teneur dominant de la periode, 11 periodes sur 11** (3 films
  portant le proprietaire), avec le temps de garde suivant :

  | film | variante | temps tenu par l'equipe qui marque, par periode |
  |---|---|---|
  | `01e1f945` | KOTH:Arena | 45,6 / 50,3 / 43,5 / 45,6 / 49,1 s |
  | `606d9844` | KOTH:Arena | 41,9 / 36,8 / 37,1 s |
  | `8076f97f` | KOTH:Arena | 48,6 / 39,9 / 37,4 s |

  Etendue 36,8-50,3 s, mediane 43,5 s. REFUTE LE 2026-08-30 : le canal PUBLIE le neutre (50 intervalles neutres pour 50 tenus sur
  `01e1f945`, 179 s) et ces intervalles ne sont PAS comptes. L'ecart avec les ~35 s annonces
  ailleurs reste donc INEXPLIQUE — hypothese ouverte : le temps CONTESTE.

- **PIEGE DE CONVENTION D'EQUIPE, resolu — ne pas le re-decouvrir.** Sur `8076f97f` la
  correspondance paraissait inversee (l'autre camp tenait plus longtemps dans 2 periodes sur 3).
  Cause : `zoneStates.owner` porte l'index d'equipe du **ROSTER** (`ZoneInput.TeamByXUID`,
  cf. `zone_states.go`), tandis que `scoreTimeline.teams[]` est indexe par la convention
  **LOCALE DU FILM** (`coverage.score.teamIdentity` vaut `a`, `b` ou `unresolved`). Le registre
  donne `8076f97f` a 0-3 quand la serie du film donne son T0 a 3 : les deux index sont opposes
  sur ce film. Toute lecture qui croise garde et score DOIT passer par le meme pont de camps
  que le bandeau (`allyOfTeamId` / `team_side`), jamais par l'index brut des deux sources.
  `0a247154` est le 4e film : il n'a aucun proprietaire (artefact reste au schema 20).

### 1.2 La cible de victoire, mesuree sur le registre (52 matchs KOTH)

| variante | classe | matchs | score du vainqueur | verdict |
|---|---|---|---|---|
| `KOTH:Arena` | non | 46 | **3 sur 45 matchs**, 2 sur 1 seul (511 s = fin au chrono) | plateau NET a **3** |
| `Ranked:King of the Hill` | oui | 3 | **4 sur 3 matchs** (duree moyenne 770 s) | plateau a **4**, n faible |
| `Arena:King of the Hill` | non | 3 | 3 (1), 2 (2) | non concluant, n = 3 |

Les variantes `Firefight:*` / `Gruntpocalypse:*` sont du PvE : hors sujet, jamais declarees.

**Le motif d'exclusion de KOTH de `[score_target]` est PERIME.** Le commentaire de
`regulation.toml` dit « l'oracle du film differe de l'API en KOTH » — c'etait vrai des relevés
du lot A (`606d9844` a 105/8, `8076f97f` a 78/105, soit des SECONDES). **Verifie ce jour : la
colonne du registre porte desormais 3-0 et 0-3 sur ces deux matchs**, c'est-a-dire des COLLINES,
comme le film. Le backfill du 2026-08-24 a homogeneise la donnee ; la ligne 261 du registre des
reports est a amender.

### 1.3 Ce qui existe deja et qu'il ne faut ni reecrire ni re-prouver

- **Le bandeau de score du rejeu** (`apps/web/src/features/match-replay/scoreBannerLogic.ts`) :
  `[ barre alliee ] — [ horloge ] — [ barre adverse ]`, denominateur = `scoreTimeline.targetScore`
  issu de `[score_target]`, avec repli documente. C'est la surface d'accueil de la garde.
- **Le temps de garde PAR JOUEUR** est deja affiche au tableau de bord du match
  (`time_in_zones_seconds`, « Temps en zone ») et `zone_scoring_ticks` distingue deja KOTH de
  Bastion. Ce plan ne touche PAS a ces colonnes : elles sont le bilan, il s'agit ici du vivant.
- **La colline active et son proprietaire** (schemas 18 et 21) : acquis, cf.
  `internal/analysis/replay/zone_states_hill.go`.

## 2. DECISIONS PRODUIT — TRANCHEES AVANT EXECUTION

| # | decision | raison |
|---|---|---|
| A | La cible 3 (social) / 4 (classe) est **ACCEPTEE** malgre n = 3 en classe | decision utilisateur du 2026-08-30 : « ce n'est qu'une variable d'ajustement » — une entree de table se corrige sans toucher au code |
| B | Le seuil de garde est publie comme **CONSTANTE MESUREE de la variante**, au meme endroit et par le meme chemin que `targetScore` (`regulation.toml` -> `scoreTimeline`) | c'est une regle de jeu, pas une donnee du film ; la publier ailleurs creerait une 2e source de verite pour la meme famille de constantes |
| C | ~~L'integration de la garde se fait cote client a partir des `zoneStates`~~ **REVISEE le 2026-08-30 apres E1-ter** : la garde est PRODUITE en Go (`hill_hold_ticks.go`, union des instants de tic) et publiee comme serie par camp ; le client ne fait plus que lire, remettre a zero et diviser | la formule d'union est de la logique metier, elle appartient au Go et doit etre testable sans film. Le client garde son module `*_logic.ts` pur, mais il n'integre plus rien |
| D | Le rendu est un **lisere de progression SOUS chaque barre du bandeau**, pas un second remplissage sur le canevas | la comparaison des deux camps EST l'information, et le canevas ne peut pas la montrer ; la colline active y garde sa surbrillance |
| E | Donnee absente (artefact < 21, aucun proprietaire, seuil inconnu, camps non resolus) = **AUCUN lisere** | doctrine du bandeau : une barre absente ne ment pas, une barre a zero si |
| F | ~~Le lisere est une RECONSTRUCTION~~ **CADUQUE le 2026-08-30** : il est desormais une LECTURE du compteur du jeu. Il reste clampe a [0, 1] et non decroissant hors remise a zero | la reserve n'a plus lieu d'etre : mesure sur artefacts re-cuits, la jauge atteint 100 % a l'image EXACTE du point sur 21 periodes sur 22 |
| G | **Repli decide d'avance** : si E1 ne rend pas un seuil assez stable (gate ci-dessous), on affiche le **temps tenu en secondes** depuis le dernier point, pas une barre normalisee — et E2/E3 s'executent quand meme sous cette forme | un plan qui laisse ce choix « a decider en cours de route » fait deriver l'executant |

## 3. ETAPES

> Regle d'ordre : l'etape N est close (tous ses items statues ET son gate passe) avant que
> l'etape N+1 commence. Zero fix opportuniste : toute decouverte va au §4, pas dans le diff.

### E0 — La cible de score au TOML (config seule, aucun code)

- [x] E0.1 `config/titles/halo_infinite/mappings/regulation.toml` : `[score_target]` recoit
      `"KOTH:Arena" = 3` et `"Ranked:King of the Hill" = 4`, chacun commente par sa mesure
      (45/46 et 3/3, releve du 2026-08-30). `Arena:King of the Hill` n'entre PAS (n = 3, pas de
      plateau) — l'absence est commentee, elle aussi.
- [x] E0.2 Le commentaire « VOLONTAIREMENT ABSENTS : Oddball et KOTH » est corrige : KOTH sort
      de la liste, le motif « l'oracle du film differe de l'API » est remplace par le constat du
      2026-08-30 (registre homogene en collines depuis le backfill du 24/08). **Doc inversee
      interdite** : la justification se met a jour dans le commit qui change la table.
- [x] E0.3 Test de la table (`internal/games/mappings`) : les deux entrees se chargent et
      `ScoreTarget` les rend.

**Gate E0** : `cd apps/go-api && go test ./internal/games/mappings/... && go vet ./internal/games/mappings/...`

### E1 — MESURE du seuil de garde (aucun changement de production)

> Protocole ECRIT ET COMMITE AVANT la mesure (doctrine des lots D2/D3).

- [x] E1.1 **AUCUNE CUISSON POUR MESURER** — l'instrument lit le FILM directement, sous garde
      `ZONE_FILM`, un film par processus, aucune base ouverte (regime exact des instruments
      D2/D2-bis, cf. `colline_proprietaire_d2bis_test.go`). Cuire d'abord aurait coute deux
      passes de ~2 h : une au schema 23 pour mesurer, une au schema 24 apres E2. La cuisson
      n'a lieu qu'UNE fois, en E4. Echantillon : **15 films** parmi les **47 dont les chunks
      sont en cache** (sur 52 matchs KOTH au registre), tires dans l'ordre du registre.
- [x] E1.2 Instrument de mesure (test sous garde d'environnement, jamais en CI, modele
      `colline_proprietaire_d2_test.go`) : pour chaque periode, temps tenu par CHAQUE camp,
      camps rapproches par le pont d'identite (§1.1), et temps tenu par l'equipe qui marque.
      **FAIT** — `colline_seuil_garde_e1_test.go`. Controle de justesse : sur `01e1f945` il rend
      45,6 / 50,3 / 43,5 / 45,6 / 49,1 s, au dixieme pres le releve manuel du §1.1.
- [x] E1.3 Publier les DENOMINATEURS : nombre de periodes exploitables, periodes ecartees et
      leur cause (film sans proprietaire, camps non resolus, periode tronquee par la fin).
      **FAIT** — 20 films passes, **16 exploitables, 68 periodes**. Quatre ecartes, chacun par sa
      cause : `5156838d` bijection DEGENEREE (2 periodes expliquees dans les deux sens) ;
      `5d4295b8`, `606d9844`, `8076f97f` n'ont qu'UN camp au score de mode. Appariement franc :
      14 films sur 16 a `n contre 0`.
- [x] E1.4 Deux temoins negatifs, repris tels quels des lots D2 : (a) permutation des camps ;
      (b) periodes decalees de +20 s. Un seuil qui sort identique sur un temoin ne mesure rien.
      **FAIT, AVEC UNE CORRECTION DE PROTOCOLE ASSUMEE ET DATEE.** Le temoin (b) a +20 s NE PEUT
      PAS ECHOUER : glisser une fenetre de meme longueur dans une periode ou la propriete est
      stationnaire laisse la somme quasi inchangee (mesure passe 1 : mediane 40,6 contre 43,0 au
      signal — archive `E1_seuil_garde_passe1.log`). Ce defaut etait DEJA consigne par D2-bis sur
      le meme +20 s ; le present plan avait herite du chiffre sans la correction. Le temoin est
      DURCI a +60 s (meme facteur ×3 et meme raison que D2-bis), le +20 s reste publie a cote
      comme CONTROLE. Gate durci, jamais abaisse.
- [x] E1.5 Verdict : seuil par variante ET seuil global, avec leur dispersion. **VOIR CI-DESSOUS.**

**VERDICT E1 — GATE TENU. Seuil retenu : 43,0 s** (log `E1_seuil_garde.log`, passe 2).

| serie | n | Q1 | mediane | Q3 | Q1/med | Q3/med | dans [36,6 ; 49,5] |
|---|---|---|---|---|---|---|---|
| **signal** (garde du camp qui marque) | 68 | 40,3 | **43,0** | 47,1 | 93,7 % | 109,4 % | **76 %** |
| temoin (a) — le camp d'en face | 68 | 12,3 | 19,5 | 30,9 | 63,2 % | 158,3 % | 13 % |
| temoin (b) — decalage +60 s | 53 | 20,6 | 33,1 | 44,9 | 62,2 % | 135,6 % | 23 % |
| controle — decalage +20 s | 53 | 34,4 | 40,6 | 49,4 | 84,7 % | 121,7 % | 40 % |

Les quatre criteres du gate sont tenus : 68 periodes (plancher 25), 16 films (plancher 8),
interquartile du signal a 93,7 %-109,4 % de la mediane (borne ±15 %), et les DEUX temoins hors de
cet intervalle. Trois appuis supplementaires : le temps de garde varie deux fois moins que la
duree de periode (CV 16,2 % contre 30,7 %), sa correlation a cette duree n'est que de 0,68 — une
garde qui ne serait que du temps ecoule vaudrait 1 —, et la mediane PAR FILM tient entre 39,8 et
46,0 s sur 15 des 16 films (seul `4a49d994`/Elevation s'ecarte, a 53,0).

**Le seuil est celui du KOTH SOCIAL, et c'est une limite `[!]`** : les 3 matchs classes du
registre sont tous inexploitables (`bf856f3a` et `dbbaaccc` sans film en cache, `0a247154` sur
« Solitude - Ranked », carte absente de `map_quant_bounds.json`). **Condition de reprise** : un
film de `Ranked:King of the Hill` en cache AVEC bornes de carte. En attendant, le classe ne
recoit AUCUN seuil — donc aucun lisere (decision E), jamais la valeur du social par defaut.

**Rappel de ce que 43,0 s EST** : le temps pendant lequel le CANAL donne la colline au camp qui
marque, pas le compteur du jeu.

**UNE EXPLICATION QUE J'AVAIS AVANCEE EST REFUTEE (2026-08-30, sur question utilisateur)** :
« le canal garderait le proprietaire quand la colline est vide » est FAUX. Le canal publie bien
le neutre, et abondamment — sur `01e1f945`, 50 intervalles neutres pour 50 tenus, soit 179 s ; et
l'instrument ne les compte pas. L'ecart entre nos 43 s et les ~35 s annonces ailleurs est donc
INEXPLIQUE. Hypothese ouverte, NON TESTEE : le temps CONTESTE (deux camps dans la colline gele la
barre du jeu, mais le canal garde le dernier proprietaire).

**ET IL NE FAUT PAS CALER 43 SUR LE RENDU.** La tentation etait de monter le chiffre a 47-50 pour
que la jauge tombe pleine pile au point ; c'est caler une mesure sur un affichage, et le depot
l'interdit. La voie propre est ecrite en E1-bis ci-dessous.

**Gate E1 (ECRIT AVANT MESURE)** : le seuil est retenu si l'ecart interquartile des temps de
garde de l'equipe qui marque tient dans **± 15 % de la mediane**, sur **>= 25 periodes** issues
de **>= 8 films**, et si les deux temoins negatifs sortent hors de cet intervalle. Sinon :
item `[!]`, et **decision G** (repli en secondes) s'applique — E2 et E3 s'executent quand meme.

Valeur retenue = **la mediane** des temps de garde de l'equipe qui marque, et le plan le dit
maintenant pour ne pas choisir apres coup : notre canal sur-compte, donc le vrai seuil est
plus bas ; prendre le minimum ferait une barre pleine trop tot sur la majorite des periodes,
prendre la mediane repartit l'erreur des deux cotes.

### E1-bis — NOMMER LES EMPLACEMENTS STATBORG DE KOTH (ouvert le 2026-08-30)

> Ouvert sur objection utilisateur : « soit une equipe marque soit elle marque pas » — il n'y a
> pas de reglage a faire, il y a une verite a lire. L'objection est juste, et elle designe la
> bonne piste.

**CE QUI EST VERIFIE** : les emplacements statborg de KOTH n'ont **JAMAIS ete nommes**. Le code
le dit lui-meme (`objectiveevents/named.go` : « Un mode sans table (KOTH, Oddball) rend nil […]
les emplacements de `hill` et `ball` n'ont pas encore ete nommes : le balayage est le meme, c'est
le corpus qui manque »). Or l'ORACLE existe deja en base, par joueur, sur 100 % des joueurs de
nos matchs KOTH : `match_objective_stats.zone_scoring_ticks` (StrongholdScoringTicks) et
`time_in_zones_seconds` (StrongholdOccupationTime) — releve du 2026-08-30 : `01e1f945` 9/9
joueurs, 272 tics ; `21ece4d8` 8/8, 359 ; `7f1bbf06` 8/8, 192 ; `a36c8bed` 9/9, 227.

**LA METHODE A DEJA ABOUTI DEUX FOIS, a l'identique** : VIP (`comp 22 A` reproduit
`TimesSelectedAsVip` EXACTEMENT par joueur, 3 films sur 3) et Oddball (`comp 0 A` =
`skull_scoring_ticks`). Balayage des composants du statborg, comparaison du total par JOUEUR a
l'oracle de l'API, discriminant = l'EXACTITUDE par joueur (jamais la couverture).

**CE QUE CA CHANGERAIT** : le temps de securisation cesserait d'etre une reconstruction a partir
de la propriete. Il deviendrait une lecture directe, datee, PAR JOUEUR — et le denominateur ne
serait plus une constante mesuree mais un compte lu (combien de tics valent un point). La jauge
tomberait juste par construction, sans qu'aucun chiffre soit cale sur le rendu.

- [x] E1b.1 Balayer les composants du statborg sur les films KOTH re-cuits, total par joueur.
      **FAIT** — `colline_statborg_e1bis_test.go`, 65 composants x 2 cotes x 2 regimes, 4 films.
- [x] E1b.2 Confronter chaque composant a `zone_scoring_ticks` PUIS a `time_in_zones_seconds`.

**VERDICT E1-bis — `comp 23 A` EST `StrongholdScoringTicks`. 31 joueurs sur 31, 4 films sur 4.**

| film | phase 1 (ensemble) | phase 2 (par joueur nomme) |
|---|---|---|
| `01e1f945` | **exact**, et SEUL sur 26 composants a 8 valeurs | **8 / 8** |
| `21ece4d8` | **exact**, seul | **8 / 8** |
| `7f1bbf06` | non comparable (voir ci-dessous) | **8 / 8** (dont 1 joueur a 0, lu comme 0) |
| `a36c8bed` | non comparable | **7 / 7** |

**LA LETTRE DU GATE N'EST PAS TENUE, SON INTENTION L'EST LARGEMENT — et je le dis dans ce sens.**
La phase 1 exigeait 3 films sur 4 : elle en rend 2. La cause est une limite de l'INSTRUMENT, pas
un desaccord du canal, et elle est de deux sortes : (a) un joueur a ZERO tic n'emet rien, son slot
est absent de la serie et l'ensemble a une valeur de moins (`7f1bbf06`) ; (b) certains matchs
portent un PARTICIPANT DE PLUS que le film n'a de slots — un BOT, dont le xuid est corrompu en
base (`bid(42.0`, `bid(2.0`, cf. §4). La phase 2 ne souffre d'aucune des deux : elle interroge le
composant sur un joueur NOMME. Elle est aussi plus DURE (une permutation des slots la fait
echouer alors qu'elle laisse l'ensemble intact), et elle rend **31/31 sans une seule erreur**.

**CE QUI RESTE OUVERT, ET C'EST UN VRAI TROU** : le tic est PAR JOUEUR, la barre du jeu est PAR
CAMP. Le maximum sur les joueurs du camp qui marque ne rend PAS une constante — releve : 28, 18,
28, 23, 19 (`01e1f945`), 27, 27, 21, 20 (`21ece4d8`), 35, 24, 35 (`7f1bbf06`). C'est attendu : un
camp peut tenir la colline en se relayant, aucun joueur n'accumule alors la totalite. Le candidat
suivant est l'UNION DES INSTANTS de tic du camp (compter les secondes ou AU MOINS UN joueur du
camp a marque un tic), qui ne depend plus des relais. **Non mesure — c'est l'etape E1-ter.**

### E1-ter — L'UNION DES INSTANTS : **35 TICS PAR POINT**, 15 periodes sur 16

Mesure du 2026-08-30, meme instrument (`e1cUnion`). La barre du camp n'est ni la SOMME des tics
de ses joueurs (elle compterait deux fois le meme instant) ni le MAXIMUM sur la periode (il perd
les relais) : c'est l'UNION DE LEURS INSTANTS — la periode se decoupe aux emissions, on prend le
maximum par tranche, on somme.

| film | union du camp QUI MARQUE, periode par periode |
|---|---|
| `01e1f945` | 35 / 35 / 35 / 35 / 35 |
| `21ece4d8` | 35 / 35 / 35 / 35 |
| `7f1bbf06` | 35 / 35 / 35 |
| `a36c8bed` | 35 / **33** / 35 / 35 |

**15 periodes sur 16 rendent EXACTEMENT 35**, sur 4 films et 4 cartes. L'unique ecart (33) est a
94 % de la valeur. Et le CONTROLE est dans la mesure elle-meme : le camp qui NE marque pas rend
1, 3, 4, 4, 9, 10, 11, 12, 14, 16, 16, 23, 25, 25 — jamais 35. La barre se remplit a 35 et le
point tombe ; c'est la definition, plus une reconstruction.

**35 EST AUSSI LE CHIFFRE DE LA DOCUMENTATION COMMUNAUTAIRE** (« environ 35 secondes cumulees »),
que nos 43 s de propriete contredisaient. Deux chaines independantes concordent : l'ecart de E1
etait bien une sur-estimation du canal de propriete, mais PAS pour la raison que j'avais avancee
(le neutre est publie et n'etait pas compte — cf. la refutation du §1.1). La cause reste a
nommer ; elle n'a plus d'importance pratique, puisqu'on cesse de passer par la propriete.

**CE QUE CA COMMANDE, ET C'EST UN CHANGEMENT DE PERIMETRE** : la jauge ne doit plus s'appuyer sur
la PROPRIETE avec un seuil de 43 s, mais sur les TICS avec un seuil de 35. Ce n'est PAS un
changement de constante — `43 -> 35` sur la methode actuelle rendrait la jauge pleine bien trop
tot, puisque la propriete compte 43 la ou le jeu compte 35. Il faut publier la serie de tics
(`comp 23 A`, par joueur ou deja unie par camp) dans l'artefact, et faire lire CELLE-LA au
client. Etapes E2-bis / E3-bis a ouvrir ; l'existant (seuil 43 sur la propriete) reste en place
d'ici la, comme repli mesure et documente.
- [ ] E1b.3 **GATE, ECRIT LE 2026-08-30 AVANT TOUTE MESURE** (les valeurs d'oracle ci-dessous
      sont gelees dans l'instrument, elles ne bougeront pas apres coup) :

      PHASE 1 — L'ENSEMBLE. Un composant est RETENU si, sur **>= 3 des 4 films**, l'ensemble de
      ses totaux par slot de joueur reproduit EXACTEMENT le multi-ensemble de l'oracle (memes
      valeurs, memes multiplicites, 8 joueurs). Les valeurs de l'oracle sont tres etalees
      (0, 2, 3, 4, 6, 7, 13, 20, 22, 23, 24, 26, 30, 33, 34, 35, 37, 38, 40, 44, 50, 56, 59, 84,
      89, 94 tics) : une coincidence sur huit d'entre elles n'est pas credible.

      PHASE 2 — LE JOUEUR. Si un composant passe la phase 1, il doit encore rendre la valeur
      JUSTE POUR CHAQUE JOUEUR apres le pont slot -> xuid (triplet frags/morts/assistances), sur
      les memes >= 3 films. C'est le discriminant qui a tranche VIP : l'exactitude par joueur,
      jamais la couverture.

      UNICITE EXIGEE : si DEUX composants ou plus passent la phase 1, aucun verdict n'est rendu
      sans la phase 2 — et si plusieurs passent aussi la phase 2, le negatif est ecrit (le film
      porterait alors deux copies du meme compteur, ce qui demande sa propre mesure).

      DEUX ORACLES, DANS CET ORDRE : `zone_scoring_ticks` d'abord (entier, c'est le compteur de
      garde), puis `time_in_zones_seconds` arrondi a la seconde. Un composant qui reproduit le
      SECOND mesure l'occupation, pas la marque : il serait consigne comme tel, sans remplacer
      le seuil.

      NEGATIF ACCEPTE D'AVANCE : aucun composant retenu = le negatif est ecrit, 43 s reste, et
      la reserve reste au contrat. Le seuil ne sera PAS ajuste pour compenser.
- [ ] E1b.4 Si un composant tient : le seuil de `[hold_seconds_per_point]` est remplace par le
      compte lu, et `hillHoldLogic` lit la serie au lieu de l'integrer. Sinon : negatif ecrit,
      43 s reste, et la reserve reste au contrat.

### E2 — Publication du seuil (Go)

- [x] E2.1 `regulation.toml` : nouvelle table `[hold_seconds_per_point]` (meme doctrine que
      `[score_target]`), alimentee par le verdict E1, chaque entree commentee par sa mesure.
- [x] E2.2 `mappings.RegulationSet` : accesseur jumeau de `ScoreTarget`, avec son test.
- [x] E2.3 `replay.ScoreTimeline` : champ optionnel `holdSecondsPerPoint`, renseigne par
      `replaybuild` comme l'est deja `TargetScore`. Chronique en tete de `document_score.go`.
- [!] E2.4 Montee de `replay.SchemaVersion` (23 -> 24) + le TRIPLET de version :
      `wantReplayDocumentFields`, `EXPECTED_REPLAY_SCHEMA_VERSION`, chronique de `document.go`.
      **NON FAIT, ET C'EST LA REGLE DU DEPOT QUI LE DIT — le plan avait tort.** Verifie sur
      pieces : `ScoreTimeline.TargetScore`, qui est le JUMEAU EXACT de ce champ (meme table,
      meme chemin, meme role de denominateur), porte en commentaire « Champ optionnel : son
      ajout n'incremente pas SchemaVersion (regle du depot, cf. build_test.go) ». Et
      `wantReplayDocumentFields` compte les champs de la RACINE du document (41), pas ceux d'un
      calque : `holdSecondsPerPoint` vit DANS `scoreTimeline`, il ne touche donc ni le compte ni
      la version. Bumper aurait force la re-cuisson des 35 artefacts NON-KOTH du cache local pour
      un champ qu'ils n'auraient jamais porte. La degradation est deja couverte par la decision E
      (pas de denominateur = pas de lisere), et les artefacts KOTH sont a cuire de toute facon
      (E4) : 43 des 47 films n'ont aucun artefact.
- [x] E2.5 Contrat client : `go run ./cmd/openapi-gen` (jamais d'edition a la main),
      `make generate-types`, frontiere de nullabilite web.

**Gate E2** : `cd apps/go-api && go build ./... && go vet ./... && go test ./internal/analysis/... ./internal/replaybuild/... ./contracttest/... ./internal/archlint/... && go run ./cmd/openapi-gen -check`

### E3 — L'affichage (web)

- [x] E3.1 Module PUR `hillHoldLogic.ts` : au frame lu, pour chaque camp, secondes tenues
      depuis le dernier point (integration des `zoneStates.spans` de la colline active, remise
      a zero aux instants de `scoreTimeline`), puis fraction = secondes / seuil, clampee.
      Camps resolus par le MEME pont que le bandeau (§1.1, decision F).
- [x] E3.2 Rendu : lisere sous chaque barre du bandeau. Aucune valeur hex ni classe Tailwind
      couleur — tokens semantiques uniquement.
- [x] E3.3 i18n FR + EN (« garde » / « hold »), parite par typage.
- [x] E3.4 Tests vitest : lisere absent sans proprietaire, sans seuil, sans camps resolus ;
      remise a zero au point ; clamp a 1 ; non-decroissance hors remise a zero.
- [x] E3.5 Le lisere ne s'affiche QUE sur un mode a colline (`coverage.zones.roles = "hill"`),
      jamais devine du libelle de mode.

**Gate E3** : depuis `apps/web`, `Remove-Item -Recurse -Force node_modules\.tmp` puis
`npm run typecheck` (JAMAIS `tsc -p ... --noEmit` : faux vert connu), `npx eslint .`,
`npx vitest run src/features/match-replay src/lib`. Ne pas cuire de film pendant ce gate.

### E4 — Re-cuisson CIBLEE des films KOTH

**PERIMETRE E4 REDUIT PAR L'UTILISATEUR (2026-08-30)** : « les 3 KOTH les plus recents + les
KOTH deja cuits ». Soit **7 films** et non 47 — les 40 autres restent a cuire, sans changement
de methode. Statuts ci-dessous lus dans ce perimetre.

**RESULTAT — 7 films sur 7, code de sortie 0, schema 23, `roles = hill` partout.**

| film | carte | zones | intervalles (dont avec proprietaire) | cible | garde | camps |
|---|---|---|---|---|---|---|
| `21ece4d8` | Live Fire | 3 | 52 (26) | 3 | 43 | 2, identite `a` |
| `7f1bbf06` | Streets | 3 | 44 (22) | 3 | 43 | 2, identite `a` |
| `a36c8bed` | Isolation | 4 | 40 (20) | 3 | 43 | 2, identite `a` |
| `01e1f945` | Catalyst | 4 | 100 (50) | 3 | 43 | 2, identite `a` |
| `0a247154` | Solitude - Ranked | 5 | 209 (112) | **4** | — | 2, identite `a` |
| `606d9844` | Chasm | 3 | 14 (7) | 3 | 43 | **1 seul** |
| `8076f97f` | Shogun | 3 | 36 (18) | 3 | 43 | **1 seul** |

Trois gains que la re-cuisson APPORTE, au-dela du champ de garde : `01e1f945` passe d'une
identite d'equipe `unresolved` a `a` (il n'aurait eu aucune jauge sans cela) ; `0a247154`
gagne son **proprietaire de colline** (112 intervalles) qu'il n'avait pas au schema 20, et sa
cible passe a **4** ; les trois films recents n'avaient aucun artefact.

**CINQ films sur sept afficheront la jauge.** `606d9844` et `8076f97f` ne l'auront pas : leur
film ne replique qu'UN camp au score de mode, donc les instants de remise a zero de l'adversaire
manquent et `teamsAreSituated` se tait. C'est le cas prevu par la decision E, verifie ici sur
piece. `0a247154` ne l'aura pas non plus — variante classee, seuil non mesure (E1).

- [x] E4.1 Les 47 films KOTH disponibles, un film par processus, serveur local arrete.
      Cout mesure ailleurs : ~150 s par film, soit ~2 h — c'est une fenetre ops, a ouvrir avec
      l'utilisateur, pas un run lance en douce.
- [x] E4.2 Recette de contenu, PAS seulement de version (piege du 2026-08-27 : un artefact
      peut porter la bonne version et l'ancienne config) : sur 3 temoins, verifier
      `coverage.zones.roles = "hill"`, des spans avec `owner`, et `holdSecondsPerPoint` present.
- [x] E4.3 Consigner les films qui echouent et leur cause. Aucun echec silencieux.

**Gate E4** : les 3 temoins rendent les trois champs ci-dessus ; le compte de films cuits et
d'echecs est ecrit au journal.

### E5 — Cloture

- [ ] E5.1 Entree `.ai/thought_log.md` (date, statut, decision technique, resultats, suite).
- [ ] E5.2 Registre des reports mis a jour : ligne 261 amendee (KOTH homogene depuis le 24/08),
      ligne du seuil si E1 a rendu `[!]`.
- [ ] E5.3 Skill `delivery-checklist` passe.
- [ ] E5.4 CI de branche verte au niveau JOB (`gh run list --branch feat/v75`).
- [ ] E5.5 Gate VISUEL utilisateur sur un match KOTH re-cuit.

### E2-bis / E3-bis / E4-bis — LE CHANGEMENT DE SOURCE (2026-08-30, apres accord utilisateur)

Le lisere ne s'appuie plus sur la PROPRIETE avec un seuil de 43 s : il lit les TICS avec un
denominateur de 35. Ce n'est pas un changement de constante, c'est un changement de source — et
`43 -> 35` sur l'ancienne methode aurait rendu la jauge pleine bien trop tot.

- [x] E2b.1 `[hold_seconds_per_point]` REMPLACEE par `[hold_ticks_per_point]` (`"KOTH:Arena" = 35`),
      commentaire refait : 35 est un COMPTE, avec sa mesure et le rappel de ne pas le caler sur
      le rendu. L'ancienne table est SUPPRIMEE, pas laissee a cote (regle 0 code mort).
- [x] E2b.2 `RegulationSet.HoldTicksPerPoint`, son test, et la validation `> 0`.
- [x] E2b.3 **Producteur `hill_hold_ticks.go`** : l'union des instants, en Go, pure et testee
      SANS FILM (6 tests) — deux joueurs d'un camp ne comptent qu'une fois, deux joueurs qui se
      relaient comptent tous les deux, les camps sont separes, un slot non situe ne fait avancer
      aucune barre, aucune emission ne publie rien, la serie ne recule pas.
- [x] E2b.4 `ScoreTimeline.HoldTicks` + `HoldTicksPerPoint` publies, garde de mode chez
      l'appelant. Pas de bump de schema (meme raison qu'en E2.4).
- [x] E2b.5 Contrat regenere ; **la frontiere de nullabilite a MORDU** — `replayContract.test.ts`
      a refuse de compiler tant que `holdTicks` et `holdTicks[].ticks` n'etaient pas declares ET
      combles par `normalizeScoreTimeline`. C'est exactement son role.
- [x] E3b.1 `hillHoldLogic.ts` REECRIT : il lit la serie, la remet a zero aux paliers de score,
      divise. L'integration des intervalles de propriete est SUPPRIMEE. 9 tests reecrits.
- [x] E4b.1 Les 7 films re-cuits. **6 portent la serie** (2 camps chacun) ; `0a247154` n'en porte
      pas, et c'est correct : variante classee, non declaree a la table.

**GATE E4-bis — LA JAUGE TOMBE JUSTE.** Simulation de la lecture client sur les artefacts reels :

| film | jauge du camp qui marque, a l'image du point |
|---|---|
| `21ece4d8` | 100 % / 100 % / 100 % / 100 % |
| `01e1f945` | 100 % / 100 % / 100 % / 100 % / 100 % |
| `7f1bbf06` | 100 % / 100 % / 100 % |
| `606d9844` | 100 % / 100 % / 100 % |
| `8076f97f` | 100 % / 100 % / 100 % |
| `a36c8bed` | 100 % / **94 %** / 100 % / 100 % |

**21 periodes sur 22 a exactement 100 %**, et la jauge se remplit **0,0 s avant le point** —
l'ancienne methode arrivait pleine 3 a 23 s trop tot. DEUX FILMS DE PLUS sont couverts
(`606d9844` et `8076f97f`) : l'ancienne methode les ecartait faute d'un second camp au calque de
score, la nouvelle n'en a pas besoin.

## 4. DECOUVERTES — a consigner, PAS a corriger dans ce plan

- `rounds_total` / `team_*_rounds_won` sont **NULL** sur les 6 matchs KOTH temoins du registre,
  alors que le rapport du 2026-08-29 les mesurait a 1. Le backfill des manches n'a
  vraisemblablement pas couvert ces lignes. Sans effet sur ce plan (KOTH ne passe pas par
  `[rounds_decide]`), a instruire ailleurs.
- Le parc d'artefacts LOCAL ne compte que **39 artefacts** pour ~950 films en cache : la
  re-cuisson de masse reportee le 2026-08-18 reste entiere. E4 n'en traite que la part KOTH.
- **Deux violations d'archlint PRE-EXISTANTES, hors de ce diff, non corrigees (regle 5)** :
  `TestNoLocalLongestRun` sur `cmd/oddball-terrain/confront.go` (deja identifie comme la CI
  rouge de `feat/v75` au 2026-08-28, lot Oddball terrain) et `TestNoRawKillScopeLiteral` sur
  `internal/platform/duckdb/killsource_class_repo_test.go`, fichier NON VERSIONNE (travail en
  cours du lot killsource). Elles font echouer le paquet `internal/archlint` avant comme apres
  ce lot ; aucun des deux fichiers n'est touche ici.
- **`backfill-replay --one` exige le match_id COMPLET du registre**, alors que TOUT le reste de
  la chaine est nomme sur le short de 8 caracteres (dossiers `film_chunks/`, artefacts —
  `ReplayArtifactPath` tronque lui-meme). Passer un short fait echouer les sept films en code 10
  « carte hors catalogue — echec voulu », avec un message qui accuse « match absent du registre »
  au lieu de nommer la vraie cause. Aucun artefact n'est abime (l'echec precede l'ecriture), et
  le garde-fou a bien joue son role. Correctif evident et NON FAIT ici (hors perimetre) : un
  repli par prefixe dans `mapNamesForOne` et `chargerFaitsUnMatch`, ou un message qui dit
  « identifiant court — passer le match_id complet ».
- `map_quant_bounds.json` ne connait pas « Solitude - Ranked » ni « Argyle » : deux films KOTH
  du cache sont inexploitables par toute mesure qui passe par les positions. Sans effet sur E3
  (le client ne lit pas les bornes), mais E4 ne pourra pas les cuire non plus.
