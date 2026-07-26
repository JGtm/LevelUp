# systemd units — installation

Install: `cp levelup-*.service levelup-*.timer /etc/systemd/system/` then `systemctl daemon-reload`.
Enable: `systemctl enable --now levelup-docker-prune.timer levelup-restic-backup.timer`.
Verify: `systemctl list-timers | grep levelup` (next run should show the scheduled time).
These units are NOT managed by git deploy — recopy manually to `/etc/systemd/system/` whenever they change (see `scripts/RESTIC_BACKUP.md`).

PENDING VPS ACTION (2026-07-26): `levelup-docker-prune.service` changed scope — it no longer
runs `docker system prune -af` (that command deleted the tagged base images and the BuildKit
build cache every night, turning every deploy into a cold build; root cause of the 25-26/07 VPS
freezes). The installed copy on the VPS still holds the old command until someone recopies it:
`cp levelup-docker-prune.service /etc/systemd/system/ && systemctl daemon-reload`. Verify with
`systemctl cat levelup-docker-prune.service` (expect three `ExecStart=` lines, no `-a`).
Cleanup (I8, 2026-07): the VPS has no cron daemon, so `/etc/cron.d/levelup-docker-prune` and `/etc/cron.d/levelup-disk-check` never ran — remove those dead files once `levelup-docker-prune.timer` is confirmed active.
