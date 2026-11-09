#!/usr/bin/env bash
# mutations_d9.sh — mutations de la regle des mains nues (correctif D9), jouees sur une copie du
# fichier source, restaure apres chaque mutation. A lancer depuis apps/go-api, GOCACHE dedie.
# Chaque mutation doit faire ROUGIR les tests de qualification (held_weapon_chain_test.go).
set -u
F=internal/games/halo_infinite/film/internal/grammar/held_weapon_changes.go
SAUVE=$(mktemp)
cp "$F" "$SAUVE"
TESTS='MainsNues|RienEnMain|DotationDeNaissance|SeCoupeAChaqueVie|NEstPasUnChangement'
mut() {
	nom=$1
	shift
	cp "$SAUVE" "$F"
	sed -i "$@" "$F"
	if cmp -s "$F" "$SAUVE"; then echo "$nom : MUTATION NON APPLIQUEE"; return; fi
	r=$(go test ./internal/games/halo_infinite/film/internal/grammar/ -run "$TESTS" -count=1 2>&1 | tail -1)
	case "$r" in ok*) echo "$nom : VERT (survit)" ;; *) echo "$nom : ROUGE" ;; esac
}
mut M1_vide_contre_mains_nues 's/case !armeEnMain(prev) \&\& !armeEnMain(ch.Family):/case false:/'
mut M2_mains_nues_sont_une_arme 's/return fam != noVariant \&\& !filmshell.IsUnarmedFamily(fam)/return fam != noVariant/'
mut M3_prise_garde_previous 's/ch.Previous, ch.Kind = noVariant, types.HeldWeaponTaken/ch.Kind = types.HeldWeaponTaken/'
mut M4_lacher_garde_famille 's/ch.Previous, ch.Family, ch.Kind = prev, noVariant, types.HeldWeaponDropped/ch.Previous, ch.Kind = prev, types.HeldWeaponDropped/'
mut M5_remise_supprimee 's/case prev == noVariant \&\& filmshell.IsUnarmedFamily(ch.Family):/case false:/'
cp "$SAUVE" "$F"
rm -f "$SAUVE"
