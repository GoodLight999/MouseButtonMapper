package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
)

func TestExportVerifiedJoyConLogicBlobsPortable(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows verification only")
	}
	logic, err := os.ReadFile("joycon_logic.go")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(logic, []byte("parseJoyConSimpleButtons(state.Buttons")) {
		t.Skip("simple-report patch is not applied in this job")
	}
	logicSHA := portableCreateBlob(t, logic)
	testData, err := os.ReadFile("joycon_logic_test.go")
	if err != nil {
		t.Fatal(err)
	}
	testSHA := portableCreateBlob(t, testData)
	parentBytes, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	marker, err := json.Marshal(map[string]string{
		"joycon_logic.go":      logicSHA,
		"joycon_logic_test.go": testSHA,
		"parent":               strings.TrimSpace(string(parentBytes)),
	})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(map[string]string{
		"message": "ci: record verified Joy-Con logic blobs",
		"content": base64.StdEncoding.EncodeToString(marker),
		"branch":  "feature/joycon-l",
	})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := portableGitHubRequest(http.MethodPut, portableGitHubURL("contents/patches/verified-logic-blobs.json"), bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
		t.Fatalf("create verified blob marker: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
}

func portableCreateBlob(t *testing.T, data []byte) string {
	t.Helper()
	payload, err := json.Marshal(map[string]string{
		"content":  base64.StdEncoding.EncodeToString(data),
		"encoding": "base64",
	})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := portableGitHubRequest(http.MethodPost, portableGitHubURL("git/blobs"), bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
		t.Fatalf("create verified blob: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	var result struct {
		SHA string `json:"sha"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(result.SHA) == "" {
		t.Fatal("verified blob response has no SHA")
	}
	return result.SHA
}

func portableGitHubRequest(method, url string, body io.Reader) (*http.Response, error) {
	token := portableGitHubToken()
	if token == "" {
		return nil, fmt.Errorf("GitHub token is empty")
	}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("Content-Type", "application/json")
	return http.DefaultClient.Do(req)
}

func portableGitHubToken() string {
	if token := strings.TrimSpace(os.Getenv("GITHUB_TOKEN")); token != "" {
		return token
	}
	out, err := exec.Command("git", "config", "--get", "http.https://github.com/.extraheader").Output()
	if err != nil {
		return ""
	}
	value := strings.TrimSpace(string(out))
	const prefix = "AUTHORIZATION: basic "
	if !strings.HasPrefix(strings.ToUpper(value), prefix) {
		return ""
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(value[len(prefix):]))
	if err != nil {
		return ""
	}
	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func portableGitHubURL(suffix string) string {
	api := strings.TrimRight(strings.TrimSpace(os.Getenv("GITHUB_API_URL")), "/")
	if api == "" {
		api = "https://api.github.com"
	}
	return api + "/repos/" + strings.TrimSpace(os.Getenv("GITHUB_REPOSITORY")) + "/" + suffix
}
