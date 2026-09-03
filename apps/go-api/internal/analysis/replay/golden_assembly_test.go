package replay

// golden_assembly_test.go — LES CHIFFRES DU CHANTIER, FIGES DANS UN DIFF LISIBLE.
//
// POURQUOI UN GOLDEN ET PAS SEULEMENT DES ASSERTIONS. Une assertion dit « 475 attendu » ; un
// golden montre CE QUI A CHANGE — quel calque, quelle cause de rejet, quel verdict, quel
// libelle d arme — dans un fichier qui se relit comme du code. Les deux coexistent ici : le
// golden porte le detail, et les tests NOMMES qui le suivent portent les chiffres publies du
// chantier, chacun avec sa phrase.
//
// AUCUN OCTET DE FILM N EST LU. Les entrees viennent de `testdata/inputs_000d5950.bin.gz`
// (cf. golden_inputs_test.go), qui les fige. Ce fichier verrouille donc l ASSEMBLAGE seul.
//
// REGENERATION (jamais d edition a la main) :
//
//	go test ./internal/analysis/replay/ -run GoldenAssembly -update

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// Les chiffres MESURES du film de reference, ecrits ici et non derives de la sortie.
//
// UNE VALEUR ATTENDUE QUI SE RECALCULE DEPUIS LE RESULTAT NE TESTE RIEN. Chacune de ces
// constantes vient d une mesure datee du chantier ; si l une bouge, c est le decodeur ou
// l assemblage qui a bouge, et le test doit le dire par son nom.
const (
	// wantShotsAttached / wantShotsAvailable : 483/519 = 93,1 %. Mesure du 2026-08-09, apres la
	// RONDE DE CORRECTION des fermetures : 484/519 avant elle, 475/519 avant les fermetures,
	// 496/519 avec le repli VOTE retire le 2026-07-28.
	//
	// LE CHIFFRE A BAISSE, ET C EST LE RESULTAT ATTENDU. La revue a montre deux deductions que
	// rien ne fondait : un candidat en trou de replication etait invisible (donc l unique
	// candidat restant n en etait pas un), et deux vies pouvaient revendiquer LA MEME mort sans
	// s exclure. Les corriger retire des attributions ; un correctif de justesse qui ferait
	// MONTER le compte serait le signal qu il a elargi au lieu de resserrer.
	//
	// CES CHIFFRES NE SE COMPARENT PAS COMME LES ETAPES D UN MEME PROGRES. 496 venait d un CHOIX
	// (4 desaccords entre sources) ; 483 vient d une DEDUCTION qui s abstient des qu il reste
	// deux candidats, et dont les refus sont comptes et publies.
	wantShotsAttached  = 483
	wantShotsAvailable = 519
	// wantClosedByShot / wantClosedByRespawn : ce que les fermetures ajoutent sur ce film.
	// La fermeture A n y ferme RIEN (ses candidates sont contestees) ; c est B qui porte les
	// 3 entrees — 5 avant la ronde de correction du 2026-08-09, dont DEUX revendiquaient la meme
	// mort qu une autre vie. Sur d autres films le partage s inverse : cf. §7.5 du verdict.
	wantClosedByShot    = 0
	wantClosedByRespawn = 3
	// wantLivesNamed / wantLivesTotal : 90 vies nommees sur 105. Les 15 restantes sont 4 vies
	// anterieures au debut reel du match et 6 survivants de fin de partie, que le film ne clot
	// par aucun evenement.
	wantLivesNamed = 90
	wantLivesTotal = 105
	// wantGrenades : 70 lancers, tous situes (65 par la naissance de leur projectile, 5 par le
	// biped de leur auteur).
	wantGrenades = 70
	// wantProjectiles : 439 trajectoires publiees.
	wantProjectiles = 439
	// wantInventory : 184 etats d inventaire publies.
	wantInventory = 184
	// wantIndexReadings : 26 chunks de replication livrent la MEME table identite -> index.
	wantIndexReadings = 26
)

// goldenAssemblyPath est la sortie figee.
func goldenAssemblyPath() string {
	return filepath.Join(goldenDir, "assembly_"+goldenFilm+".golden")
}

// buildGolden rejoue l assemblage sur les entrees figees.
func buildGolden(t *testing.T) ReplayDocument {
	t.Helper()
	g := loadGoldenInputs(t)
	opt := g.options()
	// Le catalogue vient des VRAIS mappings du titre (cf. golden_catalog_test.go) : le
	// golden fige donc aussi les libellés servis, dans les deux langues.
	opt.Labels = goldenCatalog(t)
	// L entree de catalogue de la carte (schema 8) : les tractions de grappin deQuantifient
	// leur ancre avec elle — meme source versionnee que la regeneration des entrees
	// (goldenMapQuant), donc le golden fige aussi ce calque.
	entry, err := goldenMapQuant()
	if err != nil {
		t.Fatalf("entree de catalogue de Cliffhanger illisible : %v", err)
	}
	opt.MapQuant = &entry
	return BuildFromPositions(goldenFilm, "halo_infinite", g.Positions, g.Fire, opt)
}

// TestGoldenAssembly : l assemblage rejoue rend-il toujours la meme chose ?
func TestGoldenAssembly(t *testing.T) {
	got := renderAssembly(buildGolden(t))
	path := goldenAssemblyPath()
	if *updateGolden {
		if err := os.MkdirAll(goldenDir, 0o750); err != nil {
			t.Fatalf("creation de %s : %v", goldenDir, err)
		}
		if err := os.WriteFile(path, []byte(got), 0o600); err != nil {
			t.Fatalf("ecriture de %s : %v", path, err)
		}
		t.Logf("golden reecrit : %s", path)
		return
	}
	want, err := os.ReadFile(path) //nolint:gosec // chemin fige dans le code
	if err != nil {
		t.Fatalf("golden absent : %v — regenerer avec -update", err)
	}
	if string(want) == got {
		return
	}
	t.Errorf("l assemblage a change par rapport a %s.\n%s", path, premierEcartAssembly(string(want), got))
}

// premierEcartAssembly : la premiere ligne qui differe, avec son numero. Un diff de 400 lignes
// noie l information ; ce qu on veut savoir, c est QUELLE ligne a bouge.
func premierEcartAssembly(want, got string) string {
	lw, lg := strings.Split(want, "\n"), strings.Split(got, "\n")
	for i := 0; i < len(lw) || i < len(lg); i++ {
		a, b := ligneAssembly(lw, i), ligneAssembly(lg, i)
		if a == b {
			continue
		}
		return fmt.Sprintf("premier ecart ligne %d :\n  fige   : %s\n  obtenu : %s\n"+
			"(%d ligne(s) figees, %d obtenues)", i+1, a, b, len(lw), len(lg))
	}
	return "les fichiers different sans qu aucune ligne ne differe (fin de fichier ?)"
}

func ligneAssembly(ls []string, i int) string {
	if i >= len(ls) {
		return "<absente>"
	}
	return ls[i]
}

// ---------------------------------------------------------------------------
// Les tests NOMMES : un chiffre du chantier, une phrase, un test.
// ---------------------------------------------------------------------------

// TestShotsCoverageIsFourEightyThreeOfFiveNineteen : LE CHIFFRE CENTRAL DU CHANTIER.
//
// 483 tirs rattaches sur 519 disponibles. Le denominateur compte autant que le numerateur :
// publier 483 sans dire 519 laisserait croire a l exhaustivite. Le rapport et la ventilation
// des rejets sont donc verifies ensemble.
func TestShotsCoverageIsFourEightyThreeOfFiveNineteen(t *testing.T) {
	doc := buildGolden(t)
	c := doc.Coverage.Shots
	if c.Attached != wantShotsAttached || c.Available != wantShotsAvailable {
		t.Errorf("tirs : %d/%d, attendu %d/%d — le rattachement des tirs a bouge",
			c.Attached, c.Available, wantShotsAttached, wantShotsAvailable)
	}
	if !c.Balanced() {
		t.Errorf("tirs : %d rattaches + %d sansSlot + %d ambigus + %d horsFenetre + %d nonPublies "+
			"!= %d disponibles — un chemin de rejet ne compte plus",
			c.Attached, c.NoSlot, c.Ambiguous, c.OutOfWindow, c.Unpublished, c.Available)
	}
	if doc.Coverage.Verdict["shots"] != VerdictNominal {
		t.Errorf("verdict des tirs : %q, attendu %q", doc.Coverage.Verdict["shots"], VerdictNominal)
	}
	if len(doc.Shots) != wantShotsAttached {
		t.Errorf("%d tirs publies pour %d rattaches : le filtrage par trace publiee a change",
			len(doc.Shots), c.Attached)
	}
}

// TestBridgeNamesNinetyLivesOfHundredFive : LE PONT, ET SON DENOMINATEUR.
//
// 90 vies nommees sur 105. Les 15 restantes ne sont pas un echec du decodage : ce sont des vies
// que le film ne clot par AUCUN evenement (debut de film, survivants de fin de partie). Un
// rapport publie sans son denominateur ne se juge pas.
func TestBridgeNamesNinetyLivesOfHundredFive(t *testing.T) {
	b := buildGolden(t).Coverage.Bridge
	if b.LivesNamed != wantLivesNamed || b.LivesTotal != wantLivesTotal {
		t.Errorf("vies nommees : %d/%d, attendu %d/%d — le fil des morts ou le decoupage des "+
			"vies a bouge", b.LivesNamed, b.LivesTotal, wantLivesNamed, wantLivesTotal)
	}
	// LA REGLE DE PROVENANCE, DANS SA VERSION DU 2026-08-08. Elle exigeait auparavant
	// `FromReading == Slots` — « rien d autre qu une lecture ». Les FERMETURES (closures.go) ne
	// sont pas des lectures mais des deductions par elimination, et elles alimentent le pont.
	// Ce qui est conserve est l ESPRIT de la regle : toute entree doit venir d une source NOMMEE
	// ET COMPTEE. Un ecart signale une troisieme source, non comptee — exactement ce que la
	// version d origine interdisait.
	if got := b.FromReading + b.ClosedByShot + b.ClosedByRespawn; got != b.Slots {
		t.Errorf("%d entrees du pont pour %d justifiees (%d lecture + %d fermeture A + %d "+
			"fermeture B) : une source NON COMPTEE a alimente le pont",
			b.Slots, got, b.FromReading, b.ClosedByShot, b.ClosedByRespawn)
	}
	if b.ClosedByShot != wantClosedByShot || b.ClosedByRespawn != wantClosedByRespawn {
		t.Errorf("fermetures : A=%d B=%d, attendu A=%d B=%d — le raisonnement de fermeture a bouge",
			b.ClosedByShot, b.ClosedByRespawn, wantClosedByShot, wantClosedByRespawn)
	}
	if b.SlotCollisions != 0 {
		t.Errorf("%d collision(s) de slot : un slot change de porteur, la table slot -> joueur "+
			"n est plus licite", b.SlotCollisions)
	}
	if b.IndexDisagreements != 0 {
		t.Errorf("%d identite(s) lue(s) de deux facons : la lecture d index est fausse, et une "+
			"majorite ne l arbitrerait pas — c est un vote qu on a justement retire",
			b.IndexDisagreements)
	}
	if b.IndexReadings != wantIndexReadings {
		t.Errorf("%d chunk(s) de replication concordants, attendu %d", b.IndexReadings, wantIndexReadings)
	}
}

// TestSeventyGrenadeThrowsAreAllPlaced : les 70 lancers, et LA SOURCE de chaque position.
//
// Le lancer porte deja son auteur ; ce qui se mesure ici est OU il est pose. La hierarchie est
// stricte : la naissance du projectile d abord (aucun pont), le biped ensuite. Le compte par
// source est verifie parce qu un basculement silencieux de l une vers l autre changerait la
// signification du calque sans changer son total.
func TestSeventyGrenadeThrowsAreAllPlaced(t *testing.T) {
	doc := buildGolden(t)
	c := doc.Coverage.Grenades
	if c.Available != wantGrenades {
		t.Errorf("%d lancers disponibles, attendu %d", c.Available, wantGrenades)
	}
	if !c.Balanced() {
		t.Errorf("lancers : la somme rattaches+rejets ne fait pas %d", c.Available)
	}
	bySrc := map[string]int{}
	for _, g := range doc.Grenades {
		bySrc[g.Src]++
	}
	if bySrc[GrenadeSrcProjectile]+bySrc[GrenadeSrcBiped] != len(doc.Grenades) {
		t.Errorf("un lancer publie sans source connue : %v", bySrc)
	}
	if bySrc[GrenadeSrcProjectile] == 0 {
		t.Error("aucun lancer situe par la naissance de son projectile : la source qui ne " +
			"depend d AUCUN pont a disparu")
	}
}

// TestProjectilesAndInventoryCounts : les deux calques que rien ne verrouillait.
//
// 439 trajectoires et 184 etats d inventaire. Ce sont les deux sorties dont les decodeurs
// etaient a 0 % de couverture de test avant ce jalon.
func TestProjectilesAndInventoryCounts(t *testing.T) {
	doc := buildGolden(t)
	if len(doc.Projectiles) != wantProjectiles {
		t.Errorf("%d trajectoires de projectile publiees, attendu %d", len(doc.Projectiles), wantProjectiles)
	}
	if len(doc.Inventory) != wantInventory {
		t.Errorf("%d etats d inventaire publies, attendu %d", len(doc.Inventory), wantInventory)
	}
	for _, p := range doc.Projectiles {
		if len(p.P) < 2 {
			t.Fatalf("trajectoire d un seul point de grille publiee : elle ne se dessine pas")
		}
	}
}

// TestGoldenDocumentPublishesItsOwnUncertainty : le document DIT ce qu il perd.
//
// C est l invariant de coverage.go, verifie ici sur des donnees reelles et non fabriquees :
// tout evenement disponible est soit rattache, soit rejete sous une cause NOMMEE.
func TestGoldenDocumentPublishesItsOwnUncertainty(t *testing.T) {
	doc := buildGolden(t)
	if doc.Coverage == nil {
		t.Fatal("le document ne publie pas sa couverture : publier 475 tirs sans dire que 519 " +
			"existaient laisse croire a l exhaustivite")
	}
	for name, v := range doc.Coverage.Verdict {
		if strings.HasPrefix(v, "non publiable") {
			t.Errorf("calque %q : verdict %q sur le film de reference", name, v)
		}
	}
}

// ---------------------------------------------------------------------------
// Rendu du golden
// ---------------------------------------------------------------------------

// renderAssembly rend le document sous une forme figeable : DETERMINISTE (aucune iteration de
// map non triee) et QUALIFIEE (chaque chiffre porte la phrase qui dit de quoi il est le compte).
func renderAssembly(doc ReplayDocument) string {
	var b strings.Builder
	p := func(f string, a ...any) { fmt.Fprintf(&b, f+"\n", a...) }

	p("# REJEU 2D — ASSEMBLAGE REJOUE SUR ENTREES FIGEES — film %s", doc.MatchID)
	p("#")
	p("# AUCUN OCTET DE FILM n est lu pour produire cette sortie : les entrees viennent de")
	p("# testdata/inputs_%s.bin.gz, decodees une fois et figees. Ce fichier verrouille", doc.MatchID)
	p("# l ASSEMBLAGE ; le DECODAGE est verrouille a part par la mini-bobine.")
	p("")

	p("## AXE DE TEMPS")
	p("%d frames · intervalle %d ms · duree %d ms", doc.FrameCount, doc.FrameIntervalMS, doc.DurationMS)
	p("schema %d · titre %s", doc.SchemaVersion, doc.TitleSlug)
	if doc.OriginMs != nil {
		p("origine de la frame 0 sur l horloge du fil : %d ms", *doc.OriginMs)
	} else {
		p("origine de la frame 0 : NON ETABLIE (le client retombe sur l appariement)")
	}
	renderT0Film(p, doc)
	p("")

	renderTracks(p, doc)
	renderLayerCoverage(p, "TIRS — rattaches sur DISPONIBLES, et la ventilation de tout ce qui est ecarte",
		doc.Coverage.Shots, doc.Coverage.Verdict["shots"])
	p("%d tir(s) publie(s) · %d avec un cap de visee lisible · %d avec une arme nommee",
		len(doc.Shots), countShotsWithAim(doc), countShotsWithWeapon(doc))
	p("")
	renderGrenades(p, doc)
	renderProjectiles(p, doc)
	renderInventory(p, doc)
	renderGrenadeReads(p, doc)
	renderAbilities(p, doc)
	renderEquipment(p, doc)
	renderTranslocations(p, doc)
	renderGrapple(p, doc)
	renderPlacements(p, doc)
	renderGroundWeapons(p, doc)
	renderLoadouts(p, doc)
	renderBridge(p, doc)
	renderLabels(p, doc)
	renderBounds(p, doc)
	return b.String()
}

// renderT0Film fige LE COUP D ENVOI et son verdict (cf. t0_film.go). Le refus est rendu AVEC
// ses deux chiffres : une ligne « non date » sans la rafale ni la marge ne dirait pas si le
// film est muet ou si un seul joueur a bouge.
func renderT0Film(p func(string, ...any), doc ReplayDocument) {
	c := doc.Coverage.T0Film
	if c == nil {
		p("coup d envoi : NON MESURE (sans origine, l horloge du resultat serait inconnue)")
		return
	}
	if doc.T0FilmMs == nil {
		p("coup d envoi : REFUSE (%s) · %d/%d piste(s) en mouvement · rafale %d · marge %d ms",
			c.Reason, c.Moving, c.Tracks, c.Burst, c.MarginMs)
		return
	}
	p("coup d envoi date par le film : %d ms · %d/%d piste(s) en mouvement · rafale %d · "+
		"marge depuis la frame 0 : %d ms",
		*doc.T0FilmMs, c.Moving, c.Tracks, c.Burst, c.MarginMs)
}

func renderTracks(p func(string, ...any), doc ReplayDocument) {
	named, points := 0, 0
	for _, tr := range doc.Tracks {
		points += len(tr.Points)
		if tr.XUID != "" {
			named++
		}
	}
	p("## TRACES PUBLIEES — une trace est UNE VIE, pas un joueur (le slot migre a chaque reapparition)")
	p("%d trace(s) · %d point(s) de grille", len(doc.Tracks), points)
	p("%d trace(s) NOMMEE(S) par le pont (fil des morts, puis fermetures) · %d anonyme(s) — une vie que le film ne clot",
		named, len(doc.Tracks)-named)
	p("par aucun evenement reste sans identite, et c est une LIMITE, pas une erreur")
	p("%d point(s) portent un cap de visee · %d une elevation de visee · %d un bouclier · %d une fraction de vie",
		countPoints(doc, func(pt Point) bool { return pt.H != 0 }),
		// ELEVATION : `p` non nul. Le compte est un PLANCHER et le golden le dit —
		// une visee exactement a plat s omet (contrat de Point.P), elle n est donc pas
		// comptee ici alors qu elle a bien ete lue.
		countPoints(doc, func(pt Point) bool { return pt.P != 0 }),
		countPoints(doc, func(pt Point) bool { return pt.Sh != nil }),
		countPoints(doc, func(pt Point) bool { return pt.Hp != nil }))
	p("")
	p("### ROSTER — le xuid IDENTIFIE, l index ORDONNE ; ils ne sont pas interchangeables")
	p("%d joueur(s)", len(doc.Roster))
	for _, r := range doc.Roster {
		p("  %s  idx=%d  %q", r.XUID, r.FilmIndex, r.Name)
	}
	p("")
}

func renderLayerCoverage(p func(string, ...any), titre string, c LayerCoverage, verdict string) {
	p("## %s", titre)
	p("%d rattache(s) / %d disponible(s) = %s", c.Attached, c.Available, pct(c.Attached, c.Available))
	p("rejets : slot introuvable=%d · slot ambigu=%d · hors fenetre=%d · sans trace publiee=%d",
		c.NoSlot, c.Ambiguous, c.OutOfWindow, c.Unpublished)
	p("somme de controle : rattaches + rejets = disponibles -> %d = %d (equilibre : %v)",
		c.Attached+c.NoSlot+c.Ambiguous+c.OutOfWindow+c.Unpublished, c.Available, c.Balanced())
	p("verdict : %s", verdict)
}

// grenadeRankName rend le nom EN du rang d'un lancer, ou son numéro si le document ne
// porte pas la table. Le golden compte PAR TYPE : sans nom il compterait des entiers, et
// une inversion de rangs passerait inaperçue.
func grenadeRankName(doc ReplayDocument, rank int) string {
	if rank >= 0 && rank < len(doc.GrenadeLabels) {
		return doc.GrenadeLabels[rank].En
	}
	return "rang " + strconv.Itoa(rank)
}

func renderGrenades(p func(string, ...any), doc ReplayDocument) {
	renderLayerCoverage(p, "GRENADES — le lancer porte son auteur ; ce qui se mesure est OU il est pose",
		doc.Coverage.Grenades, doc.Coverage.Verdict["grenades"])
	bySrc := map[string]int{}
	byKind := map[string]int{}
	for _, g := range doc.Grenades {
		bySrc[g.Src]++
		byKind[grenadeRankName(doc, g.Rank)]++
	}
	p("par source : projectile=%d (aucun pont) · biped=%d (le pont a servi)",
		bySrc[GrenadeSrcProjectile], bySrc[GrenadeSrcBiped])
	p("par type : %s", renderCounts(byKind))
	p("")
}

func renderProjectiles(p func(string, ...any), doc ReplayDocument) {
	pts, rest := 0, 0
	for _, pr := range doc.Projectiles {
		pts += len(pr.P)
		if pr.Rest {
			rest++
		}
	}
	p("## PROJECTILES — le dernier point est la DERNIERE POSITION REPLIQUEE, jamais un impact")
	p("%d trajectoire(s) · %d point(s) de grille", len(doc.Projectiles), pts)
	p("%d se terminent sur `projectile-at-rest-state` — le SEUL champ qui certifie une fin de vol",
		rest)
	p("")
}

func renderInventory(p func(string, ...any), doc ReplayDocument) {
	var gren, ammo, multi, mort, inexplique int
	for _, inv := range doc.Inventory {
		if len(inv.G) > 0 {
			gren++
		}
		if len(inv.Am) > 0 {
			ammo++
		}
		if inv.Cand > 1 {
			multi++
		}
		switch inv.Empty {
		case InventoryEmptyDead:
			mort++
		case InventoryEmptyUnknown:
			inexplique++
		}
	}
	p("## INVENTAIRE — lu aux images-cles, jamais interpole entre deux")
	p("%d etat(s) publie(s) · %d avec grenades lues · %d avec munitions lues",
		len(doc.Inventory), gren, ammo)
	p("%d lecture(s) de munitions a PLUSIEURS candidats : la plus longue a ete retenue et le",
		multi)
	p("nombre de candidats est publie, pour que le departage reste visible")
	// LES LECTURES VIDES SONT PUBLIEES ET MARQUEES (schema 19) : une lecture vide n'est pas une
	// absence de lecture, et sans marqueur elle EFFACE la fiche cote client.
	p("%d lecture(s) VIDE(S) : %d corroboree(s) par le fil des morts (`dead`), %d inexpliquee(s)",
		mort+inexplique, mort, inexplique)
	// LES DENOMINATEURS DU CALQUE, figes comme ceux des poses et des socles : sans eux, « 184
	// etats publies » ne dit pas si le decodeur en a lu 184 ou 400. La somme des trois derniers
	// vaut le premier — invariant verrouille a part (TestInventoryCoverageBalances).
	if c := doc.Coverage.Inventory; c == nil {
		p("aucune couverture d inventaire (rien n a ete fourni a lire)")
	} else {
		p("couverture : %d lecture(s) decodee(s) -> %d ecartee(s) avant l origine du rejeu · "+
			"%d ecartee(s) faute de piste publiee -> %d publiee(s)",
			c.Decoded, c.DroppedBeforeOrigin, c.Unpublished, c.Published)
	}
	p("rangs de grenade : %s", renderBilingualList(doc.GrenadeLabels))
	p("")
}

// renderGrenadeReads publie l AXE DES GRENADES PORTEES et sa ventilation par canal.
//
// CE QUE CE BLOC PROTEGE : le second canal. Sans lui, le golden figerait un document ou les
// lectures delta pourraient disparaitre sans qu une seule ligne bouge — un fixture qui ne rend
// pas ce qu il verrouille ne verrouille rien. La ventilation par SOURCE est le point : c est
// elle qui dirait qu un canal s est tu.
//
// L ECART D AGE EST PUBLIE parce qu il est la raison d etre du lot : entre deux images-cles la
// fiche affiche la derniere lecture connue, et le canal delta la rajeunit. Sur le corpus de
// 70 films l age median passe de 10,00 s a 8,09 s.
func renderGrenadeReads(p func(string, ...any), doc ReplayDocument) {
	bySrc := map[string]int{}
	withSel := 0
	for _, g := range doc.GrenadeReads {
		bySrc[g.Src]++
		if g.Gs != nil {
			withSel++
		}
	}
	p("## GRENADES PORTEES — un axe, DEUX canaux, chaque lecture disant d ou elle vient")
	p("%d lecture(s) · %d par image-cle (~20 s) · %d par delta (transmis AU CHANGEMENT)",
		len(doc.GrenadeReads), bySrc[GrenadeSrcKeyframe], bySrc[GrenadeSrcDelta])
	p("%d lecture(s) portent le rang SELECTIONNE — une selection ne se devine pas", withSel)
	if c := doc.Coverage.GrenadeReads; c != nil {
		p("couverture : %d image-cle · %d delta · %d ecartee(s) sans piste · canal munitions refuse : %v",
			c.FromKeyframe, c.FromDelta, c.Unpublished, c.AmmoRefused)
	} else {
		p("couverture : ABSENTE — aucun canal n a rien rendu sur ce film")
	}
	p("")
}

// renderAbilities publie le calque des CAPACITES PORTEES — le rang de palette, et par quel
// canal chaque lecture est arrivee. Les deux canaux ne voient pas la meme chose : `kf` ne peut
// rendre que 16..23, `i48` rend toute la palette. Separer leurs comptes est ce qui rend la
// couverture jugeable.
func renderAbilities(p func(string, ...any), doc ReplayDocument) {
	bySrc := map[string]int{}
	byRank := map[int]int{}
	for _, a := range doc.Abilities {
		bySrc[a.Src]++
		byRank[a.R]++
	}
	ranks := make([]int, 0, len(byRank))
	for r := range byRank {
		ranks = append(ranks, r)
	}
	sort.Ints(ranks)
	parts := make([]string, 0, len(ranks))
	named := 0
	for _, r := range ranks {
		parts = append(parts, fmt.Sprintf("%d:%d", r, byRank[r]))
		if _, ok := doc.AbilityLabels[strconv.Itoa(r)]; ok {
			named += byRank[r]
		}
	}
	p("## CAPACITE PORTEE — le RANG de palette, deux canaux pour une seule grandeur")
	p("%d lecture(s) · %d par i48 (rang complet, paquets delta) · %d par image-cle (fenetre 16..23)",
		len(doc.Abilities), bySrc[AbilitySrcI48], bySrc[AbilitySrcKeyframe])
	p("rangs observes : %s", strings.Join(parts, " "))
	p("lectures NOMMEES %d/%d — un rang hors table garde son numero, et la table est propre a"+
		" la palette du film", named, len(doc.Abilities))
	p("capacites nommees : %s", renderBilingualMap(doc.AbilityLabels))
	p("")
}

// renderEquipment publie le calque des EPISODES D ETAT ACTIF (camo, surbouclier) et sa
// couverture. Le film de reference est un FIESTA : aucun porteur d equipement rang 8/9
// (palette famille B, rangs 19-22) — et i28 queue[1] est l etat d invisibilite de l
// UNITE, quelle qu en soit la source. Controle du 2026-08-16 sur ce film : 698 lectures
// de queue[1], STRICTEMENT binaires (0:617 · 4095:81), transitions reparties sur des
// vies de rangs 19-22 et sans identite — c est le DASH du mode Fiesta qui allume le
// canal (enseignement utilisateur du 2026-08-16), PAS un power-up ramasse : ce mode ne
// pose aucun equipement au sol, et la distribution des durees des episodes camo
// (camo_duration_distribution_test.go) le confirme — aucun episode a activation unique
// ne s etale sur la duree d un ramassage. L exclusivite rang 8 de la phase A etait la
// VALIDATION du canal sur des films ou l equipement equipe etait la seule source
// observee. Le surbouclier, lui, reste a ZERO ici (temoin de forme du 2026-08-05 :
// 27 404/27 404 quanta dans [0, 64]) — un zero fige avec son denominateur.
func renderEquipment(p func(string, ...any), doc ReplayDocument) {
	p("## EQUIPEMENT ACTIF — episodes dates par vie, DEUX familles mesurees et rien d autre")
	byFam := map[string]int{}
	endRead := 0
	for _, e := range doc.EquipmentEpisodes {
		byFam[e.Fam]++
		if e.EndRead {
			endRead++
		}
	}
	p("%d episode(s) publie(s) · par famille : %s · %d fin(s) MESUREE(s) (le reste ferme a la mort)",
		len(doc.EquipmentEpisodes), renderCounts(byFam), endRead)
	if c := doc.Coverage.Equipment; c != nil {
		p("couverture : %d vie(s) publiee(s) · camo %d vie(s) / %d episode(s) · surbouclier %d vie(s) / %d episode(s)",
			c.TracksTotal, c.CamoLives, c.CamoEpisodes, c.OvershieldLives, c.OvershieldEpisodes)
	}
	p("")
}

// renderTranslocations fige le calque des TELEPORTATIONS du translocateur (schema 38) :
// datees ET SITUEES par l evenement 117 du film — jamais devinees d un seuil spatial, jamais
// datees du `spent`, jamais derivees d une discontinuite de piste. Le va-et-vient se fige
// AVEC son compteur (`positioned`) : un saut publie sans lui est une charge non lue, pas un
// saut vers l origine du monde. Le film de reference (Fiesta Cliffhanger) peut n en porter
// aucune : le zero se fige AVEC ses compteurs, sinon il se confondrait avec une lecture qui
// aurait echoue.
func renderTranslocations(p func(string, ...any), doc ReplayDocument) {
	p("## TRANSLOCATEUR — teleportations datees par l EVENEMENT du film, jamais devinees d un seuil")
	if c := doc.Coverage.Translocations; c != nil {
		p("%d evenement(s) -> %d publie(s) (%d avec va-et-vient) · %d avant l origine · %d sans piste publiee",
			c.Events, c.Published, c.Positioned, c.BeforeOrigin, c.Unpublished)
	} else {
		p("aucune couverture (rien n a ete fourni a lire)")
	}
	for _, tr := range doc.Translocations {
		if tr.FX == nil {
			p("  slot=%d t=%d (sans va-et-vient)", tr.Slot, tr.T)
			continue
		}
		p("  slot=%d t=%d (%.2f,%.2f,%.2f) -> (%.2f,%.2f,%.2f)",
			tr.Slot, tr.T, *tr.FX, *tr.FY, *tr.FZ, *tr.TX, *tr.TY, *tr.TZ)
	}
	p("")
}

// renderGrapple publie le calque des TRACTIONS DE GRAPPIN (schema 8) et sa couverture :
// la fenetre est MESUREE (du tir a l arrivee sur la trajectoire), l ancre est un point
// monde fixe prouve au gate 0 du plan grappin. Les ratés et corps non decodables sont
// figes AVEC les tractions — un compte sans ses rejets se lirait comme une exhaustivite.
func renderGrapple(p func(string, ...any), doc ReplayDocument) {
	p("## GRAPPIN — tractions datees par vie, l ancre en coordonnees monde (fenetre MESUREE)")
	if c := doc.Coverage.Grapple; c != nil {
		p("%d traction(s) sur %d vie(s) · lectures : %d tir(s) + %d accroche(s) · %d rate(s) · "+
			"%d corps non decodable(s)",
			c.Pulls, c.PullLives, c.LightReads, c.HeavyReads, c.UnpairedFires, c.BrokenBodies)
	}
	for _, l := range doc.GrappleLines {
		p("  slot=%d t=[%d, %d] ancre=(%.2f, %.2f, %.2f)", l.Slot, l.T0, l.T1, l.AX, l.AY, l.AZ)
	}
	p("")
}

// renderPlacements publie le calque des POSES D EQUIPEMENT (schema 9) et sa couverture.
//
// CE QUE CETTE SECTION FIGE, ET POURQUOI CHAQUE CHIFFRE Y EST. Le DECOUPAGE (« 9/5 ») est
// mesure DANS le film et conditionne tout le reste : s il bouge, ce sont des poses entierement
// differentes qui sortent. Les ANCRES contre les CONFIRMEES disent la selectivite reelle de
// l en-tete de creation (elle est faible, et c est l oracle de position qui fait le tri).
// NOMMEES contre AUTRES dit ce que le manifeste couvre — un `other` est un resultat, pas un
// echec. AVEC POSEUR et AVEC CAP disent ce que la mesure de proximite a rendu.
//
// LE DETAIL EST BORNE aux poses NOMMEES : elles seules seront dessinees, et lister trois cents
// objets du monde noierait le diff que ce golden existe pour rendre lisible. Leur compte, lui,
// est fige par la ligne de couverture.
func renderPlacements(p func(string, ...any), doc ReplayDocument) {
	// LE TITRE DE CETTE SECTION A ETE FAUX DEUX FOIS, ET LA SECONDE FOIS EST INSTRUCTIVE.
	//
	// Il a dit « mur et capteur sur la carte, le reste publie SANS NOM » jusqu au 2026-08-18 :
	// juste tant que le nommage venait d une diagonale (2 familles sur 21 identifiants), perime
	// des que la chaine `sofd -> sofa -> eqip` du jeu en a nomme 20 sur 21.
	//
	// Il a ensuite dit « dotation au spawn ET objets deployes », parce que le golden re-genere
	// montrait des GROUPES — deux grenades identiques a 2 cm et une capacite, meme instant, meme
	// poseur — et qu une dotation recue au spawn expliquait le groupe. **C ETAIT LE MAUVAIS BOUT
	// DE LA VIE.** La mesure du 2026-08-18 (PLAN_ORIGINE_POSES_ET_FAMILLES G.1) : ces groupes
	// naissent a 20-40 ms et 0,63 m du DERNIER POINT de leur poseur, pas de son premier. Ce n est
	// pas une dotation recue, c est un LACHER : le joueur meurt, et ce qu il portait tombe. Au
	// spawn, la mesure compte 4 poses sur 3 661 — 0,1 %, et les 4 sont des vies si courtes que
	// debut et fin se confondent.
	//
	// LA LECON QUE CE COMMENTAIRE GARDE : un groupe de creations simultanees au meme endroit ne
	// dit pas QUAND dans la vie du poseur il arrive. Il fallait mesurer les deux bouts.
	p("## OBJETS D EQUIPEMENT — objets DEPLOYES et objets LACHES a la mort, par famille et origine")
	c := doc.Coverage.Placements
	if c == nil {
		p("aucune couverture de poses (calque absent)")
		p("")
		return
	}
	p("calibre=%t decoupage=%q · %d vie(s) d objet · %d ancre(s) -> %d confirmee(s) par l oracle "+
		"de position -> %d pose(s)", c.Calibrated, c.Widths, c.Lives, c.Anchors, c.Confirmed,
		c.Placements)
	p("%d nommee(s) · %d `other` (nature non etablie : le manifeste ne les revendique pas) · "+
		"%d avec poseur mesure · %d avec cap de visee", c.Named, c.Other, c.WithOwner,
		c.WithHeading)
	// L ORIGINE EST LA LIGNE QUI DECIDE DU RENDU : seuls les DEPLOYES decrivent un geste. Les
	// trois comptes sont figes ensemble, et leur somme doit valoir le total (invariant teste).
	p("origine mesuree : %d deployee(s) · %d lachee(s) a la fin de la vie du poseur · "+
		"%d sans poseur", c.Deployed, c.Dropped, c.Unknown)
	for _, f := range clesTrieesDeCarte(c.ByFamily) {
		p("  famille %-8s %d", f, c.ByFamily[f])
	}
	for _, k := range clesTrieesDeCarte(c.ByFamilyOrigin) {
		p("  famille x origine %-28s %d", k, c.ByFamilyOrigin[k])
	}
	for _, pl := range doc.EquipmentPlacements {
		if pl.Family == equipmentFamilyOther {
			continue
		}
		cap := "aucun"
		if pl.H != nil {
			cap = fmt.Sprintf("%.1f deg", *pl.H)
		}
		p("  %s %s %s t=[%d, %d] pos=(%.2f, %.2f, %.2f) poseur=%d cap=%s",
			pl.Family, pl.Origin, pl.ID, pl.T0, pl.T1, pl.X, pl.Y, pl.Z, pl.Owner, cap)
	}
	p("")
}

// renderGroundWeapons fige les SOCLES D ARME (schema 11) et, surtout, les DENOMINATEURS qui
// disent pourquoi il y en a autant — ou aucun.
//
// LE FILM DE REFERENCE EN A ZERO, ET C EST UN RESULTAT MESURE, PAS UN TROU. `000d5950` est un
// Super Fiesta sur variante FORGE : aucun rack de carte, et 82,3 % de ses apparitions d armes
// sont des LACHERS a une mort (la part la plus elevee des huit films de la mesure, contre 62,3 %
// et 64,9 % sur les arenes classiques). Un golden qui n afficherait que « 0 socle » laisserait
// croire a une panne de decodage ; les compteurs ci-dessous distinguent les trois causes.
func renderGroundWeapons(p func(string, ...any), doc ReplayDocument) {
	p("## SOCLES D ARME AU SOL — la RECURRENCE fait le socle, et le film dit quand il se vide")
	c := doc.Coverage.GroundWeapons
	if c == nil {
		p("aucune couverture de socles (calque absent)")
		p("")
		return
	}
	p("balaye=%t · bande %d slot(s) · %d ancre(s) -> %d acceptee(s) -> %d retenue(s) par "+
		"l IDENTITE (%d ecartee(s))", c.Scanned, c.Slots, c.Anchors, c.Accepted, c.Kept,
		c.Rejected)
	// LES TROIS CLASSES DECIDENT DE TOUT : seules les apparitions NON lachees et qui n ont
	// jamais bouge (`au repos`) peuvent faire un socle.
	p("%d lachee(s) a une mort · %d apparue(s) sans mort a proximite, dont %d AU REPOS "+
		"(jamais bougee : le seul jeu qui fait des socles)", c.Dropped, c.Spawned, c.AtRest)
	p("%d grappe(s) -> %d socle(s) publie(s) · %d occupation(s) : %d datee(s) · %d sans passage "+
		"· %d jamais videe(s)", c.Clusters, c.Pads, c.Occupancies, c.Dated, c.Unknown, c.Never)
	p("%d cycle(s) ETABLI(s) — un cycle instable publie `null`, jamais un chiffre", c.Cycles)
	// LA VOIE ti=37 A SA PROPRE LIGNE : ses denominateurs sont ceux d un AUTRE balayage, et le
	// film de reference (Super Fiesta sur variante Forge) en retient ZERO sur 401 creations
	// acceptees — un mode qui ne pose aucun power-up de carte. Sans ce rapport, ce zero-la se
	// confondrait avec une lecture qui aurait echoue en bloc.
	p("POWER-UPS (voie ti=37) — balaye=%t · %d creation(s) acceptee(s) -> %d retenue(s) par "+
		"l IDENTITE `powerup_*` -> %d socle(s) publie(s)",
		c.PowerupScanned, c.PowerupAccepted, c.PowerupKept, c.PowerupPads)
	for i, pad := range doc.WeaponPads {
		cycle := "non etabli"
		if pad.Cycle != nil {
			// LES DEUX MOITIES DU DENOMINATEUR SONT FIGEES : les ecarts MESURES et ceux que le
			// socle offrait sans qu on puisse les prendre (disparition precedente non datee).
			cycle = fmt.Sprintf("%.1f s (p10 %.1f · p90 %.1f · %d ecarts mesures, %d manques)",
				pad.Cycle.MedianS, pad.Cycle.P10S, pad.Cycle.P90S, pad.Cycle.Gaps,
				pad.Cycle.Missing)
		}
		p("  socle %d %s pos=(%.2f, %.2f, %.2f) apparitions=%v cycle=%s",
			i, pad.Weapon, pad.X, pad.Y, pad.Z, pad.Spawns, cycle)
		for _, pr := range pad.Presence {
			p("    presence t0=%d prouvee jusqu a %d, absente a %d", pr.T0, pr.TLow, pr.THigh)
		}
	}
	// LE RAMASSEUR EST PUBLIE DEPUIS LE SCHEMA 29 (2026-08-31), et le golden le CHIFFRE.
	// Jusque-la `xuid` valait `null` partout, faute d oracle (88,1 % par slot de vie, 79,7 %
	// par joueur, contre >= 90 % exige) ; l evenement natif `biped_pickup` PORTE son ramasseur
	// au lieu de le deduire. Un compte a zero ne veut donc plus dire « impossible » mais
	// « aucune occupation de ce film n est couverte par le canal natif ».
	nommes, datees := 0, 0
	for _, k := range doc.PadPickups {
		if k.XUID != nil {
			nommes++
		}
		if k.T != nil {
			datees++
		}
	}
	p("%d occupation(s) achevee(s) publiee(s) · %d datee(s) a l instant exact par l evenement "+
		"natif · %d avec un ramasseur nomme", len(doc.PadPickups), datees, nommes)
	p("")
}

// clesTrieesDeCarte rend les cles d une carte de comptes, triees — un golden dont l ordre
// depend de l iteration d une map n est pas un golden.
func clesTrieesDeCarte(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func renderLoadouts(p func(string, ...any), doc ReplayDocument) {
	fams := map[string]bool{}
	for _, l := range doc.Loadouts {
		for _, w := range l.W {
			fams[w] = true
		}
	}
	p("## ARMES PORTEES — l inventaire, pas la main : le loadout ne dit pas QUELLE arme est degainee")
	p("%d loadout(s) publie(s) · %d famille(s) distincte(s)", len(doc.Loadouts), len(fams))
	p("")
}

func renderBridge(p func(string, ...any), doc ReplayDocument) {
	b := doc.Coverage.Bridge
	p("## PONT SLOT -> JOUEUR — deux sources nommees : la LECTURE, puis la FERMETURE")
	p("%d entree(s) · %d issue(s) de la lecture (un ecart NON EXPLIQUE par les fermetures "+
		"signalerait une TROISIEME SOURCE)", b.Slots, b.FromReading)
	p("fermetures : %d par le corps disponible · %d par la reapparition — une fermeture n attribue "+
		"QUE lorsqu un seul candidat reste possible, jamais par vote",
		b.ClosedByShot, b.ClosedByRespawn)
	p("refus des fermetures : %d contestee(s) (l unicite manque : deux corps, deux joueurs, deux "+
		"corps pour un meme joueur, ou deux corps pour une meme mort) · %d rejetee(s) (le corps "+
		"ne PROLONGE pas le tireur, recouvrement, ou identite hors table d index) — un controle "+
		"qui ne rejette rien ne prouve rien",
		b.ClosedContested, b.ClosedRefused)
	p("%d vie(s) nommee(s) / %d — un rapport publie sans son denominateur ne se juge pas",
		b.LivesNamed, b.LivesTotal)
	p("%d chunk(s) de replication concordants · %d desaccord(s) d identite · %d collision(s) de slot",
		b.IndexReadings, b.IndexDisagreements, b.SlotCollisions)
	p("verdict : %s", doc.Coverage.Verdict["bridge"])
	p("")
}

func renderLabels(p func(string, ...any), doc ReplayDocument) {
	p("## LIBELLES D ARMES — le tag brut reste A COTE du libelle, jamais a sa place")
	p("%d identifiant(s) nomme(s) ; un identifiant absent garde son hexadecimal a l ecran et",
		len(doc.WeaponLabels))
	p("n emprunte pas le nom d une arme voisine")
	for _, k := range sortedKeys(doc.WeaponLabels) {
		l := doc.WeaponLabels[k]
		p("  %s  en=%q fr=%q fx=%q", k, l.En, l.Fr, l.Fx)
	}
	p("")
}

// renderBilingualList / renderBilingualMap — les libelles sont BILINGUES depuis le lot
// 3.2 : le golden fige les DEUX langues, sinon une regression FR passerait sous un
// golden vert.
func renderBilingualList(in []Label) string {
	parts := make([]string, 0, len(in))
	for i, l := range in {
		parts = append(parts, fmt.Sprintf("%d:en=%q/fr=%q", i, l.En, l.Fr))
	}
	return strings.Join(parts, ", ")
}

func renderBilingualMap(in map[string]Label) string {
	keys := sortedKeys(in)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=en:%q/fr:%q", k, in[k].En, in[k].Fr))
	}
	return strings.Join(parts, ", ")
}

func renderBounds(p func(string, ...any), doc ReplayDocument) {
	p("## BORNES — l etendue des points publies, dans le repere monde partage")
	p("x [%.2f, %.2f] · y [%.2f, %.2f] · z [%.2f, %.2f]",
		doc.Bounds.MinX, doc.Bounds.MaxX, doc.Bounds.MinY, doc.Bounds.MaxY,
		doc.Bounds.MinZ, doc.Bounds.MaxZ)
}

func countPoints(doc ReplayDocument, keep func(Point) bool) int {
	n := 0
	for _, tr := range doc.Tracks {
		for _, pt := range tr.Points {
			if keep(pt) {
				n++
			}
		}
	}
	return n
}

func countShotsWithAim(doc ReplayDocument) int {
	n := 0
	for _, s := range doc.Shots {
		if s.H != 0 {
			n++
		}
	}
	return n
}

func countShotsWithWeapon(doc ReplayDocument) int {
	n := 0
	for _, s := range doc.Shots {
		if s.Weapon != "" {
			n++
		}
	}
	return n
}

func pct(a, b int) string {
	if b == 0 {
		return "aucune donnee"
	}
	return fmt.Sprintf("%.1f %%", 100*float64(a)/float64(b))
}

func renderCounts(m map[string]int) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", k, m[k]))
	}
	if len(parts) == 0 {
		return "aucun"
	}
	return strings.Join(parts, " · ")
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
