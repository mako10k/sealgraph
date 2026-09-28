package tracerecipe

import (
	"strings"
	"testing"
)

const validID = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func TestParseValidRecipeAndAlternatives(t *testing.T) {
	data := `{"runs":[{"source_start":2,"source":"a","kind":"external","length":3},{"length":1,"kind":"untraced"}],"sources":[{"file":"x","source_key":"k","name":"a"}],"schema":"sealgraph/trace-recipe/v1"}`
	recipe, err := Parse([]byte(data))
	if err != nil {
		t.Fatal(err)
	}
	if len(recipe.Sources) != 1 || recipe.Sources[0].File != "x" || recipe.Runs[0].SourceStart != 2 {
		t.Fatalf("unexpected recipe: %#v", recipe)
	}

	snapshot, err := Parse([]byte(`{"schema":"sealgraph/trace-recipe/v1","sources":[{"name":"a","snapshot":"` + validID + `"}],"runs":[{"kind":"external","length":1,"source":"a","source_start":0}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Sources[0].SnapshotID.String() != validID {
		t.Fatalf("snapshot ID not exposed: %#v", snapshot.Sources[0])
	}
}

func TestParseRejectsStructuralAndReferenceErrors(t *testing.T) {
	valid := `{"schema":"sealgraph/trace-recipe/v1","sources":[{"name":"a","source_key":"k","file":"x"}],"runs":[{"kind":"external","length":1,"source":"a","source_start":0}]}`
	cases := []string{
		strings.Replace(valid, `"sources":[`, `"sources":null,"unused":[`, 1),
		`{"schema":"sealgraph/trace-recipe/v1","sources":null,"runs":[]}`,
		`{"schema":"sealgraph/trace-recipe/v1","sources":[],"runs":null}`,
		`{"schema":"sealgraph/trace-recipe/v1","sources":[{"name":"a","source_key":"k","file":"x","extra":1}],"runs":[]}`,
		`{"schema":"sealgraph/trace-recipe/v1","sources":[{"name":"a","source_key":"k","file":"x"},{"name":"a","source_key":"k2","file":"y"}],"runs":[{"kind":"external","length":1,"source":"a","source_start":0}]}`,
		`{"schema":"sealgraph/trace-recipe/v1","sources":[{"name":"a","source_key":"k","file":"x"}],"runs":[{"kind":"external","length":1,"source":"missing","source_start":0}]}`,
		`{"schema":"sealgraph/trace-recipe/v1","sources":[{"name":"a","source_key":"k","file":"x"}],"runs":[{"kind":"untraced","length":1}],"extra":0}`,
	}
	for i, data := range cases {
		if _, err := Parse([]byte(data)); err == nil {
			t.Errorf("case %d accepted", i)
		}
	}
	if _, err := Parse([]byte(valid + `{"x":1}`)); err == nil {
		t.Error("trailing value accepted")
	}
}

func TestParseRejectsDuplicateMembersAndInvalidNumbersIDsUTF8(t *testing.T) {
	cases := []string{
		`{"schema":"sealgraph/trace-recipe/v1","schema":"sealgraph/trace-recipe/v1","sources":[],"runs":[]}`,
		`{"schema":"sealgraph/trace-recipe/v1","sources":[{"name":"a","source_key":"k","file":"x"}],"runs":[{"kind":"untraced","kind":"untraced","length":1}]}`,
		`{"schema":"sealgraph/trace-recipe/v1","sources":[{"name":"a","source_key":"k","file":"x"}],"runs":[{"kind":"untraced","length":0}]}`,
		`{"schema":"sealgraph/trace-recipe/v1","sources":[{"name":"a","source_key":"k","file":"x"}],"runs":[{"kind":"untraced","length":1.0}]}`,
		`{"schema":"sealgraph/trace-recipe/v1","sources":[{"name":"a","snapshot":"bad"}],"runs":[{"kind":"external","length":1,"source":"a","source_start":0}]}`,
	}
	for i, data := range cases {
		if _, err := Parse([]byte(data)); err == nil {
			t.Errorf("case %d accepted", i)
		}
	}
	if _, err := Parse([]byte{'{', '"', 's', 'c', 'h', 'e', 'm', 'a', '"', ':', '"', 0xff}); err == nil {
		t.Error("invalid UTF-8 accepted")
	}
}
