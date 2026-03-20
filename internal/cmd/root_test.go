package cmd

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/juanibiapina/mcpli/internal/config"
)

func TestCreateToolCommandHelpIncludesFullInputSchema(t *testing.T) {
	tool := config.Tool{
		Name:        "search_products",
		Description: "Search products by user query",
		InputSchema: []byte(`{"type":"object","properties":{"query":{"type":"string"},"filters":{"type":"object","properties":{"category":{"type":"string"}},"required":["category"]}},"required":["query"]}`),
	}

	cmd := createToolCommand("knuspr", &config.Server{}, tool)

	stdout, _ := captureOutput(t, func() {
		if err := cmd.Help(); err != nil {
			t.Fatalf("help should succeed: %v", err)
		}
	})

	assertContains(t, stdout, "json-schema")
	assertContains(t, stdout, "```json")
	assertContains(t, stdout, `"query": {`)
	assertContains(t, stdout, `"filters": {`)
	assertContains(t, stdout, `"category": {`)
	assertContains(t, stdout, "Usage:")
}

func TestCreateToolCommandInvalidJSONReturnsHelpfulError(t *testing.T) {
	tool := config.Tool{Name: "search_products", Description: "Search products"}
	server := &config.Server{URL: "http://example.com", Headers: map[string]string{}}
	cmd := createToolCommand("knuspr", server, tool)

	stdout, stderr := captureOutput(t, func() {
		err := cmd.RunE(cmd, []string{"{"})
		if err == nil {
			t.Fatal("expected invalid JSON error")
		}
		if !strings.Contains(err.Error(), "invalid JSON arguments") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !cmd.SilenceErrors || !cmd.SilenceUsage {
		t.Fatal("expected command to silence Cobra default output")
	}

	assertContains(t, stderr, "Error: invalid JSON arguments")
	assertContains(t, stdout, "Hint:")
	assertContains(t, stdout, "Usage:")
}

func TestCreateToolCommandTreatsToolErrorEnvelopeAsFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"jsonrpc":"2.0","id":3,"result":{"isError":true,"content":[{"type":"text","text":"boom"}]}}`)
	}))
	defer server.Close()

	tool := config.Tool{Name: "search_products", Description: "Search products"}
	cmd := createToolCommand("knuspr", &config.Server{URL: server.URL, Headers: map[string]string{}}, tool)

	stdout, stderr := captureOutput(t, func() {
		err := cmd.RunE(cmd, []string{`{"query":"milk"}`})
		if err == nil {
			t.Fatal("expected tool envelope error")
		}
		if !strings.Contains(err.Error(), "tool returned error response") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	assertContains(t, stderr, "Error: tool returned error response")
	assertContains(t, stdout, "Hint:")
	assertContains(t, stdout, "Usage:")
}

func captureOutput(t *testing.T, fn func()) (string, string) {
	t.Helper()

	oldStdout := os.Stdout
	oldStderr := os.Stderr

	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}
	stderrReader, stderrWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stderr pipe: %v", err)
	}

	os.Stdout = stdoutWriter
	os.Stderr = stderrWriter

	defer func() {
		os.Stdout = oldStdout
		os.Stderr = oldStderr
	}()

	fn()

	_ = stdoutWriter.Close()
	_ = stderrWriter.Close()

	stdoutBytes, err := io.ReadAll(stdoutReader)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	stderrBytes, err := io.ReadAll(stderrReader)
	if err != nil {
		t.Fatalf("read stderr: %v", err)
	}

	return string(stdoutBytes), string(stderrBytes)
}

func assertContains(t *testing.T, text, want string) {
	t.Helper()
	if !strings.Contains(text, want) {
		t.Fatalf("expected output to contain %q, got:\n%s", want, text)
	}
}

func Example_printToolInputSchema() {
	printToolInputSchema(config.Tool{InputSchema: []byte(`{"type":"object"}`)})
	fmt.Println("done")
	// Output:
	// json-schema
	// ```json
	// {
	//   "type": "object"
	// }
	// ```
	//
	// done
}
