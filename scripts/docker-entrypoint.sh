#!/bin/bash
# docker-entrypoint.sh — Fix permissions on bind-mounted volumes, then drop to appuser.
#
# Pattern standard Docker (cf. postgres, redis, nginx official images).
# Le container demarre en root, corrige les permissions une seule fois,
# puis execute la commande en tant qu'appuser via gosu.
#
# Elimine tous les problemes de UID mismatch entre l'hote (deploy=1000) et le
# container (appuser). UID 1000 = meme que deploy sur le VPS.
#
# Note : restaure le 2026-05-20 dans le sprint chore/ci-stabilization. Le
# fichier avait ete supprime par erreur dans le commit c03707aa "chore:
# supprimer Python legacy complet" — il est universel (s'applique au binaire
# Go aussi), le Dockerfile en a toujours besoin (cf. ligne 88
# ENTRYPOINT + scenarios test-deploy-precheck.yml jobs A/B/C/D).
set -e

if [ "$(id -u)" = '0' ]; then
    # Fixer les permissions sur les donnees bind-mountees
    # (silencieux sur :ro mounts — chown echoue mais ce n'est pas grave)
    chown -R appuser:appuser /app/data 2>/dev/null || true

    # Fixer les fichiers de config bind-mountes individuellement
    for f in /app/db_profiles.json /app/app_settings.json; do
        [ -e "$f" ] && chown appuser:appuser "$f" 2>/dev/null || true
    done

    # Drop vers appuser pour executer la commande
    exec gosu appuser "$@"
fi

# Si deja appuser (ou autre non-root), executer directement
exec "$@"
