package replay

// lives_export_test.go — LES QUATRE CAUSES DE FIN D'UNE VIE.
//
// Chaque cause a un mode de panne à elle, et c'est pour cela qu'elles sont distinguées :
// confondre `closure` avec `death` fabrique une mort pour un survivant (P0 de la ronde 2),
// confondre `cut` avec `death` fabrique une mort pour un joueur monté en véhicule.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/filmdec"
)

// posDe pose un point de réplication d'un slot à un instant, en microsecondes.
func posDe(slot uint32, tUS uint64) filmdec.BipedPosition {
	return filmdec.BipedPosition{Slot: slot, TimestampUS: tUS, HasWorld: true}
}

// pisteContinue pose une piste échantillonnée toutes les 100 ms de `deUS` à `aUS` inclus.
func pisteContinue(slot uint32, deUS, aUS uint64) []filmdec.BipedPosition {
	out := []filmdec.BipedPosition{}
	for t := deUS; t <= aUS; t += 100_000 {
		out = append(out, posDe(slot, t))
	}
	return out
}

// indexDe pose la table identité -> index du film, lue sans désaccord.
func indexDe(xuids ...uint64) PlayerIndexTable {
	t := PlayerIndexTable{ByXUID: map[uint64]int{}, Readings: 26}
	for i, x := range xuids {
		t.ByXUID[x] = i
	}
	return t
}

// causesParXUID indexe les causes rendues, par joueur, dans l'ordre.
func causesParXUID(vies []VieNommee) map[uint64][]string {
	out := map[uint64][]string{}
	for _, v := range vies {
		out[v.XUID] = append(out[v.XUID], v.Cause)
	}
	return out
}

// TestViesNommees_MortPuisFinDeFilm — LE CAS NOMINAL, et la priorité du nommage sur la
// structure.
//
// Le slot 1 vit de 0 à 10 s, meurt (le fil des morts l'apparie), puis réapparaît de 20 à 30 s
// et le film s'arrête. La PREMIÈRE vie sort `death` alors que la découpe l'avait fermée sur un
// trou de 10 s : sans la priorité, toute mort suivie d'un respawn sortirait « coupure ».
//
// LE SECOND SLOT N'EST PAS UN ORNEMENT : avec une seule mort, DEUX calages d'horloge
// apparient autant de morts (celui qui vise la fin de la première vie et celui qui vise la
// fin de la seconde), et `bestDeathOffset` en choisit un — la fixture ne prouverait alors
// rien. Deux morts à des instants différents ÉPINGLENT le calage à zéro. C'est la fixture
// ambiguë qui a fait rougir ce test à l'écriture, pas le code.
func TestViesNommees_MortPuisFinDeFilm(t *testing.T) {
	pos := append(pisteContinue(1, 0, 10_000_000), pisteContinue(1, 20_000_000, 30_000_000)...)
	pos = append(pos, pisteContinue(2, 0, 25_000_000)...)
	morts := []Death{{XUID: 111, TimeMS: 10_000}, {XUID: 222, TimeMS: 25_000}}
	rep := BuildIdentityRegistry(IdentityInput{Positions: pos, Deaths: morts, PlayerIndices: indexDe(111, 222)})

	vies := rep.ViesNommees()
	got := causesParXUID(vies)[111]
	if len(got) != 1 {
		t.Fatalf("vies nommees de 111 = %v, attendu 1 — sa SECONDE vie est anonyme (aucune "+
			"mort ne la nomme, aucune fermeture ne la deduit), et une vie anonyme ne sort "+
			"pas. Rapport %+v", got, rep)
	}
	if got[0] != CauseVieMort {
		t.Errorf("premiere vie = %q, attendu %q : le fil des morts apparie sa fin, et le "+
			"nommage PRIME sur le trou de reapparition", got[0], CauseVieMort)
	}
}

// TestViesNommees_HorlogeDuMatch — LES INSTANTS SORTENT SUR L'HORLOGE DU MATCH, celle de
// `match_kill_events.time_ms`.
//
// C'EST LA CONDITION DE LA JOINTURE. Le film et le fil des morts ont des horloges décalées ;
// publier des millisecondes de film rendrait les deux tables joignables avec n'importe quoi,
// et l'erreur serait un décalage CONSTANT — invisible à l'œil, fatale au résultat. Ici le
// film démarre 4 s avant le match : la vie doit sortir à 0, pas à 4 000.
func TestViesNommees_HorlogeDuMatch(t *testing.T) {
	const decalageMS = 4_000
	pos := pisteContinue(1, decalageMS*1000, decalageMS*1000+10_000_000)
	morts := []Death{{XUID: 111, TimeMS: 10_000}}
	rep := BuildIdentityRegistry(IdentityInput{Positions: pos, Deaths: morts, PlayerIndices: indexDe(111)})

	vies := rep.ViesNommees()
	if len(vies) == 0 {
		t.Fatalf("aucune vie nommee — rapport %+v", rep)
	}
	if rep.DeathOffsetMS() != decalageMS {
		t.Fatalf("DeathOffsetMS = %d, attendu %d : la fixture cale le film 4 s avant le match",
			rep.DeathOffsetMS(), decalageMS)
	}
	if vies[0].DebutMS != 0 {
		t.Errorf("debut = %d ms, attendu 0 : l'horloge du FILM (4000) n'est pas celle du MATCH",
			vies[0].DebutMS)
	}
	if vies[0].FinMS != 10_000 {
		t.Errorf("fin = %d ms, attendu 10000 (l'instant de la mort au journal)", vies[0].FinMS)
	}
}

// TestViesNommees_CoupureSansMort — UN TROU QUE RIEN N'EXPLIQUE EST UNE COUPURE, PAS UNE MORT.
//
// Le slot 1 disparaît 10 s au milieu de la partie sans qu'aucune mort ne l'apparie : c'est la
// signature d'un embarquement en véhicule, où le biped cesse d'être répliqué alors que le
// joueur est bien vivant. Le sortir en `death` fabriquerait une mort qui n'a pas eu lieu.
func TestViesNommees_CoupureSansMort(t *testing.T) {
	// Deux slots : le 1 est coupé au milieu, le 2 meurt pour que le pont existe.
	pos := append(pisteContinue(1, 0, 5_000_000), pisteContinue(1, 15_000_000, 20_000_000)...)
	pos = append(pos, pisteContinue(2, 0, 20_000_000)...)
	morts := []Death{{XUID: 222, TimeMS: 20_000}}
	rep := BuildIdentityRegistry(IdentityInput{Positions: pos, Deaths: morts, PlayerIndices: indexDe(111, 222)})

	for _, v := range rep.ViesNommees() {
		if v.Cause == CauseVieMort && v.FinMS == 5_000 {
			t.Fatalf("la vie coupee a 5 s sort %q : un trou de replication n'est pas une mort",
				v.Cause)
		}
	}
}

// TestViesNommees_AucuneVieAnonyme — une vie sans identité ne sort pas.
//
// L'ÉCRIRE SOUS UN XUID NUL FABRIQUERAIT UN JOUEUR : la table serait jointe sur `0`, et
// chaque film y ajouterait des vies attribuées au même fantôme.
func TestViesNommees_AucuneVieAnonyme(t *testing.T) {
	pos := append(pisteContinue(1, 0, 10_000_000), pisteContinue(7, 0, 10_000_000)...)
	morts := []Death{{XUID: 111, TimeMS: 10_000}}
	rep := BuildIdentityRegistry(IdentityInput{Positions: pos, Deaths: morts, PlayerIndices: indexDe(111)})

	for _, v := range rep.ViesNommees() {
		if v.XUID == 0 {
			t.Fatalf("une vie anonyme est sortie : %+v", v)
		}
	}
}

// TestViesNommees_SansPont_RienNeSort — sans fil des morts, il n'y a pas de calage d'horloge,
// donc pas d'instants de match. Publier des vies calées sur zéro les rendrait joignables avec
// n'importe quoi.
func TestViesNommees_SansPont_RienNeSort(t *testing.T) {
	rep := BuildIdentityRegistry(IdentityInput{Positions: pisteContinue(1, 0, 10_000_000), Deaths: nil, PlayerIndices: indexDe(111)})
	if got := rep.ViesNommees(); len(got) != 0 {
		t.Fatalf("vies = %+v, attendu aucune sans fil des morts", got)
	}
}

// TestViesNommees_OrdreTotal — deux passes du même film écrivent les mêmes lignes dans le même
// ordre. Sans tri total, l'ordre suivrait celui des slots, un détail d'implémentation.
func TestViesNommees_OrdreTotal(t *testing.T) {
	pos := append(pisteContinue(3, 0, 10_000_000), pisteContinue(1, 0, 10_000_000)...)
	morts := []Death{{XUID: 111, TimeMS: 10_000}, {XUID: 222, TimeMS: 10_000}}
	rep := BuildIdentityRegistry(IdentityInput{Positions: pos, Deaths: morts, PlayerIndices: indexDe(111, 222)})

	vies := rep.ViesNommees()
	for i := 1; i < len(vies); i++ {
		if vieAvant(vies[i], vies[i-1]) {
			t.Fatalf("vies mal triees en %d : %+v puis %+v", i, vies[i-1], vies[i])
		}
	}
}

// TestViesNommees_LesDeuxAxesSontOrthogonaux — LA LEÇON DU P0, FIGÉE.
//
// Une vie nommée par FERMETURE ne doit jamais sortir « morte ». La fermeture répond à « à qui
// appartient ce corps » ; elle ne dit rien de la façon dont la vie s'est terminée. Fondre les
// deux dans un seul champ obligeait à choisir entre nommer le joueur et dire qu'il a survécu,
// et le choix fait le comptait mort.
//
// FIXTURE RÉELLE, PAS TROIS LITTÉRAUX (Q8, 2026-09-07) : la version précédente construisait
// trois `VieNommee{}` à la main et vérifiait une implication sur ces littéraux — elle ne
// prouvait donc RIEN sur `ResolveSlotXUID`/`closeByRespawn`, seulement que le test savait
// écrire ce qu'il voulait lire. Celle-ci fait tourner le VRAI pont : deux vies de 111
// calibrent la fenêtre de réapparition (même gap que TestFermetureBAttribueSurUneSeuleMort-
// DansLaFenetre, 8 000 ms), puis une troisième vie ANONYME (slot 2, jamais nommée par une
// mort — aucune mort ne finit près de 21 000 ms) tombe dans cette fenêtre après la mort de
// 222 et est nommée par FERMETURE B (`closeByRespawn`, fire = nil : `ResolveSlotXUID` n'a
// aucun flux de tirs). Le film ne coupe jamais ce slot : sa vie court jusqu'au bout de ce
// qu'il montre, donc `film_end` — exactement le survivant que le P0 de la ronde 2 confondait
// avec un mort.
func TestViesNommees_LesDeuxAxesSontOrthogonaux(t *testing.T) {
	pos := pisteContinue(1, 0, 1_000_000)                          // 111, meurt a 1 000 ms
	pos = append(pos, pisteContinue(3, 9_000_000, 10_000_000)...)  // 111 reapparait, meurt a 10 000 ms
	pos = append(pos, pisteContinue(2, 20_000_000, 21_000_000)...) // ANONYME : personne ne le nomme par une mort
	morts := []Death{
		{XUID: 111, TimeMS: 1_000},
		{XUID: 111, TimeMS: 10_000},
		{XUID: 222, TimeMS: 12_000}, // ne termine aucune vie : candidat libre pour la fermeture B
	}
	rep := BuildIdentityRegistry(IdentityInput{Positions: pos, Deaths: morts, PlayerIndices: indexDe(111, 222)})

	vies := rep.ViesNommees()
	var survivant *VieNommee
	for i, v := range vies {
		if v.XUID == 222 {
			survivant = &vies[i]
		}
	}
	if survivant == nil {
		t.Fatalf("aucune vie pour 222 — la fermeture par reapparition devait nommer le slot "+
			"anonyme (20-21 s), rapport %+v", rep)
	}
	if survivant.NomPar != NomParFermeture {
		t.Fatalf("nomPar = %q, attendu %q : ce corps n'est identifie que par elimination "+
			"(aucune mort ne le termine)", survivant.NomPar, NomParFermeture)
	}
	if survivant.Cause != CauseVieFinFilm {
		t.Fatalf("%+v : une vie nommee par FERMETURE et jamais coupee ni appariee par une "+
			"mort doit sortir %q — c'est exactement le contresens du P0 de la ronde 2 "+
			"(un survivant compte mort)", *survivant, CauseVieFinFilm)
	}
}

// TestViesNommees_ToutesLesCausesSontAtteignables — SENTINELLE ANTI-ÉNUM MORTE.
//
// Une valeur d'enum qu'aucun chemin ne produit est pire qu'absente : elle laisse croire que la
// table sait faire une distinction qu'elle ne fait pas. La version initiale de ce lot mettait
// `closure` DANS `end_cause` — et rendait du même coup `film_end` et `cut` INATTEIGNABLES,
// puisqu'une vie n'est nommée que par une mort ou par une fermeture. Ce test constate que les
// trois causes de fin ont chacune un site d'écriture vivant.
func TestViesNommees_ToutesLesCausesSontAtteignables(t *testing.T) {
	// `cut` et `film_end` : la découpe seule les produit.
	brutes := buildLifeSpans(indexBySlot(append(
		pisteContinue(1, 0, 5_000_000), pisteContinue(1, 15_000_000, 20_000_000)...)))
	vues := map[string]bool{}
	for _, l := range brutes {
		vues[l.cause] = true
	}
	if !vues[CauseVieCoupure] || !vues[CauseVieFinFilm] {
		t.Fatalf("causes vues a la decoupe = %v, attendu au moins %q et %q",
			vues, CauseVieCoupure, CauseVieFinFilm)
	}
	// `death` : le nommage par le fil des morts.
	rep := BuildIdentityRegistry(IdentityInput{Positions: pisteContinue(1, 0, 10_000_000),
		Deaths: []Death{{XUID: 111, TimeMS: 10_000}}, PlayerIndices: indexDe(111)})
	trouve := false
	for _, v := range rep.ViesNommees() {
		if v.Cause == CauseVieMort {
			trouve = true
		}
	}
	if !trouve {
		t.Fatalf("aucune vie de cause %q : le nommage par le fil ne pose plus la cause", CauseVieMort)
	}
}
