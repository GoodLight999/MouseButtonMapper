package main

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"
)

func TestEmbeddedDefaultConfigIsValidJSON(t *testing.T) {
	var value map[string]any
	if err := json.Unmarshal([]byte(defaultConfigJSON), &value); err != nil {
		t.Fatalf("defaultConfigJSON is invalid: %v", err)
	}
	profiles, ok := value["Profiles"].([]any)
	if !ok || len(profiles) == 0 {
		t.Fatal("defaultConfigJSON must contain at least one profile")
	}
}

func TestWebUIRequiredControlsExistOnce(t *testing.T) {
	required := []string{
		"autoEnabled",
		"baseProfile",
		"editProfile",
		"bindingRows",
		"saveBinding",
		"ruleRows",
		"saveRule",
		"recordInput",
		"recordOutput",
		"ruleLongEnabled",
		"ruleLongMs",
		"ruleLongAction",
		"ruleLongOutput",
		"recordLongOutput",
	}
	for _, id := range required {
		needle := `id="` + id + `"`
		if count := strings.Count(webHTML, needle); count != 1 {
			t.Fatalf("required control %q occurs %d times", id, count)
		}
	}
}

func TestWebUIDoesNotContainDuplicateIDs(t *testing.T) {
	re := regexp.MustCompile(`\bid="([^"]+)"`)
	seen := map[string]bool{}
	for _, match := range re.FindAllStringSubmatch(webHTML, -1) {
		id := match[1]
		if seen[id] {
			t.Fatalf("duplicate HTML id: %s", id)
		}
		seen[id] = true
	}
}
