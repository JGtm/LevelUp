package grammar

// replis_du_film_test.go — LE RAPPORT DES REPLIS DU CONTEXTE DE FILM (lot J8.7, 2026-09-27).
//
// Ce que ces tests tiennent, maillon par maillon : le rapport SOMME tous ses champs, et chaque site
// qui decide un repli de `grammar` le NOTE au contexte qui l execute. Le dernier maillon — du
// rapport au compteur de la cuisson — est tenu par `replay/versement_des_replis_test.go`.
//
// MUTATION JOUEE (2026-09-27) : retirer `c.NoterReplis(...)` de [FilmContext.ChunkNumbers] fait
// rougir `TestLeContexteCompteLesChunksAbandonnesAuTrou` (« 0 chunk abandonne, attendu 2 »).

import (
	"os"
	"reflect"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/finalise"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// TestPlusSommeChaqueChampDuRapport : [ComptesDesReplis.Plus] nomme ses champs a la main ; un champ
// ajoute au type et oublie la serait perdu a la premiere somme. La reflexion pose une valeur
// distincte dans chaque champ et exige le double apres `r.Plus(r)`.
func TestPlusSommeChaqueChampDuRapport(t *testing.T) {
	var r ComptesDesReplis
	v := reflect.ValueOf(&r).Elem()
	for i := 0; i < v.NumField(); i++ {
		if v.Field(i).Kind() != reflect.Int {
			t.Fatalf("champ %s : %s, attendu int — un compte de repli est un entier", v.Type().Field(i).Name, v.Field(i).Kind())
		}
		v.Field(i).SetInt(int64(i + 1))
	}
	s := reflect.ValueOf(r.Plus(r))
	for i := 0; i < s.NumField(); i++ {
		if got, want := s.Field(i).Int(), int64(2*(i+1)); got != want {
			t.Errorf("Plus perd le champ %s : %d, attendu %d", s.Type().Field(i).Name, got, want)
		}
	}
}

// TestLeContexteCompteLesChunksAbandonnesAuTrou : un trou dans la numerotation abandonne les chunks
// qui le suivent ; le contexte en compte le nombre UNE fois, au premier releve.
func TestLeContexteCompteLesChunksAbandonnesAuTrou(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"chunk_00.bin", "chunk_01.bin", "chunk_02.bin", "chunk_04.bin", "chunk_05.bin"} {
		if err := os.WriteFile(dir+"/"+n, []byte{0}, 0o600); err != nil {
			t.Fatalf("ecriture de %s : %v", n, err)
		}
	}
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("LoadDir : %v", err)
	}
	fc := NewFilmContext(film)
	if got := fc.ChunkNumbers(); len(got) != 2 {
		t.Fatalf("ChunkNumbers = %v, attendu [1 2]", got)
	}
	fc.ChunkNumbers() // memorise : le second releve ne recompte pas
	if got := fc.ComptesDesReplis().ChunksApresTrouAbandonnes; got != 2 {
		t.Fatalf("%d chunk(s) abandonne(s) au rapport, attendu 2 (le 04 et le 05)", got)
	}
}

// TestLaPoseDesLargeursRendSesReplis : les deux replis de la pose des largeurs de la carte se rendent
// en donnees, et le contexte qui pose les note.
func TestLaPoseDesLargeursRendSesReplis(t *testing.T) {
	cas := []struct {
		nom  string
		lay  profile.I0Layout
		want ComptesDesReplis
	}{
		{"decoupage non detecte", profile.I0Layout{}, ComptesDesReplis{LargeursMondeParDefaut: 1}},
		{"porte trop courte", profile.I0Layout{AxisW: [3]uint{13, 13, 14}}, ComptesDesReplis{IndexDeRegionLargeurUn: 1}},
		{"carte complete", profile.I0Layout{GateBits: profile.DefaultI0GateBits, AxisW: [3]uint{13, 13, 14}}, ComptesDesReplis{}},
	}
	for _, c := range cas {
		p := ProfilDeBalayageParDefaut()
		if got := p.PoserLargeursObjetDuMondeDepuisDecoupage(c.lay); got != c.want {
			t.Errorf("%s : rapport %+v, attendu %+v", c.nom, got, c.want)
		}
		fc := NewFilmContext(nil)
		fc.PoserLargeursObjetDuMondeDepuisDecoupage(c.lay)
		if got := fc.ComptesDesReplis(); got != c.want {
			t.Errorf("%s : le contexte a note %+v, attendu %+v", c.nom, got, c.want)
		}
	}
}

// TestLesVerdictsParFilmNeSeComptentQuUneFois : le decoupage d i0 auto-detecte et le registre
// inconnu sont des verdicts PAR FILM — deux sites qui les rencontrent ne comptent qu un
// declenchement ; une detection en echec n en compte aucun.
func TestLesVerdictsParFilmNeSeComptentQuUneFois(t *testing.T) {
	fc := NewFilmContext(nil)
	fc.noterI0ParDefaut(os.ErrNotExist)
	if got := fc.ComptesDesReplis().I0PorteEtRegionParDefaut; got != 0 {
		t.Fatalf("detection en echec comptee : %d", got)
	}
	fc.noterI0ParDefaut(nil)
	fc.noterI0ParDefaut(nil)
	if got := fc.ComptesDesReplis().I0PorteEtRegionParDefaut; got != 1 {
		t.Errorf("decoupage auto-detecte : %d declenchement(s), attendu 1", got)
	}
	fc.noterRegistre(&Registry{fingerprint: KnownRegistryFingerprint})
	if got := fc.ComptesDesReplis().RegistreInconnu; got != 0 {
		t.Fatalf("registre de reference compte comme inconnu : %d", got)
	}
	fc.noterRegistre(&Registry{fingerprint: KnownRegistryFingerprint + 1})
	fc.noterRegistre(&Registry{fingerprint: KnownRegistryFingerprint + 1})
	if got := fc.ComptesDesReplis().RegistreInconnu; got != 1 {
		t.Errorf("registre inconnu : %d declenchement(s), attendu 1", got)
	}
	var nul *FilmContext
	nul.NoterReplis(ComptesDesReplis{RegistreInconnu: 1}) // sur sur nil
	if got := nul.ComptesDesReplis(); got != (ComptesDesReplis{}) {
		t.Errorf("contexte nil : rapport %+v, attendu vide", got)
	}
}

// TestLePontCompteLeFilDesMortsAuDernierNumero : sans manifeste type, l etage du pont lit le fil des
// morts au dernier numero et le note au contexte ; avec le morceau des temps forts type, rien.
func TestLePontCompteLeFilDesMortsAuDernierNumero(t *testing.T) {
	o := octetsBobineV40(t)
	for _, c := range []struct {
		nom  string
		meta []types.ChunkMeta
		want int
	}{
		{"sans manifeste", nil, 1},
		{"manifeste type", []types.ChunkMeta{{Index: 0, ChunkType: 1}, {Index: 1, ChunkType: 2},
			{Index: 2, ChunkType: finalise.ChunkTypeTempsForts}}, 0},
	} {
		film, err := source.Load(source.MemoryChunks(o), c.meta)
		if err != nil {
			t.Fatalf("%s : chargement : %v", c.nom, err)
		}
		fc := NewFilmContext(film)
		l := ScanPontDIdentite(fc, OptionsDuPont{Balayage: ScanFilmOptions{QuantaOnly: true}})
		if l.ErrMorts != nil {
			t.Fatalf("%s : fil des morts illisible : %v", c.nom, l.ErrMorts)
		}
		if got := fc.ComptesDesReplis().TempsFortsAuDernierNumero; got != c.want {
			t.Errorf("%s : %d declenchement(s) au rapport, attendu %d", c.nom, got, c.want)
		}
	}
}
