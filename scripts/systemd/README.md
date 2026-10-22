# systemd units — installation

Install: `cp levelup-*.service levelup-*.timer /etc/systemd/system/` then `systemctl daemon-reload`.
Enable: `systemctl enable --now levelup-docker-prune.timer levelup-restic-backup.timer levelup-compact-passes.timer`.
Verify: `systemctl list-timers | grep levelup` (next run should show the scheduled time).
These units are NOT managed by git deploy — recopy manually to `/etc/systemd/system/` whenever they change (see `scripts/RESTIC_BACKUP.md`).

PENDING VPS ACTION (2026-07-26): `levelup-docker-prune.service` changed scope — it no longer
runs `docker system prune -af` (that command deleted the tagged base images and the BuildKit
build cache every night, turning every deploy into a cold build; root cause of the 25-26/07 VPS
freezes). The installed copy on the VPS still holds the old command until someone recopies it:
`cp levelup-docker-prune.service /etc/systemd/system/ && systemctl daemon-reload`. Verify with
`systemctl cat levelup-docker-prune.service` (expect four `ExecStart=` lines; the only `-a` one carries `until=720h`).
Cleanup (I8, 2026-07): the VPS has no cron daemon, so `/etc/cron.d/levelup-docker-prune` and `/etc/cron.d/levelup-disk-check` never ran — remove those dead files once `levelup-docker-prune.timer` is confirmed active.

## levelup-compact-passes (nightly compaction of superseded decode passes)

`levelup-compact-passes.timer` runs `scripts/compact-passes-nuit.sh` every night at 03:30 UTC
(after the 03:00 Docker prune, before the 04:00 restic backup). The script reads the
"passes per match" gauge written by the server's data health check
(`data/global/admin_state/passes_par_match.txt`). When no table of any title is above the
threshold it exits without touching anything. Otherwise it stops `levelup`, runs
`levelup compact-passes --backup-dir /app/data/compaction-backups` (file backup first), starts
`levelup` again (always, even if the compaction fails, up to 3 attempts) and waits for `/health`. Log:
`data/logs/compact-passes-nuit.log`. Backups older than 24 h in `data/compaction-backups/` are
deleted only after a successful run. Exit code 75 = run refused without touching anything (lock
held, restic backup running), declared as success in the unit.

The script itself is updated by every deploy (it lives in the repository at
`/opt/levelup/scripts/`); the units are not.

Install (once, on the VPS, as root):

```bash
cd /opt/levelup
cp scripts/systemd/levelup-compact-passes.service scripts/systemd/levelup-compact-passes.timer /etc/systemd/system/
systemctl daemon-reload
systemctl enable --now levelup-compact-passes.timer
systemctl list-timers | grep levelup-compact-passes   # next run: 03:30 UTC
```

Check after the first night: `tail -n 50 /opt/levelup/data/logs/compact-passes-nuit.log`
("rien a faire" or "compaction reussie"). Manual run: `systemctl start levelup-compact-passes.service`.
