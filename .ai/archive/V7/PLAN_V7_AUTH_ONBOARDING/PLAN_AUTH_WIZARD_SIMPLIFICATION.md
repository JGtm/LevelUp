# Plan détaillé — Simplification Auth / Onboarding

## TL;DR

Le constat actuel est simple :

- le **launcher** couvre déjà l'installation et le démarrage de l'application
- le **wizard in-app** duplique une partie du parcours de configuration
- le wizard n'a **jamais été réellement testé en conditions réelles**
- l'intégration du **client ID embarqué** rend une partie du parcours Azure historique obsolète
- le projet conserve encore **plusieurs modèles d'authentification** qui coexistent mal

### Décision recommandée

**Sortir le wizard in-app du parcours principal** et faire du **launcher + Device Code Flow + stockage en base** la seule voie standard pour un utilisateur final.

Le wizard in-app n'est plus traité comme un assistant de configuration complet.
Deux options raisonnables existent :

1. **Option cible recommandée** : supprimer le wizard in-app comme onboarding principal et le remplacer par un écran très léger de statut / récupération.
2. **Option intermédiaire** : conserver temporairement un wizard minimal, mais uniquement comme wrapper du Device Code Flow moderne, sans Azure manuel ni collage de refresh token.

Ce plan recommande l'**option 1**.

---

## Contexte

### Situation observée

Le projet a évolué par couches successives :

- un ancien monde centré sur **Azure manuel + refresh token**
- une transition vers **MSAL Device Code Flow**
- une intégration plus récente d'un **client ID intégré au produit**
- un modèle de stockage plus propre via **DuckDB / sync_meta** par joueur

Le résultat est fonctionnel, mais l'expérience globale est devenue difficile à raisonner :

- la doc annonce une expérience simplifiée
- l'UI et certains helpers exposent encore du legacy
- le wizard donne l'impression d'être une seconde porte d'entrée, alors que le launcher fait déjà le travail d'installation et de lancement
- l'utilisateur final voit encore des concepts techniques qui devraient rester internes

### Symptômes concrets

- **Redondance produit** : launcher + wizard couvrent tous deux le "premier lancement"
- **Surface technique trop visible** : client ID, refresh token, variantes Azure
- **Cohérence partielle** entre code, doc et parcours réel
- **Maintenance plus coûteuse** : plusieurs chemins doivent rester compatibles
- **Risque UX** : un chemin non testé peut casser au moment où il est justement censé sauver l'utilisateur

---

## Problème à résoudre

### Problème principal

Le produit ne possède pas aujourd'hui **une seule source de vérité claire** pour le parcours d'onboarding et de reconnexion.

### Question produit

> Quel est le parcours normal, assumé, supporté et documenté pour un utilisateur final qui installe LevelUp et connecte son compte Xbox ?

### Réponse cible

La réponse doit devenir :

> **Launcher** pour installer et démarrer, puis **Device Code Flow moderne** pour connecter le compte, avec **stockage automatique du token en base**, sans Azure manuel ni manipulation visible du refresh token.

---

## Décision d'architecture

## Recommandation

**Le wizard in-app n'est plus un composant d'onboarding primaire.**

Il doit être :

- soit **retiré** du flux principal
- soit **réduit** à une interface minimale de récupération / reconnexion

### Ce que l'on garde

- le **Device Code Flow** moderne
- le **client ID intégré**
- le **stockage du refresh token en base DuckDB**
- le **refresh silencieux** et la couche auth moderne
- le **smoke test** post-auth si sa valeur produit est confirmée
- un **fallback CLI headless** pour les cas serveur / debug / maintenance

### Ce que l'on sort du produit standard

- la saisie manuelle du **client ID** dans l'UI
- le parcours **Azure classique** dans le wizard
- le collage manuel du **refresh token** dans l'interface
- les variantes auth-code / redirect URI / client secret pour l'utilisateur final
- la notion de **"configuration OAuth"** exposée à l'utilisateur final

---

## Pourquoi cette direction est la bonne

### Bénéfices utilisateur

- moins d'étapes
- moins de vocabulaire technique
- moins d'endroits où se tromper
- un parcours plus proche de la promesse produit actuelle

### Bénéfices développement

- moins de branches de code à maintenir
- moins d'incohérences doc / code
- moins de tests à faire vivre sur des flows secondaires
- un modèle mental plus simple : **auth moderne unique** + fallback CLI explicite

### Bénéfices support

- diagnostic plus simple
- moins d'ambiguïté dans la FAQ
- messages d'erreur plus clairs
- meilleure reproductibilité des bugs

---

## Option retenue vs alternatives

| Option | Description | Avantages | Inconvénients | Verdict |
|---|---|---|---|---|
| A | Supprimer le wizard in-app comme onboarding principal, garder un écran de récupération minimal | Architecture claire, moins de dette, UX cohérente | Demande un peu de nettoyage transversal | **Recommandée** |
| B | Garder un wizard minimal purement Device Code | Transition douce, peu risqué | Conserve une redondance launcher/app | Acceptable en étape intermédiaire |
| C | Conserver wizard + Azure manuel + legacy refresh token | Aucun gros changement immédiat | Dette, confusion, maintenance coûteuse | À éviter |

---

## Cible produit

## Parcours utilisateur final cible

### Premier lancement

1. L'utilisateur lance **LevelUp.bat** ou le launcher équivalent.
2. Le launcher prépare l'environnement et démarre l'application.
3. L'application détecte l'absence de connexion Xbox pour le joueur courant.
4. Elle affiche un **écran simple** :
   - connecter mon compte Xbox
   - relancer la connexion si besoin
   - voir l'état actuel
5. L'utilisateur effectue le **Device Code Flow**.
6. Le produit :
   - résout le gamertag / XUID
   - crée ou met à jour le profil joueur
   - persiste le token dans la base du joueur
   - lance le smoke test si ce check est gardé

### Reconnexion / recovery

1. Si le refresh silencieux échoue, l'app affiche un message simple.
2. L'utilisateur relance le **Device Code Flow**.
3. Le token est remplacé automatiquement en base.

### Cas headless / maintenance

1. Le développeur ou l'admin utilise un script CLI dédié.
2. Le script ne propose que le **Device Code Flow moderne**.
3. Le résultat est écrit à l'endroit attendu, sans exposer de variantes obsolètes.

---

## Périmètre fonctionnel

## Dans le périmètre

- simplification du parcours d'onboarding
- simplification du parcours de reconnexion
- retrait du legacy Azure visible côté UI
- clarification des sources de vérité auth
- alignement de la documentation
- réduction des chemins supportés officiellement

## Hors périmètre

- refonte totale du launcher
- refonte complète de la couche auth interne si elle fonctionne déjà
- changement du format de stockage des tokens en base
- suppression immédiate de tout code legacy interne si une étape transitoire est nécessaire

---

## État actuel à nettoyer

## 1. Deux portes d'entrée produit

Le launcher et le wizard couvrent tous deux le setup initial.

**Effet** : redondance de responsabilité.

## 2. Deux vocabulaires auth

- vocabulaire moderne : Device Code, client intégré, stockage DB, silent refresh
- vocabulaire legacy : Azure client ID, refresh token global, auth-code, redirect URI

**Effet** : confusion pour l'utilisateur et pour le dev.

## 3. Plusieurs sources de vérité techniques

- environnement global `.env.local`
- refresh token global
- refresh token par joueur
- refresh token stocké en base joueur
- couche auth moderne basée MSAL
- couche sync legacy encore appuyée sur certaines variables d'environnement

**Effet** : on ne sait plus immédiatement ce qui est normatif et ce qui n'est qu'un fallback.

## 4. Documentation hétérogène

La documentation mélange parfois :

- parcours recommandé moderne
- alternatives historiques
- variables Azure encore présentées comme nécessaires

**Effet** : la doc raconte plusieurs époques du produit à la fois.

---

## Stratégie de simplification

Le bon mouvement n'est pas une suppression brutale aveugle.

Le bon mouvement est :

1. **définir la voie officielle**
2. **marquer le reste comme fallback interne ou legacy**
3. **sortir progressivement du produit standard ce qui n'est plus nécessaire**
4. **aligner code, doc et UI sur cette vérité unique**

---

## Plan d'exécution détaillé

## Phase 0 — Cadrage et validation produit

### Objectif

Valider officiellement que le wizard in-app n'est plus le parcours principal.

### Livrables

- décision documentée
- périmètre validé
- liste des flows supportés officiellement

### Actions

- confirmer que le **launcher** est la porte d'entrée recommandée
- confirmer que le **wizard complet** n'a pas de valeur produit suffisante pour justifier sa maintenance
- confirmer si le **smoke test** reste un composant fort du premier lancement
- confirmer si l'app doit encore pouvoir initialiser un joueur depuis l'UI, même sans wizard complet

### Résultat attendu

Une phrase produit unique doit suffire :

> L'onboarding standard se fait via launcher + connexion Xbox moderne. L'UI n'expose plus de configuration Azure manuelle.

---

## Phase 1 — Définir la cible UX minimale dans l'app

### Objectif

Remplacer le wizard in-app par une expérience beaucoup plus simple.

### Cible UX recommandée

L'app n'affiche plus un wizard multi-chemins.
Elle affiche un **écran d'état / action** très court.

### Contenu de cet écran

- état de la connexion Xbox
- joueur courant ou joueur à provisionner
- bouton **Se connecter avec Xbox**
- bouton **Réessayer la connexion** si la session est expirée
- message simple en cas d'échec
- éventuellement bouton **Mode avancé / headless** renvoyant vers la doc CLI

### Ce que cet écran ne doit plus faire

- demander un client ID
- expliquer Azure
- demander un refresh token
- proposer plusieurs méthodes concurrentes

### Bénéfice

On garde la récupération in-app utile, sans conserver un second onboarding complet.

---

## Phase 2 — Faire de la couche auth moderne la seule vérité produit

### Objectif

Faire reposer le parcours officiel uniquement sur :

- client ID embarqué
- MSAL Device Code Flow
- cache MSAL / tokens stockés en base joueur
- refresh silencieux au runtime

### Actions

- auditer la couche auth moderne et la déclarer **source de vérité**
- identifier tous les appels legacy qui supposent encore :
  - `SPNKR_AZURE_CLIENT_ID`
  - `SPNKR_AZURE_CLIENT_SECRET`
  - `SPNKR_AZURE_REDIRECT_URI`
  - `SPNKR_OAUTH_REFRESH_TOKEN` global comme mécanisme primaire
- documenter les endroits où ces éléments restent nécessaires uniquement en fallback technique

### Résultat attendu

Un dev qui lit le code doit comprendre rapidement :

- **normal path** : auth moderne
- **fallback path** : scripts/headless/compat legacy

---

## Phase 3 — Réduire l'UI au strict nécessaire

### Objectif

Supprimer la redondance fonctionnelle entre launcher et wizard.

### Actions recommandées

- retirer le choix de mode **Xbox / Azure**
- retirer le parcours **Azure classique** de l'interface
- retirer le formulaire de **client ID** dans l'UI standard
- retirer le collage manuel du **refresh token** dans l'UI standard
- conserver uniquement :
  - démarrage du Device Code Flow
  - attente / polling
  - completion du flow
  - création / mise à jour du profil
  - smoke test éventuel

### Résultat attendu

L'UI n'est plus un configurateur OAuth.
Elle devient un point d'action minimal autour de la connexion Xbox.

---

## Phase 4 — Clarifier les modèles de stockage des tokens

### Objectif

Réduire la confusion entre token global, token par joueur et token en base.

### Direction recommandée

- **stockage normal** : token lié au joueur, persisté en base du joueur
- **env global** : fallback de développement / maintenance / compatibilité
- **env par joueur** : fallback avancé, non exposé au grand public

### Actions

- documenter explicitement la hiérarchie réelle des tokens
- vérifier que l'UI et les services ne parlent plus du refresh token global comme d'un prérequis normal
- isoler le support legacy dans des helpers clairement nommés

### Résultat attendu

Le refresh token reste un **détail interne**, pas un concept de parcours utilisateur.

---

## Phase 5 — Simplifier le fallback CLI

### Objectif

Conserver un seul outil de recovery headless, moderne et compréhensible.

### Recommandation

Le script de récupération ne doit supporter officiellement que le **Device Code Flow moderne**.

### Actions

- déprécier le mode **auth-code**
- déprécier les usages reposant sur **client secret + redirect URI** pour l'utilisateur final
- garder si nécessaire un chemin technique legacy non documenté publiquement pendant une courte transition
- envisager un renommage ultérieur du script pour refléter son rôle réel

### Résultat attendu

Le fallback CLI devient :

> un outil de connexion Xbox headless moderne,
> pas un couteau suisse OAuth historique.

---

## Phase 6 — Aligner la documentation

### Objectif

Faire raconter la même histoire au produit, au code et à la doc.

### Documentation à revoir en priorité

- installation
- configuration
- FAQ auth
- guide sync si nécessaire
- changelog si le changement est visible utilisateur

### Principes de rédaction

- une seule voie recommandée
- les alternatives avancées sont reléguées dans une section dédiée
- le mot **Azure** ne doit plus apparaître dans le parcours standard utilisateur
- le mot **refresh token** ne doit plus apparaître comme une étape utilisateur normale

### Résultat attendu

Un utilisateur non technique ne doit plus se demander :

> Est-ce que je dois créer une app Azure ?
> Est-ce que je dois coller un token ?
> Est-ce que le wizard est obligatoire ?

---

## Phase 7 — Vérification fonctionnelle réelle

### Objectif

Tester enfin le parcours réellement supporté, au lieu de maintenir un flow théorique.

### Plan de validation

#### Scénario A — First run standard Windows

- machine propre
- launcher
- ouverture app
- connexion Xbox
- création profil
- persistance token
- smoke test
- arrivée dashboard

#### Scénario B — Reconnexion après expiration

- token invalide ou session expirée
- relance Device Code
- mise à jour token
- reprise normale

#### Scénario C — Headless / recovery CLI

- exécution script CLI moderne
- écriture au bon endroit
- l'app repart sans configuration manuelle supplémentaire

#### Scénario D — Multi-joueurs

- vérifier que la logique par joueur reste correcte
- vérifier les endpoints player-gated
- vérifier que le token du bon joueur est utilisé

### Résultat attendu

Le flow supporté n'est plus seulement plausible, il est **vérifié end-to-end**.

---

## Impacts techniques attendus

## Fichiers / zones probablement concernés

### UI / onboarding

- composants du wizard actuel
- logique du premier lancement
- écran de recovery / auth

### Auth moderne

- provider auth
- primitives MSAL
- orchestration Device Code

### Legacy / compatibilité

- helpers tokens historiques
- scripts CLI hérités
- messages d'erreur mentionnant encore Azure manuel

### Docs

- guides d'installation
- guides de configuration
- FAQ
- changelog

---

## Risques et mitigation

## Risque 1 — Casser un fallback dev encore utile

### Risque

Tu utilises encore toi-même le refresh token legacy par habitude et rapidité.

### Mitigation

- ne pas supprimer brutalement les helpers internes dès la première passe
- séparer clairement **support dev interne** et **parcours produit officiel**
- commencer par retirer la surface UI / doc avant de purger le code profond

## Risque 2 — Régression sur les endpoints player-gated

### Risque

Le modèle multi-joueurs dépend encore d'une bonne résolution du token par joueur.

### Mitigation

- garder le stockage par joueur comme modèle normal
- vérifier explicitement les flows de lecture depuis `sync_meta`
- tester les chemins profil / rank / customization

## Risque 3 — Réduction excessive de l'UI

### Risque

En retirant le wizard, on pourrait perdre un point de recovery utile.

### Mitigation

- ne pas supprimer toute UI auth
- la remplacer par un écran minimal mais robuste

## Risque 4 — La doc reste en retard sur le code

### Risque

Le changement est bon techniquement mais continue de paraître flou à cause de la doc.

### Mitigation

- traiter la doc comme une partie du chantier, pas comme une fin de sprint facultative

---

## Critères de succès

Le chantier est réussi si :

- un utilisateur final n'a plus besoin de comprendre Azure
- un utilisateur final ne voit plus le refresh token dans le parcours normal
- il n'existe plus **deux onboarding officiels** dans le produit
- l'app et la doc désignent la même voie standard
- le fallback CLI existe toujours pour les cas avancés
- les flows multi-joueurs continuent à fonctionner
- le parcours officiellement supporté a été réellement testé end-to-end

---

## Critères d'échec

Le chantier est raté si :

- le wizard reste présent mais continue de proposer plusieurs chemins contradictoires
- la doc parle encore d'Azure manuel comme d'un prérequis normal
- les erreurs runtime continuent à citer des variables legacy comme si elles étaient requises pour tous
- le fallback dev devient plus confus qu'avant

---

## Ordre recommandé des travaux

### Ordre le plus pragmatique

1. **Documenter et valider la décision produit**
2. **Réduire la surface UI**
3. **Aligner la doc utilisateur**
4. **Stabiliser le fallback CLI moderne**
5. **Nettoyer ensuite le legacy profond**

### Pourquoi cet ordre

Parce qu'il traite d'abord la confusion visible, sans forcer une purge profonde immédiate du code qui pourrait casser des usages dev utiles.

---

## Proposition de lotissement concret

## Lot 1 — Décision + UI

**But** : retirer la redondance la plus visible.

- supprimer le mode Azure dans l'UI
- retirer saisie client ID + collage token
- remplacer le wizard par un écran simple de connexion / recovery

## Lot 2 — Docs

**But** : raconter une seule histoire.

- installation
- configuration
- FAQ
- recovery

## Lot 3 — CLI recovery

**But** : garder un outil propre pour les cas non standards.

- ne garder que Device Code moderne comme voie officielle
- déprécier auth-code / secret / redirect pour l'utilisateur final

## Lot 4 — Nettoyage legacy profond

**But** : réduire la dette technique restante.

- messages d'erreur
- helpers tokens
- commentaires/docstrings obsolètes
- branches de compatibilité devenues inutiles

## Lot 5 — Validation réelle

**But** : fermer la boucle.

- first-run
- reconnexion
- headless
- multi-joueurs

---

## Recommandation finale

La bonne décision n'est pas de garder le wizard "au cas où".

La bonne décision est :

- **assumer que le launcher fait déjà le vrai onboarding système**
- **assumer que l'app ne doit plus embarquer un configurateur OAuth historique**
- **garder seulement une UI minimale utile pour connecter/reconnecter le compte Xbox**
- **laisser les flows techniques legacy comme outils internes transitoires**, puis les purger progressivement

En résumé :

> **launcher = setup**
> **app = connexion / recovery simple**
> **CLI = fallback headless**
> **refresh token = détail interne, pas concept produit**

---

## Prochaine étape recommandée

### Étape suivante la plus rentable

Réaliser un **refacto minimal du parcours UI** avant tout grand nettoyage interne :

- supprimer le mode Azure du wizard
- supprimer les formulaires manuels
- garder uniquement le Device Code Flow moderne
- transformer le wizard en écran simple de connexion / récupération

Cette étape donnera immédiatement une architecture plus lisible, sans t'obliger à migrer tout le legacy de développement en une seule fois.
