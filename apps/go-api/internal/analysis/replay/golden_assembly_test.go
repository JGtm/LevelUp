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
	renderAbilities(p, doc)
	renderLoadouts(p, doc)
	renderBridge(p, doc)
	renderLabels(p, doc)
	renderBounds(p, doc)
	return b.String()
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
	p("%d point(s) portent un cap de visee · %d un bouclier · %d une fraction de vie",
		countPoints(doc, func(pt Point) bool { return pt.H != 0 }),
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
	var gren, ammo, multi int
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
	}
	p("## INVENTAIRE — lu aux images-cles, jamais interpole entre deux")
	p("%d etat(s) publie(s) · %d avec grenades lues · %d avec munitions lues",
		len(doc.Inventory), gren, ammo)
	p("%d lecture(s) de munitions a PLUSIEURS candidats : la plus longue a ete retenue et le",
		multi)
	p("nombre de candidats est publie, pour que le departage reste visible")
	p("rangs de grenade : %s", renderBilingualList(doc.GrenadeLabels))
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
