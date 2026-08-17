package replay

// ground_weapon_pickup_owner_test.go — ITEM 2.5 : L'ORACLE DU RAMASSAGE SUIT LE JOUEUR, PAS LE
// SLOT DE VIE (plan .ai/V7.5/replay2d/PLAN_ARMES_AU_SOL_2E_LECTURE.md, arbitrage superviseur
// apres la phase 2).
//
// POURQUOI CE FICHIER EXISTE. L'item 2.2 a relu le loadout du MEME slot de bipede que celui du
// passage (`o.Picker.Slot`). Or un slot est UNE VIE, pas un joueur : il migre a la reapparition
// (`offline_biped.go`), et vingt secondes separent deux images-cles — assez pour mourir. Le
// plafond structurel mesure a l'item 2.2 (55,3 % globalement, 71,2 % sur les socles) est donc un
// artefact de l'ORACLE, pas une propriete du phenomene (decouverte 10 du plan). L'item 2.5 rejoue
// le MEME oracle, au MEME seuil, sur la population que la phase 2 a qualifiee, en passant par le
// pont slot -> joueur.
//
// LE PONT EST CELUI DU CONSTRUCTEUR, PAS UN PONT MAISON. `buildOwners` (owners.go) aux MEMES
// entrees que `BuildFromFilm` : le fil des morts (`ScanFilmDeaths`), l'index de joueur lu dans
// les chunks de replication (`ScanFilmPlayerIndices` + `injectiveOrEmpty`), les evenements de tir
// (`ScanFilmFireEvents` -> `fireRefs`, dont les fermetures ont besoin) et les positions de bipede
// deja lues par l'instrument (`indexBySlot`). C'est `own.SlotXUID` que le document publie sur
// `Track.XUID` (`build.go`, `nameTracks`) : mesurer sur autre chose ne dirait rien de ce qui
// serait publie en phase 3.
//
// LES REGLES ET LE SEUIL, ECRITS AVANT LA MESURE :
//
//	population       les ramassages de SOCLE (`at_rest`) DATES — le seul sous-ensemble que la
//	                 phase 2 a qualifie (decouverte 11 : la disparition d'une arme `dropped` est
//	                 une despawn, pas un ramassage) ;
//	ramasseur        xuid = SlotXUID[slot du passage]. Absent => « SANS PONT » : compte, publie,
//	                 HORS denominateur ;
//	vie courante     a l'image-cle visee, le slot du MEME xuid qui y porte un loadout — quel
//	                 qu'il soit, le meme ou un nouveau apres reapparition. Si le xuid en a
//	                 PLUSIEURS a cette image-cle, le plus petit est retenu et le cas est COMPTE
//	                 et publie : il ne devrait pas exister, et le taire le rendrait invisible ;
//	loadout observable   un tel slot existe. Sinon « SANS LOADOUT OBSERVABLE » (mort a
//	                 l'image-cle, aucune vie) : compte, publie, HORS denominateur ;
//	DENOMINATEUR     ramassages de socle DATES, a pont, ayant une image-cle suivante, et dont le
//	                 xuid a un loadout OBSERVABLE a cette image-cle ;
//	accord           la famille ramassee figure dans ce loadout ;
//	temoin           un AUTRE xuid tire au sort parmi ceux qui ont un loadout observable a la
//	                 MEME image-cle — meme regle, meme instant, tirage fige ;
//	controle NOUVEAU a la DERNIERE image-cle qui PRECEDE la date du ramassage, la vie courante du
//	                 MEME xuid ne porte PAS la famille. Publie SANS seuil (le plan ne lui en
//	                 donne pas) ; un accord obtenu sur une arme deja portee ne prouve rien.
//
//	GATE 2.5   ACCORD >= 90 % sur CE denominateur => le ramasseur (xuid) est publiable en
//	           phase 3 ; sinon `[!]`, et le ramasseur vaut `null` au document. Le seuil est celui
//	           du gate 2, NON rebaisse.
//
// LECTURE SEULE, aucune base : le pont se prend par les memes lectures de film que le
// constructeur. Tourne sous la garde `GW_PICKUP`, dans le meme processus et le meme decodage que
// les items 2.1 a 2.4 — un seul film par processus.

import (
	"fmt"
	"math/rand"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// gwPickup25Tally compte l'oracle par joueur sur les ramassages de socle d'un film. Chaque
// compteur ecarte du denominateur est publie a part : un denominateur qui retrecit sans dire
// pourquoi est une mesure qu'on ne peut pas juger.
type gwPickup25Tally struct {
	// Socles : ramassages de socle DATES, avant tout ecartement. C'est la population du plan.
	Socles int
	// SansPont / SansKF / SansLoadout : les trois ecartements, comptes separement.
	SansPont, SansKF, SansLoadout int
	// Denom / Accord : le denominateur du GATE 2.5 et son accord.
	Denom, Accord int
	// WitDenom / WitAccord : le temoin (un AUTRE joueur a la meme image-cle).
	WitDenom, WitAccord int
	// MultiSlot : xuid portant un loadout sur PLUSIEURS slots a la meme image-cle (anomalie).
	MultiSlot int
	// BeforeDenom / Already / NewDenom / AgreeNew : le controle « ramassage NOUVEAU ».
	BeforeDenom, Already, NewDenom, AgreeNew int
}

// gwPickup25Ctx porte ce dont l'oracle par joueur a besoin : le film deja lu, le pont du
// constructeur, et le tirage du temoin.
type gwPickup25Ctx struct {
	f   *gwPickupFilm
	br  map[uint32]uint64
	rng *rand.Rand
}

// gwPickupReport25 publie l'ORACLE PAR JOUEUR — l'item 2.5 et son gate.
func gwPickupReport25(t *testing.T, dir string, f *gwPickupFilm, objs []gwPickupObject) {
	t.Helper()
	// LE TIRAGE DU TEMOIN A SA PROPRE SOURCE, a la MEME graine. Partager `f.rng` ferait dependre
	// le temoin de 2.5 du nombre exact de tirages consommes par les items 2.1 et 2.2 : le jour ou
	// l'un d'eux change d'un tirage, ce temoin changerait sans que rien ne le dise.
	c := &gwPickup25Ctx{
		f: f, br: gwPickupOwners(t, dir, f),
		rng: rand.New(rand.NewSource(gwPickupWitnessSeed)), //nolint:gosec // temoin reproductible
	}
	var a gwPickup25Tally
	for _, o := range objs {
		if o.Status != gwPickupStatusDated || gwPickupSet(o) != gwPickupSetAtRest {
			continue
		}
		a.Socles++
		c.add(o, &a)
	}
	t.Logf("2.5 ORACLE PAR JOUEUR [at_rest] — ramassages de socle dates %d · SANS PONT %s ·"+
		" sans image-cle suivante %d · SANS LOADOUT OBSERVABLE %s · DENOMINATEUR %d ·"+
		" ACCORD %s · TEMOIN (autre joueur, meme image-cle) %s · rapport %s",
		a.Socles, gwPadsPart(a.SansPont, a.Socles), a.SansKF,
		gwPadsPart(a.SansLoadout, a.Socles), a.Denom, gwPadsPart(a.Accord, a.Denom),
		gwPadsPart(a.WitAccord, a.WitDenom), gwPickup25Ratio(a))
	t.Logf("2.5 CONTROLE NOUVEAU [at_rest] — loadout observable a l'image-cle PRECEDENTE %d ·"+
		" portait DEJA la famille %s · ramassages NOUVEAUX %d · ACCORD sur les NOUVEAUX %s ·"+
		" xuid a PLUSIEURS slots a la meme image-cle %d",
		a.BeforeDenom, gwPadsPart(a.Already, a.BeforeDenom), a.NewDenom,
		gwPadsPart(a.AgreeNew, a.NewDenom), a.MultiSlot)
}

// add compte UN ramassage de socle date. Les trois ecartements sont exclusifs et ordonnes :
// sans pont, puis sans image-cle suivante, puis sans loadout observable.
func (c *gwPickup25Ctx) add(o gwPickupObject, a *gwPickup25Tally) {
	xuid := c.br[o.Picker.Slot]
	if xuid == 0 {
		a.SansPont++
		return
	}
	kf, ok := gwPickupNextAfter(c.f.kfTimes, o.Picker.TUS)
	if !ok {
		a.SansKF++
		return
	}
	fams, obs, multi := gwPickupXUIDLoadout(c.f, c.br, kf, xuid)
	if multi {
		a.MultiSlot++
	}
	if !obs {
		a.SansLoadout++
		return
	}
	a.Denom++
	porte := gwPickupHasFamily(fams, o.Appar.Family)
	if porte {
		a.Accord++
	}
	c.witness(kf, xuid, o.Appar.Family, a)
	c.before(o, xuid, porte, a)
}

// witness tire au sort un AUTRE joueur parmi ceux qui ont un loadout observable a la MEME
// image-cle, et dit s'il porte la famille ramassee. Sans lui, un accord eleve ne se
// distinguerait pas de la popularite de l'arme.
func (c *gwPickup25Ctx) witness(kf, xuid uint64, fam string, a *gwPickup25Tally) {
	all := gwPickupXUIDsAt(c.f, c.br, kf)
	cand := make([]uint64, 0, len(all))
	for _, x := range all {
		if x != xuid {
			cand = append(cand, x)
		}
	}
	if len(cand) == 0 {
		return
	}
	fams, obs, _ := gwPickupXUIDLoadout(c.f, c.br, kf, cand[c.rng.Intn(len(cand))])
	if !obs {
		return
	}
	a.WitDenom++
	if gwPickupHasFamily(fams, fam) {
		a.WitAccord++
	}
}

// before est le controle « ramassage NOUVEAU » : a la DERNIERE image-cle qui PRECEDE la date du
// ramassage, la vie courante du MEME xuid portait-elle deja la famille ? Le denominateur est
// celui des cas ou ce loadout anterieur est OBSERVABLE — un joueur mort a l'image-cle
// precedente ne dit rien, ni dans un sens ni dans l'autre.
func (c *gwPickup25Ctx) before(o gwPickupObject, xuid uint64, porte bool, a *gwPickup25Tally) {
	i := sort.Search(len(c.f.kfTimes), func(i int) bool { return c.f.kfTimes[i] >= o.Picker.TUS })
	if i == 0 {
		return
	}
	fams, obs, _ := gwPickupXUIDLoadout(c.f, c.br, c.f.kfTimes[i-1], xuid)
	if !obs {
		return
	}
	a.BeforeDenom++
	if gwPickupHasFamily(fams, o.Appar.Family) {
		a.Already++
		return
	}
	a.NewDenom++
	if porte {
		a.AgreeNew++
	}
}

// gwPickup25Ratio rend le rapport accord / temoin — la seule lecture qui dise si l'oracle
// mesure quelque chose. « - » quand l'un des deux denominateurs est vide ou le temoin nul.
func gwPickup25Ratio(a gwPickup25Tally) string {
	if a.Denom == 0 || a.WitDenom == 0 || a.WitAccord == 0 {
		return "-"
	}
	r := (float64(a.Accord) / float64(a.Denom)) / (float64(a.WitAccord) / float64(a.WitDenom))
	return fmt.Sprintf("%.1f", r)
}

// gwPickupOwners construit le pont slot -> joueur PAR LE CHEMIN DU CONSTRUCTEUR et publie de
// quoi le juger. Aucune degradation silencieuse : une lecture manquante est journalisee, et le
// pont vide qui en resulte se lit alors dans « SANS PONT ».
func gwPickupOwners(t *testing.T, dir string, f *gwPickupFilm) map[uint32]uint64 {
	t.Helper()
	deaths, err := ScanFilmDeaths(dir)
	if err != nil {
		t.Logf("2.5 PONT — fil des morts illisible (%v) : pont VIDE, tout sera « sans pont »", err)
		return map[uint32]uint64{}
	}
	idx, err := ScanFilmPlayerIndices(dir, rosterFromDeaths(deaths))
	if err != nil {
		t.Logf("2.5 PONT — index de joueur illisible (%v) : pont VIDE", err)
		return map[uint32]uint64{}
	}
	table, collisions := injectiveOrEmpty(idx)
	fire, err := filmdec.ScanFilmFireEvents(dir)
	if err != nil {
		t.Logf("2.5 PONT — events de tir illisibles (%v) : fermeture A privee de sa source", err)
		fire = nil
	}
	own := buildOwners(indexBySlot(f.positions), deaths, table, fireRefs(fire))
	t.Logf("2.5 PONT (constructeur) — morts %d · slots ponts %d · vies nommees %d/%d ·"+
		" par lecture %d · fermetures tir %d / reapparition %d (contestees %d, refusees %d) ·"+
		" lectures d'index %d · desaccords d'index %d · collisions de slot %d ·"+
		" index non injectif %d · joueurs distincts %d",
		len(deaths), len(own.SlotXUID), own.DeathsNamed, own.LivesTotal, own.FromDeaths,
		own.Closures.byShot, own.Closures.byRespawn, own.Closures.contested,
		own.Closures.refused, own.IndexReadings, own.IndexDisagreements, own.SlotCollisions,
		collisions, gwPickup25Distinct(own.SlotXUID))
	return own.SlotXUID
}

// gwPickupXUIDLoadout rend le loadout de la VIE COURANTE de `xuid` a l'image-cle `kf` : les
// familles portees par le slot de CE joueur qui figure a cette image-cle, quel que soit ce slot.
// Le deuxieme retour dit si le loadout est OBSERVABLE ; le troisieme signale que le xuid en a
// PLUSIEURS a la meme image-cle — le plus petit est alors retenu, et l'appelant le compte.
func gwPickupXUIDLoadout(
	f *gwPickupFilm, br map[uint32]uint64, kf, xuid uint64,
) ([]string, bool, bool) {
	at := f.loadouts[kf]
	best, found, multi := uint32(0), false, false
	for s := range at {
		if xuid == 0 || br[s] != xuid {
			continue
		}
		if found {
			multi = true
			if s > best {
				continue
			}
		}
		best, found = s, true
	}
	if !found {
		return nil, false, false
	}
	return at[best], true, multi
}

// gwPickupXUIDsAt rend, en ordre TOTAL, les xuid qui ont un loadout observable a l'image-cle.
// L'ordre est impose avant tout tirage : une map Go s'itere au hasard, et un temoin dont
// l'ordre varie d'une execution a l'autre n'est pas un temoin.
func gwPickupXUIDsAt(f *gwPickupFilm, br map[uint32]uint64, kf uint64) []uint64 {
	seen := map[uint64]bool{}
	out := make([]uint64, 0, len(f.loadouts[kf]))
	for s := range f.loadouts[kf] {
		x := br[s]
		if x == 0 || seen[x] {
			continue
		}
		seen[x] = true
		out = append(out, x)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// gwPickup25Distinct compte les joueurs distincts que le pont nomme.
func gwPickup25Distinct(br map[uint32]uint64) int {
	seen := map[uint64]bool{}
	for _, x := range br {
		seen[x] = true
	}
	return len(seen)
}

// --- TESTS DE LA REGLE (sans garde : ils tournent avec le paquet) ------------------------

// TestGwPickupLoadoutSuitLeJoueurAuTraversDuRespawn : le coeur de l'item 2.5. Le ramasseur a
// ete vu sur le slot 10 ; a l'image-cle suivante il est mort et revenu sur le slot 20. L'oracle
// par SLOT ne verrait rien ; l'oracle par JOUEUR lit le loadout du slot 20.
func TestGwPickupLoadoutSuitLeJoueurAuTraversDuRespawn(t *testing.T) {
	f := &gwPickupFilm{loadouts: map[uint64]map[uint32][]string{
		100: {20: {"S7 Sniper", "MA40 AR"}, 30: {"BR75"}},
	}}
	br := map[uint32]uint64{10: 777, 20: 777, 30: 888}
	fams, obs, multi := gwPickupXUIDLoadout(f, br, 100, 777)
	if !obs || multi || !gwPickupHasFamily(fams, "S7 Sniper") {
		t.Fatalf("le loadout de la vie courante (slot 20) attendu : %v obs=%t multi=%t",
			fams, obs, multi)
	}
	if _, obs, _ := gwPickupXUIDLoadout(f, br, 100, 999); obs {
		t.Fatalf("un xuid absent de l'image-cle n'a PAS de loadout observable")
	}
	if _, obs, _ := gwPickupXUIDLoadout(f, br, 100, 0); obs {
		t.Fatalf("un ramassage sans pont (xuid 0) ne doit jamais trouver de loadout")
	}
}

// TestGwPickupPlusieursSlotsDuMemeJoueurSontSignales : deux slots du meme xuid a la meme
// image-cle ne se fusionnent pas en silence — le plus petit gagne, et le cas est signale.
func TestGwPickupPlusieursSlotsDuMemeJoueurSontSignales(t *testing.T) {
	f := &gwPickupFilm{loadouts: map[uint64]map[uint32][]string{
		100: {12: {"Mangler"}, 40: {"Needler"}},
	}}
	br := map[uint32]uint64{12: 777, 40: 777}
	fams, obs, multi := gwPickupXUIDLoadout(f, br, 100, 777)
	if !obs || !multi || !gwPickupHasFamily(fams, "Mangler") {
		t.Fatalf("le plus petit slot (12) retenu et le cas signale : %v obs=%t multi=%t",
			fams, obs, multi)
	}
}

// TestGwPickupXUIDsAtEstTotalementOrdonne : le temoin tire dans une liste dont l'ordre ne
// depend pas de l'iteration d'une map, et les slots hors pont n'y entrent pas.
func TestGwPickupXUIDsAtEstTotalementOrdonne(t *testing.T) {
	f := &gwPickupFilm{loadouts: map[uint64]map[uint32][]string{
		100: {1: {"BR75"}, 2: {"MA40 AR"}, 3: {"Mangler"}, 4: {"Needler"}},
	}}
	br := map[uint32]uint64{1: 900, 2: 100, 3: 500} // le slot 4 n'est pas ponte
	got := gwPickupXUIDsAt(f, br, 100)
	if len(got) != 3 || got[0] != 100 || got[1] != 500 || got[2] != 900 {
		t.Fatalf("xuid tries, slots hors pont ecartes : %v", got)
	}
}
