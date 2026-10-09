package main

// livraison_publication_test.go — LA CIBLE N'EST TOUCHEE QU'APRES UN LOT COMPLET.
//
// Le script Python vidait `static/sounds/halo_infinite` de ses `hinf_*.wav` AVANT de produire
// quoi que ce soit. Toute erreur en cours de route — source illisible, vote mal forme, disque
// plein — laissait donc le depot avec une partie seulement des armes et un
// `weaponSoundVariations.ts` jamais reecrit (constat C7 de la revue R1, ou un simple exemple
// de vote vide suffisait a declencher le cas). Le lot est desormais produit dans un
// repertoire d'attente voisin, puis publie par renommage.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLivrerNEffaceRienQuandLaProductionEchoue(t *testing.T) {
	racine := t.TempDir()
	if err := livraisonEcrireJeuSynthetique(racine); err != nil {
		t.Fatalf("jeu synthetique: %v", err)
	}
	depot := filepath.Join(racine, "_depot")
	if err := livraisonEcrireDepotSynthetique(depot); err != nil {
		t.Fatalf("depot synthetique: %v", err)
	}

	// UNE SOURCE ILLISIBLE AU MILIEU DU LOT. `UNSC_relatifdrive` est traite en septieme
	// position : cinq armes sont deja produites quand la sixieme echoue.
	corrompu := filepath.Join(racine, "UNSC_relatifdrive", "0.04s_801.wav")
	if err := os.WriteFile(corrompu, []byte("ceci n'est pas un RIFF/WAVE"), 0o644); err != nil {
		t.Fatalf("corruption de la source: %v", err)
	}

	if err := livrer(filepath.Join(racine, "_donnees"), racine, depot); err == nil {
		t.Fatal("livrer a reussi sur une source illisible — il devait rendre l'erreur")
	}

	sons := filepath.Join(depot, "static", "sounds", "halo_infinite")
	entrees, err := os.ReadDir(sons)
	if err != nil {
		t.Fatalf("lecture de la cible: %v", err)
	}
	presents := map[string]bool{}
	for _, e := range entrees {
		presents[e.Name()] = true
	}
	// LES DEUX TEMOINS : l'arme perimee (que seul un lot COMPLET a le droit d'effacer) et le
	// son d'evenement du pack utilisateur, hors perimetre du miroir.
	for _, n := range []string{"hinf_perime.wav", "melee_kill.wav"} {
		if !presents[n] {
			t.Errorf("%s a disparu alors que la production a echoue : la cible ne doit etre "+
				"touchee qu'apres un lot complet", n)
		}
	}
	if presents["hinf_ravager.wav"] {
		t.Error("une arme du lot avorte est arrivee dans la cible : la publication doit etre " +
			"tout ou rien")
	}
	if len(presents) != 2 {
		t.Errorf("la cible contient %d fichier(s) (%v), attendu les 2 temoins seuls",
			len(presents), presents)
	}

	// LE REPERTOIRE D'ATTENTE NE SURVIT PAS A L'ECHEC : il est voisin de la cible (meme
	// volume, donc renommage possible), il n'a rien a faire dans le depot une fois le mode
	// termine.
	voisins, err := os.ReadDir(filepath.Dir(sons))
	if err != nil {
		t.Fatalf("lecture du dossier des sons: %v", err)
	}
	for _, v := range voisins {
		if strings.HasPrefix(v.Name(), ".livrer-") {
			t.Errorf("repertoire d'attente %q laisse derriere l'echec", v.Name())
		}
	}
}
