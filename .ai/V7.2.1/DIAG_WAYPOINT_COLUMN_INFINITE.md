> # RÉSOLU LE 2026-07-26 — NE PAS TRAVAILLER À PARTIR DE CE DOCUMENT
>
> **La cause racine annoncée en section 0 (« le second verrou, la préférence
> `showWaypointColumn` du localStorage ») est FAUSSE.** La colonne était visible et le
> réglage activé ; c'est le LOGO à l'intérieur de la cellule qui ne se rendait pas.
>
> Correctif livré : commit `c22627074`, sur `ExplorerMatchesTable.tsx` et
> `SquadSynergyHistoryTable.tsx`. **Trois causes CSS distinctes**, aucune liée à une
> capability ni à une préférence :
>
> 1. Le lien était un conteneur flex sans dimension propre, dans une cellule de tableau en
>    largeur automatique. Chrome et Firefox ne résolvent pas la taille intrinsèque de la
>    même façon dans ce cas — **Firefox écrasait l'image à zéro, Chrome non**. D'où un bug
>    invisible à toute mesure prise sous Chrome. Le lien porte désormais une taille fixe.
> 2. Attributs HTML `width`/`height` absents : le moteur ne connaissait pas la taille avant
>    de résoudre le CSS. C'est le point qui manquait côté Firefox.
> 3. L'asset est un logotype 360x160 forcé dans un carré 16x16 sans `object-fit` — écrasé en
>    traits sub-pixel sur **tous** les navigateurs, y compris ceux où il apparaissait.
>    `object-contain` rétabli.
>
> **Ce qui reste valable dans ce document** : la section 1 (mécanisme des deux verrous),
> l'établissement que `waypoint_match_url` EST bien déclarée pour Halo Infinite dans le
> descripteur Go `internal/domain/title/registry.go` (et non dans un TOML — ne pas créer
> `config/titles/halo_infinite/title.toml`), et le constat de documentation inversée (déjà
> corrigé, commit `aa3c40630`).
>
> **Ce qui est faux ou incomplet** : le résumé exécutif, la conclusion par élimination, et
> neuf points relevés par une contre-revue (inventaire des consommateurs incomplet, page
> Session écartée à tort, mécanisme de synchronisation inter-onglets, sémantique TanStack de
> `getIsVisible`, commande console non protégée contre `null`). Ils ne sont pas corrigés ici :
> le document est clos, pas maintenu.
>
> Analyse complète et leçon de méthode : `.ai/thought_log.md`, entrée du 2026-07-26
> « Logo Waypoint invisible : trois causes, une seule visible sous Firefox ».

# DIAG — Colonne Halo Waypoint invisible sur Halo Infinite (Explorer)

Date : 2026-07-26. Investigation seule, aucune modification de code.
Worktree d'enquête : `.claude/worktrees/v721-notion` (branche `feat/demo-prestige-samples`).
Prod interrogée en lecture seule (`curl`) : `https://lvelup.info`.

---

## 0. Resume executif

**La prémisse de la demande est fausse.** La capability `waypoint_match_url` **EST** déclarée
pour Halo Infinite, et la production la sert. Toute la chaîne serveur, build, déploiement et
assets est saine pour les deux titres. Il n'y a **aucun correctif de capability à faire**, et
il ne faut **pas** créer `config/titles/halo_infinite/title.toml` (il serait ignoré).

Cause racine restante, par élimination : le **second verrou**, la préférence locale
`showWaypointColumn` persistée dans `localStorage`. Point décisif souvent manqué :
**vider le cache navigateur ne vide pas le localStorage.**

Le seul défaut de code réellement établi est une **documentation inversée** (5 emplacements)
qui affirme que Halo 5 ne déclare pas la capability — faux depuis le 2026-07-24. C'est cette
doc qui a produit le faux diagnostic initial.

---

## 1. Le mécanisme exact (chaine de visibilite)

La colonne est un ET de deux verrous, dans un seul endroit :

`apps/web/src/features/explorer/ExplorerMatchesTable.tsx:780-783`

```
const internalColumnVisibility = useMemo(
    () => ({ waypoint: waypointCapability && showWaypointColumnPref }),
    [waypointCapability, showWaypointColumnPref],
  )
```

fusionné dans l'état TanStack en `ExplorerMatchesTable.tsx:793` :
`columnVisibility: { ...internalColumnVisibility, ...(columnVisibility ?? {}) }`
(la prop consommateur a la priorité).

- **Verrou 1** — `useCapability('waypoint_match_url')` (`ExplorerMatchesTable.tsx:285`)
- **Verrou 2** — `localUiPrefs.showWaypointColumn` (`ExplorerMatchesTable.tsx:286`)

La définition de la colonne elle-même est en `ExplorerMatchesTable.tsx:343-364` (`id: 'waypoint'`),
jumelle en `apps/web/src/features/squad/SquadSynergyHistoryTable.tsx:104-106`.

---

## 2. Ce que j'ai PROUVE (avec la piece)

| # | Fait établi | Preuve |
|---|---|---|
| 1 | La capability **est** déclarée pour Infinite | `apps/go-api/internal/domain/title/registry.go:307` — `CapWaypointMatchURL` dans la liste `Capabilities` du descripteur **built-in** construit par `NewRegistry()` (`registry.go:290-316`) |
| 2 | La **prod** la sert, pour les **deux** titres | `curl -s https://lvelup.info/api/v1/bootstrap` → `available_titles[halo_infinite].capabilities` **et** `[halo_5].capabilities` contiennent `waypoint_match_url` |
| 3 | Les assets sont bien en prod | `/icons/halowaypoint-white.png` → `200 image/png 8910` o ; `-black.png` → `200 image/png 8907` o — **tailles exactes** des fichiers `apps/web/public/icons/` du repo |
| 4 | Le bundle prod déployé **contient** la colonne | chunks servis `waypointUrl-sRvm6oWz.js` et `ExplorerMatchesTable-DMqzvOcd.js` ; le second contient `columnVisibility:{...useMemo(()=>({waypoint:W&&G}),[W,G]),...x??{}}` |
| 5 | Le `useCapability` déployé est **fail-open** | dans `index-2T9EiNyN.js` : `function Lo(e){return k(t=>{let n=t.availableTitles.find(...);return n?(n.capabilities??[]).includes(e):!0})}` — `:!0` = `true` si le titre est introuvable |
| 6 | Le défaut de la préférence est **`true`** | `apps/web/src/stores/settingsDraftStore.ts:93` |
| 7 | Un `localStorage` hérité ne peut pas **perdre** la clé | `settingsDraftStore.ts:182-198` : `merge` custom qui fusionne `localUiPrefs` **clé par clé** par-dessus les défauts (pas de remplacement en bloc) |
| 8 | L'Explorer ne masque **rien** | `ExplorerPage.matchesMode.tsx:350` et `ExplorerPage.playerMode.tsx:201`/`:223` ne passent **aucune** prop `columnVisibility` |
| 9 | Pas de double instance de store | `levelup-ui-prefs` n'est défini **qu'une fois** dans `index-2T9EiNyN.js` ; le chunk Explorer importe le store depuis ce même chunk (`import{...,dt as s,...}from"./index-2T9EiNyN.js"`) |
| 10 | Le `dist` local n'est pas périmé | `apps/web/dist/index.html` daté 2026-07-26 02:50, contient `assets/waypointUrl-BkDOfaJd.js` et `dist/icons/halowaypoint-*.png` |
| 11 | `appShellStore` n'est **pas** persisté | aucun middleware `persist` dans `apps/web/src/stores/appShellStore.ts` → `availableTitles` est refait à chaque chargement, pas de capabilities périmées en cache |

**Conséquence : le verrou 1 est nécessairement `true` en production pour Halo Infinite.**

---

## 3. Cause racine

### 3.1 Prouve : ce n'est PAS

- pas la capability (point 1, 2) ;
- pas l'absence de `title.toml` (voir §4, le fichier serait ignoré de toute façon) ;
- pas les assets (point 3) ;
- pas un bundle périmé, ni en prod ni en local (points 4, 10) ;
- pas un override `columnVisibility` de la page Explorer (point 8) ;
- pas un défaut de préférence à `false` (point 6) ;
- pas une régression de fusion `persist` (point 7) ;
- pas un double store zustand (point 9).

### 3.2 Suppose (non observable depuis ici)

Il ne reste que le **verrou 2**, côté navigateur de l'utilisateur :

```
localStorage['levelup-ui-prefs'] -> .state.localUiPrefs.showWaypointColumn === false
```

**Pourquoi « cache vidé » ne suffit pas** : le nettoyage du cache HTTP du navigateur ne touche
pas le `localStorage`. Une valeur `false` écrite une seule fois (bascule involontaire du toggle,
ou test) survit indéfiniment à tous les rechargements et à tous les déploiements.

Je n'ai pas pu observer ce point : aucune session authentifiée n'est disponible depuis cet
environnement (le bootstrap anonyme renvoie `auth_state: "missing"`, `available_players: []`),
et l'Explorer exige une session. **Je le signale comme hypothèse, pas comme fait.**

### 3.3 Hypothese alternative a ne pas ecarter

L'utilisateur regarde peut-être un tableau qui masque **légitimement** la colonne :
`apps/web/src/features/session-detail/SessionMatchesTable.tsx:47-49` — la variante `compact`
passe `COMPACT_HIDDEN_COLUMNS` qui contient `waypoint: false`. C'est la page **Session**, pas
l'Explorer, mais les deux rendent le même composant avec le même aspect.

---

## 4. Q1/Q2 — Pourquoi Infinite n'a pas de `title.toml`, et faut-il en creer un

### 4.1 C'est un CHOIX explicite, documente, pas un oubli

`apps/go-api/internal/domain/title/config_loader.go:1-11` (doc de package) :

> `halo_infinite` reste le descripteur BUILT-IN câblé en dur dans `NewRegistry()`
> (byte-identique, robuste même sans config) ; un `title.toml` halo_infinite est donc ignoré ici.

Historique :
- `f5b85eba0` — « registre piloté par config + provisioning DB au boot (MT-16 / day-one 2e titre) » : création de `config_loader.go`. Le mécanisme `title.toml` a été conçu **pour les titres additionnels**, pas pour rapatrier Infinite.
- `fc989ca5c` — « skeleton config/titles/halo_5 (coming_soon) » : premier et seul `title.toml` réel.
- `git log --all --diff-filter=A -- config/titles/halo_infinite/title.toml` : **aucun résultat**. Le fichier n'a jamais existé, à aucun moment de l'histoire du dépôt. Ce n'est donc pas un vestige.

### 4.2 D'ou viennent reellement les capabilities d'Infinite

Chaîne complète, à citer :

1. `registry.go:290-316` — `NewRegistry()` enregistre le descripteur Infinite en dur, avec sa
   liste `Capabilities` (dont `CapWaypointMatchURL` en `:307`).
2. `config_loader.go:233-237` — `NewRegistryFromConfig()` = `NewRegistry()` **puis**
   `LoadTitlesIntoRegistry()`.
3. `bootstrap_service.go:575-604` — `buildAvailableTitlesFrom()` projette
   `TitleDescriptor.Capabilities` en `[]string` dans `available_titles[].capabilities`.
4. Front : `apps/web/src/lib/capabilities/capabilities.ts` — `useCapability()` lit cette liste.

### 4.3 Le point critique : fusion, remplacement, ou ignore ?

**Ni fusion ni remplacement : IGNORE.** Le chargeur teste l'existence *avant* de lire le
manifeste — `config_loader.go:196-206` :

```
if reg.Exists(slug) {
    // F11 : un title.toml pour un titre built-in est SILENCIEUSEMENT ignoré.
    if _, statErr := os.Stat(filepath.Join(titlesDir, slug, "title.toml")); statErr == nil {
        logger.Warn("title_builtin_toml_ignored", ...)
    }
    continue
}
```

`halo_infinite` est déjà enregistré par `NewRegistry()` au moment de la découverte, donc
`reg.Exists("halo_infinite")` est vrai et le `continue` s'exécute **sans jamais parser le
fichier**.

**Verdict pour la question « quel est le risque de lui en créer un » :**

- Risque de **casser** des capabilities / des pages entières : **NUL**. Le fichier ne peut pas
  amputer le descripteur, même incomplet, même invalide — il n'est pas lu.
- Bénéfice : **NUL** également. Le fichier serait inerte.
- Risque réel, non technique : **piège de maintenance**. Un fichier présent mais mort fera
  croire au prochain intervenant qu'il peut y éditer les capabilities d'Infinite, et ses
  modifications seront silencieusement sans effet (seul un `WARN title_builtin_toml_ignored`
  dans les logs le trahira). C'est l'anti-pattern « dead code museum » + « doc inversée » du
  CLAUDE.md.

**Recommandation : NE PAS créer `config/titles/halo_infinite/title.toml`.**

### 4.4 Piege de lecture a connaitre : il y a DEUX systemes de capabilities

C'est la source de la confusion initiale.

| | Title-level | Data-level |
|---|---|---|
| Constantes | `internal/domain/title/registry.go` (`Cap*`) | `internal/games/adapter.go` (`CapabilityKey`) |
| Clés | plates : `waypoint_match_url`, `team_mmr` | pointées : `match.history`, `match.objective.stats` |
| Source Infinite | **Go en dur** (`registry.go:290-316`) | `config/titles/halo_infinite/mappings/capabilities.toml` |
| Source Halo 5 | `config/titles/halo_5/title.toml` | `config/titles/halo_5/mappings/capabilities.toml` |
| Rôle | **gating d'AFFICHAGE** (bootstrap → `useCapability`) | chemins de **données** serveur |

`waypoint_match_url` est **title-level**. Il n'a rien à faire dans un `capabilities.toml`.
Note : `config/titles/halo_infinite/mappings/capabilities.toml` **existe bel et bien** (contrairement
à ce que laissait entendre le cadrage) — c'est le fichier data-level.

---

## 5. Q3 — Correctif minimal et sur

**Il n'y a aucun correctif de capability à appliquer.** Rien n'est cassé côté Go / TOML / config.

Le correctif se décompose en deux volets **indépendants de la question 2** :

### Volet A — Debloquer l'utilisateur (aucune modification de code)

Dans la console du navigateur de l'utilisateur, sur `https://lvelup.info` :

```js
// 1. Diagnostiquer
JSON.parse(localStorage.getItem('levelup-ui-prefs')).state.localUiPrefs.showWaypointColumn
// attendu: true — si false ou undefined, cause confirmee
```

Si la valeur est `false` : basculer le toggle deux fois dans
Réglages → Apparence & Accessibilité → « Colonne Halo Waypoint sur les listes de matchs »,
ou en dernier recours :

```js
localStorage.removeItem('levelup-ui-prefs'); location.reload()
```

Attention : cette suppression réinitialise **aussi** thème, palette de couleurs, couleurs
d'équipe et dernier joueur sélectionné par titre (`settingsDraftStore.ts:86-94`).

### Volet B — Corriger la doc inversee (le vrai defaut de code)

La capability a été activée pour Halo 5 le 2026-07-24 (`c5dfb9bfd` — « colonne Halo Waypoint
activee pour Halo 5 (I19 correction utilisateur) ») **sans mettre à jour les commentaires**.
Cinq emplacements affirment aujourd'hui le contraire du code, et c'est précisément ce qui a
provoqué ce faux diagnostic :

| Fichier:ligne | Texte périmé | Correction |
|---|---|---|
| `apps/go-api/internal/domain/title/registry.go:116-119` | « Halo 5 : non declaree (I19) - Waypoint ne sert pas de page de detail de match Halo 5 ; le chemin d'URL est de toute facon specifique a Infinite » | Les deux titres déclarent la capability depuis le 2026-07-24 ; chemins distincts résolus par `buildWaypointMatchUrl` |
| `apps/web/src/features/explorer/ExplorerMatchesTable.tsx:23-24` | « (absente pour Halo 5) » | supprimer la parenthèse |
| `apps/web/src/features/explorer/ExplorerMatchesTable.tsx:283-284` | « absente pour Halo 5, cf. registry.go » | idem |
| `apps/web/src/features/squad/SquadSynergyHistoryTable.tsx:10` et `:101` | « (absente pour Halo 5) » | idem |
| `apps/web/src/features/settings/_settingsCards.tsx:23-24` | « Halo 5 : colonne masquée quel que soit ce réglage » | faux : la colonne s'affiche aussi sur Halo 5 |

C'est un correctif de commentaires uniquement — aucun changement de comportement, aucun gate à
rejouer au-delà du lint.

---

## 6. Q4 — L'URL Waypoint est-elle correcte pour Infinite ?

`apps/web/src/lib/match-nav/waypointUrl.ts:21-30` :

```
function waypointTitleSegment(titleSlug) { return titleSlug === 'halo_5' ? 'halo-5-guardians' : 'halo-infinite' }
export function buildWaypointMatchUrl(playerSlug, matchId, titleSlug = 'halo_infinite') {
  const seg = waypointTitleSegment(titleSlug)
  const bucket = titleSlug === 'halo_5' ? 'arena/' : ''
  return `https://www.halowaypoint.com/${seg}/players/${encodeURIComponent(playerSlug)}/matches/${bucket}${matchId}`
}
```

Pour Infinite, la fonction produit donc :
`https://www.halowaypoint.com/halo-infinite/players/{gt}/matches/{id}` — **sans** bucket
(le `arena/` est réservé à Halo 5).

### Verification effectuee, et sa limite — a lire

J'ai testé avec un identifiant de match Infinite réel (`f6315f2a-e54b-4b89-8274-bc07869d7689`,
Chocoboflor, issu de `apps/go-api/tests/fixtures/golden_values/career_top_matches_chocoboflor.json`) :

| URL | Résultat |
|---|---|
| `/halo-infinite/players/Chocoboflor/matches/f6315f2a-...` | `200 text/html` **93719** o |
| `/halo-infinite/players/Chocoboflor/matches/00000000-0000-0000-0000-000000000000` (temoin bidon) | `200 text/html` **93719** o |
| `/halo-5-guardians/players/Chocoboflor/matches/arena/5d16ff8d-...` | `200 text/html` **76003** o |

**Je ne peux donc PAS prouver par `curl` que la page de match existe** : Halo Waypoint est une
SPA rendue côté client, elle renvoie exactement le même shell (au **octet près**) pour un
identifiant réel et pour un identifiant inventé. Le code HTTP n'est pas discriminant. Je le
dis franchement plutôt que de conclure à tort.

Éléments à décharge, eux, établis :
- les deux segments de titre sont des routes **réelles et distinctes** de Waypoint (shells de
  tailles différentes : 93719 vs 76003) — aucun des deux n'est un 404 déguisé ;
- le commentaire d'en-tête `waypointUrl.ts:6-8` marque les deux formats comme
  « vérifiés (2026-07-24, fournis/valides par l'utilisateur) ».

**Risque de lien mort introduit par un correctif : nul, puisqu'aucun correctif de capability
n'est nécessaire.** Le format Infinite est en production depuis `ef80b0e53` et n'a pas été
remis en cause.

---

## 7. Q5 — Valeur par defaut de `showWaypointColumn`

`apps/web/src/stores/settingsDraftStore.ts:93` → **`showWaypointColumn: true`**.

**Il n'y a donc pas de second défaut.** La règle 11 du CLAUDE.md (« pas de flag qui laisse une
feature OFF pour plus tard ») **n'est pas violée** : la feature est livrée active.

Impact confirmé :
- **Nouvel utilisateur** : `localStorage` vide → défaut `true` → colonne visible d'emblée.
- **Démo publique** : idem. De plus le `ToggleRow` de la carte Apparence est le **seul** de la
  carte à ne pas recevoir `disabled={frozen}` (`_settingsCards.tsx:69-76`), donc la préférence
  reste modifiable même en mode démo figé — cohérent avec son caractère purement local.
- **Utilisateur existant** : `settingsDraftStore.ts:182-198` fusionne clé par clé, donc un
  `localStorage` antérieur à l'introduction de la clé hérite bien du défaut `true`.

Le seul cas `false` est une bascule explicite du toggle — ce qui ramène à l'hypothèse §3.2.

---

## 8. Q6 — Conflit avec `1b18ae609` (branche `fix/quick-wins-post-v721`, non mergee)

État vérifié : `git merge-base --is-ancestor 1b18ae609 main` → **faux**. Le commit n'est que sur
`fix/quick-wins-post-v721`. `main` est à `cea5930f9`.

### 8.1 Volet deps `useMemo` — reel, mais hors sujet

`ExplorerMatchesTable.tsx` l.725→728 et `SquadSynergyHistoryTable.tsx` l.325-328 : ajout de
`currentTitleSlug` aux dépendances du `useMemo` des colonnes. C'est un **vrai** correctif : sans
lui, au changement de titre sans démontage du composant, le `href` construit par
`buildWaypointMatchUrl(..., currentTitleSlug)` reste figé sur l'ancien slug (donc un lien
`halo-infinite` pour un match Halo 5, ou l'inverse).

**Mais il ne corrige pas le symptôme rapporté** : la visibilité de la colonne ne dépend pas de
ce `useMemo`, elle dépend de `internalColumnVisibility` (`:780-783`) dont les deps sont déjà
correctes.

### 8.2 Volet `.dockerignore` — la justification du commit est contredite par les faits

Le message de commit affirme : « tous les PNG/WebP etaient strippes de l'image prod : logo
Waypoint des tableaux, logo.png, og-default.png, emblemes H5 ». **C'est faux**, et c'est
important pour ne pas croire que ce commit va « réparer » la colonne :

- `.dockerignore:60` contient `*.png`. En syntaxe d'ignore Docker, `*` **ne traverse pas** les
  séparateurs `/` : `*.png` n'exclut que les PNG à la **racine du contexte de build**.
  `apps/web/public/icons/halowaypoint-white.png` n'a jamais été concerné.
- Preuve empirique, **sans** ce commit déployé : prod sert `/icons/halowaypoint-white.png` en
  `200 image/png` **8910** o et `/logo.png` en **226841** o — les tailles exactes des fichiers
  du repo. Les assets sont donc bien dans l'image.
- Contrôle confirmant que le commit n'est effectivement pas déployé : `/icons/does-not-exist-xyz.png`
  renvoie encore `200 text/html` 3000 o (fallback SPA), et non le « 404 franc » que le commit
  introduit.

Le changement `.dockerignore` reste **inoffensif** (négations redondantes + garde-rail
`verify-public-in-dist.mjs`), mais il ne faut en attendre aucun effet sur ce symptôme.

### 8.3 Ou poser le correctif

- **Conflit textuel** : faible. Le volet B (§5) touche les en-têtes de commentaires
  (`ExplorerMatchesTable.tsx:23-24` / `:283-284`, `SquadSynergyHistoryTable.tsx:10` / `:101`,
  `_settingsCards.tsx:23-24`, `registry.go:116-119`) ; `1b18ae609` touche
  `ExplorerMatchesTable.tsx:728` et `SquadSynergyHistoryTable.tsx:325-328`. Zones **disjointes**.
- **Recommandation** : poser le correctif de doc **sur `fix/quick-wins-post-v721`**, pas sur une
  nouvelle branche depuis `main` — ce sont les mêmes fichiers, et deux branches concurrentes sur
  `ExplorerMatchesTable.tsx` créeraient une résolution manuelle inutile.
- **Jamais directement sur `main`** : push `main` = déploiement prod automatique (CLAUDE.md).

---

## 9. Q7 — Autres capabilities dans le meme cas ?

Diff **effectif** mesuré sur la production (source : `/api/v1/bootstrap`, pas le code) :

- **Infinite seul** : `firefight`, `forge`, `season_pass`, `world.leaderboard`, `weapon_kills`,
  `team_mmr`, `damage_taken`, `expected_stats`, `objective_stats`
- **Halo 5 seul** : `native_kill_mechanics`, `weapon_accuracy`, `spartan_customizer`
- **Communes** : `matchmaking`, `ranked`, `career`, `asset.images`, `achievements`,
  `engagement`, `lusr`, `media`, `waypoint_match_url`

**Chaque écart est justifié par écrit et vérifié :**

- `firefight`, `forge`, `world.leaderboard` : exclusions **explicitement documentées** dans
  `config/titles/halo_5/title.toml` (« Exclus pour l'instant : firefight (Warzone FF, modèle !=
  FF HINF), forge (UGC HINF-shaped), world.leaderboard (scrape Waypoint HINF) »).
- `season_pass` : `registry.go:37-42` — Halo 5 n'a pas de Battlepass, inventaire REQ non servi (sonde 404).
- `weapon_kills` / `weapon_accuracy` : symétrie inverse assumée et cohérente avec le data-level
  (`match.weapon.accuracy` = `not_exposed` pour Infinite, `supported` pour H5).
- `expected_stats`, `objective_stats` : `registry.go:105-111` et `:122-131`.
- `native_kill_mechanics`, `spartan_customizer` : Halo-5-only documenté (`registry.go:52-56`, `:98-103`).

**Cohérence des miroirs scalaires vérifiée** (c'est là qu'un écart silencieux se serait niché) :

| Titre | `provides_team_mmr` | cap `team_mmr` | `provides_damage_taken` | cap `damage_taken` |
|---|---|---|---|---|
| halo_infinite | `true` | présente | `true` | présente |
| halo_5 | `false` | absente | `false` | absente |

Les invariants « un titre déclare cette cap SSI `ProvidesTeamMMR(slug)==true` » (`registry.go:66-76`)
et son équivalent `ProvidesDamageTaken` (`registry.go:78-87`) sont **respectés pour les deux titres**.

**Conclusion Q7 : aucun autre écart suspect.** `waypoint_match_url` était le seul cas, et il a
déjà été corrigé le 2026-07-24 (`c5dfb9bfd`) — seule la documentation est restée en arrière.

### Risque structurel a signaler (decouverte annexe)

Les capabilities d'Infinite ne sont modifiables **qu'en Go** (`registry.go`, recompilation +
déploiement), celles de Halo 5 **en TOML** (`title.toml`, zéro recompilation). Toute capability
« supportée des deux côtés » exige donc deux gestes, dans deux langages, dans deux fichiers.
C'est exactement l'asymétrie qui a produit la doc inversée ici : le geste TOML a été fait, le
commentaire Go a été oublié. Il n'existe **aucun garde-rail** qui vérifie la cohérence entre la
liste `knownCapabilities` (`config_loader.go:31-53`), la liste built-in Infinite
(`registry.go:295-311`) et la liste front `TITLE_CAPABILITIES`
(`apps/web/src/lib/capabilities/capabilities.ts:15-36`) — les trois sont maintenues à la main.
Piste d'amélioration hors périmètre, à noter au backlog.

---

## 10. Reste a trancher par l'utilisateur

1. **Confirmer la valeur réelle de `showWaypointColumn` dans son navigateur** (§5, volet A). Sans
   cette lecture, la cause racine reste une hypothèse — c'est le seul point bloquant du diagnostic.
2. **Confirmer sur quelle page exactement** la colonne manque : Explorer (mode Matchs / mode
   Joueur) ou Session ? La variante compacte de la page Session masque la colonne **par
   conception** (`SessionMatchesTable.tsx:47-49`) — si c'est là, il n'y a aucun bug.
3. **Confirmer l'environnement** : prod `lvelup.info` ou dev local `:8000` ? Les deux ont été
   vérifiés sains, mais ils ont des `localStorage` **distincts** (origines différentes) : un
   réglage vu ON en local n'implique rien sur prod, et réciproquement. C'est l'explication la
   plus économique de « le réglage est activé mais la colonne est absente ».
4. **Valider le volet B** (correctif de commentaires) et son placement sur
   `fix/quick-wins-post-v721`.

---

## 11. Comment verifier que c'est repare

```bash
# 1. Capability servie par la prod, pour les deux titres (attendu : 2 occurrences)
curl -s https://lvelup.info/api/v1/bootstrap | grep -o waypoint_match_url | wc -l

# 2. Assets servis (attendu : 200 image/png, 8910 et 8907 octets)
curl -s -o /dev/null -w "%{http_code} %{content_type} %{size_download}\n" \
  https://lvelup.info/icons/halowaypoint-white.png
```

```js
// 3. Preference locale, dans la console du navigateur de l'utilisateur (attendu : true)
JSON.parse(localStorage.getItem('levelup-ui-prefs')).state.localUiPrefs.showWaypointColumn
```

4. **Visuellement** : Explorer → mode Matchs. La colonne est la **2e**, immédiatement après la
   flèche d'ouverture du match, **en-tête vide**, contenu = logo Waypoint 16x16 à `opacity-60`
   (blanc en thème sombre, noir en thème clair — `waypointUrl.ts:34-36`). Elle n'est **pas
   triable** (`ExplorerMatchesTable.tsx:761-763`). Un clic ouvre un nouvel onglet vers
   `halowaypoint.com/halo-infinite/players/{gt}/matches/{id}`.

5. **Non-régression Halo 5** : basculer le titre, la colonne doit **rester** visible et pointer
   vers `/halo-5-guardians/players/{gt}/matches/arena/{id}`.

6. Tests existants couvrant les deux verrous, à rejouer si le code est touché :
   `apps/web/src/features/explorer/ExplorerMatchesTable.test.tsx:337` et `:343`,
   `apps/web/src/features/squad/SquadSynergyHistoryTable.test.tsx:149` et `:155`,
   `apps/web/src/stores/settingsDraftStore.test.ts:19` (défaut `true`),
   `apps/web/src/lib/match-nav/waypointUrl.test.ts` (formats d'URL par titre).
