package replay

// visee_composant_research_test.go — LOT F : LE DERNIER CANAL, A OFFSET VARIABLE (moteur).
//
// CE QUI RESTE APRES SIX NEGATIFS. Le chantier « visee a la lunette » a mesure et refute, dans
// cet ordre : aucun composant du registre ECS ne s'appelle « zoom » (325 inventories) ; la queue
// d'i21 est CONSTANTE sur 607 258 records ; l'evenement dedie n'existe pas dans la bobine (deux
// structures independantes — l'octet est absent des 41 M de paquets, et le film ne DECLARE que
// 123 types quand celui du zoom serait hors borne) ; aucun bit de la tete du record de degat ne
// separe zoome de non zoome (134 contre 780 tirs) ; aucun bit a POSITION FIXE des 1 024 premiers
// ni des 1 024 derniers bits du payload ne suit l'onde de zoom (lot C, 7 variantes, PUISSANCE
// mesuree) ; et aucun composant n'ENTRE au masque du bipede differemment selon l'etat de lunette
// (balayage de PRESENCE).
//
// IL EN RESTE UN, ET UN SEUL : un bit DANS LA CHARGE UTILE d'un composant deja present en
// permanence. Un tel bit est a OFFSET VARIABLE — il se deplace d'un record a l'autre au gre des
// portes des composants qui le precedent. Le balayage du lot C, qui indexait les bits depuis le
// DEBUT DU PAYLOAD, ne pouvait structurellement pas le voir : deux records portant le meme bit
// au meme endroit de leur composant le portent a deux positions absolues differentes des que le
// masque ou une porte amont change. C'est le dernier emplacement possible.
//
// CE QUE CET INSTRUMENT CHANGE, EN UNE LIGNE : l'offset est RELATIF AU DEBUT DU COMPOSANT.
//
// D'OU VIENNENT LES FRONTIERES DE COMPOSANT. De la marche ANCREE de production, pas d'une
// grammaire reecrite ici. Deux briques exportees, et rien entre les deux :
//
//	`filmdec.ScanBipedRecords` ancre les records bipedes d'un paquet delta sur la grammaire
//	d'en-tete bipede (prefixe, slot, tag, masque) — c'est le chemin que le depot qualifie de
//	ROBUSTE, par opposition a la marche sequentielle. Sous `CaptureDirs`, il publie par
//	`SetRecordMaskHook` la liste des index du masque et le bit qui suit i0.
//	`filmdec.ConsumeComponentAt` execute le DESER DE PRODUCTION d'un composant nomme a un bit
//	donne et rend le bit d'apres. Enchainee sur les index du masque, elle EST la marche
//	`walkRecordComponents` — meme dispatch `consumeByName`, meme arret au premier composant non
//	porte — vue de l'exterieur du paquet.
//
// LE CHEMIN SEQUENTIEL A ETE ESSAYE D'ABORD, ET IL EST PUBLIE COMME ECHEC (execution 1,
// 2026-08-30) : `DecodeFrameViews` ne rend que 7 records bipedes sur les trois premiers chunks
// de 00162144, et AUCUN sur les slots cibles, quelle que soit la largeur d'identifiant bas
// balayee (10 a 14). Le chemin ancre, lui, rend 141 045 positions bipedes sur le meme film.
// C'est la difference entre mesurer et croire mesurer.
//
// CE QUE LA MARCHE N'ATTEINT PAS EST MESURE, PAS SUPPOSE (mandat F5). La boucle s'arrete au
// premier composant non porte (`DesyncAt`) : au-dela, le curseur ne serait plus digne de
// confiance. Un negatif obtenu sur 30 % du record ne serait pas un negatif sur le record — donc
// la COUVERTURE est publiee avant tout verdict : part des records traverses en entier, et rang
// moyen du dernier composant consomme.
//
// LES CLASSES VIENNENT DU RELEVE THEATER DE L'UTILISATEUR (film 00162144, decalage feed->film
// +1 171 858 ms) : les episodes de `chronoEpisodes` / `chronoEpisodeMadina`, source unique
// partagee avec les instruments des phases 6, A et C. L'onde carree, ses bandes de garde et son
// controle par translation sont ceux du LOT C (`visee_onde_research_test.go`) : moteur reutilise
// tel quel, pas reecrit — c'est le seul moyen que les deux verdicts soient comparables.
//
// SEUILS ECRITS AVANT TOUTE MESURE (regle absolue du dossier ; repris du lot C, plus deux qui
// sont propres a la forme « par composant ») :
//
//	S1. CANDIDAT      : exactitude equilibree >= 0,95 avec >= 200 echantillons de CHAQUE classe.
//	S2. A SUIVRE      : exactitude equilibree >= 0,85 (meme exigence d'echantillons).
//	S3. SOUS-DIMENSIONNE : si une classe compte moins de 200 echantillons, la variante est
//	    declaree telle et son resultat n'est PAS publiable comme candidat — le verdict repose
//	    alors exclusivement sur S4.
//	S4. CONTROLE PAR TRANSLATION (le juge) : l'onde ENTIERE est translatee, fenetre comprise.
//	    p(max GLOBAL) = part des decalages ou le meilleur score TOUS COMPOSANTS ET TOUS OFFSETS
//	    confondus atteint le score observe — c'est lui qui fait foi, parce qu'il corrige de
//	    lui-meme le balayage de toutes les hypotheses du lot. p(max composant) et p(position)
//	    sont publies pour comparaison uniquement : le lot C a rattrape un faux positif a
//	    p(position) = 0,19 % que seul p(max) a refuse.
//	    VERDICT POSITIF EXIGE p(max GLOBAL) < 1 %.
//	S5. PUISSANCE     : la part des decalages temoins dont le meilleur score atteint 1,0000 est
//	    publiee AVEC le verdict. Sans elle, un negatif ne se distingue pas d'un aveuglement.
//	S6. RECEVABILITE  : un composant n'entre dans la mesure que s'il est ATTEINT sur >= 200
//	    records de la fenetre ; un decalage temoin n'est RETENU que s'il porte >= 30 echantillons
//	    de chaque classe (sinon il est compte et ecarte — un decalage a classe vide rendrait un
//	    score degenere qui gonflerait ou creuserait p(max) sans rien mesurer).
//
// LES TROIS DOMAINES, ET POURQUOI IL Y EN A TROIS. Le domaine COMPLET est celui du mandat, et
// c'est lui qui fait foi pour la question posee. Mais la PUISSANCE se paie au nombre
// d'hypotheses : balayer 138 couples avec une classe « zoome » de quelques dizaines
// d'echantillons suffit a ce qu'un decalage temoin atteigne 1,0000 par hasard, et le negatif
// devient alors un aveuglement — ce que l'execution 6 a mesure (4,75 % et 2,03 % contre 1 %
// requis). Deux domaines plus etroits sont donc mesures A COTE, et leur definition ne doit RIEN
// au resultat observe :
//
//	D1 COMPLET  tous les composants atteints. Le mandat.
//	D2 CIBLE    le seul `unit-command-tick-component`. Ce composant n'est pas choisi pour son
//	            score : il est DESIGNE par la phase 7, qui a etabli au desassemblage que l'etat
//	            de zoom d'une unite n'a que deux sources ecrites par des donnees, et que l'une
//	            est l'octet 6 de la COMMANDE JOUEUR (`FUN_1406db688` : `unite+0x462 =
//	            commande[6]`). Un composant qui s'appelle « command-tick » est cette commande.
//	            C'est l'hypothese la plus ancienne du dossier, et elle n'avait jamais eu de
//	            test a sa mesure.
//	D3 ETAT     tout sauf i0 (position) et i1 (velocite). Exclusion par PROPRIETE DU FORMAT, pas
//	            par score : ce sont des grandeurs spatiales continues dont la grammaire decrit
//	            chaque bit (porte + trois quanta d'axe + queue pour i0), sans place libre pour un
//	            drapeau d'etat — et leurs bits suivent la TRAJECTOIRE du joueur, donc separent
//	            deux intervalles de temps disjoints sans rien dire d'un etat. C'est ce
//	            confondant spatial qui detruit la puissance du domaine complet.
//
// HONNETETE DE CALENDRIER : D2 et D3 ont ete ecrits APRES l'execution 6, qui a revele le
// deficit de puissance de D1. Ni l'un ni l'autre ne depend du couple gagnant observe — D2 vient
// d'une retro-ingenierie anterieure au lot F, D3 d'une propriete de la grammaire. Le fait qu'ils
// aient ete ajoutes en cours de route est ecrit ici plutot que masque.
//
// SOUS GARDE D'ENVIRONNEMENT (COMPOSANT_FILM, qui doit pointer 00162144 : la chronologie est
// celle de CE film). Lecture seule, aucun code de production modifie.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 COMPOSANT_FILM=<repo>/data/cache/film_chunks/00162144 \
//	  go test ./internal/analysis/replay/ -run TestViseeComposant -v -timeout 90m

import (
	"fmt"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

const (
	vfFilmEnv = "COMPOSANT_FILM"
	// vfOffsetMax : nombre d'offsets relatifs balayes par composant. 64 bits couvrent la
	// charge utile entiere de la quasi-totalite des composants bipedes (i21 en fait 25 sur le
	// chemin dominant, 49 sur l'autre) ; les composants plus larges sont tronques, et le
	// PREFIXE est ce qui se compare d'un record a l'autre.
	vfOffsetMax = 64
	// vfCompEchMin / vfCtrlEchMin : les deux volets de S6.
	vfCompEchMin = 200
	vfCtrlEchMin = 30
	// vfSeuilCand / vfSeuilSuivre : S1 et S2.
	vfSeuilCand   = 0.95
	vfSeuilSuivre = 0.85
	// vfSeuilP : S4, la part maximale de decalages temoins toleree pour un verdict positif.
	vfSeuilP = 0.01
)

// vfComp est UN composant consomme dans un record, reduit a ce que la mesure regarde : son
// index d'archetype, son nom de registre, sa largeur MESUREE (frontiere a frontiere) et le
// prefixe de sa charge utile.
type vfComp struct {
	idx  int
	nom  string
	larg int
	// bits porte les min(larg, vfOffsetMax) premiers bits du composant, cadres a GAUCHE :
	// l'offset relatif o se lit bits>>(63-o)&1. Le cadrage a gauche fait que l'offset 0 est
	// le premier bit du composant quelle que soit sa largeur.
	bits uint64
}

// vfRecord est un record bipede attribue a un joueur, avec ses composants consommes.
type vfRecord struct {
	tMS   int64
	slot  uint32
	comps []vfComp
}

// vfStat porte la COUVERTURE de la marche (mandat F5) et les denominateurs du balayage.
type vfStat struct {
	// paquets delta deroules ; records marches ; records bipedes ancres sur les slots cibles.
	paquets, records, bipeds int
	// desappaires : paquets ou la liste du hook et celle des records n'ont pas la meme
	// longueur. Ces paquets sont ECARTES : un decalage silencieux attribuerait les bits d'un
	// record a un autre, et c'est exactement l'erreur que le lot precedent a payee.
	desappaires int
	// horsVie : records d'un slot cible mais hors des bornes d'une vie NOMMEE du joueur — le
	// slot avait alors migre vers quelqu'un d'autre.
	horsVie int
	// cibles : records verses dans la couverture.
	cibles int
	// entiers : records dont TOUS les composants annonces ont ete consommes ; desync : les
	// autres (la marche s'est arretee avant la fin du masque).
	entiers, desync int
	// annonces / consommes : composants annonces par les masques des records cibles, et
	// composants effectivement consommes. Leur rapport est « quelle part du record est lue ».
	annonces, consommes int
	// partSomme / partN : moyenne de la part consommee, record par record (une moyenne de
	// rapports, pas un rapport de sommes : un record long ne doit pas peser plus qu'un court).
	partSomme float64
	partN     int
	// arret : index du composant sur lequel la marche s'est arretee -> nombre de records.
	arret map[int]int
	// rangSomme / rangN : rang moyen (0-base) du dernier composant consomme.
	rangSomme float64
	rangN     int
	// presence : index de composant -> nombre de records cibles qui l'ont CONSOMME.
	presence map[int]int
	// noms : index de composant -> etiquette de registre (pour la publication).
	noms map[int]string
}

// vfNewStat rend un compteur pret a l'emploi.
func vfNewStat() vfStat {
	return vfStat{arret: map[int]int{}, presence: map[int]int{}, noms: map[int]string{}}
}

// vfLitBits lit le prefixe d'un composant, cadre a gauche sur 64 bits.
func vfLitBits(pay []byte, at, larg int) uint64 {
	n := larg
	if n > vfOffsetMax {
		n = vfOffsetMax
	}
	var v uint64
	for o := 0; o < n; o += 32 {
		w := 32
		if n-o < w {
			w = n - o
		}
		v |= uint64(filmdec.ReadBitsAtForDiag(pay, at+o, w)) << (64 - o - w)
	}
	return v
}

// vfI0TailBits est la QUEUE du composant i0, apres le vec3 quantifie : 2 bits sur le chemin
// dominant (handleSel puis regionPresent). Valeur MESUREE, documentee dans
// `filmdec/offline_aim.go` (const i0TailBits) et employee telle quelle par la marche de
// production `walkRecordComponents`. Elle est recopiee ici parce qu'elle n'est pas exportee —
// c'est la SEULE constante de grammaire que cet instrument reprend, et le CR le signale.
const vfI0TailBits = 2

// vfAncre est ce que le hook de masque publie pour UN record bipede ancre.
type vfAncre struct {
	idx     []int
	afterI0 int
}

// vfSource porte ce que la collecte doit savoir du film : le decoupage d'i0 (propre a la carte,
// LU dans le film) et l'archetype bipede du registre.
type vfSource struct {
	lay   filmdec.I0Layout
	arch  filmdec.Archetype
	blocs int
}

// vfOuvre lit le decoupage d'i0 et l'archetype bipede.
func vfOuvre(dir string) (vfSource, error) {
	var s vfSource
	lay, _, err := filmdec.DetectI0Layout(dir)
	if err != nil {
		return s, fmt.Errorf("decoupage d'i0 : %w", err)
	}
	s.lay = lay
	raw, err := filmdec.ReadFilmChunk(dir, 0)
	if err != nil {
		return s, fmt.Errorf("registre (chunk_00) illisible : %w", err)
	}
	reg, err := filmdec.ParseRegistryChunk(raw)
	if err != nil {
		return s, fmt.Errorf("registre illisible : %w", err)
	}
	s.blocs = len(reg.Archetypes)
	arch, ok := reg.Archetype(filmdec.BipedTypeIndex)
	if !ok {
		return s, fmt.Errorf("archetype bipede (ti=%d) absent d'un registre de %d blocs",
			filmdec.BipedTypeIndex, s.blocs)
	}
	s.arch = arch
	return s, nil
}

// vfMarche enchaine les desers de PRODUCTION sur les index du masque et rend, pour chaque
// composant consomme, son offset de debut et le prefixe de sa charge utile.
//
// ELLE S'ARRETE OU LA MARCHE DE PRODUCTION S'ARRETE : au premier composant sans deser porte,
// ou des que la lecture deborde du payload. Au-dela, la position du curseur ne serait plus
// digne de confiance, et mesurer du bruit vaut moins que ne rien mesurer. C'est exactement
// l'angle mort que la couverture (F5) chiffre.
func vfMarche(pay []byte, a vfAncre, s vfSource, st *vfStat) []vfComp {
	total := len(pay) * 8
	out := make([]vfComp, 0, len(a.idx))
	// i0 lui-meme : il EST dans le masque, donc il entre dans la mesure. Son debut est le bit
	// qui precede `afterI0` de toute la largeur du decoupage.
	i0 := a.afterI0 - s.lay.TotalBits()
	at := a.afterI0 + vfI0TailBits
	if i0 >= 0 && at <= total {
		out = append(out, vfComp{idx: 0, nom: vfNomI0, larg: at - i0, bits: vfLitBits(pay, i0, at-i0)})
	}
	for _, id := range a.idx[1:] {
		if id < 0 || id >= len(s.arch.Components) {
			st.arret[id]++
			return out
		}
		name := s.arch.Components[id]
		st.noms[id] = name
		if name == "" {
			st.arret[id]++
			return out
		}
		end, ported := filmdec.ConsumeComponentAt(pay, at, name, filmdec.BipedTypeIndex, s.arch.Level(id))
		if !ported || end > total || end <= at {
			st.arret[id]++
			return out
		}
		out = append(out, vfComp{idx: id, nom: name, larg: end - at, bits: vfLitBits(pay, at, end-at)})
		at = end
	}
	st.arret[-1]++
	return out
}

// vfNomI0 : l'etiquette de registre du composant de position, publiee telle quelle.
const vfNomI0 = "object-position-dynamic-precision-component"

// vfCompteCouverture verse UN record cible dans les compteurs de couverture (F5).
func vfCompteCouverture(st *vfStat, a vfAncre, comps []vfComp) {
	st.cibles++
	annonces := len(a.idx)
	st.annonces += annonces
	st.consommes += len(comps)
	if annonces > 0 {
		st.partSomme += float64(len(comps)) / float64(annonces)
		st.partN++
	}
	if len(comps) == annonces {
		st.entiers++
	} else {
		st.desync++
	}
	if len(comps) > 0 {
		st.rangSomme += float64(comps[len(comps)-1].idx)
		st.rangN++
	}
	for _, c := range comps {
		st.presence[c.idx]++
		st.noms[c.idx] = c.nom
	}
}

// vfCollecte deroule TOUT le film par la marche ancree et rend les records bipedes des slots
// cibles, decoupes en composants.
//
// LE FILM ENTIER, PAS LA SEULE FENETRE : le controle par translation a besoin d'echantillons
// partout ailleurs, sinon il n'aurait rien a comparer.
//
// L'APPARIEMENT HOOK <-> RECORD EST EXACT, ET C'EST LA RAISON DE PASSER PAR `ScanBipedRecords`
// PLUTOT QUE PAR `ScanFilmBipedPositions` : le hook tire AVANT les filtres de post-traitement
// (DropIsolated, DropTeleports), donc s'apparier au resultat filtre serait faux — c'est
// l'erreur documentee dans `visee_lunette_balayage_research_test.go`. Ici les deux listes
// viennent du meme appel non filtre, et un desaccord de longueur est compte et le paquet ecarte
// plutot que decale en silence.
func vfCollecte(dir string, s vfSource, pont vfPont, maxChunks int) (
	[]vfRecord, vfStat, error,
) {
	st := vfNewStat()
	cibles := pont.slots
	opt := filmdec.DefaultScanFilmOptions()
	opt.CaptureDirs = true
	opt.QuantaOnly = true
	var ancres []vfAncre
	filmdec.SetRecordMaskHook(func(idx []int, _ []byte, afterI0 int) {
		ancres = append(ancres, vfAncre{idx: append([]int(nil), idx...), afterI0: afterI0})
	})
	defer filmdec.SetRecordMaskHook(nil)

	var out []vfRecord
	fin := filmdec.CountFilmChunks(dir)
	if maxChunks > 0 && maxChunks < fin {
		fin = maxChunks
	}
	for c := 1; c <= fin; c++ {
		data, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range filmdec.WalkPackets(data) {
			if p.Type != filmdec.PacketTypeDelta {
				continue
			}
			st.paquets++
			pay := p.Payload(data)
			ancres = ancres[:0]
			recs := filmdec.ScanBipedRecords(pay, filmdec.NewSlotBand(cibles), s.lay, opt)
			out = append(out, vfVersePaquet(&st, recs, ancres, pay, p, s, pont)...)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].tMS < out[j].tMS })
	return out, st, nil
}

// vfVersePaquet marche les records ancres d'UN paquet.
//
// LE FILTRE PAR VIE, ET PAS SEULEMENT PAR SLOT : un slot MIGRE aux reapparitions, donc le meme
// numero designe successivement plusieurs joueurs. `ScanBipedRecords` ne sait filtrer que par
// slot ; c'est ici que le record devient celui d'une PERSONNE.
func vfVersePaquet(st *vfStat, recs []filmdec.BipedPosition, ancres []vfAncre, pay []byte,
	p filmdec.FilmPacket, s vfSource, pont vfPont,
) []vfRecord {
	st.bipeds += len(recs)
	if len(recs) != len(ancres) {
		st.desappaires++
		return nil
	}
	tMS := int64(p.TimestampUS / 1000)
	var out []vfRecord
	for i, r := range recs {
		st.records++
		if !pont.contient(r.Slot, tMS) {
			st.horsVie++
			continue
		}
		comps := vfMarche(pay, ancres[i], s, st)
		vfCompteCouverture(st, ancres[i], comps)
		if len(comps) == 0 {
			continue
		}
		out = append(out, vfRecord{tMS: tMS, slot: r.Slot, comps: comps})
	}
	return out
}

// vfFenetre compte les records de la fenetre d'analyse par classe, et rend leur plage
// temporelle. Publie AVANT toute mesure : quand une variante ne rend rien, c'est ici que se lit
// pourquoi — et une mesure vide qui ne dit pas pourquoi est un instrument mal fait.
func vfFenetre(recs []vfRecord, o ondeCarree) (dans, un, zero, garde int, t0, t1 int64) {
	for _, r := range recs {
		if t0 == 0 || r.tMS < t0 {
			t0 = r.tMS
		}
		if r.tMS > t1 {
			t1 = r.tMS
		}
		if r.tMS < ondeFeneDebutMS || r.tMS > ondeFeneFinMS {
			continue
		}
		dans++
		switch o.classe(r.tMS) {
		case 1:
			un++
		case 0:
			zero++
		default:
			garde++
		}
	}
	return dans, un, zero, garde, t0, t1
}
