# PLAN — Onboarding Device Code Flow cassé (SISU endpoint 404) + fix UX spinner infini

> Date de rédaction : 2026-07-11. Exécutant prévu : Opus. Superviseur : Guillaume.
> Origine : investigation du 2026-07-11 — wizard bloqué sur « Démarrage du Device Code Flow… »
> au premier lancement sur un PC neuf. Cause racine tracée sur pièces (cf. §1).
> Entrée thought_log associée : `[2026-07-11] Onboarding bloqué … SISU endpoint 404`.
>
> Objectif : rétablir un onboarding Device Code Flow fiable pour l'app distribuée, avec
> garde-rails empêchant la régression de repasser inaperçue, + corriger le bug UX qui masque
> l'échec derrière un spinner infini.
> Critère de succès global : onboarding device-flow fonctionnel vérifié bout-en-bout ;
> tout échec de `start` est surfacé côté UI ; un garde-rail détecte la panne d'endpoint ;
> chaque case du tracker (§4) porte un statut final `[x]`/`[~]`/`[!]`.

---

## 0. CONTRAT D'EXÉCUTION (rappel — le skill `plan-execution` fait foi)

1. **Ordre strict des lots** (§3). Ne pas commencer le lot N+1 tant que le lot N n'est pas
   CLOS. Exception : le lot B (décision produit) peut nécessiter un aller-retour utilisateur.
2. **Vérifier sur pièces avant de coder ET avant de cocher** — rouvrir chaque fichier/ligne
   cité ci-dessous ; le code a pu bouger, et surtout l'endpoint MS peut avoir re-changé.
3. **Statuts** : `[x]` fait+vérifié / `[~]` couvert ailleurs (référence) / `[!]` non traité
   (justification écrite obligatoire au Journal §5).
4. **Zéro fix opportuniste hors périmètre** : noter en §6 « Découvertes », ne pas traiter.
5. **Clôture d'un lot** : gate passé → cases statuées → ce fichier MAJ → entrée thought_log →
   point d'étape utilisateur.
6. **Git** : 1 branche dédiée (ex. `fix/auth-device-flow-sisu-404`), N commits ; jamais sur
   `main` ; demander avant tout commit/push (push main = deploy prod auto).
7. **Pas de nouveau flag laissant une feature OFF** (règle projet §11) — la correction se
   livre active. Le champ `auth_provider` existe déjà et n'est PAS un kill-switch de feature.

---

## 1. CONSTAT — cause racine (vérifiée sur pièces le 2026-07-11)

**Symptôme utilisateur** : premier `make restart` sur un PC neuf → le wizard reste bloqué
sur le spinner « Démarrage du Device Code Flow… », indéfiniment.

**Chaîne observée** :

1. `POST /api/v1/auth/device-flow/start` → **HTTP 500 `msal_init_error`**
   (logs `logs/general.log`, plusieurs occurrences 2026-07-11 20:31→20:39).
2. Provider actif au boot = **SISU** (défaut) — logs « buildTokenProvider: SISU provider
   activé (défaut) » ; sélection dans
   [cmd/server/main.go:87-111](../apps/go-api/cmd/server/main.go#L87-L111).
3. `SISUProvider.InitDeviceFlow`
   ([sisu_provider.go:112-168](../apps/go-api/internal/platform/auth/sisu_provider.go#L112-L168))
   enchaîne 3 appels réseau :
   - `requestDeviceToken` (`device.auth.xboxlive.com`) → **OK** (« Device Token obtenu »),
   - `StartXboxDeviceCode` → **ÉCHEC**,
   - `initSISUSession` (non atteint).
4. `StartXboxDeviceCode`
   ([xbox_device_code.go:49-116](../apps/go-api/internal/platform/auth/xbox_device_code.go#L49-L116))
   fait `POST https://login.live.com/oauth20_connect/device`
   (const `xboxDeviceCodeURL`, [xbox_device_code.go:25](../apps/go-api/internal/platform/auth/xbox_device_code.go#L25))
   → **HTTP 404, corps vide** (header `PPServer` présent = Passport atteint, chemin introuvable).

**Reproductions curl (indépendantes du binaire)** :
- `POST login.live.com/oauth20_connect/device` avec `scope=Xboxlive.signin Xboxlive.offline_access`
  → **404**.
- même endpoint avec `scope=service::user.auth.xboxlive.com::MBI_SSL` → **404**.
- Comparaison : `POST login.microsoftonline.com/consumers/oauth2/v2.0/devicecode` avec le
  client MSAL `e1cb35ab-…` → **200** (renvoie `user_code`/`device_code`/`verification_uri`).

**Conclusions** :
- **A** — L'endpoint natif Xbox `oauth20_connect/device` est retiré/déplacé côté Microsoft.
  La panne est **globale** (tout onboarding SISU distribué), PAS spécifique au PC de Guillaume.
- **B** — Le chemin MSAL (`consumers/oauth2/v2.0/devicecode`) fonctionne. Le contournement
  local `auth_provider=msal` (déjà posé dans `app_settings.json`, gitignored) débloque
  l'onboarding sur cette machine après restart.
- **C** — Bug UX indépendant : `useStartDeviceFlow` est appelé sans `onError` dans
  [StepDeviceCode.tsx:58-71](../apps/web/src/features/setup/StepDeviceCode.tsx#L58-L71) ; le
  garde `startFlow.isPending || (!status && !deviceFlowUserCode)`
  ([StepDeviceCode.tsx:119](../apps/web/src/features/setup/StepDeviceCode.tsx#L119)) laisse le
  spinner tourner à l'infini quand `start` échoue → l'erreur est avalée. Même schéma à
  vérifier dans `XboxLoginPage.tsx`.
- **D** — Angle mort de test : `xbox_device_code_test.go` et `sisu_provider_test.go` injectent
  des URLs `httptest` mockées → ils ne touchent JAMAIS l'endpoint réel, donc n'ont pas pu
  détecter le 404. Aucun smoke/health ne valide la joignabilité réelle du device endpoint.

---

## 2. DÉCISION PRODUIT REQUISE (à trancher avec Guillaume avant le lot C)

L'app est distribuée à des self-hosters qui ne peuvent pas tous enregistrer une app Azure —
c'est la raison d'être historique de SISU (cf. commentaire
[main.go:79-86](../apps/go-api/cmd/server/main.go#L79-L86)). Trois stratégies non exclusives :

- **Option 1 — Réparer SISU** : trouver le nouvel endpoint natif Xbox device-code (ou le
  nouveau flux) et mettre à jour `xboxDeviceCodeURL` + le polling associé. Préserve le « zéro
  app Azure ». Coût : recherche endpoint MS non documenté, risque de re-cassure.
- **Option 2 — Basculer le défaut sur MSAL** : `buildTokenProvider` renvoie MSAL par défaut.
  Fiable aujourd'hui, mais impose l'app Azure `e1cb35ab` à tous les self-hosters (couplage au
  client_id LevelUp) — à valider vs modèle de distribution.
- **Option 3 — Fallback automatique SISU→MSAL sur échec de start** : SISU reste le défaut ;
  si `InitDeviceFlow` échoue (404/erreur réseau), bascule transparente sur MSAL pour la
  tentative courante + log `legacy_source_used`-style. Robuste, mais garde le code SISU vivant.

> Recommandation exécutant : **Option 3** (robustesse immédiate + préserve SISU si MS le
> réactive) couplée à un ticket de recherche pour l'Option 1. À CONFIRMER par Guillaume —
> ne pas coder le lot C avant décision.

---

## 3. LOTS (ordre d'exécution)

### Lot A — Fix UX : surfacer l'échec de `start` (indépendant de la décision produit)
Priorité haute : corrige le « spinner infini » quelle que soit la stratégie retenue.
- Ajouter un `onError` (ou dériver `startFlow.isError`) dans
  [StepDeviceCode.tsx](../apps/web/src/features/setup/StepDeviceCode.tsx) : un échec de
  `useStartDeviceFlow` bascule sur l'UI d'erreur existante (§bloc `status==='failed'`) avec
  bouton « Réessayer » (`handleRetry` existe déjà), au lieu de rester sur le `Spinner`.
- Revoir la même condition de garde L119 pour ne pas masquer un état d'erreur.
- Vérifier/appliquer le même correctif dans
  [XboxLoginPage.tsx](../apps/web/src/features/auth/XboxLoginPage.tsx) si le schéma s'y répète.
- Tests : compléter `SetupPage.test.tsx` / `XboxLoginPage.test.tsx` avec un cas « start renvoie
  500 → message d'erreur + bouton Réessayer affichés » (garde-rail anti-régression du spinner).
- **Gate** : `make check-types` + `make test-web` verts.

### Lot B — Décision produit (§2)
- Présenter §2 à Guillaume, acter l'option (1/2/3). Consigner la décision au Journal §5.
- **Gate** : décision écrite + option retenue notée dans ce fichier.

### Lot C — Implémentation de la stratégie retenue (dépend du lot B)
Selon l'option :
- **Si Opt.1** : identifier le nouvel endpoint natif (curl exploratoire), mettre à jour
  `xboxDeviceCodeURL` + `pollXboxDeviceCode`, ré-tester bout-en-bout.
- **Si Opt.2** : `buildTokenProvider` défaut → MSAL ; MAJ commentaire main.go + doc onboarding ;
  vérifier que l'app Azure est documentée comme pré-requis self-host.
- **Si Opt.3** : wrapper qui tente SISU puis MSAL sur erreur d'`InitDeviceFlow`, avec log
  structuré du basculement. Pas de swallow d'erreur (logger la cause SISU avant fallback).
- Dans tous les cas : logger explicitement l'erreur d'`InitDeviceFlow` dans
  [auth.go:127-135](../apps/go-api/internal/api/handlers/auth.go#L127-L135) (aujourd'hui
  stockée dans l'attempt mais jamais loggée — swallowed error, anti-pattern CLAUDE.md §10).
- **Gate** : `cd apps/go-api && go test ./...` vert + onboarding device-flow réussi en réel
  (obtenir un `user_code`, aller au bout jusqu'à `authorized`).

### Lot D — Garde-rail anti-régression endpoint + doc
- Ajouter un check de joignabilité de l'endpoint device-code réel : soit un test taggé
  `integration` (opt-in réseau) qui asserte HTTP 2xx sur le start, soit une sonde de santé
  au boot qui logge un WARN si l'endpoint device-code répond 404/5xx (sans bloquer le boot).
  Objectif : la prochaine cassure MS est visible immédiatement, pas via un spinner utilisateur.
- MAJ doc onboarding (`docs/INSTALL.md` / `docs/FR/INSTALL.md`, `docs/CONFIGURATION.md`) :
  documenter `auth_provider` (`sisu` défaut vs `msal`) et le contournement.
- **Gate** : test/sonde en place et vert ; doc FR+EN à parité (règle §15).

---

## 4. TRACKER (aucune case vide à la clôture)

- [ ] A1 — `onError`/`isError` surfacé dans StepDeviceCode (plus de spinner infini)
- [ ] A2 — garde L119 revue pour ne pas masquer l'erreur
- [ ] A3 — même correctif vérifié/appliqué dans XboxLoginPage
- [ ] A4 — tests web « start 500 → erreur affichée »
- [ ] A5 — gate lot A (check-types + test-web) vert
- [ ] B1 — décision produit §2 actée par Guillaume
- [ ] C1 — implémentation option retenue
- [ ] C2 — log explicite de l'erreur InitDeviceFlow (fix swallowed error auth.go)
- [ ] C3 — onboarding device-flow réussi bout-en-bout en réel
- [ ] C4 — gate lot C (go test ./...) vert
- [ ] D1 — garde-rail joignabilité endpoint device-code
- [ ] D2 — doc onboarding + `auth_provider` MAJ (FR+EN)
- [ ] D3 — retrait/consignation du contournement local `app_settings.json` selon décision

---

## 5. JOURNAL

- **2026-07-11** — Investigation initiale. Cause racine tracée (SISU `oauth20_connect/device`
  → 404 ; MSAL `consumers/…/devicecode` → 200). Contournement local `auth_provider=msal`
  posé dans `app_settings.json` (gitignored) — actif au prochain `make restart`. Plan rédigé.
  Aucun code applicatif modifié à ce stade.

## 6. DÉCOUVERTES (hors périmètre — à ne pas traiter ici)

- (aucune pour l'instant)
