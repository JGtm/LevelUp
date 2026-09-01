package filmdec

// objectif_ti11_delta_test.go — QUEL COMPOSANT FAIT ECHOUER LE CHAINAGE DELTA.
//
// # POURQUOI LE CHEMIN DELTA, ET PAS L'IMAGE-CLE
//
// La carte du corps d'image-cle a etabli qu'il est 100 % statique : la grammaire portee
// (masque + composants) est celle du chemin DELTA, et c'est la seule ou elle s'applique. Le
// chainage delta vaut aujourd'hui 13 a 65 % selon le film, contre 87 a 99 % sur `ti=13`
// correctement ancre. Ce fichier cherche l'ecart.
//
// # UNE PREMIERE APPROCHE A ETE ABANDONNEE, ET LE DIRE EVITE DE LA REFAIRE
//
// L'idee naturelle etait de MESURER chaque largeur directement : un record delta n'a pas de
// prologue, donc un record dont le masque ne porte QU'UN composant donne sa largeur. Mesure
// faite le 2026-09-01 : le resultat est du BRUIT. Les pics valent 3 a 8 fois le plancher (contre
// 40 attendus pour un vrai signal), et les largeurs sorties sont incoherentes entre elles.
// La cause est identifiee : `worldObjectHeaderAt` n'est pas assez selectif pour designer LA fin
// d'un record sur une fenetre de 320 bits — il y a plusieurs positions candidates par record.
//
// # L'ORACLE QUI MARCHE : LE CHAINAGE, VENTILE PAR COMPOSANT PRESENT
//
// Plutot que de demander « ou finit ce record », on demande « ce record finit-il JUSTE » — un
// booleen, pas une position. Et on le ventile : d'un cote les records qui portent le composant,
// de l'autre ceux qui ne le portent pas. Une largeur juste ne change pas le taux ; une largeur
// fausse l'effondre des que le composant apparait. C'est le meme oracle qui a servi sur
// l'image-cle, applique cette fois au chemin ou la grammaire s'applique vraiment.
//
// LES TROIS BALAYAGES D'HYPOTHESE (garde par composant, bit de presence, controle sur `ti=13`)
// vivent dans `objectif_ti11_hypotheses_test.go`. Le bilan ci-dessous les couvre tous.
//
// # CE QUE CES CINQ INSTRUMENTS ONT ETABLI (2026-09-01) — A LIRE AVANT D'EN ECRIRE UN SIXIEME
//
// 1. LE PLANCHER. `worldObjectHeaderAt` passe a **3 %** des positions arbitraires
//    (`TestObjectifTi11DeltaTemoin`). Tous les taux de chainage de ce chantier se lisent contre
//    ce chiffre, et aucun ne l'avait jamais fait.
// 2. LE GARDE PAR COMPOSANT DU MODE FILM : REFUTE. L'armer fait tomber le chainage de 29,3 %
//    a 2,4 %, soit le plancher (`TestObjectifTi11DeltaGarde`).
// 3. LE BIT DE PRESENCE PAR COMPOSANT : REFUTE dans ses deux lectures — prefixe 4,5 %,
//    garde 3,1 % (`TestObjectifTi11DeltaPresence`).
// 4. LES LARGEURS SONT JUSTES. Les LECTEURS du jeu (et non les ecrivains que le premier portage
//    avait suivis) confirment i1, i3, i12, i13, i16 et i32 au bit pres.
// 5. LE VRAI FACTEUR EST LA TAILLE DE LA BANDE D'ANCRAGE, et le controle sur `ti=13` le prouve
//    (`TestObjectifTi11DeltaControleTi13`) :
//
//	archetype   bande COMBLEE (production)   bande OBSERVEE
//	ti=11        5,6 % sur 1 616 281          29,3 % sur 19 666
//	ti=13        6,3 % sur   278 670          43,7 % sur 27 409
//
//    Et par film, le chainage suit le nombre de slots : 20 slots donnent 77 % (KOTH, ti=13),
//    1 704 slots donnent 2,6 %.
//
// # LA CONSEQUENCE LA PLUS UTILE : LE REPERE DE « 87 A 99 % » N'EST PAS REPRODUCTIBLE
//
// Ce chantier se comparait a un chiffre venu de la documentation du lot C-bis. Le meme
// instrument, sur le meme archetype `ti=13` et sur ses PROPRES modes, plafonne a **77 %**
// (KOTH) et rend 32 a 56 % ailleurs. Le meilleur film de `ti=11` est a **64,9 %**. `ti=11` est
// donc DEJA dans le regime de l'archetype de reference — ce qui manquait n'etait pas un
// composant, c'etait un repere honnete.
//
// # DECOUVERTE HORS PERIMETRE, NOTEE ET NON TRAITEE
//
// La bande observee ferait passer le balayage de PRODUCTION de `ti=13`
// (`ScanFilmManagedProperties`, qui comble) de 2,6 % a 32,2 % de chainage sur un film CTF, et de
// 47,6 % a 77,0 % sur un KOTH — dix fois moins de faux records. C'est une amelioration reelle
// d'un chemin qui a des consommateurs en production : elle se decide, elle ne se glisse pas dans
// un commit de recherche.
//
// REGIME : garde `ASSAUT_CACHE`. Aucune base, aucun reseau, sentinelle memoire armee.
//
//	$env:ASSAUT_CACHE="C:/.../data/cache"
//	go test ./internal/analysis/filmdec/ -run ObjectifTi11Delta -v -timeout 60m

import (
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/filmproc"
)

// ti11DeltaBilan compte, par composant, les marches abouties et celles qui chainent.
type ti11DeltaBilan struct {
	avec, avecChaine [ti11Composants]int
	sans, sansChaine [ti11Composants]int
	marches, chaines int
	// tailles[n] = marches abouties portant n composants ; tailleschaine[n] celles qui chainent.
	tailles, taillesChaine [8]int
}

// TestObjectifTi11DeltaChainage ventile le chainage des records delta par composant present.
func TestObjectifTi11DeltaChainage(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandee : ASSAUT_CACHE requis")
	}
	g := filmproc.Arm("TestObjectifTi11DeltaChainage", filmproc.MeasureLimitGiB, func(peak uint64) {
		t.Errorf("PLAFOND MEMOIRE DEPASSE (%.2f Gio) — mesure interrompue", float64(peak)/(1<<30))
	})
	defer func() { g.Disarm() }()

	var b ti11DeltaBilan
	var arch Archetype
	for _, f := range ti11Corpus {
		dir := filepath.Join(cache, "film_chunks", f.id)
		n := CountFilmChunks(dir)
		if n == 0 {
			continue
		}
		a, _, err := objectiveArchetype(dir)
		if err != nil {
			continue
		}
		arch = a
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
				if pk.Type == PacketTypeDelta {
					ti11DeltaPayload(pk.Payload(data), band, arch, &b)
				}
			}
		}
	}

	t.Logf("########## DELTA — %d marche(s) aboutie(s), %d chainee(s) (%.1f %%)",
		b.marches, b.chaines, ti11Part(b.chaines, b.marches))
	t.Logf("PAR NOMBRE DE COMPOSANTS :")
	for n := 1; n < len(b.tailles); n++ {
		if b.tailles[n] == 0 {
			continue
		}
		t.Logf("   %d composant(s) : %5d marche(s), %5.1f %% chainees",
			n, b.tailles[n], ti11Part(b.taillesChaine[n], b.tailles[n]))
	}
	t.Logf("PAR COMPOSANT PRESENT — l'ecart entre les deux colonnes DESIGNE la largeur fausse :")
	for i := 0; i < ti11Composants; i++ {
		if b.avec[i] == 0 {
			continue
		}
		t.Logf("   i%-3d%-30s PRESENT : %5d marches, %5.1f %% chainees   |   ABSENT : %5d, %5.1f %%",
			i, ti11Nom(arch, i), b.avec[i], ti11Part(b.avecChaine[i], b.avec[i]),
			b.sans[i], ti11Part(b.sansChaine[i], b.sans[i]))
	}
}

// ti11DeltaPayload marche les records delta d'UN payload et range le chainage.
func ti11DeltaPayload(pay []byte, band map[uint32]bool, arch Archetype, b *ti11DeltaBilan) {
	total := len(pay) * 8
	limit := total - (worldObjectHeaderBits + worldObjectIndexBits)
	for p := 0; p <= limit; p++ {
		rec, ok := matchWorldObjectRecord(pay, p, band)
		if !ok || !ti11IdxDansDomaine(rec.Idx, len(arch.Components)) {
			continue
		}
		fin, done := ti11MarcheRecord(pay, rec, arch, total)
		p = rec.After
		if !done {
			continue
		}
		b.marches++
		chaine := worldObjectHeaderAt(pay, fin)
		if chaine {
			b.chaines++
		}
		if n := len(rec.Idx); n < len(b.tailles) {
			b.tailles[n]++
			if chaine {
				b.taillesChaine[n]++
			}
		}
		presents := map[int]bool{}
		for _, id := range rec.Idx {
			if id < ti11Composants {
				presents[id] = true
			}
		}
		for i := 0; i < ti11Composants; i++ {
			switch {
			case presents[i] && chaine:
				b.avec[i]++
				b.avecChaine[i]++
			case presents[i]:
				b.avec[i]++
			case chaine:
				b.sans[i]++
				b.sansChaine[i]++
			default:
				b.sans[i]++
			}
		}
	}
}

// ti11MarcheRecord rejoue les composants d'un record delta avec les desers DE PRODUCTION.
func ti11MarcheRecord(pay []byte, rec WorldObjectRecord, arch Archetype, total int) (int, bool) {
	at := rec.After
	for _, id := range rec.Idx {
		name := arch.component(id)
		if name == "" || at > total {
			return at, false
		}
		br := NewBitReader(pay)
		br.SetBitPos(at)
		_, _, ported := consumeByName(br, name, ObjectiveTypeIndex, arch.Level(id))
		if !ported || br.BitPos() > total {
			return at, false
		}
		// LE GARDE PAR COMPOSANT DU MODE FILM, s'il est arme. Il est appele ICI et non laisse
		// a `traverseComponentLoop` parce que cette marche appelle `consumeByName` en direct —
		// et un premier balayage l'avait OUBLIE, rendant deux colonnes rigoureusement
		// identiques. Un A/B qui ne bouge pas d'un iota ne mesure rien : il dit qu'on a teste
		// une bascule que le code ne lit pas.
		consumeCorruptionCheck(br)
		at = br.BitPos()
	}
	return at, true
}

// ti11IdxDansDomaine rejette un masque citant un composant hors de l'archetype.
func ti11IdxDansDomaine(idx []int, n int) bool {
	for _, id := range idx {
		if id < 0 || id >= n {
			return false
		}
	}
	return true
}
