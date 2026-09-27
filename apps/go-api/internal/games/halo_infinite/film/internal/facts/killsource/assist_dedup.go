package killsource

// assist_dedup.go — UN ENREGISTREMENT, UNE LECTURE (lot J7.6 du PLAN_SUITE_AUDIT_DECODEUR_FILM,
// constat FK-6).
//
// Le generateur de [killEventsIn] essaie CHAQUE position de bit ; une chaine d evenements valide
// peut donc etre retenue deux fois dans le meme paquet, a deux positions voisines, avec des champs
// IDENTIQUES (RE_LOG 7ter.77 : les deux morts a multi-attachement du corpus, a 15 bits d ecart). Le
// second exemplaire n est pas un second evenement — un joueur ne tue pas deux fois la meme victime
// dans le meme paquet — et le laisser libre fabriquait un couple faux pour un kill orphelin voisin
// (`feed_couples.go`, compte en `Contradiction`).
//
// C EST LE MEME GESTE que [dedup] pour les dead-states : meme paquet, memes champs, un seul
// enregistrement. Le champ `bit` designe l exemplaire garde — le PREMIER, celui du bit le plus bas.
// `end` (la fin des champs) est exclu de la cle : il suit la position de lecture, donc differe
// entre les deux exemplaires. Les retires sont COMPTES (`AssistStats.Doublons`).

// cleDeKillEvent : ce qui fait qu un enregistrement est le meme, position de lecture exclue.
type cleDeKillEvent struct {
	chunk, pidx            int
	killer, victim, assist int
	killerPct, assistPct   uint32
	flag                   int
}

// dedoublonner retire de la passe les kill-events qui repetent un enregistrement deja lu dans le
// meme paquet, en gardant celui du bit le plus bas.
func (s *assistScan) dedoublonner() {
	garde := make(map[cleDeKillEvent]int, len(s.recs))
	out := make([]killEventRec, 0, len(s.recs))
	for _, r := range s.recs {
		f := r.fields
		k := cleDeKillEvent{chunk: r.chunk, pidx: r.pidx, killer: f.killer, victim: f.victim,
			assist: f.assist, killerPct: f.killerPct, assistPct: f.assistPct, flag: f.flag}
		i, vu := garde[k]
		if !vu {
			garde[k] = len(out)
			out = append(out, r)
			continue
		}
		s.doublons++
		if r.bit < out[i].bit {
			out[i] = r
		}
	}
	s.recs = out
}
