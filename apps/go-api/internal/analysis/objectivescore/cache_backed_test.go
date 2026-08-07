package objectivescore

// cache_backed_test.go — la VÉRITÉ TERRAIN d'un film réel, opposée au décodeur.
//
// Film `7344d24f` (Strongholds:Arena), finals DB 193-112. C'est le film sur lequel la
// calibration Strongholds a été établie ; c'est donc lui qui doit continuer à la vérifier.
//
// CE TEST A ÉTÉ MORT PENDANT DES MOIS. Il tenait sa racine de cache dans une constante
// pointant un poste disparu, et son `t.Skip` faisait passer cette mort pour une absence de
// cache. Il lit désormais `FILM_CACHE_ROOT` et ne se tait plus que si la variable est
// absente — voir `filmcache_test.go`, qui porte le détail des deux régimes.
//
// CE QU'IL VERROUILLE, et qu'aucune fixture synthétique ne peut verrouiller : que le token
// 0x7B6 se trouve à sa fenêtre dans des octets que personne n'a écrits pour lui, et que la
// séquence brute lue à `token+24` est croissante sur un vrai match. Les fixtures de
// `decode_test.go`, elles, sont construites AVEC les constantes du décodeur : les décaler
// laisse ce fichier-là vert.

import "testing"

const (
	shGTShort   = "7344d24f" // Strongholds:Arena, vérité terrain de la calibration
	shGTFinalT0 = 193
	shGTFinalT1 = 112

	// shGTBrut0Fin / shGTBrut1Fin : les valeurs BRUTES lues en fin de match, avant toute
	// calibration. Ce ne sont pas des scores — c'est tout leur intérêt.
	//
	// L'en-tête du paquet établit, cross-validation du 2026-06-03 à l'appui, que la brute
	// team0 PLAFONNE à ~50 (des matchs finissant 169, 193 et 178 rendent tous 50) et que la
	// brute team1 rend des marqueurs structurels de fin de film, identiques d'un match à
	// l'autre. Le golden `films_reels` le montre sur les deux Strongholds du corpus : finals
	// API 193-112 et 200-94, brutes 50 et 32 dans les DEUX cas.
	//
	// Les figer ici est ce qui rend ce test sensible à `shOffTeam0`/`shOffTeam1`. Les
	// assertions « dernière frame == final » ne peuvent pas l'être : `calibrateByFinal` les
	// satisfait par construction.
	shGTBrut0Fin = 50
	shGTBrut1Fin = 32
)

func TestDecodeStrongholds_CacheBacked_7344d24f(t *testing.T) {
	chunks := chargerChunksFilm(t, racineCacheFilm(t), shGTShort)

	frames := DecodeStrongholds(chunks, shGTFinalT0, shGTFinalT1)
	if len(frames) == 0 {
		t.Fatalf("aucun ScoreFrame décodé sur %s (chunks=%d)", shGTShort, len(chunks))
	}

	// team0 monotone croissant.
	for i := 1; i < len(frames); i++ {
		if frames[i].Team0 < frames[i-1].Team0 {
			t.Fatalf("Team0 non monotone à i=%d : %d < %d", i, frames[i].Team0, frames[i-1].Team0)
		}
	}
	// Dernière frame == final DB exact (calibration).
	if last := frames[len(frames)-1].Team0; last != shGTFinalT0 {
		t.Errorf("dernière frame Team0 = %d, want %d (final DB)", last, shGTFinalT0)
	}
	if last := frames[len(frames)-1].Team1; last != shGTFinalT1 {
		t.Errorf("dernière frame Team1 = %d, want %d (final DB)", last, shGTFinalT1)
	}
	// La timeline démarre à 0 (score initial).
	if frames[0].Team0 != 0 {
		t.Errorf("première frame Team0 = %d, want 0", frames[0].Team0)
	}
	t.Logf("7344d24f : %d frames, team0 %d->%d, team1 %d->%d",
		len(frames), frames[0].Team0, frames[len(frames)-1].Team0,
		frames[0].Team1, frames[len(frames)-1].Team1)
}

// TestDecodeStrongholds_CacheBacked_AncrageBrut : le complément INDISPENSABLE du test
// ci-dessus, et la raison pour laquelle celui-ci ne suffisait pas.
//
// `calibrateByFinal` remet la DERNIÈRE frame exactement sur le final DB, quelle que soit la
// valeur brute lue. Les trois assertions « dernière frame == final » du test précédent sont
// donc STRUCTURELLEMENT INSENSIBLES à la position de bit : décaler `shOffTeam0` les laisse
// vertes tant que la séquence brute reste croissante. Ce test-ci regarde AVANT la
// calibration — l'ancre elle-même et les valeurs brutes.
func TestDecodeStrongholds_CacheBacked_AncrageBrut(t *testing.T) {
	chunks := chargerChunksFilm(t, racineCacheFilm(t), shGTShort)

	// L'ancre : combien de chunks type-2 portent le token, et à quelles positions de bit.
	ancres, type2 := 0, 0
	for _, c := range chunks {
		if c.ChunkType != packetType2 {
			continue
		}
		type2++
		if p, tb := anchoredPayload(c); p != nil && tb >= 0 {
			ancres++
		}
	}
	if type2 == 0 {
		t.Fatalf("%s : aucun chunk type-2 dans le manifest", shGTShort)
	}
	// L'ancre doit tenir sur la QUASI-TOTALITÉ des chunks type-2. Le seuil est bas parce
	// qu'un premier chunk de mise en place peut ne pas porter le bloc de score ; il est
	// non nul parce qu'une fenêtre de recherche fausse rendrait zéro ancre.
	if ancres*10 < type2*8 {
		t.Fatalf("%s : token 0x%X trouvé sur %d chunk(s) type-2 sur %d — la fenêtre de "+
			"recherche [%d,%d) ou le token lui-même ne correspondent plus aux octets réels",
			shGTShort, scoreToken, ancres, type2, tokenWinLo, tokenWinHi)
	}

	// Les valeurs BRUTES, avant toute calibration.
	_, raw0, raw1 := collectStrongholdsRaw(chunks)
	if len(raw0) != ancres {
		t.Fatalf("collectStrongholdsRaw rend %d valeur(s) pour %d chunk(s) ancré(s)", len(raw0), ancres)
	}
	if raw0[0] != 0 {
		t.Errorf("brut team0 de la première frame = %d, want 0 (un match commence à zéro)", raw0[0])
	}
	for i := 1; i < len(raw0); i++ {
		if raw0[i] < raw0[i-1] {
			t.Fatalf("brut team0 non monotone à i=%d : %d < %d — la séquence lue à token+%d "+
				"n'est plus un score qui monte", i, raw0[i], raw0[i-1], shOffTeam0)
		}
	}
	// Non trivial : une séquence brute plate passerait toutes les assertions de monotonie
	// et se calibrerait en zéros, ce qui est indistinguable d'un décodeur cassé.
	if dernier := raw0[len(raw0)-1]; dernier <= 0 {
		t.Fatalf("brut team0 finit à %d : la séquence est plate, le décodeur ne lit plus rien "+
			"d'exploitable à token+%d", dernier, shOffTeam0)
	}
	// Les valeurs brutes attendues, à l'unité près. Voir le commentaire des constantes :
	// c'est ce couple qui répond des deux offsets, et rien d'autre dans ce fichier.
	if got := raw0[len(raw0)-1]; got != shGTBrut0Fin {
		t.Errorf("brut team0 en fin de match = %d, want %d (plafond documenté du champ lu à "+
			"token+%d) — un offset a bougé, ou le champ n'est plus le même", got, shGTBrut0Fin, shOffTeam0)
	}
	if got := raw1[len(raw1)-1]; got != shGTBrut1Fin {
		t.Errorf("brut team1 en fin de match = %d, want %d (marqueur structurel lu à token+%d)",
			got, shGTBrut1Fin, shOffTeam1)
	}
	t.Logf("%s : %d/%d chunks type-2 ancrés, brut team0 %d->%d, brut team1 %d->%d",
		shGTShort, ancres, type2, raw0[0], raw0[len(raw0)-1], raw1[0], raw1[len(raw1)-1])
}
