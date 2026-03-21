# Guide de Configuration - LevelUp

> Configuration complète des tokens Azure, profils joueurs et paramètres de l'application.

## Table des Matières

- [Setup Wizard (recommandé)](#setup-wizard-recommandé)
- [Configuration Azure détaillée](#configuration-azure-détaillée)
- [Profils Joueurs](#profils-joueurs)
- [Variables d'Environnement](#variables-denvironnement)
- [Paramètres Application](#paramètres-application)

---

## Setup Wizard (recommandé)

**v6 — Zéro configuration pour les utilisateurs standard.** LevelUp intègre son propre client ID Azure.
Le wizard ne demande que deux choses :

1. **Votre gamertag** — saisi dans l'interface du wizard
2. **Authentification Device Code** — ouvrir `https://xbox.com/activate` et entrer le code affiché

Aucun compte Azure, aucune App Registration, aucun fichier `.env.local` requis pour une utilisation normale.

| | 🎮 Xbox Express (défaut v6) | ☁️ Azure Manuel (avancé) |
|-|------------------------------|--------------------------|
| App Registration Azure | **Non requise** (intégrée) | Requise (la vôtre) |
| Refresh token | **Auto** (Device Code) | Manuel (script) |
| gamertag + XUID | **Auto** (résolu via OAuth) | Manuel |
| Profil joueur dans `db_profiles.json` | **Auto** (créé par le wizard) | Manuel |
| Stockage du token | `stats.duckdb` (sync_meta) | `.env.local` |
| Étapes dans le wizard | **2** | **3** |

**Le wizard gère automatiquement :**
- Obtention et stockage du refresh token OAuth (dans `stats.duckdb/sync_meta`)
- Création du profil joueur dans `db_profiles.json`
- Smoke test de vérification sur 20 matchs

### Note pour les forks / développeurs

Le `LEVELUP_CLIENT_ID` intégré est lié à l'App Registration Azure de ce projet.
**Si vous forkez LevelUp**, créez votre propre App Registration Azure gratuite
(voir [§ Configuration Azure détaillée](#configuration-azure-détaillée) ci-dessous) et définissez :

```env
# .env.local
SPNKR_AZURE_CLIENT_ID=votre_propre_client_id
```

Cette variable est prioritaire sur le client ID intégré (`LEVELUP_CLIENT_ID` dans `src/auth/_constants.py`).
---

## Configuration Azure détaillée

### À propos de l'inscription Azure

> **Pourquoi Azure demande-t-il une carte bancaire ?**
>
> Azure demande une carte de crédit ou de débit lors de l'inscription principalement pour **vérifier l'identité de l'utilisateur et prévenir les fraudes**. Même si de nombreux services Azure proposent des crédits gratuits ou des niveaux gratuits, Microsoft a besoin d'un moyen de paiement valide pour :
> - Vérifier l'identité de l'utilisateur
> - Éviter les abus et les créations multiples de comptes gratuits
> - Activer la facturation automatique si l'utilisation dépasse les limites gratuites
>
> **Cela ne signifie pas que des frais seront prélevés immédiatement.** Des frais ne s'appliquent que si des services payants sont utilisés au-delà des limites gratuites.
>
> **Pour ce projet, vous ne dépasserez jamais le niveau gratuit.** LevelUp enregistre uniquement une application OAuth dans Azure Active Directory (Microsoft Entra ID), ce qui est entièrement gratuit et sans quota d’utilisation. Aucun service Azure payant n’est consommé.

> **Azure for Students — Aucune carte bancaire requise**
>
> Si vous disposez d'une adresse e-mail universitaire ou scolaire valide, vous pouvez vous inscrire à [Azure for Students](https://azure.microsoft.com/fr-fr/free/students/) gratuitement, sans carte bancaire.

### Prérequis

Pour utiliser l'API Halo Infinite via SPNKr, vous devez :

1. Avoir un compte Microsoft/Xbox
2. Créer une application dans Azure Portal
3. Obtenir un refresh token OAuth

### 1. Créer une Application Azure

1. Aller sur [Azure Portal](https://portal.azure.com/)
2. Naviguer vers **Microsoft Entra ID** → **App registrations**

   ![Microsoft Entra ID](../screenshots/azure-setup/01-entra-id.png)

3. Cliquer sur **New registration**

   ![Add App Registration](../screenshots/azure-setup/02-add-app-registration.png)

4. Configurer :
   - **Name** : `LevelUp Halo`
   - **Supported account types** : Personal Microsoft accounts only
   - **Redirect URI** : laisser vide
5. Cliquer sur **Register**

   ![Register Application](../screenshots/azure-setup/03-register-application.png)

6. Après l'enregistrement, Azure redirige vers la page **Overview** de l'application. **Copier l'Application (client) ID** — c'est votre `SPNKR_AZURE_CLIENT_ID`.

   ![Overview — Application (client) ID](../screenshots/azure-setup/03b-overview-client-id.png)

   > Il ressemble à : `xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx` (format GUID)

### 2. Activer le Device Code Flow (Client Public)

1. Dans votre application, aller à **Authentication**
2. Descendre jusqu'à **Advanced settings**
3. Mettre **Allow public client flows** à **Yes**
4. Cliquer sur **Save**

> C'est le seul paramètre nécessaire en plus de l'enregistrement. Pas de secret client, pas de redirect URI.

> **Récapitulatif — à ce stade vous n'avez besoin que d'une seule valeur :**
> - `SPNKR_AZURE_CLIENT_ID` → copié depuis la page **Overview** de l'app (étape 1.6)

### 3. Configurer le Fichier .env.local (méthode manuelle — avancé)

```bash
# Copier le template
cp .env.local.example .env.local
```

Éditer `.env.local` :

```env
# Azure Application (client ID uniquement — pas de secret requis)
SPNKR_AZURE_CLIENT_ID=xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx

# Token OAuth (à obtenir via le script ci-dessous)
SPNKR_OAUTH_REFRESH_TOKEN=
```

### 4. Obtenir le Refresh Token (méthode manuelle — avancé)

> **Si vous utilisez Xbox Express** dans le wizard, cette étape est inutile.
> Le token est obtenu automatiquement via le device code flow et stocké en base de données.

```bash
python scripts/spnkr_get_refresh_token.py --device-code
```

Ce script :
1. Affiche un code court (ex. `ABCD-1234`) et l'URL `https://microsoft.com/devicelogin`
2. Vous visitez l'URL et entrez le code dans votre navigateur
3. Après connexion, le refresh token est affiché et sauvegardé automatiquement dans `.env.local`

---

## Profils Joueurs

> **Si vous utilisez le Setup Wizard** : le profil est créé automatiquement.
> Cette section est utile pour ajouter des joueurs supplémentaires ou en mode CLI.

### Structure du Fichier db_profiles.json

```json
{
  "version": "2.1",
  "profiles": {
    "MonGamertag": {
      "xuid": "2533274823110022",
      "gamertag": "MonGamertag",
      "db_path": "data/players/MonGamertag/stats.duckdb",
      "is_default": true
    },
    "AutreJoueur": {
      "xuid": "2533274XXXXXXXXX",
      "gamertag": "AutreJoueur",
      "db_path": "data/players/AutreJoueur/stats.duckdb"
    }
  }
}
```

### Propriétés

| Propriété | Type | Description |
|-----------|------|-------------|
| `xuid` | string | Identifiant Xbox unique (16 chiffres) |
| `gamertag` | string | Nom du joueur |
| `db_path` | string | Chemin vers la base DuckDB |
| `is_default` | boolean | Joueur par défaut au lancement |

### Trouver son XUID

Plusieurs méthodes :

1. **Via le dashboard** : L'XUID s'affiche dans les paramètres
2. **Via l'API** : Lors de la première sync
3. **Via des sites tiers** : xboxgamertag.com, etc.

### Ajouter un Nouveau Joueur

**Méthode 1 — Automatique via CLI (recommandée) :**

```bash
# Par gamertag
python scripts/sync.py --add-player NouveauJoueur

# Par XUID
python scripts/sync.py --add-player 2533274823110022

# Ajout + sync complète en une seule commande
python scripts/sync.py --add-player NouveauJoueur --full --max-matches 500
```

Cette commande :
- Résout le gamertag/XUID via l'API
- Crée l'entrée dans `db_profiles.json`
- Crée le dossier `data/players/<gamertag>/`

**Méthode 2 — Manuelle :**

```bash
# Créer le dossier
mkdir -p data/players/NouveauJoueur

# Ajouter l'entrée dans db_profiles.json (voir structure ci-dessus)
# Puis synchroniser
python scripts/sync.py --player NouveauJoueur --full
```

---

## Variables d'Environnement

### Fichiers de Configuration

| Fichier | Usage | Git |
|---------|-------|-----|
| `.env.example` | Template avec valeurs par défaut | Versionné |
| `.env.local` | Configuration locale (tokens) | Ignoré |
| `.env` | Alternative à .env.local | Ignoré |

### Variables Disponibles

#### Azure / SPNKr

| Variable | Description | Requis |
|----------|-------------|--------|
| `SPNKR_AZURE_CLIENT_ID` | ID de l’application Azure (client public, sans secret) | Oui |
| `SPNKR_OAUTH_REFRESH_TOKEN` | Token de rafraîchissement global | Oui |
| `SPNKR_OAUTH_REFRESH_TOKEN_<GT>` | Token per-player (endpoints player-gated) | Non |

> **Tokens per-player** : certains endpoints Halo Waypoint (career rank, customisation)
> retournent 403 si le token Spartan n'appartient pas au joueur ciblé. Pour synchroniser
> ces données pour plusieurs joueurs, déclarez un token per-player dans `.env.local` :
>
> ```env
# Gamertag "SpartanC" → clé normalisée SPARTANC
SPNKR_OAUTH_REFRESH_TOKEN_SPARTANC=votre_refresh_token
> # Gamertag "Mon GT 2" → clé normalisée MON_GT_2
> SPNKR_OAUTH_REFRESH_TOKEN_MON_GT_2=autre_refresh_token
> ```
>
> Normalisation : `re.sub(r"[^A-Za-z0-9]", "_", gamertag.strip()).upper()`
>
> Sans token per-player, la sync du career rank est skippée (warning dans les logs)
> et l'adornment du hero banner ne s'affiche pas pour ce joueur.

#### Application

| Variable | Description | Défaut |
|----------|-------------|--------|
| `LEVELUP_DB` | Chemin vers la DB par défaut | Auto |
| `LEVELUP_DB_PATH` | Alias pour LEVELUP_DB | Auto |
| `LEVELUP_DB_READONLY` | Mode lecture seule | `0` |
| `SPNKR_PLAYER` | Joueur par défaut pour sync | Premier profil |

#### Debug

| Variable | Description | Défaut |
|----------|-------------|--------|
| `LEVELUP_DEBUG` | Mode debug global | `0` |
| `LEVELUP_DEBUG_ANTAGONISTS` | Debug calcul antagonistes | `0` |
| `STREAMLIT_DEBUG` | Debug Streamlit | `0` |

---

## Paramètres Application

### Fichier app_settings.json

```json
{
  "theme": "halo",
  "language": "fr",
  "default_player": "MonGamertag",
  "cache_ttl_seconds": 300,
  "max_matches_display": 100,
  "enable_debug_mode": false
}
```

### Paramètres Streamlit (.streamlit/config.toml)

```toml
[server]
port = 8501
headless = true

[theme]
primaryColor = "#00A2E8"
backgroundColor = "#0D1117"
secondaryBackgroundColor = "#161B22"
textColor = "#C9D1D9"
font = "sans serif"

[browser]
gatherUsageStats = false
```

---

## Sécurité

### Ne Jamais Versionner

Les fichiers suivants ne doivent **jamais** être committés :

- `.env.local`
- `.env`
- Tout fichier contenant des tokens
- `credentials.json`

Ils sont déjà dans `.gitignore`.

### Rotation des Tokens

Les refresh tokens Azure expirent après :
- 90 jours d'inactivité
- Ou selon la politique de l'organisation

Pour renouveler :

```bash
python scripts/spnkr_get_refresh_token.py
```

### Mode Production

En production (Docker, serveur) :

```env
LEVELUP_DB_READONLY=1
```

Cela empêche les modifications accidentelles de la base.

---

## Dépannage

### Token Expiré

```
Error: invalid_grant
```

**Solution** : Régénérer le refresh token :
```bash
python scripts/spnkr_get_refresh_token.py
```

### Client ID Invalide

```
Error: unauthorized_client
```

**Solution** : Vérifier le Client ID dans Azure Portal.

### Permission Refusée

```
Error: access_denied
```

**Solution** : Vérifier les permissions API dans Azure Portal.

### Base de Données Non Trouvée

```
Error: Database file not found
```

**Solution** : Vérifier le chemin dans `db_profiles.json` et créer le dossier :
```bash
mkdir -p data/players/MonGamertag
```
