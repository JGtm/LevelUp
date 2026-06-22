# Sauvegardes VPS — Restic

Sauvegarde quotidienne des données critiques du VPS (`my-vps`, `/opt/levelup`)
contre la corruption, pour pouvoir revenir en arrière.

## Quoi / où / quand

| | |
|---|---|
| **Périmètre** | DuckDB du titre (`data/titles/halo_infinite/` : metadata, shared_matches_v2, shared_pve, shared_social, players) + tokens OAuth (`data/auth/`) + config (`db_profiles.json`, `app_settings.json`, `.env.local`) |
| **Exclus** | Médias (`data/media`, ~4,5 Go ; fichiers plats peu sujets à corruption), cache, logs |
| **Repo** | `/opt/levelup/restic-repo` (local au VPS — même disque, assumé) |
| **Password** | `/opt/levelup/.restic-password` (root, 600) — **à copier hors-VPS** (sinon repo irrécupérable si la machine est perdue) |
| **Planif** | timer systemd `levelup-restic-backup.timer`, tous les jours **04:00 UTC** |
| **Rétention** | 7 quotidiens + 4 hebdomadaires (`forget --keep-daily 7 --keep-weekly 4 --prune`) |
| **Cohérence** | arrêt bref du service `levelup` pendant le snapshot (fichiers DuckDB au repos) |
| **Log** | `/opt/levelup/data/logs/restic-backup.log` |

## Installation (one-shot, déjà fait le 2026-06-19)

```bash
apt-get install -y restic
umask 077; openssl rand -base64 24 > /opt/levelup/.restic-password; chmod 600 /opt/levelup/.restic-password
RESTIC_REPOSITORY=/opt/levelup/restic-repo RESTIC_PASSWORD_FILE=/opt/levelup/.restic-password restic init
mkdir -p /opt/levelup/scripts && cp scripts/restic-backup.sh /opt/levelup/scripts/ && chmod +x /opt/levelup/scripts/restic-backup.sh
cp scripts/systemd/levelup-restic-backup.* /etc/systemd/system/
systemctl daemon-reload && systemctl enable --now levelup-restic-backup.timer
```

> Le `git reset --hard origin/main` du déploiement met à jour
> `/opt/levelup/scripts/restic-backup.sh` automatiquement ; les units systemd
> (`/etc/systemd/system/`) ne sont PAS gérées par git → les recopier manuellement
> si elles changent.

## Vérifier

```bash
export RESTIC_REPOSITORY=/opt/levelup/restic-repo RESTIC_PASSWORD_FILE=/opt/levelup/.restic-password
restic snapshots          # liste les points de restauration
restic check              # intégrité du repo
systemctl list-timers levelup-restic-backup.timer
```

## Restaurer (rollback en cas de corruption)

```bash
ssh lvelup
export RESTIC_REPOSITORY=/opt/levelup/restic-repo RESTIC_PASSWORD_FILE=/opt/levelup/.restic-password
restic snapshots                                  # choisir un ID (ou 'latest')
cd /opt/levelup && docker compose stop levelup
restic restore latest --target /opt/levelup       # reconstruit data/... + *.json sous /opt/levelup
docker compose start levelup
```

> Restauration ciblée d'un seul fichier (ex. une DB) :
> `restic restore latest --target /tmp/rv --include '**/metadata.duckdb'`
> puis copier le fichier voulu à sa place (serveur arrêté).

## Limites assumées

- Repo sur le **même disque** que les données : si le VPS/disque est perdu, le
  repo l'est aussi. Choix validé (confiance VPS). Atténuation : copier
  périodiquement le repo + le password hors-VPS si on veut une vraie résilience
  hors-site.
