package halo_infinite

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/damagetag"
)

// TestKillSourceRegistryKey_LesTroisVoies : les deux voies de resolution rendent bien ce
// qu'on attend, et le reste du monde ne remonte RIEN.
//
// Le dernier cas est le plus important : une arme a feu ordinaire ne doit PAS sortir
// d'ici comme « hors arsenal ». Elle a le droit d'avoir une cle (killicon en pose une
// pour l'icone du kill feed) — c'est le repo qui l'ecartera parce qu'elle porte un id
// numerique. Ce test documente donc la frontiere exacte : ce fichier TRADUIT, il ne
// filtre pas.
func TestKillSourceRegistryKey_LesTroisVoies(t *testing.T) {
	r := NewKillSourceRegistry()

	// Voie 1 : le repulseur, par la table des vignettes.
	if got, ok := r.KillSourceRegistryKey(0x07104b31); !ok || got != "hinf_repulsor" {
		t.Errorf("repulseur = (%q, %v), want (hinf_repulsor, true)", got, ok)
	}

	// Voie 2 : la chute et l'environnement, par la CLASSE — aucune de ces 9 lignes n'a
	// de vignette, et c'est justement pourquoi la voie 2 existe.
	var globaux int
	for _, l := range damagetag.Labels() {
		if l.Class != damagetag.ClassGlobal {
			continue
		}
		globaux++
		got, ok := r.KillSourceRegistryKey(l.Tag)
		if !ok || got != keyEnvironment {
			t.Errorf("tag %08x (DEGAT_GLOBAL) = (%q, %v), want (%s, true)",
				l.Tag, got, ok, keyEnvironment)
		}
	}
	if globaux == 0 {
		t.Fatal("aucun tag DEGAT_GLOBAL dans la table : le test ne prouve rien")
	}

	// Les bobines : toutes les lignes OBJET_EXPLOSIF qui portent une regle de banque
	// resolvent vers une cle de bobine. On ne verifie pas un tag en dur (la table
	// grandit) mais qu'AU MOINS une resolution a lieu, et qu'elle vise bien une bobine.
	var bobines int
	for _, l := range damagetag.Labels() {
		if l.Class != damagetag.ClassObjet {
			continue
		}
		if got, ok := r.KillSourceRegistryKey(l.Tag); ok {
			bobines++
			if len(got) < 10 || got[:10] != "hinf_coil_" {
				t.Errorf("tag %08x (OBJET_EXPLOSIF) -> %q, want une cle hinf_coil_*", l.Tag, got)
			}
		}
	}
	if bobines == 0 {
		t.Error("aucune bobine resolue : le pont killicon -> registre est casse")
	}

	// Une source inconnue ne rend rien — jamais de cle « par defaut ».
	if got, ok := r.KillSourceRegistryKey(0xffffffff); ok {
		t.Errorf("tag inconnu -> (%q, true), want (\"\", false)", got)
	}
}
