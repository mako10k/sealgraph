package domain

import "testing"

func TestValidateREFUsesGitCompatiblePathNames(t *testing.T) {
	valid := []string{"ROOT-001", "requirements/external/ROOT-001", "design/api.v1"}
	for _, ref := range valid {
		if err := ValidateREF(ref); err != nil {
			t.Errorf("ValidateREF(%q) = %v, want valid", ref, err)
		}
	}
	invalid := []string{"", "/root", "root/", "root//child", ".hidden", "a/.hidden", "a.lock", "a/b.lock", "a..b", "a b", "a\\b", "@", "a@{b", "a?b", "a."}
	for _, ref := range invalid {
		if err := ValidateREF(ref); err == nil {
			t.Errorf("ValidateREF(%q) succeeded, want error", ref)
		}
	}
}

func TestValidateTagNameReservesSelectorSyntax(t *testing.T) {
	for _, name := range []string{"reviewed", "release/1.0", "DEADBEEF", "設計済み"} {
		if err := ValidateTagName(name); err != nil {
			t.Errorf("ValidateTagName(%q) = %v, want valid", name, err)
		}
	}
	for _, name := range []string{"", "review@old", "deadbeef", "0123", "line\nfeed"} {
		if err := ValidateTagName(name); err == nil {
			t.Errorf("ValidateTagName(%q) succeeded, want error", name)
		}
	}
}
