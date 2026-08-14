package domain

import "testing"

func TestValidateREFUsesGitCompatiblePathNames(t *testing.T) {
	valid := []string{"ROOT-001", "requirements/external/ROOT-001", "design/api.v1", "team/@name"}
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
