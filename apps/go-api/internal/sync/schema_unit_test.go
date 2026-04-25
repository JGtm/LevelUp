package sync

import (
	"testing"
)

const helloLiteral = "hello"

// ── splitSQL ─────────────────────────────────────────────────────────────────

func TestSplitSQL_SingleStatement(t *testing.T) {
	got := splitSQL("CREATE TABLE foo (id INT);")
	if len(got) != 1 || got[0] != "CREATE TABLE foo (id INT)" {
		t.Fatalf("unexpected: %v", got)
	}
}

func TestSplitSQL_MultipleStatements(t *testing.T) {
	got := splitSQL("CREATE TABLE foo (id INT); INSERT INTO foo VALUES (1); SELECT * FROM foo;")
	if len(got) != 3 {
		t.Fatalf("expected 3 stmts, got %d", len(got))
	}
}

func TestSplitSQL_EmptyStatements(t *testing.T) {
	got := splitSQL(";;; ;")
	if len(got) != 0 {
		t.Fatalf("expected 0 stmts, got %d: %v", len(got), got)
	}
}

func TestSplitSQL_NoTrailingSemicolon(t *testing.T) {
	got := splitSQL("SELECT 1")
	if len(got) != 1 || got[0] != "SELECT 1" {
		t.Fatalf("unexpected: %v", got)
	}
}

func TestSplitSQL_WhitespaceStatements(t *testing.T) {
	got := splitSQL("  SELECT 1  ;  \n  SELECT 2  \n  ")
	if len(got) != 2 {
		t.Fatalf("expected 2 stmts, got %d: %v", len(got), got)
	}
	if got[0] != "SELECT 1" {
		t.Fatalf("unexpected first: %q", got[0])
	}
	if got[1] != "SELECT 2" {
		t.Fatalf("unexpected second: %q", got[1])
	}
}

// ── trimSpace ────────────────────────────────────────────────────────────────

func TestTrimSpace_Normal(t *testing.T) {
	if got := trimSpace("  hello  "); got != helloLiteral {
		t.Fatalf("expected 'hello', got %q", got)
	}
}

func TestTrimSpace_Tabs(t *testing.T) {
	if got := trimSpace("\t\nhello\r\n"); got != helloLiteral {
		t.Fatalf("expected 'hello', got %q", got)
	}
}

func TestTrimSpace_Empty(t *testing.T) {
	if got := trimSpace("   "); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestTrimSpace_NoWhitespace(t *testing.T) {
	if got := trimSpace(helloLiteral); got != helloLiteral {
		t.Fatalf("expected 'hello', got %q", got)
	}
}

// ── truncate ─────────────────────────────────────────────────────────────────

func TestTruncate_Short(t *testing.T) {
	if got := truncate("hi", 10); got != "hi" {
		t.Fatalf("expected 'hi', got %q", got)
	}
}

func TestTruncate_Exact(t *testing.T) {
	if got := truncate("12345", 5); got != "12345" {
		t.Fatalf("expected '12345', got %q", got)
	}
}

func TestTruncate_Long(t *testing.T) {
	if got := truncate("abcdefghij", 5); got != "abcde..." {
		t.Fatalf("expected 'abcde...', got %q", got)
	}
}
