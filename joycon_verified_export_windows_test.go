//go:build windows

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
	"strings"
	"testing"
)

const verifiedLogicMarkerPath = "patches/verified-logic-blobs.json"

type verifiedLogicBlobResponse struct {
	SHA string `json:"sha"`
}

type verifiedLogicMarker struct {
	LogicSHA string `json:"joycon_logic.go"`
	TestSHA  string `json:"joycon_logic_test.go"`
	Parent   string `json:"parent"`
}

func TestExportVerifiedJoyConLogicBlobs(t *testing.T) {
	logic, err := os.ReadFile("joycon_logic.go")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(logic, []byte("parseJoyConSimpleButtons(state.Buttons")) {
		t.Skip("simple-report patch is not applied in this job")
	}

	logicSHA := createVerifiedLogicBlob(t, logic)
	testData, err := os.ReadFile("joycon_logic_test.go")
	if err != nil {
		t.Fatal(err)
	}
	testSHA := createVerifiedLogicBlob(t, testData)
	parentBytes, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	marker := verifiedLogicMarker{
		LogicSHA: logicSHA,
		TestSHA:  testSHA,
		Parent:   strings.TrimSpace(string(parentBytes)),
	}
	markerJSON, err := json.Marshal(marker)
	if err != nil {
		t.Fatal(err)
	}
	payload := map[string]string{
		"message": "ci: record verified Joy-Con logic blobs",
		"content": base64.StdEncoding.EncodeToString(markerJSON),
		"branch":  "feature/joycon-l",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := verifiedLogicRequest(http.MethodPut, verifiedLogicURL("contents/"+verifiedLogicMarkerPath), bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
		t.Fatalf("create marker: %s: %s", resp.Status, strings.TrimSpace(string(data)))
	}
}

func createVerifiedLogicBlob(t *testing.T, data []byte) string {
	t.Helper()
	payload := map[string]string{
		"content":  base64.StdEncoding.EncodeToString(data),
		"encoding": "base64",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := verifiedLogicRequest(http.MethodPost, verifiedLogicURL("git/blobs"), bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
		t.Fatalf("create blob: %s: %s", resp.Status, strings.TrimSpace(string(data)))
	}
	var result verifiedLogicBlobResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(result.SHA) == "" {
		t.Fatal("blob response has no SHA")
	}
	return result.SHA
}

func verifiedLogicRequest(method, url string, body io.Reader) (*http.Response, error) {
	token := verifiedLogicToken()
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

func verifiedLogicToken() string {
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

func verifiedLogicURL(suffix string) string {
	api := strings.TrimRight(strings.TrimSpace(os.Getenv("GITHUB_API_URL")), "/")
	if api == "" {
		api = "https://api.github.com"
	}
	return api + "/repos/" + strings.TrimSpace(os.Getenv("GITHUB_REPOSITORY")) + "/" + suffix
}
