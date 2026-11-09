package decfilm_test

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// TestLeReadmeCiteLExempleCompile : le bloc de code de la section « Example » du README du module est
// le corps de ExampleScanDeaths, a l'octet pres (indentation d'un niveau retiree). L'exemple est
// compile et execute par `go test` ; sa copie du README ne peut donc pas diverger sans que ce test
// rougisse.
func TestLeReadmeCiteLExempleCompile(t *testing.T) {
	exemple, err := os.ReadFile("example_test.go")
	if err != nil {
		t.Fatalf("lecture de example_test.go : %v", err)
	}
	corps := regexp.MustCompile(`(?s)func ExampleScanDeaths\(\) \{\n(.*?)\n\}\n`).FindSubmatch(exemple)
	if corps == nil {
		t.Fatal("ExampleScanDeaths introuvable dans example_test.go")
	}
	var attendu []string
	for _, ligne := range strings.Split(string(corps[1]), "\n") {
		attendu = append(attendu, strings.TrimPrefix(ligne, "\t"))
	}
	readme, err := os.ReadFile("../README.md")
	if err != nil {
		t.Fatalf("lecture du README : %v", err)
	}
	bloc := regexp.MustCompile("(?s)## Example\n.*?```go\n(.*?)```").FindSubmatch(readme)
	if bloc == nil {
		t.Fatal("bloc de code de la section Example introuvable dans le README")
	}
	if got, want := strings.TrimRight(string(bloc[1]), "\n"), strings.Join(attendu, "\n"); got != want {
		t.Errorf("le README ne cite plus l'exemple compile.\nREADME :\n%s\nexample_test.go :\n%s", got, want)
	}
}
