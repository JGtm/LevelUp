package filmdec

// objectif_ti11_oracle_test.go — L'ORACLE DE LARGEUR DE `ti=11`, ET LE BALAYAGE DES BASCULES.
//
// Le recensement voisin (`objectif_ti11_masque_test.go`) chiffre le portage et constate qu'apres
// lui 2 211 records marchent jusqu'au bout contre 884 avant. Ce fichier pose la question que ce
// chiffre NE REPOND PAS : les largeurs sont-elles JUSTES ?
//
// Voir l'en-tete du fichier voisin pour le resultat d'ensemble et sa consequence — les valeurs de
// ti=11 ne sont pas exploitables tant que le chainage n'est pas monte.

import (
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/filmproc"
)

// TestObjectifTi11Oracle — L'ORACLE DE LARGEUR, ET IL DESIGNE LE COMPOSANT FAUTIF.
//
// # CE QUE LE RECENSEMENT NE POUVAIT PAS DIRE
//
// « La marche aboutit sur 98,4 % des records » teste la COUVERTURE du dispatch, jamais la
// JUSTESSE d'une largeur : un composant qui lit deux bits de trop laisse la marche aboutir,
// simplement decalee. Et la premiere lecture des valeurs (`assaut_a10_jauge_test.go`) montre
// qu'une derive subsiste.
//
// # L'ORACLE, ET POURQUOI IL EST DIGNE DE CONFIANCE
//
// La table d'image-cle est une SUITE de records `[id:32][field:26][ti:6] + corps`. Si la
// grammaire est juste, la position de fin d'un record EST l'en-tete du suivant. `readKeyframeHeader`
// exige gen != 0, slot < 8 192 et ti < 50 : la probabilite qu'une position quelconque passe ce
// test est de l'ordre de 1e-5. Un taux de chainage de quelques dizaines de pourcents n'est donc
// PAS du hasard — c'est un vrai signal.
//
// # CE QUI REND LA MESURE LOCALISANTE
//
// Le taux de chainage est calcule PAR COMPOSANT PRESENT : d'un cote les records qui portent le
// composant, de l'autre ceux qui ne le portent pas. Une largeur juste ne change pas le taux ; une
// largeur fausse l'effondre DES QUE le composant apparait. L'ecart entre les deux colonnes DESIGNE
// le coupable, au lieu de laisser un taux global qu'on ne sait pas ou attribuer.
//
//	$env:ASSAUT_CACHE="C:/.../data/cache"
//	go test ./internal/analysis/filmdec/ -run ObjectifTi11Oracle -v -timeout 40m
func TestObjectifTi11Oracle(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandee : ASSAUT_CACHE requis")
	}
	g := filmproc.Arm("TestObjectifTi11Oracle", filmproc.MeasureLimitGiB, func(peak uint64) {
		t.Errorf("PLAFOND MEMOIRE DEPASSE (%.2f Gio) — oracle interrompu", float64(peak)/(1<<30))
	})
	defer func() { g.Disarm() }()

	var avec, avecChaine, sans, sansChaine [ti11Composants]int
	total, chaines := 0, 0
	for _, f := range ti11Corpus {
		dir := filepath.Join(cache, "film_chunks", f.id)
		reg, err := ti11Registre(dir)
		if err != nil {
			continue
		}
		n := CountFilmChunks(dir)
		for c := 1; c <= n; c++ {
			data, err := ReadFilmChunk(dir, c)
			if err != nil {
				continue
			}
			for _, pk := range WalkPackets(data) {
				if pk.Type != PacketTypeKeyframe {
					continue
				}
				pay := pk.Payload(data)
				ti11OracleTable(pay, reg, &avec, &avecChaine, &sans, &sansChaine, &total, &chaines)
			}
		}
	}

	t.Logf("########## ORACLE — %d record(s) marches, %d chaines (%.1f %%)",
		total, chaines, ti11Part(chaines, total))
	arch, _ := reg0(cache)
	for i := 0; i < ti11Composants; i++ {
		if avec[i] == 0 {
			continue
		}
		t.Logf("  i%-3d%-30s PRESENT : %5d records, %5.1f %% chaines   |   ABSENT : %6d records, %5.1f %%",
			i, ti11Nom(arch, i), avec[i], ti11Part(avecChaine[i], avec[i]),
			sans[i], ti11Part(sansChaine[i], sans[i]))
	}
}

// ti11OracleTable marche les records ti=11 d'UN payload et ventile le chainage par composant.
func ti11OracleTable(pay []byte, reg *Registry,
	avec, avecChaine, sans, sansChaine *[ti11Composants]int, total, chaines *int,
) {
	bits := len(pay) * 8
	for _, r := range WalkKeyframeWorld(pay) {
		if r.TI != ti11ArchIndex {
			continue
		}
		br := NewBitReader(pay)
		br.SetBitPos(r.Bit + keyframeRecordTIBit)
		tr := TraverseEntity(br, reg, 0)
		if tr.TypeIndex != ti11ArchIndex || tr.DesyncAt >= 0 || tr.Mask>>ti11Composants != 0 {
			continue
		}
		*total++
		_, ok := readKeyframeHeader(pay, tr.EndBit, bits)
		if ok {
			*chaines++
		}
		for i := 0; i < ti11Composants; i++ {
			present := tr.Mask>>uint(i)&1 == 1
			switch {
			case present && ok:
				avec[i]++
				avecChaine[i]++
			case present:
				avec[i]++
			case ok:
				sans[i]++
				sansChaine[i]++
			default:
				sans[i]++
			}
		}
	}
}

// reg0 charge l'archetype du premier film du corpus, pour nommer les composants du bilan.
func reg0(cache string) (Archetype, bool) {
	reg, err := ti11Registre(filepath.Join(cache, "film_chunks", ti11Corpus[0].id))
	if err != nil {
		return Archetype{}, false
	}
	return reg.Archetype(ti11ArchIndex)
}

// TestObjectifTi11Calibration — LA BASCULE QUI MANQUE, CHERCHEE PAR BALAYAGE A/B.
//
// # CE QUE L'ORACLE A DIT, ET CE QU'IL LAISSE OUVERT
//
// `TestObjectifTi11Oracle` donne un resultat tranche : les records qui ne portent QUE `i0`
// chainent, ceux qui portent quoi que ce soit d'autre chainent a ZERO. Deux lectures possibles :
// ou bien treize largeurs sont fausses a la fois — improbable, elles sortent toutes du meme
// desassemblage et sont toutes des `R(n)` plats — ou bien il MANQUE UN COUT PAR COMPOSANT que le
// traverseur ne consomme pas.
//
// La seconde a un candidat nomme, et il est deja dans le depot :
// `filmComponentCorruptionCheck`. Le commentaire de `traverseComponentLoop` le decrit : en mode
// FILM, `FUN_14076cb60` lit APRES CHAQUE COMPOSANT PRESENT un `R(1)` de garde, et un `R(32)`
// sentinelle si ce bit vaut 1. Le drapeau vaut `false` par defaut. Un cout par composant explique
// exactement la forme observee : plus il y a de composants, plus l'ecart s'accumule.
//
// # LE BALAYAGE, ET POURQUOI IL EST HONNETE
//
// Deux bascules croisees (`filmComponentCorruptionCheck` x `newRecordTailBits`), le meme corpus,
// le meme oracle. La combinaison qui fait monter le chainage GAGNE ; si aucune ne le fait monter,
// c'est un negatif net et il faudra chercher ailleurs. Le critere est ecrit avant la mesure : une
// combinaison n'est retenue que si elle depasse 60 % de chainage sur les records A PLUSIEURS
// COMPOSANTS — ceux-la, precisement, que la configuration actuelle rate a 100 %.
//
// LES BASCULES SONT GLOBALES AU PROCESS : ce test detient `LockProcessDecode` et les restaure.
//
//	$env:ASSAUT_CACHE="C:/.../data/cache"
//	go test ./internal/analysis/filmdec/ -run ObjectifTi11Calibration -v -timeout 40m
func TestObjectifTi11Calibration(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandee : ASSAUT_CACHE requis")
	}
	defer LockProcessDecode()()
	g := filmproc.Arm("TestObjectifTi11Calibration", filmproc.MeasureLimitGiB, func(peak uint64) {
		t.Errorf("PLAFOND MEMOIRE DEPASSE (%.2f Gio) — calibration interrompue", float64(peak)/(1<<30))
	})
	defer func() { g.Disarm() }()
	corrAvant, tailAvant := filmComponentCorruptionCheck, newRecordTailBits
	defer func() {
		SetFilmComponentCorruptionCheck(corrAvant)
		SetNewRecordTailBits(tailAvant)
	}()

	// Trois films d'Assaut riches en records ti=11, charges UNE fois puis rejoues.
	type film struct {
		id   string
		reg  *Registry
		pays [][]byte
	}
	var films []film
	for _, id := range []string{"34bb3bc8", "9f57c612", "ce083875"} {
		dir := filepath.Join(cache, "film_chunks", id)
		reg, err := ti11Registre(dir)
		if err != nil {
			t.Logf("%s : registre illisible (%v)", id, err)
			continue
		}
		f := film{id: id, reg: reg}
		n := CountFilmChunks(dir)
		for c := 1; c <= n; c++ {
			data, err := ReadFilmChunk(dir, c)
			if err != nil {
				continue
			}
			for _, pk := range WalkPackets(data) {
				if pk.Type == PacketTypeKeyframe {
					f.pays = append(f.pays, pk.Payload(data))
				}
			}
		}
		films = append(films, f)
	}
	t.Logf("corpus : %d film(s) charges", len(films))

	for _, corr := range []bool{false, true} {
		for _, tail := range []int{0, 1, 2} {
			SetFilmComponentCorruptionCheck(corr)
			SetNewRecordTailBits(tail)
			var un, unC, plus, plusC int
			for _, f := range films {
				for _, pay := range f.pays {
					ti11CalibPayload(pay, f.reg, &un, &unC, &plus, &plusC)
				}
			}
			t.Logf("corruption=%-5v tail=%d   UN composant : %5d records %5.1f %% chaines   |   "+
				"PLUSIEURS : %5d records %5.1f %% chaines",
				corr, tail, un, ti11Part(unC, un), plus, ti11Part(plusC, plus))
		}
	}
}

// ti11CalibPayload marche les records ti=11 d'UN payload et ventile le chainage selon que le
// record porte UN seul composant ou PLUSIEURS — c'est cette seconde colonne que la calibration
// cherche a faire monter.
func ti11CalibPayload(pay []byte, reg *Registry, un, unC, plus, plusC *int) {
	bits := len(pay) * 8
	for _, r := range WalkKeyframeWorld(pay) {
		if r.TI != ti11ArchIndex {
			continue
		}
		br := NewBitReader(pay)
		br.SetBitPos(r.Bit + keyframeRecordTIBit)
		tr := TraverseEntity(br, reg, 0)
		if tr.TypeIndex != ti11ArchIndex || tr.DesyncAt >= 0 || tr.Mask>>ti11Composants != 0 {
			continue
		}
		n := 0
		for i := 0; i < ti11Composants; i++ {
			if tr.Mask>>uint(i)&1 == 1 {
				n++
			}
		}
		if n == 0 {
			continue
		}
		_, ok := readKeyframeHeader(pay, tr.EndBit, bits)
		if n == 1 {
			*un++
			if ok {
				*unC++
			}
			continue
		}
		*plus++
		if ok {
			*plusC++
		}
	}
}
