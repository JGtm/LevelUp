# Plan : Onboarding — Premier Lancement

> **Contexte** : Un utilisateur qui clone le repo depuis GitHub ne peut rien faire.  
> L'objectif est qu'un `make dev` (ou `.exe`) suivi d'un SSO Xbox suffise à tout initialiser.

---

## Constat : le wizard existe déjà

Le frontend a une implémentation complète et fonctionnelle :

| Élément | Fichier | État |
|---|---|---|
| Route `/setup` | `src/routes/setup.tsx` | ✅ existe |
| Guard `setup_required → /setup` | `src/routes/__root.tsx` ligne 70-72 | ✅ existe |
| Machine d'état 4 étapes | `src/features/setup/SetupPage.tsx` | ✅ existe |
| Étape 1 : Device Code Flow Xbox | `src/features/setup/StepDeviceCode.tsx` | ✅ existe |
| Étape 2 : création profil joueur | `src/features/setup/StepPlayer.tsx` | ✅ appelle `POST /setup/players` |
| Étape 3 : sync initial + progress | `src/features/setup/StepInitialSync.tsx` | ✅ appelle `POST /sync/initial` + polling job |
| Login Xbox (SSO redirect + Device Code) | `src/features/auth/XboxLoginPage.tsx` | ✅ existe |
| Landing post-SSO | `src/features/onboarding/OnboardingOpenSpartanPage.tsx` | ✅ existe |

**Le problème n'est donc pas le wizard — c'est le serveur qui crashe avant de répondre.**

---

## Vrai problème : crash au boot sans données

### Cause du crash

`data/warehouse/` n'existe pas → `duckdb.OpenReadWrite()` échoue → `os.Exit(1)`.

> Note : `app_settings.json` manquant ne crashe **pas** — `Load()` retourne `defaultSettings()`.

### Ce que fait déjà DuckDB si le dossier existe

`OpenReadWrite` sur un fichier inexistant **crée** la DB + les migrations appliquent le schéma complet.  
Donc : si `data/warehouse/` est créé, les fichiers `.duckdb` naissent tout seuls au premier boot.

---

## Ce qui manque réellement

### 1. `ensureWarehouseDirs()` au boot (Backend, ~1h)

Avant `runMigrations()` dans `main.go`, créer les dossiers requis :

```go
func ensureWarehouseDirs(pr *config.PathResolver, titleSlug string) error {
    dirs := []string{
        filepath.Dir(pr.SharedDBPath(titleSlug)),    // data/warehouse/
        filepath.Dir(pr.MetadataDBPath(titleSlug)),
        filepath.Dir(pr.SharedPVEDBPath(titleSlug)),
        pr.PlayerBaseDir(titleSlug),                 // data/titles/halo_infinite/players/
        filepath.Join(repoRoot, "data", "auth", "watcher_tokens"),
        filepath.Join(repoRoot, "data", "sessions"),
        filepath.Join(repoRoot, "data", "cache"),
    }
    for _, d := range dirs {
        if err := os.MkdirAll(d, 0o755); err != nil {
            return fmt.Errorf("ensureWarehouseDirs %s: %w", d, err)
        }
    }
    return nil
}
```

**Résultat** : `make dev` sans aucun fichier de données → serveur démarre → `/setup` s'affiche.

### 2. État `azure_configured` dans bootstrap (Backend, ~30min)

Le frontend ne sait pas si les credentials Azure sont manquants.  
Ajouter dans `BootstrapResponse` :

```go
AzureConfigured bool `json:"azure_configured"` // cfg.AzureClientID != ""
```

Le wizard peut alors afficher un écran préalable si `azure_configured === false` :

```
⚠️ Credentials Azure manquants
Créez .env.local à partir de .env.local.example
et renseignez SPNKR_AZURE_CLIENT_ID + SPNKR_AZURE_CLIENT_SECRET,
puis redémarrez le serveur.
```

Polling bootstrap toutes les 5s pour détecter le redémarrage.

### 3. État `halo_linked_no_profile` côté backend (à vérifier)

Le frontend a 4 états : `no_halo_link` → **`halo_linked_no_profile`** → `profile_ready_no_sync` → `ready`.  
Le backend (`bootstrap_service.go`) n'expose que 3 états. Vérifier si `halo_linked_no_profile` est bien retourné  
quand `linked_halo_identity != nil && len(players) == 0`.

---

## Flow cible après corrections

```
make dev
    │
    ├── ensureWarehouseDirs() → crée data/warehouse/ etc.
    ├── DuckDB crée les .duckdb + migrations (schéma vide)
    └── Serveur UP (setup_required=true, azure_configured=true/false)
           │
           ▼
    Navigateur → /setup (guard auto dans __root.tsx)
           │
           ├── [Si azure_configured=false] → guide .env.local (NEW)
           │
           ├── StepDeviceCode : Device Code Flow Xbox → linked_halo_identity obtenu
           │
           ├── StepPlayer : POST /setup/players (auto depuis linked_halo_identity)
           │       └── db_profiles.json ✓, stats.duckdb ✓
           │
           └── StepInitialSync : POST /sync/initial → progress bar
                   └── Dashboard opérationnel ✓
```

---

## Ordre d'exécution

1. **`ensureWarehouseDirs()`** — déblocant, ~1h, livrable seul — `fix/boot-mkdir-warehouse`
2. **`azure_configured` dans bootstrap** — embarquer dans le même commit
3. **Vérifier état `halo_linked_no_profile`** — rapide audit backend + fix si besoin
4. **Écran "azure manquant" frontend** — petit ajout dans `SetupPage.tsx`

**Pas de nouvelle page, pas de nouveau wizard.** Tout existe déjà.
