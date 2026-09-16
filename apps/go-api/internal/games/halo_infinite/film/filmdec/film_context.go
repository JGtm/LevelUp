package filmdec

// film_context.go — CE QUE TOUS LES BALAYAGES D'UN MEME FILM SE PARTAGENT.
//
// # CE QUE CE FICHIER FERME
//
// Lot 2 de PLAN_CUISSON_PERF (2026-09-03). Le lot 1 avait supprime les ~36 relectures du film :
// les balayages recoivent un `*filmsource.Film` decompresse UNE fois. Restait le second etage du
// meme defaut — chaque balayage RECALCULAIT, sur ce film deja charge, les trois memes
// derivations :
//
//	la BANDE DE SLOTS BIPEDE   `bipedSlotBand` — une marche de l'image-cle de tete de CHAQUE
//	                            chunk. Huit balayages la relevaient (positions, ramassages
//	                            natifs, et les six canaux delta).
//	le DECOUPAGE D'i0          `DetectI0LayoutOf` — six chunks marches bit a bit, plus SA
//	                            PROPRE bande. Six balayages delta le redetectaient.
//	le REGISTRE chunk_00       `ParseRegistryChunk` — une douzaine de re-analyses par cuisson,
//	                            une par accesseur d'archetype.
//
// Ces trois valeurs ne dependent QUE du film. Les recalculer par balayage etait du temps pur.
// [FilmContext] les porte, construit UNE fois par `replay.BuildFromFilm`, et les balayages le
// recoivent a la place du film.
//
// # LA REGLE DU CATALOGUE (lot 3, 2026-09-03)
//
// Le decoupage d'i0 n'est plus l'auto-detecte : c'est celui du CATALOGUE quand l'entree de carte
// est valide, l'auto-detection en repli — la regle EXACTE que les positions appliquaient deja
// seules (`replay/build_from_film.go`, doctrine heritee de `build.go:249-259`). Elle est ecrite
// UNE fois, ici, et les positions la lisent au meme endroit que les six canaux delta
// ([FilmContext.ImposedLayout]).
//
// CE QUE CELA CORRIGE, ET OU. Sur une carte a plus de deux regions de compression, l'index de
// region occupe PLUS d'un bit dans i0, et l'auto-detection ne sait pas le voir : elle rend
// toujours `GateBits = DefaultI0GateBits` (5) et `Region = 0` (cf. i0_layout.go). Sur Live Fire
// — 4 regions declarees, arene en region 1, catalogue `gate=6 region=1 12/12/11` — elle rend
// `gate=5 region=0 13/12/11` : MEME longueur totale d'i0 (41 bits), donc les balayages delta
// marchaient bel et bien, mais la porte de region ne testait qu'UN bit contre zero. Elle
// acceptait donc AUSSI les enregistrements de la region 00, dont les quanta sont exprimes dans
// une AUTRE AABB : 27 enregistrements sur 267 400 (mesure du 2026-09-03 sur `60ae07c4`), qui
// n'appartiennent pas a l'arene jouee. Le catalogue teste les DEUX bits contre 1 et les ecarte.
// Sur les 12 autres films du corpus d'equivalence, catalogue et auto-detection donnent le MEME
// decoupage, au bit pres — c'est pourquoi la correction ne mord que sur Live Fire.
//
// # LE PROFIL, LUI, EST RESOLU AU CONSTRUCTEUR (lot 2.1, D1 du PLAN_DECODEUR_FILM)
//
// [FilmContext.Profile] fait EXCEPTION a la paresse decrite ci-dessous, et l exception est
// bornee a `NewFilmContextForMap` — le constructeur de la CUISSON. Trois raisons, mesurees :
//
//	D1 L EXIGE. « Un seul objet par film, le profil resolu a la construction, les memos
//	   (registre, bande de slots) restent paresseux. » Un profil resolu au premier accesseur
//	   pourrait etre resolu DEUX fois differemment si une globale bougeait entre-temps ; c est
//	   precisement ce que « immuable » doit exclure.
//	L ORDRE DES ETAPES NE BOUGE PAS. `replay.scanFilmInputs` construit le contexte AVANT de
//	   demarrer l horloge des etapes (`s.opt.clock`) : la resolution tombe donc dans le meme
//	   intervalle que l attente du verrou et la lecture du catalogue, qui ne sont le temps
//	   d aucun balayage. Aucune etape observee ne change de date.
//	AUCUNE ERREUR NE REMONTE PAR LE CONSTRUCTEUR. Une cle absente est portee par
//	   [FilmContext.ProfileErr], typee, et JOURNALISEE ici ; le film n est pas mis de cote
//	   (D-4 d ADR 0034) et les replis existants tournent comme avant.
//
// `NewFilmContext` (sans carte : instruments, enveloppes D2) le resout au PREMIER ACCES. Ce
// constructeur-la ouvre des dizaines de contextes par test, et la resolution lit `chunk_00` —
// la payer d avance pour des contextes qui ne demandent jamais leur profil coute du temps de
// test sans rien prouver. La valeur rendue est la MEME : [ResolveProfile] est une fonction pure.
//
// # POURQUOI LA MEMOISATION EST PARESSEUSE, ET NON CALCULEE AU CONSTRUCTEUR
//
// Le lot 2 est un REFACTO PUR : les sorties doivent etre identiques a l'octet, et l'ORDRE des
// etapes observees ne doit pas bouger (garde `replay/observe_test.go`, harnais `cmd/replay-equiv`).
// Un constructeur qui calculerait tout d'avance deplacerait le premier calcul AVANT le premier
// balayage — donc avant l'installation des largeurs d'axe et avant le demarrage de l'horloge des
// etapes — et ferait travailler un film qui echoue des les positions. Paresseux, le premier
// calcul a lieu EXACTEMENT la ou il avait lieu avant (le premier balayage qui en a besoin) ;
// les suivants le lisent. C'est aussi ce qui garantit « jamais pire qu'avant » aux enveloppes D2,
// qui construisent leur propre contexte a chaque appel.
//
// # POURQUOI LE CONSTRUCTEUR NE REND PAS D'ERREUR
//
// Les trois derivations ECHOUENT sur des films legitimes — une bobine partielle n'a pas de
// `chunk_00`, un film trop court ne donne pas trois frontieres nettes dans i0 — et chaque
// balayage rend AUJOURD'HUI son propre message a son propre moment (`ErrNoRegistryChunk`,
// « decoupage i0 illisible », « aucun slot biped (ti=35) ... »). Refuser au constructeur
// changerait ces messages ET l'etape a laquelle la cuisson s'arrete : la fixture
// `replay/testdata/minifilm_000d5950` n'a ni registre ni slot bipede, et
// `replay/zero_disque_test.go` exige l'erreur EXACTE des positions. Le contexte MEMORISE donc
// l'echec au lieu de le lever, et chaque accesseur le rejoue a l'identique, autant de fois qu'on
// le lui demande.
//
// # CE QU'IL N'EST PAS
//
// Ni un cache global (aucun `var` de paquet — le ratchet `archlint/filmdec_package_vars_test.go`
// gele leur compte), ni un objet partageable entre goroutines : il n'est ni verrouille ni
// atomique, et il vit sous le meme `LockProcessDecode` que le decodage qu'il sert.

import (
	"log/slog"

	"levelup/go-api/internal/analysis/filmsource"
)

// FilmContext porte les derivations d'un film qui ne dependent que de lui : les numeros de ses
// chunks de donnees, la bande de slots bipede, le decoupage d'i0 et le registre chunk_00. Il se
// construit par [NewFilmContextForMap] (production, avec le catalogue de la carte) ou par
// [NewFilmContext] (auto-detection seule) et se passe aux balayages.
//
// Les champs sont PRIVES : un layout ou un registre se lit avec l'erreur qui va avec (cf.
// l'en-tete), et une bande de slots exposee en clair serait modifiable par son lecteur.
type FilmContext struct {
	film *filmsource.Film

	chunks    []int
	chunksLus bool

	slots    SlotBand
	slotsLus bool

	// impose est le decoupage d'i0 que la REGLE DU CATALOGUE tranche a la construction (cf.
	// resolveI0Layout). Non nil = les trois champs `lay*` ci-dessous ne servent pas : aucune
	// auto-detection n'a lieu, et c'est aussi ce qui la retire du chemin de cuisson.
	impose *I0Layout

	lay    I0Layout
	layErr error
	layLu  bool

	reg    *Registry
	regErr error
	regLu  bool

	// prof est le PROFIL du film, resolu UNE fois (cf. l en-tete) et immuable. `profLu` dit
	// s il l a ete : `NewFilmContextForMap` le pose a la construction, `NewFilmContext` au
	// premier acces.
	prof   Profile
	profLu bool

	// bal est le PROFIL DE BALAYAGE de ce decodage (lot 2.3) : ce que les lecteurs de bits
	// construits sous ce contexte portent. Il nait a l INVARIANT ; la carte du match
	// (largeurs d axe objets du monde), la version de format (decoupage MPP) et la
	// calibration de `killsource` y substituent leurs valeurs, par
	// [FilmContext.PoserProfilDeBalayage]. C est ce qui a remplace l heritage par l etat du
	// processus : rien ici n est partage entre deux films.
	bal ProfilDeBalayage
}

// ProfilDeBalayage rend le profil que les lecteurs de ce contexte portent. PAR VALEUR : un
// appelant qui modifie ce qu il recoit ne modifie pas celui du contexte.
func (c *FilmContext) ProfilDeBalayage() ProfilDeBalayage {
	if c == nil {
		return ProfilDeBalayageParDefaut()
	}
	return c.bal
}

// PoserProfilDeBalayage installe le profil que les lecteurs SUIVANTS de ce contexte porteront,
// et rend le precedent — l appelant le restaure s il ne voulait le poser que le temps d un
// balayage. C est la SEULE porte : un balayage ne pose plus rien dans le processus.
func (c *FilmContext) PoserProfilDeBalayage(p ProfilDeBalayage) ProfilDeBalayage {
	prev := c.bal
	c.bal = p
	return prev
}

// PoserMPP installe le decoupage MPP du contexte et rend le precedent.
func (c *FilmContext) PoserMPP(w MPPWidths) MPPWidths {
	prev := c.bal.MPP
	c.bal.MPP = w
	return prev
}

// LargeursObjetDuMonde rend les largeurs d axe du chemin world-object de ce contexte.
func (c *FilmContext) LargeursObjetDuMonde() PrecisionDescriptor {
	return c.ProfilDeBalayage().LargeursObjetDuMonde()
}

// PoserLargeursObjetDuMonde installe des largeurs world-object brutes sur ce contexte.
func (c *FilmContext) PoserLargeursObjetDuMonde(d PrecisionDescriptor) {
	c.bal.PoserLargeursObjetDuMonde(d)
}

// PoserLargeursObjetDuMondeDepuisDecoupage installe les largeurs d axe de la CARTE sur ce
// contexte. C est la porte de `replay.installWorldObjectPrecision` et des instruments.
func (c *FilmContext) PoserLargeursObjetDuMondeDepuisDecoupage(l I0Layout) {
	c.bal.PoserLargeursObjetDuMondeDepuisDecoupage(l)
}

// PoserParamEtat force le `param_4` du moteur pour les lecteurs de ce contexte.
func (c *FilmContext) PoserParamEtat(v uint32) { c.bal.PoserParamEtat(v) }

// NouveauLecteur construit un lecteur de bits PORTANT LE PROFIL DE CE CONTEXTE. Tout balayage
// qui lit les octets d un film sous un contexte passe par la : c est ce qui fait descendre les
// largeurs de la carte et du format jusqu aux feuilles, sans variable de paquet.
func (c *FilmContext) NouveauLecteur(buf []byte) *BitReader {
	br := NewBitReader(buf)
	br.PoserProfil(c.ProfilDeBalayage())
	return br
}

// NewFilmContext ouvre le contexte d'un film DEJA CHARGE, SANS catalogue : le decoupage d'i0 est
// l'AUTO-DETECTE. C'est la forme des enveloppes D2 `ScanFilm*(dir)` et des usages hors production
// (recherche, instruments), qui n'ont pas d'entree de carte a fournir ; la cuisson, elle, passe
// par [NewFilmContextForMap].
//
// Il ne lit rien : chaque derivation est calculee au premier accesseur qui la demande, puis
// memorisee (cf. l'en-tete du fichier).
//
// `film` nil est ACCEPTE et n'est pas une erreur : la cuisson passe un film nil quand les chunks
// sont illisibles (`replaybuild.chargerFilm`), et chaque balayage rend alors son
// [ErrNoFilmChunk] a sa place — exactement comme un repertoire vide avant le lot 1.
func NewFilmContext(film *filmsource.Film) *FilmContext {
	return &FilmContext{film: film, bal: ProfilDeBalayageParDefaut()}
}

// NewFilmContextForMap ouvre le contexte d'un film DEJA CHARGE sous LA REGLE DU CATALOGUE (cf.
// l'en-tete du fichier) : le decoupage d'i0 est celui de `entry` quand l'entree est valide,
// l'auto-detection en repli ; un decoupage deja FORCE par l'appelant (`ScanFilmOptions.Layout`,
// `replay.Options.Scan`) reste maitre. Les deux parametres acceptent nil.
//
// C'est le constructeur de la CUISSON : `replay.BuildFromFilm` le construit une fois, lit le
// decoupage tranche par [FilmContext.ImposedLayout] pour en armer les positions, et passe le
// contexte aux six canaux delta et aux ramassages natifs — un seul decoupage pour tout le film.
func NewFilmContextForMap(film *filmsource.Film, entry *MapQuantEntry, forced *I0Layout) *FilmContext {
	c := &FilmContext{film: film, impose: resolveI0Layout(forced, entry),
		bal: ProfilDeBalayageParDefaut()}
	c.prof, c.profLu = ResolveProfile(film, entry), true
	journaliserProfilIncomplet(film, c.prof)
	return c
}

// journaliserProfilIncomplet emet L UNIQUE ligne du constructeur quand une cle du film manque a
// la table de profil (principe 12 : jamais de degradation silencieuse).
//
// ELLE NE SE DECLENCHE QUE SI LE FILM PORTE SON REGISTRE. Une bobine partielle ou une fixture
// sans `chunk_00` n a pas de cle a chercher : la journaliser serait du bruit a chaque instrument,
// et l absence de chunk est deja dite par [ErrNoFilmChunk] la ou elle compte.
//
// `slog.Warn` et non `WarnContext` : ce constructeur ne prend pas de `ctx`, comme
// l installateur des largeurs d axe de la carte (`replay/world_object_precision.go`) et
// `replay.avertirFormatSansProfil`, qui journalisent de la meme facon sur le meme chemin.
//
// RECOUVREMENT ASSUME avec `avertirFormatSansProfil` sur le seul cas « format inconnu » : cette
// ligne-ci nomme le PROFIL et ses deux cles, et elle couvre aussi le chemin `killcollector`, ou
// aucun autre avertissement n existe.
func journaliserProfilIncomplet(film *filmsource.Film, p Profile) {
	if p.Err() == nil {
		return
	}
	if _, ok := FilmRegistryChunk(film); !ok {
		return
	}
	slog.Warn("profil du film INCOMPLET — une cle ecrite dans le film est absente de la table "+
		"de profil ; le film reste decode, les replis existants decident et se comptent",
		"err", p.Err(), "format", p.FormatVersion(), "build", p.Build())
}

// resolveI0Layout EST LA REGLE, ecrite une fois : le decoupage FORCE s'il y en a un, sinon celui
// du CATALOGUE quand il est valide, sinon nil — l'auto-detection.
//
// Le repli sur nil n'est pas une tolerance : une entree de catalogue anterieure au champ des
// largeurs (`axisWidths` absent, donc `Valid()` faux) doit laisser lire le film plutot
// qu'imposer des largeurs nulles, exactement comme le chemin world-object garde son defaut.
func resolveI0Layout(forced *I0Layout, entry *MapQuantEntry) *I0Layout {
	if forced != nil {
		lay := *forced
		return &lay
	}
	if entry != nil {
		if lay := entry.Layout(); lay.Valid() {
			return &lay
		}
	}
	return nil
}

// ImposedLayout rend le decoupage d'i0 que la regle du catalogue a tranche a la construction, ou
// nil quand rien ne s'impose (auto-detection). La valeur est COPIEE : un appelant qui la range
// dans ses propres options ne peut pas modifier celle du contexte.
func (c *FilmContext) ImposedLayout() *I0Layout {
	if c == nil || c.impose == nil {
		return nil
	}
	lay := *c.impose
	return &lay
}

// Profile rend le PROFIL du film, resolu une fois (cf. l en-tete) et rendu PAR VALEUR : un
// lecteur qui modifie ce qu il recoit ne modifie pas le profil du contexte.
//
// Contexte nil = le profil des INVARIANTS, sans cle et sans carte : c est ce que rend
// [ResolveProfile] sur un film nul, et un appelant qui le lit sans verifier [FilmContext.ProfileErr]
// obtient donc le cadre d image-cle et les quantums, jamais une largeur inventee.
func (c *FilmContext) Profile() Profile {
	if c == nil {
		return ResolveProfile(nil, nil)
	}
	if !c.profLu {
		c.prof, c.profLu = ResolveProfile(c.film, nil), true
	}
	return c.prof
}

// ProfileErr rend l erreur TYPEE des cles absentes de la table de profil ([ErrUnknownFormat],
// [ErrUnknownBuild]), ou nil. C est par elle que l erreur REMONTE A L APPELANT : le constructeur
// n en rend pas (cf. l en-tete), il la journalise.
func (c *FilmContext) ProfileErr() error { return c.Profile().Err() }

// Film rend le film sous-jacent, pour les balayages qui lisent des chunks sans rien deriver.
func (c *FilmContext) Film() *filmsource.Film {
	if c == nil {
		return nil
	}
	return c.film
}

// ChunkNumbers rend les numeros des chunks de DONNEES du film ([FilmChunkNumbers]), releves une
// fois. La tranche est celle du contexte : ses lecteurs la parcourent, ils ne la modifient pas.
func (c *FilmContext) ChunkNumbers() []int {
	if c == nil {
		return nil
	}
	if !c.chunksLus {
		c.chunks, c.chunksLus = FilmChunkNumbers(c.film), true
	}
	return c.chunks
}

// ChunkAt rend les octets decompresses du chunk de NUMERO `num` et ses paquets ([FilmChunkAt]).
// Rien n'est memorise : la conversion des en-tetes est deja le prix plancher (cf. film_chunks.go).
func (c *FilmContext) ChunkAt(num int) ([]byte, []FilmPacket, bool) {
	if c == nil {
		return nil, nil, false
	}
	return FilmChunkAt(c.film, num)
}

// BipedSlots rend la bande de slots bipede du film ([bipedSlotBand] sur TOUS les chunks de
// donnees), relevee une fois.
//
// Bande VIDE quand le film n'a pas de chunk de donnees : les balayages testent `len(...) == 0`
// avant d'appeler, et rendent [ErrNoFilmChunk] — la garde ci-dessous ne fait que leur epargner
// l'indexation d'une tranche vide dans `bipedSlotBand`.
//
// CE N'EST PAS LA BANDE DE `DetectI0LayoutOf`, qui releve la SIENNE sur les six premiers chunks
// seulement : deux valeurs differentes, deux calculs, et c'est pourquoi la detection garde le
// sien (cf. i0_layout.go).
func (c *FilmContext) BipedSlots() SlotBand {
	if c == nil {
		return SlotBand{}
	}
	if !c.slotsLus {
		if nums := c.ChunkNumbers(); len(nums) > 0 {
			c.slots = bipedSlotBand(c.film, nums)
		}
		c.slotsLus = true
	}
	return c.slots
}

// I0Layout rend le decoupage d'i0 du film sous la REGLE DU CATALOGUE : celui que le catalogue
// impose quand l'entree de carte est valide (jamais d'erreur — il ne se lit pas dans le film),
// sinon l'AUTO-DETECTE ([DetectI0LayoutOf]), detecte une fois. L'erreur rendue est alors celle
// de la detection, BRUTE : c'est l'appelant qui l'habille (« decoupage i0 illisible : %w »),
// comme il le faisait de l'appel direct.
func (c *FilmContext) I0Layout() (I0Layout, error) {
	if c == nil {
		return I0Layout{}, ErrNoFilmChunk
	}
	if c.impose != nil {
		return *c.impose, nil
	}
	if !c.layLu {
		c.lay, _, c.layErr = DetectI0LayoutOf(c.film)
		c.layLu = true
	}
	return c.lay, c.layErr
}

// Registry rend le registre ECS du film (le chunk NUMERO 0), analyse une fois.
//
// [ErrNoRegistryChunk] quand le film ne porte pas son registre — bobine partielle, fixture : les
// lecteurs d'archetype le disent plutot que de rendre un registre vide, qui se lirait comme
// « archetype absent du build ».
//
// C'EST LE SEUL LECTEUR DE REGISTRE DU PAQUET : les six accesseurs d'archetype
// (`bipedArchetype`, `EquipmentArchetype`, `groundWeaponArchetype`, `managedPropertyArchetype`,
// `filmArchetype`, `objectiveArchetype`) en derivent tous, chacun ne gardant que son message
// d'archetype manquant. Un garde-rail ferme la porte a un second site d'analyse
// (`archlint/no_recomputed_film_context_test.go`).
func (c *FilmContext) Registry() (*Registry, error) {
	if c == nil {
		return nil, ErrNoRegistryChunk
	}
	if !c.regLu {
		c.regLu = true
		raw, ok := FilmRegistryChunk(c.film)
		if !ok {
			c.regErr = ErrNoRegistryChunk
		} else {
			c.reg, c.regErr = ParseRegistryChunk(raw)
		}
	}
	return c.reg, c.regErr
}

// archetype rend l'archetype `ti` du registre memorise. `ok` faux = le registre a ete lu mais ne
// porte pas cet archetype ; c'est a l'accesseur nomme de dire lequel manque, avec SON message —
// les six libelles historiques sont conserves mot pour mot.
func (c *FilmContext) archetype(ti int) (Archetype, *Registry, bool, error) {
	reg, err := c.Registry()
	if err != nil {
		return Archetype{}, nil, false, err
	}
	arch, ok := reg.Archetype(ti)
	return arch, reg, ok, nil
}

// contexteDeBobine ouvre le contexte d un film DEJA CHARGE et y pose les largeurs d axe LUES
// DANS LE FILM.
//
// C EST LE CONTEXTE DES ENVELOPPES D2 (`ScanFilm*(dir)`), et la raison est ecrite dans leurs
// propres commentaires : « un instrument qui oublie `SetWorldObjectPrecisionFromLayout`
// desaligne les desers sans lever d erreur » (r11_journal, r12_socle, r9_creneaux). Jusqu au
// lot 2.3, chaque instrument devait detecter le decoupage puis l installer a la main dans une
// variable de paquet ; l enveloppe le fait desormais, DEPUIS LA MEME SOURCE (le film), et
// l oubli n existe plus.
//
// LA CUISSON NE PASSE PAS PAR LA : elle prend les largeurs du CATALOGUE de la carte
// (`replay.installWorldObjectPrecision`), jamais de l auto-detection — cf. la regle du
// catalogue en tete de ce fichier. Un decoupage illisible laisse l invariant, sans erreur :
// c est ce que le defaut de paquet faisait avant.
func contexteDeBobine(film *filmsource.Film) *FilmContext {
	fc := NewFilmContext(film)
	if lay, _, err := DetectI0LayoutOf(film); err == nil {
		fc.PoserLargeursObjetDuMondeDepuisDecoupage(lay)
	}
	return fc
}

// ContexteDeFilm ouvre le contexte d un film DEPUIS SON REPERTOIRE et y pose les largeurs d axe
// LUES DANS LE FILM (`DetectI0LayoutOf`). Rend aussi le decoupage, dont les appelants se servent
// pour journaliser.
//
// ENVELOPPE D2, HORS PRODUCTION — meme nature que les `ScanFilm*(dir)`. C est le geste que les
// instruments repetaient a la main avant le lot 2.3 : detecter le decoupage, puis l installer
// dans une variable de paquet que tout le processus prenait. La cuisson, elle, prend les
// largeurs du CATALOGUE de la carte (`replay.installWorldObjectPrecision`), jamais de
// l auto-detection : cf. la regle du catalogue en tete de ce fichier.
func ContexteDeFilm(dir string) (*FilmContext, I0Layout, error) {
	film, err := filmsource.LoadDir(dir, nil)
	if err != nil {
		return nil, I0Layout{}, err
	}
	lay, _, err := DetectI0LayoutOf(film)
	if err != nil {
		return NewFilmContext(film), I0Layout{}, err
	}
	return contexteDeBobine(film), lay, nil
}
