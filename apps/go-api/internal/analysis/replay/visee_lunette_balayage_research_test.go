package replay

// visee_lunette_balayage_research_test.go — LE BALAYAGE : QUEL COMPOSANT DU BIPEDE, S'IL EN EST
// UN, SEPARE « ZOOME » DE « PAS ZOOME » ?
//
// L'ORACLE. Il ne vient plus d'un proxy (« kill au sniper => sans doute zoome ») mais du JEU
// LUI-MEME, par deux medailles opposees que le film porte, datees a la ms et attribuees a un xuid
// (cf. `visee_medailles_research_test.go` pour le recensement et les codes) :
//
//	No Scope      (100,114)  « ... WITHOUT ZOOMING »            -> classe SANS LUNETTE
//	Counter-snipe (100,168)  « ... while YOU BOTH ARE ZOOMED »  -> classe AVEC LUNETTE
//
// CE QUE LE BALAYAGE FAIT. Il ne devine AUCUN composant. Pour chaque instant etiquete, il prend
// les records bipedes du joueur concerne dans la fenetre qui precede, et il note QUELS INDEX DE
// COMPOSANT leur masque declare. Puis il compare, index par index, la frequence entre les deux
// classes. Si un composant porte l'etat de lunette, il doit se voir : en replication delta un
// composant n'entre au masque que lorsque son etat CHANGE, donc une mise en lunette avant le tir
// doit faire apparaitre son composant chez la classe AVEC LUNETTE et pas chez l'autre.
//
// DEUX FENETRES, parce qu'elles ne repondent pas a la meme question :
//
//	PRESENCE  le composant apparait-il AU MOINS UNE FOIS dans les `adsSweepFenetreMS` ms qui
//	          precedent ? C'est la forme adaptee a la replication delta (on cherche l'EVENEMENT
//	          de mise en lunette).
//	DERNIER   le masque du DERNIER record avant l'instant. C'est la forme adaptee a un composant
//	          qui reemettrait en continu.
//
// TEMOIN OBLIGATOIRE. Le meme calcul sur une fenetre DECALEE de `adsSweepTemoinMS` ms en amont.
// Un composant qui separe aussi bien la-bas ne separe pas la lunette : il separe les joueurs, les
// films ou les armes. C'est le seul garde-fou contre un effet de population.
//
// SEUILS ECRITS AVANT LA MESURE : voir le bloc de constantes ci-dessous.
//
// D'OU VIENT LE MASQUE. De `BipedPosition.MaskBits`, qui voyage DANS le record. La premiere
// version de cet instrument passait par le hook de diagnostic `SetRecordMaskHook` et appairait
// ses appels aux positions renvoyees : c'etait FAUX, le hook tire AVANT `DropIsolated`, donc les
// deux listes n'ont pas la meme longueur et 144 films sur 148 se faisaient rejeter par le
// controle d'appariement. Le masque porte par le record supprime le probleme a la racine.
//
// PERIMETRE. Seuls les films portant au moins un Counter-snipe sont decodes (c'est la classe
// RARE) : decoder les 1369 films couterait des heures pour ajouter du « sans lunette » dont on a
// deja de reste. Le choix est journalise, pas silencieux.
//
// SOUS GARDE D'ENVIRONNEMENT (ADS_SWEEP_DIR), donc saute partout ailleurs, CI comprise.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 ADS_SWEEP_DIR=<repo>/data/cache/film_chunks \
//	  ADS_TSV=<repo>/.ai/V7.5/film_re \
//	  go test ./internal/analysis/replay/ -run TestViseeLunetteBalayage -v -timeout 240m

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"levelup/go-api/internal/analysis/filmdec"
)

const adsSweepDirEnv = "ADS_SWEEP_DIR"

// LES SEUILS, ECRITS AVANT LA MESURE.
const (
	// adsSweepFenetreMS : fenetre amont de la forme PRESENCE. 3 s, parce qu'une mise en lunette
	// precede le tir de plusieurs centaines de ms a quelques secondes, jamais davantage.
	adsSweepFenetreMS = 3000
	// adsSweepDernierMS : tolerance de la forme DERNIER — au-dela, « juste avant le tir » serait
	// un abus de langage.
	adsSweepDernierMS = 300
	// adsSweepTemoinMS : recul du temoin. 30 s en amont : assez loin pour que l'etat de lunette
	// n'ait aucune raison d'y ressembler, assez proche pour rester le meme joueur dans le meme
	// film et souvent la meme vie.
	adsSweepTemoinMS = 30000
	// adsSweepMinClasse : sous 60 instants exploitables dans la classe RARE, aucun taux n'est
	// publiable. La classe abondante est de toute facon largement au-dessus.
	adsSweepMinClasse = 60
	// adsSweepEcartSeuil / adsSweepFacteur : un index n'est retenu comme CANDIDAT que si l'ecart
	// de frequence entre les deux classes atteint 20 points ET un facteur 2, dans un sens ou dans
	// l'autre. Et il tombe si le TEMOIN montre le meme ecart.
	adsSweepEcartSeuil = 0.20
	adsSweepFacteur    = 2.0
	// adsSweepMaxIndex : borne du balayage, egale a la largeur de `BipedPosition.MaskBits`.
	adsSweepMaxIndex = 64
	// adsSweepCible : on ARRETE des que la classe rare atteint ce compte. Au-dela du seuil de
	// publication (adsSweepMinClasse), chaque film supplementaire n'achete que de la decimale, et
	// il se paie en minutes. 100 laisse une marge confortable sur 60 sans payer le corpus entier.
	adsSweepCible = 100
	// adsSweepMaxFilms : plafond dur de films decodes, pour qu un corpus pauvre en etiquettes ne
	// puisse pas transformer la mesure en run de plusieurs heures. Cale a 150 = tout le corpus
	// porteur de l etiquette RARE : mesure faite, un film coute ~8 s, donc le corpus entier coute
	// ~20 min. Un plafond plus bas ferait atterrir la classe rare JUSTE au seuil de publication et
	// risquerait un « population insuffisante » — c est-a-dire payer le run sans avoir de reponse.
	adsSweepMaxFilms = 150
	// adsSweepPalierFilms : periodicite de la sortie progressive.
	adsSweepPalierFilms = 5
)

// adsClasse : une des deux faces de l'oracle.
type adsClasse int

const (
	adsSansLunette adsClasse = iota // No Scope
	adsAvecLunette                  // Counter-snipe
	adsNbClasses
)

func (c adsClasse) String() string {
	if c == adsAvecLunette {
		return "AVEC LUNETTE (Counter-snipe)"
	}
	return "SANS LUNETTE (No Scope)"
}

// adsTally compte, par classe et par index de composant, les instants exploitables et ceux dont
// le masque porte l'index.
type adsTally struct {
	instants [adsNbClasses]int
	vus      [adsNbClasses][adsSweepMaxIndex]int
}

func (a *adsTally) ajoute(c adsClasse, masque uint64) {
	a.instants[c]++
	for id := 0; id < adsSweepMaxIndex; id++ {
		if masque&(1<<uint(id)) != 0 {
			a.vus[c][id]++
		}
	}
}

// adsSweepBilan porte les trois tables du balayage et les denominateurs de rejet.
type adsSweepBilan struct {
	presence, dernier, temoin adsTally
	// films : films decodes. Les trois compteurs de rejet sont SEPARES : un rejet muet est un
	// denominateur perdu, et on a deja paye une fois pour l'avoir appris.
	films, rejetDecodage, rejetFilDesMorts, rejetPont int
	// instantsVus / instantsSitues : etiquettes rencontrees, et celles rattachees a un slot.
	instantsVus, instantsSitues int
	// masqueTronque : records dont un index depassait 63 (jamais attendu sur un bipede).
	masqueTronque int
}

// TestViseeLunetteBalayage execute le balayage sur les films portant l'etiquette rare.
func TestViseeLunetteBalayage(t *testing.T) {
	root := os.Getenv(adsSweepDirEnv)
	if root == "" {
		t.Skipf("%s absent : balayage saute", adsSweepDirEnv)
	}
	dirs := adsListeFilms(t, root)
	var cibles []adsMedailleFilm
	for _, d := range dirs {
		f, ok := adsRecenseFilm(root, d)
		if ok && f.counter > 0 {
			cibles = append(cibles, f)
		}
	}
	t.Logf("PERIMETRE — %d films portent au moins un Counter-snipe ; ce sont les seuls decodes."+
		" Les films sans etiquette rare sont ECARTES et n'entrent dans aucun denominateur.",
		len(cibles))
	if len(cibles) == 0 {
		t.Skip("aucun film porteur de l'etiquette rare")
	}

	var b adsSweepBilan
	debut := time.Now()
	arret := ""
	for i, f := range cibles {
		adsBalayeFilm(filepath.Join(root, f.film), f, &b)
		if (i+1)%adsSweepPalierFilms == 0 {
			// SORTIE PROGRESSIVE — le run precedent a expire a 4 h en n'ayant RIEN imprime, donc
			// en ayant tout perdu. Un instrument de mesure long qui ne publie qu'a la fin est un
			// instrument mal fait : ces lignes-ci sont le correctif, pas de la decoration.
			t.Logf("  ... %d/%d films, %s ecoules · classes : %d sans lunette / %d AVEC (cible %d)",
				i+1, len(cibles), time.Since(debut).Round(time.Second),
				b.presence.instants[adsSansLunette], b.presence.instants[adsAvecLunette],
				adsSweepCible)
		}
		if b.presence.instants[adsAvecLunette] >= adsSweepCible {
			arret = "cible de la classe rare atteinte"
		} else if i+1 >= adsSweepMaxFilms {
			arret = "plafond de films atteint"
		}
		if arret != "" {
			t.Logf("ARRET ANTICIPE (%s) apres %d films : le reste du corpus n'apporterait que de la"+
				" precision sur une classe deja au-dessus du seuil.", arret, i+1)
			break
		}
	}
	t.Logf("COUT — %d films decodes en %s", b.films, time.Since(debut).Round(time.Second))
	t.Logf("REJETS — decodage %d · fil des morts %d · pont slot->xuid vide %d",
		b.rejetDecodage, b.rejetFilDesMorts, b.rejetPont)
	t.Logf("ETIQUETTES — %d rencontrees, %d rattachees a un slot (%d records a masque tronque)",
		b.instantsVus, b.instantsSitues, b.masqueTronque)

	adsSweepJournalise(t, b)
	adsSweepVerdict(t, b)
	adsSweepTSV(t, b)
}

// adsBalayeFilm decode UN film et verse ses instants etiquetes dans le bilan.
//
// LE PONT NE PASSE PAS PAR `ScanFilmPlayerIndices`, ET C'EST LE POINT DE PERFORMANCE DU FICHIER.
// Cette fonction balaie le film bit a bit pour CHAQUE joueur du roster : elle coutait a elle
// seule l'essentiel des ~1,6 min par film de la premiere version, qui a fini par expirer a 4 h
// sans rien publier. Or elle sert a nommer l'INDICE DE JOUEUR, dont ce balayage n'a aucun besoin :
// tout ce qu'il faut ici est slot -> xuid, et `nameLivesByDeaths` le donne a partir du SEUL fil
// des morts (la mort qui termine une vie nomme cette vie).
//
// Bonus de justesse, pas seulement de vitesse : on resout le slot VIE PAR VIE, en cherchant la vie
// du bon xuid qui CONTIENT l'instant. Un slot migre aux respawns ; une table slot -> xuid globale
// aurait melange deux vies d'un meme slot.
func adsBalayeFilm(dir string, f adsMedailleFilm, b *adsSweepBilan) {
	release := filmdec.LockProcessDecode()
	defer release()

	scan := filmdec.DefaultScanFilmOptions()
	scan.CaptureDirs = true
	scan.QuantaOnly = true
	pos, err := filmdec.ScanFilmBipedPositions(dir, scan)
	if err != nil || len(pos) == 0 {
		b.rejetDecodage++
		return
	}
	deaths, err := ScanFilmDeaths(dir)
	if err != nil || len(deaths) == 0 {
		b.rejetFilDesMorts++
		return
	}
	lives := buildLifeSpans(indexBySlot(pos))
	off, _ := bestDeathOffset(lives, deaths)
	if nameLivesByDeaths(lives, deaths, off) == 0 {
		b.rejetPont++
		return
	}
	b.films++

	tracks := adsIndexeMasques(pos, b)
	for i, ms := range f.instantsNoScope {
		adsVerseInstant(b, tracks, lives, f.xuidNoScope[i], int64(ms)+off, adsSansLunette)
	}
	for i, ms := range f.instantsCounter {
		adsVerseInstant(b, tracks, lives, f.xuidCounter[i], int64(ms)+off, adsAvecLunette)
	}
}

// adsSlotsDeLInstant rend les slots des vies du joueur `xuid` qui contiennent l'instant, ou qui le
// touchent a `adsSweepTemoinMS` pres (le temoin regarde en amont, parfois dans la vie precedente).
func adsSlotsDeLInstant(lives []lifeSpan, xuid uint64, tFilmMS int64) []uint32 {
	tUS := tFilmMS * 1000
	marge := int64(adsSweepTemoinMS+adsSweepFenetreMS) * 1000
	var out []uint32
	for _, l := range lives {
		if l.xuid != xuid {
			continue
		}
		if l.to < tUS-marge || l.from > tUS {
			continue
		}
		out = append(out, l.slot)
	}
	return out
}

// adsEchantillon : un record bipede reduit a ce que le balayage regarde.
type adsEchantillon struct {
	tMS    int64
	masque uint64
}

// adsIndexeMasques range les masques par slot, tries par instant.
func adsIndexeMasques(pos []filmdec.BipedPosition, b *adsSweepBilan) map[uint32][]adsEchantillon {
	out := map[uint32][]adsEchantillon{}
	for _, p := range pos {
		if p.MaskOver {
			b.masqueTronque++
		}
		out[p.Slot] = append(out[p.Slot], adsEchantillon{
			tMS: int64(p.TimestampUS) / 1000, masque: p.MaskBits,
		})
	}
	for s := range out {
		e := out[s]
		sort.Slice(e, func(i, j int) bool { return e[i].tMS < e[j].tMS })
		out[s] = e
	}
	return out
}

// adsVerseInstant verse UN instant etiquete dans les trois tables du bilan.
func adsVerseInstant(b *adsSweepBilan, tracks map[uint32][]adsEchantillon, lives []lifeSpan,
	xuid uint64, tFilm int64, c adsClasse,
) {
	b.instantsVus++
	slots := adsSlotsDeLInstant(lives, xuid, tFilm)
	pres, ok := adsUnionFenetre(tracks, slots, tFilm-adsSweepFenetreMS, tFilm)
	if !ok {
		return
	}
	b.instantsSitues++
	b.presence.ajoute(c, pres)
	if d, ok := adsDernierMasque(tracks, slots, tFilm); ok {
		b.dernier.ajoute(c, d)
	}
	if tem, ok := adsUnionFenetre(tracks, slots, tFilm-adsSweepTemoinMS-adsSweepFenetreMS,
		tFilm-adsSweepTemoinMS); ok {
		b.temoin.ajoute(c, tem)
	}
}

// adsUnionFenetre rend l'union des masques vus dans la fenetre, et false si la fenetre ne contient
// aucun record.
func adsUnionFenetre(tracks map[uint32][]adsEchantillon, slots []uint32, debut, fin int64) (uint64, bool) {
	var out uint64
	vu := false
	for _, s := range slots {
		for _, e := range tracks[s] {
			if e.tMS < debut || e.tMS > fin {
				continue
			}
			vu = true
			out |= e.masque
		}
	}
	return out, vu
}

// adsDernierMasque rend le masque du dernier record precedant l'instant, dans la tolerance.
func adsDernierMasque(tracks map[uint32][]adsEchantillon, slots []uint32, tFilm int64) (uint64, bool) {
	var best adsEchantillon
	found := false
	for _, s := range slots {
		for _, e := range tracks[s] {
			if e.tMS > tFilm || e.tMS < tFilm-adsSweepDernierMS {
				continue
			}
			if !found || e.tMS > best.tMS {
				best, found = e, true
			}
		}
	}
	return best.masque, found
}
