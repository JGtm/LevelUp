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

- [ ] **A.1 Grenade SÉLECTIONNÉE (i47)** dans l'inventaire keyframe. Grammaire ÉTABLIE
  (`.ai/V7.5/replay2d/RECETTE_LOADOUT_2026-07-27.md` §1 : `i47 = [6b masque = bitmap des compteurs
  i22 non nuls][3b sélection base-1]`, accord i22↔i47 **194/194**). Câbler dans
  `ScanFilmKeyframeInventory` (`replay/inventory_decode.go`) → nouveau champ `Inventory`
  (ex. `SelectedGrenadeRank *int`). **VALIDER offline** sur `000d5950` (le type sélectionné doit
  être cohérent avec les compteurs i22). Contrat : `go run ./cmd/openapi-gen` + `make generate-types`.
  - Gate : `go test ./internal/analysis/filmdec/ ./internal/analysis/replay/` ; `make generate-types`
    sans diff non commité ; couverture chiffrée dans le CR.
- [ ] **A.2 Assistant + PART DE DÉGÂTS % par kill.** D'ABORD vérifier sur pièces si c'est déjà
  décodé/exposé : killsource (`internal/games/halo_infinite/film/killsource/`) et
  `domain.MatchHighlightEvent`. Le POC le décode du film (bloc `assistMeta`) ; le kill-event
  component porte `+0x10` = assistant, `+0x14` = % dégâts assistant (cf. FIRE_MELEE_GRENADE_EVENTS
  §7). Si absent du contrat : l'exposer par kill dans `highlight_events` / `MatchHighlightEvent`
  (assistant nommé + % + % du tueur), avec **3 états HONNÊTES** : `nommé` / `aucun` (MESURÉ : la
  mort porte son événement, sans assistant) / `inconnu` (non lu — ne JAMAIS écrire « pas
  d'assistant » quand c'est « on ne sait pas »). Contrat regen.
  - Gate : `go test` (killsource + match_view) ; `make generate-types` ; couverture assist mesurée
    dans le CR (sur 000d5950 : combien de morts ont un assistant nommé / aucun / inconnu).
- Santé i4 : **déjà** dans `Point.Hp` (artefact) — rien côté données, c'est un oubli de câblage
  front (Phase B.1).

### Phase B — Fiches joueur (front : porter le POC dans le design app)

- [ ] **B.1 Vie (i4) + bouclier (i5)** : DEUX barres, report (`heldReading`) + estompage d'âge,
  cachées à la mort. **RÉTABLIR la santé** (supprimée en Phase 1) sur le MÊME patron que le
  bouclier — le POC la montre par report malgré la couverture faible. Tokens : vie = santé/success,
  bouclier = info. Jamais 100 % par défaut ; lacune = rien.
- [ ] **B.2 Inventaire en ICÔNES** via `WeaponIcon` + `staticAssets` : armes portées (arme EN MAIN
  à GAUCHE via `Inventory.D`/i42 déjà décodé, souligné = primaire slot 0, **animation d'échange**
  au basculement du sélecteur — cf. POC `wswapL/R`), grenades (icône + « ×N »), capacité (icône +
  nom). Repli libellé si pas d'icône. Estompage d'âge (`--fr`/`freshness`).
- [ ] **B.3 Grenade SÉLECTIONNÉE** (A.1) encadrée, token accent + nom en accent.
- [ ] **B.4 Munitions** par emplacement (dégainé surligné, index d'emplacement), **jauge de
  charge** (fraction 0..1, barre — pas un %), **respawn** (compte à rebours + barre), états
  **mort** (fond tenu, liséré, nom teinté) / **renaissance** (flash), `prefers-reduced-motion`.
  Réutiliser ce qui est correct en Phase 1.
- [ ] **B.5 Design app** : toutes strings i18n FR+EN ; **0 hex** (tokens sémantiques) ; réutilisation
  `WeaponIcon`/`heldReading`/`freshness`.
  - Gate B : `make check-types` ; `make test-web` (tests rendu + report + lacune + dégradation) ;
    `grep` anti-hex à 0 ; lint i18n vert.

### Phase C — Kill feed de la page replay (front : porter le POC dans le design app)

- [ ] **C.1 Assistant + part de dégâts %** (A.2) : 3 états (nommé/aucun/inconnu), fond bleuté
  (token) sur morts assistées, la part de dégâts collée à qui elle appartient (tueur + assistant).
- [ ] **C.2 Icônes killicon** (déjà dispo) ; format POC (tueur → victime, arme, %, assistant),
  design app.
- [ ] **C.3** i18n FR+EN + tokens.
  - Gate C : `make test-web` (rendu des 3 états d'assistant, lacune) ; anti-hex 0.

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

- (vide au démarrage)

## 7. SUIVI / REPRISE

- Avancement = les cases de ce fichier + `thought_log.md`. Reprise : lire ce plan, **OUVRIR LE POC**
  (WebFetch de l'artefact), `git log --oneline -8 feat/v75`, rouvrir la première case non cochée.
- Contrat : skill `plan-execution`. Le POC est la cible de rendu ; le design de l'app est la
  contrainte de forme. Aucune réinvention : porter, réutiliser, respecter les tokens/i18n.
