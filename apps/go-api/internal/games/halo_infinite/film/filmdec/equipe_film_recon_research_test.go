package filmdec

// equipe_film_recon_research_test.go — PHASE 3, RECENSEMENT (pas de verdict).
//
// POURQUOI CE FICHIER EXISTE. La phase 2 a ferme par la negative la question « l'equipe est
// dans l'enregistrement de slot de chunk_00 » : huit des neuf champs courts sont CONSTANTS sur
// tout le corpus. Le consommateur, lui, nomme l'equipe ailleurs — dans la trame d'etat
// (paquets de type 2), comme composant `managed-player-team-designator-component` de
// l'archetype ti=9, lu par `FUN_140f581e8` sur 4 bits (`FUN_1407ef804` :
// `ADD dword ptr [RCX+0x2c],0x4` puis `DEC R9B`, donc la valeur STOCKEE vaut le brut MOINS 1).
//
// CE QUE CET INSTRUMENT MESURE, ET RIEN DE PLUS : par film, combien de records ti=9 la table
// d'image-cle porte, quels slots ils occupent, quelle longueur ils ont, et quelle valeur de
// 4 bits sort a une liste de decalages d'en-tete CANDIDATS. Aucun seuil, aucune conclusion :
// les controles positif et negatif sont dans `equipe_film_oracle_research_test.go`, ecrits
// avant la mesure de verdict.
//
// Garde CHUNK00_FILMS (chemins Windows `C:/...`, separes par `;`). Aucun code de production
// touche.

import (
	"fmt"
	"path/filepath"
	"sort"
	"testing"
)

// equipeTI est l'index d'archetype « managed-player » (le profil du joueur) dont le composant
// i0 est `managed-player-team-designator-component`. Lu dans `testdata/ecs_table.tsv:228` et
// re-verifie ici contre le registre du film lui-meme (cf. TestEquipeFilmArchetype).
const equipeTI = 9

// equipeDesignatorBits est la largeur du champ, lue dans le code de son lecteur
// (`FUN_1407ef804`), pas supposee.
const equipeDesignatorBits = 4

// equipeEntetesCandidates sont les largeurs d'en-tete par entite que le dossier connait :
// 64 = la lecture historique du depot (`keyframeHeaderBits`), 108 = celle de `FUN_142e2bfd0`
// (`keyframeFullStateHeaderBits`), 47 = la valeur que le fork chasewoodhams annonce pour ti=9.
// Ce sont des CANDIDATS a departager par la mesure, pas des faits.
func equipeEntetesCandidates() []int { return []int{47, 64, 108} }

// equipeRecord est un record ti=9 de la table d'image-cle d'un film.
type equipeRecord struct {
	Slot, Bit, LenBits int
}

// equipePaquet regroupe les records d'un archetype trouves dans UN paquet de type 2, avec le
// payload dans lequel leurs positions de bit ont un sens.
type equipePaquet struct {
	Chunk, Index int
	Pay          []byte
	Recs         []equipeRecord
}

// equipePaquetsArch rend, pour un repertoire de film, tous les paquets de type 2 portant au
// moins un record de l'archetype `ti`, avec ces records dans l'ordre de la table.
func equipePaquetsArch(dir string, ti int) ([]equipePaquet, error) {
	n := CountFilmChunks(dir)
	if n == 0 {
		return nil, fmt.Errorf("aucun chunk film dans %s", dir)
	}
	var out []equipePaquet
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
			var recs []equipeRecord
			for _, s := range KeyframeRecordSpans(pay) {
				if s.TI == ti {
					recs = append(recs, equipeRecord{Slot: s.Slot, Bit: s.BitStart, LenBits: s.LengthBits})
				}
			}
			if len(recs) > 0 {
				out = append(out, equipePaquet{Chunk: c, Index: pk.Index, Pay: pay, Recs: recs})
			}
		}
	}
	return out, nil
}

// equipePaquetsModaux ne garde que les paquets dont le NOMBRE de records ti=9 vaut le mode.
//
// REGLE ECRITE AVANT LA MESURE, et ce n'est pas un filtre de commodite : un joueur qui arrive
// ou part en cours de match change le nombre d'entites `managed-player` repliquees, donc le
// cardinal du vecteur. Comparer un vecteur de 8 a un vecteur de 7 n'a pas de sens ; le mode
// est la composition STABLE du match, et c'est elle que l'oracle externe decrit. Les paquets
// ecartes sont comptes et publies.
func equipePaquetsModaux(pqs []equipePaquet) (gardes []equipePaquet, ecartes int) {
	comptes := map[int]int{}
	for _, pq := range pqs {
		comptes[len(pq.Recs)]++
	}
	mode, best := 0, -1
	for n, c := range comptes {
		if c > best || (c == best && n > mode) {
			mode, best = n, c
		}
	}
	for _, pq := range pqs {
		if len(pq.Recs) == mode {
			gardes = append(gardes, pq)
		} else {
			ecartes++
		}
	}
	return gardes, ecartes
}

// equipeLireDesignateur lit le champ de 4 bits a `bit`, et rend le BRUT ainsi que la valeur
// stockee par le jeu (brut - 1 ; -1 = aucune equipe).
func equipeLireDesignateur(pay []byte, bit int) (brut int, stocke int) {
	brut = int(kfReadBits(pay, bit, equipeDesignatorBits))
	return brut, brut - 1
}

// TestEquipeFilmArchetype verifie sur pieces, dans le registre du FILM (pas dans une table du
// depot), que l'archetype ti=9 porte bien `managed-player-team-designator-component` en i0.
// C'est le controle d'identite de source (methode, erreur D) : sans lui, tout ce qui suit
// pourrait lire un autre archetype.
func TestEquipeFilmArchetype(t *testing.T) {
	dirs := chunk00Films(t, "CHUNK00_FILMS")
	for _, dir := range dirs {
		_, data := readChunk00(t, dir)
		reg := parseRegistry(data)
		arch, ok := reg.Archetype(equipeTI)
		if !ok {
			t.Errorf("%s : archetype ti=%d absent du registre", filepath.Base(dir), equipeTI)
			continue
		}
		i0 := ""
		if len(arch.Components) > 0 {
			i0 = arch.Components[0]
		}
		t.Logf("%s : ti=%d porte %d composants, i0 = %q",
			filepath.Base(dir), equipeTI, len(arch.Components), i0)
		if i0 != "managed-player-team-designator-component" {
			t.Errorf("%s : i0 de ti=%d vaut %q, attendu managed-player-team-designator-component",
				filepath.Base(dir), equipeTI, i0)
		}
	}
}

// equipeReconPaquetsMax borne l'affichage du recensement : les trois premieres images-cles
// d'un film suffisent a voir la forme, le reste est du volume de journal.
const equipeReconPaquetsMax = 3

// TestEquipeFilmRecon recense les records ti=9 par film et affiche, pour chaque largeur
// d'en-tete candidate, la suite des designateurs lus. RECENSEMENT : aucun seuil.
func TestEquipeFilmRecon(t *testing.T) {
	dirs := chunk00Films(t, "CHUNK00_FILMS")
	for _, dir := range dirs {
		pqs, err := equipePaquetsArch(dir, equipeTI)
		if err != nil {
			t.Logf("%s : %v", filepath.Base(dir), err)
			continue
		}
		t.Logf("%s : %d paquets type-2 portant ti=%d", filepath.Base(dir), len(pqs), equipeTI)
		for i, pq := range pqs {
			if i >= equipeReconPaquetsMax {
				break
			}
			lens := map[int]int{}
			slots := make([]int, 0, len(pq.Recs))
			for _, r := range pq.Recs {
				lens[r.LenBits]++
				slots = append(slots, r.Slot)
			}
			t.Logf("  chunk_%02d paquet #%d : %d records ; slots %v ; longueurs %s",
				pq.Chunk, pq.Index, len(pq.Recs), slots, equipeHistoTexte(lens))
			for _, h := range equipeEntetesCandidates() {
				var bruts []int
				for _, r := range pq.Recs {
					b, _ := equipeLireDesignateur(pq.Pay, r.Bit+h)
					bruts = append(bruts, b)
				}
				t.Logf("      en-tete %3d bits : bruts %v", h, bruts)
			}
		}
	}
}

// TestEquipeFilmCensusTI recense la distribution des archetypes de la table d'image-cle : sans
// elle, un compte de records ti=9 n'a pas d'echelle de comparaison.
func TestEquipeFilmCensusTI(t *testing.T) {
	dirs := chunk00Films(t, "CHUNK00_FILMS")
	for _, dir := range dirs {
		chunk, pk, cn, err := FirstPacketOfType(dir, PacketTypeKeyframe)
		if err != nil {
			t.Logf("%s : %v", filepath.Base(dir), err)
			continue
		}
		byTI := map[int]int{}
		for _, s := range KeyframeRecordSpans(pk.Payload(chunk)) {
			byTI[s.TI]++
		}
		t.Logf("%s (chunk_%02d) : %d archetypes distincts ; %s",
			filepath.Base(dir), cn, len(byTI), equipeHistoTexte(byTI))
	}
}

// TestEquipeFilmCensusTousPaquets recense les archetypes sur TOUS les paquets de type 2 de
// TOUS les chunks d'un film, et non sur le seul premier. Sans cela, « 0 record ti=9 » ne
// distingue pas « absent du film » de « absent de la premiere image-cle ».
func TestEquipeFilmCensusTousPaquets(t *testing.T) {
	dirs := chunk00Films(t, "CHUNK00_FILMS")
	for _, dir := range dirs {
		byTI := map[int]int{}
		paquets, slotMax, records := 0, 0, 0
		for c := 1; c <= CountFilmChunks(dir); c++ {
			data, err := ReadFilmChunk(dir, c)
			if err != nil {
				continue
			}
			for _, pk := range WalkPackets(data) {
				if pk.Type != PacketTypeKeyframe {
					continue
				}
				paquets++
				for _, s := range KeyframeRecordSpans(pk.Payload(data)) {
					byTI[s.TI]++
					records++
					if s.Slot > slotMax {
						slotMax = s.Slot
					}
				}
			}
		}
		t.Logf("%s : %d paquets type-2, %d records, slot max %d ; ti=%d x%d",
			filepath.Base(dir), paquets, records, slotMax, equipeTI, byTI[equipeTI])
		t.Logf("    %s", equipeHistoTexte(byTI))
	}
}

// equipeHistoTexte rend un histogramme trie par cle, en une ligne.
func equipeHistoTexte(m map[int]int) string {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	s := ""
	for _, k := range keys {
		if s != "" {
			s += " "
		}
		s += fmt.Sprintf("%d:x%d", k, m[k])
	}
	return s
}
