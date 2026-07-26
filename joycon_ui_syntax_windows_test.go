//go:build windows

package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestJoyConUIJavaScriptSyntax(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is not installed")
	}
	path := filepath.Join(t.TempDir(), "joycon-ui.js")
	if err := os.WriteFile(path, []byte(joyConUIJS), 0o600); err != nil {
		t.Fatalf("write temporary JavaScript: %v", err)
	}
	output, err := exec.Command(node, "--check", path).CombinedOutput()
	if err != nil {
		t.Fatalf("Joy-Con UI JavaScript syntax error: %v\n%s", err, output)
	}
}

func TestJoyConUIHandlerServesOnlyJavaScriptGET(t *testing.T) {
	app := &App{}
	request := httptest.NewRequest(http.MethodGet, "/joycon-ui.js", nil)
	response := httptest.NewRecorder()
	app.webJoyConUIJS(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("GET status=%d", response.Code)
	}
	if contentType := response.Header().Get("Content-Type"); !strings.Contains(contentType, "application/javascript") {
		t.Fatalf("Content-Type=%q", contentType)
	}
	if response.Body.String() != joyConUIJS {
		t.Fatal("handler response differs from embedded Joy-Con UI")
	}

	request = httptest.NewRequest(http.MethodPost, "/joycon-ui.js", nil)
	response = httptest.NewRecorder()
	app.webJoyConUIJS(response, request)
	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST status=%d", response.Code)
	}
	if allow := response.Header().Get("Allow"); allow != http.MethodGet {
		t.Fatalf("Allow=%q", allow)
	}
}
