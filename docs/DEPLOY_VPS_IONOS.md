# Déploiement LevelUp sur VPS Ionos

> Guide complet : provisionnement, domaine, HTTPS, authentification, déploiement Git et mises à jour.
> Rédigé le 2026-04-02.

## Architecture cible

```
Internet
   │  HTTPS :443
   ▼
[Nginx] ──reverse proxy──► [Docker: levelup :8501]
   │                              │
   │  Let's Encrypt               ├── data/ (volume persistant)
   │  (Certbot)                   ├── .env.local (secrets)
   │                              └── DuckDB files
[DNS Ionos]
levelup.mondomaine.fr → IP VPS
```

---

## Phase 1 — Provisionner le VPS Ionos

### 1.1 Choix VPS

- **Offre recommandée** : VPS M ou L (2 vCPU / 4 GB RAM minimum — DuckDB est gourmand)
- **OS** : Ubuntu 22.04 LTS (support Long Term, meilleure compatibilité Docker)
- **Région** : Europe (latence FR)

### 1.2 Sécurisation initiale (SSH)

```bash
# Depuis ton poste Windows (Git Bash / WSL)
ssh root@<IP_VPS>

# Créer un utilisateur de déploiement
adduser deploy
usermod -aG sudo deploy

# Copier ta clé SSH
ssh-copy-id deploy@<IP_VPS>

# Désactiver auth par mot de passe
sed -i 's/PasswordAuthentication yes/PasswordAuthentication no/' /etc/ssh/sshd_config
systemctl restart sshd

# Firewall minimal
ufw allow OpenSSH
ufw allow 80
ufw allow 443
ufw enable
```

### 1.3 Installer Docker + Docker Compose

```bash
# Sur le VPS (en tant que deploy)
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker deploy
# Se reconnecter pour prendre en compte le groupe

docker --version
docker compose version
```

---

## Phase 2 — DNS et domaine

### 2.1 Configuration DNS Ionos

Dans le **panneau Ionos → Domaines → Gestion DNS** :

| Type | Nom | Valeur | TTL |
|------|-----|--------|-----|
| `A` | `levelup` | `<IP_VPS>` | 300 |
| `A` | `@` | `<IP_VPS>` | 300 (si apex) |

Résultat : `levelup.mondomaine.fr` pointe vers le VPS.

> **Délai** : 5 min à 24h selon le TTL précédent. Vérifier avec `dig levelup.mondomaine.fr`.

---

## Phase 3 — Nginx + HTTPS (Let's Encrypt)

### 3.1 Installer Nginx et Certbot

```bash
sudo apt install -y nginx certbot python3-certbot-nginx
```

### 3.2 Config Nginx

Créer `/etc/nginx/sites-available/levelup` :

```nginx
server {
    listen 80;
    server_name levelup.mondomaine.fr;

    # Ouvert pour la validation Certbot
    location /.well-known/acme-challenge/ {
        root /var/www/html;
    }

    location / {
        return 301 https://$host$request_uri;
    }
}

server {
    listen 443 ssl;
    server_name levelup.mondomaine.fr;

    # Certbot complète ces lignes automatiquement
    # ssl_certificate ...
    # ssl_certificate_key ...

    # WebSocket obligatoire pour Streamlit
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;

    # Timeout élevé (queries DuckDB potentiellement longues)
    proxy_read_timeout 300s;

    location / {
        proxy_pass http://127.0.0.1:8501;

        # Protection accès public
        auth_basic "LevelUp - Accès restreint";
        auth_basic_user_file /etc/nginx/.htpasswd;
    }
}
```

```bash
sudo ln -s /etc/nginx/sites-available/levelup /etc/nginx/sites-enabled/
sudo nginx -t && sudo systemctl reload nginx
```

### 3.3 Générer le certificat SSL

```bash
sudo certbot --nginx -d levelup.mondomaine.fr
# Suivre les instructions : email, CGU, redirection HTTP→HTTPS
```

Certbot modifie automatiquement la config Nginx et configure le **renouvellement automatique** via un timer systemd.

```bash
# Vérifier le renouvellement auto
sudo certbot renew --dry-run
```

---

## Phase 4 — Authentification

Deux niveaux d'auth distincts :

| Niveau | Quoi | Solution |
|--------|------|----------|
| **Accès dashboard** | Qui peut voir le site | HTTP Basic Auth via Nginx |
| **API Halo (Xbox OAuth)** | Refresh tokens Azure AD | Déjà géré via `.env.local` + `src/auth/` |

### 4.1 HTTP Basic Auth (protection accès dashboard)

```bash
sudo apt install -y apache2-utils
sudo htpasswd -c /etc/nginx/.htpasswd tonlogin
# → saisir le mot de passe

sudo nginx -t && sudo systemctl reload nginx
```

Pour ajouter un utilisateur supplémentaire (sans `-c` pour ne pas écraser) :

```bash
sudo htpasswd /etc/nginx/.htpasswd autrelogin
```

> **Alternative** : `streamlit-authenticator` pour des sessions persistantes avec cookies et gestion multi-utilisateurs.

### 4.2 Persistance des refresh tokens Halo

Les tokens Azure AD sont dans `.env.local` sur le VPS. Ils survivent aux redémarrages via le champ `env_file:` de `docker-compose.yml`. Aucune modification nécessaire.

---

## Phase 5 — Déploiement initial

### 5.1 Structure sur le VPS

```bash
# Sur le VPS
mkdir -p /opt/levelup
cd /opt/levelup

# Cloner le repo
git clone https://github.com/toi/levelup.git .
# ou via SSH : git clone git@github.com:toi/levelup.git .

# Remplir les secrets
cp .env.local.example .env.local
nano .env.local
```

### 5.2 Initialiser les fichiers de config

```bash
# Si le script existe :
bash scripts/docker_init.sh

# Sinon manuellement :
cp app_settings.json.example app_settings.json
mkdir -p data/players data/warehouse data/logs
```

### 5.3 Migrer les données depuis Windows

```bash
# Depuis le poste Windows (Git Bash)
rsync -avz --progress \
  ./data/warehouse/ \
  deploy@<IP_VPS>:/opt/levelup/data/warehouse/

rsync -avz --progress \
  ./data/players/ \
  deploy@<IP_VPS>:/opt/levelup/data/players/
```

### 5.4 Premier lancement

```bash
cd /opt/levelup
docker compose up -d --build
docker compose logs -f levelup
```

---

## Phase 6 — Mises à jour (stratégie Git)

### Option A — Script de déploiement manuel (recommandé pour démarrer)

Créer `/opt/levelup/deploy.sh` :

```bash
#!/bin/bash
set -e
cd /opt/levelup
git pull origin main
docker compose up -d --build --no-deps levelup
docker image prune -f
echo "Déployé : $(git log -1 --oneline)"
```

```bash
chmod +x /opt/levelup/deploy.sh

# Déclencher depuis son poste :
ssh deploy@<IP_VPS> '/opt/levelup/deploy.sh'
```

### Option B — GitHub Actions (CI/CD automatique)

Créer `.github/workflows/deploy.yml` dans le repo :

```yaml
name: Deploy to VPS

on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - name: Deploy via SSH
        uses: appleboy/ssh-action@v1
        with:
          host: ${{ secrets.VPS_HOST }}
          username: deploy
          key: ${{ secrets.VPS_SSH_KEY }}
          script: /opt/levelup/deploy.sh
```

**Secrets GitHub à configurer** (`Settings → Secrets → Actions`) :

| Secret | Valeur |
|--------|--------|
| `VPS_HOST` | IP du VPS |
| `VPS_SSH_KEY` | Clé privée SSH (sans passphrase) |

### Option C — Watchtower (auto-update image Docker)

Si les images sont publiées sur Docker Hub / GHCR, ajouter dans `docker-compose.yml` :

```yaml
  watchtower:
    image: containrrr/watchtower
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    command: --interval 3600 levelup
    restart: unless-stopped
```

---

## Phase 7 — Monitoring minimal

```bash
# Crontab (crontab -e) — redémarrage auto si le conteneur ne répond plus
*/5 * * * * curl -sf http://localhost:8501/_stcore/health || \
  docker compose -f /opt/levelup/docker-compose.yml restart levelup
```

---

## Checklist déploiement initial

- [ ] VPS Ionos provisionné, accès SSH fonctionnel
- [ ] Utilisateur `deploy` créé, auth par mot de passe désactivée
- [ ] Firewall : ports 22, 80, 443 ouverts
- [ ] Docker + Docker Compose installés
- [ ] DNS `A record` configuré → IP VPS (vérifier avec `dig`)
- [ ] Nginx installé et config `/etc/nginx/sites-available/levelup` créée
- [ ] Certbot → certificat SSL généré, renouvellement auto vérifié
- [ ] `/etc/nginx/.htpasswd` créé
- [ ] `.env.local` rempli sur le VPS (tokens Halo, Discord webhook)
- [ ] `app_settings.json` présent
- [ ] Données migrées via `rsync` (si migration depuis local)
- [ ] `docker compose up -d --build` → app accessible sur `https://levelup.mondomaine.fr`

---

## Résumé des choix techniques

| Sujet | Choix | Raison |
|-------|-------|--------|
| Reverse proxy | Nginx | Léger, intégration Certbot native |
| HTTPS | Let's Encrypt (Certbot) | Gratuit, renouvellement automatique |
| Auth dashboard | HTTP Basic Auth (Nginx) | Zéro dépendance, suffisant pour usage privé |
| Déploiement | `deploy.sh` + GitHub Actions | Progressif, facile à déboguer |
| Persistance data | Volume bind mount `./data` | Déjà configuré dans `docker-compose.yml` |
| Auth Halo API | `.env.local` existant | Aucune modification nécessaire |
