package profile_test

// rev_test.go — LE GATE DE [profile.Rev], SUR LE MECANISME CENTRAL (`film/revision`, lot 2.6.0).
//
// # CE QU IL TIENT, ET LES DEUX GESTES QU IL EXIGE
//
// Il hache les sources non-test de la couche `profile` sous son perimetre — la fermeture de ses
// imports (lot J3.2, DU-2 (b)), qui ne rencontre AUCUNE autre couche : `profile` n importe pas
// `source`, donc la valeur de `source.Rev` n entre plus dans son empreinte —, et compare au
// golden, qui porte le couple (revision, empreinte). Toucher la couche le fait rougir ; le
// remettre au vert demande de rouvrir la ligne de la revision — donc de DECIDER si le decodage
// change. Les trois derives sont distinguees, comme pour les trois autres couches : « les sources
// ont change », « la revision a change sans la couche », « le golden est perime ».
//
// # POURQUOI IL N Y A PAS UNE LIGNE D EMPREINTE ICI
//
// CLAUDE.md regle 6 : le calcul, la chronique, la porte et les messages sont CENTRAUX depuis le
// lot 2.6.0, et le garde-rail `archlint/no_ad_hoc_source_fingerprint_test.go` interdit la copie.
// Ce fichier ne fait que DECLARER ce qui est propre a la couche : sa racine, son amont, son
// golden, sa porte, et la QUESTION que son echec pose.

import (
	"flag"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/revision"
)

// updateProfileRev : LA PORTE DE REGENERATION DE CE GOLDEN, ET D AUCUN AUTRE. Elle est NOMMEE
// (correctif de la revue R1, P1-1) : accrochee a un `-update` generique, elle laisserait
// `go test ./...profile/ -update` refiger l empreinte d une couche CASSEE en repondant `ok`.
var updateProfileRev = flag.Bool("update-profile-rev", false,
	"reecrire testdata/profile_rev.golden (revision ET empreinte) — CE golden seulement")

const (
	// cheminGoldenProfileRev : le golden, relatif au paquet.
	cheminGoldenProfileRev = "testdata/profile_rev.golden"
	// fichierPorteurDeRevisionProfile : le fichier qui PORTE la revision, exclu du hachage.
	fichierPorteurDeRevisionProfile = "rev.go"
	// prefixeRevisionProfile : le prefixe de la serie, pour la chronique.
	prefixeRevisionProfile = "profile"
	// commandeRegenerationProfileRev : la commande complete, telle qu on la tape.
	commandeRegenerationProfileRev = "LEVELUP_UPDATE_PROFILE_REV=1 go test " +
		"./internal/games/halo_infinite/film/internal/profile/ -run TestProfileRevSuitLaCoucheProfil " +
		"-update-profile-rev"
)

// porteProfileRev / messagesProfileRev : ce que la couche declare au mecanisme central.
func porteProfileRev() revision.Porte { return revision.Porte{Nom: "profile-rev"} }

func messagesProfileRev() revision.Messages {
	return revision.Messages{
		Couche:    prefixeRevisionProfile,
		Constante: "profile.Rev",
		Forme:     "profile-AAAA-MM-JJ[.N]",
		Porte:     porteProfileRev(),
		Commande:  commandeRegenerationProfileRev,
		Question: "LA TABLE DU DECODEUR A CHANGE : une largeur, une borne, une provenance. La " +
			"couche `profile` ne lit aucun octet (ADR 0034 D-1) mais elle dit COMMENT les lire — " +
			"une ligne changee fait lire d autres bits aux memes offsets. `grammar.Rev` hache la " +
			"VALEUR de cette revision et `facts.Rev` celle de `grammar.Rev` : la monter fait " +
			"monter les deux mecaniquement, et le backlog killsource avec elles (D6).",
	}
}

// racineDeLaCoucheProfil : le paquet lui-meme, resolu par `runtime.Caller` — pas un chemin
// relatif au repertoire courant : le jour ou la couche demenage, ce test doit suivre le paquet
// et non hacher un dossier vide.
func racineDeLaCoucheProfil(t *testing.T) string {
	t.Helper()
	_, ici, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a echoue")
	}
	return filepath.Dir(ici)
}

// empreinteDeLaCoucheProfil rend l empreinte et le nombre de fichiers haches.
//
// AUCUNE VALEUR AMONT depuis le lot J3.2 : la fermeture des imports de `profile` ne rencontre
// aucune couche revisee. Une montee de `source` fait monter les couches qui LISENT des octets —
// `grammar`, `killsource`, `objectives` importent `source` —, pas la table du decodeur.
func empreinteDeLaCoucheProfil(t *testing.T) (string, int) {
	t.Helper()
	res, err := revision.EmpreinteDeCouche(racineDeLaCoucheProfil(t), "profile",
		func(rel string) bool { return rel == fichierPorteurDeRevisionProfile }, nil)
	if err != nil {
		t.Fatalf("empreinte de la couche profil : %v", err)
	}
	return res.Empreinte, res.Fichiers
}

// TestProfileRevSuitLaCoucheProfil — LE GATE.
func TestProfileRevSuitLaCoucheProfil(t *testing.T) {
	empreinte, fichiers := empreinteDeLaCoucheProfil(t)
	msg := messagesProfileRev()
	if *updateProfileRev {
		regenererGoldenProfileRev(t, empreinte)
		return
	}
	c, err := revision.LireChronique(prefixeRevisionProfile,
		[]string{fichierPorteurDeRevisionProfile}, cheminGoldenProfileRev)
	if err != nil {
		t.Fatalf("chronique de `profile` : %v", err)
	}
	courante := c.Courante()
	switch {
	case courante.Revision == profile.Rev && courante.Empreinte != empreinte:
		t.Fatal(msg.SourcesOntChange(courante.Revision, courante.Empreinte, empreinte, fichiers))
	case courante.Revision != profile.Rev && courante.Empreinte == empreinte:
		t.Fatal(msg.RevisionSeuleAChange(profile.Rev, courante.Revision, empreinte))
	case courante.Revision != profile.Rev:
		t.Fatal(msg.GoldenPerime(profile.Rev, empreinte, fichiers))
	}
}

// regenererGoldenProfileRev : la porte, avec ses DEUX verrous — et elle ne rend JAMAIS `ok`.
func regenererGoldenProfileRev(t *testing.T, empreinte string) {
	t.Helper()
	porte := porteProfileRev()
	ouverte, raison := porte.Ouverte(true, os.Getenv(porte.Variable()))
	if !ouverte {
		t.Skip(raison)
	}
	ecrit, err := porte.Reecrire(cheminGoldenProfileRev, profile.Rev, empreinte)
	if err != nil {
		t.Fatalf("regeneration du golden : %v", err)
	}
	// `go test` JETTE la sortie d un paquet qui PASSE : une reecriture annoncee par `t.Logf`
	// serait INVISIBLE avec la commande documentee, et se lirait `ok`.
	t.Fatal(ecrit)
}

// TestChroniqueDeProfileCouvreLaRevisionCourante : la revision courante a son ENTREE dans le
// godoc de `rev.go` ET la derniere ligne du golden, et les rangs se suivent sans trou.
//
// Le defaut qu il ferme est mesure cote grammaire (constat F5 de la revue de jalon M1) : la
// chronique s y etait arretee a `.12` pendant que la constante valait `.14`, et rien ne
// rougissait — les deux changements de comportement intermediaires n avaient aucune entree.
func TestChroniqueDeProfileCouvreLaRevisionCourante(t *testing.T) {
	c, err := revision.LireChronique(prefixeRevisionProfile,
		[]string{fichierPorteurDeRevisionProfile}, cheminGoldenProfileRev)
	if err != nil {
		t.Fatalf("chronique de `profile` : %v", err)
	}
	if err := c.VerifierCouverture(profile.Rev); err != nil {
		t.Fatal(messagesProfileRev().SansEntreeDeChronique(err.Error()))
	}
	// Plancher VIDE : la chronique de `profile` nait avec ce lot, elle tient la regle depuis son
	// premier rang — il n y a aucun passe a amnistier.
	if err := c.VerifierRangs(""); err != nil {
		t.Fatal(err)
	}
}
