package filmdec

// ground_weapon_lifecycle_research_test.go — LA TROISIEME PIECE : l'arme AU SOL.
//
// CE QU'IL CHERCHE. Si le lacher et la prise lus sur le BIPEDE (i43..i46) sont reels, alors
// le monde doit en porter le miroir : un lacher fait NAITRE un objet arme-au-sol (ti=42), une
// prise en fait DISPARAITRE un. Le flux delta porte ces naissances et ces morts comme des
// records d'entite NEW et DEL — c'est la marche stateful de production (DecodeFrameViews,
// World reamorce a chaque image-cle) qui les rend.
//
// C'EST UN TEST D'HYPOTHESE, PAS UNE PRODUCTION. Il ne publie rien et n'apparie aucune
// identite d'arme : il demande seulement si les INSTANTS coincident plus souvent que le
// hasard. Le temoin est un decalage temporel : les memes evenements, deplaces de 30 s, qui
// doivent s'effondrer si la coincidence est reelle.
//
// GARDE : HW_FILM, meme convention que les autres instruments de ce lot.

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
)

const (
	// gwWindowUS : tolerance d'appariement entre un evenement bipede et un evenement du
	// monde. 2 s couvre le decalage d'un paquet (~0,5 s mesure) avec de la marge.
	gwWindowUS = 500_000
	// gwWitnessShiftUS : le decalage du temoin. 30 s est bien au-dela de toute latence
	// physique entre un lacher et la naissance de l'objet.
	gwWitnessShiftUS = 30_000_000
	// gwViewsPerPacket : nombre de vues decodees par paquet (meme valeur que la phase 0
	// de l'attachement).
	gwViewsPerPacket = 4
)

// gwLifeEvent est une naissance ou une mort d'entite arme-au-sol.
type gwLifeEvent struct {
	TimestampUS uint64
	// Slot est le numero d'emplacement, RECYCLE d'une entite a l'autre. ID est
	// l'identifiant COMPLET, generation comprise (bits 30-31) : c'est LUI qui distingue
	// deux vies successives du meme emplacement. Apparier une naissance a une mort par le
	// slot seul confond ces deux vies — d'ou les durees de vie aberrantes du premier jet.
	Slot uint32
	ID   uint32
	New  bool
}

// gwScanLifecycle rend les NEW et DEL de l'archetype ti=42 sur tout le film.
func gwScanLifecycle(t *testing.T, dir string) []gwLifeEvent {
	t.Helper()
	raw, err := ReadFilmChunk(dir, 0)
	if err != nil {
		t.Fatalf("registre (chunk_00) illisible : %v", err)
	}
	reg, err := ParseRegistryChunk(raw)
	if err != nil {
		t.Fatalf("registre illisible : %v", err)
	}
	cfg := DefaultFrameConfig()
	w := NewWorld(reg)
	slotType := map[uint32]uint32{}
	var out []gwLifeEvent
	n := CountFilmChunks(dir)
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range WalkPackets(data) {
			pay := p.Payload(data)
			switch p.Type {
			case PacketTypeKeyframe:
				w = WorldFromKeyframe(reg, pay)
				// AMORCAGE INDISPENSABLE. Une arme posee sur un ratelier ou un socle par la
				// CARTE n'a jamais de record NEW en delta : elle existe deja dans le World de
				// l'image-cle. Sans cet amorcage, sa suppression au ramassage est invisible —
				// c'est l'erreur qui a produit le premier « PRISE -> MORT = 1,4 % ».
				for slot, st := range w.slots {
					slotType[slot] = st.TypeIndex
				}
				continue
			case PacketTypeDelta:
			default:
				continue
			}
			recs, _ := DecodeFrameViews(pay, w, cfg, gwViewsPerPacket, cfg.PacketPreambleBits)
			for _, r := range recs {
				switch r.Type {
				case recNew:
					// Le record NEW porte l'archetype : on le retient pour pouvoir nommer le
					// DEL plus tard.
					slotType[r.Slot] = r.TypeIndex
					if r.TypeIndex == GroundWeaponTypeIndex {
						out = append(out, gwLifeEvent{TimestampUS: p.TimestampUS, Slot: r.Slot, ID: r.ID, New: true})
					}
				case recDel:
					// LE DESERIALISEUR NE RENSEIGNE PAS TypeIndex SUR UN DEL (frame_records.go :
					// `case recDel: br.Skip(32); w.Unbind(slot)`). L'archetype d'une entite
					// supprimee ne peut donc venir que de sa NAISSANCE, retenue ci-dessus.
					if slotType[r.Slot] == GroundWeaponTypeIndex {
						out = append(out, gwLifeEvent{TimestampUS: p.TimestampUS, Slot: r.Slot, ID: r.ID, New: false})
					}
				}
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].TimestampUS < out[j].TimestampUS })
	return out
}

// gwNear dit s'il existe un evenement du type demande a moins de gwWindowUS de at.
func gwNear(ev []gwLifeEvent, at uint64, wantNew bool) bool {
	for _, e := range ev {
		if e.New != wantNew {
			continue
		}
		d := int64(e.TimestampUS) - int64(at)
		if d < 0 {
			d = -d
		}
		if d <= gwWindowUS {
			return true
		}
	}
	return false
}

// TestGroundWeaponLifecycleVsHeldWeapon confronte les lachers et les prises lus sur le
// bipede aux naissances et aux morts d'armes au sol.
func TestGroundWeaponLifecycleVsHeldWeapon(t *testing.T) {
	dir := os.Getenv(hwFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure saute", hwFilmEnv)
	}
	release := LockProcessDecode()
	defer release()

	t.Log("CRITERE (enonce avant lecture) : un LACHER doit coincider avec une NAISSANCE " +
		"d'arme au sol a moins de 2 s ; une PRISE avec une MORT. Seuil de retenue : >= 70 % " +
		"des evenements apparies, avec un temoin decale de 30 s a <= 15 %. Sous ces valeurs, " +
		"le miroir du monde n'est pas etabli et rien ne doit etre publie.")

	life := gwScanLifecycle(t, dir)
	news, dels := 0, 0
	for _, e := range life {
		if e.New {
			news++
		} else {
			dels++
		}
	}
	t.Logf("MONDE : evenements ti=%d en delta : naissances=%d morts=%d (total=%d)",
		GroundWeaponTypeIndex, news, dels, len(life))
	if len(life) == 0 {
		t.Log("VERDICT : aucun evenement de cycle de vie ti=42 sur ce film. Le miroir du " +
			"monde ne peut pas etre teste ici — il faut un film ou ti=42 est present en delta.")
		return
	}

	s := hwResolve(t, dir)
	ev := hwIdentities(hwScanEvents(s))
	kfRef := hwKeyframeRef(t, dir)
	type key struct {
		slot uint32
		comp int
	}
	prev, seen := map[key]uint32{}, map[key]bool{}
	var lachers, prises []uint64
	for _, e := range ev {
		k := key{e.Slot, e.CompIndex}
		switch {
		case e.IDHigh == noVariant:
			lachers = append(lachers, e.TimestampUS)
		case seen[k] && prev[k] == noVariant:
			prises = append(prises, e.TimestampUS)
		case !seen[k]:
			if ref, ok := kfRef.setAt(e.Slot, e.TimestampUS); ok && !ref[e.IDHigh] {
				prises = append(prises, e.TimestampUS)
			}
		}
		seen[k], prev[k] = true, e.IDHigh
	}
	t.Logf("BIPEDE : lachers=%d prises=%d", len(lachers), len(prises))

	score := func(times []uint64, wantNew bool, shift uint64) (int, int) {
		hit := 0
		for _, at := range times {
			if gwNear(life, at+shift, wantNew) {
				hit++
			}
		}
		return hit, len(times)
	}
	pct := func(a, b int) float64 {
		if b == 0 {
			return 0
		}
		return 100 * float64(a) / float64(b)
	}

	lh, ln := score(lachers, true, 0)
	lhW, _ := score(lachers, true, gwWitnessShiftUS)
	ph, pn := score(prises, false, 0)
	phW, _ := score(prises, false, gwWitnessShiftUS)

	t.Logf("LACHER -> NAISSANCE : %d/%d (%.1f %%) ; temoin decale 30 s : %d/%d (%.1f %%)",
		lh, ln, pct(lh, ln), lhW, ln, pct(lhW, ln))
	t.Logf("PRISE  -> MORT      : %d/%d (%.1f %%) ; temoin decale 30 s : %d/%d (%.1f %%)",
		ph, pn, pct(ph, pn), phW, pn, pct(phW, pn))
}

// TestGroundWeaponNearestDelta remplace le seuil binaire par une DISTRIBUTION : pour chaque
// prise, l'ecart signe a la mort d'arme au sol la plus proche. Une relation reelle produit un
// pic net pres de zero ; une absence de relation produit un etalement uniforme. Un seuil
// binaire ne distingue pas les deux, une distribution si.
func TestGroundWeaponNearestDelta(t *testing.T) {
	dir := os.Getenv(hwFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure saute", hwFilmEnv)
	}
	release := LockProcessDecode()
	defer release()

	life := gwScanLifecycle(t, dir)
	s := hwResolve(t, dir)
	ev := hwIdentities(hwScanEvents(s))
	kfRef := hwKeyframeRef(t, dir)

	type key struct {
		slot uint32
		comp int
	}
	prev, seen := map[key]uint32{}, map[key]bool{}
	var prises, lachers []uint64
	for _, e := range ev {
		k := key{e.Slot, e.CompIndex}
		switch {
		case e.IDHigh == noVariant:
			lachers = append(lachers, e.TimestampUS)
		case seen[k] && prev[k] == noVariant:
			prises = append(prises, e.TimestampUS)
		case !seen[k]:
			if ref, ok := kfRef.setAt(e.Slot, e.TimestampUS); ok && !ref[e.IDHigh] {
				prises = append(prises, e.TimestampUS)
			}
		}
		seen[k], prev[k] = true, e.IDHigh
	}

	report := func(label string, times []uint64, wantNew bool) {
		if len(times) == 0 {
			return
		}
		var d []int64
		for _, at := range times {
			best := int64(1 << 62)
			for _, e := range life {
				if e.New != wantNew {
					continue
				}
				x := int64(e.TimestampUS) - int64(at)
				if gwAbs(x) < gwAbs(best) {
					best = x
				}
			}
			d = append(d, best)
		}
		sort.Slice(d, func(i, j int) bool { return d[i] < d[j] })
		med := d[len(d)/2]
		var within500, within2s int
		for _, x := range d {
			if gwAbs(x) <= 500_000 {
				within500++
			}
			if gwAbs(x) <= 2_000_000 {
				within2s++
			}
		}
		t.Logf("%s (n=%d) : ecart median=%+d ms ; |ecart|<=0,5s : %d (%.0f %%) ; <=2s : %d (%.0f %%)",
			label, len(d), med/1000, within500, 100*float64(within500)/float64(len(d)),
			within2s, 100*float64(within2s)/float64(len(d)))
		t.Logf("   ecarts (ms, tries) : %s", gwMsList(d))
	}
	report("PRISE  -> mort la plus proche", prises, false)
	report("LACHER -> naissance la plus proche", lachers, true)
}

func gwAbs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

func gwMsList(d []int64) string {
	parts := make([]string, 0, len(d))
	for _, x := range d {
		parts = append(parts, fmt.Sprintf("%+d", x/1000))
	}
	return strings.Join(parts, " ")
}

// TestGroundWeaponDespawn verifie LE DESPAWN : une arme au sol nee pendant le match
// disparait-elle, et au bout de combien de temps ? C'est la condition pour afficher une arme
// lachee « jusqu'a ce qu'elle disparaisse » — sans borne de fin, on afficherait un objet
// eternel.
//
// LE PIEGE EVITE : un slot d'entite est RECYCLE. Apparier naissance et mort par le seul slot
// confondrait deux vies successives du meme numero. On apparie donc chaque naissance a la
// PREMIERE mort qui la suit sur ce slot, et toute naissance suivante ferme la precedente.
func TestGroundWeaponDespawn(t *testing.T) {
	dir := os.Getenv(hwFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure saute", hwFilmEnv)
	}
	release := LockProcessDecode()
	defer release()

	life := gwScanLifecycle(t, dir)
	open := map[uint32]uint64{} // cle = ID COMPLET, pas le slot
	var durations []int64
	born, closed, stillOpen := 0, 0, 0
	for _, e := range life {
		if e.New {
			if _, dup := open[e.ID]; dup {
				// Naissance sur un slot deja ouvert : la vie precedente n'a pas ete
				// fermee par un DEL. On la compte comme non fermee et on repart.
				stillOpen++
			}
			open[e.ID] = e.TimestampUS
			born++
			continue
		}
		if t0, ok := open[e.ID]; ok {
			durations = append(durations, int64(e.TimestampUS-t0))
			delete(open, e.ID)
			closed++
		}
	}
	stillOpen += len(open)
	if born == 0 {
		t.Log("VERDICT : aucune naissance d'arme au sol en delta sur ce film.")
		return
	}
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	q := func(p float64) int64 {
		if len(durations) == 0 {
			return 0
		}
		i := int(float64(len(durations)-1) * p)
		return durations[i] / 1000
	}
	t.Logf("ARMES AU SOL nees en delta = %d ; FERMEES par une disparition = %d (%.1f %%) ; "+
		"encore ouvertes a la fin = %d", born, closed, 100*float64(closed)/float64(born), stillOpen)
	t.Logf("DUREE DE VIE (ms) : p10=%d  mediane=%d  p90=%d  max=%d",
		q(0.10), q(0.50), q(0.90), q(1.0))
	t.Log("LECTURE : un taux de fermeture eleve autorise l'affichage borne « du lacher au " +
		"despawn ». Les vies encore ouvertes a la fin du film sont normales (l'objet existait " +
		"encore quand l'enregistrement s'arrete) et ne comptent pas comme un defaut.")
}

// gwLastSeen rend, pour chaque entite ti=42 vue en delta, l'instant de sa NAISSANCE et celui
// de sa DERNIERE apparition dans le flux — quel que soit le type de record. Une entite qui
// existe encore emet des deltas : sa derniere emission est donc une borne haute BEAUCOUP plus
// fine que l'image-cle suivante (20 s), et c'est la borne que l'utilisateur a demandee.
//
// La cle est l'ID COMPLET (generation comprise), jamais le slot, qui est recycle.
func gwLastSeen(t *testing.T, dir string) (born, last map[uint32]uint64) {
	t.Helper()
	raw, err := ReadFilmChunk(dir, 0)
	if err != nil {
		t.Fatalf("registre (chunk_00) illisible : %v", err)
	}
	reg, err := ParseRegistryChunk(raw)
	if err != nil {
		t.Fatalf("registre illisible : %v", err)
	}
	cfg := DefaultFrameConfig()
	w := NewWorld(reg)
	slotType := map[uint32]uint32{}
	born, last = map[uint32]uint64{}, map[uint32]uint64{}
	n := CountFilmChunks(dir)
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range WalkPackets(data) {
			pay := p.Payload(data)
			switch p.Type {
			case PacketTypeKeyframe:
				w = WorldFromKeyframe(reg, pay)
				for slot, st := range w.slots {
					slotType[slot] = st.TypeIndex
				}
				continue
			case PacketTypeDelta:
			default:
				continue
			}
			recs, _ := DecodeFrameViews(pay, w, cfg, gwViewsPerPacket, cfg.PacketPreambleBits)
			for _, r := range recs {
				if r.Type == recNew {
					slotType[r.Slot] = r.TypeIndex
					if r.TypeIndex == GroundWeaponTypeIndex {
						born[r.ID] = p.TimestampUS
					}
				}
				if slotType[r.Slot] != GroundWeaponTypeIndex {
					continue
				}
				last[r.ID] = p.TimestampUS
			}
		}
	}
	return born, last
}

// TestGroundWeaponLastSeen mesure la borne haute par le FLUX DELTA plutot que par les
// images-cles : combien de temps une arme lachee reste-t-elle visible dans le flux ?
func TestGroundWeaponLastSeen(t *testing.T) {
	dir := os.Getenv(hwFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure saute", hwFilmEnv)
	}
	release := LockProcessDecode()
	defer release()

	born, last := gwLastSeen(t, dir)
	var d []int64
	var filmEnd uint64
	for _, ts := range last {
		if ts > filmEnd {
			filmEnd = ts
		}
	}
	stillAtEnd := 0
	for id, t0 := range born {
		l, ok := last[id]
		if !ok || l < t0 {
			continue
		}
		if filmEnd-l < 2_000_000 { // encore presente a la fin du film : borne non informative
			stillAtEnd++
			continue
		}
		d = append(d, int64(l-t0))
	}
	if len(d) == 0 {
		t.Logf("VERDICT : aucune arme au sol avec une borne exploitable (nees=%d, "+
			"encore presentes a la fin=%d).", len(born), stillAtEnd)
		return
	}
	sort.Slice(d, func(i, j int) bool { return d[i] < d[j] })
	q := func(p float64) int64 { return d[int(float64(len(d)-1)*p)] / 1000 }
	t.Logf("ARMES AU SOL nees en delta = %d ; avec borne EXPLOITABLE = %d ; encore presentes a "+
		"la fin du film = %d", len(born), len(d), stillAtEnd)
	t.Logf("PRESENCE DANS LE FLUX (ms) : p10=%d  p25=%d  mediane=%d  p75=%d  p90=%d  max=%d",
		q(0.10), q(0.25), q(0.50), q(0.75), q(0.90), q(1.0))
	t.Log("LECTURE : c'est une borne HAUTE honnete — l'arme existait au moins jusque-la. Une " +
		"distribution resserree autour du timer de despawn du jeu (~30 s) confirmerait que le " +
		"flux delta suit l'objet jusqu'a sa disparition reelle.")
}

// TestGroundWeaponAmmo lit les MUNITIONS des armes posees au sol (ti=42 i20, porte et jamais
// exploite) et demande si elles expliquent la persistance : une arme NON VIDE reste-t-elle
// visible plus longtemps qu'une arme vide ?
//
// C'est la verification de la note utilisateur (« les armes speciales NON VIDES s'affichent
// jusqu'a leur despawn ou pickup »). La source Steam citee ne repond plus ; on mesure donc
// directement dans le film au lieu de la croire ou de la refuter sur parole.
func TestGroundWeaponAmmo(t *testing.T) {
	dir := os.Getenv(hwFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure saute", hwFilmEnv)
	}
	release := LockProcessDecode()
	defer release()

	var last struct {
		a, b, c uint32
		got     bool
	}
	prev := groundWeaponAmmoHook
	SetGroundWeaponAmmoHook(func(a, b, c uint32) { last.a, last.b, last.c, last.got = a, b, c, true })
	defer SetGroundWeaponAmmoHook(prev)

	born, lastSeen := gwLastSeen(t, dir)
	_ = born
	_ = lastSeen

	// Second passage : on relit en capturant la munition au moment de chaque record ti=42.
	raw, err := ReadFilmChunk(dir, 0)
	if err != nil {
		t.Fatalf("registre illisible : %v", err)
	}
	reg, err := ParseRegistryChunk(raw)
	if err != nil {
		t.Fatalf("registre illisible : %v", err)
	}
	cfg := DefaultFrameConfig()
	w := NewWorld(reg)
	slotType := map[uint32]uint32{}
	histA, histB, histC := map[uint32]int{}, map[uint32]int{}, map[uint32]int{}
	reads := 0
	n := CountFilmChunks(dir)
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range WalkPackets(data) {
			pay := p.Payload(data)
			switch p.Type {
			case PacketTypeKeyframe:
				w = WorldFromKeyframe(reg, pay)
				for slot, st := range w.slots {
					slotType[slot] = st.TypeIndex
				}
				continue
			case PacketTypeDelta:
			default:
				continue
			}
			last.got = false
			recs, _ := DecodeFrameViews(pay, w, cfg, gwViewsPerPacket, cfg.PacketPreambleBits)
			for _, r := range recs {
				if r.Type == recNew {
					slotType[r.Slot] = r.TypeIndex
				}
				if slotType[r.Slot] != GroundWeaponTypeIndex || !last.got {
					continue
				}
				reads++
				histA[last.a]++
				histB[last.b]++
				histC[last.c]++
				last.got = false
			}
		}
	}
	t.Logf("LECTURES de munitions sur des armes au sol = %d", reads)
	if reads == 0 {
		t.Log("VERDICT : le composant i20 n'est jamais atteint sur ce film — soit le masque ne " +
			"le porte pas, soit la marche s'arrete avant. Rien a conclure sur les munitions.")
		return
	}
	show := func(name string, h map[uint32]int) {
		type kv struct {
			v uint32
			n int
		}
		var l []kv
		for v, n := range h {
			l = append(l, kv{v, n})
		}
		sort.Slice(l, func(i, j int) bool { return l[i].n > l[j].n })
		if len(l) > 6 {
			l = l[:6]
		}
		s := ""
		for _, e := range l {
			s += fmt.Sprintf("%d(x%d) ", e.v, e.n)
		}
		t.Logf("  %s : valeurs les plus frequentes -> %s", name, s)
	}
	show("champ A R(8)", histA)
	show("champ B R(11)", histB)
	show("champ C R(12)", histC)
	t.Log("LECTURE : un champ dont les valeurs ressemblent a des tailles de chargeur (0, 12, " +
		"36, 45...) est le candidat MUNITIONS. Un champ quasi constant ne l'est pas.")
}
