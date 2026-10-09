# Lot C-ter — intégration des trois volets (branche `wt/cter-fusion`)

> Journal d'INTÉGRATION, pas de conception. Les trois volets ont leur propre journal
> (`LOTCTER_VOLET1.md`, `LOTCTER_VOLET2.md`, `LOTCTER_VOLET3.md`) ; celui-ci ne raconte que ce
> que la mise en commun a coûté : les fusions, ce qu'il a fallu trancher là où deux volets
> disaient la même chose autrement, la mesure des six témoins recuits, et les gates.
>
> Régime : BRANCHE UNIQUE `feat/v75`. Les lots sont des commits, la clôture est la CI verte au
> niveau JOB. Aucun merge de `wt/cter-fusion` ici, aucun push.

## 1. Ce qui a été fusionné, et dans quel ordre

Le volet 3 (jauge de capture en direct) est le SOCLE de la branche : `wt/cter-fusion` part de
son dernier commit, `80f6eb033`. Les autres volets et l'amont viennent par-dessus.

| # | Fusion | SHA | Ce qu'elle apporte |
|---|---|---|---|
| 0 | socle — volet 3 | `80f6eb033` | jauge en direct (`zoneStates[].gauge`), schéma d'artefact 18 |
| 1 | `origin/feat/v75` (= `84c2a2159`) | `fa6195299` | amont obligatoire avant d'empiler les volets |
| 2 | `wt/colline-propriete` — volet 1 | `058496b1d` | désignateur de colline KOTH (`ZoneMethodDesignator`) |
| 3 | `wt/colline-formes` — volet 2 | `fe6da07ef` | rôle `hill` au catalogue de formes + cliquet `testutil.RepoRoot()` |
| 4 | `origin/feat/v75` (= `c154f5136`) | `2711d26f9` | catalogue des socles (`mapWeaponPads`) + lâchés hors Fiesta |
| — | réparation CI (hors lot) | `784868147` | allowlists datées `cmd/variant-probe` + migration du cliquet |

L'amont a été refusionné parce qu'il avait avancé de douze commits pendant la cuisson des
témoins : deux lots d'une autre session, `d8d45d9b0` (catalogue des socles) et `e5c1f4914`
(équipements et power-ups lâchés hors Fiesta).

## 2. Ce qu'il a fallu trancher

### 2.1 Fusion du volet 1 (`058496b1d`) — l'échelle du jeu gagne

Un seul fichier en conflit git, `zone_states_hill.go`. La STRUCTURE du volet 1 est conservée
(`buildHillStates` aiguille vers `hillDesignatorOf`/`buildDesignatedHills`, repli
`buildRampHills`), mais le calcul de `Progress` des périodes reprend **l'échelle du JEU du volet
3** (`gaugeProgressOf`) au lieu de l'ancienne machinerie par excursion de match
(`scales[...].progressOf`, `zoneGaugeScales`) que le volet 3 avait supprimée partout ailleurs.
Deux mesures sont publiées sur la même échelle ou elles ne sont pas comparables — c'est tout
l'enjeu, et c'est pourquoi la résolution ne pouvait pas être « garder les deux ».

Deux incohérences de fusion TEXTUELLE, hors du hunk marqué par git, ont dû être corrigées sur
pièces après compilation : `hillPeriod.gaugeSlot` devenait un champ mort (écrit, jamais lu) une
fois le lookup par slot retiré — supprimé avec ses deux sites d'écriture ; et
`buildDesignatedHills` appelait `tallyZoneSpans`, un nom qui n'existe que dans la branche du
volet 1, au lieu de `tallyZoneStates` retenu par le volet 3. Un auto-merge propre au sens de git
n'est pas un auto-merge juste : ces deux-là compilaient mal ou dupliquaient, et rien ne les
signalait dans les marqueurs.

### 2.2 Pas de série de jauge en colline

Décision du volet 1, tenue par la fusion et VÉRIFIÉE sur les artefacts (§4) : une colline de
KOTH ne publie **aucune** série `gauge`. La jauge de capture décrit des zones SIMULTANÉES ; en
colline, l'objet qui monte n'est pas la propriété d'une zone parmi d'autres mais le compteur de
transfert de l'unique colline courante. Publier une série là serait publier une grandeur
homonyme. Garde-rail : `TestZoneStatesCollineNePublieAucuneJauge`.

### 2.3 Fusion du volet 2 (`fe6da07ef`) — l'en-tête de la jointure rafraîchi

La fusion du volet 1 avait laissé une dette EXPLICITE : `document_zones.go` portait encore la
phrase « en KOTH cette table ne sert AUCUN rôle (le catalogue de formes ne connaît aucun rôle de
colline) [...] le client n'a, aujourd'hui, aucune zone où les poser ». C'était vrai jusqu'au
volet 2, qui apporte précisément le rôle `hill` au catalogue. Le bloc a été réécrit en
« LA JOINTURE QUI COMPTE POUR LE RENDU » : Bastion sur `strongholds_zone`, KOTH sur `hill`, avec
mention de l'état d'avant pour que la ligne se lise comme une histoire et non comme une
correction muette. La godoc de `ZonesCoverage.Roles` suit (`strongholds_zone`, ou `hill` en
KOTH). Un commentaire qui décrit l'ancien défaut est un anti-patron maison (« doc inversée ») :
il se met à jour dans le commit qui bascule le comportement, pas plus tard.

Même famille, même fusion : `service/replay_map_objectives.go` citait KOTH en exemple de « mode
sans objectifs statiques ». Devenu faux (constat R1-5 de la revue) — l'exemple est remplacé par
Slayer / Land Grab / Total Control, et la ligne dit désormais que KOTH sert le rôle `hill`.
`loader_objective_roles.go` admet `mapvar.RoleHill`.

> Report de traçabilité : le handoff mentionne une résolution « ligne 160 » que je n'ai PAS pu
> rattacher à une ligne sur pièces — ni dans `document_zones.go`, ni dans `zone_states_hill.go`
> (non touché par la fusion du volet 2), ni dans les journaux de volets, ni dans les logs de
> gates. Les quatre résolutions ci-dessus sont, elles, vérifiées sur le diff. Si « ligne 160 »
> désigne autre chose, c'est à confirmer par la session qui a joué les fusions 2 et 3.

### 2.4 Re-fusion de l'amont (`2711d26f9`) — les deux côtés, toujours

Deux conflits git, tous deux résolus en GARDANT LES DEUX CÔTÉS.

`.ai/thought_log.md` : deux blocs d'entrées ajoutés en tête par les deux branches. Aucun ne
remplace l'autre ; l'ordre chronologique les sépare.

`contracttest/replay_contract_test.go` : la chronique du contrat. Notre entrée « 36 -> 36 » (la
jauge — deux champs publiés, `zoneStates[].gauge` et `coverage.zones.gaugePoints`, mais AUCUN à
la racine du document) et la leur « 36 -> 37 » (`mapWeaponPads`, un vrai champ racine servi à la
requête) coexistent. Le compte passe à **37**, et la formule de clôture à « Les douze fois » :
une entrée sans champ racine n'incrémente pas ce compteur, c'est la convention du fichier (la
base était à 12 entrées / « onze fois » / 36). Notre entrée gagne une PRÉCISION datée du
2026-08-20, parce que sa dernière phrase — « le compte reste donc 36 des DEUX côtés de la
fusion » — devenait fausse à côté de sa voisine immédiate.

Le reste a fusionné seul, y compris `ReplayCanvas.tsx` que notre volet 3 ne touche pas : il
reste à **808 lignes, pile au plafond** que leur lot a lui-même abaissé (812 -> 808,
`placementFamily.guard.test.ts:172`).

### 2.5 Un troisième rouge, né de la re-fusion et d'aucun lot

Voir §3 : le test neuf d'amont `map_weapon_pads_catalog_test.go` écrit l'échelle de remontée
« ../../../.. » à la main, ce que le cliquet neuf du volet 2
(`archlint/no_repo_root_walk_test.go`) interdit. Ni l'un ni l'autre n'était rouge seul.

## 3. Réparation CI (hors lot — statut « À ROUTER »)

La CI de `feat/v75` était rouge depuis trois push sur deux garde-rails `archlint`, pour des
causes qui n'appartiennent à aucun lot en cours : le binaire `cmd/variant-probe`, livré par une
AUTRE session (sonde de l'activation des socles, voie API refermée le 2026-08-19). Précédent
R1-9 : une CI rouge en permanence cesse d'être lue, et la première vraie régression passe
inaperçue. Deux entrées d'allowlist DATÉES du 2026-08-20, chacune avec sa justification ET sa
condition de reprise écrites dans le fichier de règle :

| Garde-rail | Nature | Entrée | Condition de reprise |
|---|---|---|---|
| `no_halowaypoint_literal_test.go` | violation RÉELLE | `cmd/variant-probe/fetch.go` | router les deux URLs par `internal/sync/haloclient/` OU supprimer l'outil |
| `no_raw_kill_scope_literal_test.go` | FAUX POSITIF | `cmd/variant-probe/main.go` | suppression de l'outil OU renommage du drapeau CLI |

Le faux positif mérite son explication : `main.go` ne touche ni `match_kill_events`, ni
`read_path`, ni la préséance entre producteurs. Son unique occurrence est le NOM d'un drapeau,
`flag.String("scan", ...)`, qui déclare le mode hors ligne de la sonde. Le mot `scan` est entré
au ratchet le 2026-08-03 comme voie de film ; la collision est purement textuelle. Le ratchet
n'a PAS été affaibli : ni le motif, ni les propriétaires, ni la marche du walk ne bougent — seul
ce fichier-là sort du périmètre. Et l'allowlist, jusqu'ici vide et documentée comme devant le
rester, gagne son **self-check** (`TestKillScopeAllowlistEntriesStayJustified`, même leçon V4d /
VF-6 que le côté halowaypoint) : une entrée dont le fichier disparaît, ou dont le littéral est
retiré, rougit. L'exception ne peut pas survivre à sa cause.

**Statut : `[!]` À ROUTER** vers la session `variant-probe` (ou vers le superviseur). Ce lot ne
corrige pas `cmd/variant-probe/*` — ce n'est pas son périmètre, et la session propriétaire est
déjà close. Ce qui est fait ici rend la CI LISIBLE, pas la dette réglée.

Le troisième rouge, lui, est bien de notre fait puisque né de la re-fusion :
`map_weapon_pads_catalog_test.go` (côté amont) est migré vers `testutil.RepoRoot()` comme le
cliquet le PRESCRIT, et son `t.Skip` tombe — `data/titles/halo_infinite/reference/map_weapon_pads.json`
EST versionné (`git ls-files` le confirme), donc son absence est une anomalie d'arbre et non un
cas normal. Le skip muet sur fichier versionné est exactement le défaut R1-1 que la garde existe
pour tuer. Trois appelants mis à jour (`map_weapon_pads_catalog_test.go` ×2,
`socles_temoins_test.go` ×1).

**Preuve** : `go test -count=1 ./internal/archlint/` VERT. **Contrôle négatif** : les deux
entrées retirées, `TestNoNewHalowaypointLiteral` rougit sur `cmd/variant-probe/fetch.go:27` et
`:86`, `TestNoRawKillScopeLiteral` sur `cmd/variant-probe/main.go:60` — et sur rien d'autre.

## 4. Les six témoins recuits — vérification structurée

Six artefacts recuits dans ce worktree (`data/cache/replays/halo_infinite/`), lus au schéma 18 et
vérifiés champ par champ. `coverage.zones` est COMPLET sur les six : `method`, `roles`,
`catalog`, `paired`, `spans`, `hillPeriods`, `gaugePoints`, `ownerChecked`, `ownerAgreed` —
aucun champ manquant.

| Témoin | Carte / mode | Méthode | Rôles | Zones | Périodes | Points de jauge | Propriétaire | Taille |
|---|---|---|---|---|---|---|---|---|
| `7344d24f` | Vagabond — Strongholds:Arena | `captures+geometry` | `strongholds_zone` | 3 (catalogue 3) | 39 intervalles, 0 active | **1 701** | 46/46 = **100 %** | 2,24 Mo |
| `696a9d7c` | Vagabond — Strongholds:Arena | `captures+geometry` | `strongholds_zone` | 3 (catalogue 3) | 37 intervalles, 0 active | **1 794** | 51/51 = **100 %** | 2,11 Mo |
| `01e1f945` | Catalyst — KOTH:Arena | `designator+geometry` | `hill` | 4 (catalogue 6) | **5 actives** | 0 | non appliqué | 1,82 Mo |
| `0a247154` | Solitude — Ranked:KOTH | `designator+geometry` | `hill` | 5 (catalogue 5) | **6 actives** | 0 | non appliqué | 2,97 Mo |
| `606d9844` | Chasm — KOTH:Arena | `designator+geometry` | `hill` | 3 (catalogue 5) | **3 actives** | 0 | non appliqué | 0,72 Mo |
| `8076f97f` | Shogun — KOTH:Arena | `designator+geometry` | `hill` | 3 (catalogue 5) | **3 actives** | 0 | non appliqué | 1,17 Mo |

Les attendus tombent tous, et plusieurs contrôles supplémentaires les serrent :

- **Bastion (×2)** : méthode `captures+geometry`, jauge nourrie, propriétaire d'accord à 100 %
  (`ownerUnpaired` = 0 des deux côtés ; `696a9d7c` publie un intervalle à propriétaire inconnu,
  publié COMME TEL et non deviné). Échelle de la jauge : `v` dans [0 ; 0,999] sur les deux, donc
  bien celle du jeu et non une valeur brute. Les rampes sont TOUTES fermées par leur retour à
  zéro — 49/49 sur `7344d24f`, 51/51 sur `696a9d7c` —, et les instants restent dans l'axe de
  frames (39..5573 pour 5 689 images ; 83..5173 pour 5 337).
- **Colline (×4)** : méthode `designator+geometry` sur les quatre, **aucune zone ne porte même le
  champ `gauge`**, et le chevauchement maximal des périodes actives vaut **1** — une seule
  colline à la fois, mesuré et non supposé.
- **`0a247154`, le cas sans jauge tag 3** : il n'a AUCUNE série de jauge à offrir, et publie donc
  exactement ce que le désignateur lui donne — six périodes actives, contiguës et sans trou
  (327-1917, 1918-2424, 2425-3753, 3754-5020, 5021-6258, 6259-7837 sur 7 838 images), sans champ
  `progress`. Un `progress` inventé à zéro y aurait été un chiffre faux ; son absence est la
  réponse juste. Les trois autres collines portent bien leur `progress` (0,967 à 0,983 : le
  sommet du compteur de transfert, cf. volet 1).
- Le propriétaire n'est pas attribué en colline (`ownerChecked` = 0, `owner` nul sur tous les
  intervalles) : la voie tag 4 est celle de Bastion, le désignateur ne dit pas qui tient.
- `7344d24f` et `696a9d7c` gardent leurs 2 slots non appariés (19 slots parlants, 3 appariés) —
  état déjà relevé au volet 3 : deux slots tag 3 qui ne sont pas des jauges de zone et restent à
  nommer. Rien de neuf, rien de régressé.

Coût machine de la cuisson : `lotCter/cout_machine.tsv` (1,4 s à 394 s par film, 31 à 36 Go).

> **[2026-09-02] La colonne « pic » de ce `.tsv` ne mesure pas le décodeur — à ne plus citer
> comme mesure mémoire.** `lotCter/run_replay_build.ps1` lance `go run ./cmd/replay-build` par
> `Start-Process -FilePath "go"` (l. 40) et échantillonne `$p.PeakWorkingSet64` (l. 48) : `$p`
> est le processus **`go` LANCEUR**, qui compile puis exécute le binaire dans un processus
> ENFANT — celui qui décode, et dont le jeu de travail n'entre jamais dans cette mesure. Les
> durées, elles, restent lisibles (bout en bout du `go run`, compilation comprise). Constat
> **C6** de `.ai/AUDIT_CUISSON_REPLAY_PERF_2026-09-02.md` ; même lecture pour les pics de
> `LOTCTER_VOLET3.md` (§6 et « recuits en schéma 18 »).

## 5. Gates

Section « APRÈS RE-FUSION D'ORIGIN » de `LOTCTER_fusion_gates.log`. Tous les `EXIT_*` à 0 :

| Gate | Verdict |
|---|---|
| `go test ./internal/archlint/` (CGO=1) | 0 — **vert désormais**, les 2 rouges de la section précédente sont traités |
| `go test ./internal/analysis/replay/` (CGO=0) | 0 |
| `go test ./contracttest/ ./internal/replaybuild/ ./internal/service/` (CGO=1) | 0 |
| `go build ./...` (CGO=1) | 0 |
| `golangci-lint --new-from-merge-base=origin/main` | 0 issue |
| web `tsc -b --force` (après purge `.tmp`) | 0 |
| web `vitest src/features/match-replay src/lib` | 154 fichiers / **1 901 tests** verts |
| web `npm run lint` | 20/20 avertissements, plafond tenu |
| `lint-cross-feature-imports.mjs` | 7 <= plafond 7 |

Le vitest gagne +4 fichiers et +52 tests par rapport à la section précédente (150/1849) : ce sont
les tests des deux lots d'amont. Aucune flakiness cette fois, sur la commande EXACTE prescrite.

Une seule anomalie d'exécution, sans rapport avec le code : `golangci-lint` a refusé de démarrer
(« parallel golangci-lint is running », verrou d'une AUTRE session, PID 29708) ; rejoué après
libération.

## 6. Ce qui reste

- `[!]` **À router** : `cmd/variant-probe` (deux allowlists datées du 2026-08-20, conditions de
  reprise écrites dans les fichiers de règle). Vers la session `variant-probe` ou le superviseur.
- `[!]` **À confirmer** : la résolution « ligne 160 » du handoff, non rattachable sur pièces (§2.3).
- Aucun push, aucun merge vers `feat/v75` : régime branche unique, la clôture est la CI verte au
  niveau JOB.
