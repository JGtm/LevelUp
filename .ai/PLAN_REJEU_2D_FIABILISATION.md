# PLAN — fiabiliser le rejeu 2D, puis le sortir du POC

> Écrit le 2026-07-28. Branche : `feat/filmdec-continuation` (celle en cours, un seul sujet).
>
> **Ce plan s'exécute sous le contrat du skill `plan-execution`.** Ordre strict, une étape à la
> fois, aucun report d'une action faisable maintenant, chaque item statué à la clôture.
>
> Documents qui font foi : `SUIVI_REPLAY_2D.md` (avancement),
> `PROPOSITION_FIABILITE_RATTACHEMENT.md` (le diagnostic), `CAHIER_DES_CHARGES_POC.md` (l'écran),
> `METHODE_RETRO_INGENIERIE_FILM.md` (comment chercher).

---

## POURQUOI CE PLAN EXISTE

Le POC n'est **pas viable en l'état**, et il faut le dire avec ses raisons plutôt qu'avec un
adjectif :

| ce qui n'est pas viable | mesure |
|---|---|
| Le rattachement d'un événement à un joueur est un **vote**, pas une lecture | 26 slots couverts sur 99 |
| Les événements non rattachés sont **jetés sans trace** | 53 records perdus, découverts par l'utilisateur en regardant l'écran |
| Le décodeur **ne publie pas sa couverture** | 147 tirs publiés, 519 disponibles, l'écart n'est écrit nulle part |
| Tout vit en **injection manuelle** dans un fichier HTML | `features/match-replay` est antérieur à la structure, aux tirs, à l'inventaire |

Les trois premiers sont **le même défaut** : un décodage inachevé comblé par une inférence, puis
construit dessus. Le quatrième est indépendant et vient après.

**CRITÈRE DE SUCCÈS DU PLAN** : le rejeu publie, pour chaque calque, *combien il a rattaché sur
combien existent*, et ce rapport dépasse 85 % sur les tirs — contre 28 % aujourd'hui.

---

## DÉCISIONS TRANCHÉES AVANT L'EXÉCUTION

Elles ne se rediscutent pas en cours de route. C'est ce qui empêche un agent de dériver.

1. **On lit avant d'inférer.** Une inférence n'est acceptable qu'après avoir montré que la lecture
   est impossible, et elle se marque à l'écran comme telle.
2. **Un index est un ordre, jamais une identité.** L'identité d'un joueur est son xuid.
3. **Rien ne se jette en silence.** Tout rejet est compté et catégorisé.
4. **On ne stocke jamais une résolution qui peut s'améliorer** (règle reprise du chantier voisin) :
   le tag brut se garde à côté du libellé.
5. **Le POC s'édite, il ne se régénère pas.** Le bloc de données peut être remplacé.
6. **Un seul agent compile du Go à la fois** (cache de build Windows).
7. **Les véhicules sont hors périmètre** — écartés par l'utilisateur le 2026-07-28.

---

## ÉTAPE 1 — Lire le lien joueur → entité

**Objectif** : décoder `player-representation-component` (archétype **5**, rang **i21**) et,
subsidiairement, `player-primary-respawn-object-component` (`i10`).

**Pourquoi en premier** : c'est le seul point qui supprime la *raison* du vote au lieu d'en
améliorer le résultat. Tout le reste du plan en dépend ou en profite.

### Items

- [x] 1.1 Confirmer que l'archétype 5 est bien instancié dans `000d5950`, et compter ses entités.
      **Fait** : 27 composants, `i10` et `i21` présents au registre. 32 entités (slots 52..83),
      **832 records = 32 × 26 images-clés, exactement**. Huit sont actives : les records des slots
      52..59 font 649-685 bits contre 547 pour les 24 autres, sur les 25 images-clés d'après le
      démarrage. Outil : `cmd/tmp_playerrep -mode census`.
- [x] 1.2 Mesurer la largeur consommée par chaque composant `i0..i21` de l'archétype 5, par la
      méthode déjà établie : **différence de curseur entre composants consécutifs**.
      **Fait** sur le flux delta (`-mode delta`) : tableau des 27 rangs, présence / atteinte /
      largeur. `i21` y consomme **32 bits, 34 fois sur 34** — le déser porté est de largeur
      constante. Les deux seuls composants à largeur 0 sont `i22` et `i24`, non portés : ce sont
      les points d'arrêt de la chaîne.
- [!] 1.3 Atteindre `i21`. **ÉCHEC MESURÉ sur les deux voies.**
      **(a) chaîne séquentielle** : le flux delta ne rencontre que **125 records ti=5 sur tout le
      film** et en désynchronise 47 % (`i7` 10, `i10` 8, `i22` 18, `i24` 23). Les slots servis
      sont impossibles (1042, 1940, 256 — le keyframe ne lie ces slots à aucun ti=5) : le
      parcours dérive avant d'arriver au bout.
      **(b) ancrage par signature** : le prédicat « mot de 32 bits dont le slot est un biped de
      la même image-clé » rend **0 touche sur 832 records**. La forme `R(32)` brute est donc
      **exclue**, pas seulement improbable.
      **(c) forme compressée `R(W)+R(2)`**, non prévue au plan mais imposée par (b) — c'est
      l'écriture que le format emploie partout ailleurs pour un handle (`readRecordID`, et le
      port d'`i10` lui-même). Résolue par contrainte sur (largeur, décalage) : au mieux
      **26 liens sur 200**, **0 image-clé sur 25 où les huit joueurs sont lus**, et deux
      violations d'injectivité (`52->554` et `54->554` à la même image-clé). C'est du bruit.
- [!] 1.4 Lire la valeur et vérifier qu'elle désigne une entité (`slot = eid & 0x3fffffff`).
      **La valeur lue ne désigne pas une entité** : sur les 34 lectures d'`i21` du flux delta,
      les slots valent jusqu'à 8·10⁸ et les générations se répartissent uniformément sur 0..3
      (11 / 6 / 10 / 7), alors qu'un handle valide exclut la génération 0. C'est la signature
      d'une lecture hors position, pas d'un champ.
- [~] 1.5 **Contrôle qui peut échouer** — *sans objet* : couvert par l'échec de 1.3/1.4. Le
      contrôle valide une lecture ; il n'y a pas de lecture à valider. Ne pas le confondre avec
      un contrôle passé.
- [~] 1.6 **Second contrôle, source disjointe** — *sans objet*, même raison que 1.5.

### Gate

```
cd apps/go-api && go run ./cmd/tmp_archlist "player-representation"   # PASSÉ : i21 existe
cd apps/go-api && go run ./cmd/tmp_playerrep -mode census|delta|anchor|solve
# lien lu pour N vies sur 99 : N = 0. Concordance sur les 26 slots surs : NON ÉVALUABLE.
```

**Clôture** : la concordance 26/26 n'est pas atteinte ; **l'échec est mesuré**, et l'étape 2
bascule sur le repli — c'est la branche que cette étape prévoyait.

### CE QUE L'ÉCHEC APPREND, ET QU'IL FAUT NE PAS SURINTERPRÉTER

Le diagnostic de `PROPOSITION_FIABILITE_RATTACHEMENT.md` reste **vrai sur le fond** : le composant
existe, il est nommé, il est au registre, et le lien joueur → entité est bien sérialisé. Ce qui est
réfuté, c'est l'estimation du coût : *« aller jusqu'à i21 est un problème de chaîne de composants »*.

Le blocage n'est pas dans `i21`, il est **en amont** : le parcours séquentiel du flux delta ne tient
pas la distance sur ce film. Tant qu'il dérive, aucun composant tardif d'aucun archétype n'est
lisible — `i21` n'est qu'un cas particulier. Réparer cela est un chantier de décodeur à part
entière, pas une étape de ce plan.

### Si l'étape échoue

Repli documenté : nommer chaque vie par **la mort qui la termine**. Déjà mesuré — 91 vies sur 99,
écart médian 0,0 image, gain 147 → 443 tirs, non-régression 125/125. **Ce repli est une inférence**
et devra être marqué comme tel à l'écran.

---

## ÉTAPE 2 — Brancher le rattachement sur la lecture

**Ne commencer que l'étape 1 close.**

### Items

- [x] 2.1 Remplacer la construction par vote de `owners.go` par la lecture. **Fait** — la lecture
      étant celle du repli, l'étape 1 ayant échoué. Nouveaux fichiers `replay/lives.go` (découpage
      des vies, calage d'horloge, nommage par les morts), `replay/bridge.go` (index de tir →
      identité) et `replay/deaths_source.go` (le fil des morts, lu dans le chunk highlight du
      film — aucune base). `buildOwners` prend désormais les morts et leur donne la préséance.
- [~] 2.2 Conserver le vote comme **repli explicite** — **ANNULÉ le 2026-07-28 par l'utilisateur** :
      « je préfère rien afficher que quelque chose de complètement faux ». Le repli est SUPPRIMÉ,
      pas conservé. Ce qui suit décrit l'état intermédiaire, avant ce retrait.
      *(état intermédiaire)* Conserver le vote comme repli explicite et compter combien de slots passent par lui.
      **Fait** : `OwnerReport.FromFallback`. Mesuré sur `000d5950` — **90 slots par la lecture,
      6 par le repli**, sur 96. Le repli ne corrige jamais une lecture : un vote qui contredit un
      fait daté est compté (`Conflicts` = 4), pas appliqué.
- [x] 2.3 Supprimer le code de vote devenu mort. **FAIT le 2026-07-28**, sur décision de
      l'utilisateur. `voteSlotOwners`, `designateShooterSlot`, `voteOwnersFromThrows`, `birthNear`,
      `nearestSlotTo` et leurs quatre constantes de seuil sont supprimées. `OwnerReport` perd
      `FromAim`, `FromThrow`, `FromFallback`, `Conflicts` et sa ventilation : ces compteurs
      n'avaient de sens que pour surveiller un repli qui n'existe plus.
      **Coût mesuré AVANT de supprimer** : tirs 496 → 475, lancers 68 → 63, slots du pont 96 → 90.
      **Gain** : désaccords entre sources 4 → **0**, et tout ce qui est affiché vient d'une lecture.
- [~] 2.4 Vérifier la non-régression — **la question est CLOSE par la suppression du repli**. Les
      24 tirs qui changeaient de propriétaire venaient tous de slots que le vote revendiquait.
      Sans vote, il n'y a plus de seconde méthode pour les revendiquer : ils ne sont plus publiés
      du tout, ni par l'une ni par l'autre. Ce qui suit décrit l'analyse faite avant ce retrait.
      *(analyse antérieure)* Vérifier la non-régression : les tirs déjà publiés gardent leur slot.
      **MESURÉ, ET LE CRITÈRE « 100 % » N'EST PAS ATTEINT** : sur les 397 tirs que les deux ponts
      publient, **373 gardent leur slot, 24 en changent, 3 disparaissent** — soit 94,0 %.
      **Deux arbitrages ont été tentés, aucun ne tranche** :
      *par le loadout des images-clés* (source pleinement disjointe) — 0 pour le nouveau slot,
      0 pour l'ancien, 4 non discriminants, **20 sans loadout lisible** ;
      *par la visée, tir par tir* (source partiellement disjointe : le vote l'agrège en majorité,
      on l'interroge ici par tir) — **2 pour le nouveau, 2 pour l'ancien**, 20 illisibles.
      La lecture est retenue par application de la décision n°1 (« on lit avant d'inférer »), et
      non parce qu'un contrôle l'aurait départagée. **À revoir avec l'utilisateur.**

### Gate

```
cd apps/go-api && go test ./internal/analysis/replay/...       # PASSÉ (+ 6 tests neufs)
cd apps/go-api && golangci-lint run ./internal/analysis/replay/...   # PASSÉ, 0 issue
# couverture : 496 tirs publies / 519 records disponibles = 95,6 %   > 85 %  -> PASSÉ
# non-regression : 373/397 = 94,0 %                                  = 100 % -> NON ATTEINT (2.4)
```

### CE QUI FONDE LA BASCULE, ET CE QUI RESTE FRAGILE

Toutes les mesures viennent de `cmd/tmp_deathnaming`, sur `000d5950`, chacune avec son témoin :

| mesure | valeur | témoin |
|---|---|---|
| vies nommées par la mort qui les termine | 90 / 105 | 10 (morts replacées au hasard) |
| écart d'appariement | médiane 34 ms, maximum 36 ms | — |
| slots changeant de porteur | **0 / 90** | — |
| tirs rattachés (chemin de production) | **496 / 519 = 95,6 %** | 398 par le vote seul |
| arme du tir dans le loadout du slot | **405 / 418 = 96,9 %** | **3,7 %** (autre slot vivant) |

**Le dernier contrôle est celui qui fonde la bascule**, parce qu'il ne partage aucune pièce avec
le rattachement : l'arme vient des records de dégât du flux de trames, le loadout du balayage des
familles dans les records de biped des images-clés. Un rapport de 26× ne s'obtient pas par
construction.

**Ce qui reste fragile, et qu'il ne faut pas maquiller** : la marge du pont index → identité est
étroite — 32 contradictions contre 39 pour la deuxième permutation. Ce n'est pas elle qui porte le
résultat, c'est le contrôle par loadout ; mais sur un film où ce contrôle serait indisponible, la
résolution ne serait pas fondée. `OwnerReport.BridgeCost`/`BridgeSecond` la publient pour que cela
reste visible.

**Correction d'un chiffre du plan** : le vote publiait **398** tirs sur ce film, pas 147. Les 147
dataient d'avant l'ajout de la source « lancers de grenade ». Le gain réel est donc 398 → 496, et
non 147 → 443. Le chiffre de départ du critère de succès (28 %) était périmé.

---

## ÉTAPE 3 — Ne plus rien jeter en silence

- [x] 3.1 `uniqueSlotFor` et ses appelants : remplacer `continue` par un **compteur catégorisé**.
      **Fait** — `uniqueSlotFor` est SUPPRIMÉE et remplacée par `slotFor` (`replay/coverage.go`),
      qui rend sa CAUSE au lieu d'un booléen. Quatre catégories, et la distinction n'est pas
      décorative : « slot introuvable » désigne le chantier du pont, « slot ambigu » celui du
      découpage des vies, « hors fenêtre » celui du décodage des positions, « sans trajectoire
      publiée » n'est pas un échec de rattachement du tout.
- [x] 3.2 Avertissement **échantillonné** au-delà d'un seuil, jamais une ligne par événement.
      **Fait** : `LayerCoverage.warnIfLossy` — un avertissement par calque et par catégorie
      dépassant 10 %, avec le nom du calque pour désigner le chantier.
- [x] 3.3 Exposer les compteurs **dans le résultat**, pas seulement dans les journaux.
      **Fait** : `ReplayDocument.Coverage`, sérialisé dans l'artefact.

### Gate

```
cd apps/go-api && go test ./internal/analysis/replay/...        # PASSÉ (+5 tests neufs)
# sur 000d5950 : 496 + 18 + 5 + 0 + 0 = 519   -> EXACT, aucune fuite.
# grenades : 68 rattachés sur 70 disponibles.
```

**L'invariant est TESTÉ, pas seulement mesuré** : `LayerCoverage.Balanced()` et quatre tests qui
couvrent les cas dégénérés (aucune position, aucun pont, aucun événement) — c'est là que les
fuites passent inaperçues, parce qu'on regarde rarement le cas vide.

---

## ÉTAPE 4 — Publier une couverture

- [x] 4.1 Chaque calque rend `{rattachés, disponibles, rejets par catégorie}`. **Fait** :
      `LayerCoverage`, publiée dans `ReplayDocument.Coverage`.
- [x] 4.2 Le POC affiche ce rapport à côté de ses autres chiffres. **Fait** — bandeau
      « **496 / 519** tirs placés / lisibles », « **68 / 70** lancers », « **nominal** verdict de
      publication », et l'infobulle du verdict ventile les rejets. Les textes explicatifs qui
      citaient encore 147 tirs et 27 lancers sont corrigés dans le même geste.
- [x] 4.3 **Porte de publication** reprise du chantier voisin. **Fait** : `verdictOf` /
      `verdictOfBridge` rendent « nominal », « partiel : … » ou « non publiable : … ». La porte
      REFUSE plutôt qu'elle n'avertit — un calque non publiable doit être retiré de l'écran, pas
      affiché avec une note en bas de page. `BridgeHealth` publie en outre la santé du pont
      lui-même : un calque peut être complet et reposer sur une résolution fragile.

### Gate

```
Le bandeau du POC porte, pour les tirs, N / M avec M = 519 sur ce film.   -> PASSÉ
```
Vérifié dans un navigateur, pas seulement dans le code : `s-sht` = « 496 / 519 », `s-gre` =
« 68 / 70 », `s-verd` = « nominal », zéro erreur JavaScript. Un défaut a d'ailleurs été trouvé
par cette vérification et corrigé — un saut de ligne réel dans une chaîne JavaScript, qui cassait
toute la page sans que le code Go n'en sache rien.

---

## ÉTAPE 5 — Un ordre n'est pas une identité

- [~] 5.1 Renommer le champ de rang d'affichage — **sans objet côté Go** : le `pi` du roster (le
      tri alphabétique) n'existe que dans le bloc de données du POC ; aucun champ Go ne le porte.
      Le renommage utile est celui de 5.2, qui est fait.
- [x] 5.2 Nommer **différemment** l'index porté par le film. **Fait** : `PlayerIndex` →
      `FilmIndex` dans `filmdec.FireEvent` et `filmdec.GrenadeThrow`, propagé à tous les
      appelants (21 fichiers). Le nom porte désormais son statut : c'est un ORDRE valable à
      l'intérieur d'un film, pas une identité. La documentation du champ dit pourquoi.
- [x] 5.3 **Garde-rail** : `internal/archlint/no_player_index_identity_test.go` interdit que
      `PlayerIndex` réapparaisse dans `analysis/filmdec` et `analysis/replay`. Les commentaires
      gardent le droit de nommer l'ancien champ — c'est ainsi qu'ils expliquent le changement.
      **Le garde-rail a été vérifié en le faisant ÉCHOUER** : réintroduire `e.PlayerIndex` dans
      `bridge.go` le fait tomber avec le fichier et la ligne. Un garde-rail qui passe toujours ne
      prouve rien.

### Gate

```
cd apps/go-api && go test ./internal/analysis/replay/... ./internal/archlint/...   # PASSÉ
cd apps/go-api && golangci-lint run ./internal/analysis/replay/...                 # PASSÉ, 0 issue
```
Le lint a trouvé deux fonctions que j'avais écrites sans jamais les appeler (`slotsOfTracks`,
`sortedSlotList`) : supprimées, conformément à la règle « zéro code mort ».

**Règle du dépôt appliquée ici** : à la troisième copie d'un pattern, on centralise **et** on pose
un garde-rail. Une factorisation sans garde-rail re-diverge.

---

## ÉTAPE 6 — Sortir du POC

**Ne commencer que les étapes 1 à 5 closes.** C'est la seule étape qui touche la production.

- [x] 6.1 Inventorier ce que `features/match-replay` porte déjà, et ce que le POC porte en plus.
      **FAIT — et l'écart est le vrai sujet de cette étape.**

      `apps/web/src/features/match-replay/` = **947 lignes**, et **deux calques** :
      `drawGeometryLayer` (les 382 props Forge) et `drawTracksLayer`. Le type
      `ReplayDocument` de `lib/api/types.ts` s'arrête à `geometryBounds`.

      | ce que le Go PRODUIT déjà | ce que le web CONNAÎT |
      |---|---|
      | tracks, bounds, frameInterval | oui |
      | geometry (382 props) | oui |
      | **structure (10 223 emprises)** — le vrai fond de carte | **non** |
      | **shots (496)** | **non** |
      | **coverage + verdicts** (livré à l'étape 4) | **non** |
      | **loadouts (150)** | **non** |
      | **grenades (68)** | **non** |
      | **projectiles (439)** | **non** |

      Et le POC porte en outre des calques que le Go **ne produit pas encore** : fil des
      éliminations, médailles, objectifs, dispositifs de carte, états d'inventaire, callouts,
      roster. Ceux-là ne sont pas un portage : ce sont des décodages à productionniser.

      **Conséquence pour le chiffrage** : 6.2 à 6.5 ne sont pas « brancher deux ou trois champs ».
      C'est étendre le contrat d'API, le type TS, les calques de rendu, l'i18n FR/EN, les jetons de
      couleur et les tests des trois couches. À faire dans sa propre session.
- [!] 6.2 Porter les calques dans l'architecture du dépôt. **NON COMMENCÉ** — arrêt propre décidé
      après 6.1 : le périmètre mesuré dépasse largement ce qui restait de session, et un portage
      partiel laisserait la production dans un état pire que l'actuel (un `ReplayDocument` qui
      porterait la moitié des calques, sans que rien ne dise lesquels).
- [x] 6.3 Côté web : query keys, strings FR **et** EN, jetons de couleur. **FAIT** — les query
      keys passaient déjà par `lib/query/keys.ts` ; onze chaînes ajoutées en FR et EN avec parité
      par typage ; aucune couleur en dur (`tokenCssVar` sur `success` / `warning` / `destructive`).
- [x] 6.4 Brancher sur une **capability**, jamais sur un slug de titre. **FAIT — et il n'y avait
      rien à corriger** : le handler ne compare aucun slug, et la feature « s'allume par présence
      d'artefact, pas par flag global ». C'est la forme la plus simple d'activation par capacité :
      un titre qui ne produit pas d'artefact n'expose pas la route. Le garde-rail
      `no_slug_comparison_test.go` couvre déjà tout `internal/`.
- [x] 6.5 Tests à chaque couche. **FAIT** : logique de couverture testée sans rendu (12 tests),
      garde local testé au niveau HTTP (`replay_local_gate_test.go` — dont un test qui vérifie
      qu'un en-tête `X-Forwarded-For` ne contourne PAS le garde), et la suite web complète reste
      verte (192 fichiers, 1 755 tests).
      *(détail de la première passe)* la logique de couverture est testée sans rendu
      (`coverageLogic.test.ts`, 12 tests portant sur des PROPRIÉTÉS — un taux qui ne vaut jamais 1
      sur un calque vide, une somme qui doit tomber juste, un pont qui n'est « lu » que s'il l'est
      entièrement). Le rendu des calques non portés n'a pas de test, faute de rendu.

### Gate

```
make check-types                                      # PASSÉ
make test-web                                         # PASSÉ — 1 755 tests, 192 fichiers
cd apps/go-api && go test ./internal/analysis/replay/... ./internal/archlint/...   # PASSÉ
```
**Le gate porte sur ce qui EST livré.** `make go-api-test` et `make go-api-lint` complets n'ont
pas été relancés sur toute la suite : le périmètre touché (replay, archlint, web) l'a été.

### LE GARDE LOCAL — posé sur demande de l'utilisateur

« On a des écarts entre la réalité et ce qu'on sait sortir. […] ok pour sortir une version
activée que sur le localhost. »

`internal/api/handlers/replay_local_gate.go` refuse toute requête non locale (404, même message
que l'absence d'artefact — un client distant n'apprend rien de l'existence de la route).

**Ce n'est pas un interrupteur « pour plus tard »**, et il porte les trois éléments que la règle
du dépôt exige d'un kill-switch :

| | |
|---|---|
| basculé le | 2026-07-28 |
| retrait cible | à la première confrontation réussie sur un SECOND film |
| critère mesurable | couverture des tirs > 85 % **et** `coverage.verdict.bridge` nominal sur deux films de cartes différentes, sans collision de trace |

Le critère se lit **dans l'artefact lui-même** : aucun jugement n'est requis pour décider du
retrait. `LEVELUP_REPLAY_PUBLIC=1` lève le garde sans recompiler.

**L'adresse de la connexion fait foi, jamais un en-tête** : `X-Forwarded-For` est fourni par le
client, s'en servir transformerait le garde en suggestion. Un test le verrouille.

### CE QUE LE POC A DE FRAGILE EN ATTENDANT

Le POC vit dans le **scratchpad de la session qui l'a produit**, dont le chemin change à chaque
session. Il n'est pas versionné (4 Mo, données incluses). C'est précisément ce que cette étape doit
supprimer. En attendant, `ETAT_DU_POC.md` porte la commande pour le retrouver et les deux commandes
pour le remettre à jour sans l'éditer à la main.

---

## BLOCAGES CONNUS, ET CE QU'ILS N'EMPÊCHENT PAS

| blocage | portée réelle |
|---|---|
| Position des objets au sol non décodable (`ti=42`, `ti=37`) | n'empêche aucune étape de ce plan |
| Palette de capacités partielle (4 index sur 11) et non globale | n'empêche pas le rattachement ; c'est un chantier séparé |
| Le film ne porte que les dégâts, pas les tirs manqués | **plafond dur** : 519 records pour 2 228 tirs. Le critère de succès est calé dessus, pas sur 100 % |

---

## DÉCOUVERTES — à consigner ici, pas à traiter

Toute découverte hors périmètre se note ici et **n'est pas corrigée** (règle du contrat
d'exécution), sauf si elle bloque le gate de l'étape en cours.

- **[étape 1] Le parcours séquentiel du flux delta ne tient pas la distance.** Sur `000d5950`,
  il ne rencontre que 125 records ti=5, en désynchronise 47 %, et sert des slots que l'image-clé
  ne lie à aucun ti=5 (1042, 1940, 256). Les masques de présence lus n'ont aucune structure
  (`i0` présent 36 fois, `i20` 48, `i24` 45, sur les mêmes 125 records d'un archétype unique).
  C'est cohérent avec la réserve déjà écrite dans `frame_records.go` — *« il reste au moins une
  faute dans le CORPS des records »* — et avec celle de `keyframe_loadout.go` — *« le masque lu
  n'est pas le vrai masque »*. **Portée** : plafonne tout composant de rang tardif, sur tous les
  archétypes. Chantier de décodeur à part entière. **Non traité ici.**
- **[étape 1] La grammaire des records d'image-clé n'est pas celle des records NEW.** Vérifié :
  aucune largeur de default-state entre 0 et 260 bits ne fait atterrir la chaîne de composants
  sur la borne du record suivant, pour aucun des 832 records ti=5 (`-mode kfchain`). L'en-tête
  `[id:32][field:26][ti:6]` est établi, la suite ne l'est pas. **Non traité ici.**
- **[étape 1] La forme `R(32)` du port d'`i21` est douteuse.** `traverse.go:655` porte
  `player-representation-component` comme un `R(32)` brut (FUN_14111ec64). Or aucun mot de
  32 bits des records ti=5 ne vaut un handle de biped, et le format écrit ses handles ailleurs
  en `R(idLowBits)+R(2)` (~13 bits). Si la largeur est fausse, elle désynchronise tout ce qui
  suit `i21` dans l'archétype 5. À rouvrir **avec le désassemblage**, pas par balayage — et hors
  de ce plan. **Non traité ici.**

---

## DEUX POINTS EN ATTENTE D'ARBITRAGE — proposés le 2026-07-28

### POINT 1 — les 24 tirs qui ont changé de slot — **RÉGLÉ le 2026-07-28, par suppression**

**L'utilisateur a tranché** : « pourquoi mettre le vote en secours alors qu'on sait que c'est pas
fiable ? Je préfère rien afficher que quelque chose de complètement faux ». Le repli voté est
supprimé. Les 24 tirs disparaissent avec lui — ils venaient tous de slots que seul le vote
revendiquait. Coût : 21 tirs et 5 lancers en moins. Gain : zéro désaccord, et 100 % de ce qui est
affiché vient d'une lecture. **La proposition ci-dessous (les marquer à l'écran) est donc caduque**,
elle est conservée pour mémoire du raisonnement.

*(proposition antérieure, caduque)*

**L'état des lieux.** Sur les 397 tirs que les deux ponts publient, 373 gardent leur slot, 24 en
changent, 3 disparaissent. Le gate exigeait 100 % ; c'est 94,0 %. Deux arbitrages ont été tentés,
**aucun n'a tranché** — par le loadout (0 contre 0, 20 sans donnée lisible), par la visée tir par
tir (2 contre 2, 20 illisibles).

**La mesure qui manquait, faite depuis, et qui change la lecture** : les 4 slots en désaccord se
ventilent en **1 désaccord avec la visée** et **3 avec le lancer de grenade**. Ce n'est pas
équivalent :

- la **visée** est une inférence — comparaison d'angles, lisible dans ~44 % des records ;
- le **lancer** porte l'index de joueur **écrit en clair** dans le film… mais son rattachement à un
  SLOT reste une inférence géométrique (le biped le plus proche de la naissance du projectile,
  médiane 0,77 unité, seuil 2,5).

Donc les 3 désaccords sérieux n'opposent pas « deux lectures » : ils opposent deux chaînes qui ont
chacune une part lue et une part inférée. Aucune n'est disqualifiée d'avance.

**CE QUE JE PROPOSE — les publier, mais les MARQUER, et mesurer la part inférée du lancer.**

1. Ne pas les jeter. Les jeter reviendrait à faire confiance à un vote dont on sait qu'il ne
   couvrait que 26 slots sur 99, contre une lecture contrôlée à 96,9 % par source disjointe.
2. Ne pas les peindre comme les autres. La frontière posée par ce plan est claire : *ce qui est lu
   se peint franchement, ce qui est déduit se peint autrement.* Un tir dont deux sources se
   contredisent n'est pas au même niveau de preuve que les 472 autres — il doit le montrer, et
   `Coverage` doit le compter dans une catégorie propre (`contested`).
3. **La mesure qui trancherait vraiment**, et qui reste à faire : pour ces 3 slots, la naissance du
   projectile était-elle NETTE (proche de 0,77) ou LIMITE (proche du seuil de 2,5) ? Si elle est
   limite, l'inférence géométrique du lancer est faible et la lecture gagne sans réserve. C'est une
   mesure courte, sur trois cas, et elle vaut mieux qu'un arbitrage de principe.

**Coût** : faible (une catégorie de couverture, un style d'affichage, une mesure sur 3 cas).

### POINT 2 — le chiffre périmé (147 contre 398)

**Ce qui s'est passé.** Le plan et le diagnostic annonçaient « 147 tirs publiés, soit 28 % ». La
mesure réelle du vote, refaite sur le chemin de production, donne **398 tirs, soit 77 %**. Les 147
dataient d'avant l'ajout de la source « lancers de grenade » au vote. Le défaut a donc été présenté
comme trois fois plus grave qu'il ne l'était.

**Ce qui n'est PAS remis en cause** : la direction du diagnostic (le vote était bien le goulot), le
critère de succès (> 85 %, atteint à 95,6 %), ni le gain (398 → 496, plus 28 lancers).

**CE QUE JE PROPOSE — une règle, parce que le cas se reproduira.**

1. **Tout chiffre publié dans un document `.ai/` porte sa date ET l'outil qui l'a produit.**
   « 147 tirs » aurait dû s'écrire « 147 tirs (`cmd/replay-build`, 2026-07-2x) ». La date seule ne
   suffit pas : c'est le nom de l'outil qui permet de refaire la mesure au lieu de la croire.
2. **Un chiffre qui fonde une décision se re-mesure avant la décision, jamais après.** Ici la
   re-mesure a eu lieu pendant l'exécution, ce qui est déjà tard ; elle aurait dû avoir lieu à la
   rédaction du plan, où elle aurait changé la formulation du problème.
3. Les documents portant l'ancien chiffre sont corrigés : le plan, `SUIVI_REPLAY_2D.md`,
   `PROPOSITION_FIABILITE_RATTACHEMENT.md` (par encart daté, sans réécrire le diagnostic — un
   diagnostic réécrit après coup ne s'évalue plus) et les textes du POC.

**Coût** : nul pour le passé (fait), faible pour l'avenir (une convention d'écriture).

---

## SUITE DU 2026-07-28 (soir) — CE QUE L'UTILISATEUR A REDRESSE, ET CE QUE LA MESURE A TRANCHE

### Le player index SE LIT — l'etape 1 abordait le mauvais objet

**L'utilisateur** : « ca fait des mois qu'on connait comment lire le player index, l'autre projet
le fait tres bien. Tu dois te tromper sur la difficulte ou aborder avec le mauvais angle. »

Il avait raison. Le chantier voisin (`filmdec-killweapon`, lot `co.pi`) a etabli que le lien
identite -> index de joueur **se lit** : le xuid figure dans le film sur 8 octets petit-boutiste,
et **les cinq bits qui le precedent portent l'index**. La fonction vivait deja dans ce depot
(`weaponv3.ResolveXuidToPI`) sans avoir jamais ete branchee sur le rejeu.

| mesure sur `000d5950` | valeur |
|---|---|
| chunks de replication donnant la MEME table | **26 sur 26** |
| desaccords | **0** |
| table obtenue | **identique** a celle que l'affectation de cout minimal produisait |

Le piege documente par le voisin est reproduit : le chunk 0 (registre) et celui des highlights
rendent **0 pour les huit xuids**. Un balayage indifferencie ecrase donc la bonne table — c'est
probablement ce qui avait fait croire que la methode ne marchait pas.

**Le pont n'a plus aucune part de choix** : le fil des morts nomme chaque vie par le xuid de sa
victime, les cinq bits donnent l'index de chaque xuid. `bridge.go` et `bestAssignment` supprimes.

**PRECISION DE VOCABULAIRE, demandee par l'utilisateur** : le film distingue le **joueur**
(archetype 5, l'emplacement — 32 dans le film, 8 occupes) du **biped** (archetype 35, le corps qui
court, qui CHANGE a chaque reapparition). L'etape 1 cherchait le lien joueur -> biped, pas le
player index. Ne plus ecrire « personnage ».

### LE LIEN JOUEUR -> BIPED N'EST PAS RESOLU CHEZ LE VOISIN — verifie le 2026-07-28

**L'utilisateur** : « on avait resolu le biped sur l'autre worktree je crois ».

**Verifie, et c'est l'inverse qui est ecrit.** `RE_LOG_KILLWEAPON.md` §8.3, mot pour mot :

> **entity-id d'un biped** = `0x4000xxxx` et **CHANGE a chaque respawn**
> (ne pas s'en servir comme id joueur).

Ce que le voisin a resolu, c'est le **player index** — « stable par joueur », handle
`0xE1500000 + idx*0x10002` — et c'est ce qui a ete branche ici (cf. `player_index.go`). Le lien
joueur -> biped, lui, n'existe chez eux ni comme acquis ni comme piste : leur pipeline n'en a pas
besoin, parce qu'il attribue des ARMES a des kills, pas des positions a des traces.

**Ce que le voisin a en plus, et qui reste a porter si le besoin revient** :

| chez le voisin | ce que c'est | statut ici |
|---|---|---|
| `resolveXuidPIStrict` | version DURCIE du resolveur d'index : exige des indices deux a deux distincts. 935/949 films a solution unique | **non porte** — notre lecture suffit sur ce film (26 chunks concordants, 0 collision), mais elle est naive |
| roster `type-8` (`solveRoster`) | les xuids dans l'ORDRE des slots, valide 8/8 sur `000d5950` | present dans `cmd/tmp_kwval` seulement |

⚠ **Le piege que le voisin signale nous concerne** : l'ancrage NAIF (`weaponv3.ResolveBest`,
premiere occurrence) rend l'indice **0 pour les huit joueurs** de ce film. Notre lecture l'evite
en n'interrogeant que les chunks de replication et en exigeant leur concordance — mais si un film
faisait diverger deux chunks, il faudrait porter `resolveXuidPIStrict`.

### Les lancers de grenade : 63 -> 70 sur 70

**L'utilisateur** : « En quoi ca pose probleme que le joueur n'a aucune vie identifiee pour le
lancer de grenade ? »

Defaut de conception, pas limite des donnees. On situait le lancer a la position du BIPED du
lanceur — donc il fallait connaitre son biped, donc le pont, donc l'echec quand la vie n'etait pas
nommee. Or **le lancer fait naitre un projectile dont la position est decodee et dont le premier
point est la main du lanceur**. La position etait deja la. **70 lancers sur 70** desormais situes.

### DEUX PISTES QUE J'AI PROPOSEES, ET QUE LA MESURE REFUTE

**Elargir la fenetre d'appariement (chutes)** — refutee. Le temoin monte dix fois plus vite que
le signal :

| fenetre | apparies | temoin (morts au hasard) | gain net |
|---|---|---|---|
| 150 ms | 90 | 5 | **+85** |
| 500 ms | 91 | 15 | +76 |
| 2 000 ms | 92 | 39 | +53 |
| 7 000 ms | 92 | 72 | +20 |

On gagnerait **deux vies** pendant que le temoin passe de 5 a 72 — a 7 s, l'appariement marche
presque aussi bien avec des morts placees au hasard. **La fenetre actuelle est la meilleure.**

**Chainer les reapparitions** — refutee. **0 orpheline sur 15 a un seul candidat** ; les autres en
ont 2 a 6. Il faudrait choisir : c'est exactement le vote que ce chantier a retire.

### LES 15 VIES NON NOMMEES, VENTILEES

| combien | quoi | recuperable ? |
|---|---|---|
| 4 | avant le debut reel du match | **non — a ignorer**, valide par l'utilisateur |
| 6 | survivants de fin de match | le film ne porte AUCUN evenement de fin (0 event de mode) |
| 5 | orphelines du milieu, dont 3 face a une mort du fil a 433 / 1 118 / 6 090 ms | non par la fenetre (cf. ci-dessus) |

**Pour les 6 survivants, la fin de match ne suffit pas** : savoir QUAND le match finit ne dit pas
QUI est chaque survivant. La piste qui reste ouverte est le **compte de morts par joueur** (l'API
le porte) : un joueur credite de 11 morts a 11 vies nommees, sa 12e se deduit par elimination.
C'est deterministe si les comptes concordent — **non mesure a ce jour**.

---

## PROTOCOLE DE REPRISE

1. Relire le contrat `plan-execution`.
2. Lire ce fichier, puis `SUIVI_REPLAY_2D.md`.
3. Reprendre à la **première case non statuée** de l'étape courante.
4. Ne pas re-décider ce qui est tranché plus haut.

**Statuts autorisés** : `[x]` fait et vérifié · `[~]` couvert ailleurs, avec la référence ·
`[!]` non traité, avec justification écrite. Aucune case vide à la clôture d'une étape.
