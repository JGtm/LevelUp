#!/bin/bash
# docker-entrypoint.sh — Fix permissions on bind-mounted volumes, then drop to appuser.
#
# Pattern standard Docker (cf. postgres, redis, nginx official images).
# Le container démarre en root, corrige les permissions une seule fois,
# puis exécute la commande en tant qu'appuser via gosu.
#
# Élimine TOUS les problèmes de UID mismatch entre l'hôte (deploy=1000)
# et le container (appuser). UID 1000 = même que deploy sur le VPS.
set -e

if [ "$(id -u)" = '0' ]; then
    # Fixer les permissions sur les données bind-mountées
    # (silencieux sur :ro mounts — chown échoue mais ce n'est pas grave)
    chown -R appuser:appuser /app/data 2>/dev/null || true

    # Fixer les fichiers de config bind-mountés individuellement
    for f in /app/db_profiles.json /app/app_settings.json; do
        [ -e "$f" ] && chown appuser:appuser "$f" 2>/dev/null || true
    done

    # Drop vers appuser pour exécuter la commande
    exec gosu appuser "$@"
fi

# Si déjà appuser (ou autre non-root), exécuter directement
exec "$@"
