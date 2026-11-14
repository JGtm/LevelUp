— Tâches et TODO centralisés

---

## 🔄 En cours

- **Version 8.2.0 (branche `release/v8.2.0`, plan `.ai/PLAN_V8_1_2_2026-11-09.md`)** : plan exécuté et relu (relecture
  finale sans P0 ni P1). Restent les gestes du user : contrôles à l'écran (items 5.9 et 7.2 du plan, artefact des
  vérifications), fusion vers `main`, tag et gestes de production (`.ai/annexes_v8_1_2/gestes_prod_8_2_0.md`). L'item 7.4
  du plan (numéros de version, notes de version, runbook) n'est pas statué au plan ; ses brouillons sont sous
  `.ai/annexes_v8_1_2/brouillon_versions_8_2_0_{FR,EN}.md`.
- **Plans de la racine de `.ai/` dont le statut dit encore « en cours » ou dont des cases restent vides** (relevé du
  2026-11-14, cases non vérifiées une à une) : `PLAN_REJEU_30IPS_COMPRESSION_2026-11-10.md` (compression par la PR #199,
  cuisson à 30 images par seconde par la PR #207 ; cases B4 à E2 à statuer par la session du chantier, C1 à C4 sont des
  gestes du user), `PLAN_REJEU_MAQUETTES_2026-11-10.md` (livré par la PR #200 ; case 7.2 ouverte) et
  `PLAN_MARCHE_COMPLETE_ET_CANAUX_2026-11-10.md` (lecture des paquets par la PR #198, décodeur v1.3.6 consommé par la PR #207 ;
  cases A2 à Z ouvertes, dont la recuisson unique du parc, geste du user). Le chantier application de bureau est clos (plan `.ai/V8/PLAN_DESKTOP_2026-11-01.md`,
  versions 8.1.0 et 8.1.1 publiées).

---

## 📋 Backlog

---

### [analysis] Six divisions « par minute » écrites à la main au lieu de `analysis.PerMinute`

Relevé le 2026-11-14 par la relecture du groupe « Rythme » de Tendances (plan `.ai/PLAN_COMPLEMENTS_V8_2_0_2026-11-14.md`, étape 2). Le helper canonique `analysis.PerMinute(total, seconds)` (`internal/analysis/indicators.go:146`) existe, mais six sites refont la division : `kpi_stats.go:115`, `explorer_target_stats.go:122`, `home_canonical_highlights_tiles.go:86`, `squad_breakdown.go:373`, `squad_breakdown_canonical.go:59` et `:160`. Règle 6 : migrer les six sites vers le helper (valeurs identiques à vérifier par témoins avant/après, l'arrondi de `total * 60 / seconds` diffère) et poser un garde-rail grep qui interdit la division en ligne dans `internal/analysis`. Dette antérieure, hors du plan. **Effort : S.**

---

### [kpi] Rendement et Résistance : la vie d'un Spartan vaut 229 ou 230 points, pas 225

**Demande de l'utilisateur (2026-11-10).** Les formules du Rendement (`225 × (frags + assistances/3) / dégâts infligés`) et de la Résistance (`dégâts subis / (225 × morts)`) reposent sur la constante `games.DefaultEffectiveHpToKill = 225.0` (`apps/go-api/internal/games/damage_model.go:18`). Mesuré le 2026-11-10 dans trois films : le compteur « dégâts subis » du jeu monte de 229 points pour une vie sans recharge de bouclier ; la table des fichiers du jeu lue par libfilmshell donne 90 de santé + 140 de bouclier = 230. À instruire dans un petit lot : trancher entre 229 et 230 sur davantage de films, puis corriger la constante et tout ce qui la cite (commentaires des formules, texte de l'infobulle « 1 vie ≈ 225 points », valeur `effective_hp_to_kill` servie par `/bootstrap`). Attention : la note composite du LUSR lit ces deux indicateurs (`sync/skill/skill_rating_composite.go`), donc le changement relève de l'ADR 0047 (révision de notation, recalcul de tous les joueurs au premier démarrage, ligne dans les notes de version). Halo 5 (115) n'est pas concerné. **Condition de reprise :** décision du user de lancer le lot. **Effort : petit (une constante et ses citations), plus le recalcul des notes.** **Repris par le plan `.ai/PLAN_COMPLEMENTS_V8_2_0_2026-11-14.md`, étape 7 (2026-11-14).**

---

### [bureau/auth] Connexion Microsoft dans le navigateur : « redirect_uri » refusé par Azure

**Constaté le 2026-11-11** (poste de l'utilisateur, v8.1.1). Le serveur construit `http://localhost:<port lié>/auth/xbox/callback` (`handlers/auth_browser_login.go`, port mémorisé dans `desktop/port`, 53891 sur ce poste). Microsoft répond `invalid_request … redirect_uri … not valid` : l'enregistrement Azure du client `e1cb35ab` ne porte pas `http://localhost/auth/xbox/callback` sous la plateforme « Applications mobiles et de bureau » (seule plateforme où le port est libre ; sous « Web » le port doit correspondre exactement). Rien dans le code ni dans les bases copiées ne change cette adresse. **Réglé côté Azure le 2026-11-11 par l'utilisateur** (adresse ajoutée à l'inscription) ; reste à vérifier depuis l'application une fois les services Halo revenus. Le code d'appareil avait échoué le même jour pour une autre raison : `settings.svc.halowaypoint.com/spartan-token` en dépassement de délai (service de jetons Halo injoignable ce jour-là, voir l'entrée du journal).

---

### [bureau/auth] Le code d'appareil avale la cause de l'échec

**Constaté le 2026-11-11.** « Impossible d'obtenir un accès Halo. Veuillez réessayer. » couvre toute erreur de l'échange XBL → XSTS → Spartan (`handlers/auth.go`, `pollDeviceFlow`, code `halo_exchange_error`), y compris un dépassement de délai des services Halo. Distinguer au moins « services Halo injoignables, réessayer plus tard » d'un refus du compte (pas de profil Xbox, mauvais compte).

---

### [accueil] « Synchronisation en cours » affiché dès qu'il n'y a aucun match, qu'une synchronisation tourne ou non

**Constaté le 2026-11-11** (issue #192). `features/home/HomePage.tsx` montre l'écran de première synchronisation sur `total_matches == 0`, sans lire l'état réel du sync. Un joueur dont aucune synchronisation ne part (cas corrigé par la 8.1.2) lisait ce message pendant une heure. Lire l'état du sync (job actif, dernière erreur) et le dire : « en cours », « en attente de connexion », « échec : … ».

---

### [bureau/retours] « App version : unknown » dans les signalements envoyés depuis l'application

**Constaté le 2026-11-11** (issue #192). Le tiroir de signalement ne connaît pas la version de l'application de bureau ; le navigateur se reconnaît à `Edg/` dans l'agent. Renseigner la version depuis `/bootstrap` ou la coquille.

---


### [replay/sons] Fins de partie multi-équipes par couleur — écran + annonceur — BLOQUÉS faute de donnée

Noté le 2026-08-27 (chantier rejeu 2D, plan `.ai/archive/V7.5/PLAN_REPLAY_CADRAGE_VICTOIRE.md`,
branche `wt/replay-cadrage-victoire`). L'écran de victoire et les sons de fin (lots B/C)
couvrent les matchs à 2 équipes + le FFA gagné ; les modes MULTI-ÉQUIPES (3+) n'ont ni
écran (décision D-B1) ni son. Or le jeu porte une famille complète d'annonces dédiées,
identifiée par transcription locale des packs annonceur :

- **8 répliques FR** « Partie terminée, l'équipe X est déclarée vainqueur » — bleue
  `1070034924`, rouge `808622693`, cyan `83592248`, mauve `186794961`, verte `265309140`,
  citron `784566745`, jaune `868146650`, orange `399957729` (ids `.wem` du pack
  `French(France)/sb_001_vo_ai_mp_announcer.pck`) ;
- **8 jumelles EN** « Game over — <color> team wins » — red `1005389916`, blue `101785491`,
  green `1010879786`, purple `100056384`, yellow `927455187`, orange `564119611`,
  lime `92374`, cyan `256805823` (pack `English(US)/…`).

**BLOQUÉ FAUTE DE DONNÉE (plan backlog du 2026-09-26, lot A5, statué `[!]` le 2026-09-27).**
Le plan d'exécution est prêt dans `.ai/archive/V7.5/PLAN_BACKLOG_2026-09-26.md`, section A5 : pièces, items
A5.0 à A5.7, et décision D-9. D-9 garde l'écran du joueur regardé, ajoute « Victoire de l'équipe
<Nom> » en défaite, et fait jouer la réplique de la couleur du gagnant pour tous. Deux faits établis
en lisant les données le 2026-09-26 bloquent le lot :
- **aucun match multi-équipes jouable** : il n'y a qu'un seul match à 3 équipes ou plus, de 2
  joueurs ou plus chacune, dans toute la base partagée locale (2025-08-13, aucun joueur suivi).
  Aucun des 111 artefacts de rejeu ne porte d'équipe au-delà de 1 ;
- **la table équipe → couleur annoncée n'existe nulle part** : les couleurs de `teamNames.ts`
  ne correspondent pas aux 8 couleurs de l'annonceur (magenta, doublon orange-rouge).

**Condition de reprise** : l'utilisateur joue une partie à 3 équipes ou plus (matchmaking ou
partie personnalisée), la synchronise, et note la couleur annoncée pour l'équipe gagnante. Le lot
peut alors partir, en commençant par la recherche hors ligne de la table dans les fichiers du jeu
installé (A5.0). Les 16 répliques sont déjà extraites et converties sous
`Desktop/Halo Infinite - Sons armes/_fin_partie/annonceur_{fr,en}_wav/` (non normalisées ; les
copies de `livraison/` y sont restées en 4 canaux). **Effort : M.**
**Constat du user le 2026-10-09 : aucun mode à plus de deux équipes (hors FFA) n'est disponible en
jeu actuellement.** Le lot reste en attente d'un tel mode.

---

### [data/objectifs] Blocs `EliminationStats` / `InfectionStats` — BLOQUÉS faute de donnée

Réduit le 2026-07-25 (v7.2.1, V721-02). Sur les 4 blocs de mode non extraits, **3 sont
livrés** (Stockpile, Extraction, et `VipStats` qui n'était même pas déclaré dans
`StatsBundle`) — 18 colonnes. Les 2 restants ne sont **pas** un manque de travail mais un
manque de donnée, établi sur payloads réels (P0, 10 matchs interrogés) :

- `EliminationStats` : les 2 seuls matchs Attrition en base ne portent **aucun** bloc de
  mode (`Stats` = `CoreStats` seul). Aucun mode Elimination parmi les 65 modes distincts
  présents en base.
- `InfectionStats` : « Survive The Undead 3.0 » est un Firefight UGC (catégorie 41), pas
  un Infection. Aucun mode Infection en base.

**Déblocage** : jouer une partie matchmaking de chacun de ces modes, puis la synchroniser. **Constat du user le 2026-10-09 : ni Élimination ni Infection ne sont disponibles en jeu actuellement.**
Le patron existe alors en triple exemplaire — **effort : S par bloc**. Interdiction
explicite d'inventer le schéma par analogie.


---

### [rejeu/trajets] Trajet de Warthog porté par l'humain alors qu'il suit un bot

Relevé le 2026-11-10. Noté pendant la v8.1.2, périmètre figé. Film `4f77afc1` : un trajet de Warthog (siège 524, images 279 à 287) porte le xuid de l'humain alors qu'il suit la piste du bot qui le précède, et 8 pistes de bot sont rattachées à l'humain du siège par le pont « premier occupant ». Corriger demande une publication des équipes et des trajets par corps, qui fait aujourd'hui bouger des équipes (joueur `3abf0258`, revenu sous deux index). **Condition de reprise :** à planifier après la 8.1.2. **Effort : M.**

---

### [rejeu/sièges] Pose des places : 13 films à siège vacant intercalé lus sans table

Relevé le 2026-11-10. Noté pendant la v8.1.2, périmètre figé. La pose des places (`sieges.go`, `siegesDuDebut`) lit encore `grammar.Lue()` et voit sans table les 13 films à siège vacant intercalé désormais confirmés. Film `23ffd885` : les chunks donnent au siège 2 l'index 8 ; une confirmation par les chunks pourrait le trancher. **Condition de reprise :** à planifier après la 8.1.2. **Effort : M.**

---

### [rejeu/schéma] Sept films du parc avec des faits d'un ancien schéma

Relevé le 2026-11-10. Noté pendant la v8.1.2, périmètre figé. Sept films du parc ont des faits d'un ancien schéma, illisibles par le binaire actuel. Le relevé en identifie six : `000d5950`, `51101d1d`, `9f57c612`, `a521164d`, `bcb6d393`, `edc5daf6`. Toute recuisson de ces films demande l'accord du user. **Condition de reprise :** à planifier après la 8.1.2, avec l'accord du user sur la recuisson. **Effort : S.**

---

### [rejeu/ti=40] Bande ti=40 : 249 parents d'occupation exclus, 6 porteurs lus à plus de 15 m

Relevé le 2026-11-10. Noté pendant la v8.1.2, périmètre figé. La bande `ti=40` exclut 249 parents d'occupation recensés (slot ayant porté un autre archétype). Six porteurs de pièce sont aussi lus à plus de 15 m à la montée. **Condition de reprise :** à planifier après la 8.1.2. **Effort : M.**

---

### [rejeu/identité] Film `5da6fd30` : six vies sans nom tant qu'il n'est pas régénéré

Relevé le 2026-11-10. Noté pendant la v8.1.2, périmètre figé. Les faits actuels du film (`Kills` nil) restent frais ; les 6 vies ne seront nommées qu'après régénération de ce seul film (étape 7.5). En prod, sans chunks, elles restent sans nom. **Condition de reprise :** à planifier après la 8.1.2, régénération de ce film seul sur accord du user. **Effort : S.**

---

### [rejeu] Calque « Morts » : les morts sans type perdent leur croix, faute de signal de fin de manche dans le document

Relevé le 2026-11-14 par la ronde 2 bis du calque « Vies ». Les vies des survivants se ferment sans mort à chaque fin de manche et le fil en fait des lignes neutres ; le calque ne marque donc que les morts neutres dont le film établit la cause (`kind` non vide, `model/livesMarks.ts`, `deathFacts`). Effet de bord : de vraies morts sans type perdent leur croix, soit une source de dégât sans pictogramme (`apps/go-api/internal/replaybuild/replaybuild.go:449-452`), soit surtout le refus d'attribution ligne par ligne (`replaybuild.go:436-440`, BTB ou alerte de santé du décodeur) où tous les suicides et toutes les chutes du match restent sans type. Compteur : `repli_repere_neutre_generique_conserve` ; exemple `58d09c44` : valeur 2, deux vraies morts sans croix. Le document ne porte pas de signal de fin de manche exploitable ici : `scoreTimeline` ne donne que le dernier palier de chaque manche (`model/roundsLogic.ts`), alors que les vies se ferment 7,6 à 9,1 s plus tard (mesure de l'en-tête de `roundsLogic.ts`, cinq bascules) ; une fenêtre fondée là-dessus serait une heuristique, pas une lecture. Piste : publier côté Go, par vie, « fin de manche » (réapparition groupée lue dans le film) et la lire dans `deathFacts` à la place du critère de type, ce qui demande une régénération. Les notes de la 8.2.0 disent que les morts sans type ne sont pas marquées. **Condition de reprise :** décision du user de régénérer (D-13 interdit un champ cuit dans le lot Vies). **Effort : M (champ de document, schéma, régénération).**

---

### [rejeu] Les effets de mort ignorent « Seul » et « Masquer » du panneau de joueur

Relevé le 2026-11-14 par la relecture du calque « Vies ». `drawKillFxLayer` (calque `morts` de `layers/replayCompose.ts`, bâti par `useReplayFx` / `model/killFx.ts`) ne reçoit pas `shownAtSlot` : « Seul » et « Masquer » de la fiche joueur n'y filtrent rien, alors que les pions, les tirs et les traces des vies s'y plient (`useReplayPlayerFocus`). `shownSlotPredicate` (commentaire de `model/playerPanelLogic.ts`, corrigé le 2026-11-14) dit que les effets de mort, objectifs, véhicules et objets posés restent hors filtre, quand les traces des vies, elles, s'y plient : la règle est à trancher avec le user avant de câbler, car elle est écrite comme voulue. Fichiers : `apps/web/src/features/match-replay/ui/ReplayCanvas.tsx` (liaison `morts`), `layers/replayDraw.ts` (`drawKillFxLayer`), `model/playerPanelLogic.ts`. **Condition de reprise :** décision du user (le filtre doit-il couper l'effet de mort ?). **Effort : S.**

---

### [web/couleurs] Infobulle `info-tooltip` ouverte au focus sans filtre clavier ; garde-rail de couleurs incomplet

Relevé le 2026-11-10. Noté pendant la v8.1.2, périmètre figé. `components/ui/info-tooltip.tsx` ouvre aussi au focus sans filtre clavier. Le garde-rail de couleurs ne couvre pas `ring-offset`, `outline`, `decoration`, `accent` ni `caret`, et un `color-allow` masque toute couleur de sa ligne. **Condition de reprise :** à planifier après la 8.1.2. **Effort : S.**

---

### [docs] ADR 0020 cite `PatternReport` ; ADR 0006 garde le quotient dans son corps

Relevé le 2026-11-10. Noté pendant la v8.1.2, périmètre figé. L'ADR 0020 cite encore `PatternReport` comme entrée du coach. L'ADR 0006 garde le quotient dans son corps, l'amendement étant censé l'annuler. **Condition de reprise :** à planifier après la 8.1.2. **Effort : S.**

---

### [licences/tests] Surnoms dérivés de pseudos réels dans des identifiants de tests

Relevé le 2026-11-10. Noté pendant la v8.1.2, périmètre figé. Après la pseudonymisation des fixtures sous `apps/` (v8.1.2, item 5.4), des surnoms tirés de gamertags réels restent dans des noms de tests et des libellés (plusieurs centaines d'occurrences, liste dans les tables hors dépôt de la pseudonymisation). La table de remplacement ne remplace que des gamertags entiers. Décision du user du 2026-11-10 : noté au backlog. **Condition de reprise :** à planifier après la 8.1.2. **Effort : M.**

---

### [licences] Pseudos réels hors de `apps/`

Relevé le 2026-11-10. Noté pendant la v8.1.2, périmètre figé. `.claude/skills/db-schema/SKILL.md`, `.gitignore`, `config/titles/halo_infinite/constants.toml`, `docs/` et `.ai/` portent encore des gamertags réels. Un chemin Windows personnel reste dans `apps/go-api/cmd/restore_one_player/main.go`. Le pseudo du mainteneur (`JGtm`) est conservé par décision du même jour. Décision du user du 2026-11-10 : noté au backlog. **Condition de reprise :** à planifier après la 8.1.2. **Effort : M.**

---

### [logs] Avertissements répétés pendant un long recalcul

Relevé le 2026-11-10 au premier démarrage de la 8.1.2 : pendant le rejeu des notes LUSR, `sharedprovider` écrit environ 200 fois « writer RW tenu au-delà du seuil — lectures gatées » pour le même écrivain (`match_recompute/lusr`), et le repli du solveur une ligne WARN par match (36). Une ligne au début et une à la fin de la tenue, avec la durée et le nombre de lectures retenues, suffiraient ; le repli peut être compté et résumé en fin de rejeu. Le user trouve ce bruit gênant (10/11) ; noté pendant la v8.1.2, périmètre figé. **Condition de reprise :** à planifier après la 8.1.2. **Effort : S.**

---

### [ci/gate-push] Budget `-timeout=300s` trop court en mode intégration

`make gate-push` sur machine chargée : `internal/archlint`, `internal/sync` et `internal/platform/duckdb` atteignent 300,2 à
300,3 s ; `internal/sync` en intégration a été observé à environ 24 minutes. Le 2026-11-14, `go test -tags=integration ./...`
(limite par défaut de 10 minutes) coupe aussi `internal/archlint`, `internal/ops` et `internal/sync` sous charge, sans test en
échec ; rejoués seuls avec `-timeout 40m` : 428 s, 249 s et 699 s, verts. Reprend le diagnostic de l'ancienne entrée « Série Go
`reste` : blocages sans test en échec » (2026-11-09, retirée) : ni paquet qui se bloque ni test rouge, un manque de temps sous
charge. À revoir : budget par paquet ou exécution séparée des trois paquets lourds. **Effort : S.**

---

### [web/tests] Le badge du rejeu n'est pas testé de bout en bout

Le test `apps/web/src/lib/replay/queries.cadence.test.tsx` s'arrête à la valeur lue par la requête (l'en-tête de cadence). Retirer
la prop `cadenceStale={data.cadenceStale}` de `ReplayPage.tsx` ne fait rougir aucun test : il faudrait monter toute la page.
**Effort : S à M.**

---

### [sécurité/deps] Alerte Dependabot glib (Rust)

`glib` 0.18.5 vient de `gtk` 0.18, dépendance Linux de Tauri (tauri 2.11.6, tao, wry, webkit2gtk, tray-icon, muda), non compilée
sous Windows (`cargo tree --target x86_64-pc-windows-msvc` ne la trouve pas). `cargo update -p glib --precise 0.20.0` échoue (`gtk`
exige `^0.18`). À écarter comme non applicable (Windows seul, ADR 0045) ou à régler à la prochaine montée de Tauri : décision du
user. **Effort : S.**

---

### [bureau/tests] `make desktop-test` laisse un binaire de serveur vide sous `binaries/`

Relevé pendant la 8.2.0 (plan 8.1.2, étape 6.5), non traité. `apps/desktop/src-tauri/build.rs` (autour de la ligne 170) écrit un
`levelup-server-x86_64-pc-windows-msvc.exe` vide sous `apps/desktop/src-tauri/binaries/` (ignoré par git) quand il est absent ; le
test `TestLesBinairesDeAppsSontCouverts` (`apps/go-api/cmd/anonymise-gamertags/binaires_couverts_test.go`) refuse ensuite ce
fichier, et la baseline de tests rougit tant qu'il n'est pas retiré à la main. À faire : que le test l'ignore, ou que `make
desktop-test` nettoie après lui. **Effort : S.**

---

## 🎮 Backlog — Coach proactif × Prestige (post-V2)

Référence : ADR 0020 — Coach proactif : pont vers Prestige. ADR 0021 — Synthèse dynamique de Template et Arc ad-hoc.

> **MàJ 2026-07-17** : les 3 extensions autrefois parkées sont LIVRÉES (train backlog) —
> V2.1 télémétrie `source` + endpoint diag, analyseur de tuning de la grammaire
> (recommandations, validation manuelle), et canal externe Discord webhook (opt-in, OFF
> par défaut). Voir « Récemment complété ».

**Arcs multi-titres** : clos le 2026-07-25 (v7.2.1, Option A, retrait `arc_titles`) — plan archivé
[.ai/archive/V7.2/PLAN_CROSS_TITLE_ARCS_2026-07.md](.ai/archive/V7.2/PLAN_CROSS_TITLE_ARCS_2026-07.md).

Enrichissements possibles (non planifiés) : funnel `coach_proposal.status` dans l'analyseur de tuning.
Réglages Discord sans interrupteur web : relais du coach, rejeu prêt, reconnexion requise, disque.

---

## ✅ Récemment complété (référence)

> Entrées antérieures à 2026-07 (ère Python/Streamlit, bascule vers Go) : `.ai/archive/BACKLOG_COMPLETE_avant_2026-07.md`.
> Les chantiers en cours se suivent dans leurs plans (`.ai/PLAN_*.md`) et dans la section « En cours » ci-dessus.

| Date | Item |
|------|------|
| 2026-11-14 | **[release] LevelUp 8.2.0** (plan `.ai/PLAN_V8_1_2_2026-11-09.md`, branche `release/v8.2.0`) : plan 8.1.2 exécuté (tous les points du backlog du 2026-11-09 sauf 17, 18, 24 et 25 : barre latérale, volet des référentiels, sélecteur de dossier, zone du tueur au killfeed, FDA nette, alertes localisées, notes des matchs non convergents, icônes du jeu, nettoyage de l'installeur, licences et identités fictives ; PR #181, #183, #184, #185, #186, #187, #189, #190), correctif 8.1.2 de synchronisation intégré (#193) ; plan d'architecture en 8 étapes (suppressions, interrupteurs, écritures, couche HTTP, stockage, multi-titre, front, clôture ; PR #191, poussé sur la release pour le reste) ; briques consommées comme modules publiés (`github.com/JGtm/...` en v1.0.1, `libs/` et `go.work` supprimés, paquet `internal/briques` ; PR #194, #201, #202, puis montées en v1.1.0 par la #205 et au décodeur v1.3.6 par la #207) ; rejeu à 30 images par seconde (#190, #207), rejeux compressés (#199), lecture complète des paquets du film (#198), fil filtrable, fiche en panneau et Mainmise (#200), destructions de véhicule et occupants tués (#203), fonds de carte tenus par le maillage de navigation (#205), alertes de sécurité des dépendances (#206). Relecture finale par trois relecteurs sans le plan : 0 P0, 0 P1, 3 P2 corrigés. Restent les gestes du user : contrôles à l'écran, fusion vers `main`, gestes de production. |
| 2026-11-08 | **[archi] Décodeur de film : brique `libs/film` publiable** (plan `.ai/PLAN_FIN_DECOUPLAGE_BRIQUE_FILM_2026-11-02.md`) : réorganisation protocole/exploitation, identifiants fictifs dans les fixtures, README/NOTICE/exemple, ADR 0043 amendée, revue adversariale close (branche `refactor/brique-film-publication`, PR vers `release/v8.1.0`). Reste : tag `libs/film/v0.1.0` (geste du user), lot 5 (jeton Spartan sur redirection, NOTICE gamefiles) et lot 6 (quatre dépôts dédiés) après le bureau et `main`. |
| 2026-11-08 | **[backlog] Lot du 2026-11-08** (plan `.ai/V8/PLAN_BACKLOG_2026-11-08.md`, branche `chore/backlog-2026-11-08`) : matchs exclus retirés du graphe de carrière et des composantes LUSR, rejeu LUSR complet en échec visible si une écriture est tenue ; moteurs servis par le provider de leur titre, profils `auth_only` écartés du recalcul de sessions, ratchet des slugs étendu à `cmd/` ; page Citations sans barre de filtres, noms sur deux lignes, erreur de règles remontée, citations réconciliées à chaque démarrage ; rafraîchissement OAuth par le dernier client ayant réussi (mémoire du processus) ; `reset-bitmasks` sur les constantes canoniques, relance de connexion gardée après succès, deux instruments sortis du corpus `gamefiles`. Clos sur preuve sans code, détail au §0 du plan : `weapon_resolver`, `match_bomb_stats`, `cmd/levelup` et étapes par titre, maillon de nom Halo 5, fixture `psa_index_repro`, lecture du précédent LUSR, filtre CI, `RunForDB`, passes CSR, première vérification de participation, `/tactical/maps`, banc Cliffhanger ; instruments `research` traités par la brique film. Revue indépendante du plan, relecture dédiée de l'authentification, relecture finale Opus. |
| 2026-11-02 | **[qol] Citations `.ai/` retirées des commentaires de code** (PR #134, entrée de backlog ouverte par la PR #127) : nettoyage fait, ratchet à plafond 0. Même PR : moteur des véhicules au mouvement dans le rejeu, tuile seule des sessions récentes. |
| 2026-11-01 | **[archi] Publication des briques : modules, README, tags** (plan `.ai/V8/PLAN_PUBLICATION_BRIQUES_2026-10-30.md`, PR #118 et #119 dans `release/v8.1.0`) : module `libs/haloapi` multi-jeu (`xblauth`, `infinite`, `halo5`, `xboxlive`, `haloapitest` ; clients Halo 5, UGC cartes, GameCMS et succès Xbox coupés en protocole / exploitation sans changement observable), README EN des trois briques, `example_test.go`, ADR 0043 ; tags `libs/{gamefiles,himap,haloapi}/v0.1.0` posés sur validation du user, `go get` vérifié hors dépôt. Reste : dépôts dédiés (entrée du backlog ci-dessus). |
| 2026-10-29 | **[archi] Découplage des briques D1, D2, D3** (audit `.ai/V8/AUDIT_DECOUPLAGE_BRIQUES_2026-10-27.md`, ADR 0041 et 0042) : `libs/gamefiles` + `libs/himap` reliés par `go.work` (PR #111), décodeur de film en module `libs/film` (PR #113), API Halo séparée en protocole et exploitation (PR #114) ; suites (gates locaux, ratchets, robustesse `haloapi`) en PR #117. |
| 2026-10-28 | **[i18n] N langues (ADR 0040)** : source unique `config/locales.toml`, manifests générés côté web et Go, libellés résolus par le serveur dans la langue de la requête, garde-rails des deux côtés ; fusionné dans `release/v8.1.0` avec la revue UI du 2026-10-28 (rangs déduits du code, défis reconnus dans une autre langue). |
| 2026-10-18 | **[données] Indicateur « données de compétence » des 157 matchs synchronisés du 25 au 30 mai** (plan `.ai/archive/V7.5/PLAN_BACKLOG_2026-10-16.md`, après E10, sur décision du user) : la donnée était en base (MMR d'équipe présent ; joueurs suivis concernés : JGtm 137 matchs, Chocoboflor 87, Madina97294 75, XxDaemonGamerxX 10, joués de nov. 2021 à mai 2026), seul le bit skill (4) manquait. `levelup reset-bitmasks` serveur arrêté, base sauvegardée avant (`data/backups/avant_reset_bitmasks_2026-10-18/`) : 157 lignes `match_registry` et 1 302 lignes `match_participants`, dry-run rejoué = 0, `levelup healthcheck` tout OK. Les bases locales étant copiées vers le VPS à la release, rien à faire en prod. |
| 2026-10-18 | **[déploiement] Recalcul des dérivés Halo 5, LUSR v2 de tous les joueurs, séquence d'après déploiement** (plan `.ai/archive/V7.5/PLAN_BACKLOG_2026-10-16.md`, E9, branche `feat/backlog-2026-10-16`) : (1) `levelup backfill --title <slug>` est honoré par `--engagement-scores`, `--citations`, `--citations-recompute-all`, `--composite-only`, `--lusr` (v2), `--perf`, `--assists-model` et `--compare-formulas` (un pilote et une table de passes remplacent six paires de boucles recopiées ; `--all` prend les profils du titre, `--csr`/`--shared-csr` refusent un autre titre, migrations de la base DU titre) ; sur Halo 5 les passes de citations répondent « non supporte » par la capability `citations.engine`, sans rien ouvrir ; `POST /backfill/start` prend un champ `title` (404 si le joueur n'est pas suivi sur ce titre), construit le moteur et le provider shared DU titre (`config.SharedProviderForTitle`, aussi utilisé par le replay LUSR de l'admin, qui passait le provider d'Infinite à un joueur Halo 5). (2) `POST /settings/sessions/recalculate` recalculait CHAQUE profil sur les bases du titre par défaut (titre vide) : un profil Halo 5 n'était jamais recalculé ; corrigé par profil, test rouge avant. (3) « Dates de paliers » : source `cmd/backfill_milestone_dates` (hors image de prod, décision au runbook). (4) Runbook : section « Post-deploy recomputes (v7.5) » (R1 à R10 : 157 matchs, dérivés Halo 5, LUSR v2 des deux titres, séquence film, sessions, accords du pool, re-seed de la démo), commandes et routes exactes. Rien n'est exécuté : geste du user. |
| 2026-10-18 | **[rejeu] Lot film : positions du coup fatal, score KOTH Lattice, Harvest, vies sans nom** (plan `.ai/archive/V7.5/PLAN_BACKLOG_2026-10-16.md`, E8, branche `feat/backlog-2026-10-16`) : (1) positions : la passe lit le journal fusionné par xuid (joueurs renommés), tient la dernière position dans la même vie, lit le siège du véhicule et place le tueur déjà mort à sa mort ; le contexte de mort lit la victime par la même lecture que `kill_positions` (source unique, tolérance de 0,5 m retirée) : tueurs sans position 1 476 -> 78 et victimes 1 028 -> 85 sur 233 films (22 535 frags), 2 contextes de mort sur 23 326 non placés contre 148 gagnés. (2) vies sans nom : 449 -> 15 sur les 36 films en cause (index des chunks hors table, chaîne de contradictions, corps de bot arrivé avant sa première déclaration), aucune vie nommée perdue sur les 2 660 films comparés. (3) score KOTH Lattice : une émission hors domaine (canal B non nul d'un slot d'équipe) remplaçait le point réel par 0 et le seuil de garde retenait un intervalle fusionné : `7de0b91d` rend 40 sans contradiction, `26602661` sans descente, 6 films changent de série, les autres KOTH inchangés. (4) Harvest : carte Forge, canevas `fo11_blank` prouvé par le level_id de la variante (426470249, unicité 1/1), catalogues à 85 cartes (bornes) et 131 (objectifs), `e449a696` et `99b8ba5c` cuisent (158 et 144 pistes, 0 vie sans nom). Révisions montées une fois : `objectives-2026-10-17`, `isolement-2026-10-17-positions`, schéma 99, `placement-2026-10-17`, `grammar-2026-10-18`, `killsource-2026-10-18`. Gate : `replay-corpus-gate` 19/19, 0 perte. Reste après la fusion (serveur arrêté, puis prod, le user prévenu) : `backfill-replay --only-existing`, `backfill-killsource`, usage-summary, `backfill-pad-tiers --force` ; contrôle visuel du user en lecture sur `7de0b91d` et `e449a696`. |
| 2026-10-17 | **[ui] Indicateur « Analyse en cours » (tuile + vue match)** (maquette validée le 2026-10-16) : pastille sur l'image de carte de la tuile, emplacements en attente (frags assistés, citations, rejeu), bandeau dans la vue match ; état donné par le serveur, rechargement tant que l'analyse court (`components/ui/analysis-pending-*.tsx`, `match-card-analysis-pending.tsx`). Fusionné dans `feat/v75`, publié avec la v8.0.0. |
| 2026-10-17 | **[perf/tactique] Chargement de la page Tactique : de 96 requêtes à 5 hors images** (plan `.ai/archive/V7.5/PLAN_BACKLOG_2026-10-16.md`, E7, branche `feat/backlog-2026-10-16`) : le chargement coûtait 2 requêtes par carte ouvrable (le calage `GET /tactical/{map}/background`, puis l'image `background.png`) en plus de 4 requêtes fixes, soit 96 pour 46 cartes. `/tactical/maps` porte désormais le calage du fond de chaque carte ouvrable (`calage_fond`, absent pour une carte sans fond figé ; même source que la route par carte ; panne journalisée sans arrêter la grille) ; la vignette lit le cadre du fond dans la carte (plus aucune requête de calage par vignette) et ne demande son image qu'une fois entrée dans le champ de vision, et seulement si la carte a un calage (hook `useVisibleUneFois`, même clé de cache que le plan principal). Repli de D-10 : pas de `<img loading="lazy">`, car un `<img>` n'emporte pas `X-LevelUp-Title` et l'API résoudrait le titre de la session. Restent 5 requêtes de données (4 fixes + le calage de la carte du plan) puis les images des vignettes visibles. Mesure établie par le code et un test à 46 cartes ; la mesure en navigateur (onglet Réseau, temps d'affichage) reste au contrôle visuel. |
| 2026-10-17 | **[dette/citations] Premiers frags du profil, capability des citations, maîtrise Carabine à impulsion** (plan `.ai/archive/V7.5/PLAN_BACKLOG_2026-10-16.md`, E6, branche `feat/backlog-2026-10-16`) : (1) `computeFKFD` lisait des colonnes absentes de `highlight_events` et avalait l'erreur : premiers frags et premières morts à 0 pour tous les titres ; il passe par le chargeur unifié des événements (repli sur le journal des frags pour Halo 5), `analysis.FirstBlood` et le nouveau `analysis.FirstDeath`, sur les matchs du joueur, Campagne exclue, erreur journalisée ; tests Infinite et Halo 5 rouges avant, verts après. (2) `citations.engine` passe à `supported` pour Infinite (TOML, repli de l'adapter, parité verte ; seul lecteur : la matrice des fonctionnalités admin). (3) Diagnostic de la maîtrise Carabine à impulsion absente de la page Citations : aucun défaut de code (ligne présente et active au seed et dans les copies à froid de la metadata, chaîne service/page rejouée sur copie, image suivie par git, les 32 maîtrises `weapon_kills:` du seed se résolvent) ; reste à lire l'état de la metadata de l'instance regardée (requête au journal du plan) et, au besoin, `levelup seed citation-mappings`. Storm : aucune citation ajoutée (l'équivalent Infinite est la Carabine à impulsion). |
| 2026-10-17 | **[dette] Dettes Go du plan du 2026-10-15** (plan `.ai/archive/V7.5/PLAN_BACKLOG_2026-10-16.md`, E5, branche `feat/backlog-2026-10-16`) : code mort supprimé avec ses tests et lignes de baseline (`FindMatchesMissingParticipantBits`, `loadAwardsForScoreboard`, `WithAwardsRepo`, `CachedHighlightEventsRepo`, `BatchBuilder.AddLUSRComponents`, paramètre `titleSlug` de `SquadUsualContexts` remplacé par `prestige.Deps.TitleSlug`, 14 réexportations de constantes) ; erreurs avalées journalisées avant la dégradation (libellés d'armes, image de carte, modes connus, nom de paire, assistances attendues) et `ApplyResolutionViews` remonte l'échec de `v_match_full` ; DDL de `v_match_full` en source unique `migration.VMatchFullViewSQL` + garde-rail archlint ; une requête abandonnée ne bascule plus en lecture live ni ne compte ni n'écrit d'ERROR sur la vue match ; garde de la Campagne étendu au registre lu sur fenêtre de temps (2 dispenses d'outils opérateur) ; `reset-bitmasks` sur `matchflags.BackfillParticipants`, mapper OpenSpartan honnête sur les médailles (`EventsLoaded` supprimé), `--title` sur 3 CLI et passage du slug à `backfill-killsource` ; `match_bomb_stats` exporté dans le snapshot (un cut ancien reste lisible) ; un ami suivi sans player DB est une absence et non plus une panne des extras. Reste : le bit skill de 157 matchs (geste de déploiement, runbook). |
| 2026-10-17 | **[data/LUSR] Le recalcul après exclusion d'un match utilisait encore LUSR v1** (plan `.ai/archive/V7.5/PLAN_BACKLOG_2026-10-16.md`, E4, branche `feat/backlog-2026-10-16`) : la v1 du LUSR est supprimée (batch, étape de post-sync, aperçu `levelup backfill --lusr --dry-run`, interrupteurs `LEVELUP_LUSR_V2_ENABLED` et `LEVELUP_LUSR_CANONICAL`, `RecomputeAfterARTRebuild`, tests v1). L'exclusion d'un match rejoue la chaîne v2 du joueur (sentinelle de reset puis rejeu ; l'écrivain partagé est pris par rafales courtes, jamais en continu) et la v2 saute les matchs `is_excluded`. Les 8 composantes (`lusr_component_history`, lues par Prestige, le profil et l'escouade), écrites jusque-là par la seule v1 donc figées depuis la bascule canonique, sont écrites par la v2 : même calcul, parité vérifiée valeur à valeur (275 valeurs identiques à 1e-12). Constat corrigé : la vue `match_skill_rank_latest` ne laissait pas la v1 masquer la v2 (`start_time` avant `written_at`), mais le graphe d'évolution (`match_skill_rank_latest_by_type`) servait bien les lignes v1. Mesure sur une copie à froid de JGtm (sauvegarde du 16/10) : composantes 7 137 lignes pour 1 028 matchs, dernière ligne du 18/07, 125 matchs notés sans composantes, puis après rejeu 8 011 lignes pour 1 153 matchs et 0 note sans composantes (rejeu complet 47 s en 24 rafales). Reste : le simulateur de formules `levelup backfill --compare-formulas` (TrueSkill v1, n'écrit rien) est conservé. |
| 2026-10-17 | **[ops/arrêt] Séquence d'arrêt du serveur bornée sous le délai de Docker** (plan `.ai/archive/V7.5/PLAN_BACKLOG_2026-10-16.md`, E3, branche `feat/backlog-2026-10-16`) : `stop_grace_period: 45s` sur `levelup` et `levelup-demo` ; budget global de 40 s depuis la demande d'arrêt (5 s gardés pour la fermeture des bases), chaque phase d'attente prend la plus petite de sa borne et du budget restant, y compris `watcherDaemon.Stop`, `BatchQueue.Close`, `workerWG.Wait` et le gate (jusque-là non bornés ou hors chronomètre) ; `slog.WarnContext` nommant la phase qui dépasse ; `duckdb.CloseAll` toujours atteint. Séquence extraite de `main()` dans `cmd/server/shutdown.go` + `shutdown_budget.go`. Tests : budget sous la grâce lue dans `docker-compose.yml` (rouge prouvé sans la clé), phases bloquées, ordre HTTP, recalculs, écrivains, fermeture. Commentaire périmé (`stop_timeout=20s`) supprimé. |
| 2026-10-17 | **[sécurité/suites] Code Xbox redemandé après une connexion réussie, et vérification de participation en 503** (plan `.ai/archive/V7.5/PLAN_BACKLOG_2026-10-16.md`, E2, branche `feat/backlog-2026-10-16`) : (1) `XboxLoginPage` ne relance plus `start` sur `attempt_not_found` une fois la tentative autorisée (la rotation de session au succès rend l'ancienne tentative introuvable) et le sondage `useDeviceFlowStatus` s'arrête dès `authorized`/`provisioned` (prédicat `isDeviceFlowSucceeded`, `setup/queries.ts`) ; test Vitest rouge avant le correctif (5 sondages au lieu de 1). (2) `RequireMatchParticipation` : cache borné (4 096 entrées, le plus ancien sort) des participations CONFIRMÉES seulement, clé titre + xuid + match, ni `false` ni erreur mémorisés, 503 fermé conservé sur échec non caché (`api/wire/participation_cache.go`, câblé dans `MatchParticipation`) ; tests : 2 appels = 1 requête, clé, borne, bout en bout avec le middleware. La première vérification d'un couple reste soumise au budget de lecture. |
| 2026-10-17 | **[ci/durée] CI raccourcie de 49 à ~16 min, mêmes tests** (plan `.ai/archive/V7.5/PLAN_BACKLOG_2026-10-16.md`, E1, branche `feat/backlog-2026-10-16`) : suite Go de couverture en 6 lots sur 6 machines, chacun en `-p 1` (définition unique `.github/ci/go-shards.txt`, dernier lot = complément calculé, `grammar` coupé en deux par une liste nommée de tests lourds `-run`/`-skip`), job de fusion (contrôle 356 paquets sur 356, gate baseline, profils additionnés, ratchet de couverture : 81,0 % identique) ; second passage `internal/sync` supprimé ; Vitest en 3 lots fusionnés ; `concurrency` (runs périmés annulés sur les branches de travail, groupe propre au run sur `main`/`feat/v75`) ; filtre de chemins sur les branches de travail (base `feat/v75`) ; `-timeout` revenu à 600 s ; `make gate-push` à deux voies (film sans DuckDB en parallèle). Garde-rails archlint : lots et partition, contournement du filtre, concurrency, voie film sans DuckDB. Run de preuve 38979844734 : 15 min 39 s. |
| 2026-10-16 | **[sécurité] Durcissement de la surface exposée + restes de la démo** — audit adversarial (9 axes, 12 P0 / 17 P1 / 36 P2, `.ai/archive/V7.5/AUDIT_SECURITE_SURFACE_EXPOSEE_2026-10-15.md`) puis plan `.ai/archive/V7.5/PLAN_SECURITE_2026-10-15.md` (lots L0-L11, deux rondes de revue adversariale, recette, E2E) ; fusionné dans `feat/v75` (c6a72cf02). nginx de prod durci le 2026-10-15/16. Reste au déploiement de la v7.5 : V-3 (`data/demo_films/`), V-5 (unité de l'ouvrier), V-6 (limites mémoire après mesure). |
| 2026-10-16 | **[lot backlog du 2026-10-15] Campagne dans les lecteurs, vue match « lecture en échec », intégrité du registre** (branche `feat/backlog-2026-10-15`, plan `.ai/archive/V7.5/PLAN_BACKLOG_2026-10-15.md`, E1 à E6) : lecteurs de la Campagne de `sync`/`ops`/`cmd` statués et garde étendu (E2), dettes Go (temps canonique, `GET /squads`) (E3), raisons `<source>_unavailable` distinctes de `<source>_empty` (E4), colonnes d'intégrité de `match_registry` (E5), deux rondes de revue adversariale corrigées (E6.2a, E6.2b). Reste : fusion dans `feat/v75` (geste du user) et les entrées ci-dessus. |
| 2026-09-27 | **[lot backlog du 2026-09-26, fusion 3] Dérogations mortes du lint inter-features, musique d'intro du rejeu, niveau des fanfares de fin** (branche `feat/backlog-2026-09-26`, plan `.ai/archive/V7.5/PLAN_BACKLOG_2026-09-26.md`, lots A3 et A4). **Lint (5)** : 16 dérogations mortes retirées ; le script échoue désormais sur toute dérogation qui ne sert plus ; README des graphes corrigé (12 wrappers, `FirstBloodLanes` dans `components/charts/`) ; lint ajouté au job web de la CI (`1ea900034`). **Musique d'intro (7)** : les 3 dernières secondes de la montée de la piste `402178411`, résolution R = 3,19 s calée sur le coup d'envoi, −18 LUFS, jouée au clic sur « Lecture » ou « Recommencer » depuis le préambule, jamais à la reprise, au saut ni dans l'export. Au premier « Lecture » après un rechargement, l'intro attend son décodage et part avec un décalage borné par la durée du préambule (`6bac387b8`, `eb57ec3c2`). **Niveau des fanfares (suite de l'item 11)** : les trois fanfares de fin mesuraient −25 LUFS au lieu des −18 prévus (réduction stéréo du 28/08 faite après la normalisation). Remises à −18 LUFS par un gain linéaire, avec un garde-rail de niveau sur les échantillons PCM (`e93c19ed9`). Fins multi-équipes (6) : `[!]`, faute de donnée (entrée ci-dessus). Écoutes de l'utilisateur après la fusion. |
| 2026-09-27 | **[lot backlog du 2026-09-26] Musique de fin et son du rejeu, classement mondial, découpeur SQL, index ART de `match_skill_rank`, garde-rail d'exclusion de la campagne, mode démo hermétique** (branche `feat/backlog-2026-09-26`, plan `.ai/archive/V7.5/PLAN_BACKLOG_2026-09-26.md`, un exécutant Opus à la fois, deux relectures adversariales puis une ronde 2 sur les corrections : 8 constats en ronde 1, dont 2 P1, et 0 P1 en ronde 2). **Musique de fin (11)** : la conclusion échappe au plafond de 8 voix dans la page, comme dans l'export (`soundOccupiesVoice`, `a373539d9`). **Son au rechargement (12)** : le lecteur s'ouvre au premier clic ou à la première touche, ou dès l'affichage si le document a déjà reçu un geste (`useAudioUnlock`, `89f311191`) ; la cause était la lecture automatique. **Classement mondial (1)** : `ErrDrainTimeout`, 3 essais espacés de 30 s sans nouveau scrape, premier tir à +2 min (`cd47936fc`). **Découpeur SQL (3)** : cœur unique `migration.ExecScriptContext`, 3 copies supprimées, garde-rail AST (`edd0054d5`). **Index de `match_skill_rank` (2)** : les trois `idx_msr_*` sont retirés par une migration et par un soin convergent rejoué à chaque ouverture (MSR et PSA) ; ratchet étendu à `sync/schema.go` ; sonde, `repair_msr_index` et `indexcheck` supprimés ; critère D-4 amendé (relatif, `EXPLAIN ANALYZE`) ; copies réelles vérifiées (`7e9ef7c15`, `0b6c11f73`). **Exclusion de la campagne (4)** : garde AST multi-racines (83 lecteurs, 21 dispenses datées), 12 lecteurs corrigés ; les chiffres Halo 5 qui comptaient la campagne changent (`a9e2192ac`, `c61c60761`). **Mode démo (8)** : tâches de fond coupées et chemins d'état redirigés par le SEUL serveur démo (`config.LoadServer`), 4 mutations en 403 `demo_mode_forbidden`. La preuve de bout en bout sur le vrai checkout donne 0 écriture (`3de419efe`, corrections B-C1 à B-C10). Écoutes de l'utilisateur (11, 12) faites après la fusion. Découvertes versées en 6 entrées ci-dessus. |
| 2026-09-19 | **[hygiène] Lot compare / armes / frontières** (branche `feat/hygiene-compare-armes`, 4 commits `e4238dea6`→`aa1a8dc2c`, exécuté par Opus sous pilotage) — **A** champ `filters` de `CompareRequest` retiré (Go + `types.ts` ; le fragment OpenAPI manuel ne le déclarait déjà pas, `FilterContextInput` conservé : 7 autres consommateurs). **B** repli « échantillon croisé » SUPPRIMÉ — mais la prémisse du backlog était fausse : la branche n'était pas morte, elle était FAUTIVE. `GetCrossMatchSample` n'excluait pas la campagne alors que `GetLocalStats` le fait ; mesuré sur copie du shared Halo 5 : pour un B présent uniquement en coop campagne, `GetLocalStats` rend 0 ligne et l'échantillon croisé rend 1 match (stats de campagne servies sous un service record matchmade). Exclusion alignée ⇒ branche morte par construction ⇒ retrait complet (service, repo, port + noop, `domain.CrossMatchSample`, `IsLocalSample`/`is_local_sample` régénéré par Huma, 2 tests, 8 lignes de baseline). **C** `buildTopWeapons` des séries temporelles délègue à `topWeaponKillRows` (départage sur le libellé) ; garde-rail étendu au motif `WeaponID <` sans propriétaire ; changement assumé : une arme sans libellé résolu n'est plus publiée (barre anonyme avant). **D** `SynthesisCards` → `components/ui/section-primitives.tsx`, `SynthesisWeaponAccuracyChart` → `components/charts/WeaponAccuracyChart.tsx` ; DEUX dérogations retirées (`timeseries=>synthesis` et `session-detail=>synthesis`) ; le ratchet du script compte les violations non déclarées (7/7, inchangé), pas les dérogations. Gates : Go build/vet/test 22 paquets, openapi-gen -check, types frais, tsc -b (cache purgé), eslint 0 erreur, lint inter-features, vitest 7984 tests — tous verts, rejoués par le pilote. Découvertes → 2 items backlog ci-dessus. |
| 2026-08-03 | **[ops/deps] Bump `echarts` 5.6.0 → 6.1.0** (CVE-2026-45249, XSS) — livré par `545b870de` (lot B4 echarts6, diff visuel joint). L'entrée « REPORTÉ » du backlog était restée après la livraison ; retirée le 2026-09-19. |
| 2026-07-26 | **[ops/prod] Écritures `app_settings.json` dans le conteneur (bind-mount fichier → rename EBUSY)** (v7.3, `branche feat/v7.3-notion-batch, lot backlog du 26/07`) — point d'écriture unique `internal/platform/atomicfile.WriteFile` : atomique (temp + rename) d'abord, repli **in-place** (truncate + un seul Write + fsync) quand le rename répond EBUSY ou que le répertoire parent refuse le temporaire. Toute AUTRE erreur de rename reste remontée (le repli couvre une contrainte d'environnement connue, pas un diagnostic manquant). Audit des écritures runtime de settings : **3 call sites**, tous migrés — `settings.Store.Save` (chemin de TOUS les toggles admin `PATCH /settings`, qui faisait un `os.WriteFile` nu, donc jamais atomique), `settings.Store.SaveTitleOverlay`, `notify.writeLastNotifiedVersion` (le bug d'origine : notif Discord « nouvelle version » rejouée à chaque redémarrage). Limite ASSUMÉE et documentée : le repli n'est pas atomique — risque borné (contenu déjà sérialisé en mémoire, un seul Write, fsync, fichiers reconstructibles). Garde-rail `archlint/no_bare_settings_write_test.go` (interdit `os.WriteFile`/`os.Rename` nus dans les packages writers de settings). Tests : rename EBUSY → repli, rename ENOSPC → erreur non masquée, temporaire impossible → repli, troncature, création. |
| 2026-07-26 | **[ops/notifs] Bruit WARN `app_release: emit` pour les comptes auth_only** (v7.3, `branche feat/v7.3-notion-batch, lot backlog du 26/07`) — filtre pur `appReleaseTargets` dans `internal/api/wire/notifications_boot.go` : les profils `auth_only` de `db_profiles.json` (5 en prod, `db_path` vide — ils n'existent que pour le pool de tokens) sont écartés AVANT toute résolution, avec une trace `DebugContext` groupée au lieu de 5 WARN par redémarrage. Découverte au passage, corrigée dans le même filtre : `LoadPlayers()` sans filtre de titre renvoie une entrée par (titre, joueur) alors que la notification est per-JOUEUR → un joueur déclaré sur 2 titres était traité deux fois ; déduplication par slug + ordre d'émission stable. 5 tests unitaires. |
| 2026-07-26 | **[archi/contrat] Reliquats V72-01 — clos** (v7.3, `branche feat/v7.3-notion-batch, lot backlog du 26/07`) — (a) **`securitySchemes`** : le contrat ne déclarait AUCUN mécanisme d'auth ; `sessionCookie` (apiKey/cookie) est désormais posé côté Go (`internal/api/openapi_security.go`) en lisant le nom du cookie à sa source unique `session.CookieName` — aucune exigence `security` par opération n'est ajoutée (une part de la surface est publique par conception ; l'inventaire route→garde reste le ratchet `bare_routes`, qui est exécutable). (b) **UI `/docs`** : `internal/api/openapi_docs.go`, montée sur le routeur RACINE (le `DocsPath` de Huma aurait enregistré la route une fois par sous-routeur, sous son préfixe) et gatée sur `IsProduction() **OU** DemoMode` — la démo est publique et ne pose pas `LEVELUP_ENV`, la gater sur la seule production l'aurait exposée. Sert le document VIVANT (`/docs/openapi.{json,yaml}`), CSP dédiée. (c) **`ApiError.details`** : `huma.SchemaTransformer` sur `humacore.apiError` restaure `oneOf: [object(additionalProperties true), array]`, perdu par le type Go `any` — corps runtime inchangé. **Statués `[!]`, non traités et pourquoi** : validation automatique des inputs + résolveurs cross-champs = CLOS par décision produit (contrat `RawBody`/400 CONSERVÉ, pas de bascule 422) ; les **7 descriptions de schéma racine** restent au fragment manuel — les rapatrier exigerait un `SchemaTransformer` sur 7 types de `internal/domain`, donc d'y importer `huma` alors que ce package n'a **aucune** dépendance externe aujourd'hui : coût architectural disproportionné pour 7 chaînes déjà présentes au contrat publié. |
| 2026-07-25 | **[data/objectifs] Stockpile + Extraction + VIP extraits** (v7.2.1, V721-02) — 18 colonnes ajoutées à `match_objective_stats` par une migration séparée incluant la recréation de `match_objective_stats_latest` (une vue `SELECT *` fige ses colonnes à la création : sans ça, les 18 colonnes restaient invisibles à TOUS les lecteurs). `VipStats` n'était même pas déclaré dans `StatsBundle`. Payloads réels capturés (P0, 10 matchs), fixtures committées, INSERT pur. Vérifié sur copie des vraies bases : 27 → 45 colonnes, 5757 lignes préservées. Restent `EliminationStats`/`InfectionStats`, bloqués faute de donnée (cf. backlog actif). |
| 2026-07-25 | **[feat/citations] 10 citations d'objectifs livrées** (v7.2.1, V721-03) — 10 retenues sur 19 (9 écartées par décision utilisateur). Paliers **calibrés sur données réelles**, ce qui a invalidé les paliers génériques initiaux : 4 de ces citations seraient restées au premier palier à vie (3, 1, 4 et 3 occurrences au total chez le meilleur joueur). Parité EN garantie par un garde-rail neuf (le nom EN n'en avait aucun, seule la description était couverte) + garde-rail d'existence des visuels sur disque. Seed vérifié : 88 → 98, 0 sans nom ni description EN. Visuels de 2 citations : bouche-trous provisoires, l'utilisateur les produit (blocage explicite avant déploiement). |
| 2026-07-25 | **[data/armes] Colonne résiduelle `weapons.name_fr` purgée** (v7.2.1, V721-05) — migration de reconstruction CTAS-swap idempotente (no-op sur une base déjà propre), clé primaire composite recréée, garde anti-perte sur le nombre de lignes. Vérifié sur copie de la vraie metadata : colonne absente, 84 lignes et PK intactes. |
| 2026-07-25 | **[archi/contrat] `DefaultStatus` + routes absentes du contrat** (v7.2.1, V721-04) — portée réelle **23 routes** et non 11 : 7 mentaient en silence (elles publiaient `200` pour un 201/202 sans que le fragment ne les corrige), 1 découverte en plus. Champ `Status` retiré partout où le statut est fixe (une seule source de vérité) + ratchet anti-retour. Et **35 routes** absentes du contrat (et non 38 : 3 étaient un artefact de comptage, deux implémentations déclarant le même contrat) montées via un harnais unique — un second harnais aurait rouvert le canal de dérive contrat/tests fermé en v7.2. Fragment manuel : 3291 → 3237 lignes. |
| 2026-07-25 | **[archi/data] Unifier la SOURCE des noms d'armes** — traductions centralisées dans un fichier TOML par titre keyé `weapon_key` (`config/titles/{halo_infinite,halo_5}/mappings/weapon_names.toml`), résolution via `weapon_key` dans `weapon_resolver.go` (:76-94), colonne `name_fr` retirée du CREATE du registre (`weapon_registry.go` :177-189), garde-rail `weapon_names_completeness_test.go`. Commits `cebe2fed9` (V72-06) + `95dd7b5e3`. |
| 2026-07-25 | **[feat/frags] Sunburst : niveau 2 pour la classe Grenade** — ventilation par type de grenade (`grenadeRoles`, `fragdist.go` :226-258) avec résidu « Autre grenade » et invariant Σ niveau 2 == kills classe respecté, i18n `frags.toml` :124-146. Commit `0eb523bb2` (V72-15.2). |
| 2026-07-25 | **[data/frags] H5 : Σ des classes du sunburst > total (double-comptage mêlée/assassinat)** — `buildGunFragClasses` retranche désormais `MechanicKills` des kills d'arme (`fragdist.go` :93-107), invariant Σ classes == total rétabli. Commit `0eb523bb2` (V72-15.3). |
| 2026-07-25 | **[archi/match-view] Retirer le fallback LIVE du Match view** — `GetMatchMeta` miss renvoie désormais `domain.ErrNotFound` (404 `match_not_found`) au lieu d'un fetch live API ; chaîne `tryCanonicalMatchView`/`WithDataAdapter`/`WithViewerGamertag`/`LoadMatchDetail` H5 entièrement supprimée (`match_view_service.go` :229-244). Commit `468154424` (V72-15.4). |
| 2026-07-18 | **[bug/h5] Fuite d'affichage campagne (287 matchs) — balayage complet** — cause : le mécanisme centralisé d'exclusion (`analysis.campaignExcludedVariantIDs` + fragments) n'était pas appliqué sur `Q5SharedHistory` (LISTE historique + compteur) ni `Q4/Q4MV` (filtres). Fix appliqué à **la source** : liste/filtres + relations/career hub (Q26/Q28/Q28Scoped/QRelationsCoreForm). Un **garde-rail structurel** (scan AST de tous les lecteurs `match_participants`/`mv_player_matches` + `xuid=?`) a révélé la fuite systémique → balayage complet des lecteurs restants (Q10 rencontres, Q19 communs, Q23/Q23b détail-match, Q29Heatmap/Q30Rival moments, Q29TopTeammates/Q30SquadShared/Q31/Q42 escouade). Exempts justifiés (allowlist) : mono-match (Q17/Q17b/Q26), filtré au call site (Q25Template, QRelationsPlayerWinRate), sur-lecture inoffensive (Q25MatchParticipants), code mort (Q30SquadMatches). Nouveau résolveur `resolveCampaignExclusionByMatchID` (sous-requête sans placeholder, sûr avant Sprintf). Tests : structural coverage + comportemental (alias/by-match-id/token). PAS de purge BDD (règle ART). Même session : code mort `Q30SquadMatches` supprimé (règle 7). |
| 2026-07-18 | **[sécurité/autZ] BOLA prestige objet-level clos** — découverte : `WithPlayerSlug` jamais câblé → les routes `{id}` pures (`GetChallenge`/`Update`/`Abandon`/`SuggestNext`/`GetArc`) + `ListMySquads`-avec-escouades échouaient en `ErrPlayerNotResolved`. Fix : middleware `prestigePlayerSlugCtx` stampe le slug du chemin (ownership-gardé) dans le contexte → répare la résolution ET clôt le BOLA par isolation player DB (défis/arcs perso en `stats.duckdb` du joueur du chemin → `{id}` étranger = 404). Défis d'escouade (shared_social, non isolés) : garde `assertMemberUser` ajoutée à `ListSquadChallenges` (`requestedBy` = slug du chemin). Tests service+handler+middleware. Même session : `JoinSquadChallenge` (rejoindre un défi d'escouade par `{id}`) reçoit la même garde d'appartenance. |
| 2026-07-17 | **[data/h5] Classification hors-arsenal + capture mécanique de kill** — 26 IDs classés (véhicule/tourelle/environnement/non-attribué/autres) au donut, exclus de l'insight coach (`16d2a09eb`) ; colonne `kill_kind` persistée à l'ingestion H5 pour cesser de jeter la mécanique (`c13e7f6bc`). Phase 2 (backfill + découpage « Capacités Spartan »/Corps-à-corps/Non-attribué) → backlog actif. Investigations : « Spartan » = bucket d'attribution sans arme (API officielle `weapon_type=Unknown`), concentré dans les modes mêlée. |
| 2026-07-17 | **[ux/relations] Lot G LIVRÉ** (revirement produit) — CSR/tier de la bête noire affiché en dégradation gracieuse (rien si absent). Justification d'abandon initiale corrigée : la donnée EST collectée (`ExtractAllSharedCSRRows` → `match_csrs_latest`) ; couverture nulle en dev car base sociale (1,8 % classé), non représentative des joueurs compétitifs cibles. Backend best-effort + chip front conditionnel + tests. Commit `8570af76a`. |
| 2026-07-17 | **[vérif finale + dettes] passe QA post-train** — audit logging (5 flux best-effort re-routés hors `general.log` vers modules dédiés + erreur avalée `decodeParams` comblée, `e6671ff89`) ; renforcement tests (branches best-effort/gardes nil, `058ba9486`/`d49a1734f`) ; dettes réglées : erreurs avalées coach/prestige/handlers, doc inversée `notifications/types.go`, schéma OpenAPI orphelin `BattlePassResponse` retiré, `SUBTIER_ROMAN` centralisé + garde-rail (3e copie trouvée → seuil franchi). Commits `d48da5912`/`556991069`/`6bb186e9b`/`3c975bde3`. Gates Go unit+intégration `-p 1` (114 ok/0 FAIL) + web (2258 tests) verts. |
| 2026-07-17 | **[data/h5] Inventaire complet des frags hors-arsenal** — `.ai/archive/V7/H5_WEAPON_LONGTAIL_UNMAPPED.md` : 26 IDs non couverts (16 649 frags = 6,2 %), dont bucket « Spartan » 8 812 frags. Base pour une classification produit (catégories véhicule/tourelle/corps-à-corps/environnement à trancher). Commit `fbbfd809a`. |
| 2026-07-17 | **[train backlog] Purges code mort** — `squad/v2` purgé (knip croisé grep, 4 types vivants conservés) ; route `GET /players/{slug}/battlepass` supprimée après preuve de mort (service `GetBattlePass` conservé, consommé par season-pass). Commits `42b317cd8`, `c9bfa0e7d`. |
| 2026-07-17 | **[train backlog] Registre armes H5** — long-tail `v_weapon_kills` : 5 armes tenues mappées (3 grenades + golf club + oddball, ~18,4 k frags) ; véhicules/tourelles/UGC documentés hors-arsenal (item résiduel ouvert). Commit `9e0a8217d`. |
| 2026-07-17 | **[train backlog] i18n tiers + cache défis** — mapping tiers CSR centralisé dans `lib/skillTiers.ts` (+ garde-rail grep) ; défis locale-aware via locale dans les query keys `home`/`seasonPass`. Commits `45200bc4c`, `7fc82f575`. |
| 2026-07-17 | **[train backlog] Release CI (GHCR)** — livré (`7bb6a4257`) puis **RETIRÉ le 2026-07-18** sur décision user (pas d'intérêt) : retour au build local sur le VPS (comportement d'origine). |
| 2026-07-17 | **[train backlog] Prod** — `/health` expose `media_tooling` (sonde ffmpeg au boot, figée) ; `Cache-Control: public, max-age=31536000, immutable` sur les assets Vite hashés (côté Go). Commits `731c3934f`, `5f486b0b1`. |
| 2026-07-17 | **[chantier Relations UX] Plan 2026-07 CLOS** — lots A (tri colonnes), B (duels cliquables → match view via `matchIndexAtX`), C (toggle « jamais affrontés », défaut masqué, migrate store v2), F (CoreCards supprimées + lien Escouade), D (volet « Quoi de neuf » : `is_revived` SQL+DTO+strip front), E (notification `rival_encounter` post-sync par watermark, garde-fous 3/sync et 7 j), H régularisé `[~]`, G `[!]` (0 % couverture CSR rivaux). Plan archivé : `.ai/archive/V7/PLAN_RELATIONS_UX_2026-07.md`. Commits `91649c01d`→`e38592ca9`. |
| 2026-07-17 | **[sécurité] BOLA acteur prestige clos** — 14 routes non-squad réconcilient l'acteur body/query avec la session (`authorizeActor`, 403 `player_forbidden`), garde-rail AST structurel anti-régression, tests 403+verts par endpoint. Commit `b8e97cb43`. Résiduel objet-level → item backlog. |
| 2026-07-17 | **[coach/prestige] V2.1 + V3 livrés** — colonne `source` (challenge + prestige_telemetry, migration `prestige_add_source_columns_v1`), source renseignée aux 4 origines (coach/user/pilot_mode/preset), endpoint diag `GET /_diag/prestige/telemetry/{slug}` ; analyseur `cmd/prestige-tuning-analyze` (recommandations grammaire, seuils 30 %/50, validation manuelle) ; canal Discord webhook pour notifications coach (opt-in `discord_notify_coach`, OFF par défaut, best-effort). Commits `709bac9ad`, `7fa643699`, `02797d4cb`, `2f30e180d`. |
