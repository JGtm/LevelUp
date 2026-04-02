# Guide de déploiement — Étapes pas à pas

> Garde ce fichier ouvert dans VS Code pendant toute la procédure.

---

## ÉTAPE 1 — Connexion au VPS

Ouvre Git Bash sur ton PC Windows et tape :

```
ssh root@212.227.206.42
```

Entre le mot de passe root fourni par Ionos quand il te le demande.

---

## ÉTAPE 2 — Créer l'utilisateur deploy

Une fois connecté au VPS, tape ces 3 commandes **une par une** :

```
apt install -y sudo
```

```
adduser deploy
```
→ Il te demande un mot de passe : invente-en un et retiens-le.
→ Pour toutes les autres questions (nom, téléphone...) : appuie juste sur Entrée.

```
usermod -aG sudo deploy
```

---

## ÉTAPE 3 — Configurer la clé SSH (pour GitHub Actions)

Toujours sur le VPS, connecté en root, tape :

```
su - deploy
```
→ Tu passes sur le compte deploy.

```
mkdir -p ~/.ssh && chmod 700 ~/.ssh
```

```
ssh-keygen -t ed25519 -C "levelup-deploy" -f ~/.ssh/id_ed25519 -N ""
```
→ Génère une paire de clés sans passphrase.

```
cat ~/.ssh/id_ed25519.pub >> ~/.ssh/authorized_keys
```

```
chmod 600 ~/.ssh/authorized_keys
```

Puis affiche la clé PRIVÉE (tu en auras besoin pour GitHub) :

```
cat ~/.ssh/id_ed25519
```
→ Copie TOUT le texte affiché (de -----BEGIN jusqu'à -----END-----).
→ Garde-le quelque part, tu en auras besoin à l'étape 7.

---

## ÉTAPE 4 — Installer Docker et Nginx

Toujours sur le VPS (en tant que deploy), tape :

```
sudo apt update
```

```
curl -fsSL https://get.docker.com | sh
```

```
sudo usermod -aG docker deploy
```

```
sudo apt install -y nginx certbot python3-certbot-nginx apache2-utils
```

---

## ÉTAPE 5 — Cloner le projet sur le VPS

```
sudo mkdir -p /opt/levelup
```

```
sudo chown deploy:deploy /opt/levelup
```

```
cd /opt/levelup
```

```
git clone https://github.com/TON_USER/levelup.git .
```
→ Remplace TON_USER par ton nom d'utilisateur GitHub.

```
cp .env.local.example .env.local
```

```
nano .env.local
```
→ Remplis les tokens Halo (copie depuis ton .env.local Windows).
→ Quand c'est fait : Ctrl+X puis Y puis Entrée pour sauvegarder.

```
bash scripts/docker_init.sh
```

---

## ÉTAPE 6 — Configurer Nginx et HTTPS

### 6a — Créer le fichier de configuration Nginx

Tape cette commande pour ouvrir un éditeur :

```
sudo nano /etc/nginx/sites-available/levelup
```

L'éditeur s'ouvre (écran noir avec du texte en bas). Copie-colle TOUT le bloc ci-dessous dedans :

```
server {
    listen 80;
    server_name lvelup.info www.lvelup.info;
    location /.well-known/acme-challenge/ { root /var/www/html; }
    location / { return 301 https://$host$request_uri; }
}

server {
    listen 443;
    server_name lvelup.info www.lvelup.info;

    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_read_timeout 300s;

    auth_basic "LevelUp - Acces restreint";
    auth_basic_user_file /etc/nginx/.htpasswd;

    location / { proxy_pass http://127.0.0.1:8501; }
}
```

Pour sauvegarder et quitter nano : appuie sur Ctrl+X, puis Y, puis Entrée.

---

### 6b — Activer la config et supprimer la config par défaut

```
sudo ln -s /etc/nginx/sites-available/levelup /etc/nginx/sites-enabled/
```

```
sudo rm -f /etc/nginx/sites-enabled/default
```

---

### 6c — Vérifier et démarrer Nginx

```
sudo nginx -t
```
→ Doit afficher : "syntax is ok" et "test is successful".
→ Si c'est bon, continue. Sinon copie l'erreur et dis-la moi.

```
sudo systemctl reload nginx
```

---

### 6d — Générer le certificat HTTPS gratuit (Let's Encrypt)

```
sudo certbot --nginx -d lvelup.info -d www.lvelup.info
```

Certbot va te poser des questions :
1. Il te demande un email → tape le tien
2. Il te demande d'accepter les CGU → tape Y
3. Il te demande si tu veux partager ton email avec l'EFF → tape N
4. Il configure HTTPS tout seul ✅

---

### 6e — Créer le mot de passe du dashboard

```
sudo htpasswd -c /etc/nginx/.htpasswd tonlogin
```
→ Remplace "tonlogin" par le login que tu veux (ex: guillaume).
→ Il te demande un mot de passe : tape-en un et retiens-le. C'est ce que tu taperas pour accéder au site.

```
sudo systemctl reload nginx
```

---

## ÉTAPE 7 — Configurer GitHub Actions

1. Va sur GitHub → ton repo → **Settings** → **Secrets and variables** → **Actions**
2. Clique **New repository secret** et ajoute ces 2 secrets :

| Nom | Valeur |
|-----|--------|
| `VPS_HOST` | `212.227.206.42` |
| `VPS_SSH_KEY` | Le texte de la clé privée copié à l'étape 3 |

---

## ÉTAPE 8 — Copier les données depuis Windows

Ouvre un **nouveau** Git Bash sur ton PC Windows (pas sur le VPS) et tape :

```
rsync -avz --progress /c/Users/Gsit/Downloads/Scripts/LevelUp/data/warehouse/ deploy@212.227.206.42:/opt/levelup/data/warehouse/
```

```
rsync -avz --progress /c/Users/Gsit/Downloads/Scripts/LevelUp/data/players/ deploy@212.227.206.42:/opt/levelup/data/players/
```

---

## ÉTAPE 9 — Premier lancement

Retourne sur le VPS et tape :

```
cd /opt/levelup
```

```
newgrp docker
```

```
docker compose up -d --build
```
→ Ça prend 5-10 minutes la première fois.

Vérifier que ça tourne :

```
docker compose logs -f levelup
```
→ Ctrl+C pour arrêter d'afficher les logs.

Ouvre https://levelup.fr dans ton navigateur — le dashboard doit apparaître !

---

## ÉTAPE 10 — DNS Ionos (à faire avant ou pendant)

Dans le panneau Ionos → **Domaines** → **levelup.fr** → **Gestion DNS** :

Ajouter deux enregistrements de type **A** :

| Type | Nom | Valeur |
|------|-----|--------|
| A | @ | 212.227.206.42 |
| A | www | 212.227.206.42 |

→ Attendre 5 à 30 minutes avant que le domaine pointe vers le VPS.

---

## CORRECTIF NGINX — Config finale à coller sur le VPS

Tape sur le VPS :

```
sudo nano /etc/nginx/sites-available/levelup
```

Dans nano, efface tout avec Ctrl+K (maintenu) puis colle exactement ce bloc :

```
server {
    listen 80;
    server_name lvelup.info www.lvelup.info;
    location /.well-known/acme-challenge/ { root /var/www/html; }
    location / { return 301 https://$host$request_uri; }
}

server {
    listen 443 ssl;
    server_name lvelup.info www.lvelup.info;

    ssl_certificate /etc/letsencrypt/live/www.lvelup.info/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/www.lvelup.info/privkey.pem;
    include /etc/letsencrypt/options-ssl-nginx.conf;
    ssl_dhparam /etc/letsencrypt/ssl-dhparams.pem;

    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_read_timeout 300s;

    auth_basic "LevelUp - Acces restreint";
    auth_basic_user_file /etc/nginx/.htpasswd;

    location / { proxy_pass http://127.0.0.1:8501; }
}
```

Ctrl+X → Y → Entrée pour sauvegarder, puis :

```
sudo nginx -t
```

```
sudo systemctl reload nginx
```

---

## C'est terminé !

À partir de maintenant :
- Chaque `git push` sur `main` depuis ton PC déploie automatiquement sur `levelup.fr`
- Le certificat HTTPS se renouvelle tout seul
