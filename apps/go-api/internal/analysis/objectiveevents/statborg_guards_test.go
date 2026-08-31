package objectiveevents

import "testing"

// statborg_guards_test.go — LES CONTROLES NEGATIFS DES GARDES D'ANCRAGE (revue R1, 2026-08-18).
//
// POURQUOI CE FICHIER EXISTE. Les tests d'A.1.0 prouvaient que la grammaire relachee LIT ce
// qu'elle doit lire (manche 2, forme dense, plafond memoire). Aucun ne prouvait qu'elle REFUSE
// ce qu'elle doit refuser — or c'est exactement ce que le relachement met en jeu : la contrainte
// « les deux en-tetes de 5 bits sont nuls » a saute, et ce sont deux autres contraintes qui la
// remplacent. Un test qui n'echouerait pas si on les inversait ne prouve rien.
//
// Chaque test ci-dessous ECHOUE si la garde qu'il vise est retiree.

// setBitsBE ecrit n bits big-endian a bitPos dans une COPIE de data. C'est l'inverse exact de
// readBitsBE, et il ne sert qu'a fabriquer des vecteurs NEGATIFS a partir de vecteurs reels :
// on part d'un enregistrement qui se decode, et on casse UNE contrainte a la fois.
func setBitsBE(data []byte, bitPos, n int, v uint64) []byte {
	out := append([]byte(nil), data...)
	for i := 0; i < n; i++ {
		bit := (v >> uint(n-1-i)) & 1
		p := bitPos + i
		mask := byte(1) << uint(7-p%8)
		if bit == 1 {
			out[p/8] |= mask
		} else {
			out[p/8] &^= mask
		}
	}
	return out
}

// TestDecodeComponentsRefuseUnCoupleDepareille — LA PREMIERE GARDE.
//
// Les deux en-tetes de 5 bits portent le MEME numero de manche : ils sont redondants dans le
// format, et c'est cette redondance qui remplace la contrainte « nuls » comme filtre
// anti-faux-positifs. Un couple depareille est un ancrage fortuit.
func TestDecodeComponentsRefuseUnCoupleDepareille(t *testing.T) {
	_, idx, at, ok := matchRecordHeader(vecRound0.data, vecRound0.bits)
	if !ok {
		t.Fatal("le vecteur de reference ne s'ancre plus — revoir les vecteurs avant ce test")
	}
	if comps, _ := decodeComponents(vecRound0.data, at, idx); len(comps) == 0 {
		t.Fatal("le vecteur de reference ne se decode plus")
	}
	// Le SECOND en-tete passe de 0 a 1 : les deux ne disent plus la meme manche.
	casse := setBitsBE(vecRound0.data, at+statHdrBits, statHdrBits, 1)
	if comps, _ := decodeComponents(casse, at, idx); len(comps) != 0 {
		t.Errorf("un couple d'en-tetes DEPAREILLE (0 puis 1) a ete accepte : %d composant(s) — "+
			"la garde qui remplace « en-tetes nuls » ne filtre plus rien", len(comps))
	}
}

// TestDecodeComponentsRefuseUneMancheHorsBorne — LA SECONDE GARDE.
//
// Le numero de manche est borne (statMaxRound) : huit manches sont au-dela de tout format
// observe, et la borne conserve deux bits de contrainte sur chacun des deux en-tetes. Sans elle,
// l'ancrage laisserait passer 151 faux positifs par film (mesure d'A.1.0).
func TestDecodeComponentsRefuseUneMancheHorsBorne(t *testing.T) {
	_, idx, at, ok := matchRecordHeader(vecRound0.data, vecRound0.bits)
	if !ok {
		t.Fatal("le vecteur de reference ne s'ancre plus")
	}
	horsBorne := uint64(statMaxRound + 1)
	casse := setBitsBE(vecRound0.data, at, statHdrBits, horsBorne)
	casse = setBitsBE(casse, at+statHdrBits, statHdrBits, horsBorne)
	// Les deux en-tetes CONCORDENT : seule la borne peut refuser ce vecteur.
	if comps, round := decodeComponents(casse, at, idx); len(comps) != 0 {
		t.Errorf("une manche %d (borne %d) a ete acceptee : %d composant(s), round=%d",
			horsBorne, statMaxRound, len(comps), round)
	}
}

// TestRealRoundsRefuseUneValeurHorsDomaine — GARDE ANTI-MANCHE-FANTOME n 1 : la borne de domaine.
//
// Aucun mode Halo Infinite ne fait marquer plus de 250 points dans une manche. Sur le CTF
// `53ce4390`, un point isole a 2 104 suffisait a faire passer une manche fantome pour reelle et
// portait le score d'equipe de 1 a 2 104.
func TestRealRoundsRefuseUneValeurHorsDomaine(t *testing.T) {
	recs := modeSerie(6, 1, 1_000, 2_104, 2_105, 2_106)
	if real := RealRounds(recs); real[1] {
		t.Errorf("une manche batie sur des valeurs de 2 104 a ete tenue pour reelle : %v — "+
			"la borne de domaine (%d) ne filtre plus", real, statMaxModeScore)
	}
	// Temoin : les MEMES emissions sous la borne font, elles, une manche reelle.
	if real := RealRounds(modeSerie(6, 1, 1_000, 1, 2, 3)); !real[1] {
		t.Error("le temoin sous la borne n'est pas retenu : le test ne prouverait rien")
	}
}

// TestRealRoundsRefuseUneMancheIsolee — GARDE n 2 : la contiguite.
//
// Les manches se jouent DANS L'ORDRE. Une « manche 5 » sans les manches 1 a 4 est un ancrage
// fortuit, quel que soit son comptage.
func TestRealRoundsRefuseUneMancheIsolee(t *testing.T) {
	real := RealRounds(modeSerie(6, 5, 1_000, 1, 2, 3))
	if real[5] {
		t.Errorf("une manche 5 sans les precedentes a ete retenue : %v", real)
	}
	if len(real) != 1 || !real[0] {
		t.Errorf("attendu le repli sur la seule manche 1, obtenu %v", real)
	}
}

// TestRealRoundsRefuseUneEmissionIsolee — GARDE n 3 : le seuil de coherence.
//
// Une manche REELLE tire une suite croissante d'au moins statMinRoundRun emissions ; un ancrage
// fortuit arrive ISOLE. Compter les enregistrements bruts ne suffisait pas.
func TestRealRoundsRefuseUneEmissionIsolee(t *testing.T) {
	var recs []StatRecord
	recs = append(recs, modeSerie(6, 0, 1_000, 1, 2, 3)...)
	recs = append(recs, modeSerie(6, 1, 9_000, 7)...) // une seule emission : pas une manche
	real := RealRounds(recs)
	if real[1] {
		t.Errorf("une manche d'UNE emission a ete retenue : %v (seuil %d)", real, statMinRoundRun)
	}
	if !real[0] {
		t.Error("la manche coherente n'est plus retenue : le test ne prouverait rien")
	}
}

// TestRealRoundsGardeLesManchesApresUneManche0Courte — LA RESERVE DU RELECTEUR, ET ELLE ETAIT
// FONDEE (revue R1, P1-7).
//
// La contiguite sortait au PREMIER trou. Une premiere manche trop courte pour tirer trois
// emissions coherentes — un camp qui s'effondre en quelques secondes — faisait donc perdre
// TOUTES les manches suivantes, y compris completes : le match retombait sur la seule manche 1
// de repli, et son total valait zero.
func TestRealRoundsGardeLesManchesApresUneManche0Courte(t *testing.T) {
	var recs []StatRecord
	recs = append(recs, modeSerie(6, 0, 1_000, 2)...) // manche 0 : une seule emission
	recs = append(recs, modeSerie(6, 1, 20_000, 1, 2, 3)...)
	recs = append(recs, modeSerie(6, 2, 40_000, 1, 2, 3)...)

	real := RealRounds(recs)
	for _, r := range []int{1, 2} {
		if !real[r] {
			t.Errorf("manche %d PERDUE alors qu'elle est complete : %v", r, real)
		}
	}
	if !real[0] {
		t.Errorf("manche 0 ecartee alors qu'une manche coherente la suit : %v", real)
	}
	if real[3] {
		t.Errorf("manche 3 retenue alors que rien ne la suit : %v", real)
	}
}

// modeSerie construit les emissions de score de MODE d'un slot pour une manche.
func modeSerie(slot, round, startMS int, values ...int64) []StatRecord {
	out := make([]StatRecord, 0, len(values))
	for i, v := range values {
		out = append(out, StatRecord{
			TimeMS: startMS + i*1_000, Slot: slot, Round: round,
			Comps: map[int]StatValue{modeScoreComp: {A: v}},
		})
	}
	return out
}

// joueurSerie fabrique une manche MATERIELLE : n enregistrements repartis sur les 8 slots de
// JOUEUR (10, 12, ... 24), un point de score de mode a la fin pour le slot qui marque.
//
// Le score de mode ne bouge qu'UNE fois — la forme exacte d'une manche d'Assaut One Bomb, celle
// que le critere de suite coherente ne peut pas admettre.
func joueurSerie(round, startMS, n int, scoreFinal int64) []StatRecord {
	out := make([]StatRecord, 0, n)
	for i := 0; i < n; i++ {
		slot := 10 + 2*(i%8)
		out = append(out, StatRecord{
			TimeMS: startMS + i*100, Slot: slot, Round: round,
			Comps: map[int]StatValue{modeScoreComp: {A: 0}},
		})
	}
	if n > 0 && scoreFinal > 0 {
		out[n-1].Comps = map[int]StatValue{modeScoreComp: {A: scoreFinal}}
	}
	return out
}

// TestRealRoundsAdmetUneMancheAUneSeuleEmissionDeScore — LE SECOND CRITERE D'ADMISSION.
//
// Une manche d'Assaut One Bomb porte au plus UNE emission de score (un point de mode = une
// explosion), donc sa plus longue suite strictement croissante vaut 2 : sous statMinRoundRun.
// Avant le second critere, seule la manche 0 survivait et 8 explosions sur 11 etaient perdues
// sur les 3 films One Bomb du corpus.
func TestRealRoundsAdmetUneMancheAUneSeuleEmissionDeScore(t *testing.T) {
	var recs []StatRecord
	for round := 0; round < 4; round++ {
		recs = append(recs, joueurSerie(round, round*100_000, 200, 1)...)
	}
	real := RealRounds(recs)
	for round := 0; round < 4; round++ {
		if !real[round] {
			t.Errorf("manche %d refusee alors qu'elle est MATERIELLE (200 enregistrements "+
				"joueur) : %v", round, real)
		}
	}
}

// TestRealRoundsRefuseUnAncrageMalgreLaPart — LE PLANCHER ABSOLU du second critere.
//
// Sur un film tres pauvre, la part seule passerait : un enregistrement contre trois fait 33 %.
// Le plancher statMinRoundRecords ferme cette porte.
func TestRealRoundsRefuseUnAncrageMalgreLaPart(t *testing.T) {
	var recs []StatRecord
	recs = append(recs, joueurSerie(0, 1_000, 3, 1)...)
	recs = append(recs, joueurSerie(1, 9_000, 1, 0)...) // 1/3 = 33 % de la part, mais 1 << plancher
	if real := RealRounds(recs); real[1] {
		t.Errorf("un ancrage isole a passe la part (33 %%) : %v — le plancher (%d) ne filtre plus",
			real, statMinRoundRecords)
	}
}

// TestRealRoundsRefuseUneMancheTropMaigreEnPart — LA PART, l'autre moitie du second critere.
//
// Le plus gros ancrage fortuit MESURE du corpus libre est `bfcd1175` manche 6 : 18
// enregistrements joueur contre 308 a la manche 0, soit 5,84 % — au-dessus du plancher
// une fois double, et pourtant un film de Slayer, mode qui n'a pas de manche. Seule la PART
// l'ecarte. Le vecteur reproduit ce rapport a l'echelle : 30 contre 600 = 5 %.
func TestRealRoundsRefuseUneMancheTropMaigreEnPart(t *testing.T) {
	var recs []StatRecord
	recs = append(recs, joueurSerie(0, 1_000, 600, 1)...)
	recs = append(recs, joueurSerie(1, 900_000, 30, 0)...) // 5 % : sous statMinRoundRecordShare
	if real := RealRounds(recs); real[1] {
		t.Errorf("une manche a 5 %% de la plus fournie a ete retenue : %v — la part (%d %%) ne "+
			"filtre plus", real, statMinRoundRecordShare)
	}
	// Temoin : la MEME manche a 21 %% (le plancher mesure d'une manche reelle) passe.
	var temoin []StatRecord
	temoin = append(temoin, joueurSerie(0, 1_000, 600, 1)...)
	temoin = append(temoin, joueurSerie(1, 900_000, 126, 0)...)
	if real := RealRounds(temoin); !real[1] {
		t.Errorf("le temoin a 21 %% est refuse : %v — le test ne prouverait rien", real)
	}
}
