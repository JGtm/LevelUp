package filmdec

// objectif_ti11_hypotheses_test.go — LES TROIS HYPOTHESES BALAYEES, ET LEUR CONTROLE.
//
// Le fichier voisin (`objectif_ti11_delta_test.go`) porte l'oracle de chainage, sa ventilation par
// composant et le PLANCHER contre lequel tout se lit. Celui-ci porte les balayages qui ont teste
// les explications candidates, et le controle qui a fini par recadrer le chantier entier.
//
// Le bilan des cinq instruments est ecrit dans l'en-tete du fichier voisin — il s'y lit d'un bloc
// plutot que coupe en deux.

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/filmproc"
)

// TestObjectifTi11DeltaGarde — LE COUT PAR COMPOSANT, TESTE LA OU IL S'APPLIQUE.
//
// # CE QUE LA VENTILATION A DIT, ET POURQUOI ELLE DESIGNE UN COUT ET NON UNE LARGEUR
//
// Le chainage delta ventile par composant rend un resultat inhabituel : `i2 formatted-text`
// chaine a **76,6 %** quand il est present, et **tous les autres composants** effondrent le taux.
// Or les LECTEURS du jeu (vtable + 0x28, et non l'ecrivain que le premier portage avait suivi)
// confirment les largeurs portees pour i1, i3, i12, i13, i16 et i32 — quatre octets pour la
// progression, quatre pour le seuil, quatre pour la reference d'objet, et ainsi de suite.
//
// Les largeurs sont donc justes. Ce qui manque est un COUT PAR COMPOSANT, et le depot en a un
// candidat nomme : `FUN_14076cb60` lit, APRES chaque composant present, un `R(1)` de garde suivi
// d'un `R(32)` sentinelle si ce bit vaut 1. Le drapeau `filmComponentCorruptionCheck` le porte, et
// il vaut `false` par defaut.
//
// CE COUT N'A JAMAIS ETE TESTE SUR LE CHEMIN DELTA. Le balayage precedent l'avait teste sur les
// IMAGES-CLES — dont on sait depuis qu'elles ne portent pas cette grammaire du tout.
//
// Et il explique la singularite de `i2` : son deserialiseur commence par un `R(1)` de presence, le
// seul de l'archetype. Sur un record ou la presence vaut zero, il consomme un bit la ou les autres
// n'en consomment aucun — exactement le bit que le garde reclamerait.
//
// LES BASCULES SONT GLOBALES AU PROCESS : ce test detient `LockProcessDecode` et les restaure.
//
//	go test ./internal/analysis/filmdec/ -run ObjectifTi11DeltaGarde -v -timeout 40m
func TestObjectifTi11DeltaGarde(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandee : ASSAUT_CACHE requis")
	}
	defer LockProcessDecode()()
	g := filmproc.Arm("TestObjectifTi11DeltaGarde", filmproc.MeasureLimitGiB, func(peak uint64) {
		t.Errorf("PLAFOND MEMOIRE DEPASSE (%.2f Gio) — mesure interrompue", float64(peak)/(1<<30))
	})
	defer func() { g.Disarm() }()
	avant := filmComponentCorruptionCheck
	defer SetFilmComponentCorruptionCheck(avant)

	type film struct {
		arch Archetype
		band map[uint32]bool
		pays [][]byte
	}
	var films []film
	for _, f := range ti11Corpus {
		dir := filepath.Join(cache, "film_chunks", f.id)
		n := CountFilmChunks(dir)
		if n == 0 {
			continue
		}
		arch, _, err := objectiveArchetype(dir)
		if err != nil {
			continue
		}
		band := observedSlotBand(dir, n, ObjectiveTypeIndex)
		if len(band) == 0 {
			continue
		}
		x := film{arch: arch, band: band}
		for c := 1; c <= n; c++ {
			data, err := ReadFilmChunk(dir, c)
			if err != nil {
				continue
			}
			for _, pk := range WalkPackets(data) {
				if pk.Type == PacketTypeDelta {
					x.pays = append(x.pays, pk.Payload(data))
				}
			}
		}
		films = append(films, x)
	}
	t.Logf("corpus : %d film(s) charges", len(films))

	for _, garde := range []bool{false, true} {
		SetFilmComponentCorruptionCheck(garde)
		var b ti11DeltaBilan
		for _, x := range films {
			for _, pay := range x.pays {
				ti11DeltaPayload(pay, x.band, x.arch, &b)
			}
		}
		t.Logf("garde=%-5v  %6d marche(s), %5.1f %% chainees   |   par taille : %s",
			garde, b.marches, ti11Part(b.chaines, b.marches), ti11TailleLigne(&b))
	}
}

// ti11TailleLigne rend le chainage par nombre de composants presents, en une ligne.
func ti11TailleLigne(b *ti11DeltaBilan) string {
	out := ""
	for n := 1; n < len(b.tailles); n++ {
		if b.tailles[n] == 0 {
			continue
		}
		if out != "" {
			out += ", "
		}
		out += fmt.Sprintf("%d comp %.1f %% (%d)", n, ti11Part(b.taillesChaine[n], b.tailles[n]), b.tailles[n])
	}
	return out
}

// TestObjectifTi11DeltaTemoin — LE CONTROLE QUI REND LES TAUX DE CHAINAGE INTERPRETABLES.
//
// # POURQUOI IL EST INDISPENSABLE, ET POURQUOI IL AURAIT DU VENIR EN PREMIER
//
// Tous les taux de chainage de ce chantier — 29 %, 33 %, 76 % — sont compares implicitement a
// ZERO. Cette comparaison ne vaut que si `worldObjectHeaderAt` ne passe QUASIMENT JAMAIS a une
// position quelconque. Ce n'est pas verifie, et deux indices disent qu'il faut le verifier :
//
//   - la mesure des largeurs par masque singleton a rendu un pic a d=3 pour DES COMPOSANTS
//     DIFFERENTS (i0, i1, i29), ce qu'aucune largeur ne peut expliquer ;
//   - 76 % de chainage pour `i2`, dont le deserialiseur ne consomme qu'UN bit sur la plupart des
//     records, est trop beau pour une grammaire dont tout le reste echoue.
//
// # LE TEMOIN
//
// Pour chaque record ancre, on teste `worldObjectHeaderAt` a une serie de decalages ARBITRAIRES,
// choisis pour ne coincider avec AUCUNE largeur portee de l'archetype (ni aucune somme de deux).
// Le taux obtenu est le PLANCHER. Un chainage qui ne le depasse pas franchement ne mesure rien.
//
//	go test ./internal/analysis/filmdec/ -run ObjectifTi11DeltaTemoin -v -timeout 40m
func TestObjectifTi11DeltaTemoin(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandee : ASSAUT_CACHE requis")
	}
	g := filmproc.Arm("TestObjectifTi11DeltaTemoin", filmproc.MeasureLimitGiB, func(peak uint64) {
		t.Errorf("PLAFOND MEMOIRE DEPASSE (%.2f Gio) — mesure interrompue", float64(peak)/(1<<30))
	})
	defer func() { g.Disarm() }()

	// Aucun de ces decalages n'est une largeur portee (1, 3, 4, 8, 14, 32) ni une somme de deux.
	arbitraires := []int{23, 29, 37, 41, 53, 59, 61, 67, 71, 73}
	hist := map[int]int{} // decalage -> nombre de records ou l'en-tete passe
	records := 0
	for _, f := range ti11Corpus {
		dir := filepath.Join(cache, "film_chunks", f.id)
		n := CountFilmChunks(dir)
		if n == 0 {
			continue
		}
		arch, _, err := objectiveArchetype(dir)
		if err != nil {
			continue
		}
		band := observedSlotBand(dir, n, ObjectiveTypeIndex)
		if len(band) == 0 {
			continue
		}
		for c := 1; c <= n; c++ {
			data, err := ReadFilmChunk(dir, c)
			if err != nil {
				continue
			}
			for _, pk := range WalkPackets(data) {
				if pk.Type != PacketTypeDelta {
					continue
				}
				records += ti11TemoinPayload(pk.Payload(data), band, arch, hist)
			}
		}
	}

	if records == 0 {
		t.Logf("aucun record ancre — le temoin ne peut rien dire.")
		return
	}
	somme := 0
	t.Logf("########## TEMOIN — %d record(s) ancre(s)", records)
	for _, d := range arbitraires {
		somme += hist[d]
		t.Logf("   decalage arbitraire +%2d : %5.1f %% des records portent un en-tete valide",
			d, ti11Part(hist[d], records))
	}
	t.Logf("PLANCHER MOYEN : %.1f %% — tout taux de chainage doit se lire CONTRE ce chiffre.",
		ti11Part(somme, records*len(arbitraires)))
	// LE PROFIL COMPLET des decalages 0 a 64, pour voir si une position domine structurellement.
	t.Logf("PROFIL 0..64 (part des records portant un en-tete a ce decalage) :")
	t.Logf("   %s", ti11CarteLigne(ti11Profil(hist, 65), records))
}

// ti11TemoinPayload teste tous les decalages de 0 a 64 plus les arbitraires, par record ancre.
func ti11TemoinPayload(pay []byte, band map[uint32]bool, arch Archetype, hist map[int]int) int {
	limit := len(pay)*8 - (worldObjectHeaderBits + worldObjectIndexBits)
	n := 0
	for p := 0; p <= limit; p++ {
		rec, ok := matchWorldObjectRecord(pay, p, band)
		if !ok || !ti11IdxDansDomaine(rec.Idx, len(arch.Components)) {
			continue
		}
		n++
		for d := 0; d <= 80; d++ {
			if worldObjectHeaderAt(pay, rec.After+d) {
				hist[d]++
			}
		}
		p = rec.After
	}
	return n
}

// ti11Profil convertit l'histogramme en tableau dense de 0 a n-1.
func ti11Profil(hist map[int]int, n int) []int {
	out := make([]int, n)
	for d, k := range hist {
		if d >= 0 && d < n {
			out[d] = k
		}
	}
	return out
}

// TestObjectifTi11DeltaPresence — L'HYPOTHESE DU BIT DE PRESENCE PAR COMPOSANT.
//
// # CE QUE LE TEMOIN A RENDU LISIBLE
//
// Le plancher de `worldObjectHeaderAt` a une position arbitraire vaut **3 %**. Les taux se lisent
// donc contre lui : 76,6 % pour `i2` est VINGT-CINQ FOIS le plancher — un signal massif ; 5,3 %
// pour les records a plusieurs composants est le plancher lui-meme — du bruit.
//
// UN SEUL COMPOSANT LIT JUSTE, et c'est le seul dont le deserialiseur commence par un `R(1)` de
// presence (`consumeObjectiveFormattedText` : si le bit vaut zero, il rend la main apres UN bit).
// Les lecteurs du jeu, eux, confirment les largeurs des autres — 32 bits pour la progression, le
// seuil, la reference d'objet. Les deux faits ne se concilient que d'une facon : **le chemin
// DELTA n'envoie pas la valeur entiere, il envoie d'abord un bit qui dit si elle a change.**
//
// # LE BALAYAGE, trois lectures, critere ecrit avant la mesure
//
//	AUCUN       la valeur est lue telle quelle (l'etat actuel du portage) ;
//	PREFIXE     un `R(1)` est consomme AVANT chaque composant, puis la valeur ;
//	GARDE       un `R(1)` est consomme, et la valeur n'est lue QUE s'il vaut 1 — la vraie
//	            semantique d'un delta.
//
// Une lecture n'est retenue que si elle porte les records a PLUSIEURS composants nettement
// au-dessus du plancher de 3 % — c'est eux, et eux seuls, que la configuration actuelle rate.
//
//	go test ./internal/analysis/filmdec/ -run ObjectifTi11DeltaPresence -v -timeout 40m
func TestObjectifTi11DeltaPresence(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandee : ASSAUT_CACHE requis")
	}
	g := filmproc.Arm("TestObjectifTi11DeltaPresence", filmproc.MeasureLimitGiB, func(peak uint64) {
		t.Errorf("PLAFOND MEMOIRE DEPASSE (%.2f Gio) — mesure interrompue", float64(peak)/(1<<30))
	})
	defer func() { g.Disarm() }()

	type film struct {
		arch Archetype
		band map[uint32]bool
		pays [][]byte
	}
	var films []film
	for _, f := range ti11Corpus {
		dir := filepath.Join(cache, "film_chunks", f.id)
		n := CountFilmChunks(dir)
		if n == 0 {
			continue
		}
		arch, _, err := objectiveArchetype(dir)
		if err != nil {
			continue
		}
		band := observedSlotBand(dir, n, ObjectiveTypeIndex)
		if len(band) == 0 {
			continue
		}
		x := film{arch: arch, band: band}
		for c := 1; c <= n; c++ {
			data, err := ReadFilmChunk(dir, c)
			if err != nil {
				continue
			}
			for _, pk := range WalkPackets(data) {
				if pk.Type == PacketTypeDelta {
					x.pays = append(x.pays, pk.Payload(data))
				}
			}
		}
		films = append(films, x)
	}
	t.Logf("corpus : %d film(s) ; PLANCHER de reference : 3 %%", len(films))

	for _, mode := range []int{ti11PresAucun, ti11PresPrefixe, ti11PresGarde} {
		var b ti11DeltaBilan
		for _, x := range films {
			for _, pay := range x.pays {
				ti11PresencePayload(pay, x.band, x.arch, mode, &b)
			}
		}
		t.Logf("%-8s %6d marche(s), %5.1f %% chainees   |   %s",
			ti11PresNom(mode), b.marches, ti11Part(b.chaines, b.marches), ti11TailleLigne(&b))
	}
}

// Les trois lectures du prefixe de composant.
const (
	ti11PresAucun = iota
	ti11PresPrefixe
	ti11PresGarde
)

func ti11PresNom(m int) string {
	switch m {
	case ti11PresPrefixe:
		return "PREFIXE"
	case ti11PresGarde:
		return "GARDE"
	}
	return "AUCUN"
}

// ti11PresencePayload marche les records delta sous la lecture demandee.
func ti11PresencePayload(pay []byte, band map[uint32]bool, arch Archetype, mode int, b *ti11DeltaBilan) {
	total := len(pay) * 8
	limit := total - (worldObjectHeaderBits + worldObjectIndexBits)
	for p := 0; p <= limit; p++ {
		rec, ok := matchWorldObjectRecord(pay, p, band)
		if !ok || !ti11IdxDansDomaine(rec.Idx, len(arch.Components)) {
			continue
		}
		fin, done := ti11MarcheAvecPresence(pay, rec, arch, total, mode)
		p = rec.After
		if !done {
			continue
		}
		b.marches++
		if worldObjectHeaderAt(pay, fin) {
			b.chaines++
			if n := len(rec.Idx); n < len(b.tailles) {
				b.taillesChaine[n]++
			}
		}
		if n := len(rec.Idx); n < len(b.tailles) {
			b.tailles[n]++
		}
	}
}

// ti11MarcheAvecPresence rejoue les composants sous la lecture demandee du prefixe.
func ti11MarcheAvecPresence(pay []byte, rec WorldObjectRecord, arch Archetype, total, mode int) (int, bool) {
	at := rec.After
	for _, id := range rec.Idx {
		name := arch.component(id)
		if name == "" || at > total {
			return at, false
		}
		br := NewBitReader(pay)
		br.SetBitPos(at)
		lire := true
		if mode != ti11PresAucun {
			bit := br.ReadBit()
			lire = mode == ti11PresPrefixe || bit
		}
		if lire {
			_, _, ported := consumeByName(br, name, ObjectiveTypeIndex, arch.Level(id))
			if !ported {
				return at, false
			}
		}
		if br.BitPos() > total {
			return at, false
		}
		at = br.BitPos()
	}
	return at, true
}
