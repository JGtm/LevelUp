package sharedprovider

// reader_drain_test.go — RÉGRESSION des deux crashs du 2026-09-16
// (« panic: sync: WaitGroup is reused before previous Wait has returned »,
// waitForDrain.func1, provider_writer.go:252-253).
//
// Tests de boîte blanche sur le SEUL comptage des lecteurs : aucun fichier
// DuckDB ouvert, aucune goroutine d'attente, donc un verdict déterministe qui
// tient sous -race et -count=50.

import (
	"sync"
	"testing"
	"time"
)

// acquireReaderForTest reproduit ce que fait Get() en StateRO : incrément sous
// p.mu puis release à usage unique.
func acquireReaderForTest(p *providerImpl) func() {
	p.mu.Lock()
	p.trackReaderLocked()
	p.mu.Unlock()
	return sync.OnceFunc(p.releaseReader)
}

// drainExpired attend le canal de drain au plus d, et rend false s'il n'a pas
// été fermé (= drain expiré, ce que fait waitForDrain sur ctx.Done()).
func drainExpired(t *testing.T, ch <-chan struct{}, d time.Duration) bool {
	t.Helper()
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ch:
		return false
	case <-timer.C:
		return true
	}
}

// TestDrainAbandonnePuisRepris : un drain qui EXPIRE avec un lecteur en vol ne
// laisse rien derrière lui. Des cycles Get/release enchaînés pendant que
// l'ancien drain est abandonné ne doivent rien réveiller ni faire paniquer, et
// un second drain doit réussir normalement.
//
// C'est la séquence exacte de production : drain timeout →
// rollbackFromDraining() repasse en StateRO → les Get reprennent → le compteur
// retombe à zéro (c'est ce réveil qui faisait paniquer l'ancienne WaitGroup).
func TestDrainAbandonnePuisRepris(t *testing.T) {
	p := &providerImpl{}

	// Un lecteur en vol qui ne relâchera pas avant la fin du premier drain.
	releaseBloqueur := acquireReaderForTest(p)

	// 1er drain : expire (le lecteur tient toujours).
	premier := p.beginDrain()
	if !drainExpired(t, premier, 50*time.Millisecond) {
		t.Fatal("le drain s'est terminé alors qu'un lecteur est en vol")
	}
	p.abandonDrain(premier)

	// N cycles Get/release pendant que l'ancien drain est abandonné.
	for i := 0; i < 200; i++ {
		acquireReaderForTest(p)()
	}
	select {
	case <-premier:
		t.Fatal("le canal du drain abandonné a été fermé — une attente lui a survécu")
	default:
	}

	// Le lecteur bloqueur relâche : le compteur retombe à zéro — c'est CE réveil
	// qui faisait paniquer l'ancienne WaitGroup. Le drain abandonné doit rester
	// muet (un canal abandonné que le release refermerait pourrait être rendu
	// DÉJÀ FERMÉ à un drain suivant, qui conclurait à tort au drain terminé et
	// fermerait le handle sous les lecteurs en vol).
	releaseBloqueur()
	if got := readersForTest(p); got != 0 {
		t.Fatalf("compteur de lecteurs = %d après tous les release (attendu 0)", got)
	}
	select {
	case <-premier:
		t.Fatal("le canal du drain abandonné a été fermé par un release ultérieur")
	default:
	}

	// 2e drain, sans lecteur : immédiat.
	second := p.beginDrain()
	if drainExpired(t, second, 50*time.Millisecond) {
		t.Fatal("le second drain n'a pas abouti alors qu'aucun lecteur n'est en vol")
	}

	// 3e drain, avec un lecteur relâché juste après : abouti par le release.
	release := acquireReaderForTest(p)
	troisieme := p.beginDrain()
	go release()
	if drainExpired(t, troisieme, time.Second) {
		t.Fatal("le drain n'a pas été réveillé par le dernier release")
	}
}

// TestDrainSousConcurrence : des lecteurs et des drains qui se chevauchent en
// continu — le motif de production (sync qui swappe pendant que l'API lit).
// Aucune panique ne doit se produire et le compteur doit revenir à zéro.
func TestDrainSousConcurrence(t *testing.T) {
	p := &providerImpl{}

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				acquireReaderForTest(p)()
			}
		}()
	}
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				ch := p.beginDrain()
				if drainExpired(t, ch, time.Millisecond) {
					p.abandonDrain(ch)
				}
			}
		}()
	}
	wg.Wait()

	if got := readersForTest(p); got != 0 {
		t.Fatalf("compteur de lecteurs = %d en fin de course (attendu 0)", got)
	}
	// Un drain final doit aboutir immédiatement : aucune attente fantôme.
	if drainExpired(t, p.beginDrain(), 50*time.Millisecond) {
		t.Fatal("drain final bloqué — le compteur ou le canal a fuité")
	}
}

// readersForTest lit le compteur sous p.mu.
func readersForTest(p *providerImpl) int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.readers
}
