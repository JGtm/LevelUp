# Installation des services systemd

## thumbnails-watcher.service

Surveille le dossier vidéo et génère les miniatures GIF à la demande.

### Prérequis

- Utilisateur `deploy` sur le VPS
- Répertoire de déploiement : `/home/deploy/levelup`
- Venv Python : `/home/deploy/levelup/.venv/`

### Installation

```bash
# Vérifier / ajuster le WorkingDirectory et --videos-dir dans le fichier .service
# avant d'installer

sudo cp packaging/thumbnails-watcher.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable thumbnails-watcher
sudo systemctl start thumbnails-watcher
```

### Vérification

```bash
sudo systemctl status thumbnails-watcher
journalctl -u thumbnails-watcher -f
```

### Logs

```
/var/log/thumbnails-watcher.log
```
