package replay

import (
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// lives_test.go — le repli « nommer la vie par la mort qui la termine ».
//
// Ces tests portent sur les PROPRIÉTÉS qui fondent le repli, pas sur des valeurs de sortie
// figées : le découpage en vies, le calage d'horloge, l'appariement, et surtout le REFUS de
// trancher quand la donnée ne le permet pas. Un test qui ne vérifierait que le cas nominal
// laisserait passer exactement les régressions qui coûtent cher ici.

// tracksOf construit l'index par slot attendu par buildLifeSpans. Le helper posAt vit
// dans shots_test.go : une seconde copie divergerait (regle du depot sur les doublons).
func tracksOf(pts ...filmdec.BipedPosition) map[uint32]slotTrack {
	return indexBySlot(pts)
}

func TestBuildLifeSpansSplitsOnGap(t *testing.T) {
	// Un même slot, deux séjours séparés par plus de lifeGapUS : deux vies.
	tr := tracksOf(
		posAt(512, 1_000_000, 0, 0, 0), posAt(512, 2_000_000, 0, 0, 0),
		posAt(512, 20_000_000, 0, 0, 0), posAt(512, 21_000_000, 0, 0, 0),
	)
	lives := buildLifeSpans(tr)
	if len(lives) != 2 {
		t.Fatalf("attendu 2 vies, obtenu %d : %+v", len(lives), lives)
	}
	if lives[0].to != 2_000_000 || lives[1].from != 20_000_000 {
		t.Errorf("bornes de vie inattendues : %+v", lives)
	}
}

func TestBuildLifeSpansKeepsContinuousTrackWhole(t *testing.T) {
	// Des échantillons rapprochés ne doivent JAMAIS être coupés : un découpage trop
	// agressif fabriquerait des vies sans mort, donc des vies jamais nommées.
	var pts []filmdec.BipedPosition
	for i := 0; i < 50; i++ {
		pts = append(pts, posAt(512, uint64(i)*16_000, 0, 0, 0))
	}
	if lives := buildLifeSpans(tracksOf(pts...)); len(lives) != 1 {
		t.Fatalf("attendu 1 vie continue, obtenu %d", len(lives))
	}
}

func TestNameLivesByDeathsJoinsOnEnd(t *testing.T) {
	// Deux vies qui se terminent à 2 s et 21 s ; deux morts aux mêmes instants, décalées
	// d'une origine. L'appariement doit rendre chaque identité à sa vie.
	tr := tracksOf(
		posAt(512, 1_000_000, 0, 0, 0), posAt(512, 2_000_000, 0, 0, 0),
		posAt(513, 20_000_000, 0, 0, 0), posAt(513, 21_000_000, 0, 0, 0),
	)
	lives := buildLifeSpans(tr)
	deaths := []Death{{XUID: 111, TimeMS: 2_000 - 500}, {XUID: 222, TimeMS: 21_000 - 500}}
	off, n := bestDeathOffset(lives, deaths)
	if n != 2 {
		t.Fatalf("attendu 2 morts appariables, obtenu %d (decalage %d)", n, off)
	}
	if got := nameLivesByDeaths(lives, deaths, off); got != 2 {
		t.Fatalf("attendu 2 vies nommees, obtenu %d", got)
	}
	byslot := map[uint32]uint64{}
	for _, l := range lives {
		byslot[l.slot] = l.xuid
	}
	if byslot[512] != 111 || byslot[513] != 222 {
		t.Errorf("identites mal posees : %+v", byslot)
	}
}

func TestNameLivesByDeathsIsDeterministic(t *testing.T) {
	// Deux vies qui finissent au MÊME instant : l'appariement doit être reproductible.
	// Sans départage explicite, l'ordre d'itération d'une map rendrait le résultat
	// instable d'une exécution à l'autre — un rejeu différent à chaque construction.
	build := func() (uint64, uint64) {
		tr := tracksOf(
			posAt(512, 1_000_000, 0, 0, 0), posAt(512, 5_000_000, 0, 0, 0),
			posAt(513, 1_000_000, 0, 0, 0), posAt(513, 5_000_000, 0, 0, 0),
		)
		lives := buildLifeSpans(tr)
		deaths := []Death{{XUID: 111, TimeMS: 5_000}, {XUID: 222, TimeMS: 5_000}}
		nameLivesByDeaths(lives, deaths, 0)
		m := map[uint32]uint64{}
		for _, l := range lives {
			m[l.slot] = l.xuid
		}
		return m[512], m[513]
	}
	a1, b1 := build()
	for i := 0; i < 20; i++ {
		if a2, b2 := build(); a2 != a1 || b2 != b1 {
			t.Fatalf("appariement non deterministe : (%d,%d) puis (%d,%d)", a1, b1, a2, b2)
		}
	}
}

func TestOwnersFromLivesRefusesToPickOnCollision(t *testing.T) {
	// Un slot dont deux vies portent des identités différentes est une contradiction : la
	// table slot -> joueur ne peut pas la représenter. On exige qu'elle soit COMPTÉE et que
	// le slot ne soit PAS publié — trancher au hasard placerait des tirs sur un innocent.
	lives := []lifeSpan{
		{slot: 512, from: 0, to: 1_000_000, xuid: 111},
		{slot: 512, from: 10_000_000, to: 11_000_000, xuid: 222},
	}
	owners, byXUID, ambigus := ownersFromLives(lives, map[uint64]int{111: 0, 222: 1})
	if len(ambigus) != 1 || !ambigus[512] {
		t.Errorf("attendu le slot 512 MARQUE ambigu, obtenu %v", ambigus)
	}
	if _, published := owners[512]; !published {
		t.Errorf("la premiere lecture doit rester ; seule la contradictoire est ecartee")
	}
	if owners[512] != 0 {
		t.Errorf("le slot doit garder la premiere identite lue, obtenu index %d", owners[512])
	}
	// Les deux tables sortent du meme parcours : elles doivent designer LE MEME joueur, sans
	// quoi un client nommerait une trace autrement que le rattachement de ses evenements.
	if byXUID[512] != 111 {
		t.Errorf("la table d'identites doit suivre la table d'index, obtenu xuid %d", byXUID[512])
	}
}

func TestNameTracksLeavesUnbridgedLivesAnonymous(t *testing.T) {
	// Une vie que le fil des morts n'a pas nommee reste SANS identite. La remplir d'un
	// « inconnu » ou du porteur d'un slot voisin serait exactement la faute qui a fait
	// supprimer le vote : mieux vaut ne rien afficher que quelque chose de faux.
	tracks := []Track{
		{Slot: 512, StartFrame: 0, EndFrame: 9},
		{Slot: 513, StartFrame: 0, EndFrame: 9},
	}
	lives := []lifeSpan{{slot: 512, from: 0, to: 1_000_000, xuid: 2533274800000001}}
	nameTracksByLives(tracks, lives, 0, 100_000)
	if tracks[0].XUID != "2533274800000001" {
		t.Errorf("la trace pontee doit porter son xuid en decimal, obtenu %q", tracks[0].XUID)
	}
	if tracks[1].XUID != "" {
		t.Errorf("la trace non pontee doit rester anonyme, obtenu %q", tracks[1].XUID)
	}
}

func TestNameTracksByLivesNamesEachOccupantOfARecycledSlot(t *testing.T) {
	// LE CAS SYLVANUS (retour user 2026-09-02) : un slot recyclé — partant remplacé par un
	// arrivant — porte DEUX vies nommées différemment. Le nommage par slot donnait tout au
	// premier ; le nommage par vie donne à chaque track l'identité de SON occupant.
	tracks := []Track{
		{Slot: 512, StartFrame: 0, EndFrame: 9},   // vie 1 : 0 à 0,9 s
		{Slot: 512, StartFrame: 80, EndFrame: 99}, // vie 2 : 8,0 à 9,9 s (après le trou)
	}
	lives := []lifeSpan{
		{slot: 512, from: 0, to: 900_000, xuid: 111},
		{slot: 512, from: 8_000_000, to: 9_900_000, xuid: 222},
	}
	nameTracksByLives(tracks, lives, 0, 100_000)
	if tracks[0].XUID != "111" || tracks[1].XUID != "222" {
		t.Errorf("chaque occupant doit garder SA vie : obtenu %q puis %q",
			tracks[0].XUID, tracks[1].XUID)
	}
}

func TestBuildRosterIsSortedAndStable(t *testing.T) {
	// L'ordre d'iteration d'une map Go est aleatoire : sans tri, l'artefact changerait
	// d'octets a chaque build sans changer de contenu, et deviendrait indiffable.
	idx := PlayerIndexTable{ByXUID: map[uint64]int{2533274800000003: 2, 2533274800000001: 0,
		2533274800000002: 1}}
	first := buildRoster(idx, nil, nil)
	if len(first) != 3 || first[0].FilmIndex != 0 || first[2].FilmIndex != 2 {
		t.Fatalf("roster mal trie : %+v", first)
	}
	if first[0].XUID != "2533274800000001" {
		t.Errorf("xuid attendu en decimal, obtenu %q", first[0].XUID)
	}
	for i := 0; i < 20; i++ {
		if got := buildRoster(idx, nil, nil); got[0].XUID != first[0].XUID || got[2].XUID != first[2].XUID {
			t.Fatalf("roster non reproductible entre deux appels : %+v puis %+v", first, got)
		}
	}
	if buildRoster(PlayerIndexTable{}, nil, nil) != nil {
		t.Errorf("sans table d'index, pas de roster invente")
	}
}

func TestBuildOwnersPublishesNothingWithoutDeaths(t *testing.T) {
	// SANS FIL DES MORTS, LE PONT EST VIDE — et c'est le comportement voulu depuis le retrait
	// du repli voté. Ce test est le garde-fou de cette décision : si un jour une seconde
	// source réapparaît « pour améliorer la couverture », il tombera. Un rejeu muet se voit ;
	// un rejeu qui pose des tirs sur le mauvais joueur ne se voit pas.
	tr := tracksOf(posAt(512, 1_000_000, 0, 0, 90), posAt(512, 2_000_000, 0, 0, 90),
		posAt(513, 1_000_000, 5, 5, 270), posAt(513, 2_000_000, 5, 5, 270))
	rep := buildOwners(tr, nil, PlayerIndexTable{ByXUID: map[uint64]int{111: 4}, Readings: 26}, nil)
	if len(rep.Owner) != 0 {
		t.Errorf("sans morts, AUCUN slot ne doit etre attribue : %+v", rep.Owner)
	}
	if rep.FromDeaths != 0 {
		t.Errorf("sans morts, la lecture ne produit rien : %+v", rep)
	}

	// SANS TABLE D'INDEX non plus, rien n'est publié : le pont a DEUX maillons lus, et il lui
	// faut les deux.
	if rep3 := buildOwners(tr, []Death{{XUID: 111, TimeMS: 2_000}}, PlayerIndexTable{}, nil); len(rep3.Owner) != 0 {
		t.Errorf("sans table d'index, AUCUN slot ne doit etre attribue : %+v", rep3.Owner)
	}

	// Avec les deux maillons, la lecture nomme le slot.
	deaths := []Death{{XUID: 111, TimeMS: 2_000}}
	rep2 := buildOwners(tr, deaths, PlayerIndexTable{ByXUID: map[uint64]int{111: 4}, Readings: 26}, nil)
	if rep2.DeathsNamed == 0 {
		t.Fatalf("attendu au moins une vie nommee, obtenu %d", rep2.DeathsNamed)
	}
	// TOUT le pont vient de la lecture : c'est l'invariant que le verdict controle aussi.
	if rep2.FromDeaths != len(rep2.Owner) {
		t.Errorf("le pont doit venir ENTIEREMENT de la lecture : %+v", rep2)
	}
}

func TestBuildRosterPublishesDeclaredBots(t *testing.T) {
	// Les bots de BOT_METADATA entrent au roster SANS xuid, avec leur nom suffixé — et un
	// bot dont l'index est déjà tenu par un humain est refusé : le fil des morts (une
	// lecture par identité) l'emporte sur un paquet de métadonnées.
	idx := PlayerIndexTable{ByXUID: map[uint64]int{2533274800000001: 0}}
	bots := []BotIdentity{
		{FilmIndex: 8, Name: "343 Aloysius [bot]"},
		{FilmIndex: 0, Name: "343 Conflit [bot]"}, // index tenu par l'humain -> refusé
		{FilmIndex: 9, Name: ""},                  // sans nom -> rien à publier
	}
	got := buildRoster(idx, nil, bots)
	if len(got) != 2 {
		t.Fatalf("attendu humain + 1 bot, obtenu %+v", got)
	}
	if got[1].XUID != "" || !got[1].Bot || got[1].Name != "343 Aloysius [bot]" || got[1].FilmIndex != 8 {
		t.Errorf("entrée bot inattendue : %+v", got[1])
	}
}

func TestNameBotTracksNamesBridgedSlotsOnly(t *testing.T) {
	// Une vie de bot n'est nommée que si le PONT attribue son slot à l'index du bot ; une
	// vie déjà nommée par un xuid n'est JAMAIS écrasée (slot recyclé humain -> bot).
	tracks := []Track{
		{Slot: 600},                           // pont -> index 8 (bot)
		{Slot: 601},                           // hors pont : reste anonyme
		{Slot: 602, XUID: "2533274800000001"}, // humain nommé : intouchable
	}
	owner := map[uint32]int{600: 8, 602: 8}
	nameBotTracks(tracks, owner, []BotIdentity{{FilmIndex: 8, Name: "343 Aloysius [bot]"}})
	if tracks[0].Bot != "343 Aloysius [bot]" {
		t.Errorf("la vie pontée vers l'index du bot doit porter son nom, obtenu %q", tracks[0].Bot)
	}
	if tracks[1].Bot != "" || tracks[2].Bot != "" || tracks[2].XUID == "" {
		t.Errorf("vies hors pont ou déjà nommées inchangées, obtenu %+v", tracks)
	}
}

// TestDeuxBotsSurUnMemeSiegeNeSecrasentPas — LE CORRECTIF P1-8 (audit du 2026-09-06).
//
// `nameByIndex[b.FilmIndex] = b.Name` : le DERNIER balayé gagnait, sans ordre garanti et sans
// aucun rapport avec la chronologie du remplacement — le pion du rejeu affichait un gamertag
// faux sur des pans entiers du match. Le cas est mesuré (RE_LOG 7ter.62 : « 343 Aloysius » puis
// « 343 PardonMy », les deux déclarant slot=8) et la doctrine est écrite quatre fonctions plus
// haut, dans `buildRoster`, qui la respecte.
//
// UN SIÈGE AMBIGU N'EST PLUS NOMMÉ PAR SIÈGE : ses vies restent libres pour `attributeSuccessions`,
// qui seul sait les départager par l'instant de bascule lu dans la base.
//
// MUTATION : revenir à `nameByIndex[b.FilmIndex] = b.Name` rougit — la piste du siège partagé
// porte un nom arbitraire au lieu de rester libre.
func TestDeuxBotsSurUnMemeSiegeNeSecrasentPas(t *testing.T) {
	bots := []BotIdentity{
		{FilmIndex: 8, Name: "343 Aloysius [bot]"},
		{FilmIndex: 8, Name: "343 PardonMy [bot]"},
		{FilmIndex: 9, Name: "343 Razzle [bot]"},
	}
	noms, partages := botNamesBySeat(bots)
	if partages != 1 {
		t.Errorf("sieges partages = %d, attendu 1", partages)
	}
	if _, nomme := noms[8]; nomme {
		t.Errorf("le siege 8 est declare par DEUX bots : il ne doit pas etre nomme par siege, "+
			"obtenu %q", noms[8])
	}
	if noms[9] != "343 Razzle [bot]" {
		t.Errorf("le siege 9 n'a qu'un bot : il doit rester nomme, obtenu %q", noms[9])
	}

	// Bout en bout : la piste du siège ambigu reste LIBRE (ni xuid, ni bot), celle du siège
	// mono-bot est nommée.
	tracks := []Track{{Slot: 100}, {Slot: 200}}
	nameBotTracks(tracks, map[uint32]int{100: 8, 200: 9}, bots)
	if tracks[0].Bot != "" {
		t.Errorf("piste du siege partage : %q, attendu libre pour le relais", tracks[0].Bot)
	}
	if tracks[1].Bot != "343 Razzle [bot]" {
		t.Errorf("piste du siege mono-bot : %q, attendu le nom du bot", tracks[1].Bot)
	}
}

// TestUnMemeBotDeclareDeuxFoisResteNomme — LA CONTRE-ÉPREUVE : l'abstention porte sur des NOMS
// distincts, pas sur des déclarations répétées. `killsource.loadBotMeta` déduplique sur (Slot,
// BotID) : un même bot peut arriver deux fois avec le même nom, et ce n'est pas une ambiguïté.
func TestUnMemeBotDeclareDeuxFoisResteNomme(t *testing.T) {
	noms, partages := botNamesBySeat([]BotIdentity{
		{FilmIndex: 8, Name: "343 Razzle [bot]"},
		{FilmIndex: 8, Name: "343 Razzle [bot]"},
	})
	if partages != 0 || noms[8] != "343 Razzle [bot]" {
		t.Errorf("meme nom deux fois : partages = %d, nom = %q ; attendu 0 et le nom",
			partages, noms[8])
	}
}
