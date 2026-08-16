# PLAN — Sons du rejeu 2D : variation RANGED et distance, IN-APP

> Branche : `feat/sons-rejeu-inapp` (a creer depuis `feat/extraction-sons-armes`).
> Ouvert le 2026-08-16. Contrat d'execution : skill `plan-execution`.
> Execution CONFIEE A UN AGENT ; le pilote relit chaque gate.

## Decisions utilisateur (fermes — ne pas rouvrir)

1. Les sons extraits restent PURS : aucune variation cuite dans les `.wav`.
2. La variation RANGED (volume/hauteur par lecture) et la distance s'appliquent COTE APP.
3. Reglages sur la PAGE D'ADMIN uniquement — pas des reglages utilisateur.
4. SIMPLICITE de calibrage exigee : 3 boutons maximum, valeurs par defaut = valeurs du jeu.

## Ce qui existe deja (ne rien reecrire)

- `apps/go-api/cmd/weapon-sounds` : parseur de banks. Le paquet RANGED est deja LOCALISE et
  VALIDE (`proprietes.go`, `lirePaquetProps(d, suite, 2)`) — ses valeurs ne sont juste pas
  exportees. Fourchette = [val - min, val + max] par propriete (volume dB, pitch centiemes).
- Les courbes de fondu des `Blend` sont decodees (`conteneurs_autres.go`) — inutile pour ce
  plan (la distance in-app est un effet applicatif), mais disponible plus tard.
- Sons retenus par vote : `Desktop/Halo Infinite - Sons armes/` + votes JSON. La LIVRAISON
  des fichiers retenus attend la fin du re-vote (33 coups) — le plan prepare le chemin et
  le format, pas le contenu final.

## Reglages (page admin, section « Sons du rejeu »)

	variation   : curseur 0-100 % (defaut 100 = fourchettes du jeu telles quelles ; 0 = off)
	distance    : curseur 0-100 % (defaut 0 = son pur ; attenuation de gain + passe-bas)
	(pas d'autre bouton)

Stockage : meme mecanisme que les autres reglages d'app (`app_settings.json` / endpoint
admin existant — DECOUVRIR le pattern en place et s'y conformer, ne pas en inventer un).

## Etapes

### Etape 1 — Decouverte (rien coder avant)

Gate : compte rendu ecrit au journal de CE fichier.

- [x] Ou vit le rejeu 2D cote web (composants, lecteur audio existant ou absent)
- [x] Pattern exact des reglages admin existants (stockage, endpoint, page, i18n FR/EN)
- [x] Ou ranger sons + manifeste pour l'app (convention assets existante,
      `static/weapons-assets/...` ou autre — suivre l'existant)

### Etape 2 — Export des fourchettes RANGED (Go, cmd/weapon-sounds)

Gate : `go build ./...` + `go vet` verts ; nouvelle sortie JSON documentee dans l'en-tete
de main.go. NE PAS lancer le module de 7,24 Go : laisser la commande ecrite au journal,
le pilote l'executera (contrainte memoire du chantier sons).

- [x] Lire les valeurs du paquet RANGED (le lecteur existe, exporter min/max par propriete)
      — le lecteur n'existait PAS : `lirePaquetProps` jetait la seconde composante. Ecrit :
      `lirePaquetLarge` + `lireVariation` (`proprietes.go`).
- [x] Les faire remonter dans le rapport par arme : fourchette volume (dB) et hauteur
      (centiemes) par (mode, perspective) — agregation : fourchette de la couche dominante.
      MODE : fait (`modes[].variation` du mode `lot-tir`). PERSPECTIVE : non modelisable,
      voir journal — la granularite livree est l'EVENEMENT, qui est ce qui porte la
      distinction 1p/3p dans les faits.
- [x] Champ `variation` dans le manifeste destine a l'app (schema ecrit, valeurs a venir)
      — schema fige en tete de `variation.go` ; pas de struct Go (aucun producteur dans ce
      depot, ce serait du code mort), le type vivant est celui du lecteur web (etape 3).

### Etape 3 — Lecteur cote web (WebAudio)

Gate : `make check-types` + `make test-web` verts ; test unitaire du calcul de variation.

- [x] Module de lecture des sons d'armes du rejeu 2D (ou extension du lecteur existant) :
      par lecture, tirage uniforme dans [min, max] x (variation/100) applique en gain
      (GainNode) et hauteur (playbackRate = 2^(cents/1200)) — `weaponSoundLogic.ts` (pur)
      + `weaponSoundPlayer.ts` (assemblage WebAudio). Aucun lecteur existant a etendre :
      la feature n'avait aucun code audio (etape 1).
- [x] Distance : chaine GainNode + BiquadFilter passe-bas, mappee sur le curseur
      (0 % = neutre absolu — AUCUN noeud dans le chemin du signal a 0) — verifie par un
      test qui compte les noeuds crees, pas seulement leurs valeurs.
- [x] Fallback sans manifeste de variation : lecture pure (aucune erreur, aucun silence)

### Etape 4 — Reglages admin

Gate : page admin affiche la section, valeurs persistees et relues ; typecheck + lint verts.

- [x] Deux curseurs, strings FR/EN via i18n.ts (parite typee), tokens de couleur semantiques
- [x] Endpoint conforme au pattern decouvert a l'etape 1

### Etape 5 — Cloture

- [x] Journal de ce plan + thought_log + delivery-checklist
- [x] Commit(s) prefixes `feat(sons-rejeu):`, PAS de push sur main

### 2026-08-16 — Etape 5 : cloture, et l'echec local qui n'en est pas un

**delivery-checklist passee.** Completude : les 12 items des etapes 1 a 5 sont statues,
aucun `[!]`. Un seul report, ecrit et justifie (branchement dans `ReplayCanvas`, dependance
de la livraison des `.wav`). Aucun TODO/FIXME introduit. Aucun garde-rail affaibli, aucune
allowlist agrandie, aucun test skippe.

Gates de cloture, tous rejoues :

	go build ./...                      rc=0
	go vet ./...                        rc=0 (hors exclusions de build habituelles)
	go test ./...                       138 paquets ok — voir la reserve ci-dessous
	npm run typecheck (cache purge)     vert
	npm run lint                        0 erreur, 19 avertissements PREEXISTANTS
	                                    (`react-hooks/incompatible-library` sur TanStack Table)
	make test-web                       412 fichiers, 3640 tests, 0 echec

`-tags=integration` NON lance : le diff ne touche ni `persist/`, ni `sync/`, ni
`migration/` (verifie sur la liste des 29 fichiers du diff).

**RESERVE SUR `go test ./...` — a lire avant de s'inquieter.** Un paquet echoue :
`internal/himap`, sur `TestBalayageCoquille` (`balayage_coquille_gamefiles_test.go`), par
DEPASSEMENT DU TIMEOUT de 10 minutes. Ce test balaie les 27 cartes du jeu depuis les
fichiers installes ; il fait `t.Skip` quand `DeployRoot()` echoue, donc il ne s'execute que
sur une machine ou le jeu est present — celle du chantier. Aucun fichier de `himap` ni de
`himodule` n'est touche par les quatre commits (verifie par `git diff --name-only`), et le
chantier n'a rien change au rendu des cartes. Echec d'environnement, pas de regression.

**Autres controles de la checklist** : aucun `fmt.Println`/`log.Printf` ajoute dans
`internal/` ; aucune couleur en dur ni classe Tailwind couleur dans les fichiers neufs ;
strings UI en FR ET EN par typage ; aucune query key inline (la section admin reutilise
`queryKeys.settings`) ; `routeTree.gen.ts` non touche ; aucun `filepath.Join(..., "data")`
ajoute. Deux fichiers restent au-dessus de 500 lignes — `internal/api/handlers/settings.go`
(675, INCHANGE : la validation neuve est allee dans `settings_replay_sound.go`) et
`internal/platform/settings/store.go` (514 -> 539, +25 lignes de champs et de mappings dans
des fonctions existantes, non extractibles sans artifice). Dette signalee, pas aggravee la
ou elle pouvait etre evitee.

**CI DE LA BRANCHE — verifiee, pas supposee.** `feat/sons-rejeu-inapp` poussee sur origin
(sans PR). Trois workflows declenches :

	Secrets (gitleaks)   success
	Deploy Pre-Check     success
	CI                   7 jobs sur 8 verts, le 8e encore en cours a la remise

Detail du workflow CI : `Go Build + Test (windows-latest)`, `Go Build + Test
(ubuntu-latest)`, `Frontend (TypeScript + Vite build)`, `Go Lint (golangci-lint)`,
`Go Lease Enforcement (ADR 0013)`, `Go Contract Test (OpenAPI YAML)` et `OpenAPI Lint` sont
TOUS verts — le contrat OpenAPI confirme au passage qu'aucune regeneration n'etait
necessaire. `E2E React (Playwright)` est saute (0 s, condition de declenchement).
SEUL RESTANT : `Go Coverage + Baseline non-regression (CGO_ENABLED=1 — ./... complet)`,
toujours en cours apres 19 minutes — c'est le job long du depot, il rejoue toute la suite
Go avec CGO. A relire par le pilote :

	gh run view 31953129280

Aucun signal rouge a ce stade.

## Hors perimetre (ne pas toucher)

- Regeneration des sons, votes, artefact de tri (chantier sons-armes)
- RTPC de couche, delais d'action (statues au plan sons-armes)
- Tout reglage expose aux utilisateurs finaux

## Journal

### 2026-08-16 — Ouverture

Piste consignee au passage (chantier sons-armes, pas ici) : l'utilisateur se souvient d'un
« pan... clic » sur la Carabine Vestige — symptome possible du delai d'action non prouve.
A verifier a l'oreille sur les rendus regeneres avant d'instruire.

### 2026-08-16 — Etape 1 : compte rendu de decouverte

Branche `feat/sons-rejeu-inapp` creee depuis `feat/extraction-sons-armes` (86c2ed4c8).

**1. Le rejeu 2D cote web.** Feature unique et plate :
`apps/web/src/features/match-replay/` (33 fichiers). Route :
`apps/web/src/routes/{-$lang}/t/$titleSlug/players/$playerSlug/matches/$matchId/replay.tsx`.
Le coeur est `ReplayCanvas.tsx` (canvas 2D, ~500 L, deja a la limite des 500 L du
CLAUDE.md) : la boucle de lecture y vit directement, l'horloge est un `useRef`
(`frameRef.current`, fractionnaire, en FRAMES) et non un state — aucun re-render par
image. Etats React de lecture : `playing`, `multiplier` (0.5/1/2/4, analogue exact d'un
`playbackRate`), scrub par `<input type="range">` non controle + `restart()`.
Publication vers React a cadence reduite (`FRAME_PUBLISH_MS = 150`).

AUDIT AUDIO : **aucun WebAudio dans `apps/web`**. Grep exhaustif de `AudioContext`,
`new Audio(`, `.wav`, `playbackRate`, `GainNode`, `BiquadFilter`, `decodeAudioData`,
`howler` sur `src/`, `public/`, `e2e/` : zero resultat pertinent. Aucune dependance audio
dans `package.json`. Le seul « audio » existant est la feature `media`
(`CoverFlowModal.tsx`, `<video>` + hls.js, pistes audio HLS) — aucune brique reutilisable.
Le lecteur de sons du rejeu part donc de zero.

Evenements sonorisables : `Shot` (`ReplayShot`, `apps/web/src/lib/api/types.ts:2568`),
champ **`w`** = identifiant d'arme (hex 64 bits) — c'est le seul chemin arme -> categorie
deja resolu cote client, via `doc.weaponLabels[w].fx` puis `familyOf()`
(`shotEffects.ts`, 8 familles `ShotFamily`). Piege documente `replayDraw.ts:211-213` :
l'ancienne interface manuscrite nommait ce champ `weapon`. `KillEvent.weaponLabel`
(nom propre) n'est PAS relie a `weaponLabels`. Les tirs/grenades sont deja dans l'horloge
du film ; les kills subissent le decalage `t0Ms`.

Conventions de test de la feature : Vitest, tests colocalises, `xxx.test.ts` pour la
logique pure / `Xxx.test.tsx` pour les composants, logique pure extraite en
`camelCaseLogic.ts` (`replayLogic.ts`, `killFeedLogic.ts`, `rosterLogic.ts`,
`equippedLogic.ts`, `coverageLogic.ts`). Fixture obligatoire `test/testDoc.ts`
(`testReplayDoc()`), avec garde-rail `testDoc.guard.test.ts` qui interdit d'appeler
`normalizeReplayDocument(` a la main. Precedent transposable pour tester un moteur audio
sans navigateur : `canvasRecording.test.ts` (Proxy qui empile les appels).

**2. Pattern des reglages admin (a copier a l'identique).** Chaine complete verifiee :
- Stockage : `apps/go-api/internal/platform/settings/store.go` (514 L) — struct
  `AppSettings` (tags JSON snake_case), `defaultSettings()`, `applyAbsentDefaults()`
  (indispensable seulement pour un defaut non-zero quand la cle est absente), `Apply()`,
  `ToResponse()`. Persistance = fichier `app_settings.json` ecrit par
  `internal/platform/atomicfile.WriteFile` (garde-rail
  `internal/archlint/no_bare_settings_write_test.go`), jamais DuckDB.
- DTO : `apps/go-api/internal/domain/settings.go` — `SettingsResponse` (valeurs) et
  `UpdateSettingsRequest` (tout en pointeur `,omitempty`, semantique PATCH partiel).
- Endpoint : `apps/go-api/internal/api/handlers/settings.go` — `GET /settings` +
  `PATCH /settings` (Huma), montes sous `RequireAuth` + `RequireAdmin`
  (`server_apiv1.go:446-460`). Validations : `if req.X != nil` +
  `humacore.NewError(400, "invalid_xxx", "message FR")`.
- OpenAPI : la sortie est `Body any`, donc `SettingsResponse` ne vient PAS de Huma mais du
  fragment manuel `api/openapi_manual_fragment.yaml` — fragment volontairement INCOMPLET
  (plusieurs reglages existants n'y figurent pas). On ne l'etend donc pas : pas de
  `make openapi-gen` / `make generate-types` a declencher.
- Web : le type `SettingsResponse` est ECRIT A LA MAIN dans
  `apps/web/src/lib/api/types.ts:393` (pas un re-export de `generated.ts`). Hooks
  `useSettings` / `useUpdateSettings` (`features/settings/queries.ts`), query key
  `queryKeys.settings` (`lib/query/keys.ts:43`). Auto-save : chaque `handleChange`
  declenche immediatement le PATCH, pas de bouton Enregistrer.
- Page : modeles `features/admin/sync/AdminSyncSettingsSection.tsx` et
  `features/admin/system/AdminBackupSection.tsx` — section admin autonome qui cable son
  propre `useSettings/useUpdateSettings`, son i18n via `getSettingsText`, et un
  `<SectionHeader title=... />`. DECISION : la section « Sons du rejeu » va dans
  `AdminSystemPage` (onglet Systeme), qui heberge deja les reglages d'instance non-sync.
- i18n : `apps/web/src/features/settings/i18n.ts` — interface `SettingsText` + `FR_TEXT` +
  `EN_TEXT`, parite garantie par le typage (`Record<Locale, SettingsText>`). C'est ce
  fichier que consomment les sections admin de reglages, pas les manifests TOML.
- Curseur : **il n'existe aucun composant `Slider`** dans `components/ui`. Le pattern
  existant est un `<input type="range" className="w-full accent-primary">` —
  `features/notifications/NotificationsSettingsTab.tsx:199-212` est le modele exact.
- Tests : `internal/platform/settings/store_test.go` (quatuor Defaults/Apply/ToResponse/
  RoundTrip par champ), `internal/api/handlers/settings_test.go`,
  `features/settings/AnalyseTab.test.tsx`.

**3. Rangement des sons + manifeste.** Convention constatee :
`static/{folder}/{titleSlug}/...`, servi par un `http.FileServer` nu
(`server_apiv1.go:1305-1309`, pas de `go:embed`), URL composee par
`internal/assets/static/urls.go` cote Go et `apps/web/src/lib/staticAssets.ts` cote web.
Les manifestes existants s'appellent tous **`index.json`** (jamais `manifest.json`) et
vivent dans le dossier du titre : `static/weapons-assets/halo_infinite/jeu/index.json`
(le seul consomme, ecrit par `cmd/weapon-icons-build`, relu par `cmd/weapon-icons-table`
avec garde-rail de regeneration `TestTableGenereeEstAJour`), plus
`abilities-assets/.../index.json` et `grenades-assets/.../index.json`.
DECISION : les `.wav` et leur manifeste iront dans
**`static/weapons-assets/halo_infinite/sons/`** avec un **`index.json`** — miroir exact du
sous-dossier `jeu/` (« ce qui vient du jeu », convention deja documentee par
`weaponIconDir = "jeu/"` dans `games/halo_infinite/adapter_asset_urls.go`). Aucun code de
service a ecrire (le FileServer sert n'importe quelle extension). A ne pas oublier a la
livraison des fichiers : `.gitattributes` liste les binaires par extension et ne mentionne
pas `*.wav`.

**4. Etat reel de `lirePaquetProps` (verifie sur pieces, `proprietes.go:128-152`).**
La fonction accepte bien une `largeur`, mais elle ne rend que la PREMIERE composante par
propriete (`d[debutVals+i*4*largeur]`) : appelee avec largeur 2, elle lit le `min` et
IGNORE le `max`. Le seul appel RANGED (l.120) jette meme son resultat et sa branche `if`
est inerte (`out.Lu = out.Lu && true` est un no-op). L'etape 2 doit donc ajouter un
lecteur qui rend les DEUX composantes, pas seulement brancher l'existant.

### 2026-08-16 — Etape 2 CLOSE : la fourchette RANGED traverse tout le rapport

**Gate.** `gofmt -l ./cmd/weapon-sounds/` : vide. `go build ./...` : rc=0. `go vet ./...` :
rc=0. `go test ./cmd/weapon-sounds/` : ok (7 tests neufs). Aucun fichier > 500 L
(`lot_tir.go` revenu a 488 apres deplacement de deux helpers vers `variation.go`).
Le module de 7,24 Go n'a PAS ete ouvert — contrainte memoire respectee.

**Ce qui a ete ecrit.** `lirePaquetLarge` rend toutes les composantes d'un AkPropBundle et
valide chacune (l'ancien lecteur ne validait que la premiere) ; `lirePaquetProps` devient
son enveloppe pour la largeur 1, comportement inchange, verifie par test. `lireVariation`
decode le paquet RANGED en `fourchetteSon` (volume dB, hauteur en centiemes).
`bank.noterProps` enregistre volume propre ET fourchette du noeud en une seule ecriture —
les quatre copies du bloc `if pr.Lu && pr.VolumeDB != 0` sont ramenees a un appel.

**Propagation.** La fourchette suit exactement le chemin du gain, deja prouve a l'etape 18
du chantier sons : `etatChemin` porte les deux, la fourchette s'ADDITIONNE le long du
chemin (chaque noeud traverse tire le sien) et s'ENVELOPPE entre variantes d'un point de
choix (le moteur n'en joue qu'une). Elle ressort en `variation` a quatre niveaux : couche
(`branches[].variation`, mode `arbre`), evenement (`armes[].evenements[].variation`, mode
`lot`), mode de tir (`modes[].variation`) et arme (mode `lot-tir`). Toujours OPTIONNELLE :
absente, le son se joue pur.

**Agregation.** Couche dominante = plus fort gain de chemin (`variationDeCouches`), puis
mode dominant pour l'arme (`variationDominante`). Une couche de renfort 20 dB en arriere ne
dicte donc pas la variation du coup.

**LA PERSPECTIVE N'EST PAS EXPORTABLE, et c'est mesure, pas suppose.** Aucune structure du
pipeline Go ne porte 1p/3p : le seul « 1p/3p » du package est la liste de verbes candidats
de `noms.go`, qui sert au hachage FNV-1 des noms, et le seul autre endroit du chantier qui
en parle est un commentaire de `conteneurs.go`. La distinction vit dans les EVENEMENTS
(une arme a typiquement un evenement de tir par perspective) et dans les noms de fichiers
rendus hors depot (`_RAFALE_M<n>_3p.wav`). La fourchette est donc exportee A LA
GRANULARITE DE L'EVENEMENT, ce qui permet au manifeste de retenir celle de l'evenement 3e
personne — le rejeu 2D etant une vue exterieure (decision deja actee au handoff sons).

**QUESTION LAISSEE OUVERTE, A TRANCHER PAR LA PREMIERE EXECUTION REELLE.** Le format donne
deux composantes par propriete sans dire laquelle est le minimum. Deux lectures restent
possibles : des OFFSETS SIGNES autour du nominal (une negative, une positive) ou deux
MAGNITUDES positives a retrancher/ajouter. Rien n'est postule : les bornes sont rendues
ORDONNEES, et l'outil imprime en fin d'execution le releve des signes observes
(`variation RANGED : N couples lus | composantes negatives …, positives …, nulles …`).
Une majorite de couples (negatif, positif) confirme les offsets signes ; que des positives
imposerait l'autre lecture et un correctif d'une ligne dans `lireVariation`.

**COMMANDES POUR LE PILOTE — a lancer hors de cette session (memoire).**
Depuis `apps/go-api`, passe 1, module de 7,24 Go, une seule ouverture :

	go run ./cmd/weapon-sounds -mode lot -module pc/globals/globals-rtx-new.module \
	  -pck "<...>/Sound/win/SFX" -banks "8827aa7e,09089e7e" -json <...>/lot1.json

Le drapeau `-banks` n'est pas optionnel : sans lui, les deux banks entierement embarquees
(Mutilator `8827aa7e`, Carabine Vestige `09089e7e`) manquent — fausse alerte deja consignee
au plan d'extraction. Puis la passe 2, sur l'autre module (0,62 Go), jamais dans le meme
processus :

	go run ./cmd/weapon-sounds -mode lot-tir -module any/globals/globals-rtx-new.module \
	  -json <...>/lot1.json -out <...>/lot_tir.json

A LIRE DANS LA SORTIE : la ligne `variation RANGED : …` de la passe 1 (elle tranche la
question ci-dessus), et le champ `variation` dans `lot_tir.json`.

### 2026-08-16 — Etape 3 CLOSE : le lecteur, et un chemin de signal qui reste vide

**Gate.** `make check-types` : vert. `make test-web` : 411 fichiers, 3633 tests passes,
0 echec. `npx eslint` sur les quatre fichiers neufs : vert. Les 33 tests neufs passent.

PIEGE D'ENVIRONNEMENT, consigne pour la prochaine session : une PREMIERE execution de
`make test-web` a rendu 10 echecs sur 3647, tous des `Test timed out in 5000ms` sur des
garde-rails qui balaient l'arborescence (`fragClass.colorSource.guard`, `admin-ui.guard`,
etc.). Relancee machine au repos, la suite passe entierement. Ce sont des timeouts de
contention (aucun `testTimeout` n'est configure dans `vite.config.ts`, donc 5 s par
defaut), pas des regressions — aucun ne touche la feature du rejeu.

**Deux fichiers, une frontiere nette.** `weaponSoundLogic.ts` ne connait pas WebAudio :
tirage, conversions d'unites et mapping du curseur y sont des fonctions pures. Le module
suit la convention de la feature (`*Logic.ts` teste sans DOM, deja appliquee par
`replayLogic`, `killFeedLogic`, `rosterLogic`). `weaponSoundPlayer.ts` n'assemble que des
noeuds. C'est ce qui rend l'exigence du plan verifiable.

**LE POINT DUR EST TESTE, PAS SEULEMENT ECRIT.** « A 0 %, aucun noeud dans le chemin du
signal » ne se verifie pas a l'oreille : un GainNode a 1 de trop est inaudible jusqu'au
jour ou l'on compare le rendu de l'app au fichier extrait. Le test utilise donc un
AudioContext ENREGISTREUR — meme principe que `canvasRecording.test.ts`, qui teste le
rendu canvas sans navigateur — et assert sur la LISTE DES NOEUDS CREES : reglages par
defaut et son sans fourchette donnent exactement `['source']` et `['source->destination']`.
Les deux gains (variation et distance) sont additionnes en decibels avant conversion : un
seul GainNode, jamais deux en serie.

**Deux choix de mapping, expliques dans le code.** (1) Le reglage de variation reduit les
BORNES de la fourchette, pas le resultat du tirage : appliquer le ratio au resultat aurait
tire tout le son vers le grave a chaque reglage intermediaire. (2) Le gain de distance
decroit lineairement en decibels et la coupure du passe-bas GEOMETRIQUEMENT (20 kHz ->
500 Hz) : une octave est un rapport, pas une difference, et un mapping lineaire aurait
rendu la premiere moitie du curseur inaudible et la seconde brutale.

**CE QUI N'EST PAS FAIT, ET POURQUOI.** Le lecteur n'est PAS branche dans `ReplayCanvas` :
aucun `.wav` ni `index.json` n'existe encore, la livraison attendant la fin du re-vote
(33 coups) — c'est ecrit noir sur blanc en tete de ce plan (« le plan prepare le chemin et
le format, pas le contenu final »). Le brancher aujourd'hui ajouterait, a chaque ouverture
du rejeu, un chargement qui ne peut rien trouver. C'est une dependance explicite du plan,
donc un report VALIDE au sens de la regle 3 du contrat d'execution — et non un « je le
ferai plus tard ». Ce qu'il restera a faire le jour de la livraison : instancier le lecteur
au premier geste de l'utilisateur (un navigateur refuse un `AudioContext` avant), appeler
`play(arme)` depuis la boucle de `ReplayCanvas` sur les tirs (`Shot.w` est la seule cle
arme deja resolue cote client), et traiter les deux ruptures d'horloge — le scrub et le
`restart()` — pour ne pas rejouer en masse des tirs passes.

### 2026-08-16 — Etape 4 CLOSE : deux curseurs, et rien d'autre

**Gate.** `go build ./...` rc=0 ; `go vet ./...` rc=0 (seuls messages : contraintes de build
qui excluent des `cmd` sans rapport, comportement normal du depot) ; `go test` vert sur
`internal/platform/settings`, `internal/api/handlers`, `internal/domain`, `internal/api`
(contrat OpenAPI) et `cmd/weapon-sounds` ; `make check-types` vert ; `eslint` vert sur les
fichiers touches ; `make test-web` vert.

**Le pattern de l'etape 1 a ete suivi a la lettre**, sans rien inventer :
`store.go` (champs + `defaultSettings` + `applyAbsentDefaults` + `Apply` + `ToResponse`)
-> `domain/settings.go` (valeur dans la reponse, pointeur `,omitempty` dans la requete)
-> validation dans le handler -> `lib/api/types.ts` (type ECRIT A LA MAIN, pas genere)
-> `features/settings/i18n.ts` (interface + FR + EN, parite garantie par le typage)
-> section admin autonome facon `AdminSyncSettingsSection` / `AdminBackupSection`.
Cles : `replay_sound_variation_percent` (defaut 100) et `replay_sound_distance_percent`
(defaut 0). Section montee dans `AdminSystemPage` (onglet Systeme), decision prise et
consignee a l'etape 1.

**Le piege du defaut, traite explicitement.** La variation vaut 100 par defaut alors que le
zero-value d'un `int` est 0 — et 0 est un reglage LEGITIME (variation coupee). Sans
reapplication « cle absente -> 100 » dans `applyAbsentDefaults`, un `app_settings.json`
ecrit avant ce lot serait lu comme « variation coupee », soit l'inverse de l'intention.
Trois tests fixent les trois cas : cle absente, 0 explicite respecte, aller-retour
save/load.

**Deux choix d'interface, testes.** (1) Une valeur hors de [0, 100] est REFUSEE (400), pas
ramenee en silence : un curseur qui affiche 150 alors que le serveur a retenu 100 ment a
l'operateur. (2) Le curseur enregistre au RELACHEMENT (`mouseUp` / `touchEnd` / `keyUp`),
pas a chaque pixel : l'auto-save du depot enverrait sinon des dizaines de PATCH pour un
seul geste. L'affichage, lui, suit le doigt. Le test verifie qu'un deplacement en cours
n'appelle PAS la mutation.

**Aucun composant `Slider` n'existe** dans `components/ui` (verifie a l'etape 1) : les deux
curseurs sont des `<input type="range" className="w-full accent-primary">`, le pattern
existant de `NotificationsSettingsTab`. Aucune couleur en dur, uniquement des tokens.

**Regeneration OpenAPI : non requise, et c'est verifie.** La sortie du handler est
`Body any` et le schema `SettingsResponse` vient du fragment manuel, deja incomplet pour
plusieurs reglages existants. `internal/api` (dont `TestOpenAPIYAMLIsUpToDate`) passe sans
regenerer, et le garde-rail web des types generes aussi.

**Documentation** : les deux cles sont decrites dans `docs/CONFIGURATION.md` ET
`docs/FR/CONFIGURATION.md` (politique bilingue, regle 15), et ajoutees a
`app_settings.example.json`.

**PIEGE D'ENVIRONNEMENT, deuxieme occurrence.** La suite web complete a de nouveau rendu un
echec unique (`lab-removal.guard.test.ts`) au premier passage. Rejoue seul, ce test passe
en 917 ms, et un grep confirme zero occurrence interdite dans `src/`. Meme cause qu'a
l'etape 3 : `testTimeout` a 5 s, contention machine. Constat a garder pour la prochaine
session — un echec isole sur un garde-rail qui balaie l'arborescence se relit avant de se
corriger.
