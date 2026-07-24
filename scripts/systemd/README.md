# systemd units — installation

Install: `cp levelup-*.service levelup-*.timer /etc/systemd/system/` then `systemctl daemon-reload`.
Enable: `systemctl enable --now levelup-docker-prune.timer levelup-restic-backup.timer`.
Verify: `systemctl list-timers | grep levelup` (next run should show the scheduled time).
These units are NOT managed by git deploy — recopy manually to `/etc/systemd/system/` whenever they change (see `scripts/RESTIC_BACKUP.md`).
Cleanup (I8, 2026-07): the VPS has no cron daemon, so `/etc/cron.d/levelup-docker-prune` and `/etc/cron.d/levelup-disk-check` never ran — remove those dead files once `levelup-docker-prune.timer` is confirmed active.
