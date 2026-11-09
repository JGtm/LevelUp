#!/usr/bin/env bash
# Verification de version unique (ADR 0045 D-15, plan A9) : refuse tout ecart entre la version
# attendue (le tag git sans son `v`, ou, hors tag, la version de Cargo.toml), `version` de
# apps/desktop/src-tauri/Cargo.toml, `version` de apps/desktop/src-tauri/tauri.conf.json et la
# version passee a `go build -ldflags "-X main.version=..."` (relue dans le binaire serveur
# quand il est fourni : `go version -m` en liste les parametres de compilation).
#
# Avec une empreinte de paquet (troisieme argument), refuse aussi un serveur qui n'embarque pas EXACTEMENT cette
# empreinte (`-X main.resourcesFingerprint=<empreinte>`, ADR 0045 amendement L24) : un moteur qui attendrait un
# autre paquet que celui livre avec lui refuserait de demarrer (heberge) ou le telechargerait en boucle (bureau).
#
# Usage : check-version.sh <version attendue> [<serveur.exe> [<empreinte du paquet>]]
#
# Sortie : une ligne `::error::` par ecart (annotation GitHub Actions), code 1 s'il y en a.
set -euo pipefail

expected="${1:?usage: check-version.sh <version attendue> [<serveur.exe> [<empreinte du paquet>]]}"
server_exe="${2:-}"
pack_fingerprint="${3:-}"
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
cargo_toml="$root/apps/desktop/src-tauri/Cargo.toml"
tauri_conf="$root/apps/desktop/src-tauri/tauri.conf.json"

fail=0
problem() {
  echo "::error::$1"
  fail=1
}

# Premiere cle `version = "..."` du fichier : celle de [package] (les dependances sont apres).
cargo_version="$(sed -n 's/^version *= *"\([^"]*\)".*/\1/p' "$cargo_toml" | head -n 1)"
conf_version="$(sed -n 's/^ *"version" *: *"\([^"]*\)".*/\1/p' "$tauri_conf" | head -n 1)"

[ -n "$cargo_version" ] || problem "version introuvable dans $cargo_toml"
[ -n "$conf_version" ] || problem "version introuvable dans $tauri_conf"
[ "$cargo_version" = "$expected" ] || problem "Cargo.toml dit $cargo_version, la version attendue est $expected (tag ou source de verite)"
[ "$conf_version" = "$expected" ] || problem "tauri.conf.json dit $conf_version, la version attendue est $expected (tag ou source de verite)"

if [ -n "$server_exe" ]; then
  if [ ! -f "$server_exe" ]; then
    problem "serveur introuvable : $server_exe"
  elif ! go version -m "$server_exe" | grep -E -q -- "main.version=$(printf '%s' "$expected" | sed 's/[.]/\./g')([\" ]|$)"; then
    embedded="$(go version -m "$server_exe" | grep -o 'main\.version=[^" ]*' | head -n 1 || true)"
    problem "le serveur compile ne porte pas -X main.version=$expected dans ses -ldflags (trouve : ${embedded:-aucun})"
  fi
fi

if [ -n "$pack_fingerprint" ]; then
  if ! printf '%s' "$pack_fingerprint" | grep -E -q '^[0-9a-f]{64}$'; then
    problem "l'empreinte du paquet n'est pas un SHA-256 en hexadecimal minuscule : $pack_fingerprint"
  elif [ -z "$server_exe" ]; then
    problem "une empreinte de paquet est donnee sans serveur a controler"
  elif [ -f "$server_exe" ] && ! go version -m "$server_exe" | grep -F -q -- "main.resourcesFingerprint=${pack_fingerprint}"; then
    embedded_fp="$(go version -m "$server_exe" | grep -o 'main.resourcesFingerprint=[^" ]*' | head -n 1 || true)"
    problem "le serveur compile ne porte pas -X main.resourcesFingerprint=$pack_fingerprint dans ses -ldflags (trouve : ${embedded_fp:-aucun})"
  fi
fi

if [ "$fail" -ne 0 ]; then
  exit 1
fi
echo "version $expected : tag/source, Cargo.toml, tauri.conf.json${server_exe:+, -ldflags du serveur}${pack_fingerprint:+, empreinte du paquet} concordent"
