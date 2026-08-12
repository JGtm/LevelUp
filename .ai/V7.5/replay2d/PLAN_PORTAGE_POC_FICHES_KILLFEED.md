# Plan — Portage du POC : fiches joueur + kill feed du rejeu 2D (au niveau POC, dans le design de l'app)

> Branche `feat/v75` (mode branche unique v7.5, JAMAIS de merge vers `main`). Rejeu OFF prod :
> garde `handlers/replay_local_gate.go` INCHANGÉ. Contrat d'exécution : skill `plan-execution`
> (ordre strict, une étape à la fois, chaque item statué, zéro fix hors périmètre, gate par
> commande). Reprise : section « Suivi » en bas.

## 0. LE PROBLÈME QU'ON CORRIGE (à lire avant tout)

La Phase 1 a **RÉINVENTÉ** les fiches joueur du rejeu, en moins bien que le POC que l'utilisateur
a bâti sur des semaines. Régressions confirmées : **santé supprimée**, inventaire en **libellés
texte** au lieu d'**icônes**, pas de **grenade sélectionnée**, pas d'**animation d'échange
d'arme** ; et le kill feed de la page replay n'a **ni l'assistant ni sa part de dégâts %**. Ce lot
ne réinvente rien : il **PORTE le POC** (comme contrat de rendu) sur les données/assets de prod,
re-exprimé dans le **design system de l'app**.

## 1. LE POC EST LA SPEC DE RENDU — le fetch et le prendre comme cible

Artefact : **https://claude.ai/code/artifact/eb7b8af2-94cb-47c6-9cdb-15af465b12ae**
(récupérable par l'outil WebFetch ; HTML complet 3,9 Mo, CSS+JS auto-documentés). **L'exécuteur
OUVRE le POC et le prend comme référence de rendu, feature par feature.** Il NE copie PAS son CSS
ni ses valeurs hex : il porte le **comportement + la disposition** dans le design de l'app.

Inventaire des features du POC (cible) :
- **Fiche joueur** : nom + KDA (frags/morts/assists, 3 couleurs) ; **VIE (i4) ET BOUCLIER (i5)**,
  deux barres, report + estompage d'âge, cachées à la mort ; **armes portées en ICÔNES** (arme en
  main à GAUCHE, souligné = primaire slot 0, **animation d'échange** au basculement du sélecteur) ;
  **grenades en icônes** + compte « ×N » ; **grenade SÉLECTIONNÉE** encadrée ; **capacité en
  icône** + nom ; **munitions** par emplacement (dégainé surligné) + **jauge de charge** ;
  **respawn** (compte à rebours + barre) ; états **mort/renaissance** animés ; estompage `--fr`
  (lecture d'inventaire vieillit → pâlit).
- **Kill feed** : tueur → victime, **arme (icône)**, horodatage, et **l'ASSISTANT + sa part de
  dégâts %** (3 états : nommé / aucun-MESURÉ / inconnu ; fond bleuté sur morts assistées).
- Capacité ACTIVE (i57) et compteur d'utilisations : **TESTÉS puis RETIRÉS du POC** (i57 constant,
  compteur non localisé offline) — **NE PAS chercher à les remettre**, ce ne sont pas des manques.

## 2. ASSETS — RÈGLE UTILISATEUR, NE PAS SE TROMPER

- Le POC utilise des **images DESSINÉES À LA MAIN** (placeholder) : **NE PAS les reprendre**.
- **Inventaire des fiches** (armes portées, grenades, capacité) → les **icônes EXTRAITES** via
  `apps/web/src/components/ui/WeaponIcon.tsx` + `apps/web/src/lib/staticAssets.ts`
  (`static/weapons-assets/halo_infinite/...`). Composant déjà existant, réutiliser.
- **Kill feed** → les **icônes killicon** (déjà câblées dans `MatchKillFeed.tsx` /
  `ReplayKillFeed.tsx`). Réutiliser.

## 3. RÈGLES UI DE L'APP — non négociables (le POC est la cible fonctionnelle, PAS la source CSS)

- **Couleurs** : tokens sémantiques UNIQUEMENT (skill `color-tokens`, `tokenCssVar`). **ZÉRO hex,
  zéro classe Tailwind couleur** dans `features/`/`components/`. Mapper les couleurs du POC aux
  tokens : vie → token santé (success), bouclier → info, mort → destructive, assist → info/accent,
  accent grenade sélectionnée → accent. Le grep anti-hex du gate doit rester à 0.
- **i18n** : toute string UI en **FR + EN** (`match-replay/i18n.ts`, parité par typage
  `Record<Locale, T>`). Pas de FR/EN en dur.
- **Réutiliser l'existant** : `WeaponIcon`, `heldReading`/`freshness` (`replayLogic.ts`), les
  parties CORRECTES de la Phase 1 (respawn, munitions, drawn-slot i42) ; skills
  `frontend-patterns` (structure features, TanStack, routing), `arch-rules` (couches Go).
- **Seuils** : fichier ≤ 500 L, fonction ≤ 80 L ; ≤ 2 copies d'un pattern.

## 4. PÉRIMÈTRE

### Phase A — Données (Go, offline-pur ; garde local inchangé)

- [x] **A.1 Grenade SÉLECTIONNÉE (i47)** dans l'inventaire keyframe. Grammaire ÉTABLIE
  (`.ai/V7.5/replay2d/RECETTE_LOADOUT_2026-07-27.md` §1 : `i47 = [6b masque = bitmap des compteurs
  i22 non nuls][3b sélection base-1]`, accord i22↔i47 **194/194**). Câbler dans
  `ScanFilmKeyframeInventory` (`replay/inventory_decode.go`) → nouveau champ `Inventory`
  (ex. `SelectedGrenadeRank *int`). **VALIDER offline** sur `000d5950` (le type sélectionné doit
  être cohérent avec les compteurs i22). Contrat : `go run ./cmd/openapi-gen` + `make generate-types`.
  - Gate : `go test ./internal/analysis/filmdec/ ./internal/analysis/replay/` ; `make generate-types`
    sans diff non commité ; couverture chiffrée dans le CR.
  - FAIT (2026-08-12). i47 n'était PAS localisé dans les keyframes (le POC affichait ses 98
    sélections par « unique compteur non nul », jamais par i47) : la position a été MESURÉE par
    instrument rejouable (`replay/i47_research_test.go`, garde `I47_FILM`) — fenêtre
    [+200..+210] bits après la fin de la DERNIÈRE famille d'arme, 69/92 lectures à +203..+205.
    Règle R5 en production (`invGrenadeSelection`) : masque == bitmap i22 + sélection ∈ masque +
    UNANIMITÉ en fenêtre, sinon non lu. Couverture 000d5950 : 92/120 records à i22 lu (74/98 à
    un type porté, 18/22 à 2+ types — l'information nouvelle), 1 ambigu refusé. Validation :
    stabilité intra-vie 26/29 ; oracle des décréments i22 : sélection AVANT lancer juste 7/7,
    dont 2/2 non tautologiques ; APRÈS 4/8 (attendu : la sélection bouge après un lancer).
    Publié `Inventory.Gs *int` (`gs` au contrat), rendement figé au test
    (`wantInvGrenadeSel = 92`) + invariant « sélection ⇒ compteur porté ».
- [x] **A.2 Assistant + PART DE DÉGÂTS % par kill.** D'ABORD vérifier sur pièces si c'est déjà
  décodé/exposé : killsource (`internal/games/halo_infinite/film/killsource/`) et
  `domain.MatchHighlightEvent`. Le POC le décode du film (bloc `assistMeta`) ; le kill-event
  component porte `+0x10` = assistant, `+0x14` = % dégâts assistant (cf. FIRE_MELEE_GRENADE_EVENTS
  §7). Si absent du contrat : l'exposer par kill dans `highlight_events` / `MatchHighlightEvent`
  (assistant nommé + % + % du tueur), avec **3 états HONNÊTES** : `nommé` / `aucun` (MESURÉ : la
  mort porte son événement, sans assistant) / `inconnu` (non lu — ne JAMAIS écrire « pas
  d'assistant » quand c'est « on ne sait pas »). Contrat regen.
  - Gate : `go test` (killsource + match_view) ; `make generate-types` ; couverture assist mesurée
    dans le CR (sur 000d5950 : combien de morts ont un assistant nommé / aucun / inconnu).
  - FAIT (2026-08-12). Vérifié sur pièces : killsource décodait DÉJÀ tout (`Assist{Name, Known,
    Extra}`, `KillerDamage`/`AssistDamage` non bornées, publication sous `LineByLinePublishable`)
    et la base portait DÉJÀ les colonnes (`assist_gamertag/assist_xuid/assist_known/
    killer_damage_pct/assist_damage_pct`, DDL 3 états). Le manque était la REMONTÉE : livré
    Q21c (`Q21cKillAssists`, sœur de Q21b, indépendante de `source_tag`, unanimité par tuple
    complet sur (tueur, instant) — validée sur DuckDB : désaccord exclu, doublon unanime
    dédoublonné), `domain.KillAssistRaw`, `GetMatchKillAssists` (port+repo, dégradation
    gracieuse), extension de `decorateKillFeed` et `MatchHighlightEvent.{AssistState
    (""/none/named), AssistGamertag, AssistTeamID, KillerDamagePct, AssistDamagePct}`.
    Couverture 000d5950 (compteurs figés killsource) : 93/93 morts attachées, 17 nommées,
    76 « aucun » mesurés, 0 inconnue ; les 17 nommées = total API à l'unité.
- Santé i4 : **déjà** dans `Point.Hp` (artefact) — rien côté données, c'est un oubli de câblage
  front (Phase B.1).

### Phase B — Fiches joueur (front : porter le POC dans le design app)

- [x] **B.1 Vie (i4) + bouclier (i5)** : DEUX barres, report (`heldReading`) + estompage d'âge,
  cachées à la mort. **RÉTABLIR la santé** (supprimée en Phase 1) sur le MÊME patron que le
  bouclier — le POC la montre par report malgré la couverture faible. Tokens : vie = santé/success,
  bouclier = info. Jamais 100 % par défaut ; lacune = rien.
  - FAIT (2026-08-12). Les deux barres existaient (santé rétablie par `71bdb589f`) ; ce lot porte
    le REPORT au patron POC : maintien sur TOUTE la vie (le flux est différentiel, non retransmis
    = inchangé ; les points appartiennent à la vie donc aucun report ne franchit une mort),
    estompage complet à 6 s (`VITALITY_FADE_MS`, même graduation que la carte), lacune = jamais
    mesuré dans la vie. Le « assumed 1.0 » du POC n'est PAS repris (règle du plan : jamais un
    plein par défaut). Test : report jusqu'à fin de vie avec âge.
  - AMENDÉ (2026-08-12, décision UTILISATEUR — retour sur CR) : le « jamais 100 % par défaut »
    du plan est ANNULÉ pour la vitalité — au spawn, vie et bouclier sont PLEINS (règle du jeu),
    et le flux différentiel ne retransmet que ce qui change : avant la première mesure d'une
    vie, la valeur juste est 1,0 (âge 0), pas une lacune. C'est la doctrine du POC (`lastOf`
    assumed), rétablie. Les libellés « bouclier/santé non transmis » sont SUPPRIMÉS. GARDE
    multi-titre : `vitalityPresence(doc)` par champ — un document dont AUCUN point ne porte
    sh/hp (titre sans décodage film) n'affiche PAS de barres (dégradation par absence de
    donnée, zéro slug).
- [x] **B.2 Inventaire en ICÔNES** via `WeaponIcon` + `staticAssets` : armes portées (arme EN MAIN
  à GAUCHE via `Inventory.D`/i42 déjà décodé, souligné = primaire slot 0, **animation d'échange**
  au basculement du sélecteur — cf. POC `wswapL/R`), grenades (icône + « ×N »), capacité (icône +
  nom). Repli libellé si pas d'icône. Estompage d'âge (`--fr`/`freshness`).
  - FAIT (2026-08-12), avec DEUX écarts au plan CONSIGNES en §6 : (1) la marque est « EN MAIN »
    (souligné + pleine encre, l'autre estompée), pas « primaire » — c'est le rendu RÉEL du POC
    final (aucune règle CSS `.prim` n'y existe) ; (2) grenades et capacités N'ONT PAS d'icône
    extraite dans le dépôt (les visuels du POC sont dessinés main, interdits par §2 ; les
    grenades du jeu sont des `eqip`+`proj`, hors du balayage `weap`) → REPLI LIBELLÉ, prévu au
    plan. Armes : icônes extraites servies par le document (`WeaponLabel.img/tinted`, jointure
    famille→tag `weap` posée par la couche titre `replaylabels` + adapter), rendues par
    `WeaponIcon` (masque teinté par `currentColor`). Animation d'échange : `drawnSwapAt`
    (bascule i42 entre keyframes) + keyframes `replay-wswap-l/r` à délai négatif (globals.css).
- [x] **B.3 Grenade SÉLECTIONNÉE** (A.1) encadrée, token accent + nom en accent.
  - FAIT (2026-08-12). `selectedGrenade` : la LECTURE (`gs`, i47) prime, la déduction « seul
    type porté » reste (dite déduite en infobulle), et 2+ types sans lecture = « sél. ? »
    (indéterminé affiché, jamais deviné). Encadré token `warning` (l'accent de l'app) : liseré
    + fond `color-mix` 13 % + nom en accent, patron POC `.gsel`.
- [x] **B.4 Munitions** par emplacement (dégainé surligné, index d'emplacement), **jauge de
  charge** (fraction 0..1, barre — pas un %), **respawn** (compte à rebours + barre), états
  **mort** (fond tenu, liséré, nom teinté) / **renaissance** (flash), `prefers-reduced-motion`.
  Réutiliser ce qui est correct en Phase 1.
  - FAIT (2026-08-12). Munitions : cellules dans l'ORDRE des armes (helper `order` partagé —
    même lecture que la rangée, jamais deux ordres), dégainée surlignée (index accent + encre
    franche), jeton fin de ligne « rangées » (mesuré, D=2) / « dégainée ? » (non lu) ; jauge
    inchangée (complément, barre). Respawn : barre d'avancement depuis la mort (mort datée par
    la fin de vie, retour lu au départ de la suivante ; sans mort datée : compte sans barre) +
    « retour ? » lacune inchangé. Mort : fond rouge tenu (`color-mix` 12 %) + liséré + nom
    teinté ; ÉCLATS courts coup-fatal (`replay-death-flash`, steps(1)×3) et réapparition
    (`replay-respawn-flash`) à délai négatif (justes après un saut de lecture) ;
    `prefers-reduced-motion: reduce` coupe les trois animations.
- [x] **B.5 Design app** : toutes strings i18n FR+EN ; **0 hex** (tokens sémantiques) ; réutilisation
  `WeaponIcon`/`heldReading`/`freshness`.
  - Gate B : `make check-types` ; `make test-web` (tests rendu + report + lacune + dégradation) ;
    `grep` anti-hex à 0 ; lint i18n vert.
  - PASSÉ (2026-08-12) : check-types vert ; test-web 3615 verts / 409 fichiers (1 flake au
    premier passage hors match-replay, non reproduit sur deux runs complets) ; grep anti-hex
    match-replay = 0 (seul hit : un commentaire préexistant qui documente le refus du littéral) ;
    parité i18n par typage `Record<Locale, ReplayText>` (compilée). 13 clés i18n ajoutées FR+EN.

### Phase C — Kill feed de la page replay (front : porter le POC dans le design app)

- [x] **C.1 Assistant + part de dégâts %** (A.2) : 3 états (nommé/aucun/inconnu), fond bleuté
  (token) sur morts assistées, la part de dégâts collée à qui elle appartient (tueur + assistant).
  - FAIT (2026-08-12). `KillEvent` étendu (assistState/assistGamertag/assistTeamID/parts,
    propagés par `collectKillEvents`) ; rendu POC : nommé = « + Nom part% · tueur part% »
    (le « + » à la couleur d'équipe de l'assistant) + fond bleuté `color-mix(info 10%)` —
    le fond affirme une contribution, jamais un trou ; « aucun » = RIEN d'affiché,
    l'information vit en infobulle (mesurée, distincte d'inconnu — 76/93 morts sur le film
    de référence, l'écrire remplirait le fil) ; « inconnu » = « ? assistant inconnu ».
- [x] **C.2 Icônes killicon** (déjà dispo) ; format POC (tueur → victime, arme, %, assistant),
  design app.
  - FAIT (2026-08-12). La VICTIME est jointe par (tueur, instant) depuis `killer_victim`
    (`attachVictims`) avec la règle d'unanimité du back : deux victimes distinctes sur la
    même clé → aucune nommée. Ligne : [icône killicon inchangée] tueur → victime,
    horodatage à droite.
- [x] **C.3** i18n FR+EN + tokens.
  - Gate C : `make test-web` (rendu des 3 états d'assistant, lacune) ; anti-hex 0.
  - PASSÉ (2026-08-12) : 208 tests match-replay + 186 match-view verts (rendu des 3 états,
    victime jointe/absente/désaccord, garde anti-hex du composant inchangée) ; 5 clés i18n
    FR+EN ajoutées.

### Multi-titre (transverse)

Le décodage film est **Halo Infinite only**. Titre sans film / H5 → champs absents (`hp`,
`selected grenade`, `assistant`) → **dégradation** : la fiche/le feed s'affichent sans ces lignes,
**zéro comparaison de slug**, zéro panic, zéro donnée d'un autre titre.

## 5. GATE / DONE DEFINITION

Par phase : gate ci-dessus (commandes exactes). Global : `make check-types` + `make test-web` +
`go test` verts ; `make generate-types` sans diff ; `make gate-push` ; **gate visuel UTILISATEUR**
= comparaison directe au POC sur `000d5950` (les témoins viennent de l'utilisateur) ; entrée
`.ai/thought_log.md` ; pousser sur `feat/v75`, **CI verte au niveau job**.

## 6. DÉCOUVERTES (à consigner ici pendant l'exécution, NE PAS traiter hors périmètre)

- (A.1, 2026-08-12) **La marque verte du POC FINAL dit « en main », pas « primaire »** : le CSS
  livré n'a AUCUNE règle `.prim` (la classe est posée par le JS mais sans effet) et son
  commentaire le dit en toutes lettres (« UNE SEULE MARQUE, ET ELLE DIT EN MAIN »). Le §4 B.2 de
  ce plan (« souligné = primaire slot 0 ») décrit une doctrine antérieure du POC. Le rendu
  RÉEL du POC fait foi : B.2 portera la marque unique « en main » (arme dégainée à gauche,
  pleine encre, soulignée ; l'autre estompée quand le sélecteur est lu ; sans sélecteur lu :
  ordre du record, encres égales, aucune marque).
- (A.1, 2026-08-12) Le POC n'a JAMAIS lu i47 (98 × « unique compteur non nul », « sél. ? » à
  2+ types) : la lecture i47 keyframe livrée en A.1 va AU-DELÀ du POC — 18 sélections nommées
  sur 22 états à 2+ types, là où le POC affichait « sél. ? ».
- (clôture, 2026-08-12) **`make go-api-test` échoue en LOCAL sur `analysis/weaponv3`**
  (`TestBuildV3Attributions_Smoke000d5950` : paquet à 61,2 s pour un budget `-timeout 60s`,
  smoke mesuré à 54 s isolé). INDÉPENDANT de ce lot — `git diff` de la base du lot sur
  `weaponv3/` et `filmdec/` est VIDE, aucun import du code touché. Dette locale de budget
  (même famille que himap : lent en local seulement) ; NON traitée ici (hors périmètre),
  la CI de branche fait foi. Tous les paquets TOUCHÉS par le lot sont verts.

## 7. SUIVI / REPRISE

- Avancement = les cases de ce fichier + `thought_log.md`. Reprise : lire ce plan, **OUVRIR LE POC**
  (WebFetch de l'artefact), `git log --oneline -8 feat/v75`, rouvrir la première case non cochée.
- Contrat : skill `plan-execution`. Le POC est la cible de rendu ; le design de l'app est la
  contrainte de forme. Aucune réinvention : porter, réutiliser, respecter les tokens/i18n.
