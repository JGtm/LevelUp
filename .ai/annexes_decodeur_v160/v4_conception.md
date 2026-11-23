# Lecture au choix des calques : conception sans code

Tête lue : `daac986` (branche `feat/v1.6.0`). Les chemins sont relatifs au dépôt du décodeur de film, sauf mention. Aucun code écrit, aucun film décodé.

Marques : **vérifié** = ligne relue (fichier:ligne). **déduit** = raisonné sans preuve ; on dit alors ce qui le vérifierait.

Mot employé : « calque » = un champ racine du document de rejeu. Son nom est la balise JSON (`replay/layers.go:18-24`, vérifié).

## 0. Résumé

1. La table « quel calque, quelle couche » existe déjà (`replay/layers.go:129`, 62 entrées) et le document dit déjà « ce calque n'a pas été produit » : une entrée absente de `layers` est une réponse (`layers.go:79-87`, `328-348`, vérifié). Aucun champ neuf n'est nécessaire.
2. Il manque la table des **dépendances** : calque -> balayages -> canaux de la marche. Elle se construit à partir des commentaires `<- FilmInputs.X` de `layers.go` et de la liste fermée des étapes (`replay/observe.go:47-93`).
3. **Fait qui change le cadrage.** Le document n'existe que s'il porte des positions : sans position, l'assemblage rend « publication seule » (`replay/build.go:82-85`, `layers.go:318-327`, vérifié). Tout calque demandé entraîne donc les positions, et les positions, pour garder leurs octets (noms des vies compris), entraînent l'identité : index des joueurs, table des sièges, équipes, fil des morts, créations, tirs (`replay/build_pistes.go:44-57`, vérifié). Cette fermeture pèse environ 73 % du temps mural (déduit, § 3).
4. Gain réaliste : **de 22 à 30 % du temps mural** pour toute sélection, quelle qu'elle soit, parce que la fermeture est presque la même. Le fil des morts seul ou le score seul n'y gagnent pas plus que les positions seules (§ 3).
5. Aucun appelant de production de LevelUp ne choisirait aujourd'hui une sélection : les lectures partielles passent déjà par des entrées dédiées. Le seul appelant partiel du document est `PortagesAuSync` (collecteur), qui compose déjà ses lectures à la main (§ 3.3).
6. Avis : **ne pas faire ce chantier pour la 1.6.0** (§ 7). Il recoupe les items de parallélisme `J6` et `J7` sur les mêmes fichiers, pour un gain que le parallélisme donne déjà à tous les appelants sans changer d'API.

## 1. Les calques et leurs dépendances

### 1.1 Les étapes de balayage (groupes)

Ordre d'appel : `replay/build_from_film.go:307-315` (vérifié). Parts : `bench/recents/recents.md` table A, moyenne des cinq films récents, en % du temps mural.

| Groupe | Contenu (étapes observées) | Source de l'appel | Part |
|---|---|---|---|
| G0 | chargement, contexte, profil, carte | `build_from_film.go:97-101` | 0,8 |
| M | marche des trames : bande de slots bipèdes (12,3), images-clés de la marche (14,9), décodage des paquets (16,0), crochets des canaux (6,5), clôture (1,1), reste (0,4) | `film_scan_mouvement.go:111-113` | 51,2 |
| P | pont d'identité : translocations, positions, créations, temps forts (fil des morts, médailles), index des joueurs, origine d'horloge | `film_scan_pont.go:45`, `internal/grammar/pont_identite.go:101-127` | 10,0 (index 6,6 ; positions 1,9 ; temps forts 1,0 ; translocations 0,6) |
| F | tirs `ScanFireEvents` | `film_scan.go:67` | 0,6 |
| K | poussées `ScanKnockbacks` | `film_scan.go:53` | 0,6 |
| E | états des images-clés : armes portées (`loadouts`), inventaire, marques de portage | `film_scan_images_cles.go:199-211` | 7,9 |
| B | dotations de naissance | `film_scan_naissances.go:138` | 0,3 |
| W | monde : lunette, poses, apparitions, socles, véhicules ; puis gardés : marque du drapeau, `ti=13` (zones, jauge), anneau de la bombe | `film_scan.go:341-399`, `film_scan_ti13.go:199` | 5,9 |
| H | portage : armes tenues, ramassages natifs, inventaire par deltas | `film_scan.go:132-171` | 0,6 |
| A | capacités : rangs, changements d'équipement, équipement porté, camouflage, grappin, impulsions, charges | `film_scan.go:228-337`, `film_scan_equipement_porte.go:168` | 1,6 |
| V | publication de la marche : états, rafales, commandes, jeu, dégâts, rechargements, horloges, morts de véhicule | `film_scan_mouvement.go:37-96` | 0,2 |
| X | grenades, projectiles, détonations | `film_scan_pont.go:99-117` | 1,2 (déduit : 0,4 + 0,6 + reste) |
| S | sièges et occupants : table du film aux listages, équipes, entrées du tableau des scores | `film_scan_pont.go:81-92`, `128-144` | 9,4 |
| Q | assemblage du document | `build.go:82-109` | 9,3 |

Total des parts : 99,7 %. Écart d'arrondi.

Dépendances entre groupes (vérifié sauf mention) :

- P dépend de M : les positions et créations passent par les lectures bipèdes que la marche recueille (`film_scan_mouvement.go:25-28`, `internal/grammar/lectures_bipedes.go:169-176`).
- B dépend des créations de P (`film_scan_naissances.go:139`). H dépend de E et B (`film_scan.go:139-140`). A dépend des positions de P (`film_scan.go:249`, naissances des vies). V dépend de W pour les morts de véhicule (`film_scan_mouvement.go:71`). Les marques de portage dépendent de E (`film_scan.go:383`).
- Le graphe plus fin des balayages, sans l'arc « états -> naissances » (non fondé), est en `k/conception.md` § 2.1 (vérifié par l'auteur de ce document, relu ici).

### 1.2 Les canaux de la marche

`marcherPourLaCuisson` (`internal/grammar/movement_states.go:196-260`, vérifié) inscrit : états de mouvement + commandes de contrôle + suppressions de bipède (`sc`), tir continu + commandes tenues, événements de jeu, dégâts, vitalités, rechargements, horloges, propriétés gérées de `ti=13` ; et, **seulement si le contexte ne les tient pas déjà**, les lectures bipèdes, les objets du monde, l'équipement porté (`movement_states.go:218-226`). Les morts d'objet s'ajoutent si le film porte des véhicules (`:232`, `marche_commune.go:33-35`).

### 1.3 La table calque -> dépendances

Un calque porte la ligne de `layers.go:129-210`. Je la prolonge par les groupes. Les calques de même ligne se traitent ensemble.

| Calques (balise JSON) | Lu dans le film (groupes, canaux) | Entrées de l'appelant (hors film) | Preuve |
|---|---|---|---|
| `schemaVersion`, `matchId`, `titleSlug`, `frameIntervalMs` | aucun | argument | `layers.go:133-136` |
| `frameCount`, `durationMs`, `bounds`, `originMs`, `tracks` | P (positions, créations, origine), M (lectures bipèdes, `sc` : commandes de contrôle et suppressions pour la fin du corps) | ; | `layers.go:140-144` ; `tracks_fin_par_commande.go` |
| `identity`, `roster` (et les noms portés par `tracks`) | P (index, fil des morts, créations), S (table du film, équipes, entités), F (références de tirs), M | bots, entrées du tableau des scores, participants, xuids du roster | `build_pistes.go:44-57` ; `layers.go:193-194` ; **déduit** : les noms de `tracks` dépendent du registre entier, à vérifier par l'égalité d'octets de `tracks` sous sélection |
| `t0FilmMs`, `roundClock`, `respawnCountdowns` | M (canal des horloges) | ; | `layers.go:150, 175-181, 258-262` |
| `shots`, `shotStats`, `hits`, `damage` | F ; M (canal des dégâts, vitalités) ; P pour l'attribution | ; | `layers.go:145, 168-170` |
| `bursts` | M (tir continu) | ; | `layers.go:146` |
| `loadouts`, `weaponLabels` | E, B | catalogue de libellés | `layers.go:147` |
| `projectiles`, `grenades` | X | ; | `layers.go:148-149` |
| `inventory`, `grenadeReads`, `abilities`, `abilityLabels` | E, H, A (rangs) | ; | `layers.go:150-152, 156-161` |
| `equipmentChanges`, `carriedEquipment`, `riftPoses`, `abilityImpulses`, `abilityCharges`, `grappleLines`, `equipmentPlacements`, `equipmentEpisodes` | A, W (poses), P (naissances) | `MapQuant` pour le grappin ; faits de morts de l'appelant pour `equipmentEpisodes` | `layers.go:153-165, 194` |
| `translocations`, `knockbacks` | P (translocations), K | ; | `layers.go:152-153` |
| `weaponChanges`, `pickups` | H, E, B | ; | `layers.go:166-167` |
| `medals`, `scoreActions`, `eliminations` | P (temps forts), M (jeu), scripts du chunk 0 | table de score de l'appelant | `layers.go:168-169, 198` |
| `weaponPads`, `padPickups`, `groundWeapons`, `objectiveObjects` | W (poses, socles) | catalogue de socles | `layers.go:172-175` |
| `vehicles`, `vehicleCycles` | W (véhicules), V (morts de véhicule) | ; | `layers.go:176-177` |
| `stances` | M (`sc`, états de mouvement) | ; | `layers.go:178` |
| `reloads` | M (rechargements) | ; | `layers.go:179` |
| `rules` | G0 (identité du film, section 2 de `chunk_00`) | ; | `layers.go:181` |
| `zoneStates` | W (`ti=13`) | catalogue de zones | `layers.go:182` |
| `neutralDeaths`, `objectives`, `scoreTimeline`, `flagCarries`, `flagReturnZone`, `vipCrown`, `skullCarries`, `bombArmings`, `bombCarries`, `bombStats`, `bombEvents` | W (marque du drapeau, anneau de la bombe, `ti=13`) pour une part ; le reste vient de l'appelant | faits du tableau des scores et du fil de dégâts, garde de mode | `layers.go:190-207` |
| `geometry`, `geometryBounds`, `structure`, `structureBounds`, `killEffects`, `grenadeLabels`, `coverage`, `layers` | aucun calque : catalogue ou mesure | ; | `layers.go:217-253` |

Remarques :

- Les sept calques à la requête (`mapObjectives`, `mapWeaponPads`, `weaponTiers`, `vehicleLabels`, `medalLabels`, `vehicleWeapons`, `vehicleScenery`) ne sont pas cuits : hors sélection (`layers.go:92-109`, vérifié).
- Une entrée de `gardesDeProduction` (`layers.go:256-293`) montre qu'une garde par calque existe déjà et qu'elle est lue au même endroit que la table. La sélection s'y ajoute comme une garde de plus : c'est le point d'entrée qui évite tout champ neuf.
- `coverage` n'est pas un calque (`layers.go:217-222`). Cinq de ses 51 champs ne sont pas `omitempty` : `shots`, `grenades`, `objectives`, `originResolved`, `bridge` (`replay/coverage.go`, `awk` sur la structure, vérifié). Dans un document partiel ils portent des zéros sans entrée dans `layers` : la lecture correcte est « `layers` d'abord ». La promesse « mêmes octets » ne vaut donc **pas** pour `coverage` (voir 3 de § 3.4).

## 2. Ce qui est incompressible

Mesure : `recents.md` table A (vérifié sur le fichier ; les films sont ceux du banc, un seul poste).

| Poste | Part | Pourquoi incompressible |
|---|---|---|
| Chargement | 0,8 % | le film doit être lu |
| Marche : bande de slots, images-clés, décodage des paquets, clôture, reste | 44,7 % | les bits de chaque composant sont lus en entier, même quand aucun canal ne les veut : les crochets ne font que publier (vérifié sur trois sites : `grammar/components_game_engine.go:76-81`, `components_managed_property.go:138-143`, `ability_energy.go:51`) ; la longueur d'un record ne se connaît qu'en le lisant |
| Crochets des canaux | 6,5 % | divisible par canal ; la part des canaux nécessaires (lectures bipèdes, `sc`) n'est pas mesurée par canal : **déduit** 3 % incompressibles |
| Positions, temps forts, translocations du pont | 3,5 % | `frameCount` n'existe qu'avec des positions (`build.go:82-85`) |
| Assemblage minimal (ouvrir, porte, pistes) | environ 4 % | **déduit** de `recents.md` table C (porte 2,2 ; pistes 1,6 ; ouvrir 0,35 en % CPU) |
| **Sous-total, sans identité** | **environ 56 %** | |
| Identité pour garder les noms de `tracks` : index (6,6), sièges et équipes (9,4), tirs (0,6) | 16,6 % | le registre d'identité consomme tout cela (`build_pistes.go:44-57`) ; **déduit**, à prouver par l'égalité d'octets de `tracks` |
| **Total incompressible pour tout calque autre que la publication** | **environ 73 %** | |

Ce que la marche ne peut pas ne pas lire : l'identité de la table de contrôle, les lectures bipèdes, `sc` (commandes de contrôle et suppressions de bipède, qui ferment les vies de `tracks`, `layers.go:142`).

## 3. Gain par sélection

Tous les gains sont **déduits** (soustraction des parts de `recents.md` ; à remesurer avec `film-bench` sur les films du banc après l'item S6).

Part évitée = états des images-clés 7,9 + monde 5,9 + capacités 1,6 + portage 0,6 + poussées 0,6 + naissances 0,3 + publication de la marche 0,2 + grenades, projectiles, détonations 1,2 + assemblage non demandé environ 5 (9,3 - 4,3) = environ 23 points ; plus, si les canaux de la marche sont retirés aussi, environ 3,5 points de crochets.

### 3.1 Sélections types

| Sélection | Ce qu'elle garde | Gain estimé (temps mural) | Remarque |
|---|---|---|---|
| Positions seules (`tracks`, `bounds`, `originMs`, `frameCount`) | M, P, S, F + identité | -23 % (balayages et assemblage), -27 % avec élagage des canaux | la fermeture d'identité est la moitié du travail restant |
| Tirs seuls (`shots`, `bursts`) | idem + canal du tir continu | -22 à -26 % | `shots` est rattaché par le registre : même fermeture que les positions |
| Fil des morts seul | P (temps forts) | document : -23 % ; **entrée dédiée sans document : environ -97 %** | le fil des morts n'est pas un calque du document ; il existe déjà `ScanDeaths` / `ScanPontDIdentite` (vérifié, `decfilm/decfilm.go:196-199`). La sélection n'apporte rien de plus |
| Tableau de score seul (`scoreTimeline`, `eliminations`, `objectives`) | positions + jeu + scripts + temps forts | -20 à -24 % | les enregistrements du tableau des scores viennent de l'appelant, lus par ses propres entrées (`statborgDuFilm`) ; ils ne passent pas par le décodage de film du document |
| Armes et capacités (`loadouts`, `weaponChanges`, `abilities`) | positions + E, B, H, A | -8 à -10 % | peu de gain : ces groupes sont ce qui reste |

Le gain est plat parce que 73 % du travail est la fermeture commune (§ 2).

### 3.2 Les appelants de LevelUp

Source : `.ai/annexes_decodeur_v160/appelants_partiels_0_7.md` du dépôt de l'application (tableaux A et B, relu).

| Appelant | Entrée aujourd'hui | Gagnerait avec une sélection ? |
|---|---|---|
| Les cinq appelants de la cuisson complète (tableau B) | `BuildFromFilmAvecFaits` | non : il leur faut tous les calques |
| Fil des morts (A2), temps forts (A17 à A20), version (A16, A21), clé du film (A8, A10), bots (A4), statborg (A5, A6), tirs bruts (A12, A13), pont du collecteur (A14), véhicules (A15) | entrées dédiées, hors du document | non : déjà partiels |
| Décodage de la source de dégât (A3, A11) | `decfilm.Decode` (killsource) | non : autre marche (partagée si le contexte l'est, `marche_commune.go:89-106`) |
| `PortagesAuSync` (collecteur du sync) | `replay.PortagesAuSync`, `replay/porteurs_au_sync.go:71-77` | **oui, seul candidat** : il compose à la main ses lectures puis appelle `BuildFromPositions` en `documentInterne` ; il lit déjà « la garde rend avant le premier octet » (`porteurs_au_sync.go:30-34`). Cette ligne manque au tableau A de l'inventaire : à y ajouter |

Conclusion : **zéro appelant de production** n'utiliserait une sélection sans l'avoir écrite pour lui, et le seul candidat réel (`PortagesAuSync`) est déjà sélectif par ses gardes de mode (drapeau, crâne, VIP, bombe).

### 3.3 Précédents dans le code

- Garde de lecture par l'appelant : `GardesDeCuisson` (`replay/gardes_de_cuisson.go`), `drapeauBalayable`, `zonesBalayables`, `bombeBalayable`. Le balayage lit ou non selon la garde ; les faits disent ce qui a été lu.
- Sélection à l'intérieur d'une passe : `ScanArmesDesImagesCles` (`internal/grammar/keyframe_etats_scan.go:79`) lit « les armes seules ».
- Sur-ensemble de faits : `FilmFactsEntete.Utilisable` / `restreindreAuxGardes` (`filmfacts_entete.go:113`, `build_from_facts.go:64`) : des faits cuits sous plus de gardes servent une demande moindre.

### 3.4 Sûreté du modèle de sélection

1. **Les calques demandés gardent leurs octets** : à prouver, jamais à supposer (§ 6, S6).
2. Un calque absent est une réponse : la table `layers` le dit déjà (`layers.go:79-87`).
3. `coverage` ne tient pas la promesse : ses comptes de replis viennent de canaux (voir § 4). Elle est hors égalité (`calquesSansCouche`, `layers.go:217-222`).

## 4. Ce que la marche peut ne pas distribuer

Question : un canal non inscrit change-t-il ce que lisent les autres ?

**Vérifié**

- Les crochets ne font que publier ; les bits sont lus avant l'appel (trois sites ci-dessus). Retirer un canal ne déplace pas le lecteur de bits.
- Les crochets sont fondus sans ordre (`distribuer_crochets.go:36-48`) : un crochet posé deux fois est refusé, pas écrasé. Les canaux reçoivent la trame dans l'ordre de la liste (`internal/grammar/distribuer.go:256-262`) ; garder l'ordre relatif des canaux restants garde le comportement.
- **Piège de repli** : un lecteur dont le canal n'est pas inscrit lance **sa propre marche** (`lectures_bipedes.go:169-176`, `objets_du_monde_lus.go:202-216`, `equipement_porte.go:305-310`). Retirer le canal sans retirer le lecteur double le coût au lieu de le réduire. La clôture des dépendances doit donc retirer les deux ensemble, et un test doit interdire toute marche après la marche principale (`TestAucunDistribuerApresLaMarche`, déjà prévu à l'item J4 de `k/conception.md`).
- **Compteurs de repli lus à travers un canal** : `m.LiaisonsParRepliDAnticipation` et `m.DebutsDeListeParRepliFermeAuBit` sont lus sur `sc` (`movement_states.go:241-242`) puis versés au compteur (`film_scan_mouvement.go:57-58`). Les canaux des morts et des vitalités versent `LocalisationsALargeurLibre` au contexte (`marche_commune.go:39-48`). Sans ces canaux, `coverage.fallbacks` change. Deux voies : les lire sur le bilan de la marche (`BilanDeMarche.Obs`) et non sur `sc`, ou laisser `coverage` hors égalité.
- Le canal des vitalités nourrit `DamagePoints` (`film_scan_mouvement.go:68-69`) : calque `damage`.

**Déduit** (à vérifier par un test « canal seul = canal dans la distribution complète », sur le corpus)

- Les occurrences interprétées sont l'union des intérêts (`distribuer.go:305-316`, `marche_trames_ranger.go:54`) : retirer un canal retire ses intérêts et peut changer le marquage `EtatInterprete` que lit un autre canal. Aucun cas trouvé en lecture, aucune preuve.
- Dans la phase des images-clés, un archétype non interprété n'est plus « corps à parcourir » (`distribuer.go:20-24`, `marche_images_cles.go:122`) : retirer l'intérêt d'un archétype change ce que la marche d'images-clés lit. Même test.
- L'observation partagée du contexte a des compteurs entiers (`k/conception.md` § 2.2) : leur lecture hors de la marche n'a été trouvée nulle part.

**Avis** : sûr pour les balayages posés après la marche (phase 1, § 6). Pour l'élagage des canaux de la marche (phase 2), la sûreté dépend du test ci-dessus et gagne environ 3,5 points : je le déconseille pour la 1.6.0.

## 5. Faits persistés et étape `killsource`

### 5.1 Les faits

Les faits sont le `FilmInputs` du balayage (`replay/build_from_facts.go:86-114`). Un balayage partiel produirait des faits incomplets. Aujourd'hui, un fichier de faits n'a que deux moyens de dire ce qu'il porte : ses gardes (`GardesDeCuisson`, version de codec 2, `filmfacts_fichier.go:127`) et le sur-ensemble.

**Réponse : oui, un choix partiel doit interdire d'écrire les faits.** Vérifié : `faitsDuBalayage` rend déjà `nil` (« aucun fait ») plutôt que des faits sans clé de cuisson (`build_from_facts.go:83-90`) ; l'hôte sait traiter ce `nil` (`completerLesFaits`, hôte, `replaybuild/filmfacts_cuisson.go:209`). On ajoute le cas « sélection partielle ». Étendre le codec pour porter la sélection ferait monter `VersionCodecFaits`, donc périmer tout le parc de faits : exclu pour la 1.6.0. Option plus tard : ajouter la sélection à `GardesDeCuisson` ; `Utilisable` refuserait des faits qui ne couvrent pas la demande.

Règle d'interface : la sélection partielle n'écrit pas de faits et ne lit pas de faits (`BuildFromFacts` rend le document complet, restreint ensuite par le même filtre que les gardes ; la restriction aux calques demandés se fait alors sans film : voir S4).

### 5.2 L'étape `killsource`

`killsource` est le décodage de la source de dégât, fait **avant** la cuisson par l'hôte (`decfilm.Decode`, A3 et A11 de l'inventaire). Il calibre le profil de balayage passé à la cuisson (`Options.ProfilDeBalayage`, `options.go:436-440`, vérifié). Conséquences :

- Si l'hôte saute `killsource` pour une sélection, le profil n'est plus le même : les positions peuvent changer d'octets (**déduit** : le profil règle traversée, largeur d'axe, `param_4`, `build_from_film.go:155-163`). La sélection ne doit pas court-circuiter `killsource` ; elle ne l'accélère pas.
- Quand le contexte est partagé, la marche est faite par `LireLaMarcheCommune` **avant** `replay` (`marche_commune.go:89-106`), avec tous les canaux de la cuisson. Une sélection qui n'est connue que de `replay.Options` n'élague donc pas la marche partagée ; seuls les balayages d'après la marche gagnent. Pour élaguer la marche, il faudrait porter la sélection sur le contexte (`PoserCanaux`, sur le modèle de `PoserFils` prévu en F2) : coût supplémentaire de l'ordre d'un item, et le gain de la phase 2 (3,5 points).
- Le coût de `killsource` n'est pas dans `recents.md` (le banc mesure `BuildFromFilm`) : le gain réel sur la chaîne de cuisson complète est plus petit que les parts ci-dessus (**non établi**, § 8).

## 6. Forme de l'entrée, garde-rail, découpage

### 6.1 Le champ

- **Où** : `OptionsVitesse.Calques` (`replay/options_vitesse.go:19`), atteint par `Options.Vitesse` (le champ que l'item F3 de `k/conception.md` ajoute déjà ; `options.go` fait 494 lignes sur un plafond de 500, vérifié par `wc -l`). Le commentaire de la ligne 16 (« sans en changer le résultat ») est à réécrire : « sans changer le résultat des calques demandés ».
- **Type** : `[]string`, noms de calques = balises JSON. Pas de masque : il y a plus de 60 calques (62 entrées), et la nomenclature stable est déjà celle de `layers.go`. Normalisation : tri, doublons retirés, noms validés contre la table ; nom inconnu = erreur typée `ErrCalqueInconnu` (ajout) ; liste vide ou nil = tous les calques. Les sept calques à la requête sont refusés (erreur claire) : ils ne sont pas cuits.
- **Valeur zéro** : nil = tous, comportement d'aujourd'hui à l'octet. Preuve : `empreinte_document` de `film-bench` identique à la référence R0 de `k/conception.md` § 4.
- **API publique** : un champ, un type d'erreur, une fonction de validation. Ajouts seulement ; contrôle `apidiff` contre `v1.5.0` à la clôture.

### 6.2 La clôture des dépendances

Table `dependancesDesCalques : map[calque] -> []groupe` (fichier neuf `replay/layers_dependances.go`, car `layers.go` fait 404 lignes) et `dependancesDesGroupes : map[groupe] -> []groupe` (P -> M, H -> E, B, etc.). Calcul : point fixe sur le graphe, sur un ensemble de groupes (masque de moins de 16 bits : un octet suffit, interne). Pas de récursion sur des noms de calque : les calques qui en lisent d'autres (`eliminations` <- `scoreActions`, `medals` ; `vehicleCycles` <- `vehicles`) sont des lignes de la table, vérifiées par le test.

Entrées de l'appelant : une sélection ne dit pas si l'appelant a fourni ses entrées. Un calque demandé dont l'entrée manque reste absent de `layers` (garde existante).

### 6.3 Le garde-rail

- `TestChaqueCalqueADesDependances` : l'union des clés de `couchesDesCalques` a une ligne dans `dependancesDesCalques` (même forme que `TestCalquesCouvrentTousLesChampsRacineCuits`, `layers_test.go:73`).
- `TestChaqueEtapeObserveeAUnGroupe` : chaque étape de `BuildFromFilmSteps()` (`observe.go:97`) appartient à un groupe.
- `TestDependancesNeNommentQueDesCalquesExistants` : une entrée périmée se retire (calqué sur `layers_test.go:92`).
- `TestFermetureEstUnPointFixe` et `TestFermetureDeTousLesCalquesEstTousLesGroupes`.

### 6.4 Découpage en items

Chaque item : une séance, un worktree. « Preuve » = égalité d'empreintes des calques demandés avec le document complet, calque par calque (empreinte du sous-arbre JSON de la clé racine), sur films du banc ; sur fixtures, `contract_fixtures_test.go` et les entrées figées de `golden_inputs_*` (sans film).

Phase 1 : balayages d'après la marche et assemblage.

**S1 : table et clôture, sans comportement** (M). Fichiers : `replay/layers_dependances.go`, `layers_dependances_test.go`. Tests : ceux du § 6.3. Dépend de : rien. Ne change : aucun octet.

**S2 : porteur et normalisation** (S). Fichiers : `replay/options_vitesse.go`, `replay/options.go` (champ `Vitesse`, partagé avec F3), `options_vitesse_test.go`. Tests : `TestCalquesZeroEstTous`, `TestCalquesInconnuRefuse`, `TestCalquesNormalises`, `TestCalquesALaRequeteRefuses`. Dépend de : S1.

**S3 : balayages gardés par la sélection** (M). Fichiers : `replay/build_from_film.go:307-315` (une garde par groupe), `replay/film_scan*.go` (garde en tête de `balayerMonde`, `balayerPortage`, `balayerCapacites`, `balayerNaissances`, états des images-clés, grenades), `observe.go`. Décision : une étape sautée est **observée avec une valeur nil**, pour que la liste fermée des étapes reste la même (`observe_test.go`, `TestObserveEtapesBuildFromFilm`). Tests : `TestSelectionSauteLesBalayagesInutiles` (compteur d'appels sur fixtures), `TestSelectionToutesEgaleComplet` (étapes observées identiques). Dépend de : S2. **Conflit** avec J6a, J6b, J6c, J7 (même fichiers).

**S4a : passes d'assemblage, positions et identité** (M). Fichiers : `replay/build.go:82-109`, `replay/layers.go` (`calquesProduits` : garde de sélection à côté de `gardesDeProduction`). Chaque passe `poser*` se fait gardée. Preuve : `TestSelectionPositionsMemesOctets` (`tracks`, `bounds`, `originMs`, `identity`, `roster`) sur fixtures puis sur films. Dépend de : S3.

**S4b : passes du tir, du combat, du score** (M). Fichiers : `build_pistes.go`, `build_score.go`, `eliminations.go`, `document_hits.go`, `document_damage.go`. Preuve : `TestSelectionTirsMemesOctets`, `TestSelectionScoreMemesOctets`. Dépend de : S4a.

**S4c : passes du monde, du portage, des capacités** (M). Fichiers : `build_calques.go`, `build_inventaire.go`, `build_ground_weapons.go`, `build_vehicles.go`, `build_zones.go`. Preuve : `TestSelectionMondeMemesOctets`, `TestSelectionCapacitesMemesOctets`. Dépend de : S4a. Parallèle avec S4b (fichiers distincts).

**S5 : pas de faits sous sélection partielle** (S). Fichiers : `replay/build_from_film.go:118`, `replay/build_from_facts.go`. Tests : `TestSelectionPartielleNEcritPasDeFaits`, `TestSelectionPartielleRejoueDepuisLesFaitsSansTrouer` (la restriction aux calques demandés depuis des faits complets rend les mêmes octets). Dépend de : S4a.

**S6 : preuve sur films et banc** (M). Fichiers : `cmd/film-bench/*` (`-calques`, colonne d'empreinte par calque), test de corpus local qui saute sans données (`TestSelectionEmpreintesParCalque`). Mesure du gain réel par sélection type. Dépend de : S4a à S5. C'est elle qui tient ou rejette le gain annoncé ici.

Phase 2 : canaux de la marche (déconseillée, voir § 7).

**S7 : lire les compteurs de repli de la marche sur le bilan, plus sur `sc`** (S). Fichiers : `internal/grammar/movement_states.go:241-242`, `replay/film_scan_mouvement.go:57-58`. Preuve : empreintes de `coverage.fallbacks` inchangées à sélection pleine.

**S8 : canaux inscrits selon la clôture** (L). Fichiers : `internal/grammar/movement_states.go:196-260`, `marche_commune.go`, `film_context.go` (`PoserCanaux`, au plafond de 500 lignes). Tests : `TestCanalSeulEgalDistributionComplete` (par canal), `TestAucunDistribuerApresLaMarche`. Dépend de : S7, S3.

Total : 6 items de la phase 1 (S1, S2, S3, S4a-c, S5, S6 : huit lignes, dont S4b et S4c en parallèle) ; 2 pour la phase 2.

## 7. Rapport avec le parallélisme et avis net

### 7.1 Conflits avec `k/conception.md`

| Item de parallélisme | Fichiers partagés avec la sélection | Nature |
|---|---|---|
| F3 | `replay/options.go` (le champ `Vitesse`), `build_from_film.go` | même champ ; le premier arrivé l'ajoute, l'autre l'utilise : à ordonner |
| J2 | `film_scan.go` (départ avant la marche), `film_scan_pont.go`, `observe_test.go` | la tâche de fond lit des valeurs que S3 peut sauter : la tâche doit connaître la sélection |
| J6a, J6b, J6c | `film_scan*.go` (scinder lire et publier) | **conflit direct** avec S3 : mêmes fonctions |
| J7 | `film_scan.go`, nouveau `film_scan_voies.go` | **conflit direct** : l'orchestrateur choisit les voies ; la sélection retire des voies |
| J8 | voie monde | la sélection qui saute le monde annule la voie : pas d'incompatibilité, mais le gain de J8 ne se cumule pas |
| I1 à I4 | `internal/grammar/keyframe_*` | aucun conflit : la sélection de phase 1 n'y touche pas |
| F2 | `film_context.go` (500 lignes) | seulement pour S8 |

Ordre sûr : si la sélection se fait, S3 vient **après** J6c, sur la forme « lire puis publier », et S4a à S6 n'ont pas de conflit (fichiers `build*.go`, `layers.go`).

### 7.2 Avis

Le gain justifie-t-il le chantier ? **Non, pas pour la 1.6.0.**

1. Gain plat de 22 à 30 % pour toute sélection, parce que le document exige les positions et l'identité (§ 2, § 3). Le fil des morts seul ou le score seul n'y gagnent pas plus que les positions.
2. Aucun appelant de production n'en profite aujourd'hui (§ 3.2). `PortagesAuSync` est le seul candidat et il compose déjà ses lectures.
3. Le parallélisme prévu donne -17 à -31 % à tous les appelants sans changer l'API ; les deux gains ne se cumulent pas (la voie monde ne sert à rien si le monde est sauté).
4. Risque : l'égalité d'octets passe par le registre d'identité, qui touche tous les calques (`layers.go:61-72`, limite écrite par l'auteur de la table), et par `coverage` qui n'y échappe pas.
5. Huit items sur des fichiers que `J6` et `J7` réécrivent.

Ce qui changerait mon avis : un besoin produit précis d'un « rejeu léger » (par exemple, un document de positions au sync, ou un aperçu), mesuré à plus de 25 % de gain sur la chaîne entière, **avec** killsource mesuré.

Version minimale utile, si on veut garder une trace : **S1 seul** (la table de dépendances et ses tests, sans comportement). Elle documente ce que lit chaque calque, sert de garde-rail à la recuisson sélective (lot 4.4 cité en `layers.go:218-226`), et coûte une séance.

## 8. Ce que je n'ai pas pu établir

1. La part de temps de chaque canal de la marche : `recents.md` donne les crochets en bloc (6,5 %) ; les 3,5 points d'élagage sont une estimation.
2. Le coût de `killsource` dans la chaîne de cuisson : absent du banc. Le gain de la sélection sur la chaîne entière est donc plus petit que les parts de la table A (non quantifié).
3. Que les octets de `tracks` ne changent pas si l'index des joueurs, les sièges ou les tirs ne sont pas lus : je les ai mis dans la fermeture par prudence (`build_pistes.go:44-57`). Un test sur films le dirait ; si faux, le gain de « positions seules » monterait vers 40 %.
4. Les passes d'assemblage entre elles : l'ordre est le contrat (`build.go:60-75`) et plusieurs passes lisent ce que la précédente a posé (par exemple `tirsDuCombat`, `a.reg`). Garder une passe sans la précédente peut échouer à l'exécution : seule l'égalité sur fixtures le dirait. Je n’ai pas relu les vingt-quatre passes une à une.
5. Les 40 sites de crochets restants (j'en ai relu trois) : « les crochets ne font que publier » reste une déduction pour eux ; `TestAucunCrochetNEstPoseSurLContexteApresLaMarche` (J4) en ferait une preuve.
6. Si `PortagesAuSync` pourrait s'écrire en une sélection sans changer ses sorties : il appelle des lectures partielles (`ScanArmesDesImagesCles`) qui n'ont pas d'équivalent dans la table de groupes.
7. La mémoire : une sélection partielle la baisse (moins de balayages retenus), mais la publication, qui fait le pic (`recents.md` table D, 55 % des octets alloués), est elle aussi partielle ; le pic réel n'est pas mesuré.
8. Les parts du banc viennent de cinq films récents sur un poste (16 coeurs) : les écarts de quelques points ne sont pas significatifs.
