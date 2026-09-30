package grammar

// bench_balayage_bits_test.go — LE BANC DU BALAYAGE BIT A BIT (lot J4.6 du
// PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25, decision DU-3 option S2).
//
// # POURQUOI CE BANC
//
// DU-3 a fixe D AVANCE la condition du lot S2 : les sept lecteurs de bits ecrits a la main
// dans cette couche (`readBitsAt`, `PeekBits`, `kfReadBits`, `kfReadBitsLoop`, `kfBitAt`,
// `invBitAt`, `invBits`) passent au lecteur canonique de la couche `source` SI le balayage ne
// regresse pas de plus de 5 %. Il faut donc un banc qui mesure CE balayage-la, par les VRAIS
// appelants de production de ces sept lecteurs, et rien d autre — ni chargement de film, ni
// ecriture, ni I/O dans la boucle.
//
// # CE QU IL APPELLE, ET QUEL LECTEUR CHAQUE APPEL EXERCE (mesure du 2026-09-26, par le
// profil de couverture du banc lui-meme)
//
//	ScanBipedPositions          positions du bipede (`offline_biped`)       readBitsAt
//	ScanBipedAimOnly            visee du bipede (`offline_aim_only`)        readBitsAt
//	ScanProjectiles             projectiles (`projectiles`)                 readBitsAt
//	ScanEquipmentCreations      creations d equipement                      PeekBits
//	ScanGrenadeThrows           lancers de grenade                          PeekBits
//	ScanVehicleEvents           liste d evenements en tete de paquet        PeekBits
//	ScanObjectDeaths            marche des morts d objet                    kfBitAt
//	ScanKeyframeLoadoutsMarche  armes portees aux images-cles (la cuisson)  kfReadBits, kfBitAt
//	ScanKeyframeInventory       inventaire des images-cles (la cuisson)     invBitAt, invBits
//
// `kfReadBitsLoop` n est le chemin d AUCUN appelant de production (position negative ou largeur
// > 64, cf. sa documentation) : le banc ne l atteint pas, et c est la mesure qui le dit.
//
// Ce tableau decrit l etat AVANT le lot (le binaire « avant » de la mesure A/B). Depuis, les sept
// lecteurs sont des conventions nommees de la source (`readBitsAt` -> `source.BitsStricts`,
// `PeekBits` et `invBits` -> `source.BitsTolerants`, `kfReadBits` -> `source.BitsBourres`,
// `kfBitAt` et `invBitAt` -> `source.BitAt`) ; le banc appelle les MEMES balayages, ce qui rend
// les deux binaires comparables.
//
// RESULTAT DU 2026-09-26 (machine chargee par d autres processus) : en priorite normale, la
// dispersion (14 a 25 % d ecart absolu median) depassait tout ecart cherche — mesure non
// concluante ; en priorite HAUTE, 17 paires alternees A,B : ecart median apparie -0,9 %,
// dispersion 1,3 point. Aucune regression mesurable ; le seuil de 5 % de DU-3 est tenu.
//
// # LA MATIERE
//
// La mini-bobine de `killsource` (`bobineFamilles`), PREFIXE CONTIGU du film 000d5950 : elle
// porte le registre (chunk_00) et la continuite que les balayages delta exigent — c est elle que
// le golden des familles (`golden_minibobine_test.go`) confronte deja aux memes points d entree.
// Celle de `BenchmarkKeyframeClosure` ne suffit pas : la bande de slots bipedes ne s y etablit
// pas, et `ScanBipedPositions` la refuse. La bobine est chargee UNE fois, hors de la boucle ; un
// `FilmContext` NEUF par tour, parce que le contexte memorise ses marches et qu un tour qui
// relirait le cache du tour precedent ne mesurerait plus le balayage.
//
// # LA MESURE A/B (charge de la machine)
//
//	go test -c -o avant.test.exe ./internal/games/halo_infinite/film/internal/grammar/
//	(apres la modification) go test -c -o apres.test.exe <meme paquet>
//	puis, en ALTERNANCE A,B,A,B... au moins dix fois chacun, depuis le repertoire du paquet :
//	avant.test.exe -test.run '^$' -test.bench '^BenchmarkBalayageBitABit$' -test.count 1 -test.benchtime 20x
//
// Comparer les MEDIANES, et la dispersion de chaque cote : une machine chargee par d autres
// processus fait varier une passe de plus que l ecart cherche.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// BenchmarkBalayageBitABit mesure les balayages de production qui appellent les sept lecteurs de
// bits de la couche, sur la mini-bobine contigue de `killsource`.
func BenchmarkBalayageBitABit(b *testing.B) {
	film, err := source.LoadDir(bobineFamilles, nil)
	if err != nil {
		b.Fatalf("mini-bobine versionnee illisible (%s) : %v", bobineFamilles, err)
	}
	known := catalogueDuFilm(NewFilmContext(film), film)
	if len(known) == 0 {
		b.Fatal("catalogue de familles vide : les balayages d image-cle ne mesureraient rien")
	}
	wr := profile.QuantRangeCEBiped() // bornes MESUREES du film 000d5950, comme le golden des familles
	opt := DefaultScanFilmOptions()
	opt.WorldRange = &wr
	for b.Loop() {
		balayerLesSeptLecteurs(b, NewFilmContext(film), opt, known)
	}
}

// balayerLesSeptLecteurs fait UN tour de balayage. Une erreur est fatale : un balayage qui
// refuse la bobine ne mesurerait plus son lecteur, et le banc rendrait un temps flatteur.
func balayerLesSeptLecteurs(b *testing.B, fc *FilmContext, opt ScanFilmOptions, known map[uint32]bool) {
	b.Helper()
	echec := func(nom string, err error) {
		if err != nil {
			b.Fatalf("%s : %v", nom, err)
		}
	}
	_, err := ScanBipedPositions(fc, opt)
	echec("ScanBipedPositions", err)
	_, err = ScanBipedAimOnly(fc)
	echec("ScanBipedAimOnly", err)
	_, err = ScanProjectiles(fc, opt.WorldRange)
	echec("ScanProjectiles", err)
	_, _, err = ScanEquipmentCreations(fc, opt.WorldRange)
	echec("ScanEquipmentCreations", err)
	_, err = ScanGrenadeThrows(fc)
	echec("ScanGrenadeThrows", err)
	_, err = ScanVehicleEvents(fc)
	echec("ScanVehicleEvents", err)
	_, _, err = ScanObjectDeaths(fc)
	echec("ScanObjectDeaths", err)
	_, _, err = ScanKeyframeLoadoutsMarche(fc, known)
	echec("ScanKeyframeLoadoutsMarche", err)
	_, _, err = ScanKeyframeInventory(fc, known, 0)
	echec("ScanKeyframeInventory", err)
}
